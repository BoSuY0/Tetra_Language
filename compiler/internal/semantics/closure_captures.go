package semantics

import (
	"fmt"

	"tetra_language/compiler/internal/frontend"
)

func collectDeferCaptures(stmts []frontend.Stmt, locals map[string]LocalInfo) map[string]frontend.Position {
	captures := make(map[string]frontend.Position)
	collectStmtCaptures(stmts, locals, map[string]bool{}, captures, captureOwnershipPaths)
	return captures
}

func collectClosureCaptures(fn *frontend.FuncDecl, locals map[string]LocalInfo) map[string]frontend.Position {
	captures := make(map[string]frontend.Position)
	bound := make(map[string]bool, len(fn.Params))
	for _, param := range fn.Params {
		bound[param.Name] = true
	}
	collectStmtCaptures(fn.Body, locals, bound, captures, captureLocalRoots)
	return captures
}

type captureMode int

const (
	captureLocalRoots captureMode = iota
	captureOwnershipPaths
)

func unsupportedFunctionTypedCaptureError(pos frontend.Position, localName, captured string) error {
	return lifetimeDiagnosticf(pos, "function-typed local '%s' captures '%s'; captures are not supported for function-typed values in this MVP (use a let-bound ptr closure and call it directly, or pass a non-capturing named function/closure symbol); closure lifetime/ABI evidence is only available for local direct calls", localName, captured)
}

func unsupportedFunctionTypedCaptureAliasError(pos frontend.Position, localName, capturedLocal string) error {
	return lifetimeDiagnosticf(pos, "function-typed local '%s' aliases capturing closure '%s'; closure lifetime/ABI evidence is only available for local direct calls", localName, capturedLocal)
}

func unsupportedFunctionTypedStorageCaptureError(pos frontend.Position, targetName string, envSlots int) error {
	if envSlots > FnPtrEnvSlotCount {
		return lifetimeDiagnosticf(pos, "function-typed storage '%s' captures %d environment slots; function-typed storage supports at most %d fnptr environment slots within the supported fnptr ABI", targetName, envSlots, FnPtrEnvSlotCount)
	}
	return lifetimeDiagnosticf(pos, "function-typed storage '%s' has unsupported captured environment size", targetName)
}

func unsupportedFunctionTypedReturnCaptureError(pos frontend.Position, valueName string, envSlots int) error {
	if envSlots > FnPtrEnvSlotCount {
		return lifetimeDiagnosticf(pos, "function-typed return '%s' captures %d environment slots; function-typed returns support at most %d fnptr environment slots within the supported fnptr ABI", valueName, envSlots, FnPtrEnvSlotCount)
	}
	return lifetimeDiagnosticf(pos, "function-typed return '%s' has unsupported captured environment size", valueName)
}

func functionCaptureSlotCount(captures []frontend.ClosureCapture, types map[string]*TypeInfo) (int, error) {
	slots := 0
	for _, capture := range captures {
		info, err := ensureTypeInfo(capture.Type.Name, types)
		if err != nil {
			return 0, err
		}
		slots += info.SlotCount
	}
	return slots, nil
}

