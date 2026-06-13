package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSurfaceFinalReadinessAcceptsBlockedPartialWithoutRequireCI(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceFinalReadinessFixture(t, dir, nil)

	err := validateSurfaceFinalReadiness(surfaceFinalReadinessOptions{
		ReportDir:      dir,
		ExpectedScope:  surfaceFinalReadinessScope,
		RequireClean:   true,
		RequirePackage: true,
	})
	if err != nil {
		t.Fatalf("validateSurfaceFinalReadiness failed: %v", err)
	}
}

func TestValidateSurfaceFinalReadinessRejectsRequireCIWhenActionsDisabled(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceFinalReadinessFixture(t, dir, nil)

	err := validateSurfaceFinalReadiness(surfaceFinalReadinessOptions{
		ReportDir:      dir,
		ExpectedScope:  surfaceFinalReadinessScope,
		RequireClean:   true,
		RequireCI:      true,
		RequirePackage: true,
	})
	if err == nil {
		t.Fatalf("expected require-ci to reject disabled Actions proof")
	}
	if !strings.Contains(err.Error(), "ci_proof_status") {
		t.Fatalf("error = %v, want ci_proof_status diagnostic", err)
	}
}

func TestValidateSurfaceFinalReadinessRejectsProductionClaimWithoutFinalSignoff(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceFinalReadinessFixture(t, dir, func(report map[string]any) {
		report["production_claim"] = true
	})

	err := validateSurfaceFinalReadiness(surfaceFinalReadinessOptions{
		ReportDir:     dir,
		ExpectedScope: surfaceFinalReadinessScope,
	})
	if err == nil {
		t.Fatalf("expected production claim without final signoff to fail")
	}
	if !strings.Contains(err.Error(), "production_claim") {
		t.Fatalf("error = %v, want production_claim diagnostic", err)
	}
}

func TestWriteSurfaceFinalReadinessUsesCleanProductAndActionsDisabled(t *testing.T) {
	root := t.TempDir()
	productDir := filepath.Join(root, "product")
	finalDir := filepath.Join(root, "final")
	writeSurfaceFinalProductFixture(t, productDir)
	writeJSONFile(t, filepath.Join(root, "actions-permissions.json"), map[string]any{
		"enabled":              false,
		"sha_pinning_required": false,
	})
	writeJSONFile(t, filepath.Join(root, "gh-runs.json"), []any{})
	clean := false

	err := validateSurfaceFinalReadiness(surfaceFinalReadinessOptions{
		ReportDir:              finalDir,
		ProductReportDir:       productDir,
		ExpectedScope:          surfaceFinalReadinessScope,
		Write:                  true,
		ActionsPermissionsPath: filepath.Join(root, "actions-permissions.json"),
		CIRunsPath:             filepath.Join(root, "gh-runs.json"),
		CurrentGitHead:         "0123456789abcdef0123456789abcdef01234567",
		GitDirty:               &clean,
	})
	if err != nil {
		t.Fatalf("write final readiness: %v", err)
	}

	writeSurfaceFinalHashManifest(t, finalDir)
	err = validateSurfaceFinalReadiness(surfaceFinalReadinessOptions{
		ReportDir:      finalDir,
		ExpectedScope:  surfaceFinalReadinessScope,
		RequireClean:   true,
		RequirePackage: true,
	})
	if err != nil {
		t.Fatalf("validate written final readiness: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(finalDir, "final-readiness.json"))
	if err != nil {
		t.Fatalf("read final-readiness.json: %v", err)
	}
	var report map[string]any
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode final-readiness.json: %v", err)
	}
	if report["ci_proof_status"] != "blocked_actions_disabled" {
		t.Fatalf("ci_proof_status = %v, want blocked_actions_disabled", report["ci_proof_status"])
	}
	if report["final_signoff"] != false {
		t.Fatalf("final_signoff = %v, want false", report["final_signoff"])
	}
}

func writeSurfaceFinalReadinessFixture(t *testing.T, dir string, mutate func(map[string]any)) {
	t.Helper()
	report := map[string]any{
		"schema":                     "tetra.surface.final-readiness.v1",
		"release_scope":              surfaceFinalReadinessScope,
		"status":                     "blocked_final_requirements",
		"producer":                   "tools/cmd/validate-surface-final-readiness",
		"git_head":                   "0123456789abcdef0123456789abcdef01234567",
		"git_dirty":                  false,
		"product_report_dir":         "reports/surface-product-v1",
		"product_summary":            "product-summary.json",
		"product_summary_git_head":   "0123456789abcdef0123456789abcdef01234567",
		"product_gate_status":        "product_gate_passed_p29_final_audit_required",
		"product_final_verdict":      "P29_FINAL_AUDIT_REQUIRED",
		"product_final_signoff":      false,
		"clean_same_commit_required": true,
		"clean_same_commit_proven":   true,
		"ci_required_gate":           true,
		"ci_proof_required":          true,
		"ci_proof_status":            "blocked_actions_disabled",
		"actions_enabled":            false,
		"package_proof_required":     true,
		"package_proof_status":       "validated",
		"package_report":             "surface-package.json",
		"artifact_hash_manifest":     "artifact-hashes.json",
		"production_claim":           false,
		"final_signoff":              false,
		"nonclaims":                  surfaceFinalReadinessNonclaims(),
		"blockers": []any{
			"remote-ci-actions-disabled",
		},
	}
	if mutate != nil {
		mutate(report)
	}
	writeJSONFile(t, filepath.Join(dir, "final-readiness.json"), report)
	writeSurfaceFinalHashManifest(t, dir)
}

func writeSurfaceFinalProductFixture(t *testing.T, dir string) {
	t.Helper()
	writeJSONFile(t, filepath.Join(dir, "product-summary.json"), map[string]any{
		"schema":           "tetra.surface.product-summary.v1",
		"release_scope":    surfaceFinalReadinessScope,
		"status":           "product_gate_passed_p29_final_audit_required",
		"git_head":         "0123456789abcdef0123456789abcdef01234567",
		"git_dirty":        false,
		"final_verdict":    "P29_FINAL_AUDIT_REQUIRED",
		"production_claim": false,
		"final_signoff":    false,
		"required_artifacts": map[string]any{
			"package": "package/package-summary.json",
		},
	})
	writeJSONFile(t, filepath.Join(dir, "package", "package-summary.json"), map[string]any{
		"schema": "tetra.surface.product-category-summary.v1",
	})
	writeJSONFile(t, filepath.Join(dir, "surface-package.json"), map[string]any{
		"schema": "tetra.surface.package.v1",
	})
}

func writeSurfaceFinalHashManifest(t *testing.T, dir string) {
	t.Helper()
	writeJSONFile(t, filepath.Join(dir, "artifact-hashes.json"), map[string]any{
		"schema": "tetra.release-artifact-hashes.v1alpha1",
		"root":   ".",
		"artifacts": []any{
			map[string]any{
				"path":   "final-readiness.json",
				"sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"size":   1,
			},
		},
	})
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
