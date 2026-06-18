package release_v030

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV030GateRejectsWrongGitHeadRuntimeSmokeEvidence(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		("| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky " +
			"| release-a | go test ./compiler/... -run FuzzParser -count=1 |"),
	})
	macosReport, windowsReport := writeReleaseV030RuntimeSmokeReports(t, root)
	raw, err := os.ReadFile(macosReport)
	if err != nil {
		t.Fatalf("read macos smoke report: %v", err)
	}
	raw = []byte(
		strings.Replace(
			string(raw),
			`"git_head": "fake-head-for-release-v030-test"`,
			`"git_head": "stale-head"`,
			1,
		),
	)
	invalidMacosReport := filepath.Join(root, "wrong-git-head-macos-runtime-smoke.json")
	if err := os.WriteFile(invalidMacosReport, raw, 0o644); err != nil {
		t.Fatalf("write invalid macos smoke report: %v", err)
	}

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
		"TETRA_MACOS_RUNTIME_SMOKE_REPORT=" + invalidMacosReport,
		"TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=" + windowsReport,
	})
	if err == nil {
		t.Fatalf("gate should reject runtime smoke evidence from the wrong Git head\n%s", out)
	}
	if !strings.Contains(
		string(out),
		("runtime smoke evidence invalid: git_head is 'stale-head', want " +
			"'fake-head-for-release-v030-test'"),
	) {
		t.Fatalf("gate did not report stale runtime evidence Git head:\n%s", out)
	}
}

func TestReleaseV030GateRejectsRunnerRuntimeSmokeEvidence(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		("| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky " +
			"| release-a | go test ./compiler/... -run FuzzParser -count=1 |"),
	})
	macosReport, windowsReport := writeReleaseV030RuntimeSmokeReports(t, root)
	raw, err := os.ReadFile(macosReport)
	if err != nil {
		t.Fatalf("read macos smoke report: %v", err)
	}
	raw = []byte(
		strings.Replace(
			string(raw),
			`"host": "macos-x64",`,
			`"host": "macos-x64",`+"\n  "+`"runner": "cross-runner",`,
			1,
		),
	)
	invalidMacosReport := filepath.Join(root, "runner-macos-runtime-smoke.json")
	if err := os.WriteFile(invalidMacosReport, raw, 0o644); err != nil {
		t.Fatalf("write invalid macos smoke report: %v", err)
	}

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
		"TETRA_MACOS_RUNTIME_SMOKE_REPORT=" + invalidMacosReport,
		"TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=" + windowsReport,
	})
	if err == nil {
		t.Fatalf("gate should reject runtime smoke evidence with a non-empty runner\n%s", out)
	}
	if !strings.Contains(
		string(out),
		"runtime smoke evidence invalid: runner is 'cross-runner', want empty host-native runtime",
	) {
		t.Fatalf("gate did not report non-native runtime evidence runner:\n%s", out)
	}
}

func TestReleaseV030GateRejectsInvalidTimestampRuntimeSmokeEvidence(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		("| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky " +
			"| release-a | go test ./compiler/... -run FuzzParser -count=1 |"),
	})
	macosReport, windowsReport := writeReleaseV030RuntimeSmokeReports(t, root)
	raw, err := os.ReadFile(macosReport)
	if err != nil {
		t.Fatalf("read macos smoke report: %v", err)
	}
	raw = []byte(
		strings.Replace(
			string(raw),
			`"timestamp": "2026-04-30T12:00:00Z"`,
			`"timestamp": "not-a-time"`,
			1,
		),
	)
	invalidMacosReport := filepath.Join(root, "invalid-timestamp-macos-runtime-smoke.json")
	if err := os.WriteFile(invalidMacosReport, raw, 0o644); err != nil {
		t.Fatalf("write invalid macos smoke report: %v", err)
	}

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
		"TETRA_MACOS_RUNTIME_SMOKE_REPORT=" + invalidMacosReport,
		"TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=" + windowsReport,
	})
	if err == nil {
		t.Fatalf("gate should reject runtime smoke evidence with a non-RFC3339 timestamp\n%s", out)
	}
	if !strings.Contains(
		string(out),
		"runtime smoke evidence invalid: timestamp is not RFC3339: 'not-a-time'",
	) {
		t.Fatalf("gate did not report invalid runtime evidence timestamp:\n%s", out)
	}
}

