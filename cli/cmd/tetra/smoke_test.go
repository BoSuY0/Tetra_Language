package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/compiler"
	"tetra_language/internal/toon"
)

func TestSmokeCommandWritesReport(t *testing.T) {
	target, ok := hostTarget()
	if !ok {
		t.Skip("host target unsupported")
	}
	report := filepath.Join(t.TempDir(), "smoke.json")
	var stdout bytes.Buffer
	code := runCLI([]string{"smoke", "--target", target, "--run=false", "--report", report}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("smoke exit code = %d, stdout=%q", code, stdout.String())
	}
	raw, err := os.ReadFile(report)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(raw), `"cases"`) || !strings.Contains(string(raw), `"islands_hello"`) {
		t.Fatalf("unexpected smoke report: %s", string(raw))
	}
	var smokeReport struct {
		Target  string `json:"target"`
		Version string `json:"version"`
		Total   int    `json:"total"`
		Passed  int    `json:"passed"`
		Failed  int    `json:"failed"`
		Cases   []struct {
			Name string `json:"name"`
			Pass bool   `json:"pass"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &smokeReport); err != nil {
		t.Fatalf("decode smoke report: %v\n%s", err, string(raw))
	}
	if smokeReport.Target != target || smokeReport.Version != compiler.Version() || len(smokeReport.Cases) == 0 {
		t.Fatalf("smoke report shape = %#v", smokeReport)
	}
	if smokeReport.Total != len(smokeReport.Cases) || smokeReport.Passed != len(smokeReport.Cases) || smokeReport.Failed != 0 {
		t.Fatalf("smoke report counts = %#v", smokeReport)
	}
}

func TestSmokeCommandWritesTOONReportMirror(t *testing.T) {
	target, ok := hostTarget()
	if !ok {
		t.Skip("host target unsupported")
	}
	report := filepath.Join(t.TempDir(), "smoke.json")
	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"smoke", "--target", target, "--run=false", "--report", report, "--report-format=both"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("smoke exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	toonPath := strings.TrimSuffix(report, ".json") + ".toon"
	raw, err := os.ReadFile(toonPath)
	if err != nil {
		t.Fatalf("read TOON report: %v", err)
	}
	jsonRaw, err := toon.ConvertTOONToJSON(raw, toon.Options{Strict: true})
	if err != nil {
		t.Fatalf("decode TOON report: %v\n%s", err, raw)
	}
	var smokeReport struct {
		Target string `json:"target"`
		Total  int    `json:"total"`
		Cases  []struct {
			Name string `json:"name"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(jsonRaw, &smokeReport); err != nil {
		t.Fatalf("decode smoke report JSON: %v\n%s", err, jsonRaw)
	}
	if smokeReport.Target != target || smokeReport.Total != len(smokeReport.Cases) || len(smokeReport.Cases) == 0 {
		t.Fatalf("smoke TOON report shape = %#v", smokeReport)
	}
}

func TestSmokeCommandBuildOnlyNativeTargetsMarkUnsupportedFilesystem(t *testing.T) {
	for _, target := range []string{"macos-x64", "windows-x64"} {
		t.Run(target, func(t *testing.T) {
			reportPath := filepath.Join(t.TempDir(), target+"-smoke.json")
			var stdout, stderr bytes.Buffer
			code := runCLI([]string{"smoke", "--target", target, "--run=false", "--report", reportPath}, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("smoke %s exit code = %d, stdout=%q stderr=%q", target, code, stdout.String(), stderr.String())
			}
			raw, err := os.ReadFile(reportPath)
			if err != nil {
				t.Fatalf("read smoke report: %v", err)
			}
			var report struct {
				Target string `json:"target"`
				Total  int    `json:"total"`
				Passed int    `json:"passed"`
				Failed int    `json:"failed"`
				Cases  []struct {
					Name               string `json:"name"`
					Unsupported        bool   `json:"unsupported"`
					ExpectedDiagnostic string `json:"expected_diagnostic"`
					Diagnostic         string `json:"diagnostic"`
					OutPath            string `json:"out_path"`
					Ran                bool   `json:"ran"`
					Pass               bool   `json:"pass"`
					Error              string `json:"error"`
				} `json:"cases"`
			}
			if err := json.Unmarshal(raw, &report); err != nil {
				t.Fatalf("decode smoke report: %v\n%s", err, raw)
			}
			if report.Target != target || report.Total == 0 || report.Passed != report.Total || report.Failed != 0 {
				t.Fatalf("unexpected smoke report counts for %s: %#v", target, report)
			}
			found := false
			for _, c := range report.Cases {
				if c.Name != "core_filesystem_smoke" {
					continue
				}
				found = true
				want := "filesystem runtime not supported on " + target
				if !c.Unsupported || c.ExpectedDiagnostic != want || !strings.Contains(c.Diagnostic, want) || c.OutPath != "" || c.Ran || !c.Pass || c.Error != "" {
					t.Fatalf("unexpected filesystem smoke case for %s: %#v", target, c)
				}
			}
			if !found {
				t.Fatalf("smoke report missing core_filesystem_smoke for %s", target)
			}
		})
	}
}

func TestSmokeCommandListsCasesAsJSON(t *testing.T) {
	var report struct {
		Target       string `json:"target"`
		BuildOnly    bool   `json:"build_only"`
		RunSupported bool   `json:"run_supported"`
		Total        int    `json:"total"`
		IslandsDebug bool   `json:"islands_debug"`
		Cases        []struct {
			Name         string `json:"name"`
			SrcPath      string `json:"src_path"`
			TargetGroup  string `json:"target_group"`
			ExpectedExit int    `json:"expected_exit"`
			DebugOnly    bool   `json:"debug_only"`
		} `json:"cases"`
		ExcludedExamples []struct {
			SrcPath string `json:"src_path"`
			Reason  string `json:"reason"`
		} `json:"excluded_examples"`
	}
	runCLIJSONStdout(t, []string{"smoke", "--list", "--format=json"}, 0, &report)
	if report.Target == "" {
		t.Fatalf("smoke list missing target: %#v", report)
	}
	if report.BuildOnly {
		t.Fatalf("default smoke list unexpectedly marked build-only: %#v", report)
	}
	if report.Total != len(report.Cases) || report.Total < 39 {
		t.Fatalf("smoke list counts = total:%d len:%d", report.Total, len(report.Cases))
	}
	var sawFlowHello bool
	var sawUINative bool
	var sawComplexControl bool
	var sawHelloT4Exclusion bool
	requiredCoreStdlib := map[string]string{
		"core_async_smoke":         "examples/core_async_smoke.tetra",
		"core_capability_smoke":    "examples/core_capability_smoke.tetra",
		"core_collections_smoke":   "examples/core_collections_smoke.tetra",
		"core_component_smoke":     "examples/core_component_smoke.tetra",
		"core_crypto_smoke":        "examples/core_crypto_smoke.tetra",
		"core_filesystem_smoke":    "examples/core_filesystem_smoke.tetra",
		"core_io_smoke":            "examples/core_io_smoke.tetra",
		"core_math_smoke":          "examples/core_math_smoke.tetra",
		"core_memory_smoke":        "examples/core_memory_smoke.tetra",
		"core_networking_smoke":    "examples/core_networking_smoke.tetra",
		"core_serialization_smoke": "examples/core_serialization_smoke.tetra",
		"core_slices_smoke":        "examples/core_slices_smoke.tetra",
		"core_strings_smoke":       "examples/core_strings_smoke.tetra",
		"core_sync_smoke":          "examples/core_sync_smoke.tetra",
		"core_testing_smoke":       "examples/core_testing_smoke.tetra",
		"core_time_smoke":          "examples/core_time_smoke.tetra",
	}
	requiredSurfaceMigrations := map[string]struct {
		src          string
		expectedExit int
	}{
		"surface_migration_ui_web_smoke":          {src: "examples/surface_migration_ui_web_smoke.tetra", expectedExit: 2},
		"surface_migration_ui_native_shell_smoke": {src: "examples/surface_migration_ui_native_shell_smoke.tetra", expectedExit: 11},
		"surface_migration_dogfood_web_ui":        {src: "examples/surface_migration_dogfood_web_ui.tetra", expectedExit: 3},
		"surface_migration_tetra_control_center":  {src: "examples/surface_migration_tetra_control_center.tetra", expectedExit: 5},
	}
	for _, c := range report.Cases {
		if c.Name == "flow_hello" && c.SrcPath == "examples/flow_hello.tetra" && c.TargetGroup == "native" && c.ExpectedExit == 0 {
			sawFlowHello = true
		}
		if c.Name == "ui_native_shell_smoke" && c.SrcPath == "examples/ui_native_shell_smoke.tetra" && c.TargetGroup == "native" && c.ExpectedExit == 0 {
			sawUINative = true
		}
		if c.Name == "complex_control_flow_smoke" && c.SrcPath == "examples/complex_control_flow_smoke.tetra" && c.TargetGroup == "native" && c.ExpectedExit == 42 {
			sawComplexControl = true
		}
		if wantSrc, ok := requiredCoreStdlib[c.Name]; ok && c.SrcPath == wantSrc && c.TargetGroup == "native" && c.ExpectedExit == 42 {
			delete(requiredCoreStdlib, c.Name)
		}
		if want, ok := requiredSurfaceMigrations[c.Name]; ok && c.SrcPath == want.src && c.TargetGroup == "native" && c.ExpectedExit == want.expectedExit {
			delete(requiredSurfaceMigrations, c.Name)
		}
	}
	if !sawFlowHello {
		t.Fatalf("smoke list missing flow_hello: %#v", report.Cases)
	}
	if !sawUINative {
		t.Fatalf("smoke list missing ui_native_shell_smoke: %#v", report.Cases)
	}
	if !sawComplexControl {
		t.Fatalf("smoke list missing complex_control_flow_smoke: %#v", report.Cases)
	}
	if len(requiredCoreStdlib) != 0 {
		t.Fatalf("smoke list missing core stdlib cases: %#v", requiredCoreStdlib)
	}
	if len(requiredSurfaceMigrations) != 0 {
		t.Fatalf("smoke list missing Surface migration cases: %#v", requiredSurfaceMigrations)
	}
	for _, exclusion := range report.ExcludedExamples {
		if exclusion.SrcPath == "examples/projects/hello_t4/src/main.t4" && strings.Contains(exclusion.Reason, report.Target) {
			sawHelloT4Exclusion = true
		}
	}
	if !sawHelloT4Exclusion {
		t.Fatalf("smoke list missing T4 example exclusion for hello_t4: %#v", report.ExcludedExamples)
	}
}

func TestSmokeCommandListsCasesAsTOON(t *testing.T) {
	var report smokeListReport
	raw := runCLITOONStdout(t, []string{"smoke", "--list", "--format=toon"}, 0, &report)
	if report.Target == "" || report.Total != len(report.Cases) || report.Total < 39 {
		t.Fatalf("smoke TOON list shape = %#v\n%s", report, raw)
	}
	if !strings.Contains(raw, "cases[") {
		t.Fatalf("smoke TOON list should use structured cases output:\n%s", raw)
	}
}

func TestSmokeCommandListsNativeSurfaceCounter(t *testing.T) {
	var report smokeListReport
	runCLIJSONStdout(t, []string{"smoke", "--list", "--target", "linux-x64", "--format=json"}, 0, &report)
	if report.Target != "linux-x64" || report.BuildOnly {
		t.Fatalf("native smoke list metadata = %#v", report)
	}
	for _, c := range report.Cases {
		if c.Name != "surface_counter" {
			continue
		}
		if c.SrcPath != "examples/surface_counter.tetra" || c.TargetGroup != "native" || c.ExpectedExit != 1 || c.Unsupported || c.ExpectedDiagnostic != "" || c.DebugOnly {
			t.Fatalf("surface_counter smoke list case = %#v", c)
		}
		return
	}
	t.Fatalf("native smoke list missing surface_counter: %#v", report.Cases)
}

func TestSmokeCommandListsNativeSurfaceTextInput(t *testing.T) {
	var report smokeListReport
	runCLIJSONStdout(t, []string{"smoke", "--list", "--target", "linux-x64", "--format=json"}, 0, &report)
	if report.Target != "linux-x64" || report.BuildOnly {
		t.Fatalf("native smoke list metadata = %#v", report)
	}
	for _, c := range report.Cases {
		if c.Name != "surface_text_input" {
			continue
		}
		if c.SrcPath != "examples/surface_text_input.tetra" || c.TargetGroup != "native" || c.ExpectedExit != 42 || c.Unsupported || c.ExpectedDiagnostic != "" || c.DebugOnly {
			t.Fatalf("surface_text_input smoke list case = %#v", c)
		}
		return
	}
	t.Fatalf("native smoke list missing surface_text_input: %#v", report.Cases)
}

func TestSmokeCommandKeepsInvalidDoubleFreeOutOfDebugList(t *testing.T) {
	var report smokeListReport
	runCLIJSONStdout(t, []string{"smoke", "--list", "--format=json", "--islands-debug"}, 0, &report)
	if !report.IslandsDebug {
		t.Fatalf("islands_debug = false")
	}
	for _, c := range report.Cases {
		if c.Name == "islands_double_free" {
			t.Fatalf("debug smoke list includes semantic-negative islands_double_free: %#v", c)
		}
	}
}

func TestSmokeCommandDefinesIslandsDebugScopeRows(t *testing.T) {
	rows := islandsDebugScopeRows()
	required := map[string]string{
		"overflow_trap":  "live_trap",
		"double_free":    "static_only_nonclaim",
		"use_after_free": "static_only_nonclaim",
		"stale_epoch":    "static_only_nonclaim",
		"wrong_island":   "static_only_nonclaim",
	}
	for _, row := range rows {
		wantStatus, ok := required[row.Name]
		if !ok {
			t.Fatalf("unexpected islands debug scope row: %#v", row)
		}
		if row.Status != wantStatus || row.Evidence == "" || row.Reason == "" {
			t.Fatalf("islands debug scope row %s = %#v, want status %s with evidence/reason", row.Name, row, wantStatus)
		}
		if row.Status == "static_only_nonclaim" && !strings.Contains(row.Reason, "no live") {
			t.Fatalf("static-only scope row %s reason missing no-live nonclaim: %q", row.Name, row.Reason)
		}
		delete(required, row.Name)
	}
	if len(required) != 0 {
		t.Fatalf("missing islands debug scope rows: %#v", required)
	}
}

func TestSmokeCommandListsWASMRuntimeTargets(t *testing.T) {
	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		var report struct {
			Target       string `json:"target"`
			BuildOnly    bool   `json:"build_only"`
			RunSupported bool   `json:"run_supported"`
			Cases        []struct {
				Name               string `json:"name"`
				SrcPath            string `json:"src_path"`
				TargetGroup        string `json:"target_group"`
				ExpectedExit       int    `json:"expected_exit"`
				Unsupported        bool   `json:"unsupported"`
				ExpectedDiagnostic string `json:"expected_diagnostic"`
			} `json:"cases"`
		}
		runCLIJSONStdout(t, []string{"smoke", "--list", "--target", target, "--format=json"}, 0, &report)
		if report.Target != target || report.BuildOnly {
			t.Fatalf("wasm smoke list metadata = %#v", report)
		}
		required := map[string]string{
			"ui_web_smoke":       "examples/ui_web_smoke.tetra",
			"core_slices_smoke":  "examples/core_slices_smoke.tetra",
			"wasm_globals_smoke": "examples/wasm_globals_smoke.tetra",
		}
		if target == "wasm32-wasi" {
			required["wasm_multi_return_2_smoke"] = "examples/wasm_multi_return_2_smoke.tetra"
			required["wasm_multi_return_3_smoke"] = "examples/wasm_multi_return_3_smoke.tetra"
			required["wasm_multi_return_4_smoke"] = "examples/wasm_multi_return_4_smoke.tetra"
		} else {
			required["surface_counter"] = "examples/surface_counter.tetra"
			required["surface_text_input"] = "examples/surface_text_input.tetra"
		}
		unsupported := map[string]string{
			"time_sleep_smoke": "runtime not supported on wasm32",
			"task_smoke":       "runtime not supported on wasm32",
			"actors_pingpong":  "runtime not supported on wasm32",
		}
		for _, c := range report.Cases {
			if wantSrc, ok := required[c.Name]; ok && c.SrcPath == wantSrc && c.TargetGroup == "wasm" && !c.Unsupported && c.ExpectedDiagnostic == "" {
				delete(required, c.Name)
			}
			if wantDiagnostic, ok := unsupported[c.Name]; ok {
				if !c.Unsupported || !strings.Contains(c.ExpectedDiagnostic, wantDiagnostic) {
					t.Fatalf("unsupported wasm case %s = %#v, want diagnostic containing %q", c.Name, c, wantDiagnostic)
				}
				delete(unsupported, c.Name)
			}
		}
		if len(required) != 0 || len(unsupported) != 0 {
			t.Fatalf("wasm smoke list missing required=%#v unsupported=%#v in %#v", required, unsupported, report.Cases)
		}
	}
}

