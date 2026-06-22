package ramcompilerrss

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunWritesCompilerRSSBundle(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("compiler RSS bundle uses linux-x64 native compiler evidence")
	}

	outDir := t.TempDir()
	memoryResetCount := 0
	result, err := Run(Options{
		OutDir:         outDir,
		GitHead:        "0123456789abcdef0123456789abcdef01234567",
		GitStatusShort: "",
		Command:        []string{"ram-p7-compiler-rss", "--test"},
		Now:            func() time.Time { return time.Unix(1700000000, 0).UTC() },
		MemoryReset:    func() { memoryResetCount++ },
		Samples:        2,
		Scenarios: []Scenario{
			{
				Name:        "small_reports_off_jobs_1_cold",
				ModuleCount: 3,
				Jobs:        1,
				WarmCache:   false,
			},
			{
				Name:        "small_reports_on_jobs_1_cold",
				ModuleCount: 3,
				Jobs:        1,
				Reports:     true,
				WarmCache:   false,
			},
			{
				Name:               "compile_error_reports_off_jobs_1_cold",
				ModuleCount:        2,
				Jobs:               1,
				ExpectCompileError: true,
			},
			{
				Name:               "compile_error_reports_on_jobs_2_warm",
				ModuleCount:        2,
				Jobs:               2,
				Reports:            true,
				WarmCache:          true,
				ExpectCompileError: true,
			},
		},
	})
	if err != nil {
		t.Fatalf("Run compiler RSS harness: %v", err)
	}
	if memoryResetCount != 8 {
		t.Fatalf("memory reset count = %d, want one reset per measured sample", memoryResetCount)
	}
	if result.OutDir != outDir {
		t.Fatalf("OutDir = %q, want %q", result.OutDir, outDir)
	}

	required := []string{
		result.ManifestPath,
		filepath.Join(outDir, "scenario-summary.json"),
		filepath.Join(outDir, "host-fingerprint.json"),
		filepath.Join(outDir, "target-scope.json"),
		filepath.Join(outDir, "command-manifest.json"),
		filepath.Join(outDir, "validator-output.txt"),
		filepath.Join(outDir, "scenarios", "small_reports_on_jobs_1_cold", "samples", "sample-01", "compiler-profile.json"),
		filepath.Join(outDir, "scenarios", "small_reports_on_jobs_1_cold", "samples", "sample-01", "out", "app"),
		filepath.Join(outDir, "scenarios", "compile_error_reports_off_jobs_1_cold", "samples", "sample-01", "compiler-profile.json"),
		filepath.Join(outDir, "scenarios", "compile_error_reports_on_jobs_2_warm", "samples", "sample-01", "compiler-profile.json"),
		filepath.Join(outDir, "scenarios", "compile_error_reports_on_jobs_2_warm", "samples", "sample-01", "out", "warmup-app"),
	}
	for _, path := range required {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("missing compiler RSS artifact %s: %v", path, err)
		}
		if info.Size() == 0 {
			t.Fatalf("empty compiler RSS artifact %s", path)
		}
	}

	rawManifest, err := os.ReadFile(result.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		t.Fatalf("manifest JSON: %v", err)
	}
	if manifest["schema"] != Schema {
		t.Fatalf("manifest schema = %v, want %s", manifest["schema"], Schema)
	}
	if manifest["git_head"] != "0123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("manifest git_head = %v", manifest["git_head"])
	}
	if manifest["scenario_summary"] != "scenario-summary.json" {
		t.Fatalf("manifest scenario_summary = %v", manifest["scenario_summary"])
	}
	if manifest["target_scope"] != "target-scope.json" {
		t.Fatalf("manifest target_scope = %v", manifest["target_scope"])
	}
	if manifest["artifact_hash_manifest"] != "artifact-hashes.json" {
		t.Fatalf("manifest artifact_hash_manifest = %v", manifest["artifact_hash_manifest"])
	}

	rawTargetScope, err := os.ReadFile(filepath.Join(outDir, "target-scope.json"))
	if err != nil {
		t.Fatal(err)
	}
	var targetScope struct {
		Schema           string `json:"schema"`
		HostTarget       string `json:"host_target"`
		CompilerTarget   string `json:"compiler_target"`
		SupportedTargets []struct {
			Target string `json:"target"`
			Status string `json:"status"`
			Reason string `json:"reason"`
		} `json:"supported_targets"`
		NonClaimTargets []struct {
			Target string `json:"target"`
			Status string `json:"status"`
			Reason string `json:"reason"`
		} `json:"non_claim_targets"`
	}
	if err := json.Unmarshal(rawTargetScope, &targetScope); err != nil {
		t.Fatalf("target scope JSON: %v", err)
	}
	if targetScope.Schema != "tetra.ram.p7-compiler-rss-target-scope.v1" ||
		targetScope.HostTarget != "linux/amd64" ||
		targetScope.CompilerTarget != "linux-x64" {
		t.Fatalf("target scope identity = %+v", targetScope)
	}
	if len(targetScope.SupportedTargets) != 1 ||
		targetScope.SupportedTargets[0].Target != "linux-x64" ||
		targetScope.SupportedTargets[0].Status != "host_rss_measured" ||
		!strings.Contains(targetScope.SupportedTargets[0].Reason, "process RSS") {
		t.Fatalf("supported target scope = %+v", targetScope.SupportedTargets)
	}
	wantNonClaims := []string{"windows-x64", "macos-x64", "macos-arm64", "linux-x86", "linux-x32", "wasm32-wasi", "wasm32-web"}
	if len(targetScope.NonClaimTargets) != len(wantNonClaims) {
		t.Fatalf("non-claim targets = %+v", targetScope.NonClaimTargets)
	}
	for i, want := range wantNonClaims {
		got := targetScope.NonClaimTargets[i]
		if got.Target != want || got.Status != "non_claim" || !strings.Contains(got.Reason, "not measured") {
			t.Fatalf("non-claim target %d = %+v, want %s non_claim not measured", i, got, want)
		}
	}

	rawSummary, err := os.ReadFile(filepath.Join(outDir, "scenario-summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	var summary struct {
		Schema            string `json:"schema"`
		ReportComparisons []struct {
			Name                  string  `json:"name"`
			ReportOffScenario     string  `json:"report_off_scenario"`
			ReportOnScenario      string  `json:"report_on_scenario"`
			ReportOffRSSMedian    uint64  `json:"report_off_rss_median_bytes"`
			ReportOnRSSMedian     uint64  `json:"report_on_rss_median_bytes"`
			BoundRSSBytes         uint64  `json:"bound_rss_bytes"`
			DeltaBytes            int64   `json:"delta_bytes"`
			Ratio                 float64 `json:"ratio"`
			EvaluationStatus      string  `json:"evaluation_status"`
			BoundBasis            string  `json:"bound_basis"`
			SameHostSameConfig    bool    `json:"same_host_same_config"`
			ReportOffSampleCount  int     `json:"report_off_sample_count"`
			ReportOnSampleCount   int     `json:"report_on_sample_count"`
			ReportOffDispersion   uint64  `json:"report_off_rss_dispersion_bytes"`
			ReportOnDispersion    uint64  `json:"report_on_rss_dispersion_bytes"`
			ReportOffRSSPeakPhase string  `json:"report_off_rss_peak_phase"`
			ReportOnRSSPeakPhase  string  `json:"report_on_rss_peak_phase"`
		} `json:"report_comparisons"`
		Scenarios []struct {
			Name                  string `json:"name"`
			PhaseProfile          string `json:"phase_profile"`
			SampleCount           int    `json:"sample_count"`
			RSSPeakBytes          uint64 `json:"rss_peak_bytes"`
			RSSMedianBytes        uint64 `json:"rss_median_bytes"`
			RSSMinBytes           uint64 `json:"rss_min_bytes"`
			RSSMaxBytes           uint64 `json:"rss_max_bytes"`
			RSSDispersionBytes    uint64 `json:"rss_dispersion_bytes"`
			WorkerCount           int    `json:"worker_count"`
			WorkerReason          string `json:"worker_reason"`
			ExecutableSHA256      string `json:"executable_sha256"`
			Reports               bool   `json:"reports"`
			WarmCache             bool   `json:"warm_cache"`
			ExpectCompileError    bool   `json:"expect_compile_error"`
			CompileErrorObserved  bool   `json:"compile_error_observed"`
			CompileError          string `json:"compile_error,omitempty"`
			CompileErrorSampleNum int    `json:"compile_error_sample_num,omitempty"`
			Samples               []struct {
				Index                int    `json:"index"`
				PhaseProfile         string `json:"phase_profile"`
				RSSPeakBytes         uint64 `json:"rss_peak_bytes"`
				RSSPeakPhase         string `json:"rss_peak_phase"`
				ExecutableSHA256     string `json:"executable_sha256,omitempty"`
				CompileErrorObserved bool   `json:"compile_error_observed,omitempty"`
				CompileError         string `json:"compile_error,omitempty"`
			} `json:"samples"`
		} `json:"scenarios"`
	}
	if err := json.Unmarshal(rawSummary, &summary); err != nil {
		t.Fatalf("summary JSON: %v", err)
	}
	if summary.Schema != SummarySchema || len(summary.Scenarios) != 4 {
		t.Fatalf("summary identity = %#v", summary)
	}
	if len(summary.ReportComparisons) != 1 {
		t.Fatalf("report comparisons = %#v, want one report-on/off pair", summary.ReportComparisons)
	}
	comparison := summary.ReportComparisons[0]
	if comparison.Name != "small_reports_jobs_1_cold" ||
		comparison.ReportOffScenario != "small_reports_off_jobs_1_cold" ||
		comparison.ReportOnScenario != "small_reports_on_jobs_1_cold" ||
		comparison.ReportOffRSSMedian == 0 ||
		comparison.ReportOnRSSMedian == 0 ||
		comparison.BoundRSSBytes == 0 ||
		comparison.Ratio <= 0 ||
		comparison.BoundBasis == "" ||
		!comparison.SameHostSameConfig ||
		comparison.ReportOffSampleCount != 2 ||
		comparison.ReportOnSampleCount != 2 ||
		comparison.ReportOffRSSPeakPhase == "" ||
		comparison.ReportOnRSSPeakPhase == "" ||
		(comparison.EvaluationStatus != "pass" && comparison.EvaluationStatus != "fail") {
		t.Fatalf("unexpected report comparison: %+v", comparison)
	}
	byName := make(map[string]struct {
		Name                  string `json:"name"`
		PhaseProfile          string `json:"phase_profile"`
		SampleCount           int    `json:"sample_count"`
		RSSPeakBytes          uint64 `json:"rss_peak_bytes"`
		RSSMedianBytes        uint64 `json:"rss_median_bytes"`
		RSSMinBytes           uint64 `json:"rss_min_bytes"`
		RSSMaxBytes           uint64 `json:"rss_max_bytes"`
		RSSDispersionBytes    uint64 `json:"rss_dispersion_bytes"`
		WorkerCount           int    `json:"worker_count"`
		WorkerReason          string `json:"worker_reason"`
		ExecutableSHA256      string `json:"executable_sha256"`
		Reports               bool   `json:"reports"`
		WarmCache             bool   `json:"warm_cache"`
		ExpectCompileError    bool   `json:"expect_compile_error"`
		CompileErrorObserved  bool   `json:"compile_error_observed"`
		CompileError          string `json:"compile_error,omitempty"`
		CompileErrorSampleNum int    `json:"compile_error_sample_num,omitempty"`
		Samples               []struct {
			Index                int    `json:"index"`
			PhaseProfile         string `json:"phase_profile"`
			RSSPeakBytes         uint64 `json:"rss_peak_bytes"`
			RSSPeakPhase         string `json:"rss_peak_phase"`
			ExecutableSHA256     string `json:"executable_sha256,omitempty"`
			CompileErrorObserved bool   `json:"compile_error_observed,omitempty"`
			CompileError         string `json:"compile_error,omitempty"`
		} `json:"samples"`
	})
	for _, scenario := range summary.Scenarios {
		byName[scenario.Name] = scenario
	}

	success := byName["small_reports_on_jobs_1_cold"]
	if success.Name == "" {
		t.Fatalf("missing success scenario in summary: %#v", summary.Scenarios)
	}
	if success.SampleCount != 2 ||
		len(success.Samples) != 2 ||
		success.PhaseProfile != "scenarios/small_reports_on_jobs_1_cold/samples/sample-01/compiler-profile.json" ||
		success.RSSPeakBytes == 0 ||
		success.RSSMedianBytes == 0 ||
		success.RSSMinBytes == 0 ||
		success.RSSMaxBytes < success.RSSMinBytes ||
		success.RSSDispersionBytes != success.RSSMaxBytes-success.RSSMinBytes ||
		success.WorkerCount != 1 ||
		strings.TrimSpace(success.WorkerReason) == "" ||
		len(success.ExecutableSHA256) != 64 ||
		success.ExpectCompileError ||
		success.CompileErrorObserved {
		t.Fatalf("unexpected success scenario summary: %+v", success)
	}
	for i, sample := range success.Samples {
		if sample.Index != i+1 ||
			!strings.Contains(sample.PhaseProfile, "samples/sample-") ||
			sample.RSSPeakBytes == 0 ||
			strings.TrimSpace(sample.RSSPeakPhase) == "" ||
			len(sample.ExecutableSHA256) != 64 ||
			sample.CompileErrorObserved ||
			sample.CompileError != "" {
			t.Fatalf("unexpected success sample %d: %+v", i, sample)
		}
	}

	failure := byName["compile_error_reports_off_jobs_1_cold"]
	if failure.Name == "" {
		t.Fatalf("missing compile-error scenario in summary: %#v", summary.Scenarios)
	}
	if failure.SampleCount != 2 ||
		len(failure.Samples) != 2 ||
		!failure.ExpectCompileError ||
		!failure.CompileErrorObserved ||
		strings.TrimSpace(failure.CompileError) == "" ||
		failure.CompileErrorSampleNum == 0 ||
		failure.RSSMedianBytes == 0 ||
		failure.ExecutableSHA256 != "" {
		t.Fatalf("unexpected compile-error scenario summary: %+v", failure)
	}
	for i, sample := range failure.Samples {
		if sample.Index != i+1 ||
			sample.RSSPeakBytes == 0 ||
			strings.TrimSpace(sample.PhaseProfile) == "" ||
			!sample.CompileErrorObserved ||
			strings.TrimSpace(sample.CompileError) == "" ||
			sample.ExecutableSHA256 != "" {
			t.Fatalf("unexpected compile-error sample %d: %+v", i, sample)
		}
	}

	warmFailure := byName["compile_error_reports_on_jobs_2_warm"]
	if warmFailure.Name == "" {
		t.Fatalf("missing warm compile-error scenario in summary: %#v", summary.Scenarios)
	}
	if warmFailure.SampleCount != 2 ||
		!warmFailure.Reports ||
		!warmFailure.WarmCache ||
		!warmFailure.ExpectCompileError ||
		!warmFailure.CompileErrorObserved ||
		warmFailure.ExecutableSHA256 != "" {
		t.Fatalf("unexpected warm compile-error scenario summary: %+v", warmFailure)
	}

	rawValidator, err := os.ReadFile(result.ValidatorOutputPath)
	if err != nil {
		t.Fatal(err)
	}
	validatorText := string(rawValidator)
	if !strings.Contains(validatorText, "result: pass") {
		t.Fatalf("validator output = %q, want pass", rawValidator)
	}
	if !strings.Contains(validatorText, TargetScopeSchema) {
		t.Fatalf("validator output = %q, want target-scope schema", rawValidator)
	}
}

