package semantics

import (
	"fmt"

	"tetra_language/compiler/internal/frontend"
)

const (
	DiagnosticCodeSafetyOwnership = "TETRA2101"
	DiagnosticCodeSafetyLifetime  = "TETRA2102"
	DiagnosticCodeSafetyEffect    = "TETRA2103"
	DiagnosticCodeSafetyPrivacy   = "TETRA2104"
	DiagnosticCodeSafetyBudget    = "TETRA2105"
)

type diagnosticError struct {
	code    string
	pos     frontend.Position
	message string
	hint    string
}

func (e *diagnosticError) Error() string {
	if e.pos.Line > 0 && e.pos.Col > 0 {
		return frontend.FormatPos(e.pos) + ": " + e.message
	}
	return e.message
}

func (e *diagnosticError) Diagnostic() frontend.Diagnostic {
	return frontend.Diagnostic{
		Code:     e.code,
		Message:  e.message,
		File:     e.pos.File,
		Line:     e.pos.Line,
		Column:   e.pos.Col,
		Severity: "error",
		Hint:     e.hint,
	}
}

func ownershipDiagnosticf(pos frontend.Position, format string, args ...interface{}) error {
	return safetyDiagnosticf(pos, DiagnosticCodeSafetyOwnership, format, args...)
}

func lifetimeDiagnosticf(pos frontend.Position, format string, args ...interface{}) error {
	return safetyDiagnosticf(pos, DiagnosticCodeSafetyLifetime, format, args...)
}

func effectDiagnosticf(pos frontend.Position, format string, args ...interface{}) error {
	return safetyDiagnosticf(pos, DiagnosticCodeSafetyEffect, format, args...)
}

func privacyDiagnosticf(pos frontend.Position, format string, args ...interface{}) error {
	return safetyDiagnosticf(pos, DiagnosticCodeSafetyPrivacy, format, args...)
}

func budgetDiagnosticf(pos frontend.Position, format string, args ...interface{}) error {
	return safetyDiagnosticf(pos, DiagnosticCodeSafetyBudget, format, args...)
}

func unsupportedFunctionValueEscapeError(pos frontend.Position, name string) error {
	return lifetimeDiagnosticf(pos, "function value '%s' cannot escape outside the supported fnptr ABI; use a declared fn(...) parameter, function-typed return, local, struct field, enum payload, or supported same-module global snapshot", name)
}

func unsupportedCallableMutableCaptureEscapeError(pos frontend.Position, kind CallableEscapeKind, name string) error {
	return lifetimeDiagnosticf(pos, "%s-escaped function value captures mutable local '%s'; mutable by-reference captures require a proven lifetime and synchronization model", kind, name)
}

func unsupportedCallableResourceCaptureEscapeError(pos frontend.Position, name, typeName string) error {
	return lifetimeDiagnosticf(pos, "escaped function value captures local '%s' of type '%s'; pointer or resource captures require an explicit ownership transfer model", name, typeName)
}

func unsupportedCapturingClosurePointerEscapeError(pos frontend.Position, name string) error {
	return lifetimeDiagnosticf(pos, "capturing closure '%s' cannot escape as raw ptr; bind it to a declared fn(...) value for the supported by-value fnptr snapshot ABI", name)
}

func unsupportedFunctionTypedExplicitTypeArgsError(pos frontend.Position, phrase string) error {
	return lifetimeDiagnosticf(pos, "explicit type arguments are not supported for %s; function-typed dispatch uses a monomorphic fnptr ABI, so remove explicit type arguments", phrase)
}

func unsupportedFunctionValueCallMessage(name string) string {
	return fmt.Sprintf(
		"function value '%s' cannot be called through the supported fnptr ABI; use a let-bound closure, function-typed local/global/struct field, enum payload, callback parameter, or direct named function symbol",
		name,
	)
}

func unsupportedFunctionValueCallError(pos frontend.Position, name string) error {
	return lifetimeDiagnosticf(pos, "%s", unsupportedFunctionValueCallMessage(name))
}

func unsupportedCallbackUnknownSemanticTargetError(pos frontend.Position, calleeName, clause string) error {
	return fmt.Errorf(
		"%s: callback argument for '%s' has no known fnptr target under semantic clause '%s'; pass a direct named function/closure symbol or a function-typed value with a stable target set",
		frontend.FormatPos(pos),
		calleeName,
		clause,
	)
}

