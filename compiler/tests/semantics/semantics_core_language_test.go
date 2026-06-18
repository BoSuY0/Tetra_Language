package compiler_test

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/testkit"
)

// ---- api_epic04_test.go ----

func TestPublicCheckAPISingleSourcePositive(t *testing.T) {
	prog, err := compiler.Parse([]byte(`
fun main(): i32 {
  return 42
}
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if checked.MainName != "main" {
		t.Fatalf("main name = %q, want main", checked.MainName)
	}
}

func TestPublicCheckAPISingleSourceNegativeDiagnostic(t *testing.T) {
	prog, err := compiler.Parse([]byte(`
fun main(): i32 {
  let x: i32 = true
  return x
}
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected semantic error")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != compiler.DiagnosticCodeSemantic {
		t.Fatalf("diagnostic code = %q, want %q", diag.Code, compiler.DiagnosticCodeSemantic)
	}
	if !strings.Contains(diag.Message, "type mismatch: expected 'i32', got 'bool'") {
		t.Fatalf("diagnostic message = %q", diag.Message)
	}
}

func TestPublicCheckAPICrossModuleWorldPositive(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"engine/math.tetra": "module engine.math\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/main.tetra":    "module app.main\nimport engine.math as math\nfun main(): i32 {\n  return math.add_one(41)\n}\n",
	})
	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.tetra")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if checked.MainName != "app.main.main" {
		t.Fatalf("main name = %q, want app.main.main", checked.MainName)
	}
	if checked.FuncSigs["engine.math.add_one"].ReturnType != "i32" {
		t.Fatalf("unexpected imported signature: %#v", checked.FuncSigs["engine.math.add_one"])
	}
}

func TestPublicCheckAPIDisplayTextForBoundaryError(t *testing.T) {
	_, err := compiler.Check(nil)
	if err == nil {
		t.Fatalf("expected nil program boundary error")
	}
	if err.Error() != "no program provided" {
		t.Fatalf("error = %q, want no program provided", err.Error())
	}
}

func writeTestFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, contents := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatalf("WriteFile(%q): %v", path, err)
		}
	}
}

// ---- argument_labels_test.go ----

func TestArgumentLabelsAcceptedByChecker(t *testing.T) {
	src := `
func add(a: Int, b: Int) -> Int:
    return a + b

func main() -> Int:
    return add(a: 40, b: 2)
`
	testkit.RequireCheckOK(t, src)
}

func TestArgumentLabelsRejectMismatchedOrder(t *testing.T) {
	src := `
func add(a: Int, b: Int) -> Int:
    return a + b

func main() -> Int:
    return add(b: 40, a: 2)
`
	testkit.RequireCheckErrorContains(t, src, "argument label mismatch")
}

// ---- array_mvp_test.go ----

func TestArrayMVPCheckAcceptsIndexAndForOnFixedArray(t *testing.T) {
	testkit.RequireCheckOK(t, `
func touch(seed: [3]Int) -> Int:
  var xs: [3]Int = seed
  xs[0] = 40
  xs[1] = 2
  xs[2] = xs[0] + xs[1]
  var total: Int = 0
  for x in xs:
    total = total + x
  return total

func main() -> Int:
  return 0
`)
}

func TestArrayMVPBuildSmoke(t *testing.T) {
	src := `func touch(seed: [3]Int) -> Int:
    var xs: [3]Int = seed
    xs[0] = 40
    xs[1] = 2
    xs[2] = xs[0] + xs[1]
    var total: Int = 0
    for x in xs:
        total = total + x
    return total

func main() -> Int:
    return 0
`
	if err := buildOnly(t, src); err != nil {
		t.Fatalf("build: %v", err)
	}
}

func TestArrayMVPBuildSupportsZeroedFixedArrayFieldGlobal(t *testing.T) {
	src := `struct ArrayBox:
    items: [2]Int

var leaked: ArrayBox

func configure(seed: [2]Int) -> Int:
    leaked.items = seed
    return leaked.items[0]

func main() -> Int:
    return 0
`
	if err := buildOnly(t, src); err != nil {
		t.Fatalf("build: %v", err)
	}
}

func TestArrayMVPBuildSupportsOptionalFixedArrayGlobal(t *testing.T) {
	src := `var maybe: [2]Int? = none

func configure(seed: [2]Int) -> Int:
    maybe = seed
    if let xs = maybe:
        return xs[0]
    else:
        return 0

func main() -> Int:
    return 0
`
	if err := buildOnly(t, src); err != nil {
		t.Fatalf("build: %v", err)
	}
}

func TestArrayMVPWasmBuildSmoke(t *testing.T) {
	src := `func touch(seed: [3]Int) -> Int:
    var xs: [3]Int = seed
    xs[0] = 40
    xs[1] = 2
    xs[2] = xs[0] + xs[1]
    var total: Int = 0
    for x in xs:
        total = total + x
    return total

func main() -> Int:
    return 0
`
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		outPath := filepath.Join(tmp, "app-"+target+".wasm")
		if _, err := compiler.BuildFileWithStatsOpt(
			srcPath,
			outPath,
			target,
			compiler.BuildOptions{Jobs: 1},
		); err != nil {
			t.Fatalf("build %s: %v", target, err)
		}
	}
}

func buildOnly(t *testing.T, src string) error {
	t.Helper()

	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "app")

	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	return compiler.BuildFile(srcPath, outPath, "linux-x64")
}

func TestArrayMVPRejectsUnsupportedElementType(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func main() -> Int:
  let xs: [2]String = 0
  return 0
`, "array element type 'str' is not supported")
}

func TestArrayMVPRejectsNonPositiveSize(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func main() -> Int:
  let xs: [0]Int = 0
  return 0
`, "array size must be positive constant")
}

func TestArrayMVPRejectsAssignmentToArrayLen(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func probe(seed: [2]Int) -> Int:
  var xs: [2]Int = seed
  xs.len = 7
  return 0

func main() -> Int:
  return 0
`, "cannot assign to fixed-array internals ('ptr'/'len'); assign elements via index instead")
}

func TestArrayMVPRejectsAssignmentToArrayPtr(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func probe(seed: [2]Int) -> Int:
  var xs: [2]Int = seed
  xs.ptr = 0
  return 0

func main() -> Int:
  return 0
`, "cannot assign to fixed-array internals ('ptr'/'len'); assign elements via index instead")
}

