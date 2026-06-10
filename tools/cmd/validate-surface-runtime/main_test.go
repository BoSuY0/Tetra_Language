package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"tetra_language/tools/validators/surface"
)

func TestValidateSurfaceRuntimeReportAcceptsHeadlessEvidence(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	artifactPath, artifactSHA, artifactSize := writeSurfaceArtifactFixture(t, artifactDir)
	tracePath, traceSHA, traceSize := writeSurfaceTraceFixture(t, artifactDir)
	reportPath := filepath.Join(dir, "surface-headless.json")
	if err := os.WriteFile(reportPath, validSurfaceRuntimeReportJSON(artifactPath, artifactSHA, artifactSize, tracePath, traceSHA, traceSize), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := validateSurfaceRuntimeReport(reportPath); err != nil {
		t.Fatalf("validateSurfaceRuntimeReport failed: %v", err)
	}
}

func TestValidateSurfaceRuntimeReportAcceptsProductionTextInputSchema(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "surface-text-input.json")
	raw := []byte(`{
  "schema": "tetra.surface.text-input.v1",
  "target": "headless",
  "source": "examples/surface_release_text_input.tetra",
  "level": "production-text-input-v1",
  "experimental": false,
  "production_claim": true,
  "storage": "owned-utf8-byte-buffer",
  "utf8_validation": true,
  "caret": true,
  "selection": true,
  "backspace": true,
  "delete": true,
  "home_end": true,
  "arrow_left_right": true,
  "composition_events": true,
  "composition_commit": true,
  "composition_cancel": true,
  "clipboard_read": true,
  "clipboard_write": true,
  "clipboard_host_abi": true,
  "clipboard_owned_copy": true,
  "composition_trace": {"start":true,"update":true,"commit":true,"cancel":true},
  "borrowed_view_storage": false,
  "safe_view_lifetime_checked": true,
  "processes": [
    {"name":"tetra build","kind":"build","path":"tetra build --target linux-x64 examples/surface_release_text_input.tetra -o /tmp/surface-artifacts/surface-release-text-input","ran":true,"pass":true,"exit_code":0},
    {"name":"surface component app","kind":"app","path":"/tmp/surface-artifacts/surface-release-text-input","ran":true,"pass":true,"exit_code":1,"expected_exit_code":1},
    {"name":"surface headless runtime","kind":"runtime","path":"tools/cmd/surface-runtime-smoke --mode headless-release-text-input","ran":true,"pass":true,"exit_code":0}
  ],
  "artifacts": [
    {"kind":"component-app","path":"/tmp/surface-artifacts/surface-release-text-input","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":4096},
    {"kind":"runner-trace","path":"/tmp/surface-artifacts/surface-runner-trace.json","sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","size":2048}
  ],
  "artifact_scan": {"root":"/tmp/surface-artifacts","files_checked":2,"forbidden_paths":[],"pass":true},
  "cases": [
    {"name":"host-provided pointer event dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"host event buffer poll_event","kind":"positive","ran":true,"pass":true},
    {"name":"pre/post event frame sequence","kind":"positive","ran":true,"pass":true},
    {"name":"component hierarchy dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component text input scalar dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"host text payload buffer","kind":"positive","ran":true,"pass":true},
    {"name":"component focus dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component accessibility metadata","kind":"positive","ran":true,"pass":true},
    {"name":"no legacy UI sidecar artifacts","kind":"positive","ran":true,"pass":true},
    {"name":"release text input ASCII insertion","kind":"positive","ran":true,"pass":true},
    {"name":"release text input UTF-8 insertion","kind":"positive","ran":true,"pass":true},
    {"name":"release text input caret home end arrows","kind":"positive","ran":true,"pass":true},
    {"name":"release text input selection replacement","kind":"positive","ran":true,"pass":true},
    {"name":"release text input backspace delete","kind":"positive","ran":true,"pass":true},
    {"name":"release text input clipboard owned copy transfer","kind":"positive","ran":true,"pass":true},
    {"name":"release text input composition start update","kind":"positive","ran":true,"pass":true},
    {"name":"release text input composition commit","kind":"positive","ran":true,"pass":true},
    {"name":"release text input composition cancel","kind":"positive","ran":true,"pass":true},
    {"name":"release text input safe view lifetime checked","kind":"positive","ran":true,"pass":true},
    {"name":"reject legacy UI evidence","kind":"negative","ran":true,"pass":true,"expected_error":"legacy UI evidence rejected"}
  ]
}`)
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := validateSurfaceRuntimeReport(reportPath); err != nil {
		t.Fatalf("validateSurfaceRuntimeReport failed: %v", err)
	}
}

func TestTextInputReleaseValidatorAcceptsProductionTextInputReport(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "surface-text-input.json")
	if err := os.WriteFile(reportPath, validProductionTextInputReportJSON(t), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := validateSurfaceRuntimeReportWithOptions(reportPath, surfaceRuntimeValidationOptions{Release: "text-input"}); err != nil {
		t.Fatalf("validateSurfaceRuntimeReportWithOptions text-input failed: %v", err)
	}
}

