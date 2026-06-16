package main

import (
	"tetra_language/tools/internal/heaptelemetry"
	"tetra_language/tools/internal/rsstelemetry"
	"time"
)

const (
	schemaLocalBenchmarkTier1  = "tetra.local_benchmark_tier1.v1"
	scopeP25RealLocalBenchmark = "p25.0_real_local_benchmark_execution_v1"
	schemaBenchmarkMemoryV1    = "tetra.local_benchmark.memory_evidence.v1"
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

type benchmarkSpec struct {
	Name             string
	Category         string
	Language         string
	AlgorithmID      string
	InputDescription string
	BuildCommandKind string
	BuildArgs        []string
	SourceRelPath    string
	BinaryRelPath    string
	Source           string
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
	ProofReport                 string                 `json:"proof_report"`
	BoundsReport                string                 `json:"bounds_report"`
	AllocationReport            string                 `json:"allocation_report"`
	PerfBlockerReport           string                 `json:"perf_blocker_report"`
	BackendReport               string                 `json:"backend_report"`
	BackendPath                 string                 `json:"backend_path"`
	BackendBlockers             []string               `json:"backend_blockers,omitempty"`
	RuntimeFeaturesRequired     []string               `json:"runtime_features_required"`
	RuntimeFeaturesLinked       []string               `json:"runtime_features_linked"`
	RuntimeFeaturesInitialized  []string               `json:"runtime_features_initialized"`
	RuntimeLazyInitBlockers     []string               `json:"runtime_lazy_init_blockers"`
	RuntimeFeatureEvidence      runtimeFeatureEvidence `json:"runtime_feature_evidence"`
	RuntimeObjectPlan           runtimeObjectPlan      `json:"runtime_object_plan"`
	BoundsLeft                  int                    `json:"bounds_left"`
	HeapAllocations             int                    `json:"heap_allocations"`
	HeapReasonCodes             []string               `json:"heap_reason_codes"`
	PerfBlockers                []string               `json:"perf_blockers"`
	OptimizerValidationMetadata optimizerValidation    `json:"optimizer_validation_metadata"`
	MemoryEvidence              *memoryEvidence        `json:"memory_evidence,omitempty"`
}

type runtimeFeatureEvidence struct {
	EvidenceClass     string `json:"evidence_class"`
	Method            string `json:"method"`
	SourceArtifact    string `json:"source_artifact,omitempty"`
	BlockedReason     string `json:"blocked_reason,omitempty"`
	UnsupportedReason string `json:"unsupported_reason,omitempty"`
}

type runtimeObjectPlan struct {
	EvidenceClass                    string   `json:"evidence_class"`
	EvidenceMethod                   string   `json:"evidence_method"`
	RuntimeUsed                      bool     `json:"runtime_used"`
	RuntimeObjectLinked              bool     `json:"runtime_object_linked"`
	RuntimeObjectInitialized         bool     `json:"runtime_object_initialized"`
	RuntimeObjectOverride            bool     `json:"runtime_object_override"`
	TimeOnlyRuntime                  bool     `json:"time_only_runtime"`
	LinuxMinimalRuntime              bool     `json:"linux_minimal_runtime"`
	RuntimeObjectFeaturesRequired    []string `json:"runtime_object_features_required"`
	RuntimeObjectFeaturesLinked      []string `json:"runtime_object_features_linked"`
	RuntimeObjectFeaturesInitialized []string `json:"runtime_object_features_initialized"`
	RuntimeObjectLazyInitBlockers    []string `json:"runtime_object_lazy_init_blockers"`
	BlockedReason                    string   `json:"blocked_reason,omitempty"`
	UnsupportedReason                string   `json:"unsupported_reason,omitempty"`
}

type memoryEvidence struct {
	Schema              string             `json:"schema"`
	HeapAllocBytes      memoryMetric       `json:"heap_alloc_bytes"`
	BytesRequested      memoryMetric       `json:"bytes_requested"`
	BytesReserved       memoryMetric       `json:"bytes_reserved"`
	BytesCommitted      memoryMetric       `json:"bytes_committed"`
	BytesReleased       memoryMetric       `json:"bytes_released"`
	BytesCopied         memoryMetric       `json:"bytes_copied"`
	RSSCurrent          memoryMetric       `json:"rss_current"`
	RSSPeak             memoryMetric       `json:"rss_peak"`
	DomainBytesEvidence memoryMetric       `json:"domain_bytes_evidence"`
	DomainBytes         []memoryDomainByte `json:"domain_bytes"`
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

type memoryDomainByte struct {
	DomainID       string `json:"domain_id"`
	Kind           string `json:"kind"`
	RequestedBytes uint64 `json:"requested_bytes,omitempty"`
	ReservedBytes  uint64 `json:"reserved_bytes,omitempty"`
	CommittedBytes uint64 `json:"committed_bytes,omitempty"`
	ReleasedBytes  uint64 `json:"released_bytes,omitempty"`
	CurrentBytes   uint64 `json:"current_bytes,omitempty"`
	PeakBytes      uint64 `json:"peak_bytes,omitempty"`
	BytesCopied    uint64 `json:"bytes_copied,omitempty"`
	EvidenceClass  string `json:"evidence_class"`
	Method         string `json:"method"`
	SourceArtifact string `json:"source_artifact,omitempty"`
}

type runtimeHeapEvidence struct {
	SourceArtifact string
	Sample         heaptelemetry.Sample
}

type runtimeRSSEvidence struct {
	SourceArtifact string
	Sample         rsstelemetry.Sample
}

type heapTelemetrySummarySample struct {
	Iteration           int    `json:"iteration"`
	Artifact            string `json:"artifact"`
	HeapCurrentBytes    uint64 `json:"heap_current_bytes"`
	HeapPeakBytes       uint64 `json:"heap_peak_bytes"`
	HeapTotalAllocBytes uint64 `json:"heap_total_alloc_bytes"`
	HeapAllocationCount uint64 `json:"heap_allocation_count"`
	BytesRequested      uint64 `json:"bytes_requested"`
	BytesReserved       uint64 `json:"bytes_reserved"`
}

type rssTelemetrySummarySample struct {
	Iteration            int    `json:"iteration"`
	Artifact             string `json:"artifact"`
	SampleCount          uint64 `json:"sample_count"`
	RSSCurrentBytes      uint64 `json:"rss_current_bytes"`
	RSSPeakBytes         uint64 `json:"rss_peak_bytes"`
	RSSPeakSource        string `json:"rss_peak_source"`
	RUMaxRSSRaw          uint64 `json:"ru_maxrss_raw,omitempty"`
	RUMaxRSSUnit         string `json:"ru_maxrss_unit,omitempty"`
	SampleIntervalMicros uint64 `json:"sample_interval_micros,omitempty"`
}

type options struct {
	OutDir     string
	Iterations int
	Timeout    time.Duration
}