func TestValidateBundleOutputRejectsTargetScopeClaimLeakage(t *testing.T) {
	summaries := []scenarioSummary{validScenarioSummaryForValidatorTest()}
	targetScope := validTargetScopeForValidatorTest()
	if got := validateBundleOutput(summaries, targetScope); !strings.Contains(got, "result: pass") {
		t.Fatalf("valid bundle validator output = %q, want pass", got)
	}

	missingNonClaim := validTargetScopeForValidatorTest()
	missingNonClaim.NonClaimTargets = missingNonClaim.NonClaimTargets[:len(missingNonClaim.NonClaimTargets)-1]
	if got := validateBundleOutput(summaries, missingNonClaim); !strings.Contains(got, "missing non_claim target wasm32-web") {
		t.Fatalf("validator output = %q, want missing wasm32-web non-claim", got)
	}

	crossTargetClaim := validTargetScopeForValidatorTest()
	crossTargetClaim.SupportedTargets = append(crossTargetClaim.SupportedTargets, targetScopeTarget{
		Target: "windows-x64",
		Status: "host_rss_measured",
		Reason: "process RSS measured",
	})
	if got := validateBundleOutput(summaries, crossTargetClaim); !strings.Contains(got, "unsupported host RSS claim for windows-x64") {
		t.Fatalf("validator output = %q, want Windows host RSS claim rejection", got)
	}

	wrongHost := validTargetScopeForValidatorTest()
	wrongHost.HostTarget = "darwin/amd64"
	if got := validateBundleOutput(summaries, wrongHost); !strings.Contains(got, "host_rss_measured requires linux/amd64 host") {
		t.Fatalf("validator output = %q, want non-linux host rejection", got)
	}

	emptyReason := validTargetScopeForValidatorTest()
	emptyReason.NonClaimTargets[0].Reason = ""
	if got := validateBundleOutput(summaries, emptyReason); !strings.Contains(got, "windows-x64 non_claim missing reason") {
		t.Fatalf("validator output = %q, want missing non-claim reason rejection", got)
	}
}

