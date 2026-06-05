package compiler_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
)

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("repo root %s is missing go.mod: %v", root, err)
	}
	return root
}

func TestFormatterPublicAPITestsLiveInFrontend(t *testing.T) {
	root := repoRoot(t)
	oldPath := filepath.Join(root, "compiler", "format_test.go")
	if _, err := os.Stat(oldPath); err == nil {
		t.Fatalf("%s must move to compiler/tests/frontend/format_test.go", oldPath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", oldPath, err)
	}
	newPath := filepath.Join(root, "compiler", "tests", "frontend", "format_test.go")
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("%s must exist: %v", newPath, err)
	}
}

func TestFormatSourceFlowMVP(t *testing.T) {
	src := []byte(`func main() -> Int
uses mem, io:
    print("hi\n")
    return 0
`)
	got, err := compiler.FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `func main() -> Int
uses io, mem:
    print("hi\n")
    return 0
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestFormatSourceDeferBlock(t *testing.T) {
	src := []byte(`func main() -> Int
uses io:
    defer:
        print("cleanup\n")
    return 0
`)
	got, err := compiler.FormatSource(src, "defer.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `func main() -> Int
uses io:
    defer:
        print("cleanup\n")
    return 0
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestFormatSourceStateAndView(t *testing.T) {
	src := []byte(`state CounterState:
    var count: Int = 0
    val title: String = "Wave 9"

view CounterView(state: CounterState):
    bind countValue: Int = state.count
    event click -> increment
    command increment:
        state.count = state.count + 1
    style width: Int = 320
    accessibility label: String = "Increment"
`)
	got, err := compiler.FormatSource(src, "ui.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	if !strings.Contains(string(got), "view CounterView(state: CounterState):") {
		t.Fatalf("formatted source missing view header:\n%s", string(got))
	}
	if !strings.Contains(string(got), "accessibility label: String = \"Increment\"") {
		t.Fatalf("formatted source missing accessibility entry:\n%s", string(got))
	}
}

func TestFormatSourceGenericStructDeclarationRoundTrip(t *testing.T) {
	src := []byte(`struct Box<T>:
    value: T
`)
	got, err := compiler.FormatSource(src, "generic_struct.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `struct Box<T>:
    value: T
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestFormatSourcePreservesReprCStructDeclaration(t *testing.T) {
	src := []byte(`repr(C) struct Header:
    tag: c_int
    ptr: ptr
`)
	got, err := compiler.FormatSource(src, "repr_c_struct.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `repr(C) struct Header:
    tag: c_int
    ptr: ptr
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
	again, err := compiler.FormatSource(got, "repr_c_struct.tetra")
	if err != nil {
		t.Fatalf("FormatSource again: %v", err)
	}
	if string(again) != want {
		t.Fatalf("format not idempotent:\n%s", string(again))
	}
}

func TestFormatSourceGenericStructConstructorAndTypeRef(t *testing.T) {
	src := []byte(`struct Box<T>:
    value: T

func main() -> Int:
    let b: Box<Int> = Box<Int>{value: 42}
    return b.value
`)
	got, err := compiler.FormatSource(src, "generic_struct_constructor.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `struct Box<T>:
    value: T

func main() -> Int:
    let b: Box<Int> = Box<Int>(value: 42)
    return b.value
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestFormatSourcePreservesFlowLineComments(t *testing.T) {
	src := []byte(`// module note
func main() -> Int:
    // before local
    let ok: Bool = true
    if ok:
        // inside if
        return 0
    // after if
    return 1
`)
	got, err := compiler.FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `// module note
func main() -> Int:
    // before local
    let ok: Bool = true
    if ok:
        // inside if
        return 0
    // after if
    return 1
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestFormatSourcePreservesFlowBlockComments(t *testing.T) {
	src := []byte(`/* module note */
func main() -> Int:
    /* before return */
    return 0
`)
	got, err := compiler.FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `/* module note */
func main() -> Int:
    /* before return */
    return 0
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestFormatSourcePreservesDocAndUIComments(t *testing.T) {
	src := []byte(`/// state model
state CounterState:
    // primary count
    var count: Int = 0

/// counter view
view CounterView(state: CounterState):
    // display binding
    bind value: Int = state.count
    event click -> increment
    command increment:
        // command body
        state.count = state.count + 1
    /* accessibility copy */
    accessibility label: String = "Increment"
`)
	got, err := compiler.FormatSource(src, "ui.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	for _, want := range []string{
		"/// state model",
		"    // primary count",
		"/// counter view",
		"    // display binding",
		"        // command body",
		"    /* accessibility copy */",
	} {
		if !strings.Contains(string(got), want) {
			t.Fatalf("formatted source missing %q:\n%s", want, string(got))
		}
	}
}

func TestFormatSourcePreservesExportAttributes(t *testing.T) {
	src := []byte(`@export("__tetra_entry")
fun tetra_entry(): i32 {
    return 0
}
`)
	got, err := compiler.FormatSource(src, "runtime.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `@export("__tetra_entry")
func tetra_entry() -> i32:
    return 0
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestFormatSourceLegacyMigrationSurfaceIsCanonicalAndIdempotent(t *testing.T) {
	src := []byte(`fun main(): i32 {
    return 0
}
`)
	once, err := compiler.FormatSource(src, "legacy.tetra")
	if err != nil {
		t.Fatalf("FormatSource once: %v", err)
	}
	want := `func main() -> i32:
    return 0
`
	if string(once) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(once), want)
	}
	twice, err := compiler.FormatSource(once, "legacy.tetra")
	if err != nil {
		t.Fatalf("FormatSource twice: %v", err)
	}
	if string(twice) != string(once) {
		t.Fatalf("format not idempotent:\nonce:\n%s\ntwice:\n%s", string(once), string(twice))
	}
}

func TestFormatSourceCommentPreservationIsIdempotent(t *testing.T) {
	src := []byte(`// suite
test "math":
    // expected arithmetic
    expect 40 + 2 == 42
`)
	once, err := compiler.FormatSource(src, "math_test.tetra")
	if err != nil {
		t.Fatalf("FormatSource once: %v", err)
	}
	twice, err := compiler.FormatSource(once, "math_test.tetra")
	if err != nil {
		t.Fatalf("FormatSource twice: %v", err)
	}
	if string(twice) != string(once) {
		t.Fatalf("format not idempotent:\nonce:\n%s\ntwice:\n%s", string(once), string(twice))
	}
}

func TestFormatSourceRejectsInlineCommentsConservatively(t *testing.T) {
	_, err := compiler.FormatSource([]byte("func main() -> Int:\n    return 0 // trailing\n"), "main.tetra")
	if err == nil {
		t.Fatalf("expected comment-preservation diagnostic")
	}
	if !strings.Contains(err.Error(), "inline comments are not supported") {
		t.Fatalf("error = %v", err)
	}
}

func TestFormatSourceInlineCommentDiagnosticHasLocation(t *testing.T) {
	_, err := compiler.FormatSource([]byte("func main() -> Int:\n    return 0 // trailing\n"), "main.tetra")
	if err == nil {
		t.Fatalf("expected comment-preservation diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != "TETRA_FMT001" || diag.File != "main.tetra" || diag.Line != 2 || diag.Column != 14 {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestFormatSourceMalformedInputDiagnosticHasStableLocation(t *testing.T) {
	_, err := compiler.FormatSource([]byte("func main() -> Int:\n\treturn 0\n"), "tabbed.tetra")
	if err == nil {
		t.Fatalf("expected malformed-input diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != "TETRA0001" || diag.File != "tabbed.tetra" || diag.Line != 2 || diag.Column != 1 || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
	if !strings.Contains(diag.Message, "tabs are not supported") {
		t.Fatalf("diagnostic message = %q", diag.Message)
	}
}

func TestFormatSourcePreservesCommentAfterSingleLineUsesHeader(t *testing.T) {
	src := []byte(`func main() -> Int uses io:
    // before return
    return 0
`)
	got, err := compiler.FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `func main() -> Int
uses io:
    // before return
    return 0
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestFormatSourceExpressionBodiedFunction(t *testing.T) {
	src := []byte(`func add(a: Int, b: Int) -> Int = a + b
func main() -> Int = add(40, 2)
`)
	got, err := compiler.FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `func add(a: Int, b: Int) -> Int:
    return a + b

func main() -> Int:
    return add(40, 2)
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestFormatSourceCallArgumentLabels(t *testing.T) {
	src := []byte(`func add(a: Int, b: Int) -> Int:
    return a + b

func main() -> Int:
    return add(a: 40, b: 2)
`)
	got, err := compiler.FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `func add(a: Int, b: Int) -> Int:
    return a + b

func main() -> Int:
    return add(a: 40, b: 2)
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestFormatSourceCollectionFor(t *testing.T) {
	src := []byte(`func main() -> Int:
    var total: Int = 0
    let text: String = "*"
    for ch in text:
        total = total + ch
    return total
`)
	got, err := compiler.FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `func main() -> Int:
    var total: Int = 0
    let text: String = "*"
    for ch in text:
        total = total + ch
    return total
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestFormatSourceBreakContinue(t *testing.T) {
	src := []byte(`func main() -> Int:
    var i: Int = 0
    while i < 10:
        i = i + 1
        if i == 3:
            continue
        if i == 6:
            break
    return i
`)
	got, err := compiler.FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `func main() -> Int:
    var i: Int = 0
    while i < 10:
        i = i + 1
        if i == 3:
            continue
        if i == 6:
            break
    return i
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestFormatSourceUnaryBang(t *testing.T) {
	src := []byte(`func main() -> Int:
    let off: Bool = false
    if !off:
        return 42
    return 1
`)
	got, err := compiler.FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `func main() -> Int:
    let off: Bool = false
    if !off:
        return 42
    return 1
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestFormatSourceCompoundAssignment(t *testing.T) {
	src := []byte(`func main() -> Int:
    var x: Int = 4
    x += 3
    x *= 6
    x -= 0
    x /= 1
    x %= 100
    return x
`)
	got, err := compiler.FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `func main() -> Int:
    var x: Int = 4
    x += 3
    x *= 6
    x -= 0
    x /= 1
    x %= 100
    return x
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestFormatSourceCompoundAssignmentTargets(t *testing.T) {
	src := []byte(`struct Box:
    x: Int

func main() -> Int:
    var b: Box = Box(x: 40)
    b.x += 2
    var xs: []i32 = make_i32(1)
    xs[0] += b.x
    return xs[0]
`)
	got, err := compiler.FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `struct Box:
    x: Int

func main() -> Int:
    var b: Box = Box(x: 40)
    b.x += 2
    var xs: []i32 = make_i32(1)
    xs[0] += b.x
    return xs[0]
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestFormatSourcePreservesImplDeclarations(t *testing.T) {
	src := []byte(`struct Vec2:
    x: Int

protocol Renderable:
    func draw(self: Vec2) -> Int

extension Vec2:
    func draw(self: Vec2) -> Int:
        return self.x

impl Vec2: Renderable

func main() -> Int:
    return Vec2.draw(Vec2(x: 42))
`)
	got, err := compiler.FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `struct Vec2:
    x: Int

protocol Renderable:
    func draw(self: Vec2) -> Int

extension Vec2:
    func draw(self: Vec2) -> Int:
        return self.x

impl Vec2: Renderable

func main() -> Int:
    return Vec2.draw(Vec2(x: 42))
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestFormatSourcePreservesActorDeclarations(t *testing.T) {
	src := []byte(`actor Worker:
    val id: Int = 7
    func run() -> Int
    uses actors:
        let me: actor = core.self()
        return 0

func main() -> Int
uses actors:
    let peer: actor = core.spawn("Worker.run")
    return 0
`)
	once, err := compiler.FormatSource(src, "actor.tetra")
	if err != nil {
		t.Fatalf("FormatSource once: %v", err)
	}
	twice, err := compiler.FormatSource(once, "actor.tetra")
	if err != nil {
		t.Fatalf("FormatSource twice: %v\nonce:\n%s", err, string(once))
	}
	if string(twice) != string(once) {
		t.Fatalf("format not idempotent:\nonce:\n%s\ntwice:\n%s", string(once), string(twice))
	}
	if !strings.Contains(string(once), "actor Worker:\n    val id: Int = 7\n    func run() -> Int\n    uses actors:") {
		t.Fatalf("formatted actor declaration missing:\n%s", string(once))
	}
}

func TestFormatSourcePreservesSemanticClauses(t *testing.T) {
	src := []byte(`func main() -> Int noalloc noblock realtime nothrow budget(10):
    return 0
`)
	got, err := compiler.FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `func main() -> Int
noalloc
noblock
realtime
nothrow
budget(10):
    return 0
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestFormatSourceOwnershipMarkersAreCanonicalAndIdempotent(t *testing.T) {
	src := []byte(`protocol BufferOps:
    func update(src: borrow []u8, dst: inout []u8, tmp: consume []u8) -> Int

closure local(read: borrow Int, write: inout Int, taken: consume Int) -> Int:
    write = write + read + taken
    return write

func mix(a: borrow Int, b: inout Int, c: consume Int, cb: borrow fn(Int) -> Int) -> Int:
    return cb(a) + b + c
`)
	once, err := compiler.FormatSource(src, "ownership.tetra")
	if err != nil {
		t.Fatalf("FormatSource once: %v", err)
	}
	twice, err := compiler.FormatSource(once, "ownership.tetra")
	if err != nil {
		t.Fatalf("FormatSource twice: %v", err)
	}
	if string(twice) != string(once) {
		t.Fatalf("format not idempotent:\nonce:\n%s\ntwice:\n%s", string(once), string(twice))
	}
	for _, want := range []string{
		"func update(src: borrow []u8, dst: inout []u8, tmp: consume []u8) -> Int",
		"closure local(read: borrow Int, write: inout Int, taken: consume Int) -> Int:",
		"func mix(a: borrow Int, b: inout Int, c: consume Int, cb: borrow fn(Int) -> Int) -> Int:",
	} {
		if !strings.Contains(string(once), want) {
			t.Fatalf("formatted source missing %q:\n%s", want, string(once))
		}
	}
}

func TestFormatSourceClosureLiteralIsIdempotent(t *testing.T) {
	src := []byte(`func main() -> Int:
    let f: ptr = fn(x: Int) -> Int:
        return x
    return 0
`)
	once, err := compiler.FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource once: %v", err)
	}
	if strings.Contains(string(once), "<expr>") {
		t.Fatalf("formatted source lost closure expression:\n%s", string(once))
	}
	twice, err := compiler.FormatSource(once, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource twice: %v", err)
	}
	if string(twice) != string(once) {
		t.Fatalf("format not idempotent:\nonce:\n%s\ntwice:\n%s", string(once), string(twice))
	}
}

func TestFormatSourceNestedMultiStmtClosureIsIdempotent(t *testing.T) {
	src := []byte(`func call(f: ptr) -> Int:
    return 0

func main() -> Int:
    return call(fn(x: Int) -> Int:
        let y: Int = x
        return y
    )
`)
	once, err := compiler.FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource once: %v", err)
	}
	if strings.Contains(string(once), "<expr>") {
		t.Fatalf("formatted source lost nested closure expression:\n%s", string(once))
	}
	if !strings.Contains(string(once), "func __closure_") {
		t.Fatalf("formatted source missing synthetic closure declaration:\n%s", string(once))
	}
	twice, err := compiler.FormatSource(once, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource twice: %v", err)
	}
	if string(twice) != string(once) {
		t.Fatalf("format not idempotent:\nonce:\n%s\ntwice:\n%s", string(once), string(twice))
	}
}

func TestFormatSourcePreservesCommentPlacementAfterExpressionBodiedFunction(t *testing.T) {
	src := []byte(`func add(a: Int, b: Int) -> Int = a + b
// keep with main
func main() -> Int = add(a: 40, b: 2)
`)
	got, err := compiler.FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `func add(a: Int, b: Int) -> Int:
    return a + b

// keep with main
func main() -> Int:
    return add(a: 40, b: 2)
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestFormatSourcePreservesCommentPlacementAfterSemanticClauseExpansion(t *testing.T) {
	src := []byte(`func worker() -> Int noalloc budget(8):
    return 1
// keep with main
func main() -> Int:
    return worker()
`)
	got, err := compiler.FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `func worker() -> Int
noalloc
budget(8):
    return 1

// keep with main
func main() -> Int:
    return worker()
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestFormatSourcePreservesCommentPlacementAfterClosureExpansion(t *testing.T) {
	src := []byte(`func main() -> Int:
    let f: ptr = fn(x: Int) -> Int = x
    // keep with return
    return 0
`)
	got, err := compiler.FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `func main() -> Int:
    let f: ptr = fn(x: Int) -> Int:
        return x
    // keep with return
    return 0
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
}

func TestFormatSourceV1SurfaceIdempotentWithComments(t *testing.T) {
	src := []byte(`// module docs
module app.main

// io import
import app.io as io

/* enum docs */
enum Mode:
    case fast
    case slow

// struct docs
struct Box:
    value: Int

protocol Runner:
    func run(self: Box) -> Int

// extension docs
extension Box:
    func run(self: Box) -> Int:
        return self.value

impl Box: Runner

// globals
const answer: Int = 42

func worker(task: Int) -> Int noalloc budget(8):
    let f: ptr = fn(x: Int) -> Int:
        return x + answer
    if task > 0:
        return task
    return answer

// entry
func main() -> Int uses io:
    let mode: Mode = Mode.fast
    var box: Box = Box(value: answer)
    if mode == Mode.fast:
        return worker(task: box.value)
    return 0
`)
	once, err := compiler.FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource once: %v", err)
	}
	twice, err := compiler.FormatSource(once, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource twice: %v", err)
	}
	if string(twice) != string(once) {
		t.Fatalf("format not idempotent:\nonce:\n%s\ntwice:\n%s", string(once), string(twice))
	}
	for _, expected := range []string{
		"// module docs",
		"/* enum docs */",
		"// extension docs",
		"// entry",
		"noalloc",
		"budget(8):",
		"fn(x: Int) -> Int:",
	} {
		if !strings.Contains(string(once), expected) {
			t.Fatalf("formatted source missing %q:\n%s", expected, string(once))
		}
	}
}

func TestFormatSourceFlowFullSurfaceIdempotent(t *testing.T) {
	src := []byte(`module app.main

import app.io as io

enum Mode:
    case fast
    case slow

enum ReadError:
    case eof

struct Box:
    value: Int

protocol Runner:
    func run(self: Box) -> Int

extension Box:
    func run(self: Box) -> Int:
        return self.value

impl Box: Runner

const answer: Int = 42

func read(flag: Bool) -> Int throws ReadError:
    if flag:
        return answer
    throw ReadError.eof

async func worker(flag: Bool) -> Int throws ReadError uses runtime:
    return try read(flag)

async func caller(flag: Bool) -> Int throws ReadError:
    return await worker(flag)

func main() -> Int uses io, runtime:
    let maybe: Int? = none
    if let value = maybe:
        return value

    var total: Int = 0
    for i in 0..<3:
        total += i

    let text: String = "*"
    for ch in text:
        total += ch

    while total < answer:
        if total == 1:
            continue
        if total > 8:
            break
        total += 1

    match Mode.fast:
    case Mode.fast:
        total += 1
    case Mode.slow:
        total += 2
    case _:
        total += 3

    unsafe:
        let mem: cap.mem = core.cap_mem()
        total += core.load_i32(core.store_i32(core.alloc_bytes(4), total, mem), mem)

    island(64) as isl:
        var buf: []UInt8 = core.island_make_u8(isl, 1)
        buf[0] = 1

    return total

test "math":
    expect 40 + 2 == 42
`)

	once, err := compiler.FormatSource(src, "full_surface.tetra")
	if err != nil {
		t.Fatalf("FormatSource once: %v", err)
	}
	twice, err := compiler.FormatSource(once, "full_surface.tetra")
	if err != nil {
		t.Fatalf("FormatSource twice: %v", err)
	}
	if string(twice) != string(once) {
		t.Fatalf("format not idempotent:\nonce:\n%s\ntwice:\n%s", string(once), string(twice))
	}
	for _, expected := range []string{
		"module app.main",
		"async func worker(flag: Bool) -> Int throws ReadError",
		"return await worker(flag)",
		"if let value = maybe:",
		"for i in 0..<3:",
		"for ch in text:",
		"match Mode.fast:",
		"unsafe:",
		"island(64) as isl:",
		"test \"math\":",
	} {
		if !strings.Contains(string(once), expected) {
			t.Fatalf("formatted source missing %q:\n%s", expected, string(once))
		}
	}
}

func TestFormatSourceExamplesAreIdempotent(t *testing.T) {
	examplesDir := filepath.Join(repoRoot(t), "examples")
	err := filepath.WalkDir(examplesDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".tetra" {
			return nil
		}
		t.Run(filepath.ToSlash(path), func(t *testing.T) {
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile: %v", err)
			}
			once, err := compiler.FormatSource(src, filepath.ToSlash(path))
			if err != nil {
				t.Fatalf("FormatSource once: %v", err)
			}
			twice, err := compiler.FormatSource(once, filepath.ToSlash(path))
			if err != nil {
				t.Fatalf("FormatSource twice: %v", err)
			}
			if string(twice) != string(once) {
				t.Fatalf("format not idempotent:\nonce:\n%s\ntwice:\n%s", string(once), string(twice))
			}
		})
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir(%s): %v", examplesDir, err)
	}
}

func TestPlan250FormatRepresentativeCorpusIsIdempotent(t *testing.T) {
	root := repoRoot(t)
	roots := []string{
		filepath.Join(root, "examples"),
		filepath.Join(root, "lib"),
		filepath.Join(root, "__rt"),
		filepath.Join(root, "compiler", "selfhostrt"),
	}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || filepath.Ext(path) != ".tetra" {
				return nil
			}
			t.Run(filepath.ToSlash(path), func(t *testing.T) {
				src, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("ReadFile: %v", err)
				}
				once, err := compiler.FormatSource(src, filepath.ToSlash(path))
				if err != nil {
					t.Fatalf("FormatSource once: %v", err)
				}
				twice, err := compiler.FormatSource(once, filepath.ToSlash(path))
				if err != nil {
					t.Fatalf("FormatSource twice: %v", err)
				}
				if string(twice) != string(once) {
					t.Fatalf("format not idempotent:\nonce:\n%s\ntwice:\n%s", string(once), string(twice))
				}
			})
			return nil
		})
		if err != nil {
			t.Fatalf("WalkDir(%s): %v", root, err)
		}
	}
}

func TestPlan250FormatBlankLineNormalizationAndNestedComments(t *testing.T) {
	src := []byte(`// module
struct Box:
    x: Int


func main() -> Int:
    // before branch
    if true:
        /* inside branch */
        return 42


    // fallback
    return 0
`)
	got, err := compiler.FormatSource(src, "comments.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `// module
struct Box:
    x: Int

func main() -> Int:
    // before branch
    if true:
        /* inside branch */
        return 42
    // fallback
    return 0
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
	again, err := compiler.FormatSource(got, "comments.tetra")
	if err != nil {
		t.Fatalf("FormatSource again: %v", err)
	}
	if string(again) != string(got) {
		t.Fatalf("format not idempotent:\nonce:\n%s\ntwice:\n%s", string(got), string(again))
	}
}

func TestPlan250FormatPreservesCapsuleMetadataAndComments(t *testing.T) {
	src := []byte(`// package metadata
capsule App:
    id: "tetra://fmt/app"
    version: "1.0.0"
    flags.enabled: true

func main() -> Int:
    return 0
`)
	got, err := compiler.FormatSource(src, "capsule_fmt.tetra")
	if err != nil {
		t.Fatalf("FormatSource: %v", err)
	}
	want := `// package metadata
capsule App:
    id: "tetra://fmt/app"
    version: "1.0.0"
    flags.enabled: true

func main() -> Int:
    return 0
`
	if string(got) != want {
		t.Fatalf("formatted source:\n%s\nwant:\n%s", string(got), want)
	}
	again, err := compiler.FormatSource(got, "capsule_fmt.tetra")
	if err != nil {
		t.Fatalf("FormatSource again: %v", err)
	}
	if string(again) != string(got) {
		t.Fatalf("format not idempotent:\nonce:\n%s\ntwice:\n%s", string(got), string(again))
	}
}

func TestPlan250FormatMalformedSyntaxDiagnostics(t *testing.T) {
	_, err := compiler.FormatSource([]byte("func main() -> Int:\n    return @\n"), "malformed_fmt.tetra")
	if err == nil {
		t.Fatalf("expected formatter diagnostic")
	}
	diag := compiler.DiagnosticFromError(err)
	if diag.Code != "TETRA0001" || diag.File != "malformed_fmt.tetra" || diag.Line != 2 || diag.Column != 12 || diag.Severity != "error" {
		t.Fatalf("diagnostic = %#v", diag)
	}
	if diag.Message != "expected expression, got ?" {
		t.Fatalf("message = %q", diag.Message)
	}
}

func TestFlowGrammarSurfaceExampleCoversCanonicalForms(t *testing.T) {
	path := filepath.Join(repoRoot(t), "examples", "flow_grammar_surface_smoke.tetra")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	src := string(raw)
	for _, want := range []string{
		"module examples.flow_grammar_surface_smoke",
		"state CounterState:",
		"view CounterView(state: CounterState):",
		"func id<T>(x: T) -> T:",
		"borrow Int",
		"inout Int",
		"consume Int",
		"let f: ptr = fn(x: Int) -> Int:",
		"let value: Int = try read(flag)",
		"let value: Int = await async_answer()",
		"test \"grammar surface\":",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("%s missing canonical form %q", filepath.ToSlash(path), want)
		}
	}
	if _, err := compiler.FormatSource(raw, filepath.ToSlash(path)); err != nil {
		t.Fatalf("FormatSource(%s): %v", filepath.ToSlash(path), err)
	}
}

func TestPlan250FlowSyntaxSourceOfTruthExamplesCompileAndFormat(t *testing.T) {
	path := filepath.Join(repoRoot(t), "docs", "spec", "flow_syntax_v1.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	section := string(raw)
	start := strings.Index(section, "## Flow Source-Of-Truth Examples")
	if start < 0 {
		t.Fatalf("missing Flow Source-Of-Truth Examples section")
	}
	section = section[start:]
	if end := strings.Index(section, "\n## Blocks And Indentation"); end >= 0 {
		section = section[:end]
	}
	snippets := extractTetraSnippets(section)
	if len(snippets) != 3 {
		t.Fatalf("source-of-truth snippets = %d, want 3", len(snippets))
	}
	for i, snippet := range snippets {
		name := "flow_source_of_truth_" + strconv.Itoa(i+1) + ".tetra"
		t.Run(name, func(t *testing.T) {
			if _, err := compiler.ParseFile([]byte(snippet), name); err != nil {
				t.Fatalf("ParseFile: %v\n%s", err, snippet)
			}
			if _, err := compiler.FormatSource([]byte(snippet), name); err != nil {
				t.Fatalf("FormatSource: %v\n%s", err, snippet)
			}
		})
	}
}

func extractTetraSnippets(markdown string) []string {
	var snippets []string
	var current strings.Builder
	inTetra := false
	for _, line := range strings.Split(markdown, "\n") {
		switch {
		case strings.HasPrefix(line, "```tetra"):
			inTetra = true
			current.Reset()
		case inTetra && strings.HasPrefix(line, "```"):
			snippets = append(snippets, strings.TrimSpace(current.String())+"\n")
			inTetra = false
		case inTetra:
			current.WriteString(line)
			current.WriteByte('\n')
		}
	}
	return snippets
}
