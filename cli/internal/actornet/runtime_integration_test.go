package actornet

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"tetra_language/compiler"
	"tetra_language/compiler/actorwire"
)

func TestLinuxRuntimeConnectsToLoopbackBroker(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 executable smoke only")
	}

	broker, stop := startTestBroker(t, Config{Addr: "127.0.0.1:0"})
	defer stop()

	_, portRaw, err := net.SplitHostPort(broker.Addr())
	if err != nil {
		t.Fatalf("split broker address: %v", err)
	}
	port, err := strconv.Atoi(portRaw)
	if err != nil {
		t.Fatalf("parse broker port: %v", err)
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "connect_broker.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses actors, runtime:
    let connected: Int = core.actor_node_connect(1, `+strconv.Itoa(port)+`)
    let status: Int = core.actor_node_status(1)
    return connected + status
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	outPath := filepath.Join(tmp, "connect-broker")
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build linux runtime smoke: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("runtime smoke timed out")
	}
	if err != nil {
		t.Fatalf("runtime smoke failed: %v output=%q", err, string(output))
	}
	if report := broker.Report(); report.AcceptedConnections != 1 {
		t.Fatalf("broker accepted connections = %d, want 1", report.AcceptedConnections)
	}
}

func TestLinuxRuntimeConnectWritesHelloFrame(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 executable smoke only")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	_, portRaw, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split listener address: %v", err)
	}
	port, err := strconv.Atoi(portRaw)
	if err != nil {
		t.Fatalf("parse listener port: %v", err)
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "connect_raw.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses actors, runtime:
    return core.actor_node_connect(1, `+strconv.Itoa(port)+`)
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "connect-raw")
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build linux runtime smoke: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	done := make(chan error, 1)
	go func() {
		output, err := cmd.CombinedOutput()
		if err != nil {
			done <- &runtimeSmokeError{err: err, output: string(output)}
			return
		}
		done <- nil
	}()

	conn, err := listener.Accept()
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	defer conn.Close()
	got := readTestFrame(t, conn)
	if got.Type != actorwire.FrameHello || got.SourceNodeID != 1 || got.DestNodeID != 1 {
		t.Fatalf("hello frame = %+v, want hello from node 1", got)
	}
	writeTestFrame(t, conn, helloAckFrame(1))

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runtime smoke failed: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("runtime smoke timed out")
	}
}

