package surface

import (
	"errors"
	"fmt"
	"strings"
)

const CrashReportSchemaV1 = "tetra.surface.crash-report.v1"

type SurfaceCrashReportV1 struct {
	Schema           string                      `json:"schema"`
	Model            string                      `json:"model"`
	ReleaseScope     string                      `json:"release_scope"`
	Producer         string                      `json:"producer"`
	Source           string                      `json:"source"`
	ReferenceApp     string                      `json:"reference_app"`
	Target           string                      `json:"target"`
	DiagnosticSchema string                      `json:"diagnostic_schema"`
	Scenarios        []SurfaceCrashScenario      `json:"scenarios"`
	Diagnostics      []SurfaceDiagnosticArtifact `json:"diagnostics"`
	TraceCollection  SurfaceCrashTraceCollection `json:"trace_collection"`
	RestartRecovery  SurfaceRestartEvidence      `json:"restart_recovery"`
	PrivacyPolicy    SurfaceCrashPrivacyPolicy   `json:"privacy_policy"`
	NegativeGuards   SurfaceCrashNegativeGuards  `json:"negative_guards"`
	Pass             bool                        `json:"pass"`
}

type SurfaceCrashScenario struct {
	Name             string `json:"name"`
	Kind             string `json:"kind"`
	Target           string `json:"target"`
	Source           string `json:"source"`
	Trigger          string `json:"trigger"`
	DiagnosticPath   string `json:"diagnostic_path"`
	DiagnosticSHA256 string `json:"diagnostic_sha256"`
	ReportWritten    bool   `json:"report_written"`
	CommandBoundary  bool   `json:"command_boundary"`
	HostCaptured     bool   `json:"host_captured"`
	Restarted        bool   `json:"restarted"`
	ContainsUserData bool   `json:"contains_user_data"`
	Pass             bool   `json:"pass"`
}

type SurfaceDiagnosticArtifact struct {
	Path             string `json:"path"`
	Kind             string `json:"kind"`
	Schema           string `json:"schema"`
	SHA256           string `json:"sha256"`
	SizeBytes        int64  `json:"size_bytes"`
	Redacted         bool   `json:"redacted"`
	ContainsUserData bool   `json:"contains_user_data"`
	Pass             bool   `json:"pass"`
}

type SurfaceCrashTraceCollection struct {
	TracePath  string `json:"trace_path"`
	LogPath    string `json:"log_path"`
	RingBuffer bool   `json:"ring_buffer"`
	MaxBytes   int64  `json:"max_bytes"`
	EventCount int    `json:"event_count"`
	Bounded    bool   `json:"bounded"`
	LocalOnly  bool   `json:"local_only"`
	Pass       bool   `json:"pass"`
}

type SurfaceRestartEvidence struct {
	Scope                string `json:"scope"`
	Target               string `json:"target"`
	RestartClaim         bool   `json:"restart_claim"`
	BeforeRun            bool   `json:"before_run"`
	FailureReportWritten bool   `json:"failure_report_written"`
	AfterRun             bool   `json:"after_run"`
	BeforeExitCode       int    `json:"before_exit_code"`
	AfterExitCode        int    `json:"after_exit_code"`
	StateRestored        string `json:"state_restored"`
	Command              string `json:"command"`
	Pass                 bool   `json:"pass"`
}

type SurfaceCrashPrivacyPolicy struct {
	Policy                   string `json:"policy"`
	RedactionVersion         string `json:"redaction_version"`
	UserDataRedacted         bool   `json:"user_data_redacted"`
	ClipboardPayloadCaptured bool   `json:"clipboard_payload_captured"`
	UserTextCaptured         bool   `json:"user_text_captured"`
	EnvDumped                bool   `json:"env_dumped"`
	HomePathCaptured         bool   `json:"home_path_captured"`
	NetworkUpload            bool   `json:"network_upload"`
	LocalOnly                bool   `json:"local_only"`
	Pass                     bool   `json:"pass"`
}

