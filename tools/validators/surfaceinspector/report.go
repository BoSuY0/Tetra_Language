package surfaceinspector

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"tetra_language/tools/validators/surface"
)

const (
	SchemaV1                      = "tetra.surface.inspector-snapshot.v1"
	LevelV1                       = "surface-inspector-json-mvp-v1"
	ReleaseScopeSurfaceV1LinuxWeb = "surface-v1-linux-web"
)

type Snapshot struct {
	Schema              string               `json:"schema"`
	Level               string               `json:"level"`
	Source              string               `json:"source"`
	Target              string               `json:"target"`
	Runtime             string               `json:"runtime"`
	ReleaseScope        string               `json:"release_scope"`
	GeneratedFrom       string               `json:"generated_from"`
	Summary             Summary              `json:"summary"`
	BlockTree           []BlockNode          `json:"block_tree"`
	MorphResolution     MorphResolution      `json:"morph_resolution"`
	LayoutBoxes         []LayoutBox          `json:"layout_boxes"`
	PaintLayers         []PaintLayer         `json:"paint_layers"`
	Events              []Event              `json:"events"`
	Focus               Focus                `json:"focus"`
	Accessibility       []AccessibilityNode  `json:"accessibility"`
	PerformanceCounters []PerformanceCounter `json:"performance_counters"`
	SourceLocations     []SourceLocation     `json:"source_locations"`
	Diagnostics         []Diagnostic         `json:"diagnostics,omitempty"`
	NegativeGuards      NegativeGuards       `json:"negative_guards"`
	NonClaims           []string             `json:"nonclaims"`
}

type Summary struct {
	ComponentCount          int  `json:"component_count"`
	LayoutBoxCount          int  `json:"layout_box_count"`
	PaintLayerCount         int  `json:"paint_layer_count"`
	EventCount              int  `json:"event_count"`
	FocusCount              int  `json:"focus_count"`
	AccessibilityNodeCount  int  `json:"accessibility_node_count"`
	PerformanceCounterCount int  `json:"performance_counter_count"`
	SourceLocationCount     int  `json:"source_location_count"`
	DocsOnly                bool `json:"docs_only"`
}

type BlockNode struct {
	ID               string   `json:"id"`
	Type             string   `json:"type"`
	Parent           string   `json:"parent,omitempty"`
	Path             []string `json:"path"`
	Bounds           Rect     `json:"bounds"`
	SourceLocationID string   `json:"source_location_id"`
	LayoutBoxID      string   `json:"layout_box_id"`
}

type Rect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type MorphResolution struct {
	Present              bool   `json:"present"`
	Schema               string `json:"schema"`
	Capsule              string `json:"capsule"`
	TokenCount           int    `json:"token_count"`
	MaterialCount        int    `json:"material_count"`
	RecipeCount          int    `json:"recipe_count"`
	MotionPresetCount    int    `json:"motion_preset_count"`
	SourceLocationID     string `json:"source_location_id"`
	ResolutionDiagnostic string `json:"resolution_diagnostic"`
}

type LayoutBox struct {
	ID               string `json:"id"`
	BlockID          string `json:"block_id"`
	Rect             Rect   `json:"rect"`
	Mode             string `json:"mode"`
	SourceLocationID string `json:"source_location_id"`
}

type PaintLayer struct {
	ID               string `json:"id"`
	BlockID          string `json:"block_id"`
	Kind             string `json:"kind"`
	Checksum         string `json:"checksum"`
	SourceLocationID string `json:"source_location_id"`
}

type Event struct {
	Order            int    `json:"order"`
	Kind             string `json:"kind"`
	TargetComponent  string `json:"target_component"`
	Handled          bool   `json:"handled"`
	SourceLocationID string `json:"source_location_id"`
}

type Focus struct {
	Order            []string `json:"order"`
	FocusedID        string   `json:"focused_id"`
	SourceLocationID string   `json:"source_location_id"`
}

type AccessibilityNode struct {
	ID               string `json:"id"`
	Role             string `json:"role"`
	Name             string `json:"name"`
	SourceLocationID string `json:"source_location_id"`
}

type PerformanceCounter struct {
	Name             string `json:"name"`
	Value            int    `json:"value"`
	Unit             string `json:"unit"`
	SourceLocationID string `json:"source_location_id"`
}

