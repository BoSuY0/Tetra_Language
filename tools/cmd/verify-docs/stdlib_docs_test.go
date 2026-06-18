package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestVerifyStableModuleDoctestCoverageRejectsPlaceholder(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "memory.tetra")
	if err := os.WriteFile(doc, []byte(strings.Join([]string{
		"// Stable docs.",
		"// Effects: mem",
		"// ```tetra doctest",
		"// func memory_doctest() -> Int:",
		"//     return 0",
		"// ```",
		"module lib.core.memory",
		"",
		"func memset_u8(dst: ptr, v: UInt8, n: Int, mem: cap.mem) -> Int",
		"uses mem:",
		"    return 0",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyStableModuleDoctestCoverage([]string{doc})
	if err == nil {
		t.Fatalf("expected placeholder doctest failure")
	}
	if !strings.Contains(err.Error(), "doctest does not reference lib.core.memory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyStableModuleDoctestCoverageAcceptsModuleReference(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "memory.tetra")
	if err := os.WriteFile(doc, []byte(strings.Join([]string{
		"// Stable docs.",
		"// Effects: mem",
		"// ```tetra doctest",
		"// func memory_doctest() -> Int:",
		"//     return lib.core.memory.memcpy_status()",
		"// ```",
		"module lib.core.memory",
		"",
		"func memcpy_status() -> Int:",
		"    return 0",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := verifyStableModuleDoctestCoverage([]string{doc}); err != nil {
		t.Fatalf("verifyStableModuleDoctestCoverage: %v", err)
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

func TestVerifyStableModuleEffectsMetadataAcceptsSplitPartDeclaredEffects(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "block.tetra")
	if err := os.WriteFile(doc, []byte(strings.Join([]string{
		"// Stable docs.",
		"// Effects: alloc, mem",
		"module lib.core.block",
		"",
		"func id(x: Int) -> Int:",
		"    return x",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	partsDir := filepath.Join(dir, "block.parts")
	if err := os.MkdirAll(partsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	partPath := filepath.Join(partsDir, "tree.tetra")
	if err := os.WriteFile(partPath, []byte(strings.Join([]string{
		"module lib.core.block",
		"",
		"func tree_init() -> Int",
		"uses alloc, mem:",
		"    return 0",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := verifyStableModuleEffectsMetadata([]string{doc}); err != nil {
		t.Fatalf("verifyStableModuleEffectsMetadata: %v", err)
	}
}

func TestVerifyStableModuleEffectsMetadataRejectsSplitPartMissingDeclaredEffects(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "block.tetra")
	if err := os.WriteFile(doc, []byte(strings.Join([]string{
		"// Stable docs.",
		"// Effects: mem",
		"module lib.core.block",
		"",
		"func id(x: Int) -> Int:",
		"    return x",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	partsDir := filepath.Join(dir, "block.parts")
	if err := os.MkdirAll(partsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	partPath := filepath.Join(partsDir, "tree.tetra")
	if err := os.WriteFile(partPath, []byte(strings.Join([]string{
		"module lib.core.block",
		"",
		"func tree_len() -> Int:",
		"    return 0",
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

func TestVerifyStableModuleEffectsMetadataRejectsSplitPartInvalidEffectWithPath(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "block.tetra")
	if err := os.WriteFile(doc, []byte(strings.Join([]string{
		"// Stable docs.",
		"// Effects: none",
		"module lib.core.block",
		"",
		"func id(x: Int) -> Int:",
		"    return x",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	partsDir := filepath.Join(dir, "block.parts")
	if err := os.MkdirAll(partsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	partPath := filepath.Join(partsDir, "tree.tetra")
	if err := os.WriteFile(partPath, []byte(strings.Join([]string{
		"module lib.core.block",
		"",
		"func tree_init() -> Int",
		"uses spooky:",
		"    return 0",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyStableModuleEffectsMetadata([]string{doc})
	if err == nil {
		t.Fatalf("expected invalid split part effect failure")
	}
	if !strings.Contains(err.Error(), partPath) {
		t.Fatalf("expected error to name split part path %q, got %v", partPath, err)
	}
	if !strings.Contains(err.Error(), "unknown stable effect") {
		t.Fatalf("unexpected error: %v", err)
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

func TestVerifyStdlibGuideRejectsMismatchedStableEffects(t *testing.T) {
	dir := t.TempDir()
	coreDir := filepath.Join(dir, "lib", "core")
	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatal(err)
	}
	modulePath := filepath.Join(coreDir, "strings.tetra")
	if err := os.WriteFile(modulePath, []byte(strings.Join([]string{
		"// Stable docs.",
		"// Effects: none",
		"// ```tetra doctest",
		"// func strings_doctest() -> Int:",
		"//     return lib.core.strings.ascii_len(\"x\")",
		"// ```",
		"module lib.core.strings",
		"",
		"func ascii_len(text: String) -> Int:",
		"    return 0",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	guidePath := filepath.Join(dir, "standard_library_guide.md")
	if err := os.WriteFile(guidePath, []byte(strings.Join([]string{
		"# Standard Library Guide",
		"",
		"| Need | Import | Example | Effects |",
		"| --- | --- | --- | --- |",
		("| String helpers | `import lib.core.strings as strings` | " +
			"`examples/core/data/core_strings_smoke.tetra` | `mem` |"),
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyStdlibGuide(guidePath, []string{modulePath}, nil)
	if err == nil {
		t.Fatalf("expected guide effects mismatch")
	}
	if !strings.Contains(err.Error(), "lib.core.strings effects mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyStdlibGuideAcceptsStableAndExperimentalMirrors(t *testing.T) {
	dir := t.TempDir()
	coreDir := filepath.Join(dir, "lib", "core")
	experimentalDir := filepath.Join(dir, "lib", "experimental")
	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(experimentalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	corePath := filepath.Join(coreDir, "strings.tetra")
	if err := os.WriteFile(corePath, []byte(strings.Join([]string{
		"// Stable docs.",
		"// Effects: none",
		"// ```tetra doctest",
		"// func strings_doctest() -> Int:",
		"//     return lib.core.strings.ascii_len(\"x\")",
		"// ```",
		"module lib.core.strings",
		"",
		"func ascii_len(text: String) -> Int:",
		"    return 0",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	experimentalPath := filepath.Join(experimentalDir, "strings.tetra")
	if err := os.WriteFile(experimentalPath, []byte(strings.Join([]string{
		"// Experimental strings helpers (no stability guarantees).",
		"//",
		"// Promotion note: v1 stable callers should use lib.core.strings directly.",
		"module lib.experimental.strings",
		"",
		"import lib.core.strings as stable_strings",
		"",
		"func ascii_len(text: String) -> Int:",
		"    return stable_strings.ascii_len(text)",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	guidePath := filepath.Join(dir, "standard_library_guide.md")
	if err := os.WriteFile(guidePath, []byte(strings.Join([]string{
		"# Standard Library Guide",
		"",
		"| Need | Import | Example | Effects |",
		"| --- | --- | --- | --- |",
		("| String helpers | `import lib.core.strings as strings` | " +
			"`examples/core/data/core_strings_smoke.tetra` | none |"),
		"",
		"## Experimental Mirrors",
		"",
		"| Experimental import | Stable replacement | Status |",
		"| --- | --- | --- |",
		("| `import lib.experimental.strings as strings` | `import " +
			"lib.core.strings as strings` | Experimental mirror; no stability " +
			"guarantees. |"),
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := verifyStdlibGuide(
		guidePath,
		[]string{corePath},
		[]string{experimentalPath},
	); err != nil {
		t.Fatalf("verifyStdlibGuide: %v", err)
	}
}

func TestVerifyExperimentalModuleMirrorsRejectsMissingPromotionNote(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lib", "experimental", "math.tetra")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Join([]string{
		"// Experimental math helpers (no stability guarantees).",
		"module lib.experimental.math",
		"",
		"import lib.core.math as stable_math",
		"",
		"func add_i32(a: Int, b: Int) -> Int:",
		"    return stable_math.add_i32(a, b)",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyExperimentalModuleMirrors([]string{path})
	if err == nil {
		t.Fatalf("expected missing promotion note failure")
	}
	if !strings.Contains(err.Error(), "missing promotion note") {
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
	path := filepath.Join(dir, "lib", "core", "math.tetra")
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
	path := filepath.Join(dir, "lib", "core", "math_v2.tetra")
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
	example := filepath.Join(dir, "examples", "core", "data", "core_math_smoke.tetra")
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

func TestVerifyEpic14ExampleIndexAcceptsRequiredCoverage(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "examples_index.md")
	examples := []string{
		"examples/smoke/basic/hello.tetra",
		"examples/flow/flow_hello.tetra",
		"examples/smoke/scalars/bool_smoke.tetra",
		"examples/smoke/control/for_range_smoke.tetra",
		"examples/smoke/control/for_collection_smoke.tetra",
		"examples/smoke/control/loop_control_smoke.tetra",
		"examples/smoke/scalars/const_smoke.tetra",
		"examples/smoke/scalars/const_bool_smoke.tetra",
		"examples/smoke/scalars/local_const_smoke.tetra",
		"examples/smoke/scalars/compound_assignment_smoke.tetra",
		"examples/smoke/types/enum_match_smoke.tetra",
		"examples/smoke/types/enum_exhaustive_match_smoke.tetra",
		"examples/smoke/types/optional_smoke.tetra",
		"examples/smoke/types/optional_match_smoke.tetra",
		"examples/smoke/errors/typed_errors_smoke.tetra",
		"examples/smoke/language/generic_smoke.tetra",
		"examples/smoke/language/generic_struct_smoke.tetra",
		"examples/smoke/language/protocol_impl_smoke.tetra",
		"examples/smoke/language/extension_smoke.tetra",
		"examples/memory/ownership/ownership_smoke.tetra",
		"examples/async/async_smoke.tetra",
		"examples/tasks/task_smoke.tetra",
		"examples/actors/actors_pingpong.tetra",
		"examples/memory/islands/islands_hello.tetra",
		"examples/memory/islands/islands_i32.tetra",
		"examples/memory/islands/islands_overflow.tetra",
		"examples/memory/raw/cap_mem_smoke.tetra",
		"examples/memory/raw/mmio_smoke.tetra",
		"examples/memory/raw/memset_smoke.tetra",
		"examples/ui/ui_web_smoke.tetra",
		"examples/ui/ui_native_shell_smoke.tetra",
		"examples/projects/hello_t4/src/main.t4",
		"examples/projects/dogfood_wasi/src/main.tetra",
		"examples/projects/dogfood_web_ui/src/main.tetra",
		"examples/projects/dogfood_cli/src/main.tetra",
		"examples/projects/dogfood_actor_task/src/main.tetra",
		"examples/projects/eco_dogfood/src/main.tetra",
	}
	headings := []string{
		"## Epic 14 Verification Commands",
		"## Troubleshooting Notes (Epic 14)",
		"### Basic language examples (`V020-0701..0705`)",
		"### Control-flow examples (`V020-0706..0710`)",
		"### Const and assignment examples (`V020-0711..0715`)",
		"### Enum/match examples (`V020-0716..0720`)",
		"### Optional/error examples (`V020-0721..0725`)",
		"### Generic/protocol/extension examples (`V020-0726..0730`)",
		"### Safety/runtime examples (`V020-0731..0735`)",
		"### Memory/capability examples (`V020-0736..0740`)",
		"### UI/WASM examples (`V020-0741..0745`)",
		"### Project dogfood examples (`V020-0746..0750`)",
	}

	lines := []string{
		"# Examples Index",
		"",
		"| Example | Purpose | Target group | Expected behavior |",
		"| --- | --- | --- | --- |",
	}
	for _, example := range examples {
		lines = append(lines, "| `"+example+"` | test entry | native | exits 0 |")
	}
	for _, heading := range headings {
		lines = append(lines, "", heading, "", "unsupported profile note", "regression note")
	}

	if err := os.WriteFile(indexPath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyEpic14ExampleIndex(indexPath); err != nil {
		t.Fatalf("verifyEpic14ExampleIndex: %v", err)
	}
}
