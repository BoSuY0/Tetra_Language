package main

import (
	"encoding/json"
	"os"
	"sort"
	"tetra_language/tools/internal/heaptelemetry"
	"tetra_language/tools/internal/rsstelemetry"
)

func collectTetraMetadata(name string, binaryPath string, optimizerArtifact string, runtimeHeap *runtimeHeapEvidence, runtimeRSS *runtimeRSSEvidence) *tetraMetadata {
	proof := binaryPath + ".proof.json"
	bounds := binaryPath + ".bounds.json"
	alloc := binaryPath + ".alloc.json"
	perf := binaryPath + ".perf.json"
	backend := binaryPath + ".backend.json"
	boundsLeft := readBoundsLeft(bounds)
	heap := readHeapAllocations(alloc)
	runtimeFeatures := readRuntimeFeatureMetadata(backend)
	return &tetraMetadata{
		ProofReport:                proof,
		BoundsReport:               bounds,
		AllocationReport:           alloc,
		PerfBlockerReport:          perf,
		BackendReport:              backend,
		BackendPath:                readBackendPath(backend),
		BackendBlockers:            readBackendBlockers(backend),
		RuntimeFeaturesRequired:    runtimeFeatures.Required,
		RuntimeFeaturesLinked:      runtimeFeatures.Linked,
		RuntimeFeaturesInitialized: runtimeFeatures.Initialized,
		RuntimeLazyInitBlockers:    runtimeFeatures.LazyInitBlockers,
		RuntimeFeatureEvidence:     runtimeFeatures.Evidence,
		RuntimeObjectPlan:          runtimeFeatures.ObjectPlan,
		BoundsLeft:                 boundsLeft,
		HeapAllocations:            heap,
		HeapReasonCodes:            readHeapReasonCodes(alloc),
		PerfBlockers:               readPerfBlockers(perf, name, heap),
		MemoryEvidence:             collectMemoryEvidence(alloc, runtimeHeap, runtimeRSS),
		OptimizerValidationMetadata: optimizerValidation{
			Status:   "current_supported_subset",
			Artifact: optimizerArtifact,
		},
	}
}

func missingTetraMetadata(binaryPath string, optimizerArtifact string) *tetraMetadata {
	return &tetraMetadata{
		ProofReport:                binaryPath + ".proof.json",
		BoundsReport:               binaryPath + ".bounds.json",
		AllocationReport:           binaryPath + ".alloc.json",
		PerfBlockerReport:          binaryPath + ".perf.json",
		BackendReport:              binaryPath + ".backend.json",
		BackendPath:                "fallback",
		RuntimeFeaturesRequired:    []string{},
		RuntimeFeaturesLinked:      []string{},
		RuntimeFeaturesInitialized: []string{},
		RuntimeLazyInitBlockers:    []string{},
		RuntimeFeatureEvidence:     blockedRuntimeFeatureEvidence("Tetra build failed before runtime feature artifacts were produced"),
		RuntimeObjectPlan:          blockedRuntimeObjectPlan("Tetra build failed before runtime object plan artifacts were produced"),
		HeapReasonCodes:            []string{},
		MemoryEvidence:             blockedMemoryEvidence("Tetra build failed before memory artifacts were produced"),
		OptimizerValidationMetadata: optimizerValidation{
			Status:   "missing_build_artifacts",
			Artifact: optimizerArtifact,
		},
	}
}

