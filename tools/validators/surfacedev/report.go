package surfacedev

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	SchemaV1           = "tetra.surface.dev-loop.v1"
	LevelFastDevLoopV1 = "surface-fast-dev-loop-v1"
	TemplateSchemaV1   = "tetra.surface.template.v1"
)

var requiredTemplates = []string{
	"surface-minimal",
	"surface-dashboard",
	"surface-form",
	"surface-editor-shell",
	"surface-tray-app",
	"surface-web-canvas",
}

type Report struct {
	Schema            string            `json:"schema"`
	Status            string            `json:"status"`
	Level             string            `json:"level"`
	ProjectRoot       string            `json:"project_root"`
	Template          string            `json:"template"`
	Entry             string            `json:"entry"`
	Source            string            `json:"source"`
	ReleaseScope      string            `json:"release_scope"`
	Mode              string            `json:"mode"`
	Reloads           []ReloadTrace     `json:"reloads"`
	Operations        []Operation       `json:"operations"`
	TemplateSmoke     TemplateSmoke     `json:"template_smoke"`
	StatePreservation StatePreservation `json:"state_preservation"`
	NegativeGuards    NegativeGuards    `json:"negative_guards"`
	NonClaims         []string          `json:"nonclaims"`
}

type ReloadTrace struct {
	Order                int      `json:"order"`
	Kind                 string   `json:"kind"`
	Source               string   `json:"source"`
	PreviousSHA256       string   `json:"previous_sha256"`
	CurrentSHA256        string   `json:"current_sha256"`
	PreviousMTimeUnixNS  int64    `json:"previous_mtime_unix_ns"`
	CurrentMTimeUnixNS   int64    `json:"current_mtime_unix_ns"`
	ChangeDetected       bool     `json:"change_detected"`
	RebuildTriggered     bool     `json:"rebuild_triggered"`
	ReloadApplied        bool     `json:"reload_applied"`
	InspectorUpdated     bool     `json:"inspector_updated"`
	ErrorOverlay         string   `json:"error_overlay"`
	StatePreserved       bool     `json:"state_preserved"`
	SourceMapEntryCount  int      `json:"source_map_entry_count"`
	ComponentSnapshotIDs []string `json:"component_snapshot_ids"`
}

type Operation struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Path   string `json:"path,omitempty"`
	Ran    bool   `json:"ran"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail"`
}

type TemplateSmoke struct {
	Templates      []string `json:"templates"`
	CreatedProject bool     `json:"created_project"`
	Checkable      bool     `json:"checkable"`
	Runnable       bool     `json:"runnable"`
	Inspectable    bool     `json:"inspectable"`
	Packageable    bool     `json:"packageable"`
}

type StatePreservation struct {
	Policy           string   `json:"policy"`
	Decision         string   `json:"decision"`
	Reason           string   `json:"reason"`
	SchemaCompatible bool     `json:"schema_compatible"`
	PreservedKeys    []string `json:"preserved_keys"`
	ResetKeys        []string `json:"reset_keys"`
}

type NegativeGuards struct {
	SourceChangeTraceRequired bool `json:"source_change_trace_required"`
	NoElectronDevServer       bool `json:"no_electron_dev_server"`
	NoReactFastRefresh        bool `json:"no_react_fast_refresh"`
	NoCSSRuntimeInjection     bool `json:"no_css_runtime_injection"`
	NoDOMHotReload            bool `json:"no_dom_hot_reload"`
}

func RequiredTemplates() []string {
	out := make([]string, len(requiredTemplates))
	copy(out, requiredTemplates)
	return out
}

func IsRequiredTemplate(template string) bool {
	template = strings.TrimSpace(template)
	for _, required := range requiredTemplates {
		if template == required {
			return true
		}
	}
	return false
}

func ValidateReport(raw []byte) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var report Report
	if err := dec.Decode(&report); err != nil {
		return err
	}
	return Validate(report)
}

