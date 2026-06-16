package surface

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type SurfacePerformanceBudgetReport struct {
	Schema            string                              `json:"schema"`
	Model             string                              `json:"model"`
	ReleaseScope      string                              `json:"release_scope"`
	Source            string                              `json:"source"`
	Target            string                              `json:"target"`
	Runtime           string                              `json:"runtime"`
	ProductionClaim   bool                                `json:"production_claim"`
	Experimental      bool                                `json:"experimental"`
	GitHead           string                              `json:"git_head"`
	PerformanceClaim  string                              `json:"performance_claim"`
	Startup           SurfaceStartupBudgetReport          `json:"startup"`
	Frame             SurfaceFrameBudgetReport            `json:"frame"`
	Scene             SurfaceSceneBudgetReport            `json:"scene"`
	Memory            SurfaceMemoryBudgetReport           `json:"memory"`
	Binary            SurfaceBinaryBudgetReport           `json:"binary"`
	CPUPowerProxy     SurfaceCPUPowerProxyReport          `json:"cpu_power_proxy"`
	Cache             SurfaceCacheBudgetReport            `json:"cache"`
	Methodology       SurfacePerformanceMethodologyReport `json:"methodology"`
	UnsupportedClaims []string                            `json:"unsupported_claims"`
	NegativeGuards    SurfacePerformanceNegativeGuards    `json:"negative_guards"`
}

type SurfaceStartupBudgetReport struct {
	LaunchToFirstFrameMS int    `json:"launch_to_first_frame_ms"`
	BudgetMS             int    `json:"budget_ms"`
	Trace                string `json:"trace"`
	Pass                 bool   `json:"pass"`
}

type SurfaceFrameBudgetReport struct {
	FrameCount    int  `json:"frame_count"`
	P50BuildMS    int  `json:"p50_build_ms"`
	P95BuildMS    int  `json:"p95_build_ms"`
	P50PresentMS  int  `json:"p50_present_ms"`
	P95PresentMS  int  `json:"p95_present_ms"`
	BudgetMS      int  `json:"budget_ms"`
	IdleLoopCount int  `json:"idle_loop_count"`
	WorkLoopCount int  `json:"work_loop_count"`
	Pass          bool `json:"pass"`
}

type SurfaceSceneBudgetReport struct {
	BlockCount           int `json:"block_count"`
	RecipeExpansionCount int `json:"recipe_expansion_count"`
	PaintCommandCount    int `json:"paint_command_count"`
	LayoutPassCount      int `json:"layout_pass_count"`
	TextRunCount         int `json:"text_run_count"`
}

type SurfaceMemoryBudgetReport struct {
	GlyphCacheBytes        int  `json:"glyph_cache_bytes"`
	AssetCacheBytes        int  `json:"asset_cache_bytes"`
	LayoutCacheBytes       int  `json:"layout_cache_bytes"`
	PaintCacheBytes        int  `json:"paint_cache_bytes"`
	FramebufferPeakBytes   int  `json:"framebuffer_peak_bytes"`
	FramebufferTotalBytes  int  `json:"framebuffer_total_bytes"`
	RSSMeasured            bool `json:"rss_measured"`
	PeakRSSBytes           int  `json:"peak_rss_bytes"`
	AllocationCount        int  `json:"allocation_count"`
	AllocationBytes        int  `json:"allocation_bytes"`
	BoundedCaches          bool `json:"bounded_caches"`
	UnboundedCacheRejected bool `json:"unbounded_cache_rejected"`
	Pass                   bool `json:"pass"`
}

type SurfaceBinaryBudgetReport struct {
	ArtifactPath string `json:"artifact_path"`
	SizeBytes    int    `json:"size_bytes"`
	BudgetBytes  int    `json:"budget_bytes"`
	Pass         bool   `json:"pass"`
}

type SurfaceCPUPowerProxyReport struct {
	IdleLoopCount     int  `json:"idle_loop_count"`
	WorkLoopCount     int  `json:"work_loop_count"`
	IdleFrameCount    int  `json:"idle_frame_count"`
	WorkFrameCount    int  `json:"work_frame_count"`
	RealPowerMeasured bool `json:"real_power_measured"`
	Pass              bool `json:"pass"`
}

