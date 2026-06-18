package surface

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ---- artifacts_validation.go ----

func validateProcesses(source string, processes []ProcessReport) []string {
	var issues []string
	if len(processes) < 3 {
		issues = append(
			issues,
			fmt.Sprintf(
				"process evidence has %d entries, want build, app, and runtime processes",
				len(processes),
			),
		)
	}
	seen := map[string]bool{}
	seenBuild := false
	seenBuildForSource := false
	seenApp := false
	seenComponentApp := false
	seenRuntime := false
	for _, process := range processes {
		if strings.TrimSpace(process.Name) == "" {
			issues = append(issues, "process name is required")
		} else if seen[process.Name] {
			issues = append(issues, fmt.Sprintf("duplicate process %s", process.Name))
		}
		seen[process.Name] = true
		switch process.Kind {
		case "build":
			seenBuild = true
			if processReferencesSource(process.Path, source) {
				seenBuildForSource = true
			}
		case "app":
			seenApp = true
			if isSurfaceComponentAppProcess(source, process) {
				seenComponentApp = true
			}
		case "runtime":
			seenRuntime = true
		default:
			issues = append(
				issues,
				fmt.Sprintf(
					"process %s kind is %q, want build, app, or runtime",
					process.Name,
					process.Kind,
				),
			)
		}
		if strings.TrimSpace(process.Path) == "" {
			issues = append(issues, fmt.Sprintf("process %s path is required", process.Name))
		} else if process.Kind == "app" && sourceLikeEvidencePath(process.Path) {
			issues = append(issues, fmt.Sprintf(("process %s path %q is not executable Surface app process "+
				"evidence"), process.Name, process.Path))
		}
		if !process.Ran {
			issues = append(issues, fmt.Sprintf("process %s did not run", process.Name))
		}
		if !process.Pass {
			issues = append(issues, fmt.Sprintf("process %s did not pass", process.Name))
		}
		if process.ExitCode == nil {
			issues = append(issues, fmt.Sprintf("process %s missing exit_code", process.Name))
			continue
		}
		wantExit := 0
		if process.ExpectedExitCode != nil {
			wantExit = *process.ExpectedExitCode
		}
		if *process.ExitCode != wantExit {
			issues = append(
				issues,
				fmt.Sprintf(
					"process %s exit_code = %d, want %d",
					process.Name,
					*process.ExitCode,
					wantExit,
				),
			)
		}
	}
	if !seenBuild {
		issues = append(issues, "process evidence missing build process")
	}
	if !seenBuildForSource {
		issues = append(
			issues,
			fmt.Sprintf("process evidence missing build process for reported source %q", source),
		)
	}
	if !seenApp {
		issues = append(issues, "process evidence missing executable Surface app process")
	}
	if !seenComponentApp {
		issues = append(
			issues,
			"process evidence missing executable Surface component app process with expected app exit",
		)
	}
	if !seenRuntime {
		issues = append(issues, "process evidence missing Surface runtime process")
	}
	return issues
}

func processReferencesSource(path string, source string) bool {
	source = normalizeEvidencePath(source)
	if source == "" {
		return false
	}
	path = normalizeEvidencePath(path)
	return strings.Contains(path, source)
}

func isSurfaceComponentAppProcess(source string, process ProcessReport) bool {
	name := strings.ToLower(strings.TrimSpace(process.Name))
	if process.Kind != "app" || !strings.Contains(name, "surface") ||
		!strings.Contains(name, "component app") {
		return false
	}
	if process.ExitCode == nil || process.ExpectedExitCode == nil {
		return false
	}
	if *process.ExitCode == 1 && *process.ExpectedExitCode == 1 {
		return true
	}
	if isSurfaceProjectTemplateSource(source) && *process.ExitCode == 0 &&
		*process.ExpectedExitCode == 0 {
		return true
	}
	if isSurfaceReferenceAppSource(source) && *process.ExitCode == 0 &&
		*process.ExpectedExitCode == 0 {
		return true
	}
	if isBlockPaintValidationComponentApp(source, process) && *process.ExitCode == 0 &&
		*process.ExpectedExitCode == 0 {
		return true
	}
	if isSurfaceFlagshipControlCenterSource(source) && *process.ExitCode == 5 &&
		*process.ExpectedExitCode == 5 {
		return true
	}
	if isSurfaceMorphRenderedFlagshipSource(source) && *process.ExitCode == 0 &&
		*process.ExpectedExitCode == 0 {
		return true
	}
	if isSurfaceMorphGuestDashboardSource(source) && *process.ExitCode == 0 &&
		*process.ExpectedExitCode == 0 {
		return true
	}
	if isSurfaceRuntimeWindowCounterSource(source) && *process.ExitCode == 0 &&
		*process.ExpectedExitCode == 0 {
		return true
	}
	return strings.Contains(name, "browser canvas") && *process.ExitCode == 0 &&
		*process.ExpectedExitCode == 0
}

func isSurfaceProjectTemplateSource(source string) bool {
	source = normalizeEvidencePath(source)
	if !strings.HasSuffix(source, "/src/main.tetra") {
		return false
	}
	parts := strings.Split(source, "/")
	for i, part := range parts {
		if part != "templates" || i+3 != len(parts)-1 {
			continue
		}
		if parts[i+1] == "" || parts[i+2] != "src" || parts[i+3] != "main.tetra" {
			continue
		}
		for _, prefix := range parts[:i] {
			if prefix == "reports" {
				return true
			}
		}
	}
	return false
}

func isSurfaceReferenceAppSource(source string) bool {
	source = normalizeEvidencePath(source)
	if strings.HasPrefix(source, "examples/surface/reference_core/surface_reference_") &&
		strings.HasSuffix(source, ".tetra") {
		return true
	}
	if strings.HasPrefix(source, "examples/surface/reference_forms/surface_reference_") &&
		strings.HasSuffix(source, ".tetra") {
		return true
	}
	return (strings.Contains(source, "/examples/surface/reference_core/surface_reference_") ||
		strings.Contains(source, "/examples/surface/reference_forms/surface_reference_")) &&
		strings.HasSuffix(source, ".tetra")
}

func isBlockPaintValidationComponentApp(source string, process ProcessReport) bool {
	return normalizeEvidencePath(
		source,
	) == "examples/surface/block_render/surface_block_paint_layers.tetra" &&
		strings.Contains(normalizeEvidencePath(process.Path), "surface-block-paint")
}

func isSurfaceFlagshipControlCenterSource(source string) bool {
	source = normalizeEvidencePath(source)
	return source == "examples/surface/migration/surface_migration_tetra_control_center.tetra" ||
		strings.HasSuffix(
			source,
			"/examples/surface/migration/surface_migration_tetra_control_center.tetra",
		)
}

func isSurfaceMorphRenderedFlagshipSource(source string) bool {
	source = normalizeEvidencePath(source)
	return source == "examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra" ||
		strings.HasSuffix(
			source,
			"/examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra",
		)
}

func isSurfaceMorphGuestDashboardSource(source string) bool {
	source = normalizeEvidencePath(source)
	return source == "examples/surface/morph_flagship/surface_morph_guest_dashboard.tetra" ||
		strings.HasSuffix(
			source,
			"/examples/surface/morph_flagship/surface_morph_guest_dashboard.tetra",
		)
}

func isSurfaceRuntimeWindowCounterSource(source string) bool {
	source = normalizeEvidencePath(source)
	return source == "examples/surface/runtime/surface_window_counter.tetra" ||
		strings.HasSuffix(source, "/examples/surface/runtime/surface_window_counter.tetra")
}

func normalizeEvidencePath(path string) string {
	path = strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	return path
}

func validateArtifacts(
	target string,
	source string,
	artifacts []ArtifactReport,
	processes []ProcessReport,
) []string {
	var issues []string
	if len(artifacts) == 0 {
		issues = append(issues, "artifact evidence is required")
	}
	seenPath := map[string]bool{}
	seenComponentAppArtifact := false
	seenCompilerOwnedLoaderArtifact := false
	seenRunnerTraceArtifact := false
	for _, artifact := range artifacts {
		kind := strings.TrimSpace(artifact.Kind)
		path := normalizeEvidencePath(artifact.Path)
		if kind == "" {
			issues = append(issues, "artifact kind is required")
		}
		if path == "" {
			issues = append(issues, fmt.Sprintf("artifact %s path is required", kind))
		} else if seenPath[path] {
			issues = append(issues, fmt.Sprintf("duplicate artifact path %s", artifact.Path))
		}
		seenPath[path] = true
		issues = append(issues, validateSurfaceArtifactPath(kind, path)...)
		if !validSHA256Digest(artifact.SHA256) {
			issues = append(
				issues,
				fmt.Sprintf("artifact %s sha256 must be sha256:<64 hex>", artifact.Path),
			)
		}
		if artifact.Size <= 0 {
			issues = append(issues, fmt.Sprintf("artifact %s size must be positive", artifact.Path))
		}
		if kind == "component-app" &&
			artifactReferencedByComponentAppProcess(source, path, processes) {
			seenComponentAppArtifact = true
		}
		if kind == "compiler-owned-loader" && strings.HasSuffix(strings.ToLower(path), ".mjs") {
			seenCompilerOwnedLoaderArtifact = true
		}
		if kind == "runner-trace" &&
			strings.HasSuffix(strings.ToLower(path), "surface-runner-trace.json") {
			seenRunnerTraceArtifact = true
		}
	}
	if !seenComponentAppArtifact {
		issues = append(
			issues,
			("artifact evidence missing Surface component app artifact " +
				"hash linked to Surface component app process"),
		)
	}
	if target == "wasm32-web" && !seenCompilerOwnedLoaderArtifact {
		issues = append(
			issues,
			"wasm32-web artifact evidence missing compiler-owned loader artifact",
		)
	}
	if (target == "headless" || target == "wasm32-web") && !seenRunnerTraceArtifact {
		issues = append(
			issues,
			fmt.Sprintf("%s artifact evidence missing Surface runner trace artifact", target),
		)
	}
	return issues
}

func validateSurfaceArtifactPath(kind string, path string) []string {
	lower := strings.ToLower(path)
	var issues []string
	if strings.Contains(lower, ".ui.") {
		issues = append(issues, fmt.Sprintf("artifact %s must not be a legacy UI sidecar", path))
	}
	if strings.HasSuffix(lower, ".html") {
		issues = append(issues, fmt.Sprintf("artifact %s must not be generated HTML UI", path))
	}
	if strings.HasSuffix(lower, ".js") {
		issues = append(
			issues,
			fmt.Sprintf("artifact %s must not be generated JavaScript UI", path),
		)
	}
	if strings.HasSuffix(lower, ".mjs") && kind != "compiler-owned-loader" {
		issues = append(
			issues,
			fmt.Sprintf(
				"artifact %s .mjs is only allowed for compiler-owned-loader evidence",
				path,
			),
		)
	}
	for _, forbidden := range []struct {
		suffix string
		model  string
	}{
		{suffix: ".jsx", model: "React"},
		{suffix: ".tsx", model: "React"},
		{suffix: ".qml", model: "Qt"},
		{suffix: ".xaml", model: "WinUI"},
		{suffix: ".xib", model: "Cocoa"},
		{suffix: ".storyboard", model: "Cocoa"},
		{suffix: ".glade", model: "GTK"},
	} {
		if strings.HasSuffix(lower, forbidden.suffix) {
			issues = append(
				issues,
				fmt.Sprintf(
					"artifact %s must not be %s user-facing UI evidence",
					path,
					forbidden.model,
				),
			)
		}
	}
	if kind == "compiler-owned-loader" && !strings.HasSuffix(lower, ".mjs") {
		issues = append(
			issues,
			fmt.Sprintf("compiler-owned loader artifact %s must be a .mjs loader", path),
		)
	}
	return issues
}

func validateArtifactScan(scan ArtifactScanReport, artifacts []ArtifactReport) []string {
	var issues []string
	root := normalizeEvidencePath(scan.Root)
	if root == "" {
		issues = append(issues, "artifact_scan.root is required")
	}
	if scan.FilesChecked <= 0 {
		issues = append(issues, "artifact_scan.files_checked must be positive")
	}
	if len(artifacts) > 0 && scan.FilesChecked < len(artifacts) {
		issues = append(
			issues,
			fmt.Sprintf(
				"artifact_scan.files_checked = %d, want at least %d reported artifacts",
				scan.FilesChecked,
				len(artifacts),
			),
		)
	}
	if !scan.Pass {
		issues = append(issues, "artifact_scan.pass must be true")
	}
	if len(scan.ForbiddenPaths) > 0 {
		issues = append(
			issues,
			fmt.Sprintf(
				"artifact_scan forbidden paths must be empty, got %d",
				len(scan.ForbiddenPaths),
			),
		)
	}
	for _, path := range scan.ForbiddenPaths {
		if strings.TrimSpace(path) == "" {
			issues = append(issues, "artifact_scan forbidden path must not be empty")
		}
	}
	for _, artifact := range artifacts {
		path := normalizeEvidencePath(artifact.Path)
		if root == "" || path == "" {
			continue
		}
		if !evidencePathUnderRoot(path, root) {
			issues = append(
				issues,
				fmt.Sprintf(
					"artifact %s is outside artifact_scan.root %s",
					artifact.Path,
					scan.Root,
				),
			)
		}
	}
	return issues
}

func evidencePathUnderRoot(path string, root string) bool {
	path = strings.TrimSuffix(normalizeEvidencePath(path), "/")
	root = strings.TrimSuffix(normalizeEvidencePath(root), "/")
	return path == root || strings.HasPrefix(path, root+"/")
}

func artifactReferencedByComponentAppProcess(
	source string,
	artifactPath string,
	processes []ProcessReport,
) bool {
	for _, process := range processes {
		if !isSurfaceComponentAppProcess(source, process) {
			continue
		}
		if strings.Contains(normalizeEvidencePath(process.Path), artifactPath) {
			return true
		}
	}
	return false
}

func validSHA256Digest(value string) bool {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "sha256:") {
		return false
	}
	hexDigest := strings.TrimPrefix(value, "sha256:")
	if len(hexDigest) != 64 {
		return false
	}
	_, err := hex.DecodeString(hexDigest)
	return err == nil
}

// ---- browser_validation.go ----

type BrowserSurfaceReport struct {
	Schema              string                            `json:"schema"`
	BrowserSurfaceLevel string                            `json:"browser_surface_level"`
	ReleaseScope        string                            `json:"release_scope"`
	Source              string                            `json:"source"`
	HostAdapter         string                            `json:"host_adapter"`
	ProductionClaim     bool                              `json:"production_claim"`
	Experimental        bool                              `json:"experimental"`
	CompilerOwnedBoot   bool                              `json:"compiler_owned_boot"`
	DOMHostCanvasOnly   bool                              `json:"dom_host_canvas_only"`
	Canvas              BrowserSurfaceCanvasReport        `json:"canvas"`
	Input               BrowserSurfaceInputReport         `json:"input"`
	Clipboard           BrowserSurfaceClipboardReport     `json:"clipboard"`
	Composition         BrowserSurfaceCompositionReport   `json:"composition"`
	Accessibility       BrowserSurfaceAccessibilityReport `json:"accessibility"`
	HostTraces          []BrowserSurfaceHostTraceReport   `json:"host_traces"`
	NegativeGuards      BrowserSurfaceNegativeGuards      `json:"negative_guards"`
}

