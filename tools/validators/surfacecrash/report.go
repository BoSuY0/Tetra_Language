package surfacecrash

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

const (
	SchemaV1                = "tetra.surface.crash-report.v1"
	LevelCrashDiagnosticsV1 = "surface-crash-diagnostics-v1"
)

type Report struct {
	Schema         string         `json:"schema"`
	Status         string         `json:"status"`
	Level          string         `json:"level"`
	Scope          string         `json:"scope"`
	ReleaseScope   string         `json:"release_scope"`
	Producer       string         `json:"producer,omitempty"`
	GitHead        string         `json:"git_head"`
	SameCommit     bool           `json:"same_commit"`
	Version        string         `json:"version,omitempty"`
	Policy         CrashPolicy    `json:"policy"`
	Crashes        []CrashEntry   `json:"crashes"`
	Operations     []Operation    `json:"operations"`
	NegativeGuards NegativeGuards `json:"negative_guards"`
	NonClaims      []string       `json:"nonclaims"`
	Cases          []CaseReport   `json:"cases"`
}

type CrashPolicy struct {
	Name                  string `json:"name"`
	RestartPolicy         string `json:"restart_policy"`
	DevOverlay            bool   `json:"dev_overlay"`
	ProductionErrorHook   bool   `json:"production_error_hook"`
	ProductionDevOverlay  bool   `json:"production_dev_overlay"`
	SecretScrubbing       bool   `json:"secret_scrubbing"`
	ExpectedCrashBoundary bool   `json:"expected_crash_boundary"`
}

type CrashEntry struct {
	ID             string         `json:"id"`
	Kind           string         `json:"kind"`
	Status         string         `json:"status"`
	Expected       bool           `json:"expected"`
	Swallowed      bool           `json:"swallowed"`
	SurfacedToUser bool           `json:"surfaced_to_user"`
	RecoveryAction string         `json:"recovery_action"`
	ExitCode       int            `json:"exit_code"`
	Source         SourceLocation `json:"source"`
	Diagnostic     Diagnostic     `json:"diagnostic"`
	Bundle         ArtifactRef    `json:"bundle"`
	SecretScan     SecretScan     `json:"secret_scan"`
}

type SourceLocation struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Function string `json:"function"`
}

type Diagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Hint     string `json:"hint,omitempty"`
}

type ArtifactRef struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type SecretScan struct {
	Scanned         bool     `json:"scanned"`
	ContainsSecrets bool     `json:"contains_secrets"`
	RedactedFields  []string `json:"redacted_fields"`
}

type Operation struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Ran  bool   `json:"ran"`
	Pass bool   `json:"pass"`
}

type NegativeGuards struct {
	CrashSwallowedAsPassRejected     bool `json:"crash_swallowed_as_pass_rejected"`
	SecretLeakRejected               bool `json:"secret_leak_rejected"`
	MissingSourceLocationRejected    bool `json:"missing_source_location_rejected"`
	MissingDiagnosticBundleRejected  bool `json:"missing_diagnostic_bundle_rejected"`
	UnsurfacedErrorRejected          bool `json:"unsurfaced_error_rejected"`
	ExpectedNegativeCrashSeparation  bool `json:"expected_negative_crash_separation"`
	ProductionDevOverlayRejected     bool `json:"production_dev_overlay_rejected"`
	SameCommitCrashArtifactsRequired bool `json:"same_commit_crash_artifacts_required"`
}

type CaseReport struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Ran  bool   `json:"ran"`
	Pass bool   `json:"pass"`
}

func ValidateReport(raw []byte) error {
	report, err := decodeReport(raw)
	if err != nil {
		return err
	}
	return Validate(report)
}