type SurfaceCacheBudgetReport struct {
	GlyphCacheBudgetBytes  int    `json:"glyph_cache_budget_bytes"`
	AssetCacheBudgetBytes  int    `json:"asset_cache_budget_bytes"`
	LayoutCacheBudgetBytes int    `json:"layout_cache_budget_bytes"`
	PaintCacheBudgetBytes  int    `json:"paint_cache_budget_bytes"`
	TotalCacheBytes        int    `json:"total_cache_bytes"`
	TotalCacheBudgetBytes  int    `json:"total_cache_budget_bytes"`
	Eviction               string `json:"eviction"`
	Pass                   bool   `json:"pass"`
}

type SurfacePerformanceMethodologyReport struct {
	Kind                                   string `json:"kind"`
	ElectronComparison                     string `json:"electron_comparison"`
	OfficialBenchmark                      bool   `json:"official_benchmark"`
	CrossMachine                           bool   `json:"cross_machine"`
	FairComparisonRequiredForElectronClaim bool   `json:"fair_comparison_required_for_electron_claim"`
}

type SurfacePerformanceNegativeGuards struct {
	BoundedCaches             bool `json:"bounded_caches"`
	UnboundedCacheRejected    bool `json:"unbounded_cache_rejected"`
	StaleReportRejected       bool `json:"stale_report_rejected"`
	NoFasterThanElectronClaim bool `json:"no_faster_than_electron_claim"`
	NoBenchmarkParityClaim    bool `json:"no_benchmark_parity_claim"`
	PeakMemoryFieldRequired   bool `json:"peak_memory_field_required"`
	NoOfficialBenchmarkClaim  bool `json:"no_official_benchmark_claim"`
}

func ValidatePerformanceBudgetReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	switch schema {
	case PerformanceBudgetSchemaV1:
		var report SurfacePerformanceBudgetReport
		if err := decodeStrict(raw, &report); err != nil {
			return err
		}
		issues := validateSurfacePerformanceBudgetReport(report, nil, "")
		if !performanceBudgetPeakRSSFieldPresent(raw, false) {
			issues = append(issues, "surface_performance_budget memory peak_rss_bytes field is required")
		}
		if len(issues) > 0 {
			return errors.New(strings.Join(issues, "; "))
		}
		return nil
	case SchemaV1:
		var report Report
		if err := decodeStrict(raw, &report); err != nil {
			return err
		}
		issues := validateSurfacePerformanceBudgetEvidence(report)
		if report.SurfacePerformanceBudget != nil && !performanceBudgetPeakRSSFieldPresent(raw, true) {
			issues = append(issues, "surface_performance_budget memory peak_rss_bytes field is required")
		}
		if len(issues) > 0 {
			return errors.New(strings.Join(issues, "; "))
		}
		return nil
	default:
		return fmt.Errorf("schema is %q, want %q or %q", schema, PerformanceBudgetSchemaV1, SchemaV1)
	}
}

func validateSurfacePerformanceBudgetEvidence(report Report) []string {
	if report.SurfacePerformanceBudget == nil {
		if isLinuxAppShellReport(report) {
			return []string{"surface_performance_budget evidence is required for linux app-shell reports"}
		}
		return nil
	}
	return validateSurfacePerformanceBudgetReport(*report.SurfacePerformanceBudget, &report, report.Source)
}

