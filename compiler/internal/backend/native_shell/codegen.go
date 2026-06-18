package native_shell

import (
	"encoding/json"
	"strconv"
	"strings"

	"tetra_language/compiler/internal/lower"
)

type shellReport struct {
	Schema   string            `json:"schema"`
	UISchema string            `json:"ui_schema,omitempty"`
	Runtime  string            `json:"runtime"`
	States   []shellStateTrace `json:"states,omitempty"`
	Views    []shellViewTrace  `json:"views,omitempty"`
}

type shellStateTrace struct {
	Name   string                 `json:"name"`
	Module string                 `json:"module,omitempty"`
	Fields []shellStateFieldTrace `json:"fields,omitempty"`
}

type shellStateFieldTrace struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Mutable bool   `json:"mutable,omitempty"`
	Const   bool   `json:"const,omitempty"`
	Value   string `json:"value"`
}

type shellViewTrace struct {
	Name      string                   `json:"name"`
	Module    string                   `json:"module,omitempty"`
	StateType string                   `json:"state_type"`
	Bindings  []shellBindingTrace      `json:"bindings,omitempty"`
	Widgets   []shellWidgetTrace       `json:"widgets,omitempty"`
	Events    []shellEventDispatch     `json:"events,omitempty"`
	Styles    []lower.UILoweredStyle   `json:"styles,omitempty"`
	A11y      []shellAccessibilityItem `json:"accessibility,omitempty"`
}

type shellWidgetTrace struct {
	ID            string                   `json:"id"`
	Kind          string                   `json:"kind"`
	Binding       string                   `json:"binding,omitempty"`
	Event         string                   `json:"event,omitempty"`
	Command       string                   `json:"command,omitempty"`
	Type          string                   `json:"type,omitempty"`
	Value         string                   `json:"value,omitempty"`
	Styles        []lower.UILoweredStyle   `json:"styles,omitempty"`
	Accessibility []shellAccessibilityItem `json:"accessibility,omitempty"`
}

type shellEventDispatch struct {
	Name       string                `json:"name"`
	Command    string                `json:"command"`
	Operations []shellOperationTrace `json:"operations,omitempty"`
	Bindings   []shellBindingTrace   `json:"bindings,omitempty"`
}

type shellOperationTrace struct {
	Kind       string `json:"kind"`
	Target     string `json:"target"`
	Value      string `json:"value,omitempty"`
	StateField string `json:"state_field,omitempty"`
	StateValue string `json:"state_value,omitempty"`
}

type shellBindingTrace struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type shellAccessibilityItem struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

