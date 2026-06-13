package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"tetra_language/tools/internal/reportdecode"
)

type diagnostic struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Severity string `json:"severity"`
	Hint     string `json:"hint,omitempty"`
}

const diagnosticArtifact = "tetra.release.v0_2_0.diagnostic-json.v1"

func main() {
	var path string
	var wantCode string
	var wantSeverity string
	var wantContains string
	var requirePosition bool
	flag.StringVar(&path, "diagnostic", "", "path to a JSON diagnostic object")
	flag.StringVar(&wantCode, "code", "", "expected diagnostic code")
	flag.StringVar(&wantSeverity, "severity", "error", "expected severity")
	flag.StringVar(&wantContains, "contains", "", "substring expected in message")
	flag.BoolVar(&requirePosition, "require-position", false, "require file, line, and column fields")
	flag.Parse()

	if path == "" {
		fmt.Fprintln(os.Stderr, "error: --diagnostic is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	diag, err := parseDiagnostic(raw)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateDiagnostic(diag, wantCode, wantSeverity, wantContains, requirePosition); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseDiagnostic(raw []byte) (diagnostic, error) {
	var diag diagnostic
	if err := reportdecode.DecodeStrict(raw, &diag); err != nil {
		return diagnostic{}, fmt.Errorf("invalid diagnostic report: %w", err)
	}
	return diag, nil
}

func validateDiagnostic(diag diagnostic, wantCode string, wantSeverity string, wantContains string, requirePosition bool) error {
	if strings.TrimSpace(diag.Code) == "" {
		return fmt.Errorf("diagnostic code is required")
	}
	if diag.Code != strings.TrimSpace(diag.Code) {
		return fmt.Errorf("diagnostic code must not contain leading/trailing whitespace")
	}
	if strings.TrimSpace(diag.Message) == "" {
		return fmt.Errorf("diagnostic message is required")
	}
	if diag.Message != strings.TrimSpace(diag.Message) {
		return fmt.Errorf("diagnostic message must not contain leading/trailing whitespace")
	}
	if strings.TrimSpace(diag.Severity) == "" {
		return fmt.Errorf("diagnostic severity is required")
	}
	if diag.Severity != strings.TrimSpace(diag.Severity) {
		return fmt.Errorf("diagnostic severity must not contain leading/trailing whitespace")
	}
	switch diag.Severity {
	case "error", "warning", "info", "hint":
	default:
		return fmt.Errorf("diagnostic severity %q is invalid", diag.Severity)
	}
	if strings.TrimSpace(diag.File) != "" && (diag.Line <= 0 || diag.Column <= 0) {
		return fmt.Errorf("diagnostic file requires positive line and column")
	}
	if strings.TrimSpace(diag.File) == "" && (diag.Line > 0 || diag.Column > 0) {
		return fmt.Errorf("diagnostic line/column require file path")
	}
	if requirePosition {
		if strings.TrimSpace(diag.File) == "" {
			return fmt.Errorf("diagnostic file is required")
		}
		if diag.Line <= 0 {
			return fmt.Errorf("diagnostic line is required")
		}
		if diag.Column <= 0 {
			return fmt.Errorf("diagnostic column is required")
		}
	}
	if wantCode != "" && diag.Code != wantCode {
		return fmt.Errorf("diagnostic code = %q, want %q", diag.Code, wantCode)
	}
	if wantSeverity != "" && diag.Severity != wantSeverity {
		return fmt.Errorf("diagnostic severity = %q, want %q", diag.Severity, wantSeverity)
	}
	if wantContains != "" && !strings.Contains(diag.Message, wantContains) {
		return fmt.Errorf("diagnostic message %q does not contain %q", diag.Message, wantContains)
	}
	return nil
}
