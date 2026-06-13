package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/internal/toon"
)

func TestValidateEcoMaterializationAcceptsValidReport(t *testing.T) {
	out, err := runEcoMaterializationValidator(t, validMaterializationReport())
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateEcoMaterializationAcceptsTOON(t *testing.T) {
	toonRaw, err := toon.ConvertJSONToTOON([]byte(validMaterializationReport()), toon.Options{Strict: true, Deterministic: true})
	if err != nil {
		t.Fatalf("json->toon: %v", err)
	}
	if err := validateEcoMaterializationFormat(toonRaw, "toon"); err != nil {
		t.Fatalf("validateEcoMaterializationFormat TOON: %v\n%s", err, toonRaw)
	}
}

func TestValidateEcoMaterializationAcceptsEmptyTarget(t *testing.T) {
	report := strings.Replace(validMaterializationReport(), `"target": "linux-x64"`, `"target": ""`, 1)
	out, err := runEcoMaterializationValidator(t, report)
	if err != nil {
		t.Fatalf("validator should accept unscoped materialization target: %v\n%s", err, out)
	}
}

func TestValidateEcoMaterializationRejectsMalformedJSON(t *testing.T) {
	out, err := runEcoMaterializationValidator(t, `{"schema": "tetra.eco.materialization.v1",`)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unexpected EOF") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoMaterializationRejectsUnknownField(t *testing.T) {
	report := strings.Replace(validMaterializationReport(), "\n  \"target\":", "\n  \"strict_extra\": true,\n  \"target\":", 1)
	out, err := runEcoMaterializationValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unknown field") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoMaterializationRejectsMissingRequiredField(t *testing.T) {
	report := strings.Replace(validMaterializationReport(), "  \"target\": \"linux-x64\",\n", "", 1)
	out, err := runEcoMaterializationValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "target is required") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoMaterializationRejectsUnsupportedSchema(t *testing.T) {
	report := strings.Replace(validMaterializationReport(), `"tetra.eco.materialization.v1"`, `"tetra.eco.materialization.v2"`, 1)
	out, err := runEcoMaterializationValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), `unsupported materialization schema "tetra.eco.materialization.v2"`) {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoMaterializationRejectsUnsupportedTarget(t *testing.T) {
	report := strings.Replace(validMaterializationReport(), `"target": "linux-x64"`, `"target": "plan9-riscv64"`, 1)
	out, err := runEcoMaterializationValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unsupported target plan9-riscv64") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoMaterializationRejectsEmptyPackagePath(t *testing.T) {
	report := strings.Replace(validMaterializationReport(), `"package_path": "dist/app.todex"`, `"package_path": ""`, 1)
	out, err := runEcoMaterializationValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "package_path is required") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoMaterializationRejectsUnnormalizedMaterializedDir(t *testing.T) {
	report := strings.Replace(validMaterializationReport(), `"materialized_dir": "out/materialized"`, `"materialized_dir": "out/../materialized"`, 1)
	out, err := runEcoMaterializationValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "materialized_dir path is not normalized") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoMaterializationRejectsBadLockHash(t *testing.T) {
	report := strings.Replace(validMaterializationReport(), `"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`, `"sha256:not-hex"`, 1)
	out, err := runEcoMaterializationValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "invalid lock_sha256") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoMaterializationRejectsEmptyTrustSnapshot(t *testing.T) {
	report := strings.Replace(validMaterializationReport(), `"trust_snapshot": "reports/tetra.trust-snapshot.json"`, `"trust_snapshot": ""`, 1)
	out, err := runEcoMaterializationValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "trust_snapshot is required when present") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func validMaterializationReport() string {
	return `{
  "schema": "tetra.eco.materialization.v1",
  "target": "linux-x64",
  "package_path": "dist/app.todex",
  "materialized_dir": "out/materialized",
  "trust_snapshot": "reports/tetra.trust-snapshot.json",
  "lock_sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}`
}

func runEcoMaterializationValidator(t *testing.T, report string) ([]byte, error) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "tetra.materialization.json")
	if err := os.WriteFile(path, []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".", "--materialization", path)
	cmd.Dir = "."
	return cmd.CombinedOutput()
}
