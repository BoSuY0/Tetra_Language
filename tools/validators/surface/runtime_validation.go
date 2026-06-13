package surface

import (
	"fmt"
	"strings"
)

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
			issues = append(issues, fmt.Sprintf("headless host_evidence.level is %q, want deterministic-headless", evidence.Level))
		}
		if evidence.Backend != "software-rgba" {
			issues = append(issues, fmt.Sprintf("headless host_evidence.backend is %q, want software-rgba", evidence.Backend))
		}
		if !evidence.Framebuffer {
			issues = append(issues, "headless host_evidence requires framebuffer=true")
		}
		if evidence.RealWindow || evidence.NativeInput {
			issues = append(issues, "headless host_evidence must not claim real_window or native_input")
		}
	case "linux-x64":
		switch evidence.Level {
		case "linux-x64-memfd-starter":
			if evidence.Backend != "memfd-rgba" {
				issues = append(issues, fmt.Sprintf("linux-x64 memfd starter host_evidence.backend is %q, want memfd-rgba", evidence.Backend))
			}
			if !evidence.Framebuffer {
				issues = append(issues, "linux-x64 memfd starter host_evidence requires framebuffer=true")
			}
			if evidence.RealWindow || evidence.NativeInput {
				issues = append(issues, "linux-x64 memfd starter host_evidence must not claim real_window or native_input")
			}
		case "linux-x64-real-window":
			if !evidence.Framebuffer || !evidence.RealWindow || !evidence.NativeInput {
				issues = append(issues, "linux-x64 real-window host_evidence requires framebuffer=true, real_window=true, and native_input=true")
			}
			if evidence.Backend == "memfd-rgba" || evidence.Backend == "software-rgba" || evidence.Backend == "node-surface-host" {
				issues = append(issues, fmt.Sprintf("linux-x64 real-window host_evidence.backend %q is not real-window evidence", evidence.Backend))
			}
			if !hasAppProcessWithExpectedExit(report.Processes, "surface linux-x64 real-window probe", 42) {
				issues = append(issues, "linux-x64 real-window host_evidence requires a Surface real-window probe app exiting 42")
			}
			if !caseNameContains(report.Cases, "linux-x64 real-window surface") {
				issues = append(issues, "linux-x64 real-window host_evidence requires linux-x64 real-window surface case evidence")
			}
			if !caseNameContains(report.Cases, "linux-x64 native input event pump") {
				issues = append(issues, "linux-x64 real-window host_evidence requires linux-x64 native input event pump case evidence")
			}
		case "linux-x64-release-window-v1":
			if evidence.Backend != "wayland-shm-rgba-release-v1" {
				issues = append(issues, fmt.Sprintf("linux release host_evidence.backend is %q, want wayland-shm-rgba-release-v1", evidence.Backend))
			}
			if !evidence.Framebuffer || !evidence.RealWindow || !evidence.NativeInput {
				issues = append(issues, "linux release host_evidence requires framebuffer=true, real_window=true, and native_input=true")
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
				issues = append(issues, "linux release host_evidence.accessibility_bridge must be true")
			}
			if !caseNameContains(report.Cases, "linux release real window presented frame") {
				issues = append(issues, "linux release host_evidence requires real window presented frame case evidence")
			}
			if !caseNameContains(report.Cases, "linux release accessibility bridge probe") {
				issues = append(issues, "linux release host_evidence requires accessibility bridge probe case evidence")
			}
		default:
			issues = append(issues, fmt.Sprintf("linux-x64 host_evidence.level is %q, want linux-x64-memfd-starter, linux-x64-real-window, or linux-x64-release-window-v1", evidence.Level))
		}
	case "wasm32-web":
		switch evidence.Level {
		case "wasm32-web-compiler-owned-loader":
			if evidence.Backend != "node-surface-host" {
				issues = append(issues, fmt.Sprintf("wasm32-web starter host_evidence.backend is %q, want node-surface-host", evidence.Backend))
			}
			if !evidence.Framebuffer {
				issues = append(issues, "wasm32-web starter host_evidence requires framebuffer=true")
			}
			if evidence.RealWindow || evidence.NativeInput {
				issues = append(issues, "wasm32-web starter host_evidence must not claim browser canvas native input")
			}
		case "wasm32-web-browser-canvas-input":
			if evidence.Backend != "browser-canvas-rgba" {
				issues = append(issues, fmt.Sprintf("wasm32-web browser canvas host_evidence.backend is %q, want browser-canvas-rgba", evidence.Backend))
			}
			if !evidence.Framebuffer || !evidence.NativeInput {
				issues = append(issues, "wasm32-web browser canvas host_evidence requires framebuffer=true and native_input=true")
			}
			if evidence.RealWindow {
				issues = append(issues, "wasm32-web browser canvas host_evidence must not claim OS real_window")
			}
			if !caseNameContains(report.Cases, "wasm32-web browser canvas surface") {
				issues = append(issues, "wasm32-web browser canvas host_evidence requires browser canvas surface case evidence")
			}
			if !caseNameContains(report.Cases, "wasm32-web browser canvas RGBA readback") {
				issues = append(issues, "wasm32-web browser canvas host_evidence requires canvas RGBA readback case evidence")
			}
		case "wasm32-web-browser-canvas-release-v1":
			if evidence.Backend != "browser-canvas-rgba-accessible" {
				issues = append(issues, fmt.Sprintf("browser release host_evidence.backend is %q, want browser-canvas-rgba-accessible", evidence.Backend))
			}
			if !evidence.Framebuffer || !evidence.NativeInput {
				issues = append(issues, "browser release host_evidence requires framebuffer=true and native_input=true")
			}
			if evidence.RealWindow {
				issues = append(issues, "browser release host_evidence must not claim OS real_window")
			}
			if !evidence.BrowserCanvas {
				issues = append(issues, "browser release host_evidence.browser_canvas must be true")
			}
			if !evidence.BrowserInput {
				issues = append(issues, "browser release host_evidence.browser_input must be true")
			}
			if !evidence.BrowserClipboard {
				issues = append(issues, "browser release host_evidence.browser_clipboard must be true")
			}
			if evidence.BrowserClipboardHarness != "deterministic-browser-clipboard-v1" {
				issues = append(issues, fmt.Sprintf("browser release host_evidence.browser_clipboard_harness is %q, want deterministic-browser-clipboard-v1", evidence.BrowserClipboardHarness))
			}
			if !evidence.BrowserComposition {
				issues = append(issues, "browser release host_evidence.browser_composition must be true")
			}
			if !evidence.BrowserAccessibilitySnapshot {
				issues = append(issues, "browser release host_evidence.browser_accessibility_snapshot must be true")
			}
			if !evidence.BrowserAccessibilityMirror {
				issues = append(issues, "browser release host_evidence.browser_accessibility_mirror must be true")
			}
			if !caseNameContains(report.Cases, "wasm32-web browser canvas surface") {
				issues = append(issues, "browser release host_evidence requires browser canvas surface case evidence")
			}
			if !caseNameContains(report.Cases, "wasm32-web browser canvas RGBA readback") {
				issues = append(issues, "browser release host_evidence requires canvas RGBA readback case evidence")
			}
		default:
			issues = append(issues, fmt.Sprintf("wasm32-web host_evidence.level is %q, want wasm32-web-compiler-owned-loader, wasm32-web-browser-canvas-input, or wasm32-web-browser-canvas-release-v1", evidence.Level))
		}
	}
	return issues
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
			issues = append(issues, fmt.Sprintf("report contains forbidden non-runtime evidence marker %q", strings.Trim(marker, " /\"")))
		}
	}
	return issues
}

