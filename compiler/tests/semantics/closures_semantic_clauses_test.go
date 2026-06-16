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
