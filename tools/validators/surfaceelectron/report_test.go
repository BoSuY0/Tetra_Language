package surfaceelectron

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSurfaceElectronComparisonAcceptsMethodFirstReport(t *testing.T) {
	if err := ValidateReport(mustComparisonJSON(t, validComparisonReport())); err != nil {
		t.Fatalf("ValidateReport returned error: %v", err)
	}
}

func TestSurfaceElectronComparisonRejectsOfficialBenchmarkClaim(t *testing.T) {
	report := validComparisonReport()
	report.Positioning.OfficialBenchmarkClaim = true
	report.NegativeGuards.OfficialBenchmarkClaimRejected = false
	err := ValidateReport(mustComparisonJSON(t, report))
	requireComparisonIssue(t, err, "official benchmark")
}

func TestSurfaceElectronComparisonRejectsCherryPickedHardware(t *testing.T) {
	report := validComparisonReport()
	report.Environment.CherryPickedHardware = true
	report.NegativeGuards.CherryPickedHardwareRejected = false
	err := ValidateReport(mustComparisonJSON(t, report))
	requireComparisonIssue(t, err, "cherry-picked")
}

func TestSurfaceElectronComparisonRejectsUnfairAppShape(t *testing.T) {
	report := validComparisonReport()
	report.AppPairs[0].SameFeatureSet = false
	report.AppPairs[0].UnfairAppShape = true
	report.NegativeGuards.UnfairAppShapeRejected = false
	err := ValidateReport(mustComparisonJSON(t, report))
	requireComparisonIssue(t, err, "app shape")
}

func TestSurfaceElectronComparisonRejectsMissingVariance(t *testing.T) {
	report := validComparisonReport()
	report.Method.VarianceReported = false
	report.Metrics[0].VarianceReported = false
	report.NegativeGuards.MissingVarianceRejected = false
	err := ValidateReport(mustComparisonJSON(t, report))
	requireComparisonIssue(t, err, "variance")
}

func TestSurfaceElectronComparisonRejectsMissingEnvironment(t *testing.T) {
	report := validComparisonReport()
	report.Environment.Hardware = ""
	report.Method.SameMeasurementTool = false
	report.NegativeGuards.MissingEnvironmentRejected = false
	err := ValidateReport(mustComparisonJSON(t, report))
	requireComparisonIssue(t, err, "environment")
}

func TestSurfaceElectronComparisonRejectsSingleSmokeFasterThanElectronClaim(t *testing.T) {
	report := validComparisonReport()
	report.Method.SampleCount = 1
	report.Positioning.FasterThanElectronClaim = true
	report.Positioning.Claim = "Surface is faster than Electron."
	report.NegativeGuards.SingleSmokeFasterThanElectronRejected = false
	err := ValidateReport(mustComparisonJSON(t, report))
	requireComparisonIssue(t, err, "faster than Electron")
}

func requireComparisonIssue(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected issue containing %q, got nil", want)
	}
	if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(want)) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func mustComparisonJSON(t *testing.T, report Report) []byte {
	t.Helper()
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func validComparisonReport() Report {
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
