package fromvalidation

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/validation"
)

func AddBoundsProofFacts(graph *Graph, report validation.ProofReport) error {
	if graph == nil {
		return fmt.Errorf("memoryfacts: nil graph")
	}
	for _, removed := range report.RemovedChecks {
		if strings.TrimSpace(removed.ProofID) == "" {
			if err := AddBoundsProofRejectionFact(
				graph,
				removed.Function,
				boundsSiteID(removed.Function, removed.Site),
				"removed bounds check without proof id",
			); err != nil {
				return err
			}
			continue
		}
		parentID, err := graph.AddFact(boundsProofGuardFact(removed))
		if err != nil {
			return err
		}
		if _, err := graph.DeriveFact(
			parentID,
			boundsRemovedWithProofFact(parentID, removed),
		); err != nil {
			return err
		}
	}
	if report.LeftChecks > 0 {
		parentID, err := graph.AddFact(boundsRetainedDynamicGuardFact(report.LeftChecks))
		if err != nil {
			return err
		}
		if _, err := graph.DeriveFact(
			parentID,
			boundsRetainedDynamicFact(parentID, report.LeftChecks),
		); err != nil {
			return err
		}
	}
	return nil
}

func AddBoundsProofRejectionFact(
	graph *Graph,
	functionID string,
	siteID string,
	reason string,
) error {
	if graph == nil {
		return fmt.Errorf("memoryfacts: nil graph")
	}
	if strings.TrimSpace(siteID) == "" {
		siteID = boundsSiteID(functionID, 0)
	}
	if strings.TrimSpace(reason) == "" {
		reason = "removed bounds check without compiler-owned proof id"
	}
	_, err := graph.AddFact(Fact{
		ID: FactID(
			fmt.Sprintf("validation:%s:%s:missing_proof", nonEmpty(functionID, "unknown"), siteID),
		),
		FunctionID:      functionID,
		SiteID:          siteID,
		SourceStage:     StageValidation,
		ProvenanceClass: ProvenanceSafeKnown,
		UnsafeClass:     UnsafeSafe,
		Claim:           "bounds_check_removal_rejected_missing_proof_id",
		ValidationState: ValidationFail,
		ValidatorName:   "bounds_proof_id_validator",
		CostClass:       CostUnsupportedRejected,
		Reason:          reason,
	})
	return err
}

func boundsProofGuardFact(removed validation.RemovedCheck) Fact {
	siteID := boundsSiteID(removed.Function, removed.Site)
	fact := Fact{
		ID:              boundsProofGuardFactID(removed),
		FunctionID:      removed.Function,
		SiteID:          siteID,
		SourceStage:     StageValidation,
		ProvenanceClass: ProvenanceSafeKnown,
		UnsafeClass:     UnsafeSafe,
		Claim:           "bounds_proof_id",
		ValidationState: ValidationPass,
		ValidatorName:   "bounds_proof_id_validator",
		CostClass:       CostInstrumentationOnly,
		Reason: fmt.Sprintf(
			"proof id %s validates %s using %s",
			removed.ProofID,
			removed.Kind,
			strings.Join(removed.FactsUsed, ","),
		),
	}
	attachBoundsProofTerm(&fact, removed)
	return fact
}

func boundsRemovedWithProofFact(parentID FactID, removed validation.RemovedCheck) Fact {
	siteID := boundsSiteID(removed.Function, removed.Site)
	fact := Fact{
		ID:              derivedFactID(parentID, "bounds_check_removed_with_proof_id"),
		FunctionID:      removed.Function,
		SiteID:          siteID,
		SourceStage:     StageValidation,
		ProvenanceClass: ProvenanceSafeKnown,
		UnsafeClass:     UnsafeSafe,
		Claim:           "bounds_check_removed_with_proof_id",
		ValidationState: ValidationPass,
		ValidatorName:   "bounds_proof_id_validator",
		CostClass:       CostZeroCostProven,
		Reason: fmt.Sprintf(
			"removed %s bounds check carries compiler-owned proof id %s",
			removed.Kind,
			removed.ProofID,
		),
	}
	attachBoundsProofTerm(&fact, removed)
	return fact
}

func boundsRetainedDynamicGuardFact(leftChecks int) Fact {
	return Fact{
		ID: FactID(
			fmt.Sprintf("validation:bounds:retained_dynamic:%d:guard", leftChecks),
		),
		SiteID:          "bounds:retained_dynamic",
		SourceStage:     StageValidation,
		ProvenanceClass: ProvenanceSafeKnown,
		UnsafeClass:     UnsafeSafe,
		Claim:           "normal_build_bounds_check_guard",
		ValidationState: ValidationPass,
		ValidatorName:   "normal_build_bounds_check_validator",
		CostClass:       CostInstrumentationOnly,
		Reason:          fmt.Sprintf("%d bounds checks remain in the normal build", leftChecks),
	}
}

func boundsRetainedDynamicFact(parentID FactID, leftChecks int) Fact {
	return Fact{
		ID:               derivedFactID(parentID, "bounds_check_retained_dynamic"),
		SiteID:           "bounds:retained_dynamic",
		SourceStage:      StageValidation,
		ProvenanceClass:  ProvenanceSafeKnown,
		UnsafeClass:      UnsafeSafe,
		Claim:            "bounds_check_retained_dynamic",
		ValidationState:  ValidationPass,
		ValidatorName:    "normal_build_bounds_check_validator",
		CostClass:        CostDynamicCheckRequired,
		NormalBuildCheck: true,
		Reason:           fmt.Sprintf("%d bounds checks remain in the normal build", leftChecks),
	}
}

func boundsProofGuardFactID(removed validation.RemovedCheck) FactID {
	return FactID(
		fmt.Sprintf(
			"validation:%s:%d:%s:proof_guard",
			nonEmpty(removed.Function, "unknown"),
			removed.Site,
			sanitizeFactIDPart(removed.ProofID),
		),
	)
}

func boundsSiteID(functionID string, site int) string {
	return fmt.Sprintf("bounds:%s:%d", nonEmpty(functionID, "unknown"), site)
}

func sanitizeFactIDPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "missing"
	}
	replacer := strings.NewReplacer(" ", "_", "\t", "_", "\n", "_")
	return replacer.Replace(value)
}

func derivedFactID(parentID FactID, suffix string) FactID {
	return FactID(fmt.Sprintf("%s:%s", parentID, suffix))
}

func nonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func attachBoundsProofTerm(fact *Fact, removed validation.RemovedCheck) {
	if fact == nil {
		return
	}
	fact.ProofID = removed.ProofID
	if removed.ProofTerm == nil {
		return
	}
	term := removed.ProofTerm
	fact.ProofKind = term.Kind
	fact.ProofSubjectBaseID = term.SubjectBaseID
	fact.ProofIndexValueID = term.IndexValueID
	fact.ProofOperation = term.Operation
	fact.ProofRange = term.Range
	fact.IslandID = term.IslandID
	fact.Epoch = term.Epoch
	fact.BaseID = term.BaseID
}
