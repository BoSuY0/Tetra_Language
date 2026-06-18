package main

import (
	"encoding/json"
	"strings"
	"testing"

	"tetra_language/tools/validators/surface"
)

func TestBlockSystemScenarioProducesMemoryBudgetEvidence(t *testing.T) {
	scenario := runBlockSystemScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless-block-system",
		SourcePath: "examples/surface_block_system.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_system.tetra -o /tmp/surface-artifacts/surface-block-system", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-system", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-block-system", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, headlessSurfaceArtifacts("/tmp/surface-artifacts/surface-block-system"), cleanArtifactScan(2), scenario)

	if report.BlockSystem == nil || report.BlockSystem.MemoryBudget == nil {
		t.Fatalf("block_system.memory_budget is nil, want bounded memory/cache evidence")
	}
	budget := report.BlockSystem.MemoryBudget
	if budget.BlockCount != len(report.Components) || budget.StressBlockCount < 128 {
		t.Fatalf("memory budget block counts = %#v, components=%d", budget, len(report.Components))
	}
	if budget.PeakFramebufferBytes != 256000 || budget.TotalFramebufferBytes != 768000 || budget.FramebufferBudgetBytes < budget.PeakFramebufferBytes {
		t.Fatalf("memory budget framebuffer bytes = %#v", budget)
	}
	if !budget.BoundedCaches || !budget.UnboundedCacheRejected || budget.PerformanceClaim != "none" {
		t.Fatalf("memory budget cache/nonclaim evidence = %#v", budget)
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v", err)
	}
}

