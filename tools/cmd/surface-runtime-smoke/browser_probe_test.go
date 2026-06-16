package main

import (
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tetra_language/tools/validators/surface"
)

func TestCollectWASM32WebBrowserCanvasProcessEvidenceRecordsBrowserTrace(t *testing.T) {
	if _, err := discoverBrowserRunner(); err != nil {
		t.Skipf("browser runner is required for wasm32-web browser canvas Surface evidence: %v", err)
	}
	source, err := resolveSurfaceSourcePath("examples/surface_browser_counter.tetra")
	if err != nil {
		t.Fatalf("resolve Surface browser source: %v", err)
	}
	evidence, err := collectWASM32WebBrowserCanvasProcessEvidence(source, t.TempDir(), "counter")
	if err != nil {
		t.Fatalf("collectWASM32WebBrowserCanvasProcessEvidence: %v", err)
	}
	if len(evidence.Frames) < 2 {
		t.Fatalf("frames = %#v, want actual browser canvas pre/post frames", evidence.Frames)
	}
	after := evidence.Frames[len(evidence.Frames)-1]
	wantFrame := renderBrowserCounterFrameRGBA(2, 1, 400, 240, true)
	if after.Order != 5 || after.Width != 400 || after.Height != 240 || after.Stride != 1600 || !after.Presented {
		t.Fatalf("last browser frame = %#v, want order-5 400x240 presented frame", after)
	}
	if want := checksumRGBA(wantFrame.Pixels); after.Checksum != want {
		t.Fatalf("last browser frame checksum = %s, want actual browser CounterApp checksum %s", after.Checksum, want)
	}
	trace := artifactByKind(evidence.Artifacts, "runner-trace")
	if trace == nil {
		t.Fatalf("artifacts = %#v, want browser canvas runner trace", evidence.Artifacts)
	}
	raw, err := os.ReadFile(trace.Path)
	if err != nil {
		t.Fatalf("read browser runner trace: %v", err)
	}
	for _, want := range []string{`"schema": "tetra.surface.browser-canvas-trace.v1"`, `"native_type": "pointerup"`, `"native_type": "keydown"`, `"native_type": "resize"`, `"native_type": "beforeinput"`, `"source_checksum"`, `"canvas_checksum"`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("browser trace missing %q:\n%s", want, raw)
		}
	}
}

func TestRunBrowserCanvasTraceRetriesPendingTrace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake browser script uses POSIX shell")
	}

	root := t.TempDir()
	hostDir := filepath.Join(root, "scripts", "tools")
	if err := os.MkdirAll(hostDir, 0o755); err != nil {
		t.Fatalf("mkdir host dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hostDir, "surface_browser_canvas_host.mjs"), []byte("export async function runSurfaceBrowserCanvas() {}\n"), 0o644); err != nil {
		t.Fatalf("write fake browser host: %v", err)
	}
	wasmPath := filepath.Join(root, "app.wasm")
	if err := os.WriteFile(wasmPath, []byte{0, 97, 115, 109}, 0o644); err != nil {
		t.Fatalf("write fake wasm: %v", err)
	}

	countPath := filepath.Join(root, "fake-browser-count")
	browserPath := filepath.Join(root, "fake-browser.sh")
	fakeBrowser := `#!/usr/bin/env bash
set -euo pipefail
count=0
if [[ -f "$FAKE_BROWSER_COUNT" ]]; then
  count="$(cat "$FAKE_BROWSER_COUNT")"
fi
count=$((count + 1))
printf '%s\n' "$count" >"$FAKE_BROWSER_COUNT"
if [[ "$count" == "1" ]]; then
  printf '<html><body><pre id="surface-trace">pending</pre></body></html>\n'
  exit 0
fi
cat <<'HTML'
<html><body><pre id="surface-trace">{"schema":"tetra.surface.browser-canvas-trace.v1","wasm_path":"fake","canvas":{"opened":true,"width":320,"height":200,"readback":true},"browser_events":[],"frames":[],"app_exit_code":1}</pre></body></html>
HTML
`
	if err := os.WriteFile(browserPath, []byte(fakeBrowser), 0o755); err != nil {
		t.Fatalf("write fake browser: %v", err)
	}
	t.Setenv("FAKE_BROWSER_COUNT", countPath)

	trace, _, exitCode, err := runBrowserCanvasTrace(root, browserPath, wasmPath, "counter")
	if err != nil {
		t.Fatalf("runBrowserCanvasTrace: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("browser exit code = %d, want 0", exitCode)
	}
	if trace.Schema != "tetra.surface.browser-canvas-trace.v1" || trace.AppExitCode != 1 {
		t.Fatalf("trace = %#v, want successful browser canvas trace", trace)
	}
	rawCount, err := os.ReadFile(countPath)
	if err != nil {
		t.Fatalf("read fake browser count: %v", err)
	}
	if strings.TrimSpace(string(rawCount)) != "2" {
		t.Fatalf("fake browser invocations = %q, want retry after pending trace", strings.TrimSpace(string(rawCount)))
	}
}

func TestWriteBrowserCanvasSurfaceTraceAcceptsFlagshipAppExit(t *testing.T) {
	pixels := []byte{0, 1, 2, 3}
	tracePath := filepath.Join(t.TempDir(), "surface-runner-trace.json")
	frames, err := writeBrowserCanvasSurfaceTrace(tracePath, "surface-block-system.wasm", browserCanvasTrace{
		Schema:      "tetra.surface.browser-canvas-trace.v1",
		Canvas:      browserCanvasTraceCanvas{Opened: true, Width: 1, Height: 1, Readback: true},
		AppExitCode: 5,
		Frames: []browserCanvasTraceFrame{
			{
				Order:           1,
				Width:           1,
				Height:          1,
				Stride:          4,
				PixelsLen:       len(pixels),
				SourcePixelsB64: base64.StdEncoding.EncodeToString(pixels),
				CanvasPixelsB64: base64.StdEncoding.EncodeToString(pixels),
			},
		},
	}, 5)
	if err != nil {
		t.Fatalf("writeBrowserCanvasSurfaceTrace failed: %v", err)
	}
	if len(frames) != 1 || frames[0].Width != 1 || frames[0].Height != 1 {
		t.Fatalf("frames = %#v, want one 1x1 frame", frames)
	}
	raw, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}
	if !strings.Contains(string(raw), `"app_exit_code": 5`) {
		t.Fatalf("trace missing flagship app exit:\n%s", raw)
	}
}

