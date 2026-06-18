package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestValidateMemoryReportAcceptsSchemaV1(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	if err := os.WriteFile(path, []byte(validSchemaV1MemoryReport()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateMemoryReport(path); err != nil {
		t.Fatalf("validateMemoryReport failed: %v", err)
	}
}

func TestValidateMemoryReportRejectsValidatedClaimWithoutFact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"source_fact_id": "fact:raw:root"`,
		`"source_fact_id": ""`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "source_fact_id") {
		t.Fatalf("validateMemoryReport error = %v, want source_fact_id rejection", err)
	}
}

func TestValidateMemoryReportRejectsStorageClaimWithoutArtifact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"lowered_artifact_id": "ir:main:alloc_bytes:0"`,
		`"lowered_artifact_id": ""`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "lowered_artifact_id") {
		t.Fatalf("validateMemoryReport error = %v, want lowered_artifact_id rejection", err)
	}
}

func TestValidateMemoryReportWithAllocReportRejectsFakeLoweredArtifactID(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"lowered_artifact_id": "ir:main:alloc_bytes:0"`,
		`"lowered_artifact_id": "ir:main:missing:Heap"`,
		1,
	)
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	allocPath := filepath.Join(dir, "alloc-report.json")
	if err := os.WriteFile(allocPath, []byte(validSchemaV2AllocationReport()), 0o644); err != nil {
		t.Fatal(err)
	}

	err := validateMemoryReportWithAllocReport(reportPath, allocPath)
	if err == nil || !strings.Contains(err.Error(), "lowered_artifact_id") ||
		!strings.Contains(err.Error(), "allocation report") {
		t.Fatalf(
			"validateMemoryReportWithAllocReport error = %v, want fake lowered_artifact_id rejection",
			err,
		)
	}
}

func TestValidateMemoryReportWithAllocReportAcceptsMatchingLoweredArtifactID(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"lowered_artifact_id": "ir:main:alloc_bytes:0"`,
		`"lowered_artifact_id": "ir:main:alloc_bytes:0:Heap"`,
		1,
	)
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	allocPath := filepath.Join(dir, "alloc-report.json")
	if err := os.WriteFile(allocPath, []byte(validSchemaV2AllocationReport()), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := validateMemoryReportWithAllocReport(reportPath, allocPath); err != nil {
		t.Fatalf("validateMemoryReportWithAllocReport failed: %v", err)
	}
}

func TestValidateMemoryReportWithAllocReportAcceptsDomainMetadata(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"lowered_artifact_id": "ir:main:alloc_bytes:0"`,
		`"lowered_artifact_id": "ir:main:alloc_bytes:0:Heap"`,
		1,
	)
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	allocReport := strings.Replace(
		validSchemaV2AllocationReport(),
		`"reason": "fixture allocation report row"`,
		`"reason": "fixture allocation report row",
          "domain": {
            "domain_id": "domain:process",
            "kind": "process",
            "owner_kind": "process",
            "owner_id": "current",
            "lifetime": "process",
            "requested_bytes": 17,
            "reserved_bytes": 32
          }`,
		1,
	)
	allocPath := filepath.Join(dir, "alloc-report.json")
	if err := os.WriteFile(allocPath, []byte(allocReport), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := validateMemoryReportWithAllocReport(reportPath, allocPath); err != nil {
		t.Fatalf("validateMemoryReportWithAllocReport rejected allocation domain metadata: %v", err)
	}
}

func TestValidateMemoryReportWithAllocReportRejectsInvalidDomainMetadata(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"lowered_artifact_id": "ir:main:alloc_bytes:0"`,
		`"lowered_artifact_id": "ir:main:alloc_bytes:0:Heap"`,
		1,
	)
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	allocReport := strings.Replace(
		validSchemaV2AllocationReport(),
		`"reason": "fixture allocation report row"`,
		`"reason": "fixture allocation report row",
          "domain": {
            "domain_id": "domain:process",
            "kind": "mystery",
            "owner_kind": "process",
            "owner_id": "current",
            "lifetime": "process"
          }`,
		1,
	)
	allocPath := filepath.Join(dir, "alloc-report.json")
	if err := os.WriteFile(allocPath, []byte(allocReport), 0o644); err != nil {
		t.Fatal(err)
	}

	err := validateMemoryReportWithAllocReport(reportPath, allocPath)
	if err == nil || !strings.Contains(err.Error(), "unknown domain kind") {
		t.Fatalf("validateMemoryReportWithAllocReport error = %v, want domain kind rejection", err)
	}
}

func TestValidateMemoryReportWithAllocReportRejectsMissingAllocationLengthContract(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"lowered_artifact_id": "ir:main:alloc_bytes:0"`,
		`"lowered_artifact_id": "ir:main:alloc_bytes:0:Heap"`,
		1,
	)
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	allocReport := validSchemaV2AllocationReport()
	for _, field := range []string{
		"builtin",
		"length_status",
		"zero_guard_status",
		"negative_guard_status",
		"overflow_guard_status",
	} {
		allocReport = removeJSONLineContaining(allocReport, `"`+field+`":`)
	}
	allocPath := filepath.Join(dir, "alloc-report.json")
	if err := os.WriteFile(allocPath, []byte(allocReport), 0o644); err != nil {
		t.Fatal(err)
	}

	err := validateMemoryReportWithAllocReport(reportPath, allocPath)
	if err == nil || !strings.Contains(err.Error(), "length_status") ||
		!strings.Contains(err.Error(), "zero_guard_status") {
		t.Fatalf(
			"validateMemoryReportWithAllocReport error = %v, want allocation length contract rejection",
			err,
		)
	}
}

func TestValidateMemoryReportWithAllocReportRejectsHeapAllocationWithoutReasonCodes(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"lowered_artifact_id": "ir:main:alloc_bytes:0"`,
		`"lowered_artifact_id": "ir:main:alloc_bytes:0:Heap"`,
		1,
	)
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	allocReport := removeJSONLineContaining(validSchemaV2AllocationReport(), `"reason_codes":`)
	allocReport = removeJSONLineContaining(allocReport, `"heap_reason_codes":`)
	allocPath := filepath.Join(dir, "alloc-report.json")
	if err := os.WriteFile(allocPath, []byte(allocReport), 0o644); err != nil {
		t.Fatal(err)
	}

	err := validateMemoryReportWithAllocReport(reportPath, allocPath)
	if err == nil || !strings.Contains(err.Error(), "heap_reason_codes") {
		t.Fatalf(
			"validateMemoryReportWithAllocReport error = %v, want heap_reason_codes rejection",
			err,
		)
	}
}

func TestValidateMemoryReportRejectsPartialStorageFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"actual_lowering_storage": "Heap"`,
		`"actual_lowering_storage": ""`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "planned_storage and actual_lowering_storage") {
		t.Fatalf("validateMemoryReport error = %v, want paired storage rejection", err)
	}
}

func TestValidateMemoryReportRejectsUnknownStorageClass(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"planned_storage": "Heap"`,
		`"planned_storage": "Mystery"`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "unknown planned_storage") {
		t.Fatalf("validateMemoryReport error = %v, want unknown storage rejection", err)
	}
}