func Validate(report Report) error {
	var issues []string
	issues = append(issues, validateIdentity(report)...)
	issues = append(issues, validatePolicy(report.Policy)...)
	issues = append(issues, validateCrashes(report.Crashes)...)
	issues = append(issues, validateOperations(report.Operations)...)
	issues = append(issues, validateNegativeGuards(report.NegativeGuards)...)
	issues = append(issues, validateNonClaims(report.NonClaims)...)
	issues = append(issues, validateCases(report.Cases)...)
	if len(issues) > 0 {
		sort.Strings(issues)
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func decodeReport(raw []byte) (Report, error) {
	var report Report
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&report); err != nil {
		return Report{}, err
	}
	if err := ensureJSONEOF(dec); err != nil {
		return Report{}, err
	}
	return report, nil
}

func ensureJSONEOF(dec *json.Decoder) error {
	var extra any
	if err := dec.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return fmt.Errorf("unexpected trailing JSON payload")
}

func validateIdentity(report Report) []string {
	var issues []string
	if report.Schema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, SchemaV1))
	}
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if report.Level != LevelCrashDiagnosticsV1 {
		issues = append(issues, fmt.Sprintf("level is %q, want %q", report.Level, LevelCrashDiagnosticsV1))
	}
	if report.Scope != "surface-v1-scoped-linux-web-crash-diagnostics" {
		issues = append(issues, fmt.Sprintf("scope is %q, want surface-v1-scoped-linux-web-crash-diagnostics", report.Scope))
	}
	if report.ReleaseScope != "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI" {
		issues = append(issues, fmt.Sprintf("release_scope is %q, want PROD_STABLE_SCOPED_LINUX_WEB_APP_UI", report.ReleaseScope))
	}
	if !validGitHead(report.GitHead) {
		issues = append(issues, "git_head must be a 40-hex same-commit revision")
	}
	if !report.SameCommit {
		issues = append(issues, "same_commit crash artifacts are required")
	}
	return issues
}

func validatePolicy(policy CrashPolicy) []string {
	var issues []string
	if policy.Name != "crash-safe-diagnostics-v1" {
		issues = append(issues, fmt.Sprintf("crash policy is %q, want crash-safe-diagnostics-v1", policy.Name))
	}
	if policy.RestartPolicy != "supervised-restart-opt-in-v1" {
		issues = append(issues, fmt.Sprintf("restart policy is %q, want supervised-restart-opt-in-v1", policy.RestartPolicy))
	}
	if !policy.DevOverlay {
		issues = append(issues, "development panic/error overlay evidence is required")
	}
	if !policy.ProductionErrorHook {
		issues = append(issues, "production error hook evidence is required")
	}
	if policy.ProductionDevOverlay {
		issues = append(issues, "production dev overlay is rejected")
	}
	if !policy.SecretScrubbing {
		issues = append(issues, "secret scrubbing policy is required")
	}
	if !policy.ExpectedCrashBoundary {
		issues = append(issues, "expected negative cases must be separated from crashes")
	}
	return issues
}

func validateCrashes(crashes []CrashEntry) []string {
	if len(crashes) == 0 {
		return []string{"crash diagnostics are required"}
	}
	var issues []string
	nonExpected := false
	expectedBoundary := false
	for i, crash := range crashes {
		prefix := fmt.Sprintf("crashes[%d]", i)
		if strings.TrimSpace(crash.ID) == "" || strings.TrimSpace(crash.Kind) == "" || strings.TrimSpace(crash.Status) == "" {
			issues = append(issues, prefix+" requires id, kind, and status")
		}
		switch crash.Status {
		case "recovered", "diagnostic", "failed":
		case "pass":
			issues = append(issues, fmt.Sprintf("crash %s swallowed as pass", crash.ID))
		default:
			issues = append(issues, fmt.Sprintf("crash %s has unsupported status %q", crash.ID, crash.Status))
		}
		if crash.Swallowed {
			issues = append(issues, fmt.Sprintf("crash %s swallowed without reported failure", crash.ID))
		}
		if !crash.SurfacedToUser {
			issues = append(issues, fmt.Sprintf("crash %s error must be surfaced to user or caller", crash.ID))
		}
		if strings.TrimSpace(crash.RecoveryAction) == "" {
			issues = append(issues, fmt.Sprintf("crash %s recovery_action is required", crash.ID))
		}
		if crash.ExitCode == 0 && !crash.Expected {
			issues = append(issues, fmt.Sprintf("crash %s must not use zero exit code for an unexpected failure", crash.ID))
		}
		issues = append(issues, validateSource(crash.ID, crash.Source)...)
		issues = append(issues, validateDiagnostic(crash.ID, crash.Diagnostic)...)
		issues = append(issues, validateBundle(crash.ID, crash.Bundle)...)
		issues = append(issues, validateSecretScan(crash.ID, crash.SecretScan, crash.Diagnostic)...)
		if crash.Expected {
			expectedBoundary = true
			if crash.Status != "diagnostic" {
				issues = append(issues, fmt.Sprintf("expected negative crash %s must be reported as diagnostic", crash.ID))
			}
		} else {
			nonExpected = true
		}
	}
	if !nonExpected {
		issues = append(issues, "at least one real failing app crash diagnostic is required")
	}
	if !expectedBoundary {
		issues = append(issues, "expected negative crash separation evidence is required")
	}
	return issues
}

