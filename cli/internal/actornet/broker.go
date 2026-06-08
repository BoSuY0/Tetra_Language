package actornet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"syscall"
	"time"

	"tetra_language/compiler/actorwire"
)

type Config struct {
	Addr       string
	ReportPath string
}

type Report struct {
	Runtime             string    `json:"runtime"`
	Transport           string    `json:"transport"`
	ListenAddr          string    `json:"listen_addr"`
	StartedAt           time.Time `json:"started_at"`
	StoppedAt           time.Time `json:"stopped_at,omitempty"`
	ConnectedNodes      int       `json:"connected_nodes"`
	AcceptedConnections int64     `json:"accepted_connections"`
	RoutedFrames        int64     `json:"routed_frames"`
	DroppedFrames       int64     `json:"dropped_frames"`
	DecodeErrors        int64     `json:"decode_errors"`
	LastError           string    `json:"last_error,omitempty"`
}

type Broker struct {
	cfg      Config
	listener net.Listener

	mu     sync.Mutex
	nodes  map[uint16]*nodeConn
	report Report
	closed bool
	done   chan struct{}
	wg     sync.WaitGroup

	closeOnce sync.Once
	closeErr  error
}

type nodeConn struct {
	nodeID  uint16
	conn    net.Conn
	writeMu sync.Mutex
}

func NewBroker(cfg Config) (*Broker, error) {
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1:0"
	}
	listener, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return nil, err
	}
	return &Broker{
		cfg:      cfg,
		listener: listener,
		nodes:    make(map[uint16]*nodeConn),
		done:     make(chan struct{}),
		report: Report{
			Runtime:    "actornet",
			Transport:  "loopback-tcp",
			ListenAddr: listener.Addr().String(),
			StartedAt:  time.Now().UTC(),
		},
	}, nil
}

func (b *Broker) Addr() string {
	return b.listener.Addr().String()
}

func (b *Broker) Serve(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	go func() {
		select {
		case <-ctx.Done():
			_ = b.Close()
		case <-b.done:
		}
	}()

	for {
		conn, err := b.listener.Accept()
		if err != nil {
			if b.isClosed() {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return nil
			}
			b.recordError(err)
			return err
		}
		b.recordAcceptedConnection()
		b.wg.Add(1)
		go func() {
			defer b.wg.Done()
			b.handleConn(conn)
		}()
	}
}

func (b *Broker) Close() error {
	b.closeOnce.Do(func() {
		b.mu.Lock()
		if b.closed {
			b.mu.Unlock()
			return
		}
		b.closed = true
		if b.done != nil {
			close(b.done)
		}
		conns := make([]net.Conn, 0, len(b.nodes))
		for _, node := range b.nodes {
			conns = append(conns, node.conn)
		}
		b.mu.Unlock()

		if err := b.listener.Close(); err != nil && b.closeErr == nil && !errors.Is(err, net.ErrClosed) {
			b.closeErr = err
		}
		for _, conn := range conns {
			_ = conn.Close()
		}
		b.wg.Wait()

		b.mu.Lock()
		b.report.StoppedAt = time.Now().UTC()
		b.report.ConnectedNodes = len(b.nodes)
		if b.cfg.ReportPath != "" {
			b.closeErr = b.writeReportLocked()
		}
		b.mu.Unlock()
	})
	return b.closeErr
}

func (b *Broker) Report() Report {
	b.mu.Lock()
	defer b.mu.Unlock()
	report := b.report
	report.ConnectedNodes = len(b.nodes)
	return report
}

func (b *Broker) handleConn(conn net.Conn) {
	defer conn.Close()

	var registeredNode uint16
	for {
		frame, err := readFrame(conn)
		if err != nil {
			if !isClosedConnError(err) {
				b.recordError(err)
			}
			if registeredNode != 0 {
				b.unregisterNode(registeredNode, conn)
			}
			return
		}

		if frame.Type == actorwire.FrameHello {
			node, ok := b.registerNode(frame.SourceNodeID, conn)
			if !ok {
				return
			}
			registeredNode = node.nodeID
			_ = node.write(actorwire.Frame{
				Type:         actorwire.FrameHelloAck,
				SourceNodeID: frame.SourceNodeID,
				DestNodeID:   frame.SourceNodeID,
				SequenceID:   frame.SequenceID,
				Status:       actorwire.StatusOK,
			})
			continue
		}

		if err := b.routeFrame(frame); err != nil {
			b.recordError(err)
		}
	}
}

