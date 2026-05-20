package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateWASMImportsAcceptsTargetAllowlists(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		imports []wasmImport
	}{
		{
			name:   "wasi",
			target: "wasm32-wasi",
			imports: []wasmImport{
				{Module: "wasi_snapshot_preview1", Name: "fd_write"},
				{Module: "wasi_snapshot_preview1", Name: "proc_exit"},
			},
		},
		{
			name:   "web",
			target: "wasm32-web",
			imports: []wasmImport{
				{Module: "tetra_web_v1", Name: "console_log"},
				{Module: "tetra_web_v1", Name: "panic"},
			},
		},
		{
			name:    "no imports",
			target:  "wasm32-web",
			imports: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateWASMImports(buildWASMModule(tt.imports), tt.target); err != nil {
				t.Fatalf("validateWASMImports: %v", err)
			}
		})
	}
}

func TestValidateWASMImportsRejectsExtraImports(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		imports []wasmImport
		want    string
	}{
		{
			name:   "wasi rejects web namespace",
			target: "wasm32-wasi",
			imports: []wasmImport{
				{Module: "wasi_snapshot_preview1", Name: "fd_write"},
				{Module: "tetra_web_v1", Name: "panic"},
			},
			want: "disallowed import tetra_web_v1.panic",
		},
		{
			name:   "web rejects wasi namespace",
			target: "wasm32-web",
			imports: []wasmImport{
				{Module: "tetra_web_v1", Name: "console_log"},
				{Module: "wasi_snapshot_preview1", Name: "fd_write"},
			},
			want: "disallowed import wasi_snapshot_preview1.fd_write",
		},
		{
			name:   "wasi rejects extra syscall",
			target: "wasm32-wasi",
			imports: []wasmImport{
				{Module: "wasi_snapshot_preview1", Name: "fd_write"},
				{Module: "wasi_snapshot_preview1", Name: "path_open"},
			},
			want: "disallowed import wasi_snapshot_preview1.path_open",
		},
		{
			name:   "web rejects extra host function",
			target: "wasm32-web",
			imports: []wasmImport{
				{Module: "tetra_web_v1", Name: "console_log"},
				{Module: "tetra_web_v1", Name: "fetch"},
			},
			want: "disallowed import tetra_web_v1.fetch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWASMImports(buildWASMModule(tt.imports), tt.target)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestValidateWASMImportsRejectsNonFunctionImports(t *testing.T) {
	err := validateWASMImports(buildWASMModule([]wasmImport{
		{Module: "wasi_snapshot_preview1", Name: "fd_write", Kind: wasmImportKindMemory},
	}), "wasm32-wasi")
	if err == nil || !strings.Contains(err.Error(), "non-function import wasi_snapshot_preview1.fd_write") {
		t.Fatalf("error = %v, want non-function import failure", err)
	}
}

func TestValidateWASMImportsRejectsMalformedModule(t *testing.T) {
	err := validateWASMImports([]byte{0x00, 0x61, 0x73}, "wasm32-web")
	if err == nil || !strings.Contains(err.Error(), "invalid wasm header") {
		t.Fatalf("error = %v, want invalid header failure", err)
	}
}

func TestValidateWASMImportReportValidatesCaseOutPaths(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "ok.wasm")
	if err := os.WriteFile(wasmPath, buildWASMModule([]wasmImport{
		{Module: "tetra_web_v1", Name: "console_log"},
	}), 0o644); err != nil {
		t.Fatalf("write wasm: %v", err)
	}
	reportPath := filepath.Join(dir, "report.json")
	report := `{"target":"wasm32-web","cases":[{"name":"ok","out_path":` + quoteJSON(wasmPath) + `,"pass":true}]}`
	if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := validateWASMImportReport(reportPath, "wasm32-web"); err != nil {
		t.Fatalf("validate report: %v", err)
	}
}

func TestValidateWASMImportReportRejectsCaseExtraImport(t *testing.T) {
	dir := t.TempDir()
	wasmPath := filepath.Join(dir, "bad.wasm")
	if err := os.WriteFile(wasmPath, buildWASMModule([]wasmImport{
		{Module: "tetra_web_v1", Name: "console_log"},
		{Module: "wasi_snapshot_preview1", Name: "fd_write"},
	}), 0o644); err != nil {
		t.Fatalf("write wasm: %v", err)
	}
	reportPath := filepath.Join(dir, "report.json")
	report := `{"target":"wasm32-web","cases":[{"name":"bad","out_path":` + quoteJSON(wasmPath) + `,"pass":true}]}`
	if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateWASMImportReport(reportPath, "wasm32-web")
	if err == nil || !strings.Contains(err.Error(), "bad") || !strings.Contains(err.Error(), "disallowed import wasi_snapshot_preview1.fd_write") {
		t.Fatalf("error = %v, want case disallowed import failure", err)
	}
}

func TestValidateWASMImportReportSkipsExpectedUnsupportedCases(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "report.json")
	report := `{"target":"wasm32-wasi","cases":[{"name":"task_smoke","out_path":"","unsupported":true,"expected_diagnostic":"unsupported symbol '__tetra_task_spawn_i32'","pass":true}]}`
	if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := validateWASMImportReport(reportPath, "wasm32-wasi"); err != nil {
		t.Fatalf("validate report: %v", err)
	}
}

func TestImportPolicyRejectsUnknownTarget(t *testing.T) {
	if _, err := importPolicy("linux-x64"); err == nil {
		t.Fatalf("expected unknown target failure")
	}
}

type wasmImport struct {
	Module string
	Name   string
	Kind   byte
}

const (
	wasmImportKindFunc   byte = 0x00
	wasmImportKindMemory byte = 0x02
)

func buildWASMModule(imports []wasmImport) []byte {
	var out bytes.Buffer
	out.Write([]byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00})
	if len(imports) == 0 {
		return out.Bytes()
	}
	var sec bytes.Buffer
	writeTestULEB(&sec, uint32(len(imports)))
	for _, imp := range imports {
		writeTestName(&sec, imp.Module)
		writeTestName(&sec, imp.Name)
		kind := imp.Kind
		if kind == 0 {
			kind = wasmImportKindFunc
		}
		sec.WriteByte(kind)
		switch kind {
		case wasmImportKindFunc:
			writeTestULEB(&sec, 0)
		case wasmImportKindMemory:
			sec.WriteByte(0x00)
			writeTestULEB(&sec, 1)
		default:
			panic("unsupported test import kind")
		}
	}
	out.WriteByte(2)
	writeTestULEB(&out, uint32(sec.Len()))
	out.Write(sec.Bytes())
	return out.Bytes()
}

func writeTestName(buf *bytes.Buffer, name string) {
	writeTestULEB(buf, uint32(len(name)))
	buf.WriteString(name)
}

func writeTestULEB(buf *bytes.Buffer, v uint32) {
	for {
		b := byte(v & 0x7f)
		v >>= 7
		if v != 0 {
			b |= 0x80
		}
		buf.WriteByte(b)
		if v == 0 {
			return
		}
	}
}

func quoteJSON(s string) string {
	var buf bytes.Buffer
	buf.WriteByte('"')
	for _, r := range s {
		if r == '\\' || r == '"' {
			buf.WriteByte('\\')
		}
		buf.WriteRune(r)
	}
	buf.WriteByte('"')
	return buf.String()
}
