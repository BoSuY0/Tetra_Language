package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"tetra_language/tools/validators/memoryprod"
)

func TestBuildReportProducesValidMemoryProductionEvidence(t *testing.T) {
	report := buildReport("tools/cmd/memory-production-smoke", []memoryprod.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "/tmp/tetra", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "memory smoke app", Kind: "app", Path: "/tmp/memory-smoke", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "memory stress", Kind: "stress", Path: "/tmp/memory-stress", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "memory fuzz", Kind: "stress", Path: "/tmp/memory-fuzz", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "actornet close-without-cancel leak coverage", Kind: "stress", Path: "go test -buildvcs=false ./cli/internal/actornet -run TestBrokerCloseWithoutCancelStopsServeWatcher -count=1", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "compiler resource finalization diagnostics", Kind: "stress", Path: "go test -buildvcs=false ./compiler/tests/runtime -run ^(TestTaskHandleFinalization|TestTaskGroupFinalization|TestIslandFinalization) -count=1", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, requiredPassingBenchmarks(), requiredPassingCases())
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := memoryprod.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestSmallHeapBenchmarkSourceCreatesEscapingSmallHeapCandidates(t *testing.T) {
	src := smallHeapBenchmarkSource(3, 32)
	for _, want := range []string{
		"func make_00() -> []u8",
		"var xs: []u8 = make_u8(32)",
		"return xs",
		"let xs_02: []u8 = make_02()",
		"return xs_00.len + xs_01.len + xs_02.len",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("smallHeapBenchmarkSource missing %q:\n%s", want, src)
		}
	}
}

func TestRequiredPassingCasesIncludeMemoryProductionEdgeCases(t *testing.T) {
	cases := requiredPassingCases()
	for _, want := range []string{
		"cap.mem unsafe boundary",
		"callable mutable capture heap escape",
		"function-typed slice aggregate borrow escape coverage",
		"raw ptr_add negative offset bounds",
		"raw ptr_add allocation upper bound",
		"raw allocation-base i32 access width",
		"raw allocation-base ptr access width",
		"raw allocation-base store_i32 access width",
		"raw allocation-base load_ptr access width",
		"raw slice negative length",
		"raw slice i32 length byte overflow",
		"allocation make zero length canonical empty",
		"allocation make negative length",
		"allocation make byte-size overflow",
		"allocation island zero length no metadata",
		"allocation island negative length",
		"allocation island byte-size overflow",
		"allocation length contract report",
		"actornet broker close-without-cancel leak smoke",
		"compiler resource finalization diagnostics",
	} {
		if !hasCase(cases, want) {
			t.Fatalf("requiredPassingCases missing %q", want)
		}
	}
}

func TestResourceFinalizationCoverageIsReleaseBlocking(t *testing.T) {
	exit0 := 0
	report := buildReport("tools/cmd/memory-production-smoke", []memoryprod.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "go build ./cli/cmd/tetra", Ran: true, Pass: true, ExitCode: &exit0},
		{Name: "memory smoke app", Kind: "app", Path: "examples/core_memory_smoke", Ran: true, Pass: true, ExitCode: &exit0},
		{Name: "memory stress", Kind: "stress", Path: "tools/cmd/memory-production-smoke", Ran: true, Pass: true, ExitCode: &exit0},
	}, requiredPassingBenchmarks(), requiredPassingCases())
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	err = memoryprod.ValidateReport(raw)
	if err == nil {
		t.Fatalf("ValidateReport accepted release evidence without leak/resource process rows")
	}
	for _, want := range []string{
		"actornet close-without-cancel leak coverage",
		"compiler resource finalization diagnostics",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("ValidateReport error missing %q:\n%v", want, err)
		}
	}
}

func TestRuntimeDiagnosticPassRequiresExpectedErrorText(t *testing.T) {
	if runtimeDiagnosticPass(processResult{exitCode: 2, output: "different runtime trap"}, 2, "negative ptr_add offset", true) {
		t.Fatalf("runtimeDiagnosticPass accepted matching exit code without expected error text")
	}
	if !runtimeDiagnosticPass(processResult{exitCode: 2, output: "fatal: negative ptr_add offset"}, 2, "negative ptr_add offset", true) {
		t.Fatalf("runtimeDiagnosticPass rejected matching exit code and expected error text")
	}
	if !runtimeDiagnosticPass(processResult{exitCode: 2, output: ""}, 2, "legacy diagnostic", false) {
		t.Fatalf("runtimeDiagnosticPass rejected legacy exit-code-only diagnostic")
	}
}

