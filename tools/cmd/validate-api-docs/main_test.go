package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAPIDocsAcceptsGeneratedShape(t *testing.T) {
	docs := `# Tetra API Docs

<!-- tetra-api-metadata: {"schema":"tetra.api.v1alpha1","api_hash":"sha256:6bb4b0eb4a5e074052e52e8f4587d7d611f8e469e98b33c4fcfa78ef965cdf2a","module_count":1,"entry_count":1} -->

## examples/flow/flow_hello.tetra

### Functions

- ` + "`func main() -> Int uses io`" + `
`
	out, err := runAPIDocsValidator(t, docs)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateAPIDocsRejectsMissingAPIMetadata(t *testing.T) {
	docs := `# Tetra API Docs

## examples/flow/flow_hello.tetra

### Functions

- ` + "`func main() -> Int uses io`" + `
`
	out, err := runAPIDocsValidator(t, docs)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing tetra-api-metadata") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateAPIDocsRejectsAPIMetadataHashMismatch(t *testing.T) {
	docs := `# Tetra API Docs

<!-- tetra-api-metadata: {"schema":"tetra.api.v1alpha1","api_hash":"sha256:0000000000000000000000000000000000000000000000000000000000000000","module_count":1,"entry_count":1} -->

## examples/flow/flow_hello.tetra

### Functions

- ` + "`func main() -> Int uses io`" + `
`
	out, err := runAPIDocsValidator(t, docs)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "API metadata api_hash mismatch") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateAPIDocsRejectsUnknownMetadataFields(t *testing.T) {
	docs := `# Tetra API Docs

<!-- tetra-api-metadata: {"schema":"tetra.api.v1alpha1","api_hash":"sha256:a5813045590b999abb9088185f7ee73c1d75b281dbbacfd2ea16fda06106dc36","module_count":1,"entry_count":1,"extra":true} -->

## examples/flow/flow_hello.tetra

### Functions

- ` + "`func main() -> Int uses io`" + `
`
	out, err := runAPIDocsValidator(t, docs)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unknown field") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateAPIDocsRejectsEmptyDocument(t *testing.T) {
	out, err := runAPIDocsValidator(t, "")
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "API docs are empty") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateAPIDocsRejectsMissingAPIEntries(t *testing.T) {
	docs := `# Tetra API Docs

<!-- tetra-api-metadata: {"schema":"tetra.api.v1alpha1","api_hash":"sha256:a38d7b1b91cba7f01c3feb0728d5bdf744b927068fee1fa4e300953670622629","module_count":1,"entry_count":0} -->

## examples/empty.tetra

### Functions
`
	out, err := runAPIDocsValidator(t, docs)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing API entry bullets") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateAPIDocsRejectsErrorOutput(t *testing.T) {
	out, err := runAPIDocsValidator(t, "error: parser failed\n")
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing # Tetra API Docs heading") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateAPIDocsRejectsDuplicateModules(t *testing.T) {
	docs := `# Tetra API Docs

<!-- tetra-api-metadata: {"schema":"tetra.api.v1alpha1","api_hash":"sha256:f7c1f415fc85300357fe5db084ac841e64b3492e60b353a5d1a7e3a2c4d39328","module_count":2,"entry_count":2} -->

## a.tetra

### Functions

- ` + "`func a() -> Int`" + `

## a.tetra

### Functions

- ` + "`func b() -> Int`" + `
`
	out, err := runAPIDocsValidator(t, docs)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "duplicate module heading") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateAPIDocsRejectsUnknownSection(t *testing.T) {
	docs := `# Tetra API Docs

<!-- tetra-api-metadata: {"schema":"tetra.api.v1alpha1","api_hash":"sha256:97530ef10760f6f6962250a0da46f4cd76a76af5d5fc31bc4a4c024e58fd6217","module_count":1,"entry_count":1} -->

## a.tetra

### Widgets

- ` + "`widget A`" + `
`
	out, err := runAPIDocsValidator(t, docs)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unknown API section") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateAPIDocsRejectsEntryBeforeSection(t *testing.T) {
	docs := `# Tetra API Docs

<!-- tetra-api-metadata: {"schema":"tetra.api.v1alpha1","api_hash":"sha256:7102027c286a490a13f5e2f20549e0f42fa88c4f2b479d3290ab99da1dc522d3","module_count":1,"entry_count":1} -->

## a.tetra

- ` + "`func a() -> Int`" + `
`
	out, err := runAPIDocsValidator(t, docs)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "API entry appears before module section") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateAPIDocsRejectsBrokenInternalLink(t *testing.T) {
	docs := `# Tetra API Docs

<!-- tetra-api-metadata: {"schema":"tetra.api.v1alpha1","api_hash":"sha256:6bb4b0eb4a5e074052e52e8f4587d7d611f8e469e98b33c4fcfa78ef965cdf2a","module_count":1,"entry_count":1} -->

See [missing](#missing-section).

## examples/flow/flow_hello.tetra

### Functions

- ` + "`func main() -> Int uses io`" + `
`
	out, err := runAPIDocsValidator(t, docs)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "broken internal link") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func runAPIDocsValidator(t *testing.T, docs string) ([]byte, error) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "api-docs.md")
	if err := os.WriteFile(path, []byte(docs), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".", "--docs", path)
	cmd.Dir = "."
	return cmd.CombinedOutput()
}
