package surface

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateReportAcceptsHeadlessSurfaceRuntimeEvidence(t *testing.T) {
	raw := validHeadlessSurfaceReportJSON()
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportAcceptsLinuxX64SurfaceRuntimeEvidence(t *testing.T) {
	raw := validLinuxX64SurfaceReportJSON()
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportAcceptsLinuxX64RealWindowSurfaceRuntimeEvidence(t *testing.T) {
	raw := validLinuxX64RealWindowSurfaceReportJSON()
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportAcceptsWASM32WebSurfaceRuntimeEvidence(t *testing.T) {
	raw := validWASM32WebSurfaceReportJSON()
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportAcceptsWASM32WebBrowserCanvasSurfaceRuntimeEvidence(t *testing.T) {
	raw := validWASM32WebBrowserCanvasSurfaceReportJSON()
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportAcceptsHeadlessTextFocusInputSurfaceRuntimeEvidence(t *testing.T) {
	raw := validHeadlessTextFocusInputSurfaceReportJSON()
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportAcceptsHeadlessComponentTreeSurfaceRuntimeEvidence(t *testing.T) {
	raw := validHeadlessComponentTreeSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

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

func TestValidateSurfaceAccessibilityMetadataTreeReport(t *testing.T) {
	raw := validHeadlessAccessibilityMetadataSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateSurfaceReleaseSummaryAcceptsScopedLinuxWebCurrent(t *testing.T) {
	raw := validSurfaceReleaseSummaryJSON()
	if err := ValidateReleaseSummary(raw); err != nil {
		t.Fatalf("ValidateReleaseSummary failed: %v\n%s", err, raw)
	}
}

func TestValidateSurfaceReleaseSummaryRejectsFakePromotionClaims(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(string) string
		want   string
	}{
		{
			name: "missing unsupported targets",
			mutate: func(raw string) string {
				return strings.Replace(raw, `  "unsupported_targets": ["macos-x64", "windows-x64", "wasm32-wasi"],
`, ``, 1)
			},
			want: "unsupported_targets",
		},
		{
			name: "experimental true",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"experimental": false`, `"experimental": true`, 1)
			},
			want: "experimental",
		},
		{
			name: "production false",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"production_claim": true`, `"production_claim": false`, 1)
			},
			want: "production_claim",
		},
		{
			name: "unsupported target in supported targets",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"supported_targets": ["headless", "linux-x64", "wasm32-web"]`, `"supported_targets": ["headless", "linux-x64", "wasm32-web", "macos-x64"]`, 1)
			},
			want: "supported_targets",
		},
		{
			name: "dom ui",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"dom_ui": false`, `"dom_ui": true`, 1)
			},
			want: "dom_ui",
		},
		{
			name: "platform widgets",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"platform_widgets": false`, `"platform_widgets": true`, 1)
			},
			want: "platform_widgets",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := []byte(tc.mutate(string(validSurfaceReleaseSummaryJSON())))
			err := ValidateReleaseSummary(raw)
			if err == nil {
				t.Fatalf("expected release summary to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

func TestValidateSurfaceTextInputReportAcceptsProductionBaseline(t *testing.T) {
	raw := validSurfaceTextInputReportJSON()
	if err := ValidateTextInputReport(raw); err != nil {
		t.Fatalf("ValidateTextInputReport failed: %v\n%s", err, raw)
	}
}

func TestValidateSurfaceTextInputReportRejectsFakeProductionClaims(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(string) string
		want   string
	}{
		{
			name: "experimental true",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"experimental": false`, `"experimental": true`, 1)
			},
			want: "experimental",
		},
		{
			name: "production false",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"production_claim": true`, `"production_claim": false`, 1)
			},
			want: "production_claim",
		},
		{
			name: "missing utf8 validation",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"utf8_validation": true`, `"utf8_validation": false`, 1)
			},
			want: "utf8_validation",
		},
		{
			name: "missing composition commit",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"composition_commit": true`, `"composition_commit": false`, 1)
			},
			want: "composition_commit",
		},
		{
			name: "missing clipboard write",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"clipboard_write": true`, `"clipboard_write": false`, 1)
			},
			want: "clipboard_write",
		},
		{
			name: "missing clipboard host abi",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"clipboard_host_abi": true`, `"clipboard_host_abi": false`, 1)
			},
			want: "clipboard_host_abi",
		},
		{
			name: "missing composition trace commit",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"commit":true`, `"commit":false`, 1)
			},
			want: "composition_trace.commit",
		},
		{
			name: "missing clipboard owned copy",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"clipboard_owned_copy": true`, `"clipboard_owned_copy": false`, 1)
			},
			want: "clipboard_owned_copy",
		},
		{
			name: "borrowed view storage",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"borrowed_view_storage": false`, `"borrowed_view_storage": true`, 1)
			},
			want: "borrowed_view_storage",
		},
		{
			name: "missing safe view lifetime",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"safe_view_lifetime_checked": true`, `"safe_view_lifetime_checked": false`, 1)
			},
			want: "safe_view_lifetime_checked",
		},
		{
			name: "missing target evidence",
			mutate: func(raw string) string {
				return strings.Replace(raw, `  "target": "headless",`+"\n", "", 1)
			},
			want: "target",
		},
		{
			name: "missing process evidence",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"processes": [
    {"name":"tetra build","kind":"build","path":"tetra build --target linux-x64 examples/surface_release_text_input.tetra -o /tmp/surface-artifacts/surface-release-text-input","ran":true,"pass":true,"exit_code":0},
    {"name":"surface component app","kind":"app","path":"/tmp/surface-artifacts/surface-release-text-input","ran":true,"pass":true,"exit_code":1,"expected_exit_code":1},
    {"name":"surface headless runtime","kind":"runtime","path":"tools/cmd/surface-runtime-smoke --mode headless-release-text-input","ran":true,"pass":true,"exit_code":0}
  ]`, `"processes": []`, 1)
			},
			want: "process evidence",
		},
		{
			name: "missing composition case evidence",
			mutate: func(raw string) string {
				return strings.Replace(raw, `    {"name":"release text input composition commit","kind":"positive","ran":true,"pass":true},
`, "", 1)
			},
			want: "composition commit",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := []byte(tc.mutate(string(validSurfaceTextInputReportJSON())))
			err := ValidateTextInputReport(raw)
			if err == nil {
				t.Fatalf("expected text-input report to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
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

func TestValidateSurfaceBrowserReleaseReport(t *testing.T) {
	raw := validWASM32WebReleaseBrowserSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateSurfaceBrowserReleaseRejectsFakeProductionClaims(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "starter loader level",
			mutate: func(report map[string]any) {
				host := report["host_evidence"].(map[string]any)
				host["level"] = "wasm32-web-compiler-owned-loader"
				host["backend"] = "node-surface-host"
				host["native_input"] = false
			},
			want: "browser release host_evidence.level",
		},
		{
			name: "missing browser clipboard",
			mutate: func(report map[string]any) {
				report["host_evidence"].(map[string]any)["browser_clipboard"] = false
			},
			want: "browser_clipboard",
		},
		{
			name: "missing composition trace",
			mutate: func(report map[string]any) {
				report["host_evidence"].(map[string]any)["browser_composition"] = false
			},
			want: "browser_composition",
		},
		{
			name: "missing accessibility snapshot",
			mutate: func(report map[string]any) {
				report["host_evidence"].(map[string]any)["browser_accessibility_snapshot"] = false
			},
			want: "browser_accessibility_snapshot",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validWASM32WebReleaseBrowserSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected browser release fake claim to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

func TestValidateSurfaceLinuxReleaseWindowReport(t *testing.T) {
	raw := validLinuxReleaseWindowSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateSurfaceLinuxReleaseWindowRejectsFakeProductionClaims(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "memfd starter level",
			mutate: func(report map[string]any) {
				host := report["host_evidence"].(map[string]any)
				host["level"] = "linux-x64-memfd-starter"
				host["backend"] = "memfd-rgba"
				host["real_window"] = false
				host["native_input"] = false
			},
			want: "linux release host_evidence.level",
		},
		{
			name: "old real window level",
			mutate: func(report map[string]any) {
				host := report["host_evidence"].(map[string]any)
				host["level"] = "linux-x64-real-window"
				host["backend"] = "wayland-shm-rgba"
			},
			want: "linux release host_evidence.level",
		},
		{
			name: "missing clipboard",
			mutate: func(report map[string]any) {
				report["host_evidence"].(map[string]any)["clipboard"] = false
			},
			want: "clipboard",
		},
		{
			name: "missing composition",
			mutate: func(report map[string]any) {
				report["host_evidence"].(map[string]any)["composition"] = false
			},
			want: "composition",
		},
		{
			name: "missing accessibility bridge",
			mutate: func(report map[string]any) {
				report["host_evidence"].(map[string]any)["accessibility_bridge"] = false
			},
			want: "accessibility_bridge",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validLinuxReleaseWindowSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected linux release fake claim to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
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

func TestValidateReportRejectsTextFocusInputMissingCaretAndDeleteEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessTextFocusInputSurfaceReportJSON()), `"caret":"1",`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"text focus input backspace delete","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected text focus input report without caret/delete evidence to fail")
	}
	for _, want := range []string{"caret", "backspace delete"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateReportRejectsTextFocusInputMissingTabRoutingEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessTextFocusInputSurfaceReportJSON()), `,
    {"name":"text focus input Tab changes focus","kind":"positive","ran":true,"pass":true}`, ``, 1)
	raw = strings.Replace(raw, `{"order":7,"kind":"key_down","target_component":"TextInputApp","dispatch_path":["TextInputApp"],"handled":true,"pass":true,"x":0,"y":0,"key":9`, `{"order":7,"kind":"key_down","target_component":"TextInputApp","dispatch_path":["TextInputApp"],"handled":true,"pass":true,"x":0,"y":0,"key":0`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected text focus input report without Tab routing evidence to fail")
	}
	for _, want := range []string{"Tab", "focus routing"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateReportRejectsMissingHostEvidence(t *testing.T) {
	raw := []byte(strings.Replace(string(validHeadlessSurfaceReportJSON()), `  "host_evidence": {"level":"deterministic-headless","backend":"software-rgba","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false},
`, ``, 1))
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected report without explicit host_evidence to fail")
	}
	if !strings.Contains(err.Error(), "host_evidence") {
		t.Fatalf("error = %v, want host_evidence diagnostic", err)
	}
}

func TestValidateReportRejectsLinuxX64ReportWithHeadlessRuntimeEvidence(t *testing.T) {
	raw := []byte(strings.Replace(string(validHeadlessSurfaceReportJSON()), `"target": "headless"`, `"target": "linux-x64"`, 1))
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected linux-x64 report with headless runtime evidence to fail")
	}
	if !strings.Contains(err.Error(), "linux-x64") || !strings.Contains(err.Error(), "surface-linux-x64") {
		t.Fatalf("error = %v, want linux-x64 runtime evidence diagnostic", err)
	}
}

func TestValidateReportRejectsLinuxX64MemfdStarterClaimingRealWindow(t *testing.T) {
	raw := strings.Replace(string(validLinuxX64SurfaceReportJSON()), `"real_window":false,"native_input":false`, `"real_window":true,"native_input":true`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected linux-x64 memfd starter real-window claim to fail")
	}
	if !strings.Contains(err.Error(), "memfd starter") || !strings.Contains(err.Error(), "real_window") {
		t.Fatalf("error = %v, want memfd starter real_window diagnostic", err)
	}
}

func TestValidateReportRejectsLinuxX64RealWindowWithoutRealWindowProbeEvidence(t *testing.T) {
	raw := strings.Replace(string(validLinuxX64SurfaceReportJSON()),
		`"host_evidence": {"level":"linux-x64-memfd-starter","backend":"memfd-rgba","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false}`,
		`"host_evidence": {"level":"linux-x64-real-window","backend":"x11-rgba","framebuffer":true,"real_window":true,"native_input":true,"user_facing_platform_widgets":false}`,
		1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected linux-x64 real-window claim without real-window process/case evidence to fail")
	}
	for _, want := range []string{"real-window", "native input"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateReportRejectsWASM32WebReportWithHeadlessRuntimeEvidence(t *testing.T) {
	raw := []byte(strings.Replace(string(validHeadlessSurfaceReportJSON()), `"target": "headless"`, `"target": "wasm32-web"`, 1))
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected wasm32-web report with headless runtime evidence to fail")
	}
	if !strings.Contains(err.Error(), "wasm32-web") || !strings.Contains(err.Error(), "surface-wasm32-web") {
		t.Fatalf("error = %v, want wasm32-web runtime evidence diagnostic", err)
	}
}

func TestValidateReportRejectsWASM32WebReportMissingCompilerOwnedLoaderEvidence(t *testing.T) {
	raw := strings.Replace(string(validWASM32WebSurfaceReportJSON()), `,
    {"name":"compiler-owned wasm Surface loader","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected wasm32-web report without compiler-owned loader evidence to fail")
	}
	if !strings.Contains(err.Error(), "compiler-owned wasm Surface loader") {
		t.Fatalf("error = %v, want compiler-owned loader evidence diagnostic", err)
	}
}

func TestValidateReportRejectsWASM32WebReportMissingActualPresentedFrameTraceEvidence(t *testing.T) {
	raw := strings.Replace(string(validWASM32WebSurfaceReportJSON()), `,
    {"order":4,"width":320,"height":200,"stride":1280,"checksum":"4444444444444444444444444444444444444444444444444444444444444444","presented":true}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"wasm32-web actual presented frame trace","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected wasm32-web report without actual presented frame trace evidence to fail")
	}
	if !strings.Contains(err.Error(), "actual presented frame trace") {
		t.Fatalf("error = %v, want actual presented frame trace evidence diagnostic", err)
	}
}

func TestValidateReportRejectsWASM32WebReportMissingImportValidatorProcess(t *testing.T) {
	raw := strings.Replace(string(validWASM32WebSurfaceReportJSON()), `,
    {"name":"surface wasm32-web import validator","kind":"runtime","path":"go run ./tools/cmd/validate-wasm-imports --target wasm32-web /tmp/surface-artifacts/surface-counter.wasm","ran":true,"pass":true,"exit_code":0}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected wasm32-web report without validate-wasm-imports process evidence to fail")
	}
	for _, want := range []string{"wasm32-web", "validate-wasm-imports"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateReportRejectsWASM32WebBrowserCanvasWithoutBrowserProcess(t *testing.T) {
	raw := strings.Replace(string(validWASM32WebBrowserCanvasSurfaceReportJSON()), `/usr/bin/chromium --headless <surface-browser-canvas-runner> wasm=/tmp/surface-artifacts/surface-browser-counter.wasm`, `node scripts/tools/web_run_module.mjs /tmp/surface-artifacts/surface-browser-counter.wasm`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected browser canvas report without Chromium process evidence to fail")
	}
	if !strings.Contains(err.Error(), "Chromium-compatible browser") {
		t.Fatalf("error = %v, want Chromium-compatible browser process diagnostic", err)
	}
}

func TestValidateReportRejectsWASM32WebBrowserCanvasMissingInputEvidence(t *testing.T) {
	raw := strings.Replace(string(validWASM32WebBrowserCanvasSurfaceReportJSON()), `,
    {"order":2,"kind":"key_down","target_component":"CounterApp","dispatch_path":["CounterApp"],"handled":true,"pass":true,"x":0,"y":0,"key":32,"width":320,"height":200,"timestamp_ms":1,"buffer_slots":[6,0,0,0,32,320,200,1,0],"before_state":{"CounterApp.count":"1","CounterApp.key_count":"0"},"after_state":{"CounterApp.count":"2","CounterApp.key_count":"1"}}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"wasm32-web browser canvas keyboard input","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected browser canvas report without keyboard input evidence to fail")
	}
	for _, want := range []string{"keyboard input", "key_down"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateReportRejectsWASM32WebReportMissingRunnerTraceArtifact(t *testing.T) {
	raw := strings.Replace(string(validWASM32WebSurfaceReportJSON()), `,
    {"kind":"runner-trace","path":"/tmp/surface-artifacts/surface-runner-trace.json","sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","size":413}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected wasm32-web report without runner trace artifact to fail")
	}
	if !strings.Contains(err.Error(), "runner trace artifact") {
		t.Fatalf("error = %v, want runner trace artifact diagnostic", err)
	}
}

func TestValidateReportRejectsHeadlessReportMissingRunnerTraceEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `,
    {"kind":"runner-trace","path":"/tmp/surface-artifacts/surface-runner-trace.json","sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","size":409}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"headless actual runner trace","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected headless report without runner trace evidence to fail")
	}
	if !strings.Contains(err.Error(), "headless actual runner trace") {
		t.Fatalf("error = %v, want headless runner trace evidence diagnostic", err)
	}
}

func TestValidateReportRejectsLinuxX64ReportMissingAppPresentedFrameEvidence(t *testing.T) {
	raw := strings.Replace(string(validLinuxX64SurfaceReportJSON()), `,
    {"order":3,"width":2,"height":2,"stride":8,"checksum":"3333333333333333333333333333333333333333333333333333333333333333","presented":true}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"linux-x64 app-presented RGBA checksum","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected linux-x64 report without app-presented frame evidence to fail")
	}
	if !strings.Contains(err.Error(), "app-presented RGBA checksum") {
		t.Fatalf("error = %v, want app-presented frame evidence diagnostic", err)
	}
}

func TestValidateReportRejectsLinuxX64ReportMissingCounterComponentAppPresentedFrameEvidence(t *testing.T) {
	raw := strings.Replace(string(validLinuxX64SurfaceReportJSON()), `,
    {"order":4,"width":320,"height":200,"stride":1280,"checksum":"4444444444444444444444444444444444444444444444444444444444444444","presented":true}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"surface linux-x64 counter app presented frame probe","kind":"app","path":"/tmp/surface-artifacts/surface-counter-present-probe","ran":true,"pass":true,"exit_code":-1,"expected_exit_code":-1}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"linux-x64 counter component app-presented frame","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected linux-x64 report without counter component app-presented frame evidence to fail")
	}
	if !strings.Contains(err.Error(), "counter component app-presented frame") {
		t.Fatalf("error = %v, want counter component app-presented frame evidence diagnostic", err)
	}
}

