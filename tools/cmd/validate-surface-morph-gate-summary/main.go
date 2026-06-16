package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

const (
	morphGateSummarySchema   = "tetra.surface.morph.gate.v1"
	morphGateReleaseScope    = "surface-morph-experimental-linux-web"
	morphGateProducer        = "scripts/release/surface/morph-gate.sh"
	morphGateSource          = "examples/surface_morph_command_palette.tetra"
	morphGateModule          = "lib.core.morph"
	morphGateSchemaUnderTest = "tetra.surface.morph.v1"
)

var morphGateReferenceRecipeApps = []string{
	"examples/surface_morph_command_palette.tetra",
	"examples/surface_morph_project_dashboard.tetra",
	"examples/surface_morph_settings.tetra",
	"examples/surface_morph_editor_shell.tetra",
	"examples/surface_morph_glass_panel.tetra",
	"examples/surface_morph_studio_shell.tetra",
}

type morphGateSummaryValidationOptions struct {
	SameCommit string
}

type surfaceMorphGateSummary struct {
	Schema                   string   `json:"schema"`
	Status                   string   `json:"status"`
	ReleaseScope             string   `json:"release_scope"`
	Producer                 string   `json:"producer"`
	GitHead                  string   `json:"git_head"`
	Version                  string   `json:"version"`
	GitDirty                 *bool    `json:"git_dirty"`
	HostOS                   string   `json:"host_os"`
	HostArch                 string   `json:"host_arch"`
	GeneratedAtUTC           string   `json:"generated_at_utc"`
	CommandLine              string   `json:"command_line"`
	Source                   string   `json:"source"`
	Module                   string   `json:"module"`
	SchemaUnderTest          string   `json:"schema_under_test"`
	TokenGraphContract       string   `json:"token_graph_contract"`
	TokenGraphValidator      string   `json:"token_graph_validator"`
	RecipeAuthoringValidator string   `json:"recipe_authoring_validator"`
	RecipeExpansionReport    string   `json:"recipe_expansion_report"`
	RecipeCount              int      `json:"recipe_count"`
	ReferenceRecipeApps      []string `json:"reference_recipe_apps"`
	DependencyGate           string   `json:"dependency_gate"`
	SameCommitValidated      *bool    `json:"same_commit_validated"`
	HeadlessReport           string   `json:"headless_report"`
	TargetEvidence           []string `json:"target_evidence"`
	CorePrimitives           []string `json:"core_primitives"`
	ForbiddenCorePrimitives  []string `json:"forbidden_core_primitives"`
	ArtifactHashesValidated  *bool    `json:"artifact_hashes_validated"`
}

