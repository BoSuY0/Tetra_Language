package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateMemory100ReportDirRejectsPlaceholderRawMemoryContract(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(
			t,
			filepath.Join(root, "raw-memory-contract", "raw-memory-contract.json"),
			memory100PlaceholderJSON("tetra.raw-memory-contract.v1", memory100TestHead),
		)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected placeholder raw memory contract to fail")
	}
	if got := err.Error(); !strings.Contains(got, "raw memory contract") ||
		!strings.Contains(got, "operation") {
		t.Fatalf("error = %v, want raw memory contract operation rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsIncompleteRawPointerMatrix(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(
			t,
			filepath.Join(root, "raw-memory-contract", "raw-memory-contract.json"),
			map[string]any{
				"schema":   "tetra.raw-memory-contract.v1",
				"status":   "pass",
				"git_head": memory100TestHead,
				"operations": []any{
					map[string]any{
						"name": "core.alloc_bytes",
						"source_artifacts": []any{
							"compiler/internal/runtimeabi/runtimeabi_test/raw_pointer_bounds_test.go",
						},
						"positive_tests": []any{"allocation-base metadata"},
					},
					map[string]any{
						"name": "core.ptr_add",
						"source_artifacts": []any{
							"compiler/internal/runtimeabi/runtimeabi_test/raw_pointer_bounds_test.go",
						},
						"negative_tests": []any{"negative offset", "allocation upper bound"},
					},
					map[string]any{
						"name": "raw_slice_from_parts",
						"source_artifacts": []any{
							"compiler/tests/semantics/semantics_memory_surface_test.go",
						},
						"negative_tests": []any{
							"outside unsafe",
							"negative length",
							"i32 byte overflow",
						},
					},
					map[string]any{
						"name":             "memcpy_u8",
						"source_artifacts": []any{"lib/core/memory/memory.tetra"},
						"positive_tests":   []any{"cap.mem helper path"},
						"negative_tests":   []any{"negative length"},
						"non_claims":       []any{"no overlapping memcpy safety claim"},
					},
					map[string]any{
						"name":             "memset_u8",
						"source_artifacts": []any{"lib/core/memory/memory.tetra"},
						"positive_tests":   []any{"cap.mem helper path"},
						"negative_tests":   []any{"negative length"},
					},
					map[string]any{
						"name":             "cap.mem",
						"source_artifacts": []any{"lib/core/base/capability.tetra"},
						"negative_tests": []any{
							"unsafe_unknown promotion rejected",
							"cap.mem overclaim rejected",
						},
						"non_claims": []any{"no arbitrary external pointer safety claim"},
					},
				},
			},
		)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected incomplete raw pointer matrix to fail")
	}
	if got := err.Error(); !strings.Contains(got, "raw memory contract") ||
		!strings.Contains(got, "access-width overflow") {
		t.Fatalf("error = %v, want access-width overflow evidence rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsMissingRawLoadStoreMetadata(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		report := memory100JSON("tetra.raw-memory-contract.v1", memory100TestHead)
		var filtered []any
		for _, operation := range report["operations"].([]any) {
			if operation.(map[string]any)["name"] == "raw_load_store_metadata" {
				continue
			}
			filtered = append(filtered, operation)
		}
		report["operations"] = filtered
		writeMemory100JSON(
			t,
			filepath.Join(root, "raw-memory-contract", "raw-memory-contract.json"),
			report,
		)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected missing raw load/store metadata evidence to fail")
	}
	if got := err.Error(); !strings.Contains(got, "raw memory contract") ||
		!strings.Contains(got, "raw_load_store_metadata") {
		t.Fatalf("error = %v, want raw load/store metadata evidence rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsPlaceholderAllocationLowering(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(
			t,
			filepath.Join(root, "allocation-lowering", "allocation-lowering-report.json"),
			memory100PlaceholderJSON("tetra.allocation-lowering.v1", memory100TestHead),
		)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected placeholder allocation lowering report to fail")
	}
	if got := err.Error(); !strings.Contains(got, "allocation lowering") ||
		!strings.Contains(got, "decision") {
		t.Fatalf("error = %v, want allocation lowering decision rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsPlaceholderProofStoreSummary(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(
			t,
			filepath.Join(root, "proof-store", "proof-store-summary.json"),
			memory100PlaceholderJSON("tetra.proof-store-summary.v1", memory100TestHead),
		)
		writeMemory100HashManifest(t, root)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected placeholder proof store summary to fail")
	}
	if got := err.Error(); !strings.Contains(got, "proof store summary") ||
		!strings.Contains(got, "summary") {
		t.Fatalf("error = %v, want proof store summary content rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsLoweringMismatchWithoutBlocker(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		report := memory100JSON("tetra.allocation-lowering.v1", memory100TestHead)
		for _, raw := range report["decisions"].([]any) {
			decision := raw.(map[string]any)
			if decision["name"] == "stack_trusted_no_escape" {
				decision["planned_storage"] = "Stack"
				decision["actual_lowering_storage"] = "Heap"
				delete(decision, "blocker_artifact")
			}
		}
		writeMemory100JSON(
			t,
			filepath.Join(root, "allocation-lowering", "allocation-lowering-report.json"),
			report,
		)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected planned/actual lowering mismatch without blocker to fail")
	}
	if got := err.Error(); !strings.Contains(got, "allocation lowering") ||
		!strings.Contains(got, "mismatch") ||
		!strings.Contains(got, "blocker_artifact") {
		t.Fatalf("error = %v, want allocation lowering mismatch blocker rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsHeapCopyBlockersWithoutImpact(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		report := memory100JSON("tetra.allocation-lowering.v1", memory100TestHead)
		for _, raw := range report["decisions"].([]any) {
			decision := raw.(map[string]any)
			switch decision["name"] {
			case "heap_fallback_blocker":
				delete(decision, "budget_impact")
			case "copy_blocker":
				delete(decision, "grade_impact")
			}
		}
		writeMemory100JSON(
			t,
			filepath.Join(root, "allocation-lowering", "allocation-lowering-report.json"),
			report,
		)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected heap/copy blocker impact omissions to fail")
	}
	if got := err.Error(); !strings.Contains(got, "allocation lowering") ||
		(!strings.Contains(got, "budget_impact") && !strings.Contains(got, "grade_impact")) {
		t.Fatalf("error = %v, want heap/copy blocker impact rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsBlockedAllocationLoweringWithoutCoveredSites(
	t *testing.T,
) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		report := memory100JSON("tetra.allocation-lowering.v1", memory100TestHead)
		for _, raw := range report["decisions"].([]any) {
			decision := raw.(map[string]any)
			if decision["name"] == "heap_fallback_blocker" {
				delete(decision, "covered_site_ids")
			}
		}
		writeMemory100JSON(
			t,
			filepath.Join(root, "allocation-lowering", "allocation-lowering-report.json"),
			report,
		)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected blocked allocation lowering without covered_site_ids to fail")
	}
	if got := err.Error(); !strings.Contains(got, "allocation lowering") ||
		!strings.Contains(got, "covered_site_ids") {
		t.Fatalf("error = %v, want allocation lowering covered_site_ids rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsCopyBlockerWhenNoCopyRowsObserved(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		report := memory100JSON("tetra.allocation-lowering.v1", memory100TestHead)
		for _, raw := range report["decisions"].([]any) {
			decision := raw.(map[string]any)
			if decision["name"] == "copy_blocker" {
				decision["status"] = "blocked"
				decision["blocker_artifact"] = "ram-contract/copy-blockers.json"
				decision["covered_site_ids"] = []any{"site:main:heap"}
			}
		}
		writeMemory100JSON(
			t,
			filepath.Join(root, "allocation-lowering", "allocation-lowering-report.json"),
			report,
		)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected copy blocker without copy RAM rows to fail")
	}
	if got := err.Error(); !strings.Contains(got, "allocation lowering") ||
		!strings.Contains(got, "copy") ||
		!strings.Contains(got, "site:main:heap") {
		t.Fatalf("error = %v, want copy covered_site_ids/RAM row mismatch rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsAllocationLoweringActualStorageMismatchingRAMPlacement(
	t *testing.T,
) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		report := memory100JSON("tetra.allocation-lowering.v1", memory100TestHead)
		for _, raw := range report["decisions"].([]any) {
			decision := raw.(map[string]any)
			if decision["name"] == "heap_fallback_blocker" {
				decision["actual_lowering_storage"] = "Stack"
			}
		}
		writeMemory100JSON(
			t,
			filepath.Join(root, "allocation-lowering", "allocation-lowering-report.json"),
			report,
		)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected actual lowering storage/RAM placement mismatch to fail")
	}
	if got := err.Error(); !strings.Contains(got, "allocation lowering") ||
		!strings.Contains(got, "actual_lowering_storage") ||
		!strings.Contains(got, "site:main:heap") {
		t.Fatalf("error = %v, want actual lowering storage/RAM placement rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsUnbackedAllocationLoweringProofDecision(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		report := memory100JSON("tetra.allocation-lowering.v1", memory100TestHead)
		for _, raw := range report["decisions"].([]any) {
			decision := raw.(map[string]any)
			switch decision["name"] {
			case "stack_trusted_no_escape", "lowering_storage_match":
				decision["status"] = "proven"
				decision["proof_artifact"] = "ram-contract/proof-store-summary.json"
			}
		}
		writeMemory100JSON(
			t,
			filepath.Join(root, "allocation-lowering", "allocation-lowering-report.json"),
			report,
		)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf(
			"expected proof-backed allocation lowering without RAM proof-backed trusted rows to fail",
		)
	}
	if got := err.Error(); !strings.Contains(got, "allocation lowering") ||
		!strings.Contains(got, "proof-backed") {
		t.Fatalf("error = %v, want allocation lowering proof-backed RAM consistency rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsTrustedRAMRowsWithoutAllocationLoweringCoverage(
	t *testing.T,
) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		proofID := "proof:stack:noescape"
		writeMemory100JSON(
			t,
			filepath.Join(root, "ram-contract", "ram-contract-report.json"),
			map[string]any{
				"schema_version": "tetra.ram-contract-report.v1",
				"git_head":       memory100TestHead,
				"target":         "linux-x64",
				"generated_by":   "test",
				"rows": []any{
					map[string]any{
						"site_id":           "site:main:stack",
						"value_id":          "stack",
						"function":          "main",
						"intent":            "allocation",
						"requested_bytes":   64,
						"bounded":           true,
						"owner":             "function:main",
						"lifetime":          "function:main",
						"escape_status":     "no_escape",
						"placement":         "stack",
						"proof_ids":         []any{proofID},
						"blockers":          []any{},
						"contract_grade":    "M1",
						"validation_status": "validated",
					},
				},
				"proofs": []any{
					map[string]any{
						"proof_id":    proofID,
						"kind":        "allocation_placement",
						"subject":     "site:main:stack stack no_escape",
						"stable_hash": strings.Repeat("a", 64),
						"status":      "proven",
					},
				},
				"summary": map[string]any{
					"row_count":      1,
					"artifact_grade": "M1",
					"heap_rows":      0,
					"copy_rows":      0,
					"unbounded_rows": 0,
					"budget_bytes":   64,
				},
				"non_claims": []any{"no Memory 100% claim", "no full formal proof claim"},
			},
		)
		writeMemory100JSON(
			t,
			filepath.Join(root, "ram-contract", "proof-store-summary.json"),
			map[string]any{
				"schema_version": "tetra.proof-store-summary.v1",
				"git_head":       memory100TestHead,
				"target":         "linux-x64",
				"generated_by":   "test",
				"proofs": []any{
					map[string]any{
						"proof_id":    proofID,
						"kind":        "allocation_placement",
						"subject":     "site:main:stack stack no_escape",
						"stable_hash": strings.Repeat("a", 64),
						"status":      "proven",
					},
				},
				"summary": map[string]any{
					"proof_count":  1,
					"proven":       1,
					"conservative": 0,
					"rejected":     0,
					"unknown":      0,
				},
				"non_claims": []any{"no full formal proof claim"},
			},
		)
		writeMemory100JSON(
			t,
			filepath.Join(root, "ram-contract", "heap-blockers.json"),
			map[string]any{
				"schema_version": "tetra.ram-blockers.v1",
				"kind":           "heap",
				"git_head":       memory100TestHead,
				"target":         "linux-x64",
				"generated_by":   "test",
				"rows":           []any{},
				"non_claims":     []any{"no Memory 100% claim"},
			},
		)
		allocation := memory100JSON("tetra.allocation-lowering.v1", memory100TestHead)
		for _, raw := range allocation["decisions"].([]any) {
			decision := raw.(map[string]any)
			if decision["name"] != "heap_fallback_blocker" {
				continue
			}
			decision["status"] = "not_observed"
			delete(decision, "blocker_artifact")
			delete(decision, "blocker_reason")
			delete(decision, "budget_impact")
			delete(decision, "grade_impact")
			delete(decision, "validator_coverage")
			delete(decision, "covered_site_ids")
			decision["source_artifacts"] = []any{
				"ram-contract/heap-blockers.json",
				"ram-contract/ram-contract-report.json",
			}
		}
		writeMemory100JSON(
			t,
			filepath.Join(root, "allocation-lowering", "allocation-lowering-report.json"),
			allocation,
		)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf(
			"expected proof-backed trusted RAM row without allocation-lowering coverage to fail",
		)
	}
	if got := err.Error(); !strings.Contains(got, "allocation lowering") ||
		!strings.Contains(got, "proof-backed trusted") ||
		!strings.Contains(got, "site:main:stack") {
		t.Fatalf("error = %v, want proof-backed trusted RAM coverage rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsUnclassifiedRAMHeapRows(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(
			t,
			filepath.Join(root, "ram-contract", "ram-contract-report.json"),
			map[string]any{
				"schema_version": "tetra.ram-contract-report.v1",
				"git_head":       memory100TestHead,
				"target":         "linux-x64",
				"generated_by":   "test",
				"rows": []any{
					map[string]any{
						"site_id":           "site:main:heap",
						"value_id":          "heap",
						"function":          "main",
						"intent":            "heap_fallback",
						"requested_bytes":   64,
						"bounded":           true,
						"owner":             "function:main",
						"lifetime":          "function:main",
						"escape_status":     "unknown",
						"placement":         "heap_bounded",
						"proof_ids":         []any{},
						"blockers":          []any{},
						"contract_grade":    "M4",
						"validation_status": "unclassified",
					},
				},
				"proofs": []any{},
				"summary": map[string]any{
					"row_count":      1,
					"artifact_grade": "M4",
					"heap_rows":      1,
					"copy_rows":      0,
					"unbounded_rows": 0,
					"budget_bytes":   64,
				},
				"non_claims": []any{"no Memory 100% claim", "no full formal proof claim"},
			},
		)
		writeMemory100JSON(
			t,
			filepath.Join(root, "ram-contract", "heap-blockers.json"),
			map[string]any{
				"schema_version": "tetra.ram-blockers.v1",
				"kind":           "heap",
				"git_head":       memory100TestHead,
				"target":         "linux-x64",
				"generated_by":   "test",
				"rows":           []any{},
				"non_claims":     []any{"no Memory 100% claim"},
			},
		)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected unclassified RAM heap row to fail")
	}
	if got := err.Error(); !strings.Contains(got, "ram-contract") ||
		(!strings.Contains(got, "unclassified") && !strings.Contains(got, "heap")) {
		t.Fatalf("error = %v, want RAM heap classification rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsFakeRAMContractFuzzOracle(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(
			t,
			filepath.Join(root, "ram-contract", "fuzz", "ram-contract-fuzz-oracle.json"),
			map[string]any{
				"schema_version": "tetra.ram-contract-fuzz-oracle.v1",
				"git_head":       memory100TestHead,
				"generated_at":   "2026-06-10T11:00:00Z",
				"summary":        map[string]any{"mutations": 0, "rejected": 0},
				"observations":   []any{},
				"non_claims":     []any{"not Memory 100%"},
			},
		)
		writeMemory100HashManifest(t, filepath.Join(root, "ram-contract"))
	})
	writeMemory100HashManifest(t, reportDir)

	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected fake RAM contract fuzz oracle to fail")
	}
	if got := err.Error(); !strings.Contains(got, "RAM contract fuzz") ||
		!strings.Contains(got, "mutation") {
		t.Fatalf("error = %v, want RAM contract fuzz mutation rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsPlaceholderLeakResourceReport(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(
			t,
			filepath.Join(root, "leak-resource", "leak-resource-report.json"),
			memory100PlaceholderJSON("tetra.leak-resource.v1", memory100TestHead),
		)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected placeholder leak/resource report to fail")
	}
	if got := err.Error(); !strings.Contains(got, "leak/resource") ||
		!strings.Contains(got, "check") {
		t.Fatalf("error = %v, want leak/resource check rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsMissingSemanticSafetyMatrix(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		_ = os.Remove(filepath.Join(root, "semantic-safety", "memory-semantic-safety-matrix.json"))
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected missing semantic safety matrix to fail")
	}
	if got := err.Error(); !strings.Contains(got, "semantic") ||
		!strings.Contains(got, "memory-semantic-safety-matrix.json") {
		t.Fatalf("error = %v, want semantic safety matrix rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsMissingProofTransitionReport(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		_ = os.Remove(filepath.Join(root, "proof-transition", "proof-transition-report.json"))
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected missing proof transition report to fail")
	}
	if got := err.Error(); !strings.Contains(got, "proof_transition_report") ||
		!strings.Contains(got, "proof-transition/proof-transition-report.json") {
		t.Fatalf("error = %v, want proof transition artifact rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsMissingRuntimeMemoryContract(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		_ = os.Remove(filepath.Join(root, "runtime-memory", "runtime-memory-contract.json"))
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected missing runtime memory contract to fail")
	}
	if got := err.Error(); !strings.Contains(got, "runtime_memory_contract") ||
		!strings.Contains(got, "runtime-memory/runtime-memory-contract.json") {
		t.Fatalf("error = %v, want runtime memory contract artifact rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsInvalidatedProofTransitionWithoutRecheck(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		report := memory100JSON("tetra.proof-transition-report.v1", memory100TestHead)
		for _, raw := range report["rows"].([]any) {
			row := raw.(map[string]any)
			if row["name"] == "optimization_invalidates_bounds_proofs" {
				row["consumer_action"] = "consumed_directly"
			}
		}
		writeMemory100JSON(
			t,
			filepath.Join(root, "proof-transition", "proof-transition-report.json"),
			report,
		)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected invalidated proof transition without recheck to fail")
	}
	if got := err.Error(); !strings.Contains(got, "proof transition") ||
		!strings.Contains(got, "consumer_action") {
		t.Fatalf("error = %v, want proof transition consumer_action rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsRuntimeMemoryTargetOverclaim(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		manifest["target_matrix"] = []any{"linux-x64", "windows-x64"}
		report := memory100JSON("tetra.runtime-memory-contract.v1", memory100TestHead)
		for _, raw := range report["rows"].([]any) {
			row := raw.(map[string]any)
			if row["target"] == "windows-x64" {
				row["included_in_memory100_target_matrix"] = true
				row["runtime_status"] = "production"
				row["memory_run"] = "yes"
				row["memory_claim_level"] = "production_host_runtime"
				delete(row, "excluded_reason")
			}
		}
		writeMemory100JSON(
			t,
			filepath.Join(root, "runtime-memory", "runtime-memory-contract.json"),
			report,
		)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected runtime memory target overclaim to fail")
	}
	if got := err.Error(); !strings.Contains(got, "runtime memory") ||
		!strings.Contains(got, "windows-x64") {
		t.Fatalf("error = %v, want runtime memory windows target overclaim rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsIncompleteSemanticSafetyMatrix(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		matrix := memory100JSON("tetra.memory-semantic-safety-matrix.v1", memory100TestHead)
		rows := matrix["rows"].([]any)
		matrix["rows"] = rows[:len(rows)-1]
		writeMemory100JSON(
			t,
			filepath.Join(root, "semantic-safety", "memory-semantic-safety-matrix.json"),
			matrix,
		)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected incomplete semantic safety matrix to fail")
	}
	if got := err.Error(); !strings.Contains(got, "semantic safety matrix") ||
		!strings.Contains(got, "actor_task_non_sendable_transfer") {
		t.Fatalf("error = %v, want missing actor/task semantic row rejection", err)
	}
}

func TestValidateMemory100ReportDirRejectsPlaceholderClaimPolicy(t *testing.T) {
	reportDir := writeMemory100Fixture(t, func(root string, manifest map[string]any) {
		writeMemory100JSON(
			t,
			filepath.Join(root, "docs-manifest", "claim-policy.json"),
			memory100PlaceholderJSON("tetra.memory-100.claim-policy.v1", memory100TestHead),
		)
	})
	err := validateMemory100ReportDir(reportDir, memory100TestHead)
	if err == nil {
		t.Fatalf("expected placeholder claim policy to fail")
	}
	if got := err.Error(); !strings.Contains(got, "claim policy") ||
		!strings.Contains(got, "forbidden_claims") {
		t.Fatalf("error = %v, want claim policy forbidden_claims rejection", err)
	}
}
