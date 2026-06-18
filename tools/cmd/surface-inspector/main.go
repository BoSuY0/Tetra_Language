package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tetra_language/tools/validators/surface"
)

type runtimeReportFlag []runtimeReportInput

type runtimeReportInput struct {
	Kind string
	Path string
}

type runtimeReport struct {
	Input               runtimeReportInput
	Raw                 map[string]any
	Schema              string
	Status              string
	Source              string
	Target              string
	MorphRenderedBeauty *surface.MorphRenderedBeautyReport
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func (f *runtimeReportFlag) String() string {
	parts := make([]string, 0, len(*f))
	for _, input := range *f {
		parts = append(parts, input.Kind+":"+input.Path)
	}
	return strings.Join(parts, ",")
}

func (f *runtimeReportFlag) Set(value string) error {
	kind, path, ok := strings.Cut(value, ":")
	if !ok {
		return fmt.Errorf("--runtime-report must be kind:path")
	}
	kind = strings.TrimSpace(kind)
	path = strings.TrimSpace(path)
	if kind == "" || path == "" {
		return fmt.Errorf("--runtime-report must include non-empty kind and path")
	}
	allowed := map[string]bool{
		"block":                 true,
		"morph":                 true,
		"morph-rendered-beauty": true,
		"app-model":             true,
		"accessibility":         true,
		"events":                true,
	}
	if !allowed[kind] {
		return fmt.Errorf("unsupported runtime report kind %q", kind)
	}
	*f = append(*f, runtimeReportInput{Kind: kind, Path: path})
	return nil
}

func run(args []string) error {
	fs := flag.NewFlagSet("surface-inspector", flag.ContinueOnError)
	var reportFlags runtimeReportFlag
	outPath := fs.String("out", "", "path to write surface-inspector.json")
	htmlPath := fs.String("html", "", "optional path to write a static HTML tool report")
	fs.Var(&reportFlags, "runtime-report", "runtime report input as kind:path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *outPath == "" {
		return fmt.Errorf("--out is required")
	}
	if len(reportFlags) == 0 {
		return fmt.Errorf("--runtime-report is required")
	}

	reports, err := readRuntimeReports(reportFlags)
	if err != nil {
		return err
	}
	report, err := buildInspectorReport(reports, *outPath, *htmlPath)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := surface.ValidateInspectorReport(raw); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(*outPath, raw, 0o644); err != nil {
		return err
	}
	if strings.TrimSpace(*htmlPath) != "" {
		if err := os.MkdirAll(filepath.Dir(*htmlPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(*htmlPath, []byte(renderInspectorHTML(report)), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func readRuntimeReports(inputs []runtimeReportInput) ([]runtimeReport, error) {
	reports := make([]runtimeReport, 0, len(inputs))
	for _, input := range inputs {
		raw, err := os.ReadFile(input.Path)
		if err != nil {
			return nil, fmt.Errorf("%s read failed: %w", input.Path, err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return nil, fmt.Errorf("%s decode failed: %w", input.Path, err)
		}
		if err := rejectHiddenState(input.Path, decoded); err != nil {
			return nil, err
		}
		report := runtimeReport{
			Input:  input,
			Raw:    decoded,
			Schema: stringField(decoded, "schema"),
			Status: stringField(decoded, "status"),
			Source: stringField(decoded, "source"),
			Target: stringField(decoded, "target"),
		}
		if input.Kind == "morph-rendered-beauty" {
			if err := surface.ValidateMorphRenderedBeautyReport(raw); err != nil {
				return nil, fmt.Errorf(
					"%s Morph rendered beauty validation failed: %w",
					input.Path,
					err,
				)
			}
			var mrb surface.MorphRenderedBeautyReport
			if err := json.Unmarshal(raw, &mrb); err != nil {
				return nil, fmt.Errorf(
					"%s Morph rendered beauty decode failed: %w",
					input.Path,
					err,
				)
			}
			report.MorphRenderedBeauty = &mrb
			report.Source = mrb.MorphEvidence.Source
			report.Target = mrb.Target
		}
		reports = append(reports, report)
	}
	return reports, nil
}

func buildInspectorReport(
	reports []runtimeReport,
	outPath string,
	htmlPath string,
) (surface.SurfaceInspectorReport, error) {
	byKind := map[string]runtimeReport{}
	for _, report := range reports {
		if _, exists := byKind[report.Input.Kind]; !exists {
			byKind[report.Input.Kind] = report
		}
	}
	for _, required := range []string{"block", "morph", "app-model", "accessibility"} {
		if _, ok := byKind[required]; !ok {
			return surface.SurfaceInspectorReport{}, fmt.Errorf(
				"runtime reports missing %s",
				required,
			)
		}
	}

	inputReports := make([]surface.SurfaceInspectorInputReport, 0, len(reports))
	sourceLocations := make([]surface.SurfaceInspectorSourceLocation, 0, 4)
	for _, report := range reports {
		pass := report.Status == "pass"
		inputReports = append(inputReports, surface.SurfaceInspectorInputReport{
			Kind:   report.Input.Kind,
			Path:   reportPathForBundle(report.Input.Path),
			Schema: report.Schema,
			Source: report.Source,
			Target: report.Target,
			Pass:   pass,
		})
	}
	for _, kind := range []string{"block", "morph", "accessibility", "app-model"} {
		report := byKind[kind]
		sourceLocations = append(sourceLocations, surface.SurfaceInspectorSourceLocation{
			Kind:   kind,
			Path:   report.Source,
			Line:   1,
			Column: 1,
		})
	}
	var morphToPixels *surface.MorphToPixelsChainReport
	if report, ok := byKind["morph-rendered-beauty"]; ok {
		chain := surface.MorphToPixelsChainFromRenderedBeauty(
			reportPathForBundle(report.Input.Path),
			*report.MorphRenderedBeauty,
		)
		morphToPixels = &chain
		sourceLocations = append(sourceLocations, surface.SurfaceInspectorSourceLocation{
			Kind:   "morph-rendered-beauty",
			Path:   chain.Source,
			Line:   1,
			Column: 1,
		})
	}
	sort.Slice(inputReports, func(i, j int) bool {
		return inputReports[i].Kind < inputReports[j].Kind
	})

	sections := surface.SurfaceInspectorSections{
		BlockTree: section(
			"block_graph.nodes + component_tree.nodes",
			countArrays(reports, "block_graph.nodes", "component_tree.nodes"),
		),
		MorphTokens: section(
			"morph.token_graph.tokens",
			countArrays(reports, "morph.token_graph.tokens"),
		),
		Layout: section(
			"layout_passes + layout_constraints + component_tree.layout_passes",
			countArrays(
				reports,
				"layout_passes",
				"layout_constraints",
				"component_tree.layout_passes",
			),
		),
		Paint: section(
			"paint_commands + paint_layers",
			countArrays(reports, "paint_commands", "paint_layers"),
		),
		Accessibility: section(
			"block_accessibility_tree.nodes + accessibility_tree.nodes",
			countArrays(reports, "block_accessibility_tree.nodes", "accessibility_tree.nodes"),
		),
		EventRoutes: section(
			"block_event_routes + events + app_model.event_bindings",
			countArrays(reports, "block_event_routes", "events", "app_model.event_bindings"),
		),
		Focus: section(
			"block_focus_transitions + app_model.focus_scopes + block_graph.focus_order",
			countArrays(
				reports,
				"block_focus_transitions",
				"app_model.focus_scopes",
				"block_graph.focus_order",
			),
		),
		PerfCounters: section(
			"surface_performance_budget + renderer.cache_stats + block_system.memory_budget",
			countObjects(
				reports,
				"surface_performance_budget",
				"renderer.cache_stats",
				"block_system.memory_budget",
			),
		),
	}
	if morphToPixels != nil {
		sections.RecipeExpansions = section(
			"morph_to_pixels.recipe_expansion_count",
			morphToPixels.RecipeExpansionCount,
		)
		sections.BlockSceneNodes = section(
			"morph_to_pixels.block_scene_node_count",
			morphToPixels.BlockSceneNodeCount,
		)
		sections.RenderCommands = section(
			"morph_to_pixels.render_command_count",
			morphToPixels.RenderCommandCount,
		)
		sections.FrameArtifacts = section(
			"morph_to_pixels.frame_artifact",
			presentCount(morphToPixels.FrameArtifact),
		)
		sections.GoldenDiff = section(
			"morph_to_pixels.golden_artifact + diff metrics",
			presentCount(morphToPixels.GoldenArtifact),
		)
	}

	return surface.SurfaceInspectorReport{
		Schema:          surface.InspectorSchemaV1,
		Model:           "surface-inspector-v1",
		ReleaseScope:    surface.ReleaseScopeSurfaceV1LinuxWeb,
		Producer:        "tools/cmd/surface-inspector",
		Source:          byKind["block"].Source,
		Target:          "headless",
		Mode:            "static-tool-report",
		InputReports:    inputReports,
		SourceLocations: sourceLocations,
		Sections:        sections,
		MorphToPixels:   morphToPixels,
		StaticArtifacts: surface.SurfaceInspectorStaticArtifacts{
			JSON:           reportPathForBundle(outPath),
			HTML:           reportPathForBundle(htmlPath),
			HTMLToolReport: strings.TrimSpace(htmlPath) != "",
		},
		HiddenState: surface.SurfaceInspectorHiddenState{
			Scanned:  true,
			Findings: []surface.SurfaceInspectorHiddenStateFinding{},
		},
		NegativeGuards: surface.SurfaceInspectorNegativeGuards{
			NoDOMRuntimeDependency:      true,
			NoBrowserDevtoolsDependency: true,
			NoReactDevtoolsDependency:   true,
			StaticHTMLToolReportOnly:    true,
			NoHiddenState:               true,
		},
		Pass: true,
	}, nil
}

func presentCount(value string) int {
	if strings.TrimSpace(value) == "" {
		return 0
	}
	return 1
}

func section(source string, count int) surface.SurfaceInspectorSection {
	return surface.SurfaceInspectorSection{
		Present: count > 0,
		Count:   count,
		Source:  source,
	}
}

func renderInspectorHTML(report surface.SurfaceInspectorReport) string {
	var b strings.Builder
	b.WriteString(
		"<!doctype html><html><head><meta charset=\"utf-8\"><title>Surface Inspector</title>",
	)
	b.WriteString(
		("<style>body{font-family:sans-" +
			"serif;margin:24px;color:#172026;background:#fff}table{border-" +
			"collapse:collapse;width:100%;max-width:960px}th,td{border:1px solid " +
			"#ccd3d8;padding:6px 8px;text-align:left}th{background:#eef3f7}</style>"),
	)
	b.WriteString("</head><body><h1>Surface Inspector</h1><p>static tool report for ")
	b.WriteString(html.EscapeString(report.ReleaseScope))
	b.WriteString(
		"</p><table><thead><tr><th>Section</th><th>Count</th><th>Source</th></tr></thead><tbody>",
	)
	for _, row := range []struct {
		Name    string
		Section surface.SurfaceInspectorSection
	}{
		{Name: "block_tree", Section: report.Sections.BlockTree},
		{Name: "morph_tokens", Section: report.Sections.MorphTokens},
		{Name: "layout", Section: report.Sections.Layout},
		{Name: "paint", Section: report.Sections.Paint},
		{Name: "accessibility", Section: report.Sections.Accessibility},
		{Name: "event_routes", Section: report.Sections.EventRoutes},
		{Name: "focus", Section: report.Sections.Focus},
		{Name: "perf_counters", Section: report.Sections.PerfCounters},
		{Name: "recipe_expansions", Section: report.Sections.RecipeExpansions},
		{Name: "block_scene_nodes", Section: report.Sections.BlockSceneNodes},
		{Name: "render_commands", Section: report.Sections.RenderCommands},
		{Name: "frame_artifacts", Section: report.Sections.FrameArtifacts},
		{Name: "golden_diff", Section: report.Sections.GoldenDiff},
	} {
		if !row.Section.Present {
			continue
		}
		b.WriteString("<tr><td>")
		b.WriteString(html.EscapeString(row.Name))
		b.WriteString("</td><td>")
		b.WriteString(fmt.Sprintf("%d", row.Section.Count))
		b.WriteString("</td><td>")
		b.WriteString(html.EscapeString(row.Section.Source))
		b.WriteString("</td></tr>")
	}
	b.WriteString("</tbody></table>")
	if report.MorphToPixels != nil {
		b.WriteString("<h2>Morph to pixels</h2><table><tbody>")
		for _, row := range []struct {
			Name  string
			Value string
		}{
			{Name: "chain_id", Value: report.MorphToPixels.ChainID},
			{Name: "source", Value: report.MorphToPixels.Source},
			{Name: "block_scene_hash", Value: report.MorphToPixels.BlockSceneHash},
			{Name: "render_command_stream_hash", Value: report.MorphToPixels.RenderCommandStreamHash},
			{Name: "frame_artifact", Value: report.MorphToPixels.FrameArtifact},
			{Name: "golden_artifact", Value: report.MorphToPixels.GoldenArtifact},
		} {
			b.WriteString("<tr><td>")
			b.WriteString(html.EscapeString(row.Name))
			b.WriteString("</td><td>")
			b.WriteString(html.EscapeString(row.Value))
			b.WriteString("</td></tr>")
		}
		b.WriteString("</tbody></table>")
	}
	b.WriteString("</body></html>\n")
	return b.String()
}

func rejectHiddenState(path string, raw map[string]any) error {
	if boolField(raw, "hidden_state") {
		return fmt.Errorf("%s contains hidden_state=true", path)
	}
	if len(arrayField(raw, "hidden_state_findings")) > 0 {
		return fmt.Errorf("%s contains hidden_state_findings", path)
	}
	if hidden, ok := raw["hidden_state"].(map[string]any); ok &&
		len(arrayField(hidden, "findings")) > 0 {
		return fmt.Errorf("%s contains hidden_state.findings", path)
	}
	return nil
}

func countArrays(reports []runtimeReport, paths ...string) int {
	count := 0
	for _, report := range reports {
		for _, path := range paths {
			count += len(arrayAt(report.Raw, path))
		}
	}
	return count
}

func countObjects(reports []runtimeReport, paths ...string) int {
	count := 0
	for _, report := range reports {
		for _, path := range paths {
			if objectAt(report.Raw, path) != nil {
				count++
			}
		}
	}
	return count
}

func arrayAt(raw map[string]any, path string) []any {
	current := any(raw)
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = object[part]
	}
	array, _ := current.([]any)
	return array
}

func objectAt(raw map[string]any, path string) map[string]any {
	current := any(raw)
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = object[part]
	}
	object, _ := current.(map[string]any)
	return object
}

func stringField(raw map[string]any, field string) string {
	value, _ := raw[field].(string)
	return value
}

func boolField(raw map[string]any, field string) bool {
	value, _ := raw[field].(bool)
	return value
}

func arrayField(raw map[string]any, field string) []any {
	value, _ := raw[field].([]any)
	return value
}

func reportPathForBundle(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) {
		return filepath.ToSlash(clean)
	}
	base := filepath.Base(clean)
	parent := filepath.Base(filepath.Dir(clean))
	if parent == "." || parent == string(filepath.Separator) || parent == "" {
		return base
	}
	return filepath.ToSlash(filepath.Join(parent, base))
}
