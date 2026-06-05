package compiler_test

import (
	"strings"
	"testing"

	compiler "tetra_language/compiler"
)

func TestGenerateInterfaceFromSourceWritesT4IStubs(t *testing.T) {
	src := []byte(`module math.core

struct Point:
    x: Int
    y: Int

func add(a: Int, b: Int) -> Int:
    return a + b

func enabled() -> Bool:
    return true
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"module math.core",
		"struct Point:",
		"func add(a: i32, b: i32) -> i32:",
		"    return 0",
		"func enabled() -> bool:",
		"    return false",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
}

func TestGenerateInterfaceFromSourcePreservesFunctionTypedParameterReturnStub(t *testing.T) {
	src := []byte(`module lib.identity

pub func identity(f: fn(Int) -> Int) -> fn(Int) -> Int:
    return f
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/identity.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"func identity(f: fn(i32) -> i32) -> fn(i32) -> i32:",
		"    return f",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "return 0") {
		t.Fatalf("function-typed parameter-return interface stub fell back to return 0:\n%s", text)
	}
}

func TestGenerateInterfaceFromSourcePreservesBorrowedReturnContract(t *testing.T) {
	src := []byte(`module lib.views

pub func view(xs: borrow []u8) -> borrow []u8:
    return xs.window(0, 1).borrow()
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/views.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"func view(xs: borrow []u8) -> borrow []u8:",
		"// tetra-interface-lifetime: return=borrow source=xs provenance=param lifetime=call",
		"    return xs",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}

	iface, err := compiler.ParseFile(out, "lib/views.t4i")
	if err != nil {
		t.Fatalf("ParseFile generated interface: %v\n%s", err, out)
	}
	app, err := compiler.ParseFile([]byte(`module app.main
import lib.views as views

func relay(xs: borrow []u8) -> borrow []u8:
    return views.view(xs)

func main() -> Int:
    return 0
`), "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile app: %v", err)
	}
	checked, err := compiler.CheckWorld(&compiler.World{
		EntryModule:      "app.main",
		Files:            []*compiler.FileAST{iface, app},
		InterfaceModules: map[string]bool{"lib.views": true},
		ByModule: map[string]*compiler.FileAST{
			"lib.views": iface,
			"app.main":  app,
		},
	})
	if err != nil {
		t.Fatalf("CheckWorld: %v\ninterface:\n%s", err, out)
	}
	if got := checked.FuncSigs["lib.views.view"].ReturnOwnership; got != "borrow" {
		t.Fatalf("imported view ReturnOwnership = %q, want borrow; interface:\n%s", got, out)
	}
}

func TestInterfaceFingerprintTracksBorrowedReturnLifetimeSource(t *testing.T) {
	srcA := []byte(`module lib.views

pub func choose(a: borrow []u8, b: borrow []u8) -> borrow []u8:
    return a.borrow()
`)
	srcB := []byte(strings.Replace(string(srcA), "return a.borrow()", "return b.borrow()", 1))

	hashA, err := compiler.InterfaceFingerprintFromSource(srcA, "lib/views.t4")
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource A: %v", err)
	}
	hashB, err := compiler.InterfaceFingerprintFromSource(srcB, "lib/views.t4")
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource B: %v", err)
	}
	if hashA == hashB {
		t.Fatalf("borrowed return lifetime source did not affect interface hash: %s", hashA)
	}
}

func TestInterfaceFingerprintRejectsTamperedBorrowedReturnLifetimeMetadata(t *testing.T) {
	src := []byte(`module lib.views

pub func view(xs: borrow []u8) -> borrow []u8:
    return xs.borrow()
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "lib/views.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	tampered := strings.Replace(string(iface), "source=xs", "source=ys", 1)
	if tampered == string(iface) {
		t.Fatalf("test fixture did not find borrowed return lifetime metadata:\n%s", iface)
	}
	_, err = compiler.InterfaceFingerprintFromT4I([]byte(tampered))
	if err == nil || !strings.Contains(err.Error(), "invalid .t4i hash") {
		t.Fatalf("InterfaceFingerprintFromT4I tampered error = %v, want invalid .t4i hash", err)
	}
}

func TestGenerateInterfaceFromSourcePreservesReturnedFunctionHandleMetadata(t *testing.T) {
	src := []byte(`module lib.maker

pub func make() -> fn(Int) -> Int:
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
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/maker.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	maker, err := compiler.ParseFile(out, "lib/maker.t4i")
	if err != nil {
		t.Fatalf("ParseFile maker interface: %v\n%s", err, out)
	}
	app, err := compiler.ParseFile([]byte(`module app.main
import lib.maker as maker

func main() -> Int:
    let cb: fn(Int) -> Int = maker.make()
    return cb(-3)
`), "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile app: %v", err)
	}
	checked, err := compiler.CheckWorld(&compiler.World{
		EntryModule:      "app.main",
		Files:            []*compiler.FileAST{maker, app},
		InterfaceModules: map[string]bool{"lib.maker": true},
		ByModule: map[string]*compiler.FileAST{
			"lib.maker": maker,
			"app.main":  app,
		},
	})
	if err != nil {
		t.Fatalf("CheckWorld: %v\ninterface:\n%s", err, out)
	}
	makeSig := checked.FuncSigs["lib.maker.make"]
	if makeSig.ReturnFunctionSymbol == "" {
		t.Fatalf("make ReturnFunctionSymbol empty; interface:\n%s", out)
	}
	if got := len(makeSig.ReturnFunctionCaptures); got != 9 {
		t.Fatalf("make ReturnFunctionCaptures = %d, want 9; sig=%#v\ninterface:\n%s", got, makeSig, out)
	}
	if string(makeSig.ReturnFunctionEscapeKind) != "heap" || !makeSig.ReturnFunctionHandleValue || makeSig.ReturnSlots != 4 {
		t.Fatalf("make returned handle metadata = (%q, %v, slots=%d), want (heap, true, 4); sig=%#v\ninterface:\n%s", makeSig.ReturnFunctionEscapeKind, makeSig.ReturnFunctionHandleValue, makeSig.ReturnSlots, makeSig, out)
	}
	foundMain := false
	for _, fn := range checked.Funcs {
		if fn.Name == "app.main.main" {
			foundMain = true
			cb := fn.Locals["cb"]
			captureCount := len(cb.FunctionCaptures) + len(cb.FunctionEscapeCaptures)
			if captureCount != 9 || string(cb.FunctionEscapeKind) != "heap" || !cb.FunctionHandleValue || cb.SlotCount != 4 {
				t.Fatalf("local cb metadata = captures:%d direct:%d escape-captures:%d escape:%q handle:%v slots:%d; want 9/heap/true/4\ninterface:\n%s", captureCount, len(cb.FunctionCaptures), len(cb.FunctionEscapeCaptures), cb.FunctionEscapeKind, cb.FunctionHandleValue, cb.SlotCount, out)
			}
			break
		}
	}
	if !foundMain {
		t.Fatalf("checked funcs missing app.main.main")
	}
}

func TestGenerateInterfaceFromSourcePreservesFunctionTypedStructFieldReturnStub(t *testing.T) {
	src := []byte(`module lib.callbacks

pub struct Holder:
    cb: fn(Int) -> Int

pub func pick(holder: Holder) -> fn(Int) -> Int:
    return holder.cb
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"struct Holder:",
		"    cb: fn(i32) -> i32",
		"func pick(holder: Holder) -> fn(i32) -> i32:",
		"    return holder.cb",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "return fn(") || strings.Contains(text, "return 0") {
		t.Fatalf("function-typed struct-field-return interface stub lost field return metadata:\n%s", text)
	}
}

func TestGenerateInterfaceFromSourcePreservesFunctionTypedNestedStructFieldReturnStub(t *testing.T) {
	src := []byte(`module lib.callbacks

pub struct Holder:
    cb: fn(Int) -> Int

pub struct Box:
    holder: Holder

pub func pick(box: Box) -> fn(Int) -> Int:
    return box.holder.cb
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"struct Holder:",
		"    cb: fn(i32) -> i32",
		"struct Box:",
		"    holder: Holder",
		"func pick(box: Box) -> fn(i32) -> i32:",
		"    return box.holder.cb",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "return fn(") || strings.Contains(text, "return 0") {
		t.Fatalf("function-typed nested-struct-field-return interface stub lost field return metadata:\n%s", text)
	}
}

func TestGenerateInterfaceFromSourcePreservesFunctionTypedStructParameterWholeReturnStub(t *testing.T) {
	src := []byte(`module lib.callbacks

pub struct Holder:
    cb: fn(Int) -> Int

pub struct Box:
    holder: Holder

pub func echo(box: Box) -> Box:
    return box
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"struct Holder:",
		"    cb: fn(i32) -> i32",
		"struct Box:",
		"    holder: Holder",
		"func echo(box: Box) -> Box:",
		"    return box",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "return 0") {
		t.Fatalf("function-typed struct-parameter whole-return interface stub lost parameter return metadata:\n%s", text)
	}
}

func TestGenerateInterfaceFromSourcePreservesFunctionTypedEnumParameterWholeReturnStub(t *testing.T) {
	src := []byte(`module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func echo(choice: MaybeCallback) -> MaybeCallback:
    return choice
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"enum MaybeCallback:",
		"    case some(fn(i32) -> i32)",
		"    case empty",
		"func echo(choice: MaybeCallback) -> MaybeCallback:",
		"    return choice",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "return 0") {
		t.Fatalf("function-typed enum-parameter whole-return interface stub lost parameter return metadata:\n%s", text)
	}
}

func TestGenerateInterfaceFromSourcePreservesReturnedAggregateClosureStub(t *testing.T) {
	src := []byte(`module lib.callbacks

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
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"enum MaybeCallback:",
		"    case some(fn(i32) -> i32)",
		"struct Box:",
		"    choice: MaybeCallback",
		"func makeBox() -> Box:",
		"    return Box(choice: MaybeCallback.some(fn(p0: i32) -> i32 = 0))",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "    return 0") {
		t.Fatalf("returned aggregate interface stub lost closure payload metadata:\n%s", text)
	}
}

func TestGenerateInterfaceFromSourcePreservesReturnedEnumClosureStub(t *testing.T) {
	src := []byte(`module lib.callbacks

pub enum MaybeCallback:
    case some(fn(Int) -> Int)
    case empty

pub func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int:
        return x + base
    )
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"enum MaybeCallback:",
		"    case some(fn(i32) -> i32)",
		"func makeChoice() -> MaybeCallback:",
		"    return MaybeCallback.some(fn(p0: i32) -> i32 = 0)",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "    return 0") {
		t.Fatalf("returned enum interface stub lost closure payload metadata:\n%s", text)
	}
}

func TestGenerateInterfaceFromSourcePreservesReturnedThrowingAggregateClosureStub(t *testing.T) {
	src := []byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub struct Box:
    choice: MaybeCallback

pub func makeBox() -> Box:
    let base: Int = 1
    return Box(choice: MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    ))
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"enum Boom:",
		"enum MaybeCallback:",
		"    case some(fn(i32) -> i32 throws Boom)",
		"struct Box:",
		"    choice: MaybeCallback",
		"func makeBox() -> Box:",
		"    return Box(choice: MaybeCallback.some(fn(p0: i32) -> i32 throws Boom = 0))",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "    return 0") {
		t.Fatalf("returned throwing aggregate interface stub lost closure payload metadata:\n%s", text)
	}
}

func TestGenerateInterfaceFromSourcePreservesReturnedThrowingStructFieldClosureStub(t *testing.T) {
	src := []byte(`module lib.callbacks

pub enum Boom:
    case bad

pub struct Holder:
    cb: fn(Int) -> Int throws Boom

pub func makeHolder() -> Holder:
    let base: Int = 1
    return Holder(cb: fn(x: Int) -> Int throws Boom:
        return x + base
    )
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"enum Boom:",
		"struct Holder:",
		"    cb: fn(i32) -> i32 throws Boom",
		"func makeHolder() -> Holder:",
		"    return Holder(cb: fn(p0: i32) -> i32 throws Boom = 0)",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "    return 0") {
		t.Fatalf("returned throwing struct-field interface stub lost closure field metadata:\n%s", text)
	}
}

func TestGenerateInterfaceFromSourcePreservesReturnedThrowingEnumClosureStub(t *testing.T) {
	src := []byte(`module lib.callbacks

pub enum Boom:
    case bad

pub enum MaybeCallback:
    case some(fn(Int) -> Int throws Boom)
    case empty

pub func makeChoice() -> MaybeCallback:
    let base: Int = 1
    return MaybeCallback.some(fn(x: Int) -> Int throws Boom:
        return x + base
    )
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "lib/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"enum Boom:",
		"enum MaybeCallback:",
		"    case some(fn(i32) -> i32 throws Boom)",
		"func makeChoice() -> MaybeCallback:",
		"    return MaybeCallback.some(fn(p0: i32) -> i32 throws Boom = 0)",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "    return 0") {
		t.Fatalf("returned throwing enum interface stub lost closure payload metadata:\n%s", text)
	}
}

func TestGenerateInterfaceFromSourceFiltersPrivateSurfaceAndHashesPublicAPI(t *testing.T) {
	src := []byte(`module math.core

import hidden.impl as impl
pub import public.types.{Vec}

pub struct Point:
    x: Int
    y: Int

struct Secret:
    value: Int

pub func add(a: Int, b: Int) -> Int:
    return a + b

func hidden() -> Int:
    return 99
`)
	out, err := compiler.GenerateInterfaceFromSource(src, "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(out)
	for _, want := range []string{
		"// t4i-hash: sha256:",
		"pub import public.types.{Vec}",
		"pub struct Point:",
		"pub func add(a: i32, b: i32) -> i32:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	for _, leak := range []string{"hidden.impl", "struct Secret", "func hidden"} {
		if strings.Contains(text, leak) {
			t.Fatalf("interface leaked %q:\n%s", leak, text)
		}
	}

	out2, err := compiler.GenerateInterfaceFromSource([]byte(strings.Replace(string(src), "return 99", "return 100", 1)), "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource second: %v", err)
	}
	if string(out2) != text {
		t.Fatalf("private body-only change should not change interface hash\nbefore:\n%s\nafter:\n%s", text, out2)
	}
}

func TestInterfaceFingerprintFromSourceIsPublicAPIStable(t *testing.T) {
	src := []byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b

func hidden() -> Int:
    return 1
`)
	hash1, err := compiler.InterfaceFingerprintFromSource(src, "math/core.t4")
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource: %v", err)
	}
	privateBodyChanged := []byte(strings.Replace(string(src), "return 1", "return 2", 1))
	hash2, err := compiler.InterfaceFingerprintFromSource(privateBodyChanged, "math/core.t4")
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource private change: %v", err)
	}
	if hash1 != hash2 {
		t.Fatalf("private implementation change changed public API hash: %s vs %s", hash1, hash2)
	}
	publicSigChanged := []byte(strings.Replace(string(src), "b: Int", "b: Bool", 1))
	hash3, err := compiler.InterfaceFingerprintFromSource(publicSigChanged, "math/core.t4")
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource public change: %v", err)
	}
	if hash1 == hash3 {
		t.Fatalf("public signature change did not change API hash: %s", hash1)
	}
}

