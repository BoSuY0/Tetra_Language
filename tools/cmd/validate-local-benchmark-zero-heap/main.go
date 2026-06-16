package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"tetra_language/tools/internal/heaptelemetry"
	"tetra_language/tools/internal/zeroheapbench"
)

const schemaBenchmarkMemoryV1 = "tetra.local_benchmark.memory_evidence.v1"

type report struct {
	Schema      string   `json:"schema"`
	Scope       string   `json:"scope"`
	GeneratedAt string   `json:"generated_at"`
	Policy      policy   `json:"policy"`
	NonClaims   []string `json:"non_claims"`
	Results     []row    `json:"results"`
}

type policy struct {
	Suite      string `json:"suite"`
	Iterations int    `json:"iterations"`
}

type row struct {
	Name               string        `json:"name"`
	Category           string        `json:"category"`
	AlgorithmID        string        `json:"algorithm_id"`
	InputDescription   string        `json:"input_description"`
	Language           string        `json:"language"`
	Status             string        `json:"status"`
	CompilerVersion    string        `json:"compiler_version"`
	BuildCommand       []string      `json:"build_command"`
	RunCommand         []string      `json:"run_command"`
	SourcePath         string        `json:"source_path"`
	BinaryPath         string        `json:"binary_path"`
	BinarySizeBytes    int64         `json:"binary_size_bytes"`
	CompileTimeMS      float64       `json:"compile_time_ms"`
	RunMeasurementsMS  []float64     `json:"run_measurements_ms"`
	MedianRuntimeMS    float64       `json:"median_runtime_ms"`
	RawOutputArtifacts []string      `json:"raw_output_artifacts"`
	TetraMetadata      tetraMetadata `json:"tetra_metadata"`
	Error              string        `json:"error,omitempty"`
}

type tetraMetadata struct {
	ProofReport       string         `json:"proof_report"`
	BoundsReport      string         `json:"bounds_report"`
	AllocationReport  string         `json:"allocation_report"`
	PerfBlockerReport string         `json:"perf_blocker_report"`
	BackendReport     string         `json:"backend_report"`
	BackendPath       string         `json:"backend_path"`
	BoundsLeft        int            `json:"bounds_left"`
	HeapAllocations   int            `json:"heap_allocations"`
	PerfBlockers      []string       `json:"perf_blockers"`
	MemoryEvidence    memoryEvidence `json:"memory_evidence"`
}

type memoryEvidence struct {
	Schema              string         `json:"schema"`
	HeapAllocBytes      memoryMetric   `json:"heap_alloc_bytes"`
	BytesRequested      memoryMetric   `json:"bytes_requested"`
	BytesReserved       memoryMetric   `json:"bytes_reserved"`
	BytesCommitted      memoryMetric   `json:"bytes_committed"`
	BytesCopied         memoryMetric   `json:"bytes_copied"`
	RSSCurrent          memoryMetric   `json:"rss_current"`
	RSSPeak             memoryMetric   `json:"rss_peak"`
	DomainBytesEvidence memoryMetric   `json:"domain_bytes_evidence"`
	DomainBytes         []domainMetric `json:"domain_bytes"`
}

type memoryMetric struct {
	Bytes             uint64 `json:"bytes,omitempty"`
	CurrentBytes      uint64 `json:"current_bytes,omitempty"`
	PeakBytes         uint64 `json:"peak_bytes,omitempty"`
	TotalAllocBytes   uint64 `json:"total_alloc_bytes,omitempty"`
	AllocationCount   uint64 `json:"allocation_count,omitempty"`
	EvidenceClass     string `json:"evidence_class"`
	Method            string `json:"method"`
	SourceArtifact    string `json:"source_artifact,omitempty"`
	UnsupportedReason string `json:"unsupported_reason,omitempty"`
	BlockedReason     string `json:"blocked_reason,omitempty"`
}

type domainMetric struct {
	DomainID      string `json:"domain_id"`
	Kind          string `json:"kind"`
	EvidenceClass string `json:"evidence_class"`
	Method        string `json:"method"`
}

func main() {
	reportPath := flag.String("report", "", "zero-heap local benchmark report JSON")
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "usage: validate-local-benchmark-zero-heap --report reports/local-benchmark-zero-heap-v1/report.json")
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
	var parsed report
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return err
	}
	return validateReport(parsed, root)
}

