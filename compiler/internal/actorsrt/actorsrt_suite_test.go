package actorsrt

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"tetra_language/compiler/internal/backend/x64"
	"tetra_language/compiler/internal/buildruntime"
	"tetra_language/compiler/internal/format/tobj"
)

// ---- actor_runtime_byte_counters_test.go ----

func TestActorRuntimeByteCounterLayoutIsStable(t *testing.T) {
	if actorSize < actorBackpressureEventsOff+8 {
		t.Fatalf(
			"actorSize=%d does not cover backpressure events offset %d",
			actorSize,
			actorBackpressureEventsOff,
		)
	}
	if actorMailboxBytesOff%8 != 0 || actorMailboxPeakBytesOff%8 != 0 ||
		actorReclaimedBytesOff%8 != 0 ||
		actorBytesCopiedOff%8 != 0 ||
		actorCopyCountOff%8 != 0 ||
		actorByteBudgetOff%8 != 0 ||
		actorOverBudgetCountOff%8 != 0 ||
		actorBackpressureEventsOff%8 != 0 {
		t.Fatalf("actor byte counter offsets must remain u64-aligned")
	}
	if maxActorMailboxBytes != maxActorMailboxMsgs*msgSize {
		t.Fatalf(
			"maxActorMailboxBytes=%d, want maxActorMailboxMsgs*msgSize=%d",
			maxActorMailboxBytes,
			maxActorMailboxMsgs*msgSize,
		)
	}
	if schedMsgPoolAllocFailuresOff != schedMsgPoolReclaimedBytesOff+8 {
		t.Fatalf("scheduler message-pool counters must remain densely packed")
	}
}

func TestLinuxRuntimeInitializesActorByteCounters(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}

	entry, ok := symbolBody(obj, "__tetra_entry")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_entry")
	}
	spawn, ok := symbolBody(obj, "__tetra_actor_spawn")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_spawn")
	}

	for _, body := range [][]byte{entry, spawn} {
		for _, off := range []int32{
			actorMailboxBytesOff,
			actorMailboxPeakBytesOff,
			actorReclaimedBytesOff,
			actorBytesCopiedOff,
			actorCopyCountOff,
			actorOverBudgetCountOff,
			actorBackpressureEventsOff,
		} {
			if !bytes.Contains(body, movMem64RdiDispRaxEncoding(off)) {
				t.Fatalf("runtime initializer missing zeroed actor counter offset %d", off)
			}
		}
		if !bytes.Contains(body, movMem64RdiDispRaxEncoding(actorByteBudgetOff)) {
			t.Fatalf("runtime initializer missing actor byte budget offset %d", actorByteBudgetOff)
		}
	}
	for _, off := range []int32{
		schedMsgPoolCapacityBytesOff,
		schedMsgPoolLiveBytesOff,
		schedMsgPoolReclaimedBytesOff,
		schedMsgPoolAllocFailuresOff,
	} {
		if !bytes.Contains(entry, movMem64RdiDispRaxEncoding(off)) {
			t.Fatalf("__tetra_entry missing scheduler message-pool counter offset %d", off)
		}
	}
}

func TestLinuxRuntimeObjectExportsActorMemoryTelemetrySnapshot(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	buildruntime.AnnotateRuntimeObjectSignatures(obj)
	if err := buildruntime.ValidateActorTelemetryRuntimeObject(obj); err != nil {
		t.Fatalf("ValidateActorTelemetryRuntimeObject: %v", err)
	}
	for _, sym := range obj.Symbols {
		if sym.Name != "__tetra_actor_memory_snapshot" {
			continue
		}
		if !sym.HasSignature || sym.ParamSlots != 1 || sym.ReturnSlots != 1 {
			t.Fatalf(
				"actor memory snapshot signature = has:%v params:%d returns:%d, want 1 -> 1",
				sym.HasSignature,
				sym.ParamSlots,
				sym.ReturnSlots,
			)
		}
		return
	}
	t.Fatalf("linux runtime missing __tetra_actor_memory_snapshot")
}

func TestLinuxRuntimeActorMemorySnapshotExposesBudgetBackpressureFields(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}
	body, ok := symbolBody(obj, "__tetra_actor_memory_snapshot")
	if !ok {
		t.Fatalf("linux runtime missing __tetra_actor_memory_snapshot")
	}
	for _, off := range []int32{
		actorMailboxBytesOff,
		actorMailboxPeakBytesOff,
		actorBytesCopiedOff,
		actorByteBudgetOff,
		actorOverBudgetCountOff,
		actorBackpressureEventsOff,
	} {
		if !bytes.Contains(body, movRaxFromR8DispEncoding(off)) {
			t.Fatalf("actor memory snapshot missing load for actor offset %d", off)
		}
	}
}

func TestLinuxSurfaceHostIPCRuntimeDoesNotUseMemfdSurfaceOpen(t *testing.T) {
	obj, err := BuildLinuxX64WithSurfaceHostIPC([]string{"main"}, SurfaceHostIPCOptions{
		SocketPath: "/tmp/tetra-surface-host.sock",
	})
	if err != nil {
		t.Fatalf("BuildLinuxX64WithSurfaceHostIPC: %v", err)
	}
	body, ok := symbolBody(obj, "__tetra_surface_open")
	if !ok {
		t.Fatalf("surface host IPC runtime missing __tetra_surface_open")
	}
	if bytes.Contains(body, movEaxImm32Encoding(linuxSysMemfdCreate)) {
		t.Fatalf("__tetra_surface_open still uses memfd_create in host-required runtime")
	}
	if !bytes.Contains(body, movEaxImm32Encoding(linuxSysSocket)) {
		t.Fatalf("__tetra_surface_open does not attempt AF_UNIX socket creation")
	}
	if !bytes.Contains(body, movEaxImm32Encoding(linuxSysConnect)) {
		t.Fatalf("__tetra_surface_open does not connect to the Surface host socket")
	}
	if !bytes.Contains(body, []byte("/tmp/tetra-surface-host.sock")) {
		t.Fatalf("__tetra_surface_open does not embed the Surface host socket path")
	}
}

