package semantics

import (
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/module"
)

func TestReturnedStructEnumPayloadFieldSignatureMetadata(t *testing.T) {
	src := []byte(`module app.main

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

struct Box:
    choice: MaybeCallback

func makeBox(cb: fn(Int) -> Int) -> Box:
    return Box(choice: MaybeCallback.some(cb))

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let box: Box = makeBox(add1)
    let choice: MaybeCallback = box.choice
    match choice:
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`)
	file, err := frontend.ParseFile(src, "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &module.World{
		EntryModule: "app.main",
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{"app.main": file},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	sig := checked.FuncSigs["app.main.makeBox"]
	if len(sig.ReturnEnumPayloadFields) == 0 {
		t.Fatalf("ReturnEnumPayloadFields is empty")
	}
	field, ok := sig.ReturnEnumPayloadFields["choice#0:0"]
	if !ok {
		t.Fatalf("ReturnEnumPayloadFields = %#v, want choice#0:0", sig.ReturnEnumPayloadFields)
	}
	if field.FunctionParamName != "cb" {
		t.Fatalf("FunctionParamName = %q, want cb", field.FunctionParamName)
	}
}

func TestReturnedStructEnumPayloadFieldCallSiteCaptureMetadata(t *testing.T) {
	callbacksSrc := []byte(`module lib.callbacks

pub func identity(cb: fn(Int) -> Int) -> fn(Int) -> Int:
    return cb
`)
	mainSrc := []byte(`module app.main
import lib.callbacks as callbacks

enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

struct Box:
    choice: MaybeCallback

func makeBox(cb: fn(Int) -> Int) -> Box:
    return Box(choice: MaybeCallback.some(cb))

func main() -> Int:
    let base: Int = 1
    let captured: ptr = fn(x: Int) -> Int:
        return x + base
    let box: Box = makeBox(callbacks.identity(captured))
    let choice: MaybeCallback = box.choice
    match choice:
    case MaybeCallback.some(cb):
        return cb(41)
    case MaybeCallback.empty:
        return 0
`)
	callbacks, err := frontend.ParseFile(callbacksSrc, "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("ParseFile callbacks: %v", err)
	}
	main, err := frontend.ParseFile(mainSrc, "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile main: %v", err)
	}
	world := &module.World{
		EntryModule: "app.main",
		Files:       []*frontend.FileAST{callbacks, main},
		ByModule: map[string]*frontend.FileAST{
			"lib.callbacks": callbacks,
			"app.main":      main,
		},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	identitySig := checked.FuncSigs["lib.callbacks.identity"]
	if identitySig.ReturnFunctionParamName != "cb" {
		t.Fatalf("identity ReturnFunctionParamName = %q, want cb", identitySig.ReturnFunctionParamName)
	}
	var mainFunc CheckedFunc
	for _, fn := range checked.Funcs {
		if fn.Name == "app.main.main" {
			mainFunc = fn
			break
		}
	}
	box := mainFunc.Locals["box"]
	field, ok := box.EnumPayloadFields["choice#0:0"]
	if !ok {
		t.Fatalf("box EnumPayloadFields = %#v, want choice#0:0", box.EnumPayloadFields)
	}
	if len(field.FunctionEscapeCaptures) == 0 && len(field.FunctionCaptures) == 0 {
		t.Fatalf("box.choice payload has no captures: %#v", field)
	}
	if field.FunctionValue == "" {
		t.Fatalf("box.choice payload has no function value: %#v", field)
	}
	choice := mainFunc.Locals["choice"]
	payload, ok := choice.EnumPayloadFunctions["0:0"]
	if !ok {
		t.Fatalf("choice EnumPayloadFunctions = %#v, want 0:0", choice.EnumPayloadFunctions)
	}
	if len(payload.FunctionEscapeCaptures) == 0 && len(payload.FunctionCaptures) == 0 {
		t.Fatalf("choice payload has no captures: %#v", payload)
	}
	if payload.FunctionValue == "" {
		t.Fatalf("choice payload has no function value: %#v", payload)
	}
	bound := mainFunc.Locals["cb"]
	if len(bound.FunctionEscapeCaptures) == 0 && len(bound.FunctionCaptures) == 0 {
		t.Fatalf("pattern cb has no captures: %#v", bound)
	}
	if bound.FunctionValue == "" {
		t.Fatalf("pattern cb has no function value: %#v", bound)
	}
}

func TestImportedReturnedStructEnumPayloadDirectClosureMetadata(t *testing.T) {
	packSrc := []byte(`module lib.pack

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    let base: Int = 1
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    ))
`)
	mainSrc := []byte(`module app.main
import lib.pack as pack

var cb: fn(Int) -> Int = add0

func add0(x: Int) -> Int:
    return x

func main() -> Int:
    let box: pack.Box = pack.makeBox()
    match box.choice:
    case pack.MaybeCallback.some(local):
        cb = local
        return 0
    case pack.MaybeCallback.empty:
        return 0
`)
	pack, err := frontend.ParseFile(packSrc, "lib/pack.t4")
	if err != nil {
		t.Fatalf("ParseFile pack: %v", err)
	}
	main, err := frontend.ParseFile(mainSrc, "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile main: %v", err)
	}
	world := &module.World{
		EntryModule: "app.main",
		Files:       []*frontend.FileAST{pack, main},
		ByModule: map[string]*frontend.FileAST{
			"lib.pack": pack,
			"app.main": main,
		},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	makeBox := checked.FuncSigs["lib.pack.makeBox"]
	field, ok := makeBox.ReturnEnumPayloadFields["choice#0:0"]
	if !ok {
		t.Fatalf("makeBox ReturnEnumPayloadFields = %#v, want choice#0:0", makeBox.ReturnEnumPayloadFields)
	}
	if !field.FunctionReturnSnapshotAlias || len(field.FunctionEscapeCaptures) == 0 || field.FunctionParamName != "" {
		t.Fatalf("makeBox returned payload metadata = %#v, want direct snapshot", field)
	}
	var mainFunc CheckedFunc
	for _, fn := range checked.Funcs {
		if fn.Name == "app.main.main" {
			mainFunc = fn
			break
		}
	}
	box := mainFunc.Locals["box"]
	field, ok = box.EnumPayloadFields["choice#0:0"]
	if !ok {
		t.Fatalf("box EnumPayloadFields = %#v, want choice#0:0", box.EnumPayloadFields)
	}
	if !field.FunctionReturnSnapshotAlias || len(field.FunctionEscapeCaptures) == 0 || field.FunctionParamName != "" {
		t.Fatalf("box returned payload metadata = %#v, want direct snapshot", field)
	}
	bound := mainFunc.Locals["local"]
	if !bound.FunctionReturnSnapshotAlias || len(bound.FunctionEscapeCaptures) == 0 || bound.FunctionParamName != "" {
		t.Fatalf("pattern local metadata = %#v, want direct snapshot", bound)
	}
}

func TestImportedReturnedEnumPayloadDirectClosureMetadata(t *testing.T) {
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
	pack, err := frontend.ParseFile(packSrc, "lib/pack.t4")
	if err != nil {
		t.Fatalf("ParseFile pack: %v", err)
	}
	main, err := frontend.ParseFile(mainSrc, "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile main: %v", err)
	}
	world := &module.World{
		EntryModule: "app.main",
		Files:       []*frontend.FileAST{pack, main},
		ByModule: map[string]*frontend.FileAST{
			"lib.pack": pack,
			"app.main": main,
		},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	makeChoice := checked.FuncSigs["lib.pack.makeChoice"]
	payload, ok := makeChoice.ReturnEnumPayloadFunctions["0:0"]
	if !ok {
		t.Fatalf("makeChoice ReturnEnumPayloadFunctions = %#v, want 0:0", makeChoice.ReturnEnumPayloadFunctions)
	}
	if !payload.FunctionReturnSnapshotAlias || len(payload.FunctionEscapeCaptures) == 0 || payload.FunctionParamName != "" {
		t.Fatalf("makeChoice returned payload metadata = %#v, want direct snapshot", payload)
	}
	var mainFunc CheckedFunc
	for _, fn := range checked.Funcs {
		if fn.Name == "app.main.main" {
			mainFunc = fn
			break
		}
	}
	choice := mainFunc.Locals["choice"]
	payload, ok = choice.EnumPayloadFunctions["0:0"]
	if !ok {
		t.Fatalf("choice EnumPayloadFunctions = %#v, want 0:0", choice.EnumPayloadFunctions)
	}
	if !payload.FunctionReturnSnapshotAlias || len(payload.FunctionEscapeCaptures) == 0 || payload.FunctionParamName != "" {
		t.Fatalf("choice returned payload metadata = %#v, want direct snapshot", payload)
	}
	bound := mainFunc.Locals["local"]
	if !bound.FunctionReturnSnapshotAlias || len(bound.FunctionEscapeCaptures) == 0 || bound.FunctionParamName != "" {
		t.Fatalf("pattern local metadata = %#v, want direct snapshot", bound)
	}
}

func TestInterfaceReturnedStructEnumPayloadInlineClosureMetadata(t *testing.T) {
	packSrc := []byte(`module lib.pack

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    return Box(choice: MaybeCallback.some(fn(p0: Int) -> Int = 0))
`)
	mainSrc := []byte(`module app.main
import lib.pack as pack

func main() -> Int:
    let box: pack.Box = pack.makeBox()
    match box.choice:
    case pack.MaybeCallback.some(local):
        return local(41)
    case pack.MaybeCallback.empty:
        return 0
`)
	pack, err := frontend.ParseFile(packSrc, "lib/pack.t4i")
	if err != nil {
		t.Fatalf("ParseFile pack: %v", err)
	}
	main, err := frontend.ParseFile(mainSrc, "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile main: %v", err)
	}
	world := &module.World{
		EntryModule:      "app.main",
		Files:            []*frontend.FileAST{pack, main},
		InterfaceModules: map[string]bool{"lib.pack": true},
		ByModule: map[string]*frontend.FileAST{
			"lib.pack": pack,
			"app.main": main,
		},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	makeBox := checked.FuncSigs["lib.pack.makeBox"]
	field, ok := makeBox.ReturnEnumPayloadFields["choice#0:0"]
	if !ok {
		t.Fatalf("makeBox ReturnEnumPayloadFields = %#v, want choice#0:0", makeBox.ReturnEnumPayloadFields)
	}
	if field.FunctionValue == "" || field.FunctionParamName != "" {
		t.Fatalf("makeBox returned payload metadata = %#v, want synthetic closure target", field)
	}
	var mainFunc CheckedFunc
	for _, fn := range checked.Funcs {
		if fn.Name == "app.main.main" {
			mainFunc = fn
			break
		}
	}
	box := mainFunc.Locals["box"]
	field, ok = box.EnumPayloadFields["choice#0:0"]
	if !ok {
		t.Fatalf("box EnumPayloadFields = %#v, want choice#0:0", box.EnumPayloadFields)
	}
	if field.FunctionValue == "" || field.FunctionParamName != "" {
		t.Fatalf("box returned payload metadata = %#v, want synthetic closure target", field)
	}
	bound := mainFunc.Locals["local"]
	if bound.FunctionValue == "" || bound.FunctionParamName != "" {
		t.Fatalf("pattern local metadata = %#v, want synthetic closure target", bound)
	}
}

func TestInterfaceReturnedStructEnumPayloadInlineThrowingClosureMetadata(t *testing.T) {
	packSrc := []byte(`module lib.pack

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    return Box(choice: MaybeCallback.some(fn(p0: Int) -> Int throws Boom = 0))
`)
	mainSrc := []byte(`module app.main
import lib.pack as pack

func caller() -> Int throws pack.Boom:
    let box: pack.Box = pack.makeBox()
    match box.choice:
    case pack.MaybeCallback.some(local):
        return try local(41)
    case pack.MaybeCallback.empty:
        return 0

func main() -> Int:
    return catch caller():
    case pack.Boom.bad:
        0
`)
	pack, err := frontend.ParseFile(packSrc, "lib/pack.t4i")
	if err != nil {
		t.Fatalf("ParseFile pack: %v", err)
	}
	main, err := frontend.ParseFile(mainSrc, "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile main: %v", err)
	}
	world := &module.World{
		EntryModule:      "app.main",
		Files:            []*frontend.FileAST{pack, main},
		InterfaceModules: map[string]bool{"lib.pack": true},
		ByModule: map[string]*frontend.FileAST{
			"lib.pack": pack,
			"app.main": main,
		},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	makeBox := checked.FuncSigs["lib.pack.makeBox"]
	field, ok := makeBox.ReturnEnumPayloadFields["choice#0:0"]
	if !ok {
		t.Fatalf("makeBox ReturnEnumPayloadFields = %#v, want choice#0:0", makeBox.ReturnEnumPayloadFields)
	}
	if field.FunctionValue == "" || field.FunctionParamName != "" || field.FunctionThrowsType != "lib.pack.Boom" {
		t.Fatalf("makeBox returned payload metadata = %#v, want synthetic throwing closure target", field)
	}
	var callerFunc CheckedFunc
	for _, fn := range checked.Funcs {
		if fn.Name == "app.main.caller" {
			callerFunc = fn
			break
		}
	}
	box := callerFunc.Locals["box"]
	field, ok = box.EnumPayloadFields["choice#0:0"]
	if !ok {
		t.Fatalf("box EnumPayloadFields = %#v, want choice#0:0", box.EnumPayloadFields)
	}
	if field.FunctionValue == "" || field.FunctionParamName != "" || field.FunctionThrowsType != "lib.pack.Boom" {
		t.Fatalf("box returned payload metadata = %#v, want synthetic throwing closure target", field)
	}
	bound := callerFunc.Locals["local"]
	if bound.FunctionValue == "" || bound.FunctionParamName != "" || bound.FunctionThrowsType != "lib.pack.Boom" {
		t.Fatalf("pattern local metadata = %#v, want synthetic throwing closure target", bound)
	}
}

func TestInterfaceReturnedStructFieldInlineThrowingClosureMetadata(t *testing.T) {
	packSrc := []byte(`module lib.pack

pub enum Boom:
    case bad

pub struct Holder:
    cb: fn(Int) -> Int throws Boom

pub func makeHolder() -> Holder:
    return Holder(cb: fn(p0: Int) -> Int throws Boom = 0)
`)
	mainSrc := []byte(`module app.main
import lib.pack as pack

func caller() -> Int throws pack.Boom:
    let holder: pack.Holder = pack.makeHolder()
    return try holder.cb(41)

func main() -> Int:
    return catch caller():
    case pack.Boom.bad:
        0
`)
	pack, err := frontend.ParseFile(packSrc, "lib/pack.t4i")
	if err != nil {
		t.Fatalf("ParseFile pack: %v", err)
	}
	main, err := frontend.ParseFile(mainSrc, "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile main: %v", err)
	}
	world := &module.World{
		EntryModule:      "app.main",
		Files:            []*frontend.FileAST{pack, main},
		InterfaceModules: map[string]bool{"lib.pack": true},
		ByModule: map[string]*frontend.FileAST{
			"lib.pack": pack,
			"app.main": main,
		},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	makeHolder := checked.FuncSigs["lib.pack.makeHolder"]
	field, ok := makeHolder.ReturnFunctionFields["cb"]
	if !ok {
		t.Fatalf("makeHolder ReturnFunctionFields = %#v, want cb", makeHolder.ReturnFunctionFields)
	}
	if field.FunctionValue == "" || field.FunctionParamName != "" || field.FunctionThrowsType != "lib.pack.Boom" {
		t.Fatalf("makeHolder returned field metadata = %#v, want synthetic throwing closure target", field)
	}
	var callerFunc CheckedFunc
	for _, fn := range checked.Funcs {
		if fn.Name == "app.main.caller" {
			callerFunc = fn
			break
		}
	}
	holder := callerFunc.Locals["holder"]
	field, ok = holder.FunctionFields["cb"]
	if !ok {
		t.Fatalf("holder FunctionFields = %#v, want cb", holder.FunctionFields)
	}
	if field.FunctionValue == "" || field.FunctionParamName != "" || field.FunctionThrowsType != "lib.pack.Boom" {
		t.Fatalf("holder returned field metadata = %#v, want synthetic throwing closure target", field)
	}
}

func TestInterfaceReturnedEnumPayloadInlineClosureMetadata(t *testing.T) {
	packSrc := []byte(`module lib.pack

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func makeChoice() -> MaybeCallback:
    return MaybeCallback.some(fn(p0: Int) -> Int = 0)
`)
	mainSrc := []byte(`module app.main
import lib.pack as pack

func main() -> Int:
    let choice: pack.MaybeCallback = pack.makeChoice()
    match choice:
    case pack.MaybeCallback.some(local):
        return local(41)
    case pack.MaybeCallback.empty:
        return 0
`)
	pack, err := frontend.ParseFile(packSrc, "lib/pack.t4i")
	if err != nil {
		t.Fatalf("ParseFile pack: %v", err)
	}
	main, err := frontend.ParseFile(mainSrc, "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile main: %v", err)
	}
	world := &module.World{
		EntryModule:      "app.main",
		Files:            []*frontend.FileAST{pack, main},
		InterfaceModules: map[string]bool{"lib.pack": true},
		ByModule: map[string]*frontend.FileAST{
			"lib.pack": pack,
			"app.main": main,
		},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	makeChoice := checked.FuncSigs["lib.pack.makeChoice"]
	payload, ok := makeChoice.ReturnEnumPayloadFunctions["0:0"]
	if !ok {
		t.Fatalf("makeChoice ReturnEnumPayloadFunctions = %#v, want 0:0", makeChoice.ReturnEnumPayloadFunctions)
	}
	if payload.FunctionValue == "" || payload.FunctionParamName != "" {
		t.Fatalf("makeChoice returned payload metadata = %#v, want synthetic closure target", payload)
	}
	var mainFunc CheckedFunc
	for _, fn := range checked.Funcs {
		if fn.Name == "app.main.main" {
			mainFunc = fn
			break
		}
	}
	choice := mainFunc.Locals["choice"]
	payload, ok = choice.EnumPayloadFunctions["0:0"]
	if !ok {
		t.Fatalf("choice EnumPayloadFunctions = %#v, want 0:0", choice.EnumPayloadFunctions)
	}
	if payload.FunctionValue == "" || payload.FunctionParamName != "" {
		t.Fatalf("choice returned payload metadata = %#v, want synthetic closure target", payload)
	}
	bound := mainFunc.Locals["local"]
	if bound.FunctionValue == "" || bound.FunctionParamName != "" {
		t.Fatalf("pattern local metadata = %#v, want synthetic closure target", bound)
	}
}

func TestInterfaceReturnedEnumPayloadInlineThrowingClosureMetadata(t *testing.T) {
	packSrc := []byte(`module lib.pack

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub func makeChoice() -> MaybeCallback:
    return MaybeCallback.some(fn(p0: Int) -> Int throws Boom = 0)
`)
	mainSrc := []byte(`module app.main
import lib.pack as pack

func caller() -> Int throws pack.Boom:
    let choice: pack.MaybeCallback = pack.makeChoice()
    match choice:
    case pack.MaybeCallback.some(local):
        return try local(41)
    case pack.MaybeCallback.empty:
        return 0

func main() -> Int:
    return catch caller():
    case pack.Boom.bad:
        0
`)
	pack, err := frontend.ParseFile(packSrc, "lib/pack.t4i")
	if err != nil {
		t.Fatalf("ParseFile pack: %v", err)
	}
	main, err := frontend.ParseFile(mainSrc, "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile main: %v", err)
	}
	world := &module.World{
		EntryModule:      "app.main",
		Files:            []*frontend.FileAST{pack, main},
		InterfaceModules: map[string]bool{"lib.pack": true},
		ByModule: map[string]*frontend.FileAST{
			"lib.pack": pack,
			"app.main": main,
		},
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	makeChoice := checked.FuncSigs["lib.pack.makeChoice"]
	payload, ok := makeChoice.ReturnEnumPayloadFunctions["0:0"]
	if !ok {
		t.Fatalf("makeChoice ReturnEnumPayloadFunctions = %#v, want 0:0", makeChoice.ReturnEnumPayloadFunctions)
	}
	if payload.FunctionValue == "" || payload.FunctionParamName != "" || payload.FunctionThrowsType != "lib.pack.Boom" {
		t.Fatalf("makeChoice returned payload metadata = %#v, want synthetic throwing closure target", payload)
	}
	var callerFunc CheckedFunc
	for _, fn := range checked.Funcs {
		if fn.Name == "app.main.caller" {
			callerFunc = fn
			break
		}
	}
	choice := callerFunc.Locals["choice"]
	payload, ok = choice.EnumPayloadFunctions["0:0"]
	if !ok {
		t.Fatalf("choice EnumPayloadFunctions = %#v, want 0:0", choice.EnumPayloadFunctions)
	}
	if payload.FunctionValue == "" || payload.FunctionParamName != "" || payload.FunctionThrowsType != "lib.pack.Boom" {
		t.Fatalf("choice returned payload metadata = %#v, want synthetic throwing closure target", payload)
	}
	bound := callerFunc.Locals["local"]
	if bound.FunctionValue == "" || bound.FunctionParamName != "" || bound.FunctionThrowsType != "lib.pack.Boom" {
		t.Fatalf("pattern local metadata = %#v, want synthetic throwing closure target", bound)
	}
}
