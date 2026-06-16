package scriptstest

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tetra_language/tools/internal/artifacts"
	"tetra_language/tools/internal/gatecontract"
)

func TestSurfaceMorphGateRunsTokenGraphContractValidator(t *testing.T) {
	root := repoRoot(t)
	contract := loadSurfaceMorphContract(t, root)
	text := readSurfaceMorphGateScript(t, root)

	if contract.ID != "surface-morph-gate-v1" {
		t.Fatalf("Surface Morph gate contract id = %q", contract.ID)
	}
	if contract.Producer != "scripts/release/surface/morph-gate.sh" || contract.Entrypoint != "scripts/release/surface/morph-gate.sh" {
		t.Fatalf("Surface Morph gate contract producer/entrypoint = %q/%q", contract.Producer, contract.Entrypoint)
	}
	if contract.ArtifactHashes == nil || !contract.ArtifactHashes.Enabled || !contract.ArtifactHashes.Required || contract.ArtifactHashes.Algorithm != "sha256" {
		t.Fatalf("Surface Morph gate artifact_hashes = %#v, want enabled required sha256", contract.ArtifactHashes)
	}

	wantReports := []string{
		"surface-morph-gate-summary.json",
		"headless/surface-headless-morph.json",
		"headless/artifact-hashes.json",
		"artifact-hashes.json",
	}
	assertSurfaceMorphRequiredReports(t, contract, wantReports)
	assertSurfaceMorphCIArtifacts(t, contract, wantReports)
	assertSurfaceMorphValidators(t, text, contract,
		"validate-surface-morph-report",
		"validate-surface-token-graph",
		"validate-surface-morph-gate-summary",
		"validate-artifact-hashes",
	)

	for _, want := range []string{
		`gate_contract="scripts/release/surface/contracts/morph-gate.json"`,
		`go run ./tools/cmd/run-gate --contract "$gate_contract" --report-dir "$report_dir_arg" --dry-run >/dev/null`,
		`"token_graph_contract": "docs/spec/surface_token_graph_contract.json"`,
		`"token_graph_validator": "validate-surface-token-graph"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Surface Morph gate missing token graph detail %q", want)
		}
	}
	assertEqualOrderedStrings(t, parseBashStringArray(t, text, "final_required_reports"), wantReports, "Surface Morph final_required_reports")
	assertOrderedFragments(t, text,
		`go run ./tools/cmd/run-gate --contract "$gate_contract" --report-dir "$report_dir_arg" --dry-run >/dev/null`,
		`surface_release_require_fresh_report_dir "$report_dir_arg"`,
		`bash scripts/release/surface/surface-headless-morph-smoke.sh --report-dir "$headless_report_dir"`,
		`validate-surface-morph-report --report "$report_dir/headless/surface-headless-morph.json"`,
		`validate-surface-token-graph --contract docs/spec/surface_token_graph_contract.json`,
		`cat > "$summary_path" <<JSON`,
		`validate-surface-morph-gate-summary --summary "$report_dir/surface-morph-gate-summary.json"`,
		`validate-artifact-hashes --write --root "$report_dir"`,
		`validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"`,
	)
}