type BrowserSurfaceCanvasReport struct {
	Opened       bool   `json:"opened"`
	Readback     bool   `json:"readback"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FrameOrder   int    `json:"frame_order"`
	ArtifactKind string `json:"artifact_kind"`
	Pass         bool   `json:"pass"`
}

type BrowserSurfaceInputReport struct {
	Pointer      bool     `json:"pointer"`
	Keyboard     bool     `json:"keyboard"`
	Text         bool     `json:"text"`
	Resize       bool     `json:"resize"`
	HostTrace    bool     `json:"host_trace"`
	NativeEvents []string `json:"native_events"`
	Pass         bool     `json:"pass"`
}

type BrowserSurfaceClipboardReport struct {
	Harness   string `json:"harness"`
	Read      bool   `json:"read"`
	Write     bool   `json:"write"`
	OwnedCopy bool   `json:"owned_copy"`
	Bytes     int    `json:"bytes"`
	Pass      bool   `json:"pass"`
}

type BrowserSurfaceCompositionReport struct {
	Start  bool `json:"start"`
	Update bool `json:"update"`
	Commit bool `json:"commit"`
	Cancel bool `json:"cancel"`
	Pass   bool `json:"pass"`
}

type BrowserSurfaceAccessibilityReport struct {
	Snapshot      bool     `json:"snapshot"`
	Mirror        bool     `json:"mirror"`
	CompilerOwned bool     `json:"compiler_owned"`
	Bounds        bool     `json:"bounds"`
	Focus         bool     `json:"focus"`
	Roles         []string `json:"roles"`
	DOMVisualUI   bool     `json:"dom_visual_ui"`
	UserJS        bool     `json:"user_js"`
	Pass          bool     `json:"pass"`
}

type BrowserSurfaceHostTraceReport struct {
	Name         string `json:"name"`
	ArtifactKind string `json:"artifact_kind"`
	Path         string `json:"path"`
	Pass         bool   `json:"pass"`
}

type BrowserSurfaceNegativeGuards struct {
	NoDOMAppUITree      bool `json:"no_dom_app_ui_tree"`
	NoUserJSAppLogic    bool `json:"no_user_js_app_logic"`
	NoNodeOnlyPromotion bool `json:"no_node_only_promotion"`
	NoLegacySidecars    bool `json:"no_legacy_sidecars"`
	NoReactRuntime      bool `json:"no_react_runtime"`
	NoPlatformWidgets   bool `json:"no_platform_widgets"`
}

func validateBrowserReleaseEvidence(report Report) []string {
	if !isBrowserReleaseReport(report) {
		return nil
	}
	var issues []string
	if report.Target != "wasm32-web" {
		issues = append(
			issues,
			fmt.Sprintf("browser release target is %q, want wasm32-web", report.Target),
		)
	}
	if !isSurfaceReleaseFormSource(report.Source) {
		issues = append(
			issues,
			fmt.Sprintf(
				("browser release source path must match "+
					"examples/surface/release/surface_release_form.tetra, got %q"),
				report.Source,
			),
		)
	}
	if report.HostEvidence.Level != "wasm32-web-browser-canvas-release-v1" {
		issues = append(
			issues,
			fmt.Sprintf(
				"browser release host_evidence.level is %q, want wasm32-web-browser-canvas-release-v1",
				report.HostEvidence.Level,
			),
		)
	}
	if report.HostEvidence.Backend != "browser-canvas-rgba-accessible" {
		issues = append(
			issues,
			fmt.Sprintf(
				"browser release host_evidence.backend is %q, want browser-canvas-rgba-accessible",
				report.HostEvidence.Backend,
			),
		)
	}
	if !report.HostEvidence.BrowserCanvas {
		issues = append(issues, "browser release host_evidence.browser_canvas must be true")
	}
	if !report.HostEvidence.BrowserInput {
		issues = append(issues, "browser release host_evidence.browser_input must be true")
	}
	if !report.HostEvidence.BrowserClipboard {
		issues = append(issues, "browser release host_evidence.browser_clipboard must be true")
	}
	if report.HostEvidence.BrowserClipboardHarness != "deterministic-browser-clipboard-v1" {
		issues = append(
			issues,
			fmt.Sprintf(
				("browser release host_evidence.browser_clipboard_harness is "+
					"%q, want deterministic-browser-clipboard-v1"),
				report.HostEvidence.BrowserClipboardHarness,
			),
		)
	}
	if !report.HostEvidence.BrowserComposition {
		issues = append(issues, "browser release host_evidence.browser_composition must be true")
	}
	if !report.HostEvidence.BrowserAccessibilitySnapshot {
		issues = append(
			issues,
			"browser release host_evidence.browser_accessibility_snapshot must be true",
		)
	}
	if !report.HostEvidence.BrowserAccessibilityMirror {
		issues = append(
			issues,
			"browser release host_evidence.browser_accessibility_mirror must be true",
		)
	}
	if !hasFrameOrderDimensions(report.Frames, 5, 560, 420, 2240) {
		issues = append(
			issues,
			"browser release requires order-5 560x420 canvas readback frame evidence",
		)
	}
	for _, required := range []string{
		"browser release Surface v1 schema",
		"browser release Chromium canvas readback",
		"browser release native pointer keyboard text resize",
		"browser release deterministic clipboard harness",
		"browser release composition trace",
		"browser release accessibility snapshot mirror",
		"browser release forbidden web sidecar rejection",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(
				issues,
				fmt.Sprintf("browser release report requires %s evidence", required),
			)
		}
	}
	if report.Toolkit == nil || report.Toolkit.ToolkitLevel != "production-widgets-v1" {
		issues = append(issues, "browser release requires production-widgets-v1 toolkit evidence")
	}
	if report.ComponentTree == nil || report.ComponentTree.DynamicLevel != "production-widgets-v1" {
		issues = append(
			issues,
			"browser release requires production-widgets-v1 component tree evidence",
		)
	}
	return issues
}

func validateBrowserSurfaceEvidence(report Report) []string {
	if !isBrowserReleaseReport(report) {
		return nil
	}
	browser := report.BrowserSurface
	if browser == nil {
		return []string{
			"browser_surface evidence is required for wasm32-web browser-canvas release reports",
		}
	}
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: browser.Schema, want: BrowserSurfaceSchemaV1},
		{
			field: "browser_surface_level",
			got:   browser.BrowserSurfaceLevel,
			want:  "browser-canvas-release-v1",
		},
		{field: "release_scope", got: browser.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
		{field: "host_adapter", got: browser.HostAdapter, want: "compiler-owned-browser-canvas-host"},
	} {
		if check.got != check.want {
			issues = append(
				issues,
				fmt.Sprintf(
					"browser_surface %s is %q, want %q",
					check.field,
					check.got,
					check.want,
				),
			)
		}
	}
	if !isSurfaceReleaseFormSource(browser.Source) {
		issues = append(
			issues,
			fmt.Sprintf(
				("browser_surface source path must match "+
					"examples/surface/release/surface_release_form.tetra, got %q"),
				browser.Source,
			),
		)
	}
	if !browser.ProductionClaim {
		issues = append(issues, "browser_surface production_claim must be true")
	}
	if browser.Experimental {
		issues = append(issues, "browser_surface experimental must be false")
	}
	if !browser.CompilerOwnedBoot {
		issues = append(issues, "browser_surface compiler_owned_boot must be true")
	}
	if !browser.DOMHostCanvasOnly {
		issues = append(issues, "browser_surface dom_host_canvas_only must be true")
	}
	if !browser.Canvas.Opened || !browser.Canvas.Readback || !browser.Canvas.Pass {
		issues = append(
			issues,
			"browser_surface canvas requires opened, readback, and pass evidence",
		)
	}
	if browser.Canvas.Width != 560 || browser.Canvas.Height != 420 ||
		browser.Canvas.FrameOrder != 5 {
		issues = append(
			issues,
			fmt.Sprintf(
				"browser_surface canvas frame is order-%d %dx%d, want order-5 560x420",
				browser.Canvas.FrameOrder,
				browser.Canvas.Width,
				browser.Canvas.Height,
			),
		)
	}
	if browser.Canvas.ArtifactKind != "runner-trace" {
		issues = append(
			issues,
			fmt.Sprintf(
				"browser_surface canvas artifact_kind is %q, want runner-trace",
				browser.Canvas.ArtifactKind,
			),
		)
	}
	if !browser.Input.Pointer || !browser.Input.Keyboard || !browser.Input.Text ||
		!browser.Input.Resize ||
		!browser.Input.HostTrace ||
		!browser.Input.Pass {
		issues = append(
			issues,
			"browser_surface input requires pointer, keyboard, text, resize, host_trace, and pass evidence",
		)
	}
	issues = append(issues, validateBrowserSurfaceNativeEvents(browser.Input.NativeEvents)...)
	if browser.Clipboard.Harness != "deterministic-browser-clipboard-v1" {
		issues = append(
			issues,
			fmt.Sprintf(
				"browser_surface clipboard.harness is %q, want deterministic-browser-clipboard-v1",
				browser.Clipboard.Harness,
			),
		)
	}
	if !browser.Clipboard.Read || !browser.Clipboard.Write || !browser.Clipboard.OwnedCopy ||
		!browser.Clipboard.Pass ||
		browser.Clipboard.Bytes <= 0 {
		issues = append(
			issues,
			"browser_surface clipboard requires read, write, owned_copy, positive bytes, and pass evidence",
		)
	}
	if !browser.Composition.Start || !browser.Composition.Update || !browser.Composition.Commit ||
		!browser.Composition.Cancel ||
		!browser.Composition.Pass {
		issues = append(
			issues,
			"browser_surface composition requires start, update, commit, cancel, and pass evidence",
		)
	}
	if !browser.Accessibility.Snapshot || !browser.Accessibility.Mirror ||
		!browser.Accessibility.CompilerOwned ||
		!browser.Accessibility.Bounds ||
		!browser.Accessibility.Focus ||
		!browser.Accessibility.Pass {
		issues = append(
			issues,
			("browser_surface accessibility requires snapshot, mirror, " +
				"compiler_owned, bounds, focus, and pass evidence"),
		)
	}
	issues = append(issues, validateBrowserSurfaceRoles(browser.Accessibility.Roles)...)
	if browser.Accessibility.DOMVisualUI {
		issues = append(issues, "browser_surface accessibility.dom_visual_ui must be false")
	}
	if browser.Accessibility.UserJS {
		issues = append(issues, "browser_surface accessibility.user_js must be false")
	}
	if !browserSurfaceHostTraceContains(browser.HostTraces, "runner-trace") {
		issues = append(
			issues,
			"browser_surface host_traces requires passing runner-trace evidence",
		)
	}
	if !artifactKindContains(report.Artifacts, "runner-trace") {
		issues = append(issues, "browser_surface requires runner-trace artifact evidence")
	}
	guards := browser.NegativeGuards
	for _, check := range []struct {
		field string
		ok    bool
	}{
		{field: "no_dom_app_ui_tree", ok: guards.NoDOMAppUITree},
		{field: "no_user_js_app_logic", ok: guards.NoUserJSAppLogic},
		{field: "no_node_only_promotion", ok: guards.NoNodeOnlyPromotion},
		{field: "no_legacy_sidecars", ok: guards.NoLegacySidecars},
		{field: "no_react_runtime", ok: guards.NoReactRuntime},
		{field: "no_platform_widgets", ok: guards.NoPlatformWidgets},
	} {
		if !check.ok {
			issues = append(
				issues,
				fmt.Sprintf("browser_surface negative_guards.%s must be true", check.field),
			)
		}
	}
	return issues
}

func validateBrowserSurfaceNativeEvents(events []string) []string {
	var issues []string
	for _, required := range []string{"pointerup", "keydown", "beforeinput", "resize"} {
		if !stringSliceContainsFold(events, required) {
			issues = append(
				issues,
				fmt.Sprintf("browser_surface input.native_events requires %s", required),
			)
		}
	}
	return issues
}

func validateBrowserSurfaceRoles(roles []string) []string {
	var issues []string
	for _, required := range []string{"root", "textbox", "checkbox", "button", "status"} {
		if !stringSliceContainsFold(roles, required) {
			issues = append(
				issues,
				fmt.Sprintf("browser_surface accessibility.roles requires %s", required),
			)
		}
	}
	return issues
}

func browserSurfaceHostTraceContains(
	traces []BrowserSurfaceHostTraceReport,
	artifactKind string,
) bool {
	for _, trace := range traces {
		if strings.TrimSpace(trace.Path) == "" || !trace.Pass {
			continue
		}
		if strings.TrimSpace(trace.ArtifactKind) == artifactKind {
			return true
		}
	}
	return false
}

func isBrowserCanvasHostEvidenceLevel(level string) bool {
	return level == "wasm32-web-browser-canvas-input" ||
		level == "wasm32-web-browser-canvas-release-v1"
}

func isBrowserReleaseReport(report Report) bool {
	if report.HostEvidence.Level == "wasm32-web-browser-canvas-release-v1" {
		return true
	}
	return caseNameContains(report.Cases, "browser release Surface v1 schema")
}

// ---- frame_provenance_validation.go ----

func validateFrameProvenance(report Report) []string {
	var issues []string
	for i, frame := range report.Frames {
		role := normalizeFrameEvidenceToken(frame.EvidenceRole)
		producer := normalizeFrameEvidenceToken(frame.Producer)
		if role != "" && !isKnownFrameEvidenceRole(role) {
			issues = append(
				issues,
				fmt.Sprintf(
					"frame %d evidence_role %q is not supported",
					frame.Order,
					frame.EvidenceRole,
				),
			)
		}
		if frame.Precomputed && !isHostProbeOnlyFrameRole(role) {
			issues = append(
				issues,
				fmt.Sprintf(
					"frame %d precomputed evidence can only be host_probe_only infrastructure evidence",
					frame.Order,
				),
			)
		}
		if !isProductVisualFrameRole(role) {
			continue
		}
		location := fmt.Sprintf("frames[%d]", i)
		if frame.Precomputed {
			issues = append(
				issues,
				fmt.Sprintf(
					"%s product visual frame %d must not be precomputed",
					location,
					frame.Order,
				),
			)
		}
		if producer != "app" {
			issues = append(
				issues,
				fmt.Sprintf(
					"%s product visual frame %d producer is %q, want app",
					location,
					frame.Order,
					frame.Producer,
				),
			)
		}
		if strings.TrimSpace(frame.AppSource) == "" {
			issues = append(
				issues,
				fmt.Sprintf(
					"%s product visual frame %d app_source is required",
					location,
					frame.Order,
				),
			)
		} else if normalizeEvidencePath(frame.AppSource) != normalizeEvidencePath(report.Source) {
			issues = append(issues, fmt.Sprintf((("%s product visual frame %d app_source %q must "+
				"match report ")+
				"source %q"), location, frame.Order, frame.AppSource, report.Source))
		}
		if !validSHA256Digest(frame.MorphRecipeHash) {
			issues = append(
				issues,
				fmt.Sprintf(
					"%s product visual frame %d morph_recipe_hash must be sha256 evidence",
					location,
					frame.Order,
				),
			)
		}
		if !validSHA256Digest(frame.BlockSceneHash) {
			issues = append(
				issues,
				fmt.Sprintf(
					"%s product visual frame %d block_scene_hash must be sha256 evidence",
					location,
					frame.Order,
				),
			)
		} else if report.BlockSceneSnapshot == nil {
			issues = append(issues, fmt.Sprintf(("%s product visual frame %d requires block_scene_snapshot "+
				"evidence"), location, frame.Order))
		} else if frame.BlockSceneHash != report.BlockSceneSnapshot.BlockSceneHash {
			issues = append(issues, fmt.Sprintf(("%s product visual frame %d block_scene_hash must match "+
				"block_scene_snapshot.block_scene_hash"), location, frame.Order))
		}
		if !validSHA256Digest(frame.RenderCommandStreamHash) {
			issues = append(
				issues,
				fmt.Sprintf(
					"%s product visual frame %d render_command_stream_hash must be sha256 evidence",
					location,
					frame.Order,
				),
			)
		} else if report.RenderCommandStream == nil {
			issues = append(issues, fmt.Sprintf((("%s product visual frame %d requires render_"+
				"command_stream ")+
				"evidence"), location, frame.Order))
		} else if frame.RenderCommandStreamHash != report.RenderCommandStream.CommandStreamHash {
			issues = append(issues, fmt.Sprintf((("%s product visual frame %d render_command_"+
				"stream_hash must ")+
				"match render_command_stream.command_stream_hash"), location, frame.Order))
		}
	}
	return issues
}

func validateBlockSystemFrameProvenance(
	frame BlockSystemFrameReport,
	runtimeFrame FrameReport,
) []string {
	var issues []string
	role := normalizeFrameEvidenceToken(frame.EvidenceRole)
	producer := normalizeFrameEvidenceToken(frame.Producer)
	if role != "" && !isKnownFrameEvidenceRole(role) {
		issues = append(
			issues,
			fmt.Sprintf(
				"block_system frame %d evidence_role %q is not supported",
				frame.Order,
				frame.EvidenceRole,
			),
		)
	}
	if frame.Precomputed && !isHostProbeOnlyFrameRole(role) {
		issues = append(
			issues,
			fmt.Sprintf(
				("block_system frame %d precomputed evidence can only be "+
					"host_probe_only infrastructure evidence"),
				frame.Order,
			),
		)
	}
	if isProductVisualFrameRole(role) {
		if frame.Precomputed {
			issues = append(
				issues,
				fmt.Sprintf(
					"block_system frame %d product visual evidence must not be precomputed",
					frame.Order,
				),
			)
		}
		if producer != "app" {
			issues = append(
				issues,
				fmt.Sprintf(
					"block_system frame %d product visual producer is %q, want app",
					frame.Order,
					frame.Producer,
				),
			)
		}
	}
	if strings.TrimSpace(frame.EvidenceRole) != "" &&
		strings.TrimSpace(runtimeFrame.EvidenceRole) != "" &&
		role != normalizeFrameEvidenceToken(runtimeFrame.EvidenceRole) {
		issues = append(
			issues,
			fmt.Sprintf(
				"block_system frame %d evidence_role must match runtime frame evidence_role",
				frame.Order,
			),
		)
	}
	if strings.TrimSpace(frame.Producer) != "" && strings.TrimSpace(runtimeFrame.Producer) != "" &&
		producer != normalizeFrameEvidenceToken(runtimeFrame.Producer) {
		issues = append(
			issues,
			fmt.Sprintf(
				"block_system frame %d producer must match runtime frame producer",
				frame.Order,
			),
		)
	}
	return issues
}

func normalizeFrameEvidenceToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func isKnownFrameEvidenceRole(role string) bool {
	switch normalizeFrameEvidenceToken(role) {
	case "product_visual", "host_probe_only", "runtime_smoke", "infrastructure_probe",
		"native_surface_live_frame":
		return true
	default:
		return false
	}
}

func isProductVisualFrameRole(role string) bool {
	return normalizeFrameEvidenceToken(role) == "product_visual"
}

func isHostProbeOnlyFrameRole(role string) bool {
	switch normalizeFrameEvidenceToken(role) {
	case "host_probe_only", "infrastructure_probe":
		return true
	default:
		return false
	}
}

// ---- performance_validation.go ----

type SurfacePerformanceBudgetReport struct {
	Schema            string                              `json:"schema"`
	Model             string                              `json:"model"`
	ReleaseScope      string                              `json:"release_scope"`
	Source            string                              `json:"source"`
	Target            string                              `json:"target"`
	Runtime           string                              `json:"runtime"`
	ProductionClaim   bool                                `json:"production_claim"`
	Experimental      bool                                `json:"experimental"`
	GitHead           string                              `json:"git_head"`
	PerformanceClaim  string                              `json:"performance_claim"`
	Startup           SurfaceStartupBudgetReport          `json:"startup"`
	Frame             SurfaceFrameBudgetReport            `json:"frame"`
	Scene             SurfaceSceneBudgetReport            `json:"scene"`
	Memory            SurfaceMemoryBudgetReport           `json:"memory"`
	Binary            SurfaceBinaryBudgetReport           `json:"binary"`
	CPUPowerProxy     SurfaceCPUPowerProxyReport          `json:"cpu_power_proxy"`
	Cache             SurfaceCacheBudgetReport            `json:"cache"`
	Methodology       SurfacePerformanceMethodologyReport `json:"methodology"`
	UnsupportedClaims []string                            `json:"unsupported_claims"`
	NegativeGuards    SurfacePerformanceNegativeGuards    `json:"negative_guards"`
}

type SurfaceStartupBudgetReport struct {
	LaunchToFirstFrameMS int    `json:"launch_to_first_frame_ms"`
	BudgetMS             int    `json:"budget_ms"`
	Trace                string `json:"trace"`
	Pass                 bool   `json:"pass"`
}

type SurfaceFrameBudgetReport struct {
	FrameCount    int  `json:"frame_count"`
	P50BuildMS    int  `json:"p50_build_ms"`
	P95BuildMS    int  `json:"p95_build_ms"`
	P50PresentMS  int  `json:"p50_present_ms"`
	P95PresentMS  int  `json:"p95_present_ms"`
	BudgetMS      int  `json:"budget_ms"`
	IdleLoopCount int  `json:"idle_loop_count"`
	WorkLoopCount int  `json:"work_loop_count"`
	Pass          bool `json:"pass"`
}

type SurfaceSceneBudgetReport struct {
	BlockCount           int `json:"block_count"`
	RecipeExpansionCount int `json:"recipe_expansion_count"`
	PaintCommandCount    int `json:"paint_command_count"`
	LayoutPassCount      int `json:"layout_pass_count"`
	TextRunCount         int `json:"text_run_count"`
}

type SurfaceMemoryBudgetReport struct {
	GlyphCacheBytes        int  `json:"glyph_cache_bytes"`
	AssetCacheBytes        int  `json:"asset_cache_bytes"`
	LayoutCacheBytes       int  `json:"layout_cache_bytes"`
	PaintCacheBytes        int  `json:"paint_cache_bytes"`
	FramebufferPeakBytes   int  `json:"framebuffer_peak_bytes"`
	FramebufferTotalBytes  int  `json:"framebuffer_total_bytes"`
	RSSMeasured            bool `json:"rss_measured"`
	PeakRSSBytes           int  `json:"peak_rss_bytes"`
	AllocationCount        int  `json:"allocation_count"`
	AllocationBytes        int  `json:"allocation_bytes"`
	BoundedCaches          bool `json:"bounded_caches"`
	UnboundedCacheRejected bool `json:"unbounded_cache_rejected"`
	Pass                   bool `json:"pass"`
}

type SurfaceBinaryBudgetReport struct {
	ArtifactPath string `json:"artifact_path"`
	SizeBytes    int    `json:"size_bytes"`
	BudgetBytes  int    `json:"budget_bytes"`
	Pass         bool   `json:"pass"`
}

type SurfaceCPUPowerProxyReport struct {
	IdleLoopCount     int  `json:"idle_loop_count"`
	WorkLoopCount     int  `json:"work_loop_count"`
	IdleFrameCount    int  `json:"idle_frame_count"`
	WorkFrameCount    int  `json:"work_frame_count"`
	RealPowerMeasured bool `json:"real_power_measured"`
	Pass              bool `json:"pass"`
}

type SurfaceCacheBudgetReport struct {
	GlyphCacheBudgetBytes  int    `json:"glyph_cache_budget_bytes"`
	AssetCacheBudgetBytes  int    `json:"asset_cache_budget_bytes"`
	LayoutCacheBudgetBytes int    `json:"layout_cache_budget_bytes"`
	PaintCacheBudgetBytes  int    `json:"paint_cache_budget_bytes"`
	TotalCacheBytes        int    `json:"total_cache_bytes"`
	TotalCacheBudgetBytes  int    `json:"total_cache_budget_bytes"`
	Eviction               string `json:"eviction"`
	Pass                   bool   `json:"pass"`
}

type SurfacePerformanceMethodologyReport struct {
	Kind                                   string `json:"kind"`
	ElectronComparison                     string `json:"electron_comparison"`
	OfficialBenchmark                      bool   `json:"official_benchmark"`
	CrossMachine                           bool   `json:"cross_machine"`
	FairComparisonRequiredForElectronClaim bool   `json:"fair_comparison_required_for_electron_claim"`
}

type SurfacePerformanceNegativeGuards struct {
	BoundedCaches             bool `json:"bounded_caches"`
	UnboundedCacheRejected    bool `json:"unbounded_cache_rejected"`
	StaleReportRejected       bool `json:"stale_report_rejected"`
	NoFasterThanElectronClaim bool `json:"no_faster_than_electron_claim"`
	NoBenchmarkParityClaim    bool `json:"no_benchmark_parity_claim"`
	PeakMemoryFieldRequired   bool `json:"peak_memory_field_required"`
	NoOfficialBenchmarkClaim  bool `json:"no_official_benchmark_claim"`
}

func ValidatePerformanceBudgetReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	switch schema {
	case PerformanceBudgetSchemaV1:
		var report SurfacePerformanceBudgetReport
		if err := decodeStrict(raw, &report); err != nil {
			return err
		}
		issues := validateSurfacePerformanceBudgetReport(report, nil, "")
		if !performanceBudgetPeakRSSFieldPresent(raw, false) {
			issues = append(
				issues,
				"surface_performance_budget memory peak_rss_bytes field is required",
			)
		}
		if len(issues) > 0 {
			return errors.New(strings.Join(issues, "; "))
		}
		return nil
	case SchemaV1:
		var report Report
		if err := decodeStrict(raw, &report); err != nil {
			return err
		}
		issues := validateSurfacePerformanceBudgetEvidence(report)
		if report.SurfacePerformanceBudget != nil &&
			!performanceBudgetPeakRSSFieldPresent(raw, true) {
			issues = append(
				issues,
				"surface_performance_budget memory peak_rss_bytes field is required",
			)
		}
		if len(issues) > 0 {
			return errors.New(strings.Join(issues, "; "))
		}
		return nil
	default:
		return fmt.Errorf(
			"schema is %q, want %q or %q",
			schema,
			PerformanceBudgetSchemaV1,
			SchemaV1,
		)
	}
}

func validateSurfacePerformanceBudgetEvidence(report Report) []string {
	if report.SurfacePerformanceBudget == nil {
		if isLinuxAppShellReport(report) {
			return []string{
				"surface_performance_budget evidence is required for linux app-shell reports",
			}
		}
		return nil
	}
	return validateSurfacePerformanceBudgetReport(
		*report.SurfacePerformanceBudget,
		&report,
		report.Source,
	)
}

func validateSurfacePerformanceBudgetReport(
	budget SurfacePerformanceBudgetReport,
	runtime *Report,
	source string,
) []string {
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: budget.Schema, want: PerformanceBudgetSchemaV1},
		{field: "model", got: budget.Model, want: "surface-performance-budget-v1"},
		{field: "release_scope", got: budget.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
	} {
		if check.got != check.want {
			issues = append(
				issues,
				fmt.Sprintf(
					"surface_performance_budget %s is %q, want %q",
					check.field,
					check.got,
					check.want,
				),
			)
		}
	}
	if strings.TrimSpace(source) != "" &&
		normalizeEvidencePath(budget.Source) != normalizeEvidencePath(source) {
		issues = append(
			issues,
			fmt.Sprintf(
				"surface_performance_budget source %q must match report source %q",
				budget.Source,
				source,
			),
		)
	}
	if strings.TrimSpace(budget.Source) == "" {
		issues = append(issues, "surface_performance_budget source is required")
	}
	if !isSupportedRuntimeTarget(budget.Target) {
		issues = append(
			issues,
			fmt.Sprintf(
				"surface_performance_budget target is %q, want headless, linux-x64, or wasm32-web",
				budget.Target,
			),
		)
	}
	if !isSupportedRuntimeName(budget.Runtime) {
		issues = append(
			issues,
			fmt.Sprintf(
				("surface_performance_budget runtime is %q, want "+
					"surface-headless, surface-linux-x64, or surface-wasm32-web"),
				budget.Runtime,
			),
		)
	}
	if runtime != nil {
		if budget.Target != runtime.Target {
			issues = append(
				issues,
				fmt.Sprintf(
					"surface_performance_budget target %q must match report target %q",
					budget.Target,
					runtime.Target,
				),
			)
		}
		if budget.Runtime != runtime.Runtime {
			issues = append(
				issues,
				fmt.Sprintf(
					"surface_performance_budget runtime %q must match report runtime %q",
					budget.Runtime,
					runtime.Runtime,
				),
			)
		}
	}
	if !budget.ProductionClaim {
		issues = append(issues, "surface_performance_budget production_claim must be true")
	}
	if budget.Experimental {
		issues = append(issues, "surface_performance_budget experimental must be false")
	}
	if !isGitHead(budget.GitHead) {
		issues = append(
			issues,
			"surface_performance_budget git_head must be a 40-character hex commit",
		)
	}
	if strings.TrimSpace(budget.PerformanceClaim) != "none" {
		issues = append(
			issues,
			fmt.Sprintf(
				"surface_performance_budget performance_claim is %q, want none",
				budget.PerformanceClaim,
			),
		)
	}
	issues = append(
		issues,
		forbiddenBlockPerformanceClaimIssues(
			"surface_performance_budget performance_claim",
			budget.PerformanceClaim,
		)...)
	issues = append(
		issues,
		forbiddenBlockPerformanceClaimIssues(
			"surface_performance_budget methodology",
			budget.Methodology.ElectronComparison,
		)...)
	issues = append(issues, validateSurfaceStartupBudget(budget.Startup)...)
	issues = append(issues, validateSurfaceFrameBudget(budget.Frame)...)
	issues = append(issues, validateSurfaceSceneBudget(budget.Scene, runtime)...)
	issues = append(issues, validateSurfaceMemoryBudget(budget.Memory, runtime)...)
	issues = append(issues, validateSurfaceBinaryBudget(budget.Binary)...)
	issues = append(issues, validateSurfaceCPUPowerProxy(budget.CPUPowerProxy)...)
	issues = append(issues, validateSurfaceCacheBudget(budget.Cache, budget.Memory)...)
	issues = append(issues, validateSurfacePerformanceMethodology(budget.Methodology)...)
	issues = append(
		issues,
		validateSurfacePerformanceUnsupportedClaims(budget.UnsupportedClaims)...)
	issues = append(issues, validateSurfacePerformanceNegativeGuards(budget.NegativeGuards)...)
	return issues
}

func validateSurfaceStartupBudget(startup SurfaceStartupBudgetReport) []string {
	var issues []string
	if startup.LaunchToFirstFrameMS <= 0 {
		issues = append(
			issues,
			"surface_performance_budget startup launch_to_first_frame_ms must be positive",
		)
	}
	if startup.BudgetMS <= 0 {
		issues = append(issues, "surface_performance_budget startup budget_ms must be positive")
	}
	if startup.BudgetMS > 0 && startup.LaunchToFirstFrameMS > startup.BudgetMS {
		issues = append(
			issues,
			fmt.Sprintf(
				"surface_performance_budget startup launch_to_first_frame_ms %d exceeds budget_ms %d",
				startup.LaunchToFirstFrameMS,
				startup.BudgetMS,
			),
		)
	}
	if strings.TrimSpace(startup.Trace) == "" {
		issues = append(issues, "surface_performance_budget startup trace is required")
	}
	if !startup.Pass {
		issues = append(issues, "surface_performance_budget startup pass must be true")
	}
	return issues
}

func validateSurfaceFrameBudget(frame SurfaceFrameBudgetReport) []string {
	var issues []string
	if frame.FrameCount <= 0 {
		issues = append(issues, "surface_performance_budget frame frame_count must be positive")
	}
	for _, check := range []struct {
		field string
		value int
	}{
		{field: "p50_build_ms", value: frame.P50BuildMS},
		{field: "p95_build_ms", value: frame.P95BuildMS},
		{field: "p50_present_ms", value: frame.P50PresentMS},
		{field: "p95_present_ms", value: frame.P95PresentMS},
		{field: "budget_ms", value: frame.BudgetMS},
	} {
		if check.value <= 0 {
			issues = append(
				issues,
				fmt.Sprintf("surface_performance_budget frame %s must be positive", check.field),
			)
		}
	}
	if frame.P95BuildMS < frame.P50BuildMS {
		issues = append(
			issues,
			"surface_performance_budget frame p95_build_ms must be >= p50_build_ms",
		)
	}
	if frame.P95PresentMS < frame.P50PresentMS {
		issues = append(
			issues,
			"surface_performance_budget frame p95_present_ms must be >= p50_present_ms",
		)
	}
	if frame.BudgetMS > 0 &&
		(frame.P95BuildMS > frame.BudgetMS || frame.P95PresentMS > frame.BudgetMS) {
		issues = append(
			issues,
			"surface_performance_budget frame p95 build/present must fit within budget_ms",
		)
	}
	if frame.IdleLoopCount < 0 || frame.WorkLoopCount < 0 {
		issues = append(
			issues,
			"surface_performance_budget frame idle/work loop counts must be non-negative",
		)
	}
	if !frame.Pass {
		issues = append(issues, "surface_performance_budget frame pass must be true")
	}
	return issues
}

func validateSurfaceSceneBudget(scene SurfaceSceneBudgetReport, runtime *Report) []string {
	var issues []string
	if scene.BlockCount <= 0 {
		issues = append(issues, "surface_performance_budget scene block_count must be positive")
	}
	for _, check := range []struct {
		field string
		value int
	}{
		{field: "recipe_expansion_count", value: scene.RecipeExpansionCount},
		{field: "paint_command_count", value: scene.PaintCommandCount},
		{field: "layout_pass_count", value: scene.LayoutPassCount},
		{field: "text_run_count", value: scene.TextRunCount},
	} {
		if check.value < 0 {
			issues = append(
				issues,
				fmt.Sprintf(
					"surface_performance_budget scene %s must be non-negative",
					check.field,
				),
			)
		}
	}
	if runtime != nil && len(runtime.Components) > 0 && scene.BlockCount < len(runtime.Components) {
		issues = append(
			issues,
			fmt.Sprintf(
				"surface_performance_budget scene block_count = %d, want at least component count %d",
				scene.BlockCount,
				len(runtime.Components),
			),
		)
	}
	return issues
}

func validateSurfaceMemoryBudget(memory SurfaceMemoryBudgetReport, runtime *Report) []string {
	var issues []string
	for _, check := range []struct {
		field string
		value int
	}{
		{field: "glyph_cache_bytes", value: memory.GlyphCacheBytes},
		{field: "asset_cache_bytes", value: memory.AssetCacheBytes},
		{field: "layout_cache_bytes", value: memory.LayoutCacheBytes},
		{field: "paint_cache_bytes", value: memory.PaintCacheBytes},
		{field: "allocation_count", value: memory.AllocationCount},
		{field: "allocation_bytes", value: memory.AllocationBytes},
	} {
		if check.value < 0 {
			issues = append(
				issues,
				fmt.Sprintf(
					"surface_performance_budget memory %s must be non-negative",
					check.field,
				),
			)
		}
	}
	if memory.FramebufferPeakBytes <= 0 {
		issues = append(
			issues,
			"surface_performance_budget memory framebuffer_peak_bytes must be positive",
		)
	}
	if memory.FramebufferTotalBytes < memory.FramebufferPeakBytes {
		issues = append(
			issues,
			"surface_performance_budget memory framebuffer_total_bytes must be >= framebuffer_peak_bytes",
		)
	}
	if runtime != nil && len(runtime.Frames) > 0 {
		peak, total := blockFramebufferByteTotals(runtime.Frames)
		if memory.FramebufferPeakBytes < peak {
			issues = append(
				issues,
				fmt.Sprintf(
					"surface_performance_budget memory framebuffer_peak_bytes = %d, want at least runtime peak %d",
					memory.FramebufferPeakBytes,
					peak,
				),
			)
		}
		if memory.FramebufferTotalBytes < total {
			issues = append(
				issues,
				fmt.Sprintf(
					("surface_performance_budget memory framebuffer_total_bytes = "+
						"%d, want at least runtime total %d"),
					memory.FramebufferTotalBytes,
					total,
				),
			)
		}
	}
	if memory.RSSMeasured {
		if memory.PeakRSSBytes <= 0 {
			issues = append(
				issues,
				"surface_performance_budget memory peak_rss_bytes must be positive when rss_measured=true",
			)
		}
	} else if memory.PeakRSSBytes != 0 {
		issues = append(issues, ("surface_performance_budget memory peak_rss_bytes must be 0 " +
			"when rss_measured=false"))
	}
	if memory.AllocationCount <= 0 {
		issues = append(
			issues,
			"surface_performance_budget memory allocation_count must be positive",
		)
	}
	cacheBytes := memory.GlyphCacheBytes + memory.AssetCacheBytes + memory.LayoutCacheBytes + memory.PaintCacheBytes
	if memory.AllocationBytes < memory.FramebufferPeakBytes+cacheBytes {
		issues = append(
			issues,
			fmt.Sprintf(
				("surface_performance_budget memory allocation_bytes = %d, "+
					"want at least framebuffer peak plus caches %d"),
				memory.AllocationBytes,
				memory.FramebufferPeakBytes+cacheBytes,
			),
		)
	}
	if !memory.BoundedCaches {
		issues = append(issues, "surface_performance_budget memory bounded_caches must be true")
	}
	if !memory.UnboundedCacheRejected {
		issues = append(
			issues,
			"surface_performance_budget memory unbounded_cache_rejected must be true",
		)
	}
	if !memory.Pass {
		issues = append(issues, "surface_performance_budget memory pass must be true")
	}
	return issues
}

func validateSurfaceBinaryBudget(binary SurfaceBinaryBudgetReport) []string {
	var issues []string
	if strings.TrimSpace(binary.ArtifactPath) == "" {
		issues = append(issues, "surface_performance_budget binary artifact_path is required")
	}
	if binary.SizeBytes <= 0 {
		issues = append(issues, "surface_performance_budget binary size_bytes must be positive")
	}
	if binary.BudgetBytes <= 0 {
		issues = append(issues, "surface_performance_budget binary budget_bytes must be positive")
	}
	if binary.BudgetBytes > 0 && binary.SizeBytes > binary.BudgetBytes {
		issues = append(
			issues,
			fmt.Sprintf(
				"surface_performance_budget binary size_bytes %d exceeds budget_bytes %d",
				binary.SizeBytes,
				binary.BudgetBytes,
			),
		)
	}
	if !binary.Pass {
		issues = append(issues, "surface_performance_budget binary pass must be true")
	}
	return issues
}

func validateSurfaceCPUPowerProxy(proxy SurfaceCPUPowerProxyReport) []string {
	var issues []string
	for _, check := range []struct {
		field string
		value int
	}{
		{field: "idle_loop_count", value: proxy.IdleLoopCount},
		{field: "work_loop_count", value: proxy.WorkLoopCount},
		{field: "idle_frame_count", value: proxy.IdleFrameCount},
		{field: "work_frame_count", value: proxy.WorkFrameCount},
	} {
		if check.value < 0 {
			issues = append(
				issues,
				fmt.Sprintf(
					"surface_performance_budget cpu_power_proxy %s must be non-negative",
					check.field,
				),
			)
		}
	}
	if proxy.RealPowerMeasured {
		issues = append(
			issues,
			("surface_performance_budget cpu_power_proxy " +
				"real_power_measured must be false unless a real power " +
				"harness is attached"),
		)
	}
	if !proxy.Pass {
		issues = append(issues, "surface_performance_budget cpu_power_proxy pass must be true")
	}
	return issues
}

func validateSurfaceCacheBudget(
	cache SurfaceCacheBudgetReport,
	memory SurfaceMemoryBudgetReport,
) []string {
	var issues []string
	for _, check := range []struct {
		field string
		value int
	}{
		{field: "glyph_cache_budget_bytes", value: cache.GlyphCacheBudgetBytes},
		{field: "asset_cache_budget_bytes", value: cache.AssetCacheBudgetBytes},
		{field: "layout_cache_budget_bytes", value: cache.LayoutCacheBudgetBytes},
		{field: "paint_cache_budget_bytes", value: cache.PaintCacheBudgetBytes},
		{field: "total_cache_budget_bytes", value: cache.TotalCacheBudgetBytes},
	} {
		if check.value <= 0 {
			issues = append(
				issues,
				fmt.Sprintf("surface_performance_budget cache %s must be positive", check.field),
			)
		}
	}
	expectedTotal := memory.GlyphCacheBytes + memory.AssetCacheBytes + memory.LayoutCacheBytes + memory.PaintCacheBytes
	if cache.TotalCacheBytes != expectedTotal {
		issues = append(
			issues,
			fmt.Sprintf(
				"surface_performance_budget cache total_cache_bytes = %d, want memory cache total %d",
				cache.TotalCacheBytes,
				expectedTotal,
			),
		)
	}
	if cache.TotalCacheBudgetBytes > 0 && cache.TotalCacheBytes > cache.TotalCacheBudgetBytes {
		issues = append(
			issues,
			"surface_performance_budget cache total_cache_bytes must fit within total_cache_budget_bytes",
		)
	}
	if strings.TrimSpace(cache.Eviction) == "" {
		issues = append(issues, "surface_performance_budget cache eviction policy is required")
	}
	if !cache.Pass {
		issues = append(issues, "surface_performance_budget cache pass must be true")
	}
	return issues
}

func validateSurfacePerformanceMethodology(
	methodology SurfacePerformanceMethodologyReport,
) []string {
	var issues []string
	if methodology.Kind != "local-deterministic-budget-v1" {
		issues = append(
			issues,
			fmt.Sprintf(
				"surface_performance_budget methodology kind is %q, want local-deterministic-budget-v1",
				methodology.Kind,
			),
		)
	}
	if strings.TrimSpace(methodology.ElectronComparison) != "none" {
		issues = append(
			issues,
			fmt.Sprintf(
				"surface_performance_budget methodology electron_comparison is %q, want none",
				methodology.ElectronComparison,
			),
		)
	}
	if methodology.OfficialBenchmark {
		issues = append(
			issues,
			"surface_performance_budget methodology official_benchmark must be false",
		)
	}
	if methodology.CrossMachine {
		issues = append(
			issues,
			"surface_performance_budget methodology cross_machine must be false",
		)
	}
	if !methodology.FairComparisonRequiredForElectronClaim {
		issues = append(
			issues,
			("surface_performance_budget methodology requires " +
				"fair_comparison_required_for_electron_claim=true"),
		)
	}
	return issues
}

func validateSurfacePerformanceUnsupportedClaims(claims []string) []string {
	var issues []string
	for _, claim := range claims {
		issues = append(
			issues,
			forbiddenBlockPerformanceClaimIssues(
				"surface_performance_budget unsupported_claims",
				claim,
			)...)
	}
	for _, required := range []string{
		"faster-than-electron",
		"lower-power-than-electron",
		"official-benchmark-result",
		"cross-machine-benchmark",
		"electron-parity-performance",
	} {
		if !containsExactText(claims, required) {
			issues = append(
				issues,
				fmt.Sprintf("surface_performance_budget unsupported_claims missing %q", required),
			)
		}
	}
	return issues
}

func validateSurfacePerformanceNegativeGuards(guards SurfacePerformanceNegativeGuards) []string {
	if guards.BoundedCaches &&
		guards.UnboundedCacheRejected &&
		guards.StaleReportRejected &&
		guards.NoFasterThanElectronClaim &&
		guards.NoBenchmarkParityClaim &&
		guards.PeakMemoryFieldRequired &&
		guards.NoOfficialBenchmarkClaim {
		return nil
	}
	return []string{
		("surface_performance_budget negative_guards must require " +
			"bounded caches, stale report rejection, peak memory field, " +
			"and no unsupported Electron benchmark claims"),
	}
}

func containsExactText(values []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == want {
			return true
		}
	}
	return false
}

func performanceBudgetPeakRSSFieldPresent(raw []byte, embedded bool) bool {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return false
	}
	var budgetRaw json.RawMessage
	if embedded {
		var ok bool
		budgetRaw, ok = root["surface_performance_budget"]
		if !ok {
			return false
		}
	} else {
		budgetRaw = raw
	}
	var budget map[string]json.RawMessage
	if err := json.Unmarshal(budgetRaw, &budget); err != nil {
		return false
	}
	var memory map[string]json.RawMessage
	if err := json.Unmarshal(budget["memory"], &memory); err != nil {
		return false
	}
	_, ok := memory["peak_rss_bytes"]
	return ok
}

// ---- raster_evidence_validation.go ----

func validateRasterProof(
	prefix string,
	want string,
	format string,
	hash string,
	width int,
	height int,
	coverage int,
	markerOnly bool,
) []string {
	var issues []string
	format = strings.TrimSpace(format)
	want = strings.TrimSpace(want)
	if markerOnly {
		issues = append(
			issues,
			fmt.Sprintf("%s marker_only must be false for %s raster evidence", prefix, want),
		)
	}
	if strings.Contains(strings.ToLower(format), "marker") {
		issues = append(
			issues,
			fmt.Sprintf("%s raster_format %q must not be marker evidence", prefix, format),
		)
	}
	if want != "" && format != want {
		issues = append(
			issues,
			fmt.Sprintf("%s raster_format is %q, want %s", prefix, format, want),
		)
	}
	if !validSHA256Digest(hash) {
		issues = append(issues, fmt.Sprintf("%s raster_hash must be sha256 evidence", prefix))
	}
	if width <= 0 || height <= 0 {
		issues = append(issues, fmt.Sprintf("%s raster dimensions must be positive", prefix))
	}
	if coverage <= 0 {
		issues = append(issues, fmt.Sprintf("%s raster_coverage must be positive", prefix))
	}
	if width > 0 && height > 0 && coverage > width*height {
		issues = append(
			issues,
			fmt.Sprintf(
				"%s raster_coverage %d exceeds raster dimensions %dx%d",
				prefix,
				coverage,
				width,
				height,
			),
		)
	}
	return issues
}

// ---- runtime_validation.go ----

func validateHostEvidence(report Report) []string {
	var issues []string
	evidence := report.HostEvidence
	if strings.TrimSpace(evidence.Level) == "" {
		issues = append(issues, "host_evidence.level is required")
	}
	if strings.TrimSpace(evidence.Backend) == "" {
		issues = append(issues, "host_evidence.backend is required")
	}
	if evidence.UserFacingPlatformWidgets {
		issues = append(issues, "host_evidence must not expose user-facing platform widgets")
	}

	switch report.Target {
	case "headless":
		if evidence.Level != "deterministic-headless" {
			issues = append(
				issues,
				fmt.Sprintf(
					"headless host_evidence.level is %q, want deterministic-headless",
					evidence.Level,
				),
			)
		}
		if evidence.Backend != "software-rgba" {
			issues = append(
				issues,
				fmt.Sprintf(
					"headless host_evidence.backend is %q, want software-rgba",
					evidence.Backend,
				),
			)
		}
		if !evidence.Framebuffer {
			issues = append(issues, "headless host_evidence requires framebuffer=true")
		}
		if evidence.RealWindow || evidence.NativeInput {
			issues = append(
				issues,
				"headless host_evidence must not claim real_window or native_input",
			)
		}
	case "linux-x64":
		switch evidence.Level {
		case "linux-x64-memfd-starter":
			if evidence.Backend != "memfd-rgba" {
				issues = append(
					issues,
					fmt.Sprintf(
						"linux-x64 memfd starter host_evidence.backend is %q, want memfd-rgba",
						evidence.Backend,
					),
				)
			}
			if !evidence.Framebuffer {
				issues = append(
					issues,
					"linux-x64 memfd starter host_evidence requires framebuffer=true",
				)
			}
			if evidence.RealWindow || evidence.NativeInput {
				issues = append(
					issues,
					"linux-x64 memfd starter host_evidence must not claim real_window or native_input",
				)
			}
		case "linux-x64-real-window":
			if !evidence.Framebuffer || !evidence.RealWindow || !evidence.NativeInput {
				issues = append(
					issues,
					("linux-x64 real-window host_evidence requires " +
						"framebuffer=true, real_window=true, and native_input=true"),
				)
			}
			if evidence.Backend == "memfd-rgba" || evidence.Backend == "software-rgba" ||
				evidence.Backend == "node-surface-host" {
				issues = append(
					issues,
					fmt.Sprintf(
						"linux-x64 real-window host_evidence.backend %q is not real-window evidence",
						evidence.Backend,
					),
				)
			}
			if !hasAppProcessWithExpectedExit(
				report.Processes,
				"surface linux-x64 real-window probe",
				42,
			) {
				issues = append(
					issues,
					"linux-x64 real-window host_evidence requires a Surface real-window probe app exiting 42",
				)
			}
			if !caseNameContains(report.Cases, "linux-x64 real-window surface") {
				issues = append(
					issues,
					"linux-x64 real-window host_evidence requires linux-x64 real-window surface case evidence",
				)
			}
			if !caseNameContains(report.Cases, "linux-x64 native input event pump") {
				issues = append(
					issues,
					"linux-x64 real-window host_evidence requires linux-x64 native input event pump case evidence",
				)
			}
		case NativeSurfaceHostLevelLinuxX64:
			if evidence.Backend != NativeSurfaceHostBackendWayland {
				issues = append(
					issues,
					fmt.Sprintf(
						"linux-x64 native Surface host host_evidence.backend is %q, want %s",
						evidence.Backend,
						NativeSurfaceHostBackendWayland,
					),
				)
			}
			if !evidence.Framebuffer || !evidence.RealWindow || !evidence.NativeInput {
				issues = append(
					issues,
					("linux-x64 native Surface host_evidence requires " +
						"framebuffer=true, real_window=true, and native_input=true"),
				)
			}
			for _, requiredCase := range []string{
				"native Surface host Wayland live window",
				"native Surface host app loop observed",
				"native Surface host close event",
				"native Surface host pointer input",
				"native Surface host keyboard input",
				"native Surface host frame presented by running app",
			} {
				if !caseNameContains(report.Cases, requiredCase) {
					issues = append(
						issues,
						fmt.Sprintf(
							"linux-x64 native Surface host_evidence requires %s case evidence",
							requiredCase,
						),
					)
				}
			}
		case "linux-x64-release-window-v1":
			if evidence.Backend != "wayland-shm-rgba-release-v1" {
				issues = append(
					issues,
					fmt.Sprintf(
						"linux release host_evidence.backend is %q, want wayland-shm-rgba-release-v1",
						evidence.Backend,
					),
				)
			}
			if !evidence.Framebuffer || !evidence.RealWindow || !evidence.NativeInput {
				issues = append(
					issues,
					("linux release host_evidence requires framebuffer=true, " +
						"real_window=true, and native_input=true"),
				)
			}
			if !evidence.TextInput {
				issues = append(issues, "linux release host_evidence.text_input must be true")
			}
			if !evidence.Clipboard {
				issues = append(issues, "linux release host_evidence.clipboard must be true")
			}
			if !evidence.Composition {
				issues = append(issues, "linux release host_evidence.composition must be true")
			}
			if !evidence.AccessibilityBridge {
				issues = append(
					issues,
					"linux release host_evidence.accessibility_bridge must be true",
				)
			}
			if !caseNameContains(report.Cases, "linux release real window presented frame") {
				issues = append(
					issues,
					"linux release host_evidence requires real window presented frame case evidence",
				)
			}
			if !caseNameContains(report.Cases, "linux release accessibility bridge probe") {
				issues = append(
					issues,
					"linux release host_evidence requires accessibility bridge probe case evidence",
				)
			}
		default:
			issues = append(
				issues,
				fmt.Sprintf(
					("linux-x64 host_evidence.level is %q, want "+
						"linux-x64-memfd-starter, linux-x64-real-window, "+
						"linux-x64-native-surface-host-v1, or "+
						"linux-x64-release-window-v1"),
					evidence.Level,
				),
			)
		}
	case "wasm32-web":
		switch evidence.Level {
		case "wasm32-web-compiler-owned-loader":
			if evidence.Backend != "node-surface-host" {
				issues = append(
					issues,
					fmt.Sprintf(
						"wasm32-web starter host_evidence.backend is %q, want node-surface-host",
						evidence.Backend,
					),
				)
			}
			if !evidence.Framebuffer {
				issues = append(
					issues,
					"wasm32-web starter host_evidence requires framebuffer=true",
				)
			}
			if evidence.RealWindow || evidence.NativeInput {
				issues = append(
					issues,
					"wasm32-web starter host_evidence must not claim browser canvas native input",
				)
			}
		case "wasm32-web-browser-canvas-input":
			if evidence.Backend != "browser-canvas-rgba" {
				issues = append(
					issues,
					fmt.Sprintf(
						"wasm32-web browser canvas host_evidence.backend is %q, want browser-canvas-rgba",
						evidence.Backend,
					),
				)
			}
			if !evidence.Framebuffer || !evidence.NativeInput {
				issues = append(
					issues,
					"wasm32-web browser canvas host_evidence requires framebuffer=true and native_input=true",
				)
			}
			if evidence.RealWindow {
				issues = append(
					issues,
					"wasm32-web browser canvas host_evidence must not claim OS real_window",
				)
			}
			if !caseNameContains(report.Cases, "wasm32-web browser canvas surface") {
				issues = append(
					issues,
					"wasm32-web browser canvas host_evidence requires browser canvas surface case evidence",
				)
			}
			if !caseNameContains(report.Cases, "wasm32-web browser canvas RGBA readback") {
				issues = append(
					issues,
					"wasm32-web browser canvas host_evidence requires canvas RGBA readback case evidence",
				)
			}
		case "wasm32-web-browser-canvas-release-v1":
			if evidence.Backend != "browser-canvas-rgba-accessible" {
				issues = append(
					issues,
					fmt.Sprintf(
						"browser release host_evidence.backend is %q, want browser-canvas-rgba-accessible",
						evidence.Backend,
					),
				)
			}
			if !evidence.Framebuffer || !evidence.NativeInput {
				issues = append(
					issues,
					"browser release host_evidence requires framebuffer=true and native_input=true",
				)
			}
			if evidence.RealWindow {
				issues = append(
					issues,
					"browser release host_evidence must not claim OS real_window",
				)
			}
			if !evidence.BrowserCanvas {
				issues = append(issues, "browser release host_evidence.browser_canvas must be true")
			}
			if !evidence.BrowserInput {
				issues = append(issues, "browser release host_evidence.browser_input must be true")
			}
			if !evidence.BrowserClipboard {
				issues = append(
					issues,
					"browser release host_evidence.browser_clipboard must be true",
				)
			}
			if evidence.BrowserClipboardHarness != "deterministic-browser-clipboard-v1" {
				issues = append(
					issues,
					fmt.Sprintf(
						("browser release host_evidence.browser_clipboard_harness is "+
							"%q, want deterministic-browser-clipboard-v1"),
						evidence.BrowserClipboardHarness,
					),
				)
			}
			if !evidence.BrowserComposition {
				issues = append(
					issues,
					"browser release host_evidence.browser_composition must be true",
				)
			}
			if !evidence.BrowserAccessibilitySnapshot {
				issues = append(
					issues,
					"browser release host_evidence.browser_accessibility_snapshot must be true",
				)
			}
			if !evidence.BrowserAccessibilityMirror {
				issues = append(
					issues,
					"browser release host_evidence.browser_accessibility_mirror must be true",
				)
			}
			if !caseNameContains(report.Cases, "wasm32-web browser canvas surface") {
				issues = append(
					issues,
					"browser release host_evidence requires browser canvas surface case evidence",
				)
			}
			if !caseNameContains(report.Cases, "wasm32-web browser canvas RGBA readback") {
				issues = append(
					issues,
					"browser release host_evidence requires canvas RGBA readback case evidence",
				)
			}
		default:
			issues = append(
				issues,
				fmt.Sprintf(
					("wasm32-web host_evidence.level is %q, want "+
						"wasm32-web-compiler-owned-loader, "+
						"wasm32-web-browser-canvas-input, or "+
						"wasm32-web-browser-canvas-release-v1"),
					evidence.Level,
				),
			)
		}
	}
	return issues
}

func validateNativeSurfaceHostEvidence(report Report) []string {
	if report.HostEvidence.Level != NativeSurfaceHostLevelLinuxX64 {
		if report.NativeSurfaceHost != nil {
			return []string{
				("native_surface_host evidence is only valid with " +
					"linux-x64-native-surface-host-v1 host_evidence.level"),
			}
		}
		return nil
	}

	var issues []string
	evidence := report.NativeSurfaceHost
	if evidence == nil {
		return []string{
			"native_surface_host evidence is required for linux-x64-native-surface-host-v1",
		}
	}
	if report.Target != "linux-x64" {
		issues = append(
			issues,
			fmt.Sprintf("native_surface_host target is %q, want linux-x64", report.Target),
		)
	}
	if report.Runtime != "surface-linux-x64" {
		issues = append(
			issues,
			fmt.Sprintf(
				"native_surface_host runtime is %q, want surface-linux-x64",
				report.Runtime,
			),
		)
	}
	if evidence.Schema != NativeSurfaceHostSchemaV1 {
		issues = append(
			issues,
			fmt.Sprintf(
				"native_surface_host.schema is %q, want %s",
				evidence.Schema,
				NativeSurfaceHostSchemaV1,
			),
		)
	}
	if evidence.Host != "wayland" {
		issues = append(
			issues,
			fmt.Sprintf("native_surface_host.host is %q, want wayland", evidence.Host),
		)
	}
	if evidence.Protocol != NativeSurfaceHostProtocolV1 {
		issues = append(
			issues,
			fmt.Sprintf(
				"native_surface_host.protocol is %q, want %s",
				evidence.Protocol,
				NativeSurfaceHostProtocolV1,
			),
		)
	}
	if evidence.AppProcessKind != "compiled-linux-x64-tetra-app" {
		issues = append(
			issues,
			fmt.Sprintf(
				"native_surface_host.app_process_kind is %q, want compiled-linux-x64-tetra-app",
				evidence.AppProcessKind,
			),
		)
	}
	if evidence.HostProcessKind != "tetra-surface-host-wayland" {
		issues = append(
			issues,
			fmt.Sprintf(
				"native_surface_host.host_process_kind is %q, want tetra-surface-host-wayland",
				evidence.HostProcessKind,
			),
		)
	}
	if evidence.AppPID <= 0 {
		issues = append(issues, "native_surface_host.app_pid must be positive")
	}
	if evidence.HostPID <= 0 {
		issues = append(issues, "native_surface_host.host_pid must be positive")
	}
	if !evidence.SurfaceOpenFromApp {
		issues = append(issues, "native_surface_host.surface_open_from_app must be true")
	}
	if !evidence.PollEventFromHost {
		issues = append(issues, "native_surface_host.poll_event_from_host must be true")
	}
	if !evidence.PresentFromAppRGBA {
		issues = append(issues, "native_surface_host.present_from_app_rgba must be true")
	}
	if !evidence.AppLoopObserved {
		issues = append(issues, "native_surface_host.app_loop_observed must be true")
	}
	if !evidence.RealWindow {
		issues = append(issues, "native_surface_host.real_window must be true")
	}
	if !evidence.RealCloseEvent {
		issues = append(issues, "native_surface_host.real_close_event must be true")
	}
	if evidence.RealPointerEventCount <= 0 {
		issues = append(issues, "native_surface_host.real_pointer_event_count must be positive")
	}
	if evidence.RealKeyEventCount <= 0 {
		issues = append(issues, "native_surface_host.real_key_event_count must be positive")
	}
	if evidence.PresentedFrameCount <= 0 {
		issues = append(issues, "native_surface_host.presented_frame_count must be positive")
	} else if len(report.Frames) > 0 && evidence.PresentedFrameCount < len(report.Frames) {
		issues = append(issues, fmt.Sprintf(("native_surface_host.presented_frame_count = %d, want at "+
			"least %d reported frames"), evidence.PresentedFrameCount, len(report.Frames)))
	}
	if evidence.PreRenderedFrameSource {
		issues = append(issues, "native_surface_host.pre_rendered_frame_source must be false")
	}
	if strings.TrimSpace(evidence.DeliveryPath) == "" {
		issues = append(issues, "native_surface_host.delivery_path is required")
	} else if evidence.DeliveryPath != "compiled-tetra-app-to-wayland-surface" {
		issues = append(issues, fmt.Sprintf(("native_surface_host.delivery_path is %q, want "+
			"compiled-tetra-app-to-wayland-surface"), evidence.DeliveryPath))
	}

	if !hasProcessNameAndPathMarkers(
		report.Processes,
		"app",
		"surface component app",
		"--surface-host",
		"wayland",
	) {
		issues = append(
			issues,
			("native_surface_host requires compiled Surface component app " +
				"process launched with --surface-host wayland"),
		)
	}
	if !hasProcessNameAndPathMarkers(
		report.Processes,
		"runtime",
		"native surface host",
		"tetra-surface-host-wayland",
	) {
		issues = append(
			issues,
			"native_surface_host requires tetra-surface-host-wayland runtime process evidence",
		)
	}
	if !eventKindContains(report.Events, "close") {
		issues = append(issues, "native_surface_host requires real close event evidence")
	}
	if !hasPointerEvent(report.Events) {
		issues = append(issues, "native_surface_host requires at least one real pointer event")
	}
	if !eventKindContains(report.Events, "key_down") {
		issues = append(issues, "native_surface_host requires at least one real key event")
	}
	if !hasRunningAppPresentedFrame(report) {
		issues = append(
			issues,
			"native_surface_host requires at least one frame produced by the running Tetra app",
		)
	}
	issues = append(issues, validateNativeSurfaceHostNoSubstitution(report)...)
	return issues
}

func hasPointerEvent(events []EventReport) bool {
	for _, event := range events {
		switch strings.ToLower(strings.TrimSpace(event.Kind)) {
		case "mouse_down", "mouse_up", "mouse_move", "pointer_down", "pointer_up", "pointer_move":
			return true
		}
	}
	return false
}

func hasRunningAppPresentedFrame(report Report) bool {
	for _, frame := range report.Frames {
		if !frame.Presented || frame.Precomputed {
			continue
		}
		if normalizeFrameEvidenceToken(frame.Producer) != "app" {
			continue
		}
		if strings.TrimSpace(frame.AppSource) != "" &&
			normalizeEvidencePath(frame.AppSource) != normalizeEvidencePath(report.Source) {
			continue
		}
		if nativeSurfaceHostImageArtifactPath(frame.ArtifactPath) {
			continue
		}
		return true
	}
	return false
}

func validateNativeSurfaceHostNoSubstitution(report Report) []string {
	var issues []string
	for _, process := range report.Processes {
		forbidden := nativeSurfaceHostForbiddenSubstitution(process.Name + " " + process.Path)
		if forbidden != "" {
			issues = append(
				issues,
				fmt.Sprintf(
					"native_surface_host process %s uses forbidden %s",
					process.Name,
					forbidden,
				),
			)
		}
	}
	for _, artifact := range report.Artifacts {
		if nativeSurfaceHostImageArtifactPath(artifact.Path) {
			issues = append(
				issues,
				fmt.Sprintf(
					"native_surface_host artifact %s must not be PNG/SVG/HTML/browser pre-rendered image evidence",
					artifact.Path,
				),
			)
		}
		forbidden := nativeSurfaceHostForbiddenSubstitution(artifact.Kind + " " + artifact.Path)
		if forbidden != "" {
			issues = append(
				issues,
				fmt.Sprintf(
					"native_surface_host artifact %s uses forbidden %s",
					artifact.Path,
					forbidden,
				),
			)
		}
	}
	for _, frame := range report.Frames {
		if frame.Precomputed {
			issues = append(
				issues,
				fmt.Sprintf(
					"native_surface_host frame %d must not be precomputed or pre-rendered",
					frame.Order,
				),
			)
		}
		if normalizeFrameEvidenceToken(frame.Producer) != "app" {
			issues = append(
				issues,
				fmt.Sprintf(
					"native_surface_host frame %d producer is %q, want app",
					frame.Order,
					frame.Producer,
				),
			)
		}
		if nativeSurfaceHostImageArtifactPath(frame.ArtifactPath) {
			issues = append(
				issues,
				fmt.Sprintf(
					("native_surface_host frame %d artifact_path %q must not be "+
						"PNG/SVG/HTML/browser pre-rendered image evidence"),
					frame.Order,
					frame.ArtifactPath,
				),
			)
		}
		forbidden := nativeSurfaceHostForbiddenSubstitution(
			frame.ArtifactPath + " " + frame.Producer + " " + frame.EvidenceRole,
		)
		if forbidden != "" {
			issues = append(
				issues,
				fmt.Sprintf(
					"native_surface_host frame %d uses forbidden %s",
					frame.Order,
					forbidden,
				),
			)
		}
	}
	if report.NativeSurfaceHost != nil {
		forbidden := nativeSurfaceHostForbiddenSubstitution(report.NativeSurfaceHost.DeliveryPath)
		if forbidden != "" {
			issues = append(
				issues,
				fmt.Sprintf("native_surface_host delivery_path uses forbidden %s", forbidden),
			)
		}
	}
	return issues
}

func nativeSurfaceHostForbiddenSubstitution(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	for _, forbidden := range []struct {
		marker     string
		diagnostic string
	}{
		{marker: "--probe-frame", diagnostic: "probe-frame substitution"},
		{marker: "real-window-probe", diagnostic: "real-window probe substitution"},
		{marker: "host_probe", diagnostic: "host probe substitution"},
		{marker: "guest_viewer", diagnostic: "viewer substitution"},
		{marker: "guest viewer", diagnostic: "viewer substitution"},
		{marker: "imagemagick", diagnostic: "ImageMagick viewer substitution"},
		{marker: "display -title", diagnostic: "ImageMagick viewer substitution"},
		{marker: "browser canvas", diagnostic: "browser/canvas substitution"},
		{marker: "browser-canvas", diagnostic: "browser/canvas substitution"},
		{marker: "wasm32-web", diagnostic: "browser/wasm substitution"},
		{marker: "screenshot runner", diagnostic: "screenshot substitution"},
		{marker: "screenshot-runner", diagnostic: "screenshot substitution"},
		{marker: "pre-rendered", diagnostic: "pre-rendered frame substitution"},
		{marker: "prerendered", diagnostic: "pre-rendered frame substitution"},
	} {
		if strings.Contains(lower, forbidden.marker) {
			return forbidden.diagnostic
		}
	}
	return ""
}

func nativeSurfaceHostImageArtifactPath(path string) bool {
	lower := strings.ToLower(strings.TrimSpace(path))
	if lower == "" {
		return false
	}
	for _, suffix := range []string{".png", ".svg", ".html", ".htm", ".mjs", ".js"} {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return strings.Contains(lower, "browser-canvas") || strings.Contains(lower, "canvas-capture")
}

func rejectNonRuntimeEvidence(raw []byte) []string {
	lower := strings.ToLower(string(raw))
	lower = strings.ReplaceAll(lower, `"stale_report_rejected"`, `"freshness_report_rejected"`)
	forbidden := []string{
		"metadata-only",
		"node-only",
		"web-only",
		"dom-only",
		"sidecar-only",
		"docs-only",
		"build-only",
		"stale",
		" fake",
		"fake/",
		"\"fake\"",
		" mock",
		"mock/",
		"\"mock\"",
		"placeholder",
		".ui.html",
		".ui.web.mjs",
		".ui.json",
		"tetra.ui.v1",
		"dom ui",
		"html ui",
		"user javascript",
		"user js",
		"react component",
		"gtk widget",
		"qt widget",
		"winui",
		"cocoa",
	}
	var issues []string
	for _, marker := range forbidden {
		if strings.Contains(lower, marker) {
			issues = append(
				issues,
				fmt.Sprintf(
					"report contains forbidden non-runtime evidence marker %q",
					strings.Trim(marker, " /\""),
				),
			)
		}
	}
	return issues
}

func validateTargetRuntimeEvidence(report Report) []string {
	var issues []string
	switch report.Target {
	case "headless":
		if report.Runtime != "surface-headless" {
			issues = append(
				issues,
				fmt.Sprintf("headless target runtime is %q, want surface-headless", report.Runtime),
			)
		}
		if !hasRuntimeProcessName(report.Processes, "headless") {
			issues = append(issues, "headless target requires a headless Surface runtime process")
		}
		if !caseNameContains(report.Cases, "headless event dispatch") {
			issues = append(issues, "headless target requires headless event dispatch evidence")
		}
		if !caseNameContains(report.Cases, "headless actual runner trace") {
			issues = append(
				issues,
				"headless target requires headless actual runner trace evidence",
			)
		}
		if isAccessibilityMetadataReport(report) {
			if !hasFrameOrderDimensions(report.Frames, 5, 480, 320, 1920) {
				issues = append(
					issues,
					("headless accessibility metadata target requires order-5 " +
						"480x320 resized headless runner trace frame evidence"),
				)
			}
		} else if isProductionToolkitReport(report) {
			if !hasFrameOrderDimensions(report.Frames, 5, 560, 420, 2240) {
				issues = append(issues, ("headless production toolkit target requires order-5 560x420 " +
					"resized headless runner trace frame evidence"))
			}
		} else if isToolkitReuseReport(report) {
			if !hasFrameOrderDimensions(report.Frames, 5, 480, 320, 1920) {
				issues = append(issues, ("headless toolkit reuse target requires order-5 480x320 " +
					"resized headless runner trace frame evidence"))
			}
		} else if isMinimalToolkitReport(report) {
			if !hasFrameOrderDimensions(report.Frames, 4, 400, 240, 1600) {
				issues = append(issues, ("headless minimal toolkit target requires order-4 400x240 " +
					"resized headless runner trace frame evidence"))
			}
		} else if isComponentTreeReport(report) {
			if !hasFrameOrderDimensions(report.Frames, 2, 400, 240, 1600) {
				issues = append(issues, ("headless component tree target requires order-2 400x240 " +
					"resized headless runner trace frame evidence"))
			}
		} else if isTextFocusInputReport(report) {
			if !hasFrameOrderDimensions(report.Frames, 2, 400, 240, 1600) {
				issues = append(issues, ("headless text focus input target requires order-2 400x240 " +
					"resized headless runner trace frame evidence"))
			}
		} else if !hasFrameOrderDimensions(report.Frames, 2, 320, 200, 1280) {
			issues = append(issues, ("headless target requires order-2 320x200 headless runner " +
				"trace frame evidence"))
		}
	case "linux-x64":
		if report.Runtime != "surface-linux-x64" {
			issues = append(
				issues,
				fmt.Sprintf(
					"linux-x64 target runtime is %q, want surface-linux-x64",
					report.Runtime,
				),
			)
		}
		if !hasRuntimeProcessName(report.Processes, "linux-x64") {
			issues = append(issues, "linux-x64 target requires a linux-x64 Surface runtime process")
		}
		if report.HostEvidence.Level == NativeSurfaceHostLevelLinuxX64 {
			if !hasProcessNameAndPathMarkers(
				report.Processes,
				"app",
				"surface component app",
				"--surface-host",
				"wayland",
			) {
				issues = append(
					issues,
					("linux-x64 native Surface host target requires compiled " +
						"component app launched with --surface-host wayland"),
				)
			}
			if !hasProcessNameAndPathMarkers(
				report.Processes,
				"runtime",
				"native surface host",
				"tetra-surface-host-wayland",
			) {
				issues = append(
					issues,
					"linux-x64 native Surface host target requires tetra-surface-host-wayland runtime process",
				)
			}
			for _, requiredCase := range []string{
				"native Surface host Wayland live window",
				"native Surface host app loop observed",
				"native Surface host close event",
				"native Surface host pointer input",
				"native Surface host keyboard input",
				"native Surface host frame presented by running app",
			} {
				if !caseNameContains(report.Cases, requiredCase) {
					issues = append(
						issues,
						fmt.Sprintf(
							"linux-x64 native Surface host target requires %s evidence",
							requiredCase,
						),
					)
				}
			}
		} else if isLinuxRealWindowHostEvidenceLevel(report.HostEvidence.Level) {
			if !hasAppProcessWithExpectedExit(report.Processes, "surface linux-x64 real-window probe", 42) {
				issues = append(issues, ("linux-x64 real-window target requires a Surface real-window " +
					"probe app exiting 42"))
			}
			if !caseNameContains(report.Cases, "linux-x64 real-window surface") {
				issues = append(issues, "linux-x64 real-window target requires real-window surface evidence")
			}
			if !caseNameContains(report.Cases, "linux-x64 native input event pump") {
				issues = append(issues, ("linux-x64 real-window target requires native input event " +
					"pump evidence"))
			}
			if !caseNameContains(report.Cases, "linux-x64 real-window resize event") {
				issues = append(issues, "linux-x64 real-window target requires resize event evidence")
			}
			if !caseNameContains(report.Cases, "linux-x64 real-window close event") {
				issues = append(issues, "linux-x64 real-window target requires close event evidence")
			}
			if isLinuxAppShellReport(report) {
				if !hasFrameOrderDimensions(report.Frames, 6, 720, 540, 2880) {
					issues = append(issues, ("linux-x64 app-shell target requires order-6 720x540 " +
						"presented app-shell frame evidence"))
				}
			} else if isLinuxReleaseWindowReport(report) {
				if !hasFrameOrderDimensions(report.Frames, 5, 560, 420, 2240) {
					issues = append(issues, ("linux-x64 release-window target requires order-5 560x420 " +
						"presented window frame evidence"))
				}
			} else if isAccessibilityMetadataReport(report) {
				if !hasFrameOrderDimensions(report.Frames, 5, 480, 320, 1920) {
					issues = append(issues, ("linux-x64 real-window accessibility metadata target " +
						"requires order-5 480x320 presented window frame evidence"))
				}
			} else if isProductionToolkitReport(report) {
				if !hasFrameOrderDimensions(report.Frames, 5, 560, 420, 2240) {
					issues = append(issues, ("linux-x64 real-window production toolkit target requires " +
						"order-5 560x420 presented window frame evidence"))
				}
			} else if isToolkitReuseReport(report) {
				if !hasFrameOrderDimensions(report.Frames, 5, 480, 320, 1920) {
					issues = append(issues, ("linux-x64 real-window toolkit reuse target requires order-5 " +
						"480x320 presented window frame evidence"))
				}
			} else if !hasFrameOrderDimensions(report.Frames, 5, 400, 240, 1600) {
				issues = append(issues, ("linux-x64 real-window target requires order-5 400x240 " +
					"presented window frame evidence"))
			}
		} else {
			if !hasAppProcessWithExpectedExit(report.Processes, "surface linux-x64 host probe", 42) {
				issues = append(issues, "linux-x64 target requires a Surface Host ABI probe app exiting 42")
			}
			if !caseNameContains(report.Cases, "linux-x64 Surface Host ABI") {
				issues = append(issues, "linux-x64 target requires linux-x64 Surface Host ABI evidence")
			}
			if !hasAppProcessWithExpectedExit(
				report.Processes,
				"surface linux-x64 event sequence probe",
				42,
			) {
				issues = append(issues, ("linux-x64 target requires a Surface event sequence probe " +
					"app exiting 42"))
			}
			if !caseNameContains(report.Cases, "linux-x64 host event sequence") {
				issues = append(issues, "linux-x64 target requires linux-x64 host event sequence evidence")
			}
			if !caseNameContains(report.Cases, "linux-x64 app-presented RGBA checksum") {
				issues = append(issues, "linux-x64 target requires app-presented RGBA checksum evidence")
			}
			if !hasFrameDimensions(report.Frames, 2, 2, 8) {
				issues = append(issues, "linux-x64 target requires a 2x2 app-presented RGBA checksum frame")
			}
			if !hasAppProcessWithExpectedExit(
				report.Processes,
				"surface linux-x64 counter app presented frame probe",
				-1,
			) {
				issues = append(issues, ("linux-x64 target requires a counter component app-presented " +
					"frame probe process"))
			}
			if !caseNameContains(report.Cases, "linux-x64 counter component app-presented frame") {
				issues = append(issues, ("linux-x64 target requires counter component app-presented " +
					"frame evidence"))
			}
			if !hasFrameOrderDimensions(report.Frames, 4, 320, 200, 1280) {
				issues = append(issues, ("linux-x64 target requires order-4 320x200 counter component " +
					"app-presented frame evidence"))
			}
		}
		if caseNameContains(report.Cases, "headless") {
			issues = append(issues, "linux-x64 target must not use headless runtime case evidence")
		}
	case "wasm32-web":
		if report.Runtime != "surface-wasm32-web" {
			issues = append(
				issues,
				fmt.Sprintf(
					"wasm32-web target runtime is %q, want surface-wasm32-web",
					report.Runtime,
				),
			)
		}
		if !hasRuntimeProcessName(report.Processes, "wasm32-web") {
			issues = append(
				issues,
				"wasm32-web target requires a wasm32-web Surface runtime process",
			)
		}
		if !hasProcessNameAndPathMarkers(
			report.Processes,
			"runtime",
			"surface wasm32-web import validator",
			"validate-wasm-imports",
			"--target wasm32-web",
		) {
			issues = append(
				issues,
				("wasm32-web target requires validate-wasm-imports runtime " +
					"process for Surface Host ABI import allowlist"),
			)
		}
		if !caseNameContains(report.Cases, "wasm32-web Surface Host ABI imports") {
			issues = append(
				issues,
				"wasm32-web target requires wasm32-web Surface Host ABI import evidence",
			)
		}
		if !caseNameContains(report.Cases, "compiler-owned wasm Surface loader") {
			issues = append(
				issues,
				"wasm32-web target requires compiler-owned wasm Surface loader evidence",
			)
		}
		if isBrowserCanvasHostEvidenceLevel(report.HostEvidence.Level) {
			requiredCases := []string{
				"wasm32-web browser canvas surface",
				"wasm32-web browser canvas RGBA readback",
				"wasm32-web browser canvas pointer input",
				"wasm32-web browser canvas keyboard input",
				"wasm32-web browser canvas text input",
				"compiler-owned browser canvas Surface host",
			}
			requiredEvents := []string{"mouse_up", "key_down", "text_input"}
			requiredFrameOrder := 5
			requiredFrameWidth := 400
			requiredFrameHeight := 240
			requiredFrameStride := 1600
			if isWASM32WebBrowserCanvasMorphRuntimeReport(report) {
				requiredCases = append(requiredCases,
					"wasm32-web browser canvas Morph rendered beauty frame readback",
					"wasm32-web browser canvas Morph rendered beauty checksum",
				)
				requiredFrameWidth = 320
				requiredFrameHeight = 200
				requiredFrameStride = 1280
				if isSurfaceMorphGuestDashboardSource(report.Source) {
					requiredFrameWidth = 1760
					requiredFrameHeight = 700
					requiredFrameStride = 7040
				}
			} else {
				requiredCases = append(requiredCases, "wasm32-web browser canvas resize input")
				requiredEvents = append(requiredEvents, "resize")
			}
			for _, required := range requiredCases {
				if !caseNameContains(report.Cases, required) {
					issues = append(
						issues,
						fmt.Sprintf(
							"wasm32-web browser canvas target requires %s evidence",
							required,
						),
					)
				}
			}
			if !hasProcessNameAndPathMarkers(
				report.Processes,
				"app",
				"surface wasm32-web browser canvas component app",
				"chromium",
			) &&
				!hasProcessNameAndPathMarkers(
					report.Processes,
					"app",
					"surface wasm32-web browser canvas component app",
					"chrome",
				) {
				issues = append(
					issues,
					"wasm32-web browser canvas target requires Chromium-compatible browser app process evidence",
				)
			}
			for _, kind := range requiredEvents {
				if !eventKindContains(report.Events, kind) {
					issues = append(
						issues,
						fmt.Sprintf(
							"wasm32-web browser canvas target requires %s event evidence",
							kind,
						),
					)
				}
			}
			if isAccessibilityMetadataReport(report) {
				if !hasFrameOrderDimensions(report.Frames, 5, 480, 320, 1920) {
					issues = append(
						issues,
						("wasm32-web browser canvas accessibility metadata target " +
							"requires order-5 480x320 canvas readback frame evidence"),
					)
				}
			} else if isProductionToolkitReport(report) {
				if !hasFrameOrderDimensions(report.Frames, 5, 560, 420, 2240) {
					issues = append(issues, ("wasm32-web browser canvas production toolkit target " +
						"requires order-5 560x420 canvas readback frame evidence"))
				}
			} else if isToolkitReuseReport(report) {
				if !hasFrameOrderDimensions(report.Frames, 5, 480, 320, 1920) {
					issues = append(issues, ("wasm32-web browser canvas toolkit reuse target requires " +
						"order-5 480x320 canvas readback frame evidence"))
				}
			} else if !hasFrameOrderDimensions(
				report.Frames,
				requiredFrameOrder,
				requiredFrameWidth,
				requiredFrameHeight,
				requiredFrameStride,
			) {
				issues = append(issues, fmt.Sprintf((("wasm32-web browser canvas target requires "+
					"order-%d %dx%d ")+
					"canvas readback frame evidence"), requiredFrameOrder, requiredFrameWidth, requiredFrameHeight))
			}
		} else {
			if !caseNameContains(report.Cases, "wasm32-web actual presented frame trace") {
				issues = append(issues, "wasm32-web target requires actual presented frame trace evidence")
			}
			if !hasFrameOrderDimensions(report.Frames, 4, 320, 200, 1280) {
				issues = append(issues, ("wasm32-web target requires order-4 320x200 actual presented " +
					"frame trace evidence"))
			}
		}
		if caseNameContains(report.Cases, "headless") {
			issues = append(issues, "wasm32-web target must not use headless runtime case evidence")
		}
	}
	return issues
}

func isWASM32WebBrowserCanvasMorphRuntimeReport(report Report) bool {
	return report.Morph != nil &&
		report.RenderCommandStream != nil &&
		report.RenderCommandStream.Renderer == "browser-canvas-rgba" &&
		report.Target == "wasm32-web" &&
		report.Runtime == "surface-wasm32-web" &&
		report.HostEvidence.Level == "wasm32-web-browser-canvas-input"
}

func IsWASM32WebBrowserCanvasMorphRuntimeReport(report Report) bool {
	return isWASM32WebBrowserCanvasMorphRuntimeReport(report)
}

func isLinuxX64RealWindowMorphRuntimeReport(report Report) bool {
	return report.Morph != nil &&
		report.RenderCommandStream != nil &&
		report.RenderCommandStream.Renderer == "wayland-shm-rgba" &&
		report.Target == "linux-x64" &&
		report.Runtime == "surface-linux-x64" &&
		report.HostEvidence.Level == "linux-x64-real-window" &&
		report.HostEvidence.Backend == "wayland-shm-rgba" &&
		report.HostEvidence.RealWindow &&
		report.HostEvidence.NativeInput
}

func validateTextFocusInputEvidence(report Report, components map[string]ComponentReport) []string {
	if !isTextFocusInputReport(report) {
		return nil
	}
	var issues []string
	for _, required := range []string{
		"text focus input click focuses TextBox",
		"text focus input Tab changes focus",
		"text focus input keyboard routes only focused component",
		"text focus input text insertion",
		"text focus input caret movement",
		"text focus input backspace delete",
		"text focus input resize preserves focus",
		"text focus input rendered frame update",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(
				issues,
				fmt.Sprintf("text focus input report requires %s evidence", required),
			)
		}
	}
	textBox, ok := components["TextBox"]
	if !ok {
		issues = append(issues, "text focus input report requires TextBox component evidence")
	} else {
		if textBox.State["buffer"] == "" {
			issues = append(issues, "TextBox state requires edited component-owned buffer")
		}
		if textBox.State["caret"] == "" {
			issues = append(issues, "TextBox state requires caret evidence")
		}
		if textBox.State["backspace_count"] == "" || textBox.State["delete_count"] == "" {
			issues = append(issues, "TextBox state requires backspace/delete evidence")
		}
	}
	button, ok := components["SubmitButton"]
	if !ok {
		issues = append(issues, "text focus input report requires SubmitButton component evidence")
	} else if button.State["press_count"] == "" {
		issues = append(issues, "SubmitButton state requires focused keyboard press evidence")
	}
	if !hasEventTargetKind(report.Events, "TextBox", "mouse_up") {
		issues = append(issues, "text focus input report requires mouse_up targeted to TextBox")
	}
	if !hasEventTargetKind(report.Events, "TextBox", "text_input") {
		issues = append(issues, "text focus input report requires text_input targeted to TextBox")
	}
	if !hasKeyEvent(report.Events, 9) {
		issues = append(issues, "text focus input report requires Tab key focus routing evidence")
	}
	if !hasKeyEvent(report.Events, 37) && !hasKeyEvent(report.Events, 39) {
		issues = append(issues, "text focus input report requires caret movement key evidence")
	}
	if !hasKeyEvent(report.Events, 8) || !hasKeyEvent(report.Events, 46) {
		issues = append(
			issues,
			"text focus input report requires backspace and delete key evidence",
		)
	}
	if !hasEventTargetKind(report.Events, "SubmitButton", "key_down") {
		issues = append(
			issues,
			"text focus input report requires keyboard event routed to focused SubmitButton",
		)
	}
	if !hasResizePreservingFocus(report.Events) {
		issues = append(
			issues,
			"text focus input report requires resize preserving focused component state",
		)
	}
	if !hasTransition(report.StateTransitions, "TextBox", "buffer") ||
		!hasTransition(report.StateTransitions, "TextBox", "caret") {
		issues = append(
			issues,
			"text focus input report requires TextBox buffer and caret state transitions",
		)
	}
	if !hasTransition(report.StateTransitions, "TextInputApp", "focused_component") {
		issues = append(issues, "text focus input report requires focus manager state transition")
	}
	return issues
}

func isTextFocusInputReport(report Report) bool {
	if strings.Contains(strings.ToLower(report.Source), "surface_textbox_app") {
		return true
	}
	return caseNameContains(report.Cases, "text focus input")
}

func isSupportedRuntimeTarget(target string) bool {
	switch target {
	case "headless", "linux-x64", "wasm32-web":
		return true
	default:
		return false
	}
}

func isSupportedRuntimeName(runtime string) bool {
	switch runtime {
	case "surface-headless", "surface-linux-x64", "surface-wasm32-web":
		return true
	default:
		return false
	}
}

// ---- security_validation.go ----

type SecurityPermissionReport struct {
	Schema                     string                            `json:"schema"`
	Model                      string                            `json:"model"`
	ReleaseScope               string                            `json:"release_scope"`
	Source                     string                            `json:"source"`
	AppShellFeatures           string                            `json:"app_shell_features"`
	ProductionClaim            bool                              `json:"production_claim"`
	Experimental               bool                              `json:"experimental"`
	DefaultDeny                bool                              `json:"default_deny"`
	ShellFeaturePolicyEnforced bool                              `json:"shell_feature_policy_enforced"`
	Capabilities               []SurfaceSecurityCapabilityReport `json:"capabilities"`
	Permissions                []SurfacePermissionReport         `json:"permissions"`
	ProcessBoundaries          []SurfaceProcessBoundaryReport    `json:"process_boundaries"`
	AssetSafety                []SurfaceAssetSafetyReport        `json:"asset_safety"`
	UnsupportedClaims          []string                          `json:"unsupported_claims"`
	NegativeGuards             SurfaceSecurityNegativeGuards     `json:"negative_guards"`
}

type SurfaceSecurityCapabilityReport struct {
	Name              string `json:"name"`
	SourceFeature     string `json:"source_feature"`
	Status            string `json:"status"`
	Allowed           bool   `json:"allowed"`
	CapabilityChecked bool   `json:"capability_checked"`
	HostTrace         bool   `json:"host_trace"`
	Policy            string `json:"policy"`
	Evidence          string `json:"evidence"`
	BlockedReason     string `json:"blocked_reason"`
	Pass              bool   `json:"pass"`
}

type SurfacePermissionReport struct {
	Name              string `json:"name"`
	Status            string `json:"status"`
	Allowed           bool   `json:"allowed"`
	CapabilityChecked bool   `json:"capability_checked"`
	BlockedReason     string `json:"blocked_reason"`
	Evidence          string `json:"evidence"`
	Pass              bool   `json:"pass"`
}

type SurfaceProcessBoundaryReport struct {
	Name              string `json:"name"`
	SchemaChecked     bool   `json:"schema_checked"`
	CapabilityChecked bool   `json:"capability_checked"`
	UserJS            bool   `json:"user_js"`
	NodeIntegration   bool   `json:"node_integration"`
	ElectronRuntime   bool   `json:"electron_runtime"`
	Pass              bool   `json:"pass"`
}

type SurfaceAssetSafetyReport struct {
	Kind                string `json:"kind"`
	LocalOnly           bool   `json:"local_only"`
	SHA256Required      bool   `json:"sha256_required"`
	SizeLimitBytes      int    `json:"size_limit_bytes"`
	NetworkFetchAllowed bool   `json:"network_fetch_allowed"`
	Parser              string `json:"parser"`
	BoundsChecked       bool   `json:"bounds_checked"`
	Pass                bool   `json:"pass"`
}

type SurfaceSecurityNegativeGuards struct {
	NoAmbientFilesystem                       bool `json:"no_ambient_filesystem"`
	NoAmbientNetwork                          bool `json:"no_ambient_network"`
	NoShellFeatureBypass                      bool `json:"no_shell_feature_bypass"`
	NoPermissionlessClipboard                 bool `json:"no_permissionless_clipboard"`
	NoNotificationDialogWithoutTargetEvidence bool `json:"no_notification_dialog_without_target_evidence"`
	NoNetworkAssetFetch                       bool `json:"no_network_asset_fetch"`
	NoUntrustedFontImageDecode                bool `json:"no_untrusted_font_image_decode"`
	NoElectronNodeIntegration                 bool `json:"no_electron_node_integration"`
	NoUserJSAppLogic                          bool `json:"no_user_js_app_logic"`
	NoDOMAppUITree                            bool `json:"no_dom_app_ui_tree"`
}

func ValidateSecurityPermissionReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	switch schema {
	case SecurityPermissionSchemaV1:
		var report SecurityPermissionReport
		if err := decodeStrict(raw, &report); err != nil {
			return err
		}
		issues := validateSecurityPermissionReport(report, nil, "")
		if len(issues) > 0 {
			return errors.New(strings.Join(issues, "; "))
		}
		return nil
	case SchemaV1:
		var report Report
		if err := decodeStrict(raw, &report); err != nil {
			return err
		}
		issues := validateSecurityPermissionEvidence(report)
		if len(issues) > 0 {
			return errors.New(strings.Join(issues, "; "))
		}
		return nil
	default:
		return fmt.Errorf(
			"schema is %q, want %q or %q",
			schema,
			SecurityPermissionSchemaV1,
			SchemaV1,
		)
	}
}

func validateSecurityPermissionEvidence(report Report) []string {
	if report.SecurityPermissions == nil {
		if isLinuxAppShellReport(report) {
			return []string{"security_permissions evidence is required for linux app-shell reports"}
		}
		return nil
	}
	var features []LinuxAppShellFeatureReport
	if report.LinuxAppShell != nil {
		features = report.LinuxAppShell.ShellFeatures
	}
	return validateSecurityPermissionReport(*report.SecurityPermissions, features, report.Source)
}

func validateSecurityPermissionReport(
	report SecurityPermissionReport,
	features []LinuxAppShellFeatureReport,
	source string,
) []string {
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: report.Schema, want: SecurityPermissionSchemaV1},
		{field: "model", got: report.Model, want: "surface-security-permission-v1"},
		{field: "release_scope", got: report.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
		{field: "app_shell_features", got: report.AppShellFeatures, want: "electron-feature-ledger-v1"},
	} {
		if check.got != check.want {
			issues = append(
				issues,
				fmt.Sprintf(
					"security_permissions %s is %q, want %q",
					check.field,
					check.got,
					check.want,
				),
			)
		}
	}
	if strings.TrimSpace(source) != "" &&
		normalizeEvidencePath(report.Source) != normalizeEvidencePath(source) {
		issues = append(
			issues,
			fmt.Sprintf(
				"security_permissions source %q must match report source %q",
				report.Source,
				source,
			),
		)
	}
	if strings.TrimSpace(report.Source) == "" {
		issues = append(issues, "security_permissions source is required")
	}
	if !report.ProductionClaim {
		issues = append(issues, "security_permissions production_claim must be true")
	}
	if report.Experimental {
		issues = append(issues, "security_permissions experimental must be false")
	}
	if !report.DefaultDeny {
		issues = append(issues, "security_permissions default_deny must be true")
	}
	if !report.ShellFeaturePolicyEnforced {
		issues = append(issues, "security_permissions shell_feature_policy_enforced must be true")
	}
	issues = append(issues, validateSecurityCapabilityRows(report.Capabilities, features)...)
	issues = append(issues, validateSecurityPermissionRows(report.Permissions)...)
	issues = append(issues, validateSurfaceSecurityProcessBoundaries(report.ProcessBoundaries)...)
	issues = append(issues, validateSurfaceSecurityAssetSafety(report.AssetSafety)...)
	issues = append(issues, validateSurfaceSecurityUnsupportedClaims(report.UnsupportedClaims)...)
	issues = append(issues, validateSurfaceSecurityNegativeGuards(report.NegativeGuards)...)
	return issues
}

func validateSecurityCapabilityRows(
	rows []SurfaceSecurityCapabilityReport,
	features []LinuxAppShellFeatureReport,
) []string {
	var issues []string
	if len(rows) == 0 {
		return []string{"security_permissions capabilities evidence is required"}
	}
	capabilities := map[string]SurfaceSecurityCapabilityReport{}
	for _, row := range rows {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			issues = append(issues, "security_permissions capability name is required")
			continue
		}
		capabilities[name] = row
		if !linuxAppShellKnownFeature(name) {
			issues = append(
				issues,
				fmt.Sprintf(
					"security_permissions capability %s is not a known app-shell feature",
					name,
				),
			)
		}
		if row.SourceFeature != name {
			issues = append(
				issues,
				fmt.Sprintf(
					"security_permissions capability %s source_feature is %q, want %q",
					name,
					row.SourceFeature,
					name,
				),
			)
		}
		if !row.CapabilityChecked || !row.HostTrace || !row.Pass {
			issues = append(
				issues,
				fmt.Sprintf(
					("security_permissions capability %s requires "+
						"capability_checked=true, host_trace=true, and pass=true"),
					name,
				),
			)
		}
		if strings.TrimSpace(row.Policy) == "" || strings.TrimSpace(row.Evidence) == "" {
			issues = append(
				issues,
				fmt.Sprintf(
					"security_permissions capability %s requires policy and evidence",
					name,
				),
			)
		}
	}
	for _, feature := range features {
		name := strings.TrimSpace(feature.Name)
		if name == "" {
			continue
		}
		capability, ok := capabilities[name]
		if !ok {
			issues = append(
				issues,
				fmt.Sprintf("security_permissions capabilities missing %s", name),
			)
			continue
		}
		issues = append(issues, validateSecurityCapabilityAgainstFeature(capability, feature)...)
	}
	return issues
}

func validateSecurityCapabilityAgainstFeature(
	capability SurfaceSecurityCapabilityReport,
	feature LinuxAppShellFeatureReport,
) []string {
	name := feature.Name
	switch feature.Status {
	case "target_evidenced", "scoped_adapter":
		var issues []string
		if capability.Status != "allowed_with_policy" || !capability.Allowed {
			issues = append(
				issues,
				fmt.Sprintf(
					"security_permissions capability %s must be allowed_with_policy for claimed app-shell feature",
					name,
				),
			)
		}
		if !capability.CapabilityChecked || !capability.HostTrace || !capability.Pass {
			issues = append(
				issues,
				fmt.Sprintf(
					"security_permissions capability %s requires checked target-host evidence",
					name,
				),
			)
		}
		if strings.TrimSpace(capability.BlockedReason) != "" {
			issues = append(
				issues,
				fmt.Sprintf(
					"security_permissions capability %s must not carry blocked_reason when allowed",
					name,
				),
			)
		}
		return issues
	case "blocked_pass":
		if capability.Status != "blocked_nonclaim" || capability.Allowed ||
			strings.TrimSpace(capability.BlockedReason) == "" {
			return []string{
				fmt.Sprintf(
					("security_permissions capability %s must remain " +
						"blocked_nonclaim and cannot bypass the P16 blocked feature " +
						"ledger"),
					name,
				),
			}
		}
		return nil
	default:
		return []string{
			fmt.Sprintf(
				"security_permissions capability %s references unsupported feature status %q",
				name,
				feature.Status,
			),
		}
	}
}

func validateSecurityPermissionRows(rows []SurfacePermissionReport) []string {
	var issues []string
	if len(rows) == 0 {
		return []string{"security_permissions permissions evidence is required"}
	}
	permissions := map[string]SurfacePermissionReport{}
	for _, row := range rows {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			issues = append(issues, "security_permissions permission name is required")
			continue
		}
		permissions[name] = row
		if !row.CapabilityChecked || !row.Pass {
			issues = append(
				issues,
				fmt.Sprintf(
					"security_permissions permission %s requires capability_checked=true and pass=true",
					name,
				),
			)
		}
		if strings.TrimSpace(row.Evidence) == "" {
			issues = append(
				issues,
				fmt.Sprintf("security_permissions permission %s evidence is required", name),
			)
		}
	}
	for _, name := range []string{"filesystem", "network"} {
		row, ok := permissions[name]
		if !ok {
			issues = append(
				issues,
				fmt.Sprintf("security_permissions permissions missing %s", name),
			)
			continue
		}
		if row.Status != "denied" || row.Allowed || strings.TrimSpace(row.BlockedReason) == "" {
			issues = append(
				issues,
				fmt.Sprintf(
					"security_permissions permission %s must be denied by default with blocked_reason",
					name,
				),
			)
		}
	}
	if row, ok := permissions["clipboard"]; !ok {
		issues = append(issues, "security_permissions permissions missing clipboard")
	} else if row.Status != "allowed_with_policy" || !row.Allowed || !row.CapabilityChecked || strings.TrimSpace(
		row.Evidence,
	) == "" {
		issues = append(issues, ("security_permissions permission clipboard must be " +
			"allowed_with_policy with host evidence"))
	}
	for _, name := range []string{"notifications", "dialogs", "shell_open_url"} {
		row, ok := permissions[name]
		if !ok {
			issues = append(
				issues,
				fmt.Sprintf("security_permissions permissions missing %s", name),
			)
			continue
		}
		if row.Status != "denied" || row.Allowed || strings.TrimSpace(row.BlockedReason) == "" {
			issues = append(
				issues,
				fmt.Sprintf(
					"security_permissions permission %s must be denied until target evidence exists",
					name,
				),
			)
		}
	}
	return issues
}

func validateSurfaceSecurityProcessBoundaries(rows []SurfaceProcessBoundaryReport) []string {
	var issues []string
	boundaries := map[string]SurfaceProcessBoundaryReport{}
	for _, row := range rows {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			issues = append(issues, "security_permissions process_boundary name is required")
			continue
		}
		boundaries[name] = row
		if !row.SchemaChecked || !row.CapabilityChecked || !row.Pass {
			issues = append(
				issues,
				fmt.Sprintf(
					("security_permissions process_boundary %s requires "+
						"schema_checked, capability_checked, and pass"),
					name,
				),
			)
		}
		if row.UserJS || row.NodeIntegration || row.ElectronRuntime {
			issues = append(
				issues,
				fmt.Sprintf(
					("security_permissions process_boundary %s must reject user "+
						"JS app logic, Node integration, and Electron runtime"),
					name,
				),
			)
		}
	}
	for _, name := range []string{
		"surface_app_to_host_abi",
		"linux_app_shell_host_adapter",
		"browser_canvas_host",
	} {
		if _, ok := boundaries[name]; !ok {
			issues = append(
				issues,
				fmt.Sprintf("security_permissions process_boundaries missing %s", name),
			)
		}
	}
	return issues
}

func validateSurfaceSecurityAssetSafety(rows []SurfaceAssetSafetyReport) []string {
	var issues []string
	assets := map[string]SurfaceAssetSafetyReport{}
	for _, row := range rows {
		kind := strings.TrimSpace(row.Kind)
		if kind == "" {
			issues = append(issues, "security_permissions asset_safety kind is required")
			continue
		}
		assets[kind] = row
		if !validBlockAssetKind(kind) {
			issues = append(
				issues,
				fmt.Sprintf("security_permissions asset_safety kind %s is unsupported", kind),
			)
		}
		if !row.LocalOnly || !row.SHA256Required || row.SizeLimitBytes <= 0 ||
			row.NetworkFetchAllowed ||
			strings.TrimSpace(row.Parser) == "" ||
			!row.BoundsChecked ||
			!row.Pass {
			issues = append(
				issues,
				fmt.Sprintf(
					("security_permissions asset_safety %s requires local_only, "+
						"sha256, positive size limit, no network fetch, parser, "+
						"bounds check, and pass"),
					kind,
				),
			)
		}
	}
	for _, kind := range []string{"font", "image", "icon"} {
		if _, ok := assets[kind]; !ok {
			issues = append(
				issues,
				fmt.Sprintf("security_permissions asset_safety missing %s", kind),
			)
		}
	}
	return issues
}

func validateSurfaceSecurityUnsupportedClaims(claims []string) []string {
	var issues []string
	for _, claim := range []string{
		"unrestricted-filesystem",
		"unrestricted-network",
		"native-permission-prompts",
		"production-notifications",
		"production-dialogs",
		"remote-asset-fetch",
		"electron-node-integration",
	} {
		if !stringSliceContainsFold(claims, claim) {
			issues = append(
				issues,
				fmt.Sprintf("security_permissions unsupported_claims requires %s", claim),
			)
		}
	}
	return issues
}

func validateSurfaceSecurityNegativeGuards(guards SurfaceSecurityNegativeGuards) []string {
	var issues []string
	for _, check := range []struct {
		field string
		ok    bool
	}{
		{field: "no_ambient_filesystem", ok: guards.NoAmbientFilesystem},
		{field: "no_ambient_network", ok: guards.NoAmbientNetwork},
		{field: "no_shell_feature_bypass", ok: guards.NoShellFeatureBypass},
		{field: "no_permissionless_clipboard", ok: guards.NoPermissionlessClipboard},
		{
			field: "no_notification_dialog_without_target_evidence",
			ok:    guards.NoNotificationDialogWithoutTargetEvidence,
		},
		{field: "no_network_asset_fetch", ok: guards.NoNetworkAssetFetch},
		{field: "no_untrusted_font_image_decode", ok: guards.NoUntrustedFontImageDecode},
		{field: "no_electron_node_integration", ok: guards.NoElectronNodeIntegration},
		{field: "no_user_js_app_logic", ok: guards.NoUserJSAppLogic},
		{field: "no_dom_app_ui_tree", ok: guards.NoDOMAppUITree},
	} {
		if !check.ok {
			issues = append(
				issues,
				fmt.Sprintf("security_permissions negative_guards.%s must be true", check.field),
			)
		}
	}
	return issues
}

// ---- target_host_validation.go ----

type TargetHostStatusReport struct {
	Schema             string                           `json:"schema"`
	Target             string                           `json:"target"`
	Status             string                           `json:"status"`
	Tier               string                           `json:"tier"`
	ReleaseScope       string                           `json:"release_scope"`
	Source             string                           `json:"source"`
	HostOS             string                           `json:"host_os"`
	HostArch           string                           `json:"host_arch"`
	Reason             string                           `json:"reason"`
	ProductionClaim    bool                             `json:"production_claim"`
	Experimental       bool                             `json:"experimental"`
	TargetHostEvidence bool                             `json:"target_host_evidence"`
	BuildOnlyEvidence  bool                             `json:"build_only_evidence"`
	BuildOnlyPromotion bool                             `json:"build_only_promotion"`
	LinuxSubstitute    bool                             `json:"linux_substitute"`
	CIArtifactRequired bool                             `json:"ci_artifact_required"`
	RequiredEvidence   TargetHostRequiredEvidenceReport `json:"required_evidence"`
	UnsupportedClaims  []string                         `json:"unsupported_claims"`
	NegativeGuards     TargetHostNegativeGuardsReport   `json:"negative_guards"`
}

type TargetHostRequiredEvidenceReport struct {
	RealWindow            bool `json:"real_window"`
	NativeInput           bool `json:"native_input"`
	Clipboard             bool `json:"clipboard"`
	DPIScaling            bool `json:"dpi_scaling"`
	AccessibilitySnapshot bool `json:"accessibility_snapshot"`
	AppShell              bool `json:"app_shell"`
}

type TargetHostNegativeGuardsReport struct {
	NoLinuxSubstitute    bool `json:"no_linux_substitute"`
	NoBuildOnlyPromotion bool `json:"no_build_only_promotion"`
	NoProductionClaim    bool `json:"no_production_claim"`
	NoDocsOnlyEvidence   bool `json:"no_docs_only_evidence"`
	NoCopiedReport       bool `json:"no_copied_report"`
	CIArtifactRequired   bool `json:"ci_artifact_required"`
}

func ValidateTargetHostStatus(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != TargetHostStatusSchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, TargetHostStatusSchemaV1)
	}

	var report TargetHostStatusReport
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}

	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: report.Schema, want: TargetHostStatusSchemaV1},
		{field: "release_scope", got: report.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
	} {
		if check.got != check.want {
			issues = append(
				issues,
				fmt.Sprintf(
					"target_host_status %s is %q, want %q",
					check.field,
					check.got,
					check.want,
				),
			)
		}
	}
	if !isTargetHostStatusTarget(report.Target) {
		issues = append(
			issues,
			fmt.Sprintf(
				"target_host_status target is %q, want windows-x64 or macos-x64",
				report.Target,
			),
		)
	}
	if strings.TrimSpace(report.Source) == "" {
		issues = append(issues, "target_host_status source is required")
	}
	if strings.TrimSpace(report.HostOS) == "" {
		issues = append(issues, "target_host_status host_os is required")
	}
	if strings.TrimSpace(report.HostArch) == "" {
		issues = append(issues, "target_host_status host_arch is required")
	}
	if strings.TrimSpace(report.Reason) == "" {
		issues = append(issues, "target_host_status reason is required")
	}
	if report.ProductionClaim {
		issues = append(
			issues,
			("target_host_status production_claim must be false without " +
				"full production target-host gate evidence"),
		)
	}
	if report.BuildOnlyEvidence {
		issues = append(
			issues,
			("target_host_status build-only evidence must be false; " +
				"build-only reports are not Surface runtime evidence"),
		)
	}
	if report.BuildOnlyPromotion {
		issues = append(issues, "target_host_status build-only promotion must be false")
	}
	if report.LinuxSubstitute {
		issues = append(
			issues,
			"target_host_status linux substitute must be false for non-Linux target-host evidence",
		)
	}
	if !report.CIArtifactRequired {
		issues = append(issues, "target_host_status ci_artifact_required must be true")
	}
	issues = append(issues, validateTargetHostNegativeGuards(report.NegativeGuards)...)

	switch report.Status {
	case "unsupported":
		issues = append(issues, validateUnsupportedTargetHostStatus(report)...)
	case "beta_target_host":
		issues = append(issues, validateBetaTargetHostStatus(report)...)
	default:
		issues = append(
			issues,
			fmt.Sprintf(
				"target_host_status status is %q, want unsupported or beta_target_host",
				report.Status,
			),
		)
	}

	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func isTargetHostStatusTarget(target string) bool {
	switch target {
	case "windows-x64", "macos-x64":
		return true
	default:
		return false
	}
}

func validateUnsupportedTargetHostStatus(report TargetHostStatusReport) []string {
	var issues []string
	if report.Tier != "UNSUPPORTED" {
		issues = append(
			issues,
			fmt.Sprintf("target_host_status tier is %q, want UNSUPPORTED", report.Tier),
		)
	}
	if report.Experimental {
		issues = append(
			issues,
			"target_host_status experimental must be false for unsupported nonclaim status",
		)
	}
	if report.TargetHostEvidence {
		issues = append(
			issues,
			"target_host_status target-host evidence must be false for unsupported nonclaim status",
		)
	}
	if targetHostRequiredEvidenceAny(report.RequiredEvidence) {
		issues = append(
			issues,
			("target_host_status unsupported required_evidence entries " +
				"must be false until real target-host evidence exists"),
		)
	}
	issues = append(
		issues,
		validateUnsupportedTargetHostClaims(report.Target, report.UnsupportedClaims)...)
	return issues
}

func validateBetaTargetHostStatus(report TargetHostStatusReport) []string {
	var issues []string
	if report.Tier != "BETA_TARGET_HOST" {
		issues = append(
			issues,
			fmt.Sprintf("target_host_status tier is %q, want BETA_TARGET_HOST", report.Tier),
		)
	}
	if !report.Experimental {
		issues = append(
			issues,
			"target_host_status experimental must be true for beta target-host status",
		)
	}
	if !report.TargetHostEvidence {
		issues = append(
			issues,
			"target_host_status target-host evidence must be true for beta target-host status",
		)
	}
	required := report.RequiredEvidence
	for _, check := range []struct {
		field string
		ok    bool
	}{
		{field: "real_window", ok: required.RealWindow},
		{field: "native_input", ok: required.NativeInput},
		{field: "clipboard", ok: required.Clipboard},
		{field: "dpi_scaling", ok: required.DPIScaling},
		{field: "accessibility_snapshot", ok: required.AccessibilitySnapshot},
	} {
		if !check.ok {
			issues = append(
				issues,
				fmt.Sprintf(
					"target_host_status beta target-host evidence requires required_evidence.%s",
					check.field,
				),
			)
		}
	}
	return issues
}

func validateTargetHostNegativeGuards(guards TargetHostNegativeGuardsReport) []string {
	var issues []string
	for _, check := range []struct {
		field string
		ok    bool
	}{
		{field: "no_linux_substitute", ok: guards.NoLinuxSubstitute},
		{field: "no_build_only_promotion", ok: guards.NoBuildOnlyPromotion},
		{field: "no_production_claim", ok: guards.NoProductionClaim},
		{field: "no_docs_only_evidence", ok: guards.NoDocsOnlyEvidence},
		{field: "no_copied_report", ok: guards.NoCopiedReport},
		{field: "ci_artifact_required", ok: guards.CIArtifactRequired},
	} {
		if !check.ok {
			issues = append(
				issues,
				fmt.Sprintf("target_host_status negative_guards.%s must be true", check.field),
			)
		}
	}
	return issues
}

func targetHostRequiredEvidenceAny(evidence TargetHostRequiredEvidenceReport) bool {
	return evidence.RealWindow ||
		evidence.NativeInput ||
		evidence.Clipboard ||
		evidence.DPIScaling ||
		evidence.AccessibilitySnapshot ||
		evidence.AppShell
}

func validateUnsupportedTargetHostClaims(target string, claims []string) []string {
	var issues []string
	required := unsupportedTargetHostClaimSet(target)
	for _, want := range required {
		if !stringSliceContainsFold(claims, want) {
			issues = append(
				issues,
				fmt.Sprintf("target_host_status unsupported_claims requires %s", want),
			)
		}
	}
	return issues
}

func unsupportedTargetHostClaimSet(target string) []string {
	switch target {
	case "windows-x64":
		return []string{
			"windows-real-window-surface",
			"windows-production-surface-nonclaim",
			"windows-target-host-runtime",
			"build-only-windows-surface-runtime",
			"linux-substitute-windows-surface-runtime",
		}
	case "macos-x64":
		return []string{
			"macos-real-window-surface",
			"macos-production-surface-nonclaim",
			"macos-target-host-runtime",
			"build-only-macos-surface-runtime",
			"linux-substitute-macos-surface-runtime",
		}
	default:
		return nil
	}
}

func isGitHead(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, ch := range value {
		if (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F') {
			continue
		}
		return false
	}
	return true
}
