package abisuite

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ctarget "tetra_language/compiler/target"
)

type RuntimeBoundaryDeps struct {
	BuildExecutable                 func(srcPath string, outPath string, target string) error
	DiagnosticFromError             func(error) DiagnosticSummary
	TargetRuntimeDiagnosticCode     string
	TargetSupportsNetRuntimeSymbols func(target string, symbols []string) bool
	RequiredNetRuntimeSymbols       func() []string
	NetRuntimeSymbolForBuiltin      func(name string) (string, bool)
}

type DiagnosticSummary struct {
	Code     string
	Message  string
	Severity string
	Hint     string
}

type RuntimeBoundaryCase struct {
	Name        string
	Source      string
	WantMessage string
}

func CheckStdlibRuntimeBoundaryDiagnostics(tgt ctarget.Target, deps RuntimeBoundaryDeps) error {
	tmpDir, err := os.MkdirTemp("", "tetra-stdlib-runtime-boundary-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	cases := []struct {
		name        string
		runtimeName string
		src         string
	}{}
	if tgt.Triple != "linux-x86" && tgt.Triple != "linux-x32" {
		cases = append(cases, struct {
			name        string
			runtimeName string
			src         string
		}{
			name:        "filesystem",
			runtimeName: "filesystem",
			src: `
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        if core.fs_exists("README.md", cap):
            return 0
    return 1
`,
		})
	}
	if !targetSupportsNetRuntimeSymbols(deps, tgt.Triple, requiredNetRuntimeSymbols(deps)) {
		cases = append(cases, struct {
			name        string
			runtimeName string
			src         string
		}{
			name:        "networking",
			runtimeName: "networking",
			src: `
func main() -> Int
uses capability, io:
    unsafe:
        let cap: cap.io = core.cap_io()
        return core.net_epoll_create(cap)
    return 1
`,
		})
	}

	for _, tc := range cases {
		srcPath := filepath.Join(tmpDir, tc.name+".tetra")
		outPath := filepath.Join(tmpDir, tc.name+"-"+tgt.Triple)
		if err := os.WriteFile(srcPath, []byte(tc.src), 0o644); err != nil {
			return err
		}
		err := buildRuntimeBoundaryExecutable(deps, srcPath, outPath, tgt.Triple)
		if err == nil {
			return fmt.Errorf("%s accepted unsupported %s stdlib runtime boundary", tgt.Triple, tc.runtimeName)
		}
		diag := runtimeBoundaryDiagnostic(deps, err)
		wantMessage := fmt.Sprintf("%s runtime not supported on %s", tc.runtimeName, tgt.Triple)
		if diag.Code != targetRuntimeDiagnosticCode(deps) || diag.Severity != "error" || diag.Message != wantMessage {
			return fmt.Errorf("%s %s runtime diagnostic = %#v, want code=%s severity=error message=%q", tgt.Triple, tc.runtimeName, diag, targetRuntimeDiagnosticCode(deps), wantMessage)
		}
		if !strings.Contains(diag.Hint, "Build this source for linux-x64") {
			return fmt.Errorf("%s %s runtime hint = %q, want linux-x64 guidance", tgt.Triple, tc.runtimeName, diag.Hint)
		}
		if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
			return fmt.Errorf("%s %s runtime rejection wrote output %s (stat err=%v)", tgt.Triple, tc.runtimeName, outPath, statErr)
		}
	}
	return nil
}

func CheckTargetRuntimeBoundaryDiagnostics(tgt ctarget.Target, deps RuntimeBoundaryDeps) error {
	cases, err := targetRuntimeBoundaryCases(tgt)
	if err != nil {
		return err
	}
	return checkRuntimeBoundaryDiagnostics(tgt, "tetra-target-runtime-boundary-*", cases, deps)
}

func CheckSurfaceDistributedRuntimeBoundaryDiagnostics(tgt ctarget.Target, deps RuntimeBoundaryDeps) error {
	return checkRuntimeBoundaryDiagnostics(tgt, "tetra-surface-distributed-runtime-boundary-*", []RuntimeBoundaryCase{
		{
			Name: "surface",
			Source: `
func main() -> Int
uses surface:
    return core.surface_open("demo", 10, 10)
`,
			WantMessage: "surface runtime not supported on " + tgt.Triple,
		},
		{
			Name: "distributed_actors",
			Source: `
func main() -> Int
uses actors, runtime:
    return core.actor_node_status(2)
`,
			WantMessage: "distributed actors runtime not supported on " + tgt.Triple,
		},
	}, deps)
}

