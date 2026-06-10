package surfaceperf

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
	SchemaV1           = "tetra.surface.perf-report.v1"
	LevelSurfacePerfV1 = "surface-performance-memory-v1"
)

type Report struct {
	Schema             string              `json:"schema"`
	Status             string              `json:"status"`
	Level              string              `json:"level"`
	Scope              string              `json:"scope"`
	ReleaseScope       string              `json:"release_scope"`
	Producer           string              `json:"producer,omitempty"`
	GitHead            string              `json:"git_head"`
	SameCommit         bool                `json:"same_commit"`
	Version            string              `json:"version,omitempty"`
	Environment        Environment         `json:"environment"`
	Targets            []TargetEvidence    `json:"targets"`
	Baselines          []Baseline          `json:"baselines"`
	Budgets            []BudgetMeasurement `json:"budgets"`
	Caches             CacheEvidence       `json:"caches"`
	ElectronComparison ElectronComparison  `json:"electron_comparison"`
	NegativeGuards     NegativeGuards      `json:"negative_guards"`
	NonClaims          []string            `json:"nonclaims"`
	Cases              []CaseReport        `json:"cases"`
}

type Environment struct {
	Hardware        string `json:"hardware"`
	OS              string `json:"os"`
	Arch            string `json:"arch"`
	Runtime         string `json:"runtime"`
	PowerProfile    string `json:"power_profile"`
	ColdWarmState   string `json:"cold_warm_state"`
	MeasurementTool string `json:"measurement_tool"`
}

type TargetEvidence struct {
	Target          string `json:"target"`
	Tier            string `json:"tier"`
	ProductionClaim bool   `json:"production_claim"`
	SmokeRan        bool   `json:"smoke_ran"`
	Pass            bool   `json:"pass"`
	Evidence        string `json:"evidence"`
}

type Baseline struct {
	Name                string `json:"name"`
	Target              string `json:"target"`
	Commit              string `json:"commit"`
	SameAppShape        bool   `json:"same_app_shape"`
	SameOSTarget        bool   `json:"same_os_target"`
	SameColdWarmState   bool   `json:"same_cold_warm_state"`
	EnvironmentCaptured bool   `json:"environment_captured"`
	Artifact            string `json:"artifact"`
}

type BudgetMeasurement struct {
	Name       string  `json:"name"`
	Unit       string  `json:"unit"`
	Budget     float64 `json:"budget"`
	Observed   float64 `json:"observed"`
	Comparator string  `json:"comparator,omitempty"`
	Pass       bool    `json:"pass"`
}

type CacheEvidence struct {
	Bounded          bool        `json:"bounded"`
	LayoutCacheBytes CacheBudget `json:"layout_cache_bytes"`
	GlyphCacheBytes  CacheBudget `json:"glyph_cache_bytes"`
	AssetCacheBytes  CacheBudget `json:"asset_cache_bytes"`
}

type CacheBudget struct {
	Name           string  `json:"name"`
	Limit          float64 `json:"limit"`
	Observed       float64 `json:"observed"`
	Bounded        bool    `json:"bounded"`
	EvictionPolicy string  `json:"eviction_policy"`
}

type ElectronComparison struct {
	Enabled                 bool   `json:"enabled"`
	SameAppShape            bool   `json:"same_app_shape"`
	SameOSTarget            bool   `json:"same_os_target"`
	SameColdWarmState       bool   `json:"same_cold_warm_state"`
	HardwareEnvironment     bool   `json:"hardware_environment"`
	StatisticallySupported  bool   `json:"statistically_supported"`
	SampleCount             int    `json:"sample_count"`
	FasterThanElectronClaim bool   `json:"faster_than_electron_claim"`
	FastestUIFrameworkClaim bool   `json:"fastest_ui_framework_claim"`
	ZeroMemoryOverheadClaim bool   `json:"zero_memory_overhead_claim"`
	ComparisonArtifact      string `json:"comparison_artifact"`
	Decision                string `json:"decision"`
}