func TestLinuxSurfaceHostIPCRuntimeUsesHostProtocolForPresentAndPoll(t *testing.T) {
	obj, err := BuildLinuxX64WithSurfaceHostIPC([]string{"main"}, SurfaceHostIPCOptions{
		SocketPath: "/tmp/tetra-surface-host.sock",
	})
	if err != nil {
		t.Fatalf("BuildLinuxX64WithSurfaceHostIPC: %v", err)
	}
	for _, name := range []string{"__tetra_surface_present_rgba", "__tetra_surface_poll_event_into"} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("surface host IPC runtime missing %s", name)
		}
		if bytes.Contains(body, movEaxImm32Encoding(linuxSysLseek)) {
			t.Fatalf("%s still uses lseek/memfd cursor behavior in host-required runtime", name)
		}
		if !bytes.Contains(body, movEaxImm32Encoding(linuxSysWrite)) {
			t.Fatalf("%s does not write a Surface host IPC request", name)
		}
		if !bytes.Contains(body, movEaxImm32Encoding(linuxSysRead)) {
			t.Fatalf("%s does not read a Surface host IPC response", name)
		}
		if !bytes.Contains(body, uint32LE(surfaceHostMagic)) {
			t.Fatalf("%s does not embed the Surface host protocol magic", name)
		}
	}
}

func TestLinuxSurfaceHostIPCRuntimeUsesHostProtocolForScalarEventAccessors(t *testing.T) {
	obj, err := BuildLinuxX64WithSurfaceHostIPC([]string{"main"}, SurfaceHostIPCOptions{
		SocketPath: "/tmp/tetra-surface-host.sock",
	})
	if err != nil {
		t.Fatalf("BuildLinuxX64WithSurfaceHostIPC: %v", err)
	}
	for _, name := range []string{
		"__tetra_surface_poll_event_kind",
		"__tetra_surface_poll_event_x",
		"__tetra_surface_poll_event_y",
		"__tetra_surface_poll_event_button",
	} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("surface host IPC runtime missing %s", name)
		}
		if !bytes.Contains(body, movEaxImm32Encoding(linuxSysWrite)) {
			t.Fatalf("%s does not write a Surface host IPC request", name)
		}
		if !bytes.Contains(body, movEaxImm32Encoding(linuxSysRead)) {
			t.Fatalf("%s does not read a Surface host IPC response", name)
		}
		if !bytes.Contains(body, uint32LE(surfaceHostMagic)) {
			t.Fatalf("%s does not embed the Surface host protocol magic", name)
		}
	}
}

func TestLinuxSurfaceHostIPCRuntimeUsesHostProtocolForTextClipboardAndComposition(t *testing.T) {
	obj, err := BuildLinuxX64WithSurfaceHostIPC([]string{"main"}, SurfaceHostIPCOptions{
		SocketPath: "/tmp/tetra-surface-host.sock",
	})
	if err != nil {
		t.Fatalf("BuildLinuxX64WithSurfaceHostIPC: %v", err)
	}
	for _, name := range []string{
		"__tetra_surface_poll_event_text_len",
		"__tetra_surface_poll_event_text_into",
		"__tetra_surface_clipboard_write_text",
		"__tetra_surface_clipboard_read_text_into",
		"__tetra_surface_poll_composition_into",
	} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("surface host IPC runtime missing %s", name)
		}
		if !bytes.Contains(body, movEaxImm32Encoding(linuxSysWrite)) {
			t.Fatalf("%s does not write a Surface host IPC request", name)
		}
		if !bytes.Contains(body, movEaxImm32Encoding(linuxSysRead)) {
			t.Fatalf("%s does not read a Surface host IPC response", name)
		}
		if !bytes.Contains(body, uint32LE(surfaceHostMagic)) {
			t.Fatalf("%s does not embed the Surface host protocol magic", name)
		}
	}
}

func TestLinuxSurfaceHostIPCBeginFrameStaysLocal(t *testing.T) {
	obj, err := BuildLinuxX64WithSurfaceHostIPC([]string{"main"}, SurfaceHostIPCOptions{
		SocketPath: "/tmp/tetra-surface-host.sock",
	})
	if err != nil {
		t.Fatalf("BuildLinuxX64WithSurfaceHostIPC: %v", err)
	}
	body, ok := symbolBody(obj, "__tetra_surface_begin_frame")
	if !ok {
		t.Fatalf("surface host IPC runtime missing __tetra_surface_begin_frame")
	}
	if bytes.Contains(body, uint32LE(surfaceHostMagic)) {
		t.Fatalf(
			"__tetra_surface_begin_frame should stay local; " +
				"host evidence is produced by open/poll/present/close",
		)
	}
	if !bytes.Contains(body, []byte{0x31, 0xC0, 0xC3}) {
		t.Fatalf("__tetra_surface_begin_frame should return local no-op success")
	}
}

