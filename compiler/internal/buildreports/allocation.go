package buildreports

import (
	"fmt"
	"reflect"
	"tetra_language/compiler/internal/allocplan"
	ctarget "tetra_language/compiler/target"
)

func WrapAllocationPlanReport(plan *allocplan.Plan, target string) AllocationPlanReport {
	claimLevel, evidenceScope, _ := AllocationPlanTargetStorageScope(target)
	if plan == nil {
		return AllocationPlanReport{
			ReportEnvelope:         ReportEnvelope{SchemaVersion: 2, Kind: "allocation_plan", Target: target},
			TargetMemoryClaimLevel: claimLevel,
			StorageEvidenceScope:   evidenceScope,
			Summary:                allocplan.Summarize(nil),
		}
	}
	return AllocationPlanReport{
		ReportEnvelope:         ReportEnvelope{SchemaVersion: 2, Kind: "allocation_plan", Target: target},
		TargetMemoryClaimLevel: claimLevel,
		StorageEvidenceScope:   evidenceScope,
		Summary:                allocplan.Summarize(plan),
		Totals:                 plan.Totals,
		Functions:              plan.Functions,
	}
}

func ValidateAllocationPlanReport(plan *allocplan.Plan, report AllocationPlanReport) error {
	if report.SchemaVersion != 2 || report.Kind != "allocation_plan" {
		return fmt.Errorf("allocation report mismatch: invalid envelope schema=%d kind=%q", report.SchemaVersion, report.Kind)
	}
	expectedClaimLevel, expectedEvidenceScope, err := AllocationPlanTargetStorageScope(report.Target)
	if err != nil {
		return fmt.Errorf("allocation report mismatch: target memory scope: %w", err)
	}
	if report.TargetMemoryClaimLevel != expectedClaimLevel {
		return fmt.Errorf("allocation report mismatch: target_memory_claim_level=%q want %q", report.TargetMemoryClaimLevel, expectedClaimLevel)
	}
	if report.StorageEvidenceScope != expectedEvidenceScope {
		return fmt.Errorf("allocation report mismatch: storage_evidence_scope=%q want %q", report.StorageEvidenceScope, expectedEvidenceScope)
	}
	expectedSummary := allocplan.Summarize(plan)
	if !reflect.DeepEqual(report.Summary, expectedSummary) {
		return fmt.Errorf("allocation report mismatch: summary does not match plan")
	}
	if plan == nil {
		if !reflect.DeepEqual(report.Totals, allocplan.Totals{}) || len(report.Functions) != 0 {
			return fmt.Errorf("allocation report mismatch: non-empty report for nil plan")
		}
		return nil
	}
	if !reflect.DeepEqual(report.Totals, plan.Totals) {
		return fmt.Errorf("allocation report mismatch: totals do not match plan")
	}
	if !reflect.DeepEqual(report.Functions, plan.Functions) {
		return fmt.Errorf("allocation report mismatch: functions do not match plan")
	}
	return nil
}

func AllocationPlanTargetStorageScope(triple string) (string, string, error) {
	tgt, err := ctarget.Parse(triple)
	if err != nil {
		return "", "", err
	}
	switch tgt.MemoryClaimLevel {
	case "production/host_runtime":
		return tgt.MemoryClaimLevel, "host_runtime_verified", nil
	case "build_lower_only unless run":
		return tgt.MemoryClaimLevel, "build_lower_only_target_host_required", nil
	case "artifact/runtime tiered":
		return tgt.MemoryClaimLevel, "artifact_runtime_tiered_safe_limited", nil
	case "build_lower_only":
		return tgt.MemoryClaimLevel, "build_lower_only", nil
	default:
		return tgt.MemoryClaimLevel, "target_capability_matrix", nil
	}
}
