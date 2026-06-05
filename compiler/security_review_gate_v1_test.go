package compiler

import (
	"strings"
	"testing"
)

func TestP24SecurityReviewGateV1CoversMasterPlanSurfacesAndArtifacts(t *testing.T) {
	report, err := BuildP24SecurityReviewGateV1Report()
	if err != nil {
		t.Fatalf("BuildP24SecurityReviewGateV1Report: %v", err)
	}
	if report.SchemaVersion != securityReviewGateV1Schema {
		t.Fatalf("schema = %q, want %q", report.SchemaVersion, securityReviewGateV1Schema)
	}
	if report.Scope != securityReviewGateV1ScopeP240 {
		t.Fatalf("scope = %q, want %q", report.Scope, securityReviewGateV1ScopeP240)
	}
	if err := ValidateP24SecurityReviewGateV1Report(report); err != nil {
		t.Fatalf("ValidateP24SecurityReviewGateV1Report: %v", err)
	}

	rows := map[SecurityReviewGateV1ID]SecurityReviewGateV1Row{}
	for _, row := range report.Rows {
		if row.ID == "" || row.Name == "" || row.Status == "" || len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 || len(row.WitnessIDs) == 0 {
			t.Fatalf("row missing required metadata: %#v", row)
		}
		rows[row.ID] = row
	}
	for _, id := range p24SecurityReviewGateV1IDs() {
		if _, ok := rows[id]; !ok {
			t.Fatalf("missing row %s: %#v", id, report.Rows)
		}
	}
	p24AssertSecurityReviewGateRow(t, rows[SecurityReviewUnsafeAPISurface], []string{"docs/spec/unsafe.md", "core.cap_mem", "core.alloc_bytes"})
	p24AssertSecurityReviewGateRow(t, rows[SecurityReviewCapabilitySurface], []string{"docs/spec/capabilities.md", "cap.mem", "uses"})
	p24AssertSecurityReviewGateRow(t, rows[SecurityReviewMemoryAllocator], []string{"RuntimeAllocationContracts", "raw-pointer-bounds-v1", "core.alloc_bytes"})
	p24AssertSecurityReviewGateRow(t, rows[SecurityReviewNetworkRuntime], []string{"IOReactorCoverage", "Linux epoll", "backpressure"})
	p24AssertSecurityReviewGateRow(t, rows[SecurityReviewActorRuntime], []string{"ActorRuntimeProductionBoundaryAudit", "message pool", "not a production"})
	p24AssertSecurityReviewGateRow(t, rows[SecurityReviewDBProtocol], []string{"ProductionPostgresCoverage", "SCRAM-SHA-256", "ErrFrameTooLarge", "ErrPoolExhausted"})
	p24AssertSecurityReviewGateRow(t, rows[SecurityReviewPackageEcoSystem], []string{"tetra.eco.publish.v1", "Tetra.lock", "validate-eco"})
	p24AssertSecurityReviewGateRow(t, rows[SecurityReviewBuildScripts], []string{"scripts/release/v1_0/security-review.sh", "Artifact Hashes", "current_release_version"})
	p24AssertSecurityReviewGateRow(t, rows[SecurityReviewSupplyChain], []string{"sha256", "go.sum", "trust snapshot", "no network trust claim"})
	p24AssertSecurityReviewGateRow(t, rows[SecurityReviewArtifactSet], []string{"security-review.md", "threat-model.md", "unsafe-surface-map.md", "capability-surface-map.md"})

	witnesses := map[string]SecurityReviewGateV1Witness{}
	for _, witness := range report.Witnesses {
		witnesses[witness.ID] = witness
	}
	allocator := witnesses[p24SecurityReviewAllocatorWitnessID]
	if !allocator.MemoryAllocatorReviewed || allocator.RuntimeAllocationContracts < 5 || allocator.RawPointerBoundsMetadataVersion != "raw-pointer-bounds-v1" {
		t.Fatalf("allocator witness = %#v, want allocation contracts and raw-pointer bounds metadata", allocator)
	}
	network := witnesses[p24SecurityReviewNetworkWitnessID]
	if !network.NetworkRuntimeReviewed || network.IOReactorRows < 10 {
		t.Fatalf("network witness = %#v, want IO reactor coverage", network)
	}
	actor := witnesses[p24SecurityReviewActorWitnessID]
	if !actor.ActorRuntimeReviewed || actor.ActorBoundaryRows < 4 {
		t.Fatalf("actor witness = %#v, want actor runtime boundary audit", actor)
	}
	db := witnesses[p24SecurityReviewDBWitnessID]
	if !db.DBProtocolReviewed || db.ProductionPostgresRows < 8 {
		t.Fatalf("DB witness = %#v, want production PostgreSQL coverage", db)
	}
	artifacts := witnesses[p24SecurityReviewArtifactsWitnessID]
	if !artifacts.SecurityReviewArtifactPresent || !artifacts.ThreatModelArtifactPresent || !artifacts.UnsafeSurfaceMapPresent || !artifacts.CapabilitySurfaceMapPresent {
		t.Fatalf("artifact witness missing required artifact: %#v", artifacts)
	}

	if !report.UnsafeAPISurfaceReviewed || !report.CapabilitySurfaceReviewed || !report.MemoryAllocatorReviewed || !report.NetworkRuntimeReviewed || !report.ActorRuntimeReviewed || !report.DBProtocolReviewed || !report.PackageEcoSystemReviewed || !report.BuildScriptsReviewed || !report.SupplyChainReviewed {
		t.Fatalf("review flags missing: %#v", report)
	}
	if !report.SecurityReviewArtifactPresent || !report.ThreatModelArtifactPresent || !report.UnsafeSurfaceMapPresent || !report.CapabilitySurfaceMapPresent {
		t.Fatalf("required artifacts missing: %#v", report.Artifacts)
	}
	for _, nonClaim := range []string{
		"security certification is not claimed",
		"external penetration test is not claimed",
		"CVE-free status is not claimed",
		"release security signoff is not claimed",
		"runtime behavior does not change",
		"safe-program semantics do not change",
		"no performance claim is made",
	} {
		if !p24SecurityReviewHasString(report.NonClaims, nonClaim) {
			t.Fatalf("missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestValidateP24SecurityReviewGateV1RejectsFakeClaimsAndWeakEvidence(t *testing.T) {
	base, err := BuildP24SecurityReviewGateV1Report()
	if err != nil {
		t.Fatalf("BuildP24SecurityReviewGateV1Report: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*SecurityReviewGateV1Report)
		want   string
	}{
		{
			name: "missing row",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.Rows = report.Rows[1:]
			},
			want: "missing row",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.Rows[0].Evidence = []string{"TODO"}
			},
			want: "placeholder",
		},
		{
			name: "missing witness",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.Rows[0].WitnessIDs = []string{"missing-witness"}
			},
			want: "missing witness",
		},
		{
			name: "missing artifact",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.Artifacts[0].Present = false
				report.SecurityReviewArtifactPresent = false
			},
			want: "security-review.md",
		},
		{
			name: "unsafe surface missing",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.UnsafeAPISurfaceReviewed = false
			},
			want: "unsafe API",
		},
		{
			name: "capability surface missing",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.CapabilitySurfaceReviewed = false
			},
			want: "capability",
		},
		{
			name: "memory allocator missing",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.MemoryAllocatorReviewed = false
			},
			want: "memory allocator",
		},
		{
			name: "network runtime missing",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.NetworkRuntimeReviewed = false
			},
			want: "network runtime",
		},
		{
			name: "actor runtime missing",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.ActorRuntimeReviewed = false
			},
			want: "actor runtime",
		},
		{
			name: "DB protocol missing",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.DBProtocolReviewed = false
			},
			want: "DB protocol",
		},
		{
			name: "package Eco missing",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.PackageEcoSystemReviewed = false
			},
			want: "package/Eco",
		},
		{
			name: "build scripts missing",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.BuildScriptsReviewed = false
			},
			want: "build scripts",
		},
		{
			name: "supply chain missing",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.SupplyChainReviewed = false
			},
			want: "supply chain",
		},
		{
			name: "security certification claim",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.SecurityCertifiedClaimed = true
			},
			want: "security certification",
		},
		{
			name: "external penetration claim",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.ExternalPenTestClaimed = true
			},
			want: "external penetration",
		},
		{
			name: "CVE free claim",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.CVEFreeClaimed = true
			},
			want: "CVE-free",
		},
		{
			name: "release signoff claim",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.ReleaseSignoffClaimed = true
			},
			want: "release signoff",
		},
		{
			name: "runtime behavior change",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.RuntimeBehaviorChanged = true
			},
			want: "runtime behavior",
		},
		{
			name: "safe semantics change",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.SafeSemanticsChanged = true
			},
			want: "safe semantics",
		},
		{
			name: "performance claim",
			mutate: func(report *SecurityReviewGateV1Report) {
				report.PerformanceClaimed = true
			},
			want: "performance",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := base
			report.Rows = append([]SecurityReviewGateV1Row(nil), base.Rows...)
			report.Witnesses = append([]SecurityReviewGateV1Witness(nil), base.Witnesses...)
			report.Artifacts = append([]SecurityReviewArtifact(nil), base.Artifacts...)
			report.NonClaims = append([]string(nil), base.NonClaims...)
			tc.mutate(&report)
			err := ValidateP24SecurityReviewGateV1Report(report)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateP24SecurityReviewGateV1Report error = %v, want %q", err, tc.want)
			}
		})
	}
}

func p24AssertSecurityReviewGateRow(t *testing.T, row SecurityReviewGateV1Row, wants []string) {
	t.Helper()
	text := strings.Join(row.Evidence, " ") + " " + strings.Join(row.Boundaries, " ")
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("row %s missing %q: %#v", row.ID, want, row)
		}
	}
}