type SurfaceCrashNegativeGuards struct {
	NoUserDataLeak                    bool `json:"no_user_data_leak"`
	NoClipboardPayload                bool `json:"no_clipboard_payload"`
	NoUserTextPayload                 bool `json:"no_user_text_payload"`
	NoEnvDump                         bool `json:"no_env_dump"`
	NoHomePathLeak                    bool `json:"no_home_path_leak"`
	NoNetworkUpload                   bool `json:"no_network_upload"`
	NoRestartClaimWithoutEvidence     bool `json:"no_restart_claim_without_evidence"`
	NoSilentFailure                   bool `json:"no_silent_failure"`
	NoDocsOnlyCrashClaim              bool `json:"no_docs_only_crash_claim"`
	NoElectronCrashReporterDependency bool `json:"no_electron_crash_reporter_dependency"`
}

func ValidateCrashReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != CrashReportSchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, CrashReportSchemaV1)
	}
	var report SurfaceCrashReportV1
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}
	issues := validateSurfaceCrashReport(report)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateSurfaceCrashReport(report SurfaceCrashReportV1) []string {
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: report.Schema, want: CrashReportSchemaV1},
		{field: "model", got: report.Model, want: "surface-crash-report-v1"},
		{field: "release_scope", got: report.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
		{field: "producer", got: report.Producer, want: "scripts/release/surface/surface-crash-report-smoke.sh"},
		{field: "target", got: report.Target, want: "linux-x64"},
		{field: "diagnostic_schema", got: report.DiagnosticSchema, want: "tetra.surface.diagnostic.v1"},
	} {
		if check.got != check.want {
			issues = append(issues, fmt.Sprintf("%s is %q, want %q", check.field, check.got, check.want))
		}
	}
	if !safeRelativeSourcePath(report.Source) {
		issues = append(issues, "source must be a safe Tetra source path")
	}
	if !surfacePackageSourceMatchesReferenceApp(report.ReferenceApp, report.Source) {
		issues = append(issues, fmt.Sprintf("reference_app %q does not match source %q", report.ReferenceApp, report.Source))
	}
	issues = append(issues, validateSurfaceCrashScenarios(report.Scenarios)...)
	issues = append(issues, validateSurfaceDiagnosticArtifacts(report.Diagnostics)...)
	issues = append(issues, validateSurfaceCrashTraceCollection(report.TraceCollection)...)
	issues = append(issues, validateSurfaceRestartEvidence(report.RestartRecovery)...)
	issues = append(issues, validateSurfaceCrashPrivacyPolicy(report.PrivacyPolicy)...)
	issues = append(issues, validateSurfaceCrashNegativeGuards(report.NegativeGuards)...)
	if !report.Pass {
		issues = append(issues, "pass must be true")
	}
	return issues
}

