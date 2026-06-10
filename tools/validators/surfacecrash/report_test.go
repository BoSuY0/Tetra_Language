package surfacecrash

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateReportAcceptsSurfaceCrashDiagnostics(t *testing.T) {
	raw := mustMarshal(t, validCrashReport())
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportRejectsCrashSwallowedAsPass(t *testing.T) {
	report := validCrashReport()
	report.Crashes[0].Status = "pass"
	report.Crashes[0].Swallowed = true
	report.Crashes[0].SurfacedToUser = false
	raw := mustMarshal(t, report)
	err := ValidateReport(raw)
	if err == nil || !strings.Contains(err.Error(), "swallowed") {
		t.Fatalf("expected swallowed crash rejection, got %v", err)
	}
}

func TestValidateReportRejectsSecretLeak(t *testing.T) {
	report := validCrashReport()
	report.Crashes[0].Diagnostic.Message = "panic while opening token=prod-secret"
	report.Crashes[0].SecretScan.ContainsSecrets = true
	raw := mustMarshal(t, report)
	err := ValidateReport(raw)
	if err == nil || !strings.Contains(err.Error(), "secret") {
		t.Fatalf("expected secret leak rejection, got %v", err)
	}
}

func TestValidateReportRejectsMissingSourceLocation(t *testing.T) {
	report := validCrashReport()
	report.Crashes[0].Source.Line = 0
	raw := mustMarshal(t, report)
	err := ValidateReport(raw)
	if err == nil || !strings.Contains(err.Error(), "source location") {
		t.Fatalf("expected source location rejection, got %v", err)
	}
}

func TestValidateReportRejectsMissingDiagnosticBundle(t *testing.T) {
	report := validCrashReport()
	report.Crashes[0].Bundle.Path = ""
	raw := mustMarshal(t, report)
	err := ValidateReport(raw)
	if err == nil || !strings.Contains(err.Error(), "bundle") {
		t.Fatalf("expected diagnostic bundle rejection, got %v", err)
	}
}

func TestValidateReportRejectsUnsurfacedError(t *testing.T) {
	report := validCrashReport()
	report.Crashes[0].SurfacedToUser = false
	raw := mustMarshal(t, report)
	err := ValidateReport(raw)
	if err == nil || !strings.Contains(err.Error(), "surfaced") {
		t.Fatalf("expected surfaced error rejection, got %v", err)
	}
}

func validCrashReport() Report {
	return Report{
		Schema:       SchemaV1,
		Status:       "pass",
		Level:        LevelCrashDiagnosticsV1,
		Scope:        "surface-v1-scoped-linux-web-crash-diagnostics",
		ReleaseScope: "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
		GitHead:      "0123456789abcdef0123456789abcdef01234567",
		SameCommit:   true,
		Policy: CrashPolicy{
			Name:                  "crash-safe-diagnostics-v1",
			RestartPolicy:         "supervised-restart-opt-in-v1",
			DevOverlay:            true,
			ProductionErrorHook:   true,
			ProductionDevOverlay:  false,
			SecretScrubbing:       true,
			ExpectedCrashBoundary: true,
		},
		Crashes: []CrashEntry{
			{
				ID:             "panic-command",
				Kind:           "panic",
				Status:         "recovered",
				Expected:       false,
				Swallowed:      false,
				SurfacedToUser: true,
				RecoveryAction: "show-error-boundary-and-restart-background-service",
				ExitCode:       70,
				Source:         SourceLocation{File: "examples/surface_crash_demo.tetra", Line: 12, Column: 5, Function: "run"},
				Diagnostic:     Diagnostic{Code: "SURFACE5001", Severity: "error", Message: "Surface command panic recovered and reported", Hint: "open the diagnostic bundle"},
				Bundle:         ArtifactRef{Path: "crash/panic-command-diagnostic.json", SHA256: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Size: 256},
				SecretScan:     SecretScan{Scanned: true, ContainsSecrets: false, RedactedFields: []string{"env", "clipboard"}},
			},
			{
				ID:             "expected-negative",
				Kind:           "expected-negative",
				Status:         "diagnostic",
				Expected:       true,
				Swallowed:      false,
				SurfacedToUser: true,
				RecoveryAction: "report-negative-case-without-crash-promotion",
				ExitCode:       1,
				Source:         SourceLocation{File: "examples/surface_crash_demo.tetra", Line: 21, Column: 9, Function: "negative_case"},
				Diagnostic:     Diagnostic{Code: "SURFACE5002", Severity: "error", Message: "Expected negative case reported separately", Hint: "negative case did not count as runtime crash"},
				Bundle:         ArtifactRef{Path: "crash/expected-negative-diagnostic.json", SHA256: "sha256:abcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd", Size: 256},
				SecretScan:     SecretScan{Scanned: true, ContainsSecrets: false, RedactedFields: []string{"env"}},
			},
		},
		Operations: []Operation{
			{Name: "crash report schema validated", Kind: "schema", Ran: true, Pass: true},
			{Name: "secret scrubbing validated", Kind: "security", Ran: true, Pass: true},
			{Name: "source locations validated", Kind: "diagnostic", Ran: true, Pass: true},
			{Name: "error surfacing validated", Kind: "diagnostic", Ran: true, Pass: true},
			{Name: "expected negative cases separated from crashes", Kind: "recovery", Ran: true, Pass: true},
		},
		NegativeGuards: NegativeGuards{
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
		Cases: []CaseReport{
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

func mustMarshal(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
