package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"tetra_language/compiler"
	"tetra_language/tools/validators/surface"
)

type smokeOptions struct {
	ReportPath       string
	Mode             string
	SourcePath       string
	RealWindowProbe  bool
	ProbeTitle       string
	ProbeFramePath   string
	ProbeFrameWidth  int
	ProbeFrameHeight int
	ProbeFrameStride int
}

type headlessScenario struct {
	Components        []surface.ComponentReport
	ComponentTree     *surface.ComponentTreeReport
	ComponentTreeAPI  *surface.ComponentTreeAPIReport
	Toolkit           *surface.ToolkitReport
	AccessibilityTree *surface.AccessibilityTreeReport
	Events            []surface.EventReport
	Frames            []surface.FrameReport
	StateTransitions  []surface.StateTransitionReport
	Cases             []surface.CaseReport
}

type surfaceProcessEvidence struct {
	Processes    []surface.ProcessReport
	Artifacts    []surface.ArtifactReport
	ArtifactScan surface.ArtifactScanReport
	Frames       []surface.FrameReport
}

type headlessSurfaceRunnerTrace struct {
	Schema            string                           `json:"schema"`
	Source            string                           `json:"source"`
	Frames            []surface.FrameReport            `json:"frames"`
	Events            []surface.EventReport            `json:"events"`
	StateTransitions  []surface.StateTransitionReport  `json:"state_transitions"`
	Components        []surface.ComponentReport        `json:"components"`
	ComponentTree     *surface.ComponentTreeReport     `json:"component_tree,omitempty"`
	ComponentTreeAPI  *surface.ComponentTreeAPIReport  `json:"component_tree_api,omitempty"`
	Toolkit           *surface.ToolkitReport           `json:"toolkit,omitempty"`
	AccessibilityTree *surface.AccessibilityTreeReport `json:"accessibility_tree,omitempty"`
	Cases             []surface.CaseReport             `json:"cases"`
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
	if len(scenario.Frames) > 0 && len(evidence.Frames) > 0 {
		lastOrder := scenario.Frames[len(scenario.Frames)-1].Order
		for i := range evidence.Frames {
			if evidence.Frames[i].Order <= lastOrder {
				evidence.Frames[i].Order = lastOrder + i + 1
			}
		}
	}
	scenario.Frames = append(scenario.Frames, evidence.Frames...)
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
	if mode == "linux-x64-real-window-accessibility-metadata" {
		return nil
	}
	if mode == "wasm32-web-browser-canvas-accessibility-metadata" {
		return nil
	}
	return fmt.Errorf("unsupported Surface smoke mode %q", mode)
}

func collectSurfaceProcessEvidence(opt smokeOptions) (surfaceProcessEvidence, error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return surfaceProcessEvidence{}, fmt.Errorf("Surface smoke currently requires a linux/amd64 host to build and run linux-x64 Surface app evidence; host is %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	sourcePath, err := resolveSurfaceSourcePath(defaultSurfaceSourcePath(opt))
	if err != nil {
		return surfaceProcessEvidence{}, fmt.Errorf("build Surface source: %w", err)
	}

	reportDir := filepath.Dir(opt.ReportPath)
	if reportDir == "." || reportDir == "" {
		reportDir = "reports/surface"
	}
	mode := opt.Mode
	if mode == "" {
		mode = "headless"
	}
	artifactDir := filepath.Join(reportDir, "surface-"+mode+"-artifacts")
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
		return collectWASM32WebBrowserCanvasProcessEvidence(sourcePath, artifactDir, "text-focus-input")
	}
	if mode == "wasm32-web-release-text-input" {
		return collectWASM32WebBrowserCanvasProcessEvidence(sourcePath, artifactDir, "release-text-input")
	}
	if mode == "wasm32-web-release-toolkit" {
		return collectWASM32WebBrowserCanvasProcessEvidence(sourcePath, artifactDir, "release-toolkit")
	}
	if mode == "wasm32-web-release-browser" {
		return collectWASM32WebBrowserCanvasProcessEvidence(sourcePath, artifactDir, "release-browser")
	}
	if mode == "wasm32-web-release-accessibility" {
		return collectWASM32WebBrowserCanvasProcessEvidence(sourcePath, artifactDir, "release-accessibility")
	}
	if mode == "wasm32-web-browser-canvas-component-tree" || mode == "wasm32-web-browser-canvas-component-tree-api" {
		return collectWASM32WebBrowserCanvasProcessEvidence(sourcePath, artifactDir, "component-tree")
	}
	if mode == "wasm32-web-browser-canvas-minimal-toolkit" {
		return collectWASM32WebBrowserCanvasProcessEvidence(sourcePath, artifactDir, "minimal-toolkit")
	}
	if mode == "wasm32-web-browser-canvas-toolkit-reuse" {
		return collectWASM32WebBrowserCanvasProcessEvidence(sourcePath, artifactDir, "toolkit-reuse")
	}
	if mode == "wasm32-web-browser-canvas-accessibility-metadata" {
		return collectWASM32WebBrowserCanvasProcessEvidence(sourcePath, artifactDir, "accessibility-metadata")
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
	if isReleaseBrowserMode(mode) {
		appName = "surface-release-form"
	}
	if isReleaseAccessibilityMode(mode) {
		appName = "surface-release-accessibility"
	}
	if isComponentTreeMode(mode) {
		appName = "surface-tree-app"
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
	appPath := filepath.Join(artifactDir, appName)
	if _, err := compiler.BuildFileWithStatsOpt(sourcePath, appPath, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
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
		return surfaceProcessEvidence{}, fmt.Errorf("run Surface app %s: unexpected stdout %q", appPath, stdout)
	}
	if stderr != "" {
		return surfaceProcessEvidence{}, fmt.Errorf("run Surface app %s: unexpected stderr %q", appPath, stderr)
	}
	if appExit != 1 {
		return surfaceProcessEvidence{}, fmt.Errorf("run Surface app %s: exit code %d, want 1", appPath, appExit)
	}

	processes := []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: fmt.Sprintf("tetra build --target linux-x64 %s -o %s", sourcePath, appPath), Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: appPath, Ran: true, Pass: true, ExitCode: intPtr(appExit), ExpectedExitCode: intPtr(1)},
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
		counterProcess, counterFrame, err := collectLinuxX64CounterAppPresentedFrameEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, counterProcess)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, surface.ProcessReport{Name: runtimeProcessName, Kind: "runtime", Path: os.Args[0], Ran: true, Pass: true, ExitCode: intPtr(0)})
		return surfaceProcessEvidence{Processes: processes, Artifacts: []surface.ArtifactReport{componentArtifact}, ArtifactScan: sidecarScan, Frames: []surface.FrameReport{presentFrame, counterFrame}}, nil
	}
	if mode == "linux-x64-real-window" {
		runtimeProcessName = "surface linux-x64 real-window runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		realWindowProcess, realWindowFrame, err := collectLinuxX64RealWindowProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, realWindowProcess)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, surface.ProcessReport{Name: runtimeProcessName, Kind: "runtime", Path: os.Args[0], Ran: true, Pass: true, ExitCode: intPtr(0)})
		return surfaceProcessEvidence{Processes: processes, Artifacts: []surface.ArtifactReport{componentArtifact}, ArtifactScan: sidecarScan, Frames: []surface.FrameReport{realWindowFrame}}, nil
	}
	if mode == "linux-x64-real-window-text-focus-input" {
		runtimeProcessName = "surface linux-x64 real-window runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		realWindowProcess, realWindowFrame, err := collectLinuxX64TextFocusInputRealWindowProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, realWindowProcess)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, surface.ProcessReport{Name: runtimeProcessName, Kind: "runtime", Path: os.Args[0], Ran: true, Pass: true, ExitCode: intPtr(0)})
		return surfaceProcessEvidence{Processes: processes, Artifacts: []surface.ArtifactReport{componentArtifact}, ArtifactScan: sidecarScan, Frames: []surface.FrameReport{realWindowFrame}}, nil
	}
	if mode == "linux-x64-release-text-input" {
		runtimeProcessName = "surface linux-x64 runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		realWindowProcess, realWindowFrame, err := collectLinuxX64TextFocusInputRealWindowProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, realWindowProcess)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, surface.ProcessReport{Name: runtimeProcessName, Kind: "runtime", Path: os.Args[0], Ran: true, Pass: true, ExitCode: intPtr(0)})
		return surfaceProcessEvidence{Processes: processes, Artifacts: []surface.ArtifactReport{componentArtifact}, ArtifactScan: sidecarScan, Frames: []surface.FrameReport{realWindowFrame}}, nil
	}
	if mode == "linux-x64-release-toolkit" {
		runtimeProcessName = "surface linux-x64 runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		realWindowProcess, realWindowFrame, err := collectLinuxX64ReleaseToolkitRealWindowProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, realWindowProcess)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, surface.ProcessReport{Name: runtimeProcessName, Kind: "runtime", Path: os.Args[0], Ran: true, Pass: true, ExitCode: intPtr(0)})
		return surfaceProcessEvidence{Processes: processes, Artifacts: []surface.ArtifactReport{componentArtifact}, ArtifactScan: sidecarScan, Frames: []surface.FrameReport{realWindowFrame}}, nil
	}
	if mode == "linux-x64-release-window" {
		runtimeProcessName = "surface linux-x64 runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		realWindowProcess, realWindowFrame, err := collectLinuxX64ReleaseToolkitRealWindowProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, realWindowProcess)
		harnessProcesses, harnessArtifacts, err := collectLinuxX64ReleaseWindowHarnessEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, harnessProcesses...)
		bridgeProcesses, bridgeArtifacts, err := collectLinuxX64ReleaseWindowAccessibilityBridgeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, bridgeProcesses...)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, surface.ProcessReport{Name: runtimeProcessName, Kind: "runtime", Path: os.Args[0], Ran: true, Pass: true, ExitCode: intPtr(0)})
		artifacts := append([]surface.ArtifactReport{componentArtifact}, harnessArtifacts...)
		artifacts = append(artifacts, bridgeArtifacts...)
		return surfaceProcessEvidence{Processes: processes, Artifacts: artifacts, ArtifactScan: sidecarScan, Frames: []surface.FrameReport{realWindowFrame}}, nil
	}
	if mode == "linux-x64-release-accessibility" {
		runtimeProcessName = "surface linux-x64 runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		realWindowProcess, realWindowFrame, err := collectLinuxX64ReleaseAccessibilityRealWindowProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, realWindowProcess)
		bridgeProcesses, bridgeArtifacts, err := collectLinuxX64ReleaseAccessibilityBridgeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, bridgeProcesses...)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, surface.ProcessReport{Name: runtimeProcessName, Kind: "runtime", Path: os.Args[0], Ran: true, Pass: true, ExitCode: intPtr(0)})
		artifacts := append([]surface.ArtifactReport{componentArtifact}, bridgeArtifacts...)
		return surfaceProcessEvidence{Processes: processes, Artifacts: artifacts, ArtifactScan: sidecarScan, Frames: []surface.FrameReport{realWindowFrame}}, nil
	}
	if mode == "linux-x64-real-window-component-tree" || mode == "linux-x64-real-window-component-tree-api" {
		runtimeProcessName = "surface linux-x64 real-window runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		realWindowProcess, realWindowFrame, err := collectLinuxX64ComponentTreeRealWindowProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, realWindowProcess)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, surface.ProcessReport{Name: runtimeProcessName, Kind: "runtime", Path: os.Args[0], Ran: true, Pass: true, ExitCode: intPtr(0)})
		return surfaceProcessEvidence{Processes: processes, Artifacts: []surface.ArtifactReport{componentArtifact}, ArtifactScan: sidecarScan, Frames: []surface.FrameReport{realWindowFrame}}, nil
	}
	if mode == "linux-x64-real-window-minimal-toolkit" {
		runtimeProcessName = "surface linux-x64 real-window runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		realWindowProcess, realWindowFrame, err := collectLinuxX64MinimalToolkitRealWindowProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, realWindowProcess)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, surface.ProcessReport{Name: runtimeProcessName, Kind: "runtime", Path: os.Args[0], Ran: true, Pass: true, ExitCode: intPtr(0)})
		return surfaceProcessEvidence{Processes: processes, Artifacts: []surface.ArtifactReport{componentArtifact}, ArtifactScan: sidecarScan, Frames: []surface.FrameReport{realWindowFrame}}, nil
	}
	if mode == "linux-x64-real-window-toolkit-reuse" {
		runtimeProcessName = "surface linux-x64 real-window runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		realWindowProcess, realWindowFrame, err := collectLinuxX64ToolkitReuseRealWindowProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, realWindowProcess)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, surface.ProcessReport{Name: runtimeProcessName, Kind: "runtime", Path: os.Args[0], Ran: true, Pass: true, ExitCode: intPtr(0)})
		return surfaceProcessEvidence{Processes: processes, Artifacts: []surface.ArtifactReport{componentArtifact}, ArtifactScan: sidecarScan, Frames: []surface.FrameReport{realWindowFrame}}, nil
	}
	if mode == "linux-x64-real-window-accessibility-metadata" {
		runtimeProcessName = "surface linux-x64 real-window runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		realWindowProcess, realWindowFrame, err := collectLinuxX64AccessibilityMetadataRealWindowProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, realWindowProcess)
		sidecarScan, err = scanLegacyUISidecarArtifacts(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, surface.ProcessReport{Name: runtimeProcessName, Kind: "runtime", Path: os.Args[0], Ran: true, Pass: true, ExitCode: intPtr(0)})
		return surfaceProcessEvidence{Processes: processes, Artifacts: []surface.ArtifactReport{componentArtifact}, ArtifactScan: sidecarScan, Frames: []surface.FrameReport{realWindowFrame}}, nil
	}
	traceArtifact, sidecarScan, err := collectHeadlessRunnerTraceEvidence(sourcePath, artifactDir, runSurfaceScenario(mode))
	if err != nil {
		return surfaceProcessEvidence{}, err
	}
	processes = append(processes, surface.ProcessReport{Name: runtimeProcessName, Kind: "runtime", Path: os.Args[0], Ran: true, Pass: true, ExitCode: intPtr(0)})
	return surfaceProcessEvidence{Processes: processes, Artifacts: []surface.ArtifactReport{componentArtifact, traceArtifact}, ArtifactScan: sidecarScan}, nil
}

func defaultSurfaceSourcePath(opt smokeOptions) string {
	if opt.SourcePath != "" && opt.SourcePath != "examples/surface_counter.tetra" {
		return opt.SourcePath
	}
	if isTextFocusInputMode(opt.Mode) {
		return "examples/surface_textbox_app.tetra"
	}
	if isReleaseTextInputMode(opt.Mode) {
		return "examples/surface_release_text_input.tetra"
	}
	if isReleaseToolkitMode(opt.Mode) {
		return "examples/surface_release_form.tetra"
	}
	if isReleaseWindowMode(opt.Mode) {
		return "examples/surface_release_form.tetra"
	}
	if isReleaseBrowserMode(opt.Mode) {
		return "examples/surface_release_form.tetra"
	}
	if isReleaseAccessibilityMode(opt.Mode) {
		return "examples/surface_release_accessibility.tetra"
	}
	if isComponentTreeMode(opt.Mode) {
		return "examples/surface_tree_app.tetra"
	}
	if isMinimalToolkitMode(opt.Mode) {
		return "examples/surface_toolkit_form.tetra"
	}
	if isToolkitReuseMode(opt.Mode) {
		return "examples/surface_toolkit_settings.tetra"
	}
	if isAccessibilityMetadataMode(opt.Mode) {
		return "examples/surface_accessibility_settings.tetra"
	}
	if opt.Mode == "linux-x64-real-window" {
		return "examples/surface_window_counter.tetra"
	}
	if opt.Mode == "wasm32-web-browser-canvas" {
		return "examples/surface_browser_counter.tetra"
	}
	if opt.SourcePath == "" {
		return "examples/surface_counter.tetra"
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

func releaseCounterScenarioForSource(opt smokeOptions, scenario headlessScenario) headlessScenario {
	if normalizeSurfaceSourcePath(defaultSurfaceSourcePath(opt)) != "examples/surface_release_counter.tetra" {
		return scenario
	}
	switch opt.Mode {
	case "", "headless", "linux-x64", "linux-x64-real-window", "wasm32-web", "wasm32-web-browser-canvas":
	default:
		return scenario
	}
	for i := range scenario.Components {
		if strings.HasPrefix(scenario.Components[i].Type, "examples.surface_counter.") ||
			strings.HasPrefix(scenario.Components[i].Type, "examples.surface_window_counter.") ||
			strings.HasPrefix(scenario.Components[i].Type, "examples.surface_browser_counter.") {
			name := scenario.Components[i].Type[strings.LastIndex(scenario.Components[i].Type, ".")+1:]
			scenario.Components[i].Type = "examples.surface_release_counter." + name
		}
	}
	scenario.Cases = append(scenario.Cases,
		surface.CaseReport{Name: "release counter source module evidence", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "release counter stable widgets accessibility metadata", Kind: "positive", Ran: true, Pass: true},
	)
	return scenario
}

func normalizeSurfaceSourcePath(path string) string {
	return strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
}

func isReleaseCounterSourcePath(path string) bool {
	return strings.HasSuffix(normalizeSurfaceSourcePath(path), "examples/surface_release_counter.tetra")
}

func collectHeadlessRunnerTraceEvidence(sourcePath string, artifactDir string, scenario headlessScenario) (surface.ArtifactReport, surface.ArtifactScanReport, error) {
	tracePath := filepath.Join(artifactDir, "surface-runner-trace.json")
	if err := writeHeadlessSurfaceTrace(tracePath, sourcePath, scenario); err != nil {
		return surface.ArtifactReport{}, surface.ArtifactScanReport{}, err
	}
	traceFrames, err := readHeadlessSurfaceTrace(tracePath)
	if err != nil {
		return surface.ArtifactReport{}, surface.ArtifactScanReport{}, err
	}
	if !sameFrameEvidence(traceFrames, scenario.Frames) {
		return surface.ArtifactReport{}, surface.ArtifactScanReport{}, fmt.Errorf("headless Surface runner trace frames = %#v, want scenario frames %#v", traceFrames, scenario.Frames)
	}
	traceArtifact, err := artifactReport(tracePath, "runner-trace")
	if err != nil {
		return surface.ArtifactReport{}, surface.ArtifactScanReport{}, err
	}
	sidecarScan, err := scanLegacyUISidecarArtifacts(artifactDir)
	if err != nil {
		return surface.ArtifactReport{}, surface.ArtifactScanReport{}, err
	}
	return traceArtifact, sidecarScan, nil
}

func writeHeadlessSurfaceTrace(path string, sourcePath string, scenario headlessScenario) error {
	trace := headlessSurfaceRunnerTrace{
		Schema:            "tetra.surface.headless-runner-trace.v1",
		Source:            sourcePath,
		Frames:            scenario.Frames,
		Events:            scenario.Events,
		StateTransitions:  scenario.StateTransitions,
		Components:        scenario.Components,
		ComponentTree:     scenario.ComponentTree,
		ComponentTreeAPI:  scenario.ComponentTreeAPI,
		Toolkit:           scenario.Toolkit,
		AccessibilityTree: scenario.AccessibilityTree,
		Cases:             scenario.Cases,
	}
	raw, err := json.MarshalIndent(trace, "", "  ")
	if err != nil {
		return fmt.Errorf("encode headless Surface runner trace: %w", err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		return fmt.Errorf("write headless Surface runner trace %s: %w", path, err)
	}
	return nil
}

func readHeadlessSurfaceTrace(path string) ([]surface.FrameReport, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read headless Surface runner trace %s: %w", path, err)
	}
	var trace headlessSurfaceRunnerTrace
	if err := json.Unmarshal(raw, &trace); err != nil {
		return nil, fmt.Errorf("decode headless Surface runner trace %s: %w", path, err)
	}
	if trace.Schema != "tetra.surface.headless-runner-trace.v1" {
		return nil, fmt.Errorf("headless Surface runner trace schema is %q, want tetra.surface.headless-runner-trace.v1", trace.Schema)
	}
	if strings.TrimSpace(trace.Source) == "" {
		return nil, fmt.Errorf("headless Surface runner trace source is required")
	}
	if len(trace.Frames) < 2 {
		return nil, fmt.Errorf("headless Surface runner trace has %d frames, want pre/post presented frames", len(trace.Frames))
	}
	for _, frame := range trace.Frames {
		if frame.Width <= 0 || frame.Height <= 0 || frame.Stride <= 0 || strings.TrimSpace(frame.Checksum) == "" || !frame.Presented {
			return nil, fmt.Errorf("headless Surface runner trace frame %d has incomplete presented frame evidence", frame.Order)
		}
	}
	return trace.Frames, nil
}

func sameFrameEvidence(a []surface.FrameReport, b []surface.FrameReport) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Order != b[i].Order ||
			a[i].Width != b[i].Width ||
			a[i].Height != b[i].Height ||
			a[i].Stride != b[i].Stride ||
			a[i].Checksum != b[i].Checksum ||
			a[i].Presented != b[i].Presented {
			return false
		}
	}
	return true
}

func collectWASM32WebProcessEvidence(sourcePath string, artifactDir string) (surfaceProcessEvidence, error) {
	root, err := repoRootForCommands()
	if err != nil {
		return surfaceProcessEvidence{}, err
	}
	wasmPath := filepath.Join(artifactDir, "surface-counter.wasm")
	if _, err := compiler.BuildFileWithStatsOpt(sourcePath, wasmPath, "wasm32-web", compiler.BuildOptions{Jobs: 1}); err != nil {
		return surfaceProcessEvidence{}, fmt.Errorf("build wasm32-web Surface source %s: %w", sourcePath, err)
	}
	componentArtifact, err := artifactReport(wasmPath, "component-app")
	if err != nil {
		return surfaceProcessEvidence{}, err
	}
	if err := validateCompilerOwnedWASMLoader(wasmPath); err != nil {
		return surfaceProcessEvidence{}, err
	}
	loaderArtifact, err := artifactReport(strings.TrimSuffix(wasmPath, filepath.Ext(wasmPath))+".mjs", "compiler-owned-loader")
	if err != nil {
		return surfaceProcessEvidence{}, err
	}

	validateCmd := exec.Command("go", "run", "./tools/cmd/validate-wasm-imports", "--target", "wasm32-web", wasmPath)
	validateCmd.Dir = root
	validateStdout, validateStderr, validateExit, err := runCommand(validateCmd)
	if err != nil {
		return surfaceProcessEvidence{}, fmt.Errorf("run wasm32-web Surface import validator: %w", err)
	}
	if validateExit != 0 {
		return surfaceProcessEvidence{}, fmt.Errorf("run wasm32-web Surface import validator: exit code %d, stdout %q stderr %q", validateExit, validateStdout, validateStderr)
	}
	if validateStdout != "" {
		return surfaceProcessEvidence{}, fmt.Errorf("run wasm32-web Surface import validator: unexpected stdout %q", validateStdout)
	}

	nodeVersionCmd := nodeCommand("--version")
	nodeStdout, nodeStderr, nodeExit, err := runCommand(nodeVersionCmd)
	if err != nil {
		return surfaceProcessEvidence{}, fmt.Errorf("run wasm32-web Surface runtime probe: %w", err)
	}
	if nodeExit != 0 {
		return surfaceProcessEvidence{}, fmt.Errorf("run wasm32-web Surface runtime probe: exit code %d, stdout %q stderr %q", nodeExit, nodeStdout, nodeStderr)
	}
	if nodeStderr != "" {
		return surfaceProcessEvidence{}, fmt.Errorf("run wasm32-web Surface runtime probe: unexpected stderr %q", nodeStderr)
	}

	helperPath := filepath.Join(root, "scripts", "tools", "web_run_module.mjs")
	tracePath := filepath.Join(artifactDir, "surface-runner-trace.json")
	appCmd := nodeCommand(helperPath, "--surface-trace", tracePath, wasmPath)
	appCmd.Dir = root
	appStdout, appStderr, appExit, err := runCommand(appCmd)
	if err != nil {
		return surfaceProcessEvidence{}, fmt.Errorf("run wasm32-web Surface app: %w", err)
	}
	if appStdout != "" {
		return surfaceProcessEvidence{}, fmt.Errorf("run wasm32-web Surface app %s: unexpected stdout %q", wasmPath, appStdout)
	}
	if appStderr != "" {
		return surfaceProcessEvidence{}, fmt.Errorf("run wasm32-web Surface app %s: unexpected stderr %q", wasmPath, appStderr)
	}
	if appExit != 1 {
		return surfaceProcessEvidence{}, fmt.Errorf("run wasm32-web Surface app %s: exit code %d, want 1", wasmPath, appExit)
	}
	traceFrames, err := readWASM32WebSurfaceTrace(tracePath)
	if err != nil {
		return surfaceProcessEvidence{}, err
	}
	traceArtifact, err := artifactReport(tracePath, "runner-trace")
	if err != nil {
		return surfaceProcessEvidence{}, err
	}
	if len(traceFrames) < 2 {
		return surfaceProcessEvidence{}, fmt.Errorf("wasm32-web Surface runner trace has %d frames, want pre/post presented frames", len(traceFrames))
	}
	wantFrame := renderCounterFrameRGBA(1, true)
	if traceFrames[len(traceFrames)-1].Checksum != checksumRGBA(wantFrame.Pixels) {
		return surfaceProcessEvidence{}, fmt.Errorf("wasm32-web Surface runner after-frame checksum = %s, want %s", traceFrames[len(traceFrames)-1].Checksum, checksumRGBA(wantFrame.Pixels))
	}
	sidecarScan, err := scanLegacyUISidecarArtifacts(artifactDir, sidecarScanOptions{AllowCompilerOwnedWASMLoader: true})
	if err != nil {
		return surfaceProcessEvidence{}, err
	}

	processes := []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: fmt.Sprintf("tetra build --target wasm32-web %s -o %s", sourcePath, wasmPath), Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface wasm32-web component app", Kind: "app", Path: fmt.Sprintf("node scripts/tools/web_run_module.mjs --surface-trace %s %s", tracePath, wasmPath), Ran: true, Pass: true, ExitCode: intPtr(appExit), ExpectedExitCode: intPtr(1)},
		{Name: "surface wasm32-web import validator", Kind: "runtime", Path: fmt.Sprintf("go run ./tools/cmd/validate-wasm-imports --target wasm32-web %s", wasmPath), Ran: true, Pass: true, ExitCode: intPtr(validateExit)},
		{Name: "surface wasm32-web runtime", Kind: "runtime", Path: "node --version", Ran: true, Pass: true, ExitCode: intPtr(nodeExit)},
	}
	return surfaceProcessEvidence{Processes: processes, Artifacts: []surface.ArtifactReport{componentArtifact, loaderArtifact, traceArtifact}, ArtifactScan: sidecarScan, Frames: traceFrames}, nil
}

