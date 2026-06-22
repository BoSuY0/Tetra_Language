package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"tetra_language/tools/internal/heaptelemetry"
	"tetra_language/tools/internal/rsstelemetry"
)

type backendRuntimeFeatureSummary struct {
	Required         []string
	Linked           []string
	Initialized      []string
	LazyInitBlockers []string
	EvidenceClass    string
	EvidenceMethod   string
	ObjectPlan       runtimeObjectPlan
}

func validateRuntimeFeatureEvidence(
	name string,
	rowStatus string,
	metadata tetraMetadata,
	root string,
) error {
	prefix := "benchmark " + name + " runtime feature evidence"
	if rowStatus != "measured" {
		if metadata.RuntimeFeatureEvidence.EvidenceClass != "blocked" {
			return fmt.Errorf(
				"%s for %s row must be blocked, got %q",
				prefix,
				rowStatus,
				metadata.RuntimeFeatureEvidence.EvidenceClass,
			)
		}
		if metadata.RuntimeFeatureEvidence.Method != "missing_build_artifacts" {
			return fmt.Errorf(
				"%s blocked method = %q, want missing_build_artifacts",
				prefix,
				metadata.RuntimeFeatureEvidence.Method,
			)
		}
		if strings.TrimSpace(metadata.RuntimeFeatureEvidence.BlockedReason) == "" {
			return fmt.Errorf("%s blocked evidence requires blocked_reason", prefix)
		}
		if len(metadata.RuntimeFeaturesRequired) != 0 || len(metadata.RuntimeFeaturesLinked) != 0 ||
			len(metadata.RuntimeFeaturesInitialized) != 0 {
			return fmt.Errorf(
				"%s for %s row must not claim runtime feature arrays",
				prefix,
				rowStatus,
			)
		}
		if metadata.RuntimeObjectPlan.EvidenceClass != "blocked" {
			return fmt.Errorf(
				"%s runtime_object_plan for %s row must be blocked, got %q",
				prefix,
				rowStatus,
				metadata.RuntimeObjectPlan.EvidenceClass,
			)
		}
		if metadata.RuntimeObjectPlan.EvidenceMethod != "missing_build_artifacts" {
			return fmt.Errorf(
				"%s runtime_object_plan blocked method = %q, want missing_build_artifacts",
				prefix,
				metadata.RuntimeObjectPlan.EvidenceMethod,
			)
		}
		if strings.TrimSpace(metadata.RuntimeObjectPlan.BlockedReason) == "" {
			return fmt.Errorf(
				"%s runtime_object_plan blocked evidence requires blocked_reason",
				prefix,
			)
		}
		return validateRuntimeFeatureLabels(prefix+" blockers", metadata.RuntimeLazyInitBlockers)
	}
	evidence := metadata.RuntimeFeatureEvidence
	if evidence.EvidenceClass != runtimeFeatureEvidenceClass {
		return fmt.Errorf(
			"%s evidence_class = %q, want %q",
			prefix,
			evidence.EvidenceClass,
			runtimeFeatureEvidenceClass,
		)
	}
	if evidence.Method != runtimeFeatureEvidenceMethod {
		return fmt.Errorf(
			"%s method = %q, want %q",
			prefix,
			evidence.Method,
			runtimeFeatureEvidenceMethod,
		)
	}
	if strings.TrimSpace(evidence.SourceArtifact) == "" {
		return fmt.Errorf("%s source_artifact is required", prefix)
	}
	if evidence.SourceArtifact != metadata.BackendReport {
		return fmt.Errorf(
			"%s source_artifact = %q, want backend_report %q",
			prefix,
			evidence.SourceArtifact,
			metadata.BackendReport,
		)
	}
	summary, err := readBackendRuntimeFeatureSummary(root, evidence.SourceArtifact)
	if err != nil {
		return fmt.Errorf("%s source_artifact: %w", prefix, err)
	}
	if summary.EvidenceClass != runtimeFeatureEvidenceClass {
		return fmt.Errorf(
			"%s backend report evidence_class = %q, want %q",
			prefix,
			summary.EvidenceClass,
			runtimeFeatureEvidenceClass,
		)
	}
	if summary.EvidenceMethod != runtimeFeatureEvidenceMethod {
		return fmt.Errorf(
			"%s backend report evidence_method = %q, want %q",
			prefix,
			summary.EvidenceMethod,
			runtimeFeatureEvidenceMethod,
		)
	}
	if err := validateRuntimeObjectPlanEvidence(
		prefix,
		metadata.RuntimeObjectPlan,
		summary.ObjectPlan,
	); err != nil {
		return err
	}
	for _, item := range []struct {
		label string
		got   []string
		want  []string
	}{
		{"runtime_features_required", metadata.RuntimeFeaturesRequired, summary.Required},
		{"runtime_features_linked", metadata.RuntimeFeaturesLinked, summary.Linked},
		{"runtime_features_initialized", metadata.RuntimeFeaturesInitialized, summary.Initialized},
		{"runtime_lazy_init_blockers", metadata.RuntimeLazyInitBlockers, summary.LazyInitBlockers},
	} {
		got := normalizedRuntimeFeatureSlice(item.got)
		want := normalizedRuntimeFeatureSlice(item.want)
		if !stringSlicesEqual(got, want) {
			return fmt.Errorf("%s %s = %v, want backend report %v", prefix, item.label, got, want)
		}
		if err := validateRuntimeFeatureLabels(prefix+" "+item.label, got); err != nil {
			return err
		}
	}
	if containsString(metadata.RuntimeFeaturesLinked, "unknown_runtime") ||
		containsString(metadata.RuntimeFeaturesInitialized, "unknown_runtime") {
		return fmt.Errorf("%s unknown_runtime must not be linked or initialized", prefix)
	}
	if containsString(metadata.RuntimeFeaturesRequired, "unknown_runtime") &&
		!hasRuntimeBlockerPrefix(metadata.RuntimeLazyInitBlockers, "unknown_runtime_call:") {
		return fmt.Errorf("%s unknown_runtime requires unknown_runtime_call blocker", prefix)
	}
	return nil
}

type allocationHeapReasonReport struct {
	Summary struct {
		HeapReasonCodes              map[string]int `json:"heap_reason_codes"`
		MemoryBackendClasses         map[string]int `json:"memory_backend_classes"`
		MemoryBackendOperations      map[string]int `json:"memory_backend_operations"`
		MemoryBackendEvidenceClasses map[string]int `json:"memory_backend_evidence_classes"`
	} `json:"summary"`
	Functions []struct {
		Function    string `json:"name"`
		Allocations []struct {
			ID                    string                           `json:"id"`
			ValueID               string                           `json:"value_id"`
			Storage               string                           `json:"storage"`
			PlannedStorage        string                           `json:"planned_storage"`
			ActualLoweringStorage string                           `json:"actual_lowering_storage"`
			RuntimePath           string                           `json:"runtime_path"`
			BytesCommitted        int64                            `json:"bytes_committed"`
			BytesReleased         int64                            `json:"bytes_released"`
			ReasonCodes           []string                         `json:"reason_codes"`
			HeapReasonCodes       []string                         `json:"heap_reason_codes"`
			MemoryBackend         *allocationMemoryBackendEvidence `json:"memory_backend"`
		} `json:"allocations"`
	} `json:"functions"`
}

