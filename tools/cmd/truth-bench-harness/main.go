package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	schemaV1                    = "tetra.truth.benchmark.v1"
	scopeP8Full                 = "p8_full"
	scopeP19GenericCollections  = "p19.1_generic_collections"
	scopeP19HTTPJSONSourceFirst = "p19.2_http_json_source_first"
	scopeP19PostgresSourceFirst = "p19.3_postgres_source_first"
	scopeP20BenchmarkMatrix     = "p20.0_benchmark_matrix"
	scopeP15ActorBenchmarkPrep  = "p15_actor_benchmark_prep"
	p19GenericCollectionsAlgoID = ("p19.1.generic_collections.hash_table.parallel_slice_" +
		"linear_lookup_i32")
)

type Manifest struct {
	Scope      string          `json:"scope,omitempty"`
	Benchmarks []BenchmarkSpec `json:"benchmarks"`
}

type BenchmarkSpec struct {
	Name                   string   `json:"name"`
	Category               string   `json:"category"`
	Language               string   `json:"language"`
	CompilerVersion        string   `json:"compiler_version"`
	AlgorithmID            string   `json:"algorithm_id,omitempty"`
	InputDescription       string   `json:"input_description,omitempty"`
	BuildCommand           []string `json:"build_command"`
	RunCommand             []string `json:"run_command"`
	Binary                 string   `json:"binary,omitempty"`
	TetraProofReports      []string `json:"tetra_proof_reports,omitempty"`
	TetraAllocationReports []string `json:"tetra_allocation_reports,omitempty"`
	TetraBoundsReports     []string `json:"tetra_bounds_reports,omitempty"`
	TetraReports           []string `json:"tetra_reports,omitempty"`
	RawOutputArtifacts     []string `json:"raw_output_artifacts,omitempty"`
}

type Report struct {
	Schema     string            `json:"schema"`
	Scope      string            `json:"scope,omitempty"`
	Generated  string            `json:"generated"`
	Host       HostInfo          `json:"host"`
	Benchmarks []BenchmarkResult `json:"benchmarks"`
	Claims     []string          `json:"claims"`
}

type HostInfo struct {
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
	CPUs      int    `json:"cpus"`
	TargetCPU string `json:"target_cpu"`
}

type BenchmarkResult struct {
	Name                   string           `json:"name"`
	Category               string           `json:"category"`
	Language               string           `json:"language"`
	CompilerVersion        string           `json:"compiler_version"`
	TargetCPU              string           `json:"target_cpu"`
	AlgorithmID            string           `json:"algorithm_id,omitempty"`
	InputDescription       string           `json:"input_description,omitempty"`
	BuildCommand           string           `json:"build_command"`
	RunCommand             string           `json:"run_command"`
	Ran                    bool             `json:"ran"`
	BuildExitCode          *int             `json:"build_exit_code,omitempty"`
	RunExitCode            *int             `json:"run_exit_code,omitempty"`
	RuntimeMS              int64            `json:"runtime_ms"`
	BinarySizeBytes        int64            `json:"binary_size_bytes"`
	TetraProofReports      []ReportArtifact `json:"tetra_proof_reports,omitempty"`
	TetraAllocationReports []ReportArtifact `json:"tetra_allocation_reports,omitempty"`
	TetraBoundsReports     []ReportArtifact `json:"tetra_bounds_reports,omitempty"`
	TetraReports           []ReportArtifact `json:"tetra_reports,omitempty"`
	RawOutputArtifacts     []ReportArtifact `json:"raw_output_artifacts,omitempty"`
}

type ReportArtifact struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

