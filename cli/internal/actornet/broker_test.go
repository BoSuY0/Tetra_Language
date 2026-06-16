package actornet

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
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
	waitForBrokerReport(t, broker, func(report Report) bool {
		return report.RoutedFrames == 1
	})

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

func waitForBrokerReport(t *testing.T, broker *Broker, done func(Report) bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if report := broker.Report(); done(report) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("broker report condition was not met before timeout: %+v", broker.Report())
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

func TestBrokerMissingDestinationNodeDownDoesNotRetryOrReconnect(t *testing.T) {
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
		Type:         actorwire.FrameSpawnReq,
		SourceNodeID: 1,
		DestNodeID:   2,
		SequenceID:   99,
		ActorID:      4,
	})

	got := readTestFrame(t, node1)
	if got.Type != actorwire.FrameNodeDown ||
		got.SourceNodeID != 2 ||
		got.DestNodeID != 1 ||
		got.SequenceID != 99 ||
		got.ActorID != 4 ||
		got.Status != actorwire.StatusNodeUnavailable {
		t.Fatalf("missing destination response = %+v, want single node_down for node 2", got)
	}
	if err := node1.SetReadDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
		t.Fatalf("setting short read deadline: %v", err)
	}
	buf := make([]byte, actorwire.FrameSize)
	if _, err := io.ReadFull(node1, buf); err == nil {
		t.Fatalf("broker emitted an unexpected second frame for one missing destination")
	} else if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
		t.Fatalf("reading optional second frame: %v, want timeout", err)
	}

	report := broker.Report()
	if report.AcceptedConnections != 1 {
		t.Fatalf("AcceptedConnections = %d, want one connection and no reconnect", report.AcceptedConnections)
	}
	if report.DroppedFrames != 1 {
		t.Fatalf("DroppedFrames = %d, want one missing-node frame", report.DroppedFrames)
	}
	if report.RoutedFrames != 0 {
		t.Fatalf("RoutedFrames = %d, want missing destination to stay status/failure evidence only", report.RoutedFrames)
	}
}

func TestBrokerRejectsDuplicateNodeWithFrameError(t *testing.T) {
	broker, stop := startTestBroker(t, Config{Addr: "127.0.0.1:0"})
	defer stop()

	primary := dialTestNode(t, broker.Addr())
	defer primary.Close()
	writeTestFrame(t, primary, actorwire.Frame{
		Type:         actorwire.FrameHello,
		SourceNodeID: 7,
		DestNodeID:   7,
	})
	if got := readTestFrame(t, primary); got.Type != actorwire.FrameHelloAck || got.Status != actorwire.StatusOK {
		t.Fatalf("primary hello ack = %+v, want hello_ack ok", got)
	}

	duplicate := dialTestNode(t, broker.Addr())
	defer duplicate.Close()
	writeTestFrame(t, duplicate, actorwire.Frame{
		Type:         actorwire.FrameHello,
		SourceNodeID: 7,
		DestNodeID:   7,
	})
	got := readTestFrame(t, duplicate)
	if got.Type != actorwire.FrameError ||
		got.SourceNodeID != 7 ||
		got.DestNodeID != 7 ||
		got.Status != actorwire.StatusDuplicateNode {
		t.Fatalf("duplicate node response = %+v, want frame_error duplicate_node", got)
	}
	waitForBrokerReport(t, broker, func(report Report) bool {
		return report.DroppedFrames == 1 && report.ConnectedNodes == 1
	})
}

func TestBrokerRejectsForgedSourceNode(t *testing.T) {
	broker, stop := startTestBroker(t, Config{Addr: "127.0.0.1:0"})
	defer stop()

	source := dialTestNode(t, broker.Addr())
	defer source.Close()
	writeTestFrame(t, source, actorwire.Frame{
		Type:         actorwire.FrameHello,
		SourceNodeID: 1,
		DestNodeID:   1,
	})
	if got := readTestFrame(t, source); got.Type != actorwire.FrameHelloAck || got.Status != actorwire.StatusOK {
		t.Fatalf("source hello ack = %+v, want hello_ack ok", got)
	}

	dest := dialTestNode(t, broker.Addr())
	defer dest.Close()
	writeTestFrame(t, dest, actorwire.Frame{
		Type:         actorwire.FrameHello,
		SourceNodeID: 2,
		DestNodeID:   2,
	})
	if got := readTestFrame(t, dest); got.Type != actorwire.FrameHelloAck || got.Status != actorwire.StatusOK {
		t.Fatalf("dest hello ack = %+v, want hello_ack ok", got)
	}

	writeTestFrame(t, source, actorwire.Frame{
		Type:         actorwire.FrameSendI32,
		SourceNodeID: 2,
		DestNodeID:   2,
		SequenceID:   41,
		Payload:      []int32{99},
	})
	got := readTestFrame(t, source)
	if got.Type != actorwire.FrameError ||
		got.SourceNodeID != 1 ||
		got.DestNodeID != 1 ||
		got.SequenceID != 41 ||
		got.Status != actorwire.StatusDecodeError {
		t.Fatalf("forged-source response = %+v, want frame_error decode_error for registered node", got)
	}
	if err := dest.SetReadDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
		t.Fatalf("setting dest read deadline: %v", err)
	}
	buf := make([]byte, actorwire.FrameSize)
	if _, err := io.ReadFull(dest, buf); err == nil {
		t.Fatalf("broker routed forged-source frame to destination")
	} else if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
		t.Fatalf("reading destination after forged frame: %v, want timeout", err)
	}
	waitForBrokerReport(t, broker, func(report Report) bool {
		return report.DroppedFrames == 1 && report.RoutedFrames == 0
	})
}

