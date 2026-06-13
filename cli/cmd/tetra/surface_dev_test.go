package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSurfaceDevCommandWritesFastRebuildReport(t *testing.T) {
	target := mustHostTarget(t)
	if target != "linux-x64" {
		t.Skip("Surface dev fast rebuild cache evidence is currently linux-x64 scoped")
	}
	dir := t.TempDir()
	entry, tokens, recipes := writeSurfaceDevFixture(t, dir)
	reportPath := filepath.Join(dir, "surface-dev-workflow.json")
	outDir := filepath.Join(dir, "dist")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{
		"surface", "dev",
		"--source", entry,
		"--target", target,
		"--out-dir", outDir,
		"--report", reportPath,
		"--change-file", "token:" + tokens,
		"--change-file", "recipe:" + recipes,
		"--change-file", "source:" + entry,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("surface dev exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report struct {
		Schema                 string `json:"schema"`
		Model                  string `json:"model"`
		ReleaseScope           string `json:"release_scope"`
		Command                string `json:"command"`
		Mode                   string `json:"mode"`
		ReloadSemantics        string `json:"reload_semantics"`
		ProcessRestartRequired bool   `json:"process_restart_required"`
		HotReloadClaim         bool   `json:"hot_reload_claim"`
		Pass                   bool   `json:"pass"`
		Steps                  []struct {
			Name            string   `json:"name"`
			Kind            string   `json:"kind"`
			ChangedPath     string   `json:"changed_path"`
			DurationMS      int64    `json:"duration_ms"`
			CompiledModules []string `json:"compiled_modules"`
			CacheHits       []string `json:"cache_hits"`
			Pass            bool     `json:"pass"`
		} `json:"steps"`
		SourceDiagnostics []struct {
			Kind     string `json:"kind"`
			Path     string `json:"path"`
			Line     int    `json:"line"`
			Column   int    `json:"column"`
			Severity string `json:"severity"`
			Pass     bool   `json:"pass"`
		} `json:"source_diagnostics"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, string(raw))
	}
	if report.Schema != "tetra.surface.dev-workflow.v1" ||
		report.Model != "surface-dev-workflow-v1" ||
		report.ReleaseScope != "surface-v1-linux-web" ||
		report.Command != "tetra surface dev" ||
		report.Mode != "fast-rebuild" ||
		report.ReloadSemantics != "fast-rebuild" ||
		!report.ProcessRestartRequired ||
		report.HotReloadClaim ||
		!report.Pass {
		t.Fatalf("unexpected report header = %#v", report)
	}
	steps := map[string]struct {
		compiled int
		cache    int
		pass     bool
	}{}
	for _, step := range report.Steps {
		if step.DurationMS < 0 || !step.Pass {
			t.Fatalf("bad step = %#v", step)
		}
		steps[step.Kind] = struct {
			compiled int
			cache    int
			pass     bool
		}{compiled: len(step.CompiledModules), cache: len(step.CacheHits), pass: step.Pass}
	}
	for _, want := range []string{"initial", "warm-cache", "token-change", "recipe-change", "source-change"} {
		if !steps[want].pass {
			t.Fatalf("missing or failed rebuild step %q in %#v", want, report.Steps)
		}
	}
	if steps["warm-cache"].compiled != 0 || steps["warm-cache"].cache == 0 {
		t.Fatalf("warm-cache step = %#v, want zero compiled modules and cache hits", steps["warm-cache"])
	}
	for _, want := range []string{"token-change", "recipe-change", "source-change"} {
		if steps[want].compiled == 0 {
			t.Fatalf("%s step = %#v, want changed module compilation", want, steps[want])
		}
	}
	diagnosticKinds := map[string]bool{}
	for _, diag := range report.SourceDiagnostics {
		if diag.Path == "" || diag.Line <= 0 || diag.Column <= 0 || diag.Severity == "" || !diag.Pass {
			t.Fatalf("bad source diagnostic = %#v", diag)
		}
		diagnosticKinds[diag.Kind] = true
	}
	for _, want := range []string{"token", "recipe", "source"} {
		if !diagnosticKinds[want] {
			t.Fatalf("missing %s source diagnostic in %#v", want, report.SourceDiagnostics)
		}
	}
}

func TestSurfaceDevCommandJSONDiagnosticIncludesSurfacePath(t *testing.T) {
	target := mustHostTarget(t)
	if target != "linux-x64" {
		t.Skip("Surface dev diagnostic smoke is currently linux-x64 scoped")
	}
	dir := t.TempDir()
	entry := filepath.Join(dir, "app", "main.tetra")
	writeCLIProjectFile(t, dir, "app/main.tetra", "module app.main\nimport lib.core.morph as morph\nfunc main() -> Int:\n    let x: Int =\n    return 0\n")
	reportPath := filepath.Join(dir, "surface-dev-workflow.json")

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{
		"surface", "dev",
		"--source", entry,
		"--target", target,
		"--diagnostics", "json",
		"--report", reportPath,
	}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("surface dev exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("stdout = %q, want empty on JSON diagnostic failure", stdout.String())
	}
	var cliDiag cliJSONDiagnostic
	if err := json.Unmarshal(stderr.Bytes(), &cliDiag); err != nil {
		t.Fatalf("decode CLI diagnostic: %v\n%s", err, stderr.String())
	}
	if filepath.Clean(cliDiag.File) != filepath.Clean(entry) || cliDiag.Line <= 0 || cliDiag.Column <= 0 || cliDiag.Severity != "error" {
		t.Fatalf("CLI diagnostic = %#v, want positioned source error for %s", cliDiag, entry)
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report struct {
		Pass              bool `json:"pass"`
		SourceDiagnostics []struct {
			Kind     string `json:"kind"`
			Path     string `json:"path"`
			Line     int    `json:"line"`
			Column   int    `json:"column"`
			Severity string `json:"severity"`
			Pass     bool   `json:"pass"`
		} `json:"source_diagnostics"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, string(raw))
	}
	if report.Pass || len(report.SourceDiagnostics) == 0 {
		t.Fatalf("report = %#v, want failing source diagnostic", report)
	}
	first := report.SourceDiagnostics[0]
	if first.Kind != "morph" || filepath.Clean(first.Path) != filepath.Clean(entry) || first.Line <= 0 || first.Column <= 0 || first.Severity != "error" || first.Pass {
		t.Fatalf("source diagnostic = %#v, want Morph-positioned failing diagnostic", first)
	}
}

func writeSurfaceDevFixture(t *testing.T, dir string) (entry string, tokens string, recipes string) {
	t.Helper()
	unique := strings.ReplaceAll(filepath.Base(dir), "-", "_")
	tokens = filepath.Join(dir, "design", "tokens.tetra")
	recipes = filepath.Join(dir, "design", "recipes.tetra")
	entry = filepath.Join(dir, "app", "main.tetra")
	writeCLIProjectFile(t, dir, "design/tokens.tetra", "module design.tokens\n// "+unique+"\nfunc accent() -> Int:\n    return 17\n")
	writeCLIProjectFile(t, dir, "design/recipes.tetra", "module design.recipes\n// "+unique+"\nfunc card() -> Int:\n    return 25\n")
	writeCLIProjectFile(t, dir, "app/main.tetra", "module app.main\n// "+unique+"\nimport design.tokens as tokens\nimport design.recipes as recipes\nfunc main() -> Int:\n    return tokens.accent() + recipes.card()\n")
	return entry, tokens, recipes
}
