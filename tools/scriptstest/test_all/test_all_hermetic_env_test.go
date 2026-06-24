package testall

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
)

const (
	testAllUnsafePromotionStep = "unsafe promotion blocker suite"
	testAllBoundsProofStep     = "bounds proof blocker suite"
	testAllMemoryFuzzStep      = "memory fuzz oracle artifact gate"
	testAllRAMContractStep     = "RAM contract fuzz oracle artifact gate"
	testAllHostLeakStep        = "host leak blocker suite"
)

func TestTestAllHermeticEnvRejectsAmbientControlMatrix(t *testing.T) {
	t.Run("ambient controls", func(t *testing.T) {
		for key, value := range map[string]string{
			"TETRA_FAKE_SKIP_UNSAFE_PROMOTION_LIST":   "1",
			"TETRA_FAKE_SKIP_BOUNDS_PROOF_LIST":       "1",
			"TETRA_FAKE_SKIP_MEMORY_FUZZ_ORACLE_LIST": "1",
			"TETRA_FAKE_SKIP_HOST_LEAK_LIST":          "1",
			"TETRA_FAKE_SKIP_RAM_CONTRACT_LIST":       "1",
			"TETRA_FAKE_SKIP_WEB_UI_SMOKE_REPORT":     "1",
			"TETRA_FAKE_SMOKE_REPORT_FAIL":            "1",
			"TETRA_FAKE_ZERO_DOCTOR_REPORT":           "1",
			"TETRA_FAIL_FMT":                          "1",
			"TETRA_FAIL_SUMMARY_VALIDATOR":            "1",
			"TETRA_FAIL_SAFETY_READINESS":             "1",
			"TETRA_WINDOWS_UI_RUNTIME_REPORT":         filepath.Join(t.TempDir(), "windows.json"),
			"TETRA_MACOS_UI_RUNTIME_REPORT":           filepath.Join(t.TempDir(), "macos.json"),
			"TETRA_SECURITY_REVIEW_SIGNOFF":           "ambient-signoff",
			"TETRA_TEST_ALL_RELEASE_VERSION":          "v9.9.9",
			"TETRA_TEST_ALL_RELEASE_ARTIFACT":         "ambient.artifact",
			"CI":                                      "true",
			"GITHUB_ACTIONS":                          "true",
			"GITHUB_EVENT_NAME":                       "pull_request",
			"GITHUB_SHA":                              "ffffffffffffffffffffffffffffffffffffffff",
			"GITHUB_REF":                              "refs/pull/6/merge",
			"GITHUB_HEAD_REF":                         "stabilize/memory-core-v2",
			"GITHUB_BASE_REF":                         "main",
			"RUNNER_TEMP":                             t.TempDir(),
			"GOFLAGS":                                 "-run=NoSuchTestFromAmbient",
			"GOWORK":                                  filepath.Join(t.TempDir(), "go.work"),
			"GOENV":                                   filepath.Join(t.TempDir(), "goenv"),
			"GIT_CONFIG_COUNT":                        "1",
			"GIT_CONFIG_KEY_0":                        "alias.test-all-hermetic",
			"GIT_CONFIG_VALUE_0":                      "!false",
		} {
			t.Setenv(key, value)
		}

		root := testAllFakeRepo(t, false)
		assertTestAllNormalRunPasses(t, root, "--quick")
		assertTestAllNormalRunPasses(t, root, "--full")
	})

	t.Run("bash env injection", func(t *testing.T) {
		hook := filepath.Join(t.TempDir(), "bashenv.sh")
		if err := os.WriteFile(
			hook,
			[]byte(strings.Join([]string{
				"export TETRA_FAKE_SKIP_BOUNDS_PROOF_LIST=1",
				"export TETRA_FAKE_SKIP_HOST_LEAK_LIST=1",
				"",
			}, "\n")),
			0o644,
		); err != nil {
			t.Fatalf("write BASH_ENV hook: %v", err)
		}
		t.Setenv("BASH_ENV", hook)
		t.Setenv("ENV", hook)

		root := testAllFakeRepo(t, false)
		assertTestAllNormalRunPasses(t, root, "--quick")
	})
}

