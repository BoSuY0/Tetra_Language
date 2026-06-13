package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"tetra_language/tools/internal/reportdecode"
)

type lspSmokeEnvelope struct {
	URI            string          `json:"uri"`
	DiagnosticsRaw json.RawMessage `json:"diagnostics"`
	SymbolsRaw     json.RawMessage `json:"symbols"`
	HoversRaw      json.RawMessage `json:"hovers"`
	Diagnostics    []lspDiagnostic
	Symbols        []lspSymbol
	Hovers         []lspHover
}

type lspDiagnostic struct {
	Message  string `json:"message"`
	Severity string `json:"severity"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Code     string `json:"code,omitempty"`
}

type lspSymbol struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
	Detail string `json:"detail,omitempty"`
}

type lspHover struct {
	Name     string `json:"name"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Contents string `json:"contents"`
}

func main() {
	var reportPath string
	var reportFormat string
	flag.StringVar(&reportPath, "report", "", "path to tetra lsp --stdio-smoke JSON report")
	flag.StringVar(&reportFormat, "format", "auto", "report format: auto, json, or toon")
	flag.Parse()

	if reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateLSPSmokeFormat(raw, reportFormat); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateLSPSmoke(raw []byte) error {
	return validateLSPSmokeFormat(raw, "auto")
}

func validateLSPSmokeFormat(raw []byte, format string) error {
	var report lspSmokeEnvelope
	if err := reportdecode.DecodeStrictFormat(raw, format, &report); err != nil {
		return err
	}
	if report.URI == "" {
		return fmt.Errorf("uri is required")
	}
	if !bytes.HasSuffix([]byte(report.URI), []byte(".tetra")) {
		return fmt.Errorf("uri must reference a .tetra file")
	}
	if err := unmarshalArray(report.DiagnosticsRaw, "diagnostics", &report.Diagnostics); err != nil {
		return err
	}
	if err := unmarshalArray(report.SymbolsRaw, "symbols", &report.Symbols); err != nil {
		return err
	}
	if err := unmarshalArray(report.HoversRaw, "hovers", &report.Hovers); err != nil {
		return err
	}
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Message == "" {
			return fmt.Errorf("diagnostic missing message")
		}
		switch diagnostic.Severity {
		case "error", "warning", "info", "hint":
		default:
			return fmt.Errorf("diagnostic %q has invalid severity %q", diagnostic.Message, diagnostic.Severity)
		}
		if diagnostic.Line <= 0 || diagnostic.Column <= 0 {
			return fmt.Errorf("diagnostic %q has invalid position", diagnostic.Message)
		}
	}
	seenSymbols := map[string]bool{}
	for _, symbol := range report.Symbols {
		if symbol.Name == "" {
			return fmt.Errorf("symbol missing name")
		}
		if symbol.Kind == "" {
			return fmt.Errorf("symbol %s missing kind", symbol.Name)
		}
		if symbol.Line <= 0 || symbol.Column <= 0 {
			return fmt.Errorf("symbol %s has invalid position", symbol.Name)
		}
		key := fmt.Sprintf("%s\x00%s\x00%d\x00%d", symbol.Name, symbol.Kind, symbol.Line, symbol.Column)
		if seenSymbols[key] {
			return fmt.Errorf("duplicate symbol %s at %d:%d", symbol.Name, symbol.Line, symbol.Column)
		}
		seenSymbols[key] = true
	}
	seenHovers := map[string]bool{}
	symbolPositions := map[string]bool{}
	for _, symbol := range report.Symbols {
		symbolPositions[fmt.Sprintf("%s\x00%d\x00%d", symbol.Name, symbol.Line, symbol.Column)] = true
	}
	for _, hover := range report.Hovers {
		if hover.Name == "" {
			return fmt.Errorf("hover missing name")
		}
		if hover.Contents == "" {
			return fmt.Errorf("hover %s missing contents", hover.Name)
		}
		if hover.Line <= 0 || hover.Column <= 0 {
			return fmt.Errorf("hover %s has invalid position", hover.Name)
		}
		key := fmt.Sprintf("%s\x00%d\x00%d", hover.Name, hover.Line, hover.Column)
		if seenHovers[key] {
			return fmt.Errorf("duplicate hover %s at %d:%d", hover.Name, hover.Line, hover.Column)
		}
		seenHovers[key] = true
		if !symbolPositions[key] {
			return fmt.Errorf("hover %s at %d:%d has no matching symbol", hover.Name, hover.Line, hover.Column)
		}
	}
	return nil
}

func unmarshalArray[T any](raw json.RawMessage, field string, out *[]T) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return fmt.Errorf("%s must be an array", field)
	}
	if bytes.Equal(trimmed, []byte("null")) || trimmed[0] != '[' {
		return fmt.Errorf("%s must be an array, not null", field)
	}
	if err := strictDecodeJSON(trimmed, out); err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}
	return nil
}

func strictDecodeJSON(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected extra JSON value")
		}
		return err
	}
	return nil
}