func TestValidateInterfaceAgainstSourceReportsPublicAPIMismatch(t *testing.T) {
	src := []byte(`module math.core

pub func add(a: Int, b: Int) -> Int:
    return a + b
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	changedSource := []byte(strings.Replace(string(src), "b: Int", "b: Bool", 1))
	err = compiler.ValidateInterfaceAgainstSource(changedSource, iface, "math/core.t4")
	if err == nil {
		t.Fatalf("expected public API mismatch")
	}
	if !strings.Contains(err.Error(), "public API mismatch") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenerateInterfaceFromSourceKeepsImportsRequiredByPublicAPI(t *testing.T) {
	src := []byte(`module math.core

import math.types as mt
import hidden.impl as hidden

pub func norm(v: mt.Vec) -> Int:
    return v.x

func private_helper(v: hidden.Secret) -> Int:
    return 0
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "math/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(iface)
	if !strings.Contains(text, "import math.types as mt") {
		t.Fatalf("interface omitted public-signature import:\n%s", text)
	}
	if strings.Contains(text, "hidden.impl") {
		t.Fatalf("interface leaked private-only import:\n%s", text)
	}
}

func TestGenerateInterfaceFromSourceEmitsTypecheckableExtensionDeclarations(t *testing.T) {
	src := []byte(`module engine.vec

pub struct Vec2:
    x: Int
    y: Int

pub extension Vec2:
    func sum(self: Vec2) -> Int:
        return self.x + self.y
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "engine/vec.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(iface)
	for _, want := range []string{
		"pub extension Vec2:",
		"func sum(self: Vec2) -> i32:",
		"    return 0",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}

	ifaceFile, err := compiler.ParseFile(iface, "engine/vec.t4i")
	if err != nil {
		t.Fatalf("ParseFile interface: %v\n%s", err, text)
	}
	ifaceFile.InterfaceHash, err = compiler.InterfaceFingerprintFromT4I(iface)
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromT4I: %v", err)
	}
	app, err := compiler.ParseFile([]byte(`module app.main
import engine.vec as vec

func main() -> Int:
    let v: vec.Vec2 = vec.Vec2(x: 40, y: 2)
    return vec.Vec2.sum(v)
`), "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile app: %v", err)
	}
	world := &compiler.World{
		EntryModule:      app.Module,
		Files:            []*compiler.FileAST{ifaceFile, app},
		ByModule:         map[string]*compiler.FileAST{ifaceFile.Module: ifaceFile, app.Module: app},
		InterfaceModules: map[string]bool{ifaceFile.Module: true},
		InterfaceHashes:  map[string]string{ifaceFile.Module: ifaceFile.InterfaceHash},
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld generated interface extension: %v\ninterface:\n%s", err, text)
	}
	if _, ok := checked.FuncSigs["engine.vec.Vec2.sum"]; !ok {
		t.Fatalf("missing extension method signature from generated interface: %#v\ninterface:\n%s", checked.FuncSigs, text)
	}
}

func TestGenerateInterfaceFromSourceEmitsProtocolImplDeclarationsBeforeFunctions(t *testing.T) {
	src := []byte(`module engine.core

pub struct Vec2:
    x: Int

pub protocol Echoable:
    func echo(self: Vec2) -> Vec2

pub extension Vec2:
    func echo(self: Vec2) -> Vec2:
        return self

impl Vec2: Echoable

pub func id<T: Echoable>(x: T) -> T:
    return x
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "engine/core.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(iface)
	for _, want := range []string{
		"pub extension Vec2:",
		"impl Vec2: Echoable",
		"pub func id<T: Echoable>(x: T) -> T:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing %q:\n%s", want, text)
		}
	}
	if strings.Index(text, "impl Vec2: Echoable") > strings.Index(text, "pub func id<T: Echoable>") {
		t.Fatalf("interface emitted impl after functions:\n%s", text)
	}

	ifaceFile, err := compiler.ParseFile(iface, "engine/core.t4i")
	if err != nil {
		t.Fatalf("ParseFile interface: %v\n%s", err, text)
	}
	ifaceFile.InterfaceHash, err = compiler.InterfaceFingerprintFromT4I(iface)
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromT4I: %v", err)
	}
	app, err := compiler.ParseFile([]byte(`module app.main
import engine.core as core

func main() -> Int:
    let v: core.Vec2 = core.Vec2(x: 42)
    let out: core.Vec2 = core.id(v)
    return out.x
`), "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile app: %v", err)
	}
	world := &compiler.World{
		EntryModule:      app.Module,
		Files:            []*compiler.FileAST{ifaceFile, app},
		ByModule:         map[string]*compiler.FileAST{ifaceFile.Module: ifaceFile, app.Module: app},
		InterfaceModules: map[string]bool{ifaceFile.Module: true},
		InterfaceHashes:  map[string]string{ifaceFile.Module: ifaceFile.InterfaceHash},
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld generated interface impl: %v\ninterface:\n%s", err, text)
	}
	if _, ok := checked.FuncSigs["engine.core.id__T_engine_2e_core_2e_Vec2"]; !ok {
		t.Fatalf("missing protocol-bound monomorphized signature from generated interface: %#v\ninterface:\n%s", checked.FuncSigs, text)
	}
}