func TestArrayMVPRejectsAssignmentToNestedArrayLen(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
struct Box:
  arr: [2]Int

func probe(b0: Box) -> Int:
  var b: Box = b0
  b.arr.len = 3
  return 0

func main() -> Int:
  return 0
`, "cannot assign to fixed-array internals ('ptr'/'len'); assign elements via index instead")
}

// ---- capsule_mvp_test.go ----

func checkCapsuleFileProgram(src string) error {
	file, err := compiler.ParseFile([]byte(src), "capsule_mvp.tetra")
	if err != nil {
		return err
	}
	world := &compiler.World{
		EntryModule: "",
		Files:       []*compiler.FileAST{file},
		ByModule:    map[string]*compiler.FileAST{"": file},
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		return err
	}
	_, err = compiler.Lower(checked)
	return err
}

func TestCapsuleMetadataPositiveNoRuntimeImpact(t *testing.T) {
	src := `
capsule App:
    id: "tetra://app"
    version: "0.1.0"
    target: "linux-x64"
    debug.enabled: true

func main() -> Int:
    return 0
`
	if err := checkCapsuleFileProgram(src); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestCapsuleMetadataSemanticDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "duplicate key",
			src: `
capsule App:
    id: "tetra://app"
    id: "tetra://dup"

func main() -> Int:
    return 0
`,
			want: "duplicate capsule metadata key 'id'",
		},
		{
			name: "invalid key shape",
			src: `
capsule App:
    ID: "tetra://app"

func main() -> Int:
    return 0
`,
			want: "invalid capsule metadata key 'ID'",
		},
		{
			name: "invalid value shape",
			src: `
capsule App:
    version: 1 + 2

func main() -> Int:
    return 0
`,
			want: "capsule metadata value for key 'version' must be a literal",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkCapsuleFileProgram(tt.src)
			if err == nil {
				t.Fatalf("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestCapsuleAndPropertyAcceptedTogether(t *testing.T) {
	src := `
capsule App:
    id: "tetra://app"

property title: Int = 7

func main() -> Int:
    return title
`
	if err := checkCapsuleFileProgram(src); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

// ---- compound_assignment_test.go ----

func TestBuildCompoundAssignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func main() -> Int:
    var x: Int = 4
    x += 3
    x *= 6
    x -= 0
    x /= 1
    x %= 100
    return x
`
	_, exitCode := buildAndRun(t, src)
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestBuildCompoundAssignmentFieldAndIndexSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
struct Box:
    x: Int

func main() -> Int
uses alloc, mem:
    var b: Box = Box(x: 20)
    b.x += 1
    var xs: []i32 = make_i32(1)
    xs[0] = 20
    xs[0] += 1
    return b.x + xs[0]
`
	_, exitCode := buildAndRun(t, src)
	if exitCode != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", exitCode)
	}
}

func TestCompoundIndexAssignmentRejectsSideEffectingTarget(t *testing.T) {
	testkit.RequireCheckErrorContains(t, `
func next() -> Int:
    return 0

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 40
    xs[next()] += 2
    return xs[0]
`, "compound index assignment target with side effects")
}

func TestCompoundIndexAssignmentAllowsStableTarget(t *testing.T) {
	testkit.RequireCheckOK(t, `
func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    xs[0] = 40
    xs[0] += 2
    return xs[0]
`)
}

// ---- const_test.go ----

func TestBuildConstGlobalSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `const base: i32 = 40
const delta = 2

func main() -> Int:
    return base + delta
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestFormatSourceConstGlobal(t *testing.T) {
	src := []byte(`const answer: i32 = 42
func main() -> Int:
    return answer
`)
	got, err := compiler.FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `const answer: i32 = 42

func main() -> Int:
    return answer
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestBuildConstBoolGlobalSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `const enabled = true

func main() -> Int:
    if enabled:
        return 42
    return 1
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildConstGlobalExpressionSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `const base: i32 = (20 + 2) * 2
const delta = 100 % 3
const enabled: Bool = (base + delta == 45) && !false

func main() -> Int:
    if enabled:
        return base + delta - 3
    return 1
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestGlobalVarConstInitializerSemantics(t *testing.T) {
	src := `
var base: Int = 40 + 2
var enabled: Bool = true && !false

func main() -> Int:
    if enabled:
        return base
    return 0
`
	file, err := compiler.ParseFile([]byte(src), "main.tetra")
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

func TestBuildGlobalVarConstInitializerSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
var base: Int = 40 + 2
var enabled: Bool = true

func main() -> Int:
    if enabled:
        return base
    return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestPropertyDefaultInitializerSemantics(t *testing.T) {
	src := `
property base: Int
property enabled: Bool
property p: ptr
property b: UInt8
property w: UInt16
property terr: task.error

func main() -> Int:
    if enabled:
        return 1
    if terr != 0:
        return 2
    return base + b + w + 42
`
	file, err := compiler.ParseFile([]byte(src), "main.tetra")
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
	globals := checked.GlobalsByModule[""]
	data := checked.GlobalDataByModule[""]
	for _, name := range []string{"base", "enabled", "p", "b", "w", "terr"} {
		info, ok := globals[name]
		if !ok {
			t.Fatalf("missing global %q", name)
		}
		if info.DataIndex < 0 || info.DataIndex >= len(data) {
			t.Fatalf("global %q data index out of range: %d", name, info.DataIndex)
		}
		if got := binary.LittleEndian.Uint64(data[info.DataIndex]); got != 0 {
			t.Fatalf("global %q default = %d, want 0", name, got)
		}
	}
}

func TestBuildPropertyDefaultInitializerSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
property base: Int
property enabled: Bool
property p: ptr
property b: UInt8
property w: UInt16
property terr: task.error

func main() -> Int:
    if enabled:
        return 1
    if terr != 0:
        return 2
    return base + b + w + 42
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestGlobalVarExpandedScalarInitializerSemantics(t *testing.T) {
	src := `
var b: UInt8 = 41
var w: UInt16 = 1
var terr: task.error = 0

func main() -> Int:
    if terr != 0:
        return 0
    return b + w
`
	file, err := compiler.ParseFile([]byte(src), "main.tetra")
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

func TestGlobalVarStringLiteralInitializerSemantics(t *testing.T) {
	src := `
var title: String = "hello"

func main() -> Int:
    return 0
`
	file, err := compiler.ParseFile([]byte(src), "main.tetra")
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
	globals := checked.GlobalsByModule[""]
	info, ok := globals["title"]
	if !ok {
		t.Fatalf("missing global %q", "title")
	}
	if info.TypeName != "str" {
		t.Fatalf("global title type = %q, want str", info.TypeName)
	}
	data := checked.GlobalDataByModule[""]
	if info.DataIndex < 0 || info.DataIndex+1 >= len(data) {
		t.Fatalf("global title data index out of range: %d len=%d", info.DataIndex, len(data))
	}
	if got := binary.LittleEndian.Uint64(data[info.DataIndex+1]); got != 5 {
		t.Fatalf("global title len slot = %d, want 5", got)
	}
}

func TestGlobalValStringLiteralInitializerSemantics(t *testing.T) {
	src := `
val title: String = "hello"

func main() -> Int:
    return 0
`
	file, err := compiler.ParseFile([]byte(src), "main.tetra")
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
	globals := checked.GlobalsByModule[""]
	info, ok := globals["title"]
	if !ok {
		t.Fatalf("missing global %q", "title")
	}
	if info.TypeName != "str" {
		t.Fatalf("global title type = %q, want str", info.TypeName)
	}
	data := checked.GlobalDataByModule[""]
	if info.DataIndex < 0 || info.DataIndex+1 >= len(data) {
		t.Fatalf("global title data index out of range: %d len=%d", info.DataIndex, len(data))
	}
	if got := binary.LittleEndian.Uint64(data[info.DataIndex+1]); got != 5 {
		t.Fatalf("global title len slot = %d, want 5", got)
	}
}

func TestGlobalStringFieldAccessSemantics(t *testing.T) {
	src := `
val greeting: String = "hello wasm"

func main() -> Int:
    return greeting.len
`
	file, err := compiler.ParseFile([]byte(src), "main.tetra")
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

func TestBuildGlobalVarStringLiteralInitializerSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
var title: String = "hello"

func main() -> Int:
    return title[1]
`
	_, code := buildAndRun(t, src)
	if code != int('e') {
		t.Fatalf("exit code mismatch: got %d, want %d", code, int('e'))
	}
}

func TestBuildGlobalVarStringFieldAccessAfterAssignmentSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
var title: String = "hello"

func main() -> Int:
    title = "bye"
    return title.len
`
	_, code := buildAndRun(t, src)
	if code != 3 {
		t.Fatalf("exit code mismatch: got %d, want 3", code)
	}
}

func TestBuildLocalStringFieldAccessSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
func main() -> Int:
    let s: String = "bye"
    return s.len
`
	_, code := buildAndRun(t, src)
	if code != 3 {
		t.Fatalf("exit code mismatch: got %d, want 3", code)
	}
}

func TestBuildPropertyStringLiteralInitializerSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
property title: String = "hello"

func main() -> Int:
    return title[4]
`
	_, code := buildAndRun(t, src)
	if code != int('o') {
		t.Fatalf("exit code mismatch: got %d, want %d", code, int('o'))
	}
}

func TestGlobalVarPtrZeroInitializerSemantics(t *testing.T) {
	src := `
var p: ptr = 0

func main() -> Int:
    return 42
`
	file, err := compiler.ParseFile([]byte(src), "main.tetra")
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

func TestBuildGlobalVarPtrZeroInitializerSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
var p: ptr = 0

func main() -> Int:
    return 42
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestGlobalVarPtrConstZeroInitializerSemantics(t *testing.T) {
	src := `
const z: Int = 0
var p: ptr = z

func main() -> Int:
    return 42
`
	file, err := compiler.ParseFile([]byte(src), "main.tetra")
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

func TestBuildGlobalVarExpandedScalarInitializerSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
var b: UInt8 = 41
var w: UInt16 = 1
var terr: task.error = 0

func main() -> Int:
    if terr != 0:
        return 0
    return b + w
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildPropertyExplicitInitializerSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `
property base: Int = 40 + 2

func main() -> Int:
    return base
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestConstGlobalExpressionDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "unknown const",
			src: `const answer: i32 = missing + 1

func main() -> Int:
    return answer
`,
			want: "unknown constant 'missing'",
		},
		{
			name: "division by zero",
			src: `const answer: i32 = 1 / 0

func main() -> Int:
    return answer
`,
			want: "division by zero in global const expression",
		},
		{
			name: "modulo by zero",
			src: `const answer: i32 = 1 % 0

func main() -> Int:
    return answer
`,
			want: "modulo by zero in global const expression",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := buildOnly(t, tt.src)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got: %v", tt.want, err)
			}
		})
	}
}

func TestGlobalVarInitializerDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "non constant int initializer call",
			src: `
var answer: Int = forty() + 2

func forty() -> Int:
    return 40

func main() -> Int:
    return answer
`,
			want: "global var 'answer' initializer must be an i32 constant expression",
		},
		{
			name: "non constant bool initializer call",
			src: `
var enabled: Bool = enabled_now()

func enabled_now() -> Bool:
    return true

func main() -> Int:
    if enabled:
        return 1
    return 0
`,
			want: "global var 'enabled' initializer must be a bool constant expression",
		},
		{
			name: "string initializer must be literal",
			src: `
var msg: String = msg_now()

func msg_now() -> String:
    return "hi"

func main() -> Int:
    return 0
`,
			want: "global var 'msg' initializer must be a string literal",
		},
		{
			name: "u8 initializer out of range",
			src: `
var b: UInt8 = 256

func main() -> Int:
    return 0
`,
			want: "global var 'b' initializer must be within 0..255 for type u8",
		},
		{
			name: "u16 initializer out of range",
			src: `
var w: UInt16 = 70000

func main() -> Int:
    return 0
`,
			want: "global var 'w' initializer must be within 0..65535 for type u16",
		},
		{
			name: "u8 initializer overflow expression",
			src: `
var b: UInt8 = 65536 * 65536

func main() -> Int:
    return 0
`,
			want: "overflow in global const expression",
		},
		{
			name: "u16 initializer overflow expression",
			src: `
var w: UInt16 = 65536 * 65536

func main() -> Int:
    return 0
`,
			want: "overflow in global const expression",
		},
		{
			name: "const int initializer overflow expression",
			src: `
const wrapped: Int = 65536 * 65536

func main() -> Int:
    return wrapped
`,
			want: "overflow in global const expression",
		},
		{
			name: "ptr initializer non-zero for var",
			src: `
var p: ptr = 1

func main() -> Int:
    return 0
`,
			want: "global var 'p' of type ptr only supports initializer 0",
		},
		{
			name: "ptr initializer non-constant for var",
			src: `
var p: ptr = zero()

func zero() -> Int:
    return 0

func main() -> Int:
    return 0
`,
			want: "global var 'p' initializer for type ptr must be a constant 0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := buildOnly(t, tt.src)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got: %v", tt.want, err)
			}
		})
	}
}

