package main

import (
	"encoding/json"
	"testing"

	"tetra_language/tools/validators/parallelprod"
)

func TestBuildReportProducesValidParallelProductionEvidence(t *testing.T) {
	report := buildReport("tools/cmd/parallel-production-smoke", []parallelprod.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "/tmp/tetra", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "parallel smoke app", Kind: "app", Path: "/tmp/parallel-smoke", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "parallel stress", Kind: "stress", Path: "/tmp/parallel-stress", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, requiredPassingCases())
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := parallelprod.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestRequiredPassingCasesIncludeParallelEdgeCases(t *testing.T) {
	cases := requiredPassingCases()
	for _, want := range []string{
		"task group cancel wakes deadline join",
		"nested cancellation propagation",
		"task actor mailbox handoff",
		"resource double join diagnostic",
		"task group use-after-close diagnostic",
	} {
		if !hasCase(cases, want) {
			t.Fatalf("requiredPassingCases missing %q", want)
		}
	}
}

func hasCase(cases []parallelprod.CaseReport, name string) bool {
	for _, c := range cases {
		if c.Name == name {
			return true
		}
	}
	return false
}
