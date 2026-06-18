package semantics

import (
	"fmt"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/module"
)

func worldFilesImportsFirst(world *module.World) []*frontend.FileAST {
	if world == nil || len(world.Files) == 0 {
		return nil
	}
	if len(world.ByModule) == 0 {
		return append([]*frontend.FileAST(nil), world.Files...)
	}
	out := make([]*frontend.FileAST, 0, len(world.Files))
	seen := map[string]bool{}
	var visit func(*frontend.FileAST)
	visit = func(file *frontend.FileAST) {
		if file == nil {
			return
		}
		key := file.Module
		if key == "" {
			key = file.Path
		}
		if seen[key] {
			return
		}
		seen[key] = true
		for _, imp := range file.Imports {
			visit(world.ByModule[imp.Path])
		}
		out = append(out, file)
	}
	for _, file := range world.Files {
		visit(file)
	}
	return out
}

func clearMatchPayloadExhaustivenessMarkers(world *module.World) {
	if world == nil {
		return
	}
	for _, file := range world.Files {
		if file == nil {
			continue
		}
		for _, fn := range file.Funcs {
			clearStmtListMatchPayloadMarkers(fn.Body)
		}
		for _, test := range file.Tests {
			clearStmtListMatchPayloadMarkers(test.Body)
		}
	}
}

func clearStmtListMatchPayloadMarkers(stmts []frontend.Stmt) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.IfStmt:
			clearStmtListMatchPayloadMarkers(s.Then)
			clearStmtListMatchPayloadMarkers(s.Else)
		case *frontend.IfLetStmt:
			clearStmtListMatchPayloadMarkers(s.Then)
			clearStmtListMatchPayloadMarkers(s.Else)
		case *frontend.WhileStmt:
			clearStmtListMatchPayloadMarkers(s.Body)
		case *frontend.ForRangeStmt:
			clearStmtListMatchPayloadMarkers(s.Body)
		case *frontend.MatchStmt:
			for i := range s.Cases {
				s.Cases[i].RequiresPayload = false
				s.Cases[i].PayloadArity = 0
				clearStmtListMatchPayloadMarkers(s.Cases[i].Body)
			}
		case *frontend.UnsafeStmt:
			clearStmtListMatchPayloadMarkers(s.Body)
		case *frontend.DeferStmt:
			clearStmtListMatchPayloadMarkers(s.Body)
		}
		clearExprMatchPayloadMarkers(stmtExpr(stmt))
	}
}

func stmtExpr(stmt frontend.Stmt) frontend.Expr {
	switch s := stmt.(type) {
	case *frontend.LetStmt:
		return s.Value
	case *frontend.AssignStmt:
		return s.Value
	case *frontend.ExprStmt:
		return s.Expr
	case *frontend.ReturnStmt:
		return s.Value
	case *frontend.ThrowStmt:
		return s.Value
	case *frontend.PrintStmt:
		return s.Value
	case *frontend.FreeStmt:
		return s.Value
	}
	return nil
}

func reportStmtListMatchPayloadDiagnostics(stmts []frontend.Stmt, module string) error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.IfStmt:
			if err := reportStmtListMatchPayloadDiagnostics(s.Then, module); err != nil {
				return err
			}
			if err := reportStmtListMatchPayloadDiagnostics(s.Else, module); err != nil {
				return err
			}
		case *frontend.IfLetStmt:
			if err := reportStmtListMatchPayloadDiagnostics(s.Then, module); err != nil {
				return err
			}
			if err := reportStmtListMatchPayloadDiagnostics(s.Else, module); err != nil {
				return err
			}
		case *frontend.WhileStmt:
			if err := reportStmtListMatchPayloadDiagnostics(s.Body, module); err != nil {
				return err
			}
		case *frontend.ForRangeStmt:
			if err := reportStmtListMatchPayloadDiagnostics(s.Body, module); err != nil {
				return err
			}
		case *frontend.MatchStmt:
			for i := range s.Cases {
				c := &s.Cases[i]
				if c.RequiresPayload {
					return payloadRequiredDiagnostic(c.At, c.Pattern, c.PayloadArity, module)
				}
				if err := reportStmtListMatchPayloadDiagnostics(c.Body, module); err != nil {
					return err
				}
			}
		case *frontend.UnsafeStmt:
			if err := reportStmtListMatchPayloadDiagnostics(s.Body, module); err != nil {
				return err
			}
		case *frontend.DeferStmt:
			if err := reportStmtListMatchPayloadDiagnostics(s.Body, module); err != nil {
				return err
			}
		}
		if err := reportExprMatchPayloadDiagnostics(stmtExpr(stmt), module); err != nil {
			return err
		}
	}
	return nil
}

