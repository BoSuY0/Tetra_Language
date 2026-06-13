package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"tetra_language/tools/validators/memoryprod"
)

func validateMemoryReportCommand(outPath string) []string {
	return []string{
		"run", "./tools/cmd/validate-memory-report",
		"--report", outPath + ".memory.json",
		"--alloc-report", outPath + ".alloc.json",
	}
}

type allocationReportSummary struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind,omitempty"`
	Summary       struct {
		AllocationCount          int            `json:"allocation_count"`
		RuntimePaths             map[string]int `json:"runtime_paths"`
		AllocatorClasses         map[string]int `json:"allocator_classes"`
		AllocatorReusePolicies   map[string]int `json:"allocator_reuse_policies"`
		RawPointerBoundsStatuses map[string]int `json:"raw_pointer_bounds_statuses"`
		RawSlicePolicies         map[string]int `json:"raw_slice_policies"`
		BytesReserved            int            `json:"bytes_reserved"`
	} `json:"summary"`
	Functions []allocationReportFunctionSummary `json:"functions,omitempty"`
}

type allocationReportFunctionSummary struct {
	Name        string                              `json:"name"`
	Allocations []allocationReportAllocationSummary `json:"allocations,omitempty"`
}

type allocationReportAllocationSummary struct {
	ID                  string `json:"id"`
	Builtin             string `json:"builtin,omitempty"`
	LengthStatus        string `json:"length_status,omitempty"`
	ZeroGuardStatus     string `json:"zero_guard_status,omitempty"`
	NegativeGuardStatus string `json:"negative_guard_status,omitempty"`
	OverflowGuardStatus string `json:"overflow_guard_status,omitempty"`
}

type memoryReportEvidenceRow struct {
	SourceFactID     string `json:"source_fact_id,omitempty"`
	ParentFactID     string `json:"parent_fact_id,omitempty"`
	Claim            string `json:"claim"`
	ClaimLevel       string `json:"claim_level,omitempty"`
	ProvenanceClass  string `json:"provenance_class,omitempty"`
	UnsafeClass      string `json:"unsafe_class,omitempty"`
	ValidatorName    string `json:"validator_name,omitempty"`
	ValidatorStatus  string `json:"validator_status,omitempty"`
	CostClass        string `json:"cost_class,omitempty"`
	NormalBuildCheck bool   `json:"normal_build_check,omitempty"`
	Reason           string `json:"reason,omitempty"`
}

type memoryReportEvidence struct {
	SchemaVersion string                    `json:"schema_version"`
	Rows          []memoryReportEvidenceRow `json:"rows"`
}

func parseAllocationReportSummary(raw []byte) (allocationReportSummary, error) {
	var report allocationReportSummary
	if err := json.Unmarshal(raw, &report); err != nil {
		return allocationReportSummary{}, fmt.Errorf("parse small heap allocation report: %w", err)
	}
	if report.SchemaVersion != 2 {
		return allocationReportSummary{}, fmt.Errorf("small heap allocation report schema_version = %d, want 2", report.SchemaVersion)
	}
	if report.Summary.AllocationCount <= 0 {
		return allocationReportSummary{}, fmt.Errorf("small heap allocation report allocation_count = %d, want positive", report.Summary.AllocationCount)
	}
	if report.Summary.RuntimePaths == nil {
		return allocationReportSummary{}, fmt.Errorf("small heap allocation report missing runtime_paths summary")
	}
	return report, nil
}

