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

func TestVerifySpecCodeBlocksChecksTetraAndT4Blocks(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "spec.md")
	body := strings.Join([]string{
		"# Spec",
		"",
		"```tetra check",
		"func main() -> Int:",
		"    return 0",
		"```",
		"",
		"```t4",
		"func helper() -> Int:",
		"    return 1",
		"```",
	}, "\n")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := verifySpecCodeBlocks([]string{doc}); err != nil {
		t.Fatalf("verifySpecCodeBlocks: %v", err)
	}
}

func TestVerifySpecCodeBlocksSkipsExplicitNonExecutableExamples(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "spec.md")
	body := strings.Join([]string{
		"# Spec",
		"",
		"```tetra pseudocode",
		"func broken(",
		"```",
		"",
		"```tetra negative",
		"func broken(",
		"```",
		"",
		"```t4 unsupported",
		"func broken(",
		"```",
	}, "\n")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := verifySpecCodeBlocks([]string{doc}); err != nil {
		t.Fatalf("verifySpecCodeBlocks: %v", err)
	}
}

func TestVerifySpecCodeBlocksRejectsParseDrift(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "spec.md")
	body := strings.Join([]string{
		"# Spec",
		"",
		"```tetra",
		"func broken(",
		"```",
	}, "\n")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifySpecCodeBlocks([]string{doc})
	if err == nil {
		t.Fatalf("expected parse drift failure")
	}
	if !strings.Contains(err.Error(), "spec block 1 parse") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifySpecCodeBlocksRejectsCheckDrift(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "spec.md")
	body := strings.Join([]string{
		"# Spec",
		"",
		"```tetra check",
		"func main() -> Int:",
		"    return missing_symbol",
		"```",
	}, "\n")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifySpecCodeBlocks([]string{doc})
	if err == nil {
		t.Fatalf("expected check drift failure")
	}
	if !strings.Contains(err.Error(), "spec block 1 check") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifySpecCodeBlocksRejectsUnterminatedBlock(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "spec.md")
	if err := os.WriteFile(doc, []byte("```tetra\nfunc main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifySpecCodeBlocks([]string{doc})
	if err == nil {
		t.Fatalf("expected unterminated spec block failure")
	}
	if !strings.Contains(err.Error(), "unterminated tetra spec block starting at line 1") {
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

func TestVerifyReleaseTruthDocsRejectsMisleadingCurrentReleaseLanguage(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "current_supported_surface.md")
	body := strings.Join([]string{
		"# Current Surface",
		"",
		"The current public baseline is v0.1.2.",
		"The current release is v0.6.",
		"Tetra is ready for v1.0.",
	}, "\n")
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyReleaseTruthDocs([]string{doc})
	if err == nil {
		t.Fatalf("expected misleading release language failure")
	}
	for _, want := range []string{"v0.1.2", "current.*v0.6", "ready for v1.0"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected %q in error, got %v", want, err)
		}
	}
}

func TestVerifyReleaseTruthDocsAllowsHistoricalTodoExclusion(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "2026-04-27-tetra-stabilization-5000-todo.md")
	body := "Historical TODO mentions current v0.6 and v0.1.2 for audit context.\n"
	if err := os.WriteFile(doc, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := verifyReleaseTruthDocs([]string{doc}); err != nil {
		t.Fatalf("verifyReleaseTruthDocs: %v", err)
	}
}

func TestCurrentReleaseTruthDocPathsCoverCurrentUserAndSpecDocs(t *testing.T) {
	paths := currentReleaseTruthDocPaths()
	text := strings.Join(paths, "\n")
	for _, want := range []string{
		"README.md",
		"docs/spec/current_supported_surface.md",
		"docs/spec/v0_2_scope.md",
		"docs/user/examples_index.md",
		"docs/user/getting_started.md",
		"docs/user/language_tour.md",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("currentReleaseTruthDocPaths missing %s in %v", want, paths)
		}
	}
	for _, forbidden := range []string{"docs/plans/", "docs/release-notes/"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("currentReleaseTruthDocPaths should not include historical %s paths: %v", forbidden, paths)
		}
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
		"| String helpers | `import lib.core.strings as strings` | `examples/core_strings_smoke.tetra` | `mem` |",
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
		"| String helpers | `import lib.core.strings as strings` | `examples/core_strings_smoke.tetra` | none |",
		"",
		"## Experimental Mirrors",
		"",
		"| Experimental import | Stable replacement | Status |",
		"| --- | --- | --- |",
		"| `import lib.experimental.strings as strings` | `import lib.core.strings as strings` | Experimental mirror; no stability guarantees. |",
	}, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := verifyStdlibGuide(guidePath, []string{corePath}, []string{experimentalPath}); err != nil {
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

func TestVerifyEpic14ExampleIndexAcceptsRequiredCoverage(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "examples_index.md")
	examples := []string{
		"examples/hello.tetra",
		"examples/flow_hello.tetra",
		"examples/bool_smoke.tetra",
		"examples/for_range_smoke.tetra",
		"examples/for_collection_smoke.tetra",
		"examples/loop_control_smoke.tetra",
		"examples/const_smoke.tetra",
		"examples/const_bool_smoke.tetra",
		"examples/local_const_smoke.tetra",
		"examples/compound_assignment_smoke.tetra",
		"examples/enum_match_smoke.tetra",
		"examples/enum_exhaustive_match_smoke.tetra",
		"examples/optional_smoke.tetra",
		"examples/optional_match_smoke.tetra",
		"examples/typed_errors_smoke.tetra",
		"examples/generic_smoke.tetra",
		"examples/generic_struct_smoke.tetra",
		"examples/protocol_impl_smoke.tetra",
		"examples/extension_smoke.tetra",
		"examples/ownership_smoke.tetra",
		"examples/async_smoke.tetra",
		"examples/task_smoke.tetra",
		"examples/actors_pingpong.tetra",
		"examples/islands_hello.tetra",
		"examples/islands_i32.tetra",
		"examples/islands_overflow.tetra",
		"examples/cap_mem_smoke.tetra",
		"examples/mmio_smoke.tetra",
		"examples/memset_smoke.tetra",
		"examples/ui_web_smoke.tetra",
		"examples/ui_native_shell_smoke.tetra",
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

func TestVerifyEpic14ExampleIndexRejectsMissingGenericStructEntry(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "examples_index.md")
	examples := []string{
		"examples/hello.tetra",
		"examples/flow_hello.tetra",
		"examples/bool_smoke.tetra",
		"examples/for_range_smoke.tetra",
		"examples/for_collection_smoke.tetra",
		"examples/loop_control_smoke.tetra",
		"examples/const_smoke.tetra",
		"examples/const_bool_smoke.tetra",
		"examples/local_const_smoke.tetra",
		"examples/compound_assignment_smoke.tetra",
		"examples/enum_match_smoke.tetra",
		"examples/enum_exhaustive_match_smoke.tetra",
		"examples/optional_smoke.tetra",
		"examples/optional_match_smoke.tetra",
		"examples/typed_errors_smoke.tetra",
		"examples/generic_smoke.tetra",
		"examples/protocol_impl_smoke.tetra",
		"examples/extension_smoke.tetra",
		"examples/ownership_smoke.tetra",
		"examples/async_smoke.tetra",
		"examples/task_smoke.tetra",
		"examples/actors_pingpong.tetra",
		"examples/islands_hello.tetra",
		"examples/islands_i32.tetra",
		"examples/islands_overflow.tetra",
		"examples/cap_mem_smoke.tetra",
		"examples/mmio_smoke.tetra",
		"examples/memset_smoke.tetra",
		"examples/ui_web_smoke.tetra",
		"examples/ui_native_shell_smoke.tetra",
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
	err := verifyEpic14ExampleIndex(indexPath)
	if err == nil {
		t.Fatalf("expected missing generic struct coverage failure")
	}
	if !strings.Contains(err.Error(), "examples/generic_struct_smoke.tetra") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyEpic14ExampleIndexRejectsMissingPrimaryT4ProjectEntry(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "examples_index.md")
	examples := []string{
		"examples/hello.tetra",
		"examples/flow_hello.tetra",
		"examples/bool_smoke.tetra",
		"examples/for_range_smoke.tetra",
		"examples/for_collection_smoke.tetra",
		"examples/loop_control_smoke.tetra",
		"examples/const_smoke.tetra",
		"examples/const_bool_smoke.tetra",
		"examples/local_const_smoke.tetra",
		"examples/compound_assignment_smoke.tetra",
		"examples/enum_match_smoke.tetra",
		"examples/enum_exhaustive_match_smoke.tetra",
		"examples/optional_smoke.tetra",
		"examples/optional_match_smoke.tetra",
		"examples/typed_errors_smoke.tetra",
		"examples/generic_smoke.tetra",
		"examples/generic_struct_smoke.tetra",
		"examples/protocol_impl_smoke.tetra",
		"examples/extension_smoke.tetra",
		"examples/ownership_smoke.tetra",
		"examples/async_smoke.tetra",
		"examples/task_smoke.tetra",
		"examples/actors_pingpong.tetra",
		"examples/islands_hello.tetra",
		"examples/islands_i32.tetra",
		"examples/islands_overflow.tetra",
		"examples/cap_mem_smoke.tetra",
		"examples/mmio_smoke.tetra",
		"examples/memset_smoke.tetra",
		"examples/ui_web_smoke.tetra",
		"examples/ui_native_shell_smoke.tetra",
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
	err := verifyEpic14ExampleIndex(indexPath)
	if err == nil {
		t.Fatalf("expected missing primary .t4 project coverage failure")
	}
	if !strings.Contains(err.Error(), "examples/projects/hello_t4/src/main.t4") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyEpic14ExampleIndexRejectsMissingEntry(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "examples_index.md")
	body := strings.Join([]string{
		"# Examples Index",
		"",
		"| Example | Purpose | Target group | Expected behavior |",
		"| --- | --- | --- | --- |",
		"| `examples/flow_hello.tetra` | test entry | native | exits 0 |",
		"## Epic 14 Verification Commands",
		"## Troubleshooting Notes (Epic 14)",
		"### Basic language examples (`V020-0701..0705`)",
		"unsupported regression note",
		"### Control-flow examples (`V020-0706..0710`)",
		"unsupported regression note",
		"### Const and assignment examples (`V020-0711..0715`)",
		"unsupported regression note",
		"### Enum/match examples (`V020-0716..0720`)",
		"unsupported regression note",
		"### Optional/error examples (`V020-0721..0725`)",
		"unsupported regression note",
		"### Generic/protocol/extension examples (`V020-0726..0730`)",
		"unsupported regression note",
		"### Safety/runtime examples (`V020-0731..0735`)",
		"unsupported regression note",
		"### Memory/capability examples (`V020-0736..0740`)",
		"unsupported regression note",
		"### UI/WASM examples (`V020-0741..0745`)",
		"unsupported regression note",
		"### Project dogfood examples (`V020-0746..0750`)",
		"unsupported regression note",
	}, "\n")
	if err := os.WriteFile(indexPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyEpic14ExampleIndex(indexPath)
	if err == nil {
		t.Fatalf("expected Epic 14 missing coverage failure")
	}
	if !strings.Contains(err.Error(), "examples/hello.tetra") {
		t.Fatalf("unexpected error: %v", err)
	}
}
