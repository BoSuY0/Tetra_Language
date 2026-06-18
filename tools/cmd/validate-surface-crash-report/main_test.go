package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surfacecrash"
)

func TestValidateSurfaceCrashReportCommandAcceptsValidReport(t *testing.T) {
	dir := t.TempDir()
	report := commandCrashReport()
	reportPath := filepath.Join(dir, "surface-crash-report.json")
	writeJSON(t, reportPath, report)

	var stdout, stderr bytes.Buffer
	code := runValidateSurfaceCrashReport([]string{"--report", reportPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "surface crash report OK") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestValidateSurfaceCrashReportCommandRejectsSecretLeak(t *testing.T) {
	dir := t.TempDir()
	report := commandCrashReport()
	report.Crashes[0].Diagnostic.Message = "panic includes password=not-redacted"
	report.Crashes[0].SecretScan.ContainsSecrets = true
	reportPath := filepath.Join(dir, "surface-crash-report.json")
	writeJSON(t, reportPath, report)

	var stdout, stderr bytes.Buffer
	code := runValidateSurfaceCrashReport([]string{"--report", reportPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected nonzero exit, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "secret") {
		t.Fatalf("stderr = %q, want secret rejection", stderr.String())
	}
}

func commandCrashReport() surfacecrash.Report {
	return surfacecrash.Report{
		Schema:       surfacecrash.SchemaV1,
		Status:       "pass",
		Level:        surfacecrash.LevelCrashDiagnosticsV1,
		Scope:        "surface-v1-scoped-linux-web-crash-diagnostics",
		ReleaseScope: "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
		GitHead:      "0123456789abcdef0123456789abcdef01234567",
		SameCommit:   true,
		Policy: surfacecrash.CrashPolicy{
			Name:                  "crash-safe-diagnostics-v1",
			RestartPolicy:         "supervised-restart-opt-in-v1",
			DevOverlay:            true,
			ProductionErrorHook:   true,
			ProductionDevOverlay:  false,
			SecretScrubbing:       true,
			ExpectedCrashBoundary: true,
		},
		Crashes: []surfacecrash.CrashEntry{
			{
				ID:             "panic-command",
				Kind:           "panic",
				Status:         "recovered",
				SurfacedToUser: true,
				RecoveryAction: "show-error-boundary-and-restart-background-service",
				ExitCode:       70,
				Source:         surfacecrash.SourceLocation{File: "examples/surface_crash_demo.tetra", Line: 12, Column: 5, Function: "run"},
				Diagnostic:     surfacecrash.Diagnostic{Code: "SURFACE5001", Severity: "error", Message: "Surface command panic recovered and reported", Hint: "open the diagnostic bundle"},
				Bundle:         surfacecrash.ArtifactRef{Path: "crash/panic-command-diagnostic.json", SHA256: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Size: 256},
				SecretScan:     surfacecrash.SecretScan{Scanned: true, ContainsSecrets: false, RedactedFields: []string{"env", "clipboard"}},
			},
			{
				ID:             "expected-negative",
				Kind:           "expected-negative",
				Status:         "diagnostic",
				Expected:       true,
				SurfacedToUser: true,
				RecoveryAction: "report-negative-case-without-crash-promotion",
				ExitCode:       1,
				Source:         surfacecrash.SourceLocation{File: "examples/surface_crash_demo.tetra", Line: 21, Column: 9, Function: "negative_case"},
				Diagnostic:     surfacecrash.Diagnostic{Code: "SURFACE5002", Severity: "error", Message: "Expected negative case reported separately", Hint: "negative case did not count as runtime crash"},
				Bundle:         surfacecrash.ArtifactRef{Path: "crash/expected-negative-diagnostic.json", SHA256: "sha256:abcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd", Size: 256},
				SecretScan:     surfacecrash.SecretScan{Scanned: true, ContainsSecrets: false, RedactedFields: []string{"env"}},
			},
		},
		Operations: []surfacecrash.Operation{
			{Name: "crash report schema validated", Kind: "schema", Ran: true, Pass: true},
			{Name: "secret scrubbing validated", Kind: "security", Ran: true, Pass: true},
			{Name: "source locations validated", Kind: "diagnostic", Ran: true, Pass: true},
			{Name: "error surfacing validated", Kind: "diagnostic", Ran: true, Pass: true},
			{Name: "expected negative cases separated from crashes", Kind: "recovery", Ran: true, Pass: true},
		},
		NegativeGuards: surfacecrash.NegativeGuards{
			CrashSwallowedAsPassRejected:     true,
			SecretLeakRejected:               true,
			MissingSourceLocationRejected:    true,
			MissingDiagnosticBundleRejected:  true,
			UnsurfacedErrorRejected:          true,
			ExpectedNegativeCrashSeparation:  true,
			ProductionDevOverlayRejected:     true,
			SameCommitCrashArtifactsRequired: true,
		},
		NonClaims: []string{
			"No automatic crash recovery beyond the scoped restart policy.",
			"No telemetry upload or external crash reporter.",
			"No secret capture in production error reports.",
			"No Electron crash reporter compatibility claim.",
		},
		Cases: []surfacecrash.CaseReport{
			{Name: "failing app produces useful diagnostics", Kind: "positive", Ran: true, Pass: true},
			{Name: "crash swallowed as pass rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "error report includes secrets rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "missing source location rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "missing diagnostic bundle rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "unsurfaced error rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "expected negative separated from crash", Kind: "negative", Ran: true, Pass: true},
			{Name: "production dev overlay rejected", Kind: "negative", Ran: true, Pass: true},
		},
	}
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}