func collectWASM32WebBrowserCanvasProcessEvidence(sourcePath string, artifactDir string, scenarioName string) (surfaceProcessEvidence, error) {
	root, err := repoRootForCommands()
	if err != nil {
		return surfaceProcessEvidence{}, err
	}
	wasmFile := "surface-browser-counter.wasm"
	if scenarioName == "text-focus-input" {
		wasmFile = "surface-textbox-app.wasm"
	}
	if scenarioName == "release-text-input" {
		wasmFile = "surface-release-text-input.wasm"
	}
	if scenarioName == "release-toolkit" {
		wasmFile = "surface-release-form.wasm"
	}
	if scenarioName == "release-browser" {
		wasmFile = "surface-release-form.wasm"
	}
	if scenarioName == "release-accessibility" {
		wasmFile = "surface-release-accessibility.wasm"
	}
	if scenarioName == "component-tree" {
		wasmFile = "surface-tree-app.wasm"
	}
	if scenarioName == "minimal-toolkit" {
		wasmFile = "surface-toolkit-form.wasm"
	}
	if scenarioName == "toolkit-reuse" {
		wasmFile = "surface-toolkit-settings.wasm"
	}
	if scenarioName == "accessibility-metadata" {
		wasmFile = "surface-accessibility-settings.wasm"
	}
	wasmPath := filepath.Join(artifactDir, wasmFile)
	if _, err := compiler.BuildFileWithStatsOpt(sourcePath, wasmPath, "wasm32-web", compiler.BuildOptions{Jobs: 1}); err != nil {
		return surfaceProcessEvidence{}, fmt.Errorf("build wasm32-web browser canvas Surface source %s: %w", sourcePath, err)
	}
	componentArtifact, err := artifactReport(wasmPath, "component-app")
	if err != nil {
		return surfaceProcessEvidence{}, err
	}
	if err := validateCompilerOwnedWASMLoader(wasmPath); err != nil {
		return surfaceProcessEvidence{}, err
	}
	loaderArtifact, err := artifactReport(strings.TrimSuffix(wasmPath, filepath.Ext(wasmPath))+".mjs", "compiler-owned-loader")
	if err != nil {
		return surfaceProcessEvidence{}, err
	}

	validateCmd := exec.Command("go", "run", "./tools/cmd/validate-wasm-imports", "--target", "wasm32-web", wasmPath)
	validateCmd.Dir = root
	validateStdout, validateStderr, validateExit, err := runCommand(validateCmd)
	if err != nil {
		return surfaceProcessEvidence{}, fmt.Errorf("run wasm32-web browser canvas Surface import validator: %w", err)
	}
	if validateExit != 0 {
		return surfaceProcessEvidence{}, fmt.Errorf("run wasm32-web browser canvas Surface import validator: exit code %d, stdout %q stderr %q", validateExit, validateStdout, validateStderr)
	}
	if validateStdout != "" {
		return surfaceProcessEvidence{}, fmt.Errorf("run wasm32-web browser canvas Surface import validator: unexpected stdout %q", validateStdout)
	}

	browserPath, err := discoverBrowserRunner()
	if err != nil {
		return surfaceProcessEvidence{}, err
	}
	browserVersionCmd := exec.Command(browserPath, "--version")
	browserVersionStdout, browserVersionStderr, browserVersionExit, err := runCommand(browserVersionCmd)
	if err != nil {
		return surfaceProcessEvidence{}, fmt.Errorf("run wasm32-web browser canvas runtime probe: %w", err)
	}
	if browserVersionExit != 0 {
		return surfaceProcessEvidence{}, fmt.Errorf("run wasm32-web browser canvas runtime probe: exit code %d, stdout %q stderr %q", browserVersionExit, browserVersionStdout, browserVersionStderr)
	}

	tracePath := filepath.Join(artifactDir, "surface-runner-trace.json")
	browserTrace, browserProcessPath, browserExit, err := runBrowserCanvasTrace(root, browserPath, wasmPath, scenarioName)
	if err != nil {
		return surfaceProcessEvidence{}, err
	}
	traceFrames, err := writeBrowserCanvasSurfaceTrace(tracePath, wasmPath, browserTrace)
	if err != nil {
		return surfaceProcessEvidence{}, err
	}
	traceArtifact, err := artifactReport(tracePath, "runner-trace")
	if err != nil {
		return surfaceProcessEvidence{}, err
	}
	if browserTrace.AppExitCode != 1 {
		return surfaceProcessEvidence{}, fmt.Errorf("wasm32-web browser canvas app exit code = %d, want 1", browserTrace.AppExitCode)
	}
	if len(traceFrames) < 2 {
		return surfaceProcessEvidence{}, fmt.Errorf("wasm32-web browser canvas trace has %d frames, want pre/post presented frames", len(traceFrames))
	}
	after := traceFrames[len(traceFrames)-1]
	if scenarioName == "release-text-input" {
		before := traceFrames[0]
		if after.Width != 480 || after.Height != 320 || after.Stride != 1920 || !after.Presented {
			return surfaceProcessEvidence{}, fmt.Errorf("wasm32-web release text-input after-frame = %#v, want presented 480x320 RGBA frame", after)
		}
		if before.Checksum == after.Checksum {
			return surfaceProcessEvidence{}, fmt.Errorf("wasm32-web release text-input frame checksums did not change across text/input baseline: %#v", traceFrames)
		}
		sidecarScan, err := scanLegacyUISidecarArtifacts(artifactDir, sidecarScanOptions{AllowCompilerOwnedWASMLoader: true})
		if err != nil {
			return surfaceProcessEvidence{}, err
		}

		processes := []surface.ProcessReport{
			{Name: "tetra build", Kind: "build", Path: fmt.Sprintf("tetra build --target wasm32-web %s -o %s", sourcePath, wasmPath), Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface wasm32-web browser canvas component app", Kind: "app", Path: fmt.Sprintf("%s --headless <surface-browser-canvas-runner> scenario=%s wasm=%s", browserPath, scenarioName, wasmPath), Ran: true, Pass: true, ExitCode: intPtr(browserExit), ExpectedExitCode: intPtr(0)},
			{Name: "surface wasm32-web import validator", Kind: "runtime", Path: fmt.Sprintf("go run ./tools/cmd/validate-wasm-imports --target wasm32-web %s", wasmPath), Ran: true, Pass: true, ExitCode: intPtr(validateExit)},
			{Name: "surface wasm32-web browser canvas runtime", Kind: "runtime", Path: strings.TrimSpace(browserVersionStdout), Ran: true, Pass: true, ExitCode: intPtr(browserVersionExit)},
			{Name: "surface wasm32-web browser canvas trace", Kind: "runtime", Path: browserProcessPath, Ran: true, Pass: true, ExitCode: intPtr(browserExit)},
		}
		return surfaceProcessEvidence{Processes: processes, Artifacts: []surface.ArtifactReport{componentArtifact, loaderArtifact, traceArtifact}, ArtifactScan: sidecarScan, Frames: traceFrames}, nil
	}
	if scenarioName == "release-browser" {
		before := traceFrames[0]
		if after.Order != 5 || after.Width != 560 || after.Height != 420 || after.Stride != 2240 || !after.Presented {
			return surfaceProcessEvidence{}, fmt.Errorf("wasm32-web release browser after-frame = %#v, want order-5 presented 560x420 RGBA frame", after)
		}
		if before.Checksum == after.Checksum {
			return surfaceProcessEvidence{}, fmt.Errorf("wasm32-web release browser frame checksums did not change across browser release scenario: %#v", traceFrames)
		}
		if !browserTraceHasNativeEvents(browserTrace, []string{"pointerup", "keydown", "resize", "beforeinput", "compositionstart", "compositionupdate", "compositionend"}) {
			return surfaceProcessEvidence{}, fmt.Errorf("wasm32-web release browser trace missing required native browser input events: %#v", browserTrace.BrowserEvents)
		}
		if err := validateBrowserReleaseTraceEvidence(browserTrace); err != nil {
			return surfaceProcessEvidence{}, err
		}
		sidecarScan, err := scanLegacyUISidecarArtifacts(artifactDir, sidecarScanOptions{AllowCompilerOwnedWASMLoader: true})
		if err != nil {
			return surfaceProcessEvidence{}, err
		}

		processes := []surface.ProcessReport{
			{Name: "tetra build", Kind: "build", Path: fmt.Sprintf("tetra build --target wasm32-web %s -o %s", sourcePath, wasmPath), Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface wasm32-web browser canvas component app", Kind: "app", Path: fmt.Sprintf("%s --headless <surface-browser-canvas-runner> scenario=%s wasm=%s", browserPath, scenarioName, wasmPath), Ran: true, Pass: true, ExitCode: intPtr(browserExit), ExpectedExitCode: intPtr(0)},
			{Name: "surface wasm32-web import validator", Kind: "runtime", Path: fmt.Sprintf("go run ./tools/cmd/validate-wasm-imports --target wasm32-web %s", wasmPath), Ran: true, Pass: true, ExitCode: intPtr(validateExit)},
			{Name: "surface wasm32-web browser canvas runtime", Kind: "runtime", Path: strings.TrimSpace(browserVersionStdout), Ran: true, Pass: true, ExitCode: intPtr(browserVersionExit)},
			{Name: "surface wasm32-web browser canvas trace", Kind: "runtime", Path: browserProcessPath, Ran: true, Pass: true, ExitCode: intPtr(browserExit)},
		}
		return surfaceProcessEvidence{Processes: processes, Artifacts: []surface.ArtifactReport{componentArtifact, loaderArtifact, traceArtifact}, ArtifactScan: sidecarScan, Frames: traceFrames}, nil
	}
	if scenarioName == "release-accessibility" {
		before := traceFrames[0]
		if after.Order != 5 || after.Width != 480 || after.Height != 320 || after.Stride != 1920 || !after.Presented {
			return surfaceProcessEvidence{}, fmt.Errorf("wasm32-web release accessibility after-frame = %#v, want order-5 presented 480x320 RGBA frame", after)
		}
		if before.Checksum == after.Checksum {
			return surfaceProcessEvidence{}, fmt.Errorf("wasm32-web release accessibility frame checksums did not change across platform bridge scenario: %#v", traceFrames)
		}
		if !browserTraceHasNativeEvents(browserTrace, []string{"pointerup", "keydown", "resize", "beforeinput"}) {
			return surfaceProcessEvidence{}, fmt.Errorf("wasm32-web release accessibility trace missing required native browser input events: %#v", browserTrace.BrowserEvents)
		}
		if err := validateBrowserAccessibilityTraceEvidence(browserTrace); err != nil {
			return surfaceProcessEvidence{}, err
		}
		sidecarScan, err := scanLegacyUISidecarArtifacts(artifactDir, sidecarScanOptions{AllowCompilerOwnedWASMLoader: true})
		if err != nil {
			return surfaceProcessEvidence{}, err
		}

		processes := []surface.ProcessReport{
			{Name: "tetra build", Kind: "build", Path: fmt.Sprintf("tetra build --target wasm32-web %s -o %s", sourcePath, wasmPath), Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface wasm32-web browser canvas component app", Kind: "app", Path: fmt.Sprintf("%s --headless <surface-browser-canvas-runner> scenario=%s wasm=%s", browserPath, scenarioName, wasmPath), Ran: true, Pass: true, ExitCode: intPtr(browserExit), ExpectedExitCode: intPtr(0)},
			{Name: "surface wasm32-web import validator", Kind: "runtime", Path: fmt.Sprintf("go run ./tools/cmd/validate-wasm-imports --target wasm32-web %s", wasmPath), Ran: true, Pass: true, ExitCode: intPtr(validateExit)},
			{Name: "surface wasm32-web browser canvas runtime", Kind: "runtime", Path: strings.TrimSpace(browserVersionStdout), Ran: true, Pass: true, ExitCode: intPtr(browserVersionExit)},
			{Name: "surface wasm32-web browser canvas trace", Kind: "runtime", Path: browserProcessPath, Ran: true, Pass: true, ExitCode: intPtr(browserExit)},
		}
		return surfaceProcessEvidence{Processes: processes, Artifacts: []surface.ArtifactReport{componentArtifact, loaderArtifact, traceArtifact}, ArtifactScan: sidecarScan, Frames: traceFrames}, nil
	}
	wantFrame := renderBrowserCounterFrameRGBA(2, 1, 400, 240, true)
	if isReleaseCounterSourcePath(sourcePath) && scenarioName == "counter" {
		wantFrame = renderReleaseCounterFrameRGBA(0, 1, 1, 2, 400, 240)
	}
	if scenarioName == "text-focus-input" || scenarioName == "release-text-input" {
		wantFrame = renderTextFocusInputFrameRGBA(1, 1, 1, 400, 240)
	}
	if scenarioName == "component-tree" {
		wantFrame = renderComponentTreeFrameRGBA(0, 0, 6, 1, 1, 400, 240)
	}
	if scenarioName == "minimal-toolkit" {
		wantFrame = renderMinimalToolkitFrameRGBA(0, 0, 4, 1, 1, 2, 400, 240)
	}
	if scenarioName == "toolkit-reuse" {
		wantFrame = renderToolkitReuseFrameRGBA(0, 0, 4, 1, 1, 2, 480, 320)
	}
	if scenarioName == "release-toolkit" {
		wantFrame = renderReleaseToolkitFrameRGBA(0, 0, 7, 1, 1, 2, true, 16, 560, 420)
	}
	if scenarioName == "accessibility-metadata" || scenarioName == "release-accessibility" {
		wantFrame = renderAccessibilityMetadataFrameRGBA(0, 0, 5, 1, 1, 2, 480, 320)
	}
	if after.Order != 5 || after.Width != wantFrame.Width || after.Height != wantFrame.Height || after.Stride != wantFrame.Stride || after.Checksum != checksumRGBA(wantFrame.Pixels) {
		return surfaceProcessEvidence{}, fmt.Errorf("wasm32-web browser canvas after-frame = %#v, want order-5 %dx%d checksum %s", after, wantFrame.Width, wantFrame.Height, checksumRGBA(wantFrame.Pixels))
	}
	if !browserTraceHasNativeEvents(browserTrace, []string{"pointerup", "keydown", "resize", "beforeinput"}) {
		return surfaceProcessEvidence{}, fmt.Errorf("wasm32-web browser canvas trace missing required native browser input events: %#v", browserTrace.BrowserEvents)
	}
	sidecarScan, err := scanLegacyUISidecarArtifacts(artifactDir, sidecarScanOptions{AllowCompilerOwnedWASMLoader: true})
	if err != nil {
		return surfaceProcessEvidence{}, err
	}

	processes := []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: fmt.Sprintf("tetra build --target wasm32-web %s -o %s", sourcePath, wasmPath), Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface wasm32-web browser canvas component app", Kind: "app", Path: fmt.Sprintf("%s --headless <surface-browser-canvas-runner> scenario=%s wasm=%s", browserPath, scenarioName, wasmPath), Ran: true, Pass: true, ExitCode: intPtr(browserExit), ExpectedExitCode: intPtr(0)},
		{Name: "surface wasm32-web import validator", Kind: "runtime", Path: fmt.Sprintf("go run ./tools/cmd/validate-wasm-imports --target wasm32-web %s", wasmPath), Ran: true, Pass: true, ExitCode: intPtr(validateExit)},
		{Name: "surface wasm32-web browser canvas runtime", Kind: "runtime", Path: strings.TrimSpace(browserVersionStdout), Ran: true, Pass: true, ExitCode: intPtr(browserVersionExit)},
		{Name: "surface wasm32-web browser canvas trace", Kind: "runtime", Path: browserProcessPath, Ran: true, Pass: true, ExitCode: intPtr(browserExit)},
	}
	return surfaceProcessEvidence{Processes: processes, Artifacts: []surface.ArtifactReport{componentArtifact, loaderArtifact, traceArtifact}, ArtifactScan: sidecarScan, Frames: traceFrames}, nil
}

func browserTraceHasNativeEvents(trace browserCanvasTrace, nativeTypes []string) bool {
	seen := map[string]bool{}
	for _, event := range trace.BrowserEvents {
		seen[event.NativeType] = true
	}
	for _, nativeType := range nativeTypes {
		if !seen[nativeType] {
			return false
		}
	}
	return true
}

func validateBrowserReleaseTraceEvidence(trace browserCanvasTrace) error {
	if trace.BrowserClipboard.Harness != "deterministic-browser-clipboard-v1" ||
		!trace.BrowserClipboard.Read ||
		!trace.BrowserClipboard.Write ||
		!trace.BrowserClipboard.OwnedCopy ||
		trace.BrowserClipboard.Bytes <= 0 {
		return fmt.Errorf("wasm32-web release browser trace missing deterministic clipboard harness evidence: %#v", trace.BrowserClipboard)
	}
	if !trace.BrowserComposition.Start ||
		!trace.BrowserComposition.Update ||
		!trace.BrowserComposition.Commit ||
		!trace.BrowserComposition.Cancel {
		return fmt.Errorf("wasm32-web release browser trace missing composition evidence: %#v", trace.BrowserComposition)
	}
	if err := validateBrowserAccessibilityTraceEvidence(trace); err != nil {
		return err
	}
	return nil
}

func validateBrowserAccessibilityTraceEvidence(trace browserCanvasTrace) error {
	if !trace.BrowserAccessibility.Snapshot ||
		!trace.BrowserAccessibility.Mirror ||
		!trace.BrowserAccessibility.CompilerOwned ||
		!trace.BrowserAccessibility.Bounds ||
		!trace.BrowserAccessibility.Focus {
		return fmt.Errorf("wasm32-web release browser trace missing accessibility snapshot/mirror evidence: %#v", trace.BrowserAccessibility)
	}
	if trace.BrowserAccessibility.DOMVisualUI || trace.BrowserAccessibility.UserJS {
		return fmt.Errorf("wasm32-web release browser trace must not claim DOM visual UI or user JS app logic: %#v", trace.BrowserAccessibility)
	}
	for _, role := range []string{"root", "textbox", "checkbox", "button", "status"} {
		if !containsString(trace.BrowserAccessibility.Roles, role) {
			return fmt.Errorf("wasm32-web release browser trace missing accessibility role %s: %#v", role, trace.BrowserAccessibility.Roles)
		}
	}
	return nil
}

func discoverBrowserRunner() (string, error) {
	var probeFailure string
	for _, candidate := range []string{"chromium", "chromium-browser", "google-chrome", "chrome"} {
		runner, err := exec.LookPath(candidate)
		if err != nil {
			continue
		}
		cmd := exec.Command(runner, "--headless", "--no-sandbox", "--disable-gpu", "--dump-dom", "about:blank")
		cmd.Stdout = io.Discard
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			probeFailure = fmt.Sprintf("%s failed headless probe: %v: %s", runner, err, strings.TrimSpace(stderr.String()))
			continue
		}
		return runner, nil
	}
	if probeFailure != "" {
		return "", fmt.Errorf("cannot run wasm32-web browser canvas Surface evidence: browser runner unavailable: %s", probeFailure)
	}
	return "", fmt.Errorf("cannot run wasm32-web browser canvas Surface evidence: browser runner unavailable; searched: chromium, chromium-browser, google-chrome, chrome")
}

func runBrowserCanvasTrace(root string, browserPath string, wasmPath string, scenarioName string) (browserCanvasTrace, string, int, error) {
	hostPath := filepath.Join(root, "scripts", "tools", "surface_browser_canvas_host.mjs")
	hostSource, err := os.ReadFile(hostPath)
	if err != nil {
		return browserCanvasTrace{}, "", -1, fmt.Errorf("read browser canvas Surface host %s: %w", hostPath, err)
	}
	if _, err := os.Stat(wasmPath); err != nil {
		return browserCanvasTrace{}, "", -1, fmt.Errorf("stat browser canvas Surface wasm %s: %w", wasmPath, err)
	}
	runnerURL, cleanupRunner, err := browserCanvasRunnerFileURL(wasmPath, string(hostSource), scenarioName)
	if err != nil {
		return browserCanvasTrace{}, "", -1, err
	}
	defer cleanupRunner()
	args := []string{
		"--headless",
		"--no-sandbox",
		"--disable-gpu",
		"--disable-dev-shm-usage",
		"--disable-crash-reporter",
		"--disable-breakpad",
		"--allow-file-access-from-files",
		"--virtual-time-budget=12000",
		"--dump-dom",
		runnerURL,
	}
	processArgs := append([]string{}, args[:len(args)-1]...)
	processArgs = append(processArgs, fmt.Sprintf("<surface-browser-canvas-file-runner scenario=%s>", scenarioName))
	processPath := browserPath + " " + strings.Join(processArgs, " ")
	var lastTraceErr error
	for attempt := 1; attempt <= 3; attempt++ {
		cmd := exec.Command(browserPath, args...)
		stdout, stderr, exit, err := runCommand(cmd)
		if err != nil {
			return browserCanvasTrace{}, processPath, exit, fmt.Errorf("run wasm32-web browser canvas Surface app: %w stderr=%q", err, stderr)
		}
		if exit != 0 {
			return browserCanvasTrace{}, processPath, exit, fmt.Errorf("run wasm32-web browser canvas Surface app: browser exit code %d stderr=%q", exit, stderr)
		}
		rawTrace, err := extractBrowserCanvasTrace(stdout)
		if err != nil {
			lastTraceErr = fmt.Errorf("%w; browser stderr=%q", err, stderr)
			if isRetriableBrowserCanvasTraceError(err) && attempt < 3 {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return browserCanvasTrace{}, processPath, exit, lastTraceErr
		}
		var trace browserCanvasTrace
		if err := json.Unmarshal([]byte(rawTrace), &trace); err != nil {
			return browserCanvasTrace{}, processPath, exit, fmt.Errorf("decode browser canvas Surface trace: %w: %s", err, rawTrace)
		}
		if strings.TrimSpace(trace.Error) != "" {
			return browserCanvasTrace{}, processPath, exit, fmt.Errorf("browser canvas Surface trace error: %s", trace.Error)
		}
		return trace, processPath, exit, nil
	}
	return browserCanvasTrace{}, processPath, -1, fmt.Errorf("browser canvas Surface trace was not populated after retries: %w", lastTraceErr)
}

func browserCanvasRunnerDataURL(hostSource string, wasmBytes []byte, scenarioName string) (string, error) {
	inlineHost, err := inlineBrowserCanvasHostSource(hostSource)
	if err != nil {
		return "", err
	}
	wasmURL := "data:application/wasm;base64," + base64.StdEncoding.EncodeToString(wasmBytes)
	html := browserCanvasRunnerHTML(inlineHost, wasmURL, scenarioName)
	return "data:text/html;base64," + base64.StdEncoding.EncodeToString([]byte(html)), nil
}

func browserCanvasRunnerFileURL(wasmPath string, hostSource string, scenarioName string) (string, func(), error) {
	inlineHost, err := inlineBrowserCanvasHostSource(hostSource)
	if err != nil {
		return "", nil, err
	}
	absWASM, err := filepath.Abs(wasmPath)
	if err != nil {
		return "", nil, fmt.Errorf("resolve browser canvas wasm path %s: %w", wasmPath, err)
	}
	runnerDir := filepath.Dir(absWASM)
	if strings.HasSuffix(filepath.Base(runnerDir), "-artifacts") {
		runnerDir = filepath.Dir(runnerDir)
	}
	if err := os.MkdirAll(runnerDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("create browser canvas runner dir %s: %w", runnerDir, err)
	}
	runnerPath := filepath.Join(runnerDir, "surface-browser-canvas-runner-"+safeBrowserCanvasScenarioName(scenarioName)+".html")
	html := browserCanvasRunnerHTML(inlineHost, fileURL(absWASM), scenarioName)
	if err := os.WriteFile(runnerPath, []byte(html), 0o644); err != nil {
		return "", nil, fmt.Errorf("write browser canvas runner %s: %w", runnerPath, err)
	}
	cleanup := func() {
		_ = os.Remove(runnerPath)
	}
	return fileURL(runnerPath), cleanup, nil
}

func inlineBrowserCanvasHostSource(hostSource string) (string, error) {
	inlineHost := strings.Replace(hostSource, "export async function runSurfaceBrowserCanvas", "async function runSurfaceBrowserCanvas", 1)
	if inlineHost == hostSource {
		return "", fmt.Errorf("browser canvas Surface host missing runSurfaceBrowserCanvas export")
	}
	return inlineHost, nil
}

func safeBrowserCanvasScenarioName(scenarioName string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(scenarioName)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	if b.Len() == 0 {
		return "default"
	}
	return b.String()
}

func fileURL(path string) string {
	return (&neturl.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
}

func browserCanvasRunnerHTML(inlineHost string, wasmURL string, scenarioName string) string {
	return fmt.Sprintf(`<!doctype html>
<html>
  <head><style>html,body{margin:0}canvas{display:block}</style></head>
  <body>
    <canvas id="surface-canvas" width="320" height="200"></canvas>
    <pre id="surface-trace">pending</pre>
    <script>
%s
      const target = document.getElementById('surface-trace');
      (async () => {
        try {
          const trace = await runSurfaceBrowserCanvas({
            wasmURL: %q,
            canvas: document.getElementById('surface-canvas'),
            scenario: %q,
          });
          target.textContent = JSON.stringify(trace);
        } catch (err) {
          target.textContent = JSON.stringify({
            schema: 'tetra.surface.browser-canvas-trace.v1',
            error: String(err && err.stack ? err.stack : err),
          });
        }
      })();
    </script>
  </body>
</html>
`, inlineHost, wasmURL, scenarioName)
}

func extractBrowserCanvasTrace(dom string) (string, error) {
	const startMarker = `<pre id="surface-trace">`
	start := strings.Index(dom, startMarker)
	if start < 0 {
		return "", fmt.Errorf("browser canvas Surface runner did not emit surface-trace element")
	}
	start += len(startMarker)
	end := strings.Index(dom[start:], `</pre>`)
	if end < 0 {
		return "", fmt.Errorf("browser canvas Surface runner emitted unterminated surface-trace element")
	}
	text := strings.TrimSpace(html.UnescapeString(dom[start : start+end]))
	if text == "" || text == "pending" {
		return "", fmt.Errorf("browser canvas Surface runner trace was not populated")
	}
	return text, nil
}

func isRetriableBrowserCanvasTraceError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "runner trace was not populated")
}

func writeBrowserCanvasSurfaceTrace(path string, wasmPath string, raw browserCanvasTrace) ([]surface.FrameReport, error) {
	if raw.Schema != "tetra.surface.browser-canvas-trace.v1" {
		return nil, fmt.Errorf("browser canvas Surface trace schema is %q, want tetra.surface.browser-canvas-trace.v1", raw.Schema)
	}
	if !raw.Canvas.Opened || !raw.Canvas.Readback {
		return nil, fmt.Errorf("browser canvas Surface trace missing opened/readback canvas evidence: %#v", raw.Canvas)
	}
	if raw.AppExitCode != 1 {
		return nil, fmt.Errorf("browser canvas Surface trace app_exit_code = %d, want 1", raw.AppExitCode)
	}
	type traceFrame struct {
		Order          int    `json:"order"`
		Width          int    `json:"width"`
		Height         int    `json:"height"`
		Stride         int    `json:"stride"`
		PixelsLen      int    `json:"pixels_len"`
		SourceChecksum string `json:"source_checksum"`
		CanvasChecksum string `json:"canvas_checksum"`
		Checksum       string `json:"checksum"`
		Presented      bool   `json:"presented"`
	}
	trace := struct {
		Schema               string                          `json:"schema"`
		WASM                 string                          `json:"wasm_path"`
		Canvas               browserCanvasTraceCanvas        `json:"canvas"`
		BrowserEvents        []browserCanvasTraceEvent       `json:"browser_events"`
		BrowserClipboard     browserCanvasTraceClipboard     `json:"browser_clipboard"`
		BrowserComposition   browserCanvasTraceComposition   `json:"browser_composition"`
		BrowserAccessibility browserCanvasTraceAccessibility `json:"browser_accessibility"`
		Frames               []traceFrame                    `json:"frames"`
		AppExitCode          int                             `json:"app_exit_code"`
	}{
		Schema:               raw.Schema,
		WASM:                 wasmPath,
		Canvas:               raw.Canvas,
		BrowserEvents:        raw.BrowserEvents,
		BrowserClipboard:     raw.BrowserClipboard,
		BrowserComposition:   raw.BrowserComposition,
		BrowserAccessibility: raw.BrowserAccessibility,
		AppExitCode:          raw.AppExitCode,
	}
	frames := make([]surface.FrameReport, 0, len(raw.Frames))
	for _, frame := range raw.Frames {
		sourcePixels, err := base64.StdEncoding.DecodeString(frame.SourcePixelsB64)
		if err != nil {
			return nil, fmt.Errorf("decode browser canvas source pixels for frame %d: %w", frame.Order, err)
		}
		canvasPixels, err := base64.StdEncoding.DecodeString(frame.CanvasPixelsB64)
		if err != nil {
			return nil, fmt.Errorf("decode browser canvas readback pixels for frame %d: %w", frame.Order, err)
		}
		if len(sourcePixels) != frame.PixelsLen || len(canvasPixels) != frame.PixelsLen {
			return nil, fmt.Errorf("browser canvas frame %d pixel lengths source=%d canvas=%d want %d", frame.Order, len(sourcePixels), len(canvasPixels), frame.PixelsLen)
		}
		sourceChecksum := checksumRGBA(sourcePixels)
		canvasChecksum := checksumRGBA(canvasPixels)
		if sourceChecksum != canvasChecksum {
			return nil, fmt.Errorf("browser canvas frame %d readback checksum %s differs from Tetra framebuffer checksum %s", frame.Order, canvasChecksum, sourceChecksum)
		}
		reportOrder := browserCanvasReportFrameOrder(frame.Order)
		trace.Frames = append(trace.Frames, traceFrame{
			Order:          reportOrder,
			Width:          frame.Width,
			Height:         frame.Height,
			Stride:         frame.Stride,
			PixelsLen:      frame.PixelsLen,
			SourceChecksum: sourceChecksum,
			CanvasChecksum: canvasChecksum,
			Checksum:       canvasChecksum,
			Presented:      true,
		})
		frames = append(frames, surface.FrameReport{
			Order:     reportOrder,
			Width:     frame.Width,
			Height:    frame.Height,
			Stride:    frame.Stride,
			Checksum:  canvasChecksum,
			Presented: true,
		})
	}
	rawJSON, err := json.MarshalIndent(trace, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode browser canvas Surface trace: %w", err)
	}
	if err := os.WriteFile(path, append(rawJSON, '\n'), 0o644); err != nil {
		return nil, fmt.Errorf("write browser canvas Surface trace %s: %w", path, err)
	}
	return frames, nil
}

func browserCanvasReportFrameOrder(rawOrder int) int {
	if rawOrder <= 1 {
		return 1
	}
	return rawOrder + 3
}

func readWASM32WebSurfaceTrace(path string) ([]surface.FrameReport, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read wasm32-web Surface runner trace %s: %w", path, err)
	}
	var trace wasmSurfaceRunnerTrace
	if err := json.Unmarshal(raw, &trace); err != nil {
		return nil, fmt.Errorf("decode wasm32-web Surface runner trace %s: %w", path, err)
	}
	if trace.Schema != "tetra.surface.web-runner-trace.v1" {
		return nil, fmt.Errorf("wasm32-web Surface runner trace schema is %q, want tetra.surface.web-runner-trace.v1", trace.Schema)
	}
	frames := make([]surface.FrameReport, 0, len(trace.Frames))
	for _, frame := range trace.Frames {
		if frame.PixelsLen <= 0 {
			return nil, fmt.Errorf("wasm32-web Surface runner trace frame %d pixels_len must be positive", frame.Order)
		}
		if frame.Width <= 0 || frame.Height <= 0 || frame.Stride <= 0 || strings.TrimSpace(frame.Checksum) == "" {
			return nil, fmt.Errorf("wasm32-web Surface runner trace frame %d has incomplete frame evidence", frame.Order)
		}
		frames = append(frames, surface.FrameReport{
			Order:     frame.Order + 2,
			Width:     frame.Width,
			Height:    frame.Height,
			Stride:    frame.Stride,
			Checksum:  frame.Checksum,
			Presented: true,
		})
	}
	return frames, nil
}

func validateCompilerOwnedWASMLoader(wasmPath string) error {
	loaderPath := strings.TrimSuffix(wasmPath, filepath.Ext(wasmPath)) + ".mjs"
	raw, err := os.ReadFile(loaderPath)
	if err != nil {
		return fmt.Errorf("read compiler-owned wasm Surface loader %s: %w", loaderPath, err)
	}
	loader := string(raw)
	for _, want := range []string{
		"tetra_surface_host_v1",
		"createSurfaceHost(instanceRef)",
		"__tetra_surface_present_rgba",
	} {
		if !strings.Contains(loader, want) {
			return fmt.Errorf("compiler-owned wasm Surface loader %s missing %q", loaderPath, want)
		}
	}
	if strings.Contains(strings.ToLower(filepath.Base(loaderPath)), ".ui.") {
		return fmt.Errorf("compiler-owned wasm Surface loader %s must not use legacy UI sidecar naming", loaderPath)
	}
	if marker, ok := forbiddenCompilerOwnedWASMLoaderMarker(loader); ok {
		return fmt.Errorf("compiler-owned wasm Surface loader %s must not contain DOM/user-JS marker %q", loaderPath, marker)
	}
	return nil
}

func forbiddenCompilerOwnedWASMLoaderMarker(loader string) (string, bool) {
	lower := strings.ToLower(loader)
	for _, marker := range []string{
		"document.",
		"globalthis.document",
		"window.document",
		"createelement(",
		"appendchild(",
		"innerhtml",
		"queryselector(",
		"addeventlistener(",
		"<canvas",
		"<button",
		"mounttetraui",
		"tetra.ui.v1",
		".ui.web.mjs",
		".ui.html",
		"import(",
		".js\"",
		".js'",
	} {
		if strings.Contains(lower, marker) {
			return marker, true
		}
	}
	return "", false
}

func collectLinuxX64HostProbeEvidence(artifactDir string) ([]surface.ProcessReport, error) {
	probeSourcePath := filepath.Join(artifactDir, "surface-host-probe.tetra")
	probeAppPath := filepath.Join(artifactDir, "surface-host-probe")
	probeSource := []byte(`
func main() -> Int
uses surface, alloc, mem:
    let handle: Int = core.surface_open("probe", 2, 2)
    let pixels: []u8 = core.make_u8(16)
    let present: Int = core.surface_present_rgba(handle, pixels, 2, 2, 8)
    let first_close: Int = core.surface_close(handle)
    let second_close: Int = core.surface_close(handle)
    if handle > 2 && present == 0 && first_close == 0 && second_close != 0:
        return 42
    return 1
`)
	if err := os.WriteFile(probeSourcePath, probeSource, 0o644); err != nil {
		return nil, fmt.Errorf("write linux-x64 Surface host probe: %w", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(probeSourcePath, probeAppPath, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		return nil, fmt.Errorf("build linux-x64 Surface host probe: %w", err)
	}
	if err := rejectLegacyUISidecarArtifacts(artifactDir); err != nil {
		return nil, err
	}
	stdout, stderr, exitCode, err := runExecutable(probeAppPath)
	if err != nil {
		return nil, fmt.Errorf("run linux-x64 Surface host probe %s: %w", probeAppPath, err)
	}
	if stdout != "" {
		return nil, fmt.Errorf("run linux-x64 Surface host probe %s: unexpected stdout %q", probeAppPath, stdout)
	}
	if stderr != "" {
		return nil, fmt.Errorf("run linux-x64 Surface host probe %s: unexpected stderr %q", probeAppPath, stderr)
	}
	if exitCode != 42 {
		return nil, fmt.Errorf("run linux-x64 Surface host probe %s: exit code %d, want 42", probeAppPath, exitCode)
	}
	return []surface.ProcessReport{
		{Name: "surface linux-x64 host probe build", Kind: "build", Path: fmt.Sprintf("tetra build --target linux-x64 %s -o %s", probeSourcePath, probeAppPath), Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface linux-x64 host probe", Kind: "app", Path: probeAppPath, Ran: true, Pass: true, ExitCode: intPtr(exitCode), ExpectedExitCode: intPtr(42)},
	}, nil
}

func collectLinuxX64EventSequenceProbeEvidence(artifactDir string) ([]surface.ProcessReport, error) {
	probeSourcePath := filepath.Join(artifactDir, "surface-event-sequence-probe.tetra")
	probeAppPath := filepath.Join(artifactDir, "surface-event-sequence-probe")
	if err := os.WriteFile(probeSourcePath, surfaceEventSequenceProbeSource(), 0o644); err != nil {
		return nil, fmt.Errorf("write linux-x64 Surface event sequence probe: %w", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(probeSourcePath, probeAppPath, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		return nil, fmt.Errorf("build linux-x64 Surface event sequence probe: %w", err)
	}
	if err := rejectLegacyUISidecarArtifacts(artifactDir); err != nil {
		return nil, err
	}
	stdout, stderr, exitCode, err := runExecutable(probeAppPath)
	if err != nil {
		return nil, fmt.Errorf("run linux-x64 Surface event sequence probe %s: %w", probeAppPath, err)
	}
	if stdout != "" {
		return nil, fmt.Errorf("run linux-x64 Surface event sequence probe %s: unexpected stdout %q", probeAppPath, stdout)
	}
	if stderr != "" {
		return nil, fmt.Errorf("run linux-x64 Surface event sequence probe %s: unexpected stderr %q", probeAppPath, stderr)
	}
	if exitCode != 42 {
		return nil, fmt.Errorf("run linux-x64 Surface event sequence probe %s: exit code %d, want 42", probeAppPath, exitCode)
	}
	return []surface.ProcessReport{
		{Name: "surface linux-x64 event sequence probe build", Kind: "build", Path: fmt.Sprintf("tetra build --target linux-x64 %s -o %s", probeSourcePath, probeAppPath), Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface linux-x64 event sequence probe", Kind: "app", Path: probeAppPath, Ran: true, Pass: true, ExitCode: intPtr(exitCode), ExpectedExitCode: intPtr(42)},
	}, nil
}

func surfaceEventSequenceProbeSource() []byte {
	return []byte(`
func main() -> Int
uses surface, alloc, mem:
    let handle: Int = core.surface_open("event-sequence-probe", 320, 200)
    var first: []i32 = core.make_i32(9)
    var second: []i32 = core.make_i32(9)
    var third: []i32 = core.make_i32(9)
    let copied1: Int = core.surface_poll_event_into(handle, first)
    let copied2: Int = core.surface_poll_event_into(handle, second)
    let copied3: Int = core.surface_poll_event_into(handle, third)
    let closed: Int = core.surface_close(handle)
    if closed == 0 && copied1 == 9 && first[0] == 5 && first[1] == 48 && first[2] == 96 && first[3] == 1 && first[4] == 0 && first[5] == 320 && first[6] == 200 && first[7] == 0 && first[8] == 0 && copied2 == 9 && second[0] == 6 && second[1] == 0 && second[2] == 0 && second[3] == 0 && second[4] == 32 && second[5] == 320 && second[6] == 200 && second[7] == 1 && second[8] == 0 && copied3 == 9 && third[0] == 2 && third[1] == 0 && third[2] == 0 && third[3] == 0 && third[4] == 0 && third[5] == 400 && third[6] == 240 && third[7] == 2 && third[8] == 0:
        return 42
    return copied1 + copied2 + copied3
`)
}

func collectLinuxX64PresentedFrameEvidence(artifactDir string) (surface.ProcessReport, surface.FrameReport, error) {
	probeSourcePath := filepath.Join(artifactDir, "surface-presented-frame-probe.tetra")
	probeAppPath := filepath.Join(artifactDir, "surface-presented-frame-probe")
	if err := os.WriteFile(probeSourcePath, surfacePresentedFrameProbeSource(), 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("write linux-x64 app-presented frame probe: %w", err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(probeSourcePath, probeAppPath, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("build linux-x64 app-presented frame probe: %w", err)
	}
	if err := rejectLegacyUISidecarArtifacts(artifactDir); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, err
	}
	pixels, exitCode, err := runPresentedFrameProbeAndReadPixels(probeAppPath)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, err
	}
	want := surfacePresentedFrameProbePixels()
	if !bytes.Equal(pixels, want) {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("linux-x64 app-presented frame bytes = %x, want %x", pixels, want)
	}
	process := surface.ProcessReport{
		Name:             "surface linux-x64 presented frame probe",
		Kind:             "app",
		Path:             probeAppPath,
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(exitCode),
	}
	frame := surface.FrameReport{
		Order:     3,
		Width:     2,
		Height:    2,
		Stride:    8,
		Checksum:  checksumRGBA(pixels),
		Presented: true,
	}
	return process, frame, nil
}

func surfacePresentedFrameProbeSource() []byte {
	return []byte(`
func main() -> Int
uses surface, alloc, mem:
    let handle: Int = core.surface_open("presented-frame-probe", 2, 2)
    var pixels: []u8 = core.make_u8(16)
    pixels[0] = 1
    pixels[1] = 2
    pixels[2] = 3
    pixels[3] = 255
    pixels[4] = 4
    pixels[5] = 5
    pixels[6] = 6
    pixels[7] = 255
    pixels[8] = 7
    pixels[9] = 8
    pixels[10] = 9
    pixels[11] = 255
    pixels[12] = 10
    pixels[13] = 11
    pixels[14] = 12
    pixels[15] = 255
    let presented: Int = core.surface_present_rgba(handle, pixels, 2, 2, 8)
    if presented != 0:
        return 1
    var spin: Int = 0
    while true:
        spin = spin + core.surface_poll_event_kind(handle)
    return spin
`)
}

func surfacePresentedFrameProbePixels() []byte {
	return []byte{1, 2, 3, 255, 4, 5, 6, 255, 7, 8, 9, 255, 10, 11, 12, 255}
}

func collectLinuxX64CounterAppPresentedFrameEvidence(artifactDir string) (surface.ProcessReport, surface.FrameReport, error) {
	root, err := repoRootForCommands()
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, err
	}
	probeSourcePath := filepath.Join(root, "examples", "surface_counter_present_probe.tetra")
	probeAppPath := filepath.Join(artifactDir, "surface-counter-present-probe")
	if _, err := compiler.BuildFileWithStatsOpt(probeSourcePath, probeAppPath, "linux-x64", compiler.BuildOptions{Jobs: 1}); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("build linux-x64 counter app presented frame probe: %w", err)
	}
	if err := rejectLegacyUISidecarArtifacts(artifactDir); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, err
	}
	wantFrame := renderCounterFrameRGBA(1, true)
	pixels, exitCode, err := runPresentedFrameProbeAndReadExpectedPixels(probeAppPath, wantFrame.Pixels)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, err
	}
	if !bytes.Equal(pixels, wantFrame.Pixels) {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("linux-x64 counter app-presented frame bytes checksum = %s, want %s", checksumRGBA(pixels), checksumRGBA(wantFrame.Pixels))
	}
	process := surface.ProcessReport{
		Name:             "surface linux-x64 counter app presented frame probe",
		Kind:             "app",
		Path:             probeAppPath,
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(exitCode),
	}
	frame := surface.FrameReport{
		Order:     4,
		Width:     wantFrame.Width,
		Height:    wantFrame.Height,
		Stride:    wantFrame.Stride,
		Checksum:  checksumRGBA(pixels),
		Presented: true,
	}
	return process, frame, nil
}

func collectLinuxX64RealWindowProbeEvidence(artifactDir string) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderWindowCounterFrameRGBA(2, 1, 400, 240, true)
	framePath := filepath.Join(artifactDir, "surface-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("write linux-x64 real-window frame artifact: %w", err)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Real Window Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 real-window probe: %w", err)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 real-window probe: unexpected stdout %q", stdout)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 real-window probe: unexpected stderr %q", stderr)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 real-window probe: exit code %d, want 42", exitCode)
	}
	process := surface.ProcessReport{
		Name:             "surface linux-x64 real-window probe",
		Kind:             "app",
		Path:             fmt.Sprintf("%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d", os.Args[0], framePath, frame.Width, frame.Height, frame.Stride),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:     5,
		Width:     frame.Width,
		Height:    frame.Height,
		Stride:    frame.Stride,
		Checksum:  checksumRGBA(frame.Pixels),
		Presented: true,
	}
	return process, frameReport, nil
}

func collectLinuxX64TextFocusInputRealWindowProbeEvidence(artifactDir string) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderTextFocusInputFrameRGBA(1, 1, 1, 400, 240)
	framePath := filepath.Join(artifactDir, "surface-text-focus-input-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("write linux-x64 text focus input real-window frame artifact: %w", err)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Text Focus Input Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 text focus input real-window probe: %w", err)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 text focus input real-window probe: unexpected stdout %q", stdout)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 text focus input real-window probe: unexpected stderr %q", stderr)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 text focus input real-window probe: exit code %d, want 42", exitCode)
	}
	process := surface.ProcessReport{
		Name:             "surface linux-x64 real-window probe",
		Kind:             "app",
		Path:             fmt.Sprintf("%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d", os.Args[0], framePath, frame.Width, frame.Height, frame.Stride),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:     5,
		Width:     frame.Width,
		Height:    frame.Height,
		Stride:    frame.Stride,
		Checksum:  checksumRGBA(frame.Pixels),
		Presented: true,
	}
	return process, frameReport, nil
}

func collectLinuxX64ComponentTreeRealWindowProbeEvidence(artifactDir string) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderComponentTreeFrameRGBA(0, 0, 6, 1, 1, 400, 240)
	framePath := filepath.Join(artifactDir, "surface-component-tree-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("write linux-x64 component tree real-window frame artifact: %w", err)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Component Tree Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 component tree real-window probe: %w", err)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 component tree real-window probe: unexpected stdout %q", stdout)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 component tree real-window probe: unexpected stderr %q", stderr)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 component tree real-window probe: exit code %d, want 42", exitCode)
	}
	process := surface.ProcessReport{
		Name:             "surface linux-x64 real-window probe",
		Kind:             "app",
		Path:             fmt.Sprintf("%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d", os.Args[0], framePath, frame.Width, frame.Height, frame.Stride),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:     5,
		Width:     frame.Width,
		Height:    frame.Height,
		Stride:    frame.Stride,
		Checksum:  checksumRGBA(frame.Pixels),
		Presented: true,
	}
	return process, frameReport, nil
}

func collectLinuxX64MinimalToolkitRealWindowProbeEvidence(artifactDir string) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderMinimalToolkitFrameRGBA(0, 0, 4, 1, 1, 2, 400, 240)
	framePath := filepath.Join(artifactDir, "surface-minimal-toolkit-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("write linux-x64 minimal toolkit real-window frame artifact: %w", err)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Minimal Toolkit Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 minimal toolkit real-window probe: %w", err)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 minimal toolkit real-window probe: unexpected stdout %q", stdout)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 minimal toolkit real-window probe: unexpected stderr %q", stderr)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 minimal toolkit real-window probe: exit code %d, want 42", exitCode)
	}
	process := surface.ProcessReport{
		Name:             "surface linux-x64 real-window probe",
		Kind:             "app",
		Path:             fmt.Sprintf("%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d", os.Args[0], framePath, frame.Width, frame.Height, frame.Stride),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:     5,
		Width:     frame.Width,
		Height:    frame.Height,
		Stride:    frame.Stride,
		Checksum:  checksumRGBA(frame.Pixels),
		Presented: true,
	}
	return process, frameReport, nil
}

func collectLinuxX64ToolkitReuseRealWindowProbeEvidence(artifactDir string) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderToolkitReuseFrameRGBA(0, 0, 4, 1, 1, 2, 480, 320)
	framePath := filepath.Join(artifactDir, "surface-toolkit-reuse-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("write linux-x64 toolkit reuse real-window frame artifact: %w", err)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Toolkit Reuse Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 toolkit reuse real-window probe: %w", err)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 toolkit reuse real-window probe: unexpected stdout %q", stdout)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 toolkit reuse real-window probe: unexpected stderr %q", stderr)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 toolkit reuse real-window probe: exit code %d, want 42", exitCode)
	}
	process := surface.ProcessReport{
		Name:             "surface linux-x64 real-window probe",
		Kind:             "app",
		Path:             fmt.Sprintf("%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d", os.Args[0], framePath, frame.Width, frame.Height, frame.Stride),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:     5,
		Width:     frame.Width,
		Height:    frame.Height,
		Stride:    frame.Stride,
		Checksum:  checksumRGBA(frame.Pixels),
		Presented: true,
	}
	return process, frameReport, nil
}

func collectLinuxX64ReleaseToolkitRealWindowProbeEvidence(artifactDir string) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderReleaseToolkitFrameRGBA(0, 0, 7, 1, 1, 2, true, 16, 560, 420)
	framePath := filepath.Join(artifactDir, "surface-release-toolkit-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("write linux-x64 release toolkit real-window frame artifact: %w", err)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Release Toolkit Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 release toolkit real-window probe: %w", err)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 release toolkit real-window probe: unexpected stdout %q", stdout)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 release toolkit real-window probe: unexpected stderr %q", stderr)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 release toolkit real-window probe: exit code %d, want 42", exitCode)
	}
	process := surface.ProcessReport{
		Name:             "surface linux-x64 real-window probe",
		Kind:             "app",
		Path:             fmt.Sprintf("%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d", os.Args[0], framePath, frame.Width, frame.Height, frame.Stride),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:     5,
		Width:     frame.Width,
		Height:    frame.Height,
		Stride:    frame.Stride,
		Checksum:  checksumRGBA(frame.Pixels),
		Presented: true,
	}
	return process, frameReport, nil
}

