package compiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStabilizationOptionalsRejectNoneForNonOptional(t *testing.T) {
	requireCheckErrorContains(t, `
func main() -> Int:
    let x: Int = none
    return x
`, "type mismatch")
}

func TestStabilizationOptionalsRejectIfLetNonOptional(t *testing.T) {
	requireCheckErrorContains(t, `
func main() -> Int:
    let x: Int = 1
    if let y = x:
        return y
    else:
        return 0
`, "if let requires optional value")
}

func TestStabilizationTypedErrorsRejectWrongThrownType(t *testing.T) {
	requireCheckErrorContains(t, `
enum ReadError:
    case eof

enum WriteError:
    case full

func read() -> Int throws ReadError:
    throw WriteError.full

func main() -> Int:
    return 0
`, "throw type mismatch")
}

func TestStabilizationTypedErrorsRejectThrowingMain(t *testing.T) {
	requireCheckErrorContains(t, `
enum E:
    case bad

func main() -> Int throws E:
    throw E.bad
`, "main must not throw")
}

func TestStabilizationOwnershipRejectDuplicateInoutArgument(t *testing.T) {
	requireCheckErrorContains(t, `
func mix(a: inout Int, b: inout Int) -> Int:
    a = a + 1
    b = b + 1
    return a + b

func main() -> Int:
    var x: Int = 1
    return mix(x, x)
`, "inout argument 'x' used more than once")
}

func TestStabilizationEffectsRequireMMIOEffects(t *testing.T) {
	requireCheckErrorContains(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let io: cap.io = core.cap_io()
        let p: ptr = core.alloc_bytes(4)
        return core.mmio_read_i32(p, io)
`, "uses effect 'io'")
}

func TestStabilizationTaskJoinRequiresRuntimeEffect(t *testing.T) {
	requireCheckErrorContains(t, `
func main() -> Int:
    return core.task_join_i32(1)
`, "uses effect 'runtime'")
}

func TestStabilizationProtocolRejectsSignatureMismatch(t *testing.T) {
	requireCheckErrorContains(t, `
struct Vec2:
    x: Int

protocol Renderable:
    func draw(self: Vec2) -> Int

extension Vec2:
    func draw(self: Vec2) -> Bool:
        return true

impl Vec2: Renderable

func main() -> Int:
    return 0
`, "return type differs")
}

func TestStabilizationJSONDiagnosticSnapshotForSemanticError(t *testing.T) {
	err := checkProgram(`
func main() -> Int:
    print("missing uses\n")
    return 0
`)
	if err == nil {
		t.Fatalf("expected semantic diagnostic")
	}
	diag := DiagnosticFromError(err)
	raw, marshalErr := json.Marshal(diag)
	if marshalErr != nil {
		t.Fatalf("Marshal: %v", marshalErr)
	}
	out := string(raw)
	for _, want := range []string{`"code":"TETRA2001"`, `"severity":"error"`, `"message":`} {
		if !strings.Contains(out, want) {
			t.Fatalf("diagnostic JSON missing %q: %s", want, out)
		}
	}
	if !strings.Contains(diag.Message, "uses effect 'io'") {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestStabilizationRejectsDuplicateMatchPatterns(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "int",
			src: `
func main() -> Int:
    match 1:
    case 1:
        return 1
    case 1:
        return 2
    case _:
        return 0
`,
		},
		{
			name: "enum",
			src: `
enum Color:
    case red

func main() -> Int:
    let color: Color = Color.red
    match color:
    case Color.red:
        return 1
    case Color.red:
        return 2
    case _:
        return 0
`,
		},
		{
			name: "optional none",
			src: `
func main() -> Int:
    let value: Int? = none
    match value:
    case none:
        return 1
    case none:
        return 2
    case _:
        return 0
`,
		},
		{
			name: "optional some",
			src: `
func main() -> Int:
    let value: Int? = 1
    match value:
    case some(x):
        return x
    case some(y):
        return y
    case _:
        return 0
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireCheckErrorContains(t, tt.src, "duplicate match pattern")
		})
	}
}

func TestStabilizationForCollectionRejectsNonCollection(t *testing.T) {
	requireCheckErrorContains(t, `
func main() -> Int:
    var total: Int = 0
    for x in 12:
        total = total + x
    return total
`, "for collection requires slice or string")
}

func TestStabilizationRejectsBreakContinueOutsideLoop(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "break",
			src: `
func main() -> Int:
    break
    return 0
`,
			want: "break outside loop",
		},
		{
			name: "continue",
			src: `
func main() -> Int:
    continue
    return 0
`,
			want: "continue outside loop",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireCheckErrorContains(t, tt.src, tt.want)
		})
	}
}

func TestStabilizationUnaryBangRejectsNonCondition(t *testing.T) {
	requireCheckErrorContains(t, `
func main() -> Int:
    let text: String = "x"
    if !text:
        return 1
    return 0
`, "unary '!' expects bool or i32/u8")
}

func TestStabilizationLogicalOperatorsRequireBool(t *testing.T) {
	requireCheckErrorContains(t, `
func main() -> Int:
    if 1 && true:
        return 1
    return 0
`, "logical operators require bool")
}

func TestStabilizationRejectsAssignToConstGlobal(t *testing.T) {
	srcPath := filepath.Join(t.TempDir(), "main.tetra")
	if err := os.WriteFile(srcPath, []byte(`
const answer: i32 = 41

func main() -> Int:
    answer = 42
    return answer
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	_, err := BuildFileWithStatsOpt(srcPath, filepath.Join(t.TempDir(), "app"), "linux-x64", BuildOptions{})
	if err == nil {
		t.Fatalf("expected const assignment error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot assign to const") {
		t.Fatalf("expected const assignment error, got: %v", err)
	}
}

func TestStabilizationRejectsAssignToLocalConst(t *testing.T) {
	requireCheckErrorContains(t, `
func main() -> Int:
    const answer: Int = 41
    answer = 42
    return answer
`, "cannot assign to const")
}

func TestStabilizationRejectsCompoundAssignToLocalConst(t *testing.T) {
	requireCheckErrorContains(t, `
func main() -> Int:
    const answer: Int = 41
    answer += 1
    return answer
`, "cannot assign to const")
}
