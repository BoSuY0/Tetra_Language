package surface

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateReportAcceptsHeadlessMinimalToolkitSurfaceRuntimeEvidence(t *testing.T) {
	raw := validHeadlessMinimalToolkitSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceToolkitReuseReport(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceProductionToolkitReport(t *testing.T) {
	raw := validHeadlessProductionToolkitSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceProductionToolkitRejectsMissingReleaseWidget(t *testing.T) {
	raw := validHeadlessProductionToolkitSurfaceReportJSON(t, func(report map[string]any) {
		toolkit := report["toolkit"].(map[string]any)
		toolkit["widget_set"] = []any{"Text", "Label", "StatusText", "Button", "TextBox", "Row", "Column", "Panel", "Stack", "Scroll", "Spacer"}
		widgets := toolkit["widgets"].([]any)
		filtered := make([]any, 0, len(widgets))
		for _, rawWidget := range widgets {
			widget := rawWidget.(map[string]any)
			if widget["kind"] == "Checkbox" {
				continue
			}
			filtered = append(filtered, widget)
		}
		toolkit["widgets"] = filtered
	})
	err := ValidateReport(raw)
	if err == nil || !strings.Contains(err.Error(), "Checkbox") {
		t.Fatalf("ValidateReport error = %v, want missing Checkbox rejection", err)
	}
}
func TestValidateSurfaceProductionToolkitRejectsSingleExampleClaim(t *testing.T) {
	raw := validHeadlessProductionToolkitSurfaceReportJSON(t, func(report map[string]any) {
		toolkit := report["toolkit"].(map[string]any)
		toolkit["example_count"] = 1
		toolkit["sources"] = []any{"examples/surface_release_form.tetra"}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected production toolkit single-example claim to fail")
	}
	for _, want := range []string{"production toolkit", "example_count"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateSurfaceToolkitRejectsSingleExampleReuseClaim(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		toolkit := report["toolkit"].(map[string]any)
		toolkit["example_count"] = 1
		toolkit["sources"] = []any{"examples/surface_toolkit_settings.tetra"}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse report with one example to fail")
	}
	for _, want := range []string{"toolkit", "example_count"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateSurfaceToolkitRejectsMissingWidgetsModule(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		toolkit := report["toolkit"].(map[string]any)
		toolkit["module"] = "examples.local.widgets"
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse report with wrong module to fail")
	}
	if !strings.Contains(err.Error(), "module") {
		t.Fatalf("error = %v, want module diagnostic", err)
	}
}
func TestValidateSurfaceToolkitRejectsProductionClaim(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		toolkit := report["toolkit"].(map[string]any)
		toolkit["production_claim"] = true
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse production claim to fail")
	}
	for _, want := range []string{"toolkit", "production"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateSurfaceToolkitRejectsDemoLocalWidgetStructs(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		toolkit := report["toolkit"].(map[string]any)
		toolkit["demo_specific_widget_structs"] = true
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse demo-local widget structs to fail")
	}
	if !strings.Contains(err.Error(), "demo_specific_widget_structs") {
		t.Fatalf("error = %v, want demo_specific_widget_structs diagnostic", err)
	}
}
func TestValidateSurfaceToolkitRejectsManualTreeBookkeeping(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		toolkit := report["toolkit"].(map[string]any)
		toolkit["manual_bookkeeping"] = true
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse manual bookkeeping to fail")
	}
	if !strings.Contains(err.Error(), "manual_bookkeeping") {
		t.Fatalf("error = %v, want manual_bookkeeping diagnostic", err)
	}
}
func TestValidateSurfaceToolkitRejectsMissingSecondTextBoxRouting(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		events := report["events"].([]any)
		var filtered []any
		for _, rawEvent := range events {
			event := rawEvent.(map[string]any)
			if event["target_component"] == "EmailTextBox" {
				continue
			}
			filtered = append(filtered, event)
		}
		report["events"] = filtered
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse missing second TextBox routing to fail")
	}
	for _, want := range []string{"EmailTextBox", "routing"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateSurfaceToolkitRejectsUnfocusedTextBoxMutation(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		events := report["events"].([]any)
		for _, rawEvent := range events {
			event := rawEvent.(map[string]any)
			if event["target_component"] == "EmailTextBox" && event["kind"] == "text_input" {
				after := event["after_state"].(map[string]any)
				after["NameTextBox.buffer"] = "AdaX"
			}
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse unfocused TextBox mutation to fail")
	}
	for _, want := range []string{"unfocused", "TextBox"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateSurfaceToolkitRejectsMissingStatusUpdate(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		var filtered []any
		for _, rawTransition := range report["state_transitions"].([]any) {
			transition := rawTransition.(map[string]any)
			if transition["component"] == "StatusText" {
				continue
			}
			filtered = append(filtered, transition)
		}
		report["state_transitions"] = filtered
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse missing StatusText update to fail")
	}
	if !strings.Contains(err.Error(), "StatusText") {
		t.Fatalf("error = %v, want StatusText diagnostic", err)
	}
}
func TestValidateSurfaceToolkitRejectsMissingResizeRelayout(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		var filtered []any
		for _, rawEvent := range report["events"].([]any) {
			event := rawEvent.(map[string]any)
			if event["kind"] == "resize" {
				continue
			}
			filtered = append(filtered, event)
		}
		report["events"] = filtered
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse missing resize relayout to fail")
	}
	if !strings.Contains(err.Error(), "resize") {
		t.Fatalf("error = %v, want resize diagnostic", err)
	}
}
func TestValidateSurfaceToolkitRejectsUnchangedFrameChecksum(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		frames := report["frames"].([]any)
		first := frames[0].(map[string]any)
		last := frames[len(frames)-1].(map[string]any)
		last["checksum"] = first["checksum"]
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse unchanged final frame to fail")
	}
	if !strings.Contains(err.Error(), "frame") {
		t.Fatalf("error = %v, want frame diagnostic", err)
	}
}
func TestValidateSurfaceToolkitRejectsDOMOrUserJSClaim(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		toolkit := report["toolkit"].(map[string]any)
		toolkit["no_dom_ui"] = false
		toolkit["no_user_js"] = false
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse DOM/user JS claim to fail")
	}
	for _, want := range []string{"no_dom_ui", "no_user_js"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateSurfaceToolkitRejectsNodeOnlyBrowserClaim(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		report["target"] = "wasm32-web"
		report["host"] = "node"
		report["host_evidence"] = map[string]any{
			"level":                        "wasm32-web-compiler-owned-loader",
			"backend":                      "node-surface-host",
			"framebuffer":                  true,
			"real_window":                  false,
			"native_input":                 false,
			"user_facing_platform_widgets": false,
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse browser evidence downgraded to Node-only to fail")
	}
	for _, want := range []string{"browser", "Node"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateSurfaceToolkitRejectsMissingArtifactScan(t *testing.T) {
	raw := validHeadlessToolkitReuseSurfaceReportJSON(t, func(report map[string]any) {
		delete(report, "artifact_scan")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected toolkit reuse report without artifact scan to fail")
	}
	if !strings.Contains(err.Error(), "artifact_scan") {
		t.Fatalf("error = %v, want artifact_scan diagnostic", err)
	}
}
func TestValidateReportRejectsMinimalToolkitMissingToolkitBlock(t *testing.T) {
	raw := validHeadlessMinimalToolkitSurfaceReportJSON(t, func(report map[string]any) {
		delete(report, "toolkit")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected minimal toolkit report without toolkit block to fail")
	}
	if !strings.Contains(err.Error(), "toolkit") {
		t.Fatalf("error = %v, want toolkit diagnostic", err)
	}
}
func TestValidateReportRejectsMinimalToolkitProductionClaim(t *testing.T) {
	raw := validHeadlessMinimalToolkitSurfaceReportJSON(t, func(report map[string]any) {
		toolkit := report["toolkit"].(map[string]any)
		toolkit["production_claim"] = true
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected minimal toolkit production claim to fail")
	}
	for _, want := range []string{"toolkit", "production"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMinimalToolkitMissingWidgetEvidence(t *testing.T) {
	raw := validHeadlessMinimalToolkitSurfaceReportJSON(t, func(report map[string]any) {
		toolkit := report["toolkit"].(map[string]any)
		widgets := toolkit["widgets"].([]any)
		toolkit["widgets"] = widgets[:len(widgets)-1]
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected minimal toolkit report without StatusText evidence to fail")
	}
	for _, want := range []string{"toolkit", "StatusText"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMinimalToolkitButtonActionWithoutFocusedDispatch(t *testing.T) {
	raw := validHeadlessMinimalToolkitSurfaceReportJSON(t, func(report map[string]any) {
		events := report["events"].([]any)
		for _, rawEvent := range events {
			event := rawEvent.(map[string]any)
			if event["target_component"] == "SubmitButton" && event["kind"] == "key_down" {
				event["dispatch_path"] = []any{"ToolkitFormApp", "Panel", "Column", "SubmitButton"}
			}
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected minimal toolkit bad Submit dispatch path to fail")
	}
	for _, want := range []string{"SubmitButton", "dispatch"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMinimalToolkitTextMutationWhileButtonFocused(t *testing.T) {
	raw := validHeadlessMinimalToolkitSurfaceReportJSON(t, func(report map[string]any) {
		events := report["events"].([]any)
		for _, rawEvent := range events {
			event := rawEvent.(map[string]any)
			if event["target_component"] == "ResetButton" && event["kind"] == "text_input" {
				after := event["after_state"].(map[string]any)
				after["TextBox.buffer"] = "BAD"
			}
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected minimal toolkit TextBox mutation while Button focused to fail")
	}
	for _, want := range []string{"TextBox", "Button focused"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func validHeadlessMinimalToolkitSurfaceReportJSON(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessComponentTreeSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode base component tree report: %v", err)
	}
	report["source"] = "examples/surface_toolkit_form.tetra"
	report["processes"] = []any{
		map[string]any{"name": "tetra build", "kind": "build", "path": "tetra build --target linux-x64 examples/surface_toolkit_form.tetra -o /tmp/surface-artifacts/surface-toolkit-form", "ran": true, "pass": true, "exit_code": 0},
		map[string]any{"name": "surface component app", "kind": "app", "path": "/tmp/surface-artifacts/surface-toolkit-form", "ran": true, "pass": true, "exit_code": 1, "expected_exit_code": 1},
		map[string]any{"name": "surface headless runtime", "kind": "runtime", "path": "tools/cmd/surface-runtime-smoke", "ran": true, "pass": true, "exit_code": 0},
	}
	report["artifacts"] = []any{
		map[string]any{"kind": "component-app", "path": "/tmp/surface-artifacts/surface-toolkit-form", "sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size": 81234},
		map[string]any{"kind": "runner-trace", "path": "/tmp/surface-artifacts/surface-runner-trace.json", "sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "size": 22000},
	}
	report["components"] = []any{
		componentMap("ToolkitFormApp", "examples.surface_toolkit_form.ToolkitFormApp", "", RectReport{X: 0, Y: 0, W: 400, H: 240}, map[string]string{"focused_id": "7", "submit_count": "1", "reset_count": "1", "status_code": "2", "width": "400", "height": "240", "accessibility_role": "none"}),
		componentMap("Panel", "lib.core.widgets.Panel", "ToolkitFormApp", RectReport{X: 0, Y: 0, W: 400, H: 240}, map[string]string{"padding": "12", "accessibility_role": "none"}),
		componentMap("Column", "lib.core.widgets.Column", "Panel", RectReport{X: 12, Y: 12, W: 376, H: 216}, map[string]string{"child_count": "4", "accessibility_role": "none"}),
		componentMap("NameLabel", "lib.core.widgets.Text", "Column", RectReport{X: 20, Y: 20, W: 360, H: 24}, map[string]string{"role": "label", "text_len": "4", "accessibility_role": "label"}),
		componentMap("TextBox", "lib.core.widgets.TextBox", "Column", RectReport{X: 20, Y: 52, W: 360, H: 44}, map[string]string{"focused": "false", "buffer": "", "text_len": "0", "caret": "0", "backspace_count": "1", "delete_count": "1", "accessibility_role": "label"}),
		componentMap("ButtonRow", "lib.core.widgets.Row", "Column", RectReport{X: 20, Y: 108, W: 360, H: 44}, map[string]string{"child_count": "2", "accessibility_role": "none"}),
		componentMap("SubmitButton", "lib.core.widgets.Button", "ButtonRow", RectReport{X: 20, Y: 108, W: 132, H: 44}, map[string]string{"focused": "false", "press_count": "1", "action": "submit", "accessibility_role": "button"}),
		componentMap("ResetButton", "lib.core.widgets.Button", "ButtonRow", RectReport{X: 164, Y: 108, W: 132, H: 44}, map[string]string{"focused": "true", "press_count": "1", "action": "reset", "accessibility_role": "button"}),
		componentMap("StatusText", "lib.core.widgets.Text", "Column", RectReport{X: 20, Y: 160, W: 360, H: 24}, map[string]string{"role": "status", "status_code": "2", "accessibility_role": "label"}),
	}
	report["component_tree"] = map[string]any{
		"schema":        "tetra.surface.component-tree.v1",
		"dynamic_level": "minimal-toolkit-widget-tree",
		"root_id":       0,
		"node_count":    9,
		"focused_id":    7,
		"nodes": []any{
			treeNodeMap(0, "ToolkitFormApp", "root", -1, 0, 1, 1, false, RectReport{X: 0, Y: 0, W: 400, H: 240}),
			treeNodeMap(1, "Panel", "panel", 0, 0, 2, 1, false, RectReport{X: 0, Y: 0, W: 400, H: 240}),
			treeNodeMap(2, "Column", "column", 1, 0, 3, 4, false, RectReport{X: 12, Y: 12, W: 376, H: 216}),
			treeNodeMap(3, "NameLabel", "text", 2, 0, -1, 0, false, RectReport{X: 20, Y: 20, W: 360, H: 24}),
			treeNodeMap(4, "TextBox", "textbox", 2, 1, -1, 0, true, RectReport{X: 20, Y: 52, W: 360, H: 44}),
			treeNodeMap(5, "ButtonRow", "row", 2, 2, 6, 2, false, RectReport{X: 20, Y: 108, W: 360, H: 44}),
			treeNodeMap(6, "SubmitButton", "button", 5, 0, -1, 0, true, RectReport{X: 20, Y: 108, W: 132, H: 44}),
			treeNodeMap(7, "ResetButton", "button", 5, 1, -1, 0, true, RectReport{X: 164, Y: 108, W: 132, H: 44}),
			treeNodeMap(8, "StatusText", "text", 2, 3, -1, 0, false, RectReport{X: 20, Y: 160, W: 360, H: 24}),
		},
		"layout_passes": []any{
			map[string]any{"component_id": 4, "pass": "initial", "bounds": rectMap(RectReport{X: 20, Y: 52, W: 280, H: 44}), "measured": map[string]any{"w": 280, "h": 44}},
			map[string]any{"component_id": 4, "pass": "resize", "bounds": rectMap(RectReport{X: 20, Y: 52, W: 360, H: 44}), "measured": map[string]any{"w": 360, "h": 44}},
			map[string]any{"component_id": 8, "pass": "status-update", "bounds": rectMap(RectReport{X: 20, Y: 160, W: 360, H: 24}), "measured": map[string]any{"w": 360, "h": 24}},
		},
		"draw_order":  []any{0, 1, 2, 3, 4, 5, 6, 7, 8},
		"focus_order": []any{4, 6, 7},
		"dispatch_paths": []any{
			map[string]any{"event": "click", "target_id": 4, "x": 40, "y": 72, "path": []any{0, 1, 2, 4}},
			map[string]any{"event": "click", "target_id": 6, "x": 40, "y": 124, "path": []any{0, 1, 2, 5, 6}},
			map[string]any{"event": "click", "target_id": 7, "x": 180, "y": 124, "path": []any{0, 1, 2, 5, 7}},
		},
	}
	report["component_tree_api"] = map[string]any{
		"schema":             "tetra.surface.component-tree-api.v1",
		"api_level":          "builder-layout-dispatch-v1",
		"source":             "examples/surface_toolkit_form.tetra",
		"manual_bookkeeping": false,
		"builder": map[string]any{
			"root_created_by":     "tree_add_root",
			"children_created_by": "tree_add_child",
			"node_count":          9,
			"capacity":            16,
			"overflow_checked":    true,
		},
		"invariants": map[string]any{
			"tree_validate_ran":          true,
			"tree_validate_status":       0,
			"parent_child_links_checked": true,
			"child_indices_checked":      true,
			"child_count_checked":        true,
			"first_child_checked":        true,
		},
		"layout_helpers": []any{
			map[string]any{"helper": "widgets.panel_content_rect", "target": "Panel", "pass": "initial", "changed_bounds": true},
			map[string]any{"helper": "widgets.column_layout", "target": "Column", "pass": "initial", "changed_bounds": true},
			map[string]any{"helper": "widgets.row_layout", "target": "ButtonRow", "pass": "initial", "changed_bounds": true},
			map[string]any{"helper": "widgets.column_layout", "target": "Column", "pass": "resize", "changed_bounds": true},
		},
		"focus_helpers": []any{
			map[string]any{"helper": "tree_focus_next", "before": "TextBox", "after": "SubmitButton"},
			map[string]any{"helper": "tree_focus_next", "before": "SubmitButton", "after": "ResetButton"},
			map[string]any{"helper": "tree_focus_next", "before": "ResetButton", "after": "TextBox"},
		},
		"hit_tests": []any{
			map[string]any{"helper": "widgets.hit_test", "x": 40, "y": 72, "target": "TextBox", "path": []any{0, 1, 2, 4}},
			map[string]any{"helper": "widgets.hit_test", "x": 180, "y": 124, "target": "ResetButton", "path": []any{0, 1, 2, 5, 7}},
		},
		"dispatch_paths": []any{
			map[string]any{"helper": "tree_build_dispatch_path", "target": "TextBox", "path": []any{0, 1, 2, 4}},
			map[string]any{"helper": "tree_build_dispatch_path", "target": "SubmitButton", "path": []any{0, 1, 2, 5, 6}},
			map[string]any{"helper": "tree_build_dispatch_path", "target": "ResetButton", "path": []any{0, 1, 2, 5, 7}},
		},
	}
	report["toolkit"] = map[string]any{
		"schema":                       "tetra.surface.toolkit.v1",
		"toolkit_level":                "minimal-widgets-v1",
		"source":                       "examples/surface_toolkit_form.tetra",
		"module":                       "lib.core.widgets",
		"experimental":                 true,
		"production_claim":             false,
		"uses_component_tree_api":      true,
		"manual_bookkeeping":           false,
		"demo_specific_widget_structs": false,
		"no_magic_widgets":             true,
		"no_platform_widgets":          true,
		"no_dom_ui":                    true,
		"no_user_js":                   true,
		"widgets": []any{
			toolkitWidgetMap("Panel", "Panel", 1, "", true),
			toolkitWidgetMap("Column", "Column", 2, "", true),
			toolkitWidgetMap("NameLabel", "Text", 3, "label", true),
			toolkitWidgetMap("TextBox", "TextBox", 4, "", true),
			toolkitWidgetMap("ButtonRow", "Row", 5, "", true),
			toolkitWidgetMap("SubmitButton", "Button", 6, "submit", true),
			toolkitWidgetMap("ResetButton", "Button", 7, "reset", true),
			toolkitWidgetMap("StatusText", "Text", 8, "status", true),
		},
		"reusable_sources": []any{
			"lib/core/widgets.tetra:panel_init",
			"lib/core/widgets.tetra:column_init",
			"lib/core/widgets.tetra:text_init",
			"lib/core/widgets.tetra:textbox_init",
			"lib/core/widgets.tetra:row_init",
			"lib/core/widgets.tetra:button_init",
		},
	}
	report["events"] = []any{
		eventMap(1, "mouse_up", "TextBox", []any{"ToolkitFormApp", "Panel", "Column", "TextBox"}, 40, 72, 0, 320, 200, map[string]string{"ToolkitFormApp.focused_id": "-1", "TextBox.focused": "false"}, map[string]string{"ToolkitFormApp.focused_id": "4", "TextBox.focused": "true"}),
		textEventMap(2, "TextBox", []any{"ToolkitFormApp", "Panel", "Column", "TextBox"}, 2, "4f4b", 320, 200, map[string]string{"TextBox.buffer": "", "TextBox.caret": "0", "TextBox.text_len": "0"}, map[string]string{"TextBox.buffer": "OK", "TextBox.caret": "2", "TextBox.text_len": "2"}),
		keyEventMap(3, "TextBox", []any{"ToolkitFormApp", "Panel", "Column", "TextBox"}, 37, 320, 200, map[string]string{"TextBox.buffer": "OK", "TextBox.caret": "2"}, map[string]string{"TextBox.buffer": "OK", "TextBox.caret": "1"}),
		keyEventMap(4, "TextBox", []any{"ToolkitFormApp", "Panel", "Column", "TextBox"}, 8, 320, 200, map[string]string{"TextBox.buffer": "OK", "TextBox.caret": "1"}, map[string]string{"TextBox.buffer": "K", "TextBox.caret": "0"}),
		keyEventMap(5, "TextBox", []any{"ToolkitFormApp", "Panel", "Column", "TextBox"}, 46, 320, 200, map[string]string{"TextBox.buffer": "K", "TextBox.caret": "0"}, map[string]string{"TextBox.buffer": "", "TextBox.caret": "0"}),
		textEventMap(6, "TextBox", []any{"ToolkitFormApp", "Panel", "Column", "TextBox"}, 1, "5a", 320, 200, map[string]string{"TextBox.buffer": "", "TextBox.caret": "0", "TextBox.text_len": "0"}, map[string]string{"TextBox.buffer": "Z", "TextBox.caret": "1", "TextBox.text_len": "1"}),
		keyEventMap(7, "ToolkitFormApp", []any{"ToolkitFormApp"}, 9, 320, 200, map[string]string{"ToolkitFormApp.focused_id": "4"}, map[string]string{"ToolkitFormApp.focused_id": "6"}),
		keyEventMap(8, "SubmitButton", []any{"ToolkitFormApp", "Panel", "Column", "ButtonRow", "SubmitButton"}, 32, 320, 200, map[string]string{"ToolkitFormApp.focused_id": "6", "ToolkitFormApp.submit_count": "0", "StatusText.status_code": "0", "TextBox.buffer": "Z"}, map[string]string{"ToolkitFormApp.focused_id": "6", "ToolkitFormApp.submit_count": "1", "StatusText.status_code": "1", "TextBox.buffer": "Z"}),
		keyEventMap(9, "ToolkitFormApp", []any{"ToolkitFormApp"}, 9, 320, 200, map[string]string{"ToolkitFormApp.focused_id": "6"}, map[string]string{"ToolkitFormApp.focused_id": "7"}),
		textEventMap(10, "ResetButton", []any{"ToolkitFormApp", "Panel", "Column", "ButtonRow", "ResetButton"}, 1, "58", 320, 200, map[string]string{"ToolkitFormApp.focused_id": "7", "TextBox.buffer": "Z"}, map[string]string{"ToolkitFormApp.focused_id": "7", "TextBox.buffer": "Z"}),
		keyEventMap(11, "ResetButton", []any{"ToolkitFormApp", "Panel", "Column", "ButtonRow", "ResetButton"}, 13, 320, 200, map[string]string{"ToolkitFormApp.focused_id": "7", "ToolkitFormApp.reset_count": "0", "StatusText.status_code": "1", "TextBox.buffer": "Z"}, map[string]string{"ToolkitFormApp.focused_id": "7", "ToolkitFormApp.reset_count": "1", "StatusText.status_code": "2", "TextBox.buffer": ""}),
		keyEventMap(12, "ToolkitFormApp", []any{"ToolkitFormApp"}, 9, 320, 200, map[string]string{"ToolkitFormApp.focused_id": "7"}, map[string]string{"ToolkitFormApp.focused_id": "4"}),
		resizeEventMap(13, "ToolkitFormApp", []any{"ToolkitFormApp"}, 400, 240, map[string]string{"ToolkitFormApp.focused_id": "4", "TextBox.bounds.w": "280", "TextBox.buffer": ""}, map[string]string{"ToolkitFormApp.focused_id": "4", "TextBox.bounds.w": "360", "TextBox.buffer": ""}),
	}
	report["frames"] = []any{
		map[string]any{"order": 1, "width": 320, "height": 200, "stride": 1280, "checksum": "1111111111111111111111111111111111111111111111111111111111111111", "presented": true},
		map[string]any{"order": 2, "width": 320, "height": 200, "stride": 1280, "checksum": "2222222222222222222222222222222222222222222222222222222222222222", "presented": true},
		map[string]any{"order": 3, "width": 320, "height": 200, "stride": 1280, "checksum": "3333333333333333333333333333333333333333333333333333333333333333", "presented": true},
		map[string]any{"order": 4, "width": 400, "height": 240, "stride": 1600, "checksum": "4444444444444444444444444444444444444444444444444444444444444444", "presented": true},
	}
	report["state_transitions"] = []any{
		map[string]any{"order": 1, "component": "ToolkitFormApp", "field": "focused_id", "before": "-1", "after": "4", "cause": "mouse_up"},
		map[string]any{"order": 2, "component": "TextBox", "field": "buffer", "before": "", "after": "OK", "cause": "text_input"},
		map[string]any{"order": 3, "component": "TextBox", "field": "caret", "before": "2", "after": "1", "cause": "key_down"},
		map[string]any{"order": 4, "component": "TextBox", "field": "buffer", "before": "OK", "after": "K", "cause": "backspace"},
		map[string]any{"order": 5, "component": "TextBox", "field": "buffer", "before": "K", "after": "", "cause": "delete"},
		map[string]any{"order": 6, "component": "TextBox", "field": "buffer", "before": "", "after": "Z", "cause": "text_input"},
		map[string]any{"order": 7, "component": "ToolkitFormApp", "field": "focused_id", "before": "4", "after": "6", "cause": "tab"},
		map[string]any{"order": 8, "component": "ToolkitFormApp", "field": "submit_count", "before": "0", "after": "1", "cause": "key_down"},
		map[string]any{"order": 9, "component": "StatusText", "field": "status_code", "before": "0", "after": "1", "cause": "submit"},
		map[string]any{"order": 10, "component": "ToolkitFormApp", "field": "focused_id", "before": "6", "after": "7", "cause": "tab"},
		map[string]any{"order": 11, "component": "TextBox", "field": "buffer", "before": "Z", "after": "", "cause": "reset"},
		map[string]any{"order": 12, "component": "ToolkitFormApp", "field": "reset_count", "before": "0", "after": "1", "cause": "key_down"},
		map[string]any{"order": 13, "component": "StatusText", "field": "status_code", "before": "1", "after": "2", "cause": "reset"},
		map[string]any{"order": 14, "component": "ToolkitFormApp", "field": "focused_id", "before": "7", "after": "4", "cause": "tab"},
		map[string]any{"order": 15, "component": "ToolkitFormApp", "field": "TextBox.bounds.w", "before": "280", "after": "360", "cause": "resize"},
	}
	report["cases"] = append(report["cases"].([]any),
		map[string]any{"name": "minimal toolkit reusable widgets", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "minimal toolkit Text widget evidence", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "minimal toolkit Button widget evidence", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "minimal toolkit TextBox widget evidence", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "minimal toolkit Row Column Panel layout", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "minimal toolkit tree api reuse", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "minimal toolkit TextBox focus input editing", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "minimal toolkit Submit action routed", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "minimal toolkit Reset action routed", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "minimal toolkit status text update", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "minimal toolkit resize relayout", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "minimal toolkit rendered frame update", "kind": "positive", "ran": true, "pass": true},
	)
	if mutate != nil {
		mutate(report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal minimal toolkit report: %v", err)
	}
	return raw
}
func validHeadlessToolkitReuseSurfaceReportJSON(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessMinimalToolkitSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode base toolkit report: %v", err)
	}
	report["source"] = "examples/surface_toolkit_settings.tetra"
	report["processes"] = []any{
		map[string]any{"name": "tetra build", "kind": "build", "path": "tetra build --target linux-x64 examples/surface_toolkit_settings.tetra -o /tmp/surface-artifacts/surface-toolkit-settings", "ran": true, "pass": true, "exit_code": 0},
		map[string]any{"name": "surface component app", "kind": "app", "path": "/tmp/surface-artifacts/surface-toolkit-settings", "ran": true, "pass": true, "exit_code": 1, "expected_exit_code": 1},
		map[string]any{"name": "surface headless runtime", "kind": "runtime", "path": "tools/cmd/surface-runtime-smoke", "ran": true, "pass": true, "exit_code": 0},
	}
	report["artifacts"] = []any{
		map[string]any{"kind": "component-app", "path": "/tmp/surface-artifacts/surface-toolkit-settings", "sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size": 81234},
		map[string]any{"kind": "runner-trace", "path": "/tmp/surface-artifacts/surface-runner-trace.json", "sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "size": 22000},
	}
	report["components"] = []any{
		componentMap("ToolkitSettingsApp", "examples.surface_toolkit_settings.ToolkitSettingsApp", "", RectReport{X: 0, Y: 0, W: 480, H: 320}, map[string]string{"focused_id": "4", "save_count": "1", "reset_count": "1", "status_code": "2", "width": "480", "height": "320", "accessibility_role": "none"}),
		componentMap("Panel", "lib.core.widgets.Panel", "ToolkitSettingsApp", RectReport{X: 0, Y: 0, W: 480, H: 320}, map[string]string{"padding": "12", "accessibility_role": "none"}),
		componentMap("Column", "lib.core.widgets.Column", "Panel", RectReport{X: 12, Y: 12, W: 456, H: 296}, map[string]string{"child_count": "6", "accessibility_role": "none"}),
		componentMap("TitleText", "lib.core.widgets.Text", "Column", RectReport{X: 20, Y: 20, W: 440, H: 24}, map[string]string{"role": "label", "text_len": "8", "accessibility_role": "label"}),
		componentMap("NameTextBox", "lib.core.widgets.TextBox", "Column", RectReport{X: 20, Y: 52, W: 440, H: 44}, map[string]string{"focused": "true", "buffer": "", "text_len": "0", "caret": "0", "accessibility_role": "label"}),
		componentMap("NameLabel", "lib.core.widgets.Text", "Column", RectReport{X: 20, Y: 104, W: 440, H: 24}, map[string]string{"role": "label", "text_len": "4", "accessibility_role": "label"}),
		componentMap("EmailTextBox", "lib.core.widgets.TextBox", "Column", RectReport{X: 20, Y: 136, W: 440, H: 44}, map[string]string{"focused": "false", "buffer": "", "text_len": "0", "caret": "0", "accessibility_role": "label"}),
		componentMap("ButtonRow", "lib.core.widgets.Row", "Column", RectReport{X: 20, Y: 192, W: 440, H: 44}, map[string]string{"child_count": "2", "accessibility_role": "none"}),
		componentMap("SaveButton", "lib.core.widgets.Button", "ButtonRow", RectReport{X: 20, Y: 192, W: 132, H: 44}, map[string]string{"focused": "false", "press_count": "1", "action": "save", "accessibility_role": "button"}),
		componentMap("ResetButton", "lib.core.widgets.Button", "ButtonRow", RectReport{X: 164, Y: 192, W: 132, H: 44}, map[string]string{"focused": "false", "press_count": "1", "action": "reset", "accessibility_role": "button"}),
		componentMap("StatusText", "lib.core.widgets.Text", "Column", RectReport{X: 20, Y: 248, W: 440, H: 24}, map[string]string{"role": "status", "status_code": "2", "accessibility_role": "label"}),
	}
	report["component_tree"] = map[string]any{
		"schema":        "tetra.surface.component-tree.v1",
		"dynamic_level": "toolkit-reuse-widget-tree",
		"root_id":       0,
		"node_count":    11,
		"focused_id":    4,
		"nodes": []any{
			treeNodeMap(0, "ToolkitSettingsApp", "root", -1, 0, 1, 1, false, RectReport{X: 0, Y: 0, W: 480, H: 320}),
			treeNodeMap(1, "Panel", "panel", 0, 0, 2, 1, false, RectReport{X: 0, Y: 0, W: 480, H: 320}),
			treeNodeMap(2, "Column", "column", 1, 0, 3, 6, false, RectReport{X: 12, Y: 12, W: 456, H: 296}),
			treeNodeMap(3, "TitleText", "text", 2, 0, -1, 0, false, RectReport{X: 20, Y: 20, W: 440, H: 24}),
			treeNodeMap(4, "NameTextBox", "textbox", 2, 1, -1, 0, true, RectReport{X: 20, Y: 52, W: 440, H: 44}),
			treeNodeMap(5, "NameLabel", "text", 2, 2, -1, 0, false, RectReport{X: 20, Y: 104, W: 440, H: 24}),
			treeNodeMap(6, "EmailTextBox", "textbox", 2, 3, -1, 0, true, RectReport{X: 20, Y: 136, W: 440, H: 44}),
			treeNodeMap(7, "ButtonRow", "row", 2, 4, 8, 2, false, RectReport{X: 20, Y: 192, W: 440, H: 44}),
			treeNodeMap(8, "SaveButton", "button", 7, 0, -1, 0, true, RectReport{X: 20, Y: 192, W: 132, H: 44}),
			treeNodeMap(9, "ResetButton", "button", 7, 1, -1, 0, true, RectReport{X: 164, Y: 192, W: 132, H: 44}),
			treeNodeMap(10, "StatusText", "text", 2, 5, -1, 0, false, RectReport{X: 20, Y: 248, W: 440, H: 24}),
		},
		"layout_passes": []any{
			map[string]any{"component_id": 4, "pass": "initial", "bounds": rectMap(RectReport{X: 20, Y: 52, W: 280, H: 44}), "measured": map[string]any{"w": 280, "h": 44}},
			map[string]any{"component_id": 6, "pass": "initial", "bounds": rectMap(RectReport{X: 20, Y: 136, W: 280, H: 44}), "measured": map[string]any{"w": 280, "h": 44}},
			map[string]any{"component_id": 4, "pass": "resize", "bounds": rectMap(RectReport{X: 20, Y: 52, W: 440, H: 44}), "measured": map[string]any{"w": 440, "h": 44}},
			map[string]any{"component_id": 6, "pass": "resize", "bounds": rectMap(RectReport{X: 20, Y: 136, W: 440, H: 44}), "measured": map[string]any{"w": 440, "h": 44}},
			map[string]any{"component_id": 10, "pass": "status-update", "bounds": rectMap(RectReport{X: 20, Y: 248, W: 440, H: 24}), "measured": map[string]any{"w": 440, "h": 24}},
		},
		"draw_order":  []any{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		"focus_order": []any{4, 6, 8, 9},
		"dispatch_paths": []any{
			map[string]any{"event": "click", "target_id": 4, "x": 40, "y": 72, "path": []any{0, 1, 2, 4}},
			map[string]any{"event": "click", "target_id": 6, "x": 40, "y": 156, "path": []any{0, 1, 2, 6}},
			map[string]any{"event": "key", "target_id": 8, "x": 40, "y": 208, "path": []any{0, 1, 2, 7, 8}},
			map[string]any{"event": "key", "target_id": 9, "x": 180, "y": 208, "path": []any{0, 1, 2, 7, 9}},
		},
	}
	report["component_tree_api"] = map[string]any{
		"schema":             "tetra.surface.component-tree-api.v1",
		"api_level":          "builder-layout-dispatch-v1",
		"source":             "examples/surface_toolkit_settings.tetra",
		"manual_bookkeeping": false,
		"builder": map[string]any{
			"root_created_by":     "tree_add_root",
			"children_created_by": "tree_add_child",
			"node_count":          11,
			"capacity":            20,
			"overflow_checked":    true,
		},
		"invariants": map[string]any{
			"tree_validate_ran":          true,
			"tree_validate_status":       0,
			"parent_child_links_checked": true,
			"child_indices_checked":      true,
			"child_count_checked":        true,
			"first_child_checked":        true,
		},
		"layout_helpers": []any{
			map[string]any{"helper": "widgets.panel_content_rect", "target": "Panel", "pass": "initial", "changed_bounds": true},
			map[string]any{"helper": "widgets.column_layout", "target": "Column", "pass": "initial", "changed_bounds": true},
			map[string]any{"helper": "widgets.row_layout", "target": "ButtonRow", "pass": "initial", "changed_bounds": true},
			map[string]any{"helper": "widgets.column_layout", "target": "Column", "pass": "resize", "changed_bounds": true},
		},
		"focus_helpers": []any{
			map[string]any{"helper": "tree_focus_next", "before": "NameTextBox", "after": "EmailTextBox"},
			map[string]any{"helper": "tree_focus_next", "before": "EmailTextBox", "after": "SaveButton"},
			map[string]any{"helper": "tree_focus_next", "before": "SaveButton", "after": "ResetButton"},
			map[string]any{"helper": "tree_focus_next", "before": "ResetButton", "after": "NameTextBox"},
		},
		"hit_tests": []any{
			map[string]any{"helper": "widgets.hit_test", "x": 40, "y": 72, "target": "NameTextBox", "path": []any{0, 1, 2, 4}},
			map[string]any{"helper": "widgets.hit_test", "x": 40, "y": 156, "target": "EmailTextBox", "path": []any{0, 1, 2, 6}},
			map[string]any{"helper": "widgets.hit_test", "x": 180, "y": 208, "target": "ResetButton", "path": []any{0, 1, 2, 7, 9}},
		},
		"dispatch_paths": []any{
			map[string]any{"helper": "tree_build_dispatch_path", "target": "NameTextBox", "path": []any{0, 1, 2, 4}},
			map[string]any{"helper": "tree_build_dispatch_path", "target": "EmailTextBox", "path": []any{0, 1, 2, 6}},
			map[string]any{"helper": "tree_build_dispatch_path", "target": "SaveButton", "path": []any{0, 1, 2, 7, 8}},
			map[string]any{"helper": "tree_build_dispatch_path", "target": "ResetButton", "path": []any{0, 1, 2, 7, 9}},
		},
	}
	report["toolkit"] = map[string]any{
		"schema":                       "tetra.surface.toolkit.v1",
		"toolkit_level":                "toolkit-reuse-v1",
		"reuse_level":                  "multi-form-widget-reuse-v1",
		"source":                       "examples/surface_toolkit_settings.tetra",
		"sources":                      []any{"examples/surface_toolkit_form.tetra", "examples/surface_toolkit_settings.tetra"},
		"module":                       "lib.core.widgets",
		"experimental":                 true,
		"production_claim":             false,
		"uses_component_tree_api":      true,
		"manual_bookkeeping":           false,
		"demo_specific_widget_structs": false,
		"no_magic_widgets":             true,
		"no_platform_widgets":          true,
		"no_dom_ui":                    true,
		"no_user_js":                   true,
		"example_count":                2,
		"text_box_count":               2,
		"button_count":                 2,
		"multi_textbox_evidence":       true,
		"multi_form_evidence":          true,
		"widgets": []any{
			toolkitWidgetMap("Panel", "Panel", 1, "", true),
			toolkitWidgetMap("Column", "Column", 2, "", true),
			toolkitWidgetMap("TitleText", "Text", 3, "label", true),
			toolkitWidgetMap("NameTextBox", "TextBox", 4, "", true),
			toolkitWidgetMap("NameLabel", "Text", 5, "label", true),
			toolkitWidgetMap("EmailTextBox", "TextBox", 6, "", true),
			toolkitWidgetMap("ButtonRow", "Row", 7, "", true),
			toolkitWidgetMap("SaveButton", "Button", 8, "save", true),
			toolkitWidgetMap("ResetButton", "Button", 9, "reset", true),
			toolkitWidgetMap("StatusText", "Text", 10, "status", true),
		},
		"reusable_sources": []any{
			"lib/core/widgets.tetra:panel_init",
			"lib/core/widgets.tetra:column_init",
			"lib/core/widgets.tetra:text_init",
			"lib/core/widgets.tetra:textbox_init",
			"lib/core/widgets.tetra:row_init",
			"lib/core/widgets.tetra:button_init",
			"lib/core/widgets.tetra:hit_test",
			"lib/core/widgets.tetra:textbox_text_input",
			"lib/core/widgets.tetra:button_key_event",
		},
	}
	report["events"] = []any{
		eventMap(1, "mouse_up", "NameTextBox", []any{"ToolkitSettingsApp", "Panel", "Column", "NameTextBox"}, 40, 72, 0, 320, 240, map[string]string{"ToolkitSettingsApp.focused_id": "-1", "NameTextBox.focused": "false"}, map[string]string{"ToolkitSettingsApp.focused_id": "4", "NameTextBox.focused": "true"}),
		textEventMap(2, "NameTextBox", []any{"ToolkitSettingsApp", "Panel", "Column", "NameTextBox"}, 3, "416461", 320, 240, map[string]string{"NameTextBox.buffer": "", "NameTextBox.caret": "0", "EmailTextBox.buffer": ""}, map[string]string{"NameTextBox.buffer": "Ada", "NameTextBox.caret": "3", "EmailTextBox.buffer": ""}),
		keyEventMap(3, "ToolkitSettingsApp", []any{"ToolkitSettingsApp"}, 9, 320, 240, map[string]string{"ToolkitSettingsApp.focused_id": "4"}, map[string]string{"ToolkitSettingsApp.focused_id": "6"}),
		textEventMap(4, "EmailTextBox", []any{"ToolkitSettingsApp", "Panel", "Column", "EmailTextBox"}, 5, "7465747261", 320, 240, map[string]string{"EmailTextBox.buffer": "", "NameTextBox.buffer": "Ada"}, map[string]string{"EmailTextBox.buffer": "tetra", "NameTextBox.buffer": "Ada"}),
		keyEventMap(5, "ToolkitSettingsApp", []any{"ToolkitSettingsApp"}, 9, 320, 240, map[string]string{"ToolkitSettingsApp.focused_id": "6"}, map[string]string{"ToolkitSettingsApp.focused_id": "8"}),
		keyEventMap(6, "SaveButton", []any{"ToolkitSettingsApp", "Panel", "Column", "ButtonRow", "SaveButton"}, 32, 320, 240, map[string]string{"ToolkitSettingsApp.focused_id": "8", "ToolkitSettingsApp.save_count": "0", "StatusText.status_code": "0"}, map[string]string{"ToolkitSettingsApp.focused_id": "8", "ToolkitSettingsApp.save_count": "1", "StatusText.status_code": "1"}),
		keyEventMap(7, "ToolkitSettingsApp", []any{"ToolkitSettingsApp"}, 9, 320, 240, map[string]string{"ToolkitSettingsApp.focused_id": "8"}, map[string]string{"ToolkitSettingsApp.focused_id": "9"}),
		keyEventMap(8, "ResetButton", []any{"ToolkitSettingsApp", "Panel", "Column", "ButtonRow", "ResetButton"}, 13, 320, 240, map[string]string{"ToolkitSettingsApp.focused_id": "9", "ToolkitSettingsApp.reset_count": "0", "StatusText.status_code": "1", "NameTextBox.buffer": "Ada", "EmailTextBox.buffer": "tetra"}, map[string]string{"ToolkitSettingsApp.focused_id": "9", "ToolkitSettingsApp.reset_count": "1", "StatusText.status_code": "2", "NameTextBox.buffer": "", "EmailTextBox.buffer": ""}),
		keyEventMap(9, "ToolkitSettingsApp", []any{"ToolkitSettingsApp"}, 9, 320, 240, map[string]string{"ToolkitSettingsApp.focused_id": "9"}, map[string]string{"ToolkitSettingsApp.focused_id": "4"}),
		resizeEventMap(10, "ToolkitSettingsApp", []any{"ToolkitSettingsApp"}, 480, 320, map[string]string{"ToolkitSettingsApp.focused_id": "4", "NameTextBox.bounds.w": "280", "EmailTextBox.bounds.w": "280"}, map[string]string{"ToolkitSettingsApp.focused_id": "4", "NameTextBox.bounds.w": "440", "EmailTextBox.bounds.w": "440"}),
	}
	report["frames"] = []any{
		map[string]any{"order": 1, "width": 320, "height": 240, "stride": 1280, "checksum": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "presented": true},
		map[string]any{"order": 2, "width": 320, "height": 240, "stride": 1280, "checksum": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "presented": true},
		map[string]any{"order": 3, "width": 320, "height": 240, "stride": 1280, "checksum": "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "presented": true},
		map[string]any{"order": 4, "width": 320, "height": 240, "stride": 1280, "checksum": "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", "presented": true},
		map[string]any{"order": 5, "width": 480, "height": 320, "stride": 1920, "checksum": "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", "presented": true},
	}
	report["state_transitions"] = []any{
		map[string]any{"order": 1, "component": "ToolkitSettingsApp", "field": "focused_id", "before": "-1", "after": "4", "cause": "mouse_up"},
		map[string]any{"order": 2, "component": "NameTextBox", "field": "buffer", "before": "", "after": "Ada", "cause": "text_input"},
		map[string]any{"order": 3, "component": "ToolkitSettingsApp", "field": "focused_id", "before": "4", "after": "6", "cause": "tab"},
		map[string]any{"order": 4, "component": "EmailTextBox", "field": "buffer", "before": "", "after": "tetra", "cause": "text_input"},
		map[string]any{"order": 5, "component": "ToolkitSettingsApp", "field": "focused_id", "before": "6", "after": "8", "cause": "tab"},
		map[string]any{"order": 6, "component": "ToolkitSettingsApp", "field": "save_count", "before": "0", "after": "1", "cause": "key_down"},
		map[string]any{"order": 7, "component": "StatusText", "field": "status_code", "before": "0", "after": "1", "cause": "save"},
		map[string]any{"order": 8, "component": "ToolkitSettingsApp", "field": "focused_id", "before": "8", "after": "9", "cause": "tab"},
		map[string]any{"order": 9, "component": "NameTextBox", "field": "buffer", "before": "Ada", "after": "", "cause": "reset"},
		map[string]any{"order": 10, "component": "EmailTextBox", "field": "buffer", "before": "tetra", "after": "", "cause": "reset"},
		map[string]any{"order": 11, "component": "ToolkitSettingsApp", "field": "reset_count", "before": "0", "after": "1", "cause": "key_down"},
		map[string]any{"order": 12, "component": "StatusText", "field": "status_code", "before": "1", "after": "2", "cause": "reset"},
		map[string]any{"order": 13, "component": "ToolkitSettingsApp", "field": "focused_id", "before": "9", "after": "4", "cause": "tab"},
		map[string]any{"order": 14, "component": "ToolkitSettingsApp", "field": "NameTextBox.bounds.w", "before": "280", "after": "440", "cause": "resize"},
		map[string]any{"order": 15, "component": "ToolkitSettingsApp", "field": "EmailTextBox.bounds.w", "before": "280", "after": "440", "cause": "resize"},
	}
	report["cases"] = append(report["cases"].([]any),
		map[string]any{"name": "toolkit reuse second example evidence", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "toolkit reuse widgets module evidence", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "toolkit reuse multi TextBox routing", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "toolkit reuse focused TextBox only mutates", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "toolkit reuse Save action routed", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "toolkit reuse Reset action routed", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "toolkit reuse StatusText updates", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "toolkit reuse resize relayout", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "toolkit reuse changed frame checksums", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "toolkit reuse no demo-local widget structs", "kind": "positive", "ran": true, "pass": true},
	)
	if mutate != nil {
		mutate(report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal toolkit reuse report: %v", err)
	}
	return raw
}
func validHeadlessProductionToolkitSurfaceReportJSON(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessToolkitReuseSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode base toolkit reuse report: %v", err)
	}
	report["source"] = "examples/surface_release_form.tetra"
	report["processes"] = []any{
		map[string]any{"name": "tetra build", "kind": "build", "path": "tetra build --target linux-x64 examples/surface_release_form.tetra -o /tmp/surface-artifacts/surface-release-form", "ran": true, "pass": true, "exit_code": 0},
		map[string]any{"name": "surface component app", "kind": "app", "path": "/tmp/surface-artifacts/surface-release-form", "ran": true, "pass": true, "exit_code": 1, "expected_exit_code": 1},
		map[string]any{"name": "surface headless runtime", "kind": "runtime", "path": "tools/cmd/surface-runtime-smoke", "ran": true, "pass": true, "exit_code": 0},
	}
	report["artifacts"] = []any{
		map[string]any{"kind": "component-app", "path": "/tmp/surface-artifacts/surface-release-form", "sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size": 98234},
		map[string]any{"kind": "runner-trace", "path": "/tmp/surface-artifacts/surface-runner-trace.json", "sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "size": 32000},
	}
	report["components"] = []any{
		componentMap("SurfaceReleaseFormApp", "examples.surface_release_form.SurfaceReleaseFormApp", "", RectReport{X: 0, Y: 0, W: 560, H: 420}, map[string]string{"focused_id": "7", "save_count": "1", "reset_count": "1", "status_code": "2", "width": "560", "height": "420", "accessibility_role": "none"}),
		componentMap("Panel", "lib.core.widgets.Panel", "SurfaceReleaseFormApp", RectReport{X: 0, Y: 0, W: 560, H: 420}, map[string]string{"padding": "16", "accessibility_role": "none"}),
		componentMap("Stack", "lib.core.widgets.Stack", "Panel", RectReport{X: 16, Y: 16, W: 528, H: 396}, map[string]string{"child_count": "1", "accessibility_role": "none"}),
		componentMap("Column", "lib.core.widgets.Column", "Stack", RectReport{X: 24, Y: 24, W: 512, H: 388}, map[string]string{"child_count": "9", "accessibility_role": "none"}),
		componentMap("TitleText", "lib.core.widgets.Text", "Column", RectReport{X: 32, Y: 32, W: 496, H: 28}, map[string]string{"role": "title", "text_len": "18", "accessibility_role": "label"}),
		componentMap("DescriptionText", "lib.core.widgets.Text", "Column", RectReport{X: 32, Y: 68, W: 496, H: 28}, map[string]string{"role": "description", "text_len": "24", "accessibility_role": "label"}),
		componentMap("NameLabel", "lib.core.widgets.Label", "Column", RectReport{X: 32, Y: 104, W: 496, H: 24}, map[string]string{"role": "label", "text_len": "4", "labelled_for": "7", "accessibility_role": "label"}),
		componentMap("NameTextBox", "lib.core.widgets.TextBox", "Column", RectReport{X: 32, Y: 132, W: 496, H: 44}, map[string]string{"focused": "true", "buffer": "Ada", "text_len": "3", "caret": "3", "accessibility_role": "label"}),
		componentMap("EmailLabel", "lib.core.widgets.Label", "Column", RectReport{X: 32, Y: 184, W: 496, H: 24}, map[string]string{"role": "label", "text_len": "5", "labelled_for": "9", "accessibility_role": "label"}),
		componentMap("EmailTextBox", "lib.core.widgets.TextBox", "Column", RectReport{X: 32, Y: 212, W: 496, H: 44}, map[string]string{"focused": "false", "buffer": "tetra", "text_len": "5", "caret": "5", "accessibility_role": "label"}),
		componentMap("SubscribeCheckbox", "lib.core.widgets.Checkbox", "Column", RectReport{X: 32, Y: 264, W: 496, H: 32}, map[string]string{"focused": "false", "checked": "true", "toggle_count": "1", "accessibility_role": "button"}),
		componentMap("TermsScroll", "lib.core.widgets.Scroll", "Column", RectReport{X: 32, Y: 304, W: 496, H: 48}, map[string]string{"offset_y": "16", "content_h": "120", "accessibility_role": "none"}),
		componentMap("TermsText", "lib.core.widgets.Text", "TermsScroll", RectReport{X: 36, Y: 308, W: 488, H: 24}, map[string]string{"role": "description", "text_len": "48", "accessibility_role": "label"}),
		componentMap("ButtonRow", "lib.core.widgets.Row", "Column", RectReport{X: 32, Y: 360, W: 496, H: 44}, map[string]string{"child_count": "4", "accessibility_role": "none"}),
		componentMap("SaveButton", "lib.core.widgets.Button", "ButtonRow", RectReport{X: 32, Y: 360, W: 132, H: 44}, map[string]string{"focused": "false", "press_count": "1", "action": "save", "accessibility_role": "button"}),
		componentMap("ResetButton", "lib.core.widgets.Button", "ButtonRow", RectReport{X: 176, Y: 360, W: 132, H: 44}, map[string]string{"focused": "false", "press_count": "1", "action": "reset", "accessibility_role": "button"}),
		componentMap("Spacer", "lib.core.widgets.Spacer", "ButtonRow", RectReport{X: 320, Y: 360, W: 16, H: 44}, map[string]string{"min_w": "16", "min_h": "44", "accessibility_role": "none"}),
		componentMap("StatusText", "lib.core.widgets.StatusText", "ButtonRow", RectReport{X: 344, Y: 360, W: 184, H: 44}, map[string]string{"role": "status", "status_code": "2", "text_len": "6", "accessibility_role": "label"}),
	}
	report["component_tree"] = map[string]any{
		"schema":        "tetra.surface.component-tree.v1",
		"dynamic_level": "production-widgets-v1",
		"root_id":       0,
		"node_count":    18,
		"focused_id":    7,
		"nodes": []any{
			treeNodeMap(0, "SurfaceReleaseFormApp", "root", -1, 0, 1, 1, false, RectReport{X: 0, Y: 0, W: 560, H: 420}),
			treeNodeMap(1, "Panel", "panel", 0, 0, 2, 1, false, RectReport{X: 0, Y: 0, W: 560, H: 420}),
			treeNodeMap(2, "Stack", "stack", 1, 0, 3, 1, false, RectReport{X: 16, Y: 16, W: 528, H: 396}),
			treeNodeMap(3, "Column", "column", 2, 0, 4, 9, false, RectReport{X: 24, Y: 24, W: 512, H: 388}),
			treeNodeMap(4, "TitleText", "text", 3, 0, -1, 0, false, RectReport{X: 32, Y: 32, W: 496, H: 28}),
			treeNodeMap(5, "DescriptionText", "text", 3, 1, -1, 0, false, RectReport{X: 32, Y: 68, W: 496, H: 28}),
			treeNodeMap(6, "NameLabel", "label", 3, 2, -1, 0, false, RectReport{X: 32, Y: 104, W: 496, H: 24}),
			treeNodeMap(7, "NameTextBox", "textbox", 3, 3, -1, 0, true, RectReport{X: 32, Y: 132, W: 496, H: 44}),
			treeNodeMap(8, "EmailLabel", "label", 3, 4, -1, 0, false, RectReport{X: 32, Y: 184, W: 496, H: 24}),
			treeNodeMap(9, "EmailTextBox", "textbox", 3, 5, -1, 0, true, RectReport{X: 32, Y: 212, W: 496, H: 44}),
			treeNodeMap(10, "SubscribeCheckbox", "checkbox", 3, 6, -1, 0, true, RectReport{X: 32, Y: 264, W: 496, H: 32}),
			treeNodeMap(11, "TermsScroll", "scroll", 3, 7, 12, 1, false, RectReport{X: 32, Y: 304, W: 496, H: 48}),
			treeNodeMap(12, "TermsText", "text", 11, 0, -1, 0, false, RectReport{X: 36, Y: 308, W: 488, H: 24}),
			treeNodeMap(13, "ButtonRow", "row", 3, 8, 14, 4, false, RectReport{X: 32, Y: 360, W: 496, H: 44}),
			treeNodeMap(14, "SaveButton", "button", 13, 0, -1, 0, true, RectReport{X: 32, Y: 360, W: 132, H: 44}),
			treeNodeMap(15, "ResetButton", "button", 13, 1, -1, 0, true, RectReport{X: 176, Y: 360, W: 132, H: 44}),
			treeNodeMap(16, "Spacer", "spacer", 13, 2, -1, 0, false, RectReport{X: 320, Y: 360, W: 16, H: 44}),
			treeNodeMap(17, "StatusText", "status", 13, 3, -1, 0, false, RectReport{X: 344, Y: 360, W: 184, H: 44}),
		},
		"layout_passes": []any{
			map[string]any{"component_id": 7, "pass": "initial", "bounds": rectMap(RectReport{X: 32, Y: 132, W: 320, H: 44}), "measured": map[string]any{"w": 320, "h": 44}},
			map[string]any{"component_id": 9, "pass": "initial", "bounds": rectMap(RectReport{X: 32, Y: 212, W: 320, H: 44}), "measured": map[string]any{"w": 320, "h": 44}},
			map[string]any{"component_id": 11, "pass": "scroll", "bounds": rectMap(RectReport{X: 32, Y: 304, W: 496, H: 48}), "measured": map[string]any{"w": 496, "h": 120}},
			map[string]any{"component_id": 7, "pass": "resize", "bounds": rectMap(RectReport{X: 32, Y: 132, W: 496, H: 44}), "measured": map[string]any{"w": 496, "h": 44}},
			map[string]any{"component_id": 9, "pass": "resize", "bounds": rectMap(RectReport{X: 32, Y: 212, W: 496, H: 44}), "measured": map[string]any{"w": 496, "h": 44}},
			map[string]any{"component_id": 17, "pass": "status-update", "bounds": rectMap(RectReport{X: 344, Y: 360, W: 184, H: 44}), "measured": map[string]any{"w": 184, "h": 44}},
		},
		"draw_order":  []any{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17},
		"focus_order": []any{7, 9, 10, 14, 15},
		"dispatch_paths": []any{
			map[string]any{"event": "click", "target_id": 7, "x": 48, "y": 148, "path": []any{0, 1, 2, 3, 7}},
			map[string]any{"event": "click", "target_id": 9, "x": 48, "y": 228, "path": []any{0, 1, 2, 3, 9}},
			map[string]any{"event": "click", "target_id": 10, "x": 48, "y": 280, "path": []any{0, 1, 2, 3, 10}},
			map[string]any{"event": "key", "target_id": 14, "x": 48, "y": 376, "path": []any{0, 1, 2, 3, 13, 14}},
			map[string]any{"event": "key", "target_id": 15, "x": 192, "y": 376, "path": []any{0, 1, 2, 3, 13, 15}},
		},
	}
	report["component_tree_api"] = map[string]any{
		"schema":             "tetra.surface.component-tree-api.v1",
		"api_level":          "builder-layout-dispatch-v1",
		"source":             "examples/surface_release_form.tetra",
		"manual_bookkeeping": false,
		"builder": map[string]any{
			"root_created_by":     "tree_add_root",
			"children_created_by": "tree_add_child",
			"node_count":          18,
			"capacity":            32,
			"overflow_checked":    true,
		},
		"invariants": map[string]any{
			"tree_validate_ran":          true,
			"tree_validate_status":       0,
			"parent_child_links_checked": true,
			"child_indices_checked":      true,
			"child_count_checked":        true,
			"first_child_checked":        true,
		},
		"layout_helpers": []any{
			map[string]any{"helper": "widgets.panel_content_rect", "target": "Panel", "pass": "initial", "changed_bounds": true},
			map[string]any{"helper": "widgets.stack_layout", "target": "Stack", "pass": "initial", "changed_bounds": true},
			map[string]any{"helper": "widgets.column_layout", "target": "Column", "pass": "initial", "changed_bounds": true},
			map[string]any{"helper": "widgets.scroll_set_offset", "target": "TermsScroll", "pass": "scroll", "changed_bounds": true},
			map[string]any{"helper": "widgets.row_layout", "target": "ButtonRow", "pass": "initial", "changed_bounds": true},
			map[string]any{"helper": "widgets.column_layout", "target": "Column", "pass": "resize", "changed_bounds": true},
		},
		"focus_helpers": []any{
			map[string]any{"helper": "tree_focus_next", "before": "NameTextBox", "after": "EmailTextBox"},
			map[string]any{"helper": "tree_focus_next", "before": "EmailTextBox", "after": "SubscribeCheckbox"},
			map[string]any{"helper": "tree_focus_next", "before": "SubscribeCheckbox", "after": "SaveButton"},
			map[string]any{"helper": "tree_focus_next", "before": "SaveButton", "after": "ResetButton"},
			map[string]any{"helper": "tree_focus_next", "before": "ResetButton", "after": "NameTextBox"},
		},
		"hit_tests": []any{
			map[string]any{"helper": "widgets.hit_test_release_form", "x": 48, "y": 148, "target": "NameTextBox", "path": []any{0, 1, 2, 3, 7}},
			map[string]any{"helper": "widgets.hit_test_release_form", "x": 48, "y": 228, "target": "EmailTextBox", "path": []any{0, 1, 2, 3, 9}},
			map[string]any{"helper": "widgets.hit_test_release_form", "x": 48, "y": 280, "target": "SubscribeCheckbox", "path": []any{0, 1, 2, 3, 10}},
			map[string]any{"helper": "widgets.hit_test_release_form", "x": 48, "y": 320, "target": "TermsScroll", "path": []any{0, 1, 2, 3, 11}},
			map[string]any{"helper": "widgets.hit_test_release_form", "x": 192, "y": 376, "target": "ResetButton", "path": []any{0, 1, 2, 3, 13, 15}},
		},
		"dispatch_paths": []any{
			map[string]any{"helper": "tree_build_dispatch_path", "target": "NameTextBox", "path": []any{0, 1, 2, 3, 7}},
			map[string]any{"helper": "tree_build_dispatch_path", "target": "EmailTextBox", "path": []any{0, 1, 2, 3, 9}},
			map[string]any{"helper": "tree_build_dispatch_path", "target": "SubscribeCheckbox", "path": []any{0, 1, 2, 3, 10}},
			map[string]any{"helper": "tree_build_dispatch_path", "target": "TermsScroll", "path": []any{0, 1, 2, 3, 11}},
			map[string]any{"helper": "tree_build_dispatch_path", "target": "SaveButton", "path": []any{0, 1, 2, 3, 13, 14}},
			map[string]any{"helper": "tree_build_dispatch_path", "target": "ResetButton", "path": []any{0, 1, 2, 3, 13, 15}},
		},
	}
	report["toolkit"] = map[string]any{
		"schema":                       "tetra.surface.toolkit.v1",
		"toolkit_level":                "production-widgets-v1",
		"release_scope":                "surface-v1-linux-web",
		"source":                       "examples/surface_release_form.tetra",
		"sources":                      []any{"examples/surface_release_form.tetra", "examples/surface_toolkit_form.tetra", "examples/surface_toolkit_settings.tetra"},
		"module":                       "lib.core.widgets",
		"style_module":                 "lib.core.style",
		"experimental":                 false,
		"production_claim":             true,
		"uses_component_tree_api":      true,
		"manual_bookkeeping":           false,
		"demo_specific_widget_structs": false,
		"no_magic_widgets":             true,
		"no_platform_widgets":          true,
		"no_dom_ui":                    true,
		"no_user_js":                   true,
		"example_count":                3,
		"text_box_count":               2,
		"button_count":                 2,
		"multi_textbox_evidence":       true,
		"multi_form_evidence":          true,
		"widget_set":                   []any{"Text", "Label", "StatusText", "Button", "TextBox", "Checkbox", "Row", "Column", "Panel", "Stack", "Scroll", "Spacer"},
		"state_set":                    []any{"normal", "focused", "hovered", "pressed", "disabled", "error"},
		"layout_features":              []any{"padding", "margin", "spacing", "min_size", "max_size", "fill", "scroll_offset"},
		"theme":                        true,
		"safe_text_storage":            true,
		"widgets": []any{
			toolkitWidgetMap("Panel", "Panel", 1, "", true),
			toolkitWidgetMap("Stack", "Stack", 2, "", true),
			toolkitWidgetMap("Column", "Column", 3, "", true),
			toolkitWidgetMap("TitleText", "Text", 4, "label", true),
			toolkitWidgetMap("DescriptionText", "Text", 5, "description", true),
			toolkitWidgetMap("NameLabel", "Label", 6, "label", true),
			toolkitWidgetMap("NameTextBox", "TextBox", 7, "", true),
			toolkitWidgetMap("EmailLabel", "Label", 8, "label", true),
			toolkitWidgetMap("EmailTextBox", "TextBox", 9, "", true),
			toolkitWidgetMap("SubscribeCheckbox", "Checkbox", 10, "", true),
			toolkitWidgetMap("TermsScroll", "Scroll", 11, "", true),
			toolkitWidgetMap("TermsText", "Text", 12, "description", true),
			toolkitWidgetMap("ButtonRow", "Row", 13, "", true),
			toolkitWidgetMap("SaveButton", "Button", 14, "save", true),
			toolkitWidgetMap("ResetButton", "Button", 15, "reset", true),
			toolkitWidgetMap("Spacer", "Spacer", 16, "", true),
			toolkitWidgetMap("StatusText", "StatusText", 17, "status", true),
		},
		"reusable_sources": []any{
			"lib/core/widgets.tetra:panel_init",
			"lib/core/widgets.tetra:column_init",
			"lib/core/widgets.tetra:text_init",
			"lib/core/widgets.tetra:label_init",
			"lib/core/widgets.tetra:status_text_init",
			"lib/core/widgets.tetra:textbox_init",
			"lib/core/widgets.tetra:checkbox_init",
			"lib/core/widgets.tetra:checkbox_toggle",
			"lib/core/widgets.tetra:row_init",
			"lib/core/widgets.tetra:stack_init",
			"lib/core/widgets.tetra:scroll_init",
			"lib/core/widgets.tetra:scroll_set_offset",
			"lib/core/widgets.tetra:spacer_init",
			"lib/core/widgets.tetra:button_init",
			"lib/core/widgets.tetra:hit_test_release_form",
			"lib/core/style.tetra:default_theme",
			"lib/core/style.tetra:style_for_state",
		},
	}
	report["events"] = []any{
		eventMap(1, "mouse_up", "NameTextBox", []any{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "NameTextBox"}, 48, 148, 0, 560, 420, map[string]string{"SurfaceReleaseFormApp.focused_id": "-1", "NameTextBox.focused": "false"}, map[string]string{"SurfaceReleaseFormApp.focused_id": "7", "NameTextBox.focused": "true"}),
		textEventMap(2, "NameTextBox", []any{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "NameTextBox"}, 3, "416461", 560, 420, map[string]string{"NameTextBox.buffer": "", "EmailTextBox.buffer": ""}, map[string]string{"NameTextBox.buffer": "Ada", "EmailTextBox.buffer": ""}),
		keyEventMap(3, "SurfaceReleaseFormApp", []any{"SurfaceReleaseFormApp"}, 9, 560, 420, map[string]string{"SurfaceReleaseFormApp.focused_id": "7"}, map[string]string{"SurfaceReleaseFormApp.focused_id": "9"}),
		textEventMap(4, "EmailTextBox", []any{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "EmailTextBox"}, 5, "7465747261", 560, 420, map[string]string{"EmailTextBox.buffer": "", "NameTextBox.buffer": "Ada"}, map[string]string{"EmailTextBox.buffer": "tetra", "NameTextBox.buffer": "Ada"}),
		keyEventMap(5, "SurfaceReleaseFormApp", []any{"SurfaceReleaseFormApp"}, 9, 560, 420, map[string]string{"SurfaceReleaseFormApp.focused_id": "9"}, map[string]string{"SurfaceReleaseFormApp.focused_id": "10"}),
		keyEventMap(6, "SubscribeCheckbox", []any{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "SubscribeCheckbox"}, 32, 560, 420, map[string]string{"SubscribeCheckbox.checked": "false", "SubscribeCheckbox.toggle_count": "0"}, map[string]string{"SubscribeCheckbox.checked": "true", "SubscribeCheckbox.toggle_count": "1"}),
		eventMap(7, "scroll", "TermsScroll", []any{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "TermsScroll"}, 48, 320, 0, 560, 420, map[string]string{"TermsScroll.offset_y": "0"}, map[string]string{"TermsScroll.offset_y": "16"}),
		keyEventMap(8, "SurfaceReleaseFormApp", []any{"SurfaceReleaseFormApp"}, 9, 560, 420, map[string]string{"SurfaceReleaseFormApp.focused_id": "10"}, map[string]string{"SurfaceReleaseFormApp.focused_id": "14"}),
		keyEventMap(9, "SaveButton", []any{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "ButtonRow", "SaveButton"}, 32, 560, 420, map[string]string{"SurfaceReleaseFormApp.save_count": "0", "StatusText.status_code": "0"}, map[string]string{"SurfaceReleaseFormApp.save_count": "1", "StatusText.status_code": "1"}),
		keyEventMap(10, "SurfaceReleaseFormApp", []any{"SurfaceReleaseFormApp"}, 9, 560, 420, map[string]string{"SurfaceReleaseFormApp.focused_id": "14"}, map[string]string{"SurfaceReleaseFormApp.focused_id": "15"}),
		keyEventMap(11, "ResetButton", []any{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "ButtonRow", "ResetButton"}, 13, 560, 420, map[string]string{"SurfaceReleaseFormApp.reset_count": "0", "StatusText.status_code": "1", "NameTextBox.buffer": "Ada", "EmailTextBox.buffer": "tetra"}, map[string]string{"SurfaceReleaseFormApp.reset_count": "1", "StatusText.status_code": "2", "NameTextBox.buffer": "", "EmailTextBox.buffer": ""}),
		keyEventMap(12, "SurfaceReleaseFormApp", []any{"SurfaceReleaseFormApp"}, 9, 560, 420, map[string]string{"SurfaceReleaseFormApp.focused_id": "15"}, map[string]string{"SurfaceReleaseFormApp.focused_id": "7"}),
		resizeEventMap(13, "SurfaceReleaseFormApp", []any{"SurfaceReleaseFormApp"}, 560, 420, map[string]string{"SurfaceReleaseFormApp.focused_id": "7", "NameTextBox.bounds.w": "320", "EmailTextBox.bounds.w": "320"}, map[string]string{"SurfaceReleaseFormApp.focused_id": "7", "NameTextBox.bounds.w": "496", "EmailTextBox.bounds.w": "496"}),
	}
	report["frames"] = []any{
		map[string]any{"order": 1, "width": 320, "height": 240, "stride": 1280, "checksum": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "presented": true},
		map[string]any{"order": 2, "width": 560, "height": 420, "stride": 2240, "checksum": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "presented": true},
		map[string]any{"order": 3, "width": 560, "height": 420, "stride": 2240, "checksum": "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "presented": true},
		map[string]any{"order": 4, "width": 560, "height": 420, "stride": 2240, "checksum": "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", "presented": true},
		map[string]any{"order": 5, "width": 560, "height": 420, "stride": 2240, "checksum": "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", "presented": true},
	}
	report["state_transitions"] = []any{
		map[string]any{"order": 1, "component": "SurfaceReleaseFormApp", "field": "focused_id", "before": "-1", "after": "7", "cause": "mouse_up"},
		map[string]any{"order": 2, "component": "NameTextBox", "field": "buffer", "before": "", "after": "Ada", "cause": "text_input"},
		map[string]any{"order": 3, "component": "EmailTextBox", "field": "buffer", "before": "", "after": "tetra", "cause": "text_input"},
		map[string]any{"order": 4, "component": "SubscribeCheckbox", "field": "checked", "before": "false", "after": "true", "cause": "key_down"},
		map[string]any{"order": 5, "component": "TermsScroll", "field": "offset_y", "before": "0", "after": "16", "cause": "scroll"},
		map[string]any{"order": 6, "component": "SurfaceReleaseFormApp", "field": "save_count", "before": "0", "after": "1", "cause": "key_down"},
		map[string]any{"order": 7, "component": "StatusText", "field": "status_code", "before": "0", "after": "1", "cause": "save"},
		map[string]any{"order": 8, "component": "NameTextBox", "field": "buffer", "before": "Ada", "after": "", "cause": "reset"},
		map[string]any{"order": 9, "component": "EmailTextBox", "field": "buffer", "before": "tetra", "after": "", "cause": "reset"},
		map[string]any{"order": 10, "component": "SurfaceReleaseFormApp", "field": "reset_count", "before": "0", "after": "1", "cause": "key_down"},
		map[string]any{"order": 11, "component": "StatusText", "field": "status_code", "before": "1", "after": "2", "cause": "reset"},
		map[string]any{"order": 12, "component": "SurfaceReleaseFormApp", "field": "focused_id", "before": "15", "after": "7", "cause": "tab"},
		map[string]any{"order": 13, "component": "SurfaceReleaseFormApp", "field": "NameTextBox.bounds.w", "before": "320", "after": "496", "cause": "resize"},
		map[string]any{"order": 14, "component": "SurfaceReleaseFormApp", "field": "EmailTextBox.bounds.w", "before": "320", "after": "496", "cause": "resize"},
	}
	report["cases"] = append(report["cases"].([]any),
		map[string]any{"name": "production toolkit required widget set", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "production toolkit style module default theme", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "production toolkit style states normal focused hovered pressed disabled error", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "production toolkit Text Label StatusText evidence", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "production toolkit Button TextBox Checkbox evidence", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "production toolkit Row Column Panel Stack Scroll Spacer layout", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "production toolkit component tree api reuse", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "production toolkit TextBox focus input editing", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "production toolkit Checkbox toggle routed", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "production toolkit Scroll offset routed", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "production toolkit Save action routed", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "production toolkit Reset action routed", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "production toolkit StatusText updates", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "production toolkit safe text storage", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "production toolkit no demo-local widget structs", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "production toolkit browser host separation", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "production toolkit rendered frame update", "kind": "positive", "ran": true, "pass": true},
	)
	if mutate != nil {
		mutate(report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal production toolkit report: %v", err)
	}
	return raw
}