func validScenarioSummaryForValidatorTest() scenarioSummary {
	return scenarioSummary{
		Name:               "small_reports_off_jobs_1_cold",
		ModuleCount:        3,
		Jobs:               1,
		SampleCount:        1,
		PhaseProfile:       "scenarios/small_reports_off_jobs_1_cold/samples/sample-01/compiler-profile.json",
		Executable:         "scenarios/small_reports_off_jobs_1_cold/samples/sample-01/out/app",
		ExecutableSHA256:   strings.Repeat("a", 64),
		RSSPeakBytes:       1024,
		RSSMedianBytes:     1024,
		RSSMinBytes:        1024,
		RSSMaxBytes:        1024,
		RSSDispersionBytes: 0,
		WorkerCount:        1,
		WorkerReason:       "single job requested",
		Samples: []sampleSummary{
			{
				Index:            1,
				PhaseProfile:     "scenarios/small_reports_off_jobs_1_cold/samples/sample-01/compiler-profile.json",
				Executable:       "scenarios/small_reports_off_jobs_1_cold/samples/sample-01/out/app",
				ExecutableSHA256: strings.Repeat("a", 64),
				RSSPeakBytes:     1024,
				RSSPeakPhase:     "module_codegen",
				WorkerCount:      1,
				WorkerReason:     "single job requested",
			},
		},
	}
}

