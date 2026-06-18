package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"tetra_language/tools/internal/heaptelemetry"
	"tetra_language/tools/internal/rsstelemetry"
)

func TestValidateReportAcceptsCompleteP25Tier1Matrix(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	if err := ValidateReportBytes(raw, dir); err != nil {
		t.Fatalf("ValidateReportBytes: %v", err)
	}
}

func TestValidateReportRejectsMissingMatrixRow(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	results := report["results"].([]map[string]any)
	rows := results[0]["rows"].([]map[string]any)
	results[0]["rows"] = rows[:len(rows)-1]
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "missing matrix row") {
		t.Fatalf("ValidateReportBytes missing row = %v, want missing matrix row", err)
	}
}

func TestValidateReportRejectsMissingTetraMetadata(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	results := report["results"].([]map[string]any)
	rows := results[0]["rows"].([]map[string]any)
	delete(rows[0], "tetra_metadata")
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "tetra metadata") {
		t.Fatalf("ValidateReportBytes missing Tetra metadata = %v, want tetra metadata", err)
	}
}

func TestValidateReportRejectsMissingTetraMemoryEvidence(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	results := report["results"].([]map[string]any)
	rows := results[0]["rows"].([]map[string]any)
	metadata := rows[0]["tetra_metadata"].(map[string]any)
	delete(metadata, "memory_evidence")

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "memory evidence") {
		t.Fatalf(
			"ValidateReportBytes missing memory evidence = %v, want memory evidence rejection",
			err,
		)
	}
}

func TestValidateReportRejectsMissingRuntimeFeatureEvidence(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	results := report["results"].([]map[string]any)
	rows := results[0]["rows"].([]map[string]any)
	metadata := rows[0]["tetra_metadata"].(map[string]any)
	delete(metadata, "runtime_feature_evidence")

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "runtime feature evidence") {
		t.Fatalf(
			("ValidateReportBytes missing runtime feature evidence = %v, want " +
				"runtime feature evidence rejection"),
			err,
		)
	}
}

func TestValidateReportRejectsRuntimeFeatureMetadataMismatch(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	results := report["results"].([]map[string]any)
	rows := results[0]["rows"].([]map[string]any)
	metadata := rows[0]["tetra_metadata"].(map[string]any)
	metadata["runtime_features_required"] = []string{"actor_runtime"}

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "runtime_features_required") {
		t.Fatalf(
			"ValidateReportBytes runtime feature mismatch = %v, want runtime_features_required rejection",
			err,
		)
	}
}

func TestValidateReportRejectsMissingRuntimeObjectPlan(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	results := report["results"].([]map[string]any)
	rows := results[0]["rows"].([]map[string]any)
	metadata := rows[0]["tetra_metadata"].(map[string]any)
	delete(metadata, "runtime_object_plan")

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "runtime_object_plan") {
		t.Fatalf(
			"ValidateReportBytes missing runtime object plan = %v, want runtime_object_plan rejection",
			err,
		)
	}
}

func TestValidateReportRejectsRuntimeObjectPlanMismatch(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	results := report["results"].([]map[string]any)
	rows := results[0]["rows"].([]map[string]any)
	metadata := rows[0]["tetra_metadata"].(map[string]any)
	plan := metadata["runtime_object_plan"].(map[string]any)
	plan["runtime_object_linked"] = true

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "runtime_object_linked") {
		t.Fatalf(
			"ValidateReportBytes runtime object plan mismatch = %v, want runtime_object_linked rejection",
			err,
		)
	}
}

func TestValidateReportRejectsHeapAllocationWithoutReasonCodes(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	row := tetraRowForCategory(t, report, "slice sum")
	setAllocationReportFixture(t, dir, row, `{
  "kind": "allocation_plan",
  "totals": {"heap": 1},
  "summary": {"bytes_requested": 64, "bytes_reserved": 64, "heap_reason_codes": {}, "domains": []},
  "functions": [{
    "name": "main",
    "allocations": [{
      "id": "xs",
      "value_id": "alloc_intent:xs",
      "storage": "Heap",
      "planned_storage": "Heap",
      "actual_lowering_storage": "Heap",
      "runtime_path": "heap",
      "reason": "unknown call escape requires conservative heap fallback"
    }]
  }]
}`)

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "heap reason") {
		t.Fatalf(
			"ValidateReportBytes heap allocation without reason codes = %v, want heap reason rejection",
			err,
		)
	}
}

func TestValidateReportRejectsHeapReasonMetadataMismatch(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	row := tetraRowForCategory(t, report, "slice sum")
	setAllocationReportFixture(t, dir, row, `{
  "kind": "allocation_plan",
  "totals": {"heap": 1},
  "summary": {"bytes_requested": 64, "bytes_reserved": 64, "heap_reason_codes": {"heap.required_unknown_call": 1}, "domains": []},
  "functions": [{
    "name": "main",
    "allocations": [{
      "id": "xs",
      "value_id": "alloc_intent:xs",
      "storage": "Heap",
      "planned_storage": "Heap",
      "actual_lowering_storage": "Heap",
      "runtime_path": "heap",
      "reason_codes": ["heap.required_unknown_call"],
      "heap_reason_codes": ["heap.required_unknown_call"],
      "reason": "unknown call escape requires conservative heap fallback"
    }]
  }]
}`)
	metadata := row["tetra_metadata"].(map[string]any)
	metadata["heap_reason_codes"] = []string{"heap.required_escape_return"}

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "metadata.heap_reason_codes") {
		t.Fatalf(
			"ValidateReportBytes heap reason metadata mismatch = %v, want metadata heap reason rejection",
			err,
		)
	}
}

func TestValidateReportRejectsAllocationWithoutMemoryBackendEvidence(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	row := tetraRowForCategory(t, report, "slice sum")
	setAllocationReportFixture(t, dir, row, `{
  "kind": "allocation_plan",
  "totals": {"heap": 1},
  "summary": {
    "bytes_requested": 64,
    "bytes_reserved": 64,
    "bytes_committed": 64,
    "bytes_released": 64,
    "heap_reason_codes": {"heap.required_unknown_call": 1},
    "memory_backend_classes": {},
    "memory_backend_operations": {},
    "domains": []
  },
  "functions": [{
    "name": "main",
    "allocations": [{
      "id": "xs",
      "value_id": "alloc_intent:xs",
      "storage": "Heap",
      "planned_storage": "Heap",
      "actual_lowering_storage": "Heap",
      "runtime_path": "heap",
      "reason_codes": ["heap.required_unknown_call"],
      "heap_reason_codes": ["heap.required_unknown_call"],
      "reason": "unknown call escape requires conservative heap fallback"
    }]
  }]
}`)
	metadata := row["tetra_metadata"].(map[string]any)
	metadata["heap_reason_codes"] = []string{"heap.required_unknown_call"}

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "memory_backend") {
		t.Fatalf(
			("ValidateReportBytes allocation without memory backend evidence " +
				"= %v, want memory_backend rejection"),
			err,
		)
	}
}

