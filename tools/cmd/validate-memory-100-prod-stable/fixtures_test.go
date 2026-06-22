package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeMemory100Fixture(t *testing.T, mutate func(root string, manifest map[string]any)) string {
	t.Helper()
	root := t.TempDir()
	generatedAt := time.Date(2026, 6, 10, 11, 0, 0, 0, time.UTC).Format(time.RFC3339)

	required := []struct {
		Path   string
		Kind   string
		Schema string
	}{
		{
			"memory-production/memory-production-linux-x64.json",
			"memory_production_report",
			"tetra.memory.production.v1",
		},
		{
			"memory-production/memory-release-manifest.json",
			"memory_release_manifest",
			"tetra.memory.release-manifest.v1",
		},
		{
			"memory-production/artifact-hashes.json",
			"memory_production_hash_manifest",
			"tetra.release-artifact-hashes.v1alpha1",
		},
		{
			"ram-contract/ram-contract-release-manifest.json",
			"ram_contract_release_manifest",
			"tetra.ram-contract.release-manifest.v1",
		},
		{
			"ram-contract/ram-contract-report.json",
			"ram_contract_report",
			"tetra.ram-contract-report.v1",
		},
		{
			"ram-contract/memory-grade-report.json",
			"ram_memory_grade_report",
			"tetra.memory-grade-report.v1",
		},
		{
			"ram-contract/proof-store-summary.json",
			"ram_proof_store_summary",
			"tetra.proof-store-summary.v1",
		},
		{
			"ram-contract/validation-pipeline-coverage.json",
			"ram_validation_pipeline_coverage",
			"tetra.validation-pipeline-coverage.v1",
		},
		{"ram-contract/heap-blockers.json", "ram_heap_blockers", "tetra.ram-blockers.v1"},
		{"ram-contract/copy-blockers.json", "ram_copy_blockers", "tetra.ram-blockers.v1"},
		{
			"ram-contract/fuzz/ram-contract-fuzz-oracle.json",
			"ram_contract_fuzz_oracle",
			"tetra.ram-contract-fuzz-oracle.v1",
		},
		{
			"ram-contract/artifact-hashes.json",
			"ram_contract_hash_manifest",
			"tetra.release-artifact-hashes.v1alpha1",
		},
		{
			"raw-memory-contract/raw-memory-contract.json",
			"raw_memory_contract_report",
			"tetra.raw-memory-contract.v1",
		},
		{
			"allocation-lowering/allocation-lowering-report.json",
			"allocation_lowering_report",
			"tetra.allocation-lowering.v1",
		},
		{
			"proof-store/proof-store-summary.json",
			"proof_store_summary",
			"tetra.proof-store-summary.v1",
		},
		{
			"proof-transition/proof-transition-report.json",
			"proof_transition_report",
			"tetra.proof-transition-report.v1",
		},
		{
			"runtime-memory/runtime-memory-contract.json",
			"runtime_memory_contract",
			"tetra.runtime-memory-contract.v1",
		},
		{
			"memory-fuzz/memory-fuzz-oracle.json",
			"memory_fuzz_oracle_report",
			"tetra.memory-fuzz.oracle.v1",
		},
		{
			"memory-fuzz/artifact-hashes.json",
			"memory_fuzz_hash_manifest",
			"tetra.release-artifact-hashes.v1alpha1",
		},
		{
			"semantic-safety/memory-semantic-safety-matrix.json",
			"memory_semantic_safety_matrix",
			"tetra.memory-semantic-safety-matrix.v1",
		},
		{
			"leak-resource/leak-resource-report.json",
			"leak_resource_report",
			"tetra.leak-resource.v1",
		},
		{
			"integrated/memory-islands-surface-production-manifest.json",
			"integrated_memory_islands_surface_manifest",
			"tetra.memory-islands-surface.production-gate.v1",
		},
		{
			"integrated/artifact-hashes.json",
			"integrated_hash_manifest",
			"tetra.release-artifact-hashes.v1alpha1",
		},
		{
			"docs-manifest/claim-policy.json",
			"docs_claim_policy",
			"tetra.memory-100.claim-policy.v1",
		},
	}

	var artifactRefs []any
	for _, req := range required {
		writeMemory100JSON(
			t,
			filepath.Join(root, filepath.FromSlash(req.Path)),
			memory100ArtifactJSON(req.Kind, req.Schema, memory100TestHead),
		)
		artifactRefs = append(artifactRefs, map[string]any{
			"path":   req.Path,
			"kind":   req.Kind,
			"schema": req.Schema,
		})
	}
	writeMemory100RAMContractReleaseManifest(
		t,
		filepath.Join(root, "ram-contract"),
		memory100TestHead,
	)
	writeMemory100HashManifest(t, filepath.Join(root, "ram-contract"))
	memoryProductionDir := filepath.Join(root, "memory-production")
	writeMemory100MemoryReleaseManifest(t, memoryProductionDir, memory100TestHead)
	writeMemory100TargetReport(t, memoryProductionDir)
	writeMemory100MemoryFuzzBundle(
		t,
		filepath.Join(memoryProductionDir, "memory-fuzz-tier1"),
		memory100TestHead,
	)
	writeMemory100IntegratedIslandProofEvidence(t, memoryProductionDir, memory100TestHead)
	writeMemory100RAMContractArtifacts(
		t,
		filepath.Join(memoryProductionDir, "ram-contract"),
		memory100TestHead,
	)
	writeMemory100RAMContractReleaseManifest(
		t,
		filepath.Join(memoryProductionDir, "ram-contract"),
		memory100TestHead,
	)
	writeMemory100HashManifest(t, filepath.Join(memoryProductionDir, "ram-contract"))
	writeMemory100HashManifest(t, memoryProductionDir)
	writeMemory100IntegratedBundle(t, filepath.Join(root, "integrated"), memory100TestHead)
	writeMemory100MemoryFuzzBundle(t, filepath.Join(root, "memory-fuzz"), memory100TestHead)
	reportPath := func(rel string) string {
		if rel == "" {
			return filepath.ToSlash(root)
		}
		return filepath.ToSlash(filepath.Join(root, filepath.FromSlash(rel)))
	}

	manifest := map[string]any{
		"schema":    "tetra.memory-100.prod-stable.v1",
		"status":    "pass",
		"verdict":   "MEMORY100_SCOPED_READY_LOCAL",
		"git_head":  memory100TestHead,
		"git_dirty": false,
		"git_status_short_branch": []any{
			"## main",
		},
		"generated_at":  generatedAt,
		"target_matrix": []any{"linux-x64"},
		"hash_manifest": "artifact-hashes.json",
		"claims": []any{
			"Memory/RAM production-stable criteria passed locally for the scoped target matrix.",
		},
		"non_claims": []any{
			"no universal Memory 100% claim",
			"no full formal proof claim",
			"no all-target memory parity claim",
			"no arbitrary unsafe external pointer safety claim",
			"no C/Rust parity or performance superiority claim",
		},
		"commands": []any{
			map[string]any{
				"name": "memory-production-gate",
				"command": ("bash scripts/release/post_v0_4/memory-production-linux-x64-" +
					"smoke.sh --report-dir ") + reportPath(
					"memory-production",
				),
			},
			map[string]any{
				"name": "ram-contract-gate",
				"command": ("bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh -" +
					"-report-dir ") + reportPath(
					"ram-contract",
				),
			},
			map[string]any{
				"name": "integrated-gate",
				"command": ("bash scripts/release/post_v0_4/memory-islands-surface-" +
					"production-gate.sh --report-dir ") + reportPath(
					"integrated",
				),
			},
			map[string]any{
				"name": "memory-fuzz-short",
				"command": "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir " + reportPath(
					"memory-fuzz",
				) + " --git-head " + memory100TestHead,
			},
			map[string]any{
				"name": "memory-fuzz-validator",
				"command": "go run ./tools/cmd/validate-memory-fuzz-oracle --report " + reportPath(
					"memory-fuzz/memory-fuzz-oracle.json",
				) + " --artifact-dir " + reportPath(
					"memory-fuzz",
				) + " --current-git-head " + memory100TestHead,
			},
			map[string]any{
				"name":    "docs-claim-policy",
				"command": "go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
			},
			map[string]any{
				"name": "artifact-hashes-write",
				"command": "go run ./tools/cmd/validate-artifact-hashes --write --root " + reportPath(
					"",
				) + " --out " + reportPath(
					"artifact-hashes.json",
				),
			},
			map[string]any{
				"name": "memory-100-validator",
				"command": "go run ./tools/cmd/validate-memory-100-prod-stable --report-dir " + reportPath(
					"",
				) + " --current-git-head " + memory100TestHead,
			},
		},
		"artifacts": artifactRefs,
	}
	if mutate != nil {
		mutate(root, manifest)
	}
	writeMemory100JSON(t, filepath.Join(root, "memory-100-prod-stable-manifest.json"), manifest)
	writeMemory100HashManifest(t, root)
	return root
}

