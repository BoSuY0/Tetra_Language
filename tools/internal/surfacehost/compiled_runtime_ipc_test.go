package surfacehost

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"tetra_language/compiler"
)

func TestCompiledLinuxSurfaceHostIPCRepeatedPollPresentRedraw(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux runtime smoke requires linux")
	}

	dir := t.TempDir()
	socketDir := shortSocketDir(t)
	socketPath := filepath.Join(socketDir, "host.sock")
	sourcePath := filepath.Join(dir, "surface_ipc_probe.tetra")
	binaryPath := filepath.Join(dir, "surface-ipc-probe")

	source := `module surface_ipc_probe

func main() -> Int
uses surface, mem, alloc:
    let handle: Int = core.surface_open("IPC Probe", 320, 200)
    var total: Int = 0
    var i: Int = 0
    while i < 4:
        var events: []i32 = core.make_i32(9)
        let copied: Int = core.surface_poll_event_into(handle, events)
        var pixels: []u8 = core.make_u8(320 * 200 * 4)
        let presented: Int = core.surface_present_rgba(handle, pixels, 320, 200, 320 * 4)
        let redraw: Int = core.surface_request_redraw(handle)
        total = total + copied + presented + redraw
        i = i + 1
    let closed: Int = core.surface_close(handle)
    return total + closed
`
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("write probe source: %v", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen fake surface host: %v", err)
	}
	defer listener.Close()

	serverDone := make(chan compiledIPCServerResult, 1)
	go runCompiledIPCFakeHost(listener, serverDone)

	_, err = compiler.BuildFileWithStatsOpt(sourcePath, binaryPath, "linux-x64", compiler.BuildOptions{
		SurfaceHostRequired:   true,
		SurfaceHostDriver:     "wayland",
		SurfaceHostProtocol:   ProtocolName,
		SurfaceHostSocketPath: socketPath,
	})
	if err != nil {
		t.Fatalf("build probe: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binaryPath)
	output, runErr := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("compiled surface probe timed out; output:\n%s", output)
	}

	var result compiledIPCServerResult
	select {
	case result = <-serverDone:
	case <-time.After(2 * time.Second):
		t.Fatal("fake host did not finish after compiled app exited")
	}
	if code, ok := exitCode(runErr); !ok || code != 44 {
		t.Fatalf(
			"run compiled surface probe: %v\nops: %v\nhost error: %v\n%s",
			runErr,
			result.ops,
			result.err,
			output,
		)
	}
	if result.err != nil {
		t.Fatalf("fake host failed after ops %v: %v", result.ops, result.err)
	}

	wantOps := []Op{OpOpen}
	for i := 0; i < 4; i++ {
		wantOps = append(wantOps, OpPollEventInto, OpPresentRGBA, OpRequestRedraw)
	}
	wantOps = append(wantOps, OpClose)
	if !reflect.DeepEqual(result.ops, wantOps) {
		t.Fatalf("unexpected op stream\nwant: %v\n got: %v", wantOps, result.ops)
	}
}

