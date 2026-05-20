package scriptstest

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type testAllSummary struct {
	Status          string `json:"status"`
	StepCount       int    `json:"step_count"`
	FailedCount     int    `json:"failed_count"`
	ReleaseVersion  string `json:"release_version"`
	ReleaseArtifact string `json:"release_artifact"`
	Steps           []struct {
		Name     string `json:"name"`
		Status   string `json:"status"`
		ExitCode *int   `json:"exit_code"`
		Command  string `json:"command"`
		Log      string `json:"log"`
	} `json:"steps"`
}

const testAllFormatterStepName = "formatter check examples lib runtime"

func hasTestAllStep(summary testAllSummary, name string) bool {
	for _, step := range summary.Steps {
		if step.Name == name && step.Status == "pass" {
			return true
		}
	}
	return false
}

func testAllStepLog(t *testing.T, summary testAllSummary, name string) string {
	t.Helper()
	for _, step := range summary.Steps {
		if step.Name == name {
			if step.Log == "" {
				t.Fatalf("step %q missing log path: %#v", name, step)
			}
			return step.Log
		}
	}
	t.Fatalf("summary missing step %q: %#v", name, summary.Steps)
	return ""
}

func readTestAllScript(t *testing.T) ([]byte, error) {
	t.Helper()
	return os.ReadFile(filepath.Join(repoRoot(t), "scripts", "ci", "test-all.sh"))
}

func readReleaseV06GateScript(t *testing.T) ([]byte, error) {
	t.Helper()
	return os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release", "v0_6", "gate.sh"))
}

func runTestAll(t *testing.T, root string, env []string, args ...string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command("bash", append([]string{"scripts/ci/test-all.sh"}, args...)...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), env...)
	cmd.Env = append(cmd.Env, "PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
	return cmd.CombinedOutput()
}

func runTestAllSplit(t *testing.T, root string, env []string, args ...string) ([]byte, []byte, error) {
	t.Helper()
	cmd := exec.Command("bash", append([]string{"scripts/ci/test-all.sh"}, args...)...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), env...)
	cmd.Env = append(cmd.Env, "PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func runTestAllFromWorkingDir(t *testing.T, root string, workingDir string, env []string, args ...string) ([]byte, error) {
	t.Helper()
	script := filepath.Join(root, "scripts", "ci", "test-all.sh")
	cmd := exec.Command("bash", append([]string{script}, args...)...)
	cmd.Dir = workingDir
	cmd.Env = append(os.Environ(), env...)
	cmd.Env = append(cmd.Env, "PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
	return cmd.CombinedOutput()
}

func assertExitCode(t *testing.T, err error, want int, output string) {
	t.Helper()
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != want {
		t.Fatalf("expected exit %d, got %v\n%s", want, err, output)
	}
}

func decodeTestAllSummary(t *testing.T, raw []byte) testAllSummary {
	t.Helper()
	var summary testAllSummary
	if err := json.Unmarshal(raw, &summary); err != nil {
		t.Fatalf("decode summary: %v\n%s", err, string(raw))
	}
	return summary
}
