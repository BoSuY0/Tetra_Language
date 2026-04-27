package semantics

import (
	"fmt"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/module"
)

func stateAsStructDecl(state *frontend.StateDecl) *frontend.StructDecl {
	if state == nil {
		return &frontend.StructDecl{}
	}
	fields := make([]frontend.FieldDecl, 0, len(state.Fields))
	for _, field := range state.Fields {
		fields = append(fields, frontend.FieldDecl{
			At:   field.At,
			Name: field.Name,
			Type: field.Type,
		})
	}
	return &frontend.StructDecl{
		At:     state.At,
		Name:   state.Name,
		Fields: fields,
	}
}

func checkUIDecls(world *module.World, checked *CheckedProgram, types map[string]*TypeInfo) error {
	if world == nil || checked == nil {
		return nil
	}

	importsByModule := make(map[string]map[string]string, len(world.Files))
	for _, file := range world.Files {
		imports, err := collectImportAliases(file)
		if err != nil {
			return err
		}
		importsByModule[file.Module] = imports
	}

	stateByName := make(map[string]CheckedUIState, len(checked.UIStates))
	stateConstFields := make(map[string]map[string]bool, len(checked.UIStates))
	for i := range checked.UIStates {
		state := checked.UIStates[i]
		stateByName[state.Name] = state
		stateConstFields[state.Name] = make(map[string]bool)
	}

	emptyGlobals := map[string]GlobalInfo{}
	for i := range checked.UIStates {
		state := &checked.UIStates[i]
		imports := importsByModule[state.Module]
		initLocals := make(map[string]LocalInfo, len(state.Decl.Fields))
		slot := 0
		for j := range state.Decl.Fields {
			field := &state.Decl.Fields[j]
			resolved, err := resolveTypeName(&field.Type, state.Module, imports)
			if err != nil {
				return err
			}
			field.Type.Name = resolved
			info, err := ensureTypeInfo(resolved, types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(field.At), err)
			}
			if field.Init == nil {
				return fmt.Errorf("%s: state field '%s' requires an initializer", frontend.FormatPos(field.At), field.Name)
			}
			exprType, _, err := checkExprWithEffects(
				field.Init,
				initLocals,
				emptyGlobals,
				checked.FuncSigs,
				types,
				state.Module,
				imports,
				newRegionState(nil),
				newEffectContext(state.Name, nil, nil, true),
				nil,
			)
			if err != nil {
				return fmt.Errorf("%s: state '%s' field '%s': %v", frontend.FormatPos(field.At), state.Name, field.Name, err)
			}
			if !typesCompatibleWithNullPtr(resolved, exprType, field.Init) {
				return fmt.Errorf("%s: state '%s' field '%s' type mismatch: expected '%s', got '%s'", frontend.FormatPos(field.At), state.Name, field.Name, resolved, exprType)
			}
			initLocals[field.Name] = LocalInfo{
				Base:      slot,
				SlotCount: info.SlotCount,
				TypeName:  resolved,
				Mutable:   field.Mutable,
				Const:     field.Const,
			}
			slot += info.SlotCount
			stateConstFields[state.Name][field.Name] = field.Const || !field.Mutable
		}
	}

	seenViews := make(map[string]struct{})
	for _, file := range world.Files {
		imports := importsByModule[file.Module]
		for i := range file.Views {
			view := file.Views[i]
			fullName := qualifyName(file.Module, view.Name)
			if _, exists := seenViews[fullName]; exists {
				return fmt.Errorf("duplicate view '%s'", fullName)
			}
			seenViews[fullName] = struct{}{}

			stateType, err := resolveTypeName(&view.StateName, file.Module, imports)
			if err != nil {
				return err
			}
			view.StateName.Name = stateType
			_, ok := stateByName[stateType]
			if !ok {
				return fmt.Errorf("%s: view '%s' references unknown state '%s'", frontend.FormatPos(view.At), fullName, stateType)
			}
			stateInfo, err := ensureTypeInfo(stateType, types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(view.At), err)
			}

			bindingNames := map[string]struct{}{}
			eventNames := map[string]struct{}{}
			styleNames := map[string]struct{}{}
			a11yNames := map[string]struct{}{}
			commandNames := map[string]struct{}{}

			baseLocals := map[string]LocalInfo{
				"state": {
					Base:      0,
					SlotCount: stateInfo.SlotCount,
					TypeName:  stateType,
					Mutable:   true,
				},
			}
			baseSlot := stateInfo.SlotCount
			baseState := newRegionState(nil)
			baseEffects := newEffectContext(fullName, nil, nil, true)
			for j := range view.Bindings {
				binding := &view.Bindings[j]
				if _, exists := bindingNames[binding.Name]; exists {
					return fmt.Errorf("%s: duplicate binding '%s'", frontend.FormatPos(binding.At), binding.Name)
				}
				bindingNames[binding.Name] = struct{}{}
				resolved, err := resolveTypeName(&binding.Type, file.Module, imports)
				if err != nil {
					return err
				}
				binding.Type.Name = resolved
				info, err := ensureTypeInfo(resolved, types)
				if err != nil {
					return fmt.Errorf("%s: %v", frontend.FormatPos(binding.At), err)
				}
				exprType, _, err := checkExprWithEffects(
					binding.Value,
					baseLocals,
					emptyGlobals,
					checked.FuncSigs,
					types,
					file.Module,
					imports,
					baseState,
					baseEffects,
					nil,
				)
				if err != nil {
					return fmt.Errorf("%s: binding '%s': %v", frontend.FormatPos(binding.At), binding.Name, err)
				}
				if !typesCompatibleWithNullPtr(resolved, exprType, binding.Value) {
					return fmt.Errorf("%s: binding '%s' type mismatch: expected '%s', got '%s'", frontend.FormatPos(binding.At), binding.Name, resolved, exprType)
				}
				baseLocals[binding.Name] = LocalInfo{
					Base:      baseSlot,
					SlotCount: info.SlotCount,
					TypeName:  resolved,
					Mutable:   false,
					Const:     true,
				}
				baseSlot += info.SlotCount
			}

			for j := range view.Commands {
				cmd := &view.Commands[j]
				if _, exists := commandNames[cmd.Name]; exists {
					return fmt.Errorf("%s: duplicate command '%s'", frontend.FormatPos(cmd.At), cmd.Name)
				}
				commandNames[cmd.Name] = struct{}{}
			}
			for j := range view.Events {
				event := &view.Events[j]
				if _, exists := eventNames[event.Name]; exists {
					return fmt.Errorf("%s: duplicate event '%s'", frontend.FormatPos(event.At), event.Name)
				}
				eventNames[event.Name] = struct{}{}
				if _, exists := commandNames[event.Command]; !exists {
					return fmt.Errorf("%s: event '%s' references unknown command '%s'", frontend.FormatPos(event.At), event.Name, event.Command)
				}
			}

			for j := range view.Styles {
				style := &view.Styles[j]
				if _, exists := styleNames[style.Name]; exists {
					return fmt.Errorf("%s: duplicate style '%s'", frontend.FormatPos(style.At), style.Name)
				}
				styleNames[style.Name] = struct{}{}
				resolved, err := resolveTypeName(&style.Type, file.Module, imports)
				if err != nil {
					return err
				}
				style.Type.Name = resolved
				if !isUIScalarType(resolved) {
					return fmt.Errorf("%s: style '%s' uses unsupported type '%s' (allowed: i32, bool, str)", frontend.FormatPos(style.At), style.Name, resolved)
				}
				exprType, _, err := checkExprWithEffects(
					style.Value,
					baseLocals,
					emptyGlobals,
					checked.FuncSigs,
					types,
					file.Module,
					imports,
					newRegionState(nil),
					baseEffects,
					nil,
				)
				if err != nil {
					return fmt.Errorf("%s: style '%s': %v", frontend.FormatPos(style.At), style.Name, err)
				}
				if !typesCompatibleWithNullPtr(resolved, exprType, style.Value) {
					return fmt.Errorf("%s: style '%s' type mismatch: expected '%s', got '%s'", frontend.FormatPos(style.At), style.Name, resolved, exprType)
				}
			}

			for j := range view.Accessibility {
				entry := &view.Accessibility[j]
				if _, exists := a11yNames[entry.Name]; exists {
					return fmt.Errorf("%s: duplicate accessibility key '%s'", frontend.FormatPos(entry.At), entry.Name)
				}
				a11yNames[entry.Name] = struct{}{}
				resolved, err := resolveTypeName(&entry.Type, file.Module, imports)
				if err != nil {
					return err
				}
				entry.Type.Name = resolved
				if !isUIScalarType(resolved) {
					return fmt.Errorf("%s: accessibility '%s' uses unsupported type '%s' (allowed: i32, bool, str)", frontend.FormatPos(entry.At), entry.Name, resolved)
				}
				exprType, _, err := checkExprWithEffects(
					entry.Value,
					baseLocals,
					emptyGlobals,
					checked.FuncSigs,
					types,
					file.Module,
					imports,
					newRegionState(nil),
					baseEffects,
					nil,
				)
				if err != nil {
					return fmt.Errorf("%s: accessibility '%s': %v", frontend.FormatPos(entry.At), entry.Name, err)
				}
				if !typesCompatibleWithNullPtr(resolved, exprType, entry.Value) {
					return fmt.Errorf("%s: accessibility '%s' type mismatch: expected '%s', got '%s'", frontend.FormatPos(entry.At), entry.Name, resolved, exprType)
				}
			}

			for j := range view.Commands {
				cmd := &view.Commands[j]
				if err := validateViewCommandStmts(cmd.Body, stateConstFields[stateType]); err != nil {
					return fmt.Errorf("%s: command '%s': %v", frontend.FormatPos(cmd.At), cmd.Name, err)
				}
				cmdLocals := cloneUILocals(baseLocals)
				slotIndex := baseSlot
				scopes := newScopeInfo()
				if err := collectLocals(
					cmd.Body,
					cmdLocals,
					&slotIndex,
					checked.FuncSigs,
					types,
					file.Module,
					imports,
					scopes,
					emptyGlobals,
				); err != nil {
					return fmt.Errorf("%s: command '%s': %v", frontend.FormatPos(cmd.At), cmd.Name, err)
				}
				cmdState := newRegionState(scopes)
				cmdEffects := newEffectContext(fullName+".command."+cmd.Name, nil, nil, false)
				if err := checkStmts(
					cmd.Body,
					cmdLocals,
					emptyGlobals,
					checked.FuncSigs,
					types,
					file.Module,
					imports,
					"i32",
					nil,
					nil,
					cmdState,
					cmdEffects,
					&functionAnalysisState{},
				); err != nil {
					return fmt.Errorf("%s: command '%s': %v", frontend.FormatPos(cmd.At), cmd.Name, err)
				}
			}

			checked.UIViews = append(checked.UIViews, CheckedUIView{
				Name:   fullName,
				Module: file.Module,
				Decl:   view,
			})
		}
	}

	return nil
}

