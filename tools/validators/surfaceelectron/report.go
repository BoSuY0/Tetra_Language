package surfaceelectron

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
	SchemaV1                  = "tetra.surface.electron-comparison-report.v1"
	LevelElectronComparisonV1 = "surface-electron-comparison-method-v1"
)

type Report struct {
	Schema         string            `json:"schema"`
	Status         string            `json:"status"`
	Level          string            `json:"level"`
	Scope          string            `json:"scope"`
	ReleaseScope   string            `json:"release_scope"`
	Producer       string            `json:"producer"`
	GitHead        string            `json:"git_head"`
	SameCommit     bool              `json:"same_commit"`
	Version        string            `json:"version,omitempty"`
	Method         Methodology       `json:"method"`
	Environment    Environment       `json:"environment"`
	AppPairs       []AppPair         `json:"app_pairs"`
	Metrics        []MetricResult    `json:"metrics"`
	Positioning    PublicPositioning `json:"public_positioning"`
	NegativeGuards NegativeGuards    `json:"negative_guards"`
	NonClaims      []string          `json:"nonclaims"`
	Cases          []CaseReport      `json:"cases"`
}

type Methodology struct {
	Published            bool   `json:"published"`
	SameAppShape         bool   `json:"same_app_shape"`
	SameFeatureSet       bool   `json:"same_feature_set"`
	SameOSTarget         bool   `json:"same_os_target"`
	SameColdWarmState    bool   `json:"same_cold_warm_state"`
	SameMeasurementTool  bool   `json:"same_measurement_tool"`
	SampleCount          int    `json:"sample_count"`
	WarmupRuns           int    `json:"warmup_runs"`
	VarianceReported     bool   `json:"variance_reported"`
	MethodArtifact       string `json:"method_artifact"`
	SurfaceArtifact      string `json:"surface_artifact"`
	ElectronArtifact     string `json:"electron_artifact"`
	ComparisonDataSource string `json:"comparison_data_source"`
}

type Environment struct {
	Hardware             string `json:"hardware"`
	OS                   string `json:"os"`
	Arch                 string `json:"arch"`
	PowerProfile         string `json:"power_profile"`
	MeasurementTool      string `json:"measurement_tool"`
	ColdWarmState        string `json:"cold_warm_state"`
	CherryPickedHardware bool   `json:"cherry_picked_hardware"`
}

type AppPair struct {
	Shape           string `json:"shape"`
	SurfaceApp      string `json:"surface_app"`
	ElectronApp     string `json:"electron_app"`
	SameFeatureSet  bool   `json:"same_feature_set"`
	SameAssets      bool   `json:"same_assets"`
	SameInputScript bool   `json:"same_input_script"`
	UnfairAppShape  bool   `json:"unfair_app_shape"`
}

type MetricResult struct {
	Name             string  `json:"name"`
	Target           string  `json:"target"`
	Unit             string  `json:"unit"`
	SurfaceMedian    float64 `json:"surface_median"`
	ElectronMedian   float64 `json:"electron_median"`
	SurfaceVariance  float64 `json:"surface_variance"`
	ElectronVariance float64 `json:"electron_variance"`
	SampleCount      int     `json:"sample_count"`
	VarianceReported bool    `json:"variance_reported"`
	SameEnvironment  bool    `json:"same_environment"`
	Competitive      bool    `json:"competitive"`
}

type PublicPositioning struct {
	Claim                       string `json:"claim"`
	CompetitiveInSupportedScope bool   `json:"competitive_in_supported_scope"`
	GeneratedFromReport         bool   `json:"generated_from_report"`
	OfficialBenchmarkClaim      bool   `json:"official_benchmark_claim"`
	FasterThanElectronClaim     bool   `json:"faster_than_electron_claim"`
	BroadElectronReplacement    bool   `json:"broad_electron_replacement"`
	ReactCSSCompatibilityClaim  bool   `json:"react_css_compatibility_claim"`
}