func TestCompiledLinuxSurfaceHostIPCWaitsForPollResponseBeforePresent(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux runtime smoke requires linux")
	}

	dir := t.TempDir()
	socketDir := shortSocketDir(t)
	socketPath := filepath.Join(socketDir, "host.sock")
	sourcePath := filepath.Join(dir, "surface_ipc_sync_probe.tetra")
	binaryPath := filepath.Join(dir, "surface-ipc-sync-probe")

	source := `module surface_ipc_sync_probe

func main() -> Int
uses surface, mem, alloc:
    let handle: Int = core.surface_open("IPC Sync Probe", 320, 200)
    var events: []i32 = core.make_i32(9)
    let copied: Int = core.surface_poll_event_into(handle, events)
    var pixels: []u8 = core.make_u8(320 * 200 * 4)
    let presented: Int = core.surface_present_rgba(handle, pixels, 320, 200, 320 * 4)
    let redraw: Int = core.surface_request_redraw(handle)
    let closed: Int = core.surface_close(handle)
    return copied + presented + redraw + closed
`
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("write sync probe source: %v", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen fake surface host: %v", err)
	}
	defer listener.Close()

	serverDone := make(chan compiledIPCServerResult, 1)
	go runCompiledIPCDelayedPollFakeHost(listener, serverDone)

	_, err = compiler.BuildFileWithStatsOpt(sourcePath, binaryPath, "linux-x64", compiler.BuildOptions{
		SurfaceHostRequired:   true,
		SurfaceHostDriver:     "wayland",
		SurfaceHostProtocol:   ProtocolName,
		SurfaceHostSocketPath: socketPath,
	})
	if err != nil {
		t.Fatalf("build sync probe: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binaryPath)
	output, runErr := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("compiled surface sync probe timed out; output:\n%s", output)
	}

	var result compiledIPCServerResult
	select {
	case result = <-serverDone:
	case <-time.After(2 * time.Second):
		t.Fatal("fake delayed host did not finish after compiled app exited")
	}
	if code, ok := exitCode(runErr); !ok || code != 11 {
		t.Fatalf(
			"run compiled surface sync probe: %v\nops: %v\nhost error: %v\n%s",
			runErr,
			result.ops,
			result.err,
			output,
		)
	}
	if result.err != nil {
		t.Fatalf("fake delayed host failed after ops %v: %v", result.ops, result.err)
	}
}

func TestCompiledLinuxSurfaceHostIPCTextClipboardAndComposition(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux runtime smoke requires linux")
	}

	dir := t.TempDir()
	socketDir := shortSocketDir(t)
	socketPath := filepath.Join(socketDir, "host.sock")
	sourcePath := filepath.Join(dir, "surface_ipc_text_probe.tetra")
	binaryPath := filepath.Join(dir, "surface-ipc-text-probe")

	successCondition := strings.Join([]string{
		"text_len == 2",
		"text_copied == 2",
		"text[0] == 72",
		"text[1] == 105",
		"clip_written == 3",
		"clip_read == 3",
		"clip_dst[0] == 88",
		"clip_dst[1] == 89",
		"clip_dst[2] == 90",
		"composition_copied == 4",
		"composition[0] == 1",
		"composition[1] == 0",
		"composition[2] == 1",
		"composition[3] == 0",
		"closed == 0",
	}, " && ")
	source := fmt.Sprintf(`module surface_ipc_text_probe

func main() -> Int
uses surface, mem, alloc:
    let handle: Int = core.surface_open("IPC Text Probe", 320, 200)
    var text: []u8 = core.make_u8(4)
    let text_len: Int = core.surface_poll_event_text_len(handle)
    let text_copied: Int = core.surface_poll_event_text_into(handle, text)
    var clip_src: []u8 = core.make_u8(3)
    clip_src[0] = 65
    clip_src[1] = 66
    clip_src[2] = 67
    let clip_written: Int = core.surface_clipboard_write_text(handle, clip_src)
    var clip_dst: []u8 = core.make_u8(5)
    let clip_read: Int = core.surface_clipboard_read_text_into(handle, clip_dst)
    var composition: []i32 = core.make_i32(4)
    let composition_copied: Int = core.surface_poll_composition_into(handle, composition)
    let closed: Int = core.surface_close(handle)
    if %s:
        return 42
    return 7
`, successCondition)
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("write text probe source: %v", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen fake surface host: %v", err)
	}
	defer listener.Close()

	serverDone := make(chan compiledIPCServerResult, 1)
	go runCompiledIPCTextFakeHost(listener, serverDone)

	_, err = compiler.BuildFileWithStatsOpt(sourcePath, binaryPath, "linux-x64", compiler.BuildOptions{
		SurfaceHostRequired:   true,
		SurfaceHostDriver:     "wayland",
		SurfaceHostProtocol:   ProtocolName,
		SurfaceHostSocketPath: socketPath,
	})
	if err != nil {
		t.Fatalf("build text probe: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binaryPath)
	output, runErr := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("compiled surface text probe timed out; output:\n%s", output)
	}

	var result compiledIPCServerResult
	select {
	case result = <-serverDone:
	case <-time.After(2 * time.Second):
		t.Fatal("fake text host did not finish after compiled app exited")
	}
	if code, ok := exitCode(runErr); !ok || code != 42 {
		t.Fatalf(
			"run compiled surface text probe: %v\nops: %v\nhost error: %v\n%s",
			runErr,
			result.ops,
			result.err,
			output,
		)
	}
	if result.err != nil {
		t.Fatalf("fake text host failed after ops %v: %v", result.ops, result.err)
	}

	wantOps := []Op{
		OpOpen,
		OpPollEventTextInto,
		OpPollEventTextInto,
		OpClipboardWriteText,
		OpClipboardReadText,
		OpPollCompositionInto,
		OpClose,
	}
	if !reflect.DeepEqual(result.ops, wantOps) {
		t.Fatalf("unexpected op stream\nwant: %v\n got: %v", wantOps, result.ops)
	}
}

