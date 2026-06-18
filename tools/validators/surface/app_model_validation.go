package surface

import (
	"fmt"
	"strings"
)

type AppModelReport struct {
	Schema                string                          `json:"schema"`
	AppModelLevel         string                          `json:"app_model_level"`
	ReleaseScope          string                          `json:"release_scope"`
	Source                string                          `json:"source"`
	Module                string                          `json:"module"`
	UsesComponentTreeAPI  bool                            `json:"uses_component_tree_api"`
	CallerOwnedState      bool                            `json:"caller_owned_state"`
	ExplicitEventBindings bool                            `json:"explicit_event_bindings"`
	DeterministicReducer  bool                            `json:"deterministic_reducer"`
	HiddenAppState        bool                            `json:"hidden_app_state"`
	ReactRuntime          bool                            `json:"react_runtime"`
	ElectronRuntime       bool                            `json:"electron_runtime"`
	DOMRuntime            bool                            `json:"dom_runtime"`
	DOMEventModel         bool                            `json:"dom_event_model"`
	UserJS                bool                            `json:"user_js"`
	PlatformWidgets       bool                            `json:"platform_widgets"`
	StateFields           []string                        `json:"state_fields"`
	CommandRegistry       []string                        `json:"command_registry"`
	EventBindings         []AppModelEventBindingReport    `json:"event_bindings"`
	CommandDispatches     []AppModelCommandDispatchReport `json:"command_dispatches"`
	NavigationTransitions []AppModelNavigationReport      `json:"navigation_transitions"`
	FocusScopeTransitions []AppModelFocusScopeReport      `json:"focus_scope_transitions"`
	AsyncTasks            []AppModelAsyncTaskReport       `json:"async_tasks"`
	UndoRedoTransitions   []AppModelUndoRedoReport        `json:"undo_redo_transitions"`
	NegativeGuards        AppModelNegativeGuardsReport    `json:"negative_guards"`
}

type AppModelEventBindingReport struct {
	Order        int      `json:"order"`
	EventOrder   int      `json:"event_order"`
	EventKind    string   `json:"event_kind"`
	Target       string   `json:"target"`
	DispatchPath []string `json:"dispatch_path"`
	Command      string   `json:"command"`
	Explicit     bool     `json:"explicit"`
}

type AppModelCommandDispatchReport struct {
	Order        int               `json:"order"`
	EventOrder   int               `json:"event_order"`
	Command      string            `json:"command"`
	Kind         string            `json:"kind"`
	Target       string            `json:"target"`
	Handled      bool              `json:"handled"`
	BeforeState  map[string]string `json:"before_state"`
	AfterState   map[string]string `json:"after_state"`
	Reversible   bool              `json:"reversible,omitempty"`
	HistoryIndex int               `json:"history_index,omitempty"`
	AsyncTaskID  string            `json:"async_task_id,omitempty"`
}

type AppModelNavigationReport struct {
	Order             int      `json:"order"`
	Command           string   `json:"command"`
	Operation         string   `json:"operation"`
	BeforeRoute       string   `json:"before_route"`
	AfterRoute        string   `json:"after_route"`
	StackBefore       []string `json:"stack_before"`
	StackAfter        []string `json:"stack_after"`
	UnderflowRejected bool     `json:"underflow_rejected"`
}

type AppModelFocusScopeReport struct {
	Order       int    `json:"order"`
	Scope       string `json:"scope"`
	BeforeFocus string `json:"before_focus"`
	AfterFocus  string `json:"after_focus"`
	Wrapped     bool   `json:"wrapped"`
	ModalTrap   bool   `json:"modal_trap"`
	Escaped     bool   `json:"escaped"`
}

type AppModelAsyncTaskReport struct {
	ID              string            `json:"id"`
	Command         string            `json:"command"`
	Operation       string            `json:"operation"`
	Status          string            `json:"status"`
	BeforeState     map[string]string `json:"before_state"`
	AfterState      map[string]string `json:"after_state"`
	CompletionOrder int               `json:"completion_order"`
	Canceled        bool              `json:"canceled"`
}

type AppModelUndoRedoReport struct {
	Order               int    `json:"order"`
	Command             string `json:"command"`
	HistoryIndex        int    `json:"history_index"`
	Operation           string `json:"operation"`
	Before              string `json:"before"`
	After               string `json:"after"`
	MatchedHistoryEntry bool   `json:"matched_history_entry"`
	Applied             bool   `json:"applied"`
}