func TestValidateReportRejectsRSSAllocationEstimateOverclaim(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	results := report["results"].([]map[string]any)
	rows := results[0]["rows"].([]map[string]any)
	metadata := rows[0]["tetra_metadata"].(map[string]any)
	memory := metadata["memory_evidence"].(map[string]any)
	memory["rss_current"] = map[string]any{
		"bytes":           4096,
		"evidence_class":  "allocation_report_estimate",
		"method":          "allocation_report_summary",
		"source_artifact": metadata["allocation_report"],
	}

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "rss_current") {
		t.Fatalf("ValidateReportBytes RSS estimate overclaim = %v, want rss_current rejection", err)
	}
}

func TestValidateReportRejectsRuntimeRSSPeakWithoutSidecarArtifact(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	results := report["results"].([]map[string]any)
	rows := results[0]["rows"].([]map[string]any)
	metadata := rows[0]["tetra_metadata"].(map[string]any)
	memory := metadata["memory_evidence"].(map[string]any)
	peak := memory["rss_peak"].(map[string]any)
	delete(peak, "source_artifact")

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "rss_peak") {
		t.Fatalf(
			"ValidateReportBytes RSS peak without sidecar artifact = %v, want rss_peak rejection",
			err,
		)
	}
}

func TestValidateReportRejectsRuntimeRSSMissingSidecarFile(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	results := report["results"].([]map[string]any)
	rows := results[0]["rows"].([]map[string]any)
	metadata := rows[0]["tetra_metadata"].(map[string]any)
	memory := metadata["memory_evidence"].(map[string]any)
	peak := memory["rss_peak"].(map[string]any)
	peak["source_artifact"] = filepath.Join("artifacts", "missing.rss.json")

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "rss_peak") {
		t.Fatalf("ValidateReportBytes missing RSS sidecar = %v, want rss_peak rejection", err)
	}
}

func TestValidateReportRejectsFakeRuntimeRSSMethods(t *testing.T) {
	for _, tc := range []struct {
		name   string
		metric string
		method string
	}{
		{name: "current memstats", metric: "rss_current", method: "MemStats"},
		{name: "current allocation report", metric: "rss_current", method: "allocation_report_summary"},
		{
			name:   "current heap method",
			metric: "rss_current",
			method: heaptelemetry.MethodLinuxX64HeapTelemetryV1,
		},
		{name: "peak memstats", metric: "rss_peak", method: "MemStats"},
		{name: "peak allocation report", metric: "rss_peak", method: "allocation_report_summary"},
		{
			name:   "peak heap method",
			metric: "rss_peak",
			method: heaptelemetry.MethodLinuxX64HeapTelemetryV1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			report := validTier1Report(t, dir)
			results := report["results"].([]map[string]any)
			rows := results[0]["rows"].([]map[string]any)
			metadata := rows[0]["tetra_metadata"].(map[string]any)
			memory := metadata["memory_evidence"].(map[string]any)
			metric := memory[tc.metric].(map[string]any)
			metric["method"] = tc.method

			raw, err := json.Marshal(report)
			if err != nil {
				t.Fatalf("marshal report: %v", err)
			}
			err = ValidateReportBytes(raw, dir)
			if err == nil || !strings.Contains(err.Error(), tc.metric) {
				t.Fatalf(
					"ValidateReportBytes fake RSS method %q = %v, want %s rejection",
					tc.method,
					err,
					tc.metric,
				)
			}
		})
	}
}

func TestValidateReportRejectsStaleRuntimeRSSSidecar(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	results := report["results"].([]map[string]any)
	rows := results[0]["rows"].([]map[string]any)
	metadata := rows[0]["tetra_metadata"].(map[string]any)
	memory := metadata["memory_evidence"].(map[string]any)
	peak := memory["rss_peak"].(map[string]any)
	peak["source_artifact"] = rssSidecarFixture(t, dir, "some_other_tetra", true)

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "program") {
		t.Fatalf("ValidateReportBytes stale RSS sidecar = %v, want program rejection", err)
	}
}

func TestValidateReportRejectsRuntimeRSSCurrentWithoutLiveSample(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	results := report["results"].([]map[string]any)
	rows := results[0]["rows"].([]map[string]any)
	metadata := rows[0]["tetra_metadata"].(map[string]any)
	memory := metadata["memory_evidence"].(map[string]any)
	sidecar := rssSidecarFixture(t, dir, rows[0]["name"].(string), false)
	current := memory["rss_current"].(map[string]any)
	current["source_artifact"] = sidecar
	peak := memory["rss_peak"].(map[string]any)
	peak["source_artifact"] = sidecar

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "rss_current") {
		t.Fatalf(
			"ValidateReportBytes RSS current without live sample = %v, want rss_current rejection",
			err,
		)
	}
}

func TestValidateReportRejectsRuntimeRSSMetricMismatch(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	results := report["results"].([]map[string]any)
	rows := results[0]["rows"].([]map[string]any)
	metadata := rows[0]["tetra_metadata"].(map[string]any)
	memory := metadata["memory_evidence"].(map[string]any)
	peak := memory["rss_peak"].(map[string]any)
	peak["peak_bytes"] = uint64(1)

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "rss_peak") {
		t.Fatalf("ValidateReportBytes RSS peak mismatch = %v, want rss_peak rejection", err)
	}
}

func TestValidateReportRejectsRSSPeakOverLocalBudget(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	policy := localRSSBudgetPolicyFixture(4096, 0, false, true)

	err = ValidateReportBytesWithRSSBudgetPolicy(raw, dir, policy)
	if err == nil || !strings.Contains(err.Error(), "rss budget") ||
		!strings.Contains(err.Error(), "integer loops") ||
		!strings.Contains(err.Error(), "rss_peak") {
		t.Fatalf(
			"ValidateReportBytesWithRSSBudgetPolicy over budget = %v, want rss budget rss_peak rejection",
			err,
		)
	}
}

