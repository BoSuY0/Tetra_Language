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

func TestGenericClosureLiteralParseCheckAndLower(t *testing.T) {
	src := []byte(`
func main() -> Int:
    let f: ptr = fn<T>(x: T) -> T:
        return x
    let v: Int = f(7)
    return v
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

func TestGenericClosureLiteralInferenceFromCallArgs(t *testing.T) {
	src := []byte(`
func main() -> Int:
    let pick: ptr = fn<T>(x: T, y: T) -> T:
        return x
    let ok: Bool = pick(true, false)
    if ok:
        return 1
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

func TestGenericClosureLiteralUnsupportedDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "escape as pointer argument",
			src: `
func use(p: ptr) -> Int:
    return 0

func main() -> Int:
    let id: ptr = fn<T>(x: T) -> T:
        return x
    return use(id)
`,
			want: "generic closure 'id' cannot be used as a pointer value",
		},
		{
			name: "mutable binding direct call unsupported",
			src: `
func main() -> Int:
    var id: ptr = fn<T>(x: T) -> T:
        return x
    return id(1)
`,
			want: "generic closure 'id' is only supported for let-bound direct local calls with inferable concrete arguments in this MVP",
		},
		{
			name: "captures unsupported",
			src: `
func main() -> Int:
    let base: Int = 41
    let id: ptr = fn<T>(x: T) -> T:
        let _: Int = base
        return x
    let _: Int = id(1)
    return 0
`,
			want: "generic closure literals do not support captures in this MVP",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := Parse([]byte(tt.src))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			_, err = Check(prog)
			if err == nil {
				t.Fatalf("expected diagnostic")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestClosureFunctionValueInvocationParseCheckAndLower(t *testing.T) {
	src := []byte(`
func main() -> Int:
    let f: ptr = fn(x: Int) -> Int:
        return x
    return f(42)
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

func TestFunctionTypedClosureValueParseCheckAndLower(t *testing.T) {
	src := []byte(`
func main() -> Int:
    let add1: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + 1
    return add1(41)
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

func TestFunctionTypedNamedClosureSymbolParseCheckAndLower(t *testing.T) {
	src := []byte(`
closure add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let f: fn(Int) -> Int = add1
    return f(41)
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

func TestFunctionTypedNamedFunctionSymbolParseCheckAndLower(t *testing.T) {
	src := []byte(`
func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let f: fn(Int) -> Int = add1
    return f(41)
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

func TestFunctionTypedCallableParamParseCheckAndLower(t *testing.T) {
	src := []byte(`
func add1(x: Int) -> Int:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let f: fn(Int) -> Int = add1
    return apply(f, 41)
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

func TestFunctionTypedDirectNamedCallableParamParseCheckAndLower(t *testing.T) {
	src := []byte(`
func add1(x: Int) -> Int:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(add1, 41)
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

func TestFunctionTypedReturnSymbolBackedValueParseCheckAndLower(t *testing.T) {
	src := []byte(`
func add1(x: Int) -> Int:
    return x + 1

func pick() -> fn(Int) -> Int:
    let f: fn(Int) -> Int = add1
    let g: fn(Int) -> Int = f
    return g

func main() -> Int:
    let cb: fn(Int) -> Int = pick()
    return cb(41)
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

func TestFunctionTypedClosureValueUnsupportedDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "literal declared effects must cover closure effects",
			src: `
func main() -> Int:
    let f: fn(Int) -> Int = fn(x: Int) -> Int
    uses io:
        print("x\n")
        return x
    return f(41)
`,
			want: "function-typed local 'f' requires effects io but function type does not declare them",
		},
		{
			name: "literal declared effects propagate to local callback call",
			src: `
func main() -> Int:
    let f: fn(Int) -> Int uses io = fn(x: Int) -> Int
    uses io:
        print("x\n")
        return x
    return f(41)
`,
			want: "function 'main' uses effect 'io' but does not declare it",
		},
		{
			name: "return generic named symbol unsupported",
			src: `
func id<T>(x: T) -> T:
    return x

func pick() -> fn(Int) -> Int:
    let f: fn(Int) -> Int = id
    return f
`,
			want: "generic function symbol",
		},
		{
			name: "return throwing named symbol unsupported",
			src: `
enum E:
    case fail

func add1(x: Int) -> Int throws E:
    return x + 1

func pick() -> fn(Int) -> Int:
    let f: fn(Int) -> Int = add1
    return f
`,
			want: "throwing function symbol",
		},
		{
			name: "return capturing closure unsupported",
			src: `
func pick() -> fn(Int) -> Int:
    let base: Int = 1
    let f: ptr = fn(x: Int) -> Int:
        return x + base
    return f
`,
			want: "capturing closure",
		},
		{
			name: "return symbol signature mismatch",
			src: `
func as_bool(flag: Bool) -> Int:
    if flag:
        return 1
    return 0

func pick() -> fn(Int) -> Int:
    let f: fn(Bool) -> Int = as_bool
    return f
`,
			want: "returned function symbol",
		},
		{
			name: "capture unsupported",
			src: `
func main() -> Int:
    let base: Int = 1
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    return f(41)
`,
			want: "captures 'base'; captures are not supported for function-typed values in this MVP",
		},
		{
			name: "escape unsupported",
			src: `
func consume(p: ptr) -> Int:
    return 0

func main() -> Int:
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x
    return consume(f)
`,
			want: "cannot escape as a first-class value in this MVP",
		},
		{
			name: "signature mismatch",
			src: `
func main() -> Int:
    let f: fn(Bool) -> Int = fn(x: Int) -> Int:
        return x
    return f(true)
`,
			want: "parameter 1 type mismatch",
		},
		{
			name: "signature mismatch from named symbol",
			src: `
func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let f: fn(Bool) -> Int = add1
    return f(true)
`,
			want: "parameter 1 type mismatch",
		},
		{
			name: "unsupported symbol kind local value",
			src: `
func main() -> Int:
    let add1: Int = 1
    let f: fn(Int) -> Int = add1
    return 0
`,
			want: "must be initialized with an immutable symbol-backed function value or direct named function/closure symbol in this MVP",
		},
		{
			name: "generic named symbol unsupported",
			src: `
func id<T>(x: T) -> T:
    return x

func main() -> Int:
    let f: fn(Int) -> Int = id
    return f(41)
`,
			want: "generic function symbol",
		},
		{
			name: "generic callback symbol unsupported in callable parameter path",
			src: `
func id<T>(x: T) -> T:
    return x

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let f: fn(Int) -> Int = id
    return apply(f, 41)
`,
			want: "generic function symbol",
		},
		{
			name: "generic direct callback symbol unsupported in callable parameter path",
			src: `
func id<T>(x: T) -> T:
    return x

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(id, 41)
`,
			want: "generic function symbol",
		},
		{
			name: "throwing named symbol unsupported",
			src: `
enum E:
    case fail

func add1(x: Int) -> Int throws E:
    return x + 1

func main() -> Int:
    let f: fn(Int) -> Int = add1
    return f(41)
`,
			want: "throwing function symbol",
		},
		{
			name: "throwing callback symbol unsupported in callable parameter path",
			src: `
enum E:
    case fail

func add1(x: Int) -> Int throws E:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let f: fn(Int) -> Int = add1
    return apply(f, 41)
`,
			want: "throwing function symbol",
		},
		{
			name: "throwing direct callback symbol unsupported in callable parameter path",
			src: `
enum E:
    case fail

func add1(x: Int) -> Int throws E:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(add1, 41)
`,
			want: "throwing function symbol",
		},
		{
			name: "capturing function-typed callback argument rejected",
			src: `
func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let base: Int = 1
    let f: ptr = fn(x: Int) -> Int:
        return x + base
    return apply(f, 41)
`,
			want: "callback argument must be a symbol-backed local function value or direct named function/closure symbol in this MVP",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := Parse([]byte(tt.src))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			_, err = Check(prog)
			if err == nil {
				t.Fatalf("expected diagnostic")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestClosureImmutableScalarCaptureParseCheckAndLower(t *testing.T) {
	src := []byte(`
func main() -> Int:
    let y: Int = 1
    let flag: Bool = true
    let f: ptr = fn(x: Int) -> Int:
        if flag:
            return x + y
        return x + y
    return f(41)
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

func TestClosureImmutableStringCaptureParseCheckAndLower(t *testing.T) {
	src := []byte(`
func main() -> Int:
    let text: String = "A"
    let f: ptr = fn(x: Int) -> Int:
        return x + text[0]
    return f(41)
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

func TestClosureImmutableStructCaptureParseCheckAndLower(t *testing.T) {
	src := []byte(`
struct Pair:
    left: Int
    right: Bool

func main() -> Int:
    let pair: Pair = Pair(left: 1, right: true)
    let f: ptr = fn(x: Int) -> Int:
        if pair.right:
            return x + pair.left
        return x
    return f(41)
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
			want: "closure capture 'y' is mutable; only immutable local Int/Bool/String and simple struct captures without ptr/resource fields are supported in this MVP",
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
			want: "closure capture 'box' has unsupported type 'PtrBox'; only immutable local Int/Bool/String and simple struct captures without ptr/resource fields are supported in this MVP",
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
			want: "closure capture 'box' has unsupported type 'ActorBox'; only immutable local Int/Bool/String and simple struct captures without ptr/resource fields are supported in this MVP",
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
			want: "capturing closure 'f' cannot be used as a pointer value; only direct local calls are supported in this MVP",
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
			want: "capturing closure literal captures 'y' but is not let-bound; only local direct calls can capture immutable Int/Bool/String values and simple structs without ptr/resource fields in this MVP",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := Parse([]byte(tt.src))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			_, err = Check(prog)
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