func TestValidateSurfaceRuntimeReportAcceptsReleaseSummarySchema(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "surface-release-summary.json")
	raw := []byte(`{
  "schema": "tetra.surface.release.v1",
  "release_scope": "surface-v1-linux-web",
  "status": "current",
  "production_claim": true,
  "experimental": false,
  "producer": "scripts/release/surface/release-gate.sh",
  "git_head": "0123456789abcdef0123456789abcdef01234567",
  "version": "tetra_language",
  "git_dirty": false,
  "host_os": "linux",
  "host_arch": "amd64",
  "generated_at_utc": "2026-06-08T16:00:00Z",
  "command_line": "bash scripts/release/surface/release-gate.sh --report-dir reports/surface-release-v1",
  "supported_targets": ["headless", "linux-x64", "wasm32-web"],
  "runtime_targets": ["linux-x64", "wasm32-web"],
  "test_targets": ["headless"],
  "unsupported_targets": ["macos-x64", "windows-x64", "wasm32-wasi"],
  "host_abi": "tetra.surface.host.v1",
  "toolkit": "production-widgets-v1",
  "text_input": "production-text-input-v1",
  "clipboard": "clipboard-text-v1",
  "ime": "composition-baseline-v1",
  "accessibility": "platform-bridge-v1",
  "browser_surface": "browser-canvas-release-v1",
  "linux_surface": "linux-x64-release-window-v1",
  "block_system": "block-system",
  "block_system_gate": "tetra.surface.block-system.gate.v1",
  "morph": "morph-capsule",
  "morph_gate": "tetra.surface.morph.gate.v1",
  "artifact_hashes_validated": true,
  "legacy_sidecars": false,
  "dom_ui": false,
  "user_js": false,
  "platform_widgets": false
}`)
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := validateSurfaceRuntimeReport(reportPath); err != nil {
		t.Fatalf("validateSurfaceRuntimeReport failed: %v", err)
	}
}

func TestValidateSurfaceRuntimeReportReleaseModeAcceptsReleaseSummary(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "surface-release-summary.json")
	if err := os.WriteFile(reportPath, validSurfaceRuntimeReleaseSummaryJSON(), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := validateSurfaceRuntimeReportWithOptions(reportPath, surfaceRuntimeValidationOptions{Release: "surface-v1"}); err != nil {
		t.Fatalf("validateSurfaceRuntimeReportWithOptions failed: %v", err)
	}
}

func TestHeadlessReleaseValidatorAcceptsHeadlessRuntimeReport(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	artifactPath, artifactSHA, artifactSize := writeSurfaceArtifactFixture(t, artifactDir)
	tracePath, traceSHA, traceSize := writeSurfaceTraceFixture(t, artifactDir)
	reportPath := filepath.Join(dir, "surface-headless-release.json")
	raw := headlessReleaseRuntimeReportJSON(artifactPath, artifactSHA, artifactSize, tracePath, traceSHA, traceSize)
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := validateSurfaceRuntimeReportWithOptions(reportPath, surfaceRuntimeValidationOptions{Release: "headless"}); err != nil {
		t.Fatalf("validateSurfaceRuntimeReportWithOptions(headless) failed: %v\n%s", err, raw)
	}
}

func TestHeadlessReleaseValidatorRejectsLinuxOrBrowserSubstitute(t *testing.T) {
	dir := t.TempDir()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	artifactPath, artifactSHA, artifactSize := writeSurfaceArtifactFixture(t, artifactDir)
	tracePath, traceSHA, traceSize := writeSurfaceTraceFixture(t, artifactDir)
	reportPath := filepath.Join(dir, "surface-linux-x64.json")
	raw := strings.Replace(
		string(headlessReleaseRuntimeReportJSON(artifactPath, artifactSHA, artifactSize, tracePath, traceSHA, traceSize)),
		`"target": "headless"`,
		`"target": "linux-x64"`,
		1,
	)
	raw = strings.Replace(raw, `"runtime": "surface-headless"`, `"runtime": "surface-linux-x64"`, 1)
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err := validateSurfaceRuntimeReportWithOptions(reportPath, surfaceRuntimeValidationOptions{Release: "headless"})
	if err == nil {
		t.Fatalf("expected linux/browser substitute to fail headless release validation")
	}
	if !strings.Contains(err.Error(), "headless") {
		t.Fatalf("error = %v, want headless diagnostic", err)
	}
}

func TestBrowserReleaseRequiresChromium(t *testing.T) {
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
	processes := report["processes"].([]any)
	for _, item := range processes {
		process := item.(map[string]any)
		if process["name"] == "surface wasm32-web browser canvas component app" {
			process["path"] = "node scripts/tools/web_run_module.mjs surface-release-form.wasm"
		}
	}
	raw, err = json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}
	if err := os.WriteFile(reportPath, raw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	err = validateWASM32WebBrowserReleaseEnvelope(surface.SchemaV1, raw)
	if err == nil {
		t.Fatalf("expected Node-only browser release substitute to fail")
	}
	if !strings.Contains(err.Error(), "Chromium-compatible browser") {
		t.Fatalf("error = %v, want Chromium-compatible browser diagnostic", err)
	}
}

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

func surfaceArtifactFixtureDir(t *testing.T, dir string) string {
	t.Helper()
	artifactDir := filepath.Join(dir, "surface-artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatalf("create artifact fixture dir: %v", err)
	}
	return artifactDir
}

func writeSurfaceArtifactFixture(t *testing.T, dir string) (string, string, int64) {
	t.Helper()
	return writeNamedSurfaceArtifactFixture(t, dir, "surface-counter", []byte("surface component artifact fixture\n"), 0o755)
}

func writeNamedSurfaceArtifactFixture(t *testing.T, dir string, name string, contents []byte, perm os.FileMode) (string, string, int64) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, contents, perm); err != nil {
		t.Fatalf("write artifact fixture %s: %v", name, err)
	}
	sum := sha256.Sum256(contents)
	return path, "sha256:" + hex.EncodeToString(sum[:]), int64(len(contents))
}

func writeSurfaceTraceFixture(t *testing.T, dir string) (string, string, int64) {
	t.Helper()
	return writeSurfaceTraceFixtureWithSourceAndFrames(t, dir, "examples/surface_counter.tetra", []surfaceTraceFrameFixture{
		{Order: 1, Width: 320, Height: 200, Stride: 1280, Checksum: "8ff0b13bb6026d76c7f61b202436f30e703d45d96730033ee6d171bcf5802f81", Presented: true},
		{Order: 2, Width: 320, Height: 200, Stride: 1280, Checksum: "9ff0b13bb6026d76c7f61b202436f30e703d45d96730033ee6d171bcf5802f82", Presented: true},
	})
}