func TestPropertyInitializerDiagnostics(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "property string initializer must be literal",
			src: `
property title: String = title_now()

func title_now() -> String:
    return "hello"

func main() -> Int:
    return 0
`,
			want: "global val 'title' initializer must be a string literal",
		},
		{
			name: "property u16 initializer out of range",
			src: `
property w: UInt16 = 70000

func main() -> Int:
    return 0
`,
			want: "global val 'w' initializer must be within 0..65535 for type u16",
		},
		{
			name: "non constant property bool initializer",
			src: `
property enabled: Bool = enabled_now()

func enabled_now() -> Bool:
    return true

func main() -> Int:
    if enabled:
        return 1
    return 0
`,
			want: "global val 'enabled' initializer must be a bool constant expression",
		},
		{
			name: "property ptr non-zero initializer",
			src: `
property p: ptr = 1

func main() -> Int:
    return 0
`,
			want: "global val 'p' of type ptr only supports initializer 0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := buildOnly(t, tt.src)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.want)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got: %v", tt.want, err)
			}
		})
	}
}

func TestBuildLocalConstSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int:
    const answer: Int = 42
    return answer
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestFormatSourceLocalConst(t *testing.T) {
	src := []byte(`func main() -> Int:
    const answer: Int = 42
    return answer
`)
	got, err := compiler.FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `func main() -> Int:
    const answer: Int = 42
    return answer
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

// ---- defer_test.go ----

func TestDeferRunsLIFOAndPreservesReturnValue(t *testing.T) {
	src := `func main() -> Int
uses io:
    defer:
        print("a")
    defer:
        print("b")
    return 42
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "ba" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 42 {
		t.Fatalf("exit code = %d, want 42", exitCode)
	}
}

func TestDeferRunsOnNestedReturnBeforeOuterCleanup(t *testing.T) {
	src := `func main() -> Int
uses io:
    defer:
        print("outer")
    if true:
        defer:
            print("inner")
        return 7
    return 1
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "innerouter" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 7 {
		t.Fatalf("exit code = %d, want 7", exitCode)
	}
}

func TestDeferRunsWhenLoopScopeExitsByBreak(t *testing.T) {
	src := `func main() -> Int
uses io:
    while true:
        defer:
            print("loop")
        break
    print("after")
    return 0
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "loopafter" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}
}

func TestDeferRunsWhenLoopScopeExitsByContinue(t *testing.T) {
	src := `func main() -> Int
uses io:
    var i: Int = 0
    while i < 2:
        i = i + 1
        defer:
            print("tick")
        continue
    return 0
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "ticktick" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}
}

func TestDeferRunsBeforeThrowReturn(t *testing.T) {
	src := `enum E:
    case bad

func fail() -> Int throws E
uses io:
    defer:
        print("cleanup")
    throw E.bad

func main() -> Int
uses io:
    return catch fail():
    case E.bad:
        3
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "cleanup" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 3 {
		t.Fatalf("exit code = %d, want 3", exitCode)
	}
}

func TestDeferRunsBeforeScopedIslandAutoFree(t *testing.T) {
	src := `func main() -> Int
uses alloc, islands, io, mem:
    island(64) as isl:
        var msg: []u8 = core.island_make_u8(isl, 2)
        msg[0] = 79
        msg[1] = 10
        defer:
            print(msg)
    return 0
`
	stdout, exitCode := buildAndRun(t, src)
	if stdout != "O\n" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}
}

func TestDeferRunsWhenCanceledTaskCheckpoints(t *testing.T) {
	src := `func worker() -> Int
uses io, runtime:
    defer:
        print("cleanup")
    let group: task.group = core.task_group_current()
    let _canceledGroup: task.group = core.task_group_cancel(group)
    let checkpoint: task.error = core.task_checkpoint()
    if checkpoint != 0:
        return 5
    return 9

func main() -> Int
uses runtime:
    let group: task.group = core.task_group_open()
    let task: task.i32 = core.task_spawn_group_i32(group, "worker")
    let result: task.result_i32 = core.task_join_result_i32(task)
    let _closed: Int = core.task_group_close(group)
    if result.error != 0:
        return 80 + result.error
    return result.value
`
	stdout, exitCode := buildAndRunWithOptions(t, src, compiler.BuildOptions{})
	if stdout != "cleanup" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 5 {
		t.Fatalf("exit code = %d, want 5", exitCode)
	}
}

func TestDeferRejectsReturnInsideCleanup(t *testing.T) {
	src := []byte(`func main() -> Int:
    defer:
        return 1
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected semantic error")
	}
	if !strings.Contains(err.Error(), "return is not allowed in defer") {
		t.Fatalf("error = %v", err)
	}
}

func TestDeferRejectsBreakToOuterLoopInsideCleanup(t *testing.T) {
	src := []byte(`func main() -> Int:
    while true:
        defer:
            break
        return 0
    return 1
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected semantic error")
	}
	if !strings.Contains(err.Error(), "break is not allowed in defer") {
		t.Fatalf("error = %v", err)
	}
}

