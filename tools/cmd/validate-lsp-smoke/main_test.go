package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateLSPSmokeAcceptsValidAnalysis(t *testing.T) {
	report := `{
  "uri": "examples/flow_hello.tetra",
  "diagnostics": [],
  "symbols": [
    {"name": "main", "kind": "function", "line": 1, "column": 5, "detail": "func main() -> Int uses io"}
  ],
  "hovers": [
    {"name": "main", "line": 1, "column": 5, "contents": "func main() -> Int uses io"}
  ]
}`
	out, err := runLSPValidator(t, report)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateLSPSmokeRejectsNullCollections(t *testing.T) {
	report := `{"uri":"sample.tetra","diagnostics":null,"symbols":null,"hovers":null}`
	out, err := runLSPValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "diagnostics must be an array") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPSmokeRejectsUnknownEnvelopeField(t *testing.T) {
	report := `{
  "uri": "sample.tetra",
  "diagnostics": [],
  "symbols": [],
  "hovers": [],
  "extra": true
}`
	out, err := runLSPValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unknown field") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPSmokeRejectsUnknownItemFields(t *testing.T) {
	tests := []struct {
		name   string
		report string
	}{
		{
			name: "diagnostic",
			report: `{
  "uri": "sample.tetra",
  "diagnostics": [{"message": "bad", "severity": "error", "line": 1, "column": 1, "extra": true}],
  "symbols": [],
  "hovers": []
}`,
		},
		{
			name: "symbol",
			report: `{
  "uri": "sample.tetra",
  "diagnostics": [],
  "symbols": [{"name": "main", "kind": "function", "line": 1, "column": 5, "extra": true}],
  "hovers": []
}`,
		},
		{
			name: "hover",
			report: `{
  "uri": "sample.tetra",
  "diagnostics": [],
  "symbols": [{"name": "main", "kind": "function", "line": 1, "column": 5}],
  "hovers": [{"name": "main", "line": 1, "column": 5, "contents": "func main() -> Int", "extra": true}]
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := runLSPValidator(t, tt.report)
			if err == nil {
				t.Fatalf("expected validator failure\n%s", out)
			}
			if !strings.Contains(string(out), "unknown field") {
				t.Fatalf("unexpected output:\n%s", out)
			}
		})
	}
}

func TestValidateLSPSmokeRejectsSymbolWithoutPosition(t *testing.T) {
	report := `{
  "uri": "sample.tetra",
  "diagnostics": [],
  "symbols": [{"name": "main", "kind": "function", "line": 0, "column": 0}],
  "hovers": []
}`
	out, err := runLSPValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "symbol main has invalid position") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPSmokeRejectsInvalidDiagnosticSeverity(t *testing.T) {
	report := `{
  "uri": "sample.tetra",
  "diagnostics": [{"message": "bad", "severity": "fatal", "line": 1, "column": 1}],
  "symbols": [],
  "hovers": []
}`
	out, err := runLSPValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "invalid severity") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPSmokeRejectsHoverWithoutSymbol(t *testing.T) {
	report := `{
  "uri": "sample.tetra",
  "diagnostics": [],
  "symbols": [],
  "hovers": [{"name": "main", "line": 1, "column": 5, "contents": "func main() -> Int"}]
}`
	out, err := runLSPValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "has no matching symbol") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPSmokeRejectsNonTetraURI(t *testing.T) {
	report := `{
  "uri": "sample.txt",
  "diagnostics": [],
  "symbols": [],
  "hovers": []
}`
	out, err := runLSPValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "uri must reference a .tetra file") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPSmokeRejectsDuplicateSymbol(t *testing.T) {
	report := `{
  "uri": "sample.tetra",
  "diagnostics": [],
  "symbols": [
    {"name": "main", "kind": "function", "line": 1, "column": 5},
    {"name": "main", "kind": "function", "line": 1, "column": 5}
  ],
  "hovers": []
}`
	out, err := runLSPValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "duplicate symbol main") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPSmokeRejectsDuplicateHover(t *testing.T) {
	report := `{
  "uri": "sample.tetra",
  "diagnostics": [],
  "symbols": [{"name": "main", "kind": "function", "line": 1, "column": 5}],
  "hovers": [
    {"name": "main", "line": 1, "column": 5, "contents": "func main() -> Int"},
    {"name": "main", "line": 1, "column": 5, "contents": "func main() -> Int"}
  ]
}`
	out, err := runLSPValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "duplicate hover main") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateLSPSmokeRejectsHoverWithoutContents(t *testing.T) {
	report := `{
  "uri": "sample.tetra",
  "diagnostics": [],
  "symbols": [{"name": "main", "kind": "function", "line": 1, "column": 5}],
  "hovers": [{"name": "main", "line": 1, "column": 5, "contents": ""}]
}`
	out, err := runLSPValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "hover main missing contents") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func runLSPValidator(t *testing.T, report string) ([]byte, error) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "lsp.json")
	if err := os.WriteFile(path, []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".", "--report", path)
	cmd.Dir = "."
	return cmd.CombinedOutput()
}
