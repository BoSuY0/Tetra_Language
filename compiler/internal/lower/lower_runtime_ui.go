package lower

import (
	"fmt"
	"sort"
	"strings"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	lowermodel "tetra_language/compiler/internal/lower/model"
	"tetra_language/compiler/internal/semantics"
	"tetra_language/compiler/target"
)

// ---- atomic.go ----

type atomicBuiltinSpec struct {
	Op        target.AtomicOp
	Order     target.MemoryOrder
	WidthBits int
	Pointer   bool
	Fence     bool
}

type atomicBuiltinOrder struct {
	Suffix string
	Order  target.MemoryOrder
}

type atomicBuiltinWidth struct {
	Suffix    string
	WidthBits int
	Pointer   bool
}

var atomicBuiltinOrders = []atomicBuiltinOrder{
	{Suffix: "relaxed", Order: target.MemoryOrderRelaxed},
	{Suffix: "acquire", Order: target.MemoryOrderAcquire},
	{Suffix: "release", Order: target.MemoryOrderRelease},
	{Suffix: "acq_rel", Order: target.MemoryOrderAcqRel},
	{Suffix: "seq_cst", Order: target.MemoryOrderSeqCst},
}

var atomicBuiltinWidths = []atomicBuiltinWidth{
	{Suffix: "u8", WidthBits: 8},
	{Suffix: "u16", WidthBits: 16},
	{Suffix: "i32", WidthBits: 32},
	{Suffix: "i64", WidthBits: 64},
	{Suffix: "ptr", Pointer: true},
}

var atomicBuiltinOps = []struct {
	Prefix string
	Op     target.AtomicOp
}{
	{Prefix: "compare_exchange_weak_", Op: target.AtomicCompareExchangeWeak},
	{Prefix: "compare_exchange_", Op: target.AtomicCompareExchange},
	{Prefix: "fetch_add_", Op: target.AtomicFetchAdd},
	{Prefix: "fetch_sub_", Op: target.AtomicFetchSub},
	{Prefix: "fetch_and_", Op: target.AtomicFetchAnd},
	{Prefix: "fetch_or_", Op: target.AtomicFetchOr},
	{Prefix: "fetch_xor_", Op: target.AtomicFetchXor},
	{Prefix: "exchange_", Op: target.AtomicExchange},
	{Prefix: "store_", Op: target.AtomicStore},
	{Prefix: "load_", Op: target.AtomicLoad},
}

func parseAtomicBuiltinName(name string) (atomicBuiltinSpec, bool, error) {
	const prefix = "core.atomic_"
	if !strings.HasPrefix(name, prefix) {
		return atomicBuiltinSpec{}, false, nil
	}
	rest := strings.TrimPrefix(name, prefix)
	if strings.HasPrefix(rest, "fence_") {
		orderSuffix := strings.TrimPrefix(rest, "fence_")
		order, ok := parseAtomicBuiltinOrder(orderSuffix)
		if !ok {
			return atomicBuiltinSpec{}, true, fmt.Errorf(
				"unsupported atomic fence memory order suffix %q",
				orderSuffix,
			)
		}
		return atomicBuiltinSpec{Op: target.AtomicFence, Order: order, Fence: true}, true, nil
	}
	for _, op := range atomicBuiltinOps {
		if !strings.HasPrefix(rest, op.Prefix) {
			continue
		}
		tail := strings.TrimPrefix(rest, op.Prefix)
		for _, width := range atomicBuiltinWidths {
			widthPrefix := width.Suffix + "_"
			if !strings.HasPrefix(tail, widthPrefix) {
				continue
			}
			orderSuffix := strings.TrimPrefix(tail, widthPrefix)
			order, ok := parseAtomicBuiltinOrder(orderSuffix)
			if !ok {
				return atomicBuiltinSpec{}, true, fmt.Errorf(
					"unsupported atomic memory order suffix %q",
					orderSuffix,
				)
			}
			if !atomicBuiltinOrderAllowed(op.Op, order) {
				return atomicBuiltinSpec{}, true, fmt.Errorf(
					"atomic %s does not support memory order %s",
					op.Op,
					order,
				)
			}
			return atomicBuiltinSpec{
				Op:        op.Op,
				Order:     order,
				WidthBits: width.WidthBits,
				Pointer:   width.Pointer,
			}, true, nil
		}
		return atomicBuiltinSpec{}, true, fmt.Errorf(
			"unsupported atomic value width in builtin %q",
			name,
		)
	}
	return atomicBuiltinSpec{}, true, fmt.Errorf("unsupported atomic builtin %q", name)
}

