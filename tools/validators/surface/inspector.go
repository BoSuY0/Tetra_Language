package surface

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

const InspectorSchemaV1 = "tetra.surface.inspector.v1"

type SurfaceInspectorReport struct {
	Schema          string                           `json:"schema"`
	Model           string                           `json:"model"`
	ReleaseScope    string                           `json:"release_scope"`
	Producer        string                           `json:"producer"`
	Source          string                           `json:"source"`
	Target          string                           `json:"target"`
	Mode            string                           `json:"mode"`
	InputReports    []SurfaceInspectorInputReport    `json:"input_reports"`
	SourceLocations []SurfaceInspectorSourceLocation `json:"source_locations"`
	Sections        SurfaceInspectorSections         `json:"sections"`
	MorphToPixels   *MorphToPixelsChainReport        `json:"morph_to_pixels,omitempty"`
	StaticArtifacts SurfaceInspectorStaticArtifacts  `json:"static_artifacts"`
	HiddenState     SurfaceInspectorHiddenState      `json:"hidden_state"`
	NegativeGuards  SurfaceInspectorNegativeGuards   `json:"negative_guards"`
	Pass            bool                             `json:"pass"`
}

type SurfaceInspectorInputReport struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Schema string `json:"schema"`
	Source string `json:"source"`
	Target string `json:"target"`
	Pass   bool   `json:"pass"`
}

type SurfaceInspectorSourceLocation struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

type SurfaceInspectorSections struct {
	BlockTree        SurfaceInspectorSection `json:"block_tree"`
	MorphTokens      SurfaceInspectorSection `json:"morph_tokens"`
	Layout           SurfaceInspectorSection `json:"layout"`
	Paint            SurfaceInspectorSection `json:"paint"`
	Accessibility    SurfaceInspectorSection `json:"accessibility"`
	EventRoutes      SurfaceInspectorSection `json:"event_routes"`
	Focus            SurfaceInspectorSection `json:"focus"`
	PerfCounters     SurfaceInspectorSection `json:"perf_counters"`
	RecipeExpansions SurfaceInspectorSection `json:"recipe_expansions,omitempty"`
	BlockSceneNodes  SurfaceInspectorSection `json:"block_scene_nodes,omitempty"`
	RenderCommands   SurfaceInspectorSection `json:"render_commands,omitempty"`
	FrameArtifacts   SurfaceInspectorSection `json:"frame_artifacts,omitempty"`
	GoldenDiff       SurfaceInspectorSection `json:"golden_diff,omitempty"`
}

type SurfaceInspectorSection struct {
	Present bool   `json:"present"`
	Count   int    `json:"count"`
	Source  string `json:"source"`
}

type SurfaceInspectorStaticArtifacts struct {
	JSON           string `json:"json"`
	HTML           string `json:"html"`
	HTMLToolReport bool   `json:"html_tool_report"`
}

type SurfaceInspectorHiddenState struct {
	Scanned  bool                                 `json:"scanned"`
	Findings []SurfaceInspectorHiddenStateFinding `json:"findings"`
}

type SurfaceInspectorHiddenStateFinding struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

type SurfaceInspectorNegativeGuards struct {
	NoDOMRuntimeDependency      bool `json:"no_dom_runtime_dependency"`
	NoBrowserDevtoolsDependency bool `json:"no_browser_devtools_dependency"`
	NoReactDevtoolsDependency   bool `json:"no_react_devtools_dependency"`
	StaticHTMLToolReportOnly    bool `json:"static_html_tool_report_only"`
	NoHiddenState               bool `json:"no_hidden_state"`
}

func ValidateInspectorReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != InspectorSchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, InspectorSchemaV1)
	}

	var report SurfaceInspectorReport
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}

	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: report.Schema, want: InspectorSchemaV1},
		{field: "model", got: report.Model, want: "surface-inspector-v1"},
		{field: "release_scope", got: report.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
		{field: "producer", got: report.Producer, want: "tools/cmd/surface-inspector"},
		{field: "target", got: report.Target, want: "headless"},
		{field: "mode", got: report.Mode, want: "static-tool-report"},
	} {
		if check.got != check.want {
			issues = append(issues, fmt.Sprintf("%s is %q, want %q", check.field, check.got, check.want))
		}
	}
	if strings.TrimSpace(report.Source) == "" {
		issues = append(issues, "source is required")
	}
	issues = append(issues, validateSurfaceInspectorInputReports(report.InputReports)...)
	issues = append(issues, validateSurfaceInspectorSourceLocations(report.SourceLocations)...)
	issues = append(issues, validateSurfaceInspectorSections(report.Sections)...)
	issues = append(issues, validateSurfaceInspectorMorphToPixels(report.InputReports, report.Sections, report.MorphToPixels)...)
	issues = append(issues, validateSurfaceInspectorStaticArtifacts(report.StaticArtifacts)...)
	issues = append(issues, validateSurfaceInspectorHiddenState(report.HiddenState)...)
	issues = append(issues, validateSurfaceInspectorNegativeGuards(report.NegativeGuards)...)
	if !report.Pass {
		issues = append(issues, "pass must be true")
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateSurfaceInspectorInputReports(reports []SurfaceInspectorInputReport) []string {
	if len(reports) == 0 {
		return []string{"input_reports are required"}
	}
	required := map[string]bool{
		"block":         false,
		"morph":         false,
		"accessibility": false,
		"app-model":     false,
	}
	var issues []string
	for _, report := range reports {
		kind := strings.TrimSpace(report.Kind)
		if kind == "" {
			issues = append(issues, "input_reports kind is required")
			continue
		}
		if _, ok := required[kind]; ok {
			required[kind] = true
		}
		if !safeRelativeReportPath(report.Path) {
			issues = append(issues, fmt.Sprintf("input_reports %s path is unsafe or empty", kind))
		}
		switch kind {
		case "morph-rendered-beauty":
			if report.Schema != MorphRenderedBeautyReportSchemaV1 {
				issues = append(issues, fmt.Sprintf("input_reports %s schema is %q, want %q", kind, report.Schema, MorphRenderedBeautyReportSchemaV1))
			}
		default:
			if report.Schema != SchemaV1 {
				issues = append(issues, fmt.Sprintf("input_reports %s schema is %q, want %q", kind, report.Schema, SchemaV1))
			}
		}
		if strings.TrimSpace(report.Source) == "" {
			issues = append(issues, fmt.Sprintf("input_reports %s source is required", kind))
		}
		if kind == "morph-rendered-beauty" {
			if !containsMorphRenderedBeautyText([]string{"headless", "linux-x64-real-window", "wasm32-web-browser-canvas"}, report.Target) {
				issues = append(issues, fmt.Sprintf("input_reports %s target %q is unsupported", kind, report.Target))
			}
		} else if report.Target != "headless" {
			issues = append(issues, fmt.Sprintf("input_reports %s target is %q, want headless", kind, report.Target))
		}
		if !report.Pass {
			issues = append(issues, fmt.Sprintf("input_reports %s pass must be true", kind))
		}
	}
	for kind, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("input_reports missing %s", kind))
		}
	}
	return issues
}

func validateSurfaceInspectorSourceLocations(locations []SurfaceInspectorSourceLocation) []string {
	if len(locations) == 0 {
		return []string{"source_locations are required"}
	}
	required := map[string]bool{
		"block":         false,
		"morph":         false,
		"accessibility": false,
		"app-model":     false,
	}
	hasMorphRenderedBeauty := false
	var issues []string
	for _, location := range locations {
		kind := strings.TrimSpace(location.Kind)
		if kind == "" {
			issues = append(issues, "source_locations kind is required")
			continue
		}
		if _, ok := required[kind]; ok {
			required[kind] = true
		}
		if kind == "morph-rendered-beauty" {
			hasMorphRenderedBeauty = true
		}
		if !safeRelativeSourcePath(location.Path) {
			issues = append(issues, fmt.Sprintf("source_locations %s path is unsafe or not a Tetra source", kind))
		}
		if location.Line <= 0 || location.Column <= 0 {
			issues = append(issues, fmt.Sprintf("source_locations %s requires positive line and column", kind))
		}
	}
	for kind, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("source_locations missing %s", kind))
		}
	}
	if hasMorphRenderedBeauty {
		// The generic loop above already validates path/line/column for the
		// optional MRB source row. No extra issue is needed here.
	}
	return issues
}

