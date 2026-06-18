package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func writeMemory100MemoryFuzzBundle(t *testing.T, dir string, gitHead string) {
	t.Helper()
	writeMemory100JSON(t, filepath.Join(dir, "memory-fuzz-oracle.json"), map[string]any{
		"schema_version": "tetra.memory-fuzz.oracle.v1",
		"scope":          "memory-production-core-v1",
		"status":         "pass",
		"tier":           "Tier 1 short CI smoke",
		"git_head":       gitHead,
	})
	if err := os.WriteFile(
		filepath.Join(dir, "summary.md"),
		[]byte("# Memory Fuzz Short Summary\n\n- tier: `Tier 1 short CI smoke`\n- report: `memory-fuzz-oracle.json`\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	writeMemory100FuzzSummary(t, dir, gitHead, dir)
	writeMemory100JSON(t, filepath.Join(dir, "island-proof-fuzz-summary.json"), map[string]any{
		"schema_version": "tetra.island-proof-fuzz-summary.v1",
		"status":         "pass",
		"total":          11,
		"rejected":       11,
		"accepted":       0,
		"cases": []any{
			map[string]any{"name": "malformed_proof_json", "status": "rejected"},
			map[string]any{"name": "stale_epoch", "status": "rejected"},
			map[string]any{"name": "mismatched_island_id", "status": "rejected"},
			map[string]any{"name": "wrong_base_allocation", "status": "rejected"},
			map[string]any{"name": "broken_dominance", "status": "rejected"},
			map[string]any{"name": "missing_proof_id", "status": "rejected"},
			map[string]any{"name": "wrong_operation", "status": "rejected"},
			map[string]any{"name": "unsafe_unknown_promotion", "status": "rejected"},
			map[string]any{"name": "noalias_broad_proof", "status": "rejected"},
			map[string]any{"name": "storage_heap_fallback", "status": "rejected"},
			map[string]any{"name": "transform_lost_metadata", "status": "rejected"},
		},
	})
	writeMemory100MemoryFuzzReproducerDirs(t, dir)
	writeMemory100HashManifest(t, dir)
}

func writeMemory100MemoryFuzzReproducerDirs(t *testing.T, dir string) {
	t.Helper()
	for _, rel := range []string{
		"reproducers/compiler-crash",
		"reproducers/miscompile",
		"reducers/miscompile",
	} {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("create memory fuzz evidence dir %s: %v", rel, err)
		}
		if err := os.WriteFile(
			filepath.Join(path, "README.md"),
			[]byte("required release evidence slot for "+rel+"\n"),
			0o644,
		); err != nil {
			t.Fatalf("write memory fuzz evidence marker %s: %v", rel, err)
		}
	}
}

func writeMemory100MemoryReleaseManifest(t *testing.T, dir string, gitHead string) {
	t.Helper()
	writeMemory100JSON(t, filepath.Join(dir, "memory-release-manifest.json"), map[string]any{
		"schema":        "tetra.memory.release-manifest.v1",
		"target":        "linux-x64",
		"git_head":      gitHead,
		"generated_at":  "2026-06-10T11:00:00Z",
		"report_dir":    ".",
		"hash_manifest": "artifact-hashes.json",
		"commands":      memory100MemoryReleaseCommandsForDir(filepath.ToSlash(dir), gitHead),
		"artifacts":     memory100MemoryReleaseArtifactRefsForDir(filepath.ToSlash(dir), gitHead),
	})
}

func writeMemory100TargetReport(t *testing.T, dir string) {
	t.Helper()
	writeMemory100JSON(t, filepath.Join(dir, "targets.json"), map[string]any{
		"supported":  []any{"linux-x64", "windows-x64", "macos-x64", "wasm32-wasi", "wasm32-web"},
		"build_only": []any{"linux-x86", "linux-x32"},
		"planned":    []any{},
		"targets": []any{
			memory100TargetReportRow(
				"linux-x64",
				"supported",
				"linux",
				"x64",
				"sysv",
				"elf",
				"",
				false,
				"host_native",
				true,
				true,
				map[string]any{
					"run_supported":              true,
					"runtime_status":             "production",
					"stdlib_status":              "production",
					"ffi_status":                 "scalar_object_smokes_partial",
					"memory_build":               "yes",
					"memory_lower":               "yes",
					"memory_run":                 "yes",
					"memory_raw_diagnostics":     "yes",
					"memory_region_lowering":     "yes/partial",
					"memory_alignment_semantics": "yes",
					"memory_claim_level":         "production/host_runtime",
					"runner_probe_command":       "tetra test --target x64 --format=json <runner-smoke.tetra>",
					"release_gate":               "scripts/release/post_v0_4/linux-native-targets-smoke.sh",
					"evidence_artifacts": []any{
						"targets.json",
						"linux-x64-abi.json",
						"linux-x64-atomic-stress.json",
						"linux-x64-fuzz.json",
						"linux-x64-runner.json",
						"linux-native-targets-brutal.json",
						"artifact-hashes.json",
					},
					"syscall_instruction":   "syscall",
					"syscall_numbering":     "x86_64",
					"syscall_arg_registers": []any{"rax", "rdi", "rsi", "rdx", "r10", "r8", "r9"},
					"syscall_error_range":   "-4095..-1",
				},
			),
			memory100TargetReportRow(
				"windows-x64",
				"supported",
				"windows",
				"x64",
				"win64",
				"pe",
				".exe",
				false,
				"host_native",
				true,
				true,
				map[string]any{
					"run_supported":              false,
					"run_unsupported_reason":     "windows-x64 cannot run on host linux/amd64",
					"memory_build":               "yes",
					"memory_lower":               "yes",
					"memory_run":                 "host-required",
					"memory_raw_diagnostics":     "host-required",
					"memory_region_lowering":     "host-required",
					"memory_alignment_semantics": "host-required",
					"memory_claim_level":         "build_lower_only unless run",
				},
			),
			memory100TargetReportRow(
				"macos-x64",
				"supported",
				"macos",
				"x64",
				"sysv",
				"macho",
				"",
				false,
				"host_native",
				true,
				true,
				map[string]any{
					"run_supported":              false,
					"run_unsupported_reason":     "macos-x64 cannot run on host linux/amd64",
					"memory_build":               "yes",
					"memory_lower":               "yes",
					"memory_run":                 "host-required",
					"memory_raw_diagnostics":     "host-required",
					"memory_region_lowering":     "host-required",
					"memory_alignment_semantics": "host-required",
					"memory_claim_level":         "build_lower_only unless run",
				},
			),
			memory100TargetReportRow(
				"wasm32-wasi",
				"supported",
				"wasi",
				"wasm32",
				"wasi",
				"wasm",
				".wasm",
				false,
				"wasi_runner",
				false,
				true,
				map[string]any{
					"run_supported":              true,
					"run_runner":                 "wasmtime",
					"memory_build":               "yes",
					"memory_lower":               "yes",
					"memory_run":                 "runner-smoke if available",
					"memory_raw_diagnostics":     "safe-only",
					"memory_region_lowering":     "limited",
					"memory_alignment_semantics": "wasm rules",
					"memory_claim_level":         "artifact/runtime tiered",
				},
			),
			memory100TargetReportRow(
				"wasm32-web",
				"supported",
				"web",
				"wasm32",
				"web",
				"wasm",
				".wasm",
				false,
				"web_runner",
				false,
				true,
				map[string]any{
					"run_supported":              true,
					"run_runner":                 "browser",
					"memory_build":               "yes",
					"memory_lower":               "yes",
					"memory_run":                 "browser-smoke if available",
					"memory_raw_diagnostics":     "safe-only",
					"memory_region_lowering":     "limited",
					"memory_alignment_semantics": "wasm rules",
					"memory_claim_level":         "artifact/runtime tiered",
				},
			),
			memory100TargetReportRow(
				"linux-x86",
				"build_only",
				"linux",
				"x86",
				"i386-sysv",
				"elf",
				"",
				true,
				"host_probed",
				false,
				false,
				map[string]any{
					"run_supported":              true,
					"runtime_status":             "partial_build_only",
					"stdlib_status":              "partial_build_only",
					"ffi_status":                 "ilp32_scalar_object_smokes_partial",
					"memory_build":               "yes",
					"memory_lower":               "yes",
					"memory_run":                 "no/host-dependent",
					"memory_raw_diagnostics":     "partial",
					"memory_region_lowering":     "partial",
					"memory_alignment_semantics": "partial",
					"memory_claim_level":         "build_lower_only",
					"runner_probe_command": ("tetra test --diagnostics=json --target x86 --" +
						"format=json <runner-smoke.tetra>"),
					"release_gate": "scripts/release/post_v0_4/linux-native-targets-smoke.sh",
					"evidence_artifacts": []any{
						"targets.json",
						"linux-x86-abi.json",
						"linux-x86-atomic-stress.json",
						"linux-x86-fuzz.json",
						"linux-x86-runner.json",
						"linux-native-targets-brutal.json",
						"artifact-hashes.json",
					},
					"syscall_instruction": "int 0x80",
					"syscall_numbering":   "i386",
					"syscall_arg_registers": []any{
						"eax",
						"ebx",
						"ecx",
						"edx",
						"esi",
						"edi",
						"ebp",
					},
					"syscall_error_range": "-4095..-1",
				},
			),
			memory100TargetReportRow(
				"linux-x32",
				"build_only",
				"linux",
				"x64",
				"x32-sysv",
				"elf",
				"",
				true,
				"host_probed",
				false,
				false,
				map[string]any{
					"run_supported":              true,
					"runtime_status":             "partial_build_only",
					"stdlib_status":              "partial_build_only",
					"ffi_status":                 "ilp32_scalar_object_smokes_partial",
					"memory_build":               "yes",
					"memory_lower":               "yes",
					"memory_run":                 "no/host-dependent",
					"memory_raw_diagnostics":     "partial",
					"memory_region_lowering":     "partial",
					"memory_alignment_semantics": "special",
					"memory_claim_level":         "build_lower_only",
					"runner_probe_command": ("tetra test --diagnostics=json --target x32 --" +
						"format=json <runner-smoke.tetra>"),
					"release_gate": "scripts/release/post_v0_4/linux-native-targets-smoke.sh",
					"evidence_artifacts": []any{
						"targets.json",
						"linux-x32-abi.json",
						"linux-x32-atomic-stress.json",
						"linux-x32-fuzz.json",
						"linux-x32-runner.json",
						"linux-native-targets-brutal.json",
						"artifact-hashes.json",
					},
					"syscall_instruction":   "syscall",
					"syscall_numbering":     "x32_syscall_bit",
					"syscall_arg_registers": []any{"rax", "rdi", "rsi", "rdx", "r10", "r8", "r9"},
					"syscall_error_range":   "-4095..-1",
				},
			),
		},
	})
}

func memory100TargetReportRow(
	triple string,
	status string,
	osName string,
	arch string,
	abi string,
	format string,
	exeExt string,
	buildOnly bool,
	runMode string,
	supportsDebug bool,
	supportsRelease bool,
	extra map[string]any,
) map[string]any {
	row := map[string]any{
		"triple":                    triple,
		"status":                    status,
		"os":                        osName,
		"arch":                      arch,
		"abi":                       abi,
		"format":                    format,
		"exe_ext":                   exeExt,
		"build_only":                buildOnly,
		"run_mode":                  runMode,
		"supports_debug_info":       supportsDebug,
		"supports_release_optimize": supportsRelease,
	}
	for key, value := range extra {
		row[key] = value
	}
	return row
}

func memory100MemoryReleaseCommandsForDir(dir string, gitHead string) []any {
	return []any{
		map[string]any{
			"name": "memory-production-smoke",
			"command": "go run ./tools/cmd/memory-production-smoke --report " + dir + ("/memory-production-" +
				"linux-x64.json --git-head ") + gitHead,
		},
		map[string]any{
			"name":    "target-report",
			"command": "go run ./cli/cmd/tetra targets --format=json > " + dir + "/targets.json",
		},
		map[string]any{
			"name":    "validate-targets",
			"command": "go run ./tools/cmd/validate-targets --report " + dir + "/targets.json",
		},
		map[string]any{
			"name":    "memory-fuzz-short",
			"command": "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir " + dir + "/memory-fuzz-tier1 --git-head " + gitHead,
		},
		map[string]any{
			"name": "validate-memory-fuzz-oracle",
			"command": "go run ./tools/cmd/validate-memory-fuzz-oracle --report " + dir + ("/memory-fuzz-tier1/" +
				"memory-fuzz-oracle.json --artifact-dir ") + dir + ("/memory-fuzz-tier1 --" +
				"current-git-head ") + gitHead,
		},
		map[string]any{
			"name": "ram-contract-gate",
			"command": ("bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh -" +
				"-report-dir ") + dir + "/ram-contract",
		},
		map[string]any{
			"name": "island-proof-verifier",
			"command": "go run ./tools/cmd/validate-island-proof --proof " + dir + ("/island-proof-" +
				"verifier.json --memory-report ") + dir + ("/island-proof-memory-" +
				"report.json --current-git-head ") + gitHead + " --require-same-commit",
		},
		map[string]any{
			"name":    "artifact-hashes-write",
			"command": "go run ./tools/cmd/validate-artifact-hashes --write --root " + dir + " --out " + dir + "/artifact-hashes.json",
		},
		map[string]any{
			"name":    "artifact-hashes-validate",
			"command": "go run ./tools/cmd/validate-artifact-hashes --manifest " + dir + "/artifact-hashes.json",
		},
	}
}

func memory100MemoryReleaseArtifactRefsForDir(dir string, gitHead string) []any {
	var artifacts []any
	for _, required := range requiredMemory100MemoryReleaseArtifacts {
		command := memory100MemoryReleaseArtifactCommand(dir, required.Kind, gitHead)
		artifacts = append(artifacts, map[string]any{
			"path":    required.Path,
			"kind":    required.Kind,
			"schema":  required.Schema,
			"target":  "linux-x64",
			"command": command,
		})
	}
	return artifacts
}

func memory100MemoryReleaseArtifactCommand(dir string, kind string, gitHead string) string {
	switch kind {
	case "memory_production_report":
		return "go run ./tools/cmd/memory-production-smoke --report " + dir + ("/memory-production-" +
			"linux-x64.json --git-head ") + gitHead
	case "target_report":
		return "go run ./cli/cmd/tetra targets --format=json > " + dir + "/targets.json"
	case "memory_fuzz_oracle_report", "memory_fuzz_summary", "memory_fuzz_island_proof_summary":
		return "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir " + dir + "/memory-fuzz-tier1 --git-head " + gitHead
	case "island_proof_verifier_report", "island_proof_memory_report":
		return "go run ./tools/cmd/validate-island-proof --proof " + dir + ("/island-proof-" +
			"verifier.json --memory-report ") + dir + ("/island-proof-memory-" +
			"report.json --current-git-head ") + gitHead + " --require-same-commit"
	case "artifact_hash_manifest":
		return "go run ./tools/cmd/validate-artifact-hashes --write --root " + dir + " --out " + dir + "/artifact-hashes.json"
	default:
		return ("bash scripts/release/post_v0_4/ram-contract-linux-x64-smoke.sh -" +
			"-report-dir ") + dir + "/ram-contract"
	}
}

func writeMemory100RAMContractReleaseManifest(t *testing.T, dir string, gitHead string) {
	t.Helper()
	reportPath := func(rel string) string {
		if rel == "" {
			return filepath.ToSlash(dir)
		}
		return filepath.ToSlash(filepath.Join(dir, filepath.FromSlash(rel)))
	}
	commands := []any{
		map[string]any{
			"name": "ram-contract-build",
			"command": ("go run ./cli/cmd/tetra build --target linux-x64 --emit-ram-" +
				"contract-report --emit-memory-report --emit-alloc-report -o fixture.o " +
				"fixture.tetra"),
		},
		map[string]any{
			"name": "validate-ram-contract-report",
			"command": "go run ./tools/cmd/validate-ram-contract-report --report " + reportPath(
				"ram-contract-report.json",
			),
		},
		map[string]any{
			"name": "validate-memory-grade-report",
			"command": "go run ./tools/cmd/validate-memory-grade-report --report " + reportPath(
				"memory-grade-report.json",
			),
		},
		map[string]any{
			"name": "validate-proof-store-summary",
			"command": "go run ./tools/cmd/validate-proof-store-summary --report " + reportPath(
				"proof-store-summary.json",
			),
		},
		map[string]any{
			"name": "validate-validation-pipeline-coverage",
			"command": "go run ./tools/cmd/validate-validation-pipeline-coverage --report " + reportPath(
				"validation-pipeline-coverage.json",
			),
		},
		map[string]any{
			"name": "validate-heap-blockers",
			"command": "go run ./tools/cmd/validate-heap-blockers --report " + reportPath(
				"heap-blockers.json",
			),
		},
		map[string]any{
			"name": "validate-copy-blockers",
			"command": "go run ./tools/cmd/validate-copy-blockers --report " + reportPath(
				"copy-blockers.json",
			),
		},
		map[string]any{
			"name": "ram-contract-fuzz-short",
			"command": "go run ./tools/cmd/ram-contract-fuzz-short --report-dir " + reportPath(
				"fuzz",
			) + " --git-head " + gitHead,
		},
		map[string]any{
			"name": "validate-ram-contract-fuzz-oracle",
			"command": "go run ./tools/cmd/validate-ram-contract-fuzz-oracle --report " + reportPath(
				"fuzz/ram-contract-fuzz-oracle.json",
			) + " --current-git-head " + gitHead,
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
			"name": "artifact-hashes-validate",
			"command": "go run ./tools/cmd/validate-artifact-hashes --manifest " + reportPath(
				"artifact-hashes.json",
			),
		},
		map[string]any{
			"name": "ram-contract-release-validator",
			"command": "go run ./tools/cmd/validate-ram-contract-release --report-dir " + reportPath(
				"",
			) + " --current-git-head " + gitHead,
		},
	}
	var artifacts []any
	for _, required := range requiredMemory100RAMContractReleaseArtifacts {
		artifacts = append(artifacts, map[string]any{
			"path":   required.Path,
			"kind":   required.Kind,
			"schema": required.Schema,
		})
	}
	writeMemory100JSON(t, filepath.Join(dir, "ram-contract-release-manifest.json"), map[string]any{
		"schema":        "tetra.ram-contract.release-manifest.v1",
		"status":        "pass",
		"target":        "linux-x64",
		"git_head":      gitHead,
		"generated_at":  "2026-06-10T11:00:00Z",
		"report_dir":    ".",
		"hash_manifest": "artifact-hashes.json",
		"commands":      commands,
		"artifacts":     artifacts,
		"non_claims": []any{
			"no Memory 100% claim",
			"no full formal proof claim",
			"no official benchmark or fastest-language claim",
			"local Linux-x64 scoped RAM contract evidence only",
		},
	})
}

func writeMemory100IntegratedBundle(t *testing.T, dir string, gitHead string) {
	t.Helper()
	memoryDir := filepath.Join(dir, "memory")
	writeMemory100JSON(
		t,
		filepath.Join(memoryDir, "memory-production-linux-x64.json"),
		memory100ArtifactJSON("memory_production_report", "tetra.memory.production.v1", gitHead),
	)
	writeMemory100MemoryReleaseManifest(t, memoryDir, gitHead)
	writeMemory100TargetReport(t, memoryDir)
	writeMemory100IntegratedIslandProofEvidence(t, memoryDir, gitHead)
	writeMemory100MemoryFuzzBundle(t, filepath.Join(memoryDir, "memory-fuzz-tier1"), gitHead)

	ramDir := filepath.Join(memoryDir, "ram-contract")
	writeMemory100RAMContractArtifacts(t, ramDir, gitHead)
	writeMemory100RAMContractReleaseManifest(t, ramDir, gitHead)
	writeMemory100HashManifest(t, ramDir)
	writeMemory100HashManifest(t, memoryDir)

	writeMemory100JSON(t, filepath.Join(dir, "islands-debug-smoke.json"), map[string]any{
		"schema":        "tetra.release.v0_2_0.smoke-report.v1",
		"target":        "linux-x64",
		"git_head":      gitHead,
		"islands_debug": true,
		"total":         1,
		"passed":        1,
		"failed":        0,
		"cases": []any{
			map[string]any{
				"name":          "islands_overflow",
				"src_path":      "examples/memory/islands/islands_overflow.tetra",
				"expected_exit": 1,
				"actual_exit":   1,
				"ran":           true,
				"pass":          true,
			},
		},
	})
	writeMemory100JSON(
		t,
		filepath.Join(dir, "surface-release-v1", "surface-release-summary.json"),
		map[string]any{
			"schema":        "tetra.surface.release.v1",
			"status":        "current",
			"git_head":      gitHead,
			"release_scope": "surface-v1-linux-web",
		},
	)
	writeMemory100HashManifest(t, filepath.Join(dir, "surface-release-v1"))
	writeMemory100JSON(
		t,
		filepath.Join(dir, "surface-experimental-regression", "summary.json"),
		map[string]any{
			"schema":   "tetra.surface.experimental-regression.v1",
			"status":   "pass",
			"git_head": gitHead,
		},
	)
	writeMemory100HashManifest(t, filepath.Join(dir, "surface-experimental-regression"))
	writeMemory100JSON(
		t,
		filepath.Join(dir, "safe-view-lifetime", "safe-view-lifetime-summary.json"),
		map[string]any{
			"schema":           "tetra.safe-view-lifetime.gate.v1",
			"status":           "pass",
			"bounded":          true,
			"release_blocking": true,
		},
	)
	writeMemory100JSON(
		t,
		filepath.Join(dir, "surface-api-stability-v1", "surface-api-stability-summary.json"),
		map[string]any{
			"schema":                  "tetra.surface.api-stability.v1",
			"status":                  "pass",
			"release_scope":           "surface-v1-linux-web",
			"docs_manifest_validated": true,
		},
	)

	writeMemory100JSON(
		t,
		filepath.Join(dir, "memory-islands-surface-production-manifest.json"),
		map[string]any{
			"schema":        "tetra.memory-islands-surface.production-gate.v1",
			"status":        "pass",
			"git_head":      gitHead,
			"generated_at":  "2026-06-10T11:00:00Z",
			"report_dir":    ".",
			"hash_manifest": "artifact-hashes.json",
			"commands":      memory100IntegratedCommandsForDir(filepath.ToSlash(dir), gitHead),
			"artifacts":     memory100IntegratedArtifactRefs(),
		},
	)
	writeMemory100HashManifest(t, dir)
}

func writeMemory100RAMContractArtifacts(t *testing.T, dir string, gitHead string) {
	t.Helper()
	ramArtifacts := []struct {
		Path   string
		Kind   string
		Schema string
	}{
		{
			"ram-contract-release-manifest.json",
			"ram_contract_release_manifest",
			"tetra.ram-contract.release-manifest.v1",
		},
		{"ram-contract-report.json", "ram_contract_report", "tetra.ram-contract-report.v1"},
		{"memory-grade-report.json", "ram_memory_grade_report", "tetra.memory-grade-report.v1"},
		{"proof-store-summary.json", "ram_proof_store_summary", "tetra.proof-store-summary.v1"},
		{
			"validation-pipeline-coverage.json",
			"ram_validation_pipeline_coverage",
			"tetra.validation-pipeline-coverage.v1",
		},
		{"heap-blockers.json", "ram_heap_blockers", "tetra.ram-blockers.v1"},
		{"copy-blockers.json", "ram_copy_blockers", "tetra.ram-blockers.v1"},
		{
			"fuzz/ram-contract-fuzz-oracle.json",
			"ram_contract_fuzz_oracle",
			"tetra.ram-contract-fuzz-oracle.v1",
		},
	}
	for _, artifact := range ramArtifacts {
		writeMemory100JSON(
			t,
			filepath.Join(dir, filepath.FromSlash(artifact.Path)),
			memory100ArtifactJSON(artifact.Kind, artifact.Schema, gitHead),
		)
	}
}

func memory100IntegratedCommandsForDir(dir string, gitHead string) []any {
	return []any{
		map[string]any{
			"name": "memory-production-gate",
			"command": ("bash scripts/release/post_v0_4/memory-production-linux-x64-" +
				"smoke.sh --report-dir ") + dir + "/memory",
		},
		map[string]any{
			"name": "islands-debug-smoke",
			"command": ("go run ./cli/cmd/tetra smoke --target linux-x64 --run=true --" +
				"islands-debug --report ") + dir + "/islands-debug-smoke.json",
		},
		map[string]any{
			"name": "validate-islands-debug-smoke",
			"command": ("go run ./tools/cmd/smoke-report-to-checklist --validate-only --" +
				"report ") + dir + "/islands-debug-smoke.json",
		},
		map[string]any{
			"name":    "surface-release-gate",
			"command": "bash scripts/release/surface/release-gate.sh --report-dir " + dir + "/surface-release-v1",
		},
		map[string]any{
			"name":    "surface-experimental-regression-gate",
			"command": "bash scripts/release/surface/gate.sh --report-dir " + dir + "/surface-experimental-regression",
		},
		map[string]any{
			"name":    "safe-view-lifetime-gate",
			"command": "bash scripts/release/safe-view-lifetime/gate.sh --report-dir " + dir + "/safe-view-lifetime",
		},
		map[string]any{
			"name":    "surface-api-stability-gate",
			"command": "bash scripts/release/surface/api-stability-gate.sh --report-dir " + dir + "/surface-api-stability-v1",
		},
		map[string]any{
			"name":    "validate-manifest",
			"command": "go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json",
		},
		map[string]any{
			"name":    "verify-docs",
			"command": "go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
		},
		map[string]any{
			"name":    "artifact-hashes-write",
			"command": "go run ./tools/cmd/validate-artifact-hashes --write --root " + dir + " --out " + dir + "/artifact-hashes.json",
		},
		map[string]any{
			"name":    "artifact-hashes-validate",
			"command": "go run ./tools/cmd/validate-artifact-hashes --manifest " + dir + "/artifact-hashes.json",
		},
		map[string]any{
			"name": "integrated-release-validator",
			"command": ("go run ./tools/cmd/validate-memory-islands-surface-production --" +
				"report-dir ") + dir + " --current-git-head " + gitHead,
		},
	}
}

func memory100IntegratedArtifactRefs() []any {
	var artifacts []any
	for _, required := range requiredMemory100IntegratedArtifacts {
		artifacts = append(artifacts, map[string]any{
			"path":   required.Path,
			"kind":   required.Kind,
			"schema": required.Schema,
		})
	}
	return artifacts
}

func writeMemory100IntegratedIslandProofEvidence(t *testing.T, memoryDir string, gitHead string) {
	t.Helper()
	writeMemory100JSON(t, filepath.Join(memoryDir, "island-proof-verifier.json"), map[string]any{
		"schema":           "tetra.island.proof.v1",
		"producer":         "tools/validators/islandproof/test-fixture",
		"producer_command": "go run ./tools/cmd/validate-island-proof",
		"git_head":         gitHead,
		"generated_at":     "2026-06-10T11:00:00Z",
		"proofs": []any{
			map[string]any{
				"proof_id":                "proof:test:island:borrow:1",
				"operation":               "island_borrow",
				"proof_kind":              "island_epoch",
				"subject_base_id":         "alloc:test:island:0",
				"island_id":               "island:test:0",
				"epoch":                   1,
				"source_fact_id":          "fact:test:island-proof:1",
				"claim":                   "island_proof_verified",
				"provenance_class":        "safe_known",
				"unsafe_class":            "safe",
				"validator_name":          "validate-island-proof",
				"validator_status":        "pass",
				"planned_storage":         "ExplicitIsland",
				"actual_lowering_storage": "ExplicitIsland",
				"dominance":               "entry dominates test island borrow",
				"distinct_live_islands":   []any{"island:test:0", "island:test:1"},
			},
		},
	})
	writeMemory100JSON(
		t,
		filepath.Join(memoryDir, "island-proof-memory-report.json"),
		map[string]any{
			"schema_version": "tetra.memory-report.v1",
			"rows": []any{
				map[string]any{
					"site_id":                 "island:test:borrow:1",
					"source_fact_id":          "fact:test:island-proof:1",
					"claim":                   "island_proof_verified",
					"claim_level":             "validated",
					"provenance_class":        "safe_known",
					"unsafe_class":            "safe",
					"alias_state":             "unique",
					"island_id":               "island:test:0",
					"epoch":                   1,
					"base_id":                 "alloc:test:island:0",
					"proof_id":                "proof:test:island:borrow:1",
					"proof_kind":              "island_epoch",
					"proof_subject_base_id":   "alloc:test:island:0",
					"proof_operation":         "island_borrow",
					"planned_storage":         "ExplicitIsland",
					"actual_lowering_storage": "ExplicitIsland",
					"validator_name":          "validate-island-proof",
					"validator_status":        "pass",
				},
			},
		},
	)
}

func writeMemory100FuzzSummary(t *testing.T, dir string, gitHead string, commandDir string) {
	t.Helper()
	commandDir = filepath.ToSlash(commandDir)
	writeMemory100JSON(t, filepath.Join(dir, "summary.json"), map[string]any{
		"schema_version":            "tetra.memory-fuzz-short.summary.v1",
		"kind":                      "tier1_short_ci_smoke",
		"tier":                      "tier1_short_ci_smoke",
		"status":                    "pass",
		"observed_failures":         0,
		"classified_failures":       0,
		"unclassified_failures":     0,
		"release_blocking_failures": 0,
		"reproducibility_seeds": []string{
			"memory-fuzz:v0:seed:1000",
			"memory-fuzz:v1:seed:1001",
			"memory-fuzz:v2:seed:1002",
			"memory-fuzz:v3:seed:1003",
			"memory-fuzz:v4:seed:1004",
			"memory-fuzz:v5:seed:1005",
			"memory-fuzz:v6:seed:1006",
			"memory-fuzz:v7:seed:1007",
			"memory-fuzz:v8:seed:1008",
			"memory-fuzz:v9:seed:1009",
			"memory-fuzz:v10:seed:1010",
			"memory-fuzz:v11:seed:1011",
		},
		"artifacts": map[string]any{
			"artifact_hashes":           "artifact-hashes.json",
			"island_proof_fuzz_summary": "island-proof-fuzz-summary.json",
			"oracle_report":             "memory-fuzz-oracle.json",
			"summary_md":                "summary.md",
			"summary_json":              "summary.json",
		},
		"commands": []any{
			map[string]any{
				"name":    "memory-fuzz-short",
				"command": "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir " + commandDir + " --git-head " + gitHead,
				"status":  "pass",
			},
			map[string]any{
				"name": "validate-memory-fuzz-oracle",
				"command": "go run ./tools/cmd/validate-memory-fuzz-oracle --report " + commandDir + ("/memory-fuzz-" +
					"oracle.json --artifact-dir ") + commandDir + " --current-git-head " + gitHead,
				"status": "pass",
			},
		},
	})
}

func memory100PlaceholderJSON(schema string, gitHead string) map[string]any {
	key := "schema"
	if strings.Contains(schema, "report") || strings.Contains(schema, "summary") ||
		strings.Contains(schema, "oracle") ||
		strings.Contains(schema, "coverage") ||
		strings.Contains(schema, "blockers") {
		key = "schema_version"
	}
	return map[string]any{
		key:        schema,
		"status":   "pass",
		"git_head": gitHead,
	}
}

func writeMemory100JSON(t *testing.T, path string, value any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeMemory100HashManifest(t *testing.T, root string) {
	t.Helper()
	type artifact struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
		Size   int64  `json:"size"`
		Schema string `json:"schema,omitempty"`
	}
	var artifacts []artifact
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "artifact-hashes.json" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(raw)
		artifacts = append(artifacts, artifact{
			Path:   rel,
			SHA256: "sha256:" + hex.EncodeToString(sum[:]),
			Size:   int64(len(raw)),
			Schema: memory100TestSchema(raw),
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	writeMemory100JSON(t, filepath.Join(root, "artifact-hashes.json"), map[string]any{
		"schema":    "tetra.release-artifact-hashes.v1alpha1",
		"root":      ".",
		"artifacts": artifacts,
	})
}

func memory100TestSchema(raw []byte) string {
	var envelope struct {
		Schema        string `json:"schema"`
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return ""
	}
	if envelope.Schema != "" {
		return envelope.Schema
	}
	return envelope.SchemaVersion
}