func TestValidateMemoryReportRejectsIslandBackedRowMissingEpoch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"source_fact_id": "fact:raw:root"`,
		`"island_id": "island:main:0",`+"\n      "+`"base_id": "alloc:main:0",`+"\n      "+`"source_fact_id": "fact:raw:root"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"planned_storage": "Heap"`,
		`"planned_storage": "ExplicitIsland"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"actual_lowering_storage": "Heap"`,
		`"actual_lowering_storage": "ExplicitIsland"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"cost_class": "zero_cost_proven"`,
		`"cost_class": "instrumentation_only"`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "epoch") {
		t.Fatalf("validateMemoryReport error = %v, want missing epoch rejection", err)
	}
}

func TestValidateMemoryReportRejectsValidatedStackClaimLoweringAsHeap(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"claim": "allocation_base_metadata"`,
		`"claim": "storage_lowering"`,
		1,
	)
	raw = strings.Replace(raw, `"source_stage": "validation"`, `"source_stage": "allocplan"`, 1)
	raw = strings.Replace(
		raw,
		`"provenance_class": "unsafe_verified_root"`,
		`"provenance_class": "safe_owned"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"unsafe_class": "unsafe_verified_root"`,
		`"unsafe_class": "safe"`,
		1,
	)
	raw = strings.Replace(raw, `"planned_storage": "Heap"`, `"planned_storage": "Stack"`, 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "validated Stack claim cannot lower as Heap") {
		t.Fatalf("validateMemoryReport error = %v, want stack heap-fallback rejection", err)
	}
}

func TestValidateMemoryReportRejectsValidatedTaskActorRegionStorageWithoutRuntimeProof(
	t *testing.T,
) {
	for _, storage := range []string{"TaskRegion", "ActorMoveRegion"} {
		t.Run(storage, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "memory-report.json")
			raw := strings.Replace(
				validSchemaV1MemoryReport(),
				`"claim": "allocation_base_metadata"`,
				`"claim": "storage_lowering"`,
				1,
			)
			raw = strings.Replace(
				raw,
				`"source_stage": "validation"`,
				`"source_stage": "allocplan"`,
				1,
			)
			raw = strings.Replace(
				raw,
				`"provenance_class": "unsafe_verified_root"`,
				`"provenance_class": "safe_owned"`,
				1,
			)
			raw = strings.Replace(
				raw,
				`"unsafe_class": "unsafe_verified_root"`,
				`"unsafe_class": "safe"`,
				1,
			)
			raw = strings.Replace(
				raw,
				`"planned_storage": "Heap"`,
				`"planned_storage": "`+storage+`"`,
				1,
			)
			raw = strings.Replace(
				raw,
				`"actual_lowering_storage": "Heap"`,
				`"actual_lowering_storage": "`+storage+`"`,
				1,
			)
			raw = strings.Replace(
				raw,
				`"lowered_artifact_id": "ir:main:alloc_bytes:0"`,
				`"lowered_artifact_id": "ir:main:boundary:`+storage+`"`,
				1,
			)
			if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}
			err := validateMemoryReport(path)
			if err == nil || !strings.Contains(err.Error(), "runtime") {
				t.Fatalf("validateMemoryReport error = %v, want runtime proof rejection", err)
			}
		})
	}
}

func TestValidateMemoryReportRejectsValidatedClaimWithoutValidatorName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"validator_name": "raw_bounds_validator"`,
		`"validator_name": ""`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "validator_name") {
		t.Fatalf("validateMemoryReport error = %v, want validator_name rejection", err)
	}
}

func TestValidateMemoryReportRejectsWhitespaceRequiredFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"source_fact_id": "fact:raw:root"`,
		`"source_fact_id": "   "`,
		1,
	)
	raw = strings.Replace(raw, `"site_id": "alloc:main:1:1"`, `"site_id": "   "`, 1)
	raw = strings.Replace(raw, `"claim": "allocation_base_metadata"`, `"claim": "   "`, 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "source_fact_id") ||
		!strings.Contains(err.Error(), "site_id") ||
		!strings.Contains(err.Error(), "claim") {
		t.Fatalf("validateMemoryReport error = %v, want whitespace required-field rejection", err)
	}
}

func TestValidateMemoryReportRejectsDuplicateSourceFactID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		"\n    }\n  ]",
		"\n    },\n    {\n      \"program_id\": \"program\",\n      \"function_id\": \"main\",\n      \"site_id\": \"alloc:main:1:2\",\n      \"source_fact_id\": \"fact:raw:root\",\n      \"source_stage\": \"validation\",\n      \"claim\": \"allocation_base_metadata\",\n      \"claim_level\": \"validated\",\n      \"provenance_class\": \"unsafe_verified_root\",\n      \"unsafe_class\": \"unsafe_verified_root\",\n      \"planned_storage\": \"Heap\",\n      \"actual_lowering_storage\": \"Heap\",\n      \"lowered_artifact_id\": \"ir:main:alloc_bytes:1\",\n      \"validator_name\": \"raw_bounds_validator\",\n      \"validator_status\": \"pass\"\n    }\n  ]",
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "duplicate source_fact_id") {
		t.Fatalf("validateMemoryReport error = %v, want duplicate source_fact_id rejection", err)
	}
}

func TestValidateMemoryReportRejectsNonDeterministicSourceFactIDOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		"\n    }\n  ]",
		"\n    },\n    {\n      \"program_id\": \"program\",\n      \"function_id\": \"main\",\n      \"site_id\": \"alloc:main:1:2\",\n      \"source_fact_id\": \"fact:aaa:root\",\n      \"source_stage\": \"validation\",\n      \"claim\": \"allocation_base_metadata\",\n      \"claim_level\": \"validated\",\n      \"provenance_class\": \"unsafe_verified_root\",\n      \"unsafe_class\": \"unsafe_verified_root\",\n      \"planned_storage\": \"Heap\",\n      \"actual_lowering_storage\": \"Heap\",\n      \"lowered_artifact_id\": \"ir:main:alloc_bytes:1\",\n      \"cost_class\": \"zero_cost_proven\",\n      \"validator_name\": \"raw_bounds_validator\",\n      \"validator_status\": \"pass\",\n      \"reason\": \"verified core.alloc_bytes root\"\n    }\n  ]",
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "deterministic") {
		t.Fatalf(
			"validateMemoryReport error = %v, want deterministic source_fact_id order rejection",
			err,
		)
	}
}

func TestValidateMemoryReportRejectsV1DerivedBorrowRowsWithoutParent(t *testing.T) {
	for _, claim := range []string{"enum_payload_contains_borrow", "generic_wrapper_contains_borrow"} {
		t.Run(claim, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "memory-report.json")
			raw := strings.Replace(
				validSchemaV1MemoryReport(),
				`"claim": "allocation_base_metadata"`,
				`"claim": "`+claim+`"`,
				1,
			)
			raw = strings.Replace(raw, `"source_stage": "validation"`, `"source_stage": "plir"`, 1)
			raw = strings.Replace(
				raw,
				`"provenance_class": "unsafe_verified_root"`,
				`"provenance_class": "safe_borrowed"`,
				1,
			)
			raw = strings.Replace(
				raw,
				`"unsafe_class": "unsafe_verified_root"`,
				`"unsafe_class": "safe"`,
				1,
			)
			if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}
			err := validateMemoryReport(path)
			if err == nil || !strings.Contains(err.Error(), "parent_fact_id") {
				t.Fatalf("validateMemoryReport error = %v, want parent_fact_id rejection", err)
			}
		})
	}
}