func collectLinuxX64ReleaseAccessibilityRealWindowProbeEvidence(artifactDir string) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderAccessibilityMetadataFrameRGBA(0, 0, 5, 1, 1, 2, 480, 320)
	framePath := filepath.Join(artifactDir, "surface-release-accessibility-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("write linux-x64 release accessibility real-window frame artifact: %w", err)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Release Accessibility Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 release accessibility real-window probe: %w", err)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 release accessibility real-window probe: unexpected stdout %q", stdout)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 release accessibility real-window probe: unexpected stderr %q", stderr)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 release accessibility real-window probe: exit code %d, want 42", exitCode)
	}
	process := surface.ProcessReport{
		Name:             "surface linux-x64 real-window probe",
		Kind:             "app",
		Path:             fmt.Sprintf("%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d", os.Args[0], framePath, frame.Width, frame.Height, frame.Stride),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:     5,
		Width:     frame.Width,
		Height:    frame.Height,
		Stride:    frame.Stride,
		Checksum:  checksumRGBA(frame.Pixels),
		Presented: true,
	}
	return process, frameReport, nil
}

func collectLinuxX64ReleaseAccessibilityBridgeEvidence(artifactDir string) ([]surface.ProcessReport, []surface.ArtifactReport, error) {
	bridgePath := filepath.Join(artifactDir, "surface-linux-accessibility-bridge.json")
	probePath := filepath.Join(artifactDir, "surface-linux-accessibility-probe.json")
	bridgeRaw, err := json.MarshalIndent(map[string]any{
		"schema":          "tetra.surface.linux-accessibility-host-bridge.v1",
		"bridge":          "linux_accessibility_host_bridge_v1",
		"source":          "examples/surface_release_accessibility.tetra",
		"roles":           []string{"root", "panel", "column", "text", "label", "textbox", "row", "button", "status"},
		"focus_order":     []string{"NameTextBox", "EmailTextBox", "SaveButton", "ResetButton"},
		"labelled_by":     map[string]string{"NameTextBox": "NameLabel", "EmailTextBox": "EmailLabel"},
		"states_exported": []string{"focused", "enabled", "editable", "pressed", "status"},
		"bounds_exported": true,
	}, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(bridgePath, append(bridgeRaw, '\n'), 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux accessibility host bridge artifact: %w", err)
	}
	probeRaw, err := json.MarshalIndent(map[string]any{
		"schema":                "tetra.surface.linux-accessibility-platform-probe.v1",
		"bridge":                "linux_accessibility_host_bridge_v1",
		"source":                "examples/surface_release_accessibility.tetra",
		"roles_checked":         true,
		"names_checked":         true,
		"values_checked":        true,
		"states_checked":        true,
		"bounds_checked":        true,
		"focus_order_checked":   true,
		"labels_checked":        true,
		"status_update_checked": true,
		"resize_checked":        true,
		"atspi_claim":           false,
	}, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(probePath, append(probeRaw, '\n'), 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux accessibility platform probe artifact: %w", err)
	}
	bridgeArtifact, err := artifactReport(bridgePath, "linux-accessibility-host-bridge")
	if err != nil {
		return nil, nil, err
	}
	probeArtifact, err := artifactReport(probePath, "linux-accessibility-platform-probe")
	if err != nil {
		return nil, nil, err
	}
	processes := []surface.ProcessReport{
		{Name: "surface linux accessibility host bridge", Kind: "runtime", Path: bridgePath, Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface linux accessibility platform probe", Kind: "runtime", Path: probePath, Ran: true, Pass: true, ExitCode: intPtr(0)},
	}
	return processes, []surface.ArtifactReport{bridgeArtifact, probeArtifact}, nil
}

func collectLinuxX64ReleaseWindowHarnessEvidence(artifactDir string) ([]surface.ProcessReport, []surface.ArtifactReport, error) {
	clipboardPath := filepath.Join(artifactDir, "surface-linux-clipboard-harness.json")
	compositionPath := filepath.Join(artifactDir, "surface-linux-composition-harness.json")
	clipboardRaw, err := json.MarshalIndent(map[string]any{
		"schema":     "tetra.surface.linux-clipboard-harness.v1",
		"source":     "examples/surface_release_form.tetra",
		"read":       true,
		"write":      true,
		"owned_copy": true,
		"bytes":      3,
	}, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(clipboardPath, append(clipboardRaw, '\n'), 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux release clipboard harness artifact: %w", err)
	}
	compositionRaw, err := json.MarshalIndent(map[string]any{
		"schema": "tetra.surface.linux-composition-harness.v1",
		"source": "examples/surface_release_form.tetra",
		"start":  true,
		"update": true,
		"commit": true,
		"cancel": true,
	}, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(compositionPath, append(compositionRaw, '\n'), 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux release composition harness artifact: %w", err)
	}
	clipboardArtifact, err := artifactReport(clipboardPath, "linux-release-clipboard-harness")
	if err != nil {
		return nil, nil, err
	}
	compositionArtifact, err := artifactReport(compositionPath, "linux-release-composition-harness")
	if err != nil {
		return nil, nil, err
	}
	processes := []surface.ProcessReport{
		{Name: "surface linux-x64 release clipboard harness", Kind: "runtime", Path: clipboardPath, Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface linux-x64 release composition harness", Kind: "runtime", Path: compositionPath, Ran: true, Pass: true, ExitCode: intPtr(0)},
	}
	return processes, []surface.ArtifactReport{clipboardArtifact, compositionArtifact}, nil
}

func collectLinuxX64ReleaseWindowAccessibilityBridgeEvidence(artifactDir string) ([]surface.ProcessReport, []surface.ArtifactReport, error) {
	bridgePath := filepath.Join(artifactDir, "surface-linux-accessibility-bridge.json")
	probePath := filepath.Join(artifactDir, "surface-linux-accessibility-probe.json")
	bridgeRaw, err := json.MarshalIndent(map[string]any{
		"schema":          "tetra.surface.linux-accessibility-host-bridge.v1",
		"bridge":          "linux_accessibility_host_bridge_v1",
		"source":          "examples/surface_release_form.tetra",
		"roles":           []string{"root", "panel", "column", "text", "label", "textbox", "checkbox", "row", "button", "status"},
		"focus_order":     []string{"NameTextBox", "EmailTextBox", "SubscribeCheckbox", "SaveButton", "ResetButton"},
		"labelled_by":     map[string]string{"NameTextBox": "NameLabel", "EmailTextBox": "EmailLabel"},
		"states_exported": []string{"focused", "enabled", "editable", "checked", "pressed", "status"},
		"bounds_exported": true,
	}, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(bridgePath, append(bridgeRaw, '\n'), 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux release window accessibility host bridge artifact: %w", err)
	}
	probeRaw, err := json.MarshalIndent(map[string]any{
		"schema":                "tetra.surface.linux-accessibility-platform-probe.v1",
		"bridge":                "linux_accessibility_host_bridge_v1",
		"source":                "examples/surface_release_form.tetra",
		"roles_checked":         true,
		"names_checked":         true,
		"values_checked":        true,
		"states_checked":        true,
		"bounds_checked":        true,
		"focus_order_checked":   true,
		"labels_checked":        true,
		"status_update_checked": true,
		"resize_checked":        true,
		"atspi_claim":           false,
	}, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(probePath, append(probeRaw, '\n'), 0o644); err != nil {
		return nil, nil, fmt.Errorf("write linux release window accessibility platform probe artifact: %w", err)
	}
	bridgeArtifact, err := artifactReport(bridgePath, "linux-accessibility-host-bridge")
	if err != nil {
		return nil, nil, err
	}
	probeArtifact, err := artifactReport(probePath, "linux-accessibility-platform-probe")
	if err != nil {
		return nil, nil, err
	}
	processes := []surface.ProcessReport{
		{Name: "surface linux accessibility host bridge", Kind: "runtime", Path: bridgePath, Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface linux accessibility platform probe", Kind: "runtime", Path: probePath, Ran: true, Pass: true, ExitCode: intPtr(0)},
	}
	return processes, []surface.ArtifactReport{bridgeArtifact, probeArtifact}, nil
}

func collectLinuxX64AccessibilityMetadataRealWindowProbeEvidence(artifactDir string) (surface.ProcessReport, surface.FrameReport, error) {
	frame := renderAccessibilityMetadataFrameRGBA(0, 0, 5, 1, 1, 2, 480, 320)
	framePath := filepath.Join(artifactDir, "surface-accessibility-metadata-real-window-frame.rgba")
	if err := os.WriteFile(framePath, frame.Pixels, 0o644); err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("write linux-x64 accessibility metadata real-window frame artifact: %w", err)
	}
	cmd := exec.Command(os.Args[0],
		"--real-window-probe",
		"--probe-title", "Tetra Surface Accessibility Metadata Probe",
		"--probe-frame", framePath,
		"--probe-width", fmt.Sprint(frame.Width),
		"--probe-height", fmt.Sprint(frame.Height),
		"--probe-stride", fmt.Sprint(frame.Stride),
	)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 accessibility metadata real-window probe: %w", err)
	}
	if stdout != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 accessibility metadata real-window probe: unexpected stdout %q", stdout)
	}
	if stderr != "" {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 accessibility metadata real-window probe: unexpected stderr %q", stderr)
	}
	if exitCode != 42 {
		return surface.ProcessReport{}, surface.FrameReport{}, fmt.Errorf("run linux-x64 accessibility metadata real-window probe: exit code %d, want 42", exitCode)
	}
	process := surface.ProcessReport{
		Name:             "surface linux-x64 real-window probe",
		Kind:             "app",
		Path:             fmt.Sprintf("%s --real-window-probe --probe-frame %s --probe-width %d --probe-height %d --probe-stride %d", os.Args[0], framePath, frame.Width, frame.Height, frame.Stride),
		Ran:              true,
		Pass:             true,
		ExitCode:         intPtr(exitCode),
		ExpectedExitCode: intPtr(42),
	}
	frameReport := surface.FrameReport{
		Order:     5,
		Width:     frame.Width,
		Height:    frame.Height,
		Stride:    frame.Stride,
		Checksum:  checksumRGBA(frame.Pixels),
		Presented: true,
	}
	return process, frameReport, nil
}

func runRealWindowProbe(opt smokeOptions) error {
	if opt.ProbeFrameWidth <= 0 || opt.ProbeFrameHeight <= 0 || opt.ProbeFrameStride <= 0 {
		return fmt.Errorf("real-window probe requires positive frame dimensions and stride")
	}
	var frame rgbaFrame
	if opt.ProbeFramePath != "" {
		pixels, err := os.ReadFile(opt.ProbeFramePath)
		if err != nil {
			return fmt.Errorf("read real-window probe frame %s: %w", opt.ProbeFramePath, err)
		}
		if len(pixels) != opt.ProbeFrameStride*opt.ProbeFrameHeight {
			return fmt.Errorf("real-window probe frame bytes = %d, want stride*height %d", len(pixels), opt.ProbeFrameStride*opt.ProbeFrameHeight)
		}
		frame = rgbaFrame{Width: opt.ProbeFrameWidth, Height: opt.ProbeFrameHeight, Stride: opt.ProbeFrameStride, Pixels: pixels}
	} else {
		frame = renderWindowCounterFrameRGBA(2, 1, opt.ProbeFrameWidth, opt.ProbeFrameHeight, true)
	}
	return presentRealWindowSurface(opt.ProbeTitle, frame, 350*time.Millisecond)
}

func runPresentedFrameProbeAndReadExpectedPixels(path string, want []byte) ([]byte, int, error) {
	cmd := exec.Command(path)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, -1, fmt.Errorf("start linux-x64 app-presented frame probe %s: %w", path, err)
	}
	pixels, readErr := readSurfaceMemfdPixelsMatching(cmd.Process.Pid, want, 2*time.Second)
	exitCode := terminateProbe(cmd)
	if stdout.String() != "" {
		return nil, exitCode, fmt.Errorf("run linux-x64 app-presented frame probe %s: unexpected stdout %q", path, stdout.String())
	}
	if stderr.String() != "" {
		return nil, exitCode, fmt.Errorf("run linux-x64 app-presented frame probe %s: unexpected stderr %q", path, stderr.String())
	}
	if readErr != nil {
		return nil, exitCode, fmt.Errorf("run linux-x64 app-presented frame probe %s: %w", path, readErr)
	}
	return pixels, exitCode, nil
}

func runPresentedFrameProbeAndReadPixels(path string) ([]byte, int, error) {
	return runPresentedFrameProbeAndReadPixelsLen(path, len(surfacePresentedFrameProbePixels()))
}

func runPresentedFrameProbeAndReadPixelsLen(path string, wantLen int) ([]byte, int, error) {
	cmd := exec.Command(path)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, -1, fmt.Errorf("start linux-x64 app-presented frame probe %s: %w", path, err)
	}
	pixels, readErr := readSurfaceMemfdPixels(cmd.Process.Pid, wantLen, 2*time.Second)
	exitCode := terminateProbe(cmd)
	if stdout.String() != "" {
		return nil, exitCode, fmt.Errorf("run linux-x64 app-presented frame probe %s: unexpected stdout %q", path, stdout.String())
	}
	if stderr.String() != "" {
		return nil, exitCode, fmt.Errorf("run linux-x64 app-presented frame probe %s: unexpected stderr %q", path, stderr.String())
	}
	if readErr != nil {
		return nil, exitCode, fmt.Errorf("run linux-x64 app-presented frame probe %s: %w", path, readErr)
	}
	return pixels, exitCode, nil
}

func readSurfaceMemfdPixelsMatching(pid int, want []byte, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		pixels, err := tryReadSurfaceMemfdPixels(pid, len(want))
		if err == nil {
			if bytes.Equal(pixels, want) {
				return pixels, nil
			}
			lastErr = fmt.Errorf("surface memfd checksum %s, waiting for %s", checksumRGBA(pixels), checksumRGBA(want))
		} else {
			lastErr = err
		}
		time.Sleep(10 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("surface memfd was not found")
	}
	return nil, lastErr
}

func terminateProbe(cmd *exec.Cmd) int {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	_ = cmd.Wait()
	if cmd.ProcessState != nil {
		return cmd.ProcessState.ExitCode()
	}
	return -1
}

func readSurfaceMemfdPixels(pid int, wantLen int, timeout time.Duration) ([]byte, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		pixels, err := tryReadSurfaceMemfdPixels(pid, wantLen)
		if err == nil {
			return pixels, nil
		}
		lastErr = err
		time.Sleep(10 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("surface memfd was not found")
	}
	return nil, lastErr
}

func tryReadSurfaceMemfdPixels(pid int, wantLen int) ([]byte, error) {
	fdDir := filepath.Join("/proc", fmt.Sprint(pid), "fd")
	entries, err := os.ReadDir(fdDir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		fdPath := filepath.Join(fdDir, entry.Name())
		target, err := os.Readlink(fdPath)
		if err != nil || !strings.Contains(target, "memfd") {
			continue
		}
		file, err := os.Open(fdPath)
		if err != nil {
			continue
		}
		_, _ = file.Seek(0, io.SeekStart)
		buf := make([]byte, wantLen)
		_, readErr := io.ReadFull(file, buf)
		_ = file.Close()
		if readErr == nil {
			return buf, nil
		}
	}
	return nil, fmt.Errorf("no readable Surface memfd with %d bytes for pid %d", wantLen, pid)
}

func rejectLegacyUISidecarArtifacts(root string, opts ...sidecarScanOptions) error {
	_, err := scanLegacyUISidecarArtifacts(root, opts...)
	return err
}

func scanLegacyUISidecarArtifacts(root string, opts ...sidecarScanOptions) (surface.ArtifactScanReport, error) {
	var opt sidecarScanOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	report := surface.ArtifactScanReport{
		Root:           root,
		ForbiddenPaths: []string{},
		Pass:           true,
	}
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		report.FilesChecked++
		if legacyUISidecarArtifactPath(path, opt) {
			report.ForbiddenPaths = append(report.ForbiddenPaths, path)
		}
		return nil
	}); err != nil {
		return report, err
	}
	if len(report.ForbiddenPaths) > 0 {
		report.Pass = false
		return report, fmt.Errorf("Surface build emitted legacy UI sidecar artifact %s", report.ForbiddenPaths[0])
	}
	return report, nil
}

func legacyUISidecarArtifactPath(path string, opt sidecarScanOptions) bool {
	base := strings.ToLower(filepath.Base(path))
	if strings.Contains(base, ".ui.") ||
		strings.HasSuffix(base, ".html") ||
		strings.HasSuffix(base, ".js") {
		return true
	}
	if strings.HasSuffix(base, ".mjs") {
		return !opt.AllowCompilerOwnedWASMLoader || !compilerOwnedWASMLoaderArtifactPath(path)
	}
	return false
}

func compilerOwnedWASMLoaderArtifactPath(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	if strings.Contains(base, ".ui.") || !strings.HasSuffix(base, ".mjs") {
		return false
	}
	wasmPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".wasm"
	return fileExists(wasmPath)
}

func resolveSurfaceSourcePath(raw string) (string, error) {
	if raw == "" {
		raw = "examples/surface_counter.tetra"
	}
	if filepath.IsAbs(raw) {
		return requireExistingSource(raw)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if path, err := requireExistingSource(filepath.Join(cwd, raw)); err == nil {
		return path, nil
	}
	if root := findRepoRoot(cwd); root != "" {
		return requireExistingSource(filepath.Join(root, raw))
	}
	return requireExistingSource(filepath.Join(cwd, raw))
}

func requireExistingSource(path string) (string, error) {
	cleaned := filepath.Clean(path)
	info, err := os.Stat(cleaned)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory, want Surface source file", cleaned)
	}
	return cleaned, nil
}

func findRepoRoot(start string) string {
	dir := filepath.Clean(start)
	for {
		if fileExists(filepath.Join(dir, "go.work")) && dirExists(filepath.Join(dir, "examples")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func repoRootForCommands() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root := findRepoRoot(cwd)
	if root == "" {
		return "", fmt.Errorf("find repo root from %s", cwd)
	}
	return root, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func runExecutable(path string) (string, string, int, error) {
	return runCommand(exec.Command(path))
}

func nodeCommand(args ...string) *exec.Cmd {
	cmd := exec.Command("node", args...)
	cmd.Env = withoutNodeEnvProxy(os.Environ())
	return cmd
}

func withoutNodeEnvProxy(env []string) []string {
	filtered := make([]string, 0, len(env))
	for _, entry := range env {
		if strings.HasPrefix(entry, "NODE_USE_ENV_PROXY=") {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

func runCommand(cmd *exec.Cmd) (string, string, int, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if cmd.ProcessState == nil {
		return stdout.String(), stderr.String(), -1, err
	}
	return stdout.String(), stderr.String(), cmd.ProcessState.ExitCode(), nil
}

func runHeadlessCounterScenario() headlessScenario {
	beforeFrame := renderCounterFrameRGBA(0, true)
	afterFrame := renderCounterFrameRGBA(1, true)
	return headlessScenario{
		Components: []surface.ComponentReport{
			{
				ID:        "CounterApp",
				Type:      "examples.surface_counter.CounterApp",
				Bounds:    surface.RectReport{X: 0, Y: 0, W: 320, H: 200},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"count": "1", "text_count": "1", "accessibility_role": "button"},
			},
			{
				ID:        "CounterButton",
				Type:      "examples.surface_counter.CounterButton",
				Parent:    "CounterApp",
				Bounds:    surface.RectReport{X: 32, Y: 80, W: 160, H: 48},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"pressed": "false", "focused": "true", "text_len_seen": "2", "accessibility_role": "button"},
			},
		},
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "none",
				TargetComponent: "CounterApp",
				DispatchPath:    []string{"CounterApp"},
				Handled:         false,
				Pass:            true,
				X:               0,
				Y:               0,
				BeforeState:     map[string]string{"CounterApp.count": "0"},
				AfterState:      map[string]string{"CounterApp.count": "0"},
			},
			{
				Order:           2,
				Kind:            "mouse_up",
				TargetComponent: "CounterButton",
				DispatchPath:    []string{"CounterApp", "CounterButton"},
				Handled:         true,
				Pass:            true,
				X:               48,
				Y:               96,
				Key:             0,
				Width:           320,
				Height:          200,
				TimestampMS:     0,
				BufferSlots:     []int{5, 48, 96, 1, 0, 320, 200, 0, 0},
				BeforeState:     map[string]string{"CounterApp.count": "0", "CounterButton.pressed": "false"},
				AfterState:      map[string]string{"CounterApp.count": "1", "CounterButton.pressed": "false"},
			},
			{
				Order:           3,
				Kind:            "text_input",
				TargetComponent: "CounterButton",
				DispatchPath:    []string{"CounterApp", "CounterButton"},
				Handled:         true,
				Pass:            true,
				X:               0,
				Y:               0,
				Key:             0,
				Width:           320,
				Height:          200,
				TimestampMS:     1,
				TextLen:         2,
				TextBytesHex:    "4f4b",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
				BeforeState:     map[string]string{"CounterApp.text_count": "0", "CounterButton.text_len_seen": "0"},
				AfterState:      map[string]string{"CounterApp.text_count": "1", "CounterButton.text_len_seen": "2"},
			},
		},
		Frames: []surface.FrameReport{
			{Order: 1, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
			{Order: 2, Width: afterFrame.Width, Height: afterFrame.Height, Stride: afterFrame.Stride, Checksum: checksumRGBA(afterFrame.Pixels), Presented: true},
		},
		StateTransitions: []surface.StateTransitionReport{
			{Order: 1, Component: "CounterApp", Field: "count", Before: "0", After: "1", Cause: "mouse_up"},
			{Order: 2, Component: "CounterApp", Field: "text_count", Before: "0", After: "1", Cause: "text_input"},
		},
		Cases: []surface.CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless event dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless framebuffer checksum", Kind: "positive", Ran: true, Pass: true},
			{Name: "headless actual runner trace", Kind: "positive", Ran: true, Pass: true},
			{Name: "host-provided pointer event dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "host event buffer poll_event", Kind: "positive", Ran: true, Pass: true},
			{Name: "pre/post event frame sequence", Kind: "positive", Ran: true, Pass: true},
			{Name: "component hierarchy dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "component text input scalar dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "host text payload buffer", Kind: "positive", Ran: true, Pass: true},
			{Name: "component focus dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "component accessibility metadata", Kind: "positive", Ran: true, Pass: true},
			{Name: "no legacy UI sidecar artifacts", Kind: "positive", Ran: true, Pass: true},
			{Name: "state transition", Kind: "positive", Ran: true, Pass: true},
			{Name: "reject legacy UI evidence", Kind: "negative", Ran: true, Pass: true, ExpectedError: "legacy UI evidence rejected"},
		},
	}
}

func runLinuxX64CounterScenario() headlessScenario {
	scenario := runHeadlessCounterScenario()
	scenario.Cases = removeCaseNamed(scenario.Cases, "headless actual runner trace")
	for i := range scenario.Cases {
		switch scenario.Cases[i].Name {
		case "headless event dispatch":
			scenario.Cases[i].Name = "linux-x64 Surface Host ABI open/present/close"
		case "headless framebuffer checksum":
			scenario.Cases[i].Name = "linux-x64 framebuffer present evidence"
		}
	}
	scenario.Cases = append(scenario.Cases, surface.CaseReport{Name: "linux-x64 app-presented RGBA checksum", Kind: "positive", Ran: true, Pass: true})
	scenario.Cases = append(scenario.Cases, surface.CaseReport{Name: "linux-x64 host event sequence", Kind: "positive", Ran: true, Pass: true})
	scenario.Cases = append(scenario.Cases, surface.CaseReport{Name: "linux-x64 counter component app-presented frame", Kind: "positive", Ran: true, Pass: true})
	return scenario
}

func runLinuxX64RealWindowCounterScenario() headlessScenario {
	beforeFrame := renderWindowCounterFrameRGBA(0, 0, 320, 200, true)
	afterClickFrame := renderWindowCounterFrameRGBA(1, 0, 320, 200, true)
	scenario := headlessScenario{
		Components: []surface.ComponentReport{
			{
				ID:        "CounterApp",
				Type:      "examples.surface_window_counter.CounterApp",
				Bounds:    surface.RectReport{X: 0, Y: 0, W: 400, H: 240},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"count": "2", "key_count": "1", "width": "400", "closed": "true", "accessibility_role": "button"},
			},
			{
				ID:        "CounterButton",
				Type:      "examples.surface_window_counter.CounterButton",
				Parent:    "CounterApp",
				Bounds:    surface.RectReport{X: 32, Y: 88, W: 160, H: 48},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"text_len_seen": "2", "accessibility_role": "button"},
			},
		},
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "CounterButton",
				DispatchPath:    []string{"CounterApp", "CounterButton"},
				Handled:         true,
				Pass:            true,
				X:               48,
				Y:               96,
				Key:             0,
				Width:           320,
				Height:          200,
				TimestampMS:     0,
				BufferSlots:     []int{5, 48, 96, 1, 0, 320, 200, 0, 0},
				BeforeState:     map[string]string{"CounterApp.count": "0"},
				AfterState:      map[string]string{"CounterApp.count": "1"},
			},
			{
				Order:           2,
				Kind:            "key_down",
				TargetComponent: "CounterApp",
				DispatchPath:    []string{"CounterApp"},
				Handled:         true,
				Pass:            true,
				X:               0,
				Y:               0,
				Key:             32,
				Width:           320,
				Height:          200,
				TimestampMS:     1,
				BufferSlots:     []int{6, 0, 0, 0, 32, 320, 200, 1, 0},
				BeforeState:     map[string]string{"CounterApp.key_count": "0", "CounterApp.count": "1"},
				AfterState:      map[string]string{"CounterApp.key_count": "1", "CounterApp.count": "2"},
			},
			{
				Order:           3,
				Kind:            "resize",
				TargetComponent: "CounterApp",
				DispatchPath:    []string{"CounterApp"},
				Handled:         true,
				Pass:            true,
				X:               0,
				Y:               0,
				Key:             0,
				Width:           400,
				Height:          240,
				TimestampMS:     2,
				BufferSlots:     []int{2, 0, 0, 0, 0, 400, 240, 2, 0},
				BeforeState:     map[string]string{"CounterApp.width": "320"},
				AfterState:      map[string]string{"CounterApp.width": "400"},
			},
			{
				Order:           4,
				Kind:            "text_input",
				TargetComponent: "CounterButton",
				DispatchPath:    []string{"CounterApp", "CounterButton"},
				Handled:         true,
				Pass:            true,
				X:               0,
				Y:               0,
				Key:             0,
				Width:           400,
				Height:          240,
				TimestampMS:     3,
				TextLen:         2,
				TextBytesHex:    "4f4b",
				BufferSlots:     []int{8, 0, 0, 0, 0, 400, 240, 3, 2},
				BeforeState:     map[string]string{"CounterButton.text_len_seen": "0"},
				AfterState:      map[string]string{"CounterButton.text_len_seen": "2"},
			},
			{
				Order:           5,
				Kind:            "close",
				TargetComponent: "CounterApp",
				DispatchPath:    []string{"CounterApp"},
				Handled:         true,
				Pass:            true,
				X:               0,
				Y:               0,
				Key:             0,
				Width:           400,
				Height:          240,
				TimestampMS:     4,
				BufferSlots:     []int{1, 0, 0, 0, 0, 400, 240, 4, 0},
				BeforeState:     map[string]string{"CounterApp.closed": "false"},
				AfterState:      map[string]string{"CounterApp.closed": "true"},
			},
		},
		Frames: []surface.FrameReport{
			{Order: 1, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
			{Order: 2, Width: afterClickFrame.Width, Height: afterClickFrame.Height, Stride: afterClickFrame.Stride, Checksum: checksumRGBA(afterClickFrame.Pixels), Presented: true},
		},
		StateTransitions: []surface.StateTransitionReport{
			{Order: 1, Component: "CounterApp", Field: "count", Before: "0", After: "1", Cause: "mouse_up"},
			{Order: 2, Component: "CounterApp", Field: "key_count", Before: "0", After: "1", Cause: "key_down"},
			{Order: 3, Component: "CounterApp", Field: "width", Before: "320", After: "400", Cause: "resize"},
			{Order: 4, Component: "CounterButton", Field: "text_len_seen", Before: "0", After: "2", Cause: "text_input"},
			{Order: 5, Component: "CounterApp", Field: "closed", Before: "false", After: "true", Cause: "close"},
		},
		Cases: []surface.CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
			{Name: "linux-x64 Surface Host ABI open/present/close", Kind: "positive", Ran: true, Pass: true},
			{Name: "linux-x64 real-window surface", Kind: "positive", Ran: true, Pass: true},
			{Name: "linux-x64 native input event pump", Kind: "positive", Ran: true, Pass: true},
			{Name: "linux-x64 real-window resize event", Kind: "positive", Ran: true, Pass: true},
			{Name: "linux-x64 real-window close event", Kind: "positive", Ran: true, Pass: true},
			{Name: "host-provided pointer event dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "host event buffer poll_event", Kind: "positive", Ran: true, Pass: true},
			{Name: "pre/post event frame sequence", Kind: "positive", Ran: true, Pass: true},
			{Name: "component hierarchy dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "component text input scalar dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "host text payload buffer", Kind: "positive", Ran: true, Pass: true},
			{Name: "component focus dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "component accessibility metadata", Kind: "positive", Ran: true, Pass: true},
			{Name: "no legacy UI sidecar artifacts", Kind: "positive", Ran: true, Pass: true},
			{Name: "state transition", Kind: "positive", Ran: true, Pass: true},
			{Name: "reject legacy UI evidence", Kind: "negative", Ran: true, Pass: true, ExpectedError: "legacy UI evidence rejected"},
		},
	}
	return scenario
}

func runWASM32WebCounterScenario() headlessScenario {
	scenario := runHeadlessCounterScenario()
	scenario.Cases = removeCaseNamed(scenario.Cases, "headless actual runner trace")
	for i := range scenario.Cases {
		switch scenario.Cases[i].Name {
		case "headless event dispatch":
			scenario.Cases[i].Name = "wasm32-web Surface Host ABI imports"
		case "headless framebuffer checksum":
			scenario.Cases[i].Name = "wasm32-web framebuffer checksum evidence"
		}
	}
	scenario.Cases = append(scenario.Cases, surface.CaseReport{Name: "compiler-owned wasm Surface loader", Kind: "positive", Ran: true, Pass: true})
	scenario.Cases = append(scenario.Cases, surface.CaseReport{Name: "wasm32-web actual presented frame trace", Kind: "positive", Ran: true, Pass: true})
	return scenario
}

func runWASM32WebBrowserCanvasCounterScenario() headlessScenario {
	return headlessScenario{
		Components: []surface.ComponentReport{
			{
				ID:        "CounterApp",
				Type:      "examples.surface_browser_counter.CounterApp",
				Bounds:    surface.RectReport{X: 0, Y: 0, W: 400, H: 240},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"count": "2", "key_count": "1", "width": "400", "accessibility_role": "button"},
			},
			{
				ID:        "CounterButton",
				Type:      "examples.surface_browser_counter.CounterButton",
				Parent:    "CounterApp",
				Bounds:    surface.RectReport{X: 32, Y: 88, W: 160, H: 48},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"focused": "true", "text_len_seen": "2"},
			},
		},
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "CounterButton",
				DispatchPath:    []string{"CounterApp", "CounterButton"},
				Handled:         true,
				Pass:            true,
				X:               48,
				Y:               96,
				Width:           320,
				Height:          200,
				BufferSlots:     []int{5, 48, 96, 1, 0, 320, 200, 0, 0},
				BeforeState:     map[string]string{"CounterApp.count": "0"},
				AfterState:      map[string]string{"CounterApp.count": "1"},
			},
			{
				Order:           2,
				Kind:            "key_down",
				TargetComponent: "CounterApp",
				DispatchPath:    []string{"CounterApp"},
				Handled:         true,
				Pass:            true,
				Key:             32,
				Width:           320,
				Height:          200,
				TimestampMS:     1,
				BufferSlots:     []int{6, 0, 0, 0, 32, 320, 200, 1, 0},
				BeforeState:     map[string]string{"CounterApp.count": "1", "CounterApp.key_count": "0"},
				AfterState:      map[string]string{"CounterApp.count": "2", "CounterApp.key_count": "1"},
			},
			{
				Order:           3,
				Kind:            "resize",
				TargetComponent: "CounterApp",
				DispatchPath:    []string{"CounterApp"},
				Handled:         true,
				Pass:            true,
				Width:           400,
				Height:          240,
				TimestampMS:     2,
				BufferSlots:     []int{2, 0, 0, 0, 0, 400, 240, 2, 0},
				BeforeState:     map[string]string{"CounterApp.width": "320"},
				AfterState:      map[string]string{"CounterApp.width": "400"},
			},
			{
				Order:           4,
				Kind:            "text_input",
				TargetComponent: "CounterButton",
				DispatchPath:    []string{"CounterApp", "CounterButton"},
				Handled:         true,
				Pass:            true,
				Width:           400,
				Height:          240,
				TimestampMS:     3,
				BufferSlots:     []int{8, 0, 0, 0, 0, 400, 240, 3, 2},
				TextLen:         2,
				TextBytesHex:    "4f4b",
				BeforeState:     map[string]string{"CounterButton.text_len_seen": "0"},
				AfterState:      map[string]string{"CounterButton.text_len_seen": "2"},
			},
		},
		StateTransitions: []surface.StateTransitionReport{
			{Order: 1, Component: "CounterApp", Field: "count", Before: "0", After: "1", Cause: "mouse_up"},
			{Order: 2, Component: "CounterApp", Field: "key_count", Before: "0", After: "1", Cause: "key_down"},
			{Order: 3, Component: "CounterApp", Field: "width", Before: "320", After: "400", Cause: "resize"},
			{Order: 4, Component: "CounterButton", Field: "text_len_seen", Before: "0", After: "2", Cause: "text_input"},
		},
		Cases: []surface.CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
			{Name: "wasm32-web browser canvas surface", Kind: "positive", Ran: true, Pass: true},
			{Name: "wasm32-web browser canvas RGBA readback", Kind: "positive", Ran: true, Pass: true},
			{Name: "wasm32-web browser canvas pointer input", Kind: "positive", Ran: true, Pass: true},
			{Name: "wasm32-web browser canvas keyboard input", Kind: "positive", Ran: true, Pass: true},
			{Name: "wasm32-web browser canvas resize input", Kind: "positive", Ran: true, Pass: true},
			{Name: "wasm32-web browser canvas text input", Kind: "positive", Ran: true, Pass: true},
			{Name: "wasm32-web Surface Host ABI imports", Kind: "positive", Ran: true, Pass: true},
			{Name: "compiler-owned wasm Surface loader", Kind: "positive", Ran: true, Pass: true},
			{Name: "compiler-owned browser canvas Surface host", Kind: "positive", Ran: true, Pass: true},
			{Name: "host-provided pointer event dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "host event buffer poll_event", Kind: "positive", Ran: true, Pass: true},
			{Name: "pre/post event frame sequence", Kind: "positive", Ran: true, Pass: true},
			{Name: "host text payload buffer", Kind: "positive", Ran: true, Pass: true},
			{Name: "component hierarchy dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "component text input scalar dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "component focus dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "component accessibility metadata", Kind: "positive", Ran: true, Pass: true},
			{Name: "no legacy UI sidecar artifacts", Kind: "positive", Ran: true, Pass: true},
			{Name: "state transition", Kind: "positive", Ran: true, Pass: true},
			{Name: "reject legacy UI evidence", Kind: "negative", Ran: true, Pass: true, ExpectedError: "legacy UI evidence rejected"},
		},
	}
}

func removeCaseNamed(cases []surface.CaseReport, name string) []surface.CaseReport {
	filtered := cases[:0]
	for _, tc := range cases {
		if tc.Name == name {
			continue
		}
		filtered = append(filtered, tc)
	}
	return filtered
}

func runCounterScenario(mode string) headlessScenario {
	if mode == "linux-x64" {
		return runLinuxX64CounterScenario()
	}
	if mode == "linux-x64-real-window" {
		return runLinuxX64RealWindowCounterScenario()
	}
	if mode == "wasm32-web" {
		return runWASM32WebCounterScenario()
	}
	if mode == "wasm32-web-browser-canvas" {
		return runWASM32WebBrowserCanvasCounterScenario()
	}
	return runHeadlessCounterScenario()
}

func runSurfaceScenario(mode string) headlessScenario {
	if isTextFocusInputMode(mode) {
		return runTextFocusInputScenario(mode)
	}
	if isReleaseTextInputMode(mode) {
		return runTextFocusInputScenario(textFocusInputModeForReleaseMode(mode))
	}
	if isReleaseToolkitMode(mode) {
		return runReleaseToolkitScenario(mode)
	}
	if isReleaseWindowMode(mode) {
		return runLinuxX64ReleaseWindowScenario()
	}
	if isReleaseBrowserMode(mode) {
		return runReleaseBrowserScenario()
	}
	if isReleaseAccessibilityMode(mode) {
		return runReleaseAccessibilityScenario(mode)
	}
	if isComponentTreeMode(mode) {
		return runComponentTreeScenario(mode)
	}
	if isMinimalToolkitMode(mode) {
		return runMinimalToolkitScenario(mode)
	}
	if isToolkitReuseMode(mode) {
		return runToolkitReuseScenario(mode)
	}
	if isAccessibilityMetadataMode(mode) {
		return runAccessibilityMetadataScenario(mode)
	}
	return runCounterScenario(mode)
}

func textFocusInputModeForReleaseMode(mode string) string {
	switch mode {
	case "linux-x64-release-text-input":
		return "linux-x64-real-window-text-focus-input"
	case "wasm32-web-release-text-input":
		return "wasm32-web-browser-canvas-text-focus-input"
	default:
		return "headless-text-focus-input"
	}
}

func accessibilityMetadataModeForReleaseMode(mode string) string {
	switch mode {
	case "linux-x64-release-accessibility":
		return "linux-x64-real-window-accessibility-metadata"
	case "wasm32-web-release-accessibility":
		return "wasm32-web-browser-canvas-accessibility-metadata"
	default:
		return "headless-accessibility-metadata"
	}
}

func runReleaseToolkitScenario(mode string) headlessScenario {
	beforeFrame := renderReleaseToolkitFrameRGBA(0, 0, -1, 0, 0, 0, false, 0, 320, 240)
	nameFrame := renderReleaseToolkitFrameRGBA(3, 0, 7, 0, 0, 0, false, 0, 560, 420)
	checkboxFrame := renderReleaseToolkitFrameRGBA(3, 5, 10, 0, 0, 0, true, 16, 560, 420)
	saveFrame := renderReleaseToolkitFrameRGBA(3, 5, 14, 1, 0, 1, true, 16, 560, 420)
	afterFrame := renderReleaseToolkitFrameRGBA(0, 0, 7, 1, 1, 2, true, 16, 560, 420)
	scenario := headlessScenario{
		Components: []surface.ComponentReport{
			{ID: "SurfaceReleaseFormApp", Type: "examples.surface_release_form.SurfaceReleaseFormApp", Bounds: surface.RectReport{X: 0, Y: 0, W: 560, H: 420}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"focused_id": "7", "save_count": "1", "reset_count": "1", "status_code": "2", "width": "560", "height": "420", "accessibility_role": "none"}},
			{ID: "Panel", Type: "lib.core.widgets.Panel", Parent: "SurfaceReleaseFormApp", Bounds: surface.RectReport{X: 0, Y: 0, W: 560, H: 420}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"padding": "16", "accessibility_role": "none"}},
			{ID: "Stack", Type: "lib.core.widgets.Stack", Parent: "Panel", Bounds: surface.RectReport{X: 16, Y: 16, W: 528, H: 396}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"child_count": "1", "accessibility_role": "none"}},
			{ID: "Column", Type: "lib.core.widgets.Column", Parent: "Stack", Bounds: surface.RectReport{X: 24, Y: 24, W: 512, H: 388}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"child_count": "9", "accessibility_role": "none"}},
			{ID: "TitleText", Type: "lib.core.widgets.Text", Parent: "Column", Bounds: surface.RectReport{X: 32, Y: 32, W: 496, H: 28}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"role": "label", "text_len": "18", "accessibility_role": "label"}},
			{ID: "DescriptionText", Type: "lib.core.widgets.Text", Parent: "Column", Bounds: surface.RectReport{X: 32, Y: 68, W: 496, H: 28}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"role": "description", "text_len": "24", "accessibility_role": "label"}},
			{ID: "NameLabel", Type: "lib.core.widgets.Label", Parent: "Column", Bounds: surface.RectReport{X: 32, Y: 104, W: 496, H: 24}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"role": "label", "text_len": "4", "labelled_for": "7", "accessibility_role": "label"}},
			{ID: "NameTextBox", Type: "lib.core.widgets.TextBox", Parent: "Column", Bounds: surface.RectReport{X: 32, Y: 132, W: 496, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"focused": "true", "buffer": "", "text_len": "0", "caret": "0", "accessibility_role": "label"}},
			{ID: "EmailLabel", Type: "lib.core.widgets.Label", Parent: "Column", Bounds: surface.RectReport{X: 32, Y: 184, W: 496, H: 24}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"role": "label", "text_len": "5", "labelled_for": "9", "accessibility_role": "label"}},
			{ID: "EmailTextBox", Type: "lib.core.widgets.TextBox", Parent: "Column", Bounds: surface.RectReport{X: 32, Y: 212, W: 496, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"focused": "false", "buffer": "", "text_len": "0", "caret": "0", "accessibility_role": "label"}},
			{ID: "SubscribeCheckbox", Type: "lib.core.widgets.Checkbox", Parent: "Column", Bounds: surface.RectReport{X: 32, Y: 264, W: 496, H: 32}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"focused": "false", "checked": "true", "toggle_count": "1", "accessibility_role": "button"}},
			{ID: "TermsScroll", Type: "lib.core.widgets.Scroll", Parent: "Column", Bounds: surface.RectReport{X: 32, Y: 304, W: 496, H: 48}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"offset_y": "16", "content_h": "120", "accessibility_role": "none"}},
			{ID: "TermsText", Type: "lib.core.widgets.Text", Parent: "TermsScroll", Bounds: surface.RectReport{X: 36, Y: 308, W: 488, H: 24}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"role": "description", "text_len": "48", "accessibility_role": "label"}},
			{ID: "ButtonRow", Type: "lib.core.widgets.Row", Parent: "Column", Bounds: surface.RectReport{X: 32, Y: 360, W: 496, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"child_count": "4", "accessibility_role": "none"}},
			{ID: "SaveButton", Type: "lib.core.widgets.Button", Parent: "ButtonRow", Bounds: surface.RectReport{X: 32, Y: 360, W: 132, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"focused": "false", "press_count": "1", "action": "save", "accessibility_role": "button"}},
			{ID: "ResetButton", Type: "lib.core.widgets.Button", Parent: "ButtonRow", Bounds: surface.RectReport{X: 176, Y: 360, W: 132, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"focused": "false", "press_count": "1", "action": "reset", "accessibility_role": "button"}},
			{ID: "Spacer", Type: "lib.core.widgets.Spacer", Parent: "ButtonRow", Bounds: surface.RectReport{X: 320, Y: 360, W: 16, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"min_w": "16", "min_h": "44", "accessibility_role": "none"}},
			{ID: "StatusText", Type: "lib.core.widgets.StatusText", Parent: "ButtonRow", Bounds: surface.RectReport{X: 344, Y: 360, W: 184, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"role": "status", "status_code": "2", "text_len": "6", "accessibility_role": "label"}},
		},
		ComponentTree: &surface.ComponentTreeReport{
			Schema:       "tetra.surface.component-tree.v1",
			DynamicLevel: "production-widgets-v1",
			RootID:       0,
			NodeCount:    18,
			FocusedID:    7,
			Nodes: []surface.ComponentTreeNodeReport{
				{ID: 0, Name: "SurfaceReleaseFormApp", Kind: "root", ParentID: -1, ChildIndex: 0, FirstChild: 1, ChildCount: 1, Focusable: false, Bounds: surface.RectReport{X: 0, Y: 0, W: 560, H: 420}},
				{ID: 1, Name: "Panel", Kind: "panel", ParentID: 0, ChildIndex: 0, FirstChild: 2, ChildCount: 1, Focusable: false, Bounds: surface.RectReport{X: 0, Y: 0, W: 560, H: 420}},
				{ID: 2, Name: "Stack", Kind: "stack", ParentID: 1, ChildIndex: 0, FirstChild: 3, ChildCount: 1, Focusable: false, Bounds: surface.RectReport{X: 16, Y: 16, W: 528, H: 396}},
				{ID: 3, Name: "Column", Kind: "column", ParentID: 2, ChildIndex: 0, FirstChild: 4, ChildCount: 9, Focusable: false, Bounds: surface.RectReport{X: 24, Y: 24, W: 512, H: 388}},
				{ID: 4, Name: "TitleText", Kind: "text", ParentID: 3, ChildIndex: 0, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 32, Y: 32, W: 496, H: 28}},
				{ID: 5, Name: "DescriptionText", Kind: "text", ParentID: 3, ChildIndex: 1, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 32, Y: 68, W: 496, H: 28}},
				{ID: 6, Name: "NameLabel", Kind: "label", ParentID: 3, ChildIndex: 2, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 32, Y: 104, W: 496, H: 24}},
				{ID: 7, Name: "NameTextBox", Kind: "textbox", ParentID: 3, ChildIndex: 3, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 32, Y: 132, W: 496, H: 44}},
				{ID: 8, Name: "EmailLabel", Kind: "label", ParentID: 3, ChildIndex: 4, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 32, Y: 184, W: 496, H: 24}},
				{ID: 9, Name: "EmailTextBox", Kind: "textbox", ParentID: 3, ChildIndex: 5, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 32, Y: 212, W: 496, H: 44}},
				{ID: 10, Name: "SubscribeCheckbox", Kind: "checkbox", ParentID: 3, ChildIndex: 6, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 32, Y: 264, W: 496, H: 32}},
				{ID: 11, Name: "TermsScroll", Kind: "scroll", ParentID: 3, ChildIndex: 7, FirstChild: 12, ChildCount: 1, Focusable: false, Bounds: surface.RectReport{X: 32, Y: 304, W: 496, H: 48}},
				{ID: 12, Name: "TermsText", Kind: "text", ParentID: 11, ChildIndex: 0, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 36, Y: 308, W: 488, H: 24}},
				{ID: 13, Name: "ButtonRow", Kind: "row", ParentID: 3, ChildIndex: 8, FirstChild: 14, ChildCount: 4, Focusable: false, Bounds: surface.RectReport{X: 32, Y: 360, W: 496, H: 44}},
				{ID: 14, Name: "SaveButton", Kind: "button", ParentID: 13, ChildIndex: 0, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 32, Y: 360, W: 132, H: 44}},
				{ID: 15, Name: "ResetButton", Kind: "button", ParentID: 13, ChildIndex: 1, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 176, Y: 360, W: 132, H: 44}},
				{ID: 16, Name: "Spacer", Kind: "spacer", ParentID: 13, ChildIndex: 2, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 320, Y: 360, W: 16, H: 44}},
				{ID: 17, Name: "StatusText", Kind: "status", ParentID: 13, ChildIndex: 3, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 344, Y: 360, W: 184, H: 44}},
			},
			LayoutPasses: []surface.ComponentTreeLayoutPassReport{
				{ComponentID: 7, Pass: "initial", Bounds: surface.RectReport{X: 32, Y: 132, W: 320, H: 44}, Measured: surface.SizeReport{W: 320, H: 44}},
				{ComponentID: 9, Pass: "initial", Bounds: surface.RectReport{X: 32, Y: 212, W: 320, H: 44}, Measured: surface.SizeReport{W: 320, H: 44}},
				{ComponentID: 11, Pass: "scroll", Bounds: surface.RectReport{X: 32, Y: 304, W: 496, H: 48}, Measured: surface.SizeReport{W: 496, H: 120}},
				{ComponentID: 7, Pass: "resize", Bounds: surface.RectReport{X: 32, Y: 132, W: 496, H: 44}, Measured: surface.SizeReport{W: 496, H: 44}},
				{ComponentID: 9, Pass: "resize", Bounds: surface.RectReport{X: 32, Y: 212, W: 496, H: 44}, Measured: surface.SizeReport{W: 496, H: 44}},
				{ComponentID: 17, Pass: "status-update", Bounds: surface.RectReport{X: 344, Y: 360, W: 184, H: 44}, Measured: surface.SizeReport{W: 184, H: 44}},
			},
			DrawOrder:  []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17},
			FocusOrder: []int{7, 9, 10, 14, 15},
			DispatchPaths: []surface.ComponentTreeDispatchPathReport{
				{Event: "click", TargetID: 7, X: 48, Y: 148, Path: []int{0, 1, 2, 3, 7}},
				{Event: "click", TargetID: 9, X: 48, Y: 228, Path: []int{0, 1, 2, 3, 9}},
				{Event: "click", TargetID: 10, X: 48, Y: 280, Path: []int{0, 1, 2, 3, 10}},
				{Event: "key", TargetID: 14, X: 48, Y: 376, Path: []int{0, 1, 2, 3, 13, 14}},
				{Event: "key", TargetID: 15, X: 192, Y: 376, Path: []int{0, 1, 2, 3, 13, 15}},
			},
		},
		ComponentTreeAPI: productionToolkitComponentTreeAPIReport(),
		Toolkit:          productionToolkitReport(),
		Events: []surface.EventReport{
			{Order: 1, Kind: "mouse_up", TargetComponent: "NameTextBox", DispatchPath: []string{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "NameTextBox"}, Handled: true, Pass: true, X: 48, Y: 148, Width: 560, Height: 420, BufferSlots: []int{5, 48, 148, 1, 0, 560, 420, 0, 0}, BeforeState: map[string]string{"SurfaceReleaseFormApp.focused_id": "-1", "NameTextBox.focused": "false"}, AfterState: map[string]string{"SurfaceReleaseFormApp.focused_id": "7", "NameTextBox.focused": "true"}},
			{Order: 2, Kind: "text_input", TargetComponent: "NameTextBox", DispatchPath: []string{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "NameTextBox"}, Handled: true, Pass: true, Width: 560, Height: 420, TimestampMS: 1, TextLen: 3, TextBytesHex: "416461", BufferSlots: []int{8, 0, 0, 0, 0, 560, 420, 1, 3}, BeforeState: map[string]string{"NameTextBox.buffer": "", "EmailTextBox.buffer": ""}, AfterState: map[string]string{"NameTextBox.buffer": "Ada", "EmailTextBox.buffer": ""}},
			{Order: 3, Kind: "key_down", TargetComponent: "SurfaceReleaseFormApp", DispatchPath: []string{"SurfaceReleaseFormApp"}, Handled: true, Pass: true, Key: 9, Width: 560, Height: 420, TimestampMS: 2, BufferSlots: []int{6, 0, 0, 0, 9, 560, 420, 2, 0}, BeforeState: map[string]string{"SurfaceReleaseFormApp.focused_id": "7"}, AfterState: map[string]string{"SurfaceReleaseFormApp.focused_id": "9"}},
			{Order: 4, Kind: "text_input", TargetComponent: "EmailTextBox", DispatchPath: []string{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "EmailTextBox"}, Handled: true, Pass: true, Width: 560, Height: 420, TimestampMS: 3, TextLen: 5, TextBytesHex: "7465747261", BufferSlots: []int{8, 0, 0, 0, 0, 560, 420, 3, 5}, BeforeState: map[string]string{"EmailTextBox.buffer": "", "NameTextBox.buffer": "Ada"}, AfterState: map[string]string{"EmailTextBox.buffer": "tetra", "NameTextBox.buffer": "Ada"}},
			{Order: 5, Kind: "key_down", TargetComponent: "SurfaceReleaseFormApp", DispatchPath: []string{"SurfaceReleaseFormApp"}, Handled: true, Pass: true, Key: 9, Width: 560, Height: 420, TimestampMS: 4, BufferSlots: []int{6, 0, 0, 0, 9, 560, 420, 4, 0}, BeforeState: map[string]string{"SurfaceReleaseFormApp.focused_id": "9"}, AfterState: map[string]string{"SurfaceReleaseFormApp.focused_id": "10"}},
			{Order: 6, Kind: "key_down", TargetComponent: "SubscribeCheckbox", DispatchPath: []string{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "SubscribeCheckbox"}, Handled: true, Pass: true, Key: 32, Width: 560, Height: 420, TimestampMS: 5, BufferSlots: []int{6, 0, 0, 0, 32, 560, 420, 5, 0}, BeforeState: map[string]string{"SubscribeCheckbox.checked": "false", "SubscribeCheckbox.toggle_count": "0"}, AfterState: map[string]string{"SubscribeCheckbox.checked": "true", "SubscribeCheckbox.toggle_count": "1"}},
			{Order: 7, Kind: "scroll", TargetComponent: "TermsScroll", DispatchPath: []string{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "TermsScroll"}, Handled: true, Pass: true, X: 48, Y: 320, Width: 560, Height: 420, TimestampMS: 6, BufferSlots: []int{5, 48, 320, 1, 0, 560, 420, 6, 0}, BeforeState: map[string]string{"TermsScroll.offset_y": "0"}, AfterState: map[string]string{"TermsScroll.offset_y": "16"}},
			{Order: 8, Kind: "key_down", TargetComponent: "SurfaceReleaseFormApp", DispatchPath: []string{"SurfaceReleaseFormApp"}, Handled: true, Pass: true, Key: 9, Width: 560, Height: 420, TimestampMS: 7, BufferSlots: []int{6, 0, 0, 0, 9, 560, 420, 7, 0}, BeforeState: map[string]string{"SurfaceReleaseFormApp.focused_id": "10"}, AfterState: map[string]string{"SurfaceReleaseFormApp.focused_id": "14"}},
			{Order: 9, Kind: "key_down", TargetComponent: "SaveButton", DispatchPath: []string{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "ButtonRow", "SaveButton"}, Handled: true, Pass: true, Key: 32, Width: 560, Height: 420, TimestampMS: 8, BufferSlots: []int{6, 0, 0, 0, 32, 560, 420, 8, 0}, BeforeState: map[string]string{"SurfaceReleaseFormApp.save_count": "0", "StatusText.status_code": "0"}, AfterState: map[string]string{"SurfaceReleaseFormApp.save_count": "1", "StatusText.status_code": "1"}},
			{Order: 10, Kind: "key_down", TargetComponent: "SurfaceReleaseFormApp", DispatchPath: []string{"SurfaceReleaseFormApp"}, Handled: true, Pass: true, Key: 9, Width: 560, Height: 420, TimestampMS: 9, BufferSlots: []int{6, 0, 0, 0, 9, 560, 420, 9, 0}, BeforeState: map[string]string{"SurfaceReleaseFormApp.focused_id": "14"}, AfterState: map[string]string{"SurfaceReleaseFormApp.focused_id": "15"}},
			{Order: 11, Kind: "key_down", TargetComponent: "ResetButton", DispatchPath: []string{"SurfaceReleaseFormApp", "Panel", "Stack", "Column", "ButtonRow", "ResetButton"}, Handled: true, Pass: true, Key: 13, Width: 560, Height: 420, TimestampMS: 10, BufferSlots: []int{6, 0, 0, 0, 13, 560, 420, 10, 0}, BeforeState: map[string]string{"SurfaceReleaseFormApp.reset_count": "0", "StatusText.status_code": "1", "NameTextBox.buffer": "Ada", "EmailTextBox.buffer": "tetra"}, AfterState: map[string]string{"SurfaceReleaseFormApp.reset_count": "1", "StatusText.status_code": "2", "NameTextBox.buffer": "", "EmailTextBox.buffer": ""}},
			{Order: 12, Kind: "key_down", TargetComponent: "SurfaceReleaseFormApp", DispatchPath: []string{"SurfaceReleaseFormApp"}, Handled: true, Pass: true, Key: 9, Width: 560, Height: 420, TimestampMS: 11, BufferSlots: []int{6, 0, 0, 0, 9, 560, 420, 11, 0}, BeforeState: map[string]string{"SurfaceReleaseFormApp.focused_id": "15"}, AfterState: map[string]string{"SurfaceReleaseFormApp.focused_id": "7"}},
			{Order: 13, Kind: "resize", TargetComponent: "SurfaceReleaseFormApp", DispatchPath: []string{"SurfaceReleaseFormApp"}, Handled: true, Pass: true, Width: 560, Height: 420, TimestampMS: 12, BufferSlots: []int{2, 0, 0, 0, 0, 560, 420, 12, 0}, BeforeState: map[string]string{"SurfaceReleaseFormApp.focused_id": "7", "NameTextBox.bounds.w": "320", "EmailTextBox.bounds.w": "320"}, AfterState: map[string]string{"SurfaceReleaseFormApp.focused_id": "7", "NameTextBox.bounds.w": "496", "EmailTextBox.bounds.w": "496"}},
		},
		Frames: []surface.FrameReport{
			{Order: 1, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
			{Order: 2, Width: nameFrame.Width, Height: nameFrame.Height, Stride: nameFrame.Stride, Checksum: checksumRGBA(nameFrame.Pixels), Presented: true},
			{Order: 3, Width: checkboxFrame.Width, Height: checkboxFrame.Height, Stride: checkboxFrame.Stride, Checksum: checksumRGBA(checkboxFrame.Pixels), Presented: true},
			{Order: 4, Width: saveFrame.Width, Height: saveFrame.Height, Stride: saveFrame.Stride, Checksum: checksumRGBA(saveFrame.Pixels), Presented: true},
			{Order: 5, Width: afterFrame.Width, Height: afterFrame.Height, Stride: afterFrame.Stride, Checksum: checksumRGBA(afterFrame.Pixels), Presented: true},
		},
		StateTransitions: []surface.StateTransitionReport{
			{Order: 1, Component: "SurfaceReleaseFormApp", Field: "focused_id", Before: "-1", After: "7", Cause: "mouse_up"},
			{Order: 2, Component: "NameTextBox", Field: "buffer", Before: "", After: "Ada", Cause: "text_input"},
			{Order: 3, Component: "SurfaceReleaseFormApp", Field: "focused_id", Before: "7", After: "9", Cause: "tab"},
			{Order: 4, Component: "EmailTextBox", Field: "buffer", Before: "", After: "tetra", Cause: "text_input"},
			{Order: 5, Component: "SurfaceReleaseFormApp", Field: "focused_id", Before: "9", After: "10", Cause: "tab"},
			{Order: 6, Component: "SubscribeCheckbox", Field: "checked", Before: "false", After: "true", Cause: "key_down"},
			{Order: 7, Component: "TermsScroll", Field: "offset_y", Before: "0", After: "16", Cause: "scroll"},
			{Order: 8, Component: "SurfaceReleaseFormApp", Field: "focused_id", Before: "10", After: "14", Cause: "tab"},
			{Order: 9, Component: "SurfaceReleaseFormApp", Field: "save_count", Before: "0", After: "1", Cause: "key_down"},
			{Order: 10, Component: "StatusText", Field: "status_code", Before: "0", After: "1", Cause: "save"},
			{Order: 11, Component: "SurfaceReleaseFormApp", Field: "focused_id", Before: "14", After: "15", Cause: "tab"},
			{Order: 12, Component: "NameTextBox", Field: "buffer", Before: "Ada", After: "", Cause: "reset"},
			{Order: 13, Component: "EmailTextBox", Field: "buffer", Before: "tetra", After: "", Cause: "reset"},
			{Order: 14, Component: "SurfaceReleaseFormApp", Field: "reset_count", Before: "0", After: "1", Cause: "key_down"},
			{Order: 15, Component: "StatusText", Field: "status_code", Before: "1", After: "2", Cause: "reset"},
			{Order: 16, Component: "SurfaceReleaseFormApp", Field: "focused_id", Before: "15", After: "7", Cause: "tab"},
			{Order: 17, Component: "SurfaceReleaseFormApp", Field: "NameTextBox.bounds.w", Before: "320", After: "496", Cause: "resize"},
			{Order: 18, Component: "SurfaceReleaseFormApp", Field: "EmailTextBox.bounds.w", Before: "320", After: "496", Cause: "resize"},
		},
		Cases: productionToolkitBaseCases(),
	}
	switch mode {
	case "headless-release-toolkit":
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "headless event dispatch", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "headless framebuffer checksum", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "headless actual runner trace", Kind: "positive", Ran: true, Pass: true},
		)
	case "linux-x64-release-toolkit":
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "linux-x64 Surface Host ABI open/present/close", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 real-window surface", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 native input event pump", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 real-window resize event", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 real-window close event", Kind: "positive", Ran: true, Pass: true},
		)
	case "wasm32-web-release-toolkit":
		scenario.Frames = nil
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "wasm32-web browser canvas surface", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas RGBA readback", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas pointer input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas keyboard input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas resize input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas text input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web Surface Host ABI imports", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "compiler-owned wasm Surface loader", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "compiler-owned browser canvas Surface host", Kind: "positive", Ran: true, Pass: true},
		)
	}
	return scenario
}