func configureClosureCaptures(
	closure *frontend.ClosureExpr,
	locals map[string]LocalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	allowMutableValueCaptures bool,
	captureBoundaryPhrase ...string,
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
	type unsupportedCapture struct {
		pos      frontend.Position
		name     string
		typeName string
	}
	var unsupported []unsupportedCapture
	for len(capturePositions) > 0 {
		name, pos, _ := firstCapture(capturePositions)
		delete(capturePositions, name)
		info, ok := locals[name]
		if !ok {
			return fmt.Errorf("%s: internal error: closure capture '%s' is missing from locals", frontend.FormatPos(pos), name)
		}
		if info.Mutable && !allowMutableValueCaptures {
			return lifetimeDiagnosticf(pos, "closure capture '%s' is mutable; %s", name, mutableClosureCaptureUnsupportedText())
		}
		if info.SurfaceFramePixelsSource != "" && len(captureBoundaryPhrase) > 0 && captureBoundaryPhrase[0] != "" {
			return lifetimeDiagnosticf(pos, "surface frame pixels cannot escape via function capture; keep Frame.pixels local to the active Surface frame")
		}
		typeRef := frontend.TypeRef{At: pos, Kind: frontend.TypeRefNamed, Name: info.TypeName}
		if !isClosureCaptureType(info.TypeName, types) {
			unsupported = append(unsupported, unsupportedCapture{pos: pos, name: name, typeName: info.TypeName})
		}
		captures = append(captures, frontend.ClosureCapture{At: pos, Name: name, Type: typeRef, Mutable: info.Mutable})
		closure.Decl.Params = append(closure.Decl.Params, frontend.ParamDecl{At: pos, Name: name, Type: typeRef})
		sig.ParamNames = append(sig.ParamNames, name)
		sig.ParamTypes = append(sig.ParamTypes, info.TypeName)
		sig.ParamOwnership = append(sig.ParamOwnership, "")
		captureParamSlots += info.SlotCount
	}
	if len(unsupported) > 0 && (len(captureBoundaryPhrase) == 0 || captureParamSlots <= FnPtrEnvSlotCount) {
		capture := unsupported[0]
		if len(captureBoundaryPhrase) > 0 && captureBoundaryPhrase[0] != "" {
			return lifetimeDiagnosticf(
				capture.pos,
				"%s captures unsupported local '%s' of type '%s'; %s",
				captureBoundaryPhrase[0],
				capture.name,
				capture.typeName,
				functionTypedCaptureSupportedSubsetText(),
			)
		}
		return lifetimeDiagnosticf(capture.pos, "closure capture '%s' has unsupported type '%s'; %s", capture.name, capture.typeName, closureCaptureSupportedSubsetText())
	}
	sig.ParamSlots += captureParamSlots
	funcs[fullName] = sig
	closure.Captures = captures
	return nil
}

func closureCaptureSupportedSubsetText() string {
	return "only immutable local Int/Bool/String, simple struct, enum, and optional captures without ptr/resource fields are supported by the direct ptr-closure capture ABI"
}

func functionTypedCaptureSupportedSubsetText() string {
	return "only immutable local Int/Bool/String, simple struct, enum, and optional captures without ptr/resource fields are supported within the supported fnptr ABI"
}

func mutableClosureCaptureUnsupportedText() string {
	return "direct ptr closure calls would observe mutable locals by reference, so use a function-typed fnptr binding for by-value snapshot capture"
}

func closureLiteralDirectCallCaptureText() string {
	return "only let-bound local direct calls can capture immutable Int/Bool/String values and simple structs without ptr/resource fields under the direct ptr-closure ABI"
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
	case TypeEnum:
		if visiting[typeName] {
			return false
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, enumCase := range info.EnumCases {
			for _, payload := range enumCase.PayloadTypes {
				if payload == "ptr" || typeContainsResourceHandle(payload, types) {
					return false
				}
				if !isClosureCaptureTypeVisiting(payload, types, visiting) {
					return false
				}
			}
		}
		return true
	case TypeOptional:
		if info.ElemType == "ptr" || typeContainsResourceHandle(info.ElemType, types) {
			return false
		}
		return isClosureCaptureTypeVisiting(info.ElemType, types, visiting)
	default:
		return false
	}
}