func TestValidateMemoryReportRejectsV2DerivedRowsWithoutParent(t *testing.T) {
	for _, claim := range []string{
		"function_value_contains_borrow",
		"callback_arg_contains_borrow",
		"callback_inout_conservative",
	} {
		t.Run(claim, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "memory-report.json")
			raw := strings.Replace(
				validSchemaV1MemoryReport(),
				`"claim": "allocation_base_metadata"`,
				`"claim": "`+claim+`"`,
				1,
			)
			raw = strings.Replace(raw, `"source_stage": "validation"`, `"source_stage": "plir"`, 1)
			raw = strings.Replace(
				raw,
				`"provenance_class": "unsafe_verified_root"`,
				`"provenance_class": "safe_borrowed"`,
				1,
			)
			raw = strings.Replace(
				raw,
				`"unsafe_class": "unsafe_verified_root"`,
				`"unsafe_class": "safe"`,
				1,
			)
			if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}
			err := validateMemoryReport(path)
			if err == nil || !strings.Contains(err.Error(), "parent_fact_id") {
				t.Fatalf("validateMemoryReport error = %v, want parent_fact_id rejection", err)
			}
		})
	}
}

func TestValidateMemoryReportRejectsV3DerivedRowsWithoutParent(t *testing.T) {
	for _, claim := range []string{
		"interface_value_contains_borrow",
		"protocol_dispatch_borrow_conservative",
		"protocol_dispatch_noalias_conservative",
	} {
		t.Run(claim, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "memory-report.json")
			raw := strings.Replace(
				validSchemaV1MemoryReport(),
				`"claim": "allocation_base_metadata"`,
				`"claim": "`+claim+`"`,
				1,
			)
			raw = strings.Replace(raw, `"source_stage": "validation"`, `"source_stage": "plir"`, 1)
			raw = strings.Replace(
				raw,
				`"provenance_class": "unsafe_verified_root"`,
				`"provenance_class": "safe_borrowed"`,
				1,
			)
			raw = strings.Replace(
				raw,
				`"unsafe_class": "unsafe_verified_root"`,
				`"unsafe_class": "safe"`,
				1,
			)
			if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}
			err := validateMemoryReport(path)
			if err == nil || !strings.Contains(err.Error(), "parent_fact_id") {
				t.Fatalf("validateMemoryReport error = %v, want parent_fact_id rejection", err)
			}
		})
	}
}

func TestValidateMemoryReportRejectsV4DerivedRowsWithoutParent(t *testing.T) {
	for _, claim := range []string{
		"async_boundary_borrow_conservative",
		"task_boundary_borrow_rejected",
		"actor_boundary_borrow_rejected",
		"boundary_noalias_conservative",
	} {
		t.Run(claim, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "memory-report.json")
			raw := strings.Replace(
				validSchemaV1MemoryReport(),
				`"claim": "allocation_base_metadata"`,
				`"claim": "`+claim+`"`,
				1,
			)
			raw = strings.Replace(raw, `"source_stage": "validation"`, `"source_stage": "plir"`, 1)
			raw = strings.Replace(
				raw,
				`"provenance_class": "unsafe_verified_root"`,
				`"provenance_class": "safe_borrowed"`,
				1,
			)
			raw = strings.Replace(
				raw,
				`"unsafe_class": "unsafe_verified_root"`,
				`"unsafe_class": "safe"`,
				1,
			)
			if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}
			err := validateMemoryReport(path)
			if err == nil || !strings.Contains(err.Error(), "parent_fact_id") {
				t.Fatalf("validateMemoryReport error = %v, want parent_fact_id rejection", err)
			}
		})
	}
}

func TestValidateMemoryReportRejectsV5DerivedRowsWithoutParent(t *testing.T) {
	for _, claim := range []string{
		"unsafe_unknown_rejected_safe_facts",
		"unsafe_verified_root_allocation_base",
	} {
		t.Run(claim, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "memory-report.json")
			raw := strings.Replace(
				memoryIdealV5UnsafeContractReport(),
				`"claim": "unsafe_unknown_rejected_safe_facts"`,
				`"claim": "`+claim+`"`,
				1,
			)
			raw = strings.Replace(
				raw,
				`"parent_fact_id": "fact:unsafe:unknown",`,
				`"parent_fact_id": "",`,
				1,
			)
			if claim == "unsafe_verified_root_allocation_base" {
				raw = strings.Replace(
					raw,
					`"provenance_class": "unsafe_unknown"`,
					`"provenance_class": "unsafe_verified_root"`,
					1,
				)
				raw = strings.Replace(
					raw,
					`"unsafe_class": "unsafe_unknown"`,
					`"unsafe_class": "unsafe_verified_root"`,
					1,
				)
				raw = strings.Replace(
					raw,
					`"claim_level": "rejected"`,
					`"claim_level": "validated"`,
					1,
				)
				raw = strings.Replace(
					raw,
					`"cost_class": "unsupported_rejected"`,
					`"cost_class": "zero_cost_proven"`,
					1,
				)
				raw = strings.Replace(
					raw,
					`"validator_name": "unsafe_unknown_fact_validator"`,
					`"validator_name": "unsafe_verified_root_bounds_validator"`,
					1,
				)
				raw = strings.Replace(
					raw,
					`"validator_status": "fail"`,
					`"validator_status": "pass"`,
					1,
				)
			}
			if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}
			err := validateMemoryReport(path)
			if err == nil || !strings.Contains(err.Error(), "parent_fact_id") {
				t.Fatalf("validateMemoryReport error = %v, want parent_fact_id rejection", err)
			}
		})
	}
}

func TestValidateMemoryReportRejectsV6BoundsRowsWithoutParent(t *testing.T) {
	for _, claim := range []string{
		"bounds_check_removed_with_proof_id",
		"raw_bounds_runtime_check_normal_build",
	} {
		t.Run(claim, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "memory-report.json")
			raw := strings.Replace(
				memoryIdealV6BoundsReport(),
				`"claim": "bounds_check_removed_with_proof_id"`,
				`"claim": "`+claim+`"`,
				1,
			)
			raw = strings.Replace(
				raw,
				`"parent_fact_id": "fact:bounds:proof-guard",`,
				`"parent_fact_id": "",`,
				1,
			)
			if claim == "raw_bounds_runtime_check_normal_build" {
				raw = strings.Replace(
					raw,
					`"validator_name": "bounds_proof_id_validator"`,
					`"validator_name": "raw_bounds_width_validator"`,
					1,
				)
				raw = strings.Replace(
					raw,
					`"cost_class": "zero_cost_proven"`,
					`"cost_class": "dynamic_check_required"`,
					1,
				)
				raw = strings.Replace(
					raw,
					`"provenance_class": "safe_known"`,
					`"provenance_class": "unsafe_checked"`,
					1,
				)
				raw = strings.Replace(
					raw,
					`"unsafe_class": "safe"`,
					`"unsafe_class": "unsafe_checked"`,
					1,
				)
				raw = strings.Replace(
					raw,
					`"source_stage": "validation"`,
					`"source_stage": "plir"`,
					1,
				)
				raw = strings.Replace(
					raw,
					`"reason": "removed bounds check has compiler-owned proof id"`,
					`"normal_build_check": true,
      "reason": "raw bounds uncertainty keeps normal-build check"`,
					1,
				)
			}
			if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}
			err := validateMemoryReport(path)
			if err == nil || !strings.Contains(err.Error(), "parent_fact_id") {
				t.Fatalf("validateMemoryReport error = %v, want parent_fact_id rejection", err)
			}
		})
	}
}

func TestValidateMemoryReportRejectsBoundsProofWithoutTypedProofFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		memoryIdealV6BoundsReport(),
		`"proof_kind": "bounds_check",`,
		`"proof_kind": "",`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "typed proof fields") ||
		!strings.Contains(err.Error(), "proof_kind") {
		t.Fatalf("validateMemoryReport error = %v, want typed proof field rejection", err)
	}
}

func TestValidateMemoryReportRejectsSafeViewRetainedBoundsWithoutParent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		memoryIdealV6BoundsReport(),
		`"parent_fact_id": "fact:bounds:retained-window",`,
		`"parent_fact_id": "",`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "parent_fact_id") {
		t.Fatalf("validateMemoryReport error = %v, want parent_fact_id rejection", err)
	}
}