func TestValidateReportRejectsLinuxX64ReportMissingEventSequenceProbeEvidence(t *testing.T) {
	raw := strings.Replace(string(validLinuxX64SurfaceReportJSON()), `,
    {"name":"surface linux-x64 event sequence probe build","kind":"build","path":"/tmp/tetra build event sequence probe","ran":true,"pass":true,"exit_code":0},
    {"name":"surface linux-x64 event sequence probe","kind":"app","path":"/tmp/surface-event-sequence-probe","ran":true,"pass":true,"exit_code":42,"expected_exit_code":42}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"linux-x64 host event sequence","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected linux-x64 report without event sequence probe evidence to fail")
	}
	if !strings.Contains(err.Error(), "event sequence") {
		t.Fatalf("error = %v, want event sequence probe evidence diagnostic", err)
	}
}

func TestValidateReportRejectsMissingPrePostEventFrameSequence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `,
    {"order":2,"width":320,"height":200,"stride":1280,"checksum":"2222222222222222222222222222222222222222222222222222222222222222","presented":true}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"pre/post event frame sequence","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without pre/post frame evidence to fail")
	}
	if !strings.Contains(err.Error(), "pre/post event frame sequence") {
		t.Fatalf("error = %v, want pre/post frame sequence diagnostic", err)
	}
}

func TestValidateReportRejectsLegacyMetadataEvidence(t *testing.T) {
	raw := []byte(`{"schema":"tetra.ui.v1","status":"pass","source":"examples/ui_web_smoke.tetra"}`)
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected legacy metadata report to fail")
	}
	if !strings.Contains(err.Error(), SchemaV1) {
		t.Fatalf("error = %v, want Surface runtime schema rejection", err)
	}
}

func TestValidateReportRejectsDocsOnlyMarkers(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"source": "examples/surface_counter.tetra"`, `"source": "docs-only surface note"`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected docs-only marker to fail")
	}
	if !strings.Contains(err.Error(), "docs-only") {
		t.Fatalf("error = %v, want docs-only rejection", err)
	}
}

func TestValidateReportRejectsForbiddenEvidenceMarkers(t *testing.T) {
	for _, tc := range []struct {
		source string
		want   string
	}{
		{source: "web-only", want: "web-only"},
		{source: "metadata-only", want: "metadata-only"},
		{source: "node-only", want: "node-only"},
		{source: "dom-only", want: "dom-only"},
		{source: "build-only", want: "build-only"},
		{source: "surface fake evidence", want: "fake"},
		{source: "surface stale evidence", want: "stale"},
		{source: "surface mock evidence", want: "mock"},
		{source: "placeholder", want: "placeholder"},
	} {
		t.Run(tc.source, func(t *testing.T) {
			raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"source": "examples/surface_counter.tetra"`, `"source": "`+tc.source+`"`, 1)
			err := ValidateReport([]byte(raw))
			if err == nil {
				t.Fatalf("expected marker %q to fail", tc.source)
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("error = %v, want marker rejection for %q", err, tc.want)
			}
		})
	}
}

func TestValidateReportRejectsLegacyUISidecarMarkers(t *testing.T) {
	for _, tc := range []struct {
		name string
		want string
	}{
		{name: "generated .ui.html sidecar", want: ".ui.html"},
		{name: "generated .ui.web.mjs sidecar", want: ".ui.web.mjs"},
		{name: "generated .ui.json sidecar", want: ".ui.json"},
		{name: "DOM UI surface", want: "dom ui"},
		{name: "user JavaScript bridge", want: "user javascript"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"pure Tetra component app"`, `"`+tc.name+`"`, 1)
			err := ValidateReport([]byte(raw))
			if err == nil {
				t.Fatalf("expected legacy UI marker %q to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("error = %v, want marker rejection for %q", err, tc.want)
			}
		})
	}
}

func TestValidateReportRejectsUserFacingPlatformWidgetMarkers(t *testing.T) {
	for _, tc := range []struct {
		name string
		want string
	}{
		{name: "React component surface", want: "react"},
		{name: "GTK widget surface", want: "gtk widget"},
		{name: "Qt widget surface", want: "qt widget"},
		{name: "WinUI widget surface", want: "winui"},
		{name: "Cocoa widget surface", want: "cocoa"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"pure Tetra component app"`, `"`+tc.name+`"`, 1)
			err := ValidateReport([]byte(raw))
			if err == nil {
				t.Fatalf("expected platform widget marker %q to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("error = %v, want marker rejection for %q", err, tc.want)
			}
		})
	}
}