type surfaceTraceFrameFixture struct {
	Order     int    `json:"order"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Stride    int    `json:"stride"`
	Checksum  string `json:"checksum"`
	Presented bool   `json:"presented"`
}

func writeSurfaceTraceFixtureWithFrames(t *testing.T, dir string, frames []surfaceTraceFrameFixture) (string, string, int64) {
	t.Helper()
	return writeSurfaceTraceFixtureWithSourceAndFrames(t, dir, "examples/surface_counter.tetra", frames)
}

func writeSurfaceTraceFixtureWithSourceAndFrames(t *testing.T, dir string, source string, frames []surfaceTraceFrameFixture) (string, string, int64) {
	t.Helper()
	path := filepath.Join(dir, "surface-runner-trace.json")
	trace := struct {
		Schema string                     `json:"schema"`
		Source string                     `json:"source"`
		Frames []surfaceTraceFrameFixture `json:"frames"`
	}{
		Schema: "tetra.surface.headless-runner-trace.v1",
		Source: source,
		Frames: frames,
	}
	raw, err := json.Marshal(trace)
	if err != nil {
		t.Fatalf("marshal trace fixture: %v", err)
	}
	contents := append(raw, '\n')
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write trace fixture: %v", err)
	}
	sum := sha256.Sum256(contents)
	return path, "sha256:" + hex.EncodeToString(sum[:]), int64(len(contents))
}

type wasmTraceFrameFixture struct {
	Order     int    `json:"order"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Stride    int    `json:"stride"`
	PixelsLen int    `json:"pixels_len"`
	Checksum  string `json:"checksum"`
}

func writeWASMTraceFixtureWithFrames(t *testing.T, dir string, frames []wasmTraceFrameFixture) (string, string, int64) {
	t.Helper()
	return writeWASMTraceFixtureWithWASMAndFrames(t, dir, filepath.Join(dir, "surface-counter.wasm"), frames)
}

func writeWASMTraceFixtureWithWASMAndFrames(t *testing.T, dir string, wasmPath string, frames []wasmTraceFrameFixture) (string, string, int64) {
	t.Helper()
	path := filepath.Join(dir, "surface-runner-trace.json")
	trace := struct {
		Schema string                  `json:"schema"`
		WASM   string                  `json:"wasm_path"`
		Frames []wasmTraceFrameFixture `json:"frames"`
	}{
		Schema: "tetra.surface.web-runner-trace.v1",
		WASM:   wasmPath,
		Frames: frames,
	}
	raw, err := json.Marshal(trace)
	if err != nil {
		t.Fatalf("marshal wasm trace fixture: %v", err)
	}
	contents := append(raw, '\n')
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write wasm trace fixture: %v", err)
	}
	sum := sha256.Sum256(contents)
	return path, "sha256:" + hex.EncodeToString(sum[:]), int64(len(contents))
}

func writeBrowserCanvasTraceFixture(t *testing.T, dir string, wasmPath string) (string, string, int64) {
	t.Helper()
	return writeBrowserCanvasTraceFixtureWithChecksums(t, dir, wasmPath,
		"1111111111111111111111111111111111111111111111111111111111111111",
		"5555555555555555555555555555555555555555555555555555555555555555",
		"5555555555555555555555555555555555555555555555555555555555555555",
	)
}

func writeBrowserCanvasTraceFixtureWithChecksums(t *testing.T, dir string, wasmPath string, firstChecksum string, secondSourceChecksum string, secondCanvasChecksum string) (string, string, int64) {
	t.Helper()
	path := filepath.Join(dir, "surface-runner-trace.json")
	trace := struct {
		Schema string `json:"schema"`
		WASM   string `json:"wasm_path"`
		Canvas struct {
			Opened   bool `json:"opened"`
			Readback bool `json:"readback"`
			Width    int  `json:"width"`
			Height   int  `json:"height"`
		} `json:"canvas"`
		BrowserEvents []runnerTraceEvent `json:"browser_events"`
		Frames        []runnerTraceFrame `json:"frames"`
		AppExitCode   int                `json:"app_exit_code"`
	}{
		Schema: "tetra.surface.browser-canvas-trace.v1",
		WASM:   wasmPath,
		BrowserEvents: []runnerTraceEvent{
			{NativeType: "pointerup", Kind: 5},
			{NativeType: "keydown", Kind: 6},
			{NativeType: "resize", Kind: 2},
			{NativeType: "beforeinput", Kind: 8},
		},
		Frames: []runnerTraceFrame{
			{Order: 1, Width: 320, Height: 200, Stride: 1280, PixelsLen: 256000, SourceChecksum: firstChecksum, CanvasChecksum: firstChecksum, Checksum: firstChecksum, Presented: true},
			{Order: 5, Width: 400, Height: 240, Stride: 1600, PixelsLen: 384000, SourceChecksum: secondSourceChecksum, CanvasChecksum: secondCanvasChecksum, Checksum: secondCanvasChecksum, Presented: true},
		},
		AppExitCode: 1,
	}
	trace.Canvas.Opened = true
	trace.Canvas.Readback = true
	trace.Canvas.Width = 400
	trace.Canvas.Height = 240
	raw, err := json.Marshal(trace)
	if err != nil {
		t.Fatalf("marshal browser canvas trace fixture: %v", err)
	}
	contents := append(raw, '\n')
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write browser canvas trace fixture: %v", err)
	}
	sum := sha256.Sum256(contents)
	return path, "sha256:" + hex.EncodeToString(sum[:]), int64(len(contents))
}

