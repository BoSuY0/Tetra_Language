package surface

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const VisualRegressionSchemaV1 = "tetra.surface.visual-regression.v1"

type VisualRegressionReport struct {
	Schema          string                               `json:"schema"`
	Status          string                               `json:"status"`
	GitHead         string                               `json:"git_head"`
	GoldenSet       string                               `json:"golden_set"`
	GoldenHash      string                               `json:"golden_hash"`
	RequiredTargets []string                             `json:"required_targets"`
	RequiredSources []string                             `json:"required_sources"`
	Apps            []VisualRegressionAppReport          `json:"apps"`
	NegativeGuards  VisualRegressionNegativeGuardsReport `json:"negative_guards"`
}

type VisualRegressionAppReport struct {
	Name         string                         `json:"name"`
	Source       string                         `json:"source"`
	ReferenceApp bool                           `json:"reference_app"`
	Targets      []VisualRegressionTargetReport `json:"targets"`
}

type VisualRegressionTargetReport struct {
	Target                string                        `json:"target"`
	RuntimeReport         string                        `json:"runtime_report"`
	RuntimeSchema         string                        `json:"runtime_schema"`
	GitHead               string                        `json:"git_head"`
	GoldenGitHead         string                        `json:"golden_git_head"`
	Renderer              string                        `json:"renderer"`
	ScreenshotOnly        bool                          `json:"screenshot_only,omitempty"`
	PNGArtifactSHA256     string                        `json:"png_artifact_sha256,omitempty"`
	BlockGraphEvidence    bool                          `json:"block_graph_evidence"`
	TokenThemeEvidence    bool                          `json:"token_theme_evidence"`
	LayoutEvidence        bool                          `json:"layout_evidence"`
	AccessibilityEvidence bool                          `json:"accessibility_evidence"`
	PerformanceEvidence   bool                          `json:"performance_evidence"`
	Frames                []VisualRegressionFrameReport `json:"frames"`
}

type VisualRegressionFrameReport struct {
	Order                 int    `json:"order"`
	Label                 string `json:"label"`
	Width                 int    `json:"width"`
	Height                int    `json:"height"`
	Stride                int    `json:"stride"`
	Checksum              string `json:"checksum"`
	GoldenChecksum        string `json:"golden_checksum"`
	ArtifactPath          string `json:"artifact_path"`
	ArtifactSHA256        string `json:"artifact_sha256"`
	ArtifactFormat        string `json:"artifact_format"`
	GoldenArtifactPath    string `json:"golden_artifact_path"`
	GoldenArtifactSHA256  string `json:"golden_artifact_sha256"`
	DiffPixels            int    `json:"diff_pixels"`
	DiffRatioMilli        int    `json:"diff_ratio_milli"`
	MaxChannelDelta       int    `json:"max_channel_delta"`
	TolerancePixels       int    `json:"tolerance_pixels"`
	ToleranceRatioMilli   int    `json:"tolerance_ratio_milli"`
	ToleranceChannelDelta int    `json:"tolerance_channel_delta"`
	Pass                  bool   `json:"pass"`
}

type VisualRegressionNegativeGuardsReport struct {
	ScreenshotOnlyRejected           bool `json:"screenshot_only_rejected"`
	StaleGoldenRejected              bool `json:"stale_golden_rejected"`
	MajorDriftRejected               bool `json:"major_drift_rejected"`
	MissingBlockGraphRejected        bool `json:"missing_block_graph_rejected"`
	MissingLayoutRejected            bool `json:"missing_layout_rejected"`
	MissingAccessibilityRejected     bool `json:"missing_accessibility_rejected"`
	MissingPerformanceRejected       bool `json:"missing_performance_rejected"`
	SelfGoldenRejected               bool `json:"self_golden_rejected"`
	MetadataChecksumRejected         bool `json:"metadata_checksum_rejected"`
	FixtureFrameOnlyRejected         bool `json:"fixture_frame_only_rejected"`
	MissingPNGOrRGBAArtifactRejected bool `json:"missing_png_or_rgba_artifact_rejected"`
}

