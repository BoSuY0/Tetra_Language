package runtimeabi

import "fmt"

const MemoryBackendContractSchemaV1 = "tetra.memory.backend-contract.v1"
const MemoryBackendAllocationEvidenceSchemaV1 = "tetra.memory.backend-allocation.v1"

type MemoryBackendOperation string

const (
	MemoryBackendReserve   MemoryBackendOperation = "reserve"
	MemoryBackendCommit    MemoryBackendOperation = "commit"
	MemoryBackendDecommit  MemoryBackendOperation = "decommit"
	MemoryBackendRelease   MemoryBackendOperation = "release"
	MemoryBackendTrim      MemoryBackendOperation = "trim"
	MemoryBackendFootprint MemoryBackendOperation = "footprint"
)

type MemoryFootprintEvidenceClass string

const (
	MemoryFootprintMeasured    MemoryFootprintEvidenceClass = "runtime_measured"
	MemoryFootprintEstimated   MemoryFootprintEvidenceClass = "allocation_report_estimate"
	MemoryFootprintUnsupported MemoryFootprintEvidenceClass = "unsupported"
	MemoryFootprintBlocked     MemoryFootprintEvidenceClass = "blocked"
)

type MemoryBackendClass string

const (
	MemoryBackendClassNone             MemoryBackendClass = "none"
	MemoryBackendClassSmallHeap        MemoryBackendClass = "small_heap"
	MemoryBackendClassRegion           MemoryBackendClass = "region"
	MemoryBackendClassLargeBackend     MemoryBackendClass = "large_backend"
	MemoryBackendClassExternal         MemoryBackendClass = "external"
	MemoryBackendClassConservativeHeap MemoryBackendClass = "conservative_heap"
	MemoryBackendClassUnknown          MemoryBackendClass = "unknown"
)

type MemoryBackendContract struct {
	Schema                 string                       `json:"schema"`
	Target                 string                       `json:"target"`
	Operations             []MemoryBackendOperation     `json:"operations"`
	MinAlignmentBytes      int                          `json:"min_alignment_bytes"`
	ReserveGranularity     int                          `json:"reserve_granularity_bytes"`
	CommitGranularity      int                          `json:"commit_granularity_bytes"`
	FootprintEvidenceClass MemoryFootprintEvidenceClass `json:"footprint_evidence_class"`
	FootprintMethod        string                       `json:"footprint_method"`
	UnsupportedReason      string                       `json:"unsupported_reason,omitempty"`
	NonClaims              []string                     `json:"non_claims,omitempty"`
}

type MemoryBackendAllocationEvidence struct {
	Schema                string                       `json:"schema"`
	BackendClass          MemoryBackendClass           `json:"backend_class"`
	Adapter               string                       `json:"adapter"`
	RuntimePath           AllocationRuntimePath        `json:"runtime_path"`
	Operations            []MemoryBackendOperation     `json:"operations,omitempty"`
	EvidenceClass         MemoryFootprintEvidenceClass `json:"evidence_class"`
	Method                string                       `json:"method"`
	ReserveBytes          int64                        `json:"reserve_bytes,omitempty"`
	CommitBytes           int64                        `json:"commit_bytes,omitempty"`
	DecommitBytes         int64                        `json:"decommit_bytes,omitempty"`
	ReleaseBytes          int64                        `json:"release_bytes,omitempty"`
	FootprintCurrentBytes int64                        `json:"footprint_current_bytes,omitempty"`
	FootprintPeakBytes    int64                        `json:"footprint_peak_bytes,omitempty"`
	UnsupportedReason     string                       `json:"unsupported_reason,omitempty"`
	BlockedReason         string                       `json:"blocked_reason,omitempty"`
}

type MemoryFootprintSample struct {
	Target            string                       `json:"target"`
	EvidenceClass     MemoryFootprintEvidenceClass `json:"evidence_class"`
	Method            string                       `json:"method"`
	CurrentBytes      int64                        `json:"current_bytes,omitempty"`
	PeakBytes         int64                        `json:"peak_bytes,omitempty"`
	UnsupportedReason string                       `json:"unsupported_reason,omitempty"`
	BlockedReason     string                       `json:"blocked_reason,omitempty"`
}

func RuntimeMemoryBackendContract(target string) MemoryBackendContract {
	contract := MemoryBackendContract{
		Schema:             MemoryBackendContractSchemaV1,
		Target:             target,
		Operations:         RequiredMemoryBackendOperations(),
		MinAlignmentBytes:  SmallHeapAlignment,
		ReserveGranularity: 4096,
		CommitGranularity:  4096,
		NonClaims: []string{
			"no zero heap for all programs claim",
			"no all-target RSS parity claim",
			"no performance claim",
		},
	}
	switch target {
	case "linux-x64":
		contract.FootprintEvidenceClass = MemoryFootprintMeasured
		contract.FootprintMethod = "linux_proc_status"
	case "wasm32-wasi", "wasm32-web":
		contract.FootprintEvidenceClass = MemoryFootprintUnsupported
		contract.FootprintMethod = "linear_memory_adapter"
		contract.UnsupportedReason = "host RSS is unavailable for the current linear memory target boundary"
	default:
		contract.FootprintEvidenceClass = MemoryFootprintUnsupported
		contract.FootprintMethod = "adapter_not_implemented"
		contract.UnsupportedReason = "target footprint adapter is not implemented in the current memory backend contract"
	}
	return contract
}

