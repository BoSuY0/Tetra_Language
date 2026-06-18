package main

import (
	"encoding/json"
	"strings"
	"testing"

	"tetra_language/tools/validators/surface"
	"tetra_language/tools/validators/surfaceinspector"
)

func TestSnapshotFromSurfaceReportIncludesDeveloperInspectorViews(t *testing.T) {
	report := inspectorSurfaceReportForTest()
	snapshot := snapshotFromSurfaceReport(report, "reports/surface-prod/P23-inspector/headless/surface-headless-block-system.json")
	raw, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	if err := surfaceinspector.ValidateSnapshot(raw); err != nil {
		t.Fatalf("ValidateSnapshot failed: %v\n%s", err, raw)
	}
	if snapshot.Schema != surfaceinspector.SchemaV1 || snapshot.Level != surfaceinspector.LevelV1 {
		t.Fatalf("snapshot identity = %s/%s, want inspector schema/level", snapshot.Schema, snapshot.Level)
	}
	if len(snapshot.BlockTree) != 2 || len(snapshot.LayoutBoxes) != 2 || len(snapshot.PaintLayers) != 2 {
		t.Fatalf("snapshot views missing tree/layout/paint evidence: %#v", snapshot)
	}
	if len(snapshot.Accessibility) != 1 || snapshot.Accessibility[0].Role != "button" {
		t.Fatalf("accessibility = %#v, want button inspector node", snapshot.Accessibility)
	}
	if !performanceCounterPresent(snapshot.PerformanceCounters, "memory_budget_bytes") {
		t.Fatalf("performance counters = %#v, want memory_budget_bytes", snapshot.PerformanceCounters)
	}
	if snapshot.BlockTree[1].SourceLocationID == "" || snapshot.LayoutBoxes[1].SourceLocationID == "" || snapshot.Events[0].SourceLocationID == "" {
		t.Fatalf("snapshot missing source locations: %#v", snapshot)
	}
}

func TestSnapshotFromSurfaceReportSynthesizesPaintAndAccessibilityForBasicSurface(t *testing.T) {
	report := inspectorSurfaceReportForTest()
	report.PaintLayers = nil
	report.PaintCommands = nil
	report.BlockAccessibilityTree = nil

	snapshot := snapshotFromSurfaceReport(report, "reports/surface-prod/P23-inspector/headless/surface-basic.json")
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	if err := surfaceinspector.ValidateSnapshot(raw); err != nil {
		t.Fatalf("ValidateSnapshot failed for synthesized views: %v\n%s", err, raw)
	}
	if len(snapshot.PaintLayers) == 0 || !strings.Contains(snapshot.PaintLayers[0].Checksum, "sha256:") {
		t.Fatalf("paint layers = %#v, want synthesized checksum", snapshot.PaintLayers)
	}
	if len(snapshot.Accessibility) == 0 || snapshot.Accessibility[0].Name == "" {
		t.Fatalf("accessibility = %#v, want synthesized accessibility view", snapshot.Accessibility)
	}
}

func performanceCounterPresent(counters []surfaceinspector.PerformanceCounter, name string) bool {
	for _, counter := range counters {
		if counter.Name == name {
			return true
		}
	}
	return false
}

