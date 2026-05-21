package compiler_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
)

func TestGenerateAPIDocs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api.tetra")
	src := `struct Vec2:
    x: Int
    y: Int

enum Color:
    case red

protocol Renderable:
    func draw(self: Vec2) -> Int

impl Vec2: Renderable

state CounterState:
    var count: Int = 0

view CounterView(state: CounterState):
    bind countValue: Int = state.count
    event click -> increment
    command increment:
        state.count = state.count + 1
    style width: Int = 320
    accessibility label: String = "Increment"

const answer: Int = 40 + 2

func add(v: borrow Vec2) -> Int
uses mem, io:
    return v.x + v.y

test "math":
    expect 40 + 2 == 42
`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	docs, err := compiler.GenerateAPIDocs([]string{path})
	if err != nil {
		t.Fatalf("GenerateAPIDocs: %v", err)
	}
	out := string(docs)
	for _, want := range []string{
		"# Tetra API Docs",
		"`Vec2`",
		"`Color`",
		"`const answer: i32`",
		"`state CounterState`",
		"`view CounterView(state: CounterState)`",
		"`style width: i32`",
		"`accessibility label: str`",
		"`protocol Renderable`",
		"`impl Vec2: Renderable`",
		"`func add(v: borrow Vec2) -> i32 uses io, mem`",
		"`math`",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("docs missing %q:\n%s", want, out)
		}
	}
}

func TestGenerateAPIDocsSkipsCapsuleManifestInProjectDirectory(t *testing.T) {
	dir := t.TempDir()
	capsule := `manifest "tetra.capsule.v1"
capsule Demo:
    id "tetra://demo"
    version "0.1.0"
`
	if err := os.WriteFile(filepath.Join(dir, "Capsule.t4"), []byte(capsule), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "main.t4"), []byte("func answer() -> Int:\n    return 42\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	docs, err := compiler.GenerateAPIDocs([]string{dir})
	if err != nil {
		t.Fatalf("GenerateAPIDocs: %v", err)
	}
	if !strings.Contains(string(docs), "`func answer() -> i32`") {
		t.Fatalf("docs missing source entry:\n%s", docs)
	}
}

func TestGenerateAPIDocsDisambiguatesDuplicateModuleHeadings(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.tetra")
	second := filepath.Join(dir, "second.tetra")
	if err := os.WriteFile(first, []byte("module app.main\n\nfunc first() -> Int:\n    return 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("module app.main\n\nfunc second() -> Int:\n    return 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	docs, err := compiler.GenerateAPIDocs([]string{first, second})
	if err != nil {
		t.Fatalf("GenerateAPIDocs: %v", err)
	}
	out := string(docs)
	for _, want := range []string{
		"## app.main (" + filepath.ToSlash(first) + ")",
		"## app.main (" + filepath.ToSlash(second) + ")",
		"`func first() -> i32`",
		"`func second() -> i32`",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("docs missing %q:\n%s", want, out)
		}
	}
}

func TestGenerateAPIDocsLabelsExperimentalModules(t *testing.T) {
	src := []byte(`module lib.experimental.math

func unstable_add(a: Int, b: Int) -> Int:
    return a + b
`)
	docs, err := compiler.GenerateAPIDocsFromSource(src, "lib/experimental/math.tetra")
	if err != nil {
		t.Fatalf("GenerateAPIDocsFromSource: %v", err)
	}
	out := string(docs)
	for _, want := range []string{
		"## lib.experimental.math (experimental)",
		"Experimental module: compatibility is not guaranteed for v1.x.",
		"- `func unstable_add(a: i32, b: i32) -> i32`",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("docs missing %q:\n%s", want, out)
		}
	}
}

func TestGenerateAPIDocsStablePublicSurfaceSnapshot(t *testing.T) {
	src := []byte(`struct Vec2:
    x: Int

protocol Renderable:
    func draw(self: Vec2) -> Int

extension Vec2:
    func draw(self: Vec2) -> Int
    uses io, mem:
        print("draw\n")
        return self.x

impl Vec2: Renderable

func id<T>(x: T) -> T:
    return x

test "draw docs":
    expect Vec2.draw(Vec2(x: 42)) == 42
`)
	docs, err := compiler.GenerateAPIDocsFromSource(src, "surface.tetra")
	if err != nil {
		t.Fatalf("GenerateAPIDocsFromSource: %v", err)
	}
	out := string(docs)
	for _, want := range []string{
		"# Tetra API Docs",
		"<!-- tetra-api-metadata:",
		"## surface.tetra",
		"### Structs",
		"- `Vec2`",
		"  - `x: i32`",
		"### Protocols",
		"- `protocol Renderable`",
		"  - `func draw(self: Vec2) -> i32`",
		"### Implementations",
		"- `impl Vec2: Renderable`",
		"### Functions",
		"- `func id<T>(x: T) -> T`",
		"### Extensions",
		"- `Vec2`",
		"  - `func Vec2.draw(self: Vec2) -> i32 uses io, mem`",
		"### Tests",
		"- `draw docs`",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("docs missing %q:\n%s", want, out)
		}
	}
	functionsStart := strings.Index(out, "### Functions")
	extensionsStart := strings.Index(out, "### Extensions")
	if functionsStart < 0 || extensionsStart < 0 || functionsStart > extensionsStart {
		t.Fatalf("unexpected section order:\n%s", out)
	}
	functionsSection := out[functionsStart:extensionsStart]
	if strings.Contains(functionsSection, "Vec2.draw") {
		t.Fatalf("extension method leaked into top-level functions:\n%s", out)
	}
}

func TestGenerateAPIDocsIncludesTestsAndDoctestFixtures(t *testing.T) {
	src := []byte("// ```tetra doctest\n" +
		"// func example_doctest() -> Int:\n" +
		"//     return 42\n" +
		"// ```\n" +
		"module docs.fixtures\n" +
		"\n" +
		"func answer() -> Int:\n" +
		"    return 42\n" +
		"\n" +
		"test \"answer docs\":\n" +
		"    expect answer() == 42\n")
	docs, err := compiler.GenerateAPIDocsFromSource(src, "fixtures.tetra")
	if err != nil {
		t.Fatalf("GenerateAPIDocsFromSource: %v", err)
	}
	out := string(docs)
	for _, want := range []string{
		"## docs.fixtures",
		"### Functions",
		"- `func answer() -> i32`",
		"### Tests",
		"- `answer docs`",
		"### Doctests",
		"- doctest 1",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("docs missing %q:\n%s", want, out)
		}
	}
}