type AppModelNegativeGuardsReport struct {
	NoHiddenAppState              bool `json:"no_hidden_app_state"`
	NoReactHooks                  bool `json:"no_react_hooks"`
	NoDOMEventModel               bool `json:"no_dom_event_model"`
	NoUserJS                      bool `json:"no_user_js"`
	NoPlatformWidgets             bool `json:"no_platform_widgets"`
	AsyncCancelNoMutation         bool `json:"async_cancel_no_mutation"`
	NavigationUnderflowRejected   bool `json:"navigation_underflow_rejected"`
	FocusScopeEscapeRejected      bool `json:"focus_scope_escape_rejected"`
	UndoRedoRequiresHistory       bool `json:"undo_redo_requires_history"`
	CommandWithoutBindingRejected bool `json:"command_without_binding_rejected"`
}

func validateAppModelEvidence(report Report) []string {
	if !isAppModelReport(report) {
		return nil
	}
	var issues []string
	if !isSurfaceAppModelSource(report.Source) {
		issues = append(issues, fmt.Sprintf("app_model source path must match examples/surface_app_model.tetra, got %q", report.Source))
	}
	if report.AppModel == nil {
		return append(issues, "app_model evidence is required for examples/surface_app_model.tetra")
	}
	app := report.AppModel
	if app.Schema != "tetra.surface.app-model.v1" {
		issues = append(issues, fmt.Sprintf("app_model schema is %q, want tetra.surface.app-model.v1", app.Schema))
	}
	if app.AppModelLevel != "explicit-command-reducer-v1" {
		issues = append(issues, fmt.Sprintf("app_model app_model_level is %q, want explicit-command-reducer-v1", app.AppModelLevel))
	}
	if app.ReleaseScope != "surface-v1-linux-web" {
		issues = append(issues, fmt.Sprintf("app_model release_scope is %q, want surface-v1-linux-web", app.ReleaseScope))
	}
	if normalizeEvidencePath(app.Source) != normalizeEvidencePath(report.Source) {
		issues = append(issues, fmt.Sprintf("app_model source %q must match report source %q", app.Source, report.Source))
	}
	if app.Module != "lib.core.surface_app" {
		issues = append(issues, fmt.Sprintf("app_model module is %q, want lib.core.surface_app", app.Module))
	}
	if !app.UsesComponentTreeAPI || !app.CallerOwnedState || !app.ExplicitEventBindings || !app.DeterministicReducer {
		issues = append(issues, "app_model requires component-tree API use, caller-owned state, explicit event bindings, and deterministic reducer evidence")
	}
	if app.HiddenAppState || app.ReactRuntime || app.ElectronRuntime || app.DOMRuntime || app.DOMEventModel || app.UserJS || app.PlatformWidgets {
		issues = append(issues, "app_model must not claim hidden app state, React/Electron/DOM runtime, DOM event model, user JS, or platform widgets")
	}
	for _, field := range []string{"route", "focused", "name_buffer", "save_count", "pending_task", "history_depth", "redo_depth"} {
		if !contains(app.StateFields, field) {
			issues = append(issues, fmt.Sprintf("app_model state_fields missing %s", field))
		}
	}
	for _, command := range []string{"focus.name", "text.insert", "nav.push.settings", "nav.back", "async.save.start", "async.save.complete", "async.save.cancel", "history.undo", "history.redo"} {
		if !contains(app.CommandRegistry, command) {
			issues = append(issues, fmt.Sprintf("app_model command_registry missing %s", command))
		}
	}

	commandSet := map[string]bool{}
	for _, command := range app.CommandRegistry {
		commandSet[command] = true
	}
	eventByOrder := map[int]EventReport{}
	for _, event := range report.Events {
		eventByOrder[event.Order] = event
	}
	bindingByEventCommand := map[string]bool{}
	issues = append(issues, validateAppModelEventBindings(app.EventBindings, eventByOrder, commandSet, bindingByEventCommand)...)
	issues = append(issues, validateAppModelCommandDispatches(app.CommandDispatches, report.Components, commandSet, bindingByEventCommand)...)
	issues = append(issues, validateAppModelNavigation(app.NavigationTransitions, commandSet)...)
	issues = append(issues, validateAppModelFocusScopes(app.FocusScopeTransitions)...)
	issues = append(issues, validateAppModelAsyncTasks(app.AsyncTasks, commandSet)...)
	issues = append(issues, validateAppModelUndoRedo(app.UndoRedoTransitions, commandSet)...)
	issues = append(issues, validateAppModelNegativeGuards(app.NegativeGuards)...)
	for _, required := range []string{
		"app model explicit event-to-command binding",
		"app model deterministic command reducer",
		"app model navigation stack",
		"app model focus scope modal trap",
		"app model async completion cancellation boundary",
		"app model undo redo history",
		"app model no React hooks DOM event model hidden JS state",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("app_model report requires %s evidence", required))
		}
	}
	return issues
}