func reportExprMatchPayloadDiagnostics(expr frontend.Expr, module string) error {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *frontend.MatchExpr:
		for i := range e.Cases {
			c := &e.Cases[i]
			if c.RequiresPayload {
				return payloadRequiredDiagnostic(c.At, c.Pattern, c.PayloadArity, module)
			}
			if err := reportExprMatchPayloadDiagnostics(c.Guard, module); err != nil {
				return err
			}
			if err := reportExprMatchPayloadDiagnostics(c.Value, module); err != nil {
				return err
			}
		}
		return reportExprMatchPayloadDiagnostics(e.Value, module)
	case *frontend.CatchExpr:
		for i := range e.Cases {
			c := &e.Cases[i]
			if c.RequiresPayload {
				return payloadRequiredDiagnostic(c.At, c.Pattern, c.PayloadArity, module)
			}
			if err := reportExprMatchPayloadDiagnostics(c.Guard, module); err != nil {
				return err
			}
			if err := reportExprMatchPayloadDiagnostics(c.Value, module); err != nil {
				return err
			}
		}
		return reportExprMatchPayloadDiagnostics(e.Call, module)
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			if err := reportExprMatchPayloadDiagnostics(arg, module); err != nil {
				return err
			}
		}
	case *frontend.UnaryExpr:
		return reportExprMatchPayloadDiagnostics(e.X, module)
	case *frontend.BinaryExpr:
		if err := reportExprMatchPayloadDiagnostics(e.Left, module); err != nil {
			return err
		}
		return reportExprMatchPayloadDiagnostics(e.Right, module)
	case *frontend.FieldAccessExpr:
		return reportExprMatchPayloadDiagnostics(e.Base, module)
	case *frontend.IndexExpr:
		if err := reportExprMatchPayloadDiagnostics(e.Base, module); err != nil {
			return err
		}
		return reportExprMatchPayloadDiagnostics(e.Index, module)
	case *frontend.TryExpr:
		return reportExprMatchPayloadDiagnostics(e.X, module)
	case *frontend.AwaitExpr:
		return reportExprMatchPayloadDiagnostics(e.X, module)
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if err := reportExprMatchPayloadDiagnostics(field.Value, module); err != nil {
				return err
			}
		}
	case *frontend.ClosureExpr:
		if e.Decl != nil {
			return reportStmtListMatchPayloadDiagnostics(e.Decl.Body, module)
		}
	}
	return nil
}

func payloadRequiredDiagnostic(pos frontend.Position, pattern frontend.Expr, arity int, module string) error {
	if field, ok := pattern.(*frontend.FieldAccessExpr); ok {
		if arity <= 0 {
			arity = 1
		}
		return fmt.Errorf("%s: enum case '%s.%s' carries %d payload value(s); use '%s.%s(%s)'", frontend.FormatPos(pos), displayTypeName(field.EnumType, module), field.Field, arity, displayTypeName(field.EnumType, module), field.Field, placeholderBindingList(arity))
	}
	return fmt.Errorf("%s: enum payload pattern requires destructuring", frontend.FormatPos(pos))
}

func clearExprMatchPayloadMarkers(expr frontend.Expr) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *frontend.MatchExpr:
		for i := range e.Cases {
			e.Cases[i].RequiresPayload = false
			e.Cases[i].PayloadArity = 0
			clearExprMatchPayloadMarkers(e.Cases[i].Guard)
			clearExprMatchPayloadMarkers(e.Cases[i].Value)
		}
		clearExprMatchPayloadMarkers(e.Value)
	case *frontend.CatchExpr:
		for i := range e.Cases {
			e.Cases[i].RequiresPayload = false
			e.Cases[i].PayloadArity = 0
			clearExprMatchPayloadMarkers(e.Cases[i].Guard)
			clearExprMatchPayloadMarkers(e.Cases[i].Value)
		}
		clearExprMatchPayloadMarkers(e.Call)
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			clearExprMatchPayloadMarkers(arg)
		}
	case *frontend.UnaryExpr:
		clearExprMatchPayloadMarkers(e.X)
	case *frontend.BinaryExpr:
		clearExprMatchPayloadMarkers(e.Left)
		clearExprMatchPayloadMarkers(e.Right)
	case *frontend.FieldAccessExpr:
		clearExprMatchPayloadMarkers(e.Base)
	case *frontend.IndexExpr:
		clearExprMatchPayloadMarkers(e.Base)
		clearExprMatchPayloadMarkers(e.Index)
	case *frontend.TryExpr:
		clearExprMatchPayloadMarkers(e.X)
	case *frontend.AwaitExpr:
		clearExprMatchPayloadMarkers(e.X)
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			clearExprMatchPayloadMarkers(field.Value)
		}
	case *frontend.ClosureExpr:
		if e.Decl != nil {
			clearStmtListMatchPayloadMarkers(e.Decl.Body)
		}
	}
}

