package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surface"
)

func TestRunValidatesSurfaceSecurityPermissionReport(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "surface-linux-x64-release-app-shell.json")
	if err := os.WriteFile(
		reportPath,
		securityPermissionRuntimeReportJSON(nil),
		0o644,
	); err != nil {
		t.Fatalf("write report: %v", err)
	}

	if err := run([]string{"--report", reportPath}); err != nil {
		t.Fatalf("run failed: %v", err)
	}
}

func TestRunRejectsNetworkAllowedByDefault(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "surface-linux-x64-release-app-shell.json")
	raw := securityPermissionRuntimeReportJSON(func(report *surface.Report) {
		for i := range report.SecurityPermissions.Permissions {
			permission := &report.SecurityPermissions.Permissions[i]
			if permission.Name != "network" {
				continue
			}
			permission.Status = "allowed_with_policy"
			permission.Allowed = true
			permission.BlockedReason = ""
		}
	})
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
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

const (
	securitySource       = "examples/surface/toolkit/surface_linux_app_shell_notes.tetra"
	securityPolicy       = "surface-app-shell-capability-policy-v1"
	securityHostEvidence = "linux-app-shell-host-trace"
)

func securityPermissionRuntimeReportJSON(mutate func(*surface.Report)) []byte {
	report := validSecurityPermissionRuntimeReport()
	if mutate != nil {
		mutate(&report)
	}

	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		panic(err)
	}
	return append(raw, '\n')
}

func validSecurityPermissionRuntimeReport() surface.Report {
	return surface.Report{
		Schema: "tetra.surface.runtime.v1",
		Status: "pass",
		Target: "linux-x64",
		Source: securitySource,
		LinuxAppShell: &surface.LinuxAppShellReport{
			Schema:          "tetra.surface.linux-app-shell.v1",
			AppShellLevel:   "linux-app-shell-subset-v1",
			ReleaseScope:    "surface-v1-linux-web",
			Source:          securitySource,
			Module:          "lib.core.surface_app_shell",
			HostAdapter:     "wayland-shm-rgba-release-v1",
			ProductionClaim: true,
			Experimental:    false,
			ShellFeatures:   securityShellFeatures(),
		},
		SecurityPermissions: &surface.SecurityPermissionReport{
			Schema:                     "tetra.surface.security-permission.v1",
			Model:                      "surface-security-permission-v1",
			ReleaseScope:               "surface-v1-linux-web",
			Source:                     securitySource,
			AppShellFeatures:           "electron-feature-ledger-v1",
			ProductionClaim:            true,
			Experimental:               false,
			DefaultDeny:                true,
			ShellFeaturePolicyEnforced: true,
			Capabilities:               securityCapabilities(),
			Permissions:                securityPermissions(),
			ProcessBoundaries:          securityProcessBoundaries(),
			AssetSafety:                securityAssetSafety(),
			UnsupportedClaims:          securityUnsupportedClaims(),
			NegativeGuards:             securityNegativeGuards(),
		},
	}
}

func securityShellFeatures() []surface.LinuxAppShellFeatureReport {
	return []surface.LinuxAppShellFeatureReport{
		shellFeature("window_lifecycle", "target_evidenced", true, ""),
		shellFeature("multi_window", "target_evidenced", true, ""),
		shellFeature("clipboard", "target_evidenced", true, ""),
		shellFeature("ime", "target_evidenced", true, ""),
		shellFeature("accessibility_bridge", "target_evidenced", true, ""),
		shellFeature("app_menu", "scoped_adapter", true, ""),
		shellFeature("crash_recovery", "scoped_adapter", true, ""),
		shellFeature("error_report", "scoped_adapter", true, ""),
		shellFeature("dialog", "blocked_pass", false, "target host dialog unavailable in CI"),
		shellFeature(
			"file_dialog",
			"blocked_pass",
			false,
			"target host file dialog unavailable in CI",
		),
		shellFeature(
			"file_picker",
			"blocked_pass",
			false,
			"target host file picker unavailable in CI",
		),
		shellFeature(
			"notification",
			"blocked_pass",
			false,
			"target host notification unavailable in CI",
		),
		shellFeature("tray", "blocked_pass", false, "target host tray unavailable in CI"),
		shellFeature(
			"deep_link",
			"blocked_pass",
			false,
			"target host deep link unavailable in CI",
		),
	}
}

func shellFeature(
	name string,
	status string,
	claimed bool,
	blockedReason string,
) surface.LinuxAppShellFeatureReport {
	return surface.LinuxAppShellFeatureReport{
		Name:             name,
		Status:           status,
		Claimed:          claimed,
		HostTrace:        true,
		BlockedReason:    blockedReason,
		NoNativeWidgetUI: true,
		Pass:             true,
	}
}

func securityCapabilities() []surface.SurfaceSecurityCapabilityReport {
	return []surface.SurfaceSecurityCapabilityReport{
		allowedSecurityCapability("window_lifecycle"),
		allowedSecurityCapability("multi_window"),
		allowedSecurityCapability("clipboard"),
		allowedSecurityCapability("ime"),
		allowedSecurityCapability("accessibility_bridge"),
		allowedSecurityCapability("app_menu"),
		allowedSecurityCapability("crash_recovery"),
		allowedSecurityCapability("error_report"),
		blockedSecurityCapability("dialog", "target host dialog unavailable in CI"),
		blockedSecurityCapability("file_dialog", "target host file dialog unavailable in CI"),
		blockedSecurityCapability("file_picker", "target host file picker unavailable in CI"),
		blockedSecurityCapability("notification", "target host notification unavailable in CI"),
		blockedSecurityCapability("tray", "target host tray unavailable in CI"),
		blockedSecurityCapability("deep_link", "target host deep link unavailable in CI"),
	}
}

