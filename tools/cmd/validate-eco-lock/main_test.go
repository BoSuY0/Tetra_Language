package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateEcoLockAcceptsDependencyGraph(t *testing.T) {
	lock := `{
  "schema": "tetra.eco.lock.v1",
  "manifest_schema": "tetra.capsule.v1",
  "permissions_model": "tetra.eco.permissions.v1",
  "graph_sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "capsules": [
    {
      "id": "tetra://app",
      "name": "App",
      "version": "0.1.0",
      "path": "/tmp/project/Tetra.capsule",
      "targets": ["linux-x64"],
      "permissions": ["io"],
      "dependencies": [{"id": "tetra://core", "version": "0.1.0"}]
    },
    {
      "id": "tetra://core",
      "name": "Core",
      "version": "0.1.0",
      "path": "/tmp/Core.capsule",
      "targets": ["linux-x64"],
      "permissions": ["io"]
    }
  ]
}`
	out, err := runEcoLockValidator(t, lock)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateEcoLockRejectsUnsupportedPermissionsModel(t *testing.T) {
	lock := `{
  "permissions_model": "tetra.eco.permissions.v2",
  "capsules": [
    {"id": "tetra://app", "name": "App", "version": "0.1.0", "path": "/tmp/app.capsule", "targets": ["linux-x64"]}
  ]
}`
	out, err := runEcoLockValidator(t, lock)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unsupported permissions model") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoLockRejectsMissingDependency(t *testing.T) {
	lock := `{
  "capsules": [
    {
      "id": "tetra://app",
      "name": "App",
      "version": "0.1.0",
      "path": "/tmp/project/Tetra.capsule",
      "targets": ["linux-x64"],
      "dependencies": [{"id": "tetra://missing", "version": "0.1.0"}]
    }
  ]
}`
	out, err := runEcoLockValidator(t, lock)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unknown dependency tetra://missing") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoLockRejectsDuplicateCapsuleID(t *testing.T) {
	lock := `{
  "capsules": [
    {"id": "tetra://dup", "name": "One", "version": "0.1.0", "path": "/tmp/one.capsule", "targets": ["linux-x64"]},
    {"id": "tetra://dup", "name": "Two", "version": "0.1.0", "path": "/tmp/two.capsule", "targets": ["linux-x64"]}
  ]
}`
	out, err := runEcoLockValidator(t, lock)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "duplicate capsule id tetra://dup") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoLockRejectsNullCapsules(t *testing.T) {
	out, err := runEcoLockValidator(t, `{"capsules":null}`)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "capsules must be an array") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoLockRejectsUnsupportedTarget(t *testing.T) {
	lock := `{
  "capsules": [
    {"id": "tetra://app", "name": "App", "version": "0.1.0", "path": "/tmp/app.capsule", "targets": ["wasm32-wasi"]}
  ]
}`
	out, err := runEcoLockValidator(t, lock)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unsupported target wasm32-wasi") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoLockRejectsDuplicateTarget(t *testing.T) {
	lock := `{
  "capsules": [
    {"id": "tetra://app", "name": "App", "version": "0.1.0", "path": "/tmp/app.capsule", "targets": ["linux-x64", "linux-x64"]}
  ]
}`
	out, err := runEcoLockValidator(t, lock)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "duplicate target linux-x64") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoLockRejectsDuplicateDependency(t *testing.T) {
	lock := `{
  "capsules": [
    {
      "id": "tetra://app",
      "name": "App",
      "version": "0.1.0",
      "path": "/tmp/app.capsule",
      "targets": ["linux-x64"],
      "dependencies": [
        {"id": "tetra://core", "version": "0.1.0"},
        {"id": "tetra://core", "version": "0.1.0"}
      ]
    },
    {"id": "tetra://core", "name": "Core", "version": "0.1.0", "path": "/tmp/core.capsule", "targets": ["linux-x64"]}
  ]
}`
	out, err := runEcoLockValidator(t, lock)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "duplicate dependency tetra://core") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoLockRejectsSelfDependency(t *testing.T) {
	lock := `{
  "capsules": [
    {
      "id": "tetra://app",
      "name": "App",
      "version": "0.1.0",
      "path": "/tmp/app.capsule",
      "targets": ["linux-x64"],
      "dependencies": [{"id": "tetra://app", "version": "0.1.0"}]
    }
  ]
}`
	out, err := runEcoLockValidator(t, lock)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "cannot depend on itself") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoLockRejectsUnknownTopLevelField(t *testing.T) {
	lock := `{
  "capsules": [
    {"id": "tetra://app", "name": "App", "version": "0.1.0", "path": "/tmp/app.capsule", "targets": ["linux-x64"]}
  ],
  "unexpected": true
}`
	out, err := runEcoLockValidator(t, lock)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unknown field") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoLockRejectsDependencyPermissionEscalation(t *testing.T) {
	lock := `{
  "capsules": [
    {
      "id": "tetra://app",
      "name": "App",
      "version": "0.1.0",
      "path": "/tmp/app.capsule",
      "targets": ["linux-x64"],
      "dependencies": [{"id": "tetra://core", "version": "0.1.0"}]
    },
    {
      "id": "tetra://core",
      "name": "Core",
      "version": "0.1.0",
      "path": "/tmp/core.capsule",
      "targets": ["linux-x64"],
      "effects": ["io"]
    }
  ]
}`
	out, err := runEcoLockValidator(t, lock)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "missing required effect io") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func runEcoLockValidator(t *testing.T, lock string) ([]byte, error) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "tetra.lock.json")
	if err := os.WriteFile(path, []byte(lock), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".", "--lock", path)
	cmd.Dir = "."
	return cmd.CombinedOutput()
}
