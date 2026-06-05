package compiler

import (
	"strings"
	"testing"
)

func TestP22FirstClassCallableCoverageProvesSafeABIWitnesses(t *testing.T) {
	report, err := BuildP22FirstClassCallableCoverage()
	if err != nil {
		t.Fatalf("BuildP22FirstClassCallableCoverage: %v", err)
	}
	if report.SchemaVersion != firstClassCallableCoverageSchemaV1 {
		t.Fatalf("schema = %q, want %q", report.SchemaVersion, firstClassCallableCoverageSchemaV1)
	}
	if report.Scope != firstClassCallableCoverageScopeP221 {
		t.Fatalf("scope = %q, want %q", report.Scope, firstClassCallableCoverageScopeP221)
	}
	if err := ValidateP22FirstClassCallableCoverage(report); err != nil {
		t.Fatalf("ValidateP22FirstClassCallableCoverage: %v", err)
	}

	rows := map[FirstClassCallableCoverageID]FirstClassCallableCoverageRow{}
	for _, row := range report.Rows {
		if row.ID == "" || row.Name == "" || len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 || len(row.WitnessIDs) == 0 {
			t.Fatalf("coverage row missing required metadata: %#v", row)
		}
		rows[row.ID] = row
	}
	for _, id := range p22FirstClassCallableCoverageIDs() {
		if _, ok := rows[id]; !ok {
			t.Fatalf("coverage missing row %s: %#v", id, report.Rows)
		}
	}

	p22AssertCallableRow(t, rows[FirstClassCallableFnPtrFastPath], []string{"FnPtrSlotCount", "9-slot", "no heap environment", "fnptr"})
	p22AssertCallableRow(t, rows[FirstClassCallableFatHandle], []string{"CallableHandleSlotCount", "4-slot handle", "IRAllocBytes", "IRMemWritePtrOffset", "IRMemReadPtrOffset"})
	p22AssertCallableRow(t, rows[FirstClassCallableCaptureSafetyClassifier], []string{"callable_escape.go", "closure_captures.go", "safe immutable by-value"})
	p22AssertCallableRow(t, rows[FirstClassCallableMutableCaptureDiagnostics], []string{"mutable by-reference capture", "global-escape", "heap-escape"})
	p22AssertCallableRow(t, rows[FirstClassCallableResourceThreadDiagnostics], []string{"pointer/resource capture", "thread-boundary callable escape"})
	p22AssertCallableRow(t, rows[FirstClassCallableFixedABIWidth], []string{"FnPtrEnvSlotCount = 8", "FnPtrSlotCount = 9", "CallableHandleSlotCount = 4", "fixed ABI width"})
	p22AssertCallableRow(t, rows[FirstClassCallableInterfaceMetadata], []string{".t4i", "ReturnFunctionHandleValue", "ReturnSlots = 4"})
	p22AssertCallableRow(t, rows[FirstClassCallableStorageCallbackPaths], []string{"aliases", "struct fields", "enum payloads", "callback arguments", "returns"})

	witnesses := map[string]FirstClassCallableABIWitness{}
	for _, witness := range report.Witnesses {
		witnesses[witness.ID] = witness
	}
	fnptr := witnesses[firstClassCallableFnPtrWitnessID]
	if fnptr.ID == "" {
		t.Fatalf("missing fnptr witness: %#v", report.Witnesses)
	}
	if fnptr.CaptureCount != 1 || fnptr.UsesHandle || fnptr.FnPtrSlotCount != 9 || fnptr.CallableHandleSlotCount != 4 || fnptr.LocalSlotCount != 9 || fnptr.AllocBytesCount != 0 {
		t.Fatalf("fnptr witness = %#v, want one-capture 9-slot fnptr without heap env", fnptr)
	}
	handle := witnesses[firstClassCallableHandleWitnessID]
	if handle.ID == "" {
		t.Fatalf("missing handle witness: %#v", report.Witnesses)
	}
	if handle.CaptureCount != 9 || !handle.UsesHandle || handle.FnPtrSlotCount != 9 || handle.CallableHandleSlotCount != 4 || handle.LocalSlotCount != 4 {
		t.Fatalf("handle witness = %#v, want nine-capture fixed 4-slot handle", handle)
	}
	if handle.AllocBytesCount != 1 || handle.EnvWriteCount != 9 || handle.EnvReadCount != 9 || handle.CallArgSlots != 10 || handle.CallRetSlots != 1 {
		t.Fatalf("handle witness IR counts = %#v, want alloc=1 writes=9 reads=9 call arg/ret=10/1", handle)
	}

	for _, nonClaim := range []string{
		"no variable-width callable ABI is claimed",
		"no exploding callable return slots are claimed",
		"no mutable by-reference capture support is claimed",
		"no pointer/resource capture support is claimed",
		"no thread-boundary callable transfer is claimed",
		"no runtime generic callable polymorphism is claimed",
		"no dynamic callable dispatch is claimed",
		"no unsafe lifetime relaxation is claimed",
		"no performance claim is made",
		"no runtime behavior change beyond the existing callable ABI is claimed",
		"safe-program semantics do not change",
	} {
		if !p22FirstClassCallableHasString(report.NonClaims, nonClaim) {
			t.Fatalf("missing non-claim %q: %#v", nonClaim, report.NonClaims)
		}
	}
}

