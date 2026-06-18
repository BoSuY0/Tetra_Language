package compiler

import (
	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/buildreports"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/semantics"
)

func buildProofReport(plirProg *plir.Program, bounds boundsReport, target string) proofReport {
	return buildreports.BuildProofReport(plirProg, bounds, target)
}

func wrapAllocationPlanReport(plan *allocplan.Plan, target string) allocationPlanReport {
	return buildreports.WrapAllocationPlanReport(plan, target)
}

func validateAllocationPlanReport(plan *allocplan.Plan, report allocationPlanReport) error {
	return buildreports.ValidateAllocationPlanReport(plan, report)
}

func allocationPlanTargetStorageScope(triple string) (string, string, error) {
	return buildreports.AllocationPlanTargetStorageScope(triple)
}

func buildBoundsReport(prog *ir.IRProgram, checked *semantics.CheckedProgram, target string) boundsReport {
	return buildreports.BuildBoundsReport(prog, checked, target)
}