func TestDeferRejectsLaterConsumeOfCapturedValue(t *testing.T) {
	src := []byte(`func take(x: consume Int) -> Int:
    return x

func main() -> Int:
    let a: Int = 1
    defer:
        let _captured: Int = a
    let b: Int = take(a)
    return b
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected semantic error")
	}
	if !strings.Contains(err.Error(), "defer cleanup captures value 'a'") {
		t.Fatalf("error = %v", err)
	}
}

func TestDeferRejectsLaterConsumeOfCapturedDescendant(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "field capture",
			src: `struct Pair:
    left: Int
    right: Int

func take(value: consume Int) -> Int:
    return value

func main() -> Int
uses io:
    let pair: Pair = Pair(left: 40, right: 2)
    defer:
        if pair.left == 40:
            print("field")
    let moved: Int = take(pair.left)
    return moved + pair.right
`,
			want: "defer cleanup captures value 'pair.left'",
		},
		{
			name: "whole struct capture",
			src: `struct Pair:
    left: Int
    right: Int

func take(value: consume Int) -> Int:
    return value

func use(pair: Pair) -> Int:
    return pair.left + pair.right

func main() -> Int
uses io:
    let pair: Pair = Pair(left: 40, right: 2)
    defer:
        if use(pair) == 42:
            print("whole")
    let moved: Int = take(pair.left)
    return moved + pair.right
`,
			want: "defer cleanup captures value 'pair'",
		},
		{
			name: "whole enum capture",
			src: `enum PairMsg:
    case both(Int, Int)
    case empty

func take(value: consume Int) -> Int:
    return value

func use(msg: PairMsg) -> Int:
    match msg:
    case PairMsg.both(left, right):
        return left + right
    case PairMsg.empty:
        return 0

func main() -> Int
uses io:
    let msg: PairMsg = PairMsg.both(40, 2)
    match msg:
    case PairMsg.both(left, right):
        defer:
            if use(msg) == 42:
                print("enum")
        let moved: Int = take(left)
        return moved + right
    case PairMsg.empty:
        return 1
`,
			want: "defer cleanup captures value 'msg'",
		},
		{
			name: "whole optional capture",
			src: `func take(value: consume Int) -> Int:
    return value

func use(maybe: Int?) -> Int:
    match maybe:
    case some(raw):
        return raw
    case none:
        return 0

func main() -> Int
uses io:
    let maybe: Int? = 42
    match maybe:
    case some(raw):
        defer:
            if use(maybe) == 42:
                print("optional")
        let moved: Int = take(raw)
        return moved
    case none:
        return 1
`,
			want: "defer cleanup captures value 'maybe'",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog, err := compiler.Parse([]byte(tc.src))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			_, err = compiler.Check(prog)
			if err == nil {
				t.Fatalf("expected semantic error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got: %v", tc.want, err)
			}
		})
	}
}

func TestDeferAllowsSiblingCaptureAfterDescendantConsume(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{
			name: "struct sibling",
			src: `struct Pair:
    left: Int
    right: Int

func take(value: consume Int) -> Int:
    return value

func main() -> Int
uses io:
    let pair: Pair = Pair(left: 40, right: 2)
    defer:
        if pair.right == 2:
            print("sibling")
    let moved: Int = take(pair.left)
    return moved + pair.right
`,
		},
		{
			name: "enum sibling payload alias",
			src: `enum PairMsg:
    case both(Int, Int)
    case empty

func take(value: consume Int) -> Int:
    return value

func main() -> Int
uses io:
    let msg: PairMsg = PairMsg.both(40, 2)
    match msg:
    case PairMsg.both(left, right):
        defer:
            if right == 2:
                print("sibling")
        let moved: Int = take(left)
        return moved + right
    case PairMsg.empty:
        return 1
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog, err := compiler.Parse([]byte(tc.src))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if _, err := compiler.Check(prog); err != nil {
				t.Fatalf("Check: %v", err)
			}
		})
	}
}

func TestDeferRejectsLaterActorTransferOfCapturedIsland(t *testing.T) {
	src := []byte(`enum MoveMsg:
    case take(island)

func worker() -> Int:
    return 0

func main() -> Int
uses actors, alloc, islands, mem:
    let peer: actor = core.spawn("worker")
    unsafe:
        var isl: island = core.island_new(16)
        defer:
            let _buf: []u8 = core.island_make_u8(isl, 1)
        let _sent: Int = core.send_typed(peer, MoveMsg.take(isl))
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected semantic error")
	}
	if !strings.Contains(err.Error(), "defer cleanup captures value 'isl'") {
		t.Fatalf("error = %v", err)
	}
}

func TestDeferBodyConsumeDoesNotPoisonPreCleanupReturn(t *testing.T) {
	src := []byte(`func take(x: consume Int) -> Int:
    return x

func main() -> Int:
    let a: Int = 1
    defer:
        let _done: Int = take(a)
    return a
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := compiler.Check(prog); err != nil {
		t.Fatalf("Check: %v", err)
	}
}

func TestDeferRejectsThrowInsideCleanup(t *testing.T) {
	src := []byte(`enum E:
    case bad

func main() -> Int throws E:
    defer:
        throw E.bad
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected semantic error")
	}
	if !strings.Contains(err.Error(), "throw is not allowed in defer") {
		t.Fatalf("error = %v", err)
	}
}

// ---- else_if_test.go ----

func TestBuildFlowElseIfSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int:
    let x: Int = 2
    if x == 1:
        return 1
    else if x == 2:
        return 42
    else:
        return 0
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildLegacyElseIfSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `fun main(): i32 {
  val x: i32 = 2
  if (x == 1) {
    return 1
  } else if (x == 2) {
    return 42
  } else {
    return 0
  }
}
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestFormatSourceElseIf(t *testing.T) {
	src := []byte(`func main() -> Int:
    let x: Int = 2
    if x == 1:
        return 1
    else:
        if x == 2:
            return 42
        else:
            return 0
`)
	got, err := compiler.FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `func main() -> Int:
    let x: Int = 2
    if x == 1:
        return 1
    else if x == 2:
        return 42
    else:
        return 0
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

// ---- epic04_semantics_test.go ----

func TestEpic04SemanticCheckerCorePositiveCase(t *testing.T) {
	prog, err := compiler.Parse([]byte(`
fun main(): i32 {
  let x: i32 = 41
  return x + 1
}
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if checked.MainName != "main" {
		t.Fatalf("main name = %q, want main", checked.MainName)
	}
	if len(checked.Funcs) != 1 {
		t.Fatalf("func count = %d, want 1", len(checked.Funcs))
	}
}

func TestEpic04SemanticCheckerCoreNegativePositionedDiagnostic(t *testing.T) {
	err := testkit.CheckProgram(`
fun main(): i32 {
  let x: i32 = true
  return 0
}
`)
	if err == nil {
		t.Fatalf("expected semantic error")
	}
	if !strings.Contains(err.Error(), "line 3:3: type mismatch: expected 'i32', got 'bool'") {
		t.Fatalf("error = %v", err)
	}
}

func TestEpic04SemanticCheckerCoreCrossModuleParity(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"engine/math.tetra": "module engine.math\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/main.tetra":    "module app.main\nimport engine.math as math\nfun main(): i32 {\n  return math.add_one(41)\n}\n",
	})

	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.tetra")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checkedWorld, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if checkedWorld.MainName != "app.main.main" {
		t.Fatalf("main name = %q, want app.main.main", checkedWorld.MainName)
	}

	singleProg, err := compiler.Parse([]byte(`
fun add_one(x: i32): i32 {
  return x + 1
}
fun main(): i32 {
  return add_one(41)
}
`))
	if err != nil {
		t.Fatalf("single parse: %v", err)
	}
	checkedSingle, err := compiler.Check(singleProg)
	if err != nil {
		t.Fatalf("single check: %v", err)
	}
	if checkedSingle.FuncSigs["add_one"].ReturnType != checkedWorld.FuncSigs["engine.math.add_one"].ReturnType {
		t.Fatalf("return types diverged between single-file and module-world checks")
	}
}