func TestSmokeCommandBuildsWASMTargetWithoutRun(t *testing.T) {
	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		var stdout bytes.Buffer
		reportPath := filepath.Join(t.TempDir(), target+"-smoke.json")
		code := runCLI([]string{"smoke", "--target", target, "--run=false", "--report", reportPath}, &stdout, &bytes.Buffer{})
		if code != 0 {
			t.Fatalf("smoke exit code = %d, stdout=%q", code, stdout.String())
		}
		var report smokeReport
		raw, err := os.ReadFile(reportPath)
		if err != nil {
			t.Fatalf("read smoke report: %v", err)
		}
		if err := json.Unmarshal(raw, &report); err != nil {
			t.Fatalf("decode smoke report: %v\n%s", err, string(raw))
		}
		if report.Target != target || report.Total == 0 {
			t.Fatalf("wasm smoke report = %#v", report)
		}
		if report.Failed != 0 || report.Passed != report.Total {
			t.Fatalf("wasm smoke counts = %#v", report)
		}
		for _, c := range report.Cases {
			if c.Unsupported {
				if c.OutPath != "" || c.ExpectedDiagnostic == "" || c.Diagnostic == "" || !c.Pass {
					t.Fatalf("unexpected unsupported wasm smoke case %s: %#v", c.Name, c)
				}
				continue
			}
			if !strings.HasSuffix(c.OutPath, ".wasm") {
				t.Fatalf("expected wasm output path, case=%#v", c)
			}
			if c.Error != "" {
				t.Fatalf("unexpected wasm smoke error for %s: %s", c.Name, c.Error)
			}
		}
	}
}

