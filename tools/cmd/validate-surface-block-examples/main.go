package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	compiler "tetra_language/compiler"
)

var defaultExamplePaths = []string{
	"examples/surface/block_apps/surface_block_command_palette.tetra",
	"examples/surface/block_apps/surface_block_project_dashboard.tetra",
	"examples/surface/block_apps/surface_block_settings.tetra",
	"examples/surface/block_apps/surface_block_editor_shell.tetra",
	"examples/surface/block_apps/surface_block_glass_panel.tetra",
}

type examplesReport struct {
	Schema         string          `json:"schema"`
	QualityLevel   string          `json:"quality_level"`
	ExampleCount   int             `json:"example_count"`
	Examples       []exampleReport `json:"examples"`
	NegativeGuards map[string]bool `json:"negative_guards"`
	FeatureTotals  map[string]int  `json:"feature_totals"`
	Pass           bool            `json:"pass"`
}

type exampleReport struct {
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

func main() {
	var reportPath string
	flag.StringVar(&reportPath, "report", "", "write a tetra.surface.block-examples.v1 report")
	flag.Parse()

	paths := flag.Args()
	if len(paths) == 0 {
		paths = defaultExamplePaths
	}

	report, err := validateExampleFiles(paths, reportPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "validate-surface-block-examples: %v\n", err)
		os.Exit(1)
	}

	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "validate-surface-block-examples: marshal report: %v\n", err)
		os.Exit(1)
	}
	if reportPath == "" {
		fmt.Println(string(raw))
		return
	}
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "validate-surface-block-examples: create report dir: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(reportPath, append(raw, '\n'), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "validate-surface-block-examples: write report: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Surface Block examples report: %s\n", reportPath)
}

func validateExampleFiles(paths []string, reportPath string) (examplesReport, error) {
	report := examplesReport{
		Schema:       "tetra.surface.block-examples.v1",
		QualityLevel: "block-only-polished-examples-v1",
		ExampleCount: len(paths),
		NegativeGuards: map[string]bool{
			"core_widget_usage_rejected":        true,
			"missing_accessibility_rejected":    true,
			"missing_hover_focus_pressed_state": true,
		},
		FeatureTotals: map[string]int{},
		Pass:          true,
	}

	artifactDir := ""
	if reportPath != "" {
		artifactDir = filepath.Join(filepath.Dir(reportPath), "surface-block-examples-artifacts")
		if err := os.MkdirAll(artifactDir, 0o755); err != nil {
			return report, fmt.Errorf("create artifact dir: %w", err)
		}
	}

	for _, path := range paths {
		example, err := validateExampleFile(path, artifactDir)
		if err != nil {
			return report, err
		}
		report.Examples = append(report.Examples, example)
		addFeatureTotals(report.FeatureTotals, example)
	}
	return report, nil
}

func validateExampleFile(path string, artifactDir string) (exampleReport, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return exampleReport{}, fmt.Errorf("%s: read: %w", path, err)
	}
	source := string(raw)
	if err := validateExampleSource(path, source); err != nil {
		return exampleReport{}, err
	}

	world, err := compiler.LoadWorld(path)
	if err != nil {
		return exampleReport{}, fmt.Errorf("%s: LoadWorld: %w", path, err)
	}
	if _, err := compiler.CheckWorld(world); err != nil {
		return exampleReport{}, fmt.Errorf("%s: CheckWorld: %w", path, err)
	}

	artifact := ""
	exitCode := 0
	if artifactDir != "" {
		artifact = filepath.Join(artifactDir, strings.TrimSuffix(filepath.Base(path), ".tetra"))
		if _, err := compiler.BuildFileWithStatsOpt(
			path,
			artifact,
			"linux-x64",
			compiler.BuildOptions{Jobs: 1},
		); err != nil {
			return exampleReport{}, fmt.Errorf("%s: BuildFileWithStatsOpt: %w", path, err)
		}
		cmd := exec.Command(artifact)
		if output, err := cmd.CombinedOutput(); err != nil {
			exitCode = commandExitCode(err)
			return exampleReport{}, fmt.Errorf(
				"%s: run exit %d: %w\n%s",
				path,
				exitCode,
				err,
				output,
			)
		}
	}

	return exampleReport{
		Path:                  path,
		BlockOnly:             true,
		Compiles:              true,
		Runs:                  artifactDir != "",
		ExitCode:              exitCode,
		ThemeTokens:           hasThemeTokens(source),
		PaintEvidence:         strings.Contains(source, "block.paint_stack"),
		LayoutEvidence:        strings.Contains(source, "block.layout_"),
		TextEvidence:          strings.Contains(source, "block.text_"),
		AssetEvidence:         strings.Contains(source, "block.asset_"),
		AccessibilityEvidence: hasAccessibilityRole(source),
		HoverEvidence:         strings.Contains(source, "block.state_selector_hover()"),
		FocusEvidence:         strings.Contains(source, "block.state_selector_focused()"),
		PressedEvidence:       strings.Contains(source, "block.state_selector_pressed()"),
		MotionEvidence:        strings.Contains(source, "block.motion_"),
		ChecksumEvidence:      strings.Contains(source, "scene_checksum"),
		Modules:               []string{"lib.core.surface", "lib.core.block"},
		Artifact:              artifact,
	}, nil
}

