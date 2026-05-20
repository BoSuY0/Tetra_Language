package semantics

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
)

func TestMatchExprInferenceBindsOptionalSomePayload(t *testing.T) {
	checked := checkMatchExprInferenceSource(t, `
func main() -> Int:
    let value: String? = "abcd"
    let score = match value:
    case some(text):
        text.len
    case none:
        0
    return score
`)
	if got := checked.Funcs[0].Locals["score"].TypeName; got != "i32" {
		t.Fatalf("score type = %q, want i32", got)
	}
	if got := checked.Funcs[0].Locals["text"].TypeName; got != "str" {
		t.Fatalf("text binding type = %q, want str", got)
	}
}

func TestMatchExprInferenceBindsEnumPayloads(t *testing.T) {
	checked := checkMatchExprInferenceSource(t, `
enum Result:
    case ok(Int, String)
    case err(Int)

func main() -> Int:
    let result: Result = Result.ok(40, "xy")
    let score = match result:
    case Result.ok(code, text):
        code + text.len
    case Result.err(errCode):
        errCode
    return score
`)
	locals := checked.Funcs[0].Locals
	if got := locals["score"].TypeName; got != "i32" {
		t.Fatalf("score type = %q, want i32", got)
	}
	if got := locals["code"].TypeName; got != "i32" {
		t.Fatalf("code binding type = %q, want i32", got)
	}
	if got := locals["errCode"].TypeName; got != "i32" {
		t.Fatalf("errCode binding type = %q, want i32", got)
	}
	if got := locals["text"].TypeName; got != "str" {
		t.Fatalf("text binding type = %q, want str", got)
	}
}

func TestMatchExprInferenceRejectsCaseTypeMismatch(t *testing.T) {
	err := checkMatchExprInferenceError(t, `
func main() -> Int:
    let value: Int? = 1
    let score = match value:
    case some(x):
        x
    case none:
        "bad"
    return score
`)
	if !strings.Contains(err.Error(), "match expression case type mismatch") {
		t.Fatalf("error = %v", err)
	}
}

func checkMatchExprInferenceSource(t *testing.T, src string) *CheckedProgram {
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

func checkMatchExprInferenceError(t *testing.T, src string) error {
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
