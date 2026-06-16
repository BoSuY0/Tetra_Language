package semantics

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/module"
	semanticspolicy "tetra_language/compiler/internal/semantics/policy"
)

func validateGenericFuncDecl(fn *frontend.FuncDecl, module string, imports map[string]string, protocolInfos map[string]genericProtocolInfo, types map[string]*TypeInfo) error {
	if len(fn.TypeParams) == 0 {
		return nil
	}
	params := map[string]struct{}{}
	for _, name := range fn.TypeParams {
		params[name] = struct{}{}
	}
	boundParams := map[string]struct{}{}
	for _, bound := range fn.TypeParamBounds {
		if _, ok := params[bound.Name]; !ok {
			return fmt.Errorf("%s: generic bound references unknown type parameter '%s'", frontend.FormatPos(bound.At), bound.Name)
		}
		boundParams[bound.Name] = struct{}{}
		if bound.Bound.Kind != frontend.TypeRefNamed || len(bound.Bound.TypeArgs) > 0 {
			return fmt.Errorf("%s: generic bound for '%s' must name a protocol", frontend.FormatPos(bound.Bound.At), bound.Name)
		}
		boundRef := bound.Bound
		resolved, err := resolveTypeName(&boundRef, module, imports)
		if err != nil {
			return err
		}
		proto, ok := protocolInfos[resolved]
		if !ok {
			if _, isType := types[resolved]; isType {
				return fmt.Errorf("%s: generic bound '%s' for '%s' must name a protocol, got non-protocol type '%s'", frontend.FormatPos(bound.Bound.At), displayTypeName(resolved, module), bound.Name, displayTypeName(resolved, module))
			}
			return fmt.Errorf("%s: unknown protocol bound '%s' for generic parameter '%s'", frontend.FormatPos(bound.Bound.At), displayTypeName(resolved, module), bound.Name)
		}
		if !symbolBelongsToModule(resolved, module) && !proto.public {
			return fmt.Errorf("%s: private protocol '%s' is not visible from module '%s'", frontend.FormatPos(bound.Bound.At), resolved, module)
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
	if err := validateGenericBoundRequirementCalls(fn.Body, boundParams); err != nil {
		return err
	}
	return nil
}

func validateGenericBoundRequirementCalls(stmts []frontend.Stmt, boundParams map[string]struct{}) error {
	if len(boundParams) == 0 {
		return nil
	}
	return walkGenericBoundRequirementCallsInStmts(stmts, boundParams)
}

func walkGenericBoundRequirementCallsInStmts(stmts []frontend.Stmt, boundParams map[string]struct{}) error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.ReturnStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Value, boundParams); err != nil {
				return err
			}
		case *frontend.ThrowStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Value, boundParams); err != nil {
				return err
			}
		case *frontend.DeferStmt:
			if err := walkGenericBoundRequirementCallsInStmts(s.Body, boundParams); err != nil {
				return err
			}
		case *frontend.PrintStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Value, boundParams); err != nil {
				return err
			}
		case *frontend.ExpectStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Cond, boundParams); err != nil {
				return err
			}
		case *frontend.FreeStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Value, boundParams); err != nil {
				return err
			}
		case *frontend.LetStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Value, boundParams); err != nil {
				return err
			}
		case *frontend.AssignStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Target, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInExpr(s.Value, boundParams); err != nil {
				return err
			}
		case *frontend.IfStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Cond, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInStmts(s.Then, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInStmts(s.Else, boundParams); err != nil {
				return err
			}
		case *frontend.IfLetStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Pattern, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInExpr(s.Value, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInStmts(s.Then, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInStmts(s.Else, boundParams); err != nil {
				return err
			}
		case *frontend.WhileStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Cond, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInStmts(s.Body, boundParams); err != nil {
				return err
			}
		case *frontend.ForRangeStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Start, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInExpr(s.End, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInExpr(s.Iterable, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInStmts(s.Body, boundParams); err != nil {
				return err
			}
		case *frontend.MatchStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Value, boundParams); err != nil {
				return err
			}
			for i := range s.Cases {
				if err := walkGenericBoundRequirementCallsInExpr(s.Cases[i].Pattern, boundParams); err != nil {
					return err
				}
				if err := walkGenericBoundRequirementCallsInExpr(s.Cases[i].Guard, boundParams); err != nil {
					return err
				}
				if err := walkGenericBoundRequirementCallsInStmts(s.Cases[i].Body, boundParams); err != nil {
					return err
				}
			}
		case *frontend.UnsafeStmt:
			if err := walkGenericBoundRequirementCallsInStmts(s.Body, boundParams); err != nil {
				return err
			}
		case *frontend.IslandStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Size, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInStmts(s.Body, boundParams); err != nil {
				return err
			}
		case *frontend.ExprStmt:
			if err := walkGenericBoundRequirementCallsInExpr(s.Expr, boundParams); err != nil {
				return err
			}
		}
	}
	return nil
}

func walkGenericBoundRequirementCallsInExpr(expr frontend.Expr, boundParams map[string]struct{}) error {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *frontend.CallExpr:
		if parts := strings.Split(e.Name, "."); len(parts) == 2 {
			if _, ok := boundParams[parts[0]]; ok {
				return fmt.Errorf("%s: calling protocol requirement '%s' through generic bound '%s' is not supported in this MVP; specialize the operation outside the generic", frontend.FormatPos(e.At), parts[1], parts[0])
			}
		}
		for _, arg := range e.Args {
			if err := walkGenericBoundRequirementCallsInExpr(arg, boundParams); err != nil {
				return err
			}
		}
	case *frontend.MatchExpr:
		if err := walkGenericBoundRequirementCallsInExpr(e.Value, boundParams); err != nil {
			return err
		}
		for i := range e.Cases {
			if err := walkGenericBoundRequirementCallsInExpr(e.Cases[i].Pattern, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInExpr(e.Cases[i].Guard, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInExpr(e.Cases[i].Value, boundParams); err != nil {
				return err
			}
		}
	case *frontend.CatchExpr:
		if err := walkGenericBoundRequirementCallsInExpr(e.Call, boundParams); err != nil {
			return err
		}
		for i := range e.Cases {
			if err := walkGenericBoundRequirementCallsInExpr(e.Cases[i].Pattern, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInExpr(e.Cases[i].Guard, boundParams); err != nil {
				return err
			}
			if err := walkGenericBoundRequirementCallsInExpr(e.Cases[i].Value, boundParams); err != nil {
				return err
			}
		}
	case *frontend.UnaryExpr:
		return walkGenericBoundRequirementCallsInExpr(e.X, boundParams)
	case *frontend.BinaryExpr:
		if err := walkGenericBoundRequirementCallsInExpr(e.Left, boundParams); err != nil {
			return err
		}
		return walkGenericBoundRequirementCallsInExpr(e.Right, boundParams)
	case *frontend.FieldAccessExpr:
		return walkGenericBoundRequirementCallsInExpr(e.Base, boundParams)
	case *frontend.IndexExpr:
		if err := walkGenericBoundRequirementCallsInExpr(e.Base, boundParams); err != nil {
			return err
		}
		return walkGenericBoundRequirementCallsInExpr(e.Index, boundParams)
	case *frontend.TryExpr:
		return walkGenericBoundRequirementCallsInExpr(e.X, boundParams)
	case *frontend.AwaitExpr:
		return walkGenericBoundRequirementCallsInExpr(e.X, boundParams)
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if err := walkGenericBoundRequirementCallsInExpr(field.Value, boundParams); err != nil {
				return err
			}
		}
	case *frontend.ClosureExpr:
		if e.Decl != nil {
			return walkGenericBoundRequirementCallsInStmts(e.Decl.Body, boundParams)
		}
	}
	return nil
}

func validateSemanticClauses(fn *frontend.FuncDecl) error {
	return semanticspolicy.ValidateSemanticClauses(fn, constI32, privacyDiagnosticf, budgetDiagnosticf)
}

func validateBudgetContexts(world *module.World, funcs map[string]FuncSig) error {
	if world == nil {
		return nil
	}
	for _, file := range world.Files {
		if file == nil || world.InterfaceModules[file.Module] {
			continue
		}
		imports, err := collectImportAliases(file)
		if err != nil {
			return err
		}
		for _, fn := range file.Funcs {
			if fn == nil || len(fn.TypeParams) > 0 {
				continue
			}
			callerName := checkedFuncFullName(file.Module, fn)
			callerSig, ok := funcs[callerName]
			if !ok {
				continue
			}
			if err := validateBudgetContextsInStmts(fn.Body, callerName, callerSig, funcs, file.Module, imports); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateBudgetContextsInStmts(stmts []frontend.Stmt, callerName string, callerSig FuncSig, funcs map[string]FuncSig, module string, imports map[string]string) error {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *frontend.LetStmt:
			if err := validateBudgetContextsInExpr(s.Value, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
		case *frontend.AssignStmt:
			if err := validateBudgetContextsInExpr(s.Target, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
			if err := validateBudgetContextsInExpr(s.Value, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
		case *frontend.ExprStmt:
			if err := validateBudgetContextsInExpr(s.Expr, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
		case *frontend.ReturnStmt:
			if err := validateBudgetContextsInExpr(s.Value, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
		case *frontend.ThrowStmt:
			if err := validateBudgetContextsInExpr(s.Value, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
		case *frontend.PrintStmt:
			if err := validateBudgetContextsInExpr(s.Value, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
		case *frontend.ExpectStmt:
			if err := validateBudgetContextsInExpr(s.Cond, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
		case *frontend.FreeStmt:
			if err := validateBudgetContextsInExpr(s.Value, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
		case *frontend.IfStmt:
			if err := validateBudgetContextsInExpr(s.Cond, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
			if err := validateBudgetContextsInStmts(s.Then, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
			if err := validateBudgetContextsInStmts(s.Else, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
		case *frontend.IfLetStmt:
			if err := validateBudgetContextsInExpr(s.Value, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
			if err := validateBudgetContextsInStmts(s.Then, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
			if err := validateBudgetContextsInStmts(s.Else, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
		case *frontend.WhileStmt:
			if err := validateBudgetContextsInExpr(s.Cond, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
			if err := validateBudgetContextsInStmts(s.Body, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
		case *frontend.ForRangeStmt:
			if s.Iterable != nil {
				if err := validateBudgetContextsInExpr(s.Iterable, callerName, callerSig, funcs, module, imports); err != nil {
					return err
				}
			} else {
				if err := validateBudgetContextsInExpr(s.Start, callerName, callerSig, funcs, module, imports); err != nil {
					return err
				}
				if err := validateBudgetContextsInExpr(s.End, callerName, callerSig, funcs, module, imports); err != nil {
					return err
				}
			}
			if err := validateBudgetContextsInStmts(s.Body, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
		case *frontend.MatchStmt:
			if err := validateBudgetContextsInExpr(s.Value, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
			for _, c := range s.Cases {
				if !c.Default {
					if err := validateBudgetContextsInExpr(c.Pattern, callerName, callerSig, funcs, module, imports); err != nil {
						return err
					}
				}
				if err := validateBudgetContextsInExpr(c.Guard, callerName, callerSig, funcs, module, imports); err != nil {
					return err
				}
				if err := validateBudgetContextsInStmts(c.Body, callerName, callerSig, funcs, module, imports); err != nil {
					return err
				}
			}
		case *frontend.UnsafeStmt:
			if err := validateBudgetContextsInStmts(s.Body, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
		case *frontend.DeferStmt:
			if err := validateBudgetContextsInStmts(s.Body, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
		case *frontend.IslandStmt:
			if err := validateBudgetContextsInExpr(s.Size, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
			if err := validateBudgetContextsInStmts(s.Body, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateBudgetContextsInExpr(expr frontend.Expr, callerName string, callerSig FuncSig, funcs map[string]FuncSig, module string, imports map[string]string) error {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *frontend.CallExpr:
		resolved := e.Name
		if builtin, ok := ResolveBuiltinAlias(resolved); ok {
			resolved = builtin
		}
		if err := validateBudgetSpawnContext(e, resolved, callerName, callerSig, funcs, module, imports); err != nil {
			return err
		}
		if targetSig, ok := funcs[resolved]; ok {
			if err := validateBudgetContextEdge(e.At, callerName, callerSig, "call to '"+resolved+"'", targetSig); err != nil {
				return err
			}
		}
		for _, arg := range e.Args {
			if err := validateBudgetContextsInExpr(arg, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
		}
	case *frontend.FieldAccessExpr:
		return validateBudgetContextsInExpr(e.Base, callerName, callerSig, funcs, module, imports)
	case *frontend.IndexExpr:
		if err := validateBudgetContextsInExpr(e.Base, callerName, callerSig, funcs, module, imports); err != nil {
			return err
		}
		return validateBudgetContextsInExpr(e.Index, callerName, callerSig, funcs, module, imports)
	case *frontend.BinaryExpr:
		if err := validateBudgetContextsInExpr(e.Left, callerName, callerSig, funcs, module, imports); err != nil {
			return err
		}
		return validateBudgetContextsInExpr(e.Right, callerName, callerSig, funcs, module, imports)
	case *frontend.UnaryExpr:
		return validateBudgetContextsInExpr(e.X, callerName, callerSig, funcs, module, imports)
	case *frontend.TryExpr:
		return validateBudgetContextsInExpr(e.X, callerName, callerSig, funcs, module, imports)
	case *frontend.AwaitExpr:
		return validateBudgetContextsInExpr(e.X, callerName, callerSig, funcs, module, imports)
	case *frontend.StructLitExpr:
		for _, field := range e.Fields {
			if err := validateBudgetContextsInExpr(field.Value, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
		}
	case *frontend.MatchExpr:
		if err := validateBudgetContextsInExpr(e.Value, callerName, callerSig, funcs, module, imports); err != nil {
			return err
		}
		for _, c := range e.Cases {
			if err := validateBudgetContextsInExpr(c.Pattern, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
			if err := validateBudgetContextsInExpr(c.Guard, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
			if err := validateBudgetContextsInExpr(c.Value, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
		}
	case *frontend.CatchExpr:
		if err := validateBudgetContextsInExpr(e.Call, callerName, callerSig, funcs, module, imports); err != nil {
			return err
		}
		for _, c := range e.Cases {
			if err := validateBudgetContextsInExpr(c.Pattern, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
			if err := validateBudgetContextsInExpr(c.Guard, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
			if err := validateBudgetContextsInExpr(c.Value, callerName, callerSig, funcs, module, imports); err != nil {
				return err
			}
		}
	case *frontend.ClosureExpr:
		// Closure declarations are validated as synthetic functions in file.Funcs.
		// Re-validating here under the outer caller budget creates false positives
		// for closures that declare their own budget context.
		return nil
	}
	return nil
}

func validateBudgetSpawnContext(call *frontend.CallExpr, resolved string, callerName string, callerSig FuncSig, funcs map[string]FuncSig, module string, imports map[string]string) error {
	workerArg := -1
	contextName := ""
	switch resolved {
	case "core.spawn":
		workerArg = 0
		contextName = "spawn target"
	case "core.spawn_remote":
		workerArg = 1
		contextName = "spawn_remote target"
	case "core.task_spawn_i32", "core.task_spawn_i32_typed":
		workerArg = 0
		contextName = strings.TrimPrefix(resolved, "core.") + " target"
	case "core.task_spawn_group_i32", "core.task_spawn_group_i32_typed":
		workerArg = 1
		contextName = strings.TrimPrefix(resolved, "core.") + " target"
	}
	if workerArg < 0 || workerArg >= len(call.Args) {
		return nil
	}
	lit, ok := call.Args[workerArg].(*frontend.StringLitExpr)
	if !ok || len(lit.Value) == 0 {
		return nil
	}
	target, err := resolveKnownCallName(string(lit.Value), funcs, module, imports, call.At)
	if err != nil {
		return err
	}
	targetSig, ok := funcs[target]
	if !ok {
		return nil
	}
	return validateBudgetContextEdge(call.At, callerName, callerSig, contextName+" '"+target+"'", targetSig)
}

func validateBudgetContextEdge(pos frontend.Position, callerName string, callerSig FuncSig, context string, targetSig FuncSig) error {
	if !targetSig.HasBudget {
		return nil
	}
	required := targetSig.Budget
	if !callerSig.HasBudget {
		return budgetDiagnosticf(pos, "budget context for %s requires caller '%s' to declare budget at least %d", context, callerName, required)
	}
	if callerSig.Budget < required {
		return budgetDiagnosticf(pos, "budget context for %s requires caller budget at least %d, got %d", context, required, callerSig.Budget)
	}
	return nil
}

type functionClausePolicy struct {
	hasNoAlloc   bool
	hasNoBlock   bool
	hasRealtime  bool
	hasBudget    bool
	budget       int32
	hasPrivacy   bool
	consentParam string
}

func parseFunctionClausePolicy(fn *frontend.FuncDecl) (functionClausePolicy, error) {
	policy, err := semanticspolicy.ParseFunctionClausePolicy(fn, constI32, privacyDiagnosticf)
	if err != nil {
		return functionClausePolicy{}, err
	}
	return functionClausePolicy{
		hasNoAlloc:   policy.HasNoAlloc,
		hasNoBlock:   policy.HasNoBlock,
		hasRealtime:  policy.HasRealtime,
		hasBudget:    policy.HasBudget,
		budget:       policy.Budget,
		hasPrivacy:   policy.HasPrivacy,
		consentParam: policy.ConsentParam,
	}, nil
}

func validateFunctionPolicyClauses(
	fn *frontend.FuncDecl,
	effects []string,
	paramTypes map[string]string,
	returnType string,
	throwsType string,
	types map[string]*TypeInfo,
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
		return budgetDiagnosticf(fn.Pos, "uses effect 'budget' requires semantic clause 'budget'")
	}
	if policy.hasBudget && !hasEffect("budget") {
		return budgetDiagnosticf(fn.Pos, "semantic clause 'budget' requires function '%s' to declare uses effect 'budget'", fn.Name)
	}
	if policy.hasNoAlloc && hasEffect("alloc") {
		return effectDiagnosticf(fn.Pos, "semantic clause 'noalloc' conflicts with declared effect 'alloc'")
	}
	if policy.hasNoBlock {
		if blocked := firstForbiddenEffect(declaredEffects, []string{"actors", "control", "io", "link", "mmio", "runtime"}); blocked != "" {
			return effectDiagnosticf(fn.Pos, "semantic clause 'noblock' conflicts with declared effect '%s'", blocked)
		}
	}
	if policy.hasRealtime {
		if !policy.hasNoAlloc {
			return effectDiagnosticf(fn.Pos, "semantic clause 'realtime' requires semantic clause 'noalloc'")
		}
		if !policy.hasNoBlock {
			return effectDiagnosticf(fn.Pos, "semantic clause 'realtime' requires semantic clause 'noblock'")
		}
		if blocked := firstForbiddenEffect(declaredEffects, []string{"actors", "alloc", "control", "io", "link", "mmio", "runtime"}); blocked != "" {
			return effectDiagnosticf(fn.Pos, "semantic clause 'realtime' conflicts with declared effect '%s'", blocked)
		}
	}
	if policy.hasPrivacy && !hasEffect("privacy") {
		return privacyDiagnosticf(fn.Pos, "semantic clause 'privacy' requires function '%s' to declare uses effect 'privacy'", fn.Name)
	}
	if hasEffect("privacy") && !policy.hasPrivacy {
		return privacyDiagnosticf(fn.Pos, "uses effect 'privacy' requires semantic clause 'privacy'")
	}

	signatureHasSecret := typeUsesSecret(returnType, types) || typeUsesSecret(throwsType, types)
	for _, paramType := range paramTypes {
		if typeUsesSecret(paramType, types) {
			signatureHasSecret = true
		}
	}
	if functionDeclSignatureUsesSecret(fn, types) {
		signatureHasSecret = true
	}
	if signatureHasSecret && !policy.hasPrivacy {
		return privacyDiagnosticf(fn.Pos, "secret types in function signature require semantic clause 'privacy'")
	}
	if signatureHasSecret && policy.consentParam == "" {
		return privacyDiagnosticf(fn.Pos, "secret types in function signature require semantic clause consent(<token>)")
	}
	if policy.consentParam != "" {
		if !policy.hasPrivacy {
			return privacyDiagnosticf(fn.Pos, "semantic clause 'consent' requires semantic clause 'privacy'")
		}
		paramType, ok := paramTypes[policy.consentParam]
		if !ok {
			return privacyDiagnosticf(fn.Pos, "semantic clause 'consent' references unknown parameter '%s'", policy.consentParam)
		}
		if paramType != "consent.token" {
			return privacyDiagnosticf(fn.Pos, "semantic clause 'consent' parameter '%s' must have type consent.token", policy.consentParam)
		}
	}
	return nil
}

func validateExportedOpaqueABISignature(module string, fn *frontend.FuncDecl, paramTypes map[string]string, returnType string, types map[string]*TypeInfo) error {
	if fn == nil || fn.ExportName == "" {
		return nil
	}
	allowRuntimeHandles := isInternalRuntimeABIExport(module, fn)
	for _, param := range fn.Params {
		paramType := paramTypes[param.Name]
		if param.Ownership != "" {
			return effectDiagnosticf(
				param.At,
				"exported function '%s' cannot expose ownership marker '%s' on parameter '%s'; export a plain FFI-safe wrapper",
				fn.Name,
				param.Ownership,
				param.Name,
			)
		}
		if isOpaqueCapabilityTokenType(paramType) {
			return effectDiagnosticf(
				param.At,
				"exported function '%s' cannot expose opaque capability token '%s' in parameter '%s'",
				fn.Name,
				paramType,
				param.Name,
			)
		}
		if isOpaqueIslandHandleType(paramType) {
			return effectDiagnosticf(
				param.At,
				"exported function '%s' cannot expose opaque island handle '%s' in parameter '%s'",
				fn.Name,
				paramType,
				param.Name,
			)
		}
		if isFunctionTypedABIValueType(paramType) {
			return effectDiagnosticf(
				param.At,
				"exported function '%s' cannot expose function-typed value '%s' in parameter '%s'",
				fn.Name,
				paramType,
				param.Name,
			)
		}
		if !allowRuntimeHandles {
			if exposure, ok := exportedBoolABIExposureForType(paramType, types); ok {
				return effectDiagnosticf(
					param.At,
					"exported function '%s' cannot expose %s '%s' in parameter '%s'",
					fn.Name,
					exposure.Kind,
					exposure.TypeName,
					param.Name,
				)
			}
		}
		if exposure, ok := exportedRawViewABIExposureForType(paramType, types); ok {
			return effectDiagnosticf(
				param.At,
				"exported function '%s' cannot expose %s '%s' in parameter '%s'",
				fn.Name,
				exposure.Kind,
				exposure.TypeName,
				param.Name,
			)
		}
		if !allowRuntimeHandles && isOpaqueRuntimeHandleType(paramType) {
			return effectDiagnosticf(
				param.At,
				"exported function '%s' cannot expose opaque runtime handle '%s' in parameter '%s'",
				fn.Name,
				paramType,
				param.Name,
			)
		}
		if exposure, ok := exportedOpaqueABIExposureForType(paramType, types, allowRuntimeHandles); ok {
			return effectDiagnosticf(
				param.At,
				"exported function '%s' cannot expose %s '%s' through parameter '%s' type '%s'",
				fn.Name,
				exposure.Kind,
				exposure.TypeName,
				param.Name,
				paramType,
			)
		}
	}
	if isOpaqueCapabilityTokenType(returnType) {
		return effectDiagnosticf(
			fn.ReturnType.At,
			"exported function '%s' cannot expose opaque capability token '%s' in return type",
			fn.Name,
			returnType,
		)
	}
	if isOpaqueIslandHandleType(returnType) {
		return effectDiagnosticf(
			fn.ReturnType.At,
			"exported function '%s' cannot expose opaque island handle '%s' in return type",
			fn.Name,
			returnType,
		)
	}
	if isFunctionTypedABIValueType(returnType) {
		return effectDiagnosticf(
			fn.ReturnType.At,
			"exported function '%s' cannot expose function-typed value '%s' in return type",
			fn.Name,
			returnType,
		)
	}
	if !allowRuntimeHandles {
		if exposure, ok := exportedBoolABIExposureForType(returnType, types); ok {
			return effectDiagnosticf(
				fn.ReturnType.At,
				"exported function '%s' cannot expose %s '%s' in return type",
				fn.Name,
				exposure.Kind,
				exposure.TypeName,
			)
		}
	}
	if exposure, ok := exportedRawViewABIExposureForType(returnType, types); ok {
		return effectDiagnosticf(
			fn.ReturnType.At,
			"exported function '%s' cannot expose %s '%s' in return type",
			fn.Name,
			exposure.Kind,
			exposure.TypeName,
		)
	}
	if !allowRuntimeHandles && isOpaqueRuntimeHandleType(returnType) {
		return effectDiagnosticf(
			fn.ReturnType.At,
			"exported function '%s' cannot expose opaque runtime handle '%s' in return type",
			fn.Name,
			returnType,
		)
	}
	if exposure, ok := exportedOpaqueABIExposureForType(returnType, types, allowRuntimeHandles); ok {
		return effectDiagnosticf(
			fn.ReturnType.At,
			"exported function '%s' cannot expose %s '%s' through return type '%s'",
			fn.Name,
			exposure.Kind,
			exposure.TypeName,
			returnType,
		)
	}
	for _, param := range fn.Params {
		paramType := paramTypes[param.Name]
		if exposure, ok := exportedDefaultStructABIExposureForType(paramType, types); ok {
			return effectDiagnosticf(
				param.At,
				"exported function '%s' parameter '%s' type '%s' requires explicit repr(C); default Tetra layout is compiler-owned and has no public ABI",
				fn.Name,
				param.Name,
				exposure.TypeName,
			)
		}
	}
	if exposure, ok := exportedDefaultStructABIExposureForType(returnType, types); ok {
		return effectDiagnosticf(
			fn.ReturnType.At,
			"exported function '%s' return type '%s' requires explicit repr(C); default Tetra layout is compiler-owned and has no public ABI",
			fn.Name,
			exposure.TypeName,
		)
	}
	return nil
}
