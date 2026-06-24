package testall

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
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
	cmd := newTestAllCommand(t, root, root, "scripts/ci/test-all.sh", env, args...)
	return cmd.CombinedOutput()
}

func runTestAllSplit(
	t *testing.T,
	root string,
	env []string,
	args ...string,
) ([]byte, []byte, error) {
	t.Helper()
	cmd := newTestAllCommand(t, root, root, "scripts/ci/test-all.sh", env, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

func runTestAllFromWorkingDir(
	t *testing.T,
	root string,
	workingDir string,
	env []string,
	args ...string,
) ([]byte, error) {
	t.Helper()
	script := filepath.Join(root, "scripts", "ci", "test-all.sh")
	cmd := newTestAllCommand(t, root, workingDir, script, env, args...)
	return cmd.CombinedOutput()
}

func newTestAllCommand(
	t *testing.T,
	root string,
	workingDir string,
	script string,
	env []string,
	args ...string,
) *exec.Cmd {
	t.Helper()
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Fatalf("find bash: %v", err)
	}
	cmd := exec.Command(bashPath, append([]string{script}, args...)...)
	cmd.Dir = workingDir
	cmd.Env = testAllHermeticEnv(t, root, env)
	return cmd
}

var testAllAllowedExplicitEnvKeys = map[string]struct{}{
	"TETRA_FAIL_FMT":                           {},
	"TETRA_FAIL_SAFETY_READINESS":              {},
	"TETRA_FAIL_SUMMARY_VALIDATOR":             {},
	"TETRA_FAKE_FORBID_TARGET_HOST_REPORT_ENV": {},
	"TETRA_FAKE_GO_LOG":                        {},
	"TETRA_FAKE_SKIP_BOUNDS_PROOF_LIST":        {},
	"TETRA_FAKE_SKIP_HOST_LEAK_LIST":           {},
	"TETRA_FAKE_SKIP_MEMORY_FUZZ_ORACLE_LIST":  {},
	"TETRA_FAKE_SKIP_RAM_CONTRACT_LIST":        {},
	"TETRA_FAKE_SKIP_UNSAFE_PROMOTION_LIST":    {},
	"TETRA_FAKE_SKIP_WEB_UI_SMOKE_REPORT":      {},
	"TETRA_FAKE_SMOKE_REPORT_FAIL":             {},
	"TETRA_FAKE_TETRA_VERSION":                 {},
	"TETRA_FAKE_ZERO_DOCTOR_REPORT":            {},
	"TETRA_TEST_ALL_RELEASE_ARTIFACT":          {},
	"TETRA_TEST_ALL_RELEASE_VERSION":           {},
	"TETRA_TEST_GO_LOG":                        {},
	"TETRA_TEST_GOFMT_LOG":                     {},
}

var testAllProtectedEnvKeys = map[string]struct{}{
	"BASH_ENV":       {},
	"ENV":            {},
	"GOENV":          {},
	"GOFLAGS":        {},
	"GOCACHE":        {},
	"GOTMPDIR":       {},
	"GOWORK":         {},
	"HOME":           {},
	"PATH":           {},
	"TEMP":           {},
	"TMP":            {},
	"TMPDIR":         {},
	"XDG_CACHE_HOME": {},
}

func testAllHermeticEnv(t *testing.T, repoRoot string, explicit []string) []string {
	t.Helper()
	runRoot := t.TempDir()
	dirs := map[string]string{
		"HOME":           filepath.Join(runRoot, "home"),
		"XDG_CACHE_HOME": filepath.Join(runRoot, "xdg-cache"),
		"GOCACHE":        filepath.Join(runRoot, "go-cache"),
		"GOTMPDIR":       filepath.Join(runRoot, "go-tmp"),
		"TMPDIR":         filepath.Join(runRoot, "tmp"),
		"TMP":            filepath.Join(runRoot, "tmp"),
		"TEMP":           filepath.Join(runRoot, "tmp"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("create hermetic env dir %s: %v", dir, err)
		}
	}

	pathValue := filepath.Join(repoRoot, "bin")
	if hostPath := os.Getenv("PATH"); hostPath != "" {
		pathValue += string(os.PathListSeparator) + hostPath
	}
	env := map[string]string{
		"GOENV":          "off",
		"GOFLAGS":        "",
		"GOTELEMETRY":    "off",
		"GOWORK":         "off",
		"HOME":           dirs["HOME"],
		"LANG":           "C",
		"LC_ALL":         "C",
		"PATH":           pathValue,
		"TEMP":           dirs["TEMP"],
		"TMP":            dirs["TMP"],
		"TMPDIR":         dirs["TMPDIR"],
		"TZ":             "UTC",
		"XDG_CACHE_HOME": dirs["XDG_CACHE_HOME"],
		"GOCACHE":        dirs["GOCACHE"],
		"GOTMPDIR":       dirs["GOTMPDIR"],
	}
	if runtime.GOOS == "windows" {
		for _, key := range []string{"SystemRoot", "ComSpec", "PATHEXT"} {
			if value := os.Getenv(key); value != "" {
				env[key] = value
			}
		}
	}

	seenExplicit := map[string]struct{}{}
	for _, entry := range explicit {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			t.Fatalf("malformed test_all environment entry %q; want KEY=VALUE", entry)
		}
		if _, dup := seenExplicit[key]; dup {
			t.Fatalf("duplicate test_all explicit environment key %q", key)
		}
		seenExplicit[key] = struct{}{}
		if _, protected := testAllProtectedEnvKeys[key]; protected {
			t.Fatalf("test_all explicit environment key %q is protected", key)
		}
		if _, allowed := testAllAllowedExplicitEnvKeys[key]; !allowed {
			t.Fatalf("test_all explicit environment key %q is not in the allowlist", key)
		}
		env[key] = value
	}

	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+env[key])
	}
	return out
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

func copyFile(src, dst string, mode os.FileMode) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, raw, mode)
}

func assertLegacyFileRemoved(t *testing.T, rel, mustUse string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(repoRoot(t), filepath.FromSlash(rel))); err == nil {
		t.Fatalf("%s must be removed; use %s", rel, mustUse)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", rel, err)
	}
}

func assertNoLegacyMention(t *testing.T, text, legacy, where string) {
	t.Helper()
	if strings.Contains(text, legacy) {
		t.Fatalf("%s must not advertise removed root-level wrapper %s", where, legacy)
	}
}

func assertOutputAvoidsRawPathUtilityErrors(t *testing.T, out []byte) {
	t.Helper()
	for _, forbidden := range []string{
		"unbound variable",
		"mkdir:",
		"dirname:",
		"find:",
		"cp:",
		"cat:",
		"sed:",
	} {
		if strings.Contains(string(out), forbidden) {
			t.Fatalf(
				"output should use controlled path hygiene errors, not raw shell utility failures:\n%s",
				out,
			)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
