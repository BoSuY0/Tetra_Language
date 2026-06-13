package main

import (
	"encoding/json"
	"testing"

	"tetra_language/tools/validators/surface"
)

func TestComponentTreeModesProduceTreeEvidence(t *testing.T) {
	for _, mode := range []string{
		"headless-component-tree",
		"linux-x64-real-window-component-tree",
		"wasm32-web-browser-canvas-component-tree",
		"headless-component-tree-api",
		"linux-x64-real-window-component-tree-api",
		"wasm32-web-browser-canvas-component-tree-api",
	} {
		t.Run(mode, func(t *testing.T) {
			if err := validateSmokeMode(mode); err != nil {
				t.Fatalf("validateSmokeMode(%s) failed: %v", mode, err)
			}
			if got := defaultSurfaceSourcePath(smokeOptions{Mode: mode, SourcePath: "examples/surface_counter.tetra"}); got != "examples/surface_tree_app.tetra" {
				t.Fatalf("defaultSurfaceSourcePath(%s) = %q, want examples/surface_tree_app.tetra", mode, got)
			}
			scenario := runComponentTreeScenario(mode)
			if scenario.ComponentTree == nil {
				t.Fatalf("component_tree missing from scenario")
			}
			if scenario.ComponentTreeAPI == nil {
				t.Fatalf("component_tree_api missing from scenario")
			}
			if scenario.ComponentTreeAPI.APILevel != "builder-layout-dispatch-v1" || scenario.ComponentTreeAPI.ManualBookkeeping {
				t.Fatalf("component_tree_api = %#v, want hardened helper evidence without manual bookkeeping", scenario.ComponentTreeAPI)
			}
			if scenario.ComponentTree.NodeCount < 7 || len(scenario.ComponentTree.Nodes) < 7 {
				t.Fatalf("component_tree = %#v, want at least 7 nodes", scenario.ComponentTree)
			}
			if !intSlicesEqual(scenario.ComponentTree.FocusOrder, []int{3, 5, 6}) {
				t.Fatalf("focus_order = %#v, want TextBox -> SubmitButton -> ResetButton", scenario.ComponentTree.FocusOrder)
			}
			for _, want := range [][]int{{0, 1, 3}, {0, 1, 4, 5}, {0, 1, 4, 6}} {
				if !componentTreeDispatchPathsContain(scenario.ComponentTree.DispatchPaths, want) {
					t.Fatalf("dispatch_paths = %#v, want %v", scenario.ComponentTree.DispatchPaths, want)
				}
			}
			for _, want := range []string{
				"component tree node count",
				"component tree parent child links",
				"component tree pointer dispatch path",
				"component tree focus traversal",
				"component tree text routed to focused TextBox",
				"component tree button action dispatch",
				"component tree resize relayout",
				"component tree rendered frame update",
				"component tree api builder node creation",
				"component tree api parent child invariants",
				"component tree api layout helper dispatch",
				"component tree api hit test helper",
				"component tree api focus helper traversal",
				"component tree api dispatch path helper",
				"component tree api no manual bookkeeping",
			} {
				if !caseNamesContain(scenario.Cases, want) {
					t.Fatalf("cases = %#v, want %q", scenario.Cases, want)
				}
			}
			reportScenario := scenario
			reportScenario.Frames = append(reportScenario.Frames, componentTreeTestFrames(mode)...)
			if len(reportScenario.Frames) < 2 || reportScenario.Frames[0].Checksum == reportScenario.Frames[len(reportScenario.Frames)-1].Checksum {
				t.Fatalf("frames = %#v, want visible framebuffer update after tree input/resize", reportScenario.Frames)
			}
			raw, err := json.Marshal(buildReport(smokeOptions{Mode: mode, SourcePath: "examples/surface_tree_app.tetra"}, "linux-x64", componentTreeTestProcesses(mode), componentTreeTestArtifacts(mode), cleanArtifactScan(3), reportScenario))
			if err != nil {
				t.Fatalf("marshal component tree report: %v", err)
			}
			if err := surface.ValidateReport(raw); err != nil {
				t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
			}
		})
	}
}

func TestMinimalToolkitModesUseToolkitFormSource(t *testing.T) {
	for _, mode := range []string{
		"headless-minimal-toolkit",
		"linux-x64-real-window-minimal-toolkit",
		"wasm32-web-browser-canvas-minimal-toolkit",
	} {
		t.Run(mode, func(t *testing.T) {
			if err := validateSmokeMode(mode); err != nil {
				t.Fatalf("validateSmokeMode(%s) failed: %v", mode, err)
			}
			if got := defaultSurfaceSourcePath(smokeOptions{Mode: mode, SourcePath: "examples/surface_counter.tetra"}); got != "examples/surface_toolkit_form.tetra" {
				t.Fatalf("defaultSurfaceSourcePath(%s) = %q, want examples/surface_toolkit_form.tetra", mode, got)
			}
		})
	}
}