func validTargetScopeForValidatorTest() targetScopeReport {
	return targetScopeReport{
		Schema:         TargetScopeSchema,
		HostTarget:     "linux/amd64",
		CompilerTarget: "linux-x64",
		SupportedTargets: []targetScopeTarget{
			{
				Target: "linux-x64",
				Status: "host_rss_measured",
				Reason: "compiler-process phase profiles record process RSS on linux/amd64 hosts",
			},
		},
		NonClaimTargets: []targetScopeTarget{
			{Target: "windows-x64", Status: "non_claim", Reason: "not measured"},
			{Target: "macos-x64", Status: "non_claim", Reason: "not measured"},
			{Target: "macos-arm64", Status: "non_claim", Reason: "not measured"},
			{Target: "linux-x86", Status: "non_claim", Reason: "not measured"},
			{Target: "linux-x32", Status: "non_claim", Reason: "not measured"},
			{Target: "wasm32-wasi", Status: "non_claim", Reason: "not measured"},
			{Target: "wasm32-web", Status: "non_claim", Reason: "not measured"},
		},
	}
}

func TestRunWarmsCompilerProcessBeforeMeasuredScenarios(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("compiler RSS bundle uses linux-x64 native compiler evidence")
	}

	outDir := t.TempDir()
	var events []string
	_, err := Run(Options{
		OutDir:      outDir,
		GitHead:     "0123456789abcdef0123456789abcdef01234567",
		Command:     []string{"ram-p7-compiler-rss", "--test"},
		Now:         func() time.Time { return time.Unix(1700000000, 0).UTC() },
		MemoryReset: func() { events = append(events, "reset") },
		ProcessWarmup: func() error {
			events = append(events, "warmup")
			return nil
		},
		Samples: 1,
		Scenarios: []Scenario{
			{Name: "small_reports_off_jobs_1_cold", ModuleCount: 3, Jobs: 1},
		},
	})
	if err != nil {
		t.Fatalf("Run compiler RSS harness: %v", err)
	}
	if len(events) < 2 || events[0] != "warmup" || events[1] != "reset" {
		t.Fatalf("events = %#v, want process warmup before first measured reset", events)
	}
}