type NegativeGuards struct {
	MissingBaselineRejected               bool `json:"missing_baseline_rejected"`
	MissingEnvironmentRejected            bool `json:"missing_environment_rejected"`
	ImpossibleNumbersRejected             bool `json:"impossible_numbers_rejected"`
	UnboundedCacheRejected                bool `json:"unbounded_cache_rejected"`
	UnsupportedElectronSpeedClaimRejected bool `json:"unsupported_electron_speed_claim_rejected"`
	FastestUIFrameworkClaimRejected       bool `json:"fastest_ui_framework_claim_rejected"`
	ZeroMemoryOverheadClaimRejected       bool `json:"zero_memory_overhead_claim_rejected"`
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
	issues = append(issues, validateEnvironment(report.Environment)...)
	issues = append(issues, validateTargets(report.Targets)...)
	issues = append(issues, validateBaselines(report.Baselines)...)
	issues = append(issues, validateBudgets(report.Budgets)...)
	issues = append(issues, validateCaches(report.Caches)...)
	issues = append(issues, validateElectronComparison(report.ElectronComparison)...)
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
	if report.Level != LevelSurfacePerfV1 {
		issues = append(issues, fmt.Sprintf("level is %q, want %q", report.Level, LevelSurfacePerfV1))
	}
	if report.Scope != "surface-v1-scoped-linux-web-performance-memory" {
		issues = append(issues, fmt.Sprintf("scope is %q, want surface-v1-scoped-linux-web-performance-memory", report.Scope))
	}
	if report.ReleaseScope != "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI" {
		issues = append(issues, fmt.Sprintf("release_scope is %q, want PROD_STABLE_SCOPED_LINUX_WEB_APP_UI", report.ReleaseScope))
	}
	if !validGitHead(report.GitHead) {
		issues = append(issues, "git_head must be a 40-hex same-commit revision")
	}
	if !report.SameCommit {
		issues = append(issues, "same_commit performance evidence is required")
	}
	return issues
}

func validateEnvironment(env Environment) []string {
	required := map[string]string{
		"environment hardware":         env.Hardware,
		"environment os":               env.OS,
		"environment arch":             env.Arch,
		"environment runtime":          env.Runtime,
		"environment power profile":    env.PowerProfile,
		"environment cold/warm state":  env.ColdWarmState,
		"environment measurement tool": env.MeasurementTool,
	}
	var issues []string
	for label, value := range required {
		if strings.TrimSpace(value) == "" {
			issues = append(issues, label+" is required")
		}
	}
	if strings.Contains(strings.ToLower(env.Hardware), "unknown") {
		issues = append(issues, "environment hardware must not be unknown")
	}
	return issues
}

func validateTargets(targets []TargetEvidence) []string {
	if len(targets) == 0 {
		return []string{"target performance status evidence is required"}
	}
	var issues []string
	required := map[string]bool{"linux-x64": false, "wasm32-web": false}
	for _, target := range targets {
		if target.Target == "linux-x64" || target.Target == "wasm32-web" {
			required[target.Target] = target.ProductionClaim && target.SmokeRan && target.Pass
		}
		if strings.TrimSpace(target.Target) == "" || strings.TrimSpace(target.Tier) == "" {
			issues = append(issues, "target evidence requires target and tier")
		}
		if strings.TrimSpace(target.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("target %s performance evidence is required", target.Target))
		}
		if target.ProductionClaim {
			if target.Target != "linux-x64" && target.Target != "wasm32-web" {
				issues = append(issues, fmt.Sprintf("target %s performance production claim is unsupported", target.Target))
			}
			if !target.SmokeRan || !target.Pass {
				issues = append(issues, fmt.Sprintf("target %s production performance claim requires passing smoke", target.Target))
			}
		}
		if target.SmokeRan && !target.Pass {
			issues = append(issues, fmt.Sprintf("target %s performance smoke failed", target.Target))
		}
	}
	for target, ok := range required {
		if !ok {
			issues = append(issues, fmt.Sprintf("target %s requires scoped production performance smoke", target))
		}
	}
	return issues
}

