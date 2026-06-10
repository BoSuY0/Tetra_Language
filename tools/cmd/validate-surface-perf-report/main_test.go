package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surfaceperf"
)

func TestValidateSurfacePerfReportCommandAcceptsValidReport(t *testing.T) {
	dir := t.TempDir()
	report := commandPerfReport()
	reportPath := filepath.Join(dir, "surface-perf-report.json")
	writePerfJSON(t, reportPath, report)

	var stdout, stderr bytes.Buffer
	code := runValidateSurfacePerfReport([]string{"--report", reportPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "surface performance report OK") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestValidateSurfacePerfReportCommandRejectsFastestClaim(t *testing.T) {
	dir := t.TempDir()
	report := commandPerfReport()
	report.ElectronComparison.FastestUIFrameworkClaim = true
	reportPath := filepath.Join(dir, "surface-perf-report.json")
	writePerfJSON(t, reportPath, report)

	var stdout, stderr bytes.Buffer
	code := runValidateSurfacePerfReport([]string{"--report", reportPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected nonzero exit, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "fastest UI framework") {
		t.Fatalf("stderr = %q, want fastest UI framework rejection", stderr.String())
	}
}

func commandPerfReport() surfaceperf.Report {
	report := surfaceperf.Report{
		Schema:       surfaceperf.SchemaV1,
		Status:       "pass",
		Level:        surfaceperf.LevelSurfacePerfV1,
		Scope:        "surface-v1-scoped-linux-web-performance-memory",
		ReleaseScope: "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
		GitHead:      "0123456789abcdef0123456789abcdef01234567",
		SameCommit:   true,
		Environment: surfaceperf.Environment{
			Hardware:        "ci-standard-8cpu-16gb",
			OS:              "linux",
			Arch:            "amd64",
			Runtime:         "go test -buildvcs=false",
			PowerProfile:    "performance governor captured",
			ColdWarmState:   "cold-and-warm",
			MeasurementTool: "surface-perf-smoke deterministic harness",
		},
		Targets: []surfaceperf.TargetEvidence{
			{Target: "linux-x64", Tier: "production", ProductionClaim: true, SmokeRan: true, Pass: true, Evidence: "linux perf smoke"},
			{Target: "wasm32-web", Tier: "production", ProductionClaim: true, SmokeRan: true, Pass: true, Evidence: "web perf smoke"},
		},
		Baselines: []surfaceperf.Baseline{
			{Name: "surface-v1-linux-cold", Target: "linux-x64", Commit: "0123456789abcdef0123456789abcdef01234567", SameAppShape: true, SameOSTarget: true, SameColdWarmState: true, EnvironmentCaptured: true, Artifact: "baselines/surface-v1-linux-cold.json"},
			{Name: "surface-v1-web-warm", Target: "wasm32-web", Commit: "0123456789abcdef0123456789abcdef01234567", SameAppShape: true, SameOSTarget: true, SameColdWarmState: true, EnvironmentCaptured: true, Artifact: "baselines/surface-v1-web-warm.json"},
		},
		Budgets: []surfaceperf.BudgetMeasurement{
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
		Caches: surfaceperf.CacheEvidence{
			Bounded:          true,
			LayoutCacheBytes: surfaceperf.CacheBudget{Name: "layout", Limit: 1048576, Observed: 262144, Bounded: true, EvictionPolicy: "bounded-lru"},
			GlyphCacheBytes:  surfaceperf.CacheBudget{Name: "glyph", Limit: 1048576, Observed: 196608, Bounded: true, EvictionPolicy: "bounded-lru"},
			AssetCacheBytes:  surfaceperf.CacheBudget{Name: "asset", Limit: 4194304, Observed: 1048576, Bounded: true, EvictionPolicy: "bounded-lru"},
		},
		ElectronComparison: surfaceperf.ElectronComparison{
			Enabled:                 true,
			SameAppShape:            true,
			SameOSTarget:            true,
			SameColdWarmState:       true,
			HardwareEnvironment:     true,
			StatisticallySupported:  false,
			FasterThanElectronClaim: false,
			FastestUIFrameworkClaim: false,
			ZeroMemoryOverheadClaim: false,
			ComparisonArtifact:      "comparisons/electron-fairness-nonclaim.json",
			Decision:                "no faster-than-Electron claim; fair comparison harness shape recorded",
		},
		NegativeGuards: surfaceperf.NegativeGuards{
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
		Cases: []surfaceperf.CaseReport{
			{Name: "surface startup and first-frame budgets pass", Kind: "positive", Ran: true, Pass: true},
			{Name: "surface steady frame and jitter budgets pass", Kind: "positive", Ran: true, Pass: true},
			{Name: "missing baseline environment rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "impossible performance numbers rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "unbounded cache rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "unsupported Electron speed claim rejected", Kind: "negative", Ran: true, Pass: true},
		},
	}
	return report
}

func writePerfJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}