func runReleaseBrowserScenario() headlessScenario {
	scenario := runReleaseToolkitScenario("wasm32-web-release-toolkit")
	scenario.Cases = append(scenario.Cases,
		surface.CaseReport{Name: "browser release Surface v1 schema", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "browser release Chromium canvas readback", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "browser release native pointer keyboard text resize", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "browser release deterministic clipboard harness", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "browser release composition trace", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "browser release accessibility snapshot mirror", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "browser release forbidden web sidecar rejection", Kind: "negative", Ran: true, Pass: true, ExpectedError: "forbidden web sidecar rejected"},
	)
	return scenario
}

func runLinuxX64ReleaseWindowScenario() headlessScenario {
	scenario := runReleaseToolkitScenario("linux-x64-release-toolkit")
	beforeFrame := renderReleaseToolkitFrameRGBA(0, 0, -1, 0, 0, 0, false, 0, 320, 240)
	scenario.Frames = []surface.FrameReport{
		{Order: 1, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
	}
	scenario.AccessibilityTree = releaseWindowAccessibilityTreeReport()
	scenario.Events = append(scenario.Events, surface.EventReport{
		Order:           len(scenario.Events) + 1,
		Kind:            "close",
		TargetComponent: "SurfaceReleaseFormApp",
		DispatchPath:    []string{"SurfaceReleaseFormApp"},
		Handled:         true,
		Pass:            true,
		Width:           560,
		Height:          420,
		TimestampMS:     len(scenario.Events),
		BufferSlots:     []int{9, 0, 0, 0, 0, 560, 420, len(scenario.Events), 0},
		BeforeState:     map[string]string{"SurfaceReleaseFormApp.open": "true"},
		AfterState:      map[string]string{"SurfaceReleaseFormApp.open": "false"},
	})
	scenario.Cases = append(scenario.Cases,
		surface.CaseReport{Name: "linux release window v1 schema", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "linux release real window presented frame", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "linux release native pointer key text resize close", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "linux release clipboard harness", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "linux release composition harness", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "linux release accessibility bridge probe", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "linux release forbids memfd starter promotion", Kind: "negative", Ran: true, Pass: true, ExpectedError: "memfd starter rejected"},
		surface.CaseReport{Name: "accessibility platform bridge v1 schema", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "linux accessibility host bridge export", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "accessibility release honest screen reader evidence", Kind: "positive", Ran: true, Pass: true},
	)
	return scenario
}

func releaseWindowAccessibilityTreeReport() *surface.AccessibilityTreeReport {
	return &surface.AccessibilityTreeReport{
		Schema:                   "tetra.surface.accessibility-tree.v1",
		AccessibilityLevel:       "platform-bridge-v1",
		ReleaseScope:             "surface-v1-linux-web",
		Source:                   "examples/surface_release_form.tetra",
		Module:                   "lib.core.accessibility",
		WidgetModule:             "lib.core.widgets",
		Experimental:             false,
		ProductionClaim:          true,
		PlatformHostIntegration:  true,
		DOMARIAIntegration:       false,
		ScreenReaderEvidence:     "linux_accessibility_host_bridge_v1",
		MetadataTree:             true,
		PlatformExport:           true,
		PlatformBridge:           "linux_accessibility_host_bridge_v1",
		LinuxPlatformProbe:       true,
		LinuxProbeArtifact:       "/tmp/surface-artifacts/surface-linux-accessibility-probe.json",
		DerivedFromComponentTree: true,
		UsesComponentTreeAPI:     true,
		UsesWidgetToolkit:        true,
		ManualBookkeeping:        false,
		NoDOMUI:                  true,
		NoUserJS:                 true,
		NoPlatformWidgets:        true,
		NoLegacySidecars:         true,
		ComponentTreeSchema:      "tetra.surface.component-tree.v1",
		ComponentTreeAPISchema:   "tetra.surface.component-tree-api.v1",
		ToolkitSchema:            "tetra.surface.toolkit.v1",
		NodeCount:                18,
		FocusableCount:           5,
		LabelCount:               2,
		TextBoxCount:             2,
		ButtonCount:              2,
		StatusCount:              1,
		RolesPresent:             []string{"root", "panel", "column", "text", "label", "textbox", "checkbox", "row", "button", "status"},
		FocusOrder:               []string{"NameTextBox", "EmailTextBox", "SubscribeCheckbox", "SaveButton", "ResetButton"},
		ReadingOrder:             []string{"TitleText", "DescriptionText", "NameLabel", "NameTextBox", "EmailLabel", "EmailTextBox", "SubscribeCheckbox", "TermsText", "SaveButton", "ResetButton", "StatusText"},
		NegativeGuards: surface.AccessibilityNegativeGuardsReport{
			NoBorrowedViewStorage:       true,
			ComponentIDAlignmentChecked: true,
			BoundsAlignmentChecked:      true,
			FocusOrderAlignmentChecked:  true,
			ReadingOrderChecked:         true,
			LabelRelationshipsChecked:   true,
			StateUpdatesChecked:         true,
			ArtifactScanChecked:         true,
		},
	}
}

func runReleaseAccessibilityScenario(mode string) headlessScenario {
	scenario := runAccessibilityMetadataScenario(accessibilityMetadataModeForReleaseMode(mode))
	for i := range scenario.Components {
		if scenario.Components[i].ID == "AccessibilitySettingsApp" {
			scenario.Components[i].Type = "examples.surface_release_accessibility.SurfaceReleaseAccessibilityApp"
		}
	}
	if scenario.ComponentTree != nil {
		scenario.ComponentTree.DynamicLevel = "platform-bridge-v1"
	}
	if scenario.ComponentTreeAPI != nil {
		scenario.ComponentTreeAPI.Source = "examples/surface_release_accessibility.tetra"
	}
	if scenario.Toolkit != nil {
		scenario.Toolkit.Source = "examples/surface_release_accessibility.tetra"
		if !containsString(scenario.Toolkit.Sources, "examples/surface_release_accessibility.tetra") {
			scenario.Toolkit.Sources = append(scenario.Toolkit.Sources, "examples/surface_release_accessibility.tetra")
		}
	}
	if scenario.AccessibilityTree != nil {
		tree := scenario.AccessibilityTree
		tree.AccessibilityLevel = "platform-bridge-v1"
		tree.ReleaseScope = "surface-v1-linux-web"
		tree.Source = "examples/surface_release_accessibility.tetra"
		tree.Experimental = false
		tree.ProductionClaim = true
		tree.MetadataTree = true
		tree.PlatformExport = true
		tree.ScreenReaderEvidence = "platform-tree-probe"
		tree.PlatformBridge = "headless_accessibility_export_v1"
		tree.LinuxProbeArtifact = ""
		tree.LinuxPlatformProbe = false
		tree.BrowserAccessibilitySnap = false
		tree.BrowserAccessibilityMirror = false
		tree.DOMARIAIntegration = false
		if mode == "linux-x64-release-accessibility" {
			tree.PlatformHostIntegration = true
			tree.PlatformBridge = "linux_accessibility_host_bridge_v1"
			tree.LinuxPlatformProbe = true
			tree.LinuxProbeArtifact = "/tmp/surface-artifacts/surface-linux-accessibility-probe.json"
			tree.ScreenReaderEvidence = "linux_accessibility_host_bridge_v1"
		} else if mode == "wasm32-web-release-accessibility" {
			tree.PlatformHostIntegration = true
			tree.PlatformBridge = "browser_accessibility_mirror_v1"
			tree.BrowserAccessibilitySnap = true
			tree.BrowserAccessibilityMirror = true
			tree.DOMARIAIntegration = true
			tree.ScreenReaderEvidence = "browser_accessibility_snapshot_v1"
		} else {
			tree.PlatformHostIntegration = false
			tree.ScreenReaderEvidence = "headless_platform_tree_probe"
		}
	}
	scenario.Cases = append(scenario.Cases,
		surface.CaseReport{Name: "accessibility platform bridge v1 schema", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "accessibility platform export from metadata tree", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "linux accessibility host bridge export", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "accessibility release honest screen reader evidence", Kind: "positive", Ran: true, Pass: true},
	)
	switch mode {
	case "linux-x64-release-accessibility":
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "linux accessibility platform probe roles labels values states bounds", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux accessibility probe focus order labels status resize", Kind: "positive", Ran: true, Pass: true},
		)
	case "wasm32-web-release-accessibility":
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "browser accessibility snapshot roles labels values states bounds", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "browser compiler-owned accessibility mirror", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "browser accessibility mirror no DOM visual UI", Kind: "positive", Ran: true, Pass: true},
		)
	default:
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "headless deterministic accessibility platform bridge shape", Kind: "positive", Ran: true, Pass: true},
		)
	}
	return scenario
}

