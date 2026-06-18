package surface

import (
	"errors"
	"fmt"
	"strings"
)

type TextInputReport struct {
	Schema                     string                          `json:"schema"`
	Target                     string                          `json:"target"`
	Source                     string                          `json:"source"`
	Level                      string                          `json:"level"`
	Experimental               bool                            `json:"experimental"`
	ProductionClaim            bool                            `json:"production_claim"`
	Storage                    string                          `json:"storage"`
	UTF8Validation             bool                            `json:"utf8_validation"`
	InvalidUTF8Rejected        bool                            `json:"invalid_utf8_rejected"`
	Caret                      bool                            `json:"caret"`
	Selection                  bool                            `json:"selection"`
	SelectionClipboardTransfer bool                            `json:"selection_clipboard_transfer"`
	Multiline                  bool                            `json:"multiline"`
	Backspace                  bool                            `json:"backspace"`
	Delete                     bool                            `json:"delete"`
	HomeEnd                    bool                            `json:"home_end"`
	ArrowLeftRight             bool                            `json:"arrow_left_right"`
	CompositionEvents          bool                            `json:"composition_events"`
	CompositionCommit          bool                            `json:"composition_commit"`
	CompositionCancel          bool                            `json:"composition_cancel"`
	ClipboardRead              bool                            `json:"clipboard_read"`
	ClipboardWrite             bool                            `json:"clipboard_write"`
	ClipboardHostABI           bool                            `json:"clipboard_host_abi"`
	ClipboardOwnedCopy         bool                            `json:"clipboard_owned_copy"`
	TargetHostCompositionTrace bool                            `json:"target_host_composition_trace"`
	CompositionTrace           CompositionTraceReport          `json:"composition_trace"`
	TextShapingPlan            TextShapingPlanReport           `json:"text_shaping_plan"`
	ReferenceTraces            []TextInputReferenceTraceReport `json:"reference_traces"`
	UnsupportedClaims          []string                        `json:"unsupported_claims"`
	RichTextProductionClaim    bool                            `json:"rich_text_production_claim"`
	BidiProductionClaim        bool                            `json:"bidi_production_claim"`
	FullEditorProductionClaim  bool                            `json:"full_editor_production_claim"`
	BorrowedViewStorage        bool                            `json:"borrowed_view_storage"`
	SafeViewLifetimeChecked    bool                            `json:"safe_view_lifetime_checked"`
	Processes                  []ProcessReport                 `json:"processes"`
	Artifacts                  []ArtifactReport                `json:"artifacts"`
	ArtifactScan               ArtifactScanReport              `json:"artifact_scan"`
	Cases                      []CaseReport                    `json:"cases"`
}

type CompositionTraceReport struct {
	Start  bool `json:"start"`
	Update bool `json:"update"`
	Commit bool `json:"commit"`
	Cancel bool `json:"cancel"`
}

type TextShapingPlanReport struct {
	QualityLevel       string `json:"quality_level"`
	FallbackFonts      bool   `json:"fallback_fonts"`
	GraphemeBoundaries string `json:"grapheme_boundaries"`
	LineBreaking       string `json:"line_breaking"`
	Bidi               string `json:"bidi"`
	RichText           string `json:"rich_text"`
}

type TextInputReferenceTraceReport struct {
	Source      string `json:"source"`
	Trace       string `json:"trace"`
	Focus       bool   `json:"focus"`
	Selection   bool   `json:"selection"`
	Clipboard   bool   `json:"clipboard"`
	Composition bool   `json:"composition"`
	Multiline   bool   `json:"multiline"`
	Pass        bool   `json:"pass"`
}

func ValidateTextInputReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != TextInputSchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, TextInputSchemaV1)
	}

	var report TextInputReport
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}

	var issues []string
	if report.Schema != TextInputSchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, TextInputSchemaV1))
	}
	switch report.Target {
	case "headless", "linux-x64", "wasm32-web":
	default:
		issues = append(issues, fmt.Sprintf("target is %q, want headless, linux-x64, or wasm32-web", report.Target))
	}
	if normalizeEvidencePath(report.Source) != "examples/surface_release_text_input.tetra" {
		issues = append(issues, fmt.Sprintf("source is %q, want examples/surface_release_text_input.tetra", report.Source))
	}
	if report.Level != "production-text-input-v1" {
		issues = append(issues, fmt.Sprintf("level is %q, want production-text-input-v1", report.Level))
	}
	if report.Experimental {
		issues = append(issues, "experimental must be false for production text-input reports")
	}
	if !report.ProductionClaim {
		issues = append(issues, "production_claim must be true for production text-input reports")
	}
	if report.Storage != "owned-utf8-byte-buffer" {
		issues = append(issues, fmt.Sprintf("storage is %q, want owned-utf8-byte-buffer", report.Storage))
	}
	for _, check := range []struct {
		field string
		ok    bool
	}{
		{field: "utf8_validation", ok: report.UTF8Validation},
		{field: "invalid_utf8_rejected", ok: report.InvalidUTF8Rejected},
		{field: "caret", ok: report.Caret},
		{field: "selection", ok: report.Selection},
		{field: "selection_clipboard_transfer", ok: report.SelectionClipboardTransfer},
		{field: "multiline", ok: report.Multiline},
		{field: "backspace", ok: report.Backspace},
		{field: "delete", ok: report.Delete},
		{field: "home_end", ok: report.HomeEnd},
		{field: "arrow_left_right", ok: report.ArrowLeftRight},
		{field: "composition_events", ok: report.CompositionEvents},
		{field: "composition_commit", ok: report.CompositionCommit},
		{field: "composition_cancel", ok: report.CompositionCancel},
		{field: "clipboard_read", ok: report.ClipboardRead},
		{field: "clipboard_write", ok: report.ClipboardWrite},
		{field: "clipboard_host_abi", ok: report.ClipboardHostABI},
		{field: "clipboard_owned_copy", ok: report.ClipboardOwnedCopy},
		{field: "target_host_composition_trace", ok: report.TargetHostCompositionTrace},
		{field: "composition_trace.start", ok: report.CompositionTrace.Start},
		{field: "composition_trace.update", ok: report.CompositionTrace.Update},
		{field: "composition_trace.commit", ok: report.CompositionTrace.Commit},
		{field: "composition_trace.cancel", ok: report.CompositionTrace.Cancel},
		{field: "safe_view_lifetime_checked", ok: report.SafeViewLifetimeChecked},
	} {
		if !check.ok {
			issues = append(issues, fmt.Sprintf("%s must be true", check.field))
		}
	}
	if report.BorrowedViewStorage {
		issues = append(issues, "borrowed_view_storage must be false")
	}
	if report.RichTextProductionClaim {
		issues = append(issues, "rich_text_production_claim must be false for text-input v1")
	}
	if report.BidiProductionClaim {
		issues = append(issues, "bidi_production_claim must be false for text-input v1")
	}
	if report.FullEditorProductionClaim {
		issues = append(issues, "full_editor_production_claim must be false for text-input v1")
	}
	issues = append(issues, validateTextShapingPlan(report.TextShapingPlan)...)
	issues = append(issues, validateTextInputReferenceTraces(report.ReferenceTraces)...)
	issues = append(issues, validateExactStringList("unsupported_claims", report.UnsupportedClaims, []string{
		"full-rich-text-editor",
		"full-bidi-shaping",
		"grapheme-cluster-caret",
		"ide-grade-editor",
	})...)
	issues = append(issues, validateProcesses(report.Source, report.Processes)...)
	issues = append(issues, validateArtifacts(report.Target, report.Source, report.Artifacts, report.Processes)...)
	issues = append(issues, validateArtifactScan(report.ArtifactScan, report.Artifacts)...)
	issues = append(issues, validateCases(report.Cases)...)
	for _, required := range []string{
		"release text input ASCII insertion",
		"release text input UTF-8 insertion",
		"release text input invalid UTF-8 rejected",
		"release text input multiline storage",
		"release text input caret home end arrows",
		"release text input selection replacement",
		"release text input selection clipboard transfer",
		"release text input backspace delete",
		"release text input clipboard owned copy transfer",
		"release text input composition start update",
		"release text input composition commit",
		"release text input composition cancel",
		"release text input shaping plan scoped",
		"settings reference text input trace",
		"editor reference text input trace",
		"release text input safe view lifetime checked",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("text-input report requires %s evidence", required))
		}
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateTextShapingPlan(plan TextShapingPlanReport) []string {
	var issues []string
	if plan.QualityLevel != "scoped-text-shaping-plan-v1" {
		issues = append(issues, fmt.Sprintf("text_shaping_plan.quality_level is %q, want scoped-text-shaping-plan-v1", plan.QualityLevel))
	}
	if !plan.FallbackFonts {
		issues = append(issues, "text_shaping_plan.fallback_fonts must be true")
	}
	if plan.GraphemeBoundaries != "byte-offset-codepoint-v1" {
		issues = append(issues, fmt.Sprintf("text_shaping_plan.grapheme_boundaries is %q, want byte-offset-codepoint-v1", plan.GraphemeBoundaries))
	}
	if plan.LineBreaking != "newline-storage-plus-wrap-plan-v1" {
		issues = append(issues, fmt.Sprintf("text_shaping_plan.line_breaking is %q, want newline-storage-plus-wrap-plan-v1", plan.LineBreaking))
	}
	if plan.Bidi != "nonclaim-full-bidi-v1" {
		issues = append(issues, fmt.Sprintf("text_shaping_plan.bidi is %q, want nonclaim-full-bidi-v1", plan.Bidi))
	}
	if plan.RichText != "nonclaim-rich-text-editor-v1" {
		issues = append(issues, fmt.Sprintf("text_shaping_plan.rich_text is %q, want nonclaim-rich-text-editor-v1", plan.RichText))
	}
	return issues
}

func validateTextInputReferenceTraces(traces []TextInputReferenceTraceReport) []string {
	required := []string{
		"examples/surface_morph_settings.tetra",
		"examples/surface_morph_editor_shell.tetra",
	}
	bySource := make(map[string]TextInputReferenceTraceReport, len(traces))
	for _, trace := range traces {
		bySource[normalizeEvidencePath(trace.Source)] = trace
	}
	var issues []string
	for _, source := range required {
		trace, ok := bySource[source]
		if !ok {
			issues = append(issues, fmt.Sprintf("reference_traces requires %s", source))
			continue
		}
		if strings.TrimSpace(trace.Trace) == "" {
			issues = append(issues, fmt.Sprintf("reference_traces %s trace is required", source))
		}
		if !trace.Pass || !trace.Focus || !trace.Selection || !trace.Clipboard || !trace.Composition || !trace.Multiline {
			issues = append(issues, fmt.Sprintf("reference_traces %s must pass focus, selection, clipboard, composition, and multiline checks", source))
		}
	}
	return issues
}