func CheckNetworkingRuntimeBoundaryDiagnostics(tgt ctarget.Target, deps RuntimeBoundaryDeps) error {
	cases := []struct {
		name    string
		uses    string
		prelude string
		expr    string
	}{
		{name: "socket_tcp4", expr: "core.net_socket_tcp4(cap)"},
		{name: "bind_tcp4_loopback", expr: "core.net_bind_tcp4_loopback(3, 0, cap)"},
		{name: "connect_tcp4_loopback", expr: "core.net_connect_tcp4_loopback(3, 0, cap)"},
		{name: "listen", expr: "core.net_listen(3, 8, cap)"},
		{name: "accept4", expr: "core.net_accept4(3, 0, cap)"},
		{name: "read", uses: "alloc, capability, io, mem", prelude: "        var buf: []u8 = make_u8(4)\n", expr: "core.net_read(3, buf, 0, 1, cap)"},
		{name: "recv", uses: "alloc, capability, io, mem", prelude: "        var buf: []u8 = make_u8(4)\n", expr: "core.net_recv(3, buf, 0, 1, cap)"},
		{name: "write", uses: "alloc, capability, io, mem", prelude: "        var buf: []u8 = make_u8(4)\n", expr: "core.net_write(3, buf, 0, 1, cap)"},
		{name: "send", uses: "alloc, capability, io, mem", prelude: "        var buf: []u8 = make_u8(4)\n", expr: "core.net_send(3, buf, 0, 1, cap)"},
		{name: "epoll_create", expr: "core.net_epoll_create(cap)"},
		{name: "epoll_ctl_add_read", expr: "core.net_epoll_ctl_add_read(4, 3, cap)"},
		{name: "epoll_ctl_add_read_write", expr: "core.net_epoll_ctl_add_read_write(4, 3, cap)"},
		{name: "epoll_ctl_mod_read", expr: "core.net_epoll_ctl_mod_read(4, 3, cap)"},
		{name: "epoll_ctl_mod_read_write", expr: "core.net_epoll_ctl_mod_read_write(4, 3, cap)"},
		{name: "epoll_ctl_delete", expr: "core.net_epoll_ctl_delete(4, 3, cap)"},
		{name: "epoll_wait_one", expr: "core.net_epoll_wait_one(4, 0, cap)"},
		{name: "epoll_wait_one_into", uses: "alloc, capability, io, mem", prelude: "        var event: []i32 = make_i32(2)\n", expr: "core.net_epoll_wait_one_into(4, event, 0, cap)"},
		{name: "set_nonblocking", expr: "core.net_set_nonblocking(3, cap)"},
		{name: "set_reuseport", expr: "core.net_set_reuseport(3, cap)"},
		{name: "set_tcp_nodelay", expr: "core.net_set_tcp_nodelay(3, cap)"},
		{name: "close", expr: "core.net_close(3, cap)"},
	}
	boundaryCases := make([]RuntimeBoundaryCase, 0, len(cases))
	for _, tc := range cases {
		builtinName := tc.expr
		if openParen := strings.IndexByte(builtinName, '('); openParen >= 0 {
			builtinName = builtinName[:openParen]
		}
		if symbol, ok := netRuntimeSymbolForBuiltin(deps, builtinName); ok && targetSupportsNetRuntimeSymbols(deps, tgt.Triple, []string{symbol}) {
			continue
		}
		uses := tc.uses
		if uses == "" {
			uses = "capability, io"
		}
		boundaryCases = append(boundaryCases, RuntimeBoundaryCase{
			Name: tc.name,
			Source: "func main() -> Int\nuses " + uses + ":\n    unsafe:\n        let cap: cap.io = core.cap_io()\n" +
				tc.prelude +
				"        return " + tc.expr + "\n    return 1\n",
			WantMessage: "networking runtime not supported on " + tgt.Triple,
		})
	}
	return checkRuntimeBoundaryDiagnostics(tgt, "tetra-networking-runtime-boundary-*", boundaryCases, deps)
}

func checkRuntimeBoundaryDiagnostics(tgt ctarget.Target, tmpPattern string, cases []RuntimeBoundaryCase, deps RuntimeBoundaryDeps) error {
	tmpDir, err := os.MkdirTemp("", tmpPattern)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	for _, tc := range cases {
		srcPath := filepath.Join(tmpDir, tc.Name+".tetra")
		outPath := filepath.Join(tmpDir, tc.Name+"-"+tgt.Triple)
		if err := os.WriteFile(srcPath, []byte(tc.Source), 0o644); err != nil {
			return err
		}
		err := buildRuntimeBoundaryExecutable(deps, srcPath, outPath, tgt.Triple)
		if err == nil {
			return fmt.Errorf("%s accepted unsupported %s target runtime boundary", tgt.Triple, tc.Name)
		}
		diag := runtimeBoundaryDiagnostic(deps, err)
		if diag.Code != targetRuntimeDiagnosticCode(deps) || diag.Severity != "error" || diag.Message != tc.WantMessage {
			return fmt.Errorf("%s %s runtime diagnostic = %#v, want code=%s severity=error message=%q", tgt.Triple, tc.Name, diag, targetRuntimeDiagnosticCode(deps), tc.WantMessage)
		}
		if !strings.Contains(diag.Hint, "Build this source for linux-x64") {
			return fmt.Errorf("%s %s runtime hint = %q, want linux-x64 guidance", tgt.Triple, tc.Name, diag.Hint)
		}
		if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
			return fmt.Errorf("%s %s runtime rejection wrote output %s (stat err=%v)", tgt.Triple, tc.Name, outPath, statErr)
		}
	}
	return nil
}