func TestDefaultScenariosIncludeSmallMediumLargeReportPairs(t *testing.T) {
	type pairKey struct {
		moduleCount int
		jobs        int
		warmCache   bool
	}
	pairs := map[pairKey]map[bool]string{}
	for _, scenario := range defaultScenarios() {
		if scenario.ExpectCompileError {
			continue
		}
		key := pairKey{
			moduleCount: scenario.ModuleCount,
			jobs:        scenario.Jobs,
			warmCache:   scenario.WarmCache,
		}
		if pairs[key] == nil {
			pairs[key] = map[bool]string{}
		}
		pairs[key][scenario.Reports] = scenario.Name
	}

	want := []pairKey{
		{moduleCount: 3, jobs: 1, warmCache: false},
		{moduleCount: 6, jobs: 2, warmCache: false},
		{moduleCount: 12, jobs: runtime.NumCPU(), warmCache: true},
	}
	for _, key := range want {
		got := pairs[key]
		if got[false] == "" || got[true] == "" {
			t.Fatalf("default report pair for %+v = %#v, want report-off and report-on scenarios", key, got)
		}
	}
}

func TestP75ScenariosCoverSourcePlanMatrix(t *testing.T) {
	scenarios, err := scenariosForMatrix(ScenarioMatrixP75)
	if err != nil {
		t.Fatalf("scenariosForMatrix(%q): %v", ScenarioMatrixP75, err)
	}
	if len(scenarios) == 0 {
		t.Fatal("p7_5 matrix has no scenarios")
	}

	type coverageKey struct {
		moduleCount        int
		jobs               int
		reports            bool
		warmCache          bool
		expectCompileError bool
	}
	coverage := map[coverageKey]string{}
	for _, scenario := range scenarios {
		key := coverageKey{
			moduleCount:        scenario.ModuleCount,
			jobs:               scenario.Jobs,
			reports:            scenario.Reports,
			warmCache:          scenario.WarmCache,
			expectCompileError: scenario.ExpectCompileError,
		}
		if previous := coverage[key]; previous != "" {
			t.Fatalf("duplicate p7_5 scenario config %+v: %q and %q", key, previous, scenario.Name)
		}
		coverage[key] = scenario.Name
		if !strings.HasPrefix(scenario.Name, "p75_") {
			t.Fatalf("scenario %q missing p75 prefix", scenario.Name)
		}
	}

	moduleCounts := []int{3, 6, 12}
	jobCounts := expectedP75JobCounts(runtime.NumCPU())
	for _, modules := range moduleCounts {
		for _, jobs := range jobCounts {
			for _, reports := range []bool{false, true} {
				for _, warm := range []bool{false, true} {
					for _, expectError := range []bool{false, true} {
						key := coverageKey{
							moduleCount:        modules,
							jobs:               jobs,
							reports:            reports,
							warmCache:          warm,
							expectCompileError: expectError,
						}
						if coverage[key] == "" {
							t.Fatalf("p7_5 matrix missing coverage for %+v", key)
						}
					}
				}
			}
		}
	}
}

