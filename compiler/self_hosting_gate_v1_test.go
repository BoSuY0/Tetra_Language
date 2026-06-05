package compiler

import (
	"strings"
	"testing"
)

func TestP23SelfHostingGateV1BlocksPromotionUntilBootstrapEvidenceExists(t *testing.T) {
	report, err := BuildP23SelfHostingGateV1Report()
	if err != nil {
		t.Fatalf("BuildP23SelfHostingGateV1Report: %v", err)
	}
	if report.SchemaVersion != selfHostingGateV1Schema {
		t.Fatalf("schema = %q, want %q", report.SchemaVersion, selfHostingGateV1Schema)
	}
	if report.Scope != selfHostingGateV1ScopeP233 {
		t.Fatalf("scope = %q, want %q", report.Scope, selfHostingGateV1ScopeP233)
	}
	if err := ValidateP23SelfHostingGateV1Report(report); err != nil {
		t.Fatalf("ValidateP23SelfHostingGateV1Report: %v", err)
	}

	rows := map[SelfHostingGateV1ID]SelfHostingGateV1Row{}
	for _, row := range report.Rows {
		if row.ID == "" || row.Name == "" || row.Status == "" || len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 || len(row.WitnessIDs) == 0 {
			t.Fatalf("row missing required metadata: %#v", row)
		}
		rows[row.ID] = row
	}
	for _, id := range p23SelfHostingGateV1IDs() {
		if _, ok := rows[id]; !ok {
			t.Fatalf("missing row %s: %#v", id, report.Rows)
		}
	}
	p23AssertSelfHostingGateRow(t, rows[SelfHostingGateSubsetDefinition], []string{"verified subset", "not self-hosting"})
	p23AssertSelfHostingGateRow(t, rows[SelfHostingGateSmallComponentCompile], []string{"small compiler component", "blocked"})
	p23AssertSelfHostingGateRow(t, rows[SelfHostingGateOutputComparison], []string{"Go compiler output", "Tetra-compiled output", "blocked"})
	p23AssertSelfHostingGateRow(t, rows[SelfHostingGateRegisterBackend], []string{"register backend", "CheckBackendMatrix", "Machine IR"})
	p23AssertSelfHostingGateRow(t, rows[SelfHostingGateOptimizerValidation], []string{"optimizer validation", "translation validation v2"})
	p23AssertSelfHostingGateRow(t, rows[SelfHostingGateAllocatorRuntime], []string{"allocator/runtime", "RuntimeAllocationContracts"})
	p23AssertSelfHostingGateRow(t, rows[SelfHostingGateStdlibSufficiency], []string{"stdlib", "RegionAwareStdlibCoverage"})
	p23AssertSelfHostingGateRow(t, rows[SelfHostingGateDeterministicBootstrap], []string{"deterministic bootstrap", "blocked"})
	p23AssertSelfHostingGateRow(t, rows[SelfHostingGateCrossPlatformBootstrap], []string{"cross-platform bootstrap", "blocked"})
	p23AssertSelfHostingGateRow(t, rows[SelfHostingGateNoSelfHostingClaim], []string{"SelfHostingClaimed=false", "GateDecision.Allowed=false"})

	witnesses := map[string]SelfHostingGateV1Witness{}
	for _, witness := range report.Witnesses {
		witnesses[witness.ID] = witness
	}
	subset := witnesses[p23SelfHostingSubsetWitnessID]
	if !subset.CompilerSubsetDefined || !strings.Contains(subset.SubsetName, "verified") {
		t.Fatalf("subset witness = %#v, want defined verified subset boundary", subset)
	}
	backend := witnesses[p23SelfHostingRegisterBackendWitnessID]
	if !backend.RegisterBackendEvidencePresent || backend.BackendMatrixLanes < 5 {
		t.Fatalf("backend witness = %#v, want register backend matrix evidence", backend)
	}
	optimizer := witnesses[p23SelfHostingOptimizerWitnessID]
	if !optimizer.OptimizerValidationEvidencePresent || optimizer.TranslationValidationRows < 6 {
		t.Fatalf("optimizer witness = %#v, want translation validation v2 evidence", optimizer)
	}
	allocator := witnesses[p23SelfHostingAllocatorRuntimeWitnessID]
	if !allocator.AllocatorRuntimeEvidencePresent || allocator.RuntimeAllocationContracts < 5 || !allocator.PerCoreSmallHeapEvidencePresent {
		t.Fatalf("allocator/runtime witness = %#v, want allocation contract and small heap evidence", allocator)
	}
	stdlib := witnesses[p23SelfHostingStdlibWitnessID]
	if !stdlib.StdlibEvidencePresent || stdlib.StdlibRows < 10 {
		t.Fatalf("stdlib witness = %#v, want region-aware stdlib evidence", stdlib)
	}

	if !report.CompilerSubsetDefined || report.SmallCompilerComponentCompiled || report.GoVsTetraOutputCompared || report.DeterministicBootstrapChain || report.CrossPlatformBootstrapStory {
		t.Fatalf("self-host progress flags = subset:%v component:%v compare:%v bootstrap:%v cross:%v",
			report.CompilerSubsetDefined,
			report.SmallCompilerComponentCompiled,
			report.GoVsTetraOutputCompared,
			report.DeterministicBootstrapChain,
			report.CrossPlatformBootstrapStory)
	}
	if !report.RegisterBackendEvidencePresent || !report.OptimizerValidationEvidencePresent || !report.AllocatorRuntimeEvidencePresent || !report.StdlibEvidencePresent {
		t.Fatalf("existing evidence flags missing: %#v", report)
	}
	if report.SelfHostingClaimed || report.GateDecision.Allowed {
		t.Fatalf("P23.3 must not promote self-hosting: %#v", report.GateDecision)
	}
	for _, missing := range []string{
		"small_compiler_component_compiled",
		"go_vs_tetra_output_compared",
		"deterministic_bootstrap_chain",
		"cross_platform_bootstrap_story",
	} {
		if !report.GateDecision.Missing(missing) {
			t.Fatalf("gate decision missing blocker %q: %#v", missing, report.GateDecision)
		}
	}
	for _, nonClaim := range []string{
		"Tetra is not self-hosting",
		"no Tetra compiler component is claimed to compile itself yet",
		"no deterministic bootstrap chain is claimed yet",
		"no cross-platform bootstrap story is claimed yet",
		"runtime behavior does not change",
		"safe-program semantics do not change",
		"no performance claim is made",
	} {
		if !p23SelfHostingGateHasString(report.NonClaims, nonClaim) {
			t.Fatalf("missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestValidateP23SelfHostingGateV1RejectsFakeClaimsAndWeakEvidence(t *testing.T) {
	base, err := BuildP23SelfHostingGateV1Report()
	if err != nil {
		t.Fatalf("BuildP23SelfHostingGateV1Report: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*SelfHostingGateV1Report)
		want   string
	}{
		{
			name: "missing row",
			mutate: func(report *SelfHostingGateV1Report) {
				report.Rows = report.Rows[1:]
			},
			want: "missing row",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *SelfHostingGateV1Report) {
				report.Rows[0].Evidence = []string{"TODO"}
			},
			want: "placeholder",
		},
		{
			name: "missing witness",
			mutate: func(report *SelfHostingGateV1Report) {
				report.Rows[0].WitnessIDs = []string{"missing-witness"}
			},
			want: "missing witness",
		},
		{
			name: "self hosting claim",
			mutate: func(report *SelfHostingGateV1Report) {
				report.SelfHostingClaimed = true
			},
			want: "self-hosting claim",
		},
		{
			name: "gate allowed",
			mutate: func(report *SelfHostingGateV1Report) {
				report.GateDecision.Allowed = true
			},
			want: "gate decision",
		},
		{
			name: "gate missing blockers omitted",
			mutate: func(report *SelfHostingGateV1Report) {
				report.GateDecision.MissingEvidence = nil
			},
			want: "gate decision",
		},
		{
			name: "compiler subset missing",
			mutate: func(report *SelfHostingGateV1Report) {
				report.CompilerSubsetDefined = false
			},
			want: "compiler subset",
		},
		{
			name: "register backend evidence missing",
			mutate: func(report *SelfHostingGateV1Report) {
				report.RegisterBackendEvidencePresent = false
			},
			want: "register backend",
		},
		{
			name: "optimizer evidence missing",
			mutate: func(report *SelfHostingGateV1Report) {
				report.OptimizerValidationEvidencePresent = false
			},
			want: "optimizer",
		},
		{
			name: "allocator runtime evidence missing",
			mutate: func(report *SelfHostingGateV1Report) {
				report.AllocatorRuntimeEvidencePresent = false
			},
			want: "allocator/runtime",
		},
		{
			name: "stdlib evidence missing",
			mutate: func(report *SelfHostingGateV1Report) {
				report.StdlibEvidencePresent = false
			},
			want: "stdlib",
		},
		{
			name: "fake compiler component",
			mutate: func(report *SelfHostingGateV1Report) {
				report.SmallCompilerComponentCompiled = true
			},
			want: "small compiler component",
		},
		{
			name: "fake output comparison",
			mutate: func(report *SelfHostingGateV1Report) {
				report.GoVsTetraOutputCompared = true
			},
			want: "output comparison",
		},
		{
			name: "fake deterministic bootstrap",
			mutate: func(report *SelfHostingGateV1Report) {
				report.DeterministicBootstrapChain = true
			},
			want: "deterministic bootstrap",
		},
		{
			name: "fake cross platform bootstrap",
			mutate: func(report *SelfHostingGateV1Report) {
				report.CrossPlatformBootstrapStory = true
			},
			want: "cross-platform bootstrap",
		},
		{
			name: "runtime behavior change",
			mutate: func(report *SelfHostingGateV1Report) {
				report.RuntimeBehaviorChanged = true
			},
			want: "runtime behavior",
		},
		{
			name: "safe semantics change",
			mutate: func(report *SelfHostingGateV1Report) {
				report.SafeSemanticsChanged = true
			},
			want: "safe semantics",
		},
		{
			name: "performance claim",
			mutate: func(report *SelfHostingGateV1Report) {
				report.PerformanceClaimed = true
			},
			want: "performance",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := base
			report.Rows = append([]SelfHostingGateV1Row(nil), base.Rows...)
			report.Witnesses = append([]SelfHostingGateV1Witness(nil), base.Witnesses...)
			report.NonClaims = append([]string(nil), base.NonClaims...)
			report.GateDecision.MissingEvidence = append([]string(nil), base.GateDecision.MissingEvidence...)
			tc.mutate(&report)
			err := ValidateP23SelfHostingGateV1Report(report)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateP23SelfHostingGateV1Report error = %v, want %q", err, tc.want)
			}
		})
	}
}

func p23AssertSelfHostingGateRow(t *testing.T, row SelfHostingGateV1Row, wants []string) {
	t.Helper()
	text := strings.Join(row.Evidence, " ") + " " + strings.Join(row.Boundaries, " ")
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("row %s missing %q: %#v", row.ID, want, row)
		}
	}
}
