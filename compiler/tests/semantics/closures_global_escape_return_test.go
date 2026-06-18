package compiler_test

import (
	"strings"
	"testing"

	compiler "tetra_language/compiler"
)

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
