package lower

import (
	"fmt"
	"sort"
	"strings"
)

const UIToolkitSchema = "tetra.ui.toolkit.v1"

type UIToolkitBundle struct {
	Schema              string           `json:"schema"`
	CompatibilitySchema string           `json:"compatibility_schema"`
	States              []UIToolkitState `json:"states,omitempty"`
	Views               []UIToolkitView  `json:"views,omitempty"`
}

type UIToolkitState struct {
	Name   string                `json:"name"`
	Module string                `json:"module"`
	Fields []UILoweredStateField `json:"fields,omitempty"`
}

type UIToolkitView struct {
	Name                string                   `json:"name"`
	Module              string                   `json:"module"`
	StateType           string                   `json:"state_type"`
	WidgetKinds         []string                 `json:"widget_kinds"`
	LayoutKinds         []string                 `json:"layout_kinds"`
	StyleStates         []string                 `json:"style_states"`
	AccessibilityFields []string                 `json:"accessibility_fields"`
	EventKinds          []string                 `json:"event_kinds"`
	StateBindingKinds   []string                 `json:"state_binding_kinds"`
	Widgets             []UIToolkitWidget        `json:"widgets"`
	Layouts             []UIToolkitLayout        `json:"layouts"`
	Styles              []UILoweredStyle         `json:"styles,omitempty"`
	Accessibility       []UILoweredAccessibility `json:"accessibility,omitempty"`
	Events              []UIToolkitEvent         `json:"events,omitempty"`
	Commands            []UIToolkitCommand       `json:"commands,omitempty"`
}

type UIToolkitWidget struct {
	ID            string          `json:"id"`
	Kind          string          `json:"kind"`
	Parent        string          `json:"parent,omitempty"`
	Binding       string          `json:"binding,omitempty"`
	Event         string          `json:"event,omitempty"`
	Command       string          `json:"command,omitempty"`
	Layout        UIToolkitLayout `json:"layout"`
	Focusable     bool            `json:"focusable"`
	Accessibility string          `json:"accessibility,omitempty"`
}

type UIToolkitLayout struct {
	Kind      string `json:"kind"`
	Mode      string `json:"mode,omitempty"`
	Order     int    `json:"order,omitempty"`
	Gap       int    `json:"gap,omitempty"`
	Min       int    `json:"min,omitempty"`
	Max       int    `json:"max,omitempty"`
	Preferred int    `json:"preferred,omitempty"`
	Overflow  string `json:"overflow,omitempty"`
}

type UIToolkitEvent struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

type UIToolkitCommand struct {
	Name           string                      `json:"name"`
	StatementCount int                         `json:"statement_count"`
	Operations     []UILoweredCommandOperation `json:"operations"`
}

func LowerUIToolkit(bundle *UILoweredBundle) (*UIToolkitBundle, error) {
	if bundle == nil {
		return nil, nil
	}
	if len(bundle.Views) == 0 {
		return nil, nil
	}
	out := &UIToolkitBundle{
		Schema:              UIToolkitSchema,
		CompatibilitySchema: bundle.Schema,
		States:              make([]UIToolkitState, 0, len(bundle.States)),
		Views:               make([]UIToolkitView, 0, len(bundle.Views)),
	}
	states := append([]UILoweredState(nil), bundle.States...)
	sort.Slice(states, func(i, j int) bool {
		if states[i].Module != states[j].Module {
			return states[i].Module < states[j].Module
		}
		return states[i].Name < states[j].Name
	})
	for _, state := range states {
		fields := append([]UILoweredStateField(nil), state.Fields...)
		sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })
		out.States = append(out.States, UIToolkitState{Name: state.Name, Module: state.Module, Fields: fields})
	}
	views := append([]UILoweredView(nil), bundle.Views...)
	sort.Slice(views, func(i, j int) bool {
		if views[i].Module != views[j].Module {
			return views[i].Module < views[j].Module
		}
		return views[i].Name < views[j].Name
	})
	for _, view := range views {
		entry, err := lowerUIToolkitView(view)
		if err != nil {
			return nil, err
		}
		out.Views = append(out.Views, entry)
	}
	return out, nil
}

