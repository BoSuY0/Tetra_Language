package surfaceexamples

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
	SchemaV1                         = "tetra.surface.example-suite-report.v1"
	LevelSurfaceProductionExamplesV1 = "surface-production-example-suite-v1"
)

type Report struct {
	Schema         string            `json:"schema"`
	Status         string            `json:"status"`
	Level          string            `json:"level"`
	Scope          string            `json:"scope"`
	ReleaseScope   string            `json:"release_scope"`
	Producer       string            `json:"producer,omitempty"`
	GitHead        string            `json:"git_head"`
	SameCommit     bool              `json:"same_commit"`
	Version        string            `json:"version,omitempty"`
	Examples       []ExampleEvidence `json:"examples"`
	Targets        []TargetEvidence  `json:"targets"`
	Ecosystem      EcosystemSeed     `json:"ecosystem"`
	NegativeGuards NegativeGuards    `json:"negative_guards"`
	NonClaims      []string          `json:"nonclaims"`
	Cases          []CaseReport      `json:"cases"`
}

type ExampleEvidence struct {
	Path                   string `json:"path"`
	Shape                  string `json:"shape"`
	Ran                    bool   `json:"ran"`
	Pass                   bool   `json:"pass"`
	Executable             bool   `json:"executable"`
	UsesBlock              bool   `json:"uses_block"`
	UsesMorph              bool   `json:"uses_morph"`
	UsesWidgets            bool   `json:"uses_widgets"`
	RequiresReact          bool   `json:"requires_react"`
	RequiresElectron       bool   `json:"requires_electron"`
	RequiresDOMRuntime     bool   `json:"requires_dom_runtime"`
	ScreenshotOnly         bool   `json:"screenshot_only"`
	HasEvents              bool   `json:"has_events"`
	HasState               bool   `json:"has_state"`
	HasAccessibility       bool   `json:"has_accessibility"`
	HasPerformanceBudget   bool   `json:"has_performance_budget"`
	HasLocalization        bool   `json:"has_localization,omitempty"`
	HasAccessibilityStress bool   `json:"has_accessibility_stress,omitempty"`
}

type TargetEvidence struct {
	Target       string `json:"target"`
	ExampleCount int    `json:"example_count"`
	Ran          bool   `json:"ran"`
	Pass         bool   `json:"pass"`
	Artifact     string `json:"artifact"`
}

type EcosystemSeed struct {
	TemplateCount        int  `json:"template_count"`
	PackageReportCount   int  `json:"package_report_count"`
	ExamplesIndexUpdated bool `json:"examples_index_updated"`
	SurfaceGuideUpdated  bool `json:"surface_guide_updated"`
	ScaffoldSmokeRan     bool `json:"scaffold_smoke_ran"`
	PackageSmokeRan      bool `json:"package_smoke_ran"`
}

type NegativeGuards struct {
	ScreenshotOnlyRejected                 bool `json:"screenshot_only_rejected"`
	ReactElectronDOMRejected               bool `json:"react_electron_dom_rejected"`
	WidgetsWhereBlockMorphRequiredRejected bool `json:"widgets_where_block_morph_required_rejected"`
	MissingShapeRejected                   bool `json:"missing_shape_rejected"`
	MissingTargetCoverageRejected          bool `json:"missing_target_coverage_rejected"`
	ToyVisualOnlyRejected                  bool `json:"toy_visual_only_rejected"`
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
	issues = append(issues, validateExamples(report.Examples)...)
	issues = append(issues, validateTargets(report.Targets)...)
	issues = append(issues, validateEcosystem(report.Ecosystem)...)
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
	if report.Level != LevelSurfaceProductionExamplesV1 {
		issues = append(issues, fmt.Sprintf("level is %q, want %q", report.Level, LevelSurfaceProductionExamplesV1))
	}
	if report.Scope != "surface-prod-realistic-app-shapes-linux-web" {
		issues = append(issues, fmt.Sprintf("scope is %q, want surface-prod-realistic-app-shapes-linux-web", report.Scope))
	}
	if report.ReleaseScope != "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI" {
		issues = append(issues, fmt.Sprintf("release_scope is %q, want PROD_STABLE_SCOPED_LINUX_WEB_APP_UI", report.ReleaseScope))
	}
	if !validGitHead(report.GitHead) {
		issues = append(issues, "git_head must be a 40-hex same-commit revision")
	}
	if !report.SameCommit {
		issues = append(issues, "same_commit example suite evidence is required")
	}
	return issues
}

func validateExamples(examples []ExampleEvidence) []string {
	required := requiredShapes()
	if len(examples) == 0 {
		return []string{"production example evidence is required"}
	}
	var issues []string
	seen := map[string]bool{}
	for _, example := range examples {
		path := strings.TrimSpace(filepath.ToSlash(example.Path))
		shape := strings.TrimSpace(example.Shape)
		if path == "" {
			issues = append(issues, "example path is required")
			continue
		}
		if seen[path] {
			issues = append(issues, fmt.Sprintf("duplicate example path %s", path))
		}
		seen[path] = true
		if !strings.HasPrefix(path, "examples/surface_prod_") || !strings.HasSuffix(path, ".tetra") {
			issues = append(issues, fmt.Sprintf("example %s must be a surface_prod_*.tetra executable", path))
		}
		if _, ok := required[shape]; ok {
			required[shape] = true
		} else {
			issues = append(issues, fmt.Sprintf("example %s has unsupported production shape %q", path, shape))
		}
		if !example.Ran || !example.Pass || !example.Executable {
			issues = append(issues, fmt.Sprintf("example %s must run and pass as an executable example", path))
		}
		if example.ScreenshotOnly {
			issues = append(issues, fmt.Sprintf("example %s is screenshot-only evidence", path))
		}
		if !example.UsesBlock || !example.UsesMorph || example.UsesWidgets {
			issues = append(issues, fmt.Sprintf("example %s must use Block/Morph and avoid widgets where Block/Morph is required", path))
		}
		if example.RequiresReact || example.RequiresElectron || example.RequiresDOMRuntime {
			issues = append(issues, fmt.Sprintf("example %s requires React/Electron/DOM runtime", path))
		}
		if !example.HasEvents || !example.HasState || !example.HasAccessibility || !example.HasPerformanceBudget {
			issues = append(issues, fmt.Sprintf("example %s must include events, state, accessibility, and performance budget evidence", path))
		}
		if shape == "localized_form" && !example.HasLocalization {
			issues = append(issues, fmt.Sprintf("example %s must include localization evidence", path))
		}
		if shape == "accessibility_heavy_form" && !example.HasAccessibilityStress {
			issues = append(issues, fmt.Sprintf("example %s must include accessibility-heavy evidence", path))
		}
	}
	for shape, found := range required {
		if !found {
			issues = append(issues, fmt.Sprintf("missing production example shape %s", shape))
		}
	}
	return issues
}

