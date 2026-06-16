package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"tetra_language/internal/toon"
)

func TestValidateEcoSeedAcceptsValidReport(t *testing.T) {
	out, err := runEcoSeedValidator(t, validSeedReport())
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateEcoSeedAcceptsTOON(t *testing.T) {
	toonRaw, err := toon.ConvertJSONToTOON([]byte(validSeedReport()), toon.Options{Strict: true, Deterministic: true})
	if err != nil {
		t.Fatalf("json->toon: %v", err)
	}
	if err := validateEcoSeedFormat(toonRaw, "toon"); err != nil {
		t.Fatalf("validateEcoSeedFormat TOON: %v\n%s", err, toonRaw)
	}
}

func TestValidateEcoSeedRejectsMalformedJSON(t *testing.T) {
	out, err := runEcoSeedValidator(t, `{"schema": "tetra.eco.seed.v1",`)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unexpected EOF") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoSeedRejectsUnknownField(t *testing.T) {
	seed := strings.Replace(validSeedReport(), "\n  \"capsules\":", "\n  \"unexpected\": true,\n  \"capsules\":", 1)
	out, err := runEcoSeedValidator(t, seed)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unknown field") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoSeedRejectsUnknownCapsuleField(t *testing.T) {
	seed := strings.Replace(validSeedReport(), `"permissions": ["io"]`, `"permissions": ["io"], "extra": true`, 1)
	out, err := runEcoSeedValidator(t, seed)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unknown field") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoSeedRejectsMissingRequiredTopLevelField(t *testing.T) {
	seed := strings.Replace(validSeedReport(), "  \"generated_at_unix\": 0,\n", "", 1)
	out, err := runEcoSeedValidator(t, seed)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "generated_at_unix is required") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoSeedRejectsSeedLockMismatch(t *testing.T) {
	seed := strings.Replace(validSeedReport(), `"version": "0.1.0",`, `"version": "0.2.0",`, 1)
	out, err := runEcoSeedValidator(t, seed)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "version mismatch") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoSeedRejectsDuplicateDependencyIDWithDifferentVersionAndPath(t *testing.T) {
	seed := strings.Replace(validSeedReport(),
		`"depends_on": [{"id": "tetra://core", "version": "0.1.0"}]`,
		`"depends_on": [{"id": "tetra://core", "version": "0.1.0"}, {"id": "tetra://core", "version": "0.2.0", "path": "alt/Core.t4"}]`,
		1,
	)
	out, err := runEcoSeedValidator(t, seed)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "duplicate dependency tetra://core") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoSeedRejectsDuplicateLockDependencyIDWithDifferentVersionAndPath(t *testing.T) {
	seed := strings.Replace(validSeedReport(),
		`"dependencies": [{"id": "tetra://core", "version": "0.1.0"}]`,
		`"dependencies": [{"id": "tetra://core", "version": "0.1.0"}, {"id": "tetra://core", "version": "0.2.0", "path": "alt/Core.t4"}]`,
		1,
	)
	out, err := runEcoSeedValidator(t, seed)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "duplicate dependency tetra://core") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateEcoSeedAcceptsPortableDependencyPath(t *testing.T) {
	seed := strings.Replace(validSeedReport(),
		`"dependencies": [{"id": "tetra://core", "version": "0.1.0"}]`,
		`"dependencies": [{"id": "tetra://core", "version": "0.1.0", "path": "deps/Core.t4"}]`,
		1,
	)
	seed = strings.Replace(seed,
		`"depends_on": [{"id": "tetra://core", "version": "0.1.0"}]`,
		`"depends_on": [{"id": "tetra://core", "version": "0.1.0", "path": "deps/Core.t4"}]`,
		1,
	)
	out, err := runEcoSeedValidator(t, seed)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateEcoSeedRejectsInvalidLockCapsulePaths(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		message string
	}{
		{name: "absolute", path: "/tmp/Capsule.t4", message: "path must be relative"},
		{name: "windows absolute", path: "C:/tmp/Capsule.t4", message: "path must be relative"},
		{name: "traversal", path: "capsules/../Capsule.t4", message: "path must not contain .."},
		{name: "empty normalization", path: ".", message: "path must not normalize to empty"},
		{name: "invalid normalization", path: "capsules//Capsule.t4", message: "path must already be normalized"},
		{name: "backslash", path: `capsules\Capsule.t4`, message: "path must use forward slashes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seed := strings.Replace(validSeedReport(), `"path": "Capsule.t4"`, `"path": `+strconv.Quote(tt.path), 1)
			out, err := runEcoSeedValidator(t, seed)
			if err == nil {
				t.Fatalf("expected validator failure\n%s", out)
			}
			if !strings.Contains(string(out), tt.message) {
				t.Fatalf("unexpected output:\n%s", out)
			}
		})
	}
}

