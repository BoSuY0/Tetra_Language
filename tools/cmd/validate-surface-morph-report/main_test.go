package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surface"
)

func TestMorphReportCLIRejectsSymlinkedReportDir(t *testing.T) {
	realDir := filepath.Join(t.TempDir(), "real")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(t.TempDir(), "linked")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	err := validateMorphReportPathSafety(filepath.Join(linkDir, "surface-headless-morph.json"))
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "symlink") {
		t.Fatalf("validateMorphReportPathSafety symlink err = %v, want symlink rejection", err)
	}
}

func TestMorphReportCLIRejectsArtifactOutsideReportDir(t *testing.T) {
	reportDir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside-artifact")
	artifact := surface.ArtifactReport{
		Kind:   "component-app",
		Path:   outside,
		SHA256: "sha256:" + strings.Repeat("a", 64),
		Size:   16,
	}
	scan := surface.ArtifactScanReport{Root: filepath.Dir(outside), FilesChecked: 1, Pass: true}
	err := validateMorphReportArtifactLocality(
		filepath.Join(reportDir, "surface-headless-morph.json"),
		scan,
		[]surface.ArtifactReport{artifact},
	)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "outside") {
		t.Fatalf(
			"validateMorphReportArtifactLocality err = %v, want outside report dir rejection",
			err,
		)
	}
}

func TestMorphReportCLIRejectsStaleArtifactHash(t *testing.T) {
	reportDir := t.TempDir()
	artifactPath := filepath.Join(reportDir, "artifact.bin")
	if err := os.WriteFile(artifactPath, []byte("fresh morph artifact"), 0o644); err != nil {
		t.Fatal(err)
	}
	stale := surface.ArtifactReport{
		Kind:   "component-app",
		Path:   artifactPath,
		SHA256: "sha256:" + strings.Repeat("b", 64),
		Size:   int64(len("fresh morph artifact")),
	}
	err := validateMorphReportArtifactFiles(reportDir, []surface.ArtifactReport{stale})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "sha256") {
		t.Fatalf("validateMorphReportArtifactFiles stale hash err = %v, want sha256 rejection", err)
	}

	sum := sha256.Sum256([]byte("fresh morph artifact"))
	valid := surface.ArtifactReport{
		Kind:   "component-app",
		Path:   artifactPath,
		SHA256: fmt.Sprintf("sha256:%x", sum),
		Size:   int64(len("fresh morph artifact")),
	}
	if err := validateMorphReportArtifactFiles(
		reportDir,
		[]surface.ArtifactReport{valid},
	); err != nil {
		t.Fatalf("validateMorphReportArtifactFiles valid artifact: %v", err)
	}
}

func TestMorphReportCLIRejectsSameCommitMismatch(t *testing.T) {
	err := validateSameCommit("abc123", "def456")
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "same-commit") {
		t.Fatalf("validateSameCommit err = %v, want same-commit mismatch", err)
	}
	if err := validateSameCommit("abc123", "abc123"); err != nil {
		t.Fatalf("validateSameCommit matching commits: %v", err)
	}
	if err := validateSameCommit("", "abc123"); err != nil {
		t.Fatalf("validateSameCommit without expectation: %v", err)
	}
}

func TestMorphReportCLITokenCapsuleRecipeBoundaryAcceptsStableStyleGraph(t *testing.T) {
	if err := surface.ValidateMorphStyleTokenBoundary(stableMorphBoundaryForCLITest()); err != nil {
		t.Fatalf("ValidateMorphStyleTokenBoundary failed: %v", err)
	}
}

func TestMorphReportCLIFakeCSSClaimRejected(t *testing.T) {
	morph := stableMorphBoundaryForCLITest()
	morph.StyleGraph.RawCSSRuntimeImportRejected = false
	err := surface.ValidateMorphStyleTokenBoundary(morph)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "raw css") {
		t.Fatalf("ValidateMorphStyleTokenBoundary err = %v, want raw CSS runtime rejection", err)
	}
}

func TestMorphReportCLIRecipeAuthoringRejectsRawBlockClaim(t *testing.T) {
	morph := stableMorphBoundaryForCLITest()
	morph.Authoring.DirectBlockPropEditing = true
	err := surface.ValidateMorphStyleTokenBoundary(morph)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "direct block prop") {
		t.Fatalf("ValidateMorphStyleTokenBoundary err = %v, want direct Block prop rejection", err)
	}
}

func TestMorphReportCLIRecipeAuthoringRejectsIncompleteProductionFamilyLibrary(t *testing.T) {
	morph := stableMorphBoundaryForCLITest()
	morph.Recipes = morph.Recipes[:4]
	morph.Authoring.RecipeCount = len(morph.Recipes)
	morph.Authoring.PolishedRecipeCount = len(morph.Recipes)
	err := surface.ValidateMorphStyleTokenBoundary(morph)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "recipe_count") {
		t.Fatalf("ValidateMorphStyleTokenBoundary err = %v, want missing stable recipe family rejection", err)
	}
}

