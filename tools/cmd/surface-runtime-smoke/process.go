package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"tetra_language/compiler"
	"tetra_language/tools/validators/surface"
)

func collectSurfaceProcessEvidence(opt smokeOptions) (surfaceProcessEvidence, error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return surfaceProcessEvidence{}, fmt.Errorf("Surface smoke currently requires a linux/amd64 host to build and run linux-x64 Surface app evidence; host is %s/%s", runtime.GOOS, runtime.GOARCH)
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
	if mode == "wasm32-web-browser-canvas-block-system" {
		return collectWASM32WebBrowserCanvasProcessEvidence(sourcePath, artifactDir, "block-system")
	}
	if isWASM32WebBrowserCanvasMorphMode(mode) {
		return collectWASM32WebBrowserCanvasProcessEvidence(sourcePath, artifactDir, "studio-shell")
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
	if _, err := compiler.BuildFileWithStatsOpt(sourcePath, appPath, "linux-x64", surfaceSmokeBuildOptions(sourcePath)); err != nil {
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
	expectedAppExit := surfaceComponentAppExpectedExitForSource(mode, sourcePath)
	if appExit != expectedAppExit {
		return surfaceProcessEvidence{}, fmt.Errorf("run Surface app %s: exit code %d, want %d", appPath, appExit, expectedAppExit)
	}

	processes := []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: fmt.Sprintf("tetra build --target linux-x64 %s -o %s", sourcePath, appPath), Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: appPath, Ran: true, Pass: true, ExitCode: intPtr(appExit), ExpectedExitCode: intPtr(expectedAppExit)},
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
	if mode == "linux-x64-release-app-shell" {
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
		appShellProcesses, appShellArtifacts, err := collectLinuxAppShellTraceEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, appShellProcesses...)
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
		artifacts := append([]surface.ArtifactReport{componentArtifact}, appShellArtifacts...)
		artifacts = append(artifacts, harnessArtifacts...)
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
	if mode == "linux-x64-real-window-block-system" {
		runtimeProcessName = "surface linux-x64 real-window runtime"
		probeProcesses, err := collectLinuxX64HostProbeEvidence(artifactDir)
		if err != nil {
			return surfaceProcessEvidence{}, err
		}
		processes = append(processes, probeProcesses...)
		realWindowProcess, realWindowFrame, err := collectLinuxX64BlockSystemRealWindowProbeEvidence(artifactDir)
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
		processes = append(processes, surface.ProcessReport{Name: runtimeProcessName, Kind: "runtime", Path: os.Args[0], Ran: true, Pass: true, ExitCode: intPtr(0)})
		return surfaceProcessEvidence{Processes: processes, Artifacts: []surface.ArtifactReport{componentArtifact}, ArtifactScan: sidecarScan, Frames: morphFrames}, nil
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
