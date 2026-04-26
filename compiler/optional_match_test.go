package compiler

import (
	"runtime"
	"strings"
	"testing"
)

func TestOptionalMatchNoneCheckAndLower(t *testing.T) {
	src := []byte(`
func maybe(flag: Bool) -> Int?:
    if flag:
        return 42
    else:
        return none

func main() -> Int:
    let value: Int? = maybe(false)
    match value:
    case none:
        return 42
    case _:
        return 1
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

func TestOptionalMatchRejectsNonNonePattern(t *testing.T) {
	src := []byte(`
func main() -> Int:
    let value: Int? = 1
    match value:
    case 1:
        return 1
    case _:
        return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected optional match pattern error")
	}
	if !strings.Contains(err.Error(), "optional match supports only 'none'") {
		t.Fatalf("error = %v", err)
	}
}

func TestOptionalMatchSomeBindingCheckAndLower(t *testing.T) {
	src := []byte(`
func maybe(flag: Bool) -> Int?:
    if flag:
        return 42
    else:
        return none

func main() -> Int:
    let value: Int? = maybe(true)
    match value:
    case some(x):
        return x
    case none:
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
	if got := checked.Funcs[1].Locals["x"].TypeName; got != "i32" {
		t.Fatalf("some binding type = %q, want i32", got)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestBuildOptionalMatchNoneSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func maybe(flag: Bool) -> Int?:
    if flag:
        return 7
    else:
        return none

func main() -> Int:
    let value: Int? = maybe(false)
    match value:
    case none:
        return 42
    case _:
        return 1
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildOptionalMatchSomeSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func maybe(flag: Bool) -> Int?:
    if flag:
        return 42
    else:
        return none

func main() -> Int:
    let value: Int? = maybe(true)
    match value:
    case some(x):
        return x
    case none:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}