func inspectorSurfaceReportForTest() surface.Report {
	return surface.Report{
		Schema:        surface.SchemaV1,
		Status:        "pass",
		Target:        "headless",
		Host:          "linux-x64",
		Runtime:       "surface-headless",
		SurfaceSchema: "tetra.surface.v1",
		Source:        "examples/surface_block_system.tetra",
		Components: []surface.ComponentReport{
			{ID: "BlockSystemApp", Type: "examples.surface_block_system.BlockSystemApp", Bounds: surface.RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "accessibility"}, State: map[string]string{"accessibility_role": "region"}},
			{ID: "SubmitBlock", Type: "lib.core.block.Button", Parent: "BlockSystemApp", Bounds: surface.RectReport{X: 24, Y: 32, W: 128, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "accessibility"}, State: map[string]string{"accessibility_role": "button", "label": "Submit", "focused": "true"}},
		},
		BlockGraph: &surface.BlockGraphReport{
			Schema:             "tetra.surface.block-graph.v1",
			APILevel:           "block-graph-api-v1",
			Source:             "examples/surface_block_system.tetra",
			RootID:             1,
			NodeCount:          2,
			LayoutOrder:        []int{1, 2},
			DrawOrder:          []int{1, 2},
			FocusOrder:         []int{2},
			AccessibilityOrder: []int{2},
			Nodes: []surface.BlockGraphNodeReport{
				{ID: 1, Name: "BlockSystemApp", ParentID: 0, ChildIndex: 0, FirstChild: 2, ChildCount: 1, AccessibilityRole: "region", Bounds: surface.RectReport{X: 0, Y: 0, W: 320, H: 200}},
				{ID: 2, Name: "SubmitBlock", ParentID: 1, ChildIndex: 0, FirstChild: 0, ChildCount: 0, Focusable: true, AccessibilityRole: "button", Bounds: surface.RectReport{X: 24, Y: 32, W: 128, H: 44}},
			},
		},
		PaintLayers: []surface.PaintLayerReport{
			{ID: "layer-root", BlockID: 1, Kind: "fill", Color: "#101820"},
			{ID: "layer-submit", BlockID: 2, Kind: "border", Color: "#3bc3ff"},
		},
		PaintCommands: []surface.PaintCommandReport{
			{Order: 1, Command: "fill", LayerID: "layer-root", BlockID: 1, Rect: surface.RectReport{X: 0, Y: 0, W: 320, H: 200}, Quality: "deterministic", Checksum: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
			{Order: 2, Command: "border", LayerID: "layer-submit", BlockID: 2, Rect: surface.RectReport{X: 24, Y: 32, W: 128, H: 44}, Quality: "deterministic", Checksum: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		},
		Events: []surface.EventReport{
			{Order: 1, Kind: "mouse_up", TargetComponent: "SubmitBlock", DispatchPath: []string{"BlockSystemApp", "SubmitBlock"}, Handled: true, Pass: true, X: 48, Y: 96, BeforeState: map[string]string{"SubmitBlock.focused": "false"}, AfterState: map[string]string{"SubmitBlock.focused": "true"}},
		},
		Frames: []surface.FrameReport{
			{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", Presented: true},
			{Order: 2, Width: 320, Height: 200, Stride: 1280, Checksum: "2222222222222222222222222222222222222222222222222222222222222222", Presented: true},
		},
		BlockAccessibilityTree: &surface.BlockAccessibilityTreeReport{
			Schema:             "tetra.surface.block-accessibility-tree.v1",
			AccessibilityLevel: "block-accessibility-metadata-v1",
			Source:             "examples/surface_block_system.tetra",
			QualityLevel:       "deterministic-block-accessibility-v1",
			NodeCount:          1,
			FocusableCount:     1,
			FocusOrder:         []int{2},
			ReadingOrder:       []int{2},
			Nodes: []surface.BlockAccessibilityNodeReport{
				{ID: 1, BlockID: 2, ParentBlockID: 1, Name: "Submit", Role: "button", Bounds: surface.RectReport{X: 24, Y: 32, W: 128, H: 44}, Visible: true, Enabled: true, Focusable: true, Focused: true, Actions: []string{"press"}, FocusIndex: 1, ReadingIndex: 1},
			},
		},
		BlockSystem: &surface.BlockSystemReport{
			Schema:       "tetra.surface.block-system.v1",
			QualityLevel: "deterministic-headless-block-system-v1",
			Source:       "examples/surface_block_system.tetra",
			Renderer:     "software-rgba-headless",
			FrameCount:   2,
			MemoryBudget: &surface.BlockMemoryBudgetReport{
				Schema:                   "tetra.surface.block-memory-budget.v1",
				Scope:                    "surface-block-system-local-budget-v1",
				BlockCount:               2,
				PeakFramebufferBytes:     256000,
				FramebufferBudgetBytes:   4194304,
				TotalCacheUsedBytes:      512,
				TotalCacheBudgetBytes:    196608,
				EstimatedAllocationBytes: 1024,
				BoundedCaches:            true,
				UnboundedCacheRejected:   true,
				PerformanceClaim:         "local deterministic budget evidence only",
			},
		},
	}
}