func TestSmokeCommandWASMReportUsesDurableArtifacts(t *testing.T) {
	for _, target := range []string{"wasm32-wasi", "wasm32-web"} {
		var stdout bytes.Buffer
		reportDir := t.TempDir()
		reportPath := filepath.Join(reportDir, target+"-smoke.json")
		code := runCLI([]string{"smoke", "--target", target, "--run=false", "--report", reportPath}, &stdout, &bytes.Buffer{})
		if code != 0 {
			t.Fatalf("smoke %s exit code = %d, stdout=%q", target, code, stdout.String())
		}
		var report smokeReport
		raw, err := os.ReadFile(reportPath)
		if err != nil {
			t.Fatalf("read smoke report: %v", err)
		}
		if err := json.Unmarshal(raw, &report); err != nil {
			t.Fatalf("decode smoke report: %v\n%s", err, string(raw))
		}
		if report.Total == 0 || len(report.Cases) != report.Total {
			t.Fatalf("unexpected smoke report shape: %#v", report)
		}
		for _, c := range report.Cases {
			if c.Unsupported {
				if c.OutPath != "" || c.ExpectedDiagnostic == "" || c.Diagnostic == "" || !c.Pass {
					t.Fatalf("%s unsupported case metadata = %#v", c.Name, c)
				}
				continue
			}
			if !strings.HasPrefix(c.OutPath, reportDir+string(os.PathSeparator)) {
				t.Fatalf("%s out_path is not under report dir: %s", c.Name, c.OutPath)
			}
			if _, err := os.Stat(c.OutPath); err != nil {
				t.Fatalf("%s out_path is not durable after smoke command: %s: %v", c.Name, c.OutPath, err)
			}
		}
	}
}

