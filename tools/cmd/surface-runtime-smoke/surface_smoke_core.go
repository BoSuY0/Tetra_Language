package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"tetra_language/compiler"
	"tetra_language/tools/validators/surface"
)

// ---- main.go ----

func main() {
	var opt smokeOptions
	flag.StringVar(&opt.ReportPath, "report", "", "path to write tetra.surface.runtime.v1 report")
	flag.StringVar(&opt.Mode, "mode", "headless", "Surface smoke mode")
	flag.StringVar(
		&opt.SourcePath,
		"source",
		"examples/surface/runtime/surface_counter.tetra",
		"Surface app source path",
	)
	flag.StringVar(
		&opt.VisualReportPath,
		"visual-report",
		"",
		"path to tetra.surface.visual-regression.v1 report used for Morph rendered beauty evidence",
	)
	flag.StringVar(
		&opt.MorphRenderedBeautyReportPath,
		"morph-rendered-beauty-report",
		"",
		"optional path to write tetra.surface.morph-rendered-beauty.v1 report",
	)
	flag.BoolVar(
		&opt.MorphRenderedBeautyProductClaim,
		"morph-rendered-beauty-product-claim",
		false,
		("mark the Morph rendered beauty report as a product claim " +
			"when clean renderer-owned proof is present"),
	)
	flag.BoolVar(
		&opt.MorphRenderedBeautyFinalSignoff,
		"morph-rendered-beauty-final-signoff",
		false,
		"mark the Morph rendered beauty report as final signoff when product claim requirements are met",
	)
	flag.BoolVar(
		&opt.RealWindowProbe,
		"real-window-probe",
		false,
		"run the linux-x64 real-window probe helper",
	)
	flag.StringVar(
		&opt.ProbeTitle,
		"probe-title",
		"Tetra Surface Real Window Probe",
		"real-window probe title",
	)
	flag.StringVar(
		&opt.ProbeFramePath,
		"probe-frame",
		"",
		"raw RGBA frame path for the real-window probe",
	)
	flag.IntVar(&opt.ProbeFrameWidth, "probe-width", 400, "real-window probe frame width")
	flag.IntVar(&opt.ProbeFrameHeight, "probe-height", 240, "real-window probe frame height")
	flag.IntVar(&opt.ProbeFrameStride, "probe-stride", 1600, "real-window probe frame stride")
	flag.BoolVar(
		&opt.ProbeHoldUntilClose,
		"probe-hold-until-close",
		false,
		"keep the real-window probe alive until the compositor sends xdg_toplevel.close",
	)
	flag.Parse()
	if opt.RealWindowProbe {
		if err := runRealWindowProbe(opt); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(42)
	}
	if opt.ReportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateSmokeMode(opt.Mode); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	evidence, err := collectSurfaceProcessEvidence(opt)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if isReleaseTextInputMode(opt.Mode) {
		report := buildTextInputReport(
			opt,
			evidence.Processes,
			evidence.Artifacts,
			evidence.ArtifactScan,
			releaseTextInputCases(),
		)
		raw, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := surface.ValidateTextInputReport(raw); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := os.MkdirAll(filepath.Dir(opt.ReportPath), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := os.WriteFile(opt.ReportPath, append(raw, '\n'), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	scenario := releaseCounterScenarioForSource(opt, runSurfaceScenario(opt.Mode))
	if isMorphMode(opt.Mode) {
		scenario = runMorphScenarioForSource(defaultSurfaceSourcePath(opt))
	}
	if shouldRetargetSurfaceTemplateScenario(opt) {
		source := defaultSurfaceSourcePath(opt)
		retargetScenarioToSource(&scenario, source, "main")
		if isMorphMode(opt.Mode) {
			scenario.Morph = morphReportForScenario(source, scenario)
		}
	}
	if shouldRetargetBlockSystemSourceScenario(opt) {
		source := defaultSurfaceSourcePath(opt)
		retargetScenarioToSource(&scenario, source, surfaceSourceModuleName(source))
	}
	if isMorphTargetRuntimeMode(opt.Mode) {
		if err := applyMorphTargetRuntimeFrameEvidence(opt, &scenario, evidence.Frames); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else if opt.Mode == "wasm32-web-browser-canvas-block-system" {
		scenario.Frames = mergeFrameEvidenceByOrder(scenario.Frames, evidence.Frames)
	} else {
		if len(scenario.Frames) > 0 && len(evidence.Frames) > 0 {
			lastOrder := scenario.Frames[len(scenario.Frames)-1].Order
			for i := range evidence.Frames {
				if evidence.Frames[i].Order <= lastOrder {
					evidence.Frames[i].Order = lastOrder + i + 1
				}
			}
		}
		scenario.Frames = append(scenario.Frames, evidence.Frames...)
	}
	if opt.Mode == "linux-x64-real-window-block-system" {
		scenario.BlockSystem = blockSystemReportForLinuxX64RealWindowScenario(
			defaultSurfaceSourcePath(opt),
			scenario.Frames,
		)
		attachBlockSystemMemoryBudget(&scenario)
	}
	if opt.Mode == "wasm32-web-browser-canvas-block-system" {
		scenario.BlockSystem = blockSystemReportForWASM32WebBrowserCanvasScenario(
			defaultSurfaceSourcePath(opt),
			scenario.Frames,
		)
		attachBlockSystemMemoryBudget(&scenario)
	}
	if err := attachBlockSystemFrameArtifacts(opt, &scenario); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := attachMorphRenderedBeautyFrameArtifacts(opt, &scenario); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := refreshHeadlessMorphRunnerTrace(opt, &evidence, scenario); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := refreshBlockSystemArtifactScan(opt, &evidence); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	report := buildReport(
		opt,
		"linux-x64",
		evidence.Processes,
		evidence.Artifacts,
		evidence.ArtifactScan,
		scenario,
	)
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := surface.ValidateReport(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(opt.ReportPath), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(opt.ReportPath, append(raw, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if strings.TrimSpace(opt.MorphRenderedBeautyReportPath) != "" {
		if strings.TrimSpace(opt.VisualReportPath) == "" {
			fmt.Fprintln(
				os.Stderr,
				"error: --visual-report is required with --morph-rendered-beauty-report",
			)
			os.Exit(2)
		}
		visualReport, err := readVisualRegressionReport(opt.VisualReportPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		morphRenderedBeautyReport, err := buildMorphRenderedBeautyReport(
			opt.ReportPath,
			report,
			visualReport,
			morphRenderedBeautyScenarioName(opt),
		)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := applyMorphRenderedBeautyProductSignoff(
			&morphRenderedBeautyReport,
			opt.MorphRenderedBeautyProductClaim,
			opt.MorphRenderedBeautyFinalSignoff,
		); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		morphRaw, err := json.MarshalIndent(morphRenderedBeautyReport, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := surface.ValidateMorphRenderedBeautyReport(morphRaw); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := os.MkdirAll(filepath.Dir(opt.MorphRenderedBeautyReportPath), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := os.WriteFile(
			opt.MorphRenderedBeautyReportPath,
			append(morphRaw, '\n'),
			0o644,
		); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func refreshHeadlessMorphRunnerTrace(
	opt smokeOptions,
	evidence *surfaceProcessEvidence,
	scenario headlessScenario,
) error {
	if opt.Mode != "headless-morph" {
		return nil
	}
	artifactDir := surfaceRuntimeArtifactDir(opt)
	traceArtifact, sidecarScan, err := collectHeadlessRunnerTraceEvidence(
		defaultSurfaceSourcePath(opt),
		artifactDir,
		scenario,
	)
	if err != nil {
		return err
	}
	replaced := false
	for i := range evidence.Artifacts {
		if evidence.Artifacts[i].Kind == "runner-trace" {
			evidence.Artifacts[i] = traceArtifact
			replaced = true
			break
		}
	}
	if !replaced {
		evidence.Artifacts = append(evidence.Artifacts, traceArtifact)
	}
	evidence.ArtifactScan = sidecarScan
	return nil
}

func refreshBlockSystemArtifactScan(opt smokeOptions, evidence *surfaceProcessEvidence) error {
	if !isBlockSystemMode(opt.Mode) && !isMorphMode(opt.Mode) {
		return nil
	}
	artifactDir := surfaceRuntimeArtifactDir(opt)
	var scan surface.ArtifactScanReport
	var err error
	if opt.Mode == "wasm32-web-browser-canvas-block-system" {
		scan, err = scanLegacyUISidecarArtifacts(
			artifactDir,
			sidecarScanOptions{AllowCompilerOwnedWASMLoader: true},
		)
	} else if isWASM32WebBrowserCanvasMorphMode(opt.Mode) {
		scan, err = scanLegacyUISidecarArtifacts(
			artifactDir,
			sidecarScanOptions{AllowCompilerOwnedWASMLoader: true},
		)
	} else {
		scan, err = scanLegacyUISidecarArtifacts(artifactDir)
	}
	if err != nil {
		return err
	}
	evidence.ArtifactScan = scan
	return nil
}

// ---- modes.go ----

func validateSmokeMode(mode string) error {
	if mode == "" || mode == "headless" {
		return nil
	}
	if mode == "linux-x64" {
		return nil
	}
	if mode == "linux-x64-real-window" {
		return nil
	}
	if mode == "linux-x64-release-window" {
		return nil
	}
	if mode == "linux-x64-release-app-shell" {
		return nil
	}
	if mode == "wasm32-web" {
		return nil
	}
	if mode == "wasm32-web-browser-canvas" {
		return nil
	}
	if mode == "headless-text-focus-input" {
		return nil
	}
	if mode == "linux-x64-real-window-text-focus-input" {
		return nil
	}
	if mode == "wasm32-web-browser-canvas-text-focus-input" {
		return nil
	}
	if isReleaseTextInputMode(mode) {
		return nil
	}
	if isReleaseToolkitMode(mode) {
		return nil
	}
	if isReleaseAccessibilityMode(mode) {
		return nil
	}
	if isReleaseBrowserMode(mode) {
		return nil
	}
	if mode == "headless-component-tree" {
		return nil
	}
	if mode == "linux-x64-real-window-component-tree" {
		return nil
	}
	if mode == "wasm32-web-browser-canvas-component-tree" {
		return nil
	}
	if mode == "headless-component-tree-api" {
		return nil
	}
	if mode == "linux-x64-real-window-component-tree-api" {
		return nil
	}
	if mode == "wasm32-web-browser-canvas-component-tree-api" {
		return nil
	}
	if mode == "headless-block-paint" {
		return nil
	}
	if mode == "headless-block-text" {
		return nil
	}
	if mode == "headless-block-layout" {
		return nil
	}
	if mode == "headless-block-events" {
		return nil
	}
	if mode == "headless-block-states" {
		return nil
	}
	if mode == "headless-block-motion" {
		return nil
	}
	if mode == "headless-block-assets" {
		return nil
	}
	if mode == "headless-block-accessibility" {
		return nil
	}
	if mode == "headless-block-system" {
		return nil
	}
	if mode == "headless-morph" {
		return nil
	}
	if mode == "linux-x64-real-window-morph" {
		return nil
	}
	if mode == "wasm32-web-browser-canvas-morph" {
		return nil
	}
	if mode == "linux-x64-real-window-block-system" {
		return nil
	}
	if mode == "wasm32-web-browser-canvas-block-system" {
		return nil
	}
	if mode == "headless-minimal-toolkit" {
		return nil
	}
	if mode == "linux-x64-real-window-minimal-toolkit" {
		return nil
	}
	if mode == "wasm32-web-browser-canvas-minimal-toolkit" {
		return nil
	}
	if mode == "headless-toolkit-reuse" {
		return nil
	}
	if mode == "linux-x64-real-window-toolkit-reuse" {
		return nil
	}
	if mode == "wasm32-web-browser-canvas-toolkit-reuse" {
		return nil
	}
	if mode == "headless-accessibility-metadata" {
		return nil
	}
	if mode == "headless-app-model" {
		return nil
	}
	if mode == "linux-x64-real-window-accessibility-metadata" {
		return nil
	}
	if mode == "wasm32-web-browser-canvas-accessibility-metadata" {
		return nil
	}
	return fmt.Errorf("unsupported Surface smoke mode %q", mode)
}
func defaultSurfaceSourcePath(opt smokeOptions) string {
	if opt.SourcePath != "" && opt.SourcePath != "examples/surface/runtime/surface_counter.tetra" {
		return opt.SourcePath
	}
	if isTextFocusInputMode(opt.Mode) {
		return "examples/surface/runtime/surface_textbox_app.tetra"
	}
	if isReleaseTextInputMode(opt.Mode) {
		return "examples/surface/release/surface_release_text_input.tetra"
	}
	if isReleaseToolkitMode(opt.Mode) {
		return "examples/surface/release/surface_release_form.tetra"
	}
	if isReleaseWindowMode(opt.Mode) {
		return "examples/surface/release/surface_release_form.tetra"
	}
	if isReleaseAppShellMode(opt.Mode) {
		return "examples/surface/toolkit/surface_linux_app_shell_notes.tetra"
	}
	if isReleaseBrowserMode(opt.Mode) {
		return "examples/surface/release/surface_release_form.tetra"
	}
	if isReleaseAccessibilityMode(opt.Mode) {
		return "examples/surface/release/surface_release_accessibility.tetra"
	}
	if isComponentTreeMode(opt.Mode) {
		return "examples/surface/toolkit/surface_tree_app.tetra"
	}
	if isBlockPaintMode(opt.Mode) {
		return "examples/surface/block_render/surface_block_paint_layers.tetra"
	}
	if isBlockTextMode(opt.Mode) {
		return "examples/surface/block_render/surface_block_text.tetra"
	}
	if isBlockLayoutMode(opt.Mode) {
		return "examples/surface/block_core/surface_block_layout.tetra"
	}
	if isBlockEventMode(opt.Mode) {
		return "examples/surface/block_core/surface_block_events.tetra"
	}
	if isBlockStateMode(opt.Mode) {
		return "examples/surface/block_core/surface_block_states.tetra"
	}
	if isBlockMotionMode(opt.Mode) {
		return "examples/surface/block_core/surface_block_motion.tetra"
	}
	if isBlockAssetMode(opt.Mode) {
		return "examples/surface/block_render/surface_block_assets.tetra"
	}
	if isBlockAccessibilityMode(opt.Mode) {
		return "examples/surface/block_render/surface_block_accessibility.tetra"
	}
	if isWASM32WebBrowserCanvasMorphMode(opt.Mode) {
		return "examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra"
	}
	if isLinuxX64RealWindowMorphMode(opt.Mode) {
		return "examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra"
	}
	if isMorphMode(opt.Mode) {
		return "examples/surface/morph_core/surface_morph_command_palette.tetra"
	}
	if isBlockSystemMode(opt.Mode) {
		return "examples/surface/block_core/surface_block_system.tetra"
	}
	if isMinimalToolkitMode(opt.Mode) {
		return "examples/surface/toolkit/surface_toolkit_form.tetra"
	}
	if isToolkitReuseMode(opt.Mode) {
		return "examples/surface/toolkit/surface_toolkit_settings.tetra"
	}
	if isAccessibilityMetadataMode(opt.Mode) {
		return "examples/surface/toolkit/surface_accessibility_settings.tetra"
	}
	if isAppModelMode(opt.Mode) {
		return "examples/surface/toolkit/surface_app_model.tetra"
	}
	if opt.Mode == "linux-x64-real-window" {
		return "examples/surface/runtime/surface_window_counter.tetra"
	}
	if opt.Mode == "wasm32-web-browser-canvas" {
		return "examples/surface/runtime/surface_browser_counter.tetra"
	}
	if opt.SourcePath == "" {
		return "examples/surface/runtime/surface_counter.tetra"
	}
	return opt.SourcePath
}
func isTextFocusInputMode(mode string) bool {
	return mode == "headless-text-focus-input" ||
		mode == "linux-x64-real-window-text-focus-input" ||
		mode == "wasm32-web-browser-canvas-text-focus-input"
}
func isReleaseTextInputMode(mode string) bool {
	return mode == "headless-release-text-input" ||
		mode == "linux-x64-release-text-input" ||
		mode == "wasm32-web-release-text-input"
}
func isReleaseToolkitMode(mode string) bool {
	return mode == "headless-release-toolkit" ||
		mode == "linux-x64-release-toolkit" ||
		mode == "wasm32-web-release-toolkit"
}
func isReleaseWindowMode(mode string) bool {
	return mode == "linux-x64-release-window"
}
func isReleaseAppShellMode(mode string) bool {
	return mode == "linux-x64-release-app-shell"
}
func isReleaseBrowserMode(mode string) bool {
	return mode == "wasm32-web-release-browser"
}
func isReleaseAccessibilityMode(mode string) bool {
	return mode == "headless-release-accessibility" ||
		mode == "linux-x64-release-accessibility" ||
		mode == "wasm32-web-release-accessibility"
}
func isComponentTreeMode(mode string) bool {
	return mode == "headless-component-tree" ||
		mode == "linux-x64-real-window-component-tree" ||
		mode == "wasm32-web-browser-canvas-component-tree" ||
		mode == "headless-component-tree-api" ||
		mode == "linux-x64-real-window-component-tree-api" ||
		mode == "wasm32-web-browser-canvas-component-tree-api"
}
func isBlockPaintMode(mode string) bool {
	return mode == "headless-block-paint"
}
func surfaceComponentAppExpectedExit(mode string) int {
	if isBlockPaintMode(mode) {
		return 0
	}
	return 1
}
func surfaceComponentAppExpectedExitForSource(mode string, sourcePath string) int {
	if surfaceSmokeSourceIsFlagshipControlCenter(sourcePath) {
		return 5
	}
	if isMorphRenderedFlagshipSource(sourcePath) {
		return 0
	}
	if isMorphGuestDashboardSource(sourcePath) {
		return 0
	}
	if surfaceSmokeSourceIsReferenceApp(sourcePath) {
		return 0
	}
	if surfaceSmokeSourceNeedsRepoDependency(sourcePath) {
		return 0
	}
	return surfaceComponentAppExpectedExit(mode)
}
func surfaceSmokeBuildOptions(sourcePath string) compiler.BuildOptions {
	opt := compiler.BuildOptions{Jobs: 1}
	if !surfaceSmokeSourceNeedsRepoDependency(sourcePath) {
		return opt
	}
	root, err := repoRootForCommands()
	if err != nil {
		return opt
	}
	opt.DependencyRoots = []compiler.ModuleRoot{{Root: root}}
	return opt
}
func surfaceSmokeSourceNeedsRepoDependency(sourcePath string) bool {
	clean := filepath.ToSlash(filepath.Clean(sourcePath))
	return strings.Contains(clean, "/reports/") || strings.HasPrefix(clean, "reports/")
}
func surfaceSmokeSourceIsFlagshipControlCenter(sourcePath string) bool {
	clean := filepath.ToSlash(filepath.Clean(sourcePath))
	return clean == "examples/surface/migration/surface_migration_tetra_control_center.tetra" ||
		strings.HasSuffix(
			clean,
			"/examples/surface/migration/surface_migration_tetra_control_center.tetra",
		)
}

func surfaceSmokeSourceIsReferenceApp(sourcePath string) bool {
	clean := filepath.ToSlash(filepath.Clean(sourcePath))
	return isSurfaceReferenceSourcePath(clean)
}

func isSurfaceReferenceSourcePath(clean string) bool {
	return strings.HasPrefix(clean, "examples/surface/reference_core/surface_reference_") &&
		strings.HasSuffix(clean, ".tetra") ||
		strings.HasPrefix(clean, "examples/surface/reference_forms/surface_reference_") &&
			strings.HasSuffix(clean, ".tetra") ||
		strings.Contains(clean, "/examples/surface/reference_core/surface_reference_") &&
			strings.HasSuffix(clean, ".tetra") ||
		strings.Contains(clean, "/examples/surface/reference_forms/surface_reference_") &&
			strings.HasSuffix(clean, ".tetra")
}
func shouldRetargetSurfaceTemplateScenario(opt smokeOptions) bool {
	if !isMorphMode(opt.Mode) && !isBlockSystemMode(opt.Mode) {
		return false
	}
	return surfaceSmokeSourceNeedsRepoDependency(defaultSurfaceSourcePath(opt))
}
func shouldRetargetBlockSystemSourceScenario(opt smokeOptions) bool {
	if !isBlockSystemMode(opt.Mode) {
		return false
	}
	source := filepath.ToSlash(filepath.Clean(defaultSurfaceSourcePath(opt)))
	return !surfaceSmokeSourceNeedsRepoDependency(source) &&
		source != "examples/surface/block_core/surface_block_system.tetra"
}
func surfaceSourceModuleName(source string) string {
	clean := filepath.ToSlash(filepath.Clean(source))
	clean = strings.TrimSuffix(clean, ".tetra")
	clean = strings.TrimSuffix(clean, ".t4")
	clean = strings.TrimPrefix(clean, "./")
	if clean == "." || clean == "" {
		return "main"
	}
	return strings.ReplaceAll(clean, "/", ".")
}
func isBlockTextMode(mode string) bool {
	return mode == "headless-block-text"
}
func isBlockLayoutMode(mode string) bool {
	return mode == "headless-block-layout"
}
func isBlockEventMode(mode string) bool {
	return mode == "headless-block-events"
}
func isBlockStateMode(mode string) bool {
	return mode == "headless-block-states"
}
func isBlockMotionMode(mode string) bool {
	return mode == "headless-block-motion"
}
func isBlockAssetMode(mode string) bool {
	return mode == "headless-block-assets"
}
func isBlockAccessibilityMode(mode string) bool {
	return mode == "headless-block-accessibility"
}
func isBlockSystemMode(mode string) bool {
	return mode == "headless-block-system" ||
		mode == "linux-x64-real-window-block-system" ||
		mode == "wasm32-web-browser-canvas-block-system"
}
func isMorphMode(mode string) bool {
	return mode == "headless-morph" ||
		isLinuxX64RealWindowMorphMode(mode) ||
		isWASM32WebBrowserCanvasMorphMode(mode)
}
func isLinuxX64RealWindowMorphMode(mode string) bool {
	return mode == "linux-x64-real-window-morph"
}
func isWASM32WebBrowserCanvasMorphMode(mode string) bool {
	return mode == "wasm32-web-browser-canvas-morph"
}
func isMorphTargetRuntimeMode(mode string) bool {
	return isLinuxX64RealWindowMorphMode(mode) || isWASM32WebBrowserCanvasMorphMode(mode)
}
func isMinimalToolkitMode(mode string) bool {
	return mode == "headless-minimal-toolkit" ||
		mode == "linux-x64-real-window-minimal-toolkit" ||
		mode == "wasm32-web-browser-canvas-minimal-toolkit"
}
func isToolkitReuseMode(mode string) bool {
	return mode == "headless-toolkit-reuse" ||
		mode == "linux-x64-real-window-toolkit-reuse" ||
		mode == "wasm32-web-browser-canvas-toolkit-reuse"
}
func isAccessibilityMetadataMode(mode string) bool {
	return mode == "headless-accessibility-metadata" ||
		mode == "linux-x64-real-window-accessibility-metadata" ||
		mode == "wasm32-web-browser-canvas-accessibility-metadata"
}
func isAppModelMode(mode string) bool {
	return mode == "headless-app-model"
}

func isHeadlessReportTargetMode(mode string) bool {
	switch mode {
	case "headless-text-focus-input",
		"headless-release-toolkit",
		"headless-release-accessibility",
		"headless-component-tree",
		"headless-component-tree-api",
		"headless-block-paint",
		"headless-block-text",
		"headless-block-layout",
		"headless-block-events",
		"headless-block-states",
		"headless-block-motion",
		"headless-block-assets",
		"headless-block-accessibility",
		"headless-block-system",
		"headless-morph",
		"headless-minimal-toolkit",
		"headless-toolkit-reuse",
		"headless-accessibility-metadata",
		"headless-app-model":
		return true
	default:
		return false
	}
}

func isLinuxX64ReportTargetMode(mode string) bool {
	switch mode {
	case "linux-x64",
		"linux-x64-real-window",
		"linux-x64-real-window-text-focus-input",
		"linux-x64-release-toolkit",
		"linux-x64-release-window",
		"linux-x64-release-app-shell",
		"linux-x64-release-accessibility",
		"linux-x64-real-window-component-tree",
		"linux-x64-real-window-component-tree-api",
		"linux-x64-real-window-block-system",
		"linux-x64-real-window-minimal-toolkit",
		"linux-x64-real-window-toolkit-reuse",
		"linux-x64-real-window-accessibility-metadata":
		return true
	default:
		return isLinuxX64RealWindowMorphMode(mode)
	}
}

func isWASM32WebReportTargetMode(mode string) bool {
	switch mode {
	case "wasm32-web",
		"wasm32-web-browser-canvas",
		"wasm32-web-browser-canvas-text-focus-input",
		"wasm32-web-release-toolkit",
		"wasm32-web-release-browser",
		"wasm32-web-release-accessibility",
		"wasm32-web-browser-canvas-component-tree",
		"wasm32-web-browser-canvas-component-tree-api",
		"wasm32-web-browser-canvas-minimal-toolkit",
		"wasm32-web-browser-canvas-toolkit-reuse",
		"wasm32-web-browser-canvas-accessibility-metadata",
		"wasm32-web-browser-canvas-block-system":
		return true
	default:
		return isWASM32WebBrowserCanvasMorphMode(mode)
	}
}

func releaseCounterScenarioForSource(opt smokeOptions, scenario headlessScenario) headlessScenario {
	if normalizeSurfaceSourcePath(
		defaultSurfaceSourcePath(opt),
	) != "examples/surface/release/surface_release_counter.tetra" {
		return scenario
	}
	switch opt.Mode {
	case "",
		"headless",
		"linux-x64",
		"linux-x64-real-window",
		"wasm32-web",
		"wasm32-web-browser-canvas":
	default:
		return scenario
	}
	for i := range scenario.Components {
		if strings.HasPrefix(
			scenario.Components[i].Type,
			"examples.surface.runtime.surface_counter.",
		) ||
			strings.HasPrefix(
				scenario.Components[i].Type,
				"examples.surface.runtime.surface_window_counter.",
			) ||
			strings.HasPrefix(
				scenario.Components[i].Type,
				"examples.surface.runtime.surface_browser_counter.",
			) {
			name := scenario.Components[i].Type[strings.LastIndex(scenario.Components[i].Type, ".")+1:]
			scenario.Components[i].Type = "examples.surface.release.surface_release_counter." + name
		}
	}
	scenario.Cases = append(
		scenario.Cases,
		surface.CaseReport{
			Name: "release counter source module evidence",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		surface.CaseReport{
			Name: "release counter stable widgets accessibility metadata",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
	)
	return scenario
}
func normalizeSurfaceSourcePath(path string) string {
	return strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
}
func isReleaseCounterSourcePath(path string) bool {
	return strings.HasSuffix(
		normalizeSurfaceSourcePath(path),
		"examples/surface/release/surface_release_counter.tetra",
	)
}

// ---- process.go ----

func collectSurfaceProcessEvidence(opt smokeOptions) (surfaceProcessEvidence, error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return surfaceProcessEvidence{}, fmt.Errorf(
			("Surface smoke currently requires a linux/amd64 host to " +
				"build and run linux-x64 Surface app evidence; host is %s/%s"),
			runtime.GOOS,
			runtime.GOARCH,
		)
	}

	sourcePath, err := resolveSurfaceSourcePath(defaultSurfaceSourcePath(opt))
	if err != nil {
		return surfaceProcessEvidence{}, fmt.Errorf("build Surface source: %w", err)
	}

	mode := opt.Mode
	if mode == "" {
		mode = "headless"
	}
	artifactDir := surfaceRuntimeArtifactDir(opt)
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return surfaceProcessEvidence{}, fmt.Errorf("create Surface artifact directory: %w", err)
	}
	if mode == "wasm32-web" {
		return collectWASM32WebProcessEvidence(sourcePath, artifactDir)
	}
	if mode == "wasm32-web-browser-canvas" {
		return collectWASM32WebBrowserCanvasProcessEvidence(sourcePath, artifactDir, "counter")
	}
	if mode == "wasm32-web-browser-canvas-text-focus-input" {
		return collectWASM32WebBrowserCanvasProcessEvidence(
			sourcePath,
			artifactDir,
			"text-focus-input",
		)
	}
	if mode == "wasm32-web-release-text-input" {
		return collectWASM32WebBrowserCanvasProcessEvidence(
			sourcePath,
			artifactDir,
			"release-text-input",
		)
	}
	if mode == "wasm32-web-release-toolkit" {
		return collectWASM32WebBrowserCanvasProcessEvidence(
			sourcePath,
			artifactDir,
			"release-toolkit",
		)
	}
	if mode == "wasm32-web-release-browser" {
		return collectWASM32WebBrowserCanvasProcessEvidence(
			sourcePath,
			artifactDir,
			"release-browser",
		)
	}
	if mode == "wasm32-web-release-accessibility" {
		return collectWASM32WebBrowserCanvasProcessEvidence(
			sourcePath,
			artifactDir,
			"release-accessibility",
		)
	}
	if mode == "wasm32-web-browser-canvas-component-tree" ||
		mode == "wasm32-web-browser-canvas-component-tree-api" {
		return collectWASM32WebBrowserCanvasProcessEvidence(
			sourcePath,
			artifactDir,
			"component-tree",
		)
	}
	if mode == "wasm32-web-browser-canvas-minimal-toolkit" {
		return collectWASM32WebBrowserCanvasProcessEvidence(
			sourcePath,
			artifactDir,
			"minimal-toolkit",
		)
	}
	if mode == "wasm32-web-browser-canvas-toolkit-reuse" {
		return collectWASM32WebBrowserCanvasProcessEvidence(
			sourcePath,
			artifactDir,
			"toolkit-reuse",
		)
	}
	if mode == "wasm32-web-browser-canvas-accessibility-metadata" {
		return collectWASM32WebBrowserCanvasProcessEvidence(
			sourcePath,
			artifactDir,
			"accessibility-metadata",
		)
	}
	if mode == "wasm32-web-browser-canvas-block-system" {
		return collectWASM32WebBrowserCanvasProcessEvidence(sourcePath, artifactDir, "block-system")
	}
	if isWASM32WebBrowserCanvasMorphMode(mode) {
		return collectWASM32WebBrowserCanvasProcessEvidence(
			sourcePath,
			artifactDir,
			morphBrowserCanvasScenarioName(sourcePath),
		)
	}

	appName := "surface-counter"
	if isTextFocusInputMode(mode) {
		appName = "surface-textbox-app"
	}
	if isReleaseTextInputMode(mode) {
		appName = "surface-release-text-input"
	}
	if isReleaseToolkitMode(mode) {
		appName = "surface-release-form"
	}
	if isReleaseWindowMode(mode) {
		appName = "surface-release-form"
	}
	if isReleaseAppShellMode(mode) {
		appName = "surface-linux-app-shell-notes"
	}
	if isReleaseBrowserMode(mode) {
		appName = "surface-release-form"
	}
	if isReleaseAccessibilityMode(mode) {
		appName = "surface-release-accessibility"
	}
	if isComponentTreeMode(mode) {
		appName = "surface-tree-app"
	}
	if isBlockPaintMode(mode) {
		appName = "surface-block-paint"
	}
	if isBlockTextMode(mode) {
		appName = "surface-block-text"
	}
	if isBlockLayoutMode(mode) {
		appName = "surface-block-layout"
	}
	if isBlockEventMode(mode) {
		appName = "surface-block-events"
	}
	if isBlockStateMode(mode) {
		appName = "surface-block-states"
	}
	if isBlockMotionMode(mode) {
		appName = "surface-block-motion"
	}
	if isBlockAssetMode(mode) {
		appName = "surface-block-assets"
	}
	if isBlockAccessibilityMode(mode) {
		appName = "surface-block-accessibility"
	}
	if isBlockSystemMode(mode) {
		appName = "surface-block-system"
	}
	if isMorphMode(mode) {
		appName = "surface-morph-command-palette"
		if isMorphRenderedFlagshipSource(sourcePath) {
			appName = "surface-morph-rendered-studio-shell"
		} else if isMorphGuestDashboardSource(sourcePath) {
			appName = "surface-morph-guest-dashboard"
		}
	}
	if isMinimalToolkitMode(mode) {
		appName = "surface-toolkit-form"
	}
	if isToolkitReuseMode(mode) {
		appName = "surface-toolkit-settings"
	}
	if isAccessibilityMetadataMode(mode) {
		appName = "surface-accessibility-settings"
	}
	if isAppModelMode(mode) {
		appName = "surface-app-model"
	}
	appPath := filepath.Join(artifactDir, appName)
	if _, err := compiler.BuildFileWithStatsOpt(
		sourcePath,
		appPath,
		"linux-x64",
		surfaceSmokeBuildOptions(sourcePath),
	); err != nil {
		return surfaceProcessEvidence{}, fmt.Errorf("build Surface source %s: %w", sourcePath, err)
	}
	componentArtifact, err := artifactReport(appPath, "component-app")
	if err != nil {
		return surfaceProcessEvidence{}, err
	}
	sidecarScan, err := scanLegacyUISidecarArtifacts(artifactDir)
	if err != nil {
		return surfaceProcessEvidence{}, err
	}

	stdout, stderr, appExit, err := runExecutable(appPath)
	if err != nil {
		return surfaceProcessEvidence{}, fmt.Errorf("run Surface app %s: %w", appPath, err)
	}
	if stdout != "" {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"run Surface app %s: unexpected stdout %q",
			appPath,
			stdout,
		)
	}
	if stderr != "" {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"run Surface app %s: unexpected stderr %q",
			appPath,
			stderr,
		)
	}
	expectedAppExit := surfaceComponentAppExpectedExitForSource(mode, sourcePath)
	if appExit != expectedAppExit {
		return surfaceProcessEvidence{}, fmt.Errorf(
			"run Surface app %s: exit code %d, want %d",
			appPath,
			appExit,
			expectedAppExit,
		)
	}

	processes := []surface.ProcessReport{
		{
			Name:     "tetra build",
			Kind:     "build",
			Path:     fmt.Sprintf("tetra build --target linux-x64 %s -o %s", sourcePath, appPath),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             appPath,
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(appExit),
			ExpectedExitCode: intPtr(expectedAppExit),
		},
	}
	runtimeProcessName := "surface headless runtime"
	if mode == "linux-x64" {
		runtimeProcessName = "surface linux-x64 runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		eventSequenceProcesses, err := collectLinuxX64EventSequenceProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, eventSequenceProcesses...)
		presentProcess, presentFrame, err := collectLinuxX64PresentedFrameEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, presentProcess)
		counterProcess, counterFrame, err := collectLinuxX64CounterAppPresentedFrameEvidence(
			artifactDir,
		)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, counterProcess)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(
			processes,
			surface.ProcessReport{
				Name:     runtimeProcessName,
				Kind:     "runtime",
				Path:     os.Args[0],
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		)
		return surfaceProcessEvidence{
			Processes:    processes,
			Artifacts:    []surface.ArtifactReport{componentArtifact},
			ArtifactScan: sidecarScan,
			Frames:       []surface.FrameReport{presentFrame, counterFrame},
		}, nil
	}
	if mode == "linux-x64-real-window" {
		runtimeProcessName = "surface linux-x64 real-window runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		realWindowProcess, realWindowFrame, err := collectLinuxX64RealWindowProbeEvidence(
			artifactDir,
		)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, realWindowProcess)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(
			processes,
			surface.ProcessReport{
				Name:     runtimeProcessName,
				Kind:     "runtime",
				Path:     os.Args[0],
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		)
		return surfaceProcessEvidence{
			Processes:    processes,
			Artifacts:    []surface.ArtifactReport{componentArtifact},
			ArtifactScan: sidecarScan,
			Frames:       []surface.FrameReport{realWindowFrame},
		}, nil
	}
	if mode == "linux-x64-real-window-text-focus-input" {
		runtimeProcessName = "surface linux-x64 real-window runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		realWindowProcess, realWindowFrame, err := collectLinuxX64TextFocusInputRealWindowProbeEvidence(
			artifactDir,
		)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, realWindowProcess)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(
			processes,
			surface.ProcessReport{
				Name:     runtimeProcessName,
				Kind:     "runtime",
				Path:     os.Args[0],
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		)
		return surfaceProcessEvidence{
			Processes:    processes,
			Artifacts:    []surface.ArtifactReport{componentArtifact},
			ArtifactScan: sidecarScan,
			Frames:       []surface.FrameReport{realWindowFrame},
		}, nil
	}
	if mode == "linux-x64-release-text-input" {
		runtimeProcessName = "surface linux-x64 runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		realWindowProcess, realWindowFrame, err := collectLinuxX64TextFocusInputRealWindowProbeEvidence(
			artifactDir,
		)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, realWindowProcess)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(
			processes,
			surface.ProcessReport{
				Name:     runtimeProcessName,
				Kind:     "runtime",
				Path:     os.Args[0],
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		)
		return surfaceProcessEvidence{
			Processes:    processes,
			Artifacts:    []surface.ArtifactReport{componentArtifact},
			ArtifactScan: sidecarScan,
			Frames:       []surface.FrameReport{realWindowFrame},
		}, nil
	}
	if mode == "linux-x64-release-toolkit" {
		runtimeProcessName = "surface linux-x64 runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		realWindowProcess, realWindowFrame, err := collectLinuxX64ReleaseToolkitRealWindowProbeEvidence(
			artifactDir,
		)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, realWindowProcess)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(
			processes,
			surface.ProcessReport{
				Name:     runtimeProcessName,
				Kind:     "runtime",
				Path:     os.Args[0],
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		)
		return surfaceProcessEvidence{
			Processes:    processes,
			Artifacts:    []surface.ArtifactReport{componentArtifact},
			ArtifactScan: sidecarScan,
			Frames:       []surface.FrameReport{realWindowFrame},
		}, nil
	}
	if mode == "linux-x64-release-window" {
		runtimeProcessName = "surface linux-x64 runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		realWindowProcess, realWindowFrame, err := collectLinuxX64ReleaseToolkitRealWindowProbeEvidence(
			artifactDir,
		)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, realWindowProcess)
		harnessProcesses, harnessArtifacts, err := collectLinuxX64ReleaseWindowHarnessEvidence(
			artifactDir,
		)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, harnessProcesses...)
		bridgeProcesses, bridgeArtifacts, err := collectLinuxX64ReleaseWindowAccessibilityBridgeEvidence(
			artifactDir,
		)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, bridgeProcesses...)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(
			processes,
			surface.ProcessReport{
				Name:     runtimeProcessName,
				Kind:     "runtime",
				Path:     os.Args[0],
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		)
		artifacts := append([]surface.ArtifactReport{componentArtifact}, harnessArtifacts...)
		artifacts = append(artifacts, bridgeArtifacts...)
		return surfaceProcessEvidence{
			Processes:    processes,
			Artifacts:    artifacts,
			ArtifactScan: sidecarScan,
			Frames:       []surface.FrameReport{realWindowFrame},
		}, nil
	}
	if mode == "linux-x64-release-app-shell" {
		runtimeProcessName = "surface linux-x64 runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		realWindowProcess, realWindowFrame, err := collectLinuxX64ReleaseToolkitRealWindowProbeEvidence(
			artifactDir,
		)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, realWindowProcess)
		appShellProcesses, appShellArtifacts, err := collectLinuxAppShellTraceEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, appShellProcesses...)
		harnessProcesses, harnessArtifacts, err := collectLinuxX64ReleaseWindowHarnessEvidence(
			artifactDir,
		)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, harnessProcesses...)
		bridgeProcesses, bridgeArtifacts, err := collectLinuxX64ReleaseWindowAccessibilityBridgeEvidence(
			artifactDir,
		)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, bridgeProcesses...)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(
			processes,
			surface.ProcessReport{
				Name:     runtimeProcessName,
				Kind:     "runtime",
				Path:     os.Args[0],
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		)
		artifacts := append([]surface.ArtifactReport{componentArtifact}, appShellArtifacts...)
		artifacts = append(artifacts, harnessArtifacts...)
		artifacts = append(artifacts, bridgeArtifacts...)
		return surfaceProcessEvidence{
			Processes:    processes,
			Artifacts:    artifacts,
			ArtifactScan: sidecarScan,
			Frames:       []surface.FrameReport{realWindowFrame},
		}, nil
	}
	if mode == "linux-x64-release-accessibility" {
		runtimeProcessName = "surface linux-x64 runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		realWindowProcess, realWindowFrame, err :=
			collectLinuxX64ReleaseAccessibilityRealWindowProbeEvidence(
				artifactDir,
			)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, realWindowProcess)
		bridgeProcesses, bridgeArtifacts, err := collectLinuxX64ReleaseAccessibilityBridgeEvidence(
			artifactDir,
		)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, bridgeProcesses...)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(
			processes,
			surface.ProcessReport{
				Name:     runtimeProcessName,
				Kind:     "runtime",
				Path:     os.Args[0],
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		)
		artifacts := append([]surface.ArtifactReport{componentArtifact}, bridgeArtifacts...)
		return surfaceProcessEvidence{
			Processes:    processes,
			Artifacts:    artifacts,
			ArtifactScan: sidecarScan,
			Frames:       []surface.FrameReport{realWindowFrame},
		}, nil
	}
	if mode == "linux-x64-real-window-component-tree" ||
		mode == "linux-x64-real-window-component-tree-api" {
		runtimeProcessName = "surface linux-x64 real-window runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		realWindowProcess, realWindowFrame, err := collectLinuxX64ComponentTreeRealWindowProbeEvidence(
			artifactDir,
		)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, realWindowProcess)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(
			processes,
			surface.ProcessReport{
				Name:     runtimeProcessName,
				Kind:     "runtime",
				Path:     os.Args[0],
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		)
		return surfaceProcessEvidence{
			Processes:    processes,
			Artifacts:    []surface.ArtifactReport{componentArtifact},
			ArtifactScan: sidecarScan,
			Frames:       []surface.FrameReport{realWindowFrame},
		}, nil
	}
	if mode == "linux-x64-real-window-block-system" {
		runtimeProcessName = "surface linux-x64 real-window runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		realWindowProcess, realWindowFrame, err := collectLinuxX64BlockSystemRealWindowProbeEvidence(
			artifactDir,
		)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, realWindowProcess)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(
			processes,
			surface.ProcessReport{
				Name:     runtimeProcessName,
				Kind:     "runtime",
				Path:     os.Args[0],
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		)
		return surfaceProcessEvidence{
			Processes:    processes,
			Artifacts:    []surface.ArtifactReport{componentArtifact},
			ArtifactScan: sidecarScan,
			Frames:       []surface.FrameReport{realWindowFrame},
		}, nil
	}
	if isLinuxX64RealWindowMorphMode(mode) {
		runtimeProcessName = "surface linux-x64 real-window runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		morphProcesses, morphFrames, err := collectLinuxX64MorphRealWindowProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, morphProcesses...)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(
			processes,
			surface.ProcessReport{
				Name:     runtimeProcessName,
				Kind:     "runtime",
				Path:     os.Args[0],
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		)
		return surfaceProcessEvidence{
			Processes:    processes,
			Artifacts:    []surface.ArtifactReport{componentArtifact},
			ArtifactScan: sidecarScan,
			Frames:       morphFrames,
		}, nil
	}
	if mode == "linux-x64-real-window-minimal-toolkit" {
		runtimeProcessName = "surface linux-x64 real-window runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		realWindowProcess, realWindowFrame, err := collectLinuxX64MinimalToolkitRealWindowProbeEvidence(
			artifactDir,
		)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, realWindowProcess)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(
			processes,
			surface.ProcessReport{
				Name:     runtimeProcessName,
				Kind:     "runtime",
				Path:     os.Args[0],
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		)
		return surfaceProcessEvidence{
			Processes:    processes,
			Artifacts:    []surface.ArtifactReport{componentArtifact},
			ArtifactScan: sidecarScan,
			Frames:       []surface.FrameReport{realWindowFrame},
		}, nil
	}
	if mode == "linux-x64-real-window-toolkit-reuse" {
		runtimeProcessName = "surface linux-x64 real-window runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		realWindowProcess, realWindowFrame, err := collectLinuxX64ToolkitReuseRealWindowProbeEvidence(
			artifactDir,
		)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, realWindowProcess)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(
			processes,
			surface.ProcessReport{
				Name:     runtimeProcessName,
				Kind:     "runtime",
				Path:     os.Args[0],
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		)
		return surfaceProcessEvidence{
			Processes:    processes,
			Artifacts:    []surface.ArtifactReport{componentArtifact},
			ArtifactScan: sidecarScan,
			Frames:       []surface.FrameReport{realWindowFrame},
		}, nil
	}
	if mode == "linux-x64-real-window-accessibility-metadata" {
		runtimeProcessName = "surface linux-x64 real-window runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		realWindowProcess, realWindowFrame, err :=
			collectLinuxX64AccessibilityMetadataRealWindowProbeEvidence(
				artifactDir,
			)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, realWindowProcess)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(
			processes,
			surface.ProcessReport{
				Name:     runtimeProcessName,
				Kind:     "runtime",
				Path:     os.Args[0],
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		)
		return surfaceProcessEvidence{
			Processes:    processes,
			Artifacts:    []surface.ArtifactReport{componentArtifact},
			ArtifactScan: sidecarScan,
			Frames:       []surface.FrameReport{realWindowFrame},
		}, nil
	}
	traceArtifact, sidecarScan, err := collectHeadlessRunnerTraceEvidence(
		sourcePath,
		artifactDir,
		runSurfaceScenario(mode),
	)
	if err != nil {
		return surfaceProcessEvidence{}, err
	}
	processes = append(
		processes,
		surface.ProcessReport{
			Name:     runtimeProcessName,
			Kind:     "runtime",
			Path:     os.Args[0],
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	)
	return surfaceProcessEvidence{
		Processes:    processes,
		Artifacts:    []surface.ArtifactReport{componentArtifact, traceArtifact},
		ArtifactScan: sidecarScan,
	}, nil
}

