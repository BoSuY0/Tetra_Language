package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateResidualRisksAcceptsOwnedBlockedRisk(t *testing.T) {
	path := writeResidualRisks(t, t.TempDir(), `{
  "schema": "tetra.release.residual-risks.v1",
  "release_version": "v0.4.0",
  "artifact": "residual-risks.json",
  "risks": [
    {
      "id": "v0.4.0-readiness-preflight",
      "severity": "critical",
      "owner": "release-owner",
      "status": "blocked",
      "summary": "readiness preflight failed",
      "evidence": "logs/01-readiness-preflight.log"
    }
  ]
}`)

	if err := validateResidualRisksFile(path, "v0.4.0"); err != nil {
		t.Fatalf("validator failed: %v", err)
	}
}

func TestValidateResidualRisksRejectsWrongVersion(t *testing.T) {
	path := writeResidualRisks(t, t.TempDir(), `{
  "schema": "tetra.release.residual-risks.v1",
  "release_version": "v0.3.0",
  "artifact": "residual-risks.json",
  "risks": []
}`)

	err := validateResidualRisksFile(path, "v0.4.0")
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), `release_version = "v0.3.0"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateResidualRisksRejectsUnownedCriticalRisk(t *testing.T) {
	path := writeResidualRisks(t, t.TempDir(), `{
  "schema": "tetra.release.residual-risks.v1",
  "release_version": "v0.4.0",
  "artifact": "residual-risks.json",
  "risks": [
    {"id":"risk-1","severity":"critical","owner":"TBD","status":"blocked"}
  ]
}`)

	err := validateResidualRisksFile(path, "v0.4.0")
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(
		err.Error(),
		"critical residual risk risk-1 requires known status and owner",
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateResidualRisksRejectsUnknownFields(t *testing.T) {
	path := writeResidualRisks(t, t.TempDir(), `{
  "schema": "tetra.release.residual-risks.v1",
  "release_version": "v0.4.0",
  "artifact": "residual-risks.json",
  "extra": true,
  "risks": []
}`)

	err := validateResidualRisksFile(path, "v0.4.0")
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateResidualRisksRejectsNullRisks(t *testing.T) {
	path := writeResidualRisks(t, t.TempDir(), `{
  "schema": "tetra.release.residual-risks.v1",
  "release_version": "v0.4.0",
  "artifact": "residual-risks.json",
  "risks": null
}`)

	err := validateResidualRisksFile(path, "v0.4.0")
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), "risks array required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeResidualRisks(t *testing.T, dir string, content string) string {
	t.Helper()
	path := filepath.Join(dir, "residual-risks.json")
	if err := os.WriteFile(path, []byte(content+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