func TestValidateP22FirstClassCallableCoverageRejectsFakeClaimsAndDrift(t *testing.T) {
	base, err := BuildP22FirstClassCallableCoverage()
	if err != nil {
		t.Fatalf("BuildP22FirstClassCallableCoverage: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*FirstClassCallableCoverageReport)
		want   string
	}{
		{
			name: "missing row",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.Rows = report.Rows[1:]
			},
			want: "missing row",
		},
		{
			name: "placeholder evidence",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.Rows[0].Evidence = []string{"TODO"}
			},
			want: "placeholder",
		},
		{
			name: "missing witness reference",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.Rows[0].WitnessIDs = []string{"missing-witness"}
			},
			want: "missing witness",
		},
		{
			name: "bad handle ABI width",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.Witnesses[1].CallableHandleSlotCount = 5
			},
			want: "fixed ABI",
		},
		{
			name: "bad handle env read count",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.Witnesses[1].EnvReadCount = 8
			},
			want: "handle witness",
		},
		{
			name: "variable ABI width claim",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.VariableABIWidthClaimed = true
			},
			want: "variable-width",
		},
		{
			name: "exploding return slots claim",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.ExplodingReturnSlotsClaimed = true
			},
			want: "exploding",
		},
		{
			name: "mutable by-ref capture claim",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.MutableByRefCaptureClaimed = true
			},
			want: "mutable by-reference",
		},
		{
			name: "pointer resource capture claim",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.PointerResourceCaptureClaimed = true
			},
			want: "pointer/resource",
		},
		{
			name: "thread transfer claim",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.ThreadBoundaryCallableTransferClaimed = true
			},
			want: "thread-boundary",
		},
		{
			name: "runtime generic callable claim",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.RuntimeGenericCallablePolymorphismClaimed = true
			},
			want: "runtime generic callable",
		},
		{
			name: "dynamic callable dispatch claim",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.DynamicCallableDispatchClaimed = true
			},
			want: "dynamic callable",
		},
		{
			name: "unsafe lifetime relaxation claim",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.UnsafeLifetimeRelaxationClaimed = true
			},
			want: "unsafe lifetime",
		},
		{
			name: "performance claim",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.PerformanceClaimed = true
			},
			want: "performance",
		},
		{
			name: "runtime behavior change claim",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.RuntimeBehaviorChanged = true
			},
			want: "runtime behavior",
		},
		{
			name: "safe semantics claim",
			mutate: func(report *FirstClassCallableCoverageReport) {
				report.SafeSemanticsChanged = true
			},
			want: "safe-program semantics",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := cloneFirstClassCallableCoverage(base)
			tc.mutate(&report)
			err := ValidateP22FirstClassCallableCoverage(report)
			if err == nil {
				t.Fatalf("ValidateP22FirstClassCallableCoverage accepted fake report: %#v", report)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func p22AssertCallableRow(t *testing.T, row FirstClassCallableCoverageRow, wants []string) {
	t.Helper()
	combined := row.Name + " " + row.Status + " " + strings.Join(row.Evidence, " ") + " " + strings.Join(row.Tests, " ") + " " + strings.Join(row.Boundaries, " ")
	for _, want := range wants {
		if !strings.Contains(combined, want) {
			t.Fatalf("row %s missing %q: %#v", row.ID, want, row)
		}
	}
}

func cloneFirstClassCallableCoverage(report FirstClassCallableCoverageReport) FirstClassCallableCoverageReport {
	report.Rows = append([]FirstClassCallableCoverageRow{}, report.Rows...)
	for i := range report.Rows {
		report.Rows[i].Evidence = append([]string{}, report.Rows[i].Evidence...)
		report.Rows[i].Tests = append([]string{}, report.Rows[i].Tests...)
		report.Rows[i].Boundaries = append([]string{}, report.Rows[i].Boundaries...)
		report.Rows[i].WitnessIDs = append([]string{}, report.Rows[i].WitnessIDs...)
	}
	report.Witnesses = append([]FirstClassCallableABIWitness{}, report.Witnesses...)
	report.NonClaims = append([]string{}, report.NonClaims...)
	return report
}

func p22FirstClassCallableHasString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
