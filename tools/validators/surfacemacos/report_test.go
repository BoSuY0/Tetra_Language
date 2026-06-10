package surfacemacos

import (
	"strings"
	"testing"
)

func TestValidateReportAcceptsMacOSNonClaimBoundary(t *testing.T) {
	raw := []byte(validMacOSNonClaimBoundaryReport())
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportAcceptsMacOSBetaTargetHostBoundary(t *testing.T) {
	raw := []byte(validMacOSBetaTargetHostBoundaryReport())
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportRejectsMacOSProductionWithoutFullTargetHostDoD(t *testing.T) {
	raw := strings.Replace(validMacOSBetaTargetHostBoundaryReport(),
		`"support_level":"beta-target-host"`,
		`"support_level":"production"`,
		1)
	raw = strings.Replace(raw, `"production_claim":false`, `"production_claim":true`, 1)
	err := ValidateReport([]byte(raw))
	requireIssue(t, err, "production")
}

func TestValidateReportRejectsLinuxHostSyntheticMacOSTargetHostEvidence(t *testing.T) {
	raw := strings.Replace(validMacOSBetaTargetHostBoundaryReport(),
		`"host":"macos-x64"`,
		`"host":"linux-x64"`,
		1)
	err := ValidateReport([]byte(raw))
	requireIssue(t, err, "target-host")
	requireIssue(t, err, "macos-x64")
}

func TestValidateReportRejectsNonNotarizedProductionDistribution(t *testing.T) {
	raw := strings.Replace(validMacOSBetaTargetHostBoundaryReport(),
		`"distribution_claim":false`,
		`"distribution_claim":true`,
		1)
	err := ValidateReport([]byte(raw))
	requireIssue(t, err, "notarized")
	requireIssue(t, err, "production distribution")
}

func TestValidateReportRejectsFullAccessibilityWithoutScreenReaderBridge(t *testing.T) {
	raw := strings.Replace(validMacOSBetaTargetHostBoundaryReport(),
		`"screen_reader_bridge":true`,
		`"screen_reader_bridge":false`,
		1)
	err := ValidateReport([]byte(raw))
	requireIssue(t, err, "screen-reader")
}

func requireIssue(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected issue containing %q, got nil", want)
	}
	if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(want)) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func validMacOSNonClaimBoundaryReport() string {
	return `{
  "schema":"tetra.surface.macos-target.v1",
  "status":"nonclaim",
  "target":"macos-x64",
  "host":"linux-x64",
  "support_level":"unsupported",
  "evidence_kind":"nonclaim-boundary",
  "surface_schema":"tetra.surface.v1",
  "app_shell_abi":"tetra.surface.app-shell.v1",
  "production_claim":false,
  "beta_claim":false,
  "blocked_reason":"macOS Surface target-host evidence requires a real macOS x64 host",
  "capabilities":{"native_window":false,"native_input":false,"clipboard":false,"ime":false,"dpi":false,"menu_bar":false,"dialogs":false,"notifications":false,"accessibility_bridge":false,"screen_reader_bridge":false,"app_shell":false},
  "packaging":{"scope":"nonclaim-boundary","signed":false,"notarized":false,"distribution_claim":false},
  "processes":[{"name":"macOS surface target boundary","kind":"validator","path":"tools/cmd/validate-surface-macos-target","ran":true,"pass":true,"exit_code":0}],
  "negative_guards":{"build_only_rejected":true,"linux_host_synthetic_rejected":true,"production_without_full_dod_rejected":true,"generic_ui_runtime_rejected":true,"non_notarized_production_rejected":true,"accessibility_without_screen_reader_bridge_rejected":true},
  "nonclaims":["macOS Surface production target","build-only macOS UI runtime","linux-host synthetic macOS report","generic tetra.ui.v1 platform UI runtime","non-notarized production distribution","full accessibility without screen-reader bridge"],
  "cases":[
    {"name":"macOS surface target remains unsupported without target-host evidence","kind":"positive","ran":true,"pass":true},
    {"name":"macOS build-only target rejected as Surface UI runtime","kind":"negative","ran":true,"pass":true},
    {"name":"linux-host synthetic macOS report rejected","kind":"negative","ran":true,"pass":true},
    {"name":"non-notarized production distribution rejected","kind":"negative","ran":true,"pass":true},
    {"name":"full accessibility without screen-reader bridge rejected","kind":"negative","ran":true,"pass":true}
  ]
}`
}

func validMacOSBetaTargetHostBoundaryReport() string {
	return `{
  "schema":"tetra.surface.macos-target.v1",
  "status":"beta",
  "target":"macos-x64",
  "host":"macos-x64",
  "support_level":"beta-target-host",
  "evidence_kind":"target-host-surface-beta",
  "surface_schema":"tetra.surface.v1",
  "app_shell_abi":"tetra.surface.app-shell.v1",
  "production_claim":false,
  "beta_claim":true,
  "blocked_reason":"",
  "capabilities":{"native_window":true,"native_input":true,"clipboard":true,"ime":true,"dpi":true,"menu_bar":true,"dialogs":true,"notifications":true,"accessibility_bridge":true,"screen_reader_bridge":true,"app_shell":true},
  "packaging":{"scope":"beta-not-production-distribution","signed":false,"notarized":false,"distribution_claim":false},
  "processes":[{"name":"macOS surface target-host smoke","kind":"runtime","path":"surface-macos-target-host-smoke","ran":true,"pass":true,"exit_code":0}],
  "negative_guards":{"build_only_rejected":true,"linux_host_synthetic_rejected":true,"production_without_full_dod_rejected":true,"generic_ui_runtime_rejected":true,"non_notarized_production_rejected":true,"accessibility_without_screen_reader_bridge_rejected":true},
  "nonclaims":["macOS Surface production target","build-only macOS UI runtime","linux-host synthetic macOS report","generic tetra.ui.v1 platform UI runtime","non-notarized production distribution","full accessibility without screen-reader bridge"],
  "cases":[
    {"name":"macOS beta target-host native window input clipboard ime dpi menu bar shell","kind":"positive","ran":true,"pass":true},
    {"name":"macOS build-only target rejected as Surface UI runtime","kind":"negative","ran":true,"pass":true},
    {"name":"linux-host synthetic macOS report rejected","kind":"negative","ran":true,"pass":true},
    {"name":"non-notarized production distribution rejected","kind":"negative","ran":true,"pass":true},
    {"name":"full accessibility without screen-reader bridge rejected","kind":"negative","ran":true,"pass":true}
  ]
}`
}
