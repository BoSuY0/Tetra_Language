package ramcompilerrss

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"tetra_language/compiler"
	"tetra_language/tools/internal/artifacts"
)

const Schema = "tetra.ram.p7-compiler-rss-bundle.v2"
const SummarySchema = "tetra.ram.p7-compiler-rss-summary.v2"
const TargetScopeSchema = "tetra.ram.p7-compiler-rss-target-scope.v1"

const (
	ScenarioMatrixDefault        = "default"
	ScenarioMatrixP75            = "p7_5"
	ScenarioMatrixRepresentative = "representative"
	ScenarioMatrixFullRepository = "full_repo"
)

const representativeSurfaceMorphFlagshipSource = "examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra"

type Options struct {
	OutDir         string
	RepoRoot       string
	GitHead        string
	GitStatusShort string
	Command        []string
	Now            func() time.Time
	MemoryReset    func()
	ProcessWarmup  func() error
	Samples        int
	Matrix         string
	Scenarios      []Scenario
}

type Scenario struct {
	Name               string   `json:"name"`
	SourcePath         string   `json:"source_path,omitempty"`
	SourcePaths        []string `json:"source_paths,omitempty"`
	ModuleCount        int      `json:"module_count"`
	Jobs               int      `json:"jobs"`
	Reports            bool     `json:"reports"`
	WarmCache          bool     `json:"warm_cache"`
	MemoryBudgetBytes  int64    `json:"memory_budget_bytes,omitempty"`
	ExpectCompileError bool     `json:"expect_compile_error,omitempty"`
}

type Result struct {
	OutDir              string
	ManifestPath        string
	SummaryPath         string
	ValidatorOutputPath string
}

type bundleManifest struct {
	Schema                 string   `json:"schema"`
	GeneratedAt            string   `json:"generated_at"`
	GitHead                string   `json:"git_head"`
	GitDirty               bool     `json:"git_dirty"`
	GitStatusShort         []string `json:"git_status_short,omitempty"`
	TargetOS               string   `json:"target_os"`
	TargetArch             string   `json:"target_arch"`
	CompilerProfileSchema  string   `json:"compiler_profile_schema"`
	ScenarioSummary        string   `json:"scenario_summary"`
	HostFingerprint        string   `json:"host_fingerprint"`
	TargetScope            string   `json:"target_scope"`
	CommandManifest        string   `json:"command_manifest"`
	ValidatorOutput        string   `json:"validator_output"`
	ArtifactHashManifest   string   `json:"artifact_hash_manifest"`
	ArtifactHashValidation string   `json:"artifact_hash_validation"`
	Notes                  []string `json:"notes,omitempty"`
}

type summaryReport struct {
	Schema            string             `json:"schema"`
	ReportComparisons []reportComparison `json:"report_comparisons,omitempty"`
	Scenarios         []scenarioSummary  `json:"scenarios"`
}

type scenarioSummary struct {
	Name                   string          `json:"name"`
	SourcePath             string          `json:"source_path,omitempty"`
	SourcePaths            []string        `json:"source_paths,omitempty"`
	SourceCount            int             `json:"source_count,omitempty"`
	ModuleCount            int             `json:"module_count"`
	Jobs                   int             `json:"jobs"`
	Reports                bool            `json:"reports"`
	WarmCache              bool            `json:"warm_cache"`
	MemoryBudgetBytes      int64           `json:"memory_budget_bytes,omitempty"`
	ExpectCompileError     bool            `json:"expect_compile_error,omitempty"`
	CompileErrorObserved   bool            `json:"compile_error_observed,omitempty"`
	CompileError           string          `json:"compile_error,omitempty"`
	CompileErrorSampleNum  int             `json:"compile_error_sample_num,omitempty"`
	SampleCount            int             `json:"sample_count"`
	Samples                []sampleSummary `json:"samples"`
	PhaseProfile           string          `json:"phase_profile"`
	CompilerProfiles       []string        `json:"compiler_profiles,omitempty"`
	CompilerProfileCount   int             `json:"compiler_profile_count,omitempty"`
	Executable             string          `json:"executable"`
	ExecutableSHA256       string          `json:"executable_sha256"`
	ExecutableCount        int             `json:"executable_count,omitempty"`
	RSSPeakBytes           uint64          `json:"rss_peak_bytes"`
	RSSMedianBytes         uint64          `json:"rss_median_bytes"`
	RSSMinBytes            uint64          `json:"rss_min_bytes"`
	RSSMaxBytes            uint64          `json:"rss_max_bytes"`
	RSSDispersionBytes     uint64          `json:"rss_dispersion_bytes"`
	RSSPeakPhase           string          `json:"rss_peak_phase"`
	GoHeapPeakBytes        uint64          `json:"go_heap_peak_alloc_bytes"`
	GoHeapMedianAllocBytes uint64          `json:"go_heap_median_alloc_bytes"`
	WorkerCount            int             `json:"worker_count"`
	WorkerReason           string          `json:"worker_reason"`
	ReportMode             string          `json:"report_mode"`
	CacheHits              []string        `json:"cache_hits,omitempty"`
	CompiledModules        []string        `json:"compiled_modules,omitempty"`
	LoweredModules         []string        `json:"lowered_modules,omitempty"`
}

type reportComparison struct {
	Name                  string   `json:"name"`
	SourcePath            string   `json:"source_path,omitempty"`
	SourcePaths           []string `json:"source_paths,omitempty"`
	ReportOffScenario     string   `json:"report_off_scenario"`
	ReportOnScenario      string   `json:"report_on_scenario"`
	ModuleCount           int      `json:"module_count"`
	Jobs                  int      `json:"jobs"`
	WarmCache             bool     `json:"warm_cache"`
	MemoryBudgetBytes     int64    `json:"memory_budget_bytes,omitempty"`
	ReportOffSampleCount  int      `json:"report_off_sample_count"`
	ReportOnSampleCount   int      `json:"report_on_sample_count"`
	ReportOffRSSMedian    uint64   `json:"report_off_rss_median_bytes"`
	ReportOnRSSMedian     uint64   `json:"report_on_rss_median_bytes"`
	ReportOffDispersion   uint64   `json:"report_off_rss_dispersion_bytes"`
	ReportOnDispersion    uint64   `json:"report_on_rss_dispersion_bytes"`
	BoundRSSBytes         uint64   `json:"bound_rss_bytes"`
	DeltaBytes            int64    `json:"delta_bytes"`
	Ratio                 float64  `json:"ratio"`
	EvaluationStatus      string   `json:"evaluation_status"`
	BoundBasis            string   `json:"bound_basis"`
	SameHostSameConfig    bool     `json:"same_host_same_config"`
	ReportOffRSSPeakPhase string   `json:"report_off_rss_peak_phase"`
	ReportOnRSSPeakPhase  string   `json:"report_on_rss_peak_phase"`
}

