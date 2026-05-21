package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/compiler/internal/ir"
	ctarget "tetra_language/compiler/target"
)

func TestAtomicIRTargetInfoUsesX32PointerWidth(t *testing.T) {
	tgt, err := ctarget.Parse("linux-x32")
	if err != nil {
		t.Fatalf("parse target: %v", err)
	}
	info, ok := atomicIRTargetInfo(ir.IRAtomicLoadPtr, tgt)
	if !ok {
		t.Fatalf("IRAtomicLoadPtr was not classified as a target atomic")
	}
	if info.widthBits != tgt.PointerWidthBits {
		t.Fatalf("x32 pointer atomic width = %d, want pointer width %d", info.widthBits, tgt.PointerWidthBits)
	}
	if info.widthBits == tgt.RegisterWidthBits {
		t.Fatalf("x32 pointer atomic width followed register width %d instead of pointer width", tgt.RegisterWidthBits)
	}
}

func TestX86RejectsI64AtomicWithTargetDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "atomic_i64_x86.tetra")
	outPath := filepath.Join(tmp, "atomic_i64_x86.tobj")
	if err := os.WriteFile(srcPath, []byte(`
func atomic_i64_probe() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let loaded: i64 = core.atomic_load_i64_acquire(p, mem)
        return 0
    return 0
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	_, err := BuildFileWithStatsOpt(srcPath, outPath, "x86", BuildOptions{Emit: EmitLibrary, Jobs: 1})
	if err == nil {
		t.Fatalf("expected x86 i64 atomic target diagnostic")
	}
	diag := DiagnosticFromError(err)
	if diag.Code != DiagnosticCodeTargetRuntime || diag.Severity != "error" {
		t.Fatalf("diagnostic identity = %#v", diag)
	}
	for _, want := range []string{
		"linux-x86",
		"atomic load",
		"64-bit",
		"unsupported atomic width 64 bits",
	} {
		if !strings.Contains(diag.Message, want) {
			t.Fatalf("message = %q, want substring %q", diag.Message, want)
		}
	}
	if !strings.Contains(diag.Hint, "Use 8/16/32-bit or pointer atomics on linux-x86") {
		t.Fatalf("hint = %q, want x86 atomic-width guidance", diag.Hint)
	}
	if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
		t.Fatalf("x86 i64 atomic rejection wrote object %s, stat error = %v", outPath, statErr)
	}
}
