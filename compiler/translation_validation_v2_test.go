package compiler

import (
	"strings"
	"testing"
)

func TestP23TranslationValidationV2CoversSupportedOptimizerSubset(t *testing.T) {
	report, err := BuildP23TranslationValidationV2()
	if err != nil {
		t.Fatalf("BuildP23TranslationValidationV2: %v", err)
	}
	if report.SchemaVersion != translationValidationV2Schema {
		t.Fatalf("schema = %q, want %q", report.SchemaVersion, translationValidationV2Schema)
	}
	if report.Scope != translationValidationV2ScopeP230 {
		t.Fatalf("scope = %q, want %q", report.Scope, translationValidationV2ScopeP230)
	}
	if err := ValidateP23TranslationValidationV2(report); err != nil {
		t.Fatalf("ValidateP23TranslationValidationV2: %v", err)
	}

	rows := map[TranslationValidationV2ID]TranslationValidationV2Row{}
	for _, row := range report.Rows {
		if row.ID == "" || row.Name == "" || row.Status == "" || len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 || len(row.WitnessIDs) == 0 {
			t.Fatalf("row missing required metadata: %#v", row)
		}
		rows[row.ID] = row
	}
	for _, id := range p23TranslationValidationV2IDs() {
		if _, ok := rows[id]; !ok {
			t.Fatalf("missing row %s: %#v", id, report.Rows)
		}
	}
	p23AssertTranslationRow(t, rows[TranslationValidationV2RegisteredPasses], []string{"RegisteredPasses", "translation_validation", "validation metadata"})
	p23AssertTranslationRow(t, rows[TranslationValidationV2SymbolicScalar], []string{"symbolic", "scalar arithmetic", "semantic local equivalence"})
	p23AssertTranslationRow(t, rows[TranslationValidationV2MemoryEquivalence], []string{"i32 slice", "memory", "backend matrix"})
	p23AssertTranslationRow(t, rows[TranslationValidationV2BoundsProofPreservation], []string{"proof facts", "missing proof id", "bounds proof"})
	p23AssertTranslationRow(t, rows[TranslationValidationV2AllocationPlanPreservation], []string{"ValidateAllocationLowering", "allocation plan"})
	p23AssertTranslationRow(t, rows[TranslationValidationV2MachineCheckableHashes], []string{"sha256", "before", "after"})

	witnesses := map[string]TranslationValidationV2Witness{}
	for _, witness := range report.Witnesses {
		witnesses[witness.ID] = witness
	}
	registered := witnesses[p23TranslationRegisteredPassesWitnessID]
	if registered.RegisteredPasses < 6 || !registered.RegisteredPassCoverageComplete || !registered.TranslationMetadataPresent {
		t.Fatalf("registered pass witness = %#v, want all registered passes covered with metadata", registered)
	}
	scalar := witnesses[p23TranslationScalarWitnessID]
	if scalar.SymbolicScalarChecks == 0 || scalar.DifferentialSamples == 0 || !scalar.SemanticMismatchRejected {
		t.Fatalf("scalar witness = %#v, want symbolic checks, samples, and mismatch rejection", scalar)
	}
	memory := witnesses[p23TranslationMemoryWitnessID]
	if memory.MemoryEquivalenceSamples == 0 || memory.DifferentialLanes < 5 || !memory.MemoryMismatchRejected {
		t.Fatalf("memory witness = %#v, want memory samples, matrix lanes, and mismatch rejection", memory)
	}
	loop := witnesses[p23TranslationLoopWitnessID]
	if loop.LoopEquivalenceSamples == 0 || loop.DifferentialLanes < 5 {
		t.Fatalf("loop witness = %#v, want loop equivalence samples and matrix lanes", loop)
	}
	call := witnesses[p23TranslationCallInliningWitnessID]
	if call.CallEquivalenceSamples == 0 || !call.BeforeHadCall || call.AfterHadCall || !call.TranslationValidated {
		t.Fatalf("call/inlining witness = %#v, want call removed by validated inlining", call)
	}
	proof := witnesses[p23TranslationProofWitnessID]
	if proof.ProofFactsCompared == 0 || !proof.BoundsProofsPreserved || !proof.MissingProofRejected {
		t.Fatalf("proof witness = %#v, want proof preservation and missing proof rejection", proof)
	}
	allocation := witnesses[p23TranslationAllocationWitnessID]
	if !allocation.AllocationPlanValidated || !allocation.AllocationDriftRejected {
		t.Fatalf("allocation witness = %#v, want allocation plan validation and drift rejection", allocation)
	}
	hash := witnesses[p23TranslationHashWitnessID]
	if !strings.HasPrefix(hash.BeforeHash, "sha256:") || !strings.HasPrefix(hash.AfterHash, "sha256:") || !hash.HashesMachineCheckable || !hash.HashesDistinct {
		t.Fatalf("hash witness = %#v, want machine-checkable distinct sha256 hashes", hash)
	}

	for _, nonClaim := range []string{
		"no full formal proof is claimed",
		"no exhaustive optimizer completeness is claimed",
		"no broad memory model or alias model is claimed",
		"no broad loop theorem prover is claimed",
		"no performance claim is made",
		"runtime behavior does not change",
		"safe-program semantics do not change",
	} {
		if !p23TranslationHasString(report.NonClaims, nonClaim) {
			t.Fatalf("missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestValidateP23TranslationValidationV2RejectsFakeClaimsAndDrift(t *testing.T) {
	base, err := BuildP23TranslationValidationV2()
	if err != nil {
		t.Fatalf("BuildP23TranslationValidationV2: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*TranslationValidationV2Report)
		want   string
	}{
		{
			name: "missing row",
			mutate: func(report *TranslationValidationV2Report) {
				report.Rows = report.Rows[1:]
			},
			want: "missing row",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *TranslationValidationV2Report) {
				report.Rows[0].Evidence = []string{"TODO"}
			},
			want: "placeholder",
		},
		{
			name: "missing witness",
			mutate: func(report *TranslationValidationV2Report) {
				report.Rows[0].WitnessIDs = []string{"missing-witness"}
			},
			want: "missing witness",
		},
		{
			name: "incomplete registered pass coverage",
			mutate: func(report *TranslationValidationV2Report) {
				report.RegisteredPassCoverageComplete = false
			},
			want: "registered pass",
		},
		{
			name: "missing scalar evidence",
			mutate: func(report *TranslationValidationV2Report) {
				report.SymbolicScalarEquivalenceSamples = 0
			},
			want: "symbolic scalar",
		},
		{
			name: "missing memory evidence",
			mutate: func(report *TranslationValidationV2Report) {
				report.MemoryEquivalenceSamples = 0
			},
			want: "memory equivalence",
		},
		{
			name: "missing proof evidence",
			mutate: func(report *TranslationValidationV2Report) {
				report.BoundsProofsPreserved = false
			},
			want: "bounds proof",
		},
		{
			name: "missing allocation evidence",
			mutate: func(report *TranslationValidationV2Report) {
				report.AllocationPlanValidated = false
			},
			want: "allocation plan",
		},
		{
			name: "missing hash evidence",
			mutate: func(report *TranslationValidationV2Report) {
				report.BeforeAfterHashesMachineCheckable = false
			},
			want: "hash",
		},
		{
			name: "full formal proof claim",
			mutate: func(report *TranslationValidationV2Report) {
				report.FullFormalProofClaimed = true
			},
			want: "full formal proof",
		},
		{
			name: "performance claim",
			mutate: func(report *TranslationValidationV2Report) {
				report.PerformanceClaimed = true
			},
			want: "performance",
		},
		{
			name: "runtime behavior claim",
			mutate: func(report *TranslationValidationV2Report) {
				report.RuntimeBehaviorChanged = true
			},
			want: "runtime behavior",
		},
		{
			name: "safe semantics claim",
			mutate: func(report *TranslationValidationV2Report) {
				report.SafeSemanticsChanged = true
			},
			want: "safe semantics",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := base
			report.Rows = append([]TranslationValidationV2Row(nil), base.Rows...)
			report.Witnesses = append([]TranslationValidationV2Witness(nil), base.Witnesses...)
			report.NonClaims = append([]string(nil), base.NonClaims...)
			tc.mutate(&report)
			err := ValidateP23TranslationValidationV2(report)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateP23TranslationValidationV2 error = %v, want %q", err, tc.want)
			}
		})
	}
}

func p23AssertTranslationRow(t *testing.T, row TranslationValidationV2Row, wants []string) {
	t.Helper()
	text := strings.Join(row.Evidence, " ") + " " + strings.Join(row.Boundaries, " ")
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("row %s missing %q: %#v", row.ID, want, row)
		}
	}
}