func validateSource(id string, source SourceLocation) []string {
	var issues []string
	if strings.TrimSpace(source.File) == "" || source.Line <= 0 || source.Column <= 0 {
		issues = append(issues, fmt.Sprintf("crash %s source location requires file, line, and column", id))
	}
	if filepath.IsAbs(source.File) || strings.Contains(filepath.Clean(source.File), "..") {
		issues = append(issues, fmt.Sprintf("crash %s source location must be a safe repo-relative path", id))
	}
	if strings.TrimSpace(source.Function) == "" {
		issues = append(issues, fmt.Sprintf("crash %s source function is required", id))
	}
	return issues
}

func validateDiagnostic(id string, diagnostic Diagnostic) []string {
	var issues []string
	if strings.TrimSpace(diagnostic.Code) == "" || strings.TrimSpace(diagnostic.Message) == "" || strings.TrimSpace(diagnostic.Severity) == "" {
		issues = append(issues, fmt.Sprintf("crash %s diagnostic requires code, message, and severity", id))
	}
	switch diagnostic.Severity {
	case "error", "warning", "info":
	default:
		issues = append(issues, fmt.Sprintf("crash %s diagnostic severity %q is invalid", id, diagnostic.Severity))
	}
	if containsSecretMarker(diagnostic.Code) || containsSecretMarker(diagnostic.Message) || containsSecretMarker(diagnostic.Hint) {
		issues = append(issues, fmt.Sprintf("crash %s diagnostic contains secret material", id))
	}
	return issues
}

func validateBundle(id string, bundle ArtifactRef) []string {
	var issues []string
	if strings.TrimSpace(bundle.Path) == "" {
		issues = append(issues, fmt.Sprintf("crash %s diagnostic bundle path is required", id))
	}
	if filepath.IsAbs(bundle.Path) || strings.Contains(filepath.Clean(bundle.Path), "..") {
		issues = append(issues, fmt.Sprintf("crash %s diagnostic bundle path must be safe and repo-relative", id))
	}
	if bundle.Size <= 0 {
		issues = append(issues, fmt.Sprintf("crash %s diagnostic bundle size must be positive", id))
	}
	if !validSHA256(bundle.SHA256) {
		issues = append(issues, fmt.Sprintf("crash %s diagnostic bundle sha256 must be sha256:<64-hex>", id))
	}
	return issues
}

func validateSecretScan(id string, scan SecretScan, diagnostic Diagnostic) []string {
	var issues []string
	if !scan.Scanned {
		issues = append(issues, fmt.Sprintf("crash %s secret scan is required", id))
	}
	if scan.ContainsSecrets {
		issues = append(issues, fmt.Sprintf("crash %s secret scan found secret material", id))
	}
	if containsSecretMarker(diagnostic.Message) || containsSecretMarker(diagnostic.Hint) {
		issues = append(issues, fmt.Sprintf("crash %s diagnostic leaks secret material", id))
	}
	if len(scan.RedactedFields) == 0 {
		issues = append(issues, fmt.Sprintf("crash %s redacted_fields evidence is required", id))
	}
	return issues
}