func TestLinuxX64RealWindowBlockSystemScenarioProducesRealWindowEvidence(t *testing.T) {
	const mode = "linux-x64-real-window-block-system"
	if err := validateSmokeMode(mode); err != nil {
		t.Fatalf("validateSmokeMode(%s) failed: %v", mode, err)
	}
	if got := defaultSurfaceSourcePath(smokeOptions{Mode: mode, SourcePath: "examples/surface_counter.tetra"}); got != "examples/surface_block_system.tetra" {
		t.Fatalf("defaultSurfaceSourcePath(%s) = %q, want examples/surface_block_system.tetra", mode, got)
	}

	scenario := runLinuxX64RealWindowBlockSystemScenario()
	windowFrame := renderBlockSystemFrameSizedRGBA(400, 240, true)
	hostProbeFrame := surface.FrameReport{
		Order:     5,
		Width:     windowFrame.Width,
		Height:    windowFrame.Height,
		Stride:    windowFrame.Stride,
		Checksum:  checksumRGBA(windowFrame.Pixels),
		Presented: true,
	}
	markHostProbeOnlyFrameEvidence(&hostProbeFrame, "/tmp/surface-artifacts/surface-block-system-real-window-frame.rgba")
	scenario.Frames = append(scenario.Frames, hostProbeFrame)
	scenario.BlockSystem = blockSystemReportForLinuxX64RealWindowScenario("examples/surface_block_system.tetra", scenario.Frames)
	attachBlockSystemMemoryBudget(&scenario)
	report := buildReport(smokeOptions{
		Mode:       mode,
		SourcePath: "examples/surface_block_system.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_block_system.tetra -o /tmp/surface-artifacts/surface-block-system", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-system", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
		{Name: "surface linux-x64 real-window probe", Kind: "app", Path: "/tmp/surface-artifacts/surface-block-system-real-window-probe", Ran: true, Pass: true, ExitCode: intPtr(42), ExpectedExitCode: intPtr(42)},
		{Name: "surface linux-x64 runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode linux-x64-real-window-block-system", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, componentAppArtifacts("/tmp/surface-artifacts/surface-block-system"), cleanArtifactScan(1), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.Target != "linux-x64" || report.Runtime != "surface-linux-x64" {
		t.Fatalf("target/runtime = %s/%s, want linux-x64/surface-linux-x64", report.Target, report.Runtime)
	}
	if report.HostEvidence.Level != "linux-x64-real-window" || !report.HostEvidence.RealWindow || !report.HostEvidence.NativeInput {
		t.Fatalf("host evidence = %#v, want linux-x64 real-window native-input evidence", report.HostEvidence)
	}
	if report.BlockSystem == nil {
		t.Fatalf("block_system is nil, want linux-x64 real-window Block evidence")
	}
	if report.BlockSystem.QualityLevel != "linux-x64-real-window-block-system-v1" || report.BlockSystem.Renderer != "wayland-shm-rgba" {
		t.Fatalf("block_system = %#v, want linux-x64 real-window quality and renderer", report.BlockSystem)
	}
	var blockWindowFrame *surface.BlockSystemFrameReport
	for i := range report.BlockSystem.Frames {
		if report.BlockSystem.Frames[i].Order == 5 {
			blockWindowFrame = &report.BlockSystem.Frames[i]
			break
		}
	}
	var runtimeWindowFrame *surface.FrameReport
	for i := range report.Frames {
		if report.Frames[i].Order == 5 {
			runtimeWindowFrame = &report.Frames[i]
			break
		}
	}
	if blockWindowFrame == nil || runtimeWindowFrame == nil || blockWindowFrame.Checksum != runtimeWindowFrame.Checksum {
		t.Fatalf("block_system frames = %#v, report frames = %#v", report.BlockSystem.Frames, report.Frames)
	}
	if !runtimeWindowFrame.Precomputed || runtimeWindowFrame.EvidenceRole != "host_probe_only" || runtimeWindowFrame.Producer != "host_probe" {
		t.Fatalf("runtime real-window frame provenance = %#v, want host_probe_only precomputed infrastructure evidence", runtimeWindowFrame)
	}
	if !blockWindowFrame.Precomputed || blockWindowFrame.EvidenceRole != runtimeWindowFrame.EvidenceRole || blockWindowFrame.Producer != runtimeWindowFrame.Producer {
		t.Fatalf("block_system real-window frame provenance = %#v, want runtime provenance %#v", blockWindowFrame, runtimeWindowFrame)
	}
	for _, want := range []string{
		"linux-x64 real-window surface",
		"linux-x64 native input event pump",
		"block system linux-x64 real-window frame presentation",
		"block system linux-x64 native input state transition",
	} {
		if !caseNamesContain(scenario.Cases, want) {
			t.Fatalf("scenario cases = %#v, want %q", scenario.Cases, want)
		}
	}
}

func TestLinuxX64ModeProducesTargetSpecificRuntimeEvidence(t *testing.T) {
	if err := validateSmokeMode("linux-x64"); err != nil {
		t.Fatalf("validateSmokeMode(linux-x64) failed: %v", err)
	}
	scenario := runLinuxX64CounterScenario()
	scenario.Frames = append(scenario.Frames, surface.FrameReport{
		Order:     3,
		Width:     2,
		Height:    2,
		Stride:    8,
		Checksum:  checksumRGBA(surfacePresentedFrameProbePixels()),
		Presented: true,
	})
	counterFrame := renderCounterFrameRGBA(1, true)
	scenario.Frames = append(scenario.Frames, surface.FrameReport{
		Order:     4,
		Width:     counterFrame.Width,
		Height:    counterFrame.Height,
		Stride:    counterFrame.Stride,
		Checksum:  checksumRGBA(counterFrame.Pixels),
		Presented: true,
	})
	report := buildReport(smokeOptions{
		Mode:       "linux-x64",
		SourcePath: "examples/surface_counter.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_counter.tetra -o /tmp/surface-artifacts/surface-counter", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-counter", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
		{Name: "surface linux-x64 host probe build", Kind: "build", Path: "/tmp/tetra build probe", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface linux-x64 host probe", Kind: "app", Path: "/tmp/surface-host-probe", Ran: true, Pass: true, ExitCode: intPtr(42), ExpectedExitCode: intPtr(42)},
		{Name: "surface linux-x64 event sequence probe build", Kind: "build", Path: "/tmp/tetra build event sequence probe", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface linux-x64 event sequence probe", Kind: "app", Path: "/tmp/surface-event-sequence-probe", Ran: true, Pass: true, ExitCode: intPtr(42), ExpectedExitCode: intPtr(42)},
		{Name: "surface linux-x64 counter app presented frame probe", Kind: "app", Path: "/tmp/surface-artifacts/surface-counter-present-probe", Ran: true, Pass: true, ExitCode: intPtr(-1), ExpectedExitCode: intPtr(-1)},
		{Name: "surface linux-x64 runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, componentAppArtifacts("/tmp/surface-artifacts/surface-counter"), cleanArtifactScan(1), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.HostEvidence.Level != "linux-x64-memfd-starter" || report.HostEvidence.Backend != "memfd-rgba" || !report.HostEvidence.Framebuffer || report.HostEvidence.RealWindow || report.HostEvidence.NativeInput {
		t.Fatalf("linux-x64 host evidence = %#v, want explicit memfd starter evidence without real-window/native-input claim", report.HostEvidence)
	}
	if !caseNamesContain(scenario.Cases, "linux-x64 Surface Host ABI") {
		t.Fatalf("linux-x64 scenario cases = %#v, want target-specific Surface Host ABI evidence", scenario.Cases)
	}
	if !caseNamesContain(scenario.Cases, "no legacy UI sidecar artifacts") {
		t.Fatalf("linux-x64 scenario cases = %#v, want no legacy sidecar evidence", scenario.Cases)
	}
	if !caseNamesContain(scenario.Cases, "linux-x64 app-presented RGBA checksum") {
		t.Fatalf("linux-x64 scenario cases = %#v, want app-presented frame checksum evidence", scenario.Cases)
	}
	if !caseNamesContain(scenario.Cases, "linux-x64 host event sequence") {
		t.Fatalf("linux-x64 scenario cases = %#v, want executable event sequence evidence", scenario.Cases)
	}
	if !caseNamesContain(scenario.Cases, "linux-x64 counter component app-presented frame") {
		t.Fatalf("linux-x64 scenario cases = %#v, want counter component app-presented frame evidence", scenario.Cases)
	}
}

func TestLinuxX64RealWindowModeProducesTargetSpecificRuntimeEvidence(t *testing.T) {
	if err := validateSmokeMode("linux-x64-real-window"); err != nil {
		t.Fatalf("validateSmokeMode(linux-x64-real-window) failed: %v", err)
	}
	scenario := runLinuxX64RealWindowCounterScenario()
	windowFrame := renderWindowCounterFrameRGBA(2, 1, 400, 240, true)
	scenario.Frames = append(scenario.Frames, surface.FrameReport{
		Order:     5,
		Width:     windowFrame.Width,
		Height:    windowFrame.Height,
		Stride:    windowFrame.Stride,
		Checksum:  checksumRGBA(windowFrame.Pixels),
		Presented: true,
	})
	report := buildReport(smokeOptions{
		Mode:       "linux-x64-real-window",
		SourcePath: "examples/surface_window_counter.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_window_counter.tetra -o /tmp/surface-artifacts/surface-window-counter", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-window-counter", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
		{Name: "surface linux-x64 real-window probe", Kind: "app", Path: "/tmp/surface-artifacts/surface-real-window-probe", Ran: true, Pass: true, ExitCode: intPtr(42), ExpectedExitCode: intPtr(42)},
		{Name: "surface linux-x64 runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, componentAppArtifacts("/tmp/surface-artifacts/surface-window-counter"), cleanArtifactScan(1), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.HostEvidence.Level != "linux-x64-real-window" || report.HostEvidence.Backend != "wayland-shm-rgba" || !report.HostEvidence.Framebuffer || !report.HostEvidence.RealWindow || !report.HostEvidence.NativeInput {
		t.Fatalf("linux-x64 real-window host evidence = %#v, want explicit real-window/native-input evidence", report.HostEvidence)
	}
	for _, want := range []string{
		"linux-x64 real-window surface",
		"linux-x64 native input event pump",
		"linux-x64 real-window resize event",
		"linux-x64 real-window close event",
	} {
		if !caseNamesContain(scenario.Cases, want) {
			t.Fatalf("linux-x64 real-window scenario cases = %#v, want %q", scenario.Cases, want)
		}
	}
	for _, want := range []string{"mouse_up", "key_down", "resize", "close"} {
		if !eventKindsContain(scenario.Events, want) {
			t.Fatalf("linux-x64 real-window events = %#v, want %s event evidence", scenario.Events, want)
		}
	}
}

func TestWASM32WebModeProducesTargetSpecificRuntimeEvidence(t *testing.T) {
	if err := validateSmokeMode("wasm32-web"); err != nil {
		t.Fatalf("validateSmokeMode(wasm32-web) failed: %v", err)
	}
	scenario := runWASM32WebCounterScenario()
	beforeFrame := renderCounterFrameRGBA(0, true)
	afterFrame := renderCounterFrameRGBA(1, true)
	scenario.Frames = append(scenario.Frames,
		surface.FrameReport{Order: 3, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
		surface.FrameReport{Order: 4, Width: afterFrame.Width, Height: afterFrame.Height, Stride: afterFrame.Stride, Checksum: checksumRGBA(afterFrame.Pixels), Presented: true},
	)
	report := buildReport(smokeOptions{
		Mode:       "wasm32-web",
		SourcePath: "examples/surface_counter.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target wasm32-web examples/surface_counter.tetra -o /tmp/surface-artifacts/surface-counter.wasm", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface wasm32-web component app", Kind: "app", Path: "node scripts/tools/web_run_module.mjs /tmp/surface-artifacts/surface-counter.wasm", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
		{Name: "surface wasm32-web import validator", Kind: "runtime", Path: "go run ./tools/cmd/validate-wasm-imports --target wasm32-web /tmp/surface-artifacts/surface-counter.wasm", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface wasm32-web runtime", Kind: "runtime", Path: "node --version", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, wasmSurfaceArtifacts(), cleanArtifactScan(3), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.HostEvidence.Level != "wasm32-web-compiler-owned-loader" || report.HostEvidence.Backend != "node-surface-host" || !report.HostEvidence.Framebuffer || report.HostEvidence.RealWindow || report.HostEvidence.NativeInput {
		t.Fatalf("wasm32-web host evidence = %#v, want compiler-owned loader evidence without production browser native-input claim", report.HostEvidence)
	}
	if !caseNamesContain(scenario.Cases, "wasm32-web Surface Host ABI imports") {
		t.Fatalf("wasm32-web scenario cases = %#v, want Surface Host ABI import evidence", scenario.Cases)
	}
	if !caseNamesContain(scenario.Cases, "compiler-owned wasm Surface loader") {
		t.Fatalf("wasm32-web scenario cases = %#v, want compiler-owned loader evidence", scenario.Cases)
	}
	if !caseNamesContain(scenario.Cases, "wasm32-web actual presented frame trace") {
		t.Fatalf("wasm32-web scenario cases = %#v, want actual presented frame trace evidence", scenario.Cases)
	}
	if caseNamesContain(scenario.Cases, "headless") {
		t.Fatalf("wasm32-web scenario cases = %#v, must not reuse headless case names", scenario.Cases)
	}
}

func TestWASM32WebBrowserCanvasModeProducesTargetSpecificRuntimeEvidence(t *testing.T) {
	if err := validateSmokeMode("wasm32-web-browser-canvas"); err != nil {
		t.Fatalf("validateSmokeMode(wasm32-web-browser-canvas) failed: %v", err)
	}
	scenario := runWASM32WebBrowserCanvasCounterScenario()
	beforeFrame := renderBrowserCounterFrameRGBA(0, 0, 320, 200, true)
	afterFrame := renderBrowserCounterFrameRGBA(2, 1, 400, 240, true)
	scenario.Frames = append(scenario.Frames,
		surface.FrameReport{Order: 1, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
		surface.FrameReport{Order: 5, Width: afterFrame.Width, Height: afterFrame.Height, Stride: afterFrame.Stride, Checksum: checksumRGBA(afterFrame.Pixels), Presented: true},
	)
	report := buildReport(smokeOptions{
		Mode:       "wasm32-web-browser-canvas",
		SourcePath: "examples/surface_browser_counter.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target wasm32-web examples/surface_browser_counter.tetra -o /tmp/surface-artifacts/surface-browser-counter.wasm", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface wasm32-web browser canvas component app", Kind: "app", Path: "/usr/bin/chromium --headless <surface-browser-canvas-runner> wasm=/tmp/surface-artifacts/surface-browser-counter.wasm", Ran: true, Pass: true, ExitCode: intPtr(0), ExpectedExitCode: intPtr(0)},
		{Name: "surface wasm32-web import validator", Kind: "runtime", Path: "go run ./tools/cmd/validate-wasm-imports --target wasm32-web /tmp/surface-artifacts/surface-browser-counter.wasm", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface wasm32-web browser canvas runtime", Kind: "runtime", Path: "Chromium test fixture", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface wasm32-web browser canvas trace", Kind: "runtime", Path: "/usr/bin/chromium --headless --dump-dom http://127.0.0.1:1/surface-browser-canvas-runner", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, wasmBrowserCanvasSurfaceArtifacts(), cleanArtifactScan(3), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.HostEvidence.Level != "wasm32-web-browser-canvas-input" || report.HostEvidence.Backend != "browser-canvas-rgba" || !report.HostEvidence.Framebuffer || !report.HostEvidence.NativeInput || report.HostEvidence.RealWindow {
		t.Fatalf("browser canvas host evidence = %#v, want browser canvas framebuffer/native-input without OS real-window claim", report.HostEvidence)
	}
	for _, want := range []string{"mouse_up", "key_down", "resize", "text_input"} {
		if !eventKindsContain(scenario.Events, want) {
			t.Fatalf("browser canvas events = %#v, want %s event evidence", scenario.Events, want)
		}
	}
	for _, want := range []string{
		"wasm32-web browser canvas surface",
		"wasm32-web browser canvas RGBA readback",
		"wasm32-web browser canvas pointer input",
		"wasm32-web browser canvas keyboard input",
		"wasm32-web browser canvas resize input",
		"wasm32-web browser canvas text input",
		"compiler-owned browser canvas Surface host",
	} {
		if !caseNamesContain(scenario.Cases, want) {
			t.Fatalf("browser canvas scenario cases = %#v, want %q", scenario.Cases, want)
		}
	}
	if len(scenario.StateTransitions) != 4 {
		t.Fatalf("browser canvas state transitions = %#v, want click/key/resize/text transitions", scenario.StateTransitions)
	}
}

func TestWASM32WebBrowserCanvasBlockSystemScenarioProducesBrowserEvidence(t *testing.T) {
	const mode = "wasm32-web-browser-canvas-block-system"
	if err := validateSmokeMode(mode); err != nil {
		t.Fatalf("validateSmokeMode(%s) failed: %v", mode, err)
	}
	if got := defaultSurfaceSourcePath(smokeOptions{Mode: mode, SourcePath: "examples/surface_counter.tetra"}); got != "examples/surface_block_system.tetra" {
		t.Fatalf("defaultSurfaceSourcePath(%s) = %q, want examples/surface_block_system.tetra", mode, got)
	}

	scenario := runWASM32WebBrowserCanvasBlockSystemScenario()
	browserFrame := renderBlockSystemFrameSizedRGBA(400, 240, true)
	scenario.Frames = append(scenario.Frames, surface.FrameReport{
		Order:     5,
		Width:     browserFrame.Width,
		Height:    browserFrame.Height,
		Stride:    browserFrame.Stride,
		Checksum:  checksumRGBA(browserFrame.Pixels),
		Presented: true,
	})
	scenario.BlockSystem = blockSystemReportForWASM32WebBrowserCanvasScenario("examples/surface_block_system.tetra", scenario.Frames)
	attachBlockSystemMemoryBudget(&scenario)
	report := buildReport(smokeOptions{
		Mode:       mode,
		SourcePath: "examples/surface_block_system.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target wasm32-web examples/surface_block_system.tetra -o /tmp/surface-artifacts/surface-block-system.wasm", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface wasm32-web browser canvas component app", Kind: "app", Path: "/usr/bin/chromium --headless <surface-browser-canvas-runner> scenario=block-system wasm=/tmp/surface-artifacts/surface-block-system.wasm", Ran: true, Pass: true, ExitCode: intPtr(0), ExpectedExitCode: intPtr(0)},
		{Name: "surface wasm32-web import validator", Kind: "runtime", Path: "go run ./tools/cmd/validate-wasm-imports --target wasm32-web /tmp/surface-artifacts/surface-block-system.wasm", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface wasm32-web browser canvas runtime", Kind: "runtime", Path: "Chromium Block-system fixture", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface wasm32-web browser canvas trace", Kind: "runtime", Path: "/usr/bin/chromium --headless --dump-dom <surface-browser-canvas-file-runner scenario=block-system>", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, []surface.ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-block-system.wasm", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 8604},
		{Kind: "compiler-owned-loader", Path: "/tmp/surface-artifacts/surface-block-system.mjs", SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Size: 4939},
		{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 1184},
	}, cleanArtifactScan(3), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.BlockSystem == nil || report.BlockSystem.QualityLevel != "wasm32-web-browser-canvas-block-system-v1" {
		t.Fatalf("block_system = %#v, want wasm32-web browser-canvas Block system evidence", report.BlockSystem)
	}
	if report.HostEvidence.Level != "wasm32-web-browser-canvas-input" || !report.HostEvidence.BrowserCanvas || !report.HostEvidence.BrowserInput || report.HostEvidence.RealWindow {
		t.Fatalf("host evidence = %#v, want browser-canvas input without OS real-window claim", report.HostEvidence)
	}
	for _, want := range []string{
		"wasm32-web browser canvas surface",
		"wasm32-web browser canvas RGBA readback",
		"block system wasm32-web browser-canvas frame readback",
		"block system wasm32-web browser-canvas native input state transition",
		"block system browser-canvas script sidecar artifact rejected",
		"block system browser-canvas html visual sidecar artifact rejected",
	} {
		if !caseNamesContain(report.Cases, want) {
			t.Fatalf("scenario cases = %#v, want %q", report.Cases, want)
		}
	}
}

func TestTextFocusInputModesProduceTextboxEvidence(t *testing.T) {
	for _, mode := range []string{
		"headless-text-focus-input",
		"linux-x64-real-window-text-focus-input",
		"wasm32-web-browser-canvas-text-focus-input",
	} {
		t.Run(mode, func(t *testing.T) {
			if err := validateSmokeMode(mode); err != nil {
				t.Fatalf("validateSmokeMode(%s) failed: %v", mode, err)
			}
			if got := defaultSurfaceSourcePath(smokeOptions{Mode: mode, SourcePath: "examples/surface_counter.tetra"}); got != "examples/surface_textbox_app.tetra" {
				t.Fatalf("defaultSurfaceSourcePath(%s) = %q, want examples/surface_textbox_app.tetra", mode, got)
			}
			scenario := runTextFocusInputScenario(mode)
			if len(scenario.Components) != 3 {
				t.Fatalf("components = %#v, want app, TextBox, and Button evidence", scenario.Components)
			}
			if scenario.Components[1].ID != "TextBox" || scenario.Components[2].ID != "SubmitButton" {
				t.Fatalf("components = %#v, want TextBox and SubmitButton child components", scenario.Components)
			}
			if scenario.Components[1].State["buffer"] != "Z" || scenario.Components[1].State["caret"] != "1" || scenario.Components[2].State["press_count"] != "1" {
				t.Fatalf("component state = %#v, want edited TextBox buffer/caret and focused Button press evidence", scenario.Components)
			}
			for _, want := range []string{
				"mouse_up",
				"text_input",
				"key_down",
				"resize",
			} {
				if !eventKindsContain(scenario.Events, want) {
					t.Fatalf("events = %#v, want %s evidence", scenario.Events, want)
				}
			}
			for _, want := range []string{
				"text focus input click focuses TextBox",
				"text focus input Tab changes focus",
				"text focus input keyboard routes only focused component",
				"text focus input text insertion",
				"text focus input caret movement",
				"text focus input backspace delete",
				"text focus input resize preserves focus",
				"text focus input rendered frame update",
			} {
				if !caseNamesContain(scenario.Cases, want) {
					t.Fatalf("cases = %#v, want %q", scenario.Cases, want)
				}
			}
			reportScenario := scenario
			reportScenario.Frames = append(reportScenario.Frames, textFocusInputTestFrames(mode)...)
			if len(reportScenario.Frames) < 2 || reportScenario.Frames[0].Checksum == reportScenario.Frames[len(reportScenario.Frames)-1].Checksum {
				t.Fatalf("frames = %#v, want visible framebuffer update after text/focus changes", reportScenario.Frames)
			}
			raw, err := json.Marshal(buildReport(smokeOptions{Mode: mode, SourcePath: "examples/surface_textbox_app.tetra"}, "linux-x64", textFocusInputTestProcesses(mode), textFocusInputTestArtifacts(mode), cleanArtifactScan(3), reportScenario))
			if err != nil {
				t.Fatalf("marshal text focus input report: %v", err)
			}
			if err := surface.ValidateReport(raw); err != nil {
				t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
			}
		})
	}
}

func TestReleaseTextInputModesProduceProductionTextInputReports(t *testing.T) {
	for _, tc := range []struct {
		mode       string
		wantTarget string
	}{
		{mode: "headless-release-text-input", wantTarget: "headless"},
		{mode: "linux-x64-release-text-input", wantTarget: "linux-x64"},
		{mode: "wasm32-web-release-text-input", wantTarget: "wasm32-web"},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			if err := validateSmokeMode(tc.mode); err != nil {
				t.Fatalf("validateSmokeMode(%s) failed: %v", tc.mode, err)
			}
			opt := smokeOptions{Mode: tc.mode, SourcePath: "examples/surface_counter.tetra"}
			if got := defaultSurfaceSourcePath(opt); got != "examples/surface_release_text_input.tetra" {
				t.Fatalf("defaultSurfaceSourcePath(%s) = %q, want examples/surface_release_text_input.tetra", tc.mode, got)
			}
			report := buildTextInputReport(
				opt,
				releaseTextInputTestProcesses(tc.mode),
				releaseTextInputTestArtifacts(tc.mode),
				cleanArtifactScan(releaseTextInputTestArtifactCount(tc.mode)),
				releaseTextInputTestCases(),
			)
			if report.Target != tc.wantTarget {
				t.Fatalf("target = %q, want %q", report.Target, tc.wantTarget)
			}
			raw, err := json.Marshal(report)
			if err != nil {
				t.Fatalf("marshal release text-input report: %v", err)
			}
			if err := surface.ValidateTextInputReport(raw); err != nil {
				t.Fatalf("ValidateTextInputReport failed: %v\n%s", err, raw)
			}
		})
	}
}

func TestHeadlessAppModelModeProducesCommandReducerEvidence(t *testing.T) {
	const mode = "headless-app-model"
	if err := validateSmokeMode(mode); err != nil {
		t.Fatalf("validateSmokeMode(%s) failed: %v", mode, err)
	}
	opt := smokeOptions{Mode: mode, SourcePath: "examples/surface_counter.tetra"}
	if got := defaultSurfaceSourcePath(opt); got != "examples/surface_app_model.tetra" {
		t.Fatalf("defaultSurfaceSourcePath(%s) = %q, want examples/surface_app_model.tetra", mode, got)
	}
	scenario := runSurfaceScenario(mode)
	if scenario.AppModel == nil {
		t.Fatalf("scenario.AppModel is nil, want app-model command/reducer evidence")
	}
	if scenario.AppModel.Schema != "tetra.surface.app-model.v1" || scenario.AppModel.AppModelLevel != "explicit-command-reducer-v1" {
		t.Fatalf("app_model = %#v, want app-model v1 explicit command reducer evidence", scenario.AppModel)
	}
	report := buildReport(opt, "linux-x64", []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_app_model.tetra -o /tmp/surface-artifacts/surface-app-model", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-app-model", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
		{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-app-model", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}, headlessSurfaceArtifacts("/tmp/surface-artifacts/surface-app-model"), cleanArtifactScan(2), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal app-model report: %v", err)
	}
	if report.Target != "headless" || report.Runtime != "surface-headless" {
		t.Fatalf("target/runtime = %q/%q, want headless/surface-headless", report.Target, report.Runtime)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	for _, want := range []string{
		"app model explicit event-to-command binding",
		"app model deterministic command reducer",
		"app model navigation stack",
		"app model focus scope modal trap",
		"app model async completion cancellation boundary",
		"app model undo redo history",
		"app model no React hooks DOM event model hidden JS state",
	} {
		if !caseNamesContain(report.Cases, want) {
			t.Fatalf("cases = %#v, want %q", report.Cases, want)
		}
	}
}

func TestReleaseCounterSourceCanBackLinuxAndBrowserCounterReports(t *testing.T) {
	for _, tc := range []struct {
		mode       string
		wantTarget string
	}{
		{mode: "linux-x64-real-window", wantTarget: "linux-x64"},
		{mode: "wasm32-web-browser-canvas", wantTarget: "wasm32-web"},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			opt := smokeOptions{Mode: tc.mode, SourcePath: "examples/surface_release_counter.tetra"}
			if got := defaultSurfaceSourcePath(opt); got != "examples/surface_release_counter.tetra" {
				t.Fatalf("defaultSurfaceSourcePath(%s) = %q, want examples/surface_release_counter.tetra", tc.mode, got)
			}
			scenario := releaseCounterScenarioForSource(opt, runSurfaceScenario(tc.mode))
			switch tc.mode {
			case "linux-x64-real-window":
				windowFrame := renderWindowCounterFrameRGBA(2, 1, 400, 240, true)
				scenario.Frames = append(scenario.Frames, surface.FrameReport{
					Order:     5,
					Width:     windowFrame.Width,
					Height:    windowFrame.Height,
					Stride:    windowFrame.Stride,
					Checksum:  checksumRGBA(windowFrame.Pixels),
					Presented: true,
				})
			case "wasm32-web-browser-canvas":
				beforeFrame := renderBrowserCounterFrameRGBA(0, 0, 320, 200, true)
				afterFrame := renderBrowserCounterFrameRGBA(2, 1, 400, 240, true)
				scenario.Frames = append(scenario.Frames,
					surface.FrameReport{Order: 1, Width: beforeFrame.Width, Height: beforeFrame.Height, Stride: beforeFrame.Stride, Checksum: checksumRGBA(beforeFrame.Pixels), Presented: true},
					surface.FrameReport{Order: 5, Width: afterFrame.Width, Height: afterFrame.Height, Stride: afterFrame.Stride, Checksum: checksumRGBA(afterFrame.Pixels), Presented: true},
				)
			}
			for _, component := range scenario.Components {
				if strings.HasPrefix(component.Type, "examples.surface_") && !strings.HasPrefix(component.Type, "examples.surface_release_counter.") {
					t.Fatalf("component type = %q, want release counter module", component.Type)
				}
			}
			processes, artifacts := releaseCounterTestEvidence(tc.mode)
			raw, err := json.Marshal(buildReport(opt, "linux-x64", processes, artifacts, cleanArtifactScan(len(artifacts)), scenario))
			if err != nil {
				t.Fatalf("marshal release counter report: %v", err)
			}
			var report surface.Report
			if err := json.Unmarshal(raw, &report); err != nil {
				t.Fatalf("decode release counter report: %v", err)
			}
			if report.Target != tc.wantTarget {
				t.Fatalf("target = %q, want %q", report.Target, tc.wantTarget)
			}
			if report.Source != "examples/surface_release_counter.tetra" {
				t.Fatalf("source = %q, want examples/surface_release_counter.tetra", report.Source)
			}
			if err := surface.ValidateReport(raw); err != nil {
				t.Fatalf("ValidateReport failed for release counter %s: %v\n%s", tc.mode, err, raw)
			}
		})
	}
}
