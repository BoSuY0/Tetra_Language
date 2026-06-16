package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNetRuntimeRejectsUnsupportedNativeTargets(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_main.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        return core.net_epoll_create(cap)
    return 1
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	for _, tc := range []struct {
		target string
		want   string
	}{
		{target: "macos-x64", want: "macos-x64"},
		{target: "windows-x64", want: "windows-x64"},
	} {
		t.Run(tc.target, func(t *testing.T) {
			outPath := filepath.Join(tmp, "net-"+tc.target)
			_, err := BuildFileWithStatsOpt(srcPath, outPath, tc.target, BuildOptions{Jobs: 1})
			if err == nil {
				t.Fatalf("expected unsupported networking runtime diagnostic")
			}
			want := "networking runtime not supported on " + tc.want
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, want %q", err, want)
			}
		})
	}
}

func TestX86NetworkingLifecycleRuntimeBuildsAndRunsWhenHostSupportsX86(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_socket_lifecycle_x86.tetra")
	outPath := filepath.Join(tmp, "net-socket-lifecycle-x86")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        if fd < 0:
            return 11
        let nb: Int = core.net_set_nonblocking(fd, cap)
        let closed: Int = core.net_close(fd, cap)
        if nb < 0:
            return 12
        if closed != 0:
            return 13
    return 0
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x86 net socket lifecycle runtime: %v", err)
	}
	assertELF32Machine(t, outPath, "x86 net socket lifecycle", 0x03)
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if stdout != "" || code != 0 {
		t.Fatalf("x86 net socket lifecycle runtime stdout=%q exit=%d, want empty/0", stdout, code)
	}
}

func TestX32NetworkingLifecycleRuntimeBuildsAndRunsWhenHostSupportsX32(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_socket_lifecycle_x32.tetra")
	outPath := filepath.Join(tmp, "net-socket-lifecycle-x32")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        if fd < 0:
            return 11
        let nb: Int = core.net_set_nonblocking(fd, cap)
        let closed: Int = core.net_close(fd, cap)
        if nb < 0:
            return 12
        if closed != 0:
            return 13
    return 0
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x32 net socket lifecycle runtime: %v", err)
	}
	assertELF32Machine(t, outPath, "x32 net socket lifecycle", 0x3e)
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if stdout != "" || code != 0 {
		t.Fatalf("x32 net socket lifecycle runtime stdout=%q exit=%d, want empty/0", stdout, code)
	}
}

func TestX86NetworkingSocketOptionsBuildsAndRunsWhenHostSupportsX86(t *testing.T) {
	testTargetNetworkingSocketOptions(t, targetNetworkingSmoke{
		target:      "x86",
		label:       "x86",
		wantMachine: 0x03,
	})
}

func TestX32NetworkingSocketOptionsBuildsAndRunsWhenHostSupportsX32(t *testing.T) {
	testTargetNetworkingSocketOptions(t, targetNetworkingSmoke{
		target:      "x32",
		label:       "x32",
		wantMachine: 0x3e,
	})
}

func TestX86NetworkingTCPClientReadWriteBuildsAndRunsWhenHostSupportsX86(t *testing.T) {
	testTargetNetworkingTCPClientReadWrite(t, targetNetworkingSmoke{
		target:      "x86",
		label:       "x86",
		wantMachine: 0x03,
	})
}

func TestX32NetworkingTCPClientReadWriteBuildsAndRunsWhenHostSupportsX32(t *testing.T) {
	testTargetNetworkingTCPClientReadWrite(t, targetNetworkingSmoke{
		target:      "x32",
		label:       "x32",
		wantMachine: 0x3e,
	})
}

func TestX86NetworkingTCPServerRecvSendBuildsAndRunsWhenHostSupportsX86(t *testing.T) {
	testTargetNetworkingTCPServerRecvSend(t, targetNetworkingSmoke{
		target:      "x86",
		label:       "x86",
		wantMachine: 0x03,
	})
}

func TestX32NetworkingTCPServerRecvSendBuildsAndRunsWhenHostSupportsX32(t *testing.T) {
	testTargetNetworkingTCPServerRecvSend(t, targetNetworkingSmoke{
		target:      "x32",
		label:       "x32",
		wantMachine: 0x3e,
	})
}

func TestX86NetworkingEpollControlLifecycleBuildsAndRunsWhenHostSupportsX86(t *testing.T) {
	testTargetNetworkingEpollControlLifecycle(t, targetNetworkingSmoke{
		target:      "x86",
		label:       "x86",
		wantMachine: 0x03,
	})
}

func TestX32NetworkingEpollControlLifecycleBuildsAndRunsWhenHostSupportsX32(t *testing.T) {
	testTargetNetworkingEpollControlLifecycle(t, targetNetworkingSmoke{
		target:      "x32",
		label:       "x32",
		wantMachine: 0x3e,
	})
}

func TestX86NetworkingEpollReadinessBuildsAndRunsWhenHostSupportsX86(t *testing.T) {
	testTargetNetworkingEpollReadiness(t, targetNetworkingSmoke{
		target:      "x86",
		label:       "x86",
		wantMachine: 0x03,
	})
}

func TestX32NetworkingEpollReadinessBuildsAndRunsWhenHostSupportsX32(t *testing.T) {
	testTargetNetworkingEpollReadiness(t, targetNetworkingSmoke{
		target:      "x32",
		label:       "x32",
		wantMachine: 0x3e,
	})
}

func TestX86NetworkingLifecycleRuntimeComposesWithTaskScheduler(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_socket_task_x86.tetra")
	outPath := filepath.Join(tmp, "net-socket-task-x86")
	if err := os.WriteFile(srcPath, []byte(`
func worker() -> Int:
    return 7

func main() -> Int
uses capability, io, runtime:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        if fd < 0:
            return 11
        let nb: Int = core.net_set_nonblocking(fd, cap)
        let closed: Int = core.net_close(fd, cap)
        if nb < 0:
            return 12
        if closed != 0:
            return 13
    let task: task.i32 = core.task_spawn_i32("worker")
    let value: Int = core.task_join_i32(task)
    return value - 7
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x86 net socket task runtime: %v", err)
	}
	assertELF32Machine(t, outPath, "x86 net socket task", 0x03)
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if stdout != "" || code != 0 {
		t.Fatalf("x86 net socket task runtime stdout=%q exit=%d, want empty/0", stdout, code)
	}
}

func TestX32NetworkingLifecycleRuntimeComposesWithTaskScheduler(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "net_socket_task_x32.tetra")
	outPath := filepath.Join(tmp, "net-socket-task-x32")
	if err := os.WriteFile(srcPath, []byte(`
func worker() -> Int:
    return 7

func main() -> Int
uses capability, io, runtime:
    unsafe:
        let cap: cap.io = core.cap_io()
        let fd: Int = core.net_socket_tcp4(cap)
        if fd < 0:
            return 11
        let nb: Int = core.net_set_nonblocking(fd, cap)
        let closed: Int = core.net_close(fd, cap)
        if nb < 0:
            return 12
        if closed != 0:
            return 13
    let task: task.i32 = core.task_spawn_i32("worker")
    let value: Int = core.task_join_i32(task)
    return value - 7
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "x32", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build x32 net socket task runtime: %v", err)
	}
	assertELF32Machine(t, outPath, "x32 net socket task", 0x3e)
	stdout, code := runBinaryOrSkipUnsupportedTarget(t, outPath)
	if stdout != "" || code != 0 {
		t.Fatalf("x32 net socket task runtime stdout=%q exit=%d, want empty/0", stdout, code)
	}
}
