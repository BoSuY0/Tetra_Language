package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"tetra_language/tools/validators/surface"
)

func main() {
	var opt smokeOptions
	flag.StringVar(&opt.ReportPath, "report", "", "path to write tetra.surface.runtime.v1 report")
	flag.StringVar(&opt.Mode, "mode", "headless", "Surface smoke mode")
	flag.StringVar(&opt.SourcePath, "source", "examples/surface_counter.tetra", "Surface app source path")
	flag.BoolVar(&opt.RealWindowProbe, "real-window-probe", false, "run the linux-x64 real-window probe helper")
	flag.StringVar(&opt.ProbeTitle, "probe-title", "Tetra Surface Real Window Probe", "real-window probe title")
	flag.StringVar(&opt.ProbeFramePath, "probe-frame", "", "raw RGBA frame path for the real-window probe")
	flag.IntVar(&opt.ProbeFrameWidth, "probe-width", 400, "real-window probe frame width")
	flag.IntVar(&opt.ProbeFrameHeight, "probe-height", 240, "real-window probe frame height")
	flag.IntVar(&opt.ProbeFrameStride, "probe-stride", 1600, "real-window probe frame stride")
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
		report := buildTextInputReport(opt, evidence.Processes, evidence.Artifacts, evidence.ArtifactScan, releaseTextInputCases())
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
	if shouldRetargetSurfaceTemplateScenario(opt) {
		source := defaultSurfaceSourcePath(opt)
		retargetScenarioToSource(&scenario, source, "main")
		if isMorphMode(opt.Mode) {
			scenario.Morph = morphReportForScenario(source, scenario)
		}
	}
	if opt.Mode == "wasm32-web-browser-canvas-block-system" {
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
		scenario.BlockSystem = blockSystemReportForLinuxX64RealWindowScenario(defaultSurfaceSourcePath(opt), scenario.Frames)
		attachBlockSystemMemoryBudget(&scenario)
	}
	if opt.Mode == "wasm32-web-browser-canvas-block-system" {
		scenario.BlockSystem = blockSystemReportForWASM32WebBrowserCanvasScenario(defaultSurfaceSourcePath(opt), scenario.Frames)
		attachBlockSystemMemoryBudget(&scenario)
	}
	report := buildReport(opt, "linux-x64", evidence.Processes, evidence.Artifacts, evidence.ArtifactScan, scenario)
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
}
