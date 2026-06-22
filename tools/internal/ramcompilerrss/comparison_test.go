package ramcompilerrss

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestCompareBundlesEvaluatesBaselineCandidateMedianGate(t *testing.T) {
	baseline := t.TempDir()
	candidate := t.TempDir()
	host := hostFingerprint{
		Schema:        "tetra.ram.p7-compiler-rss.host.v1",
		Hostname:      "same-host",
		GOOS:          "linux",
		GOARCH:        "amd64",
		NumCPU:        8,
		KernelRelease: "6.1.0-test",
		OSRelease:     "Test Linux",
		CPUModel:      "Test CPU",
	}
	writeComparisonBundle(t, baseline, host, []scenarioSummary{
		{
			Name:               "large_reports_on_jobs_4_warm",
			ModuleCount:        12,
			Jobs:               4,
			Reports:            true,
			WarmCache:          true,
			SampleCount:        5,
			RSSMedianBytes:     1000,
			RSSDispersionBytes: 40,
			RSSPeakPhase:       "module_codegen",
		},
		{
			Name:               "small_reports_off_jobs_1_cold",
			ModuleCount:        3,
			Jobs:               1,
			SampleCount:        5,
			RSSMedianBytes:     500,
			RSSDispersionBytes: 20,
			RSSPeakPhase:       "semantic_analysis",
		},
	})
	writeComparisonBundle(t, candidate, host, []scenarioSummary{
		{
			Name:               "large_reports_on_jobs_4_warm",
			ModuleCount:        12,
			Jobs:               4,
			Reports:            true,
			WarmCache:          true,
			SampleCount:        5,
			RSSMedianBytes:     1020,
			RSSDispersionBytes: 30,
			RSSPeakPhase:       "report_generation",
		},
		{
			Name:               "small_reports_off_jobs_1_cold",
			ModuleCount:        3,
			Jobs:               1,
			SampleCount:        5,
			RSSMedianBytes:     450,
			RSSDispersionBytes: 10,
			RSSPeakPhase:       "source_loading_parsing",
		},
		{
			Name:               "candidate_only_reports_off_jobs_1_cold",
			ModuleCount:        2,
			Jobs:               1,
			SampleCount:        5,
			RSSMedianBytes:     300,
			RSSDispersionBytes: 5,
		},
	})

	outPath := filepath.Join(t.TempDir(), "baseline-candidate-comparison.json")
	result, err := CompareBundles(CompareOptions{
		BaselineDir:  baseline,
		CandidateDir: candidate,
		OutPath:      outPath,
		MinSamples:   5,
		Command:      []string{"go", "run", "./tools/cmd/ram-p7-compiler-rss", "--compare-baseline-dir", baseline, "--compare-candidate-dir", candidate},
	})
	if err != nil {
		t.Fatalf("CompareBundles: %v", err)
	}
	if result.OverallStatus != "pass" || !result.SameHostSameConfig {
		t.Fatalf("comparison status = %+v, want pass same-host", result)
	}
	if result.OutPath != outPath {
		t.Fatalf("OutPath = %q, want %q", result.OutPath, outPath)
	}
	if len(result.Scenarios) != 2 {
		t.Fatalf("scenario comparisons = %#v, want two baseline scenarios", result.Scenarios)
	}
	large := result.Scenarios[0]
	if large.Name != "large_reports_on_jobs_4_warm" ||
		large.BaselineRSSMedian != 1000 ||
		large.CandidateRSSMedian != 1020 ||
		large.BoundRSSBytes != 1070 ||
		large.DeltaBytes != 20 ||
		large.Ratio != 1.02 ||
		large.EvaluationStatus != "flat" ||
		large.BoundBasis != "baseline_median_plus_observed_baseline_candidate_dispersion" ||
		large.BaselineRSSPeakPhase != "module_codegen" ||
		large.CandidateRSSPeakPhase != "report_generation" {
		t.Fatalf("large comparison = %+v", large)
	}
	small := result.Scenarios[1]
	if small.Name != "small_reports_off_jobs_1_cold" || small.EvaluationStatus != "improved" ||
		small.DeltaBytes != -50 {
		t.Fatalf("small comparison = %+v", small)
	}
	if len(result.CandidateOnlyScenarios) != 1 ||
		result.CandidateOnlyScenarios[0] != "candidate_only_reports_off_jobs_1_cold" {
		t.Fatalf("candidate-only scenarios = %#v", result.CandidateOnlyScenarios)
	}
	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	var persisted baselineCandidateComparison
	if err := json.Unmarshal(raw, &persisted); err != nil {
		t.Fatalf("persisted comparison JSON: %v", err)
	}
	if persisted.Schema != BaselineComparisonSchema || persisted.OverallStatus != "pass" {
		t.Fatalf("persisted comparison = %+v", persisted)
	}
	if len(persisted.Command) == 0 || persisted.Command[0] != "go" {
		t.Fatalf("persisted command = %#v", persisted.Command)
	}
}

