package compiler

import (
	"regexp"
	"strconv"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/lower"
	"tetra_language/compiler/internal/semantics"
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

const (
	DiagnosticCodeParse            = frontend.DiagnosticCodeParse
	DiagnosticCodeSemantic         = "TETRA2001"
	DiagnosticCodeSafetyOwnership  = semantics.DiagnosticCodeSafetyOwnership
	DiagnosticCodeSafetyLifetime   = semantics.DiagnosticCodeSafetyLifetime
	DiagnosticCodeSafetyEffect     = semantics.DiagnosticCodeSafetyEffect
	DiagnosticCodeSafetyPrivacy    = semantics.DiagnosticCodeSafetyPrivacy
	DiagnosticCodeSafetyBudget     = semantics.DiagnosticCodeSafetyBudget
	DiagnosticCodeIRVerifier       = lower.DiagnosticCodeIRVerifier
	DiagnosticCodeLowerUnsupported = lower.DiagnosticCodeLowerUnsupported
	DiagnosticCodeTargetRuntime    = "TETRA3003"
	DiagnosticCodeFormatter        = "TETRA_FMT001"
	DiagnosticCodeFormatterCheck   = "TETRA_FMT002"
)

var diagnosticPosRE = regexp.MustCompile(`^(?:(.+):)?(?:line )?([0-9]+):([0-9]+): (.*)$`)

type DiagnosticCodeInfo struct {
	Severity string
	Surface  string
}

func DiagnosticCodeRegistry() map[string]DiagnosticCodeInfo {
	return map[string]DiagnosticCodeInfo{
		DiagnosticCodeParse: {
			Severity: "error",
			Surface:  "parse/frontend",
		},
		DiagnosticCodeSemantic: {
			Severity: "error",
			Surface:  "semantic/compiler",
		},
		DiagnosticCodeSafetyOwnership: {
			Severity: "error",
			Surface:  "semantic safety/ownership",
		},
		DiagnosticCodeSafetyLifetime: {
			Severity: "error",
			Surface:  "semantic safety/lifetime",
		},
		DiagnosticCodeSafetyEffect: {
			Severity: "error",
			Surface:  "semantic safety/effect",
		},
		DiagnosticCodeSafetyPrivacy: {
			Severity: "error",
			Surface:  "semantic safety/privacy",
		},
		DiagnosticCodeSafetyBudget: {
			Severity: "error",
			Surface:  "semantic safety/budget",
		},
		DiagnosticCodeIRVerifier: {
			Severity: "error",
			Surface:  "ir verifier",
		},
		DiagnosticCodeLowerUnsupported: {
			Severity: "error",
			Surface:  "lowering unsupported",
		},
		DiagnosticCodeTargetRuntime: {
			Severity: "error",
			Surface:  "target runtime support",
		},
		DiagnosticCodeFormatter: {
			Severity: "error",
			Surface:  "formatter",
		},
		DiagnosticCodeFormatterCheck: {
			Severity: "error",
			Surface:  "formatter check",
		},
	}
}

func DiagnosticFromError(err error) Diagnostic {
	if err == nil {
		return Diagnostic{}
	}
	if coded, ok := err.(interface{ DiagnosticCode() string }); ok {
		return Diagnostic{
			Code:     defaultString(coded.DiagnosticCode(), DiagnosticCodeParse),
			Message:  err.Error(),
			Severity: "error",
		}
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
		Code:     DiagnosticCodeParse,
		Message:  msg,
		Severity: "error",
	}
	m := diagnosticPosRE.FindStringSubmatch(msg)
	if len(m) == 5 {
		diag.Code = DiagnosticCodeSemantic
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
