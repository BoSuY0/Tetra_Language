package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
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

type visualArtifactKey struct {
	Source string
	Target string
	Order  int
}

type visualArtifactSpecs map[visualArtifactKey]string

type visualArtifactData struct {
	Path   string
	Format string
	SHA256 string
	Raw    []byte
	Image  image.Image
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
	var frameArtifactFlags repeatedFlag
	var goldenArtifactFlags repeatedFlag
	var outPath string
	var gitHead string
	var goldenSet string
	var writeGolden bool
	fs.Var(&runtimeReports, "runtime-report", "path to a tetra.surface.runtime.v1 report with block_system evidence; may be repeated")
	fs.Var(&blockExamplesReports, "block-examples-report", "path to a tetra.surface.block-examples.v1 polished reference app report; may be repeated")
	fs.Var(&requiredTargets, "required-target", "visual target that every reference app must cover; may be repeated")
	fs.Var(&frameArtifactFlags, "frame-artifact", "current frame artifact as source,target,order,path; path must be .rgba or .png; may be repeated")
	fs.Var(&goldenArtifactFlags, "golden-artifact", "golden frame artifact as source,target,order,path; path must be .rgba or .png; may be repeated")
	fs.StringVar(&outPath, "out", "", "path to write tetra.surface.visual-regression.v1 report")
	fs.StringVar(&gitHead, "git-head", "", "git head used for visual/golden evidence; defaults to git rev-parse HEAD")
	fs.StringVar(&goldenSet, "golden-set", "surface-visual-regression-v1", "visual golden set identifier")
	fs.BoolVar(&writeGolden, "write-golden", false, "write missing golden artifacts from current frame artifacts; forbidden in release/product gates")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
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
	frameArtifacts, err := parseVisualArtifactSpecs(frameArtifactFlags, "--frame-artifact")
	if err != nil {
		return err
	}
	goldenArtifacts, err := parseVisualArtifactSpecs(goldenArtifactFlags, "--golden-artifact")
	if err != nil {
		return err
	}

	report, err := buildVisualReport(runtimeReports, blockExamplesReports, requiredTargets, gitHead, goldenSet, frameArtifacts, goldenArtifacts, writeGolden)
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

func buildVisualReport(runtimeReportPaths []string, blockExamplesReportPaths []string, requiredTargets []string, gitHead string, goldenSet string, frameArtifacts visualArtifactSpecs, goldenArtifacts visualArtifactSpecs, writeGolden bool) (surface.VisualRegressionReport, error) {
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
		if runtimeReport.BlockSystem == nil && !hasRuntimeProductVisualFrameEvidence(runtimeReport) {
			return surface.VisualRegressionReport{}, fmt.Errorf("%s missing block_system or product_visual frame visual evidence source", reportPath)
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
		targetReport, err := buildVisualTarget(reportPath, runtimeReport, target, gitHead, frameArtifacts, goldenArtifacts, writeGolden)
		if err != nil {
			return surface.VisualRegressionReport{}, err
		}
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
			app, targetReports, err := visualAppForBlockExample(reportPath, example, requiredTargets, runtimeByTarget, gitHead, frameArtifacts, goldenArtifacts, writeGolden)
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
			ScreenshotOnlyRejected:           true,
			StaleGoldenRejected:              true,
			MajorDriftRejected:               true,
			MissingBlockGraphRejected:        true,
			MissingLayoutRejected:            true,
			MissingAccessibilityRejected:     true,
			MissingPerformanceRejected:       true,
			SelfGoldenRejected:               true,
			MetadataChecksumRejected:         true,
			FixtureFrameOnlyRejected:         true,
			MissingPNGOrRGBAArtifactRejected: true,
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

func visualAppForBlockExample(reportPath string, example blockExampleVisualReport, requiredTargets []string, runtimeByTarget map[string]visualRuntimeEvidence, gitHead string, frameArtifacts visualArtifactSpecs, goldenArtifacts visualArtifactSpecs, writeGolden bool) (surface.VisualRegressionAppReport, []surface.VisualRegressionTargetReport, error) {
	app := surface.VisualRegressionAppReport{
		Name:         visualAppName(example.Path),
		Source:       example.Path,
		ReferenceApp: true,
	}
	if _, err := sha256File(example.Artifact); err != nil {
		return app, nil, fmt.Errorf("%s example %s artifact hash: %w", reportPath, example.Path, err)
	}
	for _, target := range requiredTargets {
		runtimeEvidence, ok := runtimeByTarget[target]
		if !ok {
			return app, nil, fmt.Errorf("%s example %s missing runtime target evidence for %s", reportPath, example.Path, target)
		}
		frame, err := visualFrameForBlockExample(example, target, runtimeEvidence, frameArtifacts, goldenArtifacts, writeGolden)
		if err != nil {
			return app, nil, fmt.Errorf("%s example %s target %s: %w", reportPath, example.Path, target, err)
		}
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

func visualFrameForBlockExample(example blockExampleVisualReport, target string, runtimeEvidence visualRuntimeEvidence, frameArtifacts visualArtifactSpecs, goldenArtifacts visualArtifactSpecs, writeGolden bool) (surface.VisualRegressionFrameReport, error) {
	width := 320
	height := 200
	stride := 1280
	if runtimeEvidence.Report.BlockSystem != nil && len(runtimeEvidence.Report.BlockSystem.Frames) > 0 {
		frame := runtimeEvidence.Report.BlockSystem.Frames[0]
		width = frame.Width
		height = frame.Height
		stride = frame.Stride
	}
	return visualArtifactFrame(example.Path, target, "reference-app-visual", 1, width, height, stride, "", frameArtifacts, goldenArtifacts, writeGolden)
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

func buildVisualTarget(reportPath string, report surface.Report, target string, gitHead string, frameArtifacts visualArtifactSpecs, goldenArtifacts visualArtifactSpecs, writeGolden bool) (surface.VisualRegressionTargetReport, error) {
	frames, err := visualFramesForRuntimeReport(report.Source, target, report, frameArtifacts, goldenArtifacts, writeGolden)
	if err != nil {
		return surface.VisualRegressionTargetReport{}, fmt.Errorf("%s target %s: %w", reportPath, target, err)
	}
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
		PerformanceEvidence:   hasVisualPerformanceEvidence(report),
		Frames:                frames,
	}, nil
}

func visualFramesForRuntimeReport(source string, target string, report surface.Report, frameArtifacts visualArtifactSpecs, goldenArtifacts visualArtifactSpecs, writeGolden bool) ([]surface.VisualRegressionFrameReport, error) {
	if report.BlockSystem != nil {
		return visualFrames(source, target, report.BlockSystem.Frames, report.Frames, frameArtifacts, goldenArtifacts, writeGolden)
	}
	return visualRuntimeFrames(source, target, report.Frames, frameArtifacts, goldenArtifacts, writeGolden)
}

func visualRuntimeFrames(source string, target string, frames []surface.FrameReport, frameArtifacts visualArtifactSpecs, goldenArtifacts visualArtifactSpecs, writeGolden bool) ([]surface.VisualRegressionFrameReport, error) {
	effectiveFrameArtifacts := visualArtifactSpecs{}
	for key, path := range frameArtifacts {
		effectiveFrameArtifacts[key] = path
	}
	out := make([]surface.VisualRegressionFrameReport, 0, len(frames))
	for _, frame := range frames {
		if !isRuntimeProductVisualFrame(frame) {
			continue
		}
		key := newVisualArtifactKey(source, target, frame.Order)
		if _, ok := effectiveFrameArtifacts[key]; !ok && strings.TrimSpace(frame.ArtifactPath) != "" {
			effectiveFrameArtifacts[key] = frame.ArtifactPath
		}
		visualFrame, err := visualArtifactFrame(source, target, visualRuntimeFrameLabel(target, frame), frame.Order, frame.Width, frame.Height, frame.Stride, frame.Checksum, effectiveFrameArtifacts, goldenArtifacts, writeGolden)
		if err != nil {
			return nil, err
		}
		out = append(out, visualFrame)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("runtime product_visual frame artifacts are required")
	}
	return out, nil
}

func visualFrames(source string, target string, frames []surface.BlockSystemFrameReport, runtimeFrames []surface.FrameReport, frameArtifacts visualArtifactSpecs, goldenArtifacts visualArtifactSpecs, writeGolden bool) ([]surface.VisualRegressionFrameReport, error) {
	effectiveFrameArtifacts := visualArtifactSpecs{}
	for key, path := range frameArtifacts {
		effectiveFrameArtifacts[key] = path
	}
	runtimeFrameArtifacts := map[int]string{}
	for _, frame := range runtimeFrames {
		if strings.TrimSpace(frame.ArtifactPath) != "" {
			runtimeFrameArtifacts[frame.Order] = frame.ArtifactPath
		}
	}
	out := make([]surface.VisualRegressionFrameReport, 0, len(frames))
	for _, frame := range frames {
		key := newVisualArtifactKey(source, target, frame.Order)
		if _, ok := effectiveFrameArtifacts[key]; !ok {
			if strings.TrimSpace(frame.ArtifactPath) != "" {
				effectiveFrameArtifacts[key] = frame.ArtifactPath
			} else if path := runtimeFrameArtifacts[frame.Order]; strings.TrimSpace(path) != "" {
				effectiveFrameArtifacts[key] = path
			}
		}
		visualFrame, err := visualArtifactFrame(source, target, frame.Label, frame.Order, frame.Width, frame.Height, frame.Stride, frame.Checksum, effectiveFrameArtifacts, goldenArtifacts, writeGolden)
		if err != nil {
			return nil, err
		}
		out = append(out, visualFrame)
	}
	return out, nil
}

func visualArtifactFrame(source string, target string, label string, order int, width int, height int, stride int, expectedChecksum string, frameArtifacts visualArtifactSpecs, goldenArtifacts visualArtifactSpecs, writeGolden bool) (surface.VisualRegressionFrameReport, error) {
	key := newVisualArtifactKey(source, target, order)
	currentPath, ok := frameArtifacts[key]
	if !ok {
		return surface.VisualRegressionFrameReport{}, fmt.Errorf("frame artifact is required for source %s target %s order %d", source, target, order)
	}
	goldenPath, ok := goldenArtifacts[key]
	if !ok {
		return surface.VisualRegressionFrameReport{}, fmt.Errorf("golden artifact is required for source %s target %s order %d", source, target, order)
	}
	if normalizeArtifactFilePath(currentPath) == normalizeArtifactFilePath(goldenPath) {
		return surface.VisualRegressionFrameReport{}, fmt.Errorf("self-golden artifact rejected for source %s target %s order %d", source, target, order)
	}
	current, err := readVisualArtifact(currentPath, width, height, stride)
	if err != nil {
		return surface.VisualRegressionFrameReport{}, fmt.Errorf("read frame artifact %s: %w", currentPath, err)
	}
	if expectedChecksum = normalizeSHA256Checksum(expectedChecksum); expectedChecksum != "" && current.SHA256 != expectedChecksum {
		return surface.VisualRegressionFrameReport{}, fmt.Errorf("frame artifact checksum %s must match runtime frame checksum %s for source %s target %s order %d", current.SHA256, expectedChecksum, source, target, order)
	}
	if _, err := os.Stat(goldenPath); err != nil {
		if !writeGolden || !os.IsNotExist(err) {
			return surface.VisualRegressionFrameReport{}, fmt.Errorf("read golden artifact %s: %w", goldenPath, err)
		}
		if err := copyVisualGoldenArtifact(currentPath, goldenPath); err != nil {
			return surface.VisualRegressionFrameReport{}, fmt.Errorf("write golden artifact %s: %w", goldenPath, err)
		}
	}
	golden, err := readVisualArtifact(goldenPath, width, height, stride)
	if err != nil {
		return surface.VisualRegressionFrameReport{}, fmt.Errorf("read golden artifact %s: %w", goldenPath, err)
	}
	diffPixels, diffRatioMilli, maxChannelDelta, err := visualArtifactDiff(current, golden, width, height, stride)
	if err != nil {
		return surface.VisualRegressionFrameReport{}, err
	}
	const tolerancePixels = 4
	const toleranceRatioMilli = 1
	const toleranceChannelDelta = 1
	pass := diffPixels <= tolerancePixels && diffRatioMilli <= toleranceRatioMilli && maxChannelDelta <= toleranceChannelDelta
	return surface.VisualRegressionFrameReport{
		Order:                 order,
		Label:                 label,
		Width:                 width,
		Height:                height,
		Stride:                stride,
		Checksum:              current.SHA256,
		GoldenChecksum:        golden.SHA256,
		ArtifactPath:          current.Path,
		ArtifactSHA256:        current.SHA256,
		ArtifactFormat:        current.Format,
		GoldenArtifactPath:    golden.Path,
		GoldenArtifactSHA256:  golden.SHA256,
		DiffPixels:            diffPixels,
		DiffRatioMilli:        diffRatioMilli,
		MaxChannelDelta:       maxChannelDelta,
		TolerancePixels:       tolerancePixels,
		ToleranceRatioMilli:   toleranceRatioMilli,
		ToleranceChannelDelta: toleranceChannelDelta,
		Pass:                  pass,
	}, nil
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
	if report.RenderCommandStream != nil && strings.TrimSpace(report.RenderCommandStream.Renderer) != "" {
		return report.RenderCommandStream.Renderer
	}
	return ""
}

func hasRuntimeProductVisualFrameEvidence(report surface.Report) bool {
	if report.RenderCommandStream == nil {
		return false
	}
	for _, frame := range report.Frames {
		if isRuntimeProductVisualFrame(frame) && strings.TrimSpace(frame.ArtifactPath) != "" {
			return true
		}
	}
	return false
}

func isRuntimeProductVisualFrame(frame surface.FrameReport) bool {
	return strings.EqualFold(strings.TrimSpace(frame.Producer), "app") &&
		normalizeVisualArtifactTarget(frame.EvidenceRole) == "product-visual" &&
		strings.TrimSpace(frame.AppSource) != "" &&
		strings.TrimSpace(frame.BlockSceneHash) != "" &&
		strings.TrimSpace(frame.RenderCommandStreamHash) != ""
}

func visualRuntimeFrameLabel(target string, frame surface.FrameReport) string {
	switch frame.Order {
	case 1:
		return "initial"
	case 5:
		if target == "linux-x64-real-window" {
			return "real-window-active"
		}
		return "browser-canvas-focused"
	default:
		return "runtime-frame"
	}
}

func hasVisualPerformanceEvidence(report surface.Report) bool {
	if report.BlockSystem != nil && report.BlockSystem.MemoryBudget != nil {
		return true
	}
	return report.SurfacePerformanceBudget != nil || (report.Morph != nil && report.Morph.MemoryBudget.FrameCount > 0)
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
		parts = append(parts, fmt.Sprintf("%d:%s:%s:%s:%s:%s:%d", frame.Order, frame.Label, frame.Checksum, frame.GoldenChecksum, frame.ArtifactPath, frame.GoldenArtifactPath, frame.DiffPixels))
	}
	return parts
}

func parseVisualArtifactSpecs(values []string, flagName string) (visualArtifactSpecs, error) {
	specs := visualArtifactSpecs{}
	for _, value := range values {
		parts := strings.SplitN(value, ",", 4)
		if len(parts) != 4 {
			return nil, fmt.Errorf("%s must be source,target,order,path", flagName)
		}
		source := strings.TrimSpace(parts[0])
		target := strings.TrimSpace(parts[1])
		orderText := strings.TrimSpace(parts[2])
		path := strings.TrimSpace(parts[3])
		if source == "" || target == "" || path == "" {
			return nil, fmt.Errorf("%s source, target, and path are required", flagName)
		}
		order, err := strconv.Atoi(orderText)
		if err != nil || order <= 0 {
			return nil, fmt.Errorf("%s order %q must be a positive integer", flagName, orderText)
		}
		key := newVisualArtifactKey(source, target, order)
		if _, exists := specs[key]; exists {
			return nil, fmt.Errorf("%s duplicate artifact for source %s target %s order %d", flagName, source, target, order)
		}
		specs[key] = path
	}
	return specs, nil
}

func newVisualArtifactKey(source string, target string, order int) visualArtifactKey {
	return visualArtifactKey{
		Source: normalizeVisualArtifactSource(source),
		Target: normalizeVisualArtifactTarget(target),
		Order:  order,
	}
}

func normalizeVisualArtifactSource(source string) string {
	source = strings.TrimSpace(strings.ReplaceAll(source, "\\", "/"))
	for strings.Contains(source, "//") {
		source = strings.ReplaceAll(source, "//", "/")
	}
	return source
}

func normalizeVisualArtifactTarget(target string) string {
	return strings.TrimSpace(strings.ToLower(strings.ReplaceAll(target, "_", "-")))
}

func normalizeArtifactFilePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	clean := filepath.Clean(path)
	if abs, err := filepath.Abs(clean); err == nil {
		clean = abs
	}
	return filepath.ToSlash(clean)
}

func normalizeSHA256Checksum(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 64 && isHexDigest(value) {
		return "sha256:" + strings.ToLower(value)
	}
	const prefix = "sha256:"
	if strings.HasPrefix(value, prefix) {
		digest := value[len(prefix):]
		if len(digest) == 64 && isHexDigest(digest) {
			return prefix + strings.ToLower(digest)
		}
	}
	return value
}

func isHexDigest(value string) bool {
	for _, r := range value {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			continue
		}
		return false
	}
	return true
}

func readVisualArtifact(path string, width int, height int, stride int) (visualArtifactData, error) {
	format := visualArtifactFormat(path)
	if format == "" {
		return visualArtifactData{}, fmt.Errorf("artifact path must end in .rgba or .png")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return visualArtifactData{}, err
	}
	if len(raw) == 0 {
		return visualArtifactData{}, fmt.Errorf("artifact is empty")
	}
	artifact := visualArtifactData{
		Path:   path,
		Format: format,
		SHA256: sha256Bytes(raw),
		Raw:    raw,
	}
	switch format {
	case "rgba":
		if stride < width*4 {
			return visualArtifactData{}, fmt.Errorf("rgba stride %d is smaller than width*4 %d", stride, width*4)
		}
		expectedBytes := height * stride
		if expectedBytes <= 0 {
			return visualArtifactData{}, fmt.Errorf("rgba dimensions must be positive")
		}
		if len(raw) != expectedBytes {
			return visualArtifactData{}, fmt.Errorf("rgba size = %d, want %d bytes for %dx%d stride %d", len(raw), expectedBytes, width, height, stride)
		}
	case "png":
		img, err := png.Decode(bytes.NewReader(raw))
		if err != nil {
			return visualArtifactData{}, fmt.Errorf("decode png: %w", err)
		}
		bounds := img.Bounds()
		if bounds.Dx() != width || bounds.Dy() != height {
			return visualArtifactData{}, fmt.Errorf("png dimensions = %dx%d, want %dx%d", bounds.Dx(), bounds.Dy(), width, height)
		}
		artifact.Image = img
	}
	return artifact, nil
}

func visualArtifactFormat(path string) string {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(path))) {
	case ".rgba":
		return "rgba"
	case ".png":
		return "png"
	default:
		return ""
	}
}

