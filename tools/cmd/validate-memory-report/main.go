package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"tetra_language/compiler/memoryvocab"
)

const reportSchemaV1 = "tetra.memory-report.v1"

type memoryReport struct {
	SchemaVersion string            `json:"schema_version"`
	Rows          []memoryReportRow `json:"rows"`
}

type memoryReportRow struct {
	ProgramID             string `json:"program_id,omitempty"`
	FunctionID            string `json:"function_id,omitempty"`
	ValueID               string `json:"value_id,omitempty"`
	IslandID              string `json:"island_id,omitempty"`
	Epoch                 int    `json:"epoch,omitempty"`
	BaseID                string `json:"base_id,omitempty"`
	SiteID                string `json:"site_id,omitempty"`
	SourceSpan            string `json:"source_span,omitempty"`
	SourceFactID          string `json:"source_fact_id,omitempty"`
	ParentFactID          string `json:"parent_fact_id,omitempty"`
	LoweredArtifactID     string `json:"lowered_artifact_id,omitempty"`
	SourceStage           string `json:"source_stage,omitempty"`
	Claim                 string `json:"claim,omitempty"`
	ClaimLevel            string `json:"claim_level,omitempty"`
	ProvenanceClass       string `json:"provenance_class,omitempty"`
	OwnerID               string `json:"owner_id,omitempty"`
	ParamIndex            *int   `json:"param_index,omitempty"`
	ParamPath             string `json:"param_path,omitempty"`
	BorrowState           string `json:"borrow_state,omitempty"`
	EscapeState           string `json:"escape_state,omitempty"`
	AliasState            string `json:"alias_state,omitempty"`
	UnsafeClass           string `json:"unsafe_class,omitempty"`
	AllocationSiteID      string `json:"allocation_site_id,omitempty"`
	PlannedStorage        string `json:"planned_storage,omitempty"`
	ActualLoweringStorage string `json:"actual_lowering_storage,omitempty"`
	ProofID               string `json:"proof_id,omitempty"`
	ProofKind             string `json:"proof_kind,omitempty"`
	ProofSubjectBaseID    string `json:"proof_subject_base_id,omitempty"`
	ProofIndexValueID     string `json:"proof_index_value_id,omitempty"`
	ProofOperation        string `json:"proof_operation,omitempty"`
	ProofRange            string `json:"proof_range,omitempty"`
	ValidatorName         string `json:"validator_name,omitempty"`
	ValidatorStatus       string `json:"validator_status,omitempty"`
	CostClass             string `json:"cost_class,omitempty"`
	NormalBuildCheck      bool   `json:"normal_build_check,omitempty"`
	Reason                string `json:"reason,omitempty"`
}

type allocationPlanReport struct {
	SchemaVersion int                        `json:"schema_version"`
	Kind          string                     `json:"kind"`
	Functions     []allocationReportFunction `json:"functions,omitempty"`
}

type allocationReportFunction struct {
	Name        string                       `json:"name"`
	Allocations []allocationReportAllocation `json:"allocations,omitempty"`
}

type allocationReportAllocation struct {
	ID                    string                  `json:"id"`
	SiteID                string                  `json:"site_id"`
	Builtin               string                  `json:"builtin,omitempty"`
	LengthStatus          string                  `json:"length_status,omitempty"`
	ZeroGuardStatus       string                  `json:"zero_guard_status,omitempty"`
	NegativeGuardStatus   string                  `json:"negative_guard_status,omitempty"`
	OverflowGuardStatus   string                  `json:"overflow_guard_status,omitempty"`
	PlannedStorage        string                  `json:"planned_storage"`
	ActualLoweringStorage string                  `json:"actual_lowering_storage"`
	ValidationStatus      string                  `json:"validation_status,omitempty"`
	LoweringStatus        string                  `json:"lowering_status,omitempty"`
	ReasonCodes           []string                `json:"reason_codes,omitempty"`
	HeapReasonCodes       []string                `json:"heap_reason_codes,omitempty"`
	Domain                *allocationMemoryDomain `json:"domain,omitempty"`
	Reason                string                  `json:"reason,omitempty"`
}

type allocationMemoryDomain struct {
	DomainID       string `json:"domain_id"`
	ParentDomainID string `json:"parent_domain_id,omitempty"`
	Kind           string `json:"kind"`
	OwnerKind      string `json:"owner_kind"`
	OwnerID        string `json:"owner_id"`
	Lifetime       string `json:"lifetime"`
	BudgetBytes    int64  `json:"budget_bytes,omitempty"`
	RequestedBytes int64  `json:"requested_bytes,omitempty"`
	ReservedBytes  int64  `json:"reserved_bytes,omitempty"`
	CommittedBytes int64  `json:"committed_bytes,omitempty"`
	ReleasedBytes  int64  `json:"released_bytes,omitempty"`
	CurrentBytes   int64  `json:"current_bytes,omitempty"`
	PeakBytes      int64  `json:"peak_bytes,omitempty"`
	CopyCount      int    `json:"copy_count,omitempty"`
	BytesCopied    int64  `json:"bytes_copied,omitempty"`
}

