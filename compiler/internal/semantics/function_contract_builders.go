package semantics

import (
	"fmt"
	"sort"

	"tetra_language/compiler/internal/semantics/model"
)

type funcSigSpec FuncSig

func buildBuiltinFuncSig(name string, spec funcSigSpec, types map[string]*TypeInfo) (FuncSig, error) {
	return finalizeFuncSigContract("builtin", name, spec, types)
}

func buildDeclaredFuncSig(name string, spec funcSigSpec, types map[string]*TypeInfo) (FuncSig, error) {
	return finalizeFuncSigContract("declared", name, spec, types)
}

func buildGenericFuncSig(name string, spec funcSigSpec, types map[string]*TypeInfo) (FuncSig, error) {
	return finalizeFuncSigContract("generic", name, spec, types)
}

func buildMonomorphizedFuncSig(name string, spec funcSigSpec, types map[string]*TypeInfo) (FuncSig, error) {
	return finalizeFuncSigContract("monomorphized", name, spec, types)
}

func buildInterfaceFuncSig(name string, spec funcSigSpec, types map[string]*TypeInfo) (FuncSig, error) {
	return finalizeFuncSigContract("interface", name, spec, types)
}

func finalizeFuncSigContract(kind string, name string, spec funcSigSpec, types map[string]*TypeInfo) (FuncSig, error) {
	sig := model.CloneFuncSig(FuncSig(spec))
	completeFuncSigParamOwnership(&sig)
	canonicalizeFuncSigEffects(&sig)
	if err := model.ValidateFuncSigContract(name, sig, types); err != nil {
		return FuncSig{}, fmt.Errorf("%s FuncSig %q: %w", kind, name, err)
	}
	return sig, nil
}

func completeFuncSigParamOwnership(sig *FuncSig) {
	if sig == nil || len(sig.ParamOwnership) == 0 || len(sig.ParamOwnership) >= len(sig.ParamTypes) {
		return
	}
	for len(sig.ParamOwnership) < len(sig.ParamTypes) {
		sig.ParamOwnership = append(sig.ParamOwnership, "")
	}
}

func canonicalizeFuncSigEffects(sig *FuncSig) {
	if sig == nil {
		return
	}
	sort.Strings(sig.Effects)
	for i := range sig.ParamFunctionEffects {
		sort.Strings(sig.ParamFunctionEffects[i])
	}
	sort.Strings(sig.ReturnFunctionEffects)
	canonicalizeFunctionFieldMapEffects(sig.ReturnFunctionFields)
	canonicalizeFunctionFieldMapEffects(sig.ReturnEnumPayloadFunctions)
	canonicalizeFunctionFieldMapEffects(sig.ReturnEnumPayloadFields)
}

func canonicalizeFunctionFieldMapEffects(fields map[string]FunctionFieldInfo) {
	for key, field := range fields {
		sort.Strings(field.FunctionEffects)
		fields[key] = field
	}
}
