package compiler_test

import (
	"strings"
	"testing"

	compiler "tetra_language/compiler"
)

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