func RequiredMemoryBackendOperations() []MemoryBackendOperation {
	return []MemoryBackendOperation{
		MemoryBackendReserve,
		MemoryBackendCommit,
		MemoryBackendDecommit,
		MemoryBackendRelease,
		MemoryBackendTrim,
		MemoryBackendFootprint,
	}
}

func (contract MemoryBackendContract) SupportsOperation(op MemoryBackendOperation) bool {
	for _, candidate := range contract.Operations {
		if candidate == op {
			return true
		}
	}
	return false
}

func ValidateMemoryBackendContract(contract MemoryBackendContract) error {
	if contract.Schema != MemoryBackendContractSchemaV1 {
		return fmt.Errorf("memory backend contract: schema = %q, want %s", contract.Schema, MemoryBackendContractSchemaV1)
	}
	if contract.Target == "" {
		return fmt.Errorf("memory backend contract: target is required")
	}
	if contract.MinAlignmentBytes <= 0 || !isPowerOfTwo(contract.MinAlignmentBytes) {
		return fmt.Errorf("memory backend contract %s: min alignment must be a positive power-of-two", contract.Target)
	}
	if contract.ReserveGranularity <= 0 || contract.CommitGranularity <= 0 {
		return fmt.Errorf("memory backend contract %s: reserve and commit granularity must be positive", contract.Target)
	}
	for _, op := range RequiredMemoryBackendOperations() {
		if !contract.SupportsOperation(op) {
			return fmt.Errorf("memory backend contract %s: missing operation %s", contract.Target, op)
		}
	}
	sample := MemoryFootprintSample{
		Target:            contract.Target,
		EvidenceClass:     contract.FootprintEvidenceClass,
		Method:            contract.FootprintMethod,
		UnsupportedReason: contract.UnsupportedReason,
	}
	if contract.FootprintEvidenceClass == MemoryFootprintMeasured || contract.FootprintEvidenceClass == MemoryFootprintEstimated {
		sample.CurrentBytes = 1
		sample.PeakBytes = 1
	}
	if err := ValidateMemoryFootprintSample(sample); err != nil {
		return fmt.Errorf("memory backend contract %s: footprint policy: %w", contract.Target, err)
	}
	return nil
}

