package structure

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDirectoryBudgetCommandBaselineAndStrictMode(t *testing.T) {
	root := repoRoot(t)
	validatorDir := filepath.Join(root, "tools", "cmd", "validate", "directory-budget")
	fixtureRepo := t.TempDir()
	baselinePath := filepath.Join(fixtureRepo, "directory-budget-baseline.json")

	for i := 0; i < 7; i++ {
		writeFixtureFile(t, fixtureRepo, filepath.Join("examples", fmt.Sprintf("example_%d.tetra", i)))
	}

	output, err := runDirectoryBudget(t, root, fixtureRepo, validatorDir, "--roots", "examples", "--write-baseline", baselinePath)
	if err != nil {
		t.Fatalf("write baseline failed: %v\n%s", err, output)
	}
	if _, err := os.Stat(baselinePath); err != nil {
		t.Fatalf("baseline was not written: %v\n%s", err, output)
	}

	output, err = runDirectoryBudget(t, root, fixtureRepo, validatorDir, "--roots", "examples", "--baseline", baselinePath)
	if err != nil {
		t.Fatalf("baseline run failed: %v\n%s", err, output)
	}

	output, err = runDirectoryBudget(t, root, fixtureRepo, validatorDir, "--roots", "examples", "--baseline", baselinePath, "--strict")
	if err == nil {
		t.Fatalf("strict run unexpectedly passed:\n%s", output)
	}
	if !strings.Contains(output, "examples: 7 active files") {
		t.Fatalf("strict run output missing violation summary:\n%s", output)
	}
}

func runDirectoryBudget(t *testing.T, root, fixtureRepo, validatorDir string, args ...string) (string, error) {
	t.Helper()

	cmdArgs := append([]string{"run", "-buildvcs=false", validatorDir}, args...)
	cmd := exec.Command("go", cmdArgs...)
	cmd.Dir = fixtureRepo
	cmd.Env = childGoEnv(t, root)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func childGoEnv(t *testing.T, root string) []string {
	t.Helper()

	env := setEnv(os.Environ(), "GOTELEMETRY", "off")
	workPath := filepath.Join(root, "go.work")
	if _, err := os.Stat(workPath); err == nil {
		env = setEnv(env, "GOWORK", workPath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat go.work: %v", err)
	}
	return env
}

func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("repo root %s missing go.mod: %v", root, err)
	}
	return root
}

func writeFixtureFile(t *testing.T, root, rel string) {
	t.Helper()

	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("fn main() {}\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}
