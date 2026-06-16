package semantics

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

type ownershipArgRef struct {
	path string
	pos  frontend.Position
}

func canonicalOwnershipAccessPath(expr frontend.Expr) (string, bool) {
	base, fields, _, ok := splitOwnershipPath(expr)
	if !ok {
		return "", false
	}
	if len(fields) == 0 || base == "" {
		return base, len(fields) == 0
	}
	path := base
	for _, field := range fields {
		path = joinOwnershipPath(path, field)
	}
	return path, true
}

func splitOwnershipPath(expr frontend.Expr) (string, []string, frontend.Position, bool) {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return e.Name, nil, e.At, true
	case *frontend.FieldAccessExpr:
		base, fields, pos, ok := splitOwnershipPath(e.Base)
		if !ok {
			return "", nil, pos, false
		}
		fields = append(fields, e.Field)
		return base, fields, e.At, true
	case *frontend.IndexExpr:
		base, fields, pos, ok := splitOwnershipPath(e.Base)
		if !ok {
			return "", nil, pos, false
		}
		fields = append(fields, splitOwnershipIndexSegment(e.Index))
		return base, fields, e.At, true
	default:
		return "", nil, expr.Pos(), false
	}
}

func splitOwnershipPathSegments(path string) []string {
	if path == "" {
		return nil
	}
	segments := make([]string, 0, 4)
	start := 0
	depth := 0
	for i := 0; i <= len(path); i++ {
		if i == len(path) || (path[i] == '.' && depth == 0) {
			if i >= start {
				segments = append(segments, path[start:i])
			}
			start = i + 1
			continue
		}
		switch path[i] {
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		}
	}
	return segments
}

func ownershipPathSegmentsMatch(left string, right string) bool {
	if left == right {
		return true
	}
	if left == "[_]" || right == "[_]" {
		return true
	}
	return false
}

func ownershipPathPrefix(prefix string, path string) bool {
	if prefix == "" || path == "" {
		return false
	}
	prefixParts := splitOwnershipPathSegments(prefix)
	pathParts := splitOwnershipPathSegments(path)
	if len(prefixParts) == 0 || len(prefixParts) > len(pathParts) {
		return false
	}
	for i := 0; i < len(prefixParts); i++ {
		if !ownershipPathSegmentsMatch(prefixParts[i], pathParts[i]) {
			return false
		}
	}
	return true
}

func ownershipPathParent(prefix string) string {
	parts := splitOwnershipPathSegments(prefix)
	if len(parts) <= 1 {
		return ""
	}
	return strings.Join(parts[:len(parts)-1], ".")
}

func splitOwnershipIndexSegment(index frontend.Expr) string {
	switch i := index.(type) {
	case *frontend.NumberExpr:
		return fmt.Sprintf("[%d]", i.Value)
	case *frontend.IdentExpr:
		return "[" + i.Name + "]"
	default:
		return "[_]"
	}
}

func joinOwnershipPath(prefix string, segment string) string {
	if segment == "" {
		return prefix
	}
	if strings.HasPrefix(segment, "[") {
		return prefix + segment
	}
	if prefix == "" {
		return segment
	}
	return prefix + "." + segment
}

func consumeLocalArgumentName(expr frontend.Expr, callee string, callback bool, phraseOverride ...string) (string, error) {
	targetPhrase := fmt.Sprintf("'%s'", callee)
	if callback {
		targetPhrase = fmt.Sprintf("callback '%s'", callee)
	}
	if len(phraseOverride) > 0 && phraseOverride[0] != "" {
		targetPhrase = phraseOverride[0]
	}
	path, ok := canonicalOwnershipAccessPath(expr)
	if !ok {
		return "", ownershipDiagnosticf(expr.Pos(), "consume argument for %s must be a local value", targetPhrase)
	}
	return path, nil
}

func checkWholeOwnershipValueAvailable(expr frontend.Expr, types map[string]*TypeInfo, module string, imports map[string]string, state *regionState) error {
	return checkWholeOwnershipValueAvailableForType(expr, "", types, module, imports, state)
}

