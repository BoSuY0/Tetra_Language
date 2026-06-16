package surface

import (
	"strings"
	"testing"
)

func TestValidatePackageReportAcceptsCompletePackageStory(t *testing.T) {
	raw := validSurfacePackageReportJSON()
	if err := ValidatePackageReport([]byte(raw)); err != nil {
		t.Fatalf("ValidatePackageReport failed: %v\n%s", err, raw)
	}
}

func TestValidatePackageReportAcceptsFlagshipPackageStory(t *testing.T) {
	raw := validSurfacePackageReportJSON()
	raw = strings.Replace(raw, `"source":"examples/surface_reference_command_palette.tetra"`, `"source":"examples/surface_morph_rendered_studio_shell.tetra"`, 1)
	raw = strings.Replace(raw, `"reference_app":"command-palette"`, `"reference_app":"studio-shell"`, 1)
	if err := ValidatePackageReport([]byte(raw)); err != nil {
		t.Fatalf("ValidatePackageReport failed for flagship package story: %v\n%s", err, raw)
	}
}

func TestValidatePackageReportRejectsUnexpectedFlagshipInstallExitCode(t *testing.T) {
	raw := validSurfacePackageReportJSON()
	raw = strings.Replace(raw, `"source":"examples/surface_reference_command_palette.tetra"`, `"source":"examples/surface_morph_rendered_studio_shell.tetra"`, 1)
	raw = strings.Replace(raw, `"reference_app":"command-palette"`, `"reference_app":"studio-shell"`, 1)
	raw = strings.Replace(raw, `"exit_code":0`, `"exit_code":1`, 1)
	err := ValidatePackageReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected unexpected flagship install exit code to fail")
	}
	if !strings.Contains(err.Error(), "expected_exit_code") {
		t.Fatalf("error = %v, want expected_exit_code diagnostic", err)
	}
}

func TestValidatePackageReportRejectsNonzeroExpectedExitForReferenceApp(t *testing.T) {
	raw := strings.Replace(validSurfacePackageReportJSON(), `"exit_code":0`, `"exit_code":1,"expected_exit_code":1`, 1)
	err := ValidatePackageReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected nonzero expected exit code for reference app to fail")
	}
	if !strings.Contains(err.Error(), "want 0") {
		t.Fatalf("error = %v, want zero-exit diagnostic", err)
	}
}

func TestValidatePackageReportRejectsSigningClaimWithoutEvidence(t *testing.T) {
	raw := strings.Replace(validSurfacePackageReportJSON(), `"signed":false`, `"signed":true`, 1)
	err := ValidatePackageReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected signed package claim to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "signed") {
		t.Fatalf("error = %v, want signed diagnostic", err)
	}
}

func TestValidatePackageReportRejectsMissingLinuxInstallRun(t *testing.T) {
	raw := strings.Replace(validSurfacePackageReportJSON(), `"install_smokes":[{"target":"linux-x64","package_path":"surface-packages/surface-command-palette-linux-x64.tar.gz","install_dir":"surface-install/linux-x64","installed_binary":"surface-install/linux-x64/bin/surface-command-palette","command":"surface-install/linux-x64/bin/surface-command-palette","exit_code":0,"artifact_hash_verified":true,"package_manifest_verified":true,"app_run":true,"pass":true}]`, `"install_smokes":[]`, 1)
	err := ValidatePackageReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing linux install/run smoke to fail")
	}
	if !strings.Contains(err.Error(), "install_smokes") {
		t.Fatalf("error = %v, want install_smokes diagnostic", err)
	}
}

func TestValidatePackageReportRejectsRemoteAssetFetch(t *testing.T) {
	raw := strings.Replace(validSurfacePackageReportJSON(), `"network_fetch_allowed":false`, `"network_fetch_allowed":true`, 1)
	err := ValidatePackageReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected remote asset fetch to fail")
	}
	if !strings.Contains(err.Error(), "network_fetch_allowed") {
		t.Fatalf("error = %v, want network_fetch_allowed diagnostic", err)
	}
}

func TestValidatePackageReportRejectsAutoUpdateClaimWithoutRuntimeEvidence(t *testing.T) {
	raw := strings.Replace(validSurfacePackageReportJSON(), `"auto_update_runtime_claim":false`, `"auto_update_runtime_claim":true`, 1)
	err := ValidatePackageReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected auto-update runtime claim to fail")
	}
	if !strings.Contains(err.Error(), "auto_update_runtime_claim") {
		t.Fatalf("error = %v, want auto_update_runtime_claim diagnostic", err)
	}
}

