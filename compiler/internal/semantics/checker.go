package semantics

import (
	"encoding/binary"
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/module"
)

func Check(prog *frontend.Program) (*CheckedProgram, error) {
	if prog == nil {
		return nil, fmt.Errorf("no program provided")
	}
	file := &frontend.FileAST{
		Module:     "",
		Capsules:   prog.Capsules,
		Enums:      prog.Enums,
		Structs:    prog.Structs,
		States:     prog.States,
		Views:      prog.Views,
		Actors:     prog.Actors,
		Protocols:  prog.Protocols,
		Extensions: prog.Extensions,
		Impls:      prog.Impls,
		Funcs:      prog.Funcs,
		Tests:      prog.Tests,
	}
	world := &module.World{
		EntryModule: "",
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{"": file},
	}
	return CheckWorldOpt(world, CheckOptions{RequireMain: true})
}

type CheckOptions struct {
	RequireMain bool
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
		return validateGlobalConstExpr(e.X, values)
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
	}
	return nil
}

func evalGlobalConstI32(expr frontend.Expr, values map[string]globalConstValue) (int32, bool) {
	switch e := expr.(type) {
	case *frontend.NumberExpr:
		return e.Value, true
	case *frontend.IdentExpr:
		v, ok := values[e.Name]
		if !ok || !isGlobalIntLikeType(v.TypeName) {
			return 0, false
		}
		return v.I32, true
	case *frontend.UnaryExpr:
		if e.Op != frontend.TokenMinus {
			return 0, false
		}
		v, ok := evalGlobalConstI32(e.X, values)
		if !ok {
			return 0, false
		}
		return -v, true
	case *frontend.BinaryExpr:
		left, ok := evalGlobalConstI32(e.Left, values)
		if !ok {
			return 0, false
		}
		right, ok := evalGlobalConstI32(e.Right, values)
		if !ok {
			return 0, false
		}
		switch e.Op {
		case frontend.TokenPlus:
			return left + right, true
		case frontend.TokenMinus:
			return left - right, true
		case frontend.TokenStar:
			return left * right, true
		case frontend.TokenSlash:
			if right == 0 {
				return 0, false
			}
			return left / right, true
		case frontend.TokenPercent:
			if right == 0 {
				return 0, false
			}
			return left % right, true
		default:
			return 0, false
		}
	default:
		return 0, false
	}
}

func isSupportedGlobalScalarType(name string) bool {
	switch name {
	case "i32", "bool", "ptr", "str", "u8", "u16", "task.error":
		return true
	default:
		return false
	}
}

