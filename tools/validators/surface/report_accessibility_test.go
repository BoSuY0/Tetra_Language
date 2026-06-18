package surface

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateSurfaceAccessibilityMetadataTreeReport(t *testing.T) {
	raw := validHeadlessAccessibilityMetadataSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceAccessibilityRejectsMissingTree(t *testing.T) {
	raw := validHeadlessAccessibilityMetadataSurfaceReportJSON(t, func(report map[string]any) {
		delete(report, "accessibility_tree")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected accessibility report without accessibility_tree to fail")
	}
	if !strings.Contains(err.Error(), "accessibility_tree") {
		t.Fatalf("error = %v, want accessibility_tree diagnostic", err)
	}
}
func TestValidateSurfaceAccessibilityRejectsClaimsAndManualBookkeeping(t *testing.T) {
	for _, tc := range []struct {
		name  string
		field string
		want  string
	}{
		{name: "production", field: "production_claim", want: "production"},
		{name: "platform", field: "platform_host_integration", want: "platform_host_integration"},
		{name: "dom", field: "dom_aria_integration", want: "dom_aria_integration"},
		{name: "manual", field: "manual_bookkeeping", want: "manual_bookkeeping"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessAccessibilityMetadataSurfaceReportJSON(t, func(report map[string]any) {
				a11y := report["accessibility_tree"].(map[string]any)
				a11y[tc.field] = true
			})
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected accessibility report with %s=true to fail", tc.field)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateSurfaceReleaseAccessibilityPlatformBridgeReport(t *testing.T) {
	raw := validLinuxReleaseAccessibilitySurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceReleaseAccessibilityRejectsMissingBridgeEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "metadata tree false",
			mutate: func(report map[string]any) {
				report["accessibility_tree"].(map[string]any)["metadata_tree"] = false
			},
			want: "metadata_tree",
		},
		{
			name: "platform export false",
			mutate: func(report map[string]any) {
				report["accessibility_tree"].(map[string]any)["platform_export"] = false
			},
			want: "platform_export",
		},
		{
			name: "linux probe false",
			mutate: func(report map[string]any) {
				report["accessibility_tree"].(map[string]any)["linux_platform_probe"] = false
			},
			want: "linux_platform_probe",
		},
		{
			name: "missing bridge evidence name",
			mutate: func(report map[string]any) {
				report["accessibility_tree"].(map[string]any)["screen_reader_evidence"] = "full-screen-reader-support"
			},
			want: "screen_reader_evidence",
		},
		{
			name: "node only browser evidence",
			mutate: func(report map[string]any) {
				report["target"] = "wasm32-web"
				report["runtime"] = "surface-wasm32-web"
				report["host_evidence"].(map[string]any)["level"] = "wasm32-web-compiler-owned-loader"
			},
			want: "browser",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validLinuxReleaseAccessibilitySurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected release accessibility report to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateSurfaceAccessibilityRejectsNodeRelationshipAndOrderMismatches(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "unknown role",
			mutate: func(report map[string]any) {
				nodes := report["accessibility_tree"].(map[string]any)["nodes"].([]any)
				nodes[5].(map[string]any)["role"] = "slider"
			},
			want: "unknown role",
		},
		{
			name: "duplicate node id",
			mutate: func(report map[string]any) {
				nodes := report["accessibility_tree"].(map[string]any)["nodes"].([]any)
				nodes[7].(map[string]any)["id"] = 5
			},
			want: "duplicate",
		},
		{
			name: "unknown component",
			mutate: func(report map[string]any) {
				nodes := report["accessibility_tree"].(map[string]any)["nodes"].([]any)
				nodes[5].(map[string]any)["component_id"] = 99
			},
			want: "component_id",
		},
		{
			name: "bounds mismatch",
			mutate: func(report map[string]any) {
				nodes := report["accessibility_tree"].(map[string]any)["nodes"].([]any)
				nodes[5].(map[string]any)["bounds"] = rectMap(RectReport{X: 1, Y: 2, W: 3, H: 4})
			},
			want: "bounds",
		},
		{
			name: "missing label",
			mutate: func(report map[string]any) {
				a11y := report["accessibility_tree"].(map[string]any)
				a11y["relationships"] = []any{
					map[string]any{"kind": "label_for", "from": "NameLabel", "to": "NameTextBox"},
					map[string]any{"kind": "labelled_by", "from": "NameTextBox", "to": "NameLabel"},
				}
			},
			want: "EmailLabel",
		},
		{
			name: "focus order",
			mutate: func(report map[string]any) {
				report["accessibility_tree"].(map[string]any)["focus_order"] = []any{"NameTextBox", "EmailTextBox", "SaveButton"}
			},
			want: "focus_order",
		},
		{
			name: "reading order",
			mutate: func(report map[string]any) {
				report["accessibility_tree"].(map[string]any)["reading_order"] = []any{"TitleText", "NameTextBox", "NameLabel", "EmailLabel", "EmailTextBox", "SaveButton", "ResetButton", "StatusText"}
			},
			want: "reading_order",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessAccessibilityMetadataSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected accessibility %s mismatch to fail", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateSurfaceAccessibilityRejectsSnapshotMismatches(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "two focused nodes",
			mutate: func(report map[string]any) {
				nodes := report["accessibility_tree"].(map[string]any)["nodes"].([]any)
				nodes[5].(map[string]any)["focused"] = true
				nodes[7].(map[string]any)["focused"] = true
			},
			want: "focused",
		},
		{
			name: "email value while wrong focus",
			mutate: func(report map[string]any) {
				for _, rawSnapshot := range report["accessibility_tree"].(map[string]any)["snapshots"].([]any) {
					snapshot := rawSnapshot.(map[string]any)
					if snapshot["name"] == "after_email_text" {
						snapshot["focused"] = "NameTextBox"
					}
				}
			},
			want: "after_email_text",
		},
		{
			name: "status unchanged after save",
			mutate: func(report map[string]any) {
				for _, rawSnapshot := range report["accessibility_tree"].(map[string]any)["snapshots"].([]any) {
					snapshot := rawSnapshot.(map[string]any)
					if snapshot["name"] == "after_save" {
						snapshot["status_value"] = "idle"
					}
				}
			},
			want: "after_save",
		},
		{
			name: "metadata checksum unchanged",
			mutate: func(report map[string]any) {
				snapshots := report["accessibility_tree"].(map[string]any)["snapshots"].([]any)
				snapshots[2].(map[string]any)["metadata_checksum"] = snapshots[1].(map[string]any)["metadata_checksum"]
			},
			want: "metadata_checksum",
		},
		{
			name: "bounds checksum unchanged",
			mutate: func(report map[string]any) {
				snapshots := report["accessibility_tree"].(map[string]any)["snapshots"].([]any)
				snapshots[len(snapshots)-1].(map[string]any)["bounds_checksum"] = snapshots[len(snapshots)-2].(map[string]any)["bounds_checksum"]
			},
			want: "bounds_checksum",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessAccessibilityMetadataSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected accessibility %s mismatch to fail", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func TestValidateSurfaceAccessibilityRejectsNodeOnlyBrowserAndLegacySidecarEvidence(t *testing.T) {
	raw := validHeadlessAccessibilityMetadataSurfaceReportJSON(t, func(report map[string]any) {
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
		t.Fatalf("expected accessibility Node-only browser evidence to fail")
	}
	if !strings.Contains(err.Error(), "browser") && !strings.Contains(err.Error(), "Node") {
		t.Fatalf("error = %v, want browser/Node diagnostic", err)
	}

	raw = validHeadlessAccessibilityMetadataSurfaceReportJSON(t, func(report map[string]any) {
		artifacts := report["artifacts"].([]any)
		report["artifacts"] = append(artifacts, map[string]any{"kind": "legacy-ui-sidecar", "path": "/tmp/surface-artifacts/accessibility.ui.json", "sha256": "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", "size": 1})
	})
	err = ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected accessibility legacy sidecar evidence to fail")
	}
	if !strings.Contains(err.Error(), ".ui.json") && !strings.Contains(err.Error(), "legacy") {
		t.Fatalf("error = %v, want legacy sidecar diagnostic", err)
	}
}
func validHeadlessAccessibilityMetadataSurfaceReportJSON(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessToolkitReuseSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode base accessibility report: %v", err)
	}
	report["source"] = "examples/surface_accessibility_settings.tetra"
	report["processes"] = []any{
		map[string]any{"name": "tetra build", "kind": "build", "path": "tetra build --target linux-x64 examples/surface_accessibility_settings.tetra -o /tmp/surface-artifacts/surface-accessibility-settings", "ran": true, "pass": true, "exit_code": 0},
		map[string]any{"name": "surface component app", "kind": "app", "path": "/tmp/surface-artifacts/surface-accessibility-settings", "ran": true, "pass": true, "exit_code": 1, "expected_exit_code": 1},
		map[string]any{"name": "surface headless runtime", "kind": "runtime", "path": "tools/cmd/surface-runtime-smoke", "ran": true, "pass": true, "exit_code": 0},
	}
	report["artifacts"] = []any{
		map[string]any{"kind": "component-app", "path": "/tmp/surface-artifacts/surface-accessibility-settings", "sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size": 90001},
		map[string]any{"kind": "runner-trace", "path": "/tmp/surface-artifacts/surface-runner-trace.json", "sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "size": 24000},
	}
	report["components"] = []any{
		componentMap("AccessibilitySettingsApp", "examples.surface_accessibility_settings.AccessibilitySettingsApp", "", RectReport{X: 0, Y: 0, W: 480, H: 320}, map[string]string{"focused_id": "5", "save_count": "1", "reset_count": "1", "status_code": "2", "width": "480", "height": "320", "accessibility_role": "root"}),
		componentMap("Panel", "lib.core.widgets.Panel", "AccessibilitySettingsApp", RectReport{X: 0, Y: 0, W: 480, H: 320}, map[string]string{"padding": "12", "accessibility_role": "panel"}),
		componentMap("Column", "lib.core.widgets.Column", "Panel", RectReport{X: 12, Y: 12, W: 456, H: 296}, map[string]string{"child_count": "7", "accessibility_role": "column"}),
		componentMap("TitleText", "lib.core.widgets.Text", "Column", RectReport{X: 20, Y: 20, W: 440, H: 24}, map[string]string{"role": "text", "text_len": "8", "accessibility_role": "text"}),
		componentMap("NameLabel", "lib.core.widgets.Text", "Column", RectReport{X: 20, Y: 52, W: 440, H: 24}, map[string]string{"role": "label", "text_len": "4", "accessibility_role": "label"}),
		componentMap("NameTextBox", "lib.core.widgets.TextBox", "Column", RectReport{X: 20, Y: 84, W: 440, H: 44}, map[string]string{"focused": "true", "buffer": "", "text_len": "0", "caret": "0", "accessibility_role": "textbox"}),
		componentMap("EmailLabel", "lib.core.widgets.Text", "Column", RectReport{X: 20, Y: 136, W: 440, H: 24}, map[string]string{"role": "label", "text_len": "5", "accessibility_role": "label"}),
		componentMap("EmailTextBox", "lib.core.widgets.TextBox", "Column", RectReport{X: 20, Y: 168, W: 440, H: 44}, map[string]string{"focused": "false", "buffer": "", "text_len": "0", "caret": "0", "accessibility_role": "textbox"}),
		componentMap("ButtonRow", "lib.core.widgets.Row", "Column", RectReport{X: 20, Y: 224, W: 440, H: 44}, map[string]string{"child_count": "2", "accessibility_role": "row"}),
		componentMap("SaveButton", "lib.core.widgets.Button", "ButtonRow", RectReport{X: 20, Y: 224, W: 132, H: 44}, map[string]string{"focused": "false", "press_count": "1", "action": "save", "accessibility_role": "button"}),
		componentMap("ResetButton", "lib.core.widgets.Button", "ButtonRow", RectReport{X: 164, Y: 224, W: 132, H: 44}, map[string]string{"focused": "false", "press_count": "1", "action": "reset", "accessibility_role": "button"}),
		componentMap("StatusText", "lib.core.widgets.Text", "Column", RectReport{X: 20, Y: 280, W: 440, H: 24}, map[string]string{"role": "status", "status_code": "2", "accessibility_role": "status"}),
	}
	report["component_tree"] = accessibilityComponentTreeMap("accessibility-metadata-tree-v1", "AccessibilitySettingsApp")
	report["component_tree_api"] = accessibilityComponentTreeAPIMap()
	report["toolkit"] = accessibilityToolkitMap()
	report["accessibility_tree"] = accessibilityTreeMap()
	report["events"] = []any{
		eventMap(1, "mouse_up", "NameTextBox", []any{"AccessibilitySettingsApp", "Panel", "Column", "NameTextBox"}, 40, 100, 0, 320, 240, map[string]string{"AccessibilitySettingsApp.focused_id": "-1", "NameTextBox.focused": "false"}, map[string]string{"AccessibilitySettingsApp.focused_id": "5", "NameTextBox.focused": "true"}),
		textEventMap(2, "NameTextBox", []any{"AccessibilitySettingsApp", "Panel", "Column", "NameTextBox"}, 3, "416461", 320, 240, map[string]string{"NameTextBox.buffer": "", "NameTextBox.caret": "0", "EmailTextBox.buffer": ""}, map[string]string{"NameTextBox.buffer": "Ada", "NameTextBox.caret": "3", "EmailTextBox.buffer": ""}),
		keyEventMap(3, "AccessibilitySettingsApp", []any{"AccessibilitySettingsApp"}, 9, 320, 240, map[string]string{"AccessibilitySettingsApp.focused_id": "5"}, map[string]string{"AccessibilitySettingsApp.focused_id": "7"}),
		textEventMap(4, "EmailTextBox", []any{"AccessibilitySettingsApp", "Panel", "Column", "EmailTextBox"}, 5, "7465747261", 320, 240, map[string]string{"EmailTextBox.buffer": "", "NameTextBox.buffer": "Ada"}, map[string]string{"EmailTextBox.buffer": "tetra", "NameTextBox.buffer": "Ada"}),
		keyEventMap(5, "AccessibilitySettingsApp", []any{"AccessibilitySettingsApp"}, 9, 320, 240, map[string]string{"AccessibilitySettingsApp.focused_id": "7"}, map[string]string{"AccessibilitySettingsApp.focused_id": "9"}),
		keyEventMap(6, "SaveButton", []any{"AccessibilitySettingsApp", "Panel", "Column", "ButtonRow", "SaveButton"}, 32, 320, 240, map[string]string{"AccessibilitySettingsApp.focused_id": "9", "AccessibilitySettingsApp.save_count": "0", "StatusText.status_code": "0"}, map[string]string{"AccessibilitySettingsApp.focused_id": "9", "AccessibilitySettingsApp.save_count": "1", "StatusText.status_code": "1"}),
		keyEventMap(7, "AccessibilitySettingsApp", []any{"AccessibilitySettingsApp"}, 9, 320, 240, map[string]string{"AccessibilitySettingsApp.focused_id": "9"}, map[string]string{"AccessibilitySettingsApp.focused_id": "10"}),
		keyEventMap(8, "ResetButton", []any{"AccessibilitySettingsApp", "Panel", "Column", "ButtonRow", "ResetButton"}, 13, 320, 240, map[string]string{"AccessibilitySettingsApp.focused_id": "10", "AccessibilitySettingsApp.reset_count": "0", "StatusText.status_code": "1", "NameTextBox.buffer": "Ada", "EmailTextBox.buffer": "tetra"}, map[string]string{"AccessibilitySettingsApp.focused_id": "10", "AccessibilitySettingsApp.reset_count": "1", "StatusText.status_code": "2", "NameTextBox.buffer": "", "EmailTextBox.buffer": ""}),
		keyEventMap(9, "AccessibilitySettingsApp", []any{"AccessibilitySettingsApp"}, 9, 320, 240, map[string]string{"AccessibilitySettingsApp.focused_id": "10"}, map[string]string{"AccessibilitySettingsApp.focused_id": "5"}),
		resizeEventMap(10, "AccessibilitySettingsApp", []any{"AccessibilitySettingsApp"}, 480, 320, map[string]string{"AccessibilitySettingsApp.focused_id": "5", "NameTextBox.bounds.w": "280", "EmailTextBox.bounds.w": "280"}, map[string]string{"AccessibilitySettingsApp.focused_id": "5", "NameTextBox.bounds.w": "440", "EmailTextBox.bounds.w": "440"}),
	}
	report["frames"] = []any{
		map[string]any{"order": 1, "width": 320, "height": 240, "stride": 1280, "checksum": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "presented": true},
		map[string]any{"order": 2, "width": 320, "height": 240, "stride": 1280, "checksum": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "presented": true},
		map[string]any{"order": 3, "width": 320, "height": 240, "stride": 1280, "checksum": "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "presented": true},
		map[string]any{"order": 4, "width": 320, "height": 240, "stride": 1280, "checksum": "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", "presented": true},
		map[string]any{"order": 5, "width": 480, "height": 320, "stride": 1920, "checksum": "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", "presented": true},
	}
	report["state_transitions"] = []any{
		map[string]any{"order": 1, "component": "AccessibilitySettingsApp", "field": "focused_id", "before": "-1", "after": "5", "cause": "mouse_up"},
		map[string]any{"order": 2, "component": "NameTextBox", "field": "buffer", "before": "", "after": "Ada", "cause": "text_input"},
		map[string]any{"order": 3, "component": "AccessibilitySettingsApp", "field": "focused_id", "before": "5", "after": "7", "cause": "tab"},
		map[string]any{"order": 4, "component": "EmailTextBox", "field": "buffer", "before": "", "after": "tetra", "cause": "text_input"},
		map[string]any{"order": 5, "component": "AccessibilitySettingsApp", "field": "focused_id", "before": "7", "after": "9", "cause": "tab"},
		map[string]any{"order": 6, "component": "AccessibilitySettingsApp", "field": "save_count", "before": "0", "after": "1", "cause": "key_down"},
		map[string]any{"order": 7, "component": "StatusText", "field": "status_code", "before": "0", "after": "1", "cause": "save"},
		map[string]any{"order": 8, "component": "AccessibilitySettingsApp", "field": "focused_id", "before": "9", "after": "10", "cause": "tab"},
		map[string]any{"order": 9, "component": "NameTextBox", "field": "buffer", "before": "Ada", "after": "", "cause": "reset"},
		map[string]any{"order": 10, "component": "EmailTextBox", "field": "buffer", "before": "tetra", "after": "", "cause": "reset"},
		map[string]any{"order": 11, "component": "AccessibilitySettingsApp", "field": "reset_count", "before": "0", "after": "1", "cause": "key_down"},
		map[string]any{"order": 12, "component": "StatusText", "field": "status_code", "before": "1", "after": "2", "cause": "reset"},
		map[string]any{"order": 13, "component": "AccessibilitySettingsApp", "field": "focused_id", "before": "10", "after": "5", "cause": "tab"},
		map[string]any{"order": 14, "component": "AccessibilitySettingsApp", "field": "NameTextBox.bounds.w", "before": "280", "after": "440", "cause": "resize"},
		map[string]any{"order": 15, "component": "AccessibilitySettingsApp", "field": "EmailTextBox.bounds.w", "before": "280", "after": "440", "cause": "resize"},
	}
	report["cases"] = append(report["cases"].([]any),
		map[string]any{"name": "accessibility metadata tree schema", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "accessibility metadata roles labels values states", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "accessibility metadata component tree alignment", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "accessibility metadata focus order alignment", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "accessibility metadata reading order", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "accessibility metadata snapshots update", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "accessibility metadata no DOM ARIA platform host claim", "kind": "positive", "ran": true, "pass": true},
	)
	if mutate != nil {
		mutate(report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal accessibility metadata report: %v", err)
	}
	return raw
}
func validLinuxReleaseAccessibilitySurfaceReportJSON(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessAccessibilityMetadataSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode base release accessibility report: %v", err)
	}
	report["target"] = "linux-x64"
	report["runtime"] = "surface-linux-x64"
	report["source"] = "examples/surface_release_accessibility.tetra"
	report["processes"] = []any{
		map[string]any{"name": "tetra build", "kind": "build", "path": "tetra build --target linux-x64 examples/surface_release_accessibility.tetra -o /tmp/surface-artifacts/surface-release-accessibility", "ran": true, "pass": true, "exit_code": 0},
		map[string]any{"name": "surface component app", "kind": "app", "path": "/tmp/surface-artifacts/surface-release-accessibility", "ran": true, "pass": true, "exit_code": 1, "expected_exit_code": 1},
		map[string]any{"name": "surface linux-x64 real-window probe", "kind": "app", "path": "/tmp/surface-artifacts/surface-accessibility-real-window-probe", "ran": true, "pass": true, "exit_code": 42, "expected_exit_code": 42},
		map[string]any{"name": "surface linux-x64 runtime", "kind": "runtime", "path": "tools/cmd/surface-runtime-smoke", "ran": true, "pass": true, "exit_code": 0},
		map[string]any{"name": "surface linux accessibility host bridge", "kind": "runtime", "path": "/tmp/surface-artifacts/surface-linux-accessibility-bridge.json", "ran": true, "pass": true, "exit_code": 0},
		map[string]any{"name": "surface linux accessibility platform probe", "kind": "runtime", "path": "/tmp/surface-artifacts/surface-linux-accessibility-probe.json", "ran": true, "pass": true, "exit_code": 0},
	}
	report["artifacts"] = []any{
		map[string]any{"kind": "component-app", "path": "/tmp/surface-artifacts/surface-release-accessibility", "sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size": 90001},
		map[string]any{"kind": "linux-accessibility-host-bridge", "path": "/tmp/surface-artifacts/surface-linux-accessibility-bridge.json", "sha256": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "size": 4096},
		map[string]any{"kind": "linux-accessibility-platform-probe", "path": "/tmp/surface-artifacts/surface-linux-accessibility-probe.json", "sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "size": 4096},
	}
	report["artifact_scan"].(map[string]any)["files_checked"] = float64(3)
	report["host_evidence"].(map[string]any)["level"] = "linux-x64-real-window"
	report["host_evidence"].(map[string]any)["backend"] = "wayland-shm-rgba"
	report["host_evidence"].(map[string]any)["real_window"] = true
	report["host_evidence"].(map[string]any)["native_input"] = true
	report["host_evidence"].(map[string]any)["accessibility_bridge"] = true
	report["components"].([]any)[0].(map[string]any)["type"] = "examples.surface_release_accessibility.AccessibilitySettingsApp"
	tree := report["accessibility_tree"].(map[string]any)
	tree["accessibility_level"] = "platform-bridge-v1"
	tree["release_scope"] = "surface-v1-linux-web"
	tree["source"] = "examples/surface_release_accessibility.tetra"
	tree["experimental"] = false
	tree["production_claim"] = true
	tree["platform_host_integration"] = true
	tree["metadata_tree"] = true
	tree["platform_export"] = true
	tree["platform_bridge"] = "linux_accessibility_host_bridge_v1"
	tree["linux_platform_probe"] = true
	tree["linux_probe_artifact"] = "/tmp/surface-artifacts/surface-linux-accessibility-probe.json"
	tree["browser_accessibility_snapshot"] = false
	tree["browser_accessibility_mirror"] = false
	tree["screen_reader_evidence"] = "linux_accessibility_host_bridge_v1"
	report["component_tree"].(map[string]any)["dynamic_level"] = "platform-bridge-v1"
	report["component_tree_api"].(map[string]any)["source"] = "examples/surface_release_accessibility.tetra"
	toolkit := report["toolkit"].(map[string]any)
	toolkit["source"] = "examples/surface_release_accessibility.tetra"
	toolkit["sources"] = append(toolkit["sources"].([]any), "examples/surface_release_accessibility.tetra")
	cases := []any{}
	for _, item := range report["cases"].([]any) {
		name, _ := item.(map[string]any)["name"].(string)
		if !strings.Contains(strings.ToLower(name), "headless") {
			cases = append(cases, item)
		}
	}
	report["cases"] = append(cases,
		map[string]any{"name": "linux-x64 real-window surface", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux-x64 native input event pump", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux-x64 real-window resize event", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux-x64 real-window close event", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "accessibility platform bridge v1 schema", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux accessibility host bridge export", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux accessibility platform probe roles labels values states bounds", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux accessibility probe focus order labels status resize", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "accessibility release honest screen reader evidence", "kind": "positive", "ran": true, "pass": true},
	)
	if mutate != nil {
		mutate(report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal release accessibility report: %v", err)
	}
	return raw
}
func releaseWindowAccessibilityTreeMap() map[string]any {
	return map[string]any{
		"schema":                      "tetra.surface.accessibility-tree.v1",
		"accessibility_level":         "platform-bridge-v1",
		"release_scope":               "surface-v1-linux-web",
		"source":                      "examples/surface_release_form.tetra",
		"module":                      "lib.core.accessibility",
		"widget_module":               "lib.core.widgets",
		"experimental":                false,
		"production_claim":            true,
		"platform_host_integration":   true,
		"dom_aria_integration":        false,
		"screen_reader_evidence":      "linux_accessibility_host_bridge_v1",
		"metadata_tree":               true,
		"platform_export":             true,
		"platform_bridge":             "linux_accessibility_host_bridge_v1",
		"linux_platform_probe":        true,
		"linux_probe_artifact":        "/tmp/surface-artifacts/surface-linux-accessibility-probe.json",
		"derived_from_component_tree": true,
		"uses_component_tree_api":     true,
		"uses_widget_toolkit":         true,
		"manual_bookkeeping":          false,
		"no_dom_ui":                   true,
		"no_user_js":                  true,
		"no_platform_widgets":         true,
		"no_legacy_sidecars":          true,
		"component_tree_schema":       "tetra.surface.component-tree.v1",
		"component_tree_api_schema":   "tetra.surface.component-tree-api.v1",
		"toolkit_schema":              "tetra.surface.toolkit.v1",
		"node_count":                  18,
		"focusable_count":             5,
		"label_count":                 2,
		"textbox_count":               2,
		"button_count":                2,
		"status_count":                1,
		"roles_present":               []any{"root", "panel", "column", "text", "label", "textbox", "checkbox", "row", "button", "status"},
		"focus_order":                 []any{"NameTextBox", "EmailTextBox", "SubscribeCheckbox", "SaveButton", "ResetButton"},
		"reading_order":               []any{"TitleText", "DescriptionText", "NameLabel", "NameTextBox", "EmailLabel", "EmailTextBox", "SubscribeCheckbox", "TermsText", "SaveButton", "ResetButton", "StatusText"},
		"nodes":                       []any{},
		"relationships":               []any{},
		"actions":                     []any{},
		"snapshots":                   []any{},
		"negative_guards": map[string]any{
			"no_borrowed_view_storage":       true,
			"component_id_alignment_checked": true,
			"bounds_alignment_checked":       true,
			"focus_order_alignment_checked":  true,
			"reading_order_checked":          true,
			"label_relationships_checked":    true,
			"state_updates_checked":          true,
			"artifact_scan_checked":          true,
		},
	}
}
func accessibilityComponentTreeMap(dynamicLevel string, rootName string) map[string]any {
	return map[string]any{
		"schema":        "tetra.surface.component-tree.v1",
		"dynamic_level": dynamicLevel,
		"root_id":       0,
		"node_count":    12,
		"focused_id":    5,
		"nodes": []any{
			treeNodeMap(0, rootName, "root", -1, 0, 1, 1, false, RectReport{X: 0, Y: 0, W: 480, H: 320}),
			treeNodeMap(1, "Panel", "panel", 0, 0, 2, 1, false, RectReport{X: 0, Y: 0, W: 480, H: 320}),
			treeNodeMap(2, "Column", "column", 1, 0, 3, 7, false, RectReport{X: 12, Y: 12, W: 456, H: 296}),
			treeNodeMap(3, "TitleText", "text", 2, 0, -1, 0, false, RectReport{X: 20, Y: 20, W: 440, H: 24}),
			treeNodeMap(4, "NameLabel", "text", 2, 1, -1, 0, false, RectReport{X: 20, Y: 52, W: 440, H: 24}),
			treeNodeMap(5, "NameTextBox", "textbox", 2, 2, -1, 0, true, RectReport{X: 20, Y: 84, W: 440, H: 44}),
			treeNodeMap(6, "EmailLabel", "text", 2, 3, -1, 0, false, RectReport{X: 20, Y: 136, W: 440, H: 24}),
			treeNodeMap(7, "EmailTextBox", "textbox", 2, 4, -1, 0, true, RectReport{X: 20, Y: 168, W: 440, H: 44}),
			treeNodeMap(8, "ButtonRow", "row", 2, 5, 9, 2, false, RectReport{X: 20, Y: 224, W: 440, H: 44}),
			treeNodeMap(9, "SaveButton", "button", 8, 0, -1, 0, true, RectReport{X: 20, Y: 224, W: 132, H: 44}),
			treeNodeMap(10, "ResetButton", "button", 8, 1, -1, 0, true, RectReport{X: 164, Y: 224, W: 132, H: 44}),
			treeNodeMap(11, "StatusText", "text", 2, 6, -1, 0, false, RectReport{X: 20, Y: 280, W: 440, H: 24}),
		},
		"layout_passes": []any{
			map[string]any{"component_id": 5, "pass": "initial", "bounds": rectMap(RectReport{X: 20, Y: 84, W: 280, H: 44}), "measured": map[string]any{"w": 280, "h": 44}},
			map[string]any{"component_id": 7, "pass": "initial", "bounds": rectMap(RectReport{X: 20, Y: 168, W: 280, H: 44}), "measured": map[string]any{"w": 280, "h": 44}},
			map[string]any{"component_id": 5, "pass": "resize", "bounds": rectMap(RectReport{X: 20, Y: 84, W: 440, H: 44}), "measured": map[string]any{"w": 440, "h": 44}},
			map[string]any{"component_id": 7, "pass": "resize", "bounds": rectMap(RectReport{X: 20, Y: 168, W: 440, H: 44}), "measured": map[string]any{"w": 440, "h": 44}},
			map[string]any{"component_id": 11, "pass": "status-update", "bounds": rectMap(RectReport{X: 20, Y: 280, W: 440, H: 24}), "measured": map[string]any{"w": 440, "h": 24}},
		},
		"draw_order":  []any{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
		"focus_order": []any{5, 7, 9, 10},
		"dispatch_paths": []any{
			map[string]any{"event": "click", "target_id": 5, "x": 40, "y": 100, "path": []any{0, 1, 2, 5}},
			map[string]any{"event": "click", "target_id": 7, "x": 40, "y": 184, "path": []any{0, 1, 2, 7}},
			map[string]any{"event": "key", "target_id": 9, "x": 40, "y": 240, "path": []any{0, 1, 2, 8, 9}},
			map[string]any{"event": "key", "target_id": 10, "x": 180, "y": 240, "path": []any{0, 1, 2, 8, 10}},
		},
	}
}
func accessibilityComponentTreeAPIMap() map[string]any {
	return map[string]any{
		"schema":             "tetra.surface.component-tree-api.v1",
		"api_level":          "builder-layout-dispatch-v1",
		"source":             "examples/surface_accessibility_settings.tetra",
		"manual_bookkeeping": false,
		"builder": map[string]any{
			"root_created_by":     "tree_add_root",
			"children_created_by": "tree_add_child",
			"node_count":          12,
			"capacity":            24,
			"overflow_checked":    true,
		},
		"invariants": map[string]any{"tree_validate_ran": true, "tree_validate_status": 0, "parent_child_links_checked": true, "child_indices_checked": true, "child_count_checked": true, "first_child_checked": true},
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
			map[string]any{"helper": "widgets.hit_test_accessibility_settings", "x": 40, "y": 100, "target": "NameTextBox", "path": []any{0, 1, 2, 5}},
			map[string]any{"helper": "widgets.hit_test_accessibility_settings", "x": 40, "y": 184, "target": "EmailTextBox", "path": []any{0, 1, 2, 7}},
			map[string]any{"helper": "widgets.hit_test_accessibility_settings", "x": 180, "y": 240, "target": "ResetButton", "path": []any{0, 1, 2, 8, 10}},
		},
		"dispatch_paths": []any{
			map[string]any{"helper": "tree_build_dispatch_path", "target": "NameTextBox", "path": []any{0, 1, 2, 5}},
			map[string]any{"helper": "tree_build_dispatch_path", "target": "EmailTextBox", "path": []any{0, 1, 2, 7}},
			map[string]any{"helper": "tree_build_dispatch_path", "target": "SaveButton", "path": []any{0, 1, 2, 8, 9}},
			map[string]any{"helper": "tree_build_dispatch_path", "target": "ResetButton", "path": []any{0, 1, 2, 8, 10}},
		},
	}
}
func accessibilityToolkitMap() map[string]any {
	return map[string]any{
		"schema":                       "tetra.surface.toolkit.v1",
		"toolkit_level":                "toolkit-reuse-v1",
		"reuse_level":                  "multi-form-widget-reuse-v1",
		"source":                       "examples/surface_accessibility_settings.tetra",
		"sources":                      []any{"examples/surface_toolkit_form.tetra", "examples/surface_toolkit_settings.tetra", "examples/surface_accessibility_settings.tetra"},
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
		"example_count":                3,
		"text_box_count":               2,
		"button_count":                 2,
		"multi_textbox_evidence":       true,
		"multi_form_evidence":          true,
		"widgets": []any{
			toolkitWidgetMap("Panel", "Panel", 1, "", true),
			toolkitWidgetMap("Column", "Column", 2, "", true),
			toolkitWidgetMap("TitleText", "Text", 3, "text", true),
			toolkitWidgetMap("NameLabel", "Text", 4, "label", true),
			toolkitWidgetMap("NameTextBox", "TextBox", 5, "", true),
			toolkitWidgetMap("EmailLabel", "Text", 6, "label", true),
			toolkitWidgetMap("EmailTextBox", "TextBox", 7, "", true),
			toolkitWidgetMap("ButtonRow", "Row", 8, "", true),
			toolkitWidgetMap("SaveButton", "Button", 9, "save", true),
			toolkitWidgetMap("ResetButton", "Button", 10, "reset", true),
			toolkitWidgetMap("StatusText", "Text", 11, "status", true),
		},
		"reusable_sources": []any{"lib/core/widgets.tetra:panel_init", "lib/core/widgets.tetra:column_init", "lib/core/widgets.tetra:text_init", "lib/core/widgets.tetra:textbox_init", "lib/core/widgets.tetra:row_init", "lib/core/widgets.tetra:button_init", "lib/core/widgets.tetra:add_accessible_textbox", "lib/core/widgets.tetra:add_accessible_button", "lib/core/widgets.tetra:add_accessible_status"},
	}
}
func accessibilityTreeMap() map[string]any {
	return map[string]any{
		"schema":                      "tetra.surface.accessibility-tree.v1",
		"accessibility_level":         "metadata-tree-v1",
		"source":                      "examples/surface_accessibility_settings.tetra",
		"module":                      "lib.core.accessibility",
		"widget_module":               "lib.core.widgets",
		"experimental":                true,
		"production_claim":            false,
		"platform_host_integration":   false,
		"dom_aria_integration":        false,
		"screen_reader_evidence":      false,
		"derived_from_component_tree": true,
		"uses_component_tree_api":     true,
		"uses_widget_toolkit":         true,
		"manual_bookkeeping":          false,
		"no_dom_ui":                   true,
		"no_user_js":                  true,
		"no_platform_widgets":         true,
		"no_legacy_sidecars":          true,
		"component_tree_schema":       "tetra.surface.component-tree.v1",
		"component_tree_api_schema":   "tetra.surface.component-tree-api.v1",
		"toolkit_schema":              "tetra.surface.toolkit.v1",
		"node_count":                  12,
		"focusable_count":             4,
		"label_count":                 2,
		"textbox_count":               2,
		"button_count":                2,
		"status_count":                1,
		"roles_present":               []any{"root", "panel", "column", "text", "label", "textbox", "row", "button", "status"},
		"nodes":                       accessibilityNodes(),
		"relationships":               accessibilityRelationships(),
		"focus_order":                 []any{"NameTextBox", "EmailTextBox", "SaveButton", "ResetButton"},
		"reading_order":               []any{"TitleText", "NameLabel", "NameTextBox", "EmailLabel", "EmailTextBox", "SaveButton", "ResetButton", "StatusText"},
		"actions": []any{
			map[string]any{"target": "NameTextBox", "action": "edit", "semantic": "text-input"},
			map[string]any{"target": "EmailTextBox", "action": "edit", "semantic": "text-input"},
			map[string]any{"target": "SaveButton", "action": "press", "semantic": "save"},
			map[string]any{"target": "ResetButton", "action": "press", "semantic": "reset"},
		},
		"snapshots":       accessibilitySnapshots(),
		"negative_guards": map[string]any{"no_borrowed_view_storage": true, "component_id_alignment_checked": true, "bounds_alignment_checked": true, "focus_order_alignment_checked": true, "reading_order_checked": true, "label_relationships_checked": true, "state_updates_checked": true, "artifact_scan_checked": true},
	}
}
func accessibilityNodes() []any {
	return []any{
		accessibilityNodeMap(0, 0, -1, "AccessibilitySettingsApp", "root", RectReport{X: 0, Y: 0, W: 480, H: 320}, false, false, false, "", "", "", 0, nil, -1, 0),
		accessibilityNodeMap(1, 1, 0, "Panel", "panel", RectReport{X: 0, Y: 0, W: 480, H: 320}, false, false, false, "", "", "", 0, nil, -1, 1),
		accessibilityNodeMap(2, 2, 1, "Column", "column", RectReport{X: 12, Y: 12, W: 456, H: 296}, false, false, false, "", "", "", 0, nil, -1, 2),
		accessibilityNodeMap(3, 3, 2, "TitleText", "text", RectReport{X: 20, Y: 20, W: 440, H: 24}, false, false, false, "", "", "title", 0, nil, -1, 3),
		accessibilityNodeMap(4, 4, 2, "NameLabel", "label", RectReport{X: 20, Y: 52, W: 440, H: 24}, false, false, false, "NameTextBox", "", "name", 0, nil, -1, 4),
		accessibilityNodeMap(5, 5, 2, "NameTextBox", "textbox", RectReport{X: 20, Y: 84, W: 440, H: 44}, true, true, true, "", "NameLabel", "name-present", 0, []any{"focus", "edit"}, 0, 5),
		accessibilityNodeMap(6, 6, 2, "EmailLabel", "label", RectReport{X: 20, Y: 136, W: 440, H: 24}, false, false, false, "EmailTextBox", "", "email", 0, nil, -1, 6),
		accessibilityNodeMap(7, 7, 2, "EmailTextBox", "textbox", RectReport{X: 20, Y: 168, W: 440, H: 44}, true, true, false, "", "EmailLabel", "email-present", 0, []any{"focus", "edit"}, 1, 7),
		accessibilityNodeMap(8, 8, 2, "ButtonRow", "row", RectReport{X: 20, Y: 224, W: 440, H: 44}, false, false, false, "", "", "", 0, nil, -1, 8),
		accessibilityNodeMap(9, 9, 8, "SaveButton", "button", RectReport{X: 20, Y: 224, W: 132, H: 44}, true, false, false, "", "", "save", 0, []any{"focus", "press", "save"}, 2, 9),
		accessibilityNodeMap(10, 10, 8, "ResetButton", "button", RectReport{X: 164, Y: 224, W: 132, H: 44}, true, false, false, "", "", "reset", 0, []any{"focus", "press", "reset"}, 3, 10),
		accessibilityNodeMap(11, 11, 2, "StatusText", "status", RectReport{X: 20, Y: 280, W: 440, H: 24}, false, false, false, "", "", "reset", 0, nil, -1, 11),
	}
}
func accessibilityNodeMap(id int, componentID int, parentID int, name string, role string, bounds RectReport, focusable bool, editable bool, focused bool, labelFor string, labelledBy string, valueKind string, valueLen int, actions []any, focusIndex int, readingIndex int) map[string]any {
	value := map[string]any{"id": id, "component_id": componentID, "parent_id": parentID, "name": name, "role": role, "bounds": rectMap(bounds), "visible": true, "enabled": true, "focusable": focusable, "focused": focused, "editable": editable, "readonly": false, "required": false, "pressed": false, "invalid": false, "focus_index": focusIndex, "reading_index": readingIndex}
	if labelFor != "" {
		value["label_for"] = labelFor
	}
	if labelledBy != "" {
		value["labelled_by"] = labelledBy
	}
	if valueKind != "" {
		value["value_kind"] = valueKind
	}
	if valueLen > 0 {
		value["value_len"] = valueLen
	}
	if actions != nil {
		value["actions"] = actions
	}
	return value
}
func accessibilityRelationships() []any {
	return []any{
		map[string]any{"kind": "label_for", "from": "NameLabel", "to": "NameTextBox"},
		map[string]any{"kind": "labelled_by", "from": "NameTextBox", "to": "NameLabel"},
		map[string]any{"kind": "label_for", "from": "EmailLabel", "to": "EmailTextBox"},
		map[string]any{"kind": "labelled_by", "from": "EmailTextBox", "to": "EmailLabel"},
	}
}
func accessibilitySnapshots() []any {
	return []any{
		accessibilitySnapshotMap("initial", 1, "", -1, -1, 0, 0, "idle", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "1111111111111111111111111111111111111111111111111111111111111111", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		accessibilitySnapshotMap("after_name_focus", 2, "NameTextBox", 5, 5, 0, 0, "idle", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "2222222222222222222222222222222222222222222222222222222222222222", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
		accessibilitySnapshotMap("after_name_text", 3, "NameTextBox", 5, 5, 3, 0, "idle", "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "3333333333333333333333333333333333333333333333333333333333333333", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
		accessibilitySnapshotMap("after_email_focus", 4, "EmailTextBox", 7, 7, 3, 0, "idle", "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", "4444444444444444444444444444444444444444444444444444444444444444", "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"),
		accessibilitySnapshotMap("after_email_text", 5, "EmailTextBox", 7, 7, 3, 5, "idle", "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", "5555555555555555555555555555555555555555555555555555555555555555", "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"),
		accessibilitySnapshotMap("after_save", 6, "SaveButton", 9, 9, 3, 5, "saved", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "6666666666666666666666666666666666666666666666666666666666666666", "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"),
		accessibilitySnapshotMap("after_reset", 7, "ResetButton", 10, 10, 0, 0, "reset", "9999999999999999999999999999999999999999999999999999999999999999", "7777777777777777777777777777777777777777777777777777777777777777", "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"),
		accessibilitySnapshotMap("after_resize", 8, "NameTextBox", 5, 5, 0, 0, "reset", "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", "8888888888888888888888888888888888888888888888888888888888888888", "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"),
	}
}
func accessibilitySnapshotMap(name string, generation int, focused string, focusedComponentID int, focusedAccessibilityNodeID int, nameLen int, emailLen int, status string, boundsChecksum string, metadataChecksum string, frameChecksum string) map[string]any {
	return map[string]any{"name": name, "generation": generation, "focused": focused, "focused_component_id": focusedComponentID, "focused_accessibility_node_id": focusedAccessibilityNodeID, "name_value_len": nameLen, "email_value_len": emailLen, "status_value": status, "bounds_checksum": boundsChecksum, "metadata_checksum": metadataChecksum, "frame_checksum": frameChecksum}
}