func validateSurfacePerformanceBudgetReport(budget SurfacePerformanceBudgetReport, runtime *Report, source string) []string {
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: budget.Schema, want: PerformanceBudgetSchemaV1},
		{field: "model", got: budget.Model, want: "surface-performance-budget-v1"},
		{field: "release_scope", got: budget.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
	} {
		if check.got != check.want {
			issues = append(issues, fmt.Sprintf("surface_performance_budget %s is %q, want %q", check.field, check.got, check.want))
		}
	}
	if strings.TrimSpace(source) != "" && normalizeEvidencePath(budget.Source) != normalizeEvidencePath(source) {
		issues = append(issues, fmt.Sprintf("surface_performance_budget source %q must match report source %q", budget.Source, source))
	}
	if strings.TrimSpace(budget.Source) == "" {
		issues = append(issues, "surface_performance_budget source is required")
	}
	if !isSupportedRuntimeTarget(budget.Target) {
		issues = append(issues, fmt.Sprintf("surface_performance_budget target is %q, want headless, linux-x64, or wasm32-web", budget.Target))
	}
	if !isSupportedRuntimeName(budget.Runtime) {
		issues = append(issues, fmt.Sprintf("surface_performance_budget runtime is %q, want surface-headless, surface-linux-x64, or surface-wasm32-web", budget.Runtime))
	}
	if runtime != nil {
		if budget.Target != runtime.Target {
			issues = append(issues, fmt.Sprintf("surface_performance_budget target %q must match report target %q", budget.Target, runtime.Target))
		}
		if budget.Runtime != runtime.Runtime {
			issues = append(issues, fmt.Sprintf("surface_performance_budget runtime %q must match report runtime %q", budget.Runtime, runtime.Runtime))
		}
	}
	if !budget.ProductionClaim {
		issues = append(issues, "surface_performance_budget production_claim must be true")
	}
	if budget.Experimental {
		issues = append(issues, "surface_performance_budget experimental must be false")
	}
	if !isGitHead(budget.GitHead) {
		issues = append(issues, "surface_performance_budget git_head must be a 40-character hex commit")
	}
	if strings.TrimSpace(budget.PerformanceClaim) != "none" {
		issues = append(issues, fmt.Sprintf("surface_performance_budget performance_claim is %q, want none", budget.PerformanceClaim))
	}
	issues = append(issues, forbiddenBlockPerformanceClaimIssues("surface_performance_budget performance_claim", budget.PerformanceClaim)...)
	issues = append(issues, forbiddenBlockPerformanceClaimIssues("surface_performance_budget methodology", budget.Methodology.ElectronComparison)...)
	issues = append(issues, validateSurfaceStartupBudget(budget.Startup)...)
	issues = append(issues, validateSurfaceFrameBudget(budget.Frame)...)
	issues = append(issues, validateSurfaceSceneBudget(budget.Scene, runtime)...)
	issues = append(issues, validateSurfaceMemoryBudget(budget.Memory, runtime)...)
	issues = append(issues, validateSurfaceBinaryBudget(budget.Binary)...)
	issues = append(issues, validateSurfaceCPUPowerProxy(budget.CPUPowerProxy)...)
	issues = append(issues, validateSurfaceCacheBudget(budget.Cache, budget.Memory)...)
	issues = append(issues, validateSurfacePerformanceMethodology(budget.Methodology)...)
	issues = append(issues, validateSurfacePerformanceUnsupportedClaims(budget.UnsupportedClaims)...)
	issues = append(issues, validateSurfacePerformanceNegativeGuards(budget.NegativeGuards)...)
	return issues
}

func validateSurfaceStartupBudget(startup SurfaceStartupBudgetReport) []string {
	var issues []string
	if startup.LaunchToFirstFrameMS <= 0 {
		issues = append(issues, "surface_performance_budget startup launch_to_first_frame_ms must be positive")
	}
	if startup.BudgetMS <= 0 {
		issues = append(issues, "surface_performance_budget startup budget_ms must be positive")
	}
	if startup.BudgetMS > 0 && startup.LaunchToFirstFrameMS > startup.BudgetMS {
		issues = append(issues, fmt.Sprintf("surface_performance_budget startup launch_to_first_frame_ms %d exceeds budget_ms %d", startup.LaunchToFirstFrameMS, startup.BudgetMS))
	}
	if strings.TrimSpace(startup.Trace) == "" {
		issues = append(issues, "surface_performance_budget startup trace is required")
	}
	if !startup.Pass {
		issues = append(issues, "surface_performance_budget startup pass must be true")
	}
	return issues
}

