package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/compiler"
)

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime.tetra")
	src := `func leak(x: borrow ptr) -> ptr:
    return x
func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'x' cannot escape via return")
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowFixedArrayAliasReturnEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_fixed_array_alias_return.tetra")
	src := `func leak(x: borrow [2]Int) -> [2]Int:
    let y: [2]Int = x
    return y
func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'x' cannot escape via return")
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowStringAliasReturnEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_string_alias_return.tetra")
	src := `func leak(x: borrow str) -> str:
    let y: str = x
    return y
func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed String return requires '-> borrow String' or '.copy()'")
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowOptionalAssignmentEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_optional_assignment.tetra")
	src := `func leak(x: borrow ptr) -> ptr?:
    var maybe: ptr? = none
    maybe = x
    return maybe
func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed local 'x' cannot escape via return")
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowSliceOptionalAssignmentEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_lifetime_slice_optional_assignment.tetra")
	src := `func leak(x: borrow []u8) -> []u8?:
    var maybe: []u8? = none
    maybe = x
    return maybe
func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "aggregate '[]u8?' contains borrowed slice field '$elem' that cannot escape through owned return")
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowSliceOptionalAssignmentCallEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantCode string
		wantText string
	}{
		{
			name: "owned",
			src: `func sink(value: []u8?) -> Int:
    return 0