func TestTestAllExplicitFailureControlsAreIsolated(t *testing.T) {
	tests := []struct {
		name string
		env  string
		step string
	}{
		{
			name: "host leak",
			env:  "TETRA_FAKE_SKIP_HOST_LEAK_LIST=1",
			step: testAllHostLeakStep,
		},
		{
			name: "bounds proof",
			env:  "TETRA_FAKE_SKIP_BOUNDS_PROOF_LIST=1",
			step: testAllBoundsProofStep,
		},
		{
			name: "unsafe promotion",
			env:  "TETRA_FAKE_SKIP_UNSAFE_PROMOTION_LIST=1",
			step: testAllUnsafePromotionStep,
		},
		{
			name: "memory fuzz",
			env:  "TETRA_FAKE_SKIP_MEMORY_FUZZ_ORACLE_LIST=1",
			step: testAllMemoryFuzzStep,
		},
		{
			name: "RAM contract",
			env:  "TETRA_FAKE_SKIP_RAM_CONTRACT_LIST=1",
			step: testAllRAMContractStep,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := testAllFakeRepo(t, false)
			reportDir := filepath.Join(root, "report")
			out, err := runTestAll(
				t,
				root,
				[]string{tt.env},
				"--quick",
				"--keep-going",
				"--json-only",
				"--report-dir",
				reportDir,
			)
			if err == nil {
				t.Fatalf("expected explicit control failure for %s\n%s", tt.step, out)
			}
			summary := decodeTestAllSummary(t, out)
			assertOnlyTestAllFailedStep(t, summary, tt.step)
			for _, blocker := range []string{
				testAllUnsafePromotionStep,
				testAllBoundsProofStep,
				testAllMemoryFuzzStep,
				testAllRAMContractStep,
				testAllHostLeakStep,
			} {
				if blocker == tt.step {
					continue
				}
				assertTestAllStepStatus(t, summary, blocker, "pass")
			}
		})
	}
}

