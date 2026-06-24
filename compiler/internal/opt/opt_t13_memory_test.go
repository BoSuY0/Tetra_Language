package opt

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/memoryfacts"
)

func TestT13LoopCanonicalizationRequiresCanonicalBoundsProof(t *testing.T) {
	prog := loopCanonicalizationProgram(ir.IRCmpLeI32, true)
	snapshot := t13MemorySnapshot(t)

	report, err := NewManager().RunWithOptions(
		prog,
		Options{MemoryFacts: snapshot},
		LoopCanonicalizationPass(),
	)
	if err != nil {
		t.Fatalf("RunWithOptions: %v", err)
	}

	row := report.Passes[0]
	if strings.Contains(row.AfterDump, "store_local local:4") {
		t.Fatalf("proof-sensitive loop rewrite ran without canonical proof:\n%s", row.AfterDump)
	}
	decision := t13FindDecisionCode(row.Decisions, DecisionCodeProofMissing)
	if decision == nil {
		t.Fatalf("missing proof decision not recorded: %#v", row.Decisions)
	}
	if decision.RewriteCategory != RewriteBoundsCheckRemoval ||
		len(decision.ProofIDs) != 1 ||
		decision.ProofIDs[0] != proofID(true) {
		t.Fatalf("missing proof decision lacks rewrite site/proof evidence: %#v", decision)
	}
	if report.MemorySnapshotBefore != snapshot.Digest() {
		t.Fatalf("before digest = %q, want %q", report.MemorySnapshotBefore, snapshot.Digest())
	}
	if len(report.MemoryDelta.Add) == 0 {
		t.Fatalf("missing proof decision did not produce optimizer memory delta: %#v", report)
	}
}

func TestT13ProofSensitiveRewriteRecordsCanonicalProofID(t *testing.T) {
	prog := loopCanonicalizationProgram(ir.IRCmpLeI32, true)
	snapshot := t13MemorySnapshot(t, t13BoundsProofFact())

	report, err := NewManager().RunWithOptions(
		prog,
		Options{MemoryFacts: snapshot},
		LoopCanonicalizationPass(),
	)
	if err != nil {
		t.Fatalf("RunWithOptions: %v", err)
	}

	row := report.Passes[0]
	if !strings.Contains(row.AfterDump, "store_local local:4") {
		t.Fatalf("canonical proof did not authorize loop rewrite:\n%s", row.AfterDump)
	}
	decision := t13FindDecisionCode(row.Decisions, DecisionCodeRewriteApplied)
	if decision == nil {
		t.Fatalf("rewrite decision not recorded: %#v", row.Decisions)
	}
	if decision.RewriteCategory != RewriteBoundsCheckRemoval ||
		len(decision.ProofIDs) != 1 ||
		decision.ProofIDs[0] != proofID(true) {
		t.Fatalf("rewrite decision lacks canonical proof id: %#v", decision)
	}
	if report.MemorySnapshotBefore != snapshot.Digest() || strings.TrimSpace(report.MemorySnapshotAfter) == "" {
		t.Fatalf(
			"memory snapshot digests not recorded before=%q after=%q",
			report.MemorySnapshotBefore,
			report.MemorySnapshotAfter,
		)
	}
}

func TestT13ProofSensitiveRewriteSkipsInvalidatedAndUnsafeProofs(t *testing.T) {
	tests := []struct {
		name string
		fact memoryfacts.Fact
		code DecisionCode
	}{
		{
			name: "invalidated",
			fact: func() memoryfacts.Fact {
				f := t13BoundsProofFact()
				f.ValidationState = memoryfacts.ValidationInvalidated
				f.Reason = "stale optimizer input"
				return f
			}(),
			code: DecisionCodeProofInvalidated,
		},
		{
			name: "unsafe",
			fact: func() memoryfacts.Fact {
				f := t13BoundsProofFact()
				f.Claim = memoryfacts.ClaimProvenanceUnknown
				f.ProvenanceClass = memoryfacts.ProvenanceUnsafeUnknown
				f.UnsafeClass = memoryfacts.UnsafeUnknown
				f.AliasState = memoryfacts.AliasUnknownConservative
				return f
			}(),
			code: DecisionCodeProofUnsafe,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog := loopCanonicalizationProgram(ir.IRCmpLeI32, true)
			snapshot := t13MemorySnapshot(t, tc.fact)

			report, err := NewManager().RunWithOptions(
				prog,
				Options{MemoryFacts: snapshot},
				LoopCanonicalizationPass(),
			)
			if err != nil {
				t.Fatalf("RunWithOptions: %v", err)
			}
			if strings.Contains(report.Passes[0].AfterDump, "store_local local:4") {
				t.Fatalf("%s proof authorized rewrite:\n%s", tc.name, report.Passes[0].AfterDump)
			}
			if decision := t13FindDecisionCode(report.Passes[0].Decisions, tc.code); decision == nil {
				t.Fatalf("%s decision code missing from %#v", tc.code, report.Passes[0].Decisions)
			}
		})
	}
}

func TestT13ManagerRejectsPreserveAndInvalidateSameProofKind(t *testing.T) {
	pass := p17ContractTestPass("bad-memory-metadata")
	pass.PreservedProofKinds = []memoryfacts.ProofKind{memoryfacts.ProofNoAlias}
	pass.InvalidatedProofKinds = []memoryfacts.ProofKind{memoryfacts.ProofNoAlias}

	_, err := NewManager().Run(validTinyProgram(), pass)
	if err == nil || !strings.Contains(err.Error(), "preserve and invalidate proof kind") {
		t.Fatalf("Run error = %v, want preserve/invalidate proof kind rejection", err)
	}
}