func checkWholeOwnershipValueAvailableForType(expr frontend.Expr, expectedType string, types map[string]*TypeInfo, module string, imports map[string]string, state *regionState) error {
	if expr == nil || state == nil {
		return nil
	}
	if path, ok := canonicalOwnershipAccessPath(expr); ok {
		if expectedType != "" && typeContainsResourceHandle(expectedType, types) {
			if err := state.checkNoConsumedProperDescendants(path, expr.Pos()); err != nil {
				return err
			}
		} else {
			if err := state.checkNoConsumedDescendants(path, expr.Pos()); err != nil {
				return err
			}
		}
	}
	switch e := expr.(type) {
	case *frontend.StructLitExpr:
		typeName, err := resolveTypeName(&e.Type, module, imports)
		if err != nil {
			return err
		}
		info, ok := types[typeName]
		if !ok || info.Kind != TypeStruct {
			return nil
		}
		for _, field := range e.Fields {
			fieldInfo, ok := info.FieldMap[field.Name]
			if !ok {
				continue
			}
			if err := checkWholeOwnershipValueAvailableForType(field.Value, fieldInfo.TypeName, types, module, imports, state); err != nil {
				return err
			}
		}
	case *frontend.CallExpr:
		if e.ResolvedType == "" {
			return nil
		}
		info, ok := types[e.ResolvedType]
		if !ok {
			return nil
		}
		switch info.Kind {
		case TypeStruct:
			for i, field := range info.Fields {
				if i >= len(e.Args) {
					break
				}
				if err := checkWholeOwnershipValueAvailableForType(e.Args[i], field.TypeName, types, module, imports, state); err != nil {
					return err
				}
			}
		case TypeEnum:
			_, caseInfo, found, err := resolveEnumCaseConstructorCall(e, types, module, imports)
			if err != nil {
				return err
			}
			if !found {
				return nil
			}
			for i, arg := range e.Args {
				if i >= len(caseInfo.PayloadTypes) {
					break
				}
				if err := checkWholeOwnershipValueAvailableForType(arg, caseInfo.PayloadTypes[i], types, module, imports, state); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func ownershipAccessPathsAlias(left, right string) bool {
	if left == "" || right == "" {
		return false
	}
	if left == right {
		return true
	}
	return ownershipPathPrefix(left, right) || ownershipPathPrefix(right, left)
}

func findOwnershipAlias(refs []ownershipArgRef, path string) (ownershipArgRef, bool) {
	for _, ref := range refs {
		if ownershipAccessPathsAlias(ref.path, path) {
			return ref, true
		}
	}
	return ownershipArgRef{}, false
}

func checkFunctionTypedCallArguments(
	e *frontend.CallExpr,
	locals map[string]LocalInfo,
	globals map[string]GlobalInfo,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	effects *effectContext,
	analysis *functionAnalysisState,
	paramTypes []string,
	paramOwnerships []string,
	valueCallPhrase string,
) ([]int, error) {
	consumeArgs := make([]string, len(e.Args))
	consumeArgTypes := make([]string, len(e.Args))
	consumeArgRefs := make([]ownershipArgRef, 0, len(e.Args))
	borrowArgs := make([]ownershipArgRef, 0, len(e.Args))
	inoutArgs := make([]ownershipArgRef, 0, len(e.Args))
	argRegions := make([]int, len(e.Args))
	for i, arg := range e.Args {
		argType, argRegion, err := checkExprWithEffects(arg, locals, globals, funcs, types, module, imports, state, effects, analysis)
		if err != nil {
			return nil, err
		}
		argRegions[i] = argRegion
		if !typesCompatibleWithNullPtr(paramTypes[i], argType, arg) {
			return nil, fmt.Errorf("%s: type mismatch for %s arg %d", frontend.FormatPos(arg.Pos()), valueCallPhrase, i+1)
		}
		paramOwnership := ownershipAt(paramOwnerships, i)
		if paramOwnership == "" {
			if borrowedName, borrowed := state.borrowedParamOwner(argRegion); borrowed {
				return nil, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed to non-borrow parameter %d of %s", borrowedName, i+1, valueCallPhrase)
			}
			if argType == "ptr" {
				if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
					return nil, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed to non-borrow parameter %d of %s", borrowedName, i+1, valueCallPhrase)
				}
			}
			if argType != "ptr" && (typeMayContainRegion(argType, types) || typeMayContainPtr(argType, types)) {
				if err := checkBorrowedEscape(arg, locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
					return ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed to non-borrow parameter %d of %s", borrowedName, i+1, valueCallPhrase)
				}); err != nil {
					return nil, err
				}
			}
			if path, ok := canonicalOwnershipAccessPath(arg); ok {
				if err := state.checkNoConsumedDescendants(path, arg.Pos()); err != nil {
					return nil, err
				}
			}
		}
		if paramOwnership == "consume" {
			name, err := consumeLocalArgumentName(arg, e.Name, true)
			if err != nil {
				return nil, err
			}
			if err := state.checkNoConsumedDescendants(name, arg.Pos()); err != nil {
				return nil, err
			}
			if borrowedName, borrowed := state.borrowedParamOwner(argRegion); borrowed {
				return nil, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be consumed by %s", borrowedName, valueCallPhrase)
			}
			if argType == "ptr" {
				if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
					return nil, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be consumed by %s", borrowedName, valueCallPhrase)
				}
			}
			if argType != "ptr" && (typeMayContainRegion(argType, types) || typeMayContainPtr(argType, types)) {
				if err := checkBorrowedEscape(arg, locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
					return ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be consumed by %s", borrowedName, valueCallPhrase)
				}); err != nil {
					return nil, err
				}
			}
			path := name
			if first, exists := findOwnershipAlias(inoutArgs, path); exists {
				return nil, ownershipDiagnosticf(arg.Pos(), "consumed argument '%s' aliases inout argument in %s (inout at %s)", path, valueCallPhrase, frontend.FormatPos(first.pos))
			}
			consumeArgs[i] = name
			consumeArgTypes[i] = argType
			consumeArgRefs = append(consumeArgRefs, ownershipArgRef{path: path, pos: arg.Pos()})
		}
		if paramOwnership == "borrow" {
			path, ok := canonicalOwnershipAccessPath(arg)
			if ok {
				if err := state.checkNoConsumedDescendants(path, arg.Pos()); err != nil {
					return nil, err
				}
				if first, exists := findOwnershipAlias(inoutArgs, path); exists {
					return nil, ownershipDiagnosticf(arg.Pos(), "borrowed argument '%s' aliases inout argument in %s (inout at %s)", path, valueCallPhrase, frontend.FormatPos(first.pos))
				}
				borrowArgs = append(borrowArgs, ownershipArgRef{path: path, pos: arg.Pos()})
			}
		}
		if paramOwnership == "inout" {
			path, ok := canonicalOwnershipAccessPath(arg)
			if !ok {
				return nil, ownershipDiagnosticf(arg.Pos(), "inout argument for %s must be a mutable local value", valueCallPhrase)
			}
			if borrowedName, borrowed := state.borrowedParamOwner(argRegion); borrowed {
				return nil, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed as inout to %s", borrowedName, valueCallPhrase)
			}
			if argType == "ptr" {
				if borrowedName, borrowed := borrowedPtrOwnerFromExpr(arg, state, nil); borrowed {
					return nil, ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed as inout to %s", borrowedName, valueCallPhrase)
				}
			}
			if argType != "ptr" && (typeMayContainRegion(argType, types) || typeMayContainPtr(argType, types)) {
				if err := checkBorrowedEscape(arg, locals, globals, funcs, types, module, imports, state, effects, analysis, func(borrowedName string) error {
					return ownershipDiagnosticf(arg.Pos(), "borrowed value derived from '%s' cannot be passed as inout to %s", borrowedName, valueCallPhrase)
				}); err != nil {
					return nil, err
				}
			}
			if err := state.checkNoConsumedDescendants(path, arg.Pos()); err != nil {
				return nil, err
			}
			targetInfo, _, err := resolveAssignTarget(arg, locals, globals, types)
			if err != nil || !targetInfo.Mutable || targetInfo.Global {
				return nil, ownershipDiagnosticf(arg.Pos(), "inout argument '%s' for %s must be mutable", path, valueCallPhrase)
			}
			if first, exists := findOwnershipAlias(inoutArgs, path); exists {
				return nil, ownershipDiagnosticf(arg.Pos(), "inout argument '%s' used more than once in %s (first at %s)", path, valueCallPhrase, frontend.FormatPos(first.pos))
			}
			if first, exists := findOwnershipAlias(borrowArgs, path); exists {
				return nil, ownershipDiagnosticf(arg.Pos(), "inout argument '%s' aliases borrowed argument in %s (borrow at %s)", path, valueCallPhrase, frontend.FormatPos(first.pos))
			}
			if first, exists := findOwnershipAlias(consumeArgRefs, path); exists {
				return nil, ownershipDiagnosticf(arg.Pos(), "inout argument '%s' aliases consumed argument in %s (consume at %s)", path, valueCallPhrase, frontend.FormatPos(first.pos))
			}
			inoutArgs = append(inoutArgs, ownershipArgRef{path: path, pos: arg.Pos()})
		}
	}
	for i, name := range consumeArgs {
		if name == "" {
			continue
		}
		for j := 0; j < i; j++ {
			if consumeArgs[j] == name {
				return nil, ownershipDiagnosticf(e.Args[i].Pos(), "value '%s' consumed more than once in %s", name, valueCallPhrase)
			}
			if resourceValuesAlias(consumeArgs[j], consumeArgTypes[j], name, consumeArgTypes[i], types, state) {
				return nil, ownershipDiagnosticf(e.Args[i].Pos(), "value '%s' consumed more than once in %s", name, valueCallPhrase)
			}
		}
		markConsumedResourceValue(name, consumeArgTypes[i], types, state, e.Args[i].Pos())
	}
	return argRegions, nil
}

// checkCallExprWithEffects intentionally keeps call validation in one ordered
// path: resolve local/builtin/imported targets, enforce semantic clauses and
// effects, validate async/throw context, then check arguments, ownership, and
// resource provenance before returning type and region metadata.