func TestBrokerRecordsMalformedFrameDecodeErrors(t *testing.T) {
	broker, stop := startTestBroker(t, Config{Addr: "127.0.0.1:0"})
	defer stop()

	for index, tc := range []struct {
		name   string
		mutate func([]byte) []byte
	}{
		{
			name: "malformed frame length",
			mutate: func(raw []byte) []byte {
				return raw[:actorwire.FrameSize-1]
			},
		},
		{
			name: "unknown frame type",
			mutate: func(raw []byte) []byte {
				binary.LittleEndian.PutUint16(raw[actorwire.FrameTypeOffset:], 0xffff)
				return raw
			},
		},
		{
			name: "bad typed slot count",
			mutate: func(raw []byte) []byte {
				binary.LittleEndian.PutUint16(raw[actorwire.FrameTypeOffset:], uint16(actorwire.FrameSendTyped))
				binary.LittleEndian.PutUint16(raw[actorwire.FrameSlotCountOffset:], actorwire.MaxPayloadSlots+1)
				return raw
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conn := dialTestNode(t, broker.Addr())
			raw := encodeTestFrameRaw(t, actorwire.Frame{
				Type:         actorwire.FrameSendI32,
				SourceNodeID: 1,
				DestNodeID:   1,
				SequenceID:   uint32(index + 1),
				Payload:      []int32{1},
			})
			writeRawTestFrame(t, conn, tc.mutate(raw))
			_ = conn.Close()
			waitForBrokerReport(t, broker, func(report Report) bool {
				return report.DecodeErrors == int64(index+1)
			})
		})
	}

	report := broker.Report()
	if report.DecodeErrors != 3 {
		t.Fatalf("DecodeErrors = %d, want 3", report.DecodeErrors)
	}
	if !strings.Contains(report.LastError, "actor wire") {
		t.Fatalf("LastError = %q, want actor wire decode error", report.LastError)
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

func TestBrokerCloseWithoutCancelStopsServeWatcher(t *testing.T) {
	broker, err := NewBroker(Config{Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("NewBroker: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		done <- broker.Serve(context.Background())
	}()

	waitForBrokerServeWatchers(t, 1)
	if err := broker.Close(); err != nil {
		t.Fatalf("broker close: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("broker serve: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("broker did not stop after Close without context cancellation")
	}
	waitForBrokerServeWatchers(t, 0)
}

func TestBrokerCloseReopenWithoutGoroutineLeak(t *testing.T) {
	waitForBrokerServeWatchers(t, 0)
	baseline := countBrokerServeWatchers()
	const cycles = 5

	for cycle := 0; cycle < cycles; cycle++ {
		broker, err := NewBroker(Config{Addr: "127.0.0.1:0"})
		if err != nil {
			t.Fatalf("cycle %d NewBroker: %v", cycle, err)
		}
		done := make(chan error, 1)
		go func() {
			done <- broker.Serve(context.Background())
		}()

		waitForBrokerServeWatchers(t, baseline+1)
		node := dialTestNode(t, broker.Addr())
		writeTestFrame(t, node, actorwire.Frame{
			Type:         actorwire.FrameHello,
			SourceNodeID: 1,
			DestNodeID:   1,
		})
		if got := readTestFrame(t, node); got.Type != actorwire.FrameHelloAck || got.DestNodeID != 1 {
			_ = node.Close()
			_ = broker.Close()
			t.Fatalf("cycle %d hello ack = %+v, want hello_ack for node 1", cycle, got)
		}

		if err := broker.Close(); err != nil {
			_ = node.Close()
			t.Fatalf("cycle %d broker close: %v", cycle, err)
		}
		_ = node.Close()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("cycle %d broker serve: %v", cycle, err)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("cycle %d broker did not stop after Close without context cancellation", cycle)
		}
		waitForBrokerServeWatchers(t, baseline)
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

func waitForBrokerServeWatchers(t *testing.T, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if got := countBrokerServeWatchers(); got == want {
			return
		} else if time.Now().After(deadline) {
			t.Fatalf("Broker.Serve watcher goroutines = %d, want %d", got, want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func countBrokerServeWatchers() int {
	var buf bytes.Buffer
	if err := pprof.Lookup("goroutine").WriteTo(&buf, 2); err != nil {
		return -1
	}
	return strings.Count(buf.String(), "tetra_language/cli/internal/actornet.(*Broker).Serve.func1")
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

func encodeTestFrameRaw(t *testing.T, frame actorwire.Frame) []byte {
	t.Helper()
	data, err := actorwire.EncodeFrame(frame)
	if err != nil {
		t.Fatalf("encoding frame %+v: %v", frame, err)
	}
	return data
}

func writeRawTestFrame(t *testing.T, conn net.Conn, data []byte) {
	t.Helper()
	if err := conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("setting write deadline: %v", err)
	}
	if _, err := conn.Write(data); err != nil {
		t.Fatalf("writing raw frame: %v", err)
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