func isGlobalIntLikeType(name string) bool {
	switch name {
	case "i32", "u8", "u16", "task.error":
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

func CheckWorld(world *module.World) (*CheckedProgram, error) {
	return CheckWorldOpt(world, CheckOptions{RequireMain: true})
}

func CheckWorldOpt(world *module.World, opt CheckOptions) (*CheckedProgram, error) {
	if world == nil || len(world.Files) == 0 {
		return nil, fmt.Errorf("no functions found")
	}
	if err := monomorphizeGenerics(world); err != nil {
		return nil, err
	}
	clearMatchPayloadExhaustivenessMarkers(world)
	if err := normalizeExtensionMethodNames(world); err != nil {
		return nil, err
	}

	types := baseTypes()

	type structContext struct {
		module  string
		imports map[string]string
		public  bool
		decl    *frontend.StructDecl
	}
	type enumContext struct {
		module  string
		imports map[string]string
		public  bool
		decl    *frontend.EnumDecl
	}
	type protocolContext struct {
		module  string
		imports map[string]string
		public  bool
		decl    *frontend.ProtocolDecl
	}
	type actorContext struct {
		module  string
		imports map[string]string
		decl    *frontend.ActorDecl
	}

	structs := make(map[string]structContext)
	enums := make(map[string]enumContext)
	protocols := make(map[string]protocolContext)
	actors := make([]actorContext, 0)
	actorStateFieldsByMethod := make(map[string]map[string]ActorStateField)
	checked := CheckedProgram{
		MainIndex:          -1,
		Types:              types,
		FuncSigs:           make(map[string]FuncSig),
		GlobalsByModule:    make(map[string]map[string]GlobalInfo),
		GlobalDataByModule: make(map[string][][]byte),
	}
	exportedSymbols := make(map[string]string)

	seenImpls := map[string]frontend.Position{}
	for _, file := range world.Files {
		module := file.Module
		imports, err := collectImportAliases(file)
		if err != nil {
			return nil, err
		}
		if err := validateCapsuleDecls(file); err != nil {
			return nil, err
		}
		for _, en := range file.Enums {
			fullName := qualifyName(module, en.Name)
			if isReservedTypeName(en.Name) {
				return nil, fmt.Errorf("%s: reserved type name '%s'", frontend.FormatPos(en.At), en.Name)
			}
			if _, exists := enums[fullName]; exists {
				return nil, fmt.Errorf("duplicate enum '%s'", fullName)
			}
			if _, exists := structs[fullName]; exists {
				return nil, fmt.Errorf("duplicate type '%s'", fullName)
			}
			enums[fullName] = enumContext{module: module, imports: imports, public: declarationIsPublic(file, en.Public), decl: en}
			checked.Enums = append(checked.Enums, CheckedEnum{Name: fullName, Module: module, Decl: en})
		}
		for _, st := range file.Structs {
			fullName := qualifyName(module, st.Name)
			if isReservedTypeName(st.Name) {
				return nil, fmt.Errorf("%s: reserved type name '%s'", frontend.FormatPos(st.At), st.Name)
			}
			if _, exists := structs[fullName]; exists {
				return nil, fmt.Errorf("duplicate struct '%s'", fullName)
			}
			if _, exists := enums[fullName]; exists {
				return nil, fmt.Errorf("duplicate type '%s'", fullName)
			}
			structs[fullName] = structContext{module: module, imports: imports, public: declarationIsPublic(file, st.Public), decl: st}
			checked.Structs = append(checked.Structs, CheckedStruct{Name: fullName, Module: module, Decl: st})
		}
		for _, st := range file.States {
			fullName := qualifyName(module, st.Name)
			if isReservedTypeName(st.Name) {
				return nil, fmt.Errorf("%s: reserved type name '%s'", frontend.FormatPos(st.At), st.Name)
			}
			if _, exists := structs[fullName]; exists {
				return nil, fmt.Errorf("duplicate state '%s'", fullName)
			}
			if _, exists := enums[fullName]; exists {
				return nil, fmt.Errorf("duplicate type '%s'", fullName)
			}
			synth := stateAsStructDecl(st)
			synth.Public = declarationIsPublic(file, st.Public)
			structs[fullName] = structContext{module: module, imports: imports, public: synth.Public, decl: synth}
			checked.UIStates = append(checked.UIStates, CheckedUIState{Name: fullName, Module: module, Decl: st})
		}
		for _, proto := range file.Protocols {
			fullName := qualifyName(module, proto.Name)
			if isReservedTypeName(proto.Name) {
				return nil, fmt.Errorf("%s: reserved type name '%s'", frontend.FormatPos(proto.At), proto.Name)
			}
			if _, exists := protocols[fullName]; exists {
				return nil, fmt.Errorf("duplicate protocol '%s'", fullName)
			}
			protocols[fullName] = protocolContext{module: module, imports: imports, public: declarationIsPublic(file, proto.Public), decl: proto}
			checked.Protocols = append(checked.Protocols, CheckedProtocol{Name: fullName, Module: module, Decl: proto})
		}
		for _, actor := range file.Actors {
			if actor == nil {
				continue
			}
			actors = append(actors, actorContext{module: module, imports: imports, decl: actor})
			fields := make(map[string]ActorStateField, len(actor.Fields))
			for _, method := range actor.Methods {
				methodName := checkedFuncFullName(module, method)
				actorStateFieldsByMethod[methodName] = fields
			}
		}
	}

	for name, ctx := range enums {
		caseMap := make(map[string]EnumCaseInfo, len(ctx.decl.Cases))
		cases := make([]EnumCaseInfo, 0, len(ctx.decl.Cases))
		for i, c := range ctx.decl.Cases {
			if _, exists := caseMap[c.Name]; exists {
				return nil, fmt.Errorf("%s: duplicate enum case '%s'", frontend.FormatPos(c.At), c.Name)
			}
			info := EnumCaseInfo{Name: c.Name, Ordinal: int32(i), SlotCount: 0}
			caseMap[c.Name] = info
			cases = append(cases, info)
		}
		types[name] = &TypeInfo{
			Name:      name,
			Kind:      TypeEnum,
			Public:    ctx.public,
			SlotCount: 1,
			EnumCases: cases,
			CaseMap:   caseMap,
		}
	}

	state := make(map[string]int)
	var buildType func(name string) (*TypeInfo, error)
	buildType = func(name string) (*TypeInfo, error) {
		if info, ok := types[name]; ok {
			return info, nil
		}
		if elem, ok := optionalElemName(name); ok {
			elemInfo, err := buildType(elem)
			if err != nil {
				return nil, err
			}
			info := &TypeInfo{
				Name:      name,
				Kind:      TypeOptional,
				Public:    true,
				SlotCount: elemInfo.SlotCount + 1,
				ElemType:  elem,
			}
			types[name] = info
			return info, nil
		}
		if elem, ok := sliceElemName(name); ok {
			if elem == "" {
				return nil, fmt.Errorf("invalid slice type '%s'", name)
			}
			if isArrayTypeName(elem) {
				return nil, fmt.Errorf("array element types are not supported yet")
			}
			if elem != "i32" && elem != "u8" && elem != "u16" && elem != "bool" {
				return nil, fmt.Errorf("slice element type '%s' is not supported", elem)
			}
			info := makeSliceTypeInfo(name, elem)
			types[name] = info
			return info, nil
		}
		if n, elem, ok := parseArrayTypeName(name); ok {
			if n <= 0 {
				return nil, fmt.Errorf("array size must be positive constant")
			}
			if !isSupportedArrayElemType(elem) {
				return nil, fmt.Errorf("array element type '%s' is not supported", elem)
			}
			info := makeArrayTypeInfo(name, elem, n)
			types[name] = info
			return info, nil
		}
		if isArrayTypeName(name) {
			return nil, fmt.Errorf("invalid array type '%s'", name)
		}
		ctx, ok := structs[name]
		if !ok {
			return nil, fmt.Errorf("unknown type '%s'", name)
		}
		switch state[name] {
		case 1:
			return nil, fmt.Errorf("%s: recursive struct '%s'", frontend.FormatPos(ctx.decl.At), name)
		case 2:
			if info, ok := types[name]; ok {
				return info, nil
			}
		}
		state[name] = 1

		fieldMap := make(map[string]FieldInfo)
		var fields []FieldInfo
		slotCount := 0
		for i := range ctx.decl.Fields {
			field := &ctx.decl.Fields[i]
			if _, exists := fieldMap[field.Name]; exists {
				return nil, fmt.Errorf("%s: duplicate field '%s'", frontend.FormatPos(field.At), field.Name)
			}
			resolved, err := resolveTypeName(&field.Type, ctx.module, ctx.imports)
			if err != nil {
				return nil, err
			}
			field.Type.Name = resolved
			fieldType, err := buildType(resolved)
			if err != nil {
				return nil, err
			}
			if err := ensureTypeVisible(resolved, fieldType, ctx.module, field.At); err != nil {
				return nil, err
			}
			info := FieldInfo{
				Name:      field.Name,
				TypeName:  resolved,
				Offset:    slotCount,
				SlotCount: fieldType.SlotCount,
			}
			fieldMap[field.Name] = info
			fields = append(fields, info)
			slotCount += fieldType.SlotCount
		}

		info := &TypeInfo{
			Name:      name,
			Kind:      TypeStruct,
			Public:    ctx.public,
			Fields:    fields,
			FieldMap:  fieldMap,
			SlotCount: slotCount,
		}
		types[name] = info
		state[name] = 2
		return info, nil
	}

	for name := range structs {
		if _, err := buildType(name); err != nil {
			return nil, err
		}
	}
	for _, ctx := range actors {
		actorName := qualifyName(ctx.module, ctx.decl.Name)
		slot := 0
		for i := range ctx.decl.Fields {
			field := &ctx.decl.Fields[i]
			resolved, err := resolveTypeName(&field.Type, ctx.module, ctx.imports)
			if err != nil {
				return nil, err
			}
			field.Type.Name = resolved
			fieldType, err := buildType(resolved)
			if err != nil {
				return nil, fmt.Errorf("%s: actor '%s' state field '%s': %v", frontend.FormatPos(field.At), displayTypeName(actorName, ctx.module), field.Name, err)
			}
			if err := ensureTypeVisible(resolved, fieldType, ctx.module, field.At); err != nil {
				return nil, err
			}
			if !isSupportedActorStateScalarType(resolved) {
				return nil, fmt.Errorf("%s: actor state field '%s' type '%s' is not supported in this MVP (supported: Int/Bool/UInt8/UInt16/task.error)", frontend.FormatPos(field.At), field.Name, resolved)
			}
			if field.Init == nil {
				return nil, fmt.Errorf("%s: actor state field '%s' requires a compile-time constant initializer", frontend.FormatPos(field.At), field.Name)
			}
			initType, initValue, ok := evaluateActorStateInitializer(field.Init)
			if !ok {
				return nil, fmt.Errorf("%s: actor state field '%s' initializer must be a compile-time constant i32/bool", frontend.FormatPos(field.At), field.Name)
			}
			if !typesCompatible(resolved, initType) {
				return nil, fmt.Errorf("%s: actor state field '%s' type mismatch: expected '%s', got '%s'", frontend.FormatPos(field.At), field.Name, resolved, initType)
			}
			stateField := ActorStateField{
				Name:     field.Name,
				Slot:     slot,
				TypeName: resolved,
				Mutable:  field.Mutable,
				Const:    field.Const,
				Init:     initValue,
			}
			for _, method := range ctx.decl.Methods {
				methodName := checkedFuncFullName(ctx.module, method)
				if fields, ok := actorStateFieldsByMethod[methodName]; ok {
					fields[field.Name] = stateField
				}
			}
			slot++
		}
	}

	for name, ctx := range enums {
		info := types[name]
		if info == nil || info.Kind != TypeEnum {
			return nil, fmt.Errorf("internal error: enum '%s' has no type info", name)
		}
		maxPayloadSlots := 0
		for i := range ctx.decl.Cases {
			declCase := &ctx.decl.Cases[i]
			caseInfo := info.EnumCases[i]
			caseInfo.PayloadTypes = caseInfo.PayloadTypes[:0]
			caseInfo.PayloadSlots = caseInfo.PayloadSlots[:0]
			totalPayloadSlots := 0
			for j := range declCase.Payload {
				payload := &declCase.Payload[j]
				resolved, err := resolveTypeName(payload, ctx.module, ctx.imports)
				if err != nil {
					return nil, err
				}
				if resolved == name {
					return nil, fmt.Errorf("%s: recursive enum payload '%s'", frontend.FormatPos(payload.At), displayTypeName(name, ctx.module))
				}
				payload.Name = resolved
				payloadInfo, err := buildType(resolved)
				if err != nil {
					return nil, fmt.Errorf("%s: enum '%s' case '%s': %v", frontend.FormatPos(payload.At), displayTypeName(name, ctx.module), declCase.Name, err)
				}
				if err := ensureTypeVisible(resolved, payloadInfo, ctx.module, payload.At); err != nil {
					return nil, err
				}
				caseInfo.PayloadTypes = append(caseInfo.PayloadTypes, resolved)
				caseInfo.PayloadSlots = append(caseInfo.PayloadSlots, payloadInfo.SlotCount)
				totalPayloadSlots += payloadInfo.SlotCount
			}
			caseInfo.SlotCount = totalPayloadSlots
			info.EnumCases[i] = caseInfo
			info.CaseMap[caseInfo.Name] = caseInfo
			if totalPayloadSlots > maxPayloadSlots {
				maxPayloadSlots = totalPayloadSlots
			}
		}
		info.SlotCount = 1 + maxPayloadSlots
	}

	for name, ctx := range protocols {
		seenReqs := map[string]struct{}{}
		for i := range ctx.decl.Requirements {
			req := &ctx.decl.Requirements[i]
			if _, exists := seenReqs[req.Name]; exists {
				return nil, fmt.Errorf("%s: duplicate protocol requirement '%s'", frontend.FormatPos(req.At), req.Name)
			}
			seenReqs[req.Name] = struct{}{}
			reqEffects, err := normalizeEffects(req.Uses, req.At)
			if err != nil {
				return nil, fmt.Errorf("%s: protocol '%s' requirement '%s': %v", frontend.FormatPos(req.At), name, req.Name, err)
			}
			req.Uses = reqEffects
			reqTypeParams := make(map[string]struct{}, len(req.TypeParams))
			for _, tp := range req.TypeParams {
				reqTypeParams[tp] = struct{}{}
			}
			retName, retIsGeneric, err := resolveProtocolRequirementTypeRef(&req.ReturnType, ctx.module, ctx.imports, reqTypeParams)
			if err != nil {
				return nil, fmt.Errorf("%s: protocol '%s' requirement '%s': %v", frontend.FormatPos(req.ReturnType.At), name, req.Name, err)
			}
			req.ReturnType.Name = retName
			if !retIsGeneric {
				retInfo, err := buildType(retName)
				if err != nil {
					return nil, fmt.Errorf("%s: protocol '%s' requirement '%s': %v", frontend.FormatPos(req.At), name, req.Name, err)
				}
				if err := ensureTypeVisible(retName, retInfo, ctx.module, req.At); err != nil {
					return nil, err
				}
			}
			if req.HasThrows {
				throwName, throwIsGeneric, err := resolveProtocolRequirementTypeRef(&req.Throws, ctx.module, ctx.imports, reqTypeParams)
				if err != nil {
					return nil, fmt.Errorf("%s: protocol '%s' requirement '%s': %v", frontend.FormatPos(req.Throws.At), name, req.Name, err)
				}
				req.Throws.Name = throwName
				if !throwIsGeneric {
					throwInfo, err := buildType(throwName)
					if err != nil {
						return nil, fmt.Errorf("%s: protocol '%s' requirement '%s': %v", frontend.FormatPos(req.At), name, req.Name, err)
					}
					if err := ensureTypeVisible(throwName, throwInfo, ctx.module, req.Throws.At); err != nil {
						return nil, err
					}
				}
			}
			for j := range req.Params {
				param := &req.Params[j]
				resolved, paramIsGeneric, err := resolveProtocolRequirementTypeRef(&param.Type, ctx.module, ctx.imports, reqTypeParams)
				if err != nil {
					return nil, fmt.Errorf("%s: protocol '%s' requirement '%s': %v", frontend.FormatPos(param.At), name, req.Name, err)
				}
				param.Type.Name = resolved
				if !paramIsGeneric {
					paramInfo, err := buildType(resolved)
					if err != nil {
						return nil, fmt.Errorf("%s: protocol '%s' requirement '%s': %v", frontend.FormatPos(param.At), name, req.Name, err)
					}
					if err := ensureTypeVisible(resolved, paramInfo, ctx.module, param.At); err != nil {
						return nil, err
					}
				}
			}
		}
	}

	builtinSigs, err := builtinFuncSigs(types)
	if err != nil {
		return nil, err
	}

	for _, file := range world.Files {
		module := file.Module
		imports, err := collectImportAliases(file)
		if err != nil {
			return nil, err
		}
		if len(file.Globals) == 0 {
			continue
		}

		fnNames := make(map[string]struct{}, len(file.Funcs))
		for _, fn := range file.Funcs {
			fnNames[fn.Name] = struct{}{}
		}

		globals := make(map[string]GlobalInfo, len(file.Globals))
		constValues := make(map[string]globalConstValue, len(file.Globals))
		var dataBlobs [][]byte
		for _, glob := range file.Globals {
			if glob == nil {
				continue
			}
			if _, exists := globals[glob.Name]; exists {
				return nil, fmt.Errorf("%s: duplicate global '%s'", frontend.FormatPos(glob.At), glob.Name)
			}
			if _, exists := fnNames[glob.Name]; exists {
				return nil, fmt.Errorf("%s: global '%s' conflicts with function '%s'", frontend.FormatPos(glob.At), glob.Name, glob.Name)
			}

			resolved := ""
			if glob.Type.Name != "" || glob.Type.Elem != nil {
				var err error
				resolved, err = resolveTypeName(&glob.Type, module, imports)
				if err != nil {
					return nil, err
				}
			}
			if resolved == "" {
				if glob.Mutable {
					return nil, fmt.Errorf("%s: global var requires an explicit type annotation", frontend.FormatPos(glob.At))
				}
				if glob.Init == nil {
					return nil, fmt.Errorf("%s: global val requires an initializer to infer its type", frontend.FormatPos(glob.At))
				}
				if err := validateGlobalConstExpr(glob.Init, constValues); err != nil {
					return nil, err
				}
				inferred, ok := inferGlobalConstExprType(glob.Init, constValues)
				if !ok {
					return nil, fmt.Errorf("%s: unsupported global val initializer (type inference supports constant numeric/bool expressions)", frontend.FormatPos(glob.At))
				}
				resolved = inferred
			}
			glob.Type.Name = resolved
			if !isSupportedGlobalScalarType(resolved) {
				return nil, fmt.Errorf("%s: global '%s' has unsupported type '%s' (allowed: i32, bool, ptr, str, u8, u16, task.error)", frontend.FormatPos(glob.At), glob.Name, resolved)
			}
			typeInfo, err := ensureTypeInfo(resolved, types)
			if err != nil {
				return nil, fmt.Errorf("%s: %v", frontend.FormatPos(glob.At), err)
			}
			stringInit := []byte(nil)
			hasStringInit := false
			if resolved == "str" {
				if glob.Init == nil {
					kind := "val"
					if glob.Mutable {
						kind = "var"
					}
					return nil, fmt.Errorf("%s: global %s '%s' initializer must be a string literal", frontend.FormatPos(glob.At), kind, glob.Name)
				}
				lit, ok := glob.Init.(*frontend.StringLitExpr)
				if !ok {
					kind := "val"
					if glob.Mutable {
						kind = "var"
					}
					return nil, fmt.Errorf("%s: global %s '%s' initializer must be a string literal", frontend.FormatPos(glob.Init.Pos()), kind, glob.Name)
				}
				hasStringInit = true
				stringInit = append([]byte(nil), lit.Value...)
			}

			dataIndex := len(dataBlobs)
			globals[glob.Name] = GlobalInfo{
				DataIndex:            dataIndex,
				TypeName:             resolved,
				Mutable:              glob.Mutable,
				Const:                glob.Const,
				HasStringLiteralInit: hasStringInit,
				StringLiteralInit:    stringInit,
			}

			slots := typeInfo.SlotCount
			if slots <= 0 {
				slots = 1
			}
			slotData := make([][]byte, slots)
			for i := 0; i < slots; i++ {
				slotData[i] = make([]byte, 8)
			}
			buf := slotData[0]
			if glob.Mutable {
				if glob.Init != nil {
					switch resolved {
					case "i32", "u8", "u16", "task.error":
						if err := validateGlobalConstExpr(glob.Init, constValues); err != nil {
							return nil, err
						}
						v, ok := evalGlobalConstI32(glob.Init, constValues)
						if !ok {
							return nil, fmt.Errorf("%s: global var '%s' initializer must be an i32 constant expression", frontend.FormatPos(glob.Init.Pos()), glob.Name)
						}
						if err := validateGlobalIntLikeRange(resolved, glob.Name, "var", glob.Init.Pos(), v); err != nil {
							return nil, err
						}
						binary.LittleEndian.PutUint64(buf, uint64(int64(v)))
					case "bool":
						if err := validateGlobalConstExpr(glob.Init, constValues); err != nil {
							return nil, err
						}
						v, ok := evalGlobalConstBool(glob.Init, constValues)
						if !ok {
							return nil, fmt.Errorf("%s: global var '%s' initializer must be a bool constant expression", frontend.FormatPos(glob.Init.Pos()), glob.Name)
						}
						if v {
							binary.LittleEndian.PutUint64(buf, 1)
						} else {
							binary.LittleEndian.PutUint64(buf, 0)
						}
					case "ptr":
						if err := validateGlobalConstExpr(glob.Init, constValues); err != nil {
							return nil, err
						}
						v, ok := evalGlobalConstI32(glob.Init, constValues)
						if !ok {
							return nil, fmt.Errorf("%s: global var '%s' initializer for type ptr must be a constant 0", frontend.FormatPos(glob.Init.Pos()), glob.Name)
						}
						if v != 0 {
							return nil, fmt.Errorf("%s: global var '%s' of type ptr only supports initializer 0", frontend.FormatPos(glob.Init.Pos()), glob.Name)
						}
						binary.LittleEndian.PutUint64(buf, 0)
					case "str":
						binary.LittleEndian.PutUint64(slotData[0], 0)
						binary.LittleEndian.PutUint64(slotData[1], uint64(len(stringInit)))
					default:
						return nil, fmt.Errorf("%s: unsupported global type '%s'", frontend.FormatPos(glob.At), resolved)
					}
				}
				dataBlobs = append(dataBlobs, slotData...)
				continue
			}
			if glob.Init == nil {
				switch resolved {
				case "ptr":
					binary.LittleEndian.PutUint64(buf, 0)
				case "i32", "u8", "u16", "task.error":
					binary.LittleEndian.PutUint64(buf, 0)
					constValues[glob.Name] = globalConstValue{TypeName: resolved, I32: 0}
				case "bool":
					binary.LittleEndian.PutUint64(buf, 0)
					constValues[glob.Name] = globalConstValue{TypeName: "bool", Bool: false}
				case "str":
					binary.LittleEndian.PutUint64(slotData[0], 0)
					binary.LittleEndian.PutUint64(slotData[1], 0)
				default:
					return nil, fmt.Errorf("%s: unsupported global type '%s'", frontend.FormatPos(glob.At), resolved)
				}
				dataBlobs = append(dataBlobs, slotData...)
				continue
			}
			switch resolved {
			case "ptr":
				if !isNullPtrLiteral(glob.Init) {
					return nil, fmt.Errorf("%s: global val '%s' of type ptr only supports initializer 0", frontend.FormatPos(glob.Init.Pos()), glob.Name)
				}
				binary.LittleEndian.PutUint64(buf, 0)
			case "i32", "u8", "u16", "task.error":
				if err := validateGlobalConstExpr(glob.Init, constValues); err != nil {
					return nil, err
				}
				v, ok := evalGlobalConstI32(glob.Init, constValues)
				if !ok {
					return nil, fmt.Errorf("%s: global val '%s' initializer must be an i32 constant expression", frontend.FormatPos(glob.Init.Pos()), glob.Name)
				}
				if err := validateGlobalIntLikeRange(resolved, glob.Name, "val", glob.Init.Pos(), v); err != nil {
					return nil, err
				}
				binary.LittleEndian.PutUint64(buf, uint64(int64(v)))
				constValues[glob.Name] = globalConstValue{TypeName: resolved, I32: v}
			case "bool":
				if err := validateGlobalConstExpr(glob.Init, constValues); err != nil {
					return nil, err
				}
				v, ok := evalGlobalConstBool(glob.Init, constValues)
				if !ok {
					return nil, fmt.Errorf("%s: global val '%s' initializer must be a bool constant expression", frontend.FormatPos(glob.Init.Pos()), glob.Name)
				}
				if v {
					binary.LittleEndian.PutUint64(buf, 1)
				} else {
					binary.LittleEndian.PutUint64(buf, 0)
				}
				constValues[glob.Name] = globalConstValue{TypeName: "bool", Bool: v}
			case "str":
				binary.LittleEndian.PutUint64(slotData[0], 0)
				binary.LittleEndian.PutUint64(slotData[1], uint64(len(stringInit)))
			default:
				return nil, fmt.Errorf("%s: unsupported global type '%s'", frontend.FormatPos(glob.At), resolved)
			}
			dataBlobs = append(dataBlobs, slotData...)
		}

		checked.GlobalsByModule[module] = globals
		checked.GlobalDataByModule[module] = dataBlobs
	}

	for _, file := range world.Files {
		module := file.Module
		imports, err := collectImportAliases(file)
		if err != nil {
			return nil, err
		}
		for _, fn := range file.Funcs {
			fullName := checkedFuncFullName(module, fn)
			if fn.ExportName != "" {
				if fn.ExportName == "core" || strings.HasPrefix(fn.ExportName, "core.") {
					return nil, fmt.Errorf("%s: @export name must not use the 'core.' namespace", frontend.FormatPos(fn.Pos))
				}
				if strings.HasPrefix(fn.ExportName, "__tetra_") && !strings.HasPrefix(module, "__") {
					return nil, fmt.Errorf("%s: @export name '%s' is reserved for internal runtime modules", frontend.FormatPos(fn.Pos), fn.ExportName)
				}
				if other, exists := exportedSymbols[fn.ExportName]; exists {
					return nil, fmt.Errorf("%s: duplicate @export name '%s' (already used by '%s')", frontend.FormatPos(fn.Pos), fn.ExportName, other)
				}
				exportedSymbols[fn.ExportName] = fullName
			}
			if _, exists := builtinSigs[fullName]; exists {
				return nil, fmt.Errorf("%s: cannot redefine builtin '%s'", frontend.FormatPos(fn.Pos), fullName)
			}
			if _, exists := checked.FuncSigs[fullName]; exists {
				return nil, fmt.Errorf("duplicate function '%s'", fullName)
			}
			if err := validateSemanticClauses(fn); err != nil {
				return nil, err
			}
			if len(fn.TypeParams) > 0 {
				if err := validateGenericFuncDecl(fn); err != nil {
					return nil, err
				}
				effects, err := normalizeEffects(fn.Uses, fn.Pos)
				if err != nil {
					return nil, err
				}
				genericParamTypes := make(map[string]string, len(fn.Params))
				for _, param := range fn.Params {
					genericParamTypes[param.Name] = genericTypeName(param.Type)
				}
				returnType := genericTypeName(fn.ReturnType)
				throwsType := ""
				if fn.HasThrows {
					throwsType = genericTypeName(fn.Throws)
				}
				if err := validateFunctionPolicyClauses(fn, effects, genericParamTypes, returnType, throwsType); err != nil {
					return nil, err
				}
				policy, err := parseFunctionClausePolicy(fn)
				if err != nil {
					return nil, err
				}
				checked.FuncSigs[fullName] = FuncSig{
					Generic:              true,
					Public:               declarationIsPublic(file, fn.Public),
					HasNoAlloc:           policy.hasNoAlloc,
					HasNoBlock:           policy.hasNoBlock,
					HasRealtime:          policy.hasRealtime,
					ParamNames:           genericParamNames(fn.Params),
					ParamTypes:           genericParamTypeNames(fn.Params),
					ParamFunctionTypes:   genericParamFunctionKinds(fn.Params),
					ParamFunctionParams:  genericParamFunctionParamTypes(fn.Params),
					ParamFunctionReturns: genericParamFunctionReturnTypes(fn.Params),
					ParamFunctionEffects: genericParamFunctionEffectTypes(fn.Params),
					ParamOwnership:       genericParamOwnership(fn.Params),
					ParamSlots:           0,
					ReturnType:           fn.ReturnType.Name,
					ThrowsType:           fn.Throws.Name,
					Async:                fn.Async,
					ReturnSlots:          0,
					ReturnRegionParam:    regionNone,
					ReturnResourceParam:  regionNone,
					ReturnResourcePath:   "",
					Effects:              effects,
				}
				continue
			}
			retName, err := resolveTypeName(&fn.ReturnType, module, imports)
			if err != nil {
				return nil, err
			}
			returnFunctionType := fn.ReturnType.Kind == frontend.TypeRefFunction
			returnFunctionParams := []string(nil)
			returnFunctionReturn := ""
			returnFunctionEffects := []string(nil)
			if returnFunctionType {
				returnFunctionParams, returnFunctionReturn, returnFunctionEffects, err = functionTypeRefSignatureAndEffects(fn.ReturnType, module, imports)
				if err != nil {
					return nil, err
				}
			}
			fn.ReturnType.Name = retName
			retInfo, err := buildType(retName)
			if err != nil {
				return nil, err
			}
			if err := ensureTypeVisible(retName, retInfo, module, fn.ReturnType.At); err != nil {
				return nil, err
			}
			throwsType := ""
			returnSlots := retInfo.SlotCount
			if fn.HasThrows {
				resolvedThrows, err := resolveTypeName(&fn.Throws, module, imports)
				if err != nil {
					return nil, err
				}
				fn.Throws.Name = resolvedThrows
				throwInfo, err := buildType(resolvedThrows)
				if err != nil {
					return nil, err
				}
				if err := ensureTypeVisible(resolvedThrows, throwInfo, module, fn.Throws.At); err != nil {
					return nil, err
				}
				throwsType = resolvedThrows
				returnSlots = throwingReturnSlots(retInfo.SlotCount, throwInfo.SlotCount)
			}
			effects, err := normalizeEffects(fn.Uses, fn.Pos)
			if err != nil {
				return nil, err
			}
			paramTypes := make([]string, 0, len(fn.Params))
			paramNames := make([]string, 0, len(fn.Params))
			paramOwnership := make([]string, 0, len(fn.Params))
			paramSlots := 0
			for i := range fn.Params {
				param := &fn.Params[i]
				resolved, err := resolveTypeName(&param.Type, module, imports)
				if err != nil {
					return nil, err
				}
				param.Type.Name = resolved
				info, err := buildType(resolved)
				if err != nil {
					return nil, err
				}
				if err := ensureTypeVisible(resolved, info, module, param.At); err != nil {
					return nil, err
				}
				paramNames = append(paramNames, param.Name)
				paramTypes = append(paramTypes, resolved)
				paramOwnership = append(paramOwnership, param.Ownership)
				paramSlots += info.SlotCount
			}
			paramTypeByName := make(map[string]string, len(paramNames))
			for i, name := range paramNames {
				paramTypeByName[name] = paramTypes[i]
			}
			if err := validateFunctionPolicyClauses(fn, effects, paramTypeByName, retName, throwsType); err != nil {
				return nil, err
			}
			policy, err := parseFunctionClausePolicy(fn)
			if err != nil {
				return nil, err
			}
			checked.FuncSigs[fullName] = FuncSig{
				Public:                declarationIsPublic(file, fn.Public),
				HasNoAlloc:            policy.hasNoAlloc,
				HasNoBlock:            policy.hasNoBlock,
				HasRealtime:           policy.hasRealtime,
				ParamNames:            paramNames,
				ParamTypes:            paramTypes,
				ParamFunctionTypes:    paramFunctionKinds(fn.Params),
				ParamFunctionParams:   paramFunctionParamTypes(fn.Params),
				ParamFunctionReturns:  paramFunctionReturnTypes(fn.Params),
				ParamFunctionEffects:  paramFunctionEffectTypes(fn.Params),
				ParamOwnership:        paramOwnership,
				ParamSlots:            paramSlots,
				ReturnType:            retName,
				ReturnFunctionType:    returnFunctionType,
				ReturnFunctionParams:  returnFunctionParams,
				ReturnFunctionReturn:  returnFunctionReturn,
				ReturnFunctionEffects: returnFunctionEffects,
				ThrowsType:            throwsType,
				Async:                 fn.Async,
				ReturnSlots:           returnSlots,
				ReturnRegionParam:     regionNone,
				ReturnResourceParam:   initialReturnResourceParam(retName, types),
				ReturnResourcePath:    "",
				Effects:               effects,
			}
		}
	}

	if err := addPublicImportFunctionAliases(world, checked.FuncSigs); err != nil {
		return nil, err
	}

	for _, file := range world.Files {
		module := file.Module
		imports, err := collectImportAliases(file)
		if err != nil {
			return nil, err
		}
		for _, impl := range file.Impls {
			typeName, err := resolveTypeName(&impl.Type, module, imports)
			if err != nil {
				return nil, err
			}
			protoName, err := resolveTypeName(&impl.Protocol, module, imports)
			if err != nil {
				return nil, err
			}
			impl.Type.Name = typeName
			impl.Protocol.Name = protoName
			if _, ok := types[typeName]; !ok {
				return nil, fmt.Errorf("%s: impl target type '%s' is not defined", frontend.FormatPos(impl.Type.At), typeName)
			}
			implKey := typeName + "->" + protoName
			if first, exists := seenImpls[implKey]; exists {
				return nil, fmt.Errorf("%s: duplicate impl conformance '%s: %s' (first at %s)", frontend.FormatPos(impl.At), typeName, protoName, frontend.FormatPos(first))
			}
			seenImpls[implKey] = impl.At
			proto, ok := protocols[protoName]
			if !ok {
				return nil, fmt.Errorf("%s: protocol '%s' is not defined", frontend.FormatPos(impl.Protocol.At), protoName)
			}
			for _, req := range proto.decl.Requirements {
				methodName := typeName + "." + req.Name
				method := findFuncDecl(world, methodName)
				if method == nil {
					return nil, fmt.Errorf("%s: type '%s' is missing protocol requirement '%s'", frontend.FormatPos(impl.At), typeName, req.Name)
				}
				if err := compareProtocolRequirement(typeName, protoName, req, method); err != nil {
					return nil, err
				}
			}
		}
	}

	for name, sig := range builtinSigs {
		sig.Public = true
		checked.FuncSigs[name] = sig
	}
	if err := checkUIDecls(world, &checked, types); err != nil {
		return nil, err
	}

	if len(checked.FuncSigs) == 0 {
		return nil, fmt.Errorf("no functions found")
	}

	funcCount := 0
	for _, file := range world.Files {
		funcCount += len(file.Funcs)
	}
	maxIter := funcCount + 1
	for iter := 0; iter < maxIter; iter++ {
		changed := false
		for _, file := range world.Files {
			module := file.Module
			interfaceModule := world.InterfaceModules[module]
			imports, err := collectImportAliases(file)
			if err != nil {
				return nil, err
			}
			globals := checked.GlobalsByModule[module]
			for _, fn := range file.Funcs {
				fullName := checkedFuncFullName(module, fn)
				if interfaceModule {
					continue
				}
				if len(fn.TypeParams) > 0 {
					continue
				}
				if len(fn.Body) == 0 {
					return nil, fmt.Errorf("function '%s' must have a body", fullName)
				}

				locals := make(map[string]LocalInfo)
				scopeInfo := newScopeInfo()
				slotIndex := 0
				for _, param := range fn.Params {
					if _, exists := locals[param.Name]; exists {
						return nil, fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(param.At), param.Name)
					}
					info, err := buildType(param.Type.Name)
					if err != nil {
						return nil, err
					}
					functionTypeValue := param.Type.Kind == frontend.TypeRefFunction
					functionParamTypes := []string(nil)
					functionReturnType := ""
					functionEffects := []string(nil)
					if functionTypeValue {
						functionParamTypes, functionReturnType, functionEffects, err = functionTypeRefSignatureAndEffects(param.Type, module, imports)
						if err != nil {
							return nil, err
						}
					}
					locals[param.Name] = LocalInfo{
						Base:               slotIndex,
						SlotCount:          info.SlotCount,
						TypeName:           param.Type.Name,
						Mutable:            param.Ownership == "inout",
						FunctionTypeValue:  functionTypeValue,
						FunctionParamTypes: functionParamTypes,
						FunctionReturnType: functionReturnType,
						FunctionEffects:    functionEffects,
					}
					scopeInfo.localScopes[param.Name] = regionNone
					slotIndex += info.SlotCount
				}
				if fields, ok := actorStateFieldsByMethod[fullName]; ok {
					if err := injectActorStateLocals(fields, locals, scopeInfo); err != nil {
						return nil, err
					}
				}
				if err := collectLocals(fn.Body, locals, &slotIndex, checked.FuncSigs, types, module, imports, scopeInfo, globals); err != nil {
					return nil, err
				}
				if !stmtListEndsWithReturnTyped(fn.Body, locals, globals, checked.FuncSigs, types, module, imports) {
					if err := reportStmtListMatchPayloadDiagnostics(fn.Body, module); err != nil {
						return nil, err
					}
					return nil, fmt.Errorf("function '%s' must end with return", fullName)
				}
				if err := reportStmtListMatchPayloadDiagnostics(fn.Body, module); err != nil {
					return nil, err
				}
				state := newRegionState(scopeInfo)
				if fields, ok := actorStateFieldsByMethod[fullName]; ok {
					state.actorStateFields = fields
				}
				initParamRegions(fn.Params, state, types)
				sig := checked.FuncSigs[fullName]
				state.throwType = sig.ThrowsType
				state.async = sig.Async
				effects := newEffectContext(fullName, sig.Effects, fn.Uses, strings.HasPrefix(module, "__"))
				borrowedParams := make(map[string]struct{})
				inoutParams := make(map[string]struct{})
				for _, param := range fn.Params {
					if param.Ownership == "borrow" {
						borrowedParams[param.Name] = struct{}{}
					} else if param.Ownership == "inout" {
						inoutParams[param.Name] = struct{}{}
					}
				}
				analysis := &functionAnalysisState{}
				if err := checkStmts(fn.Body, locals, globals, checked.FuncSigs, types, module, imports, sig.ReturnType, borrowedParams, inoutParams, state, effects, analysis); err != nil {
					return nil, err
				}
				newReturnParam := regionNone
				if state.returnRegionSet && state.returnRegion < regionNone {
					idx, ok := state.paramRegionIndex[state.returnRegion]
					if !ok {
						return nil, fmt.Errorf("%s: return region does not match parameter", frontend.FormatPos(fn.Pos))
					}
					newReturnParam = idx
				}
				if sig.ReturnRegionParam != newReturnParam {
					sig.ReturnRegionParam = newReturnParam
					checked.FuncSigs[fullName] = sig
					changed = true
				}
				newReturnResourceParam := regionNone
				newReturnResourcePath := ""
				if typeContainsResourceHandle(sig.ReturnType, types) && state.returnResourceSet {
					newReturnResourceParam = state.returnResourceParam
					newReturnResourcePath = state.returnResourcePath
				} else if typeContainsResourceHandle(sig.ReturnType, types) && state.returnResourceUnknown {
					newReturnResourceParam = regionUnknown
				}
				if sig.ReturnResourceParam != newReturnResourceParam || sig.ReturnResourcePath != newReturnResourcePath {
					sig.ReturnResourceParam = newReturnResourceParam
					sig.ReturnResourcePath = newReturnResourcePath
					checked.FuncSigs[fullName] = sig
					changed = true
				}
				if sig.TouchesMutableGlobals != analysis.touchesMutableGlobals {
					sig.TouchesMutableGlobals = analysis.touchesMutableGlobals
					checked.FuncSigs[fullName] = sig
					changed = true
				}
				if sig.ReturnFunctionSymbol != analysis.returnFunctionSymbol {
					sig.ReturnFunctionSymbol = analysis.returnFunctionSymbol
					checked.FuncSigs[fullName] = sig
					changed = true
				}
			}
		}
		if !changed {
			break
		}
		if iter == maxIter-1 {
			return nil, fmt.Errorf("region inference did not converge")
		}
	}
	for name, sig := range checked.FuncSigs {
		if typeContainsResourceHandle(sig.ReturnType, types) && sig.ReturnResourceParam == regionUnknown {
			return nil, fmt.Errorf("resource return provenance could not be inferred for function '%s'", name)
		}
	}

	for _, file := range world.Files {
		module := file.Module
		interfaceModule := world.InterfaceModules[module]
		imports, err := collectImportAliases(file)
		if err != nil {
			return nil, err
		}
		globals := checked.GlobalsByModule[module]
		for _, fn := range file.Funcs {
			fullName := checkedFuncFullName(module, fn)
			if interfaceModule {
				continue
			}
			if len(fn.TypeParams) > 0 {
				continue
			}
			if fn.Name == "main" {
				if module != world.EntryModule {
					return nil, fmt.Errorf("%s: main must be in entry module", frontend.FormatPos(fn.Pos))
				}
				if len(fn.Params) != 0 {
					return nil, fmt.Errorf("%s: main must not have parameters", frontend.FormatPos(fn.Pos))
				}
				if checked.FuncSigs[fullName].ReturnType != "i32" {
					return nil, fmt.Errorf("%s: main must return i32", frontend.FormatPos(fn.Pos))
				}
				if checked.FuncSigs[fullName].ThrowsType != "" {
					return nil, fmt.Errorf("%s: main must not throw", frontend.FormatPos(fn.Pos))
				}
				if checked.FuncSigs[fullName].Async {
					return nil, fmt.Errorf("%s: main must not be async", frontend.FormatPos(fn.Pos))
				}
				checked.MainIndex = len(checked.Funcs)
				checked.MainName = fullName
			}
			locals := make(map[string]LocalInfo)
			scopeInfo := newScopeInfo()
			slotIndex := 0
			for _, param := range fn.Params {
				if _, exists := locals[param.Name]; exists {
					return nil, fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(param.At), param.Name)
				}
				info, err := buildType(param.Type.Name)
				if err != nil {
					return nil, err
				}
				functionTypeValue := param.Type.Kind == frontend.TypeRefFunction
				functionParamTypes := []string(nil)
				functionReturnType := ""
				functionEffects := []string(nil)
				if functionTypeValue {
					functionParamTypes, functionReturnType, functionEffects, err = functionTypeRefSignatureAndEffects(param.Type, module, imports)
					if err != nil {
						return nil, err
					}
				}
				locals[param.Name] = LocalInfo{
					Base:               slotIndex,
					SlotCount:          info.SlotCount,
					TypeName:           param.Type.Name,
					Mutable:            false,
					FunctionTypeValue:  functionTypeValue,
					FunctionParamTypes: functionParamTypes,
					FunctionReturnType: functionReturnType,
					FunctionEffects:    functionEffects,
				}
				scopeInfo.localScopes[param.Name] = regionNone
				slotIndex += info.SlotCount
			}
			if fields, ok := actorStateFieldsByMethod[fullName]; ok {
				if err := injectActorStateLocals(fields, locals, scopeInfo); err != nil {
					return nil, err
				}
			}
			if err := collectLocals(fn.Body, locals, &slotIndex, checked.FuncSigs, types, module, imports, scopeInfo, globals); err != nil {
				return nil, err
			}
			sig := checked.FuncSigs[fullName]
			localSlots := slotIndex
			if sig.ThrowsType != "" {
				throwInfo, err := ensureTypeInfo(sig.ThrowsType, types)
				if err != nil {
					return nil, err
				}
				localSlots += throwingScratchSlots(throwInfo.SlotCount)
			}
			var actorState map[string]ActorStateField
			if fields, ok := actorStateFieldsByMethod[fullName]; ok {
				actorState = make(map[string]ActorStateField, len(fields))
				for name, value := range fields {
					actorState[name] = value
				}
			}
			checked.Funcs = append(checked.Funcs, CheckedFunc{
				Name:        fullName,
				Module:      module,
				Decl:        fn,
				Locals:      locals,
				ActorState:  actorState,
				LocalSlots:  localSlots,
				ParamSlots:  sig.ParamSlots,
				ReturnType:  sig.ReturnType,
				ThrowsType:  sig.ThrowsType,
				Async:       sig.Async,
				ReturnSlots: sig.ReturnSlots,
			})
		}
	}

	if checked.MainIndex == -1 {
		if opt.RequireMain {
			return nil, fmt.Errorf("missing main")
		}
	}

	return &checked, nil
}

func validateCapsuleDecls(file *frontend.FileAST) error {
	if file == nil {
		return nil
	}
	for _, capsule := range file.Capsules {
		if capsule == nil {
			continue
		}
		seen := make(map[string]struct{}, len(capsule.Entries))
		for _, entry := range capsule.Entries {
			if _, exists := seen[entry.Key]; exists {
				return fmt.Errorf("%s: duplicate capsule metadata key '%s'", frontend.FormatPos(entry.At), entry.Key)
			}
			seen[entry.Key] = struct{}{}
			if !isCapsuleMetadataKey(entry.Key) {
				return fmt.Errorf("%s: invalid capsule metadata key '%s'", frontend.FormatPos(entry.At), entry.Key)
			}
			if !isCapsuleMetadataLiteral(entry.Value) {
				return fmt.Errorf("%s: capsule metadata value for key '%s' must be a literal (string/number/bool)", frontend.FormatPos(entry.At), entry.Key)
			}
		}
	}
	return nil
}

func isCapsuleMetadataLiteral(expr frontend.Expr) bool {
	switch expr.(type) {
	case *frontend.StringLitExpr, *frontend.NumberExpr, *frontend.BoolLitExpr:
		return true
	default:
		return false
	}
}

func isCapsuleMetadataKey(key string) bool {
	if key == "" {
		return false
	}
	parts := strings.Split(key, ".")
	for _, part := range parts {
		if !isCapsuleKeySegment(part) {
			return false
		}
	}
	return true
}