func TestValidateReportAcceptsRSSPeakWithinLocalBudget(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	policy := localRSSBudgetPolicyFixture(8192, 0, false, true)

	if err := ValidateReportBytesWithRSSBudgetPolicy(raw, dir, policy); err != nil {
		t.Fatalf("ValidateReportBytesWithRSSBudgetPolicy within budget: %v", err)
	}
}

func TestValidateReportBlocksRSSBudgetWhenHostProfileDiffers(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	policy := localRSSBudgetPolicyFixture(1, 0, true, true)

	if err := ValidateReportBytesWithRSSBudgetPolicy(raw, dir, policy); err != nil {
		t.Fatalf(
			"ValidateReportBytesWithRSSBudgetPolicy host mismatch should block without failing: %v",
			err,
		)
	}
}

func TestValidateReportRejectsRSSBudgetPolicyWithoutLocalNonClaim(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	policy := localRSSBudgetPolicyFixture(8192, 0, false, false)

	err = ValidateReportBytesWithRSSBudgetPolicy(raw, dir, policy)
	if err == nil || !strings.Contains(err.Error(), "non_claim") {
		t.Fatalf(
			"ValidateReportBytesWithRSSBudgetPolicy missing local nonclaim = %v, want non_claim rejection",
			err,
		)
	}
}

func TestValidateReportRejectsGeneratedStyleRSSBudgetOverBudgetRow(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	policy := generatedStyleRSSBudgetPolicyFixture(t, report, func(category string) uint64 {
		if category == "integer loops" {
			return 4096
		}
		return 8192
	}, false)

	err = ValidateReportBytesWithRSSBudgetPolicy(raw, dir, policy)
	if err == nil || !strings.Contains(err.Error(), "rss budget") ||
		!strings.Contains(err.Error(), "integer loops") ||
		!strings.Contains(err.Error(), "rss_peak") {
		t.Fatalf(
			"generated-style RSS budget over budget = %v, want integer loops rss_peak rejection",
			err,
		)
	}
}

func TestValidateReportTreatsGeneratedStyleRSSBudgetHostMismatchAsNonApplicable(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	policy := generatedStyleRSSBudgetPolicyFixture(t, report, func(string) uint64 {
		return 1
	}, true)

	if err := ValidateReportBytesWithRSSBudgetPolicy(raw, dir, policy); err != nil {
		t.Fatalf(
			"generated-style host mismatch should be non-applicable, not cross-machine failure: %v",
			err,
		)
	}
}

func TestValidateReportRejectsRuntimeHeapEvidenceWithoutSidecarArtifact(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	results := report["results"].([]map[string]any)
	rows := results[0]["rows"].([]map[string]any)
	metadata := rows[0]["tetra_metadata"].(map[string]any)
	memory := metadata["memory_evidence"].(map[string]any)
	heap := memory["heap_alloc_bytes"].(map[string]any)
	delete(heap, "source_artifact")

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "heap_alloc_bytes") {
		t.Fatalf(
			"ValidateReportBytes heap without sidecar artifact = %v, want heap_alloc_bytes rejection",
			err,
		)
	}
}

func TestValidateReportRejectsRuntimeHeapEvidenceMissingSidecarFile(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	results := report["results"].([]map[string]any)
	rows := results[0]["rows"].([]map[string]any)
	metadata := rows[0]["tetra_metadata"].(map[string]any)
	memory := metadata["memory_evidence"].(map[string]any)
	heap := memory["heap_alloc_bytes"].(map[string]any)
	heap["source_artifact"] = filepath.Join("artifacts", "missing.heap.json")

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "heap_alloc_bytes") {
		t.Fatalf(
			"ValidateReportBytes missing heap sidecar = %v, want heap_alloc_bytes rejection",
			err,
		)
	}
}

func TestValidateReportRejectsFakeRuntimeHeapMethods(t *testing.T) {
	for _, method := range []string{"MemStats", "allocation_report_summary", "linux_proc_status"} {
		t.Run(method, func(t *testing.T) {
			dir := t.TempDir()
			report := validTier1Report(t, dir)
			results := report["results"].([]map[string]any)
			rows := results[0]["rows"].([]map[string]any)
			metadata := rows[0]["tetra_metadata"].(map[string]any)
			memory := metadata["memory_evidence"].(map[string]any)
			heap := memory["heap_alloc_bytes"].(map[string]any)
			heap["method"] = method

			raw, err := json.Marshal(report)
			if err != nil {
				t.Fatalf("marshal report: %v", err)
			}
			err = ValidateReportBytes(raw, dir)
			if err == nil || !strings.Contains(err.Error(), "heap_alloc_bytes") {
				t.Fatalf(
					"ValidateReportBytes fake heap method %q = %v, want heap_alloc_bytes rejection",
					method,
					err,
				)
			}
		})
	}
}

func TestValidateReportRejectsStaleRuntimeHeapSidecar(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	results := report["results"].([]map[string]any)
	rows := results[0]["rows"].([]map[string]any)
	metadata := rows[0]["tetra_metadata"].(map[string]any)
	memory := metadata["memory_evidence"].(map[string]any)
	heap := memory["heap_alloc_bytes"].(map[string]any)
	heap["source_artifact"] = heapSidecarFixture(t, dir, "some_other_tetra")

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "program") {
		t.Fatalf("ValidateReportBytes stale heap sidecar = %v, want program rejection", err)
	}
}

func TestValidateReportRejectsMeasuredTetraRowWithUnsupportedRuntimeHeapEvidence(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	results := report["results"].([]map[string]any)
	rows := results[0]["rows"].([]map[string]any)
	metadata := rows[0]["tetra_metadata"].(map[string]any)
	memory := metadata["memory_evidence"].(map[string]any)
	memory["heap_alloc_bytes"] = map[string]any{
		"evidence_class":     "unsupported",
		"method":             "not_collected",
		"unsupported_reason": "Tier 1 runner does not measure runtime heap bytes per benchmark process",
	}

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "heap_alloc_bytes") {
		t.Fatalf(
			"ValidateReportBytes measured unsupported heap = %v, want heap_alloc_bytes rejection",
			err,
		)
	}
}