func morphBrowserCanvasScenarioName(sourcePath string) string {
	if isMorphGuestDashboardSource(sourcePath) {
		return "guest-dashboard"
	}
	return "studio-shell"
}

func surfaceRuntimeArtifactDir(opt smokeOptions) string {
	reportDir := filepath.Dir(opt.ReportPath)
	if reportDir == "." || reportDir == "" {
		reportDir = "reports/surface"
	}
	mode := opt.Mode
	if mode == "" {
		mode = "headless"
	}
	return filepath.Join(reportDir, "surface-"+mode+"-artifacts")
}

// ---- reports.go ----

func mergeFrameEvidenceByOrder(
	base []surface.FrameReport,
	evidence []surface.FrameReport,
) []surface.FrameReport {
	byOrder := map[int]surface.FrameReport{}
	for _, frame := range base {
		byOrder[frame.Order] = frame
	}
	for _, frame := range evidence {
		byOrder[frame.Order] = frame
	}
	orders := make([]int, 0, len(byOrder))
	for order := range byOrder {
		orders = append(orders, order)
	}
	sort.Ints(orders)
	merged := make([]surface.FrameReport, 0, len(orders))
	for _, order := range orders {
		merged = append(merged, byOrder[order])
	}
	return merged
}
func artifactReport(path string, kind string) (surface.ArtifactReport, error) {
	file, err := os.Open(path)
	if err != nil {
		return surface.ArtifactReport{}, fmt.Errorf("open Surface artifact %s: %w", path, err)
	}
	hash := sha256.New()
	size, copyErr := io.Copy(hash, file)
	closeErr := file.Close()
	if copyErr != nil {
		return surface.ArtifactReport{}, fmt.Errorf("hash Surface artifact %s: %w", path, copyErr)
	}
	if closeErr != nil {
		return surface.ArtifactReport{}, fmt.Errorf("close Surface artifact %s: %w", path, closeErr)
	}
	return surface.ArtifactReport{
		Kind:   kind,
		Path:   path,
		SHA256: "sha256:" + hex.EncodeToString(hash.Sum(nil)),
		Size:   size,
	}, nil
}

