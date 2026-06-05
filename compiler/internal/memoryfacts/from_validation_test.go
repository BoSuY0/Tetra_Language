package memoryfacts

import (
	"testing"

	"tetra_language/compiler/internal/validation"
)

func TestMemoryIdealV6ProjectsBoundsProofFacts(t *testing.T) {
	graph := NewGraph("program")
	err := AddBoundsProofFacts(graph, validation.ProofReport{
		RemovedChecks: []validation.RemovedCheck{{
			Function:  "sum",
			Site:      3,
			Kind:      "index_load_i32",
			ProofID:   "proof:while:i:xs:1:1",
			FactsUsed: []string{"index_in_range", "len_stable"},
		}},
		LeftChecks: 1,
	})
	if err != nil {
		t.Fatalf("AddBoundsProofFacts: %v", err)
	}

	report := BuildReportFromGraph(graph)
	removed, ok := reportRowByClaim(report, "bounds_check_removed_with_proof_id")
	if !ok {
		t.Fatalf("report missing proof-id removal row: %+v", report.Rows)
	}
	if removed.ClaimLevel != ClaimValidated ||
		removed.CostClass != CostZeroCostProven ||
		removed.ValidatorName != "bounds_proof_id_validator" ||
		removed.ParentFactID == "" ||
		removed.SourceStage != StageValidation {
		t.Fatalf("removed bounds row = %+v, want validated zero-cost proof-id row with parent fact", removed)
	}

	retained, ok := reportRowByClaim(report, "bounds_check_retained_dynamic")
	if !ok {
		t.Fatalf("report missing retained dynamic row: %+v", report.Rows)
	}
	if retained.ClaimLevel != ClaimValidated ||
		retained.CostClass != CostDynamicCheckRequired ||
		!retained.NormalBuildCheck ||
		retained.ValidatorName != "normal_build_bounds_check_validator" {
		t.Fatalf("retained bounds row = %+v, want validated dynamic normal-build check", retained)
	}
}

func TestMemoryIdealV6ProjectsMissingProofRejection(t *testing.T) {
	graph := NewGraph("program")
	if err := AddBoundsProofRejectionFact(graph, "sum", "bounds:sum:4", "removed bounds check without proof id"); err != nil {
		t.Fatalf("AddBoundsProofRejectionFact: %v", err)
	}
	report := BuildReportFromGraph(graph)
	row, ok := reportRowByClaim(report, "bounds_check_removal_rejected_missing_proof_id")
	if !ok {
		t.Fatalf("report missing missing-proof rejection row: %+v", report.Rows)
	}
	if row.ClaimLevel != ClaimRejected ||
		row.CostClass != CostUnsupportedRejected ||
		row.ValidatorName != "bounds_proof_id_validator" ||
		row.ValidatorStatus != ValidatorFail {
		t.Fatalf("missing-proof row = %+v, want rejected proof-id validator failure", row)
	}
}
