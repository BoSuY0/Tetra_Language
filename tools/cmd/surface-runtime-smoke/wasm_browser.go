package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"tetra_language/compiler"
	"tetra_language/tools/validators/surface"
)

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
	if scenarioName == "block-system" {
		wasmFile = "surface-block-system.wasm"
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
	if scenarioName == "block-system" {
		before := traceFrames[0]
		if !browserTrace.Canvas.Readback {
			return surfaceProcessEvidence{}, fmt.Errorf("wasm32-web browser-canvas block-system trace missing RGBA readback evidence")
		}
		if after.Order != 5 || after.Width != 400 || after.Height != 240 || after.Stride != 1600 || !after.Presented {
			return surfaceProcessEvidence{}, fmt.Errorf("wasm32-web browser-canvas block-system after-frame = %#v, want order-5 presented 400x240 RGBA frame", after)
		}
		if before.Checksum == after.Checksum {
			return surfaceProcessEvidence{}, fmt.Errorf("wasm32-web browser-canvas block-system frame checksums did not change across browser input/readback: %#v", traceFrames)
		}
		if !browserTraceHasNativeEvents(browserTrace, []string{"pointerup", "keydown", "resize", "beforeinput"}) {
			return surfaceProcessEvidence{}, fmt.Errorf("wasm32-web browser-canvas block-system trace missing required native browser input events: %#v", browserTrace.BrowserEvents)
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
