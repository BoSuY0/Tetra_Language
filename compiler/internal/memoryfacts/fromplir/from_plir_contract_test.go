package fromplir

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/plir"
)

func TestBuildCopiesFunctionSummaryContractDigestToSummaryFacts(t *testing.T) {
	prog := &plir.Program{Funcs: []plir.Function{{
		Name:   "borrow_bytes",
		Module: "main",
		Summary: &plir.FunctionSummary{
			ContractSchema:      "tetra.semantic.function-contract.v1",
			ContractDigest:      "digest:borrow-bytes",
			ParamNames:          []string{"xs"},
			ParamTypes:          []string{"[]u8"},
			ParamOwnership:      []string{"borrow"},
			ReturnType:          "[]u8",
			ReturnOwnership:     "borrow",
			ReturnRegionSummary: map[string]int{"": 0},
		},
	}}}
	graph, err := Build("prog", prog)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	for _, fact := range graph.Facts() {
		if fact.FunctionID != "borrow_bytes" || fact.Claim != "may_return_region" {
			continue
		}
		if got := fact.ContractSchema; got != "tetra.semantic.function-contract.v1" {
			t.Fatalf("Fact.ContractSchema = %q", got)
		}
		if got := fact.ContractDigest; got != "digest:borrow-bytes" {
			t.Fatalf("Fact.ContractDigest = %q", got)
		}
		return
	}
	t.Fatalf("missing may_return_region fact: %#v", graph.Facts())
}

func TestBuildRejectsSourceFunctionSummaryMissingContractDigest(t *testing.T) {
	_, err := Build("prog", &plir.Program{Funcs: []plir.Function{{
		Name:   "missing_digest",
		Module: "main",
		Summary: &plir.FunctionSummary{
			ContractSchema: "tetra.semantic.function-contract.v1",
			ParamTypes:     []string{"i32"},
			ReturnType:     "i32",
		},
	}}})
	if err == nil {
		t.Fatalf("Build accepted source summary without contract digest")
	}
	if !strings.Contains(err.Error(), "missing contract digest") {
		t.Fatalf("Build error = %v, want missing contract digest", err)
	}
}