func TestValidateMemoryReportRejectsB03CopyIntoRowsWithoutParent(t *testing.T) {
	for _, claim := range []string{
		"copy_into_destination_length_check",
		"copy_into_overlap_rejected",
		"copy_into_overlap_conservative",
	} {
		t.Run(claim, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "memory-report.json")
			raw := strings.Replace(
				validSchemaV1MemoryReport(),
				`"claim": "allocation_base_metadata"`,
				`"claim": "`+claim+`"`,
				1,
			)
			raw = strings.Replace(raw, `"source_stage": "validation"`, `"source_stage": "plir"`, 1)
			raw = strings.Replace(
				raw,
				`"provenance_class": "unsafe_verified_root"`,
				`"provenance_class": "safe_known"`,
				1,
			)
			raw = strings.Replace(
				raw,
				`"unsafe_class": "unsafe_verified_root"`,
				`"unsafe_class": "safe"`,
				1,
			)
			raw = strings.Replace(
				raw,
				`"validator_name": "raw_bounds_validator"`,
				`"validator_name": "copy_into_overlap_validator"`,
				1,
			)
			if claim == "copy_into_destination_length_check" {
				raw = strings.Replace(
					raw,
					`"cost_class": "zero_cost_proven"`,
					`"cost_class": "dynamic_check_required"`,
					1,
				)
				raw = strings.Replace(
					raw,
					`"reason": "verified core.alloc_bytes root"`,
					`"normal_build_check": true,
      "reason": "copy_into destination capacity check is retained in normal builds"`,
					1,
				)
			}
			if claim == "copy_into_overlap_rejected" {
				raw = strings.Replace(
					raw,
					`"claim_level": "validated"`,
					`"claim_level": "rejected"`,
					1,
				)
				raw = strings.Replace(
					raw,
					`"cost_class": "zero_cost_proven"`,
					`"cost_class": "unsupported_rejected"`,
					1,
				)
				raw = strings.Replace(
					raw,
					`"validator_status": "pass"`,
					`"validator_status": "fail"`,
					1,
				)
			}
			if claim == "copy_into_overlap_conservative" {
				raw = strings.Replace(
					raw,
					`"claim_level": "validated"`,
					`"claim_level": "conservative"`,
					1,
				)
				raw = strings.Replace(
					raw,
					`"provenance_class": "safe_known"`,
					`"provenance_class": "unsafe_unknown"`,
					1,
				)
				raw = strings.Replace(
					raw,
					`"unsafe_class": "safe"`,
					`"unsafe_class": "unsafe_unknown"`,
					1,
				)
				raw = strings.Replace(
					raw,
					`"cost_class": "zero_cost_proven"`,
					`"cost_class": "conservative_fallback"`,
					1,
				)
				raw = strings.Replace(
					raw,
					`"validator_status": "pass"`,
					`"validator_status": "not_applicable"`,
					1,
				)
			}
			if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}
			err := validateMemoryReport(path)
			if err == nil || !strings.Contains(err.Error(), "parent_fact_id") {
				t.Fatalf("validateMemoryReport error = %v, want parent_fact_id rejection", err)
			}
		})
	}
}

func TestValidateMemoryReportRejectsTrailingData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := validSchemaV1MemoryReport() + "\n" + validSchemaV1MemoryReport()
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "trailing data") {
		t.Fatalf("validateMemoryReport error = %v, want trailing data rejection", err)
	}
}

func TestValidateMemoryReportRejectsUnknownAliasState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"reason": "verified core.alloc_bytes root"`,
		`"alias_state": "mystery_alias",`+"\n      "+`"reason": "verified core.alloc_bytes root"`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "unknown alias_state") {
		t.Fatalf("validateMemoryReport error = %v, want unknown alias_state rejection", err)
	}
}

func TestValidateMemoryReportRejectsValidatedNoAliasWithUnknownAliasState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"claim": "allocation_base_metadata"`,
		`"claim": "no_alias"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"reason": "verified core.alloc_bytes root"`,
		`"alias_state": "unknown_alias",`+"\n      "+`"reason": "verified core.alloc_bytes root"`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "validated no_alias") {
		t.Fatalf("validateMemoryReport error = %v, want validated no_alias rejection", err)
	}
}

func TestValidateMemoryReportRejectsSafeKnownFromUnsafeUnknown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"provenance_class": "unsafe_verified_root"`,
		`"provenance_class": "safe_known"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"unsafe_class": "unsafe_verified_root"`,
		`"unsafe_class": "unsafe_unknown"`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "safe_known") {
		t.Fatalf("validateMemoryReport error = %v, want safe_known rejection", err)
	}
}

func TestValidateMemoryReportRejectsSafeBorrowedFromUnsafeUnknown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"provenance_class": "unsafe_verified_root"`,
		`"provenance_class": "safe_borrowed"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"unsafe_class": "unsafe_verified_root"`,
		`"unsafe_class": "unsafe_unknown"`,
		1,
	)
	raw = strings.Replace(raw, `"claim": "allocation_base_metadata"`, `"claim": "borrowed_imm"`, 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "safe_borrowed") {
		t.Fatalf("validateMemoryReport error = %v, want safe_borrowed rejection", err)
	}
}

func TestValidateMemoryReportRejectsUnsafeUnknownOptimizationClaim(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		conservativeUnknownRawPointerReport(),
		`"claim": "checked_external_unknown"`,
		`"claim": "no_alias"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"reason": "unknown raw pointer remains conservative"`,
		`"alias_state": "mutable_exclusive",`+"\n      "+`"reason": "unknown raw pointer remains conservative"`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "unsafe_unknown") {
		t.Fatalf("validateMemoryReport error = %v, want unsafe_unknown optimization rejection", err)
	}
}

func TestValidateMemoryReportRejectsUnsafeCheckedGenericPromotion(t *testing.T) {
	for _, tc := range []struct {
		name       string
		claim      string
		provenance string
		alias      string
		want       string
	}{
		{name: "safe known", claim: "safe_known", provenance: "safe_known", want: "unsafe_checked"},
		{
			name:       "provenance known",
			claim:      "provenance_known",
			provenance: "unsafe_checked",
			want:       "unsafe_checked",
		},
		{name: "noalias", claim: "no_alias", provenance: "unsafe_checked", alias: `,
      "alias_state": "unique"`, want: "unsafe_checked"},
		{
			name:       "bounds check elimination",
			claim:      "bounds_check_eliminated",
			provenance: "unsafe_checked",
			want:       "proof id",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "memory-report.json")
			raw := strings.Replace(
				validSchemaV1MemoryReport(),
				`"claim": "allocation_base_metadata"`,
				`"claim": "`+tc.claim+`"`,
				1,
			)
			raw = strings.Replace(raw, `"source_stage": "validation"`, `"source_stage": "plir"`, 1)
			raw = strings.Replace(
				raw,
				`"provenance_class": "unsafe_verified_root"`,
				`"provenance_class": "`+tc.provenance+`"`,
				1,
			)
			raw = strings.Replace(
				raw,
				`"unsafe_class": "unsafe_verified_root"`,
				`"unsafe_class": "unsafe_checked"`+tc.alias,
				1,
			)
			if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}
			err := validateMemoryReport(path)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validateMemoryReport error = %v, want %q rejection", err, tc.want)
			}
		})
	}
}