func TestLinuxSurfaceHostIPCPresentReturnsOneOnSuccess(t *testing.T) {
	obj, err := BuildLinuxX64WithSurfaceHostIPC([]string{"main"}, SurfaceHostIPCOptions{
		SocketPath: "/tmp/tetra-surface-host.sock",
	})
	if err != nil {
		t.Fatalf("BuildLinuxX64WithSurfaceHostIPC: %v", err)
	}
	body, ok := symbolBody(obj, "__tetra_surface_present_rgba")
	if !ok {
		t.Fatalf("surface host IPC runtime missing __tetra_surface_present_rgba")
	}
	successEpilogue := append(movEaxImm32Encoding(1), 0xC9, 0xC3)
	failureEpilogue := []byte{0x31, 0xC0, 0xC9, 0xC3}
	successAt := bytes.Index(body, successEpilogue)
	failureAt := bytes.LastIndex(body, failureEpilogue)
	if successAt < 0 {
		t.Fatalf("__tetra_surface_present_rgba missing success return 1 epilogue")
	}
	if failureAt < 0 {
		t.Fatalf("__tetra_surface_present_rgba missing failure return 0 epilogue")
	}
	if successAt > failureAt {
		t.Fatalf(
			"__tetra_surface_present_rgba returns 0 on success and 1 on failure; "+
				"success epilogue at %d, failure epilogue at %d",
			successAt,
			failureAt,
		)
	}
}

func TestLinuxSurfaceHostIPCPresentUsesStreamWriteLoop(t *testing.T) {
	obj, err := BuildLinuxX64WithSurfaceHostIPC([]string{"main"}, SurfaceHostIPCOptions{
		SocketPath: "/tmp/tetra-surface-host.sock",
	})
	if err != nil {
		t.Fatalf("BuildLinuxX64WithSurfaceHostIPC: %v", err)
	}
	body, ok := symbolBody(obj, "__tetra_surface_present_rgba")
	if !ok {
		t.Fatalf("surface host IPC runtime missing __tetra_surface_present_rgba")
	}
	if bytes.Contains(body, []byte{0x48, 0x39, 0xC2}) {
		t.Fatalf(
			"__tetra_surface_present_rgba still uses single-write cmp rdx,rax instead of stream write loop",
		)
	}
	if !bytes.Contains(body, []byte{0x48, 0x01, 0xC6}) ||
		!bytes.Contains(body, []byte{0x48, 0x29, 0xC2}) {
		t.Fatalf("__tetra_surface_present_rgba missing stream pointer/remaining update loop")
	}
}

func TestLinuxRuntimeAccountsMailboxBytesOnSendAndReceive(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}

	for _, name := range []string{
		"__tetra_actor_send",
		"__tetra_actor_send_msg",
		"__tetra_actor_send_begin",
		"__tetra_actor_net_pump",
	} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("linux runtime missing %s", name)
		}
		for _, off := range []int32{
			actorMailboxBytesOff,
			actorMailboxPeakBytesOff,
			actorBytesCopiedOff,
			actorCopyCountOff,
			schedMsgPoolLiveBytesOff,
		} {
			if !bytes.Contains(body, movMem64RdiDispRaxEncoding(off)) {
				t.Fatalf("%s missing enqueue accounting for offset %d", name, off)
			}
		}
	}

	for _, name := range []string{
		"__tetra_actor_recv",
		"__tetra_actor_recv_msg",
		"__tetra_actor_recv_begin",
	} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("linux runtime missing %s", name)
		}
		for _, off := range []int32{
			actorMailboxBytesOff,
			actorReclaimedBytesOff,
			schedMsgPoolLiveBytesOff,
			schedMsgPoolReclaimedBytesOff,
		} {
			if !bytes.Contains(body, movMem64RdiDispRaxEncoding(off)) {
				t.Fatalf("%s missing receive/reclaim accounting for offset %d", name, off)
			}
		}
	}
}

func TestLinuxRuntimeAccountsMessagePoolFailuresWithoutMailboxCounters(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}

	for _, name := range []string{
		"__tetra_actor_send",
		"__tetra_actor_send_msg",
		"__tetra_actor_send_begin",
		"__tetra_actor_net_pump",
	} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("linux runtime missing %s", name)
		}
		if !bytes.Contains(body, movMem64RdiDispRaxEncoding(schedMsgPoolAllocFailuresOff)) {
			t.Fatalf("%s missing message-pool allocation failure counter", name)
		}
	}
}

func TestLinuxRuntimeBackpressurePathsAccountBudgetCounters(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build linux runtime: %v", err)
	}

	for _, name := range []string{
		"__tetra_actor_send",
		"__tetra_actor_send_msg",
		"__tetra_actor_send_begin",
	} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("linux runtime missing %s", name)
		}
		for _, off := range []int32{actorOverBudgetCountOff, actorBackpressureEventsOff} {
			if !bytes.Contains(body, movMem64RdiDispRaxEncoding(off)) {
				t.Fatalf("%s missing backpressure budget counter write for offset %d", name, off)
			}
		}
	}
}

func TestLinuxRuntimeSendFailurePathsDoNotEmitMailboxAccounting(t *testing.T) {
	raw, err := os.ReadFile(
		filepath.Join(
			repoRootFromActorsRTTest(t),
			"compiler",
			"internal",
			"actorsrt",
			"actorsrt_core.go",
		),
	)
	if err != nil {
		t.Fatalf("read actorsrt_core.go: %v", err)
	}
	source := string(raw)
	for _, name := range []string{"emitSend", "emitSendMsg", "emitSendBegin"} {
		body := functionSource(t, source, name)
		accountAt := strings.Index(body, "emitAccountMailboxEnqueueInRdi(e)")
		if accountAt < 0 {
			t.Fatalf("%s missing successful enqueue accounting call", name)
		}
		for _, marker := range []string{
			"overflowTo := len(e.Buf)",
			"fullTo := len(e.Buf)",
			"invalidTo := len(e.Buf)",
			"doneTo := len(e.Buf)",
		} {
			markerAt := strings.Index(body, marker)
			if markerAt < 0 {
				t.Fatalf("%s missing failure marker %q", name, marker)
			}
			if markerAt < accountAt {
				t.Fatalf(
					"%s failure marker %q appears before successful enqueue accounting",
					name,
					marker,
				)
			}
			if strings.Contains(body[markerAt:], "emitAccountMailboxEnqueueInRdi(e)") {
				t.Fatalf(
					"%s failure marker %q must not flow through mailbox accounting",
					name,
					marker,
				)
			}
		}
	}
}

