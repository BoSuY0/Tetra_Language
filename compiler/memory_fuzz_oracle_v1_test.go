package compiler

import (
	"strings"
	"testing"

	"tetra_language/compiler/memoryvocab"
)

func TestMemoryFuzzOracleReportCoversMPC15CategoriesAndInvariants(t *testing.T) {
	report, err := BuildMemoryFuzzOracleReport()
	if err != nil {
		t.Fatalf("BuildMemoryFuzzOracleReport: %v", err)
	}
	if report.SchemaVersion != MemoryFuzzOracleSchemaV1 {
		t.Fatalf("schema = %q, want %q", report.SchemaVersion, MemoryFuzzOracleSchemaV1)
	}
	if report.Scope != MemoryFuzzOracleScopeMPC15 {
		t.Fatalf("scope = %q, want %q", report.Scope, MemoryFuzzOracleScopeMPC15)
	}
	if err := ValidateMemoryFuzzOracleReport(report); err != nil {
		t.Fatalf("ValidateMemoryFuzzOracleReport: %v", err)
	}
	if report.Tier1ShortCISmokeCases == 0 || !report.Tier2NightlyBoundaryRecorded || !report.Tier3ReleaseBlockingBoundaryRecorded {
		t.Fatalf("tier coverage incomplete: %#v", report)
	}

	rows := map[MemoryFuzzOracleCategory]MemoryFuzzOracleRow{}
	for _, row := range report.Rows {
		if row.Category == "" || row.Tier == "" || row.ExpectedResult == "" || row.Status == "" || len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 {
			t.Fatalf("oracle row missing metadata: %#v", row)
		}
		rows[row.Category] = row
	}
	for _, category := range memoryFuzzOracleCategories() {
		if _, ok := rows[category]; !ok {
			t.Fatalf("missing oracle category %s: %#v", category, report.Rows)
		}
	}
	assertMemoryFuzzOracleRow(t, rows[MemoryFuzzOracleCheckerRejectExpected], MemoryFuzzOraclePass, []string{"checker reject", "borrow escape"})
	assertMemoryFuzzOracleRow(t, rows[MemoryFuzzOracleRuntimeTrapExpected], MemoryFuzzOraclePass, []string{"runtime trap", "bounds"})
	assertMemoryFuzzOracleRow(t, rows[MemoryFuzzOracleReferenceOutputExpected], MemoryFuzzOraclePass, []string{"compiled output", "reference"})
	assertMemoryFuzzOracleRow(t, rows[MemoryFuzzOracleCompilerCrashBug], MemoryFuzzOracleBug, []string{"compiler crash", "bug"})
	assertMemoryFuzzOracleRow(t, rows[MemoryFuzzOracleMiscompileBug], MemoryFuzzOracleBug, []string{"miscompile", "bug"})
	assertMemoryFuzzOracleRow(t, rows[MemoryFuzzOracleUnsafeUnknownOptimizedAsSafeBug], MemoryFuzzOracleBug, []string{"unsafe_unknown", "safe_known"})
	assertMemoryFuzzOracleRow(t, rows[MemoryFuzzOracleReportValidationFailureBug], MemoryFuzzOracleBug, []string{"report validation failure", "MemoryFactGraph"})

	invariants := map[MemoryFuzzInvariantID]MemoryFuzzInvariantRow{}
	for _, row := range report.Invariants {
		invariants[row.ID] = row
	}
	for _, id := range memoryFuzzInvariantIDs() {
		row, ok := invariants[id]
		if !ok {
			t.Fatalf("missing invariant %s: %#v", id, report.Invariants)
		}
		if row.Status != "covered" || len(row.Evidence) == 0 || len(row.Tests) == 0 {
			t.Fatalf("invariant %s incomplete: %#v", id, row)
		}
	}
	for _, nonClaim := range []string{
		"no exhaustive fuzzing is claimed",
		"no unsupported unsafe pointer safety is claimed",
		"no runtime behavior change",
		"no safe-program semantics change",
	} {
		if !memoryFuzzHasString(report.NonClaims, nonClaim) {
			t.Fatalf("missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestMemoryFuzzOracleRejectsUnknownVocabularyStatus(t *testing.T) {
	report, err := BuildMemoryFuzzOracleReport()
	if err != nil {
		t.Fatalf("BuildMemoryFuzzOracleReport: %v", err)
	}
	report.GeneratorSurfaces[0].Status = "looks_good_to_me"
	err = ValidateMemoryFuzzOracleReport(report)
	if err == nil {
		t.Fatalf("expected unknown generator surface status to fail")
	}
	if !strings.Contains(err.Error(), "unknown generator surface status") || !strings.Contains(err.Error(), "looks_good_to_me") {
		t.Fatalf("error = %v, want unknown status rejection", err)
	}
	if !memoryvocab.KnownMemoryFuzzStatus(memoryvocab.FuzzStatusCovered) {
		t.Fatalf("shared memory vocabulary must include covered fuzz status")
	}
}

func TestClassifyMemoryFuzzOracleObservation(t *testing.T) {
	tests := []struct {
		name     string
		category MemoryFuzzOracleCategory
		obs      MemoryFuzzObservation
		want     MemoryFuzzOracleResult
	}{
		{name: "checker reject expected", category: MemoryFuzzOracleCheckerRejectExpected, obs: MemoryFuzzObservation{CheckerRejected: true}, want: MemoryFuzzOraclePass},
		{name: "runtime trap expected", category: MemoryFuzzOracleRuntimeTrapExpected, obs: MemoryFuzzObservation{RuntimeTrapped: true}, want: MemoryFuzzOraclePass},
		{name: "reference equality expected", category: MemoryFuzzOracleReferenceOutputExpected, obs: MemoryFuzzObservation{ReferenceCompared: true, CompiledExitCode: 42, ReferenceExitCode: 42}, want: MemoryFuzzOraclePass},
		{name: "compiler crash is bug", category: MemoryFuzzOracleCompilerCrashBug, obs: MemoryFuzzObservation{CompilerCrashed: true}, want: MemoryFuzzOracleBug},
		{name: "miscompile is bug", category: MemoryFuzzOracleMiscompileBug, obs: MemoryFuzzObservation{ReferenceCompared: true, CompiledExitCode: 7, ReferenceExitCode: 9}, want: MemoryFuzzOracleBug},
		{name: "unsafe_unknown optimized as safe is bug", category: MemoryFuzzOracleUnsafeUnknownOptimizedAsSafeBug, obs: MemoryFuzzObservation{UnsafeUnknownOptimizedAsSafe: true}, want: MemoryFuzzOracleBug},
		{name: "report validation failure is bug", category: MemoryFuzzOracleReportValidationFailureBug, obs: MemoryFuzzObservation{ReportValidationFailed: true}, want: MemoryFuzzOracleBug},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyMemoryFuzzOracleObservation(tc.category, tc.obs)
			if got != tc.want {
				t.Fatalf("ClassifyMemoryFuzzOracleObservation(%s, %#v) = %q, want %q", tc.category, tc.obs, got, tc.want)
			}
		})
	}
}

func TestValidateMemoryFuzzOracleReportRejectsDrift(t *testing.T) {
	base, err := BuildMemoryFuzzOracleReport()
	if err != nil {
		t.Fatalf("BuildMemoryFuzzOracleReport: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*MemoryFuzzOracleReport)
		want   string
	}{
		{
			name: "missing oracle category",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.Rows = report.Rows[1:]
			},
			want: "missing oracle_category",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.Rows[0].Evidence = []string{"TODO"}
			},
			want: "placeholder",
		},
		{
			name: "bug category downgraded",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.RowsByCategory(MemoryFuzzOracleCompilerCrashBug).ExpectedResult = MemoryFuzzOraclePass
			},
			want: "compiler_crash_is_bug",
		},
		{
			name: "missing invariant",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.Invariants = report.Invariants[1:]
			},
			want: "missing invariant",
		},
		{
			name: "missing tier 1",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.Tier1ShortCISmokeCases = 0
			},
			want: "Tier 1",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := cloneMemoryFuzzOracleReport(base)
			tc.mutate(&report)
			err := ValidateMemoryFuzzOracleReport(report)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateMemoryFuzzOracleReport error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestMemoryFuzzOracleReportCoversV12ReleaseEvidence(t *testing.T) {
	report, err := BuildMemoryFuzzOracleReport()
	if err != nil {
		t.Fatalf("BuildMemoryFuzzOracleReport: %v", err)
	}
	if err := ValidateMemoryFuzzOracleReport(report); err != nil {
		t.Fatalf("ValidateMemoryFuzzOracleReport: %v", err)
	}

	requirements := map[MemoryFuzzRequirementID]MemoryFuzzRequirementRow{}
	for _, row := range report.Requirements {
		requirements[row.ID] = row
		if row.Status == "" || len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 {
			t.Fatalf("requirement %s missing release evidence: %#v", row.ID, row)
		}
	}
	wantRequirementStatuses := map[MemoryFuzzRequirementID]string{
		MemoryFuzzRequirementTier1V0V11Coverage:         "validated_narrow",
		MemoryFuzzRequirementCrashMiscompileArtifacts:   "validated_narrow",
		MemoryFuzzRequirementBlockingMemoryFailures:     "release_blocking",
		MemoryFuzzRequirementTier2NightlySeedTriage:     "boundary_recorded",
		MemoryFuzzRequirementTier3ReleasePassOrClassify: "release_blocking",
	}
	for _, id := range memoryFuzzRequirementIDs() {
		row, ok := requirements[id]
		if !ok {
			t.Fatalf("missing requirement %s: %#v", id, report.Requirements)
		}
		if row.Status != wantRequirementStatuses[id] {
			t.Fatalf("requirement %s status = %q, want %q", id, row.Status, wantRequirementStatuses[id])
		}
	}

	coverage := map[string]MemoryFuzzSliceCoverageRow{}
	for _, row := range report.SliceCoverage {
		coverage[row.SliceID] = row
		if row.Status != "covered" || len(row.Surface) == 0 || len(row.OracleCategories) == 0 || len(row.Invariants) == 0 || len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 {
			t.Fatalf("slice coverage %s incomplete: %#v", row.SliceID, row)
		}
	}
	for _, sliceID := range []string{"v0", "v1", "v2", "v3", "v4", "v5", "v6", "v7", "v8", "v9", "v10", "v11"} {
		if _, ok := coverage[sliceID]; !ok {
			t.Fatalf("missing deterministic Tier 1 slice coverage %s: %#v", sliceID, report.SliceCoverage)
		}
	}

	for _, kind := range []string{"tier1_short_ci_smoke_summary_json", "compiler_crash_reproducer", "miscompile_reducer", "miscompile_reproducer"} {
		if !memoryFuzzHasArtifactKind(report.Artifacts, kind) {
			t.Fatalf("missing required artifact kind %q: %#v", kind, report.Artifacts)
		}
	}

	blocking := map[MemoryFuzzBlockingCaseID]MemoryFuzzBlockingCaseRow{}
	for _, row := range report.BlockingCases {
		blocking[row.ID] = row
		if row.Status != "blocks_release" || !row.BlocksRelease || len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 {
			t.Fatalf("blocking case %s incomplete: %#v", row.ID, row)
		}
	}
	for _, id := range memoryFuzzBlockingCaseIDs() {
		if _, ok := blocking[id]; !ok {
			t.Fatalf("missing blocking case %s: %#v", id, report.BlockingCases)
		}
	}

	policies := map[MemoryFuzzTier]MemoryFuzzTierPolicyRow{}
	for _, row := range report.TierPolicies {
		policies[row.Tier] = row
	}
	tier2 := policies[MemoryFuzzTier2Nightly]
	if tier2.Status != "boundary_recorded" || !tier2.SeedsPreserved || !tier2.UnstableTriageRequired || !tier2.MinimizedReproducerRequired {
		t.Fatalf("Tier 2 policy incomplete: %#v", tier2)
	}
	tier3 := policies[MemoryFuzzTier3ReleaseFocused]
	if tier3.Status != "release_blocking" || !tier3.ReleasePromotionBlockedUntilClassified || !tier3.MinimizedReproducerRequired {
		t.Fatalf("Tier 3 policy incomplete: %#v", tier3)
	}
}

