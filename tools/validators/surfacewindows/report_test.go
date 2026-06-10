package surfacewindows

import (
	"strings"
	"testing"
)

func TestValidateReportAcceptsWindowsNonClaimBoundary(t *testing.T) {
	raw := []byte(validWindowsNonClaimBoundaryReport())
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportAcceptsWindowsBetaTargetHostBoundary(t *testing.T) {
	raw := []byte(validWindowsBetaTargetHostBoundaryReport())
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportRejectsWindowsProductionWithoutFullTargetHostDoD(t *testing.T) {
	raw := strings.Replace(validWindowsBetaTargetHostBoundaryReport(),
		`"support_level":"beta-target-host"`,
		`"support_level":"production"`,
		1)
	raw = strings.Replace(raw, `"production_claim":false`, `"production_claim":true`, 1)
	err := ValidateReport([]byte(raw))
	requireIssue(t, err, "production")
}

func TestValidateReportRejectsBuildOnlyWindowsRuntimeEvidence(t *testing.T) {
	raw := strings.Replace(validWindowsBetaTargetHostBoundaryReport(),
		`"evidence_kind":"target-host-surface-beta"`,
		`"evidence_kind":"build-only"`,
		1)
	err := ValidateReport([]byte(raw))
	requireIssue(t, err, "build-only")
}

func TestValidateReportRejectsLinuxHostSyntheticWindowsTargetHostEvidence(t *testing.T) {
	raw := strings.Replace(validWindowsBetaTargetHostBoundaryReport(),
		`"host":"windows-x64"`,
		`"host":"linux-x64"`,
		1)
	err := ValidateReport([]byte(raw))
	requireIssue(t, err, "target-host")
	requireIssue(t, err, "windows-x64")
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

func validWindowsNonClaimBoundaryReport() string {
	return `{
  "schema":"tetra.surface.windows-target.v1",
  "status":"nonclaim",
  "target":"windows-x64",
  "host":"linux-x64",
  "support_level":"unsupported",
  "evidence_kind":"nonclaim-boundary",
  "surface_schema":"tetra.surface.v1",
  "app_shell_abi":"tetra.surface.app-shell.v1",
  "production_claim":false,
  "beta_claim":false,
  "blocked_reason":"Windows Surface target-host evidence requires a real Windows x64 host",
  "capabilities":{"native_window":false,"native_input":false,"clipboard":false,"ime":false,"dpi":false,"menus":false,"dialogs":false,"notifications":false,"accessibility_bridge":false,"app_shell":false},
  "processes":[{"name":"windows surface target boundary","kind":"validator","path":"tools/cmd/validate-surface-windows-target","ran":true,"pass":true,"exit_code":0}],
  "negative_guards":{"build_only_rejected":true,"linux_host_synthetic_rejected":true,"production_without_full_dod_rejected":true,"generic_ui_runtime_rejected":true},
  "nonclaims":["Windows Surface production target","build-only Windows UI runtime","linux-host synthetic Windows report","generic tetra.ui.v1 platform UI runtime"],
  "cases":[
    {"name":"windows surface target remains unsupported without target-host evidence","kind":"positive","ran":true,"pass":true},
    {"name":"windows build-only target rejected as Surface UI runtime","kind":"negative","ran":true,"pass":true},
    {"name":"linux-host synthetic Windows report rejected","kind":"negative","ran":true,"pass":true}
  ]
}`
}

func validWindowsBetaTargetHostBoundaryReport() string {
	return `{
  "schema":"tetra.surface.windows-target.v1",
  "status":"beta",
  "target":"windows-x64",
  "host":"windows-x64",
  "support_level":"beta-target-host",
  "evidence_kind":"target-host-surface-beta",
  "surface_schema":"tetra.surface.v1",
  "app_shell_abi":"tetra.surface.app-shell.v1",
  "production_claim":false,
  "beta_claim":true,
  "blocked_reason":"",
  "capabilities":{"native_window":true,"native_input":true,"clipboard":true,"ime":true,"dpi":true,"menus":true,"dialogs":true,"notifications":true,"accessibility_bridge":true,"app_shell":true},
  "processes":[{"name":"windows surface target-host smoke","kind":"runtime","path":"surface-windows-target-host-smoke.exe","ran":true,"pass":true,"exit_code":0}],
  "negative_guards":{"build_only_rejected":true,"linux_host_synthetic_rejected":true,"production_without_full_dod_rejected":true,"generic_ui_runtime_rejected":true},
  "nonclaims":["Windows Surface production target","build-only Windows UI runtime","linux-host synthetic Windows report","generic tetra.ui.v1 platform UI runtime"],
  "cases":[
    {"name":"windows beta target-host native window input clipboard ime dpi shell","kind":"positive","ran":true,"pass":true},
    {"name":"windows build-only target rejected as Surface UI runtime","kind":"negative","ran":true,"pass":true},
    {"name":"linux-host synthetic Windows report rejected","kind":"negative","ran":true,"pass":true}
  ]
}`
}
