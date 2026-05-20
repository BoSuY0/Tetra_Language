package main

import (
	"encoding/json"
	"testing"

	"tetra_language/tools/validators/compilerprod"
)

func TestBuildReportProducesValidCompilerProductionEvidence(t *testing.T) {
	report := buildReport("tools/cmd/compiler-production-smoke", []compilerprod.ProcessReport{
		process("tetra compiler build", "build"),
		process("linux native compile", "compile"),
		process("compiler focused tests", "test"),
		process("smoke profile compile matrix", "stress"),
	}, requiredPassingCases())
	raw := mustJSON(t, report)
	if err := compilerprod.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestRequiredPassingCasesIncludeCompilerProductionEdgeCases(t *testing.T) {
	cases := requiredPassingCases()
	for _, name := range []string{
		"fresh CLI compiler build",
		"linux-x64 native compile and run",
		"linux-x64 object emission",
		"interface-only compile",
		"wasm32-wasi module emission",
		"wasm32-web module and loader emission",
		"frontend parser fixture corpus",
		"semantic diagnostics stability",
		"IR verifier diagnostics",
		"backend format emission",
		"compiler cache separates modes",
		"smoke profile compilation matrix",
	} {
		if !hasCase(cases, name) {
			t.Fatalf("requiredPassingCases missing %q", name)
		}
	}
}

func requiredPassingCases() []compilerprod.CaseReport {
	return []compilerprod.CaseReport{
		{Name: "fresh CLI compiler build", Kind: "positive", Ran: true, Pass: true},
		{Name: "version reports v0.4.0", Kind: "positive", Ran: true, Pass: true},
		{Name: "linux-x64 native compile and run", Kind: "positive", Ran: true, Pass: true},
		{Name: "linux-x64 object emission", Kind: "positive", Ran: true, Pass: true},
		{Name: "interface-only compile", Kind: "positive", Ran: true, Pass: true},
		{Name: "wasm32-wasi module emission", Kind: "positive", Ran: true, Pass: true},
		{Name: "wasm32-web module and loader emission", Kind: "positive", Ran: true, Pass: true},
		{Name: "frontend parser fixture corpus", Kind: "positive", Ran: true, Pass: true},
		{Name: "semantic diagnostics stability", Kind: "negative", Ran: true, Pass: true, ExpectedError: "semantic diagnostic"},
		{Name: "IR verifier diagnostics", Kind: "negative", Ran: true, Pass: true, ExpectedError: "IR verifier"},
		{Name: "backend format emission", Kind: "positive", Ran: true, Pass: true},
		{Name: "CLI build option diagnostics", Kind: "negative", Ran: true, Pass: true, ExpectedError: "unsupported --runtime"},
		{Name: "compiler cache separates modes", Kind: "positive", Ran: true, Pass: true},
		{Name: "smoke profile compilation matrix", Kind: "stress", Ran: true, Pass: true},
	}
}

func process(name, kind string) compilerprod.ProcessReport {
	exitZero := 0
	return compilerprod.ProcessReport{Name: name, Kind: kind, Path: name, Ran: true, Pass: true, ExitCode: &exitZero}
}

func hasCase(cases []compilerprod.CaseReport, name string) bool {
	for _, c := range cases {
		if c.Name == name {
			return true
		}
	}
	return false
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
