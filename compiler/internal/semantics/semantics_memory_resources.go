package semantics

import (
	"fmt"
	"sort"
	"strings"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/islandkernel"
	semanticspolicy "tetra_language/compiler/internal/semantics/policy"
	semanticsregions "tetra_language/compiler/internal/semantics/regions"
)

// ---- callable_escape.go ----

type callableEscapeBoundary string

const (
	callableBoundaryLocal       callableEscapeBoundary = "local"
	callableBoundaryReturn      callableEscapeBoundary = "return"
	callableBoundaryGlobal      callableEscapeBoundary = "global"
	callableBoundaryStructField callableEscapeBoundary = "struct-field"
	callableBoundaryEnumPayload callableEscapeBoundary = "enum-payload"
	callableBoundaryCallback    callableEscapeBoundary = "callback"
	callableBoundaryThread      callableEscapeBoundary = "thread"
)

func classifyCallableEscape(
	boundary callableEscapeBoundary,
	captures []frontend.ClosureCapture,
	types map[string]*TypeInfo,
) (CallableEscapeKind, bool, error) {
	slots, err := functionCaptureSlotCount(captures, types)
	if err != nil {
		return "", false, err
	}
	if capture, surfaceType, ok := surfaceEphemeralCallableCapture(captures, types); ok {
		return "", false, lifetimeDiagnosticf(
			capture.At,
			("surface value '%s' cannot escape via function capture; keep " +
				"Surface Frame/Event/DrawContext values local to the active Surface turn"),
			surfaceType,
		)
	}
	if slots <= FnPtrEnvSlotCount && boundary != callableBoundaryThread {
		return CallableEscapeLocalSnapshot, false, nil
	}

	escapeKind := CallableEscapeHeap
	if boundary == callableBoundaryGlobal {
		escapeKind = CallableEscapeGlobal
	}
	if boundary == callableBoundaryThread {
		escapeKind = CallableEscapeThread
	}
	for _, capture := range captures {
		captureDecision := islandkernel.CanCaptureClosure(islandKernelCallableCaptureRequest(capture))
		if captureDecision.Decision != islandkernel.Accept {
			return "", false, lifetimeDiagnosticf(
				capture.At,
				"closure capture '%s' rejected by island kernel (%s)",
				capture.Name,
				captureDecision.Reason.Code,
			)
		}
		if capture.Mutable {
			return "", false, unsupportedCallableMutableCaptureEscapeError(
				capture.At,
				escapeKind,
				capture.Name,
			)
		}
		if _, err := ensureTypeInfo(capture.Type.Name, types); err != nil {
			return "", false, err
		}
		if !isClosureCaptureType(capture.Type.Name, types) {
			return "", false, unsupportedCallableResourceCaptureEscapeError(
				capture.At,
				capture.Name,
				capture.Type.Name,
			)
		}
	}
	return escapeKind, true, nil
}

func islandKernelCallableCaptureRequest(capture frontend.ClosureCapture) islandkernel.EscapeRequest {
	name := strings.TrimSpace(capture.Name)
	if name == "" {
		name = "<anonymous-capture>"
	}
	return islandkernel.EscapeRequest{
		Ref: islandkernel.MemoryRef{
			BaseID:      name,
			IslandID:    "callable-capture:" + name,
			Epoch:       1,
			OwnerID:     "callable",
			Provenance:  islandkernel.ProvenanceOwned,
			UnsafeClass: islandkernel.UnsafeSafe,
		},
	}
}

func surfaceEphemeralCallableCapture(
	captures []frontend.ClosureCapture,
	types map[string]*TypeInfo,
) (frontend.ClosureCapture, string, bool) {
	for _, capture := range captures {
		if surfaceType, ok := surfaceEphemeralValueType(capture.Type.Name, types); ok {
			return capture, surfaceType, true
		}
	}
	return frontend.ClosureCapture{}, "", false
}

// ---- closure_captures.go ----

func collectDeferCaptures(
	stmts []frontend.Stmt,
	locals map[string]LocalInfo,
) map[string]frontend.Position {
	captures := make(map[string]frontend.Position)
	collectStmtCaptures(stmts, locals, map[string]bool{}, captures, captureOwnershipPaths)
	return captures
}

func collectClosureCaptures(
	fn *frontend.FuncDecl,
	locals map[string]LocalInfo,
) map[string]frontend.Position {
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
	return lifetimeDiagnosticf(
		pos,
		("function-typed local '%s' captures '%s'; captures are not " +
			"supported for function-typed values in this MVP (use a let-bound ptr " +
			"closure and call it directly, or pass a non-capturing named function/" +
			"closure symbol); closure lifetime/ABI evidence is only available for " +
			"local direct calls"),
		localName,
		captured,
	)
}

func unsupportedFunctionTypedCaptureAliasError(
	pos frontend.Position,
	localName, capturedLocal string,
) error {
	return lifetimeDiagnosticf(
		pos,
		("function-typed local '%s' aliases capturing closure '%s'; " +
			"closure lifetime/ABI evidence is only available for local direct calls"),
		localName,
		capturedLocal,
	)
}

func unsupportedFunctionTypedStorageCaptureError(
	pos frontend.Position,
	targetName string,
	envSlots int,
) error {
	if envSlots > FnPtrEnvSlotCount {
		return lifetimeDiagnosticf(
			pos,
			("function-typed storage '%s' captures %d environment slots; " +
				"function-typed storage supports at most %d fnptr environment slots " +
				"within the supported fnptr ABI"),
			targetName,
			envSlots,
			FnPtrEnvSlotCount,
		)
	}
	return lifetimeDiagnosticf(
		pos,
		"function-typed storage '%s' has unsupported captured environment size",
		targetName,
	)
}

func unsupportedFunctionTypedReturnCaptureError(
	pos frontend.Position,
	valueName string,
	envSlots int,
) error {
	if envSlots > FnPtrEnvSlotCount {
		return lifetimeDiagnosticf(
			pos,
			("function-typed return '%s' captures %d environment slots; " +
				"function-typed returns support at most %d fnptr environment slots " +
				"within the supported fnptr ABI"),
			valueName,
			envSlots,
			FnPtrEnvSlotCount,
		)
	}
	return lifetimeDiagnosticf(
		pos,
		"function-typed return '%s' has unsupported captured environment size",
		valueName,
	)
}

func functionCaptureSlotCount(
	captures []frontend.ClosureCapture,
	types map[string]*TypeInfo,
) (int, error) {
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
		return fmt.Errorf(
			"%s: internal error: closure function '%s' is missing from signature table",
			frontend.FormatPos(closure.At),
			fullName,
		)
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
			return fmt.Errorf(
				"%s: internal error: closure capture '%s' is missing from locals",
				frontend.FormatPos(pos),
				name,
			)
		}
		if info.Mutable && !allowMutableValueCaptures {
			return lifetimeDiagnosticf(
				pos,
				"closure capture '%s' is mutable; %s",
				name,
				mutableClosureCaptureUnsupportedText(),
			)
		}
		if info.SurfaceFramePixelsSource != "" && len(captureBoundaryPhrase) > 0 &&
			captureBoundaryPhrase[0] != "" {
			return lifetimeDiagnosticf(
				pos,
				("surface frame pixels cannot escape via function capture; keep " +
					"Frame.pixels local to the active Surface frame"),
			)
		}
		typeRef := frontend.TypeRef{At: pos, Kind: frontend.TypeRefNamed, Name: info.TypeName}
		if !isClosureCaptureType(info.TypeName, types) {
			unsupported = append(
				unsupported,
				unsupportedCapture{pos: pos, name: name, typeName: info.TypeName},
			)
		}
		captures = append(
			captures,
			frontend.ClosureCapture{At: pos, Name: name, Type: typeRef, Mutable: info.Mutable},
		)
		closure.Decl.Params = append(
			closure.Decl.Params,
			frontend.ParamDecl{At: pos, Name: name, Type: typeRef},
		)
		sig.ParamNames = append(sig.ParamNames, name)
		sig.ParamTypes = append(sig.ParamTypes, info.TypeName)
		sig.ParamOwnership = append(sig.ParamOwnership, "")
		captureParamSlots += info.SlotCount
	}
	if len(unsupported) > 0 &&
		(len(captureBoundaryPhrase) == 0 || captureParamSlots <= FnPtrEnvSlotCount) {
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
		return lifetimeDiagnosticf(
			capture.pos,
			"closure capture '%s' has unsupported type '%s'; %s",
			capture.name,
			capture.typeName,
			closureCaptureSupportedSubsetText(),
		)
	}
	sig.ParamSlots += captureParamSlots
	funcs[fullName] = sig
	closure.Captures = captures
	return nil
}

func closureCaptureSupportedSubsetText() string {
	return ("only immutable local Int/Bool/String, simple struct, enum, and " +
		"optional captures without ptr/resource fields are supported by the " +
		"direct ptr-closure capture ABI")
}

func functionTypedCaptureSupportedSubsetText() string {
	return ("only immutable local Int/Bool/String, simple struct, enum, and " +
		"optional captures without ptr/resource fields are supported within the " +
		"supported fnptr ABI")
}

func mutableClosureCaptureUnsupportedText() string {
	return ("direct ptr closure calls would observe mutable locals by " +
		"reference, so use a function-typed fnptr binding for by-value snapshot " +
		"capture")
}

func closureLiteralDirectCallCaptureText() string {
	return ("only let-bound local direct calls can capture immutable Int/" +
		"Bool/String values and simple structs without ptr/resource fields under " +
		"the direct ptr-closure ABI")
}

func isClosureCaptureType(typeName string, types map[string]*TypeInfo) bool {
	return isClosureCaptureTypeVisiting(typeName, types, map[string]bool{})
}