func TestT13ManagerRejectsMemoryRewriteDecisionWithoutProofID(t *testing.T) {
	pass := p17ContractTestPass("bad-memory-decision")
	pass.Decisions = func() []PassDecision {
		return []PassDecision{{
			Action:          "rewrote",
			Site:            7,
			Reason:          "missing proof id",
			DecisionCode:    DecisionCodeRewriteApplied,
			RewriteCategory: RewriteNoAliasRewrite,
		}}
	}

	_, err := NewManager().Run(validTinyProgram(), pass)
	if err == nil || !strings.Contains(err.Error(), "missing proof id") {
		t.Fatalf("Run error = %v, want missing proof id rejection", err)
	}
}

func TestT13ManagerRejectsMemoryRewriteDecisionWithNoncanonicalProofID(t *testing.T) {
	pass := p17ContractTestPass("bad-memory-proof-id")
	pass.RequiredProofKinds = []memoryfacts.ProofKind{memoryfacts.ProofBounds}
	pass.Decisions = func() []PassDecision {
		return []PassDecision{{
			Action:          "rewrote",
			Caller:          "main",
			Site:            8,
			Reason:          "bogus proof id",
			DecisionCode:    DecisionCodeRewriteApplied,
			RewriteCategory: RewriteBoundsCheckRemoval,
			ProofIDs:        []string{"proof:bogus"},
		}}
	}

	_, err := NewManager().RunWithOptions(
		validTinyProgram(),
		Options{MemoryFacts: t13MemorySnapshot(t, t13BoundsProofFact())},
		pass,
	)
	if err == nil || !strings.Contains(err.Error(), "noncanonical proof id") {
		t.Fatalf("RunWithOptions error = %v, want noncanonical proof id rejection", err)
	}
}

func TestT13ManagerRunIsNoncanonicalForMemoryProofResolution(t *testing.T) {
	report, err := NewManager().Run(validTinyProgram(), p17ContractTestPass("noncanonical-run"))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.MemorySnapshotBefore != "" || report.MemorySnapshotAfter != "" {
		t.Fatalf("Run memory snapshots = %q/%q, want empty noncanonical run", report.MemorySnapshotBefore, report.MemorySnapshotAfter)
	}

	snapshot := t13MemorySnapshot(t, t13BoundsProofFact())
	canonical, err := NewManager().RunWithOptions(
		validTinyProgram(),
		Options{MemoryFacts: snapshot},
		p17ContractTestPass("canonical-run"),
	)
	if err != nil {
		t.Fatalf("RunWithOptions: %v", err)
	}
	if canonical.MemorySnapshotBefore != snapshot.Digest() || canonical.MemorySnapshotAfter == "" {
		t.Fatalf("RunWithOptions memory snapshots = %q/%q, want canonical digest %q", canonical.MemorySnapshotBefore, canonical.MemorySnapshotAfter, snapshot.Digest())
	}
}

func TestT13ManagerRejectsInvalidatingRewriteWithoutDelta(t *testing.T) {
	pass := p17ContractTestPass("bad-invalidating-memory-decision")
	pass.InvalidatedProofKinds = []memoryfacts.ProofKind{memoryfacts.ProofNoAlias}
	pass.Decisions = func() []PassDecision {
		return []PassDecision{{
			Action:          "rewrote",
			Site:            9,
			Reason:          "no invalidation delta",
			DecisionCode:    DecisionCodeRewriteApplied,
			RewriteCategory: RewriteNoAliasRewrite,
			ProofIDs:        []string{"proof:noalias"},
		}}
	}

	_, err := NewManager().Run(validTinyProgram(), pass)
	if err == nil || !strings.Contains(err.Error(), "missing memoryfacts invalidation delta") {
		t.Fatalf("Run error = %v, want invalidation delta rejection", err)
	}
}

func t13MemorySnapshot(t *testing.T, facts ...memoryfacts.Fact) memoryfacts.Snapshot {
	t.Helper()
	graph := memoryfacts.NewGraph("t13")
	for _, fact := range facts {
		if _, err := graph.AddFact(fact); err != nil {
			t.Fatalf("AddFact(%s): %v", fact.ID, err)
		}
	}
	snapshot, err := graph.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	return snapshot
}

func t13BoundsProofFact() memoryfacts.Fact {
	return memoryfacts.Fact{
		ID:                 "fact:t13:bounds",
		FunctionID:         "main",
		SourceStage:        memoryfacts.StagePLIR,
		Claim:              memoryfacts.ClaimBoundsProofID,
		ProvenanceClass:    memoryfacts.ProvenanceSafeKnown,
		UnsafeClass:        memoryfacts.UnsafeSafe,
		AliasState:         memoryfacts.AliasUnique,
		ProofID:            proofID(true),
		ProofKind:          memoryfacts.ProofBounds,
		ProofSubjectBaseID: "xs",
		ProofIndexValueID:  "i",
		ProofOperation:     "index_load",
		ProofRange:         "0 <= i < len(xs)",
		ValidationState:    memoryfacts.ValidationPass,
		ValidatorName:      "t13_bounds_validator",
	}
}

func t13FindDecisionCode(decisions []PassDecision, code DecisionCode) *PassDecision {
	for i := range decisions {
		if decisions[i].DecisionCode == code {
			return &decisions[i]
		}
	}
	return nil
}