func TestCompiledLinuxSurfaceHostIPCScalarEventAccessors(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux runtime smoke requires linux")
	}

	dir := t.TempDir()
	socketDir := shortSocketDir(t)
	socketPath := filepath.Join(socketDir, "host.sock")
	sourcePath := filepath.Join(dir, "surface_ipc_scalar_event_probe.tetra")
	binaryPath := filepath.Join(dir, "surface-ipc-scalar-event-probe")

	source := `module surface_ipc_scalar_event_probe

func main() -> Int
uses surface:
    let handle: Int = core.surface_open("IPC Scalar Event Probe", 320, 200)
    let kind: Int = core.surface_poll_event_kind(handle)
    let x: Int = core.surface_poll_event_x(handle)
    let y: Int = core.surface_poll_event_y(handle)
    let button: Int = core.surface_poll_event_button(handle)
    let closed: Int = core.surface_close(handle)
    if kind == 5 && x == 48 && y == 96 && button == 1 && closed == 0:
        return 42
    return 7
`
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("write scalar event probe source: %v", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen fake surface host: %v", err)
	}
	defer listener.Close()

	serverDone := make(chan compiledIPCServerResult, 1)
	go runCompiledIPCScalarEventFakeHost(listener, serverDone)

	_, err = compiler.BuildFileWithStatsOpt(sourcePath, binaryPath, "linux-x64", compiler.BuildOptions{
		SurfaceHostRequired:   true,
		SurfaceHostDriver:     "wayland",
		SurfaceHostProtocol:   ProtocolName,
		SurfaceHostSocketPath: socketPath,
	})
	if err != nil {
		t.Fatalf("build scalar event probe: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binaryPath)
	output, runErr := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("compiled surface scalar event probe timed out; output:\n%s", output)
	}

	var result compiledIPCServerResult
	select {
	case result = <-serverDone:
	case <-time.After(2 * time.Second):
		t.Fatal("fake scalar event host did not finish after compiled app exited")
	}
	if code, ok := exitCode(runErr); !ok || code != 42 {
		t.Fatalf(
			"run compiled surface scalar event probe: %v\nops: %v\nhost error: %v\n%s",
			runErr,
			result.ops,
			result.err,
			output,
		)
	}
	if result.err != nil {
		t.Fatalf("fake scalar event host failed after ops %v: %v", result.ops, result.err)
	}

	wantOps := []Op{
		OpOpen,
		OpPollEventInto,
		OpPollEventInto,
		OpPollEventInto,
		OpPollEventInto,
		OpClose,
	}
	if !reflect.DeepEqual(result.ops, wantOps) {
		t.Fatalf("unexpected op stream\nwant: %v\n got: %v", wantOps, result.ops)
	}
}

