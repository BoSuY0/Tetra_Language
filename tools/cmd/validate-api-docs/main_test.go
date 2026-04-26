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

## examples/flow_hello.tetra

### Functions

- ` + "`func main() -> Int uses io`" + `
`
	out, err := runAPIDocsValidator(t, docs)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
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
