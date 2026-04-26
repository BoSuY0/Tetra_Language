package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

type smokeListReport struct {
	Total        int             `json:"total"`
	IslandsDebug bool            `json:"islands_debug"`
	Cases        []smokeListCase `json:"cases"`
}

type smokeListCase struct {
	Name         string `json:"name"`
	SrcPath      string `json:"src_path"`
	ExpectedExit int    `json:"expected_exit"`
	DebugOnly    bool   `json:"debug_only,omitempty"`
}

func main() {
	var path string
	flag.StringVar(&path, "report", "", "path to tetra smoke --list --format=json output")
	flag.Parse()
	if path == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateSmokeList(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateSmokeList(raw []byte) error {
	var report smokeListReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return fmt.Errorf("invalid smoke list JSON: %w", err)
	}
	if report.Total != len(report.Cases) {
		return fmt.Errorf("smoke list total = %d, want %d", report.Total, len(report.Cases))
	}
	if report.Total < 39 {
		return fmt.Errorf("smoke list has too few cases: %d", report.Total)
	}
	seenNames := map[string]bool{}
	seenSources := map[string]bool{}
	required := map[string]bool{
		"flow_hello":           false,
		"actors_pingpong":      false,
		"enum_match_smoke":     false,
		"effects_io_smoke":     false,
		"typed_errors_smoke":   false,
		"protocol_impl_smoke":  false,
		"core_memory_smoke":    false,
		"for_collection_smoke": false,
	}
	for _, c := range report.Cases {
		if c.Name == "" {
			return fmt.Errorf("smoke case missing name")
		}
		if seenNames[c.Name] {
			return fmt.Errorf("duplicate smoke case %s", c.Name)
		}
		seenNames[c.Name] = true
		if c.SrcPath == "" {
			return fmt.Errorf("smoke case %s missing src_path", c.Name)
		}
		if !strings.HasSuffix(c.SrcPath, ".tetra") {
			return fmt.Errorf("smoke case %s src_path must be a .tetra file", c.Name)
		}
		if seenSources[c.SrcPath] {
			return fmt.Errorf("duplicate smoke src_path %s", c.SrcPath)
		}
		seenSources[c.SrcPath] = true
		if c.ExpectedExit < 0 || c.ExpectedExit > 255 {
			return fmt.Errorf("smoke case %s expected_exit = %d, want 0..255", c.Name, c.ExpectedExit)
		}
		if _, ok := required[c.Name]; ok {
			required[c.Name] = true
		}
		if c.DebugOnly && !report.IslandsDebug {
			return fmt.Errorf("debug-only case %s present when islands_debug is false", c.Name)
		}
	}
	for name, ok := range required {
		if !ok {
			return fmt.Errorf("smoke list missing required case %s", name)
		}
	}
	return nil
}
