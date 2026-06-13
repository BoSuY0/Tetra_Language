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
	"time"
)

const integratedTestHead = "0123456789abcdef0123456789abcdef01234567"

func TestValidateIntegratedReportDirAcceptsCompleteFixture(t *testing.T) {
	reportDir := writeIntegratedFixture(t)
	writeIntegratedHashManifest(t, reportDir, nil)

	if err := validateIntegratedReportDir(reportDir, integratedTestHead); err != nil {
		t.Fatalf("validate integrated fixture: %v", err)
	}
}

func TestValidateIntegratedReportDirRejectsMissingSafeViewLifetime(t *testing.T) {
	reportDir := writeIntegratedFixture(t)
	if err := os.Remove(filepath.Join(reportDir, "safe-view-lifetime", "safe-view-lifetime-summary.json")); err != nil {
		t.Fatalf("remove safe-view summary: %v", err)
	}
	writeIntegratedHashManifest(t, reportDir, nil)

	err := validateIntegratedReportDir(reportDir, integratedTestHead)
	if err == nil {
		t.Fatalf("expected missing safe-view lifetime evidence to fail")
	}
	if !strings.Contains(err.Error(), "safe-view lifetime summary") {
		t.Fatalf("error = %v, want safe-view lifetime summary", err)
	}
}

func TestValidateIntegratedReportDirRejectsMissingRAMContractArtifacts(t *testing.T) {
	reportDir := writeIntegratedFixture(t)
	if err := os.RemoveAll(filepath.Join(reportDir, "memory", "ram-contract")); err != nil {
		t.Fatalf("remove RAM contract bundle: %v", err)
	}
	writeIntegratedHashManifest(t, reportDir, nil)

	err := validateIntegratedReportDir(reportDir, integratedTestHead)
	if err == nil {
		t.Fatalf("expected missing RAM contract evidence to fail")
	}
	got := err.Error()
	if !strings.Contains(got, "ram_contract_report") && !strings.Contains(got, "memory/ram-contract") {
		t.Fatalf("error = %v, want RAM contract evidence rejection", err)
	}
}

func TestValidateIntegratedReportDirRejectsMismatchedGitHead(t *testing.T) {
	reportDir := writeIntegratedFixture(t)
	writeJSONFile(t, filepath.Join(reportDir, "surface-release-v1", "surface-release-summary.json"), map[string]any{
		"schema":        "tetra.surface.release.v1",
		"status":        "current",
		"git_head":      "ffffffffffffffffffffffffffffffffffffffff",
		"release_scope": "surface-v1-linux-web",
	})
	writeIntegratedHashManifest(t, reportDir, nil)

	err := validateIntegratedReportDir(reportDir, integratedTestHead)
	if err == nil {
		t.Fatalf("expected mismatched git head to fail")
	}
	if !strings.Contains(err.Error(), "surface release git_head") || !strings.Contains(err.Error(), "integrated git_head") {
		t.Fatalf("error = %v, want surface/integrated git_head mismatch", err)
	}
}

func TestValidateIntegratedReportDirRejectsMissingSanitizerEvidence(t *testing.T) {
	reportDir := writeIntegratedFixture(t)
	if err := os.Remove(filepath.Join(reportDir, "islands-debug-smoke.json")); err != nil {
		t.Fatalf("remove islands debug smoke: %v", err)
	}
	writeIntegratedHashManifest(t, reportDir, nil)

	err := validateIntegratedReportDir(reportDir, integratedTestHead)
	if err == nil {
		t.Fatalf("expected missing sanitizer evidence to fail")
	}
	if !strings.Contains(err.Error(), "islands debug smoke") {
		t.Fatalf("error = %v, want islands debug smoke", err)
	}
}

func TestValidateIntegratedReportDirRejectsMissingHashEntry(t *testing.T) {
	reportDir := writeIntegratedFixture(t)
	writeIntegratedHashManifest(t, reportDir, map[string]bool{
		"safe-view-lifetime/safe-view-lifetime-summary.json": true,
	})

	err := validateIntegratedReportDir(reportDir, integratedTestHead)
	if err == nil {
		t.Fatalf("expected missing hash entry to fail")
	}
	if !strings.Contains(err.Error(), "missing integrated hash manifest entry for safe-view-lifetime/safe-view-lifetime-summary.json") {
		t.Fatalf("error = %v, want missing safe-view hash entry", err)
	}
}

