package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surface"
)

func TestCanvasFrameInputTextAccessibility(t *testing.T) {
	dir := t.TempDir()
	reportPath := writeWASM32WebBrowserReleaseRuntimeReport(t, dir)
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if err := validateWASM32WebBrowserReleaseEnvelope(surface.SchemaV1, raw); err != nil {
		t.Fatalf("validateWASM32WebBrowserReleaseEnvelope failed: %v", err)
	}
}

func TestBrowserReleaseRequiresFirstClassBrowserSurfaceEvidence(t *testing.T) {
	dir := t.TempDir()
	reportPath := writeWASM32WebBrowserReleaseRuntimeReport(t, dir)
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report map[string]any
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	delete(report, "browser_surface")
	raw, err = json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal mutated report: %v", err)
	}
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write mutated report: %v", err)
	}
	err = validateWASM32WebBrowserReleaseEnvelope(surface.SchemaV1, raw)
	if err == nil {
		t.Fatalf("expected browser release without first-class browser_surface evidence to fail")
	}
	if !strings.Contains(err.Error(), "browser_surface") {
		t.Fatalf("error = %v, want browser_surface diagnostic", err)
	}
}

func TestValidateSurfaceRuntimeReportReleaseModeRejectsOldBrowserCanvasEvidence(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	wasmPath, wasmSHA, wasmSize := writeNamedSurfaceArtifactFixture(t, artifactDir, "surface-browser-counter.wasm", []byte("\x00asm\x01\x00\x00\x00surface browser wasm fixture\n"), 0o755)
	loaderPath, loaderSHA, loaderSize := writeNamedSurfaceArtifactFixture(t, artifactDir, "surface-browser-counter.mjs", []byte(`function createSurfaceHost(instanceRef) {
  return { __tetra_surface_present_rgba() { return 0; } };
}
const imports = { tetra_surface_host_v1: createSurfaceHost({ instance: null }) };
`), 0o644)
	tracePath, traceSHA, traceSize := writeBrowserCanvasTraceFixture(t, artifactDir, wasmPath)
	reportPath := filepath.Join(dir, "surface-wasm32-web-browser-canvas.json")
	raw := validWASM32WebBrowserCanvasSurfaceRuntimeReportJSON(wasmPath, wasmSHA, wasmSize, loaderPath, loaderSHA, loaderSize, tracePath, traceSHA, traceSize)
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateSurfaceRuntimeReportWithOptions(reportPath, surfaceRuntimeValidationOptions{Release: "surface-v1"})
	if err == nil {
		t.Fatalf("expected old browser canvas evidence to fail release validation")
	}
	for _, want := range []string{"release surface-v1", "wasm32-web-browser-canvas-release-v1"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateSurfaceRuntimeReportReleaseModeRejectsUnknownRelease(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "surface-release-summary.json")
	if err := os.WriteFile(reportPath, validSurfaceRuntimeReleaseSummaryJSON(), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateSurfaceRuntimeReportWithOptions(reportPath, surfaceRuntimeValidationOptions{Release: "surface-v2"})
	if err == nil {
		t.Fatalf("expected unknown release to fail")
	}
	if !strings.Contains(err.Error(), "unsupported release") {
		t.Fatalf("error = %v, want unsupported release diagnostic", err)
	}
}

func TestValidateSurfaceRuntimeReportReleaseModeAcceptsReleaseEvidenceSlices(t *testing.T) {
	for _, tc := range []struct {
		name   string
		report surface.Report
	}{
		{
			name:   "linux toolkit slice",
			report: releaseToolkitSliceReportForTest("linux-x64", surface.HostEvidenceReport{Level: "linux-x64-real-window", Backend: "wayland-shm-rgba", Framebuffer: true, RealWindow: true, NativeInput: true}),
		},
		{
			name:   "wasm toolkit slice",
			report: releaseToolkitSliceReportForTest("wasm32-web", surface.HostEvidenceReport{Level: "wasm32-web-browser-canvas-input", Backend: "browser-canvas-rgba", Framebuffer: true, NativeInput: true}),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateSurfaceV1RuntimeReleaseReport(tc.report); err != nil {
				t.Fatalf("validateSurfaceV1RuntimeReleaseReport failed: %v", err)
			}
		})
	}
}

func TestValidateSurfaceRuntimeReportReleaseModeRejectsAccessibilityClaimsWithoutTargetEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		report surface.Report
		want   string
	}{
		{
			name:   "linux accessibility slice",
			report: releaseAccessibilitySliceReportForTest("linux-x64", surface.HostEvidenceReport{Level: "linux-x64-real-window", Backend: "wayland-shm-rgba", Framebuffer: true, RealWindow: true, NativeInput: true}),
			want:   "platform probe",
		},
		{
			name:   "wasm accessibility slice",
			report: releaseAccessibilitySliceReportForTest("wasm32-web", surface.HostEvidenceReport{Level: "wasm32-web-browser-canvas-input", Backend: "browser-canvas-rgba", Framebuffer: true, NativeInput: true}),
			want:   "browser accessibility",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSurfaceV1RuntimeReleaseReport(tc.report)
			if err == nil {
				t.Fatalf("expected incomplete accessibility release claim to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q diagnostic", err, tc.want)
			}
		})
	}
}