func TestValidateReportRejectsMissingNoLegacyUISidecarEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `,
    {"name":"no legacy UI sidecar artifacts","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing no-sidecar evidence to fail")
	}
	if !strings.Contains(err.Error(), "no legacy UI sidecar artifacts") {
		t.Fatalf("error = %v, want no legacy UI sidecar evidence diagnostic", err)
	}
}

func TestValidateReportRejectsMissingArtifactScanEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `,
  "artifact_scan": {"root":"/tmp/surface-artifacts","files_checked":2,"forbidden_paths":[],"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without artifact_scan evidence to fail")
	}
	for _, want := range []string{"artifact_scan"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateReportRejectsArtifactOutsideArtifactScanRoot(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"artifact_scan": {"root":"/tmp/surface-artifacts"`, `"artifact_scan": {"root":"/tmp/other-surface-artifacts"`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report whose artifacts are outside artifact_scan.root to fail")
	}
	for _, want := range []string{"artifact_scan.root", "outside"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateReportRejectsArtifactScanCheckingFewerFilesThanReportedArtifacts(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"files_checked":2`, `"files_checked":1`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report whose artifact_scan checked fewer files than reported artifacts to fail")
	}
	for _, want := range []string{"artifact_scan.files_checked", "reported artifacts"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateReportRejectsMissingHostProvidedPointerEventEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `,
    {"name":"host-provided pointer event dispatch","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing host-provided pointer event evidence to fail")
	}
	if !strings.Contains(err.Error(), "host-provided pointer event dispatch") {
		t.Fatalf("error = %v, want host-provided pointer event evidence diagnostic", err)
	}
}

func TestValidateReportRejectsComponentMissingMeasureLayoutAbilities(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"abilities":["measure","layout","draw","event","focus","text","accessibility"]`, `"abilities":["draw","event","focus","text","accessibility"]`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected component without measure/layout abilities to fail")
	}
	for _, want := range []string{"measure ability", "layout ability"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateReportRejectsMissingFocusAbilityAndEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"abilities":["measure","layout","draw","event","focus","text","accessibility"]`, `"abilities":["measure","layout","draw","event","text","accessibility"]`, 1)
	raw = strings.Replace(raw, `,
    {"name":"component focus dispatch","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without focus ability and case evidence to fail")
	}
	for _, want := range []string{"focus ability", "component focus dispatch"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateReportRejectsMissingAccessibilityAbilityAndEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"abilities":["measure","layout","draw","event","focus","text","accessibility"]`, `"abilities":["measure","layout","draw","event","focus","text"]`, 1)
	raw = strings.Replace(raw, `,
    {"name":"component accessibility metadata","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without accessibility ability and case evidence to fail")
	}
	for _, want := range []string{"accessibility ability", "component accessibility metadata"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateReportRejectsMissingTextAbilityAndEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"abilities":["measure","layout","draw","event","focus","text","accessibility"]`, `"abilities":["measure","layout","draw","event","focus","accessibility"]`, 1)
	raw = strings.Replace(raw, `,
    {"order":3,"kind":"text_input","target_component":"CounterButton","handled":true,"pass":true,"x":0,"y":0,"text_len":2,"text_bytes_hex":"4f4b","before_state":{"CounterApp.text_count":"0","CounterButton.text_len_seen":"0"},"after_state":{"CounterApp.text_count":"1","CounterButton.text_len_seen":"2"}}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"order":2,"component":"CounterApp","field":"text_count","before":"0","after":"1","cause":"text_input"}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"component text input scalar dispatch","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without text ability and scalar text-input evidence to fail")
	}
	for _, want := range []string{"text ability", "component text input scalar dispatch"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateReportRejectsMissingHostTextPayloadBufferEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"text_len":2,"text_bytes_hex":"4f4b",`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"host text payload buffer","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without host text payload buffer evidence to fail")
	}
	for _, want := range []string{"text payload", "host text payload buffer"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateReportRejectsMissingHostEventBufferEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"key":0,"width":320,"height":200,"timestamp_ms":0,"buffer_slots":[5,48,96,1,0,320,200,0,0],`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"host event buffer poll_event","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without host event buffer evidence to fail")
	}
	for _, want := range []string{"event buffer", "host event buffer poll_event"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateReportRejectsMissingHostEventBufferSequenceEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()),
		`"timestamp_ms":1,"text_len":2,"text_bytes_hex":"4f4b","buffer_slots":[8,0,0,0,0,320,200,1,2]`,
		`"timestamp_ms":0,"text_len":2,"text_bytes_hex":"4f4b","buffer_slots":[8,0,0,0,0,320,200,0,2]`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without host event buffer pointer/text sequence to fail")
	}
	if !strings.Contains(err.Error(), "event buffer pointer/text sequence") {
		t.Fatalf("error = %v, want host event buffer pointer/text sequence diagnostic", err)
	}
}

func TestValidateReportRejectsMissingComponentHierarchyEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `,
    {"id":"CounterButton","type":"examples.surface_counter.CounterButton","parent":"CounterApp","bounds":{"x":32,"y":80,"w":160,"h":48},"abilities":["measure","layout","draw","event","focus","text","accessibility"],"state":{"pressed":"false","focused":"true","text_len_seen":"2","accessibility_role":"button"}}`, ``, 1)
	raw = strings.Replace(raw, `"target_component":"CounterButton"`, `"target_component":"CounterApp"`, 1)
	raw = strings.Replace(raw, `,
    {"name":"component hierarchy dispatch","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without child component hierarchy evidence to fail")
	}
	for _, want := range []string{"component hierarchy", "component hierarchy dispatch"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateReportRejectsMissingComponentLayoutBoundsEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"bounds":{"x":32,"y":80,"w":160,"h":48},`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without child component bounds evidence to fail")
	}
	if !strings.Contains(err.Error(), "layout bounds") {
		t.Fatalf("error = %v, want layout bounds diagnostic", err)
	}
}

func TestValidateReportRejectsMissingEventDispatchPathEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"dispatch_path":["CounterApp","CounterButton"],`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without child dispatch_path evidence to fail")
	}
	if !strings.Contains(err.Error(), "dispatch_path") {
		t.Fatalf("error = %v, want dispatch_path diagnostic", err)
	}
}

func TestValidateReportRejectsDispatchPathSkippingParent(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"dispatch_path":["CounterApp","CounterButton"]`, `"dispatch_path":["CounterButton"]`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report with dispatch_path skipping parent to fail")
	}
	if !strings.Contains(err.Error(), "dispatch_path") || !strings.Contains(err.Error(), "parent") {
		t.Fatalf("error = %v, want dispatch_path parent diagnostic", err)
	}
}

func TestValidateReportRejectsPointerDispatchOutsideTargetBounds(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"x":48,"y":96,"key":0,"width":320,"height":200,"timestamp_ms":0,"buffer_slots":[5,48,96,1,0,320,200,0,0]`, `"x":4,"y":4,"key":0,"width":320,"height":200,"timestamp_ms":0,"buffer_slots":[5,4,4,1,0,320,200,0,0]`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected pointer dispatch outside target bounds to fail")
	}
	if !strings.Contains(err.Error(), "target bounds") {
		t.Fatalf("error = %v, want target bounds diagnostic", err)
	}
}

func TestValidateReportRejectsSourcePathAsExecutableAppProcess(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"kind":"app","path":"/tmp/surface-artifacts/surface-counter"`, `"kind":"app","path":"examples/surface_counter.tetra"`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected app process source path to fail")
	}
	if !strings.Contains(err.Error(), "executable Surface app process") {
		t.Fatalf("error = %v, want executable app path diagnostic", err)
	}
}

func TestValidateReportRejectsBuildProcessMissingReportedSource(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"path":"tetra build --target linux-x64 examples/surface_counter.tetra -o /tmp/surface-artifacts/surface-counter"`, `"path":"tetra build --target linux-x64 examples/other_surface.tetra -o /tmp/surface-artifacts/surface-counter"`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected build process without reported source to fail")
	}
	for _, want := range []string{"build process", "source"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateReportRejectsMissingSurfaceComponentAppProcess(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"name":"surface component app"`, `"name":"surface auxiliary app"`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing Surface component app process to fail")
	}
	if !strings.Contains(err.Error(), "Surface component app process") {
		t.Fatalf("error = %v, want Surface component app process diagnostic", err)
	}
}

