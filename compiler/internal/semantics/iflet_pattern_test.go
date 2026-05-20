package semantics

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func TestIfLetSomePatternBindsOptionalPayload(t *testing.T) {
	checked := checkIfLetPatternSource(t, `
func unwrap(value: Int?) -> Int:
    if let some(x) = value:
        return x
    else:
        return 0

func main() -> Int:
    return unwrap(1)
`)
	if got := checked.Funcs[0].Locals["x"].TypeName; got != "i32" {
		t.Fatalf("some binding type = %q, want i32", got)
	}
}

func TestIfLetNonePatternAcceptsOptionalValue(t *testing.T) {
	checkIfLetPatternSource(t, `
func score(value: Int?) -> Int:
    if let none = value:
        return 7
    else:
        return 1

func main() -> Int:
    return score(none)
`)
}

func TestIfLetEnumPayloadPatternBindsPayloads(t *testing.T) {
	checked := checkIfLetPatternSource(t, `
enum Result:
    case ok(Int, String)
    case err(Int)

func score(value: Result) -> Int:
    if let Result.ok(code, text) = value:
        return code + text.len
    else:
        return 0

func main() -> Int:
    return score(Result.ok(1, "x"))
`)
	fn := checked.FuncSigs["score"]
	if fn.ParamSlots != 4 {
		t.Fatalf("score param slots = %d, want 4", fn.ParamSlots)
	}
	locals := checked.Funcs[0].Locals
	if got := locals["code"].TypeName; got != "i32" {
		t.Fatalf("code binding type = %q, want i32", got)
	}
	if got := locals["text"].TypeName; got != "str" {
		t.Fatalf("text binding type = %q, want str", got)
	}
}

func TestIfLetPatternRejectsNonOptionalAndNonEnumValue(t *testing.T) {
	err := checkIfLetPatternError(t, `
func main() -> Int:
    if let some(x) = 1:
        return x
    else:
        return 0
`)
	if !strings.Contains(err.Error(), "if let pattern requires optional or enum value") {
		t.Fatalf("error = %v", err)
	}
}

func TestIfLetOptionalPatternRejectsLiteralPattern(t *testing.T) {
	err := checkIfLetPatternError(t, `
func main() -> Int:
    let value: Int? = 1
    if let 1 = value:
        return 1
    else:
        return 0
`)
	if !strings.Contains(err.Error(), "optional if let supports only 'none'") {
		t.Fatalf("error = %v", err)
	}
}

func checkIfLetPatternSource(t *testing.T, src string) *CheckedProgram {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return checked
}

func checkIfLetPatternError(t *testing.T, src string) error {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected Check error")
	}
	return err
}
