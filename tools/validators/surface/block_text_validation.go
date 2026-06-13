package surface

import (
	"fmt"
	"strings"
)

type TextMeasurementReport struct {
	ID                string     `json:"id"`
	BlockID           int        `json:"block_id"`
	TextLen           int        `json:"text_len"`
	FontFamily        string     `json:"font_family"`
	FontWeight        int        `json:"font_weight"`
	FontSize          int        `json:"font_size"`
	LineHeight        int        `json:"line_height"`
	MaxWidth          int        `json:"max_width"`
	Measured          SizeReport `json:"measured"`
	LineCount         int        `json:"line_count"`
	Wrap              string     `json:"wrap"`
	Overflow          string     `json:"overflow"`
	Ellipsis          bool       `json:"ellipsis"`
	EllipsizedTextLen int        `json:"ellipsized_text_len"`
	Align             string     `json:"align"`
	Quality           string     `json:"quality"`
	Checksum          string     `json:"checksum"`
}

type FontFallbackReport struct {
	ID              string   `json:"id"`
	RequestedFamily string   `json:"requested_family"`
	ResolvedFamily  string   `json:"resolved_family"`
	Chain           []string `json:"chain"`
	MissingGlyphs   int      `json:"missing_glyphs"`
	Coverage        string   `json:"coverage"`
}

type GlyphCacheReport struct {
	ID          string `json:"id"`
	Strategy    string `json:"strategy"`
	BudgetBytes int    `json:"budget_bytes"`
	UsedBytes   int    `json:"used_bytes"`
	EntryCount  int    `json:"entry_count"`
	Eviction    string `json:"eviction"`
	Bounded     bool   `json:"bounded"`
}

type TextRenderCommandReport struct {
	Order         int        `json:"order"`
	Command       string     `json:"command"`
	MeasurementID string     `json:"measurement_id"`
	BlockID       int        `json:"block_id"`
	Rect          RectReport `json:"rect"`
	Clip          RectReport `json:"clip"`
	Color         string     `json:"color"`
	Opacity       int        `json:"opacity"`
	Quality       string     `json:"quality"`
	Checksum      string     `json:"checksum"`
}