func validateSurfaceFrameBudget(frame SurfaceFrameBudgetReport) []string {
	var issues []string
	if frame.FrameCount <= 0 {
		issues = append(issues, "surface_performance_budget frame frame_count must be positive")
	}
	for _, check := range []struct {
		field string
		value int
	}{
		{field: "p50_build_ms", value: frame.P50BuildMS},
		{field: "p95_build_ms", value: frame.P95BuildMS},
		{field: "p50_present_ms", value: frame.P50PresentMS},
		{field: "p95_present_ms", value: frame.P95PresentMS},
		{field: "budget_ms", value: frame.BudgetMS},
	} {
		if check.value <= 0 {
			issues = append(issues, fmt.Sprintf("surface_performance_budget frame %s must be positive", check.field))
		}
	}
	if frame.P95BuildMS < frame.P50BuildMS {
		issues = append(issues, "surface_performance_budget frame p95_build_ms must be >= p50_build_ms")
	}
	if frame.P95PresentMS < frame.P50PresentMS {
		issues = append(issues, "surface_performance_budget frame p95_present_ms must be >= p50_present_ms")
	}
	if frame.BudgetMS > 0 && (frame.P95BuildMS > frame.BudgetMS || frame.P95PresentMS > frame.BudgetMS) {
		issues = append(issues, "surface_performance_budget frame p95 build/present must fit within budget_ms")
	}
	if frame.IdleLoopCount < 0 || frame.WorkLoopCount < 0 {
		issues = append(issues, "surface_performance_budget frame idle/work loop counts must be non-negative")
	}
	if !frame.Pass {
		issues = append(issues, "surface_performance_budget frame pass must be true")
	}
	return issues
}

func validateSurfaceSceneBudget(scene SurfaceSceneBudgetReport, runtime *Report) []string {
	var issues []string
	if scene.BlockCount <= 0 {
		issues = append(issues, "surface_performance_budget scene block_count must be positive")
	}
	for _, check := range []struct {
		field string
		value int
	}{
		{field: "recipe_expansion_count", value: scene.RecipeExpansionCount},
		{field: "paint_command_count", value: scene.PaintCommandCount},
		{field: "layout_pass_count", value: scene.LayoutPassCount},
		{field: "text_run_count", value: scene.TextRunCount},
	} {
		if check.value < 0 {
			issues = append(issues, fmt.Sprintf("surface_performance_budget scene %s must be non-negative", check.field))
		}
	}
	if runtime != nil && len(runtime.Components) > 0 && scene.BlockCount < len(runtime.Components) {
		issues = append(issues, fmt.Sprintf("surface_performance_budget scene block_count = %d, want at least component count %d", scene.BlockCount, len(runtime.Components)))
	}
	return issues
}

func validateSurfaceMemoryBudget(memory SurfaceMemoryBudgetReport, runtime *Report) []string {
	var issues []string
	for _, check := range []struct {
		field string
		value int
	}{
		{field: "glyph_cache_bytes", value: memory.GlyphCacheBytes},
		{field: "asset_cache_bytes", value: memory.AssetCacheBytes},
		{field: "layout_cache_bytes", value: memory.LayoutCacheBytes},
		{field: "paint_cache_bytes", value: memory.PaintCacheBytes},
		{field: "allocation_count", value: memory.AllocationCount},
		{field: "allocation_bytes", value: memory.AllocationBytes},
	} {
		if check.value < 0 {
			issues = append(issues, fmt.Sprintf("surface_performance_budget memory %s must be non-negative", check.field))
		}
	}
	if memory.FramebufferPeakBytes <= 0 {
		issues = append(issues, "surface_performance_budget memory framebuffer_peak_bytes must be positive")
	}
	if memory.FramebufferTotalBytes < memory.FramebufferPeakBytes {
		issues = append(issues, "surface_performance_budget memory framebuffer_total_bytes must be >= framebuffer_peak_bytes")
	}
	if runtime != nil && len(runtime.Frames) > 0 {
		peak, total := blockFramebufferByteTotals(runtime.Frames)
		if memory.FramebufferPeakBytes < peak {
			issues = append(issues, fmt.Sprintf("surface_performance_budget memory framebuffer_peak_bytes = %d, want at least runtime peak %d", memory.FramebufferPeakBytes, peak))
		}
		if memory.FramebufferTotalBytes < total {
			issues = append(issues, fmt.Sprintf("surface_performance_budget memory framebuffer_total_bytes = %d, want at least runtime total %d", memory.FramebufferTotalBytes, total))
		}
	}
	if memory.RSSMeasured {
		if memory.PeakRSSBytes <= 0 {
			issues = append(issues, "surface_performance_budget memory peak_rss_bytes must be positive when rss_measured=true")
		}
	} else if memory.PeakRSSBytes != 0 {
		issues = append(issues, "surface_performance_budget memory peak_rss_bytes must be 0 when rss_measured=false")
	}
	if memory.AllocationCount <= 0 {
		issues = append(issues, "surface_performance_budget memory allocation_count must be positive")
	}
	cacheBytes := memory.GlyphCacheBytes + memory.AssetCacheBytes + memory.LayoutCacheBytes + memory.PaintCacheBytes
	if memory.AllocationBytes < memory.FramebufferPeakBytes+cacheBytes {
		issues = append(issues, fmt.Sprintf("surface_performance_budget memory allocation_bytes = %d, want at least framebuffer peak plus caches %d", memory.AllocationBytes, memory.FramebufferPeakBytes+cacheBytes))
	}
	if !memory.BoundedCaches {
		issues = append(issues, "surface_performance_budget memory bounded_caches must be true")
	}
	if !memory.UnboundedCacheRejected {
		issues = append(issues, "surface_performance_budget memory unbounded_cache_rejected must be true")
	}
	if !memory.Pass {
		issues = append(issues, "surface_performance_budget memory pass must be true")
	}
	return issues
}