func writeWASM32WebBrowserReleaseRuntimeReport(t *testing.T, dir string) string {
	t.Helper()
	artifactDir := surfaceArtifactFixtureDir(t, dir)
	wasmPath, wasmSHA, wasmSize := writeNamedSurfaceArtifactFixture(t, artifactDir, "surface-release-form.wasm", []byte("\x00asm\x01\x00\x00\x00surface browser release wasm fixture\n"), 0o755)
	loaderPath, loaderSHA, loaderSize := writeNamedSurfaceArtifactFixture(t, artifactDir, "surface-release-form.mjs", []byte(`function createSurfaceHost(instanceRef) {
  return { __tetra_surface_present_rgba() { return 0; } };
}
const imports = { tetra_surface_host_v1: createSurfaceHost({ instance: null }) };
`), 0o644)
	tracePath, traceSHA, traceSize := writeBrowserReleaseTraceFixture(t, artifactDir, wasmPath)
	reportPath := filepath.Join(dir, "surface-wasm32-web-release-browser.json")
	raw := string(validWASM32WebBrowserCanvasSurfaceRuntimeReportJSON(wasmPath, wasmSHA, wasmSize, loaderPath, loaderSHA, loaderSize, tracePath, traceSHA, traceSize))
	raw = strings.Replace(raw, `"source": "examples/surface_browser_counter.tetra"`, `"source": "examples/surface_release_form.tetra"`, 1)
	raw = strings.Replace(raw,
		`"host_evidence": {"level":"wasm32-web-browser-canvas-input","backend":"browser-canvas-rgba","framebuffer":true,"real_window":false,"native_input":true,"user_facing_platform_widgets":false}`,
		`"host_evidence": {"level":"wasm32-web-browser-canvas-release-v1","backend":"browser-canvas-rgba-accessible","framebuffer":true,"real_window":false,"native_input":true,"browser_canvas":true,"browser_input":true,"browser_clipboard":true,"browser_clipboard_harness":"deterministic-browser-clipboard-v1","browser_composition":true,"browser_accessibility_snapshot":true,"browser_accessibility_mirror":true,"user_facing_platform_widgets":false}`,
		1,
	)
	raw = strings.Replace(raw, `examples/surface_browser_counter.tetra`, `examples/surface_release_form.tetra`, 1)
	raw = strings.Replace(raw, `<surface-browser-canvas-runner> wasm=`, `<surface-browser-canvas-runner> scenario=release-browser wasm=`, 1)
	if err := os.WriteFile(reportPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	return reportPath
}

func writeBrowserReleaseTraceFixture(t *testing.T, dir string, wasmPath string) (string, string, int64) {
	t.Helper()
	path := filepath.Join(dir, "surface-runner-trace.json")
	trace := runnerTraceEnvelope{
		Schema: "tetra.surface.browser-canvas-trace.v1",
		WASM:   wasmPath,
		Canvas: runnerTraceCanvas{
			Opened:   true,
			Readback: true,
			Width:    400,
			Height:   240,
		},
		BrowserEvents: []runnerTraceEvent{
			{NativeType: "pointerup", Kind: 5},
			{NativeType: "keydown", Kind: 6},
			{NativeType: "resize", Kind: 2},
			{NativeType: "beforeinput", Kind: 8},
			{NativeType: "compositionstart", Kind: 9},
			{NativeType: "compositionupdate", Kind: 9},
			{NativeType: "compositionend", Kind: 9},
		},
		BrowserClipboard: runnerTraceClipboard{
			Harness:   "deterministic-browser-clipboard-v1",
			Read:      true,
			Write:     true,
			OwnedCopy: true,
			Bytes:     13,
		},
		BrowserComposition: runnerTraceComposition{
			Start:  true,
			Update: true,
			Commit: true,
			Cancel: true,
		},
		BrowserAccessibility: runnerTraceAccessibility{
			Snapshot:      true,
			Mirror:        true,
			CompilerOwned: true,
			Roles:         []string{"root", "textbox", "checkbox", "button", "status"},
			Bounds:        true,
			Focus:         true,
		},
		Frames: []runnerTraceFrame{
			{Order: 1, Width: 320, Height: 200, Stride: 1280, PixelsLen: 256000, SourceChecksum: "1111111111111111111111111111111111111111111111111111111111111111", CanvasChecksum: "1111111111111111111111111111111111111111111111111111111111111111", Checksum: "1111111111111111111111111111111111111111111111111111111111111111", Presented: true},
			{Order: 5, Width: 400, Height: 240, Stride: 1600, PixelsLen: 384000, SourceChecksum: "5555555555555555555555555555555555555555555555555555555555555555", CanvasChecksum: "5555555555555555555555555555555555555555555555555555555555555555", Checksum: "5555555555555555555555555555555555555555555555555555555555555555", Presented: true},
		},
	}
	exit := 1
	trace.AppExitCode = &exit
	raw, err := json.Marshal(trace)
	if err != nil {
		t.Fatalf("marshal browser release trace fixture: %v", err)
	}
	contents := append(raw, '\n')
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write browser release trace fixture: %v", err)
	}
	sum := sha256.Sum256(contents)
	return path, "sha256:" + hex.EncodeToString(sum[:]), int64(len(contents))
}

