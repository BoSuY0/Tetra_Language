package scriptstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSurfaceMorphGateRunsTokenGraphContractValidator(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "surface", "morph-gate.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Surface Morph gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`go run ./tools/cmd/validate-surface-morph-report --report "$report_dir/headless/surface-headless-morph.json" --same-commit "$git_head"`,
		`go run ./tools/cmd/validate-surface-token-graph --contract docs/spec/surface_token_graph_contract.json --report "$report_dir/headless/surface-headless-morph.json" --root "$repo_root"`,
		`"token_graph_contract": "docs/spec/surface_token_graph_contract.json"`,
		`"token_graph_validator": "validate-surface-token-graph"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Surface Morph gate missing token graph detail %q", want)
		}
	}
	assertOrderedFragments(t, text,
		`validate-surface-morph-report --report "$report_dir/headless/surface-headless-morph.json"`,
		`validate-surface-token-graph --contract docs/spec/surface_token_graph_contract.json`,
		`cat > "$summary_path" <<JSON`,
		`validate-artifact-hashes --write --root "$report_dir"`,
	)
}

func TestSurfaceMorphGatePublishesRecipeAuthoringEvidence(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "surface", "morph-gate.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Surface Morph gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`"recipe_authoring_validator": "validate-surface-morph-report"`,
		`"recipe_expansion_report": "headless/surface-headless-morph.json#morph.recipe_expansions"`,
		`"recipe_count": 11`,
		`"examples/surface_morph_command_palette.tetra"`,
		`"examples/surface_morph_project_dashboard.tetra"`,
		`"examples/surface_morph_settings.tetra"`,
		`"examples/surface_morph_editor_shell.tetra"`,
		`"examples/surface_morph_glass_panel.tetra"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Surface Morph gate missing recipe authoring detail %q", want)
		}
	}
	assertOrderedFragments(t, text,
		`validate-surface-morph-report --report "$report_dir/headless/surface-headless-morph.json"`,
		`"recipe_authoring_validator": "validate-surface-morph-report"`,
		`"artifact_hashes_validated": true`,
	)
}