type NegativeGuards struct {
	OfficialBenchmarkClaimRejected        bool `json:"official_benchmark_claim_rejected"`
	CherryPickedHardwareRejected          bool `json:"cherry_picked_hardware_rejected"`
	MissingVarianceRejected               bool `json:"missing_variance_rejected"`
	MissingEnvironmentRejected            bool `json:"missing_environment_rejected"`
	UnfairAppShapeRejected                bool `json:"unfair_app_shape_rejected"`
	SingleSmokeFasterThanElectronRejected bool `json:"single_smoke_faster_than_electron_rejected"`
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
	issues = append(issues, validateMethod(report.Method)...)
	issues = append(issues, validateEnvironment(report.Environment)...)
	issues = append(issues, validateAppPairs(report.AppPairs)...)
	issues = append(issues, validateMetrics(report.Method, report.Metrics)...)
	issues = append(issues, validatePositioning(report.Positioning)...)
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
	if report.Level != LevelElectronComparisonV1 {
		issues = append(issues, fmt.Sprintf("level is %q, want %q", report.Level, LevelElectronComparisonV1))
	}
	if report.Scope != "surface-vs-electron-scoped-linux-web" {
		issues = append(issues, fmt.Sprintf("scope is %q, want surface-vs-electron-scoped-linux-web", report.Scope))
	}
	if report.ReleaseScope != "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI" {
		issues = append(issues, fmt.Sprintf("release_scope is %q, want PROD_STABLE_SCOPED_LINUX_WEB_APP_UI", report.ReleaseScope))
	}
	if strings.TrimSpace(report.Producer) == "" {
		issues = append(issues, "producer is required")
	}
	if !validGitHead(report.GitHead) {
		issues = append(issues, "git_head must be a 40-character lowercase hex commit")
	}
	if !report.SameCommit {
		issues = append(issues, "same_commit Electron comparison evidence is required")
	}
	return issues
}

func validateMethod(method Methodology) []string {
	var issues []string
	checks := []struct {
		ok    bool
		issue string
	}{
		{ok: method.Published, issue: "comparison method must be published"},
		{ok: method.SameAppShape, issue: "Electron comparison requires the same app shape"},
		{ok: method.SameFeatureSet, issue: "Electron comparison requires the same feature set"},
		{ok: method.SameOSTarget, issue: "Electron comparison requires the same OS/target"},
		{ok: method.SameColdWarmState, issue: "Electron comparison requires the same cold/warm state"},
		{ok: method.SameMeasurementTool, issue: "Electron comparison requires the same measurement tool and environment"},
		{ok: method.VarianceReported, issue: "Electron comparison requires variance reporting"},
	}
	for _, check := range checks {
		if !check.ok {
			issues = append(issues, check.issue)
		}
	}
	if method.SampleCount < 5 {
		issues = append(issues, "Electron comparison requires at least 5 samples; faster than Electron from one local smoke is rejected")
	}
	if method.WarmupRuns < 1 {
		issues = append(issues, "Electron comparison requires warmup runs")
	}
	for label, path := range map[string]string{
		"method artifact":   method.MethodArtifact,
		"Surface artifact":  method.SurfaceArtifact,
		"Electron artifact": method.ElectronArtifact,
	} {
		if err := validateSafeRelPath(path); err != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", label, err))
		}
	}
	if strings.TrimSpace(method.ComparisonDataSource) == "" {
		issues = append(issues, "comparison data source is required")
	}
	return issues
}

func validateEnvironment(env Environment) []string {
	required := map[string]string{
		"environment hardware":         env.Hardware,
		"environment os":               env.OS,
		"environment arch":             env.Arch,
		"environment power profile":    env.PowerProfile,
		"environment measurement tool": env.MeasurementTool,
		"environment cold/warm state":  env.ColdWarmState,
	}
	var issues []string
	for label, value := range required {
		if strings.TrimSpace(value) == "" {
			issues = append(issues, label+" is required")
		}
	}
	if env.CherryPickedHardware {
		issues = append(issues, "cherry-picked hardware is rejected")
	}
	return issues
}