func functionSource(t *testing.T, source string, name string) string {
	t.Helper()
	start := strings.Index(source, "func "+name+"(")
	if start < 0 {
		t.Fatalf("missing function %s", name)
	}
	open := strings.Index(source[start:], "{")
	if open < 0 {
		t.Fatalf("missing function body for %s", name)
	}
	open += start
	depth := 0
	for i := open; i < len(source); i++ {
		switch source[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return source[open+1 : i]
			}
		}
	}
	t.Fatalf("unterminated function body for %s", name)
	return ""
}

func movMem64RdiDispRaxEncoding(off int32) []byte {
	var out [7]byte
	out[0] = 0x48
	out[1] = 0x89
	out[2] = 0x87
	binary.LittleEndian.PutUint32(out[3:], uint32(off))
	return out[:]
}

func movRaxFromR8DispEncoding(off int32) []byte {
	var out [7]byte
	out[0] = 0x49
	out[1] = 0x8B
	out[2] = 0x80
	binary.LittleEndian.PutUint32(out[3:], uint32(off))
	return out[:]
}

func movEaxImm32Encoding(value uint32) []byte {
	var out [5]byte
	out[0] = 0xB8
	binary.LittleEndian.PutUint32(out[1:], value)
	return out[:]
}

func uint32LE(value uint32) []byte {
	var out [4]byte
	binary.LittleEndian.PutUint32(out[:], value)
	return out[:]
}

// ---- actor_state_symbols_test.go ----

func TestBuiltinRuntimeExportsActorStateSymbols(t *testing.T) {
	entries := []string{"main"}
	builders := []struct {
		name  string
		build func([]string) (*tobj.Object, error)
	}{
		{name: "linux-x64", build: BuildLinuxX64},
		{name: "macos-x64", build: BuildMacOSX64},
		{name: "windows-x64", build: BuildWindowsX64},
	}

	for _, tt := range builders {
		t.Run(tt.name, func(t *testing.T) {
			obj, err := tt.build(entries)
			if err != nil {
				t.Fatalf("build runtime: %v", err)
			}
			if !hasSymbol(obj.Symbols, "__tetra_actor_state_load") {
				t.Fatalf("runtime missing __tetra_actor_state_load")
			}
			if !hasSymbol(obj.Symbols, "__tetra_actor_state_store") {
				t.Fatalf("runtime missing __tetra_actor_state_store")
			}
		})
	}
}

func TestLinuxRuntimeExportsFilesystemSymbol(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main"})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	if !hasSymbol(obj.Symbols, "__tetra_fs_exists") {
		t.Fatalf("linux runtime missing __tetra_fs_exists")
	}
}

func TestLinuxRuntimeExportsNetSymbols(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main"})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	for _, name := range []string{
		"__tetra_net_socket_tcp4",
		"__tetra_net_bind_tcp4_loopback",
		"__tetra_net_connect_tcp4_loopback",
		"__tetra_net_listen",
		"__tetra_net_accept4",
		"__tetra_net_read",
		"__tetra_net_recv",
		"__tetra_net_write",
		"__tetra_net_send",
		"__tetra_net_epoll_create",
		"__tetra_net_epoll_ctl_add_read",
		"__tetra_net_epoll_ctl_add_read_write",
		"__tetra_net_epoll_ctl_mod_read",
		"__tetra_net_epoll_ctl_mod_read_write",
		"__tetra_net_epoll_ctl_delete",
		"__tetra_net_epoll_wait_one",
		"__tetra_net_epoll_wait_one_into",
		"__tetra_net_set_nonblocking",
		"__tetra_net_set_reuseport",
		"__tetra_net_set_tcp_nodelay",
		"__tetra_net_close",
	} {
		if !hasSymbol(obj.Symbols, name) {
			t.Fatalf("linux runtime missing %s", name)
		}
	}
}

func TestLinuxRuntimeExportsSurfaceSymbols(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main"})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	for _, name := range []string{
		"__tetra_surface_open",
		"__tetra_surface_close",
		"__tetra_surface_poll_event_kind",
		"__tetra_surface_poll_event_x",
		"__tetra_surface_poll_event_y",
		"__tetra_surface_poll_event_button",
		"__tetra_surface_poll_event_into",
		"__tetra_surface_poll_event_text_len",
		"__tetra_surface_poll_event_text_into",
		"__tetra_surface_clipboard_write_text",
		"__tetra_surface_clipboard_read_text_into",
		"__tetra_surface_poll_composition_into",
		"__tetra_surface_begin_frame",
		"__tetra_surface_present_rgba",
		"__tetra_surface_now_ms",
		"__tetra_surface_request_redraw",
	} {
		if !hasSymbol(obj.Symbols, name) {
			t.Fatalf("linux runtime missing %s", name)
		}
	}
}