func validateAllocationLengthContractCorrelation(cases []memoryprod.CaseReport, report allocationReportSummary) error {
	var issues []string
	issues = append(issues, validateAllocationLengthRuntimeCases(cases)...)
	requireRows := []allocationReportAllocationSummary{
		{
			Builtin:             "core.alloc_bytes",
			LengthStatus:        "invalid_length_contract",
			ZeroGuardStatus:     "invalid_precondition",
			NegativeGuardStatus: "reject_before_allocation",
			OverflowGuardStatus: "reject_before_allocation",
		},
		{
			Builtin:             "core.make_u8",
			LengthStatus:        "valid_empty_allocation",
			ZeroGuardStatus:     "valid_empty_no_allocator",
			NegativeGuardStatus: "reject_before_allocation",
			OverflowGuardStatus: "reject_before_allocation",
		},
		{
			Builtin:             "core.make_u16",
			LengthStatus:        "rejected_negative_length",
			ZeroGuardStatus:     "valid_empty_no_allocator",
			NegativeGuardStatus: "reject_before_allocation",
			OverflowGuardStatus: "reject_before_allocation",
		},
		{
			Builtin:             "core.make_i32",
			LengthStatus:        "rejected_byte_size_overflow",
			ZeroGuardStatus:     "valid_empty_no_allocator",
			NegativeGuardStatus: "reject_before_allocation",
			OverflowGuardStatus: "reject_before_allocation",
		},
		{
			Builtin:             "core.island_make_u8",
			LengthStatus:        "valid_empty_allocation",
			ZeroGuardStatus:     "valid_empty_no_metadata_access",
			NegativeGuardStatus: "reject_before_metadata_access",
			OverflowGuardStatus: "reject_before_metadata_access",
		},
		{
			Builtin:             "core.island_make_u16",
			LengthStatus:        "rejected_negative_length",
			ZeroGuardStatus:     "valid_empty_no_metadata_access",
			NegativeGuardStatus: "reject_before_metadata_access",
			OverflowGuardStatus: "reject_before_metadata_access",
		},
		{
			Builtin:             "core.island_make_i32",
			LengthStatus:        "rejected_byte_size_overflow",
			ZeroGuardStatus:     "valid_empty_no_metadata_access",
			NegativeGuardStatus: "reject_before_metadata_access",
			OverflowGuardStatus: "reject_before_metadata_access",
		},
	}
	for _, req := range requireRows {
		if !allocationReportHasLengthContract(report, req) {
			issues = append(issues, fmt.Sprintf("allocation report missing contract row builtin=%s length_status=%s zero=%s negative=%s overflow=%s", req.Builtin, req.LengthStatus, req.ZeroGuardStatus, req.NegativeGuardStatus, req.OverflowGuardStatus))
		}
	}
	if len(issues) > 0 {
		return fmt.Errorf(strings.Join(issues, "; "))
	}
	return nil
}

func validateAllocationLengthRuntimeCases(cases []memoryprod.CaseReport) []string {
	byName := map[string]memoryprod.CaseReport{}
	for _, c := range cases {
		byName[c.Name] = c
	}
	var issues []string
	for _, req := range []struct {
		name          string
		expectedError string
	}{
		{name: "allocation make zero length canonical empty"},
		{name: "allocation make negative length", expectedError: "negative allocation length"},
		{name: "allocation make byte-size overflow", expectedError: "allocation length byte overflow"},
		{name: "allocation island zero length no metadata"},
		{name: "allocation island negative length", expectedError: "negative allocation length"},
		{name: "allocation island byte-size overflow", expectedError: "allocation length byte overflow"},
	} {
		c, ok := byName[req.name]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing allocation length runtime case %q", req.name))
			continue
		}
		if !c.Ran || !c.Pass {
			issues = append(issues, fmt.Sprintf("allocation length runtime case %q did not pass", req.name))
		}
		if req.expectedError != "" && !strings.Contains(c.ExpectedError, req.expectedError) {
			issues = append(issues, fmt.Sprintf("allocation length runtime case %q expected_error = %q, want %q", req.name, c.ExpectedError, req.expectedError))
		}
	}
	return issues
}

func allocationReportHasLengthContract(report allocationReportSummary, req allocationReportAllocationSummary) bool {
	for _, fn := range report.Functions {
		for _, alloc := range fn.Allocations {
			if alloc.Builtin == req.Builtin &&
				alloc.LengthStatus == req.LengthStatus &&
				alloc.ZeroGuardStatus == req.ZeroGuardStatus &&
				alloc.NegativeGuardStatus == req.NegativeGuardStatus &&
				alloc.OverflowGuardStatus == req.OverflowGuardStatus {
				return true
			}
		}
	}
	return false
}

func parseMemoryReportClaims(raw []byte) (map[string]int, error) {
	report, err := parseMemoryReportEvidence(raw)
	if err != nil {
		return nil, err
	}
	claims := map[string]int{}
	for _, row := range report.Rows {
		claims[row.Claim]++
	}
	return claims, nil
}

func parseMemoryReportEvidence(raw []byte) (memoryReportEvidence, error) {
	var report memoryReportEvidence
	if err := json.Unmarshal(raw, &report); err != nil {
		return memoryReportEvidence{}, fmt.Errorf("parse memory report: %w", err)
	}
	if report.SchemaVersion != "tetra.memory-report.v1" {
		return memoryReportEvidence{}, fmt.Errorf("memory report schema_version = %q, want tetra.memory-report.v1", report.SchemaVersion)
	}
	return report, nil
}