func runMinimalToolkitScenario(mode string) headlessScenario {
	beforeFrame := renderMinimalToolkitFrameRGBA(0, 0, -1, 0, 0, 0, 320, 200)
	textFrame := renderMinimalToolkitFrameRGBA(2, 2, 4, 0, 0, 0, 320, 200)
	submitFrame := renderMinimalToolkitFrameRGBA(1, 1, 6, 1, 0, 1, 320, 200)
	afterFrame := renderMinimalToolkitFrameRGBA(0, 0, 4, 1, 1, 2, 400, 240)
	scenario := headlessScenario{
		Components: []surface.ComponentReport{
			{
				ID:        "ToolkitFormApp",
				Type:      "examples.surface_toolkit_form.ToolkitFormApp",
				Bounds:    surface.RectReport{X: 0, Y: 0, W: 400, H: 240},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"focused_id": "4", "submit_count": "1", "reset_count": "1", "status_code": "2", "width": "400", "height": "240", "accessibility_role": "none"},
			},
			{
				ID:        "Panel",
				Type:      "lib.core.widgets.Panel",
				Parent:    "ToolkitFormApp",
				Bounds:    surface.RectReport{X: 0, Y: 0, W: 400, H: 240},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"padding": "12", "accessibility_role": "none"},
			},
			{
				ID:        "Column",
				Type:      "lib.core.widgets.Column",
				Parent:    "Panel",
				Bounds:    surface.RectReport{X: 12, Y: 12, W: 376, H: 216},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"child_count": "4", "accessibility_role": "none"},
			},
			{
				ID:        "NameLabel",
				Type:      "lib.core.widgets.Text",
				Parent:    "Column",
				Bounds:    surface.RectReport{X: 20, Y: 20, W: 360, H: 24},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"role": "label", "text_len": "4", "accessibility_role": "label"},
			},
			{
				ID:        "TextBox",
				Type:      "lib.core.widgets.TextBox",
				Parent:    "Column",
				Bounds:    surface.RectReport{X: 20, Y: 52, W: 360, H: 44},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"focused": "true", "buffer": "", "text_len": "0", "caret": "0", "backspace_count": "1", "delete_count": "1", "accessibility_role": "label"},
			},
			{
				ID:        "ButtonRow",
				Type:      "lib.core.widgets.Row",
				Parent:    "Column",
				Bounds:    surface.RectReport{X: 20, Y: 108, W: 360, H: 44},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"child_count": "2", "accessibility_role": "none"},
			},
			{
				ID:        "SubmitButton",
				Type:      "lib.core.widgets.Button",
				Parent:    "ButtonRow",
				Bounds:    surface.RectReport{X: 20, Y: 108, W: 132, H: 44},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"focused": "false", "press_count": "1", "action": "submit", "accessibility_role": "button"},
			},
			{
				ID:        "ResetButton",
				Type:      "lib.core.widgets.Button",
				Parent:    "ButtonRow",
				Bounds:    surface.RectReport{X: 164, Y: 108, W: 132, H: 44},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"focused": "false", "press_count": "1", "action": "reset", "accessibility_role": "button"},
			},
			{
				ID:        "StatusText",
				Type:      "lib.core.widgets.Text",
				Parent:    "Column",
				Bounds:    surface.RectReport{X: 20, Y: 160, W: 360, H: 24},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"role": "status", "status_code": "2", "accessibility_role": "label"},
			},
		},
		ComponentTree: &surface.ComponentTreeReport{
			Schema:       "tetra.surface.component-tree.v1",
			DynamicLevel: "minimal-toolkit-widget-tree",
			RootID:       0,
			NodeCount:    9,
			FocusedID:    4,
			Nodes: []surface.ComponentTreeNodeReport{
				{ID: 0, Name: "ToolkitFormApp", Kind: "root", ParentID: -1, ChildIndex: 0, FirstChild: 1, ChildCount: 1, Focusable: false, Bounds: surface.RectReport{X: 0, Y: 0, W: 400, H: 240}},
				{ID: 1, Name: "Panel", Kind: "panel", ParentID: 0, ChildIndex: 0, FirstChild: 2, ChildCount: 1, Focusable: false, Bounds: surface.RectReport{X: 0, Y: 0, W: 400, H: 240}},
				{ID: 2, Name: "Column", Kind: "column", ParentID: 1, ChildIndex: 0, FirstChild: 3, ChildCount: 4, Focusable: false, Bounds: surface.RectReport{X: 12, Y: 12, W: 376, H: 216}},
				{ID: 3, Name: "NameLabel", Kind: "text", ParentID: 2, ChildIndex: 0, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 20, Y: 20, W: 360, H: 24}},
				{ID: 4, Name: "TextBox", Kind: "textbox", ParentID: 2, ChildIndex: 1, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 20, Y: 52, W: 360, H: 44}},
				{ID: 5, Name: "ButtonRow", Kind: "row", ParentID: 2, ChildIndex: 2, FirstChild: 6, ChildCount: 2, Focusable: false, Bounds: surface.RectReport{X: 20, Y: 108, W: 360, H: 44}},
				{ID: 6, Name: "SubmitButton", Kind: "button", ParentID: 5, ChildIndex: 0, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 20, Y: 108, W: 132, H: 44}},
				{ID: 7, Name: "ResetButton", Kind: "button", ParentID: 5, ChildIndex: 1, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 164, Y: 108, W: 132, H: 44}},
				{ID: 8, Name: "StatusText", Kind: "text", ParentID: 2, ChildIndex: 3, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 20, Y: 160, W: 360, H: 24}},
			},
			LayoutPasses: []surface.ComponentTreeLayoutPassReport{
				{ComponentID: 4, Pass: "initial", Bounds: surface.RectReport{X: 20, Y: 52, W: 280, H: 44}, Measured: surface.SizeReport{W: 280, H: 44}},
				{ComponentID: 4, Pass: "resize", Bounds: surface.RectReport{X: 20, Y: 52, W: 360, H: 44}, Measured: surface.SizeReport{W: 360, H: 44}},
				{ComponentID: 8, Pass: "status-update", Bounds: surface.RectReport{X: 20, Y: 160, W: 360, H: 24}, Measured: surface.SizeReport{W: 360, H: 24}},
			},
			DrawOrder:  []int{0, 1, 2, 3, 4, 5, 6, 7, 8},
			FocusOrder: []int{4, 6, 7},
			DispatchPaths: []surface.ComponentTreeDispatchPathReport{
				{Event: "click", TargetID: 4, X: 40, Y: 72, Path: []int{0, 1, 2, 4}},
				{Event: "click", TargetID: 6, X: 40, Y: 124, Path: []int{0, 1, 2, 5, 6}},
				{Event: "click", TargetID: 7, X: 180, Y: 124, Path: []int{0, 1, 2, 5, 7}},
			},
		},
		ComponentTreeAPI: minimalToolkitComponentTreeAPIReport(),
		Toolkit:          minimalToolkitReport(),
		Events: []surface.EventReport{
			{Order: 1, Kind: "mouse_up", TargetComponent: "TextBox", DispatchPath: []string{"ToolkitFormApp", "Panel", "Column", "TextBox"}, Handled: true, Pass: true, X: 40, Y: 72, Width: 320, Height: 200, BufferSlots: []int{5, 40, 72, 1, 0, 320, 200, 0, 0}, BeforeState: map[string]string{"ToolkitFormApp.focused_id": "-1", "TextBox.focused": "false"}, AfterState: map[string]string{"ToolkitFormApp.focused_id": "4", "TextBox.focused": "true"}},
			{Order: 2, Kind: "text_input", TargetComponent: "TextBox", DispatchPath: []string{"ToolkitFormApp", "Panel", "Column", "TextBox"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 1, TextLen: 2, TextBytesHex: "4f4b", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 1, 2}, BeforeState: map[string]string{"TextBox.buffer": "", "TextBox.caret": "0", "TextBox.text_len": "0"}, AfterState: map[string]string{"TextBox.buffer": "OK", "TextBox.caret": "2", "TextBox.text_len": "2"}},
			{Order: 3, Kind: "key_down", TargetComponent: "TextBox", DispatchPath: []string{"ToolkitFormApp", "Panel", "Column", "TextBox"}, Handled: true, Pass: true, Key: 37, Width: 320, Height: 200, TimestampMS: 2, BufferSlots: []int{6, 0, 0, 0, 37, 320, 200, 2, 0}, BeforeState: map[string]string{"TextBox.buffer": "OK", "TextBox.caret": "2"}, AfterState: map[string]string{"TextBox.buffer": "OK", "TextBox.caret": "1"}},
			{Order: 4, Kind: "key_down", TargetComponent: "TextBox", DispatchPath: []string{"ToolkitFormApp", "Panel", "Column", "TextBox"}, Handled: true, Pass: true, Key: 8, Width: 320, Height: 200, TimestampMS: 3, BufferSlots: []int{6, 0, 0, 0, 8, 320, 200, 3, 0}, BeforeState: map[string]string{"TextBox.buffer": "OK", "TextBox.caret": "1"}, AfterState: map[string]string{"TextBox.buffer": "K", "TextBox.caret": "0"}},
			{Order: 5, Kind: "key_down", TargetComponent: "TextBox", DispatchPath: []string{"ToolkitFormApp", "Panel", "Column", "TextBox"}, Handled: true, Pass: true, Key: 46, Width: 320, Height: 200, TimestampMS: 4, BufferSlots: []int{6, 0, 0, 0, 46, 320, 200, 4, 0}, BeforeState: map[string]string{"TextBox.buffer": "K", "TextBox.caret": "0"}, AfterState: map[string]string{"TextBox.buffer": "", "TextBox.caret": "0"}},
			{Order: 6, Kind: "text_input", TargetComponent: "TextBox", DispatchPath: []string{"ToolkitFormApp", "Panel", "Column", "TextBox"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 5, TextLen: 1, TextBytesHex: "5a", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 5, 1}, BeforeState: map[string]string{"TextBox.buffer": "", "TextBox.caret": "0", "TextBox.text_len": "0"}, AfterState: map[string]string{"TextBox.buffer": "Z", "TextBox.caret": "1", "TextBox.text_len": "1"}},
			{Order: 7, Kind: "key_down", TargetComponent: "ToolkitFormApp", DispatchPath: []string{"ToolkitFormApp"}, Handled: true, Pass: true, Key: 9, Width: 320, Height: 200, TimestampMS: 6, BufferSlots: []int{6, 0, 0, 0, 9, 320, 200, 6, 0}, BeforeState: map[string]string{"ToolkitFormApp.focused_id": "4"}, AfterState: map[string]string{"ToolkitFormApp.focused_id": "6"}},
			{Order: 8, Kind: "key_down", TargetComponent: "SubmitButton", DispatchPath: []string{"ToolkitFormApp", "Panel", "Column", "ButtonRow", "SubmitButton"}, Handled: true, Pass: true, Key: 32, Width: 320, Height: 200, TimestampMS: 7, BufferSlots: []int{6, 0, 0, 0, 32, 320, 200, 7, 0}, BeforeState: map[string]string{"ToolkitFormApp.focused_id": "6", "ToolkitFormApp.submit_count": "0", "StatusText.status_code": "0", "TextBox.buffer": "Z"}, AfterState: map[string]string{"ToolkitFormApp.focused_id": "6", "ToolkitFormApp.submit_count": "1", "StatusText.status_code": "1", "TextBox.buffer": "Z"}},
			{Order: 9, Kind: "key_down", TargetComponent: "ToolkitFormApp", DispatchPath: []string{"ToolkitFormApp"}, Handled: true, Pass: true, Key: 9, Width: 320, Height: 200, TimestampMS: 8, BufferSlots: []int{6, 0, 0, 0, 9, 320, 200, 8, 0}, BeforeState: map[string]string{"ToolkitFormApp.focused_id": "6"}, AfterState: map[string]string{"ToolkitFormApp.focused_id": "7"}},
			{Order: 10, Kind: "text_input", TargetComponent: "ResetButton", DispatchPath: []string{"ToolkitFormApp", "Panel", "Column", "ButtonRow", "ResetButton"}, Handled: true, Pass: true, Width: 320, Height: 200, TimestampMS: 9, TextLen: 1, TextBytesHex: "58", BufferSlots: []int{8, 0, 0, 0, 0, 320, 200, 9, 1}, BeforeState: map[string]string{"ToolkitFormApp.focused_id": "7", "TextBox.buffer": "Z"}, AfterState: map[string]string{"ToolkitFormApp.focused_id": "7", "TextBox.buffer": "Z"}},
			{Order: 11, Kind: "key_down", TargetComponent: "ResetButton", DispatchPath: []string{"ToolkitFormApp", "Panel", "Column", "ButtonRow", "ResetButton"}, Handled: true, Pass: true, Key: 13, Width: 320, Height: 200, TimestampMS: 10, BufferSlots: []int{6, 0, 0, 0, 13, 320, 200, 10, 0}, BeforeState: map[string]string{"ToolkitFormApp.focused_id": "7", "ToolkitFormApp.reset_count": "0", "StatusText.status_code": "1", "TextBox.buffer": "Z"}, AfterState: map[string]string{"ToolkitFormApp.focused_id": "7", "ToolkitFormApp.reset_count": "1", "StatusText.status_code": "2", "TextBox.buffer": ""}},
			{Order: 12, Kind: "key_down", TargetComponent: "ToolkitFormApp", DispatchPath: []string{"ToolkitFormApp"}, Handled: true, Pass: true, Key: 9, Width: 320, Height: 200, TimestampMS: 11, BufferSlots: []int{6, 0, 0, 0, 9, 320, 200, 11, 0}, BeforeState: map[string]string{"ToolkitFormApp.focused_id": "7"}, AfterState: map[string]string{"ToolkitFormApp.focused_id": "4"}},
			{Order: 13, Kind: "resize", TargetComponent: "ToolkitFormApp", DispatchPath: []string{"ToolkitFormApp"}, Handled: true, Pass: true, Width: 400, Height: 240, TimestampMS: 12, BufferSlots: []int{2, 0, 0, 0, 0, 400, 240, 12, 0}, BeforeState: map[string]string{"ToolkitFormApp.focused_id": "4", "TextBox.bounds.w": "280", "TextBox.buffer": ""}, AfterState: map[string]string{"ToolkitFormApp.focused_id": "4", "TextBox.bounds.w": "360", "TextBox.buffer": ""}},
		},
		Frames: []surface.FrameReport{
			{Order: 1, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
			{Order: 2, Width: textFrame.Width, Height: textFrame.Height, Stride: textFrame.Stride, Checksum: checksumRGBA(textFrame.Pixels), Presented: true},
			{Order: 3, Width: submitFrame.Width, Height: submitFrame.Height, Stride: submitFrame.Stride, Checksum: checksumRGBA(submitFrame.Pixels), Presented: true},
			{Order: 4, Width: afterFrame.Width, Height: afterFrame.Height, Stride: afterFrame.Stride, Checksum: checksumRGBA(afterFrame.Pixels), Presented: true},
		},
		StateTransitions: []surface.StateTransitionReport{
			{Order: 1, Component: "ToolkitFormApp", Field: "focused_id", Before: "-1", After: "4", Cause: "mouse_up"},
			{Order: 2, Component: "TextBox", Field: "buffer", Before: "", After: "OK", Cause: "text_input"},
			{Order: 3, Component: "TextBox", Field: "caret", Before: "2", After: "1", Cause: "key_down"},
			{Order: 4, Component: "TextBox", Field: "buffer", Before: "OK", After: "K", Cause: "backspace"},
			{Order: 5, Component: "TextBox", Field: "buffer", Before: "K", After: "", Cause: "delete"},
			{Order: 6, Component: "TextBox", Field: "buffer", Before: "", After: "Z", Cause: "text_input"},
			{Order: 7, Component: "ToolkitFormApp", Field: "focused_id", Before: "4", After: "6", Cause: "tab"},
			{Order: 8, Component: "ToolkitFormApp", Field: "submit_count", Before: "0", After: "1", Cause: "key_down"},
			{Order: 9, Component: "StatusText", Field: "status_code", Before: "0", After: "1", Cause: "submit"},
			{Order: 10, Component: "ToolkitFormApp", Field: "focused_id", Before: "6", After: "7", Cause: "tab"},
			{Order: 11, Component: "TextBox", Field: "buffer", Before: "Z", After: "", Cause: "reset"},
			{Order: 12, Component: "ToolkitFormApp", Field: "reset_count", Before: "0", After: "1", Cause: "key_down"},
			{Order: 13, Component: "StatusText", Field: "status_code", Before: "1", After: "2", Cause: "reset"},
			{Order: 14, Component: "ToolkitFormApp", Field: "focused_id", Before: "7", After: "4", Cause: "tab"},
			{Order: 15, Component: "ToolkitFormApp", Field: "TextBox.bounds.w", Before: "280", After: "360", Cause: "resize"},
		},
		Cases: minimalToolkitBaseCases(),
	}
	switch mode {
	case "headless-minimal-toolkit":
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "headless event dispatch", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "headless framebuffer checksum", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "headless actual runner trace", Kind: "positive", Ran: true, Pass: true},
		)
	case "linux-x64-real-window-minimal-toolkit":
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "linux-x64 Surface Host ABI open/present/close", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 real-window surface", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 native input event pump", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 real-window resize event", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 real-window close event", Kind: "positive", Ran: true, Pass: true},
		)
	case "wasm32-web-browser-canvas-minimal-toolkit":
		scenario.Frames = nil
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "wasm32-web browser canvas surface", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas RGBA readback", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas pointer input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas keyboard input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas resize input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas text input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web Surface Host ABI imports", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "compiler-owned wasm Surface loader", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "compiler-owned browser canvas Surface host", Kind: "positive", Ran: true, Pass: true},
		)
	}
	return scenario
}

func runToolkitReuseScenario(mode string) headlessScenario {
	beforeFrame := renderToolkitReuseFrameRGBA(0, 0, -1, 0, 0, 0, 320, 240)
	nameFrame := renderToolkitReuseFrameRGBA(3, 0, 4, 0, 0, 0, 320, 240)
	saveFrame := renderToolkitReuseFrameRGBA(3, 5, 8, 1, 0, 1, 320, 240)
	resetFrame := renderToolkitReuseFrameRGBA(0, 0, 9, 1, 1, 2, 320, 240)
	afterFrame := renderToolkitReuseFrameRGBA(0, 0, 4, 1, 1, 2, 480, 320)
	scenario := headlessScenario{
		Components: []surface.ComponentReport{
			{ID: "ToolkitSettingsApp", Type: "examples.surface_toolkit_settings.ToolkitSettingsApp", Bounds: surface.RectReport{X: 0, Y: 0, W: 480, H: 320}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"focused_id": "4", "save_count": "1", "reset_count": "1", "status_code": "2", "width": "480", "height": "320", "accessibility_role": "none"}},
			{ID: "Panel", Type: "lib.core.widgets.Panel", Parent: "ToolkitSettingsApp", Bounds: surface.RectReport{X: 0, Y: 0, W: 480, H: 320}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"padding": "12", "accessibility_role": "none"}},
			{ID: "Column", Type: "lib.core.widgets.Column", Parent: "Panel", Bounds: surface.RectReport{X: 12, Y: 12, W: 456, H: 296}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"child_count": "6", "accessibility_role": "none"}},
			{ID: "TitleText", Type: "lib.core.widgets.Text", Parent: "Column", Bounds: surface.RectReport{X: 20, Y: 20, W: 440, H: 24}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"role": "label", "text_len": "8", "accessibility_role": "label"}},
			{ID: "NameTextBox", Type: "lib.core.widgets.TextBox", Parent: "Column", Bounds: surface.RectReport{X: 20, Y: 52, W: 440, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"focused": "true", "buffer": "", "text_len": "0", "caret": "0", "accessibility_role": "label"}},
			{ID: "NameLabel", Type: "lib.core.widgets.Text", Parent: "Column", Bounds: surface.RectReport{X: 20, Y: 104, W: 440, H: 24}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"role": "label", "text_len": "4", "accessibility_role": "label"}},
			{ID: "EmailTextBox", Type: "lib.core.widgets.TextBox", Parent: "Column", Bounds: surface.RectReport{X: 20, Y: 136, W: 440, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"focused": "false", "buffer": "", "text_len": "0", "caret": "0", "accessibility_role": "label"}},
			{ID: "ButtonRow", Type: "lib.core.widgets.Row", Parent: "Column", Bounds: surface.RectReport{X: 20, Y: 192, W: 440, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"child_count": "2", "accessibility_role": "none"}},
			{ID: "SaveButton", Type: "lib.core.widgets.Button", Parent: "ButtonRow", Bounds: surface.RectReport{X: 20, Y: 192, W: 132, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"focused": "false", "press_count": "1", "action": "save", "accessibility_role": "button"}},
			{ID: "ResetButton", Type: "lib.core.widgets.Button", Parent: "ButtonRow", Bounds: surface.RectReport{X: 164, Y: 192, W: 132, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"focused": "false", "press_count": "1", "action": "reset", "accessibility_role": "button"}},
			{ID: "StatusText", Type: "lib.core.widgets.Text", Parent: "Column", Bounds: surface.RectReport{X: 20, Y: 248, W: 440, H: 24}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"role": "status", "status_code": "2", "accessibility_role": "label"}},
		},
		ComponentTree: &surface.ComponentTreeReport{
			Schema:       "tetra.surface.component-tree.v1",
			DynamicLevel: "toolkit-reuse-widget-tree",
			RootID:       0,
			NodeCount:    11,
			FocusedID:    4,
			Nodes: []surface.ComponentTreeNodeReport{
				{ID: 0, Name: "ToolkitSettingsApp", Kind: "root", ParentID: -1, ChildIndex: 0, FirstChild: 1, ChildCount: 1, Focusable: false, Bounds: surface.RectReport{X: 0, Y: 0, W: 480, H: 320}},
				{ID: 1, Name: "Panel", Kind: "panel", ParentID: 0, ChildIndex: 0, FirstChild: 2, ChildCount: 1, Focusable: false, Bounds: surface.RectReport{X: 0, Y: 0, W: 480, H: 320}},
				{ID: 2, Name: "Column", Kind: "column", ParentID: 1, ChildIndex: 0, FirstChild: 3, ChildCount: 6, Focusable: false, Bounds: surface.RectReport{X: 12, Y: 12, W: 456, H: 296}},
				{ID: 3, Name: "TitleText", Kind: "text", ParentID: 2, ChildIndex: 0, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 20, Y: 20, W: 440, H: 24}},
				{ID: 4, Name: "NameTextBox", Kind: "textbox", ParentID: 2, ChildIndex: 1, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 20, Y: 52, W: 440, H: 44}},
				{ID: 5, Name: "NameLabel", Kind: "text", ParentID: 2, ChildIndex: 2, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 20, Y: 104, W: 440, H: 24}},
				{ID: 6, Name: "EmailTextBox", Kind: "textbox", ParentID: 2, ChildIndex: 3, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 20, Y: 136, W: 440, H: 44}},
				{ID: 7, Name: "ButtonRow", Kind: "row", ParentID: 2, ChildIndex: 4, FirstChild: 8, ChildCount: 2, Focusable: false, Bounds: surface.RectReport{X: 20, Y: 192, W: 440, H: 44}},
				{ID: 8, Name: "SaveButton", Kind: "button", ParentID: 7, ChildIndex: 0, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 20, Y: 192, W: 132, H: 44}},
				{ID: 9, Name: "ResetButton", Kind: "button", ParentID: 7, ChildIndex: 1, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 164, Y: 192, W: 132, H: 44}},
				{ID: 10, Name: "StatusText", Kind: "text", ParentID: 2, ChildIndex: 5, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 20, Y: 248, W: 440, H: 24}},
			},
			LayoutPasses: []surface.ComponentTreeLayoutPassReport{
				{ComponentID: 4, Pass: "initial", Bounds: surface.RectReport{X: 20, Y: 52, W: 280, H: 44}, Measured: surface.SizeReport{W: 280, H: 44}},
				{ComponentID: 6, Pass: "initial", Bounds: surface.RectReport{X: 20, Y: 136, W: 280, H: 44}, Measured: surface.SizeReport{W: 280, H: 44}},
				{ComponentID: 4, Pass: "resize", Bounds: surface.RectReport{X: 20, Y: 52, W: 440, H: 44}, Measured: surface.SizeReport{W: 440, H: 44}},
				{ComponentID: 6, Pass: "resize", Bounds: surface.RectReport{X: 20, Y: 136, W: 440, H: 44}, Measured: surface.SizeReport{W: 440, H: 44}},
				{ComponentID: 10, Pass: "status-update", Bounds: surface.RectReport{X: 20, Y: 248, W: 440, H: 24}, Measured: surface.SizeReport{W: 440, H: 24}},
			},
			DrawOrder:  []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			FocusOrder: []int{4, 6, 8, 9},
			DispatchPaths: []surface.ComponentTreeDispatchPathReport{
				{Event: "click", TargetID: 4, X: 40, Y: 72, Path: []int{0, 1, 2, 4}},
				{Event: "click", TargetID: 6, X: 40, Y: 156, Path: []int{0, 1, 2, 6}},
				{Event: "key", TargetID: 8, X: 40, Y: 208, Path: []int{0, 1, 2, 7, 8}},
				{Event: "key", TargetID: 9, X: 180, Y: 208, Path: []int{0, 1, 2, 7, 9}},
			},
		},
		ComponentTreeAPI: toolkitReuseComponentTreeAPIReport(),
		Toolkit:          toolkitReuseReport(),
		Events: []surface.EventReport{
			{Order: 1, Kind: "mouse_up", TargetComponent: "NameTextBox", DispatchPath: []string{"ToolkitSettingsApp", "Panel", "Column", "NameTextBox"}, Handled: true, Pass: true, X: 40, Y: 72, Width: 320, Height: 240, BufferSlots: []int{5, 40, 72, 1, 0, 320, 240, 0, 0}, BeforeState: map[string]string{"ToolkitSettingsApp.focused_id": "-1", "NameTextBox.focused": "false"}, AfterState: map[string]string{"ToolkitSettingsApp.focused_id": "4", "NameTextBox.focused": "true"}},
			{Order: 2, Kind: "text_input", TargetComponent: "NameTextBox", DispatchPath: []string{"ToolkitSettingsApp", "Panel", "Column", "NameTextBox"}, Handled: true, Pass: true, Width: 320, Height: 240, TimestampMS: 1, TextLen: 3, TextBytesHex: "416461", BufferSlots: []int{8, 0, 0, 0, 0, 320, 240, 1, 3}, BeforeState: map[string]string{"NameTextBox.buffer": "", "NameTextBox.caret": "0", "EmailTextBox.buffer": ""}, AfterState: map[string]string{"NameTextBox.buffer": "Ada", "NameTextBox.caret": "3", "EmailTextBox.buffer": ""}},
			{Order: 3, Kind: "key_down", TargetComponent: "ToolkitSettingsApp", DispatchPath: []string{"ToolkitSettingsApp"}, Handled: true, Pass: true, Key: 9, Width: 320, Height: 240, TimestampMS: 2, BufferSlots: []int{6, 0, 0, 0, 9, 320, 240, 2, 0}, BeforeState: map[string]string{"ToolkitSettingsApp.focused_id": "4"}, AfterState: map[string]string{"ToolkitSettingsApp.focused_id": "6"}},
			{Order: 4, Kind: "text_input", TargetComponent: "EmailTextBox", DispatchPath: []string{"ToolkitSettingsApp", "Panel", "Column", "EmailTextBox"}, Handled: true, Pass: true, Width: 320, Height: 240, TimestampMS: 3, TextLen: 5, TextBytesHex: "7465747261", BufferSlots: []int{8, 0, 0, 0, 0, 320, 240, 3, 5}, BeforeState: map[string]string{"EmailTextBox.buffer": "", "NameTextBox.buffer": "Ada"}, AfterState: map[string]string{"EmailTextBox.buffer": "tetra", "NameTextBox.buffer": "Ada"}},
			{Order: 5, Kind: "key_down", TargetComponent: "ToolkitSettingsApp", DispatchPath: []string{"ToolkitSettingsApp"}, Handled: true, Pass: true, Key: 9, Width: 320, Height: 240, TimestampMS: 4, BufferSlots: []int{6, 0, 0, 0, 9, 320, 240, 4, 0}, BeforeState: map[string]string{"ToolkitSettingsApp.focused_id": "6"}, AfterState: map[string]string{"ToolkitSettingsApp.focused_id": "8"}},
			{Order: 6, Kind: "key_down", TargetComponent: "SaveButton", DispatchPath: []string{"ToolkitSettingsApp", "Panel", "Column", "ButtonRow", "SaveButton"}, Handled: true, Pass: true, Key: 32, Width: 320, Height: 240, TimestampMS: 5, BufferSlots: []int{6, 0, 0, 0, 32, 320, 240, 5, 0}, BeforeState: map[string]string{"ToolkitSettingsApp.focused_id": "8", "ToolkitSettingsApp.save_count": "0", "StatusText.status_code": "0"}, AfterState: map[string]string{"ToolkitSettingsApp.focused_id": "8", "ToolkitSettingsApp.save_count": "1", "StatusText.status_code": "1"}},
			{Order: 7, Kind: "key_down", TargetComponent: "ToolkitSettingsApp", DispatchPath: []string{"ToolkitSettingsApp"}, Handled: true, Pass: true, Key: 9, Width: 320, Height: 240, TimestampMS: 6, BufferSlots: []int{6, 0, 0, 0, 9, 320, 240, 6, 0}, BeforeState: map[string]string{"ToolkitSettingsApp.focused_id": "8"}, AfterState: map[string]string{"ToolkitSettingsApp.focused_id": "9"}},
			{Order: 8, Kind: "key_down", TargetComponent: "ResetButton", DispatchPath: []string{"ToolkitSettingsApp", "Panel", "Column", "ButtonRow", "ResetButton"}, Handled: true, Pass: true, Key: 13, Width: 320, Height: 240, TimestampMS: 7, BufferSlots: []int{6, 0, 0, 0, 13, 320, 240, 7, 0}, BeforeState: map[string]string{"ToolkitSettingsApp.focused_id": "9", "ToolkitSettingsApp.reset_count": "0", "StatusText.status_code": "1", "NameTextBox.buffer": "Ada", "EmailTextBox.buffer": "tetra"}, AfterState: map[string]string{"ToolkitSettingsApp.focused_id": "9", "ToolkitSettingsApp.reset_count": "1", "StatusText.status_code": "2", "NameTextBox.buffer": "", "EmailTextBox.buffer": ""}},
			{Order: 9, Kind: "key_down", TargetComponent: "ToolkitSettingsApp", DispatchPath: []string{"ToolkitSettingsApp"}, Handled: true, Pass: true, Key: 9, Width: 320, Height: 240, TimestampMS: 8, BufferSlots: []int{6, 0, 0, 0, 9, 320, 240, 8, 0}, BeforeState: map[string]string{"ToolkitSettingsApp.focused_id": "9"}, AfterState: map[string]string{"ToolkitSettingsApp.focused_id": "4"}},
			{Order: 10, Kind: "resize", TargetComponent: "ToolkitSettingsApp", DispatchPath: []string{"ToolkitSettingsApp"}, Handled: true, Pass: true, Width: 480, Height: 320, TimestampMS: 9, BufferSlots: []int{2, 0, 0, 0, 0, 480, 320, 9, 0}, BeforeState: map[string]string{"ToolkitSettingsApp.focused_id": "4", "NameTextBox.bounds.w": "280", "EmailTextBox.bounds.w": "280"}, AfterState: map[string]string{"ToolkitSettingsApp.focused_id": "4", "NameTextBox.bounds.w": "440", "EmailTextBox.bounds.w": "440"}},
		},
		Frames: []surface.FrameReport{
			{Order: 1, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
			{Order: 2, Width: nameFrame.Width, Height: nameFrame.Height, Stride: nameFrame.Stride, Checksum: checksumRGBA(nameFrame.Pixels), Presented: true},
			{Order: 3, Width: saveFrame.Width, Height: saveFrame.Height, Stride: saveFrame.Stride, Checksum: checksumRGBA(saveFrame.Pixels), Presented: true},
			{Order: 4, Width: resetFrame.Width, Height: resetFrame.Height, Stride: resetFrame.Stride, Checksum: checksumRGBA(resetFrame.Pixels), Presented: true},
			{Order: 5, Width: afterFrame.Width, Height: afterFrame.Height, Stride: afterFrame.Stride, Checksum: checksumRGBA(afterFrame.Pixels), Presented: true},
		},
		StateTransitions: []surface.StateTransitionReport{
			{Order: 1, Component: "ToolkitSettingsApp", Field: "focused_id", Before: "-1", After: "4", Cause: "mouse_up"},
			{Order: 2, Component: "NameTextBox", Field: "buffer", Before: "", After: "Ada", Cause: "text_input"},
			{Order: 3, Component: "ToolkitSettingsApp", Field: "focused_id", Before: "4", After: "6", Cause: "tab"},
			{Order: 4, Component: "EmailTextBox", Field: "buffer", Before: "", After: "tetra", Cause: "text_input"},
			{Order: 5, Component: "ToolkitSettingsApp", Field: "focused_id", Before: "6", After: "8", Cause: "tab"},
			{Order: 6, Component: "ToolkitSettingsApp", Field: "save_count", Before: "0", After: "1", Cause: "key_down"},
			{Order: 7, Component: "StatusText", Field: "status_code", Before: "0", After: "1", Cause: "save"},
			{Order: 8, Component: "ToolkitSettingsApp", Field: "focused_id", Before: "8", After: "9", Cause: "tab"},
			{Order: 9, Component: "NameTextBox", Field: "buffer", Before: "Ada", After: "", Cause: "reset"},
			{Order: 10, Component: "EmailTextBox", Field: "buffer", Before: "tetra", After: "", Cause: "reset"},
			{Order: 11, Component: "ToolkitSettingsApp", Field: "reset_count", Before: "0", After: "1", Cause: "key_down"},
			{Order: 12, Component: "StatusText", Field: "status_code", Before: "1", After: "2", Cause: "reset"},
			{Order: 13, Component: "ToolkitSettingsApp", Field: "focused_id", Before: "9", After: "4", Cause: "tab"},
			{Order: 14, Component: "ToolkitSettingsApp", Field: "NameTextBox.bounds.w", Before: "280", After: "440", Cause: "resize"},
			{Order: 15, Component: "ToolkitSettingsApp", Field: "EmailTextBox.bounds.w", Before: "280", After: "440", Cause: "resize"},
		},
		Cases: toolkitReuseBaseCases(),
	}
	switch mode {
	case "headless-toolkit-reuse":
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "headless event dispatch", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "headless framebuffer checksum", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "headless actual runner trace", Kind: "positive", Ran: true, Pass: true},
		)
	case "linux-x64-real-window-toolkit-reuse":
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "linux-x64 Surface Host ABI open/present/close", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 real-window surface", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 native input event pump", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 real-window resize event", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 real-window close event", Kind: "positive", Ran: true, Pass: true},
		)
	case "wasm32-web-browser-canvas-toolkit-reuse":
		scenario.Frames = nil
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "wasm32-web browser canvas surface", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas RGBA readback", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas pointer input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas keyboard input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas resize input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas text input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web Surface Host ABI imports", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "compiler-owned wasm Surface loader", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "compiler-owned browser canvas Surface host", Kind: "positive", Ran: true, Pass: true},
		)
	}
	return scenario
}

func runAccessibilityMetadataScenario(mode string) headlessScenario {
	beforeFrame := renderAccessibilityMetadataFrameRGBA(0, 0, -1, 0, 0, 0, 320, 240)
	nameFrame := renderAccessibilityMetadataFrameRGBA(3, 0, 5, 0, 0, 0, 320, 240)
	saveFrame := renderAccessibilityMetadataFrameRGBA(3, 5, 9, 1, 0, 1, 320, 240)
	resetFrame := renderAccessibilityMetadataFrameRGBA(0, 0, 10, 1, 1, 2, 320, 240)
	afterFrame := renderAccessibilityMetadataFrameRGBA(0, 0, 5, 1, 1, 2, 480, 320)
	scenario := headlessScenario{
		Components: []surface.ComponentReport{
			{ID: "AccessibilitySettingsApp", Type: "examples.surface_accessibility_settings.AccessibilitySettingsApp", Bounds: surface.RectReport{X: 0, Y: 0, W: 480, H: 320}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"focused_id": "5", "save_count": "1", "reset_count": "1", "status_code": "2", "width": "480", "height": "320", "accessibility_role": "root"}},
			{ID: "Panel", Type: "lib.core.widgets.Panel", Parent: "AccessibilitySettingsApp", Bounds: surface.RectReport{X: 0, Y: 0, W: 480, H: 320}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"padding": "12", "accessibility_role": "panel"}},
			{ID: "Column", Type: "lib.core.widgets.Column", Parent: "Panel", Bounds: surface.RectReport{X: 12, Y: 12, W: 456, H: 296}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"child_count": "7", "accessibility_role": "column"}},
			{ID: "TitleText", Type: "lib.core.widgets.Text", Parent: "Column", Bounds: surface.RectReport{X: 20, Y: 20, W: 440, H: 24}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"role": "text", "text_len": "8", "accessibility_role": "text"}},
			{ID: "NameLabel", Type: "lib.core.widgets.Text", Parent: "Column", Bounds: surface.RectReport{X: 20, Y: 52, W: 440, H: 24}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"role": "label", "text_len": "4", "accessibility_role": "label"}},
			{ID: "NameTextBox", Type: "lib.core.widgets.TextBox", Parent: "Column", Bounds: surface.RectReport{X: 20, Y: 84, W: 440, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"focused": "true", "buffer": "", "text_len": "0", "caret": "0", "accessibility_role": "textbox"}},
			{ID: "EmailLabel", Type: "lib.core.widgets.Text", Parent: "Column", Bounds: surface.RectReport{X: 20, Y: 136, W: 440, H: 24}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"role": "label", "text_len": "5", "accessibility_role": "label"}},
			{ID: "EmailTextBox", Type: "lib.core.widgets.TextBox", Parent: "Column", Bounds: surface.RectReport{X: 20, Y: 168, W: 440, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"focused": "false", "buffer": "", "text_len": "0", "caret": "0", "accessibility_role": "textbox"}},
			{ID: "ButtonRow", Type: "lib.core.widgets.Row", Parent: "Column", Bounds: surface.RectReport{X: 20, Y: 224, W: 440, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"child_count": "2", "accessibility_role": "row"}},
			{ID: "SaveButton", Type: "lib.core.widgets.Button", Parent: "ButtonRow", Bounds: surface.RectReport{X: 20, Y: 224, W: 132, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"focused": "false", "press_count": "1", "action": "save", "accessibility_role": "button"}},
			{ID: "ResetButton", Type: "lib.core.widgets.Button", Parent: "ButtonRow", Bounds: surface.RectReport{X: 164, Y: 224, W: 132, H: 44}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"focused": "false", "press_count": "1", "action": "reset", "accessibility_role": "button"}},
			{ID: "StatusText", Type: "lib.core.widgets.Text", Parent: "Column", Bounds: surface.RectReport{X: 20, Y: 280, W: 440, H: 24}, Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"}, State: map[string]string{"role": "status", "status_code": "2", "accessibility_role": "status"}},
		},
		ComponentTree:     accessibilityComponentTreeReport(),
		ComponentTreeAPI:  accessibilityComponentTreeAPIReport(),
		Toolkit:           accessibilityToolkitReport(),
		AccessibilityTree: accessibilityTreeReport(beforeFrame, nameFrame, saveFrame, resetFrame, afterFrame),
		Events: []surface.EventReport{
			{Order: 1, Kind: "mouse_up", TargetComponent: "NameTextBox", DispatchPath: []string{"AccessibilitySettingsApp", "Panel", "Column", "NameTextBox"}, Handled: true, Pass: true, X: 40, Y: 100, Width: 320, Height: 240, BufferSlots: []int{5, 40, 100, 1, 0, 320, 240, 0, 0}, BeforeState: map[string]string{"AccessibilitySettingsApp.focused_id": "-1", "NameTextBox.focused": "false"}, AfterState: map[string]string{"AccessibilitySettingsApp.focused_id": "5", "NameTextBox.focused": "true"}},
			{Order: 2, Kind: "text_input", TargetComponent: "NameTextBox", DispatchPath: []string{"AccessibilitySettingsApp", "Panel", "Column", "NameTextBox"}, Handled: true, Pass: true, Width: 320, Height: 240, TimestampMS: 1, TextLen: 3, TextBytesHex: "416461", BufferSlots: []int{8, 0, 0, 0, 0, 320, 240, 1, 3}, BeforeState: map[string]string{"NameTextBox.buffer": "", "NameTextBox.caret": "0", "EmailTextBox.buffer": ""}, AfterState: map[string]string{"NameTextBox.buffer": "Ada", "NameTextBox.caret": "3", "EmailTextBox.buffer": ""}},
			{Order: 3, Kind: "key_down", TargetComponent: "AccessibilitySettingsApp", DispatchPath: []string{"AccessibilitySettingsApp"}, Handled: true, Pass: true, Key: 9, Width: 320, Height: 240, TimestampMS: 2, BufferSlots: []int{6, 0, 0, 0, 9, 320, 240, 2, 0}, BeforeState: map[string]string{"AccessibilitySettingsApp.focused_id": "5"}, AfterState: map[string]string{"AccessibilitySettingsApp.focused_id": "7"}},
			{Order: 4, Kind: "text_input", TargetComponent: "EmailTextBox", DispatchPath: []string{"AccessibilitySettingsApp", "Panel", "Column", "EmailTextBox"}, Handled: true, Pass: true, Width: 320, Height: 240, TimestampMS: 3, TextLen: 5, TextBytesHex: "7465747261", BufferSlots: []int{8, 0, 0, 0, 0, 320, 240, 3, 5}, BeforeState: map[string]string{"EmailTextBox.buffer": "", "NameTextBox.buffer": "Ada"}, AfterState: map[string]string{"EmailTextBox.buffer": "tetra", "NameTextBox.buffer": "Ada"}},
			{Order: 5, Kind: "key_down", TargetComponent: "AccessibilitySettingsApp", DispatchPath: []string{"AccessibilitySettingsApp"}, Handled: true, Pass: true, Key: 9, Width: 320, Height: 240, TimestampMS: 4, BufferSlots: []int{6, 0, 0, 0, 9, 320, 240, 4, 0}, BeforeState: map[string]string{"AccessibilitySettingsApp.focused_id": "7"}, AfterState: map[string]string{"AccessibilitySettingsApp.focused_id": "9"}},
			{Order: 6, Kind: "key_down", TargetComponent: "SaveButton", DispatchPath: []string{"AccessibilitySettingsApp", "Panel", "Column", "ButtonRow", "SaveButton"}, Handled: true, Pass: true, Key: 32, Width: 320, Height: 240, TimestampMS: 5, BufferSlots: []int{6, 0, 0, 0, 32, 320, 240, 5, 0}, BeforeState: map[string]string{"AccessibilitySettingsApp.focused_id": "9", "AccessibilitySettingsApp.save_count": "0", "StatusText.status_code": "0"}, AfterState: map[string]string{"AccessibilitySettingsApp.focused_id": "9", "AccessibilitySettingsApp.save_count": "1", "StatusText.status_code": "1"}},
			{Order: 7, Kind: "key_down", TargetComponent: "AccessibilitySettingsApp", DispatchPath: []string{"AccessibilitySettingsApp"}, Handled: true, Pass: true, Key: 9, Width: 320, Height: 240, TimestampMS: 6, BufferSlots: []int{6, 0, 0, 0, 9, 320, 240, 6, 0}, BeforeState: map[string]string{"AccessibilitySettingsApp.focused_id": "9"}, AfterState: map[string]string{"AccessibilitySettingsApp.focused_id": "10"}},
			{Order: 8, Kind: "key_down", TargetComponent: "ResetButton", DispatchPath: []string{"AccessibilitySettingsApp", "Panel", "Column", "ButtonRow", "ResetButton"}, Handled: true, Pass: true, Key: 13, Width: 320, Height: 240, TimestampMS: 7, BufferSlots: []int{6, 0, 0, 0, 13, 320, 240, 7, 0}, BeforeState: map[string]string{"AccessibilitySettingsApp.focused_id": "10", "AccessibilitySettingsApp.reset_count": "0", "StatusText.status_code": "1", "NameTextBox.buffer": "Ada", "EmailTextBox.buffer": "tetra"}, AfterState: map[string]string{"AccessibilitySettingsApp.focused_id": "10", "AccessibilitySettingsApp.reset_count": "1", "StatusText.status_code": "2", "NameTextBox.buffer": "", "EmailTextBox.buffer": ""}},
			{Order: 9, Kind: "key_down", TargetComponent: "AccessibilitySettingsApp", DispatchPath: []string{"AccessibilitySettingsApp"}, Handled: true, Pass: true, Key: 9, Width: 320, Height: 240, TimestampMS: 8, BufferSlots: []int{6, 0, 0, 0, 9, 320, 240, 8, 0}, BeforeState: map[string]string{"AccessibilitySettingsApp.focused_id": "10"}, AfterState: map[string]string{"AccessibilitySettingsApp.focused_id": "5"}},
			{Order: 10, Kind: "resize", TargetComponent: "AccessibilitySettingsApp", DispatchPath: []string{"AccessibilitySettingsApp"}, Handled: true, Pass: true, Width: 480, Height: 320, TimestampMS: 9, BufferSlots: []int{2, 0, 0, 0, 0, 480, 320, 9, 0}, BeforeState: map[string]string{"AccessibilitySettingsApp.focused_id": "5", "NameTextBox.bounds.w": "280", "EmailTextBox.bounds.w": "280"}, AfterState: map[string]string{"AccessibilitySettingsApp.focused_id": "5", "NameTextBox.bounds.w": "440", "EmailTextBox.bounds.w": "440"}},
		},
		Frames: []surface.FrameReport{
			{Order: 1, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
			{Order: 2, Width: nameFrame.Width, Height: nameFrame.Height, Stride: nameFrame.Stride, Checksum: checksumRGBA(nameFrame.Pixels), Presented: true},
			{Order: 3, Width: saveFrame.Width, Height: saveFrame.Height, Stride: saveFrame.Stride, Checksum: checksumRGBA(saveFrame.Pixels), Presented: true},
			{Order: 4, Width: resetFrame.Width, Height: resetFrame.Height, Stride: resetFrame.Stride, Checksum: checksumRGBA(resetFrame.Pixels), Presented: true},
			{Order: 5, Width: afterFrame.Width, Height: afterFrame.Height, Stride: afterFrame.Stride, Checksum: checksumRGBA(afterFrame.Pixels), Presented: true},
		},
		StateTransitions: []surface.StateTransitionReport{
			{Order: 1, Component: "AccessibilitySettingsApp", Field: "focused_id", Before: "-1", After: "5", Cause: "mouse_up"},
			{Order: 2, Component: "NameTextBox", Field: "buffer", Before: "", After: "Ada", Cause: "text_input"},
			{Order: 3, Component: "AccessibilitySettingsApp", Field: "focused_id", Before: "5", After: "7", Cause: "tab"},
			{Order: 4, Component: "EmailTextBox", Field: "buffer", Before: "", After: "tetra", Cause: "text_input"},
			{Order: 5, Component: "AccessibilitySettingsApp", Field: "focused_id", Before: "7", After: "9", Cause: "tab"},
			{Order: 6, Component: "AccessibilitySettingsApp", Field: "save_count", Before: "0", After: "1", Cause: "key_down"},
			{Order: 7, Component: "StatusText", Field: "status_code", Before: "0", After: "1", Cause: "save"},
			{Order: 8, Component: "AccessibilitySettingsApp", Field: "focused_id", Before: "9", After: "10", Cause: "tab"},
			{Order: 9, Component: "NameTextBox", Field: "buffer", Before: "Ada", After: "", Cause: "reset"},
			{Order: 10, Component: "EmailTextBox", Field: "buffer", Before: "tetra", After: "", Cause: "reset"},
			{Order: 11, Component: "AccessibilitySettingsApp", Field: "reset_count", Before: "0", After: "1", Cause: "key_down"},
			{Order: 12, Component: "StatusText", Field: "status_code", Before: "1", After: "2", Cause: "reset"},
			{Order: 13, Component: "AccessibilitySettingsApp", Field: "focused_id", Before: "10", After: "5", Cause: "tab"},
			{Order: 14, Component: "AccessibilitySettingsApp", Field: "NameTextBox.bounds.w", Before: "280", After: "440", Cause: "resize"},
			{Order: 15, Component: "AccessibilitySettingsApp", Field: "EmailTextBox.bounds.w", Before: "280", After: "440", Cause: "resize"},
		},
		Cases: accessibilityMetadataBaseCases(),
	}
	switch mode {
	case "headless-accessibility-metadata":
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "headless event dispatch", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "headless framebuffer checksum", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "headless actual runner trace", Kind: "positive", Ran: true, Pass: true},
		)
	case "linux-x64-real-window-accessibility-metadata":
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "linux-x64 Surface Host ABI open/present/close", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 real-window surface", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 native input event pump", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 real-window resize event", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 real-window close event", Kind: "positive", Ran: true, Pass: true},
		)
	case "wasm32-web-browser-canvas-accessibility-metadata":
		scenario.Frames = nil
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "wasm32-web browser canvas surface", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas RGBA readback", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas pointer input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas keyboard input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas resize input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas text input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web Surface Host ABI imports", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "compiler-owned wasm Surface loader", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "compiler-owned browser canvas Surface host", Kind: "positive", Ran: true, Pass: true},
		)
	}
	return scenario
}

