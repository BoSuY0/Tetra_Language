package model

type Options struct {
	StackAllocationLowering    bool
	FunctionTempRegionLowering bool
	OwnedAllocDropLowering     bool
}

const UIBundleSchema = "tetra.ui.v0.4.0"

type UILoweredBundle struct {
	Schema string           `json:"schema"`
	States []UILoweredState `json:"states"`
	Views  []UILoweredView  `json:"views"`
}

type UILoweredState struct {
	Name   string                `json:"name"`
	Module string                `json:"module"`
	Fields []UILoweredStateField `json:"fields"`
}

type UILoweredStateField struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Mutable bool   `json:"mutable"`
	Const   bool   `json:"const"`
	Init    string `json:"init"`
}

type UILoweredView struct {
	Name          string                   `json:"name"`
	Module        string                   `json:"module"`
	StateType     string                   `json:"state_type"`
	Bindings      []UILoweredBinding       `json:"bindings"`
	Events        []UILoweredEvent         `json:"events"`
	Commands      []UILoweredCommand       `json:"commands"`
	Styles        []UILoweredStyle         `json:"styles"`
	Accessibility []UILoweredAccessibility `json:"accessibility"`
}

type UILoweredBinding struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Source string `json:"source"`
}

type UILoweredEvent struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

type UILoweredCommand struct {
	Name           string                      `json:"name"`
	StatementCount int                         `json:"statement_count"`
	Operations     []UILoweredCommandOperation `json:"operations,omitempty"`
}

type UILoweredCommandOperation struct {
	Kind   string `json:"kind"`
	Target string `json:"target"`
	Value  string `json:"value,omitempty"`
}

type UILoweredStyle struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type UILoweredAccessibility struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

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