func unsupportedGenericClosureCaptureError(pos frontend.Position, name string) error {
	return lifetimeDiagnosticf(pos, "generic closure literal captures local '%s'; generic closure captures are not supported by the production fnptr ABI; use a non-generic closure or pass captured state explicitly", name)
}

func unsupportedGenericClosureCallbackCaptureError(pos frontend.Position, name string) error {
	return lifetimeDiagnosticf(pos, "callback argument 'closure literal' captures local '%s'; generic closure captures are not supported by the production fnptr ABI; use a non-generic closure or pass captured state explicitly", name)
}

func unsupportedGenericClosurePointerEscapeError(pos frontend.Position, name string) error {
	return lifetimeDiagnosticf(pos, "%s", genericClosurePointerEscapeMessage(name))
}

func unsupportedGenericClosureDirectCallError(pos frontend.Position, name string) error {
	return lifetimeDiagnosticf(pos, "%s", genericClosureDirectCallRequirementMessage(name))
}

func genericClosurePointerEscapeMessage(name string) string {
	return fmt.Sprintf(
		"generic closure '%s' cannot be used as a pointer value; generic closure ABI support is limited to let-bound direct local calls with inferable concrete arguments",
		name,
	)
}

func genericClosureDirectCallRequirementMessage(name string) string {
	return fmt.Sprintf(
		"generic closure '%s' requires the generic direct-call closure ABI: let-bound direct local call with inferable concrete arguments",
		name,
	)
}

func unsupportedGenericCallbackSymbolError(pos frontend.Position, name string) error {
	return fmt.Errorf(
		"%s: generic function symbol '%s' cannot be used as callback argument; callback fnptr ABI requires a monomorphic target at the call site",
		frontend.FormatPos(pos),
		name,
	)
}

func unsupportedThrowingCallbackSymbolError(pos frontend.Position, name string) error {
	return fmt.Errorf(
		"%s: throwing function symbol '%s' cannot be used as callback argument; callback fnptr ABI requires the parameter's declared throws type to match",
		frontend.FormatPos(pos),
		name,
	)
}

func unsupportedImportedMutableFunctionTypedGlobalCallError(pos frontend.Position, name string) error {
	return lifetimeDiagnosticf(pos, "imported mutable function-typed global '%s' cannot be called directly across module boundary; cross-module mutable global-data ABI is not available, expose a module-local function wrapper or immutable public function-typed global", name)
}

func unsupportedImportedMutableFunctionTypedGlobalUseError(pos frontend.Position, name string) error {
	return lifetimeDiagnosticf(pos, "imported mutable function-typed global '%s' cannot be used across module boundary; cross-module mutable global-data ABI is not available, expose a module-local function wrapper or immutable public function-typed global", name)
}

func unsupportedFunctionTypedGlobalTargetError(pos frontend.Position, name string) error {
	return fmt.Errorf(
		"%s: function-typed global '%s' requires a symbol-backed function value for the supported fnptr ABI",
		frontend.FormatPos(pos),
		name,
	)
}

func unsupportedFunctionTypedGlobalSameModuleInitializerError(pos frontend.Position, name string) error {
	return fmt.Errorf(
		"%s: function-typed global '%s' initializer must be a same-module named function symbol for the supported fnptr ABI",
		frontend.FormatPos(pos),
		name,
	)
}

func unsupportedFunctionTypedGlobalImportedInitializerError(pos frontend.Position, name string) error {
	return fmt.Errorf(
		"%s: function-typed global '%s' initializer must be an imported public function symbol for the supported fnptr ABI",
		frontend.FormatPos(pos),
		name,
	)
}

func unsupportedGenericFunctionTypedGlobalInitializerError(pos frontend.Position, symbol, name string) error {
	return fmt.Errorf(
		"%s: generic function symbol '%s' cannot initialize function-typed global '%s'; global fnptr ABI requires a monomorphic target",
		frontend.FormatPos(pos),
		symbol,
		name,
	)
}

func unsupportedFunctionTypedGlobalInitializerSourceError(pos frontend.Position, name string) error {
	return fmt.Errorf(
		"%s: function-typed global '%s' must be initialized with a direct named function symbol or closure literal for the supported fnptr ABI",
		frontend.FormatPos(pos),
		name,
	)
}

