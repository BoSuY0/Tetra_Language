package main

import "time"

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

type options struct {
	OutDir     string
	Iterations int
	Timeout    time.Duration
}
