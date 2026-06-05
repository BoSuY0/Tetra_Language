package compiler

import (
	"strings"
	"testing"
)

func TestP23FuzzPropertyDifferentialReportCoversMasterPlanTargets(t *testing.T) {
	report, err := BuildP23FuzzPropertyDifferentialReport()
	if err != nil {
		t.Fatalf("BuildP23FuzzPropertyDifferentialReport: %v", err)
	}
	if report.SchemaVersion != fuzzPropertyDifferentialSchema {
		t.Fatalf("schema = %q, want %q", report.SchemaVersion, fuzzPropertyDifferentialSchema)
	}
	if report.Scope != fuzzPropertyDifferentialScopeP231 {
		t.Fatalf("scope = %q, want %q", report.Scope, fuzzPropertyDifferentialScopeP231)
	}
	if err := ValidateP23FuzzPropertyDifferentialReport(report); err != nil {
		t.Fatalf("ValidateP23FuzzPropertyDifferentialReport: %v", err)
	}

	rows := map[FuzzPropertyDifferentialID]FuzzPropertyDifferentialRow{}
	for _, row := range report.Rows {
		if row.ID == "" || row.Name == "" || row.Status == "" || len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 || len(row.WitnessIDs) == 0 {
			t.Fatalf("row missing required metadata: %#v", row)
		}
		rows[row.ID] = row
	}
	for _, id := range p23FuzzPropertyDifferentialIDs() {
		if _, ok := rows[id]; !ok {
			t.Fatalf("missing row %s: %#v", id, report.Rows)
		}
	}
	p23AssertFuzzRow(t, rows[FuzzPropertyDifferentialParserCheckerGeneratedPrograms], []string{"generated source", "Parse", "Check"})
	p23AssertFuzzRow(t, rows[FuzzPropertyDifferentialPLIRLoweringVerifierPipeline], []string{"BuildPLIR", "Lower", "VerifyIRProgram"})
	p23AssertFuzzRow(t, rows[FuzzPropertyDifferentialBackendMatrixExpansion], []string{"CheckBackendMatrix", "SSA", "Machine IR", "randomized"})
	p23AssertFuzzRow(t, rows[FuzzPropertyDifferentialNativeBackendBoundary], []string{"native backend", "Linux x64", "explicit unavailable boundary"})
	p23AssertFuzzRow(t, rows[FuzzPropertyDifferentialRuntimeAllocatorProperties], []string{"AlignRegionBytes", "negative", "overflow"})
	p23AssertFuzzRow(t, rows[FuzzPropertyDifferentialActorTransferStressBoundary], []string{"TypedActorOwnershipTransferCoverage", "stress diagnostics", "PLIR moved facts"})
	p23AssertFuzzRow(t, rows[FuzzPropertyDifferentialFuzzNightlySummaryGate], []string{"fuzz-nightly.sh", "validate-fuzz-summary", "unstable-seeds"})
	p23AssertFuzzRow(t, rows[FuzzPropertyDifferentialReducerFailureArtifacts], []string{"reduced_to_single_sample", "reproducer"})

	if report.ParserCheckerGeneratedPrograms < 4 {
		t.Fatalf("parser/checker generated programs = %d, want at least 4", report.ParserCheckerGeneratedPrograms)
	}
	if report.PLIRVerifierCases < report.ParserCheckerGeneratedPrograms || report.LoweringVerifierCases < report.ParserCheckerGeneratedPrograms {
		t.Fatalf("pipeline counts parser=%d plir=%d lowering=%d", report.ParserCheckerGeneratedPrograms, report.PLIRVerifierCases, report.LoweringVerifierCases)
	}
	if report.BackendMatrixCases == 0 || report.BackendMatrixRandomizedSamples == 0 || !report.BackendMatrixReducerRecorded {
		t.Fatalf("backend matrix coverage incomplete: %#v", report)
	}
	if report.NativeBackendHostSupported {
		if report.NativeBackendSamples == 0 {
			t.Fatalf("native host supported but native samples = 0: %#v", report)
		}
	} else if !strings.Contains(report.NativeBackendUnavailableReason, "linux/amd64") {
		t.Fatalf("native unavailable reason = %q, want linux/amd64 boundary", report.NativeBackendUnavailableReason)
	}
	if report.RuntimeAllocatorPropertyCases == 0 || !report.RuntimeAllocatorRejectsInvalid {
		t.Fatalf("allocator property coverage incomplete: %#v", report)
	}
	if !report.ActorTransferStressDiagnostics {
		t.Fatalf("actor transfer stress diagnostics not recorded: %#v", report)
	}
	if report.FuzzSummaryGateArtifacts < 4 || !report.NightlyLongFuzzBoundaryRecorded {
		t.Fatalf("fuzz summary gate incomplete: %#v", report)
	}
	for _, nonClaim := range []string{
		"no full program correctness claim is made",
		"no exhaustive fuzzing is claimed",
		"no full native differential suite is claimed",
		"no performance claim is made",
		"runtime behavior does not change",
		"safe-program semantics do not change",
	} {
		if !p23FuzzHasString(report.NonClaims, nonClaim) {
			t.Fatalf("missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestValidateP23FuzzPropertyDifferentialRejectsFakeClaimsAndDrift(t *testing.T) {
	base, err := BuildP23FuzzPropertyDifferentialReport()
	if err != nil {
		t.Fatalf("BuildP23FuzzPropertyDifferentialReport: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*FuzzPropertyDifferentialReport)
		want   string
	}{
		{
			name: "missing row",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.Rows = report.Rows[1:]
			},
			want: "missing row",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.Rows[0].Evidence = []string{"TODO"}
			},
			want: "placeholder",
		},
		{
			name: "missing witness",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.Rows[0].WitnessIDs = []string{"missing-witness"}
			},
			want: "missing witness",
		},
		{
			name: "missing parser checker coverage",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.ParserCheckerGeneratedPrograms = 0
			},
			want: "parser/checker",
		},
		{
			name: "missing randomized backend samples",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.BackendMatrixRandomizedSamples = 0
			},
			want: "randomized",
		},
		{
			name: "missing reducer",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.BackendMatrixReducerRecorded = false
			},
			want: "reducer",
		},
		{
			name: "missing actor stress",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.ActorTransferStressDiagnostics = false
			},
			want: "actor transfer",
		},
		{
			name: "missing fuzz summary artifacts",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.FuzzSummaryGateArtifacts = 0
			},
			want: "fuzz summary",
		},
		{
			name: "missing nightly boundary",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.NightlyLongFuzzBoundaryRecorded = false
			},
			want: "nightly",
		},
		{
			name: "full correctness claim",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.FullCorrectnessClaimed = true
			},
			want: "full program correctness",
		},
		{
			name: "exhaustive fuzzing claim",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.ExhaustiveFuzzingClaimed = true
			},
			want: "exhaustive fuzzing",
		},
		{
			name: "full native differential claim",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.FullNativeDifferentialClaimed = true
			},
			want: "full native differential",
		},
		{
			name: "runtime behavior claim",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.RuntimeBehaviorChanged = true
			},
			want: "runtime behavior",
		},
		{
			name: "safe semantics claim",
			mutate: func(report *FuzzPropertyDifferentialReport) {
				report.SafeSemanticsChanged = true
			},
			want: "safe semantics",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := base
			report.Rows = append([]FuzzPropertyDifferentialRow(nil), base.Rows...)
			report.Witnesses = append([]FuzzPropertyDifferentialWitness(nil), base.Witnesses...)
			report.NonClaims = append([]string(nil), base.NonClaims...)
			tc.mutate(&report)
			err := ValidateP23FuzzPropertyDifferentialReport(report)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateP23FuzzPropertyDifferentialReport error = %v, want %q", err, tc.want)
			}
		})
	}
}

func p23AssertFuzzRow(t *testing.T, row FuzzPropertyDifferentialRow, wants []string) {
	t.Helper()
	text := strings.Join(row.Evidence, " ") + " " + strings.Join(row.Boundaries, " ")
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("row %s missing %q: %#v", row.ID, want, row)
		}
	}
}