func TestValidateSurfaceRuntimeReportAcceptsHeadlessTraceAbsoluteSource(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	artifactPath, artifactSHA, artifactSize := writeSurfaceArtifactFixture(t, artifactDir)
	tracePath, traceSHA, traceSize := writeSurfaceTraceFixtureWithSourceAndFrames(t, artifactDir, "/repo/examples/surface_counter.tetra", []surfaceTraceFrameFixture{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "8ff0b13bb6026d76c7f61b202436f30e703d45d96730033ee6d171bcf5802f81", Presented: true},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, Checksum: "9ff0b13bb6026d76c7f61b202436f30e703d45d96730033ee6d171bcf5802f82", Presented: true},
	})
	reportPath := filepath.Join(dir, "surface-headless.json")
	if err := os.WriteFile(reportPath, validSurfaceRuntimeReportJSON(artifactPath, artifactSHA, artifactSize, tracePath, traceSHA, traceSize), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := validateSurfaceRuntimeReport(reportPath); err != nil {
		t.Fatalf("validateSurfaceRuntimeReport failed: %v", err)
	}
}

func TestValidateSurfaceRuntimeReportAcceptsWASM32WebEvidence(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	wasmPath, wasmSHA, wasmSize := writeNamedSurfaceArtifactFixture(t, artifactDir, "surface-counter.wasm", []byte("\x00asm\x01\x00\x00\x00surface wasm fixture\n"), 0o755)
	loaderPath, loaderSHA, loaderSize := writeNamedSurfaceArtifactFixture(t, artifactDir, "surface-counter.mjs", []byte(`function createSurfaceHost(instanceRef) {
  return { __tetra_surface_present_rgba() { return 0; } };
}
const imports = { tetra_surface_host_v1: createSurfaceHost({ instance: null }) };
`), 0o644)
	tracePath, traceSHA, traceSize := writeWASMTraceFixtureWithFrames(t, artifactDir, []wasmTraceFrameFixture{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, PixelsLen: 256000, Checksum: "3333333333333333333333333333333333333333333333333333333333333333"},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, PixelsLen: 256000, Checksum: "4444444444444444444444444444444444444444444444444444444444444444"},
	})
	reportPath := filepath.Join(dir, "surface-wasm32-web.json")
	raw := validWASM32WebSurfaceRuntimeReportJSON(wasmPath, wasmSHA, wasmSize, loaderPath, loaderSHA, loaderSize, tracePath, traceSHA, traceSize)
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := validateSurfaceRuntimeReport(reportPath); err != nil {
		t.Fatalf("validateSurfaceRuntimeReport failed: %v", err)
	}
}