func TestSmokeCommandRunsWASIWithNodeFallbackRunner(t *testing.T) {
	tmpDir := t.TempDir()
	nodeLog := filepath.Join(tmpDir, "node.log")
	fakeNode := filepath.Join(tmpDir, "node")
	if err := os.WriteFile(fakeNode, []byte(`#!/bin/sh
printf '%s\n' "$@" >> "$TETRA_FAKE_NODE_LOG"
case "$*" in
  *core_slices_smoke.wasm*) exit 0 ;;
  *) exit 0 ;;
esac
`), 0o755); err != nil {
		t.Fatalf("write fake node runner: %v", err)
	}
	t.Setenv("PATH", tmpDir)
	t.Setenv("TETRA_FAKE_NODE_LOG", nodeLog)

	var stdout, stderr bytes.Buffer
	reportPath := filepath.Join(tmpDir, "wasi-smoke.json")
	code := runCLI([]string{"smoke", "--target", "wasm32-wasi", "--run=true", "--report", reportPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("smoke exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read smoke report: %v", err)
	}
	var report struct {
		Target string `json:"target"`
		Runner string `json:"runner"`
		Total  int    `json:"total"`
		Passed int    `json:"passed"`
		Failed int    `json:"failed"`
		Cases  []struct {
			Name         string `json:"name"`
			Unsupported  bool   `json:"unsupported"`
			Ran          bool   `json:"ran"`
			ActualExit   *int   `json:"actual_exit"`
			ExpectedExit int    `json:"expected_exit"`
			Diagnostic   string `json:"diagnostic"`
			Pass         bool   `json:"pass"`
			Error        string `json:"error"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode smoke report: %v\n%s", err, string(raw))
	}
	if report.Target != "wasm32-wasi" || report.Runner != "node-wasi" {
		t.Fatalf("unexpected WASI runner report metadata: %#v", report)
	}
	if report.Total == 0 || report.Passed != report.Total || report.Failed != 0 || len(report.Cases) != report.Total {
		t.Fatalf("unexpected WASI runner report counts: %#v", report)
	}
	for _, c := range report.Cases {
		if c.Unsupported {
			if c.Ran || c.ActualExit != nil || c.Diagnostic == "" || !c.Pass || c.Error != "" {
				t.Fatalf("unexpected unsupported WASI runtime case report for %s: %#v", c.Name, c)
			}
			continue
		}
		if !c.Ran || c.ActualExit == nil || *c.ActualExit != c.ExpectedExit || !c.Pass || c.Error != "" {
			t.Fatalf("unexpected WASI runtime case report for %s: %#v", c.Name, c)
		}
	}
	logRaw, err := os.ReadFile(nodeLog)
	if err != nil {
		t.Fatalf("read fake node log: %v", err)
	}
	if !strings.Contains(string(logRaw), "scripts/tools/wasi_run_module.mjs") {
		t.Fatalf("fake node runner was not invoked through WASI helper, log=%q", string(logRaw))
	}
}

func TestSmokeCommandListWASIRunSupportedTracksRunnerAvailability(t *testing.T) {
	t.Run("runner available", func(t *testing.T) {
		restore := stubLookPath(func(name string) (string, error) {
			if name == "wasmtime" {
				return "/usr/bin/wasmtime", nil
			}
			return "", exec.ErrNotFound
		})
		defer restore()

		var report smokeListReport
		runCLIJSONStdout(t, []string{"smoke", "--list", "--target", "wasm32-wasi", "--format=json"}, 0, &report)
		if report.BuildOnly || !report.RunSupported {
			t.Fatalf("wasm32-wasi smoke list metadata with runner = %#v", report)
		}
	})

	t.Run("runner missing", func(t *testing.T) {
		restore := stubLookPath(func(name string) (string, error) {
			return "", exec.ErrNotFound
		})
		defer restore()

		var report smokeListReport
		runCLIJSONStdout(t, []string{"smoke", "--list", "--target", "wasm32-wasi", "--format=json"}, 0, &report)
		if report.BuildOnly || report.RunSupported {
			t.Fatalf("wasm32-wasi smoke list metadata without runner = %#v", report)
		}
	})
}

func TestSmokeCommandWASMTargetGroupsIncludeDogfoodWebUI(t *testing.T) {
	var report smokeListReport
	runCLIJSONStdout(t, []string{"smoke", "--list", "--target", "wasm32-web", "--format=json"}, 0, &report)
	required := map[string]string{
		"ui_web_smoke":       "examples/ui_web_smoke.tetra",
		"surface_counter":    "examples/surface_counter.tetra",
		"surface_text_input": "examples/surface_text_input.tetra",
		"dogfood_web_ui":     "examples/projects/dogfood_web_ui/src/main.tetra",
	}
	for _, c := range report.Cases {
		if wantPath, ok := required[c.Name]; ok {
			if c.SrcPath != wantPath || c.TargetGroup != "wasm" {
				t.Fatalf("case %s = %#v, want src %s in wasm group", c.Name, c, wantPath)
			}
			delete(required, c.Name)
		}
	}
	if len(required) != 0 {
		t.Fatalf("wasm smoke list missing required cases: %#v", required)
	}
}

func TestSmokeCommandRejectsFormatWithoutList(t *testing.T) {
	var stderr bytes.Buffer
	code := runCLI([]string{"smoke", "--format=json"}, &bytes.Buffer{}, &stderr)
	if code != 2 {
		t.Fatalf("smoke exit code = %d, stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--format is only supported with --list") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