func ValidateVisualReport(raw []byte) error {
	var report VisualRegressionReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return fmt.Errorf("decode Surface visual report: %w", err)
	}
	issues := validateVisualRegressionReport(report)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateVisualRegressionReport(report VisualRegressionReport) []string {
	var issues []string
	if report.Schema != VisualRegressionSchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %s", report.Schema, VisualRegressionSchemaV1))
	}
	if strings.TrimSpace(report.Status) != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if strings.TrimSpace(report.GitHead) == "" {
		issues = append(issues, "git_head is required")
	}
	if strings.TrimSpace(report.GoldenSet) == "" {
		issues = append(issues, "golden_set is required")
	}
	if !validChecksumLike(report.GoldenHash) {
		issues = append(issues, "golden_hash must be sha256 evidence")
	}
	if len(report.RequiredTargets) == 0 {
		issues = append(issues, "required_targets evidence is required")
	}
	if len(report.RequiredSources) == 0 {
		issues = append(issues, "required_sources evidence is required")
	}
	issues = append(issues, validateVisualNegativeGuards(report.NegativeGuards)...)
	if len(report.Apps) == 0 {
		issues = append(issues, "apps visual evidence is required")
	}
	sources := map[string]bool{}
	for i, app := range report.Apps {
		sources[normalizeEvidencePath(app.Source)] = true
		issues = append(issues, validateVisualApp(i, report.GitHead, report.RequiredTargets, app)...)
	}
	for _, required := range report.RequiredSources {
		if !sources[normalizeEvidencePath(required)] {
			issues = append(issues, fmt.Sprintf("missing required source %s", required))
		}
	}
	return issues
}

func validateVisualNegativeGuards(guards VisualRegressionNegativeGuardsReport) []string {
	var missing []string
	if !guards.ScreenshotOnlyRejected {
		missing = append(missing, "screenshot_only_rejected")
	}
	if !guards.StaleGoldenRejected {
		missing = append(missing, "stale_golden_rejected")
	}
	if !guards.MajorDriftRejected {
		missing = append(missing, "major_drift_rejected")
	}
	if !guards.MissingBlockGraphRejected {
		missing = append(missing, "missing_block_graph_rejected")
	}
	if !guards.MissingLayoutRejected {
		missing = append(missing, "missing_layout_rejected")
	}
	if !guards.MissingAccessibilityRejected {
		missing = append(missing, "missing_accessibility_rejected")
	}
	if !guards.MissingPerformanceRejected {
		missing = append(missing, "missing_performance_rejected")
	}
	if !guards.SelfGoldenRejected {
		missing = append(missing, "self_golden_rejected")
	}
	if !guards.MetadataChecksumRejected {
		missing = append(missing, "metadata_checksum_rejected")
	}
	if !guards.FixtureFrameOnlyRejected {
		missing = append(missing, "fixture_frame_only_rejected")
	}
	if !guards.MissingPNGOrRGBAArtifactRejected {
		missing = append(missing, "missing_png_or_rgba_artifact_rejected")
	}
	if len(missing) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("negative_guards missing %s", strings.Join(missing, ", "))}
}

func validateVisualApp(index int, reportGitHead string, requiredTargets []string, app VisualRegressionAppReport) []string {
	var issues []string
	prefix := fmt.Sprintf("apps[%d]", index)
	if strings.TrimSpace(app.Name) == "" {
		issues = append(issues, prefix+" name is required")
	}
	if strings.TrimSpace(app.Source) == "" {
		issues = append(issues, prefix+" source is required")
	}
	if !app.ReferenceApp {
		issues = append(issues, prefix+" reference_app must be true")
	}
	targets := map[string]bool{}
	for i, target := range app.Targets {
		name := normalizeVisualTarget(target.Target)
		targets[name] = true
		issues = append(issues, validateVisualTarget(fmt.Sprintf("%s.targets[%d]", prefix, i), reportGitHead, target)...)
	}
	for _, required := range requiredTargets {
		requiredName := normalizeVisualTarget(required)
		if !targets[requiredName] {
			issues = append(issues, fmt.Sprintf("%s missing required target %s", prefix, required))
		}
	}
	return issues
}