func TestValidateSurfaceRuntimeReportAcceptsWASM32WebBrowserCanvasEvidence(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	wasmPath, wasmSHA, wasmSize := writeNamedSurfaceArtifactFixture(t, artifactDir, "surface-browser-counter.wasm", []byte("\x00asm\x01\x00\x00\x00surface browser wasm fixture\n"), 0o755)
	loaderPath, loaderSHA, loaderSize := writeNamedSurfaceArtifactFixture(t, artifactDir, "surface-browser-counter.mjs", []byte(`function createSurfaceHost(instanceRef) {
  return { __tetra_surface_present_rgba() { return 0; } };
}
const imports = { tetra_surface_host_v1: createSurfaceHost({ instance: null }) };
`), 0o644)
	tracePath, traceSHA, traceSize := writeBrowserCanvasTraceFixture(t, artifactDir, wasmPath)
	reportPath := filepath.Join(dir, "surface-wasm32-web-browser-canvas.json")
	raw := validWASM32WebBrowserCanvasSurfaceRuntimeReportJSON(wasmPath, wasmSHA, wasmSize, loaderPath, loaderSHA, loaderSize, tracePath, traceSHA, traceSize)
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := validateSurfaceRuntimeReport(reportPath); err != nil {
		t.Fatalf("validateSurfaceRuntimeReport failed: %v", err)
	}
}

