package testall

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
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

const testAllDiagnosticLogLimit = 64 * 1024

const testAllDiagnosticTraceLimit = 16 * 1024

const testAllDiagnosticBundleLimit = 256 * 1024

const testAllDiagnosticStderrLimit = 32 * 1024

const testAllDiagnosticMaxTraces = 64

const testAllDiagnosticManifestMaxFiles = 256

const testAllDiagnosticManifestMaxPathLength = 512

var testAllMemoryFuzzExpectedArtifacts = []string{
	"memory-fuzz-tier1/memory-fuzz-oracle.json",
	"memory-fuzz-tier1/summary.md",
	"memory-fuzz-tier1/summary.json",
}

var testAllInvocationCounter atomic.Uint64

type testAllRunObservation struct {
	Root       string
	WorkingDir string
	Mode       string
	ReportDir  string
	Args       []string
	Stdout     []byte
	Stderr     []byte
	Combined   []byte
	Err        error
	Summary    *testAllSummary
	Evidence   string
}

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
	obs := executeObservedTestAll(
		t,
		root,
		root,
		"scripts/ci/test-all.sh",
		env,
		false,
		args...,
	)
	return obs.Combined, obs.Err
}

func runTestAllSplit(
	t *testing.T,
	root string,
	env []string,
	args ...string,
) ([]byte, []byte, error) {
	t.Helper()
	obs := executeObservedTestAll(
		t,
		root,
		root,
		"scripts/ci/test-all.sh",
		env,
		true,
		args...,
	)
	return obs.Stdout, obs.Stderr, obs.Err
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
	obs := executeObservedTestAll(t, root, workingDir, script, env, false, args...)
	return obs.Combined, obs.Err
}

func executeObservedTestAll(
	t *testing.T,
	root string,
	workingDir string,
	script string,
	env []string,
	split bool,
	args ...string,
) testAllRunObservation {
	t.Helper()
	cmdEnv := testAllHermeticEnv(t, root, env)
	return executeObservedTestAllWithEnv(t, root, workingDir, script, cmdEnv, split, args...)
}

func executeObservedTestAllWithEnv(
	t *testing.T,
	root string,
	workingDir string,
	script string,
	cmdEnv []string,
	split bool,
	args ...string,
) testAllRunObservation {
	t.Helper()
	prepareTestAllInvocationTrace(t, root)
	cmd := newTestAllCommandWithEnv(t, workingDir, script, cmdEnv, args...)
	if split {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		return observeTestAllInvocation(
			t,
			root,
			workingDir,
			args,
			append([]byte{}, stdout.Bytes()...),
			append([]byte{}, stderr.Bytes()...),
			nil,
			err,
		)
	}
	combined, err := cmd.CombinedOutput()
	return observeTestAllInvocation(
		t,
		root,
		workingDir,
		args,
		nil,
		nil,
		append([]byte{}, combined...),
		err,
	)
}

func prepareTestAllInvocationTrace(t *testing.T, root string) string {
	t.Helper()
	traceDir := filepath.Join(root, ".test-all-trace")
	if err := os.RemoveAll(traceDir); err != nil {
		t.Fatalf("reset test_all fake-go trace directory %s: %v", traceDir, err)
	}
	if err := os.MkdirAll(traceDir, 0o755); err != nil {
		t.Fatalf("create test_all fake-go trace directory %s: %v", traceDir, err)
	}
	return traceDir
}

func observeTestAllInvocation(
	t *testing.T,
	root string,
	workingDir string,
	args []string,
	stdout []byte,
	stderr []byte,
	combined []byte,
	runErr error,
) testAllRunObservation {
	t.Helper()
	invocationID := testAllInvocationCounter.Add(1)
	mode, reportDir := parseTestAllInvocation(args)
	rawSummary := combined
	if combined == nil {
		rawSummary = stdout
	}

	summaryDecodeStatus := "empty"
	summaryDecodeError := ""
	var summary *testAllSummary
	if len(rawSummary) > 0 {
		var decoded testAllSummary
		if err := json.Unmarshal(rawSummary, &decoded); err != nil {
			summaryDecodeStatus = "failed"
			summaryDecodeError = err.Error()
		} else {
			summaryDecodeStatus = "ok"
			summary = &decoded
		}
	}

	requiresEvidence := runErr != nil
	if summary != nil && (summary.Status != "pass" || summary.FailedCount != 0) {
		requiresEvidence = true
	}

	obs := testAllRunObservation{
		Root:       root,
		WorkingDir: workingDir,
		Mode:       mode,
		ReportDir:  reportDir,
		Args:       append([]string(nil), args...),
		Stdout:     append([]byte(nil), stdout...),
		Stderr:     append([]byte(nil), stderr...),
		Combined:   append([]byte(nil), combined...),
		Err:        runErr,
		Summary:    summary,
	}
	if requiresEvidence {
		obs.Evidence = buildTestAllRunnerEvidence(
			t,
			invocationID,
			obs,
			rawSummary,
			summaryDecodeStatus,
			summaryDecodeError,
		)
		t.Logf("\n%s", obs.Evidence)
	}
	return obs
}