func validSurfacePackageReportJSON() string {
	return `{"schema":"tetra.surface.package.v1","model":"surface-package-v1","release_scope":"surface-v1-linux-web","producer":"scripts/release/surface/surface-package-smoke.sh","source":"examples/surface_reference_command_palette.tetra","reference_app":"command-palette","package_format":"surface-app-package-v1","format_version":1,"artifact_root":"surface-package-work","packages":[{"target":"linux-x64","kind":"linux-x64-tar.gz","path":"surface-packages/surface-command-palette-linux-x64.tar.gz","manifest_path":"surface-package-work/linux-x64/package-manifest.json","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","asset_manifest_sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","source_sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","build_sha256":"sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd","contains_executable":true,"contains_web_bundle":false,"local_only_assets":true,"pass":true},{"target":"wasm32-web","kind":"wasm32-web-tar.gz","path":"surface-packages/surface-command-palette-wasm32-web.tar.gz","manifest_path":"surface-package-work/wasm32-web/package-manifest.json","sha256":"sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee","asset_manifest_sha256":"sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","source_sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","build_sha256":"sha256:1111111111111111111111111111111111111111111111111111111111111111","contains_executable":false,"contains_web_bundle":true,"local_only_assets":true,"pass":true}],"assets":[{"path":"surface-package-work/assets/app-icon.txt","kind":"icon","sha256":"sha256:2222222222222222222222222222222222222222222222222222222222222222","size_bytes":32,"local_only":true,"network_fetch_allowed":false,"pass":true},{"path":"surface-package-work/assets/theme-manifest.json","kind":"theme","sha256":"sha256:3333333333333333333333333333333333333333333333333333333333333333","size_bytes":64,"local_only":true,"network_fetch_allowed":false,"pass":true}],"install_smokes":[{"target":"linux-x64","package_path":"surface-packages/surface-command-palette-linux-x64.tar.gz","install_dir":"surface-install/linux-x64","installed_binary":"surface-install/linux-x64/bin/surface-command-palette","command":"surface-install/linux-x64/bin/surface-command-palette","exit_code":0,"artifact_hash_verified":true,"package_manifest_verified":true,"app_run":true,"pass":true}],"web_bundles":[{"target":"wasm32-web","package_path":"surface-packages/surface-command-palette-wasm32-web.tar.gz","web_entry":"surface-package-work/wasm32-web/index.html","wasm_artifact":"surface-package-work/wasm32-web/surface-command-palette.wasm","loader_artifact":"surface-package-work/wasm32-web/surface-command-palette.mjs","browser_canvas_host":"surface-package-work/wasm32-web/surface-browser-canvas-host.mjs","command":"tetra build --target wasm32-web","artifact_hash_verified":true,"package_manifest_verified":true,"pass":true}],"update_strategy":{"strategy":"hash-pinned-channel-manifest-v1","manifest_format":"tetra.surface.update-channel.v1","channel_manifest":"surface-updates/channel.json","current_version":"p23.0.0","latest_version":"p23.0.0","latest_package_path":"surface-packages/surface-command-palette-linux-x64.tar.gz","latest_package_sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","package_hash_pinned":true,"rollback_manifest":"surface-updates/rollback.json","signature_required_for_stable_promotion":true,"auto_update_runtime_claim":false,"network_update_claim":false,"pass":true},"signing":{"status":"nonclaim","signed":false,"notarized":false,"production_claim":false,"evidence":"","blocked_reason":"platform signing keys and CI signing evidence are not present in this release"},"notarization":{"status":"nonclaim","signed":false,"notarized":false,"production_claim":false,"evidence":"","blocked_reason":"macOS notarization evidence is unavailable because macOS Surface target host is unsupported"},"negative_guards":{"no_react_runtime":true,"no_electron_runtime":true,"no_dom_app_ui_tree":true,"no_css_runtime":true,"no_user_js_app_logic":true,"no_remote_asset_fetch":true,"no_unsigned_signing_claim":true,"no_notarization_without_platform_evidence":true,"no_auto_update_without_runtime_evidence":true,"no_docs_only_package_claim":true,"install_run_required":true,"web_bundle_required":true,"artifact_hashes_required":true},"pass":true}` + "\n"
}