func collectMemoryEvidence(allocationReport string, runtimeHeap *runtimeHeapEvidence, runtimeRSS *runtimeRSSEvidence) *memoryEvidence {
	var report struct {
		Summary struct {
			BytesRequested uint64 `json:"bytes_requested"`
			BytesReserved  uint64 `json:"bytes_reserved"`
			BytesCommitted uint64 `json:"bytes_committed"`
			BytesReleased  uint64 `json:"bytes_released"`
			Domains        []struct {
				DomainID       string `json:"domain_id"`
				Kind           string `json:"kind"`
				RequestedBytes uint64 `json:"requested_bytes"`
				ReservedBytes  uint64 `json:"reserved_bytes"`
				CommittedBytes uint64 `json:"committed_bytes"`
				ReleasedBytes  uint64 `json:"released_bytes"`
				CurrentBytes   uint64 `json:"current_bytes"`
				PeakBytes      uint64 `json:"peak_bytes"`
				BytesCopied    uint64 `json:"bytes_copied"`
			} `json:"domains"`
		} `json:"summary"`
	}
	if err := readJSON(allocationReport, &report); err != nil {
		return blockedMemoryEvidence("allocation report was unavailable while collecting benchmark memory evidence")
	}
	domains := make([]memoryDomainByte, 0, len(report.Summary.Domains))
	var copied uint64
	for _, domain := range report.Summary.Domains {
		copied += domain.BytesCopied
		domains = append(domains, memoryDomainByte{
			DomainID:       domain.DomainID,
			Kind:           domain.Kind,
			RequestedBytes: domain.RequestedBytes,
			ReservedBytes:  domain.ReservedBytes,
			CommittedBytes: domain.CommittedBytes,
			ReleasedBytes:  domain.ReleasedBytes,
			CurrentBytes:   domain.CurrentBytes,
			PeakBytes:      domain.PeakBytes,
			BytesCopied:    domain.BytesCopied,
			EvidenceClass:  "allocation_report_estimate",
			Method:         "allocation_report_summary",
			SourceArtifact: allocationReport,
		})
	}
	domainEvidence := memoryMetric{
		EvidenceClass:  "allocation_report_estimate",
		Method:         "allocation_report_summary",
		SourceArtifact: allocationReport,
	}
	if len(domains) == 0 {
		domainEvidence = unsupportedMemoryMetric("not_collected", "allocation report summary does not include memory domains")
	}
	heapMetric := unsupportedMemoryMetric("not_collected", "Tier 1 runner did not collect runtime heap telemetry for this benchmark process")
	if runtimeHeap != nil {
		heapMetric = runtimeHeapMetric(runtimeHeap)
	}
	rssCurrent := unsupportedMemoryMetric("not_collected", "Tier 1 runner does not measure process RSS per benchmark row")
	rssPeak := unsupportedMemoryMetric("not_collected", "Tier 1 runner does not measure process RSS per benchmark row")
	if runtimeRSS != nil {
		rssPeak = runtimeRSSPeakMetric(runtimeRSS)
		if runtimeRSS.Sample.SampleCount > 0 {
			rssCurrent = runtimeRSSCurrentMetric(runtimeRSS)
		} else {
			rssCurrent = blockedMemoryMetric("RSS sampler did not observe a live current RSS sample for this benchmark process")
		}
	}
	return &memoryEvidence{
		Schema:              schemaBenchmarkMemoryV1,
		HeapAllocBytes:      heapMetric,
		BytesRequested:      allocationReportMetric(report.Summary.BytesRequested, allocationReport),
		BytesReserved:       allocationReportMetric(report.Summary.BytesReserved, allocationReport),
		BytesCommitted:      allocationReportMetric(report.Summary.BytesCommitted, allocationReport),
		BytesReleased:       allocationReportMetric(report.Summary.BytesReleased, allocationReport),
		BytesCopied:         allocationReportMetric(copied, allocationReport),
		RSSCurrent:          rssCurrent,
		RSSPeak:             rssPeak,
		DomainBytesEvidence: domainEvidence,
		DomainBytes:         domains,
	}
}

func runtimeHeapMetric(evidence *runtimeHeapEvidence) memoryMetric {
	sample := evidence.Sample
	return memoryMetric{
		Bytes:           sample.HeapPeakBytes,
		CurrentBytes:    sample.HeapCurrentBytes,
		PeakBytes:       sample.HeapPeakBytes,
		TotalAllocBytes: sample.HeapTotalAllocBytes,
		AllocationCount: sample.HeapAllocationCount,
		EvidenceClass:   "runtime_measured",
		Method:          heaptelemetry.MethodLinuxX64HeapTelemetryV1,
		SourceArtifact:  evidence.SourceArtifact,
	}
}

func runtimeRSSCurrentMetric(evidence *runtimeRSSEvidence) memoryMetric {
	sample := evidence.Sample
	return memoryMetric{
		Bytes:          sample.RSSCurrentBytes,
		CurrentBytes:   sample.RSSCurrentBytes,
		EvidenceClass:  "runtime_measured",
		Method:         rsstelemetry.MethodLinuxProcfsStatusVmRSSV1,
		SourceArtifact: evidence.SourceArtifact,
	}
}

func runtimeRSSPeakMetric(evidence *runtimeRSSEvidence) memoryMetric {
	sample := evidence.Sample
	return memoryMetric{
		Bytes:          sample.RSSPeakBytes,
		PeakBytes:      sample.RSSPeakBytes,
		EvidenceClass:  "runtime_measured",
		Method:         rsstelemetry.MethodLinuxWait4RusageMaxRSSV1,
		SourceArtifact: evidence.SourceArtifact,
	}
}