type allocationMemoryBackendEvidence struct {
	Schema                string   `json:"schema"`
	BackendClass          string   `json:"backend_class"`
	Adapter               string   `json:"adapter"`
	RuntimePath           string   `json:"runtime_path"`
	Operations            []string `json:"operations"`
	EvidenceClass         string   `json:"evidence_class"`
	Method                string   `json:"method"`
	ReserveBytes          int64    `json:"reserve_bytes"`
	CommitBytes           int64    `json:"commit_bytes"`
	DecommitBytes         int64    `json:"decommit_bytes"`
	ReleaseBytes          int64    `json:"release_bytes"`
	FootprintCurrentBytes int64    `json:"footprint_current_bytes"`
	FootprintPeakBytes    int64    `json:"footprint_peak_bytes"`
	UnsupportedReason     string   `json:"unsupported_reason"`
	BlockedReason         string   `json:"blocked_reason"`
}

func validateHeapReasonEvidence(
	name string,
	rowStatus string,
	metadata tetraMetadata,
	root string,
) error {
	prefix := "benchmark " + name + " heap reason evidence"
	if rowStatus != "measured" {
		if len(metadata.HeapReasonCodes) != 0 {
			return fmt.Errorf("%s for %s row must not claim heap reason codes", prefix, rowStatus)
		}
		return nil
	}
	report, err := readAllocationHeapReasonReport(root, metadata.AllocationReport)
	if err != nil {
		return fmt.Errorf("%s allocation_report: %w", prefix, err)
	}
	observed := map[string]int{}
	for _, fn := range report.Functions {
		for _, alloc := range fn.Allocations {
			id := alloc.ID
			if strings.TrimSpace(id) == "" {
				id = alloc.ValueID
			}
			label := fn.Function + "/" + id
			usesHeap := allocationReportRowUsesHeap(
				alloc.Storage,
				alloc.PlannedStorage,
				alloc.ActualLoweringStorage,
				alloc.RuntimePath,
			)
			if !usesHeap {
				if len(alloc.HeapReasonCodes) != 0 {
					return fmt.Errorf(
						"%s allocation %s has heap_reason_codes without heap storage",
						prefix,
						label,
					)
				}
				continue
			}
			if len(alloc.HeapReasonCodes) == 0 {
				return fmt.Errorf(
					"%s allocation %s heap allocation requires heap_reason_codes",
					prefix,
					label,
				)
			}
			codes, err := validateHeapReasonCodeSlice(
				prefix+" allocation "+label,
				alloc.HeapReasonCodes,
			)
			if err != nil {
				return err
			}
			reasonCodes, err := validateReasonCodeSlice(
				prefix+" allocation "+label+" reason_codes",
				alloc.ReasonCodes,
			)
			if err != nil {
				return err
			}
			for _, code := range codes {
				if !containsString(reasonCodes, code) {
					return fmt.Errorf(
						"%s allocation %s heap reason code %q missing from reason_codes",
						prefix,
						label,
						code,
					)
				}
				observed[code]++
			}
		}
	}
	summary := normalizedHeapReasonSummary(report.Summary.HeapReasonCodes)
	if !intMapsEqual(observed, summary) {
		return fmt.Errorf(
			"%s summary.heap_reason_codes = %v, want observed heap allocation codes %v",
			prefix,
			summary,
			observed,
		)
	}
	metadataCodes, err := validateHeapReasonCodeSlice(
		prefix+" metadata.heap_reason_codes",
		metadata.HeapReasonCodes,
	)
	if err != nil {
		return err
	}
	expectedMetadataCodes := sortedMapKeys(summary)
	if !stringSlicesEqual(metadataCodes, expectedMetadataCodes) {
		return fmt.Errorf(
			"%s metadata.heap_reason_codes = %v, want allocation report summary keys %v",
			prefix,
			metadataCodes,
			expectedMetadataCodes,
		)
	}
	return nil
}

func validateAllocationMemoryBackendEvidence(
	name string,
	rowStatus string,
	metadata tetraMetadata,
	root string,
) error {
	prefix := "benchmark " + name + " memory_backend evidence"
	if rowStatus != "measured" {
		return nil
	}
	report, err := readAllocationHeapReasonReport(root, metadata.AllocationReport)
	if err != nil {
		return fmt.Errorf("%s allocation_report: %w", prefix, err)
	}
	observedClasses := map[string]int{}
	observedOps := map[string]int{}
	observedEvidenceClasses := map[string]int{}
	allocationCount := 0
	var committed int64
	var released int64
	for _, fn := range report.Functions {
		for _, alloc := range fn.Allocations {
			allocationCount++
			id := alloc.ID
			if strings.TrimSpace(id) == "" {
				id = alloc.ValueID
			}
			label := fn.Function + "/" + id
			if alloc.MemoryBackend == nil {
				return fmt.Errorf("%s allocation %s missing memory_backend", prefix, label)
			}
			if err := validateAllocationMemoryBackendRow(
				prefix+" allocation "+label,
				alloc.RuntimePath,
				alloc.BytesCommitted,
				alloc.BytesReleased,
				*alloc.MemoryBackend,
			); err != nil {
				return err
			}
			observedClasses[alloc.MemoryBackend.BackendClass]++
			observedEvidenceClasses[alloc.MemoryBackend.EvidenceClass]++
			seenOps := map[string]bool{}
			for _, op := range alloc.MemoryBackend.Operations {
				if seenOps[op] {
					continue
				}
				seenOps[op] = true
				observedOps[op]++
			}
			committed += alloc.BytesCommitted
			released += alloc.BytesReleased
		}
	}
	if allocationCount == 0 {
		return nil
	}
	if !intMapsEqual(observedClasses, normalizedIntSummary(report.Summary.MemoryBackendClasses)) {
		return fmt.Errorf(
			"%s summary.memory_backend_classes = %v, want observed %v",
			prefix,
			normalizedIntSummary(report.Summary.MemoryBackendClasses),
			observedClasses,
		)
	}
	if !intMapsEqual(observedOps, normalizedIntSummary(report.Summary.MemoryBackendOperations)) {
		return fmt.Errorf(
			"%s summary.memory_backend_operations = %v, want observed %v",
			prefix,
			normalizedIntSummary(report.Summary.MemoryBackendOperations),
			observedOps,
		)
	}
	if !intMapsEqual(
		observedEvidenceClasses,
		normalizedIntSummary(report.Summary.MemoryBackendEvidenceClasses),
	) {
		return fmt.Errorf(
			"%s summary.memory_backend_evidence_classes = %v, want observed %v",
			prefix,
			normalizedIntSummary(report.Summary.MemoryBackendEvidenceClasses),
			observedEvidenceClasses,
		)
	}
	if metadata.MemoryEvidence != nil {
		if metadata.MemoryEvidence.BytesCommitted.EvidenceClass == "allocation_report_estimate" &&
			int64(metadata.MemoryEvidence.BytesCommitted.Bytes) != committed {
			return fmt.Errorf(
				"%s metadata memory_evidence.bytes_committed = %d, want allocation summary %d",
				prefix,
				metadata.MemoryEvidence.BytesCommitted.Bytes,
				committed,
			)
		}
		if metadata.MemoryEvidence.BytesReleased.EvidenceClass == "allocation_report_estimate" &&
			int64(metadata.MemoryEvidence.BytesReleased.Bytes) != released {
			return fmt.Errorf(
				"%s metadata memory_evidence.bytes_released = %d, want allocation summary %d",
				prefix,
				metadata.MemoryEvidence.BytesReleased.Bytes,
				released,
			)
		}
	}
	return nil
}

