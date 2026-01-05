package target

import "testing"

func TestParse(t *testing.T) {
	for _, triple := range []string{"linux-x64", "windows-x64", "macos-x64"} {
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

func TestParseRejectsUnknown(t *testing.T) {
	if _, err := Parse("plan9-x64"); err == nil {
		t.Fatalf("expected error, got nil")
	}
}
