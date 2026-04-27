package frontend

import (
	"errors"
	"fmt"
	"strings"
)

const DiagnosticCodeParse = "TETRA0001"

type Diagnostic struct {
	Code     string
	Message  string
	File     string
	Line     int
	Column   int
	Severity string
	Hint     string
}

type DiagnosticError struct {
	Info Diagnostic
}

func (e *DiagnosticError) Error() string {
	if e.Info.Line > 0 && e.Info.Column > 0 {
		pos := Position{File: e.Info.File, Line: e.Info.Line, Col: e.Info.Column}
		return FormatPos(pos) + ": " + e.Info.Message
	}
	return e.Info.Message
}

func (e *DiagnosticError) Diagnostic() Diagnostic {
	info := e.Info
	if info.Code == "" {
		info.Code = DiagnosticCodeParse
	}
	if info.Severity == "" {
		info.Severity = "error"
	}
	return info
}

func DiagnosticForError(err error) (Diagnostic, bool) {
	var provider interface {
		Diagnostic() Diagnostic
	}
	if errors.As(err, &provider) {
		return provider.Diagnostic(), true
	}
	return Diagnostic{}, false
}

func diagnosticErrorf(pos Position, format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	return &DiagnosticError{Info: Diagnostic{
		Code:     DiagnosticCodeParse,
		Message:  msg,
		File:     pos.File,
		Line:     pos.Line,
		Column:   pos.Col,
		Severity: "error",
		Hint:     hintForDiagnosticMessage(msg),
	}}
}

func hintForDiagnosticMessage(msg string) string {
	switch {
	case strings.Contains(msg, "planned feature"):
		return "Use the supported v1.0 syntax surface, or keep this source behind a later-release feature gate."
	case strings.Contains(msg, "expected indented block after ':'"):
		return "Indent the block under the preceding ':' with spaces."
	case strings.Contains(msg, "invalid UTF-8 encoding"):
		return "Save the source as UTF-8 before parsing."
	case strings.Contains(msg, "inline comments are not supported"):
		return "Move the comment to its own line before formatting."
	case strings.HasPrefix(msg, "expected "):
		return "Check the nearby syntax and token order."
	default:
		return ""
	}
}
