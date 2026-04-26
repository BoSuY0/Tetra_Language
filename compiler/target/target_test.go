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
	if got := PlannedTriples(); len(got) != 2 || got[0] != "wasm32-wasi" || got[1] != "wasm32-web" {
		t.Fatalf("planned triples = %#v", got)
	}
}

func TestParseRejectsUnknown(t *testing.T) {
	if _, err := Parse("plan9-x64"); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestParseReportsPlannedWASMTargets(t *testing.T) {
	for _, triple := range []string{"wasm32-wasi", "wasm32-web"} {
		_, err := Parse(triple)
		if err == nil {
			t.Fatalf("Parse(%q): expected planned-target error", triple)
		}
		targetErr, ok := err.(UnsupportedTargetError)
		if !ok {
			t.Fatalf("Parse(%q): error type = %T, want UnsupportedTargetError", triple, err)
		}
		if !targetErr.Planned || targetErr.Triple != triple {
			t.Fatalf("planned target error = %#v", targetErr)
		}
		if !IsPlannedTarget(triple) {
			t.Fatalf("IsPlannedTarget(%q) = false", triple)
		}
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