func validateVisualTarget(prefix string, reportGitHead string, target VisualRegressionTargetReport) []string {
	var issues []string
	if strings.TrimSpace(target.Target) == "" {
		issues = append(issues, prefix+" target is required")
	}
	if strings.TrimSpace(target.RuntimeReport) == "" {
		issues = append(issues, prefix+" runtime_report is required")
	}
	if target.RuntimeSchema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("%s runtime_schema is %q, want %s", prefix, target.RuntimeSchema, SchemaV1))
	}
	if strings.TrimSpace(target.GitHead) == "" {
		issues = append(issues, prefix+" git_head is required")
	}
	if strings.TrimSpace(target.GoldenGitHead) == "" {
		issues = append(issues, prefix+" golden_git_head is required")
	}
	if strings.TrimSpace(target.GitHead) != "" && strings.TrimSpace(target.GoldenGitHead) != "" && target.GitHead != target.GoldenGitHead {
		issues = append(issues, fmt.Sprintf("%s stale golden git_head %q, want %q", prefix, target.GoldenGitHead, target.GitHead))
	}
	if strings.TrimSpace(reportGitHead) != "" && strings.TrimSpace(target.GitHead) != "" && reportGitHead != target.GitHead {
		issues = append(issues, fmt.Sprintf("%s git_head %q does not match report git_head %q", prefix, target.GitHead, reportGitHead))
	}
	if strings.TrimSpace(target.Renderer) == "" {
		issues = append(issues, prefix+" renderer is required")
	}
	if target.ScreenshotOnly {
		issues = append(issues, prefix+" screenshot-only evidence is not sufficient")
	}
	if strings.TrimSpace(target.PNGArtifactSHA256) != "" && !validChecksumLike(target.PNGArtifactSHA256) {
		issues = append(issues, prefix+" png_artifact_sha256 must be sha256 evidence")
	}
	if !target.BlockGraphEvidence {
		issues = append(issues, prefix+" block graph evidence is required")
	}
	if !target.TokenThemeEvidence {
		issues = append(issues, prefix+" token/theme conformance evidence is required")
	}
	if !target.LayoutEvidence {
		issues = append(issues, prefix+" layout evidence is required")
	}
	if !target.AccessibilityEvidence {
		issues = append(issues, prefix+" accessibility evidence is required")
	}
	if !target.PerformanceEvidence {
		issues = append(issues, prefix+" performance evidence is required")
	}
	if len(target.Frames) == 0 {
		issues = append(issues, prefix+" frame diff evidence is required")
	}
	lastOrder := 0
	for i, frame := range target.Frames {
		if frame.Order <= lastOrder {
			issues = append(issues, fmt.Sprintf("%s.frames[%d] order %d is not strictly greater than previous order %d", prefix, i, frame.Order, lastOrder))
		}
		lastOrder = frame.Order
		issues = append(issues, validateVisualFrame(fmt.Sprintf("%s.frames[%d]", prefix, i), frame)...)
	}
	return issues
}

