package functiontypes

import (
	"tetra_language/compiler/internal/frontend"
	semanticspolicy "tetra_language/compiler/internal/semantics/policy"
)

func ParamOwnership(ref frontend.TypeRef) []string {
	if ref.Kind != frontend.TypeRefFunction {
		return nil
	}
	out := make([]string, len(ref.Params))
	copy(out, ref.ParamOwnership)
	return out
}

func ReturnOwnership(ref frontend.TypeRef) string {
	if ref.Kind != frontend.TypeRefFunction {
		return ""
	}
	return ref.ReturnOwnership
}

func Effects(
	ref frontend.TypeRef,
	pos frontend.Position,
	diagnostic semanticspolicy.DiagnosticFunc,
) ([]string, error) {
	if ref.Kind != frontend.TypeRefFunction {
		return nil, nil
	}
	return semanticspolicy.NormalizeEffects(ref.Uses, pos, diagnostic)
}
