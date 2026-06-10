package surfacemigration

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

const (
	SchemaV1                = "tetra.surface.migration-report.v1"
	LevelSurfaceMigrationV1 = "surface-widget-block-migration-v1"
)

type Report struct {
	Schema         string               `json:"schema"`
	Status         string               `json:"status"`
	Level          string               `json:"level"`
	Scope          string               `json:"scope"`
	ReleaseScope   string               `json:"release_scope"`
	Producer       string               `json:"producer,omitempty"`
	GitHead        string               `json:"git_head"`
	SameCommit     bool                 `json:"same_commit"`
	Version        string               `json:"version,omitempty"`
	Policy         MigrationPolicy      `json:"policy"`
	Mappings       []WidgetMapping      `json:"mappings"`
	Examples       []ExampleEvidence    `json:"examples"`
	Diagnostics    []DiagnosticEvidence `json:"diagnostics"`
	NegativeGuards NegativeGuards       `json:"negative_guards"`
	NonClaims      []string             `json:"nonclaims"`
	Cases          []CaseReport         `json:"cases"`
}

type MigrationPolicy struct {
	CompatibilityLayer            bool `json:"compatibility_layer"`
	DocsRecommendBlockMorph       bool `json:"docs_recommend_block_morph"`
	WidgetsCoreFinalArchitecture  bool `json:"widgets_core_final_architecture"`
	DeprecationBeforeCoverage     bool `json:"deprecation_before_coverage"`
	BreakV1ExamplesAllowed        bool `json:"break_v1_examples_allowed"`
	MigrationDiagnosticsAvailable bool `json:"migration_diagnostics_available"`
}

type WidgetMapping struct {
	Widget             string `json:"widget"`
	ComponentKind      string `json:"component_kind"`
	BlockLayout        string `json:"block_layout"`
	MorphRecipe        string `json:"morph_recipe"`
	CompatibilityLayer bool   `json:"compatibility_layer"`
	BlockEquivalent    bool   `json:"block_equivalent"`
	MorphRecommended   bool   `json:"morph_recommended"`
	Deprecated         bool   `json:"deprecated"`
}

type ExampleEvidence struct {
	Path        string `json:"path"`
	Kind        string `json:"kind"`
	Ran         bool   `json:"ran"`
	Pass        bool   `json:"pass"`
	UsesWidgets bool   `json:"uses_widgets"`
	UsesBlock   bool   `json:"uses_block"`
	UsesMorph   bool   `json:"uses_morph"`
}

type DiagnosticEvidence struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Emitted bool   `json:"emitted"`
}

type NegativeGuards struct {
	WidgetsCoreFinalRejected      bool `json:"widgets_core_final_rejected"`
	BreakingV1ExamplesRejected    bool `json:"breaking_v1_examples_rejected"`
	MissingMappingRejected        bool `json:"missing_mapping_rejected"`
	DeprecationBeforeCoverage     bool `json:"deprecation_before_coverage_rejected"`
	MissingBlockMorphDocsRejected bool `json:"missing_block_morph_docs_rejected"`
}

type CaseReport struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Ran  bool   `json:"ran"`
	Pass bool   `json:"pass"`
}

func ValidateReport(raw []byte) error {
	report, err := decodeReport(raw)
	if err != nil {
		return err
	}
	return Validate(report)
}

