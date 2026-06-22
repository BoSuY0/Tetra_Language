package ramcompilerrss

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const BaselineComparisonSchema = "tetra.ram.p7-compiler-rss-baseline-comparison.v1"

type CompareOptions struct {
	BaselineDir  string
	CandidateDir string
	OutPath      string
	MinSamples   int
	Command      []string
}

type baselineCandidateComparison struct {
	Schema                 string                                `json:"schema"`
	BaselineDir            string                                `json:"baseline_dir"`
	CandidateDir           string                                `json:"candidate_dir"`
	OutPath                string                                `json:"-"`
	BaselineGitHead        string                                `json:"baseline_git_head,omitempty"`
	CandidateGitHead       string                                `json:"candidate_git_head,omitempty"`
	Command                []string                              `json:"command,omitempty"`
	SameHostSameConfig     bool                                  `json:"same_host_same_config"`
	HostMismatches         []string                              `json:"host_mismatches,omitempty"`
	MinSamples             int                                   `json:"min_samples"`
	OverallStatus          string                                `json:"overall_status"`
	BoundBasis             string                                `json:"bound_basis"`
	Scenarios              []baselineCandidateScenarioComparison `json:"scenarios"`
	CandidateOnlyScenarios []string                              `json:"candidate_only_scenarios,omitempty"`
}

type baselineCandidateScenarioComparison struct {
	Name                  string                             `json:"name"`
	ModuleCount           int                                `json:"module_count"`
	Jobs                  int                                `json:"jobs"`
	Reports               bool                               `json:"reports"`
	WarmCache             bool                               `json:"warm_cache"`
	MemoryBudgetBytes     int64                              `json:"memory_budget_bytes,omitempty"`
	ExpectCompileError    bool                               `json:"expect_compile_error,omitempty"`
	BaselineSampleCount   int                                `json:"baseline_sample_count"`
	CandidateSampleCount  int                                `json:"candidate_sample_count"`
	BaselineRSSMedian     uint64                             `json:"baseline_rss_median_bytes"`
	CandidateRSSMedian    uint64                             `json:"candidate_rss_median_bytes"`
	BaselineDispersion    uint64                             `json:"baseline_rss_dispersion_bytes"`
	CandidateDispersion   uint64                             `json:"candidate_rss_dispersion_bytes"`
	BoundRSSBytes         uint64                             `json:"bound_rss_bytes"`
	DeltaBytes            int64                              `json:"delta_bytes"`
	Ratio                 float64                            `json:"ratio"`
	EvaluationStatus      string                             `json:"evaluation_status"`
	BoundBasis            string                             `json:"bound_basis"`
	BaselineRSSPeakPhase  string                             `json:"baseline_rss_peak_phase,omitempty"`
	CandidateRSSPeakPhase string                             `json:"candidate_rss_peak_phase,omitempty"`
	PhaseComparisons      []baselineCandidatePhaseComparison `json:"phase_comparisons,omitempty"`
}

type baselineCandidatePhaseComparison struct {
	Name                 string  `json:"name"`
	BaselineSampleCount  int     `json:"baseline_sample_count"`
	CandidateSampleCount int     `json:"candidate_sample_count"`
	BaselineRSSMedian    uint64  `json:"baseline_rss_median_bytes"`
	CandidateRSSMedian   uint64  `json:"candidate_rss_median_bytes"`
	DeltaBytes           int64   `json:"delta_bytes"`
	Ratio                float64 `json:"ratio"`
}

type comparisonBundle struct {
	dir      string
	manifest bundleManifest
	host     hostFingerprint
	summary  summaryReport
}

