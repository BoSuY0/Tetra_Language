package scriptstest

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV10APIDiffWorkflowLivesInVersionedReleaseScript(t *testing.T) {
	root := repoRoot(t)
	versionedPath := filepath.Join(root, "scripts", "release", "v1_0", "api-diff.sh")
	assertLegacyFileRemoved(t, "scripts/release_v1_0_api_diff.sh", "scripts/release/v1_0/api-diff.sh directly")
	versionedRaw, err := os.ReadFile(versionedPath)
	if err != nil {
		t.Fatalf("read versioned API diff script: %v", err)
	}
	versionedText := string(versionedRaw)
	for _, want := range []string{
		"Usage: bash scripts/release/v1_0/api-diff.sh",
		`release_artifact="tetra.release.v1_0.api-diff-report.v1alpha1"`,
		"git ls-files ':(glob)examples/*.tetra'",
		"go run ./tools/cmd/gen-docs",
		"node scripts/tools/api_diff_report.mjs",
	} {
		if !strings.Contains(versionedText, want) {
			t.Fatalf("scripts/release/v1_0/api-diff.sh missing %q", want)
		}
	}
	assertNoLegacyMention(t, versionedText, "scripts/release_v1_0_api_diff.sh", "scripts/release/v1_0/api-diff.sh")
}

