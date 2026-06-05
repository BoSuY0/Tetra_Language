package compiler

import (
	"strings"
	"testing"
)

func TestP24CompatibilityStabilityV1CoversMasterPlanTargets(t *testing.T) {
	report, err := BuildP24CompatibilityStabilityV1Report()
	if err != nil {
		t.Fatalf("BuildP24CompatibilityStabilityV1Report: %v", err)
	}
	if report.SchemaVersion != compatibilityStabilityV1Schema {
		t.Fatalf("schema = %q, want %q", report.SchemaVersion, compatibilityStabilityV1Schema)
	}
	if report.Scope != compatibilityStabilityV1ScopeP242 {
		t.Fatalf("scope = %q, want %q", report.Scope, compatibilityStabilityV1ScopeP242)
	}
	if err := ValidateP24CompatibilityStabilityV1Report(report); err != nil {
		t.Fatalf("ValidateP24CompatibilityStabilityV1Report: %v", err)
	}

	rows := map[CompatibilityStabilityV1ID]CompatibilityStabilityV1Row{}
	for _, row := range report.Rows {
		if row.ID == "" || row.Name == "" || row.Status == "" || len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 || len(row.WitnessIDs) == 0 {
			t.Fatalf("row missing required metadata: %#v", row)
		}
		rows[row.ID] = row
	}
	for _, id := range p24CompatibilityStabilityV1IDs() {
		if _, ok := rows[id]; !ok {
			t.Fatalf("missing row %s: %#v", id, report.Rows)
		}
	}
	p24AssertCompatibilityStabilityRow(t, rows[CompatibilityStableDiagnosticCodes], []string{"DiagnosticCodeRegistry", "TETRA0001", "TETRA2001", "validate-diagnostic"})
	p24AssertCompatibilityStabilityRow(t, rows[CompatibilityVersionedReportSchemas], []string{"tetra.translation.validation.v2", "tetra.runtime.hardening.v1", "schema_version"})
	p24AssertCompatibilityStabilityRow(t, rows[CompatibilityManifestChecks], []string{"validate-manifest", "manifest-json.v1", "FeatureRegistry", "runtime ABI"})
	p24AssertCompatibilityStabilityRow(t, rows[CompatibilityBreakingChangeMigrationGuide], []string{"breaking_requires_review", "migration guide", "no-change"})
	p24AssertCompatibilityStabilityRow(t, rows[CompatibilityDeprecationPolicy], []string{"Deprecation Policy", "replacement path", "removals wait"})

	if !report.StableDiagnosticCodesReviewed || !report.VersionedReportSchemasReviewed || !report.ManifestCompatibilityChecksReviewed || !report.BreakingChangeMigrationGuidePresent || !report.DeprecationPolicyPresent {
		t.Fatalf("compatibility/stability flags missing: %#v", report)
	}
	for _, nonClaim := range []string{
		"full backward compatibility for all future versions is not claimed",
		"diagnostic messages are not frozen",
		"automatic migration for every breaking change is not claimed",
		"manifest/runtime ABI stability beyond current validated evidence is not claimed",
		"runtime behavior does not change",
		"safe-program semantics do not change",
		"no performance claim is made",
	} {
		if !p24CompatibilityStabilityHasString(report.NonClaims, nonClaim) {
			t.Fatalf("missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestValidateP24CompatibilityStabilityV1RejectsFakeClaimsAndWeakEvidence(t *testing.T) {
	base, err := BuildP24CompatibilityStabilityV1Report()
	if err != nil {
		t.Fatalf("BuildP24CompatibilityStabilityV1Report: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*CompatibilityStabilityV1Report)
		want   string
	}{
		{
			name: "missing row",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.Rows = report.Rows[1:]
			},
			want: "missing row",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.Rows[0].Evidence = []string{"TODO"}
			},
			want: "placeholder",
		},
		{
			name: "missing witness",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.Rows[0].WitnessIDs = []string{"missing-witness"}
			},
			want: "missing witness",
		},
		{
			name: "missing diagnostics",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.StableDiagnosticCodesReviewed = false
			},
			want: "diagnostic",
		},
		{
			name: "missing manifest checks",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.ManifestCompatibilityChecksReviewed = false
			},
			want: "manifest",
		},
		{
			name: "fake full backward compatibility",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.FullBackwardCompatibilityClaimed = true
			},
			want: "full backward compatibility",
		},
		{
			name: "fake frozen diagnostics",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.FrozenDiagnosticMessagesClaimed = true
			},
			want: "diagnostic messages",
		},
		{
			name: "fake automatic migration",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.AutomaticMigrationClaimed = true
			},
			want: "automatic migration",
		},
		{
			name: "fake manifest abi stability",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.ManifestABIStabilityClaimed = true
			},
			want: "manifest/runtime ABI",
		},
		{
			name: "breaking change without migration",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.BreakingChangesWithoutMigrationClaimed = true
			},
			want: "breaking change",
		},
		{
			name: "removal without deprecation",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.RemovalWithoutDeprecationClaimed = true
			},
			want: "deprecation",
		},
		{
			name: "runtime behavior change",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.RuntimeBehaviorChanged = true
			},
			want: "runtime behavior",
		},
		{
			name: "safe semantics change",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.SafeSemanticsChanged = true
			},
			want: "safe semantics",
		},
		{
			name: "performance claim",
			mutate: func(report *CompatibilityStabilityV1Report) {
				report.PerformanceClaimed = true
			},
			want: "performance",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := base
			report.Rows = append([]CompatibilityStabilityV1Row(nil), base.Rows...)
			report.Witnesses = append([]CompatibilityStabilityV1Witness(nil), base.Witnesses...)
			report.NonClaims = append([]string(nil), base.NonClaims...)
			tc.mutate(&report)
			err := ValidateP24CompatibilityStabilityV1Report(report)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateP24CompatibilityStabilityV1Report error = %v, want %q", err, tc.want)
			}
		})
	}
}

func p24AssertCompatibilityStabilityRow(t *testing.T, row CompatibilityStabilityV1Row, wants []string) {
	t.Helper()
	text := strings.Join(row.Evidence, " ") + " " + strings.Join(row.Boundaries, " ")
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("row %s missing %q: %#v", row.ID, want, row)
		}
	}
}