func CompareBundles(opts CompareOptions) (baselineCandidateComparison, error) {
	if strings.TrimSpace(opts.BaselineDir) == "" {
		return baselineCandidateComparison{}, fmt.Errorf("baseline dir is required")
	}
	if strings.TrimSpace(opts.CandidateDir) == "" {
		return baselineCandidateComparison{}, fmt.Errorf("candidate dir is required")
	}
	if strings.TrimSpace(opts.OutPath) == "" {
		return baselineCandidateComparison{}, fmt.Errorf("comparison out path is required")
	}
	minSamples := opts.MinSamples
	if minSamples <= 0 {
		minSamples = 5
	}
	baseline, err := loadComparisonBundle(opts.BaselineDir)
	if err != nil {
		return baselineCandidateComparison{}, fmt.Errorf("load baseline bundle: %w", err)
	}
	candidate, err := loadComparisonBundle(opts.CandidateDir)
	if err != nil {
		return baselineCandidateComparison{}, fmt.Errorf("load candidate bundle: %w", err)
	}

	hostMismatches := compareHostFingerprint(baseline.host, candidate.host)
	sameHost := len(hostMismatches) == 0
	result := baselineCandidateComparison{
		Schema:             BaselineComparisonSchema,
		BaselineDir:        opts.BaselineDir,
		CandidateDir:       opts.CandidateDir,
		OutPath:            opts.OutPath,
		BaselineGitHead:    baseline.manifest.GitHead,
		CandidateGitHead:   candidate.manifest.GitHead,
		Command:            append([]string(nil), opts.Command...),
		SameHostSameConfig: sameHost,
		HostMismatches:     hostMismatches,
		MinSamples:         minSamples,
		BoundBasis:         "baseline_median_plus_observed_baseline_candidate_dispersion",
	}

	baselineByName := scenarioSummaryByName(baseline.summary.Scenarios)
	candidateByName := scenarioSummaryByName(candidate.summary.Scenarios)
	var names []string
	for name := range baselineByName {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		base := baselineByName[name]
		cand, ok := candidateByName[name]
		if !ok {
			result.Scenarios = append(result.Scenarios, missingCandidateComparison(base, minSamples, result.BoundBasis))
			continue
		}
		comparison, err := compareScenario(baseline.dir, candidate.dir, base, cand, minSamples, sameHost, result.BoundBasis)
		if err != nil {
			return baselineCandidateComparison{}, err
		}
		result.Scenarios = append(result.Scenarios, comparison)
	}

	for name := range candidateByName {
		if _, ok := baselineByName[name]; !ok {
			result.CandidateOnlyScenarios = append(result.CandidateOnlyScenarios, name)
		}
	}
	sort.Strings(result.CandidateOnlyScenarios)
	result.OverallStatus = baselineCandidateOverallStatus(result.Scenarios, sameHost)
	if err := writeJSON(opts.OutPath, result); err != nil {
		return baselineCandidateComparison{}, err
	}
	return result, nil
}

func loadComparisonBundle(dir string) (comparisonBundle, error) {
	var manifest bundleManifest
	if err := readJSONFile(filepath.Join(dir, "compiler-rss-manifest.json"), &manifest); err != nil {
		return comparisonBundle{}, err
	}
	if manifest.Schema != Schema {
		return comparisonBundle{}, fmt.Errorf("manifest schema = %q, want %s", manifest.Schema, Schema)
	}
	hostPath := manifest.HostFingerprint
	if strings.TrimSpace(hostPath) == "" {
		hostPath = "host-fingerprint.json"
	}
	var host hostFingerprint
	if err := readJSONFile(filepath.Join(dir, hostPath), &host); err != nil {
		return comparisonBundle{}, err
	}
	summaryPath := manifest.ScenarioSummary
	if strings.TrimSpace(summaryPath) == "" {
		summaryPath = "scenario-summary.json"
	}
	var summary summaryReport
	if err := readJSONFile(filepath.Join(dir, summaryPath), &summary); err != nil {
		return comparisonBundle{}, err
	}
	if summary.Schema != SummarySchema {
		return comparisonBundle{}, fmt.Errorf("summary schema = %q, want %s", summary.Schema, SummarySchema)
	}
	return comparisonBundle{dir: dir, manifest: manifest, host: host, summary: summary}, nil
}

