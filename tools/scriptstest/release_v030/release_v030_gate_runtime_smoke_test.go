package release_v030

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV030GateRejectsBuildOnlyRuntimeSmokeEvidence(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		("| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky " +
			"| release-a | go test ./compiler/... -run FuzzParser -count=1 |"),
	})
	macosReport, windowsReport := writeReleaseV030RuntimeSmokeReports(t, root)
	raw, err := os.ReadFile(macosReport)
	if err != nil {
		t.Fatalf("read macos smoke report: %v", err)
	}
	raw = []byte(strings.Replace(string(raw), `"ran":true`, `"ran":false`, 1))
	invalidMacosReport := filepath.Join(root, "invalid-macos-runtime-smoke.json")
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
		t.Fatalf("gate should reject runtime smoke evidence with unrun cases\n%s", out)
	}
	if !strings.Contains(
		string(out),
		"runtime smoke evidence invalid: case actors_pingpong did not run",
	) {
		t.Fatalf("gate did not report strict runtime evidence failure:\n%s", out)
	}
	if _, statErr := os.Stat(
		filepath.Join(reportDir, "artifacts", "macos-runtime-smoke.json"),
	); !errors.Is(
		statErr,
		os.ErrNotExist,
	) {
		t.Fatalf(
			"invalid macOS runtime smoke report should not be archived as release evidence, stat err=%v",
			statErr,
		)
	}
}

func TestReleaseV030GateRejectsWrongVersionRuntimeSmokeEvidence(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		("| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky " +
			"| release-a | go test ./compiler/... -run FuzzParser -count=1 |"),
	})
	macosReport, windowsReport := writeReleaseV030RuntimeSmokeReports(t, root)
	raw, err := os.ReadFile(macosReport)
	if err != nil {
		t.Fatalf("read macos smoke report: %v", err)
	}
	raw = []byte(strings.Replace(string(raw), `"version": "v0.3.0"`, `"version": "v0.2.0"`, 1))
	invalidMacosReport := filepath.Join(root, "wrong-version-macos-runtime-smoke.json")
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
			"gate should reject runtime smoke evidence from the wrong release version\n%s",
			out,
		)
	}
	if !strings.Contains(
		string(out),
		"runtime smoke evidence invalid: version is 'v0.2.0', want 'v0.3.0'",
	) {
		t.Fatalf("gate did not report stale runtime evidence version:\n%s", out)
	}
}

func TestReleaseV030GateDoesNotArchivePartialRuntimeEvidenceWhenWindowsReportInvalid(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		("| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky " +
			"| release-a | go test ./compiler/... -run FuzzParser -count=1 |"),
	})
	macosReport, windowsReport := writeReleaseV030RuntimeSmokeReports(t, root)
	raw, err := os.ReadFile(windowsReport)
	if err != nil {
		t.Fatalf("read windows smoke report: %v", err)
	}
	raw = []byte(strings.Replace(string(raw), `"ran":true`, `"ran":false`, 1))
	invalidWindowsReport := filepath.Join(root, "invalid-windows-runtime-smoke.json")
	if err := os.WriteFile(invalidWindowsReport, raw, 0o644); err != nil {
		t.Fatalf("write invalid windows smoke report: %v", err)
	}

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
		"TETRA_MACOS_RUNTIME_SMOKE_REPORT=" + macosReport,
		"TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=" + invalidWindowsReport,
	})
	if err == nil {
		t.Fatalf("gate should reject invalid Windows runtime smoke evidence\n%s", out)
	}
	if !strings.Contains(
		string(out),
		"runtime smoke evidence invalid: case actors_pingpong did not run",
	) {
		t.Fatalf("gate did not report strict Windows runtime evidence failure:\n%s", out)
	}
	for _, artifact := range []string{"macos-runtime-smoke.json", "windows-runtime-smoke.json"} {
		if _, statErr := os.Stat(filepath.Join(reportDir, "artifacts", artifact)); !errors.Is(
			statErr,
			os.ErrNotExist,
		) {
			t.Fatalf(
				("partial runtime smoke report %s should not be archived after " +
					"pair validation failure, stat err=%v"),
				artifact,
				statErr,
			)
		}
	}
}

func TestReleaseV030GateDoesNotArchivePartialRuntimeEvidenceWhenRuntimeCopyFails(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		("| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky " +
			"| release-a | go test ./compiler/... -run FuzzParser -count=1 |"),
	})
	macosReport, windowsReport := writeReleaseV030RuntimeSmokeReports(t, root)
	realCP, err := exec.LookPath("cp")
	if err != nil {
		t.Fatalf("look up cp: %v", err)
	}
	fakeCP := `#!/usr/bin/env bash
set -euo pipefail
if [[ "${1:-}" == "--" ]]; then
  shift
fi
if [[ "${1:-}" == ` + shellSingleQuote(
		windowsReport,
	) + ` && "${2:-}" == *windows-runtime-smoke.json ]]; then
  printf 'partial windows runtime artifact\n' >"$2"
  exit 23
fi
exec ` + shellSingleQuote(realCP) + ` "$@"
`
	if err := os.WriteFile(filepath.Join(root, "bin", "cp"), []byte(fakeCP), 0o755); err != nil {
		t.Fatalf("write fake cp: %v", err)
	}

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
		"TETRA_MACOS_RUNTIME_SMOKE_REPORT=" + macosReport,
		"TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=" + windowsReport,
	})
	if err == nil {
		t.Fatalf("gate should fail when runtime evidence copy fails\n%s", out)
	}
	for _, artifact := range []string{"macos-runtime-smoke.json", "windows-runtime-smoke.json"} {
		if _, statErr := os.Stat(filepath.Join(reportDir, "artifacts", artifact)); !errors.Is(
			statErr,
			os.ErrNotExist,
		) {
			t.Fatalf(
				"partial runtime smoke report %s should not be archived after copy failure, stat err=%v\n%s",
				artifact,
				statErr,
				out,
			)
		}
	}
}

func TestReleaseV030GateAcceptsRuntimeSmokeSourcePathStartingWithDash(t *testing.T) {
	root := releaseV030RunnableFakeRepo(t, []string{
		("| compiler | FuzzParser | testdata/fuzz/FuzzParser/bad | flaky " +
			"| release-a | go test ./compiler/... -run FuzzParser -count=1 |"),
	})
	macosReport, windowsReport := writeReleaseV030RuntimeSmokeReports(t, root)
	dashMacosReport := "-macos-runtime-smoke.json"
	raw, err := os.ReadFile(macosReport)
	if err != nil {
		t.Fatalf("read macos runtime smoke report: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, dashMacosReport), raw, 0o644); err != nil {
		t.Fatalf("write dash-prefixed macos runtime smoke report: %v", err)
	}

	reportDir := filepath.Join(root, "report")
	out, err := runReleaseV030RunnableGateWithEnv(t, root, reportDir, []string{
		"TETRA_SECURITY_REVIEW_SIGNOFF=" + filepath.Join(root, "security-review.md"),
		"TETRA_MACOS_RUNTIME_SMOKE_REPORT=" + dashMacosReport,
		"TETRA_WINDOWS_RUNTIME_SMOKE_REPORT=" + windowsReport,
	})
	if err != nil {
		t.Fatalf("gate should accept dash-prefixed runtime smoke source path: %v\n%s", err, out)
	}
	if _, err := os.Stat(
		filepath.Join(reportDir, "artifacts", "macos-runtime-smoke.json"),
	); err != nil {
		t.Fatalf(
			"macOS runtime smoke artifact was not archived from dash-prefixed source: %v\n%s",
			err,
			out,
		)
	}
}