func TestLinuxRuntimeExportsDistributedActorSymbols(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main", "worker"})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	for _, name := range []string{
		"__tetra_actor_node_connect",
		"__tetra_actor_spawn_remote",
		"__tetra_actor_node_status",
	} {
		if !hasSymbol(obj.Symbols, name) {
			t.Fatalf("linux runtime missing %s", name)
		}
	}
}

func TestActorNetPumpIsExportedButOnlyLinuxHasRuntimePump(t *testing.T) {
	entries := []string{"main"}
	builders := []struct {
		name       string
		build      func([]string) (*tobj.Object, error)
		wantNoop   bool
		wantActive bool
	}{
		{name: "linux-x64", build: BuildLinuxX64, wantActive: true},
		{name: "macos-x64", build: BuildMacOSX64, wantNoop: true},
		{name: "windows-x64", build: BuildWindowsX64, wantNoop: true},
	}

	for _, tt := range builders {
		t.Run(tt.name, func(t *testing.T) {
			obj, err := tt.build(entries)
			if err != nil {
				t.Fatalf("build runtime: %v", err)
			}
			body, ok := symbolBody(obj, "__tetra_actor_net_pump")
			if !ok {
				t.Fatalf("runtime missing __tetra_actor_net_pump")
			}
			isNoop := len(body) >= 3 && body[0] == 0x31 && body[1] == 0xC0 && body[2] == 0xC3
			if tt.wantNoop && !isNoop {
				t.Fatalf(
					"%s __tetra_actor_net_pump must be a no-op on non-Linux targets, body prefix=% x",
					tt.name,
					bodyPrefix(body, 8),
				)
			}
			if tt.wantActive && isNoop {
				t.Fatalf("%s __tetra_actor_net_pump must be active, got no-op body", tt.name)
			}
		})
	}
}

func TestLinuxDistributedRuntimeUsesWideStackSubFor128ByteFrames(t *testing.T) {
	obj, err := BuildLinuxX64([]string{"main"})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}

	badSignedImm8Sub := []byte{0x48, 0x83, 0xEC, 0x80}
	goodImm32Sub := []byte{0x48, 0x81, 0xEC, 0x80, 0x00, 0x00, 0x00}
	for _, name := range []string{"__tetra_actor_node_connect", "__tetra_actor_net_pump"} {
		body, ok := symbolBody(obj, name)
		if !ok {
			t.Fatalf("runtime missing %s", name)
		}
		if bytes.Contains(body, badSignedImm8Sub) {
			t.Fatalf("%s uses signed imm8 stack subtraction for 128-byte frame", name)
		}
		if !bytes.Contains(body, goodImm32Sub) {
			t.Fatalf(
				"%s missing imm32 stack subtraction for 128-byte frame, prefix=% x",
				name,
				bodyPrefix(body, 16),
			)
		}
	}
}

func TestNonLinuxRuntimesDoNotExportDistributedActorSymbols(t *testing.T) {
	builders := []struct {
		name  string
		build func([]string) (*tobj.Object, error)
	}{
		{name: "macos-x64", build: BuildMacOSX64},
		{name: "windows-x64", build: BuildWindowsX64},
	}
	for _, tt := range builders {
		t.Run(tt.name, func(t *testing.T) {
			obj, err := tt.build([]string{"main", "worker"})
			if err != nil {
				t.Fatalf("build runtime: %v", err)
			}
			for _, name := range []string{
				"__tetra_actor_node_connect",
				"__tetra_actor_spawn_remote",
				"__tetra_actor_node_status",
			} {
				if hasSymbol(obj.Symbols, name) {
					t.Fatalf(
						"%s runtime must not export Linux distributed actor symbol %s",
						tt.name,
						name,
					)
				}
			}
		})
	}
}

func TestRuntimeBuildersRejectInvalidEntrySymbols(t *testing.T) {
	builders := []struct {
		name  string
		build func([]string) (*tobj.Object, error)
	}{
		{name: "linux-x64", build: BuildLinuxX64},
		{name: "macos-x64", build: BuildMacOSX64},
		{name: "windows-x64", build: BuildWindowsX64},
	}
	cases := []struct {
		name    string
		entries []string
		want    string
	}{
		{
			name:    "missing_main",
			entries: nil,
			want:    "missing entry symbols",
		},
		{
			name:    "empty_main",
			entries: []string{""},
			want:    "missing entry symbols",
		},
		{
			name:    "empty_spawn_entry",
			entries: []string{"main", ""},
			want:    "empty runtime entry symbol at index 1",
		},
		{
			name:    "duplicate_entry",
			entries: []string{"main", "worker", "worker"},
			want:    "duplicate runtime entry symbol 'worker'",
		},
	}

	for _, builder := range builders {
		for _, tc := range cases {
			t.Run(builder.name+"/"+tc.name, func(t *testing.T) {
				_, err := builder.build(tc.entries)
				if err == nil {
					t.Fatalf("expected invalid entry symbol error")
				}
				if !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("error = %v, want substring %q", err, tc.want)
				}
			})
		}
	}
}

func hasSymbol(symbols []tobj.Symbol, want string) bool {
	for _, sym := range symbols {
		if sym.Name == want {
			return true
		}
	}
	return false
}

func symbolBody(obj *tobj.Object, want string) ([]byte, bool) {
	start := -1
	end := len(obj.Code)
	for _, sym := range obj.Symbols {
		offset := int(sym.Offset)
		if sym.Name == want {
			start = offset
			continue
		}
		if start >= 0 && offset > start && offset < end {
			end = offset
		}
	}
	if start < 0 || start > len(obj.Code) || end < start {
		return nil, false
	}
	return obj.Code[start:end], true
}

