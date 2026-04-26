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
