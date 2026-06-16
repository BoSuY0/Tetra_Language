package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSurfaceProductSummaryAcceptsP29Contract(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceProductSummaryFixture(t, dir, nil, nil)

	err := validateSurfaceProductSummary(surfaceProductSummaryOptions{ReportDir: dir})
	if err != nil {
		t.Fatalf("validateSurfaceProductSummary failed: %v", err)
	}
}

func TestValidateSurfaceProductSummaryRejectsMissingP29Nonclaim(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceProductSummaryFixture(t, dir, func(summary map[string]any) {
		summary["nonclaims"] = []any{
			"all-platform-surface-parity",
			"nonclaim-macos-surface-production-support",
			"nonclaim-windows-surface-production-support",
			"wasm32-wasi-surface-ui-runtime",
			"gpu-renderer",
			"full-rich-text-editor",
			"full-screen-reader-support",
			"official-benchmark-superiority",
			"electron-api-compatibility",
			"react-api-compatibility",
			"css-cascade-compatibility",
			"user-javascript-application-logic",
		}
	}, nil)

	err := validateSurfaceProductSummary(surfaceProductSummaryOptions{ReportDir: dir})
	if err == nil {
		t.Fatalf("expected missing DOM-authored UI nonclaim to fail")
	}
	if !strings.Contains(err.Error(), "dom-authored-application-ui") {
		t.Fatalf("error = %v, want missing DOM nonclaim diagnostic", err)
	}
}

func TestValidateSurfaceProductSummaryRejectsInnerReleaseSummaryAsFinalSignoff(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceProductSummaryFixture(t, dir, func(summary map[string]any) {
		summary["canonical_final_readiness_report"] = false
		summary["release_gate_report_final_signoff"] = true
	}, nil)

	err := validateSurfaceProductSummary(surfaceProductSummaryOptions{ReportDir: dir})
	if err == nil {
		t.Fatalf("expected non-canonical final readiness summary to fail")
	}
	if !strings.Contains(err.Error(), "canonical_final_readiness_report") ||
		!strings.Contains(err.Error(), "release_gate_report_final_signoff") {
		t.Fatalf("error = %v, want canonical/final-signoff diagnostics", err)
	}
}

func TestValidateSurfaceProductSummaryRejectsMissingHashCoveredArtifact(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceProductSummaryFixture(t, dir, nil, func(artifacts []map[string]any) []map[string]any {
		var kept []map[string]any
		for _, artifact := range artifacts {
			if artifact["path"] == "visual/visual-summary.json" {
				continue
			}
			kept = append(kept, artifact)
		}
		return kept
	})

	err := validateSurfaceProductSummary(surfaceProductSummaryOptions{ReportDir: dir})
	if err == nil {
		t.Fatalf("expected missing visual summary hash coverage to fail")
	}
	if !strings.Contains(err.Error(), "visual/visual-summary.json") {
		t.Fatalf("error = %v, want missing visual summary hash diagnostic", err)
	}
}

func TestValidateSurfaceProductSummaryRejectsUnexpectedTargetMatrixEntry(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceProductSummaryFixture(t, dir, func(summary map[string]any) {
		summary["target_matrix"] = append(surfaceProductSummaryTargetMatrixFixture(), map[string]any{
			"target":           "electron-desktop",
			"status":           "current",
			"tier":             "unsupported-extra",
			"production_claim": true,
			"report":           "surface-electron-desktop.json",
		})
	}, nil)

	err := validateSurfaceProductSummary(surfaceProductSummaryOptions{ReportDir: dir})
	if err == nil {
		t.Fatalf("expected unexpected target_matrix entry to fail")
	}
	if !strings.Contains(err.Error(), `unexpected target_matrix target "electron-desktop"`) {
		t.Fatalf("error = %v, want unexpected target diagnostic", err)
	}
}

func TestValidateSurfaceProductSummaryRejectsDuplicateTargetMatrixEntry(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceProductSummaryFixture(t, dir, func(summary map[string]any) {
		summary["target_matrix"] = append(surfaceProductSummaryTargetMatrixFixture(), map[string]any{
			"target":           "linux-x64",
			"status":           "current",
			"tier":             "bounded-linux-web-scope",
			"production_claim": true,
			"report":           "surface-linux-x64-release-app-shell.json",
		})
	}, nil)

	err := validateSurfaceProductSummary(surfaceProductSummaryOptions{ReportDir: dir})
	if err == nil {
		t.Fatalf("expected duplicate target_matrix entry to fail")
	}
	if !strings.Contains(err.Error(), `duplicate target_matrix target "linux-x64"`) {
		t.Fatalf("error = %v, want duplicate target diagnostic", err)
	}
}