func TestValidateReportAcceptsRuntimeMeasuredActorDomainBytesWithHeapSidecar(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	row := tetraRowForCategory(t, report, "actor ping-pong")
	setRuntimeMeasuredActorDomainEvidence(t, dir, row, "actor:pong")

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	if err := ValidateReportBytes(raw, dir); err != nil {
		t.Fatalf("ValidateReportBytes runtime actor domain bytes: %v", err)
	}
}

func TestValidateReportRejectsRuntimeMeasuredActorDomainMissingBudgetFields(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	row := tetraRowForCategory(t, report, "actor ping-pong")
	setRuntimeMeasuredActorDomainEvidence(t, dir, row, "actor:pong")
	metadata := row["tetra_metadata"].(map[string]any)
	memory := metadata["memory_evidence"].(map[string]any)
	domains := memory["domain_bytes"].([]map[string]any)
	delete(domains[0], "mailbox_current_bytes")

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "mailbox_current_bytes") {
		t.Fatalf(
			"ValidateReportBytes runtime actor domain missing mailbox fields = %v, want rejection",
			err,
		)
	}
}

func TestValidateReportRejectsRuntimeMeasuredDomainBytesMissingOrBadSourceArtifact(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(memory map[string]any)
		want string
	}{
		{
			name: "missing domain source",
			edit: func(memory map[string]any) {
				domains := memory["domain_bytes"].([]map[string]any)
				delete(domains[0], "source_artifact")
			},
			want: "source_artifact",
		},
		{
			name: "missing evidence source",
			edit: func(memory map[string]any) {
				evidence := memory["domain_bytes_evidence"].(map[string]any)
				delete(evidence, "source_artifact")
			},
			want: "domain_bytes_evidence",
		},
		{
			name: "bad sidecar path",
			edit: func(memory map[string]any) {
				evidence := memory["domain_bytes_evidence"].(map[string]any)
				evidence["source_artifact"] = filepath.Join("artifacts", "missing.heap.json")
				domains := memory["domain_bytes"].([]map[string]any)
				domains[0]["source_artifact"] = filepath.Join("artifacts", "missing.heap.json")
			},
			want: "source_artifact",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			report := validTier1Report(t, dir)
			row := tetraRowForCategory(t, report, "actor ping-pong")
			setRuntimeMeasuredActorDomainEvidence(t, dir, row, "actor:pong")
			metadata := row["tetra_metadata"].(map[string]any)
			memory := metadata["memory_evidence"].(map[string]any)
			tc.edit(memory)

			raw, err := json.Marshal(report)
			if err != nil {
				t.Fatalf("marshal report: %v", err)
			}
			err = ValidateReportBytes(raw, dir)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateReportBytes %s = %v, want %q rejection", tc.name, err, tc.want)
			}
		})
	}
}

func TestValidateReportRejectsZeroHeapRequiredRowWithHeapAllocationCount(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	row := tetraRowForCategory(t, report, "integer loops")
	setRuntimeHeapEvidence(t, dir, row, 0, 0, 0, 1)

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "zero-heap-required") ||
		!strings.Contains(err.Error(), "allocation_count") {
		t.Fatalf(
			("ValidateReportBytes zero-heap allocation count = %v, want zero-" +
				"heap-required allocation_count rejection"),
			err,
		)
	}
}

func TestValidateReportRejectsZeroHeapRequiredRowWithHeapTotalAllocBytes(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	row := tetraRowForCategory(t, report, "integer loops")
	setRuntimeHeapEvidence(t, dir, row, 0, 0, 64, 1)

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "zero-heap-required") ||
		!strings.Contains(err.Error(), "total_alloc_bytes") {
		t.Fatalf(
			("ValidateReportBytes zero-heap total alloc bytes = %v, want zero-" +
				"heap-required total_alloc_bytes rejection"),
			err,
		)
	}
}

func TestValidateReportRejectsZeroHeapRequiredRowWithMetadataHeapAllocations(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	row := tetraRowForCategory(t, report, "integer loops")
	metadata := row["tetra_metadata"].(map[string]any)
	metadata["heap_allocations"] = 1

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "zero-heap-required") ||
		!strings.Contains(err.Error(), "heap_allocations") {
		t.Fatalf(
			("ValidateReportBytes zero-heap metadata heap allocations = %v, " +
				"want zero-heap-required heap_allocations rejection"),
			err,
		)
	}
}

func TestValidateReportRejectsZeroHeapRequiredRowWithUnsupportedHeapEvidence(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	row := tetraRowForCategory(t, report, "integer loops")
	metadata := row["tetra_metadata"].(map[string]any)
	memory := metadata["memory_evidence"].(map[string]any)
	memory["heap_alloc_bytes"] = map[string]any{
		"evidence_class":     "unsupported",
		"method":             "not_collected",
		"unsupported_reason": "runtime heap sampler disabled",
	}

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "heap_alloc_bytes") {
		t.Fatalf(
			"ValidateReportBytes zero-heap unsupported heap = %v, want heap_alloc_bytes rejection",
			err,
		)
	}
}

func TestValidateReportRejectsZeroHeapRequiredRowWithBlockedHeapEvidence(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	row := tetraRowForCategory(t, report, "integer loops")
	metadata := row["tetra_metadata"].(map[string]any)
	memory := metadata["memory_evidence"].(map[string]any)
	memory["heap_alloc_bytes"] = map[string]any{
		"evidence_class": "blocked",
		"method":         "sampler_failed",
		"blocked_reason": "runtime heap sampler failed",
	}

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "heap_alloc_bytes") {
		t.Fatalf(
			"ValidateReportBytes zero-heap blocked heap = %v, want heap_alloc_bytes rejection",
			err,
		)
	}
}

func TestValidateReportAllowsExcludedRowWithRuntimeHeapAllocations(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	row := tetraRowForCategory(t, report, "slice sum")
	metadata := row["tetra_metadata"].(map[string]any)
	metadata["heap_allocations"] = 1
	setRuntimeHeapEvidence(t, dir, row, 64, 64, 64, 1)

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	if err := ValidateReportBytes(raw, dir); err != nil {
		t.Fatalf("ValidateReportBytes excluded heap row: %v", err)
	}
}

