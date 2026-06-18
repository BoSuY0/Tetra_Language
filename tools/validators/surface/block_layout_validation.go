package surface

import (
	"fmt"
	"strings"
)

type BlockLayoutConstraintReport struct {
	ID           string     `json:"id"`
	BlockID      int        `json:"block_id"`
	Mode         string     `json:"mode"`
	WidthPolicy  string     `json:"width_policy"`
	HeightPolicy string     `json:"height_policy"`
	Min          SizeReport `json:"min"`
	Max          SizeReport `json:"max"`
	Padding      int        `json:"padding"`
	Margin       int        `json:"margin"`
	Gap          int        `json:"gap"`
	Align        string     `json:"align"`
	Justify      string     `json:"justify"`
	Overflow     string     `json:"overflow"`
	ZIndex       int        `json:"z_index"`
	Clip         bool       `json:"clip"`
}

type BlockLayoutPassReport struct {
	Order    int        `json:"order"`
	ParentID int        `json:"parent_id"`
	BlockID  int        `json:"block_id"`
	Mode     string     `json:"mode"`
	Input    RectReport `json:"input"`
	Resolved RectReport `json:"resolved"`
	Measured SizeReport `json:"measured"`
	Pass     string     `json:"pass"`
	Resize   bool       `json:"resize"`
	Clip     bool       `json:"clip"`
	ZIndex   int        `json:"z_index"`
	Checksum string     `json:"checksum"`
}

type BlockLayoutScrollReport struct {
	BlockID    int        `json:"block_id"`
	Viewport   RectReport `json:"viewport"`
	Content    SizeReport `json:"content"`
	OffsetY    int        `json:"offset_y"`
	MaxOffsetY int        `json:"max_offset_y"`
	Clipped    bool       `json:"clipped"`
	Checksum   string     `json:"checksum"`
}

type BlockLayoutDensityReport struct {
	TargetDPI      int      `json:"target_dpi"`
	ScaleMilli     int      `json:"scale_milli"`
	BaseUnitPx     int      `json:"base_unit_px"`
	RoundingPolicy string   `json:"rounding_policy"`
	PixelSnapping  bool     `json:"pixel_snapping"`
	Breakpoints    []string `json:"breakpoints"`
	Checksum       string   `json:"checksum"`
}