func TestGenerateInterfaceFromSourceKeepsImportsRequiredOnlyByImpls(t *testing.T) {
	src := []byte(`module app.impls

import engine.core as core

impl core.Vec2: core.Renderable

pub func marker() -> Int:
    return 1
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "app/impls.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(iface)
	for _, want := range []string{
		"import engine.core as core",
		"impl core.Vec2: core.Renderable",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing impl-only import surface %q:\n%s", want, text)
		}
	}
	if _, err := compiler.ParseFile(iface, "app/impls.t4i"); err != nil {
		t.Fatalf("ParseFile generated interface: %v\n%s", err, text)
	}
}

func TestGenerateInterfaceFromSourceKeepsImportsRequiredOnlyByGenericBounds(t *testing.T) {
	src := []byte(`module app.generics

import engine.core as core

pub func id<T: core.Echoable>(x: T) -> T:
    return x
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "app/generics.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(iface)
	for _, want := range []string{
		"import engine.core as core",
		"pub func id<T: core.Echoable>(x: T) -> T:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing generic-bound import surface %q:\n%s", want, text)
		}
	}
	if _, err := compiler.ParseFile(iface, "app/generics.t4i"); err != nil {
		t.Fatalf("ParseFile generated interface: %v\n%s", err, text)
	}
}