func stableMorphBoundaryForCLITest() *surface.MorphReport {
	return &surface.MorphReport{
		Capsule: surface.MorphCapsuleReport{
			Namespace: "tetra.surface.morph.app",
		},
		StyleGraph: &surface.MorphStyleGraphReport{
			Schema:                         "tetra.surface.morph.style-graph.v1",
			Namespace:                      "tetra.surface.morph.app",
			Version:                        "1",
			CSSReplacementLevel:            "typed-style-graph-candidate-v1",
			VocabularyFrozen:               true,
			TokenCategories:                []string{"color", "space", "spacing", "radius", "border", "elevation", "opacity", "typography", "type", "motion", "z", "assets", "density"},
			MaterialSlots:                  []string{"fill", "border", "radius", "shadow", "overlay"},
			AffordanceRoles:                []string{"action", "field.text", "toggle", "navigation", "region", "overlay", "status"},
			RecipeOutputs:                  []string{"Block"},
			StateSelectors:                 []string{"hover", "pressed", "focusVisible", "selected", "disabled", "error", "loading"},
			MotionProperties:               []string{"fill", "opacity", "transform"},
			OverrideOrder:                  []string{"capsule", "tokens", "materials", "affordances", "state_lenses", "motion", "recipes", "accessibility_safety"},
			ConflictDiagnostics:            []string{"alias_cycle", "duplicate_recipe", "duplicate_token_source", "unresolved_token", "raw_literal", "unsupported_css_cascade", "forbidden_runtime_import", "global_style_leak", "specificity_ambiguity", "raw_css_runtime_import"},
			ImportAllowlist:                []string{"lib.core.block", "lib.core.morph"},
			CSSCascadeImportsRejected:      true,
			DOMRuntimeImportsRejected:      true,
			ReactRuntimeImportsRejected:    true,
			ElectronRuntimeImportsRejected: true,
			SelectorEngineAbsent:           true,
			NoSpecificityScoring:           true,
			GlobalStyleLeakRejected:        true,
			SpecificityAmbiguityRejected:   true,
			RawCSSRuntimeImportRejected:    true,
		},
		Authoring: &surface.MorphAuthoringReport{
			Schema:                   "tetra.surface.morph.authoring.v1",
			Level:                    "production-recipe-authoring-v1",
			RecipeCount:              11,
			PolishedRecipeCount:      11,
			MaxAuthorFields:          12,
			RawBlockFieldCount:       80,
			Raw80FieldBlocksRejected: true,
			RecipesRequired:          true,
			DirectBlockPropEditing:   false,
			RecipeFirstAuthoring:     true,
			DesignerTokenInputs:      true,
			GeneratedBlockPropsOnly:  true,
			RawLiteralStylesRejected: true,
			NonClaims:                []string{"raw 80-field Block authoring", "CSS cascade", "selector engine", "specificity scoring"},
		},
		Recipes: stableMorphRecipesForCLITest(),
	}
}

func stableMorphRecipesForCLITest() []surface.MorphRecipeReport {
	return []surface.MorphRecipeReport{
		stableMorphRecipeForCLITest("control.action@1", "control.action", []string{"label", "icon"}, []string{"text", "action", "variant"}, []string{"pressed", "focused"}, []string{"role:button", "name", "action"}),
		stableMorphRecipeForCLITest("field.text@1", "field.text", []string{"label", "control"}, []string{"value", "on_text"}, []string{"focused", "error"}, []string{"role:textbox", "labelled_by", "value"}),
		stableMorphRecipeForCLITest("control.toggle@1", "control.toggle", []string{"label", "control"}, []string{"checked", "on_toggle"}, []string{"checked", "focused"}, []string{"role:checkbox", "checked", "name"}),
		stableMorphRecipeForCLITest("command.item@1", "command.item", []string{"icon", "title", "subtitle"}, []string{"title", "subtitle", "icon", "selected"}, []string{"selected", "focused"}, []string{"role:button", "selected", "description"}),
		stableMorphRecipeForCLITest("navigation.item@1", "navigation.item", []string{"label", "badge"}, []string{"route", "selected"}, []string{"selected", "focused"}, []string{"role:navigation", "current", "name"}),
		stableMorphRecipeForCLITest("region.panel@1", "region.panel", []string{"header", "body", "actions"}, []string{"title"}, []string{"expanded", "loading"}, []string{"role:region", "labelled_by", "bounds"}),
		stableMorphRecipeForCLITest("overlay.dialog@1", "overlay.dialog", []string{"title", "body", "actions"}, []string{"open", "dismiss"}, []string{"open", "focus_trap"}, []string{"role:dialog", "modal", "name"}),
		stableMorphRecipeForCLITest("navigation.tabs@1", "navigation.tabs", []string{"tab", "panel"}, []string{"items", "active"}, []string{"active", "focused"}, []string{"role:tablist", "selected", "controls"}),
		stableMorphRecipeForCLITest("collection.list@1", "collection.list", []string{"item", "empty"}, []string{"items", "selected"}, []string{"selected", "empty"}, []string{"role:list", "item_count", "selected"}),
		stableMorphRecipeForCLITest("collection.table-lite@1", "collection.table-lite", []string{"header", "row", "cell"}, []string{"rows", "columns"}, []string{"sorted", "selected"}, []string{"role:table", "row_count", "column_count"}),
		stableMorphRecipeForCLITest("status.message@1", "status.message", []string{"icon", "message"}, []string{"kind", "text"}, []string{"severity", "live"}, []string{"role:status", "live", "name"}),
	}
}

func stableMorphRecipeForCLITest(name string, family string, slots []string, inputs []string, state []string, a11y []string) surface.MorphRecipeReport {
	return surface.MorphRecipeReport{
		Name:                name,
		Family:              family,
		Output:              "Block",
		Slots:               slots,
		Inputs:              inputs,
		State:               state,
		Accessibility:       a11y,
		ExpandsToBlockGraph: true,
	}
}
