package lower

import (
	"fmt"

	"tetra_language/compiler/internal/frontend"
)

const (
	DiagnosticCodeIRVerifier       = "TETRA3001"
	DiagnosticCodeLowerUnsupported = "TETRA3002"
)

func irVerifierError(format string, args ...interface{}) error {
	return lowerDiagnostic(frontend.Position{}, DiagnosticCodeIRVerifier, fmt.Sprintf(format, args...), "Fix the IR producer before backend codegen.")
}

func irVerifierErrorAt(pos frontend.Position, format string, args ...interface{}) error {
	return lowerDiagnostic(pos, DiagnosticCodeIRVerifier, fmt.Sprintf(format, args...), "Fix the IR producer before backend codegen.")
}

func lowerUnsupportedError(pos frontend.Position, format string, args ...interface{}) error {
	return lowerDiagnostic(pos, DiagnosticCodeLowerUnsupported, fmt.Sprintf(format, args...), "This syntax reached lowering without a supported IR translation.")
}

func lowerDiagnostic(pos frontend.Position, code string, message string, hint string) error {
	return &frontend.DiagnosticError{Info: frontend.Diagnostic{
		Code:     code,
		Message:  message,
		File:     pos.File,
		Line:     pos.Line,
		Column:   pos.Col,
		Severity: "error",
		Hint:     hint,
	}}
}
