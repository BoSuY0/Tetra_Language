package postv04

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleasePostV04CompilerProductionSmokeScriptRunsExecutableValidator(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "post_v0_4", "compiler-production-linux-x64-smoke.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read compiler production smoke script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"go run ./tools/cmd/compiler-production-smoke --report \"$report_path\"",
		"go run ./tools/cmd/validate-compiler-production --report \"$report_path\"",
		"go run ./tools/cmd/validate-artifact-hashes --write --root \"$report_dir\" --out \"$report_dir/artifact-hashes.json\"",
		"compiler-production-linux-x64.json",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("compiler production smoke script missing %q", want)
		}
	}
}

func TestReleasePostV04MemoryParallelCompilerGateRunsOrderedLayerGates(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "post_v0_4", "memory-parallel-compiler-production-linux-x64-gate.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read memory/parallel/compiler gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`bash "$script_dir/memory-production-linux-x64-smoke.sh" --report-dir "$report_dir"`,
		`bash "$script_dir/parallel-production-linux-x64-smoke.sh" --report-dir "$report_dir"`,
		`bash "$script_dir/compiler-production-linux-x64-smoke.sh" --report-dir "$report_dir"`,
		`go run ./tools/cmd/validate-memory-production --report "$report_dir/memory-production-linux-x64.json"`,
		`go run ./tools/cmd/validate-parallel-production --report "$report_dir/parallel-production-linux-x64.json"`,
		`go run ./tools/cmd/validate-compiler-production --report "$report_dir/compiler-production-linux-x64.json"`,
		`tetra.memory.production.v1`,
		`tetra.parallel.production.v1`,
		`tetra.compiler.production.v1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("memory/parallel/compiler gate missing %q", want)
		}
	}
	if strings.Index(text, "memory-production-linux-x64-smoke.sh") > strings.Index(text, "parallel-production-linux-x64-smoke.sh") {
		t.Fatalf("memory/parallel/compiler gate must run memory before parallelism")
	}
	if strings.Index(text, "parallel-production-linux-x64-smoke.sh") > strings.Index(text, "compiler-production-linux-x64-smoke.sh") {
		t.Fatalf("memory/parallel/compiler gate must run parallelism before compiler")
	}
}

func TestReleasePostV04ScopeDocsAdvertiseCompilerProductionGate(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "docs", "release", "post_v0_4_linux_x64_memory_parallel_ui_scope.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read post-v0.4 scope docs: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`tetra.compiler.production.v1`,
		`bash scripts/release/post_v0_4/compiler-production-linux-x64-smoke.sh --report-dir <dir>`,
		`bash scripts/release/post_v0_4/memory-parallel-compiler-production-linux-x64-gate.sh --report-dir <dir>`,
		`compiler-production-linux-x64.json`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("post-v0.4 scope docs missing compiler production marker %q", want)
		}
	}
}
