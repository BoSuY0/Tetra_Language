package compiler

import (
	"strings"
	"testing"
)

func TestFormatSourceFlowMVP(t *testing.T) {
	src := []byte(`func main() -> Int
uses mem, io:
    print("hi\n")
    return 0
`)
	got, err := FormatSource(src, "main.tetra")
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
	got, err := FormatSource(src, "ui.tetra")
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
	got, err := FormatSource(src, "main.tetra")
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
	got, err := FormatSource(src, "main.tetra")
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

func TestFormatSourcePreservesExportAttributes(t *testing.T) {
	src := []byte(`@export("__tetra_entry")
fun tetra_entry(): i32 {
    return 0
}
`)
	got, err := FormatSource(src, "runtime.tetra")
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

func TestFormatSourceCommentPreservationIsIdempotent(t *testing.T) {
	src := []byte(`// suite
test "math":
    // expected arithmetic
    expect 40 + 2 == 42
`)
	once, err := FormatSource(src, "math_test.tetra")
	if err != nil {
		t.Fatalf("FormatSource once: %v", err)
	}
	twice, err := FormatSource(once, "math_test.tetra")
	if err != nil {
		t.Fatalf("FormatSource twice: %v", err)
	}
	if string(twice) != string(once) {
		t.Fatalf("format not idempotent:\nonce:\n%s\ntwice:\n%s", string(once), string(twice))
	}
}

func TestFormatSourceRejectsInlineCommentsConservatively(t *testing.T) {
	_, err := FormatSource([]byte("func main() -> Int:\n    return 0 // trailing\n"), "main.tetra")
	if err == nil {
		t.Fatalf("expected comment-preservation diagnostic")
	}
	if !strings.Contains(err.Error(), "inline comments are not supported") {
		t.Fatalf("error = %v", err)
	}
}

func TestFormatSourceInlineCommentDiagnosticHasLocation(t *testing.T) {
	_, err := FormatSource([]byte("func main() -> Int:\n    return 0 // trailing\n"), "main.tetra")
	if err == nil {
		t.Fatalf("expected comment-preservation diagnostic")
	}
	diag := DiagnosticFromError(err)
	if diag.Code != "TETRA_FMT001" || diag.File != "main.tetra" || diag.Line != 2 || diag.Column != 14 {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestFormatSourcePreservesCommentAfterSingleLineUsesHeader(t *testing.T) {
	src := []byte(`func main() -> Int uses io:
    // before return
    return 0
`)
	got, err := FormatSource(src, "main.tetra")
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
	got, err := FormatSource(src, "main.tetra")
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
	got, err := FormatSource(src, "main.tetra")
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
	got, err := FormatSource(src, "main.tetra")
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
	got, err := FormatSource(src, "main.tetra")
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
	got, err := FormatSource(src, "main.tetra")
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
	got, err := FormatSource(src, "main.tetra")
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
	got, err := FormatSource(src, "main.tetra")
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
	got, err := FormatSource(src, "main.tetra")
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

func TestFormatSourcePreservesSemanticClauses(t *testing.T) {
	src := []byte(`func main() -> Int noalloc noblock realtime nothrow budget(10):
    return 0
`)
	got, err := FormatSource(src, "main.tetra")
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

func TestFormatSourceClosureLiteralIsIdempotent(t *testing.T) {
	src := []byte(`func main() -> Int:
    let f: ptr = fn(x: Int) -> Int:
        return x
    return 0
`)
	once, err := FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource once: %v", err)
	}
	if strings.Contains(string(once), "<expr>") {
		t.Fatalf("formatted source lost closure expression:\n%s", string(once))
	}
	twice, err := FormatSource(once, "main.tetra")
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
	once, err := FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource once: %v", err)
	}
	if strings.Contains(string(once), "<expr>") {
		t.Fatalf("formatted source lost nested closure expression:\n%s", string(once))
	}
	if !strings.Contains(string(once), "func __closure_") {
		t.Fatalf("formatted source missing synthetic closure declaration:\n%s", string(once))
	}
	twice, err := FormatSource(once, "main.tetra")
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
	got, err := FormatSource(src, "main.tetra")
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
	got, err := FormatSource(src, "main.tetra")
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
	got, err := FormatSource(src, "main.tetra")
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
	once, err := FormatSource(src, "main.tetra")
	if err != nil {
		t.Fatalf("FormatSource once: %v", err)
	}
	twice, err := FormatSource(once, "main.tetra")
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

	once, err := FormatSource(src, "full_surface.tetra")
	if err != nil {
		t.Fatalf("FormatSource once: %v", err)
	}
	twice, err := FormatSource(once, "full_surface.tetra")
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