type SourceLocation struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Kind   string `json:"kind"`
}

type Diagnostic struct {
	Code             string `json:"code"`
	Message          string `json:"message"`
	SourceLocationID string `json:"source_location_id"`
}

type NegativeGuards struct {
	DocsOnlyTreeRejected                  bool `json:"docs_only_tree_rejected"`
	MissingSourceLocationsRejected        bool `json:"missing_source_locations_rejected"`
	MissingLayoutBoxesRejected            bool `json:"missing_layout_boxes_rejected"`
	MissingAccessibilityViewRejected      bool `json:"missing_accessibility_view_rejected"`
	MissingPerformanceCountersRejected    bool `json:"missing_performance_counters_rejected"`
	MissingTargetRuntimeEvidenceRejected  bool `json:"missing_target_runtime_evidence_rejected"`
	MissingEventFocusInspectionRejected   bool `json:"missing_event_focus_inspection_rejected"`
	MissingPaintLayerInspectionRejected   bool `json:"missing_paint_layer_inspection_rejected"`
	MissingMorphResolutionInspectionGuard bool `json:"missing_morph_resolution_inspection_guard"`
}

func SnapshotFromReportRaw(raw []byte, generatedFrom string) (Snapshot, error) {
	if err := surface.ValidateReport(raw); err != nil {
		return Snapshot{}, fmt.Errorf("validate Surface runtime report: %w", err)
	}
	var report surface.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		return Snapshot{}, err
	}
	return SnapshotFromReport(report, generatedFrom), nil
}

func SnapshotFromReport(report surface.Report, generatedFrom string) Snapshot {
	if strings.TrimSpace(generatedFrom) == "" {
		generatedFrom = "inline"
	}
	builder := sourceLocationBuilder{source: fallbackSource(report.Source)}
	blocks, numericNames := blockTreeFromReport(report, &builder)
	layout := layoutBoxesFromBlocks(blocks, &builder)
	paint := paintLayersFromReport(report, blocks, numericNames, &builder)
	events := eventsFromReport(report, &builder)
	focus := focusFromReport(report, blocks, numericNames, &builder)
	accessibility := accessibilityFromReport(report, blocks, numericNames, &builder)
	counters := performanceCountersFromReport(report, blocks, layout, paint, accessibility, &builder)
	morph := morphResolutionFromReport(report, &builder)
	diagnostics := []Diagnostic{{
		Code:             "surface.inspector.snapshot",
		Message:          "developer inspector snapshot captures Surface tree, style resolution, layout boxes, paint layers, events, focus, accessibility, and counters",
		SourceLocationID: builder.add("diagnostic", "snapshot"),
	}}
	snapshot := Snapshot{
		Schema:              SchemaV1,
		Level:               LevelV1,
		Source:              fallbackSource(report.Source),
		Target:              report.Target,
		Runtime:             report.Runtime,
		ReleaseScope:        ReleaseScopeSurfaceV1LinuxWeb,
		GeneratedFrom:       generatedFrom,
		BlockTree:           blocks,
		MorphResolution:     morph,
		LayoutBoxes:         layout,
		PaintLayers:         paint,
		Events:              events,
		Focus:               focus,
		Accessibility:       accessibility,
		PerformanceCounters: counters,
		Diagnostics:         diagnostics,
		NegativeGuards: NegativeGuards{
			DocsOnlyTreeRejected:                  true,
			MissingSourceLocationsRejected:        true,
			MissingLayoutBoxesRejected:            true,
			MissingAccessibilityViewRejected:      true,
			MissingPerformanceCountersRejected:    true,
			MissingTargetRuntimeEvidenceRejected:  true,
			MissingEventFocusInspectionRejected:   true,
			MissingPaintLayerInspectionRejected:   true,
			MissingMorphResolutionInspectionGuard: true,
		},
		NonClaims: []string{
			"interactive devtools UI",
			"perfect source maps",
			"production profiler",
			"browser devtools parity",
		},
	}
	snapshot.SourceLocations = builder.locations
	snapshot.Summary = Summary{
		ComponentCount:          len(snapshot.BlockTree),
		LayoutBoxCount:          len(snapshot.LayoutBoxes),
		PaintLayerCount:         len(snapshot.PaintLayers),
		EventCount:              len(snapshot.Events),
		FocusCount:              len(snapshot.Focus.Order),
		AccessibilityNodeCount:  len(snapshot.Accessibility),
		PerformanceCounterCount: len(snapshot.PerformanceCounters),
		SourceLocationCount:     len(snapshot.SourceLocations),
		DocsOnly:                false,
	}
	return snapshot
}