func isCapsuleKeySegment(seg string) bool {
	if seg == "" {
		return false
	}
	for i, r := range seg {
		switch {
		case i == 0 && r >= 'a' && r <= 'z':
			continue
		case i > 0 && r >= 'a' && r <= 'z':
			continue
		case i > 0 && r >= '0' && r <= '9':
			continue
		case i > 0 && r == '_':
			continue
		default:
			return false
		}
	}
	return true
}

func stmtListEndsWithReturn(stmts []frontend.Stmt) bool {
	if len(stmts) == 0 {
		return false
	}
	return stmtEndsWithReturn(stmts[len(stmts)-1])
}

func injectActorStateLocals(fields map[string]ActorStateField, locals map[string]LocalInfo, scopes *scopeInfo) error {
	for name, field := range fields {
		if _, exists := locals[name]; exists {
			return fmt.Errorf("duplicate local '%s'", name)
		}
		locals[name] = LocalInfo{
			Base:           -1,
			SlotCount:      1,
			TypeName:       field.TypeName,
			Mutable:        field.Mutable,
			Const:          field.Const,
			ActorField:     true,
			ActorFieldSlot: field.Slot,
			ActorFieldInit: field.Init,
		}
		if scopes != nil && scopes.localScopes != nil {
			scopes.localScopes[name] = regionNone
		}
	}
	return nil
}

func normalizeExtensionMethodNames(world *module.World) error {
	for _, file := range world.Files {
		imports, err := collectImportAliases(file)
		if err != nil {
			return err
		}
		for _, fn := range file.Funcs {
			if fn.ExtensionOf == "" {
				continue
			}
			targetRef := frontend.TypeRef{At: fn.Pos, Kind: frontend.TypeRefNamed, Name: fn.ExtensionOf}
			resolvedTarget, err := resolveTypeName(&targetRef, file.Module, imports)
			if err != nil {
				return err
			}
			methodName := extensionMethodNamePart(fn.Name)
			if methodName == "" {
				return fmt.Errorf("%s: invalid extension method name '%s'", frontend.FormatPos(fn.Pos), fn.Name)
			}
			fn.ExtensionOf = resolvedTarget
			fn.Name = resolvedTarget + "." + methodName
		}
	}
	return nil
}

func extensionMethodNamePart(name string) string {
	if idx := strings.LastIndex(name, "."); idx >= 0 && idx+1 < len(name) {
		return name[idx+1:]
	}
	return name
}

func checkedFuncFullName(module string, fn *frontend.FuncDecl) string {
	if fn != nil && fn.ExtensionOf != "" {
		return fn.Name
	}
	return qualifyName(module, fn.Name)
}

func addPublicImportFunctionAliases(world *module.World, funcs map[string]FuncSig) error {
	for _, file := range world.Files {
		for _, imp := range file.Imports {
			if !imp.Public || len(imp.Items) == 0 {
				continue
			}
			for _, item := range imp.Items {
				target := qualifyName(imp.Path, item)
				sig, ok := funcs[target]
				if !ok {
					continue
				}
				if err := ensureFuncVisible(target, sig, file.Module, imp.At); err != nil {
					return err
				}
				alias := qualifyName(file.Module, item)
				if _, exists := funcs[alias]; exists && alias != target {
					return fmt.Errorf("%s: re-export '%s' conflicts with function '%s'", frontend.FormatPos(imp.At), item, alias)
				}
				sig.Public = true
				funcs[alias] = sig
			}
		}
	}
	return nil
}

func initialReturnResourceParam(returnType string, types map[string]*TypeInfo) int {
	if typeContainsResourceHandle(returnType, types) {
		return regionUnknown
	}
	return regionNone
}

func stmtListEndsWithReturnTyped(
	stmts []frontend.Stmt,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) bool {
	if len(stmts) == 0 {
		return false
	}
	return stmtEndsWithReturnTyped(stmts[len(stmts)-1], locals, globals, funcs, types, module, imports)
}

func stmtEndsWithReturnTyped(
	stmt frontend.Stmt,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) bool {
	switch s := stmt.(type) {
	case *frontend.ReturnStmt, *frontend.ThrowStmt:
		return true
	case *frontend.IfStmt:
		return len(s.Then) > 0 && len(s.Else) > 0 &&
			stmtListEndsWithReturnTyped(s.Then, locals, globals, funcs, types, module, imports) &&
			stmtListEndsWithReturnTyped(s.Else, locals, globals, funcs, types, module, imports)
	case *frontend.IfLetStmt:
		return len(s.Then) > 0 && len(s.Else) > 0 &&
			stmtListEndsWithReturnTyped(s.Then, locals, globals, funcs, types, module, imports) &&
			stmtListEndsWithReturnTyped(s.Else, locals, globals, funcs, types, module, imports)
	case *frontend.MatchStmt:
		for _, c := range s.Cases {
			if !stmtListEndsWithReturnTyped(c.Body, locals, globals, funcs, types, module, imports) {
				return false
			}
		}
		if matchHasDefault(s) || matchHasCompleteOptionalPatterns(s) {
			return true
		}
		return matchHasCompleteEnumPatterns(s, locals, globals, funcs, types, module, imports)
	case *frontend.UnsafeStmt:
		return stmtListEndsWithReturnTyped(s.Body, locals, globals, funcs, types, module, imports)
	default:
		return false
	}
}

func matchHasDefault(s *frontend.MatchStmt) bool {
	for _, c := range s.Cases {
		if c.Default && c.Guard == nil {
			return true
		}
	}
	return false
}

func matchHasCompleteOptionalPatterns(s *frontend.MatchStmt) bool {
	hasNone := false
	hasSome := false
	for _, c := range s.Cases {
		if c.Guard != nil {
			continue
		}
		if c.Default {
			return true
		}
		if _, ok := c.Pattern.(*frontend.NoneLitExpr); ok {
			hasNone = true
		}
		if _, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
			hasSome = true
		}
	}
	return hasNone && hasSome
}

func matchHasCompleteEnumPatterns(
	s *frontend.MatchStmt,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) bool {
	scrutType, err := inferExprTypeForDecl(s.Value, locals, globals, funcs, types, module, imports)
	if err != nil {
		return false
	}
	info, ok := types[scrutType]
	if !ok || info.Kind != TypeEnum || len(info.EnumCases) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(info.EnumCases))
	for i := range s.Cases {
		c := &s.Cases[i]
		if c.Guard != nil {
			continue
		}
		if c.Default {
			return true
		}
		switch pat := c.Pattern.(type) {
		case *frontend.FieldAccessExpr:
			caseType, caseName, ok := bareEnumPatternTypeAndCase(pat, module, imports)
			if !ok || caseType != scrutType {
				return false
			}
			caseInfo, ok := info.CaseMap[caseName]
			if !ok || len(caseInfo.PayloadTypes) != 0 {
				return false
			}
			pat.EnumType = scrutType
			pat.EnumOrdinal = caseInfo.Ordinal
			seen[caseName] = struct{}{}
		case *frontend.EnumCasePatternExpr:
			caseType, caseInfo, found, err := resolveEnumCasePattern(pat, types, module, imports)
			if err != nil || !found || caseType != scrutType {
				return false
			}
			if err := validateEnumCasePatternPayload(pat, caseType, caseInfo, module); err != nil {
				return false
			}
			seen[pat.CaseName] = struct{}{}
		default:
			return false
		}
	}
	for _, enumCase := range info.EnumCases {
		if _, ok := seen[enumCase.Name]; !ok {
			return false
		}
	}
	return true
}

func matchExprHasCompleteOptionalPatterns(e *frontend.MatchExpr) bool {
	seenNone := false
	seenSome := false
	for _, c := range e.Cases {
		if c.Guard != nil {
			continue
		}
		if c.Default {
			return true
		}
		if _, ok := c.Pattern.(*frontend.NoneLitExpr); ok {
			seenNone = true
		}
		if _, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
			seenSome = true
		}
	}
	return seenNone && seenSome
}

func matchExprHasCompleteEnumPatterns(
	e *frontend.MatchExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) bool {
	scrutType, err := inferExprTypeForDecl(e.Value, locals, globals, funcs, types, module, imports)
	if err != nil {
		return false
	}
	info, ok := types[scrutType]
	if !ok || info.Kind != TypeEnum || len(info.EnumCases) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(info.EnumCases))
	for i := range e.Cases {
		c := &e.Cases[i]
		if c.Guard != nil {
			continue
		}
		if c.Default {
			return true
		}
		switch pat := c.Pattern.(type) {
		case *frontend.FieldAccessExpr:
			caseType, caseName, ok := bareEnumPatternTypeAndCase(pat, module, imports)
			if !ok || caseType != scrutType {
				return false
			}
			caseInfo, ok := info.CaseMap[caseName]
			if !ok || len(caseInfo.PayloadTypes) != 0 {
				return false
			}
			pat.EnumType = scrutType
			pat.EnumOrdinal = caseInfo.Ordinal
			seen[caseName] = struct{}{}
		case *frontend.EnumCasePatternExpr:
			caseType, caseInfo, found, err := resolveEnumCasePattern(pat, types, module, imports)
			if err != nil || !found || caseType != scrutType {
				return false
			}
			if err := validateEnumCasePatternPayload(pat, caseType, caseInfo, module); err != nil {
				return false
			}
			seen[pat.CaseName] = struct{}{}
		default:
			return false
		}
	}
	for _, enumCase := range info.EnumCases {
		if _, ok := seen[enumCase.Name]; !ok {
			return false
		}
	}
	return true
}

func inferMatchExprType(
	e *frontend.MatchExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (string, error) {
	scrutType, err := inferExprTypeForDecl(e.Value, locals, globals, funcs, types, module, imports)
	if err != nil {
		return "", err
	}
	resultType := ""
	for _, c := range e.Cases {
		armLocals := cloneLocalMap(locals)
		if !c.Default {
			if err := bindMatchPatternLocalsForInference(c.Pattern, scrutType, armLocals, types, module, imports); err != nil {
				return "", err
			}
		}
		armType, err := inferExprTypeForDecl(c.Value, armLocals, globals, funcs, types, module, imports)
		if err != nil {
			return "", err
		}
		if resultType == "" {
			resultType = armType
			continue
		}
		if !typesCompatibleWithNullPtr(resultType, armType, c.Value) {
			return "", fmt.Errorf("match expression case type mismatch: expected '%s', got '%s'", resultType, armType)
		}
	}
	if resultType == "" {
		return "", fmt.Errorf("match expression requires at least one case")
	}
	e.ResultType = resultType
	return resultType, nil
}

func inferCatchExprType(
	e *frontend.CatchExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) (string, error) {
	call, ok := e.Call.(*frontend.CallExpr)
	if !ok {
		return "", fmt.Errorf("catch expects a throwing function call")
	}
	if builtin, ok := ResolveBuiltinAlias(call.Name); ok && (builtin == "core.task_join_i32_typed" || builtin == "core.task_join_group_i32_typed") {
		if len(call.TypeArgs) != 1 {
			return "", fmt.Errorf("task_join_i32_typed expects one explicit error type argument")
		}
		errorType, err := resolveTypeName(&call.TypeArgs[0], module, imports)
		if err != nil {
			return "", err
		}
		if err := validateTypedTaskErrorType(errorType, types, call.TypeArgs[0].At); err != nil {
			return "", err
		}
		call.TypeArgs[0].Name = errorType
		e.ErrorType = errorType
		e.ResultType = "i32"
		for _, c := range e.Cases {
			armLocals := cloneLocalMap(locals)
			if !c.Default {
				if err := bindMatchPatternLocalsForInference(c.Pattern, errorType, armLocals, types, module, imports); err != nil {
					return "", err
				}
			}
			armType, err := inferExprTypeForDecl(c.Value, armLocals, globals, funcs, types, module, imports)
			if err != nil {
				return "", err
			}
			if !typesCompatibleWithNullPtr("i32", armType, c.Value) {
				return "", fmt.Errorf("catch expression case type mismatch: expected 'i32', got '%s'", armType)
			}
		}
		return "i32", nil
	}
	sig, err := resolveCallSigForInference(call, funcs, module, imports)
	if err != nil {
		return "", err
	}
	if sig.ThrowsType == "" {
		return "", fmt.Errorf("catch expects a throwing function call")
	}
	e.ErrorType = sig.ThrowsType
	e.ResultType = sig.ReturnType
	for _, c := range e.Cases {
		armLocals := cloneLocalMap(locals)
		if !c.Default {
			if err := bindMatchPatternLocalsForInference(c.Pattern, sig.ThrowsType, armLocals, types, module, imports); err != nil {
				return "", err
			}
		}
		armType, err := inferExprTypeForDecl(c.Value, armLocals, globals, funcs, types, module, imports)
		if err != nil {
			return "", err
		}
		if !typesCompatibleWithNullPtr(sig.ReturnType, armType, c.Value) {
			return "", fmt.Errorf("catch expression case type mismatch: expected '%s', got '%s'", sig.ReturnType, armType)
		}
	}
	return sig.ReturnType, nil
}

func resolveCallSigForInference(call *frontend.CallExpr, funcs map[string]FuncSig, module string, imports map[string]string) (FuncSig, error) {
	resolved := ""
	if builtin, ok := ResolveBuiltinAlias(call.Name); ok {
		resolved = builtin
	} else if _, ok := funcs[call.Name]; ok {
		resolved = call.Name
	} else {
		name, err := resolveCallName(call.Name, module, imports, call.At)
		if err != nil {
			return FuncSig{}, err
		}
		resolved = name
	}
	sig, ok := funcs[resolved]
	if !ok {
		return FuncSig{}, fmt.Errorf("unknown function '%s'", resolved)
	}
	if sig.Generic {
		return FuncSig{}, fmt.Errorf("generic function '%s' could not be monomorphized; use inferable value arguments", call.Name)
	}
	return sig, nil
}

func catchPatternType(
	pattern frontend.Expr,
	errorType string,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
) (string, error) {
	info, ok := types[errorType]
	if !ok {
		return "", fmt.Errorf("unknown type '%s'", errorType)
	}
	switch pat := pattern.(type) {
	case *frontend.SomePatternExpr:
		if info.Kind != TypeOptional {
			return "", fmt.Errorf("%s: some pattern requires optional catch value", frontend.FormatPos(pat.At))
		}
		return optionalSomePatternType, nil
	case *frontend.EnumCasePatternExpr:
		caseType, caseInfo, found, err := resolveEnumCasePattern(pat, types, module, imports)
		if err != nil {
			return "", err
		}
		if !found {
			return "", fmt.Errorf("%s: unknown enum pattern '%s.%s'", frontend.FormatPos(pat.At), pat.TypeName, pat.CaseName)
		}
		if err := validateEnumCasePatternPayload(pat, caseType, caseInfo, module); err != nil {
			return "", err
		}
		return caseType, nil
	default:
		patType, _, err := checkExprWithEffects(pattern, locals, globals, funcs, types, module, imports, state, effects, analysis)
		return patType, err
	}
}

func catchExprHasCompleteOptionalPatterns(e *frontend.CatchExpr, errorType string, types map[string]*TypeInfo) bool {
	info, ok := types[errorType]
	if !ok || info.Kind != TypeOptional {
		return false
	}
	seenSome := false
	seenNone := false
	for _, c := range e.Cases {
		if c.Guard != nil {
			continue
		}
		if c.Default {
			return true
		}
		if _, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
			seenSome = true
			continue
		}
		if _, ok := c.Pattern.(*frontend.NoneLitExpr); ok {
			seenNone = true
			continue
		}
		return false
	}
	return seenSome && seenNone
}

func catchExprHasCompleteEnumPatterns(
	e *frontend.CatchExpr,
	errorType string,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) bool {
	info, ok := types[errorType]
	if !ok || info.Kind != TypeEnum || len(info.EnumCases) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(info.EnumCases))
	for i := range e.Cases {
		c := &e.Cases[i]
		if c.Guard != nil {
			continue
		}
		if c.Default {
			return true
		}
		switch pat := c.Pattern.(type) {
		case *frontend.FieldAccessExpr:
			caseType, caseName, ok := bareEnumPatternTypeAndCase(pat, module, imports)
			if !ok || caseType != errorType {
				return false
			}
			caseInfo, ok := info.CaseMap[caseName]
			if !ok || len(caseInfo.PayloadTypes) != 0 {
				return false
			}
			pat.EnumType = errorType
			pat.EnumOrdinal = caseInfo.Ordinal
			seen[caseName] = struct{}{}
		case *frontend.EnumCasePatternExpr:
			caseType, caseInfo, found, err := resolveEnumCasePattern(pat, types, module, imports)
			if err != nil || !found || caseType != errorType {
				return false
			}
			if err := validateEnumCasePatternPayload(pat, caseType, caseInfo, module); err != nil {
				return false
			}
			seen[pat.CaseName] = struct{}{}
		default:
			return false
		}
	}
	for _, enumCase := range info.EnumCases {
		if _, ok := seen[enumCase.Name]; !ok {
			return false
		}
	}
	return true
}

func cloneLocalMap(locals map[string]LocalInfo) map[string]LocalInfo {
	out := make(map[string]LocalInfo, len(locals))
	for name, info := range locals {
		out[name] = info
	}
	return out
}

func bindMatchPatternLocalsForInference(
	pattern frontend.Expr,
	scrutType string,
	locals map[string]LocalInfo,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
) error {
	info, ok := types[scrutType]
	if !ok {
		return fmt.Errorf("unknown type '%s'", scrutType)
	}
	switch pat := pattern.(type) {
	case *frontend.SomePatternExpr:
		if info.Kind != TypeOptional {
			return fmt.Errorf("%s: some pattern requires optional match value", frontend.FormatPos(pat.At))
		}
		elemInfo, err := ensureTypeInfo(info.ElemType, types)
		if err != nil {
			return err
		}
		locals[pat.Name] = LocalInfo{SlotCount: elemInfo.SlotCount, TypeName: info.ElemType}
	case *frontend.EnumCasePatternExpr:
		caseType, caseInfo, found, err := resolveEnumCasePattern(pat, types, module, imports)
		if err != nil {
			return err
		}
		if !found || caseType != scrutType {
			return fmt.Errorf("%s: enum pattern type mismatch", frontend.FormatPos(pat.At))
		}
		if err := validateEnumCasePatternPayload(pat, caseType, caseInfo, module); err != nil {
			return err
		}
		for i, binding := range pat.Bindings {
			locals[binding] = LocalInfo{SlotCount: caseInfo.PayloadSlots[i], TypeName: caseInfo.PayloadTypes[i]}
		}
	}
	return nil
}

func validateEnumCasePatternPayload(pattern *frontend.EnumCasePatternExpr, caseType string, caseInfo EnumCaseInfo, module string) error {
	want := len(caseInfo.PayloadTypes)
	got := len(pattern.Bindings)
	if got > 0 && !pattern.HasPayload {
		pattern.HasPayload = true
	}
	if want == 0 {
		if pattern.HasPayload {
			return fmt.Errorf("%s: enum case '%s.%s' has no payload; use '%s.%s'", frontend.FormatPos(pattern.At), displayTypeName(caseType, module), pattern.CaseName, displayTypeName(caseType, module), pattern.CaseName)
		}
		if got != 0 {
			return fmt.Errorf("%s: enum case '%s.%s' pattern expects 0 binding(s), got %d", frontend.FormatPos(pattern.At), displayTypeName(caseType, module), pattern.CaseName, got)
		}
		return nil
	}
	if !pattern.HasPayload {
		return fmt.Errorf("%s: enum case '%s.%s' carries %d payload value(s); use '%s.%s(%s)'", frontend.FormatPos(pattern.At), displayTypeName(caseType, module), pattern.CaseName, want, displayTypeName(caseType, module), pattern.CaseName, placeholderBindingList(want))
	}
	if got != want {
		return fmt.Errorf("%s: enum case '%s.%s' pattern expects %d binding(s), got %d", frontend.FormatPos(pattern.At), displayTypeName(caseType, module), pattern.CaseName, want, got)
	}
	return nil
}

func placeholderBindingList(n int) string {
	if n <= 0 {
		return ""
	}
	bindings := make([]string, n)
	for i := range bindings {
		bindings[i] = fmt.Sprintf("value%d", i+1)
	}
	return strings.Join(bindings, ", ")
}

func stmtEndsWithReturn(stmt frontend.Stmt) bool {
	switch s := stmt.(type) {
	case *frontend.ReturnStmt:
		return true
	case *frontend.ThrowStmt:
		return true
	case *frontend.IfStmt:
		return len(s.Then) > 0 && len(s.Else) > 0 && stmtListEndsWithReturn(s.Then) && stmtListEndsWithReturn(s.Else)
	case *frontend.IfLetStmt:
		return len(s.Then) > 0 && len(s.Else) > 0 && stmtListEndsWithReturn(s.Then) && stmtListEndsWithReturn(s.Else)
	case *frontend.MatchStmt:
		hasDefault := false
		hasNone := false
		hasSome := false
		for _, c := range s.Cases {
			if c.Default {
				hasDefault = true
			} else if _, ok := c.Pattern.(*frontend.NoneLitExpr); ok {
				hasNone = true
			} else if _, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
				hasSome = true
			}
			if !stmtListEndsWithReturn(c.Body) {
				return false
			}
		}
		return hasDefault || (hasNone && hasSome)
	case *frontend.UnsafeStmt:
		return stmtListEndsWithReturn(s.Body)
	default:
		return false
	}
}

func validateGenericFuncDecl(fn *frontend.FuncDecl) error {
	if len(fn.TypeParams) == 0 {
		return nil
	}
	params := map[string]struct{}{}
	for _, name := range fn.TypeParams {
		params[name] = struct{}{}
	}
	for _, bound := range fn.TypeParamBounds {
		if _, ok := params[bound.Name]; !ok {
			return fmt.Errorf("%s: generic bound references unknown type parameter '%s'", frontend.FormatPos(bound.At), bound.Name)
		}
		if bound.Bound.Kind != frontend.TypeRefNamed || len(bound.Bound.TypeArgs) > 0 {
			return fmt.Errorf("%s: generic bound for '%s' must name a protocol", frontend.FormatPos(bound.Bound.At), bound.Name)
		}
	}
	if err := validateGenericTypeRef(fn.ReturnType, params); err != nil {
		return fmt.Errorf("%s: %v", frontend.FormatPos(fn.ReturnType.At), err)
	}
	if fn.HasThrows {
		if err := validateGenericTypeRef(fn.Throws, params); err != nil {
			return fmt.Errorf("%s: %v", frontend.FormatPos(fn.Throws.At), err)
		}
	}
	for _, param := range fn.Params {
		if err := validateGenericTypeRef(param.Type, params); err != nil {
			return fmt.Errorf("%s: %v", frontend.FormatPos(param.At), err)
		}
	}
	return nil
}

func validateSemanticClauses(fn *frontend.FuncDecl) error {
	seen := map[string]frontend.Position{}
	for _, clause := range fn.SemanticClauses {
		if first, exists := seen[clause.Name]; exists {
			return fmt.Errorf("%s: duplicate semantic clause '%s' (first at %s)", frontend.FormatPos(clause.At), clause.Name, frontend.FormatPos(first))
		}
		seen[clause.Name] = clause.At
		switch clause.Name {
		case "noalloc", "noblock", "realtime", "privacy":
			if clause.Value != nil {
				return fmt.Errorf("%s: semantic clause '%s' does not take arguments", frontend.FormatPos(clause.At), clause.Name)
			}
		case "nothrow":
			if clause.Value != nil {
				return fmt.Errorf("%s: semantic clause 'nothrow' does not take arguments", frontend.FormatPos(clause.At))
			}
			if fn.HasThrows {
				return fmt.Errorf("%s: semantic clause 'nothrow' conflicts with explicit throws type", frontend.FormatPos(clause.At))
			}
		case "budget":
			if clause.Value == nil {
				return fmt.Errorf("%s: semantic clause 'budget' requires an integer argument", frontend.FormatPos(clause.At))
			}
			v, ok := constI32(clause.Value)
			if !ok {
				return fmt.Errorf("%s: semantic clause 'budget' expects an integer constant argument", frontend.FormatPos(clause.Value.Pos()))
			}
			if v < 0 {
				return fmt.Errorf("%s: semantic clause 'budget' requires a non-negative value", frontend.FormatPos(clause.Value.Pos()))
			}
		case "consent":
			if clause.Value == nil {
				return fmt.Errorf("%s: semantic clause 'consent' requires a token parameter name", frontend.FormatPos(clause.At))
			}
			if _, ok := clause.Value.(*frontend.IdentExpr); !ok {
				return fmt.Errorf("%s: semantic clause 'consent' expects an identifier argument", frontend.FormatPos(clause.Value.Pos()))
			}
		default:
			return fmt.Errorf("%s: unknown semantic clause '%s'", frontend.FormatPos(clause.At), clause.Name)
		}
	}
	return nil
}