func parseTestAllInvocation(args []string) (mode string, reportDir string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--quick":
			mode = "quick"
		case "--full":
			mode = "full"
		case "--stabilization":
			mode = "stabilization"
		case "--report-dir":
			if i+1 < len(args) {
				reportDir = args[i+1]
				i++
			}
		default:
			if value, ok := strings.CutPrefix(arg, "--report-dir="); ok {
				reportDir = value
			}
		}
	}
	return mode, reportDir
}

func buildTestAllRunnerEvidence(
	t *testing.T,
	invocationID uint64,
	obs testAllRunObservation,
	rawSummary []byte,
	summaryDecodeStatus string,
	summaryDecodeError string,
) string {
	t.Helper()
	var out strings.Builder
	out.WriteString("test_all_runner_observation\n")
	fmt.Fprintf(&out, "test_all_invocation_id=%d\n", invocationID)
	fmt.Fprintf(&out, "test_all_mode=%s\n", diagnosticValue(obs.Mode, "unspecified"))
	fmt.Fprintf(&out, "working_dir_relative=%s\n", diagnosticRelativePath(obs.Root, obs.WorkingDir))
	fmt.Fprintf(&out, "report_dir_relative=%s\n", diagnosticRelativePath(obs.Root, obs.ReportDir))
	fmt.Fprintf(&out, "run_error=%s\n", diagnosticError(obs.Root, obs.Err))
	fmt.Fprintf(&out, "summary_decode_status=%s\n", summaryDecodeStatus)
	if summaryDecodeError != "" {
		fmt.Fprintf(&out, "summary_decode_error=%s\n", redactTestAllDiagnostic(obs.Root, summaryDecodeError))
	}
	if obs.Summary == nil {
		out.WriteString("summary_status=<unknown>\n")
		out.WriteString("summary_failed_count=<unknown>\n")
	} else {
		fmt.Fprintf(&out, "summary_status=%s\n", obs.Summary.Status)
		fmt.Fprintf(&out, "summary_failed_count=%d\n", obs.Summary.FailedCount)
	}
	if obs.Combined == nil {
		appendDiagnosticBytePreview(&out, obs.Root, "split_stderr_preview", obs.Stderr, testAllDiagnosticStderrLimit)
	} else if summaryDecodeStatus == "failed" {
		appendDiagnosticBytePreview(&out, obs.Root, "combined_output_preview", rawSummary, testAllDiagnosticStderrLimit)
	}
	out.WriteString(collectUnexpectedTestAllFailureForMode(
		t,
		obs.Root,
		obs.ReportDir,
		rawSummary,
		obs.Mode,
	))
	return boundTestAllDiagnosticBundle(out.String())
}