func TestGenerateInterfaceFromSourcePreservesGenericStructTypeArgsAndImports(t *testing.T) {
	src := []byte(`module app.boxes

import engine.core as core

pub struct Box<T>:
    value: T

pub func wrap(box: Box<core.Vec2>) -> Box<core.Vec2>:
    return box
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "app/boxes.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(iface)
	for _, want := range []string{
		"import engine.core as core",
		"pub struct Box<T>:",
		"pub func wrap(box: Box<core.Vec2>) -> Box<core.Vec2>:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing generic type-arg surface %q:\n%s", want, text)
		}
	}
	if _, err := compiler.ParseFile(iface, "app/boxes.t4i"); err != nil {
		t.Fatalf("ParseFile generated interface: %v\n%s", err, text)
	}
}

func TestGeneratedInterfaceGenericStructTypeArgsCheckAndLowerAcrossModules(t *testing.T) {
	core, err := compiler.ParseFile([]byte(`module engine.core

pub struct Vec2:
    x: Int
`), "engine/core.t4")
	if err != nil {
		t.Fatalf("ParseFile core: %v", err)
	}
	src := []byte(`module app.boxes

import engine.core as core

pub struct Box<T>:
    value: T

pub func wrap(box: Box<core.Vec2>) -> Box<core.Vec2>:
    return box
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "app/boxes.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(iface)
	boxes, err := compiler.ParseFile(iface, "app/boxes.t4i")
	if err != nil {
		t.Fatalf("ParseFile generated interface: %v\n%s", err, text)
	}
	boxes.InterfaceHash, err = compiler.InterfaceFingerprintFromT4I(iface)
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromT4I: %v", err)
	}
	app, err := compiler.ParseFile([]byte(`module app.main

import app.boxes as boxes
import engine.core as core

func main() -> Int:
    let v: boxes.Box<core.Vec2> = boxes.wrap(boxes.Box<core.Vec2>{value: core.Vec2{x: 42}})
    return v.value.x
`), "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile app: %v", err)
	}
	world := &compiler.World{
		EntryModule:      app.Module,
		Files:            []*compiler.FileAST{core, boxes, app},
		ByModule:         map[string]*compiler.FileAST{core.Module: core, boxes.Module: boxes, app.Module: app},
		InterfaceModules: map[string]bool{boxes.Module: true},
		InterfaceHashes:  map[string]string{boxes.Module: boxes.InterfaceHash},
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld generated interface generic struct: %v\ninterface:\n%s", err, text)
	}
	if _, ok := checked.FuncSigs["app.boxes.wrap"]; !ok {
		t.Fatalf("missing generated interface function signature: %#v\ninterface:\n%s", checked.FuncSigs, text)
	}
	if _, err := compiler.LowerModules(checked); err != nil {
		t.Fatalf("LowerModules generated interface generic struct: %v\ninterface:\n%s", err, text)
	}
}