func validateBlockLayoutEvidence(report Report) []string {
	if !hasBlockLayoutEvidence(report) {
		return nil
	}

	var issues []string
	if report.LayoutQualityLevel != "deterministic-block-layout-v1" {
		issues = append(issues, fmt.Sprintf("layout_quality_level is %q, want deterministic-block-layout-v1", report.LayoutQualityLevel))
	}
	if report.LayoutUnsupportedCSSFlexbox {
		issues = append(issues, "layout unsupported CSS flexbox claim must be false")
	}
	if len(report.LayoutFeatures) == 0 {
		issues = append(issues, "layout_features evidence is required")
	}
	if len(report.LayoutConstraints) == 0 {
		issues = append(issues, "layout_constraints evidence is required")
	}
	if len(report.LayoutPasses) == 0 {
		issues = append(issues, "layout_passes evidence is required")
	}
	if len(report.LayoutScrolls) == 0 {
		issues = append(issues, "layout_scrolls evidence is required")
	}
	if report.LayoutDensity == nil {
		issues = append(issues, "layout_density evidence is required")
	} else {
		issues = append(issues, validateBlockLayoutDensityEvidence(*report.LayoutDensity)...)
	}

	for _, feature := range []string{"stack", "row", "column", "absolute", "overlay", "grid", "dock", "scroll", "fit", "fill", "fixed", "min", "max", "aspect", "spacing", "alignment", "z-order", "clipping", "resize", "density", "stable-rounding"} {
		if !layoutFeatureContains(report.LayoutFeatures, feature) {
			issues = append(issues, fmt.Sprintf("layout_features require %s", feature))
		}
	}

	hasPolicy := map[string]bool{}
	hasMin := false
	hasMax := false
	hasSpacing := false
	hasAlignment := false
	hasClip := false
	hasZ := false
	hasAspect := false
	constraintIDs := map[string]bool{}
	for _, constraint := range report.LayoutConstraints {
		id := strings.TrimSpace(constraint.ID)
		if id == "" {
			issues = append(issues, "layout_constraints id is required")
		} else if constraintIDs[id] {
			issues = append(issues, fmt.Sprintf("layout_constraints duplicate id %q", id))
		}
		constraintIDs[id] = true
		if constraint.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("layout_constraints %q block_id must be positive", id))
		}
		if !validLayoutMode(constraint.Mode) {
			issues = append(issues, fmt.Sprintf("layout_constraints %q mode is %q, want supported Block layout mode", id, constraint.Mode))
		}
		widthPolicy := normalizeLayoutToken(constraint.WidthPolicy)
		heightPolicy := normalizeLayoutToken(constraint.HeightPolicy)
		if !validLayoutPolicy(widthPolicy) {
			issues = append(issues, fmt.Sprintf("layout_constraints %q width_policy is %q, want fixed, fit, or fill", id, constraint.WidthPolicy))
		}
		if !validLayoutPolicy(heightPolicy) {
			issues = append(issues, fmt.Sprintf("layout_constraints %q height_policy is %q, want fixed, fit, or fill", id, constraint.HeightPolicy))
		}
		hasPolicy[widthPolicy] = true
		hasPolicy[heightPolicy] = true
		if constraint.Min.W > 0 && constraint.Min.H > 0 {
			hasMin = true
		} else {
			issues = append(issues, fmt.Sprintf("layout_constraints %q min dimensions must be positive", id))
		}
		if constraint.Max.W > 0 && constraint.Max.H > 0 {
			hasMax = true
		} else {
			issues = append(issues, fmt.Sprintf("layout_constraints %q max dimensions must be positive", id))
		}
		if constraint.Max.W > 0 && constraint.Min.W > constraint.Max.W {
			issues = append(issues, fmt.Sprintf("layout_constraints %q min.w exceeds max.w", id))
		}
		if constraint.Max.H > 0 && constraint.Min.H > constraint.Max.H {
			issues = append(issues, fmt.Sprintf("layout_constraints %q min.h exceeds max.h", id))
		}
		if constraint.Padding < 0 || constraint.Margin < 0 || constraint.Gap < 0 {
			issues = append(issues, fmt.Sprintf("layout_constraints %q spacing values must be non-negative", id))
		}
		if constraint.Padding > 0 || constraint.Margin > 0 || constraint.Gap > 0 {
			hasSpacing = true
		}
		if !validLayoutAlign(constraint.Align) {
			issues = append(issues, fmt.Sprintf("layout_constraints %q align is %q, want start, center, end, or stretch", id, constraint.Align))
		} else {
			hasAlignment = true
		}
		if !validLayoutJustify(constraint.Justify) {
			issues = append(issues, fmt.Sprintf("layout_constraints %q justify is %q, want start, center, end, or space-between", id, constraint.Justify))
		}
		if !validLayoutOverflow(constraint.Overflow) {
			issues = append(issues, fmt.Sprintf("layout_constraints %q overflow is %q, want visible, clip, or scroll", id, constraint.Overflow))
		}
		if constraint.ZIndex > 0 {
			hasZ = true
		}
		if constraint.Clip || normalizeLayoutToken(constraint.Overflow) == "clip" || normalizeLayoutToken(constraint.Overflow) == "scroll" {
			hasClip = true
		}
		if strings.Contains(normalizeLayoutToken(id), "aspect") {
			hasAspect = true
		}
	}
	for _, policy := range []string{"fixed", "fit", "fill"} {
		if !hasPolicy[policy] {
			issues = append(issues, fmt.Sprintf("layout_constraints require %s sizing policy evidence", policy))
		}
	}
	if !hasMin {
		issues = append(issues, "layout_constraints require min sizing evidence")
	}
	if !hasMax {
		issues = append(issues, "layout_constraints require max sizing evidence")
	}
	if !hasSpacing {
		issues = append(issues, "layout_constraints require spacing evidence")
	}
	if !hasAlignment {
		issues = append(issues, "layout_constraints require alignment evidence")
	}
	if !hasZ {
		issues = append(issues, "layout_constraints require z-order evidence")
	}
	if !hasClip {
		issues = append(issues, "layout_constraints require clipping evidence")
	}

	passModes := map[string]bool{}
	lastOrder := 0
	hasResize := false
	for i, pass := range report.LayoutPasses {
		if pass.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("layout_passes order %d is not strictly greater than previous order %d", pass.Order, lastOrder))
		}
		lastOrder = pass.Order
		mode := normalizeLayoutToken(pass.Mode)
		if !validLayoutMode(mode) {
			issues = append(issues, fmt.Sprintf("layout_passes[%d] mode is %q, want supported Block layout mode", i, pass.Mode))
		}
		passModes[mode] = true
		if pass.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("layout_passes[%d] block_id must be positive", i))
		}
		if pass.Resolved.W <= 0 || pass.Resolved.H <= 0 || pass.Measured.W <= 0 || pass.Measured.H <= 0 {
			issues = append(issues, fmt.Sprintf("layout_passes[%d] resolved and measured dimensions must be positive", i))
		}
		if strings.TrimSpace(pass.Pass) == "" {
			issues = append(issues, fmt.Sprintf("layout_passes[%d] pass is required", i))
		}
		if !validChecksumLike(pass.Checksum) {
			issues = append(issues, fmt.Sprintf("layout_passes[%d] checksum must be sha256 evidence", i))
		}
		if pass.Resize && (pass.Input.W != pass.Resolved.W || pass.Input.H != pass.Resolved.H) {
			hasResize = true
		}
		if strings.Contains(normalizeLayoutToken(pass.Pass), "aspect") {
			hasAspect = true
		}
	}
	for _, mode := range []string{"stack", "row", "column", "absolute", "overlay", "grid", "dock", "scroll"} {
		if !passModes[mode] {
			issues = append(issues, fmt.Sprintf("layout_passes require %s mode evidence", mode))
		}
	}
	if !hasResize {
		issues = append(issues, "layout_passes require resize evidence with changed bounds")
	}
	if !hasAspect {
		issues = append(issues, "layout report requires aspect sizing evidence")
	}

	hasScroll := false
	for i, scroll := range report.LayoutScrolls {
		if scroll.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("layout_scrolls[%d] block_id must be positive", i))
		}
		if scroll.Viewport.W <= 0 || scroll.Viewport.H <= 0 || scroll.Content.W <= 0 || scroll.Content.H <= 0 {
			issues = append(issues, fmt.Sprintf("layout_scrolls[%d] viewport and content dimensions must be positive", i))
		}
		if scroll.Content.H <= scroll.Viewport.H {
			issues = append(issues, fmt.Sprintf("layout_scrolls[%d] content.h must exceed viewport.h for scroll evidence", i))
		}
		if scroll.OffsetY < 0 || scroll.MaxOffsetY < 0 || scroll.OffsetY > scroll.MaxOffsetY {
			issues = append(issues, fmt.Sprintf("layout_scrolls[%d] offset_y must be within max_offset_y", i))
		}
		if !scroll.Clipped {
			issues = append(issues, fmt.Sprintf("layout_scrolls[%d] clipped must be true", i))
		}
		if !validChecksumLike(scroll.Checksum) {
			issues = append(issues, fmt.Sprintf("layout_scrolls[%d] checksum must be sha256 evidence", i))
		}
		if scroll.Clipped && scroll.Content.H > scroll.Viewport.H && scroll.OffsetY <= scroll.MaxOffsetY {
			hasScroll = true
		}
	}
	if !hasScroll {
		issues = append(issues, "layout_scrolls require clipped scroll bounds evidence")
	}

	if !hasEventTargetKind(report.Events, "ScrollBlock", "scroll") {
		issues = append(issues, "block layout scroll requires scroll event targeted to ScrollBlock")
	}
	if !hasTransition(report.StateTransitions, "BlockLayoutApp", "width") || !hasTransition(report.StateTransitions, "ScrollBlock", "scroll_y") {
		issues = append(issues, "block layout requires resize width and scroll_y state transitions")
	}
	if len(report.Frames) < 2 || strings.TrimSpace(report.Frames[0].Checksum) == "" || report.Frames[0].Checksum == report.Frames[1].Checksum {
		issues = append(issues, "layout frame checksum evidence must show responsive layout change")
	}
	for _, required := range []string{
		"block layout nested row column",
		"block layout fit fill fixed min max",
		"block layout grid dock overlay scroll",
		"block layout clipping z-order",
		"block layout resize constraints",
		"block layout aspect density stable rounding",
		"block layout no css flexbox parity",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("layout report requires %s evidence", required))
		}
	}
	return issues
}