func writeIntegratedFixture(t *testing.T) string {
	t.Helper()

	reportDir := t.TempDir()
	generatedAt := time.Date(2026, 6, 8, 22, 55, 0, 0, time.UTC).Format(time.RFC3339)

	writeJSONFile(t, filepath.Join(reportDir, "memory", "memory-production-linux-x64.json"), map[string]any{
		"schema": "tetra.memory.production.v1",
		"status": "pass",
		"target": "linux-x64",
	})
	writeJSONFile(t, filepath.Join(reportDir, "memory", "memory-release-manifest.json"), map[string]any{
		"schema":        "tetra.memory.release-manifest.v1",
		"target":        "linux-x64",
		"git_head":      integratedTestHead,
		"generated_at":  generatedAt,
		"report_dir":    ".",
		"hash_manifest": "artifact-hashes.json",
	})
	writeJSONFile(t, filepath.Join(reportDir, "memory", "island-proof-verifier.json"), map[string]any{
		"schema":   "tetra.island.proof.v1",
		"git_head": integratedTestHead,
		"proofs":   []any{map[string]any{"proof_id": "proof:release:island:borrow:1"}},
	})
	writeJSONFile(t, filepath.Join(reportDir, "memory", "island-proof-memory-report.json"), map[string]any{
		"schema_version": "tetra.memory-report.v1",
		"rows":           []any{},
	})
	writeJSONFile(t, filepath.Join(reportDir, "memory", "memory-fuzz-tier1", "island-proof-fuzz-summary.json"), map[string]any{
		"schema_version": "tetra.island-proof-fuzz-summary.v1",
		"status":         "pass",
		"total":          11,
		"rejected":       11,
		"accepted":       0,
	})
	writeIntegratedRAMBundle(t, filepath.Join(reportDir, "memory", "ram-contract"))
	writeJSONFile(t, filepath.Join(reportDir, "memory", "artifact-hashes.json"), map[string]any{
		"schema":    "tetra.release-artifact-hashes.v1alpha1",
		"root":      ".",
		"artifacts": []any{map[string]any{"path": "memory-production-linux-x64.json", "sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size": 1}},
	})
	writeJSONFile(t, filepath.Join(reportDir, "islands-debug-smoke.json"), map[string]any{
		"timestamp":     generatedAt,
		"target":        "linux-x64",
		"host":          "linux-x64",
		"version":       "v0.6.0",
		"git_head":      integratedTestHead[:8],
		"islands_debug": true,
		"total":         1,
		"passed":        1,
		"failed":        0,
		"cases": []any{map[string]any{
			"name":          "islands_overflow",
			"src_path":      "examples/islands_overflow.tetra",
			"expected_exit": 1,
			"actual_exit":   1,
			"ran":           true,
			"pass":          true,
		}},
	})
	writeJSONFile(t, filepath.Join(reportDir, "surface-release-v1", "surface-release-summary.json"), map[string]any{
		"schema":        "tetra.surface.release.v1",
		"status":        "current",
		"git_head":      integratedTestHead,
		"release_scope": "surface-v1-linux-web",
	})
	writeJSONFile(t, filepath.Join(reportDir, "surface-release-v1", "artifact-hashes.json"), map[string]any{
		"schema":    "tetra.release-artifact-hashes.v1alpha1",
		"root":      ".",
		"artifacts": []any{map[string]any{"path": "surface-release-summary.json", "sha256": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "size": 1}},
	})
	writeJSONFile(t, filepath.Join(reportDir, "surface-experimental-regression", "artifact-hashes.json"), map[string]any{
		"schema":    "tetra.release-artifact-hashes.v1alpha1",
		"root":      ".",
		"artifacts": []any{map[string]any{"path": "surface-headless.json", "sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "size": 1}},
	})
	writeJSONFile(t, filepath.Join(reportDir, "safe-view-lifetime", "safe-view-lifetime-summary.json"), map[string]any{
		"schema":           "tetra.safe-view-lifetime.gate.v1",
		"status":           "pass",
		"bounded":          true,
		"release_blocking": true,
	})
	writeJSONFile(t, filepath.Join(reportDir, "surface-api-stability-v1", "surface-api-stability-summary.json"), map[string]any{
		"schema":                  "tetra.surface.api-stability.v1",
		"status":                  "pass",
		"release_scope":           "surface-v1-linux-web",
		"docs_manifest_validated": true,
	})
	writeJSONFile(t, filepath.Join(reportDir, "memory-islands-surface-production-manifest.json"), map[string]any{
		"schema":        "tetra.memory-islands-surface.production-gate.v1",
		"status":        "pass",
		"git_head":      integratedTestHead,
		"generated_at":  generatedAt,
		"report_dir":    ".",
		"hash_manifest": "artifact-hashes.json",
		"commands": []any{
			map[string]any{"name": "memory-production-gate", "command": "bash scripts/release/post_v0_4/memory-production-linux-x64-smoke.sh --report-dir reports/integrated/memory"},
			map[string]any{"name": "islands-debug-smoke", "command": "go run ./cli/cmd/tetra smoke --target linux-x64 --run=true --islands-debug --report reports/integrated/islands-debug-smoke.json"},
			map[string]any{"name": "validate-islands-debug-smoke", "command": "go run ./tools/cmd/smoke-report-to-checklist --validate-only --report reports/integrated/islands-debug-smoke.json"},
			map[string]any{"name": "surface-release-gate", "command": "bash scripts/release/surface/release-gate.sh --report-dir reports/integrated/surface-release-v1"},
			map[string]any{"name": "surface-experimental-regression-gate", "command": "bash scripts/release/surface/gate.sh --report-dir reports/integrated/surface-experimental-regression"},
			map[string]any{"name": "safe-view-lifetime-gate", "command": "bash scripts/release/safe-view-lifetime/gate.sh --report-dir reports/integrated/safe-view-lifetime"},
			map[string]any{"name": "surface-api-stability-gate", "command": "bash scripts/release/surface/api-stability-gate.sh --report-dir reports/integrated/surface-api-stability-v1"},
			map[string]any{"name": "validate-manifest", "command": "go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json"},
			map[string]any{"name": "verify-docs", "command": "go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json"},
			map[string]any{"name": "artifact-hashes-write", "command": "go run ./tools/cmd/validate-artifact-hashes --write --root reports/integrated --out reports/integrated/artifact-hashes.json"},
			map[string]any{"name": "artifact-hashes-validate", "command": "go run ./tools/cmd/validate-artifact-hashes --manifest reports/integrated/artifact-hashes.json"},
			map[string]any{"name": "integrated-release-validator", "command": "go run ./tools/cmd/validate-memory-islands-surface-production --report-dir reports/integrated --current-git-head " + integratedTestHead},
		},
		"artifacts": []any{
			map[string]any{"path": "memory/memory-production-linux-x64.json", "kind": "memory_production_report", "schema": "tetra.memory.production.v1"},
			map[string]any{"path": "memory/memory-release-manifest.json", "kind": "memory_release_manifest", "schema": "tetra.memory.release-manifest.v1"},
			map[string]any{"path": "memory/artifact-hashes.json", "kind": "memory_hash_manifest", "schema": "tetra.release-artifact-hashes.v1alpha1"},
			map[string]any{"path": "memory/island-proof-verifier.json", "kind": "island_proof_verifier_report", "schema": "tetra.island.proof.v1"},
			map[string]any{"path": "memory/island-proof-memory-report.json", "kind": "island_proof_memory_report", "schema": "tetra.memory-report.v1"},
			map[string]any{"path": "memory/memory-fuzz-tier1/island-proof-fuzz-summary.json", "kind": "island_proof_fuzz_summary", "schema": "tetra.island-proof-fuzz-summary.v1"},
			map[string]any{"path": "memory/ram-contract/ram-contract-release-manifest.json", "kind": "ram_contract_release_manifest", "schema": "tetra.ram-contract.release-manifest.v1"},
			map[string]any{"path": "memory/ram-contract/ram-contract-report.json", "kind": "ram_contract_report", "schema": "tetra.ram-contract-report.v1"},
			map[string]any{"path": "memory/ram-contract/memory-grade-report.json", "kind": "ram_memory_grade_report", "schema": "tetra.memory-grade-report.v1"},
			map[string]any{"path": "memory/ram-contract/proof-store-summary.json", "kind": "ram_proof_store_summary", "schema": "tetra.proof-store-summary.v1"},
			map[string]any{"path": "memory/ram-contract/validation-pipeline-coverage.json", "kind": "ram_validation_pipeline_coverage", "schema": "tetra.validation-pipeline-coverage.v1"},
			map[string]any{"path": "memory/ram-contract/heap-blockers.json", "kind": "ram_heap_blockers", "schema": "tetra.ram-blockers.v1"},
			map[string]any{"path": "memory/ram-contract/copy-blockers.json", "kind": "ram_copy_blockers", "schema": "tetra.ram-blockers.v1"},
			map[string]any{"path": "memory/ram-contract/fuzz/ram-contract-fuzz-oracle.json", "kind": "ram_contract_fuzz_oracle", "schema": "tetra.ram-contract-fuzz-oracle.v1"},
			map[string]any{"path": "memory/ram-contract/artifact-hashes.json", "kind": "ram_contract_hash_manifest", "schema": "tetra.release-artifact-hashes.v1alpha1"},
			map[string]any{"path": "islands-debug-smoke.json", "kind": "islands_debug_smoke_report", "schema": "tetra.release.v0_2_0.smoke-report.v1"},
			map[string]any{"path": "surface-release-v1/surface-release-summary.json", "kind": "surface_release_summary", "schema": "tetra.surface.release.v1"},
			map[string]any{"path": "surface-release-v1/artifact-hashes.json", "kind": "surface_release_hash_manifest", "schema": "tetra.release-artifact-hashes.v1alpha1"},
			map[string]any{"path": "surface-experimental-regression/artifact-hashes.json", "kind": "surface_experimental_hash_manifest", "schema": "tetra.release-artifact-hashes.v1alpha1"},
			map[string]any{"path": "safe-view-lifetime/safe-view-lifetime-summary.json", "kind": "safe_view_lifetime_summary", "schema": "tetra.safe-view-lifetime.gate.v1"},
			map[string]any{"path": "surface-api-stability-v1/surface-api-stability-summary.json", "kind": "surface_api_stability_summary", "schema": "tetra.surface.api-stability.v1"},
			map[string]any{"path": "artifact-hashes.json", "kind": "integrated_hash_manifest", "schema": "tetra.release-artifact-hashes.v1alpha1"},
		},
	})

	return reportDir
}

func writeIntegratedRAMBundle(t *testing.T, ramDir string) {
	t.Helper()
	files := map[string]any{
		"ram-contract-release-manifest.json": map[string]any{
			"schema":        "tetra.ram-contract.release-manifest.v1",
			"status":        "pass",
			"target":        "linux-x64",
			"git_head":      integratedTestHead,
			"hash_manifest": "artifact-hashes.json",
		},
		"ram-contract-report.json": map[string]any{
			"schema_version": "tetra.ram-contract-report.v1",
			"git_head":       integratedTestHead,
		},
		"memory-grade-report.json": map[string]any{
			"schema_version": "tetra.memory-grade-report.v1",
			"git_head":       integratedTestHead,
		},
		"proof-store-summary.json": map[string]any{
			"schema_version": "tetra.proof-store-summary.v1",
			"git_head":       integratedTestHead,
		},
		"validation-pipeline-coverage.json": map[string]any{
			"schema_version": "tetra.validation-pipeline-coverage.v1",
			"git_head":       integratedTestHead,
		},
		"heap-blockers.json": map[string]any{
			"schema_version": "tetra.ram-blockers.v1",
			"kind":           "heap",
			"git_head":       integratedTestHead,
		},
		"copy-blockers.json": map[string]any{
			"schema_version": "tetra.ram-blockers.v1",
			"kind":           "copy",
			"git_head":       integratedTestHead,
		},
		filepath.Join("fuzz", "ram-contract-fuzz-oracle.json"): map[string]any{
			"schema_version": "tetra.ram-contract-fuzz-oracle.v1",
			"git_head":       integratedTestHead,
		},
	}
	for rel, value := range files {
		writeJSONFile(t, filepath.Join(ramDir, filepath.FromSlash(rel)), value)
	}
	writeIntegratedHashManifest(t, ramDir, nil)
}

func writeJSONFile(t *testing.T, path string, value any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeIntegratedHashManifest(t *testing.T, root string, omit map[string]bool) {
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
		if rel == "artifact-hashes.json" || omit[rel] {
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
			Schema: detectIntegratedTestSchema(raw),
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk fixture: %v", err)
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	writeJSONFile(t, filepath.Join(root, "artifact-hashes.json"), map[string]any{
		"schema":    "tetra.release-artifact-hashes.v1alpha1",
		"root":      ".",
		"artifacts": artifacts,
	})
}

func detectIntegratedTestSchema(raw []byte) string {
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