func validSurfaceRuntimeReportJSON(artifactPath string, artifactSHA string, artifactSize int64, tracePath string, traceSHA string, traceSize int64) []byte {
	buildPath := "tetra build --target linux-x64 examples/surface_counter.tetra -o " + artifactPath
	artifactScanRoot := filepath.Dir(artifactPath)
	raw := `{
  "schema": "tetra.surface.runtime.v1",
  "status": "pass",
  "target": "headless",
  "host": "linux-x64",
  "runtime": "surface-headless",
  "surface_schema": "tetra.surface.v1",
  "host_abi": "tetra.surface.host-abi.v1",
  "host_evidence": {"level":"deterministic-headless","backend":"software-rgba","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false},
  "source": "examples/surface_counter.tetra",
  "processes": [
    {"name":"tetra build","kind":"build","path":__BUILD_PROCESS_PATH__,"ran":true,"pass":true,"exit_code":0},
    {"name":"surface component app","kind":"app","path":__ARTIFACT_PATH__,"ran":true,"pass":true,"exit_code":1,"expected_exit_code":1},
    {"name":"surface headless runtime","kind":"runtime","path":"tools/cmd/surface-runtime-smoke","ran":true,"pass":true,"exit_code":0}
  ],
  "artifacts": [
    {"kind":"component-app","path":__ARTIFACT_PATH__,"sha256":__ARTIFACT_SHA__,"size":__ARTIFACT_SIZE__},
    {"kind":"runner-trace","path":__TRACE_PATH__,"sha256":__TRACE_SHA__,"size":__TRACE_SIZE__}
  ],
  "artifact_scan": {"root":__ARTIFACT_SCAN_ROOT__,"files_checked":2,"forbidden_paths":[],"pass":true},
  "components": [
    {"id":"CounterApp","type":"examples.surface_counter.CounterApp","bounds":{"x":0,"y":0,"w":320,"h":200},"abilities":["measure","layout","draw","event","focus","text","accessibility"],"state":{"count":"1","text_count":"1","accessibility_role":"button"}},
    {"id":"CounterButton","type":"examples.surface_counter.CounterButton","parent":"CounterApp","bounds":{"x":32,"y":80,"w":160,"h":48},"abilities":["measure","layout","draw","event","focus","text","accessibility"],"state":{"pressed":"false","focused":"true","text_len_seen":"2","accessibility_role":"button"}}
  ],
  "events": [
    {"order":1,"kind":"none","target_component":"CounterApp","dispatch_path":["CounterApp"],"handled":false,"pass":true,"x":0,"y":0,"before_state":{"CounterApp.count":"0"},"after_state":{"CounterApp.count":"0"}},
    {"order":2,"kind":"mouse_up","target_component":"CounterButton","dispatch_path":["CounterApp","CounterButton"],"handled":true,"pass":true,"x":48,"y":96,"key":0,"width":320,"height":200,"timestamp_ms":0,"buffer_slots":[5,48,96,1,0,320,200,0,0],"before_state":{"CounterApp.count":"0","CounterButton.pressed":"false"},"after_state":{"CounterApp.count":"1","CounterButton.pressed":"false"}},
    {"order":3,"kind":"text_input","target_component":"CounterButton","dispatch_path":["CounterApp","CounterButton"],"handled":true,"pass":true,"x":0,"y":0,"key":0,"width":320,"height":200,"timestamp_ms":1,"text_len":2,"text_bytes_hex":"4f4b","buffer_slots":[8,0,0,0,0,320,200,1,2],"before_state":{"CounterApp.text_count":"0","CounterButton.text_len_seen":"0"},"after_state":{"CounterApp.text_count":"1","CounterButton.text_len_seen":"2"}}
  ],
  "frames": [
    {"order":1,"width":320,"height":200,"stride":1280,"checksum":"8ff0b13bb6026d76c7f61b202436f30e703d45d96730033ee6d171bcf5802f81","presented":true},
    {"order":2,"width":320,"height":200,"stride":1280,"checksum":"9ff0b13bb6026d76c7f61b202436f30e703d45d96730033ee6d171bcf5802f82","presented":true}
  ],
  "state_transitions": [
    {"order":1,"component":"CounterApp","field":"count","before":"0","after":"1","cause":"mouse_up"},
    {"order":2,"component":"CounterApp","field":"text_count","before":"0","after":"1","cause":"text_input"}
  ],
  "cases": [
    {"name":"pure Tetra component app","kind":"positive","ran":true,"pass":true},
    {"name":"headless event dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"headless framebuffer checksum","kind":"positive","ran":true,"pass":true},
    {"name":"headless actual runner trace","kind":"positive","ran":true,"pass":true},
    {"name":"host-provided pointer event dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"host event buffer poll_event","kind":"positive","ran":true,"pass":true},
    {"name":"pre/post event frame sequence","kind":"positive","ran":true,"pass":true},
    {"name":"component hierarchy dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component text input scalar dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"host text payload buffer","kind":"positive","ran":true,"pass":true},
    {"name":"component focus dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component accessibility metadata","kind":"positive","ran":true,"pass":true},
    {"name":"no legacy UI sidecar artifacts","kind":"positive","ran":true,"pass":true},
    {"name":"state transition","kind":"positive","ran":true,"pass":true},
    {"name":"reject legacy UI evidence","kind":"negative","ran":true,"pass":true,"expected_error":"legacy UI evidence rejected"}
  ]
}`
	raw = strings.NewReplacer(
		"__BUILD_PROCESS_PATH__", jsonString(buildPath),
		"__ARTIFACT_PATH__", jsonString(artifactPath),
		"__ARTIFACT_SHA__", jsonString(artifactSHA),
		"__ARTIFACT_SIZE__", strconv.FormatInt(artifactSize, 10),
		"__TRACE_PATH__", jsonString(tracePath),
		"__TRACE_SHA__", jsonString(traceSHA),
		"__TRACE_SIZE__", strconv.FormatInt(traceSize, 10),
		"__ARTIFACT_SCAN_ROOT__", jsonString(artifactScanRoot),
	).Replace(raw)
	return []byte(raw)
}

func validProductionTextInputReportJSON(t *testing.T) []byte {
	t.Helper()
	intRef := func(v int) *int { return &v }
	report := surface.TextInputReport{
		Schema:             surface.TextInputSchemaV1,
		Target:             "headless",
		Source:             "examples/surface_release_text_input.tetra",
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
		Processes: []surface.ProcessReport{
			{Name: "tetra build", Kind: "build", Path: "tetra build --target linux-x64 examples/surface_release_text_input.tetra -o /tmp/surface-artifacts/surface-release-text-input", Ran: true, Pass: true, ExitCode: intRef(0)},
			{Name: "surface component app", Kind: "app", Path: "/tmp/surface-artifacts/surface-release-text-input", Ran: true, Pass: true, ExitCode: intRef(1), ExpectedExitCode: intRef(1)},
			{Name: "surface headless runtime", Kind: "runtime", Path: "tools/cmd/surface-runtime-smoke --mode headless-release-text-input", Ran: true, Pass: true, ExitCode: intRef(0)},
		},
		Artifacts: []surface.ArtifactReport{
			{Kind: "component-app", Path: "/tmp/surface-artifacts/surface-release-text-input", SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Size: 4096},
			{Kind: "runner-trace", Path: "/tmp/surface-artifacts/surface-runner-trace.json", SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Size: 2048},
		},
		ArtifactScan: surface.ArtifactScanReport{Root: "/tmp/surface-artifacts", FilesChecked: 2, ForbiddenPaths: []string{}, Pass: true},
		Cases: []surface.CaseReport{
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
		},
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal text input report: %v", err)
	}
	return raw
}