func TestValidateReportAcceptsBuildFailedTetraRowWithMissingBuildArtifacts(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	results := report["results"].([]map[string]any)
	rows := results[0]["rows"].([]map[string]any)
	tetra := rows[0]
	tetra["status"] = "build_failed"
	tetra["binary_path"] = filepath.Join("artifacts", "missing-tetra-binary")
	tetra["binary_size_bytes"] = int64(0)
	tetra["run_measurements_ms"] = nil
	tetra["median_runtime_ms"] = float64(0)
	tetra["error"] = "exit status 1"
	metadata := tetra["tetra_metadata"].(map[string]any)
	metadata["proof_report"] = filepath.Join("artifacts", "missing.proof.json")
	metadata["bounds_report"] = filepath.Join("artifacts", "missing.bounds.json")
	metadata["allocation_report"] = filepath.Join("artifacts", "missing.alloc.json")
	metadata["perf_blocker_report"] = filepath.Join("artifacts", "missing.perf.json")
	metadata["backend_report"] = filepath.Join("artifacts", "missing.backend.json")
	metadata["backend_path"] = "fallback"
	metadata["optimizer_validation_metadata"] = map[string]any{
		"status":   "missing_build_artifacts",
		"artifact": report["optimizer_validation"].(map[string]any)["artifact"],
	}
	metadata["runtime_features_required"] = []string{}
	metadata["runtime_features_linked"] = []string{}
	metadata["runtime_features_initialized"] = []string{}
	metadata["runtime_lazy_init_blockers"] = []string{}
	metadata["heap_reason_codes"] = []string{}
	metadata["runtime_feature_evidence"] = map[string]any{
		"evidence_class": "blocked",
		"method":         "missing_build_artifacts",
		"blocked_reason": "Tetra build failed before runtime feature artifacts were produced",
	}
	metadata["runtime_object_plan"] = map[string]any{
		"evidence_class":                      "blocked",
		"evidence_method":                     "missing_build_artifacts",
		"runtime_object_features_required":    []string{},
		"runtime_object_features_linked":      []string{},
		"runtime_object_features_initialized": []string{},
		"runtime_object_lazy_init_blockers":   []string{},
		"blocked_reason": ("Tetra build failed before runtime object plan " +
			"artifacts were produced"),
	}
	metadata["memory_evidence"] = blockedMemoryEvidenceFixture(
		"Tetra build failed before memory artifacts were produced",
	)

	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	if err := ValidateReportBytes(raw, dir); err != nil {
		t.Fatalf("ValidateReportBytes build_failed Tetra row: %v", err)
	}
}

func TestValidateReportRejectsWeakClaimsAndUnknownClassification(t *testing.T) {
	dir := t.TempDir()
	report := validTier1Report(t, dir)
	report["non_claims"] = []string{"Tetra is the fastest language."}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "fastest-language") {
		t.Fatalf("ValidateReportBytes weak non-claims = %v, want fastest-language rejection", err)
	}

	report = validTier1Report(t, dir)
	results := report["results"].([]map[string]any)
	results[0]["classification"] = "wins everything"
	raw, err = json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "classification") {
		t.Fatalf(
			"ValidateReportBytes unknown classification = %v, want classification rejection",
			err,
		)
	}
}

func TestValidateReportRejectsStaleGitCommitWhenRootIsGitWorktree(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(
		t,
		dir,
		"-c",
		"user.name=Tetra Test",
		"-c",
		"user.email=tetra@example.invalid",
		"commit",
		"--allow-empty",
		"-m",
		"init",
	)
	report := validTier1Report(t, dir)
	host := report["host"].(map[string]any)
	host["git_commit"] = "stale"
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	err = ValidateReportBytes(raw, dir)
	if err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("ValidateReportBytes stale git commit = %v, want stale rejection", err)
	}
}

