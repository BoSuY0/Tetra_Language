package fromoptimizer

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/memoryfacts"
	"tetra_language/compiler/internal/opt"
)

func TestT13DeltaConvertsOptimizerReportToCanonicalFacts(t *testing.T) {
	report := opt.Report{
		MemorySnapshotBefore: "before-digest",
		MemorySnapshotAfter:  "after-digest",
		MemoryDelta: memoryfacts.Delta{
			Invalidate: []memoryfacts.Invalidation{{
				FactID: "fact:t13:proof",
				Reason: "loop-canonicalization site=13",
			}},
		},
		Passes: []opt.PassReport{{
			Name: "loop-canonicalization",
			Decisions: []opt.PassDecision{{
				Action:          "canonicalized",
				Caller:          "main",
				Site:            13,
				Reason:          "stable_len_le_minus_one_to_lt",
				DecisionCode:    opt.DecisionCodeRewriteApplied,
				RewriteCategory: opt.RewriteBoundsCheckRemoval,
				ProofIDs:        []string{"proof:while:i:xs:1:1"},
			}},
		}},
	}

	delta, err := Delta(report)
	if err != nil {
		t.Fatalf("Delta: %v", err)
	}
	if delta.Stage != memoryfacts.StageOptimization {
		t.Fatalf("stage = %q, want optimization", delta.Stage)
	}
	if len(delta.Invalidate) != 1 || delta.Invalidate[0].FactID != "fact:t13:proof" {
		t.Fatalf("invalidations = %#v", delta.Invalidate)
	}
	if len(delta.Add) != 1 {
		t.Fatalf("added facts = %#v, want one decision fact", delta.Add)
	}
	fact := delta.Add[0]
	if fact.SourceStage != memoryfacts.StageOptimization ||
		fact.Claim != memoryfacts.ClaimOptimizerDecision ||
		fact.ProofID != "proof:while:i:xs:1:1" ||
		fact.DecisionCode != string(opt.DecisionCodeRewriteApplied) {
		t.Fatalf("decision fact = %#v", fact)
	}
	if !strings.Contains(fact.Reason, "loop-canonicalization") ||
		!strings.Contains(fact.Reason, "site=13") {
		t.Fatalf("decision fact reason does not include pass/site: %#v", fact)
	}
}