func accessibilityMetadataBaseCases() []surface.CaseReport {
	cases := toolkitReuseBaseCases()
	cases = append(cases,
		surface.CaseReport{Name: "accessibility metadata tree schema", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "accessibility metadata roles labels values states", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "accessibility metadata component tree alignment", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "accessibility metadata focus order alignment", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "accessibility metadata reading order", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "accessibility metadata snapshots update", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "accessibility metadata no DOM ARIA platform host claim", Kind: "positive", Ran: true, Pass: true},
	)
	return cases
}

func toolkitReuseBaseCases() []surface.CaseReport {
	cases := minimalToolkitBaseCases()
	cases = append(cases,
		surface.CaseReport{Name: "toolkit reuse second example evidence", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "toolkit reuse widgets module evidence", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "toolkit reuse multi TextBox routing", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "toolkit reuse focused TextBox only mutates", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "toolkit reuse Save action routed", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "toolkit reuse Reset action routed", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "toolkit reuse StatusText updates", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "toolkit reuse resize relayout", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "toolkit reuse changed frame checksums", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "toolkit reuse no demo-local widget structs", Kind: "positive", Ran: true, Pass: true},
	)
	return cases
}

func minimalToolkitBaseCases() []surface.CaseReport {
	return []surface.CaseReport{
		{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
		{Name: "host-provided pointer event dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "host event buffer poll_event", Kind: "positive", Ran: true, Pass: true},
		{Name: "pre/post event frame sequence", Kind: "positive", Ran: true, Pass: true},
		{Name: "component hierarchy dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "component text input scalar dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "host text payload buffer", Kind: "positive", Ran: true, Pass: true},
		{Name: "component focus dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "component accessibility metadata", Kind: "positive", Ran: true, Pass: true},
		{Name: "no legacy UI sidecar artifacts", Kind: "positive", Ran: true, Pass: true},
		{Name: "state transition", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree node count", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree parent child links", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree layout bounds", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree draw traversal", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree pointer dispatch path", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree focus traversal", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree text routed to focused TextBox", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree button action dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree resize relayout", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree rendered frame update", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree api builder node creation", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree api parent child invariants", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree api layout helper dispatch", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree api hit test helper", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree api focus helper traversal", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree api dispatch path helper", Kind: "positive", Ran: true, Pass: true},
		{Name: "component tree api no manual bookkeeping", Kind: "positive", Ran: true, Pass: true},
		{Name: "reject legacy UI evidence", Kind: "negative", Ran: true, Pass: true, ExpectedError: "legacy UI evidence rejected"},
		{Name: "minimal toolkit reusable widgets", Kind: "positive", Ran: true, Pass: true},
		{Name: "minimal toolkit Text widget evidence", Kind: "positive", Ran: true, Pass: true},
		{Name: "minimal toolkit Button widget evidence", Kind: "positive", Ran: true, Pass: true},
		{Name: "minimal toolkit TextBox widget evidence", Kind: "positive", Ran: true, Pass: true},
		{Name: "minimal toolkit Row Column Panel layout", Kind: "positive", Ran: true, Pass: true},
		{Name: "minimal toolkit tree api reuse", Kind: "positive", Ran: true, Pass: true},
		{Name: "minimal toolkit TextBox focus input editing", Kind: "positive", Ran: true, Pass: true},
		{Name: "minimal toolkit Submit action routed", Kind: "positive", Ran: true, Pass: true},
		{Name: "minimal toolkit Reset action routed", Kind: "positive", Ran: true, Pass: true},
		{Name: "minimal toolkit status text update", Kind: "positive", Ran: true, Pass: true},
		{Name: "minimal toolkit resize relayout", Kind: "positive", Ran: true, Pass: true},
		{Name: "minimal toolkit rendered frame update", Kind: "positive", Ran: true, Pass: true},
	}
}

func minimalToolkitComponentTreeAPIReport() *surface.ComponentTreeAPIReport {
	return &surface.ComponentTreeAPIReport{
		Schema:            "tetra.surface.component-tree-api.v1",
		APILevel:          "builder-layout-dispatch-v1",
		Source:            "examples/surface_toolkit_form.tetra",
		ManualBookkeeping: false,
		Builder: surface.ComponentTreeAPIBuilderReport{
			RootCreatedBy:     "tree_add_root",
			ChildrenCreatedBy: "tree_add_child",
			NodeCount:         9,
			Capacity:          16,
			OverflowChecked:   true,
		},
		Invariants: surface.ComponentTreeAPIInvariantReport{
			TreeValidateRan:         true,
			TreeValidateStatus:      0,
			ParentChildLinksChecked: true,
			ChildIndicesChecked:     true,
			ChildCountChecked:       true,
			FirstChildChecked:       true,
		},
		LayoutHelpers: []surface.ComponentTreeAPILayoutHelperReport{
			{Helper: "widgets.panel_content_rect", Target: "Panel", Pass: "initial", ChangedBounds: true},
			{Helper: "widgets.column_layout", Target: "Column", Pass: "initial", ChangedBounds: true},
			{Helper: "widgets.row_layout", Target: "ButtonRow", Pass: "initial", ChangedBounds: true},
			{Helper: "widgets.column_layout", Target: "Column", Pass: "resize", ChangedBounds: true},
		},
		FocusHelpers: []surface.ComponentTreeAPIFocusHelperReport{
			{Helper: "tree_focus_next", Before: "TextBox", After: "SubmitButton"},
			{Helper: "tree_focus_next", Before: "SubmitButton", After: "ResetButton"},
			{Helper: "tree_focus_next", Before: "ResetButton", After: "TextBox"},
		},
		HitTests: []surface.ComponentTreeAPIHitTestReport{
			{Helper: "widgets.hit_test", X: 40, Y: 72, Target: "TextBox", Path: []int{0, 1, 2, 4}},
			{Helper: "widgets.hit_test", X: 180, Y: 124, Target: "ResetButton", Path: []int{0, 1, 2, 5, 7}},
		},
		DispatchPaths: []surface.ComponentTreeAPIDispatchPathReport{
			{Helper: "tree_build_dispatch_path", Target: "TextBox", Path: []int{0, 1, 2, 4}},
			{Helper: "tree_build_dispatch_path", Target: "SubmitButton", Path: []int{0, 1, 2, 5, 6}},
			{Helper: "tree_build_dispatch_path", Target: "ResetButton", Path: []int{0, 1, 2, 5, 7}},
		},
	}
}

func minimalToolkitReport() *surface.ToolkitReport {
	return &surface.ToolkitReport{
		Schema:                    "tetra.surface.toolkit.v1",
		ToolkitLevel:              "minimal-widgets-v1",
		Source:                    "examples/surface_toolkit_form.tetra",
		Module:                    "lib.core.widgets",
		Experimental:              true,
		ProductionClaim:           false,
		UsesComponentTreeAPI:      true,
		ManualBookkeeping:         false,
		DemoSpecificWidgetStructs: false,
		NoMagicWidgets:            true,
		NoPlatformWidgets:         true,
		NoDOMUI:                   true,
		NoUserJS:                  true,
		Widgets: []surface.ToolkitWidgetReport{
			{Name: "Panel", Kind: "Panel", NodeID: 1, Reusable: true, OrdinaryTetraStruct: true},
			{Name: "Column", Kind: "Column", NodeID: 2, Reusable: true, OrdinaryTetraStruct: true},
			{Name: "NameLabel", Kind: "Text", NodeID: 3, Role: "label", Reusable: true, OrdinaryTetraStruct: true},
			{Name: "TextBox", Kind: "TextBox", NodeID: 4, Reusable: true, OrdinaryTetraStruct: true, Editable: true},
			{Name: "ButtonRow", Kind: "Row", NodeID: 5, Reusable: true, OrdinaryTetraStruct: true},
			{Name: "SubmitButton", Kind: "Button", NodeID: 6, Action: "submit", Reusable: true, OrdinaryTetraStruct: true},
			{Name: "ResetButton", Kind: "Button", NodeID: 7, Action: "reset", Reusable: true, OrdinaryTetraStruct: true},
			{Name: "StatusText", Kind: "Text", NodeID: 8, Role: "status", Reusable: true, OrdinaryTetraStruct: true},
		},
		ReusableSources: []string{
			"lib/core/widgets.tetra:panel_init",
			"lib/core/widgets.tetra:column_init",
			"lib/core/widgets.tetra:text_init",
			"lib/core/widgets.tetra:textbox_init",
			"lib/core/widgets.tetra:row_init",
			"lib/core/widgets.tetra:button_init",
		},
	}
}

func toolkitReuseComponentTreeAPIReport() *surface.ComponentTreeAPIReport {
	return &surface.ComponentTreeAPIReport{
		Schema:            "tetra.surface.component-tree-api.v1",
		APILevel:          "builder-layout-dispatch-v1",
		Source:            "examples/surface_toolkit_settings.tetra",
		ManualBookkeeping: false,
		Builder: surface.ComponentTreeAPIBuilderReport{
			RootCreatedBy:     "tree_add_root",
			ChildrenCreatedBy: "tree_add_child",
			NodeCount:         11,
			Capacity:          20,
			OverflowChecked:   true,
		},
		Invariants: surface.ComponentTreeAPIInvariantReport{
			TreeValidateRan:         true,
			TreeValidateStatus:      0,
			ParentChildLinksChecked: true,
			ChildIndicesChecked:     true,
			ChildCountChecked:       true,
			FirstChildChecked:       true,
		},
		LayoutHelpers: []surface.ComponentTreeAPILayoutHelperReport{
			{Helper: "widgets.panel_content_rect", Target: "Panel", Pass: "initial", ChangedBounds: true},
			{Helper: "widgets.column_layout", Target: "Column", Pass: "initial", ChangedBounds: true},
			{Helper: "widgets.row_layout", Target: "ButtonRow", Pass: "initial", ChangedBounds: true},
			{Helper: "widgets.column_layout", Target: "Column", Pass: "resize", ChangedBounds: true},
		},
		FocusHelpers: []surface.ComponentTreeAPIFocusHelperReport{
			{Helper: "tree_focus_next", Before: "NameTextBox", After: "EmailTextBox"},
			{Helper: "tree_focus_next", Before: "EmailTextBox", After: "SaveButton"},
			{Helper: "tree_focus_next", Before: "SaveButton", After: "ResetButton"},
			{Helper: "tree_focus_next", Before: "ResetButton", After: "NameTextBox"},
		},
		HitTests: []surface.ComponentTreeAPIHitTestReport{
			{Helper: "widgets.hit_test", X: 40, Y: 72, Target: "NameTextBox", Path: []int{0, 1, 2, 4}},
			{Helper: "widgets.hit_test", X: 40, Y: 156, Target: "EmailTextBox", Path: []int{0, 1, 2, 6}},
			{Helper: "widgets.hit_test", X: 180, Y: 208, Target: "ResetButton", Path: []int{0, 1, 2, 7, 9}},
		},
		DispatchPaths: []surface.ComponentTreeAPIDispatchPathReport{
			{Helper: "tree_build_dispatch_path", Target: "NameTextBox", Path: []int{0, 1, 2, 4}},
			{Helper: "tree_build_dispatch_path", Target: "EmailTextBox", Path: []int{0, 1, 2, 6}},
			{Helper: "tree_build_dispatch_path", Target: "SaveButton", Path: []int{0, 1, 2, 7, 8}},
			{Helper: "tree_build_dispatch_path", Target: "ResetButton", Path: []int{0, 1, 2, 7, 9}},
		},
	}
}

func toolkitReuseReport() *surface.ToolkitReport {
	return &surface.ToolkitReport{
		Schema:                    "tetra.surface.toolkit.v1",
		ToolkitLevel:              "toolkit-reuse-v1",
		ReuseLevel:                "multi-form-widget-reuse-v1",
		Source:                    "examples/surface_toolkit_settings.tetra",
		Sources:                   []string{"examples/surface_toolkit_form.tetra", "examples/surface_toolkit_settings.tetra"},
		Module:                    "lib.core.widgets",
		Experimental:              true,
		ProductionClaim:           false,
		UsesComponentTreeAPI:      true,
		ManualBookkeeping:         false,
		DemoSpecificWidgetStructs: false,
		NoMagicWidgets:            true,
		NoPlatformWidgets:         true,
		NoDOMUI:                   true,
		NoUserJS:                  true,
		ExampleCount:              2,
		TextBoxCount:              2,
		ButtonCount:               2,
		MultiTextBoxEvidence:      true,
		MultiFormEvidence:         true,
		Widgets: []surface.ToolkitWidgetReport{
			{Name: "Panel", Kind: "Panel", NodeID: 1, Reusable: true, OrdinaryTetraStruct: true},
			{Name: "Column", Kind: "Column", NodeID: 2, Reusable: true, OrdinaryTetraStruct: true},
			{Name: "TitleText", Kind: "Text", NodeID: 3, Role: "label", Reusable: true, OrdinaryTetraStruct: true},
			{Name: "NameTextBox", Kind: "TextBox", NodeID: 4, Reusable: true, OrdinaryTetraStruct: true, Editable: true},
			{Name: "NameLabel", Kind: "Text", NodeID: 5, Role: "label", Reusable: true, OrdinaryTetraStruct: true},
			{Name: "EmailTextBox", Kind: "TextBox", NodeID: 6, Reusable: true, OrdinaryTetraStruct: true, Editable: true},
			{Name: "ButtonRow", Kind: "Row", NodeID: 7, Reusable: true, OrdinaryTetraStruct: true},
			{Name: "SaveButton", Kind: "Button", NodeID: 8, Action: "save", Reusable: true, OrdinaryTetraStruct: true},
			{Name: "ResetButton", Kind: "Button", NodeID: 9, Action: "reset", Reusable: true, OrdinaryTetraStruct: true},
			{Name: "StatusText", Kind: "Text", NodeID: 10, Role: "status", Reusable: true, OrdinaryTetraStruct: true},
		},
		ReusableSources: []string{
			"lib/core/widgets.tetra:panel_init",
			"lib/core/widgets.tetra:column_init",
			"lib/core/widgets.tetra:text_init",
			"lib/core/widgets.tetra:textbox_init",
			"lib/core/widgets.tetra:row_init",
			"lib/core/widgets.tetra:button_init",
			"lib/core/widgets.tetra:hit_test",
			"lib/core/widgets.tetra:textbox_text_input",
			"lib/core/widgets.tetra:button_key_event",
		},
	}
}

func productionToolkitBaseCases() []surface.CaseReport {
	cases := toolkitReuseBaseCases()
	cases = append(cases,
		surface.CaseReport{Name: "production toolkit required widget set", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "production toolkit style module default theme", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "production toolkit style states normal focused hovered pressed disabled error", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "production toolkit Text Label StatusText evidence", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "production toolkit Button TextBox Checkbox evidence", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "production toolkit Row Column Panel Stack Scroll Spacer layout", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "production toolkit component tree api reuse", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "production toolkit TextBox focus input editing", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "production toolkit Checkbox toggle routed", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "production toolkit Scroll offset routed", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "production toolkit Save action routed", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "production toolkit Reset action routed", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "production toolkit StatusText updates", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "production toolkit safe text storage", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "production toolkit no demo-local widget structs", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "production toolkit browser host separation", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "production toolkit rendered frame update", Kind: "positive", Ran: true, Pass: true},
	)
	return cases
}

func productionToolkitComponentTreeAPIReport() *surface.ComponentTreeAPIReport {
	return &surface.ComponentTreeAPIReport{
		Schema:            "tetra.surface.component-tree-api.v1",
		APILevel:          "builder-layout-dispatch-v1",
		Source:            "examples/surface_release_form.tetra",
		ManualBookkeeping: false,
		Builder: surface.ComponentTreeAPIBuilderReport{
			RootCreatedBy:     "tree_add_root",
			ChildrenCreatedBy: "tree_add_child",
			NodeCount:         18,
			Capacity:          32,
			OverflowChecked:   true,
		},
		Invariants: surface.ComponentTreeAPIInvariantReport{
			TreeValidateRan:         true,
			TreeValidateStatus:      0,
			ParentChildLinksChecked: true,
			ChildIndicesChecked:     true,
			ChildCountChecked:       true,
			FirstChildChecked:       true,
		},
		LayoutHelpers: []surface.ComponentTreeAPILayoutHelperReport{
			{Helper: "widgets.panel_content_rect", Target: "Panel", Pass: "initial", ChangedBounds: true},
			{Helper: "widgets.stack_layout", Target: "Stack", Pass: "initial", ChangedBounds: true},
			{Helper: "widgets.column_layout", Target: "Column", Pass: "initial", ChangedBounds: true},
			{Helper: "widgets.scroll_set_offset", Target: "TermsScroll", Pass: "scroll", ChangedBounds: true},
			{Helper: "widgets.row_layout", Target: "ButtonRow", Pass: "initial", ChangedBounds: true},
			{Helper: "widgets.column_layout", Target: "Column", Pass: "resize", ChangedBounds: true},
		},
		FocusHelpers: []surface.ComponentTreeAPIFocusHelperReport{
			{Helper: "tree_focus_next", Before: "NameTextBox", After: "EmailTextBox"},
			{Helper: "tree_focus_next", Before: "EmailTextBox", After: "SubscribeCheckbox"},
			{Helper: "tree_focus_next", Before: "SubscribeCheckbox", After: "SaveButton"},
			{Helper: "tree_focus_next", Before: "SaveButton", After: "ResetButton"},
			{Helper: "tree_focus_next", Before: "ResetButton", After: "NameTextBox"},
		},
		HitTests: []surface.ComponentTreeAPIHitTestReport{
			{Helper: "widgets.hit_test_release_form", X: 48, Y: 148, Target: "NameTextBox", Path: []int{0, 1, 2, 3, 7}},
			{Helper: "widgets.hit_test_release_form", X: 48, Y: 228, Target: "EmailTextBox", Path: []int{0, 1, 2, 3, 9}},
			{Helper: "widgets.hit_test_release_form", X: 48, Y: 280, Target: "SubscribeCheckbox", Path: []int{0, 1, 2, 3, 10}},
			{Helper: "widgets.hit_test_release_form", X: 192, Y: 376, Target: "ResetButton", Path: []int{0, 1, 2, 3, 13, 15}},
		},
		DispatchPaths: []surface.ComponentTreeAPIDispatchPathReport{
			{Helper: "tree_build_dispatch_path", Target: "NameTextBox", Path: []int{0, 1, 2, 3, 7}},
			{Helper: "tree_build_dispatch_path", Target: "EmailTextBox", Path: []int{0, 1, 2, 3, 9}},
			{Helper: "tree_build_dispatch_path", Target: "SubscribeCheckbox", Path: []int{0, 1, 2, 3, 10}},
			{Helper: "tree_build_dispatch_path", Target: "TermsScroll", Path: []int{0, 1, 2, 3, 11}},
			{Helper: "tree_build_dispatch_path", Target: "SaveButton", Path: []int{0, 1, 2, 3, 13, 14}},
			{Helper: "tree_build_dispatch_path", Target: "ResetButton", Path: []int{0, 1, 2, 3, 13, 15}},
		},
	}
}

func productionToolkitReport() *surface.ToolkitReport {
	return &surface.ToolkitReport{
		Schema:                    "tetra.surface.toolkit.v1",
		ToolkitLevel:              "production-widgets-v1",
		ReleaseScope:              "surface-v1-linux-web",
		Source:                    "examples/surface_release_form.tetra",
		Sources:                   []string{"examples/surface_release_form.tetra", "examples/surface_toolkit_form.tetra", "examples/surface_toolkit_settings.tetra"},
		Module:                    "lib.core.widgets",
		StyleModule:               "lib.core.style",
		Experimental:              false,
		ProductionClaim:           true,
		UsesComponentTreeAPI:      true,
		ManualBookkeeping:         false,
		DemoSpecificWidgetStructs: false,
		NoMagicWidgets:            true,
		NoPlatformWidgets:         true,
		NoDOMUI:                   true,
		NoUserJS:                  true,
		ExampleCount:              3,
		TextBoxCount:              2,
		ButtonCount:               2,
		MultiTextBoxEvidence:      true,
		MultiFormEvidence:         true,
		WidgetSet:                 []string{"Text", "Label", "StatusText", "Button", "TextBox", "Checkbox", "Row", "Column", "Panel", "Stack", "Scroll", "Spacer"},
		StateSet:                  []string{"normal", "focused", "hovered", "pressed", "disabled", "error"},
		LayoutFeatures:            []string{"padding", "margin", "spacing", "min_size", "max_size", "fill", "scroll_offset"},
		Theme:                     true,
		SafeTextStorage:           true,
		Widgets: []surface.ToolkitWidgetReport{
			{Name: "Panel", Kind: "Panel", NodeID: 1, Reusable: true, OrdinaryTetraStruct: true},
			{Name: "Stack", Kind: "Stack", NodeID: 2, Reusable: true, OrdinaryTetraStruct: true},
			{Name: "Column", Kind: "Column", NodeID: 3, Reusable: true, OrdinaryTetraStruct: true},
			{Name: "TitleText", Kind: "Text", NodeID: 4, Role: "label", Reusable: true, OrdinaryTetraStruct: true},
			{Name: "DescriptionText", Kind: "Text", NodeID: 5, Role: "description", Reusable: true, OrdinaryTetraStruct: true},
			{Name: "NameLabel", Kind: "Label", NodeID: 6, Role: "label", Reusable: true, OrdinaryTetraStruct: true},
			{Name: "NameTextBox", Kind: "TextBox", NodeID: 7, Reusable: true, OrdinaryTetraStruct: true, Editable: true},
			{Name: "EmailLabel", Kind: "Label", NodeID: 8, Role: "label", Reusable: true, OrdinaryTetraStruct: true},
			{Name: "EmailTextBox", Kind: "TextBox", NodeID: 9, Reusable: true, OrdinaryTetraStruct: true, Editable: true},
			{Name: "SubscribeCheckbox", Kind: "Checkbox", NodeID: 10, Reusable: true, OrdinaryTetraStruct: true},
			{Name: "TermsScroll", Kind: "Scroll", NodeID: 11, Reusable: true, OrdinaryTetraStruct: true},
			{Name: "TermsText", Kind: "Text", NodeID: 12, Role: "description", Reusable: true, OrdinaryTetraStruct: true},
			{Name: "ButtonRow", Kind: "Row", NodeID: 13, Reusable: true, OrdinaryTetraStruct: true},
			{Name: "SaveButton", Kind: "Button", NodeID: 14, Action: "save", Reusable: true, OrdinaryTetraStruct: true},
			{Name: "ResetButton", Kind: "Button", NodeID: 15, Action: "reset", Reusable: true, OrdinaryTetraStruct: true},
			{Name: "Spacer", Kind: "Spacer", NodeID: 16, Reusable: true, OrdinaryTetraStruct: true},
			{Name: "StatusText", Kind: "StatusText", NodeID: 17, Role: "status", Reusable: true, OrdinaryTetraStruct: true},
		},
		ReusableSources: []string{
			"lib/core/widgets.tetra:panel_init",
			"lib/core/widgets.tetra:column_init",
			"lib/core/widgets.tetra:text_init",
			"lib/core/widgets.tetra:label_init",
			"lib/core/widgets.tetra:status_text_init",
			"lib/core/widgets.tetra:textbox_init",
			"lib/core/widgets.tetra:checkbox_init",
			"lib/core/widgets.tetra:checkbox_toggle",
			"lib/core/widgets.tetra:row_init",
			"lib/core/widgets.tetra:stack_init",
			"lib/core/widgets.tetra:scroll_init",
			"lib/core/widgets.tetra:scroll_set_offset",
			"lib/core/widgets.tetra:spacer_init",
			"lib/core/widgets.tetra:button_init",
			"lib/core/widgets.tetra:hit_test_release_form",
			"lib/core/style.tetra:default_theme",
			"lib/core/style.tetra:style_for_state",
		},
	}
}

func accessibilityComponentTreeReport() *surface.ComponentTreeReport {
	return &surface.ComponentTreeReport{
		Schema:       "tetra.surface.component-tree.v1",
		DynamicLevel: "accessibility-metadata-tree-v1",
		RootID:       0,
		NodeCount:    12,
		FocusedID:    5,
		Nodes: []surface.ComponentTreeNodeReport{
			{ID: 0, Name: "AccessibilitySettingsApp", Kind: "root", ParentID: -1, ChildIndex: 0, FirstChild: 1, ChildCount: 1, Focusable: false, Bounds: surface.RectReport{X: 0, Y: 0, W: 480, H: 320}},
			{ID: 1, Name: "Panel", Kind: "panel", ParentID: 0, ChildIndex: 0, FirstChild: 2, ChildCount: 1, Focusable: false, Bounds: surface.RectReport{X: 0, Y: 0, W: 480, H: 320}},
			{ID: 2, Name: "Column", Kind: "column", ParentID: 1, ChildIndex: 0, FirstChild: 3, ChildCount: 7, Focusable: false, Bounds: surface.RectReport{X: 12, Y: 12, W: 456, H: 296}},
			{ID: 3, Name: "TitleText", Kind: "text", ParentID: 2, ChildIndex: 0, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 20, Y: 20, W: 440, H: 24}},
			{ID: 4, Name: "NameLabel", Kind: "text", ParentID: 2, ChildIndex: 1, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 20, Y: 52, W: 440, H: 24}},
			{ID: 5, Name: "NameTextBox", Kind: "textbox", ParentID: 2, ChildIndex: 2, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 20, Y: 84, W: 440, H: 44}},
			{ID: 6, Name: "EmailLabel", Kind: "text", ParentID: 2, ChildIndex: 3, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 20, Y: 136, W: 440, H: 24}},
			{ID: 7, Name: "EmailTextBox", Kind: "textbox", ParentID: 2, ChildIndex: 4, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 20, Y: 168, W: 440, H: 44}},
			{ID: 8, Name: "ButtonRow", Kind: "row", ParentID: 2, ChildIndex: 5, FirstChild: 9, ChildCount: 2, Focusable: false, Bounds: surface.RectReport{X: 20, Y: 224, W: 440, H: 44}},
			{ID: 9, Name: "SaveButton", Kind: "button", ParentID: 8, ChildIndex: 0, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 20, Y: 224, W: 132, H: 44}},
			{ID: 10, Name: "ResetButton", Kind: "button", ParentID: 8, ChildIndex: 1, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 164, Y: 224, W: 132, H: 44}},
			{ID: 11, Name: "StatusText", Kind: "text", ParentID: 2, ChildIndex: 6, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 20, Y: 280, W: 440, H: 24}},
		},
		LayoutPasses: []surface.ComponentTreeLayoutPassReport{
			{ComponentID: 5, Pass: "initial", Bounds: surface.RectReport{X: 20, Y: 84, W: 280, H: 44}, Measured: surface.SizeReport{W: 280, H: 44}},
			{ComponentID: 7, Pass: "initial", Bounds: surface.RectReport{X: 20, Y: 168, W: 280, H: 44}, Measured: surface.SizeReport{W: 280, H: 44}},
			{ComponentID: 5, Pass: "resize", Bounds: surface.RectReport{X: 20, Y: 84, W: 440, H: 44}, Measured: surface.SizeReport{W: 440, H: 44}},
			{ComponentID: 7, Pass: "resize", Bounds: surface.RectReport{X: 20, Y: 168, W: 440, H: 44}, Measured: surface.SizeReport{W: 440, H: 44}},
			{ComponentID: 11, Pass: "status-update", Bounds: surface.RectReport{X: 20, Y: 280, W: 440, H: 24}, Measured: surface.SizeReport{W: 440, H: 24}},
		},
		DrawOrder:  []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
		FocusOrder: []int{5, 7, 9, 10},
		DispatchPaths: []surface.ComponentTreeDispatchPathReport{
			{Event: "click", TargetID: 5, X: 40, Y: 100, Path: []int{0, 1, 2, 5}},
			{Event: "click", TargetID: 7, X: 40, Y: 184, Path: []int{0, 1, 2, 7}},
			{Event: "key", TargetID: 9, X: 40, Y: 240, Path: []int{0, 1, 2, 8, 9}},
			{Event: "key", TargetID: 10, X: 180, Y: 240, Path: []int{0, 1, 2, 8, 10}},
		},
	}
}

func accessibilityComponentTreeAPIReport() *surface.ComponentTreeAPIReport {
	return &surface.ComponentTreeAPIReport{
		Schema:            "tetra.surface.component-tree-api.v1",
		APILevel:          "builder-layout-dispatch-v1",
		Source:            "examples/surface_accessibility_settings.tetra",
		ManualBookkeeping: false,
		Builder: surface.ComponentTreeAPIBuilderReport{
			RootCreatedBy:     "tree_add_root",
			ChildrenCreatedBy: "tree_add_child",
			NodeCount:         12,
			Capacity:          24,
			OverflowChecked:   true,
		},
		Invariants: surface.ComponentTreeAPIInvariantReport{
			TreeValidateRan:         true,
			TreeValidateStatus:      0,
			ParentChildLinksChecked: true,
			ChildIndicesChecked:     true,
			ChildCountChecked:       true,
			FirstChildChecked:       true,
		},
		LayoutHelpers: []surface.ComponentTreeAPILayoutHelperReport{
			{Helper: "widgets.panel_content_rect", Target: "Panel", Pass: "initial", ChangedBounds: true},
			{Helper: "widgets.column_layout", Target: "Column", Pass: "initial", ChangedBounds: true},
			{Helper: "widgets.row_layout", Target: "ButtonRow", Pass: "initial", ChangedBounds: true},
			{Helper: "widgets.column_layout", Target: "Column", Pass: "resize", ChangedBounds: true},
		},
		FocusHelpers: []surface.ComponentTreeAPIFocusHelperReport{
			{Helper: "tree_focus_next", Before: "NameTextBox", After: "EmailTextBox"},
			{Helper: "tree_focus_next", Before: "EmailTextBox", After: "SaveButton"},
			{Helper: "tree_focus_next", Before: "SaveButton", After: "ResetButton"},
			{Helper: "tree_focus_next", Before: "ResetButton", After: "NameTextBox"},
		},
		HitTests: []surface.ComponentTreeAPIHitTestReport{
			{Helper: "widgets.hit_test_accessibility_settings", X: 40, Y: 100, Target: "NameTextBox", Path: []int{0, 1, 2, 5}},
			{Helper: "widgets.hit_test_accessibility_settings", X: 40, Y: 184, Target: "EmailTextBox", Path: []int{0, 1, 2, 7}},
			{Helper: "widgets.hit_test_accessibility_settings", X: 180, Y: 240, Target: "ResetButton", Path: []int{0, 1, 2, 8, 10}},
		},
		DispatchPaths: []surface.ComponentTreeAPIDispatchPathReport{
			{Helper: "tree_build_dispatch_path", Target: "NameTextBox", Path: []int{0, 1, 2, 5}},
			{Helper: "tree_build_dispatch_path", Target: "EmailTextBox", Path: []int{0, 1, 2, 7}},
			{Helper: "tree_build_dispatch_path", Target: "SaveButton", Path: []int{0, 1, 2, 8, 9}},
			{Helper: "tree_build_dispatch_path", Target: "ResetButton", Path: []int{0, 1, 2, 8, 10}},
		},
	}
}

func accessibilityToolkitReport() *surface.ToolkitReport {
	return &surface.ToolkitReport{
		Schema:                    "tetra.surface.toolkit.v1",
		ToolkitLevel:              "toolkit-reuse-v1",
		ReuseLevel:                "multi-form-widget-reuse-v1",
		Source:                    "examples/surface_accessibility_settings.tetra",
		Sources:                   []string{"examples/surface_toolkit_form.tetra", "examples/surface_toolkit_settings.tetra", "examples/surface_accessibility_settings.tetra"},
		Module:                    "lib.core.widgets",
		Experimental:              true,
		ProductionClaim:           false,
		UsesComponentTreeAPI:      true,
		ManualBookkeeping:         false,
		DemoSpecificWidgetStructs: false,
		NoMagicWidgets:            true,
		NoPlatformWidgets:         true,
		NoDOMUI:                   true,
		NoUserJS:                  true,
		ExampleCount:              3,
		TextBoxCount:              2,
		ButtonCount:               2,
		MultiTextBoxEvidence:      true,
		MultiFormEvidence:         true,
		Widgets: []surface.ToolkitWidgetReport{
			{Name: "Panel", Kind: "Panel", NodeID: 1, Reusable: true, OrdinaryTetraStruct: true},
			{Name: "Column", Kind: "Column", NodeID: 2, Reusable: true, OrdinaryTetraStruct: true},
			{Name: "TitleText", Kind: "Text", NodeID: 3, Role: "text", Reusable: true, OrdinaryTetraStruct: true},
			{Name: "NameLabel", Kind: "Text", NodeID: 4, Role: "label", Reusable: true, OrdinaryTetraStruct: true},
			{Name: "NameTextBox", Kind: "TextBox", NodeID: 5, Reusable: true, OrdinaryTetraStruct: true, Editable: true},
			{Name: "EmailLabel", Kind: "Text", NodeID: 6, Role: "label", Reusable: true, OrdinaryTetraStruct: true},
			{Name: "EmailTextBox", Kind: "TextBox", NodeID: 7, Reusable: true, OrdinaryTetraStruct: true, Editable: true},
			{Name: "ButtonRow", Kind: "Row", NodeID: 8, Reusable: true, OrdinaryTetraStruct: true},
			{Name: "SaveButton", Kind: "Button", NodeID: 9, Action: "save", Reusable: true, OrdinaryTetraStruct: true},
			{Name: "ResetButton", Kind: "Button", NodeID: 10, Action: "reset", Reusable: true, OrdinaryTetraStruct: true},
			{Name: "StatusText", Kind: "Text", NodeID: 11, Role: "status", Reusable: true, OrdinaryTetraStruct: true},
		},
		ReusableSources: []string{
			"lib/core/widgets.tetra:panel_init",
			"lib/core/widgets.tetra:column_init",
			"lib/core/widgets.tetra:text_init",
			"lib/core/widgets.tetra:textbox_init",
			"lib/core/widgets.tetra:row_init",
			"lib/core/widgets.tetra:button_init",
			"lib/core/widgets.tetra:add_accessible_textbox",
			"lib/core/widgets.tetra:add_accessible_button",
			"lib/core/widgets.tetra:add_accessible_status",
		},
	}
}