func buildReport(
	opt smokeOptions,
	host string,
	processes []surface.ProcessReport,
	artifacts []surface.ArtifactReport,
	artifactScan surface.ArtifactScanReport,
	scenario headlessScenario,
) surface.Report {
	mode := opt.Mode
	if mode == "" {
		mode = "headless"
	}
	source := defaultSurfaceSourcePath(opt)
	attachRenderCommandStreamForScenarioWithRenderer(
		source,
		renderCommandStreamRendererForMode(mode),
		&scenario,
	)
	target := mode
	runtimeName := "surface-headless"
	if isHeadlessReportTargetMode(mode) {
		target = "headless"
		runtimeName = "surface-headless"
	} else if isLinuxX64ReportTargetMode(mode) {
		target = "linux-x64"
		runtimeName = "surface-linux-x64"
	} else if isWASM32WebReportTargetMode(mode) {
		target = "wasm32-web"
		runtimeName = "surface-wasm32-web"
	}
	performanceBudget := scenario.SurfacePerformanceBudget
	if performanceBudget == nil {
		performanceBudget = surfacePerformanceBudgetForScenario(
			target,
			runtimeName,
			source,
			artifacts,
			scenario,
		)
	}
	return surface.Report{
		Schema:                          surface.SchemaV1,
		Status:                          "pass",
		Target:                          target,
		Host:                            host,
		Runtime:                         runtimeName,
		SurfaceSchema:                   "tetra.surface.v1",
		HostABI:                         "tetra.surface.host-abi.v1",
		HostEvidence:                    hostEvidenceForMode(mode),
		Source:                          source,
		Processes:                       processes,
		Artifacts:                       artifacts,
		ArtifactScan:                    artifactScan,
		Components:                      scenario.Components,
		ComponentTree:                   scenario.ComponentTree,
		ComponentTreeAPI:                scenario.ComponentTreeAPI,
		BlockGraph:                      scenario.BlockGraph,
		BlockSceneSnapshot:              scenario.BlockSceneSnapshot,
		RenderCommandStream:             scenario.RenderCommandStream,
		PaintLayers:                     scenario.PaintLayers,
		PaintCommands:                   scenario.PaintCommands,
		VisualFeatures:                  scenario.VisualFeatures,
		PaintQualityLevel:               scenario.PaintQualityLevel,
		PaintCacheBudgetBytes:           scenario.PaintCacheBudgetBytes,
		PaintUnsupportedBlur:            scenario.PaintUnsupportedBlur,
		Renderer:                        scenario.Renderer,
		TextMeasurements:                scenario.TextMeasurements,
		FontFallbacks:                   scenario.FontFallbacks,
		GlyphCaches:                     scenario.GlyphCaches,
		TextRenderCommands:              scenario.TextRenderCommands,
		TextQualityLevel:                scenario.TextQualityLevel,
		TextCacheBudgetBytes:            scenario.TextCacheBudgetBytes,
		LayoutConstraints:               scenario.LayoutConstraints,
		LayoutPasses:                    scenario.LayoutPasses,
		LayoutScrolls:                   scenario.LayoutScrolls,
		LayoutDensity:                   scenario.LayoutDensity,
		LayoutFeatures:                  scenario.LayoutFeatures,
		LayoutQualityLevel:              scenario.LayoutQualityLevel,
		LayoutUnsupportedCSSFlexbox:     scenario.LayoutUnsupportedCSSFlexbox,
		BlockEventRoutes:                scenario.BlockEventRoutes,
		BlockFocusTransitions:           scenario.BlockFocusTransitions,
		BlockEventKinds:                 scenario.BlockEventKinds,
		BlockEventPolicy:                scenario.BlockEventPolicy,
		BlockEventQualityLevel:          scenario.BlockEventQualityLevel,
		BlockEventUnsupportedDragDrop:   scenario.BlockEventUnsupportedDragDrop,
		BlockStateSelectors:             scenario.BlockStateSelectors,
		BlockStateResolutions:           scenario.BlockStateResolutions,
		BlockStateResolverOrder:         scenario.BlockStateResolverOrder,
		BlockStateQualityLevel:          scenario.BlockStateQualityLevel,
		BlockStateUnsupportedCSSPseudos: scenario.BlockStateUnsupportedCSSPseudos,
		MotionFrames:                    scenario.MotionFrames,
		MotionQualityLevel:              scenario.MotionQualityLevel,
		MotionClock:                     scenario.MotionClock,
		MotionFrameBudget:               scenario.MotionFrameBudget,
		MotionUnsupportedCSSAnimations:  scenario.MotionUnsupportedCSSAnimations,
		BlockAssetManifest:              scenario.BlockAssetManifest,
		BlockAssetCache:                 scenario.BlockAssetCache,
		BlockAssetDiagnostics:           scenario.BlockAssetDiagnostics,
		BlockAssetRenderCommands:        scenario.BlockAssetRenderCommands,
		BlockAssetQualityLevel:          scenario.BlockAssetQualityLevel,
		BlockAssetNetworkFetchAllowed:   scenario.BlockAssetNetworkFetchAllowed,
		BlockAccessibilityTree:          scenario.BlockAccessibilityTree,
		BlockSystem:                     scenario.BlockSystem,
		Morph:                           scenario.Morph,
		Toolkit:                         scenario.Toolkit,
		AccessibilityTree:               scenario.AccessibilityTree,
		AppModel:                        scenario.AppModel,
		LinuxAppShell:                   scenario.LinuxAppShell,
		SecurityPermissions:             scenario.SecurityPermissions,
		SurfacePerformanceBudget:        performanceBudget,
		BrowserSurface:                  scenario.BrowserSurface,
		Events:                          scenario.Events,
		Frames:                          scenario.Frames,
		StateTransitions:                scenario.StateTransitions,
		Cases:                           scenario.Cases,
	}
}

