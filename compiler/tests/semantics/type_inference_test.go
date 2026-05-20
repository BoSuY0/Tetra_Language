package compiler_test

import (
	"strings"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/testkit"
)

func TestLocalTypeInference(t *testing.T) {
	src := []byte(`
fun main(): i32 {
  let x = 40
  let y: i32 = 2
  return x + y
}
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, err := compiler.Check(prog); err != nil {
		t.Fatalf("check: %v", err)
	}
}

func TestFlowLetIsImmutable(t *testing.T) {
	src := []byte(`
func main() -> i32:
  let x = 1
  x = 2
  return x
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, err := compiler.Check(prog); err == nil {
		t.Fatalf("expected immutable Flow let assignment to fail")
	}
}

func TestV1CanonicalTypeNamesAndStructuralSlots(t *testing.T) {
	src := []byte(`
struct Packet:
    id: Int
    payload: String
    owned: island

func main() -> Int:
    let ok: Bool = true
    let byte: Byte = 7
    let text: String = "ok"
    return byte
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	main := checked.Funcs[0]
	if got := main.Locals["ok"].TypeName; got != "bool" {
		t.Fatalf("Bool alias resolved to %q, want bool", got)
	}
	if got := main.Locals["byte"].TypeName; got != "u8" {
		t.Fatalf("Byte alias resolved to %q, want u8", got)
	}
	if got := main.Locals["text"].TypeName; got != "str" {
		t.Fatalf("String alias resolved to %q, want str", got)
	}
	packet := checked.Types["Packet"]
	if packet == nil {
		t.Fatalf("missing Packet type")
	}
	if got := packet.SlotCount; got != 4 {
		t.Fatalf("Packet slots = %d, want 4", got)
	}
	if got := packet.FieldMap["payload"].TypeName; got != "str" {
		t.Fatalf("payload type = %q, want str", got)
	}
}

func TestV1StructConstructorsRejectInvalidFields(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "missing field",
			src: `
struct Pair:
    x: Int
    y: Int

func main() -> Int:
    let p: Pair = Pair(x: 1)
    return 0
`,
			want: "missing field 'y'",
		},
		{
			name: "unknown field",
			src: `
struct Pair:
    x: Int

func main() -> Int:
    let p: Pair = Pair(y: 1)
    return 0
`,
			want: "unknown field 'y'",
		},
		{
			name: "duplicate field",
			src: `
struct Pair:
    x: Int
    y: Int

func main() -> Int:
    let p: Pair = Pair(x: 1, x: 2)
    return 0
`,
			want: "duplicate field 'x'",
		},
		{
			name: "type mismatch",
			src: `
struct Pair:
    x: Int

func main() -> Int:
    let p: Pair = Pair(x: true)
    return 0
`,
			want: "type mismatch for field 'x'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := testkit.CheckProgram(tt.src); err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.want)
			} else if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestV1InferenceRequiresAnnotationForNoneAndUsesExpectedOptionals(t *testing.T) {
	err := testkit.CheckProgram(`
func main() -> Int:
    let value = none
    return 0
`)
	if err == nil {
		t.Fatalf("expected none inference error")
	}
	if !strings.Contains(err.Error(), "cannot infer type from 'none'") {
		t.Fatalf("error = %v", err)
	}

	if err := testkit.CheckProgram(`
func consume(value: Int?) -> Int:
    if value == none:
        return 0
    return 1

func main() -> Int:
    let value: Int? = none
    return consume(value)
`); err != nil {
		t.Fatalf("expected annotated optional none to check: %v", err)
	}
}

func TestV1APIDocsUseCanonicalBuiltinTypeNames(t *testing.T) {
	src := []byte(`
const answer: Int = 42

func audit(token: ConsentToken, secret: SecretInt, text: String, byte: Byte) -> Bool:
    return true
`)
	docs, err := compiler.GenerateAPIDocsFromSource(src, "types.tetra")
	if err != nil {
		t.Fatalf("GenerateAPIDocsFromSource: %v", err)
	}
	out := string(docs)
	for _, want := range []string{
		"`const answer: i32`",
		"`func audit(token: consent.token, secret: secret.i32, text: str, byte: u8) -> bool`",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("docs missing %q:\n%s", want, out)
		}
	}
}

func TestV1OpaqueHandleTypesAreNotInterchangeable(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "island to ptr",
			src: `
func main() -> Int
uses alloc, capability, islands, mem:
    island(64) as isl:
        let p: ptr = isl
    return 0
`,
			want: "type mismatch",
		},
		{
			name: "ptr to island",
			src: `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        let isl: island = p
        return 0
    return 0
`,
			want: "type mismatch",
		},
		{
			name: "capability families",
			src: `
func main() -> Int
uses capability, io:
    unsafe:
        let io: cap.io = core.cap_io()
        let mem: cap.mem = io
        return 0
    return 0
`,
			want: "type mismatch",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := testkit.CheckProgram(tt.src); err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.want)
			} else if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}
