package compiler_test

import (
	"strings"
	"testing"

	compiler "tetra_language/compiler"

	"tetra_language/compiler/internal/ir"
)

func TestClosureLiteralParseCheckAndLower(t *testing.T) {
	src := []byte(`
func main() -> Int:
    let f: ptr = fn(x: Int) -> Int:
        return x
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

func TestGenericClosureLiteralParseCheckAndLower(t *testing.T) {
	src := []byte(`
func main() -> Int:
    let f: ptr = fn<T>(x: T) -> T:
        return x
    let v: Int = f(7)
    return v
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
			want: "generic closure 'id' cannot be used as a pointer value; generic closure ABI support is limited to let-bound direct local calls with inferable concrete arguments",
		},
		{
			name: "mutable binding direct call unsupported",
			src: `
func main() -> Int:
    var id: ptr = fn<T>(x: T) -> T:
        return x
    return id(1)
`,
			want: "generic closure 'id' requires the generic direct-call closure ABI: let-bound direct local call with inferable concrete arguments",
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
			want: "generic closure literal captures local 'base'; generic closure captures are not supported by the production fnptr ABI; use a non-generic closure or pass captured state explicitly",
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

func TestFunctionTypedClosureValueParseCheckAndLower(t *testing.T) {
	src := []byte(`
func main() -> Int:
    let add1: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + 1
    return add1(41)
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

func TestFunctionTypedNamedClosureSymbolParseCheckAndLower(t *testing.T) {
	src := []byte(`
closure add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let f: fn(Int) -> Int = add1
    return f(41)
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

func TestFunctionTypedNamedFunctionSymbolParseCheckAndLower(t *testing.T) {
	src := []byte(`
func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let f: fn(Int) -> Int = add1
    return f(41)
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

func TestFunctionTypedDirectNamedCallableParamParseCheckAndLower(t *testing.T) {
	src := []byte(`
func add1(x: Int) -> Int:
    return x + 1

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(add1, 41)
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
			name: "return generic named symbol with uninferable type parameter",
			src: `
func keep<T>(x: Int) -> Int:
    return x

func pick() -> fn(Int) -> Int:
    let f: fn(Int) -> Int = keep
    return f
`,
			want: "cannot infer generic argument 'T' for function-typed local 'f'",
		},
		{
			name: "return direct generic named symbol with uninferable type parameter",
			src: `
func keep<T>(x: Int) -> Int:
    return x

func pick() -> fn(Int) -> Int:
    return keep
`,
			want: "cannot infer generic argument 'T' for function return",
		},
		{
			name: "return generic ptr closure alias unsupported",
			src: `
func pick() -> fn(Int) -> Int:
    let id: ptr = fn<T>(x: T) -> T:
        return x
    return id
`,
			want: "generic function symbol 'id' cannot be returned as function-typed value; return fnptr ABI requires a monomorphic target",
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
			name: "return non callable literal unsupported",
			src: `
func pick() -> fn(Int) -> Int:
    return 1
`,
			want: "function-typed return must use a supported fnptr source: closure literal, function-typed local/global/struct field, direct named function/closure symbol, or function-typed return call",
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
			name: "escape unsupported",
			src: `
func consume(p: ptr) -> Int:
    return 0

func main() -> Int:
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x
    return consume(f)
`,
			want: "cannot escape outside the supported fnptr ABI",
		},
		{
			name: "raw ptr local call unsupported",
			src: `
func main() -> Int:
    let p: ptr = 0
    return p(41)
`,
			want: "function value 'p' cannot be called through the supported fnptr ABI; use a let-bound closure, function-typed local/global/struct field, enum payload, callback parameter, or direct named function symbol",
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
			want: "function-typed local 'f' initializer must be a symbol-backed function value, target-set-backed function value, direct named function symbol, or closure literal for the supported fnptr ABI",
		},
		{
			name: "generic named symbol with uninferable type parameter",
			src: `
func keep<T>(x: Int) -> Int:
    return x

func main() -> Int:
    let f: fn(Int) -> Int = keep
    return f(41)
`,
			want: "cannot infer generic argument 'T' for function-typed local 'f'",
		},
		{
			name: "generic ptr closure alias into function typed local",
			src: `
func main() -> Int:
    let id: ptr = fn<T>(x: T) -> T:
        return x
    let f: fn(Int) -> Int = id
    return f(41)
`,
			want: "generic function symbol 'id' cannot initialize function-typed local 'f'; local fnptr ABI requires a monomorphic target",
		},
		{
			name: "generic callback symbol with uninferable type parameter",
			src: `
func keep<T>(x: Int) -> Int:
    return x

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let f: fn(Int) -> Int = keep
    return apply(f, 41)
`,
			want: "cannot infer generic argument 'T' for function-typed local 'f'",
		},
		{
			name: "generic direct callback symbol with uninferable type parameter",
			src: `
func keep<T>(x: Int) -> Int:
    return x

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(keep, 41)
`,
			want: "cannot infer generic argument 'T' for callback argument for 'apply'",
		},
		{
			name: "semantic clause callback parameter unknown target rejected",
			src: `
func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func outer(cb: fn(Int) -> Int) -> Int
noalloc:
    return apply(cb, 41)

func main() -> Int:
    return 0
`,
			want: "callback argument for 'apply' has no known fnptr target under semantic clause 'noalloc'; pass a direct named function/closure symbol or a function-typed value with a stable target set",
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := compiler.Parse([]byte(tt.src))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			_, err = compiler.Check(prog)
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

func TestClosureImmutableScalarCaptureLabeledDirectCallParseCheckAndLower(t *testing.T) {
	src := []byte(`
func main() -> Int:
    let y: Int = 1
    let f: ptr = fn(x: Int) -> Int:
        return x + y
    return f(x: 41)
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

func TestClosureImmutableScalarCaptureMixedLabeledDirectCallDiagnostic(t *testing.T) {
	src := []byte(`
func main() -> Int:
    let y: Int = 1
    let f: ptr = fn(x: Int, z: Int) -> Int:
        return x + z + y
    return f(x: 40, 1)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected mixed label diagnostic")
	}
	want := "cannot mix labeled and unlabeled arguments in captured closure 'f'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestClosureImmutableScalarCaptureExplicitTypeArgsDirectCallDiagnostic(t *testing.T) {
	src := []byte(`
func main() -> Int:
    let y: Int = 1
    let f: ptr = fn(x: Int) -> Int:
        return x + y
    return f<Int>(41)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected explicit type argument diagnostic")
	}
	want := "explicit type arguments are not supported for captured closure 'f'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
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

func TestFunctionTypedLocalAliasesCapturedPtrClosureParseCheckAndLower(t *testing.T) {
	src := []byte(`
func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let cb: fn(Int) -> Int = captured
    return cb(41)
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

func TestCapturedPtrClosureAsDirectCallbackArgumentParseCheckAndLower(t *testing.T) {
	src := []byte(`
func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    return apply(captured, 41)
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

func TestCapturedPtrClosureReturnedAsFunctionTypedValueParseCheckAndLower(t *testing.T) {
	src := []byte(`
func make() -> fn(Int) -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    return captured

func main() -> Int:
    let cb: fn(Int) -> Int = make()
    return cb(41)
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

func TestFunctionTypedMutableLocalReassignsCapturedPtrClosureParseCheckAndLower(t *testing.T) {
	src := []byte(`
func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var cb: fn(Int) -> Int = add0
    cb = captured
    return cb(41)
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

func TestFunctionTypedStructFieldStoresCapturedPtrClosureParseCheckAndLower(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let holder: Holder = Holder(cb: captured)
    return holder.cb(41)
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

func TestFunctionTypedEnumPayloadStoresCapturedPtrClosureParseCheckAndLower(t *testing.T) {
	src := []byte(`
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let choice: MaybeCallback = MaybeCallback.some(captured)
    match choice:
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
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

func TestCapturedPtrClosureCanSnapshotIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    cb = captured
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_global_escape.tetra")
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

func TestFullCallableEscapedNineCaptureReturnPassesSemanticClassification(t *testing.T) {
	src := []byte(`
func make() -> fn(Int) -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    return fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine

func main() -> Int:
    let cb: fn(Int) -> Int = make()
    return cb(-3)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	sig, ok := checked.FuncSigs["make"]
	if !ok {
		t.Fatalf("missing make signature")
	}
	if string(sig.ReturnFunctionEscapeKind) != "heap" || !sig.ReturnFunctionHandleValue {
		t.Fatalf("return function escape metadata = (%q, %v), want (heap, true)", sig.ReturnFunctionEscapeKind, sig.ReturnFunctionHandleValue)
	}
	mainIndex := -1
	for i := range checked.Funcs {
		if checked.Funcs[i].Name == "main" {
			mainIndex = i
			break
		}
	}
	if mainIndex < 0 {
		t.Fatalf("missing main function")
	}
	cbInfo, ok := checked.Funcs[mainIndex].Locals["cb"]
	if !ok {
		t.Fatalf("missing cb local metadata")
	}
	if string(cbInfo.FunctionEscapeKind) != "heap" || !cbInfo.FunctionHandleValue {
		t.Fatalf("cb escape metadata = (%q, %v), want (heap, true)", cbInfo.FunctionEscapeKind, cbInfo.FunctionHandleValue)
	}
}

func TestFullCallableEscapedGlobalNineCapturePassesSemanticClassification(t *testing.T) {
	src := []byte(`
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func install() -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    cb = fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine
    return 0

func main() -> Int:
    let _: Int = install()
    return cb(-3)
`)
	file, err := compiler.ParseFile(src, "global_nine_capture_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
}

func TestFullCallableStructFieldNineCapturePassesSemanticClassification(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

func main() -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    let holder: Holder = Holder(cb: fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine
    )
    return holder.cb(-3)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	mainIndex := -1
	for i := range checked.Funcs {
		if checked.Funcs[i].Name == "main" {
			mainIndex = i
			break
		}
	}
	if mainIndex < 0 {
		t.Fatalf("missing main function")
	}
	holderInfo, ok := checked.Funcs[mainIndex].Locals["holder"]
	if !ok {
		t.Fatalf("missing holder local metadata")
	}
	cbInfo, ok := holderInfo.FunctionFields["cb"]
	if !ok {
		t.Fatalf("holder function fields = %#v, want cb", holderInfo.FunctionFields)
	}
	if string(cbInfo.FunctionEscapeKind) != "heap" || !cbInfo.FunctionHandleValue {
		t.Fatalf("holder.cb escape metadata = (%q, %v), want (heap, true)", cbInfo.FunctionEscapeKind, cbInfo.FunctionHandleValue)
	}
}

func TestFullCallableStructFieldNineCaptureLowersHandleEnvironment(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

func main() -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    let holder: Holder = Holder(cb: fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine
    )
    return holder.cb(-3)
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
	mainFn := findIRFuncByName(irProg, "main")
	if mainFn == nil {
		t.Fatalf("missing lowered main function")
	}
	if !hasInstrKind(mainFn, ir.IRAllocBytes) {
		t.Fatalf("main IR does not allocate a callable handle environment")
	}
	if !hasInstrKind(mainFn, ir.IRMemWritePtrOffset) {
		t.Fatalf("main IR does not write callable handle environment slots")
	}
}

func TestFullCallableEnumPayloadNineCapturePassesSemanticClassification(t *testing.T) {
	src := []byte(`
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    let choice: MaybeCallback = MaybeCallback.some(fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine
    )
    match choice:
    case MaybeCallback.some(cb):
        return cb(-3)
    case MaybeCallback.empty:
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
	mainIndex := -1
	for i := range checked.Funcs {
		if checked.Funcs[i].Name == "main" {
			mainIndex = i
			break
		}
	}
	if mainIndex < 0 {
		t.Fatalf("missing main function")
	}
	choiceInfo, ok := checked.Funcs[mainIndex].Locals["choice"]
	if !ok {
		t.Fatalf("missing choice local metadata")
	}
	payloadInfo, ok := choiceInfo.EnumPayloadFunctions["0:0"]
	if !ok {
		t.Fatalf("choice enum payload functions = %#v, want 0:0", choiceInfo.EnumPayloadFunctions)
	}
	if string(payloadInfo.FunctionEscapeKind) != "heap" || !payloadInfo.FunctionHandleValue {
		t.Fatalf("choice payload escape metadata = (%q, %v), want (heap, true)", payloadInfo.FunctionEscapeKind, payloadInfo.FunctionHandleValue)
	}
	cbInfo, ok := checked.Funcs[mainIndex].Locals["cb"]
	if !ok {
		t.Fatalf("missing pattern-bound cb metadata")
	}
	if string(cbInfo.FunctionEscapeKind) != "heap" || !cbInfo.FunctionHandleValue {
		t.Fatalf("cb escape metadata = (%q, %v), want (heap, true)", cbInfo.FunctionEscapeKind, cbInfo.FunctionHandleValue)
	}
}

func TestFullCallableEnumPayloadNineCaptureLowersHandleEnvironment(t *testing.T) {
	src := []byte(`
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

func main() -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    let choice: MaybeCallback = MaybeCallback.some(fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine
    )
    match choice:
    case MaybeCallback.some(cb):
        return cb(-3)
    case MaybeCallback.empty:
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
	mainFn := findIRFuncByName(irProg, "main")
	if mainFn == nil {
		t.Fatalf("missing lowered main function")
	}
	if !hasInstrKind(mainFn, ir.IRAllocBytes) {
		t.Fatalf("main IR does not allocate a callable handle environment")
	}
	if !hasInstrKind(mainFn, ir.IRMemWritePtrOffset) {
		t.Fatalf("main IR does not write callable handle environment slots")
	}
}

func TestFullCallableGlobalEscapeRejectsMutableCaptureDiagnostic(t *testing.T) {
	src := []byte(`
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    var total: Int = 1
    cb = fn(x: Int) -> Int:
        return total + x
    return 0
`)
	file, err := compiler.ParseFile(src, "global_escape_mutable_capture.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected mutable capture global-escape diagnostic")
	}
	want := "global-escaped function value captures mutable local 'total'; mutable by-reference captures require a proven lifetime and synchronization model"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestFullCallableGlobalEscapeRejectsMutableFunctionTypedAliasDiagnostic(t *testing.T) {
	src := []byte(`
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    var total: Int = 1
    let captured: fn(Int) -> Int = fn(x: Int) -> Int:
        return total + x
    cb = captured
    return 0
`)
	file, err := compiler.ParseFile(src, "global_escape_mutable_function_typed_alias.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected mutable function-typed alias global-escape diagnostic")
	}
	want := "global-escaped function value captures mutable local 'total'; mutable by-reference captures require a proven lifetime and synchronization model"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestFullCallableGlobalEscapeRejectsMutableStructFieldDiagnostic(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    var total: Int = 1
    let holder: Holder = Holder(cb: fn(x: Int) -> Int:
        return total + x
    )
    cb = holder.cb
    return 0
`)
	file, err := compiler.ParseFile(src, "global_escape_mutable_struct_field.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected mutable struct-field global-escape diagnostic")
	}
	want := "global-escaped function value captures mutable local 'total'; mutable by-reference captures require a proven lifetime and synchronization model"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestFullCallableGlobalEscapeRejectsMutableEnumPayloadDiagnostic(t *testing.T) {
	src := []byte(`
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    var total: Int = 1
    let choice: MaybeCallback = MaybeCallback.some(fn(x: Int) -> Int:
        return total + x
    )
    match choice:
    case MaybeCallback.some(payload):
        cb = payload
        return 0
    case MaybeCallback.empty:
        return 0
`)
	file, err := compiler.ParseFile(src, "global_escape_mutable_enum_payload.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected mutable enum-payload global-escape diagnostic")
	}
	want := "global-escaped function value captures mutable local 'total'; mutable by-reference captures require a proven lifetime and synchronization model"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestFullCallableGlobalEscapeRejectsMutableReturnedClosureDiagnostic(t *testing.T) {
	src := []byte(`
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    var total: Int = 1
    return fn(x: Int) -> Int:
        return total + x

func main() -> Int:
    cb = make()
    return 0
`)
	file, err := compiler.ParseFile(src, "global_escape_mutable_returned_closure.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected mutable returned-closure global-escape diagnostic")
	}
	want := "global-escaped function value captures mutable local 'total'; mutable by-reference captures require a proven lifetime and synchronization model"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestFullCallableGlobalEscapeRejectsMutableReturnedStructFieldDiagnostic(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func make() -> Holder:
    var total: Int = 1
    return Holder(cb: fn(x: Int) -> Int:
        return total + x
    )

func main() -> Int:
    let holder: Holder = make()
    cb = holder.cb
    return 0
`)
	file, err := compiler.ParseFile(src, "global_escape_mutable_returned_struct_field.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected mutable returned-struct-field global-escape diagnostic")
	}
	want := "global-escaped function value captures mutable local 'total'; mutable by-reference captures require a proven lifetime and synchronization model"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestFullCallableGlobalEscapeRejectsMutableReturnedEnumPayloadDiagnostic(t *testing.T) {
	src := []byte(`
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func make() -> MaybeCallback:
    var total: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int:
        return total + x
    )

func main() -> Int:
    let choice: MaybeCallback = make()
    match choice:
    case MaybeCallback.some(payload):
        cb = payload
        return 0
    case MaybeCallback.empty:
        return 0
`)
	file, err := compiler.ParseFile(src, "global_escape_mutable_returned_enum_payload.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected mutable returned-enum-payload global-escape diagnostic")
	}
	want := "global-escaped function value captures mutable local 'total'; mutable by-reference captures require a proven lifetime and synchronization model"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedFullCallableGlobalEscapeRejectsMutableReturnedClosureDiagnostic(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub func make() -> fn(Int) -> Int:
    var total: Int = 1
    return fn(x: Int) -> Int:
        return total + x
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    cb = callbacks.make()
    return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported mutable returned-closure global-escape diagnostic")
	}
	want := "global-escaped function value captures mutable local 'total'; mutable by-reference captures require a proven lifetime and synchronization model"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedFullCallableGlobalEscapeRejectsMutableReturnedStructFieldDiagnostic(t *testing.T) {
	files := map[string]string{
		"lib/maker.t4": `module lib.maker

pub struct Holder:
    cb: fn(Int) -> Int

pub func makeHolder() -> Holder:
    var total: Int = 1
    return Holder(cb: fn(x: Int) -> Int:
        return total + x
    )
`,
		"app/main.t4": `module app.main
import lib.maker as maker

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let holder: maker.Holder = maker.makeHolder()
    cb = holder.cb
    return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported mutable returned-struct-field global-escape diagnostic")
	}
	want := "global-escaped function value captures mutable local 'total'; mutable by-reference captures require a proven lifetime and synchronization model"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedFullCallableGlobalEscapeRejectsMutableReturnedEnumPayloadDiagnostic(t *testing.T) {
	files := map[string]string{
		"lib/maker.t4": `module lib.maker

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func makeChoice() -> MaybeCallback:
    var total: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int:
        return total + x
    )
`,
		"app/main.t4": `module app.main
import lib.maker as maker

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let choice: maker.MaybeCallback = maker.makeChoice()
    match choice:
    case maker.MaybeCallback.some(payload):
        cb = payload
        return 0
    case maker.MaybeCallback.empty:
        return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported mutable returned-enum-payload global-escape diagnostic")
	}
	want := "global-escaped function value captures mutable local 'total'; mutable by-reference captures require a proven lifetime and synchronization model"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedStructFieldCanSnapshotIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let holder: Holder = Holder(cb: fn(x: Int) -> Int:
        return x + base
    )
    cb = holder.cb
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_struct_field_global_escape.tetra")
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

func TestCapturedFunctionTypedEnumPayloadCanSnapshotIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let choice: MaybeCallback = MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    )
    match choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0
`)
	file, err := compiler.ParseFile(src, "captured_enum_payload_global_escape.tetra")
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

func TestCapturedFunctionTypedDirectClosureWholeEnumReassignmentCanSnapshotIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    var choice: MaybeCallback = MaybeCallback.empty
    choice = MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    )
    match choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0
`)
	file, err := compiler.ParseFile(src, "captured_direct_closure_whole_enum_reassign_global_escape.tetra")
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

func TestCapturedFunctionTypedReturnCallCanSnapshotIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let base: Int = 1
    return fn(x: Int) -> Int:
        return x + base

func main() -> Int:
    cb = make()
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_return_call_global_escape.tetra")
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

func TestCapturedFunctionTypedReturnLocalCanSnapshotIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let base: Int = 1
    return fn(x: Int) -> Int:
        return x + base

func main() -> Int:
    let local: fn(Int) -> Int = make()
    cb = local
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_return_local_global_escape.tetra")
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

func TestCapturedFunctionTypedReturnCallStructFieldCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let holder: Holder = Holder(cb: identity(captured))
    cb = holder.cb
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_return_call_struct_field_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured return-call struct-field global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'; global escape requires a direct fnptr snapshot with known captures and bounded environment slots"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedReturnCallEnumPayloadCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let choice: MaybeCallback = MaybeCallback.some(identity(captured))
    match choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0
`)
	file, err := compiler.ParseFile(src, "captured_return_call_enum_payload_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured return-call enum-payload global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedParameterReturnLocalAliasCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let local: fn(Int) -> Int = identity(captured)
    cb = local
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_parameter_return_local_alias_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured parameter-return local-alias global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedParameterReturnMutableLocalReassignmentCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var local: fn(Int) -> Int = add0
    local = identity(captured)
    cb = local
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_parameter_return_mutable_local_reassign_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured parameter-return mutable-local-reassignment global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedParameterReturnStructFieldReassignmentCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var holder: Holder = Holder(cb: add0)
    holder.cb = identity(captured)
    cb = holder.cb
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_parameter_return_struct_field_reassign_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured parameter-return struct-field-reassignment global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedParameterReturnLocalAliasCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let local: fn(Int) -> Int = callbacks.identity(captured)
    cb = local
    return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured parameter-return local-alias global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedParameterReturnMutableLocalReassignmentCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var local: fn(Int) -> Int = add0
    local = callbacks.identity(captured)
    cb = local
    return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured parameter-return mutable-local-reassignment global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedParameterReturnStructFieldCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let holder: Holder = Holder(cb: callbacks.identity(captured))
    cb = holder.cb
    return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured parameter-return struct-field global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedParameterReturnStructFieldReassignmentCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var holder: Holder = Holder(cb: add0)
    holder.cb = callbacks.identity(captured)
    cb = holder.cb
    return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured parameter-return struct-field-reassignment global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedParameterReturnNestedStructFieldReassignmentCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var box: Box = Box(holder: Holder(cb: add0))
    box.holder.cb = callbacks.identity(captured)
    cb = box.holder.cb
    return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured parameter-return nested-struct-field-reassignment global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedParameterReturnWholeStructReassignmentCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var holder: Holder = Holder(cb: add0)
    holder = Holder(cb: callbacks.identity(captured))
    cb = holder.cb
    return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured parameter-return whole-struct-reassignment global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedParameterReturnStructValuedFieldReassignmentCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var box: Box = Box(holder: Holder(cb: add0))
    box.holder = Holder(cb: callbacks.identity(captured))
    cb = box.holder.cb
    return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured parameter-return struct-valued-field-reassignment global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedParameterReturnWholeNestedStructReassignmentCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var box: Box = Box(holder: Holder(cb: add0))
    box = Box(holder: Holder(cb: callbacks.identity(captured)))
    cb = box.holder.cb
    return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured parameter-return whole-nested-struct-reassignment global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedParameterReturnEnumPayloadCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let choice: MaybeCallback = MaybeCallback.some(callbacks.identity(captured))
    match choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured parameter-return enum-payload global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedParameterReturnEnumPayloadReassignmentCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var choice: MaybeCallback = MaybeCallback.empty
    choice = MaybeCallback.some(callbacks.identity(captured))
    match choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured parameter-return enum-payload-reassignment global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedParameterReturnStructFieldEnumPayloadReassignmentCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

struct Box:
    choice: MaybeCallback

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var box: Box = Box(choice: MaybeCallback.empty)
    box.choice = MaybeCallback.some(callbacks.identity(captured))
    let choice: MaybeCallback = box.choice
    match choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured parameter-return struct-field enum-payload reassignment global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedReturnedStructEnumPayloadCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

struct Box:
    choice: MaybeCallback

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func makeBox(f: fn(Int) -> Int) -> Box:
    return Box(choice: MaybeCallback.some(f))

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let box: Box = makeBox(callbacks.identity(captured))
    let choice: MaybeCallback = box.choice
    match choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured returned-struct enum-payload global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedImportedReturnedStructEnumPayloadCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f
`,
		"lib/pack.t4": `module lib.pack

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox(f: fn(Int) -> Int) -> Box:
    return Box(choice: MaybeCallback.some(f))
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks
import lib.pack as pack

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let box: pack.Box = pack.makeBox(callbacks.identity(captured))
    let choice: pack.MaybeCallback = box.choice
    match choice:
    case pack.MaybeCallback.some(local):
        cb = local
        return 0
    case pack.MaybeCallback.empty:
        return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured imported-returned-struct enum-payload global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedReturnedStructEnumPayloadWholeStructReassignmentCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

struct Box:
    choice: MaybeCallback

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func makeBox(f: fn(Int) -> Int) -> Box:
    return Box(choice: MaybeCallback.some(f))

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var box: Box = Box(choice: MaybeCallback.empty)
    box = makeBox(callbacks.identity(captured))
    match box.choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured returned-struct enum-payload whole-struct-reassignment global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedNestedReturnedStructEnumPayloadCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

struct Box:
    choice: MaybeCallback

struct Outer:
    box: Box

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func makeBox(f: fn(Int) -> Int) -> Box:
    return Box(choice: MaybeCallback.some(f))

func makeOuter(f: fn(Int) -> Int) -> Outer:
    return Outer(box: makeBox(f))

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let outer: Outer = makeOuter(callbacks.identity(captured))
    match outer.box.choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured nested-returned-struct enum-payload global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedReturnCallCanSnapshotIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/maker.t4": `module lib.maker

pub func make() -> fn(Int) -> Int:
    let base: Int = 1
    return fn(x: Int) -> Int:
        return x + base
`,
		"app/main.t4": `module app.main
import lib.maker as maker

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    cb = maker.make()
    return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err != nil {
		t.Fatalf("buildOnlyFiles: %v", err)
	}
}

func TestCapturedFunctionTypedReturnAliasChainCanSnapshotIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let base: Int = 1
    return fn(x: Int) -> Int:
        return x + base

func forward() -> fn(Int) -> Int:
    let local: fn(Int) -> Int = make()
    return local

func main() -> Int:
    cb = forward()
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_return_alias_chain_global_escape.tetra")
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

func TestCapturedFunctionTypedReturnStructFieldReassignmentCanSnapshotIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let base: Int = 1
    return fn(x: Int) -> Int:
        return x + base

func main() -> Int:
    var holder: Holder = Holder(cb: add0)
    holder.cb = make()
    cb = holder.cb
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_return_struct_field_reassign_global_escape.tetra")
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

func TestCapturedFunctionTypedReturnMutableLocalReassignmentCanSnapshotIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let base: Int = 1
    return fn(x: Int) -> Int:
        return x + base

func main() -> Int:
    var local: fn(Int) -> Int = add0
    local = make()
    cb = local
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_return_mutable_local_reassign_global_escape.tetra")
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

func TestCapturedFunctionTypedReturnEnumPayloadReassignmentCanSnapshotIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let base: Int = 1
    return fn(x: Int) -> Int:
        return x + base

func main() -> Int:
    var choice: MaybeCallback = MaybeCallback.empty
    choice = MaybeCallback.some(make())
    match choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0
`)
	file, err := compiler.ParseFile(src, "captured_return_enum_payload_reassign_global_escape.tetra")
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

func TestCapturedFunctionTypedReturnedStructEnumPayloadCanSnapshotIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

struct Box:
    choice: MaybeCallback

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func makeBox() -> Box:
    let base: Int = 1
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    ))

func main() -> Int:
    let box: Box = makeBox()
    match box.choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0
`)
	file, err := compiler.ParseFile(src, "captured_returned_struct_enum_payload_global_escape.tetra")
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

func TestCapturedFunctionTypedReturnedEnumPayloadCanSnapshotIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    )

func main() -> Int:
    let choice: MaybeCallback = makeChoice()
    match choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0
`)
	file, err := compiler.ParseFile(src, "captured_returned_enum_payload_global_escape.tetra")
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

func TestImportedCapturedFunctionTypedReturnedEnumPayloadCanSnapshotIntoGlobalFunctionValue(t *testing.T) {
	packSrc := []byte(`module lib.pack

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    )
`)
	mainSrc := []byte(`module app.main
import lib.pack as pack

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let choice: pack.MaybeCallback = pack.makeChoice()
    match choice:
    case pack.MaybeCallback.some(local):
        cb = local
        return 0
    case pack.MaybeCallback.empty:
        return 0
`)
	pack, err := compiler.ParseFile(packSrc, "lib/pack.t4")
	if err != nil {
		t.Fatalf("ParseFile pack: %v", err)
	}
	main, err := compiler.ParseFile(mainSrc, "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile main: %v", err)
	}
	world := &compiler.World{
		EntryModule: "app.main",
		Files:       []*compiler.FileAST{pack, main},
		ByModule: map[string]*compiler.FileAST{
			"lib.pack": pack,
			"app.main": main,
		},
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestCapturedFunctionTypedParameterReturnEnumPayloadReassignmentCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    var choice: MaybeCallback = MaybeCallback.empty
    choice = MaybeCallback.some(identity(captured))
    match choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0
`)
	file, err := compiler.ParseFile(src, "captured_parameter_return_enum_payload_reassign_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured parameter-return enum payload reassignment global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedReturnWholeStructReassignmentCanSnapshotIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func make() -> fn(Int) -> Int:
    let base: Int = 1
    return fn(x: Int) -> Int:
        return x + base

func main() -> Int:
    var holder: Holder = Holder(cb: add0)
    holder = Holder(cb: make())
    cb = holder.cb
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_return_whole_struct_reassign_global_escape.tetra")
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

func TestCapturedFunctionTypedDirectClosureWholeStructReassignmentCanSnapshotIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    var holder: Holder = Holder(cb: add0)
    holder = Holder(cb: fn(x: Int) -> Int:
        return x + base
    )
    cb = holder.cb
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_direct_closure_whole_struct_reassign_global_escape.tetra")
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

func TestCapturedFunctionTypedDirectClosureWholeNestedStructReassignmentCanSnapshotIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    var box: Box = Box(holder: Holder(cb: add0))
    box = Box(holder: Holder(cb: fn(x: Int) -> Int:
        return x + base
    ))
    cb = box.holder.cb
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_direct_closure_whole_nested_struct_reassign_global_escape.tetra")
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

func TestFunctionTypedParameterCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    cb = f
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`)
	file, err := compiler.ParseFile(src, "function_typed_parameter_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected function-typed parameter global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'; global escape requires a direct fnptr snapshot with known captures and bounded environment slots"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestFunctionTypedParameterLocalAliasCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    let alias: fn(Int) -> Int = f
    cb = alias
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`)
	file, err := compiler.ParseFile(src, "function_typed_parameter_local_alias_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected function-typed parameter local-alias global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestFunctionTypedParameterMutableLocalReassignmentCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    var alias: fn(Int) -> Int = add0
    alias = f
    cb = alias
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`)
	file, err := compiler.ParseFile(src, "function_typed_parameter_mutable_local_reassignment_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected function-typed parameter mutable-local-reassignment global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestFunctionTypedParameterStructFieldCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    let holder: Holder = Holder(cb: f)
    cb = holder.cb
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`)
	file, err := compiler.ParseFile(src, "function_typed_parameter_struct_field_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected function-typed parameter struct-field global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestFunctionTypedParameterEnumPayloadCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
enum Choice:
    case some(fn(Int) -> Int)

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    let choice: Choice = Choice.some(f)
    match choice:
    case Choice.some(local):
        cb = local
        return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`)
	file, err := compiler.ParseFile(src, "function_typed_parameter_enum_payload_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected function-typed parameter enum-payload global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedParameterReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f

func main() -> Int:
    let base: Int = 1
    let local: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    cb = identity(local)
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_parameter_return_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured parameter-return global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestFunctionTypedParameterReturnCallCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f

func store(f: fn(Int) -> Int) -> Int:
    cb = identity(f)
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`)
	file, err := compiler.ParseFile(src, "function_typed_parameter_return_call_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected function-typed parameter return-call global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedParameterAliasReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    let alias: fn(Int) -> Int = f
    return alias

func main() -> Int:
    let base: Int = 1
    let local: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    cb = identity(local)
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_parameter_alias_return_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured parameter alias-return global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestFunctionTypedParameterAliasReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    let alias: fn(Int) -> Int = f
    return alias

func store(f: fn(Int) -> Int) -> Int:
    cb = identity(f)
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`)
	file, err := compiler.ParseFile(src, "function_typed_parameter_alias_return_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected function-typed parameter alias-return global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedParameterFieldReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    let source: Holder = Holder(cb: f)
    return source.cb

func main() -> Int:
    let base: Int = 1
    let local: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    cb = identity(local)
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_parameter_field_return_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured parameter field-return global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestFunctionTypedParameterFieldReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    let source: Holder = Holder(cb: f)
    return source.cb

func store(f: fn(Int) -> Int) -> Int:
    cb = identity(f)
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`)
	file, err := compiler.ParseFile(src, "function_typed_parameter_field_return_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected function-typed parameter field-return global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedParameterReassignedFieldReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    var source: Holder = Holder(cb: add0)
    source.cb = f
    return source.cb

func main() -> Int:
    let base: Int = 1
    let local: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    cb = identity(local)
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_parameter_reassigned_field_return_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured parameter reassigned-field-return global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedParameterEnumPayloadReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    let choice: MaybeCallback = MaybeCallback.some(f)
    match choice:
    case MaybeCallback.some(payload):
        return payload
    case MaybeCallback.empty:
        return add0

func main() -> Int:
    let base: Int = 1
    let local: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    cb = identity(local)
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_parameter_enum_payload_return_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured parameter enum-payload-return global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedParameterReassignedEnumPayloadReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    var choice: MaybeCallback = MaybeCallback.some(add0)
    choice = MaybeCallback.some(f)
    match choice:
    case MaybeCallback.some(payload):
        return payload
    case MaybeCallback.empty:
        return add0

func main() -> Int:
    let base: Int = 1
    let local: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    cb = identity(local)
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_parameter_reassigned_enum_payload_return_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured parameter reassigned-enum-payload-return global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedStructParameterFieldReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func pick(holder: Holder) -> fn(Int) -> Int:
    return holder.cb

func main() -> Int:
    let base: Int = 1
    let holder: Holder = Holder(cb: fn(x: Int) -> Int:
        return x + base
    )
    cb = pick(holder)
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_struct_parameter_field_return_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured struct-parameter field-return global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedStructParameterFieldReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub struct Holder:
    cb: fn(Int) -> Int

pub func pick(holder: Holder) -> fn(Int) -> Int:
    return holder.cb
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let holder: callbacks.Holder = callbacks.Holder(cb: fn(x: Int) -> Int:
        return x + base
    )
    cb = callbacks.pick(holder)
    return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured struct-parameter field-return global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedInlineStructParameterFieldReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub struct Holder:
    cb: fn(Int) -> Int

pub func pick(holder: Holder) -> fn(Int) -> Int:
    return holder.cb
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    cb = callbacks.pick(callbacks.Holder(cb: fn(x: Int) -> Int:
        return x + base
    ))
    return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured inline-struct-parameter field-return global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedNestedStructParameterFieldReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub struct Holder:
    cb: fn(Int) -> Int

pub struct Box:
    holder: Holder

pub func pick(box: Box) -> fn(Int) -> Int:
    return box.holder.cb
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    cb = callbacks.pick(callbacks.Box(holder: callbacks.Holder(cb: fn(x: Int) -> Int:
        return x + base
    )))
    return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured nested-struct-parameter field-return global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedStructParameterWholeReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub struct Holder:
    cb: fn(Int) -> Int

pub struct Box:
    holder: Holder

pub func echo(box: Box) -> Box:
    return box
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let box: callbacks.Box = callbacks.echo(callbacks.Box(holder: callbacks.Holder(cb: fn(x: Int) -> Int:
        return x + base
    )))
    cb = box.holder.cb
    return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured struct-parameter whole-return global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestFunctionTypedStructParameterWholeReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func echo(box: Box) -> Box:
    return box

func store(f: fn(Int) -> Int) -> Int:
    let box: Box = echo(Box(holder: Holder(cb: f)))
    cb = box.holder.cb
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`)
	file, err := compiler.ParseFile(src, "parameter_struct_parameter_whole_return_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected function-typed struct-parameter whole-return global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedFunctionTypedStructParameterWholeReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub struct Holder:
    cb: fn(Int) -> Int

pub struct Box:
    holder: Holder

pub func echo(box: Box) -> Box:
    return box
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    let box: callbacks.Box = callbacks.echo(callbacks.Box(holder: callbacks.Holder(cb: f)))
    cb = box.holder.cb
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported function-typed struct-parameter whole-return global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedEnumParameterWholeReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func echo(choice: MaybeCallback) -> MaybeCallback:
    return choice
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let choice: callbacks.MaybeCallback = callbacks.echo(callbacks.MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    ))
    match choice:
    case callbacks.MaybeCallback.some(local):
        cb = local
        return 0
    case callbacks.MaybeCallback.empty:
        return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured enum-parameter whole-return global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestFunctionTypedEnumParameterWholeReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func echo(choice: MaybeCallback) -> MaybeCallback:
    return choice

func store(f: fn(Int) -> Int) -> Int:
    let choice: MaybeCallback = echo(MaybeCallback.some(f))
    match choice:
    case MaybeCallback.some(local):
        cb = local
        return 0
    case MaybeCallback.empty:
        return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`)
	file, err := compiler.ParseFile(src, "parameter_enum_parameter_whole_return_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected function-typed enum-parameter whole-return global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedFunctionTypedEnumParameterWholeReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func echo(choice: MaybeCallback) -> MaybeCallback:
    return choice
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    let choice: callbacks.MaybeCallback = callbacks.echo(callbacks.MaybeCallback.some(f))
    match choice:
    case callbacks.MaybeCallback.some(local):
        cb = local
        return 0
    case callbacks.MaybeCallback.empty:
        return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported function-typed enum-parameter whole-return global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedEnumParameterPayloadReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func pick(choice: MaybeCallback) -> fn(Int) -> Int:
    match choice:
    case MaybeCallback.some(payload):
        return payload
    case MaybeCallback.empty:
        return add0

func main() -> Int:
    let base: Int = 1
    let choice: MaybeCallback = MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    )
    cb = pick(choice)
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_enum_parameter_payload_return_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured enum-parameter payload-return global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedEnumParameterPayloadReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func add0(x: Int) -> Int:
    return x

pub func pick(choice: MaybeCallback) -> fn(Int) -> Int:
    match choice:
    case MaybeCallback.some(payload):
        return payload
    case MaybeCallback.empty:
        return add0
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let choice: callbacks.MaybeCallback = callbacks.MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    )
    cb = callbacks.pick(choice)
    return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured enum-parameter payload-return global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedInlineEnumParameterPayloadReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/callbacks.t4": `module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func add0(x: Int) -> Int:
    return x

pub func pick(choice: MaybeCallback) -> fn(Int) -> Int:
    match choice:
    case MaybeCallback.some(payload):
        return payload
    case MaybeCallback.empty:
        return add0
`,
		"app/main.t4": `module app.main
import lib.callbacks as callbacks

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    cb = callbacks.pick(callbacks.MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    ))
    return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured inline-enum-parameter payload-return global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedInlineParameterReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f

func main() -> Int:
    let base: Int = 1
    cb = identity(fn(x: Int) -> Int:
        return x + base
    )
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_inline_parameter_return_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured inline parameter-return global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedParameterReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/identity.t4": `module lib.identity

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f
`,
		"app/main.t4": `module app.main
import lib.identity as id

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    cb = id.identity(fn(x: Int) -> Int:
        return x + base
    )
    return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured parameter-return global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedFunctionTypedParameterReturnCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/identity.t4": `module lib.identity

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f
`,
		"app/main.t4": `module app.main
import lib.identity as id

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    cb = id.identity(f)
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported function-typed parameter-return global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedParameterReturnedStructFieldCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func pack(f: fn(Int) -> Int) -> Holder:
    return Holder(cb: f)

func main() -> Int:
    let base: Int = 1
    let local: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    let holder: Holder = pack(local)
    cb = holder.cb
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_parameter_returned_struct_field_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured parameter returned-struct-field global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestFunctionTypedParameterReturnedStructFieldCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func pack(f: fn(Int) -> Int) -> Holder:
    return Holder(cb: f)

func store(f: fn(Int) -> Int) -> Int:
    let holder: Holder = pack(f)
    cb = holder.cb
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`)
	file, err := compiler.ParseFile(src, "function_typed_parameter_returned_struct_field_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected function-typed parameter returned-struct-field global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedInlineParameterReturnedStructFieldCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func pack(f: fn(Int) -> Int) -> Holder:
    return Holder(cb: f)

func main() -> Int:
    let base: Int = 1
    let holder: Holder = pack(fn(x: Int) -> Int:
        return x + base
    )
    cb = holder.cb
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_inline_parameter_returned_struct_field_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured inline parameter returned-struct-field global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedParameterReturnedStructFieldCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/pack.t4": `module lib.pack

pub struct Holder:
    cb: fn(Int) -> Int

pub func pack(f: fn(Int) -> Int) -> Holder:
    return Holder(cb: f)
`,
		"app/main.t4": `module app.main
import lib.pack as pack

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let holder: pack.Holder = pack.pack(fn(x: Int) -> Int:
        return x + base
    )
    cb = holder.cb
    return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured parameter returned-struct-field global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedFunctionTypedParameterReturnedStructFieldCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/pack.t4": `module lib.pack

pub struct Holder:
    cb: fn(Int) -> Int

pub func pack(f: fn(Int) -> Int) -> Holder:
    return Holder(cb: f)
`,
		"app/main.t4": `module app.main
import lib.pack as pack

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    let holder: pack.Holder = pack.pack(f)
    cb = holder.cb
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported function-typed parameter returned-struct-field global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedParameterReturnedNestedStructFieldCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func pack(f: fn(Int) -> Int) -> Box:
    return Box(holder: Holder(cb: f))

func main() -> Int:
    let base: Int = 1
    let captured: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    let box: Box = pack(captured)
    cb = box.holder.cb
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_parameter_returned_nested_struct_field_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured parameter returned-nested-struct-field global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestFunctionTypedParameterReturnedNestedStructFieldCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func pack(f: fn(Int) -> Int) -> Box:
    return Box(holder: Holder(cb: f))

func store(f: fn(Int) -> Int) -> Int:
    let box: Box = pack(f)
    cb = box.holder.cb
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`)
	file, err := compiler.ParseFile(src, "parameter_returned_nested_struct_field_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected function-typed parameter returned-nested-struct-field global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedInlineParameterReturnedNestedStructFieldCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

struct Box:
    holder: Holder

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func pack(f: fn(Int) -> Int) -> Box:
    return Box(holder: Holder(cb: f))

func main() -> Int:
    let base: Int = 1
    let box: Box = pack(fn(x: Int) -> Int:
        return x + base
    )
    cb = box.holder.cb
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_inline_parameter_returned_nested_struct_field_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured inline parameter returned-nested-struct-field global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedParameterReturnedNestedStructFieldCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/pack.t4": `module lib.pack

pub struct Holder:
    cb: fn(Int) -> Int

pub struct Box:
    holder: Holder

pub func pack(f: fn(Int) -> Int) -> Box:
    return Box(holder: Holder(cb: f))
`,
		"app/main.t4": `module app.main
import lib.pack as pack

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let box: pack.Box = pack.pack(fn(x: Int) -> Int:
        return x + base
    )
    cb = box.holder.cb
    return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured parameter returned-nested-struct-field global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedFunctionTypedParameterReturnedNestedStructFieldCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/pack.t4": `module lib.pack

pub struct Holder:
    cb: fn(Int) -> Int

pub struct Box:
    holder: Holder

pub func pack(f: fn(Int) -> Int) -> Box:
    return Box(holder: Holder(cb: f))
`,
		"app/main.t4": `module app.main
import lib.pack as pack

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    let box: pack.Box = pack.pack(f)
    cb = box.holder.cb
    return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported function-typed parameter returned-nested-struct-field global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedParameterAliasReturnedStructFieldCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func pack(f: fn(Int) -> Int) -> Holder:
    let alias: fn(Int) -> Int = f
    return Holder(cb: alias)

func main() -> Int:
    let base: Int = 1
    let captured: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    let holder: Holder = pack(captured)
    cb = holder.cb
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_parameter_alias_returned_struct_field_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured parameter alias returned-struct-field global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedParameterFieldAliasReturnedStructFieldCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func pack(f: fn(Int) -> Int) -> Holder:
    let source: Holder = Holder(cb: f)
    return Holder(cb: source.cb)

func main() -> Int:
    let base: Int = 1
    let captured: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    let holder: Holder = pack(captured)
    cb = holder.cb
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_parameter_field_alias_returned_struct_field_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured parameter field-alias returned-struct-field global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedParameterReturnedEnumPayloadCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func pack(f: fn(Int) -> Int) -> MaybeCallback:
    return MaybeCallback.some(f)

func main() -> Int:
    let base: Int = 1
    let captured: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    let choice: MaybeCallback = pack(captured)
    match choice:
    case MaybeCallback.some(payload):
        cb = payload
        return 0
    case MaybeCallback.empty:
        return 0
`)
	file, err := compiler.ParseFile(src, "captured_parameter_returned_enum_payload_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured parameter returned-enum-payload global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestFunctionTypedParameterReturnedEnumPayloadCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func pack(f: fn(Int) -> Int) -> MaybeCallback:
    return MaybeCallback.some(f)

func store(f: fn(Int) -> Int) -> Int:
    let choice: MaybeCallback = pack(f)
    match choice:
    case MaybeCallback.some(payload):
        cb = payload
        return 0
    case MaybeCallback.empty:
        return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`)
	file, err := compiler.ParseFile(src, "function_typed_parameter_returned_enum_payload_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected function-typed parameter returned-enum-payload global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedParameterAliasReturnedEnumPayloadCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func pack(f: fn(Int) -> Int) -> MaybeCallback:
    let alias: fn(Int) -> Int = f
    return MaybeCallback.some(alias)

func main() -> Int:
    let base: Int = 1
    let captured: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    let choice: MaybeCallback = pack(captured)
    match choice:
    case MaybeCallback.some(payload):
        cb = payload
        return 0
    case MaybeCallback.empty:
        return 0
`)
	file, err := compiler.ParseFile(src, "captured_parameter_alias_returned_enum_payload_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured parameter alias returned-enum-payload global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedParameterFieldAliasReturnedEnumPayloadCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func pack(f: fn(Int) -> Int) -> MaybeCallback:
    let source: Holder = Holder(cb: f)
    return MaybeCallback.some(source.cb)

func main() -> Int:
    let base: Int = 1
    let captured: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    let choice: MaybeCallback = pack(captured)
    match choice:
    case MaybeCallback.some(payload):
        cb = payload
        return 0
    case MaybeCallback.empty:
        return 0
`)
	file, err := compiler.ParseFile(src, "captured_parameter_field_alias_returned_enum_payload_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured parameter field-alias returned-enum-payload global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedInlineParameterReturnedEnumPayloadCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func pack(f: fn(Int) -> Int) -> MaybeCallback:
    return MaybeCallback.some(f)

func main() -> Int:
    let base: Int = 1
    let choice: MaybeCallback = pack(fn(x: Int) -> Int:
        return x + base
    )
    match choice:
    case MaybeCallback.some(payload):
        cb = payload
        return 0
    case MaybeCallback.empty:
        return 0
`)
	file, err := compiler.ParseFile(src, "captured_inline_parameter_returned_enum_payload_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured inline parameter returned-enum-payload global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedCapturedFunctionTypedParameterReturnedEnumPayloadCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/pack.t4": `module lib.pack

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func pack(f: fn(Int) -> Int) -> MaybeCallback:
    return MaybeCallback.some(f)
`,
		"app/main.t4": `module app.main
import lib.pack as pack

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let base: Int = 1
    let choice: pack.MaybeCallback = pack.pack(fn(x: Int) -> Int:
        return x + base
    )
    match choice:
    case pack.MaybeCallback.some(payload):
        cb = payload
        return 0
    case pack.MaybeCallback.empty:
        return 0
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported captured parameter returned-enum-payload global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestImportedFunctionTypedParameterReturnedEnumPayloadCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	files := map[string]string{
		"lib/pack.t4": `module lib.pack

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func pack(f: fn(Int) -> Int) -> MaybeCallback:
    return MaybeCallback.some(f)
`,
		"app/main.t4": `module app.main
import lib.pack as pack

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func store(f: fn(Int) -> Int) -> Int:
    let choice: pack.MaybeCallback = pack.pack(f)
    match choice:
    case pack.MaybeCallback.some(payload):
        cb = payload
        return 0
    case pack.MaybeCallback.empty:
        return 0

func main() -> Int:
    let base: Int = 1
    return store(fn(x: Int) -> Int:
        return x + base
    )
`,
	}
	err := buildOnlyFiles(t, files, "app/main.t4")
	if err == nil {
		t.Fatalf("expected imported function-typed parameter returned-enum-payload global escape diagnostic")
	}
	want := "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedParameterReturnedStructFieldReassignmentCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
struct Holder:
    cb: fn(Int) -> Int

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func pack(f: fn(Int) -> Int) -> Holder:
    return Holder(cb: f)

func main() -> Int:
    let base: Int = 1
    let captured: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    var holder: Holder = Holder(cb: add0)
    holder = pack(captured)
    cb = holder.cb
    return 0
`)
	file, err := compiler.ParseFile(src, "captured_parameter_returned_struct_field_reassign_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured parameter returned-struct-field reassignment global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestCapturedFunctionTypedParameterReturnedEnumPayloadReassignmentCannotEscapeIntoGlobalFunctionValue(t *testing.T) {
	src := []byte(`
enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func pack(f: fn(Int) -> Int) -> MaybeCallback:
    return MaybeCallback.some(f)

func main() -> Int:
    let base: Int = 1
    let captured: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    var choice: MaybeCallback = MaybeCallback.empty
    choice = pack(captured)
    match choice:
    case MaybeCallback.some(payload):
        cb = payload
        return 0
    case MaybeCallback.empty:
        return 0
`)
	file, err := compiler.ParseFile(src, "captured_parameter_returned_enum_payload_reassign_global_escape.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected captured parameter returned-enum-payload reassignment global escape diagnostic")
	}
	want := "captured function value cannot be stored in global function-typed value 'cb'"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

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