func validateAppModelEventBindings(bindings []AppModelEventBindingReport, eventByOrder map[int]EventReport, commandSet map[string]bool, bindingByEventCommand map[string]bool) []string {
	var issues []string
	if len(bindings) == 0 {
		return []string{"app_model event_bindings are required"}
	}
	lastOrder := 0
	for _, binding := range bindings {
		if binding.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("app_model event binding order %d is not strictly greater than previous order %d", binding.Order, lastOrder))
		}
		lastOrder = binding.Order
		if binding.EventOrder <= 0 {
			issues = append(issues, fmt.Sprintf("app_model event binding %d event_order must be positive", binding.Order))
		}
		event, ok := eventByOrder[binding.EventOrder]
		if !ok {
			issues = append(issues, fmt.Sprintf("app_model event binding %d references missing event order %d", binding.Order, binding.EventOrder))
		} else {
			if event.Kind != binding.EventKind || event.TargetComponent != binding.Target {
				issues = append(issues, fmt.Sprintf("app_model event binding %d = %s/%s, want event %d %s/%s", binding.Order, binding.EventKind, binding.Target, event.Order, event.Kind, event.TargetComponent))
			}
			if !stringSlicesEqual(binding.DispatchPath, event.DispatchPath) {
				issues = append(issues, fmt.Sprintf("app_model event binding %d dispatch_path = %v, want event path %v", binding.Order, binding.DispatchPath, event.DispatchPath))
			}
		}
		if !binding.Explicit {
			issues = append(issues, fmt.Sprintf("app_model event binding %d must be explicit", binding.Order))
		}
		if strings.TrimSpace(binding.Command) == "" || !commandSet[binding.Command] {
			issues = append(issues, fmt.Sprintf("app_model event binding %d references unregistered command %q", binding.Order, binding.Command))
		}
		bindingByEventCommand[appModelEventCommandKey(binding.EventOrder, binding.Command)] = true
	}
	return issues
}

func validateAppModelCommandDispatches(dispatches []AppModelCommandDispatchReport, components []ComponentReport, commandSet map[string]bool, bindingByEventCommand map[string]bool) []string {
	var issues []string
	if len(dispatches) == 0 {
		return []string{"app_model command_dispatches are required"}
	}
	componentSet := map[string]bool{}
	for _, component := range components {
		componentSet[component.ID] = true
	}
	lastOrder := 0
	seenKinds := map[string]bool{}
	for _, dispatch := range dispatches {
		if dispatch.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("app_model command dispatch order %d is not strictly greater than previous order %d", dispatch.Order, lastOrder))
		}
		lastOrder = dispatch.Order
		if strings.TrimSpace(dispatch.Command) == "" || !commandSet[dispatch.Command] {
			issues = append(issues, fmt.Sprintf("app_model command dispatch %d references unregistered command %q", dispatch.Order, dispatch.Command))
		}
		if strings.TrimSpace(dispatch.Kind) == "" {
			issues = append(issues, fmt.Sprintf("app_model command dispatch %d kind is required", dispatch.Order))
		}
		seenKinds[dispatch.Kind] = true
		if strings.TrimSpace(dispatch.Target) == "" || !componentSet[dispatch.Target] {
			issues = append(issues, fmt.Sprintf("app_model command dispatch %d target %q is not in component evidence", dispatch.Order, dispatch.Target))
		}
		if !dispatch.Handled {
			issues = append(issues, fmt.Sprintf("app_model command dispatch %d must be handled", dispatch.Order))
		}
		if len(dispatch.BeforeState) == 0 || len(dispatch.AfterState) == 0 {
			issues = append(issues, fmt.Sprintf("app_model command dispatch %d requires before_state and after_state", dispatch.Order))
		}
		if dispatch.EventOrder > 0 && !bindingByEventCommand[appModelEventCommandKey(dispatch.EventOrder, dispatch.Command)] {
			issues = append(issues, fmt.Sprintf("app_model command dispatch %d has no explicit event binding for event %d command %s", dispatch.Order, dispatch.EventOrder, dispatch.Command))
		}
		if dispatch.Kind == "edit" && (!dispatch.Reversible || dispatch.HistoryIndex <= 0) {
			issues = append(issues, fmt.Sprintf("app_model edit command dispatch %d requires reversible history evidence", dispatch.Order))
		}
		if strings.HasPrefix(dispatch.Kind, "async_") && strings.TrimSpace(dispatch.AsyncTaskID) == "" {
			issues = append(issues, fmt.Sprintf("app_model async command dispatch %d requires async_task_id", dispatch.Order))
		}
	}
	for _, requiredKind := range []string{"focus", "edit", "async_start", "async_complete"} {
		if !seenKinds[requiredKind] {
			issues = append(issues, fmt.Sprintf("app_model command dispatches missing %s kind", requiredKind))
		}
	}
	return issues
}

