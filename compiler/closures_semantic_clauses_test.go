package compiler

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/ir"
)

func TestClosureLiteralParseCheckAndLower(t *testing.T) {
	src := []byte(`
func main() -> Int:
    let f: ptr = fn(x: Int) -> Int:
        return x
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestSemanticClausesParseCheckAndLower(t *testing.T) {
	src := []byte(`
func main() -> Int
uses budget
noalloc
noblock
realtime
nothrow
budget(10):
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestSemanticClauseNothrowRejectsThrows(t *testing.T) {
	src := []byte(`
enum E:
    case bad

func main() -> Int throws E nothrow:
    throw E.bad
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected nothrow/throws conflict")
	}
	if !strings.Contains(err.Error(), "nothrow") {
		t.Fatalf("error = %v", err)
	}
}

func findIRFuncByName(prog *IRProgram, name string) *IRFunc {
	for i := range prog.Funcs {
		if prog.Funcs[i].Name == name {
			return &prog.Funcs[i]
		}
	}
	return nil
}

func hasInstrKind(fn *IRFunc, kind ir.IRInstrKind) bool {
	for _, instr := range fn.Instrs {
		if instr.Kind == kind {
			return true
		}
	}
	return false
}

func TestBudgetRuntimeChecksAreLowered(t *testing.T) {
	src := []byte(`
func tick() -> Int
uses budget:
    return 1

func work() -> Int
uses budget
budget(2):
    return tick()

func main() -> Int
uses budget:
    return work()
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	work := findIRFuncByName(irProg, "work")
	if work == nil {
		t.Fatalf("missing lowered function 'work'")
	}
	if !hasInstrKind(work, ir.IRSubI32) || !hasInstrKind(work, ir.IRJmpIfZero) {
		t.Fatalf("work missing budget guard instructions: %#v", work.Instrs)
	}
}

func TestPrivacyConsentRuntimeChecksAreLowered(t *testing.T) {
	src := []byte(`
func seal(token: consent.token) -> secret.i32
uses privacy
privacy
consent(token):
    return core.secret_seal_i32(1, token)

func main() -> Int:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	seal := findIRFuncByName(irProg, "seal")
	if seal == nil {
		t.Fatalf("missing lowered function 'seal'")
	}
	if !hasInstrKind(seal, ir.IRCmpNeI32) || !hasInstrKind(seal, ir.IRJmpIfZero) {
		t.Fatalf("seal missing consent guard instructions: %#v", seal.Instrs)
	}
}
