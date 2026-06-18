package compilerprod

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateReportAcceptsCompilerProductionEvidence(t *testing.T) {
	raw := mustJSON(t, validReport())
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportRejectsPaperCompilerEvidence(t *testing.T) {
	report := validReport()
	report.Source = "docs-only-placeholder.md"
	report.Processes = nil
	report.Cases = nil
	raw := mustJSON(t, report)
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected thin compiler report to fail")
	}
	for _, want := range []string{"docs-only", "process", "case"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q: %v", want, err)
		}
	}
}

func TestValidateReportRejectsMissingWASMWebCompilerCoverage(t *testing.T) {
	report := validReport()
	report.Cases = filterCases(report.Cases, "wasm32-web module and loader emission")
	raw := mustJSON(t, report)
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing wasm32-web coverage to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "wasm32-web module and loader emission") {
		t.Fatalf("error = %v, want wasm32-web case rejection", err)
	}
}

func TestValidateReportRejectsVersionPinnedCompilerCase(t *testing.T) {
	report := validReport()
	report.Cases = replaceCaseName(report.Cases, VersionCaseName, "version reports v0.4.0")
	raw := mustJSON(t, report)
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected v0.4.0-pinned version coverage to fail")
	}
	if !strings.Contains(err.Error(), VersionCaseName) {
		t.Fatalf("error = %v, want current-version case rejection", err)
	}
}

func TestValidateReportRejectsMissingCompilerAudit(t *testing.T) {
	report := validReport()
	report.Audit = nil
	raw := mustJSON(t, report)
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing audit to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "completion audit") {
		t.Fatalf("error = %v, want completion audit rejection", err)
	}
}

