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

func shiftDiagnosticColumn(err error, delta int) error {
	if delta == 0 {
		return err
	}
	var diagErr *DiagnosticError
	if !errors.As(err, &diagErr) {
		return err
	}
	info := diagErr.Info
	if info.Column > 0 {
		info.Column += delta
		if info.Column < 1 {
			info.Column = 1
		}
	}
	return &DiagnosticError{Info: info}
}

func hintForDiagnosticMessage(msg string) string {
	switch {
	case strings.Contains(msg, "planned feature"):
		return "Use the supported v1.0 syntax surface, or keep this source behind a later-release feature gate."
	case strings.Contains(msg, "capsule requires at least one metadata entry"):
		return "Add at least one metadata entry inside the capsule block, for example: id: \"tetra://app\"."
	case strings.Contains(msg, "expected indented block after ':'"):
		return "Indent the block under the preceding ':' with spaces."
	case strings.Contains(msg, "tabs are not supported in Flow indentation"):
		return "Replace tabs with spaces in Flow-indented blocks."
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