func Validate(report Report) error {
	var issues []string
	if report.Schema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, SchemaV1))
	}
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if report.Level != LevelFastDevLoopV1 {
		issues = append(issues, fmt.Sprintf("level is %q, want %q", report.Level, LevelFastDevLoopV1))
	}
	for name, value := range map[string]string{
		"project_root":  report.ProjectRoot,
		"template":      report.Template,
		"entry":         report.Entry,
		"source":        report.Source,
		"release_scope": report.ReleaseScope,
		"mode":          report.Mode,
	} {
		if strings.TrimSpace(value) == "" {
			issues = append(issues, name+" is required")
		}
	}
	if !IsRequiredTemplate(report.Template) {
		issues = append(issues, fmt.Sprintf("template %q is not one of required Surface templates", report.Template))
	}
	if report.ReleaseScope != "surface-v1-linux-web" {
		issues = append(issues, fmt.Sprintf("release_scope is %q, want surface-v1-linux-web", report.ReleaseScope))
	}
	issues = append(issues, validateReloads(report.Reloads)...)
	issues = append(issues, validateOperations(report.Operations)...)
	issues = append(issues, validateTemplateSmoke(report.TemplateSmoke)...)
	issues = append(issues, validateStatePreservation(report.StatePreservation)...)
	issues = append(issues, validateNegativeGuards(report.NegativeGuards)...)
	issues = append(issues, validateNonClaims(report.NonClaims)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateReloads(reloads []ReloadTrace) []string {
	if len(reloads) == 0 {
		return []string{"source change trace is required for hot reload evidence"}
	}
	var issues []string
	validChangeTrace := false
	for i, reload := range reloads {
		prefix := fmt.Sprintf("reloads[%d]", i)
		if reload.Kind != "source-change-reload" {
			issues = append(issues, fmt.Sprintf("%s kind is %q, want source-change-reload", prefix, reload.Kind))
		}
		if strings.TrimSpace(reload.Source) == "" {
			issues = append(issues, prefix+" source is required")
		}
		if !isSHA256Hex(reload.PreviousSHA256) || !isSHA256Hex(reload.CurrentSHA256) {
			issues = append(issues, prefix+" source change trace requires 64-hex previous_sha256 and current_sha256")
		} else if reload.PreviousSHA256 == reload.CurrentSHA256 {
			issues = append(issues, prefix+" source change trace requires different previous_sha256 and current_sha256")
		} else if reload.ChangeDetected {
			validChangeTrace = true
		}
		if reload.PreviousMTimeUnixNS <= 0 || reload.CurrentMTimeUnixNS <= 0 {
			issues = append(issues, prefix+" source change trace requires source mtimes")
		}
		for name, ok := range map[string]bool{
			"change_detected":    reload.ChangeDetected,
			"rebuild_triggered":  reload.RebuildTriggered,
			"reload_applied":     reload.ReloadApplied,
			"inspector_updated":  reload.InspectorUpdated,
			"state_preserved":    reload.StatePreserved,
			"component_snapshot": len(reload.ComponentSnapshotIDs) > 0,
			"source_map":         reload.SourceMapEntryCount > 0,
		} {
			if !ok {
				issues = append(issues, fmt.Sprintf("%s %s must be present/true", prefix, name))
			}
		}
		if strings.TrimSpace(reload.ErrorOverlay) != "surface-inspector-diagnostics" {
			issues = append(issues, fmt.Sprintf("%s error_overlay is %q, want surface-inspector-diagnostics", prefix, reload.ErrorOverlay))
		}
	}
	if !validChangeTrace {
		issues = append(issues, "source change trace is required for hot reload evidence")
	}
	return issues
}

func validateOperations(operations []Operation) []string {
	required := map[string]bool{
		"check":   false,
		"run":     false,
		"inspect": false,
		"package": false,
	}
	var issues []string
	for i, op := range operations {
		if _, ok := required[op.Kind]; ok && op.Ran && op.Pass && strings.TrimSpace(op.Detail) != "" {
			required[op.Kind] = true
		}
		if strings.TrimSpace(op.Name) == "" || strings.TrimSpace(op.Kind) == "" {
			issues = append(issues, fmt.Sprintf("operations[%d] name and kind are required", i))
		}
	}
	for kind, ok := range required {
		if !ok {
			issues = append(issues, fmt.Sprintf("operation kind %s must be ran/pass with detail", kind))
		}
	}
	return issues
}

func validateTemplateSmoke(smoke TemplateSmoke) []string {
	var issues []string
	seen := map[string]bool{}
	for _, template := range smoke.Templates {
		seen[template] = true
	}
	for _, template := range requiredTemplates {
		if !seen[template] {
			issues = append(issues, fmt.Sprintf("template smoke missing %s", template))
		}
	}
	for name, ok := range map[string]bool{
		"created_project": smoke.CreatedProject,
		"checkable":       smoke.Checkable,
		"runnable":        smoke.Runnable,
		"inspectable":     smoke.Inspectable,
		"packageable":     smoke.Packageable,
	} {
		if !ok {
			issues = append(issues, "template smoke "+name+" must be true")
		}
	}
	return issues
}

func validateStatePreservation(state StatePreservation) []string {
	var issues []string
	if state.Policy != "schema-compatible-owned-state-only" {
		issues = append(issues, fmt.Sprintf("state_preservation.policy is %q, want schema-compatible-owned-state-only", state.Policy))
	}
	if state.Decision != "preserve" && state.Decision != "reset" {
		issues = append(issues, fmt.Sprintf("state_preservation.decision is %q, want preserve or reset", state.Decision))
	}
	if strings.TrimSpace(state.Reason) == "" {
		issues = append(issues, "state_preservation.reason is required")
	}
	if !state.SchemaCompatible {
		issues = append(issues, "state_preservation.schema_compatible must be true")
	}
	if state.Decision == "preserve" && len(state.PreservedKeys) == 0 {
		issues = append(issues, "state_preservation.preserved_keys are required when preserving state")
	}
	return issues
}

func validateNegativeGuards(guards NegativeGuards) []string {
	required := map[string]bool{
		"source_change_trace_required": guards.SourceChangeTraceRequired,
		"no_electron_dev_server":       guards.NoElectronDevServer,
		"no_react_fast_refresh":        guards.NoReactFastRefresh,
		"no_css_runtime_injection":     guards.NoCSSRuntimeInjection,
		"no_dom_hot_reload":            guards.NoDOMHotReload,
	}
	var issues []string
	for name, ok := range required {
		if !ok {
			issues = append(issues, "negative guard "+name+" must be true")
		}
	}
	return issues
}

func validateNonClaims(nonClaims []string) []string {
	required := []string{
		"browser devtools parity",
		"React Fast Refresh compatibility",
		"CSS HMR runtime",
		"state preservation across incompatible schemas",
	}
	haystack := strings.ToLower(strings.Join(nonClaims, "\n"))
	var issues []string
	for _, claim := range required {
		if !strings.Contains(haystack, strings.ToLower(claim)) {
			issues = append(issues, fmt.Sprintf("nonclaims missing %q", claim))
		}
	}
	return issues
}

func isSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