func Validate(report Report) error {
	var issues []string
	issues = append(issues, validateIdentity(report)...)
	issues = append(issues, validatePolicy(report.Policy)...)
	issues = append(issues, validateMappings(report.Mappings)...)
	issues = append(issues, validateExamples(report.Examples)...)
	issues = append(issues, validateDiagnostics(report.Diagnostics)...)
	issues = append(issues, validateNegativeGuards(report.NegativeGuards)...)
	issues = append(issues, validateNonClaims(report.NonClaims)...)
	issues = append(issues, validateCases(report.Cases)...)
	if len(issues) > 0 {
		sort.Strings(issues)
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func decodeReport(raw []byte) (Report, error) {
	var report Report
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&report); err != nil {
		return Report{}, err
	}
	if err := ensureJSONEOF(dec); err != nil {
		return Report{}, err
	}
	return report, nil
}

func ensureJSONEOF(dec *json.Decoder) error {
	var extra any
	if err := dec.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return fmt.Errorf("unexpected trailing JSON payload")
}

func validateIdentity(report Report) []string {
	var issues []string
	if report.Schema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, SchemaV1))
	}
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if report.Level != LevelSurfaceMigrationV1 {
		issues = append(issues, fmt.Sprintf("level is %q, want %q", report.Level, LevelSurfaceMigrationV1))
	}
	if report.Scope != "surface-v1-widget-compat-to-block-morph" {
		issues = append(issues, fmt.Sprintf("scope is %q, want surface-v1-widget-compat-to-block-morph", report.Scope))
	}
	if report.ReleaseScope != "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI" {
		issues = append(issues, fmt.Sprintf("release_scope is %q, want PROD_STABLE_SCOPED_LINUX_WEB_APP_UI", report.ReleaseScope))
	}
	if !validGitHead(report.GitHead) {
		issues = append(issues, "git_head must be a 40-hex same-commit revision")
	}
	if !report.SameCommit {
		issues = append(issues, "same_commit migration evidence is required")
	}
	return issues
}

func validatePolicy(policy MigrationPolicy) []string {
	var issues []string
	if !policy.CompatibilityLayer {
		issues = append(issues, "widgets must remain a compatibility layer")
	}
	if !policy.DocsRecommendBlockMorph {
		issues = append(issues, "docs must recommend Block/Morph for new production UI")
	}
	if policy.WidgetsCoreFinalArchitecture {
		issues = append(issues, "widgets core final architecture claim is rejected")
	}
	if policy.DeprecationBeforeCoverage {
		issues = append(issues, "deprecation before replacement coverage is rejected")
	}
	if policy.BreakV1ExamplesAllowed {
		issues = append(issues, "breaking v1 examples without migration is rejected")
	}
	if !policy.MigrationDiagnosticsAvailable {
		issues = append(issues, "migration diagnostics are required")
	}
	return issues
}

func validateMappings(mappings []WidgetMapping) []string {
	required := map[string]bool{"Panel": false, "Button": false, "TextBox": false, "StatusText": false}
	if len(mappings) == 0 {
		return []string{"widget-to-Block/Morph mappings are required"}
	}
	var issues []string
	seen := map[string]bool{}
	for _, mapping := range mappings {
		widget := strings.TrimSpace(mapping.Widget)
		if widget == "" {
			issues = append(issues, "mapping widget is required")
			continue
		}
		if seen[widget] {
			issues = append(issues, fmt.Sprintf("duplicate widget mapping %s", widget))
		}
		seen[widget] = true
		if _, ok := required[widget]; ok {
			required[widget] = true
		}
		if strings.TrimSpace(mapping.ComponentKind) == "" {
			issues = append(issues, fmt.Sprintf("mapping %s component_kind is required", widget))
		}
		if strings.TrimSpace(mapping.BlockLayout) == "" {
			issues = append(issues, fmt.Sprintf("mapping %s block_layout is required", widget))
		}
		if strings.TrimSpace(mapping.MorphRecipe) == "" {
			issues = append(issues, fmt.Sprintf("mapping %s morph_recipe is required", widget))
		}
		if !mapping.CompatibilityLayer {
			issues = append(issues, fmt.Sprintf("mapping %s must preserve widget compatibility layer", widget))
		}
		if !mapping.BlockEquivalent {
			issues = append(issues, fmt.Sprintf("mapping %s requires Block equivalent", widget))
		}
		if !mapping.MorphRecommended {
			issues = append(issues, fmt.Sprintf("mapping %s must recommend Morph recipe", widget))
		}
		if mapping.Deprecated {
			issues = append(issues, fmt.Sprintf("mapping %s deprecates widgets before coverage", widget))
		}
	}
	for widget, ok := range required {
		if !ok {
			issues = append(issues, fmt.Sprintf("missing widget mapping %s", widget))
		}
	}
	return issues
}