func TestReleaseV030GateRejectsLooseTimestampRuntimeSmokeEvidence(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		("| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky " +
			"| release-a | go test ./compiler/... -run FuzzParser -count=1 |"),
	})
	macosReport, windowsReport := writeReleaseV030RuntimeSmokeReports(t, root)
	raw, err := os.ReadFile(macosReport)
	if err != nil {
		t.Fatalf("read macos smoke report: %v", err)
	}
	raw = []byte(
		strings.Replace(
			string(raw),
			`"timestamp": "2026-04-30T12:00:00Z"`,
			`"timestamp": "2026-04-30 12:00:00+00:00"`,
			1,
		),
	)
	invalidMacosReport := filepath.Join(root, "loose-timestamp-macos-runtime-smoke.json")
	if err := os.WriteFile(invalidMacosReport, raw, 0o644); err != nil {
		t.Fatalf("write invalid macos smoke report: %v", err)
	}

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
		"TETRA_MACOS_RUNTIME_SMOKE_REPORT=" + invalidMacosReport,
		"TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=" + windowsReport,
	})
	if err == nil {
		t.Fatalf(
			"gate should reject runtime smoke evidence with a loose non-RFC3339 timestamp\n%s",
			out,
		)
	}
	if !strings.Contains(
		string(out),
		"runtime smoke evidence invalid: timestamp is not RFC3339: '2026-04-30 12:00:00+00:00'",
	) {
		t.Fatalf("gate did not report loose runtime evidence timestamp:\n%s", out)
	}
}

func TestReleaseV030GateRejectsMissingRequiredRuntimeSmokeCase(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		("| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky " +
			"| release-a | go test ./compiler/... -run FuzzParser -count=1 |"),
	})
	macosReport, windowsReport := writeReleaseV030RuntimeSmokeReports(t, root)
	raw, err := os.ReadFile(macosReport)
	if err != nil {
		t.Fatalf("read macos smoke report: %v", err)
	}
	text := string(raw)
	missingCase := `,` + "\n" + `    {"name":"wait_composition_smoke","src_path":"examples/async/wait_composition_smoke.tetra","out_path":"/tmp/wait_composition_smoke","expected_exit":0,"actual_exit":0,"ran":true,"pass":true}`
	text = strings.Replace(text, missingCase, "", 1)
	text = strings.Replace(text, `"total": 8`, `"total": 7`, 1)
	text = strings.Replace(text, `"passed": 8`, `"passed": 7`, 1)
	invalidMacosReport := filepath.Join(root, "missing-case-macos-runtime-smoke.json")
	if err := os.WriteFile(invalidMacosReport, []byte(text), 0o644); err != nil {
		t.Fatalf("write invalid macos smoke report: %v", err)
	}

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
		"TETRA_MACOS_RUNTIME_SMOKE_REPORT=" + invalidMacosReport,
		"TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=" + windowsReport,
	})
	if err == nil {
		t.Fatalf("gate should reject runtime smoke evidence missing a required case\n%s", out)
	}
	if !strings.Contains(
		string(out),
		"runtime smoke evidence invalid: missing required runtime case wait_composition_smoke",
	) {
		t.Fatalf("gate did not report missing required runtime smoke case:\n%s", out)
	}
}

func TestReleaseV030GateRejectsRuntimeSmokeCaseErrorText(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		("| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky " +
			"| release-a | go test ./compiler/... -run FuzzParser -count=1 |"),
	})
	macosReport, windowsReport := writeReleaseV030RuntimeSmokeReports(t, root)
	raw, err := os.ReadFile(macosReport)
	if err != nil {
		t.Fatalf("read macos smoke report: %v", err)
	}
	raw = []byte(
		strings.Replace(
			string(raw),
			`"name":"actors_pingpong"`,
			`"name":"actors_pingpong","error":"stale failure text"`,
			1,
		),
	)
	invalidMacosReport := filepath.Join(root, "case-error-text-macos-runtime-smoke.json")
	if err := os.WriteFile(invalidMacosReport, raw, 0o644); err != nil {
		t.Fatalf("write invalid macos smoke report: %v", err)
	}

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
		"TETRA_MACOS_RUNTIME_SMOKE_REPORT=" + invalidMacosReport,
		"TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=" + windowsReport,
	})
	if err == nil {
		t.Fatalf("gate should reject runtime smoke evidence with case error text\n%s", out)
	}
	if !strings.Contains(
		string(out),
		"runtime smoke evidence invalid: case actors_pingpong has error text",
	) {
		t.Fatalf("gate did not report runtime smoke case error text:\n%s", out)
	}
}

