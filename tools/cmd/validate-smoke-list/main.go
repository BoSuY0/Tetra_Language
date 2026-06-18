package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tetra_language/tools/internal/reportdecode"
)

type smokeListReport struct {
	Target           string                 `json:"target"`
	BuildOnly        bool                   `json:"build_only"`
	RunSupported     bool                   `json:"run_supported"`
	Total            int                    `json:"total"`
	IslandsDebug     bool                   `json:"islands_debug"`
	Cases            []smokeListCase        `json:"cases"`
	ExcludedExamples []smokeExcludedExample `json:"excluded_examples,omitempty"`
}

type smokeListCase struct {
	Name               string `json:"name"`
	SrcPath            string `json:"src_path"`
	TargetGroup        string `json:"target_group"`
	ExpectedExit       int    `json:"expected_exit"`
	Unsupported        bool   `json:"unsupported,omitempty"`
	ExpectedDiagnostic string `json:"expected_diagnostic,omitempty"`
	DebugOnly          bool   `json:"debug_only,omitempty"`
}

type smokeExcludedExample struct {
	SrcPath string `json:"src_path"`
	Reason  string `json:"reason"`
}

const smokeListArtifact = "tetra.release.v0_2_0.smoke-list.v1"

func main() {
	var path string
	var examplesRoot string
	var format string
	flag.StringVar(&path, "report", "", "path to tetra smoke --list structured output")
	flag.StringVar(
		&examplesRoot,
		"examples-root",
		"",
		"optional examples directory to require smoke coverage or documented exclusion",
	)
	flag.StringVar(&format, "format", "auto", "report format: auto, json, or toon")
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
	if err := validateSmokeListWithExamplesRootFormat(raw, examplesRoot, format); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func isSmokeSourceFile(path string) bool {
	return strings.HasSuffix(path, ".t4") || strings.HasSuffix(path, ".tetra")
}

func validateSmokeList(raw []byte) error {
	return validateSmokeListWithExamplesRootFormat(raw, "", "auto")
}

func validateSmokeListWithExamplesRoot(raw []byte, examplesRoot string) error {
	return validateSmokeListWithExamplesRootFormat(raw, examplesRoot, "auto")
}

func validateSmokeListWithExamplesRootFormat(raw []byte, examplesRoot string, format string) error {
	var report smokeListReport
	if err := reportdecode.DecodeStrictFormat(raw, format, &report); err != nil {
		return fmt.Errorf("invalid smoke list report: %w", err)
	}
	if report.Total != len(report.Cases) {
		return fmt.Errorf("smoke list total = %d, want %d", report.Total, len(report.Cases))
	}
	minCases := 39
	if report.BuildOnly || report.Target == "wasm32-wasi" || report.Target == "wasm32-web" {
		minCases = 5
	}
	if report.Total < minCases {
		return fmt.Errorf("smoke list has too few cases: %d", report.Total)
	}
	seenNames := map[string]bool{}
	seenSources := map[string]bool{}
	required := requiredCasesForReport(report)
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
		if !isSmokeSourceFile(c.SrcPath) {
			return fmt.Errorf("smoke case %s src_path must be a .t4 or .tetra file", c.Name)
		}
		if seenSources[c.SrcPath] {
			return fmt.Errorf("duplicate smoke src_path %s", c.SrcPath)
		}
		seenSources[c.SrcPath] = true
		if c.ExpectedExit < 0 || c.ExpectedExit > 255 {
			return fmt.Errorf(
				"smoke case %s expected_exit = %d, want 0..255",
				c.Name,
				c.ExpectedExit,
			)
		}
		if c.Unsupported {
			if c.ExpectedDiagnostic == "" {
				return fmt.Errorf("unsupported smoke case %s missing expected_diagnostic", c.Name)
			}
		} else if c.ExpectedDiagnostic != "" {
			return fmt.Errorf("smoke case %s has expected_diagnostic but is not marked unsupported", c.Name)
		}
		if report.Target != "" {
			switch c.TargetGroup {
			case "native", "wasm":
			case "":
				return fmt.Errorf("smoke case %s missing target_group", c.Name)
			default:
				return fmt.Errorf(
					"smoke case %s target_group = %q, want native or wasm",
					c.Name,
					c.TargetGroup,
				)
			}
		}
		if _, ok := required[c.Name]; ok {
			required[c.Name] = true
		}
		if c.DebugOnly && !report.IslandsDebug {
			return fmt.Errorf("debug-only case %s present when islands_debug is false", c.Name)
		}
	}
	seenExclusions := map[string]bool{}
	for _, exclusion := range report.ExcludedExamples {
		if exclusion.SrcPath == "" {
			return fmt.Errorf("smoke exclusion missing src_path")
		}
		if exclusion.Reason == "" {
			return fmt.Errorf("smoke exclusion %s missing reason", exclusion.SrcPath)
		}
		if !isSmokeSourceFile(exclusion.SrcPath) {
			return fmt.Errorf(
				"smoke exclusion %s src_path must be a .t4 or .tetra file",
				exclusion.SrcPath,
			)
		}
		if seenSources[exclusion.SrcPath] {
			return fmt.Errorf("smoke exclusion %s is also an active smoke case", exclusion.SrcPath)
		}
		if seenExclusions[exclusion.SrcPath] {
			return fmt.Errorf("duplicate smoke exclusion %s", exclusion.SrcPath)
		}
		seenExclusions[exclusion.SrcPath] = true
	}
	for name, ok := range required {
		if !ok {
			return fmt.Errorf("smoke list missing required case %s", name)
		}
	}
	if examplesRoot != "" {
		examples, err := discoverExamples(examplesRoot)
		if err != nil {
			return err
		}
		for _, example := range examples {
			if !seenSources[example] && !seenExclusions[example] {
				return fmt.Errorf(
					"example %s is not assigned to a smoke case or documented exclusion",
					example,
				)
			}
		}
	}
	return nil
}

