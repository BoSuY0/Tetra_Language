package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/compilerprod"
)

func TestValidateCompilerProductionReportAcceptsValidReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "compiler.json")
	if err := os.WriteFile(path, mustJSON(t, validCompilerReport()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateCompilerProductionReport(path); err != nil {
		t.Fatalf("validateCompilerProductionReport failed: %v", err)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestValidateCompilerProductionReportRejectsInvalidReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "compiler.json")
	report := validCompilerReport()
	report.Schema = "tetra.compiler.fake.v1"
	if err := os.WriteFile(path, mustJSON(t, report), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateCompilerProductionReport(path)
	if err == nil {
		t.Fatalf("expected invalid compiler production report to fail")
	}
	if !strings.Contains(err.Error(), compilerprod.SchemaV1) {
		t.Fatalf("error = %v, want schema rejection", err)
	}
}

func validCompilerReport() compilerprod.Report {
	exitZero := 0
	return compilerprod.Report{
		Schema:  compilerprod.SchemaV1,
		Status:  "pass",
		Target:  "linux-x64",
		Host:    "linux-x64",
		Runtime: "compiler-linux-x64",
		Source:  "tools/cmd/compiler-production-smoke",
		Processes: []compilerprod.ProcessReport{
			{Name: "tetra compiler build", Kind: "build", Path: "/tmp/tetra", Ran: true, Pass: true, ExitCode: &exitZero},
			{Name: "linux native compile", Kind: "compile", Path: "/tmp/flow-hello", Ran: true, Pass: true, ExitCode: &exitZero},
			{Name: "compiler focused tests", Kind: "test", Path: "go test ./compiler/...", Ran: true, Pass: true, ExitCode: &exitZero},
			{Name: "smoke profile compile matrix", Kind: "stress", Path: "tetra smoke --run=false", Ran: true, Pass: true, ExitCode: &exitZero},
		},
		Contracts: []compilerprod.ContractReport{
			{Name: "frontend parser and diagnostics", Status: "pass", Evidence: "parser fixtures and positioned diagnostics are verified"},
			{Name: "semantic safety and type checking", Status: "pass", Evidence: "semantic and safety diagnostics are verified"},
			{Name: "IR lowering and verifier", Status: "pass", Evidence: "lowering verifier accepts valid IR and rejects invalid IR"},
			{Name: "linux-x64 native backend and linker", Status: "pass", Evidence: "native linux-x64 compile and run evidence exists"},
			{Name: "wasm target emission", Status: "pass", Evidence: "wasm32-wasi and wasm32-web module emission evidence exists"},
			{Name: "object interface artifact pipeline", Status: "pass", Evidence: "object emission and interface-only compile are verified"},
			{Name: "CLI build check run contract", Status: "pass", Evidence: "CLI build diagnostics and version are verified"},
			{Name: "compiler cache and deterministic output", Status: "pass", Evidence: "cache mode separation and deterministic backend tests are verified"},
		},
		Cases: []compilerprod.CaseReport{
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
		},
		Audit: []compilerprod.AuditReport{
			{Requirement: "frontend parser and diagnostics", Artifact: "compiler/internal/frontend; compiler/tests/frontend", Evidence: "parser fixtures and positioned diagnostics are verified", Result: "pass"},
			{Requirement: "semantic safety and type checking", Artifact: "compiler/internal/semantics; compiler/tests/semantics; compiler/tests/safety", Evidence: "semantic and safety diagnostics are verified", Result: "pass"},
			{Requirement: "IR lowering and verifier", Artifact: "compiler/internal/lower", Evidence: "lowering verifier accepts valid IR and rejects invalid IR", Result: "pass"},
			{Requirement: "linux-x64 native backend and linker", Artifact: "compiler/internal/backend/linux_x64; compiler/internal/format/elf", Evidence: "native linux-x64 compile and run evidence exists", Result: "pass"},
			{Requirement: "wasm target emission", Artifact: "compiler/internal/backend/wasm32_wasi; compiler/internal/backend/wasm32_web", Evidence: "wasm32-wasi and wasm32-web module emission evidence exists", Result: "pass"},
			{Requirement: "object interface artifact pipeline", Artifact: "compiler/internal/format/tobj; cli/cmd/tetra/interface.go", Evidence: "object emission and interface-only compile are verified", Result: "pass"},
			{Requirement: "CLI build check run contract", Artifact: "cli/cmd/tetra/build_test.go; cli/cmd/tetra/run_test.go; docs/spec/cli_contracts.md", Evidence: "CLI build diagnostics and version are verified", Result: "pass"},
			{Requirement: "compiler cache and deterministic output", Artifact: "compiler/internal/cache; compiler/compiler_test.go", Evidence: "cache mode separation and deterministic backend tests are verified", Result: "pass"},
			{Requirement: "release-gate entrypoint", Artifact: "scripts/release/post_v0_4/compiler-production-linux-x64-smoke.sh", Evidence: "compiler gate writes and validates tetra.compiler.production.v1 evidence", Result: "pass"},
		},
	}
}