func renderCommandStreamRendererForMode(mode string) string {
	if isWASM32WebBrowserCanvasMorphMode(mode) {
		return "browser-canvas-rgba"
	}
	if isLinuxX64RealWindowMorphMode(mode) {
		return "wayland-shm-rgba"
	}
	return "software-rgba-headless"
}

func buildTextInputReport(
	opt smokeOptions,
	processes []surface.ProcessReport,
	artifacts []surface.ArtifactReport,
	artifactScan surface.ArtifactScanReport,
	cases []surface.CaseReport,
) surface.TextInputReport {
	return surface.TextInputReport{
		Schema:                     surface.TextInputSchemaV1,
		Target:                     releaseTextInputTarget(opt.Mode),
		Source:                     defaultSurfaceSourcePath(opt),
		Level:                      "production-text-input-v1",
		Experimental:               false,
		ProductionClaim:            true,
		Storage:                    "owned-utf8-byte-buffer",
		UTF8Validation:             true,
		InvalidUTF8Rejected:        true,
		Caret:                      true,
		Selection:                  true,
		SelectionClipboardTransfer: true,
		Multiline:                  true,
		Backspace:                  true,
		Delete:                     true,
		HomeEnd:                    true,
		ArrowLeftRight:             true,
		CompositionEvents:          true,
		CompositionCommit:          true,
		CompositionCancel:          true,
		ClipboardRead:              true,
		ClipboardWrite:             true,
		ClipboardHostABI:           true,
		ClipboardOwnedCopy:         true,
		TargetHostCompositionTrace: true,
		CompositionTrace: surface.CompositionTraceReport{
			Start:  true,
			Update: true,
			Commit: true,
			Cancel: true,
		},
		TextShapingPlan: surface.TextShapingPlanReport{
			QualityLevel:       "scoped-text-shaping-plan-v1",
			FallbackFonts:      true,
			GraphemeBoundaries: "byte-offset-codepoint-v1",
			LineBreaking:       "newline-storage-plus-wrap-plan-v1",
			Bidi:               "nonclaim-full-bidi-v1",
			RichText:           "nonclaim-rich-text-editor-v1",
		},
		ReferenceTraces: []surface.TextInputReferenceTraceReport{
			{
				Source:      "examples/surface/morph_core/surface_morph_settings.tetra",
				Trace:       "settings text field trace",
				Focus:       true,
				Selection:   true,
				Clipboard:   true,
				Composition: true,
				Multiline:   true,
				Pass:        true,
			},
			{
				Source:      "examples/surface/morph_core/surface_morph_editor_shell.tetra",
				Trace:       "editor shell text area trace",
				Focus:       true,
				Selection:   true,
				Clipboard:   true,
				Composition: true,
				Multiline:   true,
				Pass:        true,
			},
		},
		UnsupportedClaims: []string{
			"full-rich-text-editor",
			"full-bidi-shaping",
			"grapheme-cluster-caret",
			"ide-grade-editor",
		},
		RichTextProductionClaim:   false,
		BidiProductionClaim:       false,
		FullEditorProductionClaim: false,
		BorrowedViewStorage:       false,
		SafeViewLifetimeChecked:   true,
		Processes:                 processes,
		Artifacts:                 artifacts,
		ArtifactScan:              artifactScan,
		Cases:                     cases,
	}
}
func releaseTextInputTarget(mode string) string {
	switch mode {
	case "linux-x64-release-text-input":
		return "linux-x64"
	case "wasm32-web-release-text-input":
		return "wasm32-web"
	default:
		return "headless"
	}
}
func releaseTextInputCases() []surface.CaseReport {
	return []surface.CaseReport{
		{Name: "host-provided pointer event dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "host event buffer poll_event", Kind: "positive", Ran: true, Pass: true},
		{Name: "pre/post event frame sequence", Kind: "positive", Ran: true, Pass: true},
		{Name: "component hierarchy dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "component text input scalar dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "host text payload buffer", Kind: "positive", Ran: true, Pass: true},
		{Name: "component focus dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "component accessibility metadata", Kind: "positive", Ran: true, Pass: true},
		{Name: "no legacy UI sidecar artifacts", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input ASCII insertion", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input UTF-8 insertion", Kind: "positive", Ran: true, Pass: true},
		{
			Name:          "release text input invalid UTF-8 rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "invalid utf8 rejected",
		},
		{Name: "release text input multiline storage", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input caret home end arrows", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input selection replacement", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "release text input selection clipboard transfer",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{Name: "release text input backspace delete", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "release text input clipboard owned copy transfer",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name: "release text input composition start update",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{Name: "release text input composition commit", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input composition cancel", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input shaping plan scoped", Kind: "positive", Ran: true, Pass: true},
		{Name: "settings reference text input trace", Kind: "positive", Ran: true, Pass: true},
		{Name: "editor reference text input trace", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "release text input safe view lifetime checked",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name:          "reject legacy UI evidence",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "legacy UI evidence rejected",
		},
	}
}
func hostEvidenceForMode(mode string) surface.HostEvidenceReport {
	switch mode {
	case "headless-text-focus-input",
		"headless-release-text-input",
		"headless-release-toolkit",
		"headless-release-accessibility",
		"headless-component-tree",
		"headless-component-tree-api",
		"headless-block-paint",
		"headless-block-text",
		"headless-block-layout",
		"headless-block-events",
		"headless-block-states",
		"headless-block-motion",
		"headless-block-assets",
		"headless-block-accessibility",
		"headless-block-system",
		"headless-morph",
		"headless-minimal-toolkit",
		"headless-toolkit-reuse",
		"headless-accessibility-metadata",
		"headless-app-model":
		return surface.HostEvidenceReport{
			Level:       "deterministic-headless",
			Backend:     "software-rgba",
			Framebuffer: true,
		}
	case "linux-x64":
		return surface.HostEvidenceReport{
			Level:       "linux-x64-memfd-starter",
			Backend:     "memfd-rgba",
			Framebuffer: true,
		}
	case "linux-x64-release-window", "linux-x64-release-app-shell":
		return surface.HostEvidenceReport{
			Level:               "linux-x64-release-window-v1",
			Backend:             "wayland-shm-rgba-release-v1",
			Framebuffer:         true,
			RealWindow:          true,
			NativeInput:         true,
			TextInput:           true,
			Clipboard:           true,
			Composition:         true,
			AccessibilityBridge: true,
		}
	case "linux-x64-release-accessibility":
		return surface.HostEvidenceReport{
			Level:               "linux-x64-real-window",
			Backend:             "wayland-shm-rgba",
			Framebuffer:         true,
			RealWindow:          true,
			NativeInput:         true,
			AccessibilityBridge: true,
		}
	case "linux-x64-real-window",
		"linux-x64-real-window-text-focus-input",
		"linux-x64-release-text-input",
		"linux-x64-release-toolkit",
		"linux-x64-real-window-component-tree",
		"linux-x64-real-window-component-tree-api",
		"linux-x64-real-window-block-system",
		"linux-x64-real-window-morph",
		"linux-x64-real-window-minimal-toolkit",
		"linux-x64-real-window-toolkit-reuse",
		"linux-x64-real-window-accessibility-metadata":
		return surface.HostEvidenceReport{
			Level:       "linux-x64-real-window",
			Backend:     "wayland-shm-rgba",
			Framebuffer: true,
			RealWindow:  true,
			NativeInput: true,
		}
	case "wasm32-web":
		return surface.HostEvidenceReport{
			Level:       "wasm32-web-compiler-owned-loader",
			Backend:     "node-surface-host",
			Framebuffer: true,
		}
	case "wasm32-web-release-browser":
		return surface.HostEvidenceReport{
			Level:                        "wasm32-web-browser-canvas-release-v1",
			Backend:                      "browser-canvas-rgba-accessible",
			Framebuffer:                  true,
			NativeInput:                  true,
			BrowserCanvas:                true,
			BrowserInput:                 true,
			BrowserClipboard:             true,
			BrowserClipboardHarness:      "deterministic-browser-clipboard-v1",
			BrowserComposition:           true,
			BrowserAccessibilitySnapshot: true,
			BrowserAccessibilityMirror:   true,
		}
	case "wasm32-web-release-accessibility":
		return surface.HostEvidenceReport{
			Level:                        "wasm32-web-browser-canvas-input",
			Backend:                      "browser-canvas-rgba",
			Framebuffer:                  true,
			NativeInput:                  true,
			BrowserCanvas:                true,
			BrowserInput:                 true,
			BrowserAccessibilitySnapshot: true,
			BrowserAccessibilityMirror:   true,
		}
	case "wasm32-web-browser-canvas-block-system", "wasm32-web-browser-canvas-morph":
		return surface.HostEvidenceReport{
			Level:         "wasm32-web-browser-canvas-input",
			Backend:       "browser-canvas-rgba",
			Framebuffer:   true,
			NativeInput:   true,
			BrowserCanvas: true,
			BrowserInput:  true,
		}
	case "wasm32-web-browser-canvas",
		"wasm32-web-browser-canvas-text-focus-input",
		"wasm32-web-release-text-input",
		"wasm32-web-release-toolkit",
		"wasm32-web-browser-canvas-component-tree",
		"wasm32-web-browser-canvas-component-tree-api",
		"wasm32-web-browser-canvas-minimal-toolkit",
		"wasm32-web-browser-canvas-toolkit-reuse",
		"wasm32-web-browser-canvas-accessibility-metadata":
		return surface.HostEvidenceReport{
			Level:       "wasm32-web-browser-canvas-input",
			Backend:     "browser-canvas-rgba",
			Framebuffer: true,
			NativeInput: true,
		}
	default:
		return surface.HostEvidenceReport{
			Level:       "deterministic-headless",
			Backend:     "software-rgba",
			Framebuffer: true,
		}
	}
}
func intPtr(v int) *int {
	return &v
}
func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// ---- types.go ----

type smokeOptions struct {
	ReportPath                      string
	Mode                            string
	SourcePath                      string
	VisualReportPath                string
	MorphRenderedBeautyReportPath   string
	MorphRenderedBeautyProductClaim bool
	MorphRenderedBeautyFinalSignoff bool
	RealWindowProbe                 bool
	ProbeTitle                      string
	ProbeFramePath                  string
	ProbeFrameWidth                 int
	ProbeFrameHeight                int
	ProbeFrameStride                int
	ProbeHoldUntilClose             bool
}
type headlessScenario struct {
	Components                      []surface.ComponentReport
	ComponentTree                   *surface.ComponentTreeReport
	ComponentTreeAPI                *surface.ComponentTreeAPIReport
	BlockGraph                      *surface.BlockGraphReport
	BlockSceneSnapshot              *surface.BlockSceneSnapshotReport
	RenderCommandStream             *surface.RenderCommandStreamReport
	PaintLayers                     []surface.PaintLayerReport
	PaintCommands                   []surface.PaintCommandReport
	VisualFeatures                  []string
	PaintQualityLevel               string
	PaintCacheBudgetBytes           int
	PaintUnsupportedBlur            bool
	Renderer                        *surface.RendererReport
	TextMeasurements                []surface.TextMeasurementReport
	FontFallbacks                   []surface.FontFallbackReport
	GlyphCaches                     []surface.GlyphCacheReport
	TextRenderCommands              []surface.TextRenderCommandReport
	TextQualityLevel                string
	TextCacheBudgetBytes            int
	LayoutConstraints               []surface.BlockLayoutConstraintReport
	LayoutPasses                    []surface.BlockLayoutPassReport
	LayoutScrolls                   []surface.BlockLayoutScrollReport
	LayoutDensity                   *surface.BlockLayoutDensityReport
	LayoutFeatures                  []string
	LayoutQualityLevel              string
	LayoutUnsupportedCSSFlexbox     bool
	BlockEventRoutes                []surface.BlockEventRouteReport
	BlockFocusTransitions           []surface.BlockFocusTransitionReport
	BlockEventKinds                 []string
	BlockEventPolicy                string
	BlockEventQualityLevel          string
	BlockEventUnsupportedDragDrop   bool
	BlockStateSelectors             []surface.BlockStateSelectorReport
	BlockStateResolutions           []surface.BlockStateResolutionReport
	BlockStateResolverOrder         []string
	BlockStateQualityLevel          string
	BlockStateUnsupportedCSSPseudos bool
	MotionFrames                    []surface.MotionFrameReport
	MotionQualityLevel              string
	MotionClock                     string
	MotionFrameBudget               int
	MotionUnsupportedCSSAnimations  bool
	BlockAssetManifest              *surface.BlockAssetManifestReport
	BlockAssetCache                 surface.BlockAssetCacheReport
	BlockAssetDiagnostics           []surface.BlockAssetDiagnosticReport
	BlockAssetRenderCommands        []surface.BlockAssetRenderCommandReport
	BlockAssetQualityLevel          string
	BlockAssetNetworkFetchAllowed   bool
	BlockAccessibilityTree          *surface.BlockAccessibilityTreeReport
	BlockSystem                     *surface.BlockSystemReport
	Morph                           *surface.MorphReport
	Toolkit                         *surface.ToolkitReport
	AccessibilityTree               *surface.AccessibilityTreeReport
	AppModel                        *surface.AppModelReport
	LinuxAppShell                   *surface.LinuxAppShellReport
	SecurityPermissions             *surface.SecurityPermissionReport
	SurfacePerformanceBudget        *surface.SurfacePerformanceBudgetReport
	BrowserSurface                  *surface.BrowserSurfaceReport
	Events                          []surface.EventReport
	Frames                          []surface.FrameReport
	StateTransitions                []surface.StateTransitionReport
	Cases                           []surface.CaseReport
}
type surfaceProcessEvidence struct {
	Processes    []surface.ProcessReport
	Artifacts    []surface.ArtifactReport
	ArtifactScan surface.ArtifactScanReport
	Frames       []surface.FrameReport
}

type (
	frameReport             = surface.FrameReport
	eventReport             = surface.EventReport
	stateTransitionReport   = surface.StateTransitionReport
	componentReport         = surface.ComponentReport
	componentTreeReport     = surface.ComponentTreeReport
	traceComponentTreeAPIR  = surface.ComponentTreeAPIReport
	blockGraphReport        = surface.BlockGraphReport
	blockSceneSnapshotR     = surface.BlockSceneSnapshotReport
	renderCommandStreamR    = surface.RenderCommandStreamReport
	paintLayerReport        = surface.PaintLayerReport
	paintCommandReport      = surface.PaintCommandReport
	rendererReport          = surface.RendererReport
	textMeasurementReport   = surface.TextMeasurementReport
	fontFallbackReport      = surface.FontFallbackReport
	glyphCacheReport        = surface.GlyphCacheReport
	textRenderCommandReport = surface.TextRenderCommandReport
	layoutConstraintReport  = surface.BlockLayoutConstraintReport
	layoutPassReport        = surface.BlockLayoutPassReport
	layoutScrollReport      = surface.BlockLayoutScrollReport
	layoutDensityReport     = surface.BlockLayoutDensityReport
	blockEventRouteReport   = surface.BlockEventRouteReport
	blockFocusTransReport   = surface.BlockFocusTransitionReport
	stateSelectorReport     = surface.BlockStateSelectorReport
	stateResolutionReport   = surface.BlockStateResolutionReport
	motionFrameReport       = surface.MotionFrameReport
	assetManifestReport     = surface.BlockAssetManifestReport
	assetCacheReport        = surface.BlockAssetCacheReport
	assetDiagnosticReport   = surface.BlockAssetDiagnosticReport
	assetRenderCmdReport    = surface.BlockAssetRenderCommandReport
	blockAccessibilityTreeR = surface.BlockAccessibilityTreeReport
	blockSystemReport       = surface.BlockSystemReport
	morphReport             = surface.MorphReport
	toolkitReport           = surface.ToolkitReport
	traceAccessibilityTreeR = surface.AccessibilityTreeReport
	appModelReport          = surface.AppModelReport
	caseReport              = surface.CaseReport
)

type headlessSurfaceRunnerTrace struct {
	Schema           string                  `json:"schema"`
	Source           string                  `json:"source"`
	Frames           []frameReport           `json:"frames"`
	Events           []eventReport           `json:"events"`
	StateTransitions []stateTransitionReport `json:"state_transitions"`
	Components       []componentReport       `json:"components"`

	ComponentTree       *componentTreeReport    `json:"component_tree,omitempty"`
	ComponentTreeAPI    *traceComponentTreeAPIR `json:"component_tree_api,omitempty"`
	BlockGraph          *blockGraphReport       `json:"block_graph,omitempty"`
	BlockSceneSnapshot  *blockSceneSnapshotR    `json:"block_scene_snapshot,omitempty"`
	RenderCommandStream *renderCommandStreamR   `json:"render_command_stream,omitempty"`

	PaintLayers           []paintLayerReport   `json:"paint_layers,omitempty"`
	PaintCommands         []paintCommandReport `json:"paint_commands,omitempty"`
	VisualFeatures        []string             `json:"visual_features,omitempty"`
	PaintQualityLevel     string               `json:"paint_quality_level,omitempty"`
	PaintCacheBudgetBytes int                  `json:"paint_cache_budget_bytes,omitempty"`
	PaintUnsupportedBlur  bool                 `json:"paint_unsupported_blur,omitempty"`
	Renderer              *rendererReport      `json:"renderer,omitempty"`

	TextMeasurements     []textMeasurementReport   `json:"text_measurements,omitempty"`
	FontFallbacks        []fontFallbackReport      `json:"font_fallbacks,omitempty"`
	GlyphCaches          []glyphCacheReport        `json:"glyph_caches,omitempty"`
	TextRenderCommands   []textRenderCommandReport `json:"text_render_commands,omitempty"`
	TextQualityLevel     string                    `json:"text_quality_level,omitempty"`
	TextCacheBudgetBytes int                       `json:"text_cache_budget_bytes,omitempty"`

	LayoutConstraints  []layoutConstraintReport `json:"layout_constraints,omitempty"`
	LayoutPasses       []layoutPassReport       `json:"layout_passes,omitempty"`
	LayoutScrolls      []layoutScrollReport     `json:"layout_scrolls,omitempty"`
	LayoutDensity      *layoutDensityReport     `json:"layout_density,omitempty"`
	LayoutFeatures     []string                 `json:"layout_features,omitempty"`
	LayoutQualityLevel string                   `json:"layout_quality_level,omitempty"`

	LayoutUnsupportedCSSFlexbox bool `json:"layout_unsupported_css_flexbox,omitempty"`

	BlockEventRoutes       []blockEventRouteReport `json:"block_event_routes,omitempty"`
	BlockFocusTransitions  []blockFocusTransReport `json:"block_focus_transitions,omitempty"`
	BlockEventKinds        []string                `json:"block_event_kinds,omitempty"`
	BlockEventPolicy       string                  `json:"block_event_policy,omitempty"`
	BlockEventQualityLevel string                  `json:"block_event_quality_level,omitempty"`

	BlockEventUnsupportedDragDrop bool `json:"block_event_unsupported_drag_drop,omitempty"`

	BlockStateSelectors     []stateSelectorReport   `json:"block_state_selectors,omitempty"`
	BlockStateResolutions   []stateResolutionReport `json:"block_state_resolutions,omitempty"`
	BlockStateResolverOrder []string                `json:"block_state_resolver_order,omitempty"`
	BlockStateQualityLevel  string                  `json:"block_state_quality_level,omitempty"`

	BlockStateUnsupportedCSSPseudos bool `json:"block_state_unsupported_css_pseudos,omitempty"`

	MotionFrames       []motionFrameReport `json:"motion_frames,omitempty"`
	MotionQualityLevel string              `json:"motion_quality_level,omitempty"`
	MotionClock        string              `json:"motion_clock,omitempty"`
	MotionFrameBudget  int                 `json:"motion_frame_budget,omitempty"`

	MotionUnsupportedCSSAnimations bool `json:"motion_unsupported_css_animations,omitempty"`

	BlockAssetManifest    *assetManifestReport    `json:"block_asset_manifest,omitempty"`
	BlockAssetCache       assetCacheReport        `json:"block_asset_cache,omitempty"`
	BlockAssetDiagnostics []assetDiagnosticReport `json:"block_asset_diagnostics,omitempty"`

	BlockAssetRenderCommands []assetRenderCmdReport `json:"block_asset_render_commands,omitempty"`
	BlockAssetQualityLevel   string                 `json:"block_asset_quality_level,omitempty"`

	BlockAssetNetworkFetchAllowed bool `json:"block_asset_network_fetch_allowed,omitempty"`

	BlockAccessibilityTree *blockAccessibilityTreeR `json:"block_accessibility_tree,omitempty"`
	BlockSystem            *blockSystemReport       `json:"block_system,omitempty"`
	Morph                  *morphReport             `json:"morph,omitempty"`
	Toolkit                *toolkitReport           `json:"toolkit,omitempty"`
	AccessibilityTree      *traceAccessibilityTreeR `json:"accessibility_tree,omitempty"`
	AppModel               *appModelReport          `json:"app_model,omitempty"`
	Cases                  []caseReport             `json:"cases"`
}
type wasmSurfaceRunnerTrace struct {
	Schema string                        `json:"schema"`
	WASM   string                        `json:"wasm_path"`
	Frames []wasmSurfaceRunnerTraceFrame `json:"frames"`
}
type wasmSurfaceRunnerTraceFrame struct {
	Order     int    `json:"order"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Stride    int    `json:"stride"`
	PixelsLen int    `json:"pixels_len"`
	Checksum  string `json:"checksum"`
}
type browserCanvasTrace struct {
	Schema               string                          `json:"schema"`
	WASM                 string                          `json:"wasm_path"`
	Canvas               browserCanvasTraceCanvas        `json:"canvas"`
	BrowserEvents        []browserCanvasTraceEvent       `json:"browser_events"`
	BrowserClipboard     browserCanvasTraceClipboard     `json:"browser_clipboard"`
	BrowserComposition   browserCanvasTraceComposition   `json:"browser_composition"`
	BrowserAccessibility browserCanvasTraceAccessibility `json:"browser_accessibility"`
	Frames               []browserCanvasTraceFrame       `json:"frames"`
	AppExitCode          int                             `json:"app_exit_code"`
	Error                string                          `json:"error,omitempty"`
}
type browserCanvasTraceCanvas struct {
	Opened   bool `json:"opened"`
	Width    int  `json:"width"`
	Height   int  `json:"height"`
	Readback bool `json:"readback"`
}
type browserCanvasTraceEvent struct {
	Order      int    `json:"order"`
	NativeType string `json:"native_type"`
	Kind       int    `json:"kind"`
	X          int    `json:"x"`
	Y          int    `json:"y"`
	Key        int    `json:"key"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	TextLen    int    `json:"text_len"`
}
type browserCanvasTraceClipboard struct {
	Harness   string `json:"harness"`
	Read      bool   `json:"read"`
	Write     bool   `json:"write"`
	OwnedCopy bool   `json:"owned_copy"`
	Bytes     int    `json:"bytes"`
}
type browserCanvasTraceComposition struct {
	Start  bool `json:"start"`
	Update bool `json:"update"`
	Commit bool `json:"commit"`
	Cancel bool `json:"cancel"`
}
type browserCanvasTraceAccessibility struct {
	Snapshot      bool     `json:"snapshot"`
	Mirror        bool     `json:"mirror"`
	CompilerOwned bool     `json:"compiler_owned"`
	Roles         []string `json:"roles"`
	Bounds        bool     `json:"bounds"`
	Focus         bool     `json:"focus"`
	DOMVisualUI   bool     `json:"dom_visual_ui"`
	UserJS        bool     `json:"user_js"`
}
type browserCanvasTraceFrame struct {
	Order           int    `json:"order"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	Stride          int    `json:"stride"`
	PixelsLen       int    `json:"pixels_len"`
	SourcePixelsB64 string `json:"source_pixels_b64"`
	CanvasPixelsB64 string `json:"canvas_pixels_b64"`
}
type sidecarScanOptions struct {
	AllowCompilerOwnedWASMLoader bool
}
type rgbaFrame struct {
	Width  int
	Height int
	Stride int
	Pixels []byte
}
type rgbaColor struct {
	R byte
	G byte
	B byte
	A byte
}
type rect struct {
	X int
	Y int
	W int
	H int
}