func TestReleaseV10APIDiffRejectsMissingReportDirArgument(t *testing.T) {
	root := releaseV10APIDiffFakeRepo(t)

	out, err := runReleaseV10APIDiff(t, root, "--report-dir")
	if err == nil {
		t.Fatalf("expected missing --report-dir argument rejection\n%s", out)
	}
	if !strings.Contains(string(out), "release/v1_0/api-diff: --report-dir requires a directory") {
		t.Fatalf("missing report-dir argument output missing controlled error:\n%s", out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
}

func TestReleaseV10APIDiffRejectsNonDirectoryReportDirBeforeMkdir(t *testing.T) {
	root := releaseV10APIDiffFakeRepo(t)
	regularFile := filepath.Join(root, "report-file")
	if err := os.WriteFile(regularFile, []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	danglingSymlink := filepath.Join(root, "dangling-report-link")
	if err := os.Symlink(filepath.Join(root, "missing-report-target"), danglingSymlink); err != nil {
		t.Fatal(err)
	}
	fileTarget := filepath.Join(root, "report-file-target")
	if err := os.WriteFile(fileTarget, []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fileSymlink := filepath.Join(root, "file-report-link")
	if err := os.Symlink(fileTarget, fileSymlink); err != nil {
		t.Fatal(err)
	}

	for _, reportDir := range []string{regularFile, danglingSymlink, fileSymlink} {
		out, err := runReleaseV10APIDiff(t, root, "--report-dir", reportDir)
		if err == nil {
			t.Fatalf("expected non-directory report dir rejection for %s\n%s", reportDir, out)
		}
		for _, want := range []string{
			"release/v1_0/api-diff: refusing to use non-directory report path: " + reportDir,
			"release/v1_0/api-diff: choose a fresh --report-dir directory",
		} {
			if !strings.Contains(string(out), want) {
				t.Fatalf("non-directory report dir output missing %q:\n%s", want, out)
			}
		}
		assertOutputAvoidsRawPathUtilityErrors(t, out)
	}
}

func TestReleaseV10APIDiffRejectsSymlinkReportDirBeforeWork(t *testing.T) {
	root := releaseV10APIDiffFakeRepo(t)
	targetDir := filepath.Join(root, "report-target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	reportDir := filepath.Join(root, "report-link")
	if err := os.Symlink(targetDir, reportDir); err != nil {
		t.Fatal(err)
	}

	out, err := runReleaseV10APIDiff(t, root, "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected symlink report dir rejection\n%s", out)
	}
	if !strings.Contains(string(out), "release/v1_0/api-diff: refusing to use symlink report path: "+reportDir) {
		t.Fatalf("symlink report dir output missing controlled error:\n%s", out)
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
	if _, err := os.Stat(filepath.Join(targetDir, "api-docs.md")); !os.IsNotExist(err) {
		t.Fatalf("symlink report dir should block before writing artifacts, stat err = %v", err)
	}
}

func TestReleaseV10APIDiffRejectsNonEmptyReportDirBeforeWork(t *testing.T) {
	root := releaseV10APIDiffFakeRepo(t)
	reportDir := filepath.Join(root, "report")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportDir, "api-diff.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runReleaseV10APIDiff(t, root, "--report-dir", reportDir)
	if err == nil {
		t.Fatalf("expected non-empty report dir rejection\n%s", out)
	}
	for _, want := range []string{
		"release/v1_0/api-diff: refusing to reuse non-empty report directory: " + reportDir,
		"release/v1_0/api-diff: choose a fresh --report-dir so stale reports cannot be reused",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("non-empty report dir output missing %q:\n%s", want, out)
		}
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
}

func TestReleaseV10APIDiffRejectsDashPrefixedNonEmptyReportDirBeforeRawFind(t *testing.T) {
	root := releaseV10APIDiffFakeRepo(t)
	reportDirArg := "-stale-api-diff"
	reportDirPath := filepath.Join(root, reportDirArg)
	if err := os.MkdirAll(reportDirPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reportDirPath, "api-docs.md"), []byte("# stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := runReleaseV10APIDiff(t, root, "--report-dir", reportDirArg)
	if err == nil {
		t.Fatalf("expected dash-prefixed non-empty report dir rejection\n%s", out)
	}
	expectedReportDir := normalizeDashLeadingPathForTest(reportDirArg)
	for _, want := range []string{
		"release/v1_0/api-diff: refusing to reuse non-empty report directory: " + expectedReportDir,
		"release/v1_0/api-diff: choose a fresh --report-dir so stale reports cannot be reused",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("dash-prefixed non-empty report dir output missing %q:\n%s", want, out)
		}
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
}

func TestReleaseV10APIDiffRejectsMissingBaselineBeforeDocsWork(t *testing.T) {
	root := releaseV10APIDiffToolFakeRepo(t)
	reportDir := filepath.Join(root, "report")
	missingBaseline := filepath.Join(root, "docs", "baselines", "missing-api-baseline.json")

	out, err := runReleaseV10APIDiff(t, root, "--report-dir", reportDir, "--baseline", missingBaseline)
	if err == nil {
		t.Fatalf("expected missing baseline rejection\n%s", out)
	}
	for _, want := range []string{
		"release/v1_0/api-diff: missing baseline " + missingBaseline,
		"release/v1_0/api-diff: rerun with --write-baseline to create it",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("missing baseline output missing %q:\n%s", want, out)
		}
	}
	assertOutputAvoidsRawPathUtilityErrors(t, out)
	for _, path := range []string{
		filepath.Join(root, "gen-docs.log"),
		filepath.Join(root, "validate-api-docs.log"),
		filepath.Join(reportDir, "api-docs.md"),
		filepath.Join(reportDir, "api-diff.json"),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("missing baseline should block before docs work/artifact write: %s stat err = %v", path, err)
		}
	}
}

func TestAPIDiffReportClassifiesSignatureDriftAsChanged(t *testing.T) {
	dir := t.TempDir()
	baselinePath := filepath.Join(dir, "baseline.json")
	docsPath := filepath.Join(dir, "api-docs.md")
	diffPath := filepath.Join(dir, "diff.json")

	baseline := `{
  "schema": "tetra.api.diff-baseline.v1alpha1",
  "created_at": "2026-04-27T00:00:00Z",
  "source_docs": "baseline.md",
  "source_docs_sha256": "sha256:baseline",
  "api_metadata": {
    "schema": "tetra.api.v1alpha1",
    "api_hash": "sha256:baseline",
    "module_count": 1,
    "entry_count": 1
  },
  "symbols": [
    {
      "id": "examples/flow_hello.tetra::Functions::func main() -> Int uses io",
      "module": "examples/flow_hello.tetra",
      "section": "Functions",
      "entry": "func main() -> Int uses io",
      "symbol_hash": "sha256:old"
    }
  ]
}`
	docs := `# Tetra API Docs

<!-- tetra-api-metadata: {"schema":"tetra.api.v1alpha1","api_hash":"sha256:3df89cec58b30743f96d0339ac6acdfb6ab628dd1269b353290f4a6c04da29c2","module_count":1,"entry_count":1} -->

## examples/flow_hello.tetra

### Functions

- ` + "`func main() -> i32 uses io`" + `
`
	if err := os.WriteFile(baselinePath, []byte(baseline), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(docsPath, []byte(docs), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("node", filepath.Join(repoRoot(t), "scripts", "tools", "api_diff_report.mjs"), "--docs", docsPath, "--baseline", baselinePath, "--diff-out", diffPath, "--enforce", "none")
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("api diff report failed: %v\n%s", err, out)
	}

	raw, err := os.ReadFile(diffPath)
	if err != nil {
		t.Fatal(err)
	}
	var diff struct {
		Summary struct {
			Added   int `json:"added"`
			Removed int `json:"removed"`
			Changed int `json:"changed"`
		} `json:"summary"`
		Changes []struct {
			Kind         string `json:"kind"`
			ID           string `json:"id"`
			BeforeEntry  string `json:"before_entry"`
			AfterEntry   string `json:"after_entry"`
			ReviewStatus string `json:"review_status"`
			ReviewNote   string `json:"review_note"`
		} `json:"changes"`
		Review struct {
			Status    string   `json:"status"`
			Checklist []string `json:"checklist"`
		} `json:"review"`
	}
	if err := json.Unmarshal(raw, &diff); err != nil {
		t.Fatalf("decode diff: %v\n%s", err, string(raw))
	}
	if diff.Summary.Added != 0 || diff.Summary.Removed != 0 || diff.Summary.Changed != 1 {
		t.Fatalf("unexpected summary: %+v\n%s", diff.Summary, string(raw))
	}
	if len(diff.Changes) != 1 || diff.Changes[0].Kind != "changed" {
		t.Fatalf("expected one changed entry:\n%s", string(raw))
	}
	if !strings.Contains(diff.Changes[0].ID, "func main") {
		t.Fatalf("changed id should keep stable symbol identity: %#v", diff.Changes[0])
	}
	if diff.Changes[0].BeforeEntry != "func main() -> Int uses io" || diff.Changes[0].AfterEntry != "func main() -> i32 uses io" {
		t.Fatalf("changed entry should preserve before/after text: %#v", diff.Changes[0])
	}
	if diff.Changes[0].ReviewStatus != "breaking_requires_review" || !strings.Contains(diff.Changes[0].ReviewNote, "signature or metadata changed") {
		t.Fatalf("changed entry should include review status/note: %#v", diff.Changes[0])
	}
	if diff.Review.Status != "needs_review" || len(diff.Review.Checklist) < 3 {
		t.Fatalf("diff should include review checklist: %#v", diff.Review)
	}
}

func TestAPIDiffReportClassifiesAdditionsAndRemovalsForReview(t *testing.T) {
	dir := t.TempDir()
	baselinePath := filepath.Join(dir, "baseline.json")
	docsPath := filepath.Join(dir, "api-docs.md")
	diffPath := filepath.Join(dir, "diff.json")

	baseline := `{
  "schema": "tetra.api.diff-baseline.v1alpha1",
  "created_at": "2026-04-27T00:00:00Z",
  "source_docs": "baseline.md",
  "source_docs_sha256": "sha256:baseline",
  "api_metadata": {
    "schema": "tetra.api.v1alpha1",
    "api_hash": "sha256:baseline",
    "module_count": 1,
    "entry_count": 1
  },
  "symbols": [
    {
      "id": "examples/flow_hello.tetra::Functions::func old_api",
      "module": "examples/flow_hello.tetra",
      "section": "Functions",
      "entry": "func old_api() -> i32",
      "symbol_hash": "sha256:old"
    }
  ]
}`
	docs := `# Tetra API Docs

<!-- tetra-api-metadata: {"schema":"tetra.api.v1alpha1","api_hash":"sha256:f52b7513d8d95c9146fe3fb94cb82622b94874218f6414cff55b09a29d3b02ff","module_count":1,"entry_count":1} -->

## examples/flow_hello.tetra

### Functions

- ` + "`func new_api() -> i32`" + `
`
	if err := os.WriteFile(baselinePath, []byte(baseline), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(docsPath, []byte(docs), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("node", filepath.Join(repoRoot(t), "scripts", "tools", "api_diff_report.mjs"), "--docs", docsPath, "--baseline", baselinePath, "--diff-out", diffPath, "--enforce", "none")
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("api diff report failed: %v\n%s", err, out)
	}

	raw, err := os.ReadFile(diffPath)
	if err != nil {
		t.Fatal(err)
	}
	var diff struct {
		Changes []struct {
			Kind         string `json:"kind"`
			ReviewStatus string `json:"review_status"`
		} `json:"changes"`
	}
	if err := json.Unmarshal(raw, &diff); err != nil {
		t.Fatalf("decode diff: %v\n%s", err, string(raw))
	}
	statusByKind := map[string]string{}
	for _, change := range diff.Changes {
		statusByKind[change.Kind] = change.ReviewStatus
	}
	if statusByKind["added"] != "addition_requires_scope_review" {
		t.Fatalf("added review status = %q\n%s", statusByKind["added"], string(raw))
	}
	if statusByKind["removed"] != "breaking_requires_review" {
		t.Fatalf("removed review status = %q\n%s", statusByKind["removed"], string(raw))
	}
}

func TestAPIDiffBaselineWriteIsDeterministicWithoutWallClockTimestamp(t *testing.T) {
	dir := t.TempDir()
	docsPath := filepath.Join(dir, "api-docs.md")
	baselinePath := filepath.Join(dir, "baseline.json")
	docs := `# Tetra API Docs

<!-- tetra-api-metadata: {"schema":"tetra.api.v1alpha1","api_hash":"sha256:3df89cec58b30743f96d0339ac6acdfb6ab628dd1269b353290f4a6c04da29c2","module_count":1,"entry_count":1} -->

## examples/flow_hello.tetra

### Functions

- ` + "`func main() -> i32 uses io`" + `
`
	if err := os.WriteFile(docsPath, []byte(docs), 0o644); err != nil {
		t.Fatal(err)
	}

	run := func() []byte {
		t.Helper()
		cmd := exec.Command("node", filepath.Join(repoRoot(t), "scripts", "tools", "api_diff_report.mjs"), "--docs", docsPath, "--baseline", baselinePath, "--write-baseline")
		cmd.Dir = repoRoot(t)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("api diff baseline write failed: %v\n%s", err, out)
		}
		raw, err := os.ReadFile(baselinePath)
		if err != nil {
			t.Fatalf("read baseline: %v", err)
		}
		return raw
	}

	first := run()
	second := run()
	if string(first) != string(second) {
		t.Fatalf("baseline should be byte-stable across identical writes\nfirst:\n%s\nsecond:\n%s", string(first), string(second))
	}
	if !strings.Contains(string(first), `"created_at": "1970-01-01T00:00:00Z"`) {
		t.Fatalf("baseline should use deterministic created_at:\n%s", string(first))
	}
}

func releaseV10APIDiffFakeRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scripts", "release", "v1_0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(filepath.Join(repoRoot(t), "scripts", "release", "v1_0", "api-diff.sh"), filepath.Join(root, "scripts", "release", "v1_0", "api-diff.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func runReleaseV10APIDiff(t *testing.T, root string, args ...string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command("bash", append([]string{"scripts/release/v1_0/api-diff.sh"}, args...)...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
	return cmd.CombinedOutput()
}

func releaseV10APIDiffToolFakeRepo(t *testing.T) string {
	t.Helper()
	root := releaseV10APIDiffFakeRepo(t)
	for _, dir := range []string{
		"bin",
		"docs/baselines",
		"examples",
	} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "examples", "flow_hello.tetra"), []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "git"), []byte(`#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "ls-files" ]]; then
  printf 'examples/flow_hello.tetra\n'
fi
`), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "go"), []byte(`#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" != "run" ]]; then
  exit 0
fi
shift
tool="${1:-}"
case "$tool" in
  ./tools/cmd/gen-docs)
    printf 'gen-docs\n' >>gen-docs.log
    printf '# API Docs\n'
    ;;
  ./tools/cmd/validate-api-docs)
    printf 'validate-api-docs\n' >>validate-api-docs.log
    ;;
esac
`), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}
