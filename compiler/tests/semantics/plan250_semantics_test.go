package compiler_test

import (
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"

	"tetra_language/compiler/internal/testkit"
)

func TestPlan250CanonicalTypeDisplayPolicyCoversDiagnosticsAndDocs(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "uint8 alias",
			src: `
func main() -> Int:
    let byte: UInt8 = true
    return byte
`,
			want: "type mismatch: expected 'u8', got 'bool'",
		},
		{
			name: "string alias",
			src: `
func main() -> Int:
    let text: String = 1
    return text.len
`,
			want: "type mismatch: expected 'str', got 'i32'",
		},
		{
			name: "bool alias",
			src: `
func main() -> Int:
    let flag: Bool = 1
    if flag:
        return 1
    return 0
`,
			want: "type mismatch: expected 'bool', got 'i32'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(tt.src)
			if err == nil {
				t.Fatalf("expected canonical alias diagnostic")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}

	docs, err := compiler.GenerateAPIDocsFromSource([]byte(`
struct Packet:
    id: Int
    payload: String
    byte: UInt8
    ok: Bool

func inspect(packet: Packet, raw: ptr) -> String:
    return packet.payload
`), "canonical-types.tetra")
	if err != nil {
		t.Fatalf("GenerateAPIDocsFromSource: %v", err)
	}
	out := string(docs)
	for _, want := range []string{
		"`id: i32`",
		"`payload: str`",
		"`byte: u8`",
		"`ok: bool`",
		"`func inspect(packet: Packet, raw: ptr) -> str`",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("API docs missing canonical form %q:\n%s", want, out)
		}
	}
	for _, forbidden := range []string{"`id: Int`", "`payload: String`", "`byte: UInt8`", "`ok: Bool`", "-> String"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("API docs kept non-canonical type spelling %q:\n%s", forbidden, out)
		}
	}
}

func TestPlan250StructFieldResolutionDiagnosticsStable(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "unknown constructor field",
			src: `
struct Pair:
    x: Int

func main() -> Int:
    let p: Pair = Pair(y: 1)
    return 0
`,
			want: "unknown field 'y'",
		},
		{
			name: "unknown field access",
			src: `
struct Pair:
    x: Int

func main() -> Int:
    let p: Pair = Pair(x: 1)
    return p.y
`,
			want: "unknown field 'y'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(tt.src)
			if err == nil {
				t.Fatalf("expected diagnostic")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestPlan250ModuleBoundaryVisibilityDiagnosticStable(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"engine/math.t4": `module engine.math
pub func add(a: Int, b: Int) -> Int:
    return a + b
func hidden() -> Int:
    return 1
`,
		"app/main.t4": `module app.main
import engine.math as math
func main() -> Int:
    return math.hidden()
`,
	})
	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.t4")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected private function diagnostic")
	}
	if !strings.Contains(err.Error(), "private function 'engine.math.hidden'") {
		t.Fatalf("error = %v", err)
	}
}

func TestPlan250GenericSpecializationNamesDeterministic(t *testing.T) {
	src := []byte(`
struct Box:
    value: Int

func id<T>(x: T) -> T:
    return x

func main() -> Int:
    let a: Int = id(1)
    let b: Box = id(Box(value: 2))
    return a + b.value
`)
	firstProg, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	first, err := compiler.Check(firstProg)
	if err != nil {
		t.Fatalf("Check first: %v", err)
	}
	secondProg, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse second: %v", err)
	}
	second, err := compiler.Check(secondProg)
	if err != nil {
		t.Fatalf("Check second: %v", err)
	}
	for _, name := range []string{"id__T_i32", "id__T_Box"} {
		if _, ok := first.FuncSigs[name]; !ok {
			t.Fatalf("first check missing specialization %q in %#v", name, first.FuncSigs)
		}
		if _, ok := second.FuncSigs[name]; !ok {
			t.Fatalf("second check missing specialization %q in %#v", name, second.FuncSigs)
		}
	}
	if len(first.FuncSigs) != len(second.FuncSigs) {
		t.Fatalf("specialization count changed: first=%d second=%d", len(first.FuncSigs), len(second.FuncSigs))
	}
}

func TestPlan250CrossModuleGenericMonomorphizationAndInferenceDiagnostic(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"lib/generic.t4": `module lib.generic
pub func id<T>(x: T) -> T:
    return x
`,
		"app/main.t4": `module app.main
import lib.generic as generic
func main() -> Int:
    return generic.id(42)
`,
	})
	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.t4")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if _, ok := checked.FuncSigs["lib.generic.id__T_i32"]; !ok {
		t.Fatalf("missing cross-module specialization lib.generic.id__T_i32 in %#v", checked.FuncSigs)
	}

	err = testkit.CheckProgram(`
func make<T>() -> T:
    return 0

func main() -> Int:
    return make()
`)
	if err == nil {
		t.Fatalf("expected return-only generic inference diagnostic")
	}
	if !strings.Contains(err.Error(), "cannot infer generic argument 'T'") {
		t.Fatalf("error = %v", err)
	}
}