func appendClosureCaptureArgs(call *frontend.CallExpr, local LocalInfo) error {
	if len(local.FunctionCaptures) == 0 {
		return nil
	}
	if len(call.TypeArgs) > 0 {
		return lifetimeDiagnosticf(call.At, "explicit type arguments are not supported for captured closure '%s'", call.Name)
	}
	labeledCall := len(call.ArgLabels) > 0
	if labeledCall && len(call.ArgLabels) != len(call.Args) {
		return fmt.Errorf("%s: internal error: call argument labels are inconsistent", frontend.FormatPos(call.At))
	}
	if labeledCall {
		for i, label := range call.ArgLabels {
			if label == "" {
				return fmt.Errorf("%s: cannot mix labeled and unlabeled arguments in captured closure '%s'", frontend.FormatPos(call.Args[i].Pos()), call.Name)
			}
		}
	}
	for _, capture := range local.FunctionCaptures {
		call.Args = append(call.Args, &frontend.IdentExpr{At: capture.At, Name: capture.Name})
		if labeledCall {
			call.ArgLabels = append(call.ArgLabels, capture.Name)
		}
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

func collectStmtCaptures(stmts []frontend.Stmt, locals map[string]LocalInfo, bound map[string]bool, captures map[string]frontend.Position, mode captureMode) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.PrintStmt:
			collectExprCaptures(s.Value, locals, bound, captures, mode)
		case *frontend.ExpectStmt:
			collectExprCaptures(s.Cond, locals, bound, captures, mode)
		case *frontend.ReturnStmt:
			collectExprCaptures(s.Value, locals, bound, captures, mode)
		case *frontend.ThrowStmt:
			collectExprCaptures(s.Value, locals, bound, captures, mode)
		case *frontend.FreeStmt:
			collectExprCaptures(s.Value, locals, bound, captures, mode)
		case *frontend.LetStmt:
			collectExprCaptures(s.Value, locals, bound, captures, mode)
			bound[s.Name] = true
		case *frontend.AssignStmt:
			collectExprCaptures(s.Target, locals, bound, captures, mode)
			collectExprCaptures(s.Value, locals, bound, captures, mode)
		case *frontend.IfStmt:
			collectExprCaptures(s.Cond, locals, bound, captures, mode)
			collectStmtCaptures(s.Then, locals, cloneBoolMap(bound), captures, mode)
			collectStmtCaptures(s.Else, locals, cloneBoolMap(bound), captures, mode)
		case *frontend.IfLetStmt:
			collectExprCaptures(s.Value, locals, bound, captures, mode)
			thenBound := cloneBoolMap(bound)
			addPatternCaptureBindings(s.Pattern, s.Name, thenBound)
			collectStmtCaptures(s.Then, locals, thenBound, captures, mode)
			collectStmtCaptures(s.Else, locals, cloneBoolMap(bound), captures, mode)
		case *frontend.WhileStmt:
			collectExprCaptures(s.Cond, locals, bound, captures, mode)
			collectStmtCaptures(s.Body, locals, cloneBoolMap(bound), captures, mode)
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				collectExprCaptures(s.Iterable, locals, bound, captures, mode)
			} else {
				collectExprCaptures(s.Start, locals, bound, captures, mode)
				collectExprCaptures(s.End, locals, bound, captures, mode)
			}
			bodyBound := cloneBoolMap(bound)
			bodyBound[s.Name] = true
			collectStmtCaptures(s.Body, locals, bodyBound, captures, mode)
		case *frontend.MatchStmt:
			collectExprCaptures(s.Value, locals, bound, captures, mode)
			for _, c := range s.Cases {
				caseBound := cloneBoolMap(bound)
				addPatternCaptureBindings(c.Pattern, "", caseBound)
				if c.Guard != nil {
					collectExprCaptures(c.Guard, locals, caseBound, captures, mode)
				}
				collectStmtCaptures(c.Body, locals, caseBound, captures, mode)
			}
		case *frontend.IslandStmt:
			collectExprCaptures(s.Size, locals, bound, captures, mode)
			bodyBound := cloneBoolMap(bound)
			bodyBound[s.Name] = true
			collectStmtCaptures(s.Body, locals, bodyBound, captures, mode)
		case *frontend.UnsafeStmt:
			collectStmtCaptures(s.Body, locals, cloneBoolMap(bound), captures, mode)
		case *frontend.DeferStmt:
			collectStmtCaptures(s.Body, locals, cloneBoolMap(bound), captures, mode)
		case *frontend.ExprStmt:
			collectExprCaptures(s.Expr, locals, bound, captures, mode)
		}
	}
}