func validateBaselines(baselines []Baseline) []string {
	if len(baselines) == 0 {
		return []string{"baseline evidence is required with captured environment"}
	}
	required := map[string]bool{"linux-x64": false, "wasm32-web": false}
	var issues []string
	seen := map[string]bool{}
	for _, baseline := range baselines {
		name := strings.TrimSpace(baseline.Name)
		if name == "" {
			issues = append(issues, "baseline name is required")
		} else if seen[name] {
			issues = append(issues, fmt.Sprintf("duplicate baseline %s", name))
		}
		seen[name] = true
		if _, ok := required[baseline.Target]; ok {
			required[baseline.Target] = true
		}
		if baseline.Target == "" {
			issues = append(issues, fmt.Sprintf("baseline %s target is required", name))
		}
		if !validGitHead(baseline.Commit) {
			issues = append(issues, fmt.Sprintf("baseline %s commit must be 40-hex", name))
		}
		if !baseline.SameAppShape {
			issues = append(issues, fmt.Sprintf("baseline %s must use the same app shape", name))
		}
		if !baseline.SameOSTarget {
			issues = append(issues, fmt.Sprintf("baseline %s must use the same OS/target", name))
		}
		if !baseline.SameColdWarmState {
			issues = append(issues, fmt.Sprintf("baseline %s must use the same cold/warm state", name))
		}
		if !baseline.EnvironmentCaptured {
			issues = append(issues, fmt.Sprintf("baseline %s must capture environment", name))
		}
		if err := validateSafeRelPath(baseline.Artifact); err != nil {
			issues = append(issues, fmt.Sprintf("baseline %s artifact: %v", name, err))
		}
	}
	for target, ok := range required {
		if !ok {
			issues = append(issues, fmt.Sprintf("baseline for target %s is required", target))
		}
	}
	return issues
}

func validateBudgets(budgets []BudgetMeasurement) []string {
	required := map[string]bool{
		"startup_time":               false,
		"first_frame_time":           false,
		"steady_frame_time_p95":      false,
		"peak_rss":                   false,
		"frame_allocations":          false,
		"layout_cache_bytes":         false,
		"glyph_cache_bytes":          false,
		"asset_cache_bytes":          false,
		"binary_size":                false,
		"cpu_idle_power_proxy":       false,
		"input_latency_p95":          false,
		"animation_frame_jitter_p95": false,
	}
	if len(budgets) == 0 {
		return []string{"performance budget measurements are required"}
	}
	var issues []string
	seen := map[string]bool{}
	for _, budget := range budgets {
		name := strings.TrimSpace(budget.Name)
		if name == "" {
			issues = append(issues, "budget measurement name is required")
			continue
		}
		if seen[name] {
			issues = append(issues, fmt.Sprintf("duplicate budget measurement %s", name))
		}
		seen[name] = true
		if _, ok := required[name]; ok {
			required[name] = true
		}
		if strings.TrimSpace(budget.Unit) == "" {
			issues = append(issues, fmt.Sprintf("budget %s unit is required", name))
		}
		if budget.Budget <= 0 {
			issues = append(issues, fmt.Sprintf("budget %s budget must be positive", name))
		}
		if budget.Observed <= 0 {
			issues = append(issues, fmt.Sprintf("budget %s observed value must be positive", name))
		}
		if !budget.Pass {
			issues = append(issues, fmt.Sprintf("budget %s did not pass", name))
		}
		switch budget.Comparator {
		case "", "less_or_equal":
			if budget.Budget > 0 && budget.Observed > budget.Budget {
				issues = append(issues, fmt.Sprintf("budget %s observed %.3f exceeds budget %.3f", name, budget.Observed, budget.Budget))
			}
		case "greater_or_equal":
			if budget.Budget > 0 && budget.Observed < budget.Budget {
				issues = append(issues, fmt.Sprintf("budget %s observed %.3f below minimum %.3f", name, budget.Observed, budget.Budget))
			}
		default:
			issues = append(issues, fmt.Sprintf("budget %s comparator is %q, want less_or_equal or greater_or_equal", name, budget.Comparator))
		}
	}
	for name, ok := range required {
		if !ok {
			issues = append(issues, fmt.Sprintf("missing required performance budget %s", name))
		}
	}
	return issues
}

func validateCaches(caches CacheEvidence) []string {
	var issues []string
	if !caches.Bounded {
		issues = append(issues, "bounded cache evidence is required")
	}
	for _, cache := range []CacheBudget{caches.LayoutCacheBytes, caches.GlyphCacheBytes, caches.AssetCacheBytes} {
		name := strings.TrimSpace(cache.Name)
		if name == "" {
			issues = append(issues, "cache budget name is required")
			continue
		}
		if !cache.Bounded {
			issues = append(issues, fmt.Sprintf("cache %s must be bounded", name))
		}
		if cache.Limit <= 0 {
			issues = append(issues, fmt.Sprintf("cache %s limit must be positive", name))
		}
		if cache.Observed <= 0 {
			issues = append(issues, fmt.Sprintf("cache %s observed value must be positive", name))
		}
		if cache.Limit > 0 && cache.Observed > cache.Limit {
			issues = append(issues, fmt.Sprintf("cache %s observed %.3f exceeds limit %.3f", name, cache.Observed, cache.Limit))
		}
		if !strings.Contains(strings.ToLower(cache.EvictionPolicy), "bounded") && !strings.Contains(strings.ToLower(cache.EvictionPolicy), "lru") {
			issues = append(issues, fmt.Sprintf("cache %s requires bounded eviction policy", name))
		}
	}
	return issues
}