func parseAtomicBuiltinOrder(suffix string) (target.MemoryOrder, bool) {
	for _, order := range atomicBuiltinOrders {
		if suffix == order.Suffix {
			return order.Order, true
		}
	}
	return target.MemoryOrderUnknown, false
}

func atomicBuiltinOrderAllowed(op target.AtomicOp, order target.MemoryOrder) bool {
	switch op {
	case target.AtomicLoad:
		return order == target.MemoryOrderRelaxed || order == target.MemoryOrderAcquire ||
			order == target.MemoryOrderSeqCst
	case target.AtomicStore:
		return order == target.MemoryOrderRelaxed || order == target.MemoryOrderRelease ||
			order == target.MemoryOrderSeqCst
	case target.AtomicExchange, target.AtomicCompareExchange, target.AtomicCompareExchangeWeak,
		target.AtomicFetchAdd, target.AtomicFetchSub, target.AtomicFetchAnd, target.AtomicFetchOr, target.AtomicFetchXor:
		return order == target.MemoryOrderRelaxed || order == target.MemoryOrderAcquire ||
			order == target.MemoryOrderRelease || order == target.MemoryOrderAcqRel || order == target.MemoryOrderSeqCst
	default:
		return false
	}
}

func (l *lowerer) lowerAtomicBuiltinCall(e *frontend.CallExpr) (int, bool, error) {
	spec, ok, err := parseAtomicBuiltinName(e.Name)
	if !ok || err != nil {
		return 0, ok, err
	}
	if spec.Fence {
		if len(e.Args) != 1 {
			return 0, true, fmt.Errorf(
				"%s: atomic_fence_%s expects 1 memory capability argument",
				frontend.FormatPos(e.At),
				spec.Order,
			)
		}
		slots, err := l.lowerExpr(e.Args[0])
		if err != nil {
			return 0, true, err
		}
		if slots != 1 {
			return 0, true, fmt.Errorf(
				"%s: atomic_fence_%s expects a 1-slot memory capability",
				frontend.FormatPos(e.Args[0].Pos()),
				spec.Order,
			)
		}
		discard := l.ensureDiscardLocal()
		l.emit(ir.IRInstr{Kind: ir.IRStoreLocal, Local: discard, Pos: e.At})
		kind, err := atomicFenceKindForOrder(spec.Order)
		if err != nil {
			return 0, true, err
		}
		l.emit(ir.IRInstr{Kind: kind, Pos: e.At})
		l.emit(ir.IRInstr{Kind: ir.IRConstI32, Imm: 0, Pos: e.At})
		return 1, true, nil
	}

	expectedArgs := 0
	switch spec.Op {
	case target.AtomicLoad:
		expectedArgs = 2
	case target.AtomicStore, target.AtomicExchange,
		target.AtomicFetchAdd, target.AtomicFetchSub, target.AtomicFetchAnd, target.AtomicFetchOr, target.AtomicFetchXor:
		expectedArgs = 3
	case target.AtomicCompareExchange, target.AtomicCompareExchangeWeak:
		expectedArgs = 4
	default:
		return 0, true, fmt.Errorf(
			"%s: unsupported atomic op %s",
			frontend.FormatPos(e.At),
			spec.Op,
		)
	}
	if len(e.Args) != expectedArgs {
		return 0, true, fmt.Errorf(
			"%s: atomic %s expects %d arguments",
			frontend.FormatPos(e.At),
			spec.Op,
			expectedArgs,
		)
	}
	total := 0
	for _, arg := range e.Args {
		slots, err := l.lowerExpr(arg)
		if err != nil {
			return 0, true, err
		}
		total += slots
	}
	if total != expectedArgs {
		return 0, true, fmt.Errorf(
			"%s: atomic %s expects %d argument slots",
			frontend.FormatPos(e.At),
			spec.Op,
			expectedArgs,
		)
	}
	var kind ir.IRInstrKind
	if spec.Pointer {
		kind, err = atomicPointerKindForOp(spec.Op)
	} else {
		kind, err = atomicValueKindForOpWidth(spec.Op, spec.WidthBits)
	}
	if err != nil {
		return 0, true, err
	}
	l.emit(ir.IRInstr{Kind: kind, Pos: e.At})
	return 1, true, nil
}

