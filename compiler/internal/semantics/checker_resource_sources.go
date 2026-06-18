package semantics

import (
	"fmt"

	"tetra_language/compiler/internal/frontend"
)

func resourceSourceForCallProvenance(
	args []frontend.Expr,
	sig FuncSig,
	provenance ResourceProvenance,
	funcs map[string]FuncSig,
	types map[string]*TypeInfo,
	module string,
	imports map[string]string,
	state *regionState,
	pos frontend.Position,
) (resourceSourceResult, error) {
	if provenance.ParamIndex < 0 || provenance.ParamIndex >= len(args) || provenance.ParamIndex >= len(sig.ParamTypes) {
		return resourceSourceResult{}, fmt.Errorf("%s: invalid resource signature", frontend.FormatPos(pos))
	}
	if provenance.ParamPath == "" {
		return resourceSourceForExpr(args[provenance.ParamIndex], funcs, module, imports, state)
	}
	return resourceSourceForExprLeaf(args[provenance.ParamIndex], sig.ParamTypes[provenance.ParamIndex], provenance.ParamPath, funcs, types, module, imports, state)
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
