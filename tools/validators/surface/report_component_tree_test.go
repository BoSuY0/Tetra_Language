package surface

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateReportAcceptsHeadlessComponentTreeSurfaceRuntimeEvidence(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportRejectsComponentTreeMissingAPIEvidence(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		report.ComponentTreeAPI = nil
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree API report without component_tree_api to fail")
	}
	if !strings.Contains(err.Error(), "component_tree_api") {
		t.Fatalf("error = %v, want component_tree_api diagnostic", err)
	}
}
func TestValidateReportRejectsComponentTreeManualBookkeepingAPIEvidence(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		report.ComponentTreeAPI.ManualBookkeeping = true
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree API report with manual bookkeeping to fail")
	}
	for _, want := range []string{"component_tree_api", "manual_bookkeeping"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeAPINodeCountMismatch(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		report.ComponentTreeAPI.Builder.NodeCount = 6
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree API report with builder node count mismatch to fail")
	}
	for _, want := range []string{"builder", "node_count"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeAPIMissingTreeValidate(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		report.ComponentTreeAPI.Invariants.TreeValidateRan = false
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree API report without tree_validate evidence to fail")
	}
	for _, want := range []string{"tree_validate", "component_tree_api"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeAPIMissingRowLayoutHelper(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		report.ComponentTreeAPI.LayoutHelpers = []ComponentTreeAPILayoutHelperReport{
			{Helper: "tree_layout_column", Target: "Column", Pass: "initial", ChangedBounds: true},
			{Helper: "tree_layout_column", Target: "Column", Pass: "resize", ChangedBounds: true},
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree API report without tree_layout_row evidence to fail")
	}
	for _, want := range []string{"tree_layout_row", "component_tree_api"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeAPIMissingFocusWrap(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		report.ComponentTreeAPI.FocusHelpers = []ComponentTreeAPIFocusHelperReport{
			{Helper: "tree_focus_next", Before: "TextBox", After: "SubmitButton"},
			{Helper: "tree_focus_next", Before: "SubmitButton", After: "ResetButton"},
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree API report without ResetButton -> TextBox helper evidence to fail")
	}
	for _, want := range []string{"ResetButton", "TextBox"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeAPIHitTestPathSkippingRow(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		for i := range report.ComponentTreeAPI.HitTests {
			if report.ComponentTreeAPI.HitTests[i].Target == "ResetButton" {
				report.ComponentTreeAPI.HitTests[i].Path = []int{0, 1, 6}
			}
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree API hit-test path skipping Row to fail")
	}
	for _, want := range []string{"hit", "path"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeAPISourceMismatch(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		report.ComponentTreeAPI.Source = "examples/surface_counter.tetra"
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree API source mismatch to fail")
	}
	for _, want := range []string{"source", "component_tree_api"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeMissingEvidence(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		report.ComponentTree = nil
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree report without component_tree to fail")
	}
	if !strings.Contains(err.Error(), "component_tree") {
		t.Fatalf("error = %v, want component_tree diagnostic", err)
	}
}
func TestValidateReportRejectsHardcodedTreeClickEvidence(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		report.ComponentTree.DispatchPaths = nil
		report.Events = []EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"TextBox"},
				Handled:         true,
				Pass:            true,
				X:               40,
				Y:               72,
				Width:           320,
				Height:          200,
				BufferSlots:     []int{5, 40, 72, 1, 0, 320, 200, 0, 0},
				BeforeState:     map[string]string{"TreeApp.focused_id": "-1"},
				AfterState:      map[string]string{"TreeApp.focused_id": "3"},
			},
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected hardcoded component click evidence without root-to-leaf path to fail")
	}
	for _, want := range []string{"dispatch path", "parent"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeDispatchPathSkippingRow(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		for i := range report.ComponentTree.DispatchPaths {
			if report.ComponentTree.DispatchPaths[i].TargetID == 6 {
				report.ComponentTree.DispatchPaths[i].Path = []int{0, 1, 6}
			}
		}
		for i := range report.Events {
			if report.Events[i].TargetComponent == "ResetButton" {
				report.Events[i].DispatchPath = []string{"TreeApp", "Column", "ResetButton"}
			}
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree dispatch path skipping Row to fail")
	}
	for _, want := range []string{"dispatch path", "parent"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeTextMutationWhileButtonFocused(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		for i := range report.Events {
			if report.Events[i].Order == 6 {
				report.Events[i].AfterState["TextBox.buffer"] = "BAD"
			}
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected TextBox mutation while Button focused to fail")
	}
	for _, want := range []string{"TextBox", "Button focused"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeResizeWithoutLayoutChange(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		for i := range report.Events {
			if report.Events[i].Kind == "resize" {
				report.Events[i].AfterState["TextBox.bounds.w"] = "288"
			}
		}
		for i := range report.StateTransitions {
			if report.StateTransitions[i].Field == "TextBox.bounds.w" {
				report.StateTransitions[i].After = "288"
			}
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree resize without changed layout bounds to fail")
	}
	for _, want := range []string{"resize", "bounds"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeFocusOrderNotTreeOrder(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		report.ComponentTree.FocusOrder = []int{3, 6, 5}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree report with shuffled focus_order to fail")
	}
	for _, want := range []string{"focus_order", "TextBox -> SubmitButton -> ResetButton"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeMissingFocusWrapEvidence(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		var events []EventReport
		for _, event := range report.Events {
			if event.Kind == "key_down" && event.Key == 9 &&
				event.BeforeState["TreeApp.focused_id"] == "6" &&
				event.AfterState["TreeApp.focused_id"] == "3" {
				continue
			}
			events = append(events, event)
		}
		report.Events = events
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree report without ResetButton -> TextBox Tab wrap to fail")
	}
	for _, want := range []string{"Tab focus traversal", "ResetButton -> TextBox"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeButtonActionWithoutFocusedKeyRoute(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		for i := range report.Events {
			if report.Events[i].Kind == "key_down" &&
				(report.Events[i].TargetComponent == "SubmitButton" || report.Events[i].TargetComponent == "ResetButton") {
				report.Events[i].Kind = "mouse_up"
				report.Events[i].Key = 0
			}
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree report without focused keyboard button action route to fail")
	}
	for _, want := range []string{"button action", "keyboard"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeRowChildrenOverlap(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		for i := range report.ComponentTree.Nodes {
			if report.ComponentTree.Nodes[i].Name == "ResetButton" {
				report.ComponentTree.Nodes[i].Bounds.X = 100
			}
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree report with overlapping Row children to fail")
	}
	for _, want := range []string{"Row", "overlap"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsComponentTreeColumnChildrenOutOfOrder(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, func(report *Report) {
		for i := range report.ComponentTree.Nodes {
			if report.ComponentTree.Nodes[i].Name == "NameLabel" {
				report.ComponentTree.Nodes[i].Bounds.Y = 40
			}
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected component tree report with Column children out of visual order to fail")
	}
	for _, want := range []string{"Column", "child_index"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func validHeadlessComponentTreeSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	report := Report{
		Schema:        SchemaV1,
		Status:        "pass",
		Target:        "headless",
		Host:          "linux-x64",
		Runtime:       "surface-headless",
		SurfaceSchema: "tetra.surface.v1",
		HostABI:       "tetra.surface.host-abi.v1",
		HostEvidence: HostEvidenceReport{
			Level:       "deterministic-headless",
			Backend:     "software-rgba",
			Framebuffer: true,
		},
		Source: "examples/surface_tree_app.tetra",
		Processes: []ProcessReport{
			{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_tree_app.tetra -o /tmp/surface-artifacts/surface-tree-app", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
			{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-tree-app", Ran: true, Pass: true, ExitCode: intPtrForTest(1), ExpectedExitCode: intPtrForTest(1)},
			{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
		},
		Artifacts: []ArtifactReport{
			{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-tree-app", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 81234},
			{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 22000},
		},
		ArtifactScan: ArtifactScanReport{Root: "/tmp/surface-artifacts", FilesChecked: 2, ForbiddenPaths: []string{}, Pass: true},
		Components: []ComponentReport{
			treeComponent("TreeApp", "examples.surface_tree_app.TreeApp", "", RectReport{X: 0, Y: 0, W: 400, H: 240}, map[string]string{"focused_id": "6", "submitted_count": "1", "reset_count": "1", "width": "400", "height": "240", "accessibility_role": "none"}),
			treeComponent("Column", "examples.surface_tree_app.Column", "TreeApp", RectReport{X: 0, Y: 0, W: 400, H: 240}, map[string]string{"child_count": "3", "accessibility_role": "none"}),
			treeComponent("NameLabel", "examples.surface_tree_app.TextLabel", "Column", RectReport{X: 16, Y: 16, W: 288, H: 24}, map[string]string{"text": "Name", "accessibility_role": "label"}),
			treeComponent("TextBox", "examples.surface_tree_app.TextBox", "Column", RectReport{X: 16, Y: 48, W: 368, H: 44}, map[string]string{"focused": "false", "buffer": "", "text_len": "0", "caret": "0", "accessibility_role": "label"}),
			treeComponent("ButtonRow", "examples.surface_tree_app.Row", "Column", RectReport{X: 16, Y: 104, W: 368, H: 44}, map[string]string{"child_count": "2", "accessibility_role": "none"}),
			treeComponent("SubmitButton", "examples.surface_tree_app.Button", "ButtonRow", RectReport{X: 16, Y: 104, W: 132, H: 44}, map[string]string{"focused": "false", "press_count": "1", "accessibility_role": "button"}),
			treeComponent("ResetButton", "examples.surface_tree_app.Button", "ButtonRow", RectReport{X: 160, Y: 104, W: 132, H: 44}, map[string]string{"focused": "true", "press_count": "1", "accessibility_role": "button"}),
		},
		ComponentTree: &ComponentTreeReport{
			Schema:       "tetra.surface.component-tree.v1",
			DynamicLevel: "semi-dynamic-child-list",
			RootID:       0,
			NodeCount:    7,
			FocusedID:    6,
			Nodes: []ComponentTreeNodeReport{
				{ID: 0, Name: "TreeApp", Kind: "root", ParentID: -1, ChildIndex: 0, FirstChild: 1, ChildCount: 1, Focusable: false, Bounds: RectReport{X: 0, Y: 0, W: 400, H: 240}},
				{ID: 1, Name: "Column", Kind: "column", ParentID: 0, ChildIndex: 0, FirstChild: 2, ChildCount: 3, Focusable: false, Bounds: RectReport{X: 0, Y: 0, W: 400, H: 240}},
				{ID: 2, Name: "NameLabel", Kind: "text", ParentID: 1, ChildIndex: 0, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: RectReport{X: 16, Y: 16, W: 288, H: 24}},
				{ID: 3, Name: "TextBox", Kind: "textbox", ParentID: 1, ChildIndex: 1, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: RectReport{X: 16, Y: 48, W: 368, H: 44}},
				{ID: 4, Name: "ButtonRow", Kind: "row", ParentID: 1, ChildIndex: 2, FirstChild: 5, ChildCount: 2, Focusable: false, Bounds: RectReport{X: 16, Y: 104, W: 368, H: 44}},
				{ID: 5, Name: "SubmitButton", Kind: "button", ParentID: 4, ChildIndex: 0, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: RectReport{X: 16, Y: 104, W: 132, H: 44}},
				{ID: 6, Name: "ResetButton", Kind: "button", ParentID: 4, ChildIndex: 1, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: RectReport{X: 160, Y: 104, W: 132, H: 44}},
			},
			LayoutPasses: []ComponentTreeLayoutPassReport{
				{ComponentID: 3, Pass: "initial", Bounds: RectReport{X: 16, Y: 48, W: 288, H: 44}, Measured: SizeReport{W: 288, H: 44}},
				{ComponentID: 3, Pass: "resize", Bounds: RectReport{X: 16, Y: 48, W: 368, H: 44}, Measured: SizeReport{W: 368, H: 44}},
			},
			DrawOrder:  []int{0, 1, 2, 3, 4, 5, 6},
			FocusOrder: []int{3, 5, 6},
			DispatchPaths: []ComponentTreeDispatchPathReport{
				{Event: "click", TargetID: 3, X: 40, Y: 72, Path: []int{0, 1, 3}},
				{Event: "click", TargetID: 5, X: 32, Y: 120, Path: []int{0, 1, 4, 5}},
				{Event: "click", TargetID: 6, X: 176, Y: 120, Path: []int{0, 1, 4, 6}},
			},
		},
		ComponentTreeAPI: componentTreeAPIReportForTest(),
		Events: []EventReport{
			{Order: 1, Kind: "mouse_up", TargetComponent: "TextBox", DispatchPath: []string{"TreeApp", "Column", "TextBox"}, Handled: true, Pass: true, X: 40, Y: 72, Width: 320, Height: 200, BufferSlots: []int{5, 40, 72, 1, 0, 320, 200, 0, 0}, BeforeState: map[string]string{"TreeApp.focused_id": "-1", "TextBox.focused": "false"}, AfterState: map[string]string{"TreeApp.focused_id": "3", "TextBox.focused": "true"}},
			{Order: 2, Kind: "text_input", TargetComponent: "TextBox", DispatchPath: []string{"TreeApp", "Column", "TextBox"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 1, TextLen: 2, TextBytesHex: "4f4b", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 1, 2}, BeforeState: map[string]string{"TextBox.buffer": "", "TextBox.caret": "0"}, AfterState: map[string]string{"TextBox.buffer": "OK", "TextBox.caret": "2"}},
			{Order: 3, Kind: "key_down", TargetComponent: "TreeApp", DispatchPath: []string{"TreeApp"}, Handled: true, Pass: true, Key: 9, Width: 320, Height: 200, TimestampMS: 2, BufferSlots: []int{6, 0, 0, 0, 9, 320, 200, 2, 0}, BeforeState: map[string]string{"TreeApp.focused_id": "3"}, AfterState: map[string]string{"TreeApp.focused_id": "5"}},
			{Order: 4, Kind: "key_down", TargetComponent: "SubmitButton", DispatchPath: []string{"TreeApp", "Column", "ButtonRow", "SubmitButton"}, Handled: true, Pass: true, Key: 32, Width: 320, Height: 200, TimestampMS: 3, BufferSlots: []int{6, 0, 0, 0, 32, 320, 200, 3, 0}, BeforeState: map[string]string{"TreeApp.submitted_count": "0", "TreeApp.focused_id": "5"}, AfterState: map[string]string{"TreeApp.submitted_count": "1", "TreeApp.focused_id": "5"}},
			{Order: 5, Kind: "key_down", TargetComponent: "TreeApp", DispatchPath: []string{"TreeApp"}, Handled: true, Pass: true, Key: 9, Width: 320, Height: 200, TimestampMS: 4, BufferSlots: []int{6, 0, 0, 0, 9, 320, 200, 4, 0}, BeforeState: map[string]string{"TreeApp.focused_id": "5"}, AfterState: map[string]string{"TreeApp.focused_id": "6"}},
			{Order: 6, Kind: "text_input", TargetComponent: "ResetButton", DispatchPath: []string{"TreeApp", "Column", "ButtonRow", "ResetButton"}, Handled: false, Pass: true, Width: 320, Height: 200, TimestampMS: 5, TextLen: 1, TextBytesHex: "5a", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 5, 1}, BeforeState: map[string]string{"TreeApp.focused_id": "6", "TextBox.buffer": "OK"}, AfterState: map[string]string{"TreeApp.focused_id": "6", "TextBox.buffer": "OK"}},
			{Order: 7, Kind: "key_down", TargetComponent: "ResetButton", DispatchPath: []string{"TreeApp", "Column", "ButtonRow", "ResetButton"}, Handled: true, Pass: true, Key: 13, Width: 320, Height: 200, TimestampMS: 6, BufferSlots: []int{6, 0, 0, 0, 13, 320, 200, 6, 0}, BeforeState: map[string]string{"TreeApp.reset_count": "0", "TextBox.buffer": "OK", "TreeApp.focused_id": "6"}, AfterState: map[string]string{"TreeApp.reset_count": "1", "TextBox.buffer": "", "TreeApp.focused_id": "6"}},
			{Order: 8, Kind: "key_down", TargetComponent: "TreeApp", DispatchPath: []string{"TreeApp"}, Handled: true, Pass: true, Key: 9, Width: 320, Height: 200, TimestampMS: 7, BufferSlots: []int{6, 0, 0, 0, 9, 320, 200, 7, 0}, BeforeState: map[string]string{"TreeApp.focused_id": "6"}, AfterState: map[string]string{"TreeApp.focused_id": "3"}},
			{Order: 9, Kind: "key_down", TargetComponent: "TreeApp", DispatchPath: []string{"TreeApp"}, Handled: true, Pass: true, Key: 9, Width: 320, Height: 200, TimestampMS: 8, BufferSlots: []int{6, 0, 0, 0, 9, 320, 200, 8, 0}, BeforeState: map[string]string{"TreeApp.focused_id": "3"}, AfterState: map[string]string{"TreeApp.focused_id": "5"}},
			{Order: 10, Kind: "key_down", TargetComponent: "TreeApp", DispatchPath: []string{"TreeApp"}, Handled: true, Pass: true, Key: 9, Width: 320, Height: 200, TimestampMS: 9, BufferSlots: []int{6, 0, 0, 0, 9, 320, 200, 9, 0}, BeforeState: map[string]string{"TreeApp.focused_id": "5"}, AfterState: map[string]string{"TreeApp.focused_id": "6"}},
			{Order: 11, Kind: "resize", TargetComponent: "TreeApp", DispatchPath: []string{"TreeApp"}, Handled: true, Pass: true, Width: 400, Height: 240, TimestampMS: 10, BufferSlots: []int{2, 0, 0, 0, 0, 400, 240, 10, 0}, BeforeState: map[string]string{"TreeApp.focused_id": "6", "TextBox.bounds.w": "288"}, AfterState: map[string]string{"TreeApp.focused_id": "6", "TextBox.bounds.w": "368"}},
		},
		Frames: []FrameReport{
			{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", Presented: true},
			{Order: 2, Width: 400, Height: 240, Stride: 1600, Checksum: "2222222222222222222222222222222222222222222222222222222222222222", Presented: true},
		},
		StateTransitions: []StateTransitionReport{
			{Order: 1, Component: "TreeApp", Field: "focused_id", Before: "-1", After: "3", Cause: "mouse_up"},
			{Order: 2, Component: "TextBox", Field: "buffer", Before: "", After: "OK", Cause: "text_input"},
			{Order: 3, Component: "TreeApp", Field: "focused_id", Before: "3", After: "5", Cause: "tab"},
			{Order: 4, Component: "TreeApp", Field: "submitted_count", Before: "0", After: "1", Cause: "key_down"},
			{Order: 5, Component: "TreeApp", Field: "focused_id", Before: "5", After: "6", Cause: "tab"},
			{Order: 6, Component: "TextBox", Field: "buffer", Before: "OK", After: "", Cause: "reset"},
			{Order: 7, Component: "TreeApp", Field: "reset_count", Before: "0", After: "1", Cause: "key_down"},
			{Order: 8, Component: "TreeApp", Field: "focused_id", Before: "6", After: "3", Cause: "tab"},
			{Order: 9, Component: "TreeApp", Field: "focused_id", Before: "3", After: "5", Cause: "tab"},
			{Order: 10, Component: "TreeApp", Field: "focused_id", Before: "5", After: "6", Cause: "tab"},
			{Order: 11, Component: "TreeApp", Field: "TextBox.bounds.w", Before: "288", After: "368", Cause: "resize"},
		},
		Cases: []CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless event dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless framebuffer checksum", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless actual runner trace", Kind: "positive", Ran: true, Pass: true},
			{Name: "host-provided pointer event dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "host event buffer poll_event", Kind: "positive", Ran: true, Pass: true},
			{Name: "pre/post event frame sequence", Kind: "positive", Ran: true, Pass: true},
			{Name: "component hierarchy dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "component text input scalar dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "host text payload buffer", Kind: "positive", Ran: true, Pass: true},
			{Name: "component focus dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "component accessibility metadata", Kind: "positive", Ran: true, Pass: true},
			{Name: "no legacy UI sidecar artifacts", Kind: "positive", Ran: true, Pass: true},
			{Name: "state transition", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree node count", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree parent child links", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree layout bounds", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree draw traversal", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree pointer dispatch path", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree focus traversal", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree text routed to focused TextBox", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree button action dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree resize relayout", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree rendered frame update", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree api builder node creation", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree api parent child invariants", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree api layout helper dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree api hit test helper", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree api focus helper traversal", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree api dispatch path helper", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree api no manual bookkeeping", Kind: "positive", Ran: true, Pass: true},
			{Name: "reject legacy UI evidence", Kind: "negative", Ran: true, Pass: true, ExpectedError: "legacy UI evidence rejected"},
		},
	}
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal component tree report: %v", err)
	}
	return raw
}
func componentTreeAPIReportForTest() *ComponentTreeAPIReport {
	return &ComponentTreeAPIReport{
		Schema:            "tetra.surface.component-tree-api.v1",
		APILevel:          "builder-layout-dispatch-v1",
		Source:            "examples/surface_tree_app.tetra",
		ManualBookkeeping: false,
		Builder: ComponentTreeAPIBuilderReport{
			RootCreatedBy:     "tree_add_root",
			ChildrenCreatedBy: "tree_add_child",
			NodeCount:         7,
			Capacity:          16,
			OverflowChecked:   true,
		},
		Invariants: ComponentTreeAPIInvariantReport{
			TreeValidateRan:         true,
			TreeValidateStatus:      0,
			ParentChildLinksChecked: true,
			ChildIndicesChecked:     true,
			ChildCountChecked:       true,
			FirstChildChecked:       true,
		},
		LayoutHelpers: []ComponentTreeAPILayoutHelperReport{
			{Helper: "tree_layout_column", Target: "Column", Pass: "initial", ChangedBounds: true},
			{Helper: "tree_layout_row", Target: "ButtonRow", Pass: "initial", ChangedBounds: true},
			{Helper: "tree_layout_column", Target: "Column", Pass: "resize", ChangedBounds: true},
		},
		FocusHelpers: []ComponentTreeAPIFocusHelperReport{
			{Helper: "tree_focus_next", Before: "TextBox", After: "SubmitButton"},
			{Helper: "tree_focus_next", Before: "SubmitButton", After: "ResetButton"},
			{Helper: "tree_focus_next", Before: "ResetButton", After: "TextBox"},
		},
		HitTests: []ComponentTreeAPIHitTestReport{
			{Helper: "tree_hit_test", X: 40, Y: 72, Target: "TextBox", Path: []int{0, 1, 3}},
			{Helper: "tree_hit_test", X: 176, Y: 120, Target: "ResetButton", Path: []int{0, 1, 4, 6}},
		},
		DispatchPaths: []ComponentTreeAPIDispatchPathReport{
			{Helper: "tree_build_dispatch_path", Target: "TextBox", Path: []int{0, 1, 3}},
			{Helper: "tree_build_dispatch_path", Target: "SubmitButton", Path: []int{0, 1, 4, 5}},
			{Helper: "tree_build_dispatch_path", Target: "ResetButton", Path: []int{0, 1, 4, 6}},
		},
	}
}