func validateReport(parsed report, root string) error {
	if parsed.Schema != zeroheapbench.Schema {
		return fmt.Errorf("schema = %q, want %q", parsed.Schema, zeroheapbench.Schema)
	}
	if parsed.Scope != zeroheapbench.Scope {
		return fmt.Errorf("scope = %q, want %q", parsed.Scope, zeroheapbench.Scope)
	}
	if strings.TrimSpace(parsed.GeneratedAt) == "" {
		return fmt.Errorf("generated_at is required")
	}
	if parsed.Policy.Suite != "zero_heap_microbenchmarks" {
		return fmt.Errorf("policy suite = %q, want zero_heap_microbenchmarks", parsed.Policy.Suite)
	}
	if parsed.Policy.Iterations <= 0 {
		return fmt.Errorf("policy iterations = %d, want positive", parsed.Policy.Iterations)
	}
	if err := validateNonClaims(parsed.NonClaims); err != nil {
		return err
	}
	if len(parsed.Results) != len(zeroheapbench.Categories) {
		return fmt.Errorf("report has %d zero-heap rows, want %d", len(parsed.Results), len(zeroheapbench.Categories))
	}
	allowed := map[string]bool{}
	for _, category := range zeroheapbench.Categories {
		allowed[category] = true
	}
	seen := map[string]bool{}
	for _, result := range parsed.Results {
		if !allowed[result.Category] {
			return fmt.Errorf("unexpected zero-heap category %q", result.Category)
		}
		if seen[result.Category] {
			return fmt.Errorf("duplicate zero-heap category %q", result.Category)
		}
		seen[result.Category] = true
		if err := validateRow(result, root); err != nil {
			return err
		}
	}
	for _, category := range zeroheapbench.Categories {
		if !seen[category] {
			return fmt.Errorf("missing zero-heap category %q", category)
		}
	}
	return nil
}

func validateNonClaims(nonClaims []string) error {
	required := []string{
		"no official benchmark claim",
		"no cross-language performance claim",
		"no zero rss claim",
		"no universal zero heap claim",
	}
	seen := map[string]bool{}
	for _, claim := range nonClaims {
		claim = strings.ToLower(strings.TrimSpace(claim))
		if claim == "" {
			return fmt.Errorf("empty non-claim")
		}
		seen[claim] = true
	}
	for _, claim := range required {
		if !seen[claim] {
			return fmt.Errorf("missing non-claim %q", claim)
		}
	}
	return nil
}

func validateRow(result row, root string) error {
	prefix := "zero-heap row " + result.Name
	if strings.TrimSpace(result.Name) == "" {
		return fmt.Errorf("zero-heap row missing name")
	}
	if result.Language != "tetra" {
		return fmt.Errorf("%s language = %q, want tetra-only row", prefix, result.Language)
	}
	if result.Status != "measured" {
		return fmt.Errorf("%s status = %q, want measured", prefix, result.Status)
	}
	if strings.TrimSpace(result.AlgorithmID) == "" || strings.TrimSpace(result.InputDescription) == "" {
		return fmt.Errorf("%s missing algorithm metadata", prefix)
	}
	if len(result.BuildCommand) == 0 || len(result.RunCommand) == 0 {
		return fmt.Errorf("%s missing build/run command", prefix)
	}
	if err := requireExistingPath(root, result.SourcePath); err != nil {
		return fmt.Errorf("%s source_path: %w", prefix, err)
	}
	if err := requireExistingPath(root, result.BinaryPath); err != nil {
		return fmt.Errorf("%s binary_path: %w", prefix, err)
	}
	if result.BinarySizeBytes <= 0 {
		return fmt.Errorf("%s binary_size_bytes = %d, want positive", prefix, result.BinarySizeBytes)
	}
	if result.CompileTimeMS < 0 || result.MedianRuntimeMS <= 0 || len(result.RunMeasurementsMS) == 0 {
		return fmt.Errorf("%s missing positive timing evidence", prefix)
	}
	if len(result.RawOutputArtifacts) == 0 {
		return fmt.Errorf("%s missing raw output artifacts", prefix)
	}
	for _, artifact := range result.RawOutputArtifacts {
		if err := requireExistingPath(root, artifact); err != nil {
			return fmt.Errorf("%s raw artifact: %w", prefix, err)
		}
	}
	for label, path := range map[string]string{
		"proof report":        result.TetraMetadata.ProofReport,
		"bounds report":       result.TetraMetadata.BoundsReport,
		"allocation report":   result.TetraMetadata.AllocationReport,
		"perf blocker report": result.TetraMetadata.PerfBlockerReport,
		"backend report":      result.TetraMetadata.BackendReport,
	} {
		if err := requireExistingPath(root, path); err != nil {
			return fmt.Errorf("%s %s: %w", prefix, label, err)
		}
	}
	if result.TetraMetadata.HeapAllocations != 0 {
		return fmt.Errorf("%s zero-heap tetra_metadata.heap_allocations = %d, want 0", prefix, result.TetraMetadata.HeapAllocations)
	}
	if result.TetraMetadata.BoundsLeft < 0 {
		return fmt.Errorf("%s bounds_left = %d, want non-negative", prefix, result.TetraMetadata.BoundsLeft)
	}
	return validateMemoryEvidence(prefix, result.Name, result.TetraMetadata.MemoryEvidence, root)
}

