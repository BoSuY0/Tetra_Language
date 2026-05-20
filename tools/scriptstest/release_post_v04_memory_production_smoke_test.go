package scriptstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleasePostV04MemoryProductionSmokeScriptRunsExecutableValidator(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "post_v0_4", "memory-production-linux-x64-smoke.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read memory production smoke script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh [--report-dir DIR]",
		`memory-production-linux-x64.json`,
		`go run ./tools/cmd/memory-production-smoke`,
		`go run ./tools/cmd/validate-memory-production`,
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"`,
		`artifact-hashes.json`,
		`tetra.memory.production.v1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("memory production smoke script missing %q", want)
		}
	}
}

func TestReleasePostV04ScopeDocsAdvertiseMemoryProductionSmokeScript(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "docs", "release", "post_v0_4_linux_x64_memory_parallel_ui_scope.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read post-v0.4 scope docs: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh --report-dir <dir>`,
		`go run ./tools/cmd/memory-production-smoke --report <path>`,
		`go run ./tools/cmd/validate-memory-production --report <path>`,
		`go run ./tools/cmd/validate-artifact-hashes --manifest <dir>/artifact-hashes.json`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("post-v0.4 scope docs missing %q", want)
		}
	}
}