func TestRawSliceRuntimeDiagnosticSourcesAvoidPreCallAllocationTrap(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
	}{
		{name: "negative", source: rawSliceNegativeLengthSource},
		{name: "i32 byte overflow", source: rawSliceI32LengthOverflowSource},
	} {
		if strings.Contains(tc.source, "alloc_bytes") {
			t.Fatalf("%s raw slice diagnostic source must not contain a pre-call alloc trap:\n%s", tc.name, tc.source)
		}
		if !strings.Contains(tc.source, "let p: ptr = 0") {
			t.Fatalf("%s raw slice diagnostic source must use a non-trapping null raw pointer:\n%s", tc.name, tc.source)
		}
		if !strings.Contains(tc.source, "return xs.len + 98") {
			t.Fatalf("%s raw slice diagnostic source must keep a fallthrough data dependency on xs.len:\n%s", tc.name, tc.source)
		}
	}
}

func TestParseMemoryReportClaimsCountsRawBoundsRows(t *testing.T) {
	claims, err := parseMemoryReportClaims([]byte(`{
		"schema_version": "tetra.memory-report.v1",
		"rows": [
			{"claim": "allocation_base_metadata"},
			{"claim": "derived_allocation_offset"},
			{"claim": "rejected_negative_offset"},
			{"claim": "rejected_upper_bound"},
			{"claim": "rejected_access_width_overflow"},
			{"claim": "checked_external_unknown"},
			{"claim": "rejected_negative_length"},
			{"claim": "rejected_length_overflow"},
			{"claim": "external_unknown"},
			{"claim": "external_unknown"}
		]
	}`))
	if err != nil {
		t.Fatalf("parseMemoryReportClaims: %v", err)
	}
	for _, want := range []string{
		"allocation_base_metadata",
		"derived_allocation_offset",
		"rejected_negative_offset",
		"rejected_upper_bound",
		"rejected_access_width_overflow",
		"checked_external_unknown",
		"rejected_negative_length",
		"rejected_length_overflow",
	} {
		if claims[want] != 1 {
			t.Fatalf("claim %s count = %d, want 1 in %+v", want, claims[want], claims)
		}
	}
	if claims["external_unknown"] != 2 {
		t.Fatalf("external_unknown count = %d, want 2 in %+v", claims["external_unknown"], claims)
	}
}

func TestParseAllocationReportSummaryIncludesLengthContractRows(t *testing.T) {
	report, err := parseAllocationReportSummary([]byte(`{
		"schema_version": 2,
		"kind": "allocation_plan",
		"summary": {
			"allocation_count": 2,
			"runtime_paths": {"heap": 1, "explicit_island": 1},
			"allocator_reuse_policies": {},
			"bytes_reserved": 16
		},
		"functions": [
			{
				"name": "main",
				"allocations": [
					{
						"id": "alloc:0",
						"builtin": "core.make_u8",
						"length_status": "valid_empty_allocation",
						"zero_guard_status": "valid_empty_no_allocator",
						"negative_guard_status": "reject_before_allocation",
						"overflow_guard_status": "reject_before_allocation"
					},
					{
						"id": "alloc:1",
						"builtin": "core.island_make_i32",
						"length_status": "rejected_byte_size_overflow",
						"zero_guard_status": "valid_empty_no_metadata_access",
						"negative_guard_status": "reject_before_metadata_access",
						"overflow_guard_status": "reject_before_metadata_access"
					}
				]
			}
		]
	}`))
	if err != nil {
		t.Fatalf("parseAllocationReportSummary: %v", err)
	}
	if len(report.Functions) != 1 || len(report.Functions[0].Allocations) != 2 {
		t.Fatalf("parsed allocation rows = %+v, want function allocations", report.Functions)
	}
	if report.Functions[0].Allocations[0].LengthStatus != "valid_empty_allocation" {
		t.Fatalf("first allocation length_status = %q", report.Functions[0].Allocations[0].LengthStatus)
	}
	if report.Functions[0].Allocations[1].OverflowGuardStatus != "reject_before_metadata_access" {
		t.Fatalf("second allocation overflow_guard_status = %q", report.Functions[0].Allocations[1].OverflowGuardStatus)
	}
}

