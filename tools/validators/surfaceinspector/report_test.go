package surfaceinspector

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateSnapshotAcceptsDeveloperInspectorEvidence(t *testing.T) {
	raw := mustMarshalSnapshotForTest(t, validSnapshotForTest())
	if err := ValidateSnapshot(raw); err != nil {
		t.Fatalf("ValidateSnapshot failed: %v\n%s", err, raw)
	}
}

func TestValidateSnapshotRejectsMissingSourceLocations(t *testing.T) {
	snapshot := validSnapshotForTest()
	snapshot.SourceLocations = nil
	raw := mustMarshalSnapshotForTest(t, snapshot)

	err := ValidateSnapshot(raw)
	if err == nil || !strings.Contains(err.Error(), "source_locations") {
		t.Fatalf("ValidateSnapshot error = %v, want source_locations rejection", err)
	}
}

func TestValidateSnapshotRejectsDocsOnlyTree(t *testing.T) {
	snapshot := validSnapshotForTest()
	snapshot.Summary.DocsOnly = true
	snapshot.NegativeGuards.DocsOnlyTreeRejected = false
	raw := mustMarshalSnapshotForTest(t, snapshot)

	err := ValidateSnapshot(raw)
	if err == nil || !strings.Contains(err.Error(), "docs-only") {
		t.Fatalf("ValidateSnapshot error = %v, want docs-only rejection", err)
	}
}

func TestValidateSnapshotRejectsMissingPerformanceCounters(t *testing.T) {
	snapshot := validSnapshotForTest()
	snapshot.PerformanceCounters = nil
	raw := mustMarshalSnapshotForTest(t, snapshot)

	err := ValidateSnapshot(raw)
	if err == nil || !strings.Contains(err.Error(), "performance_counters") {
		t.Fatalf("ValidateSnapshot error = %v, want performance_counters rejection", err)
	}
}

func mustMarshalSnapshotForTest(t *testing.T, snapshot Snapshot) []byte {
	t.Helper()
	raw, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	return raw
}

func validSnapshotForTest() Snapshot {
	return Snapshot{
		Schema:        SchemaV1,
		Level:         LevelV1,
		Source:        "examples/surface_block_system.tetra",
		Target:        "headless",
		Runtime:       "surface-headless",
		ReleaseScope:  ReleaseScopeSurfaceV1LinuxWeb,
		GeneratedFrom: "reports/surface-prod/P23-inspector/headless/surface-headless-block-system.json",
		Summary: Summary{
			ComponentCount:          2,
			LayoutBoxCount:          2,
			PaintLayerCount:         2,
			EventCount:              1,
			FocusCount:              1,
			AccessibilityNodeCount:  1,
			PerformanceCounterCount: 2,
			SourceLocationCount:     12,
			DocsOnly:                false,
		},
		BlockTree: []BlockNode{
			{ID: "BlockSystemApp", Type: "examples.surface_block_system.BlockSystemApp", Path: []string{"BlockSystemApp"}, Bounds: Rect{X: 0, Y: 0, W: 320, H: 200}, SourceLocationID: "src-block-root", LayoutBoxID: "layout-BlockSystemApp"},
			{ID: "SubmitBlock", Type: "lib.core.block.Button", Parent: "BlockSystemApp", Path: []string{"BlockSystemApp", "SubmitBlock"}, Bounds: Rect{X: 24, Y: 32, W: 128, H: 44}, SourceLocationID: "src-block-submit", LayoutBoxID: "layout-SubmitBlock"},
		},
		MorphResolution: MorphResolution{
			Present:              true,
			Schema:               "tetra.surface.morph.style-graph.v1",
			Capsule:              "examples.surface_morph_command_palette",
			TokenCount:           8,
			MaterialCount:        4,
			RecipeCount:          11,
			MotionPresetCount:    2,
			SourceLocationID:     "src-morph-resolution",
			ResolutionDiagnostic: "typed style graph resolved before Block emission",
		},
		LayoutBoxes: []LayoutBox{
			{ID: "layout-BlockSystemApp", BlockID: "BlockSystemApp", Rect: Rect{X: 0, Y: 0, W: 320, H: 200}, Mode: "root", SourceLocationID: "src-layout-root"},
			{ID: "layout-SubmitBlock", BlockID: "SubmitBlock", Rect: Rect{X: 24, Y: 32, W: 128, H: 44}, Mode: "block", SourceLocationID: "src-layout-submit"},
		},
		PaintLayers: []PaintLayer{
			{ID: "paint-root", BlockID: "BlockSystemApp", Kind: "fill", Checksum: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", SourceLocationID: "src-paint-root"},
			{ID: "paint-submit", BlockID: "SubmitBlock", Kind: "border", Checksum: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", SourceLocationID: "src-paint-submit"},
		},
		Events: []Event{
			{Order: 1, Kind: "mouse_up", TargetComponent: "SubmitBlock", Handled: true, SourceLocationID: "src-event-submit"},
		},
		Focus: Focus{
			Order:            []string{"SubmitBlock"},
			FocusedID:        "SubmitBlock",
			SourceLocationID: "src-focus-submit",
		},
		Accessibility: []AccessibilityNode{
			{ID: "a11y-submit", Role: "button", Name: "Submit", SourceLocationID: "src-a11y-submit"},
		},
		PerformanceCounters: []PerformanceCounter{
			{Name: "component_count", Value: 2, Unit: "count", SourceLocationID: "src-perf-component"},
			{Name: "frame_count", Value: 3, Unit: "count", SourceLocationID: "src-perf-frame"},
		},
		SourceLocations: []SourceLocation{
			{ID: "src-block-root", Path: "examples/surface_block_system.tetra", Line: 1, Column: 1, Kind: "block"},
			{ID: "src-block-submit", Path: "examples/surface_block_system.tetra", Line: 2, Column: 1, Kind: "block"},
			{ID: "src-morph-resolution", Path: "examples/surface_block_system.tetra", Line: 3, Column: 1, Kind: "morph"},
			{ID: "src-layout-root", Path: "examples/surface_block_system.tetra", Line: 4, Column: 1, Kind: "layout"},
			{ID: "src-layout-submit", Path: "examples/surface_block_system.tetra", Line: 5, Column: 1, Kind: "layout"},
			{ID: "src-paint-root", Path: "examples/surface_block_system.tetra", Line: 6, Column: 1, Kind: "paint"},
			{ID: "src-paint-submit", Path: "examples/surface_block_system.tetra", Line: 7, Column: 1, Kind: "paint"},
			{ID: "src-event-submit", Path: "examples/surface_block_system.tetra", Line: 8, Column: 1, Kind: "event"},
			{ID: "src-focus-submit", Path: "examples/surface_block_system.tetra", Line: 9, Column: 1, Kind: "focus"},
			{ID: "src-a11y-submit", Path: "examples/surface_block_system.tetra", Line: 10, Column: 1, Kind: "accessibility"},
			{ID: "src-perf-component", Path: "examples/surface_block_system.tetra", Line: 11, Column: 1, Kind: "performance"},
			{ID: "src-perf-frame", Path: "examples/surface_block_system.tetra", Line: 12, Column: 1, Kind: "performance"},
		},
		Diagnostics: []Diagnostic{
			{Code: "surface.inspector.snapshot", Message: "developer inspector snapshot captures Block tree, Morph style resolution, layout, paint, events, focus, accessibility, and counters", SourceLocationID: "src-block-root"},
		},
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
}
