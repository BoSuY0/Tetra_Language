package main

import (
	"os"
	"path/filepath"
	"testing"

	"tetra_language/compiler"
)

func TestCheckCommandJSONDiagnosticsForSemanticError(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad.tetra")
	if err := os.WriteFile(srcPath, []byte("func main() -> Int:\n    print(\"x\")\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONDiagnosticForPath(t, srcPath, srcPath, compiler.DiagnosticCodeSafetyEffect, "uses effect 'io'")
}

func TestCheckCommandJSONDiagnosticsForGenericBorrowReturnCodes(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantText string
	}{
		{
			name: "aggregate",
			src: `
struct PtrBox:
    raw: ptr

func leak<T>(value: borrow T) -> T:
    return value

func caller(x: borrow ptr) -> PtrBox:
    return leak(PtrBox(raw: x))

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'value' cannot escape via return",
		},
		{
			name: "optional-ptr",
			src: `
func leak<T>(value: borrow T) -> T:
    return value

func caller(maybe: borrow ptr?) -> ptr?:
    return leak(maybe)

func main() -> Int:
    return 0
`,
			wantText: "borrowed local 'value' cannot escape via return",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			srcPath := filepath.Join(dir, "bad_generic_borrow_return_"+tt.name+".tetra")
			if err := os.WriteFile(srcPath, []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			assertCLIJSONLifetimeDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForCrossModuleGenericBorrowReturnCodes(t *testing.T) {
	tests := []struct {
		name     string
		libSrc   string
		appSrc   string
		wantText string
	}{
		{
			name: "aggregate",
			libSrc: `module lib.leak

pub struct PtrBox:
    raw: ptr

pub func leak<T>(value: borrow T) -> T:
    return value
`,
			appSrc: `func caller(x: borrow ptr) -> leaks.PtrBox:
    return leaks.leak(leaks.PtrBox(raw: x))
`,
			wantText: "borrowed local 'value' cannot escape via return",
		},
		{
			name: "optional-ptr",
			libSrc: `module lib.leak

pub func leak<T>(value: borrow T) -> T:
    return value
`,
			appSrc: `func caller(maybe: borrow ptr?) -> ptr?:
    return leaks.leak(maybe)
`,
			wantText: "borrowed local 'value' cannot escape via return",
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
			writeCLIProjectFile(t, dir, "src/lib/leak.t4", tt.libSrc)
			libPath := filepath.Join(dir, "src", "lib", "leak.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.leak as leaks

`+tt.appSrc+`
func main() -> Int:
    return 0
`)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			assertCLIJSONLifetimeDiagnosticForPath(t, srcPath, libPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForProtocolImplOwnershipMismatchCodes(t *testing.T) {
	t.Run("same module", func(t *testing.T) {
		dir := t.TempDir()
		srcPath := filepath.Join(dir, "bad_protocol_impl_ownership_mismatch.tetra")
		src := `
struct Box:
    value: Int

protocol Sink:
    func sink(self: consume Box) -> Int

extension Box:
    func sink(self: Box) -> Int:
        return self.value

impl Box: Sink

func main() -> Int:
    return 0
`
		if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		assertCLIJSONSemanticDiagnostic(t, srcPath, srcPath, "method 'Box.sink' does not match protocol 'Sink' requirement 'sink': parameter 1 ownership differs: expected 'consume', got 'owned'")
	})

	t.Run("cross module", func(t *testing.T) {
		dir := t.TempDir()
		writeCLIProjectFile(t, dir, "Capsule.t4", `capsule App:
    id "tetra://app"
    version "0.1.0"
    entry "src/app/main.t4"
    sources:
        src
`)
		libPath := filepath.Join(dir, "src", "lib", "model.t4")
		writeCLIProjectFile(t, dir, "src/lib/model.t4", `module lib.model

pub struct Box:
    value: Int

pub protocol Sink:
    func sink(self: consume Box) -> Int

extension Box:
    func sink(self: Box) -> Int:
        return self.value

impl Box: Sink
`)
		srcPath := filepath.Join(dir, "src", "app", "main.t4")
		writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.model as model

func main() -> Int:
    return 0
`)
		assertCLIJSONSemanticDiagnostic(t, srcPath, libPath, "method 'lib.model.Box.sink' does not match protocol 'lib.model.Sink' requirement 'sink': parameter 1 ownership differs: expected 'consume', got 'owned'")
	})
}

func TestCheckCommandJSONDiagnosticsForImportedOwnershipBorrowSliceAggregateCallEscapeCodes(t *testing.T) {
	tests := []struct {
		name     string
		libSrc   string
		appCall  string
		wantText string
	}{
		{
			name: "struct-owned",
			libSrc: `module lib.sink

pub struct BufBox:
    buf: []u8

pub func sink(value: BufBox) -> Int:
    return 0
`,
			appCall:  "return sinker.sink(sinker.BufBox(buf: x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'lib.sink.sink'",
		},
		{
			name: "struct-consume",
			libSrc: `module lib.sink

pub struct BufBox:
    buf: []u8

pub func sink(value: consume BufBox) -> Int:
    return 0
`,
			appCall:  "let box: sinker.BufBox = sinker.BufBox(buf: x)\n    return sinker.sink(box)",
			wantText: "borrowed value derived from 'x' cannot be consumed by 'lib.sink.sink'",
		},
		{
			name: "struct-inout",
			libSrc: `module lib.sink

pub struct BufBox:
    buf: []u8

pub func mutate(value: inout BufBox) -> Int:
    value = value
    return 0
`,
			appCall:  "var box: sinker.BufBox = sinker.BufBox(buf: x)\n    return sinker.mutate(box)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'lib.sink.mutate'",
		},
		{
			name: "enum-owned",
			libSrc: `module lib.sink

pub enum BufMsg:
    case send([]u8)

pub func sink(value: BufMsg) -> Int:
    return 0
`,
			appCall:  "return sinker.sink(sinker.BufMsg.send(x))",
			wantText: "borrowed value derived from 'x' cannot be passed to non-borrow parameter 1 of 'lib.sink.sink'",
		},
		{
			name: "enum-consume",
			libSrc: `module lib.sink

pub enum BufMsg:
    case send([]u8)

pub func sink(value: consume BufMsg) -> Int:
    return 0
`,
			appCall:  "let msg: sinker.BufMsg = sinker.BufMsg.send(x)\n    return sinker.sink(msg)",
			wantText: "borrowed value derived from 'x' cannot be consumed by 'lib.sink.sink'",
		},
		{
			name: "enum-inout",
			libSrc: `module lib.sink

pub enum BufMsg:
    case send([]u8)

pub func mutate(value: inout BufMsg) -> Int:
    value = value
    return 0
`,
			appCall:  "var msg: sinker.BufMsg = sinker.BufMsg.send(x)\n    return sinker.mutate(msg)",
			wantText: "borrowed value derived from 'x' cannot be passed as inout to 'lib.sink.mutate'",
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
			writeCLIProjectFile(t, dir, "src/lib/sink.t4", tt.libSrc)
			srcPath := filepath.Join(dir, "src", "app", "main.t4")
			writeCLIProjectFile(t, dir, "src/app/main.t4", `module app.main
import lib.sink as sinker

func caller(x: borrow []u8) -> Int:
    `+tt.appCall+`

func main() -> Int:
    return 0
`)
			assertCLIJSONOwnershipDiagnostic(t, srcPath, tt.wantText)
		})
	}
}

func TestCheckCommandJSONDiagnosticsForScopedIslandOptionalRegionEscapeCode(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "bad_scoped_island_optional_region.tetra")
	src := `func make() -> []u8?
uses alloc, islands, mem:
    island(16) as isl:
        var xs: []u8 = core.island_make_u8(isl, 4)
        var maybe: []u8? = none
        maybe = xs
        return maybe
    return none
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	assertCLIJSONLifetimeDiagnostic(t, srcPath, "slice from scoped island cannot escape to outer scope")
}
