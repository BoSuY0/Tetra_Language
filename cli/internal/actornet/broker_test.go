package actornet

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"tetra_language/compiler/actorwire"
)

func TestBrokerRoutesFramesBetweenLoopbackNodesAndWritesReport(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "actornet-report.json")
	broker, stop := startTestBroker(t, Config{
		Addr:       "127.0.0.1:0",
		ReportPath: reportPath,
	})
	defer stop()

	node1 := dialTestNode(t, broker.Addr())
	defer node1.Close()
	node2 := dialTestNode(t, broker.Addr())
	defer node2.Close()

	writeTestFrame(t, node1, actorwire.Frame{
		Type:         actorwire.FrameHello,
		SourceNodeID: 1,
		DestNodeID:   1,
	})
	if got := readTestFrame(t, node1); got.Type != actorwire.FrameHelloAck || got.DestNodeID != 1 {
		t.Fatalf("node1 hello ack = %+v, want hello_ack for node 1", got)
	}

	writeTestFrame(t, node2, actorwire.Frame{
		Type:         actorwire.FrameHello,
		SourceNodeID: 2,
		DestNodeID:   2,
	})
	if got := readTestFrame(t, node2); got.Type != actorwire.FrameHelloAck || got.DestNodeID != 2 {
		t.Fatalf("node2 hello ack = %+v, want hello_ack for node 2", got)
	}

	want := actorwire.Frame{
		Type:         actorwire.FrameSendTyped,
		SourceNodeID: 1,
		DestNodeID:   2,
		SequenceID:   12,
		ActorID:      7,
		Tag:          44,
		Payload:      []int32{10, 20, 30},
	}
	writeTestFrame(t, node1, want)

	got := readTestFrame(t, node2)
	assertFrame(t, got, want)

	stop()

	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("reading report: %v", err)
	}
	var report Report
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decoding report: %v", err)
	}
	if report.Runtime != "actornet" || report.Transport != "loopback-tcp" {
		t.Fatalf("report identity = %q/%q, want actornet/loopback-tcp", report.Runtime, report.Transport)
	}
	if report.RoutedFrames != 1 || report.AcceptedConnections != 2 {
		t.Fatalf("report counts = routed %d accepted %d, want 1/2", report.RoutedFrames, report.AcceptedConnections)
	}
}

func TestBrokerReportsNodeDownForMissingDestination(t *testing.T) {
	broker, stop := startTestBroker(t, Config{Addr: "127.0.0.1:0"})
	defer stop()

	node1 := dialTestNode(t, broker.Addr())
	defer node1.Close()

	writeTestFrame(t, node1, actorwire.Frame{
		Type:         actorwire.FrameHello,
		SourceNodeID: 1,
		DestNodeID:   1,
	})
	if got := readTestFrame(t, node1); got.Type != actorwire.FrameHelloAck {
		t.Fatalf("hello ack = %+v, want hello_ack", got)
	}

	writeTestFrame(t, node1, actorwire.Frame{
		Type:         actorwire.FrameSendI32,
		SourceNodeID: 1,
		DestNodeID:   2,
		SequenceID:   88,
		ActorID:      3,
		Payload:      []int32{99},
	})

	got := readTestFrame(t, node1)
	if got.Type != actorwire.FrameNodeDown ||
		got.SourceNodeID != 2 ||
		got.DestNodeID != 1 ||
		got.SequenceID != 88 ||
		got.ActorID != 3 ||
		got.Status != actorwire.StatusNodeUnavailable {
		t.Fatalf("missing destination response = %+v, want node_down for node 2", got)
	}
	if report := broker.Report(); report.DroppedFrames != 1 {
		t.Fatalf("DroppedFrames = %d, want 1", report.DroppedFrames)
	}
}