func unsupportedFunctionTypedLocalInitializerSourceError(pos frontend.Position, name string) error {
	return fmt.Errorf(
		"%s: function-typed local '%s' initializer must be a symbol-backed function value, target-set-backed function value, direct named function symbol, or closure literal for the supported fnptr ABI",
		frontend.FormatPos(pos),
		name,
	)
}

func unsupportedFunctionTypedLocalInitializerReturnCallSourceError(pos frontend.Position, name, callName string) error {
	return fmt.Errorf(
		"%s: function-typed local '%s' initializer call '%s' must resolve to a function-typed return for the supported fnptr ABI",
		frontend.FormatPos(pos),
		name,
		callName,
	)
}

func unsupportedGenericFunctionTypedLocalInitializerError(pos frontend.Position, symbol, name string) error {
	return fmt.Errorf(
		"%s: generic function symbol '%s' cannot initialize function-typed local '%s'; local fnptr ABI requires a monomorphic target",
		frontend.FormatPos(pos),
		symbol,
		name,
	)
}

func unsupportedGenericFunctionTypedStructFieldInitializerError(pos frontend.Position, symbol, name string) error {
	return fmt.Errorf(
		"%s: generic function symbol '%s' cannot initialize function-typed struct field '%s'; struct-field fnptr ABI requires a monomorphic target",
		frontend.FormatPos(pos),
		symbol,
		name,
	)
}

func unsupportedGenericFunctionTypedEnumPayloadInitializerError(pos frontend.Position, symbol, name string) error {
	return fmt.Errorf(
		"%s: generic function symbol '%s' cannot initialize function-typed enum payload '%s'; enum-payload fnptr ABI requires a monomorphic target",
		frontend.FormatPos(pos),
		symbol,
		name,
	)
}

func unsupportedThrowingFunctionTypedLocalInitializerError(pos frontend.Position, symbol, name string) error {
	return fmt.Errorf(
		"%s: throwing function symbol '%s' cannot initialize function-typed local '%s'; local fnptr ABI requires the declared throws type to match",
		frontend.FormatPos(pos),
		symbol,
		name,
	)
}

func unsupportedGenericFunctionTypedAssignmentError(pos frontend.Position, symbol, targetName string) error {
	return fmt.Errorf(
		"%s: generic function symbol '%s' cannot be assigned to function-typed target '%s'; assignment fnptr ABI requires a monomorphic target",
		frontend.FormatPos(pos),
		symbol,
		targetName,
	)
}

func unsupportedThrowingFunctionTypedAssignmentError(pos frontend.Position, symbol, targetName string) error {
	return fmt.Errorf(
		"%s: throwing function symbol '%s' cannot be assigned to function-typed target '%s'; assignment fnptr ABI requires the target's declared throws type to match",
		frontend.FormatPos(pos),
		symbol,
		targetName,
	)
}

func unsupportedFunctionTypedAssignmentSourceError(pos frontend.Position, targetName string) error {
	return fmt.Errorf(
		"%s: function-typed assignment to '%s' must use a supported fnptr source: closure literal, function-typed local/global/struct field, direct named function/closure symbol, or function-typed return call",
		frontend.FormatPos(pos),
		targetName,
	)
}

func unsupportedFunctionTypedAssignmentReturnCallSourceError(pos frontend.Position, targetName, callName string) error {
	return fmt.Errorf(
		"%s: function-typed assignment to '%s' initializer call '%s' must resolve to a function-typed return for the supported fnptr ABI",
		frontend.FormatPos(pos),
		targetName,
		callName,
	)
}

func unsupportedGenericFunctionTypedReturnError(pos frontend.Position, symbol string) error {
	return fmt.Errorf(
		"%s: generic function symbol '%s' cannot be returned as function-typed value; return fnptr ABI requires a monomorphic target",
		frontend.FormatPos(pos),
		symbol,
	)
}

func unsupportedFunctionTypedReturnSourceError(pos frontend.Position) error {
	return fmt.Errorf(
		"%s: function-typed return must use a supported fnptr source: closure literal, function-typed local/global/struct field, direct named function/closure symbol, or function-typed return call",
		frontend.FormatPos(pos),
	)
}

func safetyDiagnosticf(pos frontend.Position, code string, format string, args ...interface{}) error {
	return &diagnosticError{
		code:    code,
		pos:     pos,
		message: fmt.Sprintf(format, args...),
	}
}