func TestValidateMemoryFuzzOracleReportRejectsV12ReleaseEvidenceDrift(t *testing.T) {
	base, err := BuildMemoryFuzzOracleReport()
	if err != nil {
		t.Fatalf("BuildMemoryFuzzOracleReport: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*MemoryFuzzOracleReport)
		want   string
	}{
		{
			name: "missing requirement",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.Requirements = report.Requirements[1:]
			},
			want: "missing requirement MEM-FUZZ-001",
		},
		{
			name: "missing v11 slice coverage",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.SliceCoverage = removeMemoryFuzzSliceCoverage(report.SliceCoverage, "v11")
			},
			want: "missing slice coverage v11",
		},
		{
			name: "compiler crash reproducer missing",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.Artifacts = removeMemoryFuzzArtifactKind(report.Artifacts, "compiler_crash_reproducer")
			},
			want: "compiler_crash_reproducer",
		},
		{
			name: "miscompile reducer missing",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.Artifacts = removeMemoryFuzzArtifactKind(report.Artifacts, "miscompile_reducer")
			},
			want: "miscompile_reducer",
		},
		{
			name: "unsafe optimized as safe does not block",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.BlockingCase(MemoryFuzzBlockingUnsafeUnknownOptimizedAsSafe).BlocksRelease = false
			},
			want: "blocks_release",
		},
		{
			name: "tier 2 seed preservation dropped",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.TierPolicy(MemoryFuzzTier2Nightly).SeedsPreserved = false
			},
			want: "Tier 2 nightly fuzz seed preservation",
		},
		{
			name: "tier 3 release classification dropped",
			mutate: func(report *MemoryFuzzOracleReport) {
				report.TierPolicy(MemoryFuzzTier3ReleaseFocused).ReleasePromotionBlockedUntilClassified = false
			},
			want: "Tier 3 release-blocking memory fuzz",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := cloneMemoryFuzzOracleReport(base)
			tc.mutate(&report)
			err := ValidateMemoryFuzzOracleReport(report)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateMemoryFuzzOracleReport error = %v, want %q", err, tc.want)
			}
		})
	}
}

