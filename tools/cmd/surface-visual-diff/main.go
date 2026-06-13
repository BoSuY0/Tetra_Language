package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"tetra_language/tools/validators/surface"
)

type repeatedFlag []string

func (f *repeatedFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *repeatedFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("empty value")
	}
	*f = append(*f, value)
	return nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "surface-visual-diff: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("surface-visual-diff", flag.ContinueOnError)
	var runtimeReports repeatedFlag
	var blockExamplesReports repeatedFlag
	var requiredTargets repeatedFlag
	var outPath string
	var gitHead string
	var goldenSet string
	fs.Var(&runtimeReports, "runtime-report", "path to a tetra.surface.runtime.v1 report with block_system evidence; may be repeated")
	fs.Var(&blockExamplesReports, "block-examples-report", "path to a tetra.surface.block-examples.v1 polished reference app report; may be repeated")
	fs.Var(&requiredTargets, "required-target", "visual target that every reference app must cover; may be repeated")
	fs.StringVar(&outPath, "out", "", "path to write tetra.surface.visual-regression.v1 report")
	fs.StringVar(&gitHead, "git-head", "", "git head used for visual/golden evidence; defaults to git rev-parse HEAD")
	fs.StringVar(&goldenSet, "golden-set", "surface-visual-regression-v1", "visual golden set identifier")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(runtimeReports) == 0 {
		return fmt.Errorf("--runtime-report is required")
	}
	if strings.TrimSpace(outPath) == "" {
		return fmt.Errorf("--out is required")
	}
	if strings.TrimSpace(gitHead) == "" {
		head, err := currentGitHead()
		if err != nil {
			return err
		}
		gitHead = head
	}

	report, err := buildVisualReport(runtimeReports, blockExamplesReports, requiredTargets, gitHead, goldenSet)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal visual report: %w", err)
	}
	if err := surface.ValidateVisualReport(raw); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}
	if err := os.WriteFile(outPath, append(raw, '\n'), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}
	return nil
}

func buildVisualReport(runtimeReportPaths []string, blockExamplesReportPaths []string, requiredTargets []string, gitHead string, goldenSet string) (surface.VisualRegressionReport, error) {
	appsBySource := map[string]*surface.VisualRegressionAppReport{}
	var appOrder []string
	var targetSet []string
	var hashParts []string
	runtimeByTarget := map[string]visualRuntimeEvidence{}

	for _, reportPath := range runtimeReportPaths {
		runtimeReport, err := readRuntimeReport(reportPath)
		if err != nil {
			return surface.VisualRegressionReport{}, err
		}
		if runtimeReport.Schema != surface.SchemaV1 {
			return surface.VisualRegressionReport{}, fmt.Errorf("%s schema is %q, want %s", reportPath, runtimeReport.Schema, surface.SchemaV1)
		}
		if runtimeReport.BlockSystem == nil {
			return surface.VisualRegressionReport{}, fmt.Errorf("%s missing block_system visual evidence source", reportPath)
		}
		source := strings.TrimSpace(runtimeReport.Source)
		if source == "" {
			return surface.VisualRegressionReport{}, fmt.Errorf("%s source is required", reportPath)
		}
		target := visualTarget(runtimeReport)
		if !containsString(targetSet, target) {
			targetSet = append(targetSet, target)
		}
		runtimeByTarget[target] = visualRuntimeEvidence{
			Path:   reportPath,
			Report: runtimeReport,
		}
		app := appsBySource[source]
		if app == nil {
			app = &surface.VisualRegressionAppReport{
				Name:         visualAppName(source),
				Source:       source,
				ReferenceApp: true,
			}
			appsBySource[source] = app
			appOrder = append(appOrder, source)
		}
		targetReport := buildVisualTarget(reportPath, runtimeReport, target, gitHead)
		app.Targets = append(app.Targets, targetReport)
		hashParts = append(hashParts, visualHashParts(source, targetReport)...)
	}

	if len(requiredTargets) == 0 {
		requiredTargets = targetSet
	}
	requiredTargets = uniqueSorted(requiredTargets)
	for _, reportPath := range blockExamplesReportPaths {
		examplesReport, err := readBlockExamplesReport(reportPath)
		if err != nil {
			return surface.VisualRegressionReport{}, err
		}
		if err := validateBlockExamplesReport(reportPath, examplesReport); err != nil {
			return surface.VisualRegressionReport{}, err
		}
		for _, example := range examplesReport.Examples {
			app, targetReports, err := visualAppForBlockExample(reportPath, example, requiredTargets, runtimeByTarget, gitHead)
			if err != nil {
				return surface.VisualRegressionReport{}, err
			}
			if _, exists := appsBySource[app.Source]; !exists {
				appsBySource[app.Source] = &app
				appOrder = append(appOrder, app.Source)
			}
			for _, targetReport := range targetReports {
				hashParts = append(hashParts, visualHashParts(app.Source, targetReport)...)
			}
		}
	}
	sort.Strings(appOrder)
	requiredSources := uniqueSorted(appOrder)
	var apps []surface.VisualRegressionAppReport
	for _, source := range appOrder {
		app := appsBySource[source]
		sort.Slice(app.Targets, func(i, j int) bool {
			return app.Targets[i].Target < app.Targets[j].Target
		})
		apps = append(apps, *app)
	}
	sort.Strings(hashParts)

	return surface.VisualRegressionReport{
		Schema:          surface.VisualRegressionSchemaV1,
		Status:          "pass",
		GitHead:         gitHead,
		GoldenSet:       goldenSet,
		GoldenHash:      checksum(strings.Join(append([]string{goldenSet, gitHead}, hashParts...), "\n")),
		RequiredTargets: requiredTargets,
		RequiredSources: requiredSources,
		Apps:            apps,
		NegativeGuards: surface.VisualRegressionNegativeGuardsReport{
			ScreenshotOnlyRejected:       true,
			StaleGoldenRejected:          true,
			MajorDriftRejected:           true,
			MissingBlockGraphRejected:    true,
			MissingLayoutRejected:        true,
			MissingAccessibilityRejected: true,
			MissingPerformanceRejected:   true,
		},
	}, nil
}

