package compiler

import (
	"regexp"
	"strconv"

	"tetra_language/compiler/internal/frontend"
)

type Diagnostic struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Severity string `json:"severity"`
	Hint     string `json:"hint,omitempty"`
}

var diagnosticPosRE = regexp.MustCompile(`^(?:(.+):)?(?:line )?([0-9]+):([0-9]+): (.*)$`)

func DiagnosticFromError(err error) Diagnostic {
	if err == nil {
		return Diagnostic{}
	}
	if info, ok := frontend.DiagnosticForError(err); ok {
		return Diagnostic{
			Code:     defaultString(info.Code, "TETRA0001"),
			Message:  info.Message,
			File:     info.File,
			Line:     info.Line,
			Column:   info.Column,
			Severity: defaultString(info.Severity, "error"),
			Hint:     info.Hint,
		}
	}
	msg := err.Error()
	diag := Diagnostic{
		Code:     "TETRA0001",
		Message:  msg,
		Severity: "error",
	}
	m := diagnosticPosRE.FindStringSubmatch(msg)
	if len(m) == 5 {
		diag.Code = "TETRA2001"
		diag.File = m[1]
		diag.Line, _ = strconv.Atoi(m[2])
		diag.Column, _ = strconv.Atoi(m[3])
		diag.Message = m[4]
	}
	return diag
}

func defaultString(got string, fallback string) string {
	if got != "" {
		return got
	}
	return fallback
}
