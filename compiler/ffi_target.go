package compiler

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/abisuite"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/semantics"
)

func validateTargetExportedFFIAST(world *World, target string) error {
	if world == nil || !targetRequiresExplicitPointerExportGate(target) {
		return nil
	}
	for _, file := range world.Files {
		if file == nil {
			continue
		}
		for _, fn := range file.Funcs {
			if err := validateTargetExportedFFIDeclAST(fn, file.Module, target); err != nil {
				return err
			}
		}
		for _, actor := range file.Actors {
			if actor == nil {
				continue
			}
			for _, method := range actor.Methods {
				if err := validateTargetExportedFFIDeclAST(method, file.Module, target); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateTargetExportedFFIDeclAST(fn *frontend.FuncDecl, module string, target string) error {
	if fn == nil || fn.ExportName == "" || isInternalRuntimeExportedSymbol(module, fn.ExportName) {
		return nil
	}
	for _, param := range fn.Params {
		typeName := targetExportedFFITypeRefName(param.Type)
		if targetExportedFFIRequiresPointerBoundaryGate(target, typeName) {
			return targetExportedFFIPointerParamDiagnostic(param.At, target, fn.Name, param.Name, typeName)
		}
	}
	typeName := targetExportedFFITypeRefName(fn.ReturnType)
	if targetExportedFFIRequiresPointerBoundaryGate(target, typeName) {
		pos := fn.ReturnType.At
		if pos.Line == 0 || pos.Col == 0 {
			pos = fn.Pos
		}
		return targetExportedFFIPointerReturnDiagnostic(pos, target, fn.Name, typeName)
	}
	return nil
}

func targetExportedFFITypeRefName(ref frontend.TypeRef) string {
	switch ref.Kind {
	case frontend.TypeRefFunction:
		return "fnptr"
	case frontend.TypeRefNamed:
		return strings.TrimSpace(ref.Name)
	default:
		return strings.TrimSpace(ref.Name)
	}
}

func validateTargetExportedFFIABI(checked *semantics.CheckedProgram, target string) error {
	if checked == nil || !targetRequiresExplicitAggregateExportGate(target) {
		return nil
	}
	for _, fn := range checked.Funcs {
		decl := fn.Decl
		if decl == nil || decl.ExportName == "" || isInternalRuntimeExportedSymbol(fn.Module, decl.ExportName) {
			continue
		}
		sig, ok := checked.FuncSigs[fn.Name]
		if !ok {
			continue
		}
		for i, typeName := range sig.ParamTypes {
			if targetExportedFFIRequiresPointerBoundaryGate(target, typeName) {
				pos := decl.Pos
				paramName := fmt.Sprintf("#%d", i+1)
				if i < len(decl.Params) {
					pos = decl.Params[i].At
					paramName = decl.Params[i].Name
				}
				return targetExportedFFIPointerParamDiagnostic(pos, target, decl.Name, paramName, typeName)
			}
			if !targetExportedFFIRequiresAggregateABI(typeName, checked.Types) {
				continue
			}
			pos := decl.Pos
			paramName := fmt.Sprintf("#%d", i+1)
			if i < len(decl.Params) {
				pos = decl.Params[i].At
				paramName = decl.Params[i].Name
			}
			return targetExportedFFIAggregateParamDiagnostic(pos, target, decl.Name, paramName, typeName)
		}
		if targetExportedFFIRequiresPointerBoundaryGate(target, sig.ReturnType) {
			pos := decl.ReturnType.At
			if pos.Line == 0 || pos.Col == 0 {
				pos = decl.Pos
			}
			return targetExportedFFIPointerReturnDiagnostic(pos, target, decl.Name, sig.ReturnType)
		}
		if targetExportedFFIRequiresAggregateABI(sig.ReturnType, checked.Types) {
			pos := decl.ReturnType.At
			if pos.Line == 0 || pos.Col == 0 {
				pos = decl.Pos
			}
			return targetExportedFFIAggregateReturnDiagnostic(pos, target, decl.Name, sig.ReturnType)
		}
	}
	return nil
}

func targetRequiresExplicitAggregateExportGate(target string) bool {
	return abisuite.TargetRequiresExplicitAggregateExportGate(target)
}

func targetRequiresExplicitPointerExportGate(target string) bool {
	return abisuite.TargetRequiresExplicitPointerExportGate(target)
}

func targetExportedFFIRequiresX32PointerBoundaryGate(target, typeName string) bool {
	return abisuite.TargetExportedFFIRequiresX32PointerBoundaryGate(target, typeName)
}

func targetExportedFFIRequiresPointerBoundaryGate(target, typeName string) bool {
	return abisuite.TargetExportedFFIRequiresPointerBoundaryGate(target, typeName)
}

func translateTargetExportedFFISemanticError(err error, target string) error {
	if err == nil || !targetRequiresExplicitPointerExportGate(target) {
		return err
	}
	diag := DiagnosticFromError(err)
	if !strings.Contains(diag.Message, "cannot expose function-typed value 'fnptr'") {
		return err
	}
	fnName := quotedAfter(diag.Message, "exported function '")
	if fnName == "" {
		return err
	}
	pos := frontend.Position{File: diag.File, Line: diag.Line, Col: diag.Column}
	if strings.Contains(diag.Message, " in parameter '") {
		paramName := quotedAfter(diag.Message, " in parameter '")
		if paramName == "" {
			return err
		}
		return targetExportedFFIPointerParamDiagnostic(pos, target, fnName, paramName, "fnptr")
	}
	if strings.Contains(diag.Message, " in return type") {
		return targetExportedFFIPointerReturnDiagnostic(pos, target, fnName, "fnptr")
	}
	return err
}

func quotedAfter(s, prefix string) string {
	start := strings.Index(s, prefix)
	if start < 0 {
		return ""
	}
	rest := s[start+len(prefix):]
	end := strings.Index(rest, "'")
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func targetExportedFFIRequiresAggregateABI(typeName string, types map[string]*semantics.TypeInfo) bool {
	return abisuite.TargetExportedFFIRequiresAggregateABI(typeName, types)
}

func isInternalRuntimeExportedSymbol(module, exportName string) bool {
	return strings.HasPrefix(exportName, "__tetra_") && (module == "__rt" || strings.HasPrefix(module, "__rt."))
}

func targetExportedFFIAggregateParamDiagnostic(pos frontend.Position, target, fnName, paramName, typeName string) error {
	return &frontend.DiagnosticError{Info: frontend.Diagnostic{
		Code:     DiagnosticCodeTargetRuntime,
		Message:  fmt.Sprintf("exported function '%s' parameter '%s' type '%s' requires aggregate C ABI; aggregate C ABI is not supported on %s", fnName, paramName, typeName, target),
		File:     pos.File,
		Line:     pos.Line,
		Column:   pos.Col,
		Severity: "error",
		Hint:     "Export a scalar FFI wrapper for this target, or keep the aggregate behind a target-specific runtime object with a verified C ABI.",
	}}
}

func targetExportedFFIPointerParamDiagnostic(pos frontend.Position, target, fnName, paramName, typeName string) error {
	boundary := targetPointerCBoundaryName(target)
	return &frontend.DiagnosticError{Info: frontend.Diagnostic{
		Code:     DiagnosticCodeTargetRuntime,
		Message:  fmt.Sprintf("exported function '%s' parameter '%s' type '%s' requires the %s pointer C ABI boundary; %s pointer C ABI boundary is not verified on %s", fnName, paramName, typeName, boundary, boundary, target),
		File:     pos.File,
		Line:     pos.Line,
		Column:   pos.Col,
		Severity: "error",
		Hint:     fmt.Sprintf("Export an i32 handle wrapper for %s, or keep the pointer boundary inside a verified target-specific runtime object.", target),
	}}
}

func targetExportedFFIPointerReturnDiagnostic(pos frontend.Position, target, fnName, typeName string) error {
	boundary := targetPointerCBoundaryName(target)
	return &frontend.DiagnosticError{Info: frontend.Diagnostic{
		Code:     DiagnosticCodeTargetRuntime,
		Message:  fmt.Sprintf("exported function '%s' return type '%s' requires the %s pointer C ABI boundary; %s pointer C ABI boundary is not verified on %s", fnName, typeName, boundary, boundary, target),
		File:     pos.File,
		Line:     pos.Line,
		Column:   pos.Col,
		Severity: "error",
		Hint:     fmt.Sprintf("Export an i32 handle wrapper for %s, or keep the pointer boundary inside a verified target-specific runtime object.", target),
	}}
}

func targetPointerCBoundaryName(target string) string {
	switch target {
	case "linux-x86":
		return "i386"
	case "linux-x32":
		return "x32"
	default:
		return target
	}
}

func targetExportedFFIAggregateReturnDiagnostic(pos frontend.Position, target, fnName, typeName string) error {
	return &frontend.DiagnosticError{Info: frontend.Diagnostic{
		Code:     DiagnosticCodeTargetRuntime,
		Message:  fmt.Sprintf("exported function '%s' return type '%s' requires aggregate C ABI; aggregate C ABI is not supported on %s", fnName, typeName, target),
		File:     pos.File,
		Line:     pos.Line,
		Column:   pos.Col,
		Severity: "error",
		Hint:     "Export a scalar FFI wrapper for this target, or keep the aggregate behind a target-specific runtime object with a verified C ABI.",
	}}
}