func Render(bundle *lower.UILoweredBundle) []byte {
	if bundle == nil {
		return []byte("Tetra Native UI Shell\n(no UI metadata)\n")
	}
	if bundle.Schema != lower.UIBundleSchema {
		return []byte(
			"Tetra Native UI Shell\nunsupported UI schema: " + bundle.Schema + "\nruntime: unavailable\n",
		)
	}
	state := initialState(bundle)
	lines := []string{
		"Tetra Native UI Shell",
		"schema: " + bundle.Schema,
		"runtime: native shell command dispatch",
	}
	for _, state := range bundle.States {
		lines = append(lines, "")
		lines = append(lines, "state "+state.Name)
		for _, field := range state.Fields {
			kind := "val"
			if field.Mutable {
				kind = "var"
			} else if field.Const {
				kind = "const"
			}
			lines = append(lines, "  "+kind+" "+field.Name+": "+field.Type+" = "+field.Init)
		}
	}
	for _, view := range bundle.Views {
		lines = append(lines, "")
		lines = append(lines, "view "+view.Name+" (state: "+view.StateType+")")
		for _, binding := range view.Bindings {
			lines = append(
				lines,
				"  bind "+binding.Name+": "+binding.Type+" = "+bindingValue(state, view, binding),
			)
		}
		for _, event := range view.Events {
			lines = append(lines, "  event "+event.Name+" -> "+event.Command)
		}
		for _, cmd := range view.Commands {
			lines = append(lines, "  command "+cmd.Name+" ("+itoa(cmd.StatementCount)+" stmt)")
			for _, op := range cmd.Operations {
				lines = append(lines, "    op "+op.Kind+" "+op.Target+" "+op.Value)
			}
		}
		lines = append(lines, dispatchTranscript(state, view)...)
		for _, style := range view.Styles {
			lines = append(lines, "  style "+style.Name+": "+style.Type+" = "+style.Value)
		}
		for _, entry := range view.Accessibility {
			lines = append(lines, "  accessibility "+entry.Name+": "+entry.Type+" = "+entry.Value)
		}
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func RenderJSON(bundle *lower.UILoweredBundle) []byte {
	report := buildReport(bundle)
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return []byte(`{"schema":"tetra.ui.native-shell.v1","runtime":"unavailable"}` + "\n")
	}
	return append(raw, '\n')
}

func buildReport(bundle *lower.UILoweredBundle) shellReport {
	report := shellReport{
		Schema:  "tetra.ui.native-shell.v1",
		Runtime: "native shell command dispatch",
	}
	if bundle == nil {
		report.Runtime = "unavailable"
		return report
	}
	report.UISchema = bundle.Schema
	if bundle.Schema != lower.UIBundleSchema {
		report.Runtime = "unavailable"
		return report
	}
	state := initialState(bundle)
	for _, group := range bundle.States {
		entry := shellStateTrace{Name: group.Name, Module: group.Module}
		for _, field := range group.Fields {
			entry.Fields = append(entry.Fields, shellStateFieldTrace{
				Name:    field.Name,
				Type:    field.Type,
				Mutable: field.Mutable,
				Const:   field.Const,
				Value:   state[group.Name][field.Name],
			})
		}
		report.States = append(report.States, entry)
	}
	for _, view := range bundle.Views {
		viewTrace := shellViewTrace{
			Name:      view.Name,
			Module:    view.Module,
			StateType: view.StateType,
			Bindings:  bindingTraces(state, view),
			Styles:    append([]lower.UILoweredStyle(nil), view.Styles...),
		}
		for _, entry := range view.Accessibility {
			viewTrace.A11y = append(
				viewTrace.A11y,
				shellAccessibilityItem{Name: entry.Name, Type: entry.Type, Value: entry.Value},
			)
		}
		viewTrace.Widgets = widgetTraces(state, view, viewTrace.A11y)
		for _, event := range view.Events {
			command, ok := commandByName(view, event.Command)
			if !ok {
				continue
			}
			eventTrace := shellEventDispatch{Name: event.Name, Command: event.Command}
			for _, op := range command.Operations {
				opTrace := shellOperationTrace{Kind: op.Kind, Target: op.Target, Value: op.Value}
				if field, value, ok := applyOperation(stateForView(state, view), op); ok {
					opTrace.StateField = field
					opTrace.StateValue = value
				}
				eventTrace.Operations = append(eventTrace.Operations, opTrace)
			}
			eventTrace.Bindings = bindingTraces(state, view)
			viewTrace.Events = append(viewTrace.Events, eventTrace)
		}
		report.Views = append(report.Views, viewTrace)
	}
	return report
}

func widgetTraces(
	state map[string]map[string]string,
	view lower.UILoweredView,
	a11y []shellAccessibilityItem,
) []shellWidgetTrace {
	out := make([]shellWidgetTrace, 0, len(view.Bindings)+len(view.Events))
	for _, binding := range view.Bindings {
		out = append(out, shellWidgetTrace{
			ID:            view.Name + "." + binding.Name,
			Kind:          widgetKind(binding.Type),
			Binding:       binding.Name,
			Type:          binding.Type,
			Value:         bindingValue(state, view, binding),
			Styles:        append([]lower.UILoweredStyle(nil), view.Styles...),
			Accessibility: append([]shellAccessibilityItem(nil), a11y...),
		})
	}
	for _, event := range view.Events {
		out = append(out, shellWidgetTrace{
			ID:      view.Name + "." + event.Name,
			Kind:    "action",
			Event:   event.Name,
			Command: event.Command,
		})
	}
	return out
}

func widgetKind(typ string) string {
	if typ == "str" || typ == "String" {
		return "text"
	}
	return "value"
}

func bindingTraces(
	state map[string]map[string]string,
	view lower.UILoweredView,
) []shellBindingTrace {
	out := make([]shellBindingTrace, 0, len(view.Bindings))
	for _, binding := range view.Bindings {
		out = append(
			out,
			shellBindingTrace{
				Name:  binding.Name,
				Type:  binding.Type,
				Value: bindingValue(state, view, binding),
			},
		)
	}
	return out
}

func initialState(bundle *lower.UILoweredBundle) map[string]map[string]string {
	out := map[string]map[string]string{}
	for _, group := range bundle.States {
		fields := map[string]string{}
		for _, field := range group.Fields {
			fields[field.Name] = parseInit(field)
		}
		out[group.Name] = fields
	}
	return out
}

func parseInit(field lower.UILoweredStateField) string {
	if field.Type == "str" && len(field.Init) >= 2 && strings.HasPrefix(field.Init, `"`) &&
		strings.HasSuffix(field.Init, `"`) {
		return strings.TrimSuffix(strings.TrimPrefix(field.Init, `"`), `"`)
	}
	return field.Init
}

func bindingValue(
	state map[string]map[string]string,
	view lower.UILoweredView,
	binding lower.UILoweredBinding,
) string {
	if field, ok := stateFieldName(binding.Source); ok {
		return stateForView(state, view)[field]
	}
	return binding.Source
}

func dispatchTranscript(state map[string]map[string]string, view lower.UILoweredView) []string {
	var lines []string
	for _, event := range view.Events {
		command, ok := commandByName(view, event.Command)
		if !ok {
			continue
		}
		lines = append(lines, "  dispatch "+event.Name+" -> "+event.Command)
		for _, op := range command.Operations {
			if field, value, ok := applyOperation(stateForView(state, view), op); ok {
				lines = append(lines, "    state."+field+" = "+value)
			}
		}
		for _, binding := range view.Bindings {
			lines = append(
				lines,
				"  bind "+binding.Name+": "+binding.Type+" = "+bindingValue(state, view, binding),
			)
		}
	}
	return lines
}

func commandByName(view lower.UILoweredView, name string) (lower.UILoweredCommand, bool) {
	for _, command := range view.Commands {
		if command.Name == name {
			return command, true
		}
	}
	return lower.UILoweredCommand{}, false
}

func stateForView(state map[string]map[string]string, view lower.UILoweredView) map[string]string {
	if fields, ok := state[view.StateType]; ok {
		return fields
	}
	fields := map[string]string{}
	state[view.StateType] = fields
	return fields
}

func applyOperation(
	fields map[string]string,
	op lower.UILoweredCommandOperation,
) (string, string, bool) {
	field, ok := stateFieldName(op.Target)
	if !ok {
		return "", "", false
	}
	switch op.Kind {
	case "state_add":
		current, _ := strconv.Atoi(fields[field])
		delta, _ := strconv.Atoi(op.Value)
		fields[field] = itoa(current + delta)
	case "state_sub":
		current, _ := strconv.Atoi(fields[field])
		delta, _ := strconv.Atoi(op.Value)
		fields[field] = itoa(current - delta)
	case "state_set":
		fields[field] = parseOperationValue(fields, op.Value)
	default:
		return "", "", false
	}
	return field, fields[field], true
}

func parseOperationValue(fields map[string]string, value string) string {
	if field, ok := stateFieldName(value); ok {
		return fields[field]
	}
	if len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		return strings.TrimSuffix(strings.TrimPrefix(value, `"`), `"`)
	}
	return value
}

func stateFieldName(path string) (string, bool) {
	const prefix = "state."
	if !strings.HasPrefix(path, prefix) || len(path) == len(prefix) {
		return "", false
	}
	return strings.TrimPrefix(path, prefix), true
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + (v % 10))
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