func validateMemoryEvidence(prefix string, name string, evidence memoryEvidence, root string) error {
	if evidence.Schema != schemaBenchmarkMemoryV1 {
		return fmt.Errorf("%s memory evidence schema = %q, want %q", prefix, evidence.Schema, schemaBenchmarkMemoryV1)
	}
	if err := validateZeroHeapMetric(prefix, name, evidence.HeapAllocBytes, root); err != nil {
		return err
	}
	for metricName, metric := range map[string]memoryMetric{
		"bytes_requested":       evidence.BytesRequested,
		"bytes_reserved":        evidence.BytesReserved,
		"bytes_committed":       evidence.BytesCommitted,
		"bytes_copied":          evidence.BytesCopied,
		"rss_current":           evidence.RSSCurrent,
		"rss_peak":              evidence.RSSPeak,
		"domain_bytes_evidence": evidence.DomainBytesEvidence,
	} {
		if err := validateMetric(prefix+" "+metricName, metric, root); err != nil {
			return err
		}
	}
	return nil
}

func validateZeroHeapMetric(prefix string, name string, metric memoryMetric, root string) error {
	metricPrefix := prefix + " heap_alloc_bytes"
	if metric.EvidenceClass != "runtime_measured" {
		return fmt.Errorf("%s evidence_class = %q, want runtime_measured", metricPrefix, metric.EvidenceClass)
	}
	if metric.Method != heaptelemetry.MethodLinuxX64HeapTelemetryV1 {
		return fmt.Errorf("%s method = %q, want %q", metricPrefix, metric.Method, heaptelemetry.MethodLinuxX64HeapTelemetryV1)
	}
	if metric.Bytes != 0 || metric.CurrentBytes != 0 || metric.PeakBytes != 0 || metric.TotalAllocBytes != 0 || metric.AllocationCount != 0 {
		return fmt.Errorf("%s zero-heap counters are non-zero: %+v", metricPrefix, metric)
	}
	resolved, err := resolveExistingPath(root, metric.SourceArtifact)
	if err != nil {
		return fmt.Errorf("%s source_artifact: %w", metricPrefix, err)
	}
	sample, err := heaptelemetry.ReadFile(resolved, root)
	if err != nil {
		return fmt.Errorf("%s source_artifact: %w", metricPrefix, err)
	}
	if sample.Program != name {
		return fmt.Errorf("%s sidecar program = %q, want %q", metricPrefix, sample.Program, name)
	}
	if sample.HeapCurrentBytes != 0 || sample.HeapPeakBytes != 0 || sample.HeapTotalAllocBytes != 0 || sample.HeapAllocationCount != 0 {
		return fmt.Errorf("%s sidecar zero-heap counters are non-zero: %+v", metricPrefix, sample)
	}
	return nil
}

func validateMetric(prefix string, metric memoryMetric, root string) error {
	switch metric.EvidenceClass {
	case "allocation_report_estimate":
		if metric.Method != "allocation_report_summary" {
			return fmt.Errorf("%s allocation estimate method = %q, want allocation_report_summary", prefix, metric.Method)
		}
		if err := requireExistingPath(root, metric.SourceArtifact); err != nil {
			return fmt.Errorf("%s source_artifact: %w", prefix, err)
		}
	case "unsupported":
		if strings.TrimSpace(metric.UnsupportedReason) == "" {
			return fmt.Errorf("%s unsupported evidence requires unsupported_reason", prefix)
		}
	case "blocked":
		if strings.TrimSpace(metric.BlockedReason) == "" {
			return fmt.Errorf("%s blocked evidence requires blocked_reason", prefix)
		}
	case "runtime_measured":
		if strings.TrimSpace(metric.Method) == "" {
			return fmt.Errorf("%s runtime_measured evidence requires method", prefix)
		}
	default:
		return fmt.Errorf("%s evidence_class %q is not allowed", prefix, metric.EvidenceClass)
	}
	return nil
}

func requireExistingPath(root string, path string) error {
	_, err := resolveExistingPath(root, path)
	return err
}

func resolveExistingPath(root string, path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("path is required")
	}
	resolved := path
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(root, path)
	}
	if _, err := os.Stat(resolved); err != nil {
		return "", fmt.Errorf("%s does not exist", path)
	}
	return resolved, nil
}