func bodyPrefix(body []byte, n int) []byte {
	if len(body) < n {
		return body
	}
	return body[:n]
}

// ---- production_boundary_test.go ----

func TestActorRuntimeProductionBoundaryAuditCoversP18PlanList(t *testing.T) {
	report, err := ActorRuntimeProductionBoundaryAudit()
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateActorRuntimeProductionBoundaryAudit(report); err != nil {
		t.Fatalf("ValidateActorRuntimeProductionBoundaryAudit failed: %v", err)
	}
	if report.SchemaVersion != "tetra.runtime.actor.production_boundary.v1" {
		t.Fatalf("schema = %q", report.SchemaVersion)
	}
	if report.FullProductionClaimed {
		t.Fatalf("P18.0 audit must not claim a full production actor runtime")
	}
	if !hasActorBoundaryText(report.NonClaims, "full production actor runtime is not claimed") {
		t.Fatalf("non-claims = %#v, want full production actor runtime non-claim", report.NonClaims)
	}

	byID := map[ActorRuntimeBoundaryID]ActorRuntimeBoundaryRow{}
	for _, row := range report.Rows {
		byID[row.ID] = row
	}
	expected := []ActorRuntimeBoundaryID{
		ActorRuntimeBoundaryCurrentLimits,
		ActorRuntimeBoundarySchedulerPrototype,
		ActorRuntimeBoundaryProductionAcceptance,
		ActorRuntimeBoundaryFullClaimBlockers,
	}
	for _, id := range expected {
		if _, ok := byID[id]; !ok {
			t.Fatalf("missing P18.0 audit row %q", id)
		}
	}

	limits := byID[ActorRuntimeBoundaryCurrentLimits]
	if limits.Status != ActorRuntimeBoundaryDocumentedLimit {
		t.Fatalf(
			"current limits status = %q, want %q",
			limits.Status,
			ActorRuntimeBoundaryDocumentedLimit,
		)
	}
	for _, want := range []string{
		"maxActors=128",
		"msgPoolSize=65536",
		"maxActorMailboxMsgs=256",
		"actor_state_slots=8",
		"single-thread cooperative scheduler",
		"round-robin runnable actor fairness has bounded yield-progress evidence",
		"timed sleeping actors wake in deterministic deadline order",
		"linux-x64 distributed runtime only",
		"non-linux actor net pump is no-op",
		"mailbox full returns checked -2",
		"mailbox backpressure recovers after drain",
		"typed mailbox backpressure does not enqueue a partial payload",
		"message pool exhaustion returns checked -1",
		"drained message pool entries are reclaimed",
		"invalid actor handle sends return checked -3",
		"done actor sends return checked -4",
		"nonzero actor entry return is exposed only as done-state send failure",
		"no actor status, actor join, or actor exit-code API",
		"messages already queued in another actor mailbox remain receivable",
		"done actors are not restarted",
		"blocked actors continue to depend on normal message",
		"missing-node node_down remains checked distributed status evidence",
		"no automatic retry, restart, reconnect, or supervision",
		"task-group cancellation wakes recv_until",
		"task-group cancellation wakes actors already waiting on task_join_result_i32",
		"task_join_i32 wakes on task-group cancellation with raw zero value",
		"non-timed actor receives do not expose a cancellation result",
	} {
		if !hasActorBoundaryText(limits.RequiredFacts, want) {
			t.Fatalf("current limits row missing fact %q: %#v", want, limits.RequiredFacts)
		}
	}
	for _, want := range []string{
		"compiler/internal/actorsrt/actorsrt_core.go",
		"emitMailboxFullCheckForReceiverInEcx",
		"emitCheckedMessagePoolAlloc",
		"emitRecycleMessageNodeInRax",
		"emitInvalidActorHandleReturn",
		"emitActorDoneReturn",
		"emitBlockedDeadlineWakeCheck",
		"emitWaitingTaskWakeCheck",
		"emitCurrentTaskGroupCanceledCheck",
		"TestActorMailboxFullReturnsCheckedBackpressure",
		"TestActorMailboxBackpressureRecoversAfterSelfDrainBuildAndRun",
		"TestActorTaggedMailboxBackpressureRecoversAfterSelfDrainBuildAndRun",
		"TestActorTypedMailboxBackpressureRecoversWithoutPartialPayloadBuildAndRun",
		"TestActorMessagePoolReclaimsDrainedMessagesBuildAndRun",
		"TestActorMessagePoolExhaustionReturnsCheckedFailure",
		"TestActorInvalidHandleSendReturnsCheckedFailure",
		"TestActorSendToDoneActorReturnsCheckedFailure",
		"TestActorFailureNonzeroExitBecomesDoneWithoutRestartBuildAndRun",
		"TestActorLifecycleReceivesPendingMessageFromDoneSenderBuildAndRun",
		"TestActorLifecycleDoneActorWithPendingMailboxDoesNotStallBlockedActorsBuildAndRun",
		"TestActorFairnessYieldingWorkersBothMakeBoundedProgressBuildAndRun",
		"TestActorStarvationTimedSleepersWakeInDeadlineOrderBuildAndRun",
		"TestBrokerMissingDestinationNodeDownDoesNotRetryOrReconnect",
		"TestLinuxRuntimePumpsNodeDownIntoNodeStatus",
		"TestTaskGroupCancelWakesActorRecvUntilBeforeDeadlineBuildAndRun",
		"TestTaskGroupCancelWakesActorRecvMsgUntilBeforeDeadlineBuildAndRun",
		"TestTaskGroupCancelWhileActorWaitsOnJoinReturnsCanceledBuildAndRun",
		"TestTaskGroupCancelWhileActorWaitsOnJoinI32WakesWithZeroValueBuildAndRun",
		"TestTaskGroupCancelWakesJoinUntilBeforeDeadlineBuildAndRun",
		"TestTaskGroupCancelWakesSelect2BeforeDeadlineBuildAndRun",
		"docs/spec/runtime/actors.md",
		"TestActorNetPumpIsExportedButOnlyLinuxHasRuntimePump",
	} {
		if !strings.Contains(limits.Evidence, want) {
			t.Fatalf("current limits evidence missing %q: %s", want, limits.Evidence)
		}
	}

	prototype := byID[ActorRuntimeBoundarySchedulerPrototype]
	if prototype.Status != ActorRuntimeBoundaryPrototypeEvidence {
		t.Fatalf(
			"scheduler prototype status = %q, want %q",
			prototype.Status,
			ActorRuntimeBoundaryPrototypeEvidence,
		)
	}
	for _, want := range []string{
		"single-core FIFO compatibility",
		"two-core work stealing",
		"bounded typed mailbox",
		"zero_copy_move",
		"bytes_copied=0",
	} {
		if !hasActorBoundaryText(prototype.RequiredFacts, want) {
			t.Fatalf("scheduler prototype row missing fact %q: %#v", want, prototype.RequiredFacts)
		}
	}
	if !strings.Contains(prototype.Boundary, "not a production multi-threaded actor scheduler") {
		t.Fatalf("scheduler prototype boundary = %q", prototype.Boundary)
	}

	acceptance := byID[ActorRuntimeBoundaryProductionAcceptance]
	if acceptance.Status != ActorRuntimeBoundaryAcceptanceRequired {
		t.Fatalf(
			"production acceptance status = %q, want %q",
			acceptance.Status,
			ActorRuntimeBoundaryAcceptanceRequired,
		)
	}
	for _, want := range []string{
		"production task scheduler",
		"actor scheduler starvation/progress bound",
		"bounded mailbox backpressure",
		"message reclamation",
		"race-safety model",
		"cross-target distributed runtime gates",
		"blocking primitive by cancellation-source matrix",
		"structured concurrency",
	} {
		if !hasActorBoundaryText(acceptance.RequiredFacts, want) {
			t.Fatalf(
				"production acceptance row missing fact %q: %#v",
				want,
				acceptance.RequiredFacts,
			)
		}
	}

	blockers := byID[ActorRuntimeBoundaryFullClaimBlockers]
	if blockers.Status != ActorRuntimeBoundaryBlocked {
		t.Fatalf("blockers status = %q, want %q", blockers.Status, ActorRuntimeBoundaryBlocked)
	}
	for _, want := range []string{
		"production multi-threaded actor scheduler",
		"non-Linux-x64 distributed actor runtime",
		"full cancellation and structured concurrency",
		"full race-safety proof",
	} {
		if !hasActorBoundaryText(blockers.MissingFacts, want) {
			t.Fatalf("blockers row missing fact %q: %#v", want, blockers.MissingFacts)
		}
	}
}