func headlessReleaseRuntimeReportJSON(artifactPath string, artifactSHA string, artifactSize int64, tracePath string, traceSHA string, traceSize int64) []byte {
	return validSurfaceRuntimeReportJSON(artifactPath, artifactSHA, artifactSize, tracePath, traceSHA, traceSize)
}

func validWASM32WebSurfaceRuntimeReportJSON(wasmPath string, wasmSHA string, wasmSize int64, loaderPath string, loaderSHA string, loaderSize int64, tracePath string, traceSHA string, traceSize int64) []byte {
	raw := string(validSurfaceRuntimeReportJSON(wasmPath, wasmSHA, wasmSize, tracePath, traceSHA, traceSize))
	replacements := []struct {
		old string
		new string
	}{
		{old: `"target": "headless"`, new: `"target": "wasm32-web"`},
		{old: `"runtime": "surface-headless"`, new: `"runtime": "surface-wasm32-web"`},
		{old: `"host_evidence": {"level":"deterministic-headless","backend":"software-rgba","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false}`,
			new: `"host_evidence": {"level":"wasm32-web-compiler-owned-loader","backend":"node-surface-host","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false}`},
		{old: `"tetra build --target linux-x64 examples/surface_counter.tetra -o `, new: `"tetra build --target wasm32-web examples/surface_counter.tetra -o `},
		{old: `"name":"surface component app","kind":"app","path":` + jsonString(wasmPath) + `,"ran":true,"pass":true,"exit_code":1,"expected_exit_code":1}`,
			new: `"name":"surface wasm32-web component app","kind":"app","path":` + jsonString("node scripts/tools/web_run_module.mjs --surface-trace "+tracePath+" "+wasmPath) + `,"ran":true,"pass":true,"exit_code":1,"expected_exit_code":1},
    {"name":"surface wasm32-web import validator","kind":"runtime","path":` + jsonString("go run ./tools/cmd/validate-wasm-imports --target wasm32-web "+wasmPath) + `,"ran":true,"pass":true,"exit_code":0}`},
		{old: `"surface headless runtime"`, new: `"surface wasm32-web runtime"`},
		{old: `"headless event dispatch"`, new: `"wasm32-web Surface Host ABI imports"`},
		{old: `"headless framebuffer checksum"`, new: `"wasm32-web framebuffer checksum evidence"`},
		{old: `"headless actual runner trace"`, new: `"wasm32-web runner trace"`},
		{old: `"artifact_scan": {"root":` + jsonString(filepath.Dir(wasmPath)) + `,"files_checked":2`, new: `"artifact_scan": {"root":` + jsonString(filepath.Dir(wasmPath)) + `,"files_checked":3`},
	}
	for _, repl := range replacements {
		raw = strings.Replace(raw, repl.old, repl.new, 1)
	}
	raw = strings.Replace(raw,
		`{"kind":"component-app","path":`+jsonString(wasmPath)+`,"sha256":`+jsonString(wasmSHA)+`,"size":`+strconv.FormatInt(wasmSize, 10)+`},
    {"kind":"runner-trace","path":`+jsonString(tracePath)+`,"sha256":`+jsonString(traceSHA)+`,"size":`+strconv.FormatInt(traceSize, 10)+`}`,
		`{"kind":"component-app","path":`+jsonString(wasmPath)+`,"sha256":`+jsonString(wasmSHA)+`,"size":`+strconv.FormatInt(wasmSize, 10)+`},
    {"kind":"compiler-owned-loader","path":`+jsonString(loaderPath)+`,"sha256":`+jsonString(loaderSHA)+`,"size":`+strconv.FormatInt(loaderSize, 10)+`},
    {"kind":"runner-trace","path":`+jsonString(tracePath)+`,"sha256":`+jsonString(traceSHA)+`,"size":`+strconv.FormatInt(traceSize, 10)+`}`, 1)
	raw = strings.Replace(raw, `{"order":2,"width":320,"height":200,"stride":1280,"checksum":"9ff0b13bb6026d76c7f61b202436f30e703d45d96730033ee6d171bcf5802f82","presented":true}`,
		`{"order":2,"width":320,"height":200,"stride":1280,"checksum":"9ff0b13bb6026d76c7f61b202436f30e703d45d96730033ee6d171bcf5802f82","presented":true},
    {"order":3,"width":320,"height":200,"stride":1280,"checksum":"3333333333333333333333333333333333333333333333333333333333333333","presented":true},
    {"order":4,"width":320,"height":200,"stride":1280,"checksum":"4444444444444444444444444444444444444444444444444444444444444444","presented":true}`, 1)
	raw = strings.Replace(raw, `{"name":"wasm32-web framebuffer checksum evidence","kind":"positive","ran":true,"pass":true}`,
		`{"name":"wasm32-web framebuffer checksum evidence","kind":"positive","ran":true,"pass":true},
    {"name":"compiler-owned wasm Surface loader","kind":"positive","ran":true,"pass":true},
    {"name":"wasm32-web actual presented frame trace","kind":"positive","ran":true,"pass":true}`, 1)
	return []byte(raw)
}

