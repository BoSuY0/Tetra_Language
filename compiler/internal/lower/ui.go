package lower

import (
	"fmt"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/semantics"
)

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
	Name           string `json:"name"`
	StatementCount int    `json:"statement_count"`
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

func LowerUI(checked *semantics.CheckedProgram) (*UILoweredBundle, error) {
	if checked == nil {
		return nil, fmt.Errorf("missing checked program")
	}
	if len(checked.UIStates) == 0 && len(checked.UIViews) == 0 {
		return nil, nil
	}
	out := &UILoweredBundle{
		Schema: "tetra.ui.v1",
		States: make([]UILoweredState, 0, len(checked.UIStates)),
		Views:  make([]UILoweredView, 0, len(checked.UIViews)),
	}
	for _, state := range checked.UIStates {
		if state.Decl == nil {
			continue
		}
		entry := UILoweredState{
			Name:   state.Name,
			Module: state.Module,
			Fields: make([]UILoweredStateField, 0, len(state.Decl.Fields)),
		}
		for _, field := range state.Decl.Fields {
			entry.Fields = append(entry.Fields, UILoweredStateField{
				Name:    field.Name,
				Type:    field.Type.Name,
				Mutable: field.Mutable,
				Const:   field.Const,
				Init:    uiExprSummary(field.Init),
			})
		}
		out.States = append(out.States, entry)
	}
	for _, view := range checked.UIViews {
		if view.Decl == nil {
			continue
		}
		entry := UILoweredView{
			Name:          view.Name,
			Module:        view.Module,
			StateType:     view.Decl.StateName.Name,
			Bindings:      make([]UILoweredBinding, 0, len(view.Decl.Bindings)),
			Events:        make([]UILoweredEvent, 0, len(view.Decl.Events)),
			Commands:      make([]UILoweredCommand, 0, len(view.Decl.Commands)),
			Styles:        make([]UILoweredStyle, 0, len(view.Decl.Styles)),
			Accessibility: make([]UILoweredAccessibility, 0, len(view.Decl.Accessibility)),
		}
		for _, binding := range view.Decl.Bindings {
			entry.Bindings = append(entry.Bindings, UILoweredBinding{
				Name:   binding.Name,
				Type:   binding.Type.Name,
				Source: uiExprSummary(binding.Value),
			})
		}
		for _, event := range view.Decl.Events {
			entry.Events = append(entry.Events, UILoweredEvent{Name: event.Name, Command: event.Command})
		}
		for _, command := range view.Decl.Commands {
			entry.Commands = append(entry.Commands, UILoweredCommand{
				Name:           command.Name,
				StatementCount: len(command.Body),
			})
		}
		for _, style := range view.Decl.Styles {
			entry.Styles = append(entry.Styles, UILoweredStyle{
				Name:  style.Name,
				Type:  style.Type.Name,
				Value: uiExprSummary(style.Value),
			})
		}
		for _, a11y := range view.Decl.Accessibility {
			entry.Accessibility = append(entry.Accessibility, UILoweredAccessibility{
				Name:  a11y.Name,
				Type:  a11y.Type.Name,
				Value: uiExprSummary(a11y.Value),
			})
		}
		out.Views = append(out.Views, entry)
	}
	return out, nil
}

func uiExprSummary(expr frontend.Expr) string {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return fmt.Sprintf("%d", e.Value)
	case *frontend.BoolLitExpr:
		if e.Value {
			return "true"
		}
		return "false"
	case *frontend.StringLitExpr:
		return `"` + string(e.Value) + `"`
	case *frontend.NoneLitExpr:
		return "none"
	case *frontend.IdentExpr:
		return e.Name
	case *frontend.FieldAccessExpr:
		return uiExprSummary(e.Base) + "." + e.Field
	case *frontend.BinaryExpr:
		return uiExprSummary(e.Left) + " " + frontend.TokenName(e.Op) + " " + uiExprSummary(e.Right)
	case *frontend.UnaryExpr:
		return frontend.TokenName(e.Op) + " " + uiExprSummary(e.X)
	case *frontend.CallExpr:
		return e.Name + "(...)"
	case *frontend.StructLitExpr:
		return e.Type.Name + "{...}"
	case *frontend.IndexExpr:
		return uiExprSummary(e.Base) + "[...]"
	default:
		if expr == nil {
			return ""
		}
		return "<expr>"
	}
}