func isUIScalarType(typeName string) bool {
	switch typeName {
	case "i32", "bool", "str":
		return true
	default:
		return false
	}
}

func cloneUILocals(src map[string]LocalInfo) map[string]LocalInfo {
	out := make(map[string]LocalInfo, len(src))
	for name, info := range src {
		out[name] = info
	}
	return out
}

func validateViewCommandStmts(stmts []frontend.Stmt, stateConstFields map[string]bool) error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.ReturnStmt:
			return fmt.Errorf("%s: return is not allowed inside view commands", frontend.FormatPos(s.At))
		case *frontend.ThrowStmt:
			return fmt.Errorf("%s: throw is not allowed inside view commands", frontend.FormatPos(s.At))
		case *frontend.AssignStmt:
			if field, ok := assignedStateField(s.Target); ok {
				if field == "" {
					return fmt.Errorf("%s: assigning to 'state' directly is not allowed in view commands", frontend.FormatPos(s.At))
				}
				if stateConstFields[field] {
					return fmt.Errorf("%s: cannot assign to immutable state field '%s'", frontend.FormatPos(s.At), field)
				}
			}
		case *frontend.IfStmt:
			if err := validateViewCommandStmts(s.Then, stateConstFields); err != nil {
				return err
			}
			if err := validateViewCommandStmts(s.Else, stateConstFields); err != nil {
				return err
			}
		case *frontend.IfLetStmt:
			if err := validateViewCommandStmts(s.Then, stateConstFields); err != nil {
				return err
			}
			if err := validateViewCommandStmts(s.Else, stateConstFields); err != nil {
				return err
			}
		case *frontend.WhileStmt:
			if err := validateViewCommandStmts(s.Body, stateConstFields); err != nil {
				return err
			}
		case *frontend.ForRangeStmt:
			if err := validateViewCommandStmts(s.Body, stateConstFields); err != nil {
				return err
			}
		case *frontend.MatchStmt:
			for _, c := range s.Cases {
				if err := validateViewCommandStmts(c.Body, stateConstFields); err != nil {
					return err
				}
			}
		case *frontend.UnsafeStmt:
			if err := validateViewCommandStmts(s.Body, stateConstFields); err != nil {
				return err
			}
		case *frontend.IslandStmt:
			if err := validateViewCommandStmts(s.Body, stateConstFields); err != nil {
				return err
			}
		}
	}
	return nil
}

func assignedStateField(expr frontend.Expr) (string, bool) {
	target := expr
	if idx, ok := expr.(*frontend.IndexExpr); ok {
		target = idx.Base
	}
	base, fields, _, ok := splitFieldPath(target)
	if !ok || base != "state" {
		return "", false
	}
	if len(fields) == 0 {
		return "", true
	}
	return fields[0], true
}
