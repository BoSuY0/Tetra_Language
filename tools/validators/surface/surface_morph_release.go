package surface

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ---- crash_report.go ----

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
		{
			field: "producer",
			got:   report.Producer,
			want:  "scripts/release/surface/surface-crash-report-smoke.sh",
		},
		{field: "target", got: report.Target, want: "linux-x64"},
		{field: "diagnostic_schema", got: report.DiagnosticSchema, want: "tetra.surface.diagnostic.v1"},
	} {
		if check.got != check.want {
			issues = append(
				issues,
				fmt.Sprintf("%s is %q, want %q", check.field, check.got, check.want),
			)
		}
	}
	if !safeRelativeSourcePath(report.Source) {
		issues = append(issues, "source must be a safe Tetra source path")
	}
	if !surfacePackageSourceMatchesReferenceApp(report.ReferenceApp, report.Source) {
		issues = append(
			issues,
			fmt.Sprintf(
				"reference_app %q does not match source %q",
				report.ReferenceApp,
				report.Source,
			),
		)
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
		if !safeRelativeReportPath(scenario.DiagnosticPath) ||
			!strings.HasSuffix(scenario.DiagnosticPath, ".json") {
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
			issues = append(
				issues,
				fmt.Sprintf(
					"%s schema is %q, want tetra.surface.diagnostic.v1",
					prefix,
					artifact.Schema,
				),
			)
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
		issues = append(
			issues,
			fmt.Sprintf(
				"restart_recovery.scope is %q, want scoped-linux-x64-process-restart-v1",
				restart.Scope,
			),
		)
	}
	if restart.Target != "linux-x64" {
		issues = append(issues, "restart_recovery.target must be linux-x64")
	}
	if !restart.RestartClaim {
		issues = append(
			issues,
			"restart_recovery.restart_claim must be true for P24 scoped restart evidence",
		)
	}
	if !restart.BeforeRun || !restart.FailureReportWritten || !restart.AfterRun {
		issues = append(
			issues,
			"restart_recovery claim requires before_run, failure_report_written, and after_run evidence",
		)
	}
	if restart.BeforeExitCode != 0 || restart.AfterExitCode != 0 {
		issues = append(issues, "restart_recovery before/after exit codes must be 0")
	}
	if restart.StateRestored != "explicit-startup-state-v1" {
		issues = append(
			issues,
			fmt.Sprintf(
				"restart_recovery.state_restored is %q, want explicit-startup-state-v1",
				restart.StateRestored,
			),
		)
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
		issues = append(
			issues,
			fmt.Sprintf(
				"privacy_policy.policy is %q, want surface-non-user-data-diagnostics-v1",
				policy.Policy,
			),
		)
	}
	if policy.RedactionVersion != "surface-diagnostic-redaction-v1" {
		issues = append(
			issues,
			fmt.Sprintf(
				"privacy_policy.redaction_version is %q, want surface-diagnostic-redaction-v1",
				policy.RedactionVersion,
			),
		)
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

// ---- dev_workflow.go ----

const DevWorkflowSchemaV1 = "tetra.surface.dev-workflow.v1"

type SurfaceDevWorkflowReport struct {
	Schema                 string                           `json:"schema"`
	Model                  string                           `json:"model"`
	ReleaseScope           string                           `json:"release_scope"`
	Command                string                           `json:"command"`
	Source                 string                           `json:"source"`
	Target                 string                           `json:"target"`
	Mode                   string                           `json:"mode"`
	ReloadSemantics        string                           `json:"reload_semantics"`
	ProcessRestartRequired bool                             `json:"process_restart_required"`
	HotReloadClaim         bool                             `json:"hot_reload_claim"`
	Watch                  bool                             `json:"watch"`
	SupportedTargets       []string                         `json:"supported_targets"`
	Steps                  []SurfaceDevWorkflowStepReport   `json:"steps"`
	SourceDiagnostics      []SurfaceDevWorkflowDiagnostic   `json:"source_diagnostics"`
	MorphToPixels          *MorphToPixelsChainReport        `json:"morph_to_pixels,omitempty"`
	NegativeGuards         SurfaceDevWorkflowNegativeGuards `json:"negative_guards"`
	Pass                   bool                             `json:"pass"`
}

type SurfaceDevWorkflowStepReport struct {
	Name            string   `json:"name"`
	Kind            string   `json:"kind"`
	ChangedPath     string   `json:"changed_path"`
	OutputPath      string   `json:"output_path"`
	DurationMS      int64    `json:"duration_ms"`
	CompiledModules []string `json:"compiled_modules"`
	CacheHits       []string `json:"cache_hits"`
	Pass            bool     `json:"pass"`
	Error           string   `json:"error,omitempty"`
}

type SurfaceDevWorkflowDiagnostic struct {
	Kind     string `json:"kind"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	Pass     bool   `json:"pass"`
}

type SurfaceDevWorkflowNegativeGuards struct {
	NoHotReloadClaim                   bool `json:"no_hot_reload_claim"`
	FullRestartDocumentedAsFastRebuild bool `json:"full_restart_documented_as_fast_rebuild"`
	NoElectronDevServer                bool `json:"no_electron_dev_server"`
	NoReactFastRefresh                 bool `json:"no_react_fast_refresh"`
	NoDOMHotReload                     bool `json:"no_dom_hot_reload"`
}

func ValidateDevWorkflowReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != DevWorkflowSchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, DevWorkflowSchemaV1)
	}

	var report SurfaceDevWorkflowReport
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}

	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: report.Schema, want: DevWorkflowSchemaV1},
		{field: "model", got: report.Model, want: "surface-dev-workflow-v1"},
		{field: "release_scope", got: report.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
		{field: "command", got: report.Command, want: "tetra surface dev"},
		{field: "mode", got: report.Mode, want: "fast-rebuild"},
		{field: "reload_semantics", got: report.ReloadSemantics, want: "fast-rebuild"},
	} {
		if check.got != check.want {
			issues = append(
				issues,
				fmt.Sprintf("%s is %q, want %q", check.field, check.got, check.want),
			)
		}
	}
	if strings.TrimSpace(report.Source) == "" {
		issues = append(issues, "source is required")
	}
	if report.Target != "linux-x64" {
		issues = append(
			issues,
			fmt.Sprintf(
				"target is %q, want linux-x64 for current fast rebuild evidence",
				report.Target,
			),
		)
	}
	if !report.ProcessRestartRequired {
		issues = append(
			issues,
			"process_restart_required must be true so the report cannot imply hot reload",
		)
	}
	if report.HotReloadClaim {
		issues = append(issues, "hot reload claim is forbidden for fast rebuild evidence")
	}
	issues = append(
		issues,
		validateExactStringList(
			"supported_targets",
			report.SupportedTargets,
			[]string{"headless", "linux-x64", "wasm32-web"},
		)...)
	issues = append(issues, validateSurfaceDevWorkflowSteps(report.Steps)...)
	issues = append(issues, validateSurfaceDevWorkflowDiagnostics(report.SourceDiagnostics)...)
	if report.MorphToPixels != nil {
		issues = append(
			issues,
			validateMorphToPixelsChain("morph_to_pixels", *report.MorphToPixels, report.Source)...)
	}
	issues = append(issues, validateSurfaceDevWorkflowNegativeGuards(report.NegativeGuards)...)
	if !report.Pass {
		issues = append(issues, "pass must be true")
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateSurfaceDevWorkflowSteps(steps []SurfaceDevWorkflowStepReport) []string {
	if len(steps) == 0 {
		return []string{"steps are required"}
	}
	byKind := map[string]SurfaceDevWorkflowStepReport{}
	var issues []string
	for _, step := range steps {
		if strings.TrimSpace(step.Name) == "" {
			issues = append(issues, "step name is required")
		}
		if strings.TrimSpace(step.Kind) == "" {
			issues = append(issues, "step kind is required")
			continue
		}
		if _, exists := byKind[step.Kind]; exists {
			issues = append(issues, fmt.Sprintf("duplicate step kind %s", step.Kind))
		}
		byKind[step.Kind] = step
		if strings.TrimSpace(step.OutputPath) == "" {
			issues = append(issues, fmt.Sprintf("%s output_path is required", step.Kind))
		}
		if step.DurationMS < 0 {
			issues = append(issues, fmt.Sprintf("%s duration_ms must be non-negative", step.Kind))
		}
		if !step.Pass {
			issues = append(issues, fmt.Sprintf("%s pass must be true", step.Kind))
		}
	}
	for _, required := range []string{
		"initial",
		"warm-cache",
		"token-change",
		"recipe-change",
		"source-change",
	} {
		step, ok := byKind[required]
		if !ok {
			issues = append(issues, fmt.Sprintf("steps missing %s", required))
			continue
		}
		switch required {
		case "warm-cache":
			if len(step.CompiledModules) != 0 || len(step.CacheHits) == 0 {
				issues = append(
					issues,
					"warm-cache step must have zero compiled modules and at least one cache hit",
				)
			}
		case "token-change", "recipe-change", "source-change":
			if strings.TrimSpace(step.ChangedPath) == "" {
				issues = append(issues, fmt.Sprintf("%s changed_path is required", required))
			}
			if len(step.CompiledModules) == 0 {
				issues = append(issues, fmt.Sprintf("%s must compile the changed module", required))
			}
		}
	}
	return issues
}

func validateSurfaceDevWorkflowDiagnostics(diags []SurfaceDevWorkflowDiagnostic) []string {
	if len(diags) == 0 {
		return []string{"source_diagnostics are required"}
	}
	var issues []string
	seen := map[string]bool{}
	for _, diag := range diags {
		if strings.TrimSpace(diag.Kind) == "" {
			issues = append(issues, "source_diagnostics kind is required")
			continue
		}
		seen[diag.Kind] = true
		if strings.TrimSpace(diag.Path) == "" {
			issues = append(
				issues,
				fmt.Sprintf("source_diagnostics %s path is required", diag.Kind),
			)
		}
		if diag.Line <= 0 || diag.Column <= 0 {
			issues = append(
				issues,
				fmt.Sprintf("source_diagnostics %s requires line and column", diag.Kind),
			)
		}
		if strings.TrimSpace(diag.Code) == "" || strings.TrimSpace(diag.Message) == "" ||
			strings.TrimSpace(diag.Severity) == "" {
			issues = append(
				issues,
				fmt.Sprintf(
					"source_diagnostics %s requires code, message, and severity",
					diag.Kind,
				),
			)
		}
		if diag.Severity == "info" && !diag.Pass {
			issues = append(
				issues,
				fmt.Sprintf("source_diagnostics %s info row must pass", diag.Kind),
			)
		}
	}
	for _, required := range []string{"token", "recipe", "source"} {
		if !seen[required] {
			issues = append(issues, fmt.Sprintf("source_diagnostics missing %s", required))
		}
	}
	return issues
}

func validateSurfaceDevWorkflowNegativeGuards(guards SurfaceDevWorkflowNegativeGuards) []string {
	if guards.NoHotReloadClaim &&
		guards.FullRestartDocumentedAsFastRebuild &&
		guards.NoElectronDevServer &&
		guards.NoReactFastRefresh &&
		guards.NoDOMHotReload {
		return nil
	}
	return []string{
		("negative_guards must reject hot reload, Electron dev server, " +
			"React Fast Refresh, and DOM hot reload claims"),
	}
}

// ---- i18n.go ----

const I18nSchemaV1 = "tetra.surface.i18n.v1"

type SurfaceI18nReportV1 struct {
	Schema          string                     `json:"schema"`
	Model           string                     `json:"model"`
	ReleaseScope    string                     `json:"release_scope"`
	Producer        string                     `json:"producer"`
	Source          string                     `json:"source"`
	ReferenceApp    string                     `json:"reference_app"`
	Target          string                     `json:"target"`
	StringTables    []SurfaceI18nStringTable   `json:"string_tables"`
	LocaleSelection SurfaceI18nLocaleSelection `json:"locale_selection"`
	Lookups         []SurfaceI18nLookup        `json:"lookups"`
	FormatHooks     []SurfaceI18nFormatHook    `json:"format_hooks"`
	TextDirection   SurfaceI18nTextDirection   `json:"text_direction"`
	LocalizedForm   SurfaceI18nLocalizedForm   `json:"localized_form"`
	NegativeGuards  SurfaceI18nNegativeGuards  `json:"negative_guards"`
	Pass            bool                       `json:"pass"`
}

type SurfaceI18nStringTable struct {
	Locale     string `json:"locale"`
	EntryCount int    `json:"entry_count"`
	Checksum   string `json:"checksum"`
	Primary    bool   `json:"primary"`
	Fallback   bool   `json:"fallback"`
	Pass       bool   `json:"pass"`
}

type SurfaceI18nLocaleSelection struct {
	RequestedLocale           string `json:"requested_locale"`
	SelectedLocale            string `json:"selected_locale"`
	FallbackLocale            string `json:"fallback_locale"`
	FallbackUsed              bool   `json:"fallback_used"`
	UnsupportedLocaleRejected bool   `json:"unsupported_locale_rejected"`
	Pass                      bool   `json:"pass"`
}

type SurfaceI18nLookup struct {
	Key            string `json:"key"`
	Locale         string `json:"locale"`
	ResolvedLocale string `json:"resolved_locale"`
	Source         string `json:"source"`
	MissingKey     bool   `json:"missing_key"`
	FallbackUsed   bool   `json:"fallback_used"`
	DiagnosticCode int    `json:"diagnostic_code"`
	Pass           bool   `json:"pass"`
}

type SurfaceI18nFormatHook struct {
	Kind          string `json:"kind"`
	Locale        string `json:"locale"`
	Input         string `json:"input"`
	Output        string `json:"output"`
	Deterministic bool   `json:"deterministic"`
	ICUClaim      bool   `json:"icu_claim"`
	Pass          bool   `json:"pass"`
}

type SurfaceI18nTextDirection struct {
	DefaultDirection  string `json:"default_direction"`
	RTLPlaceholder    bool   `json:"rtl_placeholder"`
	FullBidiSupported bool   `json:"full_bidi_supported"`
	FullBidiClaim     bool   `json:"full_bidi_claim"`
	ShapingProof      bool   `json:"shaping_proof"`
	Nonclaim          string `json:"nonclaim"`
	Pass              bool   `json:"pass"`
}

type SurfaceI18nLocalizedForm struct {
	Shape                string   `json:"shape"`
	Source               string   `json:"source"`
	Imports              []string `json:"imports"`
	Compiles             bool     `json:"compiles"`
	Runs                 bool     `json:"runs"`
	ExitCode             int      `json:"exit_code"`
	LocalizedStrings     bool     `json:"localized_strings"`
	FallbackEvidence     bool     `json:"fallback_evidence"`
	MissingKeyDiagnostic bool     `json:"missing_key_diagnostic"`
	FormatHookEvidence   bool     `json:"format_hook_evidence"`
	ResolvesToBlock      bool     `json:"resolves_to_block"`
	Pass                 bool     `json:"pass"`
}

type SurfaceI18nNegativeGuards struct {
	NoFullICUClaim             bool `json:"no_full_icu_claim"`
	NoFullBidiClaim            bool `json:"no_full_bidi_claim"`
	NoRTLProductionClaim       bool `json:"no_rtl_production_claim"`
	NoMissingKeySilentFallback bool `json:"no_missing_key_silent_fallback"`
	NoDocsOnlyI18nClaim        bool `json:"no_docs_only_i18n_claim"`
	NoReactIntlRuntime         bool `json:"no_react_intl_runtime"`
	NoPlatformLocaleDependency bool `json:"no_platform_locale_dependency"`
}

func ValidateI18nReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != I18nSchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, I18nSchemaV1)
	}
	var report SurfaceI18nReportV1
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}
	issues := validateSurfaceI18nReport(report)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateSurfaceI18nReport(report SurfaceI18nReportV1) []string {
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: report.Schema, want: I18nSchemaV1},
		{field: "model", got: report.Model, want: "surface-i18n-v1"},
		{field: "release_scope", got: report.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
		{field: "producer", got: report.Producer, want: "scripts/release/surface/surface-i18n-smoke.sh"},
		{field: "reference_app", got: report.ReferenceApp, want: "localized-form"},
		{field: "target", got: report.Target, want: "linux-x64"},
	} {
		if check.got != check.want {
			issues = append(
				issues,
				fmt.Sprintf("%s is %q, want %q", check.field, check.got, check.want),
			)
		}
	}
	if !safeRelativeSourcePath(report.Source) {
		issues = append(issues, "source must be a safe Tetra source path")
	}
	if !surfacePackageSourceMatchesReferenceApp(report.ReferenceApp, report.Source) {
		issues = append(
			issues,
			fmt.Sprintf(
				"reference_app %q does not match source %q",
				report.ReferenceApp,
				report.Source,
			),
		)
	}
	issues = append(issues, validateSurfaceI18nStringTables(report.StringTables)...)
	issues = append(issues, validateSurfaceI18nLocaleSelection(report.LocaleSelection)...)
	issues = append(issues, validateSurfaceI18nLookups(report.Lookups)...)
	issues = append(issues, validateSurfaceI18nFormatHooks(report.FormatHooks)...)
	issues = append(issues, validateSurfaceI18nTextDirection(report.TextDirection)...)
	issues = append(issues, validateSurfaceI18nLocalizedForm(report.LocalizedForm)...)
	issues = append(issues, validateSurfaceI18nNegativeGuards(report.NegativeGuards)...)
	if !report.Pass {
		issues = append(issues, "pass must be true")
	}
	return issues
}

func validateSurfaceI18nStringTables(tables []SurfaceI18nStringTable) []string {
	if len(tables) < 2 {
		return []string{"string_tables require en-US and uk-UA"}
	}
	var issues []string
	seen := map[string]SurfaceI18nStringTable{}
	for _, table := range tables {
		locale := strings.TrimSpace(table.Locale)
		seen[locale] = table
		prefix := "string table " + locale
		if locale == "" {
			issues = append(issues, "string table locale is required")
		}
		if table.EntryCount <= 0 {
			issues = append(issues, prefix+" entry_count must be positive")
		}
		if !validChecksumLike(table.Checksum) {
			issues = append(issues, prefix+" checksum must be sha256 evidence")
		}
		if !table.Pass {
			issues = append(issues, prefix+" pass must be true")
		}
	}
	en, ok := seen["en-US"]
	if !ok {
		issues = append(issues, "string_tables missing en-US")
	} else if !en.Primary || en.Fallback || en.EntryCount < 5 {
		issues = append(
			issues,
			"string table en-US must be primary with at least five entries and not fallback",
		)
	}
	uk, ok := seen["uk-UA"]
	if !ok {
		issues = append(issues, "string_tables missing uk-UA")
	} else if uk.Primary || !uk.Fallback || uk.EntryCount < 4 {
		issues = append(issues, "string table uk-UA must be fallback-aware with at least four entries")
	}
	return issues
}

func validateSurfaceI18nLocaleSelection(selection SurfaceI18nLocaleSelection) []string {
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "requested_locale", got: selection.RequestedLocale, want: "uk-UA"},
		{field: "selected_locale", got: selection.SelectedLocale, want: "uk-UA"},
		{field: "fallback_locale", got: selection.FallbackLocale, want: "en-US"},
	} {
		if check.got != check.want {
			issues = append(
				issues,
				fmt.Sprintf(
					"locale_selection %s is %q, want %q",
					check.field,
					check.got,
					check.want,
				),
			)
		}
	}
	if !selection.FallbackUsed {
		issues = append(issues, "locale_selection fallback_used must be true")
	}
	if !selection.UnsupportedLocaleRejected {
		issues = append(issues, "locale_selection unsupported_locale_rejected must be true")
	}
	if !selection.Pass {
		issues = append(issues, "locale_selection pass must be true")
	}
	return issues
}

func validateSurfaceI18nLookups(lookups []SurfaceI18nLookup) []string {
	if len(lookups) < 3 {
		return []string{"lookups require primary, fallback, and missing_key evidence"}
	}
	var issues []string
	var primary, fallback, missing bool
	for _, lookup := range lookups {
		key := strings.TrimSpace(lookup.Key)
		prefix := "lookup " + key
		if key == "" || strings.TrimSpace(lookup.Locale) == "" ||
			strings.TrimSpace(lookup.ResolvedLocale) == "" {
			issues = append(issues, prefix+" key, locale, and resolved_locale are required")
		}
		if !lookup.Pass {
			issues = append(issues, prefix+" pass must be true")
		}
		switch {
		case lookup.Source == "primary" && !lookup.MissingKey && !lookup.FallbackUsed && lookup.DiagnosticCode == 0:
			primary = true
		case lookup.Source == "fallback" && !lookup.MissingKey && lookup.FallbackUsed && lookup.ResolvedLocale == "en-US" && lookup.DiagnosticCode == 0:
			fallback = true
		case lookup.Source == "missing" && lookup.MissingKey && lookup.FallbackUsed && lookup.DiagnosticCode == 2001:
			missing = true
		}
	}
	if !primary {
		issues = append(issues, "lookups missing primary locale resolution")
	}
	if !fallback {
		issues = append(issues, "lookups missing fallback locale resolution")
	}
	if !missing {
		issues = append(issues, "lookups missing missing_key diagnostic evidence")
	}
	return issues
}

func validateSurfaceI18nFormatHooks(hooks []SurfaceI18nFormatHook) []string {
	if len(hooks) < 2 {
		return []string{"format_hooks require date and number evidence"}
	}
	var issues []string
	seen := map[string]bool{}
	for _, hook := range hooks {
		kind := strings.TrimSpace(hook.Kind)
		seen[kind] = true
		if kind == "" || strings.TrimSpace(hook.Locale) == "" ||
			strings.TrimSpace(hook.Input) == "" ||
			strings.TrimSpace(hook.Output) == "" {
			issues = append(issues, "format_hooks kind, locale, input, and output are required")
		}
		if !hook.Deterministic {
			issues = append(issues, fmt.Sprintf("format_hook %s deterministic must be true", kind))
		}
		if hook.ICUClaim {
			issues = append(issues, fmt.Sprintf("format_hook %s must not claim full ICU", kind))
		}
		if !hook.Pass {
			issues = append(issues, fmt.Sprintf("format_hook %s pass must be true", kind))
		}
	}
	for _, kind := range []string{"date", "number"} {
		if !seen[kind] {
			issues = append(issues, "format_hooks missing "+kind)
		}
	}
	return issues
}

func validateSurfaceI18nTextDirection(direction SurfaceI18nTextDirection) []string {
	var issues []string
	if direction.DefaultDirection != "ltr" {
		issues = append(
			issues,
			fmt.Sprintf(
				"text_direction default_direction is %q, want ltr",
				direction.DefaultDirection,
			),
		)
	}
	if !direction.RTLPlaceholder {
		issues = append(issues, "text_direction rtl_placeholder must be true")
	}
	if direction.FullBidiSupported || direction.FullBidiClaim || direction.ShapingProof {
		issues = append(issues, "text_direction must not claim full bidi shaping without proof")
	}
	if direction.Nonclaim != "rtl-placeholder-without-full-bidi-shaping-v1" {
		issues = append(
			issues,
			fmt.Sprintf(
				"text_direction nonclaim is %q, want rtl-placeholder-without-full-bidi-shaping-v1",
				direction.Nonclaim,
			),
		)
	}
	if !direction.Pass {
		issues = append(issues, "text_direction pass must be true")
	}
	return issues
}

func validateSurfaceI18nLocalizedForm(form SurfaceI18nLocalizedForm) []string {
	var issues []string
	if form.Shape != "localized-form" {
		issues = append(
			issues,
			fmt.Sprintf("localized_form shape is %q, want localized-form", form.Shape),
		)
	}
	if !safeRelativeSourcePath(form.Source) ||
		normalizeEvidencePath(
			form.Source,
		) != "examples/surface/reference_forms/surface_reference_localized_form.tetra" {
		issues = append(
			issues,
			("localized_form source must be examples/surface/reference_forms/" +
				"surface_reference_localized_form.tetra"),
		)
	}
	for _, required := range []string{
		"lib.core.surface",
		"lib.core.block",
		"lib.core.morph",
		"lib.core.i18n",
	} {
		if !templateSmokeContainsString(form.Imports, required) {
			issues = append(issues, "localized_form imports missing "+required)
		}
	}
	for _, check := range []struct {
		field string
		ok    bool
	}{
		{field: "compiles", ok: form.Compiles},
		{field: "runs", ok: form.Runs},
		{field: "localized_strings", ok: form.LocalizedStrings},
		{field: "fallback_evidence", ok: form.FallbackEvidence},
		{field: "missing_key_diagnostic", ok: form.MissingKeyDiagnostic},
		{field: "format_hook_evidence", ok: form.FormatHookEvidence},
		{field: "resolves_to_block", ok: form.ResolvesToBlock},
		{field: "pass", ok: form.Pass},
	} {
		if !check.ok {
			issues = append(issues, "localized_form "+check.field+" must be true")
		}
	}
	if form.ExitCode != 0 {
		issues = append(
			issues,
			fmt.Sprintf("localized_form exit_code is %d, want 0", form.ExitCode),
		)
	}
	return issues
}

func validateSurfaceI18nNegativeGuards(guards SurfaceI18nNegativeGuards) []string {
	var issues []string
	for _, check := range []struct {
		field string
		ok    bool
	}{
		{field: "no_full_icu_claim", ok: guards.NoFullICUClaim},
		{field: "no_full_bidi_claim", ok: guards.NoFullBidiClaim},
		{field: "no_rtl_production_claim", ok: guards.NoRTLProductionClaim},
		{field: "no_missing_key_silent_fallback", ok: guards.NoMissingKeySilentFallback},
		{field: "no_docs_only_i18n_claim", ok: guards.NoDocsOnlyI18nClaim},
		{field: "no_react_intl_runtime", ok: guards.NoReactIntlRuntime},
		{field: "no_platform_locale_dependency", ok: guards.NoPlatformLocaleDependency},
	} {
		if !check.ok {
			issues = append(issues, "negative_guards."+check.field+" must be true")
		}
	}
	return issues
}

// ---- inspector.go ----

const InspectorSchemaV1 = "tetra.surface.inspector.v1"

type SurfaceInspectorReport struct {
	Schema          string                           `json:"schema"`
	Model           string                           `json:"model"`
	ReleaseScope    string                           `json:"release_scope"`
	Producer        string                           `json:"producer"`
	Source          string                           `json:"source"`
	Target          string                           `json:"target"`
	Mode            string                           `json:"mode"`
	InputReports    []SurfaceInspectorInputReport    `json:"input_reports"`
	SourceLocations []SurfaceInspectorSourceLocation `json:"source_locations"`
	Sections        SurfaceInspectorSections         `json:"sections"`
	MorphToPixels   *MorphToPixelsChainReport        `json:"morph_to_pixels,omitempty"`
	StaticArtifacts SurfaceInspectorStaticArtifacts  `json:"static_artifacts"`
	HiddenState     SurfaceInspectorHiddenState      `json:"hidden_state"`
	NegativeGuards  SurfaceInspectorNegativeGuards   `json:"negative_guards"`
	Pass            bool                             `json:"pass"`
}

type SurfaceInspectorInputReport struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Schema string `json:"schema"`
	Source string `json:"source"`
	Target string `json:"target"`
	Pass   bool   `json:"pass"`
}

type SurfaceInspectorSourceLocation struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

type SurfaceInspectorSections struct {
	BlockTree        SurfaceInspectorSection `json:"block_tree"`
	MorphTokens      SurfaceInspectorSection `json:"morph_tokens"`
	Layout           SurfaceInspectorSection `json:"layout"`
	Paint            SurfaceInspectorSection `json:"paint"`
	Accessibility    SurfaceInspectorSection `json:"accessibility"`
	EventRoutes      SurfaceInspectorSection `json:"event_routes"`
	Focus            SurfaceInspectorSection `json:"focus"`
	PerfCounters     SurfaceInspectorSection `json:"perf_counters"`
	RecipeExpansions SurfaceInspectorSection `json:"recipe_expansions,omitempty"`
	BlockSceneNodes  SurfaceInspectorSection `json:"block_scene_nodes,omitempty"`
	RenderCommands   SurfaceInspectorSection `json:"render_commands,omitempty"`
	FrameArtifacts   SurfaceInspectorSection `json:"frame_artifacts,omitempty"`
	GoldenDiff       SurfaceInspectorSection `json:"golden_diff,omitempty"`
}

type SurfaceInspectorSection struct {
	Present bool   `json:"present"`
	Count   int    `json:"count"`
	Source  string `json:"source"`
}

type SurfaceInspectorStaticArtifacts struct {
	JSON           string `json:"json"`
	HTML           string `json:"html"`
	HTMLToolReport bool   `json:"html_tool_report"`
}

type SurfaceInspectorHiddenState struct {
	Scanned  bool                                 `json:"scanned"`
	Findings []SurfaceInspectorHiddenStateFinding `json:"findings"`
}

type SurfaceInspectorHiddenStateFinding struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

type SurfaceInspectorNegativeGuards struct {
	NoDOMRuntimeDependency      bool `json:"no_dom_runtime_dependency"`
	NoBrowserDevtoolsDependency bool `json:"no_browser_devtools_dependency"`
	NoReactDevtoolsDependency   bool `json:"no_react_devtools_dependency"`
	StaticHTMLToolReportOnly    bool `json:"static_html_tool_report_only"`
	NoHiddenState               bool `json:"no_hidden_state"`
}

func ValidateInspectorReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != InspectorSchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, InspectorSchemaV1)
	}

	var report SurfaceInspectorReport
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}

	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: report.Schema, want: InspectorSchemaV1},
		{field: "model", got: report.Model, want: "surface-inspector-v1"},
		{field: "release_scope", got: report.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
		{field: "producer", got: report.Producer, want: "tools/cmd/surface-inspector"},
		{field: "target", got: report.Target, want: "headless"},
		{field: "mode", got: report.Mode, want: "static-tool-report"},
	} {
		if check.got != check.want {
			issues = append(
				issues,
				fmt.Sprintf("%s is %q, want %q", check.field, check.got, check.want),
			)
		}
	}
	if strings.TrimSpace(report.Source) == "" {
		issues = append(issues, "source is required")
	}
	issues = append(issues, validateSurfaceInspectorInputReports(report.InputReports)...)
	issues = append(issues, validateSurfaceInspectorSourceLocations(report.SourceLocations)...)
	issues = append(issues, validateSurfaceInspectorSections(report.Sections)...)
	issues = append(
		issues,
		validateSurfaceInspectorMorphToPixels(
			report.InputReports,
			report.Sections,
			report.MorphToPixels,
		)...)
	issues = append(issues, validateSurfaceInspectorStaticArtifacts(report.StaticArtifacts)...)
	issues = append(issues, validateSurfaceInspectorHiddenState(report.HiddenState)...)
	issues = append(issues, validateSurfaceInspectorNegativeGuards(report.NegativeGuards)...)
	if !report.Pass {
		issues = append(issues, "pass must be true")
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateSurfaceInspectorInputReports(reports []SurfaceInspectorInputReport) []string {
	if len(reports) == 0 {
		return []string{"input_reports are required"}
	}
	required := map[string]bool{
		"block":         false,
		"morph":         false,
		"accessibility": false,
		"app-model":     false,
	}
	var issues []string
	for _, report := range reports {
		kind := strings.TrimSpace(report.Kind)
		if kind == "" {
			issues = append(issues, "input_reports kind is required")
			continue
		}
		if _, ok := required[kind]; ok {
			required[kind] = true
		}
		if !safeRelativeReportPath(report.Path) {
			issues = append(issues, fmt.Sprintf("input_reports %s path is unsafe or empty", kind))
		}
		switch kind {
		case "morph-rendered-beauty":
			if report.Schema != MorphRenderedBeautyReportSchemaV1 {
				issues = append(
					issues,
					fmt.Sprintf(
						"input_reports %s schema is %q, want %q",
						kind,
						report.Schema,
						MorphRenderedBeautyReportSchemaV1,
					),
				)
			}
		default:
			if report.Schema != SchemaV1 {
				issues = append(
					issues,
					fmt.Sprintf(
						"input_reports %s schema is %q, want %q",
						kind,
						report.Schema,
						SchemaV1,
					),
				)
			}
		}
		if strings.TrimSpace(report.Source) == "" {
			issues = append(issues, fmt.Sprintf("input_reports %s source is required", kind))
		}
		if kind == "morph-rendered-beauty" {
			if !containsMorphRenderedBeautyText(
				[]string{"headless", "linux-x64-real-window", "wasm32-web-browser-canvas"},
				report.Target,
			) {
				issues = append(
					issues,
					fmt.Sprintf("input_reports %s target %q is unsupported", kind, report.Target),
				)
			}
		} else if report.Target != "headless" {
			issues = append(
				issues,
				fmt.Sprintf("input_reports %s target is %q, want headless", kind, report.Target),
			)
		}
		if !report.Pass {
			issues = append(issues, fmt.Sprintf("input_reports %s pass must be true", kind))
		}
	}
	for kind, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("input_reports missing %s", kind))
		}
	}
	return issues
}

func validateSurfaceInspectorSourceLocations(locations []SurfaceInspectorSourceLocation) []string {
	if len(locations) == 0 {
		return []string{"source_locations are required"}
	}
	required := map[string]bool{
		"block":         false,
		"morph":         false,
		"accessibility": false,
		"app-model":     false,
	}
	hasMorphRenderedBeauty := false
	var issues []string
	for _, location := range locations {
		kind := strings.TrimSpace(location.Kind)
		if kind == "" {
			issues = append(issues, "source_locations kind is required")
			continue
		}
		if _, ok := required[kind]; ok {
			required[kind] = true
		}
		if kind == "morph-rendered-beauty" {
			hasMorphRenderedBeauty = true
		}
		if !safeRelativeSourcePath(location.Path) {
			issues = append(
				issues,
				fmt.Sprintf("source_locations %s path is unsafe or not a Tetra source", kind),
			)
		}
		if location.Line <= 0 || location.Column <= 0 {
			issues = append(
				issues,
				fmt.Sprintf("source_locations %s requires positive line and column", kind),
			)
		}
	}
	for kind, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("source_locations missing %s", kind))
		}
	}
	if hasMorphRenderedBeauty {
		// The generic loop above already validates path/line/column for the
		// optional MRB source row. No extra issue is needed here.
	}
	return issues
}

func validateSurfaceInspectorSections(sections SurfaceInspectorSections) []string {
	var issues []string
	for _, check := range []struct {
		name    string
		section SurfaceInspectorSection
	}{
		{name: "block_tree", section: sections.BlockTree},
		{name: "morph_tokens", section: sections.MorphTokens},
		{name: "layout", section: sections.Layout},
		{name: "paint", section: sections.Paint},
		{name: "accessibility", section: sections.Accessibility},
		{name: "event_routes", section: sections.EventRoutes},
		{name: "focus", section: sections.Focus},
		{name: "perf_counters", section: sections.PerfCounters},
	} {
		if !check.section.Present {
			issues = append(issues, fmt.Sprintf("sections.%s present must be true", check.name))
		}
		if check.section.Count <= 0 {
			issues = append(issues, fmt.Sprintf("sections.%s count must be positive", check.name))
		}
		if strings.TrimSpace(check.section.Source) == "" {
			issues = append(issues, fmt.Sprintf("sections.%s source is required", check.name))
		}
	}
	return issues
}

func validateSurfaceInspectorMorphToPixels(
	inputs []SurfaceInspectorInputReport,
	sections SurfaceInspectorSections,
	chain *MorphToPixelsChainReport,
) []string {
	var issues []string
	expectedSource := ""
	hasMorphRenderedBeauty := false
	for _, input := range inputs {
		if input.Kind == "morph-rendered-beauty" {
			hasMorphRenderedBeauty = true
			expectedSource = input.Source
			break
		}
	}
	if !hasMorphRenderedBeauty && chain == nil {
		return nil
	}
	if hasMorphRenderedBeauty && chain == nil {
		return []string{
			"morph_to_pixels is required when morph-rendered-beauty input report is present",
		}
	}
	if chain == nil {
		return []string{"morph_to_pixels requires a morph-rendered-beauty input report"}
	}
	issues = append(
		issues,
		validateMorphToPixelsChain("morph_to_pixels", *chain, expectedSource)...)
	for _, check := range []struct {
		name    string
		section SurfaceInspectorSection
	}{
		{name: "recipe_expansions", section: sections.RecipeExpansions},
		{name: "block_scene_nodes", section: sections.BlockSceneNodes},
		{name: "render_commands", section: sections.RenderCommands},
		{name: "frame_artifacts", section: sections.FrameArtifacts},
		{name: "golden_diff", section: sections.GoldenDiff},
	} {
		if !check.section.Present {
			issues = append(
				issues,
				fmt.Sprintf(
					"sections.%s present must be true when morph_to_pixels is present",
					check.name,
				),
			)
		}
		if check.section.Count <= 0 {
			issues = append(
				issues,
				fmt.Sprintf(
					"sections.%s count must be positive when morph_to_pixels is present",
					check.name,
				),
			)
		}
		if strings.TrimSpace(check.section.Source) == "" {
			issues = append(
				issues,
				fmt.Sprintf(
					"sections.%s source is required when morph_to_pixels is present",
					check.name,
				),
			)
		}
	}
	return issues
}

func validateSurfaceInspectorStaticArtifacts(artifacts SurfaceInspectorStaticArtifacts) []string {
	var issues []string
	if !safeRelativeReportPath(artifacts.JSON) {
		issues = append(issues, "static_artifacts.json is unsafe or empty")
	}
	if strings.TrimSpace(artifacts.HTML) != "" && !safeRelativeReportPath(artifacts.HTML) {
		issues = append(issues, "static_artifacts.html is unsafe")
	}
	if strings.TrimSpace(artifacts.HTML) != "" && !artifacts.HTMLToolReport {
		issues = append(
			issues,
			"static_artifacts.html_tool_report must be true when html is present",
		)
	}
	return issues
}

func validateSurfaceInspectorHiddenState(hidden SurfaceInspectorHiddenState) []string {
	if !hidden.Scanned {
		return []string{"hidden_state.scanned must be true"}
	}
	if len(hidden.Findings) != 0 {
		return []string{"hidden_state findings must be empty"}
	}
	return nil
}

func validateSurfaceInspectorNegativeGuards(guards SurfaceInspectorNegativeGuards) []string {
	if guards.NoDOMRuntimeDependency &&
		guards.NoBrowserDevtoolsDependency &&
		guards.NoReactDevtoolsDependency &&
		guards.StaticHTMLToolReportOnly &&
		guards.NoHiddenState {
		return nil
	}
	return []string{
		("negative_guards must reject DOM runtime, browser devtools, " +
			"React devtools, non-static HTML reports, and hidden state"),
	}
}

func safeRelativeReportPath(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" || filepath.IsAbs(path) {
		return false
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return false
	}
	return true
}

func safeRelativeSourcePath(path string) bool {
	if !safeRelativeReportPath(path) {
		return false
	}
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".tetra" || ext == ".t4"
}

// ---- morph_rendered_beauty_validation.go ----

const (
	MorphRenderedBeautyContractSchemaV1 = "tetra.surface.morph-rendered-beauty.contract.v1"
	MorphRenderedBeautyReportSchemaV1   = "tetra.surface.morph-rendered-beauty.v1"
	MorphRenderedBeautyScope            = "surface-morph-rendered-beauty-linux-web"
)

type MorphRenderedBeautyContract struct {
	Schema                  string                            `json:"schema"`
	Status                  string                            `json:"status"`
	ReportSchema            string                            `json:"report_schema"`
	SurfaceScope            string                            `json:"surface_scope"`
	Pipeline                []string                          `json:"pipeline"`
	CorePrimitives          []string                          `json:"core_primitives"`
	ForbiddenCorePrimitives []string                          `json:"forbidden_core_primitives"`
	SupportedTargets        []string                          `json:"supported_targets"`
	UnsupportedTargets      []string                          `json:"unsupported_targets"`
	RequiredEvidence        []string                          `json:"required_evidence"`
	NegativeGuards          MorphRenderedBeautyNegativeGuards `json:"negative_guards"`
	NonClaims               []string                          `json:"nonclaims"`
}

type MorphRenderedBeautyNegativeGuards struct {
	MetadataOnlyRejected             bool `json:"metadata_only_rejected"`
	SelfGoldenRejected               bool `json:"self_golden_rejected"`
	PrecomputedFrameRejected         bool `json:"precomputed_frame_rejected"`
	MissingFrameArtifactRejected     bool `json:"missing_frame_artifact_rejected"`
	NoDOMUI                          bool `json:"no_dom_ui"`
	NoCSSRuntime                     bool `json:"no_css_runtime"`
	NoReactRuntime                   bool `json:"no_react_runtime"`
	NoElectronRuntime                bool `json:"no_electron_runtime"`
	NoNativeWidgets                  bool `json:"no_native_widgets"`
	NoHiddenAppState                 bool `json:"no_hidden_app_state"`
	NonBlockOutputRejected           bool `json:"non_block_output_rejected"`
	DirtyCheckoutProductionRejected  bool `json:"dirty_checkout_production_rejected"`
	UnsupportedTargetRejected        bool `json:"unsupported_target_rejected"`
	RendererOwnedStableProofRequired bool `json:"renderer_owned_stable_proof_required"`
}

type MorphRenderedBeautyReport struct {
	Schema              string                                 `json:"schema"`
	Status              string                                 `json:"status"`
	SurfaceScope        string                                 `json:"surface_scope"`
	Target              string                                 `json:"target"`
	ScenarioName        string                                 `json:"scenario_name"`
	GitHead             string                                 `json:"git_head"`
	GitCommit           string                                 `json:"git_commit"`
	GitDirty            bool                                   `json:"git_dirty"`
	ProductClaim        bool                                   `json:"product_claim"`
	FinalSignoff        bool                                   `json:"final_signoff"`
	CorePrimitives      []string                               `json:"core_primitives"`
	MorphEvidence       MorphRenderedBeautyMorphEvidence       `json:"morph_evidence"`
	BlockSceneSnapshot  MorphRenderedBeautyBlockSceneSnapshot  `json:"block_scene_snapshot"`
	RenderEvidence      MorphRenderedBeautyRenderEvidence      `json:"render_evidence"`
	RendererStableProof MorphRenderedBeautyRendererStableProof `json:"renderer_stable_proof"`
	RenderCommandStream MorphRenderedBeautyRenderCommandStream `json:"render_command_stream"`
	PixelEvidence       MorphRenderedBeautyPixelEvidence       `json:"pixel_evidence"`
	NegativeGuards      MorphRenderedBeautyNegativeGuards      `json:"negative_guards"`
	NonClaims           []string                               `json:"nonclaims"`
}

type MorphRenderedBeautyMorphEvidence struct {
	Source                 string   `json:"source"`
	SourceSHA256           string   `json:"source_sha256"`
	CapsuleHash            string   `json:"capsule_hash"`
	TokenGraphHash         string   `json:"token_graph_hash"`
	TokenCount             int      `json:"token_count"`
	TokenCategories        []string `json:"token_categories"`
	RecipeCount            int      `json:"recipe_count"`
	RecipeExpansionCount   int      `json:"recipe_expansion_count"`
	RecipeNames            []string `json:"recipe_names"`
	ResolvedMorphSceneHash string   `json:"resolved_morph_scene_hash"`
	BlockSceneSnapshotHash string   `json:"block_scene_snapshot_hash"`
}

type MorphRenderedBeautyBlockSceneSnapshot struct {
	Schema               string                                    `json:"schema"`
	SurfaceScope         string                                    `json:"surface_scope"`
	Source               string                                    `json:"source"`
	QualityLevel         string                                    `json:"quality_level"`
	CorePrimitives       []string                                  `json:"core_primitives"`
	CompactPropsOnly     bool                                      `json:"compact_props_only"`
	RecipeExpansionCount int                                       `json:"recipe_expansion_count"`
	NodeCount            int                                       `json:"node_count"`
	RichSpecHash         string                                    `json:"rich_spec_hash"`
	BlockSceneHash       string                                    `json:"block_scene_hash"`
	SpecCoverage         MorphRenderedBeautyBlockSceneSpecCoverage `json:"spec_coverage"`
}

type MorphRenderedBeautyBlockSceneSpecCoverage struct {
	Layout        bool `json:"layout"`
	Paint         bool `json:"paint"`
	Text          bool `json:"text"`
	Image         bool `json:"image"`
	Input         bool `json:"input"`
	Event         bool `json:"event"`
	State         bool `json:"state"`
	Motion        bool `json:"motion"`
	Accessibility bool `json:"accessibility"`
}

type MorphRenderedBeautyRenderEvidence struct {
	CommandStreamHash string `json:"command_stream_hash"`
	CommandCount      int    `json:"command_count"`
	Renderer          string `json:"renderer"`
}

type MorphRenderedBeautyRendererStableProof struct {
	Schema                         string `json:"schema"`
	PixelOwner                     string `json:"pixel_owner"`
	RendererOwned                  bool   `json:"renderer_owned"`
	BridgeOwnedPixels              bool   `json:"bridge_owned_pixels"`
	BlockFirst                     bool   `json:"block_first"`
	DerivedFromRenderCommandStream bool   `json:"derived_from_render_command_stream"`
	RenderCommandStreamHash        string `json:"render_command_stream_hash"`
	BlockSceneHash                 string `json:"block_scene_hash"`
	FrameChecksum                  string `json:"frame_checksum"`
	StablePromotionEligible        bool   `json:"stable_promotion_eligible"`
}

type MorphRenderedBeautyRenderCommandStream struct {
	Schema                        string                             `json:"schema"`
	Source                        string                             `json:"source"`
	SurfaceScope                  string                             `json:"surface_scope"`
	Producer                      string                             `json:"producer"`
	QualityLevel                  string                             `json:"quality_level"`
	Renderer                      string                             `json:"renderer"`
	DerivedFromBlockSceneSnapshot bool                               `json:"derived_from_block_scene_snapshot"`
	BlockSceneHash                string                             `json:"block_scene_hash"`
	FrameChecksum                 string                             `json:"frame_checksum"`
	CommandStreamHash             string                             `json:"command_stream_hash"`
	CommandCount                  int                                `json:"command_count"`
	SourceLinked                  bool                               `json:"source_linked"`
	HandcraftedFixture            bool                               `json:"handcrafted_fixture"`
	Commands                      []MorphRenderedBeautyRenderCommand `json:"commands"`
}

type MorphRenderedBeautyRenderCommand struct {
	Order          int    `json:"order"`
	Command        string `json:"command"`
	Source         string `json:"source"`
	SourceNodeID   string `json:"source_node_id"`
	Recipe         string `json:"recipe"`
	LayerID        string `json:"layer_id"`
	BlockID        int    `json:"block_id"`
	Quality        string `json:"quality"`
	Color          string `json:"color,omitempty"`
	Width          int    `json:"width,omitempty"`
	Blur           int    `json:"blur,omitempty"`
	OffsetX        int    `json:"offset_x,omitempty"`
	OffsetY        int    `json:"offset_y,omitempty"`
	RasterFormat   string `json:"raster_format,omitempty"`
	RasterHash     string `json:"raster_hash,omitempty"`
	RasterWidth    int    `json:"raster_width,omitempty"`
	RasterHeight   int    `json:"raster_height,omitempty"`
	RasterCoverage int    `json:"raster_coverage,omitempty"`
	MarkerOnly     bool   `json:"marker_only,omitempty"`
	Checksum       string `json:"checksum"`
}

type MorphRenderedBeautyPixelEvidence struct {
	FrameArtifact           string `json:"frame_artifact"`
	FrameArtifactSHA256     string `json:"frame_artifact_sha256"`
	FrameChecksum           string `json:"frame_checksum"`
	FrameProducer           string `json:"frame_producer"`
	AppSource               string `json:"app_source"`
	MorphRecipeHash         string `json:"morph_recipe_hash"`
	BlockSceneHash          string `json:"block_scene_hash"`
	RenderCommandStreamHash string `json:"render_command_stream_hash"`
	GoldenArtifact          string `json:"golden_artifact"`
	GoldenArtifactSHA256    string `json:"golden_artifact_sha256"`
	GoldenChecksum          string `json:"golden_checksum"`
	DiffPixels              int    `json:"diff_pixels"`
	DiffRatioMilli          int    `json:"diff_ratio_milli"`
	MaxChannelDelta         int    `json:"max_channel_delta"`
	PrecomputedFixtureFrame bool   `json:"precomputed_fixture_frame"`
}

func ValidateMorphRenderedBeautyContract(raw []byte) error {
	var contract MorphRenderedBeautyContract
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&contract); err != nil {
		return err
	}
	return ValidateMorphRenderedBeautyContractValue(contract)
}

func ValidateMorphRenderedBeautyReport(raw []byte) error {
	var report MorphRenderedBeautyReport
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		return err
	}
	return ValidateMorphRenderedBeautyReportValue(report)
}

func ValidateMorphRenderedBeautyContractFile(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return ValidateMorphRenderedBeautyContract(raw)
}

func ValidateMorphRenderedBeautyReportFile(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return ValidateMorphRenderedBeautyReport(raw)
}

func ValidateMorphRenderedBeautyContractValue(contract MorphRenderedBeautyContract) error {
	var issues []string
	if contract.Schema != MorphRenderedBeautyContractSchemaV1 {
		issues = append(
			issues,
			fmt.Sprintf(
				"schema is %q, want %s",
				contract.Schema,
				MorphRenderedBeautyContractSchemaV1,
			),
		)
	}
	if contract.Status != "experimental-contract" {
		issues = append(
			issues,
			fmt.Sprintf("status is %q, want experimental-contract", contract.Status),
		)
	}
	if contract.ReportSchema != MorphRenderedBeautyReportSchemaV1 {
		issues = append(
			issues,
			fmt.Sprintf(
				"report_schema is %q, want %s",
				contract.ReportSchema,
				MorphRenderedBeautyReportSchemaV1,
			),
		)
	}
	if contract.SurfaceScope != MorphRenderedBeautyScope {
		issues = append(
			issues,
			fmt.Sprintf(
				"surface_scope is %q, want %s",
				contract.SurfaceScope,
				MorphRenderedBeautyScope,
			),
		)
	}
	issues = append(issues, requireTextSequence("pipeline", contract.Pipeline, []string{
		"morph_source",
		"token_graph",
		"recipe_expansions",
		"resolved_morph_scene",
		"block_scene_snapshot",
		"render_command_stream",
		"frame_artifact",
		"pixel_golden_comparison",
		"product_claim_gate",
	})...)
	issues = append(
		issues,
		validateMorphRenderedBeautyCorePrimitives(
			contract.CorePrimitives,
			contract.ForbiddenCorePrimitives,
		)...)
	issues = append(
		issues,
		requireTextSet(
			"supported_targets",
			contract.SupportedTargets,
			[]string{"headless", "linux-x64-real-window", "wasm32-web-browser-canvas"},
		)...)
	issues = append(
		issues,
		requireTextSet(
			"unsupported_targets",
			contract.UnsupportedTargets,
			[]string{"macos", "windows", "wasm32-wasi"},
		)...)
	issues = append(issues, requireTextSet("required_evidence", contract.RequiredEvidence, []string{
		"morph_source_hash",
		"token_graph_hash",
		"token_coverage",
		"recipe_coverage",
		"recipe_expansions",
		"resolved_morph_scene_hash",
		"block_scene_snapshot_hash",
		"block_scene_snapshot_rich_specs",
		"render_command_stream_hash",
		"source_linked_render_command_stream",
		"text_icon_raster_evidence",
		"app_produced_frame",
		"morph_recipe_hash",
		"pixel_block_scene_hash",
		"pixel_render_command_stream_hash",
		"frame_artifact_sha256",
		"golden_artifact_sha256",
		"pixel_diff_metrics",
		"renderer_owned_stable_proof",
		"target_and_scenario_name",
		"same_commit_git_head",
		"same_commit_git_commit",
	})...)
	issues = append(issues, validateMorphRenderedBeautyNegativeGuards(contract.NegativeGuards)...)
	issues = append(issues, requireTextSet("nonclaims", contract.NonClaims, []string{
		"no Electron runtime claim",
		"no React runtime claim",
		"no CSS runtime claim",
		"no DOM-authored UI claim",
		"no GPU renderer production claim",
		"no macOS production claim",
		"no Windows production claim",
	})...)
	return combineMorphRenderedBeautyIssues(issues)
}

func ValidateMorphRenderedBeautyReportValue(report MorphRenderedBeautyReport) error {
	var issues []string
	if report.Schema != MorphRenderedBeautyReportSchemaV1 {
		issues = append(
			issues,
			fmt.Sprintf("schema is %q, want %s", report.Schema, MorphRenderedBeautyReportSchemaV1),
		)
	}
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if report.SurfaceScope != MorphRenderedBeautyScope {
		issues = append(
			issues,
			fmt.Sprintf(
				"surface_scope is %q, want %s",
				report.SurfaceScope,
				MorphRenderedBeautyScope,
			),
		)
	}
	if !containsMorphRenderedBeautyText(
		[]string{"headless", "linux-x64-real-window", "wasm32-web-browser-canvas"},
		report.Target,
	) {
		issues = append(
			issues,
			fmt.Sprintf(
				"target %q is not supported for Morph rendered beauty evidence",
				report.Target,
			),
		)
	}
	if strings.TrimSpace(report.ScenarioName) == "" {
		issues = append(issues, "scenario_name is required")
	}
	if !isMorphRenderedBeautyGitHead(report.GitHead) {
		issues = append(issues, "git_head must be 40 hex characters")
	}
	if !isMorphRenderedBeautyGitHead(report.GitCommit) {
		issues = append(issues, "git_commit must be 40 hex characters")
	}
	if isMorphRenderedBeautyGitHead(report.GitHead) &&
		isMorphRenderedBeautyGitHead(report.GitCommit) &&
		report.GitHead != report.GitCommit {
		issues = append(issues, "git_commit must match git_head")
	}
	if report.ProductClaim && report.GitDirty {
		issues = append(issues, "dirty checkout production claim rejected")
	}
	if report.FinalSignoff && !report.ProductClaim {
		issues = append(issues, "final_signoff requires product_claim")
	}
	issues = append(
		issues,
		validateMorphRenderedBeautyCorePrimitives(
			report.CorePrimitives,
			[]string{"Button", "Card", "TextField", "TextBox", "Sidebar", "Modal"},
		)...)
	issues = append(issues, validateMorphRenderedBeautyMorphEvidence(report.MorphEvidence)...)
	issues = append(
		issues,
		validateMorphRenderedBeautyBlockSceneSnapshot(
			report.BlockSceneSnapshot,
			report.MorphEvidence,
		)...)
	issues = append(
		issues,
		validateMorphRenderedBeautyRenderEvidence(
			report.RenderEvidence,
			report.RenderCommandStream,
		)...)
	issues = append(
		issues,
		validateMorphRenderedBeautyRenderCommandStream(
			report.RenderCommandStream,
			report.BlockSceneSnapshot,
			report.MorphEvidence,
		)...)
	issues = append(
		issues,
		validateMorphRenderedBeautyRendererStableProof(report.RendererStableProof, report)...)
	issues = append(
		issues,
		validateMorphRenderedBeautyPixelEvidence(report.PixelEvidence, report)...)
	issues = append(issues, validateMorphRenderedBeautyNegativeGuards(report.NegativeGuards)...)
	issues = append(issues, requireTextSet("nonclaims", report.NonClaims, []string{
		"no Electron runtime claim",
		"no React runtime claim",
		"no CSS runtime claim",
		"no DOM-authored UI claim",
		"no GPU renderer production claim",
		"no macOS production claim",
		"no Windows production claim",
	})...)
	return combineMorphRenderedBeautyIssues(issues)
}

func validateMorphRenderedBeautyCorePrimitives(core []string, forbidden []string) []string {
	var issues []string
	if !containsMorphRenderedBeautyText(core, "Block") {
		issues = append(issues, "core_primitives must include Block")
	}
	for _, primitive := range []string{"Button", "Card", "TextField", "TextBox", "Sidebar", "Modal"} {
		if !containsMorphRenderedBeautyText(forbidden, primitive) {
			issues = append(issues, fmt.Sprintf("forbidden_core_primitives missing %s", primitive))
		}
		if containsMorphRenderedBeautyText(core, primitive) {
			issues = append(issues, fmt.Sprintf("core_primitives must not include %s", primitive))
		}
	}
	return issues
}

func validateMorphRenderedBeautyMorphEvidence(e MorphRenderedBeautyMorphEvidence) []string {
	var issues []string
	if strings.TrimSpace(e.Source) == "" {
		issues = append(issues, "morph_evidence.source is required")
	}
	for _, check := range []struct {
		name  string
		value string
	}{
		{"morph_evidence.source_sha256", e.SourceSHA256},
		{"morph_evidence.capsule_hash", e.CapsuleHash},
		{"morph_evidence.token_graph_hash", e.TokenGraphHash},
		{"morph_evidence.resolved_morph_scene_hash", e.ResolvedMorphSceneHash},
		{"morph_evidence.block_scene_snapshot_hash", e.BlockSceneSnapshotHash},
	} {
		if !validMorphRenderedBeautySHA256(check.value) {
			issues = append(issues, check.name+" must be sha256 evidence")
		}
	}
	if e.RecipeExpansionCount <= 0 {
		issues = append(issues, "morph_evidence.recipe_expansion_count must be positive")
	}
	if e.TokenCount <= 0 {
		issues = append(issues, "morph_evidence.token_count must be positive")
	}
	issues = append(
		issues,
		requireTextSet(
			"morph_evidence.token_categories",
			e.TokenCategories,
			[]string{"color", "space", "radius", "typography", "motion", "assets"},
		)...)
	if e.RecipeCount <= 0 {
		issues = append(issues, "morph_evidence.recipe_count must be positive")
	}
	if len(e.RecipeNames) == 0 {
		issues = append(issues, "morph_evidence.recipe_names coverage is required")
	}
	if e.RecipeCount > 0 && len(e.RecipeNames) != e.RecipeCount {
		issues = append(
			issues,
			fmt.Sprintf(
				"morph_evidence.recipe_count = %d, want len(recipe_names) %d",
				e.RecipeCount,
				len(e.RecipeNames),
			),
		)
	}
	if e.RecipeExpansionCount > 0 && e.RecipeCount > 0 && e.RecipeExpansionCount < e.RecipeCount {
		issues = append(
			issues,
			"morph_evidence.recipe_expansion_count must cover every reported recipe",
		)
	}
	return issues
}

func validateMorphRenderedBeautyBlockSceneSnapshot(
	snapshot MorphRenderedBeautyBlockSceneSnapshot,
	morph MorphRenderedBeautyMorphEvidence,
) []string {
	var issues []string
	if snapshot.Schema != "tetra.surface.block-scene-snapshot.v1" {
		issues = append(
			issues,
			fmt.Sprintf(
				"block_scene_snapshot.schema is %q, want tetra.surface.block-scene-snapshot.v1",
				snapshot.Schema,
			),
		)
	}
	if snapshot.SurfaceScope != MorphRenderedBeautyScope {
		issues = append(
			issues,
			fmt.Sprintf(
				"block_scene_snapshot.surface_scope is %q, want %s",
				snapshot.SurfaceScope,
				MorphRenderedBeautyScope,
			),
		)
	}
	if strings.TrimSpace(snapshot.Source) == "" {
		issues = append(issues, "block_scene_snapshot.source is required")
	}
	if strings.TrimSpace(morph.Source) != "" &&
		strings.TrimSpace(snapshot.Source) != strings.TrimSpace(morph.Source) {
		issues = append(issues, "block_scene_snapshot.source must match morph_evidence.source")
	}
	if snapshot.QualityLevel != "rich-renderable-block-scene-v1" {
		issues = append(
			issues,
			fmt.Sprintf(
				"block_scene_snapshot.quality_level is %q, want rich-renderable-block-scene-v1",
				snapshot.QualityLevel,
			),
		)
	}
	if len(snapshot.CorePrimitives) != 1 ||
		!containsMorphRenderedBeautyText(snapshot.CorePrimitives, "Block") {
		issues = append(issues, "block_scene_snapshot.core_primitives must contain only Block")
	}
	for _, primitive := range snapshot.CorePrimitives {
		primitive = strings.TrimSpace(primitive)
		if !strings.EqualFold(primitive, "Block") {
			issues = append(
				issues,
				fmt.Sprintf("block_scene_snapshot.core_primitives must not include %s", primitive),
			)
		}
	}
	if snapshot.CompactPropsOnly {
		issues = append(issues, "block_scene_snapshot compact_props_only must be false")
	}
	if snapshot.RecipeExpansionCount <= 0 {
		issues = append(issues, "block_scene_snapshot.recipe_expansion_count must be positive")
	}
	if snapshot.NodeCount <= 0 {
		issues = append(issues, "block_scene_snapshot.node_count must be positive")
	}
	if !validMorphRenderedBeautySHA256(snapshot.RichSpecHash) {
		issues = append(issues, "block_scene_snapshot.rich_spec_hash must be sha256 evidence")
	}
	if !validMorphRenderedBeautySHA256(snapshot.BlockSceneHash) {
		issues = append(issues, "block_scene_snapshot.block_scene_hash must be sha256 evidence")
	}
	if validMorphRenderedBeautySHA256(snapshot.BlockSceneHash) &&
		snapshot.BlockSceneHash != morph.BlockSceneSnapshotHash {
		issues = append(
			issues,
			"block_scene_snapshot.block_scene_hash must match morph_evidence.block_scene_snapshot_hash",
		)
	}
	for _, check := range []struct {
		name string
		ok   bool
	}{
		{"layout", snapshot.SpecCoverage.Layout},
		{"paint", snapshot.SpecCoverage.Paint},
		{"text", snapshot.SpecCoverage.Text},
		{"image", snapshot.SpecCoverage.Image},
		{"input", snapshot.SpecCoverage.Input},
		{"event", snapshot.SpecCoverage.Event},
		{"state", snapshot.SpecCoverage.State},
		{"motion", snapshot.SpecCoverage.Motion},
		{"accessibility", snapshot.SpecCoverage.Accessibility},
	} {
		if !check.ok {
			issues = append(
				issues,
				fmt.Sprintf("block_scene_snapshot spec_coverage missing %s", check.name),
			)
		}
	}
	return issues
}

func validateMorphRenderedBeautyRenderEvidence(
	e MorphRenderedBeautyRenderEvidence,
	stream MorphRenderedBeautyRenderCommandStream,
) []string {
	var issues []string
	if !validMorphRenderedBeautySHA256(e.CommandStreamHash) {
		issues = append(issues, "render_evidence.command_stream_hash must be sha256 evidence")
	}
	if e.CommandCount <= 0 {
		issues = append(issues, "render_evidence.command_count must be positive")
	}
	if !containsMorphRenderedBeautyText(
		[]string{"software-rgba-headless", "wayland-shm-rgba", "browser-canvas-rgba"},
		e.Renderer,
	) {
		issues = append(
			issues,
			fmt.Sprintf("render_evidence.renderer %q is not allowed", e.Renderer),
		)
	}
	if strings.TrimSpace(stream.CommandStreamHash) != "" &&
		e.CommandStreamHash != stream.CommandStreamHash {
		issues = append(
			issues,
			"render_evidence.command_stream_hash must match render_command_stream.command_stream_hash",
		)
	}
	if stream.CommandCount != 0 && e.CommandCount != stream.CommandCount {
		issues = append(
			issues,
			"render_evidence.command_count must match render_command_stream.command_count",
		)
	}
	if strings.TrimSpace(stream.Renderer) != "" && e.Renderer != stream.Renderer {
		issues = append(
			issues,
			"render_evidence.renderer must match render_command_stream.renderer",
		)
	}
	return issues
}

func validateMorphRenderedBeautyRenderCommandStream(
	stream MorphRenderedBeautyRenderCommandStream,
	snapshot MorphRenderedBeautyBlockSceneSnapshot,
	morph MorphRenderedBeautyMorphEvidence,
) []string {
	var issues []string
	if stream.Schema != "tetra.surface.render-command-stream.v1" {
		issues = append(
			issues,
			fmt.Sprintf(
				"render_command_stream.schema is %q, want tetra.surface.render-command-stream.v1",
				stream.Schema,
			),
		)
	}
	if strings.TrimSpace(stream.Source) == "" {
		issues = append(issues, "render_command_stream.source is required")
	}
	if strings.TrimSpace(morph.Source) != "" &&
		strings.TrimSpace(stream.Source) != strings.TrimSpace(morph.Source) {
		issues = append(issues, "render_command_stream.source must match morph_evidence.source")
	}
	if strings.TrimSpace(snapshot.Source) != "" &&
		strings.TrimSpace(stream.Source) != strings.TrimSpace(snapshot.Source) {
		issues = append(
			issues,
			"render_command_stream.source must match block_scene_snapshot.source",
		)
	}
	if stream.SurfaceScope != MorphRenderedBeautyScope {
		issues = append(
			issues,
			fmt.Sprintf(
				"render_command_stream.surface_scope is %q, want %s",
				stream.SurfaceScope,
				MorphRenderedBeautyScope,
			),
		)
	}
	if strings.TrimSpace(stream.Producer) == "" {
		issues = append(issues, "render_command_stream.producer is required")
	}
	if stream.QualityLevel != "deterministic-render-command-stream-v1" {
		issues = append(
			issues,
			fmt.Sprintf(
				"render_command_stream.quality_level is %q, want deterministic-render-command-stream-v1",
				stream.QualityLevel,
			),
		)
	}
	if !containsMorphRenderedBeautyText(
		[]string{"software-rgba-headless", "wayland-shm-rgba", "browser-canvas-rgba"},
		stream.Renderer,
	) {
		issues = append(
			issues,
			fmt.Sprintf("render_command_stream.renderer %q is not allowed", stream.Renderer),
		)
	}
	if !stream.DerivedFromBlockSceneSnapshot {
		issues = append(
			issues,
			"render_command_stream.derived_from_block_scene_snapshot must be true",
		)
	}
	if !validMorphRenderedBeautySHA256(stream.BlockSceneHash) {
		issues = append(issues, "render_command_stream.block_scene_hash must be sha256 evidence")
	}
	if validMorphRenderedBeautySHA256(snapshot.BlockSceneHash) &&
		stream.BlockSceneHash != snapshot.BlockSceneHash {
		issues = append(
			issues,
			"render_command_stream.block_scene_hash must match block_scene_snapshot.block_scene_hash",
		)
	}
	if validMorphRenderedBeautySHA256(morph.BlockSceneSnapshotHash) &&
		stream.BlockSceneHash != morph.BlockSceneSnapshotHash {
		issues = append(
			issues,
			"render_command_stream.block_scene_hash must match morph_evidence.block_scene_snapshot_hash",
		)
	}
	if !validMorphRenderedBeautySHA256(stream.FrameChecksum) {
		issues = append(issues, "render_command_stream.frame_checksum must be sha256 evidence")
	}
	if !validMorphRenderedBeautySHA256(stream.CommandStreamHash) {
		issues = append(issues, "render_command_stream.command_stream_hash must be sha256 evidence")
	}
	if stream.CommandCount <= 0 {
		issues = append(issues, "render_command_stream.command_count must be positive")
	}
	if stream.CommandCount != len(stream.Commands) {
		issues = append(
			issues,
			fmt.Sprintf(
				"render_command_stream.command_count = %d, want len(commands) %d",
				stream.CommandCount,
				len(stream.Commands),
			),
		)
	}
	if !stream.SourceLinked {
		issues = append(issues, "render_command_stream.source_linked must be true")
	}
	if stream.HandcraftedFixture {
		issues = append(issues, "render_command_stream.handcrafted_fixture must be false")
	}
	issues = append(
		issues,
		validateMorphRenderedBeautyRenderCommands(stream.Commands, morph.Source)...)
	return issues
}

func validateMorphRenderedBeautyRendererStableProof(
	proof MorphRenderedBeautyRendererStableProof,
	report MorphRenderedBeautyReport,
) []string {
	var issues []string
	if proof.Schema != "tetra.surface.renderer-stable-proof.v1" {
		issues = append(
			issues,
			fmt.Sprintf(
				"renderer_stable_proof.schema is %q, want tetra.surface.renderer-stable-proof.v1",
				proof.Schema,
			),
		)
	}
	if !containsMorphRenderedBeautyText(
		[]string{"surface-renderer", "morph-evidence-bridge"},
		proof.PixelOwner,
	) {
		issues = append(
			issues,
			fmt.Sprintf(
				"renderer_stable_proof.pixel_owner is %q, want surface-renderer or morph-evidence-bridge",
				proof.PixelOwner,
			),
		)
	}
	for _, check := range []struct {
		name  string
		value string
	}{
		{"renderer_stable_proof.render_command_stream_hash", proof.RenderCommandStreamHash},
		{"renderer_stable_proof.block_scene_hash", proof.BlockSceneHash},
		{"renderer_stable_proof.frame_checksum", proof.FrameChecksum},
	} {
		if !validMorphRenderedBeautySHA256(check.value) {
			issues = append(issues, check.name+" must be sha256 evidence")
		}
	}
	if validMorphRenderedBeautySHA256(proof.RenderCommandStreamHash) &&
		validMorphRenderedBeautySHA256(report.RenderCommandStream.CommandStreamHash) &&
		proof.RenderCommandStreamHash != report.RenderCommandStream.CommandStreamHash {
		issues = append(
			issues,
			("renderer_stable_proof.render_command_stream_hash must match " +
				"render_command_stream.command_stream_hash"),
		)
	}
	if validMorphRenderedBeautySHA256(proof.BlockSceneHash) &&
		validMorphRenderedBeautySHA256(report.BlockSceneSnapshot.BlockSceneHash) &&
		proof.BlockSceneHash != report.BlockSceneSnapshot.BlockSceneHash {
		issues = append(
			issues,
			"renderer_stable_proof.block_scene_hash must match block_scene_snapshot.block_scene_hash",
		)
	}
	if validMorphRenderedBeautySHA256(proof.FrameChecksum) &&
		validMorphRenderedBeautySHA256(report.RenderCommandStream.FrameChecksum) &&
		proof.FrameChecksum != report.RenderCommandStream.FrameChecksum {
		issues = append(
			issues,
			"renderer_stable_proof.frame_checksum must match render_command_stream.frame_checksum",
		)
	}
	if proof.RendererOwned && proof.BridgeOwnedPixels {
		issues = append(
			issues,
			"renderer_stable_proof cannot be both renderer_owned and bridge_owned_pixels",
		)
	}
	if proof.StablePromotionEligible &&
		(!proof.RendererOwned || proof.BridgeOwnedPixels || !proof.BlockFirst || !proof.DerivedFromRenderCommandStream || proof.PixelOwner != "surface-renderer") {
		issues = append(
			issues,
			"renderer_stable_proof.stable_promotion_eligible requires renderer-owned stable proof",
		)
	}
	if (report.ProductClaim || report.FinalSignoff) &&
		(!proof.StablePromotionEligible || !proof.RendererOwned || proof.BridgeOwnedPixels || !proof.BlockFirst || !proof.DerivedFromRenderCommandStream || proof.PixelOwner != "surface-renderer") {
		issues = append(issues, "product_claim requires renderer_owned stable proof")
	}
	return issues
}

func validateMorphRenderedBeautyRenderCommands(
	commands []MorphRenderedBeautyRenderCommand,
	source string,
) []string {
	var issues []string
	seenCommands := map[string]bool{}
	lastOrder := 0
	for i, command := range commands {
		name := normalizeMorphRenderedBeautyRenderCommand(command.Command)
		if command.Order != i+1 {
			issues = append(
				issues,
				fmt.Sprintf(
					"render_command_stream.commands[%d].order = %d, want %d",
					i,
					command.Order,
					i+1,
				),
			)
		}
		if command.Order <= lastOrder {
			issues = append(
				issues,
				fmt.Sprintf(
					"render_command_stream command order %d is not strictly greater than previous order %d",
					command.Order,
					lastOrder,
				),
			)
		}
		lastOrder = command.Order
		if !isMorphRenderedBeautyRenderCommand(name) {
			issues = append(
				issues,
				fmt.Sprintf(
					"render_command_stream.commands[%d].command %q is not supported",
					i,
					command.Command,
				),
			)
		}
		if strings.TrimSpace(source) != "" &&
			strings.TrimSpace(command.Source) != strings.TrimSpace(source) {
			issues = append(
				issues,
				fmt.Sprintf(
					"render_command_stream.commands[%d].source must match morph_evidence.source",
					i,
				),
			)
		}
		if strings.TrimSpace(command.SourceNodeID) == "" {
			issues = append(
				issues,
				fmt.Sprintf("render_command_stream.commands[%d].source_node_id is required", i),
			)
		}
		if strings.TrimSpace(command.Recipe) == "" {
			issues = append(
				issues,
				fmt.Sprintf("render_command_stream.commands[%d].recipe is required", i),
			)
		}
		if command.BlockID <= 0 {
			issues = append(
				issues,
				fmt.Sprintf("render_command_stream.commands[%d].block_id must be positive", i),
			)
		}
		if strings.TrimSpace(command.LayerID) == "" {
			issues = append(
				issues,
				fmt.Sprintf("render_command_stream.commands[%d].layer_id is required", i),
			)
		}
		if strings.TrimSpace(command.Quality) == "" {
			issues = append(
				issues,
				fmt.Sprintf("render_command_stream.commands[%d].quality is required", i),
			)
		}
		if name != "radius_clip" && strings.TrimSpace(command.Color) == "" {
			issues = append(
				issues,
				fmt.Sprintf(
					"render_command_stream.commands[%d].color is required for renderer-owned pixels",
					i,
				),
			)
		}
		if name == "text" {
			issues = append(issues, validateMorphRenderedBeautyRasterProof(
				fmt.Sprintf("render_command_stream.commands[%d]", i),
				"builtin-5x7-alpha-mask-v1",
				command.RasterFormat,
				command.RasterHash,
				command.RasterWidth,
				command.RasterHeight,
				command.RasterCoverage,
				command.MarkerOnly,
			)...)
		}
		if name == "icon" {
			issues = append(issues, validateMorphRenderedBeautyRasterProof(
				fmt.Sprintf("render_command_stream.commands[%d]", i),
				"builtin-icon-mask-raster-v1",
				command.RasterFormat,
				command.RasterHash,
				command.RasterWidth,
				command.RasterHeight,
				command.RasterCoverage,
				command.MarkerOnly,
			)...)
		}
		if !validMorphRenderedBeautySHA256(command.Checksum) {
			issues = append(
				issues,
				fmt.Sprintf(
					"render_command_stream.commands[%d].checksum must be sha256 evidence",
					i,
				),
			)
		}
		seenCommands[name] = true
	}
	for _, required := range []string{
		"fill",
		"gradient",
		"image_fill",
		"border",
		"radius_clip",
		"shadow",
		"overlay",
		"outline",
		"text",
		"icon",
	} {
		if !seenCommands[required] {
			issues = append(
				issues,
				fmt.Sprintf("render_command_stream.commands require %s command", required),
			)
		}
	}
	return issues
}

func validateMorphRenderedBeautyRasterProof(
	prefix string,
	want string,
	format string,
	hash string,
	width int,
	height int,
	coverage int,
	markerOnly bool,
) []string {
	var issues []string
	if markerOnly {
		issues = append(
			issues,
			fmt.Sprintf("%s.marker_only must be false for raster evidence", prefix),
		)
	}
	if strings.Contains(strings.ToLower(strings.TrimSpace(format)), "marker") {
		issues = append(issues, fmt.Sprintf("%s.raster_format must not be marker evidence", prefix))
	}
	if format != want {
		issues = append(
			issues,
			fmt.Sprintf("%s.raster_format is %q, want %s", prefix, format, want),
		)
	}
	if !validMorphRenderedBeautySHA256(hash) {
		issues = append(issues, fmt.Sprintf("%s.raster_hash must be sha256 evidence", prefix))
	}
	if width <= 0 || height <= 0 {
		issues = append(issues, fmt.Sprintf("%s raster dimensions must be positive", prefix))
	}
	if coverage <= 0 {
		issues = append(issues, fmt.Sprintf("%s.raster_coverage must be positive", prefix))
	}
	if width > 0 && height > 0 && coverage > width*height {
		issues = append(issues, fmt.Sprintf("%s.raster_coverage exceeds raster dimensions", prefix))
	}
	return issues
}

func normalizeMorphRenderedBeautyRenderCommand(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func isMorphRenderedBeautyRenderCommand(value string) bool {
	switch normalizeMorphRenderedBeautyRenderCommand(value) {
	case "fill",
		"gradient",
		"image_fill",
		"border",
		"radius_clip",
		"shadow",
		"overlay",
		"outline",
		"text",
		"icon":
		return true
	default:
		return false
	}
}

func validateMorphRenderedBeautyPixelEvidence(
	e MorphRenderedBeautyPixelEvidence,
	report MorphRenderedBeautyReport,
) []string {
	var issues []string
	if strings.TrimSpace(e.FrameArtifact) == "" {
		issues = append(issues, "pixel_evidence.frame_artifact is required")
	}
	if strings.TrimSpace(e.GoldenArtifact) == "" {
		issues = append(issues, "pixel_evidence.golden_artifact is required")
	}
	if strings.TrimSpace(e.FrameProducer) != "app" {
		issues = append(
			issues,
			fmt.Sprintf("pixel_evidence.frame_producer is %q, want app", e.FrameProducer),
		)
	}
	if strings.TrimSpace(e.AppSource) == "" {
		issues = append(issues, "pixel_evidence.app_source is required")
	} else if e.AppSource != report.MorphEvidence.Source {
		issues = append(issues, "pixel_evidence.app_source must match morph_evidence.source")
	} else if e.AppSource != report.BlockSceneSnapshot.Source {
		issues = append(issues, "pixel_evidence.app_source must match block_scene_snapshot.source")
	} else if e.AppSource != report.RenderCommandStream.Source {
		issues = append(issues, "pixel_evidence.app_source must match render_command_stream.source")
	}
	for _, check := range []struct {
		name  string
		value string
	}{
		{"pixel_evidence.frame_artifact_sha256", e.FrameArtifactSHA256},
		{"pixel_evidence.frame_checksum", e.FrameChecksum},
		{"pixel_evidence.morph_recipe_hash", e.MorphRecipeHash},
		{"pixel_evidence.block_scene_hash", e.BlockSceneHash},
		{"pixel_evidence.render_command_stream_hash", e.RenderCommandStreamHash},
		{"pixel_evidence.golden_artifact_sha256", e.GoldenArtifactSHA256},
		{"pixel_evidence.golden_checksum", e.GoldenChecksum},
	} {
		if !validMorphRenderedBeautySHA256(check.value) {
			issues = append(issues, check.name+" must be sha256 evidence")
		}
	}
	if strings.TrimSpace(e.FrameArtifact) != "" && e.FrameArtifact == e.GoldenArtifact {
		issues = append(
			issues,
			"self-golden pixel evidence rejected: frame_artifact equals golden_artifact",
		)
	}
	if validMorphRenderedBeautySHA256(e.FrameArtifactSHA256) &&
		e.FrameArtifactSHA256 == e.GoldenArtifactSHA256 {
		issues = append(
			issues,
			"self-golden pixel evidence rejected: frame artifact hash equals golden artifact hash",
		)
	}
	if validMorphRenderedBeautySHA256(e.FrameChecksum) && e.FrameChecksum == e.GoldenChecksum {
		issues = append(
			issues,
			"self-golden pixel evidence rejected: frame checksum equals golden checksum",
		)
	}
	if validMorphRenderedBeautySHA256(e.FrameChecksum) &&
		validMorphRenderedBeautySHA256(report.RenderCommandStream.FrameChecksum) &&
		e.FrameChecksum != report.RenderCommandStream.FrameChecksum {
		issues = append(
			issues,
			"pixel_evidence.frame_checksum must match render_command_stream.frame_checksum",
		)
	}
	if validMorphRenderedBeautySHA256(e.BlockSceneHash) &&
		validMorphRenderedBeautySHA256(report.BlockSceneSnapshot.BlockSceneHash) &&
		e.BlockSceneHash != report.BlockSceneSnapshot.BlockSceneHash {
		issues = append(
			issues,
			"pixel_evidence.block_scene_hash must match block_scene_snapshot.block_scene_hash",
		)
	}
	if validMorphRenderedBeautySHA256(e.BlockSceneHash) &&
		validMorphRenderedBeautySHA256(report.MorphEvidence.BlockSceneSnapshotHash) &&
		e.BlockSceneHash != report.MorphEvidence.BlockSceneSnapshotHash {
		issues = append(
			issues,
			"pixel_evidence.block_scene_hash must match morph_evidence.block_scene_snapshot_hash",
		)
	}
	if validMorphRenderedBeautySHA256(e.BlockSceneHash) &&
		validMorphRenderedBeautySHA256(report.RenderCommandStream.BlockSceneHash) &&
		e.BlockSceneHash != report.RenderCommandStream.BlockSceneHash {
		issues = append(
			issues,
			"pixel_evidence.block_scene_hash must match render_command_stream.block_scene_hash",
		)
	}
	if validMorphRenderedBeautySHA256(e.RenderCommandStreamHash) &&
		validMorphRenderedBeautySHA256(report.RenderEvidence.CommandStreamHash) &&
		e.RenderCommandStreamHash != report.RenderEvidence.CommandStreamHash {
		issues = append(
			issues,
			"pixel_evidence.render_command_stream_hash must match render_evidence.command_stream_hash",
		)
	}
	if validMorphRenderedBeautySHA256(e.RenderCommandStreamHash) &&
		validMorphRenderedBeautySHA256(report.RenderCommandStream.CommandStreamHash) &&
		e.RenderCommandStreamHash != report.RenderCommandStream.CommandStreamHash {
		issues = append(
			issues,
			"pixel_evidence.render_command_stream_hash must match render_command_stream.command_stream_hash",
		)
	}
	if e.PrecomputedFixtureFrame {
		issues = append(issues, "precomputed fixture frame cannot be product visual evidence")
	}
	if morphRenderedBeautyFixtureFrameArtifactPath(e.FrameArtifact) {
		issues = append(
			issues,
			"fixture or precomputed frame artifact cannot be product visual evidence",
		)
	}
	if e.DiffPixels < 0 || e.DiffRatioMilli < 0 || e.MaxChannelDelta < 0 {
		issues = append(issues, "pixel diff metrics must be non-negative")
	}
	return issues
}

func morphRenderedBeautyFixtureFrameArtifactPath(path string) bool {
	clean := strings.ToLower(strings.TrimSpace(path))
	clean = strings.ReplaceAll(clean, "\\", "/")
	for _, marker := range []string{
		"/fixtures/",
		"fixtures/",
		"/fixture/",
		"fixture/",
		"/testdata/",
		"testdata/",
		"precomputed",
		"synthetic",
		"renderblocksystemframesizedrgba",
	} {
		if strings.Contains(clean, marker) {
			return true
		}
	}
	return false
}

func validateMorphRenderedBeautyNegativeGuards(guards MorphRenderedBeautyNegativeGuards) []string {
	var missing []string
	checks := []struct {
		name string
		ok   bool
	}{
		{"metadata_only_rejected", guards.MetadataOnlyRejected},
		{"self_golden_rejected", guards.SelfGoldenRejected},
		{"precomputed_frame_rejected", guards.PrecomputedFrameRejected},
		{"missing_frame_artifact_rejected", guards.MissingFrameArtifactRejected},
		{"no_dom_ui", guards.NoDOMUI},
		{"no_css_runtime", guards.NoCSSRuntime},
		{"no_react_runtime", guards.NoReactRuntime},
		{"no_electron_runtime", guards.NoElectronRuntime},
		{"no_native_widgets", guards.NoNativeWidgets},
		{"no_hidden_app_state", guards.NoHiddenAppState},
		{"non_block_output_rejected", guards.NonBlockOutputRejected},
		{"dirty_checkout_production_rejected", guards.DirtyCheckoutProductionRejected},
		{"unsupported_target_rejected", guards.UnsupportedTargetRejected},
		{"renderer_owned_stable_proof_required", guards.RendererOwnedStableProofRequired},
	}
	for _, check := range checks {
		if !check.ok {
			missing = append(missing, check.name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("negative_guards missing %s", strings.Join(missing, ", "))}
}

func requireTextSequence(field string, got []string, want []string) []string {
	var issues []string
	if len(got) != len(want) {
		issues = append(issues, fmt.Sprintf("%s length is %d, want %d", field, len(got), len(want)))
	}
	for i, value := range want {
		if i >= len(got) {
			issues = append(issues, fmt.Sprintf("%s missing %s", field, value))
			continue
		}
		if strings.TrimSpace(got[i]) != value {
			issues = append(issues, fmt.Sprintf("%s[%d] is %q, want %s", field, i, got[i], value))
		}
	}
	return issues
}

func requireTextSet(field string, got []string, want []string) []string {
	var issues []string
	for _, value := range want {
		if !containsMorphRenderedBeautyText(got, value) {
			issues = append(issues, fmt.Sprintf("%s missing %s", field, value))
		}
	}
	return issues
}

func containsMorphRenderedBeautyText(values []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == want {
			return true
		}
	}
	return false
}

func validMorphRenderedBeautySHA256(value string) bool {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "sha256:") {
		return false
	}
	digest := strings.TrimPrefix(value, "sha256:")
	if len(digest) != 64 {
		return false
	}
	for _, r := range digest {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

func isMorphRenderedBeautyGitHead(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 40 {
		return false
	}
	for _, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

func combineMorphRenderedBeautyIssues(issues []string) error {
	if len(issues) == 0 {
		return nil
	}
	sort.Strings(issues)
	return errors.New(strings.Join(issues, "; "))
}

// ---- morph_to_pixels_chain.go ----

type MorphToPixelsChainReport struct {
	ChainID                 string   `json:"chain_id"`
	ReportPath              string   `json:"report_path"`
	Schema                  string   `json:"schema"`
	Status                  string   `json:"status"`
	SurfaceScope            string   `json:"surface_scope"`
	Source                  string   `json:"source"`
	SourceSHA256            string   `json:"source_sha256"`
	Target                  string   `json:"target"`
	ScenarioName            string   `json:"scenario_name"`
	GitHead                 string   `json:"git_head"`
	GitCommit               string   `json:"git_commit"`
	GitDirty                bool     `json:"git_dirty"`
	TokenGraphHash          string   `json:"token_graph_hash"`
	TokenCount              int      `json:"token_count"`
	TokenCategories         []string `json:"token_categories"`
	RecipeCount             int      `json:"recipe_count"`
	RecipeExpansionCount    int      `json:"recipe_expansion_count"`
	RecipeNames             []string `json:"recipe_names"`
	BlockSceneHash          string   `json:"block_scene_hash"`
	BlockSceneNodeCount     int      `json:"block_scene_node_count"`
	RenderCommandStreamHash string   `json:"render_command_stream_hash"`
	RenderCommandCount      int      `json:"render_command_count"`
	Renderer                string   `json:"renderer"`
	FrameArtifact           string   `json:"frame_artifact"`
	FrameArtifactSHA256     string   `json:"frame_artifact_sha256"`
	FrameChecksum           string   `json:"frame_checksum"`
	GoldenArtifact          string   `json:"golden_artifact"`
	GoldenArtifactSHA256    string   `json:"golden_artifact_sha256"`
	GoldenChecksum          string   `json:"golden_checksum"`
	DiffPixels              int      `json:"diff_pixels"`
	DiffRatioMilli          int      `json:"diff_ratio_milli"`
	MaxChannelDelta         int      `json:"max_channel_delta"`
	ProductClaim            bool     `json:"product_claim"`
	FinalSignoff            bool     `json:"final_signoff"`
	Pass                    bool     `json:"pass"`
}

func MorphToPixelsChainFromRenderedBeauty(
	reportPath string,
	report MorphRenderedBeautyReport,
) MorphToPixelsChainReport {
	chain := MorphToPixelsChainReport{
		ReportPath:              reportPath,
		Schema:                  report.Schema,
		Status:                  report.Status,
		SurfaceScope:            report.SurfaceScope,
		Source:                  report.MorphEvidence.Source,
		SourceSHA256:            report.MorphEvidence.SourceSHA256,
		Target:                  report.Target,
		ScenarioName:            report.ScenarioName,
		GitHead:                 report.GitHead,
		GitCommit:               report.GitCommit,
		GitDirty:                report.GitDirty,
		TokenGraphHash:          report.MorphEvidence.TokenGraphHash,
		TokenCount:              report.MorphEvidence.TokenCount,
		TokenCategories:         append([]string(nil), report.MorphEvidence.TokenCategories...),
		RecipeCount:             report.MorphEvidence.RecipeCount,
		RecipeExpansionCount:    report.MorphEvidence.RecipeExpansionCount,
		RecipeNames:             append([]string(nil), report.MorphEvidence.RecipeNames...),
		BlockSceneHash:          report.BlockSceneSnapshot.BlockSceneHash,
		BlockSceneNodeCount:     report.BlockSceneSnapshot.NodeCount,
		RenderCommandStreamHash: report.RenderCommandStream.CommandStreamHash,
		RenderCommandCount:      report.RenderCommandStream.CommandCount,
		Renderer:                report.RenderCommandStream.Renderer,
		FrameArtifact:           report.PixelEvidence.FrameArtifact,
		FrameArtifactSHA256:     report.PixelEvidence.FrameArtifactSHA256,
		FrameChecksum:           report.PixelEvidence.FrameChecksum,
		GoldenArtifact:          report.PixelEvidence.GoldenArtifact,
		GoldenArtifactSHA256:    report.PixelEvidence.GoldenArtifactSHA256,
		GoldenChecksum:          report.PixelEvidence.GoldenChecksum,
		DiffPixels:              report.PixelEvidence.DiffPixels,
		DiffRatioMilli:          report.PixelEvidence.DiffRatioMilli,
		MaxChannelDelta:         report.PixelEvidence.MaxChannelDelta,
		ProductClaim:            report.ProductClaim,
		FinalSignoff:            report.FinalSignoff,
		Pass:                    report.Status == "pass",
	}
	chain.ChainID = morphToPixelsChainID(chain)
	return chain
}

func ValidateMorphToPixelsChainReport(chain MorphToPixelsChainReport, expectedSource string) error {
	issues := validateMorphToPixelsChain("morph_to_pixels", chain, expectedSource)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateMorphToPixelsChain(
	field string,
	chain MorphToPixelsChainReport,
	expectedSource string,
) []string {
	var issues []string
	if strings.TrimSpace(chain.ChainID) == "" {
		issues = append(issues, field+".chain_id is required")
	}
	if strings.TrimSpace(chain.ReportPath) == "" {
		issues = append(issues, field+".report_path is required")
	}
	if chain.Schema != MorphRenderedBeautyReportSchemaV1 {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s.schema is %q, want %s",
				field,
				chain.Schema,
				MorphRenderedBeautyReportSchemaV1,
			),
		)
	}
	if chain.Status != "pass" {
		issues = append(issues, field+".status must be pass")
	}
	if chain.SurfaceScope != MorphRenderedBeautyScope {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s.surface_scope is %q, want %s",
				field,
				chain.SurfaceScope,
				MorphRenderedBeautyScope,
			),
		)
	}
	if strings.TrimSpace(chain.Source) == "" {
		issues = append(issues, field+".source is required")
	}
	if strings.TrimSpace(expectedSource) != "" &&
		normalizeEvidencePath(chain.Source) != normalizeEvidencePath(expectedSource) {
		issues = append(issues, field+".source must match the inspected Surface source")
	}
	if !validChecksumLike(chain.SourceSHA256) {
		issues = append(issues, field+".source_sha256 must be sha256 evidence")
	}
	if !containsMorphRenderedBeautyText(
		[]string{"headless", "linux-x64-real-window", "wasm32-web-browser-canvas"},
		chain.Target,
	) {
		issues = append(issues, fmt.Sprintf("%s.target %q is not supported", field, chain.Target))
	}
	if strings.TrimSpace(chain.ScenarioName) == "" {
		issues = append(issues, field+".scenario_name is required")
	}
	if !isMorphRenderedBeautyGitHead(chain.GitHead) {
		issues = append(issues, field+".git_head must be 40 hex characters")
	}
	if !isMorphRenderedBeautyGitHead(chain.GitCommit) {
		issues = append(issues, field+".git_commit must be 40 hex characters")
	}
	if isMorphRenderedBeautyGitHead(chain.GitHead) &&
		isMorphRenderedBeautyGitHead(chain.GitCommit) &&
		chain.GitHead != chain.GitCommit {
		issues = append(issues, field+".git_commit must match git_head")
	}
	for _, check := range []struct {
		name  string
		value string
	}{
		{"token_graph_hash", chain.TokenGraphHash},
		{"block_scene_hash", chain.BlockSceneHash},
		{"render_command_stream_hash", chain.RenderCommandStreamHash},
		{"frame_artifact_sha256", chain.FrameArtifactSHA256},
		{"frame_checksum", chain.FrameChecksum},
		{"golden_artifact_sha256", chain.GoldenArtifactSHA256},
		{"golden_checksum", chain.GoldenChecksum},
	} {
		if !validChecksumLike(check.value) {
			issues = append(issues, fmt.Sprintf("%s.%s must be sha256 evidence", field, check.name))
		}
	}
	if chain.TokenCount <= 0 {
		issues = append(issues, field+".token_count must be positive")
	}
	issues = append(
		issues,
		requireTextSet(
			field+".token_categories",
			chain.TokenCategories,
			[]string{"color", "space", "radius", "typography", "motion", "assets"},
		)...)
	if chain.RecipeCount <= 0 {
		issues = append(issues, field+".recipe_count must be positive")
	}
	if len(chain.RecipeNames) != chain.RecipeCount {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s.recipe_names length = %d, want recipe_count %d",
				field,
				len(chain.RecipeNames),
				chain.RecipeCount,
			),
		)
	}
	if chain.RecipeExpansionCount < chain.RecipeCount || chain.RecipeExpansionCount <= 0 {
		issues = append(issues, field+".recipe_expansion_count must cover every recipe")
	}
	if chain.BlockSceneNodeCount <= 0 {
		issues = append(issues, field+".block_scene_node_count must be positive")
	}
	if chain.RenderCommandCount <= 0 {
		issues = append(issues, field+".render_command_count must be positive")
	}
	if strings.TrimSpace(chain.Renderer) == "" {
		issues = append(issues, field+".renderer is required")
	}
	if strings.TrimSpace(chain.FrameArtifact) == "" {
		issues = append(issues, field+".frame_artifact is required")
	}
	if strings.TrimSpace(chain.GoldenArtifact) == "" {
		issues = append(issues, field+".golden_artifact is required")
	}
	if normalizeEvidencePath(chain.FrameArtifact) == normalizeEvidencePath(chain.GoldenArtifact) {
		issues = append(issues, field+" self-golden artifact rejected")
	}
	if validChecksumLike(chain.FrameArtifactSHA256) &&
		chain.FrameArtifactSHA256 == chain.GoldenArtifactSHA256 {
		issues = append(issues, field+" self-golden artifact hash rejected")
	}
	if validChecksumLike(chain.FrameChecksum) && chain.FrameChecksum == chain.GoldenChecksum {
		issues = append(issues, field+" self-golden frame checksum rejected")
	}
	if chain.DiffPixels < 0 || chain.DiffRatioMilli < 0 || chain.MaxChannelDelta < 0 {
		issues = append(issues, field+" diff metrics must be non-negative")
	}
	if chain.FinalSignoff && !chain.ProductClaim {
		issues = append(issues, field+".final_signoff requires product_claim")
	}
	if !chain.Pass {
		issues = append(issues, field+".pass must be true")
	}
	return issues
}

func morphToPixelsChainID(chain MorphToPixelsChainReport) string {
	parts := []string{
		normalizeEvidencePath(chain.Source),
		chain.SourceSHA256,
		chain.GitCommit,
		chain.TokenGraphHash,
		chain.BlockSceneHash,
		chain.RenderCommandStreamHash,
		chain.FrameChecksum,
		chain.GoldenChecksum,
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// ---- morph_validation.go ----

type MorphReport struct {
	Schema           string                             `json:"schema"`
	QualityLevel     string                             `json:"quality_level"`
	Source           string                             `json:"source"`
	Module           string                             `json:"module"`
	SurfaceScope     string                             `json:"surface_scope"`
	Experimental     bool                               `json:"experimental"`
	ProductionClaim  bool                               `json:"production_claim"`
	GitHead          string                             `json:"git_head"`
	GitDirty         bool                               `json:"git_dirty"`
	CapsuleHash      string                             `json:"capsule_hash"`
	TokenGraphHash   string                             `json:"token_graph_hash"`
	Capsule          MorphCapsuleReport                 `json:"capsule"`
	TokenGraph       *MorphTokenGraphReport             `json:"token_graph,omitempty"`
	StyleGraph       *MorphStyleGraphReport             `json:"style_graph,omitempty"`
	Authoring        *MorphAuthoringReport              `json:"authoring,omitempty"`
	Materials        []MorphMaterialReport              `json:"materials,omitempty"`
	LayoutModes      []string                           `json:"layout_modes,omitempty"`
	TypographyRoles  []string                           `json:"typography_roles,omitempty"`
	AssetRefs        []MorphAssetRefReport              `json:"asset_refs,omitempty"`
	Affordances      []MorphAffordanceReport            `json:"affordances,omitempty"`
	StateLenses      []MorphStateLensReport             `json:"state_lenses,omitempty"`
	MotionPresets    []MorphMotionPresetReport          `json:"motion_presets,omitempty"`
	Recipes          []MorphRecipeReport                `json:"recipes,omitempty"`
	RecipeExpansions []MorphRecipeExpansionReport       `json:"recipe_expansions,omitempty"`
	RecipeApps       []MorphRecipeAppReport             `json:"recipe_apps,omitempty"`
	Accessibility    MorphAccessibilityProjectionReport `json:"accessibility"`
	EvidenceContract MorphEvidenceContractReport        `json:"evidence_contract"`
	MemoryBudget     MorphMemoryBudgetReport            `json:"memory_budget"`
	NegativeGuards   MorphNegativeGuardsReport          `json:"negative_guards"`
	NonClaims        []string                           `json:"nonclaims,omitempty"`
}

type MorphStyleGraphReport struct {
	Schema                         string   `json:"schema"`
	Namespace                      string   `json:"namespace"`
	Version                        string   `json:"version"`
	CSSReplacementLevel            string   `json:"css_replacement_level"`
	VocabularyFrozen               bool     `json:"vocabulary_frozen"`
	TokenCategories                []string `json:"token_categories"`
	MaterialSlots                  []string `json:"material_slots"`
	AffordanceRoles                []string `json:"affordance_roles"`
	RecipeOutputs                  []string `json:"recipe_outputs"`
	StateSelectors                 []string `json:"state_selectors"`
	MotionProperties               []string `json:"motion_properties"`
	OverrideOrder                  []string `json:"override_order"`
	ConflictDiagnostics            []string `json:"conflict_diagnostics"`
	ImportAllowlist                []string `json:"import_allowlist"`
	CSSCascadeImportsRejected      bool     `json:"css_cascade_imports_rejected"`
	DOMRuntimeImportsRejected      bool     `json:"dom_runtime_imports_rejected"`
	ReactRuntimeImportsRejected    bool     `json:"react_runtime_imports_rejected"`
	ElectronRuntimeImportsRejected bool     `json:"electron_runtime_imports_rejected"`
	SelectorEngineAbsent           bool     `json:"selector_engine_absent"`
	NoSpecificityScoring           bool     `json:"no_specificity_scoring"`
	GlobalStyleLeakRejected        bool     `json:"global_style_leak_rejected"`
	SpecificityAmbiguityRejected   bool     `json:"specificity_ambiguity_rejected"`
	RawCSSRuntimeImportRejected    bool     `json:"raw_css_runtime_import_rejected"`
}

type MorphAuthoringReport struct {
	Schema                   string   `json:"schema"`
	Level                    string   `json:"level"`
	RecipeCount              int      `json:"recipe_count"`
	PolishedRecipeCount      int      `json:"polished_recipe_count"`
	MaxAuthorFields          int      `json:"max_author_fields"`
	RawBlockFieldCount       int      `json:"raw_block_field_count"`
	Raw80FieldBlocksRejected bool     `json:"raw_80_field_blocks_rejected"`
	RecipesRequired          bool     `json:"recipes_required"`
	DirectBlockPropEditing   bool     `json:"direct_block_prop_editing"`
	RecipeFirstAuthoring     bool     `json:"recipe_first_authoring"`
	DesignerTokenInputs      bool     `json:"designer_token_inputs"`
	GeneratedBlockPropsOnly  bool     `json:"generated_block_props_only"`
	RawLiteralStylesRejected bool     `json:"raw_literal_styles_rejected"`
	NonClaims                []string `json:"nonclaims"`
}

type MorphCapsuleReport struct {
	Namespace       string   `json:"namespace"`
	Version         string   `json:"version"`
	CapsuleHash     string   `json:"capsule_hash"`
	Imports         []string `json:"imports"`
	ExplicitImports bool     `json:"explicit_imports"`
	NoGlobalCascade bool     `json:"no_global_cascade"`
}

type MorphTokenGraphReport struct {
	Schema                     string                           `json:"schema"`
	Namespace                  string                           `json:"namespace"`
	Version                    string                           `json:"version"`
	Hash                       string                           `json:"hash"`
	SourceOfTruth              string                           `json:"source_of_truth,omitempty"`
	ExplicitImports            bool                             `json:"explicit_imports,omitempty"`
	NoGlobalCascade            bool                             `json:"no_global_cascade,omitempty"`
	FixedOverrideOrder         []string                         `json:"fixed_override_order,omitempty"`
	Categories                 []string                         `json:"categories"`
	Tokens                     []MorphTokenReport               `json:"tokens"`
	DensityDPI                 []MorphDensityDPIReport          `json:"density_dpi,omitempty"`
	Diagnostics                MorphTokenGraphDiagnosticsReport `json:"diagnostics,omitempty"`
	AliasCycleRejected         bool                             `json:"alias_cycle_rejected"`
	DuplicateSourceRejected    bool                             `json:"duplicate_source_rejected"`
	RawLiteralsInAppCode       bool                             `json:"raw_literals_in_app_code"`
	UnresolvedFallbackRejected bool                             `json:"unresolved_fallback_rejected"`
	FallbackToRandomDefault    bool                             `json:"fallback_to_random_default"`
}

type MorphDensityDPIReport struct {
	Target         string `json:"target"`
	Token          string `json:"token"`
	TargetDPI      int    `json:"target_dpi"`
	ScaleMilli     int    `json:"scale_milli"`
	RoundingPolicy string `json:"rounding_policy"`
}

type MorphTokenGraphDiagnosticsReport struct {
	AliasCycleRejected           bool `json:"alias_cycle_rejected,omitempty"`
	MissingTokenRejected         bool `json:"missing_token_rejected,omitempty"`
	DuplicateSourceRejected      bool `json:"duplicate_source_rejected,omitempty"`
	RawLiteralRejected           bool `json:"raw_literal_rejected,omitempty"`
	UnresolvedFallbackRejected   bool `json:"unresolved_fallback_rejected,omitempty"`
	CSSRuntimeRejected           bool `json:"css_runtime_rejected,omitempty"`
	MultipleColorSourcesRejected bool `json:"multiple_color_sources_rejected,omitempty"`
	OverrideOrderRejected        bool `json:"override_order_rejected,omitempty"`
	DensityDPIRejected           bool `json:"density_dpi_rejected,omitempty"`
}

type MorphTokenReport struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Kind     string `json:"kind"`
	Value    string `json:"value"`
	Source   string `json:"source"`
	Hash     string `json:"hash"`
}

type MorphMaterialReport struct {
	Name                    string   `json:"name"`
	PaintStack              []string `json:"paint_stack"`
	Fill                    string   `json:"fill"`
	Border                  string   `json:"border"`
	Radius                  string   `json:"radius"`
	Shadow                  string   `json:"shadow"`
	Overlay                 string   `json:"overlay"`
	UnsupportedBlur         bool     `json:"unsupported_blur"`
	UnsupportedBlurRejected bool     `json:"unsupported_blur_rejected"`
}

type MorphAssetRefReport struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	SHA256     string `json:"sha256"`
	Local      bool   `json:"local"`
	FallbackID string `json:"fallback_id"`
	TintToken  string `json:"tint_token"`
}

type MorphAffordanceReport struct {
	Name                  string `json:"name"`
	Role                  string `json:"role"`
	Focusable             bool   `json:"focusable"`
	Action                string `json:"action"`
	Input                 string `json:"input"`
	ProjectsAccessibility bool   `json:"projects_accessibility"`
}

type MorphStateLensReport struct {
	Selector      string `json:"selector"`
	Property      string `json:"property"`
	Deterministic bool   `json:"deterministic"`
}

type MorphMotionPresetReport struct {
	Name              string   `json:"name"`
	DurationMS        int      `json:"duration_ms"`
	Curve             string   `json:"curve"`
	Properties        []string `json:"properties"`
	ReducedMotion     bool     `json:"reduced_motion"`
	DeterministicTime bool     `json:"deterministic_time"`
}

type MorphRecipeReport struct {
	Name                   string   `json:"name"`
	Family                 string   `json:"family,omitempty"`
	Output                 string   `json:"output"`
	Slots                  []string `json:"slots"`
	Inputs                 []string `json:"inputs"`
	State                  []string `json:"state,omitempty"`
	Accessibility          []string `json:"accessibility,omitempty"`
	ExpandsToBlockGraph    bool     `json:"expands_to_block_graph"`
	HiddenAppState         bool     `json:"hidden_app_state"`
	PlatformWidgets        bool     `json:"platform_widgets"`
	CorePrimitivePromotion bool     `json:"core_primitive_promotion"`
}

type MorphRecipeExpansionReport struct {
	Recipe       string   `json:"recipe"`
	BlockIDs     []int    `json:"block_ids"`
	SlotBindings []string `json:"slot_bindings"`
	Variant      string   `json:"variant"`
	Reported     bool     `json:"reported"`
}

type MorphRecipeAppReport struct {
	Source                  string   `json:"source"`
	Module                  string   `json:"module"`
	Recipes                 []string `json:"recipes"`
	ExpandsToBlockGraph     bool     `json:"expands_to_block_graph"`
	BlockCount              int      `json:"block_count"`
	AccessibilityProjection bool     `json:"accessibility_projection"`
	HiddenAppState          bool     `json:"hidden_app_state"`
	ReactRuntime            bool     `json:"react_runtime"`
	ElectronRuntime         bool     `json:"electron_runtime"`
	DOMRuntime              bool     `json:"dom_runtime"`
	PlatformWidgets         bool     `json:"platform_widgets"`
	OutputPrimitives        []string `json:"output_primitives"`
}

type MorphAccessibilityProjectionReport struct {
	Schema                string   `json:"schema"`
	DerivedFromBlockGraph bool     `json:"derived_from_block_graph"`
	SafetyOverridesWin    bool     `json:"safety_overrides_win"`
	SnapshotEvidence      bool     `json:"snapshot_evidence"`
	RequiredFields        []string `json:"required_fields"`
	Roles                 []string `json:"roles"`
}

type MorphEvidenceContractReport struct {
	CapsuleHash       string `json:"capsule_hash"`
	TokenGraphHash    string `json:"token_graph_hash"`
	RecipeExpansions  bool   `json:"recipe_expansions"`
	BlockTree         bool   `json:"block_tree"`
	ResolvedLayout    bool   `json:"resolved_layout"`
	PaintLayers       bool   `json:"paint_layers"`
	TextRuns          bool   `json:"text_runs"`
	MotionFrames      bool   `json:"motion_frames"`
	AssetHashes       bool   `json:"asset_hashes"`
	AccessibilityTree bool   `json:"accessibility_tree"`
	MemoryBudget      bool   `json:"memory_budget"`
	FrameChecksums    bool   `json:"frame_checksums"`
	ArtifactHashes    bool   `json:"artifact_hashes"`
}

type MorphMemoryBudgetReport struct {
	Schema                 string `json:"schema"`
	ExpandedRecipeCount    int    `json:"expanded_recipe_count"`
	BlockCount             int    `json:"block_count"`
	PaintCommandCount      int    `json:"paint_command_count"`
	LayoutPassCount        int    `json:"layout_pass_count"`
	TextRunCount           int    `json:"text_run_count"`
	MotionActiveCount      int    `json:"motion_active_count"`
	GlyphCacheBytes        int    `json:"glyph_cache_bytes"`
	AssetCacheBytes        int    `json:"asset_cache_bytes"`
	LayoutCacheBytes       int    `json:"layout_cache_bytes"`
	FramebufferBytes       int    `json:"framebuffer_bytes"`
	PeakRSSBytes           int    `json:"peak_rss_bytes"`
	AllocCount             int    `json:"alloc_count"`
	FrameCount             int    `json:"frame_count"`
	BoundedCaches          bool   `json:"bounded_caches"`
	UnboundedCacheRejected bool   `json:"unbounded_cache_rejected"`
}

type MorphNegativeGuardsReport struct {
	NoCoreWidgetPrimitives          bool `json:"no_core_widget_primitives"`
	NoDOMUI                         bool `json:"no_dom_ui"`
	NoReact                         bool `json:"no_react"`
	NoElectron                      bool `json:"no_electron"`
	NoUserJS                        bool `json:"no_user_js"`
	NoPlatformWidgets               bool `json:"no_platform_widgets"`
	MissingTokenRejected            bool `json:"missing_token_rejected"`
	AliasCycleRejected              bool `json:"alias_cycle_rejected"`
	DuplicateTokenSourceRejected    bool `json:"duplicate_token_source_rejected"`
	DuplicateRecipeNameRejected     bool `json:"duplicate_recipe_name_rejected"`
	MissingRecipeExpansionRejected  bool `json:"missing_recipe_expansion_rejected"`
	UnresolvedTokenRejected         bool `json:"unresolved_token_rejected"`
	MissingAssetRejected            bool `json:"missing_asset_rejected"`
	UnboundedCacheRejected          bool `json:"unbounded_cache_rejected"`
	FakeMotionRejected              bool `json:"fake_motion_rejected"`
	FakeAccessibilityRejected       bool `json:"fake_accessibility_rejected"`
	UnsupportedTargetRejected       bool `json:"unsupported_target_rejected"`
	DirtyCheckoutProductionRejected bool `json:"dirty_checkout_production_rejected"`
}

func validateMorphEvidence(report Report) []string {
	if report.Morph == nil {
		return nil
	}

	morph := report.Morph
	var issues []string
	if morph.Schema != "tetra.surface.morph.v1" {
		issues = append(
			issues,
			fmt.Sprintf("morph schema is %q, want tetra.surface.morph.v1", morph.Schema),
		)
	}
	if morph.QualityLevel != "deterministic-headless-morph-capsule-v1" {
		issues = append(
			issues,
			fmt.Sprintf(
				"morph quality_level is %q, want deterministic-headless-morph-capsule-v1",
				morph.QualityLevel,
			),
		)
	}
	if !isSupportedMorphRuntimeEvidence(report) {
		issues = append(
			issues,
			("morph evidence requires deterministic headless Surface runtime " +
				"evidence or wasm32-web browser-canvas runtime evidence"),
		)
	}
	if strings.TrimSpace(morph.Source) == "" {
		issues = append(issues, "morph source is required")
	}
	if !isSurfaceMorphReportSource(morph.Source) {
		issues = append(
			issues,
			fmt.Sprintf(
				("morph source is %q, want examples/surface/morph_core/surface_"+
					"morph_command_palette.tetra, a reference app, or generated Surface "+
					"project template source"),
				morph.Source,
			),
		)
	}
	if morph.Module != "lib.core.morph" {
		issues = append(
			issues,
			fmt.Sprintf("morph module is %q, want lib.core.morph", morph.Module),
		)
	}
	if morph.SurfaceScope != "surface-morph-experimental-linux-web" {
		issues = append(
			issues,
			fmt.Sprintf(
				"morph surface_scope is %q, want surface-morph-experimental-linux-web",
				morph.SurfaceScope,
			),
		)
	}
	if !morph.Experimental {
		issues = append(
			issues,
			"morph experimental must be true until clean release-candidate signoff exists",
		)
	}
	if morph.ProductionClaim && morph.GitDirty {
		issues = append(issues, "morph production claim rejected for dirty checkout")
	}
	if morph.ProductionClaim && !isGitHead(morph.GitHead) {
		issues = append(issues, "morph production claim requires git_head evidence")
	}
	if !validSHA256Digest(morph.CapsuleHash) {
		issues = append(issues, "morph capsule_hash must be sha256 evidence")
	}
	if !validSHA256Digest(morph.TokenGraphHash) {
		issues = append(issues, "morph token_graph_hash must be sha256 evidence")
	}
	if morph.Capsule.CapsuleHash != "" && morph.Capsule.CapsuleHash != morph.CapsuleHash {
		issues = append(issues, "morph capsule.capsule_hash must match morph capsule_hash")
	}
	issues = append(issues, validateMorphCapsule(morph.Capsule)...)
	issues = append(issues, validateMorphTokenGraph(morph)...)
	issues = append(issues, validateMorphMaterials(morph.Materials)...)
	issues = append(issues, validateMorphLayoutModes(morph.LayoutModes)...)
	issues = append(issues, validateMorphTypographyRoles(morph.TypographyRoles)...)
	issues = append(issues, validateMorphAssetRefs(morph.AssetRefs, report)...)
	issues = append(issues, validateMorphAffordances(morph.Affordances)...)
	issues = append(issues, validateMorphStateLenses(morph.StateLenses)...)
	issues = append(issues, validateMorphMotionPresets(morph.MotionPresets, report)...)
	issues = append(issues, validateMorphRecipes(morph.Recipes)...)
	issues = append(issues, validateMorphRecipeExpansions(morph.RecipeExpansions, report)...)
	issues = append(
		issues,
		validateMorphRecipeApps(morph.RecipeApps, morph.Recipes, morph.RecipeExpansions)...)
	issues = append(issues, validateMorphAccessibilityProjection(morph.Accessibility, report)...)
	issues = append(issues, validateMorphEvidenceContract(morph.EvidenceContract, morph, report)...)
	issues = append(issues, validateMorphMemoryBudget(morph.MemoryBudget, report)...)
	issues = append(issues, validateMorphNegativeGuards(morph.NegativeGuards)...)
	issues = append(issues, validateMorphNonClaims(morph.NonClaims)...)
	if report.BlockSystem == nil && report.Target == "headless" {
		issues = append(issues, "morph evidence requires block_system evidence")
	}
	if report.BlockGraph == nil || report.BlockAccessibilityTree == nil {
		issues = append(
			issues,
			"morph evidence requires Block graph and accessibility tree evidence",
		)
	}
	if len(report.PaintLayers) == 0 || len(report.PaintCommands) == 0 {
		issues = append(issues, "morph evidence requires resolved paint layer evidence")
	}
	if len(report.LayoutPasses) == 0 || len(report.LayoutConstraints) == 0 {
		issues = append(issues, "morph evidence requires resolved layout evidence")
	}
	if !hasBlockTextEvidence(report) {
		issues = append(issues, "morph evidence requires text run evidence")
	}
	if !hasBlockMotionEvidence(report) {
		issues = append(issues, "morph evidence requires motion frame evidence")
	}
	if !hasBlockAssetEvidence(report) {
		issues = append(issues, "morph evidence requires asset hash/cache evidence")
	}
	return issues
}

func isSupportedMorphRuntimeEvidence(report Report) bool {
	if report.Target == "headless" && report.Runtime == "surface-headless" &&
		report.HostEvidence.Level == "deterministic-headless" {
		return true
	}
	if report.Target == "wasm32-web" &&
		report.Runtime == "surface-wasm32-web" &&
		report.HostEvidence.Level == "wasm32-web-browser-canvas-input" &&
		report.HostEvidence.Backend == "browser-canvas-rgba" &&
		report.HostEvidence.Framebuffer &&
		report.HostEvidence.NativeInput &&
		report.HostEvidence.BrowserCanvas &&
		report.HostEvidence.BrowserInput &&
		!report.HostEvidence.RealWindow {
		return true
	}
	if report.Target == "linux-x64" &&
		report.Runtime == "surface-linux-x64" &&
		report.HostEvidence.Level == "linux-x64-real-window" &&
		report.HostEvidence.Backend == "wayland-shm-rgba" &&
		report.HostEvidence.Framebuffer &&
		report.HostEvidence.RealWindow &&
		report.HostEvidence.NativeInput {
		return true
	}
	return false
}

func isSurfaceMorphReportSource(source string) bool {
	source = normalizeEvidencePath(source)
	return source == "examples/surface/morph_core/surface_morph_command_palette.tetra" ||
		source == "examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra" ||
		source == "examples/surface/morph_flagship/surface_morph_guest_dashboard.tetra" ||
		isSurfaceReferenceAppSource(source) ||
		isSurfaceProjectTemplateSource(source)
}

func validateMorphCapsule(capsule MorphCapsuleReport) []string {
	var issues []string
	if capsule.Namespace != "tetra.surface.morph.app" {
		issues = append(
			issues,
			fmt.Sprintf(
				"morph capsule namespace is %q, want tetra.surface.morph.app",
				capsule.Namespace,
			),
		)
	}
	if strings.TrimSpace(capsule.Version) == "" {
		issues = append(issues, "morph capsule version is required")
	}
	if !validSHA256Digest(capsule.CapsuleHash) {
		issues = append(issues, "morph capsule_hash must be sha256 evidence")
	}
	if !capsule.ExplicitImports {
		issues = append(issues, "morph capsule requires explicit imports")
	}
	if !capsule.NoGlobalCascade {
		issues = append(issues, "morph capsule must prove no global cascade")
	}
	for _, required := range []string{"lib.core.block", "lib.core.morph"} {
		if !contains(capsule.Imports, required) {
			issues = append(issues, fmt.Sprintf("morph capsule imports must include %s", required))
		}
	}
	return issues
}

func validateMorphTokenGraph(morph *MorphReport) []string {
	var issues []string
	if morph.TokenGraph == nil {
		return []string{"morph token_graph is required"}
	}
	graph := morph.TokenGraph
	if graph.Schema != "tetra.surface.morph.token-graph.v1" {
		issues = append(
			issues,
			fmt.Sprintf(
				"morph token_graph schema is %q, want tetra.surface.morph.token-graph.v1",
				graph.Schema,
			),
		)
	}
	if graph.Namespace != morph.Capsule.Namespace {
		issues = append(issues, "morph token_graph namespace must match capsule namespace")
	}
	if graph.Hash != morph.TokenGraphHash {
		issues = append(issues, "morph token_graph hash must match morph token_graph_hash")
	}
	if !validSHA256Digest(graph.Hash) {
		issues = append(issues, "morph token_graph hash must be sha256 evidence")
	}
	for _, category := range []string{
		"color",
		"space",
		"radius",
		"border",
		"elevation",
		"opacity",
		"typography",
		"motion",
		"z",
		"assets",
		"density",
	} {
		if !containsNormalized(graph.Categories, category) {
			issues = append(
				issues,
				fmt.Sprintf("morph token_graph categories require %s", category),
			)
		}
	}
	if len(graph.Tokens) == 0 {
		issues = append(issues, "morph token_graph tokens are required")
	}
	seenIDs := map[string]string{}
	for i, token := range graph.Tokens {
		id := strings.TrimSpace(token.ID)
		if id == "" {
			issues = append(issues, fmt.Sprintf("morph token_graph tokens[%d].id is required", i))
		}
		if strings.TrimSpace(token.Category) == "" ||
			!containsNormalized(graph.Categories, token.Category) {
			issues = append(
				issues,
				fmt.Sprintf(
					"morph token_graph token %q category %q is not declared",
					token.ID,
					token.Category,
				),
			)
		}
		if strings.TrimSpace(token.Kind) == "" {
			issues = append(
				issues,
				fmt.Sprintf("morph token_graph token %q kind is required", token.ID),
			)
		}
		if strings.TrimSpace(token.Value) == "" {
			issues = append(
				issues,
				fmt.Sprintf("morph token_graph token %q value is required", token.ID),
			)
		}
		if strings.TrimSpace(token.Source) != "capsule" {
			issues = append(
				issues,
				fmt.Sprintf(
					"morph token_graph token %q source is %q, want capsule",
					token.ID,
					token.Source,
				),
			)
		}
		if !validSHA256Digest(token.Hash) {
			issues = append(
				issues,
				fmt.Sprintf("morph token_graph token %q hash must be sha256 evidence", token.ID),
			)
		}
		if previous, ok := seenIDs[id]; ok {
			issues = append(
				issues,
				fmt.Sprintf(
					"morph token_graph duplicate token %q from %s and %s",
					id,
					previous,
					token.Source,
				),
			)
		}
		seenIDs[id] = token.Source
	}
	if graph.RawLiteralsInAppCode {
		issues = append(issues, "morph token_graph rejects raw literals in app code")
	}
	if graph.FallbackToRandomDefault {
		issues = append(issues, "morph token_graph rejects fallback-to-random-default")
	}
	if !graph.AliasCycleRejected || !graph.DuplicateSourceRejected ||
		!graph.UnresolvedFallbackRejected {
		issues = append(
			issues,
			("morph token_graph negative guards require alias_cycle, " +
				"duplicate_source, and unresolved_fallback rejection"),
		)
	}
	return issues
}

func validateMorphMaterials(materials []MorphMaterialReport) []string {
	var issues []string
	if len(materials) == 0 {
		return []string{"morph materials are required"}
	}
	seenFeatures := map[string]bool{}
	for _, material := range materials {
		if strings.TrimSpace(material.Name) == "" {
			issues = append(issues, "morph material name is required")
		}
		if material.UnsupportedBlur {
			issues = append(
				issues,
				fmt.Sprintf("morph material %q must not claim unsupported blur", material.Name),
			)
		}
		if !material.UnsupportedBlurRejected {
			issues = append(
				issues,
				fmt.Sprintf(
					"morph material %q must prove unsupported blur diagnostics",
					material.Name,
				),
			)
		}
		for _, feature := range material.PaintStack {
			seenFeatures[normalizeStateToken(feature)] = true
		}
		if token, ok := forbiddenBlockCorePrimitiveToken(material.Name); ok {
			issues = append(
				issues,
				fmt.Sprintf(
					"morph material fake core primitive %s rejected in %q",
					token,
					material.Name,
				),
			)
		}
	}
	for _, feature := range []string{"fill", "border", "radius", "shadow", "overlay"} {
		if !seenFeatures[feature] {
			issues = append(
				issues,
				fmt.Sprintf("morph materials require %s paint stack evidence", feature),
			)
		}
	}
	return issues
}

func validateMorphLayoutModes(modes []string) []string {
	var issues []string
	for _, mode := range []string{
		"row",
		"column",
		"stack",
		"grid",
		"dock",
		"absolute",
		"overlay",
		"scroll",
	} {
		if !containsNormalized(modes, mode) {
			issues = append(issues, fmt.Sprintf("morph layout_modes require %s", mode))
		}
	}
	return issues
}

func validateMorphTypographyRoles(roles []string) []string {
	var issues []string
	for _, role := range []string{"title", "body", "label", "code"} {
		if !containsNormalized(roles, role) {
			issues = append(issues, fmt.Sprintf("morph typography_roles require %s", role))
		}
	}
	return issues
}

func validateMorphAssetRefs(refs []MorphAssetRefReport, report Report) []string {
	var issues []string
	if len(refs) == 0 {
		issues = append(issues, "morph asset_refs are required")
	}
	seen := map[string]bool{}
	for _, ref := range refs {
		if strings.TrimSpace(ref.ID) == "" {
			issues = append(issues, "morph asset_ref id is required")
		}
		seen[ref.ID] = true
		if ref.Kind != "icon" && ref.Kind != "font" && ref.Kind != "image" {
			issues = append(
				issues,
				fmt.Sprintf(
					"morph asset_ref %q kind is %q, want icon, font, or image",
					ref.ID,
					ref.Kind,
				),
			)
		}
		if !validSHA256Digest(ref.SHA256) {
			issues = append(
				issues,
				fmt.Sprintf("morph asset_ref %q sha256 must be sha256 evidence", ref.ID),
			)
		}
		if !ref.Local {
			issues = append(issues, fmt.Sprintf("morph asset_ref %q must be local", ref.ID))
		}
		if strings.TrimSpace(ref.FallbackID) == "" {
			issues = append(
				issues,
				fmt.Sprintf("morph asset_ref %q fallback_id is required", ref.ID),
			)
		}
		if strings.TrimSpace(ref.TintToken) == "" {
			issues = append(
				issues,
				fmt.Sprintf("morph asset_ref %q tint_token is required", ref.ID),
			)
		}
	}
	for _, required := range []string{"project.new", "command.search", "status.warning"} {
		if !seen[required] {
			issues = append(issues, fmt.Sprintf("morph asset_refs require %s", required))
		}
	}
	if report.BlockAssetNetworkFetchAllowed {
		issues = append(issues, "morph asset evidence requires network fetch disabled")
	}
	if !report.BlockAssetCache.Bounded {
		issues = append(issues, "morph asset evidence requires bounded asset cache")
	}
	return issues
}

func validateMorphAffordances(affordances []MorphAffordanceReport) []string {
	var issues []string
	seen := map[string]MorphAffordanceReport{}
	for _, affordance := range affordances {
		seen[normalizeStateToken(affordance.Name)] = affordance
		if strings.TrimSpace(affordance.Role) == "" {
			issues = append(
				issues,
				fmt.Sprintf("morph affordance %q role is required", affordance.Name),
			)
		}
		if !affordance.ProjectsAccessibility {
			issues = append(
				issues,
				fmt.Sprintf("morph affordance %q must project accessibility", affordance.Name),
			)
		}
	}
	for _, required := range []string{
		"action",
		"field.text",
		"toggle",
		"navigation",
		"region",
		"overlay",
		"status",
	} {
		key := normalizeStateToken(required)
		affordance, ok := seen[key]
		if !ok {
			issues = append(issues, fmt.Sprintf("morph affordances require %s", required))
			continue
		}
		if required == "action" &&
			(!affordance.Focusable || strings.TrimSpace(affordance.Action) == "") {
			issues = append(issues, "morph action affordance requires focusable action evidence")
		}
		if required == "field.text" &&
			(!affordance.Focusable || affordance.Input != "editable_text") {
			issues = append(issues, "morph field.text affordance requires editable text evidence")
		}
	}
	return issues
}

func validateMorphStateLenses(lenses []MorphStateLensReport) []string {
	var issues []string
	seen := map[string]bool{}
	for _, lens := range lenses {
		selector := normalizeStateToken(lens.Selector)
		seen[selector] = true
		if strings.TrimSpace(lens.Property) == "" {
			issues = append(
				issues,
				fmt.Sprintf("morph state_lens %q property is required", lens.Selector),
			)
		}
		if !lens.Deterministic {
			issues = append(
				issues,
				fmt.Sprintf("morph state_lens %q must be deterministic", lens.Selector),
			)
		}
	}
	for _, selector := range []string{
		"hover",
		"pressed",
		"focusvisible",
		"selected",
		"disabled",
		"error",
		"loading",
	} {
		if !seen[selector] {
			issues = append(issues, fmt.Sprintf("morph state_lenses require %s", selector))
		}
	}
	return issues
}

func validateMorphMotionPresets(presets []MorphMotionPresetReport, report Report) []string {
	var issues []string
	if len(presets) == 0 {
		return []string{"morph motion_presets are required"}
	}
	hasReduced := false
	for _, preset := range presets {
		if strings.TrimSpace(preset.Name) == "" {
			issues = append(issues, "morph motion_preset name is required")
		}
		if preset.DurationMS <= 0 {
			issues = append(
				issues,
				fmt.Sprintf("morph motion_preset %q duration_ms must be positive", preset.Name),
			)
		}
		if strings.TrimSpace(preset.Curve) == "" {
			issues = append(
				issues,
				fmt.Sprintf("morph motion_preset %q curve is required", preset.Name),
			)
		}
		for _, property := range []string{"fill", "opacity", "transform"} {
			if !containsNormalized(preset.Properties, property) {
				issues = append(
					issues,
					fmt.Sprintf(
						"morph motion_preset %q properties require %s",
						preset.Name,
						property,
					),
				)
			}
		}
		if preset.ReducedMotion && preset.DeterministicTime {
			hasReduced = true
		}
	}
	if !hasReduced {
		issues = append(
			issues,
			"morph motion_presets require deterministic reduced-motion evidence",
		)
	}
	if report.MotionUnsupportedCSSAnimations {
		issues = append(issues, "morph motion evidence must not claim CSS animation parity")
	}
	return issues
}

func requiredMorphRecipeNames() []string {
	return []string{
		"control.action@1",
		"field.text@1",
		"command.item@1",
		"region.panel@1",
		"form.field@1",
		"nav.item@1",
		"metric.tile@1",
		"dialog.panel@1",
		"toast.notification@1",
		"tab.item@1",
		"list.row@1",
		"app.shell@1",
		"toolbar@1",
		"split.pane@1",
		"status.bar@1",
		"settings.form@1",
		"log.row@1",
		"empty.state@1",
		"error.panel@1",
	}
}

func requiredMorphRecipeAppSources() []string {
	return []string{
		"examples/surface/morph_core/surface_morph_command_palette.tetra",
		"examples/surface/morph_core/surface_morph_project_dashboard.tetra",
		"examples/surface/morph_core/surface_morph_settings.tetra",
		"examples/surface/morph_core/surface_morph_editor_shell.tetra",
		"examples/surface/morph_core/surface_morph_glass_panel.tetra",
		"examples/surface/morph_core/surface_morph_studio_shell.tetra",
		"examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra",
	}
}

func ValidateMorphStyleTokenBoundary(morph *MorphReport) error {
	if morph == nil {
		return fmt.Errorf("morph report is required")
	}
	if morph.StyleGraph == nil {
		return fmt.Errorf("style_graph is required")
	}
	if morph.Authoring == nil {
		return fmt.Errorf("authoring is required")
	}
	if !morph.StyleGraph.RawCSSRuntimeImportRejected {
		return fmt.Errorf("raw css runtime import must be rejected")
	}
	if morph.Authoring.DirectBlockPropEditing {
		return fmt.Errorf("direct block prop editing must be rejected")
	}
	if morph.Authoring.RecipeCount < 11 ||
		morph.Authoring.PolishedRecipeCount < 11 ||
		len(morph.Recipes) < 11 {
		return fmt.Errorf("recipe_count is incomplete for stable Morph boundary")
	}
	if !morph.Authoring.RecipesRequired || !morph.Authoring.RecipeFirstAuthoring {
		return fmt.Errorf("recipe-first authoring is required")
	}
	if !morph.Authoring.Raw80FieldBlocksRejected || !morph.Authoring.RawLiteralStylesRejected {
		return fmt.Errorf("raw Block/style authoring must be rejected")
	}
	return nil
}

func validateMorphRecipes(recipes []MorphRecipeReport) []string {
	var issues []string
	if len(recipes) == 0 {
		return []string{"morph recipes are required"}
	}
	seen := map[string]bool{}
	for _, recipe := range recipes {
		name := strings.TrimSpace(recipe.Name)
		if name == "" {
			issues = append(issues, "morph recipe name is required")
		}
		if seen[name] {
			issues = append(issues, fmt.Sprintf("morph duplicate recipe name %q rejected", name))
		}
		seen[name] = true
		if token, ok := forbiddenBlockCorePrimitiveToken(name); ok {
			issues = append(
				issues,
				fmt.Sprintf("morph recipe fake core primitive %s rejected in name %q", token, name),
			)
		}
		if token, ok := forbiddenBlockCorePrimitiveToken(recipe.Output); ok {
			issues = append(
				issues,
				fmt.Sprintf(
					"morph recipe fake core primitive %s rejected in output %q",
					token,
					recipe.Output,
				),
			)
		}
		if recipe.Output != "Block" {
			issues = append(
				issues,
				fmt.Sprintf("morph recipe %q output is %q, want Block", name, recipe.Output),
			)
		}
		if len(recipe.Slots) == 0 {
			issues = append(issues, fmt.Sprintf("morph recipe %q must declare slots", name))
		}
		if len(recipe.Inputs) == 0 {
			issues = append(issues, fmt.Sprintf("morph recipe %q must declare inputs", name))
		}
		if !recipe.ExpandsToBlockGraph {
			issues = append(issues, fmt.Sprintf("morph recipe %q must expand to Block graph", name))
		}
		if recipe.HiddenAppState {
			issues = append(
				issues,
				fmt.Sprintf("morph recipe %q must not allocate hidden app state", name),
			)
		}
		if recipe.PlatformWidgets {
			issues = append(
				issues,
				fmt.Sprintf("morph recipe %q must not use platform widgets", name),
			)
		}
		if recipe.CorePrimitivePromotion {
			issues = append(
				issues,
				fmt.Sprintf("morph recipe %q core primitive promotion rejected", name),
			)
		}
	}
	for _, required := range requiredMorphRecipeNames() {
		if !seen[required] {
			issues = append(issues, fmt.Sprintf("morph recipes require %s", required))
		}
	}
	return issues
}

func validateMorphRecipeExpansions(
	expansions []MorphRecipeExpansionReport,
	report Report,
) []string {
	if len(expansions) == 0 {
		return []string{"morph recipe_expansions are required"}
	}
	var issues []string
	blockIDs := map[int]bool{}
	if report.BlockGraph != nil {
		for _, node := range report.BlockGraph.Nodes {
			blockIDs[node.ID] = true
		}
	}
	seenRecipe := map[string]bool{}
	for _, expansion := range expansions {
		if strings.TrimSpace(expansion.Recipe) == "" {
			issues = append(issues, "morph recipe_expansions recipe is required")
		}
		seenRecipe[expansion.Recipe] = true
		if len(expansion.BlockIDs) == 0 {
			issues = append(
				issues,
				fmt.Sprintf("morph recipe_expansions %q block_ids are required", expansion.Recipe),
			)
		}
		for _, blockID := range expansion.BlockIDs {
			if blockID <= 0 {
				issues = append(
					issues,
					fmt.Sprintf(
						"morph recipe_expansions %q block_id must be positive",
						expansion.Recipe,
					),
				)
			} else if len(blockIDs) > 0 && !blockIDs[blockID] {
				issues = append(
					issues,
					fmt.Sprintf(
						"morph recipe_expansions %q references missing Block ID %d",
						expansion.Recipe,
						blockID,
					),
				)
			}
		}
		if len(expansion.SlotBindings) == 0 {
			issues = append(
				issues,
				fmt.Sprintf(
					"morph recipe_expansions %q slot_bindings are required",
					expansion.Recipe,
				),
			)
		}
		if !expansion.Reported {
			issues = append(
				issues,
				fmt.Sprintf("morph recipe_expansions %q must be reported", expansion.Recipe),
			)
		}
	}
	for _, required := range requiredMorphRecipeNames() {
		if !seenRecipe[required] {
			issues = append(issues, fmt.Sprintf("morph recipe_expansions require %s", required))
		}
	}
	return issues
}

func validateMorphRecipeApps(
	apps []MorphRecipeAppReport,
	recipes []MorphRecipeReport,
	expansions []MorphRecipeExpansionReport,
) []string {
	if len(apps) == 0 {
		return []string{"morph recipe_apps are required"}
	}
	var issues []string
	knownRecipes := map[string]bool{}
	for _, recipe := range recipes {
		knownRecipes[recipe.Name] = true
	}
	expandedRecipes := map[string]bool{}
	for _, expansion := range expansions {
		expandedRecipes[expansion.Recipe] = true
	}
	seenSources := map[string]bool{}
	for _, app := range apps {
		source := normalizeEvidencePath(app.Source)
		seenSources[source] = true
		if !isSurfaceMorphRecipeAppSource(source) {
			issues = append(
				issues,
				fmt.Sprintf(
					"morph recipe_apps source %q must be a Surface Morph example",
					app.Source,
				),
			)
		}
		if strings.TrimSpace(app.Module) == "" {
			issues = append(
				issues,
				fmt.Sprintf("morph recipe_apps %q module is required", app.Source),
			)
		}
		if len(app.Recipes) == 0 {
			issues = append(
				issues,
				fmt.Sprintf("morph recipe_apps %q recipes are required", app.Source),
			)
		}
		for _, recipe := range app.Recipes {
			if !knownRecipes[recipe] {
				issues = append(
					issues,
					fmt.Sprintf(
						"morph recipe_apps %q references undeclared recipe %s",
						app.Source,
						recipe,
					),
				)
			}
			if !expandedRecipes[recipe] {
				issues = append(
					issues,
					fmt.Sprintf(
						"morph recipe_apps %q references recipe %s without expansion report",
						app.Source,
						recipe,
					),
				)
			}
		}
		if !app.ExpandsToBlockGraph {
			issues = append(
				issues,
				fmt.Sprintf("morph recipe_apps %q must expand to Block graph", app.Source),
			)
		}
		if app.BlockCount <= 0 {
			issues = append(
				issues,
				fmt.Sprintf("morph recipe_apps %q block_count must be positive", app.Source),
			)
		}
		if !app.AccessibilityProjection {
			issues = append(
				issues,
				fmt.Sprintf("morph recipe_apps %q requires accessibility projection", app.Source),
			)
		}
		if app.HiddenAppState {
			issues = append(
				issues,
				fmt.Sprintf("morph recipe_apps %q must not allocate hidden app state", app.Source),
			)
		}
		if app.ReactRuntime {
			issues = append(
				issues,
				fmt.Sprintf("morph recipe_apps %q must not use React runtime", app.Source),
			)
		}
		if app.ElectronRuntime {
			issues = append(
				issues,
				fmt.Sprintf("morph recipe_apps %q must not use Electron runtime", app.Source),
			)
		}
		if app.DOMRuntime {
			issues = append(
				issues,
				fmt.Sprintf("morph recipe_apps %q must not use DOM runtime", app.Source),
			)
		}
		if app.PlatformWidgets {
			issues = append(
				issues,
				fmt.Sprintf("morph recipe_apps %q must not use platform widgets", app.Source),
			)
		}
		if !contains(app.OutputPrimitives, "Block") {
			issues = append(
				issues,
				fmt.Sprintf("morph recipe_apps %q output_primitives require Block", app.Source),
			)
		}
		for _, primitive := range app.OutputPrimitives {
			if primitive != "Block" {
				issues = append(
					issues,
					fmt.Sprintf(
						"morph recipe_apps %q fake output primitive %s rejected",
						app.Source,
						primitive,
					),
				)
			}
		}
	}
	for _, required := range requiredMorphRecipeAppSources() {
		if !seenSources[required] {
			issues = append(issues, fmt.Sprintf("morph recipe_apps require %s", required))
		}
	}
	return issues
}

func isSurfaceMorphRecipeAppSource(source string) bool {
	source = normalizeEvidencePath(source)
	return strings.HasPrefix(source, "examples/surface/morph_core/surface_morph_") &&
		strings.HasSuffix(source, ".tetra") ||
		strings.HasPrefix(source, "examples/surface/morph_flagship/surface_morph_") &&
			strings.HasSuffix(source, ".tetra") ||
		strings.Contains(source, "/examples/surface/morph_core/surface_morph_") &&
			strings.HasSuffix(source, ".tetra") ||
		strings.Contains(source, "/examples/surface/morph_flagship/surface_morph_") &&
			strings.HasSuffix(source, ".tetra")
}

func validateMorphAccessibilityProjection(
	projection MorphAccessibilityProjectionReport,
	report Report,
) []string {
	var issues []string
	if projection.Schema != "tetra.surface.morph.accessibility-projection.v1" {
		issues = append(
			issues,
			fmt.Sprintf(
				"morph accessibility schema is %q, want tetra.surface.morph.accessibility-projection.v1",
				projection.Schema,
			),
		)
	}
	if !projection.DerivedFromBlockGraph {
		issues = append(issues, "morph accessibility must be derived from Block graph")
	}
	if !projection.SafetyOverridesWin {
		issues = append(issues, "morph accessibility safety overrides must win")
	}
	if !projection.SnapshotEvidence {
		issues = append(issues, "morph accessibility snapshot evidence is required")
	}
	for _, field := range []string{
		"role",
		"name",
		"description",
		"action",
		"state",
		"bounds",
		"focus_order",
		"reading_order",
		"labelled_by",
		"label_for",
	} {
		if !containsNormalized(projection.RequiredFields, field) {
			issues = append(
				issues,
				fmt.Sprintf("morph accessibility required_fields require %s", field),
			)
		}
	}
	for _, role := range []string{
		"button",
		"textbox",
		"checkbox",
		"navigation",
		"region",
		"dialog",
		"status",
	} {
		if !containsNormalized(projection.Roles, role) {
			issues = append(issues, fmt.Sprintf("morph accessibility roles require %s", role))
		}
	}
	if report.BlockAccessibilityTree == nil {
		issues = append(issues, "morph accessibility requires block_accessibility_tree")
	}
	return issues
}

func validateMorphEvidenceContract(
	contract MorphEvidenceContractReport,
	morph *MorphReport,
	report Report,
) []string {
	var issues []string
	if contract.CapsuleHash != morph.CapsuleHash {
		issues = append(
			issues,
			"morph evidence_contract capsule_hash must match morph capsule_hash",
		)
	}
	if contract.TokenGraphHash != morph.TokenGraphHash {
		issues = append(
			issues,
			"morph evidence_contract token_graph_hash must match morph token_graph_hash",
		)
	}
	required := []struct {
		name string
		ok   bool
	}{
		{"recipe_expansions", contract.RecipeExpansions},
		{"block_tree", contract.BlockTree && report.BlockGraph != nil},
		{"resolved_layout", contract.ResolvedLayout && len(report.LayoutPasses) > 0},
		{"paint_layers", contract.PaintLayers && len(report.PaintLayers) > 0},
		{"text_runs", contract.TextRuns && hasBlockTextEvidence(report)},
		{"motion_frames", contract.MotionFrames && len(report.MotionFrames) > 0},
		{"asset_hashes", contract.AssetHashes && report.BlockAssetManifest != nil},
		{"accessibility_tree", contract.AccessibilityTree && report.BlockAccessibilityTree != nil},
		{
			"memory_budget",
			contract.MemoryBudget && morph.MemoryBudget.Schema == "tetra.surface.morph-memory-budget.v1" &&
				morph.MemoryBudget.FrameCount > 0,
		},
		{"frame_checksums", contract.FrameChecksums && len(report.Frames) > 0},
		{"artifact_hashes", contract.ArtifactHashes && len(report.Artifacts) > 0},
	}
	for _, check := range required {
		if !check.ok {
			issues = append(
				issues,
				fmt.Sprintf("morph evidence_contract requires %s evidence", check.name),
			)
		}
	}
	return issues
}

func validateMorphMemoryBudget(budget MorphMemoryBudgetReport, report Report) []string {
	var issues []string
	if budget.Schema != "tetra.surface.morph-memory-budget.v1" {
		issues = append(
			issues,
			fmt.Sprintf(
				"morph memory_budget schema is %q, want tetra.surface.morph-memory-budget.v1",
				budget.Schema,
			),
		)
	}
	if budget.ExpandedRecipeCount <= 0 {
		issues = append(issues, "morph memory_budget expanded_recipe_count must be positive")
	}
	if report.BlockSystem != nil && report.BlockSystem.MemoryBudget != nil &&
		budget.BlockCount < report.BlockSystem.MemoryBudget.BlockCount {
		issues = append(
			issues,
			fmt.Sprintf(
				"morph memory_budget block_count = %d, want at least Block memory budget block_count %d",
				budget.BlockCount,
				report.BlockSystem.MemoryBudget.BlockCount,
			),
		)
	}
	if budget.PaintCommandCount <= 0 || budget.LayoutPassCount <= 0 || budget.TextRunCount <= 0 ||
		budget.FrameCount <= 0 {
		issues = append(
			issues,
			("morph memory_budget requires paint_command_count, layout_pass_" +
				"count, text_run_count, and frame_count"),
		)
	}
	if budget.GlyphCacheBytes < 0 || budget.AssetCacheBytes < 0 || budget.LayoutCacheBytes < 0 ||
		budget.FramebufferBytes <= 0 {
		issues = append(issues, "morph memory_budget cache/framebuffer byte fields are invalid")
	}
	if !budget.BoundedCaches || !budget.UnboundedCacheRejected {
		issues = append(
			issues,
			"morph memory_budget requires bounded caches and unbounded cache rejection",
		)
	}
	return issues
}

func validateMorphNegativeGuards(guards MorphNegativeGuardsReport) []string {
	missing := []string{}
	checks := []struct {
		name string
		ok   bool
	}{
		{"no_core_widget_primitives", guards.NoCoreWidgetPrimitives},
		{"no_dom_ui", guards.NoDOMUI},
		{"no_react", guards.NoReact},
		{"no_electron", guards.NoElectron},
		{"no_user_js", guards.NoUserJS},
		{"no_platform_widgets", guards.NoPlatformWidgets},
		{"missing_token_rejected", guards.MissingTokenRejected},
		{"alias_cycle_rejected", guards.AliasCycleRejected},
		{"duplicate_token_source_rejected", guards.DuplicateTokenSourceRejected},
		{"duplicate_recipe_name_rejected", guards.DuplicateRecipeNameRejected},
		{"missing_recipe_expansion_rejected", guards.MissingRecipeExpansionRejected},
		{"unresolved_token_rejected", guards.UnresolvedTokenRejected},
		{"missing_asset_rejected", guards.MissingAssetRejected},
		{"unbounded_cache_rejected", guards.UnboundedCacheRejected},
		{"fake_motion_rejected", guards.FakeMotionRejected},
		{"fake_accessibility_rejected", guards.FakeAccessibilityRejected},
		{"unsupported_target_rejected", guards.UnsupportedTargetRejected},
		{"dirty_checkout_production_rejected", guards.DirtyCheckoutProductionRejected},
	}
	for _, check := range checks {
		if !check.ok {
			missing = append(missing, check.name)
		}
	}
	if len(missing) > 0 {
		return []string{
			fmt.Sprintf("morph negative_guards missing %s", strings.Join(missing, ", ")),
		}
	}
	return nil
}

func validateMorphNonClaims(nonclaims []string) []string {
	var issues []string
	for _, required := range []string{
		"DOM runtime absent",
		"React runtime absent",
		"Electron claim absent",
		"platform-native widgets absent",
		"CSS cascade absent",
	} {
		if !containsTextFold(nonclaims, required) {
			issues = append(issues, fmt.Sprintf("morph nonclaims require %q", required))
		}
	}
	return issues
}

// ---- package.go ----

const SurfacePackageSchemaV1 = "tetra.surface.package.v1"

type SurfacePackageReport struct {
	Schema         string                       `json:"schema"`
	Model          string                       `json:"model"`
	ReleaseScope   string                       `json:"release_scope"`
	Producer       string                       `json:"producer"`
	Source         string                       `json:"source"`
	ReferenceApp   string                       `json:"reference_app"`
	PackageFormat  string                       `json:"package_format"`
	FormatVersion  int                          `json:"format_version"`
	ArtifactRoot   string                       `json:"artifact_root"`
	Packages       []SurfacePackageArtifact     `json:"packages"`
	Assets         []SurfacePackageAsset        `json:"assets"`
	InstallSmokes  []SurfacePackageInstallSmoke `json:"install_smokes"`
	WebBundles     []SurfacePackageWebBundle    `json:"web_bundles"`
	UpdateStrategy SurfacePackageUpdateStrategy `json:"update_strategy"`
	Signing        SurfacePackagePlatformProof  `json:"signing"`
	Notarization   SurfacePackagePlatformProof  `json:"notarization"`
	NegativeGuards SurfacePackageNegativeGuards `json:"negative_guards"`
	Pass           bool                         `json:"pass"`
}

type SurfacePackageArtifact struct {
	Target              string `json:"target"`
	Kind                string `json:"kind"`
	Path                string `json:"path"`
	ManifestPath        string `json:"manifest_path"`
	SHA256              string `json:"sha256"`
	AssetManifestSHA256 string `json:"asset_manifest_sha256"`
	SourceSHA256        string `json:"source_sha256"`
	BuildSHA256         string `json:"build_sha256"`
	ContainsExecutable  bool   `json:"contains_executable"`
	ContainsWebBundle   bool   `json:"contains_web_bundle"`
	LocalOnlyAssets     bool   `json:"local_only_assets"`
	Pass                bool   `json:"pass"`
}

type SurfacePackageAsset struct {
	Path                string `json:"path"`
	Kind                string `json:"kind"`
	SHA256              string `json:"sha256"`
	SizeBytes           int64  `json:"size_bytes"`
	LocalOnly           bool   `json:"local_only"`
	NetworkFetchAllowed bool   `json:"network_fetch_allowed"`
	Pass                bool   `json:"pass"`
}

type SurfacePackageInstallSmoke struct {
	Target                  string `json:"target"`
	PackagePath             string `json:"package_path"`
	InstallDir              string `json:"install_dir"`
	InstalledBinary         string `json:"installed_binary"`
	Command                 string `json:"command"`
	ExitCode                int    `json:"exit_code"`
	ExpectedExitCode        int    `json:"expected_exit_code"`
	ArtifactHashVerified    bool   `json:"artifact_hash_verified"`
	PackageManifestVerified bool   `json:"package_manifest_verified"`
	AppRun                  bool   `json:"app_run"`
	Pass                    bool   `json:"pass"`
}

type SurfacePackageWebBundle struct {
	Target                  string `json:"target"`
	PackagePath             string `json:"package_path"`
	WebEntry                string `json:"web_entry"`
	WASMArtifact            string `json:"wasm_artifact"`
	LoaderArtifact          string `json:"loader_artifact"`
	BrowserCanvasHost       string `json:"browser_canvas_host"`
	Command                 string `json:"command"`
	ArtifactHashVerified    bool   `json:"artifact_hash_verified"`
	PackageManifestVerified bool   `json:"package_manifest_verified"`
	Pass                    bool   `json:"pass"`
}

type SurfacePackageUpdateStrategy struct {
	Strategy                            string `json:"strategy"`
	ManifestFormat                      string `json:"manifest_format"`
	ChannelManifest                     string `json:"channel_manifest"`
	CurrentVersion                      string `json:"current_version"`
	LatestVersion                       string `json:"latest_version"`
	LatestPackagePath                   string `json:"latest_package_path"`
	LatestPackageSHA256                 string `json:"latest_package_sha256"`
	PackageHashPinned                   bool   `json:"package_hash_pinned"`
	RollbackManifest                    string `json:"rollback_manifest"`
	SignatureRequiredForStablePromotion bool   `json:"signature_required_for_stable_promotion"`
	AutoUpdateRuntimeClaim              bool   `json:"auto_update_runtime_claim"`
	NetworkUpdateClaim                  bool   `json:"network_update_claim"`
	Pass                                bool   `json:"pass"`
}

type SurfacePackagePlatformProof struct {
	Status          string `json:"status"`
	Signed          bool   `json:"signed"`
	Notarized       bool   `json:"notarized"`
	ProductionClaim bool   `json:"production_claim"`
	Evidence        string `json:"evidence"`
	BlockedReason   string `json:"blocked_reason"`
}

type SurfacePackageNegativeGuards struct {
	NoReactRuntime                        bool `json:"no_react_runtime"`
	NoElectronRuntime                     bool `json:"no_electron_runtime"`
	NoDOMAppUITree                        bool `json:"no_dom_app_ui_tree"`
	NoCSSRuntime                          bool `json:"no_css_runtime"`
	NoUserJSAppLogic                      bool `json:"no_user_js_app_logic"`
	NoRemoteAssetFetch                    bool `json:"no_remote_asset_fetch"`
	NoUnsignedSigningClaim                bool `json:"no_unsigned_signing_claim"`
	NoNotarizationWithoutPlatformEvidence bool `json:"no_notarization_without_platform_evidence"`
	NoAutoUpdateWithoutRuntimeEvidence    bool `json:"no_auto_update_without_runtime_evidence"`
	NoDocsOnlyPackageClaim                bool `json:"no_docs_only_package_claim"`
	InstallRunRequired                    bool `json:"install_run_required"`
	WebBundleRequired                     bool `json:"web_bundle_required"`
	ArtifactHashesRequired                bool `json:"artifact_hashes_required"`
}

func ValidatePackageReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != SurfacePackageSchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, SurfacePackageSchemaV1)
	}
	var report SurfacePackageReport
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}
	issues := validateSurfacePackageReport(report)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateSurfacePackageReport(report SurfacePackageReport) []string {
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: report.Schema, want: SurfacePackageSchemaV1},
		{field: "model", got: report.Model, want: "surface-package-v1"},
		{field: "release_scope", got: report.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
		{
			field: "producer",
			got:   report.Producer,
			want:  "scripts/release/surface/surface-package-smoke.sh",
		},
		{field: "package_format", got: report.PackageFormat, want: "surface-app-package-v1"},
	} {
		if check.got != check.want {
			issues = append(
				issues,
				fmt.Sprintf("%s is %q, want %q", check.field, check.got, check.want),
			)
		}
	}
	if report.FormatVersion != 1 {
		issues = append(issues, fmt.Sprintf("format_version = %d, want 1", report.FormatVersion))
	}
	if !safeRelativeSourcePath(report.Source) {
		issues = append(issues, "source must be a safe Tetra source path")
	}
	if strings.TrimSpace(report.ReferenceApp) == "" {
		issues = append(issues, "reference_app is required")
	}
	if !surfacePackageSourceMatchesReferenceApp(report.ReferenceApp, report.Source) {
		issues = append(
			issues,
			fmt.Sprintf(
				"reference_app %q does not match source %q",
				report.ReferenceApp,
				report.Source,
			),
		)
	}
	if !safeRelativeReportPath(report.ArtifactRoot) {
		issues = append(issues, "artifact_root is unsafe or empty")
	}
	issues = append(issues, validateSurfacePackageArtifacts(report.Packages)...)
	issues = append(issues, validateSurfacePackageAssets(report.Assets)...)
	issues = append(
		issues,
		validateSurfacePackageInstallSmokes(report.ReferenceApp, report.InstallSmokes)...)
	issues = append(issues, validateSurfacePackageWebBundles(report.WebBundles)...)
	issues = append(issues, validateSurfacePackageUpdateStrategy(report.UpdateStrategy)...)
	issues = append(
		issues,
		validateSurfacePackagePlatformProof("signing", report.Signing, false)...)
	issues = append(
		issues,
		validateSurfacePackagePlatformProof("notarization", report.Notarization, true)...)
	issues = append(issues, validateSurfacePackageNegativeGuards(report.NegativeGuards)...)
	if !report.Pass {
		issues = append(issues, "pass must be true")
	}
	return issues
}

func surfacePackageSourceMatchesReferenceApp(referenceApp string, source string) bool {
	want, ok := requiredSurfacePackageApps()[strings.TrimSpace(referenceApp)]
	return ok && normalizeEvidencePath(source) == want
}

func requiredSurfacePackageApps() map[string]string {
	return map[string]string{
		"command-palette": "examples/surface/reference_core/surface_reference_command_palette.tetra",
		"localized-form":  "examples/surface/reference_forms/surface_reference_localized_form.tetra",
		"migration":       "examples/surface/reference_forms/surface_reference_migration.tetra",
		"studio-shell":    "examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra",
	}
}

func validateSurfacePackageArtifacts(packages []SurfacePackageArtifact) []string {
	if len(packages) == 0 {
		return []string{"packages are required"}
	}
	var issues []string
	seen := map[string]SurfacePackageArtifact{}
	for _, pkg := range packages {
		target := strings.TrimSpace(pkg.Target)
		if target == "" {
			issues = append(issues, "package target is required")
			continue
		}
		seen[target] = pkg
		prefix := "package " + target
		if !safeRelativeReportPath(pkg.Path) || !strings.HasSuffix(pkg.Path, ".tar.gz") {
			issues = append(issues, prefix+" path must be a safe .tar.gz report path")
		}
		if !safeRelativeReportPath(pkg.ManifestPath) ||
			!strings.HasSuffix(pkg.ManifestPath, ".json") {
			issues = append(issues, prefix+" manifest_path must be a safe JSON report path")
		}
		for _, digest := range []struct {
			name  string
			value string
		}{
			{name: "sha256", value: pkg.SHA256},
			{name: "asset_manifest_sha256", value: pkg.AssetManifestSHA256},
			{name: "source_sha256", value: pkg.SourceSHA256},
			{name: "build_sha256", value: pkg.BuildSHA256},
		} {
			if !validChecksumLike(digest.value) {
				issues = append(
					issues,
					fmt.Sprintf("%s %s must be sha256 evidence", prefix, digest.name),
				)
			}
		}
		if !pkg.LocalOnlyAssets {
			issues = append(issues, prefix+" local_only_assets must be true")
		}
		switch target {
		case "linux-x64":
			if pkg.Kind != "linux-x64-tar.gz" {
				issues = append(
					issues,
					fmt.Sprintf("%s kind is %q, want linux-x64-tar.gz", prefix, pkg.Kind),
				)
			}
			if !pkg.ContainsExecutable {
				issues = append(issues, prefix+" must contain executable")
			}
			if pkg.ContainsWebBundle {
				issues = append(issues, prefix+" must not be marked as web bundle")
			}
		case "wasm32-web":
			if pkg.Kind != "wasm32-web-tar.gz" {
				issues = append(
					issues,
					fmt.Sprintf("%s kind is %q, want wasm32-web-tar.gz", prefix, pkg.Kind),
				)
			}
			if !pkg.ContainsWebBundle {
				issues = append(issues, prefix+" must contain web bundle")
			}
		default:
			issues = append(issues, fmt.Sprintf("unsupported package target %q", target))
		}
		if !pkg.Pass {
			issues = append(issues, prefix+" pass must be true")
		}
	}
	for _, target := range []string{"linux-x64", "wasm32-web"} {
		if _, ok := seen[target]; !ok {
			issues = append(issues, "packages missing "+target)
		}
	}
	return issues
}

func validateSurfacePackageAssets(assets []SurfacePackageAsset) []string {
	if len(assets) == 0 {
		return []string{"assets are required"}
	}
	var issues []string
	for _, asset := range assets {
		prefix := "asset " + strings.TrimSpace(asset.Path)
		if !safeRelativeReportPath(asset.Path) {
			issues = append(issues, "asset path is unsafe or empty")
		}
		if strings.TrimSpace(asset.Kind) == "" {
			issues = append(issues, prefix+" kind is required")
		}
		if !validChecksumLike(asset.SHA256) {
			issues = append(issues, prefix+" sha256 must be sha256 evidence")
		}
		if asset.SizeBytes <= 0 {
			issues = append(issues, prefix+" size_bytes must be positive")
		}
		if !asset.LocalOnly {
			issues = append(issues, prefix+" local_only must be true")
		}
		if asset.NetworkFetchAllowed {
			issues = append(issues, prefix+" network_fetch_allowed must be false")
		}
		if !asset.Pass {
			issues = append(issues, prefix+" pass must be true")
		}
	}
	return issues
}

func validateSurfacePackageInstallSmokes(
	referenceApp string,
	smokes []SurfacePackageInstallSmoke,
) []string {
	var issues []string
	seenLinux := false
	for _, smoke := range smokes {
		prefix := "install smoke " + strings.TrimSpace(smoke.Target)
		if smoke.Target == "linux-x64" {
			seenLinux = true
		}
		if smoke.Target != "linux-x64" {
			issues = append(issues, prefix+" target must be linux-x64")
		}
		if !safeRelativeReportPath(smoke.PackagePath) ||
			!strings.HasSuffix(smoke.PackagePath, ".tar.gz") {
			issues = append(issues, prefix+" package_path must be a safe .tar.gz path")
		}
		if !safeRelativeReportPath(smoke.InstallDir) {
			issues = append(issues, prefix+" install_dir is unsafe or empty")
		}
		if !safeRelativeReportPath(smoke.InstalledBinary) {
			issues = append(issues, prefix+" installed_binary is unsafe or empty")
		}
		if !strings.Contains(smoke.Command, smoke.InstalledBinary) {
			issues = append(issues, prefix+" command must execute installed_binary")
		}
		if smoke.ExpectedExitCode < 0 {
			issues = append(
				issues,
				fmt.Sprintf(
					"%s expected_exit_code = %d, want non-negative",
					prefix,
					smoke.ExpectedExitCode,
				),
			)
		}
		if smoke.ExpectedExitCode != 0 {
			issues = append(
				issues,
				fmt.Sprintf(
					"%s expected_exit_code = %d, want 0 for Surface package evidence",
					prefix,
					smoke.ExpectedExitCode,
				),
			)
		}
		if smoke.ExitCode != smoke.ExpectedExitCode {
			issues = append(
				issues,
				fmt.Sprintf(
					"%s exit_code = %d, want expected_exit_code %d",
					prefix,
					smoke.ExitCode,
					smoke.ExpectedExitCode,
				),
			)
		}
		for _, check := range []struct {
			name string
			ok   bool
		}{
			{name: "artifact_hash_verified", ok: smoke.ArtifactHashVerified},
			{name: "package_manifest_verified", ok: smoke.PackageManifestVerified},
			{name: "app_run", ok: smoke.AppRun},
			{name: "pass", ok: smoke.Pass},
		} {
			if !check.ok {
				issues = append(issues, fmt.Sprintf("%s %s must be true", prefix, check.name))
			}
		}
	}
	if !seenLinux {
		issues = append(issues, "install_smokes missing linux-x64 install/run evidence")
	}
	return issues
}

func validateSurfacePackageWebBundles(bundles []SurfacePackageWebBundle) []string {
	var issues []string
	seenWeb := false
	for _, bundle := range bundles {
		prefix := "web bundle " + strings.TrimSpace(bundle.Target)
		if bundle.Target == "wasm32-web" {
			seenWeb = true
		}
		if bundle.Target != "wasm32-web" {
			issues = append(issues, prefix+" target must be wasm32-web")
		}
		if !safeRelativeReportPath(bundle.PackagePath) ||
			!strings.HasSuffix(bundle.PackagePath, ".tar.gz") {
			issues = append(issues, prefix+" package_path must be a safe .tar.gz path")
		}
		if !safeRelativeReportPath(bundle.WebEntry) ||
			!strings.HasSuffix(bundle.WebEntry, ".html") {
			issues = append(issues, prefix+" web_entry must be a safe HTML path")
		}
		if !safeRelativeReportPath(bundle.WASMArtifact) ||
			!strings.HasSuffix(bundle.WASMArtifact, ".wasm") {
			issues = append(issues, prefix+" wasm_artifact must be a safe .wasm path")
		}
		if !safeRelativeReportPath(bundle.LoaderArtifact) ||
			!strings.HasSuffix(bundle.LoaderArtifact, ".mjs") {
			issues = append(issues, prefix+" loader_artifact must be a safe .mjs path")
		}
		if !safeRelativeReportPath(bundle.BrowserCanvasHost) ||
			!strings.HasSuffix(bundle.BrowserCanvasHost, ".mjs") {
			issues = append(issues, prefix+" browser_canvas_host must be a safe .mjs path")
		}
		if !strings.Contains(bundle.Command, "tetra build") ||
			!strings.Contains(bundle.Command, "wasm32-web") {
			issues = append(issues, prefix+" command must build wasm32-web")
		}
		if !bundle.ArtifactHashVerified {
			issues = append(issues, prefix+" artifact_hash_verified must be true")
		}
		if !bundle.PackageManifestVerified {
			issues = append(issues, prefix+" package_manifest_verified must be true")
		}
		if !bundle.Pass {
			issues = append(issues, prefix+" pass must be true")
		}
	}
	if !seenWeb {
		issues = append(issues, "web_bundles missing wasm32-web bundle evidence")
	}
	return issues
}

func validateSurfacePackageUpdateStrategy(strategy SurfacePackageUpdateStrategy) []string {
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{
			field: "update_strategy.strategy",
			got:   strategy.Strategy,
			want:  "hash-pinned-channel-manifest-v1",
		},
		{
			field: "update_strategy.manifest_format",
			got:   strategy.ManifestFormat,
			want:  "tetra.surface.update-channel.v1",
		},
	} {
		if check.got != check.want {
			issues = append(
				issues,
				fmt.Sprintf("%s is %q, want %q", check.field, check.got, check.want),
			)
		}
	}
	for _, path := range []struct {
		name  string
		value string
	}{
		{name: "channel_manifest", value: strategy.ChannelManifest},
		{name: "latest_package_path", value: strategy.LatestPackagePath},
		{name: "rollback_manifest", value: strategy.RollbackManifest},
	} {
		if !safeRelativeReportPath(path.value) {
			issues = append(issues, fmt.Sprintf("update_strategy.%s is unsafe or empty", path.name))
		}
	}
	if strings.TrimSpace(strategy.CurrentVersion) == "" ||
		strings.TrimSpace(strategy.LatestVersion) == "" {
		issues = append(issues, "update_strategy current_version and latest_version are required")
	}
	if !validChecksumLike(strategy.LatestPackageSHA256) {
		issues = append(issues, "update_strategy.latest_package_sha256 must be sha256 evidence")
	}
	if !strategy.PackageHashPinned {
		issues = append(issues, "update_strategy.package_hash_pinned must be true")
	}
	if !strategy.SignatureRequiredForStablePromotion {
		issues = append(
			issues,
			"update_strategy.signature_required_for_stable_promotion must be true",
		)
	}
	if strategy.AutoUpdateRuntimeClaim {
		issues = append(
			issues,
			"update_strategy.auto_update_runtime_claim must be false without runtime updater evidence",
		)
	}
	if strategy.NetworkUpdateClaim {
		issues = append(
			issues,
			"update_strategy.network_update_claim must be false without network updater evidence",
		)
	}
	if !strategy.Pass {
		issues = append(issues, "update_strategy pass must be true")
	}
	return issues
}

func validateSurfacePackagePlatformProof(
	name string,
	proof SurfacePackagePlatformProof,
	notarization bool,
) []string {
	var issues []string
	if proof.Status != "nonclaim" {
		issues = append(issues, fmt.Sprintf("%s status is %q, want nonclaim", name, proof.Status))
	}
	if proof.Signed {
		issues = append(
			issues,
			fmt.Sprintf("%s must not claim signed package without platform signing evidence", name),
		)
	}
	if notarization && proof.Notarized {
		issues = append(
			issues,
			fmt.Sprintf("%s must not claim notarization without platform evidence", name),
		)
	}
	if !notarization && proof.Notarized {
		issues = append(issues, fmt.Sprintf("%s notarized must be false", name))
	}
	if proof.ProductionClaim {
		issues = append(issues, fmt.Sprintf("%s production_claim must be false", name))
	}
	if strings.TrimSpace(proof.Evidence) != "" {
		issues = append(issues, fmt.Sprintf("%s evidence must stay empty for nonclaim", name))
	}
	if strings.TrimSpace(proof.BlockedReason) == "" {
		issues = append(issues, fmt.Sprintf("%s blocked_reason is required", name))
	}
	return issues
}

func validateSurfacePackageNegativeGuards(guards SurfacePackageNegativeGuards) []string {
	var missing []string
	for _, check := range []struct {
		name string
		ok   bool
	}{
		{name: "no_react_runtime", ok: guards.NoReactRuntime},
		{name: "no_electron_runtime", ok: guards.NoElectronRuntime},
		{name: "no_dom_app_ui_tree", ok: guards.NoDOMAppUITree},
		{name: "no_css_runtime", ok: guards.NoCSSRuntime},
		{name: "no_user_js_app_logic", ok: guards.NoUserJSAppLogic},
		{name: "no_remote_asset_fetch", ok: guards.NoRemoteAssetFetch},
		{name: "no_unsigned_signing_claim", ok: guards.NoUnsignedSigningClaim},
		{
			name: "no_notarization_without_platform_evidence",
			ok:   guards.NoNotarizationWithoutPlatformEvidence,
		},
		{name: "no_auto_update_without_runtime_evidence", ok: guards.NoAutoUpdateWithoutRuntimeEvidence},
		{name: "no_docs_only_package_claim", ok: guards.NoDocsOnlyPackageClaim},
		{name: "install_run_required", ok: guards.InstallRunRequired},
		{name: "web_bundle_required", ok: guards.WebBundleRequired},
		{name: "artifact_hashes_required", ok: guards.ArtifactHashesRequired},
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

// ---- reference_apps.go ----

const ReferenceAppsSchemaV1 = "tetra.surface.reference-app-suite.v1"

type SurfaceReferenceAppsReport struct {
	Schema          string                             `json:"schema"`
	Model           string                             `json:"model"`
	ReleaseScope    string                             `json:"release_scope"`
	Producer        string                             `json:"producer"`
	AppCount        int                                `json:"app_count"`
	RequiredTargets []string                           `json:"required_targets"`
	Apps            []SurfaceReferenceAppReport        `json:"apps"`
	VisualEvidence  SurfaceReferenceAppsVisualEvidence `json:"visual_evidence"`
	NegativeGuards  SurfaceReferenceAppsNegativeGuards `json:"negative_guards"`
	Pass            bool                               `json:"pass"`
}

type SurfaceReferenceAppReport struct {
	Shape                 string                            `json:"shape"`
	Source                string                            `json:"source"`
	Module                string                            `json:"module"`
	Imports               []string                          `json:"imports"`
	Recipes               []string                          `json:"recipes"`
	BeautyCoverage        []string                          `json:"beauty_coverage"`
	StableMorphRecipes    bool                              `json:"stable_morph_recipes"`
	ResolvesToBlock       bool                              `json:"resolves_to_block"`
	Compiles              bool                              `json:"compiles"`
	Runs                  bool                              `json:"runs"`
	ExitCode              int                               `json:"exit_code"`
	TokenThemeConformance bool                              `json:"token_theme_conformance"`
	LayoutReport          bool                              `json:"layout_report"`
	InteractionTrace      bool                              `json:"interaction_trace"`
	AccessibilitySnapshot bool                              `json:"accessibility_snapshot"`
	PerformanceBudget     bool                              `json:"performance_budget"`
	ArtifactHashes        bool                              `json:"artifact_hashes"`
	CompatibilityWidgets  bool                              `json:"compatibility_widgets"`
	InfrastructureOnly    bool                              `json:"infrastructure_only"`
	NonProductReason      string                            `json:"non_product_reason,omitempty"`
	MorphToPixels         *MorphToPixelsChainReport         `json:"morph_to_pixels,omitempty"`
	Targets               []SurfaceReferenceAppTargetReport `json:"targets"`
}

type SurfaceReferenceAppTargetReport struct {
	Target                string `json:"target"`
	RuntimeReport         string `json:"runtime_report"`
	FrameChecksum         string `json:"frame_checksum"`
	VisualDiff            bool   `json:"visual_diff"`
	InteractionTrace      bool   `json:"interaction_trace"`
	AccessibilitySnapshot bool   `json:"accessibility_snapshot"`
	PerformanceBudget     bool   `json:"performance_budget"`
	Pass                  bool   `json:"pass"`
	ScreenshotOnly        bool   `json:"screenshot_only"`
}

type SurfaceReferenceAppsVisualEvidence struct {
	Path     string `json:"path"`
	Schema   string `json:"schema"`
	AppCount int    `json:"app_count"`
	Pass     bool   `json:"pass"`
}

type SurfaceReferenceAppsNegativeGuards struct {
	ScreenshotOnlyRejected            bool `json:"screenshot_only_rejected"`
	MissingInteractionRejected        bool `json:"missing_interaction_rejected"`
	MissingAccessibilityRejected      bool `json:"missing_accessibility_rejected"`
	MissingPerformanceRejected        bool `json:"missing_performance_rejected"`
	CoreWidgetUsageRejected           bool `json:"core_widget_usage_rejected"`
	MigrationWidgetsCompatibilityOnly bool `json:"migration_widgets_compatibility_only"`
	NoReactRuntime                    bool `json:"no_react_runtime"`
	NoElectronRuntime                 bool `json:"no_electron_runtime"`
	NoDOMAppUITree                    bool `json:"no_dom_app_ui_tree"`
	NoCSSRuntime                      bool `json:"no_css_runtime"`
	NoUserJSAppLogic                  bool `json:"no_user_js_app_logic"`
}

func ValidateReferenceAppsReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != ReferenceAppsSchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, ReferenceAppsSchemaV1)
	}
	var report SurfaceReferenceAppsReport
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}
	issues := validateSurfaceReferenceAppsReport(report)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateSurfaceReferenceAppsReport(report SurfaceReferenceAppsReport) []string {
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: report.Schema, want: ReferenceAppsSchemaV1},
		{field: "model", got: report.Model, want: "surface-reference-app-suite-v1"},
		{field: "release_scope", got: report.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
		{
			field: "producer",
			got:   report.Producer,
			want:  "scripts/release/surface/surface-reference-apps-smoke.sh",
		},
	} {
		if check.got != check.want {
			issues = append(
				issues,
				fmt.Sprintf("%s is %q, want %q", check.field, check.got, check.want),
			)
		}
	}
	if report.AppCount != len(report.Apps) {
		issues = append(
			issues,
			fmt.Sprintf("app_count = %d, want len(apps) %d", report.AppCount, len(report.Apps)),
		)
	}
	issues = append(issues, validateSurfaceReferenceTargets(report.RequiredTargets)...)
	issues = append(issues, validateSurfaceReferenceApps(report.Apps, report.RequiredTargets)...)
	issues = append(
		issues,
		validateSurfaceReferenceVisualEvidence(
			report.VisualEvidence,
			len(requiredSurfaceReferenceApps()),
		)...)
	issues = append(issues, validateSurfaceReferenceNegativeGuards(report.NegativeGuards)...)
	if !report.Pass {
		issues = append(issues, "pass must be true")
	}
	return issues
}

func validateSurfaceReferenceTargets(targets []string) []string {
	var issues []string
	for _, target := range requiredSurfaceReferenceTargets() {
		if !templateSmokeContainsString(targets, target) {
			issues = append(issues, fmt.Sprintf("required_targets missing %s", target))
		}
	}
	return issues
}

func validateSurfaceReferenceApps(
	apps []SurfaceReferenceAppReport,
	requiredTargets []string,
) []string {
	required := requiredSurfaceReferenceApps()
	seen := map[string]SurfaceReferenceAppReport{}
	var issues []string
	for _, app := range apps {
		shape := strings.TrimSpace(app.Shape)
		if shape == "" {
			issues = append(issues, "apps shape is required")
			continue
		}
		if _, ok := seen[shape]; ok {
			issues = append(issues, fmt.Sprintf("duplicate reference app shape %s", shape))
		}
		seen[shape] = app
		if wantSource, ok := required[shape]; ok &&
			normalizeEvidencePath(app.Source) != wantSource {
			issues = append(
				issues,
				fmt.Sprintf("%s source is %q, want %s", shape, app.Source, wantSource),
			)
		}
		issues = append(issues, validateSurfaceReferenceApp(app, requiredTargets)...)
	}
	for shape := range required {
		if _, ok := seen[shape]; !ok {
			issues = append(issues, fmt.Sprintf("reference suite missing %s", shape))
		}
	}
	issues = append(issues, validateSurfaceReferenceBeautyCoverage(apps)...)
	if len(apps) != len(required) {
		issues = append(issues, fmt.Sprintf("apps length = %d, want %d", len(apps), len(required)))
	}
	return issues
}

func validateSurfaceReferenceApp(app SurfaceReferenceAppReport, requiredTargets []string) []string {
	shape := strings.TrimSpace(app.Shape)
	prefix := "reference app " + shape
	var issues []string
	if !safeRelativeSourcePath(app.Source) {
		issues = append(issues, prefix+" source must be a safe Tetra source path")
	}
	if strings.TrimSpace(app.Module) == "" {
		issues = append(issues, prefix+" module is required")
	}
	for _, required := range []string{"lib.core.surface", "lib.core.block", "lib.core.morph"} {
		if !templateSmokeContainsString(app.Imports, required) {
			issues = append(issues, fmt.Sprintf("%s imports missing %s", prefix, required))
		}
	}
	if app.CompatibilityWidgets && shape != "migration" {
		issues = append(
			issues,
			fmt.Sprintf("%s compatibility_widgets may be true only for migration", prefix),
		)
	}
	if shape == "migration" && !app.CompatibilityWidgets {
		issues = append(issues, "reference app migration requires compatibility_widgets evidence")
	}
	for _, imported := range app.Imports {
		lower := strings.ToLower(imported)
		if strings.Contains(lower, "lib.core.widgets") && shape != "migration" {
			issues = append(
				issues,
				fmt.Sprintf("%s imports widgets outside migration compatibility example", prefix),
			)
		}
		for _, forbidden := range []string{
			"react",
			"electron",
			"dom",
			"css",
			"javascript",
			"platform_widget",
			"native_widget",
		} {
			if strings.Contains(lower, forbidden) {
				issues = append(
					issues,
					fmt.Sprintf("%s imports forbidden runtime %q", prefix, imported),
				)
			}
		}
	}
	if len(app.Recipes) < 4 || !app.StableMorphRecipes || !app.ResolvesToBlock {
		issues = append(
			issues,
			prefix+" requires at least four stable Morph recipes that resolve to Block",
		)
	}
	if !app.InfrastructureOnly && len(app.BeautyCoverage) == 0 {
		issues = append(issues, prefix+" beauty_coverage is required for product reference apps")
	}
	if !app.Compiles || !app.Runs || app.ExitCode != 0 {
		issues = append(
			issues,
			fmt.Sprintf("%s compile/run evidence must pass with exit 0", prefix),
		)
	}
	for _, check := range []struct {
		name string
		ok   bool
	}{
		{name: "token_theme_conformance", ok: app.TokenThemeConformance},
		{name: "layout_report", ok: app.LayoutReport},
		{name: "interaction_trace", ok: app.InteractionTrace},
		{name: "accessibility_snapshot", ok: app.AccessibilitySnapshot},
		{name: "performance_budget", ok: app.PerformanceBudget},
		{name: "artifact_hashes", ok: app.ArtifactHashes},
	} {
		if !check.ok {
			issues = append(issues, fmt.Sprintf("%s %s must be true", prefix, check.name))
		}
	}
	issues = append(issues, validateSurfaceReferenceMorphToPixels(prefix, app)...)
	issues = append(
		issues,
		validateSurfaceReferenceAppTargets(prefix, app.Targets, requiredTargets)...)
	return issues
}

func validateSurfaceReferenceBeautyCoverage(apps []SurfaceReferenceAppReport) []string {
	covered := map[string]bool{}
	for _, app := range apps {
		if app.InfrastructureOnly {
			continue
		}
		for _, item := range app.BeautyCoverage {
			covered[strings.ToLower(strings.TrimSpace(item))] = true
		}
	}
	var missing []string
	for _, required := range []string{
		"command-palette",
		"dashboard",
		"settings",
		"editor-shell",
		"elevated-panel",
		"focus-state",
		"disabled-state",
	} {
		if !covered[required] {
			missing = append(missing, required)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return []string{
		fmt.Sprintf("reference beauty coverage missing %s", strings.Join(missing, ", ")),
	}
}

func validateSurfaceReferenceMorphToPixels(prefix string, app SurfaceReferenceAppReport) []string {
	if app.InfrastructureOnly {
		if strings.TrimSpace(app.NonProductReason) == "" {
			return []string{prefix + " infrastructure_only requires non_product_reason"}
		}
		if app.MorphToPixels != nil {
			return []string{
				prefix + " infrastructure_only must not provide product morph_to_pixels evidence",
			}
		}
		return nil
	}
	if strings.TrimSpace(app.NonProductReason) != "" {
		return []string{prefix + " non_product_reason is allowed only for infrastructure_only apps"}
	}
	if app.MorphToPixels == nil {
		return []string{
			prefix + " morph_to_pixels is required or the app must be marked infrastructure_only",
		}
	}
	var issues []string
	issues = append(
		issues,
		validateMorphToPixelsChain(prefix+" morph_to_pixels", *app.MorphToPixels, app.Source)...)
	if !safeRelativeReportPath(app.MorphToPixels.ReportPath) {
		issues = append(issues, prefix+" morph_to_pixels.report_path is unsafe or empty")
	}
	if !safeRelativeReportPath(app.MorphToPixels.FrameArtifact) {
		issues = append(issues, prefix+" morph_to_pixels.frame_artifact is unsafe or empty")
	}
	if !safeRelativeReportPath(app.MorphToPixels.GoldenArtifact) {
		issues = append(issues, prefix+" morph_to_pixels.golden_artifact is unsafe or empty")
	}
	if app.MorphToPixels.ProductClaim || app.MorphToPixels.FinalSignoff {
		issues = append(
			issues,
			prefix+" morph_to_pixels reference evidence must not assert product_claim or final_signoff",
		)
	}
	return issues
}

func validateSurfaceReferenceAppTargets(
	prefix string,
	targets []SurfaceReferenceAppTargetReport,
	requiredTargets []string,
) []string {
	seen := map[string]SurfaceReferenceAppTargetReport{}
	var issues []string
	for _, target := range targets {
		name := strings.TrimSpace(target.Target)
		if name == "" {
			issues = append(issues, prefix+" target name is required")
			continue
		}
		seen[name] = target
		if !safeRelativeReportPath(target.RuntimeReport) {
			issues = append(
				issues,
				fmt.Sprintf("%s %s runtime_report is unsafe or empty", prefix, name),
			)
		}
		if !validChecksumLike(target.FrameChecksum) {
			issues = append(
				issues,
				fmt.Sprintf("%s %s frame_checksum must be sha256 evidence", prefix, name),
			)
		}
		if target.ScreenshotOnly {
			issues = append(
				issues,
				fmt.Sprintf("%s %s screenshot-only evidence is not sufficient", prefix, name),
			)
		}
		for _, check := range []struct {
			name string
			ok   bool
		}{
			{name: "visual_diff", ok: target.VisualDiff},
			{name: "interaction_trace", ok: target.InteractionTrace},
			{name: "accessibility_snapshot", ok: target.AccessibilitySnapshot},
			{name: "performance_budget", ok: target.PerformanceBudget},
			{name: "pass", ok: target.Pass},
		} {
			if !check.ok {
				issues = append(
					issues,
					fmt.Sprintf("%s %s %s must be true", prefix, name, check.name),
				)
			}
		}
	}
	for _, target := range requiredTargets {
		if !templateSmokeContainsString(requiredSurfaceReferenceTargets(), target) {
			continue
		}
		if _, ok := seen[target]; !ok {
			issues = append(issues, fmt.Sprintf("%s missing required target %s", prefix, target))
		}
	}
	return issues
}

func validateSurfaceReferenceVisualEvidence(
	evidence SurfaceReferenceAppsVisualEvidence,
	appCount int,
) []string {
	var issues []string
	if !safeRelativeReportPath(evidence.Path) {
		issues = append(issues, "visual_evidence.path is unsafe or empty")
	}
	if evidence.Schema != VisualRegressionSchemaV1 {
		issues = append(
			issues,
			fmt.Sprintf(
				"visual_evidence.schema is %q, want %s",
				evidence.Schema,
				VisualRegressionSchemaV1,
			),
		)
	}
	if evidence.AppCount != appCount {
		issues = append(
			issues,
			fmt.Sprintf("visual_evidence.app_count = %d, want %d", evidence.AppCount, appCount),
		)
	}
	if !evidence.Pass {
		issues = append(issues, "visual_evidence pass must be true")
	}
	return issues
}

func validateSurfaceReferenceNegativeGuards(guards SurfaceReferenceAppsNegativeGuards) []string {
	var missing []string
	for _, check := range []struct {
		name string
		ok   bool
	}{
		{name: "screenshot_only_rejected", ok: guards.ScreenshotOnlyRejected},
		{name: "missing_interaction_rejected", ok: guards.MissingInteractionRejected},
		{name: "missing_accessibility_rejected", ok: guards.MissingAccessibilityRejected},
		{name: "missing_performance_rejected", ok: guards.MissingPerformanceRejected},
		{name: "core_widget_usage_rejected", ok: guards.CoreWidgetUsageRejected},
		{name: "migration_widgets_compatibility_only", ok: guards.MigrationWidgetsCompatibilityOnly},
		{name: "no_react_runtime", ok: guards.NoReactRuntime},
		{name: "no_electron_runtime", ok: guards.NoElectronRuntime},
		{name: "no_dom_app_ui_tree", ok: guards.NoDOMAppUITree},
		{name: "no_css_runtime", ok: guards.NoCSSRuntime},
		{name: "no_user_js_app_logic", ok: guards.NoUserJSAppLogic},
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

func requiredSurfaceReferenceApps() map[string]string {
	return map[string]string{
		"command-palette": "examples/surface/reference_core/surface_reference_command_palette.tetra",
		"settings":        "examples/surface/reference_core/surface_reference_settings.tetra",
		"dashboard":       "examples/surface/reference_core/surface_reference_dashboard.tetra",
		"editor-shell":    "examples/surface/reference_core/surface_reference_editor_shell.tetra",
		"file-manager":    "examples/surface/reference_core/surface_reference_file_manager.tetra",
		"dialog-notification": ("examples/surface/reference_core/surface_reference_dialog_" +
			"notification.tetra"),
		"localized-form": "examples/surface/reference_forms/surface_reference_localized_form.tetra",
		"accessibility-form": ("examples/surface/reference_forms/surface_reference_" +
			"accessibility_form.tetra"),
		"multi-window-notes": ("examples/surface/reference_forms/surface_reference_multi_" +
			"window_notes.tetra"),
		"migration": "examples/surface/reference_forms/surface_reference_migration.tetra",
	}
}

func requiredSurfaceReferenceTargets() []string {
	return []string{"headless", "linux-x64-real-window", "wasm32-web-browser-canvas"}
}

// ---- render_command_stream_validation.go ----

type RenderCommandStreamReport struct {
	Schema                        string                `json:"schema"`
	Source                        string                `json:"source"`
	SurfaceScope                  string                `json:"surface_scope"`
	Producer                      string                `json:"producer"`
	QualityLevel                  string                `json:"quality_level"`
	Renderer                      string                `json:"renderer"`
	DerivedFromBlockSceneSnapshot bool                  `json:"derived_from_block_scene_snapshot"`
	BlockSceneHash                string                `json:"block_scene_hash"`
	FrameChecksum                 string                `json:"frame_checksum"`
	CommandStreamHash             string                `json:"command_stream_hash"`
	CommandCount                  int                   `json:"command_count"`
	SourceLinked                  bool                  `json:"source_linked"`
	HandcraftedFixture            bool                  `json:"handcrafted_fixture"`
	Commands                      []RenderCommandReport `json:"commands"`
}

type RenderCommandReport struct {
	Order          int        `json:"order"`
	Command        string     `json:"command"`
	Source         string     `json:"source"`
	SourceNodeID   string     `json:"source_node_id"`
	Recipe         string     `json:"recipe"`
	LayerID        string     `json:"layer_id"`
	BlockID        int        `json:"block_id"`
	Rect           RectReport `json:"rect"`
	Clip           RectReport `json:"clip,omitempty"`
	Color          string     `json:"color,omitempty"`
	Radius         int        `json:"radius,omitempty"`
	Width          int        `json:"width,omitempty"`
	Blur           int        `json:"blur,omitempty"`
	OffsetX        int        `json:"offset_x,omitempty"`
	OffsetY        int        `json:"offset_y,omitempty"`
	Opacity        int        `json:"opacity,omitempty"`
	Quality        string     `json:"quality"`
	AssetID        string     `json:"asset_id,omitempty"`
	TextLen        int        `json:"text_len,omitempty"`
	RasterFormat   string     `json:"raster_format,omitempty"`
	RasterHash     string     `json:"raster_hash,omitempty"`
	RasterWidth    int        `json:"raster_width,omitempty"`
	RasterHeight   int        `json:"raster_height,omitempty"`
	RasterCoverage int        `json:"raster_coverage,omitempty"`
	MarkerOnly     bool       `json:"marker_only,omitempty"`
	Checksum       string     `json:"checksum"`
}

func validateRenderCommandStreamEvidence(report Report) []string {
	if report.RenderCommandStream == nil {
		return nil
	}

	stream := report.RenderCommandStream
	var issues []string
	if stream.Schema != "tetra.surface.render-command-stream.v1" {
		issues = append(
			issues,
			fmt.Sprintf(
				"render_command_stream schema is %q, want tetra.surface.render-command-stream.v1",
				stream.Schema,
			),
		)
	}
	if normalizeEvidencePath(stream.Source) != normalizeEvidencePath(report.Source) {
		issues = append(
			issues,
			fmt.Sprintf(
				"render_command_stream source %q must match report source %q",
				stream.Source,
				report.Source,
			),
		)
	}
	if stream.SurfaceScope != "surface-morph-rendered-beauty-linux-web" {
		issues = append(
			issues,
			fmt.Sprintf(
				"render_command_stream surface_scope is %q, want surface-morph-rendered-beauty-linux-web",
				stream.SurfaceScope,
			),
		)
	}
	if strings.TrimSpace(stream.Producer) == "" {
		issues = append(issues, "render_command_stream producer is required")
	}
	if stream.QualityLevel != "deterministic-render-command-stream-v1" {
		issues = append(
			issues,
			fmt.Sprintf(
				"render_command_stream quality_level is %q, want deterministic-render-command-stream-v1",
				stream.QualityLevel,
			),
		)
	}
	if !stringSliceContainsFold(
		[]string{"software-rgba-headless", "wayland-shm-rgba", "browser-canvas-rgba"},
		stream.Renderer,
	) {
		issues = append(
			issues,
			fmt.Sprintf("render_command_stream renderer %q is not allowed", stream.Renderer),
		)
	}
	if !stream.DerivedFromBlockSceneSnapshot {
		issues = append(
			issues,
			"render_command_stream derived_from_block_scene_snapshot must be true",
		)
	}
	if report.BlockSceneSnapshot == nil {
		issues = append(issues, "render_command_stream requires block_scene_snapshot evidence")
	} else if stream.BlockSceneHash != report.BlockSceneSnapshot.BlockSceneHash {
		issues = append(
			issues,
			"render_command_stream block_scene_hash must match block_scene_snapshot.block_scene_hash",
		)
	}
	if !validSHA256Digest(stream.BlockSceneHash) {
		issues = append(issues, "render_command_stream block_scene_hash must be sha256 evidence")
	}
	if !validChecksumLike(stream.FrameChecksum) {
		issues = append(issues, "render_command_stream frame_checksum must be sha256 evidence")
	} else if !frameChecksumPresent(report.Frames, stream.FrameChecksum) {
		issues = append(
			issues,
			"render_command_stream frame_checksum must match a presented report frame",
		)
	}
	if !validSHA256Digest(stream.CommandStreamHash) {
		issues = append(issues, "render_command_stream command_stream_hash must be sha256 evidence")
	}
	if stream.CommandCount <= 0 {
		issues = append(issues, "render_command_stream command_count must be positive")
	}
	if stream.CommandCount != len(stream.Commands) {
		issues = append(
			issues,
			fmt.Sprintf(
				"render_command_stream command_count = %d, want len(commands) %d",
				stream.CommandCount,
				len(stream.Commands),
			),
		)
	}
	if !stream.SourceLinked {
		issues = append(issues, "render_command_stream source_linked must be true")
	}
	if stream.HandcraftedFixture {
		issues = append(issues, "render_command_stream handcrafted_fixture must be false")
	}
	issues = append(issues, validateRenderCommands(report, stream.Commands)...)
	return issues
}

func validateRenderCommands(report Report, commands []RenderCommandReport) []string {
	var issues []string
	nodeByID := map[int]BlockSceneNodeReport{}
	if report.BlockSceneSnapshot != nil {
		for _, node := range report.BlockSceneSnapshot.Nodes {
			nodeByID[node.BlockID] = node
		}
	}
	seenKinds := map[string]bool{}
	lastOrder := 0
	for i, command := range commands {
		name := normalizeRenderCommandToken(command.Command)
		if command.Order != i+1 {
			issues = append(
				issues,
				fmt.Sprintf(
					"render_command_stream commands[%d] order = %d, want %d",
					i,
					command.Order,
					i+1,
				),
			)
		}
		if command.Order <= lastOrder {
			issues = append(
				issues,
				fmt.Sprintf(
					"render_command_stream command order %d is not strictly greater than previous order %d",
					command.Order,
					lastOrder,
				),
			)
		}
		lastOrder = command.Order
		if !isSupportedRenderCommand(name) {
			issues = append(
				issues,
				fmt.Sprintf(
					"render_command_stream commands[%d] command %q is not supported",
					i,
					command.Command,
				),
			)
		}
		if normalizeEvidencePath(command.Source) != normalizeEvidencePath(report.Source) {
			issues = append(
				issues,
				fmt.Sprintf(
					"render_command_stream commands[%d] source %q must match report source %q",
					i,
					command.Source,
					report.Source,
				),
			)
		}
		if command.BlockID <= 0 {
			issues = append(
				issues,
				fmt.Sprintf("render_command_stream commands[%d] block_id must be positive", i),
			)
		}
		node, hasNode := nodeByID[command.BlockID]
		if len(nodeByID) > 0 && !hasNode {
			issues = append(
				issues,
				fmt.Sprintf(
					"render_command_stream commands[%d] block_id %d is not in block_scene_snapshot",
					i,
					command.BlockID,
				),
			)
		}
		if strings.TrimSpace(command.SourceNodeID) == "" {
			issues = append(
				issues,
				fmt.Sprintf("render_command_stream commands[%d] source_node_id is required", i),
			)
		} else if hasNode && command.SourceNodeID != fmt.Sprintf("block:%d", node.BlockID) {
			issues = append(
				issues,
				fmt.Sprintf(
					"render_command_stream commands[%d] source_node_id %q must identify block:%d",
					i,
					command.SourceNodeID,
					node.BlockID,
				),
			)
		}
		if strings.TrimSpace(command.Recipe) == "" {
			issues = append(
				issues,
				fmt.Sprintf("render_command_stream commands[%d] recipe is required", i),
			)
		} else if hasNode && command.Recipe != node.Recipe {
			issues = append(
				issues,
				fmt.Sprintf(
					"render_command_stream commands[%d] recipe %q must match block_scene_snapshot node recipe %q",
					i,
					command.Recipe,
					node.Recipe,
				),
			)
		}
		if strings.TrimSpace(command.LayerID) == "" {
			issues = append(
				issues,
				fmt.Sprintf("render_command_stream commands[%d] layer_id is required", i),
			)
		}
		if command.Rect.W <= 0 || command.Rect.H <= 0 {
			issues = append(
				issues,
				fmt.Sprintf(
					"render_command_stream commands[%d] rect dimensions must be positive",
					i,
				),
			)
		}
		if command.Opacity < 0 || command.Opacity > 255 {
			issues = append(
				issues,
				fmt.Sprintf("render_command_stream commands[%d] opacity must be 0..255", i),
			)
		}
		if name != "radius_clip" && strings.TrimSpace(command.Color) == "" {
			issues = append(
				issues,
				fmt.Sprintf(
					"render_command_stream commands[%d] color is required for renderer-owned pixels",
					i,
				),
			)
		}
		if name == "radius_clip" && command.Radius <= 0 {
			issues = append(
				issues,
				fmt.Sprintf(
					"render_command_stream commands[%d] radius_clip radius must be positive",
					i,
				),
			)
		}
		if strings.TrimSpace(command.Quality) == "" {
			issues = append(
				issues,
				fmt.Sprintf("render_command_stream commands[%d] quality is required", i),
			)
		}
		if name == "text" {
			issues = append(issues, validateRasterProof(
				fmt.Sprintf("render_command_stream commands[%d]", i),
				"builtin-5x7-alpha-mask-v1",
				command.RasterFormat,
				command.RasterHash,
				command.RasterWidth,
				command.RasterHeight,
				command.RasterCoverage,
				command.MarkerOnly,
			)...)
		}
		if name == "icon" {
			issues = append(issues, validateRasterProof(
				fmt.Sprintf("render_command_stream commands[%d]", i),
				"builtin-icon-mask-raster-v1",
				command.RasterFormat,
				command.RasterHash,
				command.RasterWidth,
				command.RasterHeight,
				command.RasterCoverage,
				command.MarkerOnly,
			)...)
		}
		if !validChecksumLike(command.Checksum) {
			issues = append(
				issues,
				fmt.Sprintf(
					"render_command_stream commands[%d] checksum must be sha256 evidence",
					i,
				),
			)
		}
		seenKinds[name] = true
	}
	for _, required := range []string{
		"fill",
		"gradient",
		"image_fill",
		"border",
		"radius_clip",
		"shadow",
		"overlay",
		"outline",
		"text",
		"icon",
	} {
		if !seenKinds[required] {
			issues = append(
				issues,
				fmt.Sprintf("render_command_stream commands require %s command", required),
			)
		}
	}
	return issues
}

func normalizeRenderCommandToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func isSupportedRenderCommand(value string) bool {
	switch normalizeRenderCommandToken(value) {
	case "fill",
		"gradient",
		"image_fill",
		"border",
		"radius_clip",
		"shadow",
		"overlay",
		"outline",
		"text",
		"icon":
		return true
	default:
		return false
	}
}

func frameChecksumPresent(frames []FrameReport, checksum string) bool {
	checksum = strings.TrimSpace(checksum)
	for _, frame := range frames {
		if frame.Presented && strings.TrimSpace(frame.Checksum) == checksum {
			return true
		}
	}
	return false
}

// ---- template_smoke.go ----

const TemplateSmokeSchemaV1 = "tetra.surface.template-smoke.v1"

type SurfaceTemplateSmokeReport struct {
	Schema            string                             `json:"schema"`
	Model             string                             `json:"model"`
	ReleaseScope      string                             `json:"release_scope"`
	Producer          string                             `json:"producer"`
	Command           string                             `json:"command"`
	TemplateCount     int                                `json:"template_count"`
	Templates         []SurfaceTemplateSmokeTemplate     `json:"templates"`
	InspectorEvidence SurfaceTemplateSmokeInspector      `json:"inspector_evidence"`
	VisualEvidence    SurfaceTemplateSmokeVisual         `json:"visual_evidence"`
	MorphToPixels     *MorphToPixelsChainReport          `json:"morph_to_pixels,omitempty"`
	PackageEvidence   []SurfaceTemplateSmokePackage      `json:"package_evidence"`
	NegativeGuards    SurfaceTemplateSmokeNegativeGuards `json:"negative_guards"`
	Pass              bool                               `json:"pass"`
}

type SurfaceTemplateSmokeTemplate struct {
	Kind             string                         `json:"kind"`
	ProjectDir       string                         `json:"project_dir"`
	Source           string                         `json:"source"`
	Capsule          string                         `json:"capsule"`
	TemplateMetadata string                         `json:"template_metadata"`
	Targets          []string                       `json:"targets"`
	Imports          []string                       `json:"imports"`
	RecipeCount      int                            `json:"recipe_count"`
	BlockMorphOnly   bool                           `json:"block_morph_only"`
	UsesAppShell     bool                           `json:"uses_app_shell"`
	WebCanvas        bool                           `json:"web_canvas"`
	Commands         []SurfaceTemplateSmokeCommand  `json:"commands"`
	SourceScan       SurfaceTemplateSmokeSourceScan `json:"source_scan"`
}

type SurfaceTemplateSmokeCommand struct {
	Kind     string `json:"kind"`
	Command  string `json:"command"`
	Pass     bool   `json:"pass"`
	ExitCode int    `json:"exit_code"`
}

type SurfaceTemplateSmokeSourceScan struct {
	ReactImport     bool `json:"react_import"`
	ElectronImport  bool `json:"electron_import"`
	DOMAppUITree    bool `json:"dom_app_ui_tree"`
	CSSRuntime      bool `json:"css_runtime"`
	CoreWidgets     bool `json:"core_widgets"`
	PlatformWidgets bool `json:"platform_widgets"`
	UserJSAppLogic  bool `json:"user_js_app_logic"`
	Pass            bool `json:"pass"`
}

type SurfaceTemplateSmokeInspector struct {
	Path  string `json:"path"`
	Model string `json:"model"`
	Pass  bool   `json:"pass"`
}

type SurfaceTemplateSmokeVisual struct {
	Path   string `json:"path"`
	Schema string `json:"schema"`
	Pass   bool   `json:"pass"`
}

type SurfaceTemplateSmokePackage struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	SHA256 string `json:"sha256"`
	Pass   bool   `json:"pass"`
}

type SurfaceTemplateSmokeNegativeGuards struct {
	NoReactImport          bool `json:"no_react_import"`
	NoElectronImport       bool `json:"no_electron_import"`
	NoDOMAppUITree         bool `json:"no_dom_app_ui_tree"`
	NoCSSRuntime           bool `json:"no_css_runtime"`
	NoCoreWidgets          bool `json:"no_core_widgets"`
	NoPlatformWidgets      bool `json:"no_platform_widgets"`
	NoUserJSAppLogic       bool `json:"no_user_js_app_logic"`
	CookbookUsesBlockMorph bool `json:"cookbook_uses_block_morph"`
}

func ValidateTemplateSmokeReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != TemplateSmokeSchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, TemplateSmokeSchemaV1)
	}

	var report SurfaceTemplateSmokeReport
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}

	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: report.Schema, want: TemplateSmokeSchemaV1},
		{field: "model", got: report.Model, want: "surface-template-smoke-v1"},
		{field: "release_scope", got: report.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
		{
			field: "producer",
			got:   report.Producer,
			want:  "scripts/release/surface/surface-template-smoke.sh",
		},
		{field: "command", got: report.Command, want: "tetra new surface-app"},
	} {
		if check.got != check.want {
			issues = append(
				issues,
				fmt.Sprintf("%s is %q, want %q", check.field, check.got, check.want),
			)
		}
	}
	issues = append(
		issues,
		validateSurfaceTemplateSmokeTemplates(report.TemplateCount, report.Templates)...)
	issues = append(issues, validateSurfaceTemplateSmokeInspector(report.InspectorEvidence)...)
	issues = append(issues, validateSurfaceTemplateSmokeVisual(report.VisualEvidence)...)
	issues = append(
		issues,
		validateSurfaceTemplateSmokeMorphToPixels(report.MorphToPixels, report.Templates)...)
	issues = append(issues, validateSurfaceTemplateSmokePackages(report.PackageEvidence)...)
	issues = append(issues, validateSurfaceTemplateSmokeNegativeGuards(report.NegativeGuards)...)
	if !report.Pass {
		issues = append(issues, "pass must be true")
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateSurfaceTemplateSmokeTemplates(
	templateCount int,
	templates []SurfaceTemplateSmokeTemplate,
) []string {
	required := []string{
		"command-palette",
		"settings",
		"dashboard",
		"editor-shell",
		"studio-shell",
		"multi-window-notes",
		"web-canvas",
	}
	if templateCount != len(templates) {
		return []string{
			fmt.Sprintf(
				"template_count = %d, want len(templates) %d",
				templateCount,
				len(templates),
			),
		}
	}
	var issues []string
	if templateCount != len(required) {
		issues = append(
			issues,
			fmt.Sprintf("template_count = %d, want %d", templateCount, len(required)),
		)
	}
	seen := map[string]bool{}
	for _, tmpl := range templates {
		kind := strings.TrimSpace(tmpl.Kind)
		if kind == "" {
			issues = append(issues, "templates kind is required")
			continue
		}
		if seen[kind] {
			issues = append(issues, fmt.Sprintf("duplicate template kind %s", kind))
		}
		seen[kind] = true
		if !safeRelativeReportPath(tmpl.ProjectDir) {
			issues = append(issues, fmt.Sprintf("%s project_dir is unsafe or empty", kind))
		}
		if !safeRelativeSourcePath(tmpl.Source) {
			issues = append(issues, fmt.Sprintf("%s source is unsafe or not a Tetra source", kind))
		}
		if !safeRelativeReportPath(tmpl.Capsule) || !strings.HasSuffix(tmpl.Capsule, "Capsule.t4") {
			issues = append(issues, fmt.Sprintf("%s capsule path must be safe Capsule.t4", kind))
		}
		if !safeRelativeReportPath(tmpl.TemplateMetadata) ||
			!strings.HasSuffix(tmpl.TemplateMetadata, "surface-template.json") {
			issues = append(
				issues,
				fmt.Sprintf("%s template_metadata must be safe surface-template.json", kind),
			)
		}
		issues = append(issues, validateSurfaceTemplateTargets(kind, tmpl.Targets)...)
		issues = append(issues, validateSurfaceTemplateImports(kind, tmpl.Imports)...)
		if tmpl.RecipeCount < 4 {
			issues = append(issues, fmt.Sprintf("%s recipe_count must be at least 4", kind))
		}
		if !tmpl.BlockMorphOnly {
			issues = append(issues, fmt.Sprintf("%s block_morph_only must be true", kind))
		}
		if (kind == "multi-window-notes" || kind == "studio-shell") && !tmpl.UsesAppShell {
			issues = append(issues, fmt.Sprintf("%s uses_app_shell must be true", kind))
		}
		if kind == "web-canvas" && !tmpl.WebCanvas {
			issues = append(issues, "web-canvas web_canvas must be true")
		}
		issues = append(issues, validateSurfaceTemplateCommands(kind, tmpl.Commands)...)
		issues = append(issues, validateSurfaceTemplateSourceScan(kind, tmpl.SourceScan)...)
	}
	for _, kind := range required {
		if !seen[kind] {
			issues = append(issues, fmt.Sprintf("templates missing %s", kind))
		}
	}
	return issues
}

func validateSurfaceTemplateTargets(kind string, targets []string) []string {
	var issues []string
	if !templateSmokeContainsString(targets, "linux-x64") {
		issues = append(issues, fmt.Sprintf("%s targets missing linux-x64", kind))
	}
	if !templateSmokeContainsString(targets, "wasm32-web") {
		issues = append(issues, fmt.Sprintf("%s targets missing wasm32-web", kind))
	}
	return issues
}

func validateSurfaceTemplateImports(kind string, imports []string) []string {
	var issues []string
	for _, required := range []string{"lib.core.surface", "lib.core.block", "lib.core.morph"} {
		if !templateSmokeContainsString(imports, required) {
			issues = append(issues, fmt.Sprintf("%s imports missing %s", kind, required))
		}
	}
	if (kind == "multi-window-notes" || kind == "studio-shell") &&
		!templateSmokeContainsString(imports, "lib.core.surface_app_shell") {
		issues = append(issues, fmt.Sprintf("%s imports missing lib.core.surface_app_shell", kind))
	}
	for _, imported := range imports {
		lower := strings.ToLower(imported)
		for _, forbidden := range []string{
			"react",
			"electron",
			"dom",
			"css",
			"javascript",
			"lib.core.widgets",
			"lib.core.component",
			"platform_widget",
			"native_widget",
		} {
			if strings.Contains(lower, forbidden) {
				issues = append(
					issues,
					fmt.Sprintf(
						"%s imports forbidden runtime or core widget primitive %q",
						kind,
						imported,
					),
				)
			}
		}
	}
	return issues
}

func validateSurfaceTemplateCommands(kind string, commands []SurfaceTemplateSmokeCommand) []string {
	required := []string{"generate", "check", "build", "run", "inspect", "visual", "package"}
	var issues []string
	seen := map[string]SurfaceTemplateSmokeCommand{}
	for _, command := range commands {
		if strings.TrimSpace(command.Kind) == "" {
			issues = append(issues, fmt.Sprintf("%s command kind is required", kind))
			continue
		}
		seen[command.Kind] = command
		if strings.TrimSpace(command.Command) == "" {
			issues = append(issues, fmt.Sprintf("%s %s command is required", kind, command.Kind))
		}
		if !command.Pass {
			issues = append(issues, fmt.Sprintf("%s %s command must pass", kind, command.Kind))
		}
		if command.ExitCode != 0 {
			issues = append(
				issues,
				fmt.Sprintf("%s %s exit_code = %d, want 0", kind, command.Kind, command.ExitCode),
			)
		}
	}
	for _, commandKind := range required {
		command, ok := seen[commandKind]
		if !ok {
			issues = append(issues, fmt.Sprintf("%s commands missing %s", kind, commandKind))
			continue
		}
		switch commandKind {
		case "generate":
			if !strings.Contains(command.Command, "tetra new surface-app") ||
				!strings.Contains(command.Command, "--template "+kind) {
				issues = append(
					issues,
					fmt.Sprintf(
						"%s generate command must run tetra new surface-app --template %s",
						kind,
						kind,
					),
				)
			}
		case "check":
			if !strings.Contains(command.Command, "tetra check") {
				issues = append(issues, fmt.Sprintf("%s check command must run tetra check", kind))
			}
		case "build":
			if !strings.Contains(command.Command, "tetra build") ||
				!strings.Contains(command.Command, "linux-x64") {
				issues = append(
					issues,
					fmt.Sprintf("%s build command must run tetra build --target linux-x64", kind),
				)
			}
		case "run":
			if !strings.Contains(command.Command, "tetra run") ||
				!strings.Contains(command.Command, "linux-x64") {
				issues = append(
					issues,
					fmt.Sprintf("%s run command must run tetra run --target linux-x64", kind),
				)
			}
		case "inspect":
			if !strings.Contains(command.Command, "surface-inspector") {
				issues = append(
					issues,
					fmt.Sprintf("%s inspect command must run surface-inspector", kind),
				)
			}
		case "visual":
			if !strings.Contains(command.Command, "surface-visual-diff") {
				issues = append(
					issues,
					fmt.Sprintf("%s visual command must run surface-visual-diff", kind),
				)
			}
		case "package":
			if !strings.Contains(command.Command, "tar") {
				issues = append(
					issues,
					fmt.Sprintf("%s package command must create tar package evidence", kind),
				)
			}
		}
	}
	return issues
}

func validateSurfaceTemplateSourceScan(kind string, scan SurfaceTemplateSmokeSourceScan) []string {
	var issues []string
	for _, check := range []struct {
		name string
		bad  bool
	}{
		{name: "react_import", bad: scan.ReactImport},
		{name: "electron_import", bad: scan.ElectronImport},
		{name: "dom_app_ui_tree", bad: scan.DOMAppUITree},
		{name: "css_runtime", bad: scan.CSSRuntime},
		{name: "core_widgets", bad: scan.CoreWidgets},
		{name: "platform_widgets", bad: scan.PlatformWidgets},
		{name: "user_js_app_logic", bad: scan.UserJSAppLogic},
	} {
		if check.bad {
			issues = append(
				issues,
				fmt.Sprintf("%s source_scan %s must be false", kind, check.name),
			)
		}
	}
	if !scan.Pass {
		issues = append(issues, fmt.Sprintf("%s source_scan pass must be true", kind))
	}
	return issues
}

func validateSurfaceTemplateSmokeInspector(evidence SurfaceTemplateSmokeInspector) []string {
	var issues []string
	if !safeRelativeReportPath(evidence.Path) {
		issues = append(issues, "inspector_evidence.path is unsafe or empty")
	}
	if evidence.Model != "surface-inspector-v1" {
		issues = append(
			issues,
			fmt.Sprintf(
				"inspector_evidence.model is %q, want surface-inspector-v1",
				evidence.Model,
			),
		)
	}
	if !evidence.Pass {
		issues = append(issues, "inspector_evidence pass must be true")
	}
	return issues
}

func validateSurfaceTemplateSmokeVisual(evidence SurfaceTemplateSmokeVisual) []string {
	var issues []string
	if !safeRelativeReportPath(evidence.Path) {
		issues = append(issues, "visual_evidence.path is unsafe or empty")
	}
	if evidence.Schema != VisualRegressionSchemaV1 {
		issues = append(
			issues,
			fmt.Sprintf(
				"visual_evidence.schema is %q, want %s",
				evidence.Schema,
				VisualRegressionSchemaV1,
			),
		)
	}
	if !evidence.Pass {
		issues = append(issues, "visual_evidence pass must be true")
	}
	return issues
}

func validateSurfaceTemplateSmokeMorphToPixels(
	chain *MorphToPixelsChainReport,
	templates []SurfaceTemplateSmokeTemplate,
) []string {
	if chain == nil {
		return []string{
			"morph_to_pixels is required for generated Surface template rendered beauty evidence",
		}
	}
	var issues []string
	issues = append(issues, validateMorphToPixelsChain("morph_to_pixels", *chain, "")...)
	if !safeRelativeReportPath(chain.ReportPath) {
		issues = append(issues, "morph_to_pixels.report_path is unsafe or empty")
	}
	if !safeRelativeReportPath(chain.FrameArtifact) {
		issues = append(issues, "morph_to_pixels.frame_artifact is unsafe or empty")
	}
	if !safeRelativeReportPath(chain.GoldenArtifact) {
		issues = append(issues, "morph_to_pixels.golden_artifact is unsafe or empty")
	}
	if chain.ProductClaim || chain.FinalSignoff {
		issues = append(
			issues,
			"morph_to_pixels template smoke evidence must not assert product_claim or final_signoff",
		)
	}
	if !templateSmokeContainsSource(templates, chain.Source) {
		issues = append(
			issues,
			fmt.Sprintf(
				"morph_to_pixels source %q must match one generated template source",
				chain.Source,
			),
		)
	}
	return issues
}

func validateSurfaceTemplateSmokePackages(packages []SurfaceTemplateSmokePackage) []string {
	if len(packages) == 0 {
		return []string{"package_evidence is required"}
	}
	var issues []string
	for _, pkg := range packages {
		if !safeRelativeReportPath(pkg.Path) {
			issues = append(issues, "package_evidence path is unsafe or empty")
		}
		if pkg.Kind != "tar.gz" {
			issues = append(
				issues,
				fmt.Sprintf("package_evidence kind is %q, want tar.gz", pkg.Kind),
			)
		}
		if !strings.HasPrefix(pkg.SHA256, "sha256:") || len(pkg.SHA256) != len("sha256:")+64 {
			issues = append(issues, "package_evidence sha256 must be sha256 digest")
		}
		if !pkg.Pass {
			issues = append(issues, "package_evidence pass must be true")
		}
	}
	return issues
}

func validateSurfaceTemplateSmokeNegativeGuards(
	guards SurfaceTemplateSmokeNegativeGuards,
) []string {
	if guards.NoReactImport &&
		guards.NoElectronImport &&
		guards.NoDOMAppUITree &&
		guards.NoCSSRuntime &&
		guards.NoCoreWidgets &&
		guards.NoPlatformWidgets &&
		guards.NoUserJSAppLogic &&
		guards.CookbookUsesBlockMorph {
		return nil
	}
	return []string{
		("negative_guards must reject React, Electron, DOM app UI tree, " +
			"CSS runtime, core widgets, platform widgets, user JS app logic, and " +
			"require Block/Morph cookbook recipes"),
	}
}

func templateSmokeContainsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func templateSmokeContainsSource(templates []SurfaceTemplateSmokeTemplate, want string) bool {
	want = normalizeEvidencePath(want)
	for _, tmpl := range templates {
		if normalizeEvidencePath(tmpl.Source) == want {
			return true
		}
	}
	return false
}

// ---- token_graph.go ----

const TokenGraphContractSchemaV1 = "tetra.surface.token-graph.contract.v1"

type TokenGraphValidationOptions struct {
	Root string
}

type TokenGraphContract struct {
	Schema                  string                         `json:"schema"`
	Status                  string                         `json:"status"`
	SurfaceScope            string                         `json:"surface_scope"`
	SourceOfTruth           TokenGraphSourceOfTruth        `json:"source_of_truth"`
	RequiredCategories      []string                       `json:"required_categories"`
	RequiredTokens          []string                       `json:"required_tokens"`
	ReferenceSources        []string                       `json:"reference_sources"`
	AllowedRawLiteralScopes []TokenGraphRawLiteralScope    `json:"allowed_raw_literal_scopes"`
	ForbiddenRuntimeModels  []string                       `json:"forbidden_runtime_models"`
	OverrideOrder           []string                       `json:"override_order"`
	DensityDPI              []MorphDensityDPIReport        `json:"density_dpi"`
	DiagnosticsRequired     []string                       `json:"diagnostics_required"`
	NegativeGuards          TokenGraphNegativeGuardsReport `json:"negative_guards"`
	NonClaims               []string                       `json:"nonclaims"`
}

type TokenGraphSourceOfTruth struct {
	Module               string `json:"module"`
	Namespace            string `json:"namespace"`
	Source               string `json:"source"`
	SingleTokenGraph     bool   `json:"single_token_graph"`
	ExplicitImports      bool   `json:"explicit_imports"`
	NoGlobalCascade      bool   `json:"no_global_cascade"`
	MultipleColorSources bool   `json:"multiple_color_sources"`
}

type TokenGraphRawLiteralScope struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type TokenGraphNegativeGuardsReport struct {
	AliasCycleRejected           bool `json:"alias_cycle_rejected"`
	MissingTokenRejected         bool `json:"missing_token_rejected"`
	DuplicateSourceRejected      bool `json:"duplicate_source_rejected"`
	RawLiteralRejected           bool `json:"raw_literal_rejected"`
	UnresolvedFallbackRejected   bool `json:"unresolved_fallback_rejected"`
	CSSRuntimeRejected           bool `json:"css_runtime_rejected"`
	MultipleColorSourcesRejected bool `json:"multiple_color_sources_rejected"`
	OverrideOrderRejected        bool `json:"override_order_rejected"`
	DensityDPIRejected           bool `json:"density_dpi_rejected"`
}

func ValidateTokenGraphContract(
	contractRaw []byte,
	reportRaw []byte,
	options TokenGraphValidationOptions,
) error {
	var contract TokenGraphContract
	if err := decodeStrict(contractRaw, &contract); err != nil {
		return err
	}
	var report Report
	if err := decodeStrict(reportRaw, &report); err != nil {
		return err
	}
	issues := validateTokenGraphContractFields(contract)
	issues = append(issues, validateTokenGraphReport(contract, report)...)
	issues = append(issues, validateTokenGraphReferenceSources(contract, options.Root)...)
	if len(issues) > 0 {
		sort.Strings(issues)
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateTokenGraphContractFields(contract TokenGraphContract) []string {
	var issues []string
	if contract.Schema != TokenGraphContractSchemaV1 {
		issues = append(
			issues,
			fmt.Sprintf(
				"token_graph contract schema is %q, want %s",
				contract.Schema,
				TokenGraphContractSchemaV1,
			),
		)
	}
	if contract.Status != "current" {
		issues = append(
			issues,
			fmt.Sprintf("token_graph contract status is %q, want current", contract.Status),
		)
	}
	if contract.SurfaceScope != "surface-token-graph-linux-web" {
		issues = append(
			issues,
			fmt.Sprintf(
				"token_graph contract surface_scope is %q, want surface-token-graph-linux-web",
				contract.SurfaceScope,
			),
		)
	}
	if contract.SourceOfTruth.Module != "lib.core.morph" {
		issues = append(
			issues,
			fmt.Sprintf(
				"token_graph source_of_truth.module is %q, want lib.core.morph",
				contract.SourceOfTruth.Module,
			),
		)
	}
	if contract.SourceOfTruth.Namespace != "tetra.surface.morph.app" {
		issues = append(
			issues,
			fmt.Sprintf(
				"token_graph source_of_truth.namespace is %q, want tetra.surface.morph.app",
				contract.SourceOfTruth.Namespace,
			),
		)
	}
	if contract.SourceOfTruth.Source != "capsule" {
		issues = append(
			issues,
			fmt.Sprintf(
				"token_graph source_of_truth.source is %q, want capsule",
				contract.SourceOfTruth.Source,
			),
		)
	}
	if !contract.SourceOfTruth.SingleTokenGraph {
		issues = append(issues, "token_graph source_of_truth requires single_token_graph")
	}
	if !contract.SourceOfTruth.ExplicitImports {
		issues = append(issues, "token_graph source_of_truth requires explicit_imports")
	}
	if !contract.SourceOfTruth.NoGlobalCascade {
		issues = append(issues, "token_graph source_of_truth requires no_global_cascade")
	}
	if contract.SourceOfTruth.MultipleColorSources {
		issues = append(issues, "token_graph source_of_truth rejects multiple_color_sources")
	}
	for _, category := range requiredTokenGraphCategories() {
		if !containsNormalized(contract.RequiredCategories, category) {
			issues = append(
				issues,
				fmt.Sprintf("token_graph contract required_categories missing %s", category),
			)
		}
	}
	for _, token := range requiredTokenGraphTokens() {
		if !containsNormalized(contract.RequiredTokens, token) {
			issues = append(
				issues,
				fmt.Sprintf("token_graph contract required_tokens missing %s", token),
			)
		}
	}
	if len(contract.ReferenceSources) == 0 {
		issues = append(issues, "token_graph contract reference_sources are required")
	}
	if len(contract.AllowedRawLiteralScopes) == 0 {
		issues = append(issues, "token_graph contract allowed_raw_literal_scopes are required")
	}
	for _, runtime := range []string{
		"CSS cascade runtime",
		"DOM style runtime",
		"React runtime",
		"Electron runtime",
		"platform-native widgets",
	} {
		if !containsTextFoldTokenGraph(contract.ForbiddenRuntimeModels, runtime) {
			issues = append(
				issues,
				fmt.Sprintf("token_graph contract forbidden_runtime_models missing %s", runtime),
			)
		}
	}
	if !sameStringSetFoldTokenGraph(contract.OverrideOrder, requiredTokenGraphOverrideOrder()) {
		issues = append(
			issues,
			"token_graph contract override_order must be [base theme density variant state local]",
		)
	}
	issues = append(
		issues,
		validateTokenGraphDensityMappings(
			contract.DensityDPI,
			contract.RequiredTokens,
			"contract",
		)...)
	issues = append(
		issues,
		validateTokenGraphDiagnostics(
			contract.DiagnosticsRequired,
			contract.NegativeGuards,
			"contract",
		)...)
	for _, nonclaim := range []string{
		"no CSS cascade runtime",
		"no React runtime",
		"no Electron runtime",
		"no DOM style runtime",
		"no platform-native widgets",
	} {
		if !containsTextFoldTokenGraph(contract.NonClaims, nonclaim) {
			issues = append(
				issues,
				fmt.Sprintf("token_graph contract nonclaims missing %q", nonclaim),
			)
		}
	}
	return issues
}

func validateTokenGraphReport(contract TokenGraphContract, report Report) []string {
	var issues []string
	if report.Morph == nil {
		return []string{"token_graph report requires morph evidence"}
	}
	morph := report.Morph
	if morph.Module != contract.SourceOfTruth.Module {
		issues = append(
			issues,
			fmt.Sprintf(
				"token_graph report morph.module is %q, want %s",
				morph.Module,
				contract.SourceOfTruth.Module,
			),
		)
	}
	if morph.Capsule.Namespace != contract.SourceOfTruth.Namespace {
		issues = append(
			issues,
			fmt.Sprintf(
				"token_graph report capsule namespace is %q, want %s",
				morph.Capsule.Namespace,
				contract.SourceOfTruth.Namespace,
			),
		)
	}
	if !morph.Capsule.ExplicitImports || !morph.Capsule.NoGlobalCascade {
		issues = append(
			issues,
			"token_graph report capsule must prove explicit imports and no global cascade",
		)
	}
	if morph.TokenGraph == nil {
		return append(issues, "token_graph report morph token_graph is required")
	}
	graph := morph.TokenGraph
	if graph.Schema != "tetra.surface.morph.token-graph.v1" {
		issues = append(
			issues,
			fmt.Sprintf(
				"token_graph report schema is %q, want tetra.surface.morph.token-graph.v1",
				graph.Schema,
			),
		)
	}
	if graph.Namespace != contract.SourceOfTruth.Namespace {
		issues = append(
			issues,
			fmt.Sprintf(
				"token_graph report namespace is %q, want %s",
				graph.Namespace,
				contract.SourceOfTruth.Namespace,
			),
		)
	}
	if graph.SourceOfTruth != contract.SourceOfTruth.Source {
		issues = append(
			issues,
			fmt.Sprintf(
				"token_graph report source_of_truth is %q, want %s",
				graph.SourceOfTruth,
				contract.SourceOfTruth.Source,
			),
		)
	}
	if !graph.ExplicitImports || !graph.NoGlobalCascade {
		issues = append(
			issues,
			"token_graph report requires explicit_imports and no_global_cascade",
		)
	}
	if !sameStringSetFoldTokenGraph(graph.FixedOverrideOrder, requiredTokenGraphOverrideOrder()) {
		issues = append(
			issues,
			"token_graph report fixed_override_order must be [base theme density variant state local]",
		)
	}
	if graph.Hash != morph.TokenGraphHash || !validSHA256Digest(graph.Hash) {
		issues = append(
			issues,
			"token_graph report hash must match morph token_graph_hash and be sha256 evidence",
		)
	}
	for _, category := range contract.RequiredCategories {
		if !containsNormalized(graph.Categories, category) {
			issues = append(
				issues,
				fmt.Sprintf("token_graph report categories missing %s", category),
			)
		}
	}
	tokenIDs := map[string]MorphTokenReport{}
	for _, token := range graph.Tokens {
		id := strings.TrimSpace(token.ID)
		if id == "" {
			issues = append(issues, "token_graph report token id is required")
			continue
		}
		if previous, ok := tokenIDs[id]; ok {
			issues = append(
				issues,
				fmt.Sprintf(
					"token_graph report duplicate token %s from %s and %s",
					id,
					previous.Source,
					token.Source,
				),
			)
		}
		tokenIDs[id] = token
		if token.Source != contract.SourceOfTruth.Source {
			issues = append(
				issues,
				fmt.Sprintf(
					"token_graph report token %s source is %q, want %s",
					id,
					token.Source,
					contract.SourceOfTruth.Source,
				),
			)
		}
		if !containsNormalized(graph.Categories, token.Category) {
			issues = append(
				issues,
				fmt.Sprintf(
					"token_graph report token %s category %q is not declared",
					id,
					token.Category,
				),
			)
		}
		if !validSHA256Digest(token.Hash) {
			issues = append(
				issues,
				fmt.Sprintf("token_graph report token %s hash must be sha256 evidence", id),
			)
		}
	}
	for _, token := range contract.RequiredTokens {
		if _, ok := tokenIDs[token]; !ok {
			issues = append(
				issues,
				fmt.Sprintf("token_graph report required token missing %s", token),
			)
		}
	}
	issues = append(issues, validateTokenGraphMaterialRefs(morph.Materials, tokenIDs)...)
	issues = append(issues, validateTokenGraphAssetRefs(morph.AssetRefs, tokenIDs)...)
	if graph.RawLiteralsInAppCode {
		issues = append(issues, "token_graph report rejects raw literals in app code")
	}
	if graph.FallbackToRandomDefault {
		issues = append(issues, "token_graph report rejects fallback-to-random-default")
	}
	if !graph.AliasCycleRejected || !graph.DuplicateSourceRejected ||
		!graph.UnresolvedFallbackRejected {
		issues = append(
			issues,
			"token_graph report requires alias_cycle, duplicate_source, and unresolved_fallback rejection",
		)
	}
	issues = append(
		issues,
		validateTokenGraphDensityMappings(graph.DensityDPI, mapKeys(tokenIDs), "report")...)
	issues = append(issues, validateMorphTokenGraphDiagnostics(graph.Diagnostics)...)
	return issues
}

func validateTokenGraphMaterialRefs(
	materials []MorphMaterialReport,
	tokens map[string]MorphTokenReport,
) []string {
	var issues []string
	if len(materials) == 0 {
		return []string{"token_graph report materials are required"}
	}
	for _, material := range materials {
		for field, token := range map[string]string{
			"fill":    material.Fill,
			"border":  material.Border,
			"radius":  material.Radius,
			"shadow":  material.Shadow,
			"overlay": material.Overlay,
		} {
			token = strings.TrimSpace(token)
			if token == "" {
				continue
			}
			if _, ok := tokens[token]; !ok {
				issues = append(
					issues,
					fmt.Sprintf(
						"token_graph report material %s missing token %s for %s",
						material.Name,
						token,
						field,
					),
				)
			}
		}
	}
	return issues
}

func validateTokenGraphAssetRefs(
	refs []MorphAssetRefReport,
	tokens map[string]MorphTokenReport,
) []string {
	var issues []string
	for _, ref := range refs {
		if strings.TrimSpace(ref.TintToken) != "" {
			if _, ok := tokens[ref.TintToken]; !ok {
				issues = append(
					issues,
					fmt.Sprintf(
						"token_graph report asset_ref %s missing tint token %s",
						ref.ID,
						ref.TintToken,
					),
				)
			}
		}
		if strings.TrimSpace(ref.FallbackID) != "" {
			fallbackToken := "assets." + strings.TrimSpace(ref.FallbackID)
			if _, ok := tokens[fallbackToken]; !ok {
				issues = append(
					issues,
					fmt.Sprintf(
						"token_graph report asset_ref %s missing fallback token %s",
						ref.ID,
						fallbackToken,
					),
				)
			}
		}
	}
	return issues
}

func validateTokenGraphReferenceSources(contract TokenGraphContract, root string) []string {
	if strings.TrimSpace(root) == "" {
		return nil
	}
	var issues []string
	for _, source := range contract.ReferenceSources {
		clean := normalizeEvidencePath(source)
		path := filepath.Join(root, filepath.FromSlash(clean))
		raw, err := os.ReadFile(path)
		if err != nil {
			issues = append(
				issues,
				fmt.Sprintf("token_graph reference source %s cannot be read: %v", clean, err),
			)
			continue
		}
		if tokenGraphPathAllowsRawLiterals(clean, contract.AllowedRawLiteralScopes) {
			continue
		}
		if sourceHasRawStyleLiteral(string(raw)) {
			issues = append(
				issues,
				fmt.Sprintf(
					"token_graph reference source %s contains raw literal outside allowed scopes",
					clean,
				),
			)
		}
	}
	return issues
}

func validateTokenGraphDensityMappings(
	mappings []MorphDensityDPIReport,
	tokenIDs []string,
	label string,
) []string {
	var issues []string
	if len(mappings) == 0 {
		return []string{fmt.Sprintf("token_graph %s density_dpi mappings are required", label)}
	}
	for _, target := range []string{"headless", "linux-x64-real-window", "wasm32-web-browser-canvas"} {
		found := false
		for _, mapping := range mappings {
			if mapping.Target != target {
				continue
			}
			found = true
			if !containsNormalized(tokenIDs, mapping.Token) {
				issues = append(
					issues,
					fmt.Sprintf(
						"token_graph %s density_dpi target %s token %s is not declared",
						label,
						target,
						mapping.Token,
					),
				)
			}
			if mapping.TargetDPI < 96 {
				issues = append(
					issues,
					fmt.Sprintf(
						"token_graph %s density_dpi target %s target_dpi is %d, want >= 96",
						label,
						target,
						mapping.TargetDPI,
					),
				)
			}
			if mapping.ScaleMilli < 1000 || mapping.ScaleMilli > 4000 {
				issues = append(
					issues,
					fmt.Sprintf(
						"token_graph %s density_dpi target %s scale_milli is %d, want 1000..4000",
						label,
						target,
						mapping.ScaleMilli,
					),
				)
			}
			if normalizeTokenGraphName(mapping.RoundingPolicy) != "integer_half_up_v1" {
				issues = append(
					issues,
					fmt.Sprintf(
						"token_graph %s density_dpi target %s rounding_policy is %q, want integer-half-up-v1",
						label,
						target,
						mapping.RoundingPolicy,
					),
				)
			}
		}
		if !found {
			issues = append(
				issues,
				fmt.Sprintf("token_graph %s density_dpi missing target %s", label, target),
			)
		}
	}
	return issues
}

func validateTokenGraphDiagnostics(
	required []string,
	guards TokenGraphNegativeGuardsReport,
	label string,
) []string {
	var issues []string
	requiredNames := []string{
		"alias_cycle",
		"missing_token",
		"duplicate_source",
		"raw_literal",
		"unresolved_fallback",
		"css_runtime",
		"multiple_color_sources",
		"override_order",
		"density_dpi",
	}
	for _, name := range requiredNames {
		if !containsNormalized(required, name) {
			issues = append(
				issues,
				fmt.Sprintf("token_graph %s diagnostics_required missing %s", label, name),
			)
		}
	}
	checks := []struct {
		name string
		ok   bool
	}{
		{"alias_cycle_rejected", guards.AliasCycleRejected},
		{"missing_token_rejected", guards.MissingTokenRejected},
		{"duplicate_source_rejected", guards.DuplicateSourceRejected},
		{"raw_literal_rejected", guards.RawLiteralRejected},
		{"unresolved_fallback_rejected", guards.UnresolvedFallbackRejected},
		{"css_runtime_rejected", guards.CSSRuntimeRejected},
		{"multiple_color_sources_rejected", guards.MultipleColorSourcesRejected},
		{"override_order_rejected", guards.OverrideOrderRejected},
		{"density_dpi_rejected", guards.DensityDPIRejected},
	}
	for _, check := range checks {
		if !check.ok {
			issues = append(
				issues,
				fmt.Sprintf("token_graph %s negative_guards missing %s", label, check.name),
			)
		}
	}
	return issues
}

func validateMorphTokenGraphDiagnostics(diagnostics MorphTokenGraphDiagnosticsReport) []string {
	guards := TokenGraphNegativeGuardsReport{
		AliasCycleRejected:           diagnostics.AliasCycleRejected,
		MissingTokenRejected:         diagnostics.MissingTokenRejected,
		DuplicateSourceRejected:      diagnostics.DuplicateSourceRejected,
		RawLiteralRejected:           diagnostics.RawLiteralRejected,
		UnresolvedFallbackRejected:   diagnostics.UnresolvedFallbackRejected,
		CSSRuntimeRejected:           diagnostics.CSSRuntimeRejected,
		MultipleColorSourcesRejected: diagnostics.MultipleColorSourcesRejected,
		OverrideOrderRejected:        diagnostics.OverrideOrderRejected,
		DensityDPIRejected:           diagnostics.DensityDPIRejected,
	}
	return validateTokenGraphDiagnostics(
		[]string{
			"alias_cycle",
			"missing_token",
			"duplicate_source",
			"raw_literal",
			"unresolved_fallback",
			"css_runtime",
			"multiple_color_sources",
			"override_order",
			"density_dpi",
		},
		guards,
		"report",
	)
}

func tokenGraphPathAllowsRawLiterals(path string, scopes []TokenGraphRawLiteralScope) bool {
	path = normalizeEvidencePath(path)
	for _, scope := range scopes {
		pattern := normalizeEvidencePath(scope.Path)
		if pattern == "" {
			continue
		}
		if ok, _ := filepath.Match(pattern, path); ok {
			return true
		}
		if pattern == path {
			return true
		}
	}
	return false
}

func sourceHasRawStyleLiteral(source string) bool {
	rawNeedles := []string{"surface.Color(", "draw.Color(", "Color(r:", "#", "rgba(", "rgb("}
	for _, needle := range rawNeedles {
		if strings.Contains(source, needle) {
			return true
		}
	}
	return false
}

func requiredTokenGraphCategories() []string {
	return []string{
		"color",
		"space",
		"radius",
		"border",
		"elevation",
		"opacity",
		"typography",
		"motion",
		"z",
		"assets",
		"density",
	}
}

func requiredTokenGraphTokens() []string {
	return []string{
		"color.bg",
		"color.surface",
		"color.surfaceAlpha",
		"color.accent",
		"color.muted",
		"color.warning",
		"space.3",
		"radius.sm",
		"radius.md",
		"radius.lg",
		"border.subtle",
		"border.glass",
		"elevation.2",
		"elevation.3",
		"opacity.disabled",
		"type.label",
		"motion.fast",
		"motion.soft",
		"z.base",
		"assets.gradient.vertical",
		"assets.icon.fallback",
		"density.1x",
	}
}

func requiredTokenGraphOverrideOrder() []string {
	return []string{"base", "theme", "density", "variant", "state", "local"}
}

func normalizeTokenGraphName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, ".", "_")
	return value
}

func containsTextFoldTokenGraph(values []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == want {
			return true
		}
	}
	return false
}

func sameStringSetFoldTokenGraph(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if strings.ToLower(
			strings.TrimSpace(got[i]),
		) != strings.ToLower(
			strings.TrimSpace(want[i]),
		) {
			return false
		}
	}
	return true
}

func mapKeys(tokens map[string]MorphTokenReport) []string {
	keys := make([]string, 0, len(tokens))
	for key := range tokens {
		keys = append(keys, key)
	}
	return keys
}

// ---- visual.go ----

const VisualRegressionSchemaV1 = "tetra.surface.visual-regression.v1"

type VisualRegressionReport struct {
	Schema          string                               `json:"schema"`
	Status          string                               `json:"status"`
	GitHead         string                               `json:"git_head"`
	GoldenSet       string                               `json:"golden_set"`
	GoldenHash      string                               `json:"golden_hash"`
	RequiredTargets []string                             `json:"required_targets"`
	RequiredSources []string                             `json:"required_sources"`
	Apps            []VisualRegressionAppReport          `json:"apps"`
	NegativeGuards  VisualRegressionNegativeGuardsReport `json:"negative_guards"`
}

type VisualRegressionAppReport struct {
	Name         string                         `json:"name"`
	Source       string                         `json:"source"`
	ReferenceApp bool                           `json:"reference_app"`
	Targets      []VisualRegressionTargetReport `json:"targets"`
}

type VisualRegressionTargetReport struct {
	Target                string                        `json:"target"`
	RuntimeReport         string                        `json:"runtime_report"`
	RuntimeSchema         string                        `json:"runtime_schema"`
	GitHead               string                        `json:"git_head"`
	GoldenGitHead         string                        `json:"golden_git_head"`
	Renderer              string                        `json:"renderer"`
	ScreenshotOnly        bool                          `json:"screenshot_only,omitempty"`
	PNGArtifactSHA256     string                        `json:"png_artifact_sha256,omitempty"`
	BlockGraphEvidence    bool                          `json:"block_graph_evidence"`
	TokenThemeEvidence    bool                          `json:"token_theme_evidence"`
	LayoutEvidence        bool                          `json:"layout_evidence"`
	AccessibilityEvidence bool                          `json:"accessibility_evidence"`
	PerformanceEvidence   bool                          `json:"performance_evidence"`
	Frames                []VisualRegressionFrameReport `json:"frames"`
}

type VisualRegressionFrameReport struct {
	Order                 int    `json:"order"`
	Label                 string `json:"label"`
	Width                 int    `json:"width"`
	Height                int    `json:"height"`
	Stride                int    `json:"stride"`
	Checksum              string `json:"checksum"`
	GoldenChecksum        string `json:"golden_checksum"`
	ArtifactPath          string `json:"artifact_path"`
	ArtifactSHA256        string `json:"artifact_sha256"`
	ArtifactFormat        string `json:"artifact_format"`
	GoldenArtifactPath    string `json:"golden_artifact_path"`
	GoldenArtifactSHA256  string `json:"golden_artifact_sha256"`
	DiffPixels            int    `json:"diff_pixels"`
	DiffRatioMilli        int    `json:"diff_ratio_milli"`
	MaxChannelDelta       int    `json:"max_channel_delta"`
	TolerancePixels       int    `json:"tolerance_pixels"`
	ToleranceRatioMilli   int    `json:"tolerance_ratio_milli"`
	ToleranceChannelDelta int    `json:"tolerance_channel_delta"`
	Pass                  bool   `json:"pass"`
}

type VisualRegressionNegativeGuardsReport struct {
	ScreenshotOnlyRejected           bool `json:"screenshot_only_rejected"`
	StaleGoldenRejected              bool `json:"stale_golden_rejected"`
	MajorDriftRejected               bool `json:"major_drift_rejected"`
	MissingBlockGraphRejected        bool `json:"missing_block_graph_rejected"`
	MissingLayoutRejected            bool `json:"missing_layout_rejected"`
	MissingAccessibilityRejected     bool `json:"missing_accessibility_rejected"`
	MissingPerformanceRejected       bool `json:"missing_performance_rejected"`
	SelfGoldenRejected               bool `json:"self_golden_rejected"`
	MetadataChecksumRejected         bool `json:"metadata_checksum_rejected"`
	FixtureFrameOnlyRejected         bool `json:"fixture_frame_only_rejected"`
	MissingPNGOrRGBAArtifactRejected bool `json:"missing_png_or_rgba_artifact_rejected"`
}

func ValidateVisualReport(raw []byte) error {
	var report VisualRegressionReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return fmt.Errorf("decode Surface visual report: %w", err)
	}
	issues := validateVisualRegressionReport(report)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateVisualRegressionReport(report VisualRegressionReport) []string {
	var issues []string
	if report.Schema != VisualRegressionSchemaV1 {
		issues = append(
			issues,
			fmt.Sprintf("schema is %q, want %s", report.Schema, VisualRegressionSchemaV1),
		)
	}
	if strings.TrimSpace(report.Status) != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if strings.TrimSpace(report.GitHead) == "" {
		issues = append(issues, "git_head is required")
	}
	if strings.TrimSpace(report.GoldenSet) == "" {
		issues = append(issues, "golden_set is required")
	}
	if !validChecksumLike(report.GoldenHash) {
		issues = append(issues, "golden_hash must be sha256 evidence")
	}
	if len(report.RequiredTargets) == 0 {
		issues = append(issues, "required_targets evidence is required")
	}
	if len(report.RequiredSources) == 0 {
		issues = append(issues, "required_sources evidence is required")
	}
	issues = append(issues, validateVisualNegativeGuards(report.NegativeGuards)...)
	if len(report.Apps) == 0 {
		issues = append(issues, "apps visual evidence is required")
	}
	sources := map[string]bool{}
	for i, app := range report.Apps {
		sources[normalizeEvidencePath(app.Source)] = true
		issues = append(
			issues,
			validateVisualApp(i, report.GitHead, report.RequiredTargets, app)...)
	}
	for _, required := range report.RequiredSources {
		if !sources[normalizeEvidencePath(required)] {
			issues = append(issues, fmt.Sprintf("missing required source %s", required))
		}
	}
	return issues
}

func validateVisualNegativeGuards(guards VisualRegressionNegativeGuardsReport) []string {
	var missing []string
	if !guards.ScreenshotOnlyRejected {
		missing = append(missing, "screenshot_only_rejected")
	}
	if !guards.StaleGoldenRejected {
		missing = append(missing, "stale_golden_rejected")
	}
	if !guards.MajorDriftRejected {
		missing = append(missing, "major_drift_rejected")
	}
	if !guards.MissingBlockGraphRejected {
		missing = append(missing, "missing_block_graph_rejected")
	}
	if !guards.MissingLayoutRejected {
		missing = append(missing, "missing_layout_rejected")
	}
	if !guards.MissingAccessibilityRejected {
		missing = append(missing, "missing_accessibility_rejected")
	}
	if !guards.MissingPerformanceRejected {
		missing = append(missing, "missing_performance_rejected")
	}
	if !guards.SelfGoldenRejected {
		missing = append(missing, "self_golden_rejected")
	}
	if !guards.MetadataChecksumRejected {
		missing = append(missing, "metadata_checksum_rejected")
	}
	if !guards.FixtureFrameOnlyRejected {
		missing = append(missing, "fixture_frame_only_rejected")
	}
	if !guards.MissingPNGOrRGBAArtifactRejected {
		missing = append(missing, "missing_png_or_rgba_artifact_rejected")
	}
	if len(missing) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("negative_guards missing %s", strings.Join(missing, ", "))}
}

func validateVisualApp(
	index int,
	reportGitHead string,
	requiredTargets []string,
	app VisualRegressionAppReport,
) []string {
	var issues []string
	prefix := fmt.Sprintf("apps[%d]", index)
	if strings.TrimSpace(app.Name) == "" {
		issues = append(issues, prefix+" name is required")
	}
	if strings.TrimSpace(app.Source) == "" {
		issues = append(issues, prefix+" source is required")
	}
	if !app.ReferenceApp {
		issues = append(issues, prefix+" reference_app must be true")
	}
	targets := map[string]bool{}
	for i, target := range app.Targets {
		name := normalizeVisualTarget(target.Target)
		targets[name] = true
		issues = append(
			issues,
			validateVisualTarget(
				fmt.Sprintf("%s.targets[%d]", prefix, i),
				reportGitHead,
				target,
			)...)
	}
	for _, required := range requiredTargets {
		requiredName := normalizeVisualTarget(required)
		if !targets[requiredName] {
			issues = append(issues, fmt.Sprintf("%s missing required target %s", prefix, required))
		}
	}
	return issues
}

func validateVisualTarget(
	prefix string,
	reportGitHead string,
	target VisualRegressionTargetReport,
) []string {
	var issues []string
	if strings.TrimSpace(target.Target) == "" {
		issues = append(issues, prefix+" target is required")
	}
	if strings.TrimSpace(target.RuntimeReport) == "" {
		issues = append(issues, prefix+" runtime_report is required")
	}
	if target.RuntimeSchema != SchemaV1 {
		issues = append(
			issues,
			fmt.Sprintf("%s runtime_schema is %q, want %s", prefix, target.RuntimeSchema, SchemaV1),
		)
	}
	if strings.TrimSpace(target.GitHead) == "" {
		issues = append(issues, prefix+" git_head is required")
	}
	if strings.TrimSpace(target.GoldenGitHead) == "" {
		issues = append(issues, prefix+" golden_git_head is required")
	}
	if strings.TrimSpace(target.GitHead) != "" && strings.TrimSpace(target.GoldenGitHead) != "" &&
		target.GitHead != target.GoldenGitHead {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s stale golden git_head %q, want %q",
				prefix,
				target.GoldenGitHead,
				target.GitHead,
			),
		)
	}
	if strings.TrimSpace(reportGitHead) != "" && strings.TrimSpace(target.GitHead) != "" &&
		reportGitHead != target.GitHead {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s git_head %q does not match report git_head %q",
				prefix,
				target.GitHead,
				reportGitHead,
			),
		)
	}
	if strings.TrimSpace(target.Renderer) == "" {
		issues = append(issues, prefix+" renderer is required")
	}
	if target.ScreenshotOnly {
		issues = append(issues, prefix+" screenshot-only evidence is not sufficient")
	}
	if strings.TrimSpace(target.PNGArtifactSHA256) != "" &&
		!validChecksumLike(target.PNGArtifactSHA256) {
		issues = append(issues, prefix+" png_artifact_sha256 must be sha256 evidence")
	}
	if !target.BlockGraphEvidence {
		issues = append(issues, prefix+" block graph evidence is required")
	}
	if !target.TokenThemeEvidence {
		issues = append(issues, prefix+" token/theme conformance evidence is required")
	}
	if !target.LayoutEvidence {
		issues = append(issues, prefix+" layout evidence is required")
	}
	if !target.AccessibilityEvidence {
		issues = append(issues, prefix+" accessibility evidence is required")
	}
	if !target.PerformanceEvidence {
		issues = append(issues, prefix+" performance evidence is required")
	}
	if len(target.Frames) == 0 {
		issues = append(issues, prefix+" frame diff evidence is required")
	}
	lastOrder := 0
	for i, frame := range target.Frames {
		if frame.Order <= lastOrder {
			issues = append(
				issues,
				fmt.Sprintf(
					"%s.frames[%d] order %d is not strictly greater than previous order %d",
					prefix,
					i,
					frame.Order,
					lastOrder,
				),
			)
		}
		lastOrder = frame.Order
		issues = append(
			issues,
			validateVisualFrame(fmt.Sprintf("%s.frames[%d]", prefix, i), frame)...)
	}
	return issues
}

func validateVisualFrame(prefix string, frame VisualRegressionFrameReport) []string {
	var issues []string
	if strings.TrimSpace(frame.Label) == "" {
		issues = append(issues, prefix+" label is required")
	}
	if frame.Width <= 0 || frame.Height <= 0 || frame.Stride <= 0 {
		issues = append(issues, prefix+" dimensions and stride must be positive")
	}
	if !validChecksumLike(frame.Checksum) {
		issues = append(issues, prefix+" checksum must be sha256 evidence")
	}
	if !validChecksumLike(frame.GoldenChecksum) {
		issues = append(issues, prefix+" golden_checksum must be sha256 evidence")
	}
	artifactPath := normalizeEvidencePath(frame.ArtifactPath)
	goldenArtifactPath := normalizeEvidencePath(frame.GoldenArtifactPath)
	if artifactPath == "" {
		issues = append(issues, prefix+" artifact_path is required")
	}
	if goldenArtifactPath == "" {
		issues = append(issues, prefix+" golden_artifact_path is required")
	}
	if artifactPath != "" && goldenArtifactPath != "" && artifactPath == goldenArtifactPath {
		issues = append(issues, prefix+" self-golden artifact rejected")
	}
	if visualArtifactLooksLikeFixture(artifactPath) ||
		visualArtifactLooksLikeFixture(goldenArtifactPath) {
		issues = append(issues, prefix+" fixture frame artifact is not product visual evidence")
	}
	format := strings.ToLower(strings.TrimSpace(frame.ArtifactFormat))
	if format != "rgba" && format != "png" {
		issues = append(issues, prefix+" artifact_format must be png or rgba")
	}
	if artifactPath != "" && !visualArtifactPathHasSupportedFormat(artifactPath) {
		issues = append(issues, prefix+" artifact_path must point to a png or rgba artifact")
	}
	if goldenArtifactPath != "" && !visualArtifactPathHasSupportedFormat(goldenArtifactPath) {
		issues = append(issues, prefix+" golden_artifact_path must point to a png or rgba artifact")
	}
	if !validChecksumLike(frame.ArtifactSHA256) {
		issues = append(issues, prefix+" artifact_sha256 must be sha256 evidence")
	}
	if !validChecksumLike(frame.GoldenArtifactSHA256) {
		issues = append(issues, prefix+" golden_artifact_sha256 must be sha256 evidence")
	}
	if validChecksumLike(frame.Checksum) && validChecksumLike(frame.ArtifactSHA256) &&
		frame.Checksum != frame.ArtifactSHA256 {
		issues = append(issues, prefix+" artifact_sha256 must match checksum")
	}
	if validChecksumLike(frame.GoldenChecksum) && validChecksumLike(frame.GoldenArtifactSHA256) &&
		frame.GoldenChecksum != frame.GoldenArtifactSHA256 {
		issues = append(issues, prefix+" golden_artifact_sha256 must match golden_checksum")
	}
	if frame.DiffPixels < 0 || frame.DiffRatioMilli < 0 || frame.MaxChannelDelta < 0 {
		issues = append(issues, prefix+" visual diff metrics must be non-negative")
	}
	if frame.TolerancePixels < 0 || frame.ToleranceRatioMilli < 0 ||
		frame.ToleranceChannelDelta < 0 {
		issues = append(issues, prefix+" visual diff tolerances must be non-negative")
	}
	if frame.DiffPixels > frame.TolerancePixels ||
		frame.DiffRatioMilli > frame.ToleranceRatioMilli ||
		frame.MaxChannelDelta > frame.ToleranceChannelDelta ||
		!frame.Pass {
		issues = append(issues, fmt.Sprintf("%s visual drift exceeds tolerance", prefix))
	}
	return issues
}

func normalizeVisualTarget(value string) string {
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(value, "_", "-")))
}

func visualArtifactPathHasSupportedFormat(path string) bool {
	lower := strings.ToLower(strings.TrimSpace(path))
	return strings.HasSuffix(lower, ".rgba") || strings.HasSuffix(lower, ".png")
}

func visualArtifactLooksLikeFixture(path string) bool {
	lower := normalizeEvidencePath(strings.ToLower(path))
	return strings.Contains(lower, "/testdata/") ||
		strings.Contains(lower, "/fixtures/") ||
		strings.Contains(lower, "fixture-frame")
}

// ---- widget_migration.go ----

const WidgetMigrationSchemaV1 = "tetra.surface.widget-migration.v1"

type SurfaceWidgetMigrationReportV1 struct {
	Schema                string                                 `json:"schema"`
	Model                 string                                 `json:"model"`
	ReleaseScope          string                                 `json:"release_scope"`
	Producer              string                                 `json:"producer"`
	Source                string                                 `json:"source"`
	ReferenceApp          string                                 `json:"reference_app"`
	Target                string                                 `json:"target"`
	CompatibilityLayer    SurfaceWidgetMigrationCompatibility    `json:"compatibility_layer"`
	ReleaseWidgetSet      SurfaceWidgetMigrationReleaseSet       `json:"release_widget_set"`
	EquivalenceRows       []SurfaceWidgetMigrationEquivalence    `json:"equivalence_rows"`
	MorphRecipeMigration  SurfaceWidgetMigrationMorphRecipes     `json:"morph_recipe_migration"`
	MigrationReferenceApp SurfaceWidgetMigrationReferenceApp     `json:"migration_reference_app"`
	NegativeGuards        SurfaceWidgetMigrationNegativeGuards   `json:"negative_guards"`
	ArtifactEvidence      SurfaceWidgetMigrationArtifactEvidence `json:"artifact_evidence,omitempty"`
	Pass                  bool                                   `json:"pass"`
}

type SurfaceWidgetMigrationCompatibility struct {
	Module                      string `json:"module"`
	SupportedSurfaceV1          bool   `json:"supported_surface_v1"`
	CurrentAPIPreserved         bool   `json:"current_api_preserved"`
	APIBreakingChange           bool   `json:"api_breaking_change"`
	MigrationEquivalenceHelpers bool   `json:"migration_equivalence_helpers"`
	MigrationDocs               bool   `json:"migration_docs"`
	Pass                        bool   `json:"pass"`
}

type SurfaceWidgetMigrationReleaseSet struct {
	Widgets                 []string `json:"widgets"`
	Intact                  bool     `json:"intact"`
	NonMigrationWidgetUsage bool     `json:"non_migration_widget_usage"`
	Pass                    bool     `json:"pass"`
}

type SurfaceWidgetMigrationEquivalence struct {
	LegacyWidget    string `json:"legacy_widget"`
	LegacyFunction  string `json:"legacy_function"`
	MorphRecipe     string `json:"morph_recipe"`
	BlockExpander   string `json:"block_expander"`
	BlockKind       string `json:"block_kind"`
	LegacyResult    int    `json:"legacy_result"`
	BlockResult     int    `json:"block_result"`
	APIUnchanged    bool   `json:"api_unchanged"`
	ResolvesToBlock bool   `json:"resolves_to_block"`
	Pass            bool   `json:"pass"`
}

type SurfaceWidgetMigrationMorphRecipes struct {
	Recipes                []string `json:"recipes"`
	CorePrimitives         []string `json:"core_primitives"`
	BlockOnlyCorePrimitive bool     `json:"block_only_core_primitive"`
	WidgetsPromotedToCore  bool     `json:"widgets_promoted_to_core"`
	ResolvesToBlock        bool     `json:"resolves_to_block"`
	Pass                   bool     `json:"pass"`
}

type SurfaceWidgetMigrationReferenceApp struct {
	Shape             string   `json:"shape"`
	Source            string   `json:"source"`
	Imports           []string `json:"imports"`
	Compiles          bool     `json:"compiles"`
	Runs              bool     `json:"runs"`
	ExitCode          int      `json:"exit_code"`
	UsesWidgetsCompat bool     `json:"uses_widgets_compat"`
	UsesMorphRecipes  bool     `json:"uses_morph_recipes"`
	ResolvesToBlock   bool     `json:"resolves_to_block"`
	Pass              bool     `json:"pass"`
}

type SurfaceWidgetMigrationNegativeGuards struct {
	NoFutureCorePrimitivePromotion bool `json:"no_future_core_primitive_promotion"`
	NoWidgetPrimaryFutureCore      bool `json:"no_widget_primary_future_core"`
	NoBreakingChange               bool `json:"no_breaking_change"`
	NoDocsOnly                     bool `json:"no_docs_only"`
	NoPlatformNativeRuntimeClaims  bool `json:"no_platform_native_runtime_claims"`
}

type SurfaceWidgetMigrationArtifactEvidence struct {
	EquivalenceRowsSHA256 string `json:"equivalence_rows_sha256,omitempty"`
	SourceScanSHA256      string `json:"source_scan_sha256,omitempty"`
}

func ValidateWidgetMigrationReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != WidgetMigrationSchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, WidgetMigrationSchemaV1)
	}
	var report SurfaceWidgetMigrationReportV1
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}
	issues := validateSurfaceWidgetMigrationReport(report)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateSurfaceWidgetMigrationReport(report SurfaceWidgetMigrationReportV1) []string {
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: report.Schema, want: WidgetMigrationSchemaV1},
		{field: "model", got: report.Model, want: "surface-widget-migration-v1"},
		{field: "release_scope", got: report.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
		{
			field: "producer",
			got:   report.Producer,
			want:  "scripts/release/surface/surface-widget-migration-smoke.sh",
		},
		{
			field: "source",
			got:   report.Source,
			want:  "examples/surface/reference_forms/surface_reference_migration.tetra",
		},
		{field: "reference_app", got: report.ReferenceApp, want: "migration"},
		{field: "target", got: report.Target, want: "linux-x64"},
	} {
		if check.got != check.want {
			issues = append(
				issues,
				fmt.Sprintf("%s is %q, want %q", check.field, check.got, check.want),
			)
		}
	}
	if !safeRelativeSourcePath(report.Source) {
		issues = append(issues, "source must be a safe Tetra source path")
	}
	if !surfacePackageSourceMatchesReferenceApp(report.ReferenceApp, report.Source) {
		issues = append(
			issues,
			fmt.Sprintf(
				"reference_app %q does not match source %q",
				report.ReferenceApp,
				report.Source,
			),
		)
	}
	issues = append(
		issues,
		validateSurfaceWidgetMigrationCompatibility(report.CompatibilityLayer)...)
	issues = append(issues, validateSurfaceWidgetMigrationReleaseSet(report.ReleaseWidgetSet)...)
	issues = append(issues, validateSurfaceWidgetMigrationEquivalence(report.EquivalenceRows)...)
	issues = append(
		issues,
		validateSurfaceWidgetMigrationMorphRecipes(report.MorphRecipeMigration)...)
	issues = append(
		issues,
		validateSurfaceWidgetMigrationReferenceApp(report.MigrationReferenceApp)...)
	issues = append(issues, validateSurfaceWidgetMigrationNegativeGuards(report.NegativeGuards)...)
	issues = append(
		issues,
		validateSurfaceWidgetMigrationArtifactEvidence(report.ArtifactEvidence)...)
	if !report.Pass {
		issues = append(issues, "pass must be true")
	}
	return issues
}

func validateSurfaceWidgetMigrationCompatibility(
	layer SurfaceWidgetMigrationCompatibility,
) []string {
	var issues []string
	if layer.Module != "lib.core.widgets" {
		issues = append(
			issues,
			fmt.Sprintf("compatibility_layer module is %q, want lib.core.widgets", layer.Module),
		)
	}
	if !layer.SupportedSurfaceV1 {
		issues = append(issues, "lib.core.widgets must remain supported for Surface v1")
	}
	if !layer.CurrentAPIPreserved {
		issues = append(issues, "lib.core.widgets current API must be preserved")
	}
	if layer.APIBreakingChange {
		issues = append(issues, "lib.core.widgets API breaking change must be false")
	}
	if !layer.MigrationEquivalenceHelpers {
		issues = append(issues, "compatibility_layer must record migration equivalence helpers")
	}
	if !layer.MigrationDocs {
		issues = append(issues, "compatibility_layer requires migration docs evidence")
	}
	if !layer.Pass {
		issues = append(issues, "compatibility_layer pass must be true")
	}
	return issues
}

func validateSurfaceWidgetMigrationReleaseSet(set SurfaceWidgetMigrationReleaseSet) []string {
	var issues []string
	issues = append(
		issues,
		validateExactStringList(
			"release_widget_set.widgets",
			set.Widgets,
			surfaceWidgetMigrationReleaseWidgets(),
		)...)
	if !set.Intact {
		issues = append(issues, "release_widget_set intact must be true")
	}
	if set.NonMigrationWidgetUsage {
		issues = append(issues, "non_migration_widget_usage must be false")
	}
	if !set.Pass {
		issues = append(issues, "release_widget_set pass must be true")
	}
	return issues
}

func validateSurfaceWidgetMigrationEquivalence(rows []SurfaceWidgetMigrationEquivalence) []string {
	if len(rows) < 3 {
		return []string{"equivalence_rows require Panel, Button, and TextBox"}
	}
	required := map[string]struct {
		legacyFunction string
		recipe         string
		expander       string
	}{
		"Panel": {
			legacyFunction: "widgets.panel_init",
			recipe:         "recipe_region_panel",
			expander:       "morph.expand_region_panel",
		},
		"Button": {
			legacyFunction: "widgets.button_init",
			recipe:         "recipe_control_action",
			expander:       "morph.expand_control_action",
		},
		"TextBox": {
			legacyFunction: "widgets.textbox_init",
			recipe:         "recipe_field_text",
			expander:       "morph.expand_field_text",
		},
	}
	seen := map[string]SurfaceWidgetMigrationEquivalence{}
	var issues []string
	for _, row := range rows {
		widget := strings.TrimSpace(row.LegacyWidget)
		if widget == "" {
			issues = append(issues, "equivalence row legacy_widget is required")
			continue
		}
		if _, ok := seen[widget]; ok {
			issues = append(issues, fmt.Sprintf("duplicate equivalence row for %s", widget))
		}
		seen[widget] = row
		prefix := "equivalence row " + widget
		if row.BlockKind != "Block" {
			issues = append(issues, prefix+" block_kind must be Block")
		}
		if row.LegacyResult <= 0 || row.BlockResult <= 0 || row.LegacyResult != row.BlockResult {
			issues = append(
				issues,
				prefix+" legacy_result and block_result must match positive evidence",
			)
		}
		if !row.APIUnchanged {
			issues = append(issues, prefix+" api_unchanged must be true")
		}
		if !row.ResolvesToBlock {
			issues = append(issues, prefix+" resolves_to_block must be true")
		}
		if !row.Pass {
			issues = append(issues, prefix+" pass must be true")
		}
	}
	for widget, want := range required {
		row, ok := seen[widget]
		if !ok {
			issues = append(issues, fmt.Sprintf("equivalence_rows missing %s", widget))
			continue
		}
		if row.LegacyFunction != want.legacyFunction {
			issues = append(
				issues,
				fmt.Sprintf(
					"%s legacy_function is %q, want %q",
					widget,
					row.LegacyFunction,
					want.legacyFunction,
				),
			)
		}
		if row.MorphRecipe != want.recipe {
			issues = append(
				issues,
				fmt.Sprintf("%s morph_recipe is %q, want %q", widget, row.MorphRecipe, want.recipe),
			)
		}
		if row.BlockExpander != want.expander {
			issues = append(
				issues,
				fmt.Sprintf(
					"%s block_expander is %q, want %q",
					widget,
					row.BlockExpander,
					want.expander,
				),
			)
		}
	}
	return issues
}

func validateSurfaceWidgetMigrationMorphRecipes(
	recipes SurfaceWidgetMigrationMorphRecipes,
) []string {
	var issues []string
	issues = append(
		issues,
		validateExactStringList(
			"morph_recipe_migration.recipes",
			recipes.Recipes,
			[]string{"recipe_region_panel", "recipe_control_action", "recipe_field_text"},
		)...)
	issues = append(
		issues,
		validateExactStringList(
			"morph_recipe_migration.core_primitives",
			recipes.CorePrimitives,
			[]string{"Block"},
		)...)
	if !recipes.BlockOnlyCorePrimitive {
		issues = append(
			issues,
			"Block must be the only core primitive; future widget core primitive promotion is rejected",
		)
	}
	if recipes.WidgetsPromotedToCore {
		issues = append(issues, "widgets_promoted_to_core must be false")
	}
	if !recipes.ResolvesToBlock {
		issues = append(issues, "morph_recipe_migration resolves_to_block must be true")
	}
	if !recipes.Pass {
		issues = append(issues, "morph_recipe_migration pass must be true")
	}
	return issues
}

func validateSurfaceWidgetMigrationReferenceApp(app SurfaceWidgetMigrationReferenceApp) []string {
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "shape", got: app.Shape, want: "migration"},
		{
			field: "source",
			got:   app.Source,
			want:  "examples/surface/reference_forms/surface_reference_migration.tetra",
		},
	} {
		if check.got != check.want {
			issues = append(
				issues,
				fmt.Sprintf(
					"migration_reference_app %s is %q, want %q",
					check.field,
					check.got,
					check.want,
				),
			)
		}
	}
	if !safeRelativeSourcePath(app.Source) {
		issues = append(issues, "migration_reference_app source must be a safe Tetra source path")
	}
	for _, required := range []string{
		"lib.core.surface",
		"lib.core.block",
		"lib.core.morph",
		"lib.core.widgets",
	} {
		if !templateSmokeContainsString(app.Imports, required) {
			issues = append(
				issues,
				fmt.Sprintf("migration_reference_app imports missing %s", required),
			)
		}
	}
	if !app.Compiles {
		issues = append(issues, "migration_reference_app compiles must be true")
	}
	if !app.Runs || app.ExitCode != 0 {
		issues = append(issues, "migration_reference_app must run with exit_code 0")
	}
	if !app.UsesWidgetsCompat {
		issues = append(
			issues,
			"migration_reference_app requires lib.core.widgets compatibility usage",
		)
	}
	if !app.UsesMorphRecipes {
		issues = append(issues, "migration_reference_app requires Morph recipe usage")
	}
	if !app.ResolvesToBlock {
		issues = append(issues, "migration_reference_app resolves_to_block must be true")
	}
	if !app.Pass {
		issues = append(issues, "migration_reference_app pass must be true")
	}
	return issues
}

func validateSurfaceWidgetMigrationNegativeGuards(
	guards SurfaceWidgetMigrationNegativeGuards,
) []string {
	var issues []string
	for _, guard := range []struct {
		name string
		ok   bool
	}{
		{name: "no_future_core_primitive_promotion", ok: guards.NoFutureCorePrimitivePromotion},
		{name: "no_widget_primary_future_core", ok: guards.NoWidgetPrimaryFutureCore},
		{name: "no_breaking_change", ok: guards.NoBreakingChange},
		{name: "no_docs_only", ok: guards.NoDocsOnly},
		{name: "no_platform_native_runtime_claims", ok: guards.NoPlatformNativeRuntimeClaims},
	} {
		if !guard.ok {
			issues = append(issues, "negative guard "+guard.name+" must be true")
		}
	}
	return issues
}

func validateSurfaceWidgetMigrationArtifactEvidence(
	evidence SurfaceWidgetMigrationArtifactEvidence,
) []string {
	var issues []string
	if !validChecksumLike(evidence.EquivalenceRowsSHA256) {
		issues = append(
			issues,
			"artifact_evidence equivalence_rows_sha256 must be required sha256 evidence",
		)
	}
	if !validChecksumLike(evidence.SourceScanSHA256) {
		issues = append(
			issues,
			"artifact_evidence source_scan_sha256 must be required sha256 evidence",
		)
	}
	return issues
}

func surfaceWidgetMigrationReleaseWidgets() []string {
	return []string{
		"Text",
		"Label",
		"StatusText",
		"Button",
		"TextBox",
		"Row",
		"Column",
		"Panel",
		"Checkbox",
		"Stack",
		"Scroll",
		"Spacer",
	}
}
