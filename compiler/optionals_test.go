package compiler

import (
	"strings"
	"testing"
)

func TestOptionalNoneEqualityLowers(t *testing.T) {
	src := []byte(`
func maybe() -> Int?:
    return none

func main() -> Int:
    let value: Int? = maybe()
    if value == none:
        return 0
    else:
        return 1
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if checked.FuncSigs["maybe"].ReturnSlots != 2 {
		t.Fatalf("maybe return slots = %d, want 2", checked.FuncSigs["maybe"].ReturnSlots)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("lower: %v", err)
	}
}

func TestOptionalIfLetLowers(t *testing.T) {
	src := []byte(`
func unwrap(value: Int?) -> Int:
    if let x = value:
        return x
    else:
        return 0

func main() -> Int:
    return unwrap(none)
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if got := checked.Funcs[0].Locals["x"].TypeName; got != "i32" {
		t.Fatalf("if-let local type = %q, want i32", got)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("lower: %v", err)
	}
}

func TestOptionalImplicitSomeReturnAndLetLower(t *testing.T) {
	src := []byte(`
func maybe() -> Int?:
    return 42

func main() -> Int:
    let value: Int? = 7
    if value != none:
        return 0
    else:
        return 1
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("lower: %v", err)
	}
}

func TestOptionalAllowsMultiSlotPayload(t *testing.T) {
	src := []byte(`
func maybe(flag: Bool) -> String?:
    if flag:
        return "ok"
    else:
        return none

func length(value: String?) -> Int:
    if let s = value:
        return s.len
    else:
        return 0

func main() -> Int:
    return length(maybe(true))
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if got := checked.FuncSigs["maybe"].ReturnSlots; got != 3 {
		t.Fatalf("maybe return slots = %d, want 3", got)
	}
	if got := checked.FuncSigs["length"].ParamSlots; got != 3 {
		t.Fatalf("length param slots = %d, want 3", got)
	}
	if got := checked.Funcs[1].Locals["s"].TypeName; got != "str" {
		t.Fatalf("if-let local type = %q, want str", got)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("lower: %v", err)
	}
}

func TestOptionalMatchExhaustiveNoDefaultWithMultiSlotPayload(t *testing.T) {
	src := []byte(`
func maybe(flag: Bool) -> String?:
    if flag:
        return "ok"
    else:
        return none

func main() -> Int:
    let value: String? = maybe(true)
    match value:
    case some(s):
        return s.len
    case none:
        return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if got := checked.Funcs[1].Locals["s"].TypeName; got != "str" {
		t.Fatalf("some binding type = %q, want str", got)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("lower: %v", err)
	}
}

func TestOptionalMatchMissingSomeCaseNeedsReturn(t *testing.T) {
	src := []byte(`
func maybe(flag: Bool) -> String?:
    if flag:
        return "ok"
    else:
        return none

func main() -> Int:
    let value: String? = maybe(true)
    match value:
    case none:
        return 0
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, err = Check(prog)
	if err == nil {
		t.Fatalf("expected non-exhaustive optional match error")
	}
	if !strings.Contains(err.Error(), "must end with return") {
		t.Fatalf("error = %v", err)
	}
}

func TestOptionalStructPayloadIfLetAndMatchLower(t *testing.T) {
	src := []byte(`
struct Pair:
    x: Int
    y: Int

func maybe(flag: Bool) -> Pair?:
    if flag:
        return Pair(x: 20, y: 22)
    else:
        return none

func unwrap_if(value: Pair?) -> Int:
    if let p = value:
        return p.x + p.y
    else:
        return 0

func unwrap_match(value: Pair?) -> Int:
    match value:
    case some(p):
        return p.x + p.y
    case none:
        return 0

func main() -> Int:
    return unwrap_if(maybe(true)) + unwrap_match(maybe(false))
`)
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if got := checked.FuncSigs["maybe"].ReturnSlots; got != 3 {
		t.Fatalf("maybe return slots = %d, want 3", got)
	}
	if got := checked.Funcs[1].Locals["p"].TypeName; got != "Pair" {
		t.Fatalf("if-let payload type = %q, want Pair", got)
	}
	if got := checked.Funcs[2].Locals["p"].TypeName; got != "Pair" {
		t.Fatalf("match payload type = %q, want Pair", got)
	}
	if _, err := Lower(checked); err != nil {
		t.Fatalf("lower: %v", err)
	}
}

func TestOptionalNarrowingBindingsDoNotEscapeCaseScope(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "if let",
			src: `
func main() -> Int:
    let value: Int? = 1
    if let x = value:
        let y: Int = x
    return x
`,
		},
		{
			name: "match some",
			src: `
func main() -> Int:
    let value: Int? = 1
    match value:
    case some(x):
        let y: Int = x
    case none:
        let z: Int = 0
    return x
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkProgram(tt.src)
			if err == nil {
				t.Fatalf("expected narrowing binding scope error")
			}
			if !strings.Contains(err.Error(), "out of scope") && !strings.Contains(err.Error(), "unknown identifier") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