func TestGenerateInterfaceFromSourceKeepsImportsRequiredByFunctionTypeRefs(t *testing.T) {
	src := []byte(`module app.callbacks

import engine.core as core

pub func install(cb: fn(core.Vec2) -> core.Vec2 throws core.Boom) -> fn(core.Vec2) -> core.Vec2 throws core.Boom:
    return cb
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "app/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(iface)
	for _, want := range []string{
		"import engine.core as core",
		"pub func install(cb: fn(core.Vec2) -> core.Vec2 throws core.Boom) -> fn(core.Vec2) -> core.Vec2 throws core.Boom:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("interface missing function-type import surface %q:\n%s", want, text)
		}
	}
	if _, err := compiler.ParseFile(iface, "app/callbacks.t4i"); err != nil {
		t.Fatalf("ParseFile generated interface: %v\n%s", err, text)
	}
}

func TestGeneratedInterfaceFunctionTypeRefsCheckAndLowerAcrossModules(t *testing.T) {
	core, err := compiler.ParseFile([]byte(`module engine.core

pub struct Vec2:
    x: Int
`), "engine/core.t4")
	if err != nil {
		t.Fatalf("ParseFile core: %v", err)
	}
	src := []byte(`module app.callbacks

import engine.core as core

pub func install(cb: fn(core.Vec2) -> core.Vec2) -> fn(core.Vec2) -> core.Vec2:
    return cb
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "app/callbacks.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(iface)
	callbacks, err := compiler.ParseFile(iface, "app/callbacks.t4i")
	if err != nil {
		t.Fatalf("ParseFile generated interface: %v\n%s", err, text)
	}
	callbacks.InterfaceHash, err = compiler.InterfaceFingerprintFromT4I(iface)
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromT4I: %v", err)
	}
	app, err := compiler.ParseFile([]byte(`module app.main

import app.callbacks as callbacks
import engine.core as core

func echo(v: core.Vec2) -> core.Vec2:
    return v

func main() -> Int:
    let cb: fn(core.Vec2) -> core.Vec2 = callbacks.install(echo)
    let out: core.Vec2 = cb(core.Vec2{x: 42})
    return out.x
`), "app/main.t4")
	if err != nil {
		t.Fatalf("ParseFile app: %v", err)
	}
	world := &compiler.World{
		EntryModule:      app.Module,
		Files:            []*compiler.FileAST{core, callbacks, app},
		ByModule:         map[string]*compiler.FileAST{core.Module: core, callbacks.Module: callbacks, app.Module: app},
		InterfaceModules: map[string]bool{callbacks.Module: true},
		InterfaceHashes:  map[string]string{callbacks.Module: callbacks.InterfaceHash},
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld generated interface function type refs: %v\ninterface:\n%s", err, text)
	}
	if _, err := compiler.LowerModules(checked); err != nil {
		t.Fatalf("LowerModules generated interface function type refs: %v\ninterface:\n%s", err, text)
	}
}

