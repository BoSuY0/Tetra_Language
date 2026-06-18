package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/internal/toon"
)

func TestValidateEcoNeedMapAcceptsValidReport(t *testing.T) {
	out, err := runEcoNeedMapValidator(t, validNeedMapReport())
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateEcoNeedMapAcceptsTOON(t *testing.T) {
	toonRaw, err := toon.ConvertJSONToTOON(
		[]byte(validNeedMapReport()),
		toon.Options{Strict: true, Deterministic: true},
	)
	if err != nil {
		t.Fatalf("json->toon: %v", err)
	}
	if err := validateEcoNeedMapFormat(toonRaw, "toon"); err != nil {
		t.Fatalf("validateEcoNeedMapFormat TOON: %v\n%s", err, toonRaw)
	}
}

func TestValidateEcoNeedMapRejectsMalformedJSON(t *testing.T) {
	out, err := runEcoNeedMapValidator(t, `{"schema": "tetra.eco.needmap.v1",`)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unexpected EOF") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoNeedMapRejectsUnknownTopLevelField(t *testing.T) {
	report := strings.Replace(
		validNeedMapReport(),
		"\n  \"capsules\":",
		"\n  \"strict_extra\": true,\n  \"capsules\":",
		1,
	)
	out, err := runEcoNeedMapValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unknown field") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoNeedMapRejectsUnknownCapsuleField(t *testing.T) {
	report := strings.Replace(
		validNeedMapReport(),
		`"permissions": ["io"],`,
		`"permissions": ["io"], "unexpected": true,`,
		1,
	)
	out, err := runEcoNeedMapValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unknown field") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoNeedMapRejectsUnknownEdgeField(t *testing.T) {
	report := strings.Replace(
		validNeedMapReport(),
		`"version": "0.1.0"`,
		`"version": "0.1.0", "unexpected": true`,
		1,
	)
	out, err := runEcoNeedMapValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unknown field") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoNeedMapRejectsMissingRequiredField(t *testing.T) {
	report := strings.Replace(
		validNeedMapReport(),
		"  \"schema\": \"tetra.eco.needmap.v1\",\n",
		"",
		1,
	)
	out, err := runEcoNeedMapValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "schema is required") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoNeedMapRejectsBadLockHash(t *testing.T) {
	report := strings.Replace(
		validNeedMapReport(),
		`"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`,
		`"sha256:not-hex"`,
		1,
	)
	out, err := runEcoNeedMapValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "invalid lock_sha256") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoNeedMapRejectsEdgeToUnknownCapsule(t *testing.T) {
	report := strings.Replace(
		validNeedMapReport(),
		`"to_id": "tetra://core"`,
		`"to_id": "tetra://missing"`,
		1,
	)
	out, err := runEcoNeedMapValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "references unknown target") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoNeedMapRejectsTransitiveMismatch(t *testing.T) {
	report := strings.Replace(
		validNeedMapReport(),
		`"transitive_need_ids": ["tetra://core"]`,
		`"transitive_need_ids": []`,
		1,
	)
	out, err := runEcoNeedMapValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "transitive_need_ids mismatch") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoNeedMapRejectsTargetSetMismatch(t *testing.T) {
	report := strings.Replace(
		validNeedMapReport(),
		`  "targets": ["linux-x64"]`,
		`  "targets": ["wasm32-wasi"]`,
		1,
	)
	out, err := runEcoNeedMapValidator(t, report)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "targets mismatch") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func validNeedMapReport() string {
	return `{
  "schema": "tetra.eco.needmap.v1",
  "lock_sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "capsules": [
    {
      "id": "tetra://app",
      "version": "0.1.0",
      "targets": ["linux-x64"],
      "permissions": ["io"],
      "transitive_need_ids": ["tetra://core"]
    },
    {
      "id": "tetra://core",
      "version": "0.1.0",
      "targets": ["linux-x64"],
      "permissions": ["io"]
    }
  ],
  "edges": [
    {
      "from_id": "tetra://app",
      "to_id": "tetra://core",
      "version": "0.1.0"
    }
  ],
  "targets": ["linux-x64"]
}`
}

func runEcoNeedMapValidator(t *testing.T, report string) ([]byte, error) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "tetra.needmap.json")
	if err := os.WriteFile(path, []byte(report), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".", "--needmap", path)
	cmd.Dir = "."
	return cmd.CombinedOutput()
}