func (b *Broker) registerNode(nodeID uint16, conn net.Conn) (*nodeConn, bool) {
	b.mu.Lock()
	if existing := b.nodes[nodeID]; existing != nil && existing.conn != conn {
		b.report.DroppedFrames++
		b.mu.Unlock()
		reject := &nodeConn{nodeID: nodeID, conn: conn}
		_ = reject.write(actorwire.Frame{
			Type:         actorwire.FrameError,
			SourceNodeID: nodeID,
			DestNodeID:   nodeID,
			Status:       actorwire.StatusDuplicateNode,
		})
		_ = conn.Close()
		return nil, false
	}
	node := &nodeConn{nodeID: nodeID, conn: conn}
	b.nodes[nodeID] = node
	b.report.ConnectedNodes = len(b.nodes)
	b.mu.Unlock()
	return node, true
}

func (b *Broker) unregisterNode(nodeID uint16, conn net.Conn) {
	b.mu.Lock()
	if existing := b.nodes[nodeID]; existing != nil && existing.conn == conn {
		delete(b.nodes, nodeID)
		b.report.ConnectedNodes = len(b.nodes)
	}
	b.mu.Unlock()
}

func (b *Broker) routeFrame(frame actorwire.Frame) error {
	b.mu.Lock()
	dest := b.nodes[frame.DestNodeID]
	source := b.nodes[frame.SourceNodeID]
	if dest == nil {
		b.report.DroppedFrames++
		b.mu.Unlock()
		if source != nil {
			err := source.write(actorwire.Frame{
				Type:         actorwire.FrameNodeDown,
				SourceNodeID: frame.DestNodeID,
				DestNodeID:   frame.SourceNodeID,
				SequenceID:   frame.SequenceID,
				ActorID:      frame.ActorID,
				Status:       actorwire.StatusNodeUnavailable,
			})
			if b.handleClosedRouteWrite(frame.SourceNodeID, source.conn, err, false) {
				return nil
			}
			return err
		}
		return fmt.Errorf("actornet: destination node %d unavailable", frame.DestNodeID)
	}
	b.report.RoutedFrames++
	b.mu.Unlock()
	if err := dest.write(frame); err != nil {
		b.rollbackRoutedFrame()
		if b.handleClosedRouteWrite(frame.DestNodeID, dest.conn, err, true) {
			return nil
		}
		return err
	}
	return nil
}

func (b *Broker) rollbackRoutedFrame() {
	b.mu.Lock()
	if b.report.RoutedFrames > 0 {
		b.report.RoutedFrames--
	}
	b.mu.Unlock()
}

func (b *Broker) handleClosedRouteWrite(nodeID uint16, conn net.Conn, err error, countDrop bool) bool {
	if err == nil || !isClosedConnError(err) {
		return false
	}
	b.mu.Lock()
	if existing := b.nodes[nodeID]; existing != nil && existing.conn == conn {
		delete(b.nodes, nodeID)
	}
	if countDrop {
		b.report.DroppedFrames++
	}
	b.report.ConnectedNodes = len(b.nodes)
	b.mu.Unlock()
	_ = conn.Close()
	return true
}

func (b *Broker) recordAcceptedConnection() {
	b.mu.Lock()
	b.report.AcceptedConnections++
	b.mu.Unlock()
}

func (b *Broker) recordError(err error) {
	b.mu.Lock()
	if errors.Is(err, actorwire.ErrBadMagic) ||
		errors.Is(err, actorwire.ErrUnsupportedVersion) ||
		errors.Is(err, actorwire.ErrInvalidFrameType) ||
		errors.Is(err, actorwire.ErrInvalidNodeID) ||
		errors.Is(err, actorwire.ErrInvalidSlotCount) ||
		errors.Is(err, actorwire.ErrShortFrame) {
		b.report.DecodeErrors++
	}
	b.report.LastError = err.Error()
	b.mu.Unlock()
}

func (b *Broker) isClosed() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.closed
}

func (b *Broker) writeReportLocked() error {
	raw, err := json.MarshalIndent(b.report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(b.cfg.ReportPath, append(raw, '\n'), 0o644)
}

func (node *nodeConn) write(frame actorwire.Frame) error {
	data, err := actorwire.EncodeFrame(frame)
	if err != nil {
		return err
	}
	node.writeMu.Lock()
	defer node.writeMu.Unlock()
	for len(data) > 0 {
		n, err := node.conn.Write(data)
		if err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}

func readFrame(conn net.Conn) (actorwire.Frame, error) {
	data := make([]byte, actorwire.FrameSize)
	if _, err := io.ReadFull(conn, data); err != nil {
		return actorwire.Frame{}, err
	}
	return actorwire.DecodeFrame(data)
}

func isClosedConnError(err error) bool {
	return errors.Is(err, io.EOF) ||
		errors.Is(err, net.ErrClosed) ||
		errors.Is(err, io.ErrClosedPipe) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNRESET)
}