func TestCompiledSurfaceWindowCounterReachesPresentAgainstFakeHost(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux runtime smoke requires linux")
	}

	root := repoRoot(t)
	dir := t.TempDir()
	socketDir := shortSocketDir(t)
	socketPath := filepath.Join(socketDir, "host.sock")
	sourcePath := filepath.Join(root, "examples", "surface", "runtime", "surface_window_counter.tetra")
	binaryPath := filepath.Join(dir, "surface-window-counter")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen fake surface host: %v", err)
	}
	defer listener.Close()

	serverDone := make(chan compiledIPCServerResult, 1)
	go runCounterExampleFakeHost(listener, serverDone)

	_, err = compiler.BuildFileWithStatsOpt(sourcePath, binaryPath, "linux-x64", compiler.BuildOptions{
		ProjectRoot:           root,
		SourceRoots:           []string{root},
		SurfaceHostRequired:   true,
		SurfaceHostDriver:     "wayland",
		SurfaceHostProtocol:   ProtocolName,
		SurfaceHostSocketPath: socketPath,
	})
	if err != nil {
		t.Fatalf("build counter example: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binaryPath)
	output, runErr := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("compiled counter example timed out; output:\n%s", output)
	}

	var result compiledIPCServerResult
	select {
	case result = <-serverDone:
	case <-time.After(2 * time.Second):
		t.Fatal("counter fake host did not finish after compiled app exited")
	}
	if code, ok := exitCode(runErr); !ok || code != 1 {
		t.Fatalf(
			"run counter example: %v\nops: %v\nhost error: %v\n%s",
			runErr,
			result.ops,
			result.err,
			output,
		)
	}
	if result.err != nil {
		t.Fatalf("counter fake host failed after ops %v: %v", result.ops, result.err)
	}
}

func exitCode(err error) (int, bool) {
	if err == nil {
		return 0, true
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return 0, false
	}
	return exitErr.ExitCode(), true
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		examplesInfo, examplesErr := os.Stat(filepath.Join(dir, "examples"))
		libInfo, libErr := os.Stat(filepath.Join(dir, "lib"))
		if examplesErr == nil && libErr == nil && examplesInfo.IsDir() && libInfo.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root")
		}
		dir = parent
	}
}