func readJSONFile(path string, out any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func scenarioSummaryByName(scenarios []scenarioSummary) map[string]scenarioSummary {
	out := make(map[string]scenarioSummary, len(scenarios))
	for _, scenario := range scenarios {
		out[scenario.Name] = scenario
	}
	return out
}

func compareHostFingerprint(baseline hostFingerprint, candidate hostFingerprint) []string {
	checks := []struct {
		name string
		a    any
		b    any
	}{
		{name: "hostname", a: baseline.Hostname, b: candidate.Hostname},
		{name: "goos", a: baseline.GOOS, b: candidate.GOOS},
		{name: "goarch", a: baseline.GOARCH, b: candidate.GOARCH},
		{name: "num_cpu", a: baseline.NumCPU, b: candidate.NumCPU},
		{name: "kernel_release", a: baseline.KernelRelease, b: candidate.KernelRelease},
		{name: "os_release", a: baseline.OSRelease, b: candidate.OSRelease},
		{name: "cpu_model", a: baseline.CPUModel, b: candidate.CPUModel},
	}
	var mismatches []string
	for _, check := range checks {
		if check.a != check.b {
			mismatches = append(mismatches, check.name)
		}
	}
	return mismatches
}

func missingCandidateComparison(baseline scenarioSummary, minSamples int, boundBasis string) baselineCandidateScenarioComparison {
	return baselineCandidateScenarioComparison{
		Name:                 baseline.Name,
		ModuleCount:          baseline.ModuleCount,
		Jobs:                 baseline.Jobs,
		Reports:              baseline.Reports,
		WarmCache:            baseline.WarmCache,
		MemoryBudgetBytes:    baseline.MemoryBudgetBytes,
		ExpectCompileError:   baseline.ExpectCompileError,
		BaselineSampleCount:  baseline.SampleCount,
		BaselineRSSMedian:    baseline.RSSMedianBytes,
		BaselineDispersion:   baseline.RSSDispersionBytes,
		EvaluationStatus:     "missing_candidate",
		BoundBasis:           boundBasis,
		BaselineRSSPeakPhase: baseline.RSSPeakPhase,
	}
}

func compareScenario(baselineDir string, candidateDir string, baseline scenarioSummary, candidate scenarioSummary, minSamples int, sameHost bool, boundBasis string) (baselineCandidateScenarioComparison, error) {
	bound := baseline.RSSMedianBytes + baseline.RSSDispersionBytes + candidate.RSSDispersionBytes
	status := "flat"
	if !sameHost {
		status = "incompatible_host"
	} else if !sameScenarioConfig(baseline, candidate) {
		status = "config_mismatch"
	} else if baseline.SampleCount < minSamples || candidate.SampleCount < minSamples {
		status = "insufficient_samples"
	} else if candidate.RSSMedianBytes < baseline.RSSMedianBytes {
		status = "improved"
	} else if candidate.RSSMedianBytes > bound {
		status = "regressed"
	}
	phases, err := compareScenarioPhases(baselineDir, candidateDir, baseline, candidate)
	if err != nil {
		return baselineCandidateScenarioComparison{}, err
	}
	return baselineCandidateScenarioComparison{
		Name:                  baseline.Name,
		ModuleCount:           baseline.ModuleCount,
		Jobs:                  baseline.Jobs,
		Reports:               baseline.Reports,
		WarmCache:             baseline.WarmCache,
		MemoryBudgetBytes:     baseline.MemoryBudgetBytes,
		ExpectCompileError:    baseline.ExpectCompileError,
		BaselineSampleCount:   baseline.SampleCount,
		CandidateSampleCount:  candidate.SampleCount,
		BaselineRSSMedian:     baseline.RSSMedianBytes,
		CandidateRSSMedian:    candidate.RSSMedianBytes,
		BaselineDispersion:    baseline.RSSDispersionBytes,
		CandidateDispersion:   candidate.RSSDispersionBytes,
		BoundRSSBytes:         bound,
		DeltaBytes:            int64(candidate.RSSMedianBytes) - int64(baseline.RSSMedianBytes),
		Ratio:                 roundedRatio(candidate.RSSMedianBytes, baseline.RSSMedianBytes),
		EvaluationStatus:      status,
		BoundBasis:            boundBasis,
		BaselineRSSPeakPhase:  baseline.RSSPeakPhase,
		CandidateRSSPeakPhase: candidate.RSSPeakPhase,
		PhaseComparisons:      phases,
	}, nil
}

func sameScenarioConfig(baseline scenarioSummary, candidate scenarioSummary) bool {
	return baseline.ModuleCount == candidate.ModuleCount &&
		baseline.Jobs == candidate.Jobs &&
		baseline.Reports == candidate.Reports &&
		baseline.WarmCache == candidate.WarmCache &&
		baseline.MemoryBudgetBytes == candidate.MemoryBudgetBytes &&
		baseline.ExpectCompileError == candidate.ExpectCompileError
}

type phaseMedian struct {
	median      uint64
	sampleCount int
}

func compareScenarioPhases(baselineDir string, candidateDir string, baseline scenarioSummary, candidate scenarioSummary) ([]baselineCandidatePhaseComparison, error) {
	baselinePhases, err := scenarioPhaseMedians(baselineDir, baseline)
	if err != nil {
		return nil, fmt.Errorf("%s baseline phase profiles: %w", baseline.Name, err)
	}
	candidatePhases, err := scenarioPhaseMedians(candidateDir, candidate)
	if err != nil {
		return nil, fmt.Errorf("%s candidate phase profiles: %w", candidate.Name, err)
	}
	var out []baselineCandidatePhaseComparison
	for name, base := range baselinePhases {
		cand, ok := candidatePhases[name]
		if !ok {
			continue
		}
		out = append(out, baselineCandidatePhaseComparison{
			Name:                 name,
			BaselineSampleCount:  base.sampleCount,
			CandidateSampleCount: cand.sampleCount,
			BaselineRSSMedian:    base.median,
			CandidateRSSMedian:   cand.median,
			DeltaBytes:           int64(cand.median) - int64(base.median),
			Ratio:                roundedRatio(cand.median, base.median),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DeltaBytes != out[j].DeltaBytes {
			return out[i].DeltaBytes > out[j].DeltaBytes
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func scenarioPhaseMedians(bundleDir string, scenario scenarioSummary) (map[string]phaseMedian, error) {
	values := map[string][]uint64{}
	for _, sample := range scenario.Samples {
		if strings.TrimSpace(sample.PhaseProfile) == "" {
			continue
		}
		path, err := safeBundleRelPath(bundleDir, sample.PhaseProfile)
		if err != nil {
			return nil, err
		}
		profile, err := readProfile(path, true)
		if err != nil {
			return nil, err
		}
		for _, phase := range profile.Phases {
			if strings.TrimSpace(phase.Name) == "" || phase.RSSCurrentBytes == 0 {
				continue
			}
			values[phase.Name] = append(values[phase.Name], phase.RSSCurrentBytes)
		}
	}
	out := make(map[string]phaseMedian, len(values))
	for name, phaseValues := range values {
		sort.Slice(phaseValues, func(i, j int) bool { return phaseValues[i] < phaseValues[j] })
		out[name] = phaseMedian{
			median:      medianUint64(phaseValues),
			sampleCount: len(phaseValues),
		}
	}
	return out, nil
}

func safeBundleRelPath(root string, rel string) (string, error) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", fmt.Errorf("empty bundle-relative path")
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if filepath.IsAbs(clean) || clean == "." || clean == ".." ||
		strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe bundle-relative path %q", rel)
	}
	return filepath.Join(root, clean), nil
}

func baselineCandidateOverallStatus(scenarios []baselineCandidateScenarioComparison, sameHost bool) string {
	if !sameHost {
		return "incompatible"
	}
	if len(scenarios) == 0 {
		return "fail"
	}
	status := "pass"
	for _, scenario := range scenarios {
		switch scenario.EvaluationStatus {
		case "improved", "flat":
		case "insufficient_samples":
			if status == "pass" {
				status = "insufficient_samples"
			}
		case "config_mismatch", "missing_candidate", "incompatible_host":
			return "incompatible"
		default:
			return "fail"
		}
	}
	return status
}