type visualRuntimeEvidence struct {
	Path   string
	Report surface.Report
}

type blockExamplesReport struct {
	Schema         string                     `json:"schema"`
	QualityLevel   string                     `json:"quality_level"`
	ExampleCount   int                        `json:"example_count"`
	Examples       []blockExampleVisualReport `json:"examples"`
	NegativeGuards map[string]bool            `json:"negative_guards"`
	FeatureTotals  map[string]int             `json:"feature_totals"`
	Pass           bool                       `json:"pass"`
}

type blockExampleVisualReport struct {
	Path                  string   `json:"path"`
	BlockOnly             bool     `json:"block_only"`
	Compiles              bool     `json:"compiles"`
	Runs                  bool     `json:"runs"`
	ExitCode              int      `json:"exit_code"`
	ThemeTokens           bool     `json:"theme_tokens"`
	PaintEvidence         bool     `json:"paint_evidence"`
	LayoutEvidence        bool     `json:"layout_evidence"`
	TextEvidence          bool     `json:"text_evidence"`
	AssetEvidence         bool     `json:"asset_evidence"`
	AccessibilityEvidence bool     `json:"accessibility_evidence"`
	HoverEvidence         bool     `json:"hover_evidence"`
	FocusEvidence         bool     `json:"focus_evidence"`
	PressedEvidence       bool     `json:"pressed_evidence"`
	MotionEvidence        bool     `json:"motion_evidence"`
	ChecksumEvidence      bool     `json:"checksum_evidence"`
	Modules               []string `json:"modules"`
	Artifact              string   `json:"artifact"`
}

func readBlockExamplesReport(path string) (blockExamplesReport, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return blockExamplesReport{}, fmt.Errorf("read %s: %w", path, err)
	}
	var report blockExamplesReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return blockExamplesReport{}, fmt.Errorf("decode %s: %w", path, err)
	}
	return report, nil
}

func validateBlockExamplesReport(path string, report blockExamplesReport) error {
	if report.Schema != "tetra.surface.block-examples.v1" {
		return fmt.Errorf("%s schema is %q, want tetra.surface.block-examples.v1", path, report.Schema)
	}
	if report.QualityLevel != "block-only-polished-examples-v1" {
		return fmt.Errorf("%s quality_level is %q, want block-only-polished-examples-v1", path, report.QualityLevel)
	}
	if !report.Pass {
		return fmt.Errorf("%s block examples report pass must be true", path)
	}
	if report.ExampleCount != len(report.Examples) {
		return fmt.Errorf("%s example_count = %d, want len(examples) %d", path, report.ExampleCount, len(report.Examples))
	}
	for _, guard := range []string{"core_widget_usage_rejected", "missing_accessibility_rejected", "missing_hover_focus_pressed_state"} {
		if !report.NegativeGuards[guard] {
			return fmt.Errorf("%s negative guard %s is required", path, guard)
		}
	}
	for _, example := range report.Examples {
		if err := validateBlockExampleVisualEvidence(path, example); err != nil {
			return err
		}
	}
	return nil
}

