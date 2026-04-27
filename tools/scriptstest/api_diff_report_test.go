package scriptstest

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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
