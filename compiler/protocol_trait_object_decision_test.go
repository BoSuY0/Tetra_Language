package compiler

import (
	"strings"
	"testing"
)

func TestP22ProtocolTraitObjectDecisionKeepsStaticFastPath(t *testing.T) {
	report, err := BuildP22ProtocolTraitObjectDecision()
	if err != nil {
		t.Fatalf("BuildP22ProtocolTraitObjectDecision: %v", err)
	}
	if report.SchemaVersion != protocolTraitObjectDecisionSchemaV1 {
		t.Fatalf("schema = %q, want %q", report.SchemaVersion, protocolTraitObjectDecisionSchemaV1)
	}
	if report.Scope != protocolTraitObjectDecisionScopeP222 {
		t.Fatalf("scope = %q, want %q", report.Scope, protocolTraitObjectDecisionScopeP222)
	}
	if report.Decision != protocolTraitObjectDecisionKeepStaticOnly {
		t.Fatalf("decision = %q, want %q", report.Decision, protocolTraitObjectDecisionKeepStaticOnly)
	}
	if err := ValidateP22ProtocolTraitObjectDecision(report); err != nil {
		t.Fatalf("ValidateP22ProtocolTraitObjectDecision: %v", err)
	}

	rows := map[ProtocolTraitObjectDecisionID]ProtocolTraitObjectDecisionRow{}
	for _, row := range report.Rows {
		if row.ID == "" || row.Name == "" || row.Status == "" || row.Decision == "" || len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 || len(row.WitnessIDs) == 0 {
			t.Fatalf("row missing required metadata: %#v", row)
		}
		rows[row.ID] = row
	}
	for _, id := range p22ProtocolTraitObjectDecisionIDs() {
		if _, ok := rows[id]; !ok {
			t.Fatalf("missing row %s: %#v", id, report.Rows)
		}
	}
	p22AssertProtocolTraitRow(t, rows[ProtocolTraitStaticConformanceFastPath], []string{"static conformance", "compareProtocolRequirement", "known direct IRCall", "Vec2.draw"})
	p22AssertProtocolTraitRow(t, rows[ProtocolTraitStaticProtocolBoundGenerics], []string{"protocol-bound generics", "monomorphization", "id__T_Vec2", "no runtime generic values"})
	p22AssertProtocolTraitRow(t, rows[ProtocolTraitRuntimeExistentialDecision], []string{"keep_static_conformance_only", "unknown type 'Drawable'", "runtime protocol values remain unsupported"})
	p22AssertProtocolTraitRow(t, rows[ProtocolTraitExplicitDynamicDispatchGate], []string{"dynamic dispatch must be explicit", "report-visible", "not promoted"})
	p22AssertProtocolTraitRow(t, rows[ProtocolTraitSpecializationStaticAbstraction], []string{"P17.2", "P21.2", "known direct Stack IR function symbol", "Machine IR contains no OpCall"})
	p22AssertProtocolTraitRow(t, rows[ProtocolTraitWitnessTableBoundary], []string{"witness tables", "not emitted", "future ABI evidence"})
	p22AssertProtocolTraitRow(t, rows[ProtocolTraitTraitObjectBoundary], []string{"trait objects", "not promoted", "runtime existential"})
	p22AssertProtocolTraitRow(t, rows[ProtocolTraitRegistryDocsAlignment], []string{"FeatureRegistry", "language.protocol-conformance-mvp", "language.protocol-bound-generics-static"})

	witnesses := map[string]ProtocolTraitObjectWitness{}
	for _, witness := range report.Witnesses {
		witnesses[witness.ID] = witness
	}
	static := witnesses[protocolTraitStaticConformanceWitnessID]
	if static.ID == "" {
		t.Fatalf("missing static conformance witness: %#v", report.Witnesses)
	}
	if static.ProtocolCount != 1 || static.ImplCount != 1 || !static.HasStaticMethodSig || static.DirectCallTarget != "Vec2.draw" || !static.LoweredDirectCall {
		t.Fatalf("static conformance witness = %#v, want one protocol/impl and direct Vec2.draw IRCall", static)
	}
	generic := witnesses[protocolTraitProtocolBoundGenericWitnessID]
	if generic.ID == "" {
		t.Fatalf("missing protocol-bound generic witness: %#v", report.Witnesses)
	}
	if generic.MonomorphizedSig != "id__T_Vec2" || !generic.MonomorphizedSigConcrete || !generic.LoweredDirectCall {
		t.Fatalf("protocol-bound generic witness = %#v, want concrete id__T_Vec2 direct call", generic)
	}
	boundary := witnesses[protocolTraitRuntimeBoundaryWitnessID]
	if boundary.ID == "" {
		t.Fatalf("missing runtime boundary witness: %#v", report.Witnesses)
	}
	if !strings.Contains(boundary.RuntimeProtocolValueDiagnostic, "unknown type 'Drawable'") || !strings.Contains(boundary.GenericRequirementCallDiagnostic, "not supported in this MVP") {
		t.Fatalf("runtime boundary witness = %#v, want runtime value and generic-bound call diagnostics", boundary)
	}
	specialization := witnesses[protocolTraitSpecializationWitnessID]
	if specialization.ID == "" {
		t.Fatalf("missing specialization witness: %#v", report.Witnesses)
	}
	if specialization.InliningSchema != "tetra.optimizer.inlining_specialization.v1" || specialization.MachineSchema != "tetra.optimizer.specialization_machine_code.v1" || !specialization.KnownDirectSymbolEvidence || !specialization.SpecializationNoDynamicDispatch || !specialization.MachineNoOpCall {
		t.Fatalf("specialization witness = %#v, want P17/P21 known-direct no-dynamic-dispatch evidence", specialization)
	}

	for _, nonClaim := range []string{
		"runtime protocol values are not promoted",
		"trait objects are not promoted",
		"witness tables are not promoted",
		"dynamic dispatch is not promoted",
		"conformance-table lookup is not promoted",
		"runtime existential ABI is not designed in this slice",
		"broad protocol specialization is not claimed",
		"performance is not claimed",
		"runtime behavior does not change",
		"safe-program semantics do not change",
	} {
		if !p22ProtocolTraitHasString(report.NonClaims, nonClaim) {
			t.Fatalf("missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestValidateP22ProtocolTraitObjectDecisionRejectsFakeClaimsAndDrift(t *testing.T) {
	base, err := BuildP22ProtocolTraitObjectDecision()
	if err != nil {
		t.Fatalf("BuildP22ProtocolTraitObjectDecision: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*ProtocolTraitObjectDecisionReport)
		want   string
	}{
		{
			name: "wrong decision",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.Decision = "promote_runtime_existentials"
			},
			want: "keep_static_conformance_only",
		},
		{
			name: "missing row",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.Rows = report.Rows[1:]
			},
			want: "missing row",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.Rows[0].Evidence = []string{"TODO"}
			},
			want: "placeholder",
		},
		{
			name: "missing witness reference",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.Rows[0].WitnessIDs = []string{"missing-witness"}
			},
			want: "missing witness",
		},
		{
			name: "bad static witness",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.Witnesses[0].LoweredDirectCall = false
			},
			want: "static conformance witness",
		},
		{
			name: "bad runtime boundary witness",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.Witnesses[2].RuntimeProtocolValueDiagnostic = ""
			},
			want: "runtime boundary witness",
		},
		{
			name: "bad specialization witness",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.Witnesses[3].MachineNoOpCall = false
			},
			want: "specialization witness",
		},
		{
			name: "runtime existential claim",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.RuntimeExistentialsPromoted = true
			},
			want: "runtime existential",
		},
		{
			name: "trait object claim",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.TraitObjectsPromoted = true
			},
			want: "trait object",
		},
		{
			name: "witness table claim",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.WitnessTablesPromoted = true
			},
			want: "witness table",
		},
		{
			name: "dynamic dispatch claim",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.DynamicDispatchPromoted = true
			},
			want: "dynamic dispatch",
		},
		{
			name: "conformance table claim",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.ConformanceTableLookupPromoted = true
			},
			want: "conformance-table",
		},
		{
			name: "runtime protocol value claim",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.RuntimeProtocolValuesPromoted = true
			},
			want: "runtime protocol value",
		},
		{
			name: "broad specialization claim",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.BroadSpecializationClaimed = true
			},
			want: "broad specialization",
		},
		{
			name: "performance claim",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.PerformanceClaimed = true
			},
			want: "performance",
		},
		{
			name: "runtime behavior claim",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.RuntimeBehaviorChanged = true
			},
			want: "runtime behavior",
		},
		{
			name: "safe semantics claim",
			mutate: func(report *ProtocolTraitObjectDecisionReport) {
				report.SafeSemanticsChanged = true
			},
			want: "safe-program semantics",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := cloneProtocolTraitObjectDecision(base)
			tc.mutate(&report)
			err := ValidateP22ProtocolTraitObjectDecision(report)
			if err == nil {
				t.Fatalf("ValidateP22ProtocolTraitObjectDecision accepted fake report: %#v", report)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func p22AssertProtocolTraitRow(t *testing.T, row ProtocolTraitObjectDecisionRow, wants []string) {
	t.Helper()
	combined := row.Name + " " + row.Status + " " + row.Decision + " " + strings.Join(row.Evidence, " ") + " " + strings.Join(row.Tests, " ") + " " + strings.Join(row.Boundaries, " ")
	for _, want := range wants {
		if !strings.Contains(combined, want) {
			t.Fatalf("row %s missing %q: %#v", row.ID, want, row)
		}
	}
}

func cloneProtocolTraitObjectDecision(report ProtocolTraitObjectDecisionReport) ProtocolTraitObjectDecisionReport {
	report.Rows = append([]ProtocolTraitObjectDecisionRow{}, report.Rows...)
	for i := range report.Rows {
		report.Rows[i].Evidence = append([]string{}, report.Rows[i].Evidence...)
		report.Rows[i].Tests = append([]string{}, report.Rows[i].Tests...)
		report.Rows[i].Boundaries = append([]string{}, report.Rows[i].Boundaries...)
		report.Rows[i].WitnessIDs = append([]string{}, report.Rows[i].WitnessIDs...)
	}
	report.Witnesses = append([]ProtocolTraitObjectWitness{}, report.Witnesses...)
	report.NonClaims = append([]string{}, report.NonClaims...)
	return report
}

func p22ProtocolTraitHasString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