func TestReleaseV030GateRejectsEmptyRuntimeSmokeCaseName(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		("| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky " +
			"| release-a | go test ./compiler/... -run FuzzParser -count=1 |"),
	})
	macosReport, windowsReport := writeReleaseV030RuntimeSmokeReports(t, root)
	raw, err := os.ReadFile(macosReport)
	if err != nil {
		t.Fatalf("read macos smoke report: %v", err)
	}
	text := string(raw)
	extraCase := `,` + "\n" + `    {"name":"","src_path":"examples/empty_name.tetra","out_path":"/tmp/empty_name","expected_exit":0,"actual_exit":0,"ran":true,"pass":true}`
	text = strings.Replace(text, "\n  ]", extraCase+"\n  ]", 1)
	text = strings.Replace(text, `"total": 8`, `"total": 9`, 1)
	text = strings.Replace(text, `"passed": 8`, `"passed": 9`, 1)
	invalidMacosReport := filepath.Join(root, "empty-case-name-macos-runtime-smoke.json")
	if err := os.WriteFile(invalidMacosReport, []byte(text), 0o644); err != nil {
		t.Fatalf("write invalid macos smoke report: %v", err)
	}

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
		"TETRA_MACOS_RUNTIME_SMOKE_REPORT=" + invalidMacosReport,
		"TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=" + windowsReport,
	})
	if err == nil {
		t.Fatalf("gate should reject runtime smoke evidence with an empty case name\n%s", out)
	}
	if !strings.Contains(
		string(out),
		"runtime smoke evidence invalid: contains a case with empty name",
	) {
		t.Fatalf("gate did not report empty runtime smoke case name:\n%s", out)
	}
}

func TestReleaseV030GateRejectsNonStringRuntimeSmokeCaseName(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		("| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky " +
			"| release-a | go test ./compiler/... -run FuzzParser -count=1 |"),
	})
	macosReport, windowsReport := writeReleaseV030RuntimeSmokeReports(t, root)
	raw, err := os.ReadFile(macosReport)
	if err != nil {
		t.Fatalf("read macos smoke report: %v", err)
	}
	text := string(raw)
	extraCase := `,` + "\n" + `    {"name":123,"src_path":"examples/non_string_name.tetra","out_path":"/tmp/non_string_name","expected_exit":0,"actual_exit":0,"ran":true,"pass":true}`
	text = strings.Replace(text, "\n  ]", extraCase+"\n  ]", 1)
	text = strings.Replace(text, `"total": 8`, `"total": 9`, 1)
	text = strings.Replace(text, `"passed": 8`, `"passed": 9`, 1)
	invalidMacosReport := filepath.Join(root, "non-string-case-name-macos-runtime-smoke.json")
	if err := os.WriteFile(invalidMacosReport, []byte(text), 0o644); err != nil {
		t.Fatalf("write invalid macos smoke report: %v", err)
	}

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
		"TETRA_MACOS_RUNTIME_SMOKE_REPORT=" + invalidMacosReport,
		"TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=" + windowsReport,
	})
	if err == nil {
		t.Fatalf("gate should reject runtime smoke evidence with a non-string case name\n%s", out)
	}
	if !strings.Contains(
		string(out),
		"runtime smoke evidence invalid: case name must be a string",
	) {
		t.Fatalf("gate did not report non-string runtime smoke case name:\n%s", out)
	}
}