func fileUsesExplicitPublic(file *frontend.FileAST) bool {
	if file == nil {
		return false
	}
	for _, imp := range file.Imports {
		if imp.Public {
			return true
		}
	}
	for _, en := range file.Enums {
		if en.Public {
			return true
		}
	}
	for _, st := range file.Structs {
		if st.Public {
			return true
		}
	}
	for _, st := range file.States {
		if st.Public {
			return true
		}
	}
	for _, view := range file.Views {
		if view.Public {
			return true
		}
	}
	for _, proto := range file.Protocols {
		if proto.Public {
			return true
		}
	}
	for _, glob := range file.Globals {
		if glob.Public {
			return true
		}
	}
	for _, fn := range file.Funcs {
		if fn.Public {
			return true
		}
	}
	return false
}

func declarationIsPublic(file *frontend.FileAST, public bool) bool {
	if !fileUsesExplicitPublic(file) {
		return true
	}
	return public
}

type globalConstValue struct {
	TypeName string
	I32      int32
	Bool     bool
}

func inferGlobalConstExprType(expr frontend.Expr, values map[string]globalConstValue) (string, bool) {
	if _, ok := evalGlobalConstBool(expr, values); ok {
		return "bool", true
	}
	if _, ok := evalGlobalConstI32(expr, values); ok {
		return "i32", true
	}
	return "", false
}

func validateGlobalConstExpr(expr frontend.Expr, values map[string]globalConstValue) error {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		if _, ok := values[e.Name]; !ok {
			return fmt.Errorf("%s: unknown constant '%s' in global const expression", frontend.FormatPos(e.At), e.Name)
		}
	case *frontend.UnaryExpr:
		if err := validateGlobalConstExpr(e.X, values); err != nil {
			return err
		}
		if _, ok, overflow := evalGlobalConstI32Wide(e, values); ok && overflow {
			return fmt.Errorf("%s: overflow in global const expression", frontend.FormatPos(e.At))
		}
	case *frontend.BinaryExpr:
		if err := validateGlobalConstExpr(e.Left, values); err != nil {
			return err
		}
		if err := validateGlobalConstExpr(e.Right, values); err != nil {
			return err
		}
		switch e.Op {
		case frontend.TokenSlash, frontend.TokenPercent:
			right, ok := evalGlobalConstI32(e.Right, values)
			if ok && right == 0 {
				if e.Op == frontend.TokenSlash {
					return fmt.Errorf("%s: division by zero in global const expression", frontend.FormatPos(e.At))
				}
				return fmt.Errorf("%s: modulo by zero in global const expression", frontend.FormatPos(e.At))
			}
		}
		if _, ok, overflow := evalGlobalConstI32Wide(e, values); ok && overflow {
			return fmt.Errorf("%s: overflow in global const expression", frontend.FormatPos(e.At))
		}
	}
	return nil
}

func evalGlobalConstI32(expr frontend.Expr, values map[string]globalConstValue) (int32, bool) {
	v, ok, overflow := evalGlobalConstI32Wide(expr, values)
	if !ok || overflow {
		return 0, false
	}
	return int32(v), true
}

func evalGlobalConstI32Wide(expr frontend.Expr, values map[string]globalConstValue) (int64, bool, bool) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return int64(e.Value), true, false
	case *frontend.IdentExpr:
		v, ok := values[e.Name]
		if !ok || !isGlobalIntLikeType(v.TypeName) {
			return 0, false, false
		}
		return int64(v.I32), true, false
	case *frontend.UnaryExpr:
		if e.Op != frontend.TokenMinus {
			return 0, false, false
		}
		v, ok, overflow := evalGlobalConstI32Wide(e.X, values)
		if !ok || overflow {
			return 0, ok, overflow
		}
		return checkedConstI32(-v)
	case *frontend.BinaryExpr:
		left, ok, overflow := evalGlobalConstI32Wide(e.Left, values)
		if !ok || overflow {
			return 0, ok, overflow
		}
		right, ok, overflow := evalGlobalConstI32Wide(e.Right, values)
		if !ok || overflow {
			return 0, ok, overflow
		}
		switch e.Op {
		case frontend.TokenPlus:
			return checkedConstI32(left + right)
		case frontend.TokenMinus:
			return checkedConstI32(left - right)
		case frontend.TokenStar:
			return checkedConstI32(left * right)
		case frontend.TokenSlash:
			if right == 0 {
				return 0, false, false
			}
			return checkedConstI32(left / right)
		case frontend.TokenPercent:
			if right == 0 {
				return 0, false, false
			}
			return checkedConstI32(left % right)
		default:
			return 0, false, false
		}
	default:
		return 0, false, false
	}
}