func validateAllocationMemoryBackendRow(
	prefix string,
	runtimePath string,
	bytesCommitted int64,
	bytesReleased int64,
	evidence allocationMemoryBackendEvidence,
) error {
	if evidence.Schema != "tetra.memory.backend-allocation.v1" {
		return fmt.Errorf(
			"%s schema = %q, want tetra.memory.backend-allocation.v1",
			prefix,
			evidence.Schema,
		)
	}
	if !allowedMemoryBackendClasses[evidence.BackendClass] {
		return fmt.Errorf("%s backend_class %q is not allowed", prefix, evidence.BackendClass)
	}
	if strings.TrimSpace(evidence.Adapter) == "" {
		return fmt.Errorf("%s adapter is required", prefix)
	}
	if strings.TrimSpace(evidence.RuntimePath) == "" {
		return fmt.Errorf("%s runtime_path is required", prefix)
	}
	if evidence.RuntimePath != runtimePath {
		return fmt.Errorf(
			"%s runtime_path = %q, want allocation runtime_path %q",
			prefix,
			evidence.RuntimePath,
			runtimePath,
		)
	}
	if strings.TrimSpace(evidence.Method) == "" {
		return fmt.Errorf("%s method is required", prefix)
	}
	if err := validateEvidenceClass(prefix, evidence.EvidenceClass); err != nil {
		return err
	}
	if evidence.ReserveBytes < 0 || evidence.CommitBytes < 0 || evidence.DecommitBytes < 0 ||
		evidence.ReleaseBytes < 0 ||
		evidence.FootprintCurrentBytes < 0 ||
		evidence.FootprintPeakBytes < 0 {
		return fmt.Errorf("%s memory_backend byte counts must be non-negative", prefix)
	}
	if evidence.FootprintPeakBytes < evidence.FootprintCurrentBytes {
		return fmt.Errorf("%s footprint_peak_bytes must be >= footprint_current_bytes", prefix)
	}
	if err := validateMemoryBackendClassForRuntimePath(
		prefix,
		runtimePath,
		evidence.BackendClass,
	); err != nil {
		return err
	}
	switch evidence.EvidenceClass {
	case "runtime_measured", "allocation_report_estimate":
		if evidence.UnsupportedReason != "" || evidence.BlockedReason != "" {
			return fmt.Errorf(
				"%s measured/estimated memory_backend must not include unsupported or blocked reason",
				prefix,
			)
		}
		if len(evidence.Operations) == 0 {
			return fmt.Errorf("%s measured/estimated memory_backend requires operations", prefix)
		}
		seen := map[string]bool{}
		for _, op := range evidence.Operations {
			if !allowedMemoryBackendOperations[op] {
				return fmt.Errorf("%s memory_backend operation %q is not allowed", prefix, op)
			}
			if seen[op] {
				return fmt.Errorf("%s memory_backend operation %q is duplicated", prefix, op)
			}
			seen[op] = true
		}
		if evidence.ReserveBytes > 0 && evidence.CommitBytes > evidence.ReserveBytes {
			return fmt.Errorf("%s commit_bytes must be <= reserve_bytes", prefix)
		}
		if evidence.ReserveBytes > 0 && evidence.ReleaseBytes > evidence.ReserveBytes {
			return fmt.Errorf("%s release_bytes must be <= reserve_bytes", prefix)
		}
		if bytesCommitted != evidence.CommitBytes {
			return fmt.Errorf(
				"%s bytes_committed = %d, want memory_backend commit_bytes %d",
				prefix,
				bytesCommitted,
				evidence.CommitBytes,
			)
		}
		if bytesReleased != evidence.ReleaseBytes {
			return fmt.Errorf(
				"%s bytes_released = %d, want memory_backend release_bytes %d",
				prefix,
				bytesReleased,
				evidence.ReleaseBytes,
			)
		}
	case "unsupported":
		if strings.TrimSpace(evidence.UnsupportedReason) == "" {
			return fmt.Errorf("%s unsupported memory_backend requires unsupported_reason", prefix)
		}
		if evidence.BlockedReason != "" {
			return fmt.Errorf(
				"%s unsupported memory_backend must not include blocked_reason",
				prefix,
			)
		}
		if evidence.ReserveBytes != 0 || evidence.CommitBytes != 0 || evidence.DecommitBytes != 0 ||
			evidence.ReleaseBytes != 0 ||
			evidence.FootprintCurrentBytes != 0 ||
			evidence.FootprintPeakBytes != 0 {
			return fmt.Errorf("%s unsupported memory_backend must not include byte counts", prefix)
		}
	case "blocked":
		if strings.TrimSpace(evidence.BlockedReason) == "" {
			return fmt.Errorf("%s blocked memory_backend requires blocked_reason", prefix)
		}
		if evidence.UnsupportedReason != "" {
			return fmt.Errorf(
				"%s blocked memory_backend must not include unsupported_reason",
				prefix,
			)
		}
		if evidence.ReserveBytes != 0 || evidence.CommitBytes != 0 || evidence.DecommitBytes != 0 ||
			evidence.ReleaseBytes != 0 ||
			evidence.FootprintCurrentBytes != 0 ||
			evidence.FootprintPeakBytes != 0 {
			return fmt.Errorf("%s blocked memory_backend must not include byte counts", prefix)
		}
		if bytesCommitted != 0 || bytesReleased != 0 {
			return fmt.Errorf(
				"%s blocked memory_backend must not report committed/released bytes",
				prefix,
			)
		}
	}
	return nil
}

func validateMemoryBackendClassForRuntimePath(
	prefix string,
	runtimePath string,
	backendClass string,
) error {
	switch runtimePath {
	case "process_bump_small_heap_v0", "per_core_small_heap", "small_heap_bump":
		if backendClass != "small_heap" {
			return fmt.Errorf(
				"%s backend_class = %q, want small_heap for %s",
				prefix,
				backendClass,
				runtimePath,
			)
		}
	case "large_mmap":
		if backendClass != "large_backend" {
			return fmt.Errorf(
				"%s backend_class = %q, want large_backend for large_mmap",
				prefix,
				backendClass,
			)
		}
	case "explicit_island", "scoped_single_mapping_v0", "region":
		if backendClass != "region" {
			return fmt.Errorf(
				"%s backend_class = %q, want region for %s",
				prefix,
				backendClass,
				runtimePath,
			)
		}
	case "stack_frame", "eliminated":
		if backendClass != "none" {
			return fmt.Errorf(
				"%s backend_class = %q, want none for %s",
				prefix,
				backendClass,
				runtimePath,
			)
		}
	case "external":
		if backendClass != "external" {
			return fmt.Errorf(
				"%s backend_class = %q, want external for external runtime path",
				prefix,
				backendClass,
			)
		}
	case "heap":
		if backendClass != "conservative_heap" {
			return fmt.Errorf(
				"%s backend_class = %q, want conservative_heap for heap runtime path",
				prefix,
				backendClass,
			)
		}
	case "unknown_conservative":
		if backendClass != "unknown" {
			return fmt.Errorf(
				"%s backend_class = %q, want unknown for unknown_conservative",
				prefix,
				backendClass,
			)
		}
	}
	return nil
}