func discoverExamples(examplesRoot string) ([]string, error) {
	var examples []string
	err := filepath.WalkDir(examplesRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !isSmokeSourceFile(d.Name()) {
			return nil
		}
		rel, err := filepath.Rel(examplesRoot, path)
		if err != nil {
			return err
		}
		examples = append(examples, "examples/"+filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan examples root: %w", err)
	}
	sort.Strings(examples)
	return examples, nil
}

func requiredCasesForReport(report smokeListReport) map[string]bool {
	if report.BuildOnly || report.Target == "wasm32-wasi" || report.Target == "wasm32-web" {
		required := map[string]bool{
			"legacy_hello":              false,
			"effects_io_smoke":          false,
			"ui_web_smoke":              false,
			"core_slices_smoke":         false,
			"wasm_globals_smoke":        false,
			"wasm_multi_return_2_smoke": false,
			"wasm_multi_return_3_smoke": false,
			"wasm_multi_return_4_smoke": false,
			"dogfood_wasi":              false,
			"dogfood_web_ui":            false,
			"time_sleep_smoke":          false,
			"task_smoke":                false,
			"actors_pingpong":           false,
		}
		if report.Target == "wasm32-web" {
			required["surface_counter"] = false
			required["surface_text_input"] = false
		}
		return required
	}
	return map[string]bool{
		"flow_hello":                              false,
		"actors_pingpong":                         false,
		"enum_match_smoke":                        false,
		"effects_io_smoke":                        false,
		"typed_errors_smoke":                      false,
		"protocol_impl_smoke":                     false,
		"for_collection_smoke":                    false,
		"core_async_smoke":                        false,
		"core_capability_smoke":                   false,
		"core_collections_smoke":                  false,
		"core_component_smoke":                    false,
		"core_crypto_smoke":                       false,
		"core_filesystem_smoke":                   false,
		"core_io_smoke":                           false,
		"core_math_smoke":                         false,
		"core_memory_smoke":                       false,
		"core_networking_smoke":                   false,
		"core_serialization_smoke":                false,
		"core_slices_smoke":                       false,
		"core_strings_smoke":                      false,
		"core_sync_smoke":                         false,
		"core_testing_smoke":                      false,
		"core_time_smoke":                         false,
		"surface_counter":                         false,
		"surface_text_input":                      false,
		"surface_migration_ui_web_smoke":          false,
		"surface_migration_ui_native_shell_smoke": false,
		"surface_migration_dogfood_web_ui":        false,
		"surface_migration_tetra_control_center":  false,
	}
}