func validateTargets(targets []TargetEvidence) []string {
	required := map[string]bool{"headless": false, "linux-x64": false, "wasm32-web": false}
	if len(targets) == 0 {
		return []string{"target coverage evidence is required"}
	}
	var issues []string
	for _, target := range targets {
		name := strings.TrimSpace(target.Target)
		if _, ok := required[name]; ok {
			required[name] = true
		} else {
			issues = append(issues, fmt.Sprintf("unsupported target coverage %q", name))
		}
		if !target.Ran || !target.Pass {
			issues = append(issues, fmt.Sprintf("target %s coverage must run and pass", name))
		}
		if target.ExampleCount < len(requiredShapes()) {
			issues = append(issues, fmt.Sprintf("target %s covers %d examples, want at least %d", name, target.ExampleCount, len(requiredShapes())))
		}
		if strings.TrimSpace(target.Artifact) == "" {
			issues = append(issues, fmt.Sprintf("target %s artifact is required", name))
		}
	}
	for target, found := range required {
		if !found {
			issues = append(issues, fmt.Sprintf("missing supported target coverage %s", target))
		}
	}
	return issues
}

func validateEcosystem(seed EcosystemSeed) []string {
	var issues []string
	if seed.TemplateCount < 6 {
		issues = append(issues, "ecosystem seed requires at least six Surface templates")
	}
	if seed.PackageReportCount < 2 {
		issues = append(issues, "ecosystem seed requires package report evidence")
	}
	if !seed.ExamplesIndexUpdated {
		issues = append(issues, "examples index update evidence is required")
	}
	if !seed.SurfaceGuideUpdated {
		issues = append(issues, "surface guide update evidence is required")
	}
	if !seed.ScaffoldSmokeRan {
		issues = append(issues, "scaffold smoke evidence is required")
	}
	if !seed.PackageSmokeRan {
		issues = append(issues, "package smoke evidence is required")
	}
	return issues
}

func validateNegativeGuards(guards NegativeGuards) []string {
	required := map[string]bool{
		"screenshot-only examples rejected":                guards.ScreenshotOnlyRejected,
		"React/Electron/DOM runtime examples rejected":     guards.ReactElectronDOMRejected,
		"widgets instead of Block/Morph examples rejected": guards.WidgetsWhereBlockMorphRequiredRejected,
		"missing production shape rejected":                guards.MissingShapeRejected,
		"missing target coverage rejected":                 guards.MissingTargetCoverageRejected,
		"toy visual-only examples rejected":                guards.ToyVisualOnlyRejected,
	}
	var issues []string
	for name, ok := range required {
		if !ok {
			issues = append(issues, name+" guard is required")
		}
	}
	return issues
}

func validateNonClaims(nonclaims []string) []string {
	joined := strings.Join(nonclaims, "\n")
	required := []string{
		"do not claim broad cross-platform parity",
		"do not require React, Electron, DOM runtime UI, external CSS, or platform widgets",
		"Screenshot-only demos are not production example evidence",
	}
	var issues []string
	for _, want := range required {
		if !strings.Contains(joined, want) {
			issues = append(issues, fmt.Sprintf("nonclaims must include %q", want))
		}
	}
	return issues
}

func validateCases(cases []CaseReport) []string {
	required := map[string]bool{
		"ten realistic app shapes":                     false,
		"all scoped targets covered":                   false,
		"screenshot-only examples rejected":            false,
		"React/Electron/DOM runtime examples rejected": false,
		"widgets where Block/Morph required rejected":  false,
	}
	var issues []string
	for _, c := range cases {
		name := strings.TrimSpace(c.Name)
		if _, ok := required[name]; ok {
			required[name] = true
		}
		if name == "" || strings.TrimSpace(c.Kind) == "" {
			issues = append(issues, "case name and kind are required")
		}
		if !c.Ran || !c.Pass {
			issues = append(issues, fmt.Sprintf("case %s must run and pass", name))
		}
	}
	for name, found := range required {
		if !found {
			issues = append(issues, fmt.Sprintf("missing case %s", name))
		}
	}
	return issues
}

func requiredShapes() map[string]bool {
	return map[string]bool{
		"command_palette":          false,
		"settings":                 false,
		"project_dashboard":        false,
		"editor_shell":             false,
		"file_manager_shell":       false,
		"multi_window_notes":       false,
		"system_tray_status":       false,
		"notification_dialog":      false,
		"localized_form":           false,
		"accessibility_heavy_form": false,
	}
}

func validGitHead(value string) bool {
	if len(value) != 40 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
