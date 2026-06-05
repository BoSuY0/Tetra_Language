package compiler

import (
	"strings"
	"testing"
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