func allocationReportMetric(bytes uint64, sourceArtifact string) memoryMetric {
	return memoryMetric{
		Bytes:          bytes,
		EvidenceClass:  "allocation_report_estimate",
		Method:         "allocation_report_summary",
		SourceArtifact: sourceArtifact,
	}
}

func unsupportedMemoryMetric(method string, reason string) memoryMetric {
	return memoryMetric{
		EvidenceClass:     "unsupported",
		Method:            method,
		UnsupportedReason: reason,
	}
}

func blockedMemoryEvidence(reason string) *memoryEvidence {
	return &memoryEvidence{
		Schema:              schemaBenchmarkMemoryV1,
		HeapAllocBytes:      blockedMemoryMetric(reason),
		BytesRequested:      blockedMemoryMetric(reason),
		BytesReserved:       blockedMemoryMetric(reason),
		BytesCommitted:      blockedMemoryMetric(reason),
		BytesReleased:       blockedMemoryMetric(reason),
		BytesCopied:         blockedMemoryMetric(reason),
		RSSCurrent:          blockedMemoryMetric(reason),
		RSSPeak:             blockedMemoryMetric(reason),
		DomainBytesEvidence: blockedMemoryMetric(reason),
		DomainBytes:         []memoryDomainByte{},
	}
}

func blockedMemoryMetric(reason string) memoryMetric {
	return memoryMetric{
		EvidenceClass: "blocked",
		Method:        "missing_build_artifacts",
		BlockedReason: reason,
	}
}

func readBoundsLeft(path string) int {
	var report struct {
		Totals struct {
			Left int `json:"left"`
		} `json:"totals"`
	}
	if readJSON(path, &report) != nil {
		return 0
	}
	return report.Totals.Left
}

func readHeapAllocations(path string) int {
	var report struct {
		Totals struct {
			Heap int `json:"heap"`
		} `json:"totals"`
	}
	if readJSON(path, &report) != nil {
		return 0
	}
	return report.Totals.Heap
}

