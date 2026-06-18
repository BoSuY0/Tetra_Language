package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateReadinessBlockersAcceptsBlockedV040Artifact(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "01-readiness-preflight.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		logPath,
		[]byte("validate-v0-4-readiness: feature blocker\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	artifact := writeReadinessBlockers(t, dir, `{
  "schema": "tetra.release.v0_4_0.readiness-blockers.v1",
  "release_version": "v0.4.0",
  "artifact": "readiness-blockers.json",
  "source_log": "logs/01-readiness-preflight.log",
  "blockers": [
    {
      "id": "readiness-preflight",
      "status": "blocked",
      "summary": "v0.4.0 readiness preflight failed",
      "detail": "validate-v0-4-readiness: feature blocker"
    }
  ]
}`)

	if err := validateReadinessBlockersFile(artifact, dir, "v0.4.0"); err != nil {
		t.Fatalf("validator failed: %v", err)
	}
}

func TestValidateReadinessBlockersRejectsWrongVersion(t *testing.T) {
	dir := t.TempDir()
	artifact := writeReadinessBlockers(t, dir, `{
  "schema": "tetra.release.v0_4_0.readiness-blockers.v1",
  "release_version": "v0.3.0",
  "artifact": "readiness-blockers.json",
  "source_log": "logs/01-readiness-preflight.log",
  "blockers": [
    {"id":"readiness-preflight","status":"blocked","summary":"blocked","detail":"details"}
  ]
}`)

	err := validateReadinessBlockersFile(artifact, dir, "v0.4.0")
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), `release_version = "v0.3.0"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReadinessBlockersRejectsUnsafeSourceLog(t *testing.T) {
	dir := t.TempDir()
	artifact := writeReadinessBlockers(t, dir, `{
  "schema": "tetra.release.v0_4_0.readiness-blockers.v1",
  "release_version": "v0.4.0",
  "artifact": "readiness-blockers.json",
  "source_log": "../outside.log",
  "blockers": [
    {"id":"readiness-preflight","status":"blocked","summary":"blocked","detail":"details"}
  ]
}`)

	err := validateReadinessBlockersFile(artifact, dir, "v0.4.0")
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), "unsafe source_log") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReadinessBlockersRejectsEmptyBlockers(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "logs", "01-readiness-preflight.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, []byte("validate-v0-4-readiness: blocked\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	artifact := writeReadinessBlockers(t, dir, `{
  "schema": "tetra.release.v0_4_0.readiness-blockers.v1",
  "release_version": "v0.4.0",
  "artifact": "readiness-blockers.json",
  "source_log": "logs/01-readiness-preflight.log",
  "blockers": []
}`)

	err := validateReadinessBlockersFile(artifact, dir, "v0.4.0")
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), "blockers must not be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateReadinessBlockersRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	artifact := writeReadinessBlockers(t, dir, `{
  "schema": "tetra.release.v0_4_0.readiness-blockers.v1",
  "release_version": "v0.4.0",
  "artifact": "readiness-blockers.json",
  "source_log": "logs/01-readiness-preflight.log",
  "extra": true,
  "blockers": [
    {"id":"readiness-preflight","status":"blocked","summary":"blocked","detail":"details"}
  ]
}`)

	err := validateReadinessBlockersFile(artifact, dir, "v0.4.0")
	if err == nil {
		t.Fatalf("expected validator failure")
	}
	if !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeReadinessBlockers(t *testing.T, dir string, content string) string {
	t.Helper()
	artifact := filepath.Join(dir, "artifacts", "readiness-blockers.json")
	if err := os.MkdirAll(filepath.Dir(artifact), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifact, []byte(content+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return artifact
}