func TestValidateSurfaceRuntimeReportRejectsArtifactScanCountMismatch(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	artifactPath, artifactSHA, artifactSize := writeSurfaceArtifactFixture(t, artifactDir)
	tracePath, traceSHA, traceSize := writeSurfaceTraceFixture(t, artifactDir)
	reportPath := filepath.Join(dir, "surface-headless.json")
	raw := strings.Replace(string(validSurfaceRuntimeReportJSON(artifactPath, artifactSHA, artifactSize, tracePath, traceSHA, traceSize)), `"files_checked":2`, `"files_checked":99`, 1)
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateSurfaceRuntimeReport(reportPath)
	if err == nil {
		t.Fatalf("expected artifact_scan files_checked mismatch to fail")
	}
	for _, want := range []string{"artifact_scan", "files_checked"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateSurfaceRuntimeReportRejectsUnreportedLegacySidecarInArtifactScanRoot(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	artifactPath, artifactSHA, artifactSize := writeSurfaceArtifactFixture(t, artifactDir)
	tracePath, traceSHA, traceSize := writeSurfaceTraceFixture(t, artifactDir)
	if err := os.WriteFile(filepath.Join(artifactDir, "surface-counter.ui.html"), []byte("<div>legacy ui</div>\n"), 0o644); err != nil {
		t.Fatalf("write legacy sidecar fixture: %v", err)
	}
	reportPath := filepath.Join(dir, "surface-headless.json")
	raw := strings.Replace(string(validSurfaceRuntimeReportJSON(artifactPath, artifactSHA, artifactSize, tracePath, traceSHA, traceSize)), `"files_checked":2`, `"files_checked":3`, 1)
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateSurfaceRuntimeReport(reportPath)
	if err == nil {
		t.Fatalf("expected unreported legacy UI sidecar in artifact_scan root to fail")
	}
	for _, want := range []string{"artifact_scan", "legacy UI sidecar"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateSurfaceRuntimeReportRejectsRunnerTraceFrameMismatch(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	artifactPath, artifactSHA, artifactSize := writeSurfaceArtifactFixture(t, artifactDir)
	tracePath, traceSHA, traceSize := writeSurfaceTraceFixtureWithFrames(t, artifactDir, []surfaceTraceFrameFixture{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "8ff0b13bb6026d76c7f61b202436f30e703d45d96730033ee6d171bcf5802f81", Presented: true},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, Checksum: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Presented: true},
	})
	reportPath := filepath.Join(dir, "surface-headless.json")
	if err := os.WriteFile(reportPath, validSurfaceRuntimeReportJSON(artifactPath, artifactSHA, artifactSize, tracePath, traceSHA, traceSize), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateSurfaceRuntimeReport(reportPath)
	if err == nil {
		t.Fatalf("expected runner trace frame mismatch to fail")
	}
	for _, want := range []string{"runner trace", "frame"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateSurfaceRuntimeReportRejectsHeadlessRunnerTraceSourceMismatch(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	artifactPath, artifactSHA, artifactSize := writeSurfaceArtifactFixture(t, artifactDir)
	tracePath, traceSHA, traceSize := writeSurfaceTraceFixtureWithSourceAndFrames(t, artifactDir, "examples/other_surface.tetra", []surfaceTraceFrameFixture{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "8ff0b13bb6026d76c7f61b202436f30e703d45d96730033ee6d171bcf5802f81", Presented: true},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, Checksum: "9ff0b13bb6026d76c7f61b202436f30e703d45d96730033ee6d171bcf5802f82", Presented: true},
	})
	reportPath := filepath.Join(dir, "surface-headless.json")
	if err := os.WriteFile(reportPath, validSurfaceRuntimeReportJSON(artifactPath, artifactSHA, artifactSize, tracePath, traceSHA, traceSize), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateSurfaceRuntimeReport(reportPath)
	if err == nil {
		t.Fatalf("expected headless runner trace source mismatch to fail")
	}
	for _, want := range []string{"runner trace", "source"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateSurfaceRuntimeReportRejectsWASMRunnerTraceFrameMismatch(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	wasmPath, wasmSHA, wasmSize := writeNamedSurfaceArtifactFixture(t, artifactDir, "surface-counter.wasm", []byte("\x00asm\x01\x00\x00\x00surface wasm fixture\n"), 0o755)
	loaderPath, loaderSHA, loaderSize := writeNamedSurfaceArtifactFixture(t, artifactDir, "surface-counter.mjs", []byte(`function createSurfaceHost(instanceRef) {
  return { __tetra_surface_present_rgba() { return 0; } };
}
const imports = { tetra_surface_host_v1: createSurfaceHost({ instance: null }) };
`), 0o644)
	tracePath, traceSHA, traceSize := writeWASMTraceFixtureWithFrames(t, artifactDir, []wasmTraceFrameFixture{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, PixelsLen: 256000, Checksum: "3333333333333333333333333333333333333333333333333333333333333333"},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, PixelsLen: 256000, Checksum: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
	})
	reportPath := filepath.Join(dir, "surface-wasm32-web.json")
	raw := validWASM32WebSurfaceRuntimeReportJSON(wasmPath, wasmSHA, wasmSize, loaderPath, loaderSHA, loaderSize, tracePath, traceSHA, traceSize)
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateSurfaceRuntimeReport(reportPath)
	if err == nil {
		t.Fatalf("expected wasm runner trace frame mismatch to fail")
	}
	for _, want := range []string{"runner trace", "frame"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateSurfaceRuntimeReportRejectsWASMRunnerTraceArtifactMismatch(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	wasmPath, wasmSHA, wasmSize := writeNamedSurfaceArtifactFixture(t, artifactDir, "surface-counter.wasm", []byte("\x00asm\x01\x00\x00\x00surface wasm fixture\n"), 0o755)
	loaderPath, loaderSHA, loaderSize := writeNamedSurfaceArtifactFixture(t, artifactDir, "surface-counter.mjs", []byte(`function createSurfaceHost(instanceRef) {
  return { __tetra_surface_present_rgba() { return 0; } };
}
const imports = { tetra_surface_host_v1: createSurfaceHost({ instance: null }) };
`), 0o644)
	tracePath, traceSHA, traceSize := writeWASMTraceFixtureWithWASMAndFrames(t, artifactDir, filepath.Join(artifactDir, "other-surface.wasm"), []wasmTraceFrameFixture{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, PixelsLen: 256000, Checksum: "3333333333333333333333333333333333333333333333333333333333333333"},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, PixelsLen: 256000, Checksum: "4444444444444444444444444444444444444444444444444444444444444444"},
	})
	reportPath := filepath.Join(dir, "surface-wasm32-web.json")
	raw := validWASM32WebSurfaceRuntimeReportJSON(wasmPath, wasmSHA, wasmSize, loaderPath, loaderSHA, loaderSize, tracePath, traceSHA, traceSize)
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateSurfaceRuntimeReport(reportPath)
	if err == nil {
		t.Fatalf("expected wasm runner trace artifact mismatch to fail")
	}
	for _, want := range []string{"runner trace", "wasm"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateSurfaceRuntimeReportRejectsWASMReportWithHeadlessRunnerTrace(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	wasmPath, wasmSHA, wasmSize := writeNamedSurfaceArtifactFixture(t, artifactDir, "surface-counter.wasm", []byte("\x00asm\x01\x00\x00\x00surface wasm fixture\n"), 0o755)
	loaderPath, loaderSHA, loaderSize := writeNamedSurfaceArtifactFixture(t, artifactDir, "surface-counter.mjs", []byte(`function createSurfaceHost(instanceRef) {
  return { __tetra_surface_present_rgba() { return 0; } };
}
const imports = { tetra_surface_host_v1: createSurfaceHost({ instance: null }) };
`), 0o644)
	tracePath, traceSHA, traceSize := writeSurfaceTraceFixture(t, artifactDir)
	reportPath := filepath.Join(dir, "surface-wasm32-web.json")
	raw := validWASM32WebSurfaceRuntimeReportJSON(wasmPath, wasmSHA, wasmSize, loaderPath, loaderSHA, loaderSize, tracePath, traceSHA, traceSize)
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateSurfaceRuntimeReport(reportPath)
	if err == nil {
		t.Fatalf("expected wasm32-web report with headless runner trace schema to fail")
	}
	for _, want := range []string{"runner trace", "wasm32-web"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}

func TestValidateSurfaceRuntimeReportRejectsBrowserCanvasReportWithStarterRunnerTrace(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	wasmPath, wasmSHA, wasmSize := writeNamedSurfaceArtifactFixture(t, artifactDir, "surface-browser-counter.wasm", []byte("\x00asm\x01\x00\x00\x00surface browser wasm fixture\n"), 0o755)
	loaderPath, loaderSHA, loaderSize := writeNamedSurfaceArtifactFixture(t, artifactDir, "surface-browser-counter.mjs", []byte(`function createSurfaceHost(instanceRef) {
  return { __tetra_surface_present_rgba() { return 0; } };
}
const imports = { tetra_surface_host_v1: createSurfaceHost({ instance: null }) };
`), 0o644)
	tracePath, traceSHA, traceSize := writeWASMTraceFixtureWithWASMAndFrames(t, artifactDir, wasmPath, []wasmTraceFrameFixture{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, PixelsLen: 256000, Checksum: "1111111111111111111111111111111111111111111111111111111111111111"},
		{Order: 2, Width: 400, Height: 240, Stride: 1600, PixelsLen: 384000, Checksum: "5555555555555555555555555555555555555555555555555555555555555555"},
	})
	reportPath := filepath.Join(dir, "surface-wasm32-web-browser-canvas.json")
	raw := validWASM32WebBrowserCanvasSurfaceRuntimeReportJSON(wasmPath, wasmSHA, wasmSize, loaderPath, loaderSHA, loaderSize, tracePath, traceSHA, traceSize)
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateSurfaceRuntimeReport(reportPath)
	if err == nil {
		t.Fatalf("expected browser canvas report with starter runner trace schema to fail")
	}
	if !strings.Contains(err.Error(), "starter Node evidence") {
		t.Fatalf("error = %v, want starter Node evidence rejection", err)
	}
}

func TestValidateSurfaceRuntimeReportRejectsBrowserCanvasReadbackMismatch(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	wasmPath, wasmSHA, wasmSize := writeNamedSurfaceArtifactFixture(t, artifactDir, "surface-browser-counter.wasm", []byte("\x00asm\x01\x00\x00\x00surface browser wasm fixture\n"), 0o755)
	loaderPath, loaderSHA, loaderSize := writeNamedSurfaceArtifactFixture(t, artifactDir, "surface-browser-counter.mjs", []byte(`function createSurfaceHost(instanceRef) {
  return { __tetra_surface_present_rgba() { return 0; } };
}
const imports = { tetra_surface_host_v1: createSurfaceHost({ instance: null }) };
`), 0o644)
	tracePath, traceSHA, traceSize := writeBrowserCanvasTraceFixtureWithChecksums(t, artifactDir, wasmPath, "1111111111111111111111111111111111111111111111111111111111111111", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "5555555555555555555555555555555555555555555555555555555555555555")
	reportPath := filepath.Join(dir, "surface-wasm32-web-browser-canvas.json")
	raw := validWASM32WebBrowserCanvasSurfaceRuntimeReportJSON(wasmPath, wasmSHA, wasmSize, loaderPath, loaderSHA, loaderSize, tracePath, traceSHA, traceSize)
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateSurfaceRuntimeReport(reportPath)
	if err == nil {
		t.Fatalf("expected browser canvas source/canvas checksum mismatch to fail")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("error = %v, want checksum mismatch rejection", err)
	}
}

func TestValidateSurfaceRuntimeReportRejectsDocsOnlyEvidence(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	artifactPath, artifactSHA, artifactSize := writeSurfaceArtifactFixture(t, artifactDir)
	tracePath, traceSHA, traceSize := writeSurfaceTraceFixture(t, artifactDir)
	reportPath := filepath.Join(dir, "surface-headless.json")
	raw := strings.Replace(string(validSurfaceRuntimeReportJSON(artifactPath, artifactSHA, artifactSize, tracePath, traceSHA, traceSize)), `"source": "examples/surface_counter.tetra"`, `"source": "docs-only surface note"`, 1)
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateSurfaceRuntimeReport(reportPath)
	if err == nil {
		t.Fatalf("expected docs-only report to fail")
	}
	if !strings.Contains(err.Error(), "docs-only") {
		t.Fatalf("error = %v, want docs-only rejection", err)
	}
}

func TestValidateSurfaceRuntimeReportRejectsArtifactHashMismatch(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	artifactPath, _, artifactSize := writeSurfaceArtifactFixture(t, artifactDir)
	tracePath, traceSHA, traceSize := writeSurfaceTraceFixture(t, artifactDir)
	reportPath := filepath.Join(dir, "surface-headless.json")
	wrongSHA := "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if err := os.WriteFile(reportPath, validSurfaceRuntimeReportJSON(artifactPath, wrongSHA, artifactSize, tracePath, traceSHA, traceSize), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateSurfaceRuntimeReport(reportPath)
	if err == nil {
		t.Fatalf("expected artifact hash mismatch to fail")
	}
	if !strings.Contains(err.Error(), "artifact integrity") {
		t.Fatalf("error = %v, want artifact integrity rejection", err)
	}
}

func TestValidateSurfaceRuntimeReportRejectsMissingArtifact(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	artifactPath := filepath.Join(artifactDir, "missing-surface-counter")
	tracePath, traceSHA, traceSize := writeSurfaceTraceFixture(t, artifactDir)
	reportPath := filepath.Join(dir, "surface-headless.json")
	sha := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := os.WriteFile(reportPath, validSurfaceRuntimeReportJSON(artifactPath, sha, 49172, tracePath, traceSHA, traceSize), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateSurfaceRuntimeReport(reportPath)
	if err == nil {
		t.Fatalf("expected missing artifact to fail")
	}
	if !strings.Contains(err.Error(), "artifact integrity") {
		t.Fatalf("error = %v, want artifact integrity rejection", err)
	}
}

func TestValidateSurfaceRuntimeReportRejectsCompilerOwnedLoaderDOMUserJS(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	wasmPath, wasmSHA, wasmSize := writeNamedSurfaceArtifactFixture(t, artifactDir, "surface-counter.wasm", []byte("\x00asm\x01\x00\x00\x00surface wasm fixture\n"), 0o755)
	loaderPath, loaderSHA, loaderSize := writeNamedSurfaceArtifactFixture(t, artifactDir, "surface-counter.mjs", []byte(`function createSurfaceHost(instanceRef) {
  return { __tetra_surface_present_rgba() { return 0; } };
}
const imports = { tetra_surface_host_v1: createSurfaceHost({ instance: null }) };
document.createElement("canvas");
`), 0o644)
	tracePath, traceSHA, traceSize := writeWASMTraceFixtureWithFrames(t, artifactDir, []wasmTraceFrameFixture{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, PixelsLen: 256000, Checksum: "3333333333333333333333333333333333333333333333333333333333333333"},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, PixelsLen: 256000, Checksum: "4444444444444444444444444444444444444444444444444444444444444444"},
	})
	reportPath := filepath.Join(dir, "surface-wasm32-web.json")
	raw := validWASM32WebSurfaceRuntimeReportJSON(wasmPath, wasmSHA, wasmSize, loaderPath, loaderSHA, loaderSize, tracePath, traceSHA, traceSize)
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}

	err := validateSurfaceRuntimeReport(reportPath)
	if err == nil {
		t.Fatalf("expected compiler-owned loader DOM/user-JS artifact to fail")
	}
	if !strings.Contains(err.Error(), "DOM/user-JS marker") {
		t.Fatalf("error = %v, want DOM/user-JS marker rejection", err)
	}
}