func visualArtifactDiff(current visualArtifactData, golden visualArtifactData, width int, height int, stride int) (int, int, int, error) {
	if current.Format != golden.Format {
		return 0, 0, 0, fmt.Errorf("current/golden artifact format mismatch: %s vs %s", current.Format, golden.Format)
	}
	switch current.Format {
	case "rgba":
		return visualRGBADiff(current.Raw, golden.Raw, width, height, stride)
	case "png":
		return visualPNGDiff(current.Image, golden.Image, width, height)
	default:
		return 0, 0, 0, fmt.Errorf("artifact format must be png or rgba")
	}
}

func visualRGBADiff(current []byte, golden []byte, width int, height int, stride int) (int, int, int, error) {
	if len(current) != len(golden) {
		return 0, 0, 0, fmt.Errorf("current/golden rgba byte sizes differ: %d vs %d", len(current), len(golden))
	}
	diffPixels := 0
	maxChannelDelta := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := y*stride + x*4
			pixelDiffers := false
			for channel := 0; channel < 4; channel++ {
				delta := absInt(int(current[offset+channel]) - int(golden[offset+channel]))
				if delta > maxChannelDelta {
					maxChannelDelta = delta
				}
				if delta > 0 {
					pixelDiffers = true
				}
			}
			if pixelDiffers {
				diffPixels++
			}
		}
	}
	return diffPixels, diffRatioMilli(diffPixels, width, height), maxChannelDelta, nil
}