func validateSurfaceInspectorSections(sections SurfaceInspectorSections) []string {
	var issues []string
	for _, check := range []struct {
		name    string
		section SurfaceInspectorSection
	}{
		{name: "block_tree", section: sections.BlockTree},
		{name: "morph_tokens", section: sections.MorphTokens},
		{name: "layout", section: sections.Layout},
		{name: "paint", section: sections.Paint},
		{name: "accessibility", section: sections.Accessibility},
		{name: "event_routes", section: sections.EventRoutes},
		{name: "focus", section: sections.Focus},
		{name: "perf_counters", section: sections.PerfCounters},
	} {
		if !check.section.Present {
			issues = append(issues, fmt.Sprintf("sections.%s present must be true", check.name))
		}
		if check.section.Count <= 0 {
			issues = append(issues, fmt.Sprintf("sections.%s count must be positive", check.name))
		}
		if strings.TrimSpace(check.section.Source) == "" {
			issues = append(issues, fmt.Sprintf("sections.%s source is required", check.name))
		}
	}
	return issues
}

func validateSurfaceInspectorMorphToPixels(inputs []SurfaceInspectorInputReport, sections SurfaceInspectorSections, chain *MorphToPixelsChainReport) []string {
	var issues []string
	expectedSource := ""
	hasMorphRenderedBeauty := false
	for _, input := range inputs {
		if input.Kind == "morph-rendered-beauty" {
			hasMorphRenderedBeauty = true
			expectedSource = input.Source
			break
		}
	}
	if !hasMorphRenderedBeauty && chain == nil {
		return nil
	}
	if hasMorphRenderedBeauty && chain == nil {
		return []string{"morph_to_pixels is required when morph-rendered-beauty input report is present"}
	}
	if chain == nil {
		return []string{"morph_to_pixels requires a morph-rendered-beauty input report"}
	}
	issues = append(issues, validateMorphToPixelsChain("morph_to_pixels", *chain, expectedSource)...)
	for _, check := range []struct {
		name    string
		section SurfaceInspectorSection
	}{
		{name: "recipe_expansions", section: sections.RecipeExpansions},
		{name: "block_scene_nodes", section: sections.BlockSceneNodes},
		{name: "render_commands", section: sections.RenderCommands},
		{name: "frame_artifacts", section: sections.FrameArtifacts},
		{name: "golden_diff", section: sections.GoldenDiff},
	} {
		if !check.section.Present {
			issues = append(issues, fmt.Sprintf("sections.%s present must be true when morph_to_pixels is present", check.name))
		}
		if check.section.Count <= 0 {
			issues = append(issues, fmt.Sprintf("sections.%s count must be positive when morph_to_pixels is present", check.name))
		}
		if strings.TrimSpace(check.section.Source) == "" {
			issues = append(issues, fmt.Sprintf("sections.%s source is required when morph_to_pixels is present", check.name))
		}
	}
	return issues
}

func validateSurfaceInspectorStaticArtifacts(artifacts SurfaceInspectorStaticArtifacts) []string {
	var issues []string
	if !safeRelativeReportPath(artifacts.JSON) {
		issues = append(issues, "static_artifacts.json is unsafe or empty")
	}
	if strings.TrimSpace(artifacts.HTML) != "" && !safeRelativeReportPath(artifacts.HTML) {
		issues = append(issues, "static_artifacts.html is unsafe")
	}
	if strings.TrimSpace(artifacts.HTML) != "" && !artifacts.HTMLToolReport {
		issues = append(issues, "static_artifacts.html_tool_report must be true when html is present")
	}
	return issues
}

func validateSurfaceInspectorHiddenState(hidden SurfaceInspectorHiddenState) []string {
	if !hidden.Scanned {
		return []string{"hidden_state.scanned must be true"}
	}
	if len(hidden.Findings) != 0 {
		return []string{"hidden_state findings must be empty"}
	}
	return nil
}

func validateSurfaceInspectorNegativeGuards(guards SurfaceInspectorNegativeGuards) []string {
	if guards.NoDOMRuntimeDependency &&
		guards.NoBrowserDevtoolsDependency &&
		guards.NoReactDevtoolsDependency &&
		guards.StaticHTMLToolReportOnly &&
		guards.NoHiddenState {
		return nil
	}
	return []string{"negative_guards must reject DOM runtime, browser devtools, React devtools, non-static HTML reports, and hidden state"}
}

func safeRelativeReportPath(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" || filepath.IsAbs(path) {
		return false
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return false
	}
	return true
}

func safeRelativeSourcePath(path string) bool {
	if !safeRelativeReportPath(path) {
		return false
	}
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".tetra" || ext == ".t4"
}