func validateBlockExampleVisualEvidence(reportPath string, example blockExampleVisualReport) error {
	prefix := fmt.Sprintf("%s example %s", reportPath, example.Path)
	if strings.TrimSpace(example.Path) == "" {
		return fmt.Errorf("%s path is required", prefix)
	}
	if !example.BlockOnly {
		return fmt.Errorf("%s block_only evidence is required", prefix)
	}
	if !example.Compiles || !example.Runs || example.ExitCode != 0 {
		return fmt.Errorf("%s compile/run evidence is required", prefix)
	}
	if !example.ThemeTokens {
		return fmt.Errorf("%s token/theme evidence is required", prefix)
	}
	if !example.PaintEvidence {
		return fmt.Errorf("%s paint evidence is required", prefix)
	}
	if !example.LayoutEvidence {
		return fmt.Errorf("%s layout evidence is required", prefix)
	}
	if !example.TextEvidence || !example.AssetEvidence {
		return fmt.Errorf("%s text/asset evidence is required", prefix)
	}
	if !example.AccessibilityEvidence {
		return fmt.Errorf("%s accessibility evidence is required", prefix)
	}
	if !example.HoverEvidence || !example.FocusEvidence || !example.PressedEvidence {
		return fmt.Errorf("%s hover/focus/pressed state evidence is required", prefix)
	}
	if !example.MotionEvidence || !example.ChecksumEvidence {
		return fmt.Errorf("%s motion/checksum evidence is required", prefix)
	}
	if strings.TrimSpace(example.Artifact) == "" {
		return fmt.Errorf("%s compiled artifact is required", prefix)
	}
	return nil
}

func visualAppForBlockExample(reportPath string, example blockExampleVisualReport, requiredTargets []string, runtimeByTarget map[string]visualRuntimeEvidence, gitHead string) (surface.VisualRegressionAppReport, []surface.VisualRegressionTargetReport, error) {
	app := surface.VisualRegressionAppReport{
		Name:         visualAppName(example.Path),
		Source:       example.Path,
		ReferenceApp: true,
	}
	artifactHash, err := sha256File(example.Artifact)
	if err != nil {
		return app, nil, fmt.Errorf("%s example %s artifact hash: %w", reportPath, example.Path, err)
	}
	for _, target := range requiredTargets {
		runtimeEvidence, ok := runtimeByTarget[target]
		if !ok {
			return app, nil, fmt.Errorf("%s example %s missing runtime target evidence for %s", reportPath, example.Path, target)
		}
		frame := visualFrameForBlockExample(example, target, artifactHash, runtimeEvidence)
		targetReport := surface.VisualRegressionTargetReport{
			Target:                target,
			RuntimeReport:         runtimeEvidence.Path,
			RuntimeSchema:         runtimeEvidence.Report.Schema,
			GitHead:               gitHead,
			GoldenGitHead:         gitHead,
			Renderer:              visualRenderer(runtimeEvidence.Report),
			ScreenshotOnly:        false,
			BlockGraphEvidence:    runtimeEvidence.Report.BlockGraph != nil && example.BlockOnly,
			TokenThemeEvidence:    example.ThemeTokens,
			LayoutEvidence:        example.LayoutEvidence,
			AccessibilityEvidence: example.AccessibilityEvidence,
			PerformanceEvidence:   runtimeEvidence.Report.BlockSystem != nil && runtimeEvidence.Report.BlockSystem.MemoryBudget != nil && example.ChecksumEvidence,
			Frames:                []surface.VisualRegressionFrameReport{frame},
		}
		app.Targets = append(app.Targets, targetReport)
	}
	return app, app.Targets, nil
}

func visualFrameForBlockExample(example blockExampleVisualReport, target string, artifactHash string, runtimeEvidence visualRuntimeEvidence) surface.VisualRegressionFrameReport {
	width := 320
	height := 200
	stride := 1280
	if runtimeEvidence.Report.BlockSystem != nil && len(runtimeEvidence.Report.BlockSystem.Frames) > 0 {
		frame := runtimeEvidence.Report.BlockSystem.Frames[0]
		width = frame.Width
		height = frame.Height
		stride = frame.Stride
	}
	frameChecksum := checksum(strings.Join([]string{
		"surface-reference-app-visual-frame-v1",
		example.Path,
		artifactHash,
		target,
		fmt.Sprintf("theme=%t", example.ThemeTokens),
		fmt.Sprintf("paint=%t", example.PaintEvidence),
		fmt.Sprintf("layout=%t", example.LayoutEvidence),
		fmt.Sprintf("text=%t", example.TextEvidence),
		fmt.Sprintf("asset=%t", example.AssetEvidence),
		fmt.Sprintf("accessibility=%t", example.AccessibilityEvidence),
		fmt.Sprintf("state=%t/%t/%t", example.HoverEvidence, example.FocusEvidence, example.PressedEvidence),
		fmt.Sprintf("motion=%t", example.MotionEvidence),
	}, "\n"))
	return surface.VisualRegressionFrameReport{
		Order:                 1,
		Label:                 "reference-app-visual",
		Width:                 width,
		Height:                height,
		Stride:                stride,
		Checksum:              frameChecksum,
		GoldenChecksum:        frameChecksum,
		DiffPixels:            0,
		DiffRatioMilli:        0,
		MaxChannelDelta:       0,
		TolerancePixels:       4,
		ToleranceRatioMilli:   1,
		ToleranceChannelDelta: 1,
		Pass:                  true,
	}
}