func isSupportedGlobalScalarType(name string) bool {
	switch name {
	case "i32", "bool", "ptr", "fnptr", "str", "u8", "u16", "c_int", "c_uint", "task.error":
		return true
	default:
		return isSupportedSliceGlobalType(name) || isSupportedOptionalPtrGlobalType(name) || isSupportedOptionalSliceGlobalType(name)
	}
}

func isSupportedGlobalType(name string, types map[string]*TypeInfo) bool {
	if isSupportedGlobalScalarType(name) {
		return true
	}
	if isSupportedOptionalAggregateGlobalType(name, types) {
		return true
	}
	return isSupportedZeroedAggregateGlobalType(name, types, map[string]bool{})
}

func isSupportedOptionalAggregateGlobalType(name string, types map[string]*TypeInfo) bool {
	elem, ok := optionalElemName(name)
	if !ok {
		return false
	}
	if isSupportedGlobalScalarType(elem) {
		return false
	}
	return isSupportedZeroedAggregateGlobalType(elem, types, map[string]bool{})
}

func isSupportedZeroedAggregateGlobalType(name string, types map[string]*TypeInfo, visiting map[string]bool) bool {
	if _, elem, ok := parseArrayTypeName(name); ok {
		return isSupportedGlobalScalarType(elem)
	}
	if visiting[name] {
		return false
	}
	info, ok := types[name]
	if !ok {
		return false
	}
	visiting[name] = true
	defer delete(visiting, name)
	switch info.Kind {
	case TypeArray:
		return isSupportedGlobalScalarType(info.ElemType)
	case TypeStruct:
		for _, field := range info.Fields {
			if field.FunctionTypeValue {
				return false
			}
			if isSupportedGlobalScalarType(field.TypeName) {
				continue
			}
			if !isSupportedZeroedAggregateGlobalType(field.TypeName, types, visiting) {
				return false
			}
		}
		return true
	case TypeEnum:
		for _, enumCase := range info.EnumCases {
			for i, payloadType := range enumCase.PayloadTypes {
				if i < len(enumCase.PayloadFunctionTypes) && enumCase.PayloadFunctionTypes[i] {
					return false
				}
				if isSupportedGlobalScalarType(payloadType) {
					continue
				}
				if !isSupportedZeroedAggregateGlobalType(payloadType, types, visiting) {
					return false
				}
			}
		}
		return true
	default:
		return false
	}
}