func isClosureCaptureTypeVisiting(
	typeName string,
	types map[string]*TypeInfo,
	visiting map[string]bool,
) bool {
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
		return lifetimeDiagnosticf(
			call.At,
			"explicit type arguments are not supported for captured closure '%s'",
			call.Name,
		)
	}
	labeledCall := len(call.ArgLabels) > 0
	if labeledCall && len(call.ArgLabels) != len(call.Args) {
		return fmt.Errorf(
			"%s: internal error: call argument labels are inconsistent",
			frontend.FormatPos(call.At),
		)
	}
	if labeledCall {
		for i, label := range call.ArgLabels {
			if label == "" {
				return fmt.Errorf(
					"%s: cannot mix labeled and unlabeled arguments in captured closure '%s'",
					frontend.FormatPos(call.Args[i].Pos()),
					call.Name,
				)
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

func collectStmtCaptures(
	stmts []frontend.Stmt,
	locals map[string]LocalInfo,
	bound map[string]bool,
	captures map[string]frontend.Position,
	mode captureMode,
) {
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

func collectExprCaptures(
	expr frontend.Expr,
	locals map[string]LocalInfo,
	bound map[string]bool,
	captures map[string]frontend.Position,
	mode captureMode,
) {
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

func localOwnershipCapturePath(
	expr frontend.Expr,
	locals map[string]LocalInfo,
	bound map[string]bool,
) (string, frontend.Position, bool) {
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

// ---- effects.go ----

type effectContext struct {
	funcName         string
	declared         map[string]struct{}
	explicitDeclared map[string]struct{}
	capsulePerms     map[string]struct{}
	allowMissing     bool
	hasCapGroup      bool
}

type normalizedEffects struct {
	declared    map[string]struct{}
	explicit    map[string]struct{}
	hasCapGroup bool
}

func canonicalizeEffectName(name string) (string, bool) {
	return semanticspolicy.CanonicalizeEffectName(name)
}

func normalizeEffects(raw []string, pos frontend.Position) ([]string, error) {
	return semanticspolicy.NormalizeEffects(raw, pos, effectDiagnosticf)
}

func normalizeEffectDecl(raw []string, pos frontend.Position) (normalizedEffects, error) {
	normalized, err := semanticspolicy.NormalizeEffectDecl(raw, pos, effectDiagnosticf)
	if err != nil {
		return normalizedEffects{}, err
	}
	return normalizedEffects{
		declared:    normalized.Declared,
		explicit:    normalized.Explicit,
		hasCapGroup: normalized.HasCapGroup,
	}, nil
}

func sortedEffectSet(set map[string]struct{}) []string {
	return semanticspolicy.SortedEffectSet(set)
}

func effectSet(effects []string) map[string]struct{} {
	return semanticspolicy.EffectSet(effects)
}

func newEffectContext(
	funcName string,
	effects []string,
	raw []string,
	allowMissing bool,
) *effectContext {
	explicitDeclared := make(map[string]struct{}, len(effects))
	hasCapGroup := false
	if normalized, err := normalizeEffectDecl(raw, frontend.Position{}); err == nil {
		explicitDeclared = normalized.explicit
		hasCapGroup = normalized.hasCapGroup
	} else {
		for _, effect := range effects {
			explicitDeclared[effect] = struct{}{}
		}
	}
	return &effectContext{
		funcName:         funcName,
		declared:         effectSet(effects),
		explicitDeclared: explicitDeclared,
		allowMissing:     allowMissing,
		hasCapGroup:      hasCapGroup,
	}
}

func (ctx *effectContext) require(pos frontend.Position, effect string) error {
	if ctx == nil || ctx.allowMissing {
		return nil
	}
	if _, ok := ctx.declared[effect]; ok {
		return nil
	}
	return effectDiagnosticf(
		pos,
		"function '%s' uses effect '%s' but does not declare it",
		ctx.funcName,
		effect,
	)
}

func (ctx *effectContext) requireAll(pos frontend.Position, effects []string) error {
	for _, effect := range effects {
		if err := ctx.require(pos, effect); err != nil {
			return err
		}
	}
	return nil
}

func (ctx *effectContext) requireCapsulePermission(
	pos frontend.Position,
	permission string,
	attenuatedEffect string,
) error {
	if ctx == nil || ctx.allowMissing {
		return nil
	}
	if !ctx.hasCapGroup {
		return nil
	}
	if _, ok := ctx.explicitDeclared[attenuatedEffect]; ok {
		return nil
	}
	if _, ok := ctx.capsulePerms[permission]; ok {
		return nil
	}
	return effectDiagnosticf(
		pos,
		"function '%s' requires capsule permission '%s' for attenuated effect '%s'",
		ctx.funcName,
		permission,
		attenuatedEffect,
	)
}

// ---- memory_boundary_handoff.go ----

type MemoryBoundaryHandoffID string

const (
	MemoryBoundaryActorBorrowRejected       MemoryBoundaryHandoffID = "actor_borrow_rejected"
	MemoryBoundaryTaskBorrowRejected        MemoryBoundaryHandoffID = "task_borrow_rejected"
	MemoryBoundaryRequestRegionScoped       MemoryBoundaryHandoffID = "request_region_scoped"
	MemoryBoundaryUnsafeSafeMessageRejected MemoryBoundaryHandoffID = "unsafe_safe_message_rejected"
	MemoryBoundaryStaleEpochRejected        MemoryBoundaryHandoffID = "stale_epoch_rejected"
	MemoryBoundaryIslandMoveLinear          MemoryBoundaryHandoffID = "island_move_linear"
	MemoryBoundaryActorRuntimeNonClaim      MemoryBoundaryHandoffID = "actor_runtime_nonclaim"
)

type MemoryBoundaryHandoffStatus string

const (
	MemoryBoundaryImplementedNarrow MemoryBoundaryHandoffStatus = "implemented_narrow"
)

type MemoryBoundaryHandoffReport struct {
	SchemaVersion           string                     `json:"schema_version"`
	Rows                    []MemoryBoundaryHandoffRow `json:"rows"`
	NonClaims               []string                   `json:"non_claims"`
	FullActorRuntimeClaimed bool                       `json:"full_actor_runtime_claimed"`
}

type MemoryBoundaryHandoffRow struct {
	ID            MemoryBoundaryHandoffID     `json:"id"`
	Name          string                      `json:"name"`
	Status        MemoryBoundaryHandoffStatus `json:"status"`
	RequiredFacts []string                    `json:"required_facts"`
	Evidence      string                      `json:"evidence"`
	Boundary      string                      `json:"boundary"`
}

func MemoryBoundaryHandoffAudit() MemoryBoundaryHandoffReport {
	return MemoryBoundaryHandoffReport{
		SchemaVersion: "tetra.memory.boundary_handoff.v1",
		Rows: []MemoryBoundaryHandoffRow{
			actorBorrowRejectedRow(),
			taskBorrowRejectedRow(),
			requestRegionScopedRow(),
			unsafeSafeMessageRejectedRow(),
			staleEpochRejectedRow(),
			islandMoveLinearRow(),
			actorRuntimeNonClaimRow(),
		},
		NonClaims: []string{
			"full production actor runtime is not claimed",
			("request/task region reset evidence is local runtime scope " +
				"evidence, not global escape analysis completeness"),
			("unsafe send contracts remain checker-model evidence; safe actor/" +
				"task messages reject raw unsafe payloads"),
			"no distributed ownership protocol or benchmark superiority is claimed",
		},
		FullActorRuntimeClaimed: false,
	}
}

func ValidateMemoryBoundaryHandoffAudit(report MemoryBoundaryHandoffReport) error {
	if report.SchemaVersion != "tetra.memory.boundary_handoff.v1" {
		return fmt.Errorf("memory boundary handoff audit: schema = %q", report.SchemaVersion)
	}
	if report.FullActorRuntimeClaimed {
		return fmt.Errorf(
			"memory boundary handoff audit: full production actor runtime claim is forbidden for P10",
		)
	}
	if !containsMemoryBoundaryText(
		report.NonClaims,
		"full production actor runtime is not claimed",
	) {
		return fmt.Errorf(
			"memory boundary handoff audit: missing full production actor runtime nonclaim",
		)
	}

	expected := map[MemoryBoundaryHandoffID]bool{
		MemoryBoundaryActorBorrowRejected:       false,
		MemoryBoundaryTaskBorrowRejected:        false,
		MemoryBoundaryRequestRegionScoped:       false,
		MemoryBoundaryUnsafeSafeMessageRejected: false,
		MemoryBoundaryStaleEpochRejected:        false,
		MemoryBoundaryIslandMoveLinear:          false,
		MemoryBoundaryActorRuntimeNonClaim:      false,
	}
	rows := map[MemoryBoundaryHandoffID]MemoryBoundaryHandoffRow{}
	for _, row := range report.Rows {
		if row.ID == "" {
			return fmt.Errorf("memory boundary handoff audit: row missing id")
		}
		if _, ok := expected[row.ID]; !ok {
			return fmt.Errorf("memory boundary handoff audit: unexpected row %q", row.ID)
		}
		if expected[row.ID] {
			return fmt.Errorf("memory boundary handoff audit: duplicate row %q", row.ID)
		}
		expected[row.ID] = true
		rows[row.ID] = row
		if row.Status != MemoryBoundaryImplementedNarrow {
			return fmt.Errorf(
				"memory boundary handoff audit: row %q status = %q",
				row.ID,
				row.Status,
			)
		}
		if strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Evidence) == "" ||
			strings.TrimSpace(row.Boundary) == "" {
			return fmt.Errorf(
				"memory boundary handoff audit: row %q missing evidence or boundary",
				row.ID,
			)
		}
		if len(row.RequiredFacts) == 0 {
			return fmt.Errorf(
				"memory boundary handoff audit: row %q missing required facts",
				row.ID,
			)
		}
	}
	for id, seen := range expected {
		if !seen {
			return fmt.Errorf("memory boundary handoff audit: missing row %q", id)
		}
	}
	if len(report.Rows) != len(expected) {
		return fmt.Errorf(
			"memory boundary handoff audit: row count = %d, want %d",
			len(report.Rows),
			len(expected),
		)
	}

	if err := requireMemoryBoundaryFacts(
		rows[MemoryBoundaryActorBorrowRejected],
		"cannot send borrowed view across actor boundary",
		".copy()",
	); err != nil {
		return err
	}
	if err := requireMemoryBoundaryFacts(
		rows[MemoryBoundaryTaskBorrowRejected],
		"typed task error payload must be sendable across task boundary",
	); err != nil {
		return err
	}
	if err := requireMemoryBoundaryFacts(
		rows[MemoryBoundaryRequestRegionScoped],
		"RequestRegionScope",
		"TaskRegionScope",
		"reset",
	); err != nil {
		return err
	}
	if err := requireMemoryBoundaryFacts(
		rows[MemoryBoundaryUnsafeSafeMessageRejected],
		"ptr",
		"cap.mem",
		"typed actor message payload must be value-only",
	); err != nil {
		return err
	}
	if err := requireMemoryBoundaryFacts(
		rows[MemoryBoundaryStaleEpochRejected],
		"core.island_reset",
		"cannot use consumed value",
	); err != nil {
		return err
	}
	if err := requireMemoryBoundaryFacts(
		rows[MemoryBoundaryIslandMoveLinear],
		"core.send_typed",
		"cannot use consumed value",
		"island",
	); err != nil {
		return err
	}
	if err := requireMemoryBoundaryFacts(
		rows[MemoryBoundaryActorRuntimeNonClaim],
		"not a production actor runtime",
	); err != nil {
		return err
	}
	return nil
}

func actorBorrowRejectedRow() MemoryBoundaryHandoffRow {
	return MemoryBoundaryHandoffRow{
		ID:     MemoryBoundaryActorBorrowRejected,
		Name:   "Actor borrowed payload rejection",
		Status: MemoryBoundaryImplementedNarrow,
		RequiredFacts: []string{
			"cannot send borrowed view across actor boundary",
			".copy() is required for borrowed slice/String actor payloads",
			"borrowed aggregate actor payloads reject unless explicitly copied",
		},
		Evidence: ("compiler/tests/semantics/semantics_async_ownership_" +
			"test.go::TestBorrowedActorSendRejectedUnlessCopied; compiler/tests/" +
			"semantics/semantics_memory_surface_" +
			"test.go::TestMemoryIdealV4ActorBoundaryCopyAndBorrowDiagnostics; " +
			"compiler/internal/semantics/semantics_" +
			"expressions.go::validateActorBoundaryPayloadExpr"),
		Boundary: ("source semantics reject borrowed actor message payloads or " +
			"require explicit copy; this is checker evidence, not production actor " +
			"runtime evidence"),
	}
}

func taskBorrowRejectedRow() MemoryBoundaryHandoffRow {
	return MemoryBoundaryHandoffRow{
		ID:     MemoryBoundaryTaskBorrowRejected,
		Name:   "Task borrowed payload rejection",
		Status: MemoryBoundaryImplementedNarrow,
		RequiredFacts: []string{
			"typed task error payload must be sendable across task boundary",
			"borrowed slice/String task error payloads reject",
			"copy before typed task boundary is accepted",
		},
		Evidence: ("compiler/tests/semantics/semantics_async_ownership_" +
			"test.go::TestBorrowedTaskBoundaryTypedErrorPayloadRejected; compiler/" +
			"tests/semantics/semantics_memory_surface_" +
			"test.go::TestMemoryIdealV4TaskBoundaryCurrentSurfaceDiagnostics"),
		Boundary: ("current typed task surface has no arbitrary payload spawn API; " +
			"evidence covers typed task error payloads and rejects reference-shaped " +
			"task boundary payloads"),
	}
}

func requestRegionScopedRow() MemoryBoundaryHandoffRow {
	return MemoryBoundaryHandoffRow{
		ID:     MemoryBoundaryRequestRegionScoped,
		Name:   "Request/task region scope reset",
		Status: MemoryBoundaryImplementedNarrow,
		RequiredFacts: []string{
			"RequestRegionScope injects and resets request region storage",
			"TaskRegionScope injects and resets task region storage",
			"reset prevents request/task region data from becoming a safe cross-boundary message by default",
		},
		Evidence: ("docs/audits/memory/ram-raw/request-task-region-v1.md; compiler/" +
			"internal/httprt/request_region.go::RequestRegionScope; compiler/" +
			"internal/parallelrt/task_region.go::TaskRegionScope; compiler/internal/" +
			"httprt/request_view_" +
			"test.go::TestRequestRegionScopeInjectsRegionForHTTPJSONAndResetsAfterWri" +
			"te; compiler/internal/parallelrt/scheduler_model_" +
			"test.go::TestTaskRegionScopeInjectsRegionAndResetsAfterTask"),
		Boundary: ("request/task region evidence is scoped runtime entry behavior " +
			"and reset reporting; it is not a claim that arbitrary region-backed " +
			"data may cross actor/task/request boundaries safely"),
	}
}

func unsafeSafeMessageRejectedRow() MemoryBoundaryHandoffRow {
	return MemoryBoundaryHandoffRow{
		ID:     MemoryBoundaryUnsafeSafeMessageRejected,
		Name:   "Unsafe payload cannot become safe message",
		Status: MemoryBoundaryImplementedNarrow,
		RequiredFacts: []string{
			"ptr typed actor payload rejects with typed actor message payload must be value-only",
			"cap.mem typed actor payload rejects with typed actor message payload must be value-only",
			"unsafe send contracts are not safe typed actor message permission",
		},
		Evidence: ("compiler/tests/safety/plan250_safety_runtime_" +
			"test.go::TestPlan250SafetySendabilityAcrossModuleBoundaries; compiler/" +
			"internal/semantics/semantics_" +
			"expressions.go::validateTypedActorMessageType; compiler/internal/" +
			"actorsafety/sendability_" +
			"test.go::TestUnsafePointerRequiresExplicitUnsafeSendContract"),
		Boundary: ("raw unsafe payloads stay rejected for safe typed actor " +
			"messages; internal unsafe-send contract model evidence does not expose " +
			"a safe message surface"),
	}
}

func staleEpochRejectedRow() MemoryBoundaryHandoffRow {
	return MemoryBoundaryHandoffRow{
		ID:     MemoryBoundaryStaleEpochRejected,
		Name:   "Stale epoch after reset rejection",
		Status: MemoryBoundaryImplementedNarrow,
		RequiredFacts: []string{
			"core.island_reset consumes the prior island handle",
			"cannot use consumed value after reset",
			"stale island handle cannot be sent across actor boundary after reset",
		},
		Evidence: ("compiler/tests/runtime/resource_finalization_" +
			"test.go::TestMemoryBoundaryHandoffRejectsStaleIslandAfterResetAcrossActo" +
			"rBoundary; compiler/tests/runtime/resource_finalization_" +
			"test.go::TestIslandResetRejectsUseAfterReset; compiler/internal/" +
			"semantics/semantics_expressions.go::consumeTypedActorTransferPayloads"),
		Boundary: ("stale epoch rejection is enforced as consumed-resource " +
			"semantics before actor send; no live stale-epoch runtime sanitizer " +
			"bypass is claimed"),
	}
}

func islandMoveLinearRow() MemoryBoundaryHandoffRow {
	return MemoryBoundaryHandoffRow{
		ID:     MemoryBoundaryIslandMoveLinear,
		Name:   "Island move remains linear across actor boundary",
		Status: MemoryBoundaryImplementedNarrow,
		RequiredFacts: []string{
			"core.send_typed consumes island payloads",
			"cannot use consumed value after typed actor transfer",
			"island handle moved twice across actor boundary is rejected",
		},
		Evidence: ("compiler/compiler_suite_" +
			"test.go::TestActorsTypedMessagesIslandTransferConsumesSource; compiler/" +
			"tests/runtime/resource_finalization_" +
			"test.go::TestTypedActorTransferRejectsFieldAccessEnumPayloadAliasReuse; " +
			"compiler/internal/semantics/semantics_" +
			"expressions.go::consumeTypedActorTransferPayloads"),
		Boundary: ("linear transfer evidence covers current typed actor message " +
			"payloads and source diagnostics; it is not a distributed ownership " +
			"protocol or race-safety proof"),
	}
}

func actorRuntimeNonClaimRow() MemoryBoundaryHandoffRow {
	return MemoryBoundaryHandoffRow{
		ID:     MemoryBoundaryActorRuntimeNonClaim,
		Name:   "Actor runtime production nonclaim",
		Status: MemoryBoundaryImplementedNarrow,
		RequiredFacts: []string{
			"not a production actor runtime",
			"full production actor runtime is not claimed",
			"actor/task/request boundary handoff does not start actor runtime implementation",
		},
		Evidence: ("compiler/internal/actorsrt/actorsrt_" +
			"core.go::ActorRuntimeProductionBoundaryAudit; docs/audits/runtime/" +
			"actors/actor-runtime-production-boundary-v1.md"),
		Boundary: ("P10 proves Memory/Islands boundary handoff only; actor " +
			"production runtime remains a later plan with separate gates"),
	}
}

func requireMemoryBoundaryFacts(row MemoryBoundaryHandoffRow, wants ...string) error {
	for _, want := range wants {
		if !containsMemoryBoundaryText(row.RequiredFacts, want) {
			return fmt.Errorf("memory boundary handoff audit: row %q missing fact %q", row.ID, want)
		}
	}
	return nil
}

func containsMemoryBoundaryText(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}

// ---- region.go ----

const (
	regionNone                = semanticsregions.None
	regionUnknown             = semanticsregions.Unknown
	regionParamStart          = semanticsregions.ParamStart
	regionExplicitBorrowStart = semanticsregions.ExplicitBorrowStart
)

type branchScopeInfo struct {
	thenID int
	elseID int
}

type resourceFinalization struct {
	state          string
	pos            frontend.Position
	maybe          bool
	mayBeAvailable bool
	states         map[string]frontend.Position
}

type ownershipJoinConflict struct {
	leftLabel     string
	leftConsumed  bool
	leftPos       frontend.Position
	rightLabel    string
	rightConsumed bool
	rightPos      frontend.Position
}

type scopeInfo struct {
	localScopes     map[string]int
	localScopeSets  map[string]map[int]struct{}
	islandScopes    map[string]int
	ifScopes        map[*frontend.IfStmt]branchScopeInfo
	ifLetScopes     map[*frontend.IfLetStmt]branchScopeInfo
	whileScopes     map[*frontend.WhileStmt]int
	forScopes       map[*frontend.ForRangeStmt]int
	matchCaseScopes map[*frontend.MatchStmt][]int
	matchExprScopes map[*frontend.MatchExpr][]int
	catchExprScopes map[*frontend.CatchExpr][]int
	unsafeScopes    map[*frontend.UnsafeStmt]int
	deferScopes     map[*frontend.DeferStmt]int
	scopeStack      []int
	nextScopeID     int
}

func newScopeInfo() *scopeInfo {
	return &scopeInfo{
		localScopes:     make(map[string]int),
		localScopeSets:  make(map[string]map[int]struct{}),
		islandScopes:    make(map[string]int),
		ifScopes:        make(map[*frontend.IfStmt]branchScopeInfo),
		ifLetScopes:     make(map[*frontend.IfLetStmt]branchScopeInfo),
		whileScopes:     make(map[*frontend.WhileStmt]int),
		forScopes:       make(map[*frontend.ForRangeStmt]int),
		matchCaseScopes: make(map[*frontend.MatchStmt][]int),
		matchExprScopes: make(map[*frontend.MatchExpr][]int),
		catchExprScopes: make(map[*frontend.CatchExpr][]int),
		unsafeScopes:    make(map[*frontend.UnsafeStmt]int),
		deferScopes:     make(map[*frontend.DeferStmt]int),
	}
}

func (s *scopeInfo) currentScopeID() int {
	if len(s.scopeStack) == 0 {
		return regionNone
	}
	return s.scopeStack[len(s.scopeStack)-1]
}

func (s *scopeInfo) enterScope() int {
	id := s.nextScopeID
	s.nextScopeID++
	s.scopeStack = append(s.scopeStack, id)
	return id
}

func (s *scopeInfo) exitScope() {
	if len(s.scopeStack) == 0 {
		return
	}
	s.scopeStack = s.scopeStack[:len(s.scopeStack)-1]
}

type regionState struct {
	localScopes            map[string]int
	localScopeSets         map[string]map[int]struct{}
	islandScopes           map[string]int
	ifScopes               map[*frontend.IfStmt]branchScopeInfo
	ifLetScopes            map[*frontend.IfLetStmt]branchScopeInfo
	whileScopes            map[*frontend.WhileStmt]int
	forScopes              map[*frontend.ForRangeStmt]int
	matchCaseScopes        map[*frontend.MatchStmt][]int
	matchExprScopes        map[*frontend.MatchExpr][]int
	catchExprScopes        map[*frontend.CatchExpr][]int
	unsafeScopes           map[*frontend.UnsafeStmt]int
	deferScopes            map[*frontend.DeferStmt]int
	islandNameByID         map[int]string
	regionVars             map[string]int
	exprRegionTrees        map[frontend.Expr]map[string]int
	paramRegionIndex       map[int]int
	resourceParamIndex     map[int]int
	resourceParamPath      map[int]string
	borrowedParamRegion    map[int]string
	awaitInvalidatedBorrow map[int]frontend.Position
	nextExplicitBorrow     int
	paramNames             []string
	unknownVars            map[string]bool
	unknownConflicts       map[string]regionConflict
	reachable              bool
	consumedVars           map[string]frontend.Position
	maybeConsumedVars      map[string]ownershipJoinConflict
	ownershipAliases       map[string]string
	borrowedPtrAliases     map[string]string
	ownedRegionSliceOwners map[string]string
	consumedResources      map[int]frontend.Position
	resourceVars           map[string]int
	unknownResources       map[int]bool
	finalizedResources     map[int]resourceFinalization
	nextResourceID         int
	deferCaptureFrames     []map[string]frontend.Position
	activeScopes           []int
	activeIndex            map[int]int
	unsafeDepth            int
	loopDepth              int
	loopFlowFrames         []loopFlowFrame
	throwType              string
	allowThrowDepth        int
	allowThrowCall         *frontend.CallExpr
	allowCatchDepth        int
	allowCatchCall         *frontend.CallExpr
	async                  bool
	allowAwaitDepth        int
	allowAwaitCall         *frontend.CallExpr
	returnRegion           int
	returnRegionSet        bool
	returnRegionSummary    ReturnRegionSummary
	returnResourceParam    int
	returnResourcePath     string
	returnResourceSummary  ReturnResourceSummary
	returnResourceSet      bool
	returnResourceUnknown  bool
	throwResourceSummary   ReturnResourceSummary
	actorStateFields       map[string]ActorStateField
}

func newRegionState(scopes *scopeInfo) *regionState {
	localScopes := make(map[string]int)
	localScopeSets := make(map[string]map[int]struct{})
	islandScopes := make(map[string]int)
	var ifScopes map[*frontend.IfStmt]branchScopeInfo
	var ifLetScopes map[*frontend.IfLetStmt]branchScopeInfo
	var whileScopes map[*frontend.WhileStmt]int
	var forScopes map[*frontend.ForRangeStmt]int
	var matchCaseScopes map[*frontend.MatchStmt][]int
	var matchExprScopes map[*frontend.MatchExpr][]int
	var catchExprScopes map[*frontend.CatchExpr][]int
	var unsafeScopes map[*frontend.UnsafeStmt]int
	var deferScopes map[*frontend.DeferStmt]int
	if scopes != nil {
		localScopes = scopes.localScopes
		localScopeSets = scopes.localScopeSets
		islandScopes = scopes.islandScopes
		ifScopes = scopes.ifScopes
		ifLetScopes = scopes.ifLetScopes
		whileScopes = scopes.whileScopes
		forScopes = scopes.forScopes
		matchCaseScopes = scopes.matchCaseScopes
		matchExprScopes = scopes.matchExprScopes
		catchExprScopes = scopes.catchExprScopes
		unsafeScopes = scopes.unsafeScopes
		deferScopes = scopes.deferScopes
	}
	islandNameByID := make(map[int]string, len(islandScopes))
	for name, id := range islandScopes {
		islandNameByID[id] = name
	}
	return &regionState{
		localScopes:            localScopes,
		localScopeSets:         localScopeSets,
		islandScopes:           islandScopes,
		ifScopes:               ifScopes,
		ifLetScopes:            ifLetScopes,
		whileScopes:            whileScopes,
		forScopes:              forScopes,
		matchCaseScopes:        matchCaseScopes,
		matchExprScopes:        matchExprScopes,
		catchExprScopes:        catchExprScopes,
		unsafeScopes:           unsafeScopes,
		deferScopes:            deferScopes,
		islandNameByID:         islandNameByID,
		regionVars:             make(map[string]int),
		exprRegionTrees:        make(map[frontend.Expr]map[string]int),
		paramRegionIndex:       make(map[int]int),
		resourceParamIndex:     make(map[int]int),
		resourceParamPath:      make(map[int]string),
		borrowedParamRegion:    make(map[int]string),
		awaitInvalidatedBorrow: make(map[int]frontend.Position),
		nextExplicitBorrow:     regionExplicitBorrowStart,
		unknownConflicts:       make(map[string]regionConflict),
		unknownVars:            make(map[string]bool),
		reachable:              true,
		consumedVars:           make(map[string]frontend.Position),
		maybeConsumedVars:      make(map[string]ownershipJoinConflict),
		ownershipAliases:       make(map[string]string),
		borrowedPtrAliases:     make(map[string]string),
		ownedRegionSliceOwners: make(map[string]string),
		consumedResources:      make(map[int]frontend.Position),
		resourceVars:           make(map[string]int),
		unknownResources:       make(map[int]bool),
		finalizedResources:     make(map[int]resourceFinalization),
		nextResourceID:         1,
		activeIndex:            make(map[int]int),
	}
}

func (s *regionState) markConsumed(name string, pos frontend.Position) {
	if s == nil || name == "" {
		return
	}
	s.markConsumedDirect(name, pos)
	if source, ok := s.ownershipAliasSource(name); ok {
		s.markConsumedDirect(source, pos)
	}
}

func (s *regionState) markConsumedDirect(name string, pos frontend.Position) {
	if s == nil || name == "" {
		return
	}
	if id, ok := s.resourceID(name); ok {
		s.consumedResources[id] = pos
		return
	}
	delete(s.maybeConsumedVars, name)
	s.consumedVars[name] = pos
}

func (s *regionState) clearConsumed(name string) {
	if s == nil || name == "" {
		return
	}
	delete(s.consumedVars, name)
	delete(s.maybeConsumedVars, name)
	if source, ok := s.ownershipAliasSource(name); ok {
		delete(s.consumedVars, source)
		delete(s.maybeConsumedVars, source)
	}
}

func (s *regionState) clearConsumedTree(name string) {
	if s == nil || name == "" {
		return
	}
	s.clearConsumedTreeDirect(name)
}

func (s *regionState) clearConsumedTreeDirect(name string) {
	if s == nil || name == "" {
		return
	}
	queryName := name
	if source, ok := s.ownershipAliasSource(name); ok {
		queryName = source
	}
	for path := range s.consumedVars {
		target := path
		if source, ok := s.ownershipAliasSource(path); ok {
			target = source
		}
		if target == queryName || ownershipPathPrefix(queryName, target) {
			delete(s.consumedVars, path)
		}
	}
	for path := range s.maybeConsumedVars {
		target := path
		if source, ok := s.ownershipAliasSource(path); ok {
			target = source
		}
		if target == queryName || ownershipPathPrefix(queryName, target) {
			delete(s.maybeConsumedVars, path)
		}
	}
}

func (s *regionState) checkAssignableOwnershipPath(path string, pos frontend.Position) error {
	if s == nil || path == "" {
		return nil
	}
	parent := parentOwnershipPath(path)
	if parent == "" {
		return nil
	}
	return s.checkNotConsumed(parent, pos)
}

func (s *regionState) bindOwnershipAlias(name string, source string) {
	if s == nil || name == "" {
		return
	}
	if source == "" || source == name {
		delete(s.ownershipAliases, name)
		return
	}
	s.ownershipAliases[name] = source
}

func (s *regionState) bindBorrowedPtrAlias(name string, owner string) {
	if s == nil || name == "" {
		return
	}
	if owner == "" || owner == name {
		s.clearBorrowedPtrAliasTree(name)
		return
	}
	s.borrowedPtrAliases[name] = owner
}

func (s *regionState) clearBorrowedPtrAliasTree(name string) {
	if s == nil || name == "" {
		return
	}
	for path := range s.borrowedPtrAliases {
		if path == name || ownershipPathPrefix(name, path) {
			delete(s.borrowedPtrAliases, path)
		}
	}
}

func (s *regionState) bindOwnedRegionSliceOwner(name string, owner string) {
	if s == nil || name == "" {
		return
	}
	if owner == "" || owner == name {
		s.clearOwnedRegionSliceOwnerTree(name)
		return
	}
	s.ownedRegionSliceOwners[name] = owner
}

func (s *regionState) clearOwnedRegionSliceOwnerTree(name string) {
	if s == nil || name == "" {
		return
	}
	for path := range s.ownedRegionSliceOwners {
		if path == name || ownershipPathPrefix(name, path) {
			delete(s.ownedRegionSliceOwners, path)
		}
	}
}

func (s *regionState) ownedRegionSliceOwner(path string) (string, bool) {
	if s == nil || path == "" {
		return "", false
	}
	for probe := path; probe != ""; probe = ownershipPathParent(probe) {
		owner, ok := s.ownedRegionSliceOwners[probe]
		if !ok || owner == "" {
			continue
		}
		if probe == path {
			return owner, true
		}
		return owner + path[len(probe):], true
	}
	return "", false
}

func (s *regionState) markOwnedRegionSlicesConsumedByOwner(owner string, pos frontend.Position) {
	if s == nil || owner == "" {
		return
	}
	for path, pathOwner := range s.ownedRegionSliceOwners {
		if s.resourcePathsAlias(owner, pathOwner) {
			s.markConsumedDirect(path, pos)
		}
	}
}

func (s *regionState) liveOwnedRegionSliceForOwner(owner string) (string, bool) {
	if s == nil || owner == "" {
		return "", false
	}
	paths := make([]string, 0, len(s.ownedRegionSliceOwners))
	for path := range s.ownedRegionSliceOwners {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		pathOwner := s.ownedRegionSliceOwners[path]
		if pathOwner == "" || !s.resourcePathsAlias(owner, pathOwner) {
			continue
		}
		if _, _, _, _, consumed := s.consumedPath(path); consumed {
			continue
		}
		return path, true
	}
	return "", false
}

func (s *regionState) resourcePathsAlias(left string, right string) bool {
	if s == nil || left == "" || right == "" {
		return false
	}
	left = s.canonicalResourcePath(left)
	right = s.canonicalResourcePath(right)
	if left == right {
		return true
	}
	leftID, leftOK := s.resourceID(left)
	rightID, rightOK := s.resourceID(right)
	return leftOK && rightOK && leftID == rightID
}

func (s *regionState) canonicalResourcePath(path string) string {
	if source, ok := s.ownershipAliasSource(path); ok && source != "" {
		return source
	}
	return path
}

func (s *regionState) borrowedPtrAliasOwner(name string) (string, bool) {
	if s == nil || name == "" {
		return "", false
	}
	owner, ok := s.borrowedPtrAliases[name]
	return owner, ok && owner != ""
}

func (s *regionState) borrowedPtrAliasOwnerInTree(name string) (string, bool) {
	if s == nil || name == "" {
		return "", false
	}
	if owner, ok := s.borrowedPtrAliasOwner(name); ok {
		return owner, true
	}
	paths := make([]string, 0, len(s.borrowedPtrAliases))
	for path := range s.borrowedPtrAliases {
		if ownershipPathPrefix(name, path) {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	for _, path := range paths {
		if owner := s.borrowedPtrAliases[path]; owner != "" {
			return owner, true
		}
	}
	return "", false
}

func (s *regionState) checkNotConsumed(name string, pos frontend.Position) error {
	if s == nil || name == "" {
		return nil
	}
	if consumedName, consumedAt, conflict, maybe, ok := s.consumedPath(name); ok {
		reportName := ownershipDiagnosticPath(name, consumedName)
		if maybe {
			return ownershipDiagnosticf(
				pos,
				("cannot use consumed value '%s': value '%s' may have been " +
					"consumed after ownership join (%s: %s, %s: %s)"),
				reportName,
				reportName,
				conflict.leftLabel,
				formatOwnershipJoinState(conflict.leftConsumed, conflict.leftPos),
				conflict.rightLabel,
				formatOwnershipJoinState(conflict.rightConsumed, conflict.rightPos),
			)
		}
		return ownershipDiagnosticf(
			pos,
			"cannot use consumed value '%s' (consumed at %s)",
			reportName,
			frontend.FormatPos(consumedAt),
		)
	}
	if source, ok := s.ownershipAliasSource(name); ok {
		if consumedName, consumedAt, conflict, maybe, ok := s.consumedPath(source); ok {
			reportName := ownershipDiagnosticPath(name, consumedName)
			if maybe {
				return ownershipDiagnosticf(
					pos,
					("cannot use consumed value '%s': value '%s' may have been " +
						"consumed after ownership join (%s: %s, %s: %s)"),
					reportName,
					reportName,
					conflict.leftLabel,
					formatOwnershipJoinState(conflict.leftConsumed, conflict.leftPos),
					conflict.rightLabel,
					formatOwnershipJoinState(conflict.rightConsumed, conflict.rightPos),
				)
			}
			return ownershipDiagnosticf(
				pos,
				"cannot use consumed value '%s' (consumed at %s)",
				reportName,
				frontend.FormatPos(consumedAt),
			)
		}
	}
	return nil
}

func (s *regionState) checkNoConsumedDescendants(name string, pos frontend.Position) error {
	if s == nil || name == "" {
		return nil
	}
	queryName := name
	if source, ok := s.ownershipAliasSource(name); ok {
		queryName = source
	}
	for consumedName, consumedAt := range s.consumedVars {
		reportName := consumedName
		if source, ok := s.ownershipAliasSource(consumedName); ok {
			reportName = source
		}
		if reportName != queryName && !ownershipPathPrefix(queryName, reportName) {
			continue
		}
		if conflict, maybe := s.maybeConsumedVars[consumedName]; maybe {
			return ownershipDiagnosticf(
				pos,
				("cannot use consumed value '%s': value '%s' may have been " +
					"consumed after ownership join (%s: %s, %s: %s)"),
				reportName,
				reportName,
				conflict.leftLabel,
				formatOwnershipJoinState(conflict.leftConsumed, conflict.leftPos),
				conflict.rightLabel,
				formatOwnershipJoinState(conflict.rightConsumed, conflict.rightPos),
			)
		}
		if conflict, maybe := s.maybeConsumedVars[reportName]; maybe {
			return ownershipDiagnosticf(
				pos,
				("cannot use consumed value '%s': value '%s' may have been " +
					"consumed after ownership join (%s: %s, %s: %s)"),
				reportName,
				reportName,
				conflict.leftLabel,
				formatOwnershipJoinState(conflict.leftConsumed, conflict.leftPos),
				conflict.rightLabel,
				formatOwnershipJoinState(conflict.rightConsumed, conflict.rightPos),
			)
		}
		return ownershipDiagnosticf(
			pos,
			"cannot use consumed value '%s' (consumed at %s)",
			reportName,
			frontend.FormatPos(consumedAt),
		)
	}
	return nil
}

func (s *regionState) checkNoConsumedProperDescendants(name string, pos frontend.Position) error {
	if s == nil || name == "" {
		return nil
	}
	queryName := name
	if source, ok := s.ownershipAliasSource(name); ok {
		queryName = source
	}
	for consumedName, consumedAt := range s.consumedVars {
		reportName := consumedName
		if source, ok := s.ownershipAliasSource(consumedName); ok {
			reportName = source
		}
		if reportName == queryName || !ownershipPathPrefix(queryName, reportName) {
			continue
		}
		if conflict, maybe := s.maybeConsumedVars[consumedName]; maybe {
			return ownershipDiagnosticf(
				pos,
				("cannot use consumed value '%s': value '%s' may have been " +
					"consumed after ownership join (%s: %s, %s: %s)"),
				reportName,
				reportName,
				conflict.leftLabel,
				formatOwnershipJoinState(conflict.leftConsumed, conflict.leftPos),
				conflict.rightLabel,
				formatOwnershipJoinState(conflict.rightConsumed, conflict.rightPos),
			)
		}
		if conflict, maybe := s.maybeConsumedVars[reportName]; maybe {
			return ownershipDiagnosticf(
				pos,
				("cannot use consumed value '%s': value '%s' may have been " +
					"consumed after ownership join (%s: %s, %s: %s)"),
				reportName,
				reportName,
				conflict.leftLabel,
				formatOwnershipJoinState(conflict.leftConsumed, conflict.leftPos),
				conflict.rightLabel,
				formatOwnershipJoinState(conflict.rightConsumed, conflict.rightPos),
			)
		}
		return ownershipDiagnosticf(
			pos,
			"cannot use consumed value '%s' (consumed at %s)",
			reportName,
			frontend.FormatPos(consumedAt),
		)
	}
	return nil
}

func (s *regionState) consumedPath(
	name string,
) (string, frontend.Position, ownershipJoinConflict, bool, bool) {
	for path := name; path != ""; path = ownershipPathParent(path) {
		if consumedAt, ok := s.consumedAt(path); ok {
			consumedName := path
			if source, alias := s.ownershipAliasSource(path); alias {
				consumedName = source
			}
			conflict, maybe := s.maybeConsumedVars[consumedName]
			if !maybe && consumedName != path {
				conflict, maybe = s.maybeConsumedVars[path]
			}
			return consumedName, consumedAt, conflict, maybe, true
		}
		if probePath, alias := s.ownershipAliasSource(path); alias {
			if consumedAt, ok := s.consumedAt(probePath); ok {
				conflict, maybe := s.maybeConsumedVars[probePath]
				return probePath, consumedAt, conflict, maybe, true
			}
		}
	}
	return "", frontend.Position{}, ownershipJoinConflict{}, false, false
}

func ownershipDiagnosticPath(queryPath string, consumedPath string) string {
	if queryPath != "" && consumedPath != "" && containsSyntheticOwnershipSegment(consumedPath) &&
		!containsSyntheticOwnershipSegment(queryPath) {
		return queryPath
	}
	return consumedPath
}

func containsSyntheticOwnershipSegment(path string) bool {
	for _, segment := range strings.Split(path, ".") {
		if strings.HasPrefix(segment, "$") {
			return true
		}
	}
	return false
}

func (s *regionState) ownershipAliasSource(path string) (string, bool) {
	if s == nil || path == "" {
		return "", false
	}
	for probe := path; probe != ""; probe = ownershipPathParent(probe) {
		source, ok := s.ownershipAliases[probe]
		if !ok || source == "" {
			continue
		}
		if probe == path {
			return source, true
		}
		return source + path[len(probe):], true
	}
	return "", false
}

func parentOwnershipPath(path string) string {
	return ownershipPathParent(path)
}

func formatOwnershipJoinState(consumed bool, pos frontend.Position) string {
	if !consumed {
		return "available"
	}
	return fmt.Sprintf("consumed at %s", frontend.FormatPos(pos))
}

func (s *regionState) markResourceFinalized(name string, state string, pos frontend.Position) {
	if s == nil || name == "" || state == "" {
		return
	}
	id := s.ensureResource(name)
	s.finalizedResources[id] = resourceFinalization{state: state, pos: pos}
}

func (s *regionState) markResourceFinalizedAliases(
	name string,
	state string,
	pos frontend.Position,
) {
	if s == nil || name == "" || state == "" {
		return
	}
	id, ok := s.resourceID(name)
	if !ok {
		id = s.ensureResource(name)
	}
	for _, aliasID := range s.resourceVars {
		if aliasID == id {
			s.finalizedResources[aliasID] = resourceFinalization{state: state, pos: pos}
		}
	}
}

func (s *regionState) clearResourceFinalized(name string) {
	if s == nil || name == "" {
		return
	}
	if id, ok := s.resourceID(name); ok {
		delete(s.finalizedResources, id)
	}
}

func (s *regionState) bindResource(name string, source string, isResource bool) {
	if s == nil || name == "" {
		return
	}
	if !isResource {
		delete(s.resourceVars, name)
		return
	}
	if source != "" {
		if id, ok := s.resourceID(source); ok {
			s.resourceVars[name] = id
			return
		}
	}
	s.resourceVars[name] = s.allocateResourceID()
}

func (s *regionState) bindTransferredResource(name string, source string) {
	if s == nil || name == "" {
		return
	}
	id := s.allocateResourceID()
	s.resourceVars[name] = id
	if sourceID, ok := s.resourceID(source); ok {
		if idx, idxOK := s.resourceParamIndex[sourceID]; idxOK {
			s.resourceParamIndex[id] = idx
		}
		if path, pathOK := s.resourceParamPath[sourceID]; pathOK {
			s.resourceParamPath[id] = path
		}
	}
}

func (s *regionState) bindUnknownResource(name string) {
	if s == nil || name == "" {
		return
	}
	id := s.allocateResourceID()
	s.resourceVars[name] = id
	s.unknownResources[id] = true
}

func (s *regionState) resourceUnknown(name string) bool {
	if s == nil || name == "" {
		return false
	}
	id, ok := s.resourceID(name)
	if !ok {
		return false
	}
	return s.unknownResources[id]
}

func (s *regionState) resourceFinalization(name string) (resourceFinalization, bool) {
	if s == nil || name == "" {
		return resourceFinalization{}, false
	}
	id, ok := s.resourceID(name)
	if !ok {
		return resourceFinalization{}, false
	}
	final, ok := s.finalizedResources[id]
	return final, ok
}

func (s *regionState) checkResourceNotFinalized(name string, pos frontend.Position) error {
	if s == nil || name == "" {
		return nil
	}
	final, ok := s.resourceFinalization(name)
	if !ok || resourceFinalizationAllows(final, "closed") {
		return nil
	}
	return s.resourceFinalizationError(name, final, pos)
}

func (s *regionState) checkResourceFinalizationAllowed(
	name string,
	pos frontend.Position,
	allowed ...string,
) error {
	if s == nil || name == "" {
		return nil
	}
	final, ok := s.resourceFinalization(name)
	if !ok {
		return nil
	}
	if resourceFinalizationAllows(final, allowed...) {
		return nil
	}
	return s.resourceFinalizationError(name, final, pos)
}

func (s *regionState) resourceFinalizationError(
	name string,
	final resourceFinalization,
	pos frontend.Position,
) error {
	if final.maybe {
		states := resourceFinalizationStates(final)
		if len(states) == 1 {
			state := states[0]
			return ownershipDiagnosticf(
				pos,
				"cannot use %s resource '%s': resource may have been %s after control-flow merge (%s)",
				state,
				name,
				state,
				formatResourceFinalizationPossibilities(final),
			)
		}
		return ownershipDiagnosticf(
			pos,
			"cannot use finalized resource '%s': ambiguous finalization state after control-flow merge (%s)",
			name,
			formatResourceFinalizationPossibilities(final),
		)
	}
	return ownershipDiagnosticf(
		pos,
		"cannot use %s resource '%s' (%s at %s)",
		final.state,
		name,
		final.state,
		frontend.FormatPos(final.pos),
	)
}

func resourceFinalizationAllows(final resourceFinalization, allowed ...string) bool {
	allowedStates := make(map[string]bool, len(allowed))
	for _, state := range allowed {
		allowedStates[state] = true
	}
	for state := range resourceFinalizationStatePositions(final) {
		if !allowedStates[state] {
			return false
		}
	}
	return true
}

func resourceFinalizationStates(final resourceFinalization) []string {
	statePositions := resourceFinalizationStatePositions(final)
	states := make([]string, 0, len(statePositions))
	for state := range statePositions {
		states = append(states, state)
	}
	sort.Strings(states)
	return states
}

func resourceFinalizationStatePositions(final resourceFinalization) map[string]frontend.Position {
	states := make(map[string]frontend.Position)
	if final.state != "" {
		states[final.state] = final.pos
	}
	for state, pos := range final.states {
		if existing, ok := states[state]; ok {
			states[state] = earliestPosition(existing, pos)
			continue
		}
		states[state] = pos
	}
	return states
}

func formatResourceFinalizationPossibilities(final resourceFinalization) string {
	parts := []string{}
	if final.mayBeAvailable {
		parts = append(parts, "available")
	}
	for _, state := range resourceFinalizationStates(final) {
		pos := resourceFinalizationStatePositions(final)[state]
		parts = append(parts, fmt.Sprintf("%s at %s", state, frontend.FormatPos(pos)))
	}
	return strings.Join(parts, ", ")
}

func (s *regionState) resourceID(name string) (int, bool) {
	if s == nil || name == "" {
		return 0, false
	}
	id, ok := s.resourceVars[name]
	return id, ok
}

func (s *regionState) ensureResource(name string) int {
	if id, ok := s.resourceID(name); ok {
		return id
	}
	id := s.allocateResourceID()
	s.resourceVars[name] = id
	return id
}

func (s *regionState) allocateResourceID() int {
	if s.nextResourceID <= 0 {
		s.nextResourceID = 1
	}
	id := s.nextResourceID
	s.nextResourceID++
	return id
}

func (s *regionState) consumedAt(name string) (frontend.Position, bool) {
	if s == nil || name == "" {
		return frontend.Position{}, false
	}
	if consumedAt, ok := s.consumedVars[name]; ok {
		return consumedAt, true
	}
	if id, ok := s.resourceID(name); ok {
		consumedAt, consumed := s.consumedResources[id]
		return consumedAt, consumed
	}
	return frontend.Position{}, false
}

func isResourceHandleType(typeName string) bool {
	switch typeName {
	case "actor", "island", "task.group", "task.i32", surfaceSurfaceTypeName:
		return true
	default:
		return strings.HasPrefix(typeName, "task.i32.throws.")
	}
}

func typeContainsResourceHandle(typeName string, types map[string]*TypeInfo) bool {
	return typeContainsResourceHandleVisiting(typeName, types, map[string]bool{})
}

func typeContainsResourceHandleVisiting(
	typeName string,
	types map[string]*TypeInfo,
	visiting map[string]bool,
) bool {
	if typeName == surfaceFrameTypeName {
		return false
	}
	if isResourceHandleType(typeName) {
		return true
	}
	info, ok := types[typeName]
	if !ok {
		return false
	}
	switch info.Kind {
	case TypeStruct:
		if visiting[typeName] {
			return false
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, field := range info.Fields {
			if typeContainsResourceHandleVisiting(field.TypeName, types, visiting) {
				return true
			}
		}
	case TypeEnum:
		if visiting[typeName] {
			return false
		}
		visiting[typeName] = true
		defer delete(visiting, typeName)
		for _, c := range info.EnumCases {
			for _, payload := range c.PayloadTypes {
				if typeContainsResourceHandleVisiting(payload, types, visiting) {
					return true
				}
			}
		}
	case TypeArray, TypeOptional:
		return typeContainsResourceHandleVisiting(info.ElemType, types, visiting)
	}
	return false
}

func (s *regionState) clearResourceTree(prefix string) {
	if s == nil || prefix == "" {
		return
	}
	delete(s.resourceVars, prefix)
	prefixDot := prefix + "."
	for name := range s.resourceVars {
		if strings.HasPrefix(name, prefixDot) {
			delete(s.resourceVars, name)
		}
	}
}

func (s *regionState) clearRegionTree(prefix string) {
	if s == nil || prefix == "" {
		return
	}
	delete(s.regionVars, prefix)
	delete(s.unknownVars, prefix)
	delete(s.unknownConflicts, prefix)
	s.clearOwnedRegionSliceOwnerTree(prefix)
	prefixDot := prefix + "."
	for name := range s.regionVars {
		if strings.HasPrefix(name, prefixDot) {
			delete(s.regionVars, name)
			delete(s.unknownVars, name)
			delete(s.unknownConflicts, name)
		}
	}
}

func (s *regionState) bindRegion(name string, regionID int) {
	if s == nil || name == "" {
		return
	}
	if regionID == regionNone {
		delete(s.regionVars, name)
		delete(s.unknownVars, name)
		delete(s.unknownConflicts, name)
		return
	}
	s.regionVars[name] = regionID
	delete(s.unknownVars, name)
	delete(s.unknownConflicts, name)
}

func (s *regionState) invalidateBorrowedRegionsAfterAwait(pos frontend.Position) {
	if s == nil {
		return
	}
	if s.awaitInvalidatedBorrow == nil {
		s.awaitInvalidatedBorrow = make(map[int]frontend.Position)
	}
	for _, regionID := range s.regionVars {
		if _, borrowed := s.borrowedParamOwner(regionID); !borrowed {
			continue
		}
		if existing, exists := s.awaitInvalidatedBorrow[regionID]; exists {
			s.awaitInvalidatedBorrow[regionID] = earliestPosition(existing, pos)
			continue
		}
		s.awaitInvalidatedBorrow[regionID] = pos
	}
}

func (s *regionState) checkBorrowedRegionAfterAwait(
	regionID int,
	name string,
	pos frontend.Position,
) error {
	if s == nil || regionID == regionNone || regionID == regionUnknown {
		return nil
	}
	if _, borrowed := s.borrowedParamOwner(regionID); !borrowed {
		return nil
	}
	awaitAt, invalidated := s.awaitInvalidatedBorrow[regionID]
	if !invalidated {
		return nil
	}
	if name == "" {
		name = "<borrow>"
	}
	if awaitAt.Line == 0 {
		return lifetimeDiagnosticf(
			pos,
			"borrowed view '%s' cannot be used after await suspension",
			name,
		)
	}
	return lifetimeDiagnosticf(
		pos,
		"borrowed view '%s' cannot be used after await suspension at %s",
		name,
		frontend.FormatPos(awaitAt),
	)
}

func (s *regionState) setExprRegionTree(expr frontend.Expr, tree map[string]int) {
	if s == nil || expr == nil {
		return
	}
	if len(tree) == 0 {
		delete(s.exprRegionTrees, expr)
		return
	}
	s.exprRegionTrees[expr] = copyRegionTree(tree)
}

func (s *regionState) exprRegionTree(expr frontend.Expr) (map[string]int, bool) {
	if s == nil || expr == nil {
		return nil, false
	}
	tree, ok := s.exprRegionTrees[expr]
	if !ok {
		return nil, false
	}
	return copyRegionTree(tree), true
}

func (s *regionState) pushDeferCaptureFrame() {
	if s == nil {
		return
	}
	s.deferCaptureFrames = append(s.deferCaptureFrames, make(map[string]frontend.Position))
}

func (s *regionState) popDeferCaptureFrame() {
	if s == nil || len(s.deferCaptureFrames) == 0 {
		return
	}
	s.deferCaptureFrames = s.deferCaptureFrames[:len(s.deferCaptureFrames)-1]
}

func (s *regionState) registerDeferCaptures(captures map[string]frontend.Position) {
	if s == nil || len(captures) == 0 || len(s.deferCaptureFrames) == 0 {
		return
	}
	frame := s.deferCaptureFrames[len(s.deferCaptureFrames)-1]
	for name, pos := range captures {
		if _, exists := frame[name]; !exists {
			frame[name] = pos
		}
	}
}

func (s *regionState) checkPendingDeferCaptures(pos frontend.Position) error {
	if s == nil || (len(s.consumedVars) == 0 && len(s.consumedResources) == 0) ||
		len(s.deferCaptureFrames) == 0 {
		return nil
	}
	for i := len(s.deferCaptureFrames) - 1; i >= 0; i-- {
		for name, capturedAt := range s.deferCaptureFrames[i] {
			consumedAt, consumed := s.deferredCaptureConsumedAt(name)
			if !consumed {
				continue
			}
			if pos.Line == 0 {
				pos = consumedAt
			}
			return fmt.Errorf(
				"%s: defer cleanup captures value '%s' at %s, but it was consumed at %s before cleanup ran",
				frontend.FormatPos(pos),
				name,
				frontend.FormatPos(capturedAt),
				frontend.FormatPos(consumedAt),
			)
		}
	}
	return nil
}

func (s *regionState) deferredCaptureConsumedAt(name string) (frontend.Position, bool) {
	if s == nil || name == "" {
		return frontend.Position{}, false
	}
	queryName := name
	if source, ok := s.ownershipAliasSource(name); ok {
		queryName = source
	}
	if consumedAt, consumed := s.consumedAt(queryName); consumed {
		return consumedAt, true
	}
	for consumedName, consumedAt := range s.consumedVars {
		reportName := consumedName
		if source, ok := s.ownershipAliasSource(consumedName); ok {
			reportName = source
		}
		if reportName == queryName || ownershipPathPrefix(queryName, reportName) {
			return consumedAt, true
		}
	}
	for resourceName, resourceID := range s.resourceVars {
		consumedAt, consumed := s.consumedResources[resourceID]
		if !consumed {
			continue
		}
		reportName := resourceName
		if source, ok := s.ownershipAliasSource(resourceName); ok {
			reportName = source
		}
		if reportName == queryName || ownershipPathPrefix(queryName, reportName) {
			return consumedAt, true
		}
	}
	return frontend.Position{}, false
}

// ---- region_tree_summary.go ----

type regionConflict struct {
	leftLabel  string
	leftRegion int

	rightLabel  string
	rightRegion int
}

func (s *regionState) enterIsland(name string) error {
	id, ok := s.islandScopes[name]
	if !ok {
		return fmt.Errorf("unknown island scope '%s'", name)
	}
	s.activateScope(id)
	s.regionVars[name] = id
	s.bindResource(name, "", true)
	return nil
}

func (s *regionState) exitIsland() {
	if len(s.activeScopes) == 0 {
		return
	}
	s.deactivateScope(s.activeScopes[len(s.activeScopes)-1])
}

func (s *regionState) activateScope(id int) {
	if s == nil || id < 0 {
		return
	}
	if _, exists := s.activeIndex[id]; exists {
		return
	}
	s.activeScopes = append(s.activeScopes, id)
	s.activeIndex[id] = len(s.activeScopes) - 1
}

func (s *regionState) deactivateScope(id int) {
	if s == nil || id < 0 {
		return
	}
	idx, ok := s.activeIndex[id]
	if !ok {
		return
	}
	delete(s.activeIndex, id)
	copy(s.activeScopes[idx:], s.activeScopes[idx+1:])
	s.activeScopes = s.activeScopes[:len(s.activeScopes)-1]
	for i := idx; i < len(s.activeScopes); i++ {
		s.activeIndex[s.activeScopes[i]] = i
	}
}

func (s *regionState) isScopeActive(id int) bool {
	if id < 0 {
		return true
	}
	_, ok := s.activeIndex[id]
	return ok
}

func (s *regionState) scopeIndex(id int) (int, bool) {
	idx, ok := s.activeIndex[id]
	return idx, ok
}

func (s *regionState) isScopeWithin(targetID, regionID int) bool {
	if regionID < 0 {
		return true
	}
	if targetID < 0 {
		return false
	}
	regionIdx, ok := s.scopeIndex(regionID)
	if !ok {
		return false
	}
	targetIdx, ok := s.scopeIndex(targetID)
	if !ok {
		return false
	}
	return targetIdx >= regionIdx
}

func copyRegionVars(src map[string]int) map[string]int {
	return semanticsregions.CopyVars(src)
}

func copyRegionTree(src map[string]int) map[string]int {
	return semanticsregions.CopyTree(src)
}

func mergeRegionVars(a, b map[string]int) map[string]int {
	return semanticsregions.MergeVars(a, b)
}

func joinRegion(a, b int) int {
	return semanticsregions.Join(a, b)
}

func commonRegionFromTree(tree map[string]int) int {
	return semanticsregions.CommonFromTree(tree)
}

func constructorRegionFromTree(tree map[string]int) int {
	return semanticsregions.ConstructorFromTree(tree)
}

func regionTreeForExpr(
	typeName string,
	expr frontend.Expr,
	exprRegion int,
	types map[string]*TypeInfo,
	state *regionState,
) map[string]int {
	tree := make(map[string]int)
	appendRegionTree(tree, "", typeName, expr, exprRegion, types, state)
	return tree
}

func appendRegionTree(
	out map[string]int,
	prefix string,
	typeName string,
	expr frontend.Expr,
	exprRegion int,
	types map[string]*TypeInfo,
	state *regionState,
) {
	if !typeMayContainRegion(typeName, types) {
		return
	}
	if state != nil {
		if tree, ok := state.exprRegionTree(expr); ok {
			for leaf, regionID := range tree {
				if regionID != regionNone {
					out[joinResourcePath(prefix, leaf)] = regionID
				}
			}
			return
		}
		if sourcePrefix, ok := resourcePathForExpr(expr); ok {
			copied := false
			for _, leaf := range regionLeafPaths(typeName, types, "") {
				sourceLeaf := joinResourcePath(sourcePrefix, leaf)
				if regionID, ok := state.regionVars[sourceLeaf]; ok {
					out[joinResourcePath(prefix, leaf)] = regionID
					copied = true
				}
			}
			if copied {
				return
			}
		}
	}
	if info, ok := types[typeName]; ok && info.Kind == TypeOptional {
		appendRegionTree(
			out,
			resourceFieldPath(prefix, "$elem"),
			info.ElemType,
			expr,
			exprRegion,
			types,
			state,
		)
		return
	}
	if exprRegion == regionNone {
		return
	}
	for _, leaf := range regionLeafPaths(typeName, types, "") {
		out[joinResourcePath(prefix, leaf)] = exprRegion
	}
}

func bindRegionTreeFromExpr(
	name string,
	typeName string,
	expr frontend.Expr,
	exprRegion int,
	types map[string]*TypeInfo,
	state *regionState,
) {
	if state == nil || name == "" {
		return
	}
	state.clearRegionTree(name)
	if !typeMayContainRegion(typeName, types) {
		return
	}
	for leaf, regionID := range regionTreeForExpr(typeName, expr, exprRegion, types, state) {
		if regionID != regionNone {
			state.bindRegion(joinResourcePath(name, leaf), regionID)
		}
	}
}

func copyRegionTreeFromPath(
	dst string,
	src string,
	typeName string,
	types map[string]*TypeInfo,
	state *regionState,
) {
	if state == nil || dst == "" || src == "" {
		return
	}
	state.clearRegionTree(dst)
	if !typeMayContainRegion(typeName, types) {
		return
	}
	for _, leaf := range regionLeafPaths(typeName, types, "") {
		srcLeaf := joinResourcePath(src, leaf)
		if regionID, ok := state.regionVars[srcLeaf]; ok {
			state.bindRegion(joinResourcePath(dst, leaf), regionID)
		}
	}
}

func checkRegionTreeWithinScope(
	tree map[string]int,
	targetScopeID int,
	pos frontend.Position,
	state *regionState,
) error {
	if state == nil {
		return nil
	}
	for _, regionID := range tree {
		if regionID < 0 {
			continue
		}
		if !state.isScopeWithin(targetScopeID, regionID) {
			return lifetimeDiagnosticf(
				pos,
				"slice from scoped island cannot escape to outer scope (value: %s, target: %s)",
				formatRegionID(state, regionID),
				formatScopeID(state, targetScopeID),
			)
		}
	}
	return nil
}

func checkRegionUsable(regionID int, name string, pos frontend.Position, state *regionState) error {
	if state == nil || regionID == regionNone {
		return nil
	}
	if regionID == regionUnknown {
		return fmt.Errorf("%s: ambiguous region for '%s'", frontend.FormatPos(pos), name)
	}
	if err := state.checkBorrowedRegionAfterAwait(regionID, name, pos); err != nil {
		return err
	}
	if !state.isScopeActive(regionID) {
		return lifetimeDiagnosticf(pos, "slice from scoped island is out of scope")
	}
	return nil
}

func regionLeafPaths(typeName string, types map[string]*TypeInfo, prefix string) []string {
	return regionLeafPathsVisiting(typeName, types, prefix, map[string]bool{})
}

func regionLeafPathsVisiting(
	typeName string,
	types map[string]*TypeInfo,
	prefix string,
	visiting map[string]bool,
) []string {
	info, ok := types[typeName]
	if !ok {
		return nil
	}
	if visiting[typeName] {
		return nil
	}
	visiting[typeName] = true
	defer delete(visiting, typeName)
	switch info.Kind {
	case TypeSlice, TypeIsland, TypeStr:
		return []string{prefix}
	case TypeStruct:
		out := []string{}
		for _, field := range info.Fields {
			out = append(
				out,
				regionLeafPathsVisiting(
					field.TypeName,
					types,
					resourceFieldPath(prefix, field.Name),
					visiting,
				)...)
		}
		return out
	case TypeEnum:
		out := []string{}
		for _, c := range info.EnumCases {
			for i, payload := range c.PayloadTypes {
				out = append(
					out,
					regionLeafPathsVisiting(
						payload,
						types,
						resourceEnumPayloadPath(prefix, c.Ordinal, i),
						visiting,
					)...)
			}
		}
		return out
	case TypeArray:
		return []string{prefix}
	case TypeOptional:
		return regionLeafPathsVisiting(
			info.ElemType,
			types,
			resourceFieldPath(prefix, "$elem"),
			visiting,
		)
	default:
		return nil
	}
}

func markUnknownRegions(state *regionState) {
	if state == nil {
		return
	}
	for name := range state.unknownVars {
		if state.regionVars[name] != regionUnknown {
			delete(state.unknownVars, name)
		}
	}
	for name := range state.unknownConflicts {
		if state.regionVars[name] != regionUnknown {
			delete(state.unknownConflicts, name)
		}
	}
	for name, regionID := range state.regionVars {
		if regionID == regionUnknown {
			state.unknownVars[name] = true
			continue
		}
		delete(state.unknownVars, name)
		delete(state.unknownConflicts, name)
	}
}

func initParamRegions(params []frontend.ParamDecl, state *regionState, types map[string]*TypeInfo) {
	if state != nil && (state.paramNames == nil || len(state.paramNames) != len(params)) {
		state.paramNames = make([]string, len(params))
		for i := range params {
			state.paramNames[i] = params[i].Name
		}
	}
	next := regionParamStart
	for i := range params {
		param := params[i]
		if typeContainsResourceHandle(param.Type.Name, types) {
			for _, leaf := range resourceLeafPaths(param.Type.Name, types, "") {
				name := joinResourcePath(param.Name, leaf)
				state.bindResource(name, "", true)
				if id, ok := state.resourceID(name); ok {
					state.resourceParamIndex[id] = i
					state.resourceParamPath[id] = leaf
				}
			}
		}
		if typeMayContainRegion(param.Type.Name, types) {
			state.regionVars[param.Name] = next
			state.paramRegionIndex[next] = i
			if param.Ownership == "borrow" {
				state.borrowedParamRegion[next] = param.Name
			}
			next--
		}
		if param.Ownership == "borrow" && typeMayContainPtr(param.Type.Name, types) {
			for _, leaf := range ptrLeafPaths(param.Type.Name, types, "") {
				state.borrowedPtrAliases[joinResourcePath(param.Name, leaf)] = param.Name
			}
		}
	}
}

func (s *regionState) resourceParamOwner(name string) (int, string, bool) {
	if s == nil || name == "" {
		return 0, "", false
	}
	id, ok := s.resourceID(name)
	if !ok {
		return 0, "", false
	}
	idx, ok := s.resourceParamIndex[id]
	if !ok {
		return 0, "", false
	}
	return idx, s.resourceParamPath[id], true
}

func (s *regionState) borrowedParamOwner(regionID int) (string, bool) {
	if s == nil || regionID >= regionNone {
		return "", false
	}
	name, ok := s.borrowedParamRegion[regionID]
	return name, ok
}

func (s *regionState) bindExplicitBorrow(owner string) int {
	if s == nil {
		return regionNone
	}
	if owner == "" {
		owner = "<borrow>"
	}
	if s.nextExplicitBorrow >= regionNone {
		s.nextExplicitBorrow = regionExplicitBorrowStart
	}
	id := s.nextExplicitBorrow
	s.nextExplicitBorrow--
	s.borrowedParamRegion[id] = owner
	return id
}

func formatRegionID(state *regionState, regionID int) string {
	switch {
	case regionID == regionNone:
		return "none"
	case regionID == regionUnknown:
		return "unknown"
	case regionID >= 0:
		if state != nil {
			if name, ok := state.islandNameByID[regionID]; ok && name != "" {
				return fmt.Sprintf("isl#%d(%s)", regionID, name)
			}
		}
		return fmt.Sprintf("isl#%d", regionID)
	default:
		if state != nil {
			if idx, ok := state.paramRegionIndex[regionID]; ok {
				if idx >= 0 && idx < len(state.paramNames) {
					return fmt.Sprintf("param#%d(%s)", idx, state.paramNames[idx])
				}
				return fmt.Sprintf("param#%d", idx)
			}
		}
		return fmt.Sprintf("param(%d)", regionID)
	}
}

func formatScopeID(state *regionState, scopeID int) string {
	if scopeID == regionNone {
		return "root"
	}
	if scopeID == regionUnknown {
		return "unknown"
	}
	if state != nil {
		if name, ok := state.islandNameByID[scopeID]; ok && name != "" {
			return fmt.Sprintf("scope#%d(%s)", scopeID, name)
		}
	}
	return fmt.Sprintf("scope#%d", scopeID)
}

func recordMergeConflicts(
	state *regionState,
	leftVars, rightVars map[string]int,
	leftLabel, rightLabel string,
) {
	if state == nil {
		return
	}
	for name, left := range leftVars {
		right, ok := rightVars[name]
		if !ok {
			right = regionNone
		}
		if left == right {
			continue
		}
		if left == regionNone && right == regionNone {
			continue
		}
		state.unknownConflicts[name] = regionConflict{
			leftLabel:   leftLabel,
			leftRegion:  left,
			rightLabel:  rightLabel,
			rightRegion: right,
		}
	}
	for name, right := range rightVars {
		if _, ok := leftVars[name]; ok {
			continue
		}
		if right == regionNone {
			continue
		}
		state.unknownConflicts[name] = regionConflict{
			leftLabel:   leftLabel,
			leftRegion:  regionNone,
			rightLabel:  rightLabel,
			rightRegion: right,
		}
	}
}

func (s *regionState) enterUnsafe() {
	s.unsafeDepth++
}

func (s *regionState) exitUnsafe() {
	if s.unsafeDepth > 0 {
		s.unsafeDepth--
	}
}

func (s *regionState) inUnsafe() bool {
	return s.unsafeDepth > 0
}

func (s *regionState) recordReturnRegion(regionID int, pos frontend.Position) error {
	if regionID == regionUnknown {
		return fmt.Errorf("%s: ambiguous region for return", frontend.FormatPos(pos))
	}
	if regionID >= 0 {
		return lifetimeDiagnosticf(pos, "return from scoped island is not allowed")
	}
	if !s.returnRegionSet {
		s.returnRegion = regionID
		s.returnRegionSet = true
		return nil
	}
	if s.returnRegion != regionID {
		return fmt.Errorf(
			"%s: return mixes values from different regions (first: %s, now: %s)",
			frontend.FormatPos(pos),
			formatRegionID(s, s.returnRegion),
			formatRegionID(s, regionID),
		)
	}
	return nil
}

func (s *regionState) recordReturnRegionSummary(tree map[string]int, pos frontend.Position) error {
	if s == nil || len(tree) == 0 {
		return nil
	}
	for returnPath, regionID := range tree {
		if regionID == regionUnknown {
			return fmt.Errorf("%s: ambiguous region for return", frontend.FormatPos(pos))
		}
		if regionID >= 0 {
			return lifetimeDiagnosticf(pos, "return from scoped island is not allowed")
		}
		idx, ok := s.paramRegionIndex[regionID]
		if !ok {
			return fmt.Errorf("%s: return region does not match parameter", frontend.FormatPos(pos))
		}
		if s.returnRegionSummary == nil {
			s.returnRegionSummary = ReturnRegionSummary{}
		}
		if existing, exists := s.returnRegionSummary[returnPath]; exists {
			if existing != idx {
				return fmt.Errorf(
					"%s: return mixes region provenance for return%s (first: param#%d, now: param#%d)",
					frontend.FormatPos(pos),
					formatResourceParamPath(returnPath),
					existing,
					idx,
				)
			}
			continue
		}
		s.returnRegionSummary[returnPath] = idx
	}
	return nil
}

func (s *regionState) recordReturnResourceParam(
	paramIndex int,
	path string,
	pos frontend.Position,
) error {
	if paramIndex < 0 {
		return nil
	}
	if !s.returnResourceSet {
		s.returnResourceParam = paramIndex
		s.returnResourcePath = path
		s.returnResourceSet = true
		return nil
	}
	if s.returnResourceParam != paramIndex || s.returnResourcePath != path {
		return fmt.Errorf(
			"%s: return mixes resource provenance (first: param#%d%s, now: param#%d%s)",
			frontend.FormatPos(pos),
			s.returnResourceParam,
			formatResourceParamPath(s.returnResourcePath),
			paramIndex,
			formatResourceParamPath(path),
		)
	}
	return nil
}

func (s *regionState) recordReturnResourceSummary(
	summary ReturnResourceSummary,
	pos frontend.Position,
) error {
	if s == nil {
		return nil
	}
	for returnPath, provenances := range summary {
		for _, provenance := range provenances {
			if provenance.ParamIndex < 0 {
				continue
			}
			if s.returnResourceSummary == nil {
				s.returnResourceSummary = ReturnResourceSummary{}
			}
			existing := s.returnResourceSummary[returnPath]
			if len(existing) == 0 {
				s.returnResourceSummary[returnPath] = []ResourceProvenance{provenance}
				continue
			}
			if len(existing) == 1 && existing[0] == provenance {
				continue
			}
			first := existing[0]
			return fmt.Errorf(
				("%s: return mixes resource provenance (first: param#%d%s -> " +
					"return%s, now: param#%d%s -> return%s)"),
				frontend.FormatPos(pos),
				first.ParamIndex,
				formatResourceParamPath(first.ParamPath),
				formatResourceParamPath(returnPath),
				provenance.ParamIndex,
				formatResourceParamPath(provenance.ParamPath),
				formatResourceParamPath(returnPath),
			)
		}
	}
	if len(s.returnResourceSummary) > 0 {
		s.returnResourceSet = true
		if provenances := s.returnResourceSummary[""]; len(provenances) == 1 {
			s.returnResourceParam = provenances[0].ParamIndex
			s.returnResourcePath = provenances[0].ParamPath
		}
	}
	return nil
}

func (s *regionState) recordThrowResourceSummary(
	summary ReturnResourceSummary,
	pos frontend.Position,
) error {
	if s == nil {
		return nil
	}
	for throwPath, provenances := range summary {
		for _, provenance := range provenances {
			if provenance.ParamIndex < 0 {
				continue
			}
			if s.throwResourceSummary == nil {
				s.throwResourceSummary = ReturnResourceSummary{}
			}
			existing := s.throwResourceSummary[throwPath]
			if len(existing) == 0 {
				s.throwResourceSummary[throwPath] = []ResourceProvenance{provenance}
				continue
			}
			if len(existing) == 1 && existing[0] == provenance {
				continue
			}
			first := existing[0]
			return fmt.Errorf(
				("%s: throw mixes resource provenance (first: param#%d%s -> " +
					"throw%s, now: param#%d%s -> throw%s)"),
				frontend.FormatPos(pos),
				first.ParamIndex,
				formatResourceParamPath(first.ParamPath),
				formatResourceParamPath(throwPath),
				provenance.ParamIndex,
				formatResourceParamPath(provenance.ParamPath),
				formatResourceParamPath(throwPath),
			)
		}
	}
	return nil
}

func formatResourceParamPath(path string) string {
	if path == "" {
		return ""
	}
	return "." + path
}

func (s *regionState) recordUnknownReturnResource() {
	if s == nil {
		return
	}
	s.returnResourceUnknown = true
}

func typeMayContainRegion(typeName string, types map[string]*TypeInfo) bool {
	return typeMayContainRegionVisiting(typeName, types, map[string]bool{}, map[string]bool{})
}

func typeMayContainPtr(typeName string, types map[string]*TypeInfo) bool {
	return typeMayContainPtrVisiting(typeName, types, map[string]bool{}, map[string]bool{})
}

func typeMayContainPtrVisiting(
	typeName string,
	types map[string]*TypeInfo,
	visiting map[string]bool,
	memo map[string]bool,
) bool {
	if resolved, ok := memo[typeName]; ok {
		return resolved
	}
	if typeName == "ptr" {
		memo[typeName] = true
		return true
	}
	if typeName == "fnptr" {
		memo[typeName] = false
		return false
	}
	if visiting[typeName] {
		return false
	}
	info, ok := types[typeName]
	if !ok {
		return false
	}
	visiting[typeName] = true
	defer delete(visiting, typeName)

	result := false
	switch info.Kind {
	case TypeStruct:
		for _, field := range info.Fields {
			if typeMayContainPtrVisiting(field.TypeName, types, visiting, memo) {
				result = true
				break
			}
		}
	case TypeEnum:
		for _, c := range info.EnumCases {
			for _, payload := range c.PayloadTypes {
				if typeMayContainPtrVisiting(payload, types, visiting, memo) {
					result = true
					break
				}
			}
			if result {
				break
			}
		}
	case TypeArray, TypeOptional:
		result = typeMayContainPtrVisiting(info.ElemType, types, visiting, memo)
	default:
		result = false
	}
	memo[typeName] = result
	return result
}

func typeMayContainRegionVisiting(
	typeName string,
	types map[string]*TypeInfo,
	visiting map[string]bool,
	memo map[string]bool,
) bool {
	if resolved, ok := memo[typeName]; ok {
		return resolved
	}
	if visiting[typeName] {
		return false
	}
	info, ok := types[typeName]
	if !ok {
		return false
	}
	visiting[typeName] = true
	defer delete(visiting, typeName)

	result := false
	switch info.Kind {
	case TypeSlice:
		result = true
	case TypeIsland:
		result = true
	case TypeStr:
		result = true
	case TypeStruct:
		for _, field := range info.Fields {
			if typeMayContainRegionVisiting(field.TypeName, types, visiting, memo) {
				result = true
				break
			}
		}
	case TypeEnum:
		for _, c := range info.EnumCases {
			for _, payload := range c.PayloadTypes {
				if typeMayContainRegionVisiting(payload, types, visiting, memo) {
					result = true
					break
				}
			}
			if result {
				break
			}
		}
	case TypeArray:
		result = true
	case TypeOptional:
		result = typeMayContainRegionVisiting(info.ElemType, types, visiting, memo)
	default:
		result = false
	}
	memo[typeName] = result
	return result
}

func localScopeID(name string, state *regionState) int {
	if state == nil {
		return regionNone
	}
	if id, ok := state.localScopes[name]; ok {
		return id
	}
	return regionNone
}

func patternBindingScopeID(pattern frontend.Expr, state *regionState) int {
	if state == nil || pattern == nil {
		return regionNone
	}
	switch p := pattern.(type) {
	case *frontend.SomePatternExpr:
		return localScopeID(p.Name, state)
	case *frontend.EnumCasePatternExpr:
		for _, binding := range p.Bindings {
			if id := localScopeID(binding, state); id != regionNone {
				return id
			}
		}
	}
	return regionNone
}

func checkLocalScope(name string, state *regionState, pos frontend.Position) error {
	if state != nil {
		if ids := state.localScopeSets[name]; len(ids) > 0 {
			for id := range ids {
				if state.isScopeActive(id) {
					return nil
				}
			}
			return fmt.Errorf("%s: identifier '%s' is out of scope", frontend.FormatPos(pos), name)
		}
	}
	scopeID := localScopeID(name, state)
	if scopeID == regionNone {
		return nil
	}
	if !state.isScopeActive(scopeID) {
		return fmt.Errorf("%s: identifier '%s' is out of scope", frontend.FormatPos(pos), name)
	}
	return nil
}

func withActiveScope(state *regionState, scopeID int, run func() error) error {
	if state == nil || scopeID == regionNone {
		return run()
	}
	state.activateScope(scopeID)
	defer state.deactivateScope(scopeID)
	return run()
}