func TestValidateMemoryReportRejectsDynamicRawOffsetZeroCostPromotion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"function_id": "main"`,
		`"function_id": "read_at"`,
		1,
	)
	raw = strings.Replace(raw, `"site_id": "alloc:main:1:1"`, `"site_id": "test.tetra:6:17"`, 1)
	raw = strings.Replace(
		raw,
		`"source_span": "main.tetra:1:1"`,
		`"source_span": "test.tetra:6:17"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"source_fact_id": "fact:raw:root"`,
		`"source_fact_id": "fact:dynamic:raw-offset"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"lowered_artifact_id": "ir:main:alloc_bytes:0",`+"\n      "+`"source_stage": "validation"`,
		`"source_stage": "plir"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"claim": "allocation_base_metadata"`,
		`"claim": "derived_allocation_offset"`,
		1,
	)
	raw = strings.Replace(raw, `"claim_level": "validated"`, `"claim_level": "evidence_only"`, 1)
	raw = strings.Replace(
		raw,
		`"provenance_class": "unsafe_verified_root"`,
		`"provenance_class": "unsafe_checked"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"unsafe_class": "unsafe_verified_root"`,
		`"unsafe_class": "unsafe_checked"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"planned_storage": "Heap",`+"\n      "+`"actual_lowering_storage": "Heap",`+"\n      ",
		"",
		1,
	)
	raw = strings.Replace(
		raw,
		`"validator_name": "raw_bounds_validator"`,
		`"validator_name": ""`,
		1,
	)
	raw = strings.Replace(raw, `"validator_status": "pass"`, `"validator_status": "not_run"`, 1)
	raw = strings.Replace(
		raw,
		`"reason": "verified core.alloc_bytes root"`,
		`"reason": "core.ptr_add raw_pointer_bounds: checked_external_unknown base:p offset:n"`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "dynamic raw") ||
		!strings.Contains(err.Error(), "zero_cost_proven") {
		t.Fatalf("validateMemoryReport error = %v, want dynamic raw zero-cost rejection", err)
	}
}

func TestValidateMemoryReportRejectsValidatedCapMemAsProof(t *testing.T) {
	for _, tc := range []struct {
		name  string
		claim string
		alias string
		want  string
	}{
		{name: "safe provenance", claim: "provenance_known", want: "cap.mem"},
		{name: "noalias", claim: "no_alias", alias: `,
      "alias_state": "mutable_exclusive"`, want: "cap.mem"},
		{name: "bounds proof", claim: "index_in_range", want: "cap.mem"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "memory-report.json")
			raw := strings.Replace(
				validSchemaV1MemoryReport(),
				`"claim": "allocation_base_metadata"`,
				`"claim": "`+tc.claim+`"`,
				1,
			)
			raw = strings.Replace(raw, `"source_stage": "validation"`, `"source_stage": "plir"`, 1)
			raw = strings.Replace(
				raw,
				`"provenance_class": "unsafe_verified_root"`,
				`"provenance_class": "safe_known"`,
				1,
			)
			raw = strings.Replace(
				raw,
				`"unsafe_class": "unsafe_verified_root"`,
				`"unsafe_class": "safe"`+tc.alias,
				1,
			)
			raw = strings.Replace(
				raw,
				`"validator_name": "raw_bounds_validator"`,
				`"validator_name": "cap_mem_authorization_validator"`,
				1,
			)
			raw = strings.Replace(
				raw,
				`"reason": "verified core.alloc_bytes root"`,
				`"reason": "cap.mem authorized raw helper call"`,
				1,
			)
			raw = strings.Replace(
				raw,
				`"planned_storage": "Heap",`+"\n      "+`"actual_lowering_storage": "Heap",`+"\n      "+`"lowered_artifact_id": "ir:main:alloc_bytes:0",`,
				"",
				1,
			)
			if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}
			err := validateMemoryReport(path)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validateMemoryReport error = %v, want %q rejection", err, tc.want)
			}
		})
	}
}

func TestValidateMemoryReportRejectsMissingCostClass(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		"\n      \"cost_class\": \"zero_cost_proven\",",
		"",
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "cost_class") {
		t.Fatalf("validateMemoryReport error = %v, want cost_class rejection", err)
	}
}

func TestValidateMemoryReportRejectsUnknownCostClass(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"cost_class": "zero_cost_proven"`,
		`"cost_class": "mystery_cost"`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "unknown cost_class") {
		t.Fatalf("validateMemoryReport error = %v, want unknown cost_class rejection", err)
	}
}

func TestValidateMemoryReportRejectsConservativeZeroCost(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"claim": "allocation_base_metadata"`,
		`"claim": "callback_inout_conservative"`,
		1,
	)
	raw = strings.Replace(raw, `"claim_level": "validated"`, `"claim_level": "conservative"`, 1)
	raw = strings.Replace(raw, `"source_stage": "validation"`, `"source_stage": "plir"`, 1)
	raw = strings.Replace(
		raw,
		`"provenance_class": "unsafe_verified_root"`,
		`"provenance_class": "unsafe_unknown"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"unsafe_class": "unsafe_verified_root"`,
		`"unsafe_class": "unsafe_unknown"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"planned_storage": "Heap",`+"\n      "+`"actual_lowering_storage": "Heap",`+"\n      "+`"lowered_artifact_id": "ir:main:alloc_bytes:0",`,
		"",
		1,
	)
	raw = strings.Replace(
		raw,
		`"validator_name": "raw_bounds_validator"`,
		`"validator_name": "callback_alias_conservative_validator"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"validator_status": "pass"`,
		`"validator_status": "not_applicable"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"source_fact_id": "fact:raw:root"`,
		`"source_fact_id": "fact:callback:boundary",`+"\n      "+`"parent_fact_id": "fact:callback:borrow"`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "zero_cost_proven") {
		t.Fatalf("validateMemoryReport error = %v, want conservative zero-cost rejection", err)
	}
}

func TestValidateMemoryReportRejectsUnvalidatedStorageLoweringZeroCost(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"claim": "allocation_base_metadata"`,
		`"claim": "storage_lowering"`,
		1,
	)
	raw = strings.Replace(raw, `"claim_level": "validated"`, `"claim_level": "evidence_only"`, 1)
	raw = strings.Replace(raw, `"source_stage": "validation"`, `"source_stage": "allocplan"`, 1)
	raw = strings.Replace(
		raw,
		`"provenance_class": "unsafe_verified_root"`,
		`"provenance_class": "safe_owned"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"unsafe_class": "unsafe_verified_root"`,
		`"unsafe_class": "safe"`,
		1,
	)
	raw = strings.Replace(raw, `"planned_storage": "Heap"`, `"planned_storage": "Stack"`, 1)
	raw = strings.Replace(
		raw,
		`"actual_lowering_storage": "Heap"`,
		`"actual_lowering_storage": "Stack"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"validator_name": "raw_bounds_validator"`,
		`"validator_name": ""`,
		1,
	)
	raw = strings.Replace(raw, `"validator_status": "pass"`, `"validator_status": "not_run"`, 1)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "zero_cost_proven") {
		t.Fatalf("validateMemoryReport error = %v, want storage zero-cost proof rejection", err)
	}
}

func TestValidateMemoryReportRejectsDynamicOptimizationWithoutNormalBuildCheck(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"claim": "allocation_base_metadata"`,
		`"claim": "bounds_check_eliminated"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"cost_class": "zero_cost_proven"`,
		`"cost_class": "dynamic_check_required"`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "dynamic_check_required") ||
		!strings.Contains(err.Error(), "normal_build_check") {
		t.Fatalf("validateMemoryReport error = %v, want dynamic check rejection", err)
	}
}

func TestValidateMemoryReportRejectsBareBoundsCheckEliminatedWithoutProofID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"claim": "allocation_base_metadata"`,
		`"claim": "bounds_check_eliminated"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"provenance_class": "unsafe_verified_root"`,
		`"provenance_class": "safe_known"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"unsafe_class": "unsafe_verified_root"`,
		`"unsafe_class": "safe"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"validator_name": "raw_bounds_validator"`,
		`"validator_name": "bounds_proof_id_validator"`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "proof id") {
		t.Fatalf("validateMemoryReport error = %v, want proof id rejection", err)
	}
}

