package compiler

import (
	"strings"
	"testing"
)

func TestP24RuntimeHardeningV1CoversMasterPlanTargets(t *testing.T) {
	report, err := BuildP24RuntimeHardeningV1Report()
	if err != nil {
		t.Fatalf("BuildP24RuntimeHardeningV1Report: %v", err)
	}
	if report.SchemaVersion != runtimeHardeningV1Schema {
		t.Fatalf("schema = %q, want %q", report.SchemaVersion, runtimeHardeningV1Schema)
	}
	if report.Scope != runtimeHardeningV1ScopeP241 {
		t.Fatalf("scope = %q, want %q", report.Scope, runtimeHardeningV1ScopeP241)
	}
	if err := ValidateP24RuntimeHardeningV1Report(report); err != nil {
		t.Fatalf("ValidateP24RuntimeHardeningV1Report: %v", err)
	}

	rows := map[RuntimeHardeningV1ID]RuntimeHardeningV1Row{}
	for _, row := range report.Rows {
		if row.ID == "" || row.Name == "" || row.Status == "" || len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 || len(row.WitnessIDs) == 0 {
			t.Fatalf("row missing required metadata: %#v", row)
		}
		rows[row.ID] = row
	}
	for _, id := range p24RuntimeHardeningV1IDs() {
		if _, ok := rows[id]; !ok {
			t.Fatalf("missing row %s: %#v", id, report.Rows)
		}
	}
	p24AssertRuntimeHardeningRow(t, rows[RuntimeHardeningDeterministicTraps], []string{"trap_or_stable_status", "emitWasmTrapIf", "tetra panic"})
	p24AssertRuntimeHardeningRow(t, rows[RuntimeHardeningOOMPolicy], []string{"AllocationFailureTrapOrStatus", "reject before allocator", "stable trap/status"})
	p24AssertRuntimeHardeningRow(t, rows[RuntimeHardeningStackOverflowGuard], []string{"stack-depth consistency", "guard-page", "recursion-depth"})
	p24AssertRuntimeHardeningRow(t, rows[RuntimeHardeningIntegerOverflowSemantics], []string{"checkedNegI32", "foldConstBinaryI32", "byte-size overflow"})
	p24AssertRuntimeHardeningRow(t, rows[RuntimeHardeningAllocatorCorruptionInstrumentation], []string{"bounds_header", "stale or double free", "PerCoreSmallHeapAllocator"})
	p24AssertRuntimeHardeningRow(t, rows[RuntimeHardeningRegionUseAfterFreeInstrumentation], []string{"AllocationDebugDoubleFree", "AllocationDebugUseAfterFree", "region.temp"})
	p24AssertRuntimeHardeningRow(t, rows[RuntimeHardeningActorMailboxOverflowPolicy], []string{"ErrMailboxFull", "blocking_recv_yield", "message pool exhaustion returns checked -1", "drained message pool entries are reclaimed"})
	p24AssertRuntimeHardeningRow(t, rows[RuntimeHardeningNetworkParserLimits], []string{"ErrHeaderTooLarge", "ErrBodyTooLarge", "ErrFrameTooLarge", "ErrMalformedFrame"})

	if !report.DeterministicTrapsReviewed || !report.OOMPolicyReviewed || !report.StackOverflowGuardReviewed || !report.IntegerOverflowSemanticsAudited || !report.AllocatorCorruptionInstrumentationReviewed || !report.RegionDoubleFreeUseAfterFreeReviewed || !report.ActorMailboxOverflowPolicyReviewed || !report.NetworkParserLimitsReviewed {
		t.Fatalf("runtime hardening flags missing: %#v", report)
	}
	for _, nonClaim := range []string{
		"full runtime-hardening proof is not claimed",
		"full stack-overflow protection is not claimed",
		"OOM recovery guarantee is not claimed",
		"full allocator-corruption detection proof is not claimed",
		"production actor-mailbox promotion is not claimed",
		"runtime behavior does not change",
		"safe-program semantics do not change",
		"no performance claim is made",
	} {
		if !p24RuntimeHardeningHasString(report.NonClaims, nonClaim) {
			t.Fatalf("missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestValidateP24RuntimeHardeningV1RejectsFakeClaimsAndWeakEvidence(t *testing.T) {
	base, err := BuildP24RuntimeHardeningV1Report()
	if err != nil {
		t.Fatalf("BuildP24RuntimeHardeningV1Report: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*RuntimeHardeningV1Report)
		want   string
	}{
		{
			name: "missing row",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.Rows = report.Rows[1:]
			},
			want: "missing row",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.Rows[0].Evidence = []string{"TODO"}
			},
			want: "placeholder",
		},
		{
			name: "missing witness",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.Rows[0].WitnessIDs = []string{"missing-witness"}
			},
			want: "missing witness",
		},
		{
			name: "missing deterministic traps",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.DeterministicTrapsReviewed = false
			},
			want: "deterministic traps",
		},
		{
			name: "missing actor mailbox policy",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.ActorMailboxOverflowPolicyReviewed = false
			},
			want: "actor mailbox",
		},
		{
			name: "fake full hardening",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.FullRuntimeHardeningClaimed = true
			},
			want: "full runtime-hardening",
		},
		{
			name: "fake stack overflow protection",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.FullStackOverflowProtectionClaimed = true
			},
			want: "stack-overflow protection",
		},
		{
			name: "fake OOM recovery",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.FullOOMRecoveryClaimed = true
			},
			want: "OOM recovery",
		},
		{
			name: "fake allocator corruption detection",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.FullAllocatorCorruptionDetectionClaimed = true
			},
			want: "allocator-corruption",
		},
		{
			name: "fake production actor mailbox",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.ProductionActorMailboxClaimed = true
			},
			want: "production actor-mailbox",
		},
		{
			name: "runtime behavior change",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.RuntimeBehaviorChanged = true
			},
			want: "runtime behavior",
		},
		{
			name: "safe semantics change",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.SafeSemanticsChanged = true
			},
			want: "safe semantics",
		},
		{
			name: "performance claim",
			mutate: func(report *RuntimeHardeningV1Report) {
				report.PerformanceClaimed = true
			},
			want: "performance",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := base
			report.Rows = append([]RuntimeHardeningV1Row(nil), base.Rows...)
			report.Witnesses = append([]RuntimeHardeningV1Witness(nil), base.Witnesses...)
			report.NonClaims = append([]string(nil), base.NonClaims...)
			tc.mutate(&report)
			err := ValidateP24RuntimeHardeningV1Report(report)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateP24RuntimeHardeningV1Report error = %v, want %q", err, tc.want)
			}
		})
	}
}

func p24AssertRuntimeHardeningRow(t *testing.T, row RuntimeHardeningV1Row, wants []string) {
	t.Helper()
	text := strings.Join(row.Evidence, " ") + " " + strings.Join(row.Boundaries, " ")
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("row %s missing %q: %#v", row.ID, want, row)
		}
	}
}
