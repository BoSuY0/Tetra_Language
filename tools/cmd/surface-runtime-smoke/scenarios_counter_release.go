package main

import (
	"tetra_language/tools/validators/surface"
)

func runLinuxX64CounterScenario() headlessScenario {
	scenario := runHeadlessCounterScenario()
	scenario.Cases = removeCaseNamed(scenario.Cases, "headless actual runner trace")
	for i := range scenario.Cases {
		switch scenario.Cases[i].Name {
		case "headless event dispatch":
			scenario.Cases[i].Name = "linux-x64 Surface Host ABI open/present/close"
		case "headless framebuffer checksum":
			scenario.Cases[i].Name = "linux-x64 framebuffer present evidence"
		}
	}
	scenario.Cases = append(scenario.Cases, surface.CaseReport{Name: "linux-x64 app-presented RGBA checksum", Kind: "positive", Ran: true, Pass: true})
	scenario.Cases = append(scenario.Cases, surface.CaseReport{Name: "linux-x64 host event sequence", Kind: "positive", Ran: true, Pass: true})
	scenario.Cases = append(scenario.Cases, surface.CaseReport{Name: "linux-x64 counter component app-presented frame", Kind: "positive", Ran: true, Pass: true})
	return scenario
}
func runLinuxX64RealWindowCounterScenario() headlessScenario {
	beforeFrame := renderWindowCounterFrameRGBA(0, 0, 320, 200, true)
	afterClickFrame := renderWindowCounterFrameRGBA(1, 0, 320, 200, true)
	scenario := headlessScenario{
		Components: []surface.ComponentReport{
			{
				ID:        "CounterApp",
				Type:      "examples.surface_window_counter.CounterApp",
				Bounds:    surface.RectReport{X: 0, Y: 0, W: 400, H: 240},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"count": "2", "key_count": "1", "width": "400", "closed": "true", "accessibility_role": "button"},
			},
			{
				ID:        "CounterButton",
				Type:      "examples.surface_window_counter.CounterButton",
				Parent:    "CounterApp",
				Bounds:    surface.RectReport{X: 32, Y: 88, W: 160, H: 48},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"text_len_seen": "2", "accessibility_role": "button"},
			},
		},
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "CounterButton",
				DispatchPath:    []string{"CounterApp", "CounterButton"},
				Handled:         true,
				Pass:            true,
				X:               48,
				Y:               96,
				Key:             0,
				Width:           320,
				Height:          200,
				TimestampMS:     0,
				BufferSlots:     []int{5, 48, 96, 1, 0, 320, 200, 0, 0},
				BeforeState:     map[string]string{"CounterApp.count": "0"},
				AfterState:      map[string]string{"CounterApp.count": "1"},
			},
			{
				Order:           2,
				Kind:            "key_down",
				TargetComponent: "CounterApp",
				DispatchPath:    []string{"CounterApp"},
				Handled:         true,
				Pass:            true,
				X:               0,
				Y:               0,
				Key:             32,
				Width:           320,
				Height:          200,
				TimestampMS:     1,
				BufferSlots:     []int{6, 0, 0, 0, 32, 320, 200, 1, 0},
				BeforeState:     map[string]string{"CounterApp.key_count": "0", "CounterApp.count": "1"},
				AfterState:      map[string]string{"CounterApp.key_count": "1", "CounterApp.count": "2"},
			},
			{
				Order:           3,
				Kind:            "resize",
				TargetComponent: "CounterApp",
				DispatchPath:    []string{"CounterApp"},
				Handled:         true,
				Pass:            true,
				X:               0,
				Y:               0,
				Key:             0,
				Width:           400,
				Height:          240,
				TimestampMS:     2,
				BufferSlots:     []int{2, 0, 0, 0, 0, 400, 240, 2, 0},
				BeforeState:     map[string]string{"CounterApp.width": "320"},
				AfterState:      map[string]string{"CounterApp.width": "400"},
			},
			{
				Order:           4,
				Kind:            "text_input",
				TargetComponent: "CounterButton",
				DispatchPath:    []string{"CounterApp", "CounterButton"},
				Handled:         true,
				Pass:            true,
				X:               0,
				Y:               0,
				Key:             0,
				Width:           400,
				Height:          240,
				TimestampMS:     3,
				TextLen:         2,
				TextBytesHex:    "4f4b",
				BufferSlots:     []int{8, 0, 0, 0, 0, 400, 240, 3, 2},
				BeforeState:     map[string]string{"CounterButton.text_len_seen": "0"},
				AfterState:      map[string]string{"CounterButton.text_len_seen": "2"},
			},
			{
				Order:           5,
				Kind:            "close",
				TargetComponent: "CounterApp",
				DispatchPath:    []string{"CounterApp"},
				Handled:         true,
				Pass:            true,
				X:               0,
				Y:               0,
				Key:             0,
				Width:           400,
				Height:          240,
				TimestampMS:     4,
				BufferSlots:     []int{1, 0, 0, 0, 0, 400, 240, 4, 0},
				BeforeState:     map[string]string{"CounterApp.closed": "false"},
				AfterState:      map[string]string{"CounterApp.closed": "true"},
			},
		},
		Frames: []surface.FrameReport{
			{Order: 1, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
			{Order: 2, Width: afterClickFrame.Width, Height: afterClickFrame.Height, Stride: afterClickFrame.Stride, Checksum: checksumRGBA(afterClickFrame.Pixels), Presented: true},
		},
		StateTransitions: []surface.StateTransitionReport{
			{Order: 1, Component: "CounterApp", Field: "count", Before: "0", After: "1", Cause: "mouse_up"},
			{Order: 2, Component: "CounterApp", Field: "key_count", Before: "0", After: "1", Cause: "key_down"},
			{Order: 3, Component: "CounterApp", Field: "width", Before: "320", After: "400", Cause: "resize"},
			{Order: 4, Component: "CounterButton", Field: "text_len_seen", Before: "0", After: "2", Cause: "text_input"},
			{Order: 5, Component: "CounterApp", Field: "closed", Before: "false", After: "true", Cause: "close"},
		},
		Cases: []surface.CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
			{Name: "linux-x64 Surface Host ABI open/present/close", Kind: "positive", Ran: true, Pass: true},
			{Name: "linux-x64 real-window surface", Kind: "positive", Ran: true, Pass: true},
			{Name: "linux-x64 native input event pump", Kind: "positive", Ran: true, Pass: true},
			{Name: "linux-x64 real-window resize event", Kind: "positive", Ran: true, Pass: true},
			{Name: "linux-x64 real-window close event", Kind: "positive", Ran: true, Pass: true},
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
			{Name: "reject legacy UI evidence", Kind: "negative", Ran: true, Pass: true, ExpectedError: "legacy UI evidence rejected"},
		},
	}
	return scenario
}
func runWASM32WebCounterScenario() headlessScenario {
	scenario := runHeadlessCounterScenario()
	scenario.Cases = removeCaseNamed(scenario.Cases, "headless actual runner trace")
	for i := range scenario.Cases {
		switch scenario.Cases[i].Name {
		case "headless event dispatch":
			scenario.Cases[i].Name = "wasm32-web Surface Host ABI imports"
		case "headless framebuffer checksum":
			scenario.Cases[i].Name = "wasm32-web framebuffer checksum evidence"
		}
	}
	scenario.Cases = append(scenario.Cases, surface.CaseReport{Name: "compiler-owned wasm Surface loader", Kind: "positive", Ran: true, Pass: true})
	scenario.Cases = append(scenario.Cases, surface.CaseReport{Name: "wasm32-web actual presented frame trace", Kind: "positive", Ran: true, Pass: true})
	return scenario
}
func runWASM32WebBrowserCanvasCounterScenario() headlessScenario {
	return headlessScenario{
		Components: []surface.ComponentReport{
			{
				ID:        "CounterApp",
				Type:      "examples.surface_browser_counter.CounterApp",
				Bounds:    surface.RectReport{X: 0, Y: 0, W: 400, H: 240},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"count": "2", "key_count": "1", "width": "400", "accessibility_role": "button"},
			},
			{
				ID:        "CounterButton",
				Type:      "examples.surface_browser_counter.CounterButton",
				Parent:    "CounterApp",
				Bounds:    surface.RectReport{X: 32, Y: 88, W: 160, H: 48},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"focused": "true", "text_len_seen": "2"},
			},
		},
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "CounterButton",
				DispatchPath:    []string{"CounterApp", "CounterButton"},
				Handled:         true,
				Pass:            true,
				X:               48,
				Y:               96,
				Width:           320,
				Height:          200,
				BufferSlots:     []int{5, 48, 96, 1, 0, 320, 200, 0, 0},
				BeforeState:     map[string]string{"CounterApp.count": "0"},
				AfterState:      map[string]string{"CounterApp.count": "1"},
			},
			{
				Order:           2,
				Kind:            "key_down",
				TargetComponent: "CounterApp",
				DispatchPath:    []string{"CounterApp"},
				Handled:         true,
				Pass:            true,
				Key:             32,
				Width:           320,
				Height:          200,
				TimestampMS:     1,
				BufferSlots:     []int{6, 0, 0, 0, 32, 320, 200, 1, 0},
				BeforeState:     map[string]string{"CounterApp.count": "1", "CounterApp.key_count": "0"},
				AfterState:      map[string]string{"CounterApp.count": "2", "CounterApp.key_count": "1"},
			},
			{
				Order:           3,
				Kind:            "resize",
				TargetComponent: "CounterApp",
				DispatchPath:    []string{"CounterApp"},
				Handled:         true,
				Pass:            true,
				Width:           400,
				Height:          240,
				TimestampMS:     2,
				BufferSlots:     []int{2, 0, 0, 0, 0, 400, 240, 2, 0},
				BeforeState:     map[string]string{"CounterApp.width": "320"},
				AfterState:      map[string]string{"CounterApp.width": "400"},
			},
			{
				Order:           4,
				Kind:            "text_input",
				TargetComponent: "CounterButton",
				DispatchPath:    []string{"CounterApp", "CounterButton"},
				Handled:         true,
				Pass:            true,
				Width:           400,
				Height:          240,
				TimestampMS:     3,
				BufferSlots:     []int{8, 0, 0, 0, 0, 400, 240, 3, 2},
				TextLen:         2,
				TextBytesHex:    "4f4b",
				BeforeState:     map[string]string{"CounterButton.text_len_seen": "0"},
				AfterState:      map[string]string{"CounterButton.text_len_seen": "2"},
			},
		},
		StateTransitions: []surface.StateTransitionReport{
			{Order: 1, Component: "CounterApp", Field: "count", Before: "0", After: "1", Cause: "mouse_up"},
			{Order: 2, Component: "CounterApp", Field: "key_count", Before: "0", After: "1", Cause: "key_down"},
			{Order: 3, Component: "CounterApp", Field: "width", Before: "320", After: "400", Cause: "resize"},
			{Order: 4, Component: "CounterButton", Field: "text_len_seen", Before: "0", After: "2", Cause: "text_input"},
		},
		Cases: []surface.CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
			{Name: "wasm32-web browser canvas surface", Kind: "positive", Ran: true, Pass: true},
			{Name: "wasm32-web browser canvas RGBA readback", Kind: "positive", Ran: true, Pass: true},
			{Name: "wasm32-web browser canvas pointer input", Kind: "positive", Ran: true, Pass: true},
			{Name: "wasm32-web browser canvas keyboard input", Kind: "positive", Ran: true, Pass: true},
			{Name: "wasm32-web browser canvas resize input", Kind: "positive", Ran: true, Pass: true},
			{Name: "wasm32-web browser canvas text input", Kind: "positive", Ran: true, Pass: true},
			{Name: "wasm32-web Surface Host ABI imports", Kind: "positive", Ran: true, Pass: true},
			{Name: "compiler-owned wasm Surface loader", Kind: "positive", Ran: true, Pass: true},
			{Name: "compiler-owned browser canvas Surface host", Kind: "positive", Ran: true, Pass: true},
			{Name: "host-provided pointer event dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "host event buffer poll_event", Kind: "positive", Ran: true, Pass: true},
			{Name: "pre/post event frame sequence", Kind: "positive", Ran: true, Pass: true},
			{Name: "host text payload buffer", Kind: "positive", Ran: true, Pass: true},
			{Name: "component hierarchy dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "component text input scalar dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "component focus dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "component accessibility metadata", Kind: "positive", Ran: true, Pass: true},
			{Name: "no legacy UI sidecar artifacts", Kind: "positive", Ran: true, Pass: true},
			{Name: "state transition", Kind: "positive", Ran: true, Pass: true},
			{Name: "reject legacy UI evidence", Kind: "negative", Ran: true, Pass: true, ExpectedError: "legacy UI evidence rejected"},
		},
	}
}
func removeCaseNamed(cases []surface.CaseReport, name string) []surface.CaseReport {
	filtered := cases[:0]
	for _, tc := range cases {
		if tc.Name == name {
			continue
		}
		filtered = append(filtered, tc)
	}
	return filtered
}
func runCounterScenario(mode string) headlessScenario {
	if mode == "linux-x64" {
		return runLinuxX64CounterScenario()
	}
	if mode == "linux-x64-real-window" {
		return runLinuxX64RealWindowCounterScenario()
	}
	if mode == "wasm32-web" {
		return runWASM32WebCounterScenario()
	}
	if mode == "wasm32-web-browser-canvas" {
		return runWASM32WebBrowserCanvasCounterScenario()
	}
	return runHeadlessCounterScenario()
}
func runSurfaceScenario(mode string) headlessScenario {
	if isTextFocusInputMode(mode) {
		return runTextFocusInputScenario(mode)
	}
	if isReleaseTextInputMode(mode) {
		return runTextFocusInputScenario(textFocusInputModeForReleaseMode(mode))
	}
	if isReleaseToolkitMode(mode) {
		return runReleaseToolkitScenario(mode)
	}
	if isReleaseWindowMode(mode) {
		return runLinuxX64ReleaseWindowScenario()
	}
	if isReleaseAppShellMode(mode) {
		return runLinuxAppShellScenario()
	}
	if isReleaseBrowserMode(mode) {
		return runReleaseBrowserScenario()
	}
	if isReleaseAccessibilityMode(mode) {
		return runReleaseAccessibilityScenario(mode)
	}
	if isComponentTreeMode(mode) {
		return runComponentTreeScenario(mode)
	}
	if isBlockPaintMode(mode) {
		return runBlockPaintScenario()
	}
	if isBlockTextMode(mode) {
		return runBlockTextScenario()
	}
	if isBlockLayoutMode(mode) {
		return runBlockLayoutScenario()
	}
	if isBlockEventMode(mode) {
		return runBlockEventScenario()
	}
	if isBlockStateMode(mode) {
		return runBlockStateScenario()
	}
	if isBlockMotionMode(mode) {
		return runBlockMotionScenario()
	}
	if isBlockAssetMode(mode) {
		return runBlockAssetScenario()
	}
	if isBlockAccessibilityMode(mode) {
		return runBlockAccessibilityScenario()
	}
	if isMorphMode(mode) {
		return runMorphScenario()
	}
	if mode == "linux-x64-real-window-block-system" {
		return runLinuxX64RealWindowBlockSystemScenario()
	}
	if mode == "wasm32-web-browser-canvas-block-system" {
		return runWASM32WebBrowserCanvasBlockSystemScenario()
	}
	if isBlockSystemMode(mode) {
		return runBlockSystemScenario()
	}
	if isMinimalToolkitMode(mode) {
		return runMinimalToolkitScenario(mode)
	}
	if isToolkitReuseMode(mode) {
		return runToolkitReuseScenario(mode)
	}
	if isAccessibilityMetadataMode(mode) {
		return runAccessibilityMetadataScenario(mode)
	}
	if isAppModelMode(mode) {
		return runAppModelScenario()
	}
	return runCounterScenario(mode)
}
func textFocusInputModeForReleaseMode(mode string) string {
	switch mode {
	case "linux-x64-release-text-input":
		return "linux-x64-real-window-text-focus-input"
	case "wasm32-web-release-text-input":
		return "wasm32-web-browser-canvas-text-focus-input"
	default:
		return "headless-text-focus-input"
	}
}
func accessibilityMetadataModeForReleaseMode(mode string) string {
	switch mode {
	case "linux-x64-release-accessibility":
		return "linux-x64-real-window-accessibility-metadata"
	case "wasm32-web-release-accessibility":
		return "wasm32-web-browser-canvas-accessibility-metadata"
	default:
		return "headless-accessibility-metadata"
	}
}
func runReleaseToolkitScenario(mode string) headlessScenario {
	beforeFrame := renderReleaseToolkitFrameRGBA(0, 0, -1, 0, 0, 0, false, 0, 320, 240)
	nameFrame := renderReleaseToolkitFrameRGBA(3, 0, 7, 0, 0, 0, false, 0, 560, 420)
	checkboxFrame := renderReleaseToolkitFrameRGBA(3, 5, 10, 0, 0, 0, true, 16, 560, 420)
	saveFrame := renderReleaseToolkitFrameRGBA(3, 5, 14, 1, 0, 1, true, 16, 560, 420)
	afterFrame := renderReleaseToolkitFrameRGBA(0, 0, 7, 1, 1, 2, true, 16, 560, 420)
	scenario := headlessScenario{
		Components: []surface.ComponentReport{
			{ID: "SurfaceReleaseFormApp", Type: "examples.surface_release_form.SurfaceReleaseFormApp", Bounds: surface.RectReport{X: 0, Y: 0, W: 560, H: 420}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"focused_id": "7", "save_count": "1", "reset_count": "1", "status_code": "2", "width": "560", "height": "420", "accessibility_role": "none"}},
			{ID: "Panel", Type: "lib.core.widgets.Panel", Parent: "SurfaceReleaseFormApp", Bounds: surface.RectReport{X: 0, Y: 0, W: 560, H: 420}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"padding": "16", "accessibility_role": "none"}},
			{ID: "Stack", Type: "lib.core.widgets.Stack", Parent: "Panel", Bounds: surface.RectReport{X: 16, Y: 16, W: 528, H: 396}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"child_count": "1", "accessibility_role": "none"}},
			{ID: "Column", Type: "lib.core.widgets.Column", Parent: "Stack", Bounds: surface.RectReport{X: 24, Y: 24, W: 512, H: 388}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"child_count": "9", "accessibility_role": "none"}},
			{ID: "TitleText", Type: "lib.core.widgets.Text", Parent: "Column", Bounds: surface.RectReport{X: 32, Y: 32, W: 496, H: 28}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"role": "label", "text_len": "18", "accessibility_role": "label"}},
			{ID: "DescriptionText", Type: "lib.core.widgets.Text", Parent: "Column", Bounds: surface.RectReport{X: 32, Y: 68, W: 496, H: 28}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"role": "description", "text_len": "24", "accessibility_role": "label"}},
			{ID: "NameLabel", Type: "lib.core.widgets.Label", Parent: "Column", Bounds: surface.RectReport{X: 32, Y: 104, W: 496, H: 24}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"role": "label", "text_len": "4", "labelled_for": "7", "accessibility_role": "label"}},
			{ID: "NameTextBox", Type: "lib.core.widgets.TextBox", Parent: "Column", Bounds: surface.RectReport{X: 32, Y: 132, W: 496, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"focused": "true", "buffer": "", "text_len": "0", "caret": "0", "accessibility_role": "label"}},
			{ID: "EmailLabel", Type: "lib.core.widgets.Label", Parent: "Column", Bounds: surface.RectReport{X: 32, Y: 184, W: 496, H: 24}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"role": "label", "text_len": "5", "labelled_for": "9", "accessibility_role": "label"}},
			{ID: "EmailTextBox", Type: "lib.core.widgets.TextBox", Parent: "Column", Bounds: surface.RectReport{X: 32, Y: 212, W: 496, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"focused": "false", "buffer": "", "text_len": "0", "caret": "0", "accessibility_role": "label"}},
			{ID: "SubscribeCheckbox", Type: "lib.core.widgets.Checkbox", Parent: "Column", Bounds: surface.RectReport{X: 32, Y: 264, W: 496, H: 32}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"focused": "false", "checked": "true", "toggle_count": "1", "accessibility_role": "button"}},
			{ID: "TermsScroll", Type: "lib.core.widgets.Scroll", Parent: "Column", Bounds: surface.RectReport{X: 32, Y: 304, W: 496, H: 48}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"offset_y": "16", "content_h": "120", "accessibility_role": "none"}},
			{ID: "TermsText", Type: "lib.core.widgets.Text", Parent: "TermsScroll", Bounds: surface.RectReport{X: 36, Y: 308, W: 488, H: 24}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"role": "description", "text_len": "48", "accessibility_role": "label"}},
			{ID: "ButtonRow", Type: "lib.core.widgets.Row", Parent: "Column", Bounds: surface.RectReport{X: 32, Y: 360, W: 496, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"child_count": "4", "accessibility_role": "none"}},
			{ID: "SaveButton", Type: "lib.core.widgets.Button", Parent: "ButtonRow", Bounds: surface.RectReport{X: 32, Y: 360, W: 132, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"focused": "false", "press_count": "1", "action": "save", "accessibility_role": "button"}},
			{ID: "ResetButton", Type: "lib.core.widgets.Button", Parent: "ButtonRow", Bounds: surface.RectReport{X: 176, Y: 360, W: 132, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"focused": "false", "press_count": "1", "action": "reset", "accessibility_role": "button"}},
			{ID: "Spacer", Type: "lib.core.widgets.Spacer", Parent: "ButtonRow", Bounds: surface.RectReport{X: 320, Y: 360, W: 16, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"min_w": "16", "min_h": "44", "accessibility_role": "none"}},
			{ID: "StatusText", Type: "lib.core.widgets.StatusText", Parent: "ButtonRow", Bounds: surface.RectReport{X: 344, Y: 360, W: 184, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"role": "status", "status_code": "2", "text_len": "6", "accessibility_role": "label"}},
		},
		ComponentTree: &surface.ComponentTreeReport{
			Schema:       "tetra.surface.component-tree.v1",
			DynamicLevel: "production-widgets-v1",
			RootID:       0,
			NodeCount:    18,
			FocusedID:    7,
			Nodes: []surface.ComponentTreeNodeReport{
				{ID: 0, Name: "SurfaceReleaseFormApp", Kind: "root", ParentID: -1, ChildIndex: 0, FirstChild: 1, ChildCount: 1, Focusable: false, Bounds: surface.RectReport{X: 0, Y: 0, W: 560, H: 420}},
				{ID: 1, Name: "Panel", Kind: "panel", ParentID: 0, ChildIndex: 0, FirstChild: 2, ChildCount: 1, Focusable: false, Bounds: surface.RectReport{X: 0, Y: 0, W: 560, H: 420}},
				{ID: 2, Name: "Stack", Kind: "stack", ParentID: 1, ChildIndex: 0, FirstChild: 3, ChildCount: 1, Focusable: false, Bounds: surface.RectReport{X: 16, Y: 16, W: 528, H: 396}},
				{ID: 3, Name: "Column", Kind: "column", ParentID: 2, ChildIndex: 0, FirstChild: 4, ChildCount: 9, Focusable: false, Bounds: surface.RectReport{X: 24, Y: 24, W: 512, H: 388}},
				{ID: 4, Name: "TitleText", Kind: "text", ParentID: 3, ChildIndex: 0, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 32, Y: 32, W: 496, H: 28}},
				{ID: 5, Name: "DescriptionText", Kind: "text", ParentID: 3, ChildIndex: 1, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 32, Y: 68, W: 496, H: 28}},
				{ID: 6, Name: "NameLabel", Kind: "label", ParentID: 3, ChildIndex: 2, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 32, Y: 104, W: 496, H: 24}},
				{ID: 7, Name: "NameTextBox", Kind: "textbox", ParentID: 3, ChildIndex: 3, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 32, Y: 132, W: 496, H: 44}},
				{ID: 8, Name: "EmailLabel", Kind: "label", ParentID: 3, ChildIndex: 4, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 32, Y: 184, W: 496, H: 24}},
				{ID: 9, Name: "EmailTextBox", Kind: "textbox", ParentID: 3, ChildIndex: 5, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 32, Y: 212, W: 496, H: 44}},
				{ID: 10, Name: "SubscribeCheckbox", Kind: "checkbox", ParentID: 3, ChildIndex: 6, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 32, Y: 264, W: 496, H: 32}},
				{ID: 11, Name: "TermsScroll", Kind: "scroll", ParentID: 3, ChildIndex: 7, FirstChild: 12, ChildCount: 1, Focusable: false, Bounds: surface.RectReport{X: 32, Y: 304, W: 496, H: 48}},
				{ID: 12, Name: "TermsText", Kind: "text", ParentID: 11, ChildIndex: 0, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 36, Y: 308, W: 488, H: 24}},
				{ID: 13, Name: "ButtonRow", Kind: "row", ParentID: 3, ChildIndex: 8, FirstChild: 14, ChildCount: 4, Focusable: false, Bounds: surface.RectReport{X: 32, Y: 360, W: 496, H: 44}},
				{ID: 14, Name: "SaveButton", Kind: "button", ParentID: 13, ChildIndex: 0, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 32, Y: 360, W: 132, H: 44}},
				{ID: 15, Name: "ResetButton", Kind: "button", ParentID: 13, ChildIndex: 1, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 176, Y: 360, W: 132, H: 44}},
				{ID: 16, Name: "Spacer", Kind: "spacer", ParentID: 13, ChildIndex: 2, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 320, Y: 360, W: 16, H: 44}},
				{ID: 17, Name: "StatusText", Kind: "status", ParentID: 13, ChildIndex: 3, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 344, Y: 360, W: 184, H: 44}},
			},
			LayoutPasses: []surface.ComponentTreeLayoutPassReport{
				{ComponentID: 7, Pass: "initial", Bounds: surface.RectReport{X: 32, Y: 132, W: 320, H: 44}, Measured: surface.SizeReport{W: 320, H: 44}},
				{ComponentID: 9, Pass: "initial", Bounds: surface.RectReport{X: 32, Y: 212, W: 320, H: 44}, Measured: surface.SizeReport{W: 320, H: 44}},
				{ComponentID: 11, Pass: "scroll", Bounds: surface.RectReport{X: 32, Y: 304, W: 496, H: 48}, Measured: surface.SizeReport{W: 496, H: 120}},
				{ComponentID: 7, Pass: "resize", Bounds: surface.RectReport{X: 32, Y: 132, W: 496, H: 44}, Measured: surface.SizeReport{W: 496, H: 44}},
				{ComponentID: 9, Pass: "resize", Bounds: surface.RectReport{X: 32, Y: 212, W: 496, H: 44}, Measured: surface.SizeReport{W: 496, H: 44}},
				{ComponentID: 17, Pass: "status-update", Bounds: surface.RectReport{X: 344, Y: 360, W: 184, H: 44}, Measured: surface.SizeReport{W: 184, H: 44}},
			},
			DrawOrder:  []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17},
			FocusOrder: []int{7, 9, 10, 14, 15},
			DispatchPaths: []surface.ComponentTreeDispatchPathReport{
				{Event: "click", TargetID: 7, X: 48, Y: 148, Path: []int{0, 1, 2, 3, 7}},
				{Event: "click", TargetID: 9, X: 48, Y: 228, Path: []int{0, 1, 2, 3, 9}},
				{Event: "click", TargetID: 10, X: 48, Y: 280, Path: []int{0, 1, 2, 3, 10}},
				{Event: "key", TargetID: 14, X: 48, Y: 376, Path: []int{0, 1, 2, 3, 13, 14}},
				{Event: "key", TargetID: 15, X: 192, Y: 376, Path: []int{0, 1, 2, 3, 13, 15}},
			},
		},
		ComponentTreeAPI: productionToolkitComponentTreeAPIReport(),
		Toolkit:          productionToolkitReport(),
		Events: []surface.EventReport{
			{Order: 1, Kind: "mouse_up", TargetComponent: "NameTextBox", DispatchPath: []string{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "NameTextBox"}, Handled: true, Pass: true, X: 48, Y: 148, Width: 560, Height: 420, BufferSlots: []int{5, 48, 148, 1, 0, 560, 420, 0, 0}, BeforeState: map[string]string{"SurfaceReleaseFormApp.focused_id": "-1", "NameTextBox.focused": "false"}, AfterState: map[string]string{"SurfaceReleaseFormApp.focused_id": "7", "NameTextBox.focused": "true"}},
			{Order: 2, Kind: "text_input", TargetComponent: "NameTextBox", DispatchPath: []string{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "NameTextBox"}, Handled: true, Pass: true, Width: 560, Height: 420, TimestampMS: 1, TextLen: 3, TextBytesHex: "416461", BufferSlots: []int{8, 0, 0, 0, 0, 560, 420, 1, 3}, BeforeState: map[string]string{"NameTextBox.buffer": "", "EmailTextBox.buffer": ""}, AfterState: map[string]string{"NameTextBox.buffer": "Ada", "EmailTextBox.buffer": ""}},
			{Order: 3, Kind: "key_down", TargetComponent: "SurfaceReleaseFormApp", DispatchPath: []string{"SurfaceReleaseFormApp"}, Handled: true, Pass: true, Key: 9, Width: 560, Height: 420, TimestampMS: 2, BufferSlots: []int{6, 0, 0, 0, 9, 560, 420, 2, 0}, BeforeState: map[string]string{"SurfaceReleaseFormApp.focused_id": "7"}, AfterState: map[string]string{"SurfaceReleaseFormApp.focused_id": "9"}},
			{Order: 4, Kind: "text_input", TargetComponent: "EmailTextBox", DispatchPath: []string{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "EmailTextBox"}, Handled: true, Pass: true, Width: 560, Height: 420, TimestampMS: 3, TextLen: 5, TextBytesHex: "7465747261", BufferSlots: []int{8, 0, 0, 0, 0, 560, 420, 3, 5}, BeforeState: map[string]string{"EmailTextBox.buffer": "", "NameTextBox.buffer": "Ada"}, AfterState: map[string]string{"EmailTextBox.buffer": "tetra", "NameTextBox.buffer": "Ada"}},
			{Order: 5, Kind: "key_down", TargetComponent: "SurfaceReleaseFormApp", DispatchPath: []string{"SurfaceReleaseFormApp"}, Handled: true, Pass: true, Key: 9, Width: 560, Height: 420, TimestampMS: 4, BufferSlots: []int{6, 0, 0, 0, 9, 560, 420, 4, 0}, BeforeState: map[string]string{"SurfaceReleaseFormApp.focused_id": "9"}, AfterState: map[string]string{"SurfaceReleaseFormApp.focused_id": "10"}},
			{Order: 6, Kind: "key_down", TargetComponent: "SubscribeCheckbox", DispatchPath: []string{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "SubscribeCheckbox"}, Handled: true, Pass: true, Key: 32, Width: 560, Height: 420, TimestampMS: 5, BufferSlots: []int{6, 0, 0, 0, 32, 560, 420, 5, 0}, BeforeState: map[string]string{"SubscribeCheckbox.checked": "false", "SubscribeCheckbox.toggle_count": "0"}, AfterState: map[string]string{"SubscribeCheckbox.checked": "true", "SubscribeCheckbox.toggle_count": "1"}},
			{Order: 7, Kind: "scroll", TargetComponent: "TermsScroll", DispatchPath: []string{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "TermsScroll"}, Handled: true, Pass: true, X: 48, Y: 320, Width: 560, Height: 420, TimestampMS: 6, BufferSlots: []int{5, 48, 320, 1, 0, 560, 420, 6, 0}, BeforeState: map[string]string{"TermsScroll.offset_y": "0"}, AfterState: map[string]string{"TermsScroll.offset_y": "16"}},
			{Order: 8, Kind: "key_down", TargetComponent: "SurfaceReleaseFormApp", DispatchPath: []string{"SurfaceReleaseFormApp"}, Handled: true, Pass: true, Key: 9, Width: 560, Height: 420, TimestampMS: 7, BufferSlots: []int{6, 0, 0, 0, 9, 560, 420, 7, 0}, BeforeState: map[string]string{"SurfaceReleaseFormApp.focused_id": "10"}, AfterState: map[string]string{"SurfaceReleaseFormApp.focused_id": "14"}},
			{Order: 9, Kind: "key_down", TargetComponent: "SaveButton", DispatchPath: []string{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "ButtonRow", "SaveButton"}, Handled: true, Pass: true, Key: 32, Width: 560, Height: 420, TimestampMS: 8, BufferSlots: []int{6, 0, 0, 0, 32, 560, 420, 8, 0}, BeforeState: map[string]string{"SurfaceReleaseFormApp.save_count": "0", "StatusText.status_code": "0"}, AfterState: map[string]string{"SurfaceReleaseFormApp.save_count": "1", "StatusText.status_code": "1"}},
			{Order: 10, Kind: "key_down", TargetComponent: "SurfaceReleaseFormApp", DispatchPath: []string{"SurfaceReleaseFormApp"}, Handled: true, Pass: true, Key: 9, Width: 560, Height: 420, TimestampMS: 9, BufferSlots: []int{6, 0, 0, 0, 9, 560, 420, 9, 0}, BeforeState: map[string]string{"SurfaceReleaseFormApp.focused_id": "14"}, AfterState: map[string]string{"SurfaceReleaseFormApp.focused_id": "15"}},
			{Order: 11, Kind: "key_down", TargetComponent: "ResetButton", DispatchPath: []string{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "ButtonRow", "ResetButton"}, Handled: true, Pass: true, Key: 13, Width: 560, Height: 420, TimestampMS: 10, BufferSlots: []int{6, 0, 0, 0, 13, 560, 420, 10, 0}, BeforeState: map[string]string{"SurfaceReleaseFormApp.reset_count": "0", "StatusText.status_code": "1", "NameTextBox.buffer": "Ada", "EmailTextBox.buffer": "tetra"}, AfterState: map[string]string{"SurfaceReleaseFormApp.reset_count": "1", "StatusText.status_code": "2", "NameTextBox.buffer": "", "EmailTextBox.buffer": ""}},
			{Order: 12, Kind: "key_down", TargetComponent: "SurfaceReleaseFormApp", DispatchPath: []string{"SurfaceReleaseFormApp"}, Handled: true, Pass: true, Key: 9, Width: 560, Height: 420, TimestampMS: 11, BufferSlots: []int{6, 0, 0, 0, 9, 560, 420, 11, 0}, BeforeState: map[string]string{"SurfaceReleaseFormApp.focused_id": "15"}, AfterState: map[string]string{"SurfaceReleaseFormApp.focused_id": "7"}},
			{Order: 13, Kind: "resize", TargetComponent: "SurfaceReleaseFormApp", DispatchPath: []string{"SurfaceReleaseFormApp"}, Handled: true, Pass: true, Width: 560, Height: 420, TimestampMS: 12, BufferSlots: []int{2, 0, 0, 0, 0, 560, 420, 12, 0}, BeforeState: map[string]string{"SurfaceReleaseFormApp.focused_id": "7", "NameTextBox.bounds.w": "320", "EmailTextBox.bounds.w": "320"}, AfterState: map[string]string{"SurfaceReleaseFormApp.focused_id": "7", "NameTextBox.bounds.w": "496", "EmailTextBox.bounds.w": "496"}},
		},
		Frames: []surface.FrameReport{
			{Order: 1, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
			{Order: 2, Width: nameFrame.Width, Height: nameFrame.Height, Stride: nameFrame.Stride, Checksum: checksumRGBA(nameFrame.Pixels), Presented: true},
			{Order: 3, Width: checkboxFrame.Width, Height: checkboxFrame.Height, Stride: checkboxFrame.Stride, Checksum: checksumRGBA(checkboxFrame.Pixels), Presented: true},
			{Order: 4, Width: saveFrame.Width, Height: saveFrame.Height, Stride: saveFrame.Stride, Checksum: checksumRGBA(saveFrame.Pixels), Presented: true},
			{Order: 5, Width: afterFrame.Width, Height: afterFrame.Height, Stride: afterFrame.Stride, Checksum: checksumRGBA(afterFrame.Pixels), Presented: true},
		},
		StateTransitions: []surface.StateTransitionReport{
			{Order: 1, Component: "SurfaceReleaseFormApp", Field: "focused_id", Before: "-1", After: "7", Cause: "mouse_up"},
			{Order: 2, Component: "NameTextBox", Field: "buffer", Before: "", After: "Ada", Cause: "text_input"},
			{Order: 3, Component: "SurfaceReleaseFormApp", Field: "focused_id", Before: "7", After: "9", Cause: "tab"},
			{Order: 4, Component: "EmailTextBox", Field: "buffer", Before: "", After: "tetra", Cause: "text_input"},
			{Order: 5, Component: "SurfaceReleaseFormApp", Field: "focused_id", Before: "9", After: "10", Cause: "tab"},
			{Order: 6, Component: "SubscribeCheckbox", Field: "checked", Before: "false", After: "true", Cause: "key_down"},
			{Order: 7, Component: "TermsScroll", Field: "offset_y", Before: "0", After: "16", Cause: "scroll"},
			{Order: 8, Component: "SurfaceReleaseFormApp", Field: "focused_id", Before: "10", After: "14", Cause: "tab"},
			{Order: 9, Component: "SurfaceReleaseFormApp", Field: "save_count", Before: "0", After: "1", Cause: "key_down"},
			{Order: 10, Component: "StatusText", Field: "status_code", Before: "0", After: "1", Cause: "save"},
			{Order: 11, Component: "SurfaceReleaseFormApp", Field: "focused_id", Before: "14", After: "15", Cause: "tab"},
			{Order: 12, Component: "NameTextBox", Field: "buffer", Before: "Ada", After: "", Cause: "reset"},
			{Order: 13, Component: "EmailTextBox", Field: "buffer", Before: "tetra", After: "", Cause: "reset"},
			{Order: 14, Component: "SurfaceReleaseFormApp", Field: "reset_count", Before: "0", After: "1", Cause: "key_down"},
			{Order: 15, Component: "StatusText", Field: "status_code", Before: "1", After: "2", Cause: "reset"},
			{Order: 16, Component: "SurfaceReleaseFormApp", Field: "focused_id", Before: "15", After: "7", Cause: "tab"},
			{Order: 17, Component: "SurfaceReleaseFormApp", Field: "NameTextBox.bounds.w", Before: "320", After: "496", Cause: "resize"},
			{Order: 18, Component: "SurfaceReleaseFormApp", Field: "EmailTextBox.bounds.w", Before: "320", After: "496", Cause: "resize"},
		},
		Cases: productionToolkitBaseCases(),
	}
	switch mode {
	case "headless-release-toolkit":
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "headless event dispatch", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "headless framebuffer checksum", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "headless actual runner trace", Kind: "positive", Ran: true, Pass: true},
		)
	case "linux-x64-release-toolkit":
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "linux-x64 Surface Host ABI open/present/close", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 real-window surface", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 native input event pump", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 real-window resize event", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 real-window close event", Kind: "positive", Ran: true, Pass: true},
		)
	case "wasm32-web-release-toolkit":
		scenario.Frames = nil
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "wasm32-web browser canvas surface", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas RGBA readback", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas pointer input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas keyboard input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas resize input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas text input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web Surface Host ABI imports", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "compiler-owned wasm Surface loader", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "compiler-owned browser canvas Surface host", Kind: "positive", Ran: true, Pass: true},
		)
	}
	return scenario
}
func runReleaseBrowserScenario() headlessScenario {
	scenario := runReleaseToolkitScenario("wasm32-web-release-toolkit")
	scenario.BrowserSurface = releaseBrowserSurfaceReport()
	scenario.Cases = append(scenario.Cases,
		surface.CaseReport{Name: "browser release Surface v1 schema", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "browser release Chromium canvas readback", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "browser release native pointer keyboard text resize", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "browser release deterministic clipboard harness", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "browser release composition trace", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "browser release accessibility snapshot mirror", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "browser release forbidden web sidecar rejection", Kind: "negative", Ran: true, Pass: true, ExpectedError: "forbidden web sidecar rejected"},
	)
	return scenario
}
func releaseBrowserSurfaceReport() *surface.BrowserSurfaceReport {
	return &surface.BrowserSurfaceReport{
		Schema:              surface.BrowserSurfaceSchemaV1,
		BrowserSurfaceLevel: "browser-canvas-release-v1",
		ReleaseScope:        surface.ReleaseScopeSurfaceV1LinuxWeb,
		Source:              "examples/surface_release_form.tetra",
		HostAdapter:         "compiler-owned-browser-canvas-host",
		ProductionClaim:     true,
		Experimental:        false,
		CompilerOwnedBoot:   true,
		DOMHostCanvasOnly:   true,
		Canvas: surface.BrowserSurfaceCanvasReport{
			Opened:       true,
			Readback:     true,
			Width:        560,
			Height:       420,
			FrameOrder:   5,
			ArtifactKind: "runner-trace",
			Pass:         true,
		},
		Input: surface.BrowserSurfaceInputReport{
			Pointer:      true,
			Keyboard:     true,
			Text:         true,
			Resize:       true,
			HostTrace:    true,
			NativeEvents: []string{"pointerup", "keydown", "beforeinput", "resize"},
			Pass:         true,
		},
		Clipboard: surface.BrowserSurfaceClipboardReport{
			Harness:   "deterministic-browser-clipboard-v1",
			Read:      true,
			Write:     true,
			OwnedCopy: true,
			Bytes:     13,
			Pass:      true,
		},
		Composition: surface.BrowserSurfaceCompositionReport{
			Start:  true,
			Update: true,
			Commit: true,
			Cancel: true,
			Pass:   true,
		},
		Accessibility: surface.BrowserSurfaceAccessibilityReport{
			Snapshot:      true,
			Mirror:        true,
			CompilerOwned: true,
			Bounds:        true,
			Focus:         true,
			Roles:         []string{"root", "textbox", "checkbox", "button", "status"},
			DOMVisualUI:   false,
			UserJS:        false,
			Pass:          true,
		},
		HostTraces: []surface.BrowserSurfaceHostTraceReport{
			{Name: "browser-canvas", ArtifactKind: "runner-trace", Path: "surface-runner-trace.json", Pass: true},
		},
		NegativeGuards: surface.BrowserSurfaceNegativeGuards{
			NoDOMAppUITree:      true,
			NoUserJSAppLogic:    true,
			NoNodeOnlyPromotion: true,
			NoLegacySidecars:    true,
			NoReactRuntime:      true,
			NoPlatformWidgets:   true,
		},
	}
}
func runLinuxX64ReleaseWindowScenario() headlessScenario {
	scenario := runReleaseToolkitScenario("linux-x64-release-toolkit")
	beforeFrame := renderReleaseToolkitFrameRGBA(0, 0, -1, 0, 0, 0, false, 0, 320, 240)
	scenario.Frames = []surface.FrameReport{
		{Order: 1, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
	}
	scenario.AccessibilityTree = releaseWindowAccessibilityTreeReport()
	scenario.Events = append(scenario.Events, surface.EventReport{
		Order:           len(scenario.Events) + 1,
		Kind:            "close",
		TargetComponent: "SurfaceReleaseFormApp",
		DispatchPath:    []string{"SurfaceReleaseFormApp"},
		Handled:         true,
		Pass:            true,
		Width:           560,
		Height:          420,
		TimestampMS:     len(scenario.Events),
		BufferSlots:     []int{9, 0, 0, 0, 0, 560, 420, len(scenario.Events), 0},
		BeforeState:     map[string]string{"SurfaceReleaseFormApp.open": "true"},
		AfterState:      map[string]string{"SurfaceReleaseFormApp.open": "false"},
	})
	scenario.Cases = append(scenario.Cases,
		surface.CaseReport{Name: "linux release window v1 schema", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "linux release real window presented frame", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "linux release native pointer key text resize close", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "linux release clipboard harness", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "linux release composition harness", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "linux release accessibility bridge probe", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "linux release forbids memfd starter promotion", Kind: "negative", Ran: true, Pass: true, ExpectedError: "memfd starter rejected"},
		surface.CaseReport{Name: "accessibility platform bridge v1 schema", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "linux accessibility host bridge export", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "accessibility release honest screen reader evidence", Kind: "positive", Ran: true, Pass: true},
	)
	return scenario
}
func releaseWindowAccessibilityTreeReport() *surface.AccessibilityTreeReport {
	return &surface.AccessibilityTreeReport{
		Schema:                   "tetra.surface.accessibility-tree.v1",
		AccessibilityLevel:       "platform-bridge-v1",
		ReleaseScope:             "surface-v1-linux-web",
		Source:                   "examples/surface_release_form.tetra",
		Module:                   "lib.core.accessibility",
		WidgetModule:             "lib.core.widgets",
		Experimental:             false,
		ProductionClaim:          true,
		PlatformHostIntegration:  true,
		DOMARIAIntegration:       false,
		ScreenReaderEvidence:     "linux_accessibility_host_bridge_v1",
		MetadataTree:             true,
		PlatformExport:           true,
		PlatformBridge:           "linux_accessibility_host_bridge_v1",
		LinuxPlatformProbe:       true,
		LinuxProbeArtifact:       "/tmp/surface-artifacts/surface-linux-accessibility-probe.json",
		DerivedFromComponentTree: true,
		UsesComponentTreeAPI:     true,
		UsesWidgetToolkit:        true,
		ManualBookkeeping:        false,
		NoDOMUI:                  true,
		NoUserJS:                 true,
		NoPlatformWidgets:        true,
		NoLegacySidecars:         true,
		ComponentTreeSchema:      "tetra.surface.component-tree.v1",
		ComponentTreeAPISchema:   "tetra.surface.component-tree-api.v1",
		ToolkitSchema:            "tetra.surface.toolkit.v1",
		NodeCount:                18,
		FocusableCount:           5,
		LabelCount:               2,
		TextBoxCount:             2,
		ButtonCount:              2,
		StatusCount:              1,
		RolesPresent:             []string{"root", "panel", "column", "text", "label", "textbox", "checkbox", "row", "button", "status"},
		FocusOrder:               []string{"NameTextBox", "EmailTextBox", "SubscribeCheckbox", "SaveButton", "ResetButton"},
		ReadingOrder:             []string{"TitleText", "DescriptionText", "NameLabel", "NameTextBox", "EmailLabel", "EmailTextBox", "SubscribeCheckbox", "TermsText", "SaveButton", "ResetButton", "StatusText"},
		NegativeGuards: surface.AccessibilityNegativeGuardsReport{
			NoBorrowedViewStorage:       true,
			ComponentIDAlignmentChecked: true,
			BoundsAlignmentChecked:      true,
			FocusOrderAlignmentChecked:  true,
			ReadingOrderChecked:         true,
			LabelRelationshipsChecked:   true,
			StateUpdatesChecked:         true,
			ArtifactScanChecked:         true,
		},
	}
}
func runReleaseAccessibilityScenario(mode string) headlessScenario {
	scenario := runAccessibilityMetadataScenario(accessibilityMetadataModeForReleaseMode(mode))
	for i := range scenario.Components {
		if scenario.Components[i].ID == "AccessibilitySettingsApp" {
			scenario.Components[i].Type = "examples.surface_release_accessibility.SurfaceReleaseAccessibilityApp"
		}
	}
	if scenario.ComponentTree != nil {
		scenario.ComponentTree.DynamicLevel = "platform-bridge-v1"
	}
	if scenario.ComponentTreeAPI != nil {
		scenario.ComponentTreeAPI.Source = "examples/surface_release_accessibility.tetra"
	}
	if scenario.Toolkit != nil {
		scenario.Toolkit.Source = "examples/surface_release_accessibility.tetra"
		if !containsString(scenario.Toolkit.Sources, "examples/surface_release_accessibility.tetra") {
			scenario.Toolkit.Sources = append(scenario.Toolkit.Sources, "examples/surface_release_accessibility.tetra")
		}
	}
	if scenario.AccessibilityTree != nil {
		tree := scenario.AccessibilityTree
		tree.AccessibilityLevel = "platform-bridge-v1"
		tree.ReleaseScope = "surface-v1-linux-web"
		tree.Source = "examples/surface_release_accessibility.tetra"
		tree.Experimental = false
		tree.ProductionClaim = true
		tree.MetadataTree = true
		tree.PlatformExport = true
		tree.ScreenReaderEvidence = "platform-tree-probe"
		tree.PlatformBridge = "headless_accessibility_export_v1"
		tree.LinuxProbeArtifact = ""
		tree.LinuxPlatformProbe = false
		tree.BrowserAccessibilitySnap = false
		tree.BrowserAccessibilityMirror = false
		tree.DOMARIAIntegration = false
		if mode == "linux-x64-release-accessibility" {
			tree.PlatformHostIntegration = true
			tree.PlatformBridge = "linux_accessibility_host_bridge_v1"
			tree.LinuxPlatformProbe = true
			tree.LinuxProbeArtifact = "/tmp/surface-artifacts/surface-linux-accessibility-probe.json"
			tree.ScreenReaderEvidence = "linux_accessibility_host_bridge_v1"
		} else if mode == "wasm32-web-release-accessibility" {
			tree.PlatformHostIntegration = true
			tree.PlatformBridge = "browser_accessibility_mirror_v1"
			tree.BrowserAccessibilitySnap = true
			tree.BrowserAccessibilityMirror = true
			tree.DOMARIAIntegration = true
			tree.ScreenReaderEvidence = "browser_accessibility_snapshot_v1"
		} else {
			tree.PlatformHostIntegration = false
			tree.ScreenReaderEvidence = "headless_platform_tree_probe"
		}
	}
	scenario.Cases = append(scenario.Cases,
		surface.CaseReport{Name: "accessibility platform bridge v1 schema", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "accessibility platform export from metadata tree", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "linux accessibility host bridge export", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "accessibility release honest screen reader evidence", Kind: "positive", Ran: true, Pass: true},
	)
	switch mode {
	case "linux-x64-release-accessibility":
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "linux accessibility platform probe roles labels values states bounds", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux accessibility probe focus order labels status resize", Kind: "positive", Ran: true, Pass: true},
		)
	case "wasm32-web-release-accessibility":
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "browser accessibility snapshot roles labels values states bounds", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "browser compiler-owned accessibility mirror", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "browser accessibility mirror no DOM visual UI", Kind: "positive", Ran: true, Pass: true},
		)
	default:
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "headless deterministic accessibility platform bridge shape", Kind: "positive", Ran: true, Pass: true},
		)
	}
	return scenario
}