func targetRuntimeBoundaryCases(tgt ctarget.Target) ([]RuntimeBoundaryCase, error) {
	switch tgt.Triple {
	case "linux-x86":
		return []RuntimeBoundaryCase{
			{
				Name: "actor_fanout_over_two_task",
				Source: `
func slow() -> Int:
    return 1

func fast() -> Int:
    return 2

func extra() -> Int:
    return 3

func main() -> Int
uses runtime:
    let _slow: task.i32 = core.task_spawn_i32("slow")
    let _fast: task.i32 = core.task_spawn_i32("fast")
    let _extra: task.i32 = core.task_spawn_i32("extra")
    return 0
`,
				WantMessage: "actor fanout above 2 runtime not supported on linux-x86",
			},
			{
				Name: "actor_fanout_over_two_actors",
				Source: `
func slow() -> Int
uses actors:
    return 1

func fast() -> Int
uses actors:
    return 2

func extra() -> Int
uses actors:
    return 3

func main() -> Int
uses actors, runtime:
    let _slow: actor = core.spawn("slow")
    let _fast: actor = core.spawn("fast")
    let _extra: actor = core.spawn("extra")
    return 0
`,
				WantMessage: "actor fanout above 2 runtime not supported on linux-x86",
			},
			{
				Name: "actor_fanout_over_two_task_group",
				Source: `
func slow() -> Int:
    return 1

func fast() -> Int:
    return 2

func extra() -> Int:
    return 3

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let _slow: task.i32 = core.task_spawn_group_i32(group, "slow")
    let _fast: task.i32 = core.task_spawn_group_i32(group, "fast")
    let _extra: task.i32 = core.task_spawn_group_i32(group, "extra")
    return 0
`,
				WantMessage: "actor fanout above 2 runtime not supported on linux-x86",
			},
		}, nil
	case "linux-x32":
		return []RuntimeBoundaryCase{
			{
				Name: "actor_fanout_over_two_actors",
				Source: `
func slow() -> Int
uses actors:
    return 1

func fast() -> Int
uses actors:
    return 2

func extra() -> Int
uses actors:
    return 3

func main() -> Int
uses actors, runtime:
    let _slow: actor = core.spawn("slow")
    let _fast: actor = core.spawn("fast")
    let _extra: actor = core.spawn("extra")
    return 0
`,
				WantMessage: "actor fanout above 2 runtime not supported on linux-x32",
			},
			{
				Name: "actor_fanout_over_two_task",
				Source: `
func slow() -> Int:
    return 1

func fast() -> Int:
    return 2

func extra() -> Int:
    return 3

func main() -> Int
uses runtime:
    let _slow: task.i32 = core.task_spawn_i32("slow")
    let _fast: task.i32 = core.task_spawn_i32("fast")
    let _extra: task.i32 = core.task_spawn_i32("extra")
    return 0
`,
				WantMessage: "actor fanout above 2 runtime not supported on linux-x32",
			},
		}, nil
	default:
		return nil, fmt.Errorf("target runtime boundary suite is not defined for %s", tgt.Triple)
	}
}

func buildRuntimeBoundaryExecutable(deps RuntimeBoundaryDeps, srcPath string, outPath string, target string) error {
	if deps.BuildExecutable == nil {
		return fmt.Errorf("missing runtime boundary build executable callback")
	}
	return deps.BuildExecutable(srcPath, outPath, target)
}

func runtimeBoundaryDiagnostic(deps RuntimeBoundaryDeps, err error) DiagnosticSummary {
	if deps.DiagnosticFromError == nil {
		return DiagnosticSummary{Message: err.Error()}
	}
	return deps.DiagnosticFromError(err)
}

func targetRuntimeDiagnosticCode(deps RuntimeBoundaryDeps) string {
	if deps.TargetRuntimeDiagnosticCode != "" {
		return deps.TargetRuntimeDiagnosticCode
	}
	return "TETRA3003"
}

func targetSupportsNetRuntimeSymbols(deps RuntimeBoundaryDeps, target string, symbols []string) bool {
	if deps.TargetSupportsNetRuntimeSymbols == nil {
		return false
	}
	return deps.TargetSupportsNetRuntimeSymbols(target, symbols)
}

func requiredNetRuntimeSymbols(deps RuntimeBoundaryDeps) []string {
	if deps.RequiredNetRuntimeSymbols == nil {
		return nil
	}
	return deps.RequiredNetRuntimeSymbols()
}

func netRuntimeSymbolForBuiltin(deps RuntimeBoundaryDeps, name string) (string, bool) {
	if deps.NetRuntimeSymbolForBuiltin == nil {
		return "", false
	}
	return deps.NetRuntimeSymbolForBuiltin(name)
}
