package frontend

import (
	"errors"
	"fmt"
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
	return &DiagnosticError{Info: Diagnostic{
		Code:     DiagnosticCodeParse,
		Message:  fmt.Sprintf(format, args...),
		File:     pos.File,
		Line:     pos.Line,
		Column:   pos.Col,
		Severity: "error",
	}}
}