type functionClausePolicy struct {
	hasNoAlloc   bool
	hasNoBlock   bool
	hasRealtime  bool
	hasBudget    bool
	hasPrivacy   bool
	consentParam string
}

func parseFunctionClausePolicy(fn *frontend.FuncDecl) (functionClausePolicy, error) {
	policy := functionClausePolicy{}
	for _, clause := range fn.SemanticClauses {
		switch clause.Name {
		case "noalloc":
			policy.hasNoAlloc = true
		case "noblock":
			policy.hasNoBlock = true
		case "realtime":
			policy.hasRealtime = true
		case "budget":
			policy.hasBudget = true
		case "privacy":
			policy.hasPrivacy = true
		case "consent":
			ident, ok := clause.Value.(*frontend.IdentExpr)
			if !ok {
				return functionClausePolicy{}, fmt.Errorf("%s: semantic clause 'consent' expects an identifier argument", frontend.FormatPos(clause.At))
			}
			policy.consentParam = ident.Name
		}
	}
	return policy, nil
}

func validateFunctionPolicyClauses(
	fn *frontend.FuncDecl,
	effects []string,
	paramTypes map[string]string,
	returnType string,
	throwsType string,
) error {
	policy, err := parseFunctionClausePolicy(fn)
	if err != nil {
		return err
	}
	declaredEffects := effectSet(effects)
	hasEffect := func(name string) bool {
		_, ok := declaredEffects[name]
		return ok
	}

	if hasEffect("budget") && !policy.hasBudget {
		return fmt.Errorf("%s: uses effect 'budget' requires semantic clause 'budget'", frontend.FormatPos(fn.Pos))
	}
	if policy.hasBudget && !hasEffect("budget") {
		return fmt.Errorf("%s: semantic clause 'budget' requires function '%s' to declare uses effect 'budget'", frontend.FormatPos(fn.Pos), fn.Name)
	}
	if policy.hasNoAlloc && hasEffect("alloc") {
		return fmt.Errorf("%s: semantic clause 'noalloc' conflicts with declared effect 'alloc'", frontend.FormatPos(fn.Pos))
	}
	if policy.hasNoBlock {
		if blocked := firstForbiddenEffect(declaredEffects, []string{"actors", "control", "io", "link", "mmio", "runtime"}); blocked != "" {
			return fmt.Errorf("%s: semantic clause 'noblock' conflicts with declared effect '%s'", frontend.FormatPos(fn.Pos), blocked)
		}
	}
	if policy.hasRealtime {
		if !policy.hasNoAlloc {
			return fmt.Errorf("%s: semantic clause 'realtime' requires semantic clause 'noalloc'", frontend.FormatPos(fn.Pos))
		}
		if !policy.hasNoBlock {
			return fmt.Errorf("%s: semantic clause 'realtime' requires semantic clause 'noblock'", frontend.FormatPos(fn.Pos))
		}
		if blocked := firstForbiddenEffect(declaredEffects, []string{"actors", "alloc", "control", "io", "link", "mmio", "runtime"}); blocked != "" {
			return fmt.Errorf("%s: semantic clause 'realtime' conflicts with declared effect '%s'", frontend.FormatPos(fn.Pos), blocked)
		}
	}
	if policy.hasPrivacy && !hasEffect("privacy") {
		return fmt.Errorf("%s: semantic clause 'privacy' requires function '%s' to declare uses effect 'privacy'", frontend.FormatPos(fn.Pos), fn.Name)
	}
	if hasEffect("privacy") && !policy.hasPrivacy {
		return fmt.Errorf("%s: uses effect 'privacy' requires semantic clause 'privacy'", frontend.FormatPos(fn.Pos))
	}

	signatureHasSecret := typeUsesSecret(returnType) || typeUsesSecret(throwsType)
	for _, paramType := range paramTypes {
		if typeUsesSecret(paramType) {
			signatureHasSecret = true
		}
	}
	if signatureHasSecret && !policy.hasPrivacy {
		return fmt.Errorf("%s: secret types in function signature require semantic clause 'privacy'", frontend.FormatPos(fn.Pos))
	}
	if signatureHasSecret && policy.consentParam == "" {
		return fmt.Errorf("%s: secret types in function signature require semantic clause consent(<token>)", frontend.FormatPos(fn.Pos))
	}
	if policy.consentParam != "" {
		if !policy.hasPrivacy {
			return fmt.Errorf("%s: semantic clause 'consent' requires semantic clause 'privacy'", frontend.FormatPos(fn.Pos))
		}
		paramType, ok := paramTypes[policy.consentParam]
		if !ok {
			return fmt.Errorf("%s: semantic clause 'consent' references unknown parameter '%s'", frontend.FormatPos(fn.Pos), policy.consentParam)
		}
		if paramType != "consent.token" {
			return fmt.Errorf("%s: semantic clause 'consent' parameter '%s' must have type consent.token", frontend.FormatPos(fn.Pos), policy.consentParam)
		}
	}
	return nil
}

func firstForbiddenEffect(have map[string]struct{}, forbidden []string) string {
	for _, effect := range forbidden {
		if _, ok := have[effect]; ok {
			return effect
		}
	}
	return ""
}

func typeUsesSecret(typeName string) bool {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" {
		return false
	}
	if strings.HasPrefix(typeName, "secret.") {
		return true
	}
	if strings.HasSuffix(typeName, "?") {
		return typeUsesSecret(strings.TrimSuffix(typeName, "?"))
	}
	if strings.HasPrefix(typeName, "[]") {
		return typeUsesSecret(strings.TrimPrefix(typeName, "[]"))
	}
	return false
}

func findFuncDecl(world *module.World, name string) *frontend.FuncDecl {
	for _, file := range world.Files {
		for _, fn := range file.Funcs {
			if qualifyName(file.Module, fn.Name) == name || fn.Name == name {
				return fn
			}
		}
	}
	return nil
}

func compareProtocolRequirement(typeName, protoName string, req frontend.FuncSigDecl, method *frontend.FuncDecl) error {
	if len(req.TypeParams) != len(method.TypeParams) {
		return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': generic parameter count differs", frontend.FormatPos(method.Pos), method.Name, protoName, req.Name)
	}
	for i := range req.TypeParams {
		if req.TypeParams[i] != method.TypeParams[i] {
			return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': generic parameter %d name differs", frontend.FormatPos(method.Pos), method.Name, protoName, req.Name, i+1)
		}
	}
	if len(req.Params) != len(method.Params) {
		return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': parameter count differs", frontend.FormatPos(method.Pos), method.Name, protoName, req.Name)
	}
	for i := range req.Params {
		if genericTypeName(req.Params[i].Type) != genericTypeName(method.Params[i].Type) {
			return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': parameter %d type differs", frontend.FormatPos(method.Params[i].At), method.Name, protoName, req.Name, i+1)
		}
	}
	if genericTypeName(req.ReturnType) != genericTypeName(method.ReturnType) {
		return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': return type differs", frontend.FormatPos(method.Pos), method.Name, protoName, req.Name)
	}
	if req.Async != method.Async {
		return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': async marker differs", frontend.FormatPos(method.Pos), method.Name, protoName, req.Name)
	}
	if req.HasThrows != method.HasThrows || genericTypeName(req.Throws) != genericTypeName(method.Throws) {
		return fmt.Errorf("%s: method '%s' does not match protocol '%s' requirement '%s': throws type differs", frontend.FormatPos(method.Pos), method.Name, protoName, req.Name)
	}
	reqEffects, err := normalizeEffects(req.Uses, req.At)
	if err != nil {
		return fmt.Errorf("%s: protocol '%s' requirement '%s': %v", frontend.FormatPos(req.At), protoName, req.Name, err)
	}
	methodEffects, err := normalizeEffects(method.Uses, method.Pos)
	if err != nil {
		return err
	}
	missing := missingRequiredEffects(reqEffects, methodEffects)
	if len(missing) > 0 {
		return fmt.Errorf("%s: method '%s' for type '%s' does not match protocol '%s' requirement '%s': missing required effects %s", frontend.FormatPos(method.Pos), method.Name, typeName, protoName, req.Name, strings.Join(missing, ", "))
	}
	return nil
}

func missingRequiredEffects(required []string, declared []string) []string {
	if len(required) == 0 {
		return nil
	}
	have := make(map[string]struct{}, len(declared))
	for _, effect := range declared {
		have[effect] = struct{}{}
	}
	var missing []string
	for _, effect := range required {
		if _, ok := have[effect]; ok {
			continue
		}
		missing = append(missing, effect)
	}
	return missing
}

func resolveProtocolRequirementTypeRef(ref *frontend.TypeRef, module string, imports map[string]string, typeParams map[string]struct{}) (string, bool, error) {
	if ref == nil {
		return "", false, fmt.Errorf("missing type")
	}
	switch ref.Kind {
	case frontend.TypeRefNamed:
		if ref.Name == "" {
			return "", false, fmt.Errorf("missing type name")
		}
		if _, ok := typeParams[ref.Name]; ok {
			if len(ref.TypeArgs) > 0 {
				return "", false, fmt.Errorf("generic type parameter '%s' cannot have type arguments", ref.Name)
			}
			return ref.Name, true, nil
		}
		for i := range ref.TypeArgs {
			argName, _, err := resolveProtocolRequirementTypeRef(&ref.TypeArgs[i], module, imports, typeParams)
			if err != nil {
				return "", false, err
			}
			ref.TypeArgs[i].Name = argName
		}
		resolved, err := resolveTypeName(ref, module, imports)
		if err != nil {
			return "", false, err
		}
		ref.Name = resolved
		return resolved, false, nil
	case frontend.TypeRefSlice, frontend.TypeRefArray, frontend.TypeRefOptional:
		if ref.Elem == nil {
			return "", false, fmt.Errorf("missing element type")
		}
		elemName, elemGeneric, err := resolveProtocolRequirementTypeRef(ref.Elem, module, imports, typeParams)
		if err != nil {
			return "", false, err
		}
		ref.Elem.Name = elemName
		if elemGeneric {
			return genericTypeName(*ref), true, nil
		}
		resolved, err := resolveTypeName(ref, module, imports)
		if err != nil {
			return "", false, err
		}
		return resolved, false, nil
	case frontend.TypeRefFunction:
		anyGeneric := false
		for i := range ref.Params {
			paramName, paramGeneric, err := resolveProtocolRequirementTypeRef(&ref.Params[i], module, imports, typeParams)
			if err != nil {
				return "", false, err
			}
			ref.Params[i].Name = paramName
			anyGeneric = anyGeneric || paramGeneric
		}
		if ref.Return == nil {
			return "", false, fmt.Errorf("missing function return type")
		}
		retName, retGeneric, err := resolveProtocolRequirementTypeRef(ref.Return, module, imports, typeParams)
		if err != nil {
			return "", false, err
		}
		ref.Return.Name = retName
		if _, err := normalizeEffects(ref.Uses, ref.At); err != nil {
			return "", false, err
		}
		anyGeneric = anyGeneric || retGeneric
		if anyGeneric {
			return genericTypeName(*ref), true, nil
		}
		return "ptr", false, nil
	default:
		return "", false, fmt.Errorf("unsupported type reference in protocol requirement")
	}
}

func validateGenericTypeRef(ref frontend.TypeRef, params map[string]struct{}) error {
	switch ref.Kind {
	case frontend.TypeRefNamed:
		if ref.Name == "" {
			return fmt.Errorf("missing type name")
		}
		if _, ok := params[ref.Name]; ok {
			return nil
		}
		if _, ok := canonicalBuiltinType(ref.Name); ok {
			return nil
		}
		if strings.Contains(ref.Name, ".") {
			return nil
		}
		return nil
	case frontend.TypeRefSlice, frontend.TypeRefArray, frontend.TypeRefOptional:
		if ref.Elem == nil {
			return fmt.Errorf("%s: missing element type", frontend.FormatPos(ref.At))
		}
		return validateGenericTypeRef(*ref.Elem, params)
	case frontend.TypeRefFunction:
		for _, param := range ref.Params {
			if err := validateGenericTypeRef(param, params); err != nil {
				return err
			}
		}
		if ref.Return == nil {
			return fmt.Errorf("%s: missing function return type", frontend.FormatPos(ref.At))
		}
		if _, err := normalizeEffects(ref.Uses, ref.At); err != nil {
			return err
		}
		return validateGenericTypeRef(*ref.Return, params)
	default:
		return fmt.Errorf("%s: unsupported generic type reference kind %d", frontend.FormatPos(ref.At), ref.Kind)
	}
}

func genericParamTypeNames(params []frontend.ParamDecl) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		out = append(out, formatGenericTypeRef(param.Type))
	}
	return out
}

func genericParamNames(params []frontend.ParamDecl) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		out = append(out, param.Name)
	}
	return out
}

func genericParamOwnership(params []frontend.ParamDecl) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		out = append(out, param.Ownership)
	}
	return out
}

func genericParamFunctionKinds(params []frontend.ParamDecl) []bool {
	out := make([]bool, 0, len(params))
	for _, param := range params {
		out = append(out, param.Type.Kind == frontend.TypeRefFunction)
	}
	return out
}

func genericParamFunctionParamTypes(params []frontend.ParamDecl) [][]string {
	out := make([][]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction {
			out = append(out, nil)
			continue
		}
		row := make([]string, 0, len(param.Type.Params))
		for _, p := range param.Type.Params {
			row = append(row, formatGenericTypeRef(p))
		}
		out = append(out, row)
	}
	return out
}

func genericParamFunctionReturnTypes(params []frontend.ParamDecl) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction || param.Type.Return == nil {
			out = append(out, "")
			continue
		}
		out = append(out, formatGenericTypeRef(*param.Type.Return))
	}
	return out
}

func genericParamFunctionEffectTypes(params []frontend.ParamDecl) [][]string {
	out := make([][]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction {
			out = append(out, nil)
			continue
		}
		effects, err := normalizeEffects(param.Type.Uses, param.Type.At)
		if err != nil {
			out = append(out, nil)
			continue
		}
		out = append(out, effects)
	}
	return out
}

func paramFunctionKinds(params []frontend.ParamDecl) []bool {
	out := make([]bool, 0, len(params))
	for _, param := range params {
		out = append(out, param.Type.Kind == frontend.TypeRefFunction)
	}
	return out
}

func paramFunctionParamTypes(params []frontend.ParamDecl) [][]string {
	out := make([][]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction {
			out = append(out, nil)
			continue
		}
		row := make([]string, 0, len(param.Type.Params))
		for _, p := range param.Type.Params {
			row = append(row, p.Name)
		}
		out = append(out, row)
	}
	return out
}

func paramFunctionReturnTypes(params []frontend.ParamDecl) []string {
	out := make([]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction || param.Type.Return == nil {
			out = append(out, "")
			continue
		}
		out = append(out, param.Type.Return.Name)
	}
	return out
}

func paramFunctionEffectTypes(params []frontend.ParamDecl) [][]string {
	out := make([][]string, 0, len(params))
	for _, param := range params {
		if param.Type.Kind != frontend.TypeRefFunction {
			out = append(out, nil)
			continue
		}
		effects, err := normalizeEffects(param.Type.Uses, param.Type.At)
		if err != nil {
			out = append(out, nil)
			continue
		}
		out = append(out, effects)
	}
	return out
}

func throwingReturnSlots(successSlots, errorSlots int) int {
	if successSlots == 1 && errorSlots == 1 {
		// Preserve the compact v0.5 layout for existing single-slot typed errors.
		return 2
	}
	return successSlots + errorSlots + 1
}

func throwingScratchSlots(errorSlots int) int {
	if errorSlots <= 0 {
		return 0
	}
	return errorSlots
}

func formatGenericTypeRef(ref frontend.TypeRef) string {
	switch ref.Kind {
	case frontend.TypeRefSlice:
		if ref.Elem == nil {
			return "[]?"
		}
		return "[]" + formatGenericTypeRef(*ref.Elem)
	case frontend.TypeRefArray:
		if ref.Elem == nil {
			return fmt.Sprintf("[%d]?", ref.Len)
		}
		return fmt.Sprintf("[%d]%s", ref.Len, formatGenericTypeRef(*ref.Elem))
	case frontend.TypeRefOptional:
		if ref.Elem == nil {
			return "?"
		}
		return formatGenericTypeRef(*ref.Elem) + "?"
	case frontend.TypeRefFunction:
		parts := make([]string, 0, len(ref.Params))
		for _, param := range ref.Params {
			parts = append(parts, formatGenericTypeRef(param))
		}
		ret := "?"
		if ref.Return != nil {
			ret = formatGenericTypeRef(*ref.Return)
		}
		out := "fn(" + strings.Join(parts, ", ") + ") -> " + ret
		if len(ref.Uses) > 0 {
			out += " uses " + strings.Join(ref.Uses, ", ")
		}
		return out
	default:
		if len(ref.TypeArgs) > 0 {
			args := make([]string, 0, len(ref.TypeArgs))
			for _, arg := range ref.TypeArgs {
				args = append(args, formatGenericTypeRef(arg))
			}
			return ref.Name + "<" + strings.Join(args, ", ") + ">"
		}
		return ref.Name
	}
}

type functionAnalysisState struct {
	touchesMutableGlobals bool
	returnFunctionSymbol  string
}

func validateDeferBodyControl(stmts []frontend.Stmt, loopDepth int) error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.ReturnStmt:
			return fmt.Errorf("%s: return is not allowed in defer", frontend.FormatPos(s.At))
		case *frontend.ThrowStmt:
			return fmt.Errorf("%s: throw is not allowed in defer", frontend.FormatPos(s.At))
		case *frontend.DeferStmt:
			return fmt.Errorf("%s: nested defer is not allowed in defer", frontend.FormatPos(s.At))
		case *frontend.BreakStmt:
			if loopDepth == 0 {
				return fmt.Errorf("%s: break is not allowed in defer outside a cleanup-local loop", frontend.FormatPos(s.At))
			}
		case *frontend.ContinueStmt:
			if loopDepth == 0 {
				return fmt.Errorf("%s: continue is not allowed in defer outside a cleanup-local loop", frontend.FormatPos(s.At))
			}
		case *frontend.IfStmt:
			if err := validateDeferBodyControl(s.Then, loopDepth); err != nil {
				return err
			}
			if err := validateDeferBodyControl(s.Else, loopDepth); err != nil {
				return err
			}
		case *frontend.IfLetStmt:
			if err := validateDeferBodyControl(s.Then, loopDepth); err != nil {
				return err
			}
			if err := validateDeferBodyControl(s.Else, loopDepth); err != nil {
				return err
			}
		case *frontend.WhileStmt:
			if err := validateDeferBodyControl(s.Body, loopDepth+1); err != nil {
				return err
			}
		case *frontend.ForRangeStmt:
			if err := validateDeferBodyControl(s.Body, loopDepth+1); err != nil {
				return err
			}
		case *frontend.MatchStmt:
			for _, c := range s.Cases {
				if err := validateDeferBodyControl(c.Body, loopDepth); err != nil {
					return err
				}
			}
		case *frontend.IslandStmt:
			if err := validateDeferBodyControl(s.Body, loopDepth); err != nil {
				return err
			}
		case *frontend.UnsafeStmt:
			if err := validateDeferBodyControl(s.Body, loopDepth); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkDeferBody(
	body []frontend.Stmt,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	returnType string,
	borrowedParams map[string]struct{},
	inoutParams map[string]struct{},
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
) error {
	savedRegionVars := copyRegionVars(state.regionVars)
	savedUnknownVars := copyBoolMap(state.unknownVars)
	savedUnknownConflicts := copyRegionConflictMap(state.unknownConflicts)
	savedConsumedVars := copyConsumedVars(state.consumedVars)
	savedConsumedResources := copyConsumedResources(state.consumedResources)
	savedResourceVars := copyResourceVars(state.resourceVars)
	savedUnknownResources := copyUnknownResources(state.unknownResources)
	savedFinalizedResources := copyFinalizedResources(state.finalizedResources)
	savedNextResourceID := state.nextResourceID
	savedReturnRegion := state.returnRegion
	savedReturnRegionSet := state.returnRegionSet
	savedLoopDepth := state.loopDepth
	savedUnsafeDepth := state.unsafeDepth
	savedAllowThrowDepth := state.allowThrowDepth
	savedAllowThrowCall := state.allowThrowCall
	savedAllowCatchDepth := state.allowCatchDepth
	savedAllowCatchCall := state.allowCatchCall
	savedAllowAwaitDepth := state.allowAwaitDepth
	savedAllowAwaitCall := state.allowAwaitCall
	defer func() {
		state.regionVars = savedRegionVars
		state.unknownVars = savedUnknownVars
		state.unknownConflicts = savedUnknownConflicts
		state.consumedVars = savedConsumedVars
		state.consumedResources = savedConsumedResources
		state.resourceVars = savedResourceVars
		state.unknownResources = savedUnknownResources
		state.finalizedResources = savedFinalizedResources
		state.nextResourceID = savedNextResourceID
		state.returnRegion = savedReturnRegion
		state.returnRegionSet = savedReturnRegionSet
		state.loopDepth = savedLoopDepth
		state.unsafeDepth = savedUnsafeDepth
		state.allowThrowDepth = savedAllowThrowDepth
		state.allowThrowCall = savedAllowThrowCall
		state.allowCatchDepth = savedAllowCatchDepth
		state.allowCatchCall = savedAllowCatchCall
		state.allowAwaitDepth = savedAllowAwaitDepth
		state.allowAwaitCall = savedAllowAwaitCall
	}()
	state.loopDepth = 0
	return checkStmts(body, locals, globals, funcs, types, module, imports, returnType, borrowedParams, inoutParams, state, effects, analysis)
}

