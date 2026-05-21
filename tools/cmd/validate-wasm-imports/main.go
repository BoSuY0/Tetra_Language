package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

const (
	wasmMagicVersion = "\x00asm\x01\x00\x00\x00"
	importKindFunc   = 0x00
)

type importEntry struct {
	Module string
	Name   string
	Kind   byte
}

type smokeReport struct {
	Target string            `json:"target"`
	Cases  []smokeReportCase `json:"cases"`
}

type smokeReportCase struct {
	Name               string `json:"name"`
	OutPath            string `json:"out_path"`
	Unsupported        bool   `json:"unsupported,omitempty"`
	ExpectedDiagnostic string `json:"expected_diagnostic,omitempty"`
	Pass               bool   `json:"pass"`
}

func main() {
	var target string
	var reportPath string
	flag.StringVar(&target, "target", "", "target policy: wasm32-wasi, wasi, wasm32-web, or web")
	flag.StringVar(&reportPath, "report", "", "optional tetra smoke report whose case out_path artifacts should be validated")
	flag.Parse()

	if target == "" {
		fmt.Fprintln(os.Stderr, "error: --target is required")
		os.Exit(2)
	}
	if _, err := importPolicy(target); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if reportPath != "" {
		if err := validateWASMImportReport(reportPath, target); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	for _, path := range flag.Args() {
		if err := validateWASMImportFile(path, target); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if reportPath == "" && flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "error: provide --report or at least one wasm artifact path")
		os.Exit(2)
	}
}

func validateWASMImportReport(reportPath string, target string) error {
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		return err
	}
	var report smokeReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return fmt.Errorf("invalid smoke report JSON %s: %w", reportPath, err)
	}
	if report.Target != "" && canonicalTarget(report.Target) != canonicalTarget(target) {
		return fmt.Errorf("report target %q does not match verifier target %q", report.Target, target)
	}
	for _, c := range report.Cases {
		if c.Unsupported {
			if c.ExpectedDiagnostic == "" {
				return fmt.Errorf("report case %s is unsupported but missing expected_diagnostic", caseName(c))
			}
			if c.OutPath != "" {
				return fmt.Errorf("report case %s is unsupported but has out_path %s", caseName(c), c.OutPath)
			}
			continue
		}
		if c.OutPath == "" {
			if c.Pass {
				return fmt.Errorf("report case %s is passing but has empty out_path", caseName(c))
			}
			continue
		}
		if err := validateWASMImportFile(c.OutPath, target); err != nil {
			return fmt.Errorf("report case %s artifact %s: %w", caseName(c), c.OutPath, err)
		}
	}
	return nil
}