func readHeapReasonCodes(path string) []string {
	var report struct {
		Summary struct {
			HeapReasonCodes map[string]int `json:"heap_reason_codes"`
		} `json:"summary"`
	}
	if readJSON(path, &report) != nil || len(report.Summary.HeapReasonCodes) == 0 {
		return []string{}
	}
	codes := make([]string, 0, len(report.Summary.HeapReasonCodes))
	for code, count := range report.Summary.HeapReasonCodes {
		if code == "" || count <= 0 {
			continue
		}
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}

func readBackendPath(path string) string {
	var report struct {
		Summary struct {
			RegisterPath  int `json:"register_path"`
			StackFallback int `json:"stack_fallback"`
		} `json:"summary"`
	}
	if readJSON(path, &report) != nil {
		return "fallback"
	}
	if report.Summary.StackFallback > 0 {
		return "fallback"
	}
	if report.Summary.RegisterPath > 0 {
		return "register"
	}
	return "stack"
}

func readBackendBlockers(path string) []string {
	var report struct {
		Summary struct {
			Categories map[string]int `json:"categories"`
		} `json:"summary"`
	}
	if readJSON(path, &report) != nil {
		return nil
	}
	var blockers []string
	for category, count := range report.Summary.Categories {
		if count <= 0 || category == "register_path" {
			continue
		}
		blockers = append(blockers, category)
	}
	sort.Strings(blockers)
	return blockers
}

type runtimeFeatureMetadata struct {
	Required         []string
	Linked           []string
	Initialized      []string
	LazyInitBlockers []string
	Evidence         runtimeFeatureEvidence
	ObjectPlan       runtimeObjectPlan
}

func readRuntimeFeatureMetadata(path string) runtimeFeatureMetadata {
	var report struct {
		Summary struct {
			RuntimeFeaturesRequired      []string `json:"runtime_features_required"`
			RuntimeFeaturesLinked        []string `json:"runtime_features_linked"`
			RuntimeFeaturesInitialized   []string `json:"runtime_features_initialized"`
			RuntimeLazyInitBlockers      []string `json:"runtime_lazy_init_blockers"`
			RuntimeFeatureEvidenceClass  string   `json:"runtime_feature_evidence_class"`
			RuntimeFeatureEvidenceMethod string   `json:"runtime_feature_evidence_method"`
			RuntimeObjectPlan            struct {
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
			} `json:"runtime_object_plan"`
		} `json:"summary"`
	}
	if readJSON(path, &report) != nil {
		return runtimeFeatureMetadata{
			Required:         []string{},
			Linked:           []string{},
			Initialized:      []string{},
			LazyInitBlockers: []string{},
			Evidence:         blockedRuntimeFeatureEvidence("backend report was unavailable while collecting runtime feature evidence"),
			ObjectPlan:       blockedRuntimeObjectPlan("backend report was unavailable while collecting runtime object plan evidence"),
		}
	}
	objectPlan := report.Summary.RuntimeObjectPlan
	return runtimeFeatureMetadata{
		Required:         normalizedRuntimeFeatureSlice(report.Summary.RuntimeFeaturesRequired),
		Linked:           normalizedRuntimeFeatureSlice(report.Summary.RuntimeFeaturesLinked),
		Initialized:      normalizedRuntimeFeatureSlice(report.Summary.RuntimeFeaturesInitialized),
		LazyInitBlockers: normalizedRuntimeFeatureSlice(report.Summary.RuntimeLazyInitBlockers),
		Evidence: runtimeFeatureEvidence{
			EvidenceClass:  report.Summary.RuntimeFeatureEvidenceClass,
			Method:         report.Summary.RuntimeFeatureEvidenceMethod,
			SourceArtifact: path,
		},
		ObjectPlan: runtimeObjectPlan{
			EvidenceClass:                    objectPlan.EvidenceClass,
			EvidenceMethod:                   objectPlan.EvidenceMethod,
			RuntimeUsed:                      objectPlan.RuntimeUsed,
			RuntimeObjectLinked:              objectPlan.RuntimeObjectLinked,
			RuntimeObjectInitialized:         objectPlan.RuntimeObjectInitialized,
			RuntimeObjectOverride:            objectPlan.RuntimeObjectOverride,
			TimeOnlyRuntime:                  objectPlan.TimeOnlyRuntime,
			LinuxMinimalRuntime:              objectPlan.LinuxMinimalRuntime,
			RuntimeObjectFeaturesRequired:    normalizedRuntimeFeatureSlice(objectPlan.RuntimeObjectFeaturesRequired),
			RuntimeObjectFeaturesLinked:      normalizedRuntimeFeatureSlice(objectPlan.RuntimeObjectFeaturesLinked),
			RuntimeObjectFeaturesInitialized: normalizedRuntimeFeatureSlice(objectPlan.RuntimeObjectFeaturesInitialized),
			RuntimeObjectLazyInitBlockers:    normalizedRuntimeFeatureSlice(objectPlan.RuntimeObjectLazyInitBlockers),
		},
	}
}

func normalizedRuntimeFeatureSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func blockedRuntimeFeatureEvidence(reason string) runtimeFeatureEvidence {
	return runtimeFeatureEvidence{
		EvidenceClass: "blocked",
		Method:        "missing_build_artifacts",
		BlockedReason: reason,
	}
}

func blockedRuntimeObjectPlan(reason string) runtimeObjectPlan {
	return runtimeObjectPlan{
		EvidenceClass:                    "blocked",
		EvidenceMethod:                   "missing_build_artifacts",
		RuntimeObjectFeaturesRequired:    []string{},
		RuntimeObjectFeaturesLinked:      []string{},
		RuntimeObjectFeaturesInitialized: []string{},
		RuntimeObjectLazyInitBlockers:    []string{},
		BlockedReason:                    reason,
	}
}

func readPerfBlockers(path string, benchmark string, heapAllocations int) []string {
	var report struct {
		Benchmarks []struct {
			Benchmark   string   `json:"benchmark"`
			ReasonCodes []string `json:"reason_codes"`
		} `json:"benchmarks"`
	}
	if readJSON(path, &report) != nil {
		return nil
	}
	for _, row := range report.Benchmarks {
		if row.Benchmark == benchmark {
			return filterResolvedPerfBlockers(row.ReasonCodes, heapAllocations)
		}
	}
	return nil
}

func filterResolvedPerfBlockers(reasonCodes []string, heapAllocations int) []string {
	out := make([]string, 0, len(reasonCodes))
	for _, code := range reasonCodes {
		if heapAllocations == 0 && code == "allocation.local_call_heap_fallback" {
			continue
		}
		out = append(out, code)
	}
	return out
}

func readJSON(path string, out any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}