func TestValidateMemoryReportRejectsUnsafeUnknownZeroCost(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		conservativeUnknownRawPointerReport(),
		`"cost_class": "conservative_fallback"`,
		`"cost_class": "zero_cost_proven"`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "unsafe_unknown") ||
		!strings.Contains(err.Error(), "zero_cost_proven") {
		t.Fatalf("validateMemoryReport error = %v, want unsafe zero-cost rejection", err)
	}
}

func TestValidateMemoryReportRejectsUnsafeUnknownProvenanceKnownClaim(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		conservativeUnknownRawPointerReport(),
		`"claim": "checked_external_unknown"`,
		`"claim": "provenance_known"`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "unsafe_unknown") {
		t.Fatalf(
			"validateMemoryReport error = %v, want unsafe_unknown provenance_known rejection",
			err,
		)
	}
}

func TestValidateMemoryReportRejectsUnsafeVerifiedRootGenericClaim(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"claim": "allocation_base_metadata"`,
		`"claim": "provenance_known"`,
		1,
	)
	raw = strings.Replace(raw, `"claim_level": "validated"`, `"claim_level": "evidence_only"`, 1)
	raw = strings.Replace(
		raw,
		`"validator_name": "raw_bounds_validator"`,
		`"validator_name": ""`,
		1,
	)
	raw = strings.Replace(raw, `"validator_status": "pass"`, `"validator_status": "not_run"`, 1)
	raw = strings.Replace(
		raw,
		`"planned_storage": "Heap",`+"\n      "+`"actual_lowering_storage": "Heap",`+"\n      "+`"validator_name": ""`,
		`"validator_name": ""`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"lowered_artifact_id": "ir:main:alloc_bytes:0",`+"\n      "+`"source_stage"`,
		`"source_stage"`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "unsafe_verified_root") {
		t.Fatalf(
			"validateMemoryReport error = %v, want unsafe_verified_root generic-claim rejection",
			err,
		)
	}
}

func TestValidateMemoryReportRejectsUnsafeUnknownTrustedStorage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	raw := strings.Replace(
		validSchemaV1MemoryReport(),
		`"provenance_class": "unsafe_verified_root"`,
		`"provenance_class": "unsafe_unknown"`,
		1,
	)
	raw = strings.Replace(
		raw,
		`"unsafe_class": "unsafe_verified_root"`,
		`"unsafe_class": "unsafe_unknown"`,
		1,
	)
	raw = strings.Replace(raw, `"planned_storage": "Heap"`, `"planned_storage": "Stack"`, 1)
	raw = strings.Replace(
		raw,
		`"actual_lowering_storage": "Heap"`,
		`"actual_lowering_storage": "Stack"`,
		1,
	)
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateMemoryReport(path)
	if err == nil || !strings.Contains(err.Error(), "unsafe_unknown") {
		t.Fatalf(
			"validateMemoryReport error = %v, want unsafe_unknown trusted-storage rejection",
			err,
		)
	}
}

func TestValidateMemoryReportRejectsUnsafeVerifiedRootTrustedStorage(t *testing.T) {
	for _, storage := range []string{"Stack", "Region", "FunctionTempRegion"} {
		t.Run(storage, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "memory-report.json")
			raw := strings.Replace(
				validSchemaV1MemoryReport(),
				`"planned_storage": "Heap"`,
				`"planned_storage": "`+storage+`"`,
				1,
			)
			raw = strings.Replace(
				raw,
				`"actual_lowering_storage": "Heap"`,
				`"actual_lowering_storage": "`+storage+`"`,
				1,
			)
			raw = strings.Replace(
				raw,
				`"lowered_artifact_id": "ir:main:alloc_bytes:0"`,
				`"lowered_artifact_id": "ir:main:ffi:`+storage+`"`,
				1,
			)
			raw = strings.Replace(
				raw,
				`"reason": "verified core.alloc_bytes root"`,
				`"reason": "raw FFI root must not become trusted storage"`,
				1,
			)
			if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}
			err := validateMemoryReport(path)
			if err == nil || !strings.Contains(err.Error(), "unsafe_verified_root") ||
				!strings.Contains(err.Error(), "trusted storage") {
				t.Fatalf(
					"validateMemoryReport error = %v, want unsafe_verified_root trusted-storage rejection",
					err,
				)
			}
		})
	}
}

func TestValidateMemoryReportAcceptsConservativeUnknownRawPointer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	if err := os.WriteFile(path, []byte(conservativeUnknownRawPointerReport()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateMemoryReport(path); err != nil {
		t.Fatalf("validateMemoryReport conservative unknown raw pointer: %v", err)
	}
}

func TestValidateMemoryReportAcceptsRawSliceRejectedEvidenceRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	if err := os.WriteFile(path, []byte(rawSliceRejectedMemoryReport()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateMemoryReport(path); err != nil {
		t.Fatalf("validateMemoryReport raw slice rejected evidence: %v", err)
	}
}

func TestValidateMemoryReportAcceptsMemoryIdealV5UnsafeContractRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory-report.json")
	if err := os.WriteFile(
		path,
		[]byte(canonicalMemoryReportString(t, memoryIdealV5UnsafeContractReport())),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := validateMemoryReport(path); err != nil {
		t.Fatalf("validateMemoryReport v5 unsafe contract rows: %v", err)
	}
}

func TestValidateMemoryReportAcceptsMemoryIdealV7FFIRows(t *testing.T) {
	if err := validateMemoryReportString(
		t,
		canonicalMemoryReportString(t, memoryIdealV7FFIReport()),
	); err != nil {
		t.Fatalf("validateMemoryReport v7 FFI rows: %v", err)
	}
}

func TestValidateMemoryReportRejectsV7DerivedFFIRowsWithoutParent(t *testing.T) {
	for _, claim := range []string{
		"ffi_call_may_retain_borrow",
		"ffi_noalias_invalidated_by_external_call",
		"safe_wrapper_promotion_rejected_without_contract",
		"external_pointer_provenance_rejected",
	} {
		t.Run(claim, func(t *testing.T) {
			err := validateMemoryReportString(t, memoryIdealV7ParentlessFFIReport(claim))
			if err == nil || !strings.Contains(err.Error(), "parent_fact_id") {
				t.Fatalf("validateMemoryReport error = %v, want parent_fact_id rejection", err)
			}
		})
	}
}

func TestValidateMemoryReportRejectsValidatedCallbackInoutConservativeRow(t *testing.T) {
	raw := `{
  "schema_version": "tetra.memory-report.v1",
  "rows": [
    {
      "program_id": "program",
      "function_id": "callbackBoundary",
      "site_id": "test.tetra:5:5",
      "source_fact_id": "fact:callback:boundary",
      "parent_fact_id": "fact:callback:borrowed-mut",
      "source_stage": "plir",
      "claim": "callback_inout_conservative",
      "claim_level": "validated",
      "provenance_class": "unsafe_unknown",
      "unsafe_class": "unsafe_unknown",
      "alias_state": "invalidated_by_call",
      "cost_class": "conservative_fallback",
      "validator_name": "callback_alias_conservative_validator",
      "validator_status": "pass",
      "reason": "callback inout boundary cannot validate noalias"
    }
  ]
}`
	err := validateMemoryReportString(t, raw)
	if err == nil || !strings.Contains(err.Error(), "conservative") {
		t.Fatalf(
			"validateMemoryReport error = %v, want conservative noalias boundary validation rejection",
			err,
		)
	}
}

func validateMemoryReportString(t *testing.T, raw string) error {
	t.Helper()
	path := filepath.Join(t.TempDir(), "memory-report.json")
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	return validateMemoryReport(path)
}

func canonicalMemoryReportString(t *testing.T, raw string) string {
	t.Helper()
	var report memoryReport
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		t.Fatalf("parse memory report fixture: %v", err)
	}
	sort.SliceStable(report.Rows, func(i, j int) bool {
		return compareMemoryReportRows(report.Rows[i], report.Rows[j]) < 0
	})
	out, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal canonical memory report fixture: %v", err)
	}
	return string(out)
}