func validateExamples(examples []ExampleEvidence) []string {
	if len(examples) == 0 {
		return []string{"migration example evidence is required"}
	}
	var issues []string
	seenV1 := false
	seenMigration := false
	for _, example := range examples {
		if err := validateSafeRelPath(example.Path); err != nil {
			issues = append(issues, fmt.Sprintf("example path: %v", err))
		}
		if strings.TrimSpace(example.Kind) == "" {
			issues = append(issues, fmt.Sprintf("example %s kind is required", example.Path))
		}
		if !example.Ran {
			issues = append(issues, fmt.Sprintf("v1 example %s did not run", example.Path))
		}
		if !example.Pass {
			issues = append(issues, fmt.Sprintf("v1 example %s did not pass", example.Path))
		}
		if example.Kind == "v1-widget" && example.UsesWidgets {
			seenV1 = true
		}
		if example.Kind == "migration" && example.UsesWidgets && example.UsesBlock && example.UsesMorph {
			seenMigration = true
		}
	}
	if !seenV1 {
		issues = append(issues, "existing Surface v1 widget example evidence is required")
	}
	if !seenMigration {
		issues = append(issues, "migration example must use widgets, Block, and Morph")
	}
	return issues
}

func validateDiagnostics(diagnostics []DiagnosticEvidence) []string {
	required := map[string]bool{
		"surface.migration.use_block_morph":        false,
		"surface.migration.widgets_not_final_core": false,
	}
	var issues []string
	for _, diagnostic := range diagnostics {
		code := strings.TrimSpace(diagnostic.Code)
		if _, ok := required[code]; ok {
			required[code] = diagnostic.Emitted
		}
		if code == "" || strings.TrimSpace(diagnostic.Message) == "" {
			issues = append(issues, "diagnostic requires code and message")
		}
		if !diagnostic.Emitted {
			issues = append(issues, fmt.Sprintf("diagnostic %s was not emitted", diagnostic.Code))
		}
	}
	for code, ok := range required {
		if !ok {
			issues = append(issues, fmt.Sprintf("missing migration diagnostic %s", code))
		}
	}
	return issues
}

func validateNegativeGuards(guards NegativeGuards) []string {
	required := map[string]bool{
		"widgets core final architecture rejected":         guards.WidgetsCoreFinalRejected,
		"breaking v1 examples rejected":                    guards.BreakingV1ExamplesRejected,
		"missing mapping rejected":                         guards.MissingMappingRejected,
		"deprecation before coverage rejected":             guards.DeprecationBeforeCoverage,
		"missing Block/Morph docs recommendation rejected": guards.MissingBlockMorphDocsRejected,
	}
	var issues []string
	for label, ok := range required {
		if !ok {
			issues = append(issues, label)
		}
	}
	return issues
}

func validateNonClaims(nonclaims []string) []string {
	required := []string{"not the core final architecture", "No deprecation", "No breaking Surface v1 examples", "Block/Morph recipes"}
	haystack := strings.Join(nonclaims, "\n")
	var issues []string
	for _, want := range required {
		if !strings.Contains(haystack, want) {
			issues = append(issues, fmt.Sprintf("missing nonclaim containing %q", want))
		}
	}
	return issues
}

func validateCases(cases []CaseReport) []string {
	required := map[string]bool{
		"existing Surface v1 widget examples still pass":  false,
		"widgets map to Block/Morph recipes":              false,
		"widgets as core final architecture rejected":     false,
		"breaking v1 examples without migration rejected": false,
	}
	var issues []string
	for _, c := range cases {
		if _, ok := required[c.Name]; ok {
			required[c.Name] = c.Ran && c.Pass
		}
		if strings.TrimSpace(c.Name) == "" || strings.TrimSpace(c.Kind) == "" {
			issues = append(issues, "case report requires name and kind")
		}
		if !c.Ran || !c.Pass {
			issues = append(issues, fmt.Sprintf("case %q must run and pass", c.Name))
		}
	}
	for name, ok := range required {
		if !ok {
			issues = append(issues, fmt.Sprintf("missing or failed case %q", name))
		}
	}
	return issues
}

func validateSafeRelPath(path string) error {
	if path == "" {
		return fmt.Errorf("path is required")
	}
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute path is forbidden")
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") || clean == ".." {
		return fmt.Errorf("path must stay inside repo/report root")
	}
	return nil
}

func validGitHead(value string) bool {
	if len(value) != 40 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
