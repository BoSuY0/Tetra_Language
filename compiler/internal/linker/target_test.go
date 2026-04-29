package linker

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/format/tobj"
)

func TestLinkLinuxRejectsNonLinuxTargetObject(t *testing.T) {
	_, err := LinkLinuxX64([]*tobj.Object{{
		Target:  "windows-x64",
		Module:  "wrong",
		Code:    []byte{0xC3},
		Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
	}}, "main")
	if err == nil {
		t.Fatalf("expected mismatch error")
	}
	if !strings.Contains(err.Error(), "linker target mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLinkWindowsRejectsNonWindowsTargetObject(t *testing.T) {
	_, err := LinkWindowsX64([]*tobj.Object{{
		Target:  "linux-x64",
		Module:  "wrong",
		Code:    []byte{0xC3},
		Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
	}}, "main")
	if err == nil {
		t.Fatalf("expected mismatch error")
	}
	if !strings.Contains(err.Error(), "linker target mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLinkMachORejectsNonMacTargetObject(t *testing.T) {
	_, err := LinkMacOSX64([]*tobj.Object{{
		Target:  "linux-x64",
		Module:  "wrong",
		Code:    []byte{0xC3},
		Symbols: []tobj.Symbol{{Name: "main", Offset: 0}},
	}}, "main")
	if err == nil {
		t.Fatalf("expected mismatch error")
	}
	if !strings.Contains(err.Error(), "linker target mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}