func TestActorRuntimeProductionBoundaryAuditRejectsFakeFullProductionClaim(t *testing.T) {
	report, err := ActorRuntimeProductionBoundaryAudit()
	if err != nil {
		t.Fatal(err)
	}

	fakeClaim := report
	fakeClaim.FullProductionClaimed = true
	if err := ValidateActorRuntimeProductionBoundaryAudit(fakeClaim); err == nil ||
		!strings.Contains(err.Error(), "full production actor runtime") {
		t.Fatalf("fake full-production claim error = %v", err)
	}

	missingBlockers := cloneActorRuntimeBoundaryReport(report)
	for i := range missingBlockers.Rows {
		if missingBlockers.Rows[i].ID == ActorRuntimeBoundaryFullClaimBlockers {
			missingBlockers.Rows[i].MissingFacts = nil
		}
	}
	if err := ValidateActorRuntimeProductionBoundaryAudit(missingBlockers); err == nil ||
		!strings.Contains(err.Error(), "blockers") {
		t.Fatalf("missing blocker facts error = %v", err)
	}

	fakePromotion := cloneActorRuntimeBoundaryReport(report)
	for i := range fakePromotion.Rows {
		if fakePromotion.Rows[i].ID == ActorRuntimeBoundarySchedulerPrototype {
			fakePromotion.Rows[i].Status = ActorRuntimeBoundaryStatus("production_ready")
		}
	}
	if err := ValidateActorRuntimeProductionBoundaryAudit(fakePromotion); err == nil ||
		!strings.Contains(err.Error(), "scheduler prototype") {
		t.Fatalf("fake scheduler promotion error = %v", err)
	}

	noNonClaim := cloneActorRuntimeBoundaryReport(report)
	noNonClaim.NonClaims = nil
	if err := ValidateActorRuntimeProductionBoundaryAudit(noNonClaim); err == nil ||
		!strings.Contains(err.Error(), "non-claim") {
		t.Fatalf("missing non-claim error = %v", err)
	}
}

func hasActorBoundaryText(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}

func cloneActorRuntimeBoundaryReport(report ActorRuntimeBoundaryReport) ActorRuntimeBoundaryReport {
	clone := report
	clone.Rows = append([]ActorRuntimeBoundaryRow(nil), report.Rows...)
	clone.NonClaims = append([]string(nil), report.NonClaims...)
	return clone
}

// ---- runtime_source_parity_test.go ----