func copyBoolMap(src map[string]bool) map[string]bool {
	if len(src) == 0 {
		return make(map[string]bool)
	}
	dst := make(map[string]bool, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyRegionConflictMap(src map[string]regionConflict) map[string]regionConflict {
	if len(src) == 0 {
		return make(map[string]regionConflict)
	}
	dst := make(map[string]regionConflict, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyConsumedVars(src map[string]frontend.Position) map[string]frontend.Position {
	if len(src) == 0 {
		return make(map[string]frontend.Position)
	}
	dst := make(map[string]frontend.Position, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyConsumedResources(src map[int]frontend.Position) map[int]frontend.Position {
	if len(src) == 0 {
		return make(map[int]frontend.Position)
	}
	dst := make(map[int]frontend.Position, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyResourceVars(src map[string]int) map[string]int {
	if len(src) == 0 {
		return make(map[string]int)
	}
	dst := make(map[string]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyUnknownResources(src map[int]bool) map[int]bool {
	if len(src) == 0 {
		return make(map[int]bool)
	}
	dst := make(map[int]bool, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyFinalizedResources(src map[int]resourceFinalization) map[int]resourceFinalization {
	if len(src) == 0 {
		return make(map[int]resourceFinalization)
	}
	dst := make(map[int]resourceFinalization, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func mergeConsumedVars(a, b map[string]frontend.Position) map[string]frontend.Position {
	if len(a) == 0 && len(b) == 0 {
		return make(map[string]frontend.Position)
	}
	merged := make(map[string]frontend.Position)
	for name, left := range a {
		if right, ok := b[name]; ok {
			merged[name] = earliestPosition(left, right)
			continue
		}
		merged[name] = left
	}
	for name, right := range b {
		if _, exists := merged[name]; !exists {
			merged[name] = right
		}
	}
	return merged
}

func mergeConsumedResources(a, b map[int]frontend.Position) map[int]frontend.Position {
	if len(a) == 0 && len(b) == 0 {
		return make(map[int]frontend.Position)
	}
	merged := make(map[int]frontend.Position)
	for id, left := range a {
		if right, ok := b[id]; ok {
			merged[id] = earliestPosition(left, right)
			continue
		}
		merged[id] = left
	}
	for id, right := range b {
		if _, exists := merged[id]; !exists {
			merged[id] = right
		}
	}
	return merged
}

func mergeFinalizedResources(a, b map[int]resourceFinalization) map[int]resourceFinalization {
	if len(a) == 0 && len(b) == 0 {
		return make(map[int]resourceFinalization)
	}
	merged := make(map[int]resourceFinalization)
	for id, left := range a {
		if right, ok := b[id]; ok {
			if left.state == right.state {
				merged[id] = earliestFinalization(left, right)
				continue
			}
		}
		merged[id] = left
	}
	for id, right := range b {
		if _, exists := merged[id]; !exists {
			merged[id] = right
		}
	}
	return merged
}

func mergeUnknownResources(a, b map[int]bool) map[int]bool {
	if len(a) == 0 && len(b) == 0 {
		return make(map[int]bool)
	}
	merged := make(map[int]bool)
	for id, unknown := range a {
		if unknown {
			merged[id] = true
		}
	}
	for id, unknown := range b {
		if unknown {
			merged[id] = true
		}
	}
	return merged
}

func mergeResourceVars(state *regionState, a, b map[string]int, consumed map[int]frontend.Position, finalized map[int]resourceFinalization, unknown map[int]bool) map[string]int {
	if len(a) == 0 && len(b) == 0 {
		return make(map[string]int)
	}
	merged := make(map[string]int)
	for name, left := range a {
		right, ok := b[name]
		if !ok {
			merged[name] = left
			continue
		}
		if left == right {
			merged[name] = left
			continue
		}
		merged[name] = mergeResourceIDs(state, left, right, consumed, finalized, unknown)
	}
	for name, right := range b {
		if _, exists := merged[name]; !exists {
			merged[name] = right
		}
	}
	return merged
}

func mergeResourceIDs(state *regionState, left int, right int, consumed map[int]frontend.Position, finalized map[int]resourceFinalization, unknown map[int]bool) int {
	if state == nil {
		return left
	}
	merged := state.allocateResourceID()
	leftParam, leftParamOK := state.resourceParamIndex[left]
	rightParam, rightParamOK := state.resourceParamIndex[right]
	leftPath := state.resourceParamPath[left]
	rightPath := state.resourceParamPath[right]
	if unknown[left] || unknown[right] {
		unknown[merged] = true
	} else if leftParamOK && rightParamOK && leftParam == rightParam && leftPath == rightPath {
		state.resourceParamIndex[merged] = leftParam
		state.resourceParamPath[merged] = leftPath
	} else {
		unknown[merged] = true
	}
	leftConsumed, leftConsumedOK := consumed[left]
	rightConsumed, rightConsumedOK := consumed[right]
	switch {
	case leftConsumedOK && rightConsumedOK:
		consumed[merged] = earliestPosition(leftConsumed, rightConsumed)
	case leftConsumedOK:
		consumed[merged] = leftConsumed
	case rightConsumedOK:
		consumed[merged] = rightConsumed
	}
	leftFinal, leftFinalOK := finalized[left]
	rightFinal, rightFinalOK := finalized[right]
	switch {
	case leftFinalOK && rightFinalOK:
		if leftFinal.state == rightFinal.state {
			finalized[merged] = earliestFinalization(leftFinal, rightFinal)
		} else {
			finalized[merged] = leftFinal
		}
	case leftFinalOK:
		finalized[merged] = leftFinal
	case rightFinalOK:
		finalized[merged] = rightFinal
	}
	return merged
}

func earliestPosition(a, b frontend.Position) frontend.Position {
	if a.Line == 0 {
		return b
	}
	if b.Line == 0 {
		return a
	}
	if a.Line < b.Line || (a.Line == b.Line && a.Col <= b.Col) {
		return a
	}
	return b
}

func earliestFinalization(a, b resourceFinalization) resourceFinalization {
	if earliestPosition(a.pos, b.pos) == a.pos {
		return a
	}
	return b
}

type flowSnapshot struct {
	consumedVars       map[string]frontend.Position
	consumedResources  map[int]frontend.Position
	resourceVars       map[string]int
	unknownResources   map[int]bool
	finalizedResources map[int]resourceFinalization
}

func snapshotFlow(state *regionState) flowSnapshot {
	return flowSnapshot{
		consumedVars:       copyConsumedVars(state.consumedVars),
		consumedResources:  copyConsumedResources(state.consumedResources),
		resourceVars:       copyResourceVars(state.resourceVars),
		unknownResources:   copyUnknownResources(state.unknownResources),
		finalizedResources: copyFinalizedResources(state.finalizedResources),
	}
}

func restoreFlow(state *regionState, snap flowSnapshot) {
	state.consumedVars = copyConsumedVars(snap.consumedVars)
	state.consumedResources = copyConsumedResources(snap.consumedResources)
	state.resourceVars = copyResourceVars(snap.resourceVars)
	state.unknownResources = copyUnknownResources(snap.unknownResources)
	state.finalizedResources = copyFinalizedResources(snap.finalizedResources)
}

func mergeFlow(state *regionState, a, b flowSnapshot) {
	consumedResources := mergeConsumedResources(a.consumedResources, b.consumedResources)
	finalizedResources := mergeFinalizedResources(a.finalizedResources, b.finalizedResources)
	unknownResources := mergeUnknownResources(a.unknownResources, b.unknownResources)
	state.consumedVars = mergeConsumedVars(a.consumedVars, b.consumedVars)
	state.consumedResources = consumedResources
	state.unknownResources = unknownResources
	state.finalizedResources = finalizedResources
	state.resourceVars = mergeResourceVars(state, a.resourceVars, b.resourceVars, consumedResources, finalizedResources, unknownResources)
}

type resourceSourceResult struct {
	name      string
	known     bool
	ambiguous bool
	unknown   bool
}

func bindResourceFromExpr(
	name string,
	typeName string,
	expr frontend.Expr,
	funcs map[string]FuncSig,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	if !isResourceHandleType(typeName) {
		state.bindResource(name, "", false)
		return nil
	}
	source, err := resourceSourceForExpr(expr, funcs, module, imports, state)
	if err != nil {
		return err
	}
	if source.ambiguous {
		return fmt.Errorf("%s: resource expression mixes resource provenance", frontend.FormatPos(expr.Pos()))
	}
	if source.unknown {
		state.bindUnknownResource(name)
		return nil
	}
	sourceName := ""
	if source.known {
		sourceName = source.name
		if _, consumed := state.consumedAt(sourceName); consumed {
			state.bindTransferredResource(name, sourceName)
			return nil
		}
	}
	state.bindResource(name, sourceName, true)
	return nil
}

func bindResourceTreeFromExpr(
	name string,
	typeName string,
	expr frontend.Expr,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	if state == nil || name == "" {
		return nil
	}
	state.clearResourceTree(name)
	if !typeContainsResourceHandle(typeName, types) {
		state.bindResource(name, "", false)
		return nil
	}
	if isResourceHandleType(typeName) {
		return bindResourceFromExpr(name, typeName, expr, funcs, module, imports, state)
	}
	if sourcePrefix, ok := resourcePathForExpr(expr); ok {
		copyResourceTreeFromPath(name, sourcePrefix, typeName, types, state)
		return nil
	}
	switch e := expr.(type) {
	case *frontend.MatchExpr:
		if e.ResultLocal != "" {
			copyResourceTreeFromPath(name, e.ResultLocal, typeName, types, state)
			return nil
		}
	case *frontend.StructLitExpr:
		info, ok := types[typeName]
		if !ok || info.Kind != TypeStruct {
			markResourceTreeUnknown(name, typeName, types, state)
			return nil
		}
		byName := make(map[string]frontend.Expr, len(e.Fields))
		for _, field := range e.Fields {
			byName[field.Name] = field.Value
		}
		for _, field := range info.Fields {
			value := byName[field.Name]
			if value == nil {
				continue
			}
			if err := bindResourceTreeFromExpr(resourceFieldPath(name, field.Name), field.TypeName, value, funcs, types, module, imports, state); err != nil {
				return err
			}
		}
		return nil
	case *frontend.CallExpr:
		if info, ok := types[typeName]; ok && info.Kind == TypeStruct && e.Name == typeName {
			for i, field := range info.Fields {
				if i >= len(e.Args) {
					break
				}
				if err := bindResourceTreeFromExpr(resourceFieldPath(name, field.Name), field.TypeName, e.Args[i], funcs, types, module, imports, state); err != nil {
					return err
				}
			}
			return nil
		}
		resolved, err := resolveCheckedCallName(e.Name, funcs, module, imports, e.At)
		if err != nil {
			return err
		}
		if resolved == "core.recv_typed" {
			bindFreshResourceTree(name, typeName, types, state)
			return nil
		}
		enumType, caseInfo, ok, err := resolveEnumCaseConstructorCall(e, types, module, imports)
		if err != nil {
			return err
		}
		if ok && enumType == typeName {
			for i, arg := range e.Args {
				if i >= len(caseInfo.PayloadTypes) {
					break
				}
				if err := bindResourceTreeFromExpr(resourceEnumPayloadPath(name, caseInfo.Ordinal, i), caseInfo.PayloadTypes[i], arg, funcs, types, module, imports, state); err != nil {
					return err
				}
			}
			return nil
		}
	}
	markResourceTreeUnknown(name, typeName, types, state)
	return nil
}

func bindResourceTreeFromPathOrUnknown(dst string, src string, typeName string, types map[string]*TypeInfo, state *regionState) {
	if !typeContainsResourceHandle(typeName, types) {
		state.bindResource(dst, "", false)
		return
	}
	copyResourceTreeFromPath(dst, src, typeName, types, state)
}

func bindPatternResourceLocals(
	pattern frontend.Expr,
	fallbackName string,
	scrutineePath string,
	scrutType string,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
) error {
	if state == nil || scrutineePath == "" || !typeContainsResourceHandle(scrutType, types) {
		return nil
	}
	info, ok := types[scrutType]
	if !ok {
		return nil
	}
	if pattern == nil {
		if fallbackName == "" || info.Kind != TypeOptional {
			return nil
		}
		bindResourceTreeFromPathOrUnknown(fallbackName, resourceFieldPath(scrutineePath, "$elem"), info.ElemType, types, state)
		return nil
	}
	switch p := pattern.(type) {
	case *frontend.SomePatternExpr:
		if info.Kind != TypeOptional {
			return nil
		}
		bindResourceTreeFromPathOrUnknown(p.Name, resourceFieldPath(scrutineePath, "$elem"), info.ElemType, types, state)
	case *frontend.EnumCasePatternExpr:
		caseType, caseInfo, found, err := resolveEnumCasePattern(p, types, module, imports)
		if err != nil {
			return err
		}
		if !found || caseType != scrutType {
			return nil
		}
		for i, binding := range p.Bindings {
			if i >= len(caseInfo.PayloadTypes) {
				break
			}
			bindResourceTreeFromPathOrUnknown(binding, resourceEnumPayloadPath(scrutineePath, caseInfo.Ordinal, i), caseInfo.PayloadTypes[i], types, state)
		}
	}
	return nil
}

func copyResourceTreeFromPath(dst string, src string, typeName string, types map[string]*TypeInfo, state *regionState) {
	for _, leaf := range resourceLeafPaths(typeName, types, "") {
		dstLeaf := joinResourcePath(dst, leaf)
		srcLeaf := joinResourcePath(src, leaf)
		if id, ok := state.resourceID(srcLeaf); ok && !state.resourceUnknown(srcLeaf) {
			if _, consumed := state.consumedAt(srcLeaf); consumed {
				state.bindTransferredResource(dstLeaf, srcLeaf)
				continue
			}
			state.resourceVars[dstLeaf] = id
			continue
		}
		state.bindUnknownResource(dstLeaf)
	}
}

func markResourceTreeUnknown(prefix string, typeName string, types map[string]*TypeInfo, state *regionState) {
	for _, leaf := range resourceLeafPaths(typeName, types, "") {
		state.bindUnknownResource(joinResourcePath(prefix, leaf))
	}
}

func bindFreshResourceTree(prefix string, typeName string, types map[string]*TypeInfo, state *regionState) {
	for _, leaf := range resourceLeafPaths(typeName, types, "") {
		state.bindResource(joinResourcePath(prefix, leaf), "", true)
	}
}

func resourceLeafPaths(typeName string, types map[string]*TypeInfo, prefix string) []string {
	return resourceLeafPathsVisiting(typeName, types, prefix, map[string]bool{})
}

func resourceLeafPathsVisiting(typeName string, types map[string]*TypeInfo, prefix string, visiting map[string]bool) []string {
	if isResourceHandleType(typeName) {
		return []string{prefix}
	}
	info, ok := types[typeName]
	if !ok {
		return nil
	}
	var out []string
	switch info.Kind {
	case TypeStruct:
		if visiting[typeName] {
			return nil
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, field := range info.Fields {
			out = append(out, resourceLeafPathsVisiting(field.TypeName, types, resourceFieldPath(prefix, field.Name), visiting)...)
		}
	case TypeEnum:
		if visiting[typeName] {
			return nil
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, c := range info.EnumCases {
			for i, payload := range c.PayloadTypes {
				out = append(out, resourceLeafPathsVisiting(payload, types, resourceEnumPayloadPath(prefix, c.Ordinal, i), visiting)...)
			}
		}
	case TypeArray, TypeOptional:
		out = append(out, resourceLeafPathsVisiting(info.ElemType, types, resourceFieldPath(prefix, "$elem"), visiting)...)
	}
	return out
}

func resourcePathForExpr(expr frontend.Expr) (string, bool) {
	base, fields, _, ok := splitFieldPath(expr)
	if !ok {
		return "", false
	}
	path := base
	for _, field := range fields {
		path = resourceFieldPath(path, field)
	}
	return path, true
}

func resourceFieldPath(prefix string, field string) string {
	if prefix == "" {
		return field
	}
	return prefix + "." + field
}

func resourceEnumPayloadPath(prefix string, ordinal int32, index int) string {
	return resourceFieldPath(prefix, fmt.Sprintf("$case%d.payload%d", ordinal, index))
}

func joinResourcePath(prefix string, leaf string) string {
	if leaf == "" {
		return prefix
	}
	if prefix == "" {
		return leaf
	}
	return prefix + "." + leaf
}

func resourceSourceForPath(path string, state *regionState) resourceSourceResult {
	if state == nil || path == "" {
		return resourceSourceResult{}
	}
	if _, ok := state.resourceID(path); !ok {
		return resourceSourceResult{}
	}
	if state.resourceUnknown(path) {
		return resourceSourceResult{unknown: true}
	}
	return resourceSourceResult{name: path, known: true}
}

func resourceSourceForExpr(
	expr frontend.Expr,
	funcs map[string]FuncSig,
	module string,
	imports map[string]string,
	state *regionState,
) (resourceSourceResult, error) {
	if expr == nil || state == nil {
		return resourceSourceResult{}, nil
	}
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return resourceSourceForPath(e.Name, state), nil
	case *frontend.FieldAccessExpr:
		path, ok := resourcePathForExpr(e)
		if !ok {
			return resourceSourceResult{}, nil
		}
		return resourceSourceForPath(path, state), nil
	case *frontend.TryExpr:
		return resourceSourceForExpr(e.X, funcs, module, imports, state)
	case *frontend.AwaitExpr:
		return resourceSourceForExpr(e.X, funcs, module, imports, state)
	case *frontend.CallExpr:
		resolved, err := resolveCheckedCallName(e.Name, funcs, module, imports, e.At)
		if err != nil {
			return resourceSourceResult{}, err
		}
		sig, ok := funcs[resolved]
		if !ok || sig.ReturnResourceParam < 0 {
			if ok && sig.ReturnResourceParam == regionUnknown {
				return resourceSourceResult{unknown: true}, nil
			}
			return resourceSourceResult{}, nil
		}
		if sig.ReturnResourceParam >= len(e.Args) {
			return resourceSourceResult{}, fmt.Errorf("%s: invalid resource signature for '%s'", frontend.FormatPos(e.At), resolved)
		}
		if sig.ReturnResourcePath != "" {
			argPath, ok := resourcePathForExpr(e.Args[sig.ReturnResourceParam])
			if !ok {
				return resourceSourceResult{unknown: true}, nil
			}
			source := resourceSourceForPath(joinResourcePath(argPath, sig.ReturnResourcePath), state)
			if !source.known && !source.unknown {
				return resourceSourceResult{unknown: true}, nil
			}
			return source, nil
		}
		return resourceSourceForExpr(e.Args[sig.ReturnResourceParam], funcs, module, imports, state)
	case *frontend.MatchExpr:
		if e.ResultLocal != "" {
			source := resourceSourceForPath(e.ResultLocal, state)
			if source.known || source.unknown {
				return source, nil
			}
		}
		var merged resourceSourceResult
		set := false
		for _, c := range e.Cases {
			source, err := resourceSourceForExpr(c.Value, funcs, module, imports, state)
			if err != nil {
				return resourceSourceResult{}, err
			}
			if !set {
				merged = source
				set = true
				continue
			}
			merged = mergeResourceSourceResults(merged, source)
		}
		return merged, nil
	case *frontend.CatchExpr:
		merged, err := resourceSourceForExpr(e.Call, funcs, module, imports, state)
		if err != nil {
			return resourceSourceResult{}, err
		}
		set := true
		for _, c := range e.Cases {
			source, err := resourceSourceForExpr(c.Value, funcs, module, imports, state)
			if err != nil {
				return resourceSourceResult{}, err
			}
			if !set {
				merged = source
				set = true
				continue
			}
			merged = mergeResourceSourceResults(merged, source)
		}
		return merged, nil
	default:
		return resourceSourceResult{}, nil
	}
}

func mergeResourceSourceResults(a, b resourceSourceResult) resourceSourceResult {
	if a.ambiguous || b.ambiguous {
		return resourceSourceResult{ambiguous: true}
	}
	if a.unknown || b.unknown {
		return resourceSourceResult{unknown: true}
	}
	if !a.known && !b.known {
		return resourceSourceResult{}
	}
	if a.known && b.known && a.name == b.name {
		return a
	}
	return resourceSourceResult{ambiguous: true}
}

func resolveCheckedCallName(name string, funcs map[string]FuncSig, module string, imports map[string]string, pos frontend.Position) (string, error) {
	if builtin, ok := ResolveBuiltinAlias(name); ok {
		return builtin, nil
	}
	if _, ok := funcs[name]; ok {
		return name, nil
	}
	return resolveCallName(name, module, imports, pos)
}

func collectDeferCaptures(stmts []frontend.Stmt, locals map[string]LocalInfo) map[string]frontend.Position {
	captures := make(map[string]frontend.Position)
	collectStmtCaptures(stmts, locals, map[string]bool{}, captures)
	return captures
}

func collectClosureCaptures(fn *frontend.FuncDecl, locals map[string]LocalInfo) map[string]frontend.Position {
	captures := make(map[string]frontend.Position)
	bound := make(map[string]bool, len(fn.Params))
	for _, param := range fn.Params {
		bound[param.Name] = true
	}
	collectStmtCaptures(fn.Body, locals, bound, captures)
	return captures
}

func validateFunctionTypeLiteralBinding(
	name string,
	declared frontend.TypeRef,
	closure *frontend.ClosureExpr,
	locals map[string]LocalInfo,
	module string,
	imports map[string]string,
) error {
	if declared.Kind != frontend.TypeRefFunction {
		return nil
	}
	if closure == nil || closure.Decl == nil {
		return fmt.Errorf("%s: function-typed local '%s' must be initialized with a non-capturing closure literal in this MVP", frontend.FormatPos(declared.At), name)
	}
	if len(closure.Decl.TypeParams) > 0 {
		return fmt.Errorf("%s: generic closure literals are not supported for function-typed local '%s' in this MVP", frontend.FormatPos(closure.At), name)
	}
	if closure.Decl.HasThrows {
		return fmt.Errorf("%s: throwing closure literals are not supported for function-typed local '%s' in this MVP", frontend.FormatPos(closure.At), name)
	}
	declaredEffects, err := functionTypeRefEffects(declared, declared.At)
	if err != nil {
		return err
	}
	closureEffects, err := normalizeEffects(closure.Decl.Uses, closure.Decl.Pos)
	if err != nil {
		return err
	}
	if err := validateFunctionTypeCallableEffects(declaredEffects, closureEffects, closure.At, "function-typed local", name); err != nil {
		return err
	}
	if captured, pos, ok := firstCapture(collectClosureCaptures(closure.Decl, locals)); ok {
		return unsupportedFunctionTypedCaptureError(pos, name, captured)
	}
	if len(declared.Params) != len(closure.Decl.Params) {
		return fmt.Errorf("%s: function-typed local '%s' parameter count mismatch: expected %d, got %d", frontend.FormatPos(closure.At), name, len(declared.Params), len(closure.Decl.Params))
	}
	for i := range declared.Params {
		want, err := resolveTypeName(&declared.Params[i], module, imports)
		if err != nil {
			return err
		}
		got, err := resolveTypeName(&closure.Decl.Params[i].Type, module, imports)
		if err != nil {
			return err
		}
		if want != got {
			return fmt.Errorf("%s: function-typed local '%s' parameter %d type mismatch: expected '%s', got '%s'", frontend.FormatPos(closure.Decl.Params[i].At), name, i+1, want, got)
		}
	}
	if declared.Return == nil {
		return fmt.Errorf("%s: missing function return type", frontend.FormatPos(declared.At))
	}
	wantRet, err := resolveTypeName(declared.Return, module, imports)
	if err != nil {
		return err
	}
	gotRet, err := resolveTypeName(&closure.Decl.ReturnType, module, imports)
	if err != nil {
		return err
	}
	if wantRet != gotRet {
		return fmt.Errorf("%s: function-typed local '%s' return type mismatch: expected '%s', got '%s'", frontend.FormatPos(closure.At), name, wantRet, gotRet)
	}
	return nil
}

func unsupportedFunctionTypedCaptureError(pos frontend.Position, localName, captured string) error {
	return fmt.Errorf(
		"%s: function-typed local '%s' captures '%s'; captures are not supported for function-typed values in this MVP (use a let-bound ptr closure and call it directly, or pass a non-capturing named function/closure symbol)",
		frontend.FormatPos(pos),
		localName,
		captured,
	)
}

func validateFunctionTypeNamedSymbolBinding(
	name string,
	declared frontend.TypeRef,
	init *frontend.IdentExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	module string,
	imports map[string]string,
) (string, error) {
	if declared.Kind != frontend.TypeRefFunction {
		return "", nil
	}
	if init == nil {
		return "", fmt.Errorf("%s: function-typed local '%s' must be initialized with a named function/closure symbol in this MVP", frontend.FormatPos(declared.At), name)
	}
	if localInfo, ok := locals[init.Name]; ok {
		if !localInfo.FunctionTypeValue || localInfo.FunctionValue == "" || localInfo.Mutable {
			return "", fmt.Errorf("%s: function-typed local '%s' must be initialized with a named function/closure symbol in this MVP", frontend.FormatPos(init.At), name)
		}
		sig, ok := funcs[localInfo.FunctionValue]
		if !ok {
			return "", fmt.Errorf("%s: unknown function symbol '%s'", frontend.FormatPos(init.At), localInfo.FunctionValue)
		}
		if localInfo.GenericFunctionValue || sig.Generic {
			return "", fmt.Errorf("%s: generic function symbol '%s' is not supported for function-typed local '%s' in this MVP", frontend.FormatPos(init.At), init.Name, name)
		}
		if sig.ThrowsType != "" {
			return "", fmt.Errorf("%s: throwing function symbol '%s' is not supported for function-typed local '%s' in this MVP", frontend.FormatPos(init.At), init.Name, name)
		}
		if err := validateFunctionTypeSymbolSignature(name, declared, sig, module, imports, init.At); err != nil {
			return "", err
		}
		return localInfo.FunctionValue, nil
	}
	if _, ok := globals[init.Name]; ok {
		return "", fmt.Errorf("%s: function-typed local '%s' must be initialized with a named function/closure symbol in this MVP", frontend.FormatPos(init.At), name)
	}
	resolved, err := resolveCheckedCallName(init.Name, funcs, module, imports, init.At)
	if err != nil {
		return "", fmt.Errorf("%s: function-typed local '%s' must be initialized with a named function/closure symbol in this MVP", frontend.FormatPos(init.At), name)
	}
	sig, ok := funcs[resolved]
	if !ok {
		return "", fmt.Errorf("%s: function-typed local '%s' must be initialized with a named function/closure symbol in this MVP", frontend.FormatPos(init.At), name)
	}
	if err := ensureFuncVisible(resolved, sig, module, init.At); err != nil {
		return "", err
	}
	if sig.Generic {
		return "", fmt.Errorf("%s: generic function symbol '%s' is not supported for function-typed local '%s' in this MVP", frontend.FormatPos(init.At), init.Name, name)
	}
	if sig.ThrowsType != "" {
		return "", fmt.Errorf("%s: throwing function symbol '%s' is not supported for function-typed local '%s' in this MVP", frontend.FormatPos(init.At), init.Name, name)
	}
	if err := validateFunctionTypeSymbolSignature(name, declared, sig, module, imports, init.At); err != nil {
		return "", err
	}
	return resolved, nil
}

func validateFunctionTypeCallableEffects(declaredEffects []string, targetEffects []string, pos frontend.Position, context, rawName string) error {
	missing := missingRequiredEffects(targetEffects, declaredEffects)
	if len(missing) > 0 {
		return fmt.Errorf("%s: %s '%s' requires effects %s but function type does not declare them", frontend.FormatPos(pos), context, rawName, strings.Join(missing, ", "))
	}
	return nil
}

func validateFunctionTypeSymbolSignature(
	localName string,
	declared frontend.TypeRef,
	sig FuncSig,
	module string,
	imports map[string]string,
	pos frontend.Position,
) error {
	if len(declared.Params) != len(sig.ParamTypes) {
		return fmt.Errorf("%s: function-typed local '%s' parameter count mismatch: expected %d, got %d", frontend.FormatPos(pos), localName, len(declared.Params), len(sig.ParamTypes))
	}
	for i := range declared.Params {
		want, err := resolveTypeName(&declared.Params[i], module, imports)
		if err != nil {
			return err
		}
		got := sig.ParamTypes[i]
		if want != got {
			return fmt.Errorf("%s: function-typed local '%s' parameter %d type mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), localName, i+1, want, got)
		}
	}
	if declared.Return == nil {
		return fmt.Errorf("%s: missing function return type", frontend.FormatPos(declared.At))
	}
	wantRet, err := resolveTypeName(declared.Return, module, imports)
	if err != nil {
		return err
	}
	if wantRet != sig.ReturnType {
		return fmt.Errorf("%s: function-typed local '%s' return type mismatch: expected '%s', got '%s'", frontend.FormatPos(pos), localName, wantRet, sig.ReturnType)
	}
	declaredEffects, err := functionTypeRefEffects(declared, declared.At)
	if err != nil {
		return err
	}
	if err := validateFunctionTypeCallableEffects(declaredEffects, sig.Effects, pos, "function-typed local", localName); err != nil {
		return err
	}
	return nil
}

func validateReturnedFunctionSignature(
	callerSig FuncSig,
	returnedSig FuncSig,
	pos frontend.Position,
	rawName string,
) error {
	if !callerSig.ReturnFunctionType {
		return nil
	}
	if len(callerSig.ReturnFunctionParams) != len(returnedSig.ParamTypes) {
		return fmt.Errorf("%s: returned function symbol '%s' has incompatible parameter count: expected %d, got %d", frontend.FormatPos(pos), rawName, len(callerSig.ReturnFunctionParams), len(returnedSig.ParamTypes))
	}
	for i := range callerSig.ReturnFunctionParams {
		if callerSig.ReturnFunctionParams[i] != returnedSig.ParamTypes[i] {
			return fmt.Errorf(
				"%s: returned function symbol '%s' parameter %d type mismatch: expected '%s', got '%s'",
				frontend.FormatPos(pos),
				rawName,
				i+1,
				callerSig.ReturnFunctionParams[i],
				returnedSig.ParamTypes[i],
			)
		}
	}
	if callerSig.ReturnFunctionReturn != returnedSig.ReturnType {
		return fmt.Errorf(
			"%s: returned function symbol '%s' return type mismatch: expected '%s', got '%s'",
			frontend.FormatPos(pos),
			rawName,
			callerSig.ReturnFunctionReturn,
			returnedSig.ReturnType,
		)
	}
	if err := validateFunctionTypeCallableEffects(callerSig.ReturnFunctionEffects, returnedSig.Effects, pos, "returned function symbol", rawName); err != nil {
		return err
	}
	return nil
}

func functionTypeRefSignature(ref frontend.TypeRef, module string, imports map[string]string) ([]string, string, error) {
	if ref.Kind != frontend.TypeRefFunction {
		return nil, "", nil
	}
	paramTypes := make([]string, 0, len(ref.Params))
	for i := range ref.Params {
		tname, err := resolveTypeName(&ref.Params[i], module, imports)
		if err != nil {
			return nil, "", err
		}
		paramTypes = append(paramTypes, tname)
	}
	if ref.Return == nil {
		return nil, "", fmt.Errorf("%s: missing function return type", frontend.FormatPos(ref.At))
	}
	retType, err := resolveTypeName(ref.Return, module, imports)
	if err != nil {
		return nil, "", err
	}
	return paramTypes, retType, nil
}

func functionTypeRefEffects(ref frontend.TypeRef, pos frontend.Position) ([]string, error) {
	if ref.Kind != frontend.TypeRefFunction {
		return nil, nil
	}
	return normalizeEffects(ref.Uses, pos)
}

func functionTypeRefSignatureAndEffects(ref frontend.TypeRef, module string, imports map[string]string) ([]string, string, []string, error) {
	params, ret, err := functionTypeRefSignature(ref, module, imports)
	if err != nil {
		return nil, "", nil, err
	}
	effects, err := functionTypeRefEffects(ref, ref.At)
	if err != nil {
		return nil, "", nil, err
	}
	return params, ret, effects, nil
}

func configureClosureCaptures(
	closure *frontend.ClosureExpr,
	locals map[string]LocalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
) error {
	if closure == nil || closure.Decl == nil || len(closure.Captures) > 0 {
		return nil
	}
	capturePositions := collectClosureCaptures(closure.Decl, locals)
	if len(capturePositions) == 0 {
		return nil
	}
	fullName := qualifyName(module, closure.Name)
	sig, ok := funcs[fullName]
	if !ok {
		return fmt.Errorf("%s: internal error: closure function '%s' is missing from signature table", frontend.FormatPos(closure.At), fullName)
	}
	captures := make([]frontend.ClosureCapture, 0, len(capturePositions))
	captureParamSlots := 0
	for len(capturePositions) > 0 {
		name, pos, _ := firstCapture(capturePositions)
		delete(capturePositions, name)
		info, ok := locals[name]
		if !ok {
			return fmt.Errorf("%s: internal error: closure capture '%s' is missing from locals", frontend.FormatPos(pos), name)
		}
		if info.Mutable {
			return fmt.Errorf("%s: closure capture '%s' is mutable; %s", frontend.FormatPos(pos), name, closureCaptureSupportedSubsetText())
		}
		if !isClosureCaptureType(info.TypeName, types) {
			return fmt.Errorf("%s: closure capture '%s' has unsupported type '%s'; %s", frontend.FormatPos(pos), name, info.TypeName, closureCaptureSupportedSubsetText())
		}
		typeRef := frontend.TypeRef{At: pos, Kind: frontend.TypeRefNamed, Name: info.TypeName}
		captures = append(captures, frontend.ClosureCapture{At: pos, Name: name, Type: typeRef})
		closure.Decl.Params = append(closure.Decl.Params, frontend.ParamDecl{At: pos, Name: name, Type: typeRef})
		sig.ParamNames = append(sig.ParamNames, name)
		sig.ParamTypes = append(sig.ParamTypes, info.TypeName)
		sig.ParamOwnership = append(sig.ParamOwnership, "")
		captureParamSlots += info.SlotCount
	}
	sig.ParamSlots += captureParamSlots
	funcs[fullName] = sig
	closure.Captures = captures
	return nil
}

func closureCaptureSupportedSubsetText() string {
	return "only immutable local Int/Bool/String and simple struct captures without ptr/resource fields are supported in this MVP"
}

func closureLiteralDirectCallCaptureText() string {
	return "only local direct calls can capture immutable Int/Bool/String values and simple structs without ptr/resource fields in this MVP"
}

func isClosureCaptureType(typeName string, types map[string]*TypeInfo) bool {
	return isClosureCaptureTypeVisiting(typeName, types, map[string]bool{})
}

func isClosureCaptureTypeVisiting(typeName string, types map[string]*TypeInfo, visiting map[string]bool) bool {
	info, ok := types[typeName]
	if !ok {
		return false
	}
	switch info.Kind {
	case TypeI32, TypeBool:
		return info.SlotCount == 1
	case TypeStr:
		return true
	case TypeStruct:
		if visiting[typeName] {
			return false
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, field := range info.Fields {
			if field.TypeName == "ptr" || typeContainsResourceHandle(field.TypeName, types) {
				return false
			}
			if !isClosureCaptureTypeVisiting(field.TypeName, types, visiting) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func appendClosureCaptureArgs(call *frontend.CallExpr, local LocalInfo) error {
	if len(local.FunctionCaptures) == 0 {
		return nil
	}
	if len(call.ArgLabels) > 0 {
		return fmt.Errorf("%s: capturing closure '%s' calls do not support argument labels in this MVP", frontend.FormatPos(call.At), call.Name)
	}
	for _, capture := range local.FunctionCaptures {
		call.Args = append(call.Args, &frontend.IdentExpr{At: capture.At, Name: capture.Name})
	}
	return nil
}

func firstCapture(captures map[string]frontend.Position) (string, frontend.Position, bool) {
	firstName := ""
	firstPos := frontend.Position{}
	for name, pos := range captures {
		if firstName == "" ||
			pos.Line < firstPos.Line ||
			(pos.Line == firstPos.Line && pos.Col < firstPos.Col) ||
			(pos.Line == firstPos.Line && pos.Col == firstPos.Col && name < firstName) {
			firstName = name
			firstPos = pos
		}
	}
	return firstName, firstPos, firstName != ""
}

func collectStmtCaptures(stmts []frontend.Stmt, locals map[string]LocalInfo, bound map[string]bool, captures map[string]frontend.Position) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.PrintStmt:
			collectExprCaptures(s.Value, locals, bound, captures)
		case *frontend.ExpectStmt:
			collectExprCaptures(s.Cond, locals, bound, captures)
		case *frontend.ReturnStmt:
			collectExprCaptures(s.Value, locals, bound, captures)
		case *frontend.ThrowStmt:
			collectExprCaptures(s.Value, locals, bound, captures)
		case *frontend.FreeStmt:
			collectExprCaptures(s.Value, locals, bound, captures)
		case *frontend.LetStmt:
			collectExprCaptures(s.Value, locals, bound, captures)
			bound[s.Name] = true
		case *frontend.AssignStmt:
			collectExprCaptures(s.Target, locals, bound, captures)
			collectExprCaptures(s.Value, locals, bound, captures)
		case *frontend.IfStmt:
			collectExprCaptures(s.Cond, locals, bound, captures)
			collectStmtCaptures(s.Then, locals, cloneBoolMap(bound), captures)
			collectStmtCaptures(s.Else, locals, cloneBoolMap(bound), captures)
		case *frontend.IfLetStmt:
			collectExprCaptures(s.Value, locals, bound, captures)
			thenBound := cloneBoolMap(bound)
			addPatternCaptureBindings(s.Pattern, s.Name, thenBound)
			collectStmtCaptures(s.Then, locals, thenBound, captures)
			collectStmtCaptures(s.Else, locals, cloneBoolMap(bound), captures)
		case *frontend.WhileStmt:
			collectExprCaptures(s.Cond, locals, bound, captures)
			collectStmtCaptures(s.Body, locals, cloneBoolMap(bound), captures)
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				collectExprCaptures(s.Iterable, locals, bound, captures)
			} else {
				collectExprCaptures(s.Start, locals, bound, captures)
				collectExprCaptures(s.End, locals, bound, captures)
			}
			bodyBound := cloneBoolMap(bound)
			bodyBound[s.Name] = true
			collectStmtCaptures(s.Body, locals, bodyBound, captures)
		case *frontend.MatchStmt:
			collectExprCaptures(s.Value, locals, bound, captures)
			for _, c := range s.Cases {
				caseBound := cloneBoolMap(bound)
				addPatternCaptureBindings(c.Pattern, "", caseBound)
				if c.Guard != nil {
					collectExprCaptures(c.Guard, locals, caseBound, captures)
				}
				collectStmtCaptures(c.Body, locals, caseBound, captures)
			}
		case *frontend.IslandStmt:
			collectExprCaptures(s.Size, locals, bound, captures)
			bodyBound := cloneBoolMap(bound)
			bodyBound[s.Name] = true
			collectStmtCaptures(s.Body, locals, bodyBound, captures)
		case *frontend.UnsafeStmt:
			collectStmtCaptures(s.Body, locals, cloneBoolMap(bound), captures)
		case *frontend.DeferStmt:
			collectStmtCaptures(s.Body, locals, cloneBoolMap(bound), captures)
		case *frontend.ExprStmt:
			collectExprCaptures(s.Expr, locals, bound, captures)
		}
	}
}

func collectExprCaptures(expr frontend.Expr, locals map[string]LocalInfo, bound map[string]bool, captures map[string]frontend.Position) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		if _, ok := locals[e.Name]; ok && !bound[e.Name] {
			if _, exists := captures[e.Name]; !exists {
				captures[e.Name] = e.At
			}
		}
	case *frontend.FieldAccessExpr:
		collectExprCaptures(e.Base, locals, bound, captures)
	case *frontend.IndexExpr:
		collectExprCaptures(e.Base, locals, bound, captures)
		collectExprCaptures(e.Index, locals, bound, captures)
	case *frontend.BinaryExpr:
		collectExprCaptures(e.Left, locals, bound, captures)
		collectExprCaptures(e.Right, locals, bound, captures)
	case *frontend.UnaryExpr:
		collectExprCaptures(e.X, locals, bound, captures)
	case *frontend.TryExpr:
		collectExprCaptures(e.X, locals, bound, captures)
	case *frontend.AwaitExpr:
		collectExprCaptures(e.X, locals, bound, captures)
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			collectExprCaptures(arg, locals, bound, captures)
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			collectExprCaptures(field.Value, locals, bound, captures)
		}
	case *frontend.MatchExpr:
		collectExprCaptures(e.Value, locals, bound, captures)
		for _, c := range e.Cases {
			caseBound := cloneBoolMap(bound)
			addPatternCaptureBindings(c.Pattern, "", caseBound)
			if c.Guard != nil {
				collectExprCaptures(c.Guard, locals, caseBound, captures)
			}
			collectExprCaptures(c.Value, locals, caseBound, captures)
		}
	case *frontend.CatchExpr:
		collectExprCaptures(e.Call, locals, bound, captures)
		for _, c := range e.Cases {
			caseBound := cloneBoolMap(bound)
			addPatternCaptureBindings(c.Pattern, "", caseBound)
			if c.Guard != nil {
				collectExprCaptures(c.Guard, locals, caseBound, captures)
			}
			collectExprCaptures(c.Value, locals, caseBound, captures)
		}
	}
}

func addPatternCaptureBindings(pattern frontend.Expr, name string, bound map[string]bool) {
	if name != "" {
		bound[name] = true
	}
	switch p := pattern.(type) {
	case *frontend.IdentExpr:
		bound[p.Name] = true
	case *frontend.SomePatternExpr:
		bound[p.Name] = true
	case *frontend.EnumCasePatternExpr:
		for _, binding := range p.Bindings {
			bound[binding] = true
		}
	}
}

func cloneBoolMap(src map[string]bool) map[string]bool {
	if len(src) == 0 {
		return make(map[string]bool)
	}
	dst := make(map[string]bool, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func checkStmts(
	stmts []frontend.Stmt,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	returnType string,
	borrowedParams map[string]struct{},
	inoutParams map[string]struct{},
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
) error {
	if state != nil {
		state.pushDeferCaptureFrame()
		defer state.popDeferCaptureFrame()
	}
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.PrintStmt:
			if err := effects.require(s.At, "io"); err != nil {
				return err
			}
			tname, _, err := checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return err
			}
			if !isPrintableType(tname, types) {
				return fmt.Errorf("%s: print expects str or []u8", frontend.FormatPos(s.At))
			}
		case *frontend.BreakStmt:
			if state.loopDepth == 0 {
				return fmt.Errorf("%s: break outside loop", frontend.FormatPos(s.At))
			}
		case *frontend.ContinueStmt:
			if state.loopDepth == 0 {
				return fmt.Errorf("%s: continue outside loop", frontend.FormatPos(s.At))
			}
		case *frontend.FreeStmt:
			if err := effects.requireAll(s.At, []string{"islands", "mem"}); err != nil {
				return err
			}
			tname, _, err := checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return err
			}
			if tname != "island" {
				return fmt.Errorf("%s: free expects island, got '%s'", frontend.FormatPos(s.At), tname)
			}
			if !s.Implicit && !state.inUnsafe() {
				return fmt.Errorf("%s: free is only allowed in unsafe blocks", frontend.FormatPos(s.At))
			}
			source, err := resourceSourceForExpr(s.Value, funcs, module, imports, state)
			if err != nil {
				return err
			}
			if source.ambiguous {
				return fmt.Errorf("%s: resource expression mixes resource provenance", frontend.FormatPos(s.Value.Pos()))
			}
			if source.unknown {
				name, _ := resourcePathForExpr(s.Value)
				if name == "" {
					name = "<resource>"
				}
				return fmt.Errorf("%s: ambiguous resource provenance for '%s' after control-flow merge", frontend.FormatPos(s.Value.Pos()), name)
			}
			if source.known {
				state.markResourceFinalized(source.name, "freed", s.Value.Pos())
			}
		case *frontend.ReturnStmt:
			tname := ""
			regionID := regionNone
			handledFunctionReturn := false
			callerSig, callerSigOK := currentCallerSignature(effects, funcs)
			if id, ok := s.Value.(*frontend.IdentExpr); ok {
				if localInfo, exists := locals[id.Name]; exists && localInfo.FunctionTypeValue {
					if !callerSigOK || !callerSig.ReturnFunctionType {
						return fmt.Errorf("%s: function value '%s' cannot escape as a first-class value in this MVP; only direct local calls are supported", frontend.FormatPos(s.At), id.Name)
					}
					if localInfo.FunctionValue == "" {
						return fmt.Errorf("%s: returning function-typed value '%s' requires a symbol-backed non-capturing function value in this MVP", frontend.FormatPos(s.At), id.Name)
					}
					if localInfo.GenericFunctionValue {
						return fmt.Errorf("%s: generic function symbol '%s' is not supported for function return in this MVP", frontend.FormatPos(s.At), id.Name)
					}
					targetSig, ok := funcs[localInfo.FunctionValue]
					if !ok {
						return fmt.Errorf("%s: unknown function symbol '%s'", frontend.FormatPos(s.At), localInfo.FunctionValue)
					}
					if targetSig.Generic {
						return fmt.Errorf("%s: generic function symbol '%s' is not supported for function return in this MVP", frontend.FormatPos(s.At), id.Name)
					}
					if targetSig.ThrowsType != "" {
						return fmt.Errorf("%s: throwing function symbol '%s' is not supported for function return in this MVP", frontend.FormatPos(s.At), id.Name)
					}
					if len(localInfo.FunctionCaptures) > 0 {
						return fmt.Errorf("%s: returning function-typed value '%s' captures local values; captured function values cannot be returned in this MVP", frontend.FormatPos(s.At), id.Name)
					}
					if err := validateReturnedFunctionSignature(callerSig, targetSig, s.At, id.Name); err != nil {
						return err
					}
					if analysis != nil {
						if analysis.returnFunctionSymbol != "" && analysis.returnFunctionSymbol != localInfo.FunctionValue {
							return fmt.Errorf("%s: function return symbol mismatch across return paths: '%s' vs '%s'", frontend.FormatPos(s.At), analysis.returnFunctionSymbol, localInfo.FunctionValue)
						}
						analysis.returnFunctionSymbol = localInfo.FunctionValue
					}
					tname = localInfo.TypeName
					regionID = regionNone
					handledFunctionReturn = true
				}
			}
			if !handledFunctionReturn {
				var err error
				tname, regionID, err = checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
				if err != nil {
					return err
				}
			}
			if callerSigOK && callerSig.ReturnFunctionType && !handledFunctionReturn {
				return fmt.Errorf("%s: returning function-typed value requires a symbol-backed non-generic non-throwing function value in this MVP", frontend.FormatPos(s.At))
			}
			if typeMayContainRegion(tname, types) {
				if err := checkBorrowedEscape(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
					return fmt.Errorf("%s: borrowed local '%s' cannot escape via return", frontend.FormatPos(s.At), borrowedName)
				}); err != nil {
					return err
				}
				if id, ok := s.Value.(*frontend.IdentExpr); ok {
					if _, borrowed := borrowedParams[id.Name]; borrowed {
						return fmt.Errorf("%s: borrowed local '%s' cannot escape via return", frontend.FormatPos(s.At), id.Name)
					}
				}
			}
			if typeContainsResourceHandle(tname, types) {
				source, err := resourceSourceForExpr(s.Value, funcs, module, imports, state)
				if err != nil {
					return err
				}
				if source.ambiguous {
					return fmt.Errorf("%s: resource expression mixes resource provenance", frontend.FormatPos(s.Value.Pos()))
				}
				if source.unknown {
					state.recordUnknownReturnResource()
				} else if paramIndex, path, ok := state.resourceParamOwner(source.name); ok {
					if err := state.recordReturnResourceParam(paramIndex, path, s.At); err != nil {
						return err
					}
				}
			}
			if err := state.recordReturnRegion(regionID, s.At); err != nil {
				return err
			}
			if !typesCompatibleWithNullPtr(returnType, tname, s.Value) {
				return fmt.Errorf("%s: return type mismatch: expected '%s', got '%s'", frontend.FormatPos(s.At), returnType, tname)
			}
		case *frontend.ThrowStmt:
			if state.throwType == "" {
				return fmt.Errorf("%s: throw is only allowed in throwing functions", frontend.FormatPos(s.At))
			}
			tname, _, err := checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return err
			}
			if !typesCompatibleWithNullPtr(state.throwType, tname, s.Value) {
				return fmt.Errorf("%s: throw type mismatch: expected '%s', got '%s'", frontend.FormatPos(s.At), state.throwType, tname)
			}
		case *frontend.DeferStmt:
			if err := validateDeferBodyControl(s.Body, 0); err != nil {
				return err
			}
			scopeID := regionNone
			if state.deferScopes != nil {
				if scoped, ok := state.deferScopes[s]; ok {
					scopeID = scoped
				}
			}
			captures := collectDeferCaptures(s.Body, locals)
			if err := withActiveScope(state, scopeID, func() error {
				return checkDeferBody(s.Body, locals, globals, funcs, types, module, imports, returnType, borrowedParams, inoutParams, state, effects, analysis)
			}); err != nil {
				return err
			}
			state.registerDeferCaptures(captures)
		case *frontend.IslandStmt:
			if err := effects.requireAll(s.At, []string{"alloc", "islands", "mem"}); err != nil {
				return err
			}
			sizeType, _, err := checkExprWithEffects(s.Size, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return err
			}
			if !isInt32Like(sizeType) {
				return fmt.Errorf("%s: island size must be i32/u8", frontend.FormatPos(s.At))
			}
			if err := state.enterIsland(s.Name); err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			if err := checkStmts(s.Body, locals, globals, funcs, types, module, imports, returnType, borrowedParams, inoutParams, state, effects, analysis); err != nil {
				state.exitIsland()
				return err
			}
			state.exitIsland()
		case *frontend.LetStmt:
			state.clearConsumed(s.Name)
			resolved, err := resolveTypeName(&s.Type, module, imports)
			if err != nil {
				return err
			}
			s.Type.Name = resolved
			if _, err := ensureTypeInfo(resolved, types); err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			valType := ""
			valRegion := regionNone
			handledFunctionSymbol := false
			if s.Type.Kind == frontend.TypeRefFunction {
				if _, ok := s.Value.(*frontend.IdentExpr); ok {
					if info, exists := locals[s.Name]; exists && info.FunctionTypeValue {
						if info.FunctionValue == "" {
							return fmt.Errorf("%s: function-typed local '%s' must be initialized with a non-capturing closure literal or named function/closure symbol in this MVP", frontend.FormatPos(s.At), s.Name)
						}
						valType = resolved
						valRegion = regionNone
						handledFunctionSymbol = true
					}
				}
			}
			if !handledFunctionSymbol {
				var checkErr error
				valType, valRegion, checkErr = checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
				if checkErr != nil {
					return checkErr
				}
			}
			if !typesCompatibleWithNullPtr(resolved, valType, s.Value) {
				return fmt.Errorf("%s: type mismatch: expected '%s', got '%s'", frontend.FormatPos(s.At), resolved, valType)
			}
			if valRegion >= 0 {
				scopeID := localScopeID(s.Name, state)
				if !state.isScopeWithin(scopeID, valRegion) {
					return fmt.Errorf(
						"%s: slice from scoped island cannot escape to outer scope (value: %s, target: %s)",
						frontend.FormatPos(s.At),
						formatRegionID(state, valRegion),
						formatScopeID(state, scopeID),
					)
				}
				state.regionVars[s.Name] = valRegion
				delete(state.unknownVars, s.Name)
				delete(state.unknownConflicts, s.Name)
			} else if valRegion < regionNone {
				state.regionVars[s.Name] = valRegion
				delete(state.unknownVars, s.Name)
				delete(state.unknownConflicts, s.Name)
			} else {
				delete(state.regionVars, s.Name)
				delete(state.unknownVars, s.Name)
				delete(state.unknownConflicts, s.Name)
			}
			if err := bindResourceTreeFromExpr(s.Name, resolved, s.Value, funcs, types, module, imports, state); err != nil {
				return err
			}
		case *frontend.AssignStmt:
			if idx, ok := s.Target.(*frontend.IndexExpr); ok {
				indexType, _, err := checkExprWithEffects(idx.Index, locals, globals, funcs, types, module, imports, state, effects, analysis)
				if err != nil {
					return err
				}
				if !isInt32Like(indexType) {
					return fmt.Errorf("%s: index must be i32/u8", frontend.FormatPos(idx.At))
				}
				if _, _, err := checkExprWithEffects(idx.Base, locals, globals, funcs, types, module, imports, state, effects, analysis); err != nil {
					return err
				}
			}
			if id, ok := s.Target.(*frontend.IdentExpr); ok {
				if err := state.checkNotConsumed(id.Name, s.At); err != nil {
					return err
				}
				if g, ok := globals[id.Name]; ok {
					if !g.Mutable {
						if g.Const {
							return fmt.Errorf("%s: cannot assign to const '%s'", frontend.FormatPos(s.At), id.Name)
						}
						return fmt.Errorf("%s: cannot assign to val '%s'", frontend.FormatPos(s.At), id.Name)
					}
					valType, _, err := checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
					if err != nil {
						return err
					}
					if !typesCompatibleWithNullPtr(g.TypeName, valType, s.Value) {
						return fmt.Errorf("%s: type mismatch: expected '%s', got '%s'", frontend.FormatPos(s.At), g.TypeName, valType)
					}
					continue
				}
			}
			targetInfo, targetType, err := resolveAssignTarget(s.Target, locals, types)
			if err != nil {
				return err
			}
			if err := checkLocalScope(targetInfo.Name, state, s.At); err != nil {
				return err
			}
			if !targetInfo.Mutable {
				if targetInfo.Const {
					return fmt.Errorf("%s: cannot assign to const '%s'", frontend.FormatPos(s.At), targetInfo.Name)
				}
				return fmt.Errorf("%s: cannot assign to val '%s'", frontend.FormatPos(s.At), targetInfo.Name)
			}
			if id, ok := s.Target.(*frontend.IdentExpr); ok {
				if localInfo, exists := locals[id.Name]; exists && localInfo.FunctionTypeValue {
					return fmt.Errorf("%s: reassignment of function-typed local '%s' is not supported in this MVP", frontend.FormatPos(s.At), id.Name)
				}
			}
			valType, valRegion, err := checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return err
			}
			if !typesCompatibleWithNullPtr(targetType, valType, s.Value) {
				return fmt.Errorf("%s: type mismatch: expected '%s', got '%s'", frontend.FormatPos(s.At), targetType, valType)
			}
			if valRegion < regionNone {
				if _, outParam := inoutParams[targetInfo.Name]; outParam {
					if err := checkBorrowedInoutEscape(s.Value, targetInfo.Name, s.At, locals, globals, funcs, types, module, imports, state, effects, analysis); err != nil {
						return err
					}
				}
			}
			if _, ok := s.Target.(*frontend.IndexExpr); !ok {
				if valRegion >= 0 {
					scopeID := localScopeID(targetInfo.Name, state)
					if !state.isScopeWithin(scopeID, valRegion) {
						return fmt.Errorf(
							"%s: slice from scoped island cannot escape to outer scope (value: %s, target: %s)",
							frontend.FormatPos(s.At),
							formatRegionID(state, valRegion),
							formatScopeID(state, scopeID),
						)
					}
					state.regionVars[targetInfo.Name] = valRegion
					delete(state.unknownVars, targetInfo.Name)
					delete(state.unknownConflicts, targetInfo.Name)
				} else if valRegion < regionNone {
					state.regionVars[targetInfo.Name] = valRegion
					delete(state.unknownVars, targetInfo.Name)
					delete(state.unknownConflicts, targetInfo.Name)
				} else {
					delete(state.regionVars, targetInfo.Name)
					delete(state.unknownVars, targetInfo.Name)
					delete(state.unknownConflicts, targetInfo.Name)
				}
				targetResourceName := targetInfo.Name
				if path, ok := resourcePathForExpr(s.Target); ok {
					targetResourceName = path
				}
				if err := bindResourceTreeFromExpr(targetResourceName, targetType, s.Value, funcs, types, module, imports, state); err != nil {
					return err
				}
			}
		case *frontend.IfStmt:
			condType, _, err := checkExprWithEffects(s.Cond, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return err
			}
			if !isConditionType(condType) {
				return fmt.Errorf("%s: condition must be bool or i32/u8", frontend.FormatPos(s.At))
			}
			scopeIDs := branchScopeInfo{thenID: regionNone, elseID: regionNone}
			if scoped, ok := state.ifScopes[s]; ok {
				scopeIDs = scoped
			}
			before := copyRegionVars(state.regionVars)
			beforeFlow := snapshotFlow(state)
			state.regionVars = copyRegionVars(before)
			restoreFlow(state, beforeFlow)
			if err := withActiveScope(state, scopeIDs.thenID, func() error {
				return checkStmts(s.Then, locals, globals, funcs, types, module, imports, returnType, borrowedParams, inoutParams, state, effects, analysis)
			}); err != nil {
				return err
			}
			thenVars := copyRegionVars(state.regionVars)
			thenFlow := snapshotFlow(state)
			var elseVars map[string]int
			var elseFlow flowSnapshot
			if len(s.Else) > 0 {
				state.regionVars = copyRegionVars(before)
				restoreFlow(state, beforeFlow)
				if err := withActiveScope(state, scopeIDs.elseID, func() error {
					return checkStmts(s.Else, locals, globals, funcs, types, module, imports, returnType, borrowedParams, inoutParams, state, effects, analysis)
				}); err != nil {
					return err
				}
				elseVars = copyRegionVars(state.regionVars)
				elseFlow = snapshotFlow(state)
			} else {
				elseVars = before
				elseFlow = beforeFlow
			}
			state.regionVars = mergeRegionVars(thenVars, elseVars)
			mergeFlow(state, thenFlow, elseFlow)
			recordMergeConflicts(state, thenVars, elseVars, "then", "else")
			markUnknownRegions(state)
		case *frontend.IfLetStmt:
			valueType, _, err := checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return err
			}
			valueInfo, valueInfoOK := types[valueType]
			if s.Pattern == nil {
				if _, ok := optionalElemName(valueType); !ok {
					return fmt.Errorf("%s: if let requires optional value, got '%s'", frontend.FormatPos(s.At), valueType)
				}
			} else if !valueInfoOK || (valueInfo.Kind != TypeOptional && valueInfo.Kind != TypeEnum) {
				return fmt.Errorf("%s: if let pattern requires optional or enum value, got '%s'", frontend.FormatPos(s.At), valueType)
			} else if err := validateIfLetPattern(s.Pattern, valueType, locals, globals, funcs, types, module, imports, state, effects, analysis); err != nil {
				return err
			}
			valueResourcePath := s.ValueLocal
			if valueResourcePath != "" {
				if err := bindResourceTreeFromExpr(valueResourcePath, valueType, s.Value, funcs, types, module, imports, state); err != nil {
					return err
				}
			} else if path, ok := resourcePathForExpr(s.Value); ok {
				valueResourcePath = path
			}
			scopeIDs := branchScopeInfo{thenID: regionNone, elseID: regionNone}
			if scoped, ok := state.ifLetScopes[s]; ok {
				scopeIDs = scoped
			}
			before := copyRegionVars(state.regionVars)
			beforeFlow := snapshotFlow(state)
			state.regionVars = copyRegionVars(before)
			restoreFlow(state, beforeFlow)
			if err := withActiveScope(state, scopeIDs.thenID, func() error {
				if err := bindPatternResourceLocals(s.Pattern, s.Name, valueResourcePath, valueType, types, module, imports, state); err != nil {
					return err
				}
				return checkStmts(s.Then, locals, globals, funcs, types, module, imports, returnType, borrowedParams, inoutParams, state, effects, analysis)
			}); err != nil {
				return err
			}
			thenVars := copyRegionVars(state.regionVars)
			thenFlow := snapshotFlow(state)
			var elseVars map[string]int
			var elseFlow flowSnapshot
			if len(s.Else) > 0 {
				state.regionVars = copyRegionVars(before)
				restoreFlow(state, beforeFlow)
				if err := withActiveScope(state, scopeIDs.elseID, func() error {
					return checkStmts(s.Else, locals, globals, funcs, types, module, imports, returnType, borrowedParams, inoutParams, state, effects, analysis)
				}); err != nil {
					return err
				}
				elseVars = copyRegionVars(state.regionVars)
				elseFlow = snapshotFlow(state)
			} else {
				elseVars = before
				elseFlow = beforeFlow
			}
			state.regionVars = mergeRegionVars(thenVars, elseVars)
			mergeFlow(state, thenFlow, elseFlow)
			recordMergeConflicts(state, thenVars, elseVars, "then", "else")
			markUnknownRegions(state)
		case *frontend.WhileStmt:
			condType, _, err := checkExprWithEffects(s.Cond, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return err
			}
			if !isConditionType(condType) {
				return fmt.Errorf("%s: condition must be bool or i32/u8", frontend.FormatPos(s.At))
			}
			bodyScopeID := regionNone
			if scoped, ok := state.whileScopes[s]; ok {
				bodyScopeID = scoped
			}
			before := copyRegionVars(state.regionVars)
			beforeFlow := snapshotFlow(state)
			state.regionVars = copyRegionVars(before)
			restoreFlow(state, beforeFlow)
			state.loopDepth++
			if err := withActiveScope(state, bodyScopeID, func() error {
				return checkStmts(s.Body, locals, globals, funcs, types, module, imports, returnType, borrowedParams, inoutParams, state, effects, analysis)
			}); err != nil {
				state.loopDepth--
				return err
			}
			state.loopDepth--
			bodyVars := copyRegionVars(state.regionVars)
			bodyFlow := snapshotFlow(state)
			state.regionVars = mergeRegionVars(before, bodyVars)
			mergeFlow(state, beforeFlow, bodyFlow)
			recordMergeConflicts(state, before, bodyVars, "before", "body")
			markUnknownRegions(state)
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				iterType, _, err := checkExprWithEffects(s.Iterable, locals, globals, funcs, types, module, imports, state, effects, analysis)
				if err != nil {
					return err
				}
				elemType, err := collectionElementType(iterType, types)
				if err != nil {
					return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
				}
				if loopInfo, ok := locals[s.Name]; ok && loopInfo.TypeName != elemType {
					return fmt.Errorf("%s: for collection element type mismatch: local '%s' is %s, iterable yields %s", frontend.FormatPos(s.At), s.Name, loopInfo.TypeName, elemType)
				}
			} else {
				startType, _, err := checkExprWithEffects(s.Start, locals, globals, funcs, types, module, imports, state, effects, analysis)
				if err != nil {
					return err
				}
				endType, _, err := checkExprWithEffects(s.End, locals, globals, funcs, types, module, imports, state, effects, analysis)
				if err != nil {
					return err
				}
				if !isInt32Like(startType) || !isInt32Like(endType) {
					return fmt.Errorf("%s: for range bounds must be i32/u8", frontend.FormatPos(s.At))
				}
			}
			bodyScopeID := regionNone
			if scoped, ok := state.forScopes[s]; ok {
				bodyScopeID = scoped
			}
			before := copyRegionVars(state.regionVars)
			beforeFlow := snapshotFlow(state)
			state.regionVars = copyRegionVars(before)
			restoreFlow(state, beforeFlow)
			state.loopDepth++
			if err := withActiveScope(state, bodyScopeID, func() error {
				return checkStmts(s.Body, locals, globals, funcs, types, module, imports, returnType, borrowedParams, inoutParams, state, effects, analysis)
			}); err != nil {
				state.loopDepth--
				return err
			}
			state.loopDepth--
			bodyVars := copyRegionVars(state.regionVars)
			bodyFlow := snapshotFlow(state)
			state.regionVars = mergeRegionVars(before, bodyVars)
			mergeFlow(state, beforeFlow, bodyFlow)
			recordMergeConflicts(state, before, bodyVars, "before", "body")
			markUnknownRegions(state)
		case *frontend.MatchStmt:
			scrutType, _, err := checkExprWithEffects(s.Value, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return err
			}
			scrutInfo, scrutInfoOK := types[scrutType]
			if !isInt32Like(scrutType) {
				info, ok := types[scrutType]
				if !ok || (info.Kind != TypeEnum && info.Kind != TypeOptional) {
					return fmt.Errorf("%s: match value must be enum or i32/u8", frontend.FormatPos(s.At))
				}
			}
			scrutineeResourcePath := s.ScrutineeLocal
			if scrutineeResourcePath != "" {
				if err := bindResourceTreeFromExpr(scrutineeResourcePath, scrutType, s.Value, funcs, types, module, imports, state); err != nil {
					return err
				}
			} else if path, ok := resourcePathForExpr(s.Value); ok {
				scrutineeResourcePath = path
			}
			seenDefault := false
			seenPatterns := map[string]frontend.Position{}
			before := copyRegionVars(state.regionVars)
			beforeFlow := snapshotFlow(state)
			merged := copyRegionVars(before)
			mergedFlow := beforeFlow
			labels := []string{"fallthrough"}
			caseScopes := state.matchCaseScopes[s]
			for i, c := range s.Cases {
				if seenDefault {
					return fmt.Errorf("%s: match default must be last", frontend.FormatPos(c.At))
				}
				if c.Default {
					seenDefault = true
				} else {
					patType := ""
					if some, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
						if !scrutInfoOK || scrutInfo.Kind != TypeOptional {
							return fmt.Errorf("%s: some pattern requires optional match value", frontend.FormatPos(some.At))
						}
						patType = optionalSomePatternType
					} else if enumPat, ok := c.Pattern.(*frontend.EnumCasePatternExpr); ok {
						caseType, caseInfo, found, err := resolveEnumCasePattern(enumPat, types, module, imports)
						if err != nil {
							return err
						}
						if !found {
							return fmt.Errorf("%s: unknown enum pattern '%s.%s'", frontend.FormatPos(enumPat.At), enumPat.TypeName, enumPat.CaseName)
						}
						if err := validateEnumCasePatternPayload(enumPat, caseType, caseInfo, module); err != nil {
							return err
						}
						patType = caseType
					} else {
						var err error
						patType, _, err = checkExprWithEffects(c.Pattern, locals, globals, funcs, types, module, imports, state, effects, analysis)
						if err != nil {
							return err
						}
					}
					if scrutInfoOK && scrutInfo.Kind == TypeOptional && patType != "none" && patType != optionalSomePatternType {
						return fmt.Errorf("%s: optional match supports only 'none', 'some(name)', and '_' patterns", frontend.FormatPos(c.At))
					}
					if !matchPatternCompatible(scrutType, patType, types) {
						return fmt.Errorf("%s: match pattern type mismatch: expected '%s', got '%s'", frontend.FormatPos(c.At), scrutType, patType)
					}
					if c.Guard == nil {
						if key := matchPatternKey(c.Pattern, patType); key != "" {
							if first, exists := seenPatterns[key]; exists {
								return fmt.Errorf("%s: duplicate match pattern (first at %s)", frontend.FormatPos(c.At), frontend.FormatPos(first))
							}
							seenPatterns[key] = c.At
						}
					}
				}
				state.regionVars = copyRegionVars(before)
				restoreFlow(state, beforeFlow)
				caseScopeID := regionNone
				if i < len(caseScopes) {
					caseScopeID = caseScopes[i]
				}
				if err := withActiveScope(state, caseScopeID, func() error {
					if err := bindPatternResourceLocals(c.Pattern, "", scrutineeResourcePath, scrutType, types, module, imports, state); err != nil {
						return err
					}
					if c.Guard != nil {
						guardType, _, err := checkExprWithEffects(c.Guard, locals, globals, funcs, types, module, imports, state, effects, analysis)
						if err != nil {
							return err
						}
						if guardType != "bool" {
							return fmt.Errorf("%s: match guard must be Bool", frontend.FormatPos(c.Guard.Pos()))
						}
					}
					return checkStmts(c.Body, locals, globals, funcs, types, module, imports, returnType, borrowedParams, inoutParams, state, effects, analysis)
				}); err != nil {
					return err
				}
				caseVars := copyRegionVars(state.regionVars)
				caseFlow := snapshotFlow(state)
				state.regionVars = mergeRegionVars(merged, caseVars)
				mergeFlow(state, mergedFlow, caseFlow)
				recordMergeConflicts(state, merged, caseVars, strings.Join(labels, "/"), fmt.Sprintf("case %d", i+1))
				merged = copyRegionVars(state.regionVars)
				mergedFlow = snapshotFlow(state)
				labels = append(labels, fmt.Sprintf("case %d", i+1))
			}
			if seenDefault {
				state.regionVars = merged
				restoreFlow(state, mergedFlow)
			} else {
				state.regionVars = mergeRegionVars(before, merged)
				mergeFlow(state, beforeFlow, mergedFlow)
				recordMergeConflicts(state, before, merged, "before", "cases")
			}
			markUnknownRegions(state)
		case *frontend.UnsafeStmt:
			scopeID := regionNone
			if scoped, ok := state.unsafeScopes[s]; ok {
				scopeID = scoped
			}
			state.enterUnsafe()
			if err := withActiveScope(state, scopeID, func() error {
				return checkStmts(s.Body, locals, globals, funcs, types, module, imports, returnType, borrowedParams, inoutParams, state, effects, analysis)
			}); err != nil {
				state.exitUnsafe()
				return err
			}
			state.exitUnsafe()
		case *frontend.ExprStmt:
			_, _, err := checkExprWithEffects(s.Expr, locals, globals, funcs, types, module, imports, state, effects, analysis)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s: unsupported statement", frontend.FormatPos(s.Pos()))
		}
		if err := state.checkPendingDeferCaptures(stmt.Pos()); err != nil {
			return err
		}
	}
	return nil
}

func matchPatternCompatible(scrutType, patternType string, types map[string]*TypeInfo) bool {
	if scrutType == patternType {
		return true
	}
	if patternType == optionalSomePatternType {
		if scrutInfo, ok := types[scrutType]; ok && scrutInfo.Kind == TypeOptional {
			return true
		}
	}
	if patternType == "none" {
		if scrutInfo, ok := types[scrutType]; ok && scrutInfo.Kind == TypeOptional {
			return true
		}
	}
	if isInt32Like(scrutType) && isInt32Like(patternType) {
		return true
	}
	scrutInfo, scrutOK := types[scrutType]
	patternInfo, patternOK := types[patternType]
	if scrutOK && patternOK && (scrutInfo.Kind == TypeEnum || patternInfo.Kind == TypeEnum) {
		return false
	}
	return false
}

func validateIfLetPattern(
	pattern frontend.Expr,
	valueType string,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
) error {
	valueInfo, ok := types[valueType]
	if !ok {
		return fmt.Errorf("unknown type '%s'", valueType)
	}
	patType := ""
	if some, ok := pattern.(*frontend.SomePatternExpr); ok {
		if valueInfo.Kind != TypeOptional {
			return fmt.Errorf("%s: some pattern requires optional match value", frontend.FormatPos(some.At))
		}
		patType = optionalSomePatternType
	} else if enumPat, ok := pattern.(*frontend.EnumCasePatternExpr); ok {
		caseType, caseInfo, found, err := resolveEnumCasePattern(enumPat, types, module, imports)
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("%s: unknown enum pattern '%s.%s'", frontend.FormatPos(enumPat.At), enumPat.TypeName, enumPat.CaseName)
		}
		if err := validateEnumCasePatternPayload(enumPat, caseType, caseInfo, module); err != nil {
			return err
		}
		patType = caseType
	} else {
		var err error
		patType, _, err = checkExprWithEffects(pattern, locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return err
		}
	}
	if valueInfo.Kind == TypeOptional && patType != "none" && patType != optionalSomePatternType {
		return fmt.Errorf("%s: optional if let supports only 'none', 'some(name)', and '_' patterns", frontend.FormatPos(pattern.Pos()))
	}
	if !matchPatternCompatible(valueType, patType, types) {
		return fmt.Errorf("%s: if let pattern type mismatch: expected '%s', got '%s'", frontend.FormatPos(pattern.Pos()), valueType, patType)
	}
	return nil
}

const optionalSomePatternType = "__optional_some_pattern"

func matchPatternKey(pattern frontend.Expr, patternType string) string {
	switch p := pattern.(type) {
	case *frontend.NumberExpr:
		return fmt.Sprintf("i32:%d", p.Value)
	case *frontend.NoneLitExpr:
		return "optional:none"
	case *frontend.SomePatternExpr:
		return "optional:some"
	case *frontend.EnumCasePatternExpr:
		if p.EnumType != "" {
			return "enum:" + p.EnumType + "." + p.CaseName
		}
		return "enum:" + p.TypeName + "." + p.CaseName
	case *frontend.FieldAccessExpr:
		if p.EnumType != "" {
			return "enum:" + p.EnumType + "." + p.Field
		}
		return patternType + ":" + p.Field
	default:
		return ""
	}
}

func uniqueHiddenLocal(prefix string, pos frontend.Position, locals map[string]LocalInfo) string {
	base := fmt.Sprintf("%s_%d_%d", prefix, pos.Line, pos.Col)
	name := base
	for i := 1; ; i++ {
		if _, exists := locals[name]; !exists {
			return name
		}
		name = fmt.Sprintf("%s_%d", base, i)
	}
}

func collectionElementType(typeName string, types map[string]*TypeInfo) (string, error) {
	info, err := ensureTypeInfo(typeName, types)
	if err != nil {
		return "", err
	}
	switch info.Kind {
	case TypeStr:
		return "u8", nil
	case TypeSlice:
		return info.ElemType, nil
	case TypeArray:
		return info.ElemType, nil
	default:
		return "", fmt.Errorf("for collection requires array, slice, or string, got '%s'", typeName)
	}
}

func collectExprLocals(
	expr frontend.Expr,
	locals map[string]LocalInfo,
	slotIndex *int,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	scopes *scopeInfo,
	globals map[string]GlobalInfo,
) error {
	switch e := expr.(type) {
	case *frontend.MatchExpr:
		if err := collectExprLocals(e.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
			return err
		}
		scrutType, err := inferExprTypeForDecl(e.Value, locals, globals, funcs, types, module, imports)
		if err != nil {
			return fmt.Errorf("%s: cannot infer match value type: %v", frontend.FormatPos(e.At), err)
		}
		scrutInfo, err := ensureTypeInfo(scrutType, types)
		if err != nil {
			return fmt.Errorf("%s: %v", frontend.FormatPos(e.At), err)
		}
		if scrutInfo.SlotCount != 1 && scrutInfo.Kind != TypeOptional && scrutInfo.Kind != TypeEnum {
			return fmt.Errorf("%s: match value must be single-slot", frontend.FormatPos(e.At))
		}
		resultType, err := inferMatchExprType(e, locals, globals, funcs, types, module, imports)
		if err != nil {
			return fmt.Errorf("%s: cannot infer match expression type: %v", frontend.FormatPos(e.At), err)
		}
		resultInfo, err := ensureTypeInfo(resultType, types)
		if err != nil {
			return fmt.Errorf("%s: %v", frontend.FormatPos(e.At), err)
		}
		e.ScrutineeLocal = uniqueHiddenLocal("__match_expr_value", e.At, locals)
		locals[e.ScrutineeLocal] = LocalInfo{Base: *slotIndex, SlotCount: scrutInfo.SlotCount, TypeName: scrutType, Mutable: false}
		if scopes != nil {
			scopes.localScopes[e.ScrutineeLocal] = scopes.currentScopeID()
		}
		*slotIndex += scrutInfo.SlotCount
		e.ResultLocal = uniqueHiddenLocal("__match_expr_result", e.At, locals)
		e.ResultType = resultType
		locals[e.ResultLocal] = LocalInfo{Base: *slotIndex, SlotCount: resultInfo.SlotCount, TypeName: resultType, Mutable: false}
		if scopes != nil {
			scopes.localScopes[e.ResultLocal] = scopes.currentScopeID()
		}
		*slotIndex += resultInfo.SlotCount
		caseScopeIDs := make([]int, len(e.Cases))
		for i, c := range e.Cases {
			if scopes != nil {
				caseScopeIDs[i] = scopes.enterScope()
			} else {
				caseScopeIDs[i] = regionNone
			}
			if !c.Default {
				if err := collectPatternLocals(c.Pattern, scrutType, locals, slotIndex, types, module, imports, scopes, globals); err != nil {
					return err
				}
			}
			if c.Guard != nil {
				if err := collectExprLocals(c.Guard, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
					return err
				}
			}
			if err := collectExprLocals(c.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
			}
		}
		if scopes != nil {
			scopes.matchExprScopes[e] = caseScopeIDs
		}
	case *frontend.CatchExpr:
		if err := collectExprLocals(e.Call, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
			return err
		}
		resultType, err := inferCatchExprType(e, locals, globals, funcs, types, module, imports)
		if err != nil {
			return fmt.Errorf("%s: cannot infer catch expression type: %v", frontend.FormatPos(e.At), err)
		}
		resultInfo, err := ensureTypeInfo(resultType, types)
		if err != nil {
			return fmt.Errorf("%s: %v", frontend.FormatPos(e.At), err)
		}
		errorInfo, err := ensureTypeInfo(e.ErrorType, types)
		if err != nil {
			return fmt.Errorf("%s: %v", frontend.FormatPos(e.At), err)
		}
		e.ErrorLocal = uniqueHiddenLocal("__catch_error", e.At, locals)
		locals[e.ErrorLocal] = LocalInfo{Base: *slotIndex, SlotCount: errorInfo.SlotCount, TypeName: e.ErrorType, Mutable: false}
		if scopes != nil {
			scopes.localScopes[e.ErrorLocal] = scopes.currentScopeID()
		}
		*slotIndex += errorInfo.SlotCount
		e.ResultLocal = uniqueHiddenLocal("__catch_result", e.At, locals)
		e.ResultType = resultType
		locals[e.ResultLocal] = LocalInfo{Base: *slotIndex, SlotCount: resultInfo.SlotCount, TypeName: resultType, Mutable: false}
		if scopes != nil {
			scopes.localScopes[e.ResultLocal] = scopes.currentScopeID()
		}
		*slotIndex += resultInfo.SlotCount
		caseScopeIDs := make([]int, len(e.Cases))
		for i, c := range e.Cases {
			if scopes != nil {
				caseScopeIDs[i] = scopes.enterScope()
			} else {
				caseScopeIDs[i] = regionNone
			}
			if !c.Default {
				if err := collectPatternLocals(c.Pattern, e.ErrorType, locals, slotIndex, types, module, imports, scopes, globals); err != nil {
					return err
				}
			}
			if c.Guard != nil {
				if err := collectExprLocals(c.Guard, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
					return err
				}
			}
			if err := collectExprLocals(c.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
			}
		}
		if scopes != nil {
			scopes.catchExprScopes[e] = caseScopeIDs
		}
	case *frontend.UnaryExpr:
		return collectExprLocals(e.X, locals, slotIndex, funcs, types, module, imports, scopes, globals)
	case *frontend.BinaryExpr:
		if err := collectExprLocals(e.Left, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
			return err
		}
		return collectExprLocals(e.Right, locals, slotIndex, funcs, types, module, imports, scopes, globals)
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			if err := collectExprLocals(arg, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if err := collectExprLocals(field.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
		}
	case *frontend.FieldAccessExpr:
		if e.Base != nil {
			return collectExprLocals(e.Base, locals, slotIndex, funcs, types, module, imports, scopes, globals)
		}
	case *frontend.IndexExpr:
		if err := collectExprLocals(e.Base, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
			return err
		}
		return collectExprLocals(e.Index, locals, slotIndex, funcs, types, module, imports, scopes, globals)
	case *frontend.TryExpr:
		return collectExprLocals(e.X, locals, slotIndex, funcs, types, module, imports, scopes, globals)
	case *frontend.AwaitExpr:
		return collectExprLocals(e.X, locals, slotIndex, funcs, types, module, imports, scopes, globals)
	}
	return nil
}

func collectPatternLocals(
	pattern frontend.Expr,
	scrutType string,
	locals map[string]LocalInfo,
	slotIndex *int,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	scopes *scopeInfo,
	globals map[string]GlobalInfo,
) error {
	info, err := ensureTypeInfo(scrutType, types)
	if err != nil {
		return err
	}
	switch pat := pattern.(type) {
	case *frontend.SomePatternExpr:
		if info.Kind != TypeOptional {
			return fmt.Errorf("%s: some pattern requires optional match value", frontend.FormatPos(pat.At))
		}
		if _, exists := globals[pat.Name]; exists {
			return fmt.Errorf("%s: local '%s' conflicts with global '%s'", frontend.FormatPos(pat.At), pat.Name, pat.Name)
		}
		if _, exists := locals[pat.Name]; exists {
			return fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(pat.At), pat.Name)
		}
		elemInfo, err := ensureTypeInfo(info.ElemType, types)
		if err != nil {
			return err
		}
		locals[pat.Name] = LocalInfo{Base: *slotIndex, SlotCount: elemInfo.SlotCount, TypeName: info.ElemType, Mutable: false}
		if scopes != nil {
			scopes.localScopes[pat.Name] = scopes.currentScopeID()
		}
		*slotIndex += elemInfo.SlotCount
	case *frontend.EnumCasePatternExpr:
		caseType, caseInfo, found, err := resolveEnumCasePattern(pat, types, module, imports)
		if err != nil {
			return err
		}
		if !found || caseType != scrutType {
			return fmt.Errorf("%s: enum pattern type mismatch", frontend.FormatPos(pat.At))
		}
		if err := validateEnumCasePatternPayload(pat, caseType, caseInfo, module); err != nil {
			return err
		}
		for i, binding := range pat.Bindings {
			if _, exists := globals[binding]; exists {
				return fmt.Errorf("%s: local '%s' conflicts with global '%s'", frontend.FormatPos(pat.At), binding, binding)
			}
			if _, exists := locals[binding]; exists {
				return fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(pat.At), binding)
			}
			locals[binding] = LocalInfo{Base: *slotIndex, SlotCount: caseInfo.PayloadSlots[i], TypeName: caseInfo.PayloadTypes[i], Mutable: false}
			if scopes != nil {
				scopes.localScopes[binding] = scopes.currentScopeID()
			}
			*slotIndex += caseInfo.PayloadSlots[i]
		}
	}
	return nil
}

func collectLocals(
	stmts []frontend.Stmt,
	locals map[string]LocalInfo,
	slotIndex *int,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	scopes *scopeInfo,
	globals map[string]GlobalInfo,
) error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if _, exists := globals[s.Name]; exists {
				return fmt.Errorf("%s: local '%s' conflicts with global '%s'", frontend.FormatPos(s.At), s.Name, s.Name)
			}
			if _, exists := locals[s.Name]; exists {
				return fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(s.At), s.Name)
			}
			resolved := ""
			if s.Type.Kind == frontend.TypeRefNamed && s.Type.Name == "" {
				inferred, err := inferExprTypeForDecl(s.Value, locals, globals, funcs, types, module, imports)
				if err != nil {
					return fmt.Errorf("%s: cannot infer type for '%s': %v", frontend.FormatPos(s.At), s.Name, err)
				}
				resolved = inferred
				s.Type = frontend.TypeRef{At: s.At, Kind: frontend.TypeRefNamed, Name: inferred}
			} else {
				var err error
				resolved, err = resolveTypeName(&s.Type, module, imports)
				if err != nil {
					return err
				}
				s.Type.Name = resolved
			}
			info, err := ensureTypeInfo(resolved, types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			functionValue := ""
			genericFunctionValue := false
			var functionCaptures []frontend.ClosureCapture
			functionTypeValue := s.Type.Kind == frontend.TypeRefFunction
			functionParamTypes := []string(nil)
			functionReturnType := ""
			functionEffects := []string(nil)
			if functionTypeValue {
				functionParamTypes, functionReturnType, functionEffects, err = functionTypeRefSignatureAndEffects(s.Type, module, imports)
				if err != nil {
					return err
				}
			}
			if closure, ok := s.Value.(*frontend.ClosureExpr); ok {
				if functionTypeValue {
					if err := validateFunctionTypeLiteralBinding(s.Name, s.Type, closure, locals, module, imports); err != nil {
						return err
					}
				}
				if err := configureClosureCaptures(closure, locals, funcs, types, module); err != nil {
					return err
				}
				functionValue = qualifyName(module, closure.Name)
				genericFunctionValue = len(closure.Decl.TypeParams) > 0
				functionCaptures = append([]frontend.ClosureCapture(nil), closure.Captures...)
			} else if functionTypeValue {
				switch init := s.Value.(type) {
				case *frontend.IdentExpr:
					resolved, err := validateFunctionTypeNamedSymbolBinding(s.Name, s.Type, init, locals, globals, funcs, module, imports)
					if err != nil {
						return err
					}
					functionValue = resolved
				case *frontend.CallExpr:
					resolvedCall, err := resolveCheckedCallName(init.Name, funcs, module, imports, init.At)
					if err != nil {
						return fmt.Errorf("%s: function-typed local '%s' must be initialized with a non-capturing closure literal or named function/closure symbol in this MVP", frontend.FormatPos(s.At), s.Name)
					}
					callSig, ok := funcs[resolvedCall]
					if ok && callSig.ReturnFunctionType && callSig.ReturnFunctionSymbol != "" {
						targetSig, ok := funcs[callSig.ReturnFunctionSymbol]
						if !ok {
							return fmt.Errorf("%s: unknown returned function symbol '%s'", frontend.FormatPos(init.At), callSig.ReturnFunctionSymbol)
						}
						if targetSig.Generic {
							return fmt.Errorf("%s: generic function symbol '%s' is not supported for function-typed local '%s' in this MVP", frontend.FormatPos(init.At), callSig.ReturnFunctionSymbol, s.Name)
						}
						if targetSig.ThrowsType != "" {
							return fmt.Errorf("%s: throwing function symbol '%s' is not supported for function-typed local '%s' in this MVP", frontend.FormatPos(init.At), callSig.ReturnFunctionSymbol, s.Name)
						}
						if err := validateFunctionTypeSymbolSignature(s.Name, s.Type, targetSig, module, imports, init.At); err != nil {
							return err
						}
						functionValue = callSig.ReturnFunctionSymbol
					}
				default:
					return fmt.Errorf("%s: function-typed local '%s' must be initialized with a non-capturing closure literal or named function/closure symbol in this MVP", frontend.FormatPos(s.At), s.Name)
				}
			}
			locals[s.Name] = LocalInfo{
				Base:                 *slotIndex,
				SlotCount:            info.SlotCount,
				TypeName:             resolved,
				Mutable:              s.Mutable,
				Const:                s.Const,
				FunctionValue:        functionValue,
				GenericFunctionValue: genericFunctionValue,
				FunctionCaptures:     functionCaptures,
				FunctionTypeValue:    functionTypeValue,
				FunctionParamTypes:   functionParamTypes,
				FunctionReturnType:   functionReturnType,
				FunctionEffects:      functionEffects,
			}
			if scopes != nil {
				scopes.localScopes[s.Name] = scopes.currentScopeID()
			}
			*slotIndex += info.SlotCount
			if err := collectExprLocals(s.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
		case *frontend.IslandStmt:
			if _, exists := globals[s.Name]; exists {
				return fmt.Errorf("%s: local '%s' conflicts with global '%s'", frontend.FormatPos(s.At), s.Name, s.Name)
			}
			if _, exists := locals[s.Name]; exists {
				return fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(s.At), s.Name)
			}
			islandInfo, err := ensureTypeInfo("island", types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			scopeID := regionNone
			if scopes != nil {
				scopeID = scopes.enterScope()
				scopes.localScopes[s.Name] = scopeID
				scopes.islandScopes[s.Name] = scopeID
			}
			locals[s.Name] = LocalInfo{
				Base:      *slotIndex,
				SlotCount: islandInfo.SlotCount,
				TypeName:  "island",
				Mutable:   false,
			}
			*slotIndex += islandInfo.SlotCount
			if err := collectLocals(s.Body, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
			}
		case *frontend.IfStmt:
			if err := collectExprLocals(s.Cond, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			thenScopeID := regionNone
			elseScopeID := regionNone
			if scopes != nil {
				thenScopeID = scopes.enterScope()
			}
			if err := collectLocals(s.Then, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
			}
			if len(s.Else) > 0 {
				if scopes != nil {
					elseScopeID = scopes.enterScope()
				}
				if err := collectLocals(s.Else, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
					return err
				}
				if scopes != nil {
					scopes.exitScope()
				}
			}
			if scopes != nil {
				scopes.ifScopes[s] = branchScopeInfo{thenID: thenScopeID, elseID: elseScopeID}
			}
		case *frontend.IfLetStmt:
			if err := collectExprLocals(s.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			valueType, err := inferExprTypeForDecl(s.Value, locals, globals, funcs, types, module, imports)
			if err != nil {
				return fmt.Errorf("%s: cannot infer if-let value type: %v", frontend.FormatPos(s.At), err)
			}
			valueInfo, err := ensureTypeInfo(valueType, types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			if s.Pattern == nil && valueInfo.Kind != TypeOptional {
				return fmt.Errorf("%s: if let requires optional value, got '%s'", frontend.FormatPos(s.At), valueType)
			}
			if s.Pattern != nil && valueInfo.Kind != TypeOptional && valueInfo.Kind != TypeEnum {
				return fmt.Errorf("%s: if let pattern requires optional or enum value, got '%s'", frontend.FormatPos(s.At), valueType)
			}
			if s.Pattern == nil {
				if _, exists := globals[s.Name]; exists {
					return fmt.Errorf("%s: local '%s' conflicts with global '%s'", frontend.FormatPos(s.At), s.Name, s.Name)
				}
				if _, exists := locals[s.Name]; exists {
					return fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(s.At), s.Name)
				}
			}
			elemInfo, err := ensureTypeInfo(valueInfo.ElemType, types)
			if s.Pattern == nil && err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			s.ValueLocal = uniqueHiddenLocal("__iflet_value", s.At, locals)
			locals[s.ValueLocal] = LocalInfo{
				Base:      *slotIndex,
				SlotCount: valueInfo.SlotCount,
				TypeName:  valueType,
				Mutable:   false,
			}
			if scopes != nil {
				scopes.localScopes[s.ValueLocal] = scopes.currentScopeID()
			}
			*slotIndex += valueInfo.SlotCount
			thenScopeID := regionNone
			elseScopeID := regionNone
			if scopes != nil {
				thenScopeID = scopes.enterScope()
			}
			if s.Pattern == nil {
				locals[s.Name] = LocalInfo{
					Base:      *slotIndex,
					SlotCount: elemInfo.SlotCount,
					TypeName:  valueInfo.ElemType,
					Mutable:   false,
				}
				if scopes != nil {
					scopes.localScopes[s.Name] = scopes.currentScopeID()
				}
				*slotIndex += elemInfo.SlotCount
			} else if err := collectPatternLocals(s.Pattern, valueType, locals, slotIndex, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if err := collectLocals(s.Then, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
			}
			if len(s.Else) > 0 {
				if scopes != nil {
					elseScopeID = scopes.enterScope()
				}
				if err := collectLocals(s.Else, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
					return err
				}
				if scopes != nil {
					scopes.exitScope()
				}
			}
			if scopes != nil {
				scopes.ifLetScopes[s] = branchScopeInfo{thenID: thenScopeID, elseID: elseScopeID}
			}
		case *frontend.WhileStmt:
			if err := collectExprLocals(s.Cond, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			scopeID := regionNone
			if scopes != nil {
				scopeID = scopes.enterScope()
			}
			if err := collectLocals(s.Body, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
				scopes.whileScopes[s] = scopeID
			}
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				if err := collectExprLocals(s.Iterable, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
					return err
				}
			} else {
				if err := collectExprLocals(s.Start, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
					return err
				}
				if err := collectExprLocals(s.End, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
					return err
				}
			}
			scopeID := regionNone
			if scopes != nil {
				scopeID = scopes.enterScope()
			}
			if _, exists := globals[s.Name]; exists {
				return fmt.Errorf("%s: local '%s' conflicts with global '%s'", frontend.FormatPos(s.At), s.Name, s.Name)
			}
			if _, exists := locals[s.Name]; exists {
				return fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(s.At), s.Name)
			}
			loopType := "i32"
			var iterableInfo *TypeInfo
			if s.Iterable != nil {
				iterType, err := inferExprTypeForDecl(s.Iterable, locals, globals, funcs, types, module, imports)
				if err != nil {
					return fmt.Errorf("%s: cannot infer for collection type: %v", frontend.FormatPos(s.At), err)
				}
				elemType, err := collectionElementType(iterType, types)
				if err != nil {
					return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
				}
				iterableInfo, err = ensureTypeInfo(iterType, types)
				if err != nil {
					return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
				}
				loopType = elemType
			}
			info, err := ensureTypeInfo(loopType, types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			locals[s.Name] = LocalInfo{
				Base:      *slotIndex,
				SlotCount: info.SlotCount,
				TypeName:  loopType,
				Mutable:   false,
			}
			if scopes != nil {
				scopes.localScopes[s.Name] = scopes.currentScopeID()
			}
			*slotIndex += info.SlotCount
			if s.Iterable != nil {
				s.IterableLocal = uniqueHiddenLocal("__for_iter", s.At, locals)
				locals[s.IterableLocal] = LocalInfo{
					Base:      *slotIndex,
					SlotCount: iterableInfo.SlotCount,
					TypeName:  iterableInfo.Name,
					Mutable:   false,
				}
				if scopes != nil {
					scopes.localScopes[s.IterableLocal] = scopes.currentScopeID()
				}
				*slotIndex += iterableInfo.SlotCount
				s.IndexLocal = uniqueHiddenLocal("__for_index", s.At, locals)
				indexInfo := types["i32"]
				locals[s.IndexLocal] = LocalInfo{
					Base:      *slotIndex,
					SlotCount: indexInfo.SlotCount,
					TypeName:  "i32",
					Mutable:   false,
				}
				if scopes != nil {
					scopes.localScopes[s.IndexLocal] = scopes.currentScopeID()
				}
				*slotIndex += indexInfo.SlotCount
			}
			s.EndLocal = uniqueHiddenLocal("__for_end", s.At, locals)
			locals[s.EndLocal] = LocalInfo{
				Base:      *slotIndex,
				SlotCount: 1,
				TypeName:  "i32",
				Mutable:   false,
			}
			if scopes != nil {
				scopes.localScopes[s.EndLocal] = scopes.currentScopeID()
			}
			*slotIndex += info.SlotCount
			if err := collectLocals(s.Body, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
				scopes.forScopes[s] = scopeID
			}
		case *frontend.MatchStmt:
			if err := collectExprLocals(s.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			scrutType, err := inferExprTypeForDecl(s.Value, locals, globals, funcs, types, module, imports)
			if err != nil {
				return fmt.Errorf("%s: cannot infer match value type: %v", frontend.FormatPos(s.At), err)
			}
			info, err := ensureTypeInfo(scrutType, types)
			if err != nil {
				return fmt.Errorf("%s: %v", frontend.FormatPos(s.At), err)
			}
			if info.SlotCount != 1 && info.Kind != TypeOptional && info.Kind != TypeEnum {
				return fmt.Errorf("%s: match value must be single-slot", frontend.FormatPos(s.At))
			}
			s.ScrutineeLocal = uniqueHiddenLocal("__match_value", s.At, locals)
			locals[s.ScrutineeLocal] = LocalInfo{
				Base:      *slotIndex,
				SlotCount: info.SlotCount,
				TypeName:  scrutType,
				Mutable:   false,
			}
			if scopes != nil {
				scopes.localScopes[s.ScrutineeLocal] = scopes.currentScopeID()
			}
			*slotIndex += info.SlotCount
			caseScopeIDs := make([]int, len(s.Cases))
			for i, c := range s.Cases {
				if scopes != nil {
					caseScopeIDs[i] = scopes.enterScope()
				} else {
					caseScopeIDs[i] = regionNone
				}
				if some, ok := c.Pattern.(*frontend.SomePatternExpr); ok {
					if info.Kind != TypeOptional {
						return fmt.Errorf("%s: some pattern requires optional match value", frontend.FormatPos(some.At))
					}
					if _, exists := globals[some.Name]; exists {
						return fmt.Errorf("%s: local '%s' conflicts with global '%s'", frontend.FormatPos(some.At), some.Name, some.Name)
					}
					if _, exists := locals[some.Name]; exists {
						return fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(some.At), some.Name)
					}
					elemInfo, err := ensureTypeInfo(info.ElemType, types)
					if err != nil {
						return fmt.Errorf("%s: %v", frontend.FormatPos(some.At), err)
					}
					locals[some.Name] = LocalInfo{
						Base:      *slotIndex,
						SlotCount: elemInfo.SlotCount,
						TypeName:  info.ElemType,
						Mutable:   false,
					}
					if scopes != nil {
						scopes.localScopes[some.Name] = scopes.currentScopeID()
					}
					*slotIndex += elemInfo.SlotCount
				}
				if enumPat, ok := c.Pattern.(*frontend.EnumCasePatternExpr); ok {
					caseType, caseInfo, found, err := resolveEnumCasePattern(enumPat, types, module, imports)
					if err != nil {
						return err
					}
					if !found || caseType != scrutType {
						return fmt.Errorf("%s: enum pattern type mismatch", frontend.FormatPos(enumPat.At))
					}
					if err := validateEnumCasePatternPayload(enumPat, caseType, caseInfo, module); err != nil {
						return err
					}
					for j, binding := range enumPat.Bindings {
						if _, exists := globals[binding]; exists {
							return fmt.Errorf("%s: local '%s' conflicts with global '%s'", frontend.FormatPos(enumPat.At), binding, binding)
						}
						if _, exists := locals[binding]; exists {
							return fmt.Errorf("%s: duplicate local '%s'", frontend.FormatPos(enumPat.At), binding)
						}
						slots := 1
						if j < len(caseInfo.PayloadSlots) {
							slots = caseInfo.PayloadSlots[j]
						}
						locals[binding] = LocalInfo{
							Base:      *slotIndex,
							SlotCount: slots,
							TypeName:  caseInfo.PayloadTypes[j],
							Mutable:   false,
						}
						if scopes != nil {
							scopes.localScopes[binding] = scopes.currentScopeID()
						}
						*slotIndex += slots
					}
				}
				if c.Guard != nil {
					if err := collectExprLocals(c.Guard, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
						return err
					}
				}
				if err := collectLocals(c.Body, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
					return err
				}
				if scopes != nil {
					scopes.exitScope()
				}
			}
			if scopes != nil {
				scopes.matchCaseScopes[s] = caseScopeIDs
			}
		case *frontend.UnsafeStmt:
			scopeID := regionNone
			if scopes != nil {
				scopeID = scopes.enterScope()
			}
			if err := collectLocals(s.Body, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
				scopes.unsafeScopes[s] = scopeID
			}
		case *frontend.DeferStmt:
			scopeID := regionNone
			if scopes != nil {
				scopeID = scopes.enterScope()
			}
			if err := collectLocals(s.Body, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if scopes != nil {
				scopes.exitScope()
				scopes.deferScopes[s] = scopeID
			}
		case *frontend.ExprStmt:
			if err := collectExprLocals(s.Expr, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
		case *frontend.ReturnStmt:
			if err := collectExprLocals(s.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
		case *frontend.ThrowStmt:
			if err := collectExprLocals(s.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
		case *frontend.PrintStmt:
			if err := collectExprLocals(s.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
		case *frontend.FreeStmt:
			if err := collectExprLocals(s.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
		case *frontend.AssignStmt:
			if err := collectExprLocals(s.Target, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
			if err := collectExprLocals(s.Value, locals, slotIndex, funcs, types, module, imports, scopes, globals); err != nil {
				return err
			}
		}
	}
	return nil
}