func atomicValueKindForOpWidth(op target.AtomicOp, widthBits int) (ir.IRInstrKind, error) {
	switch op {
	case target.AtomicFence:
		return 0, fmt.Errorf("atomic fence lowering requires atomicFenceKindForOrder")
	}

	switch widthBits {
	case 8:
		return atomicValueKindForOp(op, widthBits,
			ir.IRAtomicLoadI8,
			ir.IRAtomicStoreI8,
			ir.IRAtomicExchangeI8,
			ir.IRAtomicCompareExchangeI8,
			ir.IRAtomicFetchAddI8,
			ir.IRAtomicFetchSubI8,
			ir.IRAtomicFetchAndI8,
			ir.IRAtomicFetchOrI8,
			ir.IRAtomicFetchXorI8,
		)
	case 16:
		return atomicValueKindForOp(op, widthBits,
			ir.IRAtomicLoadI16,
			ir.IRAtomicStoreI16,
			ir.IRAtomicExchangeI16,
			ir.IRAtomicCompareExchangeI16,
			ir.IRAtomicFetchAddI16,
			ir.IRAtomicFetchSubI16,
			ir.IRAtomicFetchAndI16,
			ir.IRAtomicFetchOrI16,
			ir.IRAtomicFetchXorI16,
		)
	case 32:
		return atomicValueKindForOp(op, widthBits,
			ir.IRAtomicLoadI32,
			ir.IRAtomicStoreI32,
			ir.IRAtomicExchangeI32,
			ir.IRAtomicCompareExchangeI32,
			ir.IRAtomicFetchAddI32,
			ir.IRAtomicFetchSubI32,
			ir.IRAtomicFetchAndI32,
			ir.IRAtomicFetchOrI32,
			ir.IRAtomicFetchXorI32,
		)
	case 64:
		return atomicValueKindForOp(op, widthBits,
			ir.IRAtomicLoadI64,
			ir.IRAtomicStoreI64,
			ir.IRAtomicExchangeI64,
			ir.IRAtomicCompareExchangeI64,
			ir.IRAtomicFetchAddI64,
			ir.IRAtomicFetchSubI64,
			ir.IRAtomicFetchAndI64,
			ir.IRAtomicFetchOrI64,
			ir.IRAtomicFetchXorI64,
		)
	default:
		return 0, fmt.Errorf("unsupported atomic width %d bits", widthBits)
	}
}

