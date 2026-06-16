package surface

import (
	"errors"
	"fmt"
	"strings"
)

const DevWorkflowSchemaV1 = "tetra.surface.dev-workflow.v1"

type SurfaceDevWorkflowReport struct {
	Schema                 string                           `json:"schema"`
	Model                  string                           `json:"model"`
	ReleaseScope           string                           `json:"release_scope"`
	Command                string                           `json:"command"`
	Source                 string                           `json:"source"`
	Target                 string                           `json:"target"`
	Mode                   string                           `json:"mode"`
	ReloadSemantics        string                           `json:"reload_semantics"`
	ProcessRestartRequired bool                             `json:"process_restart_required"`
	HotReloadClaim         bool                             `json:"hot_reload_claim"`
	Watch                  bool                             `json:"watch"`
	SupportedTargets       []string                         `json:"supported_targets"`
	Steps                  []SurfaceDevWorkflowStepReport   `json:"steps"`
	SourceDiagnostics      []SurfaceDevWorkflowDiagnostic   `json:"source_diagnostics"`
	MorphToPixels          *MorphToPixelsChainReport        `json:"morph_to_pixels,omitempty"`
	NegativeGuards         SurfaceDevWorkflowNegativeGuards `json:"negative_guards"`
	Pass                   bool                             `json:"pass"`
}

type SurfaceDevWorkflowStepReport struct {
	Name            string   `json:"name"`
	Kind            string   `json:"kind"`
	ChangedPath     string   `json:"changed_path"`
	OutputPath      string   `json:"output_path"`
	DurationMS      int64    `json:"duration_ms"`
	CompiledModules []string `json:"compiled_modules"`
	CacheHits       []string `json:"cache_hits"`
	Pass            bool     `json:"pass"`
	Error           string   `json:"error,omitempty"`
}