func TestParseAllocationReportSummaryAllowsReportsWithoutAllocatorReusePolicy(t *testing.T) {
	if _, err := parseAllocationReportSummary([]byte(`{
		"schema_version": 2,
		"kind": "allocation_plan",
		"summary": {
			"allocation_count": 1,
			"runtime_paths": {"explicit_island": 1},
			"bytes_reserved": 0
		},
		"functions": [
			{
				"name": "main",
				"allocations": [
					{
						"id": "alloc:0",
						"builtin": "core.island_make_u8",
						"length_status": "valid_empty_allocation",
						"zero_guard_status": "valid_empty_no_metadata_access",
						"negative_guard_status": "reject_before_metadata_access",
						"overflow_guard_status": "reject_before_metadata_access"
					}
				]
			}
		]
	}`)); err != nil {
		t.Fatalf("parseAllocationReportSummary rejected allocation report without allocator_reuse_policies: %v", err)
	}
}

func TestRawPointerBoundsMetadataSourceCoversRuntimeRawFailureClasses(t *testing.T) {
	for _, want := range []string{
		"let i32_value: Int = core.load_i32(i32_ptr, mem)",
		"let ptr_value: ptr = core.store_ptr(ptr_ptr, ptr_base, mem)",
		"let store_i32_status: Int = core.store_i32(store_i32_ptr, 123, mem)",
		"let load_ptr_value: ptr = core.load_ptr(load_ptr_ptr, mem)",
		"let raw_slice_overflow: []i32 = core.raw_slice_i32_from_parts(raw_slice_base, 536870912, mem)",
	} {
		if !strings.Contains(rawPointerBoundsMetadataSource, want) {
			t.Fatalf("rawPointerBoundsMetadataSource missing runtime-correlated source %q:\n%s", want, rawPointerBoundsMetadataSource)
		}
	}
}

func TestRawPointerBoundsMetadataReportValidationUsesPairedAllocReport(t *testing.T) {
	args := validateMemoryReportCommand("out/raw-pointer-bounds-metadata")
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"--report out/raw-pointer-bounds-metadata.memory.json",
		"--alloc-report out/raw-pointer-bounds-metadata.alloc.json",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("validate memory report command missing %q: %v", want, args)
		}
	}
}

func TestValidateRawPointerBoundsCorrelationRejectsClaimOnlyRows(t *testing.T) {
	err := validateRawPointerBoundsCorrelation(requiredPassingCases(), []byte(`{
		"schema_version": "tetra.memory-report.v1",
		"rows": [
			{"claim": "allocation_base_metadata"},
			{"claim": "derived_allocation_offset"},
			{"claim": "rejected_negative_offset"},
			{"claim": "rejected_upper_bound"},
			{"claim": "rejected_access_width_overflow"},
			{"claim": "rejected_access_width_overflow"},
			{"claim": "rejected_access_width_overflow"},
			{"claim": "rejected_access_width_overflow"},
			{"claim": "checked_external_unknown"},
			{"claim": "external_unknown"},
			{"claim": "raw_slice_verified_allocation_root"},
			{"claim": "rejected_negative_length"},
			{"claim": "raw_bounds_runtime_check_normal_build"}
		]
	}`))
	if err == nil || !strings.Contains(err.Error(), "source_fact_id") {
		t.Fatalf("validateRawPointerBoundsCorrelation error = %v, want source_fact_id rejection", err)
	}
}

func TestValidateRawPointerBoundsCorrelationRequiresParentedNormalBuildChecks(t *testing.T) {
	err := validateRawPointerBoundsCorrelation(requiredPassingCases(), []byte(correlatedRawBoundsMemoryReport(false)))
	if err == nil || !strings.Contains(err.Error(), "raw_bounds_runtime_check_normal_build") || !strings.Contains(err.Error(), "normal_build_check") {
		t.Fatalf("validateRawPointerBoundsCorrelation error = %v, want parented normal-build check rejection", err)
	}
}

func TestValidateRawPointerBoundsCorrelationAcceptsRuntimeAndReportEvidence(t *testing.T) {
	if err := validateRawPointerBoundsCorrelation(requiredPassingCases(), []byte(correlatedRawBoundsMemoryReport(true))); err != nil {
		t.Fatalf("validateRawPointerBoundsCorrelation: %v", err)
	}
}