type sampleSummary struct {
	Index                int                `json:"index"`
	SourcePaths          []string           `json:"source_paths,omitempty"`
	PhaseProfile         string             `json:"phase_profile"`
	CompilerProfiles     []string           `json:"compiler_profiles,omitempty"`
	CompilerProfileCount int                `json:"compiler_profile_count,omitempty"`
	Executable           string             `json:"executable,omitempty"`
	ExecutableSHA256     string             `json:"executable_sha256,omitempty"`
	Executables          []sampleExecutable `json:"executables,omitempty"`
	ExecutableCount      int                `json:"executable_count,omitempty"`
	RSSPeakBytes         uint64             `json:"rss_peak_bytes"`
	RSSPeakPhase         string             `json:"rss_peak_phase"`
	GoHeapPeakBytes      uint64             `json:"go_heap_peak_alloc_bytes"`
	WorkerCount          int                `json:"worker_count"`
	WorkerReason         string             `json:"worker_reason"`
	ReportMode           string             `json:"report_mode"`
	CacheHits            []string           `json:"cache_hits,omitempty"`
	CompiledModules      []string           `json:"compiled_modules,omitempty"`
	LoweredModules       []string           `json:"lowered_modules,omitempty"`
	CompileErrorObserved bool               `json:"compile_error_observed,omitempty"`
	CompileError         string             `json:"compile_error,omitempty"`
}

type sampleExecutable struct {
	SourcePath       string `json:"source_path"`
	PhaseProfile     string `json:"phase_profile"`
	Executable       string `json:"executable"`
	ExecutableSHA256 string `json:"executable_sha256"`
}

type hostFingerprint struct {
	Schema        string `json:"schema"`
	Hostname      string `json:"hostname"`
	GOOS          string `json:"goos"`
	GOARCH        string `json:"goarch"`
	NumCPU        int    `json:"num_cpu"`
	KernelRelease string `json:"kernel_release,omitempty"`
	OSRelease     string `json:"os_release,omitempty"`
	CPUModel      string `json:"cpu_model,omitempty"`
}

type targetScopeReport struct {
	Schema           string              `json:"schema"`
	HostTarget       string              `json:"host_target"`
	CompilerTarget   string              `json:"compiler_target"`
	SupportedTargets []targetScopeTarget `json:"supported_targets"`
	NonClaimTargets  []targetScopeTarget `json:"non_claim_targets"`
	Notes            []string            `json:"notes,omitempty"`
}