func validateElectronComparison(comparison ElectronComparison) []string {
	var issues []string
	if !comparison.Enabled {
		return []string{"Electron comparison fairness record is required, even as a nonclaim"}
	}
	if !comparison.SameAppShape {
		issues = append(issues, "Electron comparison requires the same app shape")
	}
	if !comparison.SameOSTarget {
		issues = append(issues, "Electron comparison requires the same OS/target")
	}
	if !comparison.SameColdWarmState {
		issues = append(issues, "Electron comparison requires the same cold/warm state")
	}
	if !comparison.HardwareEnvironment {
		issues = append(issues, "Electron comparison requires captured hardware/environment")
	}
	if err := validateSafeRelPath(comparison.ComparisonArtifact); err != nil {
		issues = append(issues, fmt.Sprintf("Electron comparison artifact: %v", err))
	}
	if strings.TrimSpace(comparison.Decision) == "" {
		issues = append(issues, "Electron comparison decision is required")
	}
	if comparison.FasterThanElectronClaim {
		if !comparison.StatisticallySupported || comparison.SampleCount < 5 || !comparison.SameAppShape || !comparison.SameOSTarget || !comparison.SameColdWarmState || !comparison.HardwareEnvironment {
			issues = append(issues, "faster-than-Electron claim requires fair statistically supported comparison with at least 5 samples")
		}
	}
	if comparison.FastestUIFrameworkClaim {
		issues = append(issues, "fastest UI framework claim is rejected")
	}
	if comparison.ZeroMemoryOverheadClaim {
		issues = append(issues, "zero memory overhead claim is rejected")
	}
	if strings.Contains(strings.ToLower(comparison.Decision), "fastest ui framework") && !strings.Contains(strings.ToLower(comparison.Decision), "no fastest") {
		issues = append(issues, "fastest UI framework decision claim is rejected")
	}
	if strings.Contains(strings.ToLower(comparison.Decision), "zero memory overhead") && !strings.Contains(strings.ToLower(comparison.Decision), "no zero") {
		issues = append(issues, "zero memory overhead decision claim is rejected")
	}
	return issues
}

func validateNegativeGuards(guards NegativeGuards) []string {
	required := map[string]bool{
		"missing baseline rejected":                 guards.MissingBaselineRejected,
		"missing environment rejected":              guards.MissingEnvironmentRejected,
		"impossible numbers rejected":               guards.ImpossibleNumbersRejected,
		"unbounded cache rejected":                  guards.UnboundedCacheRejected,
		"unsupported Electron speed claim rejected": guards.UnsupportedElectronSpeedClaimRejected,
		"fastest UI framework claim rejected":       guards.FastestUIFrameworkClaimRejected,
		"zero memory overhead claim rejected":       guards.ZeroMemoryOverheadClaimRejected,
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
	required := []string{"fastest UI framework", "faster-than-Electron", "zero memory overhead", "cross-platform desktop performance"}
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
		"surface startup and first-frame budgets pass": false,
		"surface steady frame and jitter budgets pass": false,
		"missing baseline environment rejected":        false,
		"impossible performance numbers rejected":      false,
		"unbounded cache rejected":                     false,
		"unsupported Electron speed claim rejected":    false,
	}
	var issues []string
	for _, c := range cases {
		if _, ok := required[c.Name]; ok {
			required[c.Name] = c.Ran && c.Pass
		}
		if strings.TrimSpace(c.Name) == "" || strings.TrimSpace(c.Kind) == "" {
			issues = append(issues, "case report requires name and kind")
		}
		if !c.Ran || !c.Pass {
			issues = append(issues, fmt.Sprintf("case %q must run and pass", c.Name))
		}
	}
	for name, ok := range required {
		if !ok {
			issues = append(issues, fmt.Sprintf("missing or failed case %q", name))
		}
	}
	return issues
}

func validateSafeRelPath(path string) error {
	if path == "" {
		return fmt.Errorf("path is required")
	}
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute path is forbidden")
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") || clean == ".." {
		return fmt.Errorf("path must stay inside report root")
	}
	return nil
}

func validGitHead(value string) bool {
	if len(value) != 40 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