func validTier1Report(t *testing.T, dir string) map[string]any {
	t.Helper()
	optimizer := fixture(
		t,
		dir,
		"artifacts/optimizer-validation.json",
		`{"status":"current_supported_subset"}`,
	)
	results := make([]map[string]any, 0, len(requiredP20Categories))
	for _, category := range requiredP20Categories {
		rows := make([]map[string]any, 0, len(requiredLanguages))
		for _, language := range requiredLanguages {
			name := slug(category) + "_" + language
			row := map[string]any{
				"name":                name,
				"category":            category,
				"language":            language,
				"status":              "measured",
				"compiler_version":    language + " compiler",
				"build_command":       []string{language, "build"},
				"run_command":         []string{filepath.Join("artifacts", name+".bin")},
				"source_path":         fixture(t, dir, "artifacts/"+name+".src", "source"),
				"binary_path":         fixture(t, dir, "artifacts/"+name+".bin", "binary"),
				"binary_size_bytes":   6,
				"compile_time_ms":     1.0,
				"run_measurements_ms": []float64{1, 2, 3},
				"median_runtime_ms":   2.0,
				"raw_output_artifacts": []string{
					fixture(t, dir, "artifacts/"+name+".stdout.txt", "stdout"),
					fixture(t, dir, "artifacts/"+name+".stderr.txt", ""),
				},
			}
			if language == "tetra" {
				heapCurrent := uint64(32)
				heapPeak := uint64(64)
				heapTotal := uint64(96)
				heapCount := uint64(2)
				if zeroHeapFixtureCategory(category) {
					heapCurrent = 0
					heapPeak = 0
					heapTotal = 0
					heapCount = 0
				}
				heapSidecar := heapSidecarFixtureWithTotals(
					t,
					dir,
					name,
					heapCurrent,
					heapPeak,
					heapTotal,
					heapCount,
				)
				backendReport := fixture(
					t,
					dir,
					"artifacts/"+name+".backend.json",
					`{"kind":"backend","summary":{"register_path":1,"stack_fallback":0,"runtime_features_required":[],"runtime_features_linked":[],"runtime_features_initialized":[],"runtime_lazy_init_blockers":[],"runtime_feature_evidence_class":"lowered_ir_static_plan","runtime_feature_evidence_method":"backend_report_lowered_ir_scan_v1","runtime_object_plan":{"evidence_class":"native_runtime_object_plan","evidence_method":"native_link_runtime_object_plan_v1","runtime_used":false,"runtime_object_linked":false,"runtime_object_initialized":false,"runtime_object_features_required":[],"runtime_object_features_linked":[],"runtime_object_features_initialized":[],"runtime_object_lazy_init_blockers":[]}}}`,
				)
				allocationReport := fixture(
					t,
					dir,
					"artifacts/"+name+".alloc.json",
					`{"kind":"allocation_plan","totals":{"heap":0},"summary":{"bytes_requested":128,"bytes_reserved":128,"bytes_committed":128,"bytes_released":128,"heap_reason_codes":{},"memory_backend_classes":{},"memory_backend_operations":{},"domains":[{"domain_id":"domain:process","kind":"process","requested_bytes":128,"reserved_bytes":128,"committed_bytes":128,"released_bytes":128}]},"functions":[]}`,
				)
				row["tetra_metadata"] = map[string]any{
					"proof_report": fixture(
						t,
						dir,
						"artifacts/"+name+".proof.json",
						`{"kind":"proof"}`,
					),
					"bounds_report": fixture(
						t,
						dir,
						"artifacts/"+name+".bounds.json",
						`{"kind":"bounds","totals":{"left":0}}`,
					),
					"allocation_report": allocationReport,
					"perf_blocker_report": fixture(
						t,
						dir,
						"artifacts/"+name+".perf.json",
						`{"kind":"perf","benchmarks":[]}`,
					),
					"backend_report":               backendReport,
					"backend_path":                 "register",
					"runtime_features_required":    []string{},
					"runtime_features_linked":      []string{},
					"runtime_features_initialized": []string{},
					"runtime_lazy_init_blockers":   []string{},
					"runtime_feature_evidence": map[string]any{
						"evidence_class":  "lowered_ir_static_plan",
						"method":          "backend_report_lowered_ir_scan_v1",
						"source_artifact": backendReport,
					},
					"runtime_object_plan": map[string]any{
						"evidence_class":                      "native_runtime_object_plan",
						"evidence_method":                     "native_link_runtime_object_plan_v1",
						"runtime_used":                        false,
						"runtime_object_linked":               false,
						"runtime_object_initialized":          false,
						"runtime_object_features_required":    []string{},
						"runtime_object_features_linked":      []string{},
						"runtime_object_features_initialized": []string{},
						"runtime_object_lazy_init_blockers":   []string{},
					},
					"bounds_left":       0,
					"heap_allocations":  0,
					"heap_reason_codes": []string{},
					"perf_blockers":     []string{},
					"optimizer_validation_metadata": map[string]any{
						"status":   "current_supported_subset",
						"artifact": optimizer,
					},
					"memory_evidence": memoryEvidenceFixture(name,
						allocationReport,
						heapSidecar,
						rssSidecarFixture(t, dir, name, true),
						heapCurrent,
						heapPeak,
						heapTotal,
						heapCount,
					),
				}
			}
			rows = append(rows, row)
		}
		results = append(results, map[string]any{
			"category":              category,
			"algorithm_id":          "p25." + slug(category),
			"input_description":     "deterministic local Tier 1 fixture",
			"classification":        "comparable",
			"classification_reason": "fixture rows are within the comparable threshold",
			"rows":                  rows,
		})
	}
	gitCommit := "abcdef"
	if head, ok := currentGitHead(dir); ok {
		gitCommit = head
	}
	return map[string]any{
		"schema":       schemaLocalBenchmarkTier1,
		"scope":        scopeP25RealLocalBenchmark,
		"generated_at": "2026-06-03T00:00:00Z",
		"host": map[string]any{
			"goos":       "linux",
			"goarch":     "amd64",
			"cpus":       8,
			"target_cpu": "test cpu",
			"git_commit": gitCommit,
		},
		"policy": map[string]any{
			"tier":                 "tier1_local_benchmark_evidence",
			"comparable_threshold": 0.20,
			"iterations":           3,
		},
		"non_claims": []string{
			"no fastest-language claim",
			"no official benchmark claim",
			"no cross-machine claim",
			"no TechEmpower claim",
			"no production claim",
		},
		"optimizer_validation": map[string]any{
			"status":   "current_supported_subset",
			"artifact": optimizer,
		},
		"results": results,
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, string(out))
	}
}

func memoryEvidenceFixture(
	name string,
	allocationReport string,
	heapSidecar string,
	rssSidecar string,
	heapCurrent uint64,
	heapPeak uint64,
	heapTotal uint64,
	heapCount uint64,
) map[string]any {
	return map[string]any{
		"schema": "tetra.local_benchmark.memory_evidence.v1",
		"heap_alloc_bytes": map[string]any{
			"bytes":             heapPeak,
			"current_bytes":     heapCurrent,
			"peak_bytes":        heapPeak,
			"total_alloc_bytes": heapTotal,
			"allocation_count":  heapCount,
			"evidence_class":    "runtime_measured",
			"method":            heaptelemetry.MethodLinuxX64HeapTelemetryV1,
			"source_artifact":   heapSidecar,
		},
		"bytes_requested": map[string]any{
			"bytes":           128,
			"evidence_class":  "allocation_report_estimate",
			"method":          "allocation_report_summary",
			"source_artifact": allocationReport,
		},
		"bytes_reserved": map[string]any{
			"bytes":           128,
			"evidence_class":  "allocation_report_estimate",
			"method":          "allocation_report_summary",
			"source_artifact": allocationReport,
		},
		"bytes_committed": map[string]any{
			"bytes":           128,
			"evidence_class":  "allocation_report_estimate",
			"method":          "allocation_report_summary",
			"source_artifact": allocationReport,
		},
		"bytes_released": map[string]any{
			"bytes":           128,
			"evidence_class":  "allocation_report_estimate",
			"method":          "allocation_report_summary",
			"source_artifact": allocationReport,
		},
		"bytes_copied": map[string]any{
			"bytes":           0,
			"evidence_class":  "allocation_report_estimate",
			"method":          "allocation_report_summary",
			"source_artifact": allocationReport,
		},
		"rss_current": map[string]any{
			"bytes":           4096,
			"current_bytes":   4096,
			"evidence_class":  "runtime_measured",
			"method":          rsstelemetry.MethodLinuxProcfsStatusVmRSSV1,
			"source_artifact": rssSidecar,
		},
		"rss_peak": map[string]any{
			"bytes":           8192,
			"peak_bytes":      8192,
			"evidence_class":  "runtime_measured",
			"method":          rsstelemetry.MethodLinuxWait4RusageMaxRSSV1,
			"source_artifact": rssSidecar,
		},
		"domain_bytes_evidence": map[string]any{
			"evidence_class":  "allocation_report_estimate",
			"method":          "allocation_report_summary",
			"source_artifact": allocationReport,
		},
		"domain_bytes": []map[string]any{
			{
				"domain_id":       "domain:process",
				"kind":            "process",
				"requested_bytes": 128,
				"reserved_bytes":  128,
				"committed_bytes": 128,
				"released_bytes":  128,
				"evidence_class":  "allocation_report_estimate",
				"method":          "allocation_report_summary",
				"source_artifact": allocationReport,
			},
		},
	}
}

