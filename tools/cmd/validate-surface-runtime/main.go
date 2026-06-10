package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tetra_language/tools/validators/surface"
)

func main() {
	reportPath := flag.String("report", "", "path to tetra.surface.runtime.v1 or Surface release JSON report")
	release := flag.String("release", "", "optional strict release validation mode")
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateSurfaceRuntimeReportWithOptions(*reportPath, surfaceRuntimeValidationOptions{Release: *release}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type surfaceRuntimeValidationOptions struct {
	Release string
}

func validateSurfaceRuntimeReport(path string) error {
	return validateSurfaceRuntimeReportWithOptions(path, surfaceRuntimeValidationOptions{})
}

func validateSurfaceRuntimeReportWithOptions(path string, opt surfaceRuntimeValidationOptions) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	release := strings.TrimSpace(opt.Release)
	if release != "" &&
		release != "surface-v1" &&
		release != "headless" &&
		release != "linux-x64-real-window" &&
		release != "wasm32-web-browser" &&
		release != "text-input" {
		return fmt.Errorf("unsupported release %q", release)
	}
	schema, err := surfaceReportSchema(raw)
	if err != nil {
		return err
	}
	var validationErr error
	switch schema {
	case surface.ReleaseSchemaV1:
		validationErr = surface.ValidateReleaseSummary(raw)
	case surface.TextInputSchemaV1:
		validationErr = surface.ValidateTextInputReport(raw)
	case surface.SchemaV1:
		validationErr = validateRuntimeReportWithArtifacts(path, raw)
	default:
		return fmt.Errorf("schema is %q, want %q, %q, or %q", schema, surface.SchemaV1, surface.ReleaseSchemaV1, surface.TextInputSchemaV1)
	}
	if validationErr != nil {
		return validationErr
	}
	if release == "surface-v1" {
		return validateSurfaceV1ReleaseEnvelope(schema, raw)
	}
	if release == "headless" {
		return validateHeadlessReleaseEnvelope(schema, raw)
	}
	if release == "linux-x64-real-window" {
		return validateLinuxX64RealWindowReleaseEnvelope(schema, raw)
	}
	if release == "wasm32-web-browser" {
		return validateWASM32WebBrowserReleaseEnvelope(schema, raw)
	}
	if release == "text-input" {
		return validateTextInputReleaseEnvelope(schema, raw)
	}
	return nil
}

func surfaceReportSchema(raw []byte) (string, error) {
	var envelope struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return "", err
	}
	schema := strings.TrimSpace(envelope.Schema)
	if schema == "" {
		return "", errors.New("schema is required")
	}
	return schema, nil
}

