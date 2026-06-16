package abisuite

import (
	"strings"
	"testing"

	ctarget "tetra_language/compiler/target"
)

func TestRunTargetChecksRoutesByParsedTarget(t *testing.T) {
	var routed []string
	runners := TargetCheckRunners{
		X86:  targetRoutingRunner("x86", &routed),
		X32:  targetRoutingRunner("x32", &routed),
		X64:  targetRoutingRunner("x64", &routed),
		WASM: targetRoutingRunner("wasm", &routed),
	}

	for _, tc := range []struct {
		target    string
		wantRoute string
		wantCheck string
	}{
		{target: "x86", wantRoute: "x86:linux-x86", wantCheck: "x86 check"},
		{target: "x32", wantRoute: "x32:linux-x32", wantCheck: "x32 check"},
		{target: "linux-x64", wantRoute: "x64:linux-x64", wantCheck: "x64 check"},
		{target: "wasm32-wasi", wantRoute: "wasm:wasm32-wasi", wantCheck: "wasm check"},
	} {
		t.Run(tc.target, func(t *testing.T) {
			routed = nil
			checks, err := RunTargetChecks(tc.target, runners)
			if err != nil {
				t.Fatalf("RunTargetChecks(%s): %v", tc.target, err)
			}
			if strings.Join(routed, ",") != tc.wantRoute {
				t.Fatalf("route = %#v, want %q", routed, tc.wantRoute)
			}
			if len(checks) != 1 || checks[0].Name != tc.wantCheck {
				t.Fatalf("checks = %#v, want %q", checks, tc.wantCheck)
			}
		})
	}
}

func TestRunTargetChecksRejectsMissingRunner(t *testing.T) {
	_, err := RunTargetChecks("x86", TargetCheckRunners{})
	if err == nil || !strings.Contains(err.Error(), "missing ABI suite runner for x86") {
		t.Fatalf("RunTargetChecks missing runner error = %v", err)
	}
}

func TestX64CheckPrefix(t *testing.T) {
	for _, tc := range []struct {
		target string
		want   string
	}{
		{target: "linux-x64", want: "x64"},
		{target: "windows-x64", want: "windows-x64"},
		{target: "macos-x64", want: "macos-x64"},
	} {
		t.Run(tc.target, func(t *testing.T) {
			tgt, err := ctarget.Parse(tc.target)
			if err != nil {
				t.Fatalf("Parse(%s): %v", tc.target, err)
			}
			if got := X64CheckPrefix(tgt); got != tc.want {
				t.Fatalf("X64CheckPrefix(%s) = %q, want %q", tc.target, got, tc.want)
			}
		})
	}
}

func targetRoutingRunner(name string, routed *[]string) TargetCheckRunner {
	return func(tgt ctarget.Target) []Check {
		*routed = append(*routed, name+":"+tgt.Triple)
		return []Check{{Name: name + " check"}}
	}
}