func TestValidateSurfaceProductSummaryRejectsMissingCategorySourceReport(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceProductSummaryFixture(t, dir, nil, nil)
	updateCategorySummarySourceReport(t, dir, "visual/visual-summary.json", "missing-source.json")

	err := validateSurfaceProductSummary(surfaceProductSummaryOptions{ReportDir: dir})
	if err == nil {
		t.Fatalf("expected missing category source_report to fail")
	}
	if !strings.Contains(err.Error(), `source_report "missing-source.json" read failed`) {
		t.Fatalf("error = %v, want missing source_report diagnostic", err)
	}
}

func TestValidateSurfaceProductSummaryRejectsUnhashedCategorySourceReport(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceProductSummaryFixture(t, dir, nil, nil)
	updateCategorySummarySourceReport(t, dir, "visual/visual-summary.json", "unhashed-source.json")
	writeJSON(t, filepath.Join(dir, "unhashed-source.json"), map[string]any{"schema": "fixture"})

	err := validateSurfaceProductSummary(surfaceProductSummaryOptions{ReportDir: dir})
	if err == nil {
		t.Fatalf("expected unhashed category source_report to fail")
	}
	if !strings.Contains(err.Error(), `source_report "unhashed-source.json" is not hash-covered`) {
		t.Fatalf("error = %v, want missing source hash diagnostic", err)
	}
}

func writeSurfaceProductSummaryFixture(t *testing.T, dir string, mutateSummary func(map[string]any), mutateArtifacts func([]map[string]any) []map[string]any) {
	t.Helper()

	requiredArtifacts := map[string]string{
		"product_summary":  "product-summary.json",
		"artifact_hashes":  "artifact-hashes.json",
		"visual":           "visual/visual-summary.json",
		"accessibility":    "accessibility/accessibility-summary.json",
		"performance":      "performance/performance-budget.json",
		"app_shell":        "app-shell/app-shell-summary.json",
		"package":          "package/package-summary.json",
		"reference_apps":   "reference-apps/reference-apps-summary.json",
		"claim_governance": "claim-governance/claims-summary.json",
	}
	summary := map[string]any{
		"schema":                            "tetra.surface.product-summary.v1",
		"release_scope":                     "surface-v1-linux-web",
		"status":                            "product_gate_passed_clean_same_commit_blocked",
		"producer":                          "scripts/release/surface/product-gate.sh",
		"git_head":                          "0123456789abcdef0123456789abcdef01234567",
		"git_dirty":                         true,
		"generated_at_utc":                  "2026-06-13T00:00:00Z",
		"command_line":                      "bash scripts/release/surface/product-gate.sh --report-dir reports/surface-product-v1",
		"product_gate_summary":              "surface-product-gate-summary.json",
		"release_gate_report":               "surface-release-summary.json",
		"artifact_hash_manifest":            "artifact-hashes.json",
		"release_state":                     "validated",
		"artifact_hashes":                   "validated",
		"claim_scanner":                     "validated",
		"manifest":                          "validated",
		"docs":                              "validated",
		"ci_required_gate":                  true,
		"continue_on_error_bypass_allowed":  false,
		"final_verdict_owner":               "SURFACE-BEAUTY-P29",
		"final_verdict":                     "BLOCKED_DIRTY_CHECKOUT",
		"production_claim":                  false,
		"final_signoff":                     false,
		"canonical_final_readiness_report":  true,
		"final_readiness_source":            "product-summary.json",
		"inner_release_summary_role":        "prerequisite_evidence_not_final_signoff",
		"release_gate_report_final_signoff": false,
		"clean_same_commit_required":        true,
		"clean_same_commit_proven":          false,
		"target_matrix":                     surfaceProductSummaryTargetMatrixFixture(),
		"required_artifacts":                requiredArtifacts,
		"nonclaims":                         surfaceProductSummaryNonclaimsFixture(),
	}
	if mutateSummary != nil {
		mutateSummary(summary)
	}
	writeJSON(t, filepath.Join(dir, "product-summary.json"), summary)

	sourceReportsByCategory := map[string]string{
		"visual":           "reference-visual/surface-visual-regression.json",
		"accessibility":    "surface-linux-x64-release-accessibility.json",
		"performance":      "surface-linux-x64-release-app-shell.json",
		"app-shell":        "surface-linux-x64-release-app-shell.json",
		"package":          "surface-package.json",
		"reference-apps":   "surface-reference-apps.json",
		"claim-governance": "surface-product-gate-summary.json",
	}
	for _, path := range uniqueStringValues(sourceReportsByCategory) {
		writeJSON(t, filepath.Join(dir, filepath.FromSlash(path)), map[string]any{"schema": "fixture"})
	}
	for category, path := range map[string]string{
		"visual":           "visual/visual-summary.json",
		"accessibility":    "accessibility/accessibility-summary.json",
		"performance":      "performance/performance-budget.json",
		"app-shell":        "app-shell/app-shell-summary.json",
		"package":          "package/package-summary.json",
		"reference-apps":   "reference-apps/reference-apps-summary.json",
		"claim-governance": "claim-governance/claims-summary.json",
	} {
		writeJSON(t, filepath.Join(dir, filepath.FromSlash(path)), map[string]any{
			"schema":              "tetra.surface.product-category-summary.v1",
			"release_scope":       "surface-v1-linux-web",
			"category":            category,
			"status":              "validated-evidence-summary",
			"source_report":       sourceReportsByCategory[category],
			"evidence":            "fixture",
			"git_head":            "0123456789abcdef0123456789abcdef01234567",
			"git_dirty":           true,
			"final_verdict_owner": "SURFACE-BEAUTY-P29",
			"final_verdict":       "BLOCKED_DIRTY_CHECKOUT",
			"production_claim":    false,
			"final_signoff":       false,
		})
	}

	artifacts := []map[string]any{}
	for _, path := range []string{
		"product-summary.json",
		"visual/visual-summary.json",
		"accessibility/accessibility-summary.json",
		"performance/performance-budget.json",
		"app-shell/app-shell-summary.json",
		"package/package-summary.json",
		"reference-apps/reference-apps-summary.json",
		"claim-governance/claims-summary.json",
	} {
		artifacts = append(artifacts, map[string]any{
			"path":   path,
			"sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"size":   1,
		})
	}
	for _, path := range uniqueStringValues(sourceReportsByCategory) {
		artifacts = append(artifacts, map[string]any{
			"path":   path,
			"sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"size":   1,
		})
	}
	if mutateArtifacts != nil {
		artifacts = mutateArtifacts(artifacts)
	}
	writeJSON(t, filepath.Join(dir, "artifact-hashes.json"), map[string]any{
		"schema":    "tetra.release-artifact-hashes.v1alpha1",
		"root":      ".",
		"artifacts": artifacts,
	})
}