func collectExprCaptures(expr frontend.Expr, locals map[string]LocalInfo, bound map[string]bool, captures map[string]frontend.Position, mode captureMode) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		if _, ok := locals[e.Name]; ok && !bound[e.Name] {
			recordCapture(captures, e.Name, e.At)
		}
	case *frontend.FieldAccessExpr:
		if mode == captureOwnershipPaths {
			if path, pos, ok := localOwnershipCapturePath(e, locals, bound); ok {
				recordCapture(captures, path, pos)
				return
			}
		}
		collectExprCaptures(e.Base, locals, bound, captures, mode)
	case *frontend.IndexExpr:
		if mode == captureOwnershipPaths {
			if path, pos, ok := localOwnershipCapturePath(e, locals, bound); ok {
				recordCapture(captures, path, pos)
			} else {
				collectExprCaptures(e.Base, locals, bound, captures, mode)
			}
			collectExprCaptures(e.Index, locals, bound, captures, mode)
			return
		}
		collectExprCaptures(e.Base, locals, bound, captures, mode)
		collectExprCaptures(e.Index, locals, bound, captures, mode)
	case *frontend.BinaryExpr:
		collectExprCaptures(e.Left, locals, bound, captures, mode)
		collectExprCaptures(e.Right, locals, bound, captures, mode)
	case *frontend.UnaryExpr:
		collectExprCaptures(e.X, locals, bound, captures, mode)
	case *frontend.TryExpr:
		collectExprCaptures(e.X, locals, bound, captures, mode)
	case *frontend.AwaitExpr:
		collectExprCaptures(e.X, locals, bound, captures, mode)
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			collectExprCaptures(arg, locals, bound, captures, mode)
		}
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			collectExprCaptures(field.Value, locals, bound, captures, mode)
		}
	case *frontend.MatchExpr:
		collectExprCaptures(e.Value, locals, bound, captures, mode)
		for _, c := range e.Cases {
			caseBound := cloneBoolMap(bound)
			addPatternCaptureBindings(c.Pattern, "", caseBound)
			if c.Guard != nil {
				collectExprCaptures(c.Guard, locals, caseBound, captures, mode)
			}
			collectExprCaptures(c.Value, locals, caseBound, captures, mode)
		}
	case *frontend.CatchExpr:
		collectExprCaptures(e.Call, locals, bound, captures, mode)
		for _, c := range e.Cases {
			caseBound := cloneBoolMap(bound)
			addPatternCaptureBindings(c.Pattern, "", caseBound)
			if c.Guard != nil {
				collectExprCaptures(c.Guard, locals, caseBound, captures, mode)
			}
			collectExprCaptures(c.Value, locals, caseBound, captures, mode)
		}
	}
}

func recordCapture(captures map[string]frontend.Position, name string, pos frontend.Position) {
	if name == "" {
		return
	}
	if _, exists := captures[name]; !exists {
		captures[name] = pos
	}
}

func localOwnershipCapturePath(expr frontend.Expr, locals map[string]LocalInfo, bound map[string]bool) (string, frontend.Position, bool) {
	base, _, pos, ok := splitOwnershipPath(expr)
	if !ok || base == "" || bound[base] {
		return "", pos, false
	}
	if _, ok := locals[base]; !ok {
		return "", pos, false
	}
	path, ok := canonicalOwnershipAccessPath(expr)
	if !ok {
		return "", pos, false
	}
	return path, pos, true
}

func addPatternCaptureBindings(pattern frontend.Expr, name string, bound map[string]bool) {
	if name != "" {
		bound[name] = true
	}
	if pattern == nil {
		return
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
