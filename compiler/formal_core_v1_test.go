package compiler

import (
	"strings"
	"testing"
)

func TestP23FormalCoreV1CoversMachineCheckableCoreRules(t *testing.T) {
	report, err := BuildP23FormalCoreV1Report()
	if err != nil {
		t.Fatalf("BuildP23FormalCoreV1Report: %v", err)
	}
	if report.SchemaVersion != formalCoreV1Schema {
		t.Fatalf("schema = %q, want %q", report.SchemaVersion, formalCoreV1Schema)
	}
	if report.Scope != formalCoreV1ScopeP232 {
		t.Fatalf("scope = %q, want %q", report.Scope, formalCoreV1ScopeP232)
	}
	if err := ValidateP23FormalCoreV1Report(report); err != nil {
		t.Fatalf("ValidateP23FormalCoreV1Report: %v", err)
	}

	rows := map[FormalCoreV1ID]FormalCoreV1Row{}
	for _, row := range report.Rows {
		if row.ID == "" || row.Name == "" || row.Status == "" || len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 || len(row.WitnessIDs) == 0 {
			t.Fatalf("row missing required metadata: %#v", row)
		}
		rows[row.ID] = row
	}
	for _, id := range p23FormalCoreV1IDs() {
		if _, ok := rows[id]; !ok {
			t.Fatalf("missing row %s: %#v", id, report.Rows)
		}
	}
	p23AssertFormalCoreRow(t, rows[FormalCoreV1Values], []string{"differential", "stable observable", "i32"})
	p23AssertFormalCoreRow(t, rows[FormalCoreV1BorrowsOwnedCopy], []string{"borrow", "copy", "owned"})
	p23AssertFormalCoreRow(t, rows[FormalCoreV1ProvenanceRegions], []string{"provenance", "regions", "PLIR"})
	p23AssertFormalCoreRow(t, rows[FormalCoreV1BoundsProofIDSemantics], []string{"proof id", "proof guards", "CheckBoundsProofsWithPLIR"})
	p23AssertFormalCoreRow(t, rows[FormalCoreV1AllocationLengthContract], []string{"length contract", "negative", "overflow"})
	p23AssertFormalCoreRow(t, rows[FormalCoreV1AllocationIntentLowering], []string{"allocation intent", "ValidateAllocationLowering"})
	p23AssertFormalCoreRow(t, rows[FormalCoreV1RawPointerBoundsMetadata], []string{"raw pointer bounds", "allocation-base", "external/unknown"})
	p23AssertFormalCoreRow(t, rows[FormalCoreV1CheckEliminationValidity], []string{"unchecked", "proof id", "safe-semantics"})

	witnesses := map[string]FormalCoreV1Witness{}
	for _, witness := range report.Witnesses {
		witnesses[witness.ID] = witness
	}
	spec := witnesses[p23FormalCoreSpecWitnessID]
	if !spec.FormalSpecValid || spec.FormalConcepts < 9 || spec.FormalRules < 7 {
		t.Fatalf("formal spec witness = %#v, want valid expanded concept/rule inventory", spec)
	}
	values := witnesses[p23FormalCoreValuesWitnessID]
	if values.ValueSamples == 0 || values.DifferentialLanes < 5 {
		t.Fatalf("values witness = %#v, want backend differential value evidence", values)
	}
	plir := witnesses[p23FormalCorePLIRWitnessID]
	if !plir.BorrowCopyFacts || !plir.ProvenanceRegionFacts {
		t.Fatalf("PLIR witness = %#v, want borrow/copy and provenance/region facts", plir)
	}
	proof := witnesses[p23FormalCoreProofWitnessID]
	if !proof.BoundsProofIDsChecked || !proof.MissingProofRejected || !proof.CheckEliminationValidated {
		t.Fatalf("proof witness = %#v, want proof-id validation and check-elimination rejection", proof)
	}
	allocation := witnesses[p23FormalCoreAllocationWitnessID]
	if !allocation.AllocationLengthContractsChecked || !allocation.InvalidAllocationLengthRejected || !allocation.AllocationIntentLoweringValidated || !allocation.AllocationIntentDriftRejected {
		t.Fatalf("allocation witness = %#v, want length contract and lowering validation", allocation)
	}
	raw := witnesses[p23FormalCoreRawPointerWitnessID]
	if raw.RawPointerBoundsCases < 4 || !raw.RawPointerImpossibleAddRejected || !raw.RawPointerUnknownStayedChecked {
		t.Fatalf("raw pointer witness = %#v, want allocation-base, derived, rejected, and checked-unknown metadata", raw)
	}

	for _, nonClaim := range []string{
		"no full formal proof of Tetra is claimed",
		"no broad language theorem prover is claimed",
		"unsafe policy does not change",
		"runtime behavior does not change",
		"safe-program semantics do not change",
		"no performance claim is made",
	} {
		if !p23FormalCoreHasString(report.NonClaims, nonClaim) {
			t.Fatalf("missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestValidateP23FormalCoreV1RejectsFakeClaimsAndDrift(t *testing.T) {
	base, err := BuildP23FormalCoreV1Report()
	if err != nil {
		t.Fatalf("BuildP23FormalCoreV1Report: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*FormalCoreV1Report)
		want   string
	}{
		{
			name: "missing row",
			mutate: func(report *FormalCoreV1Report) {
				report.Rows = report.Rows[1:]
			},
			want: "missing row",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *FormalCoreV1Report) {
				report.Rows[0].Evidence = []string{"TODO"}
			},
			want: "placeholder",
		},
		{
			name: "missing witness",
			mutate: func(report *FormalCoreV1Report) {
				report.Rows[0].WitnessIDs = []string{"missing-witness"}
			},
			want: "missing witness",
		},
		{
			name: "formal spec invalid",
			mutate: func(report *FormalCoreV1Report) {
				report.FormalSpecValid = false
			},
			want: "formal spec",
		},
		{
			name: "missing values",
			mutate: func(report *FormalCoreV1Report) {
				report.ValueSamples = 0
			},
			want: "values",
		},
		{
			name: "missing PLIR facts",
			mutate: func(report *FormalCoreV1Report) {
				report.BorrowCopyFacts = false
			},
			want: "borrow",
		},
		{
			name: "missing proof ids",
			mutate: func(report *FormalCoreV1Report) {
				report.BoundsProofIDsChecked = false
			},
			want: "bounds proof",
		},
		{
			name: "missing allocation length",
			mutate: func(report *FormalCoreV1Report) {
				report.AllocationLengthContractsChecked = false
			},
			want: "allocation length",
		},
		{
			name: "missing raw pointer bounds",
			mutate: func(report *FormalCoreV1Report) {
				report.RawPointerBoundsCases = 0
			},
			want: "raw pointer",
		},
		{
			name: "full formal proof claim",
			mutate: func(report *FormalCoreV1Report) {
				report.FullFormalProofClaimed = true
			},
			want: "full formal proof",
		},
		{
			name: "broad language proof claim",
			mutate: func(report *FormalCoreV1Report) {
				report.BroadLanguageProofClaimed = true
			},
			want: "broad language",
		},
		{
			name: "unsafe policy change",
			mutate: func(report *FormalCoreV1Report) {
				report.UnsafePolicyChanged = true
			},
			want: "unsafe policy",
		},
		{
			name: "runtime behavior change",
			mutate: func(report *FormalCoreV1Report) {
				report.RuntimeBehaviorChanged = true
			},
			want: "runtime behavior",
		},
		{
			name: "safe semantics change",
			mutate: func(report *FormalCoreV1Report) {
				report.SafeSemanticsChanged = true
			},
			want: "safe semantics",
		},
		{
			name: "performance claim",
			mutate: func(report *FormalCoreV1Report) {
				report.PerformanceClaimed = true
			},
			want: "performance",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := base
			report.Rows = append([]FormalCoreV1Row(nil), base.Rows...)
			report.Witnesses = append([]FormalCoreV1Witness(nil), base.Witnesses...)
			report.NonClaims = append([]string(nil), base.NonClaims...)
			tc.mutate(&report)
			err := ValidateP23FormalCoreV1Report(report)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateP23FormalCoreV1Report error = %v, want %q", err, tc.want)
			}
		})
	}
}

func p23AssertFormalCoreRow(t *testing.T, row FormalCoreV1Row, wants []string) {
	t.Helper()
	text := strings.Join(row.Evidence, " ") + " " + strings.Join(row.Boundaries, " ")
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("row %s missing %q: %#v", row.ID, want, row)
		}
	}
}