func validateRawPointerBoundsCorrelation(cases []memoryprod.CaseReport, memoryRaw []byte) error {
	report, err := parseMemoryReportEvidence(memoryRaw)
	if err != nil {
		return err
	}
	var issues []string
	issues = append(issues, validateRawRuntimeCases(cases)...)
	issues = append(issues, validateCapMemAuthorizationDiscipline(report.Rows)...)
	checksByParent := parentedRawBoundsRuntimeChecks(report.Rows)

	requireRows := []struct {
		claim           string
		minRows         int
		validate        func(memoryReportEvidenceRow) []string
		requireCheckRow bool
	}{
		{claim: "allocation_base_metadata", minRows: 1, validate: validateUnsafeVerifiedRootAllocationBaseRow},
		{claim: "derived_allocation_offset", minRows: 1, validate: validateUnsafeCheckedDynamicRow},
		{claim: "rejected_negative_offset", minRows: 1, validate: validateUnsafeCheckedRejectedRow},
		{claim: "rejected_upper_bound", minRows: 1, validate: validateUnsafeCheckedRejectedRow},
		{claim: "rejected_access_width_overflow", minRows: 4, validate: validateUnsafeCheckedRejectedRow, requireCheckRow: true},
		{claim: "checked_external_unknown", minRows: 1, validate: validateUnsafeUnknownConservativeRow},
	}
	for _, req := range requireRows {
		valid := 0
		for _, row := range report.Rows {
			if row.Claim != req.claim {
				continue
			}
			rowIssues := req.validate(row)
			if req.requireCheckRow && row.SourceFactID != "" && !checksByParent[row.SourceFactID] {
				rowIssues = append(rowIssues, fmt.Sprintf("source_fact_id %q is missing parented raw_bounds_runtime_check_normal_build normal_build_check", row.SourceFactID))
			}
			if len(rowIssues) > 0 {
				issues = append(issues, fmt.Sprintf("raw bounds claim %s row is not correlated: %s", req.claim, strings.Join(rowIssues, ", ")))
				continue
			}
			valid++
		}
		if valid < req.minRows {
			issues = append(issues, fmt.Sprintf("raw bounds claim %s has %d correlated row(s), want at least %d", req.claim, valid, req.minRows))
		}
	}
	for _, row := range report.Rows {
		if row.ProvenanceClass == "unsafe_verified_root" || row.UnsafeClass == "unsafe_verified_root" {
			switch row.Claim {
			case "allocation_base_metadata", "unsafe_verified_root_allocation_base":
			default:
				issues = append(issues, fmt.Sprintf("unsafe_verified_root row %q cannot claim %q beyond bounded allocation metadata", row.SourceFactID, row.Claim))
			}
		}
	}
	if len(issues) > 0 {
		return fmt.Errorf(strings.Join(issues, "; "))
	}
	return nil
}

func validateRawSliceGatewayCorrelation(cases []memoryprod.CaseReport, memoryRaw []byte) error {
	report, err := parseMemoryReportEvidence(memoryRaw)
	if err != nil {
		return err
	}
	var issues []string
	issues = append(issues, validateRawSliceRuntimeCases(cases)...)
	issues = append(issues, validateCapMemAuthorizationDiscipline(report.Rows)...)
	checksByParent := parentedRawBoundsRuntimeChecks(report.Rows)

	requireRows := []struct {
		claim           string
		minRows         int
		validate        func(memoryReportEvidenceRow) []string
		requireCheckRow bool
	}{
		{claim: "external_unknown", minRows: 1, validate: validateUnsafeUnknownConservativeRow},
		{claim: "raw_slice_verified_allocation_root", minRows: 1, validate: validateUnsafeCheckedDynamicRow},
		{claim: "rejected_negative_length", minRows: 1, validate: validateUnsafeCheckedRejectedRow},
		{claim: "rejected_length_overflow", minRows: 1, validate: validateUnsafeCheckedRejectedRow, requireCheckRow: true},
	}
	for _, req := range requireRows {
		valid := 0
		for _, row := range report.Rows {
			if row.Claim != req.claim {
				continue
			}
			rowIssues := req.validate(row)
			if req.requireCheckRow && row.SourceFactID != "" && !checksByParent[row.SourceFactID] {
				rowIssues = append(rowIssues, fmt.Sprintf("source_fact_id %q is missing parented raw_bounds_runtime_check_normal_build normal_build_check", row.SourceFactID))
			}
			if len(rowIssues) > 0 {
				issues = append(issues, fmt.Sprintf("raw slice claim %s row is not correlated: %s", req.claim, strings.Join(rowIssues, ", ")))
				continue
			}
			valid++
		}
		if valid < req.minRows {
			issues = append(issues, fmt.Sprintf("raw slice claim %s has %d correlated row(s), want at least %d", req.claim, valid, req.minRows))
		}
	}
	if len(issues) > 0 {
		return fmt.Errorf(strings.Join(issues, "; "))
	}
	return nil
}