func validateTargetRuntimeEvidence(report Report) []string {
	var issues []string
	switch report.Target {
	case "headless":
		if report.Runtime != "surface-headless" {
			issues = append(issues, fmt.Sprintf("headless target runtime is %q, want surface-headless", report.Runtime))
		}
		if !hasRuntimeProcessName(report.Processes, "headless") {
			issues = append(issues, "headless target requires a headless Surface runtime process")
		}
		if !caseNameContains(report.Cases, "headless event dispatch") {
			issues = append(issues, "headless target requires headless event dispatch evidence")
		}
		if !caseNameContains(report.Cases, "headless actual runner trace") {
			issues = append(issues, "headless target requires headless actual runner trace evidence")
		}
		if isAccessibilityMetadataReport(report) {
			if !hasFrameOrderDimensions(report.Frames, 5, 480, 320, 1920) {
				issues = append(issues, "headless accessibility metadata target requires order-5 480x320 resized headless runner trace frame evidence")
			}
		} else if isProductionToolkitReport(report) {
			if !hasFrameOrderDimensions(report.Frames, 5, 560, 420, 2240) {
				issues = append(issues, "headless production toolkit target requires order-5 560x420 resized headless runner trace frame evidence")
			}
		} else if isToolkitReuseReport(report) {
			if !hasFrameOrderDimensions(report.Frames, 5, 480, 320, 1920) {
				issues = append(issues, "headless toolkit reuse target requires order-5 480x320 resized headless runner trace frame evidence")
			}
		} else if isMinimalToolkitReport(report) {
			if !hasFrameOrderDimensions(report.Frames, 4, 400, 240, 1600) {
				issues = append(issues, "headless minimal toolkit target requires order-4 400x240 resized headless runner trace frame evidence")
			}
		} else if isComponentTreeReport(report) {
			if !hasFrameOrderDimensions(report.Frames, 2, 400, 240, 1600) {
				issues = append(issues, "headless component tree target requires order-2 400x240 resized headless runner trace frame evidence")
			}
		} else if isTextFocusInputReport(report) {
			if !hasFrameOrderDimensions(report.Frames, 2, 400, 240, 1600) {
				issues = append(issues, "headless text focus input target requires order-2 400x240 resized headless runner trace frame evidence")
			}
		} else if !hasFrameOrderDimensions(report.Frames, 2, 320, 200, 1280) {
			issues = append(issues, "headless target requires order-2 320x200 headless runner trace frame evidence")
		}
	case "linux-x64":
		if report.Runtime != "surface-linux-x64" {
			issues = append(issues, fmt.Sprintf("linux-x64 target runtime is %q, want surface-linux-x64", report.Runtime))
		}
		if !hasRuntimeProcessName(report.Processes, "linux-x64") {
			issues = append(issues, "linux-x64 target requires a linux-x64 Surface runtime process")
		}
		if isLinuxRealWindowHostEvidenceLevel(report.HostEvidence.Level) {
			if !hasAppProcessWithExpectedExit(report.Processes, "surface linux-x64 real-window probe", 42) {
				issues = append(issues, "linux-x64 real-window target requires a Surface real-window probe app exiting 42")
			}
			if !caseNameContains(report.Cases, "linux-x64 real-window surface") {
				issues = append(issues, "linux-x64 real-window target requires real-window surface evidence")
			}
			if !caseNameContains(report.Cases, "linux-x64 native input event pump") {
				issues = append(issues, "linux-x64 real-window target requires native input event pump evidence")
			}
			if !caseNameContains(report.Cases, "linux-x64 real-window resize event") {
				issues = append(issues, "linux-x64 real-window target requires resize event evidence")
			}
			if !caseNameContains(report.Cases, "linux-x64 real-window close event") {
				issues = append(issues, "linux-x64 real-window target requires close event evidence")
			}
			if isLinuxAppShellReport(report) {
				if !hasFrameOrderDimensions(report.Frames, 6, 720, 540, 2880) {
					issues = append(issues, "linux-x64 app-shell target requires order-6 720x540 presented app-shell frame evidence")
				}
			} else if isLinuxReleaseWindowReport(report) {
				if !hasFrameOrderDimensions(report.Frames, 5, 560, 420, 2240) {
					issues = append(issues, "linux-x64 release-window target requires order-5 560x420 presented window frame evidence")
				}
			} else if isAccessibilityMetadataReport(report) {
				if !hasFrameOrderDimensions(report.Frames, 5, 480, 320, 1920) {
					issues = append(issues, "linux-x64 real-window accessibility metadata target requires order-5 480x320 presented window frame evidence")
				}
			} else if isProductionToolkitReport(report) {
				if !hasFrameOrderDimensions(report.Frames, 5, 560, 420, 2240) {
					issues = append(issues, "linux-x64 real-window production toolkit target requires order-5 560x420 presented window frame evidence")
				}
			} else if isToolkitReuseReport(report) {
				if !hasFrameOrderDimensions(report.Frames, 5, 480, 320, 1920) {
					issues = append(issues, "linux-x64 real-window toolkit reuse target requires order-5 480x320 presented window frame evidence")
				}
			} else if !hasFrameOrderDimensions(report.Frames, 5, 400, 240, 1600) {
				issues = append(issues, "linux-x64 real-window target requires order-5 400x240 presented window frame evidence")
			}
		} else {
			if !hasAppProcessWithExpectedExit(report.Processes, "surface linux-x64 host probe", 42) {
				issues = append(issues, "linux-x64 target requires a Surface Host ABI probe app exiting 42")
			}
			if !caseNameContains(report.Cases, "linux-x64 Surface Host ABI") {
				issues = append(issues, "linux-x64 target requires linux-x64 Surface Host ABI evidence")
			}
			if !hasAppProcessWithExpectedExit(report.Processes, "surface linux-x64 event sequence probe", 42) {
				issues = append(issues, "linux-x64 target requires a Surface event sequence probe app exiting 42")
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
			if !hasAppProcessWithExpectedExit(report.Processes, "surface linux-x64 counter app presented frame probe", -1) {
				issues = append(issues, "linux-x64 target requires a counter component app-presented frame probe process")
			}
			if !caseNameContains(report.Cases, "linux-x64 counter component app-presented frame") {
				issues = append(issues, "linux-x64 target requires counter component app-presented frame evidence")
			}
			if !hasFrameOrderDimensions(report.Frames, 4, 320, 200, 1280) {
				issues = append(issues, "linux-x64 target requires order-4 320x200 counter component app-presented frame evidence")
			}
		}
		if caseNameContains(report.Cases, "headless") {
			issues = append(issues, "linux-x64 target must not use headless runtime case evidence")
		}
	case "wasm32-web":
		if report.Runtime != "surface-wasm32-web" {
			issues = append(issues, fmt.Sprintf("wasm32-web target runtime is %q, want surface-wasm32-web", report.Runtime))
		}
		if !hasRuntimeProcessName(report.Processes, "wasm32-web") {
			issues = append(issues, "wasm32-web target requires a wasm32-web Surface runtime process")
		}
		if !hasProcessNameAndPathMarkers(report.Processes, "runtime", "surface wasm32-web import validator", "validate-wasm-imports", "--target wasm32-web") {
			issues = append(issues, "wasm32-web target requires validate-wasm-imports runtime process for Surface Host ABI import allowlist")
		}
		if !caseNameContains(report.Cases, "wasm32-web Surface Host ABI imports") {
			issues = append(issues, "wasm32-web target requires wasm32-web Surface Host ABI import evidence")
		}
		if !caseNameContains(report.Cases, "compiler-owned wasm Surface loader") {
			issues = append(issues, "wasm32-web target requires compiler-owned wasm Surface loader evidence")
		}
		if isBrowserCanvasHostEvidenceLevel(report.HostEvidence.Level) {
			for _, required := range []string{
				"wasm32-web browser canvas surface",
				"wasm32-web browser canvas RGBA readback",
				"wasm32-web browser canvas pointer input",
				"wasm32-web browser canvas keyboard input",
				"wasm32-web browser canvas resize input",
				"wasm32-web browser canvas text input",
				"compiler-owned browser canvas Surface host",
			} {
				if !caseNameContains(report.Cases, required) {
					issues = append(issues, fmt.Sprintf("wasm32-web browser canvas target requires %s evidence", required))
				}
			}
			if !hasProcessNameAndPathMarkers(report.Processes, "app", "surface wasm32-web browser canvas component app", "chromium") &&
				!hasProcessNameAndPathMarkers(report.Processes, "app", "surface wasm32-web browser canvas component app", "chrome") {
				issues = append(issues, "wasm32-web browser canvas target requires Chromium-compatible browser app process evidence")
			}
			for _, kind := range []string{"mouse_up", "key_down", "resize", "text_input"} {
				if !eventKindContains(report.Events, kind) {
					issues = append(issues, fmt.Sprintf("wasm32-web browser canvas target requires %s event evidence", kind))
				}
			}
			if isAccessibilityMetadataReport(report) {
				if !hasFrameOrderDimensions(report.Frames, 5, 480, 320, 1920) {
					issues = append(issues, "wasm32-web browser canvas accessibility metadata target requires order-5 480x320 canvas readback frame evidence")
				}
			} else if isProductionToolkitReport(report) {
				if !hasFrameOrderDimensions(report.Frames, 5, 560, 420, 2240) {
					issues = append(issues, "wasm32-web browser canvas production toolkit target requires order-5 560x420 canvas readback frame evidence")
				}
			} else if isToolkitReuseReport(report) {
				if !hasFrameOrderDimensions(report.Frames, 5, 480, 320, 1920) {
					issues = append(issues, "wasm32-web browser canvas toolkit reuse target requires order-5 480x320 canvas readback frame evidence")
				}
			} else if !hasFrameOrderDimensions(report.Frames, 5, 400, 240, 1600) {
				issues = append(issues, "wasm32-web browser canvas target requires order-5 400x240 canvas readback frame evidence")
			}
		} else {
			if !caseNameContains(report.Cases, "wasm32-web actual presented frame trace") {
				issues = append(issues, "wasm32-web target requires actual presented frame trace evidence")
			}
			if !hasFrameOrderDimensions(report.Frames, 4, 320, 200, 1280) {
				issues = append(issues, "wasm32-web target requires order-4 320x200 actual presented frame trace evidence")
			}
		}
		if caseNameContains(report.Cases, "headless") {
			issues = append(issues, "wasm32-web target must not use headless runtime case evidence")
		}
	}
	return issues
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
			issues = append(issues, fmt.Sprintf("text focus input report requires %s evidence", required))
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
		issues = append(issues, "text focus input report requires backspace and delete key evidence")
	}
	if !hasEventTargetKind(report.Events, "SubmitButton", "key_down") {
		issues = append(issues, "text focus input report requires keyboard event routed to focused SubmitButton")
	}
	if !hasResizePreservingFocus(report.Events) {
		issues = append(issues, "text focus input report requires resize preserving focused component state")
	}
	if !hasTransition(report.StateTransitions, "TextBox", "buffer") || !hasTransition(report.StateTransitions, "TextBox", "caret") {
		issues = append(issues, "text focus input report requires TextBox buffer and caret state transitions")
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
