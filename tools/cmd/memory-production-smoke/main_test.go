package main

import (
	"encoding/json"
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
	} {
		if !hasCase(cases, want) {
			t.Fatalf("requiredPassingCases missing %q", want)
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