func validateRawRuntimeCases(cases []memoryprod.CaseReport) []string {
	byName := map[string]memoryprod.CaseReport{}
	for _, c := range cases {
		byName[c.Name] = c
	}
	var issues []string
	for _, req := range []struct {
		name          string
		expectedError string
	}{
		{name: "raw ptr_add negative offset bounds", expectedError: "negative ptr_add offset"},
		{name: "raw ptr_add allocation upper bound", expectedError: "allocation upper bound"},
		{name: "raw allocation-base i32 access width", expectedError: "i32 access width exceeds allocation"},
		{name: "raw allocation-base ptr access width", expectedError: "ptr access width exceeds allocation"},
		{name: "raw allocation-base store_i32 access width", expectedError: "i32 access width exceeds allocation"},
		{name: "raw allocation-base load_ptr access width", expectedError: "ptr access width exceeds allocation"},
	} {
		c, ok := byName[req.name]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing runtime raw bounds case %q", req.name))
			continue
		}
		if !c.Ran || !c.Pass {
			issues = append(issues, fmt.Sprintf("runtime raw bounds case %q did not pass", req.name))
		}
		if req.expectedError != "" && !strings.Contains(c.ExpectedError, req.expectedError) {
			issues = append(issues, fmt.Sprintf("runtime raw bounds case %q expected_error = %q, want %q", req.name, c.ExpectedError, req.expectedError))
		}
	}
	return issues
}

func validateRawSliceRuntimeCases(cases []memoryprod.CaseReport) []string {
	byName := map[string]memoryprod.CaseReport{}
	for _, c := range cases {
		byName[c.Name] = c
	}
	var issues []string
	for _, req := range []struct {
		name          string
		expectedError string
	}{
		{name: "raw slice negative length", expectedError: "negative raw slice length"},
		{name: "raw slice i32 length byte overflow", expectedError: "raw slice length byte overflow"},
	} {
		c, ok := byName[req.name]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing runtime raw slice case %q", req.name))
			continue
		}
		if !c.Ran || !c.Pass {
			issues = append(issues, fmt.Sprintf("runtime raw slice case %q did not pass", req.name))
		}
		if req.expectedError != "" && !strings.Contains(c.ExpectedError, req.expectedError) {
			issues = append(issues, fmt.Sprintf("runtime raw slice case %q expected_error = %q, want %q", req.name, c.ExpectedError, req.expectedError))
		}
	}
	return issues
}

func parentedRawBoundsRuntimeChecks(rows []memoryReportEvidenceRow) map[string]bool {
	out := map[string]bool{}
	for _, row := range rows {
		if row.Claim != "raw_bounds_runtime_check_normal_build" {
			continue
		}
		if row.SourceFactID == "" || row.ParentFactID == "" || !row.NormalBuildCheck {
			continue
		}
		if row.CostClass != "dynamic_check_required" || row.ValidatorName != "raw_bounds_width_validator" || row.ClaimLevel != "validated" {
			continue
		}
		out[row.ParentFactID] = true
	}
	return out
}

func validateUnsafeVerifiedRootAllocationBaseRow(row memoryReportEvidenceRow) []string {
	issues := validateCompilerOwnedRow(row)
	if row.ProvenanceClass != "unsafe_verified_root" || row.UnsafeClass != "unsafe_verified_root" {
		issues = append(issues, "must stay unsafe_verified_root")
	}
	if row.ClaimLevel != "validated" {
		issues = append(issues, "must be validated bounded allocation metadata")
	}
	return issues
}

func validateUnsafeCheckedDynamicRow(row memoryReportEvidenceRow) []string {
	issues := validateCompilerOwnedRow(row)
	if row.ProvenanceClass != "unsafe_checked" || row.UnsafeClass != "unsafe_checked" {
		issues = append(issues, "must stay unsafe_checked")
	}
	if row.CostClass != "dynamic_check_required" {
		issues = append(issues, "must preserve dynamic_check_required")
	}
	if !row.NormalBuildCheck {
		issues = append(issues, "must preserve normal_build_check")
	}
	return issues
}

