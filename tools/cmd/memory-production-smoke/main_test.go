package main

import (
	"encoding/json"
	"testing"

	"tetra_language/tools/validators/memoryprod"
)

func TestBuildReportProducesValidMemoryProductionEvidence(t *testing.T) {
	report := buildReport("tools/cmd/memory-production-smoke", []memoryprod.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "/tmp/tetra", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "memory smoke app", Kind: "app", Path: "/tmp/memory-smoke", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "memory stress", Kind: "stress", Path: "/tmp/memory-stress", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "memory fuzz", Kind: "stress", Path: "/tmp/memory-fuzz", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, requiredPassingCases())
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := memoryprod.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestRequiredPassingCasesIncludeMemoryProductionEdgeCases(t *testing.T) {
	cases := requiredPassingCases()
	for _, want := range []string{
		"cap.mem unsafe boundary",
		"callable mutable capture heap escape",
		"function-typed slice aggregate borrow escape coverage",
	} {
		if !hasCase(cases, want) {
			t.Fatalf("requiredPassingCases missing %q", want)
		}
	}
}

func hasCase(cases []memoryprod.CaseReport, name string) bool {
	for _, c := range cases {
		if c.Name == name {
			return true
		}
	}
	return false
}
