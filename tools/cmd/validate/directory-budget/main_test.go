package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanCountsActiveFilesAndIgnoresGenerated(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 7; i++ {
		writeFile(
			t,
			dir,
			filepath.Join("compiler", "internal", "semantics", fileName(i, ".go")),
			"package semantics\n",
		)
		writeFile(t, dir, filepath.Join("docs", "audits", fileName(i, ".md")), "# audit\n")
		writeFile(t, dir, filepath.Join("docs", "generated", fileName(i, ".md")), "# generated\n")
	}
	writeFile(t, dir, filepath.Join("compiler", "internal", "semantics", "README.md"), "# nav\n")
	writeFile(t, dir, filepath.Join("examples", ".tetra_cache", "cached.tetra"), "cached\n")

	counts, err := scanDirectories(dir, testConfig([]string{"compiler", "docs", "examples"}))
	if err != nil {
		t.Fatalf("scanDirectories: %v", err)
	}

	byDir := mapCounts(counts)
	if got := byDir["compiler/internal/semantics"].Count; got != 7 {
		t.Fatalf("compiler/internal/semantics count = %d, want 7", got)
	}
	if got := byDir["docs/audits"].Count; got != 7 {
		t.Fatalf("docs/audits count = %d, want 7", got)
	}
	if _, ok := byDir["docs/generated"]; ok {
		t.Fatalf("docs/generated should be excluded")
	}
	if _, ok := byDir["examples/.tetra_cache"]; ok {
		t.Fatalf(".tetra_cache should be excluded")
	}
}

func TestScanIgnoresOptInCustomBuildTags(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 6; i++ {
		writeFile(
			t,
			dir,
			filepath.Join("tools", "cmd", "surface-runtime-smoke", fileName(i, ".go")),
			"package main\n",
		)
	}
	writeFile(
		t,
		dir,
		filepath.Join("tools", "cmd", "surface-runtime-smoke", "linux.go"),
		"//go:build linux\n\npackage main\n",
	)
	writeFile(
		t,
		dir,
		filepath.Join("tools", "cmd", "surface-runtime-smoke", "guest.go"),
		"//go:build linux && guestviewer\n\npackage main\n",
	)

	counts, err := scanDirectories(dir, testConfig([]string{"tools"}))
	if err != nil {
		t.Fatalf("scanDirectories: %v", err)
	}

	byDir := mapCounts(counts)
	if got := byDir["tools/cmd/surface-runtime-smoke"].Count; got != 7 {
		t.Fatalf("surface-runtime-smoke count = %d, want 7 active default/platform files", got)
	}
	if strings.Contains(
		strings.Join(byDir["tools/cmd/surface-runtime-smoke"].Files, "\n"),
		"guest.go",
	) {
		t.Fatalf("custom guestviewer build-tagged file should not be counted")
	}
}

func TestBaselineAllowsExistingViolationButRejectsGrowth(t *testing.T) {
	counts := []directoryCount{
		{Path: "compiler", Count: 7},
		{Path: "examples", Count: 8},
	}
	baseline := &baselineFile{
		Schema: baselineSchema,
		Limit:  6,
		Allowances: map[string]int{
			"compiler": 7,
		},
	}

	violations := findViolations(6, counts, baseline)
	if len(violations) != 1 {
		t.Fatalf("violations = %d, want 1", len(violations))
	}
	if violations[0].Path != "examples" {
		t.Fatalf("violation path = %s, want examples", violations[0].Path)
	}
}

func TestWriteBaselineIncludesOnlyOverBudgetDirectories(t *testing.T) {
	cfg := testConfig([]string{"compiler", "docs"})
	baseline := buildBaseline(cfg, []directoryCount{
		{Path: "compiler", Count: 7},
		{Path: "docs/audits", Count: 6},
	})

	if baseline.Schema != baselineSchema {
		t.Fatalf("schema = %s", baseline.Schema)
	}
	if got := baseline.Allowances["compiler"]; got != 7 {
		t.Fatalf("compiler allowance = %d, want 7", got)
	}
	if _, ok := baseline.Allowances["docs/audits"]; ok {
		t.Fatalf("docs/audits should not be in baseline at limit")
	}
}

func TestRunReportsViolations(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 7; i++ {
		writeFile(t, dir, filepath.Join("examples", fileName(i, ".tetra")), "fn main() {}\n")
	}

	var stdout, stderr bytes.Buffer
	exitCode := runDirectoryBudget([]string{"--roots", "examples"}, &stdout, &stderr, dir)
	if exitCode != 1 {
		t.Fatalf(
			"exit = %d, want 1; stdout=%q stderr=%q",
			exitCode,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stderr.String(), "examples: 7 active files") {
		t.Fatalf("stderr missing violation summary:\n%s", stderr.String())
	}
}

func TestRunWithBaselinePassesKnownViolation(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 7; i++ {
		writeFile(t, dir, filepath.Join("examples", fileName(i, ".tetra")), "fn main() {}\n")
	}
	baselinePath := filepath.Join(dir, "baseline.json")
	writeJSON(t, baselinePath, baselineFile{
		Schema: baselineSchema,
		Limit:  6,
		Allowances: map[string]int{
			"examples": 7,
		},
	})

	var stdout, stderr bytes.Buffer
	exitCode := runDirectoryBudget(
		[]string{"--roots", "examples", "--baseline", baselinePath},
		&stdout,
		&stderr,
		dir,
	)
	if exitCode != 0 {
		t.Fatalf(
			"exit = %d, want 0; stdout=%q stderr=%q",
			exitCode,
			stdout.String(),
			stderr.String(),
		)
	}
	if !strings.Contains(stdout.String(), "directory budget OK") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunWritesBaseline(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 7; i++ {
		writeFile(
			t,
			dir,
			filepath.Join("tools", "scriptstest", fileName(i, ".go")),
			"package scriptstest\n",
		)
	}
	baselinePath := filepath.Join(dir, "budget.json")

	var stdout, stderr bytes.Buffer
	exitCode := runDirectoryBudget(
		[]string{"--roots", "tools", "--write-baseline", baselinePath},
		&stdout,
		&stderr,
		dir,
	)
	if exitCode != 0 {
		t.Fatalf(
			"exit = %d, want 0; stdout=%q stderr=%q",
			exitCode,
			stdout.String(),
			stderr.String(),
		)
	}
	loaded, err := readBaseline(baselinePath)
	if err != nil {
		t.Fatalf("readBaseline: %v", err)
	}
	if got := loaded.Allowances["tools/scriptstest"]; got != 7 {
		t.Fatalf("tools/scriptstest allowance = %d, want 7", got)
	}
}

func testConfig(roots []string) scanConfig {
	return scanConfig{
		Limit:        6,
		Roots:        roots,
		Extensions:   extensionSet([]string{".go", ".tetra", ".sh", ".mjs", ".js", ".ts"}),
		DocsMarkdown: true,
		ExcludeDirs:  stringSet([]string{".cache", ".tetra_cache", "node_modules", "generated"}),
		ExcludePaths: normalizePaths([]string{"docs/assets"}),
	}
}

func mapCounts(counts []directoryCount) map[string]directoryCount {
	out := map[string]directoryCount{}
	for _, count := range counts {
		out[count.Path] = count
	}
	return out
}

func fileName(index int, ext string) string {
	return strings.TrimSuffix(strings.Repeat("x", index+1), "") + ext
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}