func validSchemaV1MemoryReport() string {
	return `{
  "schema_version": "tetra.memory-report.v1",
  "rows": [
    {
      "program_id": "program",
      "function_id": "main",
      "site_id": "alloc:main:1:1",
      "source_span": "main.tetra:1:1",
      "source_fact_id": "fact:raw:root",
      "parent_fact_id": "",
      "lowered_artifact_id": "ir:main:alloc_bytes:0",
      "source_stage": "validation",
      "claim": "allocation_base_metadata",
      "claim_level": "validated",
      "provenance_class": "unsafe_verified_root",
      "unsafe_class": "unsafe_verified_root",
      "planned_storage": "Heap",
      "actual_lowering_storage": "Heap",
      "cost_class": "zero_cost_proven",
      "validator_name": "raw_bounds_validator",
      "validator_status": "pass",
      "reason": "verified core.alloc_bytes root"
    }
  ]
}`
}

func validSchemaV2AllocationReport() string {
	return `{
  "schema_version": 2,
  "kind": "allocation_plan",
  "target": "linux-x64",
  "functions": [
    {
      "name": "main",
      "allocations": [
        {
          "id": "alloc_bytes:0",
          "site_id": "alloc:main:1:1",
          "builtin": "core.alloc_bytes",
          "length_status": "normal_allocation",
          "zero_guard_status": "invalid_precondition",
          "negative_guard_status": "reject_before_allocation",
          "overflow_guard_status": "reject_before_allocation",
          "planned_storage": "Heap",
          "actual_lowering_storage": "Heap",
          "validation_status": "validated_heap_runtime",
          "lowering_status": "heap_runtime",
          "reason_codes": ["heap.required_dynamic_lifetime"],
          "heap_reason_codes": ["heap.required_dynamic_lifetime"],
          "reason": "fixture allocation report row"
        }
      ]
    }
  ]
}`
}