func main() {
	summaryPath := flag.String("summary", "", "path to tetra.surface.morph.gate.v1 summary")
	sameCommit := flag.String("same-commit", "", "require the summary to validate at this git commit")
	flag.Parse()
	if *summaryPath == "" {
		fmt.Fprintln(os.Stderr, "error: --summary is required")
		os.Exit(2)
	}
	if err := validateSurfaceMorphGateSummaryWithOptions(*summaryPath, morphGateSummaryValidationOptions{SameCommit: *sameCommit}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateSurfaceMorphGateSummary(path string) error {
	return validateSurfaceMorphGateSummaryWithOptions(path, morphGateSummaryValidationOptions{})
}

func validateSurfaceMorphGateSummaryWithOptions(path string, options morphGateSummaryValidationOptions) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("summary path is required")
	}

	report, err := decodeSurfaceMorphGateSummary(path)
	if err != nil {
		return err
	}
	var issues []string
	issues = append(issues, requireString("schema", report.Schema, morphGateSummarySchema)...)
	issues = append(issues, requireString("status", report.Status, "current")...)
	issues = append(issues, requireString("release_scope", report.ReleaseScope, morphGateReleaseScope)...)
	issues = append(issues, requireString("producer", report.Producer, morphGateProducer)...)
	issues = append(issues, requireString("source", report.Source, morphGateSource)...)
	issues = append(issues, requireString("module", report.Module, morphGateModule)...)
	issues = append(issues, requireString("schema_under_test", report.SchemaUnderTest, morphGateSchemaUnderTest)...)
	issues = append(issues, requireString("token_graph_contract", report.TokenGraphContract, "docs/spec/surface_token_graph_contract.json")...)
	issues = append(issues, requireString("token_graph_validator", report.TokenGraphValidator, "validate-surface-token-graph")...)
	issues = append(issues, requireString("recipe_authoring_validator", report.RecipeAuthoringValidator, "validate-surface-morph-report")...)
	issues = append(issues, requireString("recipe_expansion_report", report.RecipeExpansionReport, "headless/surface-headless-morph.json#morph.recipe_expansions")...)
	issues = append(issues, requireString("dependency_gate", report.DependencyGate, "tetra.surface.block-system.gate.v1")...)
	issues = append(issues, requireString("headless_report", report.HeadlessReport, "headless/surface-headless-morph.json")...)
	for _, check := range []struct {
		field string
		value string
	}{
		{field: "git_head", value: report.GitHead},
		{field: "version", value: report.Version},
		{field: "host_os", value: report.HostOS},
		{field: "host_arch", value: report.HostArch},
		{field: "generated_at_utc", value: report.GeneratedAtUTC},
		{field: "command_line", value: report.CommandLine},
	} {
		if strings.TrimSpace(check.value) == "" {
			issues = append(issues, fmt.Sprintf("%s is missing or empty", check.field))
		}
	}
	if report.RecipeCount != 19 {
		issues = append(issues, fmt.Sprintf("recipe_count is %d, want 19", report.RecipeCount))
	}
	issues = append(issues, requireBool("git_dirty", report.GitDirty, nil)...)
	issues = append(issues, requireBool("same_commit_validated", report.SameCommitValidated, boolPtr(true))...)
	issues = append(issues, requireBool("artifact_hashes_validated", report.ArtifactHashesValidated, boolPtr(true))...)
	issues = append(issues, requireStringSlice("reference_recipe_apps", report.ReferenceRecipeApps, morphGateReferenceRecipeApps)...)
	issues = append(issues, requireStringSlice("target_evidence", report.TargetEvidence, []string{"headless"})...)
	issues = append(issues, requireStringSlice("core_primitives", report.CorePrimitives, []string{"Block"})...)
	if len(report.ForbiddenCorePrimitives) == 0 {
		issues = append(issues, "forbidden_core_primitives must not be empty")
	}
	for i, primitive := range report.ForbiddenCorePrimitives {
		if strings.TrimSpace(primitive) == "" {
			issues = append(issues, fmt.Sprintf("forbidden_core_primitives[%d] is empty", i))
		}
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	if strings.TrimSpace(options.SameCommit) != "" {
		if err := validateSameCommit(options.SameCommit, report.GitHead); err != nil {
			return fmt.Errorf("same-commit git_head validation: %w", err)
		}
		actual, err := currentGitCommit()
		if err != nil {
			return err
		}
		if err := validateSameCommit(options.SameCommit, actual); err != nil {
			return err
		}
	}
	return nil
}

func decodeSurfaceMorphGateSummary(path string) (surfaceMorphGateSummary, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return surfaceMorphGateSummary{}, fmt.Errorf("read surface morph gate summary: %w", err)
	}
	var report surfaceMorphGateSummary
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		return surfaceMorphGateSummary{}, fmt.Errorf("decode surface morph gate summary: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return surfaceMorphGateSummary{}, fmt.Errorf("decode surface morph gate summary: unexpected trailing JSON value")
		}
		return surfaceMorphGateSummary{}, fmt.Errorf("decode surface morph gate summary: %w", err)
	}
	return report, nil
}

func requireString(field string, got string, want string) []string {
	if got != want {
		return []string{fmt.Sprintf("%s is %q, want %q", field, got, want)}
	}
	return nil
}

func requireBool(field string, got *bool, want *bool) []string {
	if got == nil {
		return []string{fmt.Sprintf("%s is missing", field)}
	}
	if want != nil && *got != *want {
		return []string{fmt.Sprintf("%s is %t, want %t", field, *got, *want)}
	}
	return nil
}

func requireStringSlice(field string, got []string, want []string) []string {
	if len(got) != len(want) {
		return []string{fmt.Sprintf("%s has %d entries, want %d", field, len(got), len(want))}
	}
	for i := range want {
		if got[i] != want[i] {
			return []string{fmt.Sprintf("%s[%d] is %q, want %q", field, i, got[i], want[i])}
		}
	}
	return nil
}

func boolPtr(value bool) *bool {
	return &value
}

func validateSameCommit(expected string, actual string) error {
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)
	if expected == "" {
		return nil
	}
	if actual == "" {
		return fmt.Errorf("same-commit validation requires current git commit evidence")
	}
	if expected == actual || strings.HasPrefix(actual, expected) || strings.HasPrefix(expected, actual) {
		return nil
	}
	return fmt.Errorf("same-commit mismatch: expected %s, got %s", expected, actual)
}

func currentGitCommit() (string, error) {
	raw, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("read current git commit: %w", err)
	}
	return strings.TrimSpace(string(raw)), nil
}