func TestEpic04SemanticCheckerCoreDisplayTextStability(t *testing.T) {
	err := testkit.CheckProgram(`
fun main(): bool {
  return true
}
`)
	if err == nil {
		t.Fatalf("expected semantic error")
	}
	if !strings.Contains(err.Error(), "main must return i32") {
		t.Fatalf("error = %v", err)
	}
}

func TestEpic04SemanticCheckerCoreBoundaryNilProgram(t *testing.T) {
	_, err := compiler.Check(nil)
	if err == nil {
		t.Fatalf("expected nil program error")
	}
	if err.Error() != "no program provided" {
		t.Fatalf("error = %q, want no program provided", err.Error())
	}
}

func TestEpic04ExpressionTypingPositiveAndInferenceCrossModule(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"engine/math.tetra": "module engine.math\nfun inc(x: i32): i32 {\n  return x + 1\n}\n",
		"app/main.tetra":    "module app.main\nimport engine.math as math\nfun main(): i32 {\n  let v = math.inc(1)\n  return v\n}\n",
	})

	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.tetra")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	mainIdx := -1
	for i := range checked.Funcs {
		if checked.Funcs[i].Name == "app.main.main" {
			mainIdx = i
			break
		}
	}
	if mainIdx < 0 {
		t.Fatalf("missing app.main.main")
	}
	if got := checked.Funcs[mainIdx].Locals["v"].TypeName; got != "i32" {
		t.Fatalf("local v type = %q, want i32", got)
	}
}

func TestEpic04ExpressionTypingNegativeDiagnostic(t *testing.T) {
	err := testkit.CheckProgram(`
fun main(): i32 {
  let ok: bool = true
  let x: i32 = ok + 1
  return x
}
`)
	if err == nil {
		t.Fatalf("expected semantic error")
	}
	if !strings.Contains(err.Error(), "arithmetic operators require i32/u8") {
		t.Fatalf("error = %v", err)
	}
}

func TestEpic04ExpressionTypingDisplayTextAndBoundary(t *testing.T) {
	err := testkit.CheckProgram(`
fun main(): i32 {
  let value = none
  return 0
}
`)
	if err == nil {
		t.Fatalf("expected inference boundary error")
	}
	if !strings.Contains(
		err.Error(),
		"cannot infer type from 'none'; add an optional type annotation",
	) {
		t.Fatalf("error = %v", err)
	}
}

func TestEpic04TypeModelPositiveOptionalSlots(t *testing.T) {
	prog, err := compiler.Parse([]byte(`
struct Box:
    value: Int?

func main() -> Int:
    let box: Box = Box(value: none)
    return 0
`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	opt := checked.Types["i32?"]
	if opt == nil {
		t.Fatalf("missing optional i32 type")
	}
	if opt.SlotCount != 2 {
		t.Fatalf("optional slot count = %d, want 2", opt.SlotCount)
	}
}

func TestEpic04TypeModelNegativeArrayBoundary(t *testing.T) {
	err := testkit.CheckProgram(`
func main() -> Int:
    let xs: [0]Int = 0
    return 0
`)
	if err == nil {
		t.Fatalf("expected array boundary error")
	}
	if !strings.Contains(err.Error(), "array size must be positive constant") {
		t.Fatalf("error = %v", err)
	}
}

func TestEpic04TypeModelCrossModuleAndDisplayText(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"engine/types.tetra": "module engine.types\nstruct Vec { x: i32, y: i32 }\n",
		"app/main.tetra":     "module app.main\nimport engine.types as t\nfun consume(v: t.Vec): i32 {\n  return v.x + v.y\n}\nfun main(): i32 {\n  return 0\n}\n",
	})
	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.tetra")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := compiler.CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	if got := checked.FuncSigs["app.main.consume"].ParamTypes[0]; got != "engine.types.Vec" {
		t.Fatalf("consume param = %q, want engine.types.Vec", got)
	}

	err = testkit.CheckProgram(`
func main() -> Int:
    let b: Byte = true
    return 0
`)
	if err == nil {
		t.Fatalf("expected type mismatch")
	}
	if !strings.Contains(err.Error(), "expected 'u8', got 'bool'") {
		t.Fatalf("error = %v", err)
	}
}

func TestEpic04LocalInferenceNegativeAndDisplayText(t *testing.T) {
	err := testkit.CheckProgram(`
fun main(): i32 {
  let x = missing(1)
  return x
}
`)
	if err == nil {
		t.Fatalf("expected unknown function inference error")
	}
	if !strings.Contains(err.Error(), "cannot infer type for 'x': unknown function 'missing'") {
		t.Fatalf("error = %v", err)
	}
}

// ---- extensions_test.go ----