func leak(x: borrow []u8) -> Int:
    var maybe: []u8? = none
    maybe = x
    return sink(maybe)

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1",
		},
		{
			name: "consume",
			src: `func sink(value: consume []u8?) -> Int:
    return 0

func leak(x: borrow []u8) -> Int:
    var maybe: []u8? = none
    maybe = x
    return sink(maybe)

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be consumed by 'sink'",
		},
		{
			name: "inout",
			src: `func leak(x: borrow []u8, out: inout []u8?) -> Int:
    var maybe: []u8? = none
    maybe = x
    out = maybe
    return 0

func main() -> Int:
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "borrowed local 'x' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_lifetime_slice_optional_assignment_"+tt.name+".tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONDiagnosticForPath(t, srcPath, srcPath, tt.wantCode, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowSliceOptionalAssignmentEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		libSrc   string
		wantCode string
		wantText string
	}{
		{
			name: "return",
			libSrc: `module lib.leak

pub func leak(x: borrow []u8) -> []u8?:
    var maybe: []u8? = none
    maybe = x
    return maybe
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "aggregate '[]u8?' contains borrowed slice field '$elem' that cannot escape through owned return",
		},
		{
			name: "owned",
			libSrc: `module lib.leak

pub func sink(value: []u8?) -> Int:
    return 0

pub func leak(x: borrow []u8) -> Int:
    var maybe: []u8? = none
    maybe = x
    return sink(maybe)
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'lib.leak.sink'",
		},
		{
			name: "consume",
			libSrc: `module lib.leak

pub func sink(value: consume []u8?) -> Int:
    return 0

pub func leak(x: borrow []u8) -> Int:
    var maybe: []u8? = none
    maybe = x
    return sink(maybe)
`,
			wantCode: compiler.DiagnosticCodeSafetyOwnership,
			wantText: "borrowed value derived from 'x' cannot be consumed by 'lib.leak.sink'",
		},
		{
			name: "inout",
			libSrc: `module lib.leak

pub func leak(x: borrow []u8, out: inout []u8?) -> Int:
    var maybe: []u8? = none
    maybe = x
    out = maybe
    return 0
`,
			wantCode: compiler.DiagnosticCodeSafetyLifetime,
			wantText: "borrowed local 'x' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			libPath := filepath.Join(dir, "src", "lib", "leak.t4")
			writeCLIProjectFile(t, dir, "src/lib/leak.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leak as leaks
func main() -> Int:
    return 0
`)
			assertCLIJSONDiagnosticForPath(t, srcPath, libPath, tt.wantCode, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowSliceStructEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "literal-return",
			src: `struct BufBox:
    buf: []u8
func leak(x: borrow []u8) -> BufBox:
    return BufBox(buf: x)

func main() -> Int:
    return 0
`,
			wantText: "aggregate 'BufBox' contains borrowed slice field 'buf' that cannot escape through owned return",
		},
		{
			name: "alias-return",
			src: `struct BufBox:
    buf: []u8

func leak(x: borrow []u8) -> BufBox:
    let box: BufBox = BufBox(buf: x)
    return box

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "inout-assignment",
			src: `struct BufBox:
    buf: []u8

func leak(read: borrow []u8, out: inout BufBox) -> Int:
    out = BufBox(buf: read)
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'read' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_lifetime_slice_struct_"+tt.name+".tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONLifetimeDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowSliceStructEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		libSrc   string
		wantText string
	}{
		{
			name: "literal-return",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub func leak(x: borrow []u8) -> BufBox:
    return BufBox(buf: x)
`,
			wantText: "aggregate 'BufBox' contains borrowed slice field 'buf' that cannot escape through owned return",
		},
		{
			name: "alias-return",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub func leak(x: borrow []u8) -> BufBox:
    let box: BufBox = BufBox(buf: x)
    return box
`,
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "inout-assignment",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub func leak(read: borrow []u8, out: inout BufBox) -> Int:
    out = BufBox(buf: read)
    return 0
`,
			wantText: "borrowed local 'read' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			libPath := filepath.Join(dir, "src", "lib", "leaks.t4")
			writeCLIProjectFile(t, dir, "src/lib/leaks.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks
func main() -> Int:
    return 0
`)
			assertCLIJSONLifetimeDiagnosticForPath(t, srcPath, libPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowNestedSliceStructEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "literal-return",
			src: `struct BufBox:
    buf: []u8

struct OuterBox:
    box: BufBox
func leak(x: borrow []u8) -> OuterBox:
    return OuterBox(box: BufBox(buf: x))

func main() -> Int:
    return 0
`,
			wantText: "aggregate 'BufBox' contains borrowed slice field 'buf' that cannot escape through owned return",
		},
		{
			name: "alias-return",
			src: `struct BufBox:
    buf: []u8

struct OuterBox:
    box: BufBox

func leak(x: borrow []u8) -> OuterBox:
    let outer: OuterBox = OuterBox(box: BufBox(buf: x))
    return outer

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "inout-assignment",
			src: `struct BufBox:
    buf: []u8

struct OuterBox:
    box: BufBox

func leak(read: borrow []u8, out: inout OuterBox) -> Int:
    out = OuterBox(box: BufBox(buf: read))
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'read' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_lifetime_nested_slice_struct_"+tt.name+".tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONLifetimeDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowNestedSliceStructEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		libSrc   string
		wantText string
	}{
		{
			name: "literal-return",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub struct OuterBox:
    box: BufBox

pub func leak(x: borrow []u8) -> OuterBox:
    return OuterBox(box: BufBox(buf: x))
`,
			wantText: "aggregate 'BufBox' contains borrowed slice field 'buf' that cannot escape through owned return",
		},
		{
			name: "alias-return",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub struct OuterBox:
    box: BufBox

pub func leak(x: borrow []u8) -> OuterBox:
    let outer: OuterBox = OuterBox(box: BufBox(buf: x))
    return outer
`,
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "inout-assignment",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub struct OuterBox:
    box: BufBox

pub func leak(read: borrow []u8, out: inout OuterBox) -> Int:
    out = OuterBox(box: BufBox(buf: read))
    return 0
`,
			wantText: "borrowed local 'read' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			libPath := filepath.Join(dir, "src", "lib", "leaks.t4")
			writeCLIProjectFile(t, dir, "src/lib/leaks.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks
func main() -> Int:
    return 0
`)
			assertCLIJSONLifetimeDiagnosticForPath(t, srcPath, libPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowNestedSliceEnumPayloadEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "literal-return",
			src: `struct BufBox:
    buf: []u8

enum OuterMsg:
    case wrap(BufBox)
    case empty
func leak(x: borrow []u8) -> OuterMsg:
    return OuterMsg.wrap(BufBox(buf: x))

func main() -> Int:
    return 0
`,
			wantText: "aggregate 'BufBox' contains borrowed slice field 'buf' that cannot escape through owned return",
		},
		{
			name: "alias-return",
			src: `struct BufBox:
    buf: []u8

enum OuterMsg:
    case wrap(BufBox)
    case empty

func leak(x: borrow []u8) -> OuterMsg:
    let msg: OuterMsg = OuterMsg.wrap(BufBox(buf: x))
    return msg

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "inout-assignment",
			src: `struct BufBox:
    buf: []u8

enum OuterMsg:
    case wrap(BufBox)
    case empty

func leak(read: borrow []u8, out: inout OuterMsg) -> Int:
    out = OuterMsg.wrap(BufBox(buf: read))
    return 0

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'read' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_lifetime_nested_slice_enum_payload_"+tt.name+".tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONLifetimeDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowNestedSliceEnumPayloadEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		libSrc   string
		wantText string
	}{
		{
			name: "literal-return",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub enum OuterMsg:
    case wrap(BufBox)
    case empty

pub func leak(x: borrow []u8) -> OuterMsg:
    return OuterMsg.wrap(BufBox(buf: x))
`,
			wantText: "aggregate 'BufBox' contains borrowed slice field 'buf' that cannot escape through owned return",
		},
		{
			name: "alias-return",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub enum OuterMsg:
    case wrap(BufBox)
    case empty

pub func leak(x: borrow []u8) -> OuterMsg:
    let msg: OuterMsg = OuterMsg.wrap(BufBox(buf: x))
    return msg
`,
			wantText: "borrowed local 'x' cannot escape via return",
		},
		{
			name: "inout-assignment",
			libSrc: `module lib.leaks

pub struct BufBox:
    buf: []u8

pub enum OuterMsg:
    case wrap(BufBox)
    case empty

pub func leak(read: borrow []u8, out: inout OuterMsg) -> Int:
    out = OuterMsg.wrap(BufBox(buf: read))
    return 0
`,
			wantText: "borrowed local 'read' cannot escape via inout assignment to 'out'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			libPath := filepath.Join(dir, "src", "lib", "leaks.t4")
			writeCLIProjectFile(t, dir, "src/lib/leaks.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks
func main() -> Int:
    return 0
`)
			assertCLIJSONLifetimeDiagnosticForPath(t, srcPath, libPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForLifetimeBorrowSliceEnumEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "direct-return",
			src: `enum BufMsg:
    case send([]u8)
func leak(x: borrow []u8) -> BufMsg:
    return BufMsg.send(x)

func main() -> Int:
    return 0
`,
			wantText: "aggregate 'BufMsg' contains borrowed slice field 'BufMsg.send[1]' that cannot escape through owned return",
		},
		{
			name: "alias-return",
			src: `enum BufMsg:
    case send([]u8)

func leak(x: borrow []u8) -> BufMsg:
    let msg: BufMsg = BufMsg.send(x)
    return msg

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'x' cannot escape via return",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_lifetime_slice_enum_"+tt.name+".tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONLifetimeDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleLifetimeBorrowSliceEnumEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		libSrc   string
		wantText string
	}{
		{
			name: "direct-return",
			libSrc: `module lib.leaks

pub enum BufMsg:
    case send([]u8)

pub func leak(x: borrow []u8) -> BufMsg:
    return BufMsg.send(x)
`,
			wantText: "aggregate 'BufMsg' contains borrowed slice field 'BufMsg.send[1]' that cannot escape through owned return",
		},
		{
			name: "alias-return",
			libSrc: `module lib.leaks

pub enum BufMsg:
    case send([]u8)

pub func leak(x: borrow []u8) -> BufMsg:
    let msg: BufMsg = BufMsg.send(x)
    return msg
`,
			wantText: "borrowed local 'x' cannot escape via return",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
			libPath := filepath.Join(dir, "src", "lib", "leaks.t4")
			writeCLIProjectFile(t, dir, "src/lib/leaks.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leaks as leaks
func main() -> Int:
    return 0
`)
			assertCLIJSONLifetimeDiagnosticForPath(t, srcPath, libPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForSafeViewBorrowedOwnedReturnCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_safe_view_owned_return.tetra")
	src := `func bad(xs: borrow []u8) -> []u8:
    return xs.borrow()

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "borrowed slice return requires '-> borrow []u8' or '.copy()'")
}

func TestCheckCommandJSONDiagnosticsForSafeViewActorBoundaryCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_safe_view_actor_boundary.tetra")
	src := `enum Msg:
    case bytes([]u8)

func main() -> Int
uses actors, alloc, mem:
    var xs: []u8 = make_u8(1)
    return core.send_typed(core.self(), Msg.bytes(xs.borrow()))
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONDiagnosticForPath(t, srcPath, srcPath, compiler.DiagnosticCodeSafetyOwnership, "cannot cross actor boundary without copy")
}

func TestCheckCommandJSONDiagnosticsForSafeViewTaskBoundaryCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_safe_view_task_boundary.tetra")
	src := `enum TaskErr:
    case bytes([]u8)

func worker() -> Int throws TaskErr:
    return 0

func main() -> Int
uses runtime:
    let task = core.task_spawn_i32_typed<TaskErr>("worker")
    return try core.task_join_i32_typed<TaskErr>(task)
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	diag := runCLIJSONDiagnostic(t, []string{"check", "--diagnostics=json", srcPath}, 1)
	if diag.Severity != "error" || !strings.Contains(diag.Message, "typed task error payload must be sendable across task boundary") {
		t.Fatalf("diagnostic = %#v", diag)
	}
}

func TestCheckCommandJSONDiagnosticsForSafeViewAggregateHiddenBorrowCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_safe_view_aggregate_return.tetra")
	src := `struct Box:
    bytes: []u8

func bad(xs: borrow []u8) -> Box:
    return Box(bytes: xs.window(0, 1).borrow())

func main() -> Int:
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "aggregate 'Box' contains borrowed slice field 'bytes' that cannot escape through owned return")
}
