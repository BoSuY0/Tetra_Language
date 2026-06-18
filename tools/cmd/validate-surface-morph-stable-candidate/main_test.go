package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSurfaceMorphStableCandidateAcceptsContract(t *testing.T) {
	path := writeMorphStableCandidateFixture(t, validMorphStableCandidateFixture(t))
	if err := validateSurfaceMorphStableCandidate(path); err != nil {
		t.Fatalf("validateSurfaceMorphStableCandidate failed: %v", err)
	}
}

func TestValidateSurfaceMorphStableCandidateRejectsMissingStableSchema(t *testing.T) {
	fixture := validMorphStableCandidateFixture(t)
	delete(fixture["stable_schemas"].(map[string]any), "variant")
	path := writeMorphStableCandidateFixture(t, fixture)

	err := validateSurfaceMorphStableCandidate(path)
	if err == nil {
		t.Fatalf("expected missing variant stable schema to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "variant") {
		t.Fatalf("error = %v, want variant diagnostic", err)
	}
}

func TestValidateSurfaceMorphStableCandidateRejectsProductionWithoutTargetEvidence(t *testing.T) {
	fixture := validMorphStableCandidateFixture(t)
	fixture["production_claim"] = true
	fixture["required_target_evidence"] = []any{"headless"}
	path := writeMorphStableCandidateFixture(t, fixture)

	err := validateSurfaceMorphStableCandidate(path)
	if err == nil {
		t.Fatalf("expected production claim without target evidence to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "target evidence") {
		t.Fatalf("error = %v, want target evidence diagnostic", err)
	}
}

func TestValidateSurfaceMorphStableCandidateRejectsRecipeOutputButton(t *testing.T) {
	fixture := validMorphStableCandidateFixture(t)
	recipe := fixture["recipe_contract"].(map[string]any)
	recipe["allowed_outputs"] = []any{"Block", "Button"}
	path := writeMorphStableCandidateFixture(t, fixture)

	err := validateSurfaceMorphStableCandidate(path)
	if err == nil {
		t.Fatalf("expected recipe output Button to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "button") {
		t.Fatalf("error = %v, want Button diagnostic", err)
	}
}

func TestValidateSurfaceMorphStableCandidateRejectsEnabledBeforeP20(t *testing.T) {
	fixture := validMorphStableCandidateFixture(t)
	fixture["validator_enabled"] = true
	path := writeMorphStableCandidateFixture(t, fixture)

	err := validateSurfaceMorphStableCandidate(path)
	if err == nil {
		t.Fatalf("expected enabled stable promotion validator to fail before P20+")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "disabled") {
		t.Fatalf("error = %v, want disabled diagnostic", err)
	}
}

func TestValidateSurfaceMorphStableCandidateRejectsMissingRendererOwnedStableProofGate(
	t *testing.T,
) {
	fixture := validMorphStableCandidateFixture(t)
	fixture["promotion_gates"] = []any{
		"validate-surface-morph-report",
		"validate-surface-claims",
		"surface block-system gate",
		"visual regression gate",
		"target-host evidence",
	}
	path := writeMorphStableCandidateFixture(t, fixture)

	err := validateSurfaceMorphStableCandidate(path)
	if err == nil {
		t.Fatalf("expected missing renderer-owned stable proof gate to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "renderer-owned stable proof") {
		t.Fatalf("error = %v, want renderer-owned stable proof diagnostic", err)
	}
}

func validMorphStableCandidateFixture(t *testing.T) map[string]any {
	t.Helper()
	raw := []byte(`{
  "schema": "tetra.surface.morph.stable-candidate.v1",
  "status": "design-freeze",
  "current_tier": "EXPERIMENTAL",
  "target_tier": "PROD_STABLE_SCOPED",
  "surface_scope": "surface-v1-linux-web",
  "production_claim": false,
  "validator_enabled": false,
  "disabled_until": "P20+",
  "required_target_evidence": ["headless", "linux-x64-real-window", "wasm32-web-browser-canvas"],
  "stable_schemas": {
    "token_graph": {"schema": "tetra.surface.morph.token-graph.v1", "required_fields": ["schema", "namespace", "version", "hash", "source_of_truth", "explicit_imports", "no_global_cascade", "fixed_override_order", "categories", "tokens", "density_dpi", "diagnostics"], "backward_compatibility": ["versioned_schema", "additive_fields_only"]},
    "material": {"schema": "tetra.surface.morph.material.v1", "required_fields": ["name", "paint_stack", "fill", "border", "radius", "shadow", "overlay"], "backward_compatibility": ["versioned_schema", "additive_fields_only"]},
    "affordance": {"schema": "tetra.surface.morph.affordance.v1", "required_fields": ["name", "role", "focusable", "action", "input", "projects_accessibility"], "backward_compatibility": ["versioned_schema", "additive_fields_only"]},
    "recipe": {"schema": "tetra.surface.morph.recipe.v1", "required_fields": ["name", "output", "slots", "inputs", "expands_to_block_graph"], "backward_compatibility": ["versioned_schema", "additive_fields_only"]},
    "variant": {"schema": "tetra.surface.morph.variant.v1", "required_fields": ["name", "state_lenses", "materials", "motion"], "backward_compatibility": ["versioned_schema", "additive_fields_only"]},
    "state_lens": {"schema": "tetra.surface.morph.state-lens.v1", "required_fields": ["selector", "property", "deterministic"], "backward_compatibility": ["versioned_schema", "additive_fields_only"]},
    "motion_preset": {"schema": "tetra.surface.morph.motion-preset.v1", "required_fields": ["name", "duration_ms", "curve", "properties", "reduced_motion", "deterministic_time"], "backward_compatibility": ["versioned_schema", "additive_fields_only"]},
    "accessibility_projection": {"schema": "tetra.surface.morph.accessibility-projection.v1", "required_fields": ["schema", "derived_from_block_graph", "safety_overrides_win", "snapshot_evidence", "required_fields", "roles"], "backward_compatibility": ["versioned_schema", "additive_fields_only"]}
  },
  "recipe_contract": {
    "allowed_outputs": ["Block"],
    "forbidden_outputs": ["Button", "Card", "TextField", "TextBox", "Sidebar", "Modal"],
    "requires_expands_to_block_graph": true,
    "requires_no_hidden_app_state": true,
    "requires_no_platform_widgets": true,
    "requires_no_core_primitive_promotion": true
  },
  "promotion_gates": ["validate-surface-morph-report", "validate-surface-claims", "surface block-system gate", "visual regression gate", "target-host evidence", "renderer-owned stable proof"],
  "nonclaims": ["not production Morph today", "no React runtime", "no Electron runtime", "no CSS cascade runtime", "no platform-native widgets"]
}`)
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	return out
}

func writeMorphStableCandidateFixture(t *testing.T, value map[string]any) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "surface-morph-stable-candidate.json")
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}