func validateBlockLayoutDensityEvidence(density BlockLayoutDensityReport) []string {
	var issues []string
	if density.TargetDPI < 96 {
		issues = append(issues, fmt.Sprintf("layout_density target_dpi is %d, want >= 96", density.TargetDPI))
	}
	if density.ScaleMilli < 1000 || density.ScaleMilli > 4000 {
		issues = append(issues, fmt.Sprintf("layout_density scale_milli is %d, want 1000..4000", density.ScaleMilli))
	}
	if density.BaseUnitPx <= 0 {
		issues = append(issues, "layout_density base_unit_px must be positive")
	}
	if normalizeLayoutToken(density.RoundingPolicy) != "integer_half_up_v1" {
		issues = append(issues, fmt.Sprintf("layout_density rounding_policy is %q, want integer-half-up-v1", density.RoundingPolicy))
	}
	if !density.PixelSnapping {
		issues = append(issues, "layout_density pixel_snapping must be true")
	}
	for _, breakpoint := range []string{"small", "medium", "large"} {
		if !layoutFeatureContains(density.Breakpoints, breakpoint) {
			issues = append(issues, fmt.Sprintf("layout_density breakpoints require %s", breakpoint))
		}
	}
	if !validChecksumLike(density.Checksum) {
		issues = append(issues, "layout_density checksum must be sha256 evidence")
	}
	return issues
}

