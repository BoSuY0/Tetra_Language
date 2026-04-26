package target

import "testing"

func TestParse(t *testing.T) {
	for _, triple := range SupportedTriples() {
		tgt, err := Parse(triple)
		if err != nil {
			t.Fatalf("Parse(%q): %v", triple, err)
		}
		if tgt.Triple != triple {
			t.Fatalf("triple mismatch: got=%q want=%q", tgt.Triple, triple)
		}
		if triple == "windows-x64" && tgt.ExeExt != ".exe" {
			t.Fatalf("windows exe ext mismatch: %q", tgt.ExeExt)
		}
		if triple != "windows-x64" && tgt.ExeExt != "" {
			t.Fatalf("non-windows exe ext mismatch: %q", tgt.ExeExt)
		}
	}
}

func TestTargetListsAreStable(t *testing.T) {
	if got := SupportedTriples(); len(got) != 3 || got[0] != "linux-x64" || got[1] != "windows-x64" || got[2] != "macos-x64" {
		t.Fatalf("supported triples = %#v", got)
	}
	if got := BuildOnlyTriples(); len(got) != 2 || got[0] != "wasm32-wasi" || got[1] != "wasm32-web" {
		t.Fatalf("build-only triples = %#v", got)
	}
	if got := PlannedTriples(); len(got) != 0 {
		t.Fatalf("planned triples = %#v", got)
	}
}

func TestParseRejectsUnknown(t *testing.T) {
	if _, err := Parse("plan9-x64"); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestParseAcceptsWASMBuildOnlyTarget(t *testing.T) {
	tgt, err := Parse("wasm32-wasi")
	if err != nil {
		t.Fatalf("Parse(wasm32-wasi): %v", err)
	}
	if tgt.Triple != "wasm32-wasi" || tgt.ExeExt != ".wasm" {
		t.Fatalf("wasm32-wasi target = %#v", tgt)
	}
	if !IsBuildOnlyTarget("wasm32-wasi") {
		t.Fatalf("IsBuildOnlyTarget(wasm32-wasi) = false")
	}
	if IsPlannedTarget("wasm32-wasi") {
		t.Fatalf("IsPlannedTarget(wasm32-wasi) = true")
	}
}

func TestParseAcceptsWASMWebBuildOnlyTarget(t *testing.T) {
	tgt, err := Parse("wasm32-web")
	if err != nil {
		t.Fatalf("Parse(wasm32-web): %v", err)
	}
	if tgt.Triple != "wasm32-web" || tgt.ExeExt != ".wasm" {
		t.Fatalf("wasm32-web target = %#v", tgt)
	}
	if !IsBuildOnlyTarget("wasm32-web") {
		t.Fatalf("IsBuildOnlyTarget(wasm32-web) = false")
	}
	if IsPlannedTarget("wasm32-web") {
		t.Fatalf("IsPlannedTarget(wasm32-web) = true")
	}
}

func TestParseRejectsUnknownAsUnplanned(t *testing.T) {
	_, err := Parse("plan9-x64")
	targetErr, ok := err.(UnsupportedTargetError)
	if !ok {
		t.Fatalf("error type = %T, want UnsupportedTargetError", err)
	}
	if targetErr.Planned {
		t.Fatalf("unknown target marked planned: %#v", targetErr)
	}
}

func TestTargetCapabilitiesForDebugInfoAndReleaseOptimize(t *testing.T) {
	native, err := Parse("linux-x64")
	if err != nil {
		t.Fatalf("Parse(linux-x64): %v", err)
	}
	if !native.SupportsDebugInfo {
		t.Fatalf("linux-x64 SupportsDebugInfo = false")
	}
	if !native.SupportsReleaseOptimize {
		t.Fatalf("linux-x64 SupportsReleaseOptimize = false")
	}

	wasmWASI, err := Parse("wasm32-wasi")
	if err != nil {
		t.Fatalf("Parse(wasm32-wasi): %v", err)
	}
	if wasmWASI.SupportsDebugInfo {
		t.Fatalf("wasm32-wasi SupportsDebugInfo = true")
	}
	if !wasmWASI.SupportsReleaseOptimize {
		t.Fatalf("wasm32-wasi SupportsReleaseOptimize = false")
	}

	wasmWeb, err := Parse("wasm32-web")
	if err != nil {
		t.Fatalf("Parse(wasm32-web): %v", err)
	}
	if wasmWeb.SupportsDebugInfo {
		t.Fatalf("wasm32-web SupportsDebugInfo = true")
	}
	if !wasmWeb.SupportsReleaseOptimize {
		t.Fatalf("wasm32-web SupportsReleaseOptimize = false")
	}
}