func TestValidateReportRejectsMissingComponentAppArtifactEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `    {"kind":"component-app","path":"/tmp/surface-artifacts/surface-counter","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":49172},
`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without Surface component app artifact hash evidence to fail")
	}
	for _, want := range []string{"artifact", "Surface component app"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateReportRejectsWASM32WebMissingCompilerOwnedLoaderArtifact(t *testing.T) {
	raw := strings.Replace(string(validWASM32WebSurfaceReportJSON()), `,
    {"kind":"compiler-owned-loader","path":"/tmp/surface-artifacts/surface-counter.mjs","sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","size":4931}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected wasm32-web report without compiler-owned loader artifact to fail")
	}
	for _, want := range []string{"compiler-owned loader artifact", "wasm32-web"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateReportRejectsGeneratedHTMLArtifactEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `{"kind":"component-app","path":"/tmp/surface-artifacts/surface-counter"`, `{"kind":"component-app","path":"/tmp/surface-artifacts/surface-counter.html"`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected generated HTML artifact evidence to fail")
	}
	if !strings.Contains(err.Error(), "generated HTML UI") {
		t.Fatalf("error = %v, want generated HTML UI diagnostic", err)
	}
}

func TestValidateReportRejectsPlatformWidgetArtifactEvidence(t *testing.T) {
	for _, tc := range []struct {
		suffix string
		want   string
	}{
		{suffix: ".jsx", want: "react"},
		{suffix: ".tsx", want: "react"},
		{suffix: ".qml", want: "qt"},
		{suffix: ".xaml", want: "winui"},
		{suffix: ".xib", want: "cocoa"},
		{suffix: ".storyboard", want: "cocoa"},
		{suffix: ".glade", want: "gtk"},
	} {
		t.Run(tc.suffix, func(t *testing.T) {
			raw := strings.ReplaceAll(string(validHeadlessSurfaceReportJSON()), `/tmp/surface-artifacts/surface-counter`, `/tmp/surface-artifacts/surface-counter`+tc.suffix)
			err := ValidateReport([]byte(raw))
			if err == nil {
				t.Fatalf("expected platform widget artifact suffix %q to fail", tc.suffix)
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("error = %v, want platform artifact rejection for %q", err, tc.want)
			}
		})
	}
}

func TestValidateReportRejectsSourceComponentModuleMismatch(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"source": "examples/surface_counter.tetra"`, `"source": "examples/other_surface.tetra"`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected source/component module mismatch to fail")
	}
	for _, want := range []string{"source module", "component type"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateReportRejectsMissingFrameChecksumAndStateTransition(t *testing.T) {
	raw := []byte(`{
  "schema": "tetra.surface.runtime.v1",
  "status": "pass",
  "target": "headless",
  "host": "linux-x64",
  "runtime": "surface-headless",
  "surface_schema": "tetra.surface.v1",
  "host_abi": "tetra.surface.host-abi.v1",
  "host_evidence": {"level":"deterministic-headless","backend":"software-rgba","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false},
  "source": "examples/surface_counter.tetra",
  "processes": [
    {"name":"tetra build","kind":"build","path":"tetra build --target linux-x64 examples/surface_counter.tetra -o /tmp/surface-artifacts/surface-counter","ran":true,"pass":true,"exit_code":0},
    {"name":"surface component app","kind":"app","path":"/tmp/surface-artifacts/surface-counter","ran":true,"pass":true,"exit_code":1,"expected_exit_code":1},
    {"name":"surface headless runtime","kind":"runtime","path":"tools/cmd/surface-runtime-smoke","ran":true,"pass":true,"exit_code":0}
  ],
  "artifacts": [
    {"kind":"component-app","path":"/tmp/surface-artifacts/surface-counter","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":49172},
    {"kind":"runner-trace","path":"/tmp/surface-artifacts/surface-runner-trace.json","sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","size":409}
  ],
  "artifact_scan": {"root":"/tmp/surface-artifacts","files_checked":2,"forbidden_paths":[],"pass":true},
  "components": [
    {"id":"CounterApp","type":"examples.surface_counter.CounterApp","bounds":{"x":0,"y":0,"w":320,"h":200},"abilities":["measure","layout","draw","event","focus","text","accessibility"],"state":{"count":"1","text_count":"1","accessibility_role":"button"}},
    {"id":"CounterButton","type":"examples.surface_counter.CounterButton","parent":"CounterApp","bounds":{"x":32,"y":80,"w":160,"h":48},"abilities":["measure","layout","draw","event","focus","text","accessibility"],"state":{"pressed":"false","focused":"true","text_len_seen":"2","accessibility_role":"button"}}
  ],
  "events": [
    {"order":1,"kind":"mouse_up","target_component":"CounterApp","dispatch_path":["CounterApp"],"handled":true,"pass":true,"x":48,"y":96,"before_state":{"CounterApp.count":"0"},"after_state":{"CounterApp.count":"1"}}
  ],
  "frames": [
    {"order":1,"width":320,"height":200,"stride":1280,"checksum":"","presented":true}
  ],
  "cases": [
    {"name":"pure Tetra component app","kind":"positive","ran":true,"pass":true},
    {"name":"headless event dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"headless framebuffer checksum","kind":"positive","ran":true,"pass":true},
    {"name":"headless actual runner trace","kind":"positive","ran":true,"pass":true},
    {"name":"host-provided pointer event dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component hierarchy dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component focus dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component accessibility metadata","kind":"positive","ran":true,"pass":true},
    {"name":"no legacy UI sidecar artifacts","kind":"positive","ran":true,"pass":true},
    {"name":"state transition","kind":"positive","ran":true,"pass":true},
    {"name":"reject legacy UI evidence","kind":"negative","ran":true,"pass":true,"expected_error":"legacy UI evidence rejected"}
  ]
}`)
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing checksum and transition to fail")
	}
	for _, want := range []string{"checksum", "state transition"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}

func validHeadlessSurfaceReportJSON() []byte {
	return []byte(`{
  "schema": "tetra.surface.runtime.v1",
  "status": "pass",
  "target": "headless",
  "host": "linux-x64",
  "runtime": "surface-headless",
  "surface_schema": "tetra.surface.v1",
  "host_abi": "tetra.surface.host-abi.v1",
  "host_evidence": {"level":"deterministic-headless","backend":"software-rgba","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false},
  "source": "examples/surface_counter.tetra",
  "processes": [
    {"name":"tetra build","kind":"build","path":"tetra build --target linux-x64 examples/surface_counter.tetra -o /tmp/surface-artifacts/surface-counter","ran":true,"pass":true,"exit_code":0},
    {"name":"surface component app","kind":"app","path":"/tmp/surface-artifacts/surface-counter","ran":true,"pass":true,"exit_code":1,"expected_exit_code":1},
    {"name":"surface headless runtime","kind":"runtime","path":"tools/cmd/surface-runtime-smoke","ran":true,"pass":true,"exit_code":0}
  ],
  "artifacts": [
    {"kind":"component-app","path":"/tmp/surface-artifacts/surface-counter","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":49172},
    {"kind":"runner-trace","path":"/tmp/surface-artifacts/surface-runner-trace.json","sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","size":409}
  ],
  "artifact_scan": {"root":"/tmp/surface-artifacts","files_checked":2,"forbidden_paths":[],"pass":true},
  "components": [
    {"id":"CounterApp","type":"examples.surface_counter.CounterApp","bounds":{"x":0,"y":0,"w":320,"h":200},"abilities":["measure","layout","draw","event","focus","text","accessibility"],"state":{"count":"1","text_count":"1","accessibility_role":"button"}},
    {"id":"CounterButton","type":"examples.surface_counter.CounterButton","parent":"CounterApp","bounds":{"x":32,"y":80,"w":160,"h":48},"abilities":["measure","layout","draw","event","focus","text","accessibility"],"state":{"pressed":"false","focused":"true","text_len_seen":"2","accessibility_role":"button"}}
  ],
  "events": [
    {"order":1,"kind":"none","target_component":"CounterApp","dispatch_path":["CounterApp"],"handled":false,"pass":true,"x":0,"y":0,"before_state":{"CounterApp.count":"0"},"after_state":{"CounterApp.count":"0"}},
    {"order":2,"kind":"mouse_up","target_component":"CounterButton","dispatch_path":["CounterApp","CounterButton"],"handled":true,"pass":true,"x":48,"y":96,"key":0,"width":320,"height":200,"timestamp_ms":0,"buffer_slots":[5,48,96,1,0,320,200,0,0],"before_state":{"CounterApp.count":"0","CounterButton.pressed":"false"},"after_state":{"CounterApp.count":"1","CounterButton.pressed":"false"}},
    {"order":3,"kind":"text_input","target_component":"CounterButton","dispatch_path":["CounterApp","CounterButton"],"handled":true,"pass":true,"x":0,"y":0,"key":0,"width":320,"height":200,"timestamp_ms":1,"text_len":2,"text_bytes_hex":"4f4b","buffer_slots":[8,0,0,0,0,320,200,1,2],"before_state":{"CounterApp.text_count":"0","CounterButton.text_len_seen":"0"},"after_state":{"CounterApp.text_count":"1","CounterButton.text_len_seen":"2"}}
  ],
  "frames": [
    {"order":1,"width":320,"height":200,"stride":1280,"checksum":"1111111111111111111111111111111111111111111111111111111111111111","presented":true},
    {"order":2,"width":320,"height":200,"stride":1280,"checksum":"2222222222222222222222222222222222222222222222222222222222222222","presented":true}
  ],
  "state_transitions": [
    {"order":1,"component":"CounterApp","field":"count","before":"0","after":"1","cause":"mouse_up"},
    {"order":2,"component":"CounterApp","field":"text_count","before":"0","after":"1","cause":"text_input"}
  ],
  "cases": [
    {"name":"pure Tetra component app","kind":"positive","ran":true,"pass":true},
    {"name":"headless event dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"headless framebuffer checksum","kind":"positive","ran":true,"pass":true},
    {"name":"headless actual runner trace","kind":"positive","ran":true,"pass":true},
    {"name":"host-provided pointer event dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"host event buffer poll_event","kind":"positive","ran":true,"pass":true},
    {"name":"pre/post event frame sequence","kind":"positive","ran":true,"pass":true},
    {"name":"component hierarchy dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component text input scalar dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"host text payload buffer","kind":"positive","ran":true,"pass":true},
    {"name":"component focus dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component accessibility metadata","kind":"positive","ran":true,"pass":true},
    {"name":"no legacy UI sidecar artifacts","kind":"positive","ran":true,"pass":true},
    {"name":"state transition","kind":"positive","ran":true,"pass":true},
    {"name":"reject legacy UI evidence","kind":"negative","ran":true,"pass":true,"expected_error":"legacy UI evidence rejected"}
  ]
}`)
}

func validSurfaceReleaseSummaryJSON() []byte {
	return []byte(`{
  "schema": "tetra.surface.release.v1",
  "release_scope": "surface-v1-linux-web",
  "status": "current",
  "production_claim": true,
  "experimental": false,
  "supported_targets": ["headless", "linux-x64", "wasm32-web"],
  "runtime_targets": ["linux-x64", "wasm32-web"],
  "test_targets": ["headless"],
  "unsupported_targets": ["macos-x64", "windows-x64", "wasm32-wasi"],
  "host_abi": "tetra.surface.host.v1",
  "toolkit": "production-widgets-v1",
  "text_input": "production-text-input-v1",
  "clipboard": "clipboard-text-v1",
  "ime": "composition-baseline-v1",
  "accessibility": "platform-bridge-v1",
  "browser_surface": "browser-canvas-release-v1",
  "linux_surface": "linux-x64-release-window-v1",
  "artifact_hashes_validated": true,
  "legacy_sidecars": false,
  "dom_ui": false,
  "user_js": false,
  "platform_widgets": false
}`)
}

func validSurfaceTextInputReportJSON() []byte {
	return []byte(`{
  "schema": "tetra.surface.text-input.v1",
  "target": "headless",
  "source": "examples/surface_release_text_input.tetra",
  "level": "production-text-input-v1",
  "experimental": false,
  "production_claim": true,
  "storage": "owned-utf8-byte-buffer",
  "utf8_validation": true,
  "caret": true,
  "selection": true,
  "backspace": true,
  "delete": true,
  "home_end": true,
  "arrow_left_right": true,
  "composition_events": true,
  "composition_commit": true,
  "composition_cancel": true,
  "clipboard_read": true,
  "clipboard_write": true,
  "clipboard_host_abi": true,
  "clipboard_owned_copy": true,
  "composition_trace": {"start":true,"update":true,"commit":true,"cancel":true},
  "borrowed_view_storage": false,
  "safe_view_lifetime_checked": true,
  "processes": [
    {"name":"tetra build","kind":"build","path":"tetra build --target linux-x64 examples/surface_release_text_input.tetra -o /tmp/surface-artifacts/surface-release-text-input","ran":true,"pass":true,"exit_code":0},
    {"name":"surface component app","kind":"app","path":"/tmp/surface-artifacts/surface-release-text-input","ran":true,"pass":true,"exit_code":1,"expected_exit_code":1},
    {"name":"surface headless runtime","kind":"runtime","path":"tools/cmd/surface-runtime-smoke --mode headless-release-text-input","ran":true,"pass":true,"exit_code":0}
  ],
  "artifacts": [
    {"kind":"component-app","path":"/tmp/surface-artifacts/surface-release-text-input","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":4096},
    {"kind":"runner-trace","path":"/tmp/surface-artifacts/surface-runner-trace.json","sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","size":2048}
  ],
  "artifact_scan": {"root":"/tmp/surface-artifacts","files_checked":2,"forbidden_paths":[],"pass":true},
  "cases": [
    {"name":"host-provided pointer event dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"host event buffer poll_event","kind":"positive","ran":true,"pass":true},
    {"name":"pre/post event frame sequence","kind":"positive","ran":true,"pass":true},
    {"name":"component hierarchy dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component text input scalar dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"host text payload buffer","kind":"positive","ran":true,"pass":true},
    {"name":"component focus dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component accessibility metadata","kind":"positive","ran":true,"pass":true},
    {"name":"no legacy UI sidecar artifacts","kind":"positive","ran":true,"pass":true},
    {"name":"release text input ASCII insertion","kind":"positive","ran":true,"pass":true},
    {"name":"release text input UTF-8 insertion","kind":"positive","ran":true,"pass":true},
    {"name":"release text input caret home end arrows","kind":"positive","ran":true,"pass":true},
    {"name":"release text input selection replacement","kind":"positive","ran":true,"pass":true},
    {"name":"release text input backspace delete","kind":"positive","ran":true,"pass":true},
    {"name":"release text input clipboard owned copy transfer","kind":"positive","ran":true,"pass":true},
    {"name":"release text input composition start update","kind":"positive","ran":true,"pass":true},
    {"name":"release text input composition commit","kind":"positive","ran":true,"pass":true},
    {"name":"release text input composition cancel","kind":"positive","ran":true,"pass":true},
    {"name":"release text input safe view lifetime checked","kind":"positive","ran":true,"pass":true},
    {"name":"reject legacy UI evidence","kind":"negative","ran":true,"pass":true,"expected_error":"legacy UI evidence rejected"}
  ]
}`)
}

func validHeadlessTextFocusInputSurfaceReportJSON() []byte {
	return []byte(`{
  "schema": "tetra.surface.runtime.v1",
  "status": "pass",
  "target": "headless",
  "host": "linux-x64",
  "runtime": "surface-headless",
  "surface_schema": "tetra.surface.v1",
  "host_abi": "tetra.surface.host-abi.v1",
  "host_evidence": {"level":"deterministic-headless","backend":"software-rgba","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false},
  "source": "examples/surface_textbox_app.tetra",
  "processes": [
    {"name":"tetra build","kind":"build","path":"tetra build --target linux-x64 examples/surface_textbox_app.tetra -o /tmp/surface-artifacts/surface-textbox-app","ran":true,"pass":true,"exit_code":0},
    {"name":"surface component app","kind":"app","path":"/tmp/surface-artifacts/surface-textbox-app","ran":true,"pass":true,"exit_code":1,"expected_exit_code":1},
    {"name":"surface headless runtime","kind":"runtime","path":"tools/cmd/surface-runtime-smoke","ran":true,"pass":true,"exit_code":0}
  ],
  "artifacts": [
    {"kind":"component-app","path":"/tmp/surface-artifacts/surface-textbox-app","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":69657},
    {"kind":"runner-trace","path":"/tmp/surface-artifacts/surface-runner-trace.json","sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","size":13015}
  ],
  "artifact_scan": {"root":"/tmp/surface-artifacts","files_checked":2,"forbidden_paths":[],"pass":true},
  "components": [
    {"id":"TextInputApp","type":"examples.surface_textbox_app.TextInputApp","bounds":{"x":0,"y":0,"w":400,"h":240},"abilities":["measure","layout","draw","event","focus","text","accessibility"],"state":{"focused_component":"SubmitButton","width":"400","height":"240","resize_count":"1","accessibility_role":"none"}},
    {"id":"TextBox","type":"examples.surface_textbox_app.TextBox","parent":"TextInputApp","bounds":{"x":32,"y":64,"w":224,"h":44},"abilities":["measure","layout","draw","event","focus","text","accessibility"],"state":{"focused":"false","buffer":"Z","caret":"1","text_len":"1","backspace_count":"1","delete_count":"1","accessibility_role":"label"}},
    {"id":"SubmitButton","type":"examples.surface_textbox_app.ActionButton","parent":"TextInputApp","bounds":{"x":32,"y":128,"w":128,"h":44},"abilities":["measure","layout","draw","event","focus","text","accessibility"],"state":{"focused":"true","press_count":"1","key_count":"1","accessibility_role":"button"}}
  ],
  "events": [
    {"order":1,"kind":"mouse_up","target_component":"TextBox","dispatch_path":["TextInputApp","TextBox"],"handled":true,"pass":true,"x":48,"y":96,"key":0,"width":320,"height":200,"timestamp_ms":0,"buffer_slots":[5,48,96,1,0,320,200,0,0],"before_state":{"TextInputApp.focused_component":"none","TextBox.focused":"false"},"after_state":{"TextInputApp.focused_component":"TextBox","TextBox.focused":"true"}},
    {"order":2,"kind":"text_input","target_component":"TextBox","dispatch_path":["TextInputApp","TextBox"],"handled":true,"pass":true,"x":0,"y":0,"key":0,"width":320,"height":200,"timestamp_ms":1,"text_len":2,"text_bytes_hex":"4f4b","buffer_slots":[8,0,0,0,0,320,200,1,2],"before_state":{"TextBox.buffer":"","TextBox.caret":"0","TextBox.text_len":"0"},"after_state":{"TextBox.buffer":"OK","TextBox.caret":"2","TextBox.text_len":"2"}},
    {"order":3,"kind":"key_down","target_component":"TextBox","dispatch_path":["TextInputApp","TextBox"],"handled":true,"pass":true,"x":0,"y":0,"key":37,"width":320,"height":200,"timestamp_ms":2,"buffer_slots":[6,0,0,0,37,320,200,2,0],"before_state":{"TextBox.buffer":"OK","TextBox.caret":"2"},"after_state":{"TextBox.buffer":"OK","TextBox.caret":"1"}},
    {"order":4,"kind":"key_down","target_component":"TextBox","dispatch_path":["TextInputApp","TextBox"],"handled":true,"pass":true,"x":0,"y":0,"key":8,"width":320,"height":200,"timestamp_ms":3,"buffer_slots":[6,0,0,0,8,320,200,3,0],"before_state":{"TextBox.buffer":"OK","TextBox.caret":"1"},"after_state":{"TextBox.buffer":"K","TextBox.caret":"0"}},
    {"order":5,"kind":"key_down","target_component":"TextBox","dispatch_path":["TextInputApp","TextBox"],"handled":true,"pass":true,"x":0,"y":0,"key":46,"width":320,"height":200,"timestamp_ms":4,"buffer_slots":[6,0,0,0,46,320,200,4,0],"before_state":{"TextBox.buffer":"K","TextBox.caret":"0"},"after_state":{"TextBox.buffer":"","TextBox.caret":"0"}},
    {"order":6,"kind":"text_input","target_component":"TextBox","dispatch_path":["TextInputApp","TextBox"],"handled":true,"pass":true,"x":0,"y":0,"key":0,"width":320,"height":200,"timestamp_ms":5,"text_len":1,"text_bytes_hex":"5a","buffer_slots":[8,0,0,0,0,320,200,5,1],"before_state":{"TextBox.buffer":"","TextBox.caret":"0","TextBox.text_len":"0"},"after_state":{"TextBox.buffer":"Z","TextBox.caret":"1","TextBox.text_len":"1"}},
    {"order":7,"kind":"key_down","target_component":"TextInputApp","dispatch_path":["TextInputApp"],"handled":true,"pass":true,"x":0,"y":0,"key":9,"width":320,"height":200,"timestamp_ms":6,"buffer_slots":[6,0,0,0,9,320,200,6,0],"before_state":{"TextInputApp.focused_component":"TextBox","TextBox.focused":"true","SubmitButton.focused":"false"},"after_state":{"TextInputApp.focused_component":"SubmitButton","TextBox.focused":"false","SubmitButton.focused":"true"}},
    {"order":8,"kind":"key_down","target_component":"SubmitButton","dispatch_path":["TextInputApp","SubmitButton"],"handled":true,"pass":true,"x":0,"y":0,"key":32,"width":320,"height":200,"timestamp_ms":7,"buffer_slots":[6,0,0,0,32,320,200,7,0],"before_state":{"SubmitButton.press_count":"0","TextBox.buffer":"Z"},"after_state":{"SubmitButton.press_count":"1","TextBox.buffer":"Z"}},
    {"order":9,"kind":"resize","target_component":"TextInputApp","dispatch_path":["TextInputApp"],"handled":true,"pass":true,"x":0,"y":0,"key":0,"width":400,"height":240,"timestamp_ms":8,"buffer_slots":[2,0,0,0,0,400,240,8,0],"before_state":{"TextInputApp.width":"320","TextInputApp.focused_component":"SubmitButton"},"after_state":{"TextInputApp.width":"400","TextInputApp.focused_component":"SubmitButton"}}
  ],
  "frames": [
    {"order":1,"width":320,"height":200,"stride":1280,"checksum":"1111111111111111111111111111111111111111111111111111111111111111","presented":true},
    {"order":2,"width":400,"height":240,"stride":1600,"checksum":"2222222222222222222222222222222222222222222222222222222222222222","presented":true}
  ],
  "state_transitions": [
    {"order":1,"component":"TextInputApp","field":"focused_component","before":"none","after":"TextBox","cause":"mouse_up"},
    {"order":2,"component":"TextBox","field":"buffer","before":"","after":"OK","cause":"text_input"},
    {"order":3,"component":"TextBox","field":"caret","before":"2","after":"1","cause":"key_down"},
    {"order":4,"component":"TextBox","field":"buffer","before":"OK","after":"K","cause":"backspace"},
    {"order":5,"component":"TextBox","field":"buffer","before":"K","after":"","cause":"delete"},
    {"order":6,"component":"TextBox","field":"buffer","before":"","after":"Z","cause":"text_input"},
    {"order":7,"component":"TextInputApp","field":"focused_component","before":"TextBox","after":"SubmitButton","cause":"tab"},
    {"order":8,"component":"SubmitButton","field":"press_count","before":"0","after":"1","cause":"key_down"},
    {"order":9,"component":"TextInputApp","field":"width","before":"320","after":"400","cause":"resize"}
  ],
  "cases": [
    {"name":"pure Tetra component app","kind":"positive","ran":true,"pass":true},
    {"name":"text focus input click focuses TextBox","kind":"positive","ran":true,"pass":true},
    {"name":"text focus input Tab changes focus","kind":"positive","ran":true,"pass":true},
    {"name":"text focus input keyboard routes only focused component","kind":"positive","ran":true,"pass":true},
    {"name":"text focus input text insertion","kind":"positive","ran":true,"pass":true},
    {"name":"text focus input caret movement","kind":"positive","ran":true,"pass":true},
    {"name":"text focus input backspace delete","kind":"positive","ran":true,"pass":true},
    {"name":"text focus input resize preserves focus","kind":"positive","ran":true,"pass":true},
    {"name":"text focus input rendered frame update","kind":"positive","ran":true,"pass":true},
    {"name":"host-provided pointer event dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"host event buffer poll_event","kind":"positive","ran":true,"pass":true},
    {"name":"pre/post event frame sequence","kind":"positive","ran":true,"pass":true},
    {"name":"component hierarchy dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component text input scalar dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"host text payload buffer","kind":"positive","ran":true,"pass":true},
    {"name":"component focus dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component accessibility metadata","kind":"positive","ran":true,"pass":true},
    {"name":"no legacy UI sidecar artifacts","kind":"positive","ran":true,"pass":true},
    {"name":"state transition","kind":"positive","ran":true,"pass":true},
    {"name":"reject legacy UI evidence","kind":"negative","ran":true,"pass":true,"expected_error":"legacy UI evidence rejected"},
    {"name":"headless event dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"headless framebuffer checksum","kind":"positive","ran":true,"pass":true},
    {"name":"headless actual runner trace","kind":"positive","ran":true,"pass":true}
  ]
}`)
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

func validWASM32WebReleaseBrowserSurfaceReportJSON(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessProductionToolkitSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode base production toolkit report: %v", err)
	}
	report["target"] = "wasm32-web"
	report["runtime"] = "surface-wasm32-web"
	report["source"] = "examples/surface_release_form.tetra"
	report["host_evidence"] = map[string]any{
		"level":                          "wasm32-web-browser-canvas-release-v1",
		"backend":                        "browser-canvas-rgba-accessible",
		"framebuffer":                    true,
		"real_window":                    false,
		"native_input":                   true,
		"browser_canvas":                 true,
		"browser_input":                  true,
		"browser_clipboard":              true,
		"browser_clipboard_harness":      "deterministic-browser-clipboard-v1",
		"browser_composition":            true,
		"browser_accessibility_snapshot": true,
		"browser_accessibility_mirror":   true,
		"user_facing_platform_widgets":   false,
	}
	report["processes"] = []any{
		map[string]any{"name": "tetra build", "kind": "build", "path": "tetra build --target wasm32-web examples/surface_release_form.tetra -o /tmp/surface-artifacts/surface-release-form.wasm", "ran": true, "pass": true, "exit_code": 0},
		map[string]any{"name": "surface wasm32-web browser canvas component app", "kind": "app", "path": "/usr/bin/chromium --headless <surface-browser-canvas-runner> scenario=release-browser wasm=/tmp/surface-artifacts/surface-release-form.wasm", "ran": true, "pass": true, "exit_code": 0, "expected_exit_code": 0},
		map[string]any{"name": "surface wasm32-web import validator", "kind": "runtime", "path": "go run ./tools/cmd/validate-wasm-imports --target wasm32-web /tmp/surface-artifacts/surface-release-form.wasm", "ran": true, "pass": true, "exit_code": 0},
		map[string]any{"name": "surface wasm32-web browser canvas runtime", "kind": "runtime", "path": "Chromium release browser fixture", "ran": true, "pass": true, "exit_code": 0},
		map[string]any{"name": "surface wasm32-web browser canvas trace", "kind": "runtime", "path": "/usr/bin/chromium --headless --dump-dom http://127.0.0.1:1/surface-browser-canvas-runner?scenario=release-browser", "ran": true, "pass": true, "exit_code": 0},
	}
	report["artifacts"] = []any{
		map[string]any{"kind": "component-app", "path": "/tmp/surface-artifacts/surface-release-form.wasm", "sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size": 9604},
		map[string]any{"kind": "compiler-owned-loader", "path": "/tmp/surface-artifacts/surface-release-form.mjs", "sha256": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "size": 4939},
		map[string]any{"kind": "runner-trace", "path": "/tmp/surface-artifacts/surface-runner-trace.json", "sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "size": 4096},
	}
	report["artifact_scan"].(map[string]any)["files_checked"] = float64(3)
	cases := make([]any, 0, len(report["cases"].([]any)))
	for _, item := range report["cases"].([]any) {
		row, ok := item.(map[string]any)
		if !ok {
			cases = append(cases, item)
			continue
		}
		name, _ := row["name"].(string)
		if strings.Contains(strings.ToLower(name), "headless") {
			continue
		}
		cases = append(cases, item)
	}
	report["cases"] = append(cases,
		map[string]any{"name": "wasm32-web Surface Host ABI imports", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "compiler-owned wasm Surface loader", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "wasm32-web browser canvas surface", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "wasm32-web browser canvas RGBA readback", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "wasm32-web browser canvas pointer input", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "wasm32-web browser canvas keyboard input", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "wasm32-web browser canvas resize input", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "wasm32-web browser canvas text input", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "compiler-owned browser canvas Surface host", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "browser release Surface v1 schema", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "browser release Chromium canvas readback", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "browser release native pointer keyboard text resize", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "browser release deterministic clipboard harness", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "browser release composition trace", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "browser release accessibility snapshot mirror", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "browser release forbidden web sidecar rejection", "kind": "negative", "ran": true, "pass": true, "expected_error": "forbidden web sidecar rejected"},
	)
	if mutate != nil {
		mutate(report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal browser release report: %v", err)
	}
	return raw
}

func validLinuxReleaseWindowSurfaceReportJSON(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessProductionToolkitSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode base production toolkit report: %v", err)
	}
	report["target"] = "linux-x64"
	report["runtime"] = "surface-linux-x64"
	report["source"] = "examples/surface_release_form.tetra"
	report["host_evidence"] = map[string]any{
		"level":                        "linux-x64-release-window-v1",
		"backend":                      "wayland-shm-rgba-release-v1",
		"framebuffer":                  true,
		"real_window":                  true,
		"native_input":                 true,
		"text_input":                   true,
		"clipboard":                    true,
		"composition":                  true,
		"accessibility_bridge":         true,
		"user_facing_platform_widgets": false,
	}
	report["processes"] = []any{
		map[string]any{"name": "tetra build", "kind": "build", "path": "tetra build --target linux-x64 examples/surface_release_form.tetra -o /tmp/surface-artifacts/surface-release-form", "ran": true, "pass": true, "exit_code": 0},
		map[string]any{"name": "surface component app", "kind": "app", "path": "/tmp/surface-artifacts/surface-release-form", "ran": true, "pass": true, "exit_code": 1, "expected_exit_code": 1},
		map[string]any{"name": "surface linux-x64 real-window probe", "kind": "app", "path": "/tmp/surface-artifacts/surface-release-window-probe", "ran": true, "pass": true, "exit_code": 42, "expected_exit_code": 42},
		map[string]any{"name": "surface linux-x64 release clipboard harness", "kind": "runtime", "path": "/tmp/surface-artifacts/surface-linux-clipboard-harness.json", "ran": true, "pass": true, "exit_code": 0},
		map[string]any{"name": "surface linux-x64 release composition harness", "kind": "runtime", "path": "/tmp/surface-artifacts/surface-linux-composition-harness.json", "ran": true, "pass": true, "exit_code": 0},
		map[string]any{"name": "surface linux accessibility host bridge", "kind": "runtime", "path": "/tmp/surface-artifacts/surface-linux-accessibility-bridge.json", "ran": true, "pass": true, "exit_code": 0},
		map[string]any{"name": "surface linux accessibility platform probe", "kind": "runtime", "path": "/tmp/surface-artifacts/surface-linux-accessibility-probe.json", "ran": true, "pass": true, "exit_code": 0},
		map[string]any{"name": "surface linux-x64 runtime", "kind": "runtime", "path": "tools/cmd/surface-runtime-smoke --mode linux-x64-release-window", "ran": true, "pass": true, "exit_code": 0},
	}
	report["artifacts"] = []any{
		map[string]any{"kind": "component-app", "path": "/tmp/surface-artifacts/surface-release-form", "sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size": 90001},
		map[string]any{"kind": "linux-accessibility-host-bridge", "path": "/tmp/surface-artifacts/surface-linux-accessibility-bridge.json", "sha256": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "size": 4096},
		map[string]any{"kind": "linux-accessibility-platform-probe", "path": "/tmp/surface-artifacts/surface-linux-accessibility-probe.json", "sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "size": 4096},
	}
	report["artifact_scan"].(map[string]any)["files_checked"] = float64(3)
	events := report["events"].([]any)
	report["events"] = append(events, map[string]any{
		"order": 14, "kind": "close", "target_component": "SurfaceReleaseFormApp",
		"dispatch_path": []any{"SurfaceReleaseFormApp"}, "handled": true, "pass": true,
		"width": 560, "height": 420, "timestamp_ms": 13,
		"buffer_slots": []any{9, 0, 0, 0, 0, 560, 420, 13, 0},
		"before_state": map[string]any{"SurfaceReleaseFormApp.open": "true"},
		"after_state":  map[string]any{"SurfaceReleaseFormApp.open": "false"},
	})
	cases := make([]any, 0, len(report["cases"].([]any)))
	for _, item := range report["cases"].([]any) {
		row, ok := item.(map[string]any)
		if !ok {
			cases = append(cases, item)
			continue
		}
		name, _ := row["name"].(string)
		if strings.Contains(strings.ToLower(name), "headless") {
			continue
		}
		cases = append(cases, item)
	}
	report["cases"] = append(cases,
		map[string]any{"name": "linux-x64 real-window surface", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux-x64 native input event pump", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux-x64 real-window resize event", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux-x64 real-window close event", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "accessibility platform bridge v1 schema", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux accessibility host bridge export", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "accessibility release honest screen reader evidence", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux release window v1 schema", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux release real window presented frame", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux release native pointer key text resize close", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux release clipboard harness", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux release composition harness", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux release accessibility bridge probe", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux release forbids memfd starter promotion", "kind": "negative", "ran": true, "pass": true, "expected_error": "memfd starter rejected"},
	)
	report["accessibility_tree"] = releaseWindowAccessibilityTreeMap()
	if mutate != nil {
		mutate(report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal linux release window report: %v", err)
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

func componentMap(id string, typ string, parent string, bounds RectReport, state map[string]string) map[string]any {
	value := map[string]any{
		"id":        id,
		"type":      typ,
		"bounds":    rectMap(bounds),
		"abilities": []any{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
		"state":     stringMapAny(state),
	}
	if parent != "" {
		value["parent"] = parent
	}
	return value
}

func treeNodeMap(id int, name string, kind string, parentID int, childIndex int, firstChild int, childCount int, focusable bool, bounds RectReport) map[string]any {
	return map[string]any{"id": id, "name": name, "kind": kind, "parent_id": parentID, "child_index": childIndex, "first_child": firstChild, "child_count": childCount, "focusable": focusable, "bounds": rectMap(bounds)}
}

func rectMap(rect RectReport) map[string]any {
	return map[string]any{"x": rect.X, "y": rect.Y, "w": rect.W, "h": rect.H}
}

func toolkitWidgetMap(name string, kind string, nodeID int, role string, reusable bool) map[string]any {
	value := map[string]any{"name": name, "kind": kind, "node_id": nodeID, "reusable": reusable, "ordinary_tetra_struct": true}
	if role != "" {
		if kind == "Button" {
			value["action"] = role
		} else {
			value["role"] = role
		}
	}
	if kind == "TextBox" {
		value["editable"] = true
	}
	return value
}

func eventMap(order int, kind string, target string, path []any, x int, y int, key int, width int, height int, before map[string]string, after map[string]string) map[string]any {
	return map[string]any{
		"order": order, "kind": kind, "target_component": target, "dispatch_path": path,
		"handled": true, "pass": true, "x": x, "y": y, "key": key, "width": width, "height": height,
		"timestamp_ms": order - 1, "buffer_slots": []any{5, x, y, 1, key, width, height, order - 1, 0},
		"before_state": stringMapAny(before), "after_state": stringMapAny(after),
	}
}

func keyEventMap(order int, target string, path []any, key int, width int, height int, before map[string]string, after map[string]string) map[string]any {
	return map[string]any{
		"order": order, "kind": "key_down", "target_component": target, "dispatch_path": path,
		"handled": true, "pass": true, "x": 0, "y": 0, "key": key, "width": width, "height": height,
		"timestamp_ms": order - 1, "buffer_slots": []any{6, 0, 0, 0, key, width, height, order - 1, 0},
		"before_state": stringMapAny(before), "after_state": stringMapAny(after),
	}
}

func textEventMap(order int, target string, path []any, textLen int, textHex string, width int, height int, before map[string]string, after map[string]string) map[string]any {
	return map[string]any{
		"order": order, "kind": "text_input", "target_component": target, "dispatch_path": path,
		"handled": true, "pass": true, "x": 0, "y": 0, "key": 0, "width": width, "height": height,
		"timestamp_ms": order - 1, "text_len": textLen, "text_bytes_hex": textHex,
		"buffer_slots": []any{8, 0, 0, 0, 0, width, height, order - 1, textLen},
		"before_state": stringMapAny(before), "after_state": stringMapAny(after),
	}
}

func resizeEventMap(order int, target string, path []any, width int, height int, before map[string]string, after map[string]string) map[string]any {
	return map[string]any{
		"order": order, "kind": "resize", "target_component": target, "dispatch_path": path,
		"handled": true, "pass": true, "x": 0, "y": 0, "key": 0, "width": width, "height": height,
		"timestamp_ms": order - 1, "buffer_slots": []any{2, 0, 0, 0, 0, width, height, order - 1, 0},
		"before_state": stringMapAny(before), "after_state": stringMapAny(after),
	}
}

func stringMapAny(values map[string]string) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func treeComponent(id string, typ string, parent string, bounds RectReport, state map[string]string) ComponentReport {
	return ComponentReport{
		ID:        id,
		Type:      typ,
		Parent:    parent,
		Bounds:    bounds,
		Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
		State:     state,
	}
}

func intPtrForTest(v int) *int {
	return &v
}

func validLinuxX64SurfaceReportJSON() []byte {
	raw := string(validHeadlessSurfaceReportJSON())
	replacements := []struct {
		old string
		new string
	}{
		{old: `"target": "headless"`, new: `"target": "linux-x64"`},
		{old: `"runtime": "surface-headless"`, new: `"runtime": "surface-linux-x64"`},
		{old: `"host_evidence": {"level":"deterministic-headless","backend":"software-rgba","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false}`,
			new: `"host_evidence": {"level":"linux-x64-memfd-starter","backend":"memfd-rgba","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false}`},
		{old: `{"name":"surface component app","kind":"app","path":"/tmp/surface-artifacts/surface-counter","ran":true,"pass":true,"exit_code":1,"expected_exit_code":1}`,
			new: `{"name":"surface component app","kind":"app","path":"/tmp/surface-artifacts/surface-counter","ran":true,"pass":true,"exit_code":1,"expected_exit_code":1},
    {"name":"surface linux-x64 host probe build","kind":"build","path":"/tmp/tetra build probe","ran":true,"pass":true,"exit_code":0},
    {"name":"surface linux-x64 host probe","kind":"app","path":"/tmp/surface-host-probe","ran":true,"pass":true,"exit_code":42,"expected_exit_code":42},
    {"name":"surface linux-x64 event sequence probe build","kind":"build","path":"/tmp/tetra build event sequence probe","ran":true,"pass":true,"exit_code":0},
    {"name":"surface linux-x64 event sequence probe","kind":"app","path":"/tmp/surface-event-sequence-probe","ran":true,"pass":true,"exit_code":42,"expected_exit_code":42},
    {"name":"surface linux-x64 counter app presented frame probe","kind":"app","path":"/tmp/surface-artifacts/surface-counter-present-probe","ran":true,"pass":true,"exit_code":-1,"expected_exit_code":-1}`},
		{old: `"surface headless runtime"`, new: `"surface linux-x64 runtime"`},
		{old: `"headless event dispatch"`, new: `"linux-x64 Surface Host ABI open/present/close"`},
		{old: `"headless framebuffer checksum"`, new: `"linux-x64 framebuffer present evidence"`},
	}
	for _, repl := range replacements {
		raw = strings.Replace(raw, repl.old, repl.new, 1)
	}
	raw = strings.Replace(raw, `,
    {"kind":"runner-trace","path":"/tmp/surface-artifacts/surface-runner-trace.json","sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","size":409}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"headless actual runner trace","kind":"positive","ran":true,"pass":true}`, ``, 1)
	raw = strings.Replace(raw, `{"order":2,"width":320,"height":200,"stride":1280,"checksum":"2222222222222222222222222222222222222222222222222222222222222222","presented":true}`,
		`{"order":2,"width":320,"height":200,"stride":1280,"checksum":"2222222222222222222222222222222222222222222222222222222222222222","presented":true},
    {"order":3,"width":2,"height":2,"stride":8,"checksum":"3333333333333333333333333333333333333333333333333333333333333333","presented":true},
    {"order":4,"width":320,"height":200,"stride":1280,"checksum":"4444444444444444444444444444444444444444444444444444444444444444","presented":true}`, 1)
	raw = strings.Replace(raw, `{"name":"linux-x64 framebuffer present evidence","kind":"positive","ran":true,"pass":true}`,
		`{"name":"linux-x64 framebuffer present evidence","kind":"positive","ran":true,"pass":true},
    {"name":"linux-x64 host event sequence","kind":"positive","ran":true,"pass":true},
    {"name":"linux-x64 app-presented RGBA checksum","kind":"positive","ran":true,"pass":true},
    {"name":"linux-x64 counter component app-presented frame","kind":"positive","ran":true,"pass":true}`, 1)
	return []byte(raw)
}

func validLinuxX64RealWindowSurfaceReportJSON() []byte {
	raw := string(validLinuxX64SurfaceReportJSON())
	replacements := []struct {
		old string
		new string
	}{
		{old: `"source": "examples/surface_counter.tetra"`, new: `"source": "examples/surface_window_counter.tetra"`},
		{old: `"host_evidence": {"level":"linux-x64-memfd-starter","backend":"memfd-rgba","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false}`,
			new: `"host_evidence": {"level":"linux-x64-real-window","backend":"wayland-shm-rgba","framebuffer":true,"real_window":true,"native_input":true,"user_facing_platform_widgets":false}`},
		{old: `examples/surface_counter.tetra`, new: `examples/surface_window_counter.tetra`},
		{old: `/tmp/surface-artifacts/surface-counter`, new: `/tmp/surface-artifacts/surface-window-counter`},
		{old: `examples.surface_counter.CounterApp`, new: `examples.surface_window_counter.CounterApp`},
		{old: `examples.surface_counter.CounterButton`, new: `examples.surface_window_counter.CounterButton`},
	}
	for _, repl := range replacements {
		raw = strings.ReplaceAll(raw, repl.old, repl.new)
	}
	raw = strings.Replace(raw, `,
    {"name":"surface linux-x64 event sequence probe build","kind":"build","path":"/tmp/tetra build event sequence probe","ran":true,"pass":true,"exit_code":0},
    {"name":"surface linux-x64 event sequence probe","kind":"app","path":"/tmp/surface-event-sequence-probe","ran":true,"pass":true,"exit_code":42,"expected_exit_code":42},
    {"name":"surface linux-x64 counter app presented frame probe","kind":"app","path":"/tmp/surface-artifacts/surface-window-counter-present-probe","ran":true,"pass":true,"exit_code":-1,"expected_exit_code":-1}`,
		`,
    {"name":"surface linux-x64 real-window probe","kind":"app","path":"/tmp/surface-artifacts/surface-real-window-probe","ran":true,"pass":true,"exit_code":42,"expected_exit_code":42}`,
		1)
	raw = strings.Replace(raw, `{"order":4,"width":320,"height":200,"stride":1280,"checksum":"4444444444444444444444444444444444444444444444444444444444444444","presented":true}`,
		`{"order":4,"width":320,"height":200,"stride":1280,"checksum":"4444444444444444444444444444444444444444444444444444444444444444","presented":true},
    {"order":5,"width":400,"height":240,"stride":1600,"checksum":"5555555555555555555555555555555555555555555555555555555555555555","presented":true}`, 1)
	raw = strings.Replace(raw, `{"order":3,"kind":"text_input","target_component":"CounterButton","dispatch_path":["CounterApp","CounterButton"],"handled":true,"pass":true,"x":0,"y":0,"key":0,"width":320,"height":200,"timestamp_ms":1,"text_len":2,"text_bytes_hex":"4f4b","buffer_slots":[8,0,0,0,0,320,200,1,2],"before_state":{"CounterApp.text_count":"0","CounterButton.text_len_seen":"0"},"after_state":{"CounterApp.text_count":"1","CounterButton.text_len_seen":"2"}}`,
		`{"order":3,"kind":"key_down","target_component":"CounterApp","dispatch_path":["CounterApp"],"handled":true,"pass":true,"x":0,"y":0,"key":32,"width":320,"height":200,"timestamp_ms":1,"buffer_slots":[6,0,0,0,32,320,200,1,0],"before_state":{"CounterApp.key_count":"0"},"after_state":{"CounterApp.key_count":"1"}},
    {"order":4,"kind":"resize","target_component":"CounterApp","dispatch_path":["CounterApp"],"handled":true,"pass":true,"x":0,"y":0,"key":0,"width":400,"height":240,"timestamp_ms":2,"buffer_slots":[2,0,0,0,0,400,240,2,0],"before_state":{"CounterApp.width":"320","CounterApp.height":"200"},"after_state":{"CounterApp.width":"400","CounterApp.height":"240"}},
    {"order":5,"kind":"text_input","target_component":"CounterButton","dispatch_path":["CounterApp","CounterButton"],"handled":true,"pass":true,"x":0,"y":0,"key":0,"width":400,"height":240,"timestamp_ms":3,"text_len":2,"text_bytes_hex":"4f4b","buffer_slots":[8,0,0,0,0,400,240,3,2],"before_state":{"CounterApp.text_count":"0","CounterButton.text_len_seen":"0"},"after_state":{"CounterApp.text_count":"1","CounterButton.text_len_seen":"2"}},
    {"order":6,"kind":"close","target_component":"CounterApp","dispatch_path":["CounterApp"],"handled":true,"pass":true,"x":0,"y":0,"key":0,"width":400,"height":240,"timestamp_ms":4,"buffer_slots":[1,0,0,0,0,400,240,4,0],"before_state":{"CounterApp.closed":"false"},"after_state":{"CounterApp.closed":"true"}}`,
		1)
	raw = strings.Replace(raw, `{"order":2,"component":"CounterApp","field":"text_count","before":"0","after":"1","cause":"text_input"}`,
		`{"order":2,"component":"CounterApp","field":"key_count","before":"0","after":"1","cause":"key_down"},
    {"order":3,"component":"CounterApp","field":"width","before":"320","after":"400","cause":"resize"},
    {"order":4,"component":"CounterApp","field":"text_count","before":"0","after":"1","cause":"text_input"},
    {"order":5,"component":"CounterApp","field":"closed","before":"false","after":"true","cause":"close"}`, 1)
	raw = strings.Replace(raw, `{"name":"linux-x64 counter component app-presented frame","kind":"positive","ran":true,"pass":true}`,
		`{"name":"linux-x64 counter component app-presented frame","kind":"positive","ran":true,"pass":true},
    {"name":"linux-x64 real-window surface","kind":"positive","ran":true,"pass":true},
    {"name":"linux-x64 native input event pump","kind":"positive","ran":true,"pass":true},
    {"name":"linux-x64 real-window resize event","kind":"positive","ran":true,"pass":true},
    {"name":"linux-x64 real-window close event","kind":"positive","ran":true,"pass":true}`, 1)
	return []byte(raw)
}

func validWASM32WebSurfaceReportJSON() []byte {
	raw := string(validHeadlessSurfaceReportJSON())
	replacements := []struct {
		old string
		new string
	}{
		{old: `"target": "headless"`, new: `"target": "wasm32-web"`},
		{old: `"runtime": "surface-headless"`, new: `"runtime": "surface-wasm32-web"`},
		{old: `"host_evidence": {"level":"deterministic-headless","backend":"software-rgba","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false}`,
			new: `"host_evidence": {"level":"wasm32-web-compiler-owned-loader","backend":"node-surface-host","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false}`},
		{old: `{"name":"surface component app","kind":"app","path":"/tmp/surface-artifacts/surface-counter","ran":true,"pass":true,"exit_code":1,"expected_exit_code":1}`,
			new: `{"name":"surface wasm32-web component app","kind":"app","path":"node scripts/tools/web_run_module.mjs /tmp/surface-artifacts/surface-counter.wasm","ran":true,"pass":true,"exit_code":1,"expected_exit_code":1},
    {"name":"surface wasm32-web import validator","kind":"runtime","path":"go run ./tools/cmd/validate-wasm-imports --target wasm32-web /tmp/surface-artifacts/surface-counter.wasm","ran":true,"pass":true,"exit_code":0}`},
		{old: `{"kind":"component-app","path":"/tmp/surface-artifacts/surface-counter","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":49172}`,
			new: `{"kind":"component-app","path":"/tmp/surface-artifacts/surface-counter.wasm","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":7502}`},
		{old: `"surface headless runtime"`, new: `"surface wasm32-web runtime"`},
		{old: `"headless event dispatch"`, new: `"wasm32-web Surface Host ABI imports"`},
		{old: `"headless framebuffer checksum"`, new: `"wasm32-web framebuffer checksum evidence"`},
		{old: `"artifact_scan": {"root":"/tmp/surface-artifacts","files_checked":2`, new: `"artifact_scan": {"root":"/tmp/surface-artifacts","files_checked":3`},
	}
	for _, repl := range replacements {
		raw = strings.Replace(raw, repl.old, repl.new, 1)
	}
	raw = strings.Replace(raw, `,
    {"kind":"runner-trace","path":"/tmp/surface-artifacts/surface-runner-trace.json","sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","size":409}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"headless actual runner trace","kind":"positive","ran":true,"pass":true}`, ``, 1)
	raw = strings.Replace(raw, `"artifacts": [
    {"kind":"component-app","path":"/tmp/surface-artifacts/surface-counter.wasm","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":7502}
  ]`, `"artifacts": [
    {"kind":"component-app","path":"/tmp/surface-artifacts/surface-counter.wasm","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":7502},
    {"kind":"compiler-owned-loader","path":"/tmp/surface-artifacts/surface-counter.mjs","sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","size":4931},
    {"kind":"runner-trace","path":"/tmp/surface-artifacts/surface-runner-trace.json","sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","size":413}
  ]`, 1)
	raw = strings.Replace(raw, `{"order":2,"width":320,"height":200,"stride":1280,"checksum":"2222222222222222222222222222222222222222222222222222222222222222","presented":true}`,
		`{"order":2,"width":320,"height":200,"stride":1280,"checksum":"2222222222222222222222222222222222222222222222222222222222222222","presented":true},
    {"order":3,"width":320,"height":200,"stride":1280,"checksum":"3333333333333333333333333333333333333333333333333333333333333333","presented":true},
    {"order":4,"width":320,"height":200,"stride":1280,"checksum":"4444444444444444444444444444444444444444444444444444444444444444","presented":true}`, 1)
	raw = strings.Replace(raw, `{"name":"wasm32-web framebuffer checksum evidence","kind":"positive","ran":true,"pass":true}`,
		`{"name":"wasm32-web framebuffer checksum evidence","kind":"positive","ran":true,"pass":true},
    {"name":"compiler-owned wasm Surface loader","kind":"positive","ran":true,"pass":true},
    {"name":"wasm32-web actual presented frame trace","kind":"positive","ran":true,"pass":true}`, 1)
	return []byte(raw)
}

func validWASM32WebBrowserCanvasSurfaceReportJSON() []byte {
	return []byte(`{
  "schema": "tetra.surface.runtime.v1",
  "status": "pass",
  "target": "wasm32-web",
  "host": "linux-x64",
  "runtime": "surface-wasm32-web",
  "surface_schema": "tetra.surface.v1",
  "host_abi": "tetra.surface.host-abi.v1",
  "host_evidence": {"level":"wasm32-web-browser-canvas-input","backend":"browser-canvas-rgba","framebuffer":true,"real_window":false,"native_input":true,"user_facing_platform_widgets":false},
  "source": "examples/surface_browser_counter.tetra",
  "processes": [
    {"name":"tetra build","kind":"build","path":"tetra build --target wasm32-web examples/surface_browser_counter.tetra -o /tmp/surface-artifacts/surface-browser-counter.wasm","ran":true,"pass":true,"exit_code":0},
    {"name":"surface wasm32-web browser canvas component app","kind":"app","path":"/usr/bin/chromium --headless <surface-browser-canvas-runner> wasm=/tmp/surface-artifacts/surface-browser-counter.wasm","ran":true,"pass":true,"exit_code":0,"expected_exit_code":0},
    {"name":"surface wasm32-web import validator","kind":"runtime","path":"go run ./tools/cmd/validate-wasm-imports --target wasm32-web /tmp/surface-artifacts/surface-browser-counter.wasm","ran":true,"pass":true,"exit_code":0},
    {"name":"surface wasm32-web browser canvas runtime","kind":"runtime","path":"Chromium fixture","ran":true,"pass":true,"exit_code":0}
  ],
  "artifacts": [
    {"kind":"component-app","path":"/tmp/surface-artifacts/surface-browser-counter.wasm","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":8604},
    {"kind":"compiler-owned-loader","path":"/tmp/surface-artifacts/surface-browser-counter.mjs","sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","size":4939},
    {"kind":"runner-trace","path":"/tmp/surface-artifacts/surface-runner-trace.json","sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","size":1184}
  ],
  "artifact_scan": {"root":"/tmp/surface-artifacts","files_checked":3,"forbidden_paths":[],"pass":true},
  "components": [
    {"id":"CounterApp","type":"examples.surface_browser_counter.CounterApp","bounds":{"x":0,"y":0,"w":400,"h":240},"abilities":["measure","layout","draw","event","focus","text","accessibility"],"state":{"count":"2","key_count":"1","width":"400","accessibility_role":"button"}},
    {"id":"CounterButton","type":"examples.surface_browser_counter.CounterButton","parent":"CounterApp","bounds":{"x":32,"y":88,"w":160,"h":48},"abilities":["measure","layout","draw","event","focus","text","accessibility"],"state":{"focused":"true","text_len_seen":"2"}}
  ],
  "events": [
    {"order":1,"kind":"mouse_up","target_component":"CounterButton","dispatch_path":["CounterApp","CounterButton"],"handled":true,"pass":true,"x":48,"y":96,"key":0,"width":320,"height":200,"timestamp_ms":0,"buffer_slots":[5,48,96,1,0,320,200,0,0],"before_state":{"CounterApp.count":"0"},"after_state":{"CounterApp.count":"1"}},
    {"order":2,"kind":"key_down","target_component":"CounterApp","dispatch_path":["CounterApp"],"handled":true,"pass":true,"x":0,"y":0,"key":32,"width":320,"height":200,"timestamp_ms":1,"buffer_slots":[6,0,0,0,32,320,200,1,0],"before_state":{"CounterApp.count":"1","CounterApp.key_count":"0"},"after_state":{"CounterApp.count":"2","CounterApp.key_count":"1"}},
    {"order":3,"kind":"resize","target_component":"CounterApp","dispatch_path":["CounterApp"],"handled":true,"pass":true,"x":0,"y":0,"key":0,"width":400,"height":240,"timestamp_ms":2,"buffer_slots":[2,0,0,0,0,400,240,2,0],"before_state":{"CounterApp.width":"320"},"after_state":{"CounterApp.width":"400"}},
    {"order":4,"kind":"text_input","target_component":"CounterButton","dispatch_path":["CounterApp","CounterButton"],"handled":true,"pass":true,"x":0,"y":0,"key":0,"width":400,"height":240,"timestamp_ms":3,"text_len":2,"text_bytes_hex":"4f4b","buffer_slots":[8,0,0,0,0,400,240,3,2],"before_state":{"CounterButton.text_len_seen":"0"},"after_state":{"CounterButton.text_len_seen":"2"}}
  ],
  "frames": [
    {"order":1,"width":320,"height":200,"stride":1280,"checksum":"1111111111111111111111111111111111111111111111111111111111111111","presented":true},
    {"order":5,"width":400,"height":240,"stride":1600,"checksum":"5555555555555555555555555555555555555555555555555555555555555555","presented":true}
  ],
  "state_transitions": [
    {"order":1,"component":"CounterApp","field":"count","before":"0","after":"1","cause":"mouse_up"},
    {"order":2,"component":"CounterApp","field":"key_count","before":"0","after":"1","cause":"key_down"},
    {"order":3,"component":"CounterApp","field":"width","before":"320","after":"400","cause":"resize"},
    {"order":4,"component":"CounterButton","field":"text_len_seen","before":"0","after":"2","cause":"text_input"}
  ],
  "cases": [
    {"name":"pure Tetra component app","kind":"positive","ran":true,"pass":true},
    {"name":"wasm32-web Surface Host ABI imports","kind":"positive","ran":true,"pass":true},
    {"name":"compiler-owned wasm Surface loader","kind":"positive","ran":true,"pass":true},
    {"name":"wasm32-web browser canvas surface","kind":"positive","ran":true,"pass":true},
    {"name":"wasm32-web browser canvas RGBA readback","kind":"positive","ran":true,"pass":true},
    {"name":"wasm32-web browser canvas pointer input","kind":"positive","ran":true,"pass":true},
    {"name":"wasm32-web browser canvas keyboard input","kind":"positive","ran":true,"pass":true},
    {"name":"wasm32-web browser canvas resize input","kind":"positive","ran":true,"pass":true},
    {"name":"wasm32-web browser canvas text input","kind":"positive","ran":true,"pass":true},
    {"name":"compiler-owned browser canvas Surface host","kind":"positive","ran":true,"pass":true},
    {"name":"host-provided pointer event dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"host event buffer poll_event","kind":"positive","ran":true,"pass":true},
    {"name":"pre/post event frame sequence","kind":"positive","ran":true,"pass":true},
    {"name":"component hierarchy dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component text input scalar dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"host text payload buffer","kind":"positive","ran":true,"pass":true},
    {"name":"component focus dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component accessibility metadata","kind":"positive","ran":true,"pass":true},
    {"name":"no legacy UI sidecar artifacts","kind":"positive","ran":true,"pass":true},
    {"name":"state transition","kind":"positive","ran":true,"pass":true},
    {"name":"reject legacy UI evidence","kind":"negative","ran":true,"pass":true,"expected_error":"legacy UI evidence rejected"}
  ]
}`)
}
