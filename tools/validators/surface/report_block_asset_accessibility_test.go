package surface

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateReportAcceptsBlockAssetEvidence(t *testing.T) {
	raw := validHeadlessBlockAssetSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportAcceptsBlockAccessibilityEvidence(t *testing.T) {
	raw := validHeadlessBlockAccessibilitySurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportRejectsIncompleteBlockAccessibilityEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "missing name for actionable focusable block",
			mutate: func(report *Report) {
				report.BlockAccessibilityTree.Nodes[1].Name = ""
			},
			want: "name",
		},
		{
			name: "label relationship mismatch",
			mutate: func(report *Report) {
				report.BlockAccessibilityTree.Nodes[1].LabelledBy = "WrongLabel"
			},
			want: "label",
		},
		{
			name: "reading order not from block graph",
			mutate: func(report *Report) {
				report.BlockAccessibilityTree.ReadingOrder = []int{4, 3, 5}
			},
			want: "reading",
		},
		{
			name: "fake screen-reader claim",
			mutate: func(report *Report) {
				report.BlockAccessibilityTree.ScreenReaderEvidence = true
			},
			want: "screen_reader",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockAccessibilitySurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected block accessibility %s evidence to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateReportRejectsIncompleteBlockAssetEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "missing asset hashes",
			mutate: func(report *Report) {
				report.BlockAssetManifest.Assets[1].SHA256 = ""
			},
			want: "sha256",
		},
		{
			name: "missing diagnostic",
			mutate: func(report *Report) {
				report.BlockAssetDiagnostics = nil
			},
			want: "diagnostic",
		},
		{
			name: "unbounded cache",
			mutate: func(report *Report) {
				report.BlockAssetCache.Bounded = false
				report.BlockAssetCache.BudgetBytes = 0
			},
			want: "cache",
		},
		{
			name: "network asset url",
			mutate: func(report *Report) {
				report.BlockAssetManifest.Assets[0].Path = "https://assets.example.test/tetra-ui.woff2"
				report.BlockAssetManifest.Assets[0].Local = false
				report.BlockAssetManifest.RemoteCount = 1
			},
			want: "network",
		},
		{
			name: "missing tint command",
			mutate: func(report *Report) {
				report.BlockAssetRenderCommands = removeBlockAssetRenderCommand(report.BlockAssetRenderCommands, "tint_icon")
			},
			want: "tint",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockAssetSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected block asset %s evidence to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func validHeadlessBlockAssetSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(validHeadlessSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode headless report: %v", err)
	}
	report.Source = "examples/surface_block_assets.tetra"
	report.Processes = []ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_assets.tetra -o /tmp/surface-artifacts/surface-block-assets", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-assets", Ran: true, Pass: true, ExitCode: intPtrForTest(1), ExpectedExitCode: intPtrForTest(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-assets", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
	}
	report.Artifacts = []ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-block-assets", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 49172},
		{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 409},
	}
	report.ArtifactScan = ArtifactScanReport{Root: "/tmp/surface-artifacts", FilesChecked: 2, ForbiddenPaths: nil, Pass: true}
	report.Components = blockAssetComponentsForTest()
	report.BlockAssetQualityLevel = "deterministic-local-block-assets-v1"
	report.BlockAssetNetworkFetchAllowed = false
	report.BlockAssetManifest = blockAssetManifestForTest(report.Source)
	report.BlockAssetCache = blockAssetCacheForTest()
	report.BlockAssetDiagnostics = blockAssetDiagnosticsForTest()
	report.BlockAssetRenderCommands = blockAssetRenderCommandsForTest()
	report.Events = blockAssetEventsForTest()
	report.Frames = []FrameReport{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", Presented: true},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, Checksum: "2222222222222222222222222222222222222222222222222222222222222222", Presented: true},
	}
	report.StateTransitions = []StateTransitionReport{
		{Order: 1, Component: "IconBlock", Field: "tint", Before: "#ffffffff", After: "#60aef4ff", Cause: "asset tint"},
		{Order: 2, Component: "ImageBlock", Field: "scale", Before: "1x", After: "2x", Cause: "asset scale"},
		{Order: 3, Component: "MissingAssetBlock", Field: "fallback", Before: "missing", After: "fallback-raster", Cause: "missing asset"},
	}
	report.Cases = append(report.Cases,
		CaseReport{Name: "block asset deterministic manifest hashes", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block asset local embedded only", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block asset bounded cache", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block asset icon tint evidence", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block asset image scale evidence", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block asset missing fallback diagnostic", Kind: "negative", Ran: true, Pass: true, ExpectedError: "missing asset"},
		CaseReport{Name: "block asset network url rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "network assets disabled"},
	)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block asset report: %v", err)
	}
	return raw
}
func validHeadlessBlockAccessibilitySurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(validHeadlessSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode headless report: %v", err)
	}
	report.Source = "examples/surface_block_accessibility.tetra"
	report.Processes = []ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_accessibility.tetra -o /tmp/surface-artifacts/surface-block-accessibility", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-accessibility", Ran: true, Pass: true, ExitCode: intPtrForTest(1), ExpectedExitCode: intPtrForTest(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-accessibility", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
	}
	report.Artifacts = []ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-block-accessibility", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 49172},
		{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 409},
	}
	report.ArtifactScan = ArtifactScanReport{Root: "/tmp/surface-artifacts", FilesChecked: 2, ForbiddenPaths: nil, Pass: true}
	report.Components = blockAccessibilityComponentsForTest()
	report.BlockGraph = blockGraphReportForTest(report.Source)
	report.BlockAccessibilityTree = blockAccessibilityTreeForTest(report.Source)
	report.Events = blockAccessibilityEventsForTest()
	report.Frames = []FrameReport{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", Presented: true},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, Checksum: "2222222222222222222222222222222222222222222222222222222222222222", Presented: true},
	}
	report.StateTransitions = []StateTransitionReport{
		{Order: 1, Component: "SubmitBlock", Field: "focused", Before: "false", After: "true", Cause: "tab"},
		{Order: 2, Component: "ResetBlock", Field: "focused", Before: "false", After: "true", Cause: "tab"},
		{Order: 3, Component: "BlockAccessibilityApp", Field: "reading_order_checked", Before: "false", After: "true", Cause: "block_graph"},
	}
	report.Cases = append(report.Cases,
		CaseReport{Name: "block graph duplicate id rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "duplicate Block ID"},
		CaseReport{Name: "block graph missing parent rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "missing parent"},
		CaseReport{Name: "block graph cycle rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "cycle"},
		CaseReport{Name: "block graph child order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block graph focus order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block graph hit-test path", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block graph accessibility order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block accessibility tree derived from block graph", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block accessibility focusable actionable name required", Kind: "negative", Ran: true, Pass: true, ExpectedError: "missing accessible name"},
		CaseReport{Name: "block accessibility label relationship mismatch rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "label relationship mismatch"},
		CaseReport{Name: "block accessibility reading order graph mismatch rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "reading order mismatch"},
		CaseReport{Name: "block accessibility screen-reader claim without platform proof rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "screen reader proof required"},
		CaseReport{Name: "block accessibility platform claim scoped metadata only", Kind: "positive", Ran: true, Pass: true},
	)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block accessibility report: %v", err)
	}
	return raw
}
func blockAccessibilityComponentsForTest() []ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}
	return []ComponentReport{
		{ID: "BlockAccessibilityApp", Type: "examples.surface_block_accessibility.BlockAccessibilityApp", Bounds: RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"focused_id": "4", "a11y_quality": "block-derived-accessibility-metadata-v1"}},
		{ID: "LabelBlock", Type: "examples.surface_block_accessibility.LabelBlock", Parent: "BlockAccessibilityApp", Bounds: RectReport{X: 24, Y: 24, W: 200, H: 24}, Abilities: abilities, State: map[string]string{"text_len": "4", "label_for": "4"}},
		{ID: "SubmitBlock", Type: "examples.surface_block_accessibility.ActionBlock", Parent: "BlockAccessibilityApp", Bounds: RectReport{X: 24, Y: 64, W: 120, H: 44}, Abilities: abilities, State: map[string]string{"focused": "true", "action": "submit"}},
		{ID: "ResetBlock", Type: "examples.surface_block_accessibility.ActionBlock", Parent: "BlockAccessibilityApp", Bounds: RectReport{X: 152, Y: 64, W: 120, H: 44}, Abilities: abilities, State: map[string]string{"focused": "false", "action": "reset"}},
	}
}
func blockAccessibilityTreeForTest(source string) *BlockAccessibilityTreeReport {
	return &BlockAccessibilityTreeReport{
		Schema:                  "tetra.surface.block-accessibility-tree.v1",
		AccessibilityLevel:      "block-metadata-tree-v1",
		Source:                  source,
		Module:                  "lib.core.block",
		QualityLevel:            "block-derived-accessibility-metadata-v1",
		BlockGraphSchema:        "tetra.surface.block-graph.v1",
		DerivedFromBlockGraph:   true,
		ManualBookkeeping:       false,
		PlatformHostIntegration: false,
		DOMARIAIntegration:      false,
		ScreenReaderEvidence:    false,
		NoDOMUI:                 true,
		NoUserJS:                true,
		NoPlatformWidgets:       true,
		NodeCount:               3,
		FocusableCount:          2,
		RolesPresent:            []string{"text", "button"},
		FocusOrder:              []int{4, 5},
		ReadingOrder:            []int{3, 4, 5},
		Nodes: []BlockAccessibilityNodeReport{
			{ID: 3, BlockID: 3, ParentBlockID: 2, Name: "LabelBlock", Role: "text", Bounds: RectReport{X: 24, Y: 24, W: 200, H: 24}, Visible: true, Enabled: true, Focusable: false, LabelFor: "SubmitBlock", FocusIndex: -1, ReadingIndex: 0},
			{ID: 4, BlockID: 4, ParentBlockID: 2, Name: "SubmitBlock", Role: "button", Description: "primary action", Bounds: RectReport{X: 24, Y: 64, W: 120, H: 44}, Visible: true, Enabled: true, Focusable: true, Focused: true, LabelledBy: "LabelBlock", Actions: []string{"focus", "press", "submit"}, FocusIndex: 0, ReadingIndex: 1},
			{ID: 5, BlockID: 5, ParentBlockID: 2, Name: "ResetBlock", Role: "button", Description: "secondary action", Bounds: RectReport{X: 152, Y: 64, W: 120, H: 44}, Visible: true, Enabled: true, Focusable: true, Actions: []string{"focus", "press", "reset"}, FocusIndex: 1, ReadingIndex: 2},
		},
		Relationships: []AccessibilityRelationshipReport{
			{Kind: "label_for", From: "LabelBlock", To: "SubmitBlock"},
			{Kind: "labelled_by", From: "SubmitBlock", To: "LabelBlock"},
		},
		Actions: []AccessibilityActionReport{
			{Target: "SubmitBlock", Action: "press", Semantic: "submit"},
			{Target: "ResetBlock", Action: "press", Semantic: "reset"},
		},
		NegativeGuards: BlockAccessibilityNegativeGuardsReport{
			FocusableActionNameChecked:    true,
			LabelRelationshipsChecked:     true,
			ReadingOrderGraphChecked:      true,
			BoundsAlignmentChecked:        true,
			FakeScreenReaderClaimRejected: true,
			ScopedPlatformClaimChecked:    true,
		},
	}
}
func blockAccessibilityEventsForTest() []EventReport {
	return []EventReport{
		{Order: 1, Kind: "mouse_up", TargetComponent: "SubmitBlock", DispatchPath: []string{"BlockAccessibilityApp", "SubmitBlock"}, Handled: true, Pass: true, X: 40, Y: 80, Width: 320, Height: 200, BufferSlots: []int{5, 40, 80, 1, 0, 320, 200, 0, 0}, BeforeState: map[string]string{"SubmitBlock.focused": "false"}, AfterState: map[string]string{"SubmitBlock.focused": "true"}},
		{Order: 2, Kind: "text_input", TargetComponent: "SubmitBlock", DispatchPath: []string{"BlockAccessibilityApp", "SubmitBlock"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 1, TextLen: 2, TextBytesHex: "4f4b", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 1, 2}, BeforeState: map[string]string{"SubmitBlock.value_len": "0"}, AfterState: map[string]string{"SubmitBlock.value_len": "2"}},
		{Order: 3, Kind: "key_down", TargetComponent: "SubmitBlock", DispatchPath: []string{"BlockAccessibilityApp", "SubmitBlock"}, Handled: true, Pass: true, Key: 13, Width: 320, Height: 200, TimestampMS: 2, BufferSlots: []int{3, 0, 0, 0, 13, 320, 200, 2, 0}, BeforeState: map[string]string{"SubmitBlock.pressed": "false"}, AfterState: map[string]string{"SubmitBlock.pressed": "true"}},
	}
}
func blockAssetComponentsForTest() []ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility", "asset"}
	return []ComponentReport{
		{ID: "BlockAssetApp", Type: "examples.surface_block_assets.BlockAssetApp", Bounds: RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"asset_quality": "deterministic-local-block-assets-v1"}},
		{ID: "IconBlock", Type: "examples.surface_block_assets.IconBlock", Parent: "BlockAssetApp", Bounds: RectReport{X: 24, Y: 36, W: 32, H: 32}, Abilities: abilities, State: map[string]string{"asset_id": "icon-settings", "tint": "#60aef4ff"}},
		{ID: "ImageBlock", Type: "examples.surface_block_assets.ImageBlock", Parent: "BlockAssetApp", Bounds: RectReport{X: 72, Y: 32, W: 96, H: 64}, Abilities: abilities, State: map[string]string{"asset_id": "image-hero", "scale": "2x"}},
		{ID: "MissingAssetBlock", Type: "examples.surface_block_assets.MissingAssetBlock", Parent: "BlockAssetApp", Bounds: RectReport{X: 24, Y: 112, W: 96, H: 32}, Abilities: abilities, State: map[string]string{"asset_id": "missing-logo", "fallback": "fallback-raster"}},
	}
}
func blockAssetManifestForTest(source string) *BlockAssetManifestReport {
	return &BlockAssetManifestReport{
		Schema:        "tetra.surface.block-assets.v1",
		Source:        source,
		Quality:       "deterministic-local-block-assets-v1",
		HashAlgorithm: "sha256",
		ManifestHash:  "sha256:9999999999999999999999999999999999999999999999999999999999999999",
		LocalOnly:     true,
		FontCount:     1,
		IconCount:     1,
		ImageCount:    1,
		EmbeddedCount: 3,
		RemoteCount:   0,
		Assets: []BlockAssetReport{
			{ID: "font-ui", Kind: "font", Path: "embedded://surface/font-ui", Embedded: true, Local: true, SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 2048, Family: "Tetra UI", CacheKey: "font-ui"},
			{ID: "icon-settings", Kind: "icon", Path: "embedded://surface/icon-settings", Embedded: true, Local: true, SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Size: 256, Width: 16, Height: 16, CacheKey: "icon-settings"},
			{ID: "image-hero", Kind: "image", Path: "embedded://surface/image-hero", Embedded: true, Local: true, SHA256: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", Size: 1024, Width: 48, Height: 32, CacheKey: "image-hero"},
		},
	}
}
func blockAssetCacheForTest() BlockAssetCacheReport {
	return BlockAssetCacheReport{ID: "asset-cache", Strategy: "bounded-lru", BudgetBytes: 65536, UsedBytes: 5376, EntryCount: 3, MaxEntries: 16, RepeatedLoads: 6, Eviction: "lru", Bounded: true}
}
func blockAssetDiagnosticsForTest() []BlockAssetDiagnosticReport {
	return []BlockAssetDiagnosticReport{
		{Order: 1, AssetID: "missing-logo", Kind: "image", Code: "missing_asset_fallback", Message: "missing local asset resolved to fallback raster", FallbackID: "fallback-raster-image", Pass: true},
		{Order: 2, AssetID: "https://assets.example.test/logo.png", Kind: "image", Code: "network_asset_rejected", Message: "network assets are disabled for Surface Block v1", RejectedURL: "https://assets.example.test/logo.png", Pass: true},
	}
}
func blockAssetRenderCommandsForTest() []BlockAssetRenderCommandReport {
	return []BlockAssetRenderCommandReport{
		{Order: 1, Command: "load_font", AssetID: "font-ui", BlockID: 1, Rect: RectReport{X: 0, Y: 0, W: 320, H: 200}, Quality: "font-manifest-metadata-v1", Checksum: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},
		{Order: 2, Command: "tint_icon", AssetID: "icon-settings", BlockID: 2, Rect: RectReport{X: 24, Y: 36, W: 32, H: 32}, Tint: "#60aef4ff", Scale: 1, Quality: "icon-tint-software-v1", Checksum: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{Order: 3, Command: "scale_image", AssetID: "image-hero", BlockID: 3, Rect: RectReport{X: 72, Y: 32, W: 96, H: 64}, Scale: 2, Quality: "nearest-scale-v1", Checksum: "sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"},
		{Order: 4, Command: "fallback_missing", AssetID: "missing-logo", BlockID: 4, Rect: RectReport{X: 24, Y: 112, W: 96, H: 32}, Quality: "fallback-raster-v1", Checksum: "sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"},
	}
}
func blockAssetEventsForTest() []EventReport {
	return []EventReport{
		{Order: 1, Kind: "mouse_up", TargetComponent: "IconBlock", DispatchPath: []string{"BlockAssetApp", "IconBlock"}, Handled: true, Pass: true, X: 32, Y: 44, Width: 320, Height: 200, TimestampMS: 0, BufferSlots: []int{5, 32, 44, 1, 0, 320, 200, 0, 0}, BeforeState: map[string]string{"IconBlock.tint": "#ffffffff"}, AfterState: map[string]string{"IconBlock.tint": "#60aef4ff"}},
		{Order: 2, Kind: "text_input", TargetComponent: "IconBlock", DispatchPath: []string{"BlockAssetApp", "IconBlock"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 1, TextLen: 2, TextBytesHex: "4f4b", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 1, 2}, BeforeState: map[string]string{"IconBlock.label": ""}, AfterState: map[string]string{"IconBlock.label": "OK"}},
	}
}
func removeBlockAssetRenderCommand(commands []BlockAssetRenderCommandReport, command string) []BlockAssetRenderCommandReport {
	filtered := commands[:0]
	for _, current := range commands {
		if current.Command == command {
			continue
		}
		filtered = append(filtered, current)
	}
	return filtered
}