func TestGenerateInterfaceFromSourcePreservesProtocolRequirementTypeParams(t *testing.T) {
	src := []byte(`module app.protocols

pub protocol Mapper:
    func map<T>(self: Int, value: T) -> T
`)
	iface, err := compiler.GenerateInterfaceFromSource(src, "app/protocols.t4")
	if err != nil {
		t.Fatalf("GenerateInterfaceFromSource: %v", err)
	}
	text := string(iface)
	want := "func map<T>(self: i32, value: T) -> T"
	if !strings.Contains(text, want) {
		t.Fatalf("interface missing protocol requirement type params %q:\n%s", want, text)
	}
	if _, err := compiler.ParseFile(iface, "app/protocols.t4i"); err != nil {
		t.Fatalf("ParseFile generated interface: %v\n%s", err, text)
	}
}

func TestInterfaceFingerprintFromSourceTracksHashOnlyPublicSurface(t *testing.T) {
	src := []byte(`module app.config

pub const build: Int = 1
`)
	hash1, err := compiler.InterfaceFingerprintFromSource(src, "app/config.t4")
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource: %v", err)
	}
	hash2, err := compiler.InterfaceFingerprintFromSource([]byte(strings.Replace(string(src), "build: Int", "build: Bool", 1)), "app/config.t4")
	if err != nil {
		t.Fatalf("InterfaceFingerprintFromSource changed: %v", err)
	}
	if hash1 == hash2 {
		t.Fatalf("public hash-only global surface change did not change API hash: %s", hash1)
	}
}