func shortSocketDir(t *testing.T) string {
	t.Helper()
	base := os.Getenv("XDG_RUNTIME_DIR")
	if base == "" {
		base = "/tmp"
	}
	dir, err := os.MkdirTemp(base, "tetra-ipc-")
	if err != nil {
		t.Fatalf("create short socket dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	return dir
}

type compiledIPCServerResult struct {
	ops []Op
	err error
}

func runCompiledIPCFakeHost(listener net.Listener, done chan<- compiledIPCServerResult) {
	var result compiledIPCServerResult
	defer func() {
		done <- result
	}()

	conn, err := listener.Accept()
	if err != nil {
		result.err = err
		return
	}
	defer conn.Close()

	const handle uint32 = 42
	var connectionHandle uint32
	for {
		if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
			result.err = err
			return
		}
		req, err := ReadRequest(conn)
		if err != nil {
			result.err = fmt.Errorf("read request: %w", err)
			return
		}
		result.ops = append(result.ops, req.Op)
		if connectionHandle != 0 && req.Op != OpOpen {
			req.Handle = connectionHandle
		}

		switch req.Op {
		case OpOpen:
			if req.Width != 320 || req.Height != 200 {
				result.err = fmt.Errorf("open size = %dx%d, want 320x200", req.Width, req.Height)
				return
			}
			if err := WriteResponse(
				conn,
				Response{RequestID: req.RequestID, Status: 0, Value0: int32(handle)},
			); err != nil {
				result.err = fmt.Errorf("write open response: %w", err)
				return
			}
			connectionHandle = handle
		case OpPollEventInto:
			if req.Handle != handle {
				result.err = fmt.Errorf("poll handle = %d, want %d", req.Handle, handle)
				return
			}
			event := Event{Kind: 0}
			if err := WriteResponse(
				conn,
				Response{RequestID: req.RequestID, Status: 0, Payload: EncodeEvent(event)},
			); err != nil {
				result.err = fmt.Errorf("write poll response: %w", err)
				return
			}
		case OpPresentRGBA:
			if req.Handle != handle {
				result.err = fmt.Errorf("present handle = %d, want %d", req.Handle, handle)
				return
			}
			if req.Width != 320 || req.Height != 200 || req.Stride != 320*4 {
				result.err = fmt.Errorf(
					"present geometry = %dx%d stride %d, want 320x200 stride 1280",
					req.Width,
					req.Height,
					req.Stride,
				)
				return
			}
			if got, want := len(req.Payload), 320*200*4; got != want {
				result.err = fmt.Errorf("present payload bytes = %d, want %d", got, want)
				return
			}
			if err := WriteResponse(
				conn,
				Response{RequestID: req.RequestID, Status: 0, Value0: 1},
			); err != nil {
				result.err = fmt.Errorf("write present response: %w", err)
				return
			}
		case OpRequestRedraw:
			if req.Handle != handle {
				result.err = fmt.Errorf("request_redraw handle = %d, want %d", req.Handle, handle)
				return
			}
			if err := WriteResponse(conn, Response{RequestID: req.RequestID, Status: 0}); err != nil {
				result.err = fmt.Errorf("write request_redraw response: %w", err)
				return
			}
		case OpClose:
			if req.Handle != handle {
				result.err = fmt.Errorf("close handle = %d, want %d", req.Handle, handle)
				return
			}
			if err := WriteResponse(conn, Response{RequestID: req.RequestID, Status: 0}); err != nil {
				result.err = fmt.Errorf("write close response: %w", err)
				return
			}
			connectionHandle = 0
			return
		default:
			result.err = fmt.Errorf("unexpected op %d", req.Op)
			return
		}
	}
}

func runCompiledIPCDelayedPollFakeHost(listener net.Listener, done chan<- compiledIPCServerResult) {
	var result compiledIPCServerResult
	defer func() {
		done <- result
	}()

	conn, err := listener.Accept()
	if err != nil {
		result.err = err
		return
	}
	defer conn.Close()

	const handle uint32 = 42
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		result.err = err
		return
	}
	openReq, err := ReadRequest(conn)
	if err != nil {
		result.err = fmt.Errorf("read open request: %w", err)
		return
	}
	result.ops = append(result.ops, openReq.Op)
	if openReq.Op != OpOpen {
		result.err = fmt.Errorf("first op = %d, want open", openReq.Op)
		return
	}
	if err := WriteResponse(
		conn,
		Response{RequestID: openReq.RequestID, Status: 0, Value0: int32(handle)},
	); err != nil {
		result.err = fmt.Errorf("write open response: %w", err)
		return
	}

	pollReq, err := ReadRequest(conn)
	if err != nil {
		result.err = fmt.Errorf("read poll request: %w", err)
		return
	}
	result.ops = append(result.ops, pollReq.Op)
	if pollReq.Op != OpPollEventInto {
		result.err = fmt.Errorf("second op = %d, want poll_event_into", pollReq.Op)
		return
	}

	if err := conn.SetReadDeadline(time.Now().Add(150 * time.Millisecond)); err != nil {
		result.err = err
		return
	}
	nextReq, err := ReadRequest(conn)
	if err == nil {
		result.ops = append(result.ops, nextReq.Op)
		result.err = fmt.Errorf("app sent op %d before reading poll_event_into response", nextReq.Op)
		return
	}
	if !isTimeout(err) {
		result.err = fmt.Errorf("read before delayed poll response: %w", err)
		return
	}

	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		result.err = err
		return
	}
	if err := WriteResponse(conn, Response{
		RequestID: pollReq.RequestID,
		Status:    0,
		Payload:   EncodeEvent(Event{Kind: 0}),
	}); err != nil {
		result.err = fmt.Errorf("write delayed poll response: %w", err)
		return
	}

	for _, want := range []Op{OpPresentRGBA, OpRequestRedraw, OpClose} {
		req, err := ReadRequest(conn)
		if err != nil {
			result.err = fmt.Errorf("read %d request: %w", want, err)
			return
		}
		result.ops = append(result.ops, req.Op)
		if req.Op != want {
			result.err = fmt.Errorf("op = %d, want %d", req.Op, want)
			return
		}
		switch req.Op {
		case OpPresentRGBA:
			if got, wantLen := len(req.Payload), 320*200*4; got != wantLen {
				result.err = fmt.Errorf("present payload bytes = %d, want %d", got, wantLen)
				return
			}
			if sent, err := readUnexpectedRequestBeforeResponse(conn); err != nil {
				result.err = fmt.Errorf("check before present response: %w", err)
				return
			} else if sent != 0 {
				result.ops = append(result.ops, sent)
				result.err = fmt.Errorf("app sent op %d before reading present response", sent)
				return
			}
			if err := WriteResponse(
				conn,
				Response{RequestID: req.RequestID, Status: 0, Value0: 1},
			); err != nil {
				result.err = fmt.Errorf("write present response: %w", err)
				return
			}
		case OpRequestRedraw:
			if sent, err := readUnexpectedRequestBeforeResponse(conn); err != nil {
				result.err = fmt.Errorf("check before request_redraw response: %w", err)
				return
			} else if sent != 0 {
				result.ops = append(result.ops, sent)
				result.err = fmt.Errorf("app sent op %d before reading request_redraw response", sent)
				return
			}
			if err := WriteResponse(conn, Response{RequestID: req.RequestID, Status: 0}); err != nil {
				result.err = fmt.Errorf("write %d response: %w", req.Op, err)
				return
			}
		default:
			if err := WriteResponse(conn, Response{RequestID: req.RequestID, Status: 0}); err != nil {
				result.err = fmt.Errorf("write %d response: %w", req.Op, err)
				return
			}
		}
	}
}