func validateAppPairs(pairs []AppPair) []string {
	required := map[string]bool{"command_palette": false, "settings": false, "project_dashboard": false}
	if len(pairs) == 0 {
		return []string{"Electron comparison app pairs are required"}
	}
	var issues []string
	for _, pair := range pairs {
		shape := strings.TrimSpace(pair.Shape)
		if _, ok := required[shape]; ok {
			required[shape] = true
		}
		if err := validateSafeRelPath(pair.SurfaceApp); err != nil {
			issues = append(issues, fmt.Sprintf("Surface app path for %s: %v", shape, err))
		}
		if err := validateSafeRelPath(pair.ElectronApp); err != nil {
			issues = append(issues, fmt.Sprintf("Electron app path for %s: %v", shape, err))
		}
		if !pair.SameFeatureSet || !pair.SameAssets || !pair.SameInputScript || pair.UnfairAppShape {
			issues = append(issues, fmt.Sprintf("unfair app shape %s: same feature set, assets, and input script are required", shape))
		}
	}
	for shape, found := range required {
		if !found {
			issues = append(issues, fmt.Sprintf("missing Electron comparison app shape %s", shape))
		}
	}
	return issues
}

func validateMetrics(method Methodology, metrics []MetricResult) []string {
	required := map[string]bool{
		"startup_time_ms":      false,
		"rss_mb":               false,
		"first_frame_ms":       false,
		"input_latency_p95_ms": false,
		"idle_cpu_percent":     false,
		"package_size_mb":      false,
	}
	if len(metrics) == 0 {
		return []string{"Electron comparison metric rows are required"}
	}
	var issues []string
	seen := map[string]bool{}
	for _, metric := range metrics {
		name := strings.TrimSpace(metric.Name)
		if _, ok := required[name]; ok {
			required[name] = true
		}
		key := metric.Target + "/" + name
		if seen[key] {
			issues = append(issues, fmt.Sprintf("duplicate Electron comparison metric %s", key))
		}
		seen[key] = true
		if strings.TrimSpace(metric.Target) != "linux-x64" {
			issues = append(issues, fmt.Sprintf("metric %s target is %q, want linux-x64 for Electron desktop comparison", name, metric.Target))
		}
		if strings.TrimSpace(metric.Unit) == "" {
			issues = append(issues, fmt.Sprintf("metric %s unit is required", name))
		}
		if metric.SurfaceMedian <= 0 || metric.ElectronMedian <= 0 {
			issues = append(issues, fmt.Sprintf("metric %s requires positive Surface and Electron medians", name))
		}
		if metric.SurfaceVariance < 0 || metric.ElectronVariance < 0 {
			issues = append(issues, fmt.Sprintf("metric %s variance must be non-negative", name))
		}
		if metric.SampleCount < method.SampleCount || metric.SampleCount < 5 {
			issues = append(issues, fmt.Sprintf("metric %s requires at least the method sample count", name))
		}
		if !metric.VarianceReported {
			issues = append(issues, fmt.Sprintf("metric %s missing variance reporting", name))
		}
		if !metric.SameEnvironment {
			issues = append(issues, fmt.Sprintf("metric %s missing same environment evidence", name))
		}
		if !metric.Competitive {
			issues = append(issues, fmt.Sprintf("metric %s is not competitive in the supported scope", name))
		}
	}
	for name, found := range required {
		if !found {
			issues = append(issues, fmt.Sprintf("missing Electron comparison metric %s", name))
		}
	}
	return issues
}

func validatePositioning(positioning PublicPositioning) []string {
	var issues []string
	claim := strings.TrimSpace(positioning.Claim)
	normalized := strings.ToLower(claim)
	if claim == "" {
		issues = append(issues, "public positioning claim is required")
	}
	if !positioning.CompetitiveInSupportedScope {
		issues = append(issues, "public positioning must be limited to competitive in supported scope")
	}
	if !positioning.GeneratedFromReport {
		issues = append(issues, "public positioning must be generated from the comparison report")
	}
	if positioning.OfficialBenchmarkClaim || strings.Contains(normalized, "official benchmark") || strings.Contains(normalized, "official benchmark superiority") {
		issues = append(issues, "official benchmark claim is rejected")
	}
	if positioning.FasterThanElectronClaim || strings.Contains(normalized, "faster than electron") || strings.Contains(normalized, "faster-than-electron") {
		issues = append(issues, "faster than Electron from one local smoke is rejected")
	}
	if positioning.BroadElectronReplacement || strings.Contains(normalized, "broad electron replacement") || strings.Contains(normalized, "full electron replacement") {
		issues = append(issues, "broad Electron replacement claim is rejected")
	}
	if positioning.ReactCSSCompatibilityClaim || strings.Contains(normalized, "react compatibility") || strings.Contains(normalized, "css compatibility") {
		issues = append(issues, "React/CSS/Electron compatibility claim is rejected")
	}
	if !strings.Contains(normalized, "competitive with electron") || !strings.Contains(normalized, "supported") {
		issues = append(issues, "public positioning must say competitive with Electron in the supported scope")
	}
	return issues
}