func memory100ArtifactJSON(kind string, schema string, gitHead string) map[string]any {
	switch kind {
	case "memory_production_report":
		caseNames := []string{
			"allocator alloc/free lifecycle",
			"allocator failure semantics",
			"allocator invalid size precondition",
			"cap.mem unsafe boundary",
			"memcpy/memset capability path",
			"runtime bounds check",
			"raw ptr_add negative offset bounds",
			"raw ptr_add allocation upper bound",
			"raw allocation-base i32 access width",
			"raw allocation-base ptr access width",
			"raw slice negative length",
			"raw slice i32 length byte overflow",
			"raw pointer bounds metadata report",
			"memcpy/memset negative length",
			"reject use-after-free",
			"reject double-free",
			"reject borrow escape",
			"reject aliasing violation",
			"callable mutable capture heap escape",
			"reject actor task transfer violation",
			"heap closure handle coverage",
			"slice struct borrow escape coverage",
			"function-typed slice aggregate borrow escape coverage",
			"actornet broker close-without-cancel leak smoke",
			"compiler resource finalization diagnostics",
			"real memory examples",
			"stress allocator reuse",
			"deterministic memcpy/memset fuzz",
		}
		var cases []any
		for _, name := range caseNames {
			kind := "positive"
			expected := ""
			lower := strings.ToLower(name)
			if strings.Contains(lower, "stress") || strings.Contains(lower, "fuzz") ||
				strings.Contains(lower, "leak smoke") ||
				strings.Contains(lower, "diagnostics") {
				kind = "stress"
			}
			if strings.Contains(lower, "reject") || strings.Contains(lower, "negative") ||
				strings.Contains(lower, "bounds") ||
				strings.Contains(lower, "overflow") ||
				strings.Contains(lower, "unsafe") ||
				strings.Contains(lower, "invalid") ||
				strings.Contains(lower, "failure") {
				kind = "negative"
				expected = "TETRA_MEMORY_CONTRACT"
			}
			row := map[string]any{"name": name, "kind": kind, "ran": true, "pass": true}
			if expected != "" {
				row["expected_error"] = expected
			}
			cases = append(cases, row)
		}
		contractNames := []string{
			"allocator runtime model",
			"allocator failure semantics",
			"ownership escape model",
			"unsafe cap.mem raw memory rules",
			"runtime bounds diagnostics",
			"raw pointer bounds metadata",
			"host resource leak and finalization checks",
			"actor task transfer rules",
		}
		var contracts []any
		for _, name := range contractNames {
			contracts = append(
				contracts,
				map[string]any{
					"name":     name,
					"status":   "pass",
					"evidence": "scoped linux-x64 release evidence",
				},
			)
		}
		auditRequirements := []string{
			"stable allocator/runtime memory model",
			"ownership/borrow/consume escape model",
			"heap, slices, structs, and closures memory coverage",
			"unsafe/cap.mem/raw memory/memcpy/memset rules",
			"runtime bounds checks and diagnostics",
			"raw pointer bounds metadata",
			"stress/fuzz evidence",
			"measured memory benchmark improvement",
			"allocator benchmark evidence classification",
			"use-after-free, double-free, borrow escape, and aliasing safety",
			"actor/task transfer safety",
			"leak/resource finalization evidence",
			"real memory examples",
			"safe memory documentation",
			"release-gate entrypoint",
		}
		var audit []any
		for _, requirement := range auditRequirements {
			audit = append(
				audit,
				map[string]any{
					"requirement": requirement,
					"artifact":    "memory-production-linux-x64.json",
					"evidence":    "scoped linux-x64 release evidence",
					"result":      "pass",
				},
			)
		}
		exitZero := 0
		return map[string]any{
			"schema":   schema,
			"status":   "pass",
			"git_head": gitHead,
			"target":   "linux-x64",
			"host":     "linux-x64",
			"runtime":  "memory-linux-x64",
			"source":   "memory-production-linux-x64-smoke.sh",
			"processes": []any{
				map[string]any{
					"name":      "memory production build",
					"kind":      "build",
					"path":      "./compiler",
					"ran":       true,
					"pass":      true,
					"exit_code": exitZero,
				},
				map[string]any{
					"name":      "memory production app smoke",
					"kind":      "app",
					"path":      "./examples/memory_ownership_demo.tetra",
					"ran":       true,
					"pass":      true,
					"exit_code": exitZero,
				},
				map[string]any{
					"name":      "actornet close-without-cancel leak coverage",
					"kind":      "stress",
					"path":      "./cli/internal/actornet TestBrokerCloseWithoutCancelStopsServeWatcher",
					"ran":       true,
					"pass":      true,
					"exit_code": exitZero,
				},
				map[string]any{
					"name": "compiler resource finalization diagnostics",
					"kind": "stress",
					"path": ("./compiler/tests/runtime TestTaskHandleFinalization " +
						"TestTaskGroupFinalization TestIslandFinalization"),
					"ran":       true,
					"pass":      true,
					"exit_code": exitZero,
				},
			},
			"benchmarks": []any{
				map[string]any{
					"name":              "small heap allocation syscall reduction",
					"kind":              "allocator",
					"metric":            "estimated_os_syscalls",
					"unit":              "syscalls",
					"evidence_class":    "allocation_report_estimate",
					"method":            "allocation_report_summary",
					"baseline_value":    100,
					"measured_value":    50,
					"improvement_ratio": 2.0,
					"evidence": ("allocation report schema v2 estimates process_bump_small_heap_v0 " +
						"allocation intents and 64KiB chunk refill syscalls; allocation_report_" +
						"estimate only, not a runtime measurement or reuse claim"),
					"ran":  true,
					"pass": true,
				},
			},
			"contracts": contracts,
			"cases":     cases,
			"audit":     audit,
		}
	case "memory_release_manifest":
		return map[string]any{
			"schema":        schema,
			"target":        "linux-x64",
			"git_head":      gitHead,
			"generated_at":  "2026-06-10T11:00:00Z",
			"report_dir":    ".",
			"hash_manifest": "artifact-hashes.json",
			"commands":      []any{},
			"artifacts":     []any{},
		}
	case "ram_contract_report":
		return map[string]any{
			"schema_version": schema,
			"git_head":       gitHead,
			"target":         "linux-x64",
			"generated_by":   "test",
			"rows": []any{
				map[string]any{
					"site_id":           "site:main:heap",
					"value_id":          "heap",
					"function":          "main",
					"intent":            "heap_fallback",
					"requested_bytes":   8192,
					"bounded":           false,
					"owner":             "function:main",
					"lifetime":          "function:main",
					"escape_status":     "unknown",
					"placement":         "heap_unbounded",
					"proof_ids":         []any{},
					"blockers":          []any{"unknown_size"},
					"contract_grade":    "M5",
					"validation_status": "conservative",
				},
			},
			"proofs": []any{},
			"summary": map[string]any{
				"row_count":      1,
				"artifact_grade": "M5",
				"heap_rows":      1,
				"copy_rows":      0,
				"unbounded_rows": 1,
				"budget_bytes":   8192,
			},
			"non_claims": []any{"no Memory 100% claim", "no full formal proof claim"},
		}
	case "ram_memory_grade_report":
		return map[string]any{
			"schema_version": schema,
			"git_head":       gitHead,
			"target":         "linux-x64",
			"generated_by":   "test",
			"artifact_grade": "M5",
			"functions":      []any{},
			"summary": map[string]any{
				"row_count":      1,
				"artifact_grade": "M5",
				"heap_rows":      1,
				"copy_rows":      0,
				"unbounded_rows": 1,
				"budget_bytes":   8192,
			},
			"non_claims": []any{"no Memory 100% claim"},
		}
	case "ram_proof_store_summary", "proof_store_summary":
		return map[string]any{
			"schema_version": schema,
			"git_head":       gitHead,
			"target":         "linux-x64",
			"generated_by":   "test",
			"proofs":         []any{},
			"summary": map[string]any{
				"proof_count":  0,
				"proven":       0,
				"conservative": 0,
				"rejected":     0,
				"unknown":      0,
			},
			"non_claims": []any{"no full formal proof claim"},
		}
	case "ram_validation_pipeline_coverage":
		return map[string]any{
			"schema_version": schema,
			"git_head":       gitHead,
			"target":         "linux-x64",
			"generated_by":   "test",
			"entries": []any{
				map[string]any{
					"entrypoint":    "BuildFileWithStatsOpt",
					"artifact_path": "ram-contract-fixture",
					"status":        "validated_by_pipeline",
					"validators":    []any{"ramcontract.ValidateReport"},
				},
				map[string]any{
					"entrypoint": "buildObjectFileWithStatsOpt",
					"status":     "formal_exemption_with_reason",
					"exemption": ("not exercised by this linux-x64 RAM release fixture; object " +
						"builds must carry their own RAM coverage evidence"),
				},
				map[string]any{
					"entrypoint": "buildLibraryObjectWithStatsOpt",
					"status":     "formal_exemption_with_reason",
					"exemption": ("not exercised by this linux-x64 RAM release fixture; library " +
						"builds must carry their own RAM coverage evidence"),
				},
				map[string]any{
					"entrypoint": "InterfaceOnly",
					"status":     "formal_exemption_with_reason",
					"exemption":  "interface-only mode does not produce a RAM artifact in this release fixture",
				},
				map[string]any{
					"entrypoint": "wasm32-wasi-build",
					"status":     "formal_exemption_with_reason",
					"exemption": ("wasm32-wasi RAM coverage is target-specific and not claimed by " +
						"this linux-x64 release fixture"),
				},
				map[string]any{
					"entrypoint": "wasm32-web-build",
					"status":     "formal_exemption_with_reason",
					"exemption": ("wasm32-web RAM coverage is target-specific and not claimed by " +
						"this linux-x64 release fixture"),
				},
				map[string]any{
					"entrypoint": "explain-report-path",
					"status":     "formal_exemption_with_reason",
					"exemption":  "explain report path is not artifact-producing in this release fixture",
				},
			},
			"non_claims": []any{"pipeline coverage is not proof completeness"},
		}
	case "ram_heap_blockers":
		return map[string]any{
			"schema_version": schema,
			"kind":           "heap",
			"git_head":       gitHead,
			"target":         "linux-x64",
			"generated_by":   "test",
			"rows": []any{
				map[string]any{
					"site_id":                "site:main:heap",
					"function":               "main",
					"intent":                 "heap_fallback",
					"placement":              "heap_unbounded",
					"blockers":               []any{"unknown_size"},
					"contract_grade":         "M5",
					"file":                   "fixtures/main.tetra",
					"line":                   3,
					"symbol":                 "main",
					"source_location_status": "available",
					"severity":               "P1",
					"reason":                 "unknown_size",
					"suggested_fix": ("add no-escape, lifetime, or bounded allocation proof " +
						"before changing this heap fallback"),
					"evidence_id":      "fact:ram:site:main:heap",
					"safe_to_optimize": false,
				},
			},
			"non_claims": []any{"no Memory 100% claim"},
		}
	case "ram_copy_blockers":
		return map[string]any{
			"schema_version": schema,
			"kind":           "copy",
			"git_head":       gitHead,
			"target":         "linux-x64",
			"generated_by":   "test",
			"rows":           []any{},
			"non_claims":     []any{"no Memory 100% claim"},
		}
	case "ram_contract_fuzz_oracle":
		mutations := []string{
			"mutated_proof_id",
			"widened_grade",
			"missing_blocker",
			"budget_drift",
			"artifact_hash_drift",
			"forbidden_nonclaim_text",
		}
		var observations []any
		for _, mutation := range mutations {
			observations = append(observations, map[string]any{
				"mutation":          mutation,
				"rejected":          true,
				"validator":         "validate-" + strings.ReplaceAll(mutation, "_", "-"),
				"validator_command": "go run ./tools/cmd/validate-ram-contract-fuzz-oracle --test-fixture",
				"exit_code":         1,
				"output_excerpt":    "fixture rejected as expected",
				"mutated_file":      "mutations/" + mutation + "/ram-contract-report.json",
				"reason":            mutation + " rejected by validator with exit code 1",
			})
		}
		return map[string]any{
			"schema_version": schema,
			"git_head":       gitHead,
			"generated_at":   "2026-06-10T11:00:00Z",
			"observations":   observations,
			"summary": map[string]any{
				"mutations": len(observations),
				"rejected":  len(observations),
			},
			"non_claims": []any{
				"not Memory 100%",
				"not a full formal proof",
				"not a performance benchmark",
			},
		}
	}
	return memory100JSON(schema, gitHead)
}