func atomicPointerKindForOp(op target.AtomicOp) (ir.IRInstrKind, error) {
	switch op {
	case target.AtomicFence:
		return 0, fmt.Errorf("atomic fence lowering requires atomicFenceKindForOrder")
	case target.AtomicLoad:
		return ir.IRAtomicLoadPtr, nil
	case target.AtomicStore:
		return ir.IRAtomicStorePtr, nil
	case target.AtomicExchange:
		return ir.IRAtomicExchangePtr, nil
	case target.AtomicCompareExchange, target.AtomicCompareExchangeWeak:
		return ir.IRAtomicCompareExchangePtr, nil
	case target.AtomicFetchAdd:
		return ir.IRAtomicFetchAddPtr, nil
	case target.AtomicFetchSub:
		return ir.IRAtomicFetchSubPtr, nil
	case target.AtomicFetchAnd:
		return ir.IRAtomicFetchAndPtr, nil
	case target.AtomicFetchOr:
		return ir.IRAtomicFetchOrPtr, nil
	case target.AtomicFetchXor:
		return ir.IRAtomicFetchXorPtr, nil
	default:
		return 0, fmt.Errorf("unsupported atomic op %s for pointer-sized value", op)
	}
}

func atomicValueKindForOp(
	op target.AtomicOp,
	widthBits int,
	load ir.IRInstrKind,
	store ir.IRInstrKind,
	exchange ir.IRInstrKind,
	compareExchange ir.IRInstrKind,
	fetchAdd ir.IRInstrKind,
	fetchSub ir.IRInstrKind,
	fetchAnd ir.IRInstrKind,
	fetchOr ir.IRInstrKind,
	fetchXor ir.IRInstrKind,
) (ir.IRInstrKind, error) {
	switch op {
	case target.AtomicLoad:
		return load, nil
	case target.AtomicStore:
		return store, nil
	case target.AtomicExchange:
		return exchange, nil
	case target.AtomicCompareExchange, target.AtomicCompareExchangeWeak:
		return compareExchange, nil
	case target.AtomicFetchAdd:
		return fetchAdd, nil
	case target.AtomicFetchSub:
		return fetchSub, nil
	case target.AtomicFetchAnd:
		return fetchAnd, nil
	case target.AtomicFetchOr:
		return fetchOr, nil
	case target.AtomicFetchXor:
		return fetchXor, nil
	default:
		return 0, fmt.Errorf("unsupported atomic op %s for %d-bit value", op, widthBits)
	}
}

func atomicFenceKindForOrder(order target.MemoryOrder) (ir.IRInstrKind, error) {
	switch order {
	case target.MemoryOrderRelaxed:
		return ir.IRAtomicFenceRelaxed, nil
	case target.MemoryOrderAcquire:
		return ir.IRAtomicFenceAcquire, nil
	case target.MemoryOrderRelease:
		return ir.IRAtomicFenceRelease, nil
	case target.MemoryOrderAcqRel:
		return ir.IRAtomicFenceAcqRel, nil
	case target.MemoryOrderSeqCst:
		return ir.IRAtomicFenceSeqCst, nil
	default:
		return 0, fmt.Errorf("unsupported atomic fence memory order %s", order)
	}
}

// ---- ui.go ----

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
			entry.Events = append(
				entry.Events,
				UILoweredEvent{Name: event.Name, Command: event.Command},
			)
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