func TestCompareBundlesFlagsRegressionAndIncompatibleHost(t *testing.T) {
	host := hostFingerprint{
		Schema:   "tetra.ram.p7-compiler-rss.host.v1",
		Hostname: "same-host",
		GOOS:     "linux",
		GOARCH:   "amd64",
		NumCPU:   8,
	}
	baseline := t.TempDir()
	candidate := t.TempDir()
	writeComparisonBundle(t, baseline, host, []scenarioSummary{
		{
			Name:               "large_reports_on_jobs_4_warm",
			ModuleCount:        12,
			Jobs:               4,
			Reports:            true,
			WarmCache:          true,
			SampleCount:        5,
			RSSMedianBytes:     1000,
			RSSDispersionBytes: 10,
		},
	})
	writeComparisonBundle(t, candidate, host, []scenarioSummary{
		{
			Name:               "large_reports_on_jobs_4_warm",
			ModuleCount:        12,
			Jobs:               4,
			Reports:            true,
			WarmCache:          true,
			SampleCount:        5,
			RSSMedianBytes:     1100,
			RSSDispersionBytes: 10,
		},
	})
	result, err := CompareBundles(CompareOptions{
		BaselineDir:  baseline,
		CandidateDir: candidate,
		OutPath:      filepath.Join(t.TempDir(), "regression.json"),
		MinSamples:   5,
	})
	if err != nil {
		t.Fatalf("CompareBundles regression: %v", err)
	}
	if result.OverallStatus != "fail" ||
		len(result.Scenarios) != 1 ||
		result.Scenarios[0].EvaluationStatus != "regressed" {
		t.Fatalf("regression result = %+v", result)
	}

	incompatible := t.TempDir()
	host.NumCPU = 16
	writeComparisonBundle(t, incompatible, host, []scenarioSummary{
		{
			Name:               "large_reports_on_jobs_4_warm",
			ModuleCount:        12,
			Jobs:               4,
			Reports:            true,
			WarmCache:          true,
			SampleCount:        5,
			RSSMedianBytes:     900,
			RSSDispersionBytes: 10,
		},
	})
	result, err = CompareBundles(CompareOptions{
		BaselineDir:  baseline,
		CandidateDir: incompatible,
		OutPath:      filepath.Join(t.TempDir(), "incompatible.json"),
		MinSamples:   5,
	})
	if err != nil {
		t.Fatalf("CompareBundles incompatible: %v", err)
	}
	if result.OverallStatus != "incompatible" ||
		result.SameHostSameConfig ||
		len(result.Scenarios) != 1 ||
		result.Scenarios[0].EvaluationStatus != "incompatible_host" {
		t.Fatalf("incompatible result = %+v", result)
	}
}