func main() {
	reportPath := flag.String("report", "", "path to tetra.memory-report.v1 JSON report")
	allocReportPath := flag.String("alloc-report", "", "optional path to paired allocation report for lowered_artifact_id validation")
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateMemoryReportWithAllocReport(*reportPath, *allocReportPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateMemoryReport(path string) error {
	return validateMemoryReportWithAllocReport(path, "")
}

func validateMemoryReportWithAllocReport(path string, allocReportPath string) error {
	report, err := readMemoryReport(path)
	if err != nil {
		return err
	}
	if err := validateReport(report); err != nil {
		return err
	}
	if strings.TrimSpace(allocReportPath) == "" {
		return nil
	}
	allocReport, err := readAllocationReport(allocReportPath)
	if err != nil {
		return err
	}
	return validateLoweredArtifactsAgainstAllocationReport(report, allocReport)
}

func readMemoryReport(path string) (memoryReport, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return memoryReport{}, err
	}
	var report memoryReport
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		return memoryReport{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return memoryReport{}, errors.New("trailing data after memory report JSON")
		}
		return memoryReport{}, fmt.Errorf("trailing data after memory report JSON: %w", err)
	}
	return report, nil
}

func readAllocationReport(path string) (allocationPlanReport, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return allocationPlanReport{}, err
	}
	var report allocationPlanReport
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&report); err != nil {
		return allocationPlanReport{}, fmt.Errorf("parse allocation report: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return allocationPlanReport{}, errors.New("trailing data after allocation report JSON")
		}
		return allocationPlanReport{}, fmt.Errorf("trailing data after allocation report JSON: %w", err)
	}
	return report, nil
}

type allocationArtifact struct {
	functionName string
	allocation   allocationReportAllocation
}