func TestToolkitReuseModesUseToolkitSettingsSource(t *testing.T) {
	for _, mode := range []string{
		"headless-toolkit-reuse",
		"linux-x64-real-window-toolkit-reuse",
		"wasm32-web-browser-canvas-toolkit-reuse",
	} {
		t.Run(mode, func(t *testing.T) {
			if err := validateSmokeMode(mode); err != nil {
				t.Fatalf("validateSmokeMode(%s) failed: %v", mode, err)
			}
			if got := defaultSurfaceSourcePath(smokeOptions{Mode: mode, SourcePath: "examples/surface_counter.tetra"}); got != "examples/surface_toolkit_settings.tetra" {
				t.Fatalf("defaultSurfaceSourcePath(%s) = %q, want examples/surface_toolkit_settings.tetra", mode, got)
			}
			scenario := runSurfaceScenario(mode)
			if scenario.Toolkit == nil {
				t.Fatalf("scenario.Toolkit is nil, want toolkit reuse evidence")
			}
			if scenario.Toolkit.ToolkitLevel != "toolkit-reuse-v1" {
				t.Fatalf("toolkit_level = %q, want toolkit-reuse-v1", scenario.Toolkit.ToolkitLevel)
			}
			textBoxes := 0
			buttons := 0
			for _, widget := range scenario.Toolkit.Widgets {
				switch widget.Kind {
				case "TextBox":
					textBoxes++
				case "Button":
					buttons++
				}
			}
			if textBoxes < 2 || buttons < 2 {
				t.Fatalf("toolkit widgets = %#v, want at least two TextBoxes and two Buttons", scenario.Toolkit.Widgets)
			}
			for _, want := range []string{
				"toolkit reuse second example evidence",
				"toolkit reuse multi TextBox routing",
				"toolkit reuse focused TextBox only mutates",
				"toolkit reuse StatusText updates",
				"toolkit reuse resize relayout",
			} {
				if !caseNamesContain(scenario.Cases, want) {
					t.Fatalf("cases = %#v, want %q", scenario.Cases, want)
				}
			}
		})
	}
}

func TestReleaseToolkitModesProduceProductionToolkitEvidence(t *testing.T) {
	for _, mode := range []string{
		"headless-release-toolkit",
		"linux-x64-release-toolkit",
		"wasm32-web-release-toolkit",
	} {
		t.Run(mode, func(t *testing.T) {
			if err := validateSmokeMode(mode); err != nil {
				t.Fatalf("validateSmokeMode(%s) failed: %v", mode, err)
			}
			if got := defaultSurfaceSourcePath(smokeOptions{Mode: mode, SourcePath: "examples/surface_counter.tetra"}); got != "examples/surface_release_form.tetra" {
				t.Fatalf("defaultSurfaceSourcePath(%s) = %q, want examples/surface_release_form.tetra", mode, got)
			}
			scenario := runSurfaceScenario(mode)
			if scenario.Toolkit == nil {
				t.Fatalf("scenario.Toolkit is nil, want production toolkit evidence")
			}
			if scenario.Toolkit.ToolkitLevel != "production-widgets-v1" {
				t.Fatalf("toolkit_level = %q, want production-widgets-v1", scenario.Toolkit.ToolkitLevel)
			}
			if scenario.Toolkit.Experimental || !scenario.Toolkit.ProductionClaim {
				t.Fatalf("toolkit release flags = experimental:%v production_claim:%v, want current production evidence", scenario.Toolkit.Experimental, scenario.Toolkit.ProductionClaim)
			}
			requiredKinds := map[string]bool{
				"Text": false, "Label": false, "StatusText": false, "Button": false,
				"TextBox": false, "Checkbox": false, "Row": false, "Column": false,
				"Panel": false, "Stack": false, "Scroll": false, "Spacer": false,
			}
			for _, widget := range scenario.Toolkit.Widgets {
				if _, ok := requiredKinds[widget.Kind]; ok {
					requiredKinds[widget.Kind] = true
				}
			}
			for kind, found := range requiredKinds {
				if !found {
					t.Fatalf("toolkit widgets = %#v, missing required kind %s", scenario.Toolkit.Widgets, kind)
				}
			}
			for _, want := range []string{
				"production toolkit required widget set",
				"production toolkit style module default theme",
				"production toolkit Checkbox toggle routed",
				"production toolkit Scroll offset routed",
				"production toolkit safe text storage",
			} {
				if !caseNamesContain(scenario.Cases, want) {
					t.Fatalf("cases = %#v, want %q", scenario.Cases, want)
				}
			}
		})
	}
}