func TestExtensionParseCheckAndLower(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int
    y: Int

extension Vec2:
    func sum(self: Vec2) -> Int:
        return self.x + self.y

func main() -> Int:
    let v: Vec2 = Vec2(x: 40, y: 2)
    return Vec2.sum(v)
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(prog.Extensions) != 1 {
		t.Fatalf("extensions = %d", len(prog.Extensions))
	}
	if len(prog.Funcs) != 2 || prog.Funcs[0].Name != "Vec2.sum" {
		t.Fatalf("funcs = %#v", prog.Funcs)
	}
	checked, err := compiler.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, ok := checked.FuncSigs["Vec2.sum"]; !ok {
		t.Fatalf("missing extension method signature")
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestExtensionMethodCanReturnOptionalPayload(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

extension Vec2:
    func nonzero(self: Vec2) -> Int?:
        if self.x == 0:
            return none
        return self.x

func main() -> Int:
    let v: Vec2 = Vec2(x: 42)
    let maybe: Int? = Vec2.nonzero(v)
    if let x = maybe:
        return x
    else:
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
	if got := checked.FuncSigs["Vec2.nonzero"].ReturnType; got != "i32?" {
		t.Fatalf("Vec2.nonzero return type = %q, want i32?", got)
	}
	if _, err := compiler.Lower(checked); err != nil {
		t.Fatalf("Lower: %v", err)
	}
}

func TestExtensionNoLongerPlannedDiagnostic(t *testing.T) {
	_, err := compiler.Parse([]byte("extension Vec2:\n"))
	if err == nil {
		t.Fatalf("expected block error, not silent success")
	}
	if strings.Contains(err.Error(), "planned feature 'extension'") {
		t.Fatalf("extension still reports planned diagnostic: %v", err)
	}
}

func TestExtensionRejectsDuplicateMethodName(t *testing.T) {
	src := []byte(`
struct Vec2:
    x: Int

extension Vec2:
    func sum(self: Vec2) -> Int:
        return self.x
    func sum(self: Vec2) -> Int:
        return self.x

func main() -> Int:
    return 0
`)
	prog, err := compiler.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = compiler.Check(prog)
	if err == nil {
		t.Fatalf("expected duplicate extension method error")
	}
	if !strings.Contains(err.Error(), "duplicate function 'Vec2.sum'") {
		t.Fatalf("error = %v", err)
	}
}

func TestImportedExtensionStaticCallAndDocsSurface(t *testing.T) {
	files := map[string]string{
		"engine/vec.tetra": `module engine.vec
struct Vec2:
    x: Int
    y: Int

extension Vec2:
    func sum(self: Vec2) -> Int:
        return self.x + self.y
`,
		"app/main.tetra": `module app.main
import engine.vec as vec

func main() -> Int:
    let v: vec.Vec2 = vec.Vec2(x: 40, y: 2)
    return vec.Vec2.sum(v)
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
	if _, ok := checked.FuncSigs["engine.vec.Vec2.sum"]; !ok {
		t.Fatalf("missing imported extension method signature: %#v", checked.FuncSigs)
	}
	if _, err := compiler.LowerModules(checked); err != nil {
		t.Fatalf("LowerModules: %v", err)
	}

	docs, err := compiler.GenerateAPIDocs(
		[]string{filepath.Join(tmp, filepath.FromSlash("engine/vec.tetra"))},
	)
	if err != nil {
		t.Fatalf("GenerateAPIDocs: %v", err)
	}
	out := string(docs)
	for _, want := range []string{
		"### Extensions",
		"- `Vec2`",
		"`func Vec2.sum(self: Vec2) -> i32`",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("docs missing %q:\n%s", want, out)
		}
	}
}

// ---- for_collection_test.go ----

func TestBuildForCollectionSliceSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int
uses alloc, islands, mem:
    var total: Int = 0
    island(128) as isl:
        var xs: []i32 = core.island_make_i32(isl, 3)
        xs[0] = 10
        xs[1] = 20
        xs[2] = 12
        for x in xs:
            total = total + x
    return total
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildForCollectionStringSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int:
    var total: Int = 0
    let text: String = "*"
    for ch in text:
        total = total + ch
    return total
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildForCollectionU8SliceSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int
uses alloc, islands, mem:
    var total: Int = 0
    island(128) as isl:
        var bytes: []u8 = core.island_make_u8(isl, 2)
        bytes[0] = 40
        bytes[1] = 2
        for b in bytes:
            total = total + b
    return total
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildForCollectionU16SliceSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int
uses alloc, islands, mem:
    var total: Int = 0
    island(128) as isl:
        var nums: []u16 = core.island_make_u16(isl, 2)
        nums[0] = 40
        nums[1] = 2
        for n in nums:
            total = total + n
    return total
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildForCollectionBoolSliceSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int
uses alloc, islands, mem:
    var total: Int = 0
    island(128) as isl:
        var flags: []bool = core.island_make_bool(isl, 3)
        flags[0] = true
        flags[1] = false
        flags[2] = true
        for flag in flags:
            if flag:
                total = total + 21
    return total
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

// ---- loop_control_test.go ----

func TestBuildWhileBreakContinueSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int:
    var i: Int = 0
    var total: Int = 0
    while i < 10:
        i = i + 1
        if i == 3:
            continue
        if i == 6:
            break
        total = total + i
    return total
`
	_, code := buildAndRun(t, src)
	if code != 12 {
		t.Fatalf("exit code mismatch: got %d, want 12", code)
	}
}

func TestBuildRangeForBreakContinueSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int:
    var total: Int = 0
    for i in 0..<10:
        if i == 2:
            continue
        if i == 5:
            break
        total = total + i
    return total
`
	_, code := buildAndRun(t, src)
	if code != 8 {
		t.Fatalf("exit code mismatch: got %d, want 8", code)
	}
}

func TestBuildCollectionForBreakContinueSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int:
    var total: Int = 0
    let text: String = "*!+"
    for ch in text:
        if ch == 33:
            continue
        if ch == 43:
            break
        total = total + ch
    return total
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildUnaryBangSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func main() -> Int:
    let off: Bool = false
    if !off && !0:
        return 42
    return 1
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildLogicalShortCircuitSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `func mark() -> Bool:
    return true

func main() -> Int:
    let left: Bool = false
    if left && mark():
        return 1
    if left || mark():
        return 42
    return 2
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

func TestBuildNestedControlFlowSmoke(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	src := `enum Mode:
    case fast
    case slow

func classify(x: Int) -> Int:
    if x > 10:
        return 10
    return x

func main() -> Int:
    var total: Int = 0
    for i in 0..<6:
        var j: Int = 0
        while j < 4:
            j += 1
            if i == 1:
                continue
            else if i == 4 && j == 2:
                break
            else:
                total += classify(i + j)
        if total > 35:
            break

    match Mode.fast:
    case Mode.fast:
        total += 1
    case Mode.slow:
        total += 2
    case _:
        total += 3

    if total == 51:
        return 42
    return total
`
	_, code := buildAndRun(t, src)
	if code != 42 {
		t.Fatalf("exit code mismatch: got %d, want 42", code)
	}
}

// ---- metadata_assignment_test.go ----

func TestRepresentationMetadataAssignmentDiagnosticsNameFieldAndSafeNonAssignable(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "slice len",
			src: `
func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(2)
    xs.len = 9
    return 0
`,
			want: "representation metadata field 'len' is not user-assignable in safe code",
		},
		{
			name: "slice reserved owner",
			src: `
func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(2)
    xs.owner_id = 9
    return 0
`,
			want: "representation metadata field 'owner_id' is not user-assignable in safe code",
		},
		{
			name: "indexed through metadata",
			src: `
func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(2)
    xs.ptr[0] = 1
    return 0
`,
			want: "representation metadata field 'ptr' is not user-assignable in safe code",
		},
		{
			name: "generic wrapper",
			src: `
struct Box<T>:
    value: T

func main() -> Int
uses alloc, mem:
    var box: Box<[]u8> = Box<[]u8>{value: make_u8(2)}
    box.value.len = 9
    return 0
`,
			want: "representation metadata field 'len' is not user-assignable in safe code",
		},
		{
			name: "string metadata",
			src: `
func main() -> Int:
    var text: String = "*"
    text.ptr = 0
    return 0
`,
			want: "representation metadata field 'ptr' is not user-assignable in safe code",
		},
		{
			name: "fixed array metadata",
			src: `
func probe(seed: [2]Int) -> Int:
    var xs: [2]Int = seed
    xs.len = 7
    return 0

func main() -> Int:
    return 0
`,
			want: "representation metadata field 'len' is not user-assignable in safe code",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testkit.RequireCheckErrorContains(t, tt.src, tt.want)
		})
	}
}