func uiCompoundStateDeltaOperation(
	target string,
	assign *frontend.AssignStmt,
) (UILoweredCommandOperation, bool) {
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

// ---- ui_toolkit.go ----

const UIToolkitSchema = lowermodel.UIToolkitSchema

type UIToolkitBundle = lowermodel.UIToolkitBundle
type UIToolkitState = lowermodel.UIToolkitState
type UIToolkitView = lowermodel.UIToolkitView
type UIToolkitWidget = lowermodel.UIToolkitWidget
type UIToolkitLayout = lowermodel.UIToolkitLayout
type UIToolkitEvent = lowermodel.UIToolkitEvent
type UIToolkitCommand = lowermodel.UIToolkitCommand

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
		out.States = append(
			out.States,
			UIToolkitState{Name: state.Name, Module: state.Module, Fields: fields},
		)
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
	sort.Slice(
		accessibility,
		func(i, j int) bool { return accessibility[i].Name < accessibility[j].Name },
	)

	entry := UIToolkitView{
		Name:      view.Name,
		Module:    view.Module,
		StateType: view.StateType,
		WidgetKinds: []string{
			"window",
			"root",
			"panel",
			"text",
			"label",
			"button",
			"input",
			"checkbox",
			"select",
			"list",
			"table",
			"dialog",
			"menu",
			"menu-item",
			"spacer",
			"divider",
		},
		LayoutKinds: []string{"stack", "row", "column", "grid", "flex", "overflow-scroll"},
		StyleStates: []string{
			"enabled",
			"disabled",
			"visible",
			"focused",
			"selected",
			"error",
		},
		AccessibilityFields: []string{
			"role",
			"label",
			"description",
			"focus_order",
			"state_metadata",
			"keyboard_activation",
		},
		EventKinds: []string{
			"activate",
			"blur",
			"change",
			"click",
			"error_recovery",
			"focus",
			"input",
			"key",
			"redraw",
			"select",
			"submit",
			"timer",
		},
		StateBindingKinds: []string{
			"scalar",
			"list",
			"table",
			"two-way-input",
			"deterministic-update-order",
		},
		Widgets:       baseToolkitWidgets(view.Name),
		Layouts:       baseToolkitLayouts(),
		Styles:        styles,
		Accessibility: accessibility,
		Events:        make([]UIToolkitEvent, 0, len(events)),
		Commands:      make([]UIToolkitCommand, 0, len(commands)),
	}
	for i, binding := range bindings {
		entry.Widgets = append(entry.Widgets, bindingWidget(view.Name, binding, i))
	}
	for i, event := range events {
		entry.Events = append(
			entry.Events,
			UIToolkitEvent{Name: event.Name, Command: event.Command},
		)
		entry.Widgets = append(entry.Widgets, eventWidget(view.Name, event, i))
	}
	for _, command := range commands {
		if command.StatementCount > 0 && len(command.Operations) == 0 {
			return UIToolkitView{}, fmt.Errorf(
				"unsupported UI toolkit command operation in %s.%s",
				view.Name,
				command.Name,
			)
		}
		for _, op := range command.Operations {
			if !supportedUIToolkitCommandOperation(op.Kind) {
				return UIToolkitView{}, fmt.Errorf(
					"unsupported UI toolkit command operation %q in %s.%s",
					op.Kind,
					view.Name,
					command.Name,
				)
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
		{
			ID:            windowID,
			Kind:          "window",
			Layout:        UIToolkitLayout{Kind: "stack", Mode: "window"},
			Accessibility: "application",
		},
		{
			ID:            rootID,
			Kind:          "root",
			Parent:        windowID,
			Binding:       "layout.root",
			Layout:        UIToolkitLayout{Kind: "column", Gap: 8},
			Accessibility: "group",
		},
		{
			ID:            panelID,
			Kind:          "panel",
			Parent:        rootID,
			Binding:       "layout.content",
			Layout:        UIToolkitLayout{Kind: "grid", Gap: 8},
			Accessibility: "group",
		},
		{
			ID:            viewName + ".spacer",
			Kind:          "spacer",
			Parent:        panelID,
			Binding:       "layout.spacer",
			Layout:        UIToolkitLayout{Kind: "flex", Order: 900},
			Accessibility: "presentation",
		},
		{
			ID:            viewName + ".divider",
			Kind:          "divider",
			Parent:        panelID,
			Binding:       "layout.divider",
			Layout:        UIToolkitLayout{Kind: "row", Order: 901},
			Accessibility: "separator",
		},
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
		ID:      viewName + ".binding." + binding.Name,
		Kind:    kind,
		Parent:  viewName + ".panel",
		Binding: binding.Source,
		Layout:  UIToolkitLayout{Kind: "grid", Order: order + 1, Min: 1, Preferred: 1},
		Focusable: kind == "input" || kind == "checkbox" || kind == "select" || kind == "list" ||
			kind == "table",
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