func hasBlockLayoutEvidence(report Report) bool {
	return len(report.LayoutConstraints) > 0 ||
		len(report.LayoutPasses) > 0 ||
		len(report.LayoutScrolls) > 0 ||
		report.LayoutDensity != nil ||
		len(report.LayoutFeatures) > 0 ||
		strings.TrimSpace(report.LayoutQualityLevel) != "" ||
		report.LayoutUnsupportedCSSFlexbox
}

func validLayoutMode(value string) bool {
	switch normalizeLayoutToken(value) {
	case "fixed", "stack", "row", "column", "absolute", "overlay", "grid", "dock", "scroll":
		return true
	default:
		return false
	}
}

func validLayoutPolicy(value string) bool {
	switch normalizeLayoutToken(value) {
	case "fixed", "fit", "fill":
		return true
	default:
		return false
	}
}

func validLayoutAlign(value string) bool {
	switch normalizeLayoutToken(value) {
	case "start", "center", "end", "stretch":
		return true
	default:
		return false
	}
}

func validLayoutJustify(value string) bool {
	switch normalizeLayoutToken(value) {
	case "start", "center", "end", "space_between":
		return true
	default:
		return false
	}
}

func validLayoutOverflow(value string) bool {
	switch normalizeLayoutToken(value) {
	case "visible", "clip", "scroll":
		return true
	default:
		return false
	}
}

func layoutFeatureContains(features []string, want string) bool {
	want = normalizeLayoutToken(want)
	for _, feature := range features {
		if normalizeLayoutToken(feature) == want {
			return true
		}
	}
	return false
}

func normalizeLayoutToken(value string) string {
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(value, "-", "_")))
}
