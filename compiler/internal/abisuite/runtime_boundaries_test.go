package abisuite

import (
	"fmt"
	"os"
	"strings"
	"testing"

	ctarget "tetra_language/compiler/target"
)

type runtimeBoundaryTestError struct {
	message string
}

func (e runtimeBoundaryTestError) Error() string {
	return e.message
}

func TestRuntimeBoundaryDiagnosticsUseCallbacks(t *testing.T) {
	var built []string
	deps := RuntimeBoundaryDeps{
		TargetRuntimeDiagnosticCode: "TETRA3003",
		BuildExecutable: func(srcPath string, outPath string, target string) error {
			built = append(built, outPath)
			base := outPath[strings.LastIndex(outPath, string(os.PathSeparator))+1:]
			switch {
			case strings.HasPrefix(base, "filesystem-"):
				return runtimeBoundaryTestError{message: "filesystem runtime not supported on " + target}
			case strings.HasPrefix(base, "networking-") || strings.Contains(base, "socket_tcp4") || strings.Contains(base, "epoll_create"):
				return runtimeBoundaryTestError{message: "networking runtime not supported on " + target}
			case strings.Contains(base, "actor_fanout"):
				return runtimeBoundaryTestError{message: "actor fanout above 2 runtime not supported on " + target}
			case strings.Contains(base, "surface"):
				return runtimeBoundaryTestError{message: "surface runtime not supported on " + target}
			case strings.Contains(base, "distributed"):
				return runtimeBoundaryTestError{message: "distributed actors runtime not supported on " + target}
			case target == "linux-x86":
				return runtimeBoundaryTestError{message: "networking runtime not supported on " + target}
			default:
				return runtimeBoundaryTestError{message: fmt.Sprintf("unexpected build %s for %s", base, target)}
			}
		},
		DiagnosticFromError: func(err error) DiagnosticSummary {
			return DiagnosticSummary{
				Code:     "TETRA3003",
				Message:  err.Error(),
				Severity: "error",
				Hint:     "Build this source for linux-x64",
			}
		},
		TargetSupportsNetRuntimeSymbols: func(target string, symbols []string) bool {
			return false
		},
		RequiredNetRuntimeSymbols: func() []string {
			return []string{"__tetra_net_socket_tcp4"}
		},
		NetRuntimeSymbolForBuiltin: func(name string) (string, bool) {
			switch name {
			case "core.net_socket_tcp4":
				return "__tetra_net_socket_tcp4", true
			case "core.net_epoll_create":
				return "__tetra_net_epoll_create", true
			default:
				return "", false
			}
		},
	}

	wasm, err := ctarget.Parse("wasm32-wasi")
	if err != nil {
		t.Fatalf("parse wasm target: %v", err)
	}
	x86, err := ctarget.Parse("linux-x86")
	if err != nil {
		t.Fatalf("parse x86 target: %v", err)
	}

	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{name: "stdlib", run: func() error { return CheckStdlibRuntimeBoundaryDiagnostics(wasm, deps) }},
		{name: "target", run: func() error { return CheckTargetRuntimeBoundaryDiagnostics(x86, deps) }},
		{name: "surface", run: func() error { return CheckSurfaceDistributedRuntimeBoundaryDiagnostics(x86, deps) }},
		{name: "networking", run: func() error { return CheckNetworkingRuntimeBoundaryDiagnostics(x86, deps) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err != nil {
				t.Fatalf("runtime boundary check: %v", err)
			}
		})
	}
	if len(built) == 0 {
		t.Fatalf("runtime boundary checks did not call BuildExecutable")
	}
}