func memory100JSON(schema string, gitHead string) map[string]any {
	if strings.HasPrefix(schema, "tetra.release-artifact-hashes.") {
		return map[string]any{
			"schema": schema,
			"root":   ".",
			"artifacts": []any{
				map[string]any{
					"path":   "placeholder.json",
					"sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					"size":   2,
				},
			},
		}
	}
	switch schema {
	case "tetra.raw-memory-contract.v1":
		return map[string]any{
			"schema":   schema,
			"status":   "pass",
			"git_head": gitHead,
			"operations": []any{
				map[string]any{
					"name": "core.alloc_bytes",
					"source_artifacts": []any{
						"compiler/internal/runtimeabi/runtimeabi_test/raw_pointer_bounds_test.go",
						"memory-production/memory-production-linux-x64.json",
					},
					"positive_tests": []any{"allocation-base metadata"},
				},
				map[string]any{
					"name": "core.ptr_add",
					"source_artifacts": []any{
						"compiler/internal/runtimeabi/runtimeabi_test/raw_pointer_bounds_test.go",
						"memory-production/memory-production-linux-x64.json",
					},
					"negative_tests": []any{
						"negative offset",
						"allocation upper bound",
						"access-width overflow",
					},
				},
				map[string]any{
					"name": "raw_slice_from_parts",
					"source_artifacts": []any{
						"compiler/internal/runtimeabi/runtimeabi_test/raw_pointer_bounds_test.go",
						"compiler/tests/semantics/semantics_memory_surface_test.go",
						"memory-production/memory-production-linux-x64.json",
					},
					"negative_tests": []any{
						"outside unsafe",
						"negative length",
						"i32 byte overflow",
					},
				},
				map[string]any{
					"name": "raw_load_store_metadata",
					"source_artifacts": []any{
						"compiler/internal/plir/plir_test/plir_test.go",
						"compiler/internal/lower/lower_suite_test.go",
						"compiler/internal/memoryfacts_test/from_plir_test.go",
						"memory-production/memory-production-linux-x64.json",
					},
					"positive_tests": []any{
						"IRMemWriteI32Offset",
						"IRMemReadI32Offset",
						"core.store_u8/core.load_u8 raw memory gateway UnsafeChecked",
					},
					"negative_tests": []any{
						"checked_external_unknown raw store/load remains conservative",
						"rejected_access_width_overflow raw load/store width rejection",
					},
					"non_claims": []any{"no arbitrary external pointer safety claim"},
				},
				map[string]any{
					"name": "memcpy_u8",
					"source_artifacts": []any{
						"lib/core/memory/memory.tetra",
						"compiler/internal/lower/lower_suite_test.go",
						"memory-production/memory-production-linux-x64.json",
					},
					"positive_tests": []any{"cap.mem helper path"},
					"negative_tests": []any{"negative length", "access-width overflow"},
					"non_claims":     []any{"no overlapping memcpy safety claim"},
				},
				map[string]any{
					"name": "memset_u8",
					"source_artifacts": []any{
						"lib/core/memory/memory.tetra",
						"compiler/internal/lower/lower_suite_test.go",
						"memory-production/memory-production-linux-x64.json",
					},
					"positive_tests": []any{"cap.mem helper path"},
					"negative_tests": []any{"negative length", "access-width overflow"},
				},
				map[string]any{
					"name": "cap.mem",
					"source_artifacts": []any{
						"lib/core/base/capability.tetra",
						"compiler/internal/ramcontract/validate_test.go",
						"memory-production/memory-production-linux-x64.json",
					},
					"negative_tests": []any{
						"unsafe_unknown promotion rejected",
						"cap.mem overclaim rejected",
					},
					"non_claims": []any{"no arbitrary external pointer safety claim"},
				},
			},
		}
	case "tetra.allocation-lowering.v1":
		return map[string]any{
			"schema":   schema,
			"status":   "pass",
			"git_head": gitHead,
			"decisions": []any{
				map[string]any{
					"name":                    "stack_trusted_no_escape",
					"status":                  "not_observed",
					"planned_storage":         "Stack",
					"actual_lowering_storage": "Stack",
					"source_artifacts":        []any{"ram-contract/ram-contract-report.json"},
				},
				map[string]any{
					"name":                    "heap_fallback_blocker",
					"status":                  "blocked",
					"planned_storage":         "Heap",
					"actual_lowering_storage": "Heap",
					"blocker_artifact":        "ram-contract/heap-blockers.json",
					"blocker_reason": ("conservative heap fallback remains explicit until no-" +
						"escape/lifetime proof is available"),
					"budget_impact": ("heap rows and budget bytes are accounted in ram-" +
						"contract/memory-grade-report.json"),
					"grade_impact": ("heap fallback rows keep conservative RAM grade instead " +
						"of trusted storage overclaim"),
					"validator_coverage": []any{
						"validate-heap-blockers",
						"validate-ram-contract-release",
					},
					"source_artifacts": []any{
						"ram-contract/heap-blockers.json",
						"ram-contract/ram-contract-report.json",
						"ram-contract/memory-grade-report.json",
					},
					"covered_site_ids": []any{"site:main:heap"},
				},
				map[string]any{
					"name":                    "copy_blocker",
					"status":                  "not_observed",
					"planned_storage":         "Copy",
					"actual_lowering_storage": "Copy",
					"source_artifacts": []any{
						"ram-contract/copy-blockers.json",
						"ram-contract/ram-contract-report.json",
					},
				},
				map[string]any{
					"name":                    "lowering_storage_match",
					"status":                  "not_observed",
					"planned_storage":         "ExplicitIsland",
					"actual_lowering_storage": "ExplicitIsland",
					"source_artifacts":        []any{"ram-contract/ram-contract-report.json"},
				},
			},
		}
	case "tetra.leak-resource.v1":
		return map[string]any{
			"schema":   schema,
			"status":   "pass",
			"git_head": gitHead,
			"checks": []any{
				map[string]any{
					"name":             "actornet_close_without_cancel",
					"kind":             "stress",
					"evidence":         "actornet broker close-without-cancel leak smoke",
					"source_artifacts": []any{"memory-production/memory-production-linux-x64.json"},
				},
				map[string]any{
					"name":             "compiler_resource_finalization",
					"kind":             "negative",
					"evidence":         "compiler resource finalization diagnostics",
					"source_artifacts": []any{"memory-production/memory-production-linux-x64.json"},
				},
				map[string]any{
					"name":     "surface_frame_escape",
					"kind":     "negative",
					"evidence": "safe-view lifetime and Surface frame escape evidence",
					"source_artifacts": []any{
						"integrated/memory-islands-surface-production-manifest.json",
					},
				},
				map[string]any{
					"name":             "actor_task_transfer",
					"kind":             "negative",
					"evidence":         "actor task transfer safety case",
					"source_artifacts": []any{"memory-production/memory-production-linux-x64.json"},
				},
			},
		}
	case "tetra.memory-semantic-safety-matrix.v1":
		return map[string]any{
			"schema":   schema,
			"status":   "pass",
			"git_head": gitHead,
			"rows": []any{
				map[string]any{
					"name":     "borrowed_view_return_escape",
					"kind":     "negative",
					"evidence": "borrowed view escape via return is rejected",
					"source_artifacts": []any{
						"compiler/tests/ownership/ownership_test.go",
						"compiler/tests/semantics/semantics_async_ownership_test.go",
					},
					"tests": []any{
						("go test ./compiler/tests/ownership ./compiler/tests/semantics -" +
							"run 'Borrow.*Return|BorrowEscape' -count=1"),
					},
				},
				map[string]any{
					"name": "borrowed_view_owned_aggregate_escape",
					"kind": "negative",
					"evidence": ("borrowed slice/ptr fields cannot escape through owned " +
						"aggregate returns, consume, or inout calls"),
					"source_artifacts": []any{"compiler/tests/ownership/ownership_test.go"},
					"tests": []any{
						("go test ./compiler/tests/ownership -run " +
							"'Borrowed.*Aggregate|BorrowedSlice.*ConsumeInout|BorrowedPtr.*Struct' -" +
							"count=1"),
					},
				},
				map[string]any{
					"name": "borrowed_text_host_boundary_copy",
					"kind": "negative",
					"evidence": ("borrowed text/view host and actor/task boundaries require " +
						"explicit copy or are rejected"),
					"source_artifacts": []any{
						"compiler/tests/semantics/semantics_memory_surface_test.go",
						"compiler/internal/actorsafety/sendability_test.go",
					},
					"tests": []any{
						("go test ./compiler/tests/semantics ./compiler/internal/" +
							"actorsafety -run 'Borrowed|StringView|copy|ActorBoundary' -count=1"),
					},
				},
				map[string]any{
					"name":             "inout_alias_escape",
					"kind":             "negative",
					"evidence":         "borrowed values cannot escape through inout assignment or aliasing",
					"source_artifacts": []any{"compiler/tests/ownership/ownership_test.go"},
					"tests": []any{
						"go test ./compiler/tests/ownership -run 'Inout|Alias|BorrowedProjection' -count=1",
					},
				},
				map[string]any{
					"name":     "surface_frame_escape",
					"kind":     "negative",
					"evidence": "Surface frame/pixels borrowed views cannot escape lifecycle boundaries",
					"source_artifacts": []any{
						"compiler/tests/semantics/semantics_memory_surface_test.go",
						"integrated/safe-view-lifetime/safe-view-lifetime-summary.json",
					},
					"tests": []any{
						"go test ./compiler/tests/semantics -run 'Surface|Frame|Pixels|SafeView' -count=1",
					},
				},
				map[string]any{
					"name":     "use_after_present_close",
					"kind":     "negative",
					"evidence": "use after present/close/free is rejected",
					"source_artifacts": []any{
						"compiler/tests/runtime/resource_finalization_test.go",
						"compiler/tests/ownership/ownership_test.go",
					},
					"tests": []any{
						("go test ./compiler/tests/runtime ./compiler/tests/ownership -" +
							"run 'Present|Close|UseAfter|Freed|Consume' -count=1"),
					},
				},
				map[string]any{
					"name": "resource_finalizer_double_close",
					"kind": "negative",
					"evidence": ("resource finalization diagnostics reject missing finalizer and " +
						"double-close/double-free cases"),
					"source_artifacts": []any{
						"compiler/tests/runtime/resource_finalization_test.go",
						"compiler/tests/safety/diagnostics/core/safety_diagnostics_test.go",
					},
					"tests": []any{
						("go test ./compiler/tests/runtime ./compiler/tests/safety/... -" +
							"run 'Resource|Finalization|Double|Close|Free' -count=1"),
					},
				},
				map[string]any{
					"name": "actor_task_non_sendable_transfer",
					"kind": "negative",
					"evidence": ("actor/task message transfer rejects borrowed or non-sendable " +
						"memory/resources unless explicitly copied/moved"),
					"source_artifacts": []any{
						"compiler/tests/ownership/actor_task/actor_task_ownership_test.go",
						"compiler/internal/actorsafety/sendability_test.go",
						"compiler/internal/actorsafety/ownership_transfer_test.go",
					},
					"tests": []any{
						("go test ./compiler/tests/ownership ./compiler/tests/ownership/" +
							"actor_task ./compiler/internal/actorsafety -run " +
							"'Actor|Task|Send|Transfer|Borrowed|NonSendable' -count=1"),
					},
				},
			},
			"non_claims": []any{
				"no production actor runtime claim",
				"no universal leak-free program claim",
				"no full formal memory safety proof claim",
			},
		}
	case "tetra.proof-transition-report.v1":
		return map[string]any{
			"schema":   schema,
			"status":   "pass",
			"git_head": gitHead,
			"rows": []any{
				map[string]any{
					"name":       "stable_hash_semantic_fields",
					"transition": "invalidated",
					"evidence": ("StableHash includes semantic dominance/lifetime/epoch/" +
						"invalidation/consumer fields and stale semantic mutation is blocked."),
					"consumer_action": "blocked_by_proof_store_validate_recheck_required",
					"source_artifacts": []any{
						"compiler/internal/proof/term.go",
						"compiler/internal/proof/validate_test.go",
					},
					"tests": []any{
						("go test ./compiler/internal/proof -run " +
							"TestProofStoreRejectsStaleStableHashForSemanticFields -count=1"),
					},
				},
				map[string]any{
					"name":       "bounds_proof_preserved_through_translation",
					"transition": "preserved",
					"evidence": ("translation validation compares proof facts and preserves " +
						"supported bounds proof evidence."),
					"before_artifact": "compiler/compiler_evidence_gates.go",
					"after_artifact":  "compiler/compiler_suite_test.go",
					"source_artifacts": []any{
						"compiler/compiler_evidence_gates.go",
						"compiler/compiler_suite_test.go",
					},
					"tests": []any{
						("go test ./compiler -run " +
							"TestP23TranslationValidationV2CoversSupportedOptimizerSubset -count=1"),
					},
				},
				map[string]any{
					"name":       "translation_missing_proof_requires_recheck",
					"transition": "requires_recheck",
					"evidence": ("missing proof id after transform is rejected and requires " +
						"recheck before unchecked use."),
					"consumer_action":  "recheck_or_block_unchecked_bounds_use",
					"source_artifacts": []any{"compiler/internal/validation/validation_test.go"},
					"tests": []any{
						("go test ./compiler/internal/validation -run " +
							"TestValidateTranslationRejectsMissingProofIDAfterTransform -count=1"),
					},
				},
				map[string]any{
					"name":       "optimization_invalidates_bounds_proofs",
					"transition": "invalidated",
					"evidence": ("optimizer proof rules require invalidated bounds facts to be " +
						"declared, and consumers must recheck before reuse."),
					"consumer_action": "recheck_required_before_consuming_invalidated_bounds_proof",
					"source_artifacts": []any{
						"compiler/internal/opt/opt_core.go",
						"compiler/internal/opt/opt_suite_test.go",
					},
					"tests": []any{
						"go test ./compiler/internal/opt -run 'Manager|Optimization' -count=1",
					},
				},
				map[string]any{
					"name":       "lowering_refines_bounds_proof_use",
					"transition": "refined",
					"evidence": ("lowering refines live bounds proof use into proof-tagged " +
						"unchecked load metadata."),
					"before_artifact": "compiler/internal/plir/plir_test/plir_test.go",
					"after_artifact":  "compiler/internal/lower/lower_suite_test.go",
					"source_artifacts": []any{
						"compiler/internal/plir/plir_test/plir_test.go",
						"compiler/internal/lower/lower_suite_test.go",
					},
					"tests": []any{
						("go test ./compiler/internal/plir ./compiler/internal/lower -run " +
							"'Proof|Invalidates|Unchecked' -count=1"),
					},
				},
				map[string]any{
					"name":       "new_proof_requires_store_reference",
					"transition": "new",
					"evidence": ("new proof use requires a proof store reference and unknown " +
						"proof ids are blocked."),
					"after_artifact": "compiler/internal/validation/validation_test.go",
					"source_artifacts": []any{
						"compiler/internal/validation/validation_test.go",
						"compiler/internal/proof/validate_test.go",
					},
					"tests": []any{
						("go test ./compiler/internal/validation ./compiler/internal/" +
							"proof -run " +
							"'UnknownLiveProof|MissingProofID|ProofStoreRejectsMissingProofID' -" +
							"count=1"),
					},
				},
			},
			"non_claims": []any{
				"no full formal proof claim",
				"no exhaustive optimizer proof-transition completeness claim",
			},
		}
	case "tetra.runtime-memory-contract.v1":
		return map[string]any{
			"schema":   schema,
			"status":   "pass",
			"git_head": gitHead,
			"rows": []any{
				map[string]any{
					"target":                              "linux-x64",
					"included_in_memory100_target_matrix": true,
					"runtime_status":                      "production",
					"memory_run":                          "yes",
					"memory_claim_level":                  "production_host_runtime",
					"evidence": ("linux-x64 runtime hardening and runtimeabi " +
						"memory evidence is covered by Memory100 gate."),
					"source_artifacts": []any{
						"memory-production/targets.json",
						"compiler/compiler_evidence_gates.go",
						"compiler/internal/runtimeabi/runtimeabi_test/runtimeabi_test.go",
					},
					"tests": []any{
						("go test ./compiler -run " +
							"'RuntimeHardening|RuntimeAllocation|RawPointerBoundsABI|ActorRuntimeProd" +
							"uctionBoundary|OOM|Stack|Allocator|Region' -count=1"),
					},
					"non_claims": []any{
						"no all-target memory parity claim",
						"no full runtime-hardening proof claim",
					},
				},
				map[string]any{
					"target":                              "windows-x64",
					"included_in_memory100_target_matrix": false,
					"runtime_status":                      "host_required",
					"memory_run":                          "host-required",
					"memory_claim_level":                  "host_required_nonclaim",
					"evidence": ("windows-x64 requires target-host runtime " +
						"evidence before Memory100 inclusion."),
					"excluded_reason": ("no windows target-host runtime memory " +
						"report in this aggregate"),
					"source_artifacts": []any{"memory-production/targets.json"},
					"tests": []any{
						"go test ./tools/cmd/validate-memory-100-prod-stable -run RuntimeMemory -count=1",
					},
					"non_claims": []any{
						"no windows-x64 runtime memory production claim",
					},
				},
				map[string]any{
					"target":                              "macos-x64",
					"included_in_memory100_target_matrix": false,
					"runtime_status":                      "host_required",
					"memory_run":                          "host-required",
					"memory_claim_level":                  "host_required_nonclaim",
					"evidence": ("macos-x64 requires target-host runtime " +
						"evidence before Memory100 inclusion."),
					"excluded_reason": ("no macos target-host runtime memory report " +
						"in this aggregate"),
					"source_artifacts": []any{"memory-production/targets.json"},
					"tests": []any{
						"go test ./tools/cmd/validate-memory-100-prod-stable -run RuntimeMemory -count=1",
					},
					"non_claims": []any{
						"no macos-x64 runtime memory production claim",
					},
				},
				map[string]any{
					"target":                              "wasm32-wasi",
					"included_in_memory100_target_matrix": false,
					"runtime_status":                      "tiered",
					"memory_run":                          "runner-smoke if available",
					"memory_claim_level":                  "artifact_runtime_tiered_nonclaim",
					"evidence": ("wasm32-wasi remains artifact/runtime tiered " +
						"and is not Memory100 production-host-runtime evidence."),
					"excluded_reason":  "not part of current Memory100 linux-x64 target matrix",
					"source_artifacts": []any{"memory-production/targets.json"},
					"tests": []any{
						"go test ./tools/cmd/validate-memory-100-prod-stable -run RuntimeMemory -count=1",
					},
					"non_claims": []any{
						"no wasm32-wasi production host-runtime memory claim",
					},
				},
				map[string]any{
					"target":                              "wasm32-web",
					"included_in_memory100_target_matrix": false,
					"runtime_status":                      "tiered",
					"memory_run":                          "browser-smoke if available",
					"memory_claim_level":                  "artifact_runtime_tiered_nonclaim",
					"evidence": ("wasm32-web remains artifact/runtime tiered " +
						"and is not Memory100 production-host-runtime evidence."),
					"excluded_reason":  "not part of current Memory100 linux-x64 target matrix",
					"source_artifacts": []any{"memory-production/targets.json"},
					"tests": []any{
						"go test ./tools/cmd/validate-memory-100-prod-stable -run RuntimeMemory -count=1",
					},
					"non_claims": []any{
						"no wasm32-web production host-runtime memory claim",
					},
				},
				map[string]any{
					"target":                              "linux-x86",
					"included_in_memory100_target_matrix": false,
					"runtime_status":                      "partial_build_only",
					"memory_run":                          "no/host-dependent",
					"memory_claim_level":                  "build_lower_only_nonclaim",
					"evidence": ("linux-x86 remains build/lower scoped for " +
						"Memory100 and is not production runtime evidence."),
					"excluded_reason": ("build/lower-only memory evidence is not " +
						"Memory100 production-host-runtime evidence"),
					"source_artifacts": []any{"memory-production/targets.json"},
					"tests": []any{
						"go test ./tools/cmd/validate-memory-100-prod-stable -run RuntimeMemory -count=1",
					},
					"non_claims": []any{
						"no linux-x86 production runtime memory claim",
					},
				},
				map[string]any{
					"target":                              "linux-x32",
					"included_in_memory100_target_matrix": false,
					"runtime_status":                      "partial_build_only",
					"memory_run":                          "no/host-dependent",
					"memory_claim_level":                  "build_lower_only_nonclaim",
					"evidence": ("linux-x32 remains build/lower scoped for " +
						"Memory100 and is not production runtime evidence."),
					"excluded_reason": ("build/lower-only memory evidence is not " +
						"Memory100 production-host-runtime evidence"),
					"source_artifacts": []any{"memory-production/targets.json"},
					"tests": []any{
						"go test ./tools/cmd/validate-memory-100-prod-stable -run RuntimeMemory -count=1",
					},
					"non_claims": []any{
						"no linux-x32 production runtime memory claim",
					},
				},
			},
			"non_claims": []any{
				"no all-target memory parity claim",
				"OOM recovery guarantee is not claimed",
				"full stack-overflow protection is not claimed",
				"full allocator-corruption detection proof is not claimed",
				"production actor runtime is not claimed",
			},
		}
	case "tetra.memory-100.claim-policy.v1":
		return map[string]any{
			"schema":   schema,
			"status":   "pass",
			"git_head": gitHead,
			"allowed_claims": []any{
				"Memory/RAM production-stable criteria passed locally for the scoped target matrix only.",
			},
			"forbidden_claims": []any{
				"Memory is 100% ready",
				"fully proven memory safety",
				"full formal proof of memory safety",
				"all targets memory-stable",
				"all-target memory parity",
				"unsafe/raw memory is safe",
				"no leaks",
			},
			"non_claims": []any{
				"no universal Memory 100% claim",
				"no full formal proof claim",
			},
		}
	}
	return memory100PlaceholderJSON(schema, gitHead)
}
