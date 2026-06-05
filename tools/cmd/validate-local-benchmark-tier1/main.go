package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	schemaLocalBenchmarkTier1  = "tetra.local_benchmark_tier1.v1"
	scopeP25RealLocalBenchmark = "p25.0_real_local_benchmark_execution_v1"
)

var requiredP20Categories = []string{
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

var requiredLanguages = []string{"tetra", "c", "cpp", "rust"}

var allowedClassifications = map[string]bool{
	"faster than C/C++/Rust locally":      true,
	"comparable":                          true,
	"slower":                              true,
	"invalid/inconclusive":                true,
	"blocked by missing feature":          true,
	"blocked by fallback backend":         true,
	"blocked by heap allocation":          true,
	"blocked by bounds check":             true,
	"blocked by actor/runtime limitation": true,
}

var allowedStatuses = map[string]bool{
	"measured":     true,
	"build_failed": true,
	"run_failed":   true,
	"blocked":      true,
	"skipped":      true,
}

var allowedBackendPaths = map[string]bool{
	"register": true,
	"stack":    true,
	"fallback": true,
}

type tier1Report struct {
	Schema              string              `json:"schema"`
	Scope               string              `json:"scope"`
	GeneratedAt         string              `json:"generated_at"`
	Host                tier1Host           `json:"host"`
	Policy              tier1Policy         `json:"policy"`
	NonClaims           []string            `json:"non_claims"`
	OptimizerValidation optimizerValidation `json:"optimizer_validation"`
	Results             []categoryResult    `json:"results"`
}

type tier1Host struct {
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
	CPUs      int    `json:"cpus"`
	TargetCPU string `json:"target_cpu"`
	GitCommit string `json:"git_commit"`
}

type tier1Policy struct {
	Tier                string  `json:"tier"`
	ComparableThreshold float64 `json:"comparable_threshold"`
	Iterations          int     `json:"iterations"`
}

type optimizerValidation struct {
	Status   string `json:"status"`
	Artifact string `json:"artifact"`
}

type categoryResult struct {
	Category             string         `json:"category"`
	AlgorithmID          string         `json:"algorithm_id"`
	InputDescription     string         `json:"input_description"`
	Classification       string         `json:"classification"`
	ClassificationReason string         `json:"classification_reason"`
	Rows                 []benchmarkRow `json:"rows"`
}

type benchmarkRow struct {
	Name               string         `json:"name"`
	Category           string         `json:"category"`
	Language           string         `json:"language"`
	Status             string         `json:"status"`
	CompilerVersion    string         `json:"compiler_version"`
	BuildCommand       []string       `json:"build_command"`
	RunCommand         []string       `json:"run_command"`
	SourcePath         string         `json:"source_path"`
	BinaryPath         string         `json:"binary_path"`
	BinarySizeBytes    int64          `json:"binary_size_bytes"`
	CompileTimeMS      float64        `json:"compile_time_ms"`
	RunMeasurementsMS  []float64      `json:"run_measurements_ms"`
	MedianRuntimeMS    float64        `json:"median_runtime_ms"`
	RawOutputArtifacts []string       `json:"raw_output_artifacts"`
	TetraMetadata      *tetraMetadata `json:"tetra_metadata,omitempty"`
	Error              string         `json:"error,omitempty"`
}

type tetraMetadata struct {
	ProofReport                 string              `json:"proof_report"`
	BoundsReport                string              `json:"bounds_report"`
	AllocationReport            string              `json:"allocation_report"`
	PerfBlockerReport           string              `json:"perf_blocker_report"`
	BackendReport               string              `json:"backend_report"`
	BackendPath                 string              `json:"backend_path"`
	BoundsLeft                  int                 `json:"bounds_left"`
	HeapAllocations             int                 `json:"heap_allocations"`
	PerfBlockers                []string            `json:"perf_blockers"`
	OptimizerValidationMetadata optimizerValidation `json:"optimizer_validation_metadata"`
}

func main() {
	reportPath := flag.String("report", "", "P25.0 local Tier 1 benchmark report JSON")
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "usage: validate-local-benchmark-tier1 --report reports/local-benchmark-tier1-v1/report.json")
		os.Exit(2)
	}
	raw, err := os.ReadFile(*reportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := ValidateReportBytes(raw, root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func ValidateReportBytes(raw []byte, root string) error {
	var report tier1Report
	if err := json.Unmarshal(raw, &report); err != nil {
		return err
	}
	return validateReport(report, root)
}

func validateReport(report tier1Report, root string) error {
	if report.Schema != schemaLocalBenchmarkTier1 {
		return fmt.Errorf("schema = %q, want %q", report.Schema, schemaLocalBenchmarkTier1)
	}
	if report.Scope != scopeP25RealLocalBenchmark {
		return fmt.Errorf("scope = %q, want %q", report.Scope, scopeP25RealLocalBenchmark)
	}
	if strings.TrimSpace(report.GeneratedAt) == "" {
		return fmt.Errorf("generated_at is required")
	}
	if err := validateHost(report.Host); err != nil {
		return err
	}
	if err := validatePolicy(report.Policy); err != nil {
		return err
	}
	if err := validateNonClaims(report.NonClaims); err != nil {
		return err
	}
	if err := validateOptimizerValidation("top-level optimizer validation", report.OptimizerValidation, root); err != nil {
		return err
	}
	seenCategories := map[string]bool{}
	resultsByCategory := map[string]categoryResult{}
	for _, result := range report.Results {
		if strings.TrimSpace(result.Category) == "" {
			return fmt.Errorf("result missing category")
		}
		if seenCategories[result.Category] {
			return fmt.Errorf("duplicate category %q", result.Category)
		}
		seenCategories[result.Category] = true
		resultsByCategory[result.Category] = result
		if err := validateCategoryResult(result, root); err != nil {
			return err
		}
	}
	for _, category := range requiredP20Categories {
		if !seenCategories[category] {
			return fmt.Errorf("missing category %q", category)
		}
	}
	if len(resultsByCategory) != len(requiredP20Categories) {
		return fmt.Errorf("report has %d categories, want %d", len(resultsByCategory), len(requiredP20Categories))
	}
	return nil
}

func validateHost(host tier1Host) error {
	if host.GOOS == "" || host.GOARCH == "" || host.CPUs <= 0 || strings.TrimSpace(host.TargetCPU) == "" || strings.TrimSpace(host.GitCommit) == "" {
		return fmt.Errorf("host metadata is incomplete: %+v", host)
	}
	return nil
}

func validatePolicy(policy tier1Policy) error {
	if policy.Tier != "tier1_local_benchmark_evidence" {
		return fmt.Errorf("policy tier = %q, want tier1_local_benchmark_evidence", policy.Tier)
	}
	if policy.ComparableThreshold <= 0 || policy.ComparableThreshold > 1 {
		return fmt.Errorf("policy comparable_threshold = %f, want 0 < threshold <= 1", policy.ComparableThreshold)
	}
	if policy.Iterations <= 0 {
		return fmt.Errorf("policy iterations = %d, want positive", policy.Iterations)
	}
	return nil
}

func validateNonClaims(nonClaims []string) error {
	required := []string{
		"no fastest-language claim",
		"no official benchmark claim",
		"no cross-machine claim",
		"no TechEmpower claim",
		"no production claim",
	}
	seen := map[string]bool{}
	for _, claim := range nonClaims {
		lower := strings.ToLower(strings.TrimSpace(claim))
		if lower == "" {
			return fmt.Errorf("empty non-claim")
		}
		switch {
		case strings.Contains(lower, "fastest") && !strings.Contains(lower, "no fastest-language"):
			return fmt.Errorf("fastest-language claim is forbidden: %q", claim)
		case strings.Contains(lower, "official") && !strings.Contains(lower, "no official"):
			return fmt.Errorf("official benchmark claim is forbidden: %q", claim)
		case strings.Contains(lower, "cross-machine") && !strings.Contains(lower, "no cross-machine"):
			return fmt.Errorf("cross-machine claim is forbidden: %q", claim)
		case strings.Contains(lower, "techempower") && !strings.Contains(lower, "no techempower"):
			return fmt.Errorf("TechEmpower claim is forbidden: %q", claim)
		case strings.Contains(lower, "production") && !strings.Contains(lower, "no production"):
			return fmt.Errorf("production claim is forbidden: %q", claim)
		}
		seen[lower] = true
	}
	for _, claim := range required {
		if !seen[strings.ToLower(claim)] {
			return fmt.Errorf("missing non-claim %q", claim)
		}
	}
	return nil
}

func validateOptimizerValidation(label string, metadata optimizerValidation, root string) error {
	if strings.TrimSpace(metadata.Status) == "" {
		return fmt.Errorf("%s missing status", label)
	}
	if strings.TrimSpace(metadata.Artifact) == "" {
		return fmt.Errorf("%s missing artifact", label)
	}
	if err := requireExistingPath(root, metadata.Artifact); err != nil {
		return fmt.Errorf("%s artifact: %w", label, err)
	}
	return nil
}

func validateCategoryResult(result categoryResult, root string) error {
	if strings.TrimSpace(result.AlgorithmID) == "" {
		return fmt.Errorf("category %q missing algorithm_id", result.Category)
	}
	if strings.TrimSpace(result.InputDescription) == "" {
		return fmt.Errorf("category %q missing input_description", result.Category)
	}
	if !allowedClassifications[result.Classification] {
		return fmt.Errorf("category %q classification %q is not allowed", result.Category, result.Classification)
	}
	if strings.TrimSpace(result.ClassificationReason) == "" {
		return fmt.Errorf("category %q missing classification_reason", result.Category)
	}
	seenLanguages := map[string]bool{}
	for _, row := range result.Rows {
		if row.Category != result.Category {
			return fmt.Errorf("row %q category = %q, want %q", row.Name, row.Category, result.Category)
		}
		if seenLanguages[row.Language] {
			return fmt.Errorf("duplicate matrix row for category %q language %q", result.Category, row.Language)
		}
		seenLanguages[row.Language] = true
		if err := validateBenchmarkRow(row, root); err != nil {
			return err
		}
	}
	for _, language := range requiredLanguages {
		if !seenLanguages[language] {
			return fmt.Errorf("missing matrix row for category %q language %q", result.Category, language)
		}
	}
	if len(seenLanguages) != len(requiredLanguages) {
		return fmt.Errorf("category %q has %d language rows, want %d", result.Category, len(seenLanguages), len(requiredLanguages))
	}
	return nil
}

func validateBenchmarkRow(row benchmarkRow, root string) error {
	if strings.TrimSpace(row.Name) == "" {
		return fmt.Errorf("benchmark row missing name")
	}
	if !languageAllowed(row.Language) {
		return fmt.Errorf("benchmark %s language %q is not allowed", row.Name, row.Language)
	}
	if !allowedStatuses[row.Status] {
		return fmt.Errorf("benchmark %s status %q is not allowed", row.Name, row.Status)
	}
	if strings.TrimSpace(row.CompilerVersion) == "" {
		return fmt.Errorf("benchmark %s missing compiler_version", row.Name)
	}
	if len(row.BuildCommand) == 0 || strings.TrimSpace(row.BuildCommand[0]) == "" {
		return fmt.Errorf("benchmark %s missing build_command", row.Name)
	}
	if len(row.RunCommand) == 0 || strings.TrimSpace(row.RunCommand[0]) == "" {
		return fmt.Errorf("benchmark %s missing run_command", row.Name)
	}
	if err := requireExistingPath(root, row.SourcePath); err != nil {
		return fmt.Errorf("benchmark %s source_path: %w", row.Name, err)
	}
	if len(row.RawOutputArtifacts) == 0 {
		return fmt.Errorf("benchmark %s missing raw_output_artifacts", row.Name)
	}
	for _, artifact := range row.RawOutputArtifacts {
		if err := requireExistingPath(root, artifact); err != nil {
			return fmt.Errorf("benchmark %s raw artifact: %w", row.Name, err)
		}
	}
	if row.CompileTimeMS < 0 {
		return fmt.Errorf("benchmark %s compile_time_ms is negative", row.Name)
	}
	if row.Status == "measured" {
		if err := requireExistingPath(root, row.BinaryPath); err != nil {
			return fmt.Errorf("benchmark %s binary_path: %w", row.Name, err)
		}
		if row.BinarySizeBytes <= 0 {
			return fmt.Errorf("benchmark %s binary_size_bytes = %d, want positive", row.Name, row.BinarySizeBytes)
		}
		if len(row.RunMeasurementsMS) == 0 || row.MedianRuntimeMS <= 0 {
			return fmt.Errorf("benchmark %s missing positive runtime measurements", row.Name)
		}
		for _, ms := range row.RunMeasurementsMS {
			if ms < 0 {
				return fmt.Errorf("benchmark %s has negative runtime measurement", row.Name)
			}
		}
	}
	if row.Language == "tetra" {
		if row.TetraMetadata == nil {
			return fmt.Errorf("benchmark %s missing tetra metadata", row.Name)
		}
		if err := validateTetraMetadata(row.Name, *row.TetraMetadata, root); err != nil {
			return err
		}
	} else if row.TetraMetadata != nil {
		return fmt.Errorf("benchmark %s non-Tetra row carries tetra metadata", row.Name)
	}
	return nil
}

func validateTetraMetadata(name string, metadata tetraMetadata, root string) error {
	for label, path := range map[string]string{
		"proof report":        metadata.ProofReport,
		"bounds report":       metadata.BoundsReport,
		"allocation report":   metadata.AllocationReport,
		"perf blocker report": metadata.PerfBlockerReport,
		"backend report":      metadata.BackendReport,
	} {
		if err := requireExistingPath(root, path); err != nil {
			return fmt.Errorf("benchmark %s tetra metadata %s: %w", name, label, err)
		}
	}
	if !allowedBackendPaths[metadata.BackendPath] {
		return fmt.Errorf("benchmark %s tetra metadata backend_path %q is not allowed", name, metadata.BackendPath)
	}
	if metadata.BoundsLeft < 0 || metadata.HeapAllocations < 0 {
		return fmt.Errorf("benchmark %s tetra metadata has negative bounds/heap totals", name)
	}
	if err := validateOptimizerValidation("benchmark "+name+" optimizer validation", metadata.OptimizerValidationMetadata, root); err != nil {
		return err
	}
	return nil
}

func requireExistingPath(root string, path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("path is required")
	}
	resolved := path
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(root, path)
	}
	if _, err := os.Stat(resolved); err != nil {
		return fmt.Errorf("%s does not exist", path)
	}
	return nil
}

func languageAllowed(language string) bool {
	for _, supported := range requiredLanguages {
		if language == supported {
			return true
		}
	}
	return false
}

func slug(value string) string {
	replacer := strings.NewReplacer("/", "_", "-", "_")
	return strings.Join(strings.Fields(replacer.Replace(strings.ToLower(value))), "_")
}

func hostDefaults() tier1Host {
	return tier1Host{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, CPUs: runtime.NumCPU()}
}