func surfaceProductSummaryTargetMatrixFixture() []any {
	return []any{
		map[string]any{"target": "headless", "status": "release-test-evidence", "tier": "evidence-target", "production_claim": false, "report": "surface-headless-release.json"},
		map[string]any{"target": "linux-x64", "status": "current", "tier": "bounded-linux-web-scope", "production_claim": true, "report": "surface-linux-x64-release-app-shell.json"},
		map[string]any{"target": "wasm32-web", "status": "current", "tier": "bounded-linux-web-scope", "production_claim": true, "report": "surface-wasm32-web-release-browser.json"},
		map[string]any{"target": "macos-x64", "status": "unsupported", "tier": "UNSUPPORTED", "production_claim": false, "report": "surface-macos-x64-target-host-status.json"},
		map[string]any{"target": "windows-x64", "status": "unsupported", "tier": "UNSUPPORTED", "production_claim": false, "report": "surface-windows-x64-target-host-status.json"},
		map[string]any{"target": "wasm32-wasi", "status": "unsupported", "tier": "UNSUPPORTED", "production_claim": false, "report": "surface-release-summary.json"},
	}
}

func updateCategorySummarySourceReport(t *testing.T, dir, path, sourceReport string) {
	t.Helper()
	fullPath := filepath.Join(dir, filepath.FromSlash(path))
	raw, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var summary map[string]any
	if err := json.Unmarshal(raw, &summary); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	summary["source_report"] = sourceReport
	writeJSON(t, fullPath, summary)
}

func uniqueStringValues(values map[string]string) []string {
	seen := map[string]bool{}
	var unique []string
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		unique = append(unique, value)
	}
	return unique
}

func surfaceProductSummaryNonclaimsFixture() []any {
	return []any{
		"all-platform-surface-parity",
		"nonclaim-macos-surface-production-support",
		"nonclaim-windows-surface-production-support",
		"wasm32-wasi-surface-ui-runtime",
		"gpu-renderer",
		"full-rich-text-editor",
		"full-screen-reader-support",
		"official-benchmark-superiority",
		"electron-api-compatibility",
		"react-api-compatibility",
		"css-cascade-compatibility",
		"dom-authored-application-ui",
		"user-javascript-application-logic",
	}
}

func writeJSON(t *testing.T, path string, value any) {
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