func TestTestAllFakeRepoIgnoresAmbientTargetHostReports(t *testing.T) {
	windowsReport := filepath.Join(t.TempDir(), "windows-ui-runtime.json")
	macosReport := filepath.Join(t.TempDir(), "macos-ui-runtime.json")
	if err := os.WriteFile(windowsReport, []byte(`{"status":"poisoned"}`), 0o644); err != nil {
		t.Fatalf("write poisoned windows report: %v", err)
	}
	if err := os.WriteFile(macosReport, []byte(`{"status":"poisoned"}`), 0o644); err != nil {
		t.Fatalf("write poisoned macos report: %v", err)
	}
	t.Setenv("TETRA_WINDOWS_UI_RUNTIME_REPORT", windowsReport)
	t.Setenv("TETRA_MACOS_UI_RUNTIME_REPORT", macosReport)

	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(
		t,
		root,
		[]string{"TETRA_FAKE_FORBID_TARGET_HOST_REPORT_ENV=1"},
		"--full",
		"--json-only",
		"--report-dir",
		reportDir,
	)
	if err != nil {
		t.Fatalf(
			"test_all full should ignore ambient target-host reports: %v\n%s\n%s",
			err,
			out,
			collectUnexpectedTestAllFailure(t, root, reportDir, out),
		)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "pass" || summary.FailedCount != 0 {
		t.Fatalf("summary status/counts = %q/%d, want pass/0: %#v",
			summary.Status,
			summary.FailedCount,
			summary.Steps,
		)
	}
}

func TestTestAllHermeticRunsDoNotCrossContaminate(t *testing.T) {
	hostRoot := testAllFakeRepo(t, false)
	boundsRoot := testAllFakeRepo(t, false)
	hostReport := filepath.Join(hostRoot, "report-host")
	boundsReport := filepath.Join(boundsRoot, "report-bounds")
	hostLog := filepath.Join(hostRoot, "fake-go-host.log")
	boundsLog := filepath.Join(boundsRoot, "fake-go-bounds.log")

	hostCmd := newTestAllCommand(
		t,
		hostRoot,
		hostRoot,
		"scripts/ci/test-all.sh",
		[]string{
			"TETRA_FAKE_SKIP_HOST_LEAK_LIST=1",
			"TETRA_FAKE_GO_LOG=" + hostLog,
		},
		"--quick",
		"--keep-going",
		"--json-only",
		"--report-dir",
		hostReport,
	)
	boundsCmd := newTestAllCommand(
		t,
		boundsRoot,
		boundsRoot,
		"scripts/ci/test-all.sh",
		[]string{
			"TETRA_FAKE_SKIP_BOUNDS_PROOF_LIST=1",
			"TETRA_FAKE_GO_LOG=" + boundsLog,
		},
		"--quick",
		"--keep-going",
		"--json-only",
		"--report-dir",
		boundsReport,
	)

	assertDistinctEnvValue(t, hostCmd.Env, boundsCmd.Env, "HOME")
	assertDistinctEnvValue(t, hostCmd.Env, boundsCmd.Env, "GOCACHE")
	assertDistinctEnvValue(t, hostCmd.Env, boundsCmd.Env, "GOTMPDIR")
	assertDistinctEnvValue(t, hostCmd.Env, boundsCmd.Env, "TMPDIR")

	type result struct {
		name string
		out  []byte
		err  error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for name, cmd := range map[string]*exec.Cmd{
		"host":   hostCmd,
		"bounds": boundsCmd,
	} {
		wg.Add(1)
		go func(name string, cmd *exec.Cmd) {
			defer wg.Done()
			out, err := cmd.CombinedOutput()
			results <- result{name: name, out: out, err: err}
		}(name, cmd)
	}
	wg.Wait()
	close(results)

	seen := map[string]result{}
	for res := range results {
		seen[res.name] = res
	}
	hostRes := seen["host"]
	boundsRes := seen["bounds"]
	if hostRes.err == nil {
		t.Fatalf("expected host-leak controlled failure\n%s", hostRes.out)
	}
	if boundsRes.err == nil {
		t.Fatalf("expected bounds-proof controlled failure\n%s", boundsRes.out)
	}
	assertOnlyTestAllFailedStep(t, decodeTestAllSummary(t, hostRes.out), testAllHostLeakStep)
	assertOnlyTestAllFailedStep(t, decodeTestAllSummary(t, boundsRes.out), testAllBoundsProofStep)

	hostLogRaw, err := os.ReadFile(hostLog)
	if err != nil {
		t.Fatalf("read host fake-go log: %v", err)
	}
	boundsLogRaw, err := os.ReadFile(boundsLog)
	if err != nil {
		t.Fatalf("read bounds fake-go log: %v", err)
	}
	if bytes.Contains(hostLogRaw, []byte(boundsLog)) ||
		bytes.Contains(boundsLogRaw, []byte(hostLog)) ||
		bytes.Contains(hostRes.out, []byte(boundsReport)) ||
		bytes.Contains(boundsRes.out, []byte(hostReport)) {
		t.Fatalf(
			"concurrent hermetic runs cross-contaminated paths\nhost out:\n%s\nbounds out:\n%s",
			hostRes.out,
			boundsRes.out,
		)
	}
}

func TestTestAllHermeticEnvHasUniqueDeterministicKeys(t *testing.T) {
	t.Setenv("TETRA_FAKE_SKIP_HOST_LEAK_LIST", "1")
	t.Setenv("BASH_ENV", filepath.Join(t.TempDir(), "ambient-bashenv.sh"))
	t.Setenv("ENV", filepath.Join(t.TempDir(), "ambient-env.sh"))

	root := testAllFakeRepo(t, false)
	env1 := testAllHermeticEnv(
		t,
		root,
		[]string{"TETRA_FAKE_SKIP_BOUNDS_PROOF_LIST=1"},
	)
	env2 := testAllHermeticEnv(
		t,
		root,
		[]string{"TETRA_FAKE_SKIP_BOUNDS_PROOF_LIST=1"},
	)

	keys1 := testAllEnvKeys(t, env1)
	keys2 := testAllEnvKeys(t, env2)
	if !sort.StringsAreSorted(keys1) {
		t.Fatalf("env keys are not sorted: %v", keys1)
	}
	if !reflect.DeepEqual(keys1, keys2) {
		t.Fatalf("same inputs produced different key ordering\n%v\n%v", keys1, keys2)
	}

	envMap := testAllEnvMap(t, env1)
	for key, want := range map[string]string{
		"GOENV":       "off",
		"GOWORK":      "off",
		"GOFLAGS":     "",
		"GOTELEMETRY": "off",
		"LANG":        "C",
		"LC_ALL":      "C",
		"TZ":          "UTC",
	} {
		if got := envMap[key]; got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
	for _, key := range []string{"HOME", "XDG_CACHE_HOME", "GOCACHE", "GOTMPDIR", "TMPDIR", "TMP", "TEMP"} {
		if got := envMap[key]; got == "" {
			t.Fatalf("%s missing from hermetic env", key)
		}
	}
	if path := envMap["PATH"]; !strings.HasPrefix(path, filepath.Join(root, "bin")+string(os.PathListSeparator)) {
		t.Fatalf("PATH = %q, want fake repo bin first", path)
	}
	for _, absent := range []string{
		"TETRA_FAKE_SKIP_HOST_LEAK_LIST",
		"BASH_ENV",
		"ENV",
		"CI",
		"GITHUB_ACTIONS",
		"TETRA_WINDOWS_UI_RUNTIME_REPORT",
		"TETRA_MACOS_UI_RUNTIME_REPORT",
	} {
		if _, ok := envMap[absent]; ok {
			t.Fatalf("ambient key %s leaked into hermetic env", absent)
		}
	}
	if got := envMap["TETRA_FAKE_SKIP_BOUNDS_PROOF_LIST"]; got != "1" {
		t.Fatalf("explicit TETRA control = %q, want 1", got)
	}
}

func TestTestAllUnexpectedFailureEvidenceIncludesBlockerLog(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(
		t,
		root,
		[]string{"TETRA_FAKE_SKIP_UNSAFE_PROMOTION_LIST=1"},
		"--quick",
		"--json-only",
		"--report-dir",
		reportDir,
	)
	if err == nil {
		t.Fatalf("expected unsafe blocker failure\n%s", out)
	}
	evidence := collectUnexpectedTestAllFailure(t, root, reportDir, out)
	for _, want := range []string{
		"unsafe promotion blocker suite",
		"missing required unsafe promotion blocker test: ./compiler/internal/memoryfacts TestMemoryFactsRejectsUnsafeUnknownToSafeKnown",
		"package=./compiler/internal/memoryfacts",
		"list_pattern=UnsafeUnknown|UnsafeVerified|Promotion",
		"skip_unsafe_present=1",
		"list_result=skipped_by_explicit_control",
	} {
		if !strings.Contains(evidence, want) {
			t.Fatalf("failure evidence missing %q:\n%s", want, evidence)
		}
	}
}

func TestTestAllNormalFakeGoTraceShowsNoSkipControls(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(t, root, nil, "--quick", "--json-only", "--report-dir", reportDir)
	if err != nil {
		t.Fatalf("test_all quick failed: %v\n%s", err, out)
	}
	evidence := collectUnexpectedTestAllFailure(t, root, reportDir, out)
	for _, want := range []string{
		"skip_unsafe_present=0",
		"skip_bounds_present=0",
		"skip_host_leak_present=0",
		"skip_memory_fuzz_present=0",
		"skip_ram_contract_present=0",
		"package=./compiler/internal/memoryfacts",
		"list_pattern=UnsafeUnknown|UnsafeVerified|Promotion",
		"package=./compiler/internal/validation",
		"list_pattern=Bounds|Proof|Unchecked",
		"list_result=normal",
	} {
		if !strings.Contains(evidence, want) {
			t.Fatalf("normal fake-go trace missing %q:\n%s", want, evidence)
		}
	}
	for _, sequence := range [][]string{
		{
			"package=./compiler/internal/memoryfacts",
			"list_pattern=UnsafeUnknown|UnsafeVerified|Promotion",
			"list_result=normal",
			"emitted_line_count=15",
		},
		{
			"package=./compiler/internal/validation",
			"list_pattern=Bounds|Proof|Unchecked",
			"list_result=normal",
			"emitted_line_count=3",
		},
	} {
		if !containsAllInOrder(evidence, sequence...) {
			t.Fatalf("normal list trace missing ordered fields %v:\n%s", sequence, evidence)
		}
	}
}

func TestTestAllFailureEvidenceRejectsEscapingLogPath(t *testing.T) {
	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatalf("create report dir: %v", err)
	}
	secret := filepath.Join(root, "secret")
	if err := os.WriteFile(secret, []byte("do not read\n"), 0o644); err != nil {
		t.Fatalf("write secret: %v", err)
	}
	rawSummary := []byte(`{
		"status":"fail",
		"failed_count":1,
		"steps":[
			{"name":"escaping step","status":"fail","exit_code":1,"command":"fake","log":"../../secret"}
		]
	}`)
	evidence := collectUnexpectedTestAllFailure(t, root, reportDir, rawSummary)
	if !strings.Contains(evidence, "rejected unsafe log path") {
		t.Fatalf("failure evidence did not reject escaping path:\n%s", evidence)
	}
	if strings.Contains(evidence, "do not read") {
		t.Fatalf("failure evidence read file outside report dir:\n%s", evidence)
	}
}

func TestTestAllFailureEvidenceDoesNotExposeAmbientSecret(t *testing.T) {
	sentinel := "sentinel-secret-value-for-test-all-observability"
	t.Setenv("TETRA_TEST_SECRET_SENTINEL", sentinel)

	root := testAllFakeRepo(t, false)
	reportDir := filepath.Join(root, "report")
	out, err := runTestAll(
		t,
		root,
		[]string{"TETRA_FAKE_SKIP_UNSAFE_PROMOTION_LIST=1"},
		"--quick",
		"--json-only",
		"--report-dir",
		reportDir,
	)
	if err == nil {
		t.Fatalf("expected unsafe blocker failure\n%s", out)
	}
	evidence := collectUnexpectedTestAllFailure(t, root, reportDir, out)
	if strings.Contains(evidence, sentinel) {
		t.Fatalf("failure evidence exposed ambient secret:\n%s", evidence)
	}
}

func assertTestAllNormalRunPasses(t *testing.T, root string, mode string) {
	t.Helper()
	reportDir := filepath.Join(root, "report-"+strings.TrimPrefix(mode, "--"))
	out, err := runTestAll(t, root, nil, mode, "--json-only", "--report-dir", reportDir)
	if err != nil {
		t.Fatalf("test_all %s should pass with ambient controls ignored: %v\n%s", mode, err, out)
	}
	summary := decodeTestAllSummary(t, out)
	if summary.Status != "pass" || summary.FailedCount != 0 {
		t.Fatalf("summary status/counts = %q/%d, want pass/0: %#v",
			summary.Status,
			summary.FailedCount,
			summary.Steps,
		)
	}
}

func assertOnlyTestAllFailedStep(t *testing.T, summary testAllSummary, wantStep string) {
	t.Helper()
	if summary.Status != "fail" || summary.FailedCount != 1 {
		t.Fatalf("summary status/counts = %q/%d, want fail/1: %#v",
			summary.Status,
			summary.FailedCount,
			summary.Steps,
		)
	}
	var failed []string
	for _, step := range summary.Steps {
		if step.Status == "fail" {
			failed = append(failed, step.Name)
		}
	}
	if !reflect.DeepEqual(failed, []string{wantStep}) {
		t.Fatalf("failed steps = %v, want [%s]; summary: %#v", failed, wantStep, summary.Steps)
	}
}

func assertTestAllStepStatus(t *testing.T, summary testAllSummary, stepName, wantStatus string) {
	t.Helper()
	for _, step := range summary.Steps {
		if step.Name == stepName {
			if step.Status != wantStatus {
				t.Fatalf("step %q status = %q, want %q", stepName, step.Status, wantStatus)
			}
			return
		}
	}
	t.Fatalf("summary missing step %q: %#v", stepName, summary.Steps)
}

func assertDistinctEnvValue(t *testing.T, envA, envB []string, key string) {
	t.Helper()
	a := testAllEnvMap(t, envA)[key]
	b := testAllEnvMap(t, envB)[key]
	if a == "" || b == "" || a == b {
		t.Fatalf("expected distinct %s values, got %q and %q", key, a, b)
	}
}

func testAllEnvKeys(t *testing.T, env []string) []string {
	t.Helper()
	keys := make([]string, 0, len(env))
	seen := map[string]struct{}{}
	for _, entry := range env {
		key, _, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			t.Fatalf("malformed environment entry %q", entry)
		}
		if _, dup := seen[key]; dup {
			t.Fatalf("duplicate environment key %q in %v", key, env)
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	return keys
}

func testAllEnvMap(t *testing.T, env []string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || key == "" {
			t.Fatalf("malformed environment entry %q", entry)
		}
		if _, dup := out[key]; dup {
			t.Fatalf("duplicate environment key %q in %v", key, env)
		}
		out[key] = value
	}
	return out
}

func containsAllInOrder(s string, needles ...string) bool {
	offset := 0
	for _, needle := range needles {
		next := strings.Index(s[offset:], needle)
		if next < 0 {
			return false
		}
		offset += next + len(needle)
	}
	return true
}