func validateBlockTextEvidence(report Report) []string {
	if !hasBlockTextEvidence(report) {
		return nil
	}

	var issues []string
	if report.TextQualityLevel != "deterministic-fallback-text-v1" {
		issues = append(issues, fmt.Sprintf("text_quality_level is %q, want deterministic-fallback-text-v1", report.TextQualityLevel))
	}
	if report.TextCacheBudgetBytes <= 0 || report.TextCacheBudgetBytes > 1024*1024 {
		issues = append(issues, fmt.Sprintf("text_cache_budget_bytes = %d, want 1..1048576", report.TextCacheBudgetBytes))
	}
	if len(report.TextMeasurements) == 0 {
		issues = append(issues, "text_measurements evidence is required")
	}
	if len(report.FontFallbacks) == 0 {
		issues = append(issues, "font_fallback evidence is required")
	}
	if len(report.GlyphCaches) == 0 {
		issues = append(issues, "glyph cache evidence is required")
	}
	if len(report.TextRenderCommands) == 0 {
		issues = append(issues, "text render command evidence is required")
	}

	measurementIDs := map[string]TextMeasurementReport{}
	hasWrapEllipsis := false
	for _, measurement := range report.TextMeasurements {
		if strings.TrimSpace(measurement.ID) == "" {
			issues = append(issues, "text_measurements id is required")
		} else if _, exists := measurementIDs[measurement.ID]; exists {
			issues = append(issues, fmt.Sprintf("text_measurements duplicate id %q", measurement.ID))
		}
		measurementIDs[measurement.ID] = measurement
		if measurement.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("text_measurements %q block_id must be positive", measurement.ID))
		}
		if measurement.TextLen <= 0 {
			issues = append(issues, fmt.Sprintf("text_measurements %q text_len must be positive", measurement.ID))
		}
		if strings.TrimSpace(measurement.FontFamily) == "" {
			issues = append(issues, fmt.Sprintf("text_measurements %q font_family is required", measurement.ID))
		}
		if measurement.FontSize <= 0 || measurement.LineHeight <= 0 || measurement.FontWeight <= 0 {
			issues = append(issues, fmt.Sprintf("text_measurements %q font metrics must be positive", measurement.ID))
		}
		if measurement.MaxWidth <= 0 || measurement.Measured.W <= 0 || measurement.Measured.H <= 0 || measurement.LineCount <= 0 {
			issues = append(issues, fmt.Sprintf("text_measurements %q measured dimensions and line_count must be positive", measurement.ID))
		}
		if measurement.MaxWidth > 0 && measurement.Measured.W > measurement.MaxWidth {
			issues = append(issues, fmt.Sprintf("text_measurements %q measured width %d exceeds max_width %d", measurement.ID, measurement.Measured.W, measurement.MaxWidth))
		}
		wrap := normalizeTextToken(measurement.Wrap)
		overflow := normalizeTextToken(measurement.Overflow)
		align := normalizeTextToken(measurement.Align)
		if wrap != "none" && wrap != "word" && wrap != "char" {
			issues = append(issues, fmt.Sprintf("text_measurements %q wrap is %q, want none, word, or char", measurement.ID, measurement.Wrap))
		}
		if overflow != "clip" && overflow != "ellipsis" {
			issues = append(issues, fmt.Sprintf("text_measurements %q overflow is %q, want clip or ellipsis", measurement.ID, measurement.Overflow))
		}
		if align != "start" && align != "center" && align != "end" {
			issues = append(issues, fmt.Sprintf("text_measurements %q align is %q, want start, center, or end", measurement.ID, measurement.Align))
		}
		if strings.TrimSpace(measurement.Quality) == "" {
			issues = append(issues, fmt.Sprintf("text_measurements %q quality is required", measurement.ID))
		}
		if !validChecksumLike(measurement.Checksum) {
			issues = append(issues, fmt.Sprintf("text_measurements %q checksum must be sha256 evidence", measurement.ID))
		}
		if measurement.Ellipsis || overflow == "ellipsis" {
			if measurement.EllipsizedTextLen <= 0 || measurement.EllipsizedTextLen >= measurement.TextLen {
				issues = append(issues, fmt.Sprintf("text_measurements %q ellipsis requires ellipsized_text_len between 1 and text_len-1", measurement.ID))
			}
		}
		if (wrap == "word" || wrap == "char") && overflow == "ellipsis" && measurement.Ellipsis && measurement.LineCount >= 1 && measurement.EllipsizedTextLen > 0 && measurement.EllipsizedTextLen < measurement.TextLen {
			hasWrapEllipsis = true
		}
	}
	if !hasWrapEllipsis {
		issues = append(issues, "text_measurements require wrap/ellipsis layout evidence")
	}

	for _, fallback := range report.FontFallbacks {
		if strings.TrimSpace(fallback.ID) == "" {
			issues = append(issues, "font_fallback id is required")
		}
		if strings.TrimSpace(fallback.RequestedFamily) == "" || strings.TrimSpace(fallback.ResolvedFamily) == "" {
			issues = append(issues, "font_fallback requested_family and resolved_family are required")
		}
		if len(fallback.Chain) < 2 {
			issues = append(issues, "font_fallback chain must include at least requested and fallback families")
		}
		if fallback.MissingGlyphs != 0 {
			issues = append(issues, fmt.Sprintf("font_fallback %q missing_glyphs = %d, want 0 for smoke coverage", fallback.ID, fallback.MissingGlyphs))
		}
		if strings.TrimSpace(fallback.Coverage) == "" {
			issues = append(issues, "font_fallback coverage is required")
		}
	}

	for _, cache := range report.GlyphCaches {
		if strings.TrimSpace(cache.ID) == "" {
			issues = append(issues, "glyph cache id is required")
		}
		if !cache.Bounded {
			issues = append(issues, "glyph cache must prove bounded=true")
		}
		if cache.BudgetBytes <= 0 || cache.BudgetBytes > report.TextCacheBudgetBytes {
			issues = append(issues, fmt.Sprintf("glyph cache %q budget_bytes = %d outside text cache budget %d", cache.ID, cache.BudgetBytes, report.TextCacheBudgetBytes))
		}
		if cache.UsedBytes < 0 || cache.UsedBytes > cache.BudgetBytes {
			issues = append(issues, fmt.Sprintf("glyph cache %q used_bytes = %d outside budget %d", cache.ID, cache.UsedBytes, cache.BudgetBytes))
		}
		if cache.EntryCount <= 0 {
			issues = append(issues, fmt.Sprintf("glyph cache %q entry_count must be positive", cache.ID))
		}
		if strings.TrimSpace(cache.Strategy) == "" || strings.TrimSpace(cache.Eviction) == "" {
			issues = append(issues, "glyph cache strategy and eviction are required")
		}
	}

	seenCommands := map[string]bool{}
	lastOrder := 0
	for i, command := range report.TextRenderCommands {
		name := normalizeTextToken(command.Command)
		if command.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("text_render_commands order %d is not strictly greater than previous order %d", command.Order, lastOrder))
		}
		lastOrder = command.Order
		if command.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("text_render_commands[%d] block_id must be positive", i))
		}
		if command.Rect.W <= 0 || command.Rect.H <= 0 || command.Clip.W <= 0 || command.Clip.H <= 0 {
			issues = append(issues, fmt.Sprintf("text_render_commands[%d] rect and clip dimensions must be positive", i))
		}
		if !rectContainsRect(command.Clip, command.Rect) && !rectContainsRect(command.Rect, command.Clip) {
			issues = append(issues, fmt.Sprintf("text_render_commands[%d] rect/clip must overlap as scoped text bounds", i))
		}
		if _, ok := measurementIDs[command.MeasurementID]; !ok {
			issues = append(issues, fmt.Sprintf("text_render_commands[%d] measurement_id %q is not in text_measurements", i, command.MeasurementID))
		}
		if name != "measure" && name != "render_glyphs" && name != "render_caret" {
			issues = append(issues, fmt.Sprintf("text_render_commands[%d] command is %q, want measure, render_glyphs, or render_caret", i, command.Command))
		}
		if strings.TrimSpace(command.Color) == "" || command.Opacity <= 0 {
			issues = append(issues, fmt.Sprintf("text_render_commands[%d] color and opacity are required", i))
		}
		if strings.TrimSpace(command.Quality) == "" {
			issues = append(issues, fmt.Sprintf("text_render_commands[%d] quality is required", i))
		}
		if !validChecksumLike(command.Checksum) {
			issues = append(issues, fmt.Sprintf("text_render_commands[%d] checksum must be sha256 evidence", i))
		}
		seenCommands[name] = true
	}
	for _, command := range []string{"measure", "render_glyphs"} {
		if !seenCommands[command] {
			issues = append(issues, fmt.Sprintf("text render commands require %s command", command))
		}
	}
	if !hasEventTargetKind(report.Events, "InputBlock", "text_input") {
		issues = append(issues, "block text editable input requires text_input targeted to InputBlock")
	}
	if !hasTransition(report.StateTransitions, "InputBlock", "buffer") || !hasTransition(report.StateTransitions, "InputBlock", "caret") {
		issues = append(issues, "block text editable input requires InputBlock buffer and caret state transitions")
	}
	if len(report.Frames) < 2 || strings.TrimSpace(report.Frames[0].Checksum) == "" || report.Frames[0].Checksum == report.Frames[1].Checksum {
		issues = append(issues, "text frame checksum evidence must show rendered text change")
	}
	for _, required := range []string{
		"block text deterministic measurement",
		"block text wrap ellipsis layout",
		"block text font fallback chain",
		"block text bounded glyph cache",
		"block text render command evidence",
		"block text editable lifetime",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("text report requires %s evidence", required))
		}
	}
	return issues
}

func hasBlockTextEvidence(report Report) bool {
	return len(report.TextMeasurements) > 0 ||
		len(report.FontFallbacks) > 0 ||
		len(report.GlyphCaches) > 0 ||
		len(report.TextRenderCommands) > 0 ||
		strings.TrimSpace(report.TextQualityLevel) != "" ||
		report.TextCacheBudgetBytes != 0
}

func normalizeTextToken(value string) string {
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(value, "-", "_")))
}