func TestLinuxRuntimeRoutesRemoteHandleSendToBroker(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 executable smoke only")
	}

	broker, stop := startTestBroker(t, Config{Addr: "127.0.0.1:0"})
	defer stop()

	_, portRaw, err := net.SplitHostPort(broker.Addr())
	if err != nil {
		t.Fatalf("split broker address: %v", err)
	}
	port, err := strconv.Atoi(portRaw)
	if err != nil {
		t.Fatalf("parse broker port: %v", err)
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "remote_send_broker.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func worker() -> Int:
    return 0

func main() -> Int
uses actors, runtime:
    let _connected: Int = core.actor_node_connect(1, `+strconv.Itoa(port)+`)
    let peer: actor = core.spawn_remote(2, "worker")
    let _sent: Int = core.send(peer, 7)
    let _tagged: Int = core.send_msg(peer, 8, 99)
    return 0
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	outPath := filepath.Join(tmp, "remote-send-broker")
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build linux runtime smoke: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("runtime smoke timed out")
	}
	if err != nil {
		t.Fatalf("runtime smoke failed: %v output=%q", err, string(output))
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		if report := broker.Report(); report.DroppedFrames > 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("broker did not observe remote frames; report=%+v", broker.Report())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestLinuxRuntimeRoutesRemoteTypedSendToPeer(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 executable smoke only")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	_, portRaw, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split listener address: %v", err)
	}
	port, err := strconv.Atoi(portRaw)
	if err != nil {
		t.Fatalf("parse broker port: %v", err)
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "remote_typed_send_peer.tetra")
	if err := os.WriteFile(srcPath, []byte(`
enum RemoteMsg:
    case ping(Int)

func worker() -> Int:
    return 0

func main() -> Int
uses actors, runtime:
    let connected: Int = core.actor_node_connect(1, `+strconv.Itoa(port)+`)
    if connected != 0:
        return 10 + connected
    let peer: actor = core.spawn_remote(2, "worker")
    let typed: Int = core.send_typed(peer, RemoteMsg.ping(11))
    if typed == 0:
        return 0
    return 20 + typed
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	outPath := filepath.Join(tmp, "remote-typed-send-peer")
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build linux runtime smoke: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	done := make(chan error, 1)
	go func() {
		output, err := cmd.CombinedOutput()
		if err != nil {
			done <- &runtimeSmokeError{err: err, output: string(output)}
			return
		}
		done <- nil
	}()

	conn, err := listener.Accept()
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	defer conn.Close()
	if got := readTestFrame(t, conn); got.Type != actorwire.FrameHello || got.SourceNodeID != 1 {
		t.Fatalf("hello frame = %+v, want hello from node 1", got)
	}
	writeTestFrame(t, conn, helloAckFrame(1))
	spawn := readTestFrame(t, conn)
	if spawn.Type != actorwire.FrameSpawnReq || spawn.SourceNodeID != 1 || spawn.DestNodeID != 2 {
		t.Fatalf("spawn frame = %+v, want spawn_req from node 1 to node 2", spawn)
	}
	typed := readTestFrame(t, conn)
	if typed.Type != actorwire.FrameSendTyped ||
		typed.SourceNodeID != 1 ||
		typed.DestNodeID != 2 ||
		typed.ActorID == 0 ||
		typed.Tag != 0 ||
		len(typed.Payload) != 1 ||
		typed.Payload[0] != 11 {
		t.Fatalf("typed frame = %+v, want send_typed ping(11) to node 2", typed)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runtime smoke failed: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("runtime smoke timed out")
	}
}

func TestLinuxRuntimePumpsRemoteSendIntoRecv(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 executable smoke only")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	_, portRaw, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split listener address: %v", err)
	}
	port, err := strconv.Atoi(portRaw)
	if err != nil {
		t.Fatalf("parse listener port: %v", err)
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "remote_recv.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses actors, runtime:
    let _connected: Int = core.actor_node_connect(1, `+strconv.Itoa(port)+`)
    let msg: Int = core.recv()
    if msg != 42:
        return 80 + msg
    return 42
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	outPath := filepath.Join(tmp, "remote-recv")
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build linux runtime smoke: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	done := make(chan struct {
		exitCode int
		output   string
		err      error
	}, 1)
	go func() {
		output, err := cmd.CombinedOutput()
		result := struct {
			exitCode int
			output   string
			err      error
		}{output: string(output)}
		if err == nil {
			done <- result
			return
		}
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			result.err = err
			done <- result
			return
		}
		result.exitCode = exitErr.ExitCode()
		done <- result
	}()

	conn, err := listener.Accept()
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	defer conn.Close()
	if got := readTestFrame(t, conn); got.Type != actorwire.FrameHello || got.SourceNodeID != 1 {
		t.Fatalf("hello frame = %+v, want hello from node 1", got)
	}
	writeTestFrames(t, conn,
		helloAckFrame(1),
		actorwire.Frame{
			Type:         actorwire.FrameSendI32,
			SourceNodeID: 2,
			DestNodeID:   1,
			SequenceID:   1,
			ActorID:      0,
			Payload:      []int32{42},
		},
	)

	select {
	case result := <-done:
		if result.err != nil {
			t.Fatalf("runtime smoke failed: %v output=%q", result.err, result.output)
		}
		if result.exitCode != 42 {
			t.Fatalf("runtime smoke exit code = %d output=%q, want recv payload 42", result.exitCode, result.output)
		}
	case <-ctx.Done():
		t.Fatalf("runtime smoke timed out")
	}
}

func TestLinuxRuntimePumpsRemoteTypedFrameIntoRecvMsg(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 executable smoke only")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	_, portRaw, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split listener address: %v", err)
	}
	port, err := strconv.Atoi(portRaw)
	if err != nil {
		t.Fatalf("parse listener port: %v", err)
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "remote_recv_typed_frame_msg.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses actors, runtime:
    let connected: Int = core.actor_node_connect(1, `+strconv.Itoa(port)+`)
    if connected != 0:
        return 10 + connected
    var raw: actor.msg = core.recv_msg()
    return raw.tag
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	outPath := filepath.Join(tmp, "remote-recv-typed-frame-msg")
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build linux runtime smoke: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	done := make(chan struct {
		exitCode int
		output   string
		err      error
	}, 1)
	go func() {
		output, err := cmd.CombinedOutput()
		result := struct {
			exitCode int
			output   string
			err      error
		}{output: string(output)}
		if err == nil {
			done <- result
			return
		}
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			result.err = err
			done <- result
			return
		}
		result.exitCode = exitErr.ExitCode()
		done <- result
	}()

	conn, err := listener.Accept()
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	defer conn.Close()
	if got := readTestFrame(t, conn); got.Type != actorwire.FrameHello || got.SourceNodeID != 1 {
		t.Fatalf("hello frame = %+v, want hello from node 1", got)
	}
	typedFrame := actorwire.Frame{
		Type:         actorwire.FrameSendTyped,
		SourceNodeID: 2,
		DestNodeID:   1,
		SequenceID:   1,
		ActorID:      0,
		Tag:          42,
		Payload:      []int32{7},
	}
	writeTestFrames(t, conn, helloAckFrame(1), typedFrame)

	select {
	case result := <-done:
		if result.err != nil {
			t.Fatalf("runtime smoke failed: %v output=%q", result.err, result.output)
		}
		if result.exitCode != 42 {
			t.Fatalf("runtime smoke exit code = %d output=%q, want typed frame tag 42", result.exitCode, result.output)
		}
	case <-ctx.Done():
		t.Fatalf("runtime smoke timed out")
	}
}

func TestLinuxRuntimePumpsRemoteTypedFrameIntoRecvTypedTagOnly(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 executable smoke only")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	_, portRaw, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split listener address: %v", err)
	}
	port, err := strconv.Atoi(portRaw)
	if err != nil {
		t.Fatalf("parse listener port: %v", err)
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "remote_recv_typed_frame_tag_only.tetra")
	if err := os.WriteFile(srcPath, []byte(`
enum RemoteMsg:
    case ping
    case reset

func main() -> Int
uses actors, runtime:
    let connected: Int = core.actor_node_connect(1, `+strconv.Itoa(port)+`)
    if connected != 0:
        return 10 + connected
    let msg: RemoteMsg = core.recv_typed<RemoteMsg>()
    match msg:
    case RemoteMsg.ping:
        return 42
    case RemoteMsg.reset:
        return 90
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	outPath := filepath.Join(tmp, "remote-recv-typed-frame-tag-only")
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build linux runtime smoke: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	done := make(chan struct {
		exitCode int
		output   string
		err      error
	}, 1)
	go func() {
		output, err := cmd.CombinedOutput()
		result := struct {
			exitCode int
			output   string
			err      error
		}{output: string(output)}
		if err == nil {
			done <- result
			return
		}
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			result.err = err
			done <- result
			return
		}
		result.exitCode = exitErr.ExitCode()
		done <- result
	}()

	conn, err := listener.Accept()
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	defer conn.Close()
	if got := readTestFrame(t, conn); got.Type != actorwire.FrameHello || got.SourceNodeID != 1 {
		t.Fatalf("hello frame = %+v, want hello from node 1", got)
	}
	writeTestFrames(t, conn,
		helloAckFrame(1),
		actorwire.Frame{
			Type:         actorwire.FrameSendTyped,
			SourceNodeID: 2,
			DestNodeID:   1,
			SequenceID:   1,
			ActorID:      0,
			Tag:          0,
		},
	)

	select {
	case result := <-done:
		if result.err != nil {
			t.Fatalf("runtime smoke failed: %v output=%q", result.err, result.output)
		}
		if result.exitCode != 42 {
			t.Fatalf("runtime smoke exit code = %d output=%q, want typed tag-only 42", result.exitCode, result.output)
		}
	case <-ctx.Done():
		t.Fatalf("runtime smoke timed out")
	}
}