func main() {
	manifestPath := flag.String("manifest", "", "benchmark manifest JSON")
	outPath := flag.String("out", "", "output report JSON")
	claimTiersOutPath := flag.String(
		"claim-tiers-out",
		"",
		"write the P20.2 claim-tier policy report JSON",
	)
	run := flag.Bool("run", false, "run build and benchmark commands instead of dry-run reporting")
	timeout := flag.Duration("timeout", 30*time.Second, "per-command timeout")
	flag.Parse()
	if *claimTiersOutPath != "" {
		report := BuildP20ClaimTierReport()
		if err := ValidateClaimTierReport(report); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		raw, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		raw = append(raw, '\n')
		if err := os.WriteFile(*claimTiersOutPath, raw, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *manifestPath == "" || *outPath == "" {
		fmt.Fprintln(
			os.Stderr,
			"usage: truth-bench-harness --manifest manifest.json --out report.json [--run]",
		)
		os.Exit(2)
	}
	manifest, err := readManifest(*manifestPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	report, err := buildReport(manifest, *run, *timeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateReport(report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(*outPath, raw, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func readManifest(path string) (Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, validateManifest(manifest)
}

func validateManifest(manifest Manifest) error {
	if len(manifest.Benchmarks) == 0 {
		return fmt.Errorf("benchmark manifest must contain at least one benchmark")
	}
	policy, err := policyForBenchmarkScope(manifest.Scope)
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	equivalence := map[string]struct {
		algorithm string
		input     string
	}{}
	for i, bench := range manifest.Benchmarks {
		if bench.Name == "" {
			return fmt.Errorf("benchmark %d missing name", i)
		}
		if !policyCategory(policy, bench.Category) {
			return fmt.Errorf(
				"benchmark %s has unsupported category %q for scope %q",
				bench.Name,
				bench.Category,
				policy.Scope,
			)
		}
		if !policyLanguage(policy, bench.Language) {
			return fmt.Errorf(
				"benchmark %s has unsupported language %q for scope %q",
				bench.Name,
				bench.Language,
				policy.Scope,
			)
		}
		key := bench.Category + "\x00" + bench.Language
		if seen[key] {
			return fmt.Errorf(
				"duplicate benchmark matrix row for category %q language %q",
				bench.Category,
				bench.Language,
			)
		}
		seen[key] = true
		if policy.RequireEquivalenceMetadata {
			if strings.TrimSpace(bench.AlgorithmID) == "" {
				return fmt.Errorf(
					"benchmark %s missing algorithm_id for scope %q",
					bench.Name,
					policy.Scope,
				)
			}
			if strings.TrimSpace(bench.InputDescription) == "" {
				return fmt.Errorf(
					"benchmark %s missing input_description for scope %q",
					bench.Name,
					policy.Scope,
				)
			}
			if prev, ok := equivalence[bench.Category]; ok {
				if bench.AlgorithmID != prev.algorithm {
					return fmt.Errorf(
						"benchmark %s algorithm_id = %q, want equivalent %q for category %q",
						bench.Name,
						bench.AlgorithmID,
						prev.algorithm,
						bench.Category,
					)
				}
				if bench.InputDescription != prev.input {
					return fmt.Errorf(
						"benchmark %s input_description differs from equivalent category %q",
						bench.Name,
						bench.Category,
					)
				}
			} else {
				equivalence[bench.Category] = struct {
					algorithm string
					input     string
				}{algorithm: bench.AlgorithmID, input: bench.InputDescription}
			}
		}
		if strings.TrimSpace(bench.CompilerVersion) == "" {
			return fmt.Errorf("benchmark %s missing compiler version", bench.Name)
		}
		if len(bench.BuildCommand) == 0 {
			return fmt.Errorf("benchmark %s missing build command", bench.Name)
		}
		if len(bench.RunCommand) == 0 {
			return fmt.Errorf("benchmark %s missing run command", bench.Name)
		}
		if policy.RequireRawOutputArtifacts && len(bench.RawOutputArtifacts) == 0 {
			return fmt.Errorf(
				"benchmark %s missing raw output artifact paths for scope %q",
				bench.Name,
				policy.Scope,
			)
		}
		if err := validateP8BuildCommand(bench); err != nil {
			return err
		}
		if bench.Language == "tetra" {
			if len(bench.TetraProofReports) == 0 || len(bench.TetraAllocationReports) == 0 ||
				len(bench.TetraBoundsReports) == 0 {
				return fmt.Errorf(
					"benchmark %s is Tetra and must list proof, allocation, and bounds report artifacts",
					bench.Name,
				)
			}
			if policy.RequireTetraReports && len(bench.TetraReports) == 0 {
				return fmt.Errorf(
					"benchmark %s is Tetra and must list performance report artifacts for scope %q",
					bench.Name,
					policy.Scope,
				)
			}
		}
	}
	for _, category := range policy.Categories {
		for _, language := range policy.Languages {
			key := category + "\x00" + language
			if !seen[key] {
				return fmt.Errorf(
					"missing benchmark matrix row for category %q language %q",
					category,
					language,
				)
			}
		}
	}
	return nil
}

func buildReport(manifest Manifest, run bool, timeout time.Duration) (Report, error) {
	if err := validateManifest(manifest); err != nil {
		return Report{}, err
	}
	report := Report{
		Schema:    schemaV1,
		Scope:     normalizeBenchmarkScope(manifest.Scope),
		Generated: time.Now().UTC().Format(time.RFC3339),
		Host: HostInfo{
			GOOS:      runtime.GOOS,
			GOARCH:    runtime.GOARCH,
			CPUs:      runtime.NumCPU(),
			TargetCPU: detectTargetCPU(),
		},
		Claims: []string{
			("No global performance claim is made by this harness; compare " +
				"only recorded benchmark rows under their captured commands and host."),
		},
	}
	for _, bench := range manifest.Benchmarks {
		row := BenchmarkResult{
			Name:                   bench.Name,
			Category:               bench.Category,
			Language:               bench.Language,
			CompilerVersion:        bench.CompilerVersion,
			TargetCPU:              report.Host.TargetCPU,
			AlgorithmID:            bench.AlgorithmID,
			InputDescription:       bench.InputDescription,
			BuildCommand:           commandString(bench.BuildCommand),
			RunCommand:             commandString(bench.RunCommand),
			TetraProofReports:      reportArtifacts(bench.TetraProofReports),
			TetraAllocationReports: reportArtifacts(bench.TetraAllocationReports),
			TetraBoundsReports:     reportArtifacts(bench.TetraBoundsReports),
			TetraReports:           reportArtifacts(bench.TetraReports),
			RawOutputArtifacts:     reportArtifacts(bench.RawOutputArtifacts),
		}
		if bench.Binary != "" {
			if info, err := os.Stat(bench.Binary); err == nil {
				row.BinarySizeBytes = info.Size()
			}
		}
		if run {
			row.Ran = true
			buildCode, err := runCommand(timeout, bench.BuildCommand)
			row.BuildExitCode = &buildCode
			if err != nil {
				report.Benchmarks = append(report.Benchmarks, row)
				return report, fmt.Errorf("benchmark %s build failed: %w", bench.Name, err)
			}
			start := time.Now()
			runCode, err := runCommand(timeout, bench.RunCommand)
			row.RunExitCode = &runCode
			row.RuntimeMS = time.Since(start).Milliseconds()
			if err != nil {
				report.Benchmarks = append(report.Benchmarks, row)
				return report, fmt.Errorf("benchmark %s run failed: %w", bench.Name, err)
			}
		}
		report.Benchmarks = append(report.Benchmarks, row)
	}
	return report, nil
}

func validateReport(report Report) error {
	if report.Schema != schemaV1 {
		return fmt.Errorf("unsupported schema %q", report.Schema)
	}
	policy, err := policyForBenchmarkScope(report.Scope)
	if err != nil {
		return err
	}
	if strings.TrimSpace(report.Generated) == "" {
		return fmt.Errorf("generated timestamp is required")
	}
	if report.Host.GOOS == "" || report.Host.GOARCH == "" || report.Host.CPUs <= 0 ||
		strings.TrimSpace(report.Host.TargetCPU) == "" {
		return fmt.Errorf("host metadata is incomplete: %+v", report.Host)
	}
	if err := validateClaims(report.Claims); err != nil {
		return err
	}
	seen := map[string]bool{}
	equivalence := map[string]struct {
		algorithm string
		input     string
	}{}
	for _, row := range report.Benchmarks {
		if row.Name == "" || !policyCategory(policy, row.Category) ||
			!policyLanguage(policy, row.Language) {
			return fmt.Errorf("invalid benchmark row identity: %+v", row)
		}
		key := row.Category + "\x00" + row.Language
		if seen[key] {
			return fmt.Errorf(
				"duplicate benchmark matrix row for category %q language %q",
				row.Category,
				row.Language,
			)
		}
		seen[key] = true
		if policy.RequireEquivalenceMetadata {
			if strings.TrimSpace(row.AlgorithmID) == "" {
				return fmt.Errorf(
					"benchmark %s missing algorithm_id for scope %q",
					row.Name,
					policy.Scope,
				)
			}
			if strings.TrimSpace(row.InputDescription) == "" {
				return fmt.Errorf(
					"benchmark %s missing input_description for scope %q",
					row.Name,
					policy.Scope,
				)
			}
			if prev, ok := equivalence[row.Category]; ok {
				if row.AlgorithmID != prev.algorithm {
					return fmt.Errorf(
						"benchmark %s algorithm_id = %q, want equivalent %q for category %q",
						row.Name,
						row.AlgorithmID,
						prev.algorithm,
						row.Category,
					)
				}
				if row.InputDescription != prev.input {
					return fmt.Errorf(
						"benchmark %s input_description differs from equivalent category %q",
						row.Name,
						row.Category,
					)
				}
			} else {
				equivalence[row.Category] = struct {
					algorithm string
					input     string
				}{algorithm: row.AlgorithmID, input: row.InputDescription}
			}
		}
		if strings.TrimSpace(row.CompilerVersion) == "" || strings.TrimSpace(row.TargetCPU) == "" {
			return fmt.Errorf("benchmark %s missing compiler version or target CPU", row.Name)
		}
		if row.TargetCPU != report.Host.TargetCPU {
			return fmt.Errorf(
				"benchmark %s target CPU = %q, want host target CPU %q",
				row.Name,
				row.TargetCPU,
				report.Host.TargetCPU,
			)
		}
		if strings.TrimSpace(row.BuildCommand) == "" || strings.TrimSpace(row.RunCommand) == "" {
			return fmt.Errorf("benchmark %s missing build or run command", row.Name)
		}
		if policy.RequireRawOutputArtifacts {
			if err := validateArtifactsExist(row.Name, "raw output", row.RawOutputArtifacts); err != nil {
				return err
			}
		}
		if row.RuntimeMS < 0 {
			return fmt.Errorf("benchmark %s has negative runtime", row.Name)
		}
		if row.BinarySizeBytes <= 0 {
			return fmt.Errorf("benchmark %s missing positive binary size", row.Name)
		}
		if row.Language == "tetra" {
			if err := validateArtifactsExist(row.Name, "proof", row.TetraProofReports); err != nil {
				return err
			}
			if err := validateArtifactsExist(
				row.Name,
				"allocation",
				row.TetraAllocationReports,
			); err != nil {
				return err
			}
			if err := validateArtifactsExist(row.Name, "bounds", row.TetraBoundsReports); err != nil {
				return err
			}
			if policy.RequireTetraReports {
				if err := validateArtifactsExist(row.Name, "performance", row.TetraReports); err != nil {
					return err
				}
			}
		}
	}
	for _, category := range policy.Categories {
		for _, language := range policy.Languages {
			key := category + "\x00" + language
			if !seen[key] {
				return fmt.Errorf(
					"missing benchmark matrix row for category %q language %q",
					category,
					language,
				)
			}
		}
	}
	return nil
}

func runCommand(timeout time.Duration, argv []string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return -1, ctx.Err()
	}
	if err == nil {
		return 0, nil
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return exit.ExitCode(), err
	}
	return -1, err
}

func reportArtifacts(paths []string) []ReportArtifact {
	out := make([]ReportArtifact, 0, len(paths))
	for _, path := range paths {
		_, err := os.Stat(path)
		out = append(out, ReportArtifact{Path: path, Exists: err == nil})
	}
	return out
}

func RequiredBenchmarkCategories() []string {
	return append([]string(nil), requiredBenchmarkCategories...)
}

func RequiredBenchmarkLanguages() []string {
	return append([]string(nil), requiredBenchmarkLanguages...)
}

type benchmarkMatrixPolicy struct {
	Scope                      string
	Categories                 []string
	Languages                  []string
	RequireEquivalenceMetadata bool
	RequireTetraReports        bool
	RequireRawOutputArtifacts  bool
}

func policyForBenchmarkScope(scope string) (benchmarkMatrixPolicy, error) {
	switch normalizeBenchmarkScope(scope) {
	case scopeP8Full:
		return benchmarkMatrixPolicy{
			Scope:      scopeP8Full,
			Categories: RequiredBenchmarkCategories(),
			Languages:  RequiredBenchmarkLanguages(),
		}, nil
	case scopeP19GenericCollections:
		return benchmarkMatrixPolicy{
			Scope:                      scopeP19GenericCollections,
			Categories:                 []string{"hash table"},
			Languages:                  []string{"tetra", "cpp", "rust"},
			RequireEquivalenceMetadata: true,
			RequireTetraReports:        true,
		}, nil
	case scopeP19HTTPJSONSourceFirst:
		return benchmarkMatrixPolicy{
			Scope:                      scopeP19HTTPJSONSourceFirst,
			Categories:                 []string{"HTTP plaintext", "HTTP JSON"},
			Languages:                  []string{"tetra"},
			RequireEquivalenceMetadata: true,
			RequireTetraReports:        true,
		}, nil
	case scopeP19PostgresSourceFirst:
		return benchmarkMatrixPolicy{
			Scope: scopeP19PostgresSourceFirst,
			Categories: []string{
				"DB single query",
				"DB multiple queries",
				"DB updates",
				"DB fortunes",
			},
			Languages:                  []string{"tetra"},
			RequireEquivalenceMetadata: true,
			RequireTetraReports:        true,
		}, nil
	case scopeP20BenchmarkMatrix:
		return benchmarkMatrixPolicy{
			Scope:                      scopeP20BenchmarkMatrix,
			Categories:                 P20BenchmarkCategories(),
			Languages:                  RequiredBenchmarkLanguages(),
			RequireEquivalenceMetadata: true,
			RequireTetraReports:        true,
			RequireRawOutputArtifacts:  true,
		}, nil
	case scopeP15ActorBenchmarkPrep:
		return benchmarkMatrixPolicy{
			Scope:                      scopeP15ActorBenchmarkPrep,
			Categories:                 P15ActorBenchmarkPrepCategories(),
			Languages:                  []string{"tetra"},
			RequireEquivalenceMetadata: true,
			RequireTetraReports:        true,
			RequireRawOutputArtifacts:  true,
		}, nil
	default:
		return benchmarkMatrixPolicy{}, fmt.Errorf("unsupported benchmark scope %q", scope)
	}
}

func normalizeBenchmarkScope(scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return scopeP8Full
	}
	return scope
}

var requiredBenchmarkCategories = []string{
	"integer loop",
	"slice sum",
	"bounds-check loop",
	"allocation microbench",
	"stack allocation",
	"region/island allocation",
	"copy/copy_into",
	"hash table",
	"JSON parse",
	"HTTP plaintext",
	"DB single query",
	"actor ping-pong",
	"actor zero-copy transfer",
	"parallel map/reduce",
}

var requiredBenchmarkLanguages = []string{"tetra", "c", "cpp", "rust"}

func P20BenchmarkCategories() []string {
	return append([]string(nil), p20BenchmarkCategories...)
}

var p20BenchmarkCategories = []string{
	"integer loops",
	"slice sum",
	"bounds-check loops",
	"function calls",
	"recursion",
	"matrix multiply",
	"hash table",
	"allocation",
	"region/island allocation",
	"JSON parse/stringify",
	"HTTP plaintext/json",
	"PostgreSQL single/multiple/update",
	"actor ping-pong",
	"parallel map/reduce",
	"startup time",
	"binary size",
	"compile time",
}

func P15ActorBenchmarkPrepCategories() []string {
	return append([]string(nil), p15ActorBenchmarkPrepCategories...)
}

var p15ActorBenchmarkPrepCategories = []string{
	"actor ping-pong",
	"actor fanout/fanin",
	"actor mailbox throughput",
	"actor backpressure latency",
	"zero_copy_move local typed mailbox",
}

func policyCategory(policy benchmarkMatrixPolicy, category string) bool {
	for _, supported := range policy.Categories {
		if category == supported {
			return true
		}
	}
	return false
}

func policyLanguage(policy benchmarkMatrixPolicy, language string) bool {
	for _, supported := range policy.Languages {
		if language == supported {
			return true
		}
	}
	return false
}

func supportedLanguage(language string) bool {
	for _, supported := range requiredBenchmarkLanguages {
		if language == supported {
			return true
		}
	}
	return false
}

func requiredCategory(category string) bool {
	for _, supported := range requiredBenchmarkCategories {
		if category == supported {
			return true
		}
	}
	return false
}

func validateP8BuildCommand(bench BenchmarkSpec) error {
	switch bench.Language {
	case "tetra":
		if bench.BuildCommand[0] != "tetra" || !containsArg(bench.BuildCommand, "--explain") {
			return fmt.Errorf(
				"benchmark %s Tetra build command must start with tetra and include --explain",
				bench.Name,
			)
		}
	case "c":
		if bench.BuildCommand[0] != "clang" || !containsArg(bench.BuildCommand, "-O3") {
			return fmt.Errorf("benchmark %s C build command must use clang -O3", bench.Name)
		}
	case "cpp":
		if bench.BuildCommand[0] != "clang++" || !containsArg(bench.BuildCommand, "-O3") {
			return fmt.Errorf("benchmark %s C++ build command must use clang++ -O3", bench.Name)
		}
	case "rust":
		if bench.BuildCommand[0] != "rustc" || !containsRustOptLevel3(bench.BuildCommand) {
			return fmt.Errorf(
				"benchmark %s Rust build command must use rustc -C opt-level=3",
				bench.Name,
			)
		}
	}
	return nil
}

func containsArg(argv []string, want string) bool {
	for _, arg := range argv {
		if arg == want {
			return true
		}
	}
	return false
}

func containsRustOptLevel3(argv []string) bool {
	for i, arg := range argv {
		if arg == "-C" && i+1 < len(argv) && argv[i+1] == "opt-level=3" {
			return true
		}
		if arg == "-C opt-level=3" {
			return true
		}
	}
	return false
}

func commandString(argv []string) string {
	return strings.Join(argv, " ")
}

func validateArtifactsExist(benchmark string, kind string, artifacts []ReportArtifact) error {
	if len(artifacts) == 0 {
		return fmt.Errorf("benchmark %s missing %s reports", benchmark, kind)
	}
	for _, artifact := range artifacts {
		if strings.TrimSpace(artifact.Path) == "" || !artifact.Exists {
			return fmt.Errorf(
				"benchmark %s has missing %s report artifact %q",
				benchmark,
				kind,
				artifact.Path,
			)
		}
	}
	return nil
}

func validateClaims(claims []string) error {
	if len(claims) == 0 {
		return fmt.Errorf("claim policy note is required")
	}
	for _, claim := range claims {
		if err := validatePerformanceClaimTextForTier(claim, 0); err != nil {
			return err
		}
	}
	return nil
}

func containsForbiddenCPlusPlusRustParityClaim(lower string) bool {
	if !strings.Contains(lower, "parity") {
		return false
	}
	if !strings.Contains(lower, "c++") && !strings.Contains(lower, "rust") {
		return false
	}
	if isExplicitNonClaimSentence(lower) {
		return false
	}
	for _, safe := range []string{
		"not claimed",
		"not proven",
		"not implied",
		"does not claim",
		"no c++/rust parity",
	} {
		if strings.Contains(lower, safe) {
			return false
		}
	}
	return true
}

func detectTargetCPU() string {
	if runtime.GOOS == "linux" {
		if raw, err := os.ReadFile("/proc/cpuinfo"); err == nil {
			for _, line := range strings.Split(string(raw), "\n") {
				if strings.HasPrefix(line, "model name") || strings.HasPrefix(line, "Hardware") {
					if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
						if cpu := strings.TrimSpace(parts[1]); cpu != "" {
							return cpu
						}
					}
				}
			}
		}
	}
	return runtime.GOOS + "/" + runtime.GOARCH
}

func slugCategory(category string) string {
	replacer := strings.NewReplacer("/", "_", "-", "_")
	return strings.Join(strings.Fields(replacer.Replace(strings.ToLower(category))), "_")
}