func validateAppModelNavigation(transitions []AppModelNavigationReport, commandSet map[string]bool) []string {
	var issues []string
	if len(transitions) == 0 {
		return []string{"app_model navigation_transitions are required"}
	}
	lastOrder := 0
	seenPush := false
	seenBack := false
	seenUnderflow := false
	for _, transition := range transitions {
		if transition.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("app_model navigation order %d is not strictly greater than previous order %d", transition.Order, lastOrder))
		}
		lastOrder = transition.Order
		if !commandSet[transition.Command] {
			issues = append(issues, fmt.Sprintf("app_model navigation %d references unregistered command %q", transition.Order, transition.Command))
		}
		switch transition.Operation {
		case "push", "replace", "back":
		default:
			issues = append(issues, fmt.Sprintf("app_model navigation %d operation is %q, want push, replace, or back", transition.Order, transition.Operation))
		}
		if transition.Operation == "push" {
			seenPush = true
		}
		if transition.Operation == "back" {
			seenBack = true
		}
		if len(transition.StackBefore) == 0 || len(transition.StackAfter) == 0 {
			issues = append(issues, fmt.Sprintf("app_model navigation %d requires stack_before and stack_after", transition.Order))
		}
		if strings.TrimSpace(transition.BeforeRoute) == "" || strings.TrimSpace(transition.AfterRoute) == "" {
			issues = append(issues, fmt.Sprintf("app_model navigation %d requires before_route and after_route", transition.Order))
		}
		if transition.UnderflowRejected {
			seenUnderflow = true
			if transition.BeforeRoute != transition.AfterRoute || !stringSlicesEqual(transition.StackBefore, transition.StackAfter) {
				issues = append(issues, fmt.Sprintf("app_model navigation %d underflow rejection must preserve route and stack", transition.Order))
			}
		}
	}
	if !seenPush || !seenBack || !seenUnderflow {
		issues = append(issues, "app_model navigation requires push, back, and underflow rejection evidence")
	}
	return issues
}

func validateAppModelFocusScopes(transitions []AppModelFocusScopeReport) []string {
	var issues []string
	if len(transitions) == 0 {
		return []string{"app_model focus_scope_transitions are required"}
	}
	lastOrder := 0
	seenModalTrap := false
	for _, transition := range transitions {
		if transition.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("app_model focus scope order %d is not strictly greater than previous order %d", transition.Order, lastOrder))
		}
		lastOrder = transition.Order
		if strings.TrimSpace(transition.Scope) == "" || strings.TrimSpace(transition.AfterFocus) == "" {
			issues = append(issues, fmt.Sprintf("app_model focus scope %d requires scope and after_focus", transition.Order))
		}
		if transition.Escaped {
			issues = append(issues, fmt.Sprintf("app_model focus scope %d escaped active scope", transition.Order))
		}
		if transition.ModalTrap && transition.Wrapped && !transition.Escaped {
			seenModalTrap = true
		}
	}
	if !seenModalTrap {
		issues = append(issues, "app_model focus scopes require modal trap wrap evidence")
	}
	return issues
}

func validateAppModelAsyncTasks(tasks []AppModelAsyncTaskReport, commandSet map[string]bool) []string {
	var issues []string
	if len(tasks) == 0 {
		return []string{"app_model async_tasks are required"}
	}
	started := map[string]bool{}
	completed := map[string]bool{}
	seenCancel := false
	for _, task := range tasks {
		if strings.TrimSpace(task.ID) == "" {
			issues = append(issues, "app_model async task id is required")
		}
		if !commandSet[task.Command] {
			issues = append(issues, fmt.Sprintf("app_model async task %s references unregistered command %q", task.ID, task.Command))
		}
		if len(task.BeforeState) == 0 || len(task.AfterState) == 0 {
			issues = append(issues, fmt.Sprintf("app_model async task %s requires before_state and after_state", task.ID))
		}
		switch task.Operation {
		case "start":
			started[task.ID] = true
			if task.Status != "pending" || task.Canceled {
				issues = append(issues, fmt.Sprintf("app_model async task %s start must be pending and not canceled", task.ID))
			}
		case "complete":
			completed[task.ID] = true
			if task.Status != "completed" || task.Canceled || task.CompletionOrder <= 0 {
				issues = append(issues, fmt.Sprintf("app_model async task %s completion requires completed status and completion_order", task.ID))
			}
		case "cancel":
			seenCancel = true
			if task.Status != "canceled" || !task.Canceled {
				issues = append(issues, fmt.Sprintf("app_model async task %s cancel must be canceled", task.ID))
			}
			if appModelCanceledTaskMutatesBusinessState(task.BeforeState, task.AfterState) {
				issues = append(issues, fmt.Sprintf("app_model async task %s canceled command must not mutate app state beyond pending_task", task.ID))
			}
		default:
			issues = append(issues, fmt.Sprintf("app_model async task %s operation is %q, want start, complete, or cancel", task.ID, task.Operation))
		}
	}
	for id := range completed {
		if !started[id] {
			issues = append(issues, fmt.Sprintf("app_model async task %s completed without matching start", id))
		}
	}
	if !seenCancel {
		issues = append(issues, "app_model async tasks require cancel evidence")
	}
	return issues
}