func TestP75ScenariosKeepReportPairsAdjacent(t *testing.T) {
	scenarios, err := scenariosForMatrix(ScenarioMatrixP75)
	if err != nil {
		t.Fatalf("scenariosForMatrix(%q): %v", ScenarioMatrixP75, err)
	}
	for i := 0; i < len(scenarios); i += 2 {
		if i+1 >= len(scenarios) {
			t.Fatalf("unpaired p7_5 scenario at index %d: %+v", i, scenarios[i])
		}
		off := scenarios[i]
		on := scenarios[i+1]
		if off.Reports || !on.Reports ||
			off.ModuleCount != on.ModuleCount ||
			off.Jobs != on.Jobs ||
			off.WarmCache != on.WarmCache ||
			off.ExpectCompileError != on.ExpectCompileError {
			t.Fatalf("p7_5 scenarios at %d/%d are not adjacent report-off/on pair: off=%+v on=%+v", i, i+1, off, on)
		}
	}
}

func TestRepresentativeScenariosUseRepoSourceReportPair(t *testing.T) {
	scenarios, err := scenariosForMatrix(ScenarioMatrixRepresentative)
	if err != nil {
		t.Fatalf("scenariosForMatrix(%q): %v", ScenarioMatrixRepresentative, err)
	}
	if len(scenarios) != 2 {
		t.Fatalf("representative scenarios = %d, want report-off/report-on pair", len(scenarios))
	}

	off, on := scenarios[0], scenarios[1]
	const source = "examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra"
	if off.SourcePath != source || on.SourcePath != source {
		t.Fatalf("representative source paths = %q/%q, want %q", off.SourcePath, on.SourcePath, source)
	}
	if off.Reports || !on.Reports ||
		off.Jobs != runtime.NumCPU() ||
		on.Jobs != runtime.NumCPU() ||
		off.WarmCache ||
		on.WarmCache ||
		off.ExpectCompileError ||
		on.ExpectCompileError {
		t.Fatalf("representative report pair = off=%+v on=%+v", off, on)
	}
}

func TestRunWritesRepresentativeSourceBundleWithoutRepoCache(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("compiler RSS bundle uses linux-x64 native compiler evidence")
	}

	repoRoot := t.TempDir()
	writeTestFile(t, filepath.Join(repoRoot, "examples", "rep", "main.t4"), `
module examples.rep.main
import lib.helper as helper
fun main(): i32 {
  return helper.value()
}
`)
	writeTestFile(t, filepath.Join(repoRoot, "lib", "helper.t4"), `
module lib.helper
fun value(): i32 {
  return 0
}
`)

	outDir := t.TempDir()
	result, err := Run(Options{
		OutDir:        outDir,
		RepoRoot:      repoRoot,
		GitHead:       "0123456789abcdef0123456789abcdef01234567",
		Command:       []string{"ram-p7-compiler-rss", "--matrix", "representative"},
		Now:           func() time.Time { return time.Unix(1700000000, 0).UTC() },
		MemoryReset:   func() {},
		ProcessWarmup: func() error { return nil },
		Samples:       1,
		Scenarios: []Scenario{
			{
				Name:       "representative_reports_off_jobs_1_cold",
				SourcePath: "examples/rep/main.t4",
				Jobs:       1,
			},
		},
	})
	if err != nil {
		t.Fatalf("Run representative compiler RSS harness: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, ".tetra_cache")); !os.IsNotExist(err) {
		t.Fatalf("repo source root cache state err=%v, want no .tetra_cache in representative repo root", err)
	}
	if _, err := os.Stat(filepath.Join(
		outDir,
		"scenarios",
		"representative_reports_off_jobs_1_cold",
		"samples",
		"sample-01",
		"src",
		"examples",
		"rep",
		"main.t4",
	)); err != nil {
		t.Fatalf("representative sample missing copied source: %v", err)
	}
	if _, err := os.Stat(filepath.Join(
		outDir,
		"scenarios",
		"representative_reports_off_jobs_1_cold",
		"samples",
		"sample-01",
		"src",
		".tetra_cache",
	)); !os.IsNotExist(err) {
		t.Fatalf("representative sample cache state err=%v, want no .tetra_cache left in bundle", err)
	}

	rawSummary, err := os.ReadFile(result.SummaryPath)
	if err != nil {
		t.Fatal(err)
	}
	var summary struct {
		Scenarios []struct {
			Name            string   `json:"name"`
			SourcePath      string   `json:"source_path"`
			CompiledModules []string `json:"compiled_modules"`
		} `json:"scenarios"`
	}
	if err := json.Unmarshal(rawSummary, &summary); err != nil {
		t.Fatalf("summary JSON: %v", err)
	}
	if len(summary.Scenarios) != 1 {
		t.Fatalf("summary scenarios = %#v", summary.Scenarios)
	}
	got := summary.Scenarios[0]
	if got.SourcePath != "examples/rep/main.t4" {
		t.Fatalf("source_path = %q, want representative source", got.SourcePath)
	}
	if !containsString(got.CompiledModules, "examples.rep.main") ||
		!containsString(got.CompiledModules, "lib.helper") {
		t.Fatalf("compiled modules = %#v, want copied representative entry plus dependency", got.CompiledModules)
	}
}