func TestSelfhostActorRuntimeSourcesMatchCanonicalRT(t *testing.T) {
	root := repoRootFromActorsRTTest(t)
	canonicalDir := filepath.Join(root, "__rt")
	selfhostDir := filepath.Join(root, "compiler", "selfhostrt")

	canonical, err := filepath.Glob(filepath.Join(canonicalDir, "actors_*.tetra"))
	if err != nil {
		t.Fatalf("glob canonical actor runtime files: %v", err)
	}
	if len(canonical) == 0 {
		t.Fatalf("no canonical actor runtime files found under %s", canonicalDir)
	}
	sort.Strings(canonical)

	for _, canonicalPath := range canonical {
		name := filepath.Base(canonicalPath)
		selfhostPath := filepath.Join(selfhostDir, name)
		t.Run(name, func(t *testing.T) {
			canonicalRaw, err := os.ReadFile(canonicalPath)
			if err != nil {
				t.Fatalf("read canonical runtime source: %v", err)
			}
			selfhostRaw, err := os.ReadFile(selfhostPath)
			if err != nil {
				t.Fatalf("read selfhost runtime source: %v", err)
			}
			if !bytes.Equal(canonicalRaw, selfhostRaw) {
				canonicalSum := sha256.Sum256(canonicalRaw)
				selfhostSum := sha256.Sum256(selfhostRaw)
				t.Fatalf(
					"selfhost actor runtime source drift for %s: __rt sha256=%x selfhostrt sha256=%x",
					name,
					canonicalSum,
					selfhostSum,
				)
			}
		})
	}

	selfhost, err := filepath.Glob(filepath.Join(selfhostDir, "actors_*.tetra"))
	if err != nil {
		t.Fatalf("glob selfhost actor runtime files: %v", err)
	}
	canonicalNames := map[string]bool{}
	for _, path := range canonical {
		canonicalNames[filepath.Base(path)] = true
	}
	for _, path := range selfhost {
		name := filepath.Base(path)
		if !canonicalNames[name] {
			t.Fatalf("selfhost actor runtime source %s has no canonical __rt peer", name)
		}
	}
}

func TestActorRuntimePOCSourcesRemainHistoricalReferences(t *testing.T) {
	root := repoRootFromActorsRTTest(t)
	historical := []string{
		filepath.Join("__rt", "actors_poc_sysv.tetra"),
		filepath.Join("__rt", "actors_poc_win64.tetra"),
		filepath.Join("compiler", "selfhostrt", "actors_poc_sysv.tetra"),
		filepath.Join("compiler", "selfhostrt", "actors_poc_win64.tetra"),
	}
	for _, rel := range historical {
		t.Run(rel, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(root, rel))
			if err != nil {
				t.Fatalf("read historical PoC runtime source: %v", err)
			}
			if !bytes.Contains(raw, []byte("actors_poc")) {
				t.Fatalf("%s does not look like a historical actors_poc module", rel)
			}
		})
	}

	productionSelectionFiles := []string{
		filepath.Join("compiler", "compiler_build_runtime.go"),
		filepath.Join("compiler", "internal", "actorsrt", "actorsrt_core.go"),
	}
	for _, rel := range productionSelectionFiles {
		t.Run(rel, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(root, rel))
			if err != nil {
				t.Fatalf("read production runtime selection file: %v", err)
			}
			if bytes.Contains(raw, []byte("actors_poc")) {
				t.Fatalf("%s promotes historical actors_poc runtime into production selection", rel)
			}
		})
	}
}

func repoRootFromActorsRTTest(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "__rt")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "compiler", "selfhostrt")); err == nil {
				return dir
			}
		}
		if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root from %s", file)
		}
		if strings.TrimSpace(parent) == "" {
			t.Fatalf("invalid parent while walking from %s", file)
		}
		dir = parent
	}
}

// ---- typed_task_slots_test.go ----

func TestEmitTaskJoinTypedSlotBounds(t *testing.T) {
	tests := []struct {
		name  string
		slots int
		ok    bool
	}{
		{name: "slot_1_rejected", slots: 1, ok: false},
		{name: "slot_2_allowed", slots: 2, ok: true},
		{name: "slot_4_allowed", slots: 4, ok: true},
		{name: "slot_5_allowed", slots: 5, ok: true},
		{name: "slot_8_allowed", slots: 8, ok: true},
		{name: "slot_9_rejected", slots: 9, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &x64.Emitter{}
			var patches []callPatch
			err := emitTaskJoinTyped(e, tt.slots, &patches)
			if tt.ok {
				if err != nil {
					t.Fatalf("emitTaskJoinTyped(%d): %v", tt.slots, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error for slot count %d", tt.slots)
			}
			if !strings.Contains(err.Error(), "unsupported typed task join slot count") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestEmitTaskJoinTypedWrapperWindowsX64SlotBounds(t *testing.T) {
	tests := []struct {
		name  string
		slots int
		ok    bool
	}{
		{name: "slot_1_rejected", slots: 1, ok: false},
		{name: "slot_2_allowed", slots: 2, ok: true},
		{name: "slot_4_allowed", slots: 4, ok: true},
		{name: "slot_5_allowed", slots: 5, ok: true},
		{name: "slot_8_allowed", slots: 8, ok: true},
		{name: "slot_9_rejected", slots: 9, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &x64.Emitter{}
			var patches []callPatch
			err := emitTaskJoinTypedWrapperWindowsX64(
				e,
				tt.slots,
				"__tetra_task_join_typed_impl",
				&patches,
			)
			if tt.ok {
				if err != nil {
					t.Fatalf("emitTaskJoinTypedWrapperWindowsX64(%d): %v", tt.slots, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error for slot count %d", tt.slots)
			}
			if !strings.Contains(err.Error(), "unsupported typed task join wrapper slots") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
