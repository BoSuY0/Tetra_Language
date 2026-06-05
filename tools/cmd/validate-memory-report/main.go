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
	ValidatorName         string `json:"validator_name,omitempty"`
	ValidatorStatus       string `json:"validator_status,omitempty"`
	CostClass             string `json:"cost_class,omitempty"`
	NormalBuildCheck      bool   `json:"normal_build_check,omitempty"`
	Reason                string `json:"reason,omitempty"`
}

func main() {
	reportPath := flag.String("report", "", "path to tetra.memory-report.v1 JSON report")
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateMemoryReport(*reportPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateMemoryReport(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var report memoryReport
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("trailing data after memory report JSON")
		}
		return fmt.Errorf("trailing data after memory report JSON: %w", err)
	}
	return validateReport(report)
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
	if row.CostClass == "dynamic_check_required" && !row.NormalBuildCheck {
		issues = append(issues, prefix+": dynamic_check_required requires normal_build_check")
	}
	if row.ClaimLevel == "validated" && row.ValidatorStatus != "pass" {
		issues = append(issues, prefix+": validated claim requires validator_status pass")
	}
	if row.ClaimLevel == "validated" && strings.TrimSpace(row.ValidatorName) == "" {
		issues = append(issues, prefix+": validated claim requires validator_name")
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
	if unsafeVerifiedRootDisallowedClaim(row.ProvenanceClass, row.UnsafeClass, row.Claim) {
		issues = append(issues, fmt.Sprintf("%s: unsafe_verified_root cannot claim %q without bounded raw metadata", prefix, row.Claim))
	}
	if unsafeUnknownRow(row) && row.ClaimLevel == "validated" && unsafeUnknownTrustedStorage(row.PlannedStorage, row.ActualLoweringStorage) {
		issues = append(issues, fmt.Sprintf("%s: unsafe_unknown cannot validate trusted storage lowering %q/%q", prefix, row.PlannedStorage, row.ActualLoweringStorage))
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

func knownSourceStage(value string) bool {
	switch value {
	case "semantics", "unsafe_gateway_lowering", "plir", "allocplan", "lowering", "validation":
		return true
	default:
		return false
	}
}

func knownProvenance(value string) bool {
	switch value {
	case "safe_known", "safe_borrowed", "safe_owned", "unsafe_unknown", "unsafe_checked", "unsafe_verified_root":
		return true
	default:
		return false
	}
}

func safeProvenance(value string) bool {
	switch value {
	case "safe_known", "safe_borrowed", "safe_owned":
		return true
	default:
		return false
	}
}

func unsafeUnknownRow(row memoryReportRow) bool {
	return row.ProvenanceClass == "unsafe_unknown" || row.UnsafeClass == "unsafe_unknown"
}

func unsafeUnknownOptimizationClaim(claim string, aliasState string) bool {
	claim = strings.ToLower(strings.TrimSpace(claim))
	switch claim {
	case "provenance_known", "no_alias", "index_in_range", "bounds_check_eliminated", "trusted_storage":
		return true
	case "safe_known", "safe_borrowed", "safe_owned":
		return true
	}
	return aliasState == "unique" || aliasState == "mutable_exclusive"
}

func memoryOptimizationClaim(claim string, aliasState string) bool {
	if unsafeUnknownOptimizationClaim(claim, aliasState) {
		return true
	}
	claim = strings.ToLower(strings.TrimSpace(claim))
	return strings.Contains(claim, "eliminated") || strings.Contains(claim, "zero_cost")
}

func broadNoAliasClaim(claim string) bool {
	claim = strings.ToLower(strings.TrimSpace(claim))
	return claim == "broad_noalias" || claim == "universal_noalias" || claim == "full_noalias_model"
}

func claimRequiresParentFactID(claim string) bool {
	switch strings.ToLower(strings.TrimSpace(claim)) {
	case "borrow_owner", "borrow_source_fact_id", "aggregate_contains_borrow",
		"optional_contains_borrow", "enum_payload_contains_borrow",
		"generic_wrapper_contains_borrow", "function_value_contains_borrow",
		"callback_arg_contains_borrow", "callback_inout_conservative",
		"interface_value_contains_borrow", "protocol_dispatch_borrow_conservative",
		"protocol_dispatch_noalias_conservative",
		"async_boundary_borrow_conservative", "task_boundary_borrow_rejected",
		"actor_boundary_borrow_rejected", "boundary_noalias_conservative",
		"unsafe_unknown_rejected_safe_facts",
		"unsafe_verified_root_allocation_base",
		"bounds_check_removed_with_proof_id",
		"raw_bounds_runtime_check_normal_build",
		"ffi_call_may_retain_borrow",
		"ffi_noalias_invalidated_by_external_call",
		"safe_wrapper_promotion_rejected_without_contract",
		"external_pointer_provenance_rejected",
		"copy_owned", "copy_source_fact_id",
		"mutable_exclusive", "start_inout_exclusive", "end_inout_exclusive",
		"no_alias_validated_narrow_unique_local",
		"no_alias_validated_narrow_sequential_inout":
		return true
	default:
		return false
	}
}

func knownCostClass(value string) bool {
	switch value {
	case "zero_cost_proven", "dynamic_check_required", "instrumentation_only", "unsupported_rejected", "conservative_fallback":
		return true
	default:
		return false
	}
}

func unsafeVerifiedRootDisallowedClaim(provenanceClass string, unsafeClass string, claim string) bool {
	if provenanceClass != "unsafe_verified_root" && unsafeClass != "unsafe_verified_root" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(claim)) {
	case "allocation_base_metadata", "unsafe_verified_root_allocation_base":
		return false
	default:
		return true
	}
}

func unsafeUnknownTrustedStorage(planned string, actual string) bool {
	return trustedStorageForUnsafeUnknown(planned) || trustedStorageForUnsafeUnknown(actual)
}

func validatedTrustedStorageHeapFallback(planned string, actual string) bool {
	if actual != "Heap" {
		return false
	}
	switch planned {
	case "Eliminated", "Register", "Stack", "Region", "ExplicitIsland",
		"FunctionTempRegion", "TaskRegion", "ActorMoveRegion":
		return true
	default:
		return false
	}
}

func trustedStorageForUnsafeUnknown(value string) bool {
	switch value {
	case "Eliminated", "Register", "Stack", "Region", "ExplicitIsland",
		"FunctionTempRegion", "TaskRegion", "ActorMoveRegion":
		return true
	default:
		return false
	}
}

func knownStorage(value string) bool {
	switch value {
	case "UnknownConservative", "Eliminated", "Register", "Heap", "Stack", "Region",
		"ExplicitIsland", "FunctionTempRegion", "TaskRegion", "ActorMoveRegion",
		"LargeMmap", "External":
		return true
	default:
		return false
	}
}

func knownAliasState(value string) bool {
	switch value {
	case "", "unique", "shared_readonly", "mutable_exclusive", "maybe_alias", "unknown_alias", "invalidated_by_call":
		return true
	default:
		return false
	}
}

func validatedNoAliasState(value string) bool {
	return value == "unique" || value == "mutable_exclusive"
}

func knownUnsafe(value string) bool {
	switch value {
	case "safe", "unsafe_unknown", "unsafe_checked", "unsafe_verified_root":
		return true
	default:
		return false
	}
}

func knownClaimLevel(value string) bool {
	switch value {
	case "validated", "evidence_only", "conservative", "rejected", "future":
		return true
	default:
		return false
	}
}

func knownValidatorStatus(value string) bool {
	switch value {
	case "pass", "fail", "not_applicable", "not_run":
		return true
	default:
		return false
	}
}

func rowRequiresArtifact(row memoryReportRow) bool {
	if row.PlannedStorage != "" || row.ActualLoweringStorage != "" {
		return true
	}
	claim := strings.ToLower(row.Claim)
	return strings.Contains(claim, "lowering") || strings.Contains(claim, "storage")
}
