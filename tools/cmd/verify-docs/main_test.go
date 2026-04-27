package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyDoctestBlocks(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(doc, []byte("```tetra doctest\nfunc main() -> Int:\n    return 0\n```\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyDoctestBlocks([]string{doc}); err != nil {
		t.Fatalf("verifyDoctestBlocks: %v", err)
	}
}

func TestVerifyDoctestBlocksRejectsUnterminatedBlock(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(doc, []byte("text\n```tetra doctest\nfunc main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := verifyDoctestBlocks([]string{doc})
	if err == nil {
		t.Fatalf("expected unterminated doctest failure")
	}
	if !strings.Contains(err.Error(), "unterminated tetra doctest block starting at line 2") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyWASMBackendPlanRequiresConcreteGateCommands(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "wasm_backend_plan.md")
	body := strings.Join([]string{
		"# WASM Backend Plan",
		"",
		"Status: planned",
		"",
		"## Targets",
		"",
		"- `wasm32-wasi`",
		"- `wasm32-web`",
		"",
		"## Phases",
		"",
		"### Phase 0: Target contract",
		"### Phase 1: WASM IR emitter",
		"### Phase 2: WASI runner",
		"### Phase 3: Web runtime",
		"### Phase 4: v1.0 release gate",
		"",
		"## Gate Commands",
		"",
		"- `go run ./tools/cmd/validate-targets`",
		"- `./tetra smoke --target wasm32-wasi --run=false`",
		"- `bash scripts/release_v1_0_gate.sh`",
		"- wasmtime",
		"- browser automation",
	}, "\n")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyWASMBackendPlan(doc, []string{"wasm32-wasi", "wasm32-web"})
	if err == nil {
		t.Fatalf("expected missing wasm32-web gate command failure")
	}
	if !strings.Contains(err.Error(), "./tetra smoke --target wasm32-web --run=false") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExtractTetraDoctestsParsesCommentFence(t *testing.T) {
	doc := strings.Join([]string{
		"// Stable module docs.",
		"// ```tetra doctest",
		"// func demo() -> Int:",
		"//     return 42",
		"// ```",
	}, "\n")
	blocks, err := extractTetraDoctests(doc)
	if err != nil {
		t.Fatalf("extractTetraDoctests: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 doctest block, got %d", len(blocks))
	}
	if !strings.Contains(blocks[0], "func demo() -> Int:") {
		t.Fatalf("unexpected doctest block: %q", blocks[0])
	}
}

func TestVerifyRequiredDoctestBlocksRejectsMissingDoctest(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "module.tetra")
	if err := os.WriteFile(doc, []byte(strings.Join([]string{
		"// Stable v0.5 module docs.",
		"module lib.core.sample",
		"",
		"func add(a: Int, b: Int) -> Int:",
		"    return a + b",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	err := verifyRequiredDoctestBlocks([]string{doc})
	if err == nil {
		t.Fatalf("expected missing doctest failure")
	}
	if !strings.Contains(err.Error(), "missing tetra doctest block") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyRequiredDoctestBlocksAcceptsCommentFenceDoctest(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "module.tetra")
	if err := os.WriteFile(doc, []byte(strings.Join([]string{
		"// Stable v0.5 module docs.",
		"// ```tetra doctest",
		"// func demo() -> Int:",
		"//     return 0",
		"// ```",
		"module lib.core.sample",
		"",
		"func add(a: Int, b: Int) -> Int:",
		"    return a + b",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyRequiredDoctestBlocks([]string{doc}); err != nil {
		t.Fatalf("verifyRequiredDoctestBlocks: %v", err)
	}
}

func TestVerifyStableModuleEffectsMetadataRejectsMissingMetadata(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "module.tetra")
	if err := os.WriteFile(doc, []byte(strings.Join([]string{
		"// Stable docs.",
		"module lib.core.sample",
		"",
		"func id(x: Int) -> Int:",
		"    return x",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	err := verifyStableModuleEffectsMetadata([]string{doc})
	if err == nil {
		t.Fatalf("expected missing effects metadata failure")
	}
	if !strings.Contains(err.Error(), "missing effects metadata") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyStableModuleEffectsMetadataAcceptsDeclaredEffects(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "module.tetra")
	if err := os.WriteFile(doc, []byte(strings.Join([]string{
		"// Stable docs.",
		"// Effects: none",
		"module lib.core.sample",
		"",
		"func id(x: Int) -> Int:",
		"    return x",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyStableModuleEffectsMetadata([]string{doc}); err != nil {
		t.Fatalf("verifyStableModuleEffectsMetadata: %v", err)
	}
}

func TestVerifyStableModuleEffectsMetadataRejectsMismatchedMetadata(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "module.tetra")
	if err := os.WriteFile(doc, []byte(strings.Join([]string{
		"// Stable docs.",
		"// Effects: none",
		"module lib.core.sample",
		"",
		"func len_i32(values: []i32) -> Int",
		"uses mem:",
		"    var count: Int = 0",
		"    for value in values:",
		"        count = count + 1",
		"    return count",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	err := verifyStableModuleEffectsMetadata([]string{doc})
	if err == nil {
		t.Fatalf("expected mismatched effects metadata failure")
	}
	if !strings.Contains(err.Error(), "effects metadata mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyStableModuleExamplesRejectsMissingExampleFile(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "sample.tetra")
	if err := os.WriteFile(doc, []byte(strings.Join([]string{
		"// Stable docs.",
		"// Effects: none",
		"module lib.core.sample",
		"",
		"func id(x: Int) -> Int:",
		"    return x",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	err := verifyStableModuleExamples([]string{doc})
	if err == nil {
		t.Fatalf("expected missing stable module example failure")
	}
	if !strings.Contains(err.Error(), "missing stable module example") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyStdlibModulePathsRejectsMismatchedCoreModule(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "core", "math.tetra")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Join([]string{
		"// Stable docs.",
		"// Effects: none",
		"module lib.experimental.math",
		"",
		"func add(a: Int, b: Int) -> Int:",
		"    return a + b",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyStdlibModulePaths([]string{path}, nil)
	if err == nil {
		t.Fatalf("expected mismatched core module failure")
	}
	if !strings.Contains(err.Error(), "expected module lib.core.math") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyStdlibModulePathsRejectsStableVersionSuffix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "core", "math_v2.tetra")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Join([]string{
		"// Stable docs.",
		"// Effects: none",
		"module lib.core.math_v2",
		"",
		"func add(a: Int, b: Int) -> Int:",
		"    return a + b",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyStdlibModulePaths([]string{path}, nil)
	if err == nil {
		t.Fatalf("expected stable version suffix failure")
	}
	if !strings.Contains(err.Error(), "stable module name must not contain version suffix") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyStableExamplesRejectExperimentalImports(t *testing.T) {
	dir := t.TempDir()
	example := filepath.Join(dir, "examples", "core_math_smoke.tetra")
	if err := os.MkdirAll(filepath.Dir(example), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(example, []byte(strings.Join([]string{
		"import lib.experimental.math as math",
		"",
		"func main() -> Int:",
		"    return math.add(40, 2)",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyStableExamplesDoNotImportExperimental([]string{example})
	if err == nil {
		t.Fatalf("expected experimental import failure")
	}
	if !strings.Contains(err.Error(), "stable example imports experimental module") {
		t.Fatalf("unexpected error: %v", err)
	}
}