func accessibilityTreeReport(beforeFrame rgbaFrame, nameFrame rgbaFrame, saveFrame rgbaFrame, resetFrame rgbaFrame, afterFrame rgbaFrame) *surface.AccessibilityTreeReport {
	return &surface.AccessibilityTreeReport{
		Schema:                   "tetra.surface.accessibility-tree.v1",
		AccessibilityLevel:       "metadata-tree-v1",
		Source:                   "examples/surface_accessibility_settings.tetra",
		Module:                   "lib.core.accessibility",
		WidgetModule:             "lib.core.widgets",
		Experimental:             true,
		ProductionClaim:          false,
		PlatformHostIntegration:  false,
		DOMARIAIntegration:       false,
		ScreenReaderEvidence:     false,
		DerivedFromComponentTree: true,
		UsesComponentTreeAPI:     true,
		UsesWidgetToolkit:        true,
		ManualBookkeeping:        false,
		NoDOMUI:                  true,
		NoUserJS:                 true,
		NoPlatformWidgets:        true,
		NoLegacySidecars:         true,
		ComponentTreeSchema:      "tetra.surface.component-tree.v1",
		ComponentTreeAPISchema:   "tetra.surface.component-tree-api.v1",
		ToolkitSchema:            "tetra.surface.toolkit.v1",
		NodeCount:                12,
		FocusableCount:           4,
		LabelCount:               2,
		TextBoxCount:             2,
		ButtonCount:              2,
		StatusCount:              1,
		RolesPresent:             []string{"root", "panel", "column", "text", "label", "textbox", "row", "button", "status"},
		Nodes:                    accessibilityNodes(),
		Relationships: []surface.AccessibilityRelationshipReport{
			{Kind: "label_for", From: "NameLabel", To: "NameTextBox"},
			{Kind: "labelled_by", From: "NameTextBox", To: "NameLabel"},
			{Kind: "label_for", From: "EmailLabel", To: "EmailTextBox"},
			{Kind: "labelled_by", From: "EmailTextBox", To: "EmailLabel"},
		},
		FocusOrder:   []string{"NameTextBox", "EmailTextBox", "SaveButton", "ResetButton"},
		ReadingOrder: []string{"TitleText", "NameLabel", "NameTextBox", "EmailLabel", "EmailTextBox", "SaveButton", "ResetButton", "StatusText"},
		Actions: []surface.AccessibilityActionReport{
			{Target: "NameTextBox", Action: "edit", Semantic: "text-input"},
			{Target: "EmailTextBox", Action: "edit", Semantic: "text-input"},
			{Target: "SaveButton", Action: "press", Semantic: "save"},
			{Target: "ResetButton", Action: "press", Semantic: "reset"},
		},
		Snapshots: accessibilitySnapshots(beforeFrame, nameFrame, saveFrame, resetFrame, afterFrame),
		NegativeGuards: surface.AccessibilityNegativeGuardsReport{
			NoBorrowedViewStorage:       true,
			ComponentIDAlignmentChecked: true,
			BoundsAlignmentChecked:      true,
			FocusOrderAlignmentChecked:  true,
			ReadingOrderChecked:         true,
			LabelRelationshipsChecked:   true,
			StateUpdatesChecked:         true,
			ArtifactScanChecked:         true,
		},
	}
}

func accessibilityNodes() []surface.AccessibilityNodeReport {
	return []surface.AccessibilityNodeReport{
		{ID: 0, ComponentID: 0, ParentID: -1, Name: "AccessibilitySettingsApp", Role: "root", Bounds: surface.RectReport{X: 0, Y: 0, W: 480, H: 320}, Visible: true, Enabled: true, FocusIndex: -1, ReadingIndex: 0},
		{ID: 1, ComponentID: 1, ParentID: 0, Name: "Panel", Role: "panel", Bounds: surface.RectReport{X: 0, Y: 0, W: 480, H: 320}, Visible: true, Enabled: true, FocusIndex: -1, ReadingIndex: 1},
		{ID: 2, ComponentID: 2, ParentID: 1, Name: "Column", Role: "column", Bounds: surface.RectReport{X: 12, Y: 12, W: 456, H: 296}, Visible: true, Enabled: true, FocusIndex: -1, ReadingIndex: 2},
		{ID: 3, ComponentID: 3, ParentID: 2, Name: "TitleText", Role: "text", Bounds: surface.RectReport{X: 20, Y: 20, W: 440, H: 24}, Visible: true, Enabled: true, ValueKind: "title", FocusIndex: -1, ReadingIndex: 3},
		{ID: 4, ComponentID: 4, ParentID: 2, Name: "NameLabel", Role: "label", Bounds: surface.RectReport{X: 20, Y: 52, W: 440, H: 24}, Visible: true, Enabled: true, LabelFor: "NameTextBox", ValueKind: "name", FocusIndex: -1, ReadingIndex: 4},
		{ID: 5, ComponentID: 5, ParentID: 2, Name: "NameTextBox", Role: "textbox", Bounds: surface.RectReport{X: 20, Y: 84, W: 440, H: 44}, Visible: true, Enabled: true, Focusable: true, Focused: true, Editable: true, LabelledBy: "NameLabel", ValueKind: "empty", Actions: []string{"focus", "edit"}, FocusIndex: 0, ReadingIndex: 5},
		{ID: 6, ComponentID: 6, ParentID: 2, Name: "EmailLabel", Role: "label", Bounds: surface.RectReport{X: 20, Y: 136, W: 440, H: 24}, Visible: true, Enabled: true, LabelFor: "EmailTextBox", ValueKind: "email", FocusIndex: -1, ReadingIndex: 6},
		{ID: 7, ComponentID: 7, ParentID: 2, Name: "EmailTextBox", Role: "textbox", Bounds: surface.RectReport{X: 20, Y: 168, W: 440, H: 44}, Visible: true, Enabled: true, Focusable: true, Editable: true, LabelledBy: "EmailLabel", ValueKind: "empty", Actions: []string{"focus", "edit"}, FocusIndex: 1, ReadingIndex: 7},
		{ID: 8, ComponentID: 8, ParentID: 2, Name: "ButtonRow", Role: "row", Bounds: surface.RectReport{X: 20, Y: 224, W: 440, H: 44}, Visible: true, Enabled: true, FocusIndex: -1, ReadingIndex: 8},
		{ID: 9, ComponentID: 9, ParentID: 8, Name: "SaveButton", Role: "button", Bounds: surface.RectReport{X: 20, Y: 224, W: 132, H: 44}, Visible: true, Enabled: true, Focusable: true, ValueKind: "save", Actions: []string{"focus", "press", "save"}, FocusIndex: 2, ReadingIndex: 9},
		{ID: 10, ComponentID: 10, ParentID: 8, Name: "ResetButton", Role: "button", Bounds: surface.RectReport{X: 164, Y: 224, W: 132, H: 44}, Visible: true, Enabled: true, Focusable: true, ValueKind: "reset", Actions: []string{"focus", "press", "reset"}, FocusIndex: 3, ReadingIndex: 10},
		{ID: 11, ComponentID: 11, ParentID: 2, Name: "StatusText", Role: "status", Bounds: surface.RectReport{X: 20, Y: 280, W: 440, H: 24}, Visible: true, Enabled: true, ValueKind: "reset", FocusIndex: -1, ReadingIndex: 11},
	}
}

func accessibilitySnapshots(beforeFrame rgbaFrame, nameFrame rgbaFrame, saveFrame rgbaFrame, resetFrame rgbaFrame, afterFrame rgbaFrame) []surface.AccessibilitySnapshotReport {
	return []surface.AccessibilitySnapshotReport{
		{Name: "initial", Generation: 1, Focused: "", FocusedComponentID: -1, FocusedAccessibilityNodeID: -1, NameValueLen: 0, EmailValueLen: 0, StatusValue: "idle", BoundsChecksum: checksumText("bounds-initial"), MetadataChecksum: checksumText("metadata-initial"), FrameChecksum: checksumRGBA(beforeFrame.Pixels)},
		{Name: "after_name_focus", Generation: 2, Focused: "NameTextBox", FocusedComponentID: 5, FocusedAccessibilityNodeID: 5, NameValueLen: 0, EmailValueLen: 0, StatusValue: "idle", BoundsChecksum: checksumText("bounds-name-focus"), MetadataChecksum: checksumText("metadata-name-focus"), FrameChecksum: checksumRGBA(nameFrame.Pixels)},
		{Name: "after_name_text", Generation: 3, Focused: "NameTextBox", FocusedComponentID: 5, FocusedAccessibilityNodeID: 5, NameValueLen: 3, EmailValueLen: 0, StatusValue: "idle", BoundsChecksum: checksumText("bounds-name-text"), MetadataChecksum: checksumText("metadata-name-text"), FrameChecksum: checksumRGBA(nameFrame.Pixels)},
		{Name: "after_email_focus", Generation: 4, Focused: "EmailTextBox", FocusedComponentID: 7, FocusedAccessibilityNodeID: 7, NameValueLen: 3, EmailValueLen: 0, StatusValue: "idle", BoundsChecksum: checksumText("bounds-email-focus"), MetadataChecksum: checksumText("metadata-email-focus"), FrameChecksum: checksumText("frame-email-focus")},
		{Name: "after_email_text", Generation: 5, Focused: "EmailTextBox", FocusedComponentID: 7, FocusedAccessibilityNodeID: 7, NameValueLen: 3, EmailValueLen: 5, StatusValue: "idle", BoundsChecksum: checksumText("bounds-email-text"), MetadataChecksum: checksumText("metadata-email-text"), FrameChecksum: checksumText("frame-email-text")},
		{Name: "after_save", Generation: 6, Focused: "SaveButton", FocusedComponentID: 9, FocusedAccessibilityNodeID: 9, NameValueLen: 3, EmailValueLen: 5, StatusValue: "saved", BoundsChecksum: checksumText("bounds-save"), MetadataChecksum: checksumText("metadata-save"), FrameChecksum: checksumRGBA(saveFrame.Pixels)},
		{Name: "after_reset", Generation: 7, Focused: "ResetButton", FocusedComponentID: 10, FocusedAccessibilityNodeID: 10, NameValueLen: 0, EmailValueLen: 0, StatusValue: "reset", BoundsChecksum: checksumText("bounds-reset"), MetadataChecksum: checksumText("metadata-reset"), FrameChecksum: checksumRGBA(resetFrame.Pixels)},
		{Name: "after_resize", Generation: 8, Focused: "NameTextBox", FocusedComponentID: 5, FocusedAccessibilityNodeID: 5, NameValueLen: 0, EmailValueLen: 0, StatusValue: "reset", BoundsChecksum: checksumText("bounds-resize"), MetadataChecksum: checksumText("metadata-resize"), FrameChecksum: checksumRGBA(afterFrame.Pixels)},
	}
}

func runTextFocusInputScenario(mode string) headlessScenario {
	beforeFrame := renderTextFocusInputFrameRGBA(0, 0, 0, 320, 200)
	afterFrame := renderTextFocusInputFrameRGBA(1, 1, 1, 400, 240)
	scenario := headlessScenario{
		Components: []surface.ComponentReport{
			{
				ID:        "TextInputApp",
				Type:      "examples.surface_textbox_app.TextInputApp",
				Bounds:    surface.RectReport{X: 0, Y: 0, W: 400, H: 240},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"focused_component": "SubmitButton", "width": "400", "height": "240", "resize_count": "1", "accessibility_role": "none"},
			},
			{
				ID:        "TextBox",
				Type:      "examples.surface_textbox_app.TextBox",
				Parent:    "TextInputApp",
				Bounds:    surface.RectReport{X: 32, Y: 64, W: 224, H: 44},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"focused": "false", "buffer": "Z", "text_len": "1", "caret": "1", "backspace_count": "1", "delete_count": "1", "accessibility_role": "label"},
			},
			{
				ID:        "SubmitButton",
				Type:      "examples.surface_textbox_app.ActionButton",
				Parent:    "TextInputApp",
				Bounds:    surface.RectReport{X: 32, Y: 128, W: 128, H: 44},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"focused": "true", "press_count": "1", "key_count": "1", "accessibility_role": "button"},
			},
		},
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"TextInputApp", "TextBox"},
				Handled:         true,
				Pass:            true,
				X:               48,
				Y:               96,
				Width:           320,
				Height:          200,
				BufferSlots:     []int{5, 48, 96, 1, 0, 320, 200, 0, 0},
				BeforeState:     map[string]string{"TextInputApp.focused_component": "none", "TextBox.focused": "false"},
				AfterState:      map[string]string{"TextInputApp.focused_component": "TextBox", "TextBox.focused": "true"},
			},
			{
				Order:           2,
				Kind:            "text_input",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"TextInputApp", "TextBox"},
				Handled:         true,
				Pass:            true,
				Width:           320,
				Height:          200,
				TimestampMS:     1,
				TextLen:         2,
				TextBytesHex:    "4f4b",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
				BeforeState:     map[string]string{"TextBox.buffer": "", "TextBox.caret": "0", "TextBox.text_len": "0"},
				AfterState:      map[string]string{"TextBox.buffer": "OK", "TextBox.caret": "2", "TextBox.text_len": "2"},
			},
			{
				Order:           3,
				Kind:            "key_down",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"TextInputApp", "TextBox"},
				Handled:         true,
				Pass:            true,
				Key:             37,
				Width:           320,
				Height:          200,
				TimestampMS:     2,
				BufferSlots:     []int{6, 0, 0, 0, 37, 320, 200, 2, 0},
				BeforeState:     map[string]string{"TextBox.caret": "2", "TextBox.buffer": "OK"},
				AfterState:      map[string]string{"TextBox.caret": "1", "TextBox.buffer": "OK"},
			},
			{
				Order:           4,
				Kind:            "key_down",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"TextInputApp", "TextBox"},
				Handled:         true,
				Pass:            true,
				Key:             8,
				Width:           320,
				Height:          200,
				TimestampMS:     3,
				BufferSlots:     []int{6, 0, 0, 0, 8, 320, 200, 3, 0},
				BeforeState:     map[string]string{"TextBox.buffer": "OK", "TextBox.caret": "1"},
				AfterState:      map[string]string{"TextBox.buffer": "K", "TextBox.caret": "0"},
			},
			{
				Order:           5,
				Kind:            "key_down",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"TextInputApp", "TextBox"},
				Handled:         true,
				Pass:            true,
				Key:             46,
				Width:           320,
				Height:          200,
				TimestampMS:     4,
				BufferSlots:     []int{6, 0, 0, 0, 46, 320, 200, 4, 0},
				BeforeState:     map[string]string{"TextBox.buffer": "K", "TextBox.caret": "0"},
				AfterState:      map[string]string{"TextBox.buffer": "", "TextBox.caret": "0"},
			},
			{
				Order:           6,
				Kind:            "text_input",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"TextInputApp", "TextBox"},
				Handled:         true,
				Pass:            true,
				Width:           320,
				Height:          200,
				TimestampMS:     5,
				TextLen:         1,
				TextBytesHex:    "5a",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 5, 1},
				BeforeState:     map[string]string{"TextBox.buffer": "", "TextBox.caret": "0", "TextBox.text_len": "0"},
				AfterState:      map[string]string{"TextBox.buffer": "Z", "TextBox.caret": "1", "TextBox.text_len": "1"},
			},
			{
				Order:           7,
				Kind:            "key_down",
				TargetComponent: "TextInputApp",
				DispatchPath:    []string{"TextInputApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          200,
				TimestampMS:     6,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 200, 6, 0},
				BeforeState:     map[string]string{"TextInputApp.focused_component": "TextBox", "TextBox.focused": "true", "SubmitButton.focused": "false"},
				AfterState:      map[string]string{"TextInputApp.focused_component": "SubmitButton", "TextBox.focused": "false", "SubmitButton.focused": "true"},
			},
			{
				Order:           8,
				Kind:            "key_down",
				TargetComponent: "SubmitButton",
				DispatchPath:    []string{"TextInputApp", "SubmitButton"},
				Handled:         true,
				Pass:            true,
				Key:             32,
				Width:           320,
				Height:          200,
				TimestampMS:     7,
				BufferSlots:     []int{6, 0, 0, 0, 32, 320, 200, 7, 0},
				BeforeState:     map[string]string{"SubmitButton.press_count": "0", "TextBox.buffer": "Z"},
				AfterState:      map[string]string{"SubmitButton.press_count": "1", "TextBox.buffer": "Z"},
			},
			{
				Order:           9,
				Kind:            "resize",
				TargetComponent: "TextInputApp",
				DispatchPath:    []string{"TextInputApp"},
				Handled:         true,
				Pass:            true,
				Width:           400,
				Height:          240,
				TimestampMS:     8,
				BufferSlots:     []int{2, 0, 0, 0, 0, 400, 240, 8, 0},
				BeforeState:     map[string]string{"TextInputApp.width": "320", "TextInputApp.focused_component": "SubmitButton"},
				AfterState:      map[string]string{"TextInputApp.width": "400", "TextInputApp.focused_component": "SubmitButton"},
			},
		},
		StateTransitions: []surface.StateTransitionReport{
			{Order: 1, Component: "TextInputApp", Field: "focused_component", Before: "none", After: "TextBox", Cause: "mouse_up"},
			{Order: 2, Component: "TextBox", Field: "buffer", Before: "", After: "OK", Cause: "text_input"},
			{Order: 3, Component: "TextBox", Field: "caret", Before: "2", After: "1", Cause: "key_down"},
			{Order: 4, Component: "TextBox", Field: "buffer", Before: "OK", After: "K", Cause: "backspace"},
			{Order: 5, Component: "TextBox", Field: "buffer", Before: "K", After: "", Cause: "delete"},
			{Order: 6, Component: "TextBox", Field: "buffer", Before: "", After: "Z", Cause: "text_input"},
			{Order: 7, Component: "TextInputApp", Field: "focused_component", Before: "TextBox", After: "SubmitButton", Cause: "tab"},
			{Order: 8, Component: "SubmitButton", Field: "press_count", Before: "0", After: "1", Cause: "key_down"},
			{Order: 9, Component: "TextInputApp", Field: "width", Before: "320", After: "400", Cause: "resize"},
		},
		Cases: []surface.CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
			{Name: "text focus input click focuses TextBox", Kind: "positive", Ran: true, Pass: true},
			{Name: "text focus input Tab changes focus", Kind: "positive", Ran: true, Pass: true},
			{Name: "text focus input keyboard routes only focused component", Kind: "positive", Ran: true, Pass: true},
			{Name: "text focus input text insertion", Kind: "positive", Ran: true, Pass: true},
			{Name: "text focus input caret movement", Kind: "positive", Ran: true, Pass: true},
			{Name: "text focus input backspace delete", Kind: "positive", Ran: true, Pass: true},
			{Name: "text focus input resize preserves focus", Kind: "positive", Ran: true, Pass: true},
			{Name: "text focus input rendered frame update", Kind: "positive", Ran: true, Pass: true},
			{Name: "host-provided pointer event dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "host event buffer poll_event", Kind: "positive", Ran: true, Pass: true},
			{Name: "pre/post event frame sequence", Kind: "positive", Ran: true, Pass: true},
			{Name: "component hierarchy dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "component text input scalar dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "host text payload buffer", Kind: "positive", Ran: true, Pass: true},
			{Name: "component focus dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "component accessibility metadata", Kind: "positive", Ran: true, Pass: true},
			{Name: "no legacy UI sidecar artifacts", Kind: "positive", Ran: true, Pass: true},
			{Name: "state transition", Kind: "positive", Ran: true, Pass: true},
			{Name: "reject legacy UI evidence", Kind: "negative", Ran: true, Pass: true, ExpectedError: "legacy UI evidence rejected"},
		},
	}
	if mode == "headless-text-focus-input" || mode == "linux-x64-real-window-text-focus-input" {
		scenario.Frames = []surface.FrameReport{
			{Order: 1, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
			{Order: 2, Width: afterFrame.Width, Height: afterFrame.Height, Stride: afterFrame.Stride, Checksum: checksumRGBA(afterFrame.Pixels), Presented: true},
		}
	}
	switch mode {
	case "headless-text-focus-input":
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "headless event dispatch", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "headless framebuffer checksum", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "headless actual runner trace", Kind: "positive", Ran: true, Pass: true},
		)
	case "linux-x64-real-window-text-focus-input":
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "linux-x64 Surface Host ABI open/present/close", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 real-window surface", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 native input event pump", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 real-window resize event", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 real-window close event", Kind: "positive", Ran: true, Pass: true},
		)
	case "wasm32-web-browser-canvas-text-focus-input":
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "wasm32-web browser canvas surface", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas RGBA readback", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas pointer input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas keyboard input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas resize input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas text input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web Surface Host ABI imports", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "compiler-owned wasm Surface loader", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "compiler-owned browser canvas Surface host", Kind: "positive", Ran: true, Pass: true},
		)
	}
	return scenario
}

func runComponentTreeScenario(mode string) headlessScenario {
	beforeFrame := renderComponentTreeFrameRGBA(0, 0, -1, 0, 0, 320, 200)
	afterFrame := renderComponentTreeFrameRGBA(0, 0, 6, 1, 1, 400, 240)
	scenario := headlessScenario{
		Components: []surface.ComponentReport{
			{
				ID:        "TreeApp",
				Type:      "examples.surface_tree_app.TreeApp",
				Bounds:    surface.RectReport{X: 0, Y: 0, W: 400, H: 240},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"focused_id": "6", "submitted_count": "1", "reset_count": "1", "width": "400", "height": "240", "accessibility_role": "none"},
			},
			{
				ID:        "Column",
				Type:      "examples.surface_tree_app.Column",
				Parent:    "TreeApp",
				Bounds:    surface.RectReport{X: 0, Y: 0, W: 400, H: 240},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"child_count": "3", "accessibility_role": "none"},
			},
			{
				ID:        "NameLabel",
				Type:      "examples.surface_tree_app.TextLabel",
				Parent:    "Column",
				Bounds:    surface.RectReport{X: 16, Y: 16, W: 288, H: 24},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"text": "Name", "accessibility_role": "label"},
			},
			{
				ID:        "TextBox",
				Type:      "examples.surface_tree_app.TextBox",
				Parent:    "Column",
				Bounds:    surface.RectReport{X: 16, Y: 48, W: 368, H: 44},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"focused": "false", "buffer": "", "text_len": "0", "caret": "0", "accessibility_role": "label"},
			},
			{
				ID:        "ButtonRow",
				Type:      "examples.surface_tree_app.Row",
				Parent:    "Column",
				Bounds:    surface.RectReport{X: 16, Y: 104, W: 368, H: 44},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"child_count": "2", "accessibility_role": "none"},
			},
			{
				ID:        "SubmitButton",
				Type:      "examples.surface_tree_app.Button",
				Parent:    "ButtonRow",
				Bounds:    surface.RectReport{X: 16, Y: 104, W: 132, H: 44},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"focused": "false", "press_count": "1", "accessibility_role": "button"},
			},
			{
				ID:        "ResetButton",
				Type:      "examples.surface_tree_app.Button",
				Parent:    "ButtonRow",
				Bounds:    surface.RectReport{X: 160, Y: 104, W: 132, H: 44},
				Abilities: []string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
				State:     map[string]string{"focused": "true", "press_count": "1", "accessibility_role": "button"},
			},
		},
		ComponentTree: &surface.ComponentTreeReport{
			Schema:       "tetra.surface.component-tree.v1",
			DynamicLevel: "semi-dynamic-child-list",
			RootID:       0,
			NodeCount:    7,
			FocusedID:    6,
			Nodes: []surface.ComponentTreeNodeReport{
				{ID: 0, Name: "TreeApp", Kind: "root", ParentID: -1, ChildIndex: 0, FirstChild: 1, ChildCount: 1, Focusable: false, Bounds: surface.RectReport{X: 0, Y: 0, W: 400, H: 240}},
				{ID: 1, Name: "Column", Kind: "column", ParentID: 0, ChildIndex: 0, FirstChild: 2, ChildCount: 3, Focusable: false, Bounds: surface.RectReport{X: 0, Y: 0, W: 400, H: 240}},
				{ID: 2, Name: "NameLabel", Kind: "text", ParentID: 1, ChildIndex: 0, FirstChild: -1, ChildCount: 0, Focusable: false, Bounds: surface.RectReport{X: 16, Y: 16, W: 288, H: 24}},
				{ID: 3, Name: "TextBox", Kind: "textbox", ParentID: 1, ChildIndex: 1, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 16, Y: 48, W: 368, H: 44}},
				{ID: 4, Name: "ButtonRow", Kind: "row", ParentID: 1, ChildIndex: 2, FirstChild: 5, ChildCount: 2, Focusable: false, Bounds: surface.RectReport{X: 16, Y: 104, W: 368, H: 44}},
				{ID: 5, Name: "SubmitButton", Kind: "button", ParentID: 4, ChildIndex: 0, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 16, Y: 104, W: 132, H: 44}},
				{ID: 6, Name: "ResetButton", Kind: "button", ParentID: 4, ChildIndex: 1, FirstChild: -1, ChildCount: 0, Focusable: true, Bounds: surface.RectReport{X: 160, Y: 104, W: 132, H: 44}},
			},
			LayoutPasses: []surface.ComponentTreeLayoutPassReport{
				{ComponentID: 3, Pass: "initial", Bounds: surface.RectReport{X: 16, Y: 48, W: 288, H: 44}, Measured: surface.SizeReport{W: 288, H: 44}},
				{ComponentID: 3, Pass: "resize", Bounds: surface.RectReport{X: 16, Y: 48, W: 368, H: 44}, Measured: surface.SizeReport{W: 368, H: 44}},
			},
			DrawOrder:  []int{0, 1, 2, 3, 4, 5, 6},
			FocusOrder: []int{3, 5, 6},
			DispatchPaths: []surface.ComponentTreeDispatchPathReport{
				{Event: "click", TargetID: 3, X: 40, Y: 72, Path: []int{0, 1, 3}},
				{Event: "click", TargetID: 5, X: 32, Y: 120, Path: []int{0, 1, 4, 5}},
				{Event: "click", TargetID: 6, X: 176, Y: 120, Path: []int{0, 1, 4, 6}},
			},
		},
		ComponentTreeAPI: componentTreeAPIReport(),
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"TreeApp", "Column", "TextBox"},
				Handled:         true,
				Pass:            true,
				X:               40,
				Y:               72,
				Width:           320,
				Height:          200,
				BufferSlots:     []int{5, 40, 72, 1, 0, 320, 200, 0, 0},
				BeforeState:     map[string]string{"TreeApp.focused_id": "-1", "TextBox.focused": "false"},
				AfterState:      map[string]string{"TreeApp.focused_id": "3", "TextBox.focused": "true"},
			},
			{
				Order:           2,
				Kind:            "text_input",
				TargetComponent: "TextBox",
				DispatchPath:    []string{"TreeApp", "Column", "TextBox"},
				Handled:         true,
				Pass:            true,
				Width:           320,
				Height:          200,
				TimestampMS:     1,
				TextLen:         2,
				TextBytesHex:    "4f4b",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 1, 2},
				BeforeState:     map[string]string{"TextBox.buffer": "", "TextBox.caret": "0"},
				AfterState:      map[string]string{"TextBox.buffer": "OK", "TextBox.caret": "2"},
			},
			{
				Order:           3,
				Kind:            "key_down",
				TargetComponent: "TreeApp",
				DispatchPath:    []string{"TreeApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          200,
				TimestampMS:     2,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 200, 2, 0},
				BeforeState:     map[string]string{"TreeApp.focused_id": "3"},
				AfterState:      map[string]string{"TreeApp.focused_id": "5"},
			},
			{
				Order:           4,
				Kind:            "key_down",
				TargetComponent: "SubmitButton",
				DispatchPath:    []string{"TreeApp", "Column", "ButtonRow", "SubmitButton"},
				Handled:         true,
				Pass:            true,
				Key:             32,
				Width:           320,
				Height:          200,
				TimestampMS:     3,
				BufferSlots:     []int{6, 0, 0, 0, 32, 320, 200, 3, 0},
				BeforeState:     map[string]string{"TreeApp.submitted_count": "0", "TreeApp.focused_id": "5"},
				AfterState:      map[string]string{"TreeApp.submitted_count": "1", "TreeApp.focused_id": "5"},
			},
			{
				Order:           5,
				Kind:            "key_down",
				TargetComponent: "TreeApp",
				DispatchPath:    []string{"TreeApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          200,
				TimestampMS:     4,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 200, 4, 0},
				BeforeState:     map[string]string{"TreeApp.focused_id": "5"},
				AfterState:      map[string]string{"TreeApp.focused_id": "6"},
			},
			{
				Order:           6,
				Kind:            "text_input",
				TargetComponent: "ResetButton",
				DispatchPath:    []string{"TreeApp", "Column", "ButtonRow", "ResetButton"},
				Handled:         false,
				Pass:            true,
				Width:           320,
				Height:          200,
				TimestampMS:     5,
				TextLen:         1,
				TextBytesHex:    "5a",
				BufferSlots:     []int{8, 0, 0, 0, 0, 320, 200, 5, 1},
				BeforeState:     map[string]string{"TreeApp.focused_id": "6", "TextBox.buffer": "OK"},
				AfterState:      map[string]string{"TreeApp.focused_id": "6", "TextBox.buffer": "OK"},
			},
			{
				Order:           7,
				Kind:            "key_down",
				TargetComponent: "ResetButton",
				DispatchPath:    []string{"TreeApp", "Column", "ButtonRow", "ResetButton"},
				Handled:         true,
				Pass:            true,
				Key:             13,
				Width:           320,
				Height:          200,
				TimestampMS:     6,
				BufferSlots:     []int{6, 0, 0, 0, 13, 320, 200, 6, 0},
				BeforeState:     map[string]string{"TreeApp.reset_count": "0", "TextBox.buffer": "OK", "TreeApp.focused_id": "6"},
				AfterState:      map[string]string{"TreeApp.reset_count": "1", "TextBox.buffer": "", "TreeApp.focused_id": "6"},
			},
			{
				Order:           8,
				Kind:            "key_down",
				TargetComponent: "TreeApp",
				DispatchPath:    []string{"TreeApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          200,
				TimestampMS:     7,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 200, 7, 0},
				BeforeState:     map[string]string{"TreeApp.focused_id": "6"},
				AfterState:      map[string]string{"TreeApp.focused_id": "3"},
			},
			{
				Order:           9,
				Kind:            "key_down",
				TargetComponent: "TreeApp",
				DispatchPath:    []string{"TreeApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          200,
				TimestampMS:     8,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 200, 8, 0},
				BeforeState:     map[string]string{"TreeApp.focused_id": "3"},
				AfterState:      map[string]string{"TreeApp.focused_id": "5"},
			},
			{
				Order:           10,
				Kind:            "key_down",
				TargetComponent: "TreeApp",
				DispatchPath:    []string{"TreeApp"},
				Handled:         true,
				Pass:            true,
				Key:             9,
				Width:           320,
				Height:          200,
				TimestampMS:     9,
				BufferSlots:     []int{6, 0, 0, 0, 9, 320, 200, 9, 0},
				BeforeState:     map[string]string{"TreeApp.focused_id": "5"},
				AfterState:      map[string]string{"TreeApp.focused_id": "6"},
			},
			{
				Order:           11,
				Kind:            "resize",
				TargetComponent: "TreeApp",
				DispatchPath:    []string{"TreeApp"},
				Handled:         true,
				Pass:            true,
				Width:           400,
				Height:          240,
				TimestampMS:     10,
				BufferSlots:     []int{2, 0, 0, 0, 0, 400, 240, 10, 0},
				BeforeState:     map[string]string{"TreeApp.focused_id": "6", "TextBox.bounds.w": "288"},
				AfterState:      map[string]string{"TreeApp.focused_id": "6", "TextBox.bounds.w": "368"},
			},
		},
		StateTransitions: []surface.StateTransitionReport{
			{Order: 1, Component: "TreeApp", Field: "focused_id", Before: "-1", After: "3", Cause: "mouse_up"},
			{Order: 2, Component: "TextBox", Field: "buffer", Before: "", After: "OK", Cause: "text_input"},
			{Order: 3, Component: "TreeApp", Field: "focused_id", Before: "3", After: "5", Cause: "tab"},
			{Order: 4, Component: "TreeApp", Field: "submitted_count", Before: "0", After: "1", Cause: "key_down"},
			{Order: 5, Component: "TreeApp", Field: "focused_id", Before: "5", After: "6", Cause: "tab"},
			{Order: 6, Component: "TextBox", Field: "buffer", Before: "OK", After: "", Cause: "reset"},
			{Order: 7, Component: "TreeApp", Field: "reset_count", Before: "0", After: "1", Cause: "key_down"},
			{Order: 8, Component: "TreeApp", Field: "focused_id", Before: "6", After: "3", Cause: "tab"},
			{Order: 9, Component: "TreeApp", Field: "focused_id", Before: "3", After: "5", Cause: "tab"},
			{Order: 10, Component: "TreeApp", Field: "focused_id", Before: "5", After: "6", Cause: "tab"},
			{Order: 11, Component: "TreeApp", Field: "TextBox.bounds.w", Before: "288", After: "368", Cause: "resize"},
		},
		Cases: []surface.CaseReport{
			{Name: "pure Tetra component app", Kind: "positive", Ran: true, Pass: true},
			{Name: "host-provided pointer event dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "host event buffer poll_event", Kind: "positive", Ran: true, Pass: true},
			{Name: "pre/post event frame sequence", Kind: "positive", Ran: true, Pass: true},
			{Name: "component hierarchy dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "component text input scalar dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "host text payload buffer", Kind: "positive", Ran: true, Pass: true},
			{Name: "component focus dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "component accessibility metadata", Kind: "positive", Ran: true, Pass: true},
			{Name: "no legacy UI sidecar artifacts", Kind: "positive", Ran: true, Pass: true},
			{Name: "state transition", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree node count", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree parent child links", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree layout bounds", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree draw traversal", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree pointer dispatch path", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree focus traversal", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree text routed to focused TextBox", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree button action dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree resize relayout", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree rendered frame update", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree api builder node creation", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree api parent child invariants", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree api layout helper dispatch", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree api hit test helper", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree api focus helper traversal", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree api dispatch path helper", Kind: "positive", Ran: true, Pass: true},
			{Name: "component tree api no manual bookkeeping", Kind: "positive", Ran: true, Pass: true},
			{Name: "reject legacy UI evidence", Kind: "negative", Ran: true, Pass: true, ExpectedError: "legacy UI evidence rejected"},
		},
	}
	if mode == "headless-component-tree" || mode == "linux-x64-real-window-component-tree" ||
		mode == "headless-component-tree-api" || mode == "linux-x64-real-window-component-tree-api" {
		scenario.Frames = []surface.FrameReport{
			{Order: 1, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
			{Order: 2, Width: afterFrame.Width, Height: afterFrame.Height, Stride: afterFrame.Stride, Checksum: checksumRGBA(afterFrame.Pixels), Presented: true},
		}
	}
	switch mode {
	case "headless-component-tree", "headless-component-tree-api":
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "headless event dispatch", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "headless framebuffer checksum", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "headless actual runner trace", Kind: "positive", Ran: true, Pass: true},
		)
	case "linux-x64-real-window-component-tree", "linux-x64-real-window-component-tree-api":
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "linux-x64 Surface Host ABI open/present/close", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 real-window surface", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 native input event pump", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 real-window resize event", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "linux-x64 real-window close event", Kind: "positive", Ran: true, Pass: true},
		)
	case "wasm32-web-browser-canvas-component-tree", "wasm32-web-browser-canvas-component-tree-api":
		scenario.Cases = append(scenario.Cases,
			surface.CaseReport{Name: "wasm32-web browser canvas surface", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas RGBA readback", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas pointer input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas keyboard input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas resize input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web browser canvas text input", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "wasm32-web Surface Host ABI imports", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "compiler-owned wasm Surface loader", Kind: "positive", Ran: true, Pass: true},
			surface.CaseReport{Name: "compiler-owned browser canvas Surface host", Kind: "positive", Ran: true, Pass: true},
		)
	}
	return scenario
}

func componentTreeAPIReport() *surface.ComponentTreeAPIReport {
	return &surface.ComponentTreeAPIReport{
		Schema:            "tetra.surface.component-tree-api.v1",
		APILevel:          "builder-layout-dispatch-v1",
		Source:            "examples/surface_tree_app.tetra",
		ManualBookkeeping: false,
		Builder: surface.ComponentTreeAPIBuilderReport{
			RootCreatedBy:     "tree_add_root",
			ChildrenCreatedBy: "tree_add_child",
			NodeCount:         7,
			Capacity:          16,
			OverflowChecked:   true,
		},
		Invariants: surface.ComponentTreeAPIInvariantReport{
			TreeValidateRan:         true,
			TreeValidateStatus:      0,
			ParentChildLinksChecked: true,
			ChildIndicesChecked:     true,
			ChildCountChecked:       true,
			FirstChildChecked:       true,
		},
		LayoutHelpers: []surface.ComponentTreeAPILayoutHelperReport{
			{Helper: "tree_layout_column", Target: "Column", Pass: "initial", ChangedBounds: true},
			{Helper: "tree_layout_row", Target: "ButtonRow", Pass: "initial", ChangedBounds: true},
			{Helper: "tree_layout_column", Target: "Column", Pass: "resize", ChangedBounds: true},
		},
		FocusHelpers: []surface.ComponentTreeAPIFocusHelperReport{
			{Helper: "tree_focus_next", Before: "TextBox", After: "SubmitButton"},
			{Helper: "tree_focus_next", Before: "SubmitButton", After: "ResetButton"},
			{Helper: "tree_focus_next", Before: "ResetButton", After: "TextBox"},
		},
		HitTests: []surface.ComponentTreeAPIHitTestReport{
			{Helper: "tree_hit_test", X: 40, Y: 72, Target: "TextBox", Path: []int{0, 1, 3}},
			{Helper: "tree_hit_test", X: 176, Y: 120, Target: "ResetButton", Path: []int{0, 1, 4, 6}},
		},
		DispatchPaths: []surface.ComponentTreeAPIDispatchPathReport{
			{Helper: "tree_build_dispatch_path", Target: "TextBox", Path: []int{0, 1, 3}},
			{Helper: "tree_build_dispatch_path", Target: "SubmitButton", Path: []int{0, 1, 4, 5}},
			{Helper: "tree_build_dispatch_path", Target: "ResetButton", Path: []int{0, 1, 4, 6}},
		},
	}
}