func TestSurfaceMorphGatePublishesRecipeAuthoringEvidence(t *testing.T) {
	root := repoRoot(t)
	contract := loadSurfaceMorphContract(t, root)
	text := readSurfaceMorphGateScript(t, root)

	report := surfaceMorphRequiredReport(t, contract, "headless/surface-headless-morph.json")
	for _, wantClaim := range []string{"morph_recipe_authoring", "surface_token_graph"} {
		if !containsString(report.ClaimRefs, wantClaim) {
			t.Fatalf("headless Surface Morph report claim_refs = %#v, want %q", report.ClaimRefs, wantClaim)
		}
	}

	for _, want := range []string{
		`"recipe_authoring_validator": "validate-surface-morph-report"`,
		`"recipe_expansion_report": "headless/surface-headless-morph.json#morph.recipe_expansions"`,
		`"recipe_count": 19`,
		`"examples/surface_morph_command_palette.tetra"`,
		`"examples/surface_morph_project_dashboard.tetra"`,
		`"examples/surface_morph_settings.tetra"`,
		`"examples/surface_morph_editor_shell.tetra"`,
		`"examples/surface_morph_glass_panel.tetra"`,
		`"examples/surface_morph_studio_shell.tetra"`,
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

func loadSurfaceMorphContract(t *testing.T, root string) gatecontract.Contract {
	t.Helper()
	contractPath := filepath.Join(root, "scripts", "release", "surface", "contracts", "morph-gate.json")
	contract, err := gatecontract.Load(contractPath)
	if err != nil {
		t.Fatalf("load Surface Morph gate contract: %v", err)
	}
	return contract
}

func readSurfaceMorphGateScript(t *testing.T, root string) string {
	t.Helper()
	path := filepath.Join(root, "scripts", "release", "surface", "morph-gate.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Surface Morph gate: %v", err)
	}
	return string(raw)
}

func assertSurfaceMorphRequiredReports(t *testing.T, contract gatecontract.Contract, want []string) {
	t.Helper()
	got := make([]string, 0, len(contract.RequiredReports))
	for _, report := range contract.RequiredReports {
		got = append(got, report.Path)
		if !report.SameCommitRequired {
			t.Fatalf("Surface Morph required report %q same_commit_required = false, want true", report.Path)
		}
		if !report.ArtifactHashRequired {
			t.Fatalf("Surface Morph required report %q artifact_hash_required = false, want true", report.Path)
		}
		if len(report.ClaimRefs) == 0 {
			t.Fatalf("Surface Morph required report %q claim_refs is empty", report.Path)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Surface Morph required report paths = %#v, want %#v", got, want)
	}

	wantSchemas := map[string]string{
		"surface-morph-gate-summary.json":      "tetra.surface.morph.gate.v1",
		"headless/surface-headless-morph.json": "tetra.surface.runtime.v1",
		"headless/artifact-hashes.json":        artifacts.HashManifestSchema,
		"artifact-hashes.json":                 artifacts.HashManifestSchema,
	}
	wantValidators := map[string]string{
		"surface-morph-gate-summary.json":      "validate-surface-morph-gate-summary",
		"headless/surface-headless-morph.json": "validate-surface-morph-report",
		"headless/artifact-hashes.json":        "validate-artifact-hashes",
		"artifact-hashes.json":                 "validate-artifact-hashes",
	}
	for path, wantSchema := range wantSchemas {
		report := surfaceMorphRequiredReport(t, contract, path)
		if report.Schema != wantSchema {
			t.Fatalf("Surface Morph required report %q schema = %q, want %q", path, report.Schema, wantSchema)
		}
		if report.Validator != wantValidators[path] {
			t.Fatalf("Surface Morph required report %q validator = %q, want %q", path, report.Validator, wantValidators[path])
		}
	}
}

func assertSurfaceMorphCIArtifacts(t *testing.T, contract gatecontract.Contract, want []string) {
	t.Helper()
	got := make([]string, 0, len(contract.CIArtifacts))
	for _, artifact := range contract.CIArtifacts {
		if !artifact.Required {
			t.Fatalf("Surface Morph ci_artifacts entry %q required = false, want true", artifact.Path)
		}
		got = append(got, artifact.Path)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Surface Morph ci_artifacts paths = %#v, want %#v", got, want)
	}
}

func assertSurfaceMorphValidators(t *testing.T, gate string, contract gatecontract.Contract, ids ...string) {
	t.Helper()
	validators := make(map[string]gatecontract.Validator, len(contract.Validators))
	for _, validator := range contract.Validators {
		if validator.ID == "" {
			t.Fatalf("Surface Morph contract contains validator with empty id")
		}
		if _, exists := validators[validator.ID]; exists {
			t.Fatalf("Surface Morph contract contains duplicate validator %q", validator.ID)
		}
		validators[validator.ID] = validator
	}
	for _, id := range ids {
		validator, ok := validators[id]
		if !ok {
			t.Fatalf("Surface Morph contract missing validator %q", id)
		}
		command := surfaceMorphGateCommandFromContract(validator.Command)
		if !strings.Contains(gate, command) {
			t.Fatalf("Surface Morph gate missing command for contract validator %q: %q", id, command)
		}
	}
}

func surfaceMorphGateCommandFromContract(command string) string {
	command = strings.ReplaceAll(command, "$REPORT_DIR", "$report_dir")
	command = strings.ReplaceAll(command, "$REPO_ROOT", "$repo_root")
	command = strings.ReplaceAll(command, "$GIT_HEAD", "$git_head")
	return command
}

func surfaceMorphRequiredReport(t *testing.T, contract gatecontract.Contract, path string) gatecontract.RequiredReport {
	t.Helper()
	for _, report := range contract.RequiredReports {
		if report.Path == path {
			return report
		}
	}
	t.Fatalf("Surface Morph contract missing required report %q", path)
	return gatecontract.RequiredReport{}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