func assertMemoryFuzzOracleRow(t *testing.T, row MemoryFuzzOracleRow, wantResult MemoryFuzzOracleResult, wants []string) {
	t.Helper()
	if row.ExpectedResult != wantResult {
		t.Fatalf("row %s expected_result = %q, want %q", row.Category, row.ExpectedResult, wantResult)
	}
	text := strings.Join(row.Evidence, " ") + " " + strings.Join(row.Boundaries, " ")
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("row %s missing %q: %#v", row.Category, want, row)
		}
	}
}

func memoryFuzzHasArtifactKind(artifacts []MemoryFuzzArtifact, kind string) bool {
	for _, artifact := range artifacts {
		if artifact.Kind == kind && artifact.Required {
			return true
		}
	}
	return false
}

func removeMemoryFuzzArtifactKind(artifacts []MemoryFuzzArtifact, kind string) []MemoryFuzzArtifact {
	var kept []MemoryFuzzArtifact
	for _, artifact := range artifacts {
		if artifact.Kind != kind {
			kept = append(kept, artifact)
		}
	}
	return kept
}

func removeMemoryFuzzSliceCoverage(rows []MemoryFuzzSliceCoverageRow, sliceID string) []MemoryFuzzSliceCoverageRow {
	var kept []MemoryFuzzSliceCoverageRow
	for _, row := range rows {
		if row.SliceID != sliceID {
			kept = append(kept, row)
		}
	}
	return kept
}