func ValidateMemoryBackendAllocationEvidence(evidence MemoryBackendAllocationEvidence) error {
	if evidence.Schema != MemoryBackendAllocationEvidenceSchemaV1 {
		return fmt.Errorf("memory backend allocation evidence: schema = %q, want %s", evidence.Schema, MemoryBackendAllocationEvidenceSchemaV1)
	}
	if !isKnownMemoryBackendClass(evidence.BackendClass) {
		return fmt.Errorf("memory backend allocation evidence: backend_class %q is not supported", evidence.BackendClass)
	}
	if evidence.Adapter == "" {
		return fmt.Errorf("memory backend allocation evidence %s: adapter is required", evidence.BackendClass)
	}
	if evidence.RuntimePath == "" {
		return fmt.Errorf("memory backend allocation evidence %s: runtime_path is required", evidence.BackendClass)
	}
	if evidence.Method == "" {
		return fmt.Errorf("memory backend allocation evidence %s: method is required", evidence.BackendClass)
	}
	if evidence.ReserveBytes < 0 || evidence.CommitBytes < 0 || evidence.DecommitBytes < 0 || evidence.ReleaseBytes < 0 ||
		evidence.FootprintCurrentBytes < 0 || evidence.FootprintPeakBytes < 0 {
		return fmt.Errorf("memory backend allocation evidence %s: byte counts must be non-negative", evidence.BackendClass)
	}
	if evidence.FootprintPeakBytes < evidence.FootprintCurrentBytes {
		return fmt.Errorf("memory backend allocation evidence %s: footprint peak bytes must be >= current bytes", evidence.BackendClass)
	}
	switch evidence.EvidenceClass {
	case MemoryFootprintMeasured, MemoryFootprintEstimated:
		if evidence.BlockedReason != "" || evidence.UnsupportedReason != "" {
			return fmt.Errorf("memory backend allocation evidence %s: measured/estimated evidence must not include blocked or unsupported reason", evidence.BackendClass)
		}
		if len(evidence.Operations) == 0 {
			return fmt.Errorf("memory backend allocation evidence %s: operations are required for measured/estimated evidence", evidence.BackendClass)
		}
		for _, op := range evidence.Operations {
			if !isKnownMemoryBackendOperation(op) {
				return fmt.Errorf("memory backend allocation evidence %s: operation %q is not supported", evidence.BackendClass, op)
			}
		}
		if evidence.ReserveBytes > 0 && evidence.CommitBytes > evidence.ReserveBytes {
			return fmt.Errorf("memory backend allocation evidence %s: commit bytes must be <= reserve bytes", evidence.BackendClass)
		}
		if evidence.ReserveBytes > 0 && evidence.ReleaseBytes > evidence.ReserveBytes {
			return fmt.Errorf("memory backend allocation evidence %s: release bytes must be <= reserve bytes", evidence.BackendClass)
		}
	case MemoryFootprintUnsupported:
		if evidence.UnsupportedReason == "" {
			return fmt.Errorf("memory backend allocation evidence %s: unsupported_reason is required", evidence.BackendClass)
		}
		if evidence.BlockedReason != "" {
			return fmt.Errorf("memory backend allocation evidence %s: unsupported evidence must not include blocked_reason", evidence.BackendClass)
		}
		if evidence.ReserveBytes != 0 || evidence.CommitBytes != 0 || evidence.DecommitBytes != 0 || evidence.ReleaseBytes != 0 ||
			evidence.FootprintCurrentBytes != 0 || evidence.FootprintPeakBytes != 0 {
			return fmt.Errorf("memory backend allocation evidence %s: unsupported evidence must not include byte counts", evidence.BackendClass)
		}
	case MemoryFootprintBlocked:
		if evidence.BlockedReason == "" {
			return fmt.Errorf("memory backend allocation evidence %s: blocked_reason is required", evidence.BackendClass)
		}
		if evidence.UnsupportedReason != "" {
			return fmt.Errorf("memory backend allocation evidence %s: blocked evidence must not include unsupported_reason", evidence.BackendClass)
		}
		if evidence.ReserveBytes != 0 || evidence.CommitBytes != 0 || evidence.DecommitBytes != 0 || evidence.ReleaseBytes != 0 ||
			evidence.FootprintCurrentBytes != 0 || evidence.FootprintPeakBytes != 0 {
			return fmt.Errorf("memory backend allocation evidence %s: blocked evidence must not include byte counts", evidence.BackendClass)
		}
	default:
		return fmt.Errorf("memory backend allocation evidence %s: evidence_class %q is not supported", evidence.BackendClass, evidence.EvidenceClass)
	}
	return nil
}

func isKnownMemoryBackendClass(class MemoryBackendClass) bool {
	switch class {
	case MemoryBackendClassNone,
		MemoryBackendClassSmallHeap,
		MemoryBackendClassRegion,
		MemoryBackendClassLargeBackend,
		MemoryBackendClassExternal,
		MemoryBackendClassConservativeHeap,
		MemoryBackendClassUnknown:
		return true
	default:
		return false
	}
}

func isKnownMemoryBackendOperation(op MemoryBackendOperation) bool {
	switch op {
	case MemoryBackendReserve,
		MemoryBackendCommit,
		MemoryBackendDecommit,
		MemoryBackendRelease,
		MemoryBackendTrim,
		MemoryBackendFootprint:
		return true
	default:
		return false
	}
}

func ValidateMemoryFootprintSample(sample MemoryFootprintSample) error {
	if sample.Target == "" {
		return fmt.Errorf("memory footprint sample: target is required")
	}
	if sample.Method == "" {
		return fmt.Errorf("memory footprint sample %s: method is required", sample.Target)
	}
	switch sample.EvidenceClass {
	case MemoryFootprintMeasured, MemoryFootprintEstimated:
		if sample.BlockedReason != "" || sample.UnsupportedReason != "" {
			return fmt.Errorf("memory footprint sample %s: measured/estimated sample must not include blocked or unsupported reason", sample.Target)
		}
		if sample.CurrentBytes < 0 || sample.PeakBytes < 0 {
			return fmt.Errorf("memory footprint sample %s: byte counts must be non-negative", sample.Target)
		}
		if sample.PeakBytes < sample.CurrentBytes {
			return fmt.Errorf("memory footprint sample %s: peak bytes must be >= current bytes", sample.Target)
		}
	case MemoryFootprintUnsupported:
		if sample.UnsupportedReason == "" {
			return fmt.Errorf("memory footprint sample %s: unsupported_reason is required", sample.Target)
		}
		if sample.CurrentBytes != 0 || sample.PeakBytes != 0 {
			return fmt.Errorf("memory footprint sample %s: unsupported sample must not include byte counts", sample.Target)
		}
	case MemoryFootprintBlocked:
		if sample.BlockedReason == "" {
			return fmt.Errorf("memory footprint sample %s: blocked_reason is required", sample.Target)
		}
		if sample.CurrentBytes != 0 || sample.PeakBytes != 0 {
			return fmt.Errorf("memory footprint sample %s: blocked sample must not include byte counts", sample.Target)
		}
	default:
		return fmt.Errorf("memory footprint sample %s: evidence_class %q is not supported", sample.Target, sample.EvidenceClass)
	}
	return nil
}
