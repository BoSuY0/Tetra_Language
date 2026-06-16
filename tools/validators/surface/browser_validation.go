package surface

import (
	"fmt"
	"strings"
)

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
		issues = append(issues, fmt.Sprintf("browser release target is %q, want wasm32-web", report.Target))
	}
	if !isSurfaceReleaseFormSource(report.Source) {
		issues = append(issues, fmt.Sprintf("browser release source path must match examples/surface_release_form.tetra, got %q", report.Source))
	}
	if report.HostEvidence.Level != "wasm32-web-browser-canvas-release-v1" {
		issues = append(issues, fmt.Sprintf("browser release host_evidence.level is %q, want wasm32-web-browser-canvas-release-v1", report.HostEvidence.Level))
	}
	if report.HostEvidence.Backend != "browser-canvas-rgba-accessible" {
		issues = append(issues, fmt.Sprintf("browser release host_evidence.backend is %q, want browser-canvas-rgba-accessible", report.HostEvidence.Backend))
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
		issues = append(issues, fmt.Sprintf("browser release host_evidence.browser_clipboard_harness is %q, want deterministic-browser-clipboard-v1", report.HostEvidence.BrowserClipboardHarness))
	}
	if !report.HostEvidence.BrowserComposition {
		issues = append(issues, "browser release host_evidence.browser_composition must be true")
	}
	if !report.HostEvidence.BrowserAccessibilitySnapshot {
		issues = append(issues, "browser release host_evidence.browser_accessibility_snapshot must be true")
	}
	if !report.HostEvidence.BrowserAccessibilityMirror {
		issues = append(issues, "browser release host_evidence.browser_accessibility_mirror must be true")
	}
	if !hasFrameOrderDimensions(report.Frames, 5, 560, 420, 2240) {
		issues = append(issues, "browser release requires order-5 560x420 canvas readback frame evidence")
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
			issues = append(issues, fmt.Sprintf("browser release report requires %s evidence", required))
		}
	}
	if report.Toolkit == nil || report.Toolkit.ToolkitLevel != "production-widgets-v1" {
		issues = append(issues, "browser release requires production-widgets-v1 toolkit evidence")
	}
	if report.ComponentTree == nil || report.ComponentTree.DynamicLevel != "production-widgets-v1" {
		issues = append(issues, "browser release requires production-widgets-v1 component tree evidence")
	}
	return issues
}

func validateBrowserSurfaceEvidence(report Report) []string {
	if !isBrowserReleaseReport(report) {
		return nil
	}
	browser := report.BrowserSurface
	if browser == nil {
		return []string{"browser_surface evidence is required for wasm32-web browser-canvas release reports"}
	}
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: browser.Schema, want: BrowserSurfaceSchemaV1},
		{field: "browser_surface_level", got: browser.BrowserSurfaceLevel, want: "browser-canvas-release-v1"},
		{field: "release_scope", got: browser.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
		{field: "host_adapter", got: browser.HostAdapter, want: "compiler-owned-browser-canvas-host"},
	} {
		if check.got != check.want {
			issues = append(issues, fmt.Sprintf("browser_surface %s is %q, want %q", check.field, check.got, check.want))
		}
	}
	if !isSurfaceReleaseFormSource(browser.Source) {
		issues = append(issues, fmt.Sprintf("browser_surface source path must match examples/surface_release_form.tetra, got %q", browser.Source))
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
		issues = append(issues, "browser_surface canvas requires opened, readback, and pass evidence")
	}
	if browser.Canvas.Width != 560 || browser.Canvas.Height != 420 || browser.Canvas.FrameOrder != 5 {
		issues = append(issues, fmt.Sprintf("browser_surface canvas frame is order-%d %dx%d, want order-5 560x420", browser.Canvas.FrameOrder, browser.Canvas.Width, browser.Canvas.Height))
	}
	if browser.Canvas.ArtifactKind != "runner-trace" {
		issues = append(issues, fmt.Sprintf("browser_surface canvas artifact_kind is %q, want runner-trace", browser.Canvas.ArtifactKind))
	}
	if !browser.Input.Pointer || !browser.Input.Keyboard || !browser.Input.Text || !browser.Input.Resize || !browser.Input.HostTrace || !browser.Input.Pass {
		issues = append(issues, "browser_surface input requires pointer, keyboard, text, resize, host_trace, and pass evidence")
	}
	issues = append(issues, validateBrowserSurfaceNativeEvents(browser.Input.NativeEvents)...)
	if browser.Clipboard.Harness != "deterministic-browser-clipboard-v1" {
		issues = append(issues, fmt.Sprintf("browser_surface clipboard.harness is %q, want deterministic-browser-clipboard-v1", browser.Clipboard.Harness))
	}
	if !browser.Clipboard.Read || !browser.Clipboard.Write || !browser.Clipboard.OwnedCopy || !browser.Clipboard.Pass || browser.Clipboard.Bytes <= 0 {
		issues = append(issues, "browser_surface clipboard requires read, write, owned_copy, positive bytes, and pass evidence")
	}
	if !browser.Composition.Start || !browser.Composition.Update || !browser.Composition.Commit || !browser.Composition.Cancel || !browser.Composition.Pass {
		issues = append(issues, "browser_surface composition requires start, update, commit, cancel, and pass evidence")
	}
	if !browser.Accessibility.Snapshot || !browser.Accessibility.Mirror || !browser.Accessibility.CompilerOwned || !browser.Accessibility.Bounds || !browser.Accessibility.Focus || !browser.Accessibility.Pass {
		issues = append(issues, "browser_surface accessibility requires snapshot, mirror, compiler_owned, bounds, focus, and pass evidence")
	}
	issues = append(issues, validateBrowserSurfaceRoles(browser.Accessibility.Roles)...)
	if browser.Accessibility.DOMVisualUI {
		issues = append(issues, "browser_surface accessibility.dom_visual_ui must be false")
	}
	if browser.Accessibility.UserJS {
		issues = append(issues, "browser_surface accessibility.user_js must be false")
	}
	if !browserSurfaceHostTraceContains(browser.HostTraces, "runner-trace") {
		issues = append(issues, "browser_surface host_traces requires passing runner-trace evidence")
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
			issues = append(issues, fmt.Sprintf("browser_surface negative_guards.%s must be true", check.field))
		}
	}
	return issues
}

func validateBrowserSurfaceNativeEvents(events []string) []string {
	var issues []string
	for _, required := range []string{"pointerup", "keydown", "beforeinput", "resize"} {
		if !stringSliceContainsFold(events, required) {
			issues = append(issues, fmt.Sprintf("browser_surface input.native_events requires %s", required))
		}
	}
	return issues
}

func validateBrowserSurfaceRoles(roles []string) []string {
	var issues []string
	for _, required := range []string{"root", "textbox", "checkbox", "button", "status"} {
		if !stringSliceContainsFold(roles, required) {
			issues = append(issues, fmt.Sprintf("browser_surface accessibility.roles requires %s", required))
		}
	}
	return issues
}

func browserSurfaceHostTraceContains(traces []BrowserSurfaceHostTraceReport, artifactKind string) bool {
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