func validWASM32WebBrowserCanvasSurfaceRuntimeReportJSON(wasmPath string, wasmSHA string, wasmSize int64, loaderPath string, loaderSHA string, loaderSize int64, tracePath string, traceSHA string, traceSize int64) []byte {
	raw := `{
  "schema": "tetra.surface.runtime.v1",
  "status": "pass",
  "target": "wasm32-web",
  "host": "linux-x64",
  "runtime": "surface-wasm32-web",
  "surface_schema": "tetra.surface.v1",
  "host_abi": "tetra.surface.host-abi.v1",
  "host_evidence": {"level":"wasm32-web-browser-canvas-input","backend":"browser-canvas-rgba","framebuffer":true,"real_window":false,"native_input":true,"user_facing_platform_widgets":false},
  "source": "examples/surface_browser_counter.tetra",
  "processes": [
    {"name":"tetra build","kind":"build","path":__BUILD_PROCESS_PATH__,"ran":true,"pass":true,"exit_code":0},
    {"name":"surface wasm32-web browser canvas component app","kind":"app","path":__BROWSER_PROCESS_PATH__,"ran":true,"pass":true,"exit_code":0,"expected_exit_code":0},
    {"name":"surface wasm32-web import validator","kind":"runtime","path":__IMPORT_VALIDATOR_PROCESS_PATH__,"ran":true,"pass":true,"exit_code":0},
    {"name":"surface wasm32-web browser canvas runtime","kind":"runtime","path":"Chromium fixture","ran":true,"pass":true,"exit_code":0}
  ],
  "artifacts": [
    {"kind":"component-app","path":__WASM_PATH__,"sha256":__WASM_SHA__,"size":__WASM_SIZE__},
    {"kind":"compiler-owned-loader","path":__LOADER_PATH__,"sha256":__LOADER_SHA__,"size":__LOADER_SIZE__},
    {"kind":"runner-trace","path":__TRACE_PATH__,"sha256":__TRACE_SHA__,"size":__TRACE_SIZE__}
  ],
  "artifact_scan": {"root":__ARTIFACT_SCAN_ROOT__,"files_checked":3,"forbidden_paths":[],"pass":true},
  "components": [
    {"id":"CounterApp","type":"examples.surface_browser_counter.CounterApp","bounds":{"x":0,"y":0,"w":400,"h":240},"abilities":["measure","layout","draw","event","focus","text","accessibility"],"state":{"count":"2","key_count":"1","width":"400","accessibility_role":"button"}},
    {"id":"CounterButton","type":"examples.surface_browser_counter.CounterButton","parent":"CounterApp","bounds":{"x":32,"y":88,"w":160,"h":48},"abilities":["measure","layout","draw","event","focus","text","accessibility"],"state":{"focused":"true","text_len_seen":"2"}}
  ],
  "events": [
    {"order":1,"kind":"mouse_up","target_component":"CounterButton","dispatch_path":["CounterApp","CounterButton"],"handled":true,"pass":true,"x":48,"y":96,"key":0,"width":320,"height":200,"timestamp_ms":0,"buffer_slots":[5,48,96,1,0,320,200,0,0],"before_state":{"CounterApp.count":"0"},"after_state":{"CounterApp.count":"1"}},
    {"order":2,"kind":"key_down","target_component":"CounterApp","dispatch_path":["CounterApp"],"handled":true,"pass":true,"x":0,"y":0,"key":32,"width":320,"height":200,"timestamp_ms":1,"buffer_slots":[6,0,0,0,32,320,200,1,0],"before_state":{"CounterApp.count":"1","CounterApp.key_count":"0"},"after_state":{"CounterApp.count":"2","CounterApp.key_count":"1"}},
    {"order":3,"kind":"resize","target_component":"CounterApp","dispatch_path":["CounterApp"],"handled":true,"pass":true,"x":0,"y":0,"key":0,"width":400,"height":240,"timestamp_ms":2,"buffer_slots":[2,0,0,0,0,400,240,2,0],"before_state":{"CounterApp.width":"320"},"after_state":{"CounterApp.width":"400"}},
    {"order":4,"kind":"text_input","target_component":"CounterButton","dispatch_path":["CounterApp","CounterButton"],"handled":true,"pass":true,"x":0,"y":0,"key":0,"width":400,"height":240,"timestamp_ms":3,"text_len":2,"text_bytes_hex":"4f4b","buffer_slots":[8,0,0,0,0,400,240,3,2],"before_state":{"CounterButton.text_len_seen":"0"},"after_state":{"CounterButton.text_len_seen":"2"}}
  ],
  "frames": [
    {"order":1,"width":320,"height":200,"stride":1280,"checksum":"1111111111111111111111111111111111111111111111111111111111111111","presented":true},
    {"order":5,"width":400,"height":240,"stride":1600,"checksum":"5555555555555555555555555555555555555555555555555555555555555555","presented":true}
  ],
  "state_transitions": [
    {"order":1,"component":"CounterApp","field":"count","before":"0","after":"1","cause":"mouse_up"},
    {"order":2,"component":"CounterApp","field":"key_count","before":"0","after":"1","cause":"key_down"},
    {"order":3,"component":"CounterApp","field":"width","before":"320","after":"400","cause":"resize"},
    {"order":4,"component":"CounterButton","field":"text_len_seen","before":"0","after":"2","cause":"text_input"}
  ],
  "cases": [
    {"name":"pure Tetra component app","kind":"positive","ran":true,"pass":true},
    {"name":"wasm32-web Surface Host ABI imports","kind":"positive","ran":true,"pass":true},
    {"name":"compiler-owned wasm Surface loader","kind":"positive","ran":true,"pass":true},
    {"name":"wasm32-web browser canvas surface","kind":"positive","ran":true,"pass":true},
    {"name":"wasm32-web browser canvas RGBA readback","kind":"positive","ran":true,"pass":true},
    {"name":"wasm32-web browser canvas pointer input","kind":"positive","ran":true,"pass":true},
    {"name":"wasm32-web browser canvas keyboard input","kind":"positive","ran":true,"pass":true},
    {"name":"wasm32-web browser canvas resize input","kind":"positive","ran":true,"pass":true},
    {"name":"wasm32-web browser canvas text input","kind":"positive","ran":true,"pass":true},
    {"name":"compiler-owned browser canvas Surface host","kind":"positive","ran":true,"pass":true},
    {"name":"host-provided pointer event dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"host event buffer poll_event","kind":"positive","ran":true,"pass":true},
    {"name":"pre/post event frame sequence","kind":"positive","ran":true,"pass":true},
    {"name":"component hierarchy dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component text input scalar dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"host text payload buffer","kind":"positive","ran":true,"pass":true},
    {"name":"component focus dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component accessibility metadata","kind":"positive","ran":true,"pass":true},
    {"name":"no legacy UI sidecar artifacts","kind":"positive","ran":true,"pass":true},
    {"name":"state transition","kind":"positive","ran":true,"pass":true},
    {"name":"reject legacy UI evidence","kind":"negative","ran":true,"pass":true,"expected_error":"legacy UI evidence rejected"}
  ]
}`
	raw = strings.NewReplacer(
		"__BUILD_PROCESS_PATH__", jsonString("tetra build --target wasm32-web examples/surface_browser_counter.tetra -o "+wasmPath),
		"__BROWSER_PROCESS_PATH__", jsonString("/usr/bin/chromium --headless <surface-browser-canvas-runner> wasm="+wasmPath),
		"__IMPORT_VALIDATOR_PROCESS_PATH__", jsonString("go run ./tools/cmd/validate-wasm-imports --target wasm32-web "+wasmPath),
		"__WASM_PATH__", jsonString(wasmPath),
		"__WASM_SHA__", jsonString(wasmSHA),
		"__WASM_SIZE__", strconv.FormatInt(wasmSize, 10),
		"__LOADER_PATH__", jsonString(loaderPath),
		"__LOADER_SHA__", jsonString(loaderSHA),
		"__LOADER_SIZE__", strconv.FormatInt(loaderSize, 10),
		"__TRACE_PATH__", jsonString(tracePath),
		"__TRACE_SHA__", jsonString(traceSHA),
		"__TRACE_SIZE__", strconv.FormatInt(traceSize, 10),
		"__ARTIFACT_SCAN_ROOT__", jsonString(filepath.Dir(wasmPath)),
	).Replace(raw)
	return []byte(raw)
}

