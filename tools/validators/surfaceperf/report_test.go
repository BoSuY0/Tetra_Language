package surfaceperf

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSurfacePerfValidateReportAcceptsScopedPerformanceEvidence(t *testing.T) {
	if err := ValidateReport(mustReportJSON(t, validPerfReport())); err != nil {
		t.Fatalf("ValidateReport returned error: %v", err)
	}
}

func TestSurfacePerfValidateReportRejectsMissingBaselineEnvironment(t *testing.T) {
	report := validPerfReport()
	report.Baselines = nil
	report.Environment.Hardware = ""

	err := ValidateReport(mustReportJSON(t, report))
	if err == nil {
		t.Fatal("expected missing baseline/environment to be rejected")
	}
	if !strings.Contains(err.Error(), "baseline") || !strings.Contains(err.Error(), "environment") {
		t.Fatalf("error = %q, want baseline and environment rejection", err.Error())
	}
}

func TestSurfacePerfValidateReportRejectsImpossibleNumbers(t *testing.T) {
	report := validPerfReport()
	report.Budgets[0].Observed = 0
	report.Budgets[1].Observed = -1

	err := ValidateReport(mustReportJSON(t, report))
	if err == nil {
		t.Fatal("expected impossible performance numbers to be rejected")
	}
	if !strings.Contains(err.Error(), "positive") {
		t.Fatalf("error = %q, want positive value rejection", err.Error())
	}
}

func TestSurfacePerfValidateReportRejectsUnboundedCaches(t *testing.T) {
	report := validPerfReport()
	report.Caches.Bounded = false
	report.Caches.LayoutCacheBytes.Bounded = false

	err := ValidateReport(mustReportJSON(t, report))
	if err == nil {
		t.Fatal("expected unbounded cache evidence to be rejected")
	}
	if !strings.Contains(err.Error(), "bounded") || !strings.Contains(err.Error(), "cache") {
		t.Fatalf("error = %q, want bounded cache rejection", err.Error())
	}
}

func TestSurfacePerfValidateReportRejectsUnsupportedElectronSpeedClaim(t *testing.T) {
	report := validPerfReport()
	report.ElectronComparison.FasterThanElectronClaim = true
	report.ElectronComparison.StatisticallySupported = false
	report.ElectronComparison.SampleCount = 2

	err := ValidateReport(mustReportJSON(t, report))
	if err == nil {
		t.Fatal("expected unsupported Electron speed claim to be rejected")
	}
	if !strings.Contains(err.Error(), "Electron") || !strings.Contains(err.Error(), "statistically supported") {
		t.Fatalf("error = %q, want Electron statistical support rejection", err.Error())
	}
}