func TestReleaseBrowserModeProducesBrowserCanvasReleaseEvidence(t *testing.T) {
	const mode = "wasm32-web-release-browser"
	if err := validateSmokeMode(mode); err != nil {
		t.Fatalf("validateSmokeMode(%s) failed: %v", mode, err)
	}
	if got := defaultSurfaceSourcePath(smokeOptions{Mode: mode, SourcePath: "examples/surface_counter.tetra"}); got != "examples/surface_release_form.tetra" {
		t.Fatalf("defaultSurfaceSourcePath(%s) = %q, want examples/surface_release_form.tetra", mode, got)
	}
	scenario := runSurfaceScenario(mode)
	if scenario.Toolkit == nil {
		t.Fatalf("scenario.Toolkit is nil, want production toolkit evidence")
	}
	for _, want := range []string{
		"browser release Surface v1 schema",
		"browser release Chromium canvas readback",
		"browser release native pointer keyboard text resize",
		"browser release deterministic clipboard harness",
		"browser release composition trace",
		"browser release accessibility snapshot mirror",
		"browser release forbidden web sidecar rejection",
	} {
		if !caseNamesContain(scenario.Cases, want) {
			t.Fatalf("cases = %#v, want %q", scenario.Cases, want)
		}
	}
	scenario.Frames = append(scenario.Frames, releaseBrowserTestFrames()...)
	raw, err := json.Marshal(buildReport(
		smokeOptions{Mode: mode, SourcePath: "examples/surface_release_form.tetra"},
		"linux-x64",
		releaseBrowserTestProcesses(),
		releaseBrowserTestArtifacts(),
		cleanArtifactScan(3),
		scenario,
	))
	if err != nil {
		t.Fatalf("marshal release browser report: %v", err)
	}
	var report surface.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode release browser report: %v", err)
	}
	if report.Target != "wasm32-web" {
		t.Fatalf("target = %q, want wasm32-web", report.Target)
	}
	if report.HostEvidence.Level != "wasm32-web-browser-canvas-release-v1" ||
		report.HostEvidence.Backend != "browser-canvas-rgba-accessible" ||
		!report.HostEvidence.BrowserCanvas ||
		!report.HostEvidence.BrowserInput ||
		!report.HostEvidence.BrowserClipboard ||
		report.HostEvidence.BrowserClipboardHarness != "deterministic-browser-clipboard-v1" ||
		!report.HostEvidence.BrowserComposition ||
		!report.HostEvidence.BrowserAccessibilitySnapshot ||
		!report.HostEvidence.BrowserAccessibilityMirror {
		t.Fatalf("host evidence = %#v, want strict browser release evidence", report.HostEvidence)
	}
	if report.BrowserSurface == nil ||
		report.BrowserSurface.Schema != surface.BrowserSurfaceSchemaV1 ||
		report.BrowserSurface.BrowserSurfaceLevel != "browser-canvas-release-v1" ||
		!report.BrowserSurface.DOMHostCanvasOnly ||
		!report.BrowserSurface.NegativeGuards.NoDOMAppUITree ||
		!report.BrowserSurface.NegativeGuards.NoUserJSAppLogic ||
		!report.BrowserSurface.NegativeGuards.NoNodeOnlyPromotion {
		t.Fatalf("browser surface evidence = %#v, want strict browser_surface P13 evidence", report.BrowserSurface)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed for release browser: %v\n%s", err, raw)
	}
}

