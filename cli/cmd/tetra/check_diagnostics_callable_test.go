package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckCommandJSONDiagnosticsForCallableMutableCaptureGlobalEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_callable_global_escape.tetra")
	src := `var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    var total: Int = 1
    cb = fn(x: Int) -> Int:
        return total + x
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "global-escaped function value captures mutable local 'total'")
}

func TestCheckCommandJSONDiagnosticsForCapturedCallableGlobalStorageCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_captured_callable_global_storage.tetra")
	src := `struct Holder:
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
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "captured function value cannot be stored in global function-typed value 'cb'")
}

func TestCheckCommandJSONDiagnosticsForFunctionTypedParameterGlobalStorageCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_function_parameter_global_storage.tetra")
	src := `var cb: fn(Int) -> Int = add0

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
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "function-typed parameter 'f' cannot be stored in global function-typed value 'cb'")
}

func TestCheckCommandJSONDiagnosticsForFunctionValueUnsupportedEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_function_value_escape.tetra")
	src := `func add1(x: Int) -> Int:
    return x + 1

func take_ptr(x: ptr) -> Int:
    return 0

func main() -> Int:
    let f: fn(Int) -> Int = add1
    return take_ptr(f)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "function value 'f' cannot escape outside the supported fnptr ABI")
}

func TestCheckCommandJSONDiagnosticsForCapturingClosureRawPointerEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_capturing_closure_raw_pointer_escape.tetra")
	src := `func choose(p: ptr) -> Int:
    return 0

func main() -> Int:
    let y: Int = 1
    let f: ptr = fn(x: Int) -> Int:
        return x + y
    return choose(f)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "capturing closure 'f' cannot escape as raw ptr")
}

func TestCheckCommandJSONDiagnosticsForCallableResourceCaptureEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_callable_resource_capture_escape.tetra")
	src := `struct PtrBox:
    p: ptr

func pick() -> fn(Int) -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let box: PtrBox = PtrBox(p: 0)
    return fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x + one + two + three + four + five + six + seven + eight

func main() -> Int:
    let f: fn(Int) -> Int = pick()
    return f(41)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "escaped function value captures local 'box' of type 'PtrBox'")
}

func TestCheckCommandJSONDiagnosticsForCallableMutableCaptureHeapEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_callable_mutable_capture_heap_escape.tetra")
	src := `func pick() -> fn(Int) -> Int:
    var total: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    return fn(x: Int) -> Int:
        return x + total + two + three + four + five + six + seven + eight + nine

func main() -> Int:
    let f: fn(Int) -> Int = pick()
    return f(41)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "heap-escaped function value captures mutable local 'total'")
}

func TestCheckCommandJSONDiagnosticsForGenericClosureCaptureCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_generic_closure_capture.tetra")
	src := `func main() -> Int:
    let base: Int = 1
    let f: fn(Int) -> Int = fn<T>(x: T) -> T:
        let _: Int = base
        return x
    return f(41)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "generic closure literal captures local 'base'")
}

func TestCheckCommandJSONDiagnosticsForGenericCallbackClosureCaptureCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_generic_callback_closure_capture.tetra")
	src := `func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    let base: Int = 1
    return apply(fn<T>(x: T) -> T:
        let _: Int = base
        return x
    , 41)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "callback argument 'closure literal' captures local 'base'")
}

func TestCheckCommandJSONDiagnosticsForFunctionTypedStorageUnsupportedCaptureCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_function_typed_storage_capture.tetra")
	src := `struct PtrBox:
    p: ptr

func main() -> Int:
    let box: PtrBox = PtrBox(p: 0)
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "function-typed storage 'f' captures unsupported local 'box'")
}

func TestCheckCommandJSONDiagnosticsForFunctionTypedReturnUnsupportedCaptureCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_function_typed_return_capture.tetra")
	src := `struct PtrBox:
    p: ptr

func pick() -> fn(Int) -> Int:
    let box: PtrBox = PtrBox(p: 0)
    return fn(x: Int) -> Int:
        let p: ptr = box.p
        let _: ptr = p
        return x

func main() -> Int:
    let f: fn(Int) -> Int = pick()
    return f(41)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "function-typed return 'closure literal' captures unsupported local 'box'")
}

func TestCheckCommandJSONDiagnosticsForCapturedClosureExplicitTypeArgsCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_captured_closure_explicit_type_args.tetra")
	src := `func main() -> Int:
    let y: Int = 1
    let f: ptr = fn(x: Int) -> Int:
        return x + y
    return f<Int>(41)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "explicit type arguments are not supported for captured closure 'f'")
}

func TestCheckCommandJSONDiagnosticsForFunctionTypedExplicitTypeArgsCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_function_typed_explicit_type_args.tetra")
	src := `func main() -> Int:
    let base: Int = 1
    let f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    return f<Int>(41)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "explicit type arguments are not supported for function-typed callback 'f'")
}

func TestCheckCommandJSONDiagnosticsForUnsupportedFunctionValueCallCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_function_value_call.tetra")
	src := `func main() -> Int:
    let p: ptr = 0
    return p(41)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "function value 'p' cannot be called through the supported fnptr ABI")
}

func TestCheckCommandJSONDiagnosticsForGenericClosurePointerEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_generic_closure_pointer_escape.tetra")
	src := `func use(p: ptr) -> Int:
    return 0

func main() -> Int:
    let id: ptr = fn<T>(x: T) -> T:
        return x
    return use(id)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "generic closure 'id' cannot be used as a pointer value")
}

func TestCheckCommandJSONDiagnosticsForGenericClosureDirectCallRequirementCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_generic_closure_direct_call_requirement.tetra")
	src := `func main() -> Int:
    var id: ptr = fn<T>(x: T) -> T:
        return x
    return id(1)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "generic closure 'id' requires the generic direct-call closure ABI")
}