func TestCompareBundlesRecordsPhaseMedianDeltas(t *testing.T) {
	baseline := t.TempDir()
	candidate := t.TempDir()
	host := hostFingerprint{
		Schema:   "tetra.ram.p7-compiler-rss.host.v1",
		Hostname: "same-host",
		GOOS:     "linux",
		GOARCH:   "amd64",
		NumCPU:   8,
	}
	baseScenario := scenarioSummary{
		Name:               "medium_reports_on_jobs_2_cold",
		ModuleCount:        6,
		Jobs:               2,
		Reports:            true,
		SampleCount:        3,
		RSSMedianBytes:     1000,
		RSSDispersionBytes: 20,
	}
	candidateScenario := baseScenario
	candidateScenario.RSSMedianBytes = 1200
	candidateScenario.RSSDispersionBytes = 10
	baseScenario.Samples = writePhaseProfiles(t, baseline, baseScenario.Name, [][]compilerProfilePhase{
		{
			{Name: "source_loading_parsing", RSSCurrentBytes: 900},
			{Name: "semantic_analysis", RSSCurrentBytes: 920},
			{Name: "report_generation", RSSCurrentBytes: 950},
		},
		{
			{Name: "source_loading_parsing", RSSCurrentBytes: 910},
			{Name: "semantic_analysis", RSSCurrentBytes: 930},
			{Name: "report_generation", RSSCurrentBytes: 960},
		},
		{
			{Name: "source_loading_parsing", RSSCurrentBytes: 920},
			{Name: "semantic_analysis", RSSCurrentBytes: 940},
			{Name: "report_generation", RSSCurrentBytes: 970},
		},
	})
	candidateScenario.Samples = writePhaseProfiles(t, candidate, candidateScenario.Name, [][]compilerProfilePhase{
		{
			{Name: "source_loading_parsing", RSSCurrentBytes: 1100},
			{Name: "semantic_analysis", RSSCurrentBytes: 1200},
			{Name: "report_generation", RSSCurrentBytes: 1300},
		},
		{
			{Name: "source_loading_parsing", RSSCurrentBytes: 1110},
			{Name: "semantic_analysis", RSSCurrentBytes: 1210},
			{Name: "report_generation", RSSCurrentBytes: 1310},
		},
		{
			{Name: "source_loading_parsing", RSSCurrentBytes: 1120},
			{Name: "semantic_analysis", RSSCurrentBytes: 1220},
			{Name: "report_generation", RSSCurrentBytes: 1320},
		},
	})
	writeComparisonBundle(t, baseline, host, []scenarioSummary{baseScenario})
	writeComparisonBundle(t, candidate, host, []scenarioSummary{candidateScenario})

	result, err := CompareBundles(CompareOptions{
		BaselineDir:  baseline,
		CandidateDir: candidate,
		OutPath:      filepath.Join(t.TempDir(), "comparison.json"),
		MinSamples:   3,
	})
	if err != nil {
		t.Fatalf("CompareBundles: %v", err)
	}
	if len(result.Scenarios) != 1 {
		t.Fatalf("scenarios = %#v", result.Scenarios)
	}
	phases := result.Scenarios[0].PhaseComparisons
	if len(phases) != 3 {
		t.Fatalf("phase comparisons = %#v, want three phases", phases)
	}
	top := phases[0]
	if top.Name != "report_generation" ||
		top.BaselineRSSMedian != 960 ||
		top.CandidateRSSMedian != 1310 ||
		top.DeltaBytes != 350 ||
		top.Ratio != 1.3646 ||
		top.BaselineSampleCount != 3 ||
		top.CandidateSampleCount != 3 {
		t.Fatalf("top phase comparison = %+v", top)
	}
}

func writeComparisonBundle(t *testing.T, dir string, host hostFingerprint, scenarios []scenarioSummary) {
	t.Helper()
	if err := writeJSON(filepath.Join(dir, "host-fingerprint.json"), host); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(filepath.Join(dir, "compiler-rss-manifest.json"), bundleManifest{
		Schema:          Schema,
		GitHead:         "0123456789abcdef0123456789abcdef01234567",
		ScenarioSummary: "scenario-summary.json",
		HostFingerprint: "host-fingerprint.json",
	}); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(filepath.Join(dir, "scenario-summary.json"), summaryReport{
		Schema:    SummarySchema,
		Scenarios: scenarios,
	}); err != nil {
		t.Fatal(err)
	}
}

func writePhaseProfiles(t *testing.T, dir string, scenarioName string, samples [][]compilerProfilePhase) []sampleSummary {
	t.Helper()
	var out []sampleSummary
	for i, phases := range samples {
		rel := filepath.ToSlash(filepath.Join(
			"scenarios",
			scenarioName,
			"samples",
			fmt.Sprintf("sample-%02d", i+1),
			"compiler-profile.json",
		))
		if err := writeJSON(filepath.Join(dir, filepath.FromSlash(rel)), compilerPhaseProfile{
			Schema:       "tetra.compiler.phase-profile.v1",
			RSSSupported: true,
			RSSPeakBytes: 1,
			WorkerCount:  1,
			WorkerReason: "test",
			Phases:       phases,
		}); err != nil {
			t.Fatal(err)
		}
		out = append(out, sampleSummary{
			Index:        i + 1,
			PhaseProfile: rel,
		})
	}
	return out
}