func removeJSONLineContaining(raw string, needle string) string {
	lines := strings.Split(raw, "\n")
	kept := lines[:0]
	for _, line := range lines {
		if strings.Contains(line, needle) {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

func memoryIdealV5UnsafeContractReport() string {
	return `{
  "schema_version": "tetra.memory-report.v1",
  "rows": [
    {
      "program_id": "program",
      "function_id": "main",
      "site_id": "raw:main:1:1",
      "source_fact_id": "fact:unsafe:unknown:rejected",
      "parent_fact_id": "fact:unsafe:unknown",
      "source_stage": "plir",
      "claim": "unsafe_unknown_rejected_safe_facts",
      "claim_level": "rejected",
      "provenance_class": "unsafe_unknown",
      "unsafe_class": "unsafe_unknown",
      "cost_class": "unsupported_rejected",
      "validator_name": "unsafe_unknown_fact_validator",
      "validator_status": "fail",
      "reason": "unsafe_unknown raw pointer cannot produce safe facts or noalias"
    },
    {
      "program_id": "program",
      "function_id": "main",
      "site_id": "alloc:main:p",
      "source_fact_id": "fact:unsafe:verified:allocation-base",
      "parent_fact_id": "fact:raw:root",
      "source_stage": "allocplan",
      "claim": "unsafe_verified_root_allocation_base",
      "claim_level": "validated",
      "provenance_class": "unsafe_verified_root",
      "unsafe_class": "unsafe_verified_root",
      "cost_class": "zero_cost_proven",
      "validator_name": "unsafe_verified_root_bounds_validator",
      "validator_status": "pass",
      "reason": "core.alloc_bytes verified root may project bounded allocation-base metadata"
    },
    {
      "program_id": "program",
      "function_id": "main",
      "site_id": "raw:main:2:1",
      "source_fact_id": "fact:unsafe:runtime-contract",
      "source_stage": "plir",
      "claim": "unsafe_contract_runtime_checkable",
      "claim_level": "validated",
      "provenance_class": "unsafe_checked",
      "unsafe_class": "unsafe_checked",
      "cost_class": "dynamic_check_required",
      "normal_build_check": true,
      "validator_name": "unsafe_runtime_contract_validator",
      "validator_status": "pass",
      "reason": "nonnull/alignment/length are runtime-checkable unsafe contracts"
    },
    {
      "program_id": "program",
      "function_id": "main",
      "site_id": "raw:main:3:1",
      "source_fact_id": "fact:unsafe:static-contract",
      "source_stage": "plir",
      "claim": "unsafe_contract_static_untrusted",
      "claim_level": "conservative",
      "provenance_class": "unsafe_unknown",
      "unsafe_class": "unsafe_unknown",
      "alias_state": "invalidated_by_call",
      "cost_class": "conservative_fallback",
      "validator_name": "unsafe_static_contract_validator",
      "validator_status": "not_applicable",
      "reason": "unsafe noalias/lifetime/region contracts remain static-untrusted"
    }
  ]
}`
}

func memoryIdealV6BoundsReport() string {
	return `{
  "schema_version": "tetra.memory-report.v1",
  "rows": [
    {
      "program_id": "program",
      "function_id": "sum",
      "site_id": "bounds:sum:3",
      "source_fact_id": "fact:bounds:sum:3:removed",
      "parent_fact_id": "fact:bounds:proof-guard",
      "source_stage": "validation",
      "claim": "bounds_check_removed_with_proof_id",
      "claim_level": "validated",
      "provenance_class": "safe_known",
      "unsafe_class": "safe",
      "cost_class": "zero_cost_proven",
      "proof_id": "proof:while:i:xs:1:1",
      "proof_kind": "bounds_check",
      "proof_subject_base_id": "xs",
      "proof_index_value_id": "local:i",
      "proof_operation": "index_load",
      "proof_range": "i in [0, xs.len)",
      "validator_name": "bounds_proof_id_validator",
      "validator_status": "pass",
      "reason": "removed bounds check has compiler-owned proof id"
    },
    {
      "program_id": "program",
      "function_id": "sum",
      "site_id": "bounds:sum:4",
      "source_fact_id": "fact:bounds:sum:4:retained",
      "parent_fact_id": "fact:bounds:retained-window",
      "source_stage": "validation",
      "claim": "bounds_check_retained_dynamic",
      "claim_level": "validated",
      "provenance_class": "safe_known",
      "unsafe_class": "safe",
      "cost_class": "dynamic_check_required",
      "normal_build_check": true,
      "validator_name": "normal_build_bounds_check_validator",
      "validator_status": "pass",
      "reason": "no proof id exists, so bounds check remains in normal build"
    },
    {
      "program_id": "program",
      "function_id": "sum",
      "site_id": "bounds:sum:5",
      "source_fact_id": "fact:bounds:sum:5:missing-proof",
      "source_stage": "validation",
      "claim": "bounds_check_removal_rejected_missing_proof_id",
      "claim_level": "rejected",
      "provenance_class": "safe_known",
      "unsafe_class": "safe",
      "cost_class": "unsupported_rejected",
      "validator_name": "bounds_proof_id_validator",
      "validator_status": "fail",
      "reason": "removed bounds check without proof id is rejected"
    },
    {
      "program_id": "program",
      "function_id": "raw",
      "site_id": "raw:load:1",
      "source_fact_id": "fact:raw:bounds:check",
      "parent_fact_id": "fact:raw:gateway",
      "source_stage": "plir",
      "claim": "raw_bounds_runtime_check_normal_build",
      "claim_level": "validated",
      "provenance_class": "unsafe_checked",
      "unsafe_class": "unsafe_checked",
      "cost_class": "dynamic_check_required",
      "normal_build_check": true,
      "validator_name": "raw_bounds_width_validator",
      "validator_status": "pass",
      "reason": "raw bounds uncertainty keeps a normal-build check or trap"
    }
  ]
}`
}

func memoryIdealV7FFIReport() string {
	return `{
  "schema_version": "tetra.memory-report.v1",
  "rows": [
    {
      "program_id": "program",
      "function_id": "ffiV7",
      "site_id": "ffi:external:1",
      "source_fact_id": "fact:ffi:external",
      "source_stage": "plir",
      "claim": "ffi_pointer_external_unknown",
      "claim_level": "conservative",
      "provenance_class": "unsafe_unknown",
      "unsafe_class": "unsafe_unknown",
      "cost_class": "conservative_fallback",
      "validator_name": "external_pointer_provenance_validator",
      "validator_status": "not_applicable",
      "reason": "external pointer remains unsafe_unknown"
    },
    {
      "program_id": "program",
      "function_id": "ffiV7",
      "site_id": "ffi:call:1",
      "source_fact_id": "fact:ffi:retain-borrow",
      "parent_fact_id": "fact:ffi:external",
      "source_stage": "plir",
      "claim": "ffi_call_may_retain_borrow",
      "claim_level": "conservative",
      "provenance_class": "unsafe_unknown",
      "unsafe_class": "unsafe_unknown",
      "cost_class": "conservative_fallback",
      "validator_name": "ffi_lifetime_conservative_validator",
      "validator_status": "not_applicable",
      "reason": "external call may retain borrowed pointer"
    },
    {
      "program_id": "program",
      "function_id": "ffiV7",
      "site_id": "ffi:wrap:1",
      "source_fact_id": "fact:ffi:safe-wrapper-rejected",
      "parent_fact_id": "fact:ffi:external",
      "source_stage": "plir",
      "claim": "safe_wrapper_promotion_rejected_without_contract",
      "claim_level": "rejected",
      "provenance_class": "unsafe_unknown",
      "unsafe_class": "unsafe_unknown",
      "cost_class": "unsupported_rejected",
      "validator_name": "safe_wrapper_promotion_validator",
      "validator_status": "fail",
      "reason": "safe wrapper promotion from external pointer requires compiler-owned contract"
    },
    {
      "program_id": "program",
      "function_id": "ffiV7",
      "site_id": "ffi:call:2",
      "source_fact_id": "fact:ffi:noalias-invalidated",
      "parent_fact_id": "fact:ffi:external",
      "source_stage": "plir",
      "claim": "ffi_noalias_invalidated_by_external_call",
      "claim_level": "conservative",
      "provenance_class": "unsafe_unknown",
      "unsafe_class": "unsafe_unknown",
      "alias_state": "invalidated_by_call",
      "cost_class": "conservative_fallback",
      "validator_name": "ffi_noalias_conservative_validator",
      "validator_status": "not_applicable",
      "reason": "external call invalidates broad noalias"
    },
    {
      "program_id": "program",
      "function_id": "ffiV7",
      "site_id": "ffi:external:2",
      "source_fact_id": "fact:ffi:external-provenance-rejected",
      "parent_fact_id": "fact:ffi:external",
      "source_stage": "plir",
      "claim": "external_pointer_provenance_rejected",
      "claim_level": "rejected",
      "provenance_class": "unsafe_unknown",
      "unsafe_class": "unsafe_unknown",
      "cost_class": "unsupported_rejected",
      "validator_name": "external_pointer_provenance_validator",
      "validator_status": "fail",
      "reason": "external pointer cannot become provenance_known without compiler-owned proof"
    }
  ]
}`
}

func memoryIdealV7ParentlessFFIReport(claim string) string {
	level := "conservative"
	status := "not_applicable"
	cost := "conservative_fallback"
	alias := ""
	if claim == "safe_wrapper_promotion_rejected_without_contract" ||
		claim == "external_pointer_provenance_rejected" {
		level = "rejected"
		status = "fail"
		cost = "unsupported_rejected"
	}
	if claim == "ffi_noalias_invalidated_by_external_call" {
		alias = "\n      \"alias_state\": \"invalidated_by_call\","
	}
	return `{
  "schema_version": "tetra.memory-report.v1",
  "rows": [
    {
      "program_id": "program",
      "function_id": "ffiV7",
      "site_id": "ffi:call:parentless",
      "source_fact_id": "fact:ffi:parentless",
      "source_stage": "plir",
      "claim": "` + claim + `",
      "claim_level": "` + level + `",
      "provenance_class": "unsafe_unknown",
      "unsafe_class": "unsafe_unknown",` + alias + `
      "cost_class": "` + cost + `",
      "validator_name": "memory_report_schema_v1",
      "validator_status": "` + status + `",
      "reason": "v7 derived FFI row without parent should be rejected"
    }
  ]
}`
}

func rawSliceRejectedMemoryReport() string {
	return `{
  "schema_version": "tetra.memory-report.v1",
  "rows": [
    {
      "program_id": "program",
      "function_id": "main",
      "site_id": "raw-slice:main:2:1",
      "source_span": "main.tetra:2:1",
      "source_fact_id": "fact:raw-slice:negative",
      "source_stage": "plir",
      "claim": "rejected_negative_length",
      "claim_level": "evidence_only",
      "provenance_class": "unsafe_checked",
      "unsafe_class": "unsafe_checked",
      "cost_class": "unsupported_rejected",
      "validator_name": "memory_report_schema_v1",
      "validator_status": "not_applicable",
      "reason": "negative raw slice length rejected before view construction"
    },
    {
      "program_id": "program",
      "function_id": "main",
      "site_id": "raw-slice:main:3:1",
      "source_span": "main.tetra:3:1",
      "source_fact_id": "fact:raw-slice:overflow",
      "source_stage": "plir",
      "claim": "rejected_length_overflow",
      "claim_level": "evidence_only",
      "provenance_class": "unsafe_checked",
      "unsafe_class": "unsafe_checked",
      "cost_class": "unsupported_rejected",
      "validator_name": "memory_report_schema_v1",
      "validator_status": "not_applicable",
      "reason": "raw slice length byte computation overflow rejected before view construction"
    }
  ]
}`
}

func conservativeUnknownRawPointerReport() string {
	return `{
  "schema_version": "tetra.memory-report.v1",
  "rows": [
    {
      "program_id": "program",
      "function_id": "main",
      "site_id": "raw:main:2:1",
      "source_span": "main.tetra:2:1",
      "source_fact_id": "fact:raw:unknown",
      "source_stage": "plir",
      "claim": "checked_external_unknown",
      "claim_level": "conservative",
      "provenance_class": "unsafe_unknown",
      "unsafe_class": "unsafe_unknown",
      "cost_class": "conservative_fallback",
      "validator_name": "memory_report_schema_v1",
      "validator_status": "not_applicable",
      "reason": "unknown raw pointer remains conservative"
    }
  ]
}`
}