type targetScopeTarget struct {
	Target string `json:"target"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type commandManifest struct {
	Schema   string          `json:"schema"`
	Commands []commandRecord `json:"commands"`
}

type commandRecord struct {
	Name    string   `json:"name"`
	Command []string `json:"command"`
}

type compilerPhaseProfile struct {
	Schema               string                 `json:"schema"`
	ReportMode           string                 `json:"report_mode"`
	WorkerCount          int                    `json:"worker_count"`
	WorkerReason         string                 `json:"worker_reason"`
	GoHeapPeakAllocBytes uint64                 `json:"go_heap_peak_alloc_bytes"`
	RSSPeakBytes         uint64                 `json:"rss_peak_bytes"`
	RSSSupported         bool                   `json:"rss_supported"`
	Phases               []compilerProfilePhase `json:"phases"`
}

type compilerProfilePhase struct {
	Name            string `json:"name"`
	RSSCurrentBytes uint64 `json:"rss_current_bytes"`
}

func Run(opts Options) (Result, error) {
	if strings.TrimSpace(opts.OutDir) == "" {
		return Result{}, fmt.Errorf("compiler RSS out dir is required")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return Result{}, fmt.Errorf("compiler RSS harness requires linux/amd64")
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	scenarios := opts.Scenarios
	if len(scenarios) == 0 {
		var err error
		scenarios, err = scenariosForMatrix(opts.Matrix)
		if err != nil {
			return Result{}, err
		}
	}
	samples := opts.Samples
	if samples <= 0 {
		samples = 1
	}
	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return Result{}, err
	}
	memoryReset := opts.MemoryReset
	if memoryReset == nil {
		memoryReset = releaseProcessMemoryBeforeMeasuredBuild
	}
	processWarmup := opts.ProcessWarmup
	if processWarmup == nil {
		processWarmup = func() error { return warmCompilerProcess(opts.OutDir) }
	}
	if err := processWarmup(); err != nil {
		return Result{}, fmt.Errorf("compiler RSS process warmup: %w", err)
	}

	var summaries []scenarioSummary
	for _, scenario := range scenarios {
		summary, err := runScenario(opts.OutDir, opts.RepoRoot, scenario, samples, memoryReset)
		if err != nil {
			return Result{}, err
		}
		summaries = append(summaries, summary)
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].Name < summaries[j].Name })

	reportComparisons := buildReportComparisons(summaries)
	summaryPath := filepath.Join(opts.OutDir, "scenario-summary.json")
	if err := writeJSON(summaryPath, summaryReport{
		Schema:            SummarySchema,
		ReportComparisons: reportComparisons,
		Scenarios:         summaries,
	}); err != nil {
		return Result{}, err
	}

	targetScope := readTargetScope()
	validatorOutputPath := filepath.Join(opts.OutDir, "validator-output.txt")
	validatorOutput := validateBundleOutput(summaries, targetScope)
	if err := os.WriteFile(validatorOutputPath, []byte(validatorOutput), 0o644); err != nil {
		return Result{}, err
	}
	if !strings.Contains(validatorOutput, "result: pass") {
		return Result{}, fmt.Errorf("compiler RSS validator failed")
	}

	hostPath := filepath.Join(opts.OutDir, "host-fingerprint.json")
	if err := writeJSON(hostPath, readHostFingerprint()); err != nil {
		return Result{}, err
	}
	targetScopePath := filepath.Join(opts.OutDir, "target-scope.json")
	if err := writeJSON(targetScopePath, targetScope); err != nil {
		return Result{}, err
	}
	hashPlan, err := artifacts.NewHashCommandPlan(opts.OutDir)
	if err != nil {
		return Result{}, err
	}
	commandPath := filepath.Join(opts.OutDir, "command-manifest.json")
	if err := writeJSON(commandPath, commandManifest{
		Schema: "tetra.ram.p7-compiler-rss.commands.v1",
		Commands: []commandRecord{
			{Name: "ram-p7-compiler-rss", Command: append([]string(nil), opts.Command...)},
			{Name: hashPlan.Write.Name, Command: hashPlan.Write.Args},
			{Name: hashPlan.Validate.Name, Command: hashPlan.Validate.Args},
		},
	}); err != nil {
		return Result{}, err
	}

	manifestPath := filepath.Join(opts.OutDir, "compiler-rss-manifest.json")
	if err := writeJSON(manifestPath, bundleManifest{
		Schema:                 Schema,
		GeneratedAt:            now().UTC().Format(time.RFC3339),
		GitHead:                strings.TrimSpace(opts.GitHead),
		GitDirty:               strings.TrimSpace(opts.GitStatusShort) != "",
		GitStatusShort:         splitNonEmptyLines(opts.GitStatusShort),
		TargetOS:               runtime.GOOS,
		TargetArch:             runtime.GOARCH,
		CompilerProfileSchema:  "tetra.compiler.phase-profile.v1",
		ScenarioSummary:        "scenario-summary.json",
		HostFingerprint:        "host-fingerprint.json",
		TargetScope:            "target-scope.json",
		CommandManifest:        "command-manifest.json",
		ValidatorOutput:        "validator-output.txt",
		ArtifactHashManifest:   artifacts.HashManifestName,
		ArtifactHashValidation: "artifact-hashes-validation.txt",
		Notes: []string{
			"P7 compiler RSS bundle records compiler-process phase profile evidence",
			"RSS values are host-local diagnostics, not cross-host performance claims",
		},
	}); err != nil {
		return Result{}, err
	}
	return Result{
		OutDir:              opts.OutDir,
		ManifestPath:        manifestPath,
		SummaryPath:         summaryPath,
		ValidatorOutputPath: validatorOutputPath,
	}, nil
}

func releaseProcessMemoryBeforeMeasuredBuild() {
	runtime.GC()
	debug.FreeOSMemory()
}

func warmCompilerProcess(root string) error {
	dir, err := os.MkdirTemp(root, ".process-warmup-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	srcRoot := filepath.Join(dir, "src")
	if err := writeScenarioSources(srcRoot, 1, false); err != nil {
		return err
	}
	entry := filepath.Join(srcRoot, "app", "main.t4")
	outPath := filepath.Join(dir, "out", "warmup-app")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	if _, err := buildScenario(entry, outPath, "", Scenario{ModuleCount: 1, Jobs: 1}, scenarioBuildRoots{}); err != nil {
		return err
	}
	releaseProcessMemoryBeforeMeasuredBuild()
	return nil
}

func defaultScenarios() []Scenario {
	return []Scenario{
		{Name: "small_reports_off_jobs_1_cold", ModuleCount: 3, Jobs: 1},
		{Name: "small_reports_on_jobs_1_cold", ModuleCount: 3, Jobs: 1, Reports: true},
		{Name: "medium_reports_off_jobs_2_cold", ModuleCount: 6, Jobs: 2},
		{Name: "medium_reports_on_jobs_2_cold", ModuleCount: 6, Jobs: 2, Reports: true},
		{Name: "large_reports_off_jobs_cpu_warm", ModuleCount: 12, Jobs: runtime.NumCPU(), WarmCache: true},
		{Name: "large_reports_on_jobs_cpu_warm", ModuleCount: 12, Jobs: runtime.NumCPU(), Reports: true, WarmCache: true},
		{Name: "compile_error_reports_off_jobs_1_cold", ModuleCount: 3, Jobs: 1, ExpectCompileError: true},
	}
}

func scenariosForMatrix(matrix string) ([]Scenario, error) {
	switch strings.ToLower(strings.TrimSpace(matrix)) {
	case "", ScenarioMatrixDefault:
		return defaultScenarios(), nil
	case ScenarioMatrixP75, "p75", "p7.5":
		return p75Scenarios(), nil
	case ScenarioMatrixRepresentative, "representative_large", "p7_representative":
		return representativeScenarios(), nil
	case ScenarioMatrixFullRepository, "full-repo", "full_repository":
		return fullRepositoryScenarios(), nil
	default:
		return nil, fmt.Errorf("unsupported compiler RSS scenario matrix %q", matrix)
	}
}

func representativeScenarios() []Scenario {
	return []Scenario{
		{
			Name:        "representative_surface_morph_flagship_reports_off_jobs_cpu_cold",
			SourcePath:  representativeSurfaceMorphFlagshipSource,
			ModuleCount: 4,
			Jobs:        runtime.NumCPU(),
		},
		{
			Name:        "representative_surface_morph_flagship_reports_on_jobs_cpu_cold",
			SourcePath:  representativeSurfaceMorphFlagshipSource,
			ModuleCount: 4,
			Jobs:        runtime.NumCPU(),
			Reports:     true,
		},
	}
}

func fullRepositoryScenarios() []Scenario {
	sourcePaths := fullRepositoryLinuxX64SmokeProfileSourcePaths()
	return []Scenario{
		{
			Name:        "full_repo_linux_x64_smoke_profile_reports_off_jobs_cpu_cold",
			SourcePaths: sourcePaths,
			ModuleCount: len(sourcePaths),
			Jobs:        runtime.NumCPU(),
		},
		{
			Name:        "full_repo_linux_x64_smoke_profile_reports_on_jobs_cpu_cold",
			SourcePaths: sourcePaths,
			ModuleCount: len(sourcePaths),
			Jobs:        runtime.NumCPU(),
			Reports:     true,
		},
	}
}

func fullRepositoryLinuxX64SmokeProfileSourcePaths() []string {
	paths := []string{
		"examples/memory/islands/islands_hello.tetra",
		"examples/memory/islands/islands_i32.tetra",
		"examples/memory/islands/islands_overflow.tetra",
		"examples/memory/raw/mmio_smoke.tetra",
		"examples/memory/raw/cap_mem_smoke.tetra",
		"examples/memory/raw/memset_smoke.tetra",
		"examples/actors/actors_pingpong.tetra",
		"examples/actors/actor_sleep_pingpong.tetra",
		"examples/flow/flow_hello.tetra",
		"examples/flow/flow_struct_smoke.tetra",
		"examples/flow/flow_islands_smoke.tetra",
		"examples/flow/flow_unsafe_cap_mem_smoke.tetra",
		"examples/ui/ui_native_shell_smoke.tetra",
		"examples/smoke/scalars/bool_smoke.tetra",
		"examples/smoke/control/for_range_smoke.tetra",
		"examples/smoke/control/for_collection_smoke.tetra",
		"examples/smoke/control/for_collection_u8_smoke.tetra",
		"examples/smoke/control/loop_control_smoke.tetra",
		"examples/smoke/control/complex_control_flow_smoke.tetra",
		"examples/smoke/scalars/unary_not_smoke.tetra",
		"examples/smoke/scalars/const_smoke.tetra",
		"examples/smoke/scalars/const_bool_smoke.tetra",
		"examples/smoke/scalars/local_const_smoke.tetra",
		"examples/smoke/scalars/compound_assignment_smoke.tetra",
		"examples/smoke/control/else_if_smoke.tetra",
		"examples/smoke/types/enum_match_smoke.tetra",
		"examples/smoke/types/enum_exhaustive_match_smoke.tetra",
		"examples/effects/effects_io_smoke.tetra",
		"examples/effects/effects_mem_smoke.tetra",
		"examples/effects/effects_actors_smoke.tetra",
		"examples/smoke/types/optional_smoke.tetra",
		"examples/smoke/types/optional_match_smoke.tetra",
		"examples/smoke/types/optional_match_some_smoke.tetra",
		"examples/memory/ownership/ownership_smoke.tetra",
		"examples/smoke/errors/typed_errors_smoke.tetra",
		"examples/async/async_smoke.tetra",
		"examples/tasks/task_smoke.tetra",
		"examples/async/time_sleep_smoke.tetra",
		"examples/tasks/task_sleep_deadline_smoke.tetra",
		"examples/tasks/task_join_wait_smoke.tetra",
		"examples/tasks/task_group_cancel_smoke.tetra",
		"examples/tasks/task_group_lifecycle_smoke.tetra",
		"examples/async/deadline_aware_waits_smoke.tetra",
		"examples/async/wait_composition_smoke.tetra",
		"examples/core/data/core_math_smoke.tetra",
		"examples/core/memory/core_memory_smoke.tetra",
		"examples/core/data/core_strings_smoke.tetra",
		"examples/core/data/core_slices_smoke.tetra",
		"examples/core/platform/core_io_smoke.tetra",
		"examples/core/runtime/core_testing_smoke.tetra",
		"examples/core/data/core_collections_smoke.tetra",
		"examples/core/surface/core_component_smoke.tetra",
		"examples/core/data/core_serialization_smoke.tetra",
		"examples/core/platform/core_filesystem_smoke.tetra",
		"examples/core/platform/core_networking_smoke.tetra",
		"examples/async/core_async_smoke.tetra",
		"examples/core/runtime/core_sync_smoke.tetra",
		"examples/core/platform/core_time_smoke.tetra",
		"examples/core/memory/core_crypto_smoke.tetra",
		"examples/core/memory/core_capability_smoke.tetra",
		"examples/smoke/language/extension_smoke.tetra",
		"examples/smoke/language/generic_smoke.tetra",
		"examples/smoke/language/protocol_impl_smoke.tetra",
		"examples/surface/runtime/surface_counter.tetra",
		"examples/surface/runtime/surface_text_input.tetra",
		"examples/surface/migration/surface_migration_ui_web_smoke.tetra",
		"examples/surface/migration/surface_migration_ui_native_shell_smoke.tetra",
		"examples/surface/migration/surface_migration_dogfood_web_ui.tetra",
		"examples/surface/migration/surface_migration_tetra_control_center.tetra",
		"examples/projects/dogfood_cli/src/main.tetra",
		"examples/projects/dogfood_actor_task/src/main.tetra",
	}
	return append([]string(nil), paths...)
}

type p75JobCase struct {
	name  string
	count int
}

func p75Scenarios() []Scenario {
	sizes := []struct {
		name  string
		count int
	}{
		{name: "small", count: 3},
		{name: "medium", count: 6},
		{name: "large", count: 12},
	}
	outcomes := []struct {
		name               string
		expectCompileError bool
	}{
		{name: "success"},
		{name: "compile_error", expectCompileError: true},
	}
	reportModes := []struct {
		name    string
		reports bool
	}{
		{name: "off"},
		{name: "on", reports: true},
	}
	cacheModes := []struct {
		name      string
		warmCache bool
	}{
		{name: "cold"},
		{name: "warm", warmCache: true},
	}

	var scenarios []Scenario
	for _, size := range sizes {
		for _, job := range p75JobCases(runtime.NumCPU()) {
			for _, cache := range cacheModes {
				for _, outcome := range outcomes {
					for _, report := range reportModes {
						scenarios = append(scenarios, Scenario{
							Name: fmt.Sprintf(
								"p75_%s_%s_reports_%s_jobs_%s_%s",
								size.name,
								outcome.name,
								report.name,
								job.name,
								cache.name,
							),
							ModuleCount:        size.count,
							Jobs:               job.count,
							Reports:            report.reports,
							WarmCache:          cache.warmCache,
							ExpectCompileError: outcome.expectCompileError,
						})
					}
				}
			}
		}
	}
	return scenarios
}

func p75JobCases(numCPU int) []p75JobCase {
	if numCPU <= 0 {
		numCPU = 1
	}
	candidates := []p75JobCase{
		{name: "1", count: 1},
		{name: "2", count: 2},
		{name: "4", count: 4},
		{name: "cpu", count: numCPU},
	}
	seen := map[int]struct{}{}
	var out []p75JobCase
	for _, candidate := range candidates {
		if _, ok := seen[candidate.count]; ok {
			continue
		}
		seen[candidate.count] = struct{}{}
		out = append(out, candidate)
	}
	return out
}

func runScenario(root string, repoRoot string, scenario Scenario, samples int, memoryReset func()) (scenarioSummary, error) {
	if strings.TrimSpace(scenario.Name) == "" {
		return scenarioSummary{}, fmt.Errorf("scenario name is required")
	}
	name, err := safeScenarioName(scenario.Name)
	if err != nil {
		return scenarioSummary{}, err
	}
	if scenario.ModuleCount <= 0 {
		if len(scenario.SourcePaths) > 0 {
			scenario.ModuleCount = len(scenario.SourcePaths)
		} else {
			scenario.ModuleCount = 3
		}
	}
	if scenario.Jobs <= 0 {
		scenario.Jobs = 1
	}
	dir := filepath.Join(root, "scenarios", name)
	if err := os.RemoveAll(dir); err != nil {
		return scenarioSummary{}, err
	}
	var rawSamples []sampleSummary
	for i := 1; i <= samples; i++ {
		sample, err := runScenarioSample(dir, name, repoRoot, scenario, i, memoryReset)
		if err != nil {
			return scenarioSummary{}, err
		}
		rawSamples = append(rawSamples, sample)
	}
	aggregate := aggregateSamples(rawSamples)
	representative := rawSamples[0]
	sourcePaths := normalizedScenarioSourcePaths(scenario)
	return scenarioSummary{
		Name:                   name,
		SourcePath:             strings.TrimSpace(filepath.ToSlash(scenario.SourcePath)),
		SourcePaths:            sourcePaths,
		SourceCount:            len(sourcePaths),
		ModuleCount:            scenario.ModuleCount,
		Jobs:                   scenario.Jobs,
		Reports:                scenario.Reports,
		WarmCache:              scenario.WarmCache,
		MemoryBudgetBytes:      scenario.MemoryBudgetBytes,
		ExpectCompileError:     scenario.ExpectCompileError,
		CompileErrorObserved:   aggregate.compileErrorObserved,
		CompileError:           aggregate.compileError,
		CompileErrorSampleNum:  aggregate.compileErrorSampleNum,
		SampleCount:            len(rawSamples),
		Samples:                rawSamples,
		PhaseProfile:           representative.PhaseProfile,
		CompilerProfiles:       append([]string(nil), representative.CompilerProfiles...),
		CompilerProfileCount:   representative.CompilerProfileCount,
		Executable:             representative.Executable,
		ExecutableSHA256:       representative.ExecutableSHA256,
		ExecutableCount:        representative.ExecutableCount,
		RSSPeakBytes:           aggregate.rssMedian,
		RSSMedianBytes:         aggregate.rssMedian,
		RSSMinBytes:            aggregate.rssMin,
		RSSMaxBytes:            aggregate.rssMax,
		RSSDispersionBytes:     aggregate.rssMax - aggregate.rssMin,
		RSSPeakPhase:           representative.RSSPeakPhase,
		GoHeapPeakBytes:        aggregate.goHeapMedian,
		GoHeapMedianAllocBytes: aggregate.goHeapMedian,
		WorkerCount:            representative.WorkerCount,
		WorkerReason:           representative.WorkerReason,
		ReportMode:             representative.ReportMode,
		CacheHits:              append([]string(nil), representative.CacheHits...),
		CompiledModules:        append([]string(nil), representative.CompiledModules...),
		LoweredModules:         append([]string(nil), representative.LoweredModules...),
	}, nil
}

type sampleAggregate struct {
	rssMedian             uint64
	rssMin                uint64
	rssMax                uint64
	goHeapMedian          uint64
	compileErrorObserved  bool
	compileError          string
	compileErrorSampleNum int
}

func aggregateSamples(samples []sampleSummary) sampleAggregate {
	var rssValues []uint64
	var heapValues []uint64
	agg := sampleAggregate{}
	for _, sample := range samples {
		if sample.RSSPeakBytes > 0 {
			rssValues = append(rssValues, sample.RSSPeakBytes)
		}
		if sample.GoHeapPeakBytes > 0 {
			heapValues = append(heapValues, sample.GoHeapPeakBytes)
		}
		if sample.CompileErrorObserved && !agg.compileErrorObserved {
			agg.compileErrorObserved = true
			agg.compileError = sample.CompileError
			agg.compileErrorSampleNum = sample.Index
		}
	}
	if len(rssValues) > 0 {
		sort.Slice(rssValues, func(i, j int) bool { return rssValues[i] < rssValues[j] })
		agg.rssMedian = medianUint64(rssValues)
		agg.rssMin = rssValues[0]
		agg.rssMax = rssValues[len(rssValues)-1]
	}
	if len(heapValues) > 0 {
		sort.Slice(heapValues, func(i, j int) bool { return heapValues[i] < heapValues[j] })
		agg.goHeapMedian = medianUint64(heapValues)
	}
	return agg
}

func medianUint64(values []uint64) uint64 {
	if len(values) == 0 {
		return 0
	}
	mid := len(values) / 2
	if len(values)%2 == 1 {
		return values[mid]
	}
	return (values[mid-1] / 2) + (values[mid] / 2) + ((values[mid-1]%2)+(values[mid]%2))/2
}

type reportComparisonKey struct {
	sourcePath        string
	sourcePathsKey    string
	moduleCount       int
	jobs              int
	warmCache         bool
	memoryBudgetBytes int64
}

func buildReportComparisons(summaries []scenarioSummary) []reportComparison {
	type pair struct {
		off *scenarioSummary
		on  *scenarioSummary
	}
	pairs := map[reportComparisonKey]pair{}
	for i := range summaries {
		summary := &summaries[i]
		if summary.ExpectCompileError {
			continue
		}
		key := reportComparisonKey{
			sourcePath:        summary.SourcePath,
			sourcePathsKey:    strings.Join(summary.SourcePaths, "\n"),
			moduleCount:       summary.ModuleCount,
			jobs:              summary.Jobs,
			warmCache:         summary.WarmCache,
			memoryBudgetBytes: summary.MemoryBudgetBytes,
		}
		p := pairs[key]
		if summary.Reports {
			p.on = summary
		} else {
			p.off = summary
		}
		pairs[key] = p
	}

	var keys []reportComparisonKey
	for key, p := range pairs {
		if p.off != nil && p.on != nil {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].sourcePath != keys[j].sourcePath {
			return keys[i].sourcePath < keys[j].sourcePath
		}
		if keys[i].sourcePathsKey != keys[j].sourcePathsKey {
			return keys[i].sourcePathsKey < keys[j].sourcePathsKey
		}
		if keys[i].moduleCount != keys[j].moduleCount {
			return keys[i].moduleCount < keys[j].moduleCount
		}
		if keys[i].jobs != keys[j].jobs {
			return keys[i].jobs < keys[j].jobs
		}
		if keys[i].warmCache != keys[j].warmCache {
			return !keys[i].warmCache && keys[j].warmCache
		}
		return keys[i].memoryBudgetBytes < keys[j].memoryBudgetBytes
	})

	var out []reportComparison
	for _, key := range keys {
		p := pairs[key]
		out = append(out, evaluateReportComparison(*p.off, *p.on))
	}
	return out
}

func evaluateReportComparison(off scenarioSummary, on scenarioSummary) reportComparison {
	bound := off.RSSMedianBytes + off.RSSDispersionBytes + on.RSSDispersionBytes
	status := "pass"
	if off.SampleCount < 2 || on.SampleCount < 2 {
		status = "insufficient_samples"
	} else if on.RSSMedianBytes > bound {
		status = "fail"
	}
	return reportComparison{
		Name:                  reportComparisonName(off.Name, on.Name),
		SourcePath:            on.SourcePath,
		SourcePaths:           append([]string(nil), on.SourcePaths...),
		ReportOffScenario:     off.Name,
		ReportOnScenario:      on.Name,
		ModuleCount:           on.ModuleCount,
		Jobs:                  on.Jobs,
		WarmCache:             on.WarmCache,
		MemoryBudgetBytes:     on.MemoryBudgetBytes,
		ReportOffSampleCount:  off.SampleCount,
		ReportOnSampleCount:   on.SampleCount,
		ReportOffRSSMedian:    off.RSSMedianBytes,
		ReportOnRSSMedian:     on.RSSMedianBytes,
		ReportOffDispersion:   off.RSSDispersionBytes,
		ReportOnDispersion:    on.RSSDispersionBytes,
		BoundRSSBytes:         bound,
		DeltaBytes:            int64(on.RSSMedianBytes) - int64(off.RSSMedianBytes),
		Ratio:                 roundedRatio(on.RSSMedianBytes, off.RSSMedianBytes),
		EvaluationStatus:      status,
		BoundBasis:            "report_off_median_plus_observed_off_on_dispersion",
		SameHostSameConfig:    true,
		ReportOffRSSPeakPhase: off.RSSPeakPhase,
		ReportOnRSSPeakPhase:  on.RSSPeakPhase,
	}
}

func reportComparisonName(offName string, onName string) string {
	name := strings.TrimSpace(onName)
	if name == "" {
		name = strings.TrimSpace(offName)
	}
	name = strings.Replace(name, "_reports_on_", "_reports_", 1)
	name = strings.Replace(name, "_reports_off_", "_reports_", 1)
	return name
}

func roundedRatio(numerator uint64, denominator uint64) float64 {
	if denominator == 0 {
		return 0
	}
	ratio := float64(numerator) / float64(denominator)
	return float64(uint64(ratio*10000+0.5)) / 10000
}

func runScenarioSample(
	dir string,
	scenarioName string,
	repoRoot string,
	scenario Scenario,
	sampleIndex int,
	memoryReset func(),
) (sampleSummary, error) {
	sampleName := fmt.Sprintf("sample-%02d", sampleIndex)
	sampleDir := filepath.Join(dir, "samples", sampleName)
	srcRoot := filepath.Join(sampleDir, "src")
	sources, roots, err := prepareScenarioSources(srcRoot, repoRoot, scenario)
	if err != nil {
		return sampleSummary{}, err
	}
	if !scenario.WarmCache {
		if err := os.RemoveAll(filepath.Join(srcRoot, ".tetra_cache")); err != nil {
			return sampleSummary{}, err
		}
	} else {
		if scenario.ExpectCompileError {
			if err := writeScenarioMain(srcRoot, scenario.ModuleCount, false); err != nil {
				return sampleSummary{}, err
			}
		}
		for i, source := range sources {
			outPath := sampleOutputPath(sampleDir, len(sources), i, source.RelPath, "warmup-app")
			if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
				return sampleSummary{}, err
			}
			if _, err := buildScenario(source.Entry, outPath, "", scenario, roots); err != nil {
				return sampleSummary{}, fmt.Errorf("warm cache build %s %s %s: %w", scenarioName, sampleName, source.RelPath, err)
			}
		}
		if scenario.ExpectCompileError {
			if err := writeScenarioMain(srcRoot, scenario.ModuleCount, true); err != nil {
				return sampleSummary{}, err
			}
		}
	}
	if memoryReset != nil {
		memoryReset()
	}

	summary := sampleSummary{
		Index:       sampleIndex,
		SourcePaths: normalizedScenarioSourcePaths(scenario),
	}
	var executableHashes []string
	var cacheHits []string
	var compiledModules []string
	var loweredModules []string
	for i, source := range sources {
		outPath := sampleOutputPath(sampleDir, len(sources), i, source.RelPath, "app")
		profilePath := sampleProfilePath(sampleDir, len(sources), i, source.RelPath)
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return sampleSummary{}, err
		}
		if err := os.MkdirAll(filepath.Dir(profilePath), 0o755); err != nil {
			return sampleSummary{}, err
		}

		stats, err := buildScenario(source.Entry, outPath, profilePath, scenario, roots)
		compileErr := err
		if compileErr != nil && !scenario.ExpectCompileError {
			return sampleSummary{}, compileErr
		}
		if compileErr == nil && scenario.ExpectCompileError {
			return sampleSummary{}, fmt.Errorf("%s %s compiled successfully; expected compile error", scenarioName, sampleName)
		}
		profile, err := readProfile(profilePath, scenario.ExpectCompileError)
		if err != nil {
			return sampleSummary{}, err
		}
		profileRel := filepath.ToSlash(filepath.Join("scenarios", scenarioName, "samples", sampleName, relPathFromSample(sampleDir, profilePath)))
		summary.CompilerProfiles = append(summary.CompilerProfiles, profileRel)
		if profile.RSSPeakBytes >= summary.RSSPeakBytes {
			summary.RSSPeakBytes = profile.RSSPeakBytes
			summary.RSSPeakPhase = peakPhaseName(profile)
			summary.WorkerCount = profile.WorkerCount
			summary.WorkerReason = profile.WorkerReason
			summary.ReportMode = profile.ReportMode
			summary.PhaseProfile = profileRel
		}
		if profile.GoHeapPeakAllocBytes > summary.GoHeapPeakBytes {
			summary.GoHeapPeakBytes = profile.GoHeapPeakAllocBytes
		}
		if compileErr != nil {
			summary.CompileErrorObserved = true
			summary.CompileError = compileErr.Error()
			continue
		}
		executableHash, err := fileSHA256(outPath)
		if err != nil {
			return sampleSummary{}, err
		}
		executableRel := filepath.ToSlash(filepath.Join("scenarios", scenarioName, "samples", sampleName, relPathFromSample(sampleDir, outPath)))
		summary.Executables = append(summary.Executables, sampleExecutable{
			SourcePath:       source.RelPath,
			PhaseProfile:     profileRel,
			Executable:       executableRel,
			ExecutableSHA256: executableHash,
		})
		executableHashes = append(executableHashes, executableHash)
		if summary.Executable == "" {
			summary.Executable = executableRel
		}
		if stats != nil {
			cacheHits = append(cacheHits, stats.CacheHits...)
			compiledModules = append(compiledModules, stats.CompiledModules...)
			loweredModules = append(loweredModules, stats.LoweredModules...)
		}
	}
	if err := os.RemoveAll(filepath.Join(srcRoot, ".tetra_cache")); err != nil {
		return sampleSummary{}, err
	}
	summary.CompilerProfileCount = len(summary.CompilerProfiles)
	summary.ExecutableCount = len(summary.Executables)
	if len(executableHashes) == 1 {
		summary.ExecutableSHA256 = executableHashes[0]
	} else if len(executableHashes) > 1 {
		summary.ExecutableSHA256 = aggregateHashes(executableHashes)
	}
	if len(cacheHits) > 0 {
		summary.CacheHits = uniqueSortedStrings(cacheHits)
	}
	if len(compiledModules) > 0 {
		summary.CompiledModules = uniqueSortedStrings(compiledModules)
	}
	if len(loweredModules) > 0 {
		summary.LoweredModules = uniqueSortedStrings(loweredModules)
	}
	return summary, nil
}

type preparedScenarioSource struct {
	RelPath string
	Entry   string
}

type scenarioBuildRoots struct {
	ProjectRoot     string
	DependencyRoots []compiler.ModuleRoot
}

func prepareScenarioSources(srcRoot string, repoRoot string, scenario Scenario) ([]preparedScenarioSource, scenarioBuildRoots, error) {
	if strings.TrimSpace(scenario.SourcePath) != "" && len(scenario.SourcePaths) > 0 {
		return nil, scenarioBuildRoots{}, fmt.Errorf("scenario %q cannot set both source_path and source_paths", scenario.Name)
	}
	if strings.TrimSpace(scenario.SourcePath) == "" && len(scenario.SourcePaths) == 0 {
		if err := writeScenarioSources(srcRoot, scenario.ModuleCount, scenario.ExpectCompileError); err != nil {
			return nil, scenarioBuildRoots{}, err
		}
		return []preparedScenarioSource{{
			RelPath: "app/main.t4",
			Entry:   filepath.Join(srcRoot, "app", "main.t4"),
		}}, scenarioBuildRoots{}, nil
	}
	if scenario.ExpectCompileError {
		return nil, scenarioBuildRoots{}, fmt.Errorf("repo source scenario %q cannot expect compile errors", scenario.Name)
	}
	resolvedRepoRoot, err := resolveRepoRoot(repoRoot)
	if err != nil {
		return nil, scenarioBuildRoots{}, err
	}
	var paths []string
	if strings.TrimSpace(scenario.SourcePath) != "" {
		paths = []string{scenario.SourcePath}
	} else {
		paths = append([]string(nil), scenario.SourcePaths...)
	}
	var sources []preparedScenarioSource
	seen := map[string]struct{}{}
	for _, path := range paths {
		sourceRel, err := cleanRepresentativeSourcePath(path)
		if err != nil {
			return nil, scenarioBuildRoots{}, err
		}
		if _, ok := seen[sourceRel]; ok {
			return nil, scenarioBuildRoots{}, fmt.Errorf("scenario %q has duplicate source path %q", scenario.Name, sourceRel)
		}
		seen[sourceRel] = struct{}{}
		srcPath := filepath.Join(resolvedRepoRoot, filepath.FromSlash(sourceRel))
		raw, err := os.ReadFile(srcPath)
		if err != nil {
			return nil, scenarioBuildRoots{}, fmt.Errorf("read repo source %s: %w", sourceRel, err)
		}
		entry := filepath.Join(srcRoot, filepath.FromSlash(sourceRel))
		if err := writeFile(entry, raw); err != nil {
			return nil, scenarioBuildRoots{}, err
		}
		sources = append(sources, preparedScenarioSource{RelPath: sourceRel, Entry: entry})
	}
	return sources, scenarioBuildRoots{
		ProjectRoot:     srcRoot,
		DependencyRoots: []compiler.ModuleRoot{{Root: resolvedRepoRoot}},
	}, nil
}

func cleanRepresentativeSourcePath(path string) (string, error) {
	path = strings.TrimSpace(filepath.ToSlash(path))
	if path == "" {
		return "", fmt.Errorf("representative source path is required")
	}
	if filepath.IsAbs(path) || strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("representative source path %q must be repo-relative", path)
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." || strings.HasPrefix(clean, "../") || clean == ".." {
		return "", fmt.Errorf("representative source path %q must stay inside the repo", path)
	}
	if ext := filepath.Ext(clean); ext != ".t4" && ext != ".tetra" {
		return "", fmt.Errorf("representative source path %q must be a .t4 or .tetra file", path)
	}
	return clean, nil
}

func normalizedScenarioSourcePaths(scenario Scenario) []string {
	var paths []string
	if strings.TrimSpace(scenario.SourcePath) != "" {
		paths = append(paths, strings.TrimSpace(filepath.ToSlash(scenario.SourcePath)))
	}
	for _, path := range scenario.SourcePaths {
		path = strings.TrimSpace(filepath.ToSlash(path))
		if path != "" {
			paths = append(paths, path)
		}
	}
	return append([]string(nil), paths...)
}

func sampleProfilePath(sampleDir string, sourceCount int, index int, sourcePath string) string {
	if sourceCount <= 1 {
		return filepath.Join(sampleDir, "compiler-profile.json")
	}
	return filepath.Join(sampleDir, "compiler-profiles", sampleArtifactStem(index, sourcePath)+".json")
}

func sampleOutputPath(sampleDir string, sourceCount int, index int, sourcePath string, defaultName string) string {
	if sourceCount <= 1 {
		return filepath.Join(sampleDir, "out", defaultName)
	}
	return filepath.Join(sampleDir, "out", sampleArtifactStem(index, sourcePath))
}

func sampleArtifactStem(index int, sourcePath string) string {
	stem := strings.TrimSuffix(filepath.ToSlash(sourcePath), filepath.Ext(sourcePath))
	var b strings.Builder
	for _, r := range stem {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			if b.Len() > 0 {
				b.WriteByte('_')
			}
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		out = "source"
	}
	if len(out) > 96 {
		out = out[len(out)-96:]
		out = strings.TrimLeft(out, "_")
		if out == "" {
			out = "source"
		}
	}
	return fmt.Sprintf("%02d-%s", index+1, out)
}

func relPathFromSample(sampleDir string, path string) string {
	rel, err := filepath.Rel(sampleDir, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func aggregateHashes(hashes []string) string {
	h := sha256.New()
	for _, hash := range hashes {
		h.Write([]byte(hash))
		h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func uniqueSortedStrings(items []string) []string {
	seen := map[string]struct{}{}
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		seen[item] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for item := range seen {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func resolveRepoRoot(root string) (string, error) {
	if strings.TrimSpace(root) != "" {
		abs, err := filepath.Abs(root)
		if err != nil {
			return "", err
		}
		if info, err := os.Stat(abs); err != nil {
			return "", fmt.Errorf("stat repo root %s: %w", abs, err)
		} else if !info.IsDir() {
			return "", fmt.Errorf("repo root %s is not a directory", abs)
		}
		return abs, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if fileExists(filepath.Join(dir, "go.mod")) && dirExists(filepath.Join(dir, "compiler")) {
			return dir, nil
		}
		next := filepath.Dir(dir)
		if next == dir {
			return "", fmt.Errorf("could not resolve repo root from %s", dir)
		}
		dir = next
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func buildScenario(
	entry string,
	outPath string,
	profilePath string,
	scenario Scenario,
	roots scenarioBuildRoots,
) (*compiler.BuildStats, error) {
	opt := compiler.BuildOptions{
		Jobs:                    scenario.Jobs,
		MemoryBudgetBytes:       scenario.MemoryBudgetBytes,
		EmitCompilerPhaseReport: strings.TrimSpace(profilePath) != "",
		CompilerPhaseReportPath: profilePath,
	}
	if strings.TrimSpace(roots.ProjectRoot) != "" {
		opt.ProjectRoot = roots.ProjectRoot
	}
	if len(roots.DependencyRoots) > 0 {
		opt.DependencyRoots = append([]compiler.ModuleRoot(nil), roots.DependencyRoots...)
	}
	if scenario.Reports {
		opt.EmitProof = true
		opt.EmitBoundsReport = true
		opt.EmitAllocReport = true
		opt.EmitMemoryReport = true
	}
	stats, err := compiler.BuildFileWithStatsOpt(entry, outPath, "linux-x64", opt)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func writeScenarioSources(root string, moduleCount int, expectCompileError bool) error {
	for i := 0; i < moduleCount; i++ {
		var src string
		if i == 0 {
			src = "module bench.m0\nfun value_0(x: i32): i32 {\n  return x + 1\n}\n"
		} else {
			src = fmt.Sprintf(
				"module bench.m%d\nimport bench.m%d as prev\nfun value_%d(x: i32): i32 {\n  return prev.value_%d(x) + 1\n}\n",
				i,
				i-1,
				i,
				i-1,
			)
		}
		if err := writeFile(filepath.Join(root, "bench", fmt.Sprintf("m%d.t4", i)), []byte(src)); err != nil {
			return err
		}
	}
	return writeScenarioMain(root, moduleCount, expectCompileError)
}

func writeScenarioMain(root string, moduleCount int, expectCompileError bool) error {
	mainSrc := fmt.Sprintf(
		"module app.main\nimport bench.m%d as last\nfun main(): i32 {\n  return last.value_%d(0)\n}\n",
		moduleCount-1,
		moduleCount-1,
	)
	if expectCompileError {
		mainSrc = fmt.Sprintf(
			"module app.main\nimport bench.m%d as last\nfun main(): i32 {\n  return last.missing_value_%d(0)\n}\n",
			moduleCount-1,
			moduleCount-1,
		)
	}
	return writeFile(filepath.Join(root, "app", "main.t4"), []byte(mainSrc))
}

func readProfile(path string, allowPartial bool) (compilerPhaseProfile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return compilerPhaseProfile{}, err
	}
	var profile compilerPhaseProfile
	if err := json.Unmarshal(raw, &profile); err != nil {
		return compilerPhaseProfile{}, err
	}
	if profile.Schema != "tetra.compiler.phase-profile.v1" {
		return compilerPhaseProfile{}, fmt.Errorf("compiler profile schema = %q", profile.Schema)
	}
	if !profile.RSSSupported || profile.RSSPeakBytes == 0 {
		return compilerPhaseProfile{}, fmt.Errorf("compiler profile missing supported RSS peak")
	}
	if !allowPartial && profile.WorkerCount <= 0 {
		return compilerPhaseProfile{}, fmt.Errorf("compiler profile worker_count = %d", profile.WorkerCount)
	}
	if strings.TrimSpace(profile.WorkerReason) == "" {
		return compilerPhaseProfile{}, fmt.Errorf("compiler profile missing worker_reason")
	}
	return profile, nil
}

func peakPhaseName(profile compilerPhaseProfile) string {
	peakPhase := ""
	var peakPhaseRSS uint64
	for _, phase := range profile.Phases {
		if phase.RSSCurrentBytes > peakPhaseRSS {
			peakPhaseRSS = phase.RSSCurrentBytes
			peakPhase = phase.Name
		}
	}
	return peakPhase
}

func validateBundleOutput(summaries []scenarioSummary, targetScope targetScopeReport) string {
	issues := validateSummaryIssues(summaries)
	issues = append(issues, validateTargetScopeIssues(targetScope)...)
	return formatCompilerRSSValidatorOutput(
		[]string{SummarySchema, TargetScopeSchema},
		issues,
	)
}

func validateSummaryOutput(summaries []scenarioSummary) string {
	return formatCompilerRSSValidatorOutput(
		[]string{SummarySchema},
		validateSummaryIssues(summaries),
	)
}

func validateSummaryIssues(summaries []scenarioSummary) []string {
	var issues []string
	if len(summaries) == 0 {
		issues = append(issues, "no scenarios")
	}
	for _, summary := range summaries {
		if strings.TrimSpace(summary.PhaseProfile) == "" {
			issues = append(issues, summary.Name+": missing phase profile")
		}
		if summary.SourceCount > 0 && len(summary.SourcePaths) != summary.SourceCount {
			issues = append(issues, summary.Name+": invalid source path count")
		}
		if summary.RSSPeakBytes == 0 {
			issues = append(issues, summary.Name+": missing RSS peak")
		}
		if summary.SampleCount <= 0 || len(summary.Samples) != summary.SampleCount {
			issues = append(issues, summary.Name+": invalid sample count")
		}
		if summary.RSSMedianBytes == 0 || summary.RSSMinBytes == 0 || summary.RSSMaxBytes == 0 {
			issues = append(issues, summary.Name+": missing RSS sample statistics")
		}
		if summary.RSSMaxBytes < summary.RSSMinBytes ||
			summary.RSSDispersionBytes != summary.RSSMaxBytes-summary.RSSMinBytes {
			issues = append(issues, summary.Name+": invalid RSS dispersion")
		}
		if !summary.ExpectCompileError && summary.WorkerCount <= 0 {
			issues = append(issues, summary.Name+": missing worker count")
		}
		if summary.ExpectCompileError {
			if !summary.CompileErrorObserved || strings.TrimSpace(summary.CompileError) == "" {
				issues = append(issues, summary.Name+": expected compile error was not observed")
			}
			if summary.ExecutableSHA256 != "" {
				issues = append(issues, summary.Name+": compile-error scenario has executable hash")
			}
			continue
		}
		if summary.CompileErrorObserved {
			issues = append(issues, summary.Name+": unexpected compile error")
		}
		if summary.SourceCount > 0 {
			if summary.ExecutableCount != summary.SourceCount {
				issues = append(issues, summary.Name+": executable count does not match source count")
			}
			if summary.CompilerProfileCount != summary.SourceCount {
				issues = append(issues, summary.Name+": compiler profile count does not match source count")
			}
		}
		if len(summary.ExecutableSHA256) != 64 {
			issues = append(issues, summary.Name+": invalid executable sha256")
		}
	}
	return issues
}

func validateTargetScopeIssues(targetScope targetScopeReport) []string {
	var issues []string
	if targetScope.Schema != TargetScopeSchema {
		issues = append(issues, "target_scope: schema = "+targetScope.Schema)
	}
	if strings.TrimSpace(targetScope.HostTarget) == "" {
		issues = append(issues, "target_scope: missing host_target")
	}
	if targetScope.CompilerTarget != "linux-x64" {
		issues = append(issues, "target_scope: compiler_target must be linux-x64")
	}

	supported := map[string]targetScopeTarget{}
	for _, target := range targetScope.SupportedTargets {
		if strings.TrimSpace(target.Target) == "" {
			issues = append(issues, "target_scope: supported target missing name")
			continue
		}
		if _, exists := supported[target.Target]; exists {
			issues = append(issues, "target_scope: duplicate supported target "+target.Target)
		}
		supported[target.Target] = target
		if strings.TrimSpace(target.Reason) == "" {
			issues = append(issues, target.Target+" "+target.Status+" missing reason")
		}
		if target.Status == "host_rss_measured" {
			if target.Target != "linux-x64" {
				issues = append(issues, "unsupported host RSS claim for "+target.Target)
			}
			if targetScope.HostTarget != "linux/amd64" {
				issues = append(issues, "host_rss_measured requires linux/amd64 host")
			}
		}
	}
	if linux, ok := supported["linux-x64"]; !ok {
		issues = append(issues, "target_scope: missing supported linux-x64 target")
	} else if linux.Status != "host_rss_measured" {
		issues = append(issues, "target_scope: linux-x64 status must be host_rss_measured")
	}

	nonClaims := map[string]targetScopeTarget{}
	for _, target := range targetScope.NonClaimTargets {
		if strings.TrimSpace(target.Target) == "" {
			issues = append(issues, "target_scope: non_claim target missing name")
			continue
		}
		if _, exists := nonClaims[target.Target]; exists {
			issues = append(issues, "target_scope: duplicate non_claim target "+target.Target)
		}
		nonClaims[target.Target] = target
		if target.Status != "non_claim" {
			issues = append(issues, target.Target+" status must be non_claim")
		}
		if strings.TrimSpace(target.Reason) == "" {
			issues = append(issues, target.Target+" non_claim missing reason")
		}
		if _, claimed := supported[target.Target]; claimed {
			issues = append(issues, target.Target+" cannot be both supported and non_claim")
		}
	}
	for _, required := range requiredTargetScopeNonClaims() {
		if _, ok := nonClaims[required]; !ok {
			issues = append(issues, "missing non_claim target "+required)
		}
	}
	return issues
}

func requiredTargetScopeNonClaims() []string {
	return []string{
		"windows-x64",
		"macos-x64",
		"macos-arm64",
		"linux-x86",
		"linux-x32",
		"wasm32-wasi",
		"wasm32-web",
	}
}

func formatCompilerRSSValidatorOutput(schemas []string, issues []string) string {
	if len(issues) > 0 {
		return "validator: ramcompilerrss.Run\nschemas:\n- " + strings.Join(schemas, "\n- ") +
			"\nresult: fail\nissues:\n- " + strings.Join(issues, "\n- ") + "\n"
	}
	return "validator: ramcompilerrss.Run\nschemas:\n- " + strings.Join(schemas, "\n- ") +
		"\nresult: pass\n"
}

func safeScenarioName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("scenario name is required")
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return "", fmt.Errorf("unsafe scenario name %q", name)
	}
	if strings.Contains(name, "..") || filepath.IsAbs(name) {
		return "", fmt.Errorf("unsafe scenario name %q", name)
	}
	return name, nil
}

func readHostFingerprint() hostFingerprint {
	hostname, _ := os.Hostname()
	return hostFingerprint{
		Schema:        "tetra.ram.p7-compiler-rss.host.v1",
		Hostname:      hostname,
		GOOS:          runtime.GOOS,
		GOARCH:        runtime.GOARCH,
		NumCPU:        runtime.NumCPU(),
		KernelRelease: strings.TrimSpace(readOptionalFile("/proc/sys/kernel/osrelease")),
		OSRelease:     osReleaseName(readOptionalFile("/etc/os-release")),
		CPUModel:      cpuModelName(readOptionalFile("/proc/cpuinfo")),
	}
}

func readTargetScope() targetScopeReport {
	return targetScopeReport{
		Schema:         TargetScopeSchema,
		HostTarget:     runtime.GOOS + "/" + runtime.GOARCH,
		CompilerTarget: "linux-x64",
		SupportedTargets: []targetScopeTarget{
			{
				Target: "linux-x64",
				Status: "host_rss_measured",
				Reason: "compiler-process phase profiles record process RSS on linux/amd64 hosts for the linux-x64 compiler target",
			},
		},
		NonClaimTargets: []targetScopeTarget{
			{
				Target: "windows-x64",
				Status: "non_claim",
				Reason: "not measured by this linux/amd64 process RSS harness; Windows reserve/commit/release semantics need target-specific evidence",
			},
			{
				Target: "macos-x64",
				Status: "non_claim",
				Reason: "not measured by this linux/amd64 process RSS harness; macOS RSS/release semantics need target-specific evidence",
			},
			{
				Target: "macos-arm64",
				Status: "non_claim",
				Reason: "not measured by this linux/amd64 process RSS harness; macOS arm64 RSS/release semantics need target-specific evidence",
			},
			{
				Target: "linux-x86",
				Status: "non_claim",
				Reason: "not measured by this linux/amd64 process RSS harness; 32-bit size arithmetic and release semantics need target-specific evidence",
			},
			{
				Target: "linux-x32",
				Status: "non_claim",
				Reason: "not measured by this linux/amd64 process RSS harness; x32 size arithmetic and release semantics need target-specific evidence",
			},
			{
				Target: "wasm32-wasi",
				Status: "non_claim",
				Reason: "not measured by this linux/amd64 process RSS harness; WASM linear-memory high-water semantics need target-specific evidence",
			},
			{
				Target: "wasm32-web",
				Status: "non_claim",
				Reason: "not measured by this linux/amd64 process RSS harness; browser/WASM linear-memory semantics need target-specific evidence",
			},
		},
		Notes: []string{
			"Linux-only compiler RSS evidence must not be used as a cross-target memory lifecycle claim.",
			"Targets listed as non_claim may still have build-only compiler support outside this RSS evidence bundle.",
		},
	}
}

func osReleaseName(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
		}
	}
	return ""
}

func cpuModelName(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

func readOptionalFile(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(raw)
}

func splitNonEmptyLines(raw string) []string {
	var lines []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func fileSHA256(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(file)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