func validateExampleSource(path string, source string) error {
	for _, forbidden := range []string{
		"import lib.core.widgets",
		"lib.core.widgets",
		"widgets.",
		"widgets.Button",
		"widgets.TextBox",
		"Button(",
		"TextBox(",
		"Card(",
		"Modal(",
	} {
		if strings.Contains(source, forbidden) {
			return fmt.Errorf("%s: forbidden widget/component marker %q", path, forbidden)
		}
	}

	lower := strings.ToLower(source)
	for _, forbidden := range []string{
		"react",
		"electron",
		"dom ui",
		"user js",
		"user javascript",
		".ui.html",
		".ui.json",
		".ui.web.mjs",
	} {
		if strings.Contains(lower, forbidden) {
			return fmt.Errorf("%s: forbidden platform/sidecar marker %q", path, forbidden)
		}
	}

	required := []struct {
		name string
		ok   bool
	}{
		{
			name: "surface import",
			ok:   strings.Contains(source, "import lib.core.surface as surface"),
		},
		{name: "block import", ok: strings.Contains(source, "import lib.core.block as block")},
		{name: "dark/light theme tokens", ok: hasThemeTokens(source)},
		{name: "layout evidence", ok: strings.Contains(source, "block.layout_")},
		{name: "paint stack evidence", ok: strings.Contains(source, "block.paint_stack")},
		{name: "text evidence", ok: strings.Contains(source, "block.text_")},
		{name: "asset evidence", ok: strings.Contains(source, "block.asset_")},
		{name: "accessibility role evidence", ok: hasAccessibilityRole(source)},
		{
			name: "hover state evidence",
			ok:   strings.Contains(source, "block.state_selector_hover()"),
		},
		{
			name: "focus state evidence",
			ok:   strings.Contains(source, "block.state_selector_focused()"),
		},
		{
			name: "pressed state evidence",
			ok:   strings.Contains(source, "block.state_selector_pressed()"),
		},
		{name: "motion evidence", ok: strings.Contains(source, "block.motion_")},
		{name: "scene checksum evidence", ok: strings.Contains(source, "scene_checksum")},
	}
	for _, requirement := range required {
		if !requirement.ok {
			return fmt.Errorf("%s: missing %s", path, requirement.name)
		}
	}
	return nil
}

func hasThemeTokens(source string) bool {
	return strings.Contains(source, "theme_dark") && strings.Contains(source, "theme_light")
}

func hasAccessibilityRole(source string) bool {
	for _, marker := range []string{
		"block.accessibility_button(",
		"block.accessibility_text(",
		"block.accessibility_label_for(",
		"block.accessibility_button_labelled_by(",
		"block.accessibility_textbox_labelled_by(",
	} {
		if strings.Contains(source, marker) {
			return true
		}
	}
	return false
}

func addFeatureTotals(totals map[string]int, example exampleReport) {
	if example.ThemeTokens {
		totals["theme_tokens"]++
	}
	if example.PaintEvidence {
		totals["paint"]++
	}
	if example.LayoutEvidence {
		totals["layout"]++
	}
	if example.TextEvidence {
		totals["text"]++
	}
	if example.AssetEvidence {
		totals["asset"]++
	}
	if example.AccessibilityEvidence {
		totals["accessibility"]++
	}
	if example.HoverEvidence && example.FocusEvidence && example.PressedEvidence {
		totals["hover_focus_pressed"]++
	}
	if example.MotionEvidence {
		totals["motion"]++
	}
	if example.ChecksumEvidence {
		totals["checksum"]++
	}
}

func commandExitCode(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}