func validateOperations(operations []Operation) []string {
	required := map[string]bool{
		"crash report schema validated":                  false,
		"secret scrubbing validated":                     false,
		"source locations validated":                     false,
		"error surfacing validated":                      false,
		"expected negative cases separated from crashes": false,
	}
	var issues []string
	for i, op := range operations {
		if strings.TrimSpace(op.Name) == "" || strings.TrimSpace(op.Kind) == "" {
			issues = append(issues, fmt.Sprintf("operations[%d] requires name and kind", i))
		}
		if !op.Ran || !op.Pass {
			issues = append(issues, fmt.Sprintf("operation %q must run and pass", op.Name))
		}
		if _, ok := required[op.Name]; ok {
			required[op.Name] = true
		}
	}
	for name, ok := range required {
		if !ok {
			issues = append(issues, fmt.Sprintf("operation %q is required", name))
		}
	}
	return issues
}

func validateNegativeGuards(guards NegativeGuards) []string {
	checks := map[string]bool{
		"crash swallowed as pass rejection":       guards.CrashSwallowedAsPassRejected,
		"secret leak rejection":                   guards.SecretLeakRejected,
		"missing source location rejection":       guards.MissingSourceLocationRejected,
		"missing diagnostic bundle rejection":     guards.MissingDiagnosticBundleRejected,
		"unsurfaced error rejection":              guards.UnsurfacedErrorRejected,
		"expected negative crash separation":      guards.ExpectedNegativeCrashSeparation,
		"production dev overlay rejection":        guards.ProductionDevOverlayRejected,
		"same-commit crash artifacts requirement": guards.SameCommitCrashArtifactsRequired,
	}
	var issues []string
	for name, ok := range checks {
		if !ok {
			issues = append(issues, name+" guard is required")
		}
	}
	return issues
}

func validateNonClaims(nonClaims []string) []string {
	if len(nonClaims) == 0 {
		return []string{"crash diagnostics nonclaims are required"}
	}
	joined := strings.ToLower(strings.Join(nonClaims, "\n"))
	var issues []string
	for _, required := range []string{"automatic crash recovery", "telemetry", "secret", "electron crash reporter"} {
		if !strings.Contains(joined, required) {
			issues = append(issues, fmt.Sprintf("crash diagnostics nonclaims must mention %s boundary", required))
		}
	}
	return issues
}

func validateCases(cases []CaseReport) []string {
	required := map[string]bool{
		"failing app produces useful diagnostics": false,
		"crash swallowed as pass rejected":        false,
		"error report includes secrets rejected":  false,
		"missing source location rejected":        false,
		"missing diagnostic bundle rejected":      false,
		"unsurfaced error rejected":               false,
		"expected negative separated from crash":  false,
		"production dev overlay rejected":         false,
	}
	var issues []string
	for i, c := range cases {
		if strings.TrimSpace(c.Name) == "" || strings.TrimSpace(c.Kind) == "" {
			issues = append(issues, fmt.Sprintf("cases[%d] requires name and kind", i))
		}
		if !c.Ran || !c.Pass {
			issues = append(issues, fmt.Sprintf("case %q must run and pass", c.Name))
		}
		if _, ok := required[c.Name]; ok {
			required[c.Name] = true
		}
	}
	for name, ok := range required {
		if !ok {
			issues = append(issues, fmt.Sprintf("case %q is required", name))
		}
	}
	return issues
}

func validGitHead(value string) bool {
	if len(value) != 40 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validSHA256(value string) bool {
	if !strings.HasPrefix(value, "sha256:") {
		return false
	}
	raw := strings.TrimPrefix(value, "sha256:")
	if len(raw) != 64 {
		return false
	}
	_, err := hex.DecodeString(raw)
	return err == nil
}

func containsSecretMarker(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{"token=", "password=", "api_key=", "secret=", "bearer "} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
