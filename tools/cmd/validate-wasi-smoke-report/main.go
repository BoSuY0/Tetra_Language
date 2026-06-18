package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

type validationMode string

const (
	validationModeRuntime  validationMode = "runtime"
	validationModeArtifact validationMode = "artifact"
)

var wasiSmokeCases = []struct {
	name               string
	src                string
	expectedExit       int
	expectedDiagnostic string
}{
	{name: "legacy_hello", src: "examples/smoke/basic/hello.tetra", expectedExit: 0},
	{name: "effects_io_smoke", src: "examples/effects/effects_io_smoke.tetra", expectedExit: 0},
	{name: "ui_web_smoke", src: "examples/ui/ui_web_smoke.tetra", expectedExit: 0},
	{name: "core_slices_smoke", src: "examples/core/data/core_slices_smoke.tetra", expectedExit: 0},
	{name: "wasm_globals_smoke", src: "examples/wasm/wasm_globals_smoke.tetra", expectedExit: 0},
	{
		name:         "wasm_multi_return_2_smoke",
		src:          "examples/wasm/wasm_multi_return_2_smoke.tetra",
		expectedExit: 0,
	},
	{
		name:         "wasm_multi_return_3_smoke",
		src:          "examples/wasm/wasm_multi_return_3_smoke.tetra",
		expectedExit: 0,
	},
	{
		name:         "wasm_multi_return_4_smoke",
		src:          "examples/wasm/wasm_multi_return_4_smoke.tetra",
		expectedExit: 0,
	},
	{name: "dogfood_wasi", src: "examples/projects/dogfood_wasi/src/main.tetra", expectedExit: 0},
	{
		name:         "dogfood_web_ui",
		src:          "examples/projects/dogfood_web_ui/src/main.tetra",
		expectedExit: 0,
	},
	{
		name:               "time_sleep_smoke",
		src:                "examples/async/time_sleep_smoke.tetra",
		expectedExit:       0,
		expectedDiagnostic: "runtime not supported on wasm32",
	},
	{
		name:               "task_smoke",
		src:                "examples/tasks/task_smoke.tetra",
		expectedExit:       42,
		expectedDiagnostic: "runtime not supported on wasm32",
	},
	{
		name:               "actors_pingpong",
		src:                "examples/actors/actors_pingpong.tetra",
		expectedExit:       0,
		expectedDiagnostic: "runtime not supported on wasm32",
	},
}

type wasiSmokeReport struct {
	Timestamp    string                `json:"timestamp"`
	Target       string                `json:"target"`
	BuildOnly    bool                  `json:"build_only"`
	Runner       string                `json:"runner,omitempty"`
	Host         string                `json:"host"`
	Version      string                `json:"version"`
	GitHead      string                `json:"git_head,omitempty"`
	IslandsDebug bool                  `json:"islands_debug"`
	Total        int                   `json:"total"`
	Passed       int                   `json:"passed"`
	Failed       int                   `json:"failed"`
	Cases        []wasiSmokeReportCase `json:"cases"`
}

type wasiSmokeReportCase struct {
	Name               string `json:"name"`
	SrcPath            string `json:"src_path"`
	OutPath            string `json:"out_path"`
	ExpectedExit       int    `json:"expected_exit"`
	Unsupported        bool   `json:"unsupported,omitempty"`
	ExpectedDiagnostic string `json:"expected_diagnostic,omitempty"`
	Diagnostic         string `json:"diagnostic,omitempty"`
	ActualExit         *int   `json:"actual_exit,omitempty"`
	Ran                bool   `json:"ran"`
	Pass               bool   `json:"pass"`
	Error              string `json:"error,omitempty"`
}