func TestValidateRawPointerBoundsCorrelationRejectsCapMemProofRows(t *testing.T) {
	err := validateRawPointerBoundsCorrelation(requiredPassingCases(), []byte(correlatedRawBoundsMemoryReportWithCapMemProof()))
	if err == nil || !strings.Contains(err.Error(), "cap.mem") || !strings.Contains(err.Error(), "no_alias") {
		t.Fatalf("validateRawPointerBoundsCorrelation error = %v, want cap.mem no_alias proof rejection", err)
	}
}

func TestValidateRawSliceGatewayCorrelationRejectsClaimOnlyRows(t *testing.T) {
	err := validateRawSliceGatewayCorrelation(requiredPassingCases(), []byte(`{
		"schema_version": "tetra.memory-report.v1",
		"rows": [
			{"claim": "external_unknown"},
			{"claim": "raw_slice_verified_allocation_root"},
			{"claim": "rejected_negative_length"},
			{"claim": "rejected_length_overflow"},
			{"claim": "raw_bounds_runtime_check_normal_build"}
		]
	}`))
	if err == nil || !strings.Contains(err.Error(), "source_fact_id") {
		t.Fatalf("validateRawSliceGatewayCorrelation error = %v, want source_fact_id rejection", err)
	}
}

func TestValidateRawSliceGatewayCorrelationRequiresParentedOverflowCheck(t *testing.T) {
	err := validateRawSliceGatewayCorrelation(requiredPassingCases(), []byte(correlatedRawBoundsMemoryReport(false)))
	if err == nil || !strings.Contains(err.Error(), "rejected_length_overflow") || !strings.Contains(err.Error(), "raw_bounds_runtime_check_normal_build") {
		t.Fatalf("validateRawSliceGatewayCorrelation error = %v, want raw-slice overflow normal-build check rejection", err)
	}
}

func TestValidateRawSliceGatewayCorrelationAcceptsRuntimeAndReportEvidence(t *testing.T) {
	if err := validateRawSliceGatewayCorrelation(requiredPassingCases(), []byte(correlatedRawBoundsMemoryReport(true))); err != nil {
		t.Fatalf("validateRawSliceGatewayCorrelation: %v", err)
	}
}

func requiredPassingBenchmarks() []memoryprod.BenchmarkReport {
	return []memoryprod.BenchmarkReport{{
		Name:             "small heap allocation syscall reduction",
		Kind:             "allocator",
		Metric:           "estimated_os_syscalls",
		Unit:             "syscalls",
		BaselineValue:    64,
		MeasuredValue:    1,
		ImprovementRatio: 64,
		Evidence:         "allocation report schema v2 shows 64 per_core_small_heap rows with same_core_same_size_class_free_list reuse policy inside one 64KiB chunk refill",
		Ran:              true,
		Pass:             true,
	}}
}

func hasCase(cases []memoryprod.CaseReport, name string) bool {
	for _, c := range cases {
		if c.Name == name {
			return true
		}
	}
	return false
}