func TestBrokerTreatsClosedDestinationWriteAsDroppedFrame(t *testing.T) {
	destWriter, destReader := net.Pipe()
	defer destWriter.Close()
	_ = destReader.Close()

	sourceWriter, sourceReader := net.Pipe()
	defer sourceWriter.Close()
	defer sourceReader.Close()

	broker := &Broker{
		nodes: map[uint16]*nodeConn{
			1: {nodeID: 1, conn: sourceWriter},
			2: {nodeID: 2, conn: destWriter},
		},
	}

	err := broker.routeFrame(actorwire.Frame{
		Type:         actorwire.FrameSendI32,
		SourceNodeID: 1,
		DestNodeID:   2,
		SequenceID:   91,
		Payload:      []int32{4},
	})
	if err != nil {
		t.Fatalf("routeFrame closed destination error = %v, want nil", err)
	}
	report := broker.Report()
	if report.LastError != "" {
		t.Fatalf("LastError = %q, want empty for closed destination write", report.LastError)
	}
	if report.DroppedFrames != 1 {
		t.Fatalf("DroppedFrames = %d, want 1", report.DroppedFrames)
	}
	if report.RoutedFrames != 0 {
		t.Fatalf("RoutedFrames = %d, want 0 for undelivered frame", report.RoutedFrames)
	}
	if report.ConnectedNodes != 1 {
		t.Fatalf("ConnectedNodes = %d, want stale destination removed", report.ConnectedNodes)
	}
}

func TestClosedConnectionErrorsIncludePeerReset(t *testing.T) {
	err := &net.OpError{Op: "read", Net: "tcp", Err: os.NewSyscallError("read", syscall.ECONNRESET)}
	if !isClosedConnError(err) {
		t.Fatalf("peer reset should be treated as a closed connection")
	}
}

func startTestBroker(t *testing.T, cfg Config) (*Broker, func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	broker, err := NewBroker(cfg)
	if err != nil {
		t.Fatalf("NewBroker: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		done <- broker.Serve(ctx)
	}()

	var once sync.Once
	stop := func() {
		once.Do(func() {
			cancel()
			if err := broker.Close(); err != nil {
				t.Fatalf("broker close: %v", err)
			}
			select {
			case err := <-done:
				if err != nil && !errors.Is(err, context.Canceled) {
					t.Fatalf("broker serve: %v", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatalf("broker did not stop")
			}
		})
	}
	return broker, stop
}

func dialTestNode(t *testing.T, addr string) net.Conn {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatalf("dialing %s: %v", addr, err)
	}
	return conn
}

func writeTestFrame(t *testing.T, conn net.Conn, frame actorwire.Frame) {
	t.Helper()
	data, err := actorwire.EncodeFrame(frame)
	if err != nil {
		t.Fatalf("encoding frame %+v: %v", frame, err)
	}
	if err := conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("setting write deadline: %v", err)
	}
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("writing frame: %v", err)
	}
}

func readTestFrame(t *testing.T, conn net.Conn) actorwire.Frame {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("setting read deadline: %v", err)
	}
	data := make([]byte, actorwire.FrameSize)
	if _, err := io.ReadFull(conn, data); err != nil {
		t.Fatalf("reading frame: %v", err)
	}
	frame, err := actorwire.DecodeFrame(data)
	if err != nil {
		t.Fatalf("decoding frame: %v", err)
	}
	return frame
}

func assertFrame(t *testing.T, got actorwire.Frame, want actorwire.Frame) {
	t.Helper()
	if got.Type != want.Type ||
		got.SourceNodeID != want.SourceNodeID ||
		got.DestNodeID != want.DestNodeID ||
		got.SequenceID != want.SequenceID ||
		got.ActorID != want.ActorID ||
		got.Tag != want.Tag ||
		got.Status != want.Status {
		t.Fatalf("frame header = %+v, want %+v", got, want)
	}
	if len(got.Payload) != len(want.Payload) {
		t.Fatalf("payload length = %d, want %d", len(got.Payload), len(want.Payload))
	}
	for i := range got.Payload {
		if got.Payload[i] != want.Payload[i] {
			t.Fatalf("payload[%d] = %d, want %d", i, got.Payload[i], want.Payload[i])
		}
	}
}