func allowedSecurityCapability(name string) surface.SurfaceSecurityCapabilityReport {
	return securityCapability(name, "allowed_with_policy", true, "")
}

func blockedSecurityCapability(
	name string,
	blockedReason string,
) surface.SurfaceSecurityCapabilityReport {
	return securityCapability(name, "blocked_nonclaim", false, blockedReason)
}

func securityCapability(
	name string,
	status string,
	allowed bool,
	blockedReason string,
) surface.SurfaceSecurityCapabilityReport {
	return surface.SurfaceSecurityCapabilityReport{
		Name:              name,
		SourceFeature:     name,
		Status:            status,
		Allowed:           allowed,
		CapabilityChecked: true,
		HostTrace:         true,
		Policy:            securityPolicy,
		Evidence:          securityHostEvidence,
		BlockedReason:     blockedReason,
		Pass:              true,
	}
}

func securityPermissions() []surface.SurfacePermissionReport {
	return []surface.SurfacePermissionReport{
		deniedSecurityPermission(
			"filesystem",
			"ambient filesystem denied in default template",
			"default-deny-policy",
		),
		deniedSecurityPermission(
			"network",
			"ambient network denied in default template",
			"default-deny-policy",
		),
		allowedSecurityPermission("clipboard", securityHostEvidence),
		deniedSecurityPermission(
			"notifications",
			"notification target evidence absent",
			"blocked-pass-nonclaim",
		),
		deniedSecurityPermission(
			"dialogs",
			"dialog target evidence absent",
			"blocked-pass-nonclaim",
		),
		deniedSecurityPermission(
			"shell_open_url",
			"shell open-url denied in default template",
			"default-deny-policy",
		),
	}
}

func allowedSecurityPermission(name string, evidence string) surface.SurfacePermissionReport {
	return surface.SurfacePermissionReport{
		Name:              name,
		Status:            "allowed_with_policy",
		Allowed:           true,
		CapabilityChecked: true,
		Evidence:          evidence,
		Pass:              true,
	}
}

func deniedSecurityPermission(
	name string,
	blockedReason string,
	evidence string,
) surface.SurfacePermissionReport {
	return surface.SurfacePermissionReport{
		Name:              name,
		Status:            "denied",
		Allowed:           false,
		CapabilityChecked: true,
		BlockedReason:     blockedReason,
		Evidence:          evidence,
		Pass:              true,
	}
}

func securityProcessBoundaries() []surface.SurfaceProcessBoundaryReport {
	return []surface.SurfaceProcessBoundaryReport{
		securityProcessBoundary("surface_app_to_host_abi"),
		securityProcessBoundary("linux_app_shell_host_adapter"),
		securityProcessBoundary("browser_canvas_host"),
	}
}

func securityProcessBoundary(name string) surface.SurfaceProcessBoundaryReport {
	return surface.SurfaceProcessBoundaryReport{
		Name:              name,
		SchemaChecked:     true,
		CapabilityChecked: true,
		UserJS:            false,
		NodeIntegration:   false,
		ElectronRuntime:   false,
		Pass:              true,
	}
}

func securityAssetSafety() []surface.SurfaceAssetSafetyReport {
	return []surface.SurfaceAssetSafetyReport{
		securityAssetSafetyItem("font", 1048576, "bounded-font-metadata-v1"),
		securityAssetSafetyItem("image", 2097152, "bounded-image-header-v1"),
		securityAssetSafetyItem("icon", 262144, "bounded-icon-header-v1"),
	}
}

func securityAssetSafetyItem(
	kind string,
	sizeLimitBytes int,
	parser string,
) surface.SurfaceAssetSafetyReport {
	return surface.SurfaceAssetSafetyReport{
		Kind:                kind,
		LocalOnly:           true,
		SHA256Required:      true,
		SizeLimitBytes:      sizeLimitBytes,
		NetworkFetchAllowed: false,
		Parser:              parser,
		BoundsChecked:       true,
		Pass:                true,
	}
}

func securityUnsupportedClaims() []string {
	return []string{
		"unrestricted-filesystem",
		"unrestricted-network",
		"native-permission-prompts",
		"production-notifications",
		"production-dialogs",
		"remote-asset-fetch",
		"electron-node-integration",
	}
}

func securityNegativeGuards() surface.SurfaceSecurityNegativeGuards {
	return surface.SurfaceSecurityNegativeGuards{
		NoAmbientFilesystem:                       true,
		NoAmbientNetwork:                          true,
		NoShellFeatureBypass:                      true,
		NoPermissionlessClipboard:                 true,
		NoNotificationDialogWithoutTargetEvidence: true,
		NoNetworkAssetFetch:                       true,
		NoUntrustedFontImageDecode:                true,
		NoElectronNodeIntegration:                 true,
		NoUserJSAppLogic:                          true,
		NoDOMAppUITree:                            true,
	}
}