func correlatedRawBoundsMemoryReport(includeChecks bool) string {
	rows := []string{
		capMemAuthorizationReportRow("fact:cap-mem-authorization"),
		rawMemoryReportRow("fact:alloc", "", "allocation_base_metadata", "validated", "unsafe_verified_root", "unsafe_verified_root", "zero_cost_proven", false, "raw_bounds_validator"),
		rawMemoryReportRow("fact:derived", "", "derived_allocation_offset", "evidence_only", "unsafe_checked", "unsafe_checked", "dynamic_check_required", true, "raw_bounds_validator"),
		rawMemoryReportRow("fact:negative", "", "rejected_negative_offset", "evidence_only", "unsafe_checked", "unsafe_checked", "unsupported_rejected", false, "raw_bounds_validator"),
		rawMemoryReportRow("fact:upper", "", "rejected_upper_bound", "evidence_only", "unsafe_checked", "unsafe_checked", "unsupported_rejected", false, "raw_bounds_validator"),
		rawMemoryReportRow("fact:width-load-i32", "", "rejected_access_width_overflow", "evidence_only", "unsafe_checked", "unsafe_checked", "unsupported_rejected", false, "raw_bounds_validator"),
		rawMemoryReportRow("fact:width-store-ptr", "", "rejected_access_width_overflow", "evidence_only", "unsafe_checked", "unsafe_checked", "unsupported_rejected", false, "raw_bounds_validator"),
		rawMemoryReportRow("fact:width-store-i32", "", "rejected_access_width_overflow", "evidence_only", "unsafe_checked", "unsafe_checked", "unsupported_rejected", false, "raw_bounds_validator"),
		rawMemoryReportRow("fact:width-load-ptr", "", "rejected_access_width_overflow", "evidence_only", "unsafe_checked", "unsafe_checked", "unsupported_rejected", false, "raw_bounds_validator"),
		rawMemoryReportRow("fact:unknown", "", "checked_external_unknown", "conservative", "unsafe_unknown", "unsafe_unknown", "conservative_fallback", false, "raw_bounds_validator"),
		rawMemoryReportRow("fact:external-slice", "", "external_unknown", "conservative", "unsafe_unknown", "unsafe_unknown", "conservative_fallback", false, "raw_bounds_validator"),
		rawMemoryReportRow("fact:raw-slice-root", "", "raw_slice_verified_allocation_root", "evidence_only", "unsafe_checked", "unsafe_checked", "dynamic_check_required", true, "raw_bounds_validator"),
		rawMemoryReportRow("fact:raw-slice-negative", "", "rejected_negative_length", "evidence_only", "unsafe_checked", "unsafe_checked", "unsupported_rejected", false, "raw_bounds_validator"),
		rawMemoryReportRow("fact:raw-slice-overflow", "", "rejected_length_overflow", "evidence_only", "unsafe_checked", "unsafe_checked", "unsupported_rejected", false, "raw_bounds_validator"),
	}
	if includeChecks {
		for _, parent := range []string{
			"fact:width-load-i32",
			"fact:width-store-ptr",
			"fact:width-store-i32",
			"fact:width-load-ptr",
			"fact:raw-slice-overflow",
		} {
			rows = append(rows, rawMemoryReportRow("fact:check:"+parent, parent, "raw_bounds_runtime_check_normal_build", "validated", "unsafe_checked", "unsafe_checked", "dynamic_check_required", true, "raw_bounds_width_validator"))
		}
	}
	return `{
		"schema_version": "tetra.memory-report.v1",
		"rows": [
			` + strings.Join(rows, ",\n") + `
		]
	}`
}

func correlatedRawBoundsMemoryReportWithCapMemProof() string {
	report := correlatedRawBoundsMemoryReport(true)
	badRow := rawMemoryReportRow("fact:cap-mem-noalias", "", "no_alias", "validated", "safe_known", "safe", "zero_cost_proven", false, "cap_mem_authorization_validator")
	return strings.Replace(report, "\n\t\t]\n\t}", ",\n"+badRow+"\n\t\t]\n\t}", 1)
}

func capMemAuthorizationReportRow(sourceFactID string) string {
	return fmt.Sprintf(`{
			"source_fact_id": %q,
			"source_stage": "plir",
			"claim": "cap_mem_authorization_only",
			"claim_level": "evidence_only",
			"provenance_class": "unsafe_checked",
			"unsafe_class": "unsafe_checked",
			"cost_class": "instrumentation_only",
			"validator_status": "not_run",
			"reason": "cap.mem authorizes raw operations only"
		}`, sourceFactID)
}

func rawMemoryReportRow(sourceFactID, parentFactID, claim, claimLevel, provenance, unsafeClass, cost string, normalBuildCheck bool, validator string) string {
	parent := ""
	if parentFactID != "" {
		parent = fmt.Sprintf(`,
			"parent_fact_id": %q`, parentFactID)
	}
	normalCheck := ""
	if normalBuildCheck {
		normalCheck = `,
			"normal_build_check": true`
	}
	return fmt.Sprintf(`{
			"source_fact_id": %q%s,
			"source_stage": "plir",
			"claim": %q,
			"claim_level": %q,
			"provenance_class": %q,
			"unsafe_class": %q,
			"cost_class": %q%s,
			"validator_name": %q,
			"validator_status": "pass"
		}`, sourceFactID, parent, claim, claimLevel, provenance, unsafeClass, cost, normalCheck, validator)
}