func validateSurfaceCrashScenarios(scenarios []SurfaceCrashScenario) []string {
	if len(scenarios) == 0 {
		return []string{"scenarios are required"}
	}
	var issues []string
	seen := map[string]SurfaceCrashScenario{}
	for _, scenario := range scenarios {
		kind := strings.TrimSpace(scenario.Kind)
		if kind == "" {
			issues = append(issues, "scenario kind is required")
			continue
		}
		seen[kind] = scenario
		prefix := "scenario " + kind
		if strings.TrimSpace(scenario.Name) == "" || strings.TrimSpace(scenario.Trigger) == "" {
			issues = append(issues, prefix+" name and trigger are required")
		}
		if scenario.Target != "linux-x64" {
			issues = append(issues, prefix+" target must be linux-x64")
		}
		if !safeRelativeSourcePath(scenario.Source) {
			issues = append(issues, prefix+" source must be a safe Tetra source path")
		}
		if !safeRelativeReportPath(scenario.DiagnosticPath) || !strings.HasSuffix(scenario.DiagnosticPath, ".json") {
			issues = append(issues, prefix+" diagnostic_path must be a safe JSON path")
		}
		if !validChecksumLike(scenario.DiagnosticSHA256) {
			issues = append(issues, prefix+" diagnostic_sha256 must be sha256 evidence")
		}
		if scenario.ContainsUserData {
			issues = append(issues, prefix+" must not contain user data")
		}
		if !scenario.ReportWritten || !scenario.Pass {
			issues = append(issues, prefix+" report_written and pass must be true")
		}
	}
	for _, kind := range []string{"command_failure", "host_crash", "restart_recovery"} {
		scenario, ok := seen[kind]
		if !ok {
			issues = append(issues, "scenarios missing "+kind)
			continue
		}
		switch kind {
		case "command_failure":
			if !scenario.CommandBoundary {
				issues = append(issues, "command_failure scenario requires command_boundary")
			}
		case "host_crash":
			if !scenario.HostCaptured {
				issues = append(issues, "host_crash scenario requires host_captured")
			}
		case "restart_recovery":
			if !scenario.Restarted {
				issues = append(issues, "restart_recovery scenario requires restarted evidence")
			}
		}
	}
	return issues
}

func validateSurfaceDiagnosticArtifacts(artifacts []SurfaceDiagnosticArtifact) []string {
	if len(artifacts) < 3 {
		return []string{"at least three diagnostic artifacts are required"}
	}
	var issues []string
	seenKinds := map[string]bool{}
	for _, artifact := range artifacts {
		kind := strings.TrimSpace(artifact.Kind)
		seenKinds[kind] = true
		prefix := "diagnostic artifact " + kind
		if !safeRelativeReportPath(artifact.Path) || !strings.HasSuffix(artifact.Path, ".json") {
			issues = append(issues, prefix+" path must be a safe JSON path")
		}
		if artifact.Schema != "tetra.surface.diagnostic.v1" {
			issues = append(issues, fmt.Sprintf("%s schema is %q, want tetra.surface.diagnostic.v1", prefix, artifact.Schema))
		}
		if !validChecksumLike(artifact.SHA256) {
			issues = append(issues, prefix+" sha256 must be sha256 evidence")
		}
		if artifact.SizeBytes <= 0 {
			issues = append(issues, prefix+" size_bytes must be positive")
		}
		if !artifact.Redacted {
			issues = append(issues, prefix+" redacted must be true")
		}
		if artifact.ContainsUserData {
			issues = append(issues, prefix+" must not contain user data")
		}
		if !artifact.Pass {
			issues = append(issues, prefix+" pass must be true")
		}
	}
	for _, kind := range []string{"command_failure", "host_crash", "restart_recovery"} {
		if !seenKinds[kind] {
			issues = append(issues, "diagnostic artifacts missing "+kind)
		}
	}
	return issues
}

func validateSurfaceCrashTraceCollection(trace SurfaceCrashTraceCollection) []string {
	var issues []string
	if !safeRelativeReportPath(trace.TracePath) || !strings.HasSuffix(trace.TracePath, ".json") {
		issues = append(issues, "trace_collection.trace_path must be a safe JSON path")
	}
	if !safeRelativeReportPath(trace.LogPath) {
		issues = append(issues, "trace_collection.log_path must be safe")
	}
	if !trace.RingBuffer || !trace.Bounded || !trace.LocalOnly || !trace.Pass {
		issues = append(issues, "trace_collection must be bounded local-only ring-buffer evidence")
	}
	if trace.MaxBytes <= 0 || trace.MaxBytes > 65536 {
		issues = append(issues, "trace_collection.max_bytes must be bounded at 65536 bytes or less")
	}
	if trace.EventCount < 3 {
		issues = append(issues, "trace_collection.event_count must cover at least three events")
	}
	return issues
}

