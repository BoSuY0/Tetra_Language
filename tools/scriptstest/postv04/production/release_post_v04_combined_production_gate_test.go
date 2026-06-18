package postv04_production

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleasePostV04CombinedProductionGateRunsOrderedLayerGates(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(
		root,
		"scripts",
		"release",
		"post_v0_4",
		"memory-parallel-ui-production-linux-x64-gate.sh",
	)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read combined post-v0.4 production gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		("Usage: bash scripts/release/post_v0_4/memory-parallel-ui-" +
			"production-linux-x64-gate.sh [--report-dir DIR]"),
		`bash "$script_dir/memory-production-linux-x64-smoke.sh" --report-dir "$report_dir"`,
		`bash "$script_dir/parallel-production-linux-x64-smoke.sh" --report-dir "$report_dir"`,
		`bash "$script_dir/ui-production-runtime-linux-x64-smoke.sh" --report-dir "$report_dir"`,
		`go run ./tools/cmd/validate-memory-production --report "$report_dir/memory-production-linux-x64.json"`,
		`go run ./tools/cmd/validate-parallel-production --report "$report_dir/parallel-production-linux-x64.json"`,
		`go run ./tools/cmd/validate-ui-production-runtime --report "$report_dir/ui-production-runtime-linux-x64.json"`,
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
		`tetra.memory.production.v1`,
		`tetra.parallel.production.v1`,
		`tetra.ui.desktop-runtime.v1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("combined production gate missing %q", want)
		}
	}
	if strings.Index(
		text,
		"memory-production-linux-x64-smoke.sh",
	) > strings.Index(
		text,
		"parallel-production-linux-x64-smoke.sh",
	) {
		t.Fatalf("combined production gate must run memory before parallelism")
	}
	if strings.Index(
		text,
		"parallel-production-linux-x64-smoke.sh",
	) > strings.Index(
		text,
		"ui-production-runtime-linux-x64-smoke.sh",
	) {
		t.Fatalf("combined production gate must run parallelism before UI")
	}
	if strings.Index(
		text,
		"validate-memory-production",
	) > strings.Index(
		text,
		"validate-parallel-production",
	) {
		t.Fatalf("combined production gate must validate memory before parallelism")
	}
	if strings.Index(
		text,
		"validate-parallel-production",
	) > strings.Index(
		text,
		"validate-ui-production-runtime",
	) {
		t.Fatalf("combined production gate must validate parallelism before UI")
	}
}

func TestReleasePostV04ScopeDocsAdvertiseCombinedProductionGate(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(
		root,
		"docs",
		"release",
		"production",
		"post_v0_4_linux_x64_memory_parallel_ui_scope.md",
	)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read post-v0.4 scope docs: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`bash scripts/release/post_v0_4/memory-parallel-ui-production-linux-x64-gate.sh --report-dir <dir>`,
		`memory-production-linux-x64.json`,
		`parallel-production-linux-x64.json`,
		`ui-production-runtime-linux-x64.json`,
		`artifact-hashes.json`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("post-v0.4 scope docs missing %q", want)
		}
	}
}