func TestPlan250ProtocolConformanceAndDynamicDispatchBoundaries(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "requirement signature return mismatch",
			src: `
struct Vec2:
    x: Int

protocol Drawable:
    func draw(self: Vec2) -> Int

extension Vec2:
    func draw(self: Vec2) -> Bool:
        return true

impl Vec2: Drawable
`,
			want: "return type differs",
		},
		{
			name: "generic bound requirement call unsupported",
			src: `
struct Vec2:
    x: Int

protocol Drawable:
    func draw(self: Vec2) -> Int

extension Vec2:
    func draw(self: Vec2) -> Int:
        return self.x

impl Vec2: Drawable

func render<T: Drawable>(value: T) -> Int:
    return Drawable.draw(value)

func main() -> Int:
    return render(Vec2(x: 1))
`,
			want: "unknown function 'Drawable.draw'",
		},
		{
			name: "protocol runtime value unsupported",
			src: `
struct Vec2:
    x: Int

protocol Drawable:
    func draw(self: Vec2) -> Int

func main() -> Int:
    let value: Drawable = Vec2(x: 1)
    return 0
`,
			want: "unknown type 'Drawable'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(tt.src)
			if err == nil {
				t.Fatalf("expected diagnostic")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestPlan250EnumPayloadOptionalTypedErrorAndExtensionBoundaries(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "enum constructor arity",
			src: `
enum Result:
    case ok(Int)

func main() -> Int:
    let r: Result = Result.ok()
    return 0
`,
			want: "expects 1 payload argument(s), got 0",
		},
		{
			name: "enum constructor payload type",
			src: `
enum Result:
    case ok(Int)

func main() -> Int:
    let r: Result = Result.ok(true)
    return 0
`,
			want: "payload 1 expects 'i32', got 'bool'",
		},
		{
			name: "default before explicit enum case",
			src: `
enum Result:
    case ok
    case err

func main() -> Int:
    match Result.ok:
    case _:
        return 0
    case Result.err:
        return 1
`,
			want: "match default must be last",
		},
		{
			name: "catch guarded case is not exhaustive",
			src: `
enum ReadError:
    case denied(Int)

func read() -> Int throws ReadError:
    throw ReadError.denied(1)

func main() -> Int:
    let value: Int = catch read():
    case ReadError.denied(code) if code == 1:
        code
    return value
`,
			want: "catch expression must be exhaustive",
		},
		{
			name: "extension duplicate deterministic",
			src: `
struct Vec2:
    x: Int

extension Vec2:
    func sum(self: Vec2) -> Int:
        return self.x

extension Vec2:
    func sum(self: Vec2) -> Int:
        return self.x
`,
			want: "duplicate function 'Vec2.sum'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(tt.src)
			if err == nil {
				t.Fatalf("expected diagnostic")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestPlan250EnumUnguardedMatchAndCatchExhaustiveness(t *testing.T) {
	src := []byte(`
enum Color:
    case red
    case green

enum ReadError:
    case eof
    case denied(Int)

func classify(color: Color) -> Int:
    match color:
    case Color.red:
        return 1
    case Color.green:
        return 2

func read(flag: Bool) -> Int throws ReadError:
    if flag:
        return 40
    throw ReadError.denied(2)

func main() -> Int:
    let recovered: Int = catch read(false):
    case ReadError.eof:
        0
    case ReadError.denied(code):
        code
    return classify(Color.green) + recovered
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

	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "unguarded match expression missing enum case",
			src: `
enum Result:
    case ok(Int)
    case err(Int)

func main() -> Int:
    let value: Int = match Result.ok(1):
    case Result.ok(code):
        code
    return value
`,
			want: "match expression must be exhaustive",
		},
		{
			name: "unguarded catch expression missing enum case",
			src: `
enum ReadError:
    case eof
    case denied(Int)

func read() -> Int throws ReadError:
    throw ReadError.eof

func main() -> Int:
    return catch read():
    case ReadError.eof:
        0
`,
			want: "catch expression must be exhaustive",
		},
		{
			name: "guarded default match expression is not exhaustive",
			src: `
func main() -> Int:
    let value: Int = match 7:
    case _ if false:
        99
    return value
`,
			want: "match expression must be exhaustive",
		},
		{
			name: "guarded default catch expression is not exhaustive",
			src: `
enum ReadError:
    case denied(Int)

func read() -> Int throws ReadError:
    throw ReadError.denied(7)

func main() -> Int:
    let value: Int = catch read():
    case _ if false:
        99
    return value
`,
			want: "catch expression must be exhaustive",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(tt.src)
			if err == nil {
				t.Fatalf("expected diagnostic")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestPlan250OptionalTypedErrorSupportedBoundary(t *testing.T) {
	src := []byte(`
enum ReadError:
    case denied(Int)

func read(flag: Bool) -> Int? throws ReadError:
    if flag:
        return 42
    throw ReadError.denied(7)

func main() -> Int:
    let maybe: Int? = catch read(false):
    case ReadError.denied(code):
        code
    if let value = maybe:
        return value
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
	if got := checked.FuncSigs["read"].ReturnType; got != "i32?" {
		t.Fatalf("read return type = %q, want i32?", got)
	}
	if got := checked.FuncSigs["read"].ThrowsType; got != "ReadError" {
		t.Fatalf("read throws type = %q, want ReadError", got)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestPlan250ExtensionResolutionOrderStableAcrossImports(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"engine/core.t4": `module engine.core
pub struct Vec2:
    x: Int
`,
		"engine/ext.t4": `module engine.ext
import engine.core as core

extension core.Vec2:
    func sum(self: core.Vec2) -> Int:
        return self.x
`,
		"app/main.t4": `module app.main
import engine.ext as ext
import engine.core as core

func main() -> Int:
    let v: core.Vec2 = core.Vec2(x: 42)
    return core.Vec2.sum(v)
`,
	})
	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.t4")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if _, ok := checked.FuncSigs["engine.core.Vec2.sum"]; !ok {
		t.Fatalf("missing imported extension signature: %#v", checked.FuncSigs)
	}
	if _, err := compiler.LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}
}

func TestPlan250FunctionTypeLocalBindingAndCallbackBoundaries(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "function typed local rejects capturing generic closure literal reassignment",
			src: `
func main() -> Int:
    let base: Int = 1
    var f: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + 1
    f = fn<T>(x: T) -> T:
        let _: Int = base
        return x
    return f(0)
`,
			want: "generic closure captures are not supported by the production fnptr ABI; use a non-generic closure or pass captured state explicitly",
		},
		{
			name: "throwing callback symbol unsupported",
			src: `
enum Boom:
    case bad

func fail(x: Int) -> Int throws Boom:
    throw Boom.bad

func apply(cb: fn(Int) -> Int, x: Int) -> Int:
    return cb(x)

func main() -> Int:
    return apply(fail, 1)
`,
			want: "throwing function symbol 'fail' cannot be used as callback argument; callback fnptr ABI requires the parameter's declared throws type to match",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testkit.CheckProgram(tt.src)
			if err == nil {
				t.Fatalf("expected diagnostic")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestPlan250CapsuleMetadataHasNoRuntimeCoupling(t *testing.T) {
	compile := func(src string) (*compiler.CheckedProgram, *compiler.IRProgram, error) {
		file, err := compiler.ParseFile([]byte(src), "plan250_capsule.tetra")
		if err != nil {
			return nil, nil, err
		}
		world := &compiler.World{
			EntryModule: "",
			Files:       []*compiler.FileAST{file},
			ByModule:    map[string]*compiler.FileAST{"": file},
		}
		checked, err := compiler.CheckWorld(world)
		if err != nil {
			return nil, nil, err
		}
		irProg, err := compiler.Lower(checked)
		if err != nil {
			return nil, nil, err
		}
		return checked, irProg, nil
	}

	withCapsule := `
capsule App:
    id: "tetra://plan250"
    version: "1.0.0"

func main() -> Int:
    return 0
`
	withoutCapsule := `
func main() -> Int:
    return 0
`
	withChecked, withIR, err := compile(withCapsule)
	if err != nil {
		t.Fatalf("compile with capsule: %v", err)
	}
	withoutChecked, withoutIR, err := compile(withoutCapsule)
	if err != nil {
		t.Fatalf("compile without capsule: %v", err)
	}
	if len(withChecked.Funcs) != len(withoutChecked.Funcs) || len(withChecked.Types) != len(withoutChecked.Types) {
		t.Fatalf("capsule changed semantic function/type counts: with funcs=%d types=%d without funcs=%d types=%d", len(withChecked.Funcs), len(withChecked.Types), len(withoutChecked.Funcs), len(withoutChecked.Types))
	}
	if withChecked.MainName != withoutChecked.MainName || withIR.MainName != withoutIR.MainName {
		t.Fatalf("capsule changed main metadata: checked %q/%q ir %q/%q", withChecked.MainName, withoutChecked.MainName, withIR.MainName, withoutIR.MainName)
	}
	withMain := findIRFunc(t, withIR.Funcs, "main")
	withoutMain := findIRFunc(t, withoutIR.Funcs, "main")
	if withMain.ParamSlots != withoutMain.ParamSlots || withMain.LocalSlots != withoutMain.LocalSlots || withMain.ReturnSlots != withoutMain.ReturnSlots || len(withMain.Instrs) != len(withoutMain.Instrs) {
		t.Fatalf("capsule changed lowered main shape: with=%#v without=%#v", withMain, withoutMain)
	}
}