func TestValidateEcoSeedRejectsInvalidDependencyPaths(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		message string
	}{
		{name: "absolute", path: "/tmp/Core.t4", message: "path must be relative"},
		{name: "traversal", path: "deps/../Core.t4", message: "path must not contain .."},
		{name: "empty", path: "", message: "path must not be empty"},
		{name: "invalid normalization", path: "./Core.t4", message: "path must already be normalized"},
		{name: "backslash", path: `deps\Core.t4`, message: "path must use forward slashes"},
	}
	for _, tt := range tests {
		t.Run("lock "+tt.name, func(t *testing.T) {
			seed := strings.Replace(validSeedReport(),
				`"dependencies": [{"id": "tetra://core", "version": "0.1.0"}]`,
				`"dependencies": [{"id": "tetra://core", "version": "0.1.0", "path": `+strconv.Quote(tt.path)+`}]`,
				1,
			)
			out, err := runEcoSeedValidator(t, seed)
			if err == nil {
				t.Fatalf("expected validator failure\n%s", out)
			}
			if !strings.Contains(string(out), tt.message) {
				t.Fatalf("unexpected output:\n%s", out)
			}
		})
		t.Run("seed "+tt.name, func(t *testing.T) {
			seed := strings.Replace(validSeedReport(),
				`"depends_on": [{"id": "tetra://core", "version": "0.1.0"}]`,
				`"depends_on": [{"id": "tetra://core", "version": "0.1.0", "path": `+strconv.Quote(tt.path)+`}]`,
				1,
			)
			out, err := runEcoSeedValidator(t, seed)
			if err == nil {
				t.Fatalf("expected validator failure\n%s", out)
			}
			if !strings.Contains(string(out), tt.message) {
				t.Fatalf("unexpected output:\n%s", out)
			}
		})
	}
}

func TestValidateEcoSeedRejectsInvalidArtifactPaths(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		message string
	}{
		{name: "absolute", path: "/tmp/App.t4i", message: "path must be relative"},
		{name: "windows absolute slash", path: "C:/tmp/App.t4i", message: "path must be relative"},
		{name: "windows absolute backslash", path: `C:\tmp\App.t4i`, message: "path must use forward slashes"},
		{name: "traversal", path: "artifacts/../App.t4i", message: "path must not contain .."},
		{name: "empty normalization", path: ".", message: "path must not normalize to empty"},
		{name: "invalid normalization", path: "artifacts//App.t4i", message: "path must already be normalized"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seed := seedReportWithLockArtifact(tt.path)
			out, err := runEcoSeedValidator(t, seed)
			if err == nil {
				t.Fatalf("expected validator failure\n%s", out)
			}
			if !strings.Contains(string(out), tt.message) {
				t.Fatalf("unexpected output:\n%s", out)
			}
		})
	}
}

func validSeedReport() string {
	return `{
  "schema": "tetra.eco.seed.v1",
  "generated_at_unix": 0,
  "lock": {
    "schema": "tetra.eco.lock.v1",
    "manifest_schema": "tetra.capsule.v1",
    "permissions_model": "tetra.eco.permissions.v1",
    "generated_at_unix": 0,
    "capsules": [
      {
        "id": "tetra://app",
        "name": "App",
        "version": "0.1.0",
        "path": "Capsule.t4",
        "targets": ["linux-x64"],
        "permissions": ["io"],
        "dependencies": [{"id": "tetra://core", "version": "0.1.0"}]
      },
      {
        "id": "tetra://core",
        "name": "Core",
        "version": "0.1.0",
        "path": "Core.t4",
        "targets": ["linux-x64"],
        "permissions": ["io"]
      }
    ]
  },
  "capsules": [
    {
      "id": "tetra://app",
      "name": "App",
      "version": "0.1.0",
      "targets": ["linux-x64"],
      "permissions": ["io"],
      "depends_on": [{"id": "tetra://core", "version": "0.1.0"}]
    },
    {
      "id": "tetra://core",
      "name": "Core",
      "version": "0.1.0",
      "targets": ["linux-x64"],
      "permissions": ["io"]
    }
  ]
}`
}

func seedReportWithLockArtifact(path string) string {
	return strings.Replace(validSeedReport(),
		`"dependencies": [{"id": "tetra://core", "version": "0.1.0"}]`,
		`"dependencies": [{"id": "tetra://core", "version": "0.1.0"}],
        "artifacts": [{"kind": "interface", "path": `+strconv.Quote(path)+`}]`,
		1,
	)
}

func runEcoSeedValidator(t *testing.T, seed string) ([]byte, error) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "tetra.seed.json")
	if err := os.WriteFile(path, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".", "--seed", path)
	cmd.Dir = "."
	return cmd.CombinedOutput()
}