func validateAppModelUndoRedo(transitions []AppModelUndoRedoReport, commandSet map[string]bool) []string {
	var issues []string
	if len(transitions) == 0 {
		return []string{"app_model undo_redo_transitions are required"}
	}
	lastOrder := 0
	seen := map[string]bool{}
	for _, transition := range transitions {
		if transition.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("app_model undo/redo order %d is not strictly greater than previous order %d", transition.Order, lastOrder))
		}
		lastOrder = transition.Order
		if !commandSet[transition.Command] {
			issues = append(issues, fmt.Sprintf("app_model undo/redo %d references unregistered command %q", transition.Order, transition.Command))
		}
		switch transition.Operation {
		case "record", "undo", "redo":
			seen[transition.Operation] = true
		default:
			issues = append(issues, fmt.Sprintf("app_model undo/redo %d operation is %q, want record, undo, or redo", transition.Order, transition.Operation))
		}
		if transition.HistoryIndex <= 0 || !transition.MatchedHistoryEntry || !transition.Applied {
			issues = append(issues, fmt.Sprintf("app_model undo/redo %d requires matched applied history entry", transition.Order))
		}
		if transition.Before == transition.After {
			issues = append(issues, fmt.Sprintf("app_model undo/redo %d must change value", transition.Order))
		}
	}
	for _, required := range []string{"record", "undo", "redo"} {
		if !seen[required] {
			issues = append(issues, fmt.Sprintf("app_model undo_redo_transitions missing %s operation", required))
		}
	}
	return issues
}

func validateAppModelNegativeGuards(guards AppModelNegativeGuardsReport) []string {
	missing := []string{}
	checks := []struct {
		name string
		ok   bool
	}{
		{"no_hidden_app_state", guards.NoHiddenAppState},
		{"no_react_hooks", guards.NoReactHooks},
		{"no_dom_event_model", guards.NoDOMEventModel},
		{"no_user_js", guards.NoUserJS},
		{"no_platform_widgets", guards.NoPlatformWidgets},
		{"async_cancel_no_mutation", guards.AsyncCancelNoMutation},
		{"navigation_underflow_rejected", guards.NavigationUnderflowRejected},
		{"focus_scope_escape_rejected", guards.FocusScopeEscapeRejected},
		{"undo_redo_requires_history", guards.UndoRedoRequiresHistory},
		{"command_without_binding_rejected", guards.CommandWithoutBindingRejected},
	}
	for _, check := range checks {
		if !check.ok {
			missing = append(missing, check.name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("app_model negative_guards missing %s", strings.Join(missing, ", "))}
}

func appModelEventCommandKey(eventOrder int, command string) string {
	return fmt.Sprintf("%d:%s", eventOrder, command)
}

func appModelCanceledTaskMutatesBusinessState(before map[string]string, after map[string]string) bool {
	for key, beforeValue := range before {
		afterValue, ok := after[key]
		if !ok || beforeValue == afterValue {
			continue
		}
		if key != "pending_task" {
			return true
		}
	}
	for key := range after {
		if _, ok := before[key]; !ok && key != "pending_task" {
			return true
		}
	}
	return false
}

func isAppModelReport(report Report) bool {
	if isSurfaceAppModelSource(report.Source) {
		return true
	}
	if report.AppModel != nil {
		return true
	}
	return caseNameContains(report.Cases, "app model")
}

func isSurfaceAppModelSource(source string) bool {
	return strings.HasSuffix(normalizeEvidencePath(source), "examples/surface_app_model.tetra")
}