func main() {
	var reportPath string
	var modeRaw string
	flag.StringVar(&reportPath, "report", "", "path to wasm32-wasi smoke JSON report")
	flag.StringVar(
		&modeRaw,
		"mode",
		string(validationModeRuntime),
		"validation mode: runtime or artifact",
	)
	flag.Parse()

	if reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	mode := validationMode(modeRaw)
	if mode != validationModeRuntime && mode != validationModeArtifact {
		fmt.Fprintf(os.Stderr, "error: unsupported --mode %q\n", modeRaw)
		os.Exit(2)
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateWASISmokeReport(raw, mode); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateWASISmokeReport(raw []byte, mode validationMode) error {
	var report wasiSmokeReport
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&report); err != nil {
		return fmt.Errorf("invalid WASI smoke report JSON: %w", err)
	}
	if dec.More() {
		return fmt.Errorf("invalid WASI smoke report JSON: trailing data")
	}
	return validateWASISmokeReportContract(report, mode)
}

func validateWASISmokeReportContract(report wasiSmokeReport, mode validationMode) error {
	if report.Timestamp == "" {
		return fmt.Errorf("WASI smoke report missing timestamp")
	}
	if _, err := time.Parse(time.RFC3339, report.Timestamp); err != nil {
		return fmt.Errorf("WASI smoke report timestamp is not RFC3339: %w", err)
	}
	if report.Target != "wasm32-wasi" {
		return fmt.Errorf("WASI smoke report target = %q, want wasm32-wasi", report.Target)
	}
	if report.BuildOnly {
		return fmt.Errorf(
			"WASI smoke report build_only = true, want false for supported wasm32-wasi",
		)
	}
	if report.Host == "" {
		return fmt.Errorf("WASI smoke report missing host")
	}
	if report.Version == "" || !strings.HasPrefix(report.Version, "v") {
		return fmt.Errorf("WASI smoke report version must start with v")
	}
	switch mode {
	case validationModeRuntime:
		if report.Runner == "" {
			return fmt.Errorf(
				("WASI runtime report missing WASI runner; missing-runner state " +
					"must fail before producing a runtime report"),
			)
		}
		if report.Runner != "wasmtime" && report.Runner != "node-wasi" {
			return fmt.Errorf(
				"WASI runtime report runner = %q, want wasmtime or node-wasi",
				report.Runner,
			)
		}
	case validationModeArtifact:
		if report.Runner != "" {
			return fmt.Errorf(
				"WASI artifact/import preflight report cannot include runner %q",
				report.Runner,
			)
		}
	default:
		return fmt.Errorf("unsupported WASI smoke validation mode %q", mode)
	}
	if len(report.Cases) != len(wasiSmokeCases) {
		return fmt.Errorf(
			"WASI smoke report case count = %d, want %d",
			len(report.Cases),
			len(wasiSmokeCases),
		)
	}
	if report.Total != len(report.Cases) {
		return fmt.Errorf("WASI smoke report total = %d, want %d", report.Total, len(report.Cases))
	}
	passed := 0
	for i, c := range report.Cases {
		if err := validateWASISmokeCase(i, c, mode); err != nil {
			return err
		}
		if c.Pass {
			passed++
		}
	}
	failed := len(report.Cases) - passed
	if report.Passed != passed || report.Failed != failed {
		return fmt.Errorf(
			"WASI smoke report counts mismatch: got passed=%d failed=%d, computed passed=%d failed=%d",
			report.Passed,
			report.Failed,
			passed,
			failed,
		)
	}
	if report.Failed != 0 {
		return fmt.Errorf("WASI smoke report contains %d failed cases", report.Failed)
	}
	return nil
}

func validateWASISmokeCase(index int, c wasiSmokeReportCase, mode validationMode) error {
	want := wasiSmokeCases[index]
	if c.Name != want.name {
		return fmt.Errorf("WASI smoke case[%d] name = %q, want %q", index, c.Name, want.name)
	}
	if c.SrcPath != want.src {
		return fmt.Errorf("WASI smoke case %s src_path = %q, want %q", c.Name, c.SrcPath, want.src)
	}
	if c.ExpectedExit != want.expectedExit {
		return fmt.Errorf(
			"WASI smoke case %s expected_exit = %d, want %d",
			c.Name,
			c.ExpectedExit,
			want.expectedExit,
		)
	}
	if want.expectedDiagnostic != "" {
		if !c.Unsupported {
			return fmt.Errorf("WASI smoke case %s must be marked unsupported", c.Name)
		}
		if c.ExpectedDiagnostic != want.expectedDiagnostic {
			return fmt.Errorf(
				"WASI smoke case %s expected_diagnostic = %q, want %q",
				c.Name,
				c.ExpectedDiagnostic,
				want.expectedDiagnostic,
			)
		}
		if c.Diagnostic == "" || !strings.Contains(c.Diagnostic, want.expectedDiagnostic) {
			return fmt.Errorf(
				"WASI smoke case %s diagnostic = %q, want containing %q",
				c.Name,
				c.Diagnostic,
				want.expectedDiagnostic,
			)
		}
		if c.OutPath != "" {
			return fmt.Errorf("WASI unsupported case %s cannot include out_path", c.Name)
		}
		if c.Ran {
			return fmt.Errorf("WASI unsupported case %s ran unexpectedly", c.Name)
		}
		if c.ActualExit != nil {
			return fmt.Errorf("WASI unsupported case %s cannot include actual_exit", c.Name)
		}
		if c.Error != "" {
			return fmt.Errorf("WASI unsupported case %s includes error text", c.Name)
		}
		if !c.Pass {
			return fmt.Errorf("WASI unsupported case %s did not pass", c.Name)
		}
		return nil
	}
	if c.Unsupported || c.ExpectedDiagnostic != "" || c.Diagnostic != "" {
		return fmt.Errorf(
			"WASI smoke case %s has unsupported diagnostic metadata unexpectedly",
			c.Name,
		)
	}
	if c.OutPath == "" || !strings.HasSuffix(c.OutPath, ".wasm") {
		return fmt.Errorf("WASI smoke case %s out_path must end with .wasm", c.Name)
	}
	if c.Error != "" {
		return fmt.Errorf("WASI smoke case %s includes error text", c.Name)
	}
	if !c.Pass {
		return fmt.Errorf("WASI smoke case %s did not pass", c.Name)
	}
	switch mode {
	case validationModeRuntime:
		if !c.Ran {
			return fmt.Errorf(
				("WASI runtime case %s did not run; missing-runner state must not " +
					"be recorded as a passing runtime report"),
				c.Name,
			)
		}
		if c.ActualExit == nil {
			return fmt.Errorf("WASI runtime case %s ran without actual_exit", c.Name)
		}
		if *c.ActualExit < 0 || *c.ActualExit > 255 {
			return fmt.Errorf(
				"WASI runtime case %s actual_exit = %d, want 0..255",
				c.Name,
				*c.ActualExit,
			)
		}
		if *c.ActualExit != c.ExpectedExit {
			return fmt.Errorf(
				"WASI runtime case %s actual_exit = %d, want %d",
				c.Name,
				*c.ActualExit,
				c.ExpectedExit,
			)
		}
	case validationModeArtifact:
		if c.Ran {
			return fmt.Errorf("WASI artifact/import preflight case %s ran unexpectedly", c.Name)
		}
		if c.ActualExit != nil {
			return fmt.Errorf(
				"WASI artifact/import preflight case %s cannot include actual_exit",
				c.Name,
			)
		}
	}
	return nil
}
