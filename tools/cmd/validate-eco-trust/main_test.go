package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/internal/toon"
)

func TestValidateEcoTrustAcceptsValidSnapshot(t *testing.T) {
	out, err := runEcoTrustValidator(t, validTrustSnapshot())
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateEcoTrustAcceptsTOON(t *testing.T) {
	toonRaw, err := toon.ConvertJSONToTOON(
		[]byte(validTrustSnapshot()),
		toon.Options{Strict: true, Deterministic: true},
	)
	if err != nil {
		t.Fatalf("json->toon: %v", err)
	}
	if err := validateEcoTrustSnapshotFormat(toonRaw, "toon"); err != nil {
		t.Fatalf("validateEcoTrustSnapshotFormat TOON: %v\n%s", err, toonRaw)
	}
}

func TestValidateEcoTrustRejectsMalformedJSON(t *testing.T) {
	out, err := runEcoTrustValidator(t, `{"schema": "tetra.eco.trust-snapshot.v1",`)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unexpected EOF") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoTrustRejectsUnknownTopLevelField(t *testing.T) {
	snapshot := strings.Replace(
		validTrustSnapshot(),
		"\n  \"record_count\":",
		"\n  \"strict_extra\": true,\n  \"record_count\":",
		1,
	)
	out, err := runEcoTrustValidator(t, snapshot)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unknown field") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoTrustRejectsUnknownCapsuleField(t *testing.T) {
	snapshot := strings.Replace(
		validTrustSnapshot(),
		`"trust_score": 95,`,
		`"trust_score": 95, "unexpected": true,`,
		1,
	)
	out, err := runEcoTrustValidator(t, snapshot)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unknown field") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoTrustRejectsMissingRequiredTopLevelField(t *testing.T) {
	snapshot := strings.Replace(validTrustSnapshot(), "  \"generated_at_unix\": 0,\n", "", 1)
	out, err := runEcoTrustValidator(t, snapshot)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "generated_at_unix is required") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoTrustRejectsMissingRequiredCapsuleField(t *testing.T) {
	snapshot := strings.Replace(validTrustSnapshot(), `      "trust_tier": "high",`+"\n", "", 1)
	out, err := runEcoTrustValidator(t, snapshot)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "trust_tier is required") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoTrustRejectsBadHash(t *testing.T) {
	snapshot := strings.Replace(
		validTrustSnapshot(),
		`"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`,
		`"sha256:not-hex"`,
		1,
	)
	out, err := runEcoTrustValidator(t, snapshot)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "invalid lock_sha256") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoTrustRejectsNegativeRecordCount(t *testing.T) {
	snapshot := strings.Replace(
		validTrustSnapshot(),
		`  "record_count": 2,`,
		`  "record_count": -1,`,
		1,
	)
	out, err := runEcoTrustValidator(t, snapshot)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "record_count must not be negative") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoTrustRejectsRecordCountMismatch(t *testing.T) {
	snapshot := strings.Replace(
		validTrustSnapshot(),
		`  "record_count": 2,`,
		`  "record_count": 3,`,
		1,
	)
	out, err := runEcoTrustValidator(t, snapshot)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "record_count mismatch") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoTrustRejectsTierScoreMismatch(t *testing.T) {
	snapshot := strings.Replace(
		validTrustSnapshot(),
		`"trust_tier": "high"`,
		`"trust_tier": "medium"`,
		1,
	)
	out, err := runEcoTrustValidator(t, snapshot)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "trust_tier mismatch") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoTrustRejectsDuplicateCapsule(t *testing.T) {
	snapshot := strings.Replace(
		validTrustSnapshot(),
		`"id": "tetra://core"`,
		`"id": "tetra://app"`,
		1,
	)
	out, err := runEcoTrustValidator(t, snapshot)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "duplicate capsule id") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func validTrustSnapshot() string {
	return `{
  "schema": "tetra.eco.trust-snapshot.v1",
  "generated_at_unix": 0,
  "lock_sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "vault_sha256": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
  "record_count": 2,
  "capsules": [
    {
      "id": "tetra://app",
      "version": "0.1.0",
      "permissions": ["io"],
      "trust_tier": "high",
      "trust_score": 95,
      "trust_reasons": ["permissions=io"]
    },
    {
      "id": "tetra://core",
      "version": "0.1.0",
      "permissions": ["mem", "mmio", "capability"],
      "trust_tier": "low",
      "trust_score": 55,
      "trust_reasons": ["permissions=mem,mmio,capability"]
    }
  ]
}`
}

func runEcoTrustValidator(t *testing.T, snapshot string) ([]byte, error) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "tetra.trust-snapshot.json")
	if err := os.WriteFile(path, []byte(snapshot), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".", "--trust", path)
	cmd.Dir = "."
	return cmd.CombinedOutput()
}