func validateNegativeGuards(guards NegativeGuards) []string {
	required := map[string]bool{
		"official benchmark claim rejected":          guards.OfficialBenchmarkClaimRejected,
		"cherry-picked hardware rejected":            guards.CherryPickedHardwareRejected,
		"missing variance rejected":                  guards.MissingVarianceRejected,
		"missing environment rejected":               guards.MissingEnvironmentRejected,
		"unfair app shape rejected":                  guards.UnfairAppShapeRejected,
		"single-smoke faster-than-Electron rejected": guards.SingleSmokeFasterThanElectronRejected,
	}
	var issues []string
	for label, ok := range required {
		if !ok {
			issues = append(issues, label)
		}
	}
	return issues
}

func validateNonClaims(nonclaims []string) []string {
	required := []string{
		"official benchmark superiority",
		"faster-than-Electron claim from one local smoke",
		"broad Electron replacement",
		"React/CSS/Electron compatibility",
		"arbitrary Electron app migration",
	}
	haystack := strings.Join(nonclaims, "\n")
	var issues []string
	for _, want := range required {
		if !strings.Contains(haystack, want) {
			issues = append(issues, fmt.Sprintf("missing nonclaim containing %q", want))
		}
	}
	return issues
}

func validateCases(cases []CaseReport) []string {
	required := map[string]bool{
		"official benchmark claim rejected":                false,
		"cherry-picked hardware rejected":                  false,
		"missing variance rejected":                        false,
		"missing environment rejected":                     false,
		"unfair app shape rejected":                        false,
		"single-smoke faster-than-Electron claim rejected": false,
	}
	var issues []string
	for _, c := range cases {
		if _, ok := required[c.Name]; ok {
			required[c.Name] = c.Ran && c.Pass
		}
		if strings.TrimSpace(c.Name) == "" || strings.TrimSpace(c.Kind) == "" {
			issues = append(issues, "case reports require name and kind")
		}
		if !c.Ran || !c.Pass {
			issues = append(issues, fmt.Sprintf("case %s must run and pass", c.Name))
		}
	}
	for name, ok := range required {
		if !ok {
			issues = append(issues, fmt.Sprintf("missing passed case %s", name))
		}
	}
	return issues
}

func validateSafeRelPath(path string) error {
	path = strings.TrimSpace(filepath.ToSlash(path))
	if path == "" {
		return fmt.Errorf("path is required")
	}
	if filepath.IsAbs(path) || strings.HasPrefix(path, "../") || strings.Contains(path, "/../") {
		return fmt.Errorf("path %q must be relative and stay inside the report", path)
	}
	return nil
}

func validGitHead(value string) bool {
	if len(value) != 40 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && strings.ToLower(value) == value
}