func lowerUIToolkitView(view UILoweredView) (UIToolkitView, error) {
	events := append([]UILoweredEvent(nil), view.Events...)
	sort.Slice(events, func(i, j int) bool { return events[i].Name < events[j].Name })
	commands := append([]UILoweredCommand(nil), view.Commands...)
	sort.Slice(commands, func(i, j int) bool { return commands[i].Name < commands[j].Name })
	bindings := append([]UILoweredBinding(nil), view.Bindings...)
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].Name < bindings[j].Name })
	styles := append([]UILoweredStyle(nil), view.Styles...)
	sort.Slice(styles, func(i, j int) bool { return styles[i].Name < styles[j].Name })
	accessibility := append([]UILoweredAccessibility(nil), view.Accessibility...)
	sort.Slice(accessibility, func(i, j int) bool { return accessibility[i].Name < accessibility[j].Name })

	entry := UIToolkitView{
		Name:                view.Name,
		Module:              view.Module,
		StateType:           view.StateType,
		WidgetKinds:         []string{"window", "root", "panel", "text", "label", "button", "input", "checkbox", "select", "list", "table", "dialog", "menu", "menu-item", "spacer", "divider"},
		LayoutKinds:         []string{"stack", "row", "column", "grid", "flex", "overflow-scroll"},
		StyleStates:         []string{"enabled", "disabled", "visible", "focused", "selected", "error"},
		AccessibilityFields: []string{"role", "label", "description", "focus_order", "state_metadata", "keyboard_activation"},
		EventKinds:          []string{"activate", "blur", "change", "click", "error_recovery", "focus", "input", "key", "redraw", "select", "submit", "timer"},
		StateBindingKinds:   []string{"scalar", "list", "table", "two-way-input", "deterministic-update-order"},
		Widgets:             baseToolkitWidgets(view.Name),
		Layouts:             baseToolkitLayouts(),
		Styles:              styles,
		Accessibility:       accessibility,
		Events:              make([]UIToolkitEvent, 0, len(events)),
		Commands:            make([]UIToolkitCommand, 0, len(commands)),
	}
	for i, binding := range bindings {
		entry.Widgets = append(entry.Widgets, bindingWidget(view.Name, binding, i))
	}
	for i, event := range events {
		entry.Events = append(entry.Events, UIToolkitEvent{Name: event.Name, Command: event.Command})
		entry.Widgets = append(entry.Widgets, eventWidget(view.Name, event, i))
	}
	for _, command := range commands {
		if command.StatementCount > 0 && len(command.Operations) == 0 {
			return UIToolkitView{}, fmt.Errorf("unsupported UI toolkit command operation in %s.%s", view.Name, command.Name)
		}
		for _, op := range command.Operations {
			if !supportedUIToolkitCommandOperation(op.Kind) {
				return UIToolkitView{}, fmt.Errorf("unsupported UI toolkit command operation %q in %s.%s", op.Kind, view.Name, command.Name)
			}
		}
		entry.Commands = append(entry.Commands, UIToolkitCommand{
			Name:           command.Name,
			StatementCount: command.StatementCount,
			Operations:     append([]UILoweredCommandOperation(nil), command.Operations...),
		})
	}
	return entry, nil
}

func baseToolkitWidgets(viewName string) []UIToolkitWidget {
	windowID := viewName + ".window"
	rootID := viewName + ".root"
	panelID := viewName + ".panel"
	return []UIToolkitWidget{
		{ID: windowID, Kind: "window", Layout: UIToolkitLayout{Kind: "stack", Mode: "window"}, Accessibility: "application"},
		{ID: rootID, Kind: "root", Parent: windowID, Binding: "layout.root", Layout: UIToolkitLayout{Kind: "column", Gap: 8}, Accessibility: "group"},
		{ID: panelID, Kind: "panel", Parent: rootID, Binding: "layout.content", Layout: UIToolkitLayout{Kind: "grid", Gap: 8}, Accessibility: "group"},
		{ID: viewName + ".spacer", Kind: "spacer", Parent: panelID, Binding: "layout.spacer", Layout: UIToolkitLayout{Kind: "flex", Order: 900}, Accessibility: "presentation"},
		{ID: viewName + ".divider", Kind: "divider", Parent: panelID, Binding: "layout.divider", Layout: UIToolkitLayout{Kind: "row", Order: 901}, Accessibility: "separator"},
	}
}

func baseToolkitLayouts() []UIToolkitLayout {
	return []UIToolkitLayout{
		{Kind: "stack", Mode: "root"},
		{Kind: "row", Gap: 8},
		{Kind: "column", Gap: 8},
		{Kind: "grid", Gap: 8},
		{Kind: "flex", Min: 1, Preferred: 1, Max: 1},
		{Kind: "overflow-scroll", Overflow: "scroll"},
	}
}

func bindingWidget(viewName string, binding UILoweredBinding, order int) UIToolkitWidget {
	kind := toolkitWidgetKindForBinding(binding)
	return UIToolkitWidget{
		ID:            viewName + ".binding." + binding.Name,
		Kind:          kind,
		Parent:        viewName + ".panel",
		Binding:       binding.Source,
		Layout:        UIToolkitLayout{Kind: "grid", Order: order + 1, Min: 1, Preferred: 1},
		Focusable:     kind == "input" || kind == "checkbox" || kind == "select" || kind == "list" || kind == "table",
		Accessibility: toolkitAccessibilityRole(kind),
	}
}

func eventWidget(viewName string, event UILoweredEvent, order int) UIToolkitWidget {
	return UIToolkitWidget{
		ID:            viewName + ".event." + event.Name,
		Kind:          toolkitWidgetKindForEvent(event.Name),
		Parent:        viewName + ".panel",
		Binding:       "command." + event.Command,
		Event:         event.Name,
		Command:       event.Command,
		Layout:        UIToolkitLayout{Kind: "row", Order: order + 100},
		Focusable:     true,
		Accessibility: "button",
	}
}

func toolkitWidgetKindForBinding(binding UILoweredBinding) string {
	name := strings.ToLower(binding.Name)
	typ := strings.ToLower(binding.Type)
	switch {
	case strings.Contains(name, "input"):
		return "input"
	case strings.Contains(name, "label"):
		return "label"
	case strings.Contains(name, "list") || strings.Contains(name, "items"):
		return "list"
	case strings.Contains(name, "table") || strings.Contains(name, "rows"):
		return "table"
	case strings.Contains(name, "select") || strings.Contains(name, "mode"):
		return "select"
	case typ == "bool":
		return "checkbox"
	default:
		return "text"
	}
}

func toolkitWidgetKindForEvent(eventName string) string {
	switch eventName {
	case "submit":
		return "dialog"
	case "activate":
		return "menu-item"
	default:
		return "button"
	}
}

func toolkitAccessibilityRole(kind string) string {
	switch kind {
	case "input":
		return "textbox"
	case "checkbox":
		return "checkbox"
	case "select":
		return "combobox"
	case "list":
		return "listbox"
	case "table":
		return "grid"
	case "label":
		return "label"
	default:
		return kind
	}
}

func supportedUIToolkitCommandOperation(kind string) bool {
	switch kind {
	case "state_set", "state_add", "state_sub":
		return true
	default:
		return false
	}
}
