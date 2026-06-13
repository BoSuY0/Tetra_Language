package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunValidatesSurfaceSecurityPermissionReport(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "surface-linux-x64-release-app-shell.json")
	if err := os.WriteFile(reportPath, []byte(validSecurityPermissionRuntimeReportJSON()), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}

	if err := run([]string{"--report", reportPath}); err != nil {
		t.Fatalf("run failed: %v", err)
	}
}

func TestRunRejectsNetworkAllowedByDefault(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "surface-linux-x64-release-app-shell.json")
	raw := strings.Replace(validSecurityPermissionRuntimeReportJSON(), `"name":"network","status":"denied","allowed":false`, `"name":"network","status":"allowed_with_policy","allowed":true`, 1)
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}

	err := run([]string{"--report", reportPath})
	if err == nil {
		t.Fatalf("expected network default allow to fail")
	}
	if !strings.Contains(err.Error(), "network") {
		t.Fatalf("error = %v, want network diagnostic", err)
	}
}

func validSecurityPermissionRuntimeReportJSON() string {
	return `{
  "schema": "tetra.surface.runtime.v1",
  "status": "pass",
  "target": "linux-x64",
  "source": "examples/surface_linux_app_shell_notes.tetra",
  "linux_app_shell": {
    "schema": "tetra.surface.linux-app-shell.v1",
    "app_shell_level": "linux-app-shell-subset-v1",
    "release_scope": "surface-v1-linux-web",
    "source": "examples/surface_linux_app_shell_notes.tetra",
    "module": "lib.core.surface_app_shell",
    "host_adapter": "wayland-shm-rgba-release-v1",
    "production_claim": true,
    "experimental": false,
    "shell_features": [
      {"name":"window_lifecycle","status":"target_evidenced","claimed":true,"host_trace":true,"blocked_reason":"","no_native_widget_ui":true,"pass":true},
      {"name":"multi_window","status":"target_evidenced","claimed":true,"host_trace":true,"blocked_reason":"","no_native_widget_ui":true,"pass":true},
      {"name":"clipboard","status":"target_evidenced","claimed":true,"host_trace":true,"blocked_reason":"","no_native_widget_ui":true,"pass":true},
      {"name":"ime","status":"target_evidenced","claimed":true,"host_trace":true,"blocked_reason":"","no_native_widget_ui":true,"pass":true},
      {"name":"accessibility_bridge","status":"target_evidenced","claimed":true,"host_trace":true,"blocked_reason":"","no_native_widget_ui":true,"pass":true},
      {"name":"app_menu","status":"scoped_adapter","claimed":true,"host_trace":true,"blocked_reason":"","no_native_widget_ui":true,"pass":true},
      {"name":"crash_recovery","status":"scoped_adapter","claimed":true,"host_trace":true,"blocked_reason":"","no_native_widget_ui":true,"pass":true},
      {"name":"error_report","status":"scoped_adapter","claimed":true,"host_trace":true,"blocked_reason":"","no_native_widget_ui":true,"pass":true},
      {"name":"dialog","status":"blocked_pass","claimed":false,"host_trace":true,"blocked_reason":"target host dialog unavailable in CI","no_native_widget_ui":true,"pass":true},
      {"name":"file_dialog","status":"blocked_pass","claimed":false,"host_trace":true,"blocked_reason":"target host file dialog unavailable in CI","no_native_widget_ui":true,"pass":true},
      {"name":"file_picker","status":"blocked_pass","claimed":false,"host_trace":true,"blocked_reason":"target host file picker unavailable in CI","no_native_widget_ui":true,"pass":true},
      {"name":"notification","status":"blocked_pass","claimed":false,"host_trace":true,"blocked_reason":"target host notification unavailable in CI","no_native_widget_ui":true,"pass":true},
      {"name":"tray","status":"blocked_pass","claimed":false,"host_trace":true,"blocked_reason":"target host tray unavailable in CI","no_native_widget_ui":true,"pass":true},
      {"name":"deep_link","status":"blocked_pass","claimed":false,"host_trace":true,"blocked_reason":"target host deep link unavailable in CI","no_native_widget_ui":true,"pass":true}
    ]
  },
  "security_permissions": {
    "schema":"tetra.surface.security-permission.v1",
    "model":"surface-security-permission-v1",
    "release_scope":"surface-v1-linux-web",
    "source":"examples/surface_linux_app_shell_notes.tetra",
    "app_shell_features":"electron-feature-ledger-v1",
    "production_claim":true,
    "experimental":false,
    "default_deny":true,
    "shell_feature_policy_enforced":true,
    "capabilities":[
      {"name":"window_lifecycle","source_feature":"window_lifecycle","status":"allowed_with_policy","allowed":true,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"","pass":true},
      {"name":"multi_window","source_feature":"multi_window","status":"allowed_with_policy","allowed":true,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"","pass":true},
      {"name":"clipboard","source_feature":"clipboard","status":"allowed_with_policy","allowed":true,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"","pass":true},
      {"name":"ime","source_feature":"ime","status":"allowed_with_policy","allowed":true,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"","pass":true},
      {"name":"accessibility_bridge","source_feature":"accessibility_bridge","status":"allowed_with_policy","allowed":true,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"","pass":true},
      {"name":"app_menu","source_feature":"app_menu","status":"allowed_with_policy","allowed":true,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"","pass":true},
      {"name":"crash_recovery","source_feature":"crash_recovery","status":"allowed_with_policy","allowed":true,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"","pass":true},
      {"name":"error_report","source_feature":"error_report","status":"allowed_with_policy","allowed":true,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"","pass":true},
      {"name":"dialog","source_feature":"dialog","status":"blocked_nonclaim","allowed":false,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"target host dialog unavailable in CI","pass":true},
      {"name":"file_dialog","source_feature":"file_dialog","status":"blocked_nonclaim","allowed":false,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"target host file dialog unavailable in CI","pass":true},
      {"name":"file_picker","source_feature":"file_picker","status":"blocked_nonclaim","allowed":false,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"target host file picker unavailable in CI","pass":true},
      {"name":"notification","source_feature":"notification","status":"blocked_nonclaim","allowed":false,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"target host notification unavailable in CI","pass":true},
      {"name":"tray","source_feature":"tray","status":"blocked_nonclaim","allowed":false,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"target host tray unavailable in CI","pass":true},
      {"name":"deep_link","source_feature":"deep_link","status":"blocked_nonclaim","allowed":false,"capability_checked":true,"host_trace":true,"policy":"surface-app-shell-capability-policy-v1","evidence":"linux-app-shell-host-trace","blocked_reason":"target host deep link unavailable in CI","pass":true}
    ],
    "permissions":[
      {"name":"filesystem","status":"denied","allowed":false,"capability_checked":true,"blocked_reason":"ambient filesystem denied in default template","evidence":"default-deny-policy","pass":true},
      {"name":"network","status":"denied","allowed":false,"capability_checked":true,"blocked_reason":"ambient network denied in default template","evidence":"default-deny-policy","pass":true},
      {"name":"clipboard","status":"allowed_with_policy","allowed":true,"capability_checked":true,"blocked_reason":"","evidence":"linux-app-shell-host-trace","pass":true},
      {"name":"notifications","status":"denied","allowed":false,"capability_checked":true,"blocked_reason":"notification target evidence absent","evidence":"blocked-pass-nonclaim","pass":true},
      {"name":"dialogs","status":"denied","allowed":false,"capability_checked":true,"blocked_reason":"dialog target evidence absent","evidence":"blocked-pass-nonclaim","pass":true},
      {"name":"shell_open_url","status":"denied","allowed":false,"capability_checked":true,"blocked_reason":"shell open-url denied in default template","evidence":"default-deny-policy","pass":true}
    ],
    "process_boundaries":[
      {"name":"surface_app_to_host_abi","schema_checked":true,"capability_checked":true,"user_js":false,"node_integration":false,"electron_runtime":false,"pass":true},
      {"name":"linux_app_shell_host_adapter","schema_checked":true,"capability_checked":true,"user_js":false,"node_integration":false,"electron_runtime":false,"pass":true},
      {"name":"browser_canvas_host","schema_checked":true,"capability_checked":true,"user_js":false,"node_integration":false,"electron_runtime":false,"pass":true}
    ],
    "asset_safety":[
      {"kind":"font","local_only":true,"sha256_required":true,"size_limit_bytes":1048576,"network_fetch_allowed":false,"parser":"bounded-font-metadata-v1","bounds_checked":true,"pass":true},
      {"kind":"image","local_only":true,"sha256_required":true,"size_limit_bytes":2097152,"network_fetch_allowed":false,"parser":"bounded-image-header-v1","bounds_checked":true,"pass":true},
      {"kind":"icon","local_only":true,"sha256_required":true,"size_limit_bytes":262144,"network_fetch_allowed":false,"parser":"bounded-icon-header-v1","bounds_checked":true,"pass":true}
    ],
    "unsupported_claims":["unrestricted-filesystem","unrestricted-network","native-permission-prompts","production-notifications","production-dialogs","remote-asset-fetch","electron-node-integration"],
    "negative_guards":{"no_ambient_filesystem":true,"no_ambient_network":true,"no_shell_feature_bypass":true,"no_permissionless_clipboard":true,"no_notification_dialog_without_target_evidence":true,"no_network_asset_fetch":true,"no_untrusted_font_image_decode":true,"no_electron_node_integration":true,"no_user_js_app_logic":true,"no_dom_app_ui_tree":true}
  }
}`
}
