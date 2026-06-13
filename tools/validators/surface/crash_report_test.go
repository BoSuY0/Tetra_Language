package surface

import (
	"strings"
	"testing"
)

func TestValidateCrashReportAcceptsCompleteScopedEvidence(t *testing.T) {
	raw := validCrashReportJSON()
	if err := ValidateCrashReport([]byte(raw)); err != nil {
		t.Fatalf("ValidateCrashReport failed: %v\n%s", err, raw)
	}
}

func TestValidateCrashReportRejectsRestartClaimWithoutEvidence(t *testing.T) {
	raw := strings.Replace(validCrashReportJSON(), `"after_run":true`, `"after_run":false`, 1)
	err := ValidateCrashReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing restart evidence to fail")
	}
	if !strings.Contains(err.Error(), "restart_recovery") {
		t.Fatalf("error = %v, want restart_recovery diagnostic", err)
	}
}

func TestValidateCrashReportRejectsUserDataLeak(t *testing.T) {
	raw := strings.Replace(validCrashReportJSON(), `"user_text_captured":false`, `"user_text_captured":true`, 1)
	err := ValidateCrashReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected user text leak to fail")
	}
	if !strings.Contains(err.Error(), "user_text_captured") {
		t.Fatalf("error = %v, want user_text_captured diagnostic", err)
	}
}

func TestValidateCrashReportRejectsMissingHostCrashScenario(t *testing.T) {
	raw := strings.Replace(validCrashReportJSON(), crashScenarioJSON("host_crash"), "", 1)
	err := ValidateCrashReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing host crash scenario to fail")
	}
	if !strings.Contains(err.Error(), "host_crash") {
		t.Fatalf("error = %v, want host_crash diagnostic", err)
	}
}

func validCrashReportJSON() string {
	return `{"schema":"tetra.surface.crash-report.v1","model":"surface-crash-report-v1","release_scope":"surface-v1-linux-web","producer":"scripts/release/surface/surface-crash-report-smoke.sh","source":"examples/surface_reference_command_palette.tetra","reference_app":"command-palette","target":"linux-x64","diagnostic_schema":"tetra.surface.diagnostic.v1","scenarios":[` + crashScenarioJSON("command_failure") + crashScenarioJSON("host_crash") + crashScenarioJSON("restart_recovery") + `],"diagnostics":[{"path":"surface-crash/command-failure.json","kind":"command_failure","schema":"tetra.surface.diagnostic.v1","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size_bytes":256,"redacted":true,"contains_user_data":false,"pass":true},{"path":"surface-crash/host-crash.json","kind":"host_crash","schema":"tetra.surface.diagnostic.v1","sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","size_bytes":256,"redacted":true,"contains_user_data":false,"pass":true},{"path":"surface-crash/restart-recovery.json","kind":"restart_recovery","schema":"tetra.surface.diagnostic.v1","sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","size_bytes":256,"redacted":true,"contains_user_data":false,"pass":true}],"trace_collection":{"trace_path":"surface-crash/surface-app-trace.json","log_path":"surface-crash/surface-app.log","ring_buffer":true,"max_bytes":4096,"event_count":4,"bounded":true,"local_only":true,"pass":true},"restart_recovery":{"scope":"scoped-linux-x64-process-restart-v1","target":"linux-x64","restart_claim":true,"before_run":true,"failure_report_written":true,"after_run":true,"before_exit_code":0,"after_exit_code":0,"state_restored":"explicit-startup-state-v1","command":"surface-crash-work/surface-command-palette-linux-x64","pass":true},"privacy_policy":{"policy":"surface-non-user-data-diagnostics-v1","redaction_version":"surface-diagnostic-redaction-v1","user_data_redacted":true,"clipboard_payload_captured":false,"user_text_captured":false,"env_dumped":false,"home_path_captured":false,"network_upload":false,"local_only":true,"pass":true},"negative_guards":{"no_user_data_leak":true,"no_clipboard_payload":true,"no_user_text_payload":true,"no_env_dump":true,"no_home_path_leak":true,"no_network_upload":true,"no_restart_claim_without_evidence":true,"no_silent_failure":true,"no_docs_only_crash_claim":true,"no_electron_crash_reporter_dependency":true},"pass":true}` + "\n"
}

func crashScenarioJSON(kind string) string {
	switch kind {
	case "command_failure":
		return `{"name":"command failure boundary","kind":"command_failure","target":"linux-x64","source":"examples/surface_reference_command_palette.tetra","trigger":"command.palette.missing","diagnostic_path":"surface-crash/command-failure.json","diagnostic_sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","report_written":true,"command_boundary":true,"host_captured":false,"restarted":false,"contains_user_data":false,"pass":true},`
	case "host_crash":
		return `{"name":"host crash capture","kind":"host_crash","target":"linux-x64","source":"examples/surface_reference_command_palette.tetra","trigger":"surface-host panic harness","diagnostic_path":"surface-crash/host-crash.json","diagnostic_sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","report_written":true,"command_boundary":false,"host_captured":true,"restarted":false,"contains_user_data":false,"pass":true},`
	case "restart_recovery":
		return `{"name":"restart after diagnostic","kind":"restart_recovery","target":"linux-x64","source":"examples/surface_reference_command_palette.tetra","trigger":"restart after command failure report","diagnostic_path":"surface-crash/restart-recovery.json","diagnostic_sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","report_written":true,"command_boundary":false,"host_captured":false,"restarted":true,"contains_user_data":false,"pass":true}`
	default:
		return ""
	}
}
