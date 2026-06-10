package surface

import (
	"encoding/json"
	"fmt"
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

func TestValidateSurfaceReleaseSummaryAcceptsBlockSystemAndMorphGateMetadata(t *testing.T) {
	raw := validSurfaceReleaseSummaryJSON()
	if err := ValidateReleaseSummary(raw); err != nil {
		t.Fatalf("ValidateReleaseSummary failed with Block-system/Morph gate metadata: %v\n%s", err, raw)
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
		{
			name: "missing block system",
			mutate: func(raw string) string {
				return strings.Replace(raw, `  "block_system": "block-system",
`, ``, 1)
			},
			want: "block_system",
		},
		{
			name: "wrong block system gate",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"block_system_gate": "tetra.surface.block-system.gate.v1"`, `"block_system_gate": "tetra.surface.block-system.fake"`, 1)
			},
			want: "block_system_gate",
		},
		{
			name: "missing morph",
			mutate: func(raw string) string {
				return strings.Replace(raw, `  "morph": "morph-capsule",
`, ``, 1)
			},
			want: "morph",
		},
		{
			name: "wrong morph gate",
			mutate: func(raw string) string {
				return strings.Replace(raw, `"morph_gate": "tetra.surface.morph.gate.v1"`, `"morph_gate": "tetra.surface.morph.invalid"`, 1)
			},
			want: "morph_gate",
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

func TestValidateSurfaceReleaseSummaryRejectsStaleProducerMetadata(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "missing producer",
			mutate: func(report map[string]any) {
				delete(report, "producer")
			},
			want: "producer",
		},
		{
			name: "stale git head",
			mutate: func(report map[string]any) {
				report["git_head"] = "unknown"
			},
			want: "git_head",
		},
		{
			name: "missing command line",
			mutate: func(report map[string]any) {
				delete(report, "command_line")
			},
			want: "command_line",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var report map[string]any
			if err := json.Unmarshal(validSurfaceReleaseSummaryJSON(), &report); err != nil {
				t.Fatalf("decode release summary: %v", err)
			}
			tc.mutate(report)
			raw, err := json.Marshal(report)
			if err != nil {
				t.Fatalf("marshal release summary: %v", err)
			}
			err = ValidateReleaseSummary(raw)
			if err == nil {
				t.Fatalf("expected stale producer metadata to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

func TestHeadlessReleaseRequiresBuiltBinary(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `    {"name":"tetra build","kind":"build","path":"tetra build --target linux-x64 examples/surface_counter.tetra -o /tmp/surface-artifacts/surface-counter","ran":true,"pass":true,"exit_code":0},
`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected headless report without build process to fail")
	}
	if !strings.Contains(err.Error(), "build process") {
		t.Fatalf("error = %v, want build process diagnostic", err)
	}
}

func TestHeadlessRunnerTraceMatchesReport(t *testing.T) {
	raw := mutateHeadlessSurfaceReport(t, func(report map[string]any) {
		frames := report["frames"].([]any)
		first := frames[0].(map[string]any)
		second := frames[1].(map[string]any)
		second["checksum"] = first["checksum"]
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected unchanged pre/post headless frame checksum evidence to fail")
	}
	if !strings.Contains(err.Error(), "pre/post") {
		t.Fatalf("error = %v, want pre/post frame diagnostic", err)
	}
}

func TestHeadlessRejectsMetadataOnlyFrame(t *testing.T) {
	raw := mutateHeadlessSurfaceReport(t, func(report map[string]any) {
		frames := report["frames"].([]any)
		first := frames[0].(map[string]any)
		first["checksum"] = ""
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected metadata-only headless frame to fail")
	}
	if !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("error = %v, want checksum diagnostic", err)
	}
}

func TestHeadlessNoLegacySidecars(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()),
		`    {"name":"no legacy UI sidecar artifacts","kind":"positive","ran":true,"pass":true},
`,
		``,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing no-legacy sidecar case to fail")
	}
	if !strings.Contains(err.Error(), "no legacy UI sidecar") {
		t.Fatalf("error = %v, want no legacy sidecar diagnostic", err)
	}
}

func mutateHeadlessSurfaceReport(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode headless report: %v", err)
	}
	mutate(report)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal headless report: %v", err)
	}
	return raw
}

func TestValidateSurfaceTextInputReportAcceptsProductionBaseline(t *testing.T) {
	raw := validSurfaceTextInputReportJSON()
	if err := ValidateTextInputReport(raw); err != nil {
		t.Fatalf("ValidateTextInputReport failed: %v\n%s", err, raw)
	}
}

func TestValidateSurfaceTextInputReportRequiresTextPipelineEvidence(t *testing.T) {
	raw := mutateSurfaceTextInputReportJSON(t, func(report map[string]any) {
		delete(report, "text_pipeline")
	})
	err := ValidateTextInputReport(raw)
	if err == nil {
		t.Fatalf("expected text-input report without text_pipeline to fail")
	}
	if !strings.Contains(err.Error(), "text_pipeline") {
		t.Fatalf("error = %v, want text_pipeline diagnostic", err)
	}
}

func TestValidateSurfaceTextInputReportAcceptsScopedTextPipelineEvidence(t *testing.T) {
	raw := mutateSurfaceTextInputReportJSON(t, func(report map[string]any) {
		report["text_pipeline"] = validSurfaceTextPipelineEvidence()
	})
	if err := ValidateTextInputReport(raw); err != nil {
		t.Fatalf("ValidateTextInputReport with text_pipeline failed: %v\n%s", err, raw)
	}
}

func TestValidateSurfaceTextInputReportRejectsTextPipelineOverclaims(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "full unicode editor semantics",
			mutate: func(pipeline map[string]any) {
				scope := pipeline["shaping_scope"].(map[string]any)
				scope["tier"] = "tier3-full-editor"
				scope["full_unicode_editor_semantics"] = true
			},
			want: "full Unicode editor semantics",
		},
		{
			name: "missing font fallback",
			mutate: func(pipeline map[string]any) {
				pipeline["font_fallbacks"] = []any{}
			},
			want: "font fallback",
		},
		{
			name: "unbounded glyph cache",
			mutate: func(pipeline map[string]any) {
				pipeline["bounded_caches"] = false
				caches := pipeline["glyph_caches"].([]any)
				caches[0].(map[string]any)["bounded"] = false
			},
			want: "bounded",
		},
		{
			name: "unsupported scripts nonclaim removed",
			mutate: func(pipeline map[string]any) {
				scope := pipeline["shaping_scope"].(map[string]any)
				scope["unsupported_scripts"] = []any{}
				pipeline["nonclaims"] = []any{"rich text"}
			},
			want: "unsupported scripts",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := mutateSurfaceTextInputReportJSON(t, func(report map[string]any) {
				pipeline := validSurfaceTextPipelineEvidence()
				tc.mutate(pipeline)
				report["text_pipeline"] = pipeline
			})
			err := ValidateTextInputReport(raw)
			if err == nil {
				t.Fatalf("expected text_pipeline overclaim to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

func TestValidateSurfaceTextInputReportRequiresTextEditingEvidence(t *testing.T) {
	raw := mutateSurfaceTextInputReportJSON(t, func(report map[string]any) {
		delete(report, "text_editing")
	})
	err := ValidateTextInputReport(raw)
	if err == nil {
		t.Fatalf("expected text-input report without text_editing to fail")
	}
	if !strings.Contains(err.Error(), "text_editing") {
		t.Fatalf("error = %v, want text_editing diagnostic", err)
	}
}

func TestValidateSurfaceTextInputReportAcceptsScopedTextEditingEvidence(t *testing.T) {
	raw := mutateSurfaceTextInputReportJSON(t, func(report map[string]any) {
		report["text_editing"] = validSurfaceTextEditingEvidence("headless")
	})
	if err := ValidateTextInputReport(raw); err != nil {
		t.Fatalf("ValidateTextInputReport with text_editing failed: %v\n%s", err, raw)
	}
}

func TestValidateSurfaceTextInputReportRejectsTextEditingOverclaims(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "missing target IME trace",
			mutate: func(editing map[string]any) {
				editing["ime_traces"] = []any{}
			},
			want: "IME target trace",
		},
		{
			name: "borrowed text buffer crosses host boundary",
			mutate: func(editing map[string]any) {
				host := editing["host_boundary"].(map[string]any)
				host["borrowed_text_buffer_crosses_host"] = true
			},
			want: "borrowed text buffer",
		},
		{
			name: "rich text claim",
			mutate: func(editing map[string]any) {
				editing["rich_text"] = true
				editing["nonclaims"] = []any{"native platform text controls"}
			},
			want: "rich text",
		},
		{
			name: "missing undo unit boundaries",
			mutate: func(editing map[string]any) {
				editing["undo_units"] = []any{}
			},
			want: "undo",
		},
		{
			name: "clipboard borrowed view",
			mutate: func(editing map[string]any) {
				transfers := editing["clipboard_transfers"].([]any)
				transfers[0].(map[string]any)["owned_copy"] = false
				transfers[0].(map[string]any)["borrowed_view"] = true
			},
			want: "clipboard",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := mutateSurfaceTextInputReportJSON(t, func(report map[string]any) {
				editing := validSurfaceTextEditingEvidence("headless")
				tc.mutate(editing)
				report["text_editing"] = editing
			})
			err := ValidateTextInputReport(raw)
			if err == nil {
				t.Fatalf("expected text_editing overclaim to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
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

func validSurfaceTextEditingEvidence(target string) map[string]any {
	return map[string]any{
		"schema":   "tetra.surface.text-editing.v1",
		"level":    "production-editing-basics-v1",
		"target":   target,
		"producer": "tools/cmd/surface-runtime-smoke",
		"editable_blocks": []any{
			map[string]any{"id": "ReleaseTextBox", "kind": "TextBox", "storage": "owned-utf8-byte-buffer", "forms_safe": true, "command_palette_search_safe": true, "max_bytes": float64(1024), "utf8_validation": true},
		},
		"edit_operations": []any{
			map[string]any{"order": float64(1), "action": "insert_text", "target": "ReleaseTextBox", "before_text_len": float64(0), "after_text_len": float64(3), "before_caret": float64(0), "after_caret": float64(3), "selection_before": map[string]any{"anchor": float64(0), "focus": float64(0)}, "selection_after": map[string]any{"anchor": float64(3), "focus": float64(3)}, "undo_unit_id": "insert-ada", "checksum": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
			map[string]any{"order": float64(2), "action": "move_caret_left", "target": "ReleaseTextBox", "before_text_len": float64(3), "after_text_len": float64(3), "before_caret": float64(3), "after_caret": float64(2), "selection_before": map[string]any{"anchor": float64(3), "focus": float64(3)}, "selection_after": map[string]any{"anchor": float64(2), "focus": float64(2)}, "undo_unit_id": "navigation-left", "checksum": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
			map[string]any{"order": float64(3), "action": "replace_selection", "target": "ReleaseTextBox", "before_text_len": float64(5), "after_text_len": float64(4), "before_caret": float64(1), "after_caret": float64(2), "selection_before": map[string]any{"anchor": float64(1), "focus": float64(4)}, "selection_after": map[string]any{"anchor": float64(2), "focus": float64(2)}, "undo_unit_id": "replace-selection", "checksum": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
			map[string]any{"order": float64(4), "action": "composition_commit", "target": "ReleaseTextBox", "before_text_len": float64(4), "after_text_len": float64(5), "before_caret": float64(4), "after_caret": float64(5), "selection_before": map[string]any{"anchor": float64(4), "focus": float64(4)}, "selection_after": map[string]any{"anchor": float64(5), "focus": float64(5)}, "undo_unit_id": "ime-commit", "checksum": "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
			map[string]any{"order": float64(5), "action": "clipboard_write", "target": "ReleaseTextBox", "before_text_len": float64(5), "after_text_len": float64(5), "before_caret": float64(5), "after_caret": float64(5), "selection_before": map[string]any{"anchor": float64(0), "focus": float64(5)}, "selection_after": map[string]any{"anchor": float64(0), "focus": float64(5)}, "undo_unit_id": "clipboard-copy", "checksum": "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},
			map[string]any{"order": float64(6), "action": "clipboard_read", "target": "ReleaseTextBox", "before_text_len": float64(0), "after_text_len": float64(5), "before_caret": float64(0), "after_caret": float64(5), "selection_before": map[string]any{"anchor": float64(0), "focus": float64(0)}, "selection_after": map[string]any{"anchor": float64(5), "focus": float64(5)}, "undo_unit_id": "clipboard-paste", "checksum": "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		},
		"selection_model": map[string]any{
			"caret_movement":        []any{"left", "right", "home", "end"},
			"selection_replacement": true,
			"scalar_boundary_clamp": true,
			"caret_rects":           []any{map[string]any{"x": float64(32), "y": float64(64), "w": float64(2), "h": float64(18)}},
			"selection_rects":       []any{map[string]any{"x": float64(32), "y": float64(64), "w": float64(24), "h": float64(18)}},
		},
		"ime_traces": []any{
			map[string]any{"target": target, "start": true, "update": true, "commit": true, "cancel": true, "event_count": float64(4), "composition_span": map[string]any{"kind": "composition", "byte_start": float64(0), "byte_end": float64(3), "scalar_start": float64(0), "scalar_end": float64(3), "rect": map[string]any{"x": float64(32), "y": float64(64), "w": float64(24), "h": float64(18)}}, "committed_text_owned_copy": true},
		},
		"clipboard_transfers": []any{
			map[string]any{"direction": "write", "host_abi": "__tetra_surface_clipboard_write_text", "bytes": float64(5), "utf8_valid": true, "owned_copy": true, "borrowed_view": false, "checksum": "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
			map[string]any{"direction": "read", "host_abi": "__tetra_surface_clipboard_read_text_into", "bytes": float64(5), "utf8_valid": true, "owned_copy": true, "borrowed_view": false, "checksum": "sha256:2222222222222222222222222222222222222222222222222222222222222222"},
		},
		"undo_units": []any{
			map[string]any{"id": "insert-ada", "operation_orders": []any{float64(1)}, "boundary": "text-input-operation", "reversible": true, "coalesced": false},
			map[string]any{"id": "navigation-left", "operation_orders": []any{float64(2)}, "boundary": "caret-navigation", "reversible": true, "coalesced": false},
			map[string]any{"id": "replace-selection", "operation_orders": []any{float64(3)}, "boundary": "selection-replacement", "reversible": true, "coalesced": false},
			map[string]any{"id": "ime-commit", "operation_orders": []any{float64(4)}, "boundary": "composition-commit", "reversible": true, "coalesced": false},
			map[string]any{"id": "clipboard-copy", "operation_orders": []any{float64(5)}, "boundary": "clipboard-copy", "reversible": true, "coalesced": false},
			map[string]any{"id": "clipboard-paste", "operation_orders": []any{float64(6)}, "boundary": "clipboard-paste", "reversible": true, "coalesced": false},
		},
		"validation_diagnostics": []any{
			map[string]any{"name": "invalid UTF-8 rejected", "ran": true, "pass": true},
			map[string]any{"name": "borrowed text buffer rejected at host boundary", "ran": true, "pass": true},
			map[string]any{"name": "IME claim without target trace rejected", "ran": true, "pass": true},
			map[string]any{"name": "rich text claim rejected", "ran": true, "pass": true},
		},
		"host_boundary": map[string]any{
			"copy_safe":                         true,
			"clipboard_owned_copy":              true,
			"composition_owned_copy":            true,
			"borrowed_text_buffer_crosses_host": false,
		},
		"forms_safe":                  true,
		"command_palette_search_safe": true,
		"rich_text":                   false,
		"nonclaims": []any{
			"rich text",
			"full editor-grade text semantics",
			"native platform text controls",
		},
		"negative_guards": map[string]any{
			"ime_without_target_trace_rejected": true,
			"borrowed_text_buffer_rejected":     true,
			"rich_text_claim_rejected":          true,
			"unsafe_clipboard_alias_rejected":   true,
			"invalid_utf8_rejected":             true,
		},
	}
}

func mutateSurfaceTextInputReportJSON(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validSurfaceTextInputReportJSON(), &report); err != nil {
		t.Fatalf("decode text-input report: %v", err)
	}
	mutate(report)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal text-input report: %v", err)
	}
	return raw
}

func validSurfaceTextPipelineEvidence() map[string]any {
	return map[string]any{
		"schema":                        "tetra.surface.text-pipeline.v1",
		"level":                         "scoped-latin-utf8-text-pipeline-v1",
		"engine":                        "deterministic-tetra-text-shaper",
		"platform_widget_text_controls": false,
		"font_manifest": []any{
			map[string]any{"id": "tetra-ui-regular", "family": "Tetra UI", "style": "normal", "weight": float64(400), "source": "embedded:tetra-ui-regular", "sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "bytes": float64(32768)},
			map[string]any{"id": "noto-sans-fallback", "family": "Noto Sans", "style": "normal", "weight": float64(400), "source": "system:fontconfig/noto-sans", "sha256": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "bytes": float64(65536)},
		},
		"font_fallbacks": []any{
			map[string]any{"id": "release-fallback", "requested_family": "Tetra UI", "resolved_family": "Noto Sans", "chain": []any{"Tetra UI", "Noto Sans", "monospace"}, "missing_glyphs": float64(0), "coverage": "latin-plus-basic-utf8-smoke"},
		},
		"glyph_runs": []any{
			map[string]any{"id": "latin-run", "font_family": "Tetra UI", "script": "Latin", "direction": "ltr", "shaping": "tier1-latin-simple", "text_len": float64(5), "byte_start": float64(0), "byte_end": float64(5), "scalar_start": float64(0), "scalar_end": float64(5), "glyph_count": float64(5), "glyph_ids": []any{float64(36), float64(69), float64(70), float64(32), float64(71)}, "advances": []any{float64(8), float64(8), float64(8), float64(4), float64(8)}, "clusters": []any{float64(0), float64(1), float64(2), float64(3), float64(4)}, "baseline": float64(14), "checksum": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
			map[string]any{"id": "fallback-run", "font_family": "Noto Sans", "script": "Common", "direction": "ltr", "shaping": "tier1-fallback-simple", "text_len": float64(1), "byte_start": float64(5), "byte_end": float64(7), "scalar_start": float64(5), "scalar_end": float64(6), "glyph_count": float64(1), "glyph_ids": []any{float64(9731)}, "advances": []any{float64(9)}, "clusters": []any{float64(5)}, "baseline": float64(14), "checksum": "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
		},
		"glyph_caches": []any{
			map[string]any{"id": "release-glyph-cache", "strategy": "bounded-lru", "budget_bytes": float64(65536), "used_bytes": float64(8192), "entry_count": float64(24), "eviction": "lru", "bounded": true},
		},
		"cache_budget_bytes":       float64(65536),
		"glyph_cache_budget_bytes": float64(65536),
		"glyph_cache_used_bytes":   float64(8192),
		"bounded_caches":           true,
		"cache_eviction":           "lru",
		"unicode_boundaries": map[string]any{
			"utf8_storage":        true,
			"scalar_boundaries":   true,
			"cluster_boundaries":  true,
			"latin_tier":          true,
			"combining_marks":     false,
			"bidi":                false,
			"unsupported_scripts": []any{"Arabic", "Devanagari", "Thai"},
			"boundary_cases":      []any{"ASCII insertion", "UTF-8 scalar insertion", "cluster caret clamp"},
		},
		"shaping_scope": map[string]any{
			"tier":                          "tier1-latin-utf8",
			"supported_scripts":             []any{"Latin", "Common"},
			"unsupported_scripts":           []any{"Arabic", "Devanagari", "Thai"},
			"engine_decision":               "deterministic embedded shaper until HarfBuzz-class evidence exists",
			"full_unicode_editor_semantics": false,
			"bidi":                          false,
			"combining_marks":               false,
			"system_library_integration":    "not required for Tier 1; future HarfBuzz-class gate",
			"platform_widgets":              false,
		},
		"measurements": []any{
			map[string]any{"id": "release-label-measure", "block_id": float64(1), "text_len": float64(18), "font_family": "Tetra UI", "font_weight": float64(400), "font_size": float64(14), "line_height": float64(18), "max_width": float64(120), "measured": map[string]any{"w": float64(108), "h": float64(18)}, "line_count": float64(1), "wrap": "none", "overflow": "clip", "ellipsis": false, "ellipsized_text_len": float64(18), "align": "start", "quality": "deterministic-metrics-v1", "checksum": "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},
			map[string]any{"id": "release-ellipsis-measure", "block_id": float64(2), "text_len": float64(32), "font_family": "Tetra UI", "font_weight": float64(400), "font_size": float64(14), "line_height": float64(18), "max_width": float64(96), "measured": map[string]any{"w": float64(96), "h": float64(36)}, "line_count": float64(2), "wrap": "word", "overflow": "ellipsis", "ellipsis": true, "ellipsized_text_len": float64(20), "align": "start", "quality": "deterministic-metrics-v1", "checksum": "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		},
		"measurement_consistency": map[string]any{
			"same_input_same_metrics":     true,
			"target_independent_baseline": true,
			"max_delta_px":                float64(0),
			"cases":                       []any{"latin measurement repeat", "fallback measurement repeat", "ellipsis measurement repeat"},
		},
		"layout": map[string]any{
			"wrap":        true,
			"ellipsis":    true,
			"alignment":   []any{"start", "center", "end"},
			"baseline":    true,
			"line_height": true,
		},
		"caret_rects": []any{
			map[string]any{"x": float64(32), "y": float64(64), "w": float64(2), "h": float64(18)},
		},
		"selection_rects": []any{
			map[string]any{"x": float64(32), "y": float64(64), "w": float64(18), "h": float64(18)},
		},
		"ime_composition_spans": []any{
			map[string]any{"kind": "composition", "byte_start": float64(0), "byte_end": float64(3), "scalar_start": float64(0), "scalar_end": float64(3), "rect": map[string]any{"x": float64(32), "y": float64(64), "w": float64(24), "h": float64(18)}},
		},
		"nonclaims": []any{
			"full Unicode editor semantics",
			"bidi production shaping",
			"complex script shaping without HarfBuzz-class evidence",
			"platform widget text controls",
		},
		"negative_guards": map[string]any{
			"full_unicode_editor_without_tests_rejected": true,
			"missing_font_fallback_rejected":             true,
			"unbounded_glyph_cache_rejected":             true,
			"platform_widget_text_controls_rejected":     true,
		},
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

func TestValidateSurfaceReleaseAccessibilityRequiresProductionTargetEvidence(t *testing.T) {
	raw := validLinuxReleaseAccessibilitySurfaceReportJSON(t, func(report map[string]any) {
		delete(report, "accessibility_target")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected release accessibility report without accessibility_target to fail")
	}
	if !strings.Contains(err.Error(), "accessibility_target") {
		t.Fatalf("error = %v, want accessibility_target diagnostic", err)
	}
}

func TestValidateSurfaceBrowserReleaseReport(t *testing.T) {
	raw := validWASM32WebReleaseBrowserSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateSurfaceBrowserReleaseRequiresFirstClassBrowserCanvasTargetEvidence(t *testing.T) {
	raw := validWASM32WebReleaseBrowserSurfaceReportJSON(t, func(report map[string]any) {
		delete(report, "browser_canvas_target")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected browser release report without browser_canvas_target to fail")
	}
	if !strings.Contains(err.Error(), "browser_canvas_target") {
		t.Fatalf("error = %v, want browser_canvas_target diagnostic", err)
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
		{
			name: "dom snapshot renderer promotion",
			mutate: func(report map[string]any) {
				target := report["browser_canvas_target"].(map[string]any)
				target["dom_ui"] = true
				target["negative_guards"].(map[string]any)["dom_snapshot_renderer_rejected"] = false
			},
			want: "DOM snapshot",
		},
		{
			name: "user js command dispatch promotion",
			mutate: func(report map[string]any) {
				target := report["browser_canvas_target"].(map[string]any)
				target["user_js_app_logic"] = true
				target["negative_guards"].(map[string]any)["user_js_command_dispatch_rejected"] = false
			},
			want: "user JS",
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

func TestValidateSurfaceLinuxReleaseWindowRequiresProductionHostAdapterEvidence(t *testing.T) {
	raw := validLinuxReleaseWindowSurfaceReportJSON(t, func(report map[string]any) {
		delete(report, "linux_host_adapter")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected linux release-window report without linux_host_adapter evidence to fail")
	}
	if !strings.Contains(err.Error(), "linux_host_adapter") {
		t.Fatalf("error = %v, want linux_host_adapter diagnostic", err)
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
			name: "blocked display counted as pass",
			mutate: func(report map[string]any) {
				host := report["host_evidence"].(map[string]any)
				host["real_window"] = false
				adapter := report["linux_host_adapter"].(map[string]any)
				adapter["real_window"] = false
				adapter["target_host_trace"] = ""
				guards := adapter["negative_guards"].(map[string]any)
				guards["blocked_display_pass_rejected"] = false
			},
			want: "blocked display",
		},
		{
			name: "missing app shell",
			mutate: func(report map[string]any) {
				report["linux_host_adapter"].(map[string]any)["app_shell"] = false
			},
			want: "app_shell",
		},
		{
			name: "missing packaging scope",
			mutate: func(report map[string]any) {
				packaging := report["linux_host_adapter"].(map[string]any)["packaging"].(map[string]any)
				packaging["scope"] = ""
			},
			want: "packaging scope",
		},
		{
			name: "missing target-host trace",
			mutate: func(report map[string]any) {
				report["linux_host_adapter"].(map[string]any)["target_host_trace"] = ""
			},
			want: "target_host_trace",
		},
		{
			name: "missing IME evidence",
			mutate: func(report map[string]any) {
				report["linux_host_adapter"].(map[string]any)["ime"] = false
			},
			want: "IME",
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
		{
			name: "full screen reader overclaim",
			mutate: func(report map[string]any) {
				report["accessibility_target"].(map[string]any)["full_screen_reader_claim"] = true
			},
			want: "screen-reader",
		},
		{
			name: "desktop aria bridge used as proof",
			mutate: func(report map[string]any) {
				target := report["accessibility_target"].(map[string]any)
				target["negative_guards"].(map[string]any)["aria_dom_desktop_bridge_rejected"] = false
			},
			want: "ARIA/DOM",
		},
		{
			name: "target count does not match tree",
			mutate: func(report map[string]any) {
				report["accessibility_target"].(map[string]any)["named_node_count"] = 0
			},
			want: "named_node_count",
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

func TestValidateReportAcceptsBlockGraphEvidence(t *testing.T) {
	raw := validHeadlessBlockGraphSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportRejectsIncompleteBlockGraphEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "manual bookkeeping",
			mutate: func(report *Report) {
				report.BlockGraph.ManualBookkeeping = true
			},
			want: "manual_bookkeeping",
		},
		{
			name: "missing duplicate guard",
			mutate: func(report *Report) {
				report.BlockGraph.Invariants.DuplicateIDRejected = false
			},
			want: "duplicate_id",
		},
		{
			name: "missing parent",
			mutate: func(report *Report) {
				report.BlockGraph.Nodes[4].ParentID = 99
			},
			want: "parent_id",
		},
		{
			name: "cycle",
			mutate: func(report *Report) {
				report.BlockGraph.Nodes[1].ParentID = 5
			},
			want: "cycle",
		},
		{
			name: "child order",
			mutate: func(report *Report) {
				report.BlockGraph.ChildOrders[1].Children = []int{3, 5, 4}
			},
			want: "child_orders",
		},
		{
			name: "focus order",
			mutate: func(report *Report) {
				report.BlockGraph.FocusOrder = []int{5, 4}
			},
			want: "focus_order",
		},
		{
			name: "hit path",
			mutate: func(report *Report) {
				report.BlockGraph.HitTests[0].Path = []int{1, 5}
			},
			want: "hit_tests",
		},
		{
			name: "accessibility order",
			mutate: func(report *Report) {
				report.BlockGraph.AccessibilityOrder = []int{4, 5}
			},
			want: "accessibility_order",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockGraphSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected block graph %s evidence to fail", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

func TestValidateReportRejectsBlockGraphWithoutABIAndResolvedSceneContract(t *testing.T) {
	raw := mutateBlockGraphJSONForTest(t, func(blockGraph map[string]any) {
		delete(blockGraph, "abi")
		delete(blockGraph, "resolved_scene")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected Block graph without ABI/resolved scene contract to fail")
	}
	for _, want := range []string{"abi", "resolved_scene"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateReportRejectsBlockGraphResolvedSceneOrderMismatch(t *testing.T) {
	raw := mutateBlockGraphJSONForTest(t, func(blockGraph map[string]any) {
		if scene, ok := blockGraph["resolved_scene"].(map[string]any); ok {
			scene["draw_order"] = []any{5, 4, 3, 2, 1}
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected Block graph resolved_scene order mismatch to fail")
	}
	if !strings.Contains(err.Error(), "resolved_scene") || !strings.Contains(err.Error(), "draw_order") {
		t.Fatalf("error = %v, want resolved_scene draw_order diagnostic", err)
	}
}

func mutateBlockGraphJSONForTest(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessBlockGraphSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode Block graph report: %v", err)
	}
	blockGraph, ok := report["block_graph"].(map[string]any)
	if !ok {
		t.Fatalf("block_graph missing or wrong type: %#v", report["block_graph"])
	}
	mutate(blockGraph)
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal Block graph report: %v", err)
	}
	return raw
}

func TestValidateReportAcceptsBlockPaintEvidence(t *testing.T) {
	raw := validHeadlessBlockPaintSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportRejectsIncompleteBlockPaintEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "missing fill",
			mutate: func(report *Report) {
				report.VisualFeatures = removeString(report.VisualFeatures, "fill")
			},
			want: "fill",
		},
		{
			name: "missing border",
			mutate: func(report *Report) {
				report.PaintLayers = removePaintLayerKind(report.PaintLayers, "border")
			},
			want: "border",
		},
		{
			name: "missing radius",
			mutate: func(report *Report) {
				report.PaintLayers[0].Radius = 0
				report.PaintLayers[1].Radius = 0
				report.PaintLayers[2].Radius = 0
				report.PaintLayers[4].Radius = 0
			},
			want: "radius",
		},
		{
			name: "missing shadow",
			mutate: func(report *Report) {
				report.PaintCommands = removePaintCommand(report.PaintCommands, "shadow")
			},
			want: "shadow",
		},
		{
			name: "missing outline",
			mutate: func(report *Report) {
				report.VisualFeatures = removeString(report.VisualFeatures, "outline")
			},
			want: "outline",
		},
		{
			name: "unsupported blur",
			mutate: func(report *Report) {
				report.PaintUnsupportedBlur = true
				report.VisualFeatures = append(report.VisualFeatures, "blur")
			},
			want: "unsupported blur",
		},
		{
			name: "command order",
			mutate: func(report *Report) {
				report.PaintCommands[0], report.PaintCommands[1] = report.PaintCommands[1], report.PaintCommands[0]
			},
			want: "paint_commands",
		},
		{
			name: "unchanged frames",
			mutate: func(report *Report) {
				report.Frames[1].Checksum = report.Frames[0].Checksum
			},
			want: "paint frame",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockPaintSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected block paint %s evidence to fail", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

func TestValidateReportRejectsBlockPaintWithoutRendererSceneContract(t *testing.T) {
	raw := mutateBlockPaintJSONForTest(t, func(report map[string]any) {
		delete(report, "renderer_scene")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected Block paint report without renderer_scene to fail")
	}
	if !strings.Contains(err.Error(), "renderer_scene") {
		t.Fatalf("error = %v, want renderer_scene diagnostic", err)
	}
}

func TestValidateReportRejectsBlockPaintRendererSceneMissingProductionCommands(t *testing.T) {
	raw := mutateBlockPaintJSONForTest(t, func(report map[string]any) {
		scene, ok := report["renderer_scene"].(map[string]any)
		if !ok {
			return
		}
		commands, _ := scene["commands"].([]any)
		filtered := make([]any, 0, len(commands))
		for _, entry := range commands {
			command, ok := entry.(map[string]any)
			if !ok {
				filtered = append(filtered, entry)
				continue
			}
			switch command["command"] {
			case "image", "text", "clip", "transform":
				continue
			default:
				filtered = append(filtered, entry)
			}
		}
		scene["commands"] = filtered
		scene["command_count"] = len(filtered)
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected Block paint renderer_scene missing production commands to fail")
	}
	for _, want := range []string{"renderer_scene", "image"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateReportRejectsBlockPaintRendererSceneUnsupportedVisualClaim(t *testing.T) {
	raw := mutateBlockPaintJSONForTest(t, func(report map[string]any) {
		scene, ok := report["renderer_scene"].(map[string]any)
		if !ok {
			return
		}
		scene["unsupported_visuals"] = []any{
			map[string]any{"feature": "blur", "rejected": false, "reason": ""},
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected Block paint renderer_scene unsupported visual claim to fail")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("error = %v, want unsupported diagnostic", err)
	}
}

func mutateBlockPaintJSONForTest(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessBlockPaintSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode Block paint report: %v", err)
	}
	mutate(report)
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal Block paint report: %v", err)
	}
	return raw
}

func TestValidateReportAcceptsBlockTextEvidence(t *testing.T) {
	raw := validHeadlessBlockTextSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportRejectsIncompleteBlockTextEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "missing measurement",
			mutate: func(report *Report) {
				report.TextMeasurements = nil
			},
			want: "text_measurements",
		},
		{
			name: "wrap ellipsis mismatch",
			mutate: func(report *Report) {
				report.TextMeasurements[0].EllipsizedTextLen = report.TextMeasurements[0].TextLen
			},
			want: "ellipsis",
		},
		{
			name: "missing fallback chain",
			mutate: func(report *Report) {
				report.FontFallbacks = nil
			},
			want: "font_fallback",
		},
		{
			name: "unbounded glyph cache",
			mutate: func(report *Report) {
				report.GlyphCaches[0].Bounded = false
			},
			want: "glyph cache",
		},
		{
			name: "missing render command",
			mutate: func(report *Report) {
				report.TextRenderCommands = nil
			},
			want: "text render",
		},
		{
			name: "unchanged frames",
			mutate: func(report *Report) {
				report.Frames[1].Checksum = report.Frames[0].Checksum
			},
			want: "text frame",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockTextSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected block text %s evidence to fail", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

func TestValidateReportAcceptsBlockLayoutEvidence(t *testing.T) {
	raw := validHeadlessBlockLayoutSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportRejectsIncompleteBlockLayoutEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "missing grid",
			mutate: func(report *Report) {
				report.LayoutPasses = removeBlockLayoutPassMode(report.LayoutPasses, "grid")
				report.LayoutFeatures = removeString(report.LayoutFeatures, "grid")
			},
			want: "grid",
		},
		{
			name: "missing dock",
			mutate: func(report *Report) {
				report.LayoutPasses = removeBlockLayoutPassMode(report.LayoutPasses, "dock")
				report.LayoutFeatures = removeString(report.LayoutFeatures, "dock")
			},
			want: "dock",
		},
		{
			name: "missing scroll",
			mutate: func(report *Report) {
				report.LayoutScrolls = nil
				report.LayoutFeatures = removeString(report.LayoutFeatures, "scroll")
			},
			want: "scroll",
		},
		{
			name: "missing resize",
			mutate: func(report *Report) {
				for i := range report.LayoutPasses {
					report.LayoutPasses[i].Resize = false
				}
				report.LayoutFeatures = removeString(report.LayoutFeatures, "resize")
			},
			want: "resize",
		},
		{
			name: "unsupported css flexbox",
			mutate: func(report *Report) {
				report.LayoutUnsupportedCSSFlexbox = true
			},
			want: "CSS flexbox",
		},
		{
			name: "missing min max",
			mutate: func(report *Report) {
				report.LayoutConstraints[0].Min = SizeReport{}
				report.LayoutConstraints[0].Max = SizeReport{}
				report.LayoutFeatures = removeString(removeString(report.LayoutFeatures, "min"), "max")
			},
			want: "min",
		},
		{
			name: "unchanged frames",
			mutate: func(report *Report) {
				report.Frames[1].Checksum = report.Frames[0].Checksum
			},
			want: "layout frame",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockLayoutSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected block layout %s evidence to fail", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

func TestValidateReportRequiresLayoutEngineEvidence(t *testing.T) {
	raw := mutateBlockLayoutReportJSON(t, func(report map[string]any) {
		delete(report, "layout_engine")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected block layout report without layout_engine to fail")
	}
	if !strings.Contains(err.Error(), "layout_engine") {
		t.Fatalf("error = %v, want layout_engine diagnostic", err)
	}
}

func TestValidateReportAcceptsLayoutEngineEvidence(t *testing.T) {
	raw := mutateBlockLayoutReportJSON(t, func(report map[string]any) {
		report["layout_engine"] = validSurfaceLayoutEngineEvidence()
	})
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport with layout_engine failed: %v\n%s", err, raw)
	}
}

func TestValidateReportRejectsLayoutEngineOverclaims(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "css parity claim",
			mutate: func(engine map[string]any) {
				engine["css_flexbox_grid_parity"] = true
				engine["nonclaims"] = []any{"native platform widgets"}
			},
			want: "CSS",
		},
		{
			name: "accidental overflow hidden",
			mutate: func(engine map[string]any) {
				overflow := engine["overflow_rules"].(map[string]any)
				overflow["accidental_hidden"] = true
			},
			want: "overflow",
		},
		{
			name: "unbounded cache",
			mutate: func(engine map[string]any) {
				cache := engine["cache_budget"].(map[string]any)
				cache["bounded"] = false
				cache["unbounded_cache_rejected"] = false
			},
			want: "cache",
		},
		{
			name: "missing density evidence",
			mutate: func(engine map[string]any) {
				engine["density"] = map[string]any{"scale": float64(1)}
			},
			want: "density",
		},
		{
			name: "missing invalidation",
			mutate: func(engine map[string]any) {
				engine["invalidations"] = []any{}
			},
			want: "invalidation",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := mutateBlockLayoutReportJSON(t, func(report map[string]any) {
				engine := validSurfaceLayoutEngineEvidence()
				tc.mutate(engine)
				report["layout_engine"] = engine
			})
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected layout_engine overclaim to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

func TestValidateReportAcceptsBlockEventFocusEvidence(t *testing.T) {
	raw := validHeadlessBlockEventSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportRejectsIncompleteBlockEventFocusEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "missing nested hit path",
			mutate: func(report *Report) {
				report.BlockEventRoutes[0].HitTestPath = []int{1, 4}
			},
			want: "hit_test_path",
		},
		{
			name: "disabled click delivered",
			mutate: func(report *Report) {
				report.BlockEventRoutes[1].Delivered = true
				report.BlockEventRoutes[1].Rejected = false
				report.BlockEventRoutes[1].RejectReason = ""
			},
			want: "disabled",
		},
		{
			name: "unfocused text accepted",
			mutate: func(report *Report) {
				report.BlockEventRoutes[2].Delivered = true
				report.BlockEventRoutes[2].Rejected = false
				report.BlockEventRoutes[2].FocusedID = 5
				report.BlockEventRoutes[2].RejectReason = ""
			},
			want: "unfocused",
		},
		{
			name: "missing tab wrap",
			mutate: func(report *Report) {
				report.BlockFocusTransitions[1].Wrapped = false
			},
			want: "wrap",
		},
		{
			name: "unsupported drag drop",
			mutate: func(report *Report) {
				report.BlockEventUnsupportedDragDrop = true
			},
			want: "drag",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockEventSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected block event/focus %s evidence to fail", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

func TestValidateReportAcceptsBlockStateSelectorEvidence(t *testing.T) {
	raw := validHeadlessBlockStateSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportRejectsIncompleteBlockStateSelectorEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "wrong resolver order",
			mutate: func(report *Report) {
				report.BlockStateResolverOrder = []string{"base", "hover", "variant", "pressed", "focused", "selected", "disabled", "error", "loading", "motion"}
			},
			want: "resolver order",
		},
		{
			name: "missing hover selector",
			mutate: func(report *Report) {
				report.BlockStateSelectors = report.BlockStateSelectors[1:]
			},
			want: "hover",
		},
		{
			name: "pressed scale not applied",
			mutate: func(report *Report) {
				for i := range report.BlockStateResolutions {
					if report.BlockStateResolutions[i].Selector == "pressed" && report.BlockStateResolutions[i].Property == "layout.scale" {
						report.BlockStateResolutions[i].Applied = false
						report.BlockStateResolutions[i].After = report.BlockStateResolutions[i].Before
					}
				}
			},
			want: "pressed",
		},
		{
			name: "disabled transition missing",
			mutate: func(report *Report) {
				filtered := report.StateTransitions[:0]
				for _, transition := range report.StateTransitions {
					if transition.Component != "StateBlock" || transition.Field != "disabled" {
						filtered = append(filtered, transition)
					}
				}
				report.StateTransitions = filtered
			},
			want: "disabled",
		},
		{
			name: "unsupported css pseudo claim",
			mutate: func(report *Report) {
				report.BlockStateUnsupportedCSSPseudos = true
			},
			want: "css",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockStateSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected block state %s evidence to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

func TestValidateReportAcceptsBlockMotionEvidence(t *testing.T) {
	raw := validHeadlessBlockMotionSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportRequiresProductionAnimationSchedulerEvidence(t *testing.T) {
	raw := validHeadlessBlockMotionSurfaceReportJSON(t, nil)
	var report map[string]any
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode block motion report: %v", err)
	}
	delete(report, "animation_scheduler")
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block motion report without animation scheduler: %v", err)
	}

	err = ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected block motion report without animation_scheduler to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "animation_scheduler") {
		t.Fatalf("error = %v, want animation_scheduler diagnostic", err)
	}
}

func TestValidateReportRejectsIncompleteBlockMotionEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "missing motion frames",
			mutate: func(report *Report) {
				report.MotionFrames = nil
			},
			want: "motion_frames",
		},
		{
			name: "reduced motion keeps scheduling",
			mutate: func(report *Report) {
				for i := range report.MotionFrames {
					if report.MotionFrames[i].ReducedMotion {
						report.MotionFrames[i].Scheduled = true
						report.MotionFrames[i].Settled = false
					}
				}
			},
			want: "reduced",
		},
		{
			name: "completion keeps scheduling",
			mutate: func(report *Report) {
				report.MotionFrames[len(report.MotionFrames)-2].Scheduled = true
				report.MotionFrames[len(report.MotionFrames)-2].Settled = false
			},
			want: "settled",
		},
		{
			name: "opacity not interpolated",
			mutate: func(report *Report) {
				for i := range report.MotionFrames {
					report.MotionFrames[i].Opacity = 80
				}
			},
			want: "opacity",
		},
		{
			name: "unsupported css animations",
			mutate: func(report *Report) {
				report.MotionUnsupportedCSSAnimations = true
			},
			want: "css",
		},
		{
			name: "scheduler missing reduced motion guard",
			mutate: func(report *Report) {
				report.AnimationScheduler.NegativeGuards.MissingReducedMotion = false
			},
			want: "reduced",
		},
		{
			name: "scheduler frame timing mismatch",
			mutate: func(report *Report) {
				report.AnimationScheduler.MaxFrameDeltaMS = 16
			},
			want: "max_frame_delta",
		},
		{
			name: "scheduler hidden loop allowed",
			mutate: func(report *Report) {
				report.AnimationScheduler.HiddenLoopRejected = false
			},
			want: "hidden",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockMotionSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected block motion %s evidence to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

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

func TestValidateReportAcceptsHeadlessBlockSystemGoldenChecksumEvidence(t *testing.T) {
	raw := validHeadlessBlockSystemSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportAcceptsMorphCapsuleEvidence(t *testing.T) {
	raw := validHeadlessMorphSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed with Morph evidence: %v\n%s", err, raw)
	}
}

func TestValidateReportRejectsIncompleteMorphCapsuleEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "missing token graph",
			mutate: func(morph map[string]any) {
				delete(morph, "token_graph")
			},
			want: "token_graph",
		},
		{
			name: "fake core primitive recipe",
			mutate: func(morph map[string]any) {
				recipes := morph["recipes"].([]any)
				recipe := recipes[0].(map[string]any)
				recipe["output"] = "Button"
				recipe["core_primitive_promotion"] = true
			},
			want: "Button",
		},
		{
			name: "dirty production signoff",
			mutate: func(morph map[string]any) {
				morph["production_claim"] = true
				morph["git_dirty"] = true
			},
			want: "dirty checkout",
		},
		{
			name: "missing recipe expansion",
			mutate: func(morph map[string]any) {
				morph["recipe_expansions"] = []any{}
			},
			want: "recipe_expansions",
		},
		{
			name: "missing production recipe family",
			mutate: func(morph map[string]any) {
				recipes := morph["recipes"].([]any)
				morph["recipes"] = recipes[:4]
				authoring := morph["authoring"].(map[string]any)
				authoring["recipe_count"] = 4
				authoring["polished_recipe_count"] = 4
			},
			want: "recipe_count",
		},
		{
			name: "missing recipe state declaration",
			mutate: func(morph map[string]any) {
				recipes := morph["recipes"].([]any)
				recipe := recipes[0].(map[string]any)
				delete(recipe, "state")
			},
			want: "state",
		},
		{
			name: "missing recipe accessibility projection",
			mutate: func(morph map[string]any) {
				recipes := morph["recipes"].([]any)
				recipe := recipes[0].(map[string]any)
				delete(recipe, "a11y")
			},
			want: "a11y",
		},
		{
			name: "hidden app state in recipe",
			mutate: func(morph map[string]any) {
				recipes := morph["recipes"].([]any)
				recipe := recipes[0].(map[string]any)
				recipe["hidden_app_state"] = true
			},
			want: "hidden app state",
		},
		{
			name: "platform widget recipe output",
			mutate: func(morph map[string]any) {
				recipes := morph["recipes"].([]any)
				recipe := recipes[0].(map[string]any)
				recipe["platform_widgets"] = true
			},
			want: "platform widgets",
		},
		{
			name: "unreported recipe expansion",
			mutate: func(morph map[string]any) {
				expansions := morph["recipe_expansions"].([]any)
				expansion := expansions[0].(map[string]any)
				expansion["reported"] = false
			},
			want: "reported",
		},
		{
			name: "missing stable style graph diagnostics",
			mutate: func(morph map[string]any) {
				delete(morph, "style_graph")
			},
			want: "style_graph",
		},
		{
			name: "missing css replacement level",
			mutate: func(morph map[string]any) {
				if styleGraph, ok := morph["style_graph"].(map[string]any); ok {
					delete(styleGraph, "css_replacement_level")
				}
			},
			want: "css_replacement_level",
		},
		{
			name: "missing override order",
			mutate: func(morph map[string]any) {
				if styleGraph, ok := morph["style_graph"].(map[string]any); ok {
					delete(styleGraph, "override_order")
				}
			},
			want: "override_order",
		},
		{
			name: "ambiguous override order",
			mutate: func(morph map[string]any) {
				if styleGraph, ok := morph["style_graph"].(map[string]any); ok {
					styleGraph["override_order"] = []any{"tokens", "capsule", "materials", "affordances", "state_lenses", "motion", "recipes", "accessibility_safety"}
				}
			},
			want: "fixed and deterministic",
		},
		{
			name: "runtime framework imports rejected",
			mutate: func(morph map[string]any) {
				capsule := morph["capsule"].(map[string]any)
				capsule["imports"] = []any{"lib.core.block", "lib.core.morph", "react", "electron", "web.css"}
			},
			want: "forbidden import",
		},
		{
			name: "css cascade diagnostic required",
			mutate: func(morph map[string]any) {
				if styleGraph, ok := morph["style_graph"].(map[string]any); ok {
					styleGraph["css_cascade_imports_rejected"] = false
				}
			},
			want: "css cascade",
		},
		{
			name: "global style leak rejected",
			mutate: func(morph map[string]any) {
				if styleGraph, ok := morph["style_graph"].(map[string]any); ok {
					styleGraph["global_style_leak_rejected"] = false
				}
			},
			want: "global style leak",
		},
		{
			name: "specificity ambiguity rejected",
			mutate: func(morph map[string]any) {
				if styleGraph, ok := morph["style_graph"].(map[string]any); ok {
					styleGraph["specificity_ambiguity_rejected"] = false
				}
			},
			want: "specificity",
		},
		{
			name: "raw css runtime import rejected",
			mutate: func(morph map[string]any) {
				if styleGraph, ok := morph["style_graph"].(map[string]any); ok {
					styleGraph["raw_css_runtime_import_rejected"] = false
				}
			},
			want: "raw CSS",
		},
		{
			name: "missing recipe authoring evidence",
			mutate: func(morph map[string]any) {
				delete(morph, "authoring")
			},
			want: "authoring",
		},
		{
			name: "raw block field authoring rejected",
			mutate: func(morph map[string]any) {
				if authoring, ok := morph["authoring"].(map[string]any); ok {
					authoring["raw_80_field_blocks_rejected"] = false
				}
			},
			want: "80-field",
		},
		{
			name: "direct block prop editing rejected",
			mutate: func(morph map[string]any) {
				if authoring, ok := morph["authoring"].(map[string]any); ok {
					authoring["direct_block_prop_editing"] = true
				}
			},
			want: "direct Block prop",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessMorphSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected incomplete Morph evidence to fail")
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

func TestValidateReportAcceptsLinuxX64RealWindowBlockSystemEvidence(t *testing.T) {
	raw := validLinuxX64RealWindowBlockSystemSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportAcceptsWASM32WebBrowserCanvasBlockSystemEvidence(t *testing.T) {
	raw := validWASM32WebBrowserCanvasBlockSystemSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportRejectsBlockSystemWithoutSoftwareRendererEvidence(t *testing.T) {
	var report map[string]any
	if err := json.Unmarshal(validHeadlessBlockSystemSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode Block system report: %v", err)
	}
	delete(report, "software_renderer")
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal Block system report: %v", err)
	}
	err = ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected Block system report without software_renderer evidence to fail")
	}
	if !strings.Contains(err.Error(), "software_renderer") {
		t.Fatalf("error = %v, want software_renderer diagnostic", err)
	}
}

func TestValidateReportRejectsIncompleteSoftwareRendererEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "missing alpha mode",
			mutate: func(report *Report) {
				report.SoftwareRenderer.AlphaBlendMode = ""
			},
			want: "alpha",
		},
		{
			name: "missing clip checksum",
			mutate: func(report *Report) {
				report.SoftwareRenderer.Frames[0].ClipChecksum = ""
			},
			want: "clip",
		},
		{
			name: "missing dpi",
			mutate: func(report *Report) {
				report.SoftwareRenderer.Frames[0].DPI = 0
			},
			want: "dpi",
		},
		{
			name: "use after present",
			mutate: func(report *Report) {
				report.SoftwareRenderer.Frames[0].UseAfterPresentRejected = false
			},
			want: "use_after_present",
		},
		{
			name: "frame alias",
			mutate: func(report *Report) {
				report.SoftwareRenderer.NegativeGuards.FrameAliasRejected = false
			},
			want: "frame_alias",
		},
		{
			name: "runtime checksum mismatch",
			mutate: func(report *Report) {
				report.SoftwareRenderer.Frames[0].PixelChecksum = "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
			},
			want: "pixel_checksum",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockSystemSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected incomplete software renderer %s evidence to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

func TestValidateReportRejectsLinuxX64BlockSystemHeadlessPromotion(t *testing.T) {
	raw := validLinuxX64RealWindowBlockSystemSurfaceReportJSON(t, func(report *Report) {
		report.Target = "headless"
		report.Runtime = "surface-headless"
		report.HostEvidence = HostEvidenceReport{Level: "deterministic-headless", Backend: "software-rgba", Framebuffer: true}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected linux-x64 Block system report promoted from headless evidence to fail")
	}
	for _, want := range []string{"linux-x64", "real-window"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateReportRejectsLinuxX64BlockSystemMissingRealWindowPresentation(t *testing.T) {
	raw := validLinuxX64RealWindowBlockSystemSurfaceReportJSON(t, func(report *Report) {
		report.Frames = nil
		report.BlockSystem.Frames[0].Order = 2
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected linux-x64 Block system report without real-window frame presentation to fail")
	}
	for _, want := range []string{"real-window", "frame"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateReportRejectsWASM32WebBlockSystemFakeBrowserClaims(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "node-only browser promotion",
			mutate: func(report *Report) {
				report.HostEvidence = HostEvidenceReport{Level: "wasm32-web-compiler-owned-loader", Backend: "node-surface-host", Framebuffer: true}
			},
			want: "browser-canvas",
		},
		{
			name: "missing browser canvas RGBA readback",
			mutate: func(report *Report) {
				report.Frames = report.Frames[:1]
				report.BlockSystem.Frames = report.BlockSystem.Frames[:1]
				report.BlockSystem.FrameCount = 1
				filtered := report.Cases[:0]
				for _, tc := range report.Cases {
					if tc.Name == "wasm32-web browser canvas RGBA readback" {
						continue
					}
					filtered = append(filtered, tc)
				}
				report.Cases = filtered
			},
			want: "RGBA readback",
		},
		{
			name: "user JS artifact",
			mutate: func(report *Report) {
				report.Artifacts = append(report.Artifacts, ArtifactReport{
					Kind:   "user-js",
					Path:   "/tmp/surface-artifacts/surface-block-system.user.js",
					SHA256: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
					Size:   128,
				})
				report.ArtifactScan.FilesChecked++
			},
			want: "user JS",
		},
		{
			name: "DOM UI artifact",
			mutate: func(report *Report) {
				report.Artifacts = append(report.Artifacts, ArtifactReport{
					Kind:   "dom-ui",
					Path:   "/tmp/surface-artifacts/surface-block-system.dom.html",
					SHA256: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
					Size:   256,
				})
				report.ArtifactScan.FilesChecked++
			},
			want: "DOM UI",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validWASM32WebBrowserCanvasBlockSystemSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected wasm32-web browser-canvas Block system fake claim to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

func TestValidateReportRejectsIncompleteHeadlessBlockSystemGoldenChecksumEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "missing frame checksum",
			mutate: func(report *Report) {
				report.BlockSystem.Frames[0].Checksum = ""
			},
			want: "checksum",
		},
		{
			name: "nondeterministic repeat checksum",
			mutate: func(report *Report) {
				report.BlockSystem.Frames[1].RepeatChecksum = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
			},
			want: "nondeterministic",
		},
		{
			name: "missing paint evidence",
			mutate: func(report *Report) {
				report.PaintLayers = nil
				report.PaintCommands = nil
				report.BlockSystem.Frames[0].PaintEvidence = false
			},
			want: "paint",
		},
		{
			name: "missing layout evidence",
			mutate: func(report *Report) {
				report.LayoutPasses = nil
				report.LayoutConstraints = nil
				report.BlockSystem.Frames[0].LayoutEvidence = false
			},
			want: "layout",
		},
		{
			name: "missing accessibility evidence",
			mutate: func(report *Report) {
				report.BlockAccessibilityTree = nil
				report.BlockSystem.Frames[0].AccessibilityEvidence = false
			},
			want: "accessibility",
		},
		{
			name: "golden mismatch",
			mutate: func(report *Report) {
				report.BlockSystem.Frames[0].GoldenChecksum = "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
			},
			want: "golden",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockSystemSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected headless Block system %s evidence to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

func TestValidateReportRejectsIncompleteBlockSystemReadinessEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "missing app model",
			mutate: func(report *Report) {
				report.AppModel = nil
			},
			want: "app_model",
		},
		{
			name: "missing app event trace",
			mutate: func(report *Report) {
				report.AppModel.EventTraces = nil
			},
			want: "event trace",
		},
		{
			name: "disabled dispatch guard missing",
			mutate: func(report *Report) {
				report.AppModel.DisabledDispatchRejected = false
			},
			want: "disabled dispatch",
		},
		{
			name: "unfocused text guard missing",
			mutate: func(report *Report) {
				report.AppModel.UnfocusedTextRejected = false
			},
			want: "unfocused text",
		},
		{
			name: "react runtime claim guard missing",
			mutate: func(report *Report) {
				report.AppModel.ReactRuntimeClaimRejected = false
			},
			want: "React runtime claim",
		},
		{
			name: "react hook command rejected",
			mutate: func(report *Report) {
				report.AppModel.Commands[0].ReactHook = true
			},
			want: "React hooks",
		},
		{
			name: "missing text measurement",
			mutate: func(report *Report) {
				report.TextMeasurements = nil
				report.FontFallbacks = nil
				report.GlyphCaches = nil
				report.TextRenderCommands = nil
				report.TextQualityLevel = ""
				report.TextCacheBudgetBytes = 0
			},
			want: "text",
		},
		{
			name: "missing state selector",
			mutate: func(report *Report) {
				report.BlockStateSelectors = nil
				report.BlockStateResolutions = nil
				report.BlockStateResolverOrder = nil
				report.BlockStateQualityLevel = ""
			},
			want: "state",
		},
		{
			name: "missing motion frames",
			mutate: func(report *Report) {
				report.MotionFrames = nil
				report.MotionQualityLevel = ""
				report.MotionClock = ""
				report.MotionFrameBudget = 0
			},
			want: "motion",
		},
		{
			name: "missing asset cache",
			mutate: func(report *Report) {
				report.BlockAssetManifest = nil
				report.BlockAssetQualityLevel = ""
				report.BlockAssetCache = BlockAssetCacheReport{}
				report.BlockAssetDiagnostics = nil
				report.BlockAssetRenderCommands = nil
			},
			want: "asset",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockSystemSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected Block system %s to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

func TestValidateReportRejectsBlockSystemMissingKeyboardUXEvidence(t *testing.T) {
	raw := validHeadlessBlockSystemSurfaceReportJSON(t, func(report *Report) {
		report.KeyboardUX = nil
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected Block System report without keyboard_ux evidence to fail")
	}
	if !strings.Contains(err.Error(), "keyboard_ux") {
		t.Fatalf("error = %v, want keyboard_ux rejection", err)
	}
}

func TestValidateReportRejectsBlockSystemMissingAppShellEvidence(t *testing.T) {
	raw := validHeadlessBlockSystemSurfaceReportJSON(t, func(report *Report) {
		report.AppShell = nil
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected Block System report without app_shell evidence to fail")
	}
	if !strings.Contains(err.Error(), "app_shell") {
		t.Fatalf("error = %v, want app_shell rejection", err)
	}
}

func TestValidateReportRejectsIncompleteAppShellEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "missing menu host trace",
			mutate: func(report *Report) {
				report.AppShell.Menus[0].HostReportID = ""
			},
			want: "menu",
		},
		{
			name: "notification without host report",
			mutate: func(report *Report) {
				report.AppShell.Notifications[0].HostReportID = "missing.notification.host"
			},
			want: "notification",
		},
		{
			name: "unsupported feature silent no-op",
			mutate: func(report *Report) {
				report.AppShell.Diagnostics[0].SilentNoop = true
				report.AppShell.NegativeGuards.UnsupportedFeatureSilentNoopRejected = false
			},
			want: "silent no-op",
		},
		{
			name: "platform widget shell supported",
			mutate: func(report *Report) {
				report.AppShell.Capabilities = append(report.AppShell.Capabilities, AppShellCapabilityReport{
					Kind:              "platform_widget_shell",
					Supported:         true,
					HostTraceRequired: true,
				})
			},
			want: "platform_widget",
		},
		{
			name: "permission denial without diagnostic",
			mutate: func(report *Report) {
				report.AppShell.Permissions[1].DiagnosticID = ""
			},
			want: "permission",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockSystemSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected incomplete app_shell %s to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

func TestValidateReportRejectsIncompleteKeyboardUXEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "focusable accessible name missing",
			mutate: func(report *Report) {
				report.KeyboardUX.FocusOrder[0].AccessibleName = ""
				report.KeyboardUX.FocusOrder[0].LabelledBy = ""
			},
			want: "accessible name",
		},
		{
			name: "overlay focus leak not rejected",
			mutate: func(report *Report) {
				report.KeyboardUX.FocusTraps[0].LeakRejected = false
				report.KeyboardUX.OverlayFocusLeakRejected = false
			},
			want: "focus leak",
		},
		{
			name: "shortcut conflict not diagnosed",
			mutate: func(report *Report) {
				report.KeyboardUX.ShortcutConflicts[0].Diagnosed = false
				report.KeyboardUX.ShortcutConflictRejected = false
			},
			want: "shortcut conflict",
		},
		{
			name: "missing undo redo stack",
			mutate: func(report *Report) {
				report.KeyboardUX.UndoRedoStacks = nil
			},
			want: "undo/redo",
		},
		{
			name: "command palette keyboard script missing",
			mutate: func(report *Report) {
				report.KeyboardUX.KeyboardScripts[0].Pass = false
			},
			want: "command_palette",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockSystemSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected incomplete keyboard_ux %s to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

func TestValidateReportRejectsIncompleteBlockSystemMemoryBudget(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "missing memory budget",
			mutate: func(report *Report) {
				report.BlockSystem.MemoryBudget = nil
			},
			want: "block_system memory_budget is required",
		},
		{
			name: "unbounded caches",
			mutate: func(report *Report) {
				report.BlockSystem.MemoryBudget = blockMemoryBudgetForTest(report)
				report.BlockSystem.MemoryBudget.BoundedCaches = false
			},
			want: "bounded_caches",
		},
		{
			name: "mismatched framebuffer total",
			mutate: func(report *Report) {
				report.BlockSystem.MemoryBudget = blockMemoryBudgetForTest(report)
				report.BlockSystem.MemoryBudget.TotalFramebufferBytes++
			},
			want: "total_framebuffer_bytes",
		},
		{
			name: "broad electron claim",
			mutate: func(report *Report) {
				report.BlockSystem.MemoryBudget = blockMemoryBudgetForTest(report)
				report.BlockSystem.MemoryBudget.PerformanceClaim = "faster than " + "Electron"
			},
			want: "Electron",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockSystemSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected incomplete Block memory budget report to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

func TestValidateReportAcceptsBlockSystemMemoryBudgetEvidence(t *testing.T) {
	raw := validHeadlessBlockSystemSurfaceReportJSON(t, func(report *Report) {
		report.BlockSystem.MemoryBudget = blockMemoryBudgetForTest(report)
	})
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed with Block memory budget evidence: %v\n%s", err, raw)
	}
}

func TestValidateReportRejectsFakeBlockCorePrimitiveClaims(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "Button component type",
			mutate: func(report *Report) {
				report.Components[3].Type = "examples.surface_block_system.Button"
			},
			want: "Button",
		},
		{
			name: "Card block graph node",
			mutate: func(report *Report) {
				report.BlockGraph.Nodes[1].Name = "Card"
			},
			want: "Card",
		},
		{
			name: "TextField accessibility node",
			mutate: func(report *Report) {
				report.BlockAccessibilityTree.Nodes[0].Name = "TextField"
			},
			want: "TextField",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockSystemSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected fake Block core primitive claim %s to fail", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
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

func TestValidateReportRequiresProductionAssetPipelineEvidence(t *testing.T) {
	raw := validHeadlessBlockAssetSurfaceReportJSON(t, nil)
	var report map[string]any
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode block asset report: %v", err)
	}
	delete(report, "asset_pipeline")
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block asset report: %v", err)
	}
	err = ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected block asset report without asset_pipeline to fail")
	}
	if !strings.Contains(err.Error(), "asset_pipeline") {
		t.Fatalf("error = %v, want asset_pipeline diagnostic", err)
	}
}

func TestValidateReportRejectsIncompleteProductionAssetPipelineEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Report)
		want   string
	}{
		{
			name: "missing vector decoder",
			mutate: func(report *Report) {
				report.AssetPipeline.VectorDecoder = ""
			},
			want: "vector_decoder",
		},
		{
			name: "unsafe SVG guard disabled",
			mutate: func(report *Report) {
				report.AssetPipeline.NegativeGuards.UnsafeSVGRejected = false
			},
			want: "unsafe SVG",
		},
		{
			name: "unbounded cache mismatch",
			mutate: func(report *Report) {
				report.AssetPipeline.CacheBounded = false
				report.AssetPipeline.CacheBudgetBytes = 0
			},
			want: "cache",
		},
		{
			name: "remote font in manifest",
			mutate: func(report *Report) {
				report.BlockAssetManifest.Assets[0].Path = "https://assets.example.test/tetra-ui.woff2"
				report.BlockAssetManifest.Assets[0].Embedded = false
				report.BlockAssetManifest.Assets[0].Local = false
				report.BlockAssetManifest.RemoteCount = 1
			},
			want: "remote font",
		},
		{
			name: "decoder without hash validation",
			mutate: func(report *Report) {
				report.AssetPipeline.NegativeGuards.DecoderWithoutHashRejected = false
			},
			want: "hash validation",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessBlockAssetSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected incomplete asset pipeline %s evidence to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
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
  "producer": "scripts/release/surface/release-gate.sh",
  "git_head": "0123456789abcdef0123456789abcdef01234567",
  "version": "tetra_language",
  "git_dirty": false,
  "host_os": "linux",
  "host_arch": "amd64",
  "generated_at_utc": "2026-06-08T16:00:00Z",
  "command_line": "bash scripts/release/surface/release-gate.sh --report-dir reports/surface-release-v1",
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
  "block_system": "block-system",
  "block_system_gate": "tetra.surface.block-system.gate.v1",
  "morph": "morph-capsule",
  "morph_gate": "tetra.surface.morph.gate.v1",
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
  "text_pipeline": {
    "schema": "tetra.surface.text-pipeline.v1",
    "level": "scoped-latin-utf8-text-pipeline-v1",
    "engine": "deterministic-tetra-text-shaper",
    "platform_widget_text_controls": false,
    "font_manifest": [
      {"id":"tetra-ui-regular","family":"Tetra UI","style":"normal","weight":400,"source":"embedded:tetra-ui-regular","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","bytes":32768},
      {"id":"noto-sans-fallback","family":"Noto Sans","style":"normal","weight":400,"source":"system:fontconfig/noto-sans","sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","bytes":65536}
    ],
    "font_fallbacks": [
      {"id":"release-fallback","requested_family":"Tetra UI","resolved_family":"Noto Sans","chain":["Tetra UI","Noto Sans","monospace"],"missing_glyphs":0,"coverage":"latin-plus-basic-utf8-smoke"}
    ],
    "glyph_runs": [
      {"id":"latin-run","font_family":"Tetra UI","script":"Latin","direction":"ltr","shaping":"tier1-latin-simple","text_len":5,"byte_start":0,"byte_end":5,"scalar_start":0,"scalar_end":5,"glyph_count":5,"glyph_ids":[36,69,70,32,71],"advances":[8,8,8,4,8],"clusters":[0,1,2,3,4],"baseline":14,"checksum":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
      {"id":"fallback-run","font_family":"Noto Sans","script":"Common","direction":"ltr","shaping":"tier1-fallback-simple","text_len":1,"byte_start":5,"byte_end":7,"scalar_start":5,"scalar_end":6,"glyph_count":1,"glyph_ids":[9731],"advances":[9],"clusters":[5],"baseline":14,"checksum":"sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"}
    ],
    "glyph_caches": [
      {"id":"release-glyph-cache","strategy":"bounded-lru","budget_bytes":65536,"used_bytes":8192,"entry_count":24,"eviction":"lru","bounded":true}
    ],
    "cache_budget_bytes": 65536,
    "glyph_cache_budget_bytes": 65536,
    "glyph_cache_used_bytes": 8192,
    "bounded_caches": true,
    "cache_eviction": "lru",
    "unicode_boundaries": {
      "utf8_storage": true,
      "scalar_boundaries": true,
      "cluster_boundaries": true,
      "latin_tier": true,
      "combining_marks": false,
      "bidi": false,
      "unsupported_scripts": ["Arabic","Devanagari","Thai"],
      "boundary_cases": ["ASCII insertion","UTF-8 scalar insertion","cluster caret clamp"]
    },
    "shaping_scope": {
      "tier": "tier1-latin-utf8",
      "supported_scripts": ["Latin","Common"],
      "unsupported_scripts": ["Arabic","Devanagari","Thai"],
      "engine_decision": "deterministic embedded shaper until HarfBuzz-class evidence exists",
      "full_unicode_editor_semantics": false,
      "bidi": false,
      "combining_marks": false,
      "system_library_integration": "not required for Tier 1; future HarfBuzz-class gate",
      "platform_widgets": false
    },
    "measurements": [
      {"id":"release-label-measure","block_id":1,"text_len":18,"font_family":"Tetra UI","font_weight":400,"font_size":14,"line_height":18,"max_width":120,"measured":{"w":108,"h":18},"line_count":1,"wrap":"none","overflow":"clip","ellipsis":false,"ellipsized_text_len":18,"align":"start","quality":"deterministic-metrics-v1","checksum":"sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},
      {"id":"release-ellipsis-measure","block_id":2,"text_len":32,"font_family":"Tetra UI","font_weight":400,"font_size":14,"line_height":18,"max_width":96,"measured":{"w":96,"h":36},"line_count":2,"wrap":"word","overflow":"ellipsis","ellipsis":true,"ellipsized_text_len":20,"align":"start","quality":"deterministic-metrics-v1","checksum":"sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"}
    ],
    "measurement_consistency": {
      "same_input_same_metrics": true,
      "target_independent_baseline": true,
      "max_delta_px": 0,
      "cases": ["latin measurement repeat","fallback measurement repeat","ellipsis measurement repeat"]
    },
    "layout": {"wrap":true,"ellipsis":true,"alignment":["start","center","end"],"baseline":true,"line_height":true},
    "caret_rects": [{"x":32,"y":64,"w":2,"h":18}],
    "selection_rects": [{"x":32,"y":64,"w":18,"h":18}],
    "ime_composition_spans": [
      {"kind":"composition","byte_start":0,"byte_end":3,"scalar_start":0,"scalar_end":3,"rect":{"x":32,"y":64,"w":24,"h":18}}
    ],
    "nonclaims": [
      "full Unicode editor semantics",
      "bidi production shaping",
      "complex script shaping without HarfBuzz-class evidence",
      "platform widget text controls"
    ],
    "negative_guards": {
      "full_unicode_editor_without_tests_rejected": true,
      "missing_font_fallback_rejected": true,
      "unbounded_glyph_cache_rejected": true,
      "platform_widget_text_controls_rejected": true
    }
  },
  "text_editing": {
    "schema": "tetra.surface.text-editing.v1",
    "level": "production-editing-basics-v1",
    "target": "headless",
    "producer": "tools/cmd/surface-runtime-smoke",
    "editable_blocks": [
      {"id":"ReleaseTextBox","kind":"TextBox","storage":"owned-utf8-byte-buffer","forms_safe":true,"command_palette_search_safe":true,"max_bytes":1024,"utf8_validation":true}
    ],
    "edit_operations": [
      {"order":1,"action":"insert_text","target":"ReleaseTextBox","before_text_len":0,"after_text_len":3,"before_caret":0,"after_caret":3,"selection_before":{"anchor":0,"focus":0},"selection_after":{"anchor":3,"focus":3},"undo_unit_id":"insert-ada","checksum":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
      {"order":2,"action":"move_caret_left","target":"ReleaseTextBox","before_text_len":3,"after_text_len":3,"before_caret":3,"after_caret":2,"selection_before":{"anchor":3,"focus":3},"selection_after":{"anchor":2,"focus":2},"undo_unit_id":"navigation-left","checksum":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
      {"order":3,"action":"replace_selection","target":"ReleaseTextBox","before_text_len":5,"after_text_len":4,"before_caret":1,"after_caret":2,"selection_before":{"anchor":1,"focus":4},"selection_after":{"anchor":2,"focus":2},"undo_unit_id":"replace-selection","checksum":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
      {"order":4,"action":"composition_commit","target":"ReleaseTextBox","before_text_len":4,"after_text_len":5,"before_caret":4,"after_caret":5,"selection_before":{"anchor":4,"focus":4},"selection_after":{"anchor":5,"focus":5},"undo_unit_id":"ime-commit","checksum":"sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
      {"order":5,"action":"clipboard_write","target":"ReleaseTextBox","before_text_len":5,"after_text_len":5,"before_caret":5,"after_caret":5,"selection_before":{"anchor":0,"focus":5},"selection_after":{"anchor":0,"focus":5},"undo_unit_id":"clipboard-copy","checksum":"sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},
      {"order":6,"action":"clipboard_read","target":"ReleaseTextBox","before_text_len":0,"after_text_len":5,"before_caret":0,"after_caret":5,"selection_before":{"anchor":0,"focus":0},"selection_after":{"anchor":5,"focus":5},"undo_unit_id":"clipboard-paste","checksum":"sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"}
    ],
    "selection_model": {
      "caret_movement": ["left","right","home","end"],
      "selection_replacement": true,
      "scalar_boundary_clamp": true,
      "caret_rects": [{"x":32,"y":64,"w":2,"h":18}],
      "selection_rects": [{"x":32,"y":64,"w":24,"h":18}]
    },
    "ime_traces": [
      {"target":"headless","start":true,"update":true,"commit":true,"cancel":true,"event_count":4,"composition_span":{"kind":"composition","byte_start":0,"byte_end":3,"scalar_start":0,"scalar_end":3,"rect":{"x":32,"y":64,"w":24,"h":18}},"committed_text_owned_copy":true}
    ],
    "clipboard_transfers": [
      {"direction":"write","host_abi":"__tetra_surface_clipboard_write_text","bytes":5,"utf8_valid":true,"owned_copy":true,"borrowed_view":false,"checksum":"sha256:1111111111111111111111111111111111111111111111111111111111111111"},
      {"direction":"read","host_abi":"__tetra_surface_clipboard_read_text_into","bytes":5,"utf8_valid":true,"owned_copy":true,"borrowed_view":false,"checksum":"sha256:2222222222222222222222222222222222222222222222222222222222222222"}
    ],
    "undo_units": [
      {"id":"insert-ada","operation_orders":[1],"boundary":"text-input-operation","reversible":true,"coalesced":false},
      {"id":"navigation-left","operation_orders":[2],"boundary":"caret-navigation","reversible":true,"coalesced":false},
      {"id":"replace-selection","operation_orders":[3],"boundary":"selection-replacement","reversible":true,"coalesced":false},
      {"id":"ime-commit","operation_orders":[4],"boundary":"composition-commit","reversible":true,"coalesced":false},
      {"id":"clipboard-copy","operation_orders":[5],"boundary":"clipboard-copy","reversible":true,"coalesced":false},
      {"id":"clipboard-paste","operation_orders":[6],"boundary":"clipboard-paste","reversible":true,"coalesced":false}
    ],
    "validation_diagnostics": [
      {"name":"invalid UTF-8 rejected","ran":true,"pass":true},
      {"name":"borrowed text buffer rejected at host boundary","ran":true,"pass":true},
      {"name":"IME claim without target trace rejected","ran":true,"pass":true},
      {"name":"rich text claim rejected","ran":true,"pass":true}
    ],
    "host_boundary": {
      "copy_safe": true,
      "clipboard_owned_copy": true,
      "composition_owned_copy": true,
      "borrowed_text_buffer_crosses_host": false
    },
    "forms_safe": true,
    "command_palette_search_safe": true,
    "rich_text": false,
    "nonclaims": [
      "rich text",
      "full editor-grade text semantics",
      "native platform text controls"
    ],
    "negative_guards": {
      "ime_without_target_trace_rejected": true,
      "borrowed_text_buffer_rejected": true,
      "rich_text_claim_rejected": true,
      "unsafe_clipboard_alias_rejected": true,
      "invalid_utf8_rejected": true
    }
  },
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
    {"name":"release text editing target IME trace","kind":"positive","ran":true,"pass":true},
    {"name":"release text editing clipboard owned copies","kind":"positive","ran":true,"pass":true},
    {"name":"release text editing undo unit boundaries","kind":"positive","ran":true,"pass":true},
    {"name":"release text editing validation diagnostics","kind":"positive","ran":true,"pass":true},
    {"name":"release text editing rich text nonclaim","kind":"positive","ran":true,"pass":true},
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

func validHeadlessBlockGraphSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(validHeadlessComponentTreeSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode component tree report: %v", err)
	}
	report.BlockGraph = blockGraphReportForTest(report.Source)
	report.Cases = append(report.Cases,
		CaseReport{Name: "block graph duplicate id rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "duplicate Block ID"},
		CaseReport{Name: "block graph missing parent rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "missing parent"},
		CaseReport{Name: "block graph cycle rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "cycle"},
		CaseReport{Name: "block graph child order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block graph focus order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block graph hit-test path", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block graph accessibility order", Kind: "positive", Ran: true, Pass: true},
	)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block graph report: %v", err)
	}
	return raw
}

func validHeadlessBlockPaintSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(validHeadlessBlockGraphSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode block graph report: %v", err)
	}
	report.PaintQualityLevel = "deterministic-software-paint-v1"
	report.PaintCacheBudgetBytes = 65536
	report.PaintUnsupportedBlur = false
	report.PaintLayers = blockPaintLayersForTest()
	report.PaintCommands = blockPaintCommandsForTest()
	report.VisualFeatures = []string{"fill", "gradient", "border", "radius", "shadow", "outline"}
	report.RendererScene = rendererSceneForTest(report.Source, "software-rgba-headless")
	report.SoftwareRenderer = softwareRendererForTest(report.Source, report.Target, "software-rgba-headless", report.Frames)
	report.Cases = append(report.Cases,
		CaseReport{Name: "block paint fill border radius shadow outline", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block paint deterministic command order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block paint frame checksum changed", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block paint unsupported blur rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "unsupported blur"},
	)
	report.Cases = append(report.Cases, softwareRendererCasesForTest()...)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block paint report: %v", err)
	}
	return raw
}

func validHeadlessBlockTextSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(validHeadlessSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode headless report: %v", err)
	}
	report.Source = "examples/surface_block_text.tetra"
	report.Processes = []ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_text.tetra -o /tmp/surface-artifacts/surface-block-text", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-text", Ran: true, Pass: true, ExitCode: intPtrForTest(1), ExpectedExitCode: intPtrForTest(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-text", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
	}
	report.Artifacts = []ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-block-text", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 49172},
		{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 409},
	}
	report.ArtifactScan = ArtifactScanReport{Root: "/tmp/surface-artifacts", FilesChecked: 2, ForbiddenPaths: nil, Pass: true}
	report.Components = blockTextComponentsForTest()
	report.TextQualityLevel = "deterministic-fallback-text-v1"
	report.TextCacheBudgetBytes = 65536
	report.TextMeasurements = blockTextMeasurementsForTest()
	report.FontFallbacks = blockFontFallbacksForTest()
	report.GlyphCaches = blockGlyphCachesForTest()
	report.TextRenderCommands = blockTextRenderCommandsForTest()
	report.Events = blockTextEventsForTest()
	report.Frames = []FrameReport{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", Presented: true},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, Checksum: "2222222222222222222222222222222222222222222222222222222222222222", Presented: true},
	}
	report.StateTransitions = []StateTransitionReport{
		{Order: 1, Component: "BlockTextApp", Field: "focused_id", Before: "0", After: "3", Cause: "mouse_up"},
		{Order: 2, Component: "InputBlock", Field: "buffer", Before: "", After: "OKd0a2", Cause: "text_input"},
		{Order: 3, Component: "InputBlock", Field: "caret", Before: "0", After: "4", Cause: "text_input"},
	}
	report.Cases = append(report.Cases,
		CaseReport{Name: "block text deterministic measurement", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block text wrap ellipsis layout", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block text font fallback chain", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block text bounded glyph cache", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block text render command evidence", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block text editable lifetime", Kind: "positive", Ran: true, Pass: true},
	)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block text report: %v", err)
	}
	return raw
}

func validHeadlessBlockLayoutSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(validHeadlessSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode headless report: %v", err)
	}
	report.Source = "examples/surface_block_layout.tetra"
	report.Processes = []ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_layout.tetra -o /tmp/surface-artifacts/surface-block-layout", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-layout", Ran: true, Pass: true, ExitCode: intPtrForTest(1), ExpectedExitCode: intPtrForTest(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-layout", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
	}
	report.Artifacts = []ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-block-layout", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 49172},
		{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 409},
	}
	report.ArtifactScan = ArtifactScanReport{Root: "/tmp/surface-artifacts", FilesChecked: 2, ForbiddenPaths: nil, Pass: true}
	report.Components = blockLayoutComponentsForTest()
	report.LayoutQualityLevel = "deterministic-block-layout-v1"
	report.LayoutUnsupportedCSSFlexbox = false
	report.LayoutFeatures = []string{"stack", "row", "column", "absolute", "overlay", "grid", "dock", "scroll", "fit", "fill", "fixed", "min", "max", "spacing", "alignment", "z-order", "clipping", "resize"}
	report.LayoutConstraints = blockLayoutConstraintsForTest()
	report.LayoutPasses = blockLayoutPassesForTest()
	report.LayoutScrolls = blockLayoutScrollsForTest()
	report.LayoutEngine = blockLayoutEngineForTest("headless")
	report.Frames = []FrameReport{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", Presented: true},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, Checksum: "2222222222222222222222222222222222222222222222222222222222222222", Presented: true},
		{Order: 3, Width: 480, Height: 260, Stride: 1920, Checksum: "3333333333333333333333333333333333333333333333333333333333333333", Presented: true},
	}
	report.StateTransitions = []StateTransitionReport{
		{Order: 1, Component: "RowBlock", Field: "pressed", Before: "false", After: "true", Cause: "mouse_up"},
		{Order: 2, Component: "RowBlock", Field: "text_len_seen", Before: "0", After: "2", Cause: "text_input"},
		{Order: 3, Component: "BlockLayoutApp", Field: "width", Before: "320", After: "480", Cause: "resize"},
		{Order: 4, Component: "ScrollBlock", Field: "scroll_y", Before: "0", After: "32", Cause: "scroll"},
	}
	report.Events = []EventReport{
		{Order: 1, Kind: "mouse_up", TargetComponent: "RowBlock", DispatchPath: []string{"BlockLayoutApp", "ColumnBlock", "RowBlock"}, Handled: true, Pass: true, X: 32, Y: 32, Width: 320, Height: 200, BufferSlots: []int{5, 32, 32, 1, 0, 320, 200, 0, 0}, BeforeState: map[string]string{"RowBlock.pressed": "false"}, AfterState: map[string]string{"RowBlock.pressed": "true"}},
		{Order: 2, Kind: "text_input", TargetComponent: "RowBlock", DispatchPath: []string{"BlockLayoutApp", "ColumnBlock", "RowBlock"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 1, TextLen: 2, TextBytesHex: "4f4b", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 1, 2}, BeforeState: map[string]string{"RowBlock.text_len_seen": "0"}, AfterState: map[string]string{"RowBlock.text_len_seen": "2"}},
		{Order: 3, Kind: "resize", TargetComponent: "BlockLayoutApp", DispatchPath: []string{"BlockLayoutApp"}, Handled: true, Pass: true, Width: 480, Height: 260, TimestampMS: 2, BufferSlots: []int{6, 0, 0, 0, 0, 480, 260, 2, 0}, BeforeState: map[string]string{"BlockLayoutApp.width": "320"}, AfterState: map[string]string{"BlockLayoutApp.width": "480"}},
		{Order: 4, Kind: "scroll", TargetComponent: "ScrollBlock", DispatchPath: []string{"BlockLayoutApp", "ScrollBlock"}, Handled: true, Pass: true, X: 260, Y: 80, Width: 480, Height: 260, TimestampMS: 3, BufferSlots: []int{7, 260, 80, 0, 0, 480, 260, 3, 0}, BeforeState: map[string]string{"ScrollBlock.scroll_y": "0"}, AfterState: map[string]string{"ScrollBlock.scroll_y": "32"}},
	}
	report.Cases = append(report.Cases,
		CaseReport{Name: "block layout nested row column", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block layout fit fill fixed min max", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block layout grid dock overlay scroll", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block layout clipping z-order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block layout resize constraints", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block layout no css flexbox parity", Kind: "negative", Ran: true, Pass: true, ExpectedError: "CSS flexbox parity nonclaim"},
	)
	report.Cases = append(report.Cases, blockLayoutEngineCasesForTest()...)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block layout report: %v", err)
	}
	return raw
}

func mutateBlockLayoutReportJSON(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessBlockLayoutSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode block layout report: %v", err)
	}
	mutate(report)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal block layout report: %v", err)
	}
	return raw
}

func validSurfaceLayoutEngineEvidence() map[string]any {
	return map[string]any{
		"schema":                    "tetra.surface.layout-engine.v1",
		"level":                     "production-layout-engine-v1",
		"producer":                  "tools/cmd/surface-runtime-smoke",
		"target":                    "headless",
		"quality":                   "deterministic-responsive-layout-v1",
		"css_flexbox_grid_parity":   false,
		"platform_widget_layout":    false,
		"modes":                     []any{"row", "column", "stack", "grid", "dock", "absolute", "overlay", "scroll"},
		"responsive_profiles":       []any{"app shell", "settings forms", "dashboards", "editor shells"},
		"constraint_features":       []any{"min", "max", "fit", "fill", "fixed", "density", "overflow", "clip"},
		"min_max_constraints":       true,
		"fit_fill_fixed":            true,
		"density_independent":       true,
		"stable_under_resize":       true,
		"layout_cache_keyed_by_dpi": true,
		"density": map[string]any{
			"schema":             "tetra.surface.layout-density.v1",
			"scale":              float64(2),
			"dpi":                float64(192),
			"device_pixel_ratio": float64(2),
			"snap_to_pixel_grid": true,
			"target_independent": true,
			"cases":              []any{"headless scale 1", "linux-x64 scale 2", "wasm32-web devicePixelRatio 2"},
		},
		"overflow_rules": map[string]any{
			"explicit":                      true,
			"clip_required":                 true,
			"scroll_bounds_checked":         true,
			"accidental_hidden":             false,
			"hidden_requires_explicit_clip": true,
			"visible_overflow_preserved":    true,
			"clipped_block_ids":             []any{float64(1), float64(4), float64(5), float64(7), float64(8)},
		},
		"invalidations": []any{
			map[string]any{"cause": "resize", "dirty_root": "BlockLayoutApp", "affected_modes": []any{"column", "row", "grid", "dock", "overlay", "scroll"}, "before_checksum": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "after_checksum": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "cache_entries_reused": float64(2), "cache_entries_invalidated": float64(6), "full_tree_relayout": true},
			map[string]any{"cause": "scroll", "dirty_root": "ScrollBlock", "affected_modes": []any{"scroll"}, "before_checksum": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "after_checksum": "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", "cache_entries_reused": float64(6), "cache_entries_invalidated": float64(1), "full_tree_relayout": false},
		},
		"cache_budget": map[string]any{
			"schema":                   "tetra.surface.layout-cache.v1",
			"strategy":                 "bounded-lru",
			"budget_bytes":             float64(65536),
			"used_bytes":               float64(9216),
			"entry_count":              float64(9),
			"max_entries":              float64(64),
			"bounded":                  true,
			"eviction":                 "lru",
			"unbounded_cache_rejected": true,
		},
		"negative_guards": map[string]any{
			"css_flexbox_grid_parity_rejected":    true,
			"accidental_overflow_hidden_rejected": true,
			"unbounded_layout_cache_rejected":     true,
			"missing_dpi_density_rejected":        true,
			"invalid_invalidation_rejected":       true,
		},
		"nonclaims": []any{
			"CSS flexbox/grid parity",
			"browser CSS layout engine",
			"platform widget layout",
			"unbounded layout cache",
		},
	}
}

func validHeadlessBlockEventSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(validHeadlessSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode headless report: %v", err)
	}
	report.Source = "examples/surface_block_events.tetra"
	report.Processes = []ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_events.tetra -o /tmp/surface-artifacts/surface-block-events", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-events", Ran: true, Pass: true, ExitCode: intPtrForTest(1), ExpectedExitCode: intPtrForTest(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-events", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
	}
	report.Artifacts = []ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-block-events", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 49172},
		{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 409},
	}
	report.ArtifactScan = ArtifactScanReport{Root: "/tmp/surface-artifacts", FilesChecked: 2, ForbiddenPaths: nil, Pass: true}
	report.Components = blockEventComponentsForTest()
	report.BlockGraph = blockEventGraphReportForTest(report.Source)
	report.BlockEventQualityLevel = "deterministic-block-events-v1"
	report.BlockEventPolicy = "capture-bubble-direct-v1"
	report.BlockEventUnsupportedDragDrop = false
	report.BlockEventKinds = []string{"pointer_enter", "pointer_leave", "pointer_move", "pointer_down", "pointer_up", "click", "double_click", "key", "text", "focus", "blur", "scroll", "resize", "close", "frame"}
	report.BlockEventRoutes = blockEventRoutesForTest()
	report.BlockFocusTransitions = blockFocusTransitionsForTest()
	report.Events = blockEventRuntimeEventsForTest()
	report.Frames = []FrameReport{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", Presented: true},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, Checksum: "2222222222222222222222222222222222222222222222222222222222222222", Presented: true},
	}
	report.StateTransitions = []StateTransitionReport{
		{Order: 1, Component: "BlockEventApp", Field: "focused_id", Before: "0", After: "4", Cause: "click"},
		{Order: 2, Component: "InputBlock", Field: "buffer", Before: "", After: "OK", Cause: "text_input"},
		{Order: 3, Component: "BlockEventApp", Field: "focused_id", Before: "4", After: "6", Cause: "tab"},
		{Order: 4, Component: "BlockEventApp", Field: "focused_id", Before: "6", After: "4", Cause: "tab"},
	}
	report.Cases = append(report.Cases,
		CaseReport{Name: "block graph duplicate id rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "duplicate Block ID"},
		CaseReport{Name: "block graph missing parent rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "missing parent"},
		CaseReport{Name: "block graph cycle rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "cycle"},
		CaseReport{Name: "block graph child order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block graph focus order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block graph hit-test path", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block graph accessibility order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block event nested hit-test path", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block event capture bubble direct policy", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block event disabled click rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "disabled Block"},
		CaseReport{Name: "block event text input focused only", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block focus tab order graph-derived", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block event no complex drag claim", Kind: "negative", Ran: true, Pass: true, ExpectedError: "drag-and-drop nonclaim"},
	)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block event report: %v", err)
	}
	return raw
}

func validHeadlessBlockStateSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(validHeadlessSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode headless report: %v", err)
	}
	report.Source = "examples/surface_block_states.tetra"
	report.Processes = []ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_states.tetra -o /tmp/surface-artifacts/surface-block-states", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-states", Ran: true, Pass: true, ExitCode: intPtrForTest(1), ExpectedExitCode: intPtrForTest(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-states", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
	}
	report.Artifacts = []ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-block-states", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 49172},
		{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 409},
	}
	report.ArtifactScan = ArtifactScanReport{Root: "/tmp/surface-artifacts", FilesChecked: 2, ForbiddenPaths: nil, Pass: true}
	report.Components = blockStateComponentsForTest()
	report.BlockStateQualityLevel = "deterministic-block-state-resolver-v1"
	report.BlockStateResolverOrder = []string{"base", "variant", "hover", "pressed", "focused", "selected", "disabled", "error", "loading", "motion"}
	report.BlockStateUnsupportedCSSPseudos = false
	report.BlockStateSelectors = blockStateSelectorsForTest()
	report.BlockStateResolutions = blockStateResolutionsForTest()
	report.Events = blockStateEventsForTest()
	report.Frames = []FrameReport{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", Presented: true},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, Checksum: "2222222222222222222222222222222222222222222222222222222222222222", Presented: true},
	}
	report.StateTransitions = []StateTransitionReport{
		{Order: 1, Component: "StateBlock", Field: "selector_flags", Before: "0", After: "127", Cause: "pointer/key/state input"},
		{Order: 2, Component: "StateBlock", Field: "resolved_fill", Before: "#20262eff", After: "#2d9bf0ff", Cause: "hover"},
		{Order: 3, Component: "StateBlock", Field: "resolved_scale", Before: "100", After: "97", Cause: "pressed"},
		{Order: 4, Component: "StateBlock", Field: "disabled", Before: "false", After: "true", Cause: "disabled selector"},
		{Order: 5, Component: "StateBlock", Field: "error", Before: "false", After: "true", Cause: "error selector"},
		{Order: 6, Component: "StateBlock", Field: "loading", Before: "false", After: "true", Cause: "loading selector"},
	}
	report.Cases = append(report.Cases,
		CaseReport{Name: "block state selector resolver order", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block state hover fill override", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block state pressed scale override", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block state focus selected metadata", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block state disabled error loading overrides", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block state frame checksum changed", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block state no css pseudo parity", Kind: "negative", Ran: true, Pass: true, ExpectedError: "css pseudo nonclaim"},
	)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block state report: %v", err)
	}
	return raw
}

func validHeadlessBlockMotionSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(validHeadlessSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode headless report: %v", err)
	}
	report.Source = "examples/surface_block_motion.tetra"
	report.Processes = []ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_motion.tetra -o /tmp/surface-artifacts/surface-block-motion", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-motion", Ran: true, Pass: true, ExitCode: intPtrForTest(1), ExpectedExitCode: intPtrForTest(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-motion", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
	}
	report.Artifacts = []ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-block-motion", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 49172},
		{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 409},
	}
	report.ArtifactScan = ArtifactScanReport{Root: "/tmp/surface-artifacts", FilesChecked: 2, ForbiddenPaths: nil, Pass: true}
	report.Components = blockMotionComponentsForTest()
	report.MotionQualityLevel = "deterministic-block-motion-v1"
	report.MotionClock = "deterministic-test-clock-v1"
	report.MotionFrameBudget = 4
	report.MotionUnsupportedCSSAnimations = false
	report.MotionFrames = blockMotionFramesForTest()
	report.AnimationScheduler = animationSchedulerForTest(report.Source, report.MotionFrames, report.MotionFrameBudget, "headless", "surface-headless")
	report.Events = blockMotionEventsForTest()
	report.Frames = []FrameReport{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", Presented: true},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, Checksum: "2222222222222222222222222222222222222222222222222222222222222222", Presented: true},
		{Order: 3, Width: 320, Height: 200, Stride: 1280, Checksum: "3333333333333333333333333333333333333333333333333333333333333333", Presented: true},
	}
	report.StateTransitions = []StateTransitionReport{
		{Order: 1, Component: "MotionBlock", Field: "opacity", Before: "80", After: "200", Cause: "motion frame"},
		{Order: 2, Component: "MotionBlock", Field: "color", Before: "#203040ff", After: "#60aef4ff", Cause: "motion frame"},
		{Order: 3, Component: "MotionBlock", Field: "scale", Before: "100", After: "108", Cause: "motion frame"},
		{Order: 4, Component: "MotionBlock", Field: "translate_x", Before: "0", After: "12", Cause: "motion frame"},
		{Order: 5, Component: "MotionBlock", Field: "motion_complete", Before: "false", After: "true", Cause: "duration elapsed"},
		{Order: 6, Component: "MotionBlock", Field: "reduced_motion", Before: "false", After: "true", Cause: "accessibility setting"},
	}
	report.Cases = append(report.Cases,
		CaseReport{Name: "block motion deterministic test clock", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block motion opacity color transform frames", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block motion reduced motion instant settle", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block motion completion stops scheduling", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block motion frame checksum changed", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block motion no css animation parity", Kind: "negative", Ran: true, Pass: true, ExpectedError: "css animation nonclaim"},
		CaseReport{Name: "block motion frame scheduler timeline", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block motion invalidation lifecycle", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block motion frame timing evidence", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block motion hidden loop rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "hidden animation loop rejected"},
	)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block motion report: %v", err)
	}
	return raw
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
	report.AssetPipeline = assetPipelineForTest(report.Source, report.BlockAssetManifest, report.BlockAssetCache, report.BlockAssetDiagnostics, report.BlockAssetRenderCommands)
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
		CaseReport{Name: "block asset vector safe decode", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block asset missing fallback diagnostic", Kind: "negative", Ran: true, Pass: true, ExpectedError: "missing asset"},
		CaseReport{Name: "block asset network url rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "network assets disabled"},
		CaseReport{Name: "block asset unsafe SVG rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "unsafe svg"},
		CaseReport{Name: "block asset remote font rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "remote font"},
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

func validHeadlessBlockSystemSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(validHeadlessSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode headless report: %v", err)
	}
	report.Source = "examples/surface_block_system.tetra"
	report.Processes = []ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_system.tetra -o /tmp/surface-artifacts/surface-block-system", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-system", Ran: true, Pass: true, ExitCode: intPtrForTest(1), ExpectedExitCode: intPtrForTest(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-system", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
	}
	report.Artifacts = []ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-block-system", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 49172},
		{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 409},
	}
	report.ArtifactScan = ArtifactScanReport{Root: "/tmp/surface-artifacts", FilesChecked: 2, ForbiddenPaths: nil, Pass: true}
	report.Components = blockSystemComponentsForTest()
	report.Components = append(report.Components, retargetBlockSystemComponentsForTest(blockTextComponentsForTest())...)
	report.Components = append(report.Components, retargetBlockSystemComponentsForTest(blockStateComponentsForTest())...)
	report.Components = append(report.Components, retargetBlockSystemComponentsForTest(blockMotionComponentsForTest())...)
	report.Components = append(report.Components, retargetBlockSystemComponentsForTest(blockAssetComponentsForTest())...)
	report.BlockGraph = blockGraphReportForTest(report.Source)
	report.PaintQualityLevel = "deterministic-software-paint-v1"
	report.PaintCacheBudgetBytes = 65536
	report.PaintUnsupportedBlur = false
	report.PaintLayers = blockPaintLayersForTest()
	report.PaintCommands = blockPaintCommandsForTest()
	report.VisualFeatures = []string{"fill", "gradient", "border", "radius", "shadow", "outline"}
	report.TextQualityLevel = "deterministic-fallback-text-v1"
	report.TextCacheBudgetBytes = 65536
	report.TextMeasurements = blockTextMeasurementsForTest()
	report.FontFallbacks = blockFontFallbacksForTest()
	report.GlyphCaches = blockGlyphCachesForTest()
	report.TextRenderCommands = blockTextRenderCommandsForTest()
	report.LayoutQualityLevel = "deterministic-block-layout-v1"
	report.LayoutUnsupportedCSSFlexbox = false
	report.LayoutFeatures = []string{"stack", "row", "column", "absolute", "overlay", "grid", "dock", "scroll", "fit", "fill", "fixed", "min", "max", "spacing", "alignment", "z-order", "clipping", "resize"}
	report.LayoutConstraints = blockLayoutConstraintsForTest()
	report.LayoutPasses = blockLayoutPassesForTest()
	report.LayoutScrolls = blockLayoutScrollsForTest()
	report.LayoutEngine = blockLayoutEngineForTest("headless")
	report.BlockStateQualityLevel = "deterministic-block-state-resolver-v1"
	report.BlockStateUnsupportedCSSPseudos = false
	report.BlockStateResolverOrder = []string{"base", "variant", "hover", "pressed", "focused", "selected", "disabled", "error", "loading", "motion"}
	report.BlockStateSelectors = blockStateSelectorsForTest()
	report.BlockStateResolutions = blockStateResolutionsForTest()
	report.AppModel = appModelForTest("headless")
	report.KeyboardUX = keyboardUXForTest("headless")
	report.AppShell = appShellForTest("headless")
	report.MotionQualityLevel = "deterministic-block-motion-v1"
	report.MotionClock = "deterministic-test-clock-v1"
	report.MotionFrameBudget = 4
	report.MotionUnsupportedCSSAnimations = false
	report.MotionFrames = blockMotionFramesForTest()
	report.BlockAssetQualityLevel = "deterministic-local-block-assets-v1"
	report.BlockAssetNetworkFetchAllowed = false
	report.BlockAssetManifest = blockAssetManifestForTest(report.Source)
	report.BlockAssetCache = blockAssetCacheForTest()
	report.BlockAssetDiagnostics = blockAssetDiagnosticsForTest()
	report.BlockAssetRenderCommands = blockAssetRenderCommandsForTest()
	report.AssetPipeline = assetPipelineForTest(report.Source, report.BlockAssetManifest, report.BlockAssetCache, report.BlockAssetDiagnostics, report.BlockAssetRenderCommands)
	report.BlockAccessibilityTree = blockAccessibilityTreeForTest(report.Source)
	report.Events = blockSystemEventsForTest()
	report.Events = appendEventReportsWithNextOrder(report.Events,
		blockTextEventsForTest(),
		blockStateEventsForTest(),
		blockMotionEventsForTest(),
		blockAssetEventsForTest(),
	)
	report.Frames = []FrameReport{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", Presented: true},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, Checksum: "2222222222222222222222222222222222222222222222222222222222222222", Presented: true},
		{Order: 3, Width: 320, Height: 200, Stride: 1280, Checksum: "3333333333333333333333333333333333333333333333333333333333333333", Presented: true},
	}
	report.StateTransitions = []StateTransitionReport{
		{Order: 1, Component: "SubmitBlock", Field: "focused", Before: "false", After: "true", Cause: "tab"},
		{Order: 2, Component: "ResetBlock", Field: "focused", Before: "false", After: "true", Cause: "tab"},
		{Order: 3, Component: "BlockSystemApp", Field: "reading_order_checked", Before: "false", After: "true", Cause: "block_graph"},
		{Order: 4, Component: "BlockLayoutApp", Field: "width", Before: "320", After: "480", Cause: "resize"},
		{Order: 5, Component: "ScrollBlock", Field: "scroll_y", Before: "0", After: "32", Cause: "scroll"},
	}
	report.StateTransitions = appendStateTransitionReportsWithNextOrder(report.StateTransitions, blockSystemReadinessTransitionsForTest())
	report.BlockSystem = &BlockSystemReport{
		Schema:       "tetra.surface.block-system.v1",
		QualityLevel: "deterministic-headless-block-system-v1",
		Source:       report.Source,
		Renderer:     "software-rgba-headless",
		GoldenSet:    "surface-block-system-golden-v1",
		FrameCount:   3,
		GoldenHash:   "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		Frames: []BlockSystemFrameReport{
			{Order: 1, Label: "initial", Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", RepeatChecksum: "1111111111111111111111111111111111111111111111111111111111111111", GoldenChecksum: "1111111111111111111111111111111111111111111111111111111111111111", PaintEvidence: true, LayoutEvidence: true, AccessibilityEvidence: true},
			{Order: 2, Label: "focused", Width: 320, Height: 200, Stride: 1280, Checksum: "2222222222222222222222222222222222222222222222222222222222222222", RepeatChecksum: "2222222222222222222222222222222222222222222222222222222222222222", GoldenChecksum: "2222222222222222222222222222222222222222222222222222222222222222", PaintEvidence: true, LayoutEvidence: true, AccessibilityEvidence: true},
			{Order: 3, Label: "motion", Width: 320, Height: 200, Stride: 1280, Checksum: "3333333333333333333333333333333333333333333333333333333333333333", RepeatChecksum: "3333333333333333333333333333333333333333333333333333333333333333", GoldenChecksum: "3333333333333333333333333333333333333333333333333333333333333333", PaintEvidence: true, LayoutEvidence: true, AccessibilityEvidence: true},
		},
		NegativeGuards: BlockSystemNegativeGuardsReport{
			MissingFrameChecksumRejected:         true,
			NondeterministicChecksumRejected:     true,
			MissingPaintEvidenceRejected:         true,
			MissingLayoutEvidenceRejected:        true,
			MissingAccessibilityEvidenceRejected: true,
		},
	}
	report.RendererScene = rendererSceneForTest(report.Source, report.BlockSystem.Renderer)
	report.SoftwareRenderer = softwareRendererForTest(report.Source, report.Target, report.BlockSystem.Renderer, report.Frames)
	report.BlockSystem.MemoryBudget = blockMemoryBudgetForTest(&report)
	report.Cases = append(report.Cases, blockSystemCasesForTest()...)
	report.Cases = append(report.Cases, appShellCasesForTest()...)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal block system report: %v", err)
	}
	return raw
}

func validHeadlessMorphSurfaceReportJSON(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessBlockSystemSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode headless Block system report: %v", err)
	}
	morph := validMorphEvidenceMap()
	if mutate != nil {
		mutate(morph)
	}
	report["morph"] = morph
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal Morph report: %v", err)
	}
	return raw
}

func validMorphEvidenceMap() map[string]any {
	return map[string]any{
		"schema":            "tetra.surface.morph.v1",
		"quality_level":     "deterministic-headless-morph-capsule-v1",
		"source":            "examples/surface_morph_command_palette.tetra",
		"module":            "lib.core.morph",
		"surface_scope":     "surface-morph-experimental-linux-web",
		"experimental":      true,
		"production_claim":  false,
		"git_head":          "e2c19b8ee276158f8eb2c54cf61e11bd84952893",
		"git_dirty":         true,
		"capsule_hash":      "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"token_graph_hash":  "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"capsule":           validMorphCapsuleMap(),
		"token_graph":       validMorphTokenGraphMap(),
		"style_graph":       validMorphStyleGraphMap(),
		"authoring":         validMorphAuthoringMap(),
		"materials":         validMorphMaterials(),
		"layout_modes":      []any{"row", "column", "stack", "grid", "dock", "absolute", "overlay", "scroll"},
		"typography_roles":  []any{"title", "body", "label", "code"},
		"asset_refs":        validMorphAssetRefs(),
		"affordances":       validMorphAffordances(),
		"state_lenses":      validMorphStateLenses(),
		"motion_presets":    validMorphMotionPresets(),
		"recipes":           validMorphRecipes(),
		"recipe_expansions": validMorphRecipeExpansions(),
		"accessibility":     validMorphAccessibilityProjectionMap(),
		"evidence_contract": validMorphEvidenceContractMap(),
		"memory_budget":     validMorphMemoryBudgetMap(),
		"negative_guards":   validMorphNegativeGuardsMap(),
		"nonclaims":         []any{"DOM runtime absent", "React runtime absent", "Electron claim absent", "platform-native widgets absent", "full screen-reader production absent", "CSS cascade absent"},
	}
}

func validMorphCapsuleMap() map[string]any {
	return map[string]any{
		"namespace":         "tetra.surface.morph.app",
		"version":           "1",
		"capsule_hash":      "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"imports":           []any{"lib.core.block", "lib.core.morph"},
		"explicit_imports":  true,
		"no_global_cascade": true,
	}
}

func validMorphTokenGraphMap() map[string]any {
	return map[string]any{
		"schema":                       "tetra.surface.morph.token-graph.v1",
		"namespace":                    "tetra.surface.morph.app",
		"version":                      "1",
		"hash":                         "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"categories":                   []any{"color", "space", "spacing", "radius", "border", "elevation", "opacity", "typography", "type", "motion", "z", "assets", "density"},
		"tokens":                       validMorphTokens(),
		"alias_cycle_rejected":         true,
		"duplicate_source_rejected":    true,
		"raw_literals_in_app_code":     false,
		"unresolved_fallback_rejected": true,
		"fallback_to_random_default":   false,
	}
}

func validMorphStyleGraphMap() map[string]any {
	return map[string]any{
		"schema":                            "tetra.surface.morph.style-graph.v1",
		"namespace":                         "tetra.surface.morph.app",
		"version":                           "1",
		"css_replacement_level":             "typed-style-graph-candidate-v1",
		"vocabulary_frozen":                 true,
		"token_categories":                  []any{"color", "space", "spacing", "radius", "border", "elevation", "opacity", "typography", "type", "motion", "z", "assets", "density"},
		"material_slots":                    []any{"fill", "border", "radius", "shadow", "overlay"},
		"affordance_roles":                  []any{"action", "field.text", "toggle", "navigation", "region", "overlay", "status"},
		"recipe_outputs":                    []any{"Block"},
		"state_selectors":                   []any{"hover", "pressed", "focusVisible", "selected", "disabled", "error", "loading"},
		"motion_properties":                 []any{"fill", "opacity", "transform"},
		"override_order":                    []any{"capsule", "tokens", "materials", "affordances", "state_lenses", "motion", "recipes", "accessibility_safety"},
		"conflict_diagnostics":              []any{"alias_cycle", "duplicate_recipe", "duplicate_token_source", "unresolved_token", "raw_literal", "unsupported_css_cascade", "forbidden_runtime_import", "global_style_leak", "specificity_ambiguity", "raw_css_runtime_import"},
		"import_allowlist":                  []any{"lib.core.block", "lib.core.morph"},
		"css_cascade_imports_rejected":      true,
		"dom_runtime_imports_rejected":      true,
		"react_runtime_imports_rejected":    true,
		"electron_runtime_imports_rejected": true,
		"selector_engine_absent":            true,
		"no_specificity_scoring":            true,
		"global_style_leak_rejected":        true,
		"specificity_ambiguity_rejected":    true,
		"raw_css_runtime_import_rejected":   true,
	}
}

func validMorphAuthoringMap() map[string]any {
	return map[string]any{
		"schema":                       "tetra.surface.morph.authoring.v1",
		"level":                        "production-recipe-authoring-v1",
		"recipe_count":                 11,
		"polished_recipe_count":        11,
		"max_author_fields":            12,
		"raw_block_field_count":        80,
		"raw_80_field_blocks_rejected": true,
		"recipes_required":             true,
		"direct_block_prop_editing":    false,
		"recipe_first_authoring":       true,
		"designer_token_inputs":        true,
		"generated_block_props_only":   true,
		"raw_literal_styles_rejected":  true,
		"nonclaims":                    []any{"raw 80-field Block authoring", "CSS cascade", "selector engine", "specificity scoring"},
	}
}

func validMorphTokens() []any {
	return []any{
		map[string]any{"id": "color.bg", "category": "color", "kind": "rgba", "value": "#0b0f14ff", "source": "capsule", "hash": "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		map[string]any{"id": "space.3", "category": "space", "kind": "px", "value": "12", "source": "capsule", "hash": "sha256:2222222222222222222222222222222222222222222222222222222222222222"},
		map[string]any{"id": "radius.md", "category": "radius", "kind": "px", "value": "10", "source": "capsule", "hash": "sha256:3333333333333333333333333333333333333333333333333333333333333333"},
		map[string]any{"id": "type.label", "category": "typography", "kind": "font", "value": "Tetra UI 13 600 18", "source": "capsule", "hash": "sha256:4444444444444444444444444444444444444444444444444444444444444444"},
		map[string]any{"id": "motion.fast", "category": "motion", "kind": "transition", "value": "120 ease.out", "source": "capsule", "hash": "sha256:5555555555555555555555555555555555555555555555555555555555555555"},
	}
}

func validMorphMaterials() []any {
	return []any{
		map[string]any{"name": "surface.base", "paint_stack": []any{"fill", "border", "radius"}, "fill": "color.surface", "border": "border.subtle", "radius": "radius.md", "shadow": "", "overlay": "", "unsupported_blur": false, "unsupported_blur_rejected": true},
		map[string]any{"name": "surface.elevated", "paint_stack": []any{"fill", "border", "radius", "shadow"}, "fill": "color.surface", "border": "border.subtle", "radius": "radius.md", "shadow": "elevation.2", "overlay": "", "unsupported_blur": false, "unsupported_blur_rejected": true},
		map[string]any{"name": "control.primary", "paint_stack": []any{"fill", "radius"}, "fill": "color.accent", "border": "", "radius": "radius.sm", "shadow": "", "overlay": "", "unsupported_blur": false, "unsupported_blur_rejected": true},
		map[string]any{"name": "translucent.panel", "paint_stack": []any{"fill", "border", "radius", "shadow", "overlay"}, "fill": "color.surfaceAlpha", "border": "border.glass", "radius": "radius.lg", "shadow": "elevation.3", "overlay": "gradient.vertical", "unsupported_blur": false, "unsupported_blur_rejected": true},
	}
}

func validMorphAssetRefs() []any {
	return []any{
		map[string]any{"id": "project.new", "kind": "icon", "sha256": "sha256:6666666666666666666666666666666666666666666666666666666666666666", "local": true, "fallback_id": "icon.fallback", "tint_token": "color.accent"},
		map[string]any{"id": "command.search", "kind": "icon", "sha256": "sha256:7777777777777777777777777777777777777777777777777777777777777777", "local": true, "fallback_id": "icon.fallback", "tint_token": "color.muted"},
		map[string]any{"id": "status.warning", "kind": "icon", "sha256": "sha256:8888888888888888888888888888888888888888888888888888888888888888", "local": true, "fallback_id": "icon.fallback", "tint_token": "color.warning"},
	}
}

func validMorphAffordances() []any {
	return []any{
		map[string]any{"name": "action", "role": "button", "focusable": true, "action": "activate", "input": "", "projects_accessibility": true},
		map[string]any{"name": "field.text", "role": "textbox", "focusable": true, "action": "edit", "input": "editable_text", "projects_accessibility": true},
		map[string]any{"name": "toggle", "role": "checkbox", "focusable": true, "action": "toggle", "input": "toggle", "projects_accessibility": true},
		map[string]any{"name": "navigation", "role": "navigation", "focusable": false, "action": "", "input": "", "projects_accessibility": true},
		map[string]any{"name": "region", "role": "region", "focusable": false, "action": "", "input": "", "projects_accessibility": true},
		map[string]any{"name": "overlay", "role": "dialog", "focusable": true, "action": "dismiss", "input": "focus_trap", "projects_accessibility": true},
		map[string]any{"name": "status", "role": "status", "focusable": false, "action": "", "input": "", "projects_accessibility": true},
	}
}

func validMorphStateLenses() []any {
	return []any{
		map[string]any{"selector": "hover", "property": "paint.overlay", "deterministic": true},
		map[string]any{"selector": "pressed", "property": "transform.scale", "deterministic": true},
		map[string]any{"selector": "focusVisible", "property": "paint.outline", "deterministic": true},
		map[string]any{"selector": "selected", "property": "accessibility.selected", "deterministic": true},
		map[string]any{"selector": "disabled", "property": "input.disabled", "deterministic": true},
		map[string]any{"selector": "error", "property": "paint.outline_color", "deterministic": true},
		map[string]any{"selector": "loading", "property": "text.content", "deterministic": true},
	}
}

func validMorphMotionPresets() []any {
	return []any{
		map[string]any{"name": "motion.fast", "duration_ms": 120, "curve": "ease.out", "properties": []any{"fill", "opacity", "transform"}, "reduced_motion": true, "deterministic_time": true},
		map[string]any{"name": "motion.soft", "duration_ms": 180, "curve": "ease.inOut", "properties": []any{"fill", "opacity", "transform"}, "reduced_motion": true, "deterministic_time": true},
	}
}

func validMorphRecipes() []any {
	return []any{
		validMorphRecipe("control.action@1", "control.action", []any{"label", "icon"}, []any{"text", "action", "variant"}, []any{"pressed", "focused"}, []any{"role:button", "name", "action"}),
		validMorphRecipe("field.text@1", "field.text", []any{"label", "control"}, []any{"value", "on_text"}, []any{"focused", "error"}, []any{"role:textbox", "labelled_by", "value"}),
		validMorphRecipe("control.toggle@1", "control.toggle", []any{"label", "control"}, []any{"checked", "on_toggle"}, []any{"checked", "focused"}, []any{"role:checkbox", "checked", "name"}),
		validMorphRecipe("command.item@1", "command.item", []any{"icon", "title", "subtitle"}, []any{"title", "subtitle", "icon", "selected"}, []any{"selected", "focused"}, []any{"role:button", "selected", "description"}),
		validMorphRecipe("navigation.item@1", "navigation.item", []any{"label", "badge"}, []any{"route", "selected"}, []any{"selected", "focused"}, []any{"role:navigation", "current", "name"}),
		validMorphRecipe("region.panel@1", "region.panel", []any{"header", "body", "actions"}, []any{"title"}, []any{"expanded", "loading"}, []any{"role:region", "labelled_by", "bounds"}),
		validMorphRecipe("overlay.dialog@1", "overlay.dialog", []any{"title", "body", "actions"}, []any{"open", "dismiss"}, []any{"open", "focus_trap"}, []any{"role:dialog", "modal", "name"}),
		validMorphRecipe("navigation.tabs@1", "navigation.tabs", []any{"tab", "panel"}, []any{"items", "active"}, []any{"active", "focused"}, []any{"role:tablist", "selected", "controls"}),
		validMorphRecipe("collection.list@1", "collection.list", []any{"item", "empty"}, []any{"items", "selected"}, []any{"selected", "empty"}, []any{"role:list", "item_count", "selected"}),
		validMorphRecipe("collection.table-lite@1", "collection.table-lite", []any{"header", "row", "cell"}, []any{"rows", "columns"}, []any{"sorted", "selected"}, []any{"role:table", "row_count", "column_count"}),
		validMorphRecipe("status.message@1", "status.message", []any{"icon", "message"}, []any{"kind", "text"}, []any{"severity", "live"}, []any{"role:status", "live", "name"}),
	}
}

func validMorphRecipe(name string, family string, slots []any, inputs []any, state []any, a11y []any) map[string]any {
	return map[string]any{
		"name":                     name,
		"family":                   family,
		"output":                   "Block",
		"slots":                    slots,
		"inputs":                   inputs,
		"state":                    state,
		"a11y":                     a11y,
		"expands_to_block_graph":   true,
		"hidden_app_state":         false,
		"platform_widgets":         false,
		"core_primitive_promotion": false,
	}
}

func validMorphRecipeExpansions() []any {
	return []any{
		map[string]any{"recipe": "control.action@1", "block_ids": []any{4}, "slot_bindings": []any{"label", "icon"}, "variant": "primary", "reported": true},
		map[string]any{"recipe": "field.text@1", "block_ids": []any{3}, "slot_bindings": []any{"label", "control"}, "variant": "default", "reported": true},
		map[string]any{"recipe": "control.toggle@1", "block_ids": []any{5}, "slot_bindings": []any{"label", "control"}, "variant": "checked", "reported": true},
		map[string]any{"recipe": "command.item@1", "block_ids": []any{4, 5}, "slot_bindings": []any{"icon", "title", "subtitle"}, "variant": "selected", "reported": true},
		map[string]any{"recipe": "navigation.item@1", "block_ids": []any{4}, "slot_bindings": []any{"label", "badge"}, "variant": "current", "reported": true},
		map[string]any{"recipe": "region.panel@1", "block_ids": []any{2}, "slot_bindings": []any{"header", "body", "actions"}, "variant": "elevated", "reported": true},
		map[string]any{"recipe": "overlay.dialog@1", "block_ids": []any{2, 4, 5}, "slot_bindings": []any{"title", "body", "actions"}, "variant": "modal", "reported": true},
		map[string]any{"recipe": "navigation.tabs@1", "block_ids": []any{2, 4, 5}, "slot_bindings": []any{"tab", "panel"}, "variant": "compact", "reported": true},
		map[string]any{"recipe": "collection.list@1", "block_ids": []any{2, 4, 5}, "slot_bindings": []any{"item", "empty"}, "variant": "virtual-lite", "reported": true},
		map[string]any{"recipe": "collection.table-lite@1", "block_ids": []any{2, 4, 5}, "slot_bindings": []any{"header", "row", "cell"}, "variant": "dense", "reported": true},
		map[string]any{"recipe": "status.message@1", "block_ids": []any{5}, "slot_bindings": []any{"icon", "message"}, "variant": "warning", "reported": true},
	}
}

func validMorphAccessibilityProjectionMap() map[string]any {
	return map[string]any{
		"schema":                   "tetra.surface.morph.accessibility-projection.v1",
		"derived_from_block_graph": true,
		"safety_overrides_win":     true,
		"snapshot_evidence":        true,
		"required_fields":          []any{"role", "name", "description", "action", "state", "bounds", "focus_order", "reading_order", "labelled_by", "label_for"},
		"roles":                    []any{"button", "textbox", "checkbox", "navigation", "region", "dialog", "status"},
	}
}

func validMorphEvidenceContractMap() map[string]any {
	return map[string]any{
		"capsule_hash":       "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"token_graph_hash":   "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"recipe_expansions":  true,
		"block_tree":         true,
		"resolved_layout":    true,
		"paint_layers":       true,
		"text_runs":          true,
		"motion_frames":      true,
		"asset_hashes":       true,
		"accessibility_tree": true,
		"memory_budget":      true,
		"frame_checksums":    true,
		"artifact_hashes":    true,
	}
}

func validMorphMemoryBudgetMap() map[string]any {
	return map[string]any{
		"schema":                   "tetra.surface.morph-memory-budget.v1",
		"expanded_recipe_count":    4,
		"block_count":              24,
		"paint_command_count":      6,
		"layout_pass_count":        8,
		"text_run_count":           3,
		"motion_active_count":      1,
		"glyph_cache_bytes":        4096,
		"asset_cache_bytes":        5376,
		"layout_cache_bytes":       8192,
		"framebuffer_bytes":        256000,
		"peak_rss_bytes":           0,
		"alloc_count":              0,
		"frame_count":              3,
		"bounded_caches":           true,
		"unbounded_cache_rejected": true,
	}
}

func validMorphNegativeGuardsMap() map[string]any {
	return map[string]any{
		"no_core_widget_primitives":          true,
		"no_dom_ui":                          true,
		"no_react":                           true,
		"no_electron":                        true,
		"no_user_js":                         true,
		"no_platform_widgets":                true,
		"missing_token_rejected":             true,
		"alias_cycle_rejected":               true,
		"duplicate_token_source_rejected":    true,
		"duplicate_recipe_name_rejected":     true,
		"missing_recipe_expansion_rejected":  true,
		"unresolved_token_rejected":          true,
		"missing_asset_rejected":             true,
		"unbounded_cache_rejected":           true,
		"fake_motion_rejected":               true,
		"fake_accessibility_rejected":        true,
		"unsupported_target_rejected":        true,
		"dirty_checkout_production_rejected": true,
	}
}

func validLinuxX64RealWindowBlockSystemSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(validHeadlessBlockSystemSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode headless Block system report: %v", err)
	}
	report.Target = "linux-x64"
	if report.LayoutEngine != nil {
		report.LayoutEngine.Target = report.Target
	}
	if report.AppModel != nil {
		report.AppModel.Target = report.Target
	}
	if report.KeyboardUX != nil {
		report.KeyboardUX.Target = report.Target
	}
	if report.AppShell != nil {
		retargetAppShellForTest(report.AppShell, report.Target)
	}
	report.Runtime = "surface-linux-x64"
	report.HostEvidence = HostEvidenceReport{
		Level:       "linux-x64-real-window",
		Backend:     "wayland-shm-rgba",
		Framebuffer: true,
		RealWindow:  true,
		NativeInput: true,
	}
	report.Processes = []ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_system.tetra -o /tmp/surface-artifacts/surface-block-system", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-system", Ran: true, Pass: true, ExitCode: intPtrForTest(1), ExpectedExitCode: intPtrForTest(1)},
		{Name: "surface linux-x64 real-window probe", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-system-real-window-probe", Ran: true, Pass: true, ExitCode: intPtrForTest(42), ExpectedExitCode: intPtrForTest(42)},
		{Name: "surface linux-x64 runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode linux-x64-real-window-block-system", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
	}
	report.Artifacts = []ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-block-system", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 49172},
	}
	report.ArtifactScan = ArtifactScanReport{Root: "/tmp/surface-artifacts", FilesChecked: 1, ForbiddenPaths: nil, Pass: true}
	report.Frames = []FrameReport{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", Presented: true},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, Checksum: "2222222222222222222222222222222222222222222222222222222222222222", Presented: true},
		{Order: 3, Width: 320, Height: 200, Stride: 1280, Checksum: "3333333333333333333333333333333333333333333333333333333333333333", Presented: true},
		{Order: 5, Width: 400, Height: 240, Stride: 1600, Checksum: "5555555555555555555555555555555555555555555555555555555555555555", Presented: true},
	}
	report.BlockSystem.QualityLevel = "linux-x64-real-window-block-system-v1"
	report.BlockSystem.Renderer = "wayland-shm-rgba"
	report.BlockSystem.GoldenSet = "surface-block-system-linux-x64-real-window-v1"
	report.BlockSystem.FrameCount = 4
	report.BlockSystem.GoldenHash = "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	report.BlockSystem.Frames = []BlockSystemFrameReport{
		{Order: 1, Label: "initial", Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", RepeatChecksum: "1111111111111111111111111111111111111111111111111111111111111111", GoldenChecksum: "1111111111111111111111111111111111111111111111111111111111111111", PaintEvidence: true, LayoutEvidence: true, AccessibilityEvidence: true},
		{Order: 2, Label: "focused", Width: 320, Height: 200, Stride: 1280, Checksum: "2222222222222222222222222222222222222222222222222222222222222222", RepeatChecksum: "2222222222222222222222222222222222222222222222222222222222222222", GoldenChecksum: "2222222222222222222222222222222222222222222222222222222222222222", PaintEvidence: true, LayoutEvidence: true, AccessibilityEvidence: true},
		{Order: 3, Label: "motion", Width: 320, Height: 200, Stride: 1280, Checksum: "3333333333333333333333333333333333333333333333333333333333333333", RepeatChecksum: "3333333333333333333333333333333333333333333333333333333333333333", GoldenChecksum: "3333333333333333333333333333333333333333333333333333333333333333", PaintEvidence: true, LayoutEvidence: true, AccessibilityEvidence: true},
		{Order: 5, Label: "real-window-focused", Width: 400, Height: 240, Stride: 1600, Checksum: "5555555555555555555555555555555555555555555555555555555555555555", RepeatChecksum: "5555555555555555555555555555555555555555555555555555555555555555", GoldenChecksum: "5555555555555555555555555555555555555555555555555555555555555555", PaintEvidence: true, LayoutEvidence: true, AccessibilityEvidence: true},
	}
	report.Events = appendEventReportsWithNextOrder(report.Events, []EventReport{
		{Kind: "resize", TargetComponent: "BlockSystemApp", DispatchPath: []string{"BlockSystemApp"}, Handled: true, Pass: true, Width: 400, Height: 240, TimestampMS: 4, BufferSlots: []int{2, 0, 0, 0, 0, 400, 240, 4, 0}, BeforeState: map[string]string{"BlockSystemApp.width": "320"}, AfterState: map[string]string{"BlockSystemApp.width": "400"}},
		{Kind: "close", TargetComponent: "BlockSystemApp", DispatchPath: []string{"BlockSystemApp"}, Handled: true, Pass: true, Width: 400, Height: 240, TimestampMS: 5, BufferSlots: []int{1, 0, 0, 0, 0, 400, 240, 5, 0}, BeforeState: map[string]string{"BlockSystemApp.closed": "false"}, AfterState: map[string]string{"BlockSystemApp.closed": "true"}},
	})
	report.StateTransitions = appendStateTransitionReportsWithNextOrder(report.StateTransitions, []StateTransitionReport{
		{Component: "SubmitBlock", Field: "pressed", Before: "false", After: "true", Cause: "key_down"},
		{Component: "BlockSystemApp", Field: "width", Before: "320", After: "400", Cause: "resize"},
		{Component: "BlockSystemApp", Field: "closed", Before: "false", After: "true", Cause: "close"},
	})
	report.Cases = blockSystemLinuxX64RealWindowCasesForTest()
	report.RendererScene = rendererSceneForTest(report.Source, report.BlockSystem.Renderer)
	report.SoftwareRenderer = softwareRendererForTest(report.Source, report.Target, report.BlockSystem.Renderer, report.Frames)
	report.BlockSystem.MemoryBudget = blockMemoryBudgetForTest(&report)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal linux-x64 real-window Block system report: %v", err)
	}
	return raw
}

func validWASM32WebBrowserCanvasBlockSystemSurfaceReportJSON(t *testing.T, mutate func(*Report)) []byte {
	t.Helper()
	var report Report
	if err := json.Unmarshal(validHeadlessBlockSystemSurfaceReportJSON(t, nil), &report); err != nil {
		t.Fatalf("decode headless Block system report: %v", err)
	}
	report.Target = "wasm32-web"
	if report.LayoutEngine != nil {
		report.LayoutEngine.Target = report.Target
	}
	if report.AppModel != nil {
		report.AppModel.Target = report.Target
	}
	if report.KeyboardUX != nil {
		report.KeyboardUX.Target = report.Target
	}
	if report.AppShell != nil {
		retargetAppShellForTest(report.AppShell, report.Target)
	}
	report.Runtime = "surface-wasm32-web"
	report.HostEvidence = HostEvidenceReport{
		Level:         "wasm32-web-browser-canvas-input",
		Backend:       "browser-canvas-rgba",
		Framebuffer:   true,
		NativeInput:   true,
		BrowserCanvas: true,
		BrowserInput:  true,
	}
	report.Processes = []ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target wasm32-web examples/surface_block_system.tetra -o /tmp/surface-artifacts/surface-block-system.wasm", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
		{Name: "surface wasm32-web browser canvas component app", Kind: "app", Path: "/usr/bin/chromium --headless <surface-browser-canvas-runner> scenario=block-system wasm=/tmp/surface-artifacts/surface-block-system.wasm", Ran: true, Pass: true, ExitCode: intPtrForTest(0), ExpectedExitCode: intPtrForTest(0)},
		{Name: "surface wasm32-web import validator", Kind: "runtime", Path: "go run ./tools/cmd/validate-wasm-imports --target wasm32-web /tmp/surface-artifacts/surface-block-system.wasm", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
		{Name: "surface wasm32-web browser canvas runtime", Kind: "runtime", Path: "Chromium Block-system fixture", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
		{Name: "surface wasm32-web browser canvas trace", Kind: "runtime", Path: "/usr/bin/chromium --headless --dump-dom <surface-browser-canvas-file-runner scenario=block-system>", Ran: true, Pass: true, ExitCode: intPtrForTest(0)},
	}
	report.Artifacts = []ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-block-system.wasm", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 8604},
		{Kind: "compiler-owned-loader", Path: "/tmp/surface-artifacts/surface-block-system.mjs", SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Size: 4939},
		{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 1184},
	}
	report.ArtifactScan = ArtifactScanReport{Root: "/tmp/surface-artifacts", FilesChecked: 3, ForbiddenPaths: nil, Pass: true}
	report.Frames = []FrameReport{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", Presented: true},
		{Order: 3, Width: 320, Height: 200, Stride: 1280, Checksum: "3333333333333333333333333333333333333333333333333333333333333333", Presented: true},
		{Order: 5, Width: 400, Height: 240, Stride: 1600, Checksum: "5555555555555555555555555555555555555555555555555555555555555555", Presented: true},
	}
	report.BlockSystem.QualityLevel = "wasm32-web-browser-canvas-block-system-v1"
	report.BlockSystem.Renderer = "browser-canvas-rgba"
	report.BlockSystem.GoldenSet = "surface-block-system-wasm32-web-browser-canvas-v1"
	report.BlockSystem.FrameCount = 3
	report.BlockSystem.GoldenHash = "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	report.BlockSystem.Frames = []BlockSystemFrameReport{
		{Order: 1, Label: "initial", Width: 320, Height: 200, Stride: 1280, Checksum: "1111111111111111111111111111111111111111111111111111111111111111", RepeatChecksum: "1111111111111111111111111111111111111111111111111111111111111111", GoldenChecksum: "1111111111111111111111111111111111111111111111111111111111111111", PaintEvidence: true, LayoutEvidence: true, AccessibilityEvidence: true},
		{Order: 3, Label: "motion", Width: 320, Height: 200, Stride: 1280, Checksum: "3333333333333333333333333333333333333333333333333333333333333333", RepeatChecksum: "3333333333333333333333333333333333333333333333333333333333333333", GoldenChecksum: "3333333333333333333333333333333333333333333333333333333333333333", PaintEvidence: true, LayoutEvidence: true, AccessibilityEvidence: true},
		{Order: 5, Label: "browser-canvas-focused", Width: 400, Height: 240, Stride: 1600, Checksum: "5555555555555555555555555555555555555555555555555555555555555555", RepeatChecksum: "5555555555555555555555555555555555555555555555555555555555555555", GoldenChecksum: "5555555555555555555555555555555555555555555555555555555555555555", PaintEvidence: true, LayoutEvidence: true, AccessibilityEvidence: true},
	}
	report.Events = appendEventReportsWithNextOrder(report.Events, []EventReport{
		{Kind: "resize", TargetComponent: "BlockSystemApp", DispatchPath: []string{"BlockSystemApp"}, Handled: true, Pass: true, Width: 400, Height: 240, TimestampMS: 4, BufferSlots: []int{2, 0, 0, 0, 0, 400, 240, 4, 0}, BeforeState: map[string]string{"BlockSystemApp.width": "320"}, AfterState: map[string]string{"BlockSystemApp.width": "400"}},
	})
	report.StateTransitions = appendStateTransitionReportsWithNextOrder(report.StateTransitions, []StateTransitionReport{
		{Component: "SubmitBlock", Field: "pressed", Before: "false", After: "true", Cause: "key_down"},
		{Component: "BlockSystemApp", Field: "width", Before: "320", After: "400", Cause: "resize"},
	})
	report.Cases = blockSystemWASM32WebBrowserCanvasCasesForTest()
	report.RendererScene = rendererSceneForTest(report.Source, report.BlockSystem.Renderer)
	report.SoftwareRenderer = softwareRendererForTest(report.Source, report.Target, report.BlockSystem.Renderer, report.Frames)
	report.BlockSystem.MemoryBudget = blockMemoryBudgetForTest(&report)
	if mutate != nil {
		mutate(&report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal wasm32-web browser-canvas Block system report: %v", err)
	}
	return raw
}

func appModelForTest(target string) *SurfaceAppModelReport {
	return &SurfaceAppModelReport{
		Schema:                 AppModelSchemaV1,
		Level:                  "production-app-model-v1",
		Target:                 target,
		StateStoreLevel:        "owned-state-store-v1",
		CommandPolicy:          "typed-command-dispatch-v1",
		EventRoutingPolicy:     "block-event-trace-v1",
		AsyncCommandPolicy:     "actor-task-safe-boundary-v1",
		NavigationFocusPolicy:  "navigation-focus-scopes-v1",
		ShortcutScopePolicy:    "scoped-shortcuts-v1",
		ErrorPropagationPolicy: "command-error-propagation-v1",
		RedrawSchedulingPolicy: "explicit-redraw-invalidation-v1",
		ActorTaskBoundary:      "safe-app-model-boundary-v1",
		AppSurfaces:            []string{"command_palette", "dashboard", "settings", "editor_shell"},
		StateStores: []AppStateStoreReport{
			{ID: "store.command_palette", Owner: "command_palette", Scope: "overlay", Fields: []string{"query", "selected", "open"}, ComputedBindings: []string{"filtered_commands"}, Invalidates: []string{"command_list", "focus_scope"}, SnapshotBefore: "query='',selected=0,open=false", SnapshotAfter: "query='open',selected=1,open=true"},
			{ID: "store.dashboard", Owner: "dashboard", Scope: "page", Fields: []string{"cards", "filter", "loading"}, ComputedBindings: []string{"visible_cards"}, Invalidates: []string{"dashboard_grid"}, SnapshotBefore: "cards=3,loading=true", SnapshotAfter: "cards=4,loading=false"},
			{ID: "store.settings", Owner: "settings", Scope: "form", Fields: []string{"theme", "density", "saved"}, ComputedBindings: []string{"dirty"}, Invalidates: []string{"settings_form"}, SnapshotBefore: "theme=dark,density=comfortable,saved=true", SnapshotAfter: "theme=light,density=compact,saved=false"},
			{ID: "store.editor_shell", Owner: "editor_shell", Scope: "document", Fields: []string{"buffer", "selection", "undo_depth", "redo_depth"}, ComputedBindings: []string{"line_count"}, Invalidates: []string{"editor_view", "status"}, SnapshotBefore: "buffer='',undo=0,redo=0", SnapshotAfter: "buffer='OK',undo=1,redo=0"},
		},
		Commands: []AppCommandReport{
			{ID: "command_palette.open", Kind: "command", Source: "shortcut", Target: "SubmitBlock", StoreID: "store.command_palette", EventTraceID: "trace.open_palette", Mutates: []string{"open", "query"}, RequestsRedraw: true, SafeBoundary: true},
			{ID: "command_palette.choose", Kind: "command", Source: "click", Target: "SubmitBlock", StoreID: "store.command_palette", EventTraceID: "trace.choose_command", Mutates: []string{"selected"}, RequestsRedraw: true, SafeBoundary: true},
			{ID: "dashboard.refresh", Kind: "async", Source: "timer", Target: "PanelBlock", StoreID: "store.dashboard", EventTraceID: "trace.async_refresh", Mutates: []string{"cards", "loading"}, RequestsRedraw: true, ErrorPath: "dashboard.error", Async: true, SafeBoundary: true},
			{ID: "settings.save", Kind: "command", Source: "click", Target: "ResetBlock", StoreID: "store.settings", EventTraceID: "trace.disabled_save", Mutates: []string{"saved"}, RequestsRedraw: true, ErrorPath: "settings.error", SafeBoundary: true},
			{ID: "editor.insert_text", Kind: "text-edit", Source: "text", Target: "InputBlock", StoreID: "store.editor_shell", EventTraceID: "trace.editor_text", Mutates: []string{"buffer", "selection", "undo_depth"}, RequestsRedraw: true, SafeBoundary: true},
			{ID: "editor.undo", Kind: "undo", Source: "shortcut", Target: "InputBlock", StoreID: "store.editor_shell", EventTraceID: "trace.undo", Mutates: []string{"buffer", "undo_depth", "redo_depth"}, RequestsRedraw: true, SafeBoundary: true},
			{ID: "editor.redo", Kind: "redo", Source: "shortcut", Target: "InputBlock", StoreID: "store.editor_shell", EventTraceID: "trace.redo", Mutates: []string{"buffer", "undo_depth", "redo_depth"}, RequestsRedraw: true, SafeBoundary: true},
			{ID: "navigation.focus_next", Kind: "navigation", Source: "key", Target: "ResetBlock", StoreID: "store.command_palette", EventTraceID: "trace.focus_next", Mutates: []string{"selected"}, RequestsRedraw: true, SafeBoundary: true},
			{ID: "shortcut.run", Kind: "shortcut", Source: "key", Target: "SubmitBlock", StoreID: "store.command_palette", EventTraceID: "trace.shortcut", Mutates: []string{"selected"}, RequestsRedraw: true, SafeBoundary: true},
		},
		EventTraces: []AppEventTraceReport{
			{Order: 1, ID: "trace.open_palette", EventKind: "key", TargetBlock: "SubmitBlock", CommandID: "command_palette.open", StoreID: "store.command_palette", DispatchPath: []int{1, 2, 4}, StateBefore: "open=false", StateAfter: "open=true", Delivered: true, FocusedBlock: "SubmitBlock"},
			{Order: 2, ID: "trace.choose_command", EventKind: "click", TargetBlock: "SubmitBlock", CommandID: "command_palette.choose", StoreID: "store.command_palette", DispatchPath: []int{1, 2, 4}, StateBefore: "selected=0", StateAfter: "selected=1", Delivered: true, FocusedBlock: "SubmitBlock"},
			{Order: 3, ID: "trace.disabled_save", EventKind: "click", TargetBlock: "ResetBlock", CommandID: "settings.save", StoreID: "store.settings", DispatchPath: []int{1, 2, 5}, StateBefore: "saved=true", StateAfter: "saved=true", Rejected: true, RejectReason: "disabled control"},
			{Order: 4, ID: "trace.unfocused_text", EventKind: "text_input", TargetBlock: "InputBlock", CommandID: "editor.insert_text", StoreID: "store.editor_shell", DispatchPath: []int{1, 2, 4}, StateBefore: "buffer=''", StateAfter: "buffer=''", Rejected: true, RejectReason: "unfocused text target", FocusedBlock: "ResetBlock", TextLen: 2},
			{Order: 5, ID: "trace.editor_text", EventKind: "text_input", TargetBlock: "InputBlock", CommandID: "editor.insert_text", StoreID: "store.editor_shell", DispatchPath: []int{1, 2, 4}, StateBefore: "buffer=''", StateAfter: "buffer='OK'", Delivered: true, FocusedBlock: "InputBlock", TextLen: 2},
			{Order: 6, ID: "trace.focus_next", EventKind: "key", TargetBlock: "ResetBlock", CommandID: "navigation.focus_next", StoreID: "store.command_palette", DispatchPath: []int{1, 2, 5}, StateBefore: "selected=1", StateAfter: "selected=2", Delivered: true, FocusedBlock: "ResetBlock"},
			{Order: 7, ID: "trace.shortcut", EventKind: "key", TargetBlock: "SubmitBlock", CommandID: "shortcut.run", StoreID: "store.command_palette", DispatchPath: []int{1, 2, 4}, StateBefore: "selected=2", StateAfter: "selected=3", Delivered: true, FocusedBlock: "SubmitBlock"},
			{Order: 8, ID: "trace.async_refresh", EventKind: "frame", TargetBlock: "PanelBlock", CommandID: "dashboard.refresh", StoreID: "store.dashboard", DispatchPath: []int{1, 2}, StateBefore: "loading=true", StateAfter: "loading=false", Delivered: true},
			{Order: 9, ID: "trace.undo", EventKind: "key", TargetBlock: "InputBlock", CommandID: "editor.undo", StoreID: "store.editor_shell", DispatchPath: []int{1, 2, 4}, StateBefore: "undo=1,redo=0", StateAfter: "undo=0,redo=1", Delivered: true, FocusedBlock: "InputBlock"},
			{Order: 10, ID: "trace.redo", EventKind: "key", TargetBlock: "InputBlock", CommandID: "editor.redo", StoreID: "store.editor_shell", DispatchPath: []int{1, 2, 4}, StateBefore: "undo=0,redo=1", StateAfter: "undo=1,redo=0", Delivered: true, FocusedBlock: "InputBlock"},
		},
		AsyncCommands: []AppAsyncCommandReport{
			{ID: "async.dashboard.refresh", CommandID: "dashboard.refresh", Policy: "actor-task-safe-boundary-v1", Boundary: "safe-app-model-boundary-v1", Started: true, Completed: true, ErrorPropagated: true, SafeBoundary: true},
		},
		NavigationSteps: []AppNavigationStepReport{
			{Order: 1, Kind: "focus_next", Before: "SubmitBlock", After: "ResetBlock", FocusScope: "command_palette", GraphDerived: true},
			{Order: 2, Kind: "focus_trap", Before: "EditorDialog", After: "EditorDialog", FocusScope: "editor_shell", FocusTrap: true, GraphDerived: true},
		},
		ShortcutScopes: []AppShortcutScopeReport{
			{ID: "shortcuts.global", Scope: "global", Commands: []string{"command_palette.open", "shortcut.run"}},
			{ID: "shortcuts.command_palette", Scope: "command_palette", Commands: []string{"command_palette.choose", "navigation.focus_next"}, FocusOnly: true},
			{ID: "shortcuts.editor", Scope: "editor_shell", Commands: []string{"editor.undo", "editor.redo", "editor.insert_text"}, FocusOnly: true},
		},
		ErrorReports: []AppErrorReport{
			{CommandID: "dashboard.refresh", Code: "async_load_failed", Propagated: true, Handled: true, StoreID: "store.dashboard"},
			{CommandID: "settings.save", Code: "disabled_control", Propagated: true, Handled: true, StoreID: "store.settings"},
		},
		RedrawRequests: []AppRedrawRequestReport{
			{Order: 1, Cause: "command_palette.open", StoreID: "store.command_palette", Invalidation: "overlay", FrameBefore: 1, FrameAfter: 2, BeforeChecksum: "1111111111111111111111111111111111111111111111111111111111111111", AfterChecksum: "2222222222222222222222222222222222222222222222222222222222222222"},
			{Order: 2, Cause: "editor.insert_text", StoreID: "store.editor_shell", Invalidation: "editor_view", FrameBefore: 2, FrameAfter: 3, BeforeChecksum: "2222222222222222222222222222222222222222222222222222222222222222", AfterChecksum: "3333333333333333333333333333333333333333333333333333333333333333"},
		},
		ReactRuntimeAbsent:         true,
		ReactHooksAbsent:           true,
		DOMEventsAbsent:            true,
		UserJSAbsent:               true,
		MissingEventTraceRejected:  true,
		DisabledDispatchRejected:   true,
		UnfocusedTextRejected:      true,
		UnsafeTaskBoundaryRejected: true,
		ReactRuntimeClaimRejected:  true,
		NonClaims:                  []string{"React runtime", "React hooks", "DOM events", "user-authored script logic", "unsafe actor/task boundary"},
	}
}

func keyboardUXForTest(target string) *SurfaceKeyboardUXReport {
	return &SurfaceKeyboardUXReport{
		Schema:                   KeyboardUXSchemaV1,
		Level:                    "production-keyboard-ux-v1",
		Target:                   target,
		FocusOrderPolicy:         "graph-focus-order-v1",
		FocusTrapPolicy:          "overlay-focus-trap-v1",
		RovingFocusPolicy:        "roving-focus-v1",
		KeyboardActivationPolicy: "keyboard-activation-v1",
		ShortcutConflictPolicy:   "scoped-shortcut-conflict-v1",
		UndoRedoPolicy:           "bounded-undo-redo-stack-v1",
		Surfaces:                 []string{"command_palette", "search", "settings_form", "editor_shell"},
		FocusOrder: []KeyboardFocusNodeReport{
			{Order: 1, BlockID: 4, Name: "InputBlock", Role: "textbox", Scope: "editor_shell", AccessibleName: "Editor input", LabelledBy: "LabelBlock", KeyboardReachable: true},
			{Order: 2, BlockID: 5, Name: "ResetBlock", Role: "button", Scope: "command_palette", AccessibleName: "Run command", KeyboardReachable: true},
		},
		FocusTransitions: []KeyboardFocusTransitionReport{
			{Order: 1, Key: "Tab", Before: 4, After: 5, Scope: "command_palette", Direction: "next", GraphDerived: true},
			{Order: 2, Key: "Shift_Tab", Before: 5, After: 4, Scope: "command_palette", Direction: "previous", Wrapped: true, GraphDerived: true},
			{Order: 3, Key: "ArrowDown", Before: 4, After: 5, Scope: "command_palette", Direction: "roving-next", GraphDerived: true},
		},
		FocusTraps: []KeyboardFocusTrapReport{
			{ID: "trap.command_palette", Scope: "command_palette", OverlayBlock: 2, EntryBlock: 4, FirstBlock: 4, LastBlock: 5, RestoreBlock: 4, LeakRejected: true, EscapeCloses: true},
		},
		RovingFocusGroups: []KeyboardRovingFocusGroupReport{
			{ID: "roving.command_results", Scope: "command_palette", ActiveBlock: 4, Items: []int{4, 5}, ArrowKeys: true, HomeEnd: true, Wrap: true, ConflictScope: "command_palette"},
		},
		KeyBindings: []KeyboardBindingReport{
			{ID: "binding.enter.activate", Key: "Enter", Scope: "command_palette", CommandID: "shortcut.run", BlockID: 5, Source: "key", Delivered: true},
			{ID: "binding.space.activate", Key: "Space", Scope: "command_palette", CommandID: "shortcut.run", BlockID: 5, Source: "key", Delivered: true},
			{ID: "binding.tab.next", Key: "Tab", Scope: "command_palette", CommandID: "navigation.focus_next", BlockID: 4, Source: "key", Delivered: true},
			{ID: "binding.shift_tab.prev", Key: "Shift_Tab", Scope: "command_palette", CommandID: "navigation.focus_next", BlockID: 5, Source: "key", Delivered: true},
			{ID: "binding.escape.close", Key: "Escape", Scope: "command_palette", CommandID: "command_palette.open", BlockID: 4, Source: "key", Delivered: true},
			{ID: "binding.ctrl_k.palette", Key: "Ctrl_K", Scope: "global", CommandID: "command_palette.open", BlockID: 4, Source: "shortcut", Delivered: true},
			{ID: "binding.ctrl_z.undo", Key: "Ctrl_Z", Scope: "editor_shell", CommandID: "editor.undo", BlockID: 4, Source: "shortcut", Delivered: true},
			{ID: "binding.ctrl_y.redo", Key: "Ctrl_Y", Scope: "editor_shell", CommandID: "editor.redo", BlockID: 4, Source: "shortcut", Delivered: true},
		},
		ShortcutConflicts: []KeyboardShortcutConflictReport{
			{Key: "Ctrl+K", Scope: "command_palette", CommandIDs: []string{"command_palette.open", "shortcut.run"}, Diagnosed: true, Rejected: true, Message: "Ctrl+K conflict is scoped and rejected"},
		},
		UndoRedoStacks: []KeyboardUndoRedoStackReport{
			{ID: "undo.editor_shell", Scope: "editor_shell", StoreID: "store.editor_shell", UndoCommandID: "editor.undo", RedoCommandID: "editor.redo", Units: []string{"insert-OK"}, Before: "undo=1,redo=0", AfterUndo: "undo=0,redo=1", AfterRedo: "undo=1,redo=0", Bounded: true, KeyboardDriven: true},
		},
		KeyboardScripts: []KeyboardScriptReport{
			{ID: "script.command_palette", Surface: "command_palette", Steps: []string{"Ctrl_K", "ArrowDown", "Enter"}, KeyboardOnly: true, FinalFocus: "ResetBlock", Pass: true},
			{ID: "script.search", Surface: "search", Steps: []string{"Ctrl_K", "type query", "Enter"}, KeyboardOnly: true, FinalFocus: "InputBlock", Pass: true},
			{ID: "script.settings", Surface: "settings_form", Steps: []string{"Tab", "Space", "Enter"}, KeyboardOnly: true, FinalFocus: "ResetBlock", Pass: true},
			{ID: "script.editor", Surface: "editor_shell", Steps: []string{"type OK", "Ctrl_Z", "Ctrl_Y"}, KeyboardOnly: true, FinalFocus: "InputBlock", Pass: true},
		},
		FocusableNameRejected:     true,
		OverlayFocusLeakRejected:  true,
		ShortcutConflictRejected:  true,
		PointerOnlyActionRejected: true,
		UnknownShortcutRejected:   true,
		UndoWithoutStackRejected:  true,
		NonClaims:                 []string{"pointer-only app", "global shortcut bypass", "screen-reader parity", "native platform widgets"},
	}
}

func appShellForTest(target string) *SurfaceAppShellReport {
	return &SurfaceAppShellReport{
		Schema:      AppShellSchemaV1,
		Level:       "production-app-shell-host-abi-v1",
		Target:      target,
		HostABI:     "tetra.surface.host-abi.v1",
		ShellPolicy: "block-app-shell-host-abi-v1",
		Capabilities: []AppShellCapabilityReport{
			{Kind: "window", Supported: true, HostTraceRequired: true},
			{Kind: "lifecycle", Supported: true, HostTraceRequired: true},
			{Kind: "menu", Supported: true, HostTraceRequired: true},
			{Kind: "context_menu", Supported: true, HostTraceRequired: true},
			{Kind: "dialog", Supported: true, HostTraceRequired: true},
			{Kind: "notification", Supported: true, HostTraceRequired: true},
			{Kind: "tray", Supported: true, HostTraceRequired: true},
			{Kind: "cursor", Supported: true, HostTraceRequired: true},
			{Kind: "drag_drop", Supported: true, HostTraceRequired: true},
			{Kind: "permission", Supported: true, HostTraceRequired: true},
			{Kind: "clipboard", Supported: true, HostTraceRequired: true},
			{Kind: "ime", Supported: true, HostTraceRequired: true},
			{Kind: "dpi_scale", Supported: true, HostTraceRequired: true},
			{Kind: "open_url", Supported: true, HostTraceRequired: true},
			{Kind: "open_file", Supported: true, HostTraceRequired: true},
			{Kind: "platform_widget_shell", Supported: false, DiagnosticRequired: true},
			{Kind: "native_global_menu", Supported: false, DiagnosticRequired: true},
		},
		HostReports: []AppShellHostReport{
			{ID: "host.window.create", Kind: "window", Target: target, TraceID: "trace.shell.window.create", Action: "create_show_focus_resize", Delivered: true, TimestampMS: 1},
			{ID: "host.lifecycle.ready", Kind: "lifecycle", Target: target, TraceID: "trace.shell.lifecycle.ready", Action: "app_start_ready_activate_before_quit_quit", Delivered: true, TimestampMS: 2},
			{ID: "host.menu.app", Kind: "menu", Target: target, TraceID: "trace.shell.menu.app", Action: "install_app_menu_and_dispatch", Delivered: true, TimestampMS: 3},
			{ID: "host.menu.context", Kind: "context_menu", Target: target, TraceID: "trace.shell.menu.context", Action: "show_context_menu_and_dispatch", Delivered: true, TimestampMS: 4},
			{ID: "host.dialog.file", Kind: "dialog", Target: target, TraceID: "trace.shell.dialog.file", Action: "file_open_picker", Delivered: true, TimestampMS: 5},
			{ID: "host.notification.deliver", Kind: "notification", Target: target, TraceID: "trace.shell.notification.deliver", Action: "notify", Delivered: true, TimestampMS: 6},
			{ID: "host.tray.install", Kind: "tray", Target: target, TraceID: "trace.shell.tray.install", Action: "install_status_item", Delivered: true, TimestampMS: 7},
			{ID: "host.cursor.apply", Kind: "cursor", Target: target, TraceID: "trace.shell.cursor.apply", Action: "apply_pointer_text_grab", Delivered: true, TimestampMS: 8},
			{ID: "host.drag_drop.drop", Kind: "drag_drop", Target: target, TraceID: "trace.shell.drag_drop.drop", Action: "drag_enter_drop", Delivered: true, TimestampMS: 9},
			{ID: "host.permission.query", Kind: "permission", Target: target, TraceID: "trace.shell.permission.query", Action: "query_permissions", Delivered: true, TimestampMS: 10},
			{ID: "host.clipboard.roundtrip", Kind: "clipboard", Target: target, TraceID: "trace.shell.clipboard.roundtrip", Action: "read_write_text", Delivered: true, TimestampMS: 11},
			{ID: "host.ime.preedit", Kind: "ime", Target: target, TraceID: "trace.shell.ime.preedit", Action: "composition_start_update_commit", Delivered: true, TimestampMS: 12},
			{ID: "host.dpi.scale", Kind: "dpi_scale", Target: target, TraceID: "trace.shell.dpi.scale", Action: "scale_change_resize", Delivered: true, TimestampMS: 13},
			{ID: "host.open.url", Kind: "open_url", Target: target, TraceID: "trace.shell.open.url", Action: "open_url", Delivered: true, TimestampMS: 14},
			{ID: "host.open.file", Kind: "open_file", Target: target, TraceID: "trace.shell.open.file", Action: "open_file", Delivered: true, TimestampMS: 15},
		},
		Windows: []AppShellWindowReport{
			{ID: "window.main", Title: "Surface Block System", Width: 400, Height: 240, MinWidth: 320, MinHeight: 200, Visible: true, Focused: true, Resizable: true, DPIAware: true, HostReportID: "host.window.create", ActionTraceID: "trace.shell.window.create"},
		},
		Lifecycle: []AppShellLifecycleReport{
			{Order: 1, Kind: "app_start", Delivered: true, HostReportID: "host.lifecycle.ready", ActionTraceID: "trace.shell.lifecycle.start"},
			{Order: 2, Kind: "ready", Delivered: true, HostReportID: "host.lifecycle.ready", ActionTraceID: "trace.shell.lifecycle.ready"},
			{Order: 3, Kind: "activate", Delivered: true, HostReportID: "host.lifecycle.ready", ActionTraceID: "trace.shell.lifecycle.activate"},
			{Order: 4, Kind: "before_quit", Delivered: true, HostReportID: "host.lifecycle.ready", ActionTraceID: "trace.shell.lifecycle.before_quit"},
			{Order: 5, Kind: "quit", Delivered: true, HostReportID: "host.lifecycle.ready", ActionTraceID: "trace.shell.lifecycle.quit"},
		},
		Menus: []AppShellMenuReport{
			{ID: "menu.app", Kind: "menu", Items: []string{"File/New", "File/Open", "App/Quit"}, TargetBlock: "BlockSystemApp", CommandID: "command_palette.open", Delivered: true, HostReportID: "host.menu.app", ActionTraceID: "trace.shell.menu.app"},
			{ID: "menu.context.editor", Kind: "context_menu", Items: []string{"Copy", "Paste", "Inspect"}, TargetBlock: "InputBlock", CommandID: "editor.insert_text", Delivered: true, HostReportID: "host.menu.context", ActionTraceID: "trace.shell.menu.context"},
		},
		Dialogs: []AppShellDialogReport{
			{ID: "dialog.file_open", Kind: "file_open", Title: "Open Tetra File", Result: "examples/surface_block_system.tetra", Delivered: true, HostReportID: "host.dialog.file", ActionTraceID: "trace.shell.dialog.file"},
			{ID: "dialog.message", Kind: "message", Title: "Surface Ready", Result: "ok", Delivered: true, HostReportID: "host.dialog.file", ActionTraceID: "trace.shell.dialog.message"},
		},
		TrayItems: []AppShellTrayReport{
			{ID: "tray.status", MenuID: "menu.app", Tooltip: "Surface Block System", ClickAction: "command_palette.open", Delivered: true, HostReportID: "host.tray.install", ActionTraceID: "trace.shell.tray.click"},
		},
		Notifications: []AppShellNotificationReport{
			{ID: "notification.ready", Title: "Surface Ready", Body: "Block System rendered", PermissionID: "permission.notification", Delivered: true, HostReportID: "host.notification.deliver"},
		},
		Cursors: []AppShellCursorReport{
			{Kind: "pointer", TargetBlock: "SubmitBlock", Applied: true, HostReportID: "host.cursor.apply"},
			{Kind: "text", TargetBlock: "InputBlock", Applied: true, HostReportID: "host.cursor.apply"},
			{Kind: "grab", TargetBlock: "ScrollBlock", Applied: true, HostReportID: "host.cursor.apply"},
		},
		DragDrop: []AppShellDragDropReport{
			{ID: "drag.enter", Kind: "drag_enter", SourceBlock: "InputBlock", TargetBlock: "PanelBlock", MIME: "text/plain", Delivered: true, HostReportID: "host.drag_drop.drop", ActionTraceID: "trace.shell.drag.enter"},
			{ID: "drag.drop", Kind: "drop", SourceBlock: "InputBlock", TargetBlock: "PanelBlock", MIME: "text/plain", Delivered: true, HostReportID: "host.drag_drop.drop", ActionTraceID: "trace.shell.drag.drop"},
		},
		Permissions: []AppShellPermissionReport{
			{ID: "permission.notification", Kind: "notification", Mode: "ask", Granted: true},
			{ID: "permission.filesystem", Kind: "filesystem", Mode: "scoped-open-file", Rejected: true, DiagnosticID: "diag.filesystem.denied"},
			{ID: "permission.clipboard", Kind: "clipboard", Mode: "focused-window", Granted: true},
		},
		OpenRequests: []AppShellOpenRequestReport{
			{ID: "open.docs", Kind: "open_url", Target: "https://tetra.local/surface", Delivered: true, HostReportID: "host.open.url"},
			{ID: "open.project", Kind: "open_file", Target: "examples/surface_block_system.tetra", Delivered: true, HostReportID: "host.open.file"},
		},
		DPI: []AppShellDPIReport{
			{ID: "dpi.primary", ScaleMille: 1000, Width: 400, Height: 240, HostReportID: "host.dpi.scale"},
			{ID: "dpi.hidpi", ScaleMille: 2000, Width: 800, Height: 480, HostReportID: "host.dpi.scale"},
		},
		Diagnostics: []AppShellDiagnosticReport{
			{ID: "diag.platform_widget_shell", Capability: "platform_widget_shell", Code: "platform_widget_shell_rejected", Message: "Surface app shell must not delegate UI to platform widgets", Unsupported: true, Rejected: true, Target: target},
			{ID: "diag.native_global_menu", Capability: "native_global_menu", Code: "native_global_menu_nonclaim", Message: "Native global menu parity is not claimed without target-host proof", Unsupported: true, Rejected: true, Target: target},
			{ID: "diag.filesystem.denied", Capability: "filesystem", Code: "permission_denied", Message: "Filesystem access outside picker grant rejected", Unsupported: true, Rejected: true, Target: target},
		},
		NegativeGuards: AppShellNegativeGuardsReport{
			MissingMenuHostTraceRejected:          true,
			NotificationWithoutHostReportRejected: true,
			UnsupportedFeatureSilentNoopRejected:  true,
			WindowWithoutLifecycleRejected:        true,
			PermissionWithoutDiagnosticRejected:   true,
			PlatformWidgetShellRejected:           true,
		},
		NonClaims: []string{"Electron shell dependency", "React runtime", "DOM-rendered interface", "platform-native widgets", "silent no-op unsupported host features"},
	}
}

func retargetAppShellForTest(shell *SurfaceAppShellReport, target string) {
	shell.Target = target
	for i := range shell.HostReports {
		shell.HostReports[i].Target = target
	}
	for i := range shell.Diagnostics {
		shell.Diagnostics[i].Target = target
	}
}

func blockSystemComponentsForTest() []ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}
	return []ComponentReport{
		{ID: "BlockSystemApp", Type: "examples.surface_block_system.BlockSystemApp", Bounds: RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"focused_id": "4", "quality": "deterministic-headless-block-system-v1"}},
		{ID: "PanelBlock", Type: "examples.surface_block_system.PanelBlock", Parent: "BlockSystemApp", Bounds: RectReport{X: 16, Y: 16, W: 288, H: 168}, Abilities: abilities, State: map[string]string{"paint_layers": "5"}},
		{ID: "LabelBlock", Type: "examples.surface_block_system.LabelBlock", Parent: "PanelBlock", Bounds: RectReport{X: 24, Y: 24, W: 200, H: 24}, Abilities: abilities, State: map[string]string{"text_len": "4", "label_for": "4"}},
		{ID: "SubmitBlock", Type: "examples.surface_block_system.ActionBlock", Parent: "PanelBlock", Bounds: RectReport{X: 24, Y: 64, W: 120, H: 44}, Abilities: abilities, State: map[string]string{"focused": "true", "action": "submit"}},
		{ID: "ResetBlock", Type: "examples.surface_block_system.ActionBlock", Parent: "PanelBlock", Bounds: RectReport{X: 152, Y: 64, W: 120, H: 44}, Abilities: abilities, State: map[string]string{"focused": "false", "action": "reset"}},
		{ID: "BlockLayoutApp", Type: "examples.surface_block_system.BlockLayoutApp", Bounds: RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"width": "480", "layout_quality": "deterministic-block-layout-v1"}},
		{ID: "ScrollBlock", Type: "examples.surface_block_system.ScrollBlock", Parent: "BlockLayoutApp", Bounds: RectReport{X: 236, Y: 72, W: 72, H: 80}, Abilities: abilities, State: map[string]string{"scroll_y": "32"}},
	}
}

func blockSystemEventsForTest() []EventReport {
	return []EventReport{
		{Order: 1, Kind: "mouse_up", TargetComponent: "SubmitBlock", DispatchPath: []string{"BlockSystemApp", "PanelBlock", "SubmitBlock"}, Handled: true, Pass: true, X: 40, Y: 80, Width: 320, Height: 200, BufferSlots: []int{5, 40, 80, 1, 0, 320, 200, 0, 0}, BeforeState: map[string]string{"SubmitBlock.focused": "false"}, AfterState: map[string]string{"SubmitBlock.focused": "true"}},
		{Order: 2, Kind: "text_input", TargetComponent: "SubmitBlock", DispatchPath: []string{"BlockSystemApp", "PanelBlock", "SubmitBlock"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 1, TextLen: 2, TextBytesHex: "4f4b", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 1, 2}, BeforeState: map[string]string{"SubmitBlock.value_len": "0"}, AfterState: map[string]string{"SubmitBlock.value_len": "2"}},
		{Order: 3, Kind: "key_down", TargetComponent: "SubmitBlock", DispatchPath: []string{"BlockSystemApp", "PanelBlock", "SubmitBlock"}, Handled: true, Pass: true, Key: 13, Width: 320, Height: 200, TimestampMS: 2, BufferSlots: []int{3, 0, 0, 0, 13, 320, 200, 2, 0}, BeforeState: map[string]string{"SubmitBlock.pressed": "false"}, AfterState: map[string]string{"SubmitBlock.pressed": "true"}},
		{Order: 4, Kind: "scroll", TargetComponent: "ScrollBlock", DispatchPath: []string{"BlockLayoutApp", "ScrollBlock"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 3, BufferSlots: []int{7, 0, 0, 0, 0, 320, 200, 3, 0}, BeforeState: map[string]string{"ScrollBlock.scroll_y": "0"}, AfterState: map[string]string{"ScrollBlock.scroll_y": "32"}},
	}
}

func retargetBlockSystemComponentsForTest(components []ComponentReport) []ComponentReport {
	retargeted := make([]ComponentReport, len(components))
	for i, component := range components {
		component.Type = "examples.surface_block_system." + typeBaseName(component.Type)
		retargeted[i] = component
	}
	return retargeted
}

func typeBaseName(value string) string {
	index := strings.LastIndex(value, ".")
	if index < 0 {
		return value
	}
	return value[index+1:]
}

func appendEventReportsWithNextOrder(events []EventReport, additions ...[]EventReport) []EventReport {
	nextOrder := 0
	if len(events) > 0 {
		nextOrder = events[len(events)-1].Order
	}
	for _, group := range additions {
		for _, event := range group {
			nextOrder++
			event.Order = nextOrder
			events = append(events, event)
		}
	}
	return events
}

func appendStateTransitionReportsWithNextOrder(transitions []StateTransitionReport, additions ...[]StateTransitionReport) []StateTransitionReport {
	nextOrder := 0
	if len(transitions) > 0 {
		nextOrder = transitions[len(transitions)-1].Order
	}
	for _, group := range additions {
		for _, transition := range group {
			nextOrder++
			transition.Order = nextOrder
			transitions = append(transitions, transition)
		}
	}
	return transitions
}

func blockSystemReadinessTransitionsForTest() []StateTransitionReport {
	return []StateTransitionReport{
		{Order: 1, Component: "InputBlock", Field: "buffer", Before: "", After: "OKd0a2", Cause: "text_input"},
		{Order: 2, Component: "InputBlock", Field: "caret", Before: "0", After: "4", Cause: "text_input"},
		{Order: 3, Component: "StateBlock", Field: "selector_flags", Before: "0", After: "127", Cause: "pointer/key/state input"},
		{Order: 4, Component: "StateBlock", Field: "resolved_fill", Before: "#20262eff", After: "#2d9bf0ff", Cause: "hover"},
		{Order: 5, Component: "StateBlock", Field: "resolved_scale", Before: "100", After: "97", Cause: "pressed"},
		{Order: 6, Component: "StateBlock", Field: "disabled", Before: "false", After: "true", Cause: "disabled selector"},
		{Order: 7, Component: "StateBlock", Field: "error", Before: "false", After: "true", Cause: "error selector"},
		{Order: 8, Component: "StateBlock", Field: "loading", Before: "false", After: "true", Cause: "loading selector"},
		{Order: 9, Component: "MotionBlock", Field: "opacity", Before: "80", After: "200", Cause: "motion frame"},
		{Order: 10, Component: "MotionBlock", Field: "color", Before: "#203040ff", After: "#60aef4ff", Cause: "motion frame"},
		{Order: 11, Component: "MotionBlock", Field: "scale", Before: "100", After: "108", Cause: "motion frame"},
		{Order: 12, Component: "MotionBlock", Field: "translate_x", Before: "0", After: "12", Cause: "motion frame"},
		{Order: 13, Component: "MotionBlock", Field: "motion_complete", Before: "false", After: "true", Cause: "duration elapsed"},
		{Order: 14, Component: "MotionBlock", Field: "reduced_motion", Before: "false", After: "true", Cause: "accessibility setting"},
		{Order: 15, Component: "IconBlock", Field: "tint", Before: "#ffffffff", After: "#60aef4ff", Cause: "asset tint"},
		{Order: 16, Component: "ImageBlock", Field: "scale", Before: "1x", After: "2x", Cause: "asset scale"},
		{Order: 17, Component: "MissingAssetBlock", Field: "fallback", Before: "missing", After: "fallback-raster", Cause: "missing asset"},
	}
}

func blockSystemCasesForTest() []CaseReport {
	return []CaseReport{
		{Name: "block graph duplicate id rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "duplicate Block ID"},
		{Name: "block graph missing parent rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "missing parent"},
		{Name: "block graph cycle rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "cycle"},
		{Name: "block graph child order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block graph focus order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block graph hit-test path", Kind: "positive", Ran: true, Pass: true},
		{Name: "block graph accessibility order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block paint fill border radius shadow outline", Kind: "positive", Ran: true, Pass: true},
		{Name: "block paint deterministic command order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block paint frame checksum changed", Kind: "positive", Ran: true, Pass: true},
		{Name: "block paint unsupported blur rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "unsupported blur"},
		{Name: "software renderer deterministic raster", Kind: "positive", Ran: true, Pass: true},
		{Name: "software renderer alpha clipping", Kind: "positive", Ran: true, Pass: true},
		{Name: "software renderer golden export", Kind: "positive", Ran: true, Pass: true},
		{Name: "software renderer resize scale dpi", Kind: "positive", Ran: true, Pass: true},
		{Name: "software renderer use-after-present rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "use-after-present"},
		{Name: "software renderer frame alias rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "frame alias"},
		{Name: "software renderer browser promotion without runtime rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "browser promotion without runtime"},
		{Name: "block text deterministic measurement", Kind: "positive", Ran: true, Pass: true},
		{Name: "block text wrap ellipsis layout", Kind: "positive", Ran: true, Pass: true},
		{Name: "block text font fallback chain", Kind: "positive", Ran: true, Pass: true},
		{Name: "block text bounded glyph cache", Kind: "positive", Ran: true, Pass: true},
		{Name: "block text render command evidence", Kind: "positive", Ran: true, Pass: true},
		{Name: "block text editable lifetime", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout nested row column", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout fit fill fixed min max", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout grid dock overlay scroll", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout clipping z-order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout resize constraints", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout no css flexbox parity", Kind: "negative", Ran: true, Pass: true, ExpectedError: "CSS flexbox parity nonclaim"},
		{Name: "block layout DPI density", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout invalidation cache budget", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout responsive app shell settings dashboard editor", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout overflow explicit clip scroll", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout no css grid parity", Kind: "negative", Ran: true, Pass: true, ExpectedError: "CSS grid parity nonclaim"},
		{Name: "block state selector resolver order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state hover fill override", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state pressed scale override", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state focus selected metadata", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state disabled error loading overrides", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state frame checksum changed", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state no css pseudo parity", Kind: "negative", Ran: true, Pass: true, ExpectedError: "css pseudo nonclaim"},
		{Name: "app model owned state stores", Kind: "positive", Ran: true, Pass: true},
		{Name: "app model typed commands", Kind: "positive", Ran: true, Pass: true},
		{Name: "app model Block event trace", Kind: "positive", Ran: true, Pass: true},
		{Name: "app model async safe boundary", Kind: "positive", Ran: true, Pass: true},
		{Name: "app model navigation focus scopes", Kind: "positive", Ran: true, Pass: true},
		{Name: "app model shortcut scopes", Kind: "positive", Ran: true, Pass: true},
		{Name: "app model error propagation", Kind: "positive", Ran: true, Pass: true},
		{Name: "app model redraw scheduling", Kind: "positive", Ran: true, Pass: true},
		{Name: "app model disabled dispatch rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "disabled dispatch"},
		{Name: "app model unfocused text rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "unfocused text"},
		{Name: "app model no React runtime", Kind: "negative", Ran: true, Pass: true, ExpectedError: "React runtime rejected"},
		{Name: "keyboard ux focus order", Kind: "positive", Ran: true, Pass: true},
		{Name: "keyboard ux focus trap", Kind: "positive", Ran: true, Pass: true},
		{Name: "keyboard ux roving focus", Kind: "positive", Ran: true, Pass: true},
		{Name: "keyboard ux activation", Kind: "positive", Ran: true, Pass: true},
		{Name: "keyboard ux shortcut conflict rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "shortcut conflict"},
		{Name: "keyboard ux undo redo", Kind: "positive", Ran: true, Pass: true},
		{Name: "keyboard ux command palette script", Kind: "positive", Ran: true, Pass: true},
		{Name: "keyboard ux settings script", Kind: "positive", Ran: true, Pass: true},
		{Name: "keyboard ux editor script", Kind: "positive", Ran: true, Pass: true},
		{Name: "keyboard ux accessible names", Kind: "negative", Ran: true, Pass: true, ExpectedError: "accessible name required"},
		{Name: "block motion deterministic test clock", Kind: "positive", Ran: true, Pass: true},
		{Name: "block motion opacity color transform frames", Kind: "positive", Ran: true, Pass: true},
		{Name: "block motion reduced motion instant settle", Kind: "positive", Ran: true, Pass: true},
		{Name: "block motion completion stops scheduling", Kind: "positive", Ran: true, Pass: true},
		{Name: "block motion frame checksum changed", Kind: "positive", Ran: true, Pass: true},
		{Name: "block motion no css animation parity", Kind: "negative", Ran: true, Pass: true, ExpectedError: "css animation nonclaim"},
		{Name: "block asset deterministic manifest hashes", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset local embedded only", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset bounded cache", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset icon tint evidence", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset image scale evidence", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset vector safe decode", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset missing fallback diagnostic", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset network url rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "network asset rejected"},
		{Name: "block asset unsafe SVG rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "unsafe svg"},
		{Name: "block asset remote font rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "remote font"},
		{Name: "block accessibility tree derived from block graph", Kind: "positive", Ran: true, Pass: true},
		{Name: "block accessibility focusable actionable name required", Kind: "negative", Ran: true, Pass: true, ExpectedError: "missing accessible name"},
		{Name: "block accessibility label relationship mismatch rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "label relationship mismatch"},
		{Name: "block accessibility reading order graph mismatch rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "reading order mismatch"},
		{Name: "block accessibility screen-reader claim without platform proof rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "screen reader proof required"},
		{Name: "block accessibility platform claim scoped metadata only", Kind: "positive", Ran: true, Pass: true},
		{Name: "block system headless golden checksums", Kind: "positive", Ran: true, Pass: true},
		{Name: "block system deterministic repeat checksum", Kind: "positive", Ran: true, Pass: true},
		{Name: "block system missing frame checksum rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "frame checksum required"},
		{Name: "block system nondeterministic checksum rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "repeat checksum mismatch"},
		{Name: "block system missing paint evidence rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "paint evidence required"},
		{Name: "block system missing layout evidence rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "layout evidence required"},
		{Name: "block system missing accessibility evidence rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "accessibility evidence required"},
		{Name: "block system bounded memory budget", Kind: "positive", Ran: true, Pass: true},
		{Name: "block system stress render loop budget", Kind: "positive", Ran: true, Pass: true},
		{Name: "block system performance nonclaim", Kind: "negative", Ran: true, Pass: true, ExpectedError: "Electron comparison benchmark not claimed"},
	}
}

func appShellCasesForTest() []CaseReport {
	return []CaseReport{
		{Name: "app shell window lifecycle", Kind: "positive", Ran: true, Pass: true},
		{Name: "app shell menu action trace", Kind: "positive", Ran: true, Pass: true},
		{Name: "app shell dialog picker trace", Kind: "positive", Ran: true, Pass: true},
		{Name: "app shell notification host report", Kind: "positive", Ran: true, Pass: true},
		{Name: "app shell tray cursor drag drop", Kind: "positive", Ran: true, Pass: true},
		{Name: "app shell permissions diagnostics", Kind: "positive", Ran: true, Pass: true},
		{Name: "app shell unsupported feature rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "unsupported host feature rejected"},
		{Name: "app shell menu claim without host trace rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "menu target-host trace required"},
		{Name: "app shell notification claim without host report rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "notification host report required"},
	}
}

func blockSystemLinuxX64RealWindowCasesForTest() []CaseReport {
	cases := []CaseReport{
		{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
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
	}
	for _, tc := range blockSystemCasesForTest() {
		name := strings.ToLower(tc.Name)
		if strings.Contains(name, "headless") {
			continue
		}
		if strings.Contains(name, "deterministic repeat checksum") {
			continue
		}
		cases = append(cases, tc)
	}
	cases = append(cases, appShellCasesForTest()...)
	cases = append(cases,
		CaseReport{Name: "linux-x64 real-window surface", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "linux-x64 native input event pump", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "linux-x64 real-window resize event", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "linux-x64 real-window close event", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block system linux-x64 real-window frame presentation", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block system linux-x64 native input state transition", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block system linux-x64 real-window checksum", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block system missing real-window presentation rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "real-window presentation required"},
		CaseReport{Name: "block system missing native input state transition rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "native input required"},
	)
	return cases
}

func blockSystemWASM32WebBrowserCanvasCasesForTest() []CaseReport {
	cases := []CaseReport{
		{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
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
	}
	for _, tc := range blockSystemCasesForTest() {
		name := strings.ToLower(tc.Name)
		if strings.Contains(name, "headless") {
			continue
		}
		if strings.Contains(name, "deterministic repeat checksum") {
			continue
		}
		cases = append(cases, tc)
	}
	cases = append(cases, appShellCasesForTest()...)
	cases = append(cases,
		CaseReport{Name: "wasm32-web browser canvas surface", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "wasm32-web browser canvas RGBA readback", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "wasm32-web browser canvas pointer input", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "wasm32-web browser canvas keyboard input", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "wasm32-web browser canvas resize input", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "wasm32-web browser canvas text input", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "wasm32-web Surface Host ABI imports", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "compiler-owned wasm Surface loader", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "compiler-owned browser canvas Surface host", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block system wasm32-web browser-canvas frame readback", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block system wasm32-web browser-canvas native input state transition", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block system wasm32-web browser-canvas checksum", Kind: "positive", Ran: true, Pass: true},
		CaseReport{Name: "block system browser-canvas node runtime substitution rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "browser evidence required"},
		CaseReport{Name: "block system browser-canvas missing RGBA readback rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "RGBA readback required"},
		CaseReport{Name: "block system browser-canvas script sidecar artifact rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "script artifact rejected"},
		CaseReport{Name: "block system browser-canvas html visual sidecar artifact rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "html artifact rejected"},
	)
	return cases
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

func blockTextComponentsForTest() []ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}
	return []ComponentReport{
		{ID: "BlockTextApp", Type: "examples.surface_block_text.BlockTextApp", Bounds: RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"focused_id": "3", "text_quality": "deterministic-fallback-text-v1"}},
		{ID: "TextBlock", Type: "examples.surface_block_text.TextSurfaceBlock", Parent: "BlockTextApp", Bounds: RectReport{X: 12, Y: 10, W: 96, H: 40}, Abilities: abilities, State: map[string]string{"text_len": "28", "line_count": "2", "ellipsis": "true"}},
		{ID: "InputBlock", Type: "examples.surface_block_text.EditableTextBlock", Parent: "BlockTextApp", Bounds: RectReport{X: 12, Y: 58, W: 144, H: 36}, Abilities: abilities, State: map[string]string{"buffer": "OKd0a2", "caret": "4", "editable": "true"}},
	}
}

func blockEventComponentsForTest() []ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}
	return []ComponentReport{
		{ID: "BlockEventApp", Type: "examples.surface_block_events.BlockEventApp", Bounds: RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"focused_id": "4", "event_quality": "deterministic-block-events-v1"}},
		{ID: "PanelBlock", Type: "examples.surface_block_events.PanelBlock", Parent: "BlockEventApp", Bounds: RectReport{X: 16, Y: 16, W: 288, H: 168}, Abilities: abilities, State: map[string]string{"role": "panel"}},
		{ID: "LabelBlock", Type: "examples.surface_block_events.LabelBlock", Parent: "PanelBlock", Bounds: RectReport{X: 24, Y: 24, W: 200, H: 24}, Abilities: abilities, State: map[string]string{"text_len": "10"}},
		{ID: "InputBlock", Type: "examples.surface_block_events.InputBlock", Parent: "PanelBlock", Bounds: RectReport{X: 24, Y: 64, W: 120, H: 44}, Abilities: abilities, State: map[string]string{"editable": "true", "focused": "true", "buffer": "OK"}},
		{ID: "DisabledBlock", Type: "examples.surface_block_events.DisabledBlock", Parent: "PanelBlock", Bounds: RectReport{X: 152, Y: 64, W: 120, H: 44}, Abilities: abilities, State: map[string]string{"disabled": "true"}},
		{ID: "ActionBlock", Type: "examples.surface_block_events.ActionBlock", Parent: "PanelBlock", Bounds: RectReport{X: 24, Y: 120, W: 120, H: 44}, Abilities: abilities, State: map[string]string{"focused": "false"}},
	}
}

func blockStateComponentsForTest() []ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility", "state"}
	return []ComponentReport{
		{ID: "BlockStateApp", Type: "examples.surface_block_states.BlockStateApp", Bounds: RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"state_quality": "deterministic-block-state-resolver-v1"}},
		{ID: "StateBlock", Type: "examples.surface_block_states.StateBlock", Parent: "BlockStateApp", Bounds: RectReport{X: 24, Y: 40, W: 168, H: 56}, Abilities: abilities, State: map[string]string{"selector_flags": "127", "variant": "2", "disabled": "true", "error": "true", "loading": "true"}},
		{ID: "StatusBlock", Type: "examples.surface_block_states.StatusBlock", Parent: "BlockStateApp", Bounds: RectReport{X: 24, Y: 112, W: 168, H: 32}, Abilities: abilities, State: map[string]string{"selected": "true", "focused": "true"}},
	}
}

func blockMotionComponentsForTest() []ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility", "state", "motion"}
	return []ComponentReport{
		{ID: "BlockMotionApp", Type: "examples.surface_block_motion.BlockMotionApp", Bounds: RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"motion_quality": "deterministic-block-motion-v1"}},
		{ID: "MotionBlock", Type: "examples.surface_block_motion.MotionBlock", Parent: "BlockMotionApp", Bounds: RectReport{X: 24, Y: 44, W: 176, H: 64}, Abilities: abilities, State: map[string]string{"opacity": "200", "scale": "108", "translate_x": "12", "complete": "true"}},
	}
}

func blockAssetComponentsForTest() []ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility", "asset"}
	return []ComponentReport{
		{ID: "BlockAssetApp", Type: "examples.surface_block_assets.BlockAssetApp", Bounds: RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"asset_quality": "deterministic-local-block-assets-v1"}},
		{ID: "IconBlock", Type: "examples.surface_block_assets.IconBlock", Parent: "BlockAssetApp", Bounds: RectReport{X: 24, Y: 36, W: 32, H: 32}, Abilities: abilities, State: map[string]string{"asset_id": "icon-settings", "tint": "#60aef4ff"}},
		{ID: "ImageBlock", Type: "examples.surface_block_assets.ImageBlock", Parent: "BlockAssetApp", Bounds: RectReport{X: 72, Y: 32, W: 96, H: 64}, Abilities: abilities, State: map[string]string{"asset_id": "image-hero", "scale": "2x"}},
		{ID: "VectorBlock", Type: "examples.surface_block_assets.VectorBlock", Parent: "BlockAssetApp", Bounds: RectReport{X: 184, Y: 40, W: 40, H: 32}, Abilities: abilities, State: map[string]string{"asset_id": "vector-logo", "decoder": "svg-tiny-static-sanitized-v1"}},
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
		VectorCount:   1,
		EmbeddedCount: 4,
		RemoteCount:   0,
		Assets: []BlockAssetReport{
			{ID: "font-ui", Kind: "font", Path: "embedded://surface/font-ui", Embedded: true, Local: true, SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 2048, Family: "Tetra UI", CacheKey: "font-ui"},
			{ID: "icon-settings", Kind: "icon", Path: "embedded://surface/icon-settings", Embedded: true, Local: true, SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Size: 256, Width: 16, Height: 16, CacheKey: "icon-settings"},
			{ID: "image-hero", Kind: "image", Path: "embedded://surface/image-hero", Embedded: true, Local: true, SHA256: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", Size: 1024, Width: 48, Height: 32, CacheKey: "image-hero"},
			{ID: "vector-logo", Kind: "vector", Path: "embedded://surface/vector-logo.svg", Embedded: true, Local: true, SHA256: "sha256:1212121212121212121212121212121212121212121212121212121212121212", Size: 512, Width: 24, Height: 24, CacheKey: "vector-logo"},
		},
	}
}

func blockAssetCacheForTest() BlockAssetCacheReport {
	return BlockAssetCacheReport{ID: "asset-cache", Strategy: "bounded-lru", BudgetBytes: 65536, UsedBytes: 5888, EntryCount: 4, MaxEntries: 16, RepeatedLoads: 6, Eviction: "lru", Bounded: true}
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
		{Order: 4, Command: "render_vector", AssetID: "vector-logo", BlockID: 4, Rect: RectReport{X: 184, Y: 40, W: 40, H: 32}, Quality: "svg-tiny-static-sanitized-v1", Checksum: "sha256:7878787878787878787878787878787878787878787878787878787878787878"},
		{Order: 5, Command: "fallback_missing", AssetID: "missing-logo", BlockID: 5, Rect: RectReport{X: 24, Y: 112, W: 96, H: 32}, Quality: "fallback-raster-v1", Checksum: "sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"},
	}
}

func assetPipelineForTest(source string, manifest *BlockAssetManifestReport, cache BlockAssetCacheReport, diagnostics []BlockAssetDiagnosticReport, commands []BlockAssetRenderCommandReport) *SurfaceAssetPipelineReport {
	return &SurfaceAssetPipelineReport{
		Schema:                    AssetPipelineSchemaV1,
		Level:                     "production-asset-pipeline-v1",
		Source:                    source,
		ReleaseScope:              ReleaseScopeSurfaceV1LinuxWeb,
		ManifestSchema:            manifest.Schema,
		ManifestHash:              manifest.ManifestHash,
		HashAlgorithm:             manifest.HashAlgorithm,
		LocalOnly:                 manifest.LocalOnly,
		NetworkFetchAllowed:       false,
		FontCount:                 manifest.FontCount,
		IconCount:                 manifest.IconCount,
		ImageCount:                manifest.ImageCount,
		VectorCount:               manifest.VectorCount,
		AssetCount:                len(manifest.Assets),
		DecoderPolicy:             "safe-local-asset-decoders-v1",
		FontDecoder:               "font-table-hash-verified-v1",
		IconDecoder:               "icon-mask-tint-rgba-v1",
		ImageDecoder:              "png-rgba-bounds-checked-v1",
		VectorDecoder:             "svg-tiny-static-sanitized-v1",
		FontHashesVerified:        true,
		IconTintPipeline:          true,
		ImageBoundsChecked:        true,
		RasterDecodeBoundsChecked: true,
		VectorSanitized:           true,
		SVGScriptRejected:         true,
		SVGExternalRefRejected:    true,
		CacheStrategy:             cache.Strategy,
		CacheBudgetBytes:          cache.BudgetBytes,
		CacheUsedBytes:            cache.UsedBytes,
		CacheEntryCount:           cache.EntryCount,
		CacheBounded:              cache.Bounded,
		RenderCommandCount:        len(commands),
		DiagnosticCount:           len(diagnostics),
		NegativeGuards: AssetPipelineNegativeGuardsReport{
			MissingHashRejected:          true,
			RemoteFontRejected:           true,
			NetworkFetchRejected:         true,
			UnboundedCacheRejected:       true,
			MissingAssetFallbackRequired: true,
			UnsafeSVGRejected:            true,
			OversizedRasterRejected:      true,
			DecoderWithoutHashRejected:   true,
		},
		NonClaims: []string{
			"network assets",
			"remote fonts",
			"untrusted svg scripting",
			"full SVG/CSS/SMIL",
			"arbitrary image codecs",
		},
	}
}

func blockMemoryBudgetForTest(report *Report) *BlockMemoryBudgetReport {
	peakFramebufferBytes, totalFramebufferBytes := blockFramebufferByteTotals(report.Frames)
	cacheUsedBytes := len(report.PaintCommands)*2048 + 4096 + report.BlockAssetCache.UsedBytes
	return &BlockMemoryBudgetReport{
		Schema:                   "tetra.surface.block-memory-budget.v1",
		Scope:                    "surface-block-system-local-budget-v1",
		BlockCount:               len(report.Components),
		StressBlockCount:         128,
		RenderLoopCount:          32,
		StateLoopCount:           len(report.StateTransitions),
		MotionFrameCount:         len(report.MotionFrames),
		InputEventCount:          len(report.Events),
		PaintCommandCount:        len(report.PaintCommands),
		TextRenderCommandCount:   len(report.TextRenderCommands),
		AssetRenderCommandCount:  len(report.BlockAssetRenderCommands),
		PeakFramebufferBytes:     peakFramebufferBytes,
		TotalFramebufferBytes:    totalFramebufferBytes,
		FramebufferBudgetBytes:   1048576,
		PaintCacheUsedBytes:      len(report.PaintCommands) * 2048,
		PaintCacheBudgetBytes:    report.PaintCacheBudgetBytes,
		TextCacheUsedBytes:       4096,
		TextCacheBudgetBytes:     report.TextCacheBudgetBytes,
		AssetCacheUsedBytes:      report.BlockAssetCache.UsedBytes,
		AssetCacheBudgetBytes:    report.BlockAssetCache.BudgetBytes,
		TotalCacheUsedBytes:      cacheUsedBytes,
		TotalCacheBudgetBytes:    report.PaintCacheBudgetBytes + report.TextCacheBudgetBytes + report.BlockAssetCache.BudgetBytes,
		EstimatedAllocationBytes: totalFramebufferBytes + cacheUsedBytes,
		RSSMeasured:              false,
		PeakRSSBytes:             0,
		BoundedCaches:            true,
		UnboundedCacheRejected:   true,
		StressScene:              "deterministic-block-stress-128",
		PerformanceClaim:         "none",
		NonClaims: []string{
			"no Electron comparison benchmark",
			"no broad performance superiority claim",
			"RSS is optional host evidence and not required for this local budget",
		},
	}
}

func blockAssetEventsForTest() []EventReport {
	return []EventReport{
		{Order: 1, Kind: "mouse_up", TargetComponent: "IconBlock", DispatchPath: []string{"BlockAssetApp", "IconBlock"}, Handled: true, Pass: true, X: 32, Y: 44, Width: 320, Height: 200, TimestampMS: 0, BufferSlots: []int{5, 32, 44, 1, 0, 320, 200, 0, 0}, BeforeState: map[string]string{"IconBlock.tint": "#ffffffff"}, AfterState: map[string]string{"IconBlock.tint": "#60aef4ff"}},
		{Order: 2, Kind: "text_input", TargetComponent: "IconBlock", DispatchPath: []string{"BlockAssetApp", "IconBlock"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 1, TextLen: 2, TextBytesHex: "4f4b", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 1, 2}, BeforeState: map[string]string{"IconBlock.label": ""}, AfterState: map[string]string{"IconBlock.label": "OK"}},
	}
}

func blockLayoutComponentsForTest() []ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}
	return []ComponentReport{
		{ID: "BlockLayoutApp", Type: "examples.surface_block_layout.BlockLayoutApp", Bounds: RectReport{X: 0, Y: 0, W: 320, H: 200}, Abilities: abilities, State: map[string]string{"layout_quality": "deterministic-block-layout-v1"}},
		{ID: "ColumnBlock", Type: "examples.surface_block_layout.ColumnBlock", Parent: "BlockLayoutApp", Bounds: RectReport{X: 12, Y: 12, W: 296, H: 176}, Abilities: abilities, State: map[string]string{"mode": "column", "gap": "8"}},
		{ID: "RowBlock", Type: "examples.surface_block_layout.RowBlock", Parent: "ColumnBlock", Bounds: RectReport{X: 24, Y: 24, W: 272, H: 48}, Abilities: abilities, State: map[string]string{"mode": "row", "gap": "6"}},
		{ID: "GridBlock", Type: "examples.surface_block_layout.GridBlock", Parent: "ColumnBlock", Bounds: RectReport{X: 24, Y: 80, W: 132, H: 72}, Abilities: abilities, State: map[string]string{"mode": "grid", "columns": "2"}},
		{ID: "DockBlock", Type: "examples.surface_block_layout.DockBlock", Parent: "ColumnBlock", Bounds: RectReport{X: 164, Y: 80, W: 132, H: 72}, Abilities: abilities, State: map[string]string{"mode": "dock"}},
		{ID: "OverlayBlock", Type: "examples.surface_block_layout.OverlayBlock", Parent: "BlockLayoutApp", Bounds: RectReport{X: 220, Y: 20, W: 72, H: 40}, Abilities: abilities, State: map[string]string{"mode": "overlay", "z": "4"}},
		{ID: "ScrollBlock", Type: "examples.surface_block_layout.ScrollBlock", Parent: "BlockLayoutApp", Bounds: RectReport{X: 236, Y: 72, W: 72, H: 80}, Abilities: abilities, State: map[string]string{"mode": "scroll", "clipped": "true"}},
	}
}

func blockMotionFramesForTest() []MotionFrameReport {
	return []MotionFrameReport{
		{Order: 1, BlockID: 2, Trigger: "hover", TimestampMS: 0, DurationMS: 120, DelayMS: 0, Progress: 0, Easing: "linear", Opacity: 80, Color: "#203040ff", TranslateX: 0, TranslateY: 0, Scale: 100, ReducedMotion: false, Scheduled: true, Settled: false, Checksum: "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		{Order: 2, BlockID: 2, Trigger: "hover", TimestampMS: 60, DurationMS: 120, DelayMS: 0, Progress: 500, Easing: "linear", Opacity: 140, Color: "#407094ff", TranslateX: 6, TranslateY: 0, Scale: 104, ReducedMotion: false, Scheduled: true, Settled: false, Checksum: "sha256:2222222222222222222222222222222222222222222222222222222222222222"},
		{Order: 3, BlockID: 2, Trigger: "hover", TimestampMS: 120, DurationMS: 120, DelayMS: 0, Progress: 1000, Easing: "linear", Opacity: 200, Color: "#60aef4ff", TranslateX: 12, TranslateY: 0, Scale: 108, ReducedMotion: false, Scheduled: false, Settled: true, Checksum: "sha256:3333333333333333333333333333333333333333333333333333333333333333"},
		{Order: 4, BlockID: 2, Trigger: "reduced_motion", TimestampMS: 121, DurationMS: 120, DelayMS: 0, Progress: 1000, Easing: "linear", Opacity: 200, Color: "#60aef4ff", TranslateX: 12, TranslateY: 0, Scale: 108, ReducedMotion: true, Scheduled: false, Settled: true, Checksum: "sha256:4444444444444444444444444444444444444444444444444444444444444444"},
	}
}

func animationSchedulerForTest(source string, frames []MotionFrameReport, budget int, target string, runtime string) *SurfaceAnimationSchedulerReport {
	scheduled, settled, reduced := motionFrameCounts(frames)
	return &SurfaceAnimationSchedulerReport{
		Schema:                     AnimationSchedulerSchemaV1,
		Level:                      "production-animation-scheduler-v1",
		Source:                     source,
		ReleaseScope:               ReleaseScopeSurfaceV1LinuxWeb,
		MotionQualityLevel:         "deterministic-block-motion-v1",
		MotionClock:                "deterministic-test-clock-v1",
		SchedulerPolicy:            "deterministic-motion-frame-scheduler-v1",
		TimelinePolicy:             "stable-motion-timeline-v1",
		InvalidationPolicy:         "motion-dirty-block-invalidation-v1",
		LifecyclePolicy:            "start-interpolate-settle-stop-v1",
		ReducedMotionPolicy:        "instant-settle-no-schedule-v1",
		FrameCount:                 len(frames),
		FrameBudget:                budget,
		ScheduledFrameCount:        scheduled,
		SettledFrameCount:          settled,
		ReducedMotionFrameCount:    reduced,
		TargetFrameIntervalMS:      16,
		MaxFrameDeltaMS:            maxMotionFrameDeltaMS(frames),
		JitterBudgetMS:             4,
		TransitionProperties:       []string{"opacity", "color", "transform", "translate", "scale"},
		DeterministicTimeline:      true,
		FrameTimingEvidence:        true,
		InvalidationEvidence:       true,
		LifecycleEvidence:          true,
		ReducedMotionEvidence:      true,
		VisualDeltaEvidence:        true,
		CSSAnimationParityRejected: true,
		HiddenLoopRejected:         true,
		TargetSmoke: []AnimationSchedulerTargetSmokeReport{
			{Target: target, Runtime: runtime, FrameCount: len(frames), FrameTimingEvidence: true, VisualDeltaEvidence: true, ArtifactHashEvidence: true, Pass: true},
		},
		NegativeGuards: AnimationSchedulerNegativeGuardsReport{
			CSSAnimationParityRejected: true,
			HiddenLoopRejected:         true,
			MissingReducedMotion:       true,
			MissingFrameTiming:         true,
			UnboundedFrameSchedule:     true,
			UnchangedVisualFrame:       true,
		},
		NonClaims: []string{
			"CSS animation runtime",
			"global animation cascade",
			"requestAnimationFrame parity",
			"GPU compositor timing",
			"unbounded hidden animation loops",
		},
	}
}

func blockMotionEventsForTest() []EventReport {
	return []EventReport{
		{Order: 1, Kind: "mouse_up", TargetComponent: "MotionBlock", DispatchPath: []string{"BlockMotionApp", "MotionBlock"}, Handled: true, Pass: true, X: 48, Y: 72, Width: 320, Height: 200, TimestampMS: 0, BufferSlots: []int{5, 48, 72, 1, 0, 320, 200, 0, 0}, BeforeState: map[string]string{"MotionBlock.hovered": "false"}, AfterState: map[string]string{"MotionBlock.hovered": "true"}},
		{Order: 2, Kind: "text_input", TargetComponent: "MotionBlock", DispatchPath: []string{"BlockMotionApp", "MotionBlock"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 1, TextLen: 2, TextBytesHex: "4f4b", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 1, 2}, BeforeState: map[string]string{"MotionBlock.buffer": ""}, AfterState: map[string]string{"MotionBlock.buffer": "OK"}},
	}
}

func blockStateSelectorsForTest() []BlockStateSelectorReport {
	return []BlockStateSelectorReport{
		{Order: 1, Name: "hover", BlockID: 2, Flags: 1, Hovered: true},
		{Order: 2, Name: "pressed", BlockID: 2, Flags: 2, Pressed: true},
		{Order: 3, Name: "focused", BlockID: 2, Flags: 4, Focused: true},
		{Order: 4, Name: "selected", BlockID: 2, Flags: 8, Selected: true},
		{Order: 5, Name: "disabled", BlockID: 2, Flags: 16, Disabled: true},
		{Order: 6, Name: "error", BlockID: 2, Flags: 32, Error: true},
		{Order: 7, Name: "loading", BlockID: 2, Flags: 64, Loading: true},
	}
}

func blockStateResolutionsForTest() []BlockStateResolutionReport {
	return []BlockStateResolutionReport{
		{Order: 1, BlockID: 2, Selector: "hover", ResolverStep: "hover", Property: "paint.fill", Before: "#20262eff", After: "#2d9bf0ff", Applied: true},
		{Order: 2, BlockID: 2, Selector: "pressed", ResolverStep: "pressed", Property: "layout.scale", Before: "100", After: "97", Applied: true},
		{Order: 3, BlockID: 2, Selector: "focused", ResolverStep: "focused", Property: "paint.outline", Before: "none", After: "focus-ring", Applied: true},
		{Order: 4, BlockID: 2, Selector: "selected", ResolverStep: "selected", Property: "accessibility.selected", Before: "false", After: "true", Applied: true},
		{Order: 5, BlockID: 2, Selector: "disabled", ResolverStep: "disabled", Property: "input.disabled", Before: "false", After: "true", Applied: true},
		{Order: 6, BlockID: 2, Selector: "disabled", ResolverStep: "disabled", Property: "text.opacity", Before: "255", After: "112", Applied: true},
		{Order: 7, BlockID: 2, Selector: "error", ResolverStep: "error", Property: "paint.outline_color", Before: "#7aa2f7ff", After: "#ff5f57ff", Applied: true},
		{Order: 8, BlockID: 2, Selector: "loading", ResolverStep: "loading", Property: "text.content", Before: "Run", After: "Loading", Applied: true},
		{Order: 9, BlockID: 2, Selector: "motion", ResolverStep: "motion", Property: "motion.transition_ms", Before: "0", After: "120", Applied: true},
	}
}

func blockStateEventsForTest() []EventReport {
	return []EventReport{
		{Order: 1, Kind: "mouse_up", TargetComponent: "StateBlock", DispatchPath: []string{"BlockStateApp", "StateBlock"}, Handled: true, Pass: true, X: 40, Y: 56, Width: 320, Height: 200, TimestampMS: 0, BufferSlots: []int{5, 40, 56, 1, 0, 320, 200, 0, 0}, BeforeState: map[string]string{"StateBlock.selected": "false"}, AfterState: map[string]string{"StateBlock.selected": "true"}},
		{Order: 2, Kind: "mouse_move", TargetComponent: "StateBlock", DispatchPath: []string{"BlockStateApp", "StateBlock"}, Handled: true, Pass: true, X: 40, Y: 56, Width: 320, Height: 200, TimestampMS: 1, BufferSlots: []int{2, 40, 56, 0, 0, 320, 200, 1, 0}, BeforeState: map[string]string{"StateBlock.hovered": "false"}, AfterState: map[string]string{"StateBlock.hovered": "true"}},
		{Order: 3, Kind: "mouse_down", TargetComponent: "StateBlock", DispatchPath: []string{"BlockStateApp", "StateBlock"}, Handled: true, Pass: true, X: 40, Y: 56, Width: 320, Height: 200, TimestampMS: 2, BufferSlots: []int{4, 40, 56, 1, 0, 320, 200, 2, 0}, BeforeState: map[string]string{"StateBlock.pressed": "false"}, AfterState: map[string]string{"StateBlock.pressed": "true"}},
		{Order: 4, Kind: "text_input", TargetComponent: "StateBlock", DispatchPath: []string{"BlockStateApp", "StateBlock"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 3, TextLen: 2, TextBytesHex: "4f4b", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 3, 2}, BeforeState: map[string]string{"StateBlock.buffer": ""}, AfterState: map[string]string{"StateBlock.buffer": "OK"}},
		{Order: 5, Kind: "key_down", TargetComponent: "StateBlock", DispatchPath: []string{"BlockStateApp", "StateBlock"}, Handled: true, Pass: true, Key: 9, Width: 320, Height: 200, TimestampMS: 4, BufferSlots: []int{3, 0, 0, 0, 9, 320, 200, 4, 0}, BeforeState: map[string]string{"StateBlock.focused": "false"}, AfterState: map[string]string{"StateBlock.focused": "true"}},
	}
}

func blockEventGraphReportForTest(source string) *BlockGraphReport {
	graph := &BlockGraphReport{
		Schema:            "tetra.surface.block-graph.v1",
		APILevel:          "block-tree-builder-v1",
		Source:            source,
		ManualBookkeeping: false,
		Builder: BlockGraphBuilderReport{
			RootCreatedBy:     "tree_add_root",
			ChildrenCreatedBy: "tree_add_child",
			NodeCount:         6,
			Capacity:          8,
			OverflowChecked:   true,
		},
		Invariants: BlockGraphInvariantReport{
			TreeValidateRan:         true,
			TreeValidateStatus:      0,
			DuplicateIDRejected:     true,
			MissingParentRejected:   true,
			CycleRejected:           true,
			ParentChildLinksChecked: true,
			ChildOrderChecked:       true,
			FocusOrderChecked:       true,
			HitTestPathChecked:      true,
			AccessibilityChecked:    true,
		},
		RootID:    1,
		NodeCount: 6,
		Nodes: []BlockGraphNodeReport{
			{ID: 1, Name: "BlockEventApp", ParentID: -1, ChildIndex: 0, FirstChild: 2, ChildCount: 1, Focusable: false, AccessibilityRole: "none", Bounds: RectReport{X: 0, Y: 0, W: 320, H: 200}},
			{ID: 2, Name: "PanelBlock", ParentID: 1, ChildIndex: 0, FirstChild: 3, ChildCount: 4, Focusable: false, AccessibilityRole: "none", Bounds: RectReport{X: 16, Y: 16, W: 288, H: 168}},
			{ID: 3, Name: "LabelBlock", ParentID: 2, ChildIndex: 0, FirstChild: -1, ChildCount: 0, Focusable: false, AccessibilityRole: "text", Bounds: RectReport{X: 24, Y: 24, W: 200, H: 24}},
			{ID: 4, Name: "InputBlock", ParentID: 2, ChildIndex: 1, FirstChild: -1, ChildCount: 0, Focusable: true, AccessibilityRole: "textbox", Bounds: RectReport{X: 24, Y: 64, W: 120, H: 44}},
			{ID: 5, Name: "DisabledBlock", ParentID: 2, ChildIndex: 2, FirstChild: -1, ChildCount: 0, Focusable: false, AccessibilityRole: "button", Bounds: RectReport{X: 152, Y: 64, W: 120, H: 44}},
			{ID: 6, Name: "ActionBlock", ParentID: 2, ChildIndex: 3, FirstChild: -1, ChildCount: 0, Focusable: true, AccessibilityRole: "button", Bounds: RectReport{X: 24, Y: 120, W: 120, H: 44}},
		},
		ChildOrders: []BlockGraphChildOrderReport{
			{ParentID: 1, Children: []int{2}},
			{ParentID: 2, Children: []int{3, 4, 5, 6}},
		},
		LayoutOrder:        []int{1, 2, 3, 4, 5, 6},
		DrawOrder:          []int{1, 2, 3, 4, 5, 6},
		FocusOrder:         []int{4, 6},
		AccessibilityOrder: []int{3, 4, 5, 6},
		HitTests: []BlockGraphPathReport{
			{Helper: "tree_hit_test_path", Event: "click", TargetID: 4, X: 40, Y: 80, Path: []int{1, 2, 4}},
			{Helper: "tree_hit_test_path", Event: "click", TargetID: 5, X: 180, Y: 80, Path: []int{1, 2, 5}},
		},
		DispatchPaths: []BlockGraphPathReport{
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 4, Path: []int{1, 2, 4}},
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 5, Path: []int{1, 2, 5}},
			{Helper: "tree_build_dispatch_path", Event: "key", TargetID: 6, Path: []int{1, 2, 6}},
		},
	}
	return withBlockGraphABIAndResolvedSceneForTest(graph)
}

func blockLayoutConstraintsForTest() []BlockLayoutConstraintReport {
	return []BlockLayoutConstraintReport{
		{ID: "root-column", BlockID: 1, Mode: "column", WidthPolicy: "fixed", HeightPolicy: "fixed", Min: SizeReport{W: 320, H: 200}, Max: SizeReport{W: 480, H: 260}, Padding: 12, Margin: 0, Gap: 8, Align: "stretch", Justify: "start", Overflow: "clip", ZIndex: 0, Clip: true},
		{ID: "row-fill", BlockID: 3, Mode: "row", WidthPolicy: "fill", HeightPolicy: "fixed", Min: SizeReport{W: 160, H: 40}, Max: SizeReport{W: 296, H: 64}, Padding: 6, Margin: 0, Gap: 6, Align: "center", Justify: "space-between", Overflow: "visible", ZIndex: 1, Clip: false},
		{ID: "text-fit", BlockID: 8, Mode: "absolute", WidthPolicy: "fit", HeightPolicy: "fit", Min: SizeReport{W: 32, H: 18}, Max: SizeReport{W: 160, H: 40}, Padding: 4, Margin: 0, Gap: 0, Align: "start", Justify: "start", Overflow: "clip", ZIndex: 2, Clip: true},
		{ID: "overlay-z", BlockID: 6, Mode: "overlay", WidthPolicy: "fixed", HeightPolicy: "fixed", Min: SizeReport{W: 72, H: 40}, Max: SizeReport{W: 72, H: 40}, Padding: 0, Margin: 0, Gap: 0, Align: "end", Justify: "start", Overflow: "visible", ZIndex: 4, Clip: false},
	}
}

func blockLayoutPassesForTest() []BlockLayoutPassReport {
	return []BlockLayoutPassReport{
		{Order: 1, ParentID: 0, BlockID: 1, Mode: "column", Input: RectReport{X: 0, Y: 0, W: 320, H: 200}, Resolved: RectReport{X: 12, Y: 12, W: 296, H: 176}, Measured: SizeReport{W: 296, H: 176}, Pass: "initial", Resize: false, Clip: true, ZIndex: 0, Checksum: "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		{Order: 2, ParentID: 1, BlockID: 2, Mode: "stack", Input: RectReport{X: 12, Y: 12, W: 296, H: 176}, Resolved: RectReport{X: 12, Y: 12, W: 296, H: 176}, Measured: SizeReport{W: 296, H: 176}, Pass: "initial", Resize: false, Clip: false, ZIndex: 0, Checksum: "sha256:2222222222222222222222222222222222222222222222222222222222222222"},
		{Order: 3, ParentID: 2, BlockID: 3, Mode: "row", Input: RectReport{X: 24, Y: 24, W: 272, H: 48}, Resolved: RectReport{X: 24, Y: 24, W: 272, H: 48}, Measured: SizeReport{W: 272, H: 48}, Pass: "nested", Resize: false, Clip: false, ZIndex: 1, Checksum: "sha256:3333333333333333333333333333333333333333333333333333333333333333"},
		{Order: 4, ParentID: 2, BlockID: 4, Mode: "grid", Input: RectReport{X: 24, Y: 80, W: 132, H: 72}, Resolved: RectReport{X: 24, Y: 80, W: 63, H: 34}, Measured: SizeReport{W: 63, H: 34}, Pass: "grid-cell", Resize: false, Clip: true, ZIndex: 1, Checksum: "sha256:4444444444444444444444444444444444444444444444444444444444444444"},
		{Order: 5, ParentID: 2, BlockID: 5, Mode: "dock", Input: RectReport{X: 164, Y: 80, W: 132, H: 72}, Resolved: RectReport{X: 164, Y: 80, W: 132, H: 24}, Measured: SizeReport{W: 132, H: 24}, Pass: "dock-top", Resize: false, Clip: true, ZIndex: 1, Checksum: "sha256:5555555555555555555555555555555555555555555555555555555555555555"},
		{Order: 6, ParentID: 1, BlockID: 6, Mode: "overlay", Input: RectReport{X: 220, Y: 20, W: 72, H: 40}, Resolved: RectReport{X: 220, Y: 20, W: 72, H: 40}, Measured: SizeReport{W: 72, H: 40}, Pass: "overlay-z-order", Resize: false, Clip: false, ZIndex: 4, Checksum: "sha256:6666666666666666666666666666666666666666666666666666666666666666"},
		{Order: 7, ParentID: 1, BlockID: 7, Mode: "scroll", Input: RectReport{X: 236, Y: 72, W: 72, H: 80}, Resolved: RectReport{X: 236, Y: 72, W: 72, H: 80}, Measured: SizeReport{W: 72, H: 160}, Pass: "scroll-clip", Resize: false, Clip: true, ZIndex: 2, Checksum: "sha256:7777777777777777777777777777777777777777777777777777777777777777"},
		{Order: 8, ParentID: 1, BlockID: 8, Mode: "absolute", Input: RectReport{X: 32, Y: 152, W: 0, H: 0}, Resolved: RectReport{X: 32, Y: 152, W: 96, H: 20}, Measured: SizeReport{W: 96, H: 20}, Pass: "fit-text", Resize: false, Clip: true, ZIndex: 2, Checksum: "sha256:8888888888888888888888888888888888888888888888888888888888888888"},
		{Order: 9, ParentID: 0, BlockID: 1, Mode: "column", Input: RectReport{X: 0, Y: 0, W: 480, H: 260}, Resolved: RectReport{X: 12, Y: 12, W: 456, H: 236}, Measured: SizeReport{W: 456, H: 236}, Pass: "resize", Resize: true, Clip: true, ZIndex: 0, Checksum: "sha256:9999999999999999999999999999999999999999999999999999999999999999"},
	}
}

func blockLayoutScrollsForTest() []BlockLayoutScrollReport {
	return []BlockLayoutScrollReport{
		{BlockID: 7, Viewport: RectReport{X: 236, Y: 72, W: 72, H: 80}, Content: SizeReport{W: 72, H: 160}, OffsetY: 32, MaxOffsetY: 80, Clipped: true, Checksum: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}
}

func blockLayoutEngineForTest(target string) *BlockLayoutEngineReport {
	return &BlockLayoutEngineReport{
		Schema:                LayoutEngineSchemaV1,
		Level:                 "production-layout-engine-v1",
		Producer:              "tools/cmd/surface-runtime-smoke",
		Target:                target,
		Quality:               "deterministic-responsive-layout-v1",
		CSSFlexboxGridParity:  false,
		PlatformWidgetLayout:  false,
		Modes:                 []string{"row", "column", "stack", "grid", "dock", "absolute", "overlay", "scroll"},
		ResponsiveProfiles:    []string{"app shell", "settings forms", "dashboards", "editor shells"},
		ConstraintFeatures:    []string{"min", "max", "fit", "fill", "fixed", "density", "overflow", "clip"},
		MinMaxConstraints:     true,
		FitFillFixed:          true,
		DensityIndependent:    true,
		StableUnderResize:     true,
		LayoutCacheKeyedByDPI: true,
		Density: BlockLayoutDensityReport{
			Schema:            "tetra.surface.layout-density.v1",
			Scale:             2,
			DPI:               192,
			DevicePixelRatio:  2,
			SnapToPixelGrid:   true,
			TargetIndependent: true,
			Cases:             []string{"headless scale 1", "linux-x64 scale 2", "wasm32-web devicePixelRatio 2"},
		},
		OverflowRules: BlockLayoutOverflowRulesReport{
			Explicit:                   true,
			ClipRequired:               true,
			ScrollBoundsChecked:        true,
			AccidentalHidden:           false,
			HiddenRequiresExplicitClip: true,
			VisibleOverflowPreserved:   true,
			ClippedBlockIDs:            []int{1, 4, 5, 7, 8},
		},
		Invalidations: []BlockLayoutInvalidationReport{
			{Cause: "resize", DirtyRoot: "BlockLayoutApp", AffectedModes: []string{"column", "row", "grid", "dock", "overlay", "scroll"}, BeforeChecksum: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", AfterChecksum: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", CacheEntriesReused: 2, CacheEntriesInvalidated: 6, FullTreeRelayout: true},
			{Cause: "scroll", DirtyRoot: "ScrollBlock", AffectedModes: []string{"scroll"}, BeforeChecksum: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", AfterChecksum: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", CacheEntriesReused: 6, CacheEntriesInvalidated: 1, FullTreeRelayout: false},
		},
		CacheBudget: BlockLayoutCacheBudgetReport{
			Schema:                 "tetra.surface.layout-cache.v1",
			Strategy:               "bounded-lru",
			BudgetBytes:            65536,
			UsedBytes:              9216,
			EntryCount:             9,
			MaxEntries:             64,
			Bounded:                true,
			Eviction:               "lru",
			UnboundedCacheRejected: true,
		},
		NegativeGuards: BlockLayoutEngineNegativeGuardsReport{
			CSSFlexboxGridParityRejected:     true,
			AccidentalOverflowHiddenRejected: true,
			UnboundedLayoutCacheRejected:     true,
			MissingDPIDensityRejected:        true,
			InvalidInvalidationRejected:      true,
		},
		NonClaims: []string{
			"CSS flexbox/grid parity",
			"browser CSS layout engine",
			"platform widget layout",
			"unbounded layout cache",
		},
	}
}

func blockLayoutEngineCasesForTest() []CaseReport {
	return []CaseReport{
		{Name: "block layout DPI density", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout invalidation cache budget", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout responsive app shell settings dashboard editor", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout overflow explicit clip scroll", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout no css grid parity", Kind: "negative", Ran: true, Pass: true, ExpectedError: "CSS grid parity nonclaim"},
	}
}

func blockEventRoutesForTest() []BlockEventRouteReport {
	return []BlockEventRouteReport{
		{Order: 1, Kind: "click", Policy: "capture-bubble-direct-v1", TargetID: 4, TargetName: "InputBlock", HitTestPath: []int{1, 2, 4}, DispatchPath: []int{1, 2, 4}, CapturePath: []int{1, 2}, BubblePath: []int{2, 1}, DirectTargetID: 4, Delivered: true, Rejected: false, FocusedID: 4, Editable: true, Disabled: false},
		{Order: 2, Kind: "click", Policy: "capture-bubble-direct-v1", TargetID: 5, TargetName: "DisabledBlock", HitTestPath: []int{1, 2, 5}, DispatchPath: []int{1, 2, 5}, CapturePath: []int{1, 2}, BubblePath: []int{2, 1}, DirectTargetID: 5, Delivered: false, Rejected: true, RejectReason: "disabled", FocusedID: 4, Editable: false, Disabled: true},
		{Order: 3, Kind: "text", Policy: "direct-to-focused-editable-v1", TargetID: 4, TargetName: "InputBlock", DispatchPath: []int{1, 2, 4}, DirectTargetID: 4, Delivered: false, Rejected: true, RejectReason: "unfocused", FocusedID: 6, Editable: true, TextLen: 2, TextBytesHex: "4f4b"},
		{Order: 4, Kind: "text", Policy: "direct-to-focused-editable-v1", TargetID: 4, TargetName: "InputBlock", DispatchPath: []int{1, 2, 4}, DirectTargetID: 4, Delivered: true, Rejected: false, FocusedID: 4, Editable: true, TextLen: 2, TextBytesHex: "4f4b"},
		{Order: 5, Kind: "key", Policy: "direct-to-focused-v1", TargetID: 6, TargetName: "ActionBlock", DispatchPath: []int{1, 2, 6}, DirectTargetID: 6, Delivered: true, Rejected: false, FocusedID: 6, Editable: false, Disabled: false},
	}
}

func blockFocusTransitionsForTest() []BlockFocusTransitionReport {
	return []BlockFocusTransitionReport{
		{Order: 1, Helper: "tree_focus_next", BeforeID: 4, AfterID: 6, Direction: "tab", GraphDerived: true, Wrapped: false},
		{Order: 2, Helper: "tree_focus_next", BeforeID: 6, AfterID: 4, Direction: "tab", GraphDerived: true, Wrapped: true},
	}
}

func blockTextEventsForTest() []EventReport {
	return []EventReport{
		{Order: 1, Kind: "mouse_up", TargetComponent: "InputBlock", DispatchPath: []string{"BlockTextApp", "InputBlock"}, Handled: true, Pass: true, X: 20, Y: 64, Width: 320, Height: 200, BufferSlots: []int{5, 20, 64, 1, 0, 320, 200, 0, 0}, BeforeState: map[string]string{"BlockTextApp.focused_id": "0", "InputBlock.focused": "false"}, AfterState: map[string]string{"BlockTextApp.focused_id": "3", "InputBlock.focused": "true"}},
		{Order: 2, Kind: "text_input", TargetComponent: "InputBlock", DispatchPath: []string{"BlockTextApp", "InputBlock"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 1, TextLen: 4, TextBytesHex: "4f4bd0a2", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 1, 4}, BeforeState: map[string]string{"InputBlock.buffer": "", "InputBlock.caret": "0"}, AfterState: map[string]string{"InputBlock.buffer": "OKd0a2", "InputBlock.caret": "4"}},
	}
}

func blockEventRuntimeEventsForTest() []EventReport {
	return []EventReport{
		{Order: 1, Kind: "mouse_up", TargetComponent: "InputBlock", DispatchPath: []string{"BlockEventApp", "PanelBlock", "InputBlock"}, Handled: true, Pass: true, X: 40, Y: 80, Width: 320, Height: 200, BufferSlots: []int{5, 40, 80, 1, 0, 320, 200, 0, 0}, BeforeState: map[string]string{"BlockEventApp.focused_id": "0", "InputBlock.focused": "false"}, AfterState: map[string]string{"BlockEventApp.focused_id": "4", "InputBlock.focused": "true"}},
		{Order: 2, Kind: "text_input", TargetComponent: "InputBlock", DispatchPath: []string{"BlockEventApp", "PanelBlock", "InputBlock"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 1, TextLen: 2, TextBytesHex: "4f4b", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 1, 2}, BeforeState: map[string]string{"InputBlock.buffer": "", "InputBlock.caret": "0"}, AfterState: map[string]string{"InputBlock.buffer": "OK", "InputBlock.caret": "2"}},
		{Order: 3, Kind: "key_down", TargetComponent: "BlockEventApp", DispatchPath: []string{"BlockEventApp"}, Handled: true, Pass: true, Key: 9, Width: 320, Height: 200, TimestampMS: 2, BufferSlots: []int{3, 0, 0, 0, 9, 320, 200, 2, 0}, BeforeState: map[string]string{"BlockEventApp.focused_id": "4"}, AfterState: map[string]string{"BlockEventApp.focused_id": "6"}},
		{Order: 4, Kind: "key_down", TargetComponent: "BlockEventApp", DispatchPath: []string{"BlockEventApp"}, Handled: true, Pass: true, Key: 9, Width: 320, Height: 200, TimestampMS: 3, BufferSlots: []int{3, 0, 0, 0, 9, 320, 200, 3, 0}, BeforeState: map[string]string{"BlockEventApp.focused_id": "6"}, AfterState: map[string]string{"BlockEventApp.focused_id": "4"}},
	}
}

func blockPaintLayersForTest() []PaintLayerReport {
	return []PaintLayerReport{
		{ID: "root-fill", BlockID: 1, Kind: "fill", Color: "#346ecfff", Radius: 8, Opacity: 255},
		{ID: "root-gradient", BlockID: 1, Kind: "gradient", Color: "#54b484ff", Radius: 8, Opacity: 255},
		{ID: "root-border", BlockID: 1, Kind: "border", Color: "#e2eaf2ff", Radius: 8, Width: 1, Opacity: 255},
		{ID: "root-shadow", BlockID: 1, Kind: "shadow", Color: "#00000058", Blur: 12, OffsetX: 0, OffsetY: 4, Opacity: 88},
		{ID: "root-outline", BlockID: 1, Kind: "outline", Color: "#f4cd5cff", Radius: 10, Width: 2, Opacity: 255},
	}
}

func blockPaintCommandsForTest() []PaintCommandReport {
	return []PaintCommandReport{
		{Order: 1, Command: "fill", LayerID: "root-fill", BlockID: 1, Rect: RectReport{X: 12, Y: 10, W: 64, H: 28}, Radius: 8, Quality: "rounded-rect-v1", Checksum: "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		{Order: 2, Command: "gradient", LayerID: "root-gradient", BlockID: 1, Rect: RectReport{X: 12, Y: 10, W: 64, H: 28}, Radius: 8, Quality: "two-stop-linear-v1", Checksum: "sha256:2222222222222222222222222222222222222222222222222222222222222222"},
		{Order: 3, Command: "border", LayerID: "root-border", BlockID: 1, Rect: RectReport{X: 12, Y: 10, W: 64, H: 28}, Radius: 8, Quality: "rounded-outline-v1", Checksum: "sha256:3333333333333333333333333333333333333333333333333333333333333333"},
		{Order: 4, Command: "shadow", LayerID: "root-shadow", BlockID: 1, Rect: RectReport{X: 12, Y: 10, W: 64, H: 28}, Radius: 8, Quality: "box-shadow-approx-v1", Checksum: "sha256:4444444444444444444444444444444444444444444444444444444444444444"},
		{Order: 5, Command: "outline", LayerID: "root-outline", BlockID: 1, Rect: RectReport{X: 10, Y: 8, W: 68, H: 32}, Radius: 10, Quality: "rounded-outline-v1", Checksum: "sha256:5555555555555555555555555555555555555555555555555555555555555555"},
	}
}

func rendererSceneForTest(source string, renderer string) *RendererSceneReport {
	commands := []RendererPaintCommandReport{
		{Order: 1, Command: "fill", Source: "paint_layers", LayerID: "root-fill", BlockID: 1, Rect: RectReport{X: 12, Y: 10, W: 64, H: 28}, Clip: RectReport{X: 12, Y: 10, W: 64, H: 28}, Radius: 8, Quality: "rounded-rect-v1", Checksum: "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		{Order: 2, Command: "gradient", Source: "paint_layers", LayerID: "root-gradient", BlockID: 1, Rect: RectReport{X: 12, Y: 10, W: 64, H: 28}, Clip: RectReport{X: 12, Y: 10, W: 64, H: 28}, Radius: 8, Quality: "two-stop-linear-v1", Checksum: "sha256:2222222222222222222222222222222222222222222222222222222222222222"},
		{Order: 3, Command: "border", Source: "paint_layers", LayerID: "root-border", BlockID: 1, Rect: RectReport{X: 12, Y: 10, W: 64, H: 28}, Clip: RectReport{X: 12, Y: 10, W: 64, H: 28}, Radius: 8, Quality: "rounded-outline-v1", Checksum: "sha256:3333333333333333333333333333333333333333333333333333333333333333"},
		{Order: 4, Command: "radius", Source: "paint_layers", LayerID: "root-fill", BlockID: 1, Rect: RectReport{X: 12, Y: 10, W: 64, H: 28}, Clip: RectReport{X: 12, Y: 10, W: 64, H: 28}, Radius: 8, Quality: "radius-mask-v1", Checksum: "sha256:4444444444444444444444444444444444444444444444444444444444444444"},
		{Order: 5, Command: "shadow", Source: "paint_layers", LayerID: "root-shadow", BlockID: 1, Rect: RectReport{X: 12, Y: 14, W: 68, H: 32}, Clip: RectReport{X: 0, Y: 0, W: 320, H: 200}, Radius: 8, Quality: "box-shadow-approx-v1", Checksum: "sha256:5555555555555555555555555555555555555555555555555555555555555555"},
		{Order: 6, Command: "outline", Source: "paint_layers", LayerID: "root-outline", BlockID: 1, Rect: RectReport{X: 10, Y: 8, W: 68, H: 32}, Clip: RectReport{X: 0, Y: 0, W: 320, H: 200}, Radius: 10, Quality: "rounded-outline-v1", Checksum: "sha256:6666666666666666666666666666666666666666666666666666666666666666"},
		{Order: 7, Command: "image", Source: "block_asset_render_commands", AssetID: "image-hero", BlockID: 3, Rect: RectReport{X: 72, Y: 32, W: 96, H: 64}, Clip: RectReport{X: 72, Y: 32, W: 96, H: 64}, Quality: "nearest-scale-v1", Checksum: "sha256:7777777777777777777777777777777777777777777777777777777777777777"},
		{Order: 8, Command: "text", Source: "text_render_commands", TextMeasurementID: "input-measure", BlockID: 6, Rect: RectReport{X: 12, Y: 48, W: 120, H: 18}, Clip: RectReport{X: 12, Y: 48, W: 144, H: 36}, Quality: "deterministic-glyph-markers-v1", Checksum: "sha256:8888888888888888888888888888888888888888888888888888888888888888"},
		{Order: 9, Command: "clip", Source: "layout_passes", BlockID: 7, Rect: RectReport{X: 236, Y: 72, W: 72, H: 80}, Clip: RectReport{X: 236, Y: 72, W: 72, H: 80}, Quality: "scroll-clip-v1", Checksum: "sha256:9999999999999999999999999999999999999999999999999999999999999999"},
		{Order: 10, Command: "transform", Source: "motion_frames", BlockID: 2, Rect: RectReport{X: 24, Y: 40, W: 168, H: 56}, Clip: RectReport{X: 0, Y: 0, W: 320, H: 200}, Transform: "scale(0.97)", Quality: "deterministic-transform-v1", Checksum: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}
	return &RendererSceneReport{
		Schema:              "tetra.surface.renderer-scene.v1",
		Version:             "1.0.0",
		Source:              source,
		Renderer:            renderer,
		ResolvedSceneSchema: "tetra.surface.resolved-scene.v1",
		PaintCommandSchema:  "tetra.surface.paint-command.v1",
		Deterministic:       true,
		CommandCount:        len(commands),
		CommandOrder:        []string{"fill", "gradient", "border", "radius", "shadow", "outline", "image", "text", "clip", "transform"},
		CommandHash:         "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Commands:            commands,
		UnsupportedVisuals: []RendererUnsupportedVisualReport{
			{Feature: "blur", Rejected: true, Reason: "unsupported until software blur gate exists"},
			{Feature: "backdrop_blur", Rejected: true, Reason: "unsupported without compositor/backdrop evidence"},
		},
	}
}

func softwareRendererForTest(source string, target string, backend string, frames []FrameReport) *SoftwareRendererReport {
	rendererFrames := make([]SoftwareRendererFrameReport, 0, len(frames))
	for _, frame := range frames {
		rendererFrames = append(rendererFrames, SoftwareRendererFrameReport{
			Order:                   frame.Order,
			Width:                   frame.Width,
			Height:                  frame.Height,
			Stride:                  frame.Stride,
			Scale:                   1,
			DPI:                     96,
			PixelChecksum:           frame.Checksum,
			RepeatChecksum:          frame.Checksum,
			GoldenChecksum:          frame.Checksum,
			GoldenArtifact:          fmt.Sprintf("goldens/surface/%s/frame-%02d.rgba", target, frame.Order),
			AlphaChecksum:           "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			ClipChecksum:            "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			Presented:               frame.Presented,
			UseAfterPresentRejected: true,
			AliasViolationRejected:  true,
		})
	}
	return &SoftwareRendererReport{
		Schema:              "tetra.surface.software-renderer.v1",
		QualityLevel:        "software-rgba-production-hardening-v1",
		Source:              source,
		Target:              target,
		Backend:             backend,
		PixelFormat:         "rgba8",
		AlphaBlendMode:      "source-over-rgba8-v1",
		ClipMode:            "rect-scissor-v1",
		RasterDeterministic: true,
		GoldenExport:        true,
		ResizeScaleDPI:      true,
		NoUseAfterPresent:   true,
		NoFrameAlias:        true,
		FeatureSummary:      []string{"deterministic-raster", "alpha-blending", "clipping", "frame-checksum", "golden-export", "resize", "scale", "dpi"},
		Frames:              rendererFrames,
		NegativeGuards: SoftwareRendererNegativeGuardsReport{
			MetadataOnlyFrameRejected:        true,
			UnchangedChecksumRejected:        true,
			MissingGoldenRejected:            true,
			MissingAlphaRejected:             true,
			MissingClipRejected:              true,
			MissingDPIRejected:               true,
			UseAfterPresentRejected:          true,
			FrameAliasRejected:               true,
			NodeOnlyBrowserPromotionRejected: true,
		},
	}
}

func softwareRendererCasesForTest() []CaseReport {
	return []CaseReport{
		{Name: "software renderer deterministic raster", Kind: "positive", Ran: true, Pass: true},
		{Name: "software renderer alpha clipping", Kind: "positive", Ran: true, Pass: true},
		{Name: "software renderer golden export", Kind: "positive", Ran: true, Pass: true},
		{Name: "software renderer resize scale dpi", Kind: "positive", Ran: true, Pass: true},
		{Name: "software renderer use-after-present rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "use-after-present"},
		{Name: "software renderer frame alias rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "frame alias"},
		{Name: "software renderer browser promotion without runtime rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "browser promotion without runtime"},
	}
}

func blockTextMeasurementsForTest() []TextMeasurementReport {
	return []TextMeasurementReport{
		{ID: "title-measure", BlockID: 2, TextLen: 28, FontFamily: "Tetra UI", FontWeight: 600, FontSize: 16, LineHeight: 20, MaxWidth: 96, Measured: SizeReport{W: 96, H: 40}, LineCount: 2, Wrap: "word", Overflow: "ellipsis", Ellipsis: true, EllipsizedTextLen: 16, Align: "start", Quality: "deterministic-metrics-v1", Checksum: "sha256:6666666666666666666666666666666666666666666666666666666666666666"},
		{ID: "input-measure", BlockID: 6, TextLen: 4, FontFamily: "Tetra UI", FontWeight: 400, FontSize: 14, LineHeight: 18, MaxWidth: 120, Measured: SizeReport{W: 34, H: 18}, LineCount: 1, Wrap: "none", Overflow: "clip", Ellipsis: false, EllipsizedTextLen: 4, Align: "start", Quality: "deterministic-metrics-v1", Checksum: "sha256:7777777777777777777777777777777777777777777777777777777777777777"},
	}
}

func blockFontFallbacksForTest() []FontFallbackReport {
	return []FontFallbackReport{
		{ID: "ui-fallback", RequestedFamily: "Tetra UI", ResolvedFamily: "Tetra UI Fallback", Chain: []string{"Tetra UI", "Noto Sans", "monospace"}, MissingGlyphs: 0, Coverage: "ascii-plus-basic-utf8-smoke"},
	}
}

func blockGlyphCachesForTest() []GlyphCacheReport {
	return []GlyphCacheReport{
		{ID: "glyph-cache", Strategy: "bounded-lru", BudgetBytes: 65536, UsedBytes: 4096, EntryCount: 12, Eviction: "lru", Bounded: true},
	}
}

func blockTextRenderCommandsForTest() []TextRenderCommandReport {
	return []TextRenderCommandReport{
		{Order: 1, Command: "measure", MeasurementID: "title-measure", BlockID: 2, Rect: RectReport{X: 12, Y: 10, W: 96, H: 40}, Clip: RectReport{X: 12, Y: 10, W: 96, H: 40}, Color: "#edf2f7ff", Opacity: 255, Quality: "deterministic-text-measure-v1", Checksum: "sha256:8888888888888888888888888888888888888888888888888888888888888888"},
		{Order: 2, Command: "render_glyphs", MeasurementID: "title-measure", BlockID: 2, Rect: RectReport{X: 12, Y: 10, W: 96, H: 40}, Clip: RectReport{X: 12, Y: 10, W: 96, H: 40}, Color: "#edf2f7ff", Opacity: 255, Quality: "deterministic-glyph-markers-v1", Checksum: "sha256:9999999999999999999999999999999999999999999999999999999999999999"},
		{Order: 3, Command: "render_caret", MeasurementID: "input-measure", BlockID: 6, Rect: RectReport{X: 12, Y: 48, W: 120, H: 18}, Clip: RectReport{X: 12, Y: 48, W: 144, H: 36}, Color: "#f4cd5cff", Opacity: 255, Quality: "deterministic-caret-v1", Checksum: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}
}

func removeString(values []string, value string) []string {
	filtered := values[:0]
	for _, current := range values {
		if current == value {
			continue
		}
		filtered = append(filtered, current)
	}
	return filtered
}

func removePaintLayerKind(layers []PaintLayerReport, kind string) []PaintLayerReport {
	filtered := layers[:0]
	for _, layer := range layers {
		if layer.Kind == kind {
			continue
		}
		filtered = append(filtered, layer)
	}
	return filtered
}

func removePaintCommand(commands []PaintCommandReport, command string) []PaintCommandReport {
	filtered := commands[:0]
	for _, current := range commands {
		if current.Command == command {
			continue
		}
		filtered = append(filtered, current)
	}
	return filtered
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

func removeBlockLayoutPassMode(passes []BlockLayoutPassReport, mode string) []BlockLayoutPassReport {
	filtered := passes[:0]
	for _, current := range passes {
		if normalizeLayoutToken(current.Mode) == normalizeLayoutToken(mode) {
			continue
		}
		filtered = append(filtered, current)
	}
	return filtered
}

func blockGraphReportForTest(source string) *BlockGraphReport {
	graph := &BlockGraphReport{
		Schema:            "tetra.surface.block-graph.v1",
		APILevel:          "block-tree-builder-v1",
		Source:            source,
		ManualBookkeeping: false,
		Builder: BlockGraphBuilderReport{
			RootCreatedBy:     "tree_add_root",
			ChildrenCreatedBy: "tree_add_child",
			NodeCount:         5,
			Capacity:          8,
			OverflowChecked:   true,
		},
		Invariants: BlockGraphInvariantReport{
			TreeValidateRan:         true,
			TreeValidateStatus:      0,
			DuplicateIDRejected:     true,
			MissingParentRejected:   true,
			CycleRejected:           true,
			ParentChildLinksChecked: true,
			ChildOrderChecked:       true,
			FocusOrderChecked:       true,
			HitTestPathChecked:      true,
			AccessibilityChecked:    true,
		},
		RootID:    1,
		NodeCount: 5,
		Nodes: []BlockGraphNodeReport{
			{ID: 1, Name: "RootBlock", ParentID: -1, ChildIndex: 0, FirstChild: 2, ChildCount: 1, Focusable: false, AccessibilityRole: "none", Bounds: RectReport{X: 0, Y: 0, W: 320, H: 200}},
			{ID: 2, Name: "PanelBlock", ParentID: 1, ChildIndex: 0, FirstChild: 3, ChildCount: 3, Focusable: false, AccessibilityRole: "none", Bounds: RectReport{X: 16, Y: 16, W: 288, H: 168}},
			{ID: 3, Name: "LabelBlock", ParentID: 2, ChildIndex: 0, FirstChild: -1, ChildCount: 0, Focusable: false, AccessibilityRole: "text", Bounds: RectReport{X: 24, Y: 24, W: 200, H: 24}},
			{ID: 4, Name: "SubmitBlock", ParentID: 2, ChildIndex: 1, FirstChild: -1, ChildCount: 0, Focusable: true, AccessibilityRole: "button", Bounds: RectReport{X: 24, Y: 64, W: 120, H: 44}},
			{ID: 5, Name: "ResetBlock", ParentID: 2, ChildIndex: 2, FirstChild: -1, ChildCount: 0, Focusable: true, AccessibilityRole: "button", Bounds: RectReport{X: 152, Y: 64, W: 120, H: 44}},
		},
		ChildOrders: []BlockGraphChildOrderReport{
			{ParentID: 1, Children: []int{2}},
			{ParentID: 2, Children: []int{3, 4, 5}},
		},
		LayoutOrder:        []int{1, 2, 3, 4, 5},
		DrawOrder:          []int{1, 2, 3, 4, 5},
		FocusOrder:         []int{4, 5},
		AccessibilityOrder: []int{3, 4, 5},
		HitTests: []BlockGraphPathReport{
			{Helper: "tree_hit_test_path", Event: "click", TargetID: 5, X: 180, Y: 80, Path: []int{1, 2, 5}},
		},
		DispatchPaths: []BlockGraphPathReport{
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 4, Path: []int{1, 2, 4}},
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 5, Path: []int{1, 2, 5}},
		},
	}
	return withBlockGraphABIAndResolvedSceneForTest(graph)
}

func withBlockGraphABIAndResolvedSceneForTest(graph *BlockGraphReport) *BlockGraphReport {
	graph.ABI = blockGraphABIForTest()
	graph.ResolvedScene = blockResolvedSceneForGraphForTest(graph)
	return graph
}

func blockGraphABIForTest() *BlockGraphABIReport {
	return &BlockGraphABIReport{
		Schema:            "tetra.surface.block-abi.v1",
		Version:           "1.0.0",
		BlockType:         "lib.core.block.Block",
		TreeType:          "lib.core.block.BlockTree",
		PropsType:         "lib.core.block.BlockProps",
		ResolvedBlockType: "tetra.surface.renderer.ResolvedBlock",
		ResolvedSceneType: "tetra.surface.renderer.ResolvedScene",
		StableFields: []string{
			"id",
			"parent_id",
			"child_index",
			"bounds",
			"layout_order",
			"draw_order",
			"hit_test_order",
			"focus_order",
			"accessibility_order",
		},
		Compatibility: []string{
			"semver-compatible",
			"additive-fields-only",
			"same-commit-validator",
		},
	}
}

func blockResolvedSceneForGraphForTest(graph *BlockGraphReport) *BlockResolvedSceneReport {
	return &BlockResolvedSceneReport{
		Schema:                   "tetra.surface.resolved-scene.v1",
		SourceGraphSchema:        graph.Schema,
		RootID:                   graph.RootID,
		NodeOrder:                blockGraphNodeOrderForTest(graph),
		LayoutOrder:              append([]int(nil), graph.LayoutOrder...),
		DrawOrder:                append([]int(nil), graph.DrawOrder...),
		HitTestOrder:             blockGraphHitTestOrderForTest(graph),
		FocusOrder:               append([]int(nil), graph.FocusOrder...),
		AccessibilityOrder:       append([]int(nil), graph.AccessibilityOrder...),
		TreeOrderStable:          true,
		DrawOrderStable:          true,
		HitTestOrderStable:       true,
		FocusOrderStable:         true,
		AccessibilityOrderStable: true,
	}
}

func blockGraphNodeOrderForTest(graph *BlockGraphReport) []int {
	order := make([]int, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		order = append(order, node.ID)
	}
	return order
}

func blockGraphHitTestOrderForTest(graph *BlockGraphReport) []int {
	order := make([]int, 0, len(graph.HitTests))
	for _, hit := range graph.HitTests {
		order = append(order, hit.TargetID)
	}
	return order
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
	report["accessibility_target"] = releaseAccessibilityTargetMap("linux-x64", "surface-linux-x64", "linux_accessibility_host_bridge_v1", "linux-accessibility-platform-probe-v1", true, false, false)
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

func releaseAccessibilityTargetMap(target string, runtime string, platformBridge string, inspector string, hostBridge bool, browserSnapshot bool, browserMirror bool) map[string]any {
	return map[string]any{
		"schema":                   "tetra.surface.accessibility-target.v1",
		"level":                    "production-accessibility-target-v1",
		"target":                   target,
		"runtime":                  runtime,
		"release_scope":            "surface-v1-linux-web",
		"tree_schema":              "tetra.surface.accessibility-tree.v1",
		"platform_bridge":          platformBridge,
		"inspector":                inspector,
		"screen_reader_evidence":   platformBridge,
		"metadata_tree":            true,
		"platform_export":          true,
		"host_bridge":              hostBridge,
		"browser_snapshot":         browserSnapshot,
		"browser_mirror":           browserMirror,
		"full_screen_reader_claim": false,
		"role_count":               9,
		"named_node_count":         12,
		"state_node_count":         12,
		"relationship_count":       4,
		"action_count":             4,
		"focus_order_count":        4,
		"reading_order_count":      8,
		"snapshot_count":           8,
		"negative_guards": map[string]any{
			"focusable_unnamed_rejected":                true,
			"aria_dom_desktop_bridge_rejected":          true,
			"full_atspi_without_screen_reader_rejected": true,
			"metadata_platform_overclaim_rejected":      true,
			"shuffled_focus_order_rejected":             true,
			"shuffled_reading_order_rejected":           true,
		},
		"nonclaims": []any{
			"full screen-reader parity",
			"desktop aria bridge",
			"metadata platform overclaim",
			"unnamed focusable block",
			"AT-SPI full support",
		},
	}
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
	report["browser_canvas_target"] = map[string]any{
		"schema":                 "tetra.surface.browser-canvas-target.v1",
		"level":                  "wasm32-web-first-class-browser-canvas-target-v1",
		"target":                 "wasm32-web",
		"runtime":                "surface-wasm32-web",
		"host_abi":               "tetra.surface.host-abi.v1",
		"backend":                "browser-canvas-rgba-accessible",
		"trace_schema":           "tetra.surface.browser-canvas-trace.v1",
		"compiler_owned_boot":    true,
		"compiler_owned_loader":  true,
		"user_js_app_logic":      false,
		"dom_ui":                 false,
		"react_runtime":          false,
		"browser_canvas":         true,
		"browser_input":          true,
		"browser_clipboard":      true,
		"browser_composition":    true,
		"accessibility_snapshot": true,
		"accessibility_mirror":   true,
		"frame_checksum_count":   2,
		"event_kinds":            []any{"mouse_up", "key_down", "resize", "text_input"},
		"artifact_kinds":         []any{"component-app", "compiler-owned-loader", "runner-trace"},
		"negative_guards": map[string]any{
			"node_only_rejected":                true,
			"dom_snapshot_renderer_rejected":    true,
			"user_js_command_dispatch_rejected": true,
			"metadata_sidecar_rejected":         true,
			"legacy_sidecar_rejected":           true,
			"react_runtime_rejected":            true,
		},
		"nonclaims": []any{
			"DOM snapshot renderer",
			"user script command dispatch",
			"React runtime",
			"node runtime substitution",
			"metadata sidecar",
			"legacy sidecar",
		},
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
	report["linux_host_adapter"] = linuxHostAdapterMap()
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
		map[string]any{"name": "linux production host adapter target-host trace", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux production app shell adapter", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux production packaging scope", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux production blocked display not pass", "kind": "negative", "ran": true, "pass": true, "expected_error": "blocked display pass rejected"},
		map[string]any{"name": "linux production offscreen promotion rejected", "kind": "negative", "ran": true, "pass": true, "expected_error": "headless promotion rejected"},
		map[string]any{"name": "linux production old real-window promotion rejected", "kind": "negative", "ran": true, "pass": true, "expected_error": "old real-window promotion rejected"},
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

func linuxHostAdapterMap() map[string]any {
	return map[string]any{
		"schema":               "tetra.surface.linux-host-adapter.v1",
		"level":                "linux-x64-production-host-adapter-v1",
		"target":               "linux-x64",
		"runtime":              "surface-linux-x64",
		"host_abi":             "tetra.surface.host-abi.v1",
		"app_shell_abi":        "tetra.surface.app-shell.v1",
		"backend":              "wayland-shm-rgba-release-v1",
		"display_protocol":     "wayland",
		"window_system":        "wayland-shm-rgba",
		"target_host_trace":    "surface-linux-x64-release-window-target-host-trace-v1",
		"real_window":          true,
		"framebuffer":          true,
		"native_input":         true,
		"text_input":           true,
		"ime":                  true,
		"clipboard":            true,
		"composition":          true,
		"accessibility_bridge": true,
		"app_shell":            true,
		"packaging": map[string]any{
			"scope":           "linux-x64-unpacked-binary-v1",
			"artifact_kind":   "component-app",
			"artifact_path":   "/tmp/surface-artifacts/surface-release-form",
			"unpacked_binary": true,
			"installable":     false,
			"signed":          false,
			"auto_update":     false,
			"full_packaging":  false,
		},
		"traces": []any{
			map[string]any{"id": "linux-trace-real-window", "kind": "real_window", "target": "linux-x64", "artifact": "/tmp/surface-artifacts/surface-release-window-probe", "delivered": true, "target_host": true},
			map[string]any{"id": "linux-trace-framebuffer", "kind": "framebuffer", "target": "linux-x64", "artifact": "/tmp/surface-artifacts/surface-release-window-probe", "delivered": true, "target_host": true},
			map[string]any{"id": "linux-trace-native-input", "kind": "native_input", "target": "linux-x64", "artifact": "/tmp/surface-artifacts/surface-release-window-probe", "delivered": true, "target_host": true},
			map[string]any{"id": "linux-trace-text-input", "kind": "text_input", "target": "linux-x64", "artifact": "/tmp/surface-artifacts/surface-release-window-probe", "delivered": true, "target_host": true},
			map[string]any{"id": "linux-trace-ime", "kind": "ime", "target": "linux-x64", "artifact": "/tmp/surface-artifacts/surface-linux-composition-harness.json", "delivered": true, "target_host": true},
			map[string]any{"id": "linux-trace-clipboard", "kind": "clipboard", "target": "linux-x64", "artifact": "/tmp/surface-artifacts/surface-linux-clipboard-harness.json", "delivered": true, "target_host": true},
			map[string]any{"id": "linux-trace-composition", "kind": "composition", "target": "linux-x64", "artifact": "/tmp/surface-artifacts/surface-linux-composition-harness.json", "delivered": true, "target_host": true},
			map[string]any{"id": "linux-trace-accessibility", "kind": "accessibility_bridge", "target": "linux-x64", "artifact": "/tmp/surface-artifacts/surface-linux-accessibility-bridge.json", "delivered": true, "target_host": true},
			map[string]any{"id": "linux-trace-app-shell", "kind": "app_shell", "target": "linux-x64", "artifact": "tetra.surface.app-shell.v1", "delivered": true, "target_host": true},
			map[string]any{"id": "linux-trace-packaging", "kind": "packaging_scope", "target": "linux-x64", "artifact": "/tmp/surface-artifacts/surface-release-form", "delivered": true, "target_host": true},
		},
		"diagnostics": []any{
			map[string]any{"id": "linux-diag-blocked-display", "kind": "blocked_display_not_pass", "message": "missing WAYLAND_DISPLAY/DISPLAY must produce blocked report, not pass", "rejected": true},
			map[string]any{"id": "linux-diag-headless-promotion", "kind": "headless_promotion_rejected", "message": "headless-only evidence cannot promote to linux production", "rejected": true},
			map[string]any{"id": "linux-diag-old-real-window", "kind": "old_real_window_promotion_rejected", "message": "linux-x64-real-window evidence cannot promote to release-window production", "rejected": true},
		},
		"negative_guards": map[string]any{
			"headless_promotion_rejected":        true,
			"blocked_display_pass_rejected":      true,
			"old_real_window_promotion_rejected": true,
			"missing_app_shell_rejected":         true,
			"missing_packaging_scope_rejected":   true,
			"missing_target_host_trace_rejected": true,
		},
		"nonclaims": []any{
			"offscreen-only production",
			"blocked display pass",
			"full installer packaging",
			"signed packages",
			"auto-update",
		},
	}
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