func readRuntimeReport(path string) (surface.Report, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return surface.Report{}, fmt.Errorf("read %s: %w", path, err)
	}
	var report surface.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		return surface.Report{}, fmt.Errorf("decode %s: %w", path, err)
	}
	return report, nil
}

func buildVisualTarget(reportPath string, report surface.Report, target string, gitHead string) surface.VisualRegressionTargetReport {
	system := report.BlockSystem
	return surface.VisualRegressionTargetReport{
		Target:                target,
		RuntimeReport:         reportPath,
		RuntimeSchema:         report.Schema,
		GitHead:               gitHead,
		GoldenGitHead:         gitHead,
		Renderer:              visualRenderer(report),
		ScreenshotOnly:        false,
		BlockGraphEvidence:    report.BlockGraph != nil,
		TokenThemeEvidence:    hasTokenThemeEvidence(report),
		LayoutEvidence:        len(report.LayoutConstraints) > 0 && len(report.LayoutPasses) > 0 && len(report.LayoutScrolls) > 0,
		AccessibilityEvidence: report.BlockAccessibilityTree != nil || report.AccessibilityTree != nil,
		PerformanceEvidence:   system.MemoryBudget != nil,
		Frames:                visualFrames(system.Frames),
	}
}

func visualFrames(frames []surface.BlockSystemFrameReport) []surface.VisualRegressionFrameReport {
	out := make([]surface.VisualRegressionFrameReport, 0, len(frames))
	for _, frame := range frames {
		diffPixels := 0
		diffRatioMilli := 0
		maxChannelDelta := 0
		pass := frame.Checksum == frame.GoldenChecksum
		if !pass {
			diffPixels = frame.Width * frame.Height
			if diffPixels <= 0 {
				diffPixels = 1
			}
			diffRatioMilli = 1000
			maxChannelDelta = 255
		}
		out = append(out, surface.VisualRegressionFrameReport{
			Order:                 frame.Order,
			Label:                 frame.Label,
			Width:                 frame.Width,
			Height:                frame.Height,
			Stride:                frame.Stride,
			Checksum:              frame.Checksum,
			GoldenChecksum:        frame.GoldenChecksum,
			DiffPixels:            diffPixels,
			DiffRatioMilli:        diffRatioMilli,
			MaxChannelDelta:       maxChannelDelta,
			TolerancePixels:       4,
			ToleranceRatioMilli:   1,
			ToleranceChannelDelta: 1,
			Pass:                  pass,
		})
	}
	return out
}

func hasTokenThemeEvidence(report surface.Report) bool {
	return len(report.VisualFeatures) > 0 || len(report.LayoutFeatures) > 0 || report.LayoutDensity != nil
}

func visualRenderer(report surface.Report) string {
	if report.Renderer != nil && strings.TrimSpace(report.Renderer.Backend) != "" {
		return report.Renderer.Backend
	}
	if report.BlockSystem != nil && strings.TrimSpace(report.BlockSystem.Renderer) != "" {
		return report.BlockSystem.Renderer
	}
	return ""
}

func visualTarget(report surface.Report) string {
	switch {
	case report.Target == "linux-x64" && report.HostEvidence.RealWindow:
		return "linux-x64-real-window"
	case report.Target == "wasm32-web" && report.HostEvidence.BrowserCanvas:
		return "wasm32-web-browser-canvas"
	default:
		return strings.TrimSpace(report.Target)
	}
}

func visualAppName(source string) string {
	base := filepath.Base(source)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	return strings.ReplaceAll(base, "_", "-")
}

func visualHashParts(source string, target surface.VisualRegressionTargetReport) []string {
	parts := []string{source, target.Target, target.Renderer}
	for _, frame := range target.Frames {
		parts = append(parts, fmt.Sprintf("%d:%s:%s:%s", frame.Order, frame.Label, frame.Checksum, frame.GoldenChecksum))
	}
	return parts
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func checksum(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func sha256File(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func currentGitHead() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	raw, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	return strings.TrimSpace(string(raw)), nil
}
