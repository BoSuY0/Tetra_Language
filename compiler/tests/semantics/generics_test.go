package compiler_test

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/opt"
	"tetra_language/compiler/internal/testkit"
)

func TestGenericFunctionParseCheckAndDocs(t *testing.T) {
	src := []byte(`
func id<T>(x: T) -> T:
    return x

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := prog.Funcs[0].TypeParams; len(got) != 1 || got[0] != "T" {
		t.Fatalf("type params = %#v", got)
	}
	if _, err := compiler.Check(prog); err != nil {
		t.Fatalf("Check: %v", err)
	}
	docs, err := compiler.GenerateAPIDocsFromSource(src, "generics.tetra")
	if err != nil {
		t.Fatalf("GenerateAPIDocsFromSource: %v", err)
	}
	if !strings.Contains(string(docs), "`func id<T>(x: T) -> T`") {
		t.Fatalf("docs = %s", string(docs))
	}
}

func TestGenericFunctionMonomorphizedCall(t *testing.T) {
	src := []byte(`
func id<T>(x: T) -> T:
    return x

func main() -> Int:
    return id(42)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	sig, ok := checked.FuncSigs["id__T_i32"]
	if !ok {
		t.Fatalf("missing monomorphized signature: %#v", checked.FuncSigs)
	}
	if sig.Generic {
		t.Fatalf("id__T_i32 should be concrete after monomorphization: %#v", sig)
	}
	if sig.ParamSlots != 1 || sig.ReturnSlots != 1 || sig.ReturnType != "i32" {
		t.Fatalf("id__T_i32 ABI = params %d returns %d type %q, want params 1 returns 1 type i32", sig.ParamSlots, sig.ReturnSlots, sig.ReturnType)
	}
	irProg, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	idFn := findIRFunc(t, irProg.Funcs, "id__T_i32")
	if idFn.ParamSlots != 1 || idFn.ReturnSlots != 1 {
		t.Fatalf("lowered id__T_i32 ABI = params %d returns %d, want params 1 returns 1", idFn.ParamSlots, idFn.ReturnSlots)
	}
	mainFn := findIRFunc(t, irProg.Funcs, "main")
	if !hasIRCall(mainFn, "id__T_i32") {
		t.Fatalf("main did not call monomorphized id__T_i32: %#v", mainFn.Instrs)
	}
}

func TestP9GenericIdentityDisappearsAfterSmallPureInlining(t *testing.T) {
	src := []byte(`
func id<T>(x: T) -> T:
    return x

func main() -> Int:
    return id(42)
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
	before := findIRFunc(t, irProg.Funcs, "main")
	if !hasIRCall(before, "id__T_i32") {
		t.Fatalf("pre-optimization main did not call id__T_i32: %#v", before.Instrs)
	}
	if _, err := opt.NewManager().Run(irProg, opt.InlineSmallPurePass()); err != nil {
		t.Fatalf("InlineSmallPurePass: %v", err)
	}
	after := findIRFunc(t, irProg.Funcs, "main")
	if hasIRCall(after, "id__T_i32") {
		t.Fatalf("generic identity call survived specialization/inlining: %#v", after.Instrs)
	}
}

func TestP17GenericWrapperDisappearsAfterSmallPureInlining(t *testing.T) {
	src := []byte(`
func id<T>(x: T) -> T:
    return x

func wrap<T>(x: T) -> T:
    return id(x)

func main() -> Int:
    return wrap(42)
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
	before := findIRFunc(t, irProg.Funcs, "main")
	if !hasIRCall(before, "wrap__T_i32") {
		t.Fatalf("pre-optimization main did not call wrap__T_i32: %#v", before.Instrs)
	}
	report, err := opt.NewManager().Run(irProg, opt.InlineSmallPurePass())
	if err != nil {
		t.Fatalf("InlineSmallPurePass: %v", err)
	}
	row := report.Passes[0]
	if !hasOptDecision(row.Decisions, "inlined", "main", "wrap__T_i32", "small_pure_wrapper") {
		t.Fatalf("missing wrapper inline decision in %#v", row.Decisions)
	}
	if !hasOptDecision(row.Decisions, "inlined", "main", "id__T_i32", "small_pure") {
		t.Fatalf("missing nested identity inline decision in %#v", row.Decisions)
	}
	after := findIRFunc(t, irProg.Funcs, "main")
	if hasIRCall(after, "wrap__T_i32") || hasIRCall(after, "id__T_i32") {
		t.Fatalf("generic wrapper call survived specialization/inlining: %#v", after.Instrs)
	}
}

func TestGenericFunctionProtocolBoundConformancePasses(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Echoable:
    func echo(self: Vec2) -> Vec2

extension Vec2:
    func echo(self: Vec2) -> Vec2:
        return self

impl Vec2: Echoable

func id<T: Echoable>(x: T) -> T:
    return x

func main() -> Int:
    let v: Vec2 = Vec2(x: 42)
    let out: Vec2 = id(v)
    return out.x
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	sig, ok := checked.FuncSigs["id__T_Vec2"]
	if !ok {
		t.Fatalf("missing protocol-bound monomorphized signature: %#v", checked.FuncSigs)
	}
	if sig.Generic {
		t.Fatalf("id__T_Vec2 should be concrete after monomorphization: %#v", sig)
	}
	if sig.ParamSlots != 1 || sig.ReturnSlots != 1 || sig.ReturnType != "Vec2" {
		t.Fatalf("id__T_Vec2 ABI = params %d returns %d type %q, want params 1 returns 1 type Vec2", sig.ParamSlots, sig.ReturnSlots, sig.ReturnType)
	}
	irProg, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	idFn := findIRFunc(t, irProg.Funcs, "id__T_Vec2")
	if idFn.ParamSlots != 1 || idFn.ReturnSlots != 1 {
		t.Fatalf("lowered id__T_Vec2 ABI = params %d returns %d, want params 1 returns 1", idFn.ParamSlots, idFn.ReturnSlots)
	}
	mainFn := findIRFunc(t, irProg.Funcs, "main")
	if !hasIRCall(mainFn, "id__T_Vec2") {
		t.Fatalf("main did not call protocol-bound id__T_Vec2: %#v", mainFn.Instrs)
	}
}

func hasOptDecision(decisions []opt.PassDecision, action string, caller string, callee string, reason string) bool {
	for _, decision := range decisions {
		if decision.Action == action && decision.Caller == caller && decision.Callee == callee && decision.Reason == reason {
			return true
		}
	}
	return false
}

func TestGenericFunctionProtocolBoundRejectsMissingImpl(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Echoable:
    func echo(self: Vec2) -> Vec2

func id<T: Echoable>(x: T) -> T:
    return x

func main() -> Int:
    let v: Vec2 = Vec2(x: 42)
    let out: Vec2 = id(v)
    return out.x
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected protocol-bound conformance diagnostic")
	}
	if !strings.Contains(err.Error(), "generic argument 'Vec2' does not satisfy bound 'Echoable' for 'T'") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericFunctionProtocolBoundRejectsMismatchedImplSignature(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Echoable:
    func echo(self: Vec2) -> Vec2

extension Vec2:
    func echo(self: Vec2) -> Int:
        return self.x

impl Vec2: Echoable

func id<T: Echoable>(x: T) -> T:
    return x

func main() -> Int:
    let v: Vec2 = Vec2(x: 42)
    let out: Vec2 = id(v)
    return out.x
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected protocol-bound conformance diagnostic")
	}
	if !strings.Contains(err.Error(), "return type differs") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericFunctionProtocolBoundCrossModuleConformancePasses(t *testing.T) {
	files := map[string]string{
		"engine/core.tetra": `module engine.core
pub struct Vec2:
    x: Int

pub protocol Echoable:
    func echo(self: Vec2) -> Vec2

extension Vec2:
    func echo(self: Vec2) -> Vec2:
        return self

impl Vec2: Echoable

pub func id<T: Echoable>(x: T) -> T:
    return x
`,
		"app/main.tetra": `module app.main
import engine.core as core

func main() -> Int:
    let v: core.Vec2 = core.Vec2(x: 42)
    let out: core.Vec2 = core.id(v)
    return out.x
`,
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))

	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if _, ok := checked.FuncSigs["engine.core.id__T_engine_2e_core_2e_Vec2"]; !ok {
		t.Fatalf("missing cross-module protocol-bound monomorphized signature: %#v", checked.FuncSigs)
	}
	if _, err := compiler.LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
}

func TestGenericFunctionProtocolBoundCrossModuleRejectsMissingImpl(t *testing.T) {
	files := map[string]string{
		"engine/core.tetra": `module engine.core
pub struct Vec2:
    x: Int

pub protocol Echoable:
    func echo(self: Vec2) -> Vec2

pub func id<T: Echoable>(x: T) -> T:
    return x
`,
		"app/main.tetra": `module app.main
import engine.core as core

func main() -> Int:
    let v: core.Vec2 = core.Vec2(x: 42)
    let out: core.Vec2 = core.id(v)
    return out.x
`,
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))

	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected cross-module protocol-bound conformance diagnostic")
	}
	if !strings.Contains(err.Error(), "generic argument 'Vec2' does not satisfy bound 'Echoable' for 'T'") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericFunctionProtocolBoundRejectsUnknownProtocolBound(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

func id<T: MissingProtocol>(x: T) -> T:
    return x

func main() -> Int:
    let v: Vec2 = Vec2(x: 42)
    let out: Vec2 = id(v)
    return out.x
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected unknown protocol bound diagnostic")
	}
	if !strings.Contains(err.Error(), "unknown protocol bound 'MissingProtocol' for generic parameter 'T'") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericFunctionProtocolBoundRejectsNonProtocolBound(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

func id<T: Vec2>(x: T) -> T:
    return x

func main() -> Int:
    let v: Vec2 = Vec2(x: 42)
    let out: Vec2 = id(v)
    return out.x
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected non-protocol bound diagnostic")
	}
	if !strings.Contains(err.Error(), "generic bound 'Vec2' for 'T' must name a protocol") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericFunctionProtocolBoundRejectsPrivateCrossModuleProtocolBound(t *testing.T) {
	files := map[string]string{
		"engine/core.tetra": `module engine.core
pub struct Vec2:
    x: Int

protocol HiddenEchoable:
    func echo(self: Vec2) -> Vec2

extension Vec2:
    func echo(self: Vec2) -> Vec2:
        return self

impl Vec2: HiddenEchoable

pub func id<T: HiddenEchoable>(x: T) -> T:
    return x
`,
		"app/main.tetra": `module app.main
import engine.core as core

func main() -> Int:
    let v: core.Vec2 = core.Vec2(x: 42)
    let out: core.Vec2 = core.id(v)
    return out.x
`,
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))

	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected private protocol bound visibility diagnostic")
	}
	if !strings.Contains(err.Error(), "private protocol 'engine.core.HiddenEchoable' is not visible from module 'app.main'") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericFunctionProtocolBoundRequirementCallUnsupported(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Echoable:
    func echo(self: Vec2) -> Vec2

extension Vec2:
    func echo(self: Vec2) -> Vec2:
        return self

impl Vec2: Echoable

func echoThroughBound<T: Echoable>(x: T) -> T:
    return T.echo(x)

func main() -> Int:
    let v: Vec2 = Vec2(x: 42)
    let out: Vec2 = echoThroughBound(v)
    return out.x
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected unsupported generic-bound requirement call diagnostic")
	}
	if !strings.Contains(err.Error(), "calling protocol requirement 'echo' through generic bound 'T' is not supported in this MVP") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericStructSameModuleMonomorphizedHappyPath(t *testing.T) {
	src := []byte(`
struct Box<T>:
    value: T

func main() -> Int:
    let b: Box<Int> = Box<Int>{value: 42}
    return b.value
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	box, ok := checked.Types["Box__T_i32"]
	if !ok {
		t.Fatalf("missing monomorphized struct type: %#v", checked.Types)
	}
	if box.SlotCount != 1 || len(box.Fields) != 1 || box.Fields[0].TypeName != "i32" || box.Fields[0].SlotCount != 1 {
		t.Fatalf("Box__T_i32 layout = %#v, want one i32 field", box)
	}
	if _, exists := checked.Types["Box"]; exists {
		t.Fatalf("generic struct template should not remain in checked types: %#v", checked.Types["Box"])
	}
	irProg, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	mainFn := findIRFunc(t, irProg.Funcs, "main")
	if mainFn.ReturnSlots != 1 {
		t.Fatalf("main ReturnSlots = %d, want 1", mainFn.ReturnSlots)
	}
}

func TestGenericFunctionReturningGenericStructMonomorphizesStruct(t *testing.T) {
	src := []byte(`
struct Box<T>:
    value: T

func make<T>(x: T) -> Box<T>:
    return Box<T>{value: x}

func main() -> Int:
    let b: Box<Int> = make(42)
    return b.value
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	box, ok := checked.Types["Box__T_i32"]
	if !ok {
		t.Fatalf("missing monomorphized struct type: %#v", checked.Types)
	}
	if box.SlotCount != 1 || len(box.Fields) != 1 || box.Fields[0].TypeName != "i32" {
		t.Fatalf("Box__T_i32 layout = %#v, want one i32 field", box)
	}
	sig, ok := checked.FuncSigs["make__T_i32"]
	if !ok {
		t.Fatalf("missing monomorphized function signature: %#v", checked.FuncSigs)
	}
	if sig.ReturnType != "Box__T_i32" {
		t.Fatalf("make__T_i32 return type = %q, want Box__T_i32", sig.ReturnType)
	}
	if sig.ParamSlots != 1 || sig.ReturnSlots != 1 {
		t.Fatalf("make__T_i32 ABI = params %d returns %d, want params 1 returns 1", sig.ParamSlots, sig.ReturnSlots)
	}
	irProg, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	makeFn := findIRFunc(t, irProg.Funcs, "make__T_i32")
	if makeFn.ParamSlots != 1 || makeFn.ReturnSlots != 1 {
		t.Fatalf("lowered make__T_i32 ABI = params %d returns %d, want params 1 returns 1", makeFn.ParamSlots, makeFn.ReturnSlots)
	}
	mainFn := findIRFunc(t, irProg.Funcs, "main")
	if !hasIRCall(mainFn, "make__T_i32") {
		t.Fatalf("main did not call monomorphized make__T_i32: %#v", mainFn.Instrs)
	}
}

func TestGenericFunctionInfersThroughGenericStructParameter(t *testing.T) {
	src := []byte(`
struct Box<T>:
    value: T

func get_or<T>(box: Box<T>, fallback: T) -> T:
    if fallback == 0:
        return box.value
    return fallback

func main() -> Int:
    let b: Box<Int> = Box<Int>{value: 42}
    return get_or(b, 0)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, ok := checked.Types["Box__T_i32"]; !ok {
		t.Fatalf("missing monomorphized Box type: %#v", checked.Types)
	}
	sig, ok := checked.FuncSigs["get_or__T_i32"]
	if !ok {
		t.Fatalf("missing generic-struct-parameter monomorphized function: %#v", checked.FuncSigs)
	}
	if sig.ReturnType != "i32" || sig.ParamSlots != 2 || sig.ReturnSlots != 1 {
		t.Fatalf("get_or__T_i32 signature = %#v, want concrete i32 return with two params", sig)
	}
	irProg, err := compiler.Lower(checked)
	if err != nil {
		t.Fatalf("Lower: %v", err)
	}
	mainFn := findIRFunc(t, irProg.Funcs, "main")
	if !hasIRCall(mainFn, "get_or__T_i32") {
		t.Fatalf("main did not call get_or__T_i32: %#v", mainFn.Instrs)
	}
}

func TestStableGenericCollectionSourceAPIMonomorphizesVecAndHashMap(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": `module app.main
import lib.core.collections as collections

func main() -> Int
uses alloc, mem:
    var nums: []i32 = core.make_i32(3)
    nums[0] = 7
    nums[1] = 42
    nums[2] = 5
    let vec: collections.Vec<Int> = collections.vec_from_slice(nums)
    let second: Int = collections.vec_get_or(vec, 1, 0)
    let first: Int = collections.vec_first_or(vec, 0)

    var keys: []i32 = core.make_i32(2)
    var values: []i32 = core.make_i32(2)
    keys[0] = 7
    keys[1] = 9
    values[0] = 99
    values[1] = 11
    let map: collections.HashMap<Int, Int> = collections.hash_map_from_slices(keys, values)
    let found: Int = collections.hash_map_get_i32_i32_or(map, 7, 0)

    var byte_keys: []u8 = core.make_u8(1)
    var byte_values: []i32 = core.make_i32(1)
    byte_keys[0] = 2
    byte_values[0] = 5
    let byte_map: collections.HashMap<UInt8, Int> = collections.hash_map_from_slices(byte_keys, byte_values)
    let byte_key: UInt8 = 2
    let byte_found: Int = collections.hash_map_get_u8_i32_or(byte_map, byte_key, 0)

    if collections.vec_len(vec) == 3 && collections.hash_map_len(map) == 2 && second == 42 && first == 7 && found == 99 && byte_found == 5:
        return 42
    return 1
`,
	})
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.t4"))
	world, err := compiler.LoadWorldOpt(entry, compiler.WorldOptions{
		DependencyRoots: []compiler.ModuleRoot{{Root: testkit.RepoRoot(t)}},
	})
	if err != nil {
		t.Fatalf("LoadWorldOpt: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	for _, want := range []string{
		"lib.core.collections.Vec__T_i32",
		"lib.core.collections.HashMap__K_i32__V_i32",
		"lib.core.collections.HashMap__K_u8__V_i32",
	} {
		if _, ok := checked.Types[want]; !ok {
			t.Fatalf("missing monomorphized collection type %q in %#v", want, checked.Types)
		}
	}
	for _, want := range []string{
		"lib.core.collections.vec_from_slice__T_i32",
		"lib.core.collections.vec_get_or__T_i32",
		"lib.core.collections.hash_map_from_slices__K_i32__V_i32",
		"lib.core.collections.hash_map_from_slices__K_u8__V_i32",
		"lib.core.collections.hash_map_get_i32_i32_or",
		"lib.core.collections.hash_map_get_u8_i32_or",
	} {
		if _, ok := checked.FuncSigs[want]; !ok {
			t.Fatalf("missing stable generic collection function %q in %#v", want, checked.FuncSigs)
		}
	}
	if _, err := compiler.LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
}

func TestGenericStructRejectsMissingTypeArgs(t *testing.T) {
	src := []byte(`
struct Box<T>:
    value: T

func main() -> Int:
    let b: Box = Box<Int>{value: 42}
    return b.value
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected missing type argument diagnostic")
	}
	if !strings.Contains(err.Error(), "generic struct 'Box' requires 1 type argument") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericStructRejectsInvalidArity(t *testing.T) {
	src := []byte(`
struct Box<T>:
    value: T

func main() -> Int:
    let b: Box<Int, Bool> = Box<Int>{value: 42}
    return b.value
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected invalid arity diagnostic")
	}
	if !strings.Contains(err.Error(), "generic struct 'Box' expects 1 type argument, got 2") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericFunctionInfersOptionalParameterElement(t *testing.T) {
	src := []byte(`
func unwrap<T>(value: T?) -> T:
    if let x = value:
        return x
    else:
        return 0

func main() -> Int:
    let value: Int? = 42
    return unwrap(value)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, ok := checked.FuncSigs["unwrap__T_i32"]; !ok {
		t.Fatalf("missing optional monomorphized signature: %#v", checked.FuncSigs)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestGenericFunctionUnsupportedArgDiagnostic(t *testing.T) {
	src := []byte(`
func id<T>(x: T) -> T:
    return x

func main() -> Int:
    return id(unknown)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected generic inference diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot infer generic argument") {
		t.Fatalf("error = %v", err)
	}
	if strings.Contains(err.Error(), "v0.5") {
		t.Fatalf("generic diagnostic should be versionless: %v", err)
	}
}

func TestGenericFunctionRejectsAmbiguousReturnOnlyInference(t *testing.T) {
	src := []byte(`
func zero<T>() -> T:
    return 0

func main() -> Int:
    return zero()
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected generic ambiguity diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot infer generic argument 'T'") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericFunctionCrossModuleMonomorphizedCall(t *testing.T) {
	files := map[string]string{
		"engine/util.tetra": `module engine.util
func id<T>(x: T) -> T:
    return x
`,
		"app/main.tetra": `module app.main
import engine.util as util

func main() -> Int:
    return util.id(42)
`,
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))

	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if _, ok := checked.FuncSigs["engine.util.id__T_i32"]; !ok {
		t.Fatalf("missing cross-module monomorphized signature: %#v", checked.FuncSigs)
	}
	if _, err := compiler.LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
}

func TestGenericStructCrossModuleMonomorphizedHappyPath(t *testing.T) {
	files := map[string]string{
		"engine/box.tetra": `module engine.box
pub struct Box<T>:
    value: T
`,
		"app/main.tetra": `module app.main
import engine.box as box

func main() -> Int:
    let b: box.Box<Int> = box.Box<Int>{value: 42}
    return b.value
`,
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))

	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if _, ok := checked.Types["engine.box.Box__T_i32"]; !ok {
		t.Fatalf("missing cross-module monomorphized struct type: %#v", checked.Types)
	}
	if _, err := compiler.LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
}

func TestGenericFunctionMonomorphizedNamesAvoidTypeCollisions(t *testing.T) {
	files := map[string]string{
		"a.tetra": `module a
struct b_c:
    x: Int
`,
		"a_b.tetra": `module a_b
struct c:
    y: Int
`,
		"util/gen.tetra": `module util.gen
func id<T>(x: T) -> T:
    return x
`,
		"app/main.tetra": `module app.main
import util.gen as util
import a as a
import a_b as ab

func main() -> Int:
    let first: a.b_c = a.b_c{x: 1}
    let second: ab.c = ab.c{y: 2}
    let firstOut: a.b_c = util.id(first)
    let secondOut: ab.c = util.id(second)
    let x: Int = firstOut.x
    let y: Int = secondOut.y
    return x + y
`,
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))

	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	var names []string
	for name := range checked.FuncSigs {
		if strings.HasPrefix(name, "util.gen.id__") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) != 2 {
		t.Fatalf("monomorphized util.id variants = %v, want 2 distinct variants", names)
	}
	if names[0] == names[1] {
		t.Fatalf("colliding monomorphized names: %v", names)
	}
	if names[0] != "util.gen.id__T_a_2e_b__c" || names[1] != "util.gen.id__T_a__b_2e_c" {
		t.Fatalf("monomorphized util.id variants = %v, want deterministic non-colliding names", names)
	}
	if _, err := compiler.LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
}

func TestGenericFunctionMultiTypeParametersMonomorphized(t *testing.T) {
	src := []byte(`
func choose<T, U>(left: T, right: U) -> T:
    return left

func main() -> Int:
    return choose(42, false)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	sig, ok := checked.FuncSigs["choose__T_i32__U_bool"]
	if !ok {
		t.Fatalf("missing multi-type monomorphized signature: %#v", checked.FuncSigs)
	}
	if sig.ReturnType != "i32" {
		t.Fatalf("choose__T_i32__U_bool return type = %q, want i32", sig.ReturnType)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestGenericStructMultiTypeParametersMonomorphized(t *testing.T) {
	src := []byte(`
struct Pair<T, U>:
    left: T
    right: U

func main() -> Int:
    let p: Pair<Int, Bool> = Pair<Int, Bool>{left: 42, right: true}
    if p.right:
        return p.left
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
	if _, ok := checked.Types["Pair__T_i32__U_bool"]; !ok {
		t.Fatalf("missing multi-type monomorphized struct type: %#v", checked.Types)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestGenericStructMonomorphizedNamesAvoidTypeCollisions(t *testing.T) {
	files := map[string]string{
		"a.tetra": `module a
pub struct b_c:
    x: Int
`,
		"a_b.tetra": `module a_b
pub struct c:
    y: Int
`,
		"util/box.tetra": `module util.box
pub struct Box<T>:
    value: T
`,
		"app/main.tetra": `module app.main
import a as a
import a_b as ab
import util.box as box

func main() -> Int:
    let first: box.Box<a.b_c> = box.Box<a.b_c>{value: a.b_c{x: 1}}
    let second: box.Box<ab.c> = box.Box<ab.c>{value: ab.c{y: 2}}
    return first.value.x + second.value.y
`,
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.tetra"))

	world, err := compiler.LoadWorld(entry)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	var names []string
	for name := range checked.Types {
		if strings.HasPrefix(name, "util.box.Box__") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	want := []string{"util.box.Box__T_a_2e_b__c", "util.box.Box__T_a__b_2e_c"}
	if len(names) != len(want) || names[0] != want[0] || names[1] != want[1] {
		t.Fatalf("monomorphized Box variants = %v, want %v", names, want)
	}
	if _, err := compiler.LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
}

func TestGenericStructOptionalFieldMonomorphized(t *testing.T) {
	src := []byte(`
struct MaybeBox<T>:
    value: T?

func main() -> Int:
    let b: MaybeBox<Int> = MaybeBox<Int>{value: none}
    if let x = b.value:
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
	info, ok := checked.Types["MaybeBox__T_i32"]
	if !ok {
		t.Fatalf("missing optional generic struct type: %#v", checked.Types)
	}
	field := info.FieldMap["value"]
	if field.TypeName != "i32?" {
		t.Fatalf("MaybeBox__T_i32.value type = %q, want i32?", field.TypeName)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestGenericStructNestedGenericFieldExplicitDiagnostic(t *testing.T) {
	src := []byte(`
struct Box<T>:
    value: T

struct Outer<T>:
    inner: Box<T>

func main() -> Int:
    let outer: Outer<Int> = Outer<Int>{inner: Box<Int>{value: 42}}
    return outer.inner.value
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected nested generic struct diagnostic")
	}
	if !strings.Contains(err.Error(), "nested generic struct instantiation") || !strings.Contains(err.Error(), "Outer__T_i32.inner") {
		t.Fatalf("error = %v", err)
	}
}

func TestGenericFunctionReturnOnlyInferenceDiagnosticStable(t *testing.T) {
	src := []byte(`
func make<T>() -> T:
    return 0

func main() -> Int:
    return make()
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected return-only inference diagnostic")
	}
	for _, want := range []string{"line 6:12", "cannot infer generic argument 'T' for 'make'"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want substring %q", err, want)
		}
	}
	if strings.Contains(err.Error(), "v0.") || strings.Contains(err.Error(), "MVP") {
		t.Fatalf("return-only inference diagnostic should remain stable and versionless: %v", err)
	}
}

func TestGenericFunctionDuplicateRecursiveWorkMonomorphizesOnce(t *testing.T) {
	src := []byte(`
func down<T>(x: T, n: Int) -> T:
    if n == 0:
        return x
    return down(x, n - 1)

func main() -> Int:
    return down(42, 2)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	count := 0
	for name := range checked.FuncSigs {
		if name == "down__T_i32" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("down__T_i32 signatures = %d, want 1", count)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestGenericStructFunctionTypeArgumentRejected(t *testing.T) {
	src := []byte(`
struct Holder<T>:
    cb: T

func add1(x: Int) -> Int:
    return x + 1

func main() -> Int:
    let holder: Holder<fn(Int) -> Int> = Holder<fn(Int) -> Int>{cb: add1}
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected function type argument diagnostic")
	}
	want := "generic struct 'Holder' type argument 'T' uses function type; generic struct instantiation cannot carry function-typed values under the supported fnptr ABI"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v", err)
	}
}