func validateLoweredArtifactsAgainstAllocationReport(report memoryReport, allocReport allocationPlanReport) error {
	var issues []string
	if allocReport.SchemaVersion != 2 || allocReport.Kind != "allocation_plan" {
		issues = append(issues, fmt.Sprintf("allocation report schema_version/kind = %d/%q, want 2/allocation_plan", allocReport.SchemaVersion, allocReport.Kind))
	}
	artifacts := map[string]allocationArtifact{}
	for _, fn := range allocReport.Functions {
		for _, alloc := range fn.Allocations {
			if strings.TrimSpace(fn.Name) == "" || strings.TrimSpace(alloc.ID) == "" || strings.TrimSpace(alloc.ActualLoweringStorage) == "" {
				continue
			}
			issues = append(issues, validateAllocationArtifactContract(fmt.Sprintf("allocation report %s/%s", fn.Name, alloc.ID), alloc)...)
			id := fmt.Sprintf("ir:%s:%s:%s", fn.Name, alloc.ID, alloc.ActualLoweringStorage)
			artifacts[id] = allocationArtifact{functionName: fn.Name, allocation: alloc}
		}
	}
	for i, row := range report.Rows {
		if !rowRequiresArtifact(row) || strings.TrimSpace(row.LoweredArtifactID) == "" {
			continue
		}
		artifact, ok := artifacts[row.LoweredArtifactID]
		if !ok {
			issues = append(issues, fmt.Sprintf("row %d: lowered_artifact_id %q is not present in allocation report", i, row.LoweredArtifactID))
			continue
		}
		issues = append(issues, validateRowAgainstAllocationArtifact(i, row, artifact)...)
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateAllocationArtifactContract(prefix string, alloc allocationReportAllocation) []string {
	var issues []string
	builtin := strings.TrimSpace(alloc.Builtin)
	lengthStatus := strings.TrimSpace(alloc.LengthStatus)
	zeroGuard := strings.TrimSpace(alloc.ZeroGuardStatus)
	negativeGuard := strings.TrimSpace(alloc.NegativeGuardStatus)
	overflowGuard := strings.TrimSpace(alloc.OverflowGuardStatus)
	if builtin == "" {
		issues = append(issues, fmt.Sprintf("%s: builtin is required for allocation length contract evidence", prefix))
	}
	if lengthStatus == "" {
		issues = append(issues, fmt.Sprintf("%s: length_status is required for allocation length contract evidence", prefix))
	} else if !knownAllocationLengthStatus(lengthStatus) {
		issues = append(issues, fmt.Sprintf("%s: unknown length_status %q", prefix, lengthStatus))
	}
	if zeroGuard == "" {
		issues = append(issues, fmt.Sprintf("%s: zero_guard_status is required for allocation length contract evidence", prefix))
	} else if !knownAllocationZeroGuard(zeroGuard) {
		issues = append(issues, fmt.Sprintf("%s: unknown zero_guard_status %q", prefix, zeroGuard))
	}
	if negativeGuard == "" {
		issues = append(issues, fmt.Sprintf("%s: negative_guard_status is required for allocation length contract evidence", prefix))
	} else if !knownAllocationRejectGuard(negativeGuard) {
		issues = append(issues, fmt.Sprintf("%s: unknown negative_guard_status %q", prefix, negativeGuard))
	}
	if overflowGuard == "" {
		issues = append(issues, fmt.Sprintf("%s: overflow_guard_status is required for allocation length contract evidence", prefix))
	} else if !knownAllocationRejectGuard(overflowGuard) {
		issues = append(issues, fmt.Sprintf("%s: unknown overflow_guard_status %q", prefix, overflowGuard))
	}
	if allocationArtifactUsesHeap(alloc) {
		if len(alloc.HeapReasonCodes) == 0 {
			issues = append(issues, fmt.Sprintf("%s: heap allocation requires heap_reason_codes", prefix))
		}
		issues = append(issues, validateAllocationReasonCodes(prefix, alloc.ReasonCodes, alloc.HeapReasonCodes)...)
	} else if len(alloc.HeapReasonCodes) > 0 {
		issues = append(issues, fmt.Sprintf("%s: heap_reason_codes require heap storage", prefix))
	}
	if builtin == "" || zeroGuard == "" || negativeGuard == "" || overflowGuard == "" {
		return issues
	}
	wantZero, wantNegative, wantOverflow, ok := allocationGuardContractForBuiltin(builtin)
	if !ok {
		return issues
	}
	if zeroGuard != wantZero {
		issues = append(issues, fmt.Sprintf("%s: builtin %q zero_guard_status %q does not match contract %q", prefix, builtin, zeroGuard, wantZero))
	}
	if negativeGuard != wantNegative {
		issues = append(issues, fmt.Sprintf("%s: builtin %q negative_guard_status %q does not match contract %q", prefix, builtin, negativeGuard, wantNegative))
	}
	if overflowGuard != wantOverflow {
		issues = append(issues, fmt.Sprintf("%s: builtin %q overflow_guard_status %q does not match contract %q", prefix, builtin, overflowGuard, wantOverflow))
	}
	issues = append(issues, validateAllocationMemoryDomain(prefix, alloc.Domain)...)
	return issues
}

func validateAllocationMemoryDomain(prefix string, domain *allocationMemoryDomain) []string {
	if domain == nil {
		return nil
	}
	var issues []string
	if strings.TrimSpace(domain.DomainID) == "" {
		issues = append(issues, prefix+": domain_id is required")
	}
	if !knownAllocationDomainKind(domain.Kind) {
		issues = append(issues, fmt.Sprintf("%s: unknown domain kind %q", prefix, domain.Kind))
	}
	if strings.TrimSpace(domain.OwnerKind) == "" {
		issues = append(issues, prefix+": domain owner_kind is required")
	}
	if strings.TrimSpace(domain.OwnerID) == "" {
		issues = append(issues, prefix+": domain owner_id is required")
	}
	if strings.TrimSpace(domain.Lifetime) == "" {
		issues = append(issues, prefix+": domain lifetime is required")
	}
	for name, value := range map[string]int64{
		"budget_bytes":    domain.BudgetBytes,
		"requested_bytes": domain.RequestedBytes,
		"reserved_bytes":  domain.ReservedBytes,
		"committed_bytes": domain.CommittedBytes,
		"released_bytes":  domain.ReleasedBytes,
		"current_bytes":   domain.CurrentBytes,
		"peak_bytes":      domain.PeakBytes,
		"bytes_copied":    domain.BytesCopied,
	} {
		if value < 0 {
			issues = append(issues, fmt.Sprintf("%s: domain %s must not be negative", prefix, name))
		}
	}
	if domain.CopyCount < 0 {
		issues = append(issues, prefix+": domain copy_count must not be negative")
	}
	if domain.PeakBytes < domain.CurrentBytes {
		issues = append(issues, prefix+": domain peak_bytes must be >= current_bytes")
	}
	if domain.BytesCopied > 0 && domain.CopyCount == 0 {
		issues = append(issues, prefix+": domain bytes_copied requires copy_count")
	}
	return issues
}

func allocationArtifactUsesHeap(alloc allocationReportAllocation) bool {
	return alloc.PlannedStorage == "Heap" || alloc.ActualLoweringStorage == "Heap"
}

func validateAllocationReasonCodes(prefix string, reasonCodes []string, heapReasonCodes []string) []string {
	var issues []string
	seen := map[string]bool{}
	for _, code := range reasonCodes {
		if strings.TrimSpace(code) == "" {
			issues = append(issues, fmt.Sprintf("%s: reason_codes contains an empty entry", prefix))
			continue
		}
		if strings.TrimSpace(code) != code {
			issues = append(issues, fmt.Sprintf("%s: reason_codes contains untrimmed entry %q", prefix, code))
		}
		if seen[code] {
			issues = append(issues, fmt.Sprintf("%s: reason_codes contains duplicate entry %q", prefix, code))
		}
		seen[code] = true
	}
	heapSeen := map[string]bool{}
	for _, code := range heapReasonCodes {
		if strings.TrimSpace(code) == "" {
			issues = append(issues, fmt.Sprintf("%s: heap_reason_codes contains an empty entry", prefix))
			continue
		}
		if strings.TrimSpace(code) != code {
			issues = append(issues, fmt.Sprintf("%s: heap_reason_codes contains untrimmed entry %q", prefix, code))
		}
		if heapSeen[code] {
			issues = append(issues, fmt.Sprintf("%s: heap_reason_codes contains duplicate entry %q", prefix, code))
		}
		heapSeen[code] = true
		if !knownHeapReasonCode(code) {
			issues = append(issues, fmt.Sprintf("%s: unknown heap_reason_code %q", prefix, code))
		}
		if !seen[code] {
			issues = append(issues, fmt.Sprintf("%s: heap_reason_code %q missing from reason_codes", prefix, code))
		}
	}
	return issues
}

func knownHeapReasonCode(code string) bool {
	switch code {
	case "heap.required_escape_return",
		"heap.required_unknown_call",
		"heap.required_actor_boundary",
		"heap.required_task_boundary",
		"heap.required_dynamic_lifetime",
		"heap.required_large_object",
		"heap.required_ffi_external",
		"heap.required_backend_lowering_unavailable",
		"heap.required_region_lowering_unavailable":
		return true
	default:
		return false
	}
}

func knownAllocationDomainKind(kind string) bool {
	switch kind {
	case "process", "task", "actor", "island", "request", "external":
		return true
	default:
		return false
	}
}

func knownAllocationLengthStatus(status string) bool {
	switch status {
	case "runtime_guarded",
		"valid_empty_allocation",
		"normal_allocation",
		"rejected_negative_length",
		"rejected_byte_size_overflow",
		"invalid_length_contract":
		return true
	default:
		return false
	}
}

func knownAllocationZeroGuard(status string) bool {
	switch status {
	case "invalid_precondition",
		"valid_empty_no_allocator",
		"valid_empty_no_metadata_access":
		return true
	default:
		return false
	}
}

func knownAllocationRejectGuard(status string) bool {
	switch status {
	case "reject_before_allocation", "reject_before_metadata_access":
		return true
	default:
		return false
	}
}

func allocationGuardContractForBuiltin(builtin string) (zero string, negative string, overflow string, ok bool) {
	switch {
	case builtin == "core.alloc_bytes":
		return "invalid_precondition", "reject_before_allocation", "reject_before_allocation", true
	case strings.HasPrefix(builtin, "core.island_make_"):
		return "valid_empty_no_metadata_access", "reject_before_metadata_access", "reject_before_metadata_access", true
	case strings.HasPrefix(builtin, "core.make_"),
		strings.HasPrefix(builtin, "core.slice_copy_"),
		builtin == "core.string_copy":
		return "valid_empty_no_allocator", "reject_before_allocation", "reject_before_allocation", true
	default:
		return "", "", "", false
	}
}

func validateRowAgainstAllocationArtifact(index int, row memoryReportRow, artifact allocationArtifact) []string {
	prefix := fmt.Sprintf("row %d", index)
	var issues []string
	alloc := artifact.allocation
	if row.FunctionID != "" && row.FunctionID != artifact.functionName {
		issues = append(issues, fmt.Sprintf("%s: function_id %q does not match allocation report function %q for lowered_artifact_id %q", prefix, row.FunctionID, artifact.functionName, row.LoweredArtifactID))
	}
	if row.AllocationSiteID != "" && row.AllocationSiteID != alloc.ID {
		issues = append(issues, fmt.Sprintf("%s: allocation_site_id %q does not match allocation report id %q for lowered_artifact_id %q", prefix, row.AllocationSiteID, alloc.ID, row.LoweredArtifactID))
	}
	if row.SiteID != "" && alloc.SiteID != "" && row.SiteID != alloc.SiteID {
		issues = append(issues, fmt.Sprintf("%s: site_id %q does not match allocation report site_id %q for lowered_artifact_id %q", prefix, row.SiteID, alloc.SiteID, row.LoweredArtifactID))
	}
	if row.PlannedStorage != "" && row.PlannedStorage != alloc.PlannedStorage {
		issues = append(issues, fmt.Sprintf("%s: planned_storage %q does not match allocation report planned_storage %q for lowered_artifact_id %q", prefix, row.PlannedStorage, alloc.PlannedStorage, row.LoweredArtifactID))
	}
	if row.ActualLoweringStorage != "" && row.ActualLoweringStorage != alloc.ActualLoweringStorage {
		issues = append(issues, fmt.Sprintf("%s: actual_lowering_storage %q does not match allocation report actual_lowering_storage %q for lowered_artifact_id %q", prefix, row.ActualLoweringStorage, alloc.ActualLoweringStorage, row.LoweredArtifactID))
	}
	return issues
}

func validateReport(report memoryReport) error {
	var issues []string
	if report.SchemaVersion != reportSchemaV1 {
		issues = append(issues, fmt.Sprintf("schema_version is %q, want %q", report.SchemaVersion, reportSchemaV1))
	}
	if len(report.Rows) == 0 {
		issues = append(issues, "rows are required")
	}
	for i, row := range report.Rows {
		issues = append(issues, validateRow(i, row)...)
	}
	if index, previous, current, ok := firstNonDeterministicReportRow(report.Rows); ok {
		issues = append(issues, fmt.Sprintf("row %d: non-deterministic memory report row order: source_fact_id %q sorts before previous source_fact_id %q", index, current.SourceFactID, previous.SourceFactID))
	}
	seenFactIDs := map[string]int{}
	for i, row := range report.Rows {
		sourceFactID := strings.TrimSpace(row.SourceFactID)
		if sourceFactID == "" {
			continue
		}
		if previous, ok := seenFactIDs[sourceFactID]; ok {
			issues = append(issues, fmt.Sprintf("row %d: duplicate source_fact_id %q also used by row %d", i, sourceFactID, previous))
			continue
		}
		seenFactIDs[sourceFactID] = i
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func firstNonDeterministicReportRow(rows []memoryReportRow) (int, memoryReportRow, memoryReportRow, bool) {
	for i := 1; i < len(rows); i++ {
		if compareMemoryReportRows(rows[i-1], rows[i]) > 0 {
			return i, rows[i-1], rows[i], true
		}
	}
	return 0, memoryReportRow{}, memoryReportRow{}, false
}

func compareMemoryReportRows(a memoryReportRow, b memoryReportRow) int {
	for _, pair := range []struct {
		a string
		b string
	}{
		{a.SourceFactID, b.SourceFactID},
		{a.ProgramID, b.ProgramID},
		{a.FunctionID, b.FunctionID},
		{a.SiteID, b.SiteID},
		{a.ValueID, b.ValueID},
		{a.Claim, b.Claim},
		{a.SourceStage, b.SourceStage},
		{a.ParentFactID, b.ParentFactID},
		{a.LoweredArtifactID, b.LoweredArtifactID},
	} {
		if pair.a < pair.b {
			return -1
		}
		if pair.a > pair.b {
			return 1
		}
	}
	return 0
}

func validateRow(index int, row memoryReportRow) []string {
	prefix := fmt.Sprintf("row %d", index)
	var issues []string
	if strings.TrimSpace(row.SourceFactID) == "" {
		issues = append(issues, prefix+": source_fact_id is required")
	}
	if strings.TrimSpace(row.SiteID) == "" {
		issues = append(issues, prefix+": site_id is required")
	}
	if strings.TrimSpace(row.Claim) == "" {
		issues = append(issues, prefix+": claim is required")
	} else if !knownReportClaim(row.Claim) {
		issues = append(issues, fmt.Sprintf("%s: unknown memory report claim %q", prefix, row.Claim))
	}
	if !knownSourceStage(row.SourceStage) {
		issues = append(issues, fmt.Sprintf("%s: unknown source_stage %q", prefix, row.SourceStage))
	}
	if !knownProvenance(row.ProvenanceClass) {
		issues = append(issues, fmt.Sprintf("%s: unknown provenance_class %q", prefix, row.ProvenanceClass))
	}
	if !knownUnsafe(row.UnsafeClass) {
		issues = append(issues, fmt.Sprintf("%s: unknown unsafe_class %q", prefix, row.UnsafeClass))
	}
	if !knownClaimLevel(row.ClaimLevel) {
		issues = append(issues, fmt.Sprintf("%s: unknown claim_level %q", prefix, row.ClaimLevel))
	}
	if !knownValidatorStatus(row.ValidatorStatus) {
		issues = append(issues, fmt.Sprintf("%s: unknown validator_status %q", prefix, row.ValidatorStatus))
	}
	if !knownAliasState(row.AliasState) {
		issues = append(issues, fmt.Sprintf("%s: unknown alias_state %q", prefix, row.AliasState))
	}
	if !knownCostClass(row.CostClass) {
		issues = append(issues, fmt.Sprintf("%s: unknown cost_class %q", prefix, row.CostClass))
	}
	if islandBackedReportRow(row) && row.Epoch <= 0 {
		issues = append(issues, prefix+": island-backed row requires positive epoch")
	}
	if row.Epoch > 0 && strings.TrimSpace(row.IslandID) == "" {
		issues = append(issues, prefix+": epoch requires island_id")
	}
	if strings.TrimSpace(row.IslandID) != "" && strings.TrimSpace(row.BaseID) == "" {
		issues = append(issues, prefix+": island_id requires base_id")
	}
	if dynamicRawRuntimeCheckCostDisallowed(row.Claim, row.CostClass) {
		issues = append(issues, fmt.Sprintf("%s: dynamic raw check claim %q must use dynamic_check_required, got %s", prefix, row.Claim, row.CostClass))
	}
	if memoryvocab.ZeroCostProvenClaimDisallowed(row.Claim, row.CostClass, row.ClaimLevel, row.PlannedStorage, row.ActualLoweringStorage) {
		issues = append(issues, fmt.Sprintf("%s: zero_cost_proven requires validated compiler-owned proof for claim %q", prefix, row.Claim))
	}
	if row.CostClass == "dynamic_check_required" && !row.NormalBuildCheck {
		issues = append(issues, prefix+": dynamic_check_required requires normal_build_check")
	}
	if row.ClaimLevel == "validated" && row.ValidatorStatus != "pass" {
		issues = append(issues, prefix+": validated claim requires validator_status pass")
	}
	if row.ClaimLevel == "validated" && strings.TrimSpace(row.ValidatorName) == "" {
		issues = append(issues, prefix+": validated claim requires validator_name")
	}
	if row.ClaimLevel == "validated" && boundsTypedProofClaim(row.Claim) {
		if missing := missingTypedProofFields(row.ProofID, row.ProofKind, row.ProofSubjectBaseID, row.ProofIndexValueID, row.ProofOperation, row.ProofRange); len(missing) > 0 {
			issues = append(issues, fmt.Sprintf("%s: validated bounds proof claim %q requires typed proof fields: %s", prefix, row.Claim, strings.Join(missing, ", ")))
		}
	}
	if row.ClaimLevel == "validated" && memoryvocab.IslandKernelClaimValidatorMismatch(row.Claim, row.ValidatorName) {
		issues = append(issues, fmt.Sprintf("%s: validated island claim %q requires validator_name %q", prefix, row.Claim, memoryvocab.RequiredIslandKernelClaimValidator(row.Claim)))
	}
	if safeProvenance(row.ProvenanceClass) && row.UnsafeClass == "unsafe_unknown" {
		issues = append(issues, fmt.Sprintf("%s: %s claim cannot be sourced from unsafe_unknown", prefix, row.ProvenanceClass))
	}
	if unsafeUnknownRow(row) && unsafeUnknownOptimizationClaim(row.Claim, row.AliasState) {
		issues = append(issues, fmt.Sprintf("%s: unsafe_unknown cannot authorize optimization claim %q", prefix, row.Claim))
	}
	if broadNoAliasClaim(row.Claim) {
		issues = append(issues, fmt.Sprintf("%s: broad noalias claim %q is outside Memory Ideal v0", prefix, row.Claim))
	}
	if row.ClaimLevel == "validated" && conservativeNoAliasBoundaryClaim(row.Claim) {
		issues = append(issues, fmt.Sprintf("%s: conservative noalias boundary claim %q cannot be validated", prefix, row.Claim))
	}
	if claimRequiresParentFactID(row.Claim) && strings.TrimSpace(row.ParentFactID) == "" {
		issues = append(issues, fmt.Sprintf("%s: derived claim %q requires parent_fact_id", prefix, row.Claim))
	}
	if row.Claim == "copy_owned" && row.ProvenanceClass != "safe_owned" {
		issues = append(issues, prefix+": copy_owned requires safe_owned provenance")
	}
	if unsafeUnknownRow(row) && row.CostClass == "zero_cost_proven" {
		issues = append(issues, fmt.Sprintf("%s: unsafe_unknown cannot claim %s", prefix, row.CostClass))
	}
	if row.CostClass == "dynamic_check_required" && memoryOptimizationClaim(row.Claim, row.AliasState) && !row.NormalBuildCheck {
		issues = append(issues, fmt.Sprintf("%s: dynamic_check_required optimization claim %q requires normal_build_check", prefix, row.Claim))
	}
	if bareBoundsCheckEliminatedClaim(row.Claim) {
		issues = append(issues, fmt.Sprintf("%s: bounds_check_eliminated requires compiler-owned proof id; use bounds_check_removed_with_proof_id", prefix))
	}
	if unsafeVerifiedRootDisallowedClaim(row.ProvenanceClass, row.UnsafeClass, row.Claim) {
		issues = append(issues, fmt.Sprintf("%s: unsafe_verified_root cannot claim %q without bounded raw metadata", prefix, row.Claim))
	}
	if unsafeCheckedDisallowedClaim(row.ProvenanceClass, row.UnsafeClass, row.Claim) {
		issues = append(issues, fmt.Sprintf("%s: unsafe_checked cannot claim %q outside checked runtime/bounds evidence", prefix, row.Claim))
	}
	if capMemDisallowedProofClaim(row.Claim, row.ValidatorName, row.Reason) {
		issues = append(issues, fmt.Sprintf("%s: cap.mem authorization cannot claim %q; cap.mem authorizes raw operations only and does not prove pointer validity, bounds, ownership, noalias, or safe provenance", prefix, row.Claim))
	}
	if row.ClaimLevel == "validated" && unsafeExternalRootTrustedStorage(row.ProvenanceClass, row.UnsafeClass, row.PlannedStorage, row.ActualLoweringStorage) {
		issues = append(issues, fmt.Sprintf("%s: unsafe/external root %s/%s cannot validate trusted storage lowering %q/%q", prefix, row.ProvenanceClass, row.UnsafeClass, row.PlannedStorage, row.ActualLoweringStorage))
	}
	if row.ClaimLevel == "validated" && strings.Contains(row.Claim, "no_alias") && !validatedNoAliasState(row.AliasState) {
		issues = append(issues, fmt.Sprintf("%s: validated no_alias requires unique or mutable_exclusive alias_state, got %q", prefix, row.AliasState))
	}
	if row.ParamIndex != nil && *row.ParamIndex < 0 {
		issues = append(issues, fmt.Sprintf("%s: negative param_index %d", prefix, *row.ParamIndex))
	}
	issues = append(issues, validateRowStorage(index, row)...)
	if rowRequiresArtifact(row) && row.LoweredArtifactID == "" {
		issues = append(issues, prefix+": storage/lowering claim requires lowered_artifact_id")
	}
	if row.ClaimLevel == "validated" && validatedTrustedStorageHeapFallback(row.PlannedStorage, row.ActualLoweringStorage) {
		issues = append(issues, fmt.Sprintf("%s: validated %s claim cannot lower as Heap", prefix, row.PlannedStorage))
	}
	if row.ClaimLevel == "validated" && runtimeProofRequiredStorage(row.PlannedStorage, row.ActualLoweringStorage) {
		issues = append(issues, fmt.Sprintf("%s: validated runtime boundary storage %q/%q requires production runtime proof", prefix, row.PlannedStorage, row.ActualLoweringStorage))
	}
	return issues
}

func validateRowStorage(index int, row memoryReportRow) []string {
	prefix := fmt.Sprintf("row %d", index)
	var issues []string
	if (row.PlannedStorage == "") != (row.ActualLoweringStorage == "") {
		issues = append(issues, prefix+": planned_storage and actual_lowering_storage must be present together")
	}
	if row.PlannedStorage != "" && !knownStorage(row.PlannedStorage) {
		issues = append(issues, fmt.Sprintf("%s: unknown planned_storage %q", prefix, row.PlannedStorage))
	}
	if row.ActualLoweringStorage != "" && !knownStorage(row.ActualLoweringStorage) {
		issues = append(issues, fmt.Sprintf("%s: unknown actual_lowering_storage %q", prefix, row.ActualLoweringStorage))
	}
	return issues
}

func boundsTypedProofClaim(claim string) bool {
	return claim == "bounds_proof_id" || claim == "bounds_check_removed_with_proof_id"
}

func missingTypedProofFields(proofID string, proofKind string, subjectBaseID string, indexValueID string, operation string, proofRange string) []string {
	var missing []string
	if strings.TrimSpace(proofID) == "" {
		missing = append(missing, "proof_id")
	}
	if strings.TrimSpace(proofKind) == "" {
		missing = append(missing, "proof_kind")
	}
	if strings.TrimSpace(subjectBaseID) == "" {
		missing = append(missing, "proof_subject_base_id")
	}
	if strings.TrimSpace(indexValueID) == "" {
		missing = append(missing, "proof_index_value_id")
	}
	if strings.TrimSpace(operation) == "" {
		missing = append(missing, "proof_operation")
	}
	if strings.TrimSpace(proofRange) == "" {
		missing = append(missing, "proof_range")
	}
	return missing
}

func knownSourceStage(value string) bool {
	return memoryvocab.KnownSourceStage(value)
}

func knownProvenance(value string) bool {
	return memoryvocab.KnownProvenanceClass(value)
}

func safeProvenance(value string) bool {
	return memoryvocab.SafeProvenanceClass(value)
}

func unsafeUnknownRow(row memoryReportRow) bool {
	return memoryvocab.UnsafeUnknownRow(row.ProvenanceClass, row.UnsafeClass)
}

func unsafeUnknownOptimizationClaim(claim string, aliasState string) bool {
	return memoryvocab.UnsafeUnknownOptimizationClaim(claim, aliasState)
}

func memoryOptimizationClaim(claim string, aliasState string) bool {
	return memoryvocab.MemoryOptimizationClaim(claim, aliasState)
}

func bareBoundsCheckEliminatedClaim(claim string) bool {
	return memoryvocab.BareBoundsCheckEliminatedClaim(claim)
}

func dynamicRawRuntimeCheckCostDisallowed(claim string, costClass string) bool {
	return memoryvocab.DynamicRawRuntimeCheckCostDisallowed(claim, costClass)
}

func unsafeCheckedDisallowedClaim(provenanceClass string, unsafeClass string, claim string) bool {
	return memoryvocab.UnsafeCheckedDisallowedClaim(provenanceClass, unsafeClass, claim)
}

func capMemDisallowedProofClaim(claim string, validatorName string, reason string) bool {
	return memoryvocab.CapMemDisallowedProofClaim(claim, validatorName, reason)
}

func broadNoAliasClaim(claim string) bool {
	return memoryvocab.BroadNoAliasClaim(claim)
}

func conservativeNoAliasBoundaryClaim(claim string) bool {
	return memoryvocab.ConservativeNoAliasBoundaryClaim(claim)
}

func claimRequiresParentFactID(claim string) bool {
	return memoryvocab.ClaimRequiresParentFactID(claim)
}

func knownCostClass(value string) bool {
	return memoryvocab.KnownCostClass(value)
}

func knownReportClaim(value string) bool {
	return memoryvocab.KnownReportClaim(value)
}

func unsafeVerifiedRootDisallowedClaim(provenanceClass string, unsafeClass string, claim string) bool {
	return memoryvocab.UnsafeVerifiedRootDisallowedClaim(provenanceClass, unsafeClass, claim)
}

func unsafeUnknownTrustedStorage(planned string, actual string) bool {
	return memoryvocab.UnsafeUnknownTrustedStorage(planned, actual)
}

func unsafeExternalRootTrustedStorage(provenanceClass string, unsafeClass string, planned string, actual string) bool {
	return memoryvocab.UnsafeExternalRootTrustedStorage(provenanceClass, unsafeClass, planned, actual)
}

func validatedTrustedStorageHeapFallback(planned string, actual string) bool {
	return memoryvocab.ValidatedTrustedStorageHeapFallback(planned, actual)
}

func runtimeProofRequiredStorage(planned string, actual string) bool {
	return memoryvocab.RuntimeProofRequiredStorage(planned, actual)
}

func trustedStorageForUnsafeUnknown(value string) bool {
	return memoryvocab.UnsafeUnknownTrustedStorage(value, "")
}

func knownStorage(value string) bool {
	return memoryvocab.KnownStorageClass(value)
}

func knownAliasState(value string) bool {
	return memoryvocab.KnownAliasState(value)
}

func validatedNoAliasState(value string) bool {
	return memoryvocab.ValidatedNoAliasState(value)
}

func knownUnsafe(value string) bool {
	return memoryvocab.KnownUnsafeClass(value)
}

func knownClaimLevel(value string) bool {
	return memoryvocab.KnownClaimLevel(value)
}

func knownValidatorStatus(value string) bool {
	return memoryvocab.KnownValidatorStatus(value)
}

func islandBackedReportRow(row memoryReportRow) bool {
	return strings.TrimSpace(row.IslandID) != "" || row.PlannedStorage == memoryvocab.StorageExplicitIsland || row.ActualLoweringStorage == memoryvocab.StorageExplicitIsland
}

func rowRequiresArtifact(row memoryReportRow) bool {
	return memoryvocab.RowRequiresArtifact(row.PlannedStorage, row.ActualLoweringStorage, row.Claim)
}
