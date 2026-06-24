package fromoptimizer

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/memoryfacts"
	"tetra_language/compiler/internal/opt"
)

func Delta(report opt.Report) (memoryfacts.Delta, error) {
	delta := report.MemoryDelta
	if delta.Stage == "" {
		delta.Stage = memoryfacts.StageOptimization
	}
	if delta.Stage != memoryfacts.StageOptimization {
		return memoryfacts.Delta{}, fmt.Errorf("fromoptimizer: report delta stage %q is not optimization", delta.Stage)
	}
	if len(delta.Add) == 0 {
		for passIndex, pass := range report.Passes {
			for decisionIndex, decision := range pass.Decisions {
				if decision.DecisionCode == "" &&
					decision.RewriteCategory == "" &&
					len(decision.ProofIDs) == 0 {
					continue
				}
				delta.Add = append(delta.Add, decisionFact(passIndex, decisionIndex, pass, decision))
			}
		}
	}
	return delta, nil
}

func decisionFact(
	passIndex int,
	decisionIndex int,
	pass opt.PassReport,
	decision opt.PassDecision,
) memoryfacts.Fact {
	proofIDs := cleanStrings(decision.ProofIDs)
	proofFactIDs := cleanStrings(decision.ProofFactIDs)
	fact := memoryfacts.Fact{
		ID: memoryfacts.FactID(fmt.Sprintf(
			"optimizer:decision:%03d:%03d:%s:%d",
			passIndex,
			decisionIndex,
			safeFactPart(pass.Name+"-"+decision.Action),
			decision.Site,
		)),
		FunctionID:      decision.Caller,
		SiteID:          fmt.Sprintf("%d", decision.Site),
		SourceStage:     memoryfacts.StageOptimization,
		Claim:           memoryfacts.ClaimOptimizerDecision,
		ProvenanceClass: memoryfacts.ProvenanceSafeKnown,
		UnsafeClass:     memoryfacts.UnsafeSafe,
		ProofID:         firstString(proofIDs),
		DecisionCode:    string(decision.DecisionCode),
		Reason: fmt.Sprintf(
			"%s site=%d action=%s reason=%s",
			pass.Name,
			decision.Site,
			decision.Action,
			decision.Reason,
		),
	}
	if len(proofFactIDs) > 0 {
		fact.ParentFactID = memoryfacts.FactID(proofFactIDs[0])
	}
	if decision.DecisionCode == opt.DecisionCodeProofUnsafe {
		fact.ProvenanceClass = memoryfacts.ProvenanceUnsafeUnknown
		fact.UnsafeClass = memoryfacts.UnsafeUnknown
		fact.AliasState = memoryfacts.AliasUnknownConservative
	}
	return fact
}

func cleanStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func safeFactPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer(
		" ", "_",
		":", "_",
		"/", "_",
		"\\", "_",
		"\t", "_",
		"\n", "_",
	)
	return replacer.Replace(value)
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