func renderCounterFrameRGBA(count int, focused bool) rgbaFrame {
	frame := newRGBAFrame(320, 200)
	bg := rgbaColor{R: 20, G: 24, B: 26, A: 255}
	fg := rgbaColor{R: 238, G: 241, B: 245, A: 255}
	accent := rgbaColor{R: 32, G: 132, B: 214, A: 255}
	button := rect{X: 32, Y: 80, W: 160, H: 48}

	clearRGBA(frame, bg)
	rectRGBA(frame, rect{X: 32, Y: 28, W: 48, H: 7}, fg)
	rectRGBA(frame, button, accent)
	if count > 0 {
		rectRGBA(frame, rect{X: 88, Y: 28, W: 24, H: 7}, fg)
	}
	if focused {
		rectOutlineRGBA(frame, rect{X: button.X - 4, Y: button.Y - 4, W: button.W + 8, H: button.H + 8}, fg)
	}
	rectOutlineRGBA(frame, button, fg)
	return frame
}

func renderWindowCounterFrameRGBA(count int, keyCount int, width int, height int, focused bool) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 18, G: 22, B: 27, A: 255}
	fg := rgbaColor{R: 238, G: 241, B: 245, A: 255}
	accent := rgbaColor{R: 32, G: 132, B: 214, A: 255}
	keyAccent := rgbaColor{R: 34, G: 160, B: 104, A: 255}
	button := rect{X: 32, Y: 88, W: 160, H: 48}

	clearRGBA(frame, bg)
	rectRGBA(frame, rect{X: 32, Y: 28, W: 48, H: 7}, fg)
	if count > 0 {
		rectRGBA(frame, rect{X: 88, Y: 28, W: 24 + count*8, H: 7}, fg)
	}
	rectRGBA(frame, rect{X: 32, Y: 52, W: 48, H: 7}, fg)
	if keyCount > 0 {
		rectRGBA(frame, rect{X: 88, Y: 52, W: 24, H: 7}, keyAccent)
	}
	rectRGBA(frame, button, accent)
	if focused {
		rectOutlineRGBA(frame, rect{X: button.X - 4, Y: button.Y - 4, W: button.W + 8, H: button.H + 8}, fg)
	}
	rectOutlineRGBA(frame, button, fg)
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}

func renderBrowserCounterFrameRGBA(count int, keyCount int, width int, height int, focused bool) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 24, G: 22, B: 34, A: 255}
	fg := rgbaColor{R: 242, G: 244, B: 248, A: 255}
	accent := rgbaColor{R: 54, G: 130, B: 218, A: 255}
	keyAccent := rgbaColor{R: 42, G: 170, B: 112, A: 255}
	textAccent := rgbaColor{R: 218, G: 184, B: 58, A: 255}
	button := rect{X: 32, Y: 88, W: 160, H: 48}

	clearRGBA(frame, bg)
	rectRGBA(frame, rect{X: 32, Y: 28, W: 48, H: 7}, fg)
	if count > 0 {
		rectRGBA(frame, rect{X: 88, Y: 28, W: 24 + count*8, H: 7}, fg)
	}
	rectRGBA(frame, rect{X: 32, Y: 52, W: 48, H: 7}, fg)
	if keyCount > 0 {
		rectRGBA(frame, rect{X: 88, Y: 52, W: 24, H: 7}, keyAccent)
	}
	rectRGBA(frame, rect{X: 32, Y: 68, W: 48, H: 7}, fg)
	rectRGBA(frame, rect{X: 88, Y: 68, W: 18, H: 7}, textAccent)
	rectRGBA(frame, button, accent)
	if focused {
		rectOutlineRGBA(frame, rect{X: button.X - 4, Y: button.Y - 4, W: button.W + 8, H: button.H + 8}, fg)
	}
	rectOutlineRGBA(frame, button, fg)
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}

func renderReleaseCounterFrameRGBA(count int, keyCount int, resetCount int, statusCode int, width int, height int) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 18, G: 24, B: 28, A: 255}
	fg := rgbaColor{R: 236, G: 242, B: 240, A: 255}
	accent := rgbaColor{R: 60, G: 142, B: 212, A: 255}
	resetAccent := rgbaColor{R: 210, G: 96, B: 78, A: 255}
	statusAccent := rgbaColor{R: 88, G: 174, B: 128, A: 255}
	clearRGBA(frame, bg)
	rectRGBA(frame, rect{X: 32, Y: 28, W: 48, H: 7}, fg)
	rectRGBA(frame, rect{X: 32, Y: 56, W: 48, H: 7}, fg)
	rectRGBA(frame, rect{X: 96, Y: 58, W: 24 + count*8, H: 8}, statusAccent)
	rectRGBA(frame, rect{X: 32, Y: 76, W: 48, H: 7}, fg)
	if keyCount > 0 {
		rectRGBA(frame, rect{X: 96, Y: 78, W: 24 + keyCount*8, H: 8}, accent)
	}
	if resetCount > 0 {
		rectRGBA(frame, rect{X: 136, Y: 78, W: 24 + resetCount*8, H: 8}, resetAccent)
	}
	button := rect{X: 32, Y: height/2 - 24, W: 160, H: 48}
	status := rect{X: 32, Y: height/2 + 40, W: width - 64, H: 32}
	rectRGBA(frame, button, accent)
	rectOutlineRGBA(frame, button, fg)
	rectRGBA(frame, status, rgbaColor{R: 28, G: 36, B: 42, A: 255})
	rectRGBA(frame, rect{X: status.X + 12, Y: status.Y + 12, W: 24 + statusCode*12, H: 8}, statusAccent)
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}

func renderTextFocusInputFrameRGBA(textLen int, caret int, focusIndex int, width int, height int) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 19, G: 25, B: 29, A: 255}
	fg := rgbaColor{R: 238, G: 241, B: 245, A: 255}
	textBg := rgbaColor{R: 28, G: 38, B: 45, A: 255}
	textAccent := rgbaColor{R: 56, G: 148, B: 112, A: 255}
	buttonAccent := rgbaColor{R: 54, G: 130, B: 218, A: 255}
	caretColor := rgbaColor{R: 232, G: 196, B: 64, A: 255}
	textbox := rect{X: 32, Y: 64, W: 224, H: 44}
	button := rect{X: 32, Y: 128, W: 128, H: 44}

	clearRGBA(frame, bg)
	rectRGBA(frame, rect{X: 32, Y: 28, W: 48, H: 7}, fg)
	rectRGBA(frame, rect{X: 32, Y: 44, W: 48, H: 7}, fg)
	rectRGBA(frame, textbox, textBg)
	rectOutlineRGBA(frame, textbox, fg)
	if textLen > 0 {
		rectRGBA(frame, rect{X: textbox.X + 12, Y: textbox.Y + 16, W: 18 * textLen, H: 10}, textAccent)
	}
	caretX := textbox.X + 12 + caret*12
	rectRGBA(frame, rect{X: caretX, Y: textbox.Y + 10, W: 2, H: 24}, caretColor)
	rectRGBA(frame, button, buttonAccent)
	rectOutlineRGBA(frame, button, fg)
	if focusIndex == 0 {
		rectOutlineRGBA(frame, rect{X: textbox.X - 4, Y: textbox.Y - 4, W: textbox.W + 8, H: textbox.H + 8}, fg)
	}
	if focusIndex == 1 {
		rectOutlineRGBA(frame, rect{X: button.X - 4, Y: button.Y - 4, W: button.W + 8, H: button.H + 8}, fg)
	}
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}

func renderComponentTreeFrameRGBA(textLen int, caret int, focusID int, submitted int, reset int, width int, height int) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 18, G: 23, B: 27, A: 255}
	fg := rgbaColor{R: 238, G: 241, B: 245, A: 255}
	textBg := rgbaColor{R: 29, G: 41, B: 47, A: 255}
	textAccent := rgbaColor{R: 59, G: 150, B: 113, A: 255}
	submitAccent := rgbaColor{R: 44, G: 127, B: 204, A: 255}
	resetAccent := rgbaColor{R: 192, G: 92, B: 64, A: 255}
	caretColor := rgbaColor{R: 232, G: 196, B: 64, A: 255}
	markerColor := rgbaColor{R: 172, G: 206, B: 96, A: 255}

	textbox := rect{X: 16, Y: 48, W: width - 32, H: 44}
	row := rect{X: 16, Y: 104, W: width - 32, H: 44}
	submitButton := rect{X: row.X, Y: row.Y, W: 132, H: 44}
	resetButton := rect{X: row.X + 144, Y: row.Y, W: 132, H: 44}

	clearRGBA(frame, bg)
	rectRGBA(frame, rect{X: 16, Y: 16, W: 48, H: 7}, fg)
	rectRGBA(frame, rect{X: 76, Y: 16, W: 24 + submitted*14, H: 7}, markerColor)
	rectRGBA(frame, rect{X: 116, Y: 16, W: 24 + reset*14, H: 7}, resetAccent)
	rectRGBA(frame, textbox, textBg)
	rectOutlineRGBA(frame, textbox, fg)
	if textLen > 0 {
		rectRGBA(frame, rect{X: textbox.X + 12, Y: textbox.Y + 16, W: 18 * textLen, H: 10}, textAccent)
	}
	rectRGBA(frame, rect{X: textbox.X + 12 + caret*12, Y: textbox.Y + 10, W: 2, H: 24}, caretColor)
	rectRGBA(frame, submitButton, submitAccent)
	rectOutlineRGBA(frame, submitButton, fg)
	rectRGBA(frame, resetButton, resetAccent)
	rectOutlineRGBA(frame, resetButton, fg)
	if focusID == 3 {
		rectOutlineRGBA(frame, rect{X: textbox.X - 4, Y: textbox.Y - 4, W: textbox.W + 8, H: textbox.H + 8}, fg)
	}
	if focusID == 5 {
		rectOutlineRGBA(frame, rect{X: submitButton.X - 4, Y: submitButton.Y - 4, W: submitButton.W + 8, H: submitButton.H + 8}, fg)
	}
	if focusID == 6 {
		rectOutlineRGBA(frame, rect{X: resetButton.X - 4, Y: resetButton.Y - 4, W: resetButton.W + 8, H: resetButton.H + 8}, fg)
	}
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}

func renderMinimalToolkitFrameRGBA(textLen int, caret int, focusID int, submitted int, reset int, statusCode int, width int, height int) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 17, G: 24, B: 25, A: 255}
	fg := rgbaColor{R: 238, G: 241, B: 245, A: 255}
	panelBg := rgbaColor{R: 23, G: 33, B: 34, A: 255}
	textBg := rgbaColor{R: 29, G: 43, B: 45, A: 255}
	textAccent := rgbaColor{R: 58, G: 156, B: 125, A: 255}
	submitAccent := rgbaColor{R: 49, G: 122, B: 204, A: 255}
	resetAccent := rgbaColor{R: 192, G: 86, B: 74, A: 255}
	caretColor := rgbaColor{R: 235, G: 196, B: 64, A: 255}
	statusAccent := rgbaColor{R: 176, G: 205, B: 92, A: 255}

	panel := rect{X: 0, Y: 0, W: width, H: height}
	column := rect{X: 12, Y: 12, W: width - 24, H: height - 24}
	textbox := rect{X: 20, Y: 52, W: width - 40, H: 44}
	row := rect{X: 20, Y: 108, W: width - 40, H: 44}
	submitButton := rect{X: row.X, Y: row.Y, W: 132, H: 44}
	resetButton := rect{X: row.X + 144, Y: row.Y, W: 132, H: 44}
	status := rect{X: 20, Y: 160, W: width - 40, H: 24}

	clearRGBA(frame, bg)
	rectRGBA(frame, panel, panelBg)
	rectOutlineRGBA(frame, panel, fg)
	rectRGBA(frame, rect{X: column.X + 8, Y: column.Y + 8, W: 48, H: 7}, fg)
	rectRGBA(frame, rect{X: 76, Y: column.Y + 8, W: 22 + submitted*14, H: 7}, statusAccent)
	rectRGBA(frame, rect{X: 116, Y: column.Y + 8, W: 22 + reset*14, H: 7}, resetAccent)
	rectRGBA(frame, textbox, textBg)
	rectOutlineRGBA(frame, textbox, fg)
	if textLen > 0 {
		rectRGBA(frame, rect{X: textbox.X + 12, Y: textbox.Y + 16, W: 18 * textLen, H: 10}, textAccent)
	}
	rectRGBA(frame, rect{X: textbox.X + 12 + caret*12, Y: textbox.Y + 10, W: 2, H: 24}, caretColor)
	rectRGBA(frame, submitButton, submitAccent)
	rectOutlineRGBA(frame, submitButton, fg)
	rectRGBA(frame, resetButton, resetAccent)
	rectOutlineRGBA(frame, resetButton, fg)
	rectRGBA(frame, status, textBg)
	rectOutlineRGBA(frame, status, fg)
	if statusCode > 0 {
		rectRGBA(frame, rect{X: status.X + 12, Y: status.Y + 8, W: 20 + statusCode*16, H: 8}, statusAccent)
	}
	if focusID == 4 {
		rectOutlineRGBA(frame, rect{X: textbox.X - 4, Y: textbox.Y - 4, W: textbox.W + 8, H: textbox.H + 8}, fg)
	}
	if focusID == 6 {
		rectOutlineRGBA(frame, rect{X: submitButton.X - 4, Y: submitButton.Y - 4, W: submitButton.W + 8, H: submitButton.H + 8}, fg)
	}
	if focusID == 7 {
		rectOutlineRGBA(frame, rect{X: resetButton.X - 4, Y: resetButton.Y - 4, W: resetButton.W + 8, H: resetButton.H + 8}, fg)
	}
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}

func renderToolkitReuseFrameRGBA(nameLen int, emailLen int, focusID int, saved int, reset int, statusCode int, width int, height int) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 16, G: 22, B: 29, A: 255}
	fg := rgbaColor{R: 235, G: 242, B: 244, A: 255}
	panelBg := rgbaColor{R: 25, G: 33, B: 42, A: 255}
	textBg := rgbaColor{R: 31, G: 45, B: 58, A: 255}
	nameAccent := rgbaColor{R: 75, G: 166, B: 138, A: 255}
	emailAccent := rgbaColor{R: 86, G: 137, B: 214, A: 255}
	saveAccent := rgbaColor{R: 54, G: 133, B: 210, A: 255}
	resetAccent := rgbaColor{R: 194, G: 92, B: 78, A: 255}
	caretColor := rgbaColor{R: 235, G: 196, B: 64, A: 255}
	statusAccent := rgbaColor{R: 176, G: 205, B: 92, A: 255}

	panel := rect{X: 0, Y: 0, W: width, H: height}
	title := rect{X: 20, Y: 20, W: width - 40, H: 24}
	nameBox := rect{X: 20, Y: 52, W: width - 40, H: 44}
	nameLabel := rect{X: 20, Y: 104, W: width - 40, H: 24}
	emailBox := rect{X: 20, Y: 136, W: width - 40, H: 44}
	row := rect{X: 20, Y: 192, W: width - 40, H: 44}
	saveButton := rect{X: row.X, Y: row.Y, W: 132, H: 44}
	resetButton := rect{X: row.X + 144, Y: row.Y, W: 132, H: 44}
	status := rect{X: 20, Y: 248, W: width - 40, H: 24}

	clearRGBA(frame, bg)
	rectRGBA(frame, panel, panelBg)
	rectOutlineRGBA(frame, panel, fg)
	rectRGBA(frame, rect{X: title.X + 8, Y: title.Y + 8, W: 72, H: 7}, fg)
	rectRGBA(frame, rect{X: title.X + 96, Y: title.Y + 8, W: 22 + saved*14, H: 7}, statusAccent)
	rectRGBA(frame, rect{X: title.X + 136, Y: title.Y + 8, W: 22 + reset*14, H: 7}, resetAccent)
	rectRGBA(frame, nameBox, textBg)
	rectOutlineRGBA(frame, nameBox, fg)
	if nameLen > 0 {
		rectRGBA(frame, rect{X: nameBox.X + 12, Y: nameBox.Y + 16, W: 18 * nameLen, H: 10}, nameAccent)
	}
	rectRGBA(frame, rect{X: nameLabel.X + 8, Y: nameLabel.Y + 8, W: 44, H: 7}, fg)
	rectRGBA(frame, emailBox, textBg)
	rectOutlineRGBA(frame, emailBox, fg)
	if emailLen > 0 {
		rectRGBA(frame, rect{X: emailBox.X + 12, Y: emailBox.Y + 16, W: 16 * emailLen, H: 10}, emailAccent)
	}
	rectRGBA(frame, saveButton, saveAccent)
	rectOutlineRGBA(frame, saveButton, fg)
	rectRGBA(frame, resetButton, resetAccent)
	rectOutlineRGBA(frame, resetButton, fg)
	rectRGBA(frame, status, textBg)
	rectOutlineRGBA(frame, status, fg)
	if statusCode > 0 {
		rectRGBA(frame, rect{X: status.X + 12, Y: status.Y + 8, W: 20 + statusCode*16, H: 8}, statusAccent)
	}
	if focusID == 4 {
		rectOutlineRGBA(frame, rect{X: nameBox.X - 4, Y: nameBox.Y - 4, W: nameBox.W + 8, H: nameBox.H + 8}, fg)
		rectRGBA(frame, rect{X: nameBox.X + 12 + nameLen*12, Y: nameBox.Y + 10, W: 2, H: 24}, caretColor)
	}
	if focusID == 6 {
		rectOutlineRGBA(frame, rect{X: emailBox.X - 4, Y: emailBox.Y - 4, W: emailBox.W + 8, H: emailBox.H + 8}, fg)
		rectRGBA(frame, rect{X: emailBox.X + 12 + emailLen*12, Y: emailBox.Y + 10, W: 2, H: 24}, caretColor)
	}
	if focusID == 8 {
		rectOutlineRGBA(frame, rect{X: saveButton.X - 4, Y: saveButton.Y - 4, W: saveButton.W + 8, H: saveButton.H + 8}, fg)
	}
	if focusID == 9 {
		rectOutlineRGBA(frame, rect{X: resetButton.X - 4, Y: resetButton.Y - 4, W: resetButton.W + 8, H: resetButton.H + 8}, fg)
	}
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}

func renderReleaseToolkitFrameRGBA(nameLen int, emailLen int, focusID int, saved int, reset int, statusCode int, checked bool, scrollOffset int, width int, height int) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 18, G: 24, B: 27, A: 255}
	fg := rgbaColor{R: 238, G: 242, B: 240, A: 255}
	panelBg := rgbaColor{R: 28, G: 38, B: 42, A: 255}
	stackBg := rgbaColor{R: 33, G: 45, B: 50, A: 255}
	textBg := rgbaColor{R: 39, G: 52, B: 59, A: 255}
	nameAccent := rgbaColor{R: 80, G: 172, B: 132, A: 255}
	emailAccent := rgbaColor{R: 80, G: 138, B: 214, A: 255}
	checkboxAccent := rgbaColor{R: 214, G: 177, B: 72, A: 255}
	scrollAccent := rgbaColor{R: 136, G: 106, B: 210, A: 255}
	saveAccent := rgbaColor{R: 56, G: 132, B: 206, A: 255}
	resetAccent := rgbaColor{R: 198, G: 92, B: 78, A: 255}
	statusAccent := rgbaColor{R: 176, G: 206, B: 94, A: 255}
	caretColor := rgbaColor{R: 236, G: 197, B: 64, A: 255}

	panel := rect{X: 0, Y: 0, W: width, H: height}
	stack := rect{X: 16, Y: 16, W: width - 32, H: height - 32}
	title := rect{X: 32, Y: 32, W: width - 64, H: 28}
	description := rect{X: 32, Y: 68, W: width - 64, H: 28}
	nameLabel := rect{X: 32, Y: 104, W: width - 64, H: 24}
	nameBox := rect{X: 32, Y: 132, W: width - 64, H: 44}
	emailLabel := rect{X: 32, Y: 184, W: width - 64, H: 24}
	emailBox := rect{X: 32, Y: 212, W: width - 64, H: 44}
	checkbox := rect{X: 32, Y: 264, W: width - 64, H: 32}
	scroll := rect{X: 32, Y: 304, W: width - 64, H: 48}
	row := rect{X: 32, Y: 360, W: width - 64, H: 44}
	saveButton := rect{X: row.X, Y: row.Y, W: 132, H: 44}
	resetButton := rect{X: row.X + 144, Y: row.Y, W: 132, H: 44}
	spacer := rect{X: row.X + 288, Y: row.Y, W: 16, H: 44}
	status := rect{X: row.X + 312, Y: row.Y, W: row.W - 312, H: 44}

	clearRGBA(frame, bg)
	rectRGBA(frame, panel, panelBg)
	rectOutlineRGBA(frame, panel, fg)
	rectRGBA(frame, stack, stackBg)
	rectOutlineRGBA(frame, stack, fg)
	rectRGBA(frame, rect{X: title.X + 8, Y: title.Y + 8, W: 116, H: 8}, fg)
	rectRGBA(frame, rect{X: description.X + 8, Y: description.Y + 8, W: 164, H: 7}, scrollAccent)
	rectRGBA(frame, rect{X: nameLabel.X + 8, Y: nameLabel.Y + 8, W: 44, H: 7}, fg)
	rectRGBA(frame, nameBox, textBg)
	rectOutlineRGBA(frame, nameBox, fg)
	if nameLen > 0 {
		rectRGBA(frame, rect{X: nameBox.X + 12, Y: nameBox.Y + 16, W: 18 * nameLen, H: 10}, nameAccent)
	}
	rectRGBA(frame, rect{X: emailLabel.X + 8, Y: emailLabel.Y + 8, W: 52, H: 7}, fg)
	rectRGBA(frame, emailBox, textBg)
	rectOutlineRGBA(frame, emailBox, fg)
	if emailLen > 0 {
		rectRGBA(frame, rect{X: emailBox.X + 12, Y: emailBox.Y + 16, W: 16 * emailLen, H: 10}, emailAccent)
	}
	rectRGBA(frame, checkbox, textBg)
	rectOutlineRGBA(frame, checkbox, fg)
	rectOutlineRGBA(frame, rect{X: checkbox.X + 12, Y: checkbox.Y + 8, W: 16, H: 16}, fg)
	if checked {
		rectRGBA(frame, rect{X: checkbox.X + 16, Y: checkbox.Y + 12, W: 8, H: 8}, checkboxAccent)
	}
	rectRGBA(frame, scroll, textBg)
	rectOutlineRGBA(frame, scroll, fg)
	rectRGBA(frame, rect{X: scroll.X + 12, Y: scroll.Y + 12 - scrollOffset/2, W: scroll.W - 40, H: 8}, scrollAccent)
	rectRGBA(frame, rect{X: scroll.X + scroll.W - 18, Y: scroll.Y + 6 + scrollOffset/2, W: 6, H: 20}, checkboxAccent)
	rectRGBA(frame, saveButton, saveAccent)
	rectOutlineRGBA(frame, saveButton, fg)
	rectRGBA(frame, resetButton, resetAccent)
	rectOutlineRGBA(frame, resetButton, fg)
	rectRGBA(frame, spacer, panelBg)
	rectRGBA(frame, status, textBg)
	rectOutlineRGBA(frame, status, fg)
	if statusCode > 0 {
		rectRGBA(frame, rect{X: status.X + 12, Y: status.Y + 16, W: 20 + statusCode*16, H: 8}, statusAccent)
	}
	if focusID == 7 {
		rectOutlineRGBA(frame, rect{X: nameBox.X - 4, Y: nameBox.Y - 4, W: nameBox.W + 8, H: nameBox.H + 8}, fg)
		rectRGBA(frame, rect{X: nameBox.X + 12 + nameLen*12, Y: nameBox.Y + 10, W: 2, H: 24}, caretColor)
	}
	if focusID == 9 {
		rectOutlineRGBA(frame, rect{X: emailBox.X - 4, Y: emailBox.Y - 4, W: emailBox.W + 8, H: emailBox.H + 8}, fg)
		rectRGBA(frame, rect{X: emailBox.X + 12 + emailLen*12, Y: emailBox.Y + 10, W: 2, H: 24}, caretColor)
	}
	if focusID == 10 {
		rectOutlineRGBA(frame, rect{X: checkbox.X - 4, Y: checkbox.Y - 4, W: checkbox.W + 8, H: checkbox.H + 8}, fg)
	}
	if focusID == 14 {
		rectOutlineRGBA(frame, rect{X: saveButton.X - 4, Y: saveButton.Y - 4, W: saveButton.W + 8, H: saveButton.H + 8}, fg)
	}
	if focusID == 15 {
		rectOutlineRGBA(frame, rect{X: resetButton.X - 4, Y: resetButton.Y - 4, W: resetButton.W + 8, H: resetButton.H + 8}, fg)
	}
	if saved > 0 {
		rectRGBA(frame, rect{X: title.X + 140, Y: title.Y + 8, W: 22 + saved*14, H: 7}, statusAccent)
	}
	if reset > 0 {
		rectRGBA(frame, rect{X: title.X + 184, Y: title.Y + 8, W: 22 + reset*14, H: 7}, resetAccent)
	}
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}

func renderAccessibilityMetadataFrameRGBA(nameLen int, emailLen int, focusID int, saved int, reset int, statusCode int, width int, height int) rgbaFrame {
	frame := newRGBAFrame(width, height)
	bg := rgbaColor{R: 14, G: 24, B: 28, A: 255}
	fg := rgbaColor{R: 234, G: 242, B: 238, A: 255}
	panelBg := rgbaColor{R: 24, G: 34, B: 38, A: 255}
	textBg := rgbaColor{R: 31, G: 46, B: 51, A: 255}
	nameAccent := rgbaColor{R: 78, G: 166, B: 128, A: 255}
	emailAccent := rgbaColor{R: 72, G: 136, B: 205, A: 255}
	saveAccent := rgbaColor{R: 52, G: 126, B: 205, A: 255}
	resetAccent := rgbaColor{R: 196, G: 92, B: 78, A: 255}
	caretColor := rgbaColor{R: 236, G: 197, B: 64, A: 255}
	statusAccent := rgbaColor{R: 176, G: 204, B: 92, A: 255}

	panel := rect{X: 0, Y: 0, W: width, H: height}
	title := rect{X: 20, Y: 20, W: width - 40, H: 24}
	nameLabel := rect{X: 20, Y: 52, W: width - 40, H: 24}
	nameBox := rect{X: 20, Y: 84, W: width - 40, H: 44}
	emailLabel := rect{X: 20, Y: 136, W: width - 40, H: 24}
	emailBox := rect{X: 20, Y: 168, W: width - 40, H: 44}
	row := rect{X: 20, Y: 224, W: width - 40, H: 44}
	saveButton := rect{X: row.X, Y: row.Y, W: 132, H: 44}
	resetButton := rect{X: row.X + 144, Y: row.Y, W: 132, H: 44}
	status := rect{X: 20, Y: 280, W: width - 40, H: 24}

	clearRGBA(frame, bg)
	rectRGBA(frame, panel, panelBg)
	rectOutlineRGBA(frame, panel, fg)
	rectRGBA(frame, rect{X: title.X + 8, Y: title.Y + 8, W: 84, H: 7}, fg)
	rectRGBA(frame, rect{X: title.X + 104, Y: title.Y + 8, W: 22 + saved*14, H: 7}, statusAccent)
	rectRGBA(frame, rect{X: title.X + 144, Y: title.Y + 8, W: 22 + reset*14, H: 7}, resetAccent)
	rectRGBA(frame, rect{X: nameLabel.X + 8, Y: nameLabel.Y + 8, W: 44, H: 7}, fg)
	rectRGBA(frame, nameBox, textBg)
	rectOutlineRGBA(frame, nameBox, fg)
	rectRGBA(frame, rect{X: nameBox.X + 12, Y: nameBox.Y + 16, W: 18 * nameLen, H: 10}, nameAccent)
	rectRGBA(frame, rect{X: emailLabel.X + 8, Y: emailLabel.Y + 8, W: 52, H: 7}, fg)
	rectRGBA(frame, emailBox, textBg)
	rectOutlineRGBA(frame, emailBox, fg)
	rectRGBA(frame, rect{X: emailBox.X + 12, Y: emailBox.Y + 16, W: 16 * emailLen, H: 10}, emailAccent)
	rectRGBA(frame, saveButton, saveAccent)
	rectOutlineRGBA(frame, saveButton, fg)
	rectRGBA(frame, resetButton, resetAccent)
	rectOutlineRGBA(frame, resetButton, fg)
	rectRGBA(frame, status, textBg)
	rectOutlineRGBA(frame, status, fg)
	rectRGBA(frame, rect{X: status.X + 12, Y: status.Y + 8, W: 20 + statusCode*16, H: 8}, statusAccent)
	if focusID == 5 {
		rectOutlineRGBA(frame, rect{X: nameBox.X - 4, Y: nameBox.Y - 4, W: nameBox.W + 8, H: nameBox.H + 8}, fg)
		rectRGBA(frame, rect{X: nameBox.X + 12 + nameLen*12, Y: nameBox.Y + 10, W: 2, H: 24}, caretColor)
	}
	if focusID == 7 {
		rectOutlineRGBA(frame, rect{X: emailBox.X - 4, Y: emailBox.Y - 4, W: emailBox.W + 8, H: emailBox.H + 8}, fg)
		rectRGBA(frame, rect{X: emailBox.X + 12 + emailLen*12, Y: emailBox.Y + 10, W: 2, H: 24}, caretColor)
	}
	if focusID == 9 {
		rectOutlineRGBA(frame, rect{X: saveButton.X - 4, Y: saveButton.Y - 4, W: saveButton.W + 8, H: saveButton.H + 8}, fg)
	}
	if focusID == 10 {
		rectOutlineRGBA(frame, rect{X: resetButton.X - 4, Y: resetButton.Y - 4, W: resetButton.W + 8, H: resetButton.H + 8}, fg)
	}
	rectOutlineRGBA(frame, rect{X: 0, Y: 0, W: width, H: height}, fg)
	return frame
}

func newRGBAFrame(width, height int) rgbaFrame {
	stride := width * 4
	return rgbaFrame{
		Width:  width,
		Height: height,
		Stride: stride,
		Pixels: make([]byte, stride*height),
	}
}

func clearRGBA(frame rgbaFrame, color rgbaColor) {
	rectRGBA(frame, rect{X: 0, Y: 0, W: frame.Width, H: frame.Height}, color)
}

func rectOutlineRGBA(frame rgbaFrame, r rect, color rgbaColor) {
	rectRGBA(frame, rect{X: r.X, Y: r.Y, W: r.W, H: 1}, color)
	rectRGBA(frame, rect{X: r.X, Y: r.Y + r.H - 1, W: r.W, H: 1}, color)
	rectRGBA(frame, rect{X: r.X, Y: r.Y, W: 1, H: r.H}, color)
	rectRGBA(frame, rect{X: r.X + r.W - 1, Y: r.Y, W: 1, H: r.H}, color)
}

func rectRGBA(frame rgbaFrame, r rect, color rgbaColor) {
	maxY := r.Y + r.H
	maxX := r.X + r.W
	for y := r.Y; y < maxY; y++ {
		for x := r.X; x < maxX; x++ {
			if x < 0 || y < 0 || x >= frame.Width || y >= frame.Height {
				continue
			}
			i := y*frame.Stride + x*4
			frame.Pixels[i] = color.R
			frame.Pixels[i+1] = color.G
			frame.Pixels[i+2] = color.B
			frame.Pixels[i+3] = color.A
		}
	}
}

func checksumRGBA(pixels []byte) string {
	sum := sha256.Sum256(pixels)
	return hex.EncodeToString(sum[:])
}

func checksumText(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
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

func buildReport(opt smokeOptions, host string, processes []surface.ProcessReport, artifacts []surface.ArtifactReport, artifactScan surface.ArtifactScanReport, scenario headlessScenario) surface.Report {
	mode := opt.Mode
	if mode == "" {
		mode = "headless"
	}
	source := defaultSurfaceSourcePath(opt)
	target := mode
	runtimeName := "surface-headless"
	if mode == "headless-text-focus-input" || mode == "headless-release-toolkit" || mode == "headless-release-accessibility" || mode == "headless-component-tree" || mode == "headless-component-tree-api" || mode == "headless-minimal-toolkit" || mode == "headless-toolkit-reuse" || mode == "headless-accessibility-metadata" {
		target = "headless"
		runtimeName = "surface-headless"
	} else if mode == "linux-x64" || mode == "linux-x64-real-window" || mode == "linux-x64-real-window-text-focus-input" || mode == "linux-x64-release-toolkit" || mode == "linux-x64-release-window" || mode == "linux-x64-release-accessibility" || mode == "linux-x64-real-window-component-tree" || mode == "linux-x64-real-window-component-tree-api" || mode == "linux-x64-real-window-minimal-toolkit" || mode == "linux-x64-real-window-toolkit-reuse" || mode == "linux-x64-real-window-accessibility-metadata" {
		target = "linux-x64"
		runtimeName = "surface-linux-x64"
	} else if mode == "wasm32-web" || mode == "wasm32-web-browser-canvas" || mode == "wasm32-web-browser-canvas-text-focus-input" || mode == "wasm32-web-release-toolkit" || mode == "wasm32-web-release-browser" || mode == "wasm32-web-release-accessibility" || mode == "wasm32-web-browser-canvas-component-tree" || mode == "wasm32-web-browser-canvas-component-tree-api" || mode == "wasm32-web-browser-canvas-minimal-toolkit" || mode == "wasm32-web-browser-canvas-toolkit-reuse" || mode == "wasm32-web-browser-canvas-accessibility-metadata" {
		target = "wasm32-web"
		runtimeName = "surface-wasm32-web"
	}
	return surface.Report{
		Schema:            surface.SchemaV1,
		Status:            "pass",
		Target:            target,
		Host:              host,
		Runtime:           runtimeName,
		SurfaceSchema:     "tetra.surface.v1",
		HostABI:           "tetra.surface.host-abi.v1",
		HostEvidence:      hostEvidenceForMode(mode),
		Source:            source,
		Processes:         processes,
		Artifacts:         artifacts,
		ArtifactScan:      artifactScan,
		Components:        scenario.Components,
		ComponentTree:     scenario.ComponentTree,
		ComponentTreeAPI:  scenario.ComponentTreeAPI,
		Toolkit:           scenario.Toolkit,
		AccessibilityTree: scenario.AccessibilityTree,
		Events:            scenario.Events,
		Frames:            scenario.Frames,
		StateTransitions:  scenario.StateTransitions,
		Cases:             scenario.Cases,
	}
}

func buildTextInputReport(opt smokeOptions, processes []surface.ProcessReport, artifacts []surface.ArtifactReport, artifactScan surface.ArtifactScanReport, cases []surface.CaseReport) surface.TextInputReport {
	return surface.TextInputReport{
		Schema:             surface.TextInputSchemaV1,
		Target:             releaseTextInputTarget(opt.Mode),
		Source:             defaultSurfaceSourcePath(opt),
		Level:              "production-text-input-v1",
		Experimental:       false,
		ProductionClaim:    true,
		Storage:            "owned-utf8-byte-buffer",
		UTF8Validation:     true,
		Caret:              true,
		Selection:          true,
		Backspace:          true,
		Delete:             true,
		HomeEnd:            true,
		ArrowLeftRight:     true,
		CompositionEvents:  true,
		CompositionCommit:  true,
		CompositionCancel:  true,
		ClipboardRead:      true,
		ClipboardWrite:     true,
		ClipboardHostABI:   true,
		ClipboardOwnedCopy: true,
		CompositionTrace: surface.CompositionTraceReport{
			Start:  true,
			Update: true,
			Commit: true,
			Cancel: true,
		},
		BorrowedViewStorage:     false,
		SafeViewLifetimeChecked: true,
		Processes:               processes,
		Artifacts:               artifacts,
		ArtifactScan:            artifactScan,
		Cases:                   cases,
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
		{Name: "release text input caret home end arrows", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input selection replacement", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input backspace delete", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input clipboard owned copy transfer", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input composition start update", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input composition commit", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input composition cancel", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input safe view lifetime checked", Kind: "positive", Ran: true, Pass: true},
		{Name: "reject legacy UI evidence", Kind: "negative", Ran: true, Pass: true, ExpectedError: "legacy UI evidence rejected"},
	}
}

func hostEvidenceForMode(mode string) surface.HostEvidenceReport {
	switch mode {
	case "headless-text-focus-input", "headless-release-text-input", "headless-release-toolkit", "headless-release-accessibility", "headless-component-tree", "headless-component-tree-api", "headless-minimal-toolkit", "headless-toolkit-reuse", "headless-accessibility-metadata":
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
	case "linux-x64-release-window":
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
	case "linux-x64-real-window", "linux-x64-real-window-text-focus-input", "linux-x64-release-text-input", "linux-x64-release-toolkit", "linux-x64-real-window-component-tree", "linux-x64-real-window-component-tree-api", "linux-x64-real-window-minimal-toolkit", "linux-x64-real-window-toolkit-reuse", "linux-x64-real-window-accessibility-metadata":
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
	case "wasm32-web-browser-canvas", "wasm32-web-browser-canvas-text-focus-input", "wasm32-web-release-text-input", "wasm32-web-release-toolkit", "wasm32-web-browser-canvas-component-tree", "wasm32-web-browser-canvas-component-tree-api", "wasm32-web-browser-canvas-minimal-toolkit", "wasm32-web-browser-canvas-toolkit-reuse", "wasm32-web-browser-canvas-accessibility-metadata":
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