func validateSurfaceBinaryBudget(binary SurfaceBinaryBudgetReport) []string {
	var issues []string
	if strings.TrimSpace(binary.ArtifactPath) == "" {
		issues = append(issues, "surface_performance_budget binary artifact_path is required")
	}
	if binary.SizeBytes <= 0 {
		issues = append(issues, "surface_performance_budget binary size_bytes must be positive")
	}
	if binary.BudgetBytes <= 0 {
		issues = append(issues, "surface_performance_budget binary budget_bytes must be positive")
	}
	if binary.BudgetBytes > 0 && binary.SizeBytes > binary.BudgetBytes {
		issues = append(issues, fmt.Sprintf("surface_performance_budget binary size_bytes %d exceeds budget_bytes %d", binary.SizeBytes, binary.BudgetBytes))
	}
	if !binary.Pass {
		issues = append(issues, "surface_performance_budget binary pass must be true")
	}
	return issues
}

func validateSurfaceCPUPowerProxy(proxy SurfaceCPUPowerProxyReport) []string {
	var issues []string
	for _, check := range []struct {
		field string
		value int
	}{
		{field: "idle_loop_count", value: proxy.IdleLoopCount},
		{field: "work_loop_count", value: proxy.WorkLoopCount},
		{field: "idle_frame_count", value: proxy.IdleFrameCount},
		{field: "work_frame_count", value: proxy.WorkFrameCount},
	} {
		if check.value < 0 {
			issues = append(issues, fmt.Sprintf("surface_performance_budget cpu_power_proxy %s must be non-negative", check.field))
		}
	}
	if proxy.RealPowerMeasured {
		issues = append(issues, "surface_performance_budget cpu_power_proxy real_power_measured must be false unless a real power harness is attached")
	}
	if !proxy.Pass {
		issues = append(issues, "surface_performance_budget cpu_power_proxy pass must be true")
	}
	return issues
}