func heapSidecarFixture(t *testing.T, dir string, name string) string {
	t.Helper()
	return heapSidecarFixtureWithTotals(t, dir, name, 32, 64, 96, 2)
}

func heapSidecarFixtureWithTotals(
	t *testing.T,
	dir string,
	name string,
	current uint64,
	peak uint64,
	total uint64,
	count uint64,
) string {
	t.Helper()
	sample := map[string]any{
		"schema":                 heaptelemetry.Schema,
		"target":                 heaptelemetry.TargetLinuxX64,
		"method":                 heaptelemetry.MethodLinuxX64HeapTelemetryV1,
		"program":                name,
		"pid":                    1234,
		"exit_status":            0,
		"heap_current_bytes":     current,
		"heap_peak_bytes":        peak,
		"heap_total_alloc_bytes": total,
		"heap_allocation_count":  count,
		"bytes_requested":        total,
		"bytes_reserved":         uint64(0),
		"notes":                  []string{"test fixture"},
	}
	if peak > 0 {
		sample["bytes_reserved"] = uint64(4096)
	}
	if count > 0 {
		sample["allocation_paths"] = map[string]uint64{"small_heap_bump": count}
	}
	if current > 0 || peak > 0 {
		sample["domain_bytes"] = []map[string]any{{
			"domain_id":     "domain:process",
			"kind":          "process",
			"current_bytes": current,
			"peak_bytes":    peak,
		}}
	}
	raw, err := json.Marshal(sample)
	if err != nil {
		t.Fatalf("marshal heap sidecar fixture: %v", err)
	}
	return fixture(t, dir, "artifacts/"+name+".heap.json", string(raw))
}

func setRuntimeHeapEvidence(
	t *testing.T,
	dir string,
	row map[string]any,
	current uint64,
	peak uint64,
	total uint64,
	count uint64,
) {
	t.Helper()
	name := row["name"].(string)
	metadata := row["tetra_metadata"].(map[string]any)
	memory := metadata["memory_evidence"].(map[string]any)
	sidecar := heapSidecarFixtureWithTotals(t, dir, name, current, peak, total, count)
	memory["heap_alloc_bytes"] = map[string]any{
		"bytes":             peak,
		"current_bytes":     current,
		"peak_bytes":        peak,
		"total_alloc_bytes": total,
		"allocation_count":  count,
		"evidence_class":    "runtime_measured",
		"method":            heaptelemetry.MethodLinuxX64HeapTelemetryV1,
		"source_artifact":   sidecar,
	}
}

func setRuntimeMeasuredActorDomainEvidence(
	t *testing.T,
	dir string,
	row map[string]any,
	domainID string,
) {
	t.Helper()
	name := row["name"].(string)
	sidecar := heapSidecarFixtureWithDomain(t, dir, name, domainID)
	metadata := row["tetra_metadata"].(map[string]any)
	memory := metadata["memory_evidence"].(map[string]any)
	memory["heap_alloc_bytes"] = map[string]any{
		"bytes":             uint64(192),
		"current_bytes":     uint64(128),
		"peak_bytes":        uint64(192),
		"total_alloc_bytes": uint64(256),
		"allocation_count":  uint64(4),
		"evidence_class":    "runtime_measured",
		"method":            heaptelemetry.MethodLinuxX64HeapTelemetryV1,
		"source_artifact":   sidecar,
	}
	memory["bytes_copied"] = map[string]any{
		"bytes":           uint64(64),
		"evidence_class":  "runtime_measured",
		"method":          heaptelemetry.MethodLinuxX64HeapTelemetryV1,
		"source_artifact": sidecar,
	}
	memory["domain_bytes_evidence"] = map[string]any{
		"evidence_class":  "runtime_measured",
		"method":          heaptelemetry.MethodLinuxX64HeapTelemetryV1,
		"source_artifact": sidecar,
	}
	memory["domain_bytes"] = []map[string]any{
		{
			"domain_id":             domainID,
			"kind":                  "actor",
			"requested_bytes":       uint64(256),
			"reserved_bytes":        uint64(512),
			"committed_bytes":       uint64(384),
			"current_bytes":         uint64(128),
			"peak_bytes":            uint64(192),
			"bytes_copied":          uint64(64),
			"mailbox_current_bytes": uint64(128),
			"mailbox_peak_bytes":    uint64(192),
			"byte_budget":           uint64(1024),
			"over_budget_count":     uint64(3),
			"backpressure_events":   uint64(4),
			"evidence_class":        "runtime_measured",
			"method":                heaptelemetry.MethodLinuxX64HeapTelemetryV1,
			"source_artifact":       sidecar,
		},
	}
}

func heapSidecarFixtureWithDomain(t *testing.T, dir string, name string, domainID string) string {
	t.Helper()
	sample := map[string]any{
		"schema":                 heaptelemetry.Schema,
		"target":                 heaptelemetry.TargetLinuxX64,
		"method":                 heaptelemetry.MethodLinuxX64HeapTelemetryV1,
		"program":                name,
		"pid":                    1234,
		"exit_status":            0,
		"heap_current_bytes":     uint64(128),
		"heap_peak_bytes":        uint64(192),
		"heap_total_alloc_bytes": uint64(256),
		"heap_allocation_count":  uint64(4),
		"bytes_requested":        uint64(256),
		"bytes_reserved":         uint64(512),
		"allocation_paths":       map[string]uint64{"actor_runtime": 4},
		"domain_bytes": []map[string]any{{
			"domain_id":             domainID,
			"kind":                  "actor",
			"requested_bytes":       uint64(256),
			"reserved_bytes":        uint64(512),
			"committed_bytes":       uint64(384),
			"current_bytes":         uint64(128),
			"peak_bytes":            uint64(192),
			"bytes_copied":          uint64(64),
			"mailbox_current_bytes": uint64(128),
			"mailbox_peak_bytes":    uint64(192),
			"byte_budget":           uint64(1024),
			"over_budget_count":     uint64(3),
			"backpressure_events":   uint64(4),
		}},
		"notes": []string{"test fixture"},
	}
	raw, err := json.Marshal(sample)
	if err != nil {
		t.Fatalf("marshal heap sidecar fixture: %v", err)
	}
	return fixture(t, dir, "artifacts/"+name+".actor.heap.json", string(raw))
}

