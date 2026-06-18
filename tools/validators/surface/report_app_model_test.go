package surface

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateSurfaceAppModelReport(t *testing.T) {
	raw := validHeadlessAppModelSurfaceReportJSON(t, nil)
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateSurfaceAppModelRejectsIncompleteEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		want   string
	}{
		{
			name: "hidden app state",
			mutate: func(report map[string]any) {
				report["app_model"].(map[string]any)["hidden_app_state"] = true
			},
			want: "hidden app state",
		},
		{
			name: "React runtime",
			mutate: func(report map[string]any) {
				report["app_model"].(map[string]any)["react_runtime"] = true
			},
			want: "React",
		},
		{
			name: "DOM event model",
			mutate: func(report map[string]any) {
				report["app_model"].(map[string]any)["dom_event_model"] = true
			},
			want: "DOM event model",
		},
		{
			name: "command without binding",
			mutate: func(report map[string]any) {
				app := report["app_model"].(map[string]any)
				bindings := app["event_bindings"].([]any)
				app["event_bindings"] = bindings[:len(bindings)-1]
			},
			want: "no explicit event binding",
		},
		{
			name: "async complete without start",
			mutate: func(report map[string]any) {
				app := report["app_model"].(map[string]any)
				tasks := app["async_tasks"].([]any)
				app["async_tasks"] = tasks[1:]
			},
			want: "completed without matching start",
		},
		{
			name: "async cancel mutates state",
			mutate: func(report map[string]any) {
				app := report["app_model"].(map[string]any)
				tasks := app["async_tasks"].([]any)
				cancel := tasks[2].(map[string]any)
				cancel["after_state"] = map[string]any{"pending_task": "0", "save_count": "2"}
			},
			want: "canceled command must not mutate app state",
		},
		{
			name: "navigation underflow drift",
			mutate: func(report map[string]any) {
				app := report["app_model"].(map[string]any)
				nav := app["navigation_transitions"].([]any)
				underflow := nav[2].(map[string]any)
				underflow["after_route"] = "settings"
			},
			want: "underflow rejection must preserve route and stack",
		},
		{
			name: "focus scope escape",
			mutate: func(report map[string]any) {
				app := report["app_model"].(map[string]any)
				focus := app["focus_scope_transitions"].([]any)
				modal := focus[1].(map[string]any)
				modal["escaped"] = true
			},
			want: "escaped active scope",
		},
		{
			name: "undo redo without history",
			mutate: func(report map[string]any) {
				app := report["app_model"].(map[string]any)
				history := app["undo_redo_transitions"].([]any)
				undo := history[1].(map[string]any)
				undo["matched_history_entry"] = false
			},
			want: "matched applied history entry",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := validHeadlessAppModelSurfaceReportJSON(t, tc.mutate)
			err := ValidateReport(raw)
			if err == nil {
				t.Fatalf("expected app_model %s evidence to fail", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}
func validHeadlessAppModelSurfaceReportJSON(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode base headless report: %v", err)
	}
	report["source"] = "examples/surface_app_model.tetra"
	report["processes"] = []any{
		map[string]any{"name": "tetra build", "kind": "build", "path": "tetra build --target linux-x64 examples/surface_app_model.tetra -o /tmp/surface-artifacts/surface-app-model", "ran": true, "pass": true, "exit_code": 0},
		map[string]any{"name": "surface component app", "kind": "app", "path": "/tmp/surface-artifacts/surface-app-model", "ran": true, "pass": true, "exit_code": 1, "expected_exit_code": 1},
		map[string]any{"name": "surface headless runtime", "kind": "runtime", "path": "tools/cmd/surface-runtime-smoke", "ran": true, "pass": true, "exit_code": 0},
	}
	report["artifacts"] = []any{
		map[string]any{"kind": "component-app", "path": "/tmp/surface-artifacts/surface-app-model", "sha256": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "size": 98234},
		map[string]any{"kind": "runner-trace", "path": "/tmp/surface-artifacts/surface-runner-trace.json", "sha256": "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "size": 26000},
	}
	report["components"] = []any{
		componentMap("AppModelApp", "examples.surface_app_model.AppModelApp", "", RectReport{X: 0, Y: 0, W: 480, H: 320}, map[string]string{"route": "settings", "focused": "NameField", "save_count": "1", "pending_task": "0", "history_depth": "1", "redo_depth": "0", "accessibility_role": "none"}),
		componentMap("NameField", "examples.surface_app_model.NameField", "AppModelApp", RectReport{X: 32, Y: 80, W: 240, H: 44}, map[string]string{"focused": "true", "buffer": "Ada", "caret": "3", "accessibility_role": "textbox"}),
		componentMap("SaveButton", "examples.surface_app_model.SaveButton", "AppModelApp", RectReport{X: 32, Y: 144, W: 132, H: 44}, map[string]string{"focused": "false", "press_count": "1", "action": "save", "accessibility_role": "button"}),
	}
	report["events"] = []any{
		eventMap(1, "mouse_up", "NameField", []any{"AppModelApp", "NameField"}, 48, 96, 0, 480, 320, map[string]string{"AppModelApp.focused": ""}, map[string]string{"AppModelApp.focused": "NameField"}),
		textEventMap(2, "NameField", []any{"AppModelApp", "NameField"}, 3, "416461", 480, 320, map[string]string{"NameField.buffer": ""}, map[string]string{"NameField.buffer": "Ada"}),
		keyEventMap(3, "SaveButton", []any{"AppModelApp", "SaveButton"}, 13, 480, 320, map[string]string{"AppModelApp.save_count": "0"}, map[string]string{"AppModelApp.save_count": "1"}),
	}
	report["state_transitions"] = []any{
		map[string]any{"order": 1, "component": "AppModelApp", "field": "focused", "before": "", "after": "NameField", "cause": "focus"},
		map[string]any{"order": 2, "component": "NameField", "field": "buffer", "before": "", "after": "Ada", "cause": "command.insert_text"},
		map[string]any{"order": 3, "component": "AppModelApp", "field": "route", "before": "home", "after": "settings", "cause": "command.navigate"},
		map[string]any{"order": 4, "component": "AppModelApp", "field": "pending_task", "before": "1", "after": "0", "cause": "command.async_complete"},
		map[string]any{"order": 5, "component": "AppModelApp", "field": "history_depth", "before": "0", "after": "1", "cause": "command.undoable"},
		map[string]any{"order": 6, "component": "AppModelApp", "field": "save_count", "before": "0", "after": "1", "cause": "command.save"},
	}
	report["app_model"] = map[string]any{
		"schema":                  "tetra.surface.app-model.v1",
		"app_model_level":         "explicit-command-reducer-v1",
		"release_scope":           "surface-v1-linux-web",
		"source":                  "examples/surface_app_model.tetra",
		"module":                  "lib.core.surface_app",
		"uses_component_tree_api": true,
		"caller_owned_state":      true,
		"explicit_event_bindings": true,
		"deterministic_reducer":   true,
		"hidden_app_state":        false,
		"react_runtime":           false,
		"electron_runtime":        false,
		"dom_runtime":             false,
		"dom_event_model":         false,
		"user_js":                 false,
		"platform_widgets":        false,
		"state_fields":            []any{"route", "focused", "name_buffer", "save_count", "pending_task", "history_depth", "redo_depth"},
		"command_registry":        []any{"focus.name", "text.insert", "nav.push.settings", "nav.back", "async.save.start", "async.save.complete", "async.save.cancel", "history.undo", "history.redo"},
		"event_bindings":          validAppModelEventBindings(),
		"command_dispatches":      validAppModelCommandDispatches(),
		"navigation_transitions":  validAppModelNavigationTransitions(),
		"focus_scope_transitions": validAppModelFocusScopeTransitions(),
		"async_tasks":             validAppModelAsyncTasks(),
		"undo_redo_transitions":   validAppModelUndoRedoTransitions(),
		"negative_guards":         validAppModelNegativeGuards(),
	}
	report["cases"] = append(report["cases"].([]any),
		map[string]any{"name": "app model explicit event-to-command binding", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "app model deterministic command reducer", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "app model navigation stack", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "app model focus scope modal trap", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "app model async completion cancellation boundary", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "app model undo redo history", "kind": "positive", "ran": true, "pass": true},
		map[string]any{"name": "app model no React hooks DOM event model hidden JS state", "kind": "positive", "ran": true, "pass": true},
	)
	if mutate != nil {
		mutate(report)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal app model report: %v", err)
	}
	return raw
}
func validAppModelEventBindings() []any {
	return []any{
		map[string]any{"order": 1, "event_order": 1, "event_kind": "mouse_up", "target": "NameField", "dispatch_path": []any{"AppModelApp", "NameField"}, "command": "focus.name", "explicit": true},
		map[string]any{"order": 2, "event_order": 2, "event_kind": "text_input", "target": "NameField", "dispatch_path": []any{"AppModelApp", "NameField"}, "command": "text.insert", "explicit": true},
		map[string]any{"order": 3, "event_order": 3, "event_kind": "key_down", "target": "SaveButton", "dispatch_path": []any{"AppModelApp", "SaveButton"}, "command": "async.save.start", "explicit": true},
	}
}
func validAppModelCommandDispatches() []any {
	return []any{
		map[string]any{"order": 1, "event_order": 1, "command": "focus.name", "kind": "focus", "target": "NameField", "handled": true, "before_state": map[string]any{"focused": ""}, "after_state": map[string]any{"focused": "NameField"}},
		map[string]any{"order": 2, "event_order": 2, "command": "text.insert", "kind": "edit", "target": "NameField", "handled": true, "reversible": true, "history_index": 1, "before_state": map[string]any{"name_buffer": ""}, "after_state": map[string]any{"name_buffer": "Ada"}},
		map[string]any{"order": 3, "event_order": 3, "command": "async.save.start", "kind": "async_start", "target": "SaveButton", "handled": true, "async_task_id": "save-1", "before_state": map[string]any{"pending_task": "0"}, "after_state": map[string]any{"pending_task": "1"}},
		map[string]any{"order": 4, "event_order": 0, "command": "async.save.complete", "kind": "async_complete", "target": "AppModelApp", "handled": true, "async_task_id": "save-1", "before_state": map[string]any{"pending_task": "1", "save_count": "0"}, "after_state": map[string]any{"pending_task": "0", "save_count": "1"}},
	}
}
func validAppModelNavigationTransitions() []any {
	return []any{
		map[string]any{"order": 1, "command": "nav.push.settings", "operation": "push", "before_route": "home", "after_route": "settings", "stack_before": []any{"home"}, "stack_after": []any{"home", "settings"}, "underflow_rejected": false},
		map[string]any{"order": 2, "command": "nav.back", "operation": "back", "before_route": "settings", "after_route": "home", "stack_before": []any{"home", "settings"}, "stack_after": []any{"home"}, "underflow_rejected": false},
		map[string]any{"order": 3, "command": "nav.back", "operation": "back", "before_route": "home", "after_route": "home", "stack_before": []any{"home"}, "stack_after": []any{"home"}, "underflow_rejected": true},
	}
}
func validAppModelFocusScopeTransitions() []any {
	return []any{
		map[string]any{"order": 1, "scope": "main", "before_focus": "", "after_focus": "NameField", "wrapped": false, "modal_trap": false, "escaped": false},
		map[string]any{"order": 2, "scope": "dialog", "before_focus": "DialogCancel", "after_focus": "DialogConfirm", "wrapped": true, "modal_trap": true, "escaped": false},
	}
}
func validAppModelAsyncTasks() []any {
	return []any{
		map[string]any{"id": "save-1", "command": "async.save.start", "operation": "start", "status": "pending", "before_state": map[string]any{"pending_task": "0"}, "after_state": map[string]any{"pending_task": "1"}, "completion_order": 0, "canceled": false},
		map[string]any{"id": "save-1", "command": "async.save.complete", "operation": "complete", "status": "completed", "before_state": map[string]any{"pending_task": "1"}, "after_state": map[string]any{"pending_task": "0"}, "completion_order": 4, "canceled": false},
		map[string]any{"id": "save-2", "command": "async.save.cancel", "operation": "cancel", "status": "canceled", "before_state": map[string]any{"pending_task": "1", "save_count": "1"}, "after_state": map[string]any{"pending_task": "0", "save_count": "1"}, "completion_order": 0, "canceled": true},
	}
}
func validAppModelUndoRedoTransitions() []any {
	return []any{
		map[string]any{"order": 1, "command": "text.insert", "history_index": 1, "operation": "record", "before": "", "after": "Ada", "matched_history_entry": true, "applied": true},
		map[string]any{"order": 2, "command": "history.undo", "history_index": 1, "operation": "undo", "before": "Ada", "after": "", "matched_history_entry": true, "applied": true},
		map[string]any{"order": 3, "command": "history.redo", "history_index": 1, "operation": "redo", "before": "", "after": "Ada", "matched_history_entry": true, "applied": true},
	}
}
func validAppModelNegativeGuards() map[string]any {
	return map[string]any{
		"no_hidden_app_state":              true,
		"no_react_hooks":                   true,
		"no_dom_event_model":               true,
		"no_user_js":                       true,
		"no_platform_widgets":              true,
		"async_cancel_no_mutation":         true,
		"navigation_underflow_rejected":    true,
		"focus_scope_escape_rejected":      true,
		"undo_redo_requires_history":       true,
		"command_without_binding_rejected": true,
	}
}