func TestRunWritesBatchSourceBundleForFullRepositoryWorkload(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("compiler RSS bundle uses linux-x64 native compiler evidence")
	}

	repoRoot := t.TempDir()
	writeTestFile(t, filepath.Join(repoRoot, "examples", "batch", "first.tetra"), `
module examples.batch.first
import lib.helper as helper
func main() -> Int:
    return helper.value()
`)
	writeTestFile(t, filepath.Join(repoRoot, "examples", "batch", "second.tetra"), `
module examples.batch.second
import lib.helper as helper
func main() -> Int:
    return helper.value()
`)
	writeTestFile(t, filepath.Join(repoRoot, "lib", "helper.tetra"), `
module lib.helper
func value() -> Int:
    return 0
`)

	outDir := t.TempDir()
	result, err := Run(Options{
		OutDir:        outDir,
		RepoRoot:      repoRoot,
		GitHead:       "0123456789abcdef0123456789abcdef01234567",
		Command:       []string{"ram-p7-compiler-rss", "--matrix", "full_repo"},
		Now:           func() time.Time { return time.Unix(1700000000, 0).UTC() },
		MemoryReset:   func() {},
		ProcessWarmup: func() error { return nil },
		Samples:       1,
		Scenarios: []Scenario{
			{
				Name: "full_repo_linux_x64_smoke_profile_reports_off_jobs_1_cold",
				SourcePaths: []string{
					"examples/batch/first.tetra",
					"examples/batch/second.tetra",
				},
				Jobs: 1,
			},
		},
	})
	if err != nil {
		t.Fatalf("Run batch compiler RSS harness: %v", err)
	}
	for _, rel := range []string{
		filepath.Join("examples", "batch", "first.tetra"),
		filepath.Join("examples", "batch", "second.tetra"),
	} {
		if _, err := os.Stat(filepath.Join(
			outDir,
			"scenarios",
			"full_repo_linux_x64_smoke_profile_reports_off_jobs_1_cold",
			"samples",
			"sample-01",
			"src",
			rel,
		)); err != nil {
			t.Fatalf("batch sample missing copied source %s: %v", rel, err)
		}
	}

	rawSummary, err := os.ReadFile(result.SummaryPath)
	if err != nil {
		t.Fatal(err)
	}
	var summary struct {
		Scenarios []struct {
			Name                 string   `json:"name"`
			SourcePaths          []string `json:"source_paths"`
			SourceCount          int      `json:"source_count"`
			ExecutableCount      int      `json:"executable_count"`
			CompilerProfileCount int      `json:"compiler_profile_count"`
			ExecutableSHA256     string   `json:"executable_sha256"`
			CompiledModules      []string `json:"compiled_modules"`
			Samples              []struct {
				ExecutableCount      int `json:"executable_count"`
				CompilerProfileCount int `json:"compiler_profile_count"`
				Executables          []struct {
					SourcePath       string `json:"source_path"`
					Executable       string `json:"executable"`
					ExecutableSHA256 string `json:"executable_sha256"`
					PhaseProfile     string `json:"phase_profile"`
				} `json:"executables"`
			} `json:"samples"`
		} `json:"scenarios"`
	}
	if err := json.Unmarshal(rawSummary, &summary); err != nil {
		t.Fatalf("summary JSON: %v", err)
	}
	if len(summary.Scenarios) != 1 {
		t.Fatalf("summary scenarios = %#v", summary.Scenarios)
	}
	got := summary.Scenarios[0]
	if got.SourceCount != 2 ||
		got.ExecutableCount != 2 ||
		got.CompilerProfileCount != 2 ||
		len(got.ExecutableSHA256) != 64 ||
		!containsString(got.SourcePaths, "examples/batch/first.tetra") ||
		!containsString(got.SourcePaths, "examples/batch/second.tetra") {
		t.Fatalf("batch scenario summary = %+v", got)
	}
	if len(got.Samples) != 1 ||
		got.Samples[0].ExecutableCount != 2 ||
		got.Samples[0].CompilerProfileCount != 2 ||
		len(got.Samples[0].Executables) != 2 {
		t.Fatalf("batch sample summary = %+v", got.Samples)
	}
	for _, executable := range got.Samples[0].Executables {
		if !strings.HasPrefix(executable.SourcePath, "examples/batch/") ||
			strings.TrimSpace(executable.Executable) == "" ||
			len(executable.ExecutableSHA256) != 64 ||
			strings.TrimSpace(executable.PhaseProfile) == "" {
			t.Fatalf("batch executable summary = %+v", executable)
		}
	}
	if !containsString(got.CompiledModules, "examples.batch.first") ||
		!containsString(got.CompiledModules, "examples.batch.second") ||
		!containsString(got.CompiledModules, "lib.helper") {
		t.Fatalf("compiled modules = %#v, want both entries plus dependency", got.CompiledModules)
	}
}