func TestLinuxRuntimePumpsRemoteTypedFrameIntoRecvTyped(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 executable smoke only")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	_, portRaw, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split listener address: %v", err)
	}
	port, err := strconv.Atoi(portRaw)
	if err != nil {
		t.Fatalf("parse listener port: %v", err)
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "remote_recv_typed_frame.tetra")
	if err := os.WriteFile(srcPath, []byte(`
enum RemoteMsg:
    case ping(Int, Int)
    case reset

func main() -> Int
uses actors, runtime:
    let connected: Int = core.actor_node_connect(1, `+strconv.Itoa(port)+`)
    if connected != 0:
        return 10 + connected
    let msg: RemoteMsg = core.recv_typed<RemoteMsg>()
    match msg:
    case RemoteMsg.ping(lhs, rhs):
        return lhs + rhs
    case RemoteMsg.reset:
        return 90
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	outPath := filepath.Join(tmp, "remote-recv-typed-frame")
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build linux runtime smoke: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	done := make(chan struct {
		exitCode int
		output   string
		err      error
	}, 1)
	go func() {
		output, err := cmd.CombinedOutput()
		result := struct {
			exitCode int
			output   string
			err      error
		}{output: string(output)}
		if err == nil {
			done <- result
			return
		}
		exitErr, ok := err.(*exec.ExitError)
		if !ok {
			result.err = err
			done <- result
			return
		}
		result.exitCode = exitErr.ExitCode()
		done <- result
	}()

	conn, err := listener.Accept()
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	defer conn.Close()
	if got := readTestFrame(t, conn); got.Type != actorwire.FrameHello || got.SourceNodeID != 1 {
		t.Fatalf("hello frame = %+v, want hello from node 1", got)
	}
	writeTestFrames(t, conn,
		helloAckFrame(1),
		actorwire.Frame{
			Type:         actorwire.FrameSendTyped,
			SourceNodeID: 2,
			DestNodeID:   1,
			SequenceID:   1,
			ActorID:      0,
			Tag:          0,
			Payload:      []int32{20, 22},
		},
	)

	select {
	case result := <-done:
		if result.err != nil {
			t.Fatalf("runtime smoke failed: %v output=%q", result.err, result.output)
		}
		if result.exitCode != 42 {
			t.Fatalf("runtime smoke exit code = %d output=%q, want typed payload sum 42", result.exitCode, result.output)
		}
	case <-ctx.Done():
		t.Fatalf("runtime smoke timed out")
	}
}

func TestLinuxRuntimePumpsNodeDownIntoNodeStatus(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 executable smoke only")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	_, portRaw, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split listener address: %v", err)
	}
	port, err := strconv.Atoi(portRaw)
	if err != nil {
		t.Fatalf("parse listener port: %v", err)
	}

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "remote_node_down.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses actors, runtime:
    let connected: Int = core.actor_node_connect(1, `+strconv.Itoa(port)+`)
    if connected != 0:
        return 10 + connected
    let before: Int = core.actor_node_status(1)
    if before != 0:
        return 20 + before
    let msg: Int = core.recv()
    if msg != 7:
        return 40 + msg
    let _poll: actor.recv_result_i32 = core.recv_poll()
    let after: Int = core.actor_node_status(1)
    if after == 1:
        return 0
    return 30 + after
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	outPath := filepath.Join(tmp, "remote-node-down")
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build linux runtime smoke: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, outPath)
	done := make(chan error, 1)
	go func() {
		output, err := cmd.CombinedOutput()
		if err != nil {
			done <- &runtimeSmokeError{err: err, output: string(output)}
			return
		}
		done <- nil
	}()

	conn, err := listener.Accept()
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	defer conn.Close()
	if got := readTestFrame(t, conn); got.Type != actorwire.FrameHello || got.SourceNodeID != 1 {
		t.Fatalf("hello frame = %+v, want hello from node 1", got)
	}
	writeTestFrames(t, conn,
		helloAckFrame(1),
		actorwire.Frame{
			Type:         actorwire.FrameSendI32,
			SourceNodeID: 2,
			DestNodeID:   1,
			SequenceID:   1,
			Payload:      []int32{7},
		},
		actorwire.Frame{
			Type:         actorwire.FrameNodeDown,
			SourceNodeID: 2,
			DestNodeID:   1,
			SequenceID:   2,
			Status:       actorwire.StatusNodeUnavailable,
		},
	)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runtime smoke failed: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("runtime smoke timed out")
	}
}

type runtimeSmokeError struct {
	err    error
	output string
}

func (e *runtimeSmokeError) Error() string {
	return e.err.Error() + " output=" + strconv.Quote(e.output)
}

func helloAckFrame(nodeID uint16) actorwire.Frame {
	return actorwire.Frame{
		Type:         actorwire.FrameHelloAck,
		SourceNodeID: nodeID,
		DestNodeID:   nodeID,
		Status:       actorwire.StatusOK,
	}
}

func writeTestFrames(t *testing.T, conn net.Conn, frames ...actorwire.Frame) {
	t.Helper()
	var data []byte
	for _, frame := range frames {
		encoded, err := actorwire.EncodeFrame(frame)
		if err != nil {
			t.Fatalf("encoding frame %+v: %v", frame, err)
		}
		data = append(data, encoded...)
	}
	if err := conn.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("setting write deadline: %v", err)
	}
	for len(data) > 0 {
		n, err := conn.Write(data)
		if err != nil {
			t.Fatalf("writing frames: %v", err)
		}
		data = data[n:]
	}
}