func validateSurfaceRestartEvidence(restart SurfaceRestartEvidence) []string {
	var issues []string
	if restart.Scope != "scoped-linux-x64-process-restart-v1" {
		issues = append(issues, fmt.Sprintf("restart_recovery.scope is %q, want scoped-linux-x64-process-restart-v1", restart.Scope))
	}
	if restart.Target != "linux-x64" {
		issues = append(issues, "restart_recovery.target must be linux-x64")
	}
	if !restart.RestartClaim {
		issues = append(issues, "restart_recovery.restart_claim must be true for P24 scoped restart evidence")
	}
	if !restart.BeforeRun || !restart.FailureReportWritten || !restart.AfterRun {
		issues = append(issues, "restart_recovery claim requires before_run, failure_report_written, and after_run evidence")
	}
	if restart.BeforeExitCode != 0 || restart.AfterExitCode != 0 {
		issues = append(issues, "restart_recovery before/after exit codes must be 0")
	}
	if restart.StateRestored != "explicit-startup-state-v1" {
		issues = append(issues, fmt.Sprintf("restart_recovery.state_restored is %q, want explicit-startup-state-v1", restart.StateRestored))
	}
	if strings.TrimSpace(restart.Command) == "" {
		issues = append(issues, "restart_recovery.command is required")
	}
	if !restart.Pass {
		issues = append(issues, "restart_recovery pass must be true")
	}
	return issues
}

func validateSurfaceCrashPrivacyPolicy(policy SurfaceCrashPrivacyPolicy) []string {
	var issues []string
	if policy.Policy != "surface-non-user-data-diagnostics-v1" {
		issues = append(issues, fmt.Sprintf("privacy_policy.policy is %q, want surface-non-user-data-diagnostics-v1", policy.Policy))
	}
	if policy.RedactionVersion != "surface-diagnostic-redaction-v1" {
		issues = append(issues, fmt.Sprintf("privacy_policy.redaction_version is %q, want surface-diagnostic-redaction-v1", policy.RedactionVersion))
	}
	if !policy.UserDataRedacted || !policy.LocalOnly || !policy.Pass {
		issues = append(issues, "privacy_policy must redact user data, remain local-only, and pass")
	}
	for _, leak := range []struct {
		name string
		bad  bool
	}{
		{name: "clipboard_payload_captured", bad: policy.ClipboardPayloadCaptured},
		{name: "user_text_captured", bad: policy.UserTextCaptured},
		{name: "env_dumped", bad: policy.EnvDumped},
		{name: "home_path_captured", bad: policy.HomePathCaptured},
		{name: "network_upload", bad: policy.NetworkUpload},
	} {
		if leak.bad {
			issues = append(issues, fmt.Sprintf("privacy_policy.%s must be false", leak.name))
		}
	}
	return issues
}

func validateSurfaceCrashNegativeGuards(guards SurfaceCrashNegativeGuards) []string {
	var missing []string
	for _, check := range []struct {
		name string
		ok   bool
	}{
		{name: "no_user_data_leak", ok: guards.NoUserDataLeak},
		{name: "no_clipboard_payload", ok: guards.NoClipboardPayload},
		{name: "no_user_text_payload", ok: guards.NoUserTextPayload},
		{name: "no_env_dump", ok: guards.NoEnvDump},
		{name: "no_home_path_leak", ok: guards.NoHomePathLeak},
		{name: "no_network_upload", ok: guards.NoNetworkUpload},
		{name: "no_restart_claim_without_evidence", ok: guards.NoRestartClaimWithoutEvidence},
		{name: "no_silent_failure", ok: guards.NoSilentFailure},
		{name: "no_docs_only_crash_claim", ok: guards.NoDocsOnlyCrashClaim},
		{name: "no_electron_crash_reporter_dependency", ok: guards.NoElectronCrashReporterDependency},
	} {
		if !check.ok {
			missing = append(missing, check.name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("negative_guards missing %s", strings.Join(missing, ", "))}
}