func TestFullRepositoryScenariosUseLinuxX64SmokeProfileReportPair(t *testing.T) {
	scenarios, err := scenariosForMatrix(ScenarioMatrixFullRepository)
	if err != nil {
		t.Fatalf("scenariosForMatrix(%q): %v", ScenarioMatrixFullRepository, err)
	}
	if len(scenarios) != 2 {
		t.Fatalf("full_repo scenarios = %d, want report-off/report-on pair", len(scenarios))
	}

	off, on := scenarios[0], scenarios[1]
	if off.Reports || !on.Reports ||
		off.Jobs != runtime.NumCPU() ||
		on.Jobs != runtime.NumCPU() ||
		off.WarmCache ||
		on.WarmCache ||
		off.ExpectCompileError ||
		on.ExpectCompileError {
		t.Fatalf("full_repo report pair = off=%+v on=%+v", off, on)
	}
	if len(off.SourcePaths) < 50 || len(on.SourcePaths) != len(off.SourcePaths) {
		t.Fatalf("full_repo source paths off=%d on=%d, want shared smoke profile", len(off.SourcePaths), len(on.SourcePaths))
	}
	for _, want := range []string{
		"examples/memory/islands/islands_hello.tetra",
		"examples/flow/flow_hello.tetra",
		"examples/core/data/core_math_smoke.tetra",
		"examples/projects/dogfood_cli/src/main.tetra",
	} {
		if !containsString(off.SourcePaths, want) || !containsString(on.SourcePaths, want) {
			t.Fatalf("full_repo source paths missing %s", want)
		}
	}
}

func TestScenariosForMatrixRejectsUnknownMatrix(t *testing.T) {
	_, err := scenariosForMatrix("fantasy")
	if err == nil || !strings.Contains(err.Error(), "unsupported compiler RSS scenario matrix") {
		t.Fatalf("unknown matrix error = %v", err)
	}
}

func expectedP75JobCounts(numCPU int) []int {
	cases := p75JobCases(numCPU)
	out := make([]int, 0, len(cases))
	for _, c := range cases {
		out = append(out, c.count)
	}
	return out
}

func writeTestFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(body)+"\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func TestReportComparisonEvaluatesBoundFromSampleDispersion(t *testing.T) {
	off := scenarioSummary{
		Name:               "large_reports_off_jobs_4_warm",
		ModuleCount:        12,
		Jobs:               4,
		WarmCache:          true,
		SampleCount:        5,
		RSSMedianBytes:     1000,
		RSSDispersionBytes: 30,
		RSSPeakPhase:       "module_codegen",
	}
	on := scenarioSummary{
		Name:               "large_reports_on_jobs_4_warm",
		ModuleCount:        12,
		Jobs:               4,
		Reports:            true,
		WarmCache:          true,
		SampleCount:        5,
		RSSMedianBytes:     1040,
		RSSDispersionBytes: 20,
		RSSPeakPhase:       "report_generation",
	}
	comparisons := buildReportComparisons([]scenarioSummary{on, off})
	if len(comparisons) != 1 {
		t.Fatalf("comparisons = %#v, want one pair", comparisons)
	}
	got := comparisons[0]
	if got.Name != "large_reports_jobs_4_warm" ||
		got.ReportOffScenario != off.Name ||
		got.ReportOnScenario != on.Name ||
		got.BoundRSSBytes != 1050 ||
		got.DeltaBytes != 40 ||
		got.Ratio != 1.04 ||
		got.EvaluationStatus != "pass" ||
		got.BoundBasis != "report_off_median_plus_observed_off_on_dispersion" ||
		!got.SameHostSameConfig {
		t.Fatalf("comparison = %+v", got)
	}

	on.RSSMedianBytes = 1100
	comparisons = buildReportComparisons([]scenarioSummary{off, on})
	if len(comparisons) != 1 || comparisons[0].EvaluationStatus != "fail" {
		t.Fatalf("failing comparison = %#v, want fail", comparisons)
	}

	on.SampleCount = 1
	comparisons = buildReportComparisons([]scenarioSummary{off, on})
	if len(comparisons) != 1 || comparisons[0].EvaluationStatus != "insufficient_samples" {
		t.Fatalf("insufficient comparison = %#v, want insufficient_samples", comparisons)
	}
}
