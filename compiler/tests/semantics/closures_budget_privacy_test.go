package compiler_test

import (
	"strings"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/ir"
)

func TestCapturedFunctionTypedReturnNestedStructReassignmentCanSnapshotIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let base: Int = 1
    return fn(x: Int) -> Int:
        return x + base

func main() -> Int:
    var box: Box = Box(holder: Holder(cb: add0))
    box = Box(holder: Holder(cb: make()))
    cb = box.holder.cb
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_return_nested_struct_reassign_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestCapturedFunctionTypedReturnNestedStructFieldReassignmentCanSnapshotIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let base: Int = 1
    return fn(x: Int) -> Int:
        return x + base

func main() -> Int:
    var box: Box = Box(holder: Holder(cb: add0))
    box.holder.cb = make()
    cb = box.holder.cb
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_return_nested_struct_field_reassign_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestClosureCaptureDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "mutable local",
			src: `
func main() -> Int:
    var y: Int = 1
    let f: ptr = fn(x: Int) -> Int:
        return x + y
    return f(41)
`,
			want: "closure capture 'y' is mutable; direct ptr closure calls would observe mutable locals by reference, so use a function-typed fnptr binding for by-value snapshot capture",
		},
		{
			name: "struct with ptr field",
			src: `
struct PtrBox:
    p: ptr

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    let f: ptr = fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    return f(41)
`,
			want: "closure capture 'box' has unsupported type 'PtrBox'; only immutable local Int/Bool/String, simple struct, enum, and optional captures without ptr/resource fields are supported by the direct ptr-closure capture ABI",
		},
		{
			name: "struct with resource field",
			src: `
struct ActorBox:
    peer: actor

func use(box: ActorBox) -> Int:
    let f: ptr = fn(x: Int) -> Int:
        let p: actor = box.peer
        let _: actor = p
        return x
    return f(1)

func main() -> Int:
    return 0
`,
			want: "closure capture 'box' has unsupported type 'ActorBox'; only immutable local Int/Bool/String, simple struct, enum, and optional captures without ptr/resource fields are supported by the direct ptr-closure capture ABI",
		},
		{
			name: "escaping pointer value",
			src: `
func choose(p: ptr) -> Int:
    return 0

func main() -> Int:
    let y: Int = 1
    let f: ptr = fn(x: Int) -> Int:
        return x + y
    return choose(f)
`,
			want: "capturing closure 'f' cannot escape as raw ptr",
		},
		{
			name: "direct non let bound",
			src: `
func choose(p: ptr) -> Int:
    return 0

func main() -> Int:
    let y: Int = 1
    return choose(fn(x: Int) -> Int:
        return x + y
    )
`,
			want: "capturing closure literal captures 'y' but is not let-bound; only let-bound local direct calls can capture immutable Int/Bool/String values and simple structs without ptr/resource fields under the direct ptr-closure ABI",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := compiler.Parse([]byte(tt.src))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			_, err = compiler.Check(prog)
			if err == nil {
				t.Fatalf("expected capture diagnostic")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
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
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := compiler.Lower(checked); err != nil {
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
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected nothrow/throws conflict")
	}
	if !strings.Contains(err.Error(), "nothrow") {
		t.Fatalf("error = %v", err)
	}
}

func findIRFuncByName(prog *compiler.IRProgram, name string) *compiler.IRFunc {
	for i := range prog.Funcs {
		if prog.Funcs[i].Name == name {
			return &prog.Funcs[i]
		}
	}
	return nil
}

func hasInstrKind(fn *compiler.IRFunc, kind ir.IRInstrKind) bool {
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
uses budget
budget(1):
    return 1

func work() -> Int
uses budget
budget(2):
    return tick()

func main() -> Int
uses budget
budget(4):
    return work()
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := compiler.Lower(checked)
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

func TestBudgetFailureABIReturnAndThrowShapesAreLowered(t *testing.T) {
	src := []byte(`
struct Pair:
    x: Int
    y: Int

enum CompactTrap:
    case exhausted
    case other

enum WideTrap:
    case exhausted(Int)
    case other(Int)

func pair() -> Pair
uses budget
budget(0):
    return Pair(x: 7, y: 8)

func compact() -> Int throws CompactTrap
uses budget
budget(0):
    return 9

func wide() -> Int throws WideTrap
uses budget
budget(0):
    return 9

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}

	pair := findIRFuncByName(irProg, "pair")
	if pair == nil {
		t.Fatalf("missing lowered function 'pair'")
	}
	assertBudgetFailureTail(t, pair, []ir.IRInstrKind{
		ir.IRLabel,
		ir.IRConstI32,
		ir.IRConstI32,
		ir.IRReturn,
	})
	assertBudgetFailureTailImms(t, pair, []int32{0, 0})

	compact := findIRFuncByName(irProg, "compact")
	if compact == nil {
		t.Fatalf("missing lowered function 'compact'")
	}
	assertBudgetFailureTail(t, compact, []ir.IRInstrKind{
		ir.IRLabel,
		ir.IRConstI32,
		ir.IRConstI32,
		ir.IRReturn,
	})
	assertBudgetFailureTailImms(t, compact, []int32{0, 1})

	wide := findIRFuncByName(irProg, "wide")
	if wide == nil {
		t.Fatalf("missing lowered function 'wide'")
	}
	assertBudgetFailureTail(t, wide, []ir.IRInstrKind{
		ir.IRLabel,
		ir.IRConstI32,
		ir.IRConstI32,
		ir.IRConstI32,
		ir.IRConstI32,
		ir.IRReturn,
	})
	assertBudgetFailureTailImms(t, wide, []int32{0, 0, 0, 1})
}

func assertBudgetFailureTail(t *testing.T, fn *compiler.IRFunc, want []ir.IRInstrKind) {
	t.Helper()
	if fn.Policy.FailLabel < 0 {
		t.Fatalf("%s missing policy failure label", fn.Name)
	}
	start := -1
	for i, instr := range fn.Instrs {
		if instr.Kind == ir.IRLabel && instr.Label == fn.Policy.FailLabel {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("%s missing policy failure label %d: %#v", fn.Name, fn.Policy.FailLabel, fn.Instrs)
	}
	got := fn.Instrs[start:]
	if len(got) != len(want) {
		t.Fatalf("%s budget failure tail length = %d, want %d: %#v", fn.Name, len(got), len(want), got)
	}
	for i, kind := range want {
		if got[i].Kind != kind {
			t.Fatalf("%s budget failure tail[%d] = %v, want %v: %#v", fn.Name, i, got[i].Kind, kind, got)
		}
	}
}

func assertBudgetFailureTailImms(t *testing.T, fn *compiler.IRFunc, want []int32) {
	t.Helper()
	start := -1
	for i, instr := range fn.Instrs {
		if instr.Kind == ir.IRLabel && instr.Label == fn.Policy.FailLabel {
			start = i
			break
		}
	}
	if start < 0 {
		t.Fatalf("%s missing policy failure label %d", fn.Name, fn.Policy.FailLabel)
	}
	var got []int32
	for _, instr := range fn.Instrs[start:] {
		if instr.Kind == ir.IRConstI32 {
			got = append(got, instr.Imm)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("%s budget failure const count = %d, want %d: got %v", fn.Name, len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s budget failure const[%d] = %d, want %d: got %v", fn.Name, i, got[i], want[i], got)
		}
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
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	irProg, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	seal := findIRFuncByName(irProg, "seal")
	if seal == nil {
		t.Fatalf("missing lowered function 'seal'")
	}
	if !hasInstrKind(seal, ir.IRCmpEqI32) || !hasInstrKind(seal, ir.IRJmpIfZero) {
		t.Fatalf("seal missing consent guard instructions: %#v", seal.Instrs)
	}
}