func validPerfReport() Report {
	return Report{
		Schema:       SchemaV1,
		Status:       "pass",
		Level:        LevelSurfacePerfV1,
		Scope:        "surface-v1-scoped-linux-web-performance-memory",
		ReleaseScope: "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
		GitHead:      "0123456789abcdef0123456789abcdef01234567",
		SameCommit:   true,
		Environment: Environment{
			Hardware:        "ci-standard-8cpu-16gb",
			OS:              "linux",
			Arch:            "amd64",
			Runtime:         "go test -buildvcs=false",
			PowerProfile:    "performance governor captured",
			ColdWarmState:   "cold-and-warm",
			MeasurementTool: "surface-perf-smoke deterministic harness",
		},
		Targets: []TargetEvidence{
			{Target: "linux-x64", Tier: "production", ProductionClaim: true, SmokeRan: true, Pass: true, Evidence: "linux-x64 real-window Surface perf smoke"},
			{Target: "wasm32-web", Tier: "production", ProductionClaim: true, SmokeRan: true, Pass: true, Evidence: "wasm32-web browser-canvas Surface perf smoke"},
			{Target: "windows-x64", Tier: "nonclaim", ProductionClaim: false, SmokeRan: false, Pass: true, Evidence: "blocked until Windows target-host perf evidence exists"},
			{Target: "macos-x64", Tier: "nonclaim", ProductionClaim: false, SmokeRan: false, Pass: true, Evidence: "blocked until macOS target-host perf evidence exists"},
		},
		Baselines: []Baseline{
			{Name: "surface-v1-linux-cold", Target: "linux-x64", Commit: "0123456789abcdef0123456789abcdef01234567", SameAppShape: true, SameOSTarget: true, SameColdWarmState: true, EnvironmentCaptured: true, Artifact: "baselines/surface-v1-linux-cold.json"},
			{Name: "surface-v1-web-warm", Target: "wasm32-web", Commit: "0123456789abcdef0123456789abcdef01234567", SameAppShape: true, SameOSTarget: true, SameColdWarmState: true, EnvironmentCaptured: true, Artifact: "baselines/surface-v1-web-warm.json"},
		},
		Budgets: []BudgetMeasurement{
			{Name: "startup_time", Unit: "ms", Budget: 250, Observed: 120, Pass: true},
			{Name: "first_frame_time", Unit: "ms", Budget: 120, Observed: 64, Pass: true},
			{Name: "steady_frame_time_p95", Unit: "ms", Budget: 16.7, Observed: 11.2, Pass: true},
			{Name: "peak_rss", Unit: "mb", Budget: 160, Observed: 78, Pass: true},
			{Name: "frame_allocations", Unit: "allocs/frame", Budget: 12, Observed: 4, Pass: true},
			{Name: "layout_cache_bytes", Unit: "bytes", Budget: 1048576, Observed: 262144, Pass: true},
			{Name: "glyph_cache_bytes", Unit: "bytes", Budget: 1048576, Observed: 196608, Pass: true},
			{Name: "asset_cache_bytes", Unit: "bytes", Budget: 4194304, Observed: 1048576, Pass: true},
			{Name: "binary_size", Unit: "bytes", Budget: 52428800, Observed: 9437184, Pass: true},
			{Name: "cpu_idle_power_proxy", Unit: "percent", Budget: 85, Observed: 92, Comparator: "greater_or_equal", Pass: true},
			{Name: "input_latency_p95", Unit: "ms", Budget: 50, Observed: 18, Pass: true},
			{Name: "animation_frame_jitter_p95", Unit: "ms", Budget: 4, Observed: 1.4, Pass: true},
		},
		Caches: CacheEvidence{
			Bounded:          true,
			LayoutCacheBytes: CacheBudget{Name: "layout", Limit: 1048576, Observed: 262144, Bounded: true, EvictionPolicy: "bounded-lru"},
			GlyphCacheBytes:  CacheBudget{Name: "glyph", Limit: 1048576, Observed: 196608, Bounded: true, EvictionPolicy: "bounded-lru"},
			AssetCacheBytes:  CacheBudget{Name: "asset", Limit: 4194304, Observed: 1048576, Bounded: true, EvictionPolicy: "bounded-lru"},
		},
		ElectronComparison: ElectronComparison{
			Enabled:                 true,
			SameAppShape:            true,
			SameOSTarget:            true,
			SameColdWarmState:       true,
			HardwareEnvironment:     true,
			StatisticallySupported:  false,
			SampleCount:             0,
			FasterThanElectronClaim: false,
			FastestUIFrameworkClaim: false,
			ZeroMemoryOverheadClaim: false,
			ComparisonArtifact:      "comparisons/electron-fairness-nonclaim.json",
			Decision:                "no faster-than-Electron claim; fair comparison harness shape recorded",
		},
		NegativeGuards: NegativeGuards{
			MissingBaselineRejected:               true,
			MissingEnvironmentRejected:            true,
			ImpossibleNumbersRejected:             true,
			UnboundedCacheRejected:                true,
			UnsupportedElectronSpeedClaimRejected: true,
			FastestUIFrameworkClaimRejected:       true,
			ZeroMemoryOverheadClaimRejected:       true,
		},
		NonClaims: []string{
			"No fastest UI framework claim.",
			"No faster-than-Electron claim without fair statistically supported comparison.",
			"No zero memory overhead claim.",
			"No cross-platform desktop performance parity claim.",
		},
		Cases: []CaseReport{
			{Name: "surface startup and first-frame budgets pass", Kind: "positive", Ran: true, Pass: true},
			{Name: "surface steady frame and jitter budgets pass", Kind: "positive", Ran: true, Pass: true},
			{Name: "missing baseline environment rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "impossible performance numbers rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "unbounded cache rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "unsupported Electron speed claim rejected", Kind: "negative", Ran: true, Pass: true},
		},
	}
}

func mustReportJSON(t *testing.T, report Report) []byte {
	t.Helper()
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