func ValidFixtureReport() Report {
	return Report{
		Schema:       SchemaV1,
		Status:       "pass",
		Level:        LevelElectronComparisonV1,
		Scope:        "surface-vs-electron-scoped-linux-web",
		ReleaseScope: "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
		Producer:     "tools/cmd/surface-electron-comparison",
		GitHead:      "0123456789abcdef0123456789abcdef01234567",
		SameCommit:   true,
		Method: Methodology{
			Published:            true,
			SameAppShape:         true,
			SameFeatureSet:       true,
			SameOSTarget:         true,
			SameColdWarmState:    true,
			SameMeasurementTool:  true,
			SampleCount:          7,
			WarmupRuns:           2,
			VarianceReported:     true,
			MethodArtifact:       "method/surface-vs-electron-method.md",
			SurfaceArtifact:      "apps/surface-command-palette.tetra",
			ElectronArtifact:     "apps/electron-command-palette/package.json",
			ComparisonDataSource: "deterministic local harness with published method",
		},
		Environment: Environment{
			Hardware:             "ci-standard-8cpu-16gb",
			OS:                   "linux",
			Arch:                 "amd64",
			PowerProfile:         "performance governor captured",
			MeasurementTool:      "surface-electron-comparison deterministic harness",
			ColdWarmState:        "cold-and-warm",
			CherryPickedHardware: false,
		},
		AppPairs: []AppPair{
			{Shape: "command_palette", SurfaceApp: "examples/surface_prod_command_palette.tetra", ElectronApp: "benchmarks/electron/command_palette", SameFeatureSet: true, SameAssets: true, SameInputScript: true},
			{Shape: "settings", SurfaceApp: "examples/surface_prod_settings_app.tetra", ElectronApp: "benchmarks/electron/settings", SameFeatureSet: true, SameAssets: true, SameInputScript: true},
			{Shape: "project_dashboard", SurfaceApp: "examples/surface_prod_project_dashboard.tetra", ElectronApp: "benchmarks/electron/project_dashboard", SameFeatureSet: true, SameAssets: true, SameInputScript: true},
		},
		Metrics: []MetricResult{
			{Name: "startup_time_ms", Target: "linux-x64", Unit: "ms", SurfaceMedian: 120, ElectronMedian: 180, SurfaceVariance: 5.1, ElectronVariance: 7.2, SampleCount: 7, VarianceReported: true, SameEnvironment: true, Competitive: true},
			{Name: "rss_mb", Target: "linux-x64", Unit: "mb", SurfaceMedian: 78, ElectronMedian: 145, SurfaceVariance: 2.4, ElectronVariance: 6.9, SampleCount: 7, VarianceReported: true, SameEnvironment: true, Competitive: true},
			{Name: "first_frame_ms", Target: "linux-x64", Unit: "ms", SurfaceMedian: 64, ElectronMedian: 95, SurfaceVariance: 4.2, ElectronVariance: 5.8, SampleCount: 7, VarianceReported: true, SameEnvironment: true, Competitive: true},
			{Name: "input_latency_p95_ms", Target: "linux-x64", Unit: "ms", SurfaceMedian: 18, ElectronMedian: 24, SurfaceVariance: 1.1, ElectronVariance: 1.9, SampleCount: 7, VarianceReported: true, SameEnvironment: true, Competitive: true},
			{Name: "idle_cpu_percent", Target: "linux-x64", Unit: "percent", SurfaceMedian: 1.2, ElectronMedian: 2.6, SurfaceVariance: 0.2, ElectronVariance: 0.5, SampleCount: 7, VarianceReported: true, SameEnvironment: true, Competitive: true},
			{Name: "package_size_mb", Target: "linux-x64", Unit: "mb", SurfaceMedian: 9, ElectronMedian: 88, SurfaceVariance: 0.1, ElectronVariance: 1.4, SampleCount: 7, VarianceReported: true, SameEnvironment: true, Competitive: true},
		},
		Positioning: PublicPositioning{
			Claim:                       "Surface is competitive with Electron in the supported Linux/web scope.",
			CompetitiveInSupportedScope: true,
			GeneratedFromReport:         true,
			OfficialBenchmarkClaim:      false,
			FasterThanElectronClaim:     false,
			BroadElectronReplacement:    false,
			ReactCSSCompatibilityClaim:  false,
		},
		NegativeGuards: NegativeGuards{
			OfficialBenchmarkClaimRejected:        true,
			CherryPickedHardwareRejected:          true,
			MissingVarianceRejected:               true,
			MissingEnvironmentRejected:            true,
			UnfairAppShapeRejected:                true,
			SingleSmokeFasterThanElectronRejected: true,
		},
		NonClaims: []string{
			"No official benchmark superiority claim.",
			"No faster-than-Electron claim from one local smoke.",
			"No broad Electron replacement claim.",
			"No React/CSS/Electron compatibility claim.",
			"No arbitrary Electron app migration claim.",
		},
		Cases: []CaseReport{
			{Name: "official benchmark claim rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "cherry-picked hardware rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "missing variance rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "missing environment rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "unfair app shape rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "single-smoke faster-than-Electron claim rejected", Kind: "negative", Ran: true, Pass: true},
		},
	}
}