func validateSurfaceCacheBudget(cache SurfaceCacheBudgetReport, memory SurfaceMemoryBudgetReport) []string {
	var issues []string
	for _, check := range []struct {
		field string
		value int
	}{
		{field: "glyph_cache_budget_bytes", value: cache.GlyphCacheBudgetBytes},
		{field: "asset_cache_budget_bytes", value: cache.AssetCacheBudgetBytes},
		{field: "layout_cache_budget_bytes", value: cache.LayoutCacheBudgetBytes},
		{field: "paint_cache_budget_bytes", value: cache.PaintCacheBudgetBytes},
		{field: "total_cache_budget_bytes", value: cache.TotalCacheBudgetBytes},
	} {
		if check.value <= 0 {
			issues = append(issues, fmt.Sprintf("surface_performance_budget cache %s must be positive", check.field))
		}
	}
	expectedTotal := memory.GlyphCacheBytes + memory.AssetCacheBytes + memory.LayoutCacheBytes + memory.PaintCacheBytes
	if cache.TotalCacheBytes != expectedTotal {
		issues = append(issues, fmt.Sprintf("surface_performance_budget cache total_cache_bytes = %d, want memory cache total %d", cache.TotalCacheBytes, expectedTotal))
	}
	if cache.TotalCacheBudgetBytes > 0 && cache.TotalCacheBytes > cache.TotalCacheBudgetBytes {
		issues = append(issues, "surface_performance_budget cache total_cache_bytes must fit within total_cache_budget_bytes")
	}
	if strings.TrimSpace(cache.Eviction) == "" {
		issues = append(issues, "surface_performance_budget cache eviction policy is required")
	}
	if !cache.Pass {
		issues = append(issues, "surface_performance_budget cache pass must be true")
	}
	return issues
}

func validateSurfacePerformanceMethodology(methodology SurfacePerformanceMethodologyReport) []string {
	var issues []string
	if methodology.Kind != "local-deterministic-budget-v1" {
		issues = append(issues, fmt.Sprintf("surface_performance_budget methodology kind is %q, want local-deterministic-budget-v1", methodology.Kind))
	}
	if strings.TrimSpace(methodology.ElectronComparison) != "none" {
		issues = append(issues, fmt.Sprintf("surface_performance_budget methodology electron_comparison is %q, want none", methodology.ElectronComparison))
	}
	if methodology.OfficialBenchmark {
		issues = append(issues, "surface_performance_budget methodology official_benchmark must be false")
	}
	if methodology.CrossMachine {
		issues = append(issues, "surface_performance_budget methodology cross_machine must be false")
	}
	if !methodology.FairComparisonRequiredForElectronClaim {
		issues = append(issues, "surface_performance_budget methodology requires fair_comparison_required_for_electron_claim=true")
	}
	return issues
}

func validateSurfacePerformanceUnsupportedClaims(claims []string) []string {
	var issues []string
	for _, claim := range claims {
		issues = append(issues, forbiddenBlockPerformanceClaimIssues("surface_performance_budget unsupported_claims", claim)...)
	}
	for _, required := range []string{
		"faster-than-electron",
		"lower-power-than-electron",
		"official-benchmark-result",
		"cross-machine-benchmark",
		"electron-parity-performance",
	} {
		if !containsExactText(claims, required) {
			issues = append(issues, fmt.Sprintf("surface_performance_budget unsupported_claims missing %q", required))
		}
	}
	return issues
}

func validateSurfacePerformanceNegativeGuards(guards SurfacePerformanceNegativeGuards) []string {
	if guards.BoundedCaches &&
		guards.UnboundedCacheRejected &&
		guards.StaleReportRejected &&
		guards.NoFasterThanElectronClaim &&
		guards.NoBenchmarkParityClaim &&
		guards.PeakMemoryFieldRequired &&
		guards.NoOfficialBenchmarkClaim {
		return nil
	}
	return []string{"surface_performance_budget negative_guards must require bounded caches, stale report rejection, peak memory field, and no unsupported Electron benchmark claims"}
}

func containsExactText(values []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == want {
			return true
		}
	}
	return false
}

func performanceBudgetPeakRSSFieldPresent(raw []byte, embedded bool) bool {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return false
	}
	var budgetRaw json.RawMessage
	if embedded {
		var ok bool
		budgetRaw, ok = root["surface_performance_budget"]
		if !ok {
			return false
		}
	} else {
		budgetRaw = raw
	}
	var budget map[string]json.RawMessage
	if err := json.Unmarshal(budgetRaw, &budget); err != nil {
		return false
	}
	var memory map[string]json.RawMessage
	if err := json.Unmarshal(budget["memory"], &memory); err != nil {
		return false
	}
	_, ok := memory["peak_rss_bytes"]
	return ok
}