func runCompiledIPCScalarEventFakeHost(listener net.Listener, done chan<- compiledIPCServerResult) {
	var result compiledIPCServerResult
	defer func() {
		done <- result
	}()

	conn, err := listener.Accept()
	if err != nil {
		result.err = err
		return
	}
	defer conn.Close()

	const handle uint32 = 42
	var connectionHandle uint32
	eventPolls := 0
	for {
		if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
			result.err = err
			return
		}
		req, err := ReadRequest(conn)
		if err != nil {
			result.err = fmt.Errorf("read request: %w", err)
			return
		}
		result.ops = append(result.ops, req.Op)
		if connectionHandle != 0 && req.Op != OpOpen {
			req.Handle = connectionHandle
		}

		switch req.Op {
		case OpOpen:
			if err := WriteResponse(
				conn,
				Response{RequestID: req.RequestID, Status: 0, Value0: int32(handle)},
			); err != nil {
				result.err = fmt.Errorf("write open response: %w", err)
				return
			}
			connectionHandle = handle
		case OpPollEventInto:
			if req.Handle != handle {
				result.err = fmt.Errorf("poll handle = %d, want %d", req.Handle, handle)
				return
			}
			eventPolls++
			event := Event{Kind: 5, X: 48, Y: 96, Button: 1, Width: 320, Height: 200}
			if err := WriteResponse(
				conn,
				Response{RequestID: req.RequestID, Status: 0, Payload: EncodeEvent(event)},
			); err != nil {
				result.err = fmt.Errorf("write poll response: %w", err)
				return
			}
			if eventPolls > 4 {
				result.err = fmt.Errorf("too many scalar event polls: %d", eventPolls)
				return
			}
		case OpClose:
			if req.Handle != handle {
				result.err = fmt.Errorf("close handle = %d, want %d", req.Handle, handle)
				return
			}
			if eventPolls != 4 {
				result.err = fmt.Errorf("scalar event poll count = %d, want 4", eventPolls)
				return
			}
			if err := WriteResponse(conn, Response{RequestID: req.RequestID, Status: 0}); err != nil {
				result.err = fmt.Errorf("write close response: %w", err)
				return
			}
			connectionHandle = 0
			return
		default:
			result.err = fmt.Errorf("unexpected op %d", req.Op)
			return
		}
	}
}