func diagnosticValue(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func diagnosticError(root string, err error) string {
	if err == nil {
		return "<nil>"
	}
	return redactTestAllDiagnostic(root, err.Error())
}

func diagnosticRelativePath(root string, path string) string {
	if path == "" {
		return "<empty>"
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "<unresolved>"
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "<unresolved>"
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "<outside-fake-repo>"
	}
	return filepath.ToSlash(filepath.Clean(rel))
}

func appendDiagnosticBytePreview(
	out *strings.Builder,
	root string,
	label string,
	raw []byte,
	limit int64,
) {
	hash := sha256.Sum256(raw)
	fmt.Fprintf(out, "%s_bytes=%d\n", label, len(raw))
	fmt.Fprintf(out, "%s_sha256=%x\n", label, hash[:])
	out.WriteString(label)
	out.WriteString(":\n")
	preview := raw
	truncated := int64(len(preview)) > limit
	if truncated {
		preview = preview[:limit]
	}
	if len(preview) == 0 {
		out.WriteString("  <empty>\n")
	} else {
		out.WriteString(indentDiagnostic(redactTestAllDiagnostic(root, string(preview)), "  "))
		if preview[len(preview)-1] != '\n' {
			out.WriteByte('\n')
		}
	}
	if truncated {
		fmt.Fprintf(out, "%s_truncated=true\n", label)
	}
}

func boundTestAllDiagnosticBundle(input string) string {
	if len(input) <= testAllDiagnosticBundleLimit {
		return input
	}
	const marker = "\ndiagnostics_truncated=true\n"
	limit := testAllDiagnosticBundleLimit - len(marker)
	if limit < 0 {
		return marker
	}
	return input[:limit] + marker
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
	return newTestAllCommandWithEnv(
		t,
		workingDir,
		script,
		testAllHermeticEnv(t, root, env),
		args...,
	)
}

func newTestAllCommandWithEnv(
	t *testing.T,
	workingDir string,
	script string,
	cmdEnv []string,
	args ...string,
) *exec.Cmd {
	t.Helper()
	bashPath, err := exec.LookPath("bash")
	if err != nil {
		t.Fatalf("find bash: %v", err)
	}
	cmd := exec.Command(bashPath, append([]string{script}, args...)...)
	cmd.Dir = workingDir
	cmd.Env = cmdEnv
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

func collectUnexpectedTestAllFailure(
	t *testing.T,
	root string,
	reportDir string,
	rawSummary []byte,
) string {
	t.Helper()
	return collectUnexpectedTestAllFailureForMode(t, root, reportDir, rawSummary, "")
}

func collectUnexpectedTestAllFailureForMode(
	t *testing.T,
	root string,
	reportDir string,
	rawSummary []byte,
	mode string,
) string {
	t.Helper()
	var out strings.Builder
	out.WriteString("test_all unexpected failure evidence\n")
	if mode != "" {
		fmt.Fprintf(&out, "test_all_mode=%s\n", strings.TrimPrefix(mode, "--"))
	}
	out.WriteString("summary:\n")
	out.WriteString(redactTestAllDiagnostic(root, string(rawSummary)))
	if len(rawSummary) == 0 || rawSummary[len(rawSummary)-1] != '\n' {
		out.WriteByte('\n')
	}

	var summary testAllSummary
	if err := json.Unmarshal(rawSummary, &summary); err != nil {
		fmt.Fprintf(&out, "summary_decode_error=%v\n", err)
		appendFakeGoTraceDiagnostics(t, &out, root)
		appendReportArtifactManifest(&out, root, reportDir)
		return out.String()
	}

	var failed []struct {
		Name     string
		ExitCode *int
		Command  string
		Log      string
	}
	for _, step := range summary.Steps {
		if step.Status != "fail" {
			continue
		}
		failed = append(failed, struct {
			Name     string
			ExitCode *int
			Command  string
			Log      string
		}{
			Name:     step.Name,
			ExitCode: step.ExitCode,
			Command:  step.Command,
			Log:      step.Log,
		})
	}
	fmt.Fprintf(&out, "failed_step_count=%d\n", len(failed))
	for _, step := range failed {
		out.WriteString("failed_step:\n")
		fmt.Fprintf(&out, "  name=%s\n", step.Name)
		if step.ExitCode == nil {
			out.WriteString("  exit_code=<nil>\n")
		} else {
			fmt.Fprintf(&out, "  exit_code=%d\n", *step.ExitCode)
		}
		fmt.Fprintf(&out, "  command=%s\n", redactTestAllDiagnostic(root, step.Command))
		fmt.Fprintf(&out, "  log=%s\n", step.Log)
		content, status := readTestAllStepLogForDiagnostic(root, reportDir, step.Log)
		fmt.Fprintf(&out, "  log_status=%s\n", status)
		if content != "" {
			out.WriteString("  log_content:\n")
			out.WriteString(indentDiagnostic(redactTestAllDiagnostic(root, content), "    "))
			if content[len(content)-1] != '\n' {
				out.WriteByte('\n')
			}
		}
	}
	appendFakeGoTraceDiagnostics(t, &out, root)
	appendReportArtifactManifest(&out, root, reportDir)
	return out.String()
}

func readTestAllStepLogForDiagnostic(root, reportDir, relLog string) (string, string) {
	if relLog == "" {
		return "", "missing log path"
	}
	if filepath.IsAbs(relLog) {
		return "", "rejected unsafe log path: absolute path"
	}
	cleanLog := filepath.Clean(relLog)
	if cleanLog == "." || cleanLog == ".." || strings.HasPrefix(cleanLog, ".."+string(os.PathSeparator)) {
		return "", "rejected unsafe log path: escapes report dir"
	}
	absReportDir, err := filepath.Abs(reportDir)
	if err != nil {
		return "", "cannot resolve report dir: " + err.Error()
	}
	candidate := filepath.Join(reportDir, cleanLog)
	absCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return "", "cannot resolve log path: " + err.Error()
	}
	rel, err := filepath.Rel(absReportDir, absCandidate)
	if err != nil {
		return "", "cannot verify log path: " + err.Error()
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", "rejected unsafe log path: escapes report dir"
	}
	content, sha, truncated, err := readDiagnosticFilePreview(absCandidate, testAllDiagnosticLogLimit)
	if err != nil {
		return "", "cannot read log: " + redactTestAllDiagnostic(root, err.Error())
	}
	status := fmt.Sprintf("read sha256=%s bytes=%d", sha, len(content))
	if truncated {
		status += " truncated=true"
	}
	return content, status
}

func appendFakeGoTraceDiagnostics(t *testing.T, out *strings.Builder, root string) {
	t.Helper()
	traceDir := filepath.Join(root, ".test-all-trace")
	entries, err := os.ReadDir(traceDir)
	if err != nil {
		fmt.Fprintf(out, "fake_go_trace_status=unavailable: %v\n", err)
		return
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".trace") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	fmt.Fprintf(out, "fake_go_trace_count=%d\n", len(names))
	if len(names) > testAllDiagnosticMaxTraces {
		names = names[:testAllDiagnosticMaxTraces]
		out.WriteString("fake_go_trace_truncated=true\n")
	}
	for _, name := range names {
		path := filepath.Join(traceDir, name)
		content, sha, truncated, err := readDiagnosticFilePreview(path, testAllDiagnosticTraceLimit)
		out.WriteString("fake_go_trace:\n")
		fmt.Fprintf(out, "  file=%s\n", name)
		if err != nil {
			fmt.Fprintf(out, "  status=read_error: %v\n", err)
			continue
		}
		fmt.Fprintf(out, "  status=read sha256=%s bytes=%d", sha, len(content))
		if truncated {
			out.WriteString(" truncated=true")
		}
		out.WriteByte('\n')
		out.WriteString(indentDiagnostic(redactTestAllDiagnostic(root, content), "    "))
		if content != "" && content[len(content)-1] != '\n' {
			out.WriteByte('\n')
		}
	}
}

func appendReportArtifactManifest(out *strings.Builder, root string, reportDir string) {
	out.WriteString("report_artifact_manifest:\n")
	if reportDir == "" {
		out.WriteString("  status=unavailable: report_dir_missing\n")
		appendMemoryFuzzExpectedArtifactManifest(out, root, reportDir)
		return
	}
	absReportDir, err := filepath.Abs(reportDir)
	if err != nil {
		fmt.Fprintf(out, "  status=cannot_resolve_report_dir: %s\n", redactTestAllDiagnostic(root, err.Error()))
		appendMemoryFuzzExpectedArtifactManifest(out, root, reportDir)
		return
	}
	info, err := os.Lstat(absReportDir)
	if err != nil {
		fmt.Fprintf(out, "  status=unavailable: %s\n", redactTestAllDiagnostic(root, err.Error()))
		appendMemoryFuzzExpectedArtifactManifest(out, root, reportDir)
		return
	}
	if !info.IsDir() {
		out.WriteString("  status=unavailable: report_dir_not_directory\n")
		appendMemoryFuzzExpectedArtifactManifest(out, root, reportDir)
		return
	}

	type manifestEntry struct {
		path   string
		size   int64
		sha256 string
		err    string
	}
	var entries []manifestEntry
	truncated := false
	walkErr := filepath.WalkDir(absReportDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path == absReportDir {
			return nil
		}
		if len(entries) >= testAllDiagnosticManifestMaxFiles {
			truncated = true
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil || !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(absReportDir, path)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return nil
		}
		rel = filepath.ToSlash(filepath.Clean(rel))
		if len(rel) > testAllDiagnosticManifestMaxPathLength {
			truncated = true
			return nil
		}
		size, sha, err := hashDiagnosticFile(path)
		entry := manifestEntry{path: rel, size: size, sha256: sha}
		if err != nil {
			entry.err = redactTestAllDiagnostic(root, err.Error())
		}
		entries = append(entries, entry)
		return nil
	})
	if walkErr != nil {
		fmt.Fprintf(out, "  walk_status=%s\n", redactTestAllDiagnostic(root, walkErr.Error()))
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].path < entries[j].path })
	fmt.Fprintf(out, "  file_count=%d\n", len(entries))
	if truncated {
		out.WriteString("  truncated=true\n")
	}
	for _, entry := range entries {
		if entry.err != "" {
			fmt.Fprintf(out, "  path=%s present=true read_error=%s\n", entry.path, entry.err)
			continue
		}
		fmt.Fprintf(out, "  path=%s present=true size=%d sha256=%s\n", entry.path, entry.size, entry.sha256)
	}
	appendMemoryFuzzExpectedArtifactManifest(out, root, reportDir)
}