func validateRuntimeReportWithArtifacts(path string, raw []byte) error {
	if err := surface.ValidateReport(raw); err != nil {
		return err
	}
	var report surface.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		return err
	}
	reportDir := filepath.Dir(path)
	var issues []string
	if err := validateArtifactIntegrity(reportDir, report); err != nil {
		issues = append(issues, err.Error())
	}
	if err := validateArtifactScanIntegrity(reportDir, report); err != nil {
		issues = append(issues, err.Error())
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateSurfaceV1ReleaseEnvelope(schema string, raw []byte) error {
	switch schema {
	case surface.ReleaseSchemaV1:
		return nil
	case surface.TextInputSchemaV1:
		var report surface.TextInputReport
		if err := json.Unmarshal(raw, &report); err != nil {
			return err
		}
		if report.Level != "production-text-input-v1" || report.Experimental || !report.ProductionClaim {
			return fmt.Errorf("release surface-v1 text-input report requires production-text-input-v1, experimental=false, and production_claim=true")
		}
		return nil
	case surface.SchemaV1:
		var report surface.Report
		if err := json.Unmarshal(raw, &report); err != nil {
			return err
		}
		return validateSurfaceV1RuntimeReleaseReport(report)
	default:
		return fmt.Errorf("release surface-v1 does not accept schema %q", schema)
	}
}

func validateHeadlessReleaseEnvelope(schema string, raw []byte) error {
	if schema != surface.SchemaV1 {
		return fmt.Errorf("release headless schema is %q, want %q", schema, surface.SchemaV1)
	}
	var report surface.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		return err
	}
	var issues []string
	if report.Target != "headless" {
		issues = append(issues, fmt.Sprintf("release headless target is %q, want headless", report.Target))
	}
	if report.Runtime != "surface-headless" {
		issues = append(issues, fmt.Sprintf("release headless runtime is %q, want surface-headless", report.Runtime))
	}
	if report.HostEvidence.Level != "deterministic-headless" {
		issues = append(issues, fmt.Sprintf("release headless host_evidence.level is %q, want deterministic-headless", report.HostEvidence.Level))
	}
	if report.HostEvidence.Backend != "software-rgba" {
		issues = append(issues, fmt.Sprintf("release headless host_evidence.backend is %q, want software-rgba", report.HostEvidence.Backend))
	}
	if !report.HostEvidence.Framebuffer {
		issues = append(issues, "release headless host_evidence.framebuffer must be true")
	}
	if report.HostEvidence.RealWindow || report.HostEvidence.NativeInput || report.HostEvidence.UserFacingPlatformWidgets {
		issues = append(issues, "release headless must not claim real window, native input, or platform widgets")
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateLinuxX64RealWindowReleaseEnvelope(schema string, raw []byte) error {
	if schema != surface.SchemaV1 {
		return fmt.Errorf("release linux-x64-real-window schema is %q, want %q", schema, surface.SchemaV1)
	}
	var report surface.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		return err
	}
	var issues []string
	if report.Target != "linux-x64" {
		issues = append(issues, fmt.Sprintf("release linux-x64-real-window target is %q, want linux-x64", report.Target))
	}
	if report.Runtime != "surface-linux-x64" {
		issues = append(issues, fmt.Sprintf("release linux-x64-real-window runtime is %q, want surface-linux-x64", report.Runtime))
	}
	if report.HostEvidence.Level != "linux-x64-release-window-v1" {
		issues = append(issues, fmt.Sprintf("release linux-x64-real-window host_evidence.level is %q, want linux-x64-release-window-v1", report.HostEvidence.Level))
	}
	if report.HostEvidence.Backend != "wayland-shm-rgba-release-v1" {
		issues = append(issues, fmt.Sprintf("release linux-x64-real-window host_evidence.backend is %q, want wayland-shm-rgba-release-v1", report.HostEvidence.Backend))
	}
	for _, check := range []struct {
		name string
		ok   bool
	}{
		{name: "framebuffer", ok: report.HostEvidence.Framebuffer},
		{name: "real_window", ok: report.HostEvidence.RealWindow},
		{name: "native_input", ok: report.HostEvidence.NativeInput},
		{name: "text_input", ok: report.HostEvidence.TextInput},
		{name: "clipboard", ok: report.HostEvidence.Clipboard},
		{name: "composition", ok: report.HostEvidence.Composition},
		{name: "accessibility_bridge", ok: report.HostEvidence.AccessibilityBridge},
	} {
		if !check.ok {
			issues = append(issues, fmt.Sprintf("release linux-x64-real-window host_evidence.%s must be true", check.name))
		}
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateWASM32WebBrowserReleaseEnvelope(schema string, raw []byte) error {
	if schema != surface.SchemaV1 {
		return fmt.Errorf("release wasm32-web-browser schema is %q, want %q", schema, surface.SchemaV1)
	}
	var report surface.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		return err
	}
	var issues []string
	if report.Target != "wasm32-web" {
		issues = append(issues, fmt.Sprintf("release wasm32-web-browser target is %q, want wasm32-web", report.Target))
	}
	if report.Runtime != "surface-wasm32-web" {
		issues = append(issues, fmt.Sprintf("release wasm32-web-browser runtime is %q, want surface-wasm32-web", report.Runtime))
	}
	if !isSurfaceReleaseFormSource(report.Source) {
		issues = append(issues, fmt.Sprintf("release wasm32-web-browser source is %q, want examples/surface_release_form.tetra", report.Source))
	}
	if report.HostEvidence.Level != "wasm32-web-browser-canvas-release-v1" {
		issues = append(issues, fmt.Sprintf("release wasm32-web-browser host_evidence.level is %q, want wasm32-web-browser-canvas-release-v1", report.HostEvidence.Level))
	}
	if report.HostEvidence.Backend != "browser-canvas-rgba-accessible" {
		issues = append(issues, fmt.Sprintf("release wasm32-web-browser host_evidence.backend is %q, want browser-canvas-rgba-accessible", report.HostEvidence.Backend))
	}
	for _, check := range []struct {
		name string
		ok   bool
	}{
		{name: "framebuffer", ok: report.HostEvidence.Framebuffer},
		{name: "native_input", ok: report.HostEvidence.NativeInput},
		{name: "browser_canvas", ok: report.HostEvidence.BrowserCanvas},
		{name: "browser_input", ok: report.HostEvidence.BrowserInput},
		{name: "browser_clipboard", ok: report.HostEvidence.BrowserClipboard},
		{name: "browser_composition", ok: report.HostEvidence.BrowserComposition},
		{name: "browser_accessibility_snapshot", ok: report.HostEvidence.BrowserAccessibilitySnapshot},
		{name: "browser_accessibility_mirror", ok: report.HostEvidence.BrowserAccessibilityMirror},
	} {
		if !check.ok {
			issues = append(issues, fmt.Sprintf("release wasm32-web-browser host_evidence.%s must be true", check.name))
		}
	}
	if report.HostEvidence.RealWindow {
		issues = append(issues, "release wasm32-web-browser must not claim OS real_window")
	}
	if report.HostEvidence.BrowserClipboardHarness != "deterministic-browser-clipboard-v1" {
		issues = append(issues, fmt.Sprintf("release wasm32-web-browser browser_clipboard_harness is %q, want deterministic-browser-clipboard-v1", report.HostEvidence.BrowserClipboardHarness))
	}
	if !runtimeReportHasProcessNameAndPathMarkers(report.Processes, "app", "surface wasm32-web browser canvas component app", "chromium") &&
		!runtimeReportHasProcessNameAndPathMarkers(report.Processes, "app", "surface wasm32-web browser canvas component app", "chrome") {
		issues = append(issues, "release wasm32-web-browser requires Chromium-compatible browser app process evidence")
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateTextInputReleaseEnvelope(schema string, raw []byte) error {
	if schema != surface.TextInputSchemaV1 {
		return fmt.Errorf("release text-input schema is %q, want %q", schema, surface.TextInputSchemaV1)
	}
	var report surface.TextInputReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return err
	}
	var issues []string
	if normalizeEvidencePath(report.Source) != "examples/surface_release_text_input.tetra" {
		issues = append(issues, fmt.Sprintf("release text-input source is %q, want examples/surface_release_text_input.tetra", report.Source))
	}
	if report.Level != "production-text-input-v1" {
		issues = append(issues, fmt.Sprintf("release text-input level is %q, want production-text-input-v1", report.Level))
	}
	if report.Experimental || !report.ProductionClaim {
		issues = append(issues, "release text-input requires experimental=false and production_claim=true")
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func runtimeReportHasProcessNameAndPathMarkers(processes []surface.ProcessReport, kind string, nameMarker string, pathMarkers ...string) bool {
	nameMarker = strings.ToLower(strings.TrimSpace(nameMarker))
	for _, process := range processes {
		if strings.TrimSpace(process.Kind) != kind {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(process.Name))
		path := strings.ToLower(strings.TrimSpace(process.Path))
		if nameMarker != "" && !strings.Contains(name, nameMarker) {
			continue
		}
		ok := true
		for _, marker := range pathMarkers {
			if !strings.Contains(path, strings.ToLower(strings.TrimSpace(marker))) {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func validateSurfaceV1RuntimeReleaseReport(report surface.Report) error {
	var issues []string
	releaseForm := isSurfaceReleaseFormSource(report.Source)
	releaseAccessibility := isSurfaceReleaseAccessibilitySource(report.Source)
	if !releaseForm && !releaseAccessibility {
		issues = append(issues, fmt.Sprintf("release surface-v1 source is %q, want release Surface example", report.Source))
		if report.Target == "wasm32-web" && report.HostEvidence.Level != "wasm32-web-browser-canvas-release-v1" {
			issues = append(issues, fmt.Sprintf("release surface-v1 wasm32-web host_evidence.level is %q, want wasm32-web-browser-canvas-release-v1 for non-release browser evidence", report.HostEvidence.Level))
		}
		if report.Target == "linux-x64" && report.HostEvidence.Level != "linux-x64-release-window-v1" {
			issues = append(issues, fmt.Sprintf("release surface-v1 linux-x64 host_evidence.level is %q, want linux-x64-release-window-v1 for non-release linux evidence", report.HostEvidence.Level))
		}
	}
	if report.HostEvidence.UserFacingPlatformWidgets {
		issues = append(issues, "release surface-v1 forbids user-facing platform widgets")
	}
	switch report.Target {
	case "linux-x64":
		issues = append(issues, validateSurfaceV1LinuxHostEvidence(report)...)
	case "wasm32-web":
		issues = append(issues, validateSurfaceV1BrowserHostEvidence(report)...)
	case "headless":
		if !strings.Contains(report.Source, "surface_release_") {
			issues = append(issues, fmt.Sprintf("release surface-v1 headless source is %q, want release Surface example", report.Source))
		}
	default:
		issues = append(issues, fmt.Sprintf("release surface-v1 target is %q, want headless, linux-x64, or wasm32-web", report.Target))
	}
	if (report.Target == "linux-x64" || report.Target == "wasm32-web") && releaseForm {
		issues = append(issues, validateSurfaceV1ReleaseToolkit(report)...)
	}
	if (report.Target == "linux-x64" || report.Target == "wasm32-web") && (releaseAccessibility || isSurfaceV1FinalLinuxWindowReport(report)) {
		issues = append(issues, validateSurfaceV1ReleaseAccessibilityTree(report)...)
	}
	if report.Target == "linux-x64" || report.Target == "wasm32-web" {
		if report.ComponentTree == nil {
			issues = append(issues, "release surface-v1 runtime report requires component-tree schema")
		}
		if report.ComponentTreeAPI == nil {
			issues = append(issues, "release surface-v1 runtime report requires component-tree-api schema")
		}
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateSurfaceV1LinuxHostEvidence(report surface.Report) []string {
	if isSurfaceV1FinalLinuxWindowReport(report) {
		return validateSurfaceV1FinalLinuxWindowHostEvidence(report)
	}
	var issues []string
	if report.HostEvidence.Level != "linux-x64-real-window" {
		issues = append(issues, fmt.Sprintf("release surface-v1 linux-x64 host_evidence.level is %q, want linux-x64-real-window or linux-x64-release-window-v1", report.HostEvidence.Level))
	}
	if report.HostEvidence.Backend != "wayland-shm-rgba" {
		issues = append(issues, fmt.Sprintf("release surface-v1 linux-x64 host_evidence.backend is %q, want wayland-shm-rgba", report.HostEvidence.Backend))
	}
	for _, check := range []struct {
		name string
		ok   bool
	}{
		{name: "framebuffer", ok: report.HostEvidence.Framebuffer},
		{name: "real_window", ok: report.HostEvidence.RealWindow},
		{name: "native_input", ok: report.HostEvidence.NativeInput},
	} {
		if !check.ok {
			issues = append(issues, fmt.Sprintf("release surface-v1 linux-x64 host_evidence.%s must be true", check.name))
		}
	}
	return issues
}

func validateSurfaceV1FinalLinuxWindowHostEvidence(report surface.Report) []string {
	var issues []string
	if report.HostEvidence.Level != "linux-x64-release-window-v1" {
		issues = append(issues, fmt.Sprintf("release surface-v1 linux-x64 host_evidence.level is %q, want linux-x64-release-window-v1", report.HostEvidence.Level))
	}
	if report.HostEvidence.Backend != "wayland-shm-rgba-release-v1" {
		issues = append(issues, fmt.Sprintf("release surface-v1 linux-x64 host_evidence.backend is %q, want wayland-shm-rgba-release-v1", report.HostEvidence.Backend))
	}
	for _, check := range []struct {
		name string
		ok   bool
	}{
		{name: "framebuffer", ok: report.HostEvidence.Framebuffer},
		{name: "real_window", ok: report.HostEvidence.RealWindow},
		{name: "native_input", ok: report.HostEvidence.NativeInput},
		{name: "text_input", ok: report.HostEvidence.TextInput},
		{name: "clipboard", ok: report.HostEvidence.Clipboard},
		{name: "composition", ok: report.HostEvidence.Composition},
		{name: "accessibility_bridge", ok: report.HostEvidence.AccessibilityBridge},
	} {
		if !check.ok {
			issues = append(issues, fmt.Sprintf("release surface-v1 linux-x64 host_evidence.%s must be true", check.name))
		}
	}
	return issues
}

func validateSurfaceV1BrowserHostEvidence(report surface.Report) []string {
	if isSurfaceV1FinalBrowserReleaseReport(report) {
		return validateSurfaceV1FinalBrowserHostEvidence(report)
	}
	var issues []string
	if report.HostEvidence.Level != "wasm32-web-browser-canvas-input" {
		issues = append(issues, fmt.Sprintf("release surface-v1 wasm32-web host_evidence.level is %q, want wasm32-web-browser-canvas-input or wasm32-web-browser-canvas-release-v1", report.HostEvidence.Level))
	}
	if report.HostEvidence.Backend != "browser-canvas-rgba" {
		issues = append(issues, fmt.Sprintf("release surface-v1 wasm32-web host_evidence.backend is %q, want browser-canvas-rgba", report.HostEvidence.Backend))
	}
	for _, check := range []struct {
		name string
		ok   bool
	}{
		{name: "framebuffer", ok: report.HostEvidence.Framebuffer},
		{name: "native_input", ok: report.HostEvidence.NativeInput},
	} {
		if !check.ok {
			issues = append(issues, fmt.Sprintf("release surface-v1 wasm32-web host_evidence.%s must be true", check.name))
		}
	}
	return issues
}

func validateSurfaceV1FinalBrowserHostEvidence(report surface.Report) []string {
	var issues []string
	if report.HostEvidence.Level != "wasm32-web-browser-canvas-release-v1" {
		issues = append(issues, fmt.Sprintf("release surface-v1 wasm32-web host_evidence.level is %q, want wasm32-web-browser-canvas-release-v1", report.HostEvidence.Level))
	}
	if report.HostEvidence.Backend != "browser-canvas-rgba-accessible" {
		issues = append(issues, fmt.Sprintf("release surface-v1 wasm32-web host_evidence.backend is %q, want browser-canvas-rgba-accessible", report.HostEvidence.Backend))
	}
	for _, check := range []struct {
		name string
		ok   bool
	}{
		{name: "framebuffer", ok: report.HostEvidence.Framebuffer},
		{name: "browser_canvas", ok: report.HostEvidence.BrowserCanvas},
		{name: "browser_input", ok: report.HostEvidence.BrowserInput},
		{name: "browser_clipboard", ok: report.HostEvidence.BrowserClipboard},
		{name: "browser_composition", ok: report.HostEvidence.BrowserComposition},
		{name: "browser_accessibility_snapshot", ok: report.HostEvidence.BrowserAccessibilitySnapshot},
		{name: "browser_accessibility_mirror", ok: report.HostEvidence.BrowserAccessibilityMirror},
	} {
		if !check.ok {
			issues = append(issues, fmt.Sprintf("release surface-v1 wasm32-web host_evidence.%s must be true", check.name))
		}
	}
	if report.HostEvidence.BrowserClipboardHarness != "deterministic-browser-clipboard-v1" {
		issues = append(issues, fmt.Sprintf("release surface-v1 wasm32-web browser_clipboard_harness is %q, want deterministic-browser-clipboard-v1", report.HostEvidence.BrowserClipboardHarness))
	}
	return issues
}

func validateSurfaceV1ReleaseToolkit(report surface.Report) []string {
	var issues []string
	if report.Toolkit == nil {
		return []string{"release surface-v1 runtime report requires production toolkit schema"}
	}
	if report.Toolkit.ToolkitLevel != "production-widgets-v1" || !report.Toolkit.ProductionClaim || report.Toolkit.Experimental {
		issues = append(issues, "release surface-v1 toolkit must be production-widgets-v1 with production_claim=true and experimental=false")
	}
	if report.Toolkit.ReleaseScope != surface.ReleaseScopeSurfaceV1LinuxWeb {
		issues = append(issues, fmt.Sprintf("release surface-v1 toolkit release_scope is %q, want %q", report.Toolkit.ReleaseScope, surface.ReleaseScopeSurfaceV1LinuxWeb))
	}
	if !report.Toolkit.NoDOMUI || !report.Toolkit.NoUserJS || !report.Toolkit.NoPlatformWidgets {
		issues = append(issues, "release surface-v1 toolkit must reject DOM UI, user JS, and platform widgets")
	}
	return issues
}

func validateSurfaceV1ReleaseAccessibilityTree(report surface.Report) []string {
	var issues []string
	tree := report.AccessibilityTree
	if tree == nil {
		return []string{"release surface-v1 runtime report requires platform accessibility bridge schema"}
	}
	if tree.AccessibilityLevel != "platform-bridge-v1" || !tree.ProductionClaim || tree.Experimental {
		issues = append(issues, "release surface-v1 accessibility_tree must be platform-bridge-v1 with production_claim=true and experimental=false")
	}
	if tree.ReleaseScope != surface.ReleaseScopeSurfaceV1LinuxWeb {
		issues = append(issues, fmt.Sprintf("release surface-v1 accessibility_tree release_scope is %q, want %q", tree.ReleaseScope, surface.ReleaseScopeSurfaceV1LinuxWeb))
	}
	if !tree.MetadataTree || !tree.PlatformExport {
		issues = append(issues, "release surface-v1 accessibility_tree requires metadata_tree and platform_export")
	}
	switch report.Target {
	case "headless":
		if tree.PlatformHostIntegration || tree.LinuxPlatformProbe || strings.TrimSpace(tree.LinuxProbeArtifact) != "" || tree.BrowserAccessibilitySnap || tree.BrowserAccessibilityMirror {
			issues = append(issues, "release surface-v1 headless accessibility must not claim linux platform probe or browser accessibility mirror")
		}
		if tree.PlatformBridge != "headless_accessibility_export_v1" {
			issues = append(issues, fmt.Sprintf("release surface-v1 headless accessibility platform_bridge is %q, want headless_accessibility_export_v1", tree.PlatformBridge))
		}
		screenReaderEvidence, ok := accessibilityEvidenceString(tree.ScreenReaderEvidence)
		if !ok || screenReaderEvidence != "headless_platform_tree_probe" {
			issues = append(issues, fmt.Sprintf("release surface-v1 headless accessibility screen_reader_evidence is %q, want headless_platform_tree_probe", screenReaderEvidence))
		}
	case "linux-x64":
		if !report.HostEvidence.AccessibilityBridge {
			issues = append(issues, "release surface-v1 linux accessibility host_evidence.accessibility_bridge must be true")
		}
		if !tree.PlatformHostIntegration || tree.PlatformBridge != "linux_accessibility_host_bridge_v1" || !tree.LinuxPlatformProbe || strings.TrimSpace(tree.LinuxProbeArtifact) == "" {
			issues = append(issues, "release surface-v1 linux accessibility requires linux platform probe bridge evidence")
		}
		screenReaderEvidence, ok := accessibilityEvidenceString(tree.ScreenReaderEvidence)
		if !ok || screenReaderEvidence != "linux_accessibility_host_bridge_v1" {
			issues = append(issues, fmt.Sprintf("release surface-v1 linux accessibility screen_reader_evidence is %q, want linux_accessibility_host_bridge_v1", screenReaderEvidence))
		}
		if !runtimeReportHasProcessNameAndPathMarkers(report.Processes, "runtime", "surface linux accessibility platform probe") {
			issues = append(issues, "release surface-v1 linux accessibility requires surface linux accessibility platform probe process evidence")
		}
		if !runtimeReportHasArtifactKind(report.Artifacts, "linux-accessibility-platform-probe") {
			issues = append(issues, "release surface-v1 linux accessibility requires linux-accessibility-platform-probe artifact")
		}
	case "wasm32-web":
		if !report.HostEvidence.BrowserAccessibilitySnapshot || !report.HostEvidence.BrowserAccessibilityMirror {
			issues = append(issues, "release surface-v1 browser accessibility requires host_evidence browser accessibility snapshot/mirror")
		}
		if !tree.PlatformHostIntegration || tree.PlatformBridge != "browser_accessibility_mirror_v1" || !tree.BrowserAccessibilitySnap || !tree.BrowserAccessibilityMirror {
			issues = append(issues, "release surface-v1 browser accessibility requires browser accessibility snapshot/mirror evidence")
		}
		screenReaderEvidence, ok := accessibilityEvidenceString(tree.ScreenReaderEvidence)
		if !ok || screenReaderEvidence != "browser_accessibility_snapshot_v1" {
			issues = append(issues, fmt.Sprintf("release surface-v1 browser accessibility screen_reader_evidence is %q, want browser_accessibility_snapshot_v1", screenReaderEvidence))
		}
		if !runtimeReportHasProcessNameAndPathMarkers(report.Processes, "runtime", "surface wasm32-web browser canvas trace") {
			issues = append(issues, "release surface-v1 browser accessibility requires browser canvas trace process evidence")
		}
		if !runtimeReportHasArtifactKind(report.Artifacts, "runner-trace") {
			issues = append(issues, "release surface-v1 browser accessibility requires runner-trace artifact")
		}
	}
	return issues
}

func accessibilityEvidenceString(value any) (string, bool) {
	text, ok := value.(string)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(text), true
}

func runtimeReportHasArtifactKind(artifacts []surface.ArtifactReport, kind string) bool {
	for _, artifact := range artifacts {
		if strings.TrimSpace(artifact.Kind) == kind {
			return true
		}
	}
	return false
}

func isSurfaceReleaseFormSource(source string) bool {
	return normalizeEvidencePath(source) == "examples/surface_release_form.tetra"
}

func isSurfaceReleaseAccessibilitySource(source string) bool {
	return normalizeEvidencePath(source) == "examples/surface_release_accessibility.tetra"
}

func isSurfaceV1FinalLinuxWindowReport(report surface.Report) bool {
	return report.HostEvidence.Level == "linux-x64-release-window-v1" ||
		reportHasCase(report, "linux release window v1 schema")
}

func isSurfaceV1FinalBrowserReleaseReport(report surface.Report) bool {
	return report.HostEvidence.Level == "wasm32-web-browser-canvas-release-v1" ||
		reportHasCase(report, "browser release Surface v1 schema")
}

func reportHasCase(report surface.Report, name string) bool {
	for _, tc := range report.Cases {
		if tc.Name == name {
			return true
		}
	}
	return false
}

func validateArtifactIntegrity(reportDir string, report surface.Report) error {
	var issues []string
	for _, artifact := range report.Artifacts {
		path := strings.TrimSpace(artifact.Path)
		if path == "" {
			continue
		}
		actualPath := resolveReportEvidencePath(reportDir, path)
		size, digest, err := hashFile(actualPath)
		if err != nil {
			issues = append(issues, fmt.Sprintf("artifact integrity %s: %v", path, err))
			continue
		}
		if size != artifact.Size {
			issues = append(issues, fmt.Sprintf("artifact integrity %s size = %d, want %d", path, size, artifact.Size))
		}
		if digest != strings.ToLower(strings.TrimSpace(artifact.SHA256)) {
			issues = append(issues, fmt.Sprintf("artifact integrity %s sha256 = %s, want %s", path, digest, artifact.SHA256))
		}
		if strings.TrimSpace(artifact.Kind) == "compiler-owned-loader" {
			if err := validateCompilerOwnedLoaderArtifact(actualPath); err != nil {
				issues = append(issues, fmt.Sprintf("artifact integrity %s: %v", path, err))
			}
		}
		if strings.TrimSpace(artifact.Kind) == "runner-trace" {
			if err := validateRunnerTraceArtifact(actualPath, report); err != nil {
				issues = append(issues, fmt.Sprintf("artifact integrity %s: %v", path, err))
			}
		}
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func resolveReportEvidencePath(reportDir string, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	cleanPath := filepath.Clean(path)
	reportRelative := filepath.Join(reportDir, cleanPath)
	if _, err := os.Stat(reportRelative); err == nil {
		return reportRelative
	}
	if _, err := os.Stat(cleanPath); err == nil {
		return cleanPath
	}
	return reportRelative
}

func validateCompilerOwnedLoaderArtifact(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if marker, ok := forbiddenCompilerOwnedLoaderMarker(string(raw)); ok {
		return fmt.Errorf("compiler-owned loader must not contain DOM/user-JS marker %q", marker)
	}
	return nil
}

type runnerTraceEnvelope struct {
	Schema               string                   `json:"schema"`
	Source               string                   `json:"source"`
	WASM                 string                   `json:"wasm_path"`
	Canvas               runnerTraceCanvas        `json:"canvas"`
	BrowserEvents        []runnerTraceEvent       `json:"browser_events"`
	BrowserClipboard     runnerTraceClipboard     `json:"browser_clipboard"`
	BrowserComposition   runnerTraceComposition   `json:"browser_composition"`
	BrowserAccessibility runnerTraceAccessibility `json:"browser_accessibility"`
	Frames               []runnerTraceFrame       `json:"frames"`
	AppExitCode          *int                     `json:"app_exit_code"`
}

type runnerTraceFrame struct {
	Order          int    `json:"order"`
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	Stride         int    `json:"stride"`
	PixelsLen      int    `json:"pixels_len"`
	SourceChecksum string `json:"source_checksum"`
	CanvasChecksum string `json:"canvas_checksum"`
	Checksum       string `json:"checksum"`
	Presented      bool   `json:"presented"`
}

type runnerTraceCanvas struct {
	Opened   bool `json:"opened"`
	Readback bool `json:"readback"`
	Width    int  `json:"width"`
	Height   int  `json:"height"`
}

type runnerTraceEvent struct {
	NativeType string `json:"native_type"`
	Kind       int    `json:"kind"`
}

type runnerTraceClipboard struct {
	Harness   string `json:"harness"`
	Read      bool   `json:"read"`
	Write     bool   `json:"write"`
	OwnedCopy bool   `json:"owned_copy"`
	Bytes     int    `json:"bytes"`
}

type runnerTraceComposition struct {
	Start  bool `json:"start"`
	Update bool `json:"update"`
	Commit bool `json:"commit"`
	Cancel bool `json:"cancel"`
}

type runnerTraceAccessibility struct {
	Snapshot      bool     `json:"snapshot"`
	Mirror        bool     `json:"mirror"`
	CompilerOwned bool     `json:"compiler_owned"`
	Roles         []string `json:"roles"`
	Bounds        bool     `json:"bounds"`
	Focus         bool     `json:"focus"`
	DOMVisualUI   bool     `json:"dom_visual_ui"`
	UserJS        bool     `json:"user_js"`
}

func validateRunnerTraceArtifact(path string, report surface.Report) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var trace runnerTraceEnvelope
	if err := json.Unmarshal(raw, &trace); err != nil {
		return fmt.Errorf("decode runner trace: %w", err)
	}
	var frames []surface.FrameReport
	switch trace.Schema {
	case "tetra.surface.headless-runner-trace.v1":
		if report.Target != "headless" {
			return fmt.Errorf("runner trace schema %q is not valid for %s target", trace.Schema, report.Target)
		}
		if !sameEvidencePathReference(trace.Source, report.Source) {
			return fmt.Errorf("runner trace source is %q, want reported source %q", trace.Source, report.Source)
		}
		for _, frame := range trace.Frames {
			frames = append(frames, surface.FrameReport{
				Order:     frame.Order,
				Width:     frame.Width,
				Height:    frame.Height,
				Stride:    frame.Stride,
				Checksum:  frame.Checksum,
				Presented: frame.Presented,
			})
		}
	case "tetra.surface.web-runner-trace.v1":
		if report.Target != "wasm32-web" {
			return fmt.Errorf("runner trace schema %q is not valid for %s target", trace.Schema, report.Target)
		}
		if report.HostEvidence.Level == "wasm32-web-browser-canvas-input" {
			return fmt.Errorf("runner trace schema %q is starter Node evidence, not browser canvas evidence", trace.Schema)
		}
		if !runnerTraceWASMMatchesComponentArtifact(trace.WASM, report.Artifacts) {
			return fmt.Errorf("runner trace wasm_path %q does not match reported wasm component artifact", trace.WASM)
		}
		for _, frame := range trace.Frames {
			if frame.PixelsLen <= 0 {
				return fmt.Errorf("runner trace frame %d pixels_len must be positive", frame.Order)
			}
			frames = append(frames, surface.FrameReport{
				Order:     frame.Order + 2,
				Width:     frame.Width,
				Height:    frame.Height,
				Stride:    frame.Stride,
				Checksum:  frame.Checksum,
				Presented: true,
			})
		}
	case "tetra.surface.browser-canvas-trace.v1":
		if report.Target != "wasm32-web" {
			return fmt.Errorf("runner trace schema %q is not valid for %s target", trace.Schema, report.Target)
		}
		if !isBrowserCanvasHostEvidenceLevel(report.HostEvidence.Level) {
			return fmt.Errorf("browser canvas runner trace requires browser canvas host_evidence.level, got %q", report.HostEvidence.Level)
		}
		if !trace.Canvas.Opened || !trace.Canvas.Readback {
			return fmt.Errorf("browser canvas runner trace missing opened/readback canvas evidence")
		}
		if trace.AppExitCode == nil || *trace.AppExitCode != 1 {
			return fmt.Errorf("browser canvas runner trace app_exit_code = %v, want 1", trace.AppExitCode)
		}
		if !runnerTraceWASMMatchesComponentArtifact(trace.WASM, report.Artifacts) {
			return fmt.Errorf("runner trace wasm_path %q does not match reported wasm component artifact", trace.WASM)
		}
		if !runnerTraceHasBrowserNativeEvents(trace.BrowserEvents, []string{"pointerup", "keydown", "resize", "beforeinput"}) {
			return fmt.Errorf("browser canvas runner trace missing pointer/key/resize/text native event evidence")
		}
		if report.HostEvidence.Level == "wasm32-web-browser-canvas-release-v1" {
			if err := validateBrowserReleaseTrace(trace); err != nil {
				return err
			}
		}
		if isSurfaceReleaseAccessibilitySource(report.Source) {
			if err := validateBrowserAccessibilityRunnerTrace(trace); err != nil {
				return err
			}
		}
		for _, frame := range trace.Frames {
			if frame.PixelsLen <= 0 {
				return fmt.Errorf("browser canvas runner trace frame %d pixels_len must be positive", frame.Order)
			}
			if strings.TrimSpace(frame.SourceChecksum) == "" || strings.TrimSpace(frame.CanvasChecksum) == "" {
				return fmt.Errorf("browser canvas runner trace frame %d missing source/canvas checksum readback evidence", frame.Order)
			}
			if frame.SourceChecksum != frame.CanvasChecksum || frame.Checksum != frame.CanvasChecksum {
				return fmt.Errorf("browser canvas runner trace frame %d checksum mismatch source=%s canvas=%s checksum=%s", frame.Order, frame.SourceChecksum, frame.CanvasChecksum, frame.Checksum)
			}
			frames = append(frames, surface.FrameReport{
				Order:     frame.Order,
				Width:     frame.Width,
				Height:    frame.Height,
				Stride:    frame.Stride,
				Checksum:  frame.Checksum,
				Presented: frame.Presented,
			})
		}
	default:
		return fmt.Errorf("runner trace schema is %q, want tetra.surface.headless-runner-trace.v1, tetra.surface.web-runner-trace.v1, or tetra.surface.browser-canvas-trace.v1", trace.Schema)
	}
	if len(frames) < 2 {
		return fmt.Errorf("runner trace has %d frames, want at least 2", len(frames))
	}
	for _, frame := range frames {
		if frame.Width <= 0 || frame.Height <= 0 || frame.Stride <= 0 || strings.TrimSpace(frame.Checksum) == "" || !frame.Presented {
			return fmt.Errorf("runner trace frame %d has incomplete presented frame evidence", frame.Order)
		}
		if !reportHasFrame(report.Frames, frame) {
			return fmt.Errorf("runner trace frame %d does not match reported Surface frame evidence", frame.Order)
		}
	}
	return nil
}

func runnerTraceHasBrowserNativeEvents(events []runnerTraceEvent, nativeTypes []string) bool {
	seen := map[string]bool{}
	for _, event := range events {
		seen[strings.ToLower(strings.TrimSpace(event.NativeType))] = true
	}
	for _, nativeType := range nativeTypes {
		if !seen[strings.ToLower(nativeType)] {
			return false
		}
	}
	return true
}

func validateBrowserReleaseTrace(trace runnerTraceEnvelope) error {
	if !runnerTraceHasBrowserNativeEvents(trace.BrowserEvents, []string{"compositionstart", "compositionupdate", "compositionend"}) {
		return fmt.Errorf("browser release runner trace missing composition native event evidence")
	}
	if trace.BrowserClipboard.Harness != "deterministic-browser-clipboard-v1" ||
		!trace.BrowserClipboard.Read ||
		!trace.BrowserClipboard.Write ||
		!trace.BrowserClipboard.OwnedCopy ||
		trace.BrowserClipboard.Bytes <= 0 {
		return fmt.Errorf("browser release runner trace missing deterministic clipboard harness evidence")
	}
	if !trace.BrowserComposition.Start ||
		!trace.BrowserComposition.Update ||
		!trace.BrowserComposition.Commit ||
		!trace.BrowserComposition.Cancel {
		return fmt.Errorf("browser release runner trace missing composition trace evidence")
	}
	return validateBrowserAccessibilityRunnerTrace(trace)
}

func validateBrowserAccessibilityRunnerTrace(trace runnerTraceEnvelope) error {
	if !trace.BrowserAccessibility.Snapshot ||
		!trace.BrowserAccessibility.Mirror ||
		!trace.BrowserAccessibility.CompilerOwned ||
		!trace.BrowserAccessibility.Bounds ||
		!trace.BrowserAccessibility.Focus {
		return fmt.Errorf("browser release runner trace missing accessibility snapshot/mirror evidence")
	}
	if trace.BrowserAccessibility.DOMVisualUI || trace.BrowserAccessibility.UserJS {
		return fmt.Errorf("browser release runner trace must not claim DOM visual UI or user JS app logic")
	}
	for _, role := range []string{"root", "textbox", "checkbox", "button", "status"} {
		if !containsString(trace.BrowserAccessibility.Roles, role) {
			return fmt.Errorf("browser release runner trace missing accessibility role %s", role)
		}
	}
	return nil
}

func isBrowserCanvasHostEvidenceLevel(level string) bool {
	return level == "wasm32-web-browser-canvas-input" ||
		level == "wasm32-web-browser-canvas-release-v1"
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func runnerTraceWASMMatchesComponentArtifact(wasmPath string, artifacts []surface.ArtifactReport) bool {
	wasmPath = normalizeEvidencePath(wasmPath)
	if wasmPath == "" {
		return false
	}
	for _, artifact := range artifacts {
		if strings.TrimSpace(artifact.Kind) != "component-app" {
			continue
		}
		path := normalizeEvidencePath(artifact.Path)
		if strings.HasSuffix(strings.ToLower(path), ".wasm") && path == wasmPath {
			return true
		}
	}
	return false
}

func sameEvidencePathReference(actual string, expected string) bool {
	actual = normalizeEvidencePath(actual)
	expected = normalizeEvidencePath(expected)
	if actual == "" || expected == "" {
		return false
	}
	return actual == expected || strings.HasSuffix(actual, "/"+expected) || strings.HasSuffix(expected, "/"+actual)
}

func normalizeEvidencePath(path string) string {
	path = strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	return path
}

func reportHasFrame(frames []surface.FrameReport, want surface.FrameReport) bool {
	for _, frame := range frames {
		if frame.Order == want.Order &&
			frame.Width == want.Width &&
			frame.Height == want.Height &&
			frame.Stride == want.Stride &&
			frame.Checksum == want.Checksum &&
			frame.Presented == want.Presented {
			return true
		}
	}
	return false
}

func forbiddenCompilerOwnedLoaderMarker(loader string) (string, bool) {
	lower := strings.ToLower(loader)
	for _, marker := range []string{
		"document.",
		"globalthis.document",
		"window.document",
		"createelement(",
		"appendchild(",
		"innerhtml",
		"queryselector(",
		"addeventlistener(",
		"<canvas",
		"<button",
		"mounttetraui",
		"tetra.ui.v1",
		".ui.web.mjs",
		".ui.html",
		"import(",
		".js\"",
		".js'",
	} {
		if strings.Contains(lower, marker) {
			return marker, true
		}
	}
	return "", false
}

func validateArtifactScanIntegrity(reportDir string, report surface.Report) error {
	scan := report.ArtifactScan
	root := strings.TrimSpace(scan.Root)
	if root == "" {
		return nil
	}
	actualRoot := resolveReportEvidencePath(reportDir, root)
	filesChecked, forbiddenPaths, err := scanArtifactFiles(actualRoot, compilerOwnedLoaderPaths(reportDir, report.Artifacts))
	if err != nil {
		return fmt.Errorf("artifact_scan integrity %s: %v", root, err)
	}
	var issues []string
	if filesChecked != scan.FilesChecked {
		issues = append(issues, fmt.Sprintf("artifact_scan.files_checked = %d, actual files under %s = %d", scan.FilesChecked, root, filesChecked))
	}
	if len(forbiddenPaths) > 0 {
		issues = append(issues, fmt.Sprintf("artifact_scan found legacy UI sidecar artifact %s", forbiddenPaths[0]))
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func compilerOwnedLoaderPaths(reportDir string, artifacts []surface.ArtifactReport) map[string]bool {
	paths := map[string]bool{}
	for _, artifact := range artifacts {
		if strings.TrimSpace(artifact.Kind) != "compiler-owned-loader" {
			continue
		}
		path := strings.TrimSpace(artifact.Path)
		if path == "" {
			continue
		}
		path = resolveReportEvidencePath(reportDir, path)
		paths[filepath.Clean(path)] = true
	}
	return paths
}

func scanArtifactFiles(root string, compilerOwnedLoaders map[string]bool) (int, []string, error) {
	filesChecked := 0
	var forbiddenPaths []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		filesChecked++
		if legacyUISidecarArtifactPath(path, compilerOwnedLoaders) {
			forbiddenPaths = append(forbiddenPaths, path)
		}
		return nil
	})
	return filesChecked, forbiddenPaths, err
}

func legacyUISidecarArtifactPath(path string, compilerOwnedLoaders map[string]bool) bool {
	base := strings.ToLower(filepath.Base(path))
	if strings.Contains(base, ".ui.") ||
		strings.HasSuffix(base, ".html") ||
		strings.HasSuffix(base, ".js") {
		return true
	}
	if strings.HasSuffix(base, ".mjs") {
		return !compilerOwnedLoaders[filepath.Clean(path)]
	}
	return false
}

func hashFile(path string) (int64, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return 0, "", err
	}
	return size, "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}
