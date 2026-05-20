package compiler_test

import (
	"strings"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/testkit"
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
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if checked.FuncSigs["maybe"].ReturnSlots != 2 {
		t.Fatalf("maybe return slots = %d, want 2", checked.FuncSigs["maybe"].ReturnSlots)
	}
	if _, err := compiler.Lower(checked); err != nil {
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
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if got := checked.Funcs[0].Locals["x"].TypeName; got != "i32" {
		t.Fatalf("if-let local type = %q, want i32", got)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("lower: %v", err)
	}
}

func TestOptionalIfLetSomePatternCheckAndLower(t *testing.T) {
	src := []byte(`
func unwrap(value: Int?) -> Int:
    if let some(x) = value:
        return x
    else:
        return 0

func main() -> Int:
    return unwrap(41)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if got := checked.Funcs[0].Locals["x"].TypeName; got != "i32" {
		t.Fatalf("some binding type = %q, want i32", got)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("lower: %v", err)
	}
}

func TestOptionalIfLetNonePatternCheckAndLower(t *testing.T) {
	src := []byte(`
func score(value: Int?) -> Int:
    if let none = value:
        return 7
    else:
        return 1

func main() -> Int:
    return score(none)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("lower: %v", err)
	}
}

func TestOptionalIfLetPatternRejectsNonOptionalValue(t *testing.T) {
	src := []byte(`
func main() -> Int:
    if let some(x) = 1:
        return x
    else:
        return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected if-let pattern type error")
	}
	if !strings.Contains(err.Error(), "if let pattern requires optional or enum value") {
		t.Fatalf("error = %v", err)
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
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("lower: %v", err)
	}
}

func TestOptionalSmallIntLiteralPayloadsCheckAndLower(t *testing.T) {
	testkit.RequireFileCheckOK(t, `
func main() -> Int:
    let byte: UInt8? = 255
    let word: UInt16? = 65535
    var assigned_byte: UInt8? = none
    assigned_byte = 255
    var assigned_word: UInt16? = none
    assigned_word = 65535
    return 0
`)
}

func TestOptionalSmallIntLiteralPayloadsRejectOutOfRange(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "u8 local initializer",
			src: `
func main() -> Int:
    let maybe: UInt8? = 300
    return 0
`,
			want: "type mismatch: expected 'u8?', got 'i32'",
		},
		{
			name: "u16 local initializer",
			src: `
func main() -> Int:
    let maybe: UInt16? = 70000
    return 0
`,
			want: "type mismatch: expected 'u16?', got 'i32'",
		},
		{
			name: "u8 assignment",
			src: `
func main() -> Int:
    var maybe: UInt8? = none
    maybe = 300
    return 0
`,
			want: "type mismatch: expected 'u8?', got 'i32'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.RequireCheckErrorContains(t, tt.src, tt.want)
		})
	}
}

func TestNestedOptionalLiteralPayloadsCheckAndLower(t *testing.T) {
	testkit.RequireFileCheckOK(t, `
func main() -> Int:
    let nested: Int?? = 42
    var assigned: Int?? = none
    assigned = 42
    let inner: Int? = 42
    let from_inner: Int?? = inner
    return 0
`)
}

func TestNestedOptionalReturnPayloadCheckAndLower(t *testing.T) {
	testkit.RequireFileCheckOK(t, `
func make_nested() -> Int??:
    return 42

func main() -> Int:
    let nested: Int?? = make_nested()
    return 0
`)
}

func TestNestedOptionalSmallIntLiteralPayloadsRejectOutOfRange(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "nested u8 initializer",
			src: `
func main() -> Int:
    let maybe: UInt8?? = 300
    return 0
`,
			want: "type mismatch: expected 'u8??', got 'i32'",
		},
		{
			name: "nested u16 assignment",
			src: `
func main() -> Int:
    var maybe: UInt16?? = none
    maybe = 70000
    return 0
`,
			want: "type mismatch: expected 'u16??', got 'i32'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.RequireCheckErrorContains(t, tt.src, tt.want)
		})
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
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check(prog)
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
	if _, err := compiler.Lower(checked); err != nil {
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
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if got := checked.Funcs[1].Locals["s"].TypeName; got != "str" {
		t.Fatalf("some binding type = %q, want str", got)
	}
	if _, err := compiler.Lower(checked); err != nil {
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
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, err = compiler.Check(prog)
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
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check(prog)
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
	if _, err := compiler.Lower(checked); err != nil {
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
			err := testkit.CheckProgram(tt.src)
			if err == nil {
				t.Fatalf("expected narrowing binding scope error")
			}
			if !strings.Contains(err.Error(), "out of scope") && !strings.Contains(err.Error(), "unknown identifier") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
