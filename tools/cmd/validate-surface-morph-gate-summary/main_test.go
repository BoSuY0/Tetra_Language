package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSurfaceMorphGateSummaryAcceptsCurrentFixture(t *testing.T) {
	path := writeMorphGateSummaryFixture(t, nil)

	if err := validateSurfaceMorphGateSummary(path); err != nil {
		t.Fatalf("validateSurfaceMorphGateSummary failed: %v", err)
	}
}

func TestValidateSurfaceMorphGateSummaryRejectsWrongRecipeCount(t *testing.T) {
	path := writeMorphGateSummaryFixture(t, func(summary map[string]any) {
		summary["recipe_count"] = 18
	})

	err := validateSurfaceMorphGateSummary(path)
	if err == nil || !strings.Contains(err.Error(), "recipe_count") {
		t.Fatalf("validateSurfaceMorphGateSummary err = %v, want recipe_count rejection", err)
	}
}

func TestValidateSurfaceMorphGateSummaryRejectsWrongReferenceApps(t *testing.T) {
	path := writeMorphGateSummaryFixture(t, func(summary map[string]any) {
		summary["reference_recipe_apps"] = []string{
			"examples/surface_morph_command_palette.tetra",
			"examples/surface_morph_settings.tetra",
			"examples/surface_morph_project_dashboard.tetra",
			"examples/surface_morph_editor_shell.tetra",
			"examples/surface_morph_glass_panel.tetra",
			"examples/surface_morph_studio_shell.tetra",
		}
	})

	err := validateSurfaceMorphGateSummary(path)
	if err == nil || !strings.Contains(err.Error(), "reference_recipe_apps") {
		t.Fatalf("validateSurfaceMorphGateSummary err = %v, want reference_recipe_apps rejection", err)
	}
}

func TestValidateSurfaceMorphGateSummaryRejectsMissingReferenceApps(t *testing.T) {
	path := writeMorphGateSummaryFixture(t, func(summary map[string]any) {
		delete(summary, "reference_recipe_apps")
	})

	err := validateSurfaceMorphGateSummary(path)
	if err == nil || !strings.Contains(err.Error(), "reference_recipe_apps") {
		t.Fatalf("validateSurfaceMorphGateSummary err = %v, want reference_recipe_apps missing rejection", err)
	}
}

func TestValidateSurfaceMorphGateSummaryRejectsSameCommitGitHeadMismatch(t *testing.T) {
	path := writeMorphGateSummaryFixture(t, func(summary map[string]any) {
		summary["git_head"] = "def456"
	})

	err := validateSurfaceMorphGateSummaryWithOptions(path, morphGateSummaryValidationOptions{SameCommit: "abc123"})
	if err == nil {
		t.Fatal("validateSurfaceMorphGateSummaryWithOptions returned nil, want same-commit/git_head rejection")
	}
	errText := strings.ToLower(err.Error())
	if !strings.Contains(errText, "same-commit") || !strings.Contains(errText, "git_head") {
		t.Fatalf("validateSurfaceMorphGateSummaryWithOptions err = %v, want same-commit/git_head rejection", err)
	}
}

func TestValidateSurfaceMorphGateSummaryRejectsStaleSameCommit(t *testing.T) {
	err := validateSameCommit("abc123", "def456")
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "same-commit") {
		t.Fatalf("validateSameCommit err = %v, want same-commit mismatch", err)
	}
	if err := validateSameCommit("abc123", "abc123"); err != nil {
		t.Fatalf("validateSameCommit exact match: %v", err)
	}
	if err := validateSameCommit("abc123", "abc123ffff"); err != nil {
		t.Fatalf("validateSameCommit expected prefix: %v", err)
	}
	if err := validateSameCommit("abc123ffff", "abc123"); err != nil {
		t.Fatalf("validateSameCommit actual prefix: %v", err)
	}
}

func writeMorphGateSummaryFixture(t *testing.T, mutate func(map[string]any)) string {
	t.Helper()
	summary := map[string]any{
		"schema":                     "tetra.surface.morph.gate.v1",
		"status":                     "current",
		"release_scope":              "surface-morph-experimental-linux-web",
		"producer":                   "scripts/release/surface/morph-gate.sh",
		"git_head":                   "0123456789abcdef0123456789abcdef01234567",
		"version":                    "tetra_language",
		"git_dirty":                  true,
		"host_os":                    "linux",
		"host_arch":                  "amd64",
		"generated_at_utc":           "2026-06-16T00:00:00Z",
		"command_line":               "bash scripts/release/surface/morph-gate.sh --report-dir reports/surface-morph/gate",
		"source":                     "examples/surface_morph_command_palette.tetra",
		"module":                     "lib.core.morph",
		"schema_under_test":          "tetra.surface.morph.v1",
		"token_graph_contract":       "docs/spec/surface_token_graph_contract.json",
		"token_graph_validator":      "validate-surface-token-graph",
		"recipe_authoring_validator": "validate-surface-morph-report",
		"recipe_expansion_report":    "headless/surface-headless-morph.json#morph.recipe_expansions",
		"recipe_count":               19,
		"reference_recipe_apps": []string{
			"examples/surface_morph_command_palette.tetra",
			"examples/surface_morph_project_dashboard.tetra",
			"examples/surface_morph_settings.tetra",
			"examples/surface_morph_editor_shell.tetra",
			"examples/surface_morph_glass_panel.tetra",
			"examples/surface_morph_studio_shell.tetra",
		},
		"dependency_gate":           "tetra.surface.block-system.gate.v1",
		"same_commit_validated":     true,
		"headless_report":           "headless/surface-headless-morph.json",
		"target_evidence":           []string{"headless"},
		"core_primitives":           []string{"Block"},
		"forbidden_core_primitives": []string{"Button", "Card", "TextField", "TextBox", "Sidebar", "Modal"},
		"artifact_hashes_validated": true,
	}
	if mutate != nil {
		mutate(summary)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "surface-morph-gate-summary.json")
	raw, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