func validReport() Report {
	exitZero := 0
	return Report{
		Schema:  SchemaV1,
		Status:  "pass",
		Target:  "linux-x64",
		Host:    "linux-x64",
		Runtime: "compiler-linux-x64",
		Source:  "tools/cmd/compiler-production-smoke",
		Processes: []ProcessReport{
			{
				Name:     "tetra compiler build",
				Kind:     "build",
				Path:     "/tmp/tetra",
				Ran:      true,
				Pass:     true,
				ExitCode: &exitZero,
			},
			{
				Name:     "linux native compile",
				Kind:     "compile",
				Path:     "/tmp/flow-hello",
				Ran:      true,
				Pass:     true,
				ExitCode: &exitZero,
			},
			{
				Name:     "compiler focused tests",
				Kind:     "test",
				Path:     "go test ./compiler/...",
				Ran:      true,
				Pass:     true,
				ExitCode: &exitZero,
			},
			{
				Name:     "smoke profile compile matrix",
				Kind:     "stress",
				Path:     "tetra smoke --run=false",
				Ran:      true,
				Pass:     true,
				ExitCode: &exitZero,
			},
		},
		Contracts: []ContractReport{
			{
				Name:     "frontend parser and diagnostics",
				Status:   "pass",
				Evidence: "parser fixtures and positioned diagnostics are verified",
			},
			{
				Name:     "semantic safety and type checking",
				Status:   "pass",
				Evidence: "semantic and safety diagnostics are verified",
			},
			{
				Name:     "IR lowering and verifier",
				Status:   "pass",
				Evidence: "lowering verifier accepts valid IR and rejects invalid IR",
			},
			{
				Name:     "linux-x64 native backend and linker",
				Status:   "pass",
				Evidence: "native linux-x64 compile and run evidence exists",
			},
			{
				Name:     "wasm target emission",
				Status:   "pass",
				Evidence: "wasm32-wasi and wasm32-web module emission evidence exists",
			},
			{
				Name:     "object interface artifact pipeline",
				Status:   "pass",
				Evidence: "object emission and interface-only compile are verified",
			},
			{
				Name:     "CLI build check run contract",
				Status:   "pass",
				Evidence: "CLI build diagnostics and version are verified",
			},
			{
				Name:     "compiler cache and deterministic output",
				Status:   "pass",
				Evidence: "cache mode separation and deterministic backend tests are verified",
			},
		},
		Cases: []CaseReport{
			{Name: "fresh CLI compiler build", Kind: "positive", Ran: true, Pass: true},
			{Name: VersionCaseName, Kind: "positive", Ran: true, Pass: true},
			{Name: "linux-x64 native compile and run", Kind: "positive", Ran: true, Pass: true},
			{Name: "linux-x64 object emission", Kind: "positive", Ran: true, Pass: true},
			{Name: "interface-only compile", Kind: "positive", Ran: true, Pass: true},
			{Name: "wasm32-wasi module emission", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "wasm32-web module and loader emission",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{Name: "frontend parser fixture corpus", Kind: "positive", Ran: true, Pass: true},
			{
				Name:          "semantic diagnostics stability",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "semantic diagnostic",
			},
			{
				Name:          "IR verifier diagnostics",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "IR verifier",
			},
			{Name: "backend format emission", Kind: "positive", Ran: true, Pass: true},
			{
				Name:          "CLI build option diagnostics",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "unsupported --runtime",
			},
			{Name: "compiler cache separates modes", Kind: "positive", Ran: true, Pass: true},
			{Name: "smoke profile compilation matrix", Kind: "stress", Ran: true, Pass: true},
		},
		Audit: []AuditReport{
			{
				Requirement: "frontend parser and diagnostics",
				Artifact:    "compiler/internal/frontend; compiler/tests/frontend",
				Evidence:    "parser fixtures and positioned diagnostics are verified",
				Result:      "pass",
			},
			{
				Requirement: "semantic safety and type checking",
				Artifact:    "compiler/internal/semantics; compiler/tests/semantics; compiler/tests/safety",
				Evidence:    "semantic and safety diagnostics are verified",
				Result:      "pass",
			},
			{
				Requirement: "IR lowering and verifier",
				Artifact:    "compiler/internal/lower",
				Evidence:    "lowering verifier accepts valid IR and rejects invalid IR",
				Result:      "pass",
			},
			{
				Requirement: "linux-x64 native backend and linker",
				Artifact:    "compiler/internal/backend/linux_x64; compiler/internal/format/elf",
				Evidence:    "native linux-x64 compile and run evidence exists",
				Result:      "pass",
			},
			{
				Requirement: "wasm target emission",
				Artifact:    "compiler/internal/backend/wasm32_wasi; compiler/internal/backend/wasm32_web",
				Evidence:    "wasm32-wasi and wasm32-web module emission evidence exists",
				Result:      "pass",
			},
			{
				Requirement: "object interface artifact pipeline",
				Artifact:    "compiler/internal/format/tobj; cli/cmd/tetra/tetra_commands.go",
				Evidence:    "object emission and interface-only compile are verified",
				Result:      "pass",
			},
			{
				Requirement: "CLI build check run contract",
				Artifact: ("cli/cmd/tetra/tetra_suite_test.go; cli/cmd/tetra/tetra_suite_" +
					"test.go; docs/spec/policy/cli_contracts.md"),
				Evidence: "CLI build diagnostics and version are verified",
				Result:   "pass",
			},
			{
				Requirement: "compiler cache and deterministic output",
				Artifact:    "compiler/internal/cache; compiler/compiler_suite_test.go",
				Evidence:    "cache mode separation and deterministic backend tests are verified",
				Result:      "pass",
			},
			{
				Requirement: "release-gate entrypoint",
				Artifact:    "scripts/release/post_v0_4/compiler-production-linux-x64-smoke.sh",
				Evidence:    "compiler gate writes and validates tetra.compiler.production.v1 evidence",
				Result:      "pass",
			},
		},
	}
}

func replaceCaseName(cases []CaseReport, oldName, newName string) []CaseReport {
	var out []CaseReport
	for _, c := range cases {
		if c.Name == oldName {
			c.Name = newName
		}
		out = append(out, c)
	}
	return out
}

func filterCases(cases []CaseReport, drop string) []CaseReport {
	var out []CaseReport
	for _, c := range cases {
		if c.Name != drop {
			out = append(out, c)
		}
	}
	return out
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