func runCompiledIPCTextFakeHost(listener net.Listener, done chan<- compiledIPCServerResult) {
	var result compiledIPCServerResult
	defer func() {
		done <- result
	}()

	conn, err := listener.Accept()
	if err != nil {
		result.err = err
		return
	}
	defer conn.Close()

	const handle uint32 = 42
	var connectionHandle uint32
	textPolls := 0
	for {
		if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
			result.err = err
			return
		}
		req, err := ReadRequest(conn)
		if err != nil {
			result.err = fmt.Errorf("read request: %w", err)
			return
		}
		result.ops = append(result.ops, req.Op)
		if connectionHandle != 0 && req.Op != OpOpen {
			req.Handle = connectionHandle
		}

		switch req.Op {
		case OpOpen:
			if err := WriteResponse(
				conn,
				Response{RequestID: req.RequestID, Status: 0, Value0: int32(handle)},
			); err != nil {
				result.err = fmt.Errorf("write open response: %w", err)
				return
			}
			connectionHandle = handle
		case OpPollEventTextInto:
			if req.Handle != handle {
				result.err = fmt.Errorf("poll text handle = %d, want %d", req.Handle, handle)
				return
			}
			textPolls++
			resp := Response{RequestID: req.RequestID, Status: 0, Value0: 2}
			switch textPolls {
			case 1:
				if req.Width != 0 {
					result.err = fmt.Errorf("text len request width = %d, want 0", req.Width)
					return
				}
			case 2:
				if req.Width != 4 {
					result.err = fmt.Errorf("text copy request width = %d, want 4", req.Width)
					return
				}
				resp.Payload = []byte("Hi")
			default:
				result.err = fmt.Errorf("unexpected text poll %d", textPolls)
				return
			}
			if err := WriteResponse(conn, resp); err != nil {
				result.err = fmt.Errorf("write text response: %w", err)
				return
			}
		case OpClipboardWriteText:
			if req.Handle != handle {
				result.err = fmt.Errorf("clipboard write handle = %d, want %d", req.Handle, handle)
				return
			}
			if string(req.Payload) != "ABC" {
				result.err = fmt.Errorf("clipboard write payload = %q, want ABC", req.Payload)
				return
			}
			if err := WriteResponse(
				conn,
				Response{RequestID: req.RequestID, Status: 0, Value0: int32(len(req.Payload))},
			); err != nil {
				result.err = fmt.Errorf("write clipboard write response: %w", err)
				return
			}
		case OpClipboardReadText:
			if req.Handle != handle {
				result.err = fmt.Errorf("clipboard read handle = %d, want %d", req.Handle, handle)
				return
			}
			if req.Width != 5 {
				result.err = fmt.Errorf("clipboard read request width = %d, want 5", req.Width)
				return
			}
			if err := WriteResponse(
				conn,
				Response{RequestID: req.RequestID, Status: 0, Value0: 3, Payload: []byte("XYZ")},
			); err != nil {
				result.err = fmt.Errorf("write clipboard read response: %w", err)
				return
			}
		case OpPollCompositionInto:
			if req.Handle != handle {
				result.err = fmt.Errorf("composition handle = %d, want %d", req.Handle, handle)
				return
			}
			if err := WriteResponse(
				conn,
				Response{
					RequestID: req.RequestID,
					Status:    0,
					Payload:   EncodeComposition([4]int32{1, 0, 1, 0}),
				},
			); err != nil {
				result.err = fmt.Errorf("write composition response: %w", err)
				return
			}
		case OpClose:
			if req.Handle != handle {
				result.err = fmt.Errorf("close handle = %d, want %d", req.Handle, handle)
				return
			}
			if err := WriteResponse(conn, Response{RequestID: req.RequestID, Status: 0}); err != nil {
				result.err = fmt.Errorf("write close response: %w", err)
				return
			}
			connectionHandle = 0
			return
		default:
			result.err = fmt.Errorf("unexpected op %d", req.Op)
			return
		}
	}
}