func ValidateSnapshot(raw []byte) error {
	if bytes.Contains(bytes.ToLower(raw), []byte("dom ui")) {
		return errors.New("snapshot must not claim DOM UI inspection")
	}
	var snapshot Snapshot
	if err := decodeStrict(raw, &snapshot); err != nil {
		return err
	}
	var issues []string
	if snapshot.Schema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", snapshot.Schema, SchemaV1))
	}
	if snapshot.Level != LevelV1 {
		issues = append(issues, fmt.Sprintf("level is %q, want %q", snapshot.Level, LevelV1))
	}
	if strings.TrimSpace(snapshot.Source) == "" {
		issues = append(issues, "source is required")
	}
	if snapshot.Target != "headless" && snapshot.Target != "linux-x64" && snapshot.Target != "wasm32-web" {
		issues = append(issues, fmt.Sprintf("target is %q, want headless, linux-x64, or wasm32-web", snapshot.Target))
	}
	if snapshot.Runtime != "surface-headless" && snapshot.Runtime != "surface-linux-x64" && snapshot.Runtime != "surface-wasm32-web" {
		issues = append(issues, fmt.Sprintf("runtime is %q, want surface-headless, surface-linux-x64, or surface-wasm32-web", snapshot.Runtime))
	}
	if snapshot.ReleaseScope != ReleaseScopeSurfaceV1LinuxWeb {
		issues = append(issues, fmt.Sprintf("release_scope is %q, want %q", snapshot.ReleaseScope, ReleaseScopeSurfaceV1LinuxWeb))
	}
	if strings.TrimSpace(snapshot.GeneratedFrom) == "" {
		issues = append(issues, "generated_from is required")
	}
	if snapshot.Summary.DocsOnly || !snapshot.NegativeGuards.DocsOnlyTreeRejected {
		issues = append(issues, "docs-only tree snapshots must be rejected")
	}
	issues = append(issues, validateInspectorCounts(snapshot)...)
	locations, locationIssues := validateSourceLocations(snapshot.SourceLocations)
	issues = append(issues, locationIssues...)
	issues = append(issues, validateInspectorViews(snapshot, locations)...)
	issues = append(issues, validateInspectorNegativeGuards(snapshot.NegativeGuards)...)
	issues = append(issues, validateInspectorNonClaims(snapshot.NonClaims)...)
	if len(issues) > 0 {
		sort.Strings(issues)
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

type sourceLocationBuilder struct {
	source    string
	nextLine  int
	seen      map[string]int
	locations []SourceLocation
}

func (b *sourceLocationBuilder) add(kind string, ref string) string {
	if b.nextLine == 0 {
		b.nextLine = 1
	}
	if b.seen == nil {
		b.seen = map[string]int{}
	}
	slug := slugify(kind + "-" + ref)
	b.seen[slug]++
	id := "src-" + slug
	if b.seen[slug] > 1 {
		id = fmt.Sprintf("%s-%d", id, b.seen[slug])
	}
	b.locations = append(b.locations, SourceLocation{
		ID:     id,
		Path:   b.source,
		Line:   b.nextLine,
		Column: 1,
		Kind:   kind,
	})
	b.nextLine++
	return id
}

func blockTreeFromReport(report surface.Report, builder *sourceLocationBuilder) ([]BlockNode, map[int]string) {
	numericNames := map[int]string{}
	for i, component := range report.Components {
		numericNames[i+1] = component.ID
	}
	if report.BlockGraph != nil {
		for _, node := range report.BlockGraph.Nodes {
			numericNames[node.ID] = node.Name
		}
	}
	if len(report.Components) > 0 {
		parentPaths := map[string][]string{}
		blocks := make([]BlockNode, 0, len(report.Components))
		for _, component := range report.Components {
			path := append([]string{}, parentPaths[component.Parent]...)
			if len(path) == 0 && component.Parent != "" {
				path = append(path, component.Parent)
			}
			path = append(path, component.ID)
			parentPaths[component.ID] = path
			blocks = append(blocks, BlockNode{
				ID:               component.ID,
				Type:             fallbackString(component.Type, "surface.component"),
				Parent:           component.Parent,
				Path:             path,
				Bounds:           rectFromSurface(component.Bounds),
				SourceLocationID: builder.add("block", component.ID),
				LayoutBoxID:      "layout-" + component.ID,
			})
		}
		return blocks, numericNames
	}
	if report.BlockGraph != nil && len(report.BlockGraph.Nodes) > 0 {
		parentNames := map[int]string{}
		for _, node := range report.BlockGraph.Nodes {
			parentNames[node.ID] = numericNames[node.ParentID]
		}
		blocks := make([]BlockNode, 0, len(report.BlockGraph.Nodes))
		for _, node := range report.BlockGraph.Nodes {
			parent := parentNames[node.ID]
			path := []string{node.Name}
			if parent != "" {
				path = []string{parent, node.Name}
			}
			blocks = append(blocks, BlockNode{
				ID:               node.Name,
				Type:             fallbackString(node.AccessibilityRole, "block"),
				Parent:           parent,
				Path:             path,
				Bounds:           rectFromSurface(node.Bounds),
				SourceLocationID: builder.add("block", node.Name),
				LayoutBoxID:      "layout-" + node.Name,
			})
		}
		return blocks, numericNames
	}
	return nil, numericNames
}

func layoutBoxesFromBlocks(blocks []BlockNode, builder *sourceLocationBuilder) []LayoutBox {
	boxes := make([]LayoutBox, 0, len(blocks))
	for _, block := range blocks {
		mode := "block"
		if block.Parent == "" {
			mode = "root"
		}
		boxes = append(boxes, LayoutBox{
			ID:               block.LayoutBoxID,
			BlockID:          block.ID,
			Rect:             block.Bounds,
			Mode:             mode,
			SourceLocationID: builder.add("layout", block.ID),
		})
	}
	return boxes
}

func paintLayersFromReport(report surface.Report, blocks []BlockNode, numericNames map[int]string, builder *sourceLocationBuilder) []PaintLayer {
	if len(report.PaintCommands) > 0 {
		layers := make([]PaintLayer, 0, len(report.PaintCommands))
		for _, command := range report.PaintCommands {
			id := fallbackString(command.LayerID, fmt.Sprintf("paint-%d", command.Order))
			blockID := fallbackString(numericNames[command.BlockID], fmt.Sprintf("block-%d", command.BlockID))
			layers = append(layers, PaintLayer{
				ID:               id,
				BlockID:          blockID,
				Kind:             fallbackString(command.Command, "paint"),
				Checksum:         normalizeChecksum(command.Checksum, id+blockID),
				SourceLocationID: builder.add("paint", id),
			})
		}
		return layers
	}
	if len(report.PaintLayers) > 0 {
		layers := make([]PaintLayer, 0, len(report.PaintLayers))
		for _, layer := range report.PaintLayers {
			blockID := fallbackString(numericNames[layer.BlockID], fmt.Sprintf("block-%d", layer.BlockID))
			layers = append(layers, PaintLayer{
				ID:               fallbackString(layer.ID, "paint-"+blockID),
				BlockID:          blockID,
				Kind:             fallbackString(layer.Kind, "paint"),
				Checksum:         checksumFor(layer.ID + layer.Kind + blockID + layer.Color),
				SourceLocationID: builder.add("paint", fallbackString(layer.ID, blockID)),
			})
		}
		return layers
	}
	layers := make([]PaintLayer, 0, len(blocks))
	for _, block := range blocks {
		id := "paint-" + block.ID
		layers = append(layers, PaintLayer{
			ID:               id,
			BlockID:          block.ID,
			Kind:             "synthetic-fill",
			Checksum:         checksumFor(id + block.Type),
			SourceLocationID: builder.add("paint", id),
		})
	}
	return layers
}

func eventsFromReport(report surface.Report, builder *sourceLocationBuilder) []Event {
	events := make([]Event, 0, len(report.Events))
	for _, event := range report.Events {
		events = append(events, Event{
			Order:            event.Order,
			Kind:             fallbackString(event.Kind, "event"),
			TargetComponent:  fallbackString(event.TargetComponent, "surface"),
			Handled:          event.Handled,
			SourceLocationID: builder.add("event", fmt.Sprintf("%d-%s", event.Order, event.Kind)),
		})
	}
	return events
}

func focusFromReport(report surface.Report, blocks []BlockNode, numericNames map[int]string, builder *sourceLocationBuilder) Focus {
	order := []string{}
	if report.BlockGraph != nil {
		for _, id := range report.BlockGraph.FocusOrder {
			if name := numericNames[id]; name != "" {
				order = append(order, name)
			}
		}
	}
	if len(order) == 0 {
		for _, component := range report.Components {
			if strings.EqualFold(component.State["focused"], "true") || containsString(component.Abilities, "focus") {
				order = append(order, component.ID)
			}
		}
	}
	if len(order) == 0 && len(blocks) > 0 {
		order = append(order, blocks[0].ID)
	}
	focused := ""
	if len(order) > 0 {
		focused = order[len(order)-1]
	}
	for _, component := range report.Components {
		if strings.EqualFold(component.State["focused"], "true") {
			focused = component.ID
			break
		}
	}
	return Focus{Order: order, FocusedID: focused, SourceLocationID: builder.add("focus", fallbackString(focused, "surface"))}
}

func accessibilityFromReport(report surface.Report, blocks []BlockNode, numericNames map[int]string, builder *sourceLocationBuilder) []AccessibilityNode {
	if report.BlockAccessibilityTree != nil && len(report.BlockAccessibilityTree.Nodes) > 0 {
		nodes := make([]AccessibilityNode, 0, len(report.BlockAccessibilityTree.Nodes))
		for _, node := range report.BlockAccessibilityTree.Nodes {
			blockID := fallbackString(numericNames[node.BlockID], fmt.Sprintf("block-%d", node.BlockID))
			nodes = append(nodes, AccessibilityNode{
				ID:               fmt.Sprintf("a11y-%d", node.ID),
				Role:             fallbackString(node.Role, "group"),
				Name:             fallbackString(node.Name, blockID),
				SourceLocationID: builder.add("accessibility", blockID),
			})
		}
		return nodes
	}
	if report.AccessibilityTree != nil && len(report.AccessibilityTree.Nodes) > 0 {
		nodes := make([]AccessibilityNode, 0, len(report.AccessibilityTree.Nodes))
		for _, node := range report.AccessibilityTree.Nodes {
			nodes = append(nodes, AccessibilityNode{
				ID:               fmt.Sprintf("a11y-%d", node.ID),
				Role:             fallbackString(node.Role, "group"),
				Name:             fallbackString(node.Name, fmt.Sprintf("component-%d", node.ComponentID)),
				SourceLocationID: builder.add("accessibility", fmt.Sprintf("%d", node.ID)),
			})
		}
		return nodes
	}
	nodes := make([]AccessibilityNode, 0, len(blocks))
	for _, block := range blocks {
		role := "group"
		name := block.ID
		for _, component := range report.Components {
			if component.ID == block.ID {
				role = fallbackString(component.State["accessibility_role"], role)
				name = fallbackString(component.State["label"], component.ID)
				break
			}
		}
		nodes = append(nodes, AccessibilityNode{
			ID:               "a11y-" + block.ID,
			Role:             role,
			Name:             name,
			SourceLocationID: builder.add("accessibility", block.ID),
		})
	}
	return nodes
}

func performanceCountersFromReport(report surface.Report, blocks []BlockNode, layout []LayoutBox, paint []PaintLayer, accessibility []AccessibilityNode, builder *sourceLocationBuilder) []PerformanceCounter {
	counters := []PerformanceCounter{
		{Name: "component_count", Value: len(blocks), Unit: "count", SourceLocationID: builder.add("performance", "component-count")},
		{Name: "layout_box_count", Value: len(layout), Unit: "count", SourceLocationID: builder.add("performance", "layout-box-count")},
		{Name: "paint_layer_count", Value: len(paint), Unit: "count", SourceLocationID: builder.add("performance", "paint-layer-count")},
		{Name: "event_count", Value: len(report.Events), Unit: "count", SourceLocationID: builder.add("performance", "event-count")},
		{Name: "frame_count", Value: len(report.Frames), Unit: "count", SourceLocationID: builder.add("performance", "frame-count")},
		{Name: "accessibility_node_count", Value: len(accessibility), Unit: "count", SourceLocationID: builder.add("performance", "accessibility-node-count")},
	}
	if report.BlockSystem != nil && report.BlockSystem.MemoryBudget != nil {
		budget := report.BlockSystem.MemoryBudget
		value := budget.FramebufferBudgetBytes + budget.TotalCacheBudgetBytes
		if value == 0 {
			value = budget.EstimatedAllocationBytes
		}
		counters = append(counters, PerformanceCounter{Name: "memory_budget_bytes", Value: value, Unit: "bytes", SourceLocationID: builder.add("performance", "memory-budget")})
	}
	return counters
}

func morphResolutionFromReport(report surface.Report, builder *sourceLocationBuilder) MorphResolution {
	locationID := builder.add("morph", "resolution")
	if report.Morph == nil {
		return MorphResolution{
			Present:              false,
			Schema:               "block-only",
			Capsule:              "none",
			SourceLocationID:     locationID,
			ResolutionDiagnostic: "no Morph capsule in runtime report; inspector recorded Block-only style boundary",
		}
	}
	schema := report.Morph.Schema
	if report.Morph.TokenGraph != nil && report.Morph.TokenGraph.Schema != "" {
		schema = report.Morph.TokenGraph.Schema
	}
	return MorphResolution{
		Present:              true,
		Schema:               fallbackString(schema, "tetra.surface.morph.v1"),
		Capsule:              fallbackString(report.Morph.Capsule.Namespace, report.Morph.Module),
		TokenCount:           morphTokenCount(report.Morph),
		MaterialCount:        len(report.Morph.Materials),
		RecipeCount:          len(report.Morph.Recipes),
		MotionPresetCount:    len(report.Morph.MotionPresets),
		SourceLocationID:     locationID,
		ResolutionDiagnostic: "Morph style graph resolved into Surface inspector snapshot",
	}
}

func validateInspectorCounts(snapshot Snapshot) []string {
	var issues []string
	if len(snapshot.BlockTree) == 0 {
		issues = append(issues, "block_tree is required")
	}
	if len(snapshot.LayoutBoxes) == 0 {
		issues = append(issues, "layout_boxes are required")
	}
	if len(snapshot.PaintLayers) == 0 {
		issues = append(issues, "paint_layers are required")
	}
	if len(snapshot.Events) == 0 {
		issues = append(issues, "events are required")
	}
	if len(snapshot.Focus.Order) == 0 {
		issues = append(issues, "focus order is required")
	}
	if len(snapshot.Accessibility) == 0 {
		issues = append(issues, "accessibility view is required")
	}
	if len(snapshot.PerformanceCounters) == 0 {
		issues = append(issues, "performance_counters are required")
	}
	if len(snapshot.SourceLocations) == 0 {
		issues = append(issues, "source_locations are required")
	}
	checks := []struct {
		name string
		got  int
		want int
	}{
		{"summary.component_count", snapshot.Summary.ComponentCount, len(snapshot.BlockTree)},
		{"summary.layout_box_count", snapshot.Summary.LayoutBoxCount, len(snapshot.LayoutBoxes)},
		{"summary.paint_layer_count", snapshot.Summary.PaintLayerCount, len(snapshot.PaintLayers)},
		{"summary.event_count", snapshot.Summary.EventCount, len(snapshot.Events)},
		{"summary.focus_count", snapshot.Summary.FocusCount, len(snapshot.Focus.Order)},
		{"summary.accessibility_node_count", snapshot.Summary.AccessibilityNodeCount, len(snapshot.Accessibility)},
		{"summary.performance_counter_count", snapshot.Summary.PerformanceCounterCount, len(snapshot.PerformanceCounters)},
		{"summary.source_location_count", snapshot.Summary.SourceLocationCount, len(snapshot.SourceLocations)},
	}
	for _, check := range checks {
		if check.got != check.want {
			issues = append(issues, fmt.Sprintf("%s is %d, want %d", check.name, check.got, check.want))
		}
	}
	return issues
}

func validateSourceLocations(locations []SourceLocation) (map[string]SourceLocation, []string) {
	var issues []string
	seen := map[string]SourceLocation{}
	for i, location := range locations {
		if strings.TrimSpace(location.ID) == "" {
			issues = append(issues, fmt.Sprintf("source_locations[%d].id is required", i))
			continue
		}
		if _, ok := seen[location.ID]; ok {
			issues = append(issues, fmt.Sprintf("source_locations id %q is duplicated", location.ID))
		}
		if strings.TrimSpace(location.Path) == "" {
			issues = append(issues, fmt.Sprintf("source_locations[%s].path is required", location.ID))
		}
		if location.Line <= 0 || location.Column <= 0 {
			issues = append(issues, fmt.Sprintf("source_locations[%s] must have positive line and column", location.ID))
		}
		if strings.TrimSpace(location.Kind) == "" {
			issues = append(issues, fmt.Sprintf("source_locations[%s].kind is required", location.ID))
		}
		seen[location.ID] = location
	}
	return seen, issues
}

func validateInspectorViews(snapshot Snapshot, locations map[string]SourceLocation) []string {
	var issues []string
	blockIDs := map[string]bool{}
	layoutIDs := map[string]bool{}
	for _, box := range snapshot.LayoutBoxes {
		layoutIDs[box.ID] = true
	}
	for _, block := range snapshot.BlockTree {
		blockIDs[block.ID] = true
		if strings.TrimSpace(block.ID) == "" || strings.TrimSpace(block.Type) == "" {
			issues = append(issues, "block_tree nodes require id and type")
		}
		if len(block.Path) == 0 {
			issues = append(issues, fmt.Sprintf("block_tree[%s].path is required", block.ID))
		}
		if !locationExists(locations, block.SourceLocationID) {
			issues = append(issues, fmt.Sprintf("block_tree[%s] source_location_id is missing or unknown", block.ID))
		}
		if !layoutIDs[block.LayoutBoxID] {
			issues = append(issues, fmt.Sprintf("block_tree[%s] layout_box_id %q is missing", block.ID, block.LayoutBoxID))
		}
	}
	if !locationExists(locations, snapshot.MorphResolution.SourceLocationID) {
		issues = append(issues, "morph_resolution source_location_id is missing or unknown")
	}
	if snapshot.MorphResolution.Present && (strings.TrimSpace(snapshot.MorphResolution.Schema) == "" || strings.TrimSpace(snapshot.MorphResolution.Capsule) == "") {
		issues = append(issues, "morph_resolution present snapshots require schema and capsule")
	}
	if !snapshot.MorphResolution.Present && strings.TrimSpace(snapshot.MorphResolution.ResolutionDiagnostic) == "" {
		issues = append(issues, "morph_resolution absent snapshots require a diagnostic")
	}
	for _, box := range snapshot.LayoutBoxes {
		if strings.TrimSpace(box.ID) == "" || !blockIDs[box.BlockID] || strings.TrimSpace(box.Mode) == "" {
			issues = append(issues, fmt.Sprintf("layout_boxes[%s] must reference a known block and mode", box.ID))
		}
		if !locationExists(locations, box.SourceLocationID) {
			issues = append(issues, fmt.Sprintf("layout_boxes[%s] source_location_id is missing or unknown", box.ID))
		}
	}
	for _, layer := range snapshot.PaintLayers {
		if strings.TrimSpace(layer.ID) == "" || !blockIDs[layer.BlockID] || strings.TrimSpace(layer.Kind) == "" {
			issues = append(issues, fmt.Sprintf("paint_layers[%s] must reference a known block and kind", layer.ID))
		}
		if !strings.HasPrefix(layer.Checksum, "sha256:") {
			issues = append(issues, fmt.Sprintf("paint_layers[%s].checksum must be sha256-prefixed", layer.ID))
		}
		if !locationExists(locations, layer.SourceLocationID) {
			issues = append(issues, fmt.Sprintf("paint_layers[%s] source_location_id is missing or unknown", layer.ID))
		}
	}
	for _, event := range snapshot.Events {
		if event.Order <= 0 || strings.TrimSpace(event.Kind) == "" || !blockIDs[event.TargetComponent] {
			issues = append(issues, fmt.Sprintf("events[%d] must reference a known target component and kind", event.Order))
		}
		if !locationExists(locations, event.SourceLocationID) {
			issues = append(issues, fmt.Sprintf("events[%d] source_location_id is missing or unknown", event.Order))
		}
	}
	if strings.TrimSpace(snapshot.Focus.FocusedID) == "" || !blockIDs[snapshot.Focus.FocusedID] {
		issues = append(issues, "focus.focused_id must reference a known block")
	}
	if !locationExists(locations, snapshot.Focus.SourceLocationID) {
		issues = append(issues, "focus source_location_id is missing or unknown")
	}
	for _, node := range snapshot.Accessibility {
		if strings.TrimSpace(node.ID) == "" || strings.TrimSpace(node.Role) == "" || strings.TrimSpace(node.Name) == "" {
			issues = append(issues, fmt.Sprintf("accessibility[%s] requires id, role, and name", node.ID))
		}
		if !locationExists(locations, node.SourceLocationID) {
			issues = append(issues, fmt.Sprintf("accessibility[%s] source_location_id is missing or unknown", node.ID))
		}
	}
	for _, counter := range snapshot.PerformanceCounters {
		if strings.TrimSpace(counter.Name) == "" || strings.TrimSpace(counter.Unit) == "" || counter.Value < 0 {
			issues = append(issues, fmt.Sprintf("performance_counters[%s] requires name, unit, and non-negative value", counter.Name))
		}
		if !locationExists(locations, counter.SourceLocationID) {
			issues = append(issues, fmt.Sprintf("performance_counters[%s] source_location_id is missing or unknown", counter.Name))
		}
	}
	for _, diagnostic := range snapshot.Diagnostics {
		if !locationExists(locations, diagnostic.SourceLocationID) {
			issues = append(issues, fmt.Sprintf("diagnostics[%s] source_location_id is missing or unknown", diagnostic.Code))
		}
	}
	return issues
}

func validateInspectorNegativeGuards(guards NegativeGuards) []string {
	checks := []struct {
		name string
		ok   bool
	}{
		{"docs_only_tree_rejected", guards.DocsOnlyTreeRejected},
		{"missing_source_locations_rejected", guards.MissingSourceLocationsRejected},
		{"missing_layout_boxes_rejected", guards.MissingLayoutBoxesRejected},
		{"missing_accessibility_view_rejected", guards.MissingAccessibilityViewRejected},
		{"missing_performance_counters_rejected", guards.MissingPerformanceCountersRejected},
		{"missing_target_runtime_evidence_rejected", guards.MissingTargetRuntimeEvidenceRejected},
		{"missing_event_focus_inspection_rejected", guards.MissingEventFocusInspectionRejected},
		{"missing_paint_layer_inspection_rejected", guards.MissingPaintLayerInspectionRejected},
		{"missing_morph_resolution_inspection_guard", guards.MissingMorphResolutionInspectionGuard},
	}
	var issues []string
	for _, check := range checks {
		if !check.ok {
			issues = append(issues, "negative guard "+check.name+" must be true")
		}
	}
	return issues
}

func validateInspectorNonClaims(nonClaims []string) []string {
	required := []string{"interactive devtools UI", "perfect source maps", "production profiler", "browser devtools parity"}
	var issues []string
	for _, want := range required {
		if !containsString(nonClaims, want) {
			issues = append(issues, "nonclaims must include "+want)
		}
	}
	return issues
}

func decodeStrict(raw []byte, dst any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("trailing data after JSON document")
	}
	return nil
}

func locationExists(locations map[string]SourceLocation, id string) bool {
	if strings.TrimSpace(id) == "" {
		return false
	}
	_, ok := locations[id]
	return ok
}

func rectFromSurface(rect surface.RectReport) Rect {
	return Rect{X: rect.X, Y: rect.Y, W: rect.W, H: rect.H}
}

func fallbackSource(source string) string {
	return fallbackString(source, "unknown-surface-source.tetra")
}

func fallbackString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func slugify(value string) string {
	value = strings.ToLower(value)
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func checksumFor(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func normalizeChecksum(value string, seed string) string {
	if strings.HasPrefix(value, "sha256:") {
		return value
	}
	if strings.TrimSpace(value) != "" && len(value) == 64 {
		return "sha256:" + value
	}
	return checksumFor(seed)
}

func morphTokenCount(morph *surface.MorphReport) int {
	if morph == nil || morph.TokenGraph == nil {
		return 0
	}
	return len(morph.TokenGraph.Tokens)
}