func validateVisualFrame(prefix string, frame VisualRegressionFrameReport) []string {
	var issues []string
	if strings.TrimSpace(frame.Label) == "" {
		issues = append(issues, prefix+" label is required")
	}
	if frame.Width <= 0 || frame.Height <= 0 || frame.Stride <= 0 {
		issues = append(issues, prefix+" dimensions and stride must be positive")
	}
	if !validChecksumLike(frame.Checksum) {
		issues = append(issues, prefix+" checksum must be sha256 evidence")
	}
	if !validChecksumLike(frame.GoldenChecksum) {
		issues = append(issues, prefix+" golden_checksum must be sha256 evidence")
	}
	artifactPath := normalizeEvidencePath(frame.ArtifactPath)
	goldenArtifactPath := normalizeEvidencePath(frame.GoldenArtifactPath)
	if artifactPath == "" {
		issues = append(issues, prefix+" artifact_path is required")
	}
	if goldenArtifactPath == "" {
		issues = append(issues, prefix+" golden_artifact_path is required")
	}
	if artifactPath != "" && goldenArtifactPath != "" && artifactPath == goldenArtifactPath {
		issues = append(issues, prefix+" self-golden artifact rejected")
	}
	if visualArtifactLooksLikeFixture(artifactPath) || visualArtifactLooksLikeFixture(goldenArtifactPath) {
		issues = append(issues, prefix+" fixture frame artifact is not product visual evidence")
	}
	format := strings.ToLower(strings.TrimSpace(frame.ArtifactFormat))
	if format != "rgba" && format != "png" {
		issues = append(issues, prefix+" artifact_format must be png or rgba")
	}
	if artifactPath != "" && !visualArtifactPathHasSupportedFormat(artifactPath) {
		issues = append(issues, prefix+" artifact_path must point to a png or rgba artifact")
	}
	if goldenArtifactPath != "" && !visualArtifactPathHasSupportedFormat(goldenArtifactPath) {
		issues = append(issues, prefix+" golden_artifact_path must point to a png or rgba artifact")
	}
	if !validChecksumLike(frame.ArtifactSHA256) {
		issues = append(issues, prefix+" artifact_sha256 must be sha256 evidence")
	}
	if !validChecksumLike(frame.GoldenArtifactSHA256) {
		issues = append(issues, prefix+" golden_artifact_sha256 must be sha256 evidence")
	}
	if validChecksumLike(frame.Checksum) && validChecksumLike(frame.ArtifactSHA256) && frame.Checksum != frame.ArtifactSHA256 {
		issues = append(issues, prefix+" artifact_sha256 must match checksum")
	}
	if validChecksumLike(frame.GoldenChecksum) && validChecksumLike(frame.GoldenArtifactSHA256) && frame.GoldenChecksum != frame.GoldenArtifactSHA256 {
		issues = append(issues, prefix+" golden_artifact_sha256 must match golden_checksum")
	}
	if frame.DiffPixels < 0 || frame.DiffRatioMilli < 0 || frame.MaxChannelDelta < 0 {
		issues = append(issues, prefix+" visual diff metrics must be non-negative")
	}
	if frame.TolerancePixels < 0 || frame.ToleranceRatioMilli < 0 || frame.ToleranceChannelDelta < 0 {
		issues = append(issues, prefix+" visual diff tolerances must be non-negative")
	}
	if frame.DiffPixels > frame.TolerancePixels ||
		frame.DiffRatioMilli > frame.ToleranceRatioMilli ||
		frame.MaxChannelDelta > frame.ToleranceChannelDelta ||
		!frame.Pass {
		issues = append(issues, fmt.Sprintf("%s visual drift exceeds tolerance", prefix))
	}
	return issues
}

func normalizeVisualTarget(value string) string {
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(value, "_", "-")))
}

func visualArtifactPathHasSupportedFormat(path string) bool {
	lower := strings.ToLower(strings.TrimSpace(path))
	return strings.HasSuffix(lower, ".rgba") || strings.HasSuffix(lower, ".png")
}

func visualArtifactLooksLikeFixture(path string) bool {
	lower := normalizeEvidencePath(strings.ToLower(path))
	return strings.Contains(lower, "/testdata/") ||
		strings.Contains(lower, "/fixtures/") ||
		strings.Contains(lower, "fixture-frame")
}