func setAllocationReportFixture(t *testing.T, dir string, row map[string]any, content string) {
	t.Helper()
	metadata := row["tetra_metadata"].(map[string]any)
	rel, ok := metadata["allocation_report"].(string)
	if !ok || strings.TrimSpace(rel) == "" {
		t.Fatalf("row missing allocation_report metadata: %#v", metadata)
	}
	fixture(t, dir, rel, content)
}

func tetraRowForCategory(t *testing.T, report map[string]any, category string) map[string]any {
	t.Helper()
	results := report["results"].([]map[string]any)
	for _, result := range results {
		if result["category"] != category {
			continue
		}
		rows := result["rows"].([]map[string]any)
		for _, row := range rows {
			if row["language"] == "tetra" {
				return row
			}
		}
	}
	t.Fatalf("missing Tetra row for category %q", category)
	return nil
}

func zeroHeapFixtureCategory(category string) bool {
	switch category {
	case "integer loops", "function calls", "hash table", "startup time":
		return true
	default:
		return false
	}
}

func rssSidecarFixture(t *testing.T, dir string, name string, liveSample bool) string {
	t.Helper()
	sampleCount := uint64(2)
	current := uint64(4096)
	samples := `[{"unix_nano":110,"rss_bytes":4096},{"unix_nano":120,"rss_bytes":4096}]`
	if !liveSample {
		sampleCount = 0
		current = 0
		samples = `[]`
	}
	return fixture(t, dir, "artifacts/"+name+".rss.json", `{
		"schema":"`+rsstelemetry.Schema+`",
		"method":"`+rsstelemetry.MethodLinuxProcfsWait4RSSSamplerV1+`",
		"program":"`+name+`",
		"pid":1234,
		"target_os":"linux",
		"target_arch":"amd64",
		"started_unix_nano":100,
		"finished_unix_nano":200,
		"exit_status":0,
		"sample_interval_micros":500,
		"sample_count":`+itoaUint64(sampleCount)+`,
		"rss_current_bytes":`+itoaUint64(current)+`,
		"rss_peak_bytes":8192,
		"rss_peak_source":"`+rsstelemetry.PeakSourceWait4RusageMaxRSS+`",
		"ru_maxrss_raw":8,
		"ru_maxrss_unit":"`+rsstelemetry.UnitKilobytes+`",
		"samples":`+samples+`,
		"notes":["test fixture"]
	}`)
}

func localRSSBudgetPolicyFixture(
	peakBudget uint64,
	variancePercent float64,
	hostMismatch bool,
	includeLocalNonClaims bool,
) []byte {
	host := map[string]any{
		"goos":       "linux",
		"goarch":     "amd64",
		"cpus":       8,
		"target_cpu": "test cpu",
	}
	if hostMismatch {
		host["target_cpu"] = "different cpu"
	}
	nonClaims := []string{}
	if includeLocalNonClaims {
		nonClaims = []string{
			"local RSS budget only",
			"no cross-machine RSS claim",
			"no official benchmark claim",
		}
	}
	policy := map[string]any{
		"schema":       "tetra.local_benchmark.rss_budget_policy.v1",
		"target":       "linux-x64",
		"host_profile": host,
		"budgets": []map[string]any{
			{
				"category":                 "integer loops",
				"language":                 "tetra",
				"rss_peak_budget_bytes":    peakBudget,
				"allowed_variance_percent": variancePercent,
				"reason":                   "fixture local budget",
			},
		},
		"non_claims": nonClaims,
	}
	raw, err := json.Marshal(policy)
	if err != nil {
		panic(err)
	}
	return raw
}

func generatedStyleRSSBudgetPolicyFixture(
	t *testing.T,
	report map[string]any,
	budgetFor func(category string) uint64,
	hostMismatch bool,
) []byte {
	t.Helper()
	reportHost := report["host"].(map[string]any)
	host := map[string]any{
		"goos":       reportHost["goos"],
		"goarch":     reportHost["goarch"],
		"cpus":       reportHost["cpus"],
		"target_cpu": reportHost["target_cpu"],
		"git_commit": reportHost["git_commit"],
	}
	if hostMismatch {
		host["target_cpu"] = "different generated-style cpu"
	}
	var budgets []map[string]any
	for _, result := range report["results"].([]map[string]any) {
		category := result["category"].(string)
		for _, row := range result["rows"].([]map[string]any) {
			if row["language"] != "tetra" {
				continue
			}
			budgets = append(budgets, map[string]any{
				"category":                 category,
				"language":                 "tetra",
				"rss_peak_budget_bytes":    budgetFor(category),
				"allowed_variance_percent": 5,
				"reason":                   "generated local host-pinned RSS budget from row rss_peak evidence",
			})
		}
	}
	policy := map[string]any{
		"schema":       "tetra.local_benchmark.rss_budget_policy.v1",
		"target":       "linux-x64",
		"host_profile": host,
		"budgets":      budgets,
		"non_claims": []string{
			"local RSS budget only",
			"no cross-machine RSS claim",
			"no official benchmark claim",
		},
	}
	raw, err := json.Marshal(policy)
	if err != nil {
		t.Fatalf("marshal generated-style RSS policy: %v", err)
	}
	return raw
}

func itoaUint64(value uint64) string {
	return strconv.FormatUint(value, 10)
}

func blockedMemoryEvidenceFixture(reason string) map[string]any {
	metric := func() map[string]any {
		return map[string]any{
			"evidence_class": "blocked",
			"method":         "missing_build_artifacts",
			"blocked_reason": reason,
		}
	}
	return map[string]any{
		"schema":                "tetra.local_benchmark.memory_evidence.v1",
		"heap_alloc_bytes":      metric(),
		"bytes_requested":       metric(),
		"bytes_reserved":        metric(),
		"bytes_committed":       metric(),
		"bytes_released":        metric(),
		"bytes_copied":          metric(),
		"rss_current":           metric(),
		"rss_peak":              metric(),
		"domain_bytes_evidence": metric(),
		"domain_bytes":          []map[string]any{},
	}
}

func fixture(t *testing.T, dir string, rel string, content string) string {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", rel, err)
	}
	return rel
}