type SurfaceDevWorkflowDiagnostic struct {
	Kind     string `json:"kind"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	Pass     bool   `json:"pass"`
}

type SurfaceDevWorkflowNegativeGuards struct {
	NoHotReloadClaim                   bool `json:"no_hot_reload_claim"`
	FullRestartDocumentedAsFastRebuild bool `json:"full_restart_documented_as_fast_rebuild"`
	NoElectronDevServer                bool `json:"no_electron_dev_server"`
	NoReactFastRefresh                 bool `json:"no_react_fast_refresh"`
	NoDOMHotReload                     bool `json:"no_dom_hot_reload"`
}

func ValidateDevWorkflowReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != DevWorkflowSchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, DevWorkflowSchemaV1)
	}

	var report SurfaceDevWorkflowReport
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}

	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: report.Schema, want: DevWorkflowSchemaV1},
		{field: "model", got: report.Model, want: "surface-dev-workflow-v1"},
		{field: "release_scope", got: report.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
		{field: "command", got: report.Command, want: "tetra surface dev"},
		{field: "mode", got: report.Mode, want: "fast-rebuild"},
		{field: "reload_semantics", got: report.ReloadSemantics, want: "fast-rebuild"},
	} {
		if check.got != check.want {
			issues = append(issues, fmt.Sprintf("%s is %q, want %q", check.field, check.got, check.want))
		}
	}
	if strings.TrimSpace(report.Source) == "" {
		issues = append(issues, "source is required")
	}
	if report.Target != "linux-x64" {
		issues = append(issues, fmt.Sprintf("target is %q, want linux-x64 for current fast rebuild evidence", report.Target))
	}
	if !report.ProcessRestartRequired {
		issues = append(issues, "process_restart_required must be true so the report cannot imply hot reload")
	}
	if report.HotReloadClaim {
		issues = append(issues, "hot reload claim is forbidden for fast rebuild evidence")
	}
	issues = append(issues, validateExactStringList("supported_targets", report.SupportedTargets, []string{"headless", "linux-x64", "wasm32-web"})...)
	issues = append(issues, validateSurfaceDevWorkflowSteps(report.Steps)...)
	issues = append(issues, validateSurfaceDevWorkflowDiagnostics(report.SourceDiagnostics)...)
	if report.MorphToPixels != nil {
		issues = append(issues, validateMorphToPixelsChain("morph_to_pixels", *report.MorphToPixels, report.Source)...)
	}
	issues = append(issues, validateSurfaceDevWorkflowNegativeGuards(report.NegativeGuards)...)
	if !report.Pass {
		issues = append(issues, "pass must be true")
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateSurfaceDevWorkflowSteps(steps []SurfaceDevWorkflowStepReport) []string {
	if len(steps) == 0 {
		return []string{"steps are required"}
	}
	byKind := map[string]SurfaceDevWorkflowStepReport{}
	var issues []string
	for _, step := range steps {
		if strings.TrimSpace(step.Name) == "" {
			issues = append(issues, "step name is required")
		}
		if strings.TrimSpace(step.Kind) == "" {
			issues = append(issues, "step kind is required")
			continue
		}
		if _, exists := byKind[step.Kind]; exists {
			issues = append(issues, fmt.Sprintf("duplicate step kind %s", step.Kind))
		}
		byKind[step.Kind] = step
		if strings.TrimSpace(step.OutputPath) == "" {
			issues = append(issues, fmt.Sprintf("%s output_path is required", step.Kind))
		}
		if step.DurationMS < 0 {
			issues = append(issues, fmt.Sprintf("%s duration_ms must be non-negative", step.Kind))
		}
		if !step.Pass {
			issues = append(issues, fmt.Sprintf("%s pass must be true", step.Kind))
		}
	}
	for _, required := range []string{"initial", "warm-cache", "token-change", "recipe-change", "source-change"} {
		step, ok := byKind[required]
		if !ok {
			issues = append(issues, fmt.Sprintf("steps missing %s", required))
			continue
		}
		switch required {
		case "warm-cache":
			if len(step.CompiledModules) != 0 || len(step.CacheHits) == 0 {
				issues = append(issues, "warm-cache step must have zero compiled modules and at least one cache hit")
			}
		case "token-change", "recipe-change", "source-change":
			if strings.TrimSpace(step.ChangedPath) == "" {
				issues = append(issues, fmt.Sprintf("%s changed_path is required", required))
			}
			if len(step.CompiledModules) == 0 {
				issues = append(issues, fmt.Sprintf("%s must compile the changed module", required))
			}
		}
	}
	return issues
}

func validateSurfaceDevWorkflowDiagnostics(diags []SurfaceDevWorkflowDiagnostic) []string {
	if len(diags) == 0 {
		return []string{"source_diagnostics are required"}
	}
	var issues []string
	seen := map[string]bool{}
	for _, diag := range diags {
		if strings.TrimSpace(diag.Kind) == "" {
			issues = append(issues, "source_diagnostics kind is required")
			continue
		}
		seen[diag.Kind] = true
		if strings.TrimSpace(diag.Path) == "" {
			issues = append(issues, fmt.Sprintf("source_diagnostics %s path is required", diag.Kind))
		}
		if diag.Line <= 0 || diag.Column <= 0 {
			issues = append(issues, fmt.Sprintf("source_diagnostics %s requires line and column", diag.Kind))
		}
		if strings.TrimSpace(diag.Code) == "" || strings.TrimSpace(diag.Message) == "" || strings.TrimSpace(diag.Severity) == "" {
			issues = append(issues, fmt.Sprintf("source_diagnostics %s requires code, message, and severity", diag.Kind))
		}
		if diag.Severity == "info" && !diag.Pass {
			issues = append(issues, fmt.Sprintf("source_diagnostics %s info row must pass", diag.Kind))
		}
	}
	for _, required := range []string{"token", "recipe", "source"} {
		if !seen[required] {
			issues = append(issues, fmt.Sprintf("source_diagnostics missing %s", required))
		}
	}
	return issues
}

func validateSurfaceDevWorkflowNegativeGuards(guards SurfaceDevWorkflowNegativeGuards) []string {
	if guards.NoHotReloadClaim &&
		guards.FullRestartDocumentedAsFastRebuild &&
		guards.NoElectronDevServer &&
		guards.NoReactFastRefresh &&
		guards.NoDOMHotReload {
		return nil
	}
	return []string{"negative_guards must reject hot reload, Electron dev server, React Fast Refresh, and DOM hot reload claims"}
}