func readAllocationHeapReasonReport(root string, path string) (allocationHeapReasonReport, error) {
	resolved, err := resolveExistingPath(root, path)
	if err != nil {
		return allocationHeapReasonReport{}, err
	}
	raw, err := os.ReadFile(resolved)
	if err != nil {
		return allocationHeapReasonReport{}, err
	}
	var report allocationHeapReasonReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return allocationHeapReasonReport{}, err
	}
	return report, nil
}

func allocationReportRowUsesHeap(
	storage string,
	planned string,
	actual string,
	runtimePath string,
) bool {
	for _, value := range []string{storage, planned, actual} {
		if value == "Heap" {
			return true
		}
	}
	switch runtimePath {
	case "heap", "process_bump_small_heap_v0", "per_core_small_heap", "large_mmap":
		return true
	default:
		return false
	}
}

func validateHeapReasonCodeSlice(prefix string, values []string) ([]string, error) {
	codes, err := validateReasonCodeSlice(prefix, values)
	if err != nil {
		return nil, err
	}
	for _, code := range codes {
		if !allowedHeapReasonCodes[code] {
			return nil, fmt.Errorf("%s contains unknown heap reason code %q", prefix, code)
		}
	}
	return codes, nil
}

func validateReasonCodeSlice(prefix string, values []string) ([]string, error) {
	if len(values) == 0 {
		return []string{}, nil
	}
	out := append([]string(nil), values...)
	sort.Strings(out)
	seen := map[string]bool{}
	for _, value := range out {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return nil, fmt.Errorf("%s contains empty reason code", prefix)
		}
		if trimmed != value {
			return nil, fmt.Errorf("%s contains untrimmed reason code %q", prefix, value)
		}
		if seen[value] {
			return nil, fmt.Errorf("%s contains duplicate reason code %q", prefix, value)
		}
		seen[value] = true
	}
	return out, nil
}

func normalizedHeapReasonSummary(values map[string]int) map[string]int {
	out := map[string]int{}
	for code, count := range values {
		if count <= 0 {
			continue
		}
		out[code] = count
	}
	return out
}

func normalizedIntSummary(values map[string]int) map[string]int {
	out := map[string]int{}
	for key, count := range values {
		if strings.TrimSpace(key) == "" || count <= 0 {
			continue
		}
		out[key] = count
	}
	return out
}

func intMapsEqual(a map[string]int, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for key, av := range a {
		if b[key] != av {
			return false
		}
	}
	return true
}

func sortedMapKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func readBackendRuntimeFeatureSummary(
	root string,
	path string,
) (backendRuntimeFeatureSummary, error) {
	resolved, err := resolveExistingPath(root, path)
	if err != nil {
		return backendRuntimeFeatureSummary{}, err
	}
	raw, err := os.ReadFile(resolved)
	if err != nil {
		return backendRuntimeFeatureSummary{}, err
	}
	var report struct {
		Summary struct {
			RuntimeFeaturesRequired      []string          `json:"runtime_features_required"`
			RuntimeFeaturesLinked        []string          `json:"runtime_features_linked"`
			RuntimeFeaturesInitialized   []string          `json:"runtime_features_initialized"`
			RuntimeLazyInitBlockers      []string          `json:"runtime_lazy_init_blockers"`
			RuntimeFeatureEvidenceClass  string            `json:"runtime_feature_evidence_class"`
			RuntimeFeatureEvidenceMethod string            `json:"runtime_feature_evidence_method"`
			RuntimeObjectPlan            runtimeObjectPlan `json:"runtime_object_plan"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		return backendRuntimeFeatureSummary{}, err
	}
	return backendRuntimeFeatureSummary{
		Required:         report.Summary.RuntimeFeaturesRequired,
		Linked:           report.Summary.RuntimeFeaturesLinked,
		Initialized:      report.Summary.RuntimeFeaturesInitialized,
		LazyInitBlockers: report.Summary.RuntimeLazyInitBlockers,
		EvidenceClass:    report.Summary.RuntimeFeatureEvidenceClass,
		EvidenceMethod:   report.Summary.RuntimeFeatureEvidenceMethod,
		ObjectPlan:       report.Summary.RuntimeObjectPlan,
	}, nil
}

func validateRuntimeObjectPlanEvidence(
	prefix string,
	got runtimeObjectPlan,
	want runtimeObjectPlan,
) error {
	if got.EvidenceClass != runtimeObjectPlanEvidenceClass {
		return fmt.Errorf(
			"%s runtime_object_plan evidence_class = %q, want %q",
			prefix,
			got.EvidenceClass,
			runtimeObjectPlanEvidenceClass,
		)
	}
	if got.EvidenceMethod != runtimeObjectPlanEvidenceMethod {
		return fmt.Errorf(
			"%s runtime_object_plan evidence_method = %q, want %q",
			prefix,
			got.EvidenceMethod,
			runtimeObjectPlanEvidenceMethod,
		)
	}
	if want.EvidenceClass != runtimeObjectPlanEvidenceClass {
		return fmt.Errorf(
			"%s backend report runtime_object_plan evidence_class = %q, want %q",
			prefix,
			want.EvidenceClass,
			runtimeObjectPlanEvidenceClass,
		)
	}
	if want.EvidenceMethod != runtimeObjectPlanEvidenceMethod {
		return fmt.Errorf(
			"%s backend report runtime_object_plan evidence_method = %q, want %q",
			prefix,
			want.EvidenceMethod,
			runtimeObjectPlanEvidenceMethod,
		)
	}
	for _, item := range []struct {
		label string
		got   bool
		want  bool
	}{
		{"runtime_used", got.RuntimeUsed, want.RuntimeUsed},
		{"runtime_object_linked", got.RuntimeObjectLinked, want.RuntimeObjectLinked},
		{"runtime_object_initialized", got.RuntimeObjectInitialized, want.RuntimeObjectInitialized},
		{"runtime_object_override", got.RuntimeObjectOverride, want.RuntimeObjectOverride},
		{"time_only_runtime", got.TimeOnlyRuntime, want.TimeOnlyRuntime},
		{"linux_minimal_runtime", got.LinuxMinimalRuntime, want.LinuxMinimalRuntime},
	} {
		if item.got != item.want {
			return fmt.Errorf(
				"%s runtime_object_plan %s = %v, want backend report %v",
				prefix,
				item.label,
				item.got,
				item.want,
			)
		}
	}
	for _, item := range []struct {
		label string
		got   []string
		want  []string
	}{
		{
			"runtime_object_features_required",
			got.RuntimeObjectFeaturesRequired,
			want.RuntimeObjectFeaturesRequired,
		},
		{
			"runtime_object_features_linked",
			got.RuntimeObjectFeaturesLinked,
			want.RuntimeObjectFeaturesLinked,
		},
		{
			"runtime_object_features_initialized",
			got.RuntimeObjectFeaturesInitialized,
			want.RuntimeObjectFeaturesInitialized,
		},
		{
			"runtime_object_lazy_init_blockers",
			got.RuntimeObjectLazyInitBlockers,
			want.RuntimeObjectLazyInitBlockers,
		},
	} {
		gotValues := normalizedRuntimeFeatureSlice(item.got)
		wantValues := normalizedRuntimeFeatureSlice(item.want)
		if !stringSlicesEqual(gotValues, wantValues) {
			return fmt.Errorf(
				"%s runtime_object_plan %s = %v, want backend report %v",
				prefix,
				item.label,
				gotValues,
				wantValues,
			)
		}
		if err := validateRuntimeFeatureLabels(
			prefix+" runtime_object_plan "+item.label,
			gotValues,
		); err != nil {
			return err
		}
	}
	if !got.RuntimeUsed &&
		(len(got.RuntimeObjectFeaturesLinked) != 0 || len(got.RuntimeObjectFeaturesInitialized) != 0) {
		return fmt.Errorf(
			"%s runtime_object_plan has linked/initialized features while runtime_used=false",
			prefix,
		)
	}
	return nil
}

func validateRuntimeFeatureLabels(prefix string, values []string) error {
	seen := map[string]bool{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return fmt.Errorf("%s contains empty runtime feature label", prefix)
		}
		if trimmed != value {
			return fmt.Errorf("%s contains untrimmed runtime feature label %q", prefix, value)
		}
		if seen[value] {
			return fmt.Errorf("%s contains duplicate runtime feature label %q", prefix, value)
		}
		seen[value] = true
	}
	return nil
}

func normalizedRuntimeFeatureSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func stringSlicesEqual(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func hasRuntimeBlockerPrefix(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func zeroHeapRequiredCategory(category string) bool {
	return zeroHeapRequiredCategories[category]
}

func validateZeroHeapRequirement(
	name string,
	category string,
	rowStatus string,
	metadata tetraMetadata,
	root string,
) error {
	if rowStatus != "measured" || !zeroHeapRequiredCategory(category) {
		return nil
	}
	prefix := fmt.Sprintf("benchmark %s zero-heap-required category %q", name, category)
	if metadata.HeapAllocations != 0 {
		return fmt.Errorf(
			"%s tetra_metadata.heap_allocations = %d, want 0",
			prefix,
			metadata.HeapAllocations,
		)
	}
	if metadata.MemoryEvidence == nil {
		return fmt.Errorf("%s missing memory evidence", prefix)
	}
	metric := metadata.MemoryEvidence.HeapAllocBytes
	if metric.EvidenceClass != "runtime_measured" {
		return fmt.Errorf(
			"%s heap_alloc_bytes evidence_class = %q, want runtime_measured",
			prefix,
			metric.EvidenceClass,
		)
	}
	if metric.TotalAllocBytes != 0 {
		return fmt.Errorf(
			"%s heap_alloc_bytes total_alloc_bytes = %d, want 0",
			prefix,
			metric.TotalAllocBytes,
		)
	}
	if metric.AllocationCount != 0 {
		return fmt.Errorf(
			"%s heap_alloc_bytes allocation_count = %d, want 0",
			prefix,
			metric.AllocationCount,
		)
	}
	if metric.Bytes != 0 {
		return fmt.Errorf("%s heap_alloc_bytes bytes = %d, want 0", prefix, metric.Bytes)
	}
	if metric.CurrentBytes != 0 {
		return fmt.Errorf(
			"%s heap_alloc_bytes current_bytes = %d, want 0",
			prefix,
			metric.CurrentBytes,
		)
	}
	if metric.PeakBytes != 0 {
		return fmt.Errorf("%s heap_alloc_bytes peak_bytes = %d, want 0", prefix, metric.PeakBytes)
	}

	resolved, err := resolveExistingPath(root, metric.SourceArtifact)
	if err != nil {
		return fmt.Errorf("%s heap_alloc_bytes source_artifact: %w", prefix, err)
	}
	sample, err := heaptelemetry.ReadFile(resolved, root)
	if err != nil {
		return fmt.Errorf("%s heap_alloc_bytes source_artifact: %w", prefix, err)
	}
	if sample.Program != name {
		return fmt.Errorf(
			"%s heap_alloc_bytes sidecar program = %q, want %q",
			prefix,
			sample.Program,
			name,
		)
	}
	if sample.HeapTotalAllocBytes != 0 {
		return fmt.Errorf(
			"%s heap_alloc_bytes sidecar total_alloc_bytes = %d, want 0",
			prefix,
			sample.HeapTotalAllocBytes,
		)
	}
	if sample.HeapAllocationCount != 0 {
		return fmt.Errorf(
			"%s heap_alloc_bytes sidecar allocation_count = %d, want 0",
			prefix,
			sample.HeapAllocationCount,
		)
	}
	if sample.HeapCurrentBytes != 0 {
		return fmt.Errorf(
			"%s heap_alloc_bytes sidecar current_bytes = %d, want 0",
			prefix,
			sample.HeapCurrentBytes,
		)
	}
	if sample.HeapPeakBytes != 0 {
		return fmt.Errorf(
			"%s heap_alloc_bytes sidecar peak_bytes = %d, want 0",
			prefix,
			sample.HeapPeakBytes,
		)
	}
	return nil
}

func validateMemoryEvidence(
	name string,
	rowStatus string,
	evidence *memoryEvidence,
	root string,
) error {
	if evidence == nil {
		return fmt.Errorf("benchmark %s missing memory evidence", name)
	}
	if evidence.Schema != schemaBenchmarkMemoryV1 {
		return fmt.Errorf(
			"benchmark %s memory evidence schema = %q, want %q",
			name,
			evidence.Schema,
			schemaBenchmarkMemoryV1,
		)
	}
	if err := validateHeapAllocBytesMetric(
		name,
		rowStatus,
		evidence.HeapAllocBytes,
		root,
	); err != nil {
		return err
	}
	if err := validateRSSMetrics(
		name,
		rowStatus,
		evidence.RSSCurrent,
		evidence.RSSPeak,
		root,
	); err != nil {
		return err
	}
	for _, item := range []struct {
		name            string
		metric          memoryMetric
		requireArtifact bool
	}{
		{"bytes_requested", evidence.BytesRequested, true},
		{"bytes_reserved", evidence.BytesReserved, true},
		{"bytes_committed", evidence.BytesCommitted, true},
		{"bytes_released", evidence.BytesReleased, true},
		{"bytes_copied", evidence.BytesCopied, true},
		{"domain_bytes_evidence", evidence.DomainBytesEvidence, false},
	} {
		if err := validateMemoryMetric(
			name,
			item.name,
			item.metric,
			root,
			false,
			item.requireArtifact,
		); err != nil {
			return err
		}
	}
	var runtimeDomainSample *heaptelemetry.Sample
	if evidence.DomainBytesEvidence.EvidenceClass == "runtime_measured" {
		sample, err := validateRuntimeMeasuredDomainBytesEvidence(
			name,
			evidence.DomainBytesEvidence,
			root,
		)
		if err != nil {
			return err
		}
		runtimeDomainSample = &sample
	}
	if len(evidence.DomainBytes) == 0 {
		switch evidence.DomainBytesEvidence.EvidenceClass {
		case "unsupported", "blocked":
		default:
			return fmt.Errorf(
				"benchmark %s memory evidence domain_bytes empty requires unsupported or blocked evidence",
				name,
			)
		}
	}
	for i, domain := range evidence.DomainBytes {
		if strings.TrimSpace(domain.DomainID) == "" {
			return fmt.Errorf(
				"benchmark %s memory evidence domain_bytes[%d] missing domain_id",
				name,
				i,
			)
		}
		if !allowedMemoryDomainKinds[domain.Kind] {
			return fmt.Errorf(
				"benchmark %s memory evidence domain_bytes[%d] kind %q is not allowed",
				name,
				i,
				domain.Kind,
			)
		}
		if err := validateEvidenceClass(
			"benchmark "+name+" memory evidence domain_bytes",
			domain.EvidenceClass,
		); err != nil {
			return err
		}
		if strings.TrimSpace(domain.Method) == "" {
			return fmt.Errorf(
				"benchmark %s memory evidence domain_bytes[%d] missing method",
				name,
				i,
			)
		}
		switch domain.EvidenceClass {
		case "allocation_report_estimate":
			if domain.Method != "allocation_report_summary" {
				return fmt.Errorf(
					("benchmark %s memory evidence domain_bytes[%d] allocation " +
						"estimate method = %q, want allocation_report_summary"),
					name,
					i,
					domain.Method,
				)
			}
			if err := requireExistingPath(root, domain.SourceArtifact); err != nil {
				return fmt.Errorf(
					"benchmark %s memory evidence domain_bytes[%d] source_artifact: %w",
					name,
					i,
					err,
				)
			}
		case "runtime_measured":
			if err := validateRuntimeMeasuredDomainByte(
				name,
				i,
				domain,
				evidence.DomainBytesEvidence,
				runtimeDomainSample,
				root,
			); err != nil {
				return err
			}
		}
	}
	if evidence.BytesCopied.EvidenceClass == "runtime_measured" {
		if err := validateRuntimeMeasuredBytesCopiedMetric(
			name,
			evidence.BytesCopied,
			evidence.DomainBytesEvidence,
			runtimeDomainSample,
			root,
		); err != nil {
			return err
		}
	}
	return nil
}

func validateRuntimeMeasuredDomainBytesEvidence(
	name string,
	metric memoryMetric,
	root string,
) (heaptelemetry.Sample, error) {
	prefix := "benchmark " + name + " memory evidence domain_bytes_evidence"
	if metric.Method != heaptelemetry.MethodLinuxX64HeapTelemetryV1 {
		return heaptelemetry.Sample{}, fmt.Errorf(
			"%s method = %q, want %q",
			prefix,
			metric.Method,
			heaptelemetry.MethodLinuxX64HeapTelemetryV1,
		)
	}
	if strings.TrimSpace(metric.SourceArtifact) == "" {
		return heaptelemetry.Sample{}, fmt.Errorf(
			"%s source_artifact is required for runtime_measured evidence",
			prefix,
		)
	}
	sample, err := readHeapDomainSidecar(name, prefix, metric.SourceArtifact, root)
	if err != nil {
		return heaptelemetry.Sample{}, err
	}
	if len(sample.DomainBytes) == 0 {
		return heaptelemetry.Sample{}, fmt.Errorf("%s sidecar domain_bytes is empty", prefix)
	}
	return sample, nil
}

func validateRuntimeMeasuredDomainByte(
	name string,
	index int,
	domain memoryDomainByte,
	aggregate memoryMetric,
	aggregateSample *heaptelemetry.Sample,
	root string,
) error {
	prefix := fmt.Sprintf("benchmark %s memory evidence domain_bytes[%d]", name, index)
	if aggregate.EvidenceClass != "runtime_measured" {
		return fmt.Errorf(
			"%s runtime_measured domain requires runtime_measured domain_bytes_evidence, got %q",
			prefix,
			aggregate.EvidenceClass,
		)
	}
	if domain.Method != heaptelemetry.MethodLinuxX64HeapTelemetryV1 {
		return fmt.Errorf(
			"%s method = %q, want %q",
			prefix,
			domain.Method,
			heaptelemetry.MethodLinuxX64HeapTelemetryV1,
		)
	}
	if strings.TrimSpace(domain.SourceArtifact) == "" {
		return fmt.Errorf("%s source_artifact is required for runtime_measured evidence", prefix)
	}
	if strings.TrimSpace(aggregate.SourceArtifact) != "" &&
		domain.SourceArtifact != aggregate.SourceArtifact {
		return fmt.Errorf(
			"%s source_artifact = %q, want domain_bytes_evidence source_artifact %q",
			prefix,
			domain.SourceArtifact,
			aggregate.SourceArtifact,
		)
	}
	if domain.ReleasedBytes != 0 {
		return fmt.Errorf(
			"%s released_bytes = %d, but heap telemetry sidecar domains do not carry released_bytes",
			prefix,
			domain.ReleasedBytes,
		)
	}
	sample, err := runtimeDomainSidecarSample(
		name,
		prefix,
		domain.SourceArtifact,
		aggregate,
		aggregateSample,
		root,
	)
	if err != nil {
		return err
	}
	observed, ok := heapSidecarDomain(sample, domain.DomainID, domain.Kind)
	if !ok {
		return fmt.Errorf("%s not found in heap telemetry sidecar", prefix)
	}
	if domain.Kind == "actor" {
		if !domain.ActorDomainFieldsSet {
			return fmt.Errorf(
				("%s actor runtime domain missing mailbox_current_bytes/mailbox_peak_bytes/" +
					"stack_live_bytes/stack_reserved_bytes/stack_retained_bytes/" +
					"stack_released_bytes/byte_budget/over_budget_count/backpressure_events"),
				prefix,
			)
		}
		if !observed.ActorDomainFieldsSet {
			return fmt.Errorf("%s sidecar actor domain missing mailbox_current_bytes/stack_live_bytes", prefix)
		}
	}
	for _, item := range []struct {
		label string
		got   uint64
		want  uint64
	}{
		{"requested_bytes", domain.RequestedBytes, observed.RequestedBytes},
		{"reserved_bytes", domain.ReservedBytes, observed.ReservedBytes},
		{"committed_bytes", domain.CommittedBytes, observed.CommittedBytes},
		{"current_bytes", domain.CurrentBytes, observed.CurrentBytes},
		{"peak_bytes", domain.PeakBytes, observed.PeakBytes},
		{"bytes_copied", domain.BytesCopied, observed.BytesCopied},
		{"mailbox_current_bytes", domain.MailboxCurrentBytes, observed.MailboxCurrentBytes},
		{"mailbox_peak_bytes", domain.MailboxPeakBytes, observed.MailboxPeakBytes},
		{"stack_live_bytes", domain.StackLiveBytes, observed.StackLiveBytes},
		{"stack_reserved_bytes", domain.StackReservedBytes, observed.StackReservedBytes},
		{"stack_retained_bytes", domain.StackRetainedBytes, observed.StackRetainedBytes},
		{"stack_released_bytes", domain.StackReleasedBytes, observed.StackReleasedBytes},
		{"byte_budget", domain.ByteBudget, observed.ByteBudget},
		{"over_budget_count", domain.OverBudgetCount, observed.OverBudgetCount},
		{"backpressure_events", domain.BackpressureEvents, observed.BackpressureEvents},
	} {
		if domain.Kind != "actor" && strings.Contains(item.label, "budget") {
			continue
		}
		if domain.Kind != "actor" && strings.Contains(item.label, "backpressure") {
			continue
		}
		if domain.Kind != "actor" && strings.HasPrefix(item.label, "mailbox_") {
			continue
		}
		if domain.Kind != "actor" && strings.HasPrefix(item.label, "stack_") {
			continue
		}
		if item.got != item.want {
			return fmt.Errorf(
				"%s %s = %d, want sidecar %d",
				prefix,
				item.label,
				item.got,
				item.want,
			)
		}
	}
	return nil
}

func validateRuntimeMeasuredBytesCopiedMetric(
	name string,
	metric memoryMetric,
	aggregate memoryMetric,
	aggregateSample *heaptelemetry.Sample,
	root string,
) error {
	prefix := "benchmark " + name + " memory evidence bytes_copied"
	if metric.Method != heaptelemetry.MethodLinuxX64HeapTelemetryV1 {
		return fmt.Errorf(
			"%s method = %q, want %q",
			prefix,
			metric.Method,
			heaptelemetry.MethodLinuxX64HeapTelemetryV1,
		)
	}
	if strings.TrimSpace(metric.SourceArtifact) == "" {
		return fmt.Errorf("%s source_artifact is required for runtime_measured evidence", prefix)
	}
	if aggregate.EvidenceClass == "runtime_measured" &&
		strings.TrimSpace(aggregate.SourceArtifact) != "" &&
		metric.SourceArtifact != aggregate.SourceArtifact {
		return fmt.Errorf(
			"%s source_artifact = %q, want domain_bytes_evidence source_artifact %q",
			prefix,
			metric.SourceArtifact,
			aggregate.SourceArtifact,
		)
	}
	sample, err := runtimeDomainSidecarSample(
		name,
		prefix,
		metric.SourceArtifact,
		aggregate,
		aggregateSample,
		root,
	)
	if err != nil {
		return err
	}
	var copied uint64
	for _, domain := range sample.DomainBytes {
		copied += domain.BytesCopied
	}
	if metric.Bytes != copied {
		return fmt.Errorf(
			"%s bytes = %d, want sidecar domain bytes_copied sum %d",
			prefix,
			metric.Bytes,
			copied,
		)
	}
	return nil
}

func runtimeDomainSidecarSample(
	name string,
	prefix string,
	sourceArtifact string,
	aggregate memoryMetric,
	aggregateSample *heaptelemetry.Sample,
	root string,
) (heaptelemetry.Sample, error) {
	if aggregateSample != nil && sourceArtifact == aggregate.SourceArtifact {
		return *aggregateSample, nil
	}
	return readHeapDomainSidecar(name, prefix, sourceArtifact, root)
}

func readHeapDomainSidecar(
	name string,
	prefix string,
	sourceArtifact string,
	root string,
) (heaptelemetry.Sample, error) {
	resolved, err := resolveExistingPath(root, sourceArtifact)
	if err != nil {
		return heaptelemetry.Sample{}, fmt.Errorf("%s source_artifact: %w", prefix, err)
	}
	sample, err := heaptelemetry.ReadFile(resolved, root)
	if err != nil {
		return heaptelemetry.Sample{}, fmt.Errorf("%s source_artifact: %w", prefix, err)
	}
	if sample.Program != name {
		return heaptelemetry.Sample{}, fmt.Errorf(
			"%s sidecar program = %q, want %q",
			prefix,
			sample.Program,
			name,
		)
	}
	return sample, nil
}

func heapSidecarDomain(
	sample heaptelemetry.Sample,
	domainID string,
	kind string,
) (heaptelemetry.DomainBytes, bool) {
	for _, domain := range sample.DomainBytes {
		if domain.DomainID == domainID && domain.Kind == kind {
			return domain, true
		}
	}
	return heaptelemetry.DomainBytes{}, false
}

func validateRSSMetrics(
	name string,
	rowStatus string,
	current memoryMetric,
	peak memoryMetric,
	root string,
) error {
	if rowStatus != "measured" {
		if err := validateMemoryMetric(name, "rss_current", current, root, true, false); err != nil {
			return err
		}
		if err := validateMemoryMetric(name, "rss_peak", peak, root, true, false); err != nil {
			return err
		}
		if current.EvidenceClass != "blocked" {
			return fmt.Errorf(
				"benchmark %s memory evidence rss_current for %s row must be blocked, got %q",
				name,
				rowStatus,
				current.EvidenceClass,
			)
		}
		if peak.EvidenceClass != "blocked" {
			return fmt.Errorf(
				"benchmark %s memory evidence rss_peak for %s row must be blocked, got %q",
				name,
				rowStatus,
				peak.EvidenceClass,
			)
		}
		return nil
	}

	peakSample, err := validateRSSPeakMetric(name, peak, root)
	if err != nil {
		return err
	}
	return validateRSSCurrentMetric(name, current, peak.SourceArtifact, peakSample, root)
}

func validateRSSPeakMetric(
	name string,
	metric memoryMetric,
	root string,
) (rsstelemetry.Sample, error) {
	prefix := "benchmark " + name + " memory evidence rss_peak"
	if err := validateEvidenceClass(prefix, metric.EvidenceClass); err != nil {
		return rsstelemetry.Sample{}, err
	}
	if metric.EvidenceClass != "runtime_measured" {
		return rsstelemetry.Sample{}, fmt.Errorf(
			"%s for measured linux Tetra row must be runtime_measured, got %q",
			prefix,
			metric.EvidenceClass,
		)
	}
	if metric.Method != rsstelemetry.MethodLinuxWait4RusageMaxRSSV1 {
		return rsstelemetry.Sample{}, fmt.Errorf(
			"%s method = %q, want %q",
			prefix,
			metric.Method,
			rsstelemetry.MethodLinuxWait4RusageMaxRSSV1,
		)
	}
	if strings.TrimSpace(metric.SourceArtifact) == "" {
		return rsstelemetry.Sample{}, fmt.Errorf(
			"%s source_artifact is required for runtime_measured evidence",
			prefix,
		)
	}
	resolved, err := resolveExistingPath(root, metric.SourceArtifact)
	if err != nil {
		return rsstelemetry.Sample{}, fmt.Errorf("%s source_artifact: %w", prefix, err)
	}
	sample, err := rsstelemetry.ReadFile(resolved, root)
	if err != nil {
		return rsstelemetry.Sample{}, fmt.Errorf("%s source_artifact: %w", prefix, err)
	}
	if sample.Program != name {
		return rsstelemetry.Sample{}, fmt.Errorf(
			"%s sidecar program = %q, want %q",
			prefix,
			sample.Program,
			name,
		)
	}
	if sample.RSSPeakBytes == 0 {
		return rsstelemetry.Sample{}, fmt.Errorf("%s sidecar rss_peak_bytes is zero", prefix)
	}
	if metric.Bytes != 0 && metric.Bytes != sample.RSSPeakBytes {
		return rsstelemetry.Sample{}, fmt.Errorf(
			"%s bytes = %d, want sidecar rss_peak_bytes %d",
			prefix,
			metric.Bytes,
			sample.RSSPeakBytes,
		)
	}
	if metric.PeakBytes != 0 && metric.PeakBytes != sample.RSSPeakBytes {
		return rsstelemetry.Sample{}, fmt.Errorf(
			"%s peak_bytes = %d, want sidecar rss_peak_bytes %d",
			prefix,
			metric.PeakBytes,
			sample.RSSPeakBytes,
		)
	}
	return sample, nil
}

func validateRSSCurrentMetric(
	name string,
	metric memoryMetric,
	peakArtifact string,
	peakSample rsstelemetry.Sample,
	root string,
) error {
	prefix := "benchmark " + name + " memory evidence rss_current"
	if err := validateEvidenceClass(prefix, metric.EvidenceClass); err != nil {
		return err
	}
	if metric.EvidenceClass == "blocked" {
		if strings.TrimSpace(metric.BlockedReason) == "" {
			return fmt.Errorf("%s blocked evidence requires blocked_reason", prefix)
		}
		if peakSample.SampleCount > 0 {
			return fmt.Errorf(
				"%s is blocked but RSS sidecar has %d live samples",
				prefix,
				peakSample.SampleCount,
			)
		}
		return nil
	}
	if metric.EvidenceClass != "runtime_measured" {
		return fmt.Errorf(
			"%s for measured linux Tetra row must be runtime_measured or blocked, got %q",
			prefix,
			metric.EvidenceClass,
		)
	}
	if metric.Method != rsstelemetry.MethodLinuxProcfsStatusVmRSSV1 {
		return fmt.Errorf(
			"%s method = %q, want %q",
			prefix,
			metric.Method,
			rsstelemetry.MethodLinuxProcfsStatusVmRSSV1,
		)
	}
	if strings.TrimSpace(metric.SourceArtifact) == "" {
		return fmt.Errorf("%s source_artifact is required for runtime_measured evidence", prefix)
	}
	if metric.SourceArtifact != peakArtifact {
		return fmt.Errorf(
			"%s source_artifact = %q, want rss_peak source_artifact %q",
			prefix,
			metric.SourceArtifact,
			peakArtifact,
		)
	}
	resolved, err := resolveExistingPath(root, metric.SourceArtifact)
	if err != nil {
		return fmt.Errorf("%s source_artifact: %w", prefix, err)
	}
	sample, err := rsstelemetry.ReadFile(resolved, root)
	if err != nil {
		return fmt.Errorf("%s source_artifact: %w", prefix, err)
	}
	if sample.Program != name {
		return fmt.Errorf("%s sidecar program = %q, want %q", prefix, sample.Program, name)
	}
	if sample.SampleCount == 0 {
		return fmt.Errorf(
			"%s cannot be runtime_measured because sidecar sample_count is zero",
			prefix,
		)
	}
	if metric.Bytes != 0 && metric.Bytes != sample.RSSCurrentBytes {
		return fmt.Errorf(
			"%s bytes = %d, want sidecar rss_current_bytes %d",
			prefix,
			metric.Bytes,
			sample.RSSCurrentBytes,
		)
	}
	if metric.CurrentBytes != 0 && metric.CurrentBytes != sample.RSSCurrentBytes {
		return fmt.Errorf(
			"%s current_bytes = %d, want sidecar rss_current_bytes %d",
			prefix,
			metric.CurrentBytes,
			sample.RSSCurrentBytes,
		)
	}
	return nil
}

func validateHeapAllocBytesMetric(
	name string,
	rowStatus string,
	metric memoryMetric,
	root string,
) error {
	prefix := "benchmark " + name + " memory evidence heap_alloc_bytes"
	if rowStatus != "measured" {
		if err := validateMemoryMetric(name, "heap_alloc_bytes", metric, root, false, false); err != nil {
			return err
		}
		if metric.EvidenceClass != "blocked" {
			return fmt.Errorf(
				"%s for %s row must be blocked, got %q",
				prefix,
				rowStatus,
				metric.EvidenceClass,
			)
		}
		return nil
	}
	if err := validateEvidenceClass(prefix, metric.EvidenceClass); err != nil {
		return err
	}
	if metric.EvidenceClass != "runtime_measured" {
		return fmt.Errorf(
			"%s for measured linux-x64 Tetra row must be runtime_measured, got %q",
			prefix,
			metric.EvidenceClass,
		)
	}
	if metric.Method != heaptelemetry.MethodLinuxX64HeapTelemetryV1 {
		return fmt.Errorf(
			"%s method = %q, want %q",
			prefix,
			metric.Method,
			heaptelemetry.MethodLinuxX64HeapTelemetryV1,
		)
	}
	if strings.TrimSpace(metric.SourceArtifact) == "" {
		return fmt.Errorf("%s source_artifact is required for runtime_measured evidence", prefix)
	}
	resolved, err := resolveExistingPath(root, metric.SourceArtifact)
	if err != nil {
		return fmt.Errorf("%s source_artifact: %w", prefix, err)
	}
	sample, err := heaptelemetry.ReadFile(resolved, root)
	if err != nil {
		return fmt.Errorf("%s source_artifact: %w", prefix, err)
	}
	if sample.Program != name {
		return fmt.Errorf("%s sidecar program = %q, want %q", prefix, sample.Program, name)
	}
	if metric.Bytes != 0 && metric.Bytes != sample.HeapPeakBytes {
		return fmt.Errorf(
			"%s bytes = %d, want sidecar heap_peak_bytes %d",
			prefix,
			metric.Bytes,
			sample.HeapPeakBytes,
		)
	}
	if metric.CurrentBytes != 0 && metric.CurrentBytes != sample.HeapCurrentBytes {
		return fmt.Errorf(
			"%s current_bytes = %d, want sidecar heap_current_bytes %d",
			prefix,
			metric.CurrentBytes,
			sample.HeapCurrentBytes,
		)
	}
	if metric.PeakBytes != 0 && metric.PeakBytes != sample.HeapPeakBytes {
		return fmt.Errorf(
			"%s peak_bytes = %d, want sidecar heap_peak_bytes %d",
			prefix,
			metric.PeakBytes,
			sample.HeapPeakBytes,
		)
	}
	if metric.TotalAllocBytes != 0 && metric.TotalAllocBytes != sample.HeapTotalAllocBytes {
		return fmt.Errorf(
			"%s total_alloc_bytes = %d, want sidecar heap_total_alloc_bytes %d",
			prefix,
			metric.TotalAllocBytes,
			sample.HeapTotalAllocBytes,
		)
	}
	if metric.AllocationCount != 0 && metric.AllocationCount != sample.HeapAllocationCount {
		return fmt.Errorf(
			"%s allocation_count = %d, want sidecar heap_allocation_count %d",
			prefix,
			metric.AllocationCount,
			sample.HeapAllocationCount,
		)
	}
	return nil
}

func validateMemoryMetric(
	name string,
	metricName string,
	metric memoryMetric,
	root string,
	rssMetric bool,
	requireArtifact bool,
) error {
	prefix := "benchmark " + name + " memory evidence " + metricName
	if err := validateEvidenceClass(prefix, metric.EvidenceClass); err != nil {
		return err
	}
	if strings.TrimSpace(metric.Method) == "" {
		return fmt.Errorf("%s missing method", prefix)
	}
	if rssMetric {
		if metric.EvidenceClass == "allocation_report_estimate" {
			return fmt.Errorf("%s must not use allocation_report_estimate as RSS evidence", prefix)
		}
		if metric.EvidenceClass == "runtime_measured" &&
			strings.EqualFold(metric.Method, "MemStats") {
			return fmt.Errorf("%s must not claim MemStats as runtime-measured process RSS", prefix)
		}
	}
	switch metric.EvidenceClass {
	case "allocation_report_estimate":
		if metric.Method != "allocation_report_summary" {
			return fmt.Errorf(
				"%s allocation estimate method = %q, want allocation_report_summary",
				prefix,
				metric.Method,
			)
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
		if requireArtifact && strings.TrimSpace(metric.SourceArtifact) == "" {
			return fmt.Errorf(
				"%s source_artifact is required for runtime_measured evidence",
				prefix,
			)
		}
		if requireArtifact {
			if err := requireExistingPath(root, metric.SourceArtifact); err != nil {
				return fmt.Errorf("%s source_artifact: %w", prefix, err)
			}
		}
	}
	return nil
}

func validateEvidenceClass(prefix string, evidenceClass string) error {
	if !allowedMemoryEvidenceClasses[evidenceClass] {
		return fmt.Errorf("%s evidence_class %q is not allowed", prefix, evidenceClass)
	}
	return nil
}