func validateWASMImportFile(path string, target string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := validateWASMImports(raw, target); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func validateWASMImports(raw []byte, target string) error {
	allowed, err := importPolicy(target)
	if err != nil {
		return err
	}
	imports, err := parseWASMImports(raw)
	if err != nil {
		return err
	}
	for _, imp := range imports {
		if imp.Kind != importKindFunc {
			return fmt.Errorf("non-function import %s.%s kind=0x%02x is not allowed", imp.Module, imp.Name, imp.Kind)
		}
		if !allowed[imp.Module+"."+imp.Name] {
			return fmt.Errorf("disallowed import %s.%s for target %s", imp.Module, imp.Name, canonicalTarget(target))
		}
	}
	return nil
}

func importPolicy(target string) (map[string]bool, error) {
	switch canonicalTarget(target) {
	case "wasm32-wasi":
		return map[string]bool{
			"wasi_snapshot_preview1.fd_write":  true,
			"wasi_snapshot_preview1.proc_exit": true,
		}, nil
	case "wasm32-web":
		return map[string]bool{
			"tetra_web_v0.4.0.console_log": true,
			"tetra_web_v0.4.0.panic":       true,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported target %q", target)
	}
}

func canonicalTarget(target string) string {
	switch target {
	case "wasi":
		return "wasm32-wasi"
	case "web":
		return "wasm32-web"
	default:
		return target
	}
}

func parseWASMImports(raw []byte) ([]importEntry, error) {
	if len(raw) < 8 || string(raw[:8]) != wasmMagicVersion {
		return nil, fmt.Errorf("invalid wasm header")
	}
	r := bytes.NewReader(raw[8:])
	var imports []importEntry
	seenImportSection := false
	for r.Len() > 0 {
		sectionID, err := r.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("truncated section id")
		}
		size, err := readULEB(r)
		if err != nil {
			return nil, fmt.Errorf("section %d size: %w", sectionID, err)
		}
		if size > uint64(r.Len()) {
			return nil, fmt.Errorf("section %d size %d exceeds remaining module bytes %d", sectionID, size, r.Len())
		}
		payload := make([]byte, int(size))
		if _, err := r.Read(payload); err != nil {
			return nil, fmt.Errorf("section %d payload: %w", sectionID, err)
		}
		if sectionID != 2 {
			continue
		}
		if seenImportSection {
			return nil, fmt.Errorf("duplicate import section")
		}
		seenImportSection = true
		parsed, err := parseImportSection(payload)
		if err != nil {
			return nil, err
		}
		imports = parsed
	}
	return imports, nil
}

func parseImportSection(payload []byte) ([]importEntry, error) {
	r := bytes.NewReader(payload)
	count, err := readULEB(r)
	if err != nil {
		return nil, fmt.Errorf("import count: %w", err)
	}
	if count > uint64(len(payload)) {
		return nil, fmt.Errorf("import count %d exceeds import section bytes %d", count, len(payload))
	}
	imports := make([]importEntry, 0, count)
	for i := uint64(0); i < count; i++ {
		module, err := readName(r)
		if err != nil {
			return nil, fmt.Errorf("import[%d] module: %w", i, err)
		}
		name, err := readName(r)
		if err != nil {
			return nil, fmt.Errorf("import[%d] name: %w", i, err)
		}
		kind, err := r.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("import[%d] kind: %w", i, err)
		}
		if err := skipImportDesc(r, kind); err != nil {
			return nil, fmt.Errorf("import[%d] %s.%s descriptor: %w", i, module, name, err)
		}
		imports = append(imports, importEntry{Module: module, Name: name, Kind: kind})
	}
	if r.Len() != 0 {
		return nil, fmt.Errorf("import section has %d trailing bytes", r.Len())
	}
	return imports, nil
}

func skipImportDesc(r *bytes.Reader, kind byte) error {
	switch kind {
	case 0x00:
		_, err := readULEB(r)
		return err
	case 0x01:
		if _, err := r.ReadByte(); err != nil {
			return err
		}
		return skipLimits(r)
	case 0x02:
		return skipLimits(r)
	case 0x03:
		if _, err := r.ReadByte(); err != nil {
			return err
		}
		_, err := r.ReadByte()
		return err
	default:
		return fmt.Errorf("unknown import kind 0x%02x", kind)
	}
}

func skipLimits(r *bytes.Reader) error {
	flags, err := readULEB(r)
	if err != nil {
		return err
	}
	if _, err := readULEB(r); err != nil {
		return err
	}
	switch flags {
	case 0x00:
		return nil
	case 0x01:
		_, err := readULEB(r)
		return err
	default:
		return fmt.Errorf("unsupported limits flags 0x%x", flags)
	}
}

func readName(r *bytes.Reader) (string, error) {
	n, err := readULEB(r)
	if err != nil {
		return "", err
	}
	if n > uint64(r.Len()) {
		return "", fmt.Errorf("length %d exceeds remaining bytes %d", n, r.Len())
	}
	buf := make([]byte, int(n))
	if _, err := r.Read(buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func readULEB(r *bytes.Reader) (uint64, error) {
	var result uint64
	for shift := uint(0); shift < 64; shift += 7 {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		result |= uint64(b&0x7f) << shift
		if b&0x80 == 0 {
			return result, nil
		}
	}
	return 0, fmt.Errorf("uleb128 value is too large")
}

func caseName(c smokeReportCase) string {
	if c.Name != "" {
		return c.Name
	}
	return "<unnamed>"
}