func collectGlobalArrayBackings(typeName string, offset int, types map[string]*TypeInfo, visiting map[string]bool) []GlobalArrayBackingInfo {
	info, ok := types[typeName]
	if !ok || info == nil {
		return nil
	}
	switch info.Kind {
	case TypeArray:
		return []GlobalArrayBackingInfo{{
			HeaderOffset: offset,
			ElemType:     info.ElemType,
			Len:          info.ArrayLen,
		}}
	case TypeStruct:
		if visiting[typeName] {
			return nil
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		var out []GlobalArrayBackingInfo
		for _, field := range info.Fields {
			out = append(out, collectGlobalArrayBackings(field.TypeName, offset+field.Offset, types, visiting)...)
		}
		return out
	default:
		return nil
	}
}

func isSupportedOptionalSliceGlobalType(name string) bool {
	elem, ok := optionalElemName(name)
	if !ok {
		return false
	}
	sliceElem, ok := sliceElemName(elem)
	if !ok {
		return false
	}
	return isSupportedGlobalSliceElemType(sliceElem)
}

func isSupportedOptionalPtrGlobalType(name string) bool {
	elem, ok := optionalElemName(name)
	return ok && elem == "ptr"
}

func isSupportedSliceGlobalType(name string) bool {
	sliceElem, ok := sliceElemName(name)
	if !ok {
		return false
	}
	return isSupportedGlobalSliceElemType(sliceElem)
}

func isSupportedGlobalSliceElemType(sliceElem string) bool {
	switch sliceElem {
	case "i32", "u8", "u16", "c_int", "c_uint", "bool",
		"usize", "isize", "size_t", "ssize_t", "native_int", "native_uint", "c_long", "c_ulong":
		return true
	default:
		return false
	}
}

func isGlobalIntLikeType(name string) bool {
	switch name {
	case "i32", "u8", "u16", "c_int", "c_uint", "task.error",
		"usize", "isize", "size_t", "ssize_t", "native_int", "native_uint", "c_long", "c_ulong":
		return true
	default:
		return false
	}
}

func validateGlobalIntLikeRange(typeName, globalName, kind string, pos frontend.Position, v int32) error {
	switch typeName {
	case "u8":
		if v < 0 || v > 255 {
			return fmt.Errorf("%s: global %s '%s' initializer must be within 0..255 for type u8", frontend.FormatPos(pos), kind, globalName)
		}
	case "u16":
		if v < 0 || v > 65535 {
			return fmt.Errorf("%s: global %s '%s' initializer must be within 0..65535 for type u16", frontend.FormatPos(pos), kind, globalName)
		}
	case "c_uint":
		if v < 0 {
			return fmt.Errorf("%s: global %s '%s' initializer must be non-negative for type c_uint", frontend.FormatPos(pos), kind, globalName)
		}
	case "usize", "size_t", "native_uint", "c_ulong":
		if v < 0 {
			return fmt.Errorf("%s: global %s '%s' initializer must be non-negative for type %s", frontend.FormatPos(pos), kind, globalName, typeName)
		}
	}
	return nil
}

func evalGlobalConstBool(expr frontend.Expr, values map[string]globalConstValue) (bool, bool) {
	switch e := expr.(type) {
	case *frontend.BoolLitExpr:
		return e.Value, true
	case *frontend.IdentExpr:
		v, ok := values[e.Name]
		if !ok || v.TypeName != "bool" {
			return false, false
		}
		return v.Bool, true
	case *frontend.UnaryExpr:
		if e.Op != frontend.TokenBang {
			return false, false
		}
		v, ok := evalGlobalConstBool(e.X, values)
		if !ok {
			return false, false
		}
		return !v, true
	case *frontend.BinaryExpr:
		switch e.Op {
		case frontend.TokenAmpAmp:
			left, ok := evalGlobalConstBool(e.Left, values)
			if !ok {
				return false, false
			}
			right, ok := evalGlobalConstBool(e.Right, values)
			if !ok {
				return false, false
			}
			return left && right, true
		case frontend.TokenPipePipe:
			left, ok := evalGlobalConstBool(e.Left, values)
			if !ok {
				return false, false
			}
			right, ok := evalGlobalConstBool(e.Right, values)
			if !ok {
				return false, false
			}
			return left || right, true
		case frontend.TokenEqEq, frontend.TokenBangEq:
			if left, ok := evalGlobalConstI32(e.Left, values); ok {
				right, ok := evalGlobalConstI32(e.Right, values)
				if !ok {
					return false, false
				}
				if e.Op == frontend.TokenEqEq {
					return left == right, true
				}
				return left != right, true
			}
			left, ok := evalGlobalConstBool(e.Left, values)
			if !ok {
				return false, false
			}
			right, ok := evalGlobalConstBool(e.Right, values)
			if !ok {
				return false, false
			}
			if e.Op == frontend.TokenEqEq {
				return left == right, true
			}
			return left != right, true
		case frontend.TokenLess, frontend.TokenLessEq, frontend.TokenGreater, frontend.TokenGreaterEq:
			left, ok := evalGlobalConstI32(e.Left, values)
			if !ok {
				return false, false
			}
			right, ok := evalGlobalConstI32(e.Right, values)
			if !ok {
				return false, false
			}
			switch e.Op {
			case frontend.TokenLess:
				return left < right, true
			case frontend.TokenLessEq:
				return left <= right, true
			case frontend.TokenGreater:
				return left > right, true
			case frontend.TokenGreaterEq:
				return left >= right, true
			}
		}
	}
	return false, false
}

func evaluateActorStateInitializer(expr frontend.Expr) (string, int32, bool) {
	if v, ok := evalGlobalConstI32(expr, nil); ok {
		return "i32", v, true
	}
	if v, ok := evalGlobalConstBool(expr, nil); ok {
		if v {
			return "bool", 1, true
		}
		return "bool", 0, true
	}
	return "", 0, false
}