func TestBrowserCanvasRunnerDataURLInlinesHostAndWASM(t *testing.T) {
	url, err := browserCanvasRunnerDataURL("export async function runSurfaceBrowserCanvas() { return {schema:'ok'}; }\n", []byte{0, 97, 115, 109, 1, 0, 0, 0}, "counter")
	if err != nil {
		t.Fatalf("browserCanvasRunnerDataURL: %v", err)
	}
	const prefix = "data:text/html;base64,"
	if !strings.HasPrefix(url, prefix) {
		gotPrefix := url
		if len(gotPrefix) > len(prefix) {
			gotPrefix = gotPrefix[:len(prefix)]
		}
		t.Fatalf("runner URL prefix = %q, want %q", gotPrefix, prefix)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(url, prefix))
	if err != nil {
		t.Fatalf("decode runner data URL: %v", err)
	}
	html := string(raw)
	for _, want := range []string{
		"async function runSurfaceBrowserCanvas()",
		"data:application/wasm;base64,",
		`scenario: "counter"`,
		`id="surface-trace"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("runner HTML missing %q:\n%s", want, html)
		}
	}
	for _, forbidden := range []string{"127.0.0.1", "localhost", "export async function runSurfaceBrowserCanvas"} {
		if strings.Contains(html, forbidden) {
			t.Fatalf("runner HTML must not contain %q:\n%s", forbidden, html)
		}
	}
}

func TestBrowserCanvasRunnerFileURLAvoidsLocalhostAndCleansUp(t *testing.T) {
	dir := t.TempDir()
	artifactDir := filepath.Join(dir, "surface-artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatalf("mkdir artifact dir: %v", err)
	}
	wasmPath := filepath.Join(artifactDir, "app.wasm")
	if err := os.WriteFile(wasmPath, []byte{0, 97, 115, 109, 1, 0, 0, 0}, 0o644); err != nil {
		t.Fatalf("write wasm: %v", err)
	}
	url, cleanup, err := browserCanvasRunnerFileURL(wasmPath, "export async function runSurfaceBrowserCanvas() { return {schema:'ok'}; }\n", "counter")
	if err != nil {
		t.Fatalf("browserCanvasRunnerFileURL: %v", err)
	}
	if cleanup == nil {
		t.Fatalf("cleanup is nil")
	}
	if !strings.HasPrefix(url, "file://") {
		t.Fatalf("runner URL = %q, want file:// URL", url)
	}
	if strings.Contains(url, "127.0.0.1") || strings.Contains(url, "localhost") {
		t.Fatalf("runner URL must not use localhost: %q", url)
	}
	runnerPath := strings.TrimPrefix(url, "file://")
	raw, err := os.ReadFile(runnerPath)
	if err != nil {
		t.Fatalf("read runner file: %v", err)
	}
	html := string(raw)
	for _, want := range []string{"async function runSurfaceBrowserCanvas()", "file://", `scenario: "counter"`} {
		if !strings.Contains(html, want) {
			t.Fatalf("runner HTML missing %q:\n%s", want, html)
		}
	}
	cleanup()
	if _, err := os.Stat(runnerPath); !os.IsNotExist(err) {
		t.Fatalf("runner cleanup stat err = %v, want removed", err)
	}
}

func TestCollectLinuxX64PresentedFrameEvidenceReadsAppPresentedRGBA(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	process, frame, err := collectLinuxX64PresentedFrameEvidence(t.TempDir())
	if err != nil {
		t.Fatalf("collectLinuxX64PresentedFrameEvidence: %v", err)
	}
	if process.Name != "surface linux-x64 presented frame probe" || process.Kind != "app" || !process.Ran || !process.Pass {
		t.Fatalf("process = %#v, want passing app probe process", process)
	}
	if frame.Order != 3 || frame.Width != 2 || frame.Height != 2 || frame.Stride != 8 || !frame.Presented {
		t.Fatalf("frame = %#v, want order-3 2x2 app-presented frame evidence", frame)
	}
	if want := checksumRGBA(surfacePresentedFrameProbePixels()); frame.Checksum != want {
		t.Fatalf("frame checksum = %s, want app-presented RGBA checksum %s", frame.Checksum, want)
	}
}

func TestCollectLinuxX64CounterAppPresentedFrameEvidenceReadsCounterDraw(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	process, frame, err := collectLinuxX64CounterAppPresentedFrameEvidence(t.TempDir())
	if err != nil {
		t.Fatalf("collectLinuxX64CounterAppPresentedFrameEvidence: %v", err)
	}
	if process.Name != "surface linux-x64 counter app presented frame probe" || process.Kind != "app" || !process.Ran || !process.Pass {
		t.Fatalf("process = %#v, want passing counter app presented frame probe", process)
	}
	if frame.Order != 4 || frame.Width != 320 || frame.Height != 200 || frame.Stride != 1280 || !frame.Presented {
		t.Fatalf("frame = %#v, want order-4 320x200 counter app presented frame evidence", frame)
	}
	wantFrame := renderCounterFrameRGBA(1, true)
	if want := checksumRGBA(wantFrame.Pixels); frame.Checksum != want {
		t.Fatalf("frame checksum = %s, want counter app RGBA checksum %s", frame.Checksum, want)
	}
}

func TestCollectLinuxX64EventSequenceProbeEvidenceRunsHostABISequence(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	processes, err := collectLinuxX64EventSequenceProbeEvidence(t.TempDir())
	if err != nil {
		t.Fatalf("collectLinuxX64EventSequenceProbeEvidence: %v", err)
	}
	if len(processes) != 2 {
		t.Fatalf("processes = %#v, want build and app probe", processes)
	}
	app := processes[1]
	if app.Name != "surface linux-x64 event sequence probe" || app.Kind != "app" || !app.Ran || !app.Pass {
		t.Fatalf("app process = %#v, want passing event sequence app probe", app)
	}
	if app.ExitCode == nil || *app.ExitCode != 42 || app.ExpectedExitCode == nil || *app.ExpectedExitCode != 42 {
		t.Fatalf("app process exit evidence = %#v, want 42/42", app)
	}
}

func TestRejectLegacyUISidecarArtifactsAllowsCompilerOwnedWASMLoader(t *testing.T) {
	tmp := t.TempDir()
	wasmPath := filepath.Join(tmp, "surface_counter.wasm")
	loaderPath := filepath.Join(tmp, "surface_counter.mjs")
	if err := os.WriteFile(wasmPath, []byte{0, 'a', 's', 'm'}, 0o644); err != nil {
		t.Fatalf("write wasm: %v", err)
	}
	if err := os.WriteFile(loaderPath, []byte("export {};"), 0o644); err != nil {
		t.Fatalf("write compiler-owned loader: %v", err)
	}
	if err := rejectLegacyUISidecarArtifacts(tmp, sidecarScanOptions{AllowCompilerOwnedWASMLoader: true}); err != nil {
		t.Fatalf("compiler-owned wasm loader should be allowed: %v", err)
	}

	legacyPath := filepath.Join(tmp, "surface_counter.ui.web.mjs")
	if err := os.WriteFile(legacyPath, []byte("export {};"), 0o644); err != nil {
		t.Fatalf("write legacy web sidecar: %v", err)
	}
	err := rejectLegacyUISidecarArtifacts(tmp, sidecarScanOptions{AllowCompilerOwnedWASMLoader: true})
	if err == nil {
		t.Fatalf("expected legacy UI web sidecar detection to fail")
	}
	if !strings.Contains(err.Error(), "surface_counter.ui.web.mjs") {
		t.Fatalf("error = %v, want legacy sidecar path", err)
	}
}

func TestRejectLegacyUISidecarArtifactsDetectsLegacySidecars(t *testing.T) {
	tmp := t.TempDir()
	legacyPath := filepath.Join(tmp, "surface_counter.ui.html")
	if err := os.WriteFile(legacyPath, []byte("<div>legacy</div>"), 0o644); err != nil {
		t.Fatalf("write legacy sidecar: %v", err)
	}
	err := rejectLegacyUISidecarArtifacts(tmp)
	if err == nil {
		t.Fatalf("expected legacy UI sidecar detection to fail")
	}
	if !strings.Contains(err.Error(), "surface_counter.ui.html") {
		t.Fatalf("error = %v, want legacy sidecar path", err)
	}
}

func TestValidateCompilerOwnedWASMLoaderRejectsDOMAndUserJS(t *testing.T) {
	tests := []struct {
		name   string
		marker string
	}{
		{name: "DOM UI creation", marker: `document.createElement("canvas");`},
		{name: "legacy UI shell import", marker: `import("./surface_counter.ui.web.mjs");`},
		{name: "user JavaScript import", marker: `import("./user.js");`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			wasmPath := filepath.Join(tmp, "surface_counter.wasm")
			loaderPath := strings.TrimSuffix(wasmPath, filepath.Ext(wasmPath)) + ".mjs"
			if err := os.WriteFile(wasmPath, []byte{0, 'a', 's', 'm'}, 0o644); err != nil {
				t.Fatalf("write wasm: %v", err)
			}
			loader := `function createSurfaceHost(instanceRef) {
  return { __tetra_surface_present_rgba() { return 0; } };
}
const imports = { tetra_surface_host_v1: createSurfaceHost({ instance: null }) };
` + tt.marker + "\n"
			if err := os.WriteFile(loaderPath, []byte(loader), 0o644); err != nil {
				t.Fatalf("write loader: %v", err)
			}

			err := validateCompilerOwnedWASMLoader(wasmPath)
			if err == nil {
				t.Fatalf("expected compiler-owned loader marker %q to be rejected", tt.marker)
			}
			if !strings.Contains(err.Error(), "must not contain") {
				t.Fatalf("error = %v, want forbidden marker diagnostic", err)
			}
		})
	}
}

func TestNodeCommandDropsEnvProxyWarningFlag(t *testing.T) {
	env := withoutNodeEnvProxy([]string{
		"PATH=/usr/bin",
		"NODE_USE_ENV_PROXY=1",
		"NO_COLOR=1",
	})
	for _, entry := range env {
		if strings.HasPrefix(entry, "NODE_USE_ENV_PROXY=") {
			t.Fatalf("env = %#v, want NODE_USE_ENV_PROXY removed for local node smoke commands", env)
		}
	}
	if !stringSlicesEqual(env, []string{"PATH=/usr/bin", "NO_COLOR=1"}) {
		t.Fatalf("env = %#v, want unrelated entries preserved", env)
	}
}

func TestSurfaceRuntimeSmokeRejectsMissingSource(t *testing.T) {
	tmp := t.TempDir()
	reportPath := filepath.Join(tmp, "surface-headless.json")
	missingSource := filepath.Join(tmp, "missing_surface_app.tetra")
	cmd := exec.Command("go", "run", ".", "--mode", "headless", "--report", reportPath, "--source", missingSource)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected missing Surface source to fail, output:\n%s", out)
	}
	if got := string(out); !strings.Contains(got, "build Surface source") && !strings.Contains(got, "missing_surface_app.tetra") {
		t.Fatalf("output = %s, want missing source/build diagnostic", got)
	}
}

func caseNamesContain(cases []surface.CaseReport, want string) bool {
	for _, c := range cases {
		if strings.Contains(c.Name, want) {
			return true
		}
	}
	return false
}

func eventKindsContain(events []surface.EventReport, want string) bool {
	for _, event := range events {
		if event.Kind == want {
			return true
		}
	}
	return false
}

func artifactByKind(artifacts []surface.ArtifactReport, kind string) *surface.ArtifactReport {
	for i := range artifacts {
		if artifacts[i].Kind == kind {
			return &artifacts[i]
		}
	}
	return nil
}

func componentAbilitiesContainAll(got []string, want []string) bool {
	seen := map[string]bool{}
	for _, ability := range got {
		seen[ability] = true
	}
	for _, ability := range want {
		if !seen[ability] {
			return false
		}
	}
	return true
}

func intSlicesEqual(a []int, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func stringSlicesEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