func TestReleaseV030GateRejectsNonBooleanRuntimeSmokeCaseStatus(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		("| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky " +
			"| release-a | go test ./compiler/... -run FuzzParser -count=1 |"),
	})
	macosReport, windowsReport := writeReleaseV030RuntimeSmokeReports(t, root)
	raw, err := os.ReadFile(macosReport)
	if err != nil {
		t.Fatalf("read macos smoke report: %v", err)
	}
	text := string(raw)
	extraCase := `,` + "\n" + `    {"name":"non_boolean_status","src_path":"examples/non_boolean_status.tetra","out_path":"/tmp/non_boolean_status","expected_exit":0,"actual_exit":0,"ran":"true","pass":"true"}`
	text = strings.Replace(text, "\n  ]", extraCase+"\n  ]", 1)
	text = strings.Replace(text, `"total": 8`, `"total": 9`, 1)
	text = strings.Replace(text, `"passed": 8`, `"passed": 9`, 1)
	invalidMacosReport := filepath.Join(root, "non-boolean-case-status-macos-runtime-smoke.json")
	if err := os.WriteFile(invalidMacosReport, []byte(text), 0o644); err != nil {
		t.Fatalf("write invalid macos smoke report: %v", err)
	}

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
		"TETRA_MACOS_RUNTIME_SMOKE_REPORT=" + invalidMacosReport,
		"TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=" + windowsReport,
	})
	if err == nil {
		t.Fatalf(
			"gate should reject runtime smoke evidence with non-boolean case status fields\n%s",
			out,
		)
	}
	if !strings.Contains(
		string(out),
		"runtime smoke evidence invalid: case non_boolean_status pass must be a boolean",
	) {
		t.Fatalf("gate did not report non-boolean runtime smoke case status:\n%s", out)
	}
}

func TestReleaseV030GateRejectsNonIntegerRuntimeSmokeCaseExitFields(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		("| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky " +
			"| release-a | go test ./compiler/... -run FuzzParser -count=1 |"),
	})
	macosReport, windowsReport := writeReleaseV030RuntimeSmokeReports(t, root)
	raw, err := os.ReadFile(macosReport)
	if err != nil {
		t.Fatalf("read macos smoke report: %v", err)
	}
	text := string(raw)
	extraCase := `,` + "\n" + `    {"name":"non_integer_exit","src_path":"examples/non_integer_exit.tetra","out_path":"/tmp/non_integer_exit","expected_exit":"0","actual_exit":"0","ran":true,"pass":true}`
	text = strings.Replace(text, "\n  ]", extraCase+"\n  ]", 1)
	text = strings.Replace(text, `"total": 8`, `"total": 9`, 1)
	text = strings.Replace(text, `"passed": 8`, `"passed": 9`, 1)
	invalidMacosReport := filepath.Join(root, "non-integer-case-exit-macos-runtime-smoke.json")
	if err := os.WriteFile(invalidMacosReport, []byte(text), 0o644); err != nil {
		t.Fatalf("write invalid macos smoke report: %v", err)
	}

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
		"TETRA_MACOS_RUNTIME_SMOKE_REPORT=" + invalidMacosReport,
		"TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=" + windowsReport,
	})
	if err == nil {
		t.Fatalf(
			"gate should reject runtime smoke evidence with non-integer case exit fields\n%s",
			out,
		)
	}
	if !strings.Contains(
		string(out),
		"runtime smoke evidence invalid: case non_integer_exit expected_exit must be an integer",
	) {
		t.Fatalf("gate did not report non-integer runtime smoke case exit field:\n%s", out)
	}
}

func TestReleaseV030GateRejectsNonIntegerRuntimeSmokeCounts(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		("| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky " +
			"| release-a | go test ./compiler/... -run FuzzParser -count=1 |"),
	})
	macosReport, windowsReport := writeReleaseV030RuntimeSmokeReports(t, root)
	raw, err := os.ReadFile(macosReport)
	if err != nil {
		t.Fatalf("read macos smoke report: %v", err)
	}
	text := strings.Replace(string(raw), `"failed": 0`, `"failed": false`, 1)
	invalidMacosReport := filepath.Join(root, "non-integer-counts-macos-runtime-smoke.json")
	if err := os.WriteFile(invalidMacosReport, []byte(text), 0o644); err != nil {
		t.Fatalf("write invalid macos smoke report: %v", err)
	}

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
		"TETRA_MACOS_RUNTIME_SMOKE_REPORT=" + invalidMacosReport,
		"TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=" + windowsReport,
	})
	if err == nil {
		t.Fatalf(
			"gate should reject runtime smoke evidence with non-integer summary counts\n%s",
			out,
		)
	}
	if !strings.Contains(string(out), "runtime smoke evidence invalid: failed must be an integer") {
		t.Fatalf("gate did not report non-integer runtime smoke summary count:\n%s", out)
	}
}