func appendMemoryFuzzExpectedArtifactManifest(out *strings.Builder, root string, reportDir string) {
	out.WriteString("memory_fuzz_expected_artifacts:\n")
	for _, rel := range testAllMemoryFuzzExpectedArtifacts {
		if reportDir == "" {
			fmt.Fprintf(out, "  path=%s present=false report_dir_missing=true\n", rel)
			continue
		}
		path := filepath.Join(reportDir, filepath.FromSlash(rel))
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			fmt.Fprintf(out, "  path=%s present=false\n", rel)
			continue
		}
		if err != nil {
			fmt.Fprintf(out, "  path=%s present=unknown error=%s\n", rel, redactTestAllDiagnostic(root, err.Error()))
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			fmt.Fprintf(out, "  path=%s present=false non_regular=true\n", rel)
			continue
		}
		size, sha, err := hashDiagnosticFile(path)
		if err != nil {
			fmt.Fprintf(out, "  path=%s present=unknown error=%s\n", rel, redactTestAllDiagnostic(root, err.Error()))
			continue
		}
		fmt.Fprintf(out, "  path=%s present=true size=%d sha256=%s\n", rel, size, sha)
	}
}

func hashDiagnosticFile(path string) (int64, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer file.Close()

	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return size, "", err
	}
	return size, fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func readDiagnosticFilePreview(path string, limit int64) (string, string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", "", false, err
	}
	defer file.Close()

	hash := sha256.New()
	raw, err := io.ReadAll(io.LimitReader(io.TeeReader(file, hash), limit+1))
	if err != nil {
		return "", "", false, err
	}
	truncated := int64(len(raw)) > limit
	if truncated {
		raw = raw[:limit]
	}
	if _, err := io.Copy(hash, file); err != nil {
		return "", "", false, err
	}
	return string(raw), fmt.Sprintf("%x", hash.Sum(nil)), truncated, nil
}

func redactTestAllDiagnostic(root string, input string) string {
	out := strings.ReplaceAll(input, filepath.Clean(root), "<fake-repo>")
	if absRoot, err := filepath.Abs(root); err == nil {
		out = strings.ReplaceAll(out, absRoot, "<fake-repo>")
	}
	lines := strings.SplitAfter(out, "\n")
	for i, line := range lines {
		upper := strings.ToUpper(line)
		for _, marker := range []string{
			"AUTHORIZATION",
			"CREDENTIAL",
			"PASSWORD",
			"SECRET",
			"TOKEN",
		} {
			if strings.Contains(upper, marker) {
				lines[i] = "<redacted sensitive line>\n"
				break
			}
		}
	}
	return strings.Join(lines, "")
}

func indentDiagnostic(input string, prefix string) string {
	if input == "" {
		return prefix + "<empty>\n"
	}
	lines := strings.SplitAfter(input, "\n")
	var out strings.Builder
	for _, line := range lines {
		if line == "" {
			continue
		}
		out.WriteString(prefix)
		out.WriteString(line)
	}
	return out.String()
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
