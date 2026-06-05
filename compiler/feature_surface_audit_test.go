package compiler

import (
	"strings"
	"testing"
)

func TestP22FeatureSurfaceAuditCoversMasterPlanCategoriesAndRegistryBoundaries(t *testing.T) {
	report := BuildP22FeatureSurfaceAudit()
	if report.SchemaVersion != featureSurfaceAuditSchemaV1 {
		t.Fatalf("feature surface audit schema = %q, want %q", report.SchemaVersion, featureSurfaceAuditSchemaV1)
	}
	if report.Scope != featureSurfaceAuditScopeP220 {
		t.Fatalf("feature surface audit scope = %q, want %q", report.Scope, featureSurfaceAuditScopeP220)
	}
	if err := ValidateP22FeatureSurfaceAudit(report); err != nil {
		t.Fatalf("ValidateP22FeatureSurfaceAudit: %v", err)
	}

	rows := map[FeatureSurfaceAuditCategory]FeatureSurfaceAuditRow{}
	for _, row := range report.Rows {
		if row.Category == "" || row.Decision == "" || len(row.Evidence) == 0 || len(row.Boundaries) == 0 || len(row.RequiredPromotionEvidence) == 0 {
			t.Fatalf("P22.0 row missing required metadata: %#v", row)
		}
		rows[row.Category] = row
	}
	for _, category := range p22FeatureSurfaceAuditCategories() {
		if _, ok := rows[category]; !ok {
			t.Fatalf("P22.0 audit missing category %s: %#v", category, report.Rows)
		}
	}

	p22AssertFeatureSurfaceRow(t, rows[FeatureSurfaceFirstClassCallables],
		[]string{"language.callable-mvp", "language.callable-level1", "language.callable-level2", "language.full-first-class-callables"},
		[]string{"fixed 4-slot callable handle", "mutable by-reference capture", "thread-boundary callable escape", "P22.1"})
	p22AssertFeatureSurfaceRow(t, rows[FeatureSurfaceClosures],
		[]string{"language.callable-level2", "language.full-first-class-callables"},
		[]string{"safe by-value captures", "pointer/resource capture", "generic closure", "same-branch evidence"})
	p22AssertFeatureSurfaceRow(t, rows[FeatureSurfaceProtocolsTraitObjects],
		[]string{"language.protocol-conformance-mvp", "language.protocol-bound-generics-static"},
		[]string{"static conformance", "no witness tables", "trait objects", "runtime protocol values", "P22.2"})
	p22AssertFeatureSurfaceRow(t, rows[FeatureSurfaceRuntimeGenerics],
		[]string{"language.generics-mvp", "language.protocol-bound-generics-static"},
		[]string{"statically monomorphized", "runtime generic values", "explicit type arguments", "generic structs", "higher-ranked generics"})
	p22AssertFeatureSurfaceRow(t, rows[FeatureSurfaceAdvancedEnumsPatternMatching],
		[]string{"language.enum-payload-match"},
		[]string{"positional enum payload", "nested destructuring patterns", "guard expansion", "richer payload algebra"})
	p22AssertFeatureSurfaceRow(t, rows[FeatureSurfaceAsyncTypedErrors],
		[]string{"language.task-handles-mvp", "language.resource-lifetime-mvp"},
		[]string{"try await", "typed-error", "await try", "cancellation"})
	p22AssertFeatureSurfaceRow(t, rows[FeatureSurfaceStructuredConcurrency],
		[]string{"actors.task-transfer-safety", "language.task-handles-mvp", "actors.distributed-runtime"},
		[]string{"conservative local MVP", "full cancellation", "full race-safety proof", "broader structured-concurrency guarantees"})
	p22AssertFeatureSurfaceRow(t, rows[FeatureSurfaceModulesPackages],
		[]string{"language.globals-properties-capsule-mvp", "eco.local-package-lifecycle"},
		[]string{"local package lifecycle", "capsule metadata", "distributed EcoNet"})
	p22AssertFeatureSurfaceRow(t, rows[FeatureSurfaceMacrosMetaprogramming],
		nil,
		[]string{"no current macro/metaprogramming feature", "post-v1", "same-branch evidence"})
	p22AssertFeatureSurfaceRow(t, rows[FeatureSurfaceUISurface],
		[]string{"ui.metadata-v1", "ui.surface-core", "ui.surface-linux-x64", "ui.surface-web-wasm", "ui.native-runtime", "ui.platform-runtime", "ui.surface-macos-x64", "ui.surface-windows-x64", "ui.surface-wasm32-wasi"},
		[]string{"Linux-x64", "wasm32-web", "macOS", "Windows", "cross-platform"})
	p22AssertFeatureSurfaceRow(t, rows[FeatureSurfaceEcoCapsules],
		[]string{"language.globals-properties-capsule-mvp", "eco.local-package-lifecycle", "eco.distributed-network"},
		[]string{"local Eco", "proof-carrying capsules", "distributed EcoNet", "post-v1"})

	for _, nonClaim := range []string{
		"no full v1 language guarantee is claimed",
		"no runtime generic values are claimed",
		"no trait objects or runtime protocol values are claimed",
		"no macro/metaprogramming system is claimed",
		"no full structured concurrency guarantee is claimed",
		"no cross-platform production UI runtime is claimed",
		"no distributed EcoNet or proof-carrying capsule promotion is claimed",
		"no performance claim is made",
		"safe-program semantics do not change",
	} {
		if !p22FeatureSurfaceHasString(report.NonClaims, nonClaim) {
			t.Fatalf("P22.0 audit missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestValidateP22FeatureSurfaceAuditRejectsFakePromotionAndDrift(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*FeatureSurfaceAuditReport)
		want   string
	}{
		{
			name: "report level promotion without same branch evidence",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.PromotedWithoutSameBranchEvidence = true
			},
			want: "same-branch",
		},
		{
			name: "row promotion without same branch evidence",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.Rows[0].PromotedInThisAudit = true
				report.Rows[0].SameBranchEvidence = false
			},
			want: "same-branch",
		},
		{
			name: "full v1 claim",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.FullV1GuaranteesClaimed = true
			},
			want: "full v1",
		},
		{
			name: "runtime generic values claim",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.RuntimeGenericValuesClaimed = true
			},
			want: "runtime generic",
		},
		{
			name: "trait objects claim",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.TraitObjectsClaimed = true
			},
			want: "trait object",
		},
		{
			name: "macro system claim",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.MacroSystemClaimed = true
			},
			want: "macro",
		},
		{
			name: "structured concurrency claim",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.StructuredConcurrencyClaimed = true
			},
			want: "structured concurrency",
		},
		{
			name: "cross platform UI runtime claim",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.CrossPlatformUIRuntimeClaimed = true
			},
			want: "cross-platform",
		},
		{
			name: "distributed Eco claim",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.DistributedEcoClaimed = true
			},
			want: "distributed Eco",
		},
		{
			name: "proof carrying capsules claim",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.ProofCarryingCapsulesClaimed = true
			},
			want: "proof-carrying",
		},
		{
			name: "performance claim",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.PerformanceClaimed = true
			},
			want: "performance",
		},
		{
			name: "safe semantics change",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.SafeSemanticsChanged = true
			},
			want: "safe-program semantics",
		},
		{
			name: "missing category",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.Rows = report.Rows[1:]
			},
			want: "missing category",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.Rows[0].Evidence = []string{"TODO fill this in"}
			},
			want: "placeholder",
		},
		{
			name: "unknown feature ID",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.Rows[0].FeatureIDs = append(report.Rows[0].FeatureIDs, "language.fake-runtime-generics")
				report.Rows[0].RegistryStatuses["language.fake-runtime-generics"] = FeatureStatusCurrent
			},
			want: "unknown feature",
		},
		{
			name: "registry status drift",
			mutate: func(report *FeatureSurfaceAuditReport) {
				report.Rows[0].RegistryStatuses[report.Rows[0].FeatureIDs[0]] = FeatureStatusPostV1
			},
			want: "registry status drift",
		},
		{
			name: "macro row invents current feature",
			mutate: func(report *FeatureSurfaceAuditReport) {
				for i := range report.Rows {
					if report.Rows[i].Category == FeatureSurfaceMacrosMetaprogramming {
						report.Rows[i].FeatureIDs = []string{"language.macro-system"}
						report.Rows[i].RegistryStatuses = map[string]FeatureStatus{"language.macro-system": FeatureStatusCurrent}
						return
					}
				}
			},
			want: "unknown feature",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := cloneFeatureSurfaceAuditReport(BuildP22FeatureSurfaceAudit())
			tc.mutate(&report)
			err := ValidateP22FeatureSurfaceAudit(report)
			if err == nil {
				t.Fatalf("ValidateP22FeatureSurfaceAudit accepted fake report: %#v", report)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func p22AssertFeatureSurfaceRow(t *testing.T, row FeatureSurfaceAuditRow, featureIDs []string, wants []string) {
	t.Helper()
	for _, id := range featureIDs {
		if !p22FeatureSurfaceHasString(row.FeatureIDs, id) {
			t.Fatalf("row %s missing feature id %s: %#v", row.Category, id, row)
		}
		if row.RegistryStatuses[id] == "" {
			t.Fatalf("row %s missing registry status for %s: %#v", row.Category, id, row.RegistryStatuses)
		}
	}
	combined := row.Name + " " + row.Decision + " " + strings.Join(row.Evidence, " ") + " " + strings.Join(row.Boundaries, " ") + " " + strings.Join(row.RequiredPromotionEvidence, " ")
	for _, want := range wants {
		if !strings.Contains(combined, want) {
			t.Fatalf("row %s missing %q: %#v", row.Category, want, row)
		}
	}
}

func cloneFeatureSurfaceAuditReport(report FeatureSurfaceAuditReport) FeatureSurfaceAuditReport {
	report.Rows = append([]FeatureSurfaceAuditRow{}, report.Rows...)
	for i := range report.Rows {
		report.Rows[i].FeatureIDs = append([]string{}, report.Rows[i].FeatureIDs...)
		report.Rows[i].Evidence = append([]string{}, report.Rows[i].Evidence...)
		report.Rows[i].Boundaries = append([]string{}, report.Rows[i].Boundaries...)
		report.Rows[i].RequiredPromotionEvidence = append([]string{}, report.Rows[i].RequiredPromotionEvidence...)
		registryStatuses := report.Rows[i].RegistryStatuses
		report.Rows[i].RegistryStatuses = map[string]FeatureStatus{}
		for id, status := range registryStatuses {
			report.Rows[i].RegistryStatuses[id] = status
		}
	}
	report.NonClaims = append([]string{}, report.NonClaims...)
	return report
}