func TestLinuxX64ReleaseWindowModeProducesReleaseEvidence(t *testing.T) {
	const mode = "linux-x64-release-window"
	if err := validateSmokeMode(mode); err != nil {
		t.Fatalf("validateSmokeMode(%s) failed: %v", mode, err)
	}
	if got := defaultSurfaceSourcePath(smokeOptions{Mode: mode, SourcePath: "examples/surface_counter.tetra"}); got != "examples/surface_release_form.tetra" {
		t.Fatalf("defaultSurfaceSourcePath(%s) = %q, want examples/surface_release_form.tetra", mode, got)
	}
	scenario := runSurfaceScenario(mode)
	if scenario.Toolkit == nil {
		t.Fatalf("scenario.Toolkit is nil, want production toolkit evidence")
	}
	if scenario.AccessibilityTree == nil {
		t.Fatalf("scenario.AccessibilityTree is nil, want linux release accessibility bridge evidence")
	}
	for _, want := range []string{
		"linux release window v1 schema",
		"linux release real window presented frame",
		"linux release native pointer key text resize close",
		"linux release clipboard harness",
		"linux release composition harness",
		"linux release accessibility bridge probe",
		"linux release forbids memfd starter promotion",
	} {
		if !caseNamesContain(scenario.Cases, want) {
			t.Fatalf("cases = %#v, want %q", scenario.Cases, want)
		}
	}
	scenario.Frames = releaseWindowTestFrames()
	raw, err := json.Marshal(buildReport(
		smokeOptions{Mode: mode, SourcePath: "examples/surface_release_form.tetra"},
		"linux-x64",
		releaseWindowTestProcesses(),
		releaseWindowTestArtifacts(),
		cleanArtifactScan(3),
		scenario,
	))
	if err != nil {
		t.Fatalf("marshal release window report: %v", err)
	}
	var report surface.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode release window report: %v", err)
	}
	if report.Target != "linux-x64" {
		t.Fatalf("target = %q, want linux-x64", report.Target)
	}
	if report.HostEvidence.Level != "linux-x64-release-window-v1" ||
		report.HostEvidence.Backend != "wayland-shm-rgba-release-v1" ||
		!report.HostEvidence.Framebuffer ||
		!report.HostEvidence.RealWindow ||
		!report.HostEvidence.NativeInput ||
		!report.HostEvidence.TextInput ||
		!report.HostEvidence.Clipboard ||
		!report.HostEvidence.Composition ||
		!report.HostEvidence.AccessibilityBridge {
		t.Fatalf("host evidence = %#v, want strict linux release window evidence", report.HostEvidence)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed for release window: %v\n%s", err, raw)
	}
}

func TestReleaseAccessibilityModesProducePlatformBridgeEvidence(t *testing.T) {
	for _, tc := range []struct {
		mode       string
		wantTarget string
	}{
		{mode: "headless-release-accessibility", wantTarget: "headless"},
		{mode: "linux-x64-release-accessibility", wantTarget: "linux-x64"},
		{mode: "wasm32-web-release-accessibility", wantTarget: "wasm32-web"},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			if err := validateSmokeMode(tc.mode); err != nil {
				t.Fatalf("validateSmokeMode(%s) failed: %v", tc.mode, err)
			}
			if got := defaultSurfaceSourcePath(smokeOptions{Mode: tc.mode, SourcePath: "examples/surface_counter.tetra"}); got != "examples/surface_release_accessibility.tetra" {
				t.Fatalf("defaultSurfaceSourcePath(%s) = %q, want examples/surface_release_accessibility.tetra", tc.mode, got)
			}
			scenario := runSurfaceScenario(tc.mode)
			if scenario.AccessibilityTree == nil {
				t.Fatalf("scenario.AccessibilityTree is nil, want platform bridge evidence")
			}
			tree := scenario.AccessibilityTree
			if tree.AccessibilityLevel != "platform-bridge-v1" {
				t.Fatalf("accessibility_level = %q, want platform-bridge-v1", tree.AccessibilityLevel)
			}
			if tree.ReleaseScope != "surface-v1-linux-web" {
				t.Fatalf("release_scope = %q, want surface-v1-linux-web", tree.ReleaseScope)
			}
			if tree.Experimental || !tree.ProductionClaim {
				t.Fatalf("release accessibility flags = experimental:%v production_claim:%v, want current production evidence", tree.Experimental, tree.ProductionClaim)
			}
			for _, want := range []string{
				"accessibility platform bridge v1 schema",
				"accessibility platform export from metadata tree",
				"accessibility release honest screen reader evidence",
			} {
				if !caseNamesContain(scenario.Cases, want) {
					t.Fatalf("cases = %#v, want %q", scenario.Cases, want)
				}
			}
			processes := releaseAccessibilityTestProcesses(tc.mode)
			artifacts := releaseAccessibilityTestArtifacts(tc.mode)
			scenario.Frames = append(scenario.Frames, releaseAccessibilityTestFrames(tc.mode)...)
			raw, err := json.Marshal(buildReport(smokeOptions{Mode: tc.mode, SourcePath: "examples/surface_release_accessibility.tetra"}, "linux-x64", processes, artifacts, cleanArtifactScan(len(artifacts)), scenario))
			if err != nil {
				t.Fatalf("marshal release accessibility report: %v", err)
			}
			var envelope struct {
				Target string `json:"target"`
			}
			if err := json.Unmarshal(raw, &envelope); err != nil {
				t.Fatalf("decode release accessibility report: %v", err)
			}
			if envelope.Target != tc.wantTarget {
				t.Fatalf("target = %q, want %q", envelope.Target, tc.wantTarget)
			}
			if err := surface.ValidateReport(raw); err != nil {
				t.Fatalf("ValidateReport failed for %s: %v\n%s", tc.mode, err, raw)
			}
		})
	}
}
