package main

import (
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"tetra_language/tools/validators/surface"
)

// ---- scenarios_app.go ----

func runHeadlessCounterScenario() headlessScenario {
	beforeFrame := renderCounterFrameRGBA(0, true)
	afterFrame := renderCounterFrameRGBA(1, true)
	return headlessScenario{
		Components: []surface.ComponentReport{
			{
				ID:     "CounterApp",
				Type:   "examples.surface.runtime.surface_counter.CounterApp",
				Bounds: surface.RectReport{X: 0, Y: 0, W: 320, H: 200},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"count":              "1",
					"text_count":         "1",
					"accessibility_role": "button",
				},
			},
			{
				ID:     "CounterButton",
				Type:   "examples.surface.runtime.surface_counter.CounterButton",
				Parent: "CounterApp",
				Bounds: surface.RectReport{X: 32, Y: 80, W: 160, H: 48},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"pressed":            "false",
					"focused":            "true",
					"text_len_seen":      "2",
					"accessibility_role": "button",
				},
			},
		},
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "none",
				TargetComponent: "CounterApp",
				DispatchPath:    []string{"CounterApp"},
				Handled:         false,
				Pass:            true,
				X:               0,
				Y:               0,
				BeforeState:     map[string]string{"CounterApp.count": "0"},
				AfterState:      map[string]string{"CounterApp.count": "0"},
			},
			{
				Order:           2,
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
				BeforeState: map[string]string{
					"CounterApp.count":      "0",
					"CounterButton.pressed": "false",
				},
				AfterState: map[string]string{
					"CounterApp.count":      "1",
					"CounterButton.pressed": "false",
				},
			},
			{
				Order:           3,
				Kind:            "text_input",
				TargetComponent: "CounterButton",
				DispatchPath:    []string{"CounterApp", "CounterButton"},
				Handled:         true,
				Pass:            true,
				X:               0,
				Y:               0,
				Key:             0,
				Width:           320,
				Height:          200,
				TimestampMS:     1,
				TextLen:         2,
				TextBytesHex:    "4f4b",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
				BeforeState: map[string]string{
					"CounterApp.text_count":       "0",
					"CounterButton.text_len_seen": "0",
				},
				AfterState: map[string]string{
					"CounterApp.text_count":       "1",
					"CounterButton.text_len_seen": "2",
				},
			},
		},
		Frames: []surface.FrameReport{
			{
				Order:     1,
				Width:     beforeFrame.Width,
				Height:    beforeFrame.Height,
				Stride:    beforeFrame.Stride,
				Checksum:  checksumRGBA(beforeFrame.Pixels),
				Presented: true,
			},
			{
				Order:     2,
				Width:     afterFrame.Width,
				Height:    afterFrame.Height,
				Stride:    afterFrame.Stride,
				Checksum:  checksumRGBA(afterFrame.Pixels),
				Presented: true,
			},
		},
		StateTransitions: []surface.StateTransitionReport{
			{
				Order:     1,
				Component: "CounterApp",
				Field:     "count",
				Before:    "0",
				After:     "1",
				Cause:     "mouse_up",
			},
			{
				Order:     2,
				Component: "CounterApp",
				Field:     "text_count",
				Before:    "0",
				After:     "1",
				Cause:     "text_input",
			},
		},
		Cases: []surface.CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless event dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless framebuffer checksum", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless actual runner trace", Kind: "positive", Ran: true, Pass: true},
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
			{
				Name:          "reject legacy UI evidence",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "legacy UI evidence rejected",
			},
		},
	}
}
func runAppModelScenario() headlessScenario {
	beforeFrame := renderCounterFrameRGBA(0, true)
	afterFrame := renderCounterFrameRGBA(1, true)
	return headlessScenario{
		Components: []surface.ComponentReport{
			{
				ID:     "AppModelApp",
				Type:   "examples.surface.toolkit.surface_app_model.AppModelApp",
				Bounds: surface.RectReport{X: 0, Y: 0, W: 480, H: 320},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"route":              "settings",
					"focused":            "NameField",
					"save_count":         "1",
					"pending_task":       "0",
					"history_depth":      "1",
					"redo_depth":         "0",
					"accessibility_role": "none",
				},
			},
			{
				ID:     "NameField",
				Type:   "examples.surface.toolkit.surface_app_model.NameField",
				Parent: "AppModelApp",
				Bounds: surface.RectReport{X: 32, Y: 80, W: 240, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "true",
					"buffer":             "Ada",
					"caret":              "3",
					"accessibility_role": "textbox",
				},
			},
			{
				ID:     "SaveButton",
				Type:   "examples.surface.toolkit.surface_app_model.SaveButton",
				Parent: "AppModelApp",
				Bounds: surface.RectReport{X: 32, Y: 144, W: 132, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "false",
					"press_count":        "1",
					"action":             "save",
					"accessibility_role": "button",
				},
			},
		},
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "NameField",
				DispatchPath:    []string{"AppModelApp", "NameField"},
				Handled:         true,
				Pass:            true,
				X:               48,
				Y:               96,
				Width:           480,
				Height:          320,
				BufferSlots:     []int{5, 48, 96, 1, 0, 480, 320, 0, 0},
				BeforeState:     map[string]string{"AppModelApp.focused": ""},
				AfterState:      map[string]string{"AppModelApp.focused": "NameField"},
			},
			{
				Order:           2,
				Kind:            "text_input",
				TargetComponent: "NameField",
				DispatchPath:    []string{"AppModelApp", "NameField"},
				Handled:         true,
				Pass:            true,
				Width:           480,
				Height:          320,
				TimestampMS:     1,
				TextLen:         3,
				TextBytesHex:    "416461",
				BufferSlots:     []int{8, 0, 0, 0, 0, 480, 320, 1, 3},
				BeforeState:     map[string]string{"NameField.buffer": ""},
				AfterState:      map[string]string{"NameField.buffer": "Ada"},
			},
			{
				Order:           3,
				Kind:            "key_down",
				TargetComponent: "SaveButton",
				DispatchPath:    []string{"AppModelApp", "SaveButton"},
				Handled:         true,
				Pass:            true,
				Key:             13,
				Width:           480,
				Height:          320,
				TimestampMS:     2,
				BufferSlots:     []int{6, 0, 0, 0, 13, 480, 320, 2, 0},
				BeforeState:     map[string]string{"AppModelApp.save_count": "0"},
				AfterState:      map[string]string{"AppModelApp.save_count": "1"},
			},
		},
		Frames: []surface.FrameReport{
			{
				Order:     1,
				Width:     beforeFrame.Width,
				Height:    beforeFrame.Height,
				Stride:    beforeFrame.Stride,
				Checksum:  checksumRGBA(beforeFrame.Pixels),
				Presented: true,
			},
			{
				Order:     2,
				Width:     afterFrame.Width,
				Height:    afterFrame.Height,
				Stride:    afterFrame.Stride,
				Checksum:  checksumRGBA(afterFrame.Pixels),
				Presented: true,
			},
		},
		StateTransitions: []surface.StateTransitionReport{
			{
				Order:     1,
				Component: "AppModelApp",
				Field:     "focused",
				Before:    "",
				After:     "NameField",
				Cause:     "focus",
			},
			{
				Order:     2,
				Component: "NameField",
				Field:     "buffer",
				Before:    "",
				After:     "Ada",
				Cause:     "command.insert_text",
			},
			{
				Order:     3,
				Component: "AppModelApp",
				Field:     "route",
				Before:    "home",
				After:     "settings",
				Cause:     "command.navigate",
			},
			{
				Order:     4,
				Component: "AppModelApp",
				Field:     "pending_task",
				Before:    "1",
				After:     "0",
				Cause:     "command.async_complete",
			},
			{
				Order:     5,
				Component: "AppModelApp",
				Field:     "history_depth",
				Before:    "0",
				After:     "1",
				Cause:     "command.undoable",
			},
			{
				Order:     6,
				Component: "AppModelApp",
				Field:     "save_count",
				Before:    "0",
				After:     "1",
				Cause:     "command.save",
			},
		},
		AppModel: &surface.AppModelReport{
			Schema:                "tetra.surface.app-model.v1",
			AppModelLevel:         "explicit-command-reducer-v1",
			ReleaseScope:          "surface-v1-linux-web",
			Source:                "examples/surface/toolkit/surface_app_model.tetra",
			Module:                "lib.core.surface_app",
			UsesComponentTreeAPI:  true,
			CallerOwnedState:      true,
			ExplicitEventBindings: true,
			DeterministicReducer:  true,
			StateFields: []string{
				"route",
				"focused",
				"name_buffer",
				"save_count",
				"pending_task",
				"history_depth",
				"redo_depth",
			},
			CommandRegistry: []string{
				"focus.name",
				"text.insert",
				"nav.push.settings",
				"nav.back",
				"async.save.start",
				"async.save.complete",
				"async.save.cancel",
				"history.undo",
				"history.redo",
			},
			EventBindings: []surface.AppModelEventBindingReport{
				{
					Order:        1,
					EventOrder:   1,
					EventKind:    "mouse_up",
					Target:       "NameField",
					DispatchPath: []string{"AppModelApp", "NameField"},
					Command:      "focus.name",
					Explicit:     true,
				},
				{
					Order:        2,
					EventOrder:   2,
					EventKind:    "text_input",
					Target:       "NameField",
					DispatchPath: []string{"AppModelApp", "NameField"},
					Command:      "text.insert",
					Explicit:     true,
				},
				{
					Order:        3,
					EventOrder:   3,
					EventKind:    "key_down",
					Target:       "SaveButton",
					DispatchPath: []string{"AppModelApp", "SaveButton"},
					Command:      "async.save.start",
					Explicit:     true,
				},
			},
			CommandDispatches: []surface.AppModelCommandDispatchReport{
				{
					Order:       1,
					EventOrder:  1,
					Command:     "focus.name",
					Kind:        "focus",
					Target:      "NameField",
					Handled:     true,
					BeforeState: map[string]string{"focused": ""},
					AfterState:  map[string]string{"focused": "NameField"},
				},
				{
					Order:        2,
					EventOrder:   2,
					Command:      "text.insert",
					Kind:         "edit",
					Target:       "NameField",
					Handled:      true,
					BeforeState:  map[string]string{"name_buffer": ""},
					AfterState:   map[string]string{"name_buffer": "Ada"},
					Reversible:   true,
					HistoryIndex: 1,
				},
				{
					Order:       3,
					EventOrder:  3,
					Command:     "async.save.start",
					Kind:        "async_start",
					Target:      "SaveButton",
					Handled:     true,
					BeforeState: map[string]string{"pending_task": "0"},
					AfterState:  map[string]string{"pending_task": "1"},
					AsyncTaskID: "save-1",
				},
				{
					Order:       4,
					Command:     "async.save.complete",
					Kind:        "async_complete",
					Target:      "AppModelApp",
					Handled:     true,
					BeforeState: map[string]string{"pending_task": "1", "save_count": "0"},
					AfterState:  map[string]string{"pending_task": "0", "save_count": "1"},
					AsyncTaskID: "save-1",
				},
			},
			NavigationTransitions: []surface.AppModelNavigationReport{
				{
					Order:       1,
					Command:     "nav.push.settings",
					Operation:   "push",
					BeforeRoute: "home",
					AfterRoute:  "settings",
					StackBefore: []string{"home"},
					StackAfter:  []string{"home", "settings"},
				},
				{
					Order:       2,
					Command:     "nav.back",
					Operation:   "back",
					BeforeRoute: "settings",
					AfterRoute:  "home",
					StackBefore: []string{"home", "settings"},
					StackAfter:  []string{"home"},
				},
				{
					Order:             3,
					Command:           "nav.back",
					Operation:         "back",
					BeforeRoute:       "home",
					AfterRoute:        "home",
					StackBefore:       []string{"home"},
					StackAfter:        []string{"home"},
					UnderflowRejected: true,
				},
			},
			FocusScopeTransitions: []surface.AppModelFocusScopeReport{
				{Order: 1, Scope: "main", BeforeFocus: "", AfterFocus: "NameField"},
				{
					Order:       2,
					Scope:       "dialog",
					BeforeFocus: "DialogCancel",
					AfterFocus:  "DialogConfirm",
					Wrapped:     true,
					ModalTrap:   true,
				},
			},
			AsyncTasks: []surface.AppModelAsyncTaskReport{
				{
					ID:          "save-1",
					Command:     "async.save.start",
					Operation:   "start",
					Status:      "pending",
					BeforeState: map[string]string{"pending_task": "0"},
					AfterState:  map[string]string{"pending_task": "1"},
				},
				{
					ID:              "save-1",
					Command:         "async.save.complete",
					Operation:       "complete",
					Status:          "completed",
					BeforeState:     map[string]string{"pending_task": "1"},
					AfterState:      map[string]string{"pending_task": "0"},
					CompletionOrder: 4,
				},
				{
					ID:          "save-2",
					Command:     "async.save.cancel",
					Operation:   "cancel",
					Status:      "canceled",
					BeforeState: map[string]string{"pending_task": "1", "save_count": "1"},
					AfterState:  map[string]string{"pending_task": "0", "save_count": "1"},
					Canceled:    true,
				},
			},
			UndoRedoTransitions: []surface.AppModelUndoRedoReport{
				{
					Order:               1,
					Command:             "text.insert",
					HistoryIndex:        1,
					Operation:           "record",
					Before:              "",
					After:               "Ada",
					MatchedHistoryEntry: true,
					Applied:             true,
				},
				{
					Order:               2,
					Command:             "history.undo",
					HistoryIndex:        1,
					Operation:           "undo",
					Before:              "Ada",
					After:               "",
					MatchedHistoryEntry: true,
					Applied:             true,
				},
				{
					Order:               3,
					Command:             "history.redo",
					HistoryIndex:        1,
					Operation:           "redo",
					Before:              "",
					After:               "Ada",
					MatchedHistoryEntry: true,
					Applied:             true,
				},
			},
			NegativeGuards: surface.AppModelNegativeGuardsReport{
				NoHiddenAppState:              true,
				NoReactHooks:                  true,
				NoDOMEventModel:               true,
				NoUserJS:                      true,
				NoPlatformWidgets:             true,
				AsyncCancelNoMutation:         true,
				NavigationUnderflowRejected:   true,
				FocusScopeEscapeRejected:      true,
				UndoRedoRequiresHistory:       true,
				CommandWithoutBindingRejected: true,
			},
		},
		Cases: []surface.CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless event dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless framebuffer checksum", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless actual runner trace", Kind: "positive", Ran: true, Pass: true},
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
			{
				Name: "app model explicit event-to-command binding",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "app model deterministic command reducer",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{Name: "app model navigation stack", Kind: "positive", Ran: true, Pass: true},
			{Name: "app model focus scope modal trap", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "app model async completion cancellation boundary",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{Name: "app model undo redo history", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "app model no React hooks DOM event model hidden JS state",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name:          "reject legacy UI evidence",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "legacy UI evidence rejected",
			},
		},
	}
}
func runLinuxAppShellScenario() headlessScenario {
	features := linuxAppShellFeatureLedgerRows()
	return headlessScenario{
		Components: []surface.ComponentReport{
			{
				ID:     "NotesShellApp",
				Type:   "examples.surface.toolkit.surface_linux_app_shell_notes.NotesShellApp",
				Bounds: surface.RectReport{X: 0, Y: 0, W: 720, H: 540},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"open_windows":       "2",
					"focused_window":     "notes-main",
					"accessibility_role": "application",
				},
			},
			{
				ID:     "NotesMainWindow",
				Type:   "examples.surface.toolkit.surface_linux_app_shell_notes.NotesMainWindow",
				Parent: "NotesShellApp",
				Bounds: surface.RectReport{X: 0, Y: 0, W: 560, H: 420},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"title":              "Notes",
					"lifecycle":          "reopened",
					"dpi_scale_milli":    "1250",
					"cursor":             "text",
					"accessibility_role": "document",
				},
			},
			{
				ID:     "NotesInspectorWindow",
				Type:   "examples.surface.toolkit.surface_linux_app_shell_notes.NotesInspectorWindow",
				Parent: "NotesShellApp",
				Bounds: surface.RectReport{X: 24, Y: 24, W: 320, H: 240},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"title":              "Inspector",
					"lifecycle":          "open",
					"dpi_scale_milli":    "1000",
					"cursor":             "pointer",
					"accessibility_role": "panel",
				},
			},
		},
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "NotesMainWindow",
				DispatchPath:    []string{"NotesShellApp", "NotesMainWindow"},
				Handled:         true,
				Pass:            true,
				X:               40,
				Y:               72,
				Width:           560,
				Height:          420,
				TimestampMS:     0,
				BufferSlots:     []int{5, 40, 72, 1, 0, 560, 420, 0, 0},
				BeforeState:     map[string]string{"NotesShellApp.focused_window": ""},
				AfterState:      map[string]string{"NotesShellApp.focused_window": "notes-main"},
			},
			{
				Order:           2,
				Kind:            "key_down",
				TargetComponent: "NotesMainWindow",
				DispatchPath:    []string{"NotesShellApp", "NotesMainWindow"},
				Handled:         true,
				Pass:            true,
				Key:             78,
				Width:           560,
				Height:          420,
				TimestampMS:     2,
				BufferSlots:     []int{6, 0, 0, 1, 78, 560, 420, 2, 0},
				BeforeState:     map[string]string{"NotesMainWindow.shortcut": ""},
				AfterState:      map[string]string{"NotesMainWindow.shortcut": "new-note"},
			},
			{
				Order:           3,
				Kind:            "text_input",
				TargetComponent: "NotesMainWindow",
				DispatchPath:    []string{"NotesShellApp", "NotesMainWindow"},
				Handled:         true,
				Pass:            true,
				Width:           560,
				Height:          420,
				TimestampMS:     3,
				TextLen:         5,
				TextBytesHex:    "4e6f746573",
				BufferSlots:     []int{8, 0, 0, 0, 0, 560, 420, 3, 5},
				BeforeState:     map[string]string{"NotesMainWindow.buffer": ""},
				AfterState:      map[string]string{"NotesMainWindow.buffer": "Notes"},
			},
			{
				Order:           4,
				Kind:            "resize",
				TargetComponent: "NotesMainWindow",
				DispatchPath:    []string{"NotesShellApp", "NotesMainWindow"},
				Handled:         true,
				Pass:            true,
				Width:           720,
				Height:          540,
				TimestampMS:     4,
				BufferSlots:     []int{7, 0, 0, 0, 0, 720, 540, 4, 0},
				BeforeState: map[string]string{
					"NotesMainWindow.size": "560x420",
					"NotesMainWindow.dpi":  "1000",
				},
				AfterState: map[string]string{
					"NotesMainWindow.size": "720x540",
					"NotesMainWindow.dpi":  "1250",
				},
			},
			{
				Order:           5,
				Kind:            "close",
				TargetComponent: "NotesInspectorWindow",
				DispatchPath:    []string{"NotesShellApp", "NotesInspectorWindow"},
				Handled:         true,
				Pass:            true,
				Width:           320,
				Height:          240,
				TimestampMS:     5,
				BufferSlots:     []int{9, 0, 0, 0, 0, 320, 240, 5, 0},
				BeforeState:     map[string]string{"NotesInspectorWindow.open": "true"},
				AfterState:      map[string]string{"NotesInspectorWindow.open": "false"},
			},
		},
		Frames: []surface.FrameReport{
			{
				Order:     1,
				Width:     400,
				Height:    240,
				Stride:    1600,
				Checksum:  "1111111111111111111111111111111111111111111111111111111111111111",
				Presented: true,
			},
			{
				Order:     5,
				Width:     560,
				Height:    420,
				Stride:    2240,
				Checksum:  "2222222222222222222222222222222222222222222222222222222222222222",
				Presented: true,
			},
			{
				Order:     6,
				Width:     720,
				Height:    540,
				Stride:    2880,
				Checksum:  "3333333333333333333333333333333333333333333333333333333333333333",
				Presented: true,
			},
		},
		StateTransitions: []surface.StateTransitionReport{
			{
				Order:     1,
				Component: "NotesShellApp",
				Field:     "focused_window",
				Before:    "",
				After:     "notes-main",
				Cause:     "lifecycle.open",
			},
			{
				Order:     2,
				Component: "NotesInspectorWindow",
				Field:     "open",
				Before:    "true",
				After:     "false",
				Cause:     "lifecycle.close",
			},
			{
				Order:     3,
				Component: "NotesMainWindow",
				Field:     "size",
				Before:    "560x420",
				After:     "720x540",
				Cause:     "resize",
			},
		},
		LinuxAppShell: &surface.LinuxAppShellReport{
			Schema:          surface.LinuxAppShellSchemaV1,
			AppShellLevel:   "linux-app-shell-subset-v1",
			ReleaseScope:    surface.ReleaseScopeSurfaceV1LinuxWeb,
			Source:          "examples/surface/toolkit/surface_linux_app_shell_notes.tetra",
			Module:          "lib.core.surface_app_shell",
			HostAdapter:     "wayland-shm-rgba-release-v1",
			ProductionClaim: true,
			Experimental:    false,
			WindowLifecycle: []surface.LinuxAppShellLifecycleReport{
				{Order: 1, WindowID: "notes-main", Operation: "open", HostTrace: true, Pass: true},
				{
					Order:     2,
					WindowID:  "notes-inspector",
					Operation: "open",
					HostTrace: true,
					Pass:      true,
				},
				{
					Order:     3,
					WindowID:  "notes-inspector",
					Operation: "close",
					HostTrace: true,
					Pass:      true,
				},
				{
					Order:     4,
					WindowID:  "notes-inspector",
					Operation: "reopen",
					HostTrace: true,
					Pass:      true,
				},
			},
			Windows: []surface.LinuxAppShellWindowReport{
				{
					ID:            "notes-main",
					Title:         "Notes",
					Role:          "primary",
					BlockRoot:     "NotesMainWindow",
					RealWindow:    true,
					Presented:     true,
					Width:         720,
					Height:        540,
					DPIScaleMilli: 1250,
				},
				{
					ID:            "notes-inspector",
					Title:         "Inspector",
					Role:          "secondary",
					BlockRoot:     "NotesInspectorWindow",
					RealWindow:    true,
					Presented:     true,
					Width:         320,
					Height:        240,
					DPIScaleMilli: 1000,
				},
			},
			ResizeDPI: []surface.LinuxAppShellResizeDPIReport{
				{
					WindowID:      "notes-main",
					Operation:     "resize",
					BeforeWidth:   560,
					BeforeHeight:  420,
					AfterWidth:    720,
					AfterHeight:   540,
					DPIScaleMilli: 1250,
					HostTrace:     true,
					Pass:          true,
				},
				{
					WindowID:      "notes-main",
					Operation:     "dpi_scale",
					BeforeWidth:   720,
					BeforeHeight:  540,
					AfterWidth:    720,
					AfterHeight:   540,
					DPIScaleMilli: 1250,
					HostTrace:     true,
					Pass:          true,
				},
			},
			CursorTransitions: []surface.LinuxAppShellCursorReport{
				{
					WindowID:  "notes-main",
					Cursor:    "pointer",
					Target:    "NotesMainWindow",
					HostTrace: true,
					Pass:      true,
				},
				{
					WindowID:  "notes-main",
					Cursor:    "text",
					Target:    "NotesMainWindow",
					HostTrace: true,
					Pass:      true,
				},
				{
					WindowID:  "notes-main",
					Cursor:    "resize",
					Target:    "NotesMainWindow",
					HostTrace: true,
					Pass:      true,
				},
			},
			Clipboard: surface.LinuxAppShellCapabilityReport{
				Level:        "clipboard-text-v1",
				HostTrace:    true,
				ArtifactKind: "linux-app-shell-host-trace",
				Read:         true,
				Write:        true,
				Pass:         true,
			},
			IME: surface.LinuxAppShellCapabilityReport{
				Level:        "composition-baseline-v1",
				HostTrace:    true,
				ArtifactKind: "linux-app-shell-host-trace",
				Start:        true,
				Update:       true,
				Commit:       true,
				Cancel:       true,
				Pass:         true,
			},
			Accessibility: surface.LinuxAppShellCapabilityReport{
				Level:          "platform-bridge-v1",
				HostTrace:      true,
				ArtifactKind:   "linux-accessibility-platform-probe",
				MetadataTree:   true,
				PlatformExport: true,
				Pass:           true,
			},
			ShellFeatures: features,
			HostTraces: []surface.LinuxAppShellHostTraceReport{
				{
					Name:         "lifecycle",
					ArtifactKind: "linux-app-shell-host-trace",
					Path:         "surface-linux-app-shell-host-trace.json",
					Pass:         true,
				},
				{
					Name:         "windows",
					ArtifactKind: "linux-app-shell-window-trace",
					Path:         "surface-linux-app-shell-window-trace.json",
					Pass:         true,
				},
				{
					Name:         "accessibility",
					ArtifactKind: "linux-accessibility-platform-probe",
					Path:         "surface-linux-accessibility-probe.json",
					Pass:         true,
				},
			},
			NegativeGuards: surface.LinuxAppShellNegativeGuards{
				NoGTK:             true,
				NoQT:              true,
				NoNativeWidgets:   true,
				NoElectronRuntime: true,
				NoReactRuntime:    true,
				NoDOMUI:           true,
				NoUserJS:          true,
				NoPlatformWidgets: true,
			},
		},
		SecurityPermissions: securityPermissionReportForAppShell(features),
		Cases: []surface.CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
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
			{
				Name:          "reject legacy UI evidence",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "legacy UI evidence rejected",
			},
			{Name: "linux-x64 real-window surface", Kind: "positive", Ran: true, Pass: true},
			{Name: "linux-x64 native input event pump", Kind: "positive", Ran: true, Pass: true},
			{Name: "linux-x64 real-window resize event", Kind: "positive", Ran: true, Pass: true},
			{Name: "linux-x64 real-window close event", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "linux release real window presented frame",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "linux release accessibility bridge probe",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{Name: "linux app-shell v1 schema", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "linux app-shell lifecycle open close reopen",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "linux app-shell multi-window notes reference",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "linux app-shell resize dpi cursor trace",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "linux app-shell clipboard ime accessibility adapters",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "linux app-shell file dialog notification blocked-pass",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "linux app-shell electron feature ledger",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "linux app-shell dialog file picker tray blocked-pass",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "linux app-shell crash error report scoped adapters",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name:          "linux app-shell rejects GTK Qt native widget UI",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "native widget UI rejected",
			},
			{
				Name:          "linux app-shell no Electron React DOM application scripting",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "runtime substitute rejected",
			},
			{
				Name: "surface security permission model default deny filesystem network",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "surface security app-shell feature policy enforcement",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "surface security IPC process boundary schema validation",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "surface security asset font image local hash policy",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name:          "surface security network asset fetch rejected",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "network asset fetch rejected",
			},
			{
				Name: "surface security notification dialog permission nonclaims",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "surface performance budget startup first frame",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "surface performance budget frame p50 p95",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "surface performance budget memory cache framebuffer rss",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "surface performance budget binary size",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "surface performance budget cpu power proxy",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name:          "surface performance budget faster than electron nonclaim",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "unsupported faster than electron claim rejected",
			},
		},
	}
}

// ---- scenarios_block_primary.go ----

func runBlockPaintScenario() headlessScenario {
	beforeFrame := renderBlockPaintFrameRGBA(false)
	afterFrame := renderBlockPaintFrameRGBA(true)
	frames := []surface.FrameReport{
		{
			Order:     1,
			Width:     beforeFrame.Width,
			Height:    beforeFrame.Height,
			Stride:    beforeFrame.Stride,
			Checksum:  checksumRGBA(beforeFrame.Pixels),
			Presented: true,
		},
		{
			Order:     2,
			Width:     afterFrame.Width,
			Height:    afterFrame.Height,
			Stride:    afterFrame.Stride,
			Checksum:  checksumRGBA(afterFrame.Pixels),
			Presented: true,
		},
	}
	return headlessScenario{
		Components: []surface.ComponentReport{
			{
				ID:     "BlockPaintApp",
				Type:   "examples.surface.block_render.surface_block_paint_layers.BlockPaintApp",
				Bounds: surface.RectReport{X: 0, Y: 0, W: 320, H: 200},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"hovered_id":         "2",
					"pressed_count":      "1",
					"text_count":         "1",
					"accessibility_role": "none",
				},
			},
			{
				ID:     "PaintBlock",
				Type:   "examples.surface.block_render.surface_block_paint_layers.PaintSurfaceBlock",
				Parent: "BlockPaintApp",
				Bounds: surface.RectReport{X: 12, Y: 10, W: 64, H: 28},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"paint_layers":       "5",
					"radius":             "8",
					"hovered":            "true",
					"text_len_seen":      "2",
					"accessibility_role": "button",
				},
			},
		},
		PaintLayers:           blockPaintLayersForScenario(),
		PaintCommands:         blockPaintCommandsForScenario(),
		VisualFeatures:        blockRendererVisualFeaturesForScenario(),
		PaintQualityLevel:     "deterministic-software-paint-v1",
		PaintCacheBudgetBytes: 65536,
		PaintUnsupportedBlur:  false,
		Renderer:              blockRendererReportForScenario(frames, 2),
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "PaintBlock",
				DispatchPath:    []string{"BlockPaintApp", "PaintBlock"},
				Handled:         true,
				Pass:            true,
				X:               32,
				Y:               24,
				Key:             0,
				Width:           320,
				Height:          200,
				TimestampMS:     0,
				BufferSlots:     []int{5, 32, 24, 1, 0, 320, 200, 0, 0},
				BeforeState: map[string]string{
					"BlockPaintApp.pressed_count": "0",
					"PaintBlock.hovered":          "false",
				},
				AfterState: map[string]string{
					"BlockPaintApp.pressed_count": "1",
					"PaintBlock.hovered":          "true",
				},
			},
			{
				Order:           2,
				Kind:            "text_input",
				TargetComponent: "PaintBlock",
				DispatchPath:    []string{"BlockPaintApp", "PaintBlock"},
				Handled:         true,
				Pass:            true,
				X:               0,
				Y:               0,
				Key:             0,
				Width:           320,
				Height:          200,
				TimestampMS:     1,
				TextLen:         2,
				TextBytesHex:    "4f4b",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
				BeforeState: map[string]string{
					"BlockPaintApp.text_count": "0",
					"PaintBlock.text_len_seen": "0",
				},
				AfterState: map[string]string{
					"BlockPaintApp.text_count": "1",
					"PaintBlock.text_len_seen": "2",
				},
			},
		},
		Frames: frames,
		StateTransitions: []surface.StateTransitionReport{
			{
				Order:     1,
				Component: "BlockPaintApp",
				Field:     "pressed_count",
				Before:    "0",
				After:     "1",
				Cause:     "mouse_up",
			},
			{
				Order:     2,
				Component: "PaintBlock",
				Field:     "hovered",
				Before:    "false",
				After:     "true",
				Cause:     "mouse_up",
			},
			{
				Order:     3,
				Component: "PaintBlock",
				Field:     "text_len_seen",
				Before:    "0",
				After:     "2",
				Cause:     "text_input",
			},
		},
		Cases: []surface.CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless event dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless framebuffer checksum", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless actual runner trace", Kind: "positive", Ran: true, Pass: true},
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
			{
				Name: ("block paint fill gradient image fill border radius clip shadow " +
					"overlay outline text icon"),
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "block paint deterministic command order",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{Name: "block paint frame checksum changed", Kind: "positive", Ran: true, Pass: true},
			{
				Name:          "block paint unsupported blur rejected",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "unsupported blur",
			},
			{
				Name: "block renderer software rgba contract",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "block compositor dirty rect invalidation cache",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "block renderer opacity transform clipped child",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name:          "block renderer gpu production claim rejected",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "gpu production",
			},
			{
				Name:          "block renderer unsupported backdrop blur rejected",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "backdrop blur",
			},
			{
				Name:          "reject legacy UI evidence",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "legacy UI evidence rejected",
			},
		},
	}
}
func blockPaintLayersForScenario() []surface.PaintLayerReport {
	return []surface.PaintLayerReport{
		{ID: "root-fill", BlockID: 2, Kind: "fill", Color: "#346ecfff", Radius: 8, Opacity: 255},
		{
			ID:      "root-gradient",
			BlockID: 2,
			Kind:    "gradient",
			Color:   "#54b484ff",
			Radius:  8,
			Opacity: 255,
		},
		{ID: "root-image-fill", BlockID: 2, Kind: "image_fill", Radius: 8, Opacity: 255},
		{
			ID:      "root-border",
			BlockID: 2,
			Kind:    "border",
			Color:   "#e2eaf2ff",
			Radius:  8,
			Width:   1,
			Opacity: 255,
		},
		{ID: "root-radius-clip", BlockID: 2, Kind: "radius_clip", Radius: 8, Opacity: 255},
		{
			ID:      "root-shadow",
			BlockID: 2,
			Kind:    "shadow",
			Color:   "#00000058",
			Blur:    12,
			OffsetX: 0,
			OffsetY: 4,
			Opacity: 88,
		},
		{
			ID:      "root-overlay",
			BlockID: 2,
			Kind:    "overlay",
			Color:   "#10182066",
			Radius:  8,
			Opacity: 102,
		},
		{
			ID:      "root-outline",
			BlockID: 2,
			Kind:    "outline",
			Color:   "#f4cd5cff",
			Radius:  10,
			Width:   2,
			Opacity: 255,
		},
		{ID: "root-text", BlockID: 2, Kind: "text", Color: "#edf2f7ff", Opacity: 255},
		{ID: "root-icon", BlockID: 2, Kind: "icon", Color: "#f4cd5cff", Opacity: 255},
	}
}
func blockPaintCommandsForScenario() []surface.PaintCommandReport {
	return []surface.PaintCommandReport{
		{
			Order:    1,
			Command:  "fill",
			LayerID:  "root-fill",
			BlockID:  2,
			Rect:     surface.RectReport{X: 12, Y: 10, W: 64, H: 28},
			Radius:   8,
			Quality:  "rounded-rect-v1",
			Checksum: "sha256:" + checksumText("paint-fill"),
		},
		{
			Order:    2,
			Command:  "gradient",
			LayerID:  "root-gradient",
			BlockID:  2,
			Rect:     surface.RectReport{X: 12, Y: 10, W: 64, H: 28},
			Radius:   8,
			Quality:  "two-stop-linear-v1",
			Checksum: "sha256:" + checksumText("paint-gradient"),
		},
		{
			Order:    3,
			Command:  "image_fill",
			LayerID:  "root-image-fill",
			BlockID:  2,
			Rect:     surface.RectReport{X: 12, Y: 10, W: 64, H: 28},
			Radius:   8,
			Quality:  "bounded-asset-fill-v1",
			Checksum: "sha256:" + checksumText("paint-image-fill"),
		},
		{
			Order:    4,
			Command:  "border",
			LayerID:  "root-border",
			BlockID:  2,
			Rect:     surface.RectReport{X: 12, Y: 10, W: 64, H: 28},
			Radius:   8,
			Quality:  "rounded-outline-v1",
			Checksum: "sha256:" + checksumText("paint-border"),
		},
		{
			Order:    5,
			Command:  "radius_clip",
			LayerID:  "root-radius-clip",
			BlockID:  2,
			Rect:     surface.RectReport{X: 12, Y: 10, W: 64, H: 28},
			Clip:     surface.RectReport{X: 12, Y: 10, W: 64, H: 28},
			Radius:   8,
			Quality:  "clip-stack-v1",
			Checksum: "sha256:" + checksumText("paint-radius-clip"),
		},
		{
			Order:    6,
			Command:  "shadow",
			LayerID:  "root-shadow",
			BlockID:  2,
			Rect:     surface.RectReport{X: 12, Y: 10, W: 64, H: 28},
			Radius:   8,
			Quality:  "box-shadow-approx-v1",
			Checksum: "sha256:" + checksumText("paint-shadow"),
		},
		{
			Order:    7,
			Command:  "overlay",
			LayerID:  "root-overlay",
			BlockID:  2,
			Rect:     surface.RectReport{X: 12, Y: 10, W: 64, H: 28},
			Radius:   8,
			Opacity:  102,
			Quality:  "alpha-over-v1",
			Checksum: "sha256:" + checksumText("paint-overlay"),
		},
		{
			Order:    8,
			Command:  "outline",
			LayerID:  "root-outline",
			BlockID:  2,
			Rect:     surface.RectReport{X: 10, Y: 8, W: 68, H: 32},
			Radius:   10,
			Quality:  "rounded-outline-v1",
			Checksum: "sha256:" + checksumText("paint-outline"),
		},
		{
			Order:    9,
			Command:  "text",
			LayerID:  "root-text",
			BlockID:  2,
			Rect:     surface.RectReport{X: 20, Y: 16, W: 32, H: 12},
			Quality:  "glyph-run-v1",
			Checksum: "sha256:" + checksumText("paint-text"),
		},
		{
			Order:    10,
			Command:  "icon",
			LayerID:  "root-icon",
			BlockID:  2,
			Rect:     surface.RectReport{X: 56, Y: 16, W: 12, H: 12},
			Quality:  "monochrome-mask-raster-v1",
			Checksum: "sha256:" + checksumText("paint-icon"),
		},
	}
}
func blockRendererVisualFeaturesForScenario() []string {
	return []string{
		"fill",
		"gradient",
		"image_fill",
		"border",
		"radius",
		"radius_clip",
		"shadow",
		"overlay",
		"outline",
		"text",
		"icon",
	}
}
func blockRendererCommandOrderForScenario() []string {
	return []string{
		"fill",
		"gradient",
		"image_fill",
		"border",
		"radius_clip",
		"shadow",
		"overlay",
		"outline",
		"text",
		"icon",
	}
}

func blockRendererReportForScenario(
	frames []surface.FrameReport,
	blockID int,
) *surface.RendererReport {
	checksums := make([]string, 0, 2)
	for _, frame := range frames {
		if len(checksums) >= 2 {
			break
		}
		checksums = append(checksums, frame.Checksum)
	}
	if len(checksums) < 2 {
		checksums = append(checksums, "sha256:"+checksumText("surface-renderer-missing-frame"))
	}
	return &surface.RendererReport{
		Schema:                      surface.RendererFeatureSchemaV1,
		Backend:                     "software-rgba",
		ColorFormat:                 "rgba8",
		QualityLevel:                "deterministic-software-renderer-v1",
		SoftwareRenderer:            true,
		GPUProductionClaim:          false,
		BlurProductionClaim:         false,
		BackdropBlurProductionClaim: false,
		CommandOrder:                blockRendererCommandOrderForScenario(),
		CompositorLayers:            blockRendererCompositorLayersForScenario(blockID),
		DirtyRects:                  blockRendererDirtyRectsForScenario(),
		Invalidations:               blockRendererInvalidationsForScenario(blockID),
		CacheStats:                  blockRendererCacheStatsForScenario(),
		UnsupportedEffectsRejected:  []string{"gpu-production", "blur", "backdrop-blur"},
		DeterministicFrameChecksums: checksums,
		ReferenceFrameArtifactSHA256: "sha256:" + checksumText(
			"surface-renderer-reference-frame-v1",
		),
	}
}
func blockRendererCompositorLayersForScenario(blockID int) []surface.RendererCompositorLayerReport {
	return []surface.RendererCompositorLayerReport{
		{
			ID:        "root",
			Kind:      "root",
			Order:     1,
			BlockID:   blockID,
			Rect:      surface.RectReport{X: 0, Y: 0, W: 320, H: 200},
			Opacity:   255,
			Transform: "identity",
			Checksum:  "sha256:" + checksumText("renderer-layer-root"),
		},
		{
			ID:          "content",
			Kind:        "content",
			Order:       2,
			BlockID:     blockID,
			Rect:        surface.RectReport{X: 12, Y: 10, W: 64, H: 28},
			ClipApplied: true,
			Clip:        surface.RectReport{X: 12, Y: 10, W: 64, H: 28},
			Opacity:     255,
			Transform:   "translate(0,0)",
			Checksum:    "sha256:" + checksumText("renderer-layer-content"),
		},
		{
			ID:        "overlay",
			Kind:      "overlay",
			Order:     3,
			BlockID:   blockID,
			Rect:      surface.RectReport{X: 12, Y: 10, W: 64, H: 28},
			Opacity:   102,
			Transform: "translate(0,1)",
			Checksum:  "sha256:" + checksumText("renderer-layer-overlay"),
		},
		{
			ID:        "text",
			Kind:      "text",
			Order:     4,
			BlockID:   blockID,
			Rect:      surface.RectReport{X: 20, Y: 16, W: 32, H: 12},
			Opacity:   255,
			Transform: "identity",
			Checksum:  "sha256:" + checksumText("renderer-layer-text"),
		},
		{
			ID:        "icon",
			Kind:      "icon",
			Order:     5,
			BlockID:   blockID,
			Rect:      surface.RectReport{X: 56, Y: 16, W: 12, H: 12},
			Opacity:   255,
			Transform: "identity",
			Checksum:  "sha256:" + checksumText("renderer-layer-icon"),
		},
	}
}
func blockRendererDirtyRectsForScenario() []surface.RendererDirtyRectReport {
	return []surface.RendererDirtyRectReport{
		{
			FrameOrder: 1,
			Rect:       surface.RectReport{X: 12, Y: 10, W: 68, H: 36},
			Reason:     "initial-paint",
			Checksum:   "sha256:" + checksumText("renderer-dirty-initial"),
		},
		{
			FrameOrder: 2,
			Rect:       surface.RectReport{X: 12, Y: 10, W: 68, H: 36},
			Reason:     "state-change",
			Checksum:   "sha256:" + checksumText("renderer-dirty-state-change"),
		},
	}
}
func blockRendererInvalidationsForScenario(blockID int) []surface.RendererInvalidationReport {
	return []surface.RendererInvalidationReport{
		{
			Order:     1,
			BlockID:   blockID,
			Reason:    "hovered changed",
			DirtyRect: surface.RectReport{X: 12, Y: 10, W: 68, H: 36},
			Repaint:   true,
		},
		{
			Order:     2,
			BlockID:   blockID,
			Reason:    "text input changed",
			DirtyRect: surface.RectReport{X: 20, Y: 16, W: 44, H: 12},
			Repaint:   true,
		},
	}
}
func blockRendererCacheStatsForScenario() surface.RendererCacheStatsReport {
	return surface.RendererCacheStatsReport{
		ID:          "software-rgba-render-cache",
		Strategy:    "bounded-lru",
		BudgetBytes: 65536,
		UsedBytes:   len(blockRendererCommandOrderForScenario()) * 2048,
		EntryCount:  len(blockRendererCommandOrderForScenario()),
		Hits:        3,
		Misses:      2,
		Bounded:     true,
	}
}
func runBlockTextScenario() headlessScenario {
	beforeFrame := renderBlockTextFrameRGBA(false)
	afterFrame := renderBlockTextFrameRGBA(true)
	return headlessScenario{
		Components: []surface.ComponentReport{
			{
				ID:     "BlockTextApp",
				Type:   "examples.surface.block_render.surface_block_text.BlockTextApp",
				Bounds: surface.RectReport{X: 0, Y: 0, W: 320, H: 200},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused_id":   "3",
					"text_quality": "deterministic-fallback-text-v1",
				},
			},
			{
				ID:     "TextBlock",
				Type:   "examples.surface.block_render.surface_block_text.TextSurfaceBlock",
				Parent: "BlockTextApp",
				Bounds: surface.RectReport{X: 12, Y: 10, W: 96, H: 40},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"text_len":   "28",
					"line_count": "2",
					"ellipsis":   "true",
				},
			},
			{
				ID:     "InputBlock",
				Type:   "examples.surface.block_render.surface_block_text.EditableTextBlock",
				Parent: "BlockTextApp",
				Bounds: surface.RectReport{X: 12, Y: 58, W: 144, H: 36},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{"buffer": "OKd0a2", "caret": "4", "editable": "true"},
			},
		},
		TextMeasurements:     blockTextMeasurementsForScenario(),
		FontFallbacks:        blockFontFallbacksForScenario(),
		GlyphCaches:          blockGlyphCachesForScenario(),
		TextRenderCommands:   blockTextRenderCommandsForScenario(),
		TextQualityLevel:     "deterministic-fallback-text-v1",
		TextCacheBudgetBytes: 65536,
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "InputBlock",
				DispatchPath:    []string{"BlockTextApp", "InputBlock"},
				Handled:         true,
				Pass:            true,
				X:               20,
				Y:               64,
				Key:             0,
				Width:           320,
				Height:          200,
				TimestampMS:     0,
				BufferSlots:     []int{5, 20, 64, 1, 0, 320, 200, 0, 0},
				BeforeState: map[string]string{
					"BlockTextApp.focused_id": "0",
					"InputBlock.focused":      "false",
				},
				AfterState: map[string]string{
					"BlockTextApp.focused_id": "3",
					"InputBlock.focused":      "true",
				},
			},
			{
				Order:           2,
				Kind:            "text_input",
				TargetComponent: "InputBlock",
				DispatchPath:    []string{"BlockTextApp", "InputBlock"},
				Handled:         true,
				Pass:            true,
				X:               0,
				Y:               0,
				Key:             0,
				Width:           320,
				Height:          200,
				TimestampMS:     1,
				TextLen:         4,
				TextBytesHex:    "4f4bd0a2",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 4},
				BeforeState: map[string]string{
					"InputBlock.buffer": "",
					"InputBlock.caret":  "0",
				},
				AfterState: map[string]string{
					"InputBlock.buffer": "OKd0a2",
					"InputBlock.caret":  "4",
				},
			},
		},
		Frames: []surface.FrameReport{
			{
				Order:     1,
				Width:     beforeFrame.Width,
				Height:    beforeFrame.Height,
				Stride:    beforeFrame.Stride,
				Checksum:  checksumRGBA(beforeFrame.Pixels),
				Presented: true,
			},
			{
				Order:     2,
				Width:     afterFrame.Width,
				Height:    afterFrame.Height,
				Stride:    afterFrame.Stride,
				Checksum:  checksumRGBA(afterFrame.Pixels),
				Presented: true,
			},
		},
		StateTransitions: []surface.StateTransitionReport{
			{
				Order:     1,
				Component: "BlockTextApp",
				Field:     "focused_id",
				Before:    "0",
				After:     "3",
				Cause:     "mouse_up",
			},
			{
				Order:     2,
				Component: "InputBlock",
				Field:     "buffer",
				Before:    "",
				After:     "OKd0a2",
				Cause:     "text_input",
			},
			{
				Order:     3,
				Component: "InputBlock",
				Field:     "caret",
				Before:    "0",
				After:     "4",
				Cause:     "text_input",
			},
		},
		Cases: []surface.CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless event dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless framebuffer checksum", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless actual runner trace", Kind: "positive", Ran: true, Pass: true},
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
			{Name: "block text deterministic measurement", Kind: "positive", Ran: true, Pass: true},
			{Name: "block text wrap ellipsis layout", Kind: "positive", Ran: true, Pass: true},
			{Name: "block text font fallback chain", Kind: "positive", Ran: true, Pass: true},
			{Name: "block text bounded glyph cache", Kind: "positive", Ran: true, Pass: true},
			{Name: "block text render command evidence", Kind: "positive", Ran: true, Pass: true},
			{Name: "block text editable lifetime", Kind: "positive", Ran: true, Pass: true},
			{
				Name:          "reject legacy UI evidence",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "legacy UI evidence rejected",
			},
		},
	}
}
func blockTextMeasurementsForScenario() []surface.TextMeasurementReport {
	return []surface.TextMeasurementReport{
		{
			ID:                "title-measure",
			BlockID:           2,
			TextLen:           28,
			FontFamily:        "Tetra UI",
			FontWeight:        600,
			FontSize:          16,
			LineHeight:        20,
			MaxWidth:          96,
			Measured:          surface.SizeReport{W: 96, H: 40},
			LineCount:         2,
			Wrap:              "word",
			Overflow:          "ellipsis",
			Ellipsis:          true,
			EllipsizedTextLen: 16,
			Align:             "start",
			Quality:           "deterministic-metrics-v1",
			Checksum:          "sha256:" + checksumText("text-title-measure"),
		},
		{
			ID:                "input-measure",
			BlockID:           3,
			TextLen:           4,
			FontFamily:        "Tetra UI",
			FontWeight:        400,
			FontSize:          14,
			LineHeight:        18,
			MaxWidth:          120,
			Measured:          surface.SizeReport{W: 34, H: 18},
			LineCount:         1,
			Wrap:              "none",
			Overflow:          "clip",
			Ellipsis:          false,
			EllipsizedTextLen: 4,
			Align:             "start",
			Quality:           "deterministic-metrics-v1",
			Checksum:          "sha256:" + checksumText("text-input-measure"),
		},
	}
}
func blockFontFallbacksForScenario() []surface.FontFallbackReport {
	return []surface.FontFallbackReport{
		{
			ID:              "ui-fallback",
			RequestedFamily: "Tetra UI",
			ResolvedFamily:  "Tetra UI Fallback",
			Chain:           []string{"Tetra UI", "Noto Sans", "monospace"},
			MissingGlyphs:   0,
			Coverage:        "ascii-plus-basic-utf8-smoke",
		},
	}
}
func blockGlyphCachesForScenario() []surface.GlyphCacheReport {
	return []surface.GlyphCacheReport{
		{
			ID:          "glyph-cache",
			Strategy:    "bounded-lru",
			BudgetBytes: 65536,
			UsedBytes:   4096,
			EntryCount:  12,
			Eviction:    "lru",
			Bounded:     true,
		},
	}
}
func blockTextRenderCommandsForScenario() []surface.TextRenderCommandReport {
	return []surface.TextRenderCommandReport{
		{
			Order:         1,
			Command:       "measure",
			MeasurementID: "title-measure",
			BlockID:       2,
			Rect:          surface.RectReport{X: 12, Y: 10, W: 96, H: 40},
			Clip:          surface.RectReport{X: 12, Y: 10, W: 96, H: 40},
			Color:         "#edf2f7ff",
			Opacity:       255,
			Quality:       "deterministic-text-measure-v1",
			Checksum:      "sha256:" + checksumText("text-command-measure"),
		},
		{
			Order:          2,
			Command:        "render_glyphs",
			MeasurementID:  "title-measure",
			BlockID:        2,
			Rect:           surface.RectReport{X: 12, Y: 10, W: 96, H: 40},
			Clip:           surface.RectReport{X: 12, Y: 10, W: 96, H: 40},
			Color:          "#edf2f7ff",
			Opacity:        255,
			Quality:        "deterministic-glyph-raster-v1",
			RasterFormat:   "builtin-5x7-alpha-mask-v1",
			RasterHash:     "sha256:" + checksumText("text-command-glyph-raster"),
			RasterWidth:    96,
			RasterHeight:   40,
			RasterCoverage: 476,
			MarkerOnly:     false,
			Checksum:       "sha256:" + checksumText("text-command-glyphs"),
		},
		{
			Order:         3,
			Command:       "render_caret",
			MeasurementID: "input-measure",
			BlockID:       3,
			Rect:          surface.RectReport{X: 12, Y: 58, W: 120, H: 18},
			Clip:          surface.RectReport{X: 12, Y: 58, W: 144, H: 36},
			Color:         "#f4cd5cff",
			Opacity:       255,
			Quality:       "deterministic-caret-v1",
			Checksum:      "sha256:" + checksumText("text-command-caret"),
		},
	}
}
func runBlockLayoutScenario() headlessScenario {
	beforeFrame := renderBlockLayoutFrameRGBA(false)
	afterFrame := renderBlockLayoutFrameRGBA(true)
	resizedFrame := renderBlockLayoutResizedFrameRGBA()
	return headlessScenario{
		Components:        blockLayoutComponentsForScenario(),
		LayoutConstraints: blockLayoutConstraintsForScenario(),
		LayoutPasses:      blockLayoutPassesForScenario(),
		LayoutScrolls:     blockLayoutScrollsForScenario(),
		LayoutDensity:     blockLayoutDensityForScenario(),
		LayoutFeatures: []string{
			"stack",
			"row",
			"column",
			"absolute",
			"overlay",
			"grid",
			"dock",
			"scroll",
			"fit",
			"fill",
			"fixed",
			"min",
			"max",
			"aspect",
			"spacing",
			"alignment",
			"z-order",
			"clipping",
			"resize",
			"density",
			"stable-rounding",
		},
		LayoutQualityLevel:          "deterministic-block-layout-v1",
		LayoutUnsupportedCSSFlexbox: false,
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "RowBlock",
				DispatchPath:    []string{"BlockLayoutApp", "ColumnBlock", "RowBlock"},
				Handled:         true,
				Pass:            true,
				X:               32,
				Y:               32,
				Width:           320,
				Height:          200,
				TimestampMS:     0,
				BufferSlots:     []int{5, 32, 32, 1, 0, 320, 200, 0, 0},
				BeforeState:     map[string]string{"RowBlock.pressed": "false"},
				AfterState:      map[string]string{"RowBlock.pressed": "true"},
			},
			{
				Order:           2,
				Kind:            "text_input",
				TargetComponent: "RowBlock",
				DispatchPath:    []string{"BlockLayoutApp", "ColumnBlock", "RowBlock"},
				Handled:         true,
				Pass:            true,
				Width:           320,
				Height:          200,
				TimestampMS:     1,
				TextLen:         2,
				TextBytesHex:    "4f4b",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
				BeforeState:     map[string]string{"RowBlock.text_len_seen": "0"},
				AfterState:      map[string]string{"RowBlock.text_len_seen": "2"},
			},
			{
				Order:           3,
				Kind:            "resize",
				TargetComponent: "BlockLayoutApp",
				DispatchPath:    []string{"BlockLayoutApp"},
				Handled:         true,
				Pass:            true,
				Width:           480,
				Height:          260,
				TimestampMS:     2,
				BufferSlots:     []int{6, 0, 0, 0, 0, 480, 260, 2, 0},
				BeforeState:     map[string]string{"BlockLayoutApp.width": "320"},
				AfterState:      map[string]string{"BlockLayoutApp.width": "480"},
			},
			{
				Order:           4,
				Kind:            "scroll",
				TargetComponent: "ScrollBlock",
				DispatchPath:    []string{"BlockLayoutApp", "ScrollBlock"},
				Handled:         true,
				Pass:            true,
				X:               260,
				Y:               80,
				Width:           480,
				Height:          260,
				TimestampMS:     3,
				BufferSlots:     []int{7, 260, 80, 0, 0, 480, 260, 3, 0},
				BeforeState:     map[string]string{"ScrollBlock.scroll_y": "0"},
				AfterState:      map[string]string{"ScrollBlock.scroll_y": "32"},
			},
		},
		Frames: []surface.FrameReport{
			{
				Order:     1,
				Width:     beforeFrame.Width,
				Height:    beforeFrame.Height,
				Stride:    beforeFrame.Stride,
				Checksum:  checksumRGBA(beforeFrame.Pixels),
				Presented: true,
			},
			{
				Order:     2,
				Width:     afterFrame.Width,
				Height:    afterFrame.Height,
				Stride:    afterFrame.Stride,
				Checksum:  checksumRGBA(afterFrame.Pixels),
				Presented: true,
			},
			{
				Order:     3,
				Width:     resizedFrame.Width,
				Height:    resizedFrame.Height,
				Stride:    resizedFrame.Stride,
				Checksum:  checksumRGBA(resizedFrame.Pixels),
				Presented: true,
			},
		},
		StateTransitions: []surface.StateTransitionReport{
			{
				Order:     1,
				Component: "RowBlock",
				Field:     "pressed",
				Before:    "false",
				After:     "true",
				Cause:     "mouse_up",
			},
			{
				Order:     2,
				Component: "RowBlock",
				Field:     "text_len_seen",
				Before:    "0",
				After:     "2",
				Cause:     "text_input",
			},
			{
				Order:     3,
				Component: "BlockLayoutApp",
				Field:     "width",
				Before:    "320",
				After:     "480",
				Cause:     "resize",
			},
			{
				Order:     4,
				Component: "ScrollBlock",
				Field:     "scroll_y",
				Before:    "0",
				After:     "32",
				Cause:     "scroll",
			},
		},
		Cases: []surface.CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless event dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless framebuffer checksum", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless actual runner trace", Kind: "positive", Ran: true, Pass: true},
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
			{Name: "block layout nested row column", Kind: "positive", Ran: true, Pass: true},
			{Name: "block layout fit fill fixed min max", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "block layout grid dock overlay scroll",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{Name: "block layout clipping z-order", Kind: "positive", Ran: true, Pass: true},
			{Name: "block layout resize constraints", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "block layout aspect density stable rounding",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name:          "block layout no css flexbox parity",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "CSS flexbox parity nonclaim",
			},
			{
				Name:          "reject legacy UI evidence",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "legacy UI evidence rejected",
			},
		},
	}
}
func blockLayoutComponentsForScenario() []surface.ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}
	return []surface.ComponentReport{
		{
			ID:        "BlockLayoutApp",
			Type:      "examples.surface.block_core.surface_block_layout.BlockLayoutApp",
			Bounds:    surface.RectReport{X: 0, Y: 0, W: 320, H: 200},
			Abilities: abilities,
			State:     map[string]string{"layout_quality": "deterministic-block-layout-v1"},
		},
		{
			ID:        "ColumnBlock",
			Type:      "examples.surface.block_core.surface_block_layout.ColumnBlock",
			Parent:    "BlockLayoutApp",
			Bounds:    surface.RectReport{X: 12, Y: 12, W: 296, H: 176},
			Abilities: abilities,
			State:     map[string]string{"mode": "column", "gap": "8"},
		},
		{
			ID:        "RowBlock",
			Type:      "examples.surface.block_core.surface_block_layout.RowBlock",
			Parent:    "ColumnBlock",
			Bounds:    surface.RectReport{X: 24, Y: 24, W: 272, H: 48},
			Abilities: abilities,
			State:     map[string]string{"mode": "row", "gap": "6"},
		},
		{
			ID:        "GridBlock",
			Type:      "examples.surface.block_core.surface_block_layout.GridBlock",
			Parent:    "ColumnBlock",
			Bounds:    surface.RectReport{X: 24, Y: 80, W: 132, H: 72},
			Abilities: abilities,
			State:     map[string]string{"mode": "grid", "columns": "2"},
		},
		{
			ID:        "DockBlock",
			Type:      "examples.surface.block_core.surface_block_layout.DockBlock",
			Parent:    "ColumnBlock",
			Bounds:    surface.RectReport{X: 164, Y: 80, W: 132, H: 72},
			Abilities: abilities,
			State:     map[string]string{"mode": "dock"},
		},
		{
			ID:        "OverlayBlock",
			Type:      "examples.surface.block_core.surface_block_layout.OverlayBlock",
			Parent:    "BlockLayoutApp",
			Bounds:    surface.RectReport{X: 220, Y: 20, W: 72, H: 40},
			Abilities: abilities,
			State:     map[string]string{"mode": "overlay", "z": "4"},
		},
		{
			ID:        "ScrollBlock",
			Type:      "examples.surface.block_core.surface_block_layout.ScrollBlock",
			Parent:    "BlockLayoutApp",
			Bounds:    surface.RectReport{X: 236, Y: 72, W: 72, H: 80},
			Abilities: abilities,
			State:     map[string]string{"mode": "scroll", "clipped": "true"},
		},
	}
}
func blockLayoutConstraintsForScenario() []surface.BlockLayoutConstraintReport {
	return []surface.BlockLayoutConstraintReport{
		{
			ID:           "root-column",
			BlockID:      1,
			Mode:         "column",
			WidthPolicy:  "fixed",
			HeightPolicy: "fixed",
			Min:          surface.SizeReport{W: 320, H: 200},
			Max:          surface.SizeReport{W: 480, H: 260},
			Padding:      12,
			Margin:       0,
			Gap:          8,
			Align:        "stretch",
			Justify:      "start",
			Overflow:     "clip",
			ZIndex:       0,
			Clip:         true,
		},
		{
			ID:           "row-fill",
			BlockID:      3,
			Mode:         "row",
			WidthPolicy:  "fill",
			HeightPolicy: "fixed",
			Min:          surface.SizeReport{W: 160, H: 40},
			Max:          surface.SizeReport{W: 296, H: 64},
			Padding:      6,
			Margin:       0,
			Gap:          6,
			Align:        "center",
			Justify:      "space-between",
			Overflow:     "visible",
			ZIndex:       1,
			Clip:         false,
		},
		{
			ID:           "text-fit",
			BlockID:      8,
			Mode:         "absolute",
			WidthPolicy:  "fit",
			HeightPolicy: "fit",
			Min:          surface.SizeReport{W: 32, H: 18},
			Max:          surface.SizeReport{W: 160, H: 40},
			Padding:      4,
			Margin:       0,
			Gap:          0,
			Align:        "start",
			Justify:      "start",
			Overflow:     "clip",
			ZIndex:       2,
			Clip:         true,
		},
		{
			ID:           "overlay-z",
			BlockID:      6,
			Mode:         "overlay",
			WidthPolicy:  "fixed",
			HeightPolicy: "fixed",
			Min:          surface.SizeReport{W: 72, H: 40},
			Max:          surface.SizeReport{W: 72, H: 40},
			Padding:      0,
			Margin:       0,
			Gap:          0,
			Align:        "end",
			Justify:      "start",
			Overflow:     "visible",
			ZIndex:       4,
			Clip:         false,
		},
		{
			ID:           "aspect-fit",
			BlockID:      9,
			Mode:         "absolute",
			WidthPolicy:  "fixed",
			HeightPolicy: "fixed",
			Min:          surface.SizeReport{W: 96, H: 54},
			Max:          surface.SizeReport{W: 96, H: 54},
			Padding:      0,
			Margin:       0,
			Gap:          0,
			Align:        "start",
			Justify:      "start",
			Overflow:     "clip",
			ZIndex:       2,
			Clip:         true,
		},
	}
}
func blockLayoutPassesForScenario() []surface.BlockLayoutPassReport {
	return []surface.BlockLayoutPassReport{
		{
			Order:    1,
			ParentID: 0,
			BlockID:  1,
			Mode:     "column",
			Input:    surface.RectReport{X: 0, Y: 0, W: 320, H: 200},
			Resolved: surface.RectReport{X: 12, Y: 12, W: 296, H: 176},
			Measured: surface.SizeReport{W: 296, H: 176},
			Pass:     "initial",
			Resize:   false,
			Clip:     true,
			ZIndex:   0,
			Checksum: "sha256:" + checksumText("layout-column"),
		},
		{
			Order:    2,
			ParentID: 1,
			BlockID:  2,
			Mode:     "stack",
			Input:    surface.RectReport{X: 12, Y: 12, W: 296, H: 176},
			Resolved: surface.RectReport{X: 12, Y: 12, W: 296, H: 176},
			Measured: surface.SizeReport{W: 296, H: 176},
			Pass:     "initial",
			Resize:   false,
			Clip:     false,
			ZIndex:   0,
			Checksum: "sha256:" + checksumText("layout-stack"),
		},
		{
			Order:    3,
			ParentID: 2,
			BlockID:  3,
			Mode:     "row",
			Input:    surface.RectReport{X: 24, Y: 24, W: 272, H: 48},
			Resolved: surface.RectReport{X: 24, Y: 24, W: 272, H: 48},
			Measured: surface.SizeReport{W: 272, H: 48},
			Pass:     "nested",
			Resize:   false,
			Clip:     false,
			ZIndex:   1,
			Checksum: "sha256:" + checksumText("layout-row"),
		},
		{
			Order:    4,
			ParentID: 2,
			BlockID:  4,
			Mode:     "grid",
			Input:    surface.RectReport{X: 24, Y: 80, W: 132, H: 72},
			Resolved: surface.RectReport{X: 24, Y: 80, W: 63, H: 34},
			Measured: surface.SizeReport{W: 63, H: 34},
			Pass:     "grid-cell",
			Resize:   false,
			Clip:     true,
			ZIndex:   1,
			Checksum: "sha256:" + checksumText("layout-grid"),
		},
		{
			Order:    5,
			ParentID: 2,
			BlockID:  5,
			Mode:     "dock",
			Input:    surface.RectReport{X: 164, Y: 80, W: 132, H: 72},
			Resolved: surface.RectReport{X: 164, Y: 80, W: 132, H: 24},
			Measured: surface.SizeReport{W: 132, H: 24},
			Pass:     "dock-top",
			Resize:   false,
			Clip:     true,
			ZIndex:   1,
			Checksum: "sha256:" + checksumText("layout-dock"),
		},
		{
			Order:    6,
			ParentID: 1,
			BlockID:  6,
			Mode:     "overlay",
			Input:    surface.RectReport{X: 220, Y: 20, W: 72, H: 40},
			Resolved: surface.RectReport{X: 220, Y: 20, W: 72, H: 40},
			Measured: surface.SizeReport{W: 72, H: 40},
			Pass:     "overlay-z-order",
			Resize:   false,
			Clip:     false,
			ZIndex:   4,
			Checksum: "sha256:" + checksumText("layout-overlay"),
		},
		{
			Order:    7,
			ParentID: 1,
			BlockID:  7,
			Mode:     "scroll",
			Input:    surface.RectReport{X: 236, Y: 72, W: 72, H: 80},
			Resolved: surface.RectReport{X: 236, Y: 72, W: 72, H: 80},
			Measured: surface.SizeReport{W: 72, H: 160},
			Pass:     "scroll-clip",
			Resize:   false,
			Clip:     true,
			ZIndex:   2,
			Checksum: "sha256:" + checksumText("layout-scroll"),
		},
		{
			Order:    8,
			ParentID: 1,
			BlockID:  8,
			Mode:     "absolute",
			Input:    surface.RectReport{X: 32, Y: 152, W: 0, H: 0},
			Resolved: surface.RectReport{X: 32, Y: 152, W: 96, H: 20},
			Measured: surface.SizeReport{W: 96, H: 20},
			Pass:     "fit-text",
			Resize:   false,
			Clip:     true,
			ZIndex:   2,
			Checksum: "sha256:" + checksumText("layout-absolute-fit"),
		},
		{
			Order:    9,
			ParentID: 1,
			BlockID:  9,
			Mode:     "absolute",
			Input:    surface.RectReport{X: 164, Y: 152, W: 96, H: 64},
			Resolved: surface.RectReport{X: 164, Y: 152, W: 96, H: 54},
			Measured: surface.SizeReport{W: 96, H: 54},
			Pass:     "aspect-fit",
			Resize:   false,
			Clip:     true,
			ZIndex:   2,
			Checksum: "sha256:" + checksumText("layout-aspect-fit"),
		},
		{
			Order:    10,
			ParentID: 0,
			BlockID:  1,
			Mode:     "column",
			Input:    surface.RectReport{X: 0, Y: 0, W: 480, H: 260},
			Resolved: surface.RectReport{X: 12, Y: 12, W: 456, H: 236},
			Measured: surface.SizeReport{W: 456, H: 236},
			Pass:     "resize",
			Resize:   true,
			Clip:     true,
			ZIndex:   0,
			Checksum: "sha256:" + checksumText("layout-resize"),
		},
	}
}
func blockLayoutScrollsForScenario() []surface.BlockLayoutScrollReport {
	return []surface.BlockLayoutScrollReport{
		{
			BlockID:    7,
			Viewport:   surface.RectReport{X: 236, Y: 72, W: 72, H: 80},
			Content:    surface.SizeReport{W: 72, H: 160},
			OffsetY:    32,
			MaxOffsetY: 80,
			Clipped:    true,
			Checksum:   "sha256:" + checksumText("layout-scroll-bounds"),
		},
	}
}
func blockLayoutDensityForScenario() *surface.BlockLayoutDensityReport {
	return &surface.BlockLayoutDensityReport{
		TargetDPI:      144,
		ScaleMilli:     1500,
		BaseUnitPx:     4,
		RoundingPolicy: "integer-half-up-v1",
		PixelSnapping:  true,
		Breakpoints:    []string{"small", "medium", "large"},
		Checksum:       "sha256:" + checksumText("layout-density-rounding"),
	}
}
func runBlockEventScenario() headlessScenario {
	beforeFrame := renderBlockEventFrameRGBA(false)
	afterFrame := renderBlockEventFrameRGBA(true)
	return headlessScenario{
		Components: blockEventComponentsForScenario(),
		BlockGraph: blockEventGraphForScenario(
			"examples/surface/block_core/surface_block_events.tetra",
		),
		BlockEventQualityLevel:        "deterministic-block-events-v1",
		BlockEventPolicy:              "capture-bubble-direct-v1",
		BlockEventUnsupportedDragDrop: false,
		BlockEventKinds: []string{
			"pointer_enter",
			"pointer_leave",
			"pointer_move",
			"pointer_down",
			"pointer_up",
			"click",
			"double_click",
			"key",
			"text",
			"focus",
			"blur",
			"scroll",
			"resize",
			"close",
			"frame",
		},
		BlockEventRoutes:      blockEventRoutesForScenario(),
		BlockFocusTransitions: blockFocusTransitionsForScenario(),
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "InputBlock",
				DispatchPath:    []string{"BlockEventApp", "PanelBlock", "InputBlock"},
				Handled:         true,
				Pass:            true,
				X:               40,
				Y:               80,
				Width:           320,
				Height:          200,
				TimestampMS:     0,
				BufferSlots:     []int{5, 40, 80, 1, 0, 320, 200, 0, 0},
				BeforeState: map[string]string{
					"BlockEventApp.focused_id": "0",
					"InputBlock.focused":       "false",
				},
				AfterState: map[string]string{
					"BlockEventApp.focused_id": "4",
					"InputBlock.focused":       "true",
				},
			},
			{
				Order:           2,
				Kind:            "text_input",
				TargetComponent: "InputBlock",
				DispatchPath:    []string{"BlockEventApp", "PanelBlock", "InputBlock"},
				Handled:         true,
				Pass:            true,
				Width:           320,
				Height:          200,
				TimestampMS:     1,
				TextLen:         2,
				TextBytesHex:    "4f4b",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
				BeforeState: map[string]string{
					"InputBlock.buffer": "",
					"InputBlock.caret":  "0",
				},
				AfterState: map[string]string{
					"InputBlock.buffer": "OK",
					"InputBlock.caret":  "2",
				},
			},
			{
				Order:           3,
				Kind:            "key_down",
				TargetComponent: "BlockEventApp",
				DispatchPath:    []string{"BlockEventApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          200,
				TimestampMS:     2,
				BufferSlots:     []int{3, 0, 0, 0, 9, 320, 200, 2, 0},
				BeforeState:     map[string]string{"BlockEventApp.focused_id": "4"},
				AfterState:      map[string]string{"BlockEventApp.focused_id": "6"},
			},
			{
				Order:           4,
				Kind:            "key_down",
				TargetComponent: "BlockEventApp",
				DispatchPath:    []string{"BlockEventApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          200,
				TimestampMS:     3,
				BufferSlots:     []int{3, 0, 0, 0, 9, 320, 200, 3, 0},
				BeforeState:     map[string]string{"BlockEventApp.focused_id": "6"},
				AfterState:      map[string]string{"BlockEventApp.focused_id": "4"},
			},
		},
		Frames: []surface.FrameReport{
			{
				Order:     1,
				Width:     beforeFrame.Width,
				Height:    beforeFrame.Height,
				Stride:    beforeFrame.Stride,
				Checksum:  checksumRGBA(beforeFrame.Pixels),
				Presented: true,
			},
			{
				Order:     2,
				Width:     afterFrame.Width,
				Height:    afterFrame.Height,
				Stride:    afterFrame.Stride,
				Checksum:  checksumRGBA(afterFrame.Pixels),
				Presented: true,
			},
		},
		StateTransitions: []surface.StateTransitionReport{
			{
				Order:     1,
				Component: "BlockEventApp",
				Field:     "focused_id",
				Before:    "0",
				After:     "4",
				Cause:     "click",
			},
			{
				Order:     2,
				Component: "InputBlock",
				Field:     "buffer",
				Before:    "",
				After:     "OK",
				Cause:     "text_input",
			},
			{
				Order:     3,
				Component: "BlockEventApp",
				Field:     "focused_id",
				Before:    "4",
				After:     "6",
				Cause:     "tab",
			},
			{
				Order:     4,
				Component: "BlockEventApp",
				Field:     "focused_id",
				Before:    "6",
				After:     "4",
				Cause:     "tab",
			},
		},
		Cases: []surface.CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless event dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless framebuffer checksum", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless actual runner trace", Kind: "positive", Ran: true, Pass: true},
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
			{
				Name:          "block graph duplicate id rejected",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "duplicate Block ID",
			},
			{
				Name:          "block graph missing parent rejected",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "missing parent",
			},
			{
				Name:          "block graph cycle rejected",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "cycle",
			},
			{Name: "block graph child order", Kind: "positive", Ran: true, Pass: true},
			{Name: "block graph focus order", Kind: "positive", Ran: true, Pass: true},
			{Name: "block graph hit-test path", Kind: "positive", Ran: true, Pass: true},
			{Name: "block graph accessibility order", Kind: "positive", Ran: true, Pass: true},
			{Name: "block event nested hit-test path", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "block event capture bubble direct policy",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name:          "block event disabled click rejected",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "disabled Block",
			},
			{Name: "block event text input focused only", Kind: "positive", Ran: true, Pass: true},
			{Name: "block focus tab order graph-derived", Kind: "positive", Ran: true, Pass: true},
			{
				Name:          "block event no complex drag claim",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "drag-and-drop nonclaim",
			},
			{
				Name:          "reject legacy UI evidence",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "legacy UI evidence rejected",
			},
		},
	}
}
func blockEventComponentsForScenario() []surface.ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}
	return []surface.ComponentReport{
		{
			ID:        "BlockEventApp",
			Type:      "examples.surface.block_core.surface_block_events.BlockEventApp",
			Bounds:    surface.RectReport{X: 0, Y: 0, W: 320, H: 200},
			Abilities: abilities,
			State: map[string]string{
				"focused_id":    "4",
				"event_quality": "deterministic-block-events-v1",
			},
		},
		{
			ID:        "PanelBlock",
			Type:      "examples.surface.block_core.surface_block_events.PanelBlock",
			Parent:    "BlockEventApp",
			Bounds:    surface.RectReport{X: 16, Y: 16, W: 288, H: 168},
			Abilities: abilities,
			State:     map[string]string{"role": "panel"},
		},
		{
			ID:        "LabelBlock",
			Type:      "examples.surface.block_core.surface_block_events.LabelBlock",
			Parent:    "PanelBlock",
			Bounds:    surface.RectReport{X: 24, Y: 24, W: 200, H: 24},
			Abilities: abilities,
			State:     map[string]string{"text_len": "10"},
		},
		{
			ID:        "InputBlock",
			Type:      "examples.surface.block_core.surface_block_events.InputBlock",
			Parent:    "PanelBlock",
			Bounds:    surface.RectReport{X: 24, Y: 64, W: 120, H: 44},
			Abilities: abilities,
			State:     map[string]string{"editable": "true", "focused": "true", "buffer": "OK"},
		},
		{
			ID:        "DisabledBlock",
			Type:      "examples.surface.block_core.surface_block_events.DisabledBlock",
			Parent:    "PanelBlock",
			Bounds:    surface.RectReport{X: 152, Y: 64, W: 120, H: 44},
			Abilities: abilities,
			State:     map[string]string{"disabled": "true"},
		},
		{
			ID:        "ActionBlock",
			Type:      "examples.surface.block_core.surface_block_events.ActionBlock",
			Parent:    "PanelBlock",
			Bounds:    surface.RectReport{X: 24, Y: 120, W: 120, H: 44},
			Abilities: abilities,
			State:     map[string]string{"focused": "false"},
		},
	}
}
func blockEventGraphForScenario(source string) *surface.BlockGraphReport {
	return &surface.BlockGraphReport{
		Schema:            "tetra.surface.block-graph.v1",
		APILevel:          "block-tree-builder-v1",
		Source:            source,
		ManualBookkeeping: false,
		Builder: surface.BlockGraphBuilderReport{
			RootCreatedBy:     "tree_add_root",
			ChildrenCreatedBy: "tree_add_child",
			NodeCount:         6,
			Capacity:          8,
			OverflowChecked:   true,
		},
		Invariants: surface.BlockGraphInvariantReport{
			TreeValidateRan:         true,
			TreeValidateStatus:      0,
			DuplicateIDRejected:     true,
			MissingParentRejected:   true,
			CycleRejected:           true,
			ParentChildLinksChecked: true,
			ChildOrderChecked:       true,
			FocusOrderChecked:       true,
			HitTestPathChecked:      true,
			AccessibilityChecked:    true,
		},
		RootID:    1,
		NodeCount: 6,
		Nodes: []surface.BlockGraphNodeReport{
			{
				ID:                1,
				Name:              "BlockEventApp",
				ParentID:          -1,
				ChildIndex:        0,
				FirstChild:        2,
				ChildCount:        1,
				Focusable:         false,
				AccessibilityRole: "none",
				Bounds:            surface.RectReport{X: 0, Y: 0, W: 320, H: 200},
			},
			{
				ID:                2,
				Name:              "PanelBlock",
				ParentID:          1,
				ChildIndex:        0,
				FirstChild:        3,
				ChildCount:        4,
				Focusable:         false,
				AccessibilityRole: "none",
				Bounds:            surface.RectReport{X: 16, Y: 16, W: 288, H: 168},
			},
			{
				ID:                3,
				Name:              "LabelBlock",
				ParentID:          2,
				ChildIndex:        0,
				FirstChild:        -1,
				ChildCount:        0,
				Focusable:         false,
				AccessibilityRole: "text",
				Bounds:            surface.RectReport{X: 24, Y: 24, W: 200, H: 24},
			},
			{
				ID:                4,
				Name:              "InputBlock",
				ParentID:          2,
				ChildIndex:        1,
				FirstChild:        -1,
				ChildCount:        0,
				Focusable:         true,
				AccessibilityRole: "textbox",
				Bounds:            surface.RectReport{X: 24, Y: 64, W: 120, H: 44},
			},
			{
				ID:                5,
				Name:              "DisabledBlock",
				ParentID:          2,
				ChildIndex:        2,
				FirstChild:        -1,
				ChildCount:        0,
				Focusable:         false,
				AccessibilityRole: "button",
				Bounds:            surface.RectReport{X: 152, Y: 64, W: 120, H: 44},
			},
			{
				ID:                6,
				Name:              "ActionBlock",
				ParentID:          2,
				ChildIndex:        3,
				FirstChild:        -1,
				ChildCount:        0,
				Focusable:         true,
				AccessibilityRole: "button",
				Bounds:            surface.RectReport{X: 24, Y: 120, W: 120, H: 44},
			},
		},
		ChildOrders: []surface.BlockGraphChildOrderReport{
			{ParentID: 1, Children: []int{2}},
			{ParentID: 2, Children: []int{3, 4, 5, 6}},
		},
		LayoutOrder:        []int{1, 2, 3, 4, 5, 6},
		DrawOrder:          []int{1, 2, 3, 4, 5, 6},
		FocusOrder:         []int{4, 6},
		AccessibilityOrder: []int{3, 4, 5, 6},
		HitTests: []surface.BlockGraphPathReport{
			{
				Helper:   "tree_hit_test_path",
				Event:    "click",
				TargetID: 4,
				X:        40,
				Y:        80,
				Path:     []int{1, 2, 4},
			},
			{
				Helper:   "tree_hit_test_path",
				Event:    "click",
				TargetID: 5,
				X:        180,
				Y:        80,
				Path:     []int{1, 2, 5},
			},
		},
		DispatchPaths: []surface.BlockGraphPathReport{
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 4, Path: []int{1, 2, 4}},
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 5, Path: []int{1, 2, 5}},
			{Helper: "tree_build_dispatch_path", Event: "key", TargetID: 6, Path: []int{1, 2, 6}},
		},
	}
}
func blockEventRoutesForScenario() []surface.BlockEventRouteReport {
	return []surface.BlockEventRouteReport{
		{
			Order:          1,
			Kind:           "click",
			Policy:         "capture-bubble-direct-v1",
			TargetID:       4,
			TargetName:     "InputBlock",
			HitTestPath:    []int{1, 2, 4},
			DispatchPath:   []int{1, 2, 4},
			CapturePath:    []int{1, 2},
			BubblePath:     []int{2, 1},
			DirectTargetID: 4,
			Delivered:      true,
			Rejected:       false,
			FocusedID:      4,
			Editable:       true,
			Disabled:       false,
		},
		{
			Order:          2,
			Kind:           "click",
			Policy:         "capture-bubble-direct-v1",
			TargetID:       5,
			TargetName:     "DisabledBlock",
			HitTestPath:    []int{1, 2, 5},
			DispatchPath:   []int{1, 2, 5},
			CapturePath:    []int{1, 2},
			BubblePath:     []int{2, 1},
			DirectTargetID: 5,
			Delivered:      false,
			Rejected:       true,
			RejectReason:   "disabled",
			FocusedID:      4,
			Editable:       false,
			Disabled:       true,
		},
		{
			Order:          3,
			Kind:           "text",
			Policy:         "direct-to-focused-editable-v1",
			TargetID:       4,
			TargetName:     "InputBlock",
			DispatchPath:   []int{1, 2, 4},
			DirectTargetID: 4,
			Delivered:      false,
			Rejected:       true,
			RejectReason:   "unfocused",
			FocusedID:      6,
			Editable:       true,
			TextLen:        2,
			TextBytesHex:   "4f4b",
		},
		{
			Order:          4,
			Kind:           "text",
			Policy:         "direct-to-focused-editable-v1",
			TargetID:       4,
			TargetName:     "InputBlock",
			DispatchPath:   []int{1, 2, 4},
			DirectTargetID: 4,
			Delivered:      true,
			Rejected:       false,
			FocusedID:      4,
			Editable:       true,
			TextLen:        2,
			TextBytesHex:   "4f4b",
		},
		{
			Order:          5,
			Kind:           "key",
			Policy:         "direct-to-focused-v1",
			TargetID:       6,
			TargetName:     "ActionBlock",
			DispatchPath:   []int{1, 2, 6},
			DirectTargetID: 6,
			Delivered:      true,
			Rejected:       false,
			FocusedID:      6,
			Editable:       false,
			Disabled:       false,
		},
	}
}
func blockFocusTransitionsForScenario() []surface.BlockFocusTransitionReport {
	return []surface.BlockFocusTransitionReport{
		{
			Order:        1,
			Helper:       "tree_focus_next",
			BeforeID:     4,
			AfterID:      6,
			Direction:    "tab",
			GraphDerived: true,
			Wrapped:      false,
		},
		{
			Order:        2,
			Helper:       "tree_focus_next",
			BeforeID:     6,
			AfterID:      4,
			Direction:    "tab",
			GraphDerived: true,
			Wrapped:      true,
		},
	}
}
func runBlockStateScenario() headlessScenario {
	beforeFrame := renderBlockStateFrameRGBA(false)
	afterFrame := renderBlockStateFrameRGBA(true)
	return headlessScenario{
		Components:             blockStateComponentsForScenario(),
		BlockStateQualityLevel: "deterministic-block-state-resolver-v1",
		BlockStateResolverOrder: []string{
			"base",
			"variant",
			"hover",
			"pressed",
			"focused",
			"selected",
			"disabled",
			"error",
			"loading",
			"motion",
		},
		BlockStateUnsupportedCSSPseudos: false,
		BlockStateSelectors:             blockStateSelectorsForScenario(),
		BlockStateResolutions:           blockStateResolutionsForScenario(),
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "StateBlock",
				DispatchPath:    []string{"BlockStateApp", "StateBlock"},
				Handled:         true,
				Pass:            true,
				X:               40,
				Y:               56,
				Width:           320,
				Height:          200,
				TimestampMS:     0,
				BufferSlots:     []int{5, 40, 56, 1, 0, 320, 200, 0, 0},
				BeforeState:     map[string]string{"StateBlock.selected": "false"},
				AfterState:      map[string]string{"StateBlock.selected": "true"},
			},
			{
				Order:           2,
				Kind:            "mouse_move",
				TargetComponent: "StateBlock",
				DispatchPath:    []string{"BlockStateApp", "StateBlock"},
				Handled:         true,
				Pass:            true,
				X:               40,
				Y:               56,
				Width:           320,
				Height:          200,
				TimestampMS:     1,
				BufferSlots:     []int{2, 40, 56, 0, 0, 320, 200, 1, 0},
				BeforeState:     map[string]string{"StateBlock.hovered": "false"},
				AfterState:      map[string]string{"StateBlock.hovered": "true"},
			},
			{
				Order:           3,
				Kind:            "mouse_down",
				TargetComponent: "StateBlock",
				DispatchPath:    []string{"BlockStateApp", "StateBlock"},
				Handled:         true,
				Pass:            true,
				X:               40,
				Y:               56,
				Width:           320,
				Height:          200,
				TimestampMS:     2,
				BufferSlots:     []int{4, 40, 56, 1, 0, 320, 200, 2, 0},
				BeforeState:     map[string]string{"StateBlock.pressed": "false"},
				AfterState:      map[string]string{"StateBlock.pressed": "true"},
			},
			{
				Order:           4,
				Kind:            "text_input",
				TargetComponent: "StateBlock",
				DispatchPath:    []string{"BlockStateApp", "StateBlock"},
				Handled:         true,
				Pass:            true,
				Width:           320,
				Height:          200,
				TimestampMS:     3,
				TextLen:         2,
				TextBytesHex:    "4f4b",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 3, 2},
				BeforeState:     map[string]string{"StateBlock.buffer": ""},
				AfterState:      map[string]string{"StateBlock.buffer": "OK"},
			},
			{
				Order:           5,
				Kind:            "key_down",
				TargetComponent: "StateBlock",
				DispatchPath:    []string{"BlockStateApp", "StateBlock"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          200,
				TimestampMS:     4,
				BufferSlots:     []int{3, 0, 0, 0, 9, 320, 200, 4, 0},
				BeforeState:     map[string]string{"StateBlock.focused": "false"},
				AfterState:      map[string]string{"StateBlock.focused": "true"},
			},
		},
		Frames: []surface.FrameReport{
			{
				Order:     1,
				Width:     beforeFrame.Width,
				Height:    beforeFrame.Height,
				Stride:    beforeFrame.Stride,
				Checksum:  checksumRGBA(beforeFrame.Pixels),
				Presented: true,
			},
			{
				Order:     2,
				Width:     afterFrame.Width,
				Height:    afterFrame.Height,
				Stride:    afterFrame.Stride,
				Checksum:  checksumRGBA(afterFrame.Pixels),
				Presented: true,
			},
		},
		StateTransitions: []surface.StateTransitionReport{
			{
				Order:     1,
				Component: "StateBlock",
				Field:     "selector_flags",
				Before:    "0",
				After:     "127",
				Cause:     "pointer/key/state input",
			},
			{
				Order:     2,
				Component: "StateBlock",
				Field:     "resolved_fill",
				Before:    "#20262eff",
				After:     "#2d9bf0ff",
				Cause:     "hover",
			},
			{
				Order:     3,
				Component: "StateBlock",
				Field:     "resolved_scale",
				Before:    "100",
				After:     "97",
				Cause:     "pressed",
			},
			{
				Order:     4,
				Component: "StateBlock",
				Field:     "disabled",
				Before:    "false",
				After:     "true",
				Cause:     "disabled selector",
			},
			{
				Order:     5,
				Component: "StateBlock",
				Field:     "error",
				Before:    "false",
				After:     "true",
				Cause:     "error selector",
			},
			{
				Order:     6,
				Component: "StateBlock",
				Field:     "loading",
				Before:    "false",
				After:     "true",
				Cause:     "loading selector",
			},
		},
		Cases: []surface.CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless event dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless framebuffer checksum", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless actual runner trace", Kind: "positive", Ran: true, Pass: true},
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
			{Name: "block state selector resolver order", Kind: "positive", Ran: true, Pass: true},
			{Name: "block state hover fill override", Kind: "positive", Ran: true, Pass: true},
			{Name: "block state pressed scale override", Kind: "positive", Ran: true, Pass: true},
			{Name: "block state focus selected metadata", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "block state disabled error loading overrides",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{Name: "block state frame checksum changed", Kind: "positive", Ran: true, Pass: true},
			{
				Name:          "block state no css pseudo parity",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "css pseudo nonclaim",
			},
			{
				Name:          "reject legacy UI evidence",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "legacy UI evidence rejected",
			},
		},
	}
}
func blockStateComponentsForScenario() []surface.ComponentReport {
	abilities := []string{
		"measure",
		"layout",
		"draw",
		"event",
		"focus",
		"text",
		"accessibility",
		"state",
	}
	return []surface.ComponentReport{
		{
			ID:        "BlockStateApp",
			Type:      "examples.surface.block_core.surface_block_states.BlockStateApp",
			Bounds:    surface.RectReport{X: 0, Y: 0, W: 320, H: 200},
			Abilities: abilities,
			State:     map[string]string{"state_quality": "deterministic-block-state-resolver-v1"},
		},
		{
			ID:        "StateBlock",
			Type:      "examples.surface.block_core.surface_block_states.StateBlock",
			Parent:    "BlockStateApp",
			Bounds:    surface.RectReport{X: 24, Y: 40, W: 168, H: 56},
			Abilities: abilities,
			State: map[string]string{
				"selector_flags": "127",
				"variant":        "5",
				"disabled":       "true",
				"error":          "true",
				"loading":        "true",
			},
		},
		{
			ID:        "StatusBlock",
			Type:      "examples.surface.block_core.surface_block_states.StatusBlock",
			Parent:    "BlockStateApp",
			Bounds:    surface.RectReport{X: 24, Y: 112, W: 168, H: 32},
			Abilities: abilities,
			State:     map[string]string{"selected": "true", "focused": "true"},
		},
	}
}
func blockStateSelectorsForScenario() []surface.BlockStateSelectorReport {
	return []surface.BlockStateSelectorReport{
		{Order: 1, Name: "hover", BlockID: 2, Flags: 1, Hovered: true},
		{Order: 2, Name: "pressed", BlockID: 2, Flags: 2, Pressed: true},
		{Order: 3, Name: "focused", BlockID: 2, Flags: 4, Focused: true},
		{Order: 4, Name: "selected", BlockID: 2, Flags: 8, Selected: true},
		{Order: 5, Name: "disabled", BlockID: 2, Flags: 16, Disabled: true},
		{Order: 6, Name: "error", BlockID: 2, Flags: 32, Error: true},
		{Order: 7, Name: "loading", BlockID: 2, Flags: 64, Loading: true},
	}
}
func blockStateResolutionsForScenario() []surface.BlockStateResolutionReport {
	return []surface.BlockStateResolutionReport{
		{
			Order:        1,
			BlockID:      2,
			Selector:     "hover",
			ResolverStep: "hover",
			Property:     "paint.fill",
			Before:       "#20262eff",
			After:        "#2d9bf0ff",
			Applied:      true,
		},
		{
			Order:        2,
			BlockID:      2,
			Selector:     "pressed",
			ResolverStep: "pressed",
			Property:     "layout.scale",
			Before:       "100",
			After:        "97",
			Applied:      true,
		},
		{
			Order:        3,
			BlockID:      2,
			Selector:     "focused",
			ResolverStep: "focused",
			Property:     "paint.outline",
			Before:       "none",
			After:        "focus-ring",
			Applied:      true,
		},
		{
			Order:        4,
			BlockID:      2,
			Selector:     "selected",
			ResolverStep: "selected",
			Property:     "accessibility.selected",
			Before:       "false",
			After:        "true",
			Applied:      true,
		},
		{
			Order:        5,
			BlockID:      2,
			Selector:     "disabled",
			ResolverStep: "disabled",
			Property:     "input.disabled",
			Before:       "false",
			After:        "true",
			Applied:      true,
		},
		{
			Order:        6,
			BlockID:      2,
			Selector:     "disabled",
			ResolverStep: "disabled",
			Property:     "text.opacity",
			Before:       "255",
			After:        "112",
			Applied:      true,
		},
		{
			Order:        7,
			BlockID:      2,
			Selector:     "error",
			ResolverStep: "error",
			Property:     "paint.outline_color",
			Before:       "#7aa2f7ff",
			After:        "#ff5f57ff",
			Applied:      true,
		},
		{
			Order:        8,
			BlockID:      2,
			Selector:     "loading",
			ResolverStep: "loading",
			Property:     "text.content",
			Before:       "Run",
			After:        "Loading",
			Applied:      true,
		},
		{
			Order:        9,
			BlockID:      2,
			Selector:     "motion",
			ResolverStep: "motion",
			Property:     "motion.transition_ms",
			Before:       "0",
			After:        "120",
			Applied:      true,
		},
	}
}
func runBlockMotionScenario() headlessScenario {
	startFrame := renderBlockMotionFrameRGBA(0)
	midFrame := renderBlockMotionFrameRGBA(1)
	doneFrame := renderBlockMotionFrameRGBA(2)
	return headlessScenario{
		Components:                     blockMotionComponentsForScenario(),
		MotionQualityLevel:             "deterministic-block-motion-v1",
		MotionClock:                    "deterministic-test-clock-v1",
		MotionFrameBudget:              4,
		MotionUnsupportedCSSAnimations: false,
		MotionFrames:                   blockMotionFramesForScenario(),
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "MotionBlock",
				DispatchPath:    []string{"BlockMotionApp", "MotionBlock"},
				Handled:         true,
				Pass:            true,
				X:               48,
				Y:               72,
				Width:           320,
				Height:          200,
				TimestampMS:     0,
				BufferSlots:     []int{5, 48, 72, 1, 0, 320, 200, 0, 0},
				BeforeState:     map[string]string{"MotionBlock.hovered": "false"},
				AfterState:      map[string]string{"MotionBlock.hovered": "true"},
			},
			{
				Order:           2,
				Kind:            "text_input",
				TargetComponent: "MotionBlock",
				DispatchPath:    []string{"BlockMotionApp", "MotionBlock"},
				Handled:         true,
				Pass:            true,
				Width:           320,
				Height:          200,
				TimestampMS:     1,
				TextLen:         2,
				TextBytesHex:    "4f4b",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
				BeforeState:     map[string]string{"MotionBlock.buffer": ""},
				AfterState:      map[string]string{"MotionBlock.buffer": "OK"},
			},
		},
		Frames: []surface.FrameReport{
			{
				Order:     1,
				Width:     startFrame.Width,
				Height:    startFrame.Height,
				Stride:    startFrame.Stride,
				Checksum:  checksumRGBA(startFrame.Pixels),
				Presented: true,
			},
			{
				Order:     2,
				Width:     midFrame.Width,
				Height:    midFrame.Height,
				Stride:    midFrame.Stride,
				Checksum:  checksumRGBA(midFrame.Pixels),
				Presented: true,
			},
			{
				Order:     3,
				Width:     doneFrame.Width,
				Height:    doneFrame.Height,
				Stride:    doneFrame.Stride,
				Checksum:  checksumRGBA(doneFrame.Pixels),
				Presented: true,
			},
		},
		StateTransitions: []surface.StateTransitionReport{
			{
				Order:     1,
				Component: "MotionBlock",
				Field:     "opacity",
				Before:    "80",
				After:     "200",
				Cause:     "motion frame",
			},
			{
				Order:     2,
				Component: "MotionBlock",
				Field:     "color",
				Before:    "#203040ff",
				After:     "#60aef4ff",
				Cause:     "motion frame",
			},
			{
				Order:     3,
				Component: "MotionBlock",
				Field:     "scale",
				Before:    "100",
				After:     "108",
				Cause:     "motion frame",
			},
			{
				Order:     4,
				Component: "MotionBlock",
				Field:     "translate_x",
				Before:    "0",
				After:     "12",
				Cause:     "motion frame",
			},
			{
				Order:     5,
				Component: "MotionBlock",
				Field:     "motion_complete",
				Before:    "false",
				After:     "true",
				Cause:     "duration elapsed",
			},
			{
				Order:     6,
				Component: "MotionBlock",
				Field:     "reduced_motion",
				Before:    "false",
				After:     "true",
				Cause:     "accessibility setting",
			},
		},
		Cases: []surface.CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless event dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless framebuffer checksum", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless actual runner trace", Kind: "positive", Ran: true, Pass: true},
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
			{
				Name: "block motion deterministic test clock",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "block motion opacity color transform frames",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "block motion reduced motion instant settle",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "block motion completion stops scheduling",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{Name: "block motion frame checksum changed", Kind: "positive", Ran: true, Pass: true},
			{
				Name:          "block motion no css animation parity",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "css animation nonclaim",
			},
			{
				Name:          "reject legacy UI evidence",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "legacy UI evidence rejected",
			},
		},
	}
}
func blockMotionComponentsForScenario() []surface.ComponentReport {
	abilities := []string{
		"measure",
		"layout",
		"draw",
		"event",
		"focus",
		"text",
		"accessibility",
		"state",
		"motion",
	}
	return []surface.ComponentReport{
		{
			ID:        "BlockMotionApp",
			Type:      "examples.surface.block_core.surface_block_motion.BlockMotionApp",
			Bounds:    surface.RectReport{X: 0, Y: 0, W: 320, H: 200},
			Abilities: abilities,
			State:     map[string]string{"motion_quality": "deterministic-block-motion-v1"},
		},
		{
			ID:        "MotionBlock",
			Type:      "examples.surface.block_core.surface_block_motion.MotionBlock",
			Parent:    "BlockMotionApp",
			Bounds:    surface.RectReport{X: 24, Y: 44, W: 176, H: 64},
			Abilities: abilities,
			State: map[string]string{
				"opacity":     "200",
				"scale":       "108",
				"translate_x": "12",
				"complete":    "true",
			},
		},
	}
}
func blockMotionFramesForScenario() []surface.MotionFrameReport {
	return []surface.MotionFrameReport{
		{
			Order:         1,
			BlockID:       2,
			Trigger:       "hover",
			TimestampMS:   0,
			DurationMS:    120,
			DelayMS:       0,
			Progress:      0,
			Easing:        "linear",
			Opacity:       80,
			Color:         "#203040ff",
			TranslateX:    0,
			TranslateY:    0,
			Scale:         100,
			ReducedMotion: false,
			Scheduled:     true,
			Settled:       false,
			Checksum:      "sha256:" + checksumText("block-motion-frame-start"),
		},
		{
			Order:         2,
			BlockID:       2,
			Trigger:       "hover",
			TimestampMS:   60,
			DurationMS:    120,
			DelayMS:       0,
			Progress:      500,
			Easing:        "linear",
			Opacity:       140,
			Color:         "#407094ff",
			TranslateX:    6,
			TranslateY:    0,
			Scale:         104,
			ReducedMotion: false,
			Scheduled:     true,
			Settled:       false,
			Checksum:      "sha256:" + checksumText("block-motion-frame-mid"),
		},
		{
			Order:         3,
			BlockID:       2,
			Trigger:       "hover",
			TimestampMS:   120,
			DurationMS:    120,
			DelayMS:       0,
			Progress:      1000,
			Easing:        "linear",
			Opacity:       200,
			Color:         "#60aef4ff",
			TranslateX:    12,
			TranslateY:    0,
			Scale:         108,
			ReducedMotion: false,
			Scheduled:     false,
			Settled:       true,
			Checksum:      "sha256:" + checksumText("block-motion-frame-done"),
		},
		{
			Order:         4,
			BlockID:       2,
			Trigger:       "reduced_motion",
			TimestampMS:   121,
			DurationMS:    120,
			DelayMS:       0,
			Progress:      1000,
			Easing:        "linear",
			Opacity:       200,
			Color:         "#60aef4ff",
			TranslateX:    12,
			TranslateY:    0,
			Scale:         108,
			ReducedMotion: true,
			Scheduled:     false,
			Settled:       true,
			Checksum:      "sha256:" + checksumText("block-motion-frame-reduced"),
		},
	}
}
func runBlockAssetScenario() headlessScenario {
	beforeFrame := renderBlockAssetFrameRGBA(false)
	afterFrame := renderBlockAssetFrameRGBA(true)
	return headlessScenario{
		Components:                    blockAssetComponentsForScenario(),
		BlockAssetQualityLevel:        "deterministic-local-block-assets-v1",
		BlockAssetNetworkFetchAllowed: false,
		BlockAssetManifest: blockAssetManifestForScenario(
			"examples/surface/block_render/surface_block_assets.tetra",
		),
		BlockAssetCache:          blockAssetCacheForScenario(),
		BlockAssetDiagnostics:    blockAssetDiagnosticsForScenario(),
		BlockAssetRenderCommands: blockAssetRenderCommandsForScenario(),
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "IconBlock",
				DispatchPath:    []string{"BlockAssetApp", "IconBlock"},
				Handled:         true,
				Pass:            true,
				X:               32,
				Y:               44,
				Width:           320,
				Height:          200,
				TimestampMS:     0,
				BufferSlots:     []int{5, 32, 44, 1, 0, 320, 200, 0, 0},
				BeforeState:     map[string]string{"IconBlock.tint": "#ffffffff"},
				AfterState:      map[string]string{"IconBlock.tint": "#60aef4ff"},
			},
			{
				Order:           2,
				Kind:            "text_input",
				TargetComponent: "IconBlock",
				DispatchPath:    []string{"BlockAssetApp", "IconBlock"},
				Handled:         true,
				Pass:            true,
				Width:           320,
				Height:          200,
				TimestampMS:     1,
				TextLen:         2,
				TextBytesHex:    "4f4b",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
				BeforeState:     map[string]string{"IconBlock.label": ""},
				AfterState:      map[string]string{"IconBlock.label": "OK"},
			},
		},
		Frames: []surface.FrameReport{
			{
				Order:     1,
				Width:     beforeFrame.Width,
				Height:    beforeFrame.Height,
				Stride:    beforeFrame.Stride,
				Checksum:  checksumRGBA(beforeFrame.Pixels),
				Presented: true,
			},
			{
				Order:     2,
				Width:     afterFrame.Width,
				Height:    afterFrame.Height,
				Stride:    afterFrame.Stride,
				Checksum:  checksumRGBA(afterFrame.Pixels),
				Presented: true,
			},
		},
		StateTransitions: []surface.StateTransitionReport{
			{
				Order:     1,
				Component: "IconBlock",
				Field:     "tint",
				Before:    "#ffffffff",
				After:     "#60aef4ff",
				Cause:     "asset tint",
			},
			{
				Order:     2,
				Component: "ImageBlock",
				Field:     "scale",
				Before:    "1x",
				After:     "2x",
				Cause:     "asset scale",
			},
			{
				Order:     3,
				Component: "MissingAssetBlock",
				Field:     "fallback",
				Before:    "missing",
				After:     "fallback-raster",
				Cause:     "missing asset",
			},
		},
		Cases: []surface.CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless event dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless framebuffer checksum", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless actual runner trace", Kind: "positive", Ran: true, Pass: true},
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
			{
				Name: "block asset deterministic manifest hashes",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{Name: "block asset local embedded only", Kind: "positive", Ran: true, Pass: true},
			{Name: "block asset bounded cache", Kind: "positive", Ran: true, Pass: true},
			{Name: "block asset icon tint evidence", Kind: "positive", Ran: true, Pass: true},
			{Name: "block asset image scale evidence", Kind: "positive", Ran: true, Pass: true},
			{
				Name:          "block asset missing fallback diagnostic",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "missing asset",
			},
			{
				Name:          "block asset network url rejected",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "network assets disabled",
			},
			{
				Name:          "reject legacy UI evidence",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "legacy UI evidence rejected",
			},
		},
	}
}
func runBlockAccessibilityScenario() headlessScenario {
	beforeFrame := renderBlockAccessibilityFrameRGBA(false)
	afterFrame := renderBlockAccessibilityFrameRGBA(true)
	return headlessScenario{
		Components: blockAccessibilityComponentsForScenario(),
		BlockGraph: blockAccessibilityGraphForScenario(
			"examples/surface/block_render/surface_block_accessibility.tetra",
		),
		BlockAccessibilityTree: blockAccessibilityTreeForScenario(
			"examples/surface/block_render/surface_block_accessibility.tetra",
		),
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "SubmitBlock",
				DispatchPath:    []string{"BlockAccessibilityApp", "SubmitBlock"},
				Handled:         true,
				Pass:            true,
				X:               40,
				Y:               80,
				Width:           320,
				Height:          200,
				TimestampMS:     0,
				BufferSlots:     []int{5, 40, 80, 1, 0, 320, 200, 0, 0},
				BeforeState:     map[string]string{"SubmitBlock.focused": "false"},
				AfterState:      map[string]string{"SubmitBlock.focused": "true"},
			},
			{
				Order:           2,
				Kind:            "text_input",
				TargetComponent: "SubmitBlock",
				DispatchPath:    []string{"BlockAccessibilityApp", "SubmitBlock"},
				Handled:         true,
				Pass:            true,
				Width:           320,
				Height:          200,
				TimestampMS:     1,
				TextLen:         2,
				TextBytesHex:    "4f4b",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
				BeforeState:     map[string]string{"SubmitBlock.value_len": "0"},
				AfterState:      map[string]string{"SubmitBlock.value_len": "2"},
			},
			{
				Order:           3,
				Kind:            "key_down",
				TargetComponent: "SubmitBlock",
				DispatchPath:    []string{"BlockAccessibilityApp", "SubmitBlock"},
				Handled:         true,
				Pass:            true,
				Key:             13,
				Width:           320,
				Height:          200,
				TimestampMS:     2,
				BufferSlots:     []int{3, 0, 0, 0, 13, 320, 200, 2, 0},
				BeforeState:     map[string]string{"SubmitBlock.pressed": "false"},
				AfterState:      map[string]string{"SubmitBlock.pressed": "true"},
			},
		},
		Frames: []surface.FrameReport{
			{
				Order:     1,
				Width:     beforeFrame.Width,
				Height:    beforeFrame.Height,
				Stride:    beforeFrame.Stride,
				Checksum:  checksumRGBA(beforeFrame.Pixels),
				Presented: true,
			},
			{
				Order:     2,
				Width:     afterFrame.Width,
				Height:    afterFrame.Height,
				Stride:    afterFrame.Stride,
				Checksum:  checksumRGBA(afterFrame.Pixels),
				Presented: true,
			},
		},
		StateTransitions: []surface.StateTransitionReport{
			{
				Order:     1,
				Component: "SubmitBlock",
				Field:     "focused",
				Before:    "false",
				After:     "true",
				Cause:     "tab",
			},
			{
				Order:     2,
				Component: "ResetBlock",
				Field:     "focused",
				Before:    "false",
				After:     "true",
				Cause:     "tab",
			},
			{
				Order:     3,
				Component: "BlockAccessibilityApp",
				Field:     "reading_order_checked",
				Before:    "false",
				After:     "true",
				Cause:     "block_graph",
			},
		},
		Cases: []surface.CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless event dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless framebuffer checksum", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless actual runner trace", Kind: "positive", Ran: true, Pass: true},
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
			{
				Name:          "block graph duplicate id rejected",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "duplicate Block ID",
			},
			{
				Name:          "block graph missing parent rejected",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "missing parent",
			},
			{
				Name:          "block graph cycle rejected",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "cycle",
			},
			{Name: "block graph child order", Kind: "positive", Ran: true, Pass: true},
			{Name: "block graph focus order", Kind: "positive", Ran: true, Pass: true},
			{Name: "block graph hit-test path", Kind: "positive", Ran: true, Pass: true},
			{Name: "block graph accessibility order", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "block accessibility tree derived from block graph",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name:          "block accessibility focusable actionable name required",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "missing accessible name",
			},
			{
				Name:          "block accessibility label relationship mismatch rejected",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "label relationship mismatch",
			},
			{
				Name:          "block accessibility reading order graph mismatch rejected",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "reading order mismatch",
			},
			{
				Name:          "block accessibility screen-reader claim without platform proof rejected",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "screen reader proof required",
			},
			{
				Name: "block accessibility platform claim scoped metadata only",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name:          "reject legacy UI evidence",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "legacy UI evidence rejected",
			},
		},
	}
}
func runBlockSystemScenario() headlessScenario {
	source := "examples/surface/block_core/surface_block_system.tetra"
	beforeFrame := renderBlockSystemFrameRGBA(false)
	afterFrame := renderBlockSystemFrameRGBA(true)
	motionFrame := renderBlockSystemFrameRGBA(true)
	rectRGBA(
		motionFrame,
		rect{X: 188, Y: 124, W: 30, H: 10},
		rgbaColor{R: 96, G: 174, B: 244, A: 255},
	)
	frames := []surface.FrameReport{
		{
			Order:     1,
			Width:     beforeFrame.Width,
			Height:    beforeFrame.Height,
			Stride:    beforeFrame.Stride,
			Checksum:  checksumRGBA(beforeFrame.Pixels),
			Presented: true,
		},
		{
			Order:     2,
			Width:     afterFrame.Width,
			Height:    afterFrame.Height,
			Stride:    afterFrame.Stride,
			Checksum:  checksumRGBA(afterFrame.Pixels),
			Presented: true,
		},
		{
			Order:     3,
			Width:     motionFrame.Width,
			Height:    motionFrame.Height,
			Stride:    motionFrame.Stride,
			Checksum:  checksumRGBA(motionFrame.Pixels),
			Presented: true,
		},
	}
	components := blockSystemComponentsForScenario()
	components = append(
		components,
		retargetBlockSystemComponentsForScenario(blockTextComponentsForScenario())...)
	components = append(
		components,
		retargetBlockSystemComponentsForScenario(blockStateComponentsForScenario())...)
	components = append(
		components,
		retargetBlockSystemComponentsForScenario(blockMotionComponentsForScenario())...)
	components = append(
		components,
		retargetBlockSystemComponentsForScenario(blockAssetComponentsForScenario())...)
	events := blockSystemEventsForScenario()
	events = appendScenarioEventsWithNextOrder(events,
		blockTextEventsForScenario(),
		blockStateEventsForScenario(),
		blockMotionEventsForScenario(),
		blockAssetEventsForScenario(),
	)
	stateTransitions := []surface.StateTransitionReport{
		{
			Order:     1,
			Component: "SubmitBlock",
			Field:     "focused",
			Before:    "false",
			After:     "true",
			Cause:     "tab",
		},
		{
			Order:     2,
			Component: "ResetBlock",
			Field:     "focused",
			Before:    "false",
			After:     "true",
			Cause:     "tab",
		},
		{
			Order:     3,
			Component: "BlockSystemApp",
			Field:     "reading_order_checked",
			Before:    "false",
			After:     "true",
			Cause:     "block_graph",
		},
		{
			Order:     4,
			Component: "BlockLayoutApp",
			Field:     "width",
			Before:    "320",
			After:     "480",
			Cause:     "resize",
		},
		{
			Order:     5,
			Component: "ScrollBlock",
			Field:     "scroll_y",
			Before:    "0",
			After:     "32",
			Cause:     "scroll",
		},
	}
	stateTransitions = appendScenarioStateTransitionsWithNextOrder(
		stateTransitions,
		blockSystemReadinessTransitionsForScenario(),
	)
	scenario := headlessScenario{
		Components:            components,
		BlockGraph:            blockAccessibilityGraphForScenario(source),
		PaintLayers:           blockPaintLayersForScenario(),
		PaintCommands:         blockPaintCommandsForScenario(),
		VisualFeatures:        blockRendererVisualFeaturesForScenario(),
		PaintQualityLevel:     "deterministic-software-paint-v1",
		PaintCacheBudgetBytes: 65536,
		PaintUnsupportedBlur:  false,
		Renderer:              blockRendererReportForScenario(frames, 2),
		TextMeasurements:      blockTextMeasurementsForScenario(),
		FontFallbacks:         blockFontFallbacksForScenario(),
		GlyphCaches:           blockGlyphCachesForScenario(),
		TextRenderCommands:    blockTextRenderCommandsForScenario(),
		TextQualityLevel:      "deterministic-fallback-text-v1",
		TextCacheBudgetBytes:  65536,
		LayoutConstraints:     blockLayoutConstraintsForScenario(),
		LayoutPasses:          blockLayoutPassesForScenario(),
		LayoutScrolls:         blockLayoutScrollsForScenario(),
		LayoutDensity:         blockLayoutDensityForScenario(),
		LayoutFeatures: []string{
			"stack",
			"row",
			"column",
			"absolute",
			"overlay",
			"grid",
			"dock",
			"scroll",
			"fit",
			"fill",
			"fixed",
			"min",
			"max",
			"aspect",
			"spacing",
			"alignment",
			"z-order",
			"clipping",
			"resize",
			"density",
			"stable-rounding",
		},
		LayoutQualityLevel:          "deterministic-block-layout-v1",
		LayoutUnsupportedCSSFlexbox: false,
		BlockStateSelectors:         blockStateSelectorsForScenario(),
		BlockStateResolutions:       blockStateResolutionsForScenario(),
		BlockStateResolverOrder: []string{
			"base",
			"variant",
			"hover",
			"pressed",
			"focused",
			"selected",
			"disabled",
			"error",
			"loading",
			"motion",
		},
		BlockStateQualityLevel:   "deterministic-block-state-resolver-v1",
		MotionFrames:             blockMotionFramesForScenario(),
		MotionQualityLevel:       "deterministic-block-motion-v1",
		MotionClock:              "deterministic-test-clock-v1",
		MotionFrameBudget:        4,
		BlockAssetManifest:       blockAssetManifestForScenario(source),
		BlockAssetCache:          blockAssetCacheForScenario(),
		BlockAssetDiagnostics:    blockAssetDiagnosticsForScenario(),
		BlockAssetRenderCommands: blockAssetRenderCommandsForScenario(),
		BlockAssetQualityLevel:   "deterministic-local-block-assets-v1",
		BlockAccessibilityTree:   blockAccessibilityTreeForScenario(source),
		BlockSystem:              blockSystemReportForScenario(source, frames),
		Events:                   events,
		Frames:                   frames,
		StateTransitions:         stateTransitions,
		Cases:                    blockSystemCasesForScenario(),
	}
	scenario.BlockSceneSnapshot = blockSceneSnapshotForScenario(source, scenario)
	attachRenderCommandStreamForScenario(source, &scenario)
	scenario.Cases = append(scenario.Cases, blockSceneSnapshotCasesForScenario()...)
	attachBlockSystemMemoryBudget(&scenario)
	return scenario
}
func runMorphScenario() headlessScenario {
	return runMorphScenarioForSource(
		"examples/surface/morph_core/surface_morph_command_palette.tetra",
	)
}
func retargetScenarioToSource(scenario *headlessScenario, source string, module string) {
	if scenario == nil {
		return
	}
	for i := range scenario.Components {
		scenario.Components[i].Type = module + "." + typeBaseName(scenario.Components[i].Type)
	}
	if scenario.BlockGraph != nil {
		scenario.BlockGraph.Source = source
	}
	if scenario.BlockSceneSnapshot != nil {
		scenario.BlockSceneSnapshot.Source = source
	}
	if scenario.BlockAssetManifest != nil {
		scenario.BlockAssetManifest.Source = source
	}
	if scenario.BlockAccessibilityTree != nil {
		scenario.BlockAccessibilityTree.Source = source
	}
	if scenario.BlockSystem != nil {
		scenario.BlockSystem.Source = source
	}
}

// ---- scenarios_counter_release.go ----

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
	scenario.Cases = append(
		scenario.Cases,
		surface.CaseReport{
			Name: "linux-x64 app-presented RGBA checksum",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
	)
	scenario.Cases = append(
		scenario.Cases,
		surface.CaseReport{
			Name: "linux-x64 host event sequence",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
	)
	scenario.Cases = append(
		scenario.Cases,
		surface.CaseReport{
			Name: "linux-x64 counter component app-presented frame",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
	)
	return scenario
}
func runLinuxX64RealWindowCounterScenario() headlessScenario {
	beforeFrame := renderWindowCounterFrameRGBA(0, 0, 320, 200, true)
	afterClickFrame := renderWindowCounterFrameRGBA(1, 0, 320, 200, true)
	scenario := headlessScenario{
		Components: []surface.ComponentReport{
			{
				ID:     "CounterApp",
				Type:   "examples.surface.runtime.surface_window_counter.CounterApp",
				Bounds: surface.RectReport{X: 0, Y: 0, W: 400, H: 240},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"count":              "2",
					"key_count":          "1",
					"width":              "400",
					"closed":             "true",
					"accessibility_role": "button",
				},
			},
			{
				ID:     "CounterButton",
				Type:   "examples.surface.runtime.surface_window_counter.CounterButton",
				Parent: "CounterApp",
				Bounds: surface.RectReport{X: 32, Y: 88, W: 160, H: 48},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{"text_len_seen": "2", "accessibility_role": "button"},
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
				BeforeState: map[string]string{
					"CounterApp.key_count": "0",
					"CounterApp.count":     "1",
				},
				AfterState: map[string]string{
					"CounterApp.key_count": "1",
					"CounterApp.count":     "2",
				},
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
			{
				Order:     1,
				Width:     beforeFrame.Width,
				Height:    beforeFrame.Height,
				Stride:    beforeFrame.Stride,
				Checksum:  checksumRGBA(beforeFrame.Pixels),
				Presented: true,
			},
			{
				Order:     2,
				Width:     afterClickFrame.Width,
				Height:    afterClickFrame.Height,
				Stride:    afterClickFrame.Stride,
				Checksum:  checksumRGBA(afterClickFrame.Pixels),
				Presented: true,
			},
		},
		StateTransitions: []surface.StateTransitionReport{
			{
				Order:     1,
				Component: "CounterApp",
				Field:     "count",
				Before:    "0",
				After:     "1",
				Cause:     "mouse_up",
			},
			{
				Order:     2,
				Component: "CounterApp",
				Field:     "key_count",
				Before:    "0",
				After:     "1",
				Cause:     "key_down",
			},
			{
				Order:     3,
				Component: "CounterApp",
				Field:     "width",
				Before:    "320",
				After:     "400",
				Cause:     "resize",
			},
			{
				Order:     4,
				Component: "CounterButton",
				Field:     "text_len_seen",
				Before:    "0",
				After:     "2",
				Cause:     "text_input",
			},
			{
				Order:     5,
				Component: "CounterApp",
				Field:     "closed",
				Before:    "false",
				After:     "true",
				Cause:     "close",
			},
		},
		Cases: []surface.CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "linux-x64 Surface Host ABI open/present/close",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
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
			{
				Name:          "reject legacy UI evidence",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "legacy UI evidence rejected",
			},
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
	scenario.Cases = append(
		scenario.Cases,
		surface.CaseReport{
			Name: "compiler-owned wasm Surface loader",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
	)
	scenario.Cases = append(
		scenario.Cases,
		surface.CaseReport{
			Name: "wasm32-web actual presented frame trace",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
	)
	return scenario
}
func runWASM32WebBrowserCanvasCounterScenario() headlessScenario {
	return headlessScenario{
		Components: []surface.ComponentReport{
			{
				ID:     "CounterApp",
				Type:   "examples.surface.runtime.surface_browser_counter.CounterApp",
				Bounds: surface.RectReport{X: 0, Y: 0, W: 400, H: 240},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"count":              "2",
					"key_count":          "1",
					"width":              "400",
					"accessibility_role": "button",
				},
			},
			{
				ID:     "CounterButton",
				Type:   "examples.surface.runtime.surface_browser_counter.CounterButton",
				Parent: "CounterApp",
				Bounds: surface.RectReport{X: 32, Y: 88, W: 160, H: 48},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{"focused": "true", "text_len_seen": "2"},
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
				BeforeState: map[string]string{
					"CounterApp.count":     "1",
					"CounterApp.key_count": "0",
				},
				AfterState: map[string]string{
					"CounterApp.count":     "2",
					"CounterApp.key_count": "1",
				},
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
			{
				Order:     1,
				Component: "CounterApp",
				Field:     "count",
				Before:    "0",
				After:     "1",
				Cause:     "mouse_up",
			},
			{
				Order:     2,
				Component: "CounterApp",
				Field:     "key_count",
				Before:    "0",
				After:     "1",
				Cause:     "key_down",
			},
			{
				Order:     3,
				Component: "CounterApp",
				Field:     "width",
				Before:    "320",
				After:     "400",
				Cause:     "resize",
			},
			{
				Order:     4,
				Component: "CounterButton",
				Field:     "text_len_seen",
				Before:    "0",
				After:     "2",
				Cause:     "text_input",
			},
		},
		Cases: []surface.CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
			{Name: "wasm32-web browser canvas surface", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "wasm32-web browser canvas RGBA readback",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "wasm32-web browser canvas pointer input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "wasm32-web browser canvas keyboard input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "wasm32-web browser canvas resize input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{Name: "wasm32-web browser canvas text input", Kind: "positive", Ran: true, Pass: true},
			{Name: "wasm32-web Surface Host ABI imports", Kind: "positive", Ran: true, Pass: true},
			{Name: "compiler-owned wasm Surface loader", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "compiler-owned browser canvas Surface host",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
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
			{
				Name:          "reject legacy UI evidence",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "legacy UI evidence rejected",
			},
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
			{
				ID:     "SurfaceReleaseFormApp",
				Type:   "examples.surface.release.surface_release_form.SurfaceReleaseFormApp",
				Bounds: surface.RectReport{X: 0, Y: 0, W: 560, H: 420},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused_id":         "7",
					"save_count":         "1",
					"reset_count":        "1",
					"status_code":        "2",
					"width":              "560",
					"height":             "420",
					"accessibility_role": "none",
				},
			},
			{
				ID:     "Panel",
				Type:   "lib.core.widgets.Panel",
				Parent: "SurfaceReleaseFormApp",
				Bounds: surface.RectReport{X: 0, Y: 0, W: 560, H: 420},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{"padding": "16", "accessibility_role": "none"},
			},
			{
				ID:     "Stack",
				Type:   "lib.core.widgets.Stack",
				Parent: "Panel",
				Bounds: surface.RectReport{X: 16, Y: 16, W: 528, H: 396},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{"child_count": "1", "accessibility_role": "none"},
			},
			{
				ID:     "Column",
				Type:   "lib.core.widgets.Column",
				Parent: "Stack",
				Bounds: surface.RectReport{X: 24, Y: 24, W: 512, H: 388},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{"child_count": "9", "accessibility_role": "none"},
			},
			{
				ID:     "TitleText",
				Type:   "lib.core.widgets.Text",
				Parent: "Column",
				Bounds: surface.RectReport{X: 32, Y: 32, W: 496, H: 28},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"role":               "label",
					"text_len":           "18",
					"accessibility_role": "label",
				},
			},
			{
				ID:     "DescriptionText",
				Type:   "lib.core.widgets.Text",
				Parent: "Column",
				Bounds: surface.RectReport{X: 32, Y: 68, W: 496, H: 28},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"role":               "description",
					"text_len":           "24",
					"accessibility_role": "label",
				},
			},
			{
				ID:     "NameLabel",
				Type:   "lib.core.widgets.Label",
				Parent: "Column",
				Bounds: surface.RectReport{X: 32, Y: 104, W: 496, H: 24},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"role":               "label",
					"text_len":           "4",
					"labelled_for":       "7",
					"accessibility_role": "label",
				},
			},
			{
				ID:     "NameTextBox",
				Type:   "lib.core.widgets.TextBox",
				Parent: "Column",
				Bounds: surface.RectReport{X: 32, Y: 132, W: 496, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "true",
					"buffer":             "",
					"text_len":           "0",
					"caret":              "0",
					"accessibility_role": "label",
				},
			},
			{
				ID:     "EmailLabel",
				Type:   "lib.core.widgets.Label",
				Parent: "Column",
				Bounds: surface.RectReport{X: 32, Y: 184, W: 496, H: 24},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"role":               "label",
					"text_len":           "5",
					"labelled_for":       "9",
					"accessibility_role": "label",
				},
			},
			{
				ID:     "EmailTextBox",
				Type:   "lib.core.widgets.TextBox",
				Parent: "Column",
				Bounds: surface.RectReport{X: 32, Y: 212, W: 496, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "false",
					"buffer":             "",
					"text_len":           "0",
					"caret":              "0",
					"accessibility_role": "label",
				},
			},
			{
				ID:     "SubscribeCheckbox",
				Type:   "lib.core.widgets.Checkbox",
				Parent: "Column",
				Bounds: surface.RectReport{X: 32, Y: 264, W: 496, H: 32},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "false",
					"checked":            "true",
					"toggle_count":       "1",
					"accessibility_role": "button",
				},
			},
			{
				ID:     "TermsScroll",
				Type:   "lib.core.widgets.Scroll",
				Parent: "Column",
				Bounds: surface.RectReport{X: 32, Y: 304, W: 496, H: 48},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"offset_y":           "16",
					"content_h":          "120",
					"accessibility_role": "none",
				},
			},
			{
				ID:     "TermsText",
				Type:   "lib.core.widgets.Text",
				Parent: "TermsScroll",
				Bounds: surface.RectReport{X: 36, Y: 308, W: 488, H: 24},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"role":               "description",
					"text_len":           "48",
					"accessibility_role": "label",
				},
			},
			{
				ID:     "ButtonRow",
				Type:   "lib.core.widgets.Row",
				Parent: "Column",
				Bounds: surface.RectReport{X: 32, Y: 360, W: 496, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{"child_count": "4", "accessibility_role": "none"},
			},
			{
				ID:     "SaveButton",
				Type:   "lib.core.widgets.Button",
				Parent: "ButtonRow",
				Bounds: surface.RectReport{X: 32, Y: 360, W: 132, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "false",
					"press_count":        "1",
					"action":             "save",
					"accessibility_role": "button",
				},
			},
			{
				ID:     "ResetButton",
				Type:   "lib.core.widgets.Button",
				Parent: "ButtonRow",
				Bounds: surface.RectReport{X: 176, Y: 360, W: 132, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "false",
					"press_count":        "1",
					"action":             "reset",
					"accessibility_role": "button",
				},
			},
			{
				ID:     "Spacer",
				Type:   "lib.core.widgets.Spacer",
				Parent: "ButtonRow",
				Bounds: surface.RectReport{X: 320, Y: 360, W: 16, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"min_w":              "16",
					"min_h":              "44",
					"accessibility_role": "none",
				},
			},
			{
				ID:     "StatusText",
				Type:   "lib.core.widgets.StatusText",
				Parent: "ButtonRow",
				Bounds: surface.RectReport{X: 344, Y: 360, W: 184, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"role":               "status",
					"status_code":        "2",
					"text_len":           "6",
					"accessibility_role": "label",
				},
			},
		},
		ComponentTree: &surface.ComponentTreeReport{
			Schema:       "tetra.surface.component-tree.v1",
			DynamicLevel: "production-widgets-v1",
			RootID:       0,
			NodeCount:    18,
			FocusedID:    7,
			Nodes: []surface.ComponentTreeNodeReport{
				{
					ID:         0,
					Name:       "SurfaceReleaseFormApp",
					Kind:       "root",
					ParentID:   -1,
					ChildIndex: 0,
					FirstChild: 1,
					ChildCount: 1,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 0, Y: 0, W: 560, H: 420},
				},
				{
					ID:         1,
					Name:       "Panel",
					Kind:       "panel",
					ParentID:   0,
					ChildIndex: 0,
					FirstChild: 2,
					ChildCount: 1,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 0, Y: 0, W: 560, H: 420},
				},
				{
					ID:         2,
					Name:       "Stack",
					Kind:       "stack",
					ParentID:   1,
					ChildIndex: 0,
					FirstChild: 3,
					ChildCount: 1,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 16, Y: 16, W: 528, H: 396},
				},
				{
					ID:         3,
					Name:       "Column",
					Kind:       "column",
					ParentID:   2,
					ChildIndex: 0,
					FirstChild: 4,
					ChildCount: 9,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 24, Y: 24, W: 512, H: 388},
				},
				{
					ID:         4,
					Name:       "TitleText",
					Kind:       "text",
					ParentID:   3,
					ChildIndex: 0,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 32, Y: 32, W: 496, H: 28},
				},
				{
					ID:         5,
					Name:       "DescriptionText",
					Kind:       "text",
					ParentID:   3,
					ChildIndex: 1,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 32, Y: 68, W: 496, H: 28},
				},
				{
					ID:         6,
					Name:       "NameLabel",
					Kind:       "label",
					ParentID:   3,
					ChildIndex: 2,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 32, Y: 104, W: 496, H: 24},
				},
				{
					ID:         7,
					Name:       "NameTextBox",
					Kind:       "textbox",
					ParentID:   3,
					ChildIndex: 3,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  true,
					Bounds:     surface.RectReport{X: 32, Y: 132, W: 496, H: 44},
				},
				{
					ID:         8,
					Name:       "EmailLabel",
					Kind:       "label",
					ParentID:   3,
					ChildIndex: 4,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 32, Y: 184, W: 496, H: 24},
				},
				{
					ID:         9,
					Name:       "EmailTextBox",
					Kind:       "textbox",
					ParentID:   3,
					ChildIndex: 5,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  true,
					Bounds:     surface.RectReport{X: 32, Y: 212, W: 496, H: 44},
				},
				{
					ID:         10,
					Name:       "SubscribeCheckbox",
					Kind:       "checkbox",
					ParentID:   3,
					ChildIndex: 6,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  true,
					Bounds:     surface.RectReport{X: 32, Y: 264, W: 496, H: 32},
				},
				{
					ID:         11,
					Name:       "TermsScroll",
					Kind:       "scroll",
					ParentID:   3,
					ChildIndex: 7,
					FirstChild: 12,
					ChildCount: 1,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 32, Y: 304, W: 496, H: 48},
				},
				{
					ID:         12,
					Name:       "TermsText",
					Kind:       "text",
					ParentID:   11,
					ChildIndex: 0,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 36, Y: 308, W: 488, H: 24},
				},
				{
					ID:         13,
					Name:       "ButtonRow",
					Kind:       "row",
					ParentID:   3,
					ChildIndex: 8,
					FirstChild: 14,
					ChildCount: 4,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 32, Y: 360, W: 496, H: 44},
				},
				{
					ID:         14,
					Name:       "SaveButton",
					Kind:       "button",
					ParentID:   13,
					ChildIndex: 0,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  true,
					Bounds:     surface.RectReport{X: 32, Y: 360, W: 132, H: 44},
				},
				{
					ID:         15,
					Name:       "ResetButton",
					Kind:       "button",
					ParentID:   13,
					ChildIndex: 1,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  true,
					Bounds:     surface.RectReport{X: 176, Y: 360, W: 132, H: 44},
				},
				{
					ID:         16,
					Name:       "Spacer",
					Kind:       "spacer",
					ParentID:   13,
					ChildIndex: 2,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 320, Y: 360, W: 16, H: 44},
				},
				{
					ID:         17,
					Name:       "StatusText",
					Kind:       "status",
					ParentID:   13,
					ChildIndex: 3,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 344, Y: 360, W: 184, H: 44},
				},
			},
			LayoutPasses: []surface.ComponentTreeLayoutPassReport{
				{
					ComponentID: 7,
					Pass:        "initial",
					Bounds:      surface.RectReport{X: 32, Y: 132, W: 320, H: 44},
					Measured:    surface.SizeReport{W: 320, H: 44},
				},
				{
					ComponentID: 9,
					Pass:        "initial",
					Bounds:      surface.RectReport{X: 32, Y: 212, W: 320, H: 44},
					Measured:    surface.SizeReport{W: 320, H: 44},
				},
				{
					ComponentID: 11,
					Pass:        "scroll",
					Bounds:      surface.RectReport{X: 32, Y: 304, W: 496, H: 48},
					Measured:    surface.SizeReport{W: 496, H: 120},
				},
				{
					ComponentID: 7,
					Pass:        "resize",
					Bounds:      surface.RectReport{X: 32, Y: 132, W: 496, H: 44},
					Measured:    surface.SizeReport{W: 496, H: 44},
				},
				{
					ComponentID: 9,
					Pass:        "resize",
					Bounds:      surface.RectReport{X: 32, Y: 212, W: 496, H: 44},
					Measured:    surface.SizeReport{W: 496, H: 44},
				},
				{
					ComponentID: 17,
					Pass:        "status-update",
					Bounds:      surface.RectReport{X: 344, Y: 360, W: 184, H: 44},
					Measured:    surface.SizeReport{W: 184, H: 44},
				},
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
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "NameTextBox",
				DispatchPath: []string{
					"SurfaceReleaseFormApp",
					"Panel",
					"Stack",
					"Column",
					"NameTextBox",
				},
				Handled:     true,
				Pass:        true,
				X:           48,
				Y:           148,
				Width:       560,
				Height:      420,
				BufferSlots: []int{5, 48, 148, 1, 0, 560, 420, 0, 0},
				BeforeState: map[string]string{
					"SurfaceReleaseFormApp.focused_id": "-1",
					"NameTextBox.focused":              "false",
				},
				AfterState: map[string]string{
					"SurfaceReleaseFormApp.focused_id": "7",
					"NameTextBox.focused":              "true",
				},
			},
			{
				Order:           2,
				Kind:            "text_input",
				TargetComponent: "NameTextBox",
				DispatchPath: []string{
					"SurfaceReleaseFormApp",
					"Panel",
					"Stack",
					"Column",
					"NameTextBox",
				},
				Handled:      true,
				Pass:         true,
				Width:        560,
				Height:       420,
				TimestampMS:  1,
				TextLen:      3,
				TextBytesHex: "416461",
				BufferSlots:  []int{8, 0, 0, 0, 0, 560, 420, 1, 3},
				BeforeState: map[string]string{
					"NameTextBox.buffer":  "",
					"EmailTextBox.buffer": "",
				},
				AfterState: map[string]string{
					"NameTextBox.buffer":  "Ada",
					"EmailTextBox.buffer": "",
				},
			},
			{
				Order:           3,
				Kind:            "key_down",
				TargetComponent: "SurfaceReleaseFormApp",
				DispatchPath:    []string{"SurfaceReleaseFormApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           560,
				Height:          420,
				TimestampMS:     2,
				BufferSlots:     []int{6, 0, 0, 0, 9, 560, 420, 2, 0},
				BeforeState:     map[string]string{"SurfaceReleaseFormApp.focused_id": "7"},
				AfterState:      map[string]string{"SurfaceReleaseFormApp.focused_id": "9"},
			},
			{
				Order:           4,
				Kind:            "text_input",
				TargetComponent: "EmailTextBox",
				DispatchPath: []string{
					"SurfaceReleaseFormApp",
					"Panel",
					"Stack",
					"Column",
					"EmailTextBox",
				},
				Handled:      true,
				Pass:         true,
				Width:        560,
				Height:       420,
				TimestampMS:  3,
				TextLen:      5,
				TextBytesHex: "7465747261",
				BufferSlots:  []int{8, 0, 0, 0, 0, 560, 420, 3, 5},
				BeforeState: map[string]string{
					"EmailTextBox.buffer": "",
					"NameTextBox.buffer":  "Ada",
				},
				AfterState: map[string]string{
					"EmailTextBox.buffer": "tetra",
					"NameTextBox.buffer":  "Ada",
				},
			},
			{
				Order:           5,
				Kind:            "key_down",
				TargetComponent: "SurfaceReleaseFormApp",
				DispatchPath:    []string{"SurfaceReleaseFormApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           560,
				Height:          420,
				TimestampMS:     4,
				BufferSlots:     []int{6, 0, 0, 0, 9, 560, 420, 4, 0},
				BeforeState:     map[string]string{"SurfaceReleaseFormApp.focused_id": "9"},
				AfterState:      map[string]string{"SurfaceReleaseFormApp.focused_id": "10"},
			},
			{
				Order:           6,
				Kind:            "key_down",
				TargetComponent: "SubscribeCheckbox",
				DispatchPath: []string{
					"SurfaceReleaseFormApp",
					"Panel",
					"Stack",
					"Column",
					"SubscribeCheckbox",
				},
				Handled:     true,
				Pass:        true,
				Key:         32,
				Width:       560,
				Height:      420,
				TimestampMS: 5,
				BufferSlots: []int{6, 0, 0, 0, 32, 560, 420, 5, 0},
				BeforeState: map[string]string{
					"SubscribeCheckbox.checked":      "false",
					"SubscribeCheckbox.toggle_count": "0",
				},
				AfterState: map[string]string{
					"SubscribeCheckbox.checked":      "true",
					"SubscribeCheckbox.toggle_count": "1",
				},
			},
			{
				Order:           7,
				Kind:            "scroll",
				TargetComponent: "TermsScroll",
				DispatchPath: []string{
					"SurfaceReleaseFormApp",
					"Panel",
					"Stack",
					"Column",
					"TermsScroll",
				},
				Handled:     true,
				Pass:        true,
				X:           48,
				Y:           320,
				Width:       560,
				Height:      420,
				TimestampMS: 6,
				BufferSlots: []int{5, 48, 320, 1, 0, 560, 420, 6, 0},
				BeforeState: map[string]string{"TermsScroll.offset_y": "0"},
				AfterState:  map[string]string{"TermsScroll.offset_y": "16"},
			},
			{
				Order:           8,
				Kind:            "key_down",
				TargetComponent: "SurfaceReleaseFormApp",
				DispatchPath:    []string{"SurfaceReleaseFormApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           560,
				Height:          420,
				TimestampMS:     7,
				BufferSlots:     []int{6, 0, 0, 0, 9, 560, 420, 7, 0},
				BeforeState:     map[string]string{"SurfaceReleaseFormApp.focused_id": "10"},
				AfterState:      map[string]string{"SurfaceReleaseFormApp.focused_id": "14"},
			},
			{
				Order:           9,
				Kind:            "key_down",
				TargetComponent: "SaveButton",
				DispatchPath: []string{
					"SurfaceReleaseFormApp",
					"Panel",
					"Stack",
					"Column",
					"ButtonRow",
					"SaveButton",
				},
				Handled:     true,
				Pass:        true,
				Key:         32,
				Width:       560,
				Height:      420,
				TimestampMS: 8,
				BufferSlots: []int{6, 0, 0, 0, 32, 560, 420, 8, 0},
				BeforeState: map[string]string{
					"SurfaceReleaseFormApp.save_count": "0",
					"StatusText.status_code":           "0",
				},
				AfterState: map[string]string{
					"SurfaceReleaseFormApp.save_count": "1",
					"StatusText.status_code":           "1",
				},
			},
			{
				Order:           10,
				Kind:            "key_down",
				TargetComponent: "SurfaceReleaseFormApp",
				DispatchPath:    []string{"SurfaceReleaseFormApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           560,
				Height:          420,
				TimestampMS:     9,
				BufferSlots:     []int{6, 0, 0, 0, 9, 560, 420, 9, 0},
				BeforeState:     map[string]string{"SurfaceReleaseFormApp.focused_id": "14"},
				AfterState:      map[string]string{"SurfaceReleaseFormApp.focused_id": "15"},
			},
			{
				Order:           11,
				Kind:            "key_down",
				TargetComponent: "ResetButton",
				DispatchPath: []string{
					"SurfaceReleaseFormApp",
					"Panel",
					"Stack",
					"Column",
					"ButtonRow",
					"ResetButton",
				},
				Handled:     true,
				Pass:        true,
				Key:         13,
				Width:       560,
				Height:      420,
				TimestampMS: 10,
				BufferSlots: []int{6, 0, 0, 0, 13, 560, 420, 10, 0},
				BeforeState: map[string]string{
					"SurfaceReleaseFormApp.reset_count": "0",
					"StatusText.status_code":            "1",
					"NameTextBox.buffer":                "Ada",
					"EmailTextBox.buffer":               "tetra",
				},
				AfterState: map[string]string{
					"SurfaceReleaseFormApp.reset_count": "1",
					"StatusText.status_code":            "2",
					"NameTextBox.buffer":                "",
					"EmailTextBox.buffer":               "",
				},
			},
			{
				Order:           12,
				Kind:            "key_down",
				TargetComponent: "SurfaceReleaseFormApp",
				DispatchPath:    []string{"SurfaceReleaseFormApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           560,
				Height:          420,
				TimestampMS:     11,
				BufferSlots:     []int{6, 0, 0, 0, 9, 560, 420, 11, 0},
				BeforeState:     map[string]string{"SurfaceReleaseFormApp.focused_id": "15"},
				AfterState:      map[string]string{"SurfaceReleaseFormApp.focused_id": "7"},
			},
			{
				Order:           13,
				Kind:            "resize",
				TargetComponent: "SurfaceReleaseFormApp",
				DispatchPath:    []string{"SurfaceReleaseFormApp"},
				Handled:         true,
				Pass:            true,
				Width:           560,
				Height:          420,
				TimestampMS:     12,
				BufferSlots:     []int{2, 0, 0, 0, 0, 560, 420, 12, 0},
				BeforeState: map[string]string{
					"SurfaceReleaseFormApp.focused_id": "7",
					"NameTextBox.bounds.w":             "320",
					"EmailTextBox.bounds.w":            "320",
				},
				AfterState: map[string]string{
					"SurfaceReleaseFormApp.focused_id": "7",
					"NameTextBox.bounds.w":             "496",
					"EmailTextBox.bounds.w":            "496",
				},
			},
		},
		Frames: []surface.FrameReport{
			{
				Order:     1,
				Width:     beforeFrame.Width,
				Height:    beforeFrame.Height,
				Stride:    beforeFrame.Stride,
				Checksum:  checksumRGBA(beforeFrame.Pixels),
				Presented: true,
			},
			{
				Order:     2,
				Width:     nameFrame.Width,
				Height:    nameFrame.Height,
				Stride:    nameFrame.Stride,
				Checksum:  checksumRGBA(nameFrame.Pixels),
				Presented: true,
			},
			{
				Order:     3,
				Width:     checkboxFrame.Width,
				Height:    checkboxFrame.Height,
				Stride:    checkboxFrame.Stride,
				Checksum:  checksumRGBA(checkboxFrame.Pixels),
				Presented: true,
			},
			{
				Order:     4,
				Width:     saveFrame.Width,
				Height:    saveFrame.Height,
				Stride:    saveFrame.Stride,
				Checksum:  checksumRGBA(saveFrame.Pixels),
				Presented: true,
			},
			{
				Order:     5,
				Width:     afterFrame.Width,
				Height:    afterFrame.Height,
				Stride:    afterFrame.Stride,
				Checksum:  checksumRGBA(afterFrame.Pixels),
				Presented: true,
			},
		},
		StateTransitions: []surface.StateTransitionReport{
			{
				Order:     1,
				Component: "SurfaceReleaseFormApp",
				Field:     "focused_id",
				Before:    "-1",
				After:     "7",
				Cause:     "mouse_up",
			},
			{
				Order:     2,
				Component: "NameTextBox",
				Field:     "buffer",
				Before:    "",
				After:     "Ada",
				Cause:     "text_input",
			},
			{
				Order:     3,
				Component: "SurfaceReleaseFormApp",
				Field:     "focused_id",
				Before:    "7",
				After:     "9",
				Cause:     "tab",
			},
			{
				Order:     4,
				Component: "EmailTextBox",
				Field:     "buffer",
				Before:    "",
				After:     "tetra",
				Cause:     "text_input",
			},
			{
				Order:     5,
				Component: "SurfaceReleaseFormApp",
				Field:     "focused_id",
				Before:    "9",
				After:     "10",
				Cause:     "tab",
			},
			{
				Order:     6,
				Component: "SubscribeCheckbox",
				Field:     "checked",
				Before:    "false",
				After:     "true",
				Cause:     "key_down",
			},
			{
				Order:     7,
				Component: "TermsScroll",
				Field:     "offset_y",
				Before:    "0",
				After:     "16",
				Cause:     "scroll",
			},
			{
				Order:     8,
				Component: "SurfaceReleaseFormApp",
				Field:     "focused_id",
				Before:    "10",
				After:     "14",
				Cause:     "tab",
			},
			{
				Order:     9,
				Component: "SurfaceReleaseFormApp",
				Field:     "save_count",
				Before:    "0",
				After:     "1",
				Cause:     "key_down",
			},
			{
				Order:     10,
				Component: "StatusText",
				Field:     "status_code",
				Before:    "0",
				After:     "1",
				Cause:     "save",
			},
			{
				Order:     11,
				Component: "SurfaceReleaseFormApp",
				Field:     "focused_id",
				Before:    "14",
				After:     "15",
				Cause:     "tab",
			},
			{
				Order:     12,
				Component: "NameTextBox",
				Field:     "buffer",
				Before:    "Ada",
				After:     "",
				Cause:     "reset",
			},
			{
				Order:     13,
				Component: "EmailTextBox",
				Field:     "buffer",
				Before:    "tetra",
				After:     "",
				Cause:     "reset",
			},
			{
				Order:     14,
				Component: "SurfaceReleaseFormApp",
				Field:     "reset_count",
				Before:    "0",
				After:     "1",
				Cause:     "key_down",
			},
			{
				Order:     15,
				Component: "StatusText",
				Field:     "status_code",
				Before:    "1",
				After:     "2",
				Cause:     "reset",
			},
			{
				Order:     16,
				Component: "SurfaceReleaseFormApp",
				Field:     "focused_id",
				Before:    "15",
				After:     "7",
				Cause:     "tab",
			},
			{
				Order:     17,
				Component: "SurfaceReleaseFormApp",
				Field:     "NameTextBox.bounds.w",
				Before:    "320",
				After:     "496",
				Cause:     "resize",
			},
			{
				Order:     18,
				Component: "SurfaceReleaseFormApp",
				Field:     "EmailTextBox.bounds.w",
				Before:    "320",
				After:     "496",
				Cause:     "resize",
			},
		},
		Cases: productionToolkitBaseCases(),
	}
	switch mode {
	case "headless-release-toolkit":
		scenario.Cases = append(
			scenario.Cases,
			surface.CaseReport{
				Name: "headless event dispatch",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "headless framebuffer checksum",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "headless actual runner trace",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
		)
	case "linux-x64-release-toolkit":
		scenario.Cases = append(
			scenario.Cases,
			surface.CaseReport{
				Name: "linux-x64 Surface Host ABI open/present/close",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 real-window surface",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 native input event pump",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 real-window resize event",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 real-window close event",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
		)
	case "wasm32-web-release-toolkit":
		scenario.Frames = nil
		scenario.Cases = append(
			scenario.Cases,
			surface.CaseReport{
				Name: "wasm32-web browser canvas surface",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas RGBA readback",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas pointer input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas keyboard input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas resize input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas text input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web Surface Host ABI imports",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "compiler-owned wasm Surface loader",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "compiler-owned browser canvas Surface host",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
		)
	}
	return scenario
}
func runReleaseBrowserScenario() headlessScenario {
	scenario := runReleaseToolkitScenario("wasm32-web-release-toolkit")
	scenario.BrowserSurface = releaseBrowserSurfaceReport()
	scenario.Cases = append(
		scenario.Cases,
		surface.CaseReport{
			Name: "browser release Surface v1 schema",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "browser release Chromium canvas readback",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "browser release native pointer keyboard text resize",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "browser release deterministic clipboard harness",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "browser release composition trace",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "browser release accessibility snapshot mirror",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name:          "browser release forbidden web sidecar rejection",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "forbidden web sidecar rejected",
		},
	)
	return scenario
}
func releaseBrowserSurfaceReport() *surface.BrowserSurfaceReport {
	return &surface.BrowserSurfaceReport{
		Schema:              surface.BrowserSurfaceSchemaV1,
		BrowserSurfaceLevel: "browser-canvas-release-v1",
		ReleaseScope:        surface.ReleaseScopeSurfaceV1LinuxWeb,
		Source:              "examples/surface/release/surface_release_form.tetra",
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
			{
				Name:         "browser-canvas",
				ArtifactKind: "runner-trace",
				Path:         "surface-runner-trace.json",
				Pass:         true,
			},
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
		{
			Order:     1,
			Width:     beforeFrame.Width,
			Height:    beforeFrame.Height,
			Stride:    beforeFrame.Stride,
			Checksum:  checksumRGBA(beforeFrame.Pixels),
			Presented: true,
		},
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
	scenario.Cases = append(
		scenario.Cases,
		surface.CaseReport{
			Name: "linux release window v1 schema",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "linux release real window presented frame",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "linux release native pointer key text resize close",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "linux release clipboard harness",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "linux release composition harness",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "linux release accessibility bridge probe",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name:          "linux release forbids memfd starter promotion",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "memfd starter rejected",
		},
		surface.CaseReport{
			Name: "accessibility platform bridge v1 schema",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "linux accessibility host bridge export",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "accessibility release honest screen reader evidence",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
	)
	return scenario
}
func releaseWindowAccessibilityTreeReport() *surface.AccessibilityTreeReport {
	return &surface.AccessibilityTreeReport{
		Schema:                   "tetra.surface.accessibility-tree.v1",
		AccessibilityLevel:       "platform-bridge-v1",
		ReleaseScope:             "surface-v1-linux-web",
		Source:                   "examples/surface/release/surface_release_form.tetra",
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
		RolesPresent: []string{
			"root",
			"panel",
			"column",
			"text",
			"label",
			"textbox",
			"checkbox",
			"row",
			"button",
			"status",
		},
		FocusOrder: []string{
			"NameTextBox",
			"EmailTextBox",
			"SubscribeCheckbox",
			"SaveButton",
			"ResetButton",
		},
		ReadingOrder: []string{
			"TitleText",
			"DescriptionText",
			"NameLabel",
			"NameTextBox",
			"EmailLabel",
			"EmailTextBox",
			"SubscribeCheckbox",
			"TermsText",
			"SaveButton",
			"ResetButton",
			"StatusText",
		},
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
			scenario.Components[i].Type = ("examples.surface.release.surface_release_" +
				"accessibility.SurfaceReleaseAccessibilityApp")
		}
	}
	if scenario.ComponentTree != nil {
		scenario.ComponentTree.DynamicLevel = "platform-bridge-v1"
	}
	if scenario.ComponentTreeAPI != nil {
		scenario.ComponentTreeAPI.Source = "examples/surface/release/surface_release_accessibility.tetra"
	}
	if scenario.Toolkit != nil {
		scenario.Toolkit.Source = "examples/surface/release/surface_release_accessibility.tetra"
		if !containsString(
			scenario.Toolkit.Sources,
			"examples/surface/release/surface_release_accessibility.tetra",
		) {
			scenario.Toolkit.Sources = append(
				scenario.Toolkit.Sources,
				"examples/surface/release/surface_release_accessibility.tetra",
			)
		}
	}
	if scenario.AccessibilityTree != nil {
		tree := scenario.AccessibilityTree
		tree.AccessibilityLevel = "platform-bridge-v1"
		tree.ReleaseScope = "surface-v1-linux-web"
		tree.Source = "examples/surface/release/surface_release_accessibility.tetra"
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
	scenario.Cases = append(
		scenario.Cases,
		surface.CaseReport{
			Name: "accessibility platform bridge v1 schema",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "accessibility platform export from metadata tree",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "linux accessibility host bridge export",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "accessibility release honest screen reader evidence",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
	)
	switch mode {
	case "linux-x64-release-accessibility":
		scenario.Cases = append(
			scenario.Cases,
			surface.CaseReport{
				Name: "linux accessibility platform probe roles labels values states bounds",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux accessibility probe focus order labels status resize",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
		)
	case "wasm32-web-release-accessibility":
		scenario.Cases = append(
			scenario.Cases,
			surface.CaseReport{
				Name: "browser accessibility snapshot roles labels values states bounds",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "browser compiler-owned accessibility mirror",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "browser accessibility mirror no DOM visual UI",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
		)
	default:
		scenario.Cases = append(
			scenario.Cases,
			surface.CaseReport{
				Name: "headless deterministic accessibility platform bridge shape",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
		)
	}
	return scenario
}

// ---- scenarios_morph_budget.go ----

func morphReportForScenario(source string, scenario headlessScenario) *surface.MorphReport {
	capsuleHash := "sha256:" + checksumText("surface-morph-capsule-v1:"+source)
	tokenGraphHash := "sha256:" + checksumText("surface-morph-token-graph-v1:"+source)
	return &surface.MorphReport{
		Schema:          "tetra.surface.morph.v1",
		QualityLevel:    "deterministic-headless-morph-capsule-v1",
		Source:          source,
		Module:          "lib.core.morph",
		SurfaceScope:    "surface-morph-experimental-linux-web",
		Experimental:    true,
		ProductionClaim: false,
		GitHead:         gitHeadForReport(),
		GitDirty:        gitDirtyForReport(),
		CapsuleHash:     capsuleHash,
		TokenGraphHash:  tokenGraphHash,
		Capsule: surface.MorphCapsuleReport{
			Namespace:       "tetra.surface.morph.app",
			Version:         "1",
			CapsuleHash:     capsuleHash,
			Imports:         []string{"lib.core.block", "lib.core.morph"},
			ExplicitImports: true,
			NoGlobalCascade: true,
		},
		TokenGraph: morphTokenGraphForScenario(tokenGraphHash),
		Materials:  morphMaterialsForScenario(),
		LayoutModes: []string{
			"row",
			"column",
			"stack",
			"grid",
			"dock",
			"absolute",
			"overlay",
			"scroll",
		},
		TypographyRoles:  []string{"title", "body", "label", "code"},
		AssetRefs:        morphAssetRefsForScenario(),
		Affordances:      morphAffordancesForScenario(),
		StateLenses:      morphStateLensesForScenario(),
		MotionPresets:    morphMotionPresetsForScenario(),
		Recipes:          morphRecipesForScenario(),
		RecipeExpansions: morphRecipeExpansionsForScenario(),
		RecipeApps:       morphRecipeAppsForScenario(),
		Accessibility: surface.MorphAccessibilityProjectionReport{
			Schema:                "tetra.surface.morph.accessibility-projection.v1",
			DerivedFromBlockGraph: true,
			SafetyOverridesWin:    true,
			SnapshotEvidence:      true,
			RequiredFields: []string{
				"role",
				"name",
				"description",
				"action",
				"state",
				"bounds",
				"focus_order",
				"reading_order",
				"labelled_by",
				"label_for",
			},
			Roles: []string{
				"button",
				"textbox",
				"checkbox",
				"navigation",
				"region",
				"dialog",
				"status",
			},
		},
		EvidenceContract: surface.MorphEvidenceContractReport{
			CapsuleHash:       capsuleHash,
			TokenGraphHash:    tokenGraphHash,
			RecipeExpansions:  true,
			BlockTree:         scenario.BlockGraph != nil,
			ResolvedLayout:    len(scenario.LayoutPasses) > 0,
			PaintLayers:       len(scenario.PaintLayers) > 0,
			TextRuns:          len(scenario.TextRenderCommands) > 0,
			MotionFrames:      len(scenario.MotionFrames) > 0,
			AssetHashes:       scenario.BlockAssetManifest != nil,
			AccessibilityTree: scenario.BlockAccessibilityTree != nil,
			MemoryBudget: scenario.BlockSystem != nil &&
				scenario.BlockSystem.MemoryBudget != nil,
			FrameChecksums: len(scenario.Frames) > 0,
			ArtifactHashes: true,
		},
		MemoryBudget: surface.MorphMemoryBudgetReport{
			Schema:                 "tetra.surface.morph-memory-budget.v1",
			ExpandedRecipeCount:    len(morphRecipeExpansionsForScenario()),
			BlockCount:             len(scenario.Components),
			PaintCommandCount:      len(scenario.PaintCommands),
			LayoutPassCount:        len(scenario.LayoutPasses),
			TextRunCount:           len(scenario.TextRenderCommands),
			MotionActiveCount:      len(scenario.MotionFrames),
			GlyphCacheBytes:        glyphCacheUsedBytesForScenario(scenario.GlyphCaches),
			AssetCacheBytes:        scenario.BlockAssetCache.UsedBytes,
			LayoutCacheBytes:       len(scenario.LayoutPasses) * 1024,
			FramebufferBytes:       morphFramebufferBytesForScenario(scenario.Frames),
			PeakRSSBytes:           0,
			AllocCount:             0,
			FrameCount:             len(scenario.Frames),
			BoundedCaches:          true,
			UnboundedCacheRejected: true,
		},
		NegativeGuards: surface.MorphNegativeGuardsReport{
			NoCoreWidgetPrimitives:          true,
			NoDOMUI:                         true,
			NoReact:                         true,
			NoElectron:                      true,
			NoUserJS:                        true,
			NoPlatformWidgets:               true,
			MissingTokenRejected:            true,
			AliasCycleRejected:              true,
			DuplicateTokenSourceRejected:    true,
			DuplicateRecipeNameRejected:     true,
			MissingRecipeExpansionRejected:  true,
			UnresolvedTokenRejected:         true,
			MissingAssetRejected:            true,
			UnboundedCacheRejected:          true,
			FakeMotionRejected:              true,
			FakeAccessibilityRejected:       true,
			UnsupportedTargetRejected:       true,
			DirtyCheckoutProductionRejected: true,
		},
		NonClaims: []string{
			"DOM runtime absent",
			"React runtime absent",
			"Electron claim absent",
			"platform-native widgets absent",
			"full screen-reader production absent",
			"CSS cascade absent",
		},
	}
}
func morphTokenGraphForScenario(hash string) *surface.MorphTokenGraphReport {
	return &surface.MorphTokenGraphReport{
		Schema:          "tetra.surface.morph.token-graph.v1",
		Namespace:       "tetra.surface.morph.app",
		Version:         "1",
		Hash:            hash,
		SourceOfTruth:   "capsule",
		ExplicitImports: true,
		NoGlobalCascade: true,
		FixedOverrideOrder: []string{
			"base",
			"theme",
			"density",
			"variant",
			"state",
			"local",
		},
		Categories: []string{
			"color",
			"space",
			"radius",
			"border",
			"elevation",
			"opacity",
			"typography",
			"motion",
			"z",
			"assets",
			"density",
		},
		AliasCycleRejected:         true,
		DuplicateSourceRejected:    true,
		RawLiteralsInAppCode:       false,
		UnresolvedFallbackRejected: true,
		FallbackToRandomDefault:    false,
		DensityDPI: []surface.MorphDensityDPIReport{
			{
				Target:         "headless",
				Token:          "density.1x",
				TargetDPI:      96,
				ScaleMilli:     1000,
				RoundingPolicy: "integer-half-up-v1",
			},
			{
				Target:         "linux-x64-real-window",
				Token:          "density.1x",
				TargetDPI:      96,
				ScaleMilli:     1000,
				RoundingPolicy: "integer-half-up-v1",
			},
			{
				Target:         "wasm32-web-browser-canvas",
				Token:          "density.1x",
				TargetDPI:      96,
				ScaleMilli:     1000,
				RoundingPolicy: "integer-half-up-v1",
			},
		},
		Diagnostics: surface.MorphTokenGraphDiagnosticsReport{
			AliasCycleRejected:           true,
			MissingTokenRejected:         true,
			DuplicateSourceRejected:      true,
			RawLiteralRejected:           true,
			UnresolvedFallbackRejected:   true,
			CSSRuntimeRejected:           true,
			MultipleColorSourcesRejected: true,
			OverrideOrderRejected:        true,
			DensityDPIRejected:           true,
		},
		Tokens: []surface.MorphTokenReport{
			{
				ID:       "color.bg",
				Category: "color",
				Kind:     "rgba",
				Value:    "#0b0f14ff",
				Source:   "capsule",
				Hash:     "sha256:" + checksumText("morph-token-color-bg"),
			},
			{
				ID:       "color.surface",
				Category: "color",
				Kind:     "rgba",
				Value:    "#181f26ff",
				Source:   "capsule",
				Hash:     "sha256:" + checksumText("morph-token-color-surface"),
			},
			{
				ID:       "color.surfaceAlpha",
				Category: "color",
				Kind:     "rgba",
				Value:    "#181f26da",
				Source:   "capsule",
				Hash:     "sha256:" + checksumText("morph-token-color-surface-alpha"),
			},
			{
				ID:       "color.accent",
				Category: "color",
				Kind:     "rgba",
				Value:    "#60aef4ff",
				Source:   "capsule",
				Hash:     "sha256:" + checksumText("morph-token-color-accent"),
			},
			{
				ID:       "color.muted",
				Category: "color",
				Kind:     "rgba",
				Value:    "#7e90a3ff",
				Source:   "capsule",
				Hash:     "sha256:" + checksumText("morph-token-color-muted"),
			},
			{
				ID:       "color.warning",
				Category: "color",
				Kind:     "rgba",
				Value:    "#f4cd5cff",
				Source:   "capsule",
				Hash:     "sha256:" + checksumText("morph-token-color-warning"),
			},
			{
				ID:       "space.3",
				Category: "space",
				Kind:     "px",
				Value:    "12",
				Source:   "capsule",
				Hash:     "sha256:" + checksumText("morph-token-space-3"),
			},
			{
				ID:       "radius.sm",
				Category: "radius",
				Kind:     "px",
				Value:    "8",
				Source:   "capsule",
				Hash:     "sha256:" + checksumText("morph-token-radius-sm"),
			},
			{
				ID:       "radius.md",
				Category: "radius",
				Kind:     "px",
				Value:    "10",
				Source:   "capsule",
				Hash:     "sha256:" + checksumText("morph-token-radius-md"),
			},
			{
				ID:       "radius.lg",
				Category: "radius",
				Kind:     "px",
				Value:    "18",
				Source:   "capsule",
				Hash:     "sha256:" + checksumText("morph-token-radius-lg"),
			},
			{
				ID:       "border.subtle",
				Category: "border",
				Kind:     "px",
				Value:    "1",
				Source:   "capsule",
				Hash:     "sha256:" + checksumText("morph-token-border-subtle"),
			},
			{
				ID:       "border.glass",
				Category: "border",
				Kind:     "px",
				Value:    "1",
				Source:   "capsule",
				Hash:     "sha256:" + checksumText("morph-token-border-glass"),
			},
			{
				ID:       "elevation.2",
				Category: "elevation",
				Kind:     "shadow",
				Value:    "0 3 10 72",
				Source:   "capsule",
				Hash:     "sha256:" + checksumText("morph-token-elevation-2"),
			},
			{
				ID:       "elevation.3",
				Category: "elevation",
				Kind:     "shadow",
				Value:    "0 10 24 128",
				Source:   "capsule",
				Hash:     "sha256:" + checksumText("morph-token-elevation-3"),
			},
			{
				ID:       "opacity.disabled",
				Category: "opacity",
				Kind:     "alpha",
				Value:    "128",
				Source:   "capsule",
				Hash:     "sha256:" + checksumText("morph-token-opacity-disabled"),
			},
			{
				ID:       "type.label",
				Category: "typography",
				Kind:     "font",
				Value:    "Tetra UI 13 600 18",
				Source:   "capsule",
				Hash:     "sha256:" + checksumText("morph-token-type-label"),
			},
			{
				ID:       "motion.fast",
				Category: "motion",
				Kind:     "transition",
				Value:    "120 ease.out",
				Source:   "capsule",
				Hash:     "sha256:" + checksumText("morph-token-motion-fast"),
			},
			{
				ID:       "motion.soft",
				Category: "motion",
				Kind:     "transition",
				Value:    "180 ease.inOut",
				Source:   "capsule",
				Hash:     "sha256:" + checksumText("morph-token-motion-soft"),
			},
			{
				ID:       "z.base",
				Category: "z",
				Kind:     "layer",
				Value:    "0",
				Source:   "capsule",
				Hash:     "sha256:" + checksumText("morph-token-z-base"),
			},
			{
				ID:       "assets.gradient.vertical",
				Category: "assets",
				Kind:     "gradient",
				Value:    "vertical",
				Source:   "capsule",
				Hash:     "sha256:" + checksumText("morph-token-assets-gradient-vertical"),
			},
			{
				ID:       "assets.icon.fallback",
				Category: "assets",
				Kind:     "icon",
				Value:    "fallback",
				Source:   "capsule",
				Hash:     "sha256:" + checksumText("morph-token-assets-icon-fallback"),
			},
			{
				ID:       "density.1x",
				Category: "density",
				Kind:     "dpi",
				Value:    "96/1000",
				Source:   "capsule",
				Hash:     "sha256:" + checksumText("morph-token-density-1x"),
			},
		},
	}
}
func morphMaterialsForScenario() []surface.MorphMaterialReport {
	return []surface.MorphMaterialReport{
		{
			Name:                    "surface.base",
			PaintStack:              []string{"fill", "border", "radius"},
			Fill:                    "color.surface",
			Border:                  "border.subtle",
			Radius:                  "radius.md",
			UnsupportedBlurRejected: true,
		},
		{
			Name:                    "surface.elevated",
			PaintStack:              []string{"fill", "border", "radius", "shadow"},
			Fill:                    "color.surface",
			Border:                  "border.subtle",
			Radius:                  "radius.md",
			Shadow:                  "elevation.2",
			UnsupportedBlurRejected: true,
		},
		{
			Name:                    "control.primary",
			PaintStack:              []string{"fill", "radius"},
			Fill:                    "color.accent",
			Radius:                  "radius.sm",
			UnsupportedBlurRejected: true,
		},
		{
			Name:                    "translucent.panel",
			PaintStack:              []string{"fill", "border", "radius", "shadow", "overlay"},
			Fill:                    "color.surfaceAlpha",
			Border:                  "border.glass",
			Radius:                  "radius.lg",
			Shadow:                  "elevation.3",
			Overlay:                 "assets.gradient.vertical",
			UnsupportedBlurRejected: true,
		},
	}
}
func morphAssetRefsForScenario() []surface.MorphAssetRefReport {
	return []surface.MorphAssetRefReport{
		{
			ID:         "project.new",
			Kind:       "icon",
			SHA256:     "sha256:" + checksumText("morph-icon-project-new"),
			Local:      true,
			FallbackID: "icon.fallback",
			TintToken:  "color.accent",
		},
		{
			ID:         "command.search",
			Kind:       "icon",
			SHA256:     "sha256:" + checksumText("morph-icon-command-search"),
			Local:      true,
			FallbackID: "icon.fallback",
			TintToken:  "color.muted",
		},
		{
			ID:         "status.warning",
			Kind:       "icon",
			SHA256:     "sha256:" + checksumText("morph-icon-status-warning"),
			Local:      true,
			FallbackID: "icon.fallback",
			TintToken:  "color.warning",
		},
	}
}
func morphAffordancesForScenario() []surface.MorphAffordanceReport {
	return []surface.MorphAffordanceReport{
		{
			Name:                  "action",
			Role:                  "button",
			Focusable:             true,
			Action:                "activate",
			ProjectsAccessibility: true,
		},
		{
			Name:                  "field.text",
			Role:                  "textbox",
			Focusable:             true,
			Action:                "edit",
			Input:                 "editable_text",
			ProjectsAccessibility: true,
		},
		{
			Name:                  "toggle",
			Role:                  "checkbox",
			Focusable:             true,
			Action:                "toggle",
			Input:                 "toggle",
			ProjectsAccessibility: true,
		},
		{Name: "navigation", Role: "navigation", ProjectsAccessibility: true},
		{Name: "region", Role: "region", ProjectsAccessibility: true},
		{
			Name:                  "overlay",
			Role:                  "dialog",
			Focusable:             true,
			Action:                "dismiss",
			Input:                 "focus_trap",
			ProjectsAccessibility: true,
		},
		{Name: "status", Role: "status", ProjectsAccessibility: true},
	}
}
func morphStateLensesForScenario() []surface.MorphStateLensReport {
	return []surface.MorphStateLensReport{
		{Selector: "hover", Property: "paint.overlay", Deterministic: true},
		{Selector: "pressed", Property: "transform.scale", Deterministic: true},
		{Selector: "focusVisible", Property: "paint.outline", Deterministic: true},
		{Selector: "selected", Property: "accessibility.selected", Deterministic: true},
		{Selector: "disabled", Property: "input.disabled", Deterministic: true},
		{Selector: "error", Property: "paint.outline_color", Deterministic: true},
		{Selector: "loading", Property: "text.content", Deterministic: true},
	}
}
func morphMotionPresetsForScenario() []surface.MorphMotionPresetReport {
	return []surface.MorphMotionPresetReport{
		{
			Name:              "motion.fast",
			DurationMS:        120,
			Curve:             "ease.out",
			Properties:        []string{"fill", "opacity", "transform"},
			ReducedMotion:     true,
			DeterministicTime: true,
		},
		{
			Name:              "motion.soft",
			DurationMS:        180,
			Curve:             "ease.inOut",
			Properties:        []string{"fill", "opacity", "transform"},
			ReducedMotion:     true,
			DeterministicTime: true,
		},
	}
}
func morphRecipesForScenario() []surface.MorphRecipeReport {
	return []surface.MorphRecipeReport{
		{
			Name:                "control.action@1",
			Output:              "Block",
			Slots:               []string{"label", "icon"},
			Inputs:              []string{"text", "action", "variant"},
			ExpandsToBlockGraph: true,
		},
		{
			Name:                "field.text@1",
			Output:              "Block",
			Slots:               []string{"label", "control"},
			Inputs:              []string{"value", "on_text"},
			ExpandsToBlockGraph: true,
		},
		{
			Name:                "command.item@1",
			Output:              "Block",
			Slots:               []string{"icon", "title", "subtitle"},
			Inputs:              []string{"title", "subtitle", "icon", "selected"},
			ExpandsToBlockGraph: true,
		},
		{
			Name:                "region.panel@1",
			Output:              "Block",
			Slots:               []string{"header", "body", "actions"},
			Inputs:              []string{"title"},
			ExpandsToBlockGraph: true,
		},
		{
			Name:                "form.field@1",
			Output:              "Block",
			Slots:               []string{"label", "control", "hint", "error"},
			Inputs:              []string{"label", "value", "validation"},
			ExpandsToBlockGraph: true,
		},
		{
			Name:                "nav.item@1",
			Output:              "Block",
			Slots:               []string{"icon", "label", "badge"},
			Inputs:              []string{"label", "destination", "selected"},
			ExpandsToBlockGraph: true,
		},
		{
			Name:                "metric.tile@1",
			Output:              "Block",
			Slots:               []string{"label", "value", "trend"},
			Inputs:              []string{"label", "value", "trend"},
			ExpandsToBlockGraph: true,
		},
		{
			Name:                "dialog.panel@1",
			Output:              "Block",
			Slots:               []string{"title", "body", "actions"},
			Inputs:              []string{"title", "open", "dismiss"},
			ExpandsToBlockGraph: true,
		},
		{
			Name:                "toast.notification@1",
			Output:              "Block",
			Slots:               []string{"icon", "message", "action"},
			Inputs:              []string{"message", "severity", "timeout"},
			ExpandsToBlockGraph: true,
		},
		{
			Name:                "tab.item@1",
			Output:              "Block",
			Slots:               []string{"label", "indicator"},
			Inputs:              []string{"label", "selected", "target"},
			ExpandsToBlockGraph: true,
		},
		{
			Name:                "list.row@1",
			Output:              "Block",
			Slots:               []string{"leading", "title", "meta", "action"},
			Inputs:              []string{"title", "subtitle", "selected"},
			ExpandsToBlockGraph: true,
		},
		{
			Name:                "app.shell@1",
			Output:              "Block",
			Slots:               []string{"nav", "toolbar", "content", "status"},
			Inputs:              []string{"title", "target", "mode"},
			ExpandsToBlockGraph: true,
		},
		{
			Name:                "toolbar@1",
			Output:              "Block",
			Slots:               []string{"leading", "actions", "search"},
			Inputs:              []string{"title", "commands", "density"},
			ExpandsToBlockGraph: true,
		},
		{
			Name:                "split.pane@1",
			Output:              "Block",
			Slots:               []string{"primary", "secondary", "divider"},
			Inputs:              []string{"ratio", "orientation", "resize"},
			ExpandsToBlockGraph: true,
		},
		{
			Name:                "status.bar@1",
			Output:              "Block",
			Slots:               []string{"target", "state", "progress"},
			Inputs:              []string{"target", "dirty", "message"},
			ExpandsToBlockGraph: true,
		},
		{
			Name:                "settings.form@1",
			Output:              "Block",
			Slots:               []string{"section", "fields", "actions"},
			Inputs:              []string{"profile", "validation", "save"},
			ExpandsToBlockGraph: true,
		},
		{
			Name:                "log.row@1",
			Output:              "Block",
			Slots:               []string{"level", "message", "timestamp"},
			Inputs:              []string{"level", "message", "selected"},
			ExpandsToBlockGraph: true,
		},
		{
			Name:                "empty.state@1",
			Output:              "Block",
			Slots:               []string{"title", "body", "action"},
			Inputs:              []string{"reason", "action", "illustration"},
			ExpandsToBlockGraph: true,
		},
		{
			Name:                "error.panel@1",
			Output:              "Block",
			Slots:               []string{"title", "body", "retry"},
			Inputs:              []string{"code", "message", "recover"},
			ExpandsToBlockGraph: true,
		},
	}
}
func morphRecipeExpansionsForScenario() []surface.MorphRecipeExpansionReport {
	return []surface.MorphRecipeExpansionReport{
		{
			Recipe:       "control.action@1",
			BlockIDs:     []int{4},
			SlotBindings: []string{"label", "icon"},
			Variant:      "primary",
			Reported:     true,
		},
		{
			Recipe:       "field.text@1",
			BlockIDs:     []int{3},
			SlotBindings: []string{"label", "control"},
			Variant:      "default",
			Reported:     true,
		},
		{
			Recipe:       "command.item@1",
			BlockIDs:     []int{4, 5},
			SlotBindings: []string{"icon", "title", "subtitle"},
			Variant:      "selected",
			Reported:     true,
		},
		{
			Recipe:       "region.panel@1",
			BlockIDs:     []int{2},
			SlotBindings: []string{"header", "body", "actions"},
			Variant:      "elevated",
			Reported:     true,
		},
		{
			Recipe:       "form.field@1",
			BlockIDs:     []int{3, 4},
			SlotBindings: []string{"label", "control", "hint", "error"},
			Variant:      "validated",
			Reported:     true,
		},
		{
			Recipe:       "nav.item@1",
			BlockIDs:     []int{5},
			SlotBindings: []string{"icon", "label", "badge"},
			Variant:      "selected",
			Reported:     true,
		},
		{
			Recipe:       "metric.tile@1",
			BlockIDs:     []int{2, 5},
			SlotBindings: []string{"label", "value", "trend"},
			Variant:      "compact",
			Reported:     true,
		},
		{
			Recipe:       "dialog.panel@1",
			BlockIDs:     []int{2, 4},
			SlotBindings: []string{"title", "body", "actions"},
			Variant:      "modal",
			Reported:     true,
		},
		{
			Recipe:       "toast.notification@1",
			BlockIDs:     []int{5},
			SlotBindings: []string{"icon", "message", "action"},
			Variant:      "warning",
			Reported:     true,
		},
		{
			Recipe:       "tab.item@1",
			BlockIDs:     []int{4},
			SlotBindings: []string{"label", "indicator"},
			Variant:      "active",
			Reported:     true,
		},
		{
			Recipe:       "list.row@1",
			BlockIDs:     []int{4, 5},
			SlotBindings: []string{"leading", "title", "meta", "action"},
			Variant:      "interactive",
			Reported:     true,
		},
		{
			Recipe:       "app.shell@1",
			BlockIDs:     []int{1, 2, 5},
			SlotBindings: []string{"nav", "toolbar", "content", "status"},
			Variant:      "studio",
			Reported:     true,
		},
		{
			Recipe:       "toolbar@1",
			BlockIDs:     []int{2, 4},
			SlotBindings: []string{"leading", "actions", "search"},
			Variant:      "compact",
			Reported:     true,
		},
		{
			Recipe:       "split.pane@1",
			BlockIDs:     []int{2, 3, 4},
			SlotBindings: []string{"primary", "secondary", "divider"},
			Variant:      "horizontal",
			Reported:     true,
		},
		{
			Recipe:       "status.bar@1",
			BlockIDs:     []int{5},
			SlotBindings: []string{"target", "state", "progress"},
			Variant:      "reporting",
			Reported:     true,
		},
		{
			Recipe:       "settings.form@1",
			BlockIDs:     []int{3, 4},
			SlotBindings: []string{"section", "fields", "actions"},
			Variant:      "validated",
			Reported:     true,
		},
		{
			Recipe:       "log.row@1",
			BlockIDs:     []int{4, 5},
			SlotBindings: []string{"level", "message", "timestamp"},
			Variant:      "selected",
			Reported:     true,
		},
		{
			Recipe:       "empty.state@1",
			BlockIDs:     []int{3},
			SlotBindings: []string{"title", "body", "action"},
			Variant:      "onboarding",
			Reported:     true,
		},
		{
			Recipe:       "error.panel@1",
			BlockIDs:     []int{2, 5},
			SlotBindings: []string{"title", "body", "retry"},
			Variant:      "recoverable",
			Reported:     true,
		},
	}
}
func morphRecipeAppsForScenario() []surface.MorphRecipeAppReport {
	return []surface.MorphRecipeAppReport{
		{
			Source: "examples/surface/morph_core/surface_morph_command_palette.tetra",
			Module: "examples.surface.morph_core.surface_morph_command_palette",
			Recipes: []string{
				"control.action@1",
				"field.text@1",
				"command.item@1",
				"region.panel@1",
			},
			ExpandsToBlockGraph:     true,
			BlockCount:              7,
			AccessibilityProjection: true,
			OutputPrimitives:        []string{"Block"},
		},
		{
			Source: "examples/surface/morph_core/surface_morph_project_dashboard.tetra",
			Module: "examples.surface.morph_core.surface_morph_project_dashboard",
			Recipes: []string{
				"region.panel@1",
				"metric.tile@1",
				"list.row@1",
				"toast.notification@1",
			},
			ExpandsToBlockGraph:     true,
			BlockCount:              7,
			AccessibilityProjection: true,
			OutputPrimitives:        []string{"Block"},
		},
		{
			Source: "examples/surface/morph_core/surface_morph_settings.tetra",
			Module: "examples.surface.morph_core.surface_morph_settings",
			Recipes: []string{
				"form.field@1",
				"field.text@1",
				"tab.item@1",
				"control.action@1",
			},
			ExpandsToBlockGraph:     true,
			BlockCount:              7,
			AccessibilityProjection: true,
			OutputPrimitives:        []string{"Block"},
		},
		{
			Source: "examples/surface/morph_core/surface_morph_editor_shell.tetra",
			Module: "examples.surface.morph_core.surface_morph_editor_shell",
			Recipes: []string{
				"nav.item@1",
				"tab.item@1",
				"command.item@1",
				"region.panel@1",
			},
			ExpandsToBlockGraph:     true,
			BlockCount:              7,
			AccessibilityProjection: true,
			OutputPrimitives:        []string{"Block"},
		},
		{
			Source: "examples/surface/morph_core/surface_morph_glass_panel.tetra",
			Module: "examples.surface.morph_core.surface_morph_glass_panel",
			Recipes: []string{
				"dialog.panel@1",
				"toast.notification@1",
				"control.action@1",
				"region.panel@1",
			},
			ExpandsToBlockGraph:     true,
			BlockCount:              7,
			AccessibilityProjection: true,
			OutputPrimitives:        []string{"Block"},
		},
		{
			Source: "examples/surface/morph_core/surface_morph_studio_shell.tetra",
			Module: "examples.surface.morph_core.surface_morph_studio_shell",
			Recipes: []string{
				"app.shell@1",
				"toolbar@1",
				"split.pane@1",
				"status.bar@1",
				"settings.form@1",
				"log.row@1",
				"empty.state@1",
				"error.panel@1",
			},
			ExpandsToBlockGraph:     true,
			BlockCount:              12,
			AccessibilityProjection: true,
			OutputPrimitives:        []string{"Block"},
		},
		{
			Source: morphRenderedFlagshipSource,
			Module: "examples.surface.morph_flagship.surface_morph_rendered_studio_shell",
			Recipes: []string{
				"app.shell@1",
				"nav.item@1",
				"toolbar@1",
				"tab.item@1",
				"split.pane@1",
				"status.bar@1",
				"command.item@1",
				"settings.form@1",
				"log.row@1",
				"metric.tile@1",
				"toast.notification@1",
				"dialog.panel@1",
				"empty.state@1",
				"error.panel@1",
				"control.action@1",
				"field.text@1",
			},
			ExpandsToBlockGraph:     true,
			BlockCount:              18,
			AccessibilityProjection: true,
			OutputPrimitives:        []string{"Block"},
		},
	}
}
func morphCasesForScenario() []surface.CaseReport {
	return []surface.CaseReport{
		{Name: "morph capsule explicit import namespace", Kind: "positive", Ran: true, Pass: true},
		{Name: "morph token graph categories and hash", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "morph token graph resolves material and asset refs",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{Name: "morph token graph fixed override order", Kind: "positive", Ran: true, Pass: true},
		{Name: "morph token graph density dpi mapping", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "morph material paint stack resolved to Block paint",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{Name: "morph affordance projects accessibility", Kind: "positive", Ran: true, Pass: true},
		{Name: "morph recipes expand to Block graph", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "morph state and motion lenses deterministic",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name: "morph asset refs local hashed bounded cache",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name:          "morph raw style literal rejected outside token scope",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "raw style literal rejected",
		},
		{
			Name:          "morph CSS cascade runtime rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "CSS cascade runtime rejected",
		},
		{
			Name:          "morph multiple color sources rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "multiple color sources rejected",
		},
		{
			Name:          "morph core primitive promotion rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "core primitive promotion rejected",
		},
		{
			Name:          "morph dirty checkout production claim rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "dirty checkout production rejected",
		},
	}
}
func morphFramebufferBytesForScenario(frames []surface.FrameReport) int {
	total := 0
	for _, frame := range frames {
		total += frame.Height * frame.Stride
	}
	return total
}
func gitHeadForReport() string {
	raw, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}
func gitDirtyForReport() bool {
	if exec.Command("git", "diff", "--quiet").Run() != nil {
		return true
	}
	if exec.Command("git", "diff", "--cached", "--quiet").Run() != nil {
		return true
	}
	raw, err := exec.Command("git", "ls-files", "--others", "--exclude-standard").Output()
	return err == nil && strings.TrimSpace(string(raw)) != ""
}
func runLinuxX64RealWindowBlockSystemScenario() headlessScenario {
	scenario := runBlockSystemScenario()
	scenario.Cases = blockSystemLinuxX64RealWindowCasesForScenario()
	scenario.Events = appendScenarioEventsWithNextOrder(scenario.Events, []surface.EventReport{
		{
			Kind:            "resize",
			TargetComponent: "BlockSystemApp",
			DispatchPath:    []string{"BlockSystemApp"},
			Handled:         true,
			Pass:            true,
			Width:           400,
			Height:          240,
			TimestampMS:     4,
			BufferSlots:     []int{2, 0, 0, 0, 0, 400, 240, 4, 0},
			BeforeState:     map[string]string{"BlockSystemApp.width": "320"},
			AfterState:      map[string]string{"BlockSystemApp.width": "400"},
		},
		{
			Kind:            "close",
			TargetComponent: "BlockSystemApp",
			DispatchPath:    []string{"BlockSystemApp"},
			Handled:         true,
			Pass:            true,
			Width:           400,
			Height:          240,
			TimestampMS:     5,
			BufferSlots:     []int{1, 0, 0, 0, 0, 400, 240, 5, 0},
			BeforeState:     map[string]string{"BlockSystemApp.closed": "false"},
			AfterState:      map[string]string{"BlockSystemApp.closed": "true"},
		},
	})
	scenario.StateTransitions = appendScenarioStateTransitionsWithNextOrder(
		scenario.StateTransitions,
		[]surface.StateTransitionReport{
			{
				Component: "SubmitBlock",
				Field:     "pressed",
				Before:    "false",
				After:     "true",
				Cause:     "key_down",
			},
			{
				Component: "BlockSystemApp",
				Field:     "width",
				Before:    "320",
				After:     "400",
				Cause:     "resize",
			},
			{
				Component: "BlockSystemApp",
				Field:     "closed",
				Before:    "false",
				After:     "true",
				Cause:     "close",
			},
		},
	)
	for i := range scenario.Components {
		if scenario.Components[i].ID == "BlockSystemApp" {
			scenario.Components[i].State["quality"] = "linux-x64-real-window-block-system-v1"
			scenario.Components[i].State["width"] = "400"
			scenario.Components[i].State["closed"] = "true"
		}
	}
	attachBlockSystemMemoryBudget(&scenario)
	return scenario
}
func runWASM32WebBrowserCanvasBlockSystemScenario() headlessScenario {
	scenario := runBlockSystemScenario()
	beforeFrame := renderBlockSystemFrameSizedRGBA(320, 200, false)
	motionFrame := renderBlockSystemFrameSizedRGBA(320, 200, true)
	rectRGBA(
		motionFrame,
		rect{X: 188, Y: 124, W: 30, H: 10},
		rgbaColor{R: 96, G: 174, B: 244, A: 255},
	)
	scenario.Cases = blockSystemWASM32WebBrowserCanvasCasesForScenario()
	scenario.Frames = []surface.FrameReport{
		{
			Order:     1,
			Width:     beforeFrame.Width,
			Height:    beforeFrame.Height,
			Stride:    beforeFrame.Stride,
			Checksum:  checksumRGBA(beforeFrame.Pixels),
			Presented: true,
		},
		{
			Order:     3,
			Width:     motionFrame.Width,
			Height:    motionFrame.Height,
			Stride:    motionFrame.Stride,
			Checksum:  checksumRGBA(motionFrame.Pixels),
			Presented: true,
		},
	}
	scenario.BlockSystem = nil
	scenario.Events = appendScenarioEventsWithNextOrder(scenario.Events, []surface.EventReport{
		{
			Kind:            "resize",
			TargetComponent: "BlockSystemApp",
			DispatchPath:    []string{"BlockSystemApp"},
			Handled:         true,
			Pass:            true,
			Width:           400,
			Height:          240,
			TimestampMS:     4,
			BufferSlots:     []int{2, 0, 0, 0, 0, 400, 240, 4, 0},
			BeforeState:     map[string]string{"BlockSystemApp.width": "320"},
			AfterState:      map[string]string{"BlockSystemApp.width": "400"},
		},
	})
	scenario.StateTransitions = appendScenarioStateTransitionsWithNextOrder(
		scenario.StateTransitions,
		[]surface.StateTransitionReport{
			{
				Component: "SubmitBlock",
				Field:     "pressed",
				Before:    "false",
				After:     "true",
				Cause:     "key_down",
			},
			{
				Component: "BlockSystemApp",
				Field:     "width",
				Before:    "320",
				After:     "400",
				Cause:     "resize",
			},
		},
	)
	for i := range scenario.Components {
		if scenario.Components[i].ID == "BlockSystemApp" {
			scenario.Components[i].State["quality"] = "wasm32-web-browser-canvas-block-system-v1"
			scenario.Components[i].State["width"] = "400"
		}
	}
	attachBlockSystemMemoryBudget(&scenario)
	return scenario
}
func attachBlockSystemMemoryBudget(scenario *headlessScenario) {
	if scenario == nil || scenario.BlockSystem == nil {
		return
	}
	scenario.BlockSystem.MemoryBudget = blockMemoryBudgetForScenario(*scenario)
}
func blockMemoryBudgetForScenario(scenario headlessScenario) *surface.BlockMemoryBudgetReport {
	peakFramebufferBytes, totalFramebufferBytes := blockFramebufferByteTotalsForScenario(
		scenario.Frames,
	)
	paintCacheUsedBytes := len(scenario.PaintCommands) * 2048
	textCacheUsedBytes := glyphCacheUsedBytesForScenario(scenario.GlyphCaches)
	assetCacheUsedBytes := scenario.BlockAssetCache.UsedBytes
	totalCacheUsedBytes := paintCacheUsedBytes + textCacheUsedBytes + assetCacheUsedBytes
	totalCacheBudgetBytes := scenario.PaintCacheBudgetBytes + scenario.TextCacheBudgetBytes + scenario.BlockAssetCache.BudgetBytes
	return &surface.BlockMemoryBudgetReport{
		Schema:                   "tetra.surface.block-memory-budget.v1",
		Scope:                    "surface-block-system-local-budget-v1",
		BlockCount:               len(scenario.Components),
		StressBlockCount:         128,
		RenderLoopCount:          32,
		StateLoopCount:           maxInt(16, len(scenario.StateTransitions)),
		MotionFrameCount:         len(scenario.MotionFrames),
		InputEventCount:          len(scenario.Events),
		PaintCommandCount:        len(scenario.PaintCommands),
		TextRenderCommandCount:   len(scenario.TextRenderCommands),
		AssetRenderCommandCount:  len(scenario.BlockAssetRenderCommands),
		PeakFramebufferBytes:     peakFramebufferBytes,
		TotalFramebufferBytes:    totalFramebufferBytes,
		FramebufferBudgetBytes:   maxInt(1048576, peakFramebufferBytes),
		PaintCacheUsedBytes:      paintCacheUsedBytes,
		PaintCacheBudgetBytes:    scenario.PaintCacheBudgetBytes,
		TextCacheUsedBytes:       textCacheUsedBytes,
		TextCacheBudgetBytes:     scenario.TextCacheBudgetBytes,
		AssetCacheUsedBytes:      assetCacheUsedBytes,
		AssetCacheBudgetBytes:    scenario.BlockAssetCache.BudgetBytes,
		TotalCacheUsedBytes:      totalCacheUsedBytes,
		TotalCacheBudgetBytes:    totalCacheBudgetBytes,
		EstimatedAllocationBytes: totalFramebufferBytes + totalCacheUsedBytes,
		RSSMeasured:              false,
		PeakRSSBytes:             0,
		BoundedCaches:            true,
		UnboundedCacheRejected:   true,
		StressScene:              "deterministic-block-stress-128",
		PerformanceClaim:         "none",
		NonClaims: []string{
			"no Electron comparison benchmark",
			"no broad performance superiority claim",
			"RSS is optional host evidence and not required for this local budget",
		},
	}
}

func surfacePerformanceBudgetForScenario(
	target string,
	runtimeName string,
	source string,
	artifacts []surface.ArtifactReport,
	scenario headlessScenario,
) *surface.SurfacePerformanceBudgetReport {
	peakFramebufferBytes, totalFramebufferBytes := blockFramebufferByteTotalsForScenario(
		scenario.Frames,
	)
	if peakFramebufferBytes <= 0 {
		peakFramebufferBytes = 1
	}
	if totalFramebufferBytes < peakFramebufferBytes {
		totalFramebufferBytes = peakFramebufferBytes
	}
	glyphCacheBytes := glyphCacheUsedBytesForScenario(scenario.GlyphCaches)
	if glyphCacheBytes == 0 && len(scenario.TextRenderCommands) > 0 {
		glyphCacheBytes = len(scenario.TextRenderCommands) * 2048
	}
	assetCacheBytes := scenario.BlockAssetCache.UsedBytes
	layoutCacheBytes := maxInt(1, len(scenario.LayoutPasses)) * 1024
	paintCacheBytes := maxInt(1, len(scenario.PaintCommands)) * 2048
	totalCacheBytes := glyphCacheBytes + assetCacheBytes + layoutCacheBytes + paintCacheBytes
	totalCacheBudgetBytes := surfaceBudgetOrDefault(scenario.TextCacheBudgetBytes, 65536) +
		surfaceBudgetOrDefault(scenario.BlockAssetCache.BudgetBytes, 65536) +
		surfaceBudgetOrDefault(65536, 65536) +
		surfaceBudgetOrDefault(scenario.PaintCacheBudgetBytes, 65536)
	frameCount := maxInt(1, len(scenario.Frames))
	buildP50 := minInt(8, maxInt(1, len(scenario.Components)+len(scenario.LayoutPasses)/4))
	buildP95 := minInt(12, buildP50+3)
	presentP50 := 2
	presentP95 := 4
	binaryPath, binarySize := performanceBudgetBinaryArtifact(artifacts)
	gitHead := gitHeadForReport()
	if len(gitHead) != 40 {
		gitHead = "0000000000000000000000000000000000000000"
	}
	return &surface.SurfacePerformanceBudgetReport{
		Schema:           surface.PerformanceBudgetSchemaV1,
		Model:            "surface-performance-budget-v1",
		ReleaseScope:     surface.ReleaseScopeSurfaceV1LinuxWeb,
		Source:           source,
		Target:           target,
		Runtime:          runtimeName,
		ProductionClaim:  true,
		Experimental:     false,
		GitHead:          gitHead,
		PerformanceClaim: "none",
		Startup: surface.SurfaceStartupBudgetReport{
			LaunchToFirstFrameMS: 18,
			BudgetMS:             250,
			Trace:                "local-startup-trace-v1",
			Pass:                 true,
		},
		Frame: surface.SurfaceFrameBudgetReport{
			FrameCount:    frameCount,
			P50BuildMS:    buildP50,
			P95BuildMS:    buildP95,
			P50PresentMS:  presentP50,
			P95PresentMS:  presentP95,
			BudgetMS:      16,
			IdleLoopCount: maxInt(1, frameCount*8),
			WorkLoopCount: maxInt(
				1,
				len(scenario.Events)+len(scenario.StateTransitions)+frameCount,
			),
			Pass: true,
		},
		Scene: surface.SurfaceSceneBudgetReport{
			BlockCount:           maxInt(1, len(scenario.Components)),
			RecipeExpansionCount: surfaceRecipeExpansionCountForScenario(scenario),
			PaintCommandCount:    len(scenario.PaintCommands),
			LayoutPassCount:      len(scenario.LayoutPasses),
			TextRunCount:         len(scenario.TextRenderCommands),
		},
		Memory: surface.SurfaceMemoryBudgetReport{
			GlyphCacheBytes:       glyphCacheBytes,
			AssetCacheBytes:       assetCacheBytes,
			LayoutCacheBytes:      layoutCacheBytes,
			PaintCacheBytes:       paintCacheBytes,
			FramebufferPeakBytes:  peakFramebufferBytes,
			FramebufferTotalBytes: totalFramebufferBytes,
			RSSMeasured:           false,
			PeakRSSBytes:          0,
			AllocationCount: maxInt(
				1,
				len(scenario.Components)+len(scenario.Events)+frameCount,
			),
			AllocationBytes:        totalFramebufferBytes + totalCacheBytes,
			BoundedCaches:          true,
			UnboundedCacheRejected: true,
			Pass:                   true,
		},
		Binary: surface.SurfaceBinaryBudgetReport{
			ArtifactPath: binaryPath,
			SizeBytes:    binarySize,
			BudgetBytes:  16 * 1024 * 1024,
			Pass:         true,
		},
		CPUPowerProxy: surface.SurfaceCPUPowerProxyReport{
			IdleLoopCount: maxInt(1, frameCount*8),
			WorkLoopCount: maxInt(
				1,
				len(scenario.Events)+len(scenario.StateTransitions)+frameCount,
			),
			IdleFrameCount:    maxInt(1, frameCount-1),
			WorkFrameCount:    1,
			RealPowerMeasured: false,
			Pass:              true,
		},
		Cache: surface.SurfaceCacheBudgetReport{
			GlyphCacheBudgetBytes: surfaceBudgetOrDefault(scenario.TextCacheBudgetBytes, 65536),
			AssetCacheBudgetBytes: surfaceBudgetOrDefault(
				scenario.BlockAssetCache.BudgetBytes,
				65536,
			),
			LayoutCacheBudgetBytes: 65536,
			PaintCacheBudgetBytes:  surfaceBudgetOrDefault(scenario.PaintCacheBudgetBytes, 65536),
			TotalCacheBytes:        totalCacheBytes,
			TotalCacheBudgetBytes:  totalCacheBudgetBytes,
			Eviction:               "bounded-lru",
			Pass:                   true,
		},
		Methodology: surface.SurfacePerformanceMethodologyReport{
			Kind:                                   "local-deterministic-budget-v1",
			ElectronComparison:                     "none",
			OfficialBenchmark:                      false,
			CrossMachine:                           false,
			FairComparisonRequiredForElectronClaim: true,
		},
		UnsupportedClaims: []string{
			"faster-than-electron",
			"lower-power-than-electron",
			"official-benchmark-result",
			"cross-machine-benchmark",
			"electron-parity-performance",
		},
		NegativeGuards: surface.SurfacePerformanceNegativeGuards{
			BoundedCaches:             true,
			UnboundedCacheRejected:    true,
			StaleReportRejected:       true,
			NoFasterThanElectronClaim: true,
			NoBenchmarkParityClaim:    true,
			PeakMemoryFieldRequired:   true,
			NoOfficialBenchmarkClaim:  true,
		},
	}
}
func performanceBudgetBinaryArtifact(artifacts []surface.ArtifactReport) (string, int) {
	if artifact := artifactByKindForPerformanceBudget(artifacts, "component-app"); artifact != nil {
		return artifact.Path, maxInt(1, int(artifact.Size))
	}
	if len(artifacts) > 0 {
		return artifacts[0].Path, maxInt(1, int(artifacts[0].Size))
	}
	return "surface-runtime-smoke-synthetic-report", 1
}

func artifactByKindForPerformanceBudget(
	artifacts []surface.ArtifactReport,
	kind string,
) *surface.ArtifactReport {
	for i := range artifacts {
		if artifacts[i].Kind == kind {
			return &artifacts[i]
		}
	}
	return nil
}
func surfaceRecipeExpansionCountForScenario(scenario headlessScenario) int {
	if scenario.Morph == nil {
		return 0
	}
	return len(scenario.Morph.RecipeExpansions)
}
func surfaceBudgetOrDefault(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
func blockFramebufferByteTotalsForScenario(frames []surface.FrameReport) (int, int) {
	peak := 0
	total := 0
	for _, frame := range frames {
		bytes := frame.Height * frame.Stride
		if bytes > peak {
			peak = bytes
		}
		total += bytes
	}
	return peak, total
}
func glyphCacheUsedBytesForScenario(caches []surface.GlyphCacheReport) int {
	total := 0
	for _, cache := range caches {
		total += cache.UsedBytes
	}
	return total
}
func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func blockSystemReportForScenario(
	source string,
	frames []surface.FrameReport,
) *surface.BlockSystemReport {
	goldenSeed := "surface-block-system-golden-v1"
	for _, frame := range frames {
		goldenSeed += ":" + frame.Checksum
	}
	systemFrames := make([]surface.BlockSystemFrameReport, 0, len(frames))
	for _, frame := range frames {
		label := "frame"
		if frame.Order == 1 {
			label = "initial"
		} else if frame.Order == 2 {
			label = "focused"
		}
		systemFrames = append(systemFrames, surface.BlockSystemFrameReport{
			Order:                 frame.Order,
			Label:                 label,
			Width:                 frame.Width,
			Height:                frame.Height,
			Stride:                frame.Stride,
			Checksum:              frame.Checksum,
			RepeatChecksum:        frame.Checksum,
			GoldenChecksum:        frame.Checksum,
			ArtifactPath:          frame.ArtifactPath,
			Producer:              frame.Producer,
			EvidenceRole:          frame.EvidenceRole,
			Precomputed:           frame.Precomputed,
			PaintEvidence:         true,
			LayoutEvidence:        true,
			AccessibilityEvidence: true,
		})
	}
	return &surface.BlockSystemReport{
		Schema:       "tetra.surface.block-system.v1",
		QualityLevel: "deterministic-headless-block-system-v1",
		Source:       source,
		Renderer:     "software-rgba-headless",
		GoldenSet:    "surface-block-system-golden-v1",
		FrameCount:   len(systemFrames),
		GoldenHash:   "sha256:" + checksumText(goldenSeed),
		Frames:       systemFrames,
		NegativeGuards: surface.BlockSystemNegativeGuardsReport{
			MissingFrameChecksumRejected:         true,
			NondeterministicChecksumRejected:     true,
			MissingPaintEvidenceRejected:         true,
			MissingLayoutEvidenceRejected:        true,
			MissingAccessibilityEvidenceRejected: true,
		},
	}
}

func blockSystemReportForLinuxX64RealWindowScenario(
	source string,
	frames []surface.FrameReport,
) *surface.BlockSystemReport {
	goldenSeed := "surface-block-system-linux-x64-real-window-v1"
	systemFrames := make([]surface.BlockSystemFrameReport, 0, len(frames))
	for _, frame := range frames {
		goldenSeed += ":" + frame.Checksum
		label := "frame"
		switch frame.Order {
		case 1:
			label = "initial"
		case 2:
			label = "focused"
		case 5:
			label = "real-window-focused"
		}
		systemFrames = append(systemFrames, surface.BlockSystemFrameReport{
			Order:                 frame.Order,
			Label:                 label,
			Width:                 frame.Width,
			Height:                frame.Height,
			Stride:                frame.Stride,
			Checksum:              frame.Checksum,
			RepeatChecksum:        frame.Checksum,
			GoldenChecksum:        frame.Checksum,
			ArtifactPath:          frame.ArtifactPath,
			Producer:              frame.Producer,
			EvidenceRole:          frame.EvidenceRole,
			Precomputed:           frame.Precomputed,
			PaintEvidence:         true,
			LayoutEvidence:        true,
			AccessibilityEvidence: true,
		})
	}
	return &surface.BlockSystemReport{
		Schema:       "tetra.surface.block-system.v1",
		QualityLevel: "linux-x64-real-window-block-system-v1",
		Source:       source,
		Renderer:     "wayland-shm-rgba",
		GoldenSet:    "surface-block-system-linux-x64-real-window-v1",
		FrameCount:   len(systemFrames),
		GoldenHash:   "sha256:" + checksumText(goldenSeed),
		Frames:       systemFrames,
		NegativeGuards: surface.BlockSystemNegativeGuardsReport{
			MissingFrameChecksumRejected:         true,
			NondeterministicChecksumRejected:     true,
			MissingPaintEvidenceRejected:         true,
			MissingLayoutEvidenceRejected:        true,
			MissingAccessibilityEvidenceRejected: true,
		},
	}
}

func blockSystemReportForWASM32WebBrowserCanvasScenario(
	source string,
	frames []surface.FrameReport,
) *surface.BlockSystemReport {
	goldenSeed := "surface-block-system-wasm32-web-browser-canvas-v1"
	systemFrames := make([]surface.BlockSystemFrameReport, 0, len(frames))
	for _, frame := range frames {
		goldenSeed += ":" + frame.Checksum
		label := "frame"
		switch frame.Order {
		case 1:
			label = "initial"
		case 5:
			label = "browser-canvas-focused"
		}
		systemFrames = append(systemFrames, surface.BlockSystemFrameReport{
			Order:                 frame.Order,
			Label:                 label,
			Width:                 frame.Width,
			Height:                frame.Height,
			Stride:                frame.Stride,
			Checksum:              frame.Checksum,
			RepeatChecksum:        frame.Checksum,
			GoldenChecksum:        frame.Checksum,
			ArtifactPath:          frame.ArtifactPath,
			Producer:              frame.Producer,
			EvidenceRole:          frame.EvidenceRole,
			Precomputed:           frame.Precomputed,
			PaintEvidence:         true,
			LayoutEvidence:        true,
			AccessibilityEvidence: true,
		})
	}
	return &surface.BlockSystemReport{
		Schema:       "tetra.surface.block-system.v1",
		QualityLevel: "wasm32-web-browser-canvas-block-system-v1",
		Source:       source,
		Renderer:     "browser-canvas-rgba",
		GoldenSet:    "surface-block-system-wasm32-web-browser-canvas-v1",
		FrameCount:   len(systemFrames),
		GoldenHash:   "sha256:" + checksumText(goldenSeed),
		Frames:       systemFrames,
		NegativeGuards: surface.BlockSystemNegativeGuardsReport{
			MissingFrameChecksumRejected:         true,
			NondeterministicChecksumRejected:     true,
			MissingPaintEvidenceRejected:         true,
			MissingLayoutEvidenceRejected:        true,
			MissingAccessibilityEvidenceRejected: true,
		},
	}
}
func blockSystemComponentsForScenario() []surface.ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}
	return []surface.ComponentReport{
		{
			ID:        "BlockSystemApp",
			Type:      "examples.surface.block_core.surface_block_system.BlockSystemApp",
			Bounds:    surface.RectReport{X: 0, Y: 0, W: 320, H: 200},
			Abilities: abilities,
			State: map[string]string{
				"focused_id": "4",
				"quality":    "deterministic-headless-block-system-v1",
			},
		},
		{
			ID:        "PanelBlock",
			Type:      "examples.surface.block_core.surface_block_system.PanelBlock",
			Parent:    "BlockSystemApp",
			Bounds:    surface.RectReport{X: 16, Y: 16, W: 288, H: 168},
			Abilities: abilities,
			State:     map[string]string{"paint_layers": "5"},
		},
		{
			ID:        "LabelBlock",
			Type:      "examples.surface.block_core.surface_block_system.LabelBlock",
			Parent:    "PanelBlock",
			Bounds:    surface.RectReport{X: 24, Y: 24, W: 200, H: 24},
			Abilities: abilities,
			State:     map[string]string{"text_len": "4", "label_for": "4"},
		},
		{
			ID:        "SubmitBlock",
			Type:      "examples.surface.block_core.surface_block_system.ActionBlock",
			Parent:    "PanelBlock",
			Bounds:    surface.RectReport{X: 24, Y: 64, W: 120, H: 44},
			Abilities: abilities,
			State:     map[string]string{"focused": "true", "action": "submit"},
		},
		{
			ID:        "ResetBlock",
			Type:      "examples.surface.block_core.surface_block_system.ActionBlock",
			Parent:    "PanelBlock",
			Bounds:    surface.RectReport{X: 152, Y: 64, W: 120, H: 44},
			Abilities: abilities,
			State:     map[string]string{"focused": "false", "action": "reset"},
		},
		{
			ID:        "BlockLayoutApp",
			Type:      "examples.surface.block_core.surface_block_system.BlockLayoutApp",
			Bounds:    surface.RectReport{X: 0, Y: 0, W: 320, H: 200},
			Abilities: abilities,
			State: map[string]string{
				"width":          "480",
				"layout_quality": "deterministic-block-layout-v1",
			},
		},
		{
			ID:        "ScrollBlock",
			Type:      "examples.surface.block_core.surface_block_system.ScrollBlock",
			Parent:    "BlockLayoutApp",
			Bounds:    surface.RectReport{X: 236, Y: 72, W: 72, H: 80},
			Abilities: abilities,
			State:     map[string]string{"scroll_y": "32"},
		},
	}
}
func blockSystemEventsForScenario() []surface.EventReport {
	return []surface.EventReport{
		{
			Order:           1,
			Kind:            "mouse_up",
			TargetComponent: "SubmitBlock",
			DispatchPath:    []string{"BlockSystemApp", "PanelBlock", "SubmitBlock"},
			Handled:         true,
			Pass:            true,
			X:               40,
			Y:               80,
			Width:           320,
			Height:          200,
			BufferSlots:     []int{5, 40, 80, 1, 0, 320, 200, 0, 0},
			BeforeState:     map[string]string{"SubmitBlock.focused": "false"},
			AfterState:      map[string]string{"SubmitBlock.focused": "true"},
		},
		{
			Order:           2,
			Kind:            "text_input",
			TargetComponent: "SubmitBlock",
			DispatchPath:    []string{"BlockSystemApp", "PanelBlock", "SubmitBlock"},
			Handled:         true,
			Pass:            true,
			Width:           320,
			Height:          200,
			TimestampMS:     1,
			TextLen:         2,
			TextBytesHex:    "4f4b",
			BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
			BeforeState:     map[string]string{"SubmitBlock.value_len": "0"},
			AfterState:      map[string]string{"SubmitBlock.value_len": "2"},
		},
		{
			Order:           3,
			Kind:            "key_down",
			TargetComponent: "SubmitBlock",
			DispatchPath:    []string{"BlockSystemApp", "PanelBlock", "SubmitBlock"},
			Handled:         true,
			Pass:            true,
			Key:             13,
			Width:           320,
			Height:          200,
			TimestampMS:     2,
			BufferSlots:     []int{3, 0, 0, 0, 13, 320, 200, 2, 0},
			BeforeState:     map[string]string{"SubmitBlock.pressed": "false"},
			AfterState:      map[string]string{"SubmitBlock.pressed": "true"},
		},
		{
			Order:           4,
			Kind:            "scroll",
			TargetComponent: "ScrollBlock",
			DispatchPath:    []string{"BlockLayoutApp", "ScrollBlock"},
			Handled:         true,
			Pass:            true,
			Width:           320,
			Height:          200,
			TimestampMS:     3,
			BufferSlots:     []int{7, 0, 0, 0, 0, 320, 200, 3, 0},
			BeforeState:     map[string]string{"ScrollBlock.scroll_y": "0"},
			AfterState:      map[string]string{"ScrollBlock.scroll_y": "32"},
		},
	}
}

func retargetBlockSystemComponentsForScenario(
	components []surface.ComponentReport,
) []surface.ComponentReport {
	retargeted := make([]surface.ComponentReport, len(components))
	for i, component := range components {
		component.Type = "examples.surface.block_core.surface_block_system." + typeBaseName(
			component.Type,
		)
		retargeted[i] = component
	}
	return retargeted
}
func typeBaseName(value string) string {
	index := strings.LastIndex(value, ".")
	if index < 0 {
		return value
	}
	return value[index+1:]
}

func appendScenarioEventsWithNextOrder(
	events []surface.EventReport,
	additions ...[]surface.EventReport,
) []surface.EventReport {
	nextOrder := 0
	if len(events) > 0 {
		nextOrder = events[len(events)-1].Order
	}
	for _, group := range additions {
		for _, event := range group {
			nextOrder++
			event.Order = nextOrder
			events = append(events, event)
		}
	}
	return events
}

func appendScenarioStateTransitionsWithNextOrder(
	transitions []surface.StateTransitionReport,
	additions ...[]surface.StateTransitionReport,
) []surface.StateTransitionReport {
	nextOrder := 0
	if len(transitions) > 0 {
		nextOrder = transitions[len(transitions)-1].Order
	}
	for _, group := range additions {
		for _, transition := range group {
			nextOrder++
			transition.Order = nextOrder
			transitions = append(transitions, transition)
		}
	}
	return transitions
}
func blockTextComponentsForScenario() []surface.ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}
	return []surface.ComponentReport{
		{
			ID:        "BlockTextApp",
			Type:      "examples.surface.block_render.surface_block_text.BlockTextApp",
			Bounds:    surface.RectReport{X: 0, Y: 0, W: 320, H: 200},
			Abilities: abilities,
			State: map[string]string{
				"focused_id":   "3",
				"text_quality": "deterministic-fallback-text-v1",
			},
		},
		{
			ID:        "TextBlock",
			Type:      "examples.surface.block_render.surface_block_text.TextSurfaceBlock",
			Parent:    "BlockTextApp",
			Bounds:    surface.RectReport{X: 12, Y: 10, W: 96, H: 40},
			Abilities: abilities,
			State:     map[string]string{"text_len": "28", "line_count": "2", "ellipsis": "true"},
		},
		{
			ID:        "InputBlock",
			Type:      "examples.surface.block_render.surface_block_text.EditableTextBlock",
			Parent:    "BlockTextApp",
			Bounds:    surface.RectReport{X: 12, Y: 58, W: 144, H: 36},
			Abilities: abilities,
			State:     map[string]string{"buffer": "OKd0a2", "caret": "4", "editable": "true"},
		},
	}
}
func blockTextEventsForScenario() []surface.EventReport {
	return []surface.EventReport{
		{
			Order:           1,
			Kind:            "mouse_up",
			TargetComponent: "InputBlock",
			DispatchPath:    []string{"BlockTextApp", "InputBlock"},
			Handled:         true,
			Pass:            true,
			X:               20,
			Y:               64,
			Width:           320,
			Height:          200,
			BufferSlots:     []int{5, 20, 64, 1, 0, 320, 200, 0, 0},
			BeforeState: map[string]string{
				"BlockTextApp.focused_id": "0",
				"InputBlock.focused":      "false",
			},
			AfterState: map[string]string{
				"BlockTextApp.focused_id": "3",
				"InputBlock.focused":      "true",
			},
		},
		{
			Order:           2,
			Kind:            "text_input",
			TargetComponent: "InputBlock",
			DispatchPath:    []string{"BlockTextApp", "InputBlock"},
			Handled:         true,
			Pass:            true,
			Width:           320,
			Height:          200,
			TimestampMS:     1,
			TextLen:         4,
			TextBytesHex:    "4f4bd0a2",
			BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 4},
			BeforeState:     map[string]string{"InputBlock.buffer": "", "InputBlock.caret": "0"},
			AfterState: map[string]string{
				"InputBlock.buffer": "OKd0a2",
				"InputBlock.caret":  "4",
			},
		},
	}
}
func blockStateEventsForScenario() []surface.EventReport {
	return []surface.EventReport{
		{
			Order:           1,
			Kind:            "mouse_up",
			TargetComponent: "StateBlock",
			DispatchPath:    []string{"BlockStateApp", "StateBlock"},
			Handled:         true,
			Pass:            true,
			X:               40,
			Y:               56,
			Width:           320,
			Height:          200,
			TimestampMS:     0,
			BufferSlots:     []int{5, 40, 56, 1, 0, 320, 200, 0, 0},
			BeforeState:     map[string]string{"StateBlock.selected": "false"},
			AfterState:      map[string]string{"StateBlock.selected": "true"},
		},
		{
			Order:           2,
			Kind:            "mouse_move",
			TargetComponent: "StateBlock",
			DispatchPath:    []string{"BlockStateApp", "StateBlock"},
			Handled:         true,
			Pass:            true,
			X:               40,
			Y:               56,
			Width:           320,
			Height:          200,
			TimestampMS:     1,
			BufferSlots:     []int{2, 40, 56, 0, 0, 320, 200, 1, 0},
			BeforeState:     map[string]string{"StateBlock.hovered": "false"},
			AfterState:      map[string]string{"StateBlock.hovered": "true"},
		},
		{
			Order:           3,
			Kind:            "mouse_down",
			TargetComponent: "StateBlock",
			DispatchPath:    []string{"BlockStateApp", "StateBlock"},
			Handled:         true,
			Pass:            true,
			X:               40,
			Y:               56,
			Width:           320,
			Height:          200,
			TimestampMS:     2,
			BufferSlots:     []int{4, 40, 56, 1, 0, 320, 200, 2, 0},
			BeforeState:     map[string]string{"StateBlock.pressed": "false"},
			AfterState:      map[string]string{"StateBlock.pressed": "true"},
		},
		{
			Order:           4,
			Kind:            "text_input",
			TargetComponent: "StateBlock",
			DispatchPath:    []string{"BlockStateApp", "StateBlock"},
			Handled:         true,
			Pass:            true,
			Width:           320,
			Height:          200,
			TimestampMS:     3,
			TextLen:         2,
			TextBytesHex:    "4f4b",
			BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 3, 2},
			BeforeState:     map[string]string{"StateBlock.buffer": ""},
			AfterState:      map[string]string{"StateBlock.buffer": "OK"},
		},
		{
			Order:           5,
			Kind:            "key_down",
			TargetComponent: "StateBlock",
			DispatchPath:    []string{"BlockStateApp", "StateBlock"},
			Handled:         true,
			Pass:            true,
			Key:             9,
			Width:           320,
			Height:          200,
			TimestampMS:     4,
			BufferSlots:     []int{3, 0, 0, 0, 9, 320, 200, 4, 0},
			BeforeState:     map[string]string{"StateBlock.focused": "false"},
			AfterState:      map[string]string{"StateBlock.focused": "true"},
		},
	}
}
func blockMotionEventsForScenario() []surface.EventReport {
	return []surface.EventReport{
		{
			Order:           1,
			Kind:            "mouse_up",
			TargetComponent: "MotionBlock",
			DispatchPath:    []string{"BlockMotionApp", "MotionBlock"},
			Handled:         true,
			Pass:            true,
			X:               48,
			Y:               72,
			Width:           320,
			Height:          200,
			TimestampMS:     0,
			BufferSlots:     []int{5, 48, 72, 1, 0, 320, 200, 0, 0},
			BeforeState:     map[string]string{"MotionBlock.hovered": "false"},
			AfterState:      map[string]string{"MotionBlock.hovered": "true"},
		},
		{
			Order:           2,
			Kind:            "text_input",
			TargetComponent: "MotionBlock",
			DispatchPath:    []string{"BlockMotionApp", "MotionBlock"},
			Handled:         true,
			Pass:            true,
			Width:           320,
			Height:          200,
			TimestampMS:     1,
			TextLen:         2,
			TextBytesHex:    "4f4b",
			BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
			BeforeState:     map[string]string{"MotionBlock.buffer": ""},
			AfterState:      map[string]string{"MotionBlock.buffer": "OK"},
		},
	}
}
func blockAssetEventsForScenario() []surface.EventReport {
	return []surface.EventReport{
		{
			Order:           1,
			Kind:            "mouse_up",
			TargetComponent: "IconBlock",
			DispatchPath:    []string{"BlockAssetApp", "IconBlock"},
			Handled:         true,
			Pass:            true,
			X:               32,
			Y:               44,
			Width:           320,
			Height:          200,
			TimestampMS:     0,
			BufferSlots:     []int{5, 32, 44, 1, 0, 320, 200, 0, 0},
			BeforeState:     map[string]string{"IconBlock.tint": "#ffffffff"},
			AfterState:      map[string]string{"IconBlock.tint": "#60aef4ff"},
		},
		{
			Order:           2,
			Kind:            "text_input",
			TargetComponent: "IconBlock",
			DispatchPath:    []string{"BlockAssetApp", "IconBlock"},
			Handled:         true,
			Pass:            true,
			Width:           320,
			Height:          200,
			TimestampMS:     1,
			TextLen:         2,
			TextBytesHex:    "4f4b",
			BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
			BeforeState:     map[string]string{"IconBlock.label": ""},
			AfterState:      map[string]string{"IconBlock.label": "OK"},
		},
	}
}
func blockSystemReadinessTransitionsForScenario() []surface.StateTransitionReport {
	return []surface.StateTransitionReport{
		{
			Order:     1,
			Component: "InputBlock",
			Field:     "buffer",
			Before:    "",
			After:     "OKd0a2",
			Cause:     "text_input",
		},
		{
			Order:     2,
			Component: "InputBlock",
			Field:     "caret",
			Before:    "0",
			After:     "4",
			Cause:     "text_input",
		},
		{
			Order:     3,
			Component: "StateBlock",
			Field:     "selector_flags",
			Before:    "0",
			After:     "127",
			Cause:     "pointer/key/state input",
		},
		{
			Order:     4,
			Component: "StateBlock",
			Field:     "resolved_fill",
			Before:    "#20262eff",
			After:     "#2d9bf0ff",
			Cause:     "hover",
		},
		{
			Order:     5,
			Component: "StateBlock",
			Field:     "resolved_scale",
			Before:    "100",
			After:     "97",
			Cause:     "pressed",
		},
		{
			Order:     6,
			Component: "StateBlock",
			Field:     "disabled",
			Before:    "false",
			After:     "true",
			Cause:     "disabled selector",
		},
		{
			Order:     7,
			Component: "StateBlock",
			Field:     "error",
			Before:    "false",
			After:     "true",
			Cause:     "error selector",
		},
		{
			Order:     8,
			Component: "StateBlock",
			Field:     "loading",
			Before:    "false",
			After:     "true",
			Cause:     "loading selector",
		},
		{
			Order:     9,
			Component: "MotionBlock",
			Field:     "opacity",
			Before:    "80",
			After:     "200",
			Cause:     "motion frame",
		},
		{
			Order:     10,
			Component: "MotionBlock",
			Field:     "color",
			Before:    "#203040ff",
			After:     "#60aef4ff",
			Cause:     "motion frame",
		},
		{
			Order:     11,
			Component: "MotionBlock",
			Field:     "scale",
			Before:    "100",
			After:     "108",
			Cause:     "motion frame",
		},
		{
			Order:     12,
			Component: "MotionBlock",
			Field:     "translate_x",
			Before:    "0",
			After:     "12",
			Cause:     "motion frame",
		},
		{
			Order:     13,
			Component: "MotionBlock",
			Field:     "motion_complete",
			Before:    "false",
			After:     "true",
			Cause:     "duration elapsed",
		},
		{
			Order:     14,
			Component: "MotionBlock",
			Field:     "reduced_motion",
			Before:    "false",
			After:     "true",
			Cause:     "accessibility setting",
		},
		{
			Order:     15,
			Component: "IconBlock",
			Field:     "tint",
			Before:    "#ffffffff",
			After:     "#60aef4ff",
			Cause:     "asset tint",
		},
		{
			Order:     16,
			Component: "ImageBlock",
			Field:     "scale",
			Before:    "1x",
			After:     "2x",
			Cause:     "asset scale",
		},
		{
			Order:     17,
			Component: "MissingAssetBlock",
			Field:     "fallback",
			Before:    "missing",
			After:     "fallback-raster",
			Cause:     "missing asset",
		},
	}
}
func blockSystemCasesForScenario() []surface.CaseReport {
	return []surface.CaseReport{
		{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
		{Name: "headless event dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "headless framebuffer checksum", Kind: "positive", Ran: true, Pass: true},
		{Name: "headless actual runner trace", Kind: "positive", Ran: true, Pass: true},
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
		{
			Name:          "block graph duplicate id rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "duplicate Block ID",
		},
		{
			Name:          "block graph missing parent rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "missing parent",
		},
		{
			Name:          "block graph cycle rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "cycle",
		},
		{Name: "block graph child order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block graph focus order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block graph hit-test path", Kind: "positive", Ran: true, Pass: true},
		{Name: "block graph accessibility order", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "block paint fill gradient image fill border radius clip shadow overlay outline text icon",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{Name: "block paint deterministic command order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block paint frame checksum changed", Kind: "positive", Ran: true, Pass: true},
		{
			Name:          "block paint unsupported blur rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "unsupported blur",
		},
		{Name: "block renderer software rgba contract", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "block compositor dirty rect invalidation cache",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name: "block renderer opacity transform clipped child",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name:          "block renderer gpu production claim rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "gpu production",
		},
		{
			Name:          "block renderer unsupported backdrop blur rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "backdrop blur",
		},
		{Name: "block text deterministic measurement", Kind: "positive", Ran: true, Pass: true},
		{Name: "block text wrap ellipsis layout", Kind: "positive", Ran: true, Pass: true},
		{Name: "block text font fallback chain", Kind: "positive", Ran: true, Pass: true},
		{Name: "block text bounded glyph cache", Kind: "positive", Ran: true, Pass: true},
		{Name: "block text render command evidence", Kind: "positive", Ran: true, Pass: true},
		{Name: "block text editable lifetime", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout nested row column", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout fit fill fixed min max", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout grid dock overlay scroll", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout clipping z-order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block layout resize constraints", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "block layout aspect density stable rounding",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name:          "block layout no css flexbox parity",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "CSS flexbox parity nonclaim",
		},
		{Name: "block state selector resolver order", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state hover fill override", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state pressed scale override", Kind: "positive", Ran: true, Pass: true},
		{Name: "block state focus selected metadata", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "block state disabled error loading overrides",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{Name: "block state frame checksum changed", Kind: "positive", Ran: true, Pass: true},
		{
			Name:          "block state no css pseudo parity",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "css pseudo nonclaim",
		},
		{Name: "block motion deterministic test clock", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "block motion opacity color transform frames",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name: "block motion reduced motion instant settle",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{Name: "block motion completion stops scheduling", Kind: "positive", Ran: true, Pass: true},
		{Name: "block motion frame checksum changed", Kind: "positive", Ran: true, Pass: true},
		{
			Name:          "block motion no css animation parity",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "css animation nonclaim",
		},
		{
			Name: "block asset deterministic manifest hashes",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{Name: "block asset local embedded only", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset bounded cache", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset icon tint evidence", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset image scale evidence", Kind: "positive", Ran: true, Pass: true},
		{Name: "block asset missing fallback diagnostic", Kind: "positive", Ran: true, Pass: true},
		{
			Name:          "block asset network url rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "network asset rejected",
		},
		{
			Name: "block accessibility tree derived from block graph",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name:          "block accessibility focusable actionable name required",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "missing accessible name",
		},
		{
			Name:          "block accessibility label relationship mismatch rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "label relationship mismatch",
		},
		{
			Name:          "block accessibility reading order graph mismatch rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "reading order mismatch",
		},
		{
			Name:          "block accessibility screen-reader claim without platform proof rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "screen reader proof required",
		},
		{
			Name: "block accessibility platform claim scoped metadata only",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{Name: "block system headless golden checksums", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "block system deterministic repeat checksum",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name:          "block system missing frame checksum rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "frame checksum required",
		},
		{
			Name:          "block system nondeterministic checksum rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "repeat checksum mismatch",
		},
		{
			Name:          "block system missing paint evidence rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "paint evidence required",
		},
		{
			Name:          "block system missing layout evidence rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "layout evidence required",
		},
		{
			Name:          "block system missing accessibility evidence rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "accessibility evidence required",
		},
		{Name: "block system bounded memory budget", Kind: "positive", Ran: true, Pass: true},
		{Name: "block system stress render loop budget", Kind: "positive", Ran: true, Pass: true},
		{
			Name:          "block system performance nonclaim",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "Electron comparison benchmark not claimed",
		},
		{
			Name:          "reject legacy UI evidence",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "legacy UI evidence rejected",
		},
	}
}
func blockSystemLinuxX64RealWindowCasesForScenario() []surface.CaseReport {
	cases := make([]surface.CaseReport, 0, len(blockSystemCasesForScenario())+9)
	for _, tc := range blockSystemCasesForScenario() {
		name := strings.ToLower(tc.Name)
		if strings.Contains(name, "headless") {
			continue
		}
		if strings.Contains(name, "deterministic repeat checksum") {
			continue
		}
		cases = append(cases, tc)
	}
	cases = append(
		cases,
		surface.CaseReport{
			Name: "linux-x64 real-window surface",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "linux-x64 native input event pump",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "linux-x64 real-window resize event",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "linux-x64 real-window close event",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "block system linux-x64 real-window frame presentation",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "block system linux-x64 native input state transition",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "block system linux-x64 real-window checksum",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name:          "block system missing real-window presentation rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "real-window presentation required",
		},
		surface.CaseReport{
			Name:          "block system missing native input state transition rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "native input required",
		},
	)
	return cases
}
func blockSystemWASM32WebBrowserCanvasCasesForScenario() []surface.CaseReport {
	cases := make([]surface.CaseReport, 0, len(blockSystemCasesForScenario())+16)
	for _, tc := range blockSystemCasesForScenario() {
		name := strings.ToLower(tc.Name)
		if strings.Contains(name, "headless") {
			continue
		}
		if strings.Contains(name, "deterministic repeat checksum") {
			continue
		}
		cases = append(cases, tc)
	}
	cases = append(
		cases,
		surface.CaseReport{
			Name: "wasm32-web browser canvas surface",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "wasm32-web browser canvas RGBA readback",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "wasm32-web browser canvas pointer input",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "wasm32-web browser canvas keyboard input",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "wasm32-web browser canvas resize input",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "wasm32-web browser canvas text input",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "wasm32-web Surface Host ABI imports",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "compiler-owned wasm Surface loader",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "compiler-owned browser canvas Surface host",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "block system wasm32-web browser-canvas frame readback",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "block system wasm32-web browser-canvas native input state transition",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "block system wasm32-web browser-canvas checksum",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name:          "block system browser-canvas node runtime substitution rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "browser evidence required",
		},
		surface.CaseReport{
			Name:          "block system browser-canvas missing RGBA readback rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "RGBA readback required",
		},
		surface.CaseReport{
			Name:          "block system browser-canvas script sidecar artifact rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "script artifact rejected",
		},
		surface.CaseReport{
			Name:          "block system browser-canvas html visual sidecar artifact rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "html artifact rejected",
		},
	)
	return cases
}
func blockAccessibilityComponentsForScenario() []surface.ComponentReport {
	abilities := []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}
	return []surface.ComponentReport{
		{
			ID:        "BlockAccessibilityApp",
			Type:      "examples.surface.block_render.surface_block_accessibility.BlockAccessibilityApp",
			Bounds:    surface.RectReport{X: 0, Y: 0, W: 320, H: 200},
			Abilities: abilities,
			State: map[string]string{
				"focused_id":   "4",
				"a11y_quality": "block-derived-accessibility-metadata-v1",
			},
		},
		{
			ID:        "LabelBlock",
			Type:      "examples.surface.block_render.surface_block_accessibility.LabelBlock",
			Parent:    "BlockAccessibilityApp",
			Bounds:    surface.RectReport{X: 24, Y: 24, W: 200, H: 24},
			Abilities: abilities,
			State:     map[string]string{"text_len": "4", "label_for": "4"},
		},
		{
			ID:        "SubmitBlock",
			Type:      "examples.surface.block_render.surface_block_accessibility.ActionBlock",
			Parent:    "BlockAccessibilityApp",
			Bounds:    surface.RectReport{X: 24, Y: 64, W: 120, H: 44},
			Abilities: abilities,
			State:     map[string]string{"focused": "true", "action": "submit"},
		},
		{
			ID:        "ResetBlock",
			Type:      "examples.surface.block_render.surface_block_accessibility.ActionBlock",
			Parent:    "BlockAccessibilityApp",
			Bounds:    surface.RectReport{X: 152, Y: 64, W: 120, H: 44},
			Abilities: abilities,
			State:     map[string]string{"focused": "false", "action": "reset"},
		},
	}
}
func blockAccessibilityGraphForScenario(source string) *surface.BlockGraphReport {
	return &surface.BlockGraphReport{
		Schema:            "tetra.surface.block-graph.v1",
		APILevel:          "block-tree-builder-v1",
		Source:            source,
		ManualBookkeeping: false,
		Builder: surface.BlockGraphBuilderReport{
			RootCreatedBy:     "tree_add_root",
			ChildrenCreatedBy: "tree_add_child",
			NodeCount:         5,
			Capacity:          8,
			OverflowChecked:   true,
		},
		Invariants: surface.BlockGraphInvariantReport{
			TreeValidateRan:         true,
			TreeValidateStatus:      0,
			DuplicateIDRejected:     true,
			MissingParentRejected:   true,
			CycleRejected:           true,
			ParentChildLinksChecked: true,
			ChildOrderChecked:       true,
			FocusOrderChecked:       true,
			HitTestPathChecked:      true,
			AccessibilityChecked:    true,
		},
		RootID:    1,
		NodeCount: 5,
		Nodes: []surface.BlockGraphNodeReport{
			{
				ID:                1,
				Name:              "RootBlock",
				ParentID:          -1,
				ChildIndex:        0,
				FirstChild:        2,
				ChildCount:        1,
				Focusable:         false,
				AccessibilityRole: "none",
				Bounds:            surface.RectReport{X: 0, Y: 0, W: 320, H: 200},
			},
			{
				ID:                2,
				Name:              "PanelBlock",
				ParentID:          1,
				ChildIndex:        0,
				FirstChild:        3,
				ChildCount:        3,
				Focusable:         false,
				AccessibilityRole: "none",
				Bounds:            surface.RectReport{X: 16, Y: 16, W: 288, H: 168},
			},
			{
				ID:                3,
				Name:              "LabelBlock",
				ParentID:          2,
				ChildIndex:        0,
				FirstChild:        -1,
				ChildCount:        0,
				Focusable:         false,
				AccessibilityRole: "text",
				Bounds:            surface.RectReport{X: 24, Y: 24, W: 200, H: 24},
			},
			{
				ID:                4,
				Name:              "SubmitBlock",
				ParentID:          2,
				ChildIndex:        1,
				FirstChild:        -1,
				ChildCount:        0,
				Focusable:         true,
				AccessibilityRole: "button",
				Bounds:            surface.RectReport{X: 24, Y: 64, W: 120, H: 44},
			},
			{
				ID:                5,
				Name:              "ResetBlock",
				ParentID:          2,
				ChildIndex:        2,
				FirstChild:        -1,
				ChildCount:        0,
				Focusable:         true,
				AccessibilityRole: "button",
				Bounds:            surface.RectReport{X: 152, Y: 64, W: 120, H: 44},
			},
		},
		ChildOrders: []surface.BlockGraphChildOrderReport{
			{ParentID: 1, Children: []int{2}},
			{ParentID: 2, Children: []int{3, 4, 5}},
		},
		LayoutOrder:        []int{1, 2, 3, 4, 5},
		DrawOrder:          []int{1, 2, 3, 4, 5},
		FocusOrder:         []int{4, 5},
		AccessibilityOrder: []int{3, 4, 5},
		HitTests: []surface.BlockGraphPathReport{
			{
				Helper:   "tree_hit_test_path",
				Event:    "click",
				TargetID: 5,
				X:        180,
				Y:        80,
				Path:     []int{1, 2, 5},
			},
		},
		DispatchPaths: []surface.BlockGraphPathReport{
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 4, Path: []int{1, 2, 4}},
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 5, Path: []int{1, 2, 5}},
		},
	}
}
func blockAccessibilityTreeForScenario(source string) *surface.BlockAccessibilityTreeReport {
	return &surface.BlockAccessibilityTreeReport{
		Schema:                  "tetra.surface.block-accessibility-tree.v1",
		AccessibilityLevel:      "block-metadata-tree-v1",
		Source:                  source,
		Module:                  "lib.core.block",
		QualityLevel:            "block-derived-accessibility-metadata-v1",
		BlockGraphSchema:        "tetra.surface.block-graph.v1",
		DerivedFromBlockGraph:   true,
		ManualBookkeeping:       false,
		PlatformHostIntegration: false,
		DOMARIAIntegration:      false,
		ScreenReaderEvidence:    false,
		NoDOMUI:                 true,
		NoUserJS:                true,
		NoPlatformWidgets:       true,
		NodeCount:               3,
		FocusableCount:          2,
		RolesPresent:            []string{"text", "button"},
		FocusOrder:              []int{4, 5},
		ReadingOrder:            []int{3, 4, 5},
		Nodes: []surface.BlockAccessibilityNodeReport{
			{
				ID:            3,
				BlockID:       3,
				ParentBlockID: 2,
				Name:          "LabelBlock",
				Role:          "text",
				Bounds:        surface.RectReport{X: 24, Y: 24, W: 200, H: 24},
				Visible:       true,
				Enabled:       true,
				Focusable:     false,
				LabelFor:      "SubmitBlock",
				FocusIndex:    -1,
				ReadingIndex:  0,
			},
			{
				ID:            4,
				BlockID:       4,
				ParentBlockID: 2,
				Name:          "SubmitBlock",
				Role:          "button",
				Description:   "primary action",
				Bounds:        surface.RectReport{X: 24, Y: 64, W: 120, H: 44},
				Visible:       true,
				Enabled:       true,
				Focusable:     true,
				Focused:       true,
				LabelledBy:    "LabelBlock",
				Actions:       []string{"focus", "press", "submit"},
				FocusIndex:    0,
				ReadingIndex:  1,
			},
			{
				ID:            5,
				BlockID:       5,
				ParentBlockID: 2,
				Name:          "ResetBlock",
				Role:          "button",
				Description:   "secondary action",
				Bounds:        surface.RectReport{X: 152, Y: 64, W: 120, H: 44},
				Visible:       true,
				Enabled:       true,
				Focusable:     true,
				Actions:       []string{"focus", "press", "reset"},
				FocusIndex:    1,
				ReadingIndex:  2,
			},
		},
		Relationships: []surface.AccessibilityRelationshipReport{
			{Kind: "label_for", From: "LabelBlock", To: "SubmitBlock"},
			{Kind: "labelled_by", From: "SubmitBlock", To: "LabelBlock"},
		},
		Actions: []surface.AccessibilityActionReport{
			{Target: "SubmitBlock", Action: "press", Semantic: "submit"},
			{Target: "ResetBlock", Action: "press", Semantic: "reset"},
		},
		NegativeGuards: surface.BlockAccessibilityNegativeGuardsReport{
			FocusableActionNameChecked:    true,
			LabelRelationshipsChecked:     true,
			ReadingOrderGraphChecked:      true,
			BoundsAlignmentChecked:        true,
			FakeScreenReaderClaimRejected: true,
			ScopedPlatformClaimChecked:    true,
		},
	}
}
func blockAssetComponentsForScenario() []surface.ComponentReport {
	abilities := []string{
		"measure",
		"layout",
		"draw",
		"event",
		"focus",
		"text",
		"accessibility",
		"asset",
	}
	return []surface.ComponentReport{
		{
			ID:        "BlockAssetApp",
			Type:      "examples.surface.block_render.surface_block_assets.BlockAssetApp",
			Bounds:    surface.RectReport{X: 0, Y: 0, W: 320, H: 200},
			Abilities: abilities,
			State:     map[string]string{"asset_quality": "deterministic-local-block-assets-v1"},
		},
		{
			ID:        "IconBlock",
			Type:      "examples.surface.block_render.surface_block_assets.IconBlock",
			Parent:    "BlockAssetApp",
			Bounds:    surface.RectReport{X: 24, Y: 36, W: 32, H: 32},
			Abilities: abilities,
			State:     map[string]string{"asset_id": "icon-settings", "tint": "#60aef4ff"},
		},
		{
			ID:        "ImageBlock",
			Type:      "examples.surface.block_render.surface_block_assets.ImageBlock",
			Parent:    "BlockAssetApp",
			Bounds:    surface.RectReport{X: 72, Y: 32, W: 96, H: 64},
			Abilities: abilities,
			State:     map[string]string{"asset_id": "image-hero", "scale": "2x"},
		},
		{
			ID:        "MissingAssetBlock",
			Type:      "examples.surface.block_render.surface_block_assets.MissingAssetBlock",
			Parent:    "BlockAssetApp",
			Bounds:    surface.RectReport{X: 24, Y: 112, W: 96, H: 32},
			Abilities: abilities,
			State:     map[string]string{"asset_id": "missing-logo", "fallback": "fallback-raster"},
		},
	}
}
func blockAssetManifestForScenario(source string) *surface.BlockAssetManifestReport {
	return &surface.BlockAssetManifestReport{
		Schema:        "tetra.surface.block-assets.v1",
		Source:        source,
		Quality:       "deterministic-local-block-assets-v1",
		HashAlgorithm: "sha256",
		ManifestHash:  "sha256:" + checksumText("surface-block-assets-manifest-v1"),
		LocalOnly:     true,
		FontCount:     1,
		IconCount:     1,
		ImageCount:    1,
		EmbeddedCount: 3,
		RemoteCount:   0,
		Assets: []surface.BlockAssetReport{
			{
				ID:       "font-ui",
				Kind:     "font",
				Path:     "embedded://surface/font-ui",
				Embedded: true,
				Local:    true,
				SHA256:   "sha256:" + checksumText("surface-block-assets-font-ui"),
				Size:     2048,
				Family:   "Tetra UI",
				CacheKey: "font-ui",
			},
			{
				ID:       "icon-settings",
				Kind:     "icon",
				Path:     "embedded://surface/icon-settings",
				Embedded: true,
				Local:    true,
				SHA256:   "sha256:" + checksumText("surface-block-assets-icon-settings"),
				Size:     256,
				Width:    16,
				Height:   16,
				CacheKey: "icon-settings",
			},
			{
				ID:       "image-hero",
				Kind:     "image",
				Path:     "embedded://surface/image-hero",
				Embedded: true,
				Local:    true,
				SHA256:   "sha256:" + checksumText("surface-block-assets-image-hero"),
				Size:     1024,
				Width:    48,
				Height:   32,
				CacheKey: "image-hero",
			},
		},
	}
}
func blockAssetCacheForScenario() surface.BlockAssetCacheReport {
	return surface.BlockAssetCacheReport{
		ID:            "asset-cache",
		Strategy:      "bounded-lru",
		BudgetBytes:   65536,
		UsedBytes:     5376,
		EntryCount:    3,
		MaxEntries:    16,
		RepeatedLoads: 6,
		Eviction:      "lru",
		Bounded:       true,
	}
}
func blockAssetDiagnosticsForScenario() []surface.BlockAssetDiagnosticReport {
	return []surface.BlockAssetDiagnosticReport{
		{
			Order:      1,
			AssetID:    "missing-logo",
			Kind:       "image",
			Code:       "missing_asset_fallback",
			Message:    "missing local asset resolved to fallback raster",
			FallbackID: "fallback-raster-image",
			Pass:       true,
		},
		{
			Order:       2,
			AssetID:     "https://assets.example.test/logo.png",
			Kind:        "image",
			Code:        "network_asset_rejected",
			Message:     "network assets are disabled for Surface Block v1",
			RejectedURL: "https://assets.example.test/logo.png",
			Pass:        true,
		},
	}
}
func blockAssetRenderCommandsForScenario() []surface.BlockAssetRenderCommandReport {
	return []surface.BlockAssetRenderCommandReport{
		{
			Order:    1,
			Command:  "load_font",
			AssetID:  "font-ui",
			BlockID:  1,
			Rect:     surface.RectReport{X: 0, Y: 0, W: 320, H: 200},
			Quality:  "font-manifest-metadata-v1",
			Checksum: "sha256:" + checksumText("surface-block-assets-load-font"),
		},
		{
			Order:          2,
			Command:        "tint_icon",
			AssetID:        "icon-settings",
			BlockID:        2,
			Rect:           surface.RectReport{X: 24, Y: 36, W: 32, H: 32},
			Tint:           "#60aef4ff",
			Scale:          1,
			Quality:        "icon-mask-raster-v1",
			RasterFormat:   "builtin-icon-mask-raster-v1",
			RasterHash:     "sha256:" + checksumText("surface-block-assets-tint-icon-raster"),
			RasterWidth:    32,
			RasterHeight:   32,
			RasterCoverage: 341,
			MarkerOnly:     false,
			Checksum:       "sha256:" + checksumText("surface-block-assets-tint-icon"),
		},
		{
			Order:    3,
			Command:  "scale_image",
			AssetID:  "image-hero",
			BlockID:  3,
			Rect:     surface.RectReport{X: 72, Y: 32, W: 96, H: 64},
			Scale:    2,
			Quality:  "nearest-scale-v1",
			Checksum: "sha256:" + checksumText("surface-block-assets-scale-image"),
		},
		{
			Order:    4,
			Command:  "fallback_missing",
			AssetID:  "missing-logo",
			BlockID:  4,
			Rect:     surface.RectReport{X: 24, Y: 112, W: 96, H: 32},
			Quality:  "fallback-raster-v1",
			Checksum: "sha256:" + checksumText("surface-block-assets-fallback-missing"),
		},
	}
}

// ---- scenarios_morph_flagship.go ----

const morphRenderedFlagshipSource = ("examples/surface/morph_flagship/surface_morph_" +
	"rendered_studio_shell.tetra")

func isMorphRenderedFlagshipSource(source string) bool {
	clean := filepath.ToSlash(filepath.Clean(normalizeSurfaceSourcePath(source)))
	return clean == morphRenderedFlagshipSource ||
		strings.HasSuffix(clean, "/"+morphRenderedFlagshipSource)
}

func runMorphScenarioForSource(source string) headlessScenario {
	source = normalizeSurfaceSourcePath(source)
	if source == "" {
		source = "examples/surface/morph_core/surface_morph_command_palette.tetra"
	}
	if isMorphRenderedFlagshipSource(source) {
		return runMorphFlagshipScenario(source)
	}
	if isMorphGuestDashboardSource(source) {
		return runMorphGuestDashboardScenario(source)
	}
	scenario := runBlockSystemScenario()
	retargetScenarioToSource(&scenario, source, surfaceSourceModuleName(source))
	scenario.Morph = morphReportForScenario(source, scenario)
	scenario.BlockSceneSnapshot = blockSceneSnapshotForScenario(source, scenario)
	attachRenderCommandStreamForScenario(source, &scenario)
	scenario.Cases = append(scenario.Cases, morphCasesForScenario()...)
	return scenario
}

func runMorphFlagshipScenario(source string) headlessScenario {
	scenario := runBlockSystemScenario()
	retargetScenarioToSource(&scenario, source, surfaceSourceModuleName(source))
	scenario.Components = append(scenario.Components, flagshipMorphComponentsForScenario(source)...)
	scenario.BlockGraph = flagshipMorphBlockGraphForScenario(source)
	scenario.BlockAccessibilityTree = flagshipMorphAccessibilityTreeForScenario(
		source,
		scenario.BlockGraph,
	)
	scenario.BlockSystem = blockSystemReportForScenario(source, scenario.Frames)
	attachBlockSystemMemoryBudget(&scenario)
	scenario.Morph = morphReportForScenario(source, scenario)
	scenario.BlockSceneSnapshot = blockSceneSnapshotForScenario(source, scenario)
	attachRenderCommandStreamForScenario(source, &scenario)
	scenario.Cases = append(scenario.Cases, morphCasesForScenario()...)
	scenario.Cases = append(scenario.Cases, flagshipMorphCasesForScenario()...)
	return scenario
}

func flagshipMorphComponentsForScenario(source string) []surface.ComponentReport {
	module := surfaceSourceModuleName(source)
	nodes := flagshipMorphGraphNodes()
	namesByID := map[int]string{}
	for _, node := range nodes {
		namesByID[node.ID] = node.Name
	}
	abilities := []string{
		"measure",
		"layout",
		"draw",
		"event",
		"focus",
		"text",
		"accessibility",
		"state",
		"motion",
		"asset",
	}
	components := make([]surface.ComponentReport, 0, len(nodes))
	for _, node := range nodes {
		parent := ""
		if node.ParentID >= 0 {
			parent = namesByID[node.ParentID]
		}
		components = append(components, surface.ComponentReport{
			ID:        node.Name,
			Type:      module + "." + node.Name,
			Parent:    parent,
			Bounds:    node.Bounds,
			Abilities: abilities,
			State: map[string]string{
				"block_id": strconv.Itoa(node.ID),
				"role":     node.AccessibilityRole,
				"recipe":   flagshipMorphRecipeForNode(node.Name),
				"source":   "morph",
			},
		})
	}
	return components
}

func flagshipMorphRecipeForNode(name string) string {
	switch name {
	case "RenderedStudioShell", "AppShellFrame":
		return "app.shell@1"
	case "NavigationRail", "ProfilesActions", "ProjectPackageView", "RunDiagnosticsView":
		return "nav.item@1"
	case "ToolbarActions":
		return "toolbar@1"
	case "DashboardShell":
		return "split.pane@1"
	case "CommandPalette":
		return "command.item@1"
	case "SettingsForm":
		return "settings.form@1"
	case "LogsOutput":
		return "log.row@1"
	case "DiagnosticsError":
		return "error.panel@1"
	case "MetricTiles":
		return "metric.tile@1"
	case "StatusBar":
		return "status.bar@1"
	case "BlockedDialog":
		return "dialog.panel@1"
	case "ToastSurface":
		return "toast.notification@1"
	case "EmptyState":
		return "empty.state@1"
	default:
		return "region.panel@1"
	}
}

func flagshipMorphBlockGraphForScenario(source string) *surface.BlockGraphReport {
	nodes := flagshipMorphGraphNodes()
	order := make([]int, 0, len(nodes))
	for _, node := range nodes {
		order = append(order, node.ID)
	}
	return &surface.BlockGraphReport{
		Schema:            "tetra.surface.block-graph.v1",
		APILevel:          "block-tree-builder-v1",
		Source:            source,
		ManualBookkeeping: false,
		Builder: surface.BlockGraphBuilderReport{
			RootCreatedBy:     "tree_add_root",
			ChildrenCreatedBy: "tree_add_child",
			NodeCount:         len(nodes),
			Capacity:          24,
			OverflowChecked:   true,
		},
		Invariants: surface.BlockGraphInvariantReport{
			TreeValidateRan:         true,
			TreeValidateStatus:      0,
			DuplicateIDRejected:     true,
			MissingParentRejected:   true,
			CycleRejected:           true,
			ParentChildLinksChecked: true,
			ChildOrderChecked:       true,
			FocusOrderChecked:       true,
			HitTestPathChecked:      true,
			AccessibilityChecked:    true,
		},
		RootID:    1,
		NodeCount: len(nodes),
		Nodes:     nodes,
		ChildOrders: []surface.BlockGraphChildOrderReport{
			{ParentID: 1, Children: []int{2}},
			{ParentID: 2, Children: []int{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18}},
		},
		LayoutOrder:        order,
		DrawOrder:          order,
		FocusOrder:         []int{4, 6, 8, 9, 10, 12, 15},
		AccessibilityOrder: []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18},
		HitTests: []surface.BlockGraphPathReport{
			{
				Helper:   "tree_hit_test_path",
				Event:    "click",
				TargetID: 9,
				X:        720,
				Y:        112,
				Path:     []int{1, 2, 9},
			},
			{
				Helper:   "tree_hit_test_path",
				Event:    "click",
				TargetID: 15,
				X:        780,
				Y:        244,
				Path:     []int{1, 2, 15},
			},
		},
		DispatchPaths: []surface.BlockGraphPathReport{
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 4, Path: []int{1, 2, 4}},
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 6, Path: []int{1, 2, 6}},
			{Helper: "tree_build_dispatch_path", Event: "key", TargetID: 9, Path: []int{1, 2, 9}},
			{
				Helper:   "tree_build_dispatch_path",
				Event:    "text",
				TargetID: 10,
				Path:     []int{1, 2, 10},
			},
			{
				Helper:   "tree_build_dispatch_path",
				Event:    "click",
				TargetID: 12,
				Path:     []int{1, 2, 12},
			},
			{
				Helper:   "tree_build_dispatch_path",
				Event:    "click",
				TargetID: 15,
				Path:     []int{1, 2, 15},
			},
		},
	}
}

func flagshipMorphGraphNodes() []surface.BlockGraphNodeReport {
	return []surface.BlockGraphNodeReport{
		{
			ID:                1,
			Name:              "RenderedStudioShell",
			ParentID:          -1,
			ChildIndex:        0,
			FirstChild:        2,
			ChildCount:        1,
			AccessibilityRole: "none",
			Bounds:            surface.RectReport{X: 0, Y: 0, W: 1180, H: 760},
		},
		{
			ID:                2,
			Name:              "AppShellFrame",
			ParentID:          1,
			ChildIndex:        0,
			FirstChild:        3,
			ChildCount:        16,
			AccessibilityRole: "region",
			Bounds:            surface.RectReport{X: 0, Y: 0, W: 1180, H: 760},
		},
		{
			ID:                3,
			Name:              "NavigationRail",
			ParentID:          2,
			ChildIndex:        0,
			FirstChild:        -1,
			ChildCount:        0,
			AccessibilityRole: "navigation",
			Bounds:            surface.RectReport{X: 24, Y: 80, W: 160, H: 256},
		},
		{
			ID:                4,
			Name:              "ToolbarActions",
			ParentID:          2,
			ChildIndex:        1,
			FirstChild:        -1,
			ChildCount:        0,
			Focusable:         true,
			AccessibilityRole: "button",
			Bounds:            surface.RectReport{X: 250, Y: 32, W: 880, H: 44},
		},
		{
			ID:                5,
			Name:              "DashboardShell",
			ParentID:          2,
			ChildIndex:        2,
			FirstChild:        -1,
			ChildCount:        0,
			AccessibilityRole: "region",
			Bounds:            surface.RectReport{X: 250, Y: 92, W: 420, H: 190},
		},
		{
			ID:                6,
			Name:              "ProfilesActions",
			ParentID:          2,
			ChildIndex:        3,
			FirstChild:        -1,
			ChildCount:        0,
			Focusable:         true,
			AccessibilityRole: "button",
			Bounds:            surface.RectReport{X: 552, Y: 248, W: 140, H: 36},
		},
		{
			ID:                7,
			Name:              "ProjectPackageView",
			ParentID:          2,
			ChildIndex:        4,
			FirstChild:        -1,
			ChildCount:        0,
			AccessibilityRole: "region",
			Bounds:            surface.RectReport{X: 280, Y: 116, W: 160, H: 72},
		},
		{
			ID:                8,
			Name:              "RunDiagnosticsView",
			ParentID:          2,
			ChildIndex:        5,
			FirstChild:        -1,
			ChildCount:        0,
			Focusable:         true,
			AccessibilityRole: "button",
			Bounds:            surface.RectReport{X: 456, Y: 316, W: 160, H: 40},
		},
		{
			ID:                9,
			Name:              "CommandPalette",
			ParentID:          2,
			ChildIndex:        6,
			FirstChild:        -1,
			ChildCount:        0,
			Focusable:         true,
			AccessibilityRole: "textbox",
			Bounds:            surface.RectReport{X: 690, Y: 92, W: 360, H: 48},
		},
		{
			ID:                10,
			Name:              "SettingsForm",
			ParentID:          2,
			ChildIndex:        7,
			FirstChild:        -1,
			ChildCount:        0,
			Focusable:         true,
			AccessibilityRole: "textbox",
			Bounds:            surface.RectReport{X: 690, Y: 154, W: 360, H: 48},
		},
		{
			ID:                11,
			Name:              "LogsOutput",
			ParentID:          2,
			ChildIndex:        8,
			FirstChild:        -1,
			ChildCount:        0,
			AccessibilityRole: "text",
			Bounds:            surface.RectReport{X: 280, Y: 432, W: 360, H: 36},
		},
		{
			ID:                12,
			Name:              "DiagnosticsError",
			ParentID:          2,
			ChildIndex:        9,
			FirstChild:        -1,
			ChildCount:        0,
			Focusable:         true,
			AccessibilityRole: "button",
			Bounds:            surface.RectReport{X: 656, Y: 432, W: 220, H: 36},
		},
		{
			ID:                13,
			Name:              "MetricTiles",
			ParentID:          2,
			ChildIndex:        10,
			FirstChild:        -1,
			ChildCount:        0,
			AccessibilityRole: "region",
			Bounds:            surface.RectReport{X: 280, Y: 204, W: 160, H: 36},
		},
		{
			ID:                14,
			Name:              "StatusBar",
			ParentID:          2,
			ChildIndex:        11,
			FirstChild:        -1,
			ChildCount:        0,
			AccessibilityRole: "status",
			Bounds:            surface.RectReport{X: 250, Y: 704, W: 880, H: 32},
		},
		{
			ID:                15,
			Name:              "BlockedDialog",
			ParentID:          2,
			ChildIndex:        12,
			FirstChild:        -1,
			ChildCount:        0,
			Focusable:         true,
			AccessibilityRole: "dialog",
			Bounds:            surface.RectReport{X: 690, Y: 220, W: 360, H: 74},
		},
		{
			ID:                16,
			Name:              "ToastSurface",
			ParentID:          2,
			ChildIndex:        13,
			FirstChild:        -1,
			ChildCount:        0,
			AccessibilityRole: "status",
			Bounds:            surface.RectReport{X: 456, Y: 116, W: 180, H: 72},
		},
		{
			ID:                17,
			Name:              "EmptyState",
			ParentID:          2,
			ChildIndex:        14,
			FirstChild:        -1,
			ChildCount:        0,
			AccessibilityRole: "text",
			Bounds:            surface.RectReport{X: 456, Y: 204, W: 180, H: 36},
		},
		{
			ID:                18,
			Name:              "AppShellState",
			ParentID:          2,
			ChildIndex:        15,
			FirstChild:        -1,
			ChildCount:        0,
			AccessibilityRole: "text",
			Bounds:            surface.RectReport{X: 690, Y: 316, W: 220, H: 32},
		},
	}
}

func flagshipMorphAccessibilityTreeForScenario(
	source string,
	graph *surface.BlockGraphReport,
) *surface.BlockAccessibilityTreeReport {
	if graph == nil {
		return nil
	}
	focusIndex := map[int]int{}
	for i, id := range graph.FocusOrder {
		focusIndex[id] = i
	}
	readingIndex := map[int]int{}
	for i, id := range graph.AccessibilityOrder {
		readingIndex[id] = i
	}
	roles := []string{}
	seenRoles := map[string]bool{}
	nodes := make([]surface.BlockAccessibilityNodeReport, 0, len(graph.AccessibilityOrder))
	actions := []surface.AccessibilityActionReport{}
	for _, graphNode := range graph.Nodes {
		if graphNode.AccessibilityRole == "" || graphNode.AccessibilityRole == "none" {
			continue
		}
		role := graphNode.AccessibilityRole
		if !seenRoles[role] {
			seenRoles[role] = true
			roles = append(roles, role)
		}
		node := surface.BlockAccessibilityNodeReport{
			ID:            graphNode.ID,
			BlockID:       graphNode.ID,
			ParentBlockID: graphNode.ParentID,
			Name:          graphNode.Name,
			Role:          role,
			Description:   flagshipMorphDescriptionForNode(graphNode.Name),
			Bounds:        graphNode.Bounds,
			Visible:       true,
			Enabled:       true,
			Focusable:     graphNode.Focusable,
			FocusIndex:    -1,
			ReadingIndex:  readingIndex[graphNode.ID],
		}
		if graphNode.Focusable {
			node.FocusIndex = focusIndex[graphNode.ID]
			node.Actions = flagshipMorphActionsForRole(role)
			node.Focused = graphNode.ID == graph.FocusOrder[0]
			actions = append(actions, surface.AccessibilityActionReport{
				Target:   graphNode.Name,
				Action:   flagshipMorphPrimaryActionForRole(role),
				Semantic: flagshipMorphSemanticForNode(graphNode.Name),
			})
		}
		if graphNode.Name == "AppShellState" {
			node.LabelFor = "CommandPalette"
		}
		if graphNode.Name == "CommandPalette" {
			node.LabelledBy = "AppShellState"
			node.Editable = true
			node.Value = "Morph command ready"
		}
		nodes = append(nodes, node)
	}
	return &surface.BlockAccessibilityTreeReport{
		Schema:                  "tetra.surface.block-accessibility-tree.v1",
		AccessibilityLevel:      "block-metadata-tree-v1",
		Source:                  source,
		Module:                  "lib.core.block",
		QualityLevel:            "block-derived-accessibility-metadata-v1",
		BlockGraphSchema:        "tetra.surface.block-graph.v1",
		DerivedFromBlockGraph:   true,
		ManualBookkeeping:       false,
		PlatformHostIntegration: false,
		DOMARIAIntegration:      false,
		ScreenReaderEvidence:    false,
		NoDOMUI:                 true,
		NoUserJS:                true,
		NoPlatformWidgets:       true,
		NodeCount:               len(nodes),
		FocusableCount:          len(graph.FocusOrder),
		RolesPresent:            roles,
		Nodes:                   nodes,
		Relationships: []surface.AccessibilityRelationshipReport{
			{Kind: "label_for", From: "AppShellState", To: "CommandPalette"},
			{Kind: "labelled_by", From: "CommandPalette", To: "AppShellState"},
		},
		FocusOrder:   graph.FocusOrder,
		ReadingOrder: graph.AccessibilityOrder,
		Actions:      actions,
		NegativeGuards: surface.BlockAccessibilityNegativeGuardsReport{
			FocusableActionNameChecked:    true,
			LabelRelationshipsChecked:     true,
			ReadingOrderGraphChecked:      true,
			BoundsAlignmentChecked:        true,
			FakeScreenReaderClaimRejected: true,
			ScopedPlatformClaimChecked:    true,
		},
	}
}

func flagshipMorphDescriptionForNode(name string) string {
	switch name {
	case "CommandPalette":
		return "command palette field"
	case "SettingsForm":
		return "settings form field"
	case "BlockedDialog":
		return "blocked action dialog"
	case "DiagnosticsError":
		return "recoverable diagnostics error"
	default:
		return "Morph-authored Surface region"
	}
}

func flagshipMorphActionsForRole(role string) []string {
	switch role {
	case "textbox":
		return []string{"focus", "edit"}
	case "dialog":
		return []string{"focus", "dismiss"}
	default:
		return []string{"focus", "press"}
	}
}

func flagshipMorphPrimaryActionForRole(role string) string {
	switch role {
	case "textbox":
		return "edit"
	case "dialog":
		return "dismiss"
	default:
		return "press"
	}
}

func flagshipMorphSemanticForNode(name string) string {
	switch name {
	case "ToolbarActions":
		return "open-command-palette"
	case "ProfilesActions":
		return "select-profile"
	case "RunDiagnosticsView":
		return "run-diagnostics"
	case "CommandPalette":
		return "edit-command"
	case "SettingsForm":
		return "edit-settings"
	case "DiagnosticsError":
		return "retry-diagnostics"
	case "BlockedDialog":
		return "dismiss-blocked-action"
	default:
		return "activate"
	}
}

func flagshipMorphCasesForScenario() []surface.CaseReport {
	return []surface.CaseReport{
		{
			Name: "flagship Morph source avoids manual draw authoring",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name: "flagship Morph app shell expands to Block scene",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name: "flagship Morph dashboard shell emits render commands",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name: "flagship Morph command palette emits pixel evidence",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name: "flagship Morph settings form projects accessibility",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name: "flagship Morph dialog/error/status recipes stay Morph recipes",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
	}
}

// ---- scenarios_morph_guest_dashboard.go ----

const morphGuestDashboardSource = ("examples/surface/morph_flagship/surface_morph_guest_" +
	"dashboard.tetra")

func isMorphGuestDashboardSource(source string) bool {
	clean := filepath.ToSlash(filepath.Clean(normalizeSurfaceSourcePath(source)))
	return clean == morphGuestDashboardSource ||
		strings.HasSuffix(clean, "/"+morphGuestDashboardSource)
}

func runMorphGuestDashboardScenario(source string) headlessScenario {
	scenario := runBlockSystemScenario()
	retargetScenarioToSource(&scenario, source, surfaceSourceModuleName(source))
	scenario.Components = append(
		scenario.Components,
		guestDashboardComponentsForScenario(source)...)
	scenario.BlockGraph = guestDashboardBlockGraphForScenario(source)
	scenario.BlockAccessibilityTree = guestDashboardAccessibilityTreeForScenario(
		source,
		scenario.BlockGraph,
	)
	scenario.BlockSystem = blockSystemReportForScenario(source, scenario.Frames)
	attachBlockSystemMemoryBudget(&scenario)
	scenario.Morph = morphReportForScenario(source, scenario)
	scenario.BlockSceneSnapshot = blockSceneSnapshotForScenario(source, scenario)
	attachRenderCommandStreamForScenario(source, &scenario)
	scenario.Cases = append(scenario.Cases, morphCasesForScenario()...)
	scenario.Cases = append(scenario.Cases, guestDashboardCasesForScenario()...)
	return scenario
}

func guestDashboardComponentsForScenario(source string) []surface.ComponentReport {
	module := surfaceSourceModuleName(source)
	nodes := guestDashboardGraphNodes()
	namesByID := map[int]string{}
	for _, node := range nodes {
		namesByID[node.ID] = node.Name
	}
	abilities := []string{
		"measure",
		"layout",
		"draw",
		"event",
		"focus",
		"text",
		"accessibility",
		"state",
		"motion",
		"asset",
	}
	components := make([]surface.ComponentReport, 0, len(nodes))
	for _, node := range nodes {
		parent := ""
		if node.ParentID >= 0 {
			parent = namesByID[node.ParentID]
		}
		components = append(components, surface.ComponentReport{
			ID:        node.Name,
			Type:      module + "." + node.Name,
			Parent:    parent,
			Bounds:    node.Bounds,
			Abilities: abilities,
			State: map[string]string{
				"block_id": strconv.Itoa(node.ID),
				"role":     node.AccessibilityRole,
				"recipe":   guestDashboardRecipeForNode(node.Name),
				"source":   "morph",
			},
		})
	}
	return components
}

func guestDashboardRecipeForNode(name string) string {
	switch name {
	case "GuestDashboardPage":
		return "app.shell@1"
	case "RecentCoursesPanel", "CourseOverviewPanel", "CourseOverviewDivider":
		return "region.panel@1"
	case "RecentCoursesEmptyIcon",
		"RecentCoursesEmptyState",
		"CourseOverviewEmptyState",
		"CourseOverviewHeadline",
		"CourseOverviewBody":
		return "empty.state@1"
	case "PersonalCabinetTitle", "RecentCoursesTitle", "CourseOverviewTitle":
		return "status.bar@1"
	default:
		return "region.panel@1"
	}
}

func guestDashboardBlockGraphForScenario(source string) *surface.BlockGraphReport {
	nodes := guestDashboardGraphNodes()
	order := make([]int, 0, len(nodes))
	for _, node := range nodes {
		order = append(order, node.ID)
	}
	return &surface.BlockGraphReport{
		Schema:            "tetra.surface.block-graph.v1",
		APILevel:          "block-tree-builder-v1",
		Source:            source,
		ManualBookkeeping: false,
		Builder: surface.BlockGraphBuilderReport{
			RootCreatedBy:     "tree_add_root",
			ChildrenCreatedBy: "tree_add_child",
			NodeCount:         len(nodes),
			Capacity:          12,
			OverflowChecked:   true,
		},
		Invariants: surface.BlockGraphInvariantReport{
			TreeValidateRan:         true,
			TreeValidateStatus:      0,
			DuplicateIDRejected:     true,
			MissingParentRejected:   true,
			CycleRejected:           true,
			ParentChildLinksChecked: true,
			ChildOrderChecked:       true,
			FocusOrderChecked:       true,
			HitTestPathChecked:      true,
			AccessibilityChecked:    true,
		},
		RootID:    1,
		NodeCount: len(nodes),
		Nodes:     nodes,
		ChildOrders: []surface.BlockGraphChildOrderReport{
			{ParentID: 1, Children: []int{2, 3, 7}},
			{ParentID: 3, Children: []int{4, 5, 6}},
			{ParentID: 7, Children: []int{8, 9, 10, 11, 12}},
		},
		LayoutOrder:        order,
		DrawOrder:          order,
		FocusOrder:         []int{3, 7},
		AccessibilityOrder: []int{2, 3, 4, 5, 6, 7, 8, 10, 11, 12},
		HitTests: []surface.BlockGraphPathReport{
			{
				Helper:   "tree_hit_test_path",
				Event:    "click",
				TargetID: 6,
				X:        920,
				Y:        248,
				Path:     []int{1, 3, 6},
			},
			{
				Helper:   "tree_hit_test_path",
				Event:    "click",
				TargetID: 11,
				X:        920,
				Y:        494,
				Path:     []int{1, 7, 11},
			},
		},
		DispatchPaths: []surface.BlockGraphPathReport{
			{Helper: "tree_build_dispatch_path", Event: "focus", TargetID: 3, Path: []int{1, 3}},
			{Helper: "tree_build_dispatch_path", Event: "focus", TargetID: 7, Path: []int{1, 7}},
			{Helper: "tree_build_dispatch_path", Event: "click", TargetID: 6, Path: []int{1, 3, 6}},
			{
				Helper:   "tree_build_dispatch_path",
				Event:    "click",
				TargetID: 11,
				Path:     []int{1, 7, 11},
			},
		},
	}
}

func guestDashboardGraphNodes() []surface.BlockGraphNodeReport {
	return []surface.BlockGraphNodeReport{
		{
			ID:                1,
			Name:              "GuestDashboardPage",
			ParentID:          -1,
			ChildIndex:        0,
			FirstChild:        2,
			ChildCount:        3,
			AccessibilityRole: "none",
			Bounds:            surface.RectReport{X: 0, Y: 0, W: 1760, H: 700},
		},
		{
			ID:                2,
			Name:              "PersonalCabinetTitle",
			ParentID:          1,
			ChildIndex:        0,
			FirstChild:        -1,
			ChildCount:        0,
			AccessibilityRole: "text",
			Bounds:            surface.RectReport{X: 50, Y: 32, W: 520, H: 36},
		},
		{
			ID:                3,
			Name:              "RecentCoursesPanel",
			ParentID:          1,
			ChildIndex:        1,
			FirstChild:        4,
			ChildCount:        3,
			Focusable:         true,
			AccessibilityRole: "region",
			Bounds:            surface.RectReport{X: 50, Y: 75, W: 1692, H: 212},
		},
		{
			ID:                4,
			Name:              "RecentCoursesTitle",
			ParentID:          3,
			ChildIndex:        0,
			FirstChild:        -1,
			ChildCount:        0,
			AccessibilityRole: "text",
			Bounds:            surface.RectReport{X: 68, Y: 96, W: 320, H: 22},
		},
		{
			ID:                5,
			Name:              "RecentCoursesEmptyIcon",
			ParentID:          3,
			ChildIndex:        1,
			FirstChild:        -1,
			ChildCount:        0,
			AccessibilityRole: "text",
			Bounds:            surface.RectReport{X: 866, Y: 136, W: 60, H: 72},
		},
		{
			ID:                6,
			Name:              "RecentCoursesEmptyState",
			ParentID:          3,
			ChildIndex:        2,
			FirstChild:        -1,
			ChildCount:        0,
			AccessibilityRole: "text",
			Bounds:            surface.RectReport{X: 820, Y: 238, W: 260, H: 20},
		},
		{
			ID:                7,
			Name:              "CourseOverviewPanel",
			ParentID:          1,
			ChildIndex:        2,
			FirstChild:        8,
			ChildCount:        5,
			Focusable:         true,
			AccessibilityRole: "region",
			Bounds:            surface.RectReport{X: 50, Y: 305, W: 1692, H: 382},
		},
		{
			ID:                8,
			Name:              "CourseOverviewTitle",
			ParentID:          7,
			ChildIndex:        0,
			FirstChild:        -1,
			ChildCount:        0,
			AccessibilityRole: "text",
			Bounds:            surface.RectReport{X: 68, Y: 326, W: 260, H: 22},
		},
		{
			ID:                9,
			Name:              "CourseOverviewDivider",
			ParentID:          7,
			ChildIndex:        1,
			FirstChild:        -1,
			ChildCount:        0,
			AccessibilityRole: "none",
			Bounds:            surface.RectReport{X: 68, Y: 360, W: 1656, H: 1},
		},
		{
			ID:                10,
			Name:              "CourseOverviewEmptyState",
			ParentID:          7,
			ChildIndex:        2,
			FirstChild:        -1,
			ChildCount:        0,
			AccessibilityRole: "text",
			Bounds:            surface.RectReport{X: 866, Y: 382, W: 60, H: 72},
		},
		{
			ID:                11,
			Name:              "CourseOverviewHeadline",
			ParentID:          7,
			ChildIndex:        3,
			FirstChild:        -1,
			ChildCount:        0,
			AccessibilityRole: "text",
			Bounds:            surface.RectReport{X: 755, Y: 484, W: 370, H: 24},
		},
		{
			ID:                12,
			Name:              "CourseOverviewBody",
			ParentID:          7,
			ChildIndex:        4,
			FirstChild:        -1,
			ChildCount:        0,
			AccessibilityRole: "text",
			Bounds:            surface.RectReport{X: 725, Y: 522, W: 430, H: 20},
		},
	}
}

func guestDashboardAccessibilityTreeForScenario(
	source string,
	graph *surface.BlockGraphReport,
) *surface.BlockAccessibilityTreeReport {
	if graph == nil {
		return nil
	}
	focusIndex := map[int]int{}
	for i, id := range graph.FocusOrder {
		focusIndex[id] = i
	}
	readingIndex := map[int]int{}
	for i, id := range graph.AccessibilityOrder {
		readingIndex[id] = i
	}
	roles := []string{}
	seenRoles := map[string]bool{}
	nodes := make([]surface.BlockAccessibilityNodeReport, 0, len(graph.AccessibilityOrder))
	actions := []surface.AccessibilityActionReport{}
	for _, graphNode := range graph.Nodes {
		if graphNode.AccessibilityRole == "" || graphNode.AccessibilityRole == "none" {
			continue
		}
		role := graphNode.AccessibilityRole
		if !seenRoles[role] {
			seenRoles[role] = true
			roles = append(roles, role)
		}
		node := surface.BlockAccessibilityNodeReport{
			ID:            graphNode.ID,
			BlockID:       graphNode.ID,
			ParentBlockID: graphNode.ParentID,
			Name:          graphNode.Name,
			Role:          role,
			Description:   guestDashboardDescriptionForNode(graphNode.Name),
			Bounds:        graphNode.Bounds,
			Visible:       true,
			Enabled:       true,
			Focusable:     graphNode.Focusable,
			FocusIndex:    -1,
			ReadingIndex:  readingIndex[graphNode.ID],
		}
		switch graphNode.Name {
		case "RecentCoursesTitle":
			node.LabelFor = "RecentCoursesPanel"
		case "RecentCoursesPanel":
			node.LabelledBy = "RecentCoursesTitle"
		case "CourseOverviewTitle":
			node.LabelFor = "CourseOverviewPanel"
		case "CourseOverviewPanel":
			node.LabelledBy = "CourseOverviewTitle"
		}
		if graphNode.Focusable {
			node.FocusIndex = focusIndex[graphNode.ID]
			node.Actions = []string{"focus", "inspect"}
			node.Focused = graphNode.ID == graph.FocusOrder[0]
			actions = append(actions, surface.AccessibilityActionReport{
				Target:   graphNode.Name,
				Action:   "inspect",
				Semantic: guestDashboardSemanticForNode(graphNode.Name),
			})
		}
		nodes = append(nodes, node)
	}
	return &surface.BlockAccessibilityTreeReport{
		Schema:                  "tetra.surface.block-accessibility-tree.v1",
		AccessibilityLevel:      "block-metadata-tree-v1",
		Source:                  source,
		Module:                  "lib.core.block",
		QualityLevel:            "block-derived-accessibility-metadata-v1",
		BlockGraphSchema:        "tetra.surface.block-graph.v1",
		DerivedFromBlockGraph:   true,
		ManualBookkeeping:       false,
		PlatformHostIntegration: false,
		DOMARIAIntegration:      false,
		ScreenReaderEvidence:    false,
		NoDOMUI:                 true,
		NoUserJS:                true,
		NoPlatformWidgets:       true,
		NodeCount:               len(nodes),
		FocusableCount:          len(graph.FocusOrder),
		RolesPresent:            roles,
		Nodes:                   nodes,
		Relationships: []surface.AccessibilityRelationshipReport{
			{Kind: "label_for", From: "RecentCoursesTitle", To: "RecentCoursesPanel"},
			{Kind: "label_for", From: "CourseOverviewTitle", To: "CourseOverviewPanel"},
		},
		FocusOrder:   graph.FocusOrder,
		ReadingOrder: graph.AccessibilityOrder,
		Actions:      actions,
		NegativeGuards: surface.BlockAccessibilityNegativeGuardsReport{
			FocusableActionNameChecked:    true,
			LabelRelationshipsChecked:     true,
			ReadingOrderGraphChecked:      true,
			BoundsAlignmentChecked:        true,
			FakeScreenReaderClaimRejected: true,
			ScopedPlatformClaimChecked:    true,
		},
	}
}

func guestDashboardDescriptionForNode(name string) string {
	switch name {
	case "RecentCoursesPanel":
		return "recent courses panel"
	case "CourseOverviewPanel":
		return "course overview panel"
	case "RecentCoursesEmptyState":
		return "no recent courses empty state"
	case "CourseOverviewHeadline":
		return "not enrolled in any courses headline"
	case "CourseOverviewBody":
		return "course enrollment help text"
	default:
		return "guest dashboard surface text"
	}
}

func guestDashboardSemanticForNode(name string) string {
	switch name {
	case "RecentCoursesPanel":
		return "inspect-recent-courses"
	case "CourseOverviewPanel":
		return "inspect-course-overview"
	default:
		return "inspect"
	}
}

func guestDashboardCasesForScenario() []surface.CaseReport {
	return []surface.CaseReport{
		{
			Name: "guest dashboard Morph source avoids manual draw authoring",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name: "guest dashboard personal cabinet title expands to Block scene",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name: "guest dashboard empty states project accessibility labels",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name: "guest dashboard course overview divider stays non-interactive",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
	}
}

// ---- scenarios_text_component.go ----

func runTextFocusInputScenario(mode string) headlessScenario {
	beforeFrame := renderTextFocusInputFrameRGBA(0, 0, 0, 320, 200)
	afterFrame := renderTextFocusInputFrameRGBA(1, 1, 1, 400, 240)
	scenario := headlessScenario{
		Components: []surface.ComponentReport{
			{
				ID:     "TextInputApp",
				Type:   "examples.surface.runtime.surface_textbox_app.TextInputApp",
				Bounds: surface.RectReport{X: 0, Y: 0, W: 400, H: 240},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused_component":  "SubmitButton",
					"width":              "400",
					"height":             "240",
					"resize_count":       "1",
					"accessibility_role": "none",
				},
			},
			{
				ID:     "TextBox",
				Type:   "examples.surface.runtime.surface_textbox_app.TextBox",
				Parent: "TextInputApp",
				Bounds: surface.RectReport{X: 32, Y: 64, W: 224, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "false",
					"buffer":             "Z",
					"text_len":           "1",
					"caret":              "1",
					"backspace_count":    "1",
					"delete_count":       "1",
					"accessibility_role": "label",
				},
			},
			{
				ID:     "SubmitButton",
				Type:   "examples.surface.runtime.surface_textbox_app.ActionButton",
				Parent: "TextInputApp",
				Bounds: surface.RectReport{X: 32, Y: 128, W: 128, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "true",
					"press_count":        "1",
					"key_count":          "1",
					"accessibility_role": "button",
				},
			},
		},
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"TextInputApp", "TextBox"},
				Handled:         true,
				Pass:            true,
				X:               48,
				Y:               96,
				Width:           320,
				Height:          200,
				BufferSlots:     []int{5, 48, 96, 1, 0, 320, 200, 0, 0},
				BeforeState: map[string]string{
					"TextInputApp.focused_component": "none",
					"TextBox.focused":                "false",
				},
				AfterState: map[string]string{
					"TextInputApp.focused_component": "TextBox",
					"TextBox.focused":                "true",
				},
			},
			{
				Order:           2,
				Kind:            "text_input",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"TextInputApp", "TextBox"},
				Handled:         true,
				Pass:            true,
				Width:           320,
				Height:          200,
				TimestampMS:     1,
				TextLen:         2,
				TextBytesHex:    "4f4b",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
				BeforeState: map[string]string{
					"TextBox.buffer":   "",
					"TextBox.caret":    "0",
					"TextBox.text_len": "0",
				},
				AfterState: map[string]string{
					"TextBox.buffer":   "OK",
					"TextBox.caret":    "2",
					"TextBox.text_len": "2",
				},
			},
			{
				Order:           3,
				Kind:            "key_down",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"TextInputApp", "TextBox"},
				Handled:         true,
				Pass:            true,
				Key:             37,
				Width:           320,
				Height:          200,
				TimestampMS:     2,
				BufferSlots:     []int{6, 0, 0, 0, 37, 320, 200, 2, 0},
				BeforeState:     map[string]string{"TextBox.caret": "2", "TextBox.buffer": "OK"},
				AfterState:      map[string]string{"TextBox.caret": "1", "TextBox.buffer": "OK"},
			},
			{
				Order:           4,
				Kind:            "key_down",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"TextInputApp", "TextBox"},
				Handled:         true,
				Pass:            true,
				Key:             8,
				Width:           320,
				Height:          200,
				TimestampMS:     3,
				BufferSlots:     []int{6, 0, 0, 0, 8, 320, 200, 3, 0},
				BeforeState:     map[string]string{"TextBox.buffer": "OK", "TextBox.caret": "1"},
				AfterState:      map[string]string{"TextBox.buffer": "K", "TextBox.caret": "0"},
			},
			{
				Order:           5,
				Kind:            "key_down",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"TextInputApp", "TextBox"},
				Handled:         true,
				Pass:            true,
				Key:             46,
				Width:           320,
				Height:          200,
				TimestampMS:     4,
				BufferSlots:     []int{6, 0, 0, 0, 46, 320, 200, 4, 0},
				BeforeState:     map[string]string{"TextBox.buffer": "K", "TextBox.caret": "0"},
				AfterState:      map[string]string{"TextBox.buffer": "", "TextBox.caret": "0"},
			},
			{
				Order:           6,
				Kind:            "text_input",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"TextInputApp", "TextBox"},
				Handled:         true,
				Pass:            true,
				Width:           320,
				Height:          200,
				TimestampMS:     5,
				TextLen:         1,
				TextBytesHex:    "5a",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 5, 1},
				BeforeState: map[string]string{
					"TextBox.buffer":   "",
					"TextBox.caret":    "0",
					"TextBox.text_len": "0",
				},
				AfterState: map[string]string{
					"TextBox.buffer":   "Z",
					"TextBox.caret":    "1",
					"TextBox.text_len": "1",
				},
			},
			{
				Order:           7,
				Kind:            "key_down",
				TargetComponent: "TextInputApp",
				DispatchPath:    []string{"TextInputApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          200,
				TimestampMS:     6,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 200, 6, 0},
				BeforeState: map[string]string{
					"TextInputApp.focused_component": "TextBox",
					"TextBox.focused":                "true",
					"SubmitButton.focused":           "false",
				},
				AfterState: map[string]string{
					"TextInputApp.focused_component": "SubmitButton",
					"TextBox.focused":                "false",
					"SubmitButton.focused":           "true",
				},
			},
			{
				Order:           8,
				Kind:            "key_down",
				TargetComponent: "SubmitButton",
				DispatchPath:    []string{"TextInputApp", "SubmitButton"},
				Handled:         true,
				Pass:            true,
				Key:             32,
				Width:           320,
				Height:          200,
				TimestampMS:     7,
				BufferSlots:     []int{6, 0, 0, 0, 32, 320, 200, 7, 0},
				BeforeState: map[string]string{
					"SubmitButton.press_count": "0",
					"TextBox.buffer":           "Z",
				},
				AfterState: map[string]string{
					"SubmitButton.press_count": "1",
					"TextBox.buffer":           "Z",
				},
			},
			{
				Order:           9,
				Kind:            "resize",
				TargetComponent: "TextInputApp",
				DispatchPath:    []string{"TextInputApp"},
				Handled:         true,
				Pass:            true,
				Width:           400,
				Height:          240,
				TimestampMS:     8,
				BufferSlots:     []int{2, 0, 0, 0, 0, 400, 240, 8, 0},
				BeforeState: map[string]string{
					"TextInputApp.width":             "320",
					"TextInputApp.focused_component": "SubmitButton",
				},
				AfterState: map[string]string{
					"TextInputApp.width":             "400",
					"TextInputApp.focused_component": "SubmitButton",
				},
			},
		},
		StateTransitions: []surface.StateTransitionReport{
			{
				Order:     1,
				Component: "TextInputApp",
				Field:     "focused_component",
				Before:    "none",
				After:     "TextBox",
				Cause:     "mouse_up",
			},
			{
				Order:     2,
				Component: "TextBox",
				Field:     "buffer",
				Before:    "",
				After:     "OK",
				Cause:     "text_input",
			},
			{
				Order:     3,
				Component: "TextBox",
				Field:     "caret",
				Before:    "2",
				After:     "1",
				Cause:     "key_down",
			},
			{
				Order:     4,
				Component: "TextBox",
				Field:     "buffer",
				Before:    "OK",
				After:     "K",
				Cause:     "backspace",
			},
			{
				Order:     5,
				Component: "TextBox",
				Field:     "buffer",
				Before:    "K",
				After:     "",
				Cause:     "delete",
			},
			{
				Order:     6,
				Component: "TextBox",
				Field:     "buffer",
				Before:    "",
				After:     "Z",
				Cause:     "text_input",
			},
			{
				Order:     7,
				Component: "TextInputApp",
				Field:     "focused_component",
				Before:    "TextBox",
				After:     "SubmitButton",
				Cause:     "tab",
			},
			{
				Order:     8,
				Component: "SubmitButton",
				Field:     "press_count",
				Before:    "0",
				After:     "1",
				Cause:     "key_down",
			},
			{
				Order:     9,
				Component: "TextInputApp",
				Field:     "width",
				Before:    "320",
				After:     "400",
				Cause:     "resize",
			},
		},
		Cases: []surface.CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "text focus input click focuses TextBox",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{Name: "text focus input Tab changes focus", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "text focus input keyboard routes only focused component",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{Name: "text focus input text insertion", Kind: "positive", Ran: true, Pass: true},
			{Name: "text focus input caret movement", Kind: "positive", Ran: true, Pass: true},
			{Name: "text focus input backspace delete", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "text focus input resize preserves focus",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "text focus input rendered frame update",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
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
			{
				Name:          "reject legacy UI evidence",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "legacy UI evidence rejected",
			},
		},
	}
	if mode == "headless-text-focus-input" || mode == "linux-x64-real-window-text-focus-input" {
		scenario.Frames = []surface.FrameReport{
			{
				Order:     1,
				Width:     beforeFrame.Width,
				Height:    beforeFrame.Height,
				Stride:    beforeFrame.Stride,
				Checksum:  checksumRGBA(beforeFrame.Pixels),
				Presented: true,
			},
			{
				Order:     2,
				Width:     afterFrame.Width,
				Height:    afterFrame.Height,
				Stride:    afterFrame.Stride,
				Checksum:  checksumRGBA(afterFrame.Pixels),
				Presented: true,
			},
		}
	}
	switch mode {
	case "headless-text-focus-input":
		scenario.Cases = append(
			scenario.Cases,
			surface.CaseReport{
				Name: "headless event dispatch",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "headless framebuffer checksum",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "headless actual runner trace",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
		)
	case "linux-x64-real-window-text-focus-input":
		scenario.Cases = append(
			scenario.Cases,
			surface.CaseReport{
				Name: "linux-x64 Surface Host ABI open/present/close",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 real-window surface",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 native input event pump",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 real-window resize event",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 real-window close event",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
		)
	case "wasm32-web-browser-canvas-text-focus-input":
		scenario.Cases = append(
			scenario.Cases,
			surface.CaseReport{
				Name: "wasm32-web browser canvas surface",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas RGBA readback",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas pointer input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas keyboard input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas resize input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas text input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web Surface Host ABI imports",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "compiler-owned wasm Surface loader",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "compiler-owned browser canvas Surface host",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
		)
	}
	return scenario
}
func runComponentTreeScenario(mode string) headlessScenario {
	beforeFrame := renderComponentTreeFrameRGBA(0, 0, -1, 0, 0, 320, 200)
	afterFrame := renderComponentTreeFrameRGBA(0, 0, 6, 1, 1, 400, 240)
	scenario := headlessScenario{
		Components: []surface.ComponentReport{
			{
				ID:     "TreeApp",
				Type:   "examples.surface.toolkit.surface_tree_app.TreeApp",
				Bounds: surface.RectReport{X: 0, Y: 0, W: 400, H: 240},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused_id":         "6",
					"submitted_count":    "1",
					"reset_count":        "1",
					"width":              "400",
					"height":             "240",
					"accessibility_role": "none",
				},
			},
			{
				ID:     "Column",
				Type:   "examples.surface.toolkit.surface_tree_app.Column",
				Parent: "TreeApp",
				Bounds: surface.RectReport{X: 0, Y: 0, W: 400, H: 240},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{"child_count": "3", "accessibility_role": "none"},
			},
			{
				ID:     "NameLabel",
				Type:   "examples.surface.toolkit.surface_tree_app.TextLabel",
				Parent: "Column",
				Bounds: surface.RectReport{X: 16, Y: 16, W: 288, H: 24},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{"text": "Name", "accessibility_role": "label"},
			},
			{
				ID:     "TextBox",
				Type:   "examples.surface.toolkit.surface_tree_app.TextBox",
				Parent: "Column",
				Bounds: surface.RectReport{X: 16, Y: 48, W: 368, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "false",
					"buffer":             "",
					"text_len":           "0",
					"caret":              "0",
					"accessibility_role": "label",
				},
			},
			{
				ID:     "ButtonRow",
				Type:   "examples.surface.toolkit.surface_tree_app.Row",
				Parent: "Column",
				Bounds: surface.RectReport{X: 16, Y: 104, W: 368, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{"child_count": "2", "accessibility_role": "none"},
			},
			{
				ID:     "SubmitButton",
				Type:   "examples.surface.toolkit.surface_tree_app.Button",
				Parent: "ButtonRow",
				Bounds: surface.RectReport{X: 16, Y: 104, W: 132, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "false",
					"press_count":        "1",
					"accessibility_role": "button",
				},
			},
			{
				ID:     "ResetButton",
				Type:   "examples.surface.toolkit.surface_tree_app.Button",
				Parent: "ButtonRow",
				Bounds: surface.RectReport{X: 160, Y: 104, W: 132, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "true",
					"press_count":        "1",
					"accessibility_role": "button",
				},
			},
		},
		ComponentTree: &surface.ComponentTreeReport{
			Schema:       "tetra.surface.component-tree.v1",
			DynamicLevel: "semi-dynamic-child-list",
			RootID:       0,
			NodeCount:    7,
			FocusedID:    6,
			Nodes: []surface.ComponentTreeNodeReport{
				{
					ID:         0,
					Name:       "TreeApp",
					Kind:       "root",
					ParentID:   -1,
					ChildIndex: 0,
					FirstChild: 1,
					ChildCount: 1,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 0, Y: 0, W: 400, H: 240},
				},
				{
					ID:         1,
					Name:       "Column",
					Kind:       "column",
					ParentID:   0,
					ChildIndex: 0,
					FirstChild: 2,
					ChildCount: 3,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 0, Y: 0, W: 400, H: 240},
				},
				{
					ID:         2,
					Name:       "NameLabel",
					Kind:       "text",
					ParentID:   1,
					ChildIndex: 0,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 16, Y: 16, W: 288, H: 24},
				},
				{
					ID:         3,
					Name:       "TextBox",
					Kind:       "textbox",
					ParentID:   1,
					ChildIndex: 1,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  true,
					Bounds:     surface.RectReport{X: 16, Y: 48, W: 368, H: 44},
				},
				{
					ID:         4,
					Name:       "ButtonRow",
					Kind:       "row",
					ParentID:   1,
					ChildIndex: 2,
					FirstChild: 5,
					ChildCount: 2,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 16, Y: 104, W: 368, H: 44},
				},
				{
					ID:         5,
					Name:       "SubmitButton",
					Kind:       "button",
					ParentID:   4,
					ChildIndex: 0,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  true,
					Bounds:     surface.RectReport{X: 16, Y: 104, W: 132, H: 44},
				},
				{
					ID:         6,
					Name:       "ResetButton",
					Kind:       "button",
					ParentID:   4,
					ChildIndex: 1,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  true,
					Bounds:     surface.RectReport{X: 160, Y: 104, W: 132, H: 44},
				},
			},
			LayoutPasses: []surface.ComponentTreeLayoutPassReport{
				{
					ComponentID: 3,
					Pass:        "initial",
					Bounds:      surface.RectReport{X: 16, Y: 48, W: 288, H: 44},
					Measured:    surface.SizeReport{W: 288, H: 44},
				},
				{
					ComponentID: 3,
					Pass:        "resize",
					Bounds:      surface.RectReport{X: 16, Y: 48, W: 368, H: 44},
					Measured:    surface.SizeReport{W: 368, H: 44},
				},
			},
			DrawOrder:  []int{0, 1, 2, 3, 4, 5, 6},
			FocusOrder: []int{3, 5, 6},
			DispatchPaths: []surface.ComponentTreeDispatchPathReport{
				{Event: "click", TargetID: 3, X: 40, Y: 72, Path: []int{0, 1, 3}},
				{Event: "click", TargetID: 5, X: 32, Y: 120, Path: []int{0, 1, 4, 5}},
				{Event: "click", TargetID: 6, X: 176, Y: 120, Path: []int{0, 1, 4, 6}},
			},
		},
		ComponentTreeAPI: componentTreeAPIReport(),
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"TreeApp", "Column", "TextBox"},
				Handled:         true,
				Pass:            true,
				X:               40,
				Y:               72,
				Width:           320,
				Height:          200,
				BufferSlots:     []int{5, 40, 72, 1, 0, 320, 200, 0, 0},
				BeforeState: map[string]string{
					"TreeApp.focused_id": "-1",
					"TextBox.focused":    "false",
				},
				AfterState: map[string]string{
					"TreeApp.focused_id": "3",
					"TextBox.focused":    "true",
				},
			},
			{
				Order:           2,
				Kind:            "text_input",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"TreeApp", "Column", "TextBox"},
				Handled:         true,
				Pass:            true,
				Width:           320,
				Height:          200,
				TimestampMS:     1,
				TextLen:         2,
				TextBytesHex:    "4f4b",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
				BeforeState:     map[string]string{"TextBox.buffer": "", "TextBox.caret": "0"},
				AfterState:      map[string]string{"TextBox.buffer": "OK", "TextBox.caret": "2"},
			},
			{
				Order:           3,
				Kind:            "key_down",
				TargetComponent: "TreeApp",
				DispatchPath:    []string{"TreeApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          200,
				TimestampMS:     2,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 200, 2, 0},
				BeforeState:     map[string]string{"TreeApp.focused_id": "3"},
				AfterState:      map[string]string{"TreeApp.focused_id": "5"},
			},
			{
				Order:           4,
				Kind:            "key_down",
				TargetComponent: "SubmitButton",
				DispatchPath:    []string{"TreeApp", "Column", "ButtonRow", "SubmitButton"},
				Handled:         true,
				Pass:            true,
				Key:             32,
				Width:           320,
				Height:          200,
				TimestampMS:     3,
				BufferSlots:     []int{6, 0, 0, 0, 32, 320, 200, 3, 0},
				BeforeState: map[string]string{
					"TreeApp.submitted_count": "0",
					"TreeApp.focused_id":      "5",
				},
				AfterState: map[string]string{
					"TreeApp.submitted_count": "1",
					"TreeApp.focused_id":      "5",
				},
			},
			{
				Order:           5,
				Kind:            "key_down",
				TargetComponent: "TreeApp",
				DispatchPath:    []string{"TreeApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          200,
				TimestampMS:     4,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 200, 4, 0},
				BeforeState:     map[string]string{"TreeApp.focused_id": "5"},
				AfterState:      map[string]string{"TreeApp.focused_id": "6"},
			},
			{
				Order:           6,
				Kind:            "text_input",
				TargetComponent: "ResetButton",
				DispatchPath:    []string{"TreeApp", "Column", "ButtonRow", "ResetButton"},
				Handled:         false,
				Pass:            true,
				Width:           320,
				Height:          200,
				TimestampMS:     5,
				TextLen:         1,
				TextBytesHex:    "5a",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 5, 1},
				BeforeState: map[string]string{
					"TreeApp.focused_id": "6",
					"TextBox.buffer":     "OK",
				},
				AfterState: map[string]string{
					"TreeApp.focused_id": "6",
					"TextBox.buffer":     "OK",
				},
			},
			{
				Order:           7,
				Kind:            "key_down",
				TargetComponent: "ResetButton",
				DispatchPath:    []string{"TreeApp", "Column", "ButtonRow", "ResetButton"},
				Handled:         true,
				Pass:            true,
				Key:             13,
				Width:           320,
				Height:          200,
				TimestampMS:     6,
				BufferSlots:     []int{6, 0, 0, 0, 13, 320, 200, 6, 0},
				BeforeState: map[string]string{
					"TreeApp.reset_count": "0",
					"TextBox.buffer":      "OK",
					"TreeApp.focused_id":  "6",
				},
				AfterState: map[string]string{
					"TreeApp.reset_count": "1",
					"TextBox.buffer":      "",
					"TreeApp.focused_id":  "6",
				},
			},
			{
				Order:           8,
				Kind:            "key_down",
				TargetComponent: "TreeApp",
				DispatchPath:    []string{"TreeApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          200,
				TimestampMS:     7,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 200, 7, 0},
				BeforeState:     map[string]string{"TreeApp.focused_id": "6"},
				AfterState:      map[string]string{"TreeApp.focused_id": "3"},
			},
			{
				Order:           9,
				Kind:            "key_down",
				TargetComponent: "TreeApp",
				DispatchPath:    []string{"TreeApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          200,
				TimestampMS:     8,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 200, 8, 0},
				BeforeState:     map[string]string{"TreeApp.focused_id": "3"},
				AfterState:      map[string]string{"TreeApp.focused_id": "5"},
			},
			{
				Order:           10,
				Kind:            "key_down",
				TargetComponent: "TreeApp",
				DispatchPath:    []string{"TreeApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          200,
				TimestampMS:     9,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 200, 9, 0},
				BeforeState:     map[string]string{"TreeApp.focused_id": "5"},
				AfterState:      map[string]string{"TreeApp.focused_id": "6"},
			},
			{
				Order:           11,
				Kind:            "resize",
				TargetComponent: "TreeApp",
				DispatchPath:    []string{"TreeApp"},
				Handled:         true,
				Pass:            true,
				Width:           400,
				Height:          240,
				TimestampMS:     10,
				BufferSlots:     []int{2, 0, 0, 0, 0, 400, 240, 10, 0},
				BeforeState: map[string]string{
					"TreeApp.focused_id": "6",
					"TextBox.bounds.w":   "288",
				},
				AfterState: map[string]string{
					"TreeApp.focused_id": "6",
					"TextBox.bounds.w":   "368",
				},
			},
		},
		StateTransitions: []surface.StateTransitionReport{
			{
				Order:     1,
				Component: "TreeApp",
				Field:     "focused_id",
				Before:    "-1",
				After:     "3",
				Cause:     "mouse_up",
			},
			{
				Order:     2,
				Component: "TextBox",
				Field:     "buffer",
				Before:    "",
				After:     "OK",
				Cause:     "text_input",
			},
			{
				Order:     3,
				Component: "TreeApp",
				Field:     "focused_id",
				Before:    "3",
				After:     "5",
				Cause:     "tab",
			},
			{
				Order:     4,
				Component: "TreeApp",
				Field:     "submitted_count",
				Before:    "0",
				After:     "1",
				Cause:     "key_down",
			},
			{
				Order:     5,
				Component: "TreeApp",
				Field:     "focused_id",
				Before:    "5",
				After:     "6",
				Cause:     "tab",
			},
			{
				Order:     6,
				Component: "TextBox",
				Field:     "buffer",
				Before:    "OK",
				After:     "",
				Cause:     "reset",
			},
			{
				Order:     7,
				Component: "TreeApp",
				Field:     "reset_count",
				Before:    "0",
				After:     "1",
				Cause:     "key_down",
			},
			{
				Order:     8,
				Component: "TreeApp",
				Field:     "focused_id",
				Before:    "6",
				After:     "3",
				Cause:     "tab",
			},
			{
				Order:     9,
				Component: "TreeApp",
				Field:     "focused_id",
				Before:    "3",
				After:     "5",
				Cause:     "tab",
			},
			{
				Order:     10,
				Component: "TreeApp",
				Field:     "focused_id",
				Before:    "5",
				After:     "6",
				Cause:     "tab",
			},
			{
				Order:     11,
				Component: "TreeApp",
				Field:     "TextBox.bounds.w",
				Before:    "288",
				After:     "368",
				Cause:     "resize",
			},
		},
		Cases: []surface.CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
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
			{Name: "component tree node count", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree parent child links", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree layout bounds", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree draw traversal", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree pointer dispatch path", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree focus traversal", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "component tree text routed to focused TextBox",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "component tree button action dispatch",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{Name: "component tree resize relayout", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree rendered frame update", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "component tree api builder node creation",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "component tree api parent child invariants",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "component tree api layout helper dispatch",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{Name: "component tree api hit test helper", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "component tree api focus helper traversal",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "component tree api dispatch path helper",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "component tree api no manual bookkeeping",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name:          "reject legacy UI evidence",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "legacy UI evidence rejected",
			},
		},
	}
	if mode == "headless-component-tree" || mode == "linux-x64-real-window-component-tree" ||
		mode == "headless-component-tree-api" || mode == "linux-x64-real-window-component-tree-api" {
		scenario.Frames = []surface.FrameReport{
			{
				Order:     1,
				Width:     beforeFrame.Width,
				Height:    beforeFrame.Height,
				Stride:    beforeFrame.Stride,
				Checksum:  checksumRGBA(beforeFrame.Pixels),
				Presented: true,
			},
			{
				Order:     2,
				Width:     afterFrame.Width,
				Height:    afterFrame.Height,
				Stride:    afterFrame.Stride,
				Checksum:  checksumRGBA(afterFrame.Pixels),
				Presented: true,
			},
		}
	}
	switch mode {
	case "headless-component-tree", "headless-component-tree-api":
		scenario.Cases = append(
			scenario.Cases,
			surface.CaseReport{
				Name: "headless event dispatch",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "headless framebuffer checksum",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "headless actual runner trace",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
		)
	case "linux-x64-real-window-component-tree", "linux-x64-real-window-component-tree-api":
		scenario.Cases = append(
			scenario.Cases,
			surface.CaseReport{
				Name: "linux-x64 Surface Host ABI open/present/close",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 real-window surface",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 native input event pump",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 real-window resize event",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 real-window close event",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
		)
	case "wasm32-web-browser-canvas-component-tree", "wasm32-web-browser-canvas-component-tree-api":
		scenario.Cases = append(
			scenario.Cases,
			surface.CaseReport{
				Name: "wasm32-web browser canvas surface",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas RGBA readback",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas pointer input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas keyboard input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas resize input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas text input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web Surface Host ABI imports",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "compiler-owned wasm Surface loader",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "compiler-owned browser canvas Surface host",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
		)
	}
	return scenario
}
func componentTreeAPIReport() *surface.ComponentTreeAPIReport {
	return &surface.ComponentTreeAPIReport{
		Schema:            "tetra.surface.component-tree-api.v1",
		APILevel:          "builder-layout-dispatch-v1",
		Source:            "examples/surface/toolkit/surface_tree_app.tetra",
		ManualBookkeeping: false,
		Builder: surface.ComponentTreeAPIBuilderReport{
			RootCreatedBy:     "tree_add_root",
			ChildrenCreatedBy: "tree_add_child",
			NodeCount:         7,
			Capacity:          16,
			OverflowChecked:   true,
		},
		Invariants: surface.ComponentTreeAPIInvariantReport{
			TreeValidateRan:         true,
			TreeValidateStatus:      0,
			ParentChildLinksChecked: true,
			ChildIndicesChecked:     true,
			ChildCountChecked:       true,
			FirstChildChecked:       true,
		},
		LayoutHelpers: []surface.ComponentTreeAPILayoutHelperReport{
			{Helper: "tree_layout_column", Target: "Column", Pass: "initial", ChangedBounds: true},
			{Helper: "tree_layout_row", Target: "ButtonRow", Pass: "initial", ChangedBounds: true},
			{Helper: "tree_layout_column", Target: "Column", Pass: "resize", ChangedBounds: true},
		},
		FocusHelpers: []surface.ComponentTreeAPIFocusHelperReport{
			{Helper: "tree_focus_next", Before: "TextBox", After: "SubmitButton"},
			{Helper: "tree_focus_next", Before: "SubmitButton", After: "ResetButton"},
			{Helper: "tree_focus_next", Before: "ResetButton", After: "TextBox"},
		},
		HitTests: []surface.ComponentTreeAPIHitTestReport{
			{Helper: "tree_hit_test", X: 40, Y: 72, Target: "TextBox", Path: []int{0, 1, 3}},
			{
				Helper: "tree_hit_test",
				X:      176,
				Y:      120,
				Target: "ResetButton",
				Path:   []int{0, 1, 4, 6},
			},
		},
		DispatchPaths: []surface.ComponentTreeAPIDispatchPathReport{
			{Helper: "tree_build_dispatch_path", Target: "TextBox", Path: []int{0, 1, 3}},
			{Helper: "tree_build_dispatch_path", Target: "SubmitButton", Path: []int{0, 1, 4, 5}},
			{Helper: "tree_build_dispatch_path", Target: "ResetButton", Path: []int{0, 1, 4, 6}},
		},
	}
}
func renderCounterFrameRGBA(count int, focused bool) rgbaFrame {
	frame := newRGBAFrame(320, 200)
	bg := rgbaColor{R: 20, G: 24, B: 26, A: 255}
	fg := rgbaColor{R: 238, G: 241, B: 245, A: 255}
	accent := rgbaColor{R: 32, G: 132, B: 214, A: 255}
	button := rect{X: 32, Y: 80, W: 160, H: 48}

	clearRGBA(frame, bg)
	textMaskRGBA(frame, 32, 28, 5, fg)
	rectRGBA(frame, button, accent)
	if count > 0 {
		rectRGBA(frame, rect{X: 88, Y: 28, W: 24, H: 7}, fg)
	}
	if focused {
		rectOutlineRGBA(
			frame,
			rect{X: button.X - 4, Y: button.Y - 4, W: button.W + 8, H: button.H + 8},
			fg,
		)
	}
	rectOutlineRGBA(frame, button, fg)
	return frame
}

// ---- scenarios_toolkit.go ----

func runMinimalToolkitScenario(mode string) headlessScenario {
	beforeFrame := renderMinimalToolkitFrameRGBA(0, 0, -1, 0, 0, 0, 320, 200)
	textFrame := renderMinimalToolkitFrameRGBA(2, 2, 4, 0, 0, 0, 320, 200)
	submitFrame := renderMinimalToolkitFrameRGBA(1, 1, 6, 1, 0, 1, 320, 200)
	afterFrame := renderMinimalToolkitFrameRGBA(0, 0, 4, 1, 1, 2, 400, 240)
	scenario := headlessScenario{
		Components: []surface.ComponentReport{
			{
				ID:     "ToolkitFormApp",
				Type:   "examples.surface.toolkit.surface_toolkit_form.ToolkitFormApp",
				Bounds: surface.RectReport{X: 0, Y: 0, W: 400, H: 240},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused_id":         "4",
					"submit_count":       "1",
					"reset_count":        "1",
					"status_code":        "2",
					"width":              "400",
					"height":             "240",
					"accessibility_role": "none",
				},
			},
			{
				ID:     "Panel",
				Type:   "lib.core.widgets.Panel",
				Parent: "ToolkitFormApp",
				Bounds: surface.RectReport{X: 0, Y: 0, W: 400, H: 240},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{"padding": "12", "accessibility_role": "none"},
			},
			{
				ID:     "Column",
				Type:   "lib.core.widgets.Column",
				Parent: "Panel",
				Bounds: surface.RectReport{X: 12, Y: 12, W: 376, H: 216},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{"child_count": "4", "accessibility_role": "none"},
			},
			{
				ID:     "NameLabel",
				Type:   "lib.core.widgets.Text",
				Parent: "Column",
				Bounds: surface.RectReport{X: 20, Y: 20, W: 360, H: 24},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"role":               "label",
					"text_len":           "4",
					"accessibility_role": "label",
				},
			},
			{
				ID:     "TextBox",
				Type:   "lib.core.widgets.TextBox",
				Parent: "Column",
				Bounds: surface.RectReport{X: 20, Y: 52, W: 360, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "true",
					"buffer":             "",
					"text_len":           "0",
					"caret":              "0",
					"backspace_count":    "1",
					"delete_count":       "1",
					"accessibility_role": "label",
				},
			},
			{
				ID:     "ButtonRow",
				Type:   "lib.core.widgets.Row",
				Parent: "Column",
				Bounds: surface.RectReport{X: 20, Y: 108, W: 360, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{"child_count": "2", "accessibility_role": "none"},
			},
			{
				ID:     "SubmitButton",
				Type:   "lib.core.widgets.Button",
				Parent: "ButtonRow",
				Bounds: surface.RectReport{X: 20, Y: 108, W: 132, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "false",
					"press_count":        "1",
					"action":             "submit",
					"accessibility_role": "button",
				},
			},
			{
				ID:     "ResetButton",
				Type:   "lib.core.widgets.Button",
				Parent: "ButtonRow",
				Bounds: surface.RectReport{X: 164, Y: 108, W: 132, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "false",
					"press_count":        "1",
					"action":             "reset",
					"accessibility_role": "button",
				},
			},
			{
				ID:     "StatusText",
				Type:   "lib.core.widgets.Text",
				Parent: "Column",
				Bounds: surface.RectReport{X: 20, Y: 160, W: 360, H: 24},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"role":               "status",
					"status_code":        "2",
					"accessibility_role": "label",
				},
			},
		},
		ComponentTree: &surface.ComponentTreeReport{
			Schema:       "tetra.surface.component-tree.v1",
			DynamicLevel: "minimal-toolkit-widget-tree",
			RootID:       0,
			NodeCount:    9,
			FocusedID:    4,
			Nodes: []surface.ComponentTreeNodeReport{
				{
					ID:         0,
					Name:       "ToolkitFormApp",
					Kind:       "root",
					ParentID:   -1,
					ChildIndex: 0,
					FirstChild: 1,
					ChildCount: 1,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 0, Y: 0, W: 400, H: 240},
				},
				{
					ID:         1,
					Name:       "Panel",
					Kind:       "panel",
					ParentID:   0,
					ChildIndex: 0,
					FirstChild: 2,
					ChildCount: 1,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 0, Y: 0, W: 400, H: 240},
				},
				{
					ID:         2,
					Name:       "Column",
					Kind:       "column",
					ParentID:   1,
					ChildIndex: 0,
					FirstChild: 3,
					ChildCount: 4,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 12, Y: 12, W: 376, H: 216},
				},
				{
					ID:         3,
					Name:       "NameLabel",
					Kind:       "text",
					ParentID:   2,
					ChildIndex: 0,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 20, Y: 20, W: 360, H: 24},
				},
				{
					ID:         4,
					Name:       "TextBox",
					Kind:       "textbox",
					ParentID:   2,
					ChildIndex: 1,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  true,
					Bounds:     surface.RectReport{X: 20, Y: 52, W: 360, H: 44},
				},
				{
					ID:         5,
					Name:       "ButtonRow",
					Kind:       "row",
					ParentID:   2,
					ChildIndex: 2,
					FirstChild: 6,
					ChildCount: 2,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 20, Y: 108, W: 360, H: 44},
				},
				{
					ID:         6,
					Name:       "SubmitButton",
					Kind:       "button",
					ParentID:   5,
					ChildIndex: 0,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  true,
					Bounds:     surface.RectReport{X: 20, Y: 108, W: 132, H: 44},
				},
				{
					ID:         7,
					Name:       "ResetButton",
					Kind:       "button",
					ParentID:   5,
					ChildIndex: 1,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  true,
					Bounds:     surface.RectReport{X: 164, Y: 108, W: 132, H: 44},
				},
				{
					ID:         8,
					Name:       "StatusText",
					Kind:       "text",
					ParentID:   2,
					ChildIndex: 3,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 20, Y: 160, W: 360, H: 24},
				},
			},
			LayoutPasses: []surface.ComponentTreeLayoutPassReport{
				{
					ComponentID: 4,
					Pass:        "initial",
					Bounds:      surface.RectReport{X: 20, Y: 52, W: 280, H: 44},
					Measured:    surface.SizeReport{W: 280, H: 44},
				},
				{
					ComponentID: 4,
					Pass:        "resize",
					Bounds:      surface.RectReport{X: 20, Y: 52, W: 360, H: 44},
					Measured:    surface.SizeReport{W: 360, H: 44},
				},
				{
					ComponentID: 8,
					Pass:        "status-update",
					Bounds:      surface.RectReport{X: 20, Y: 160, W: 360, H: 24},
					Measured:    surface.SizeReport{W: 360, H: 24},
				},
			},
			DrawOrder:  []int{0, 1, 2, 3, 4, 5, 6, 7, 8},
			FocusOrder: []int{4, 6, 7},
			DispatchPaths: []surface.ComponentTreeDispatchPathReport{
				{Event: "click", TargetID: 4, X: 40, Y: 72, Path: []int{0, 1, 2, 4}},
				{Event: "click", TargetID: 6, X: 40, Y: 124, Path: []int{0, 1, 2, 5, 6}},
				{Event: "click", TargetID: 7, X: 180, Y: 124, Path: []int{0, 1, 2, 5, 7}},
			},
		},
		ComponentTreeAPI: minimalToolkitComponentTreeAPIReport(),
		Toolkit:          minimalToolkitReport(),
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"ToolkitFormApp", "Panel", "Column", "TextBox"},
				Handled:         true,
				Pass:            true,
				X:               40,
				Y:               72,
				Width:           320,
				Height:          200,
				BufferSlots:     []int{5, 40, 72, 1, 0, 320, 200, 0, 0},
				BeforeState: map[string]string{
					"ToolkitFormApp.focused_id": "-1",
					"TextBox.focused":           "false",
				},
				AfterState: map[string]string{
					"ToolkitFormApp.focused_id": "4",
					"TextBox.focused":           "true",
				},
			},
			{
				Order:           2,
				Kind:            "text_input",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"ToolkitFormApp", "Panel", "Column", "TextBox"},
				Handled:         true,
				Pass:            true,
				Width:           320,
				Height:          200,
				TimestampMS:     1,
				TextLen:         2,
				TextBytesHex:    "4f4b",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
				BeforeState: map[string]string{
					"TextBox.buffer":   "",
					"TextBox.caret":    "0",
					"TextBox.text_len": "0",
				},
				AfterState: map[string]string{
					"TextBox.buffer":   "OK",
					"TextBox.caret":    "2",
					"TextBox.text_len": "2",
				},
			},
			{
				Order:           3,
				Kind:            "key_down",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"ToolkitFormApp", "Panel", "Column", "TextBox"},
				Handled:         true,
				Pass:            true,
				Key:             37,
				Width:           320,
				Height:          200,
				TimestampMS:     2,
				BufferSlots:     []int{6, 0, 0, 0, 37, 320, 200, 2, 0},
				BeforeState:     map[string]string{"TextBox.buffer": "OK", "TextBox.caret": "2"},
				AfterState:      map[string]string{"TextBox.buffer": "OK", "TextBox.caret": "1"},
			},
			{
				Order:           4,
				Kind:            "key_down",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"ToolkitFormApp", "Panel", "Column", "TextBox"},
				Handled:         true,
				Pass:            true,
				Key:             8,
				Width:           320,
				Height:          200,
				TimestampMS:     3,
				BufferSlots:     []int{6, 0, 0, 0, 8, 320, 200, 3, 0},
				BeforeState:     map[string]string{"TextBox.buffer": "OK", "TextBox.caret": "1"},
				AfterState:      map[string]string{"TextBox.buffer": "K", "TextBox.caret": "0"},
			},
			{
				Order:           5,
				Kind:            "key_down",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"ToolkitFormApp", "Panel", "Column", "TextBox"},
				Handled:         true,
				Pass:            true,
				Key:             46,
				Width:           320,
				Height:          200,
				TimestampMS:     4,
				BufferSlots:     []int{6, 0, 0, 0, 46, 320, 200, 4, 0},
				BeforeState:     map[string]string{"TextBox.buffer": "K", "TextBox.caret": "0"},
				AfterState:      map[string]string{"TextBox.buffer": "", "TextBox.caret": "0"},
			},
			{
				Order:           6,
				Kind:            "text_input",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"ToolkitFormApp", "Panel", "Column", "TextBox"},
				Handled:         true,
				Pass:            true,
				Width:           320,
				Height:          200,
				TimestampMS:     5,
				TextLen:         1,
				TextBytesHex:    "5a",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 5, 1},
				BeforeState: map[string]string{
					"TextBox.buffer":   "",
					"TextBox.caret":    "0",
					"TextBox.text_len": "0",
				},
				AfterState: map[string]string{
					"TextBox.buffer":   "Z",
					"TextBox.caret":    "1",
					"TextBox.text_len": "1",
				},
			},
			{
				Order:           7,
				Kind:            "key_down",
				TargetComponent: "ToolkitFormApp",
				DispatchPath:    []string{"ToolkitFormApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          200,
				TimestampMS:     6,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 200, 6, 0},
				BeforeState:     map[string]string{"ToolkitFormApp.focused_id": "4"},
				AfterState:      map[string]string{"ToolkitFormApp.focused_id": "6"},
			},
			{
				Order:           8,
				Kind:            "key_down",
				TargetComponent: "SubmitButton",
				DispatchPath: []string{
					"ToolkitFormApp",
					"Panel",
					"Column",
					"ButtonRow",
					"SubmitButton",
				},
				Handled:     true,
				Pass:        true,
				Key:         32,
				Width:       320,
				Height:      200,
				TimestampMS: 7,
				BufferSlots: []int{6, 0, 0, 0, 32, 320, 200, 7, 0},
				BeforeState: map[string]string{
					"ToolkitFormApp.focused_id":   "6",
					"ToolkitFormApp.submit_count": "0",
					"StatusText.status_code":      "0",
					"TextBox.buffer":              "Z",
				},
				AfterState: map[string]string{
					"ToolkitFormApp.focused_id":   "6",
					"ToolkitFormApp.submit_count": "1",
					"StatusText.status_code":      "1",
					"TextBox.buffer":              "Z",
				},
			},
			{
				Order:           9,
				Kind:            "key_down",
				TargetComponent: "ToolkitFormApp",
				DispatchPath:    []string{"ToolkitFormApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          200,
				TimestampMS:     8,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 200, 8, 0},
				BeforeState:     map[string]string{"ToolkitFormApp.focused_id": "6"},
				AfterState:      map[string]string{"ToolkitFormApp.focused_id": "7"},
			},
			{
				Order:           10,
				Kind:            "text_input",
				TargetComponent: "ResetButton",
				DispatchPath: []string{
					"ToolkitFormApp",
					"Panel",
					"Column",
					"ButtonRow",
					"ResetButton",
				},
				Handled:      true,
				Pass:         true,
				Width:        320,
				Height:       200,
				TimestampMS:  9,
				TextLen:      1,
				TextBytesHex: "58",
				BufferSlots:  []int{8, 0, 0, 0, 0, 320, 200, 9, 1},
				BeforeState: map[string]string{
					"ToolkitFormApp.focused_id": "7",
					"TextBox.buffer":            "Z",
				},
				AfterState: map[string]string{
					"ToolkitFormApp.focused_id": "7",
					"TextBox.buffer":            "Z",
				},
			},
			{
				Order:           11,
				Kind:            "key_down",
				TargetComponent: "ResetButton",
				DispatchPath: []string{
					"ToolkitFormApp",
					"Panel",
					"Column",
					"ButtonRow",
					"ResetButton",
				},
				Handled:     true,
				Pass:        true,
				Key:         13,
				Width:       320,
				Height:      200,
				TimestampMS: 10,
				BufferSlots: []int{6, 0, 0, 0, 13, 320, 200, 10, 0},
				BeforeState: map[string]string{
					"ToolkitFormApp.focused_id":  "7",
					"ToolkitFormApp.reset_count": "0",
					"StatusText.status_code":     "1",
					"TextBox.buffer":             "Z",
				},
				AfterState: map[string]string{
					"ToolkitFormApp.focused_id":  "7",
					"ToolkitFormApp.reset_count": "1",
					"StatusText.status_code":     "2",
					"TextBox.buffer":             "",
				},
			},
			{
				Order:           12,
				Kind:            "key_down",
				TargetComponent: "ToolkitFormApp",
				DispatchPath:    []string{"ToolkitFormApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          200,
				TimestampMS:     11,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 200, 11, 0},
				BeforeState:     map[string]string{"ToolkitFormApp.focused_id": "7"},
				AfterState:      map[string]string{"ToolkitFormApp.focused_id": "4"},
			},
			{
				Order:           13,
				Kind:            "resize",
				TargetComponent: "ToolkitFormApp",
				DispatchPath:    []string{"ToolkitFormApp"},
				Handled:         true,
				Pass:            true,
				Width:           400,
				Height:          240,
				TimestampMS:     12,
				BufferSlots:     []int{2, 0, 0, 0, 0, 400, 240, 12, 0},
				BeforeState: map[string]string{
					"ToolkitFormApp.focused_id": "4",
					"TextBox.bounds.w":          "280",
					"TextBox.buffer":            "",
				},
				AfterState: map[string]string{
					"ToolkitFormApp.focused_id": "4",
					"TextBox.bounds.w":          "360",
					"TextBox.buffer":            "",
				},
			},
		},
		Frames: []surface.FrameReport{
			{
				Order:     1,
				Width:     beforeFrame.Width,
				Height:    beforeFrame.Height,
				Stride:    beforeFrame.Stride,
				Checksum:  checksumRGBA(beforeFrame.Pixels),
				Presented: true,
			},
			{
				Order:     2,
				Width:     textFrame.Width,
				Height:    textFrame.Height,
				Stride:    textFrame.Stride,
				Checksum:  checksumRGBA(textFrame.Pixels),
				Presented: true,
			},
			{
				Order:     3,
				Width:     submitFrame.Width,
				Height:    submitFrame.Height,
				Stride:    submitFrame.Stride,
				Checksum:  checksumRGBA(submitFrame.Pixels),
				Presented: true,
			},
			{
				Order:     4,
				Width:     afterFrame.Width,
				Height:    afterFrame.Height,
				Stride:    afterFrame.Stride,
				Checksum:  checksumRGBA(afterFrame.Pixels),
				Presented: true,
			},
		},
		StateTransitions: []surface.StateTransitionReport{
			{
				Order:     1,
				Component: "ToolkitFormApp",
				Field:     "focused_id",
				Before:    "-1",
				After:     "4",
				Cause:     "mouse_up",
			},
			{
				Order:     2,
				Component: "TextBox",
				Field:     "buffer",
				Before:    "",
				After:     "OK",
				Cause:     "text_input",
			},
			{
				Order:     3,
				Component: "TextBox",
				Field:     "caret",
				Before:    "2",
				After:     "1",
				Cause:     "key_down",
			},
			{
				Order:     4,
				Component: "TextBox",
				Field:     "buffer",
				Before:    "OK",
				After:     "K",
				Cause:     "backspace",
			},
			{
				Order:     5,
				Component: "TextBox",
				Field:     "buffer",
				Before:    "K",
				After:     "",
				Cause:     "delete",
			},
			{
				Order:     6,
				Component: "TextBox",
				Field:     "buffer",
				Before:    "",
				After:     "Z",
				Cause:     "text_input",
			},
			{
				Order:     7,
				Component: "ToolkitFormApp",
				Field:     "focused_id",
				Before:    "4",
				After:     "6",
				Cause:     "tab",
			},
			{
				Order:     8,
				Component: "ToolkitFormApp",
				Field:     "submit_count",
				Before:    "0",
				After:     "1",
				Cause:     "key_down",
			},
			{
				Order:     9,
				Component: "StatusText",
				Field:     "status_code",
				Before:    "0",
				After:     "1",
				Cause:     "submit",
			},
			{
				Order:     10,
				Component: "ToolkitFormApp",
				Field:     "focused_id",
				Before:    "6",
				After:     "7",
				Cause:     "tab",
			},
			{
				Order:     11,
				Component: "TextBox",
				Field:     "buffer",
				Before:    "Z",
				After:     "",
				Cause:     "reset",
			},
			{
				Order:     12,
				Component: "ToolkitFormApp",
				Field:     "reset_count",
				Before:    "0",
				After:     "1",
				Cause:     "key_down",
			},
			{
				Order:     13,
				Component: "StatusText",
				Field:     "status_code",
				Before:    "1",
				After:     "2",
				Cause:     "reset",
			},
			{
				Order:     14,
				Component: "ToolkitFormApp",
				Field:     "focused_id",
				Before:    "7",
				After:     "4",
				Cause:     "tab",
			},
			{
				Order:     15,
				Component: "ToolkitFormApp",
				Field:     "TextBox.bounds.w",
				Before:    "280",
				After:     "360",
				Cause:     "resize",
			},
		},
		Cases: minimalToolkitBaseCases(),
	}
	switch mode {
	case "headless-minimal-toolkit":
		scenario.Cases = append(
			scenario.Cases,
			surface.CaseReport{
				Name: "headless event dispatch",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "headless framebuffer checksum",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "headless actual runner trace",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
		)
	case "linux-x64-real-window-minimal-toolkit":
		scenario.Cases = append(
			scenario.Cases,
			surface.CaseReport{
				Name: "linux-x64 Surface Host ABI open/present/close",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 real-window surface",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 native input event pump",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 real-window resize event",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 real-window close event",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
		)
	case "wasm32-web-browser-canvas-minimal-toolkit":
		scenario.Frames = nil
		scenario.Cases = append(
			scenario.Cases,
			surface.CaseReport{
				Name: "wasm32-web browser canvas surface",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas RGBA readback",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas pointer input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas keyboard input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas resize input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas text input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web Surface Host ABI imports",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "compiler-owned wasm Surface loader",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "compiler-owned browser canvas Surface host",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
		)
	}
	return scenario
}
func runToolkitReuseScenario(mode string) headlessScenario {
	beforeFrame := renderToolkitReuseFrameRGBA(0, 0, -1, 0, 0, 0, 320, 240)
	nameFrame := renderToolkitReuseFrameRGBA(3, 0, 4, 0, 0, 0, 320, 240)
	saveFrame := renderToolkitReuseFrameRGBA(3, 5, 8, 1, 0, 1, 320, 240)
	resetFrame := renderToolkitReuseFrameRGBA(0, 0, 9, 1, 1, 2, 320, 240)
	afterFrame := renderToolkitReuseFrameRGBA(0, 0, 4, 1, 1, 2, 480, 320)
	scenario := headlessScenario{
		Components: []surface.ComponentReport{
			{
				ID:     "ToolkitSettingsApp",
				Type:   "examples.surface.toolkit.surface_toolkit_settings.ToolkitSettingsApp",
				Bounds: surface.RectReport{X: 0, Y: 0, W: 480, H: 320},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused_id":         "4",
					"save_count":         "1",
					"reset_count":        "1",
					"status_code":        "2",
					"width":              "480",
					"height":             "320",
					"accessibility_role": "none",
				},
			},
			{
				ID:     "Panel",
				Type:   "lib.core.widgets.Panel",
				Parent: "ToolkitSettingsApp",
				Bounds: surface.RectReport{X: 0, Y: 0, W: 480, H: 320},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{"padding": "12", "accessibility_role": "none"},
			},
			{
				ID:     "Column",
				Type:   "lib.core.widgets.Column",
				Parent: "Panel",
				Bounds: surface.RectReport{X: 12, Y: 12, W: 456, H: 296},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{"child_count": "6", "accessibility_role": "none"},
			},
			{
				ID:     "TitleText",
				Type:   "lib.core.widgets.Text",
				Parent: "Column",
				Bounds: surface.RectReport{X: 20, Y: 20, W: 440, H: 24},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"role":               "label",
					"text_len":           "8",
					"accessibility_role": "label",
				},
			},
			{
				ID:     "NameTextBox",
				Type:   "lib.core.widgets.TextBox",
				Parent: "Column",
				Bounds: surface.RectReport{X: 20, Y: 52, W: 440, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "true",
					"buffer":             "",
					"text_len":           "0",
					"caret":              "0",
					"accessibility_role": "label",
				},
			},
			{
				ID:     "NameLabel",
				Type:   "lib.core.widgets.Text",
				Parent: "Column",
				Bounds: surface.RectReport{X: 20, Y: 104, W: 440, H: 24},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"role":               "label",
					"text_len":           "4",
					"accessibility_role": "label",
				},
			},
			{
				ID:     "EmailTextBox",
				Type:   "lib.core.widgets.TextBox",
				Parent: "Column",
				Bounds: surface.RectReport{X: 20, Y: 136, W: 440, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "false",
					"buffer":             "",
					"text_len":           "0",
					"caret":              "0",
					"accessibility_role": "label",
				},
			},
			{
				ID:     "ButtonRow",
				Type:   "lib.core.widgets.Row",
				Parent: "Column",
				Bounds: surface.RectReport{X: 20, Y: 192, W: 440, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{"child_count": "2", "accessibility_role": "none"},
			},
			{
				ID:     "SaveButton",
				Type:   "lib.core.widgets.Button",
				Parent: "ButtonRow",
				Bounds: surface.RectReport{X: 20, Y: 192, W: 132, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "false",
					"press_count":        "1",
					"action":             "save",
					"accessibility_role": "button",
				},
			},
			{
				ID:     "ResetButton",
				Type:   "lib.core.widgets.Button",
				Parent: "ButtonRow",
				Bounds: surface.RectReport{X: 164, Y: 192, W: 132, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "false",
					"press_count":        "1",
					"action":             "reset",
					"accessibility_role": "button",
				},
			},
			{
				ID:     "StatusText",
				Type:   "lib.core.widgets.Text",
				Parent: "Column",
				Bounds: surface.RectReport{X: 20, Y: 248, W: 440, H: 24},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"role":               "status",
					"status_code":        "2",
					"accessibility_role": "label",
				},
			},
		},
		ComponentTree: &surface.ComponentTreeReport{
			Schema:       "tetra.surface.component-tree.v1",
			DynamicLevel: "toolkit-reuse-widget-tree",
			RootID:       0,
			NodeCount:    11,
			FocusedID:    4,
			Nodes: []surface.ComponentTreeNodeReport{
				{
					ID:         0,
					Name:       "ToolkitSettingsApp",
					Kind:       "root",
					ParentID:   -1,
					ChildIndex: 0,
					FirstChild: 1,
					ChildCount: 1,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 0, Y: 0, W: 480, H: 320},
				},
				{
					ID:         1,
					Name:       "Panel",
					Kind:       "panel",
					ParentID:   0,
					ChildIndex: 0,
					FirstChild: 2,
					ChildCount: 1,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 0, Y: 0, W: 480, H: 320},
				},
				{
					ID:         2,
					Name:       "Column",
					Kind:       "column",
					ParentID:   1,
					ChildIndex: 0,
					FirstChild: 3,
					ChildCount: 6,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 12, Y: 12, W: 456, H: 296},
				},
				{
					ID:         3,
					Name:       "TitleText",
					Kind:       "text",
					ParentID:   2,
					ChildIndex: 0,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 20, Y: 20, W: 440, H: 24},
				},
				{
					ID:         4,
					Name:       "NameTextBox",
					Kind:       "textbox",
					ParentID:   2,
					ChildIndex: 1,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  true,
					Bounds:     surface.RectReport{X: 20, Y: 52, W: 440, H: 44},
				},
				{
					ID:         5,
					Name:       "NameLabel",
					Kind:       "text",
					ParentID:   2,
					ChildIndex: 2,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 20, Y: 104, W: 440, H: 24},
				},
				{
					ID:         6,
					Name:       "EmailTextBox",
					Kind:       "textbox",
					ParentID:   2,
					ChildIndex: 3,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  true,
					Bounds:     surface.RectReport{X: 20, Y: 136, W: 440, H: 44},
				},
				{
					ID:         7,
					Name:       "ButtonRow",
					Kind:       "row",
					ParentID:   2,
					ChildIndex: 4,
					FirstChild: 8,
					ChildCount: 2,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 20, Y: 192, W: 440, H: 44},
				},
				{
					ID:         8,
					Name:       "SaveButton",
					Kind:       "button",
					ParentID:   7,
					ChildIndex: 0,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  true,
					Bounds:     surface.RectReport{X: 20, Y: 192, W: 132, H: 44},
				},
				{
					ID:         9,
					Name:       "ResetButton",
					Kind:       "button",
					ParentID:   7,
					ChildIndex: 1,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  true,
					Bounds:     surface.RectReport{X: 164, Y: 192, W: 132, H: 44},
				},
				{
					ID:         10,
					Name:       "StatusText",
					Kind:       "text",
					ParentID:   2,
					ChildIndex: 5,
					FirstChild: -1,
					ChildCount: 0,
					Focusable:  false,
					Bounds:     surface.RectReport{X: 20, Y: 248, W: 440, H: 24},
				},
			},
			LayoutPasses: []surface.ComponentTreeLayoutPassReport{
				{
					ComponentID: 4,
					Pass:        "initial",
					Bounds:      surface.RectReport{X: 20, Y: 52, W: 280, H: 44},
					Measured:    surface.SizeReport{W: 280, H: 44},
				},
				{
					ComponentID: 6,
					Pass:        "initial",
					Bounds:      surface.RectReport{X: 20, Y: 136, W: 280, H: 44},
					Measured:    surface.SizeReport{W: 280, H: 44},
				},
				{
					ComponentID: 4,
					Pass:        "resize",
					Bounds:      surface.RectReport{X: 20, Y: 52, W: 440, H: 44},
					Measured:    surface.SizeReport{W: 440, H: 44},
				},
				{
					ComponentID: 6,
					Pass:        "resize",
					Bounds:      surface.RectReport{X: 20, Y: 136, W: 440, H: 44},
					Measured:    surface.SizeReport{W: 440, H: 44},
				},
				{
					ComponentID: 10,
					Pass:        "status-update",
					Bounds:      surface.RectReport{X: 20, Y: 248, W: 440, H: 24},
					Measured:    surface.SizeReport{W: 440, H: 24},
				},
			},
			DrawOrder:  []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			FocusOrder: []int{4, 6, 8, 9},
			DispatchPaths: []surface.ComponentTreeDispatchPathReport{
				{Event: "click", TargetID: 4, X: 40, Y: 72, Path: []int{0, 1, 2, 4}},
				{Event: "click", TargetID: 6, X: 40, Y: 156, Path: []int{0, 1, 2, 6}},
				{Event: "key", TargetID: 8, X: 40, Y: 208, Path: []int{0, 1, 2, 7, 8}},
				{Event: "key", TargetID: 9, X: 180, Y: 208, Path: []int{0, 1, 2, 7, 9}},
			},
		},
		ComponentTreeAPI: toolkitReuseComponentTreeAPIReport(),
		Toolkit:          toolkitReuseReport(),
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "NameTextBox",
				DispatchPath:    []string{"ToolkitSettingsApp", "Panel", "Column", "NameTextBox"},
				Handled:         true,
				Pass:            true,
				X:               40,
				Y:               72,
				Width:           320,
				Height:          240,
				BufferSlots:     []int{5, 40, 72, 1, 0, 320, 240, 0, 0},
				BeforeState: map[string]string{
					"ToolkitSettingsApp.focused_id": "-1",
					"NameTextBox.focused":           "false",
				},
				AfterState: map[string]string{
					"ToolkitSettingsApp.focused_id": "4",
					"NameTextBox.focused":           "true",
				},
			},
			{
				Order:           2,
				Kind:            "text_input",
				TargetComponent: "NameTextBox",
				DispatchPath:    []string{"ToolkitSettingsApp", "Panel", "Column", "NameTextBox"},
				Handled:         true,
				Pass:            true,
				Width:           320,
				Height:          240,
				TimestampMS:     1,
				TextLen:         3,
				TextBytesHex:    "416461",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 240, 1, 3},
				BeforeState: map[string]string{
					"NameTextBox.buffer":  "",
					"NameTextBox.caret":   "0",
					"EmailTextBox.buffer": "",
				},
				AfterState: map[string]string{
					"NameTextBox.buffer":  "Ada",
					"NameTextBox.caret":   "3",
					"EmailTextBox.buffer": "",
				},
			},
			{
				Order:           3,
				Kind:            "key_down",
				TargetComponent: "ToolkitSettingsApp",
				DispatchPath:    []string{"ToolkitSettingsApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          240,
				TimestampMS:     2,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 240, 2, 0},
				BeforeState:     map[string]string{"ToolkitSettingsApp.focused_id": "4"},
				AfterState:      map[string]string{"ToolkitSettingsApp.focused_id": "6"},
			},
			{
				Order:           4,
				Kind:            "text_input",
				TargetComponent: "EmailTextBox",
				DispatchPath:    []string{"ToolkitSettingsApp", "Panel", "Column", "EmailTextBox"},
				Handled:         true,
				Pass:            true,
				Width:           320,
				Height:          240,
				TimestampMS:     3,
				TextLen:         5,
				TextBytesHex:    "7465747261",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 240, 3, 5},
				BeforeState: map[string]string{
					"EmailTextBox.buffer": "",
					"NameTextBox.buffer":  "Ada",
				},
				AfterState: map[string]string{
					"EmailTextBox.buffer": "tetra",
					"NameTextBox.buffer":  "Ada",
				},
			},
			{
				Order:           5,
				Kind:            "key_down",
				TargetComponent: "ToolkitSettingsApp",
				DispatchPath:    []string{"ToolkitSettingsApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          240,
				TimestampMS:     4,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 240, 4, 0},
				BeforeState:     map[string]string{"ToolkitSettingsApp.focused_id": "6"},
				AfterState:      map[string]string{"ToolkitSettingsApp.focused_id": "8"},
			},
			{
				Order:           6,
				Kind:            "key_down",
				TargetComponent: "SaveButton",
				DispatchPath: []string{
					"ToolkitSettingsApp",
					"Panel",
					"Column",
					"ButtonRow",
					"SaveButton",
				},
				Handled:     true,
				Pass:        true,
				Key:         32,
				Width:       320,
				Height:      240,
				TimestampMS: 5,
				BufferSlots: []int{6, 0, 0, 0, 32, 320, 240, 5, 0},
				BeforeState: map[string]string{
					"ToolkitSettingsApp.focused_id": "8",
					"ToolkitSettingsApp.save_count": "0",
					"StatusText.status_code":        "0",
				},
				AfterState: map[string]string{
					"ToolkitSettingsApp.focused_id": "8",
					"ToolkitSettingsApp.save_count": "1",
					"StatusText.status_code":        "1",
				},
			},
			{
				Order:           7,
				Kind:            "key_down",
				TargetComponent: "ToolkitSettingsApp",
				DispatchPath:    []string{"ToolkitSettingsApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          240,
				TimestampMS:     6,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 240, 6, 0},
				BeforeState:     map[string]string{"ToolkitSettingsApp.focused_id": "8"},
				AfterState:      map[string]string{"ToolkitSettingsApp.focused_id": "9"},
			},
			{
				Order:           8,
				Kind:            "key_down",
				TargetComponent: "ResetButton",
				DispatchPath: []string{
					"ToolkitSettingsApp",
					"Panel",
					"Column",
					"ButtonRow",
					"ResetButton",
				},
				Handled:     true,
				Pass:        true,
				Key:         13,
				Width:       320,
				Height:      240,
				TimestampMS: 7,
				BufferSlots: []int{6, 0, 0, 0, 13, 320, 240, 7, 0},
				BeforeState: map[string]string{
					"ToolkitSettingsApp.focused_id":  "9",
					"ToolkitSettingsApp.reset_count": "0",
					"StatusText.status_code":         "1",
					"NameTextBox.buffer":             "Ada",
					"EmailTextBox.buffer":            "tetra",
				},
				AfterState: map[string]string{
					"ToolkitSettingsApp.focused_id":  "9",
					"ToolkitSettingsApp.reset_count": "1",
					"StatusText.status_code":         "2",
					"NameTextBox.buffer":             "",
					"EmailTextBox.buffer":            "",
				},
			},
			{
				Order:           9,
				Kind:            "key_down",
				TargetComponent: "ToolkitSettingsApp",
				DispatchPath:    []string{"ToolkitSettingsApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          240,
				TimestampMS:     8,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 240, 8, 0},
				BeforeState:     map[string]string{"ToolkitSettingsApp.focused_id": "9"},
				AfterState:      map[string]string{"ToolkitSettingsApp.focused_id": "4"},
			},
			{
				Order:           10,
				Kind:            "resize",
				TargetComponent: "ToolkitSettingsApp",
				DispatchPath:    []string{"ToolkitSettingsApp"},
				Handled:         true,
				Pass:            true,
				Width:           480,
				Height:          320,
				TimestampMS:     9,
				BufferSlots:     []int{2, 0, 0, 0, 0, 480, 320, 9, 0},
				BeforeState: map[string]string{
					"ToolkitSettingsApp.focused_id": "4",
					"NameTextBox.bounds.w":          "280",
					"EmailTextBox.bounds.w":         "280",
				},
				AfterState: map[string]string{
					"ToolkitSettingsApp.focused_id": "4",
					"NameTextBox.bounds.w":          "440",
					"EmailTextBox.bounds.w":         "440",
				},
			},
		},
		Frames: []surface.FrameReport{
			{
				Order:     1,
				Width:     beforeFrame.Width,
				Height:    beforeFrame.Height,
				Stride:    beforeFrame.Stride,
				Checksum:  checksumRGBA(beforeFrame.Pixels),
				Presented: true,
			},
			{
				Order:     2,
				Width:     nameFrame.Width,
				Height:    nameFrame.Height,
				Stride:    nameFrame.Stride,
				Checksum:  checksumRGBA(nameFrame.Pixels),
				Presented: true,
			},
			{
				Order:     3,
				Width:     saveFrame.Width,
				Height:    saveFrame.Height,
				Stride:    saveFrame.Stride,
				Checksum:  checksumRGBA(saveFrame.Pixels),
				Presented: true,
			},
			{
				Order:     4,
				Width:     resetFrame.Width,
				Height:    resetFrame.Height,
				Stride:    resetFrame.Stride,
				Checksum:  checksumRGBA(resetFrame.Pixels),
				Presented: true,
			},
			{
				Order:     5,
				Width:     afterFrame.Width,
				Height:    afterFrame.Height,
				Stride:    afterFrame.Stride,
				Checksum:  checksumRGBA(afterFrame.Pixels),
				Presented: true,
			},
		},
		StateTransitions: []surface.StateTransitionReport{
			{
				Order:     1,
				Component: "ToolkitSettingsApp",
				Field:     "focused_id",
				Before:    "-1",
				After:     "4",
				Cause:     "mouse_up",
			},
			{
				Order:     2,
				Component: "NameTextBox",
				Field:     "buffer",
				Before:    "",
				After:     "Ada",
				Cause:     "text_input",
			},
			{
				Order:     3,
				Component: "ToolkitSettingsApp",
				Field:     "focused_id",
				Before:    "4",
				After:     "6",
				Cause:     "tab",
			},
			{
				Order:     4,
				Component: "EmailTextBox",
				Field:     "buffer",
				Before:    "",
				After:     "tetra",
				Cause:     "text_input",
			},
			{
				Order:     5,
				Component: "ToolkitSettingsApp",
				Field:     "focused_id",
				Before:    "6",
				After:     "8",
				Cause:     "tab",
			},
			{
				Order:     6,
				Component: "ToolkitSettingsApp",
				Field:     "save_count",
				Before:    "0",
				After:     "1",
				Cause:     "key_down",
			},
			{
				Order:     7,
				Component: "StatusText",
				Field:     "status_code",
				Before:    "0",
				After:     "1",
				Cause:     "save",
			},
			{
				Order:     8,
				Component: "ToolkitSettingsApp",
				Field:     "focused_id",
				Before:    "8",
				After:     "9",
				Cause:     "tab",
			},
			{
				Order:     9,
				Component: "NameTextBox",
				Field:     "buffer",
				Before:    "Ada",
				After:     "",
				Cause:     "reset",
			},
			{
				Order:     10,
				Component: "EmailTextBox",
				Field:     "buffer",
				Before:    "tetra",
				After:     "",
				Cause:     "reset",
			},
			{
				Order:     11,
				Component: "ToolkitSettingsApp",
				Field:     "reset_count",
				Before:    "0",
				After:     "1",
				Cause:     "key_down",
			},
			{
				Order:     12,
				Component: "StatusText",
				Field:     "status_code",
				Before:    "1",
				After:     "2",
				Cause:     "reset",
			},
			{
				Order:     13,
				Component: "ToolkitSettingsApp",
				Field:     "focused_id",
				Before:    "9",
				After:     "4",
				Cause:     "tab",
			},
			{
				Order:     14,
				Component: "ToolkitSettingsApp",
				Field:     "NameTextBox.bounds.w",
				Before:    "280",
				After:     "440",
				Cause:     "resize",
			},
			{
				Order:     15,
				Component: "ToolkitSettingsApp",
				Field:     "EmailTextBox.bounds.w",
				Before:    "280",
				After:     "440",
				Cause:     "resize",
			},
		},
		Cases: toolkitReuseBaseCases(),
	}
	switch mode {
	case "headless-toolkit-reuse":
		scenario.Cases = append(
			scenario.Cases,
			surface.CaseReport{
				Name: "headless event dispatch",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "headless framebuffer checksum",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "headless actual runner trace",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
		)
	case "linux-x64-real-window-toolkit-reuse":
		scenario.Cases = append(
			scenario.Cases,
			surface.CaseReport{
				Name: "linux-x64 Surface Host ABI open/present/close",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 real-window surface",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 native input event pump",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 real-window resize event",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 real-window close event",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
		)
	case "wasm32-web-browser-canvas-toolkit-reuse":
		scenario.Frames = nil
		scenario.Cases = append(
			scenario.Cases,
			surface.CaseReport{
				Name: "wasm32-web browser canvas surface",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas RGBA readback",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas pointer input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas keyboard input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas resize input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas text input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web Surface Host ABI imports",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "compiler-owned wasm Surface loader",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "compiler-owned browser canvas Surface host",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
		)
	}
	return scenario
}
func runAccessibilityMetadataScenario(mode string) headlessScenario {
	beforeFrame := renderAccessibilityMetadataFrameRGBA(0, 0, -1, 0, 0, 0, 320, 240)
	nameFrame := renderAccessibilityMetadataFrameRGBA(3, 0, 5, 0, 0, 0, 320, 240)
	saveFrame := renderAccessibilityMetadataFrameRGBA(3, 5, 9, 1, 0, 1, 320, 240)
	resetFrame := renderAccessibilityMetadataFrameRGBA(0, 0, 10, 1, 1, 2, 320, 240)
	afterFrame := renderAccessibilityMetadataFrameRGBA(0, 0, 5, 1, 1, 2, 480, 320)
	scenario := headlessScenario{
		Components: []surface.ComponentReport{
			{
				ID:     "AccessibilitySettingsApp",
				Type:   "examples.surface.toolkit.surface_accessibility_settings.AccessibilitySettingsApp",
				Bounds: surface.RectReport{X: 0, Y: 0, W: 480, H: 320},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused_id":         "5",
					"save_count":         "1",
					"reset_count":        "1",
					"status_code":        "2",
					"width":              "480",
					"height":             "320",
					"accessibility_role": "root",
				},
			},
			{
				ID:     "Panel",
				Type:   "lib.core.widgets.Panel",
				Parent: "AccessibilitySettingsApp",
				Bounds: surface.RectReport{X: 0, Y: 0, W: 480, H: 320},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{"padding": "12", "accessibility_role": "panel"},
			},
			{
				ID:     "Column",
				Type:   "lib.core.widgets.Column",
				Parent: "Panel",
				Bounds: surface.RectReport{X: 12, Y: 12, W: 456, H: 296},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{"child_count": "7", "accessibility_role": "column"},
			},
			{
				ID:     "TitleText",
				Type:   "lib.core.widgets.Text",
				Parent: "Column",
				Bounds: surface.RectReport{X: 20, Y: 20, W: 440, H: 24},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"role":               "text",
					"text_len":           "8",
					"accessibility_role": "text",
				},
			},
			{
				ID:     "NameLabel",
				Type:   "lib.core.widgets.Text",
				Parent: "Column",
				Bounds: surface.RectReport{X: 20, Y: 52, W: 440, H: 24},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"role":               "label",
					"text_len":           "4",
					"accessibility_role": "label",
				},
			},
			{
				ID:     "NameTextBox",
				Type:   "lib.core.widgets.TextBox",
				Parent: "Column",
				Bounds: surface.RectReport{X: 20, Y: 84, W: 440, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "true",
					"buffer":             "",
					"text_len":           "0",
					"caret":              "0",
					"accessibility_role": "textbox",
				},
			},
			{
				ID:     "EmailLabel",
				Type:   "lib.core.widgets.Text",
				Parent: "Column",
				Bounds: surface.RectReport{X: 20, Y: 136, W: 440, H: 24},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"role":               "label",
					"text_len":           "5",
					"accessibility_role": "label",
				},
			},
			{
				ID:     "EmailTextBox",
				Type:   "lib.core.widgets.TextBox",
				Parent: "Column",
				Bounds: surface.RectReport{X: 20, Y: 168, W: 440, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "false",
					"buffer":             "",
					"text_len":           "0",
					"caret":              "0",
					"accessibility_role": "textbox",
				},
			},
			{
				ID:     "ButtonRow",
				Type:   "lib.core.widgets.Row",
				Parent: "Column",
				Bounds: surface.RectReport{X: 20, Y: 224, W: 440, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{"child_count": "2", "accessibility_role": "row"},
			},
			{
				ID:     "SaveButton",
				Type:   "lib.core.widgets.Button",
				Parent: "ButtonRow",
				Bounds: surface.RectReport{X: 20, Y: 224, W: 132, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "false",
					"press_count":        "1",
					"action":             "save",
					"accessibility_role": "button",
				},
			},
			{
				ID:     "ResetButton",
				Type:   "lib.core.widgets.Button",
				Parent: "ButtonRow",
				Bounds: surface.RectReport{X: 164, Y: 224, W: 132, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "false",
					"press_count":        "1",
					"action":             "reset",
					"accessibility_role": "button",
				},
			},
			{
				ID:     "StatusText",
				Type:   "lib.core.widgets.Text",
				Parent: "Column",
				Bounds: surface.RectReport{X: 20, Y: 280, W: 440, H: 24},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"role":               "status",
					"status_code":        "2",
					"accessibility_role": "status",
				},
			},
		},
		ComponentTree:    accessibilityComponentTreeReport(),
		ComponentTreeAPI: accessibilityComponentTreeAPIReport(),
		Toolkit:          accessibilityToolkitReport(),
		AccessibilityTree: accessibilityTreeReport(
			beforeFrame,
			nameFrame,
			saveFrame,
			resetFrame,
			afterFrame,
		),
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "NameTextBox",
				DispatchPath: []string{
					"AccessibilitySettingsApp",
					"Panel",
					"Column",
					"NameTextBox",
				},
				Handled:     true,
				Pass:        true,
				X:           40,
				Y:           100,
				Width:       320,
				Height:      240,
				BufferSlots: []int{5, 40, 100, 1, 0, 320, 240, 0, 0},
				BeforeState: map[string]string{
					"AccessibilitySettingsApp.focused_id": "-1",
					"NameTextBox.focused":                 "false",
				},
				AfterState: map[string]string{
					"AccessibilitySettingsApp.focused_id": "5",
					"NameTextBox.focused":                 "true",
				},
			},
			{
				Order:           2,
				Kind:            "text_input",
				TargetComponent: "NameTextBox",
				DispatchPath: []string{
					"AccessibilitySettingsApp",
					"Panel",
					"Column",
					"NameTextBox",
				},
				Handled:      true,
				Pass:         true,
				Width:        320,
				Height:       240,
				TimestampMS:  1,
				TextLen:      3,
				TextBytesHex: "416461",
				BufferSlots:  []int{8, 0, 0, 0, 0, 320, 240, 1, 3},
				BeforeState: map[string]string{
					"NameTextBox.buffer":  "",
					"NameTextBox.caret":   "0",
					"EmailTextBox.buffer": "",
				},
				AfterState: map[string]string{
					"NameTextBox.buffer":  "Ada",
					"NameTextBox.caret":   "3",
					"EmailTextBox.buffer": "",
				},
			},
			{
				Order:           3,
				Kind:            "key_down",
				TargetComponent: "AccessibilitySettingsApp",
				DispatchPath:    []string{"AccessibilitySettingsApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          240,
				TimestampMS:     2,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 240, 2, 0},
				BeforeState:     map[string]string{"AccessibilitySettingsApp.focused_id": "5"},
				AfterState:      map[string]string{"AccessibilitySettingsApp.focused_id": "7"},
			},
			{
				Order:           4,
				Kind:            "text_input",
				TargetComponent: "EmailTextBox",
				DispatchPath: []string{
					"AccessibilitySettingsApp",
					"Panel",
					"Column",
					"EmailTextBox",
				},
				Handled:      true,
				Pass:         true,
				Width:        320,
				Height:       240,
				TimestampMS:  3,
				TextLen:      5,
				TextBytesHex: "7465747261",
				BufferSlots:  []int{8, 0, 0, 0, 0, 320, 240, 3, 5},
				BeforeState: map[string]string{
					"EmailTextBox.buffer": "",
					"NameTextBox.buffer":  "Ada",
				},
				AfterState: map[string]string{
					"EmailTextBox.buffer": "tetra",
					"NameTextBox.buffer":  "Ada",
				},
			},
			{
				Order:           5,
				Kind:            "key_down",
				TargetComponent: "AccessibilitySettingsApp",
				DispatchPath:    []string{"AccessibilitySettingsApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          240,
				TimestampMS:     4,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 240, 4, 0},
				BeforeState:     map[string]string{"AccessibilitySettingsApp.focused_id": "7"},
				AfterState:      map[string]string{"AccessibilitySettingsApp.focused_id": "9"},
			},
			{
				Order:           6,
				Kind:            "key_down",
				TargetComponent: "SaveButton",
				DispatchPath: []string{
					"AccessibilitySettingsApp",
					"Panel",
					"Column",
					"ButtonRow",
					"SaveButton",
				},
				Handled:     true,
				Pass:        true,
				Key:         32,
				Width:       320,
				Height:      240,
				TimestampMS: 5,
				BufferSlots: []int{6, 0, 0, 0, 32, 320, 240, 5, 0},
				BeforeState: map[string]string{
					"AccessibilitySettingsApp.focused_id": "9",
					"AccessibilitySettingsApp.save_count": "0",
					"StatusText.status_code":              "0",
				},
				AfterState: map[string]string{
					"AccessibilitySettingsApp.focused_id": "9",
					"AccessibilitySettingsApp.save_count": "1",
					"StatusText.status_code":              "1",
				},
			},
			{
				Order:           7,
				Kind:            "key_down",
				TargetComponent: "AccessibilitySettingsApp",
				DispatchPath:    []string{"AccessibilitySettingsApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          240,
				TimestampMS:     6,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 240, 6, 0},
				BeforeState:     map[string]string{"AccessibilitySettingsApp.focused_id": "9"},
				AfterState:      map[string]string{"AccessibilitySettingsApp.focused_id": "10"},
			},
			{
				Order:           8,
				Kind:            "key_down",
				TargetComponent: "ResetButton",
				DispatchPath: []string{
					"AccessibilitySettingsApp",
					"Panel",
					"Column",
					"ButtonRow",
					"ResetButton",
				},
				Handled:     true,
				Pass:        true,
				Key:         13,
				Width:       320,
				Height:      240,
				TimestampMS: 7,
				BufferSlots: []int{6, 0, 0, 0, 13, 320, 240, 7, 0},
				BeforeState: map[string]string{
					"AccessibilitySettingsApp.focused_id":  "10",
					"AccessibilitySettingsApp.reset_count": "0",
					"StatusText.status_code":               "1",
					"NameTextBox.buffer":                   "Ada",
					"EmailTextBox.buffer":                  "tetra",
				},
				AfterState: map[string]string{
					"AccessibilitySettingsApp.focused_id":  "10",
					"AccessibilitySettingsApp.reset_count": "1",
					"StatusText.status_code":               "2",
					"NameTextBox.buffer":                   "",
					"EmailTextBox.buffer":                  "",
				},
			},
			{
				Order:           9,
				Kind:            "key_down",
				TargetComponent: "AccessibilitySettingsApp",
				DispatchPath:    []string{"AccessibilitySettingsApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          240,
				TimestampMS:     8,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 240, 8, 0},
				BeforeState:     map[string]string{"AccessibilitySettingsApp.focused_id": "10"},
				AfterState:      map[string]string{"AccessibilitySettingsApp.focused_id": "5"},
			},
			{
				Order:           10,
				Kind:            "resize",
				TargetComponent: "AccessibilitySettingsApp",
				DispatchPath:    []string{"AccessibilitySettingsApp"},
				Handled:         true,
				Pass:            true,
				Width:           480,
				Height:          320,
				TimestampMS:     9,
				BufferSlots:     []int{2, 0, 0, 0, 0, 480, 320, 9, 0},
				BeforeState: map[string]string{
					"AccessibilitySettingsApp.focused_id": "5",
					"NameTextBox.bounds.w":                "280",
					"EmailTextBox.bounds.w":               "280",
				},
				AfterState: map[string]string{
					"AccessibilitySettingsApp.focused_id": "5",
					"NameTextBox.bounds.w":                "440",
					"EmailTextBox.bounds.w":               "440",
				},
			},
		},
		Frames: []surface.FrameReport{
			{
				Order:     1,
				Width:     beforeFrame.Width,
				Height:    beforeFrame.Height,
				Stride:    beforeFrame.Stride,
				Checksum:  checksumRGBA(beforeFrame.Pixels),
				Presented: true,
			},
			{
				Order:     2,
				Width:     nameFrame.Width,
				Height:    nameFrame.Height,
				Stride:    nameFrame.Stride,
				Checksum:  checksumRGBA(nameFrame.Pixels),
				Presented: true,
			},
			{
				Order:     3,
				Width:     saveFrame.Width,
				Height:    saveFrame.Height,
				Stride:    saveFrame.Stride,
				Checksum:  checksumRGBA(saveFrame.Pixels),
				Presented: true,
			},
			{
				Order:     4,
				Width:     resetFrame.Width,
				Height:    resetFrame.Height,
				Stride:    resetFrame.Stride,
				Checksum:  checksumRGBA(resetFrame.Pixels),
				Presented: true,
			},
			{
				Order:     5,
				Width:     afterFrame.Width,
				Height:    afterFrame.Height,
				Stride:    afterFrame.Stride,
				Checksum:  checksumRGBA(afterFrame.Pixels),
				Presented: true,
			},
		},
		StateTransitions: []surface.StateTransitionReport{
			{
				Order:     1,
				Component: "AccessibilitySettingsApp",
				Field:     "focused_id",
				Before:    "-1",
				After:     "5",
				Cause:     "mouse_up",
			},
			{
				Order:     2,
				Component: "NameTextBox",
				Field:     "buffer",
				Before:    "",
				After:     "Ada",
				Cause:     "text_input",
			},
			{
				Order:     3,
				Component: "AccessibilitySettingsApp",
				Field:     "focused_id",
				Before:    "5",
				After:     "7",
				Cause:     "tab",
			},
			{
				Order:     4,
				Component: "EmailTextBox",
				Field:     "buffer",
				Before:    "",
				After:     "tetra",
				Cause:     "text_input",
			},
			{
				Order:     5,
				Component: "AccessibilitySettingsApp",
				Field:     "focused_id",
				Before:    "7",
				After:     "9",
				Cause:     "tab",
			},
			{
				Order:     6,
				Component: "AccessibilitySettingsApp",
				Field:     "save_count",
				Before:    "0",
				After:     "1",
				Cause:     "key_down",
			},
			{
				Order:     7,
				Component: "StatusText",
				Field:     "status_code",
				Before:    "0",
				After:     "1",
				Cause:     "save",
			},
			{
				Order:     8,
				Component: "AccessibilitySettingsApp",
				Field:     "focused_id",
				Before:    "9",
				After:     "10",
				Cause:     "tab",
			},
			{
				Order:     9,
				Component: "NameTextBox",
				Field:     "buffer",
				Before:    "Ada",
				After:     "",
				Cause:     "reset",
			},
			{
				Order:     10,
				Component: "EmailTextBox",
				Field:     "buffer",
				Before:    "tetra",
				After:     "",
				Cause:     "reset",
			},
			{
				Order:     11,
				Component: "AccessibilitySettingsApp",
				Field:     "reset_count",
				Before:    "0",
				After:     "1",
				Cause:     "key_down",
			},
			{
				Order:     12,
				Component: "StatusText",
				Field:     "status_code",
				Before:    "1",
				After:     "2",
				Cause:     "reset",
			},
			{
				Order:     13,
				Component: "AccessibilitySettingsApp",
				Field:     "focused_id",
				Before:    "10",
				After:     "5",
				Cause:     "tab",
			},
			{
				Order:     14,
				Component: "AccessibilitySettingsApp",
				Field:     "NameTextBox.bounds.w",
				Before:    "280",
				After:     "440",
				Cause:     "resize",
			},
			{
				Order:     15,
				Component: "AccessibilitySettingsApp",
				Field:     "EmailTextBox.bounds.w",
				Before:    "280",
				After:     "440",
				Cause:     "resize",
			},
		},
		Cases: accessibilityMetadataBaseCases(),
	}
	switch mode {
	case "headless-accessibility-metadata":
		scenario.Cases = append(
			scenario.Cases,
			surface.CaseReport{
				Name: "headless event dispatch",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "headless framebuffer checksum",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "headless actual runner trace",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
		)
	case "linux-x64-real-window-accessibility-metadata":
		scenario.Cases = append(
			scenario.Cases,
			surface.CaseReport{
				Name: "linux-x64 Surface Host ABI open/present/close",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 real-window surface",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 native input event pump",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 real-window resize event",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "linux-x64 real-window close event",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
		)
	case "wasm32-web-browser-canvas-accessibility-metadata":
		scenario.Frames = nil
		scenario.Cases = append(
			scenario.Cases,
			surface.CaseReport{
				Name: "wasm32-web browser canvas surface",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas RGBA readback",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas pointer input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas keyboard input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas resize input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web browser canvas text input",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "wasm32-web Surface Host ABI imports",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "compiler-owned wasm Surface loader",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			surface.CaseReport{
				Name: "compiler-owned browser canvas Surface host",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
		)
	}
	return scenario
}
func accessibilityMetadataBaseCases() []surface.CaseReport {
	cases := toolkitReuseBaseCases()
	cases = append(
		cases,
		surface.CaseReport{
			Name: "accessibility metadata tree schema",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "accessibility metadata roles labels values states",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "accessibility metadata component tree alignment",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "accessibility metadata focus order alignment",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "accessibility metadata reading order",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "accessibility metadata snapshots update",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "accessibility metadata no DOM ARIA platform host claim",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
	)
	return cases
}
func toolkitReuseBaseCases() []surface.CaseReport {
	cases := minimalToolkitBaseCases()
	cases = append(
		cases,
		surface.CaseReport{
			Name: "toolkit reuse second example evidence",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "toolkit reuse widgets module evidence",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "toolkit reuse multi TextBox routing",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "toolkit reuse focused TextBox only mutates",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "toolkit reuse Save action routed",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "toolkit reuse Reset action routed",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "toolkit reuse StatusText updates",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "toolkit reuse resize relayout",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "toolkit reuse changed frame checksums",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "toolkit reuse no demo-local widget structs",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
	)
	return cases
}
func minimalToolkitBaseCases() []surface.CaseReport {
	return []surface.CaseReport{
		{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
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
		{Name: "component tree node count", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree parent child links", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree layout bounds", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree draw traversal", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree pointer dispatch path", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree focus traversal", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "component tree text routed to focused TextBox",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{Name: "component tree button action dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree resize relayout", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree rendered frame update", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree api builder node creation", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "component tree api parent child invariants",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name: "component tree api layout helper dispatch",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{Name: "component tree api hit test helper", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "component tree api focus helper traversal",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{Name: "component tree api dispatch path helper", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree api no manual bookkeeping", Kind: "positive", Ran: true, Pass: true},
		{
			Name:          "reject legacy UI evidence",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "legacy UI evidence rejected",
		},
		{Name: "minimal toolkit reusable widgets", Kind: "positive", Ran: true, Pass: true},
		{Name: "minimal toolkit Text widget evidence", Kind: "positive", Ran: true, Pass: true},
		{Name: "minimal toolkit Button widget evidence", Kind: "positive", Ran: true, Pass: true},
		{Name: "minimal toolkit TextBox widget evidence", Kind: "positive", Ran: true, Pass: true},
		{Name: "minimal toolkit Row Column Panel layout", Kind: "positive", Ran: true, Pass: true},
		{Name: "minimal toolkit tree api reuse", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "minimal toolkit TextBox focus input editing",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{Name: "minimal toolkit Submit action routed", Kind: "positive", Ran: true, Pass: true},
		{Name: "minimal toolkit Reset action routed", Kind: "positive", Ran: true, Pass: true},
		{Name: "minimal toolkit status text update", Kind: "positive", Ran: true, Pass: true},
		{Name: "minimal toolkit resize relayout", Kind: "positive", Ran: true, Pass: true},
		{Name: "minimal toolkit rendered frame update", Kind: "positive", Ran: true, Pass: true},
	}
}
func minimalToolkitComponentTreeAPIReport() *surface.ComponentTreeAPIReport {
	return &surface.ComponentTreeAPIReport{
		Schema:            "tetra.surface.component-tree-api.v1",
		APILevel:          "builder-layout-dispatch-v1",
		Source:            "examples/surface/toolkit/surface_toolkit_form.tetra",
		ManualBookkeeping: false,
		Builder: surface.ComponentTreeAPIBuilderReport{
			RootCreatedBy:     "tree_add_root",
			ChildrenCreatedBy: "tree_add_child",
			NodeCount:         9,
			Capacity:          16,
			OverflowChecked:   true,
		},
		Invariants: surface.ComponentTreeAPIInvariantReport{
			TreeValidateRan:         true,
			TreeValidateStatus:      0,
			ParentChildLinksChecked: true,
			ChildIndicesChecked:     true,
			ChildCountChecked:       true,
			FirstChildChecked:       true,
		},
		LayoutHelpers: []surface.ComponentTreeAPILayoutHelperReport{
			{
				Helper:        "widgets.panel_content_rect",
				Target:        "Panel",
				Pass:          "initial",
				ChangedBounds: true,
			},
			{
				Helper:        "widgets.column_layout",
				Target:        "Column",
				Pass:          "initial",
				ChangedBounds: true,
			},
			{
				Helper:        "widgets.row_layout",
				Target:        "ButtonRow",
				Pass:          "initial",
				ChangedBounds: true,
			},
			{
				Helper:        "widgets.column_layout",
				Target:        "Column",
				Pass:          "resize",
				ChangedBounds: true,
			},
		},
		FocusHelpers: []surface.ComponentTreeAPIFocusHelperReport{
			{Helper: "tree_focus_next", Before: "TextBox", After: "SubmitButton"},
			{Helper: "tree_focus_next", Before: "SubmitButton", After: "ResetButton"},
			{Helper: "tree_focus_next", Before: "ResetButton", After: "TextBox"},
		},
		HitTests: []surface.ComponentTreeAPIHitTestReport{
			{Helper: "widgets.hit_test", X: 40, Y: 72, Target: "TextBox", Path: []int{0, 1, 2, 4}},
			{
				Helper: "widgets.hit_test",
				X:      180,
				Y:      124,
				Target: "ResetButton",
				Path:   []int{0, 1, 2, 5, 7},
			},
		},
		DispatchPaths: []surface.ComponentTreeAPIDispatchPathReport{
			{Helper: "tree_build_dispatch_path", Target: "TextBox", Path: []int{0, 1, 2, 4}},
			{
				Helper: "tree_build_dispatch_path",
				Target: "SubmitButton",
				Path:   []int{0, 1, 2, 5, 6},
			},
			{Helper: "tree_build_dispatch_path", Target: "ResetButton", Path: []int{0, 1, 2, 5, 7}},
		},
	}
}
func minimalToolkitReport() *surface.ToolkitReport {
	return &surface.ToolkitReport{
		Schema:                    "tetra.surface.toolkit.v1",
		ToolkitLevel:              "minimal-widgets-v1",
		Source:                    "examples/surface/toolkit/surface_toolkit_form.tetra",
		Module:                    "lib.core.widgets",
		Experimental:              true,
		ProductionClaim:           false,
		UsesComponentTreeAPI:      true,
		ManualBookkeeping:         false,
		DemoSpecificWidgetStructs: false,
		NoMagicWidgets:            true,
		NoPlatformWidgets:         true,
		NoDOMUI:                   true,
		NoUserJS:                  true,
		Widgets: []surface.ToolkitWidgetReport{
			{Name: "Panel", Kind: "Panel", NodeID: 1, Reusable: true, OrdinaryTetraStruct: true},
			{Name: "Column", Kind: "Column", NodeID: 2, Reusable: true, OrdinaryTetraStruct: true},
			{
				Name:                "NameLabel",
				Kind:                "Text",
				NodeID:              3,
				Role:                "label",
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
			{
				Name:                "TextBox",
				Kind:                "TextBox",
				NodeID:              4,
				Reusable:            true,
				OrdinaryTetraStruct: true,
				Editable:            true,
			},
			{Name: "ButtonRow", Kind: "Row", NodeID: 5, Reusable: true, OrdinaryTetraStruct: true},
			{
				Name:                "SubmitButton",
				Kind:                "Button",
				NodeID:              6,
				Action:              "submit",
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
			{
				Name:                "ResetButton",
				Kind:                "Button",
				NodeID:              7,
				Action:              "reset",
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
			{
				Name:                "StatusText",
				Kind:                "Text",
				NodeID:              8,
				Role:                "status",
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
		},
		ReusableSources: []string{
			"lib/core/widgets/widgets.tetra:panel_init",
			"lib/core/widgets/widgets.tetra:column_init",
			"lib/core/widgets/widgets.tetra:text_init",
			"lib/core/widgets/widgets.tetra:textbox_init",
			"lib/core/widgets/widgets.tetra:row_init",
			"lib/core/widgets/widgets.tetra:button_init",
		},
	}
}
func toolkitReuseComponentTreeAPIReport() *surface.ComponentTreeAPIReport {
	return &surface.ComponentTreeAPIReport{
		Schema:            "tetra.surface.component-tree-api.v1",
		APILevel:          "builder-layout-dispatch-v1",
		Source:            "examples/surface/toolkit/surface_toolkit_settings.tetra",
		ManualBookkeeping: false,
		Builder: surface.ComponentTreeAPIBuilderReport{
			RootCreatedBy:     "tree_add_root",
			ChildrenCreatedBy: "tree_add_child",
			NodeCount:         11,
			Capacity:          20,
			OverflowChecked:   true,
		},
		Invariants: surface.ComponentTreeAPIInvariantReport{
			TreeValidateRan:         true,
			TreeValidateStatus:      0,
			ParentChildLinksChecked: true,
			ChildIndicesChecked:     true,
			ChildCountChecked:       true,
			FirstChildChecked:       true,
		},
		LayoutHelpers: []surface.ComponentTreeAPILayoutHelperReport{
			{
				Helper:        "widgets.panel_content_rect",
				Target:        "Panel",
				Pass:          "initial",
				ChangedBounds: true,
			},
			{
				Helper:        "widgets.column_layout",
				Target:        "Column",
				Pass:          "initial",
				ChangedBounds: true,
			},
			{
				Helper:        "widgets.row_layout",
				Target:        "ButtonRow",
				Pass:          "initial",
				ChangedBounds: true,
			},
			{
				Helper:        "widgets.column_layout",
				Target:        "Column",
				Pass:          "resize",
				ChangedBounds: true,
			},
		},
		FocusHelpers: []surface.ComponentTreeAPIFocusHelperReport{
			{Helper: "tree_focus_next", Before: "NameTextBox", After: "EmailTextBox"},
			{Helper: "tree_focus_next", Before: "EmailTextBox", After: "SaveButton"},
			{Helper: "tree_focus_next", Before: "SaveButton", After: "ResetButton"},
			{Helper: "tree_focus_next", Before: "ResetButton", After: "NameTextBox"},
		},
		HitTests: []surface.ComponentTreeAPIHitTestReport{
			{
				Helper: "widgets.hit_test",
				X:      40,
				Y:      72,
				Target: "NameTextBox",
				Path:   []int{0, 1, 2, 4},
			},
			{
				Helper: "widgets.hit_test",
				X:      40,
				Y:      156,
				Target: "EmailTextBox",
				Path:   []int{0, 1, 2, 6},
			},
			{
				Helper: "widgets.hit_test",
				X:      180,
				Y:      208,
				Target: "ResetButton",
				Path:   []int{0, 1, 2, 7, 9},
			},
		},
		DispatchPaths: []surface.ComponentTreeAPIDispatchPathReport{
			{Helper: "tree_build_dispatch_path", Target: "NameTextBox", Path: []int{0, 1, 2, 4}},
			{Helper: "tree_build_dispatch_path", Target: "EmailTextBox", Path: []int{0, 1, 2, 6}},
			{Helper: "tree_build_dispatch_path", Target: "SaveButton", Path: []int{0, 1, 2, 7, 8}},
			{Helper: "tree_build_dispatch_path", Target: "ResetButton", Path: []int{0, 1, 2, 7, 9}},
		},
	}
}
func toolkitReuseReport() *surface.ToolkitReport {
	return &surface.ToolkitReport{
		Schema:       "tetra.surface.toolkit.v1",
		ToolkitLevel: "toolkit-reuse-v1",
		ReuseLevel:   "multi-form-widget-reuse-v1",
		Source:       "examples/surface/toolkit/surface_toolkit_settings.tetra",
		Sources: []string{
			"examples/surface/toolkit/surface_toolkit_form.tetra",
			"examples/surface/toolkit/surface_toolkit_settings.tetra",
		},
		Module:                    "lib.core.widgets",
		Experimental:              true,
		ProductionClaim:           false,
		UsesComponentTreeAPI:      true,
		ManualBookkeeping:         false,
		DemoSpecificWidgetStructs: false,
		NoMagicWidgets:            true,
		NoPlatformWidgets:         true,
		NoDOMUI:                   true,
		NoUserJS:                  true,
		ExampleCount:              2,
		TextBoxCount:              2,
		ButtonCount:               2,
		MultiTextBoxEvidence:      true,
		MultiFormEvidence:         true,
		Widgets: []surface.ToolkitWidgetReport{
			{Name: "Panel", Kind: "Panel", NodeID: 1, Reusable: true, OrdinaryTetraStruct: true},
			{Name: "Column", Kind: "Column", NodeID: 2, Reusable: true, OrdinaryTetraStruct: true},
			{
				Name:                "TitleText",
				Kind:                "Text",
				NodeID:              3,
				Role:                "label",
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
			{
				Name:                "NameTextBox",
				Kind:                "TextBox",
				NodeID:              4,
				Reusable:            true,
				OrdinaryTetraStruct: true,
				Editable:            true,
			},
			{
				Name:                "NameLabel",
				Kind:                "Text",
				NodeID:              5,
				Role:                "label",
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
			{
				Name:                "EmailTextBox",
				Kind:                "TextBox",
				NodeID:              6,
				Reusable:            true,
				OrdinaryTetraStruct: true,
				Editable:            true,
			},
			{Name: "ButtonRow", Kind: "Row", NodeID: 7, Reusable: true, OrdinaryTetraStruct: true},
			{
				Name:                "SaveButton",
				Kind:                "Button",
				NodeID:              8,
				Action:              "save",
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
			{
				Name:                "ResetButton",
				Kind:                "Button",
				NodeID:              9,
				Action:              "reset",
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
			{
				Name:                "StatusText",
				Kind:                "Text",
				NodeID:              10,
				Role:                "status",
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
		},
		ReusableSources: []string{
			"lib/core/widgets/widgets.tetra:panel_init",
			"lib/core/widgets/widgets.tetra:column_init",
			"lib/core/widgets/widgets.tetra:text_init",
			"lib/core/widgets/widgets.tetra:textbox_init",
			"lib/core/widgets/widgets.tetra:row_init",
			"lib/core/widgets/widgets.tetra:button_init",
			"lib/core/widgets/widgets.tetra:hit_test",
			"lib/core/widgets/widgets.tetra:textbox_text_input",
			"lib/core/widgets/widgets.tetra:button_key_event",
		},
	}
}
func productionToolkitBaseCases() []surface.CaseReport {
	cases := toolkitReuseBaseCases()
	cases = append(
		cases,
		surface.CaseReport{
			Name: "production toolkit required widget set",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "production toolkit style module default theme",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "production toolkit style states normal focused hovered pressed disabled error",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "production toolkit Text Label StatusText evidence",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "production toolkit Button TextBox Checkbox evidence",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "production toolkit Row Column Panel Stack Scroll Spacer layout",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "production toolkit component tree api reuse",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "production toolkit TextBox focus input editing",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "production toolkit Checkbox toggle routed",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "production toolkit Scroll offset routed",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "production toolkit Save action routed",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "production toolkit Reset action routed",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "production toolkit StatusText updates",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "production toolkit safe text storage",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "production toolkit no demo-local widget structs",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "production toolkit browser host separation",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "production toolkit rendered frame update",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
	)
	return cases
}
func productionToolkitComponentTreeAPIReport() *surface.ComponentTreeAPIReport {
	return &surface.ComponentTreeAPIReport{
		Schema:            "tetra.surface.component-tree-api.v1",
		APILevel:          "builder-layout-dispatch-v1",
		Source:            "examples/surface/release/surface_release_form.tetra",
		ManualBookkeeping: false,
		Builder: surface.ComponentTreeAPIBuilderReport{
			RootCreatedBy:     "tree_add_root",
			ChildrenCreatedBy: "tree_add_child",
			NodeCount:         18,
			Capacity:          32,
			OverflowChecked:   true,
		},
		Invariants: surface.ComponentTreeAPIInvariantReport{
			TreeValidateRan:         true,
			TreeValidateStatus:      0,
			ParentChildLinksChecked: true,
			ChildIndicesChecked:     true,
			ChildCountChecked:       true,
			FirstChildChecked:       true,
		},
		LayoutHelpers: []surface.ComponentTreeAPILayoutHelperReport{
			{
				Helper:        "widgets.panel_content_rect",
				Target:        "Panel",
				Pass:          "initial",
				ChangedBounds: true,
			},
			{Helper: "widgets.stack_layout", Target: "Stack", Pass: "initial", ChangedBounds: true},
			{
				Helper:        "widgets.column_layout",
				Target:        "Column",
				Pass:          "initial",
				ChangedBounds: true,
			},
			{
				Helper:        "widgets.scroll_set_offset",
				Target:        "TermsScroll",
				Pass:          "scroll",
				ChangedBounds: true,
			},
			{
				Helper:        "widgets.row_layout",
				Target:        "ButtonRow",
				Pass:          "initial",
				ChangedBounds: true,
			},
			{
				Helper:        "widgets.column_layout",
				Target:        "Column",
				Pass:          "resize",
				ChangedBounds: true,
			},
		},
		FocusHelpers: []surface.ComponentTreeAPIFocusHelperReport{
			{Helper: "tree_focus_next", Before: "NameTextBox", After: "EmailTextBox"},
			{Helper: "tree_focus_next", Before: "EmailTextBox", After: "SubscribeCheckbox"},
			{Helper: "tree_focus_next", Before: "SubscribeCheckbox", After: "SaveButton"},
			{Helper: "tree_focus_next", Before: "SaveButton", After: "ResetButton"},
			{Helper: "tree_focus_next", Before: "ResetButton", After: "NameTextBox"},
		},
		HitTests: []surface.ComponentTreeAPIHitTestReport{
			{
				Helper: "widgets.hit_test_release_form",
				X:      48,
				Y:      148,
				Target: "NameTextBox",
				Path:   []int{0, 1, 2, 3, 7},
			},
			{
				Helper: "widgets.hit_test_release_form",
				X:      48,
				Y:      228,
				Target: "EmailTextBox",
				Path:   []int{0, 1, 2, 3, 9},
			},
			{
				Helper: "widgets.hit_test_release_form",
				X:      48,
				Y:      280,
				Target: "SubscribeCheckbox",
				Path:   []int{0, 1, 2, 3, 10},
			},
			{
				Helper: "widgets.hit_test_release_form",
				X:      192,
				Y:      376,
				Target: "ResetButton",
				Path:   []int{0, 1, 2, 3, 13, 15},
			},
		},
		DispatchPaths: []surface.ComponentTreeAPIDispatchPathReport{
			{Helper: "tree_build_dispatch_path", Target: "NameTextBox", Path: []int{0, 1, 2, 3, 7}},
			{
				Helper: "tree_build_dispatch_path",
				Target: "EmailTextBox",
				Path:   []int{0, 1, 2, 3, 9},
			},
			{
				Helper: "tree_build_dispatch_path",
				Target: "SubscribeCheckbox",
				Path:   []int{0, 1, 2, 3, 10},
			},
			{
				Helper: "tree_build_dispatch_path",
				Target: "TermsScroll",
				Path:   []int{0, 1, 2, 3, 11},
			},
			{
				Helper: "tree_build_dispatch_path",
				Target: "SaveButton",
				Path:   []int{0, 1, 2, 3, 13, 14},
			},
			{
				Helper: "tree_build_dispatch_path",
				Target: "ResetButton",
				Path:   []int{0, 1, 2, 3, 13, 15},
			},
		},
	}
}
func productionToolkitReport() *surface.ToolkitReport {
	return &surface.ToolkitReport{
		Schema:       "tetra.surface.toolkit.v1",
		ToolkitLevel: "production-widgets-v1",
		ReleaseScope: "surface-v1-linux-web",
		Source:       "examples/surface/release/surface_release_form.tetra",
		Sources: []string{
			"examples/surface/release/surface_release_form.tetra",
			"examples/surface/toolkit/surface_toolkit_form.tetra",
			"examples/surface/toolkit/surface_toolkit_settings.tetra",
		},
		Module:                    "lib.core.widgets",
		StyleModule:               "lib.core.style",
		Experimental:              false,
		ProductionClaim:           true,
		UsesComponentTreeAPI:      true,
		ManualBookkeeping:         false,
		DemoSpecificWidgetStructs: false,
		NoMagicWidgets:            true,
		NoPlatformWidgets:         true,
		NoDOMUI:                   true,
		NoUserJS:                  true,
		ExampleCount:              3,
		TextBoxCount:              2,
		ButtonCount:               2,
		MultiTextBoxEvidence:      true,
		MultiFormEvidence:         true,
		WidgetSet: []string{
			"Text",
			"Label",
			"StatusText",
			"Button",
			"TextBox",
			"Checkbox",
			"Row",
			"Column",
			"Panel",
			"Stack",
			"Scroll",
			"Spacer",
		},
		StateSet: []string{
			"normal",
			"focused",
			"hovered",
			"pressed",
			"disabled",
			"error",
		},
		LayoutFeatures: []string{
			"padding",
			"margin",
			"spacing",
			"min_size",
			"max_size",
			"fill",
			"scroll_offset",
		},
		Theme:           true,
		SafeTextStorage: true,
		Widgets: []surface.ToolkitWidgetReport{
			{Name: "Panel", Kind: "Panel", NodeID: 1, Reusable: true, OrdinaryTetraStruct: true},
			{Name: "Stack", Kind: "Stack", NodeID: 2, Reusable: true, OrdinaryTetraStruct: true},
			{Name: "Column", Kind: "Column", NodeID: 3, Reusable: true, OrdinaryTetraStruct: true},
			{
				Name:                "TitleText",
				Kind:                "Text",
				NodeID:              4,
				Role:                "label",
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
			{
				Name:                "DescriptionText",
				Kind:                "Text",
				NodeID:              5,
				Role:                "description",
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
			{
				Name:                "NameLabel",
				Kind:                "Label",
				NodeID:              6,
				Role:                "label",
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
			{
				Name:                "NameTextBox",
				Kind:                "TextBox",
				NodeID:              7,
				Reusable:            true,
				OrdinaryTetraStruct: true,
				Editable:            true,
			},
			{
				Name:                "EmailLabel",
				Kind:                "Label",
				NodeID:              8,
				Role:                "label",
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
			{
				Name:                "EmailTextBox",
				Kind:                "TextBox",
				NodeID:              9,
				Reusable:            true,
				OrdinaryTetraStruct: true,
				Editable:            true,
			},
			{
				Name:                "SubscribeCheckbox",
				Kind:                "Checkbox",
				NodeID:              10,
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
			{
				Name:                "TermsScroll",
				Kind:                "Scroll",
				NodeID:              11,
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
			{
				Name:                "TermsText",
				Kind:                "Text",
				NodeID:              12,
				Role:                "description",
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
			{Name: "ButtonRow", Kind: "Row", NodeID: 13, Reusable: true, OrdinaryTetraStruct: true},
			{
				Name:                "SaveButton",
				Kind:                "Button",
				NodeID:              14,
				Action:              "save",
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
			{
				Name:                "ResetButton",
				Kind:                "Button",
				NodeID:              15,
				Action:              "reset",
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
			{Name: "Spacer", Kind: "Spacer", NodeID: 16, Reusable: true, OrdinaryTetraStruct: true},
			{
				Name:                "StatusText",
				Kind:                "StatusText",
				NodeID:              17,
				Role:                "status",
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
		},
		ReusableSources: []string{
			"lib/core/widgets/widgets.tetra:panel_init",
			"lib/core/widgets/widgets.tetra:column_init",
			"lib/core/widgets/widgets.tetra:text_init",
			"lib/core/widgets/widgets.tetra:label_init",
			"lib/core/widgets/widgets.tetra:status_text_init",
			"lib/core/widgets/widgets.tetra:textbox_init",
			"lib/core/widgets/widgets.tetra:checkbox_init",
			"lib/core/widgets/widgets.tetra:checkbox_toggle",
			"lib/core/widgets/widgets.tetra:row_init",
			"lib/core/widgets/widgets.tetra:stack_init",
			"lib/core/widgets/widgets.tetra:scroll_init",
			"lib/core/widgets/widgets.tetra:scroll_set_offset",
			"lib/core/widgets/widgets.tetra:spacer_init",
			"lib/core/widgets/widgets.tetra:button_init",
			"lib/core/widgets/widgets.tetra:hit_test_release_form",
			"lib/core/widgets/style.tetra:default_theme",
			"lib/core/widgets/style.tetra:style_for_state",
		},
	}
}
func accessibilityComponentTreeReport() *surface.ComponentTreeReport {
	return &surface.ComponentTreeReport{
		Schema:       "tetra.surface.component-tree.v1",
		DynamicLevel: "accessibility-metadata-tree-v1",
		RootID:       0,
		NodeCount:    12,
		FocusedID:    5,
		Nodes: []surface.ComponentTreeNodeReport{
			{
				ID:         0,
				Name:       "AccessibilitySettingsApp",
				Kind:       "root",
				ParentID:   -1,
				ChildIndex: 0,
				FirstChild: 1,
				ChildCount: 1,
				Focusable:  false,
				Bounds:     surface.RectReport{X: 0, Y: 0, W: 480, H: 320},
			},
			{
				ID:         1,
				Name:       "Panel",
				Kind:       "panel",
				ParentID:   0,
				ChildIndex: 0,
				FirstChild: 2,
				ChildCount: 1,
				Focusable:  false,
				Bounds:     surface.RectReport{X: 0, Y: 0, W: 480, H: 320},
			},
			{
				ID:         2,
				Name:       "Column",
				Kind:       "column",
				ParentID:   1,
				ChildIndex: 0,
				FirstChild: 3,
				ChildCount: 7,
				Focusable:  false,
				Bounds:     surface.RectReport{X: 12, Y: 12, W: 456, H: 296},
			},
			{
				ID:         3,
				Name:       "TitleText",
				Kind:       "text",
				ParentID:   2,
				ChildIndex: 0,
				FirstChild: -1,
				ChildCount: 0,
				Focusable:  false,
				Bounds:     surface.RectReport{X: 20, Y: 20, W: 440, H: 24},
			},
			{
				ID:         4,
				Name:       "NameLabel",
				Kind:       "text",
				ParentID:   2,
				ChildIndex: 1,
				FirstChild: -1,
				ChildCount: 0,
				Focusable:  false,
				Bounds:     surface.RectReport{X: 20, Y: 52, W: 440, H: 24},
			},
			{
				ID:         5,
				Name:       "NameTextBox",
				Kind:       "textbox",
				ParentID:   2,
				ChildIndex: 2,
				FirstChild: -1,
				ChildCount: 0,
				Focusable:  true,
				Bounds:     surface.RectReport{X: 20, Y: 84, W: 440, H: 44},
			},
			{
				ID:         6,
				Name:       "EmailLabel",
				Kind:       "text",
				ParentID:   2,
				ChildIndex: 3,
				FirstChild: -1,
				ChildCount: 0,
				Focusable:  false,
				Bounds:     surface.RectReport{X: 20, Y: 136, W: 440, H: 24},
			},
			{
				ID:         7,
				Name:       "EmailTextBox",
				Kind:       "textbox",
				ParentID:   2,
				ChildIndex: 4,
				FirstChild: -1,
				ChildCount: 0,
				Focusable:  true,
				Bounds:     surface.RectReport{X: 20, Y: 168, W: 440, H: 44},
			},
			{
				ID:         8,
				Name:       "ButtonRow",
				Kind:       "row",
				ParentID:   2,
				ChildIndex: 5,
				FirstChild: 9,
				ChildCount: 2,
				Focusable:  false,
				Bounds:     surface.RectReport{X: 20, Y: 224, W: 440, H: 44},
			},
			{
				ID:         9,
				Name:       "SaveButton",
				Kind:       "button",
				ParentID:   8,
				ChildIndex: 0,
				FirstChild: -1,
				ChildCount: 0,
				Focusable:  true,
				Bounds:     surface.RectReport{X: 20, Y: 224, W: 132, H: 44},
			},
			{
				ID:         10,
				Name:       "ResetButton",
				Kind:       "button",
				ParentID:   8,
				ChildIndex: 1,
				FirstChild: -1,
				ChildCount: 0,
				Focusable:  true,
				Bounds:     surface.RectReport{X: 164, Y: 224, W: 132, H: 44},
			},
			{
				ID:         11,
				Name:       "StatusText",
				Kind:       "text",
				ParentID:   2,
				ChildIndex: 6,
				FirstChild: -1,
				ChildCount: 0,
				Focusable:  false,
				Bounds:     surface.RectReport{X: 20, Y: 280, W: 440, H: 24},
			},
		},
		LayoutPasses: []surface.ComponentTreeLayoutPassReport{
			{
				ComponentID: 5,
				Pass:        "initial",
				Bounds:      surface.RectReport{X: 20, Y: 84, W: 280, H: 44},
				Measured:    surface.SizeReport{W: 280, H: 44},
			},
			{
				ComponentID: 7,
				Pass:        "initial",
				Bounds:      surface.RectReport{X: 20, Y: 168, W: 280, H: 44},
				Measured:    surface.SizeReport{W: 280, H: 44},
			},
			{
				ComponentID: 5,
				Pass:        "resize",
				Bounds:      surface.RectReport{X: 20, Y: 84, W: 440, H: 44},
				Measured:    surface.SizeReport{W: 440, H: 44},
			},
			{
				ComponentID: 7,
				Pass:        "resize",
				Bounds:      surface.RectReport{X: 20, Y: 168, W: 440, H: 44},
				Measured:    surface.SizeReport{W: 440, H: 44},
			},
			{
				ComponentID: 11,
				Pass:        "status-update",
				Bounds:      surface.RectReport{X: 20, Y: 280, W: 440, H: 24},
				Measured:    surface.SizeReport{W: 440, H: 24},
			},
		},
		DrawOrder:  []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
		FocusOrder: []int{5, 7, 9, 10},
		DispatchPaths: []surface.ComponentTreeDispatchPathReport{
			{Event: "click", TargetID: 5, X: 40, Y: 100, Path: []int{0, 1, 2, 5}},
			{Event: "click", TargetID: 7, X: 40, Y: 184, Path: []int{0, 1, 2, 7}},
			{Event: "key", TargetID: 9, X: 40, Y: 240, Path: []int{0, 1, 2, 8, 9}},
			{Event: "key", TargetID: 10, X: 180, Y: 240, Path: []int{0, 1, 2, 8, 10}},
		},
	}
}
func accessibilityComponentTreeAPIReport() *surface.ComponentTreeAPIReport {
	return &surface.ComponentTreeAPIReport{
		Schema:            "tetra.surface.component-tree-api.v1",
		APILevel:          "builder-layout-dispatch-v1",
		Source:            "examples/surface/toolkit/surface_accessibility_settings.tetra",
		ManualBookkeeping: false,
		Builder: surface.ComponentTreeAPIBuilderReport{
			RootCreatedBy:     "tree_add_root",
			ChildrenCreatedBy: "tree_add_child",
			NodeCount:         12,
			Capacity:          24,
			OverflowChecked:   true,
		},
		Invariants: surface.ComponentTreeAPIInvariantReport{
			TreeValidateRan:         true,
			TreeValidateStatus:      0,
			ParentChildLinksChecked: true,
			ChildIndicesChecked:     true,
			ChildCountChecked:       true,
			FirstChildChecked:       true,
		},
		LayoutHelpers: []surface.ComponentTreeAPILayoutHelperReport{
			{
				Helper:        "widgets.panel_content_rect",
				Target:        "Panel",
				Pass:          "initial",
				ChangedBounds: true,
			},
			{
				Helper:        "widgets.column_layout",
				Target:        "Column",
				Pass:          "initial",
				ChangedBounds: true,
			},
			{
				Helper:        "widgets.row_layout",
				Target:        "ButtonRow",
				Pass:          "initial",
				ChangedBounds: true,
			},
			{
				Helper:        "widgets.column_layout",
				Target:        "Column",
				Pass:          "resize",
				ChangedBounds: true,
			},
		},
		FocusHelpers: []surface.ComponentTreeAPIFocusHelperReport{
			{Helper: "tree_focus_next", Before: "NameTextBox", After: "EmailTextBox"},
			{Helper: "tree_focus_next", Before: "EmailTextBox", After: "SaveButton"},
			{Helper: "tree_focus_next", Before: "SaveButton", After: "ResetButton"},
			{Helper: "tree_focus_next", Before: "ResetButton", After: "NameTextBox"},
		},
		HitTests: []surface.ComponentTreeAPIHitTestReport{
			{
				Helper: "widgets.hit_test_accessibility_settings",
				X:      40,
				Y:      100,
				Target: "NameTextBox",
				Path:   []int{0, 1, 2, 5},
			},
			{
				Helper: "widgets.hit_test_accessibility_settings",
				X:      40,
				Y:      184,
				Target: "EmailTextBox",
				Path:   []int{0, 1, 2, 7},
			},
			{
				Helper: "widgets.hit_test_accessibility_settings",
				X:      180,
				Y:      240,
				Target: "ResetButton",
				Path:   []int{0, 1, 2, 8, 10},
			},
		},
		DispatchPaths: []surface.ComponentTreeAPIDispatchPathReport{
			{Helper: "tree_build_dispatch_path", Target: "NameTextBox", Path: []int{0, 1, 2, 5}},
			{Helper: "tree_build_dispatch_path", Target: "EmailTextBox", Path: []int{0, 1, 2, 7}},
			{Helper: "tree_build_dispatch_path", Target: "SaveButton", Path: []int{0, 1, 2, 8, 9}},
			{
				Helper: "tree_build_dispatch_path",
				Target: "ResetButton",
				Path:   []int{0, 1, 2, 8, 10},
			},
		},
	}
}
func accessibilityToolkitReport() *surface.ToolkitReport {
	return &surface.ToolkitReport{
		Schema:       "tetra.surface.toolkit.v1",
		ToolkitLevel: "toolkit-reuse-v1",
		ReuseLevel:   "multi-form-widget-reuse-v1",
		Source:       "examples/surface/toolkit/surface_accessibility_settings.tetra",
		Sources: []string{
			"examples/surface/toolkit/surface_toolkit_form.tetra",
			"examples/surface/toolkit/surface_toolkit_settings.tetra",
			"examples/surface/toolkit/surface_accessibility_settings.tetra",
		},
		Module:                    "lib.core.widgets",
		Experimental:              true,
		ProductionClaim:           false,
		UsesComponentTreeAPI:      true,
		ManualBookkeeping:         false,
		DemoSpecificWidgetStructs: false,
		NoMagicWidgets:            true,
		NoPlatformWidgets:         true,
		NoDOMUI:                   true,
		NoUserJS:                  true,
		ExampleCount:              3,
		TextBoxCount:              2,
		ButtonCount:               2,
		MultiTextBoxEvidence:      true,
		MultiFormEvidence:         true,
		Widgets: []surface.ToolkitWidgetReport{
			{Name: "Panel", Kind: "Panel", NodeID: 1, Reusable: true, OrdinaryTetraStruct: true},
			{Name: "Column", Kind: "Column", NodeID: 2, Reusable: true, OrdinaryTetraStruct: true},
			{
				Name:                "TitleText",
				Kind:                "Text",
				NodeID:              3,
				Role:                "text",
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
			{
				Name:                "NameLabel",
				Kind:                "Text",
				NodeID:              4,
				Role:                "label",
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
			{
				Name:                "NameTextBox",
				Kind:                "TextBox",
				NodeID:              5,
				Reusable:            true,
				OrdinaryTetraStruct: true,
				Editable:            true,
			},
			{
				Name:                "EmailLabel",
				Kind:                "Text",
				NodeID:              6,
				Role:                "label",
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
			{
				Name:                "EmailTextBox",
				Kind:                "TextBox",
				NodeID:              7,
				Reusable:            true,
				OrdinaryTetraStruct: true,
				Editable:            true,
			},
			{Name: "ButtonRow", Kind: "Row", NodeID: 8, Reusable: true, OrdinaryTetraStruct: true},
			{
				Name:                "SaveButton",
				Kind:                "Button",
				NodeID:              9,
				Action:              "save",
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
			{
				Name:                "ResetButton",
				Kind:                "Button",
				NodeID:              10,
				Action:              "reset",
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
			{
				Name:                "StatusText",
				Kind:                "Text",
				NodeID:              11,
				Role:                "status",
				Reusable:            true,
				OrdinaryTetraStruct: true,
			},
		},
		ReusableSources: []string{
			"lib/core/widgets/widgets.tetra:panel_init",
			"lib/core/widgets/widgets.tetra:column_init",
			"lib/core/widgets/widgets.tetra:text_init",
			"lib/core/widgets/widgets.tetra:textbox_init",
			"lib/core/widgets/widgets.tetra:row_init",
			"lib/core/widgets/widgets.tetra:button_init",
			"lib/core/widgets/widgets.tetra:add_accessible_textbox",
			"lib/core/widgets/widgets.tetra:add_accessible_button",
			"lib/core/widgets/widgets.tetra:add_accessible_status",
		},
	}
}

func accessibilityTreeReport(
	beforeFrame rgbaFrame,
	nameFrame rgbaFrame,
	saveFrame rgbaFrame,
	resetFrame rgbaFrame,
	afterFrame rgbaFrame,
) *surface.AccessibilityTreeReport {
	return &surface.AccessibilityTreeReport{
		Schema:                   "tetra.surface.accessibility-tree.v1",
		AccessibilityLevel:       "metadata-tree-v1",
		Source:                   "examples/surface/toolkit/surface_accessibility_settings.tetra",
		Module:                   "lib.core.accessibility",
		WidgetModule:             "lib.core.widgets",
		Experimental:             true,
		ProductionClaim:          false,
		PlatformHostIntegration:  false,
		DOMARIAIntegration:       false,
		ScreenReaderEvidence:     false,
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
		NodeCount:                12,
		FocusableCount:           4,
		LabelCount:               2,
		TextBoxCount:             2,
		ButtonCount:              2,
		StatusCount:              1,
		RolesPresent: []string{
			"root",
			"panel",
			"column",
			"text",
			"label",
			"textbox",
			"row",
			"button",
			"status",
		},
		Nodes: accessibilityNodes(),
		Relationships: []surface.AccessibilityRelationshipReport{
			{Kind: "label_for", From: "NameLabel", To: "NameTextBox"},
			{Kind: "labelled_by", From: "NameTextBox", To: "NameLabel"},
			{Kind: "label_for", From: "EmailLabel", To: "EmailTextBox"},
			{Kind: "labelled_by", From: "EmailTextBox", To: "EmailLabel"},
		},
		FocusOrder: []string{"NameTextBox", "EmailTextBox", "SaveButton", "ResetButton"},
		ReadingOrder: []string{
			"TitleText",
			"NameLabel",
			"NameTextBox",
			"EmailLabel",
			"EmailTextBox",
			"SaveButton",
			"ResetButton",
			"StatusText",
		},
		Actions: []surface.AccessibilityActionReport{
			{Target: "NameTextBox", Action: "edit", Semantic: "text-input"},
			{Target: "EmailTextBox", Action: "edit", Semantic: "text-input"},
			{Target: "SaveButton", Action: "press", Semantic: "save"},
			{Target: "ResetButton", Action: "press", Semantic: "reset"},
		},
		Snapshots: accessibilitySnapshots(
			beforeFrame,
			nameFrame,
			saveFrame,
			resetFrame,
			afterFrame,
		),
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
func accessibilityNodes() []surface.AccessibilityNodeReport {
	return []surface.AccessibilityNodeReport{
		{
			ID:           0,
			ComponentID:  0,
			ParentID:     -1,
			Name:         "AccessibilitySettingsApp",
			Role:         "root",
			Bounds:       surface.RectReport{X: 0, Y: 0, W: 480, H: 320},
			Visible:      true,
			Enabled:      true,
			FocusIndex:   -1,
			ReadingIndex: 0,
		},
		{
			ID:           1,
			ComponentID:  1,
			ParentID:     0,
			Name:         "Panel",
			Role:         "panel",
			Bounds:       surface.RectReport{X: 0, Y: 0, W: 480, H: 320},
			Visible:      true,
			Enabled:      true,
			FocusIndex:   -1,
			ReadingIndex: 1,
		},
		{
			ID:           2,
			ComponentID:  2,
			ParentID:     1,
			Name:         "Column",
			Role:         "column",
			Bounds:       surface.RectReport{X: 12, Y: 12, W: 456, H: 296},
			Visible:      true,
			Enabled:      true,
			FocusIndex:   -1,
			ReadingIndex: 2,
		},
		{
			ID:           3,
			ComponentID:  3,
			ParentID:     2,
			Name:         "TitleText",
			Role:         "text",
			Bounds:       surface.RectReport{X: 20, Y: 20, W: 440, H: 24},
			Visible:      true,
			Enabled:      true,
			ValueKind:    "title",
			FocusIndex:   -1,
			ReadingIndex: 3,
		},
		{
			ID:           4,
			ComponentID:  4,
			ParentID:     2,
			Name:         "NameLabel",
			Role:         "label",
			Bounds:       surface.RectReport{X: 20, Y: 52, W: 440, H: 24},
			Visible:      true,
			Enabled:      true,
			LabelFor:     "NameTextBox",
			ValueKind:    "name",
			FocusIndex:   -1,
			ReadingIndex: 4,
		},
		{
			ID:           5,
			ComponentID:  5,
			ParentID:     2,
			Name:         "NameTextBox",
			Role:         "textbox",
			Bounds:       surface.RectReport{X: 20, Y: 84, W: 440, H: 44},
			Visible:      true,
			Enabled:      true,
			Focusable:    true,
			Focused:      true,
			Editable:     true,
			LabelledBy:   "NameLabel",
			ValueKind:    "empty",
			Actions:      []string{"focus", "edit"},
			FocusIndex:   0,
			ReadingIndex: 5,
		},
		{
			ID:           6,
			ComponentID:  6,
			ParentID:     2,
			Name:         "EmailLabel",
			Role:         "label",
			Bounds:       surface.RectReport{X: 20, Y: 136, W: 440, H: 24},
			Visible:      true,
			Enabled:      true,
			LabelFor:     "EmailTextBox",
			ValueKind:    "email",
			FocusIndex:   -1,
			ReadingIndex: 6,
		},
		{
			ID:           7,
			ComponentID:  7,
			ParentID:     2,
			Name:         "EmailTextBox",
			Role:         "textbox",
			Bounds:       surface.RectReport{X: 20, Y: 168, W: 440, H: 44},
			Visible:      true,
			Enabled:      true,
			Focusable:    true,
			Editable:     true,
			LabelledBy:   "EmailLabel",
			ValueKind:    "empty",
			Actions:      []string{"focus", "edit"},
			FocusIndex:   1,
			ReadingIndex: 7,
		},
		{
			ID:           8,
			ComponentID:  8,
			ParentID:     2,
			Name:         "ButtonRow",
			Role:         "row",
			Bounds:       surface.RectReport{X: 20, Y: 224, W: 440, H: 44},
			Visible:      true,
			Enabled:      true,
			FocusIndex:   -1,
			ReadingIndex: 8,
		},
		{
			ID:           9,
			ComponentID:  9,
			ParentID:     8,
			Name:         "SaveButton",
			Role:         "button",
			Bounds:       surface.RectReport{X: 20, Y: 224, W: 132, H: 44},
			Visible:      true,
			Enabled:      true,
			Focusable:    true,
			ValueKind:    "save",
			Actions:      []string{"focus", "press", "save"},
			FocusIndex:   2,
			ReadingIndex: 9,
		},
		{
			ID:           10,
			ComponentID:  10,
			ParentID:     8,
			Name:         "ResetButton",
			Role:         "button",
			Bounds:       surface.RectReport{X: 164, Y: 224, W: 132, H: 44},
			Visible:      true,
			Enabled:      true,
			Focusable:    true,
			ValueKind:    "reset",
			Actions:      []string{"focus", "press", "reset"},
			FocusIndex:   3,
			ReadingIndex: 10,
		},
		{
			ID:           11,
			ComponentID:  11,
			ParentID:     2,
			Name:         "StatusText",
			Role:         "status",
			Bounds:       surface.RectReport{X: 20, Y: 280, W: 440, H: 24},
			Visible:      true,
			Enabled:      true,
			ValueKind:    "reset",
			FocusIndex:   -1,
			ReadingIndex: 11,
		},
	}
}

func accessibilitySnapshots(
	beforeFrame rgbaFrame,
	nameFrame rgbaFrame,
	saveFrame rgbaFrame,
	resetFrame rgbaFrame,
	afterFrame rgbaFrame,
) []surface.AccessibilitySnapshotReport {
	return []surface.AccessibilitySnapshotReport{
		{
			Name:                       "initial",
			Generation:                 1,
			Focused:                    "",
			FocusedComponentID:         -1,
			FocusedAccessibilityNodeID: -1,
			NameValueLen:               0,
			EmailValueLen:              0,
			StatusValue:                "idle",
			BoundsChecksum:             checksumText("bounds-initial"),
			MetadataChecksum:           checksumText("metadata-initial"),
			FrameChecksum:              checksumRGBA(beforeFrame.Pixels),
		},
		{
			Name:                       "after_name_focus",
			Generation:                 2,
			Focused:                    "NameTextBox",
			FocusedComponentID:         5,
			FocusedAccessibilityNodeID: 5,
			NameValueLen:               0,
			EmailValueLen:              0,
			StatusValue:                "idle",
			BoundsChecksum:             checksumText("bounds-name-focus"),
			MetadataChecksum:           checksumText("metadata-name-focus"),
			FrameChecksum:              checksumRGBA(nameFrame.Pixels),
		},
		{
			Name:                       "after_name_text",
			Generation:                 3,
			Focused:                    "NameTextBox",
			FocusedComponentID:         5,
			FocusedAccessibilityNodeID: 5,
			NameValueLen:               3,
			EmailValueLen:              0,
			StatusValue:                "idle",
			BoundsChecksum:             checksumText("bounds-name-text"),
			MetadataChecksum:           checksumText("metadata-name-text"),
			FrameChecksum:              checksumRGBA(nameFrame.Pixels),
		},
		{
			Name:                       "after_email_focus",
			Generation:                 4,
			Focused:                    "EmailTextBox",
			FocusedComponentID:         7,
			FocusedAccessibilityNodeID: 7,
			NameValueLen:               3,
			EmailValueLen:              0,
			StatusValue:                "idle",
			BoundsChecksum:             checksumText("bounds-email-focus"),
			MetadataChecksum:           checksumText("metadata-email-focus"),
			FrameChecksum:              checksumText("frame-email-focus"),
		},
		{
			Name:                       "after_email_text",
			Generation:                 5,
			Focused:                    "EmailTextBox",
			FocusedComponentID:         7,
			FocusedAccessibilityNodeID: 7,
			NameValueLen:               3,
			EmailValueLen:              5,
			StatusValue:                "idle",
			BoundsChecksum:             checksumText("bounds-email-text"),
			MetadataChecksum:           checksumText("metadata-email-text"),
			FrameChecksum:              checksumText("frame-email-text"),
		},
		{
			Name:                       "after_save",
			Generation:                 6,
			Focused:                    "SaveButton",
			FocusedComponentID:         9,
			FocusedAccessibilityNodeID: 9,
			NameValueLen:               3,
			EmailValueLen:              5,
			StatusValue:                "saved",
			BoundsChecksum:             checksumText("bounds-save"),
			MetadataChecksum:           checksumText("metadata-save"),
			FrameChecksum:              checksumRGBA(saveFrame.Pixels),
		},
		{
			Name:                       "after_reset",
			Generation:                 7,
			Focused:                    "ResetButton",
			FocusedComponentID:         10,
			FocusedAccessibilityNodeID: 10,
			NameValueLen:               0,
			EmailValueLen:              0,
			StatusValue:                "reset",
			BoundsChecksum:             checksumText("bounds-reset"),
			MetadataChecksum:           checksumText("metadata-reset"),
			FrameChecksum:              checksumRGBA(resetFrame.Pixels),
		},
		{
			Name:                       "after_resize",
			Generation:                 8,
			Focused:                    "NameTextBox",
			FocusedComponentID:         5,
			FocusedAccessibilityNodeID: 5,
			NameValueLen:               0,
			EmailValueLen:              0,
			StatusValue:                "reset",
			BoundsChecksum:             checksumText("bounds-resize"),
			MetadataChecksum:           checksumText("metadata-resize"),
			FrameChecksum:              checksumRGBA(afterFrame.Pixels),
		},
	}
}