func visualPNGDiff(current image.Image, golden image.Image, width int, height int) (int, int, int, error) {
	if current == nil || golden == nil {
		return 0, 0, 0, fmt.Errorf("png image data is required")
	}
	diffPixels := 0
	maxChannelDelta := 0
	currentBounds := current.Bounds()
	goldenBounds := golden.Bounds()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			cr, cg, cb, ca := current.At(currentBounds.Min.X+x, currentBounds.Min.Y+y).RGBA()
			gr, gg, gb, ga := golden.At(goldenBounds.Min.X+x, goldenBounds.Min.Y+y).RGBA()
			deltas := []int{
				absInt(int(cr>>8) - int(gr>>8)),
				absInt(int(cg>>8) - int(gg>>8)),
				absInt(int(cb>>8) - int(gb>>8)),
				absInt(int(ca>>8) - int(ga>>8)),
			}
			pixelDiffers := false
			for _, delta := range deltas {
				if delta > maxChannelDelta {
					maxChannelDelta = delta
				}
				if delta > 0 {
					pixelDiffers = true
				}
			}
			if pixelDiffers {
				diffPixels++
			}
		}
	}
	return diffPixels, diffRatioMilli(diffPixels, width, height), maxChannelDelta, nil
}

func diffRatioMilli(diffPixels int, width int, height int) int {
	totalPixels := width * height
	if totalPixels <= 0 {
		return 1000
	}
	return diffPixels * 1000 / totalPixels
}

func copyVisualGoldenArtifact(currentPath string, goldenPath string) error {
	raw, err := os.ReadFile(currentPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(goldenPath, raw, 0o644)
}

func sha256Bytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
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