func TestReleaseV030GateRejectsNonBooleanRuntimeSmokeIslandsDebug(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		("| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky " +
			"| release-a | go test ./compiler/... -run FuzzParser -count=1 |"),
	})
	macosReport, windowsReport := writeReleaseV030RuntimeSmokeReports(t, root)
	raw, err := os.ReadFile(macosReport)
	if err != nil {
		t.Fatalf("read macos smoke report: %v", err)
	}
	text := strings.Replace(string(raw), `"islands_debug": false`, `"islands_debug": "false"`, 1)
	invalidMacosReport := filepath.Join(root, "non-boolean-islands-debug-macos-runtime-smoke.json")
	if err := os.WriteFile(invalidMacosReport, []byte(text), 0o644); err != nil {
		t.Fatalf("write invalid macos smoke report: %v", err)
	}

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
		"TETRA_MACOS_RUNTIME_SMOKE_REPORT=" + invalidMacosReport,
		"TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=" + windowsReport,
	})
	if err == nil {
		t.Fatalf(
			"gate should reject runtime smoke evidence with non-boolean islands_debug\n%s",
			out,
		)
	}
	if !strings.Contains(
		string(out),
		"runtime smoke evidence invalid: islands_debug must be a boolean",
	) {
		t.Fatalf("gate did not report non-boolean runtime smoke islands_debug:\n%s", out)
	}
}

func TestReleaseV030GateRejectsNonBooleanRuntimeSmokeBuildOnly(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		("| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky " +
			"| release-a | go test ./compiler/... -run FuzzParser -count=1 |"),
	})
	macosReport, windowsReport := writeReleaseV030RuntimeSmokeReports(t, root)
	raw, err := os.ReadFile(macosReport)
	if err != nil {
		t.Fatalf("read macos smoke report: %v", err)
	}
	text := strings.Replace(
		string(raw),
		`"host": "macos-x64",`,
		`"host": "macos-x64",`+"\n  "+`"build_only": 0,`,
		1,
	)
	invalidMacosReport := filepath.Join(root, "non-boolean-build-only-macos-runtime-smoke.json")
	if err := os.WriteFile(invalidMacosReport, []byte(text), 0o644); err != nil {
		t.Fatalf("write invalid macos smoke report: %v", err)
	}

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
		"TETRA_MACOS_RUNTIME_SMOKE_REPORT=" + invalidMacosReport,
		"TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=" + windowsReport,
	})
	if err == nil {
		t.Fatalf("gate should reject runtime smoke evidence with non-boolean build_only\n%s", out)
	}
	if !strings.Contains(
		string(out),
		"runtime smoke evidence invalid: build_only must be a boolean",
	) {
		t.Fatalf("gate did not report non-boolean runtime smoke build_only:\n%s", out)
	}
}

func TestReleaseV030GateRejectsNonStringRuntimeSmokeRunner(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		("| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky " +
			"| release-a | go test ./compiler/... -run FuzzParser -count=1 |"),
	})
	macosReport, windowsReport := writeReleaseV030RuntimeSmokeReports(t, root)
	raw, err := os.ReadFile(macosReport)
	if err != nil {
		t.Fatalf("read macos smoke report: %v", err)
	}
	text := strings.Replace(
		string(raw),
		`"host": "macos-x64",`,
		`"host": "macos-x64",`+"\n  "+`"runner": 0,`,
		1,
	)
	invalidMacosReport := filepath.Join(root, "non-string-runner-macos-runtime-smoke.json")
	if err := os.WriteFile(invalidMacosReport, []byte(text), 0o644); err != nil {
		t.Fatalf("write invalid macos smoke report: %v", err)
	}

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
		"TETRA_MACOS_RUNTIME_SMOKE_REPORT=" + invalidMacosReport,
		"TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=" + windowsReport,
	})
	if err == nil {
		t.Fatalf("gate should reject runtime smoke evidence with a non-string runner\n%s", out)
	}
	if !strings.Contains(string(out), "runtime smoke evidence invalid: runner must be a string") {
		t.Fatalf("gate did not report non-string runtime smoke runner:\n%s", out)
	}
}
