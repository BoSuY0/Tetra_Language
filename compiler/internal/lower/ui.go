package lower

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
	lowermodel "tetra_language/compiler/internal/lower/model"
	"tetra_language/compiler/internal/semantics"
)

const UIBundleSchema = lowermodel.UIBundleSchema

type UILoweredBundle = lowermodel.UILoweredBundle
type UILoweredState = lowermodel.UILoweredState
type UILoweredStateField = lowermodel.UILoweredStateField
type UILoweredView = lowermodel.UILoweredView
type UILoweredBinding = lowermodel.UILoweredBinding
type UILoweredEvent = lowermodel.UILoweredEvent
type UILoweredCommand = lowermodel.UILoweredCommand
type UILoweredCommandOperation = lowermodel.UILoweredCommandOperation
type UILoweredStyle = lowermodel.UILoweredStyle
type UILoweredAccessibility = lowermodel.UILoweredAccessibility

func LowerUI(checked *semantics.CheckedProgram) (*UILoweredBundle, error) {
	if checked == nil {
		return nil, fmt.Errorf("missing checked program")
	}
	if len(checked.UIStates) == 0 && len(checked.UIViews) == 0 {
		return nil, nil
	}
	out := &UILoweredBundle{
		Schema: UIBundleSchema,
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
				Operations:     uiCommandOperations(command.Body),
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

func uiCommandOperations(stmts []frontend.Stmt) []UILoweredCommandOperation {
	ops := make([]UILoweredCommandOperation, 0, len(stmts))
	for _, stmt := range stmts {
		assign, ok := stmt.(*frontend.AssignStmt)
		if !ok {
			continue
		}
		target := uiExprSummary(assign.Target)
		if !strings.HasPrefix(target, "state.") {
			continue
		}
		if delta, ok := uiCompoundStateDeltaOperation(target, assign); ok {
			ops = append(ops, delta)
			continue
		}
		if delta, ok := uiStateDeltaOperation(target, assign.Value); ok && assign.Op == 0 {
			ops = append(ops, delta)
			continue
		}
		ops = append(ops, UILoweredCommandOperation{
			Kind:   "state_set",
			Target: target,
			Value:  uiExprSummary(assign.Value),
		})
	}
	return ops
}

func uiCompoundStateDeltaOperation(target string, assign *frontend.AssignStmt) (UILoweredCommandOperation, bool) {
	if assign == nil || assign.CompoundValue == nil {
		return UILoweredCommandOperation{}, false
	}
	kind := ""
	switch assign.Op {
	case frontend.TokenPlus:
		kind = "state_add"
	case frontend.TokenMinus:
		kind = "state_sub"
	default:
		return UILoweredCommandOperation{}, false
	}
	number, ok := assign.CompoundValue.(*frontend.NumberExpr)
	if !ok {
		return UILoweredCommandOperation{}, false
	}
	return UILoweredCommandOperation{
		Kind:   kind,
		Target: target,
		Value:  fmt.Sprintf("%d", number.Value),
	}, true
}

func uiStateDeltaOperation(target string, expr frontend.Expr) (UILoweredCommandOperation, bool) {
	binary, ok := expr.(*frontend.BinaryExpr)
	if !ok || (binary.Op != frontend.TokenPlus && binary.Op != frontend.TokenMinus) {
		return UILoweredCommandOperation{}, false
	}
	if uiExprSummary(binary.Left) != target {
		return UILoweredCommandOperation{}, false
	}
	number, ok := binary.Right.(*frontend.NumberExpr)
	if !ok {
		return UILoweredCommandOperation{}, false
	}
	kind := "state_add"
	if binary.Op == frontend.TokenMinus {
		kind = "state_sub"
	}
	return UILoweredCommandOperation{
		Kind:   kind,
		Target: target,
		Value:  fmt.Sprintf("%d", number.Value),
	}, true
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