func runCounterExampleFakeHost(listener net.Listener, done chan<- compiledIPCServerResult) {
	var result compiledIPCServerResult
	defer func() {
		done <- result
	}()

	conn, err := listener.Accept()
	if err != nil {
		result.err = err
		return
	}
	defer conn.Close()

	const handle uint32 = 42
	var connectionHandle uint32
	var presentCount int
	var pollCount int
	for {
		if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
			result.err = err
			return
		}
		req, err := ReadRequest(conn)
		if err != nil {
			result.err = fmt.Errorf("read request: %w", err)
			return
		}
		result.ops = append(result.ops, req.Op)
		if connectionHandle != 0 && req.Op != OpOpen {
			req.Handle = connectionHandle
		}

		switch req.Op {
		case OpOpen:
			if err := WriteResponse(
				conn,
				Response{RequestID: req.RequestID, Status: 0, Value0: int32(handle)},
			); err != nil {
				result.err = fmt.Errorf("write open response: %w", err)
				return
			}
			connectionHandle = handle
		case OpPollEventInto:
			pollCount++
			if presentCount == 0 && pollCount > 3 {
				result.err = fmt.Errorf("counter polled %d times before first present", pollCount)
				return
			}
			kind := int32(0)
			if presentCount > 0 {
				kind = 1
			}
			if err := WriteResponse(conn, Response{
				RequestID: req.RequestID,
				Status:    0,
				Payload:   EncodeEvent(Event{Kind: kind, Width: 320, Height: 200}),
			}); err != nil {
				result.err = fmt.Errorf("write poll response: %w", err)
				return
			}
		case OpPresentRGBA:
			presentCount++
			if req.Width != 320 || req.Height != 200 || req.Stride != 320*4 {
				result.err = fmt.Errorf(
					"present geometry = %dx%d stride %d, want 320x200 stride 1280",
					req.Width,
					req.Height,
					req.Stride,
				)
				return
			}
			if got, wantLen := len(req.Payload), 320*200*4; got != wantLen {
				result.err = fmt.Errorf("present payload bytes = %d, want %d", got, wantLen)
				return
			}
			if err := WriteResponse(
				conn,
				Response{RequestID: req.RequestID, Status: 0, Value0: 1},
			); err != nil {
				result.err = fmt.Errorf("write present response: %w", err)
				return
			}
		case OpRequestRedraw:
			if err := WriteResponse(conn, Response{RequestID: req.RequestID, Status: 0}); err != nil {
				result.err = fmt.Errorf("write request_redraw response: %w", err)
				return
			}
		case OpClose:
			if presentCount == 0 {
				result.err = fmt.Errorf("counter closed without presenting a frame")
				return
			}
			if err := WriteResponse(conn, Response{RequestID: req.RequestID, Status: 0}); err != nil {
				result.err = fmt.Errorf("write close response: %w", err)
				return
			}
			connectionHandle = 0
			return
		default:
			result.err = fmt.Errorf("unexpected op %d", req.Op)
			return
		}
	}
}

func readUnexpectedRequestBeforeResponse(conn net.Conn) (Op, error) {
	if err := conn.SetReadDeadline(time.Now().Add(150 * time.Millisecond)); err != nil {
		return 0, err
	}
	nextReq, err := ReadRequest(conn)
	if err == nil {
		return nextReq.Op, nil
	}
	if !isTimeout(err) {
		return 0, err
	}
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		return 0, err
	}
	return 0, nil
}
