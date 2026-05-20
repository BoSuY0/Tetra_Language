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
		if triple == "wasm32-wasi" || triple == "wasm32-web" {
			if tgt.ExeExt != ".wasm" {
				t.Fatalf("wasm exe ext mismatch: %q", tgt.ExeExt)
			}
			continue
		}
		if triple != "windows-x64" && tgt.ExeExt != "" {
			t.Fatalf("native non-windows exe ext mismatch: %q", tgt.ExeExt)
		}
	}
}

func TestTargetListsAreStable(t *testing.T) {
	if got := SupportedTriples(); len(got) != 5 || got[0] != "linux-x64" || got[1] != "windows-x64" || got[2] != "macos-x64" || got[3] != "wasm32-wasi" || got[4] != "wasm32-web" {
		t.Fatalf("supported triples = %#v", got)
	}
	if got := BuildOnlyTriples(); len(got) != 0 {
		t.Fatalf("build-only triples = %#v", got)
	}
	if got := PlannedTriples(); len(got) != 0 {
		t.Fatalf("planned triples = %#v", got)
	}
	if got := ActorRuntimeTriples(); len(got) != 3 || got[0] != "linux-x64" || got[1] != "macos-x64" || got[2] != "windows-x64" {
		t.Fatalf("actor runtime triples = %#v", got)
	}
}

func TestTargetStatusValues(t *testing.T) {
	cases := []struct {
		triple string
		status Status
	}{
		{"linux-x64", StatusSupported},
		{"windows-x64", StatusSupported},
		{"macos-x64", StatusSupported},
		{"wasm32-wasi", StatusSupported},
		{"wasm32-web", StatusSupported},
	}
	for _, tc := range cases {
		tgt, err := Parse(tc.triple)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tc.triple, err)
		}
		if tgt.Status != tc.status {
			t.Fatalf("Parse(%q).Status = %q, want %q", tc.triple, tgt.Status, tc.status)
		}
	}
	if StatusSupported.String() != "supported" || StatusBuildOnly.String() != "build_only" || StatusPlanned.String() != "planned" {
		t.Fatalf("unexpected status strings: %q %q %q", StatusSupported, StatusBuildOnly, StatusPlanned)
	}
	if RunModeHostNative.String() != "host_native" || RunModeWASIRunner.String() != "wasi_runner" || RunModeWebRunner.String() != "web_runner" {
		t.Fatalf("unexpected run mode strings: %q %q %q", RunModeHostNative, RunModeWASIRunner, RunModeWebRunner)
	}
}

func TestParseRejectsUnknown(t *testing.T) {
	if _, err := Parse("plan9-x64"); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestParseAcceptsWASIRuntimeTarget(t *testing.T) {
	tgt, err := Parse("wasm32-wasi")
	if err != nil {
		t.Fatalf("Parse(wasm32-wasi): %v", err)
	}
	if tgt.Triple != "wasm32-wasi" || tgt.ExeExt != ".wasm" {
		t.Fatalf("wasm32-wasi target = %#v", tgt)
	}
	if IsBuildOnlyTarget("wasm32-wasi") {
		t.Fatalf("IsBuildOnlyTarget(wasm32-wasi) = true")
	}
	if IsPlannedTarget("wasm32-wasi") {
		t.Fatalf("IsPlannedTarget(wasm32-wasi) = true")
	}
}

func TestParseAcceptsWASMWebRuntimeTarget(t *testing.T) {
	tgt, err := Parse("wasm32-web")
	if err != nil {
		t.Fatalf("Parse(wasm32-web): %v", err)
	}
	if tgt.Triple != "wasm32-web" || tgt.ExeExt != ".wasm" {
		t.Fatalf("wasm32-web target = %#v", tgt)
	}
	if IsBuildOnlyTarget("wasm32-web") {
		t.Fatalf("IsBuildOnlyTarget(wasm32-web) = true")
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

func TestCurrentTargetContractIncludesRunnableWASMTargets(t *testing.T) {
	all := All()
	if len(all) != len(SupportedTriples()) {
		t.Fatalf("All() = %#v, want supported triples %#v", all, SupportedTriples())
	}
	for _, tgt := range all {
		if tgt.Status != StatusSupported || IsBuildOnlyTarget(tgt.Triple) || IsPlannedTarget(tgt.Triple) {
			t.Fatalf("All() included non-supported target: %#v", tgt)
		}
	}
	for _, triple := range WASMTriples() {
		tgt, err := Parse(triple)
		if err != nil {
			t.Fatalf("Parse(%q): %v", triple, err)
		}
		if tgt.Status != StatusSupported || IsBuildOnlyTarget(triple) || IsPlannedTarget(triple) {
			t.Fatalf("WASM target %s contract drifted: %#v", triple, tgt)
		}
		if tgt.Arch != ArchWASM32 || tgt.Format != FormatWASM || tgt.ExeExt != ".wasm" || tgt.SupportsDebugInfo {
			t.Fatalf("WASM target %s metadata drifted: %#v", triple, tgt)
		}
		if triple == "wasm32-wasi" && (tgt.RunMode != RunModeWASIRunner || tgt.RunRunner != "wasmtime") {
			t.Fatalf("WASM target %s runner metadata drifted: %#v", triple, tgt)
		}
		if triple == "wasm32-web" && (tgt.RunMode != RunModeWebRunner || tgt.RunRunner != "") {
			t.Fatalf("WASM target %s runner metadata drifted: %#v", triple, tgt)
		}
	}
}