func validateUnsafeCheckedRejectedRow(row memoryReportEvidenceRow) []string {
	issues := validateCompilerOwnedRow(row)
	if row.ProvenanceClass != "unsafe_checked" || row.UnsafeClass != "unsafe_checked" {
		issues = append(issues, "must stay unsafe_checked")
	}
	if row.CostClass != "unsupported_rejected" {
		issues = append(issues, "must preserve unsupported_rejected")
	}
	return issues
}

func validateUnsafeUnknownConservativeRow(row memoryReportEvidenceRow) []string {
	issues := validateCompilerOwnedRow(row)
	if row.ProvenanceClass != "unsafe_unknown" || row.UnsafeClass != "unsafe_unknown" {
		issues = append(issues, "must stay unsafe_unknown")
	}
	if row.ClaimLevel != "conservative" {
		issues = append(issues, "must remain conservative")
	}
	if row.CostClass != "conservative_fallback" {
		issues = append(issues, "must preserve conservative_fallback")
	}
	return issues
}

func validateCapMemAuthorizationDiscipline(rows []memoryReportEvidenceRow) []string {
	var issues []string
	authRows := 0
	proofClaims := map[string]bool{
		"provenance_known":                   true,
		"no_alias":                           true,
		"index_in_range":                     true,
		"bounds_proof_id":                    true,
		"bounds_check_eliminated":            true,
		"bounds_check_removed_with_proof_id": true,
	}
	for _, row := range rows {
		if row.Claim == "cap_mem_authorization_only" {
			authRows++
			rowIssues := validateCompilerOwnedRow(row)
			if row.ProvenanceClass != "unsafe_checked" || row.UnsafeClass != "unsafe_checked" {
				rowIssues = append(rowIssues, "must stay unsafe_checked authorization evidence")
			}
			if row.ClaimLevel != "evidence_only" {
				rowIssues = append(rowIssues, "must remain evidence_only")
			}
			if row.CostClass != "instrumentation_only" {
				rowIssues = append(rowIssues, "must remain instrumentation_only")
			}
			if row.ValidatorName != "" || row.ValidatorStatus != "not_run" {
				rowIssues = append(rowIssues, "must not be treated as a validated proof")
			}
			if row.NormalBuildCheck {
				rowIssues = append(rowIssues, "must not masquerade as a bounds check")
			}
			if len(rowIssues) > 0 {
				issues = append(issues, fmt.Sprintf("cap.mem authorization row is not evidence-only: %s", strings.Join(rowIssues, ", ")))
			}
		}
		if strings.Contains(strings.ToLower(row.ValidatorName+" "+row.Reason), "cap_mem") ||
			strings.Contains(strings.ToLower(row.ValidatorName+" "+row.Reason), "cap.mem") {
			if proofClaims[row.Claim] {
				issues = append(issues, fmt.Sprintf("cap.mem authorization cannot validate proof claim %q", row.Claim))
			}
		}
	}
	if authRows == 0 {
		issues = append(issues, "missing cap_mem_authorization_only evidence row")
	}
	return issues
}

func validateCompilerOwnedRow(row memoryReportEvidenceRow) []string {
	var issues []string
	if strings.TrimSpace(row.SourceFactID) == "" {
		issues = append(issues, "missing source_fact_id")
	}
	return issues
}

func ceilDiv(n, d int) int {
	if d <= 0 {
		return 0
	}
	return (n + d - 1) / d
}

func smallHeapBenchmarkSource(allocationCount, bytesPerAllocation int) string {
	var b strings.Builder
	for i := 0; i < allocationCount; i++ {
		fmt.Fprintf(&b, "func make_%02d() -> []u8\n", i)
		b.WriteString("uses alloc, mem:\n")
		fmt.Fprintf(&b, "    var xs: []u8 = make_u8(%d)\n", bytesPerAllocation)
		b.WriteString("    return xs\n\n")
	}
	b.WriteString("func main() -> Int\n")
	b.WriteString("uses alloc, mem:\n")
	for i := 0; i < allocationCount; i++ {
		fmt.Fprintf(&b, "    let xs_%02d: []u8 = make_%02d()\n", i, i)
	}
	b.WriteString("    return ")
	for i := 0; i < allocationCount; i++ {
		if i > 0 {
			b.WriteString(" + ")
		}
		fmt.Fprintf(&b, "xs_%02d.len", i)
	}
	b.WriteString("\n")
	return b.String()
}
