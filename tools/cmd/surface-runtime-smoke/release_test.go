package main

import (
	"os/exec"
	"testing"

	"tetra_language/tools/validators/surface"
)

func releaseCounterTestEvidence(mode string) ([]surface.ProcessReport, []surface.ArtifactReport) {
	switch mode {
	case "wasm32-web-browser-canvas":
		return []surface.ProcessReport{
			{Name: "tetra build", Kind: "build", Path: "tetra build --target wasm32-web examples/surface_release_counter.tetra -o /tmp/surface-artifacts/surface-browser-counter.wasm", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface wasm32-web browser canvas component app", Kind: "app", Path: "/usr/bin/chromium --headless <surface-browser-canvas-runner> wasm=/tmp/surface-artifacts/surface-browser-counter.wasm", Ran: true, Pass: true, ExitCode: intPtr(0), ExpectedExitCode: intPtr(0)},
			{Name: "surface wasm32-web import validator", Kind: "runtime", Path: "go run ./tools/cmd/validate-wasm-imports --target wasm32-web /tmp/surface-artifacts/surface-browser-counter.wasm", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface wasm32-web browser canvas runtime", Kind: "runtime", Path: "Chromium test fixture", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface wasm32-web browser canvas trace", Kind: "runtime", Path: "/usr/bin/chromium --headless --dump-dom http://127.0.0.1:1/surface-browser-canvas-runner", Ran: true, Pass: true, ExitCode: intPtr(0)},
		}, wasmBrowserCanvasSurfaceArtifacts()
	default:
		return []surface.ProcessReport{
			{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_release_counter.tetra -o /tmp/surface-artifacts/surface-window-counter", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-window-counter", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
			{Name: "surface linux-x64 real-window probe", Kind: "app", Path: "/tmp/surface-artifacts/surface-real-window-probe", Ran: true, Pass: true, ExitCode: intPtr(42), ExpectedExitCode: intPtr(42)},
			{Name: "surface linux-x64 runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke", Ran: true, Pass: true, ExitCode: intPtr(0)},
		}, componentAppArtifacts("/tmp/surface-artifacts/surface-window-counter")
	}
}

func releaseWindowTestProcesses() []surface.ProcessReport {
	return []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_release_form.tetra -o /tmp/surface-artifacts/surface-release-form", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-release-form", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
		{Name: "surface linux-x64 real-window probe", Kind: "app", Path: "/tmp/surface-artifacts/surface-release-window-probe", Ran: true, Pass: true, ExitCode: intPtr(42), ExpectedExitCode: intPtr(42)},
		{Name: "surface linux-x64 release clipboard harness", Kind: "runtime", Path: "/tmp/surface-artifacts/surface-linux-clipboard-harness.json", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface linux-x64 release composition harness", Kind: "runtime", Path: "/tmp/surface-artifacts/surface-linux-composition-harness.json", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface linux accessibility host bridge", Kind: "runtime", Path: "/tmp/surface-artifacts/surface-linux-accessibility-bridge.json", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface linux accessibility platform probe", Kind: "runtime", Path: "/tmp/surface-artifacts/surface-linux-accessibility-probe.json", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface linux-x64 runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode linux-x64-release-window", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}
}

func releaseWindowTestArtifacts() []surface.ArtifactReport {
	return []surface.ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-release-form", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 90001},
		{Kind: "linux-accessibility-host-bridge", Path: "/tmp/surface-artifacts/surface-linux-accessibility-bridge.json", SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Size: 4096},
		{Kind: "linux-accessibility-platform-probe", Path: "/tmp/surface-artifacts/surface-linux-accessibility-probe.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 4096},
	}
}

func releaseWindowTestFrames() []surface.FrameReport {
	before := renderReleaseToolkitFrameRGBA(0, 0, -1, 0, 0, 0, false, 0, 320, 240)
	after := renderReleaseToolkitFrameRGBA(0, 0, 7, 1, 1, 2, true, 16, 560, 420)
	return []surface.FrameReport{
		{Order: 1, Width: before.Width, Height: before.Height, Stride: before.Stride, Checksum: checksumRGBA(before.Pixels), Presented: true},
		{Order: 5, Width: after.Width, Height: after.Height, Stride: after.Stride, Checksum: checksumRGBA(after.Pixels), Presented: true},
	}
}

func releaseBrowserTestProcesses() []surface.ProcessReport {
	return []surface.ProcessReport{
		{Name: "tetra build", Kind: "build", Path: "tetra build --target wasm32-web examples/surface_release_form.tetra -o /tmp/surface-artifacts/surface-release-form.wasm", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface wasm32-web browser canvas component app", Kind: "app", Path: "/usr/bin/chromium --headless <surface-browser-canvas-runner> scenario=release-browser wasm=/tmp/surface-artifacts/surface-release-form.wasm", Ran: true, Pass: true, ExitCode: intPtr(0), ExpectedExitCode: intPtr(0)},
		{Name: "surface wasm32-web import validator", Kind: "runtime", Path: "go run ./tools/cmd/validate-wasm-imports --target wasm32-web /tmp/surface-artifacts/surface-release-form.wasm", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface wasm32-web browser canvas runtime", Kind: "runtime", Path: "Chromium release browser fixture", Ran: true, Pass: true, ExitCode: intPtr(0)},
		{Name: "surface wasm32-web browser canvas trace", Kind: "runtime", Path: "/usr/bin/chromium --headless --dump-dom http://127.0.0.1:1/surface-browser-canvas-runner?scenario=release-browser", Ran: true, Pass: true, ExitCode: intPtr(0)},
	}
}

func releaseBrowserTestArtifacts() []surface.ArtifactReport {
	return []surface.ArtifactReport{
		{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-release-form.wasm", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 9604},
		{Kind: "compiler-owned-loader", Path: "/tmp/surface-artifacts/surface-release-form.mjs", SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Size: 4939},
		{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 4096},
	}
}

func releaseBrowserTestFrames() []surface.FrameReport {
	before := renderReleaseToolkitFrameRGBA(0, 0, -1, 0, 0, 0, false, 0, 320, 240)
	after := renderReleaseToolkitFrameRGBA(0, 0, 7, 1, 1, 2, true, 16, 560, 420)
	return []surface.FrameReport{
		{Order: 1, Width: before.Width, Height: before.Height, Stride: before.Stride, Checksum: checksumRGBA(before.Pixels), Presented: true},
		{Order: 5, Width: after.Width, Height: after.Height, Stride: after.Stride, Checksum: checksumRGBA(after.Pixels), Presented: true},
	}
}

func releaseAccessibilityTestFrames(mode string) []surface.FrameReport {
	if mode != "wasm32-web-release-accessibility" {
		return nil
	}
	before := renderAccessibilityMetadataFrameRGBA(0, 0, -1, 0, 0, 0, 320, 240)
	after := renderAccessibilityMetadataFrameRGBA(0, 0, 5, 1, 1, 2, 480, 320)
	return []surface.FrameReport{
		{Order: 1, Width: before.Width, Height: before.Height, Stride: before.Stride, Checksum: checksumRGBA(before.Pixels), Presented: true},
		{Order: 5, Width: after.Width, Height: after.Height, Stride: after.Stride, Checksum: checksumRGBA(after.Pixels), Presented: true},
	}
}

func TestCollectWASM32WebProcessEvidenceRecordsPresentedFrameTrace(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for wasm32-web Surface runner evidence")
	}
	source, err := resolveSurfaceSourcePath("examples/surface_counter.tetra")
	if err != nil {
		t.Fatalf("resolve Surface source: %v", err)
	}
	evidence, err := collectWASM32WebProcessEvidence(source, t.TempDir())
	if err != nil {
		t.Fatalf("collectWASM32WebProcessEvidence: %v", err)
	}
	if len(evidence.Frames) < 2 {
		t.Fatalf("frames = %#v, want actual wasm-presented pre/post frames", evidence.Frames)
	}
	if evidence.ArtifactScan.FilesChecked < 3 {
		t.Fatalf("artifact scan = %#v, want wasm, loader, and runner trace checked", evidence.ArtifactScan)
	}
	after := evidence.Frames[len(evidence.Frames)-1]
	if after.Order != 4 || after.Width != 320 || after.Height != 200 || after.Stride != 1280 || !after.Presented {
		t.Fatalf("last wasm frame = %#v, want order-4 320x200 presented frame", after)
	}
	wantFrame := renderCounterFrameRGBA(1, true)
	if want := checksumRGBA(wantFrame.Pixels); after.Checksum != want {
		t.Fatalf("last wasm frame checksum = %s, want actual CounterApp checksum %s", after.Checksum, want)
	}
}

func textFocusInputTestProcesses(mode string) []surface.ProcessReport {
	switch mode {
	case "linux-x64-real-window-text-focus-input":
		return []surface.ProcessReport{
			{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_textbox_app.tetra -o /tmp/surface-artifacts/surface-textbox-app", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-textbox-app", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
			{Name: "surface linux-x64 real-window probe", Kind: "app", Path: "/tmp/surface-artifacts/surface-real-window-probe", Ran: true, Pass: true, ExitCode: intPtr(42), ExpectedExitCode: intPtr(42)},
			{Name: "surface linux-x64 runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke", Ran: true, Pass: true, ExitCode: intPtr(0)},
		}
	case "wasm32-web-browser-canvas-text-focus-input":
		return []surface.ProcessReport{
			{Name: "tetra build", Kind: "build", Path: "tetra build --target wasm32-web examples/surface_textbox_app.tetra -o /tmp/surface-artifacts/surface-textbox-app.wasm", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface wasm32-web browser canvas component app", Kind: "app", Path: "/usr/bin/chromium --headless <surface-browser-canvas-runner> wasm=/tmp/surface-artifacts/surface-textbox-app.wasm", Ran: true, Pass: true, ExitCode: intPtr(0), ExpectedExitCode: intPtr(0)},
			{Name: "surface wasm32-web import validator", Kind: "runtime", Path: "go run ./tools/cmd/validate-wasm-imports --target wasm32-web /tmp/surface-artifacts/surface-textbox-app.wasm", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface wasm32-web browser canvas runtime", Kind: "runtime", Path: "Chromium test fixture", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface wasm32-web browser canvas trace", Kind: "runtime", Path: "/usr/bin/chromium --headless --dump-dom http://127.0.0.1:1/surface-browser-canvas-runner?scenario=text-focus-input", Ran: true, Pass: true, ExitCode: intPtr(0)},
		}
	default:
		return []surface.ProcessReport{
			{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_textbox_app.tetra -o /tmp/surface-artifacts/surface-textbox-app", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-textbox-app", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
			{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke", Ran: true, Pass: true, ExitCode: intPtr(0)},
		}
	}
}

func textFocusInputTestArtifacts(mode string) []surface.ArtifactReport {
	switch mode {
	case "wasm32-web-browser-canvas-text-focus-input":
		return []surface.ArtifactReport{
			{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-textbox-app.wasm", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 8604},
			{Kind: "compiler-owned-loader", Path: "/tmp/surface-artifacts/surface-textbox-app.mjs", SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Size: 4939},
			{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 1184},
		}
	case "headless-text-focus-input":
		return headlessSurfaceArtifacts("/tmp/surface-artifacts/surface-textbox-app")
	default:
		return componentAppArtifacts("/tmp/surface-artifacts/surface-textbox-app")
	}
}

func textFocusInputTestFrames(mode string) []surface.FrameReport {
	switch mode {
	case "linux-x64-real-window-text-focus-input":
		frame := renderTextFocusInputFrameRGBA(1, 1, 1, 400, 240)
		return []surface.FrameReport{{Order: 5, Width: frame.Width, Height: frame.Height, Stride: frame.Stride, Checksum: checksumRGBA(frame.Pixels), Presented: true}}
	case "wasm32-web-browser-canvas-text-focus-input":
		before := renderTextFocusInputFrameRGBA(0, 0, 0, 320, 200)
		after := renderTextFocusInputFrameRGBA(1, 1, 1, 400, 240)
		return []surface.FrameReport{
			{Order: 1, Width: before.Width, Height: before.Height, Stride: before.Stride, Checksum: checksumRGBA(before.Pixels), Presented: true},
			{Order: 5, Width: after.Width, Height: after.Height, Stride: after.Stride, Checksum: checksumRGBA(after.Pixels), Presented: true},
		}
	default:
		return nil
	}
}

func releaseTextInputTestProcesses(mode string) []surface.ProcessReport {
	switch mode {
	case "linux-x64-release-text-input":
		return []surface.ProcessReport{
			{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_release_text_input.tetra -o /tmp/surface-artifacts/surface-release-text-input", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-release-text-input", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
			{Name: "surface linux-x64 real-window probe", Kind: "app", Path: "/tmp/surface-artifacts/surface-real-window-probe", Ran: true, Pass: true, ExitCode: intPtr(42), ExpectedExitCode: intPtr(42)},
			{Name: "surface linux-x64 runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode linux-x64-release-text-input", Ran: true, Pass: true, ExitCode: intPtr(0)},
		}
	case "wasm32-web-release-text-input":
		return []surface.ProcessReport{
			{Name: "tetra build", Kind: "build", Path: "tetra build --target wasm32-web examples/surface_release_text_input.tetra -o /tmp/surface-artifacts/surface-release-text-input.wasm", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface wasm32-web browser canvas component app", Kind: "app", Path: "/usr/bin/chromium --headless <surface-browser-canvas-runner> wasm=/tmp/surface-artifacts/surface-release-text-input.wasm", Ran: true, Pass: true, ExitCode: intPtr(0), ExpectedExitCode: intPtr(0)},
			{Name: "surface wasm32-web import validator", Kind: "runtime", Path: "go run ./tools/cmd/validate-wasm-imports --target wasm32-web /tmp/surface-artifacts/surface-release-text-input.wasm", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface wasm32-web browser canvas runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode wasm32-web-release-text-input", Ran: true, Pass: true, ExitCode: intPtr(0)},
		}
	default:
		return []surface.ProcessReport{
			{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_release_text_input.tetra -o /tmp/surface-artifacts/surface-release-text-input", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-release-text-input", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
			{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-release-text-input", Ran: true, Pass: true, ExitCode: intPtr(0)},
		}
	}
}

func releaseTextInputTestArtifacts(mode string) []surface.ArtifactReport {
	switch mode {
	case "wasm32-web-release-text-input":
		return []surface.ArtifactReport{
			{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-release-text-input.wasm", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 8604},
			{Kind: "compiler-owned-loader", Path: "/tmp/surface-artifacts/surface-release-text-input.mjs", SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Size: 4939},
			{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 1184},
		}
	case "headless-release-text-input":
		return headlessSurfaceArtifacts("/tmp/surface-artifacts/surface-release-text-input")
	default:
		return componentAppArtifacts("/tmp/surface-artifacts/surface-release-text-input")
	}
}

func releaseTextInputTestArtifactCount(mode string) int {
	if mode == "wasm32-web-release-text-input" {
		return 3
	}
	if mode == "headless-release-text-input" {
		return 2
	}
	return 1
}

func releaseAccessibilityTestProcesses(mode string) []surface.ProcessReport {
	switch mode {
	case "linux-x64-release-accessibility":
		return []surface.ProcessReport{
			{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_release_accessibility.tetra -o /tmp/surface-artifacts/surface-release-accessibility", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-release-accessibility", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
			{Name: "surface linux-x64 real-window probe", Kind: "app", Path: "/tmp/surface-artifacts/surface-accessibility-real-window-probe", Ran: true, Pass: true, ExitCode: intPtr(42), ExpectedExitCode: intPtr(42)},
			{Name: "surface linux-x64 runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode linux-x64-release-accessibility", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface linux accessibility host bridge", Kind: "runtime", Path: "/tmp/surface-artifacts/surface-linux-accessibility-bridge.json", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface linux accessibility platform probe", Kind: "runtime", Path: "/tmp/surface-artifacts/surface-linux-accessibility-probe.json", Ran: true, Pass: true, ExitCode: intPtr(0)},
		}
	case "wasm32-web-release-accessibility":
		return []surface.ProcessReport{
			{Name: "tetra build", Kind: "build", Path: "tetra build --target wasm32-web examples/surface_release_accessibility.tetra -o /tmp/surface-artifacts/surface-release-accessibility.wasm", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface wasm32-web browser canvas component app", Kind: "app", Path: "/usr/bin/chromium --headless <surface-browser-canvas-runner> scenario=release-accessibility wasm=/tmp/surface-artifacts/surface-release-accessibility.wasm", Ran: true, Pass: true, ExitCode: intPtr(0), ExpectedExitCode: intPtr(0)},
			{Name: "surface wasm32-web import validator", Kind: "runtime", Path: "go run ./tools/cmd/validate-wasm-imports --target wasm32-web /tmp/surface-artifacts/surface-release-accessibility.wasm", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface wasm32-web browser canvas runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode wasm32-web-release-accessibility", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface wasm32-web browser canvas trace", Kind: "runtime", Path: "/usr/bin/chromium --headless --dump-dom http://127.0.0.1:1/surface-browser-canvas-runner?scenario=release-accessibility", Ran: true, Pass: true, ExitCode: intPtr(0)},
		}
	default:
		return []surface.ProcessReport{
			{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_release_accessibility.tetra -o /tmp/surface-artifacts/surface-release-accessibility", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-release-accessibility", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
			{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-release-accessibility", Ran: true, Pass: true, ExitCode: intPtr(0)},
		}
	}
}

func releaseAccessibilityTestArtifacts(mode string) []surface.ArtifactReport {
	switch mode {
	case "linux-x64-release-accessibility":
		artifacts := componentAppArtifacts("/tmp/surface-artifacts/surface-release-accessibility")
		return append(artifacts,
			surface.ArtifactReport{Kind: "linux-accessibility-host-bridge", Path: "/tmp/surface-artifacts/surface-linux-accessibility-bridge.json", SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Size: 4096},
			surface.ArtifactReport{Kind: "linux-accessibility-platform-probe", Path: "/tmp/surface-artifacts/surface-linux-accessibility-probe.json", SHA256: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", Size: 4096},
		)
	case "wasm32-web-release-accessibility":
		return []surface.ArtifactReport{
			{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-release-accessibility.wasm", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 9004},
			{Kind: "compiler-owned-loader", Path: "/tmp/surface-artifacts/surface-release-accessibility.mjs", SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Size: 4939},
			{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 4096},
		}
	default:
		return headlessSurfaceArtifacts("/tmp/surface-artifacts/surface-release-accessibility")
	}
}

func releaseTextInputTestCases() []surface.CaseReport {
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
		{Name: "release text input invalid UTF-8 rejected", Kind: "negative", Ran: true, Pass: true, ExpectedError: "invalid utf8 rejected"},
		{Name: "release text input multiline storage", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input caret home end arrows", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input selection replacement", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input selection clipboard transfer", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input backspace delete", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input clipboard owned copy transfer", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input composition start update", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input composition commit", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input composition cancel", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input shaping plan scoped", Kind: "positive", Ran: true, Pass: true},
		{Name: "settings reference text input trace", Kind: "positive", Ran: true, Pass: true},
		{Name: "editor reference text input trace", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input safe view lifetime checked", Kind: "positive", Ran: true, Pass: true},
		{Name: "reject legacy UI evidence", Kind: "negative", Ran: true, Pass: true, ExpectedError: "legacy UI evidence rejected"},
	}
}

func componentTreeTestProcesses(mode string) []surface.ProcessReport {
	switch mode {
	case "linux-x64-real-window-component-tree", "linux-x64-real-window-component-tree-api":
		return []surface.ProcessReport{
			{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_tree_app.tetra -o /tmp/surface-artifacts/surface-tree-app", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-tree-app", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
			{Name: "surface linux-x64 real-window probe", Kind: "app", Path: "/tmp/surface-artifacts/surface-real-window-probe", Ran: true, Pass: true, ExitCode: intPtr(42), ExpectedExitCode: intPtr(42)},
			{Name: "surface linux-x64 runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke", Ran: true, Pass: true, ExitCode: intPtr(0)},
		}
	case "wasm32-web-browser-canvas-component-tree", "wasm32-web-browser-canvas-component-tree-api":
		return []surface.ProcessReport{
			{Name: "tetra build", Kind: "build", Path: "tetra build --target wasm32-web examples/surface_tree_app.tetra -o /tmp/surface-artifacts/surface-tree-app.wasm", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface wasm32-web browser canvas component app", Kind: "app", Path: "/usr/bin/chromium --headless <surface-browser-canvas-runner> scenario=component-tree wasm=/tmp/surface-artifacts/surface-tree-app.wasm", Ran: true, Pass: true, ExitCode: intPtr(0), ExpectedExitCode: intPtr(0)},
			{Name: "surface wasm32-web import validator", Kind: "runtime", Path: "go run ./tools/cmd/validate-wasm-imports --target wasm32-web /tmp/surface-artifacts/surface-tree-app.wasm", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface wasm32-web browser canvas runtime", Kind: "runtime", Path: "Chromium test fixture", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface wasm32-web browser canvas trace", Kind: "runtime", Path: "/usr/bin/chromium --headless --dump-dom http://127.0.0.1:1/surface-browser-canvas-runner?scenario=component-tree", Ran: true, Pass: true, ExitCode: intPtr(0)},
		}
	default:
		return []surface.ProcessReport{
			{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_tree_app.tetra -o /tmp/surface-artifacts/surface-tree-app", Ran: true, Pass: true, ExitCode: intPtr(0)},
			{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-tree-app", Ran: true, Pass: true, ExitCode: intPtr(1), ExpectedExitCode: intPtr(1)},
			{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke", Ran: true, Pass: true, ExitCode: intPtr(0)},
		}
	}
}

func componentTreeTestArtifacts(mode string) []surface.ArtifactReport {
	switch mode {
	case "wasm32-web-browser-canvas-component-tree", "wasm32-web-browser-canvas-component-tree-api":
		return []surface.ArtifactReport{
			{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-tree-app.wasm", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 9004},
			{Kind: "compiler-owned-loader", Path: "/tmp/surface-artifacts/surface-tree-app.mjs", SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Size: 4939},
			{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Size: 22184},
		}
	case "headless-component-tree", "headless-component-tree-api":
		return headlessSurfaceArtifacts("/tmp/surface-artifacts/surface-tree-app")
	default:
		return componentAppArtifacts("/tmp/surface-artifacts/surface-tree-app")
	}
}

func componentTreeTestFrames(mode string) []surface.FrameReport {
	switch mode {
	case "linux-x64-real-window-component-tree", "linux-x64-real-window-component-tree-api":
		frame := renderComponentTreeFrameRGBA(0, 0, 6, 1, 1, 400, 240)
		return []surface.FrameReport{{Order: 5, Width: frame.Width, Height: frame.Height, Stride: frame.Stride, Checksum: checksumRGBA(frame.Pixels), Presented: true}}
	case "wasm32-web-browser-canvas-component-tree", "wasm32-web-browser-canvas-component-tree-api":
		before := renderComponentTreeFrameRGBA(0, 0, -1, 0, 0, 320, 200)
		after := renderComponentTreeFrameRGBA(0, 0, 6, 1, 1, 400, 240)
		return []surface.FrameReport{
			{Order: 1, Width: before.Width, Height: before.Height, Stride: before.Stride, Checksum: checksumRGBA(before.Pixels), Presented: true},
			{Order: 5, Width: after.Width, Height: after.Height, Stride: after.Stride, Checksum: checksumRGBA(after.Pixels), Presented: true},
		}
	default:
		return nil
	}
}

func componentTreeDispatchPathsContain(paths []surface.ComponentTreeDispatchPathReport, want []int) bool {
	for _, path := range paths {
		if intSlicesEqual(path.Path, want) {
			return true
		}
	}
	return false
}