func TestRepresentationMetadataAssignmentRejectsImportedSliceField(t *testing.T) {
	requireCheckWorldFilesErrorContains(t, map[string]string{
		"src/lib/model.t4": `module lib.model

pub struct BufferBox:
    bytes: []u8
`,
		"src/app/main.t4": `module app.main
import lib.model as model

func main() -> Int
uses alloc, mem:
    var box: model.BufferBox = model.BufferBox(bytes: make_u8(2))
    box.bytes.len = 9
    return 0
`,
	}, "src/app/main.t4", "representation metadata field 'len' is not user-assignable in safe code")
}

// ---- module_boundary_test.go ----

func TestModuleBoundaryAllowsPublicImportedFunction(t *testing.T) {
	tmp := t.TempDir()
	writeCompilerModuleFiles(t, tmp, map[string]string{
		"engine/math.t4": `module engine.math
pub func add(a: Int, b: Int) -> Int:
    return a + b
func hidden() -> Int:
    return 99
`,
		"app/main.t4": `module app.main
import engine.math as math
func main() -> Int:
    return math.add(40, 2)
`,
	})

	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.t4")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
}

func TestModuleBoundaryRejectsPrivateImportedFunction(t *testing.T) {
	tmp := t.TempDir()
	writeCompilerModuleFiles(t, tmp, map[string]string{
		"engine/math.t4": `module engine.math
pub func add(a: Int, b: Int) -> Int:
    return a + b
func hidden() -> Int:
    return 99
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

func TestSelectiveImportResolvesPublicFunctionAndType(t *testing.T) {
	tmp := t.TempDir()
	writeCompilerModuleFiles(t, tmp, map[string]string{
		"engine/math.t4": `module engine.math
pub struct Vec { x: Int }
pub func add(a: Int, b: Int) -> Int:
    return a + b
`,
		"app/main.t4": `module app.main
import engine.math.{add, Vec}
struct Holder { value: Vec }
func main() -> Int:
    return add(40, 2)
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
	if got := checked.FuncSigs["app.main.main"].ReturnType; got != "i32" {
		t.Fatalf("main return = %q, want i32", got)
	}
	if _, ok := checked.Types["engine.math.Vec"]; !ok {
		t.Fatalf("missing selected imported type engine.math.Vec")
	}
}

func TestSelectiveImportRejectsDuplicateImportedSymbol(t *testing.T) {
	tmp := t.TempDir()
	writeCompilerModuleFiles(t, tmp, map[string]string{
		"a/one.t4": `module a.one
pub func pick() -> Int:
    return 1
`,
		"b/two.t4": `module b.two
pub func pick() -> Int:
    return 2
`,
		"app/main.t4": `module app.main
import a.one.{pick}
import b.two.{pick}
func main() -> Int:
    return pick()
`,
	})

	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.t4")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	_, err = compiler.CheckWorld(world)
	if err == nil {
		t.Fatalf("expected duplicate selective import diagnostic")
	}
	if !strings.Contains(err.Error(), "duplicate import alias 'pick'") {
		t.Fatalf("error = %v", err)
	}
}

func TestPublicReExportSupportsSelectiveImport(t *testing.T) {
	tmp := t.TempDir()
	writeCompilerModuleFiles(t, tmp, map[string]string{
		"math/core.t4": `module math.core
pub func add(a: Int, b: Int) -> Int:
    return a + b
`,
		"math/prelude.t4": `module math.prelude
pub import math.core.{add}
`,
		"app/main.t4": `module app.main
import math.prelude.{add}
func main() -> Int:
    return add(40, 2)
`,
	})

	world, err := compiler.LoadWorld(filepath.Join(tmp, filepath.FromSlash("app/main.t4")))
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
}

func writeCompilerModuleFiles(t *testing.T, base string, files map[string]string) {
	t.Helper()
	for rel, src := range files {
		path := filepath.Join(base, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
}

// ---- plan250_semantics_test.go ----

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
	for _, forbidden := range []string{
		"`id: Int`",
		"`payload: String`",
		"`byte: UInt8`",
		"`ok: Bool`",
		"-> String",
	} {
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
		t.Fatalf(
			"specialization count changed: first=%d second=%d",
			len(first.FuncSigs),
			len(second.FuncSigs),
		)
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
		t.Fatalf(
			"missing cross-module specialization lib.generic.id__T_i32 in %#v",
			checked.FuncSigs,
		)
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
			want: ("generic closure captures are not supported by the production " +
				"fnptr ABI; use a non-generic closure or pass captured state explicitly"),
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
			want: ("throwing function symbol 'fail' cannot be used as callback " +
				"argument; callback fnptr ABI requires the parameter's declared throws " +
				"type to match"),
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
	if len(withChecked.Funcs) != len(withoutChecked.Funcs) ||
		len(withChecked.Types) != len(withoutChecked.Types) {
		t.Fatalf(
			("capsule changed semantic function/type counts: with funcs=%d " +
				"types=%d without funcs=%d types=%d"),
			len(withChecked.Funcs),
			len(withChecked.Types),
			len(withoutChecked.Funcs),
			len(withoutChecked.Types),
		)
	}
	if withChecked.MainName != withoutChecked.MainName || withIR.MainName != withoutIR.MainName {
		t.Fatalf(
			"capsule changed main metadata: checked %q/%q ir %q/%q",
			withChecked.MainName,
			withoutChecked.MainName,
			withIR.MainName,
			withoutIR.MainName,
		)
	}
	withMain := findIRFunc(t, withIR.Funcs, "main")
	withoutMain := findIRFunc(t, withoutIR.Funcs, "main")
	if withMain.ParamSlots != withoutMain.ParamSlots ||
		withMain.LocalSlots != withoutMain.LocalSlots ||
		withMain.ReturnSlots != withoutMain.ReturnSlots ||
		len(withMain.Instrs) != len(withoutMain.Instrs) {
		t.Fatalf("capsule changed lowered main shape: with=%#v without=%#v", withMain, withoutMain)
	}
}
