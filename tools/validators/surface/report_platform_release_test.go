package surface

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateSurfaceBrowserReleaseReport(t *testing.T) {
	raw := validWASM32WebReleaseBrowserSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceBrowserReleaseRequiresFirstClassBrowserSurfaceEvidence(t *testing.T) {
	raw := validWASM32WebReleaseBrowserSurfaceReportJSON(t, func(report map[string]any) {
		delete(report, "browser_surface")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected browser release without first-class browser_surface evidence to fail")
	}
	if !strings.Contains(err.Error(), "browser_surface") {
		t.Fatalf("error = %v, want browser_surface diagnostic", err)
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
func TestValidateSurfaceLinuxAppShellReport(t *testing.T) {
	raw := validLinuxAppShellSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceLinuxAppShellRejectsMissingP17SecurityPermissions(t *testing.T) {
	raw := validLinuxAppShellSurfaceReportJSON(t, func(report map[string]any) {
		delete(report, "security_permissions")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing P17 security permissions to fail")
	}
	if !strings.Contains(err.Error(), "security_permissions") {
		t.Fatalf("error = %v, want security_permissions diagnostic", err)
	}
}
func TestValidateSurfaceLinuxAppShellRejectsPermissionBypassForBlockedFeatures(t *testing.T) {
	raw := validLinuxAppShellSurfaceReportJSON(t, func(report map[string]any) {
		security := report["security_permissions"].(map[string]any)
		capabilities := security["capabilities"].([]any)
		for _, capability := range capabilities {
			row := capability.(map[string]any)
			if row["name"] == "notification" {
				row["status"] = "allowed_with_policy"
				row["allowed"] = true
				row["blocked_reason"] = ""
			}
		}
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected permission bypass for blocked notification feature to fail")
	}
	if !strings.Contains(err.Error(), "notification") || !strings.Contains(err.Error(), "security_permissions") {
		t.Fatalf("error = %v, want security_permissions notification diagnostic", err)
	}
}
func TestValidateSurfaceLinuxAppShellRejectsMissingP18PerformanceBudget(t *testing.T) {
	raw := validLinuxAppShellSurfaceReportJSON(t, func(report map[string]any) {
		delete(report, "surface_performance_budget")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing P18 performance budget to fail")
	}
	if !strings.Contains(err.Error(), "surface_performance_budget") {
		t.Fatalf("error = %v, want surface_performance_budget diagnostic", err)
	}
}
func TestValidateSurfacePerformanceBudgetRejectsFasterThanElectronClaim(t *testing.T) {
	raw := validLinuxAppShellSurfaceReportJSON(t, func(report map[string]any) {
		budget := report["surface_performance_budget"].(map[string]any)
		budget["performance_claim"] = "faster than Electron"
		budget["methodology"].(map[string]any)["electron_comparison"] = "faster than Electron"
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected fake faster-than-Electron claim to fail")
	}
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "faster than electron") && !strings.Contains(err.Error(), "surface_performance_budget") {
		t.Fatalf("error = %v, want faster than Electron performance diagnostic", err)
	}
}
func TestValidateSurfacePerformanceBudgetRejectsMissingPeakMemoryField(t *testing.T) {
	raw := validLinuxAppShellSurfaceReportJSON(t, func(report map[string]any) {
		budget := report["surface_performance_budget"].(map[string]any)
		memory := budget["memory"].(map[string]any)
		delete(memory, "peak_rss_bytes")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing peak memory field to fail")
	}
	if !strings.Contains(err.Error(), "peak_rss_bytes") {
		t.Fatalf("error = %v, want peak_rss_bytes diagnostic", err)
	}
}
func TestValidateSurfaceLinuxAppShellRejectsMissingP16FeatureLedgerRows(t *testing.T) {
	raw := validLinuxAppShellSurfaceReportJSON(t, func(report map[string]any) {
		appShell := report["linux_app_shell"].(map[string]any)
		appShell["shell_features"] = withoutLinuxAppShellFeature(p16LinuxAppShellFeaturesForTest(), "tray")
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing P16 tray ledger row to fail")
	}
	if !strings.Contains(err.Error(), "tray") {
		t.Fatalf("error = %v, want tray diagnostic", err)
	}
}
func TestValidateSurfaceLinuxAppShellRejectsTrayClaimWithoutTargetEvidence(t *testing.T) {
	raw := validLinuxAppShellSurfaceReportJSON(t, func(report map[string]any) {
		appShell := report["linux_app_shell"].(map[string]any)
		features := p16LinuxAppShellFeaturesForTest()
		for _, feature := range features {
			row := feature.(map[string]any)
			if row["name"] == "tray" {
				row["status"] = "scoped_adapter"
				row["claimed"] = true
				row["blocked_reason"] = ""
			}
		}
		appShell["shell_features"] = features
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected tray claim without target evidence to fail")
	}
	if !strings.Contains(err.Error(), "tray") || !strings.Contains(err.Error(), "target evidence") {
		t.Fatalf("error = %v, want tray target evidence diagnostic", err)
	}
}
func TestValidateSurfaceLinuxAppShellRejectsFakeProductionClaims(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "memfd starter host level",
			mutate: func(report map[string]any) {
				host := report["host_evidence"].(map[string]any)
				host["level"] = "linux-x64-memfd-starter"
				host["backend"] = "memfd-rgba"
				host["real_window"] = false
				host["native_input"] = false
			},
			want: "linux app-shell host_evidence.level",
		},
		{
			name: "missing reopen lifecycle",
			mutate: func(report map[string]any) {
				appShell := report["linux_app_shell"].(map[string]any)
				appShell["window_lifecycle"] = []any{
					map[string]any{"order": 1, "window_id": "notes-main", "operation": "open", "host_trace": true, "pass": true},
					map[string]any{"order": 2, "window_id": "notes-main", "operation": "close", "host_trace": true, "pass": true},
				}
			},
			want: "reopen",
		},
		{
			name: "native widget UI substitute",
			mutate: func(report map[string]any) {
				appShell := report["linux_app_shell"].(map[string]any)
				appShell["negative_guards"].(map[string]any)["no_gtk"] = false
			},
			want: "GTK/Qt/native widget UI",
		},
		{
			name: "file dialog claimed without blocked pass",
			mutate: func(report map[string]any) {
				appShell := report["linux_app_shell"].(map[string]any)
				features := appShell["shell_features"].([]any)
				for _, feature := range features {
					row := feature.(map[string]any)
					if row["name"] == "file_dialog" {
						row["status"] = "claimed-native-dialog"
						row["claimed"] = true
					}
				}
			},
			want: "file_dialog",
		},
		{
			name: "missing host trace artifact",
			mutate: func(report map[string]any) {
				artifacts := report["artifacts"].([]any)
				report["artifacts"] = artifacts[:1]
			},
			want: "linux-app-shell-host-trace",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validLinuxAppShellSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected linux app-shell fake claim to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
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
	report["browser_surface"] = validBrowserSurfaceEvidenceMap()
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
func validBrowserSurfaceEvidenceMap() map[string]any {
	return map[string]any{
		"schema":                "tetra.surface.browser-surface.v1",
		"browser_surface_level": "browser-canvas-release-v1",
		"release_scope":         "surface-v1-linux-web",
		"source":                "examples/surface_release_form.tetra",
		"host_adapter":          "compiler-owned-browser-canvas-host",
		"production_claim":      true,
		"experimental":          false,
		"compiler_owned_boot":   true,
		"dom_host_canvas_only":  true,
		"canvas": map[string]any{
			"opened":        true,
			"readback":      true,
			"width":         560,
			"height":        420,
			"frame_order":   5,
			"artifact_kind": "runner-trace",
			"pass":          true,
		},
		"input": map[string]any{
			"pointer":       true,
			"keyboard":      true,
			"text":          true,
			"resize":        true,
			"host_trace":    true,
			"native_events": []any{"pointerup", "keydown", "beforeinput", "resize"},
			"pass":          true,
		},
		"clipboard": map[string]any{
			"harness":    "deterministic-browser-clipboard-v1",
			"read":       true,
			"write":      true,
			"owned_copy": true,
			"bytes":      13,
			"pass":       true,
		},
		"composition": map[string]any{
			"start":  true,
			"update": true,
			"commit": true,
			"cancel": true,
			"pass":   true,
		},
		"accessibility": map[string]any{
			"snapshot":       true,
			"mirror":         true,
			"compiler_owned": true,
			"bounds":         true,
			"focus":          true,
			"roles":          []any{"root", "textbox", "checkbox", "button", "status"},
			"dom_visual_ui":  false,
			"user_js":        false,
			"pass":           true,
		},
		"host_traces": []any{
			map[string]any{"name": "browser-canvas", "artifact_kind": "runner-trace", "path": "/tmp/surface-artifacts/surface-runner-trace.json", "pass": true},
		},
		"negative_guards": map[string]any{
			"no_dom_app_ui_tree":     true,
			"no_user_js_app_logic":   true,
			"no_node_only_promotion": true,
			"no_legacy_sidecars":     true,
			"no_react_runtime":       true,
			"no_platform_widgets":    true,
		},
	}
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
func validLinuxAppShellSurfaceReportJSON(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode base headless report: %v", err)
	}
	report["target"] = "linux-x64"
	report["runtime"] = "surface-linux-x64"
	report["source"] = "examples/surface_linux_app_shell_notes.tetra"
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
		map[string]any{"name": "tetra build", "kind": "build", "path": "tetra build --target linux-x64 examples/surface_linux_app_shell_notes.tetra -o /tmp/surface-artifacts/surface-linux-app-shell-notes", "ran": true, "pass": true, "exit_code": 0},
		map[string]any{"name": "surface component app", "kind": "app", "path": "/tmp/surface-artifacts/surface-linux-app-shell-notes", "ran": true, "pass": true, "exit_code": 1, "expected_exit_code": 1},
		map[string]any{"name": "surface linux-x64 real-window probe", "kind": "app", "path": "/tmp/surface-artifacts/surface-linux-app-shell-window-probe", "ran": true, "pass": true, "exit_code": 42, "expected_exit_code": 42},
		map[string]any{"name": "surface linux app-shell host trace", "kind": "runtime", "path": "/tmp/surface-artifacts/surface-linux-app-shell-host-trace.json", "ran": true, "pass": true, "exit_code": 0},
		map[string]any{"name": "surface linux app-shell window trace", "kind": "runtime", "path": "/tmp/surface-artifacts/surface-linux-app-shell-window-trace.json", "ran": true, "pass": true, "exit_code": 0},
		map[string]any{"name": "surface linux-x64 release clipboard harness", "kind": "runtime", "path": "/tmp/surface-artifacts/surface-linux-clipboard-harness.json", "ran": true, "pass": true, "exit_code": 0},
		map[string]any{"name": "surface linux-x64 release composition harness", "kind": "runtime", "path": "/tmp/surface-artifacts/surface-linux-composition-harness.json", "ran": true, "pass": true, "exit_code": 0},
		map[string]any{"name": "surface linux accessibility platform probe", "kind": "runtime", "path": "/tmp/surface-artifacts/surface-linux-accessibility-probe.json", "ran": true, "pass": true, "exit_code": 0},
		map[string]any{"name": "surface linux-x64 runtime", "kind": "runtime", "path": "tools/cmd/surface-runtime-smoke --mode linux-x64-release-app-shell", "ran": true, "pass": true, "exit_code": 0},
	}
	report["artifacts"] = []any{
		map[string]any{"kind": "component-app", "path": "/tmp/surface-artifacts/surface-linux-app-shell-notes", "sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size": 90001},
		map[string]any{"kind": "linux-app-shell-host-trace", "path": "/tmp/surface-artifacts/surface-linux-app-shell-host-trace.json", "sha256": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "size": 4096},
		map[string]any{"kind": "linux-app-shell-window-trace", "path": "/tmp/surface-artifacts/surface-linux-app-shell-window-trace.json", "sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "size": 4096},
		map[string]any{"kind": "linux-accessibility-platform-probe", "path": "/tmp/surface-artifacts/surface-linux-accessibility-probe.json", "sha256": "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", "size": 4096},
	}
	report["artifact_scan"] = map[string]any{"root": "/tmp/surface-artifacts", "files_checked": 4, "forbidden_paths": []any{}, "pass": true}
	report["components"] = []any{
		componentMap("NotesShellApp", "examples.surface_linux_app_shell_notes.NotesShellApp", "", RectReport{X: 0, Y: 0, W: 720, H: 540}, map[string]string{"open_windows": "2", "focused_window": "notes-main", "accessibility_role": "application"}),
		componentMap("NotesMainWindow", "examples.surface_linux_app_shell_notes.NotesMainWindow", "NotesShellApp", RectReport{X: 0, Y: 0, W: 560, H: 420}, map[string]string{"title": "Notes", "lifecycle": "reopened", "dpi_scale_milli": "1250", "cursor": "text", "accessibility_role": "document"}),
		componentMap("NotesInspectorWindow", "examples.surface_linux_app_shell_notes.NotesInspectorWindow", "NotesShellApp", RectReport{X: 24, Y: 24, W: 320, H: 240}, map[string]string{"title": "Inspector", "lifecycle": "open", "dpi_scale_milli": "1000", "cursor": "pointer", "accessibility_role": "panel"}),
	}
	report["events"] = []any{
		map[string]any{"order": 1, "kind": "mouse_up", "target_component": "NotesMainWindow", "dispatch_path": []any{"NotesShellApp", "NotesMainWindow"}, "handled": true, "pass": true, "x": 40, "y": 72, "key": 0, "width": 560, "height": 420, "timestamp_ms": 0, "buffer_slots": []any{5, 40, 72, 1, 0, 560, 420, 0, 0}, "before_state": map[string]any{"NotesShellApp.focused_window": ""}, "after_state": map[string]any{"NotesShellApp.focused_window": "notes-main"}},
		map[string]any{"order": 2, "kind": "key_down", "target_component": "NotesMainWindow", "dispatch_path": []any{"NotesShellApp", "NotesMainWindow"}, "handled": true, "pass": true, "x": 0, "y": 0, "key": 78, "width": 560, "height": 420, "timestamp_ms": 2, "buffer_slots": []any{6, 0, 0, 1, 78, 560, 420, 2, 0}, "before_state": map[string]any{"NotesMainWindow.shortcut": ""}, "after_state": map[string]any{"NotesMainWindow.shortcut": "new-note"}},
		map[string]any{"order": 3, "kind": "text_input", "target_component": "NotesMainWindow", "dispatch_path": []any{"NotesShellApp", "NotesMainWindow"}, "handled": true, "pass": true, "x": 0, "y": 0, "key": 0, "width": 560, "height": 420, "timestamp_ms": 3, "text_len": 5, "text_bytes_hex": "4e6f746573", "buffer_slots": []any{8, 0, 0, 0, 0, 560, 420, 3, 5}, "before_state": map[string]any{"NotesMainWindow.buffer": ""}, "after_state": map[string]any{"NotesMainWindow.buffer": "Notes"}},
		map[string]any{"order": 4, "kind": "resize", "target_component": "NotesMainWindow", "dispatch_path": []any{"NotesShellApp", "NotesMainWindow"}, "handled": true, "pass": true, "width": 720, "height": 540, "timestamp_ms": 4, "buffer_slots": []any{7, 0, 0, 0, 0, 720, 540, 4, 0}, "before_state": map[string]any{"NotesMainWindow.size": "560x420", "NotesMainWindow.dpi": "1000"}, "after_state": map[string]any{"NotesMainWindow.size": "720x540", "NotesMainWindow.dpi": "1250"}},
		map[string]any{"order": 5, "kind": "close", "target_component": "NotesInspectorWindow", "dispatch_path": []any{"NotesShellApp", "NotesInspectorWindow"}, "handled": true, "pass": true, "width": 320, "height": 240, "timestamp_ms": 5, "buffer_slots": []any{9, 0, 0, 0, 0, 320, 240, 5, 0}, "before_state": map[string]any{"NotesInspectorWindow.open": "true"}, "after_state": map[string]any{"NotesInspectorWindow.open": "false"}},
	}
	report["frames"] = []any{
		map[string]any{"order": 1, "width": 400, "height": 240, "stride": 1600, "checksum": "1111111111111111111111111111111111111111111111111111111111111111", "presented": true},
		map[string]any{"order": 5, "width": 560, "height": 420, "stride": 2240, "checksum": "2222222222222222222222222222222222222222222222222222222222222222", "presented": true},
		map[string]any{"order": 6, "width": 720, "height": 540, "stride": 2880, "checksum": "3333333333333333333333333333333333333333333333333333333333333333", "presented": true},
	}
	report["state_transitions"] = []any{
		map[string]any{"order": 1, "component": "NotesShellApp", "field": "focused_window", "before": "", "after": "notes-main", "cause": "lifecycle.open"},
		map[string]any{"order": 2, "component": "NotesInspectorWindow", "field": "open", "before": "true", "after": "false", "cause": "lifecycle.close"},
		map[string]any{"order": 3, "component": "NotesMainWindow", "field": "size", "before": "560x420", "after": "720x540", "cause": "resize"},
	}
	report["cases"] = []any{
		map[string]any{"name": "pure Tetra component app", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "host-provided pointer event dispatch", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "host event buffer poll_event", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "pre/post event frame sequence", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "component hierarchy dispatch", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "component text input scalar dispatch", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "host text payload buffer", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "component focus dispatch", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "component accessibility metadata", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "no legacy UI sidecar artifacts", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "state transition", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "reject legacy UI evidence", "kind": "negative", "ran": true, "pass": true, "expected_error": "legacy UI evidence rejected"},
		map[string]any{"name": "linux-x64 real-window surface", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux-x64 native input event pump", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux-x64 real-window resize event", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux-x64 real-window close event", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux release real window presented frame", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux release accessibility bridge probe", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux app-shell v1 schema", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux app-shell lifecycle open close reopen", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux app-shell multi-window notes reference", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux app-shell resize dpi cursor trace", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux app-shell clipboard ime accessibility adapters", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux app-shell file dialog notification blocked-pass", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux app-shell electron feature ledger", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux app-shell dialog file picker tray blocked-pass", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux app-shell crash error report scoped adapters", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "linux app-shell rejects GTK Qt native widget UI", "kind": "negative", "ran": true, "pass": true, "expected_error": "native widget UI rejected"},
		map[string]any{"name": "linux app-shell no Electron React DOM application scripting", "kind": "negative", "ran": true, "pass": true, "expected_error": "runtime substitute rejected"},
	}
	report["linux_app_shell"] = validLinuxAppShellEvidenceMap()
	report["security_permissions"] = validSurfaceSecurityPermissionsMap(p16LinuxAppShellFeaturesForTest())
	report["surface_performance_budget"] = validSurfacePerformanceBudgetMap("linux-x64", "surface-linux-x64", "examples/surface_linux_app_shell_notes.tetra")
	if mutate != nil {
		mutate(report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal linux app-shell report: %v", err)
	}
	return raw
}
func validSurfacePerformanceBudgetMap(target string, runtimeName string, source string) map[string]any {
	return map[string]any{
		"schema":             "tetra.surface.performance-budget.v1",
		"model":              "surface-performance-budget-v1",
		"release_scope":      "surface-v1-linux-web",
		"source":             source,
		"target":             target,
		"runtime":            runtimeName,
		"production_claim":   true,
		"experimental":       false,
		"git_head":           "0123456789abcdef0123456789abcdef01234567",
		"performance_claim":  "none",
		"startup":            map[string]any{"launch_to_first_frame_ms": 18, "budget_ms": 250, "trace": "local-startup-trace-v1", "pass": true},
		"frame":              map[string]any{"frame_count": 3, "p50_build_ms": 4, "p95_build_ms": 7, "p50_present_ms": 3, "p95_present_ms": 6, "budget_ms": 16, "idle_loop_count": 24, "work_loop_count": 6, "pass": true},
		"scene":              map[string]any{"block_count": 3, "recipe_expansion_count": 0, "paint_command_count": 10, "layout_pass_count": 4, "text_run_count": 2},
		"memory":             map[string]any{"glyph_cache_bytes": 4096, "asset_cache_bytes": 5376, "layout_cache_bytes": 4096, "paint_cache_bytes": 10240, "framebuffer_peak_bytes": 1555200, "framebuffer_total_bytes": 2880000, "rss_measured": false, "peak_rss_bytes": 0, "allocation_count": 42, "allocation_bytes": 2903808, "bounded_caches": true, "unbounded_cache_rejected": true, "pass": true},
		"binary":             map[string]any{"artifact_path": "/tmp/surface-artifacts/surface-linux-app-shell-notes", "size_bytes": 90001, "budget_bytes": 16777216, "pass": true},
		"cpu_power_proxy":    map[string]any{"idle_loop_count": 24, "work_loop_count": 6, "idle_frame_count": 2, "work_frame_count": 1, "real_power_measured": false, "pass": true},
		"cache":              map[string]any{"glyph_cache_budget_bytes": 65536, "asset_cache_budget_bytes": 65536, "layout_cache_budget_bytes": 65536, "paint_cache_budget_bytes": 65536, "total_cache_bytes": 23808, "total_cache_budget_bytes": 262144, "eviction": "bounded-lru", "pass": true},
		"methodology":        map[string]any{"kind": "local-deterministic-budget-v1", "electron_comparison": "none", "official_benchmark": false, "cross_machine": false, "fair_comparison_required_for_electron_claim": true},
		"unsupported_claims": []any{"faster-than-electron", "lower-power-than-electron", "official-benchmark-result", "cross-machine-benchmark", "electron-parity-performance"},
		"negative_guards":    map[string]any{"bounded_caches": true, "unbounded_cache_rejected": true, "stale_report_rejected": true, "no_faster_than_electron_claim": true, "no_benchmark_parity_claim": true, "peak_memory_field_required": true, "no_official_benchmark_claim": true},
	}
}
func validLinuxAppShellEvidenceMap() map[string]any {
	return map[string]any{
		"schema":           "tetra.surface.linux-app-shell.v1",
		"app_shell_level":  "linux-app-shell-subset-v1",
		"release_scope":    "surface-v1-linux-web",
		"source":           "examples/surface_linux_app_shell_notes.tetra",
		"module":           "lib.core.surface_app_shell",
		"host_adapter":     "wayland-shm-rgba-release-v1",
		"production_claim": true,
		"experimental":     false,
		"window_lifecycle": []any{
			map[string]any{"order": 1, "window_id": "notes-main", "operation": "open", "host_trace": true, "pass": true},
			map[string]any{"order": 2, "window_id": "notes-inspector", "operation": "open", "host_trace": true, "pass": true},
			map[string]any{"order": 3, "window_id": "notes-inspector", "operation": "close", "host_trace": true, "pass": true},
			map[string]any{"order": 4, "window_id": "notes-inspector", "operation": "reopen", "host_trace": true, "pass": true},
		},
		"windows": []any{
			map[string]any{"id": "notes-main", "title": "Notes", "role": "primary", "block_root": "NotesMainWindow", "real_window": true, "presented": true, "width": 720, "height": 540, "dpi_scale_milli": 1250},
			map[string]any{"id": "notes-inspector", "title": "Inspector", "role": "secondary", "block_root": "NotesInspectorWindow", "real_window": true, "presented": true, "width": 320, "height": 240, "dpi_scale_milli": 1000},
		},
		"resize_dpi": []any{
			map[string]any{"window_id": "notes-main", "operation": "resize", "before_width": 560, "before_height": 420, "after_width": 720, "after_height": 540, "dpi_scale_milli": 1250, "host_trace": true, "pass": true},
			map[string]any{"window_id": "notes-main", "operation": "dpi_scale", "before_width": 720, "before_height": 540, "after_width": 720, "after_height": 540, "dpi_scale_milli": 1250, "host_trace": true, "pass": true},
		},
		"cursor_transitions": []any{
			map[string]any{"window_id": "notes-main", "cursor": "pointer", "target": "NotesMainWindow", "host_trace": true, "pass": true},
			map[string]any{"window_id": "notes-main", "cursor": "text", "target": "NotesMainWindow", "host_trace": true, "pass": true},
			map[string]any{"window_id": "notes-main", "cursor": "resize", "target": "NotesMainWindow", "host_trace": true, "pass": true},
		},
		"clipboard":      map[string]any{"level": "clipboard-text-v1", "host_trace": true, "artifact_kind": "linux-app-shell-host-trace", "read": true, "write": true, "pass": true},
		"ime":            map[string]any{"level": "composition-baseline-v1", "host_trace": true, "artifact_kind": "linux-app-shell-host-trace", "start": true, "update": true, "commit": true, "cancel": true, "pass": true},
		"accessibility":  map[string]any{"level": "platform-bridge-v1", "host_trace": true, "artifact_kind": "linux-accessibility-platform-probe", "metadata_tree": true, "platform_export": true, "pass": true},
		"shell_features": p16LinuxAppShellFeaturesForTest(),
		"host_traces": []any{
			map[string]any{"name": "lifecycle", "artifact_kind": "linux-app-shell-host-trace", "path": "/tmp/surface-artifacts/surface-linux-app-shell-host-trace.json", "pass": true},
			map[string]any{"name": "windows", "artifact_kind": "linux-app-shell-window-trace", "path": "/tmp/surface-artifacts/surface-linux-app-shell-window-trace.json", "pass": true},
			map[string]any{"name": "accessibility", "artifact_kind": "linux-accessibility-platform-probe", "path": "/tmp/surface-artifacts/surface-linux-accessibility-probe.json", "pass": true},
		},
		"negative_guards": map[string]any{
			"no_gtk":              true,
			"no_qt":               true,
			"no_native_widgets":   true,
			"no_electron_runtime": true,
			"no_react_runtime":    true,
			"no_dom_ui":           true,
			"no_user_js":          true,
			"no_platform_widgets": true,
		},
	}
}
func p16LinuxAppShellFeaturesForTest() []any {
	return []any{
		map[string]any{"name": "app_menu", "status": "scoped_adapter", "claimed": true, "host_trace": true, "blocked_reason": "", "no_native_widget_ui": true, "pass": true},
		map[string]any{"name": "window_lifecycle", "status": "target_evidenced", "claimed": true, "host_trace": true, "blocked_reason": "", "no_native_widget_ui": true, "pass": true},
		map[string]any{"name": "multi_window", "status": "target_evidenced", "claimed": true, "host_trace": true, "blocked_reason": "", "no_native_widget_ui": true, "pass": true},
		map[string]any{"name": "clipboard", "status": "target_evidenced", "claimed": true, "host_trace": true, "blocked_reason": "", "no_native_widget_ui": true, "pass": true},
		map[string]any{"name": "ime", "status": "target_evidenced", "claimed": true, "host_trace": true, "blocked_reason": "", "no_native_widget_ui": true, "pass": true},
		map[string]any{"name": "accessibility_bridge", "status": "target_evidenced", "claimed": true, "host_trace": true, "blocked_reason": "", "no_native_widget_ui": true, "pass": true},
		map[string]any{"name": "crash_recovery", "status": "scoped_adapter", "claimed": true, "host_trace": true, "blocked_reason": "", "no_native_widget_ui": true, "pass": true},
		map[string]any{"name": "error_report", "status": "scoped_adapter", "claimed": true, "host_trace": true, "blocked_reason": "", "no_native_widget_ui": true, "pass": true},
		map[string]any{"name": "dialog", "status": "blocked_pass", "claimed": false, "host_trace": true, "blocked_reason": "target host dialog unavailable in CI", "no_native_widget_ui": true, "pass": true},
		map[string]any{"name": "file_dialog", "status": "blocked_pass", "claimed": false, "host_trace": true, "blocked_reason": "target host file dialog unavailable in CI", "no_native_widget_ui": true, "pass": true},
		map[string]any{"name": "file_picker", "status": "blocked_pass", "claimed": false, "host_trace": true, "blocked_reason": "target host file picker unavailable in CI", "no_native_widget_ui": true, "pass": true},
		map[string]any{"name": "notification", "status": "blocked_pass", "claimed": false, "host_trace": true, "blocked_reason": "target host notification unavailable in CI", "no_native_widget_ui": true, "pass": true},
		map[string]any{"name": "tray", "status": "blocked_pass", "claimed": false, "host_trace": true, "blocked_reason": "target host tray unavailable in CI", "no_native_widget_ui": true, "pass": true},
		map[string]any{"name": "deep_link", "status": "blocked_pass", "claimed": false, "host_trace": true, "blocked_reason": "target host deep link unavailable in CI", "no_native_widget_ui": true, "pass": true},
	}
}
func validSurfaceSecurityPermissionsMap(features []any) map[string]any {
	capabilities := make([]any, 0, len(features))
	for _, feature := range features {
		row := feature.(map[string]any)
		name := row["name"].(string)
		status, allowed := mapSecurityCapabilityStatus(row["status"].(string))
		blockedReason := ""
		if value, ok := row["blocked_reason"].(string); ok {
			blockedReason = value
		}
		capabilities = append(capabilities, map[string]any{
			"name":               name,
			"source_feature":     name,
			"status":             status,
			"allowed":            allowed,
			"capability_checked": true,
			"host_trace":         true,
			"policy":             "surface-app-shell-capability-policy-v1",
			"evidence":           "linux-app-shell-host-trace",
			"blocked_reason":     blockedReason,
			"pass":               true,
		})
	}
	return map[string]any{
		"schema":                        "tetra.surface.security-permission.v1",
		"model":                         "surface-security-permission-v1",
		"release_scope":                 "surface-v1-linux-web",
		"source":                        "examples/surface_linux_app_shell_notes.tetra",
		"app_shell_features":            "electron-feature-ledger-v1",
		"production_claim":              true,
		"experimental":                  false,
		"default_deny":                  true,
		"shell_feature_policy_enforced": true,
		"capabilities":                  capabilities,
		"permissions": []any{
			map[string]any{"name": "filesystem", "status": "denied", "allowed": false, "capability_checked": true, "blocked_reason": "ambient filesystem denied in default template", "evidence": "default-deny-policy", "pass": true},
			map[string]any{"name": "network", "status": "denied", "allowed": false, "capability_checked": true, "blocked_reason": "ambient network denied in default template", "evidence": "default-deny-policy", "pass": true},
			map[string]any{"name": "clipboard", "status": "allowed_with_policy", "allowed": true, "capability_checked": true, "blocked_reason": "", "evidence": "linux-app-shell-host-trace", "pass": true},
			map[string]any{"name": "notifications", "status": "denied", "allowed": false, "capability_checked": true, "blocked_reason": "notification target evidence absent", "evidence": "blocked-pass-nonclaim", "pass": true},
			map[string]any{"name": "dialogs", "status": "denied", "allowed": false, "capability_checked": true, "blocked_reason": "dialog target evidence absent", "evidence": "blocked-pass-nonclaim", "pass": true},
			map[string]any{"name": "shell_open_url", "status": "denied", "allowed": false, "capability_checked": true, "blocked_reason": "shell open-url denied in default template", "evidence": "default-deny-policy", "pass": true},
		},
		"process_boundaries": []any{
			map[string]any{"name": "surface_app_to_host_abi", "schema_checked": true, "capability_checked": true, "user_js": false, "node_integration": false, "electron_runtime": false, "pass": true},
			map[string]any{"name": "linux_app_shell_host_adapter", "schema_checked": true, "capability_checked": true, "user_js": false, "node_integration": false, "electron_runtime": false, "pass": true},
			map[string]any{"name": "browser_canvas_host", "schema_checked": true, "capability_checked": true, "user_js": false, "node_integration": false, "electron_runtime": false, "pass": true},
		},
		"asset_safety": []any{
			map[string]any{"kind": "font", "local_only": true, "sha256_required": true, "size_limit_bytes": 1048576, "network_fetch_allowed": false, "parser": "bounded-font-metadata-v1", "bounds_checked": true, "pass": true},
			map[string]any{"kind": "image", "local_only": true, "sha256_required": true, "size_limit_bytes": 2097152, "network_fetch_allowed": false, "parser": "bounded-image-header-v1", "bounds_checked": true, "pass": true},
			map[string]any{"kind": "icon", "local_only": true, "sha256_required": true, "size_limit_bytes": 262144, "network_fetch_allowed": false, "parser": "bounded-icon-header-v1", "bounds_checked": true, "pass": true},
		},
		"unsupported_claims": []any{
			"unrestricted-filesystem",
			"unrestricted-network",
			"native-permission-prompts",
			"production-notifications",
			"production-dialogs",
			"remote-asset-fetch",
			"electron-node-integration",
		},
		"negative_guards": map[string]any{
			"no_ambient_filesystem":                          true,
			"no_ambient_network":                             true,
			"no_shell_feature_bypass":                        true,
			"no_permissionless_clipboard":                    true,
			"no_notification_dialog_without_target_evidence": true,
			"no_network_asset_fetch":                         true,
			"no_untrusted_font_image_decode":                 true,
			"no_electron_node_integration":                   true,
			"no_user_js_app_logic":                           true,
			"no_dom_app_ui_tree":                             true,
		},
	}
}
func mapSecurityCapabilityStatus(featureStatus string) (string, bool) {
	switch featureStatus {
	case "target_evidenced", "scoped_adapter":
		return "allowed_with_policy", true
	case "blocked_pass":
		return "blocked_nonclaim", false
	default:
		return "unknown", false
	}
}
func withoutLinuxAppShellFeature(features []any, name string) []any {
	filtered := make([]any, 0, len(features))
	for _, feature := range features {
		row := feature.(map[string]any)
		if row["name"] == name {
			continue
		}
		filtered = append(filtered, feature)
	}
	return filtered
}
