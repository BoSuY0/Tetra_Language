package compiler_test

import (
	"strings"
	"testing"

	compiler "tetra_language/compiler"
)

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