func validSurfaceRuntimeReleaseSummaryJSON() []byte {
	return []byte(`{
  "schema": "tetra.surface.release.v1",
  "release_scope": "surface-v1-linux-web",
  "status": "current",
  "production_claim": true,
  "experimental": false,
  "producer": "scripts/release/surface/release-gate.sh",
  "git_head": "0123456789abcdef0123456789abcdef01234567",
  "version": "tetra_language",
  "git_dirty": false,
  "host_os": "linux",
  "host_arch": "amd64",
  "generated_at_utc": "2026-06-08T16:00:00Z",
  "command_line": "bash scripts/release/surface/release-gate.sh --report-dir reports/surface-release-v1",
  "supported_targets": ["headless", "linux-x64", "wasm32-web"],
  "runtime_targets": ["linux-x64", "wasm32-web"],
  "test_targets": ["headless"],
  "unsupported_targets": ["macos-x64", "windows-x64", "wasm32-wasi"],
  "host_abi": "tetra.surface.host.v1",
  "toolkit": "production-widgets-v1",
  "text_input": "production-text-input-v1",
  "clipboard": "clipboard-text-v1",
  "ime": "composition-baseline-v1",
  "accessibility": "platform-bridge-v1",
  "browser_surface": "browser-canvas-release-v1",
  "linux_surface": "linux-x64-release-window-v1",
  "block_system": "block-system",
  "block_system_gate": "tetra.surface.block-system.gate.v1",
  "morph": "morph-capsule",
  "morph_gate": "tetra.surface.morph.gate.v1",
  "artifact_hashes_validated": true,
  "legacy_sidecars": false,
  "dom_ui": false,
  "user_js": false,
  "platform_widgets": false
}`)
}

func releaseToolkitSliceReportForTest(target string, host surface.HostEvidenceReport) surface.Report {
	return surface.Report{
		Schema:           surface.SchemaV1,
		Status:           "pass",
		Target:           target,
		Source:           "examples/surface_release_form.tetra",
		HostEvidence:     host,
		ComponentTree:    &surface.ComponentTreeReport{},
		ComponentTreeAPI: &surface.ComponentTreeAPIReport{},
		Toolkit: &surface.ToolkitReport{
			ToolkitLevel:         "production-widgets-v1",
			ReleaseScope:         surface.ReleaseScopeSurfaceV1LinuxWeb,
			Experimental:         false,
			ProductionClaim:      true,
			NoDOMUI:              true,
			NoUserJS:             true,
			NoPlatformWidgets:    true,
			UsesComponentTreeAPI: true,
		},
	}
}

func releaseAccessibilitySliceReportForTest(target string, host surface.HostEvidenceReport) surface.Report {
	tree := &surface.AccessibilityTreeReport{
		AccessibilityLevel:         "platform-bridge-v1",
		ReleaseScope:               surface.ReleaseScopeSurfaceV1LinuxWeb,
		Experimental:               false,
		ProductionClaim:            true,
		MetadataTree:               true,
		PlatformExport:             true,
		PlatformBridge:             "platform-tree-probe",
		PlatformHostIntegration:    true,
		BrowserAccessibilitySnap:   target == "wasm32-web",
		BrowserAccessibilityMirror: target == "wasm32-web",
	}
	if target == "linux-x64" {
		tree.PlatformBridge = "linux_accessibility_host_bridge_v1"
		tree.LinuxPlatformProbe = true
		tree.LinuxProbeArtifact = "/tmp/surface-artifacts/surface-linux-accessibility-probe.json"
	}
	return surface.Report{
		Schema:            surface.SchemaV1,
		Status:            "pass",
		Target:            target,
		Source:            "examples/surface_release_accessibility.tetra",
		HostEvidence:      host,
		ComponentTree:     &surface.ComponentTreeReport{},
		ComponentTreeAPI:  &surface.ComponentTreeAPIReport{},
		AccessibilityTree: tree,
	}
}

func jsonString(value string) string {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(raw)
}
