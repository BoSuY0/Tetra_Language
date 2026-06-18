package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"tetra_language/tools/validators/surface"
)

// ---- browser_probe_test.go ----

func TestCollectWASM32WebBrowserCanvasProcessEvidenceRecordsBrowserTrace(t *testing.T) {
	if _, err := discoverBrowserRunner(); err != nil {
		t.Skipf(
			"browser runner is required for wasm32-web browser canvas Surface evidence: %v",
			err,
		)
	}
	source, err := resolveSurfaceSourcePath(
		"examples/surface/runtime/surface_browser_counter.tetra",
	)
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
	if after.Order != 5 || after.Width != 400 || after.Height != 240 || after.Stride != 1600 ||
		!after.Presented {
		t.Fatalf("last browser frame = %#v, want order-5 400x240 presented frame", after)
	}
	if want := checksumRGBA(wantFrame.Pixels); after.Checksum != want {
		t.Fatalf(
			"last browser frame checksum = %s, want actual browser CounterApp checksum %s",
			after.Checksum,
			want,
		)
	}
	trace := artifactByKind(evidence.Artifacts, "runner-trace")
	if trace == nil {
		t.Fatalf("artifacts = %#v, want browser canvas runner trace", evidence.Artifacts)
	}
	raw, err := os.ReadFile(trace.Path)
	if err != nil {
		t.Fatalf("read browser runner trace: %v", err)
	}
	for _, want := range []string{
		`"schema": "tetra.surface.browser-canvas-trace.v1"`,
		`"native_type": "pointerup"`,
		`"native_type": "keydown"`,
		`"native_type": "resize"`,
		`"native_type": "beforeinput"`,
		`"source_checksum"`,
		`"canvas_checksum"`,
	} {
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
	if err := os.WriteFile(
		filepath.Join(hostDir, "surface_browser_canvas_host.mjs"),
		[]byte("export async function runSurfaceBrowserCanvas() {}\n"),
		0o644,
	); err != nil {
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
		t.Fatalf(
			"fake browser invocations = %q, want retry after pending trace",
			strings.TrimSpace(string(rawCount)),
		)
	}
}

func TestWriteBrowserCanvasSurfaceTraceAcceptsFlagshipAppExit(t *testing.T) {
	pixels := []byte{0, 1, 2, 3}
	tracePath := filepath.Join(t.TempDir(), "surface-runner-trace.json")
	frames, err := writeBrowserCanvasSurfaceTrace(
		tracePath,
		"surface-block-system.wasm",
		browserCanvasTrace{
			Schema: "tetra.surface.browser-canvas-trace.v1",
			Canvas: browserCanvasTraceCanvas{
				Opened:   true,
				Width:    1,
				Height:   1,
				Readback: true,
			},
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
		},
		5,
	)
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
	url, err := browserCanvasRunnerDataURL(
		"export async function runSurfaceBrowserCanvas() { return {schema:'ok'}; }\n",
		[]byte{0, 97, 115, 109, 1, 0, 0, 0},
		"counter",
	)
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
	for _, forbidden := range []string{
		"127.0.0.1",
		"localhost",
		"export async function runSurfaceBrowserCanvas",
	} {
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
	url, cleanup, err := browserCanvasRunnerFileURL(
		wasmPath,
		"export async function runSurfaceBrowserCanvas() { return {schema:'ok'}; }\n",
		"counter",
	)
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
	if process.Name != "surface linux-x64 presented frame probe" || process.Kind != "app" ||
		!process.Ran ||
		!process.Pass {
		t.Fatalf("process = %#v, want passing app probe process", process)
	}
	if frame.Order != 3 || frame.Width != 2 || frame.Height != 2 || frame.Stride != 8 ||
		!frame.Presented {
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
	if process.Name != "surface linux-x64 counter app presented frame probe" ||
		process.Kind != "app" ||
		!process.Ran ||
		!process.Pass {
		t.Fatalf("process = %#v, want passing counter app presented frame probe", process)
	}
	if frame.Order != 4 || frame.Width != 320 || frame.Height != 200 || frame.Stride != 1280 ||
		!frame.Presented {
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
	if app.Name != "surface linux-x64 event sequence probe" || app.Kind != "app" || !app.Ran ||
		!app.Pass {
		t.Fatalf("app process = %#v, want passing event sequence app probe", app)
	}
	if app.ExitCode == nil || *app.ExitCode != 42 || app.ExpectedExitCode == nil ||
		*app.ExpectedExitCode != 42 {
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
	if err := rejectLegacyUISidecarArtifacts(
		tmp,
		sidecarScanOptions{AllowCompilerOwnedWASMLoader: true},
	); err != nil {
		t.Fatalf("compiler-owned wasm loader should be allowed: %v", err)
	}

	legacyPath := filepath.Join(tmp, "surface_counter.ui.web.mjs")
	if err := os.WriteFile(legacyPath, []byte("export {};"), 0o644); err != nil {
		t.Fatalf("write legacy web sidecar: %v", err)
	}
	err := rejectLegacyUISidecarArtifacts(
		tmp,
		sidecarScanOptions{AllowCompilerOwnedWASMLoader: true},
	)
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
			t.Fatalf(
				"env = %#v, want NODE_USE_ENV_PROXY removed for local node smoke commands",
				env,
			)
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
	cmd := exec.Command(
		"go",
		"run",
		".",
		"--mode",
		"headless",
		"--report",
		reportPath,
		"--source",
		missingSource,
	)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected missing Surface source to fail, output:\n%s", out)
	}
	if got := string(out); !strings.Contains(got, "build Surface source") &&
		!strings.Contains(got, "missing_surface_app.tetra") {
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

// ---- component_release_modes_test.go ----

func TestComponentTreeModesProduceTreeEvidence(t *testing.T) {
	for _, mode := range []string{
		"headless-component-tree",
		"linux-x64-real-window-component-tree",
		"wasm32-web-browser-canvas-component-tree",
		"headless-component-tree-api",
		"linux-x64-real-window-component-tree-api",
		"wasm32-web-browser-canvas-component-tree-api",
	} {
		t.Run(mode, func(t *testing.T) {
			if err := validateSmokeMode(mode); err != nil {
				t.Fatalf("validateSmokeMode(%s) failed: %v", mode, err)
			}
			if got := defaultSurfaceSourcePath(
				smokeOptions{Mode: mode, SourcePath: "examples/surface/runtime/surface_counter.tetra"},
			); got != "examples/surface/toolkit/surface_tree_app.tetra" {
				t.Fatalf(
					"defaultSurfaceSourcePath(%s) = %q, want examples/surface/toolkit/surface_tree_app.tetra",
					mode,
					got,
				)
			}
			scenario := runComponentTreeScenario(mode)
			if scenario.ComponentTree == nil {
				t.Fatalf("component_tree missing from scenario")
			}
			if scenario.ComponentTreeAPI == nil {
				t.Fatalf("component_tree_api missing from scenario")
			}
			if scenario.ComponentTreeAPI.APILevel != "builder-layout-dispatch-v1" ||
				scenario.ComponentTreeAPI.ManualBookkeeping {
				t.Fatalf(
					"component_tree_api = %#v, want hardened helper evidence without manual bookkeeping",
					scenario.ComponentTreeAPI,
				)
			}
			if scenario.ComponentTree.NodeCount < 7 || len(scenario.ComponentTree.Nodes) < 7 {
				t.Fatalf("component_tree = %#v, want at least 7 nodes", scenario.ComponentTree)
			}
			if !intSlicesEqual(scenario.ComponentTree.FocusOrder, []int{3, 5, 6}) {
				t.Fatalf(
					"focus_order = %#v, want TextBox -> SubmitButton -> ResetButton",
					scenario.ComponentTree.FocusOrder,
				)
			}
			for _, want := range [][]int{{0, 1, 3}, {0, 1, 4, 5}, {0, 1, 4, 6}} {
				if !componentTreeDispatchPathsContain(scenario.ComponentTree.DispatchPaths, want) {
					t.Fatalf(
						"dispatch_paths = %#v, want %v",
						scenario.ComponentTree.DispatchPaths,
						want,
					)
				}
			}
			for _, want := range []string{
				"component tree node count",
				"component tree parent child links",
				"component tree pointer dispatch path",
				"component tree focus traversal",
				"component tree text routed to focused TextBox",
				"component tree button action dispatch",
				"component tree resize relayout",
				"component tree rendered frame update",
				"component tree api builder node creation",
				"component tree api parent child invariants",
				"component tree api layout helper dispatch",
				"component tree api hit test helper",
				"component tree api focus helper traversal",
				"component tree api dispatch path helper",
				"component tree api no manual bookkeeping",
			} {
				if !caseNamesContain(scenario.Cases, want) {
					t.Fatalf("cases = %#v, want %q", scenario.Cases, want)
				}
			}
			reportScenario := scenario
			reportScenario.Frames = append(reportScenario.Frames, componentTreeTestFrames(mode)...)
			if len(reportScenario.Frames) < 2 ||
				reportScenario.Frames[0].Checksum == reportScenario.Frames[len(
					reportScenario.Frames,
				)-1].Checksum {
				t.Fatalf(
					"frames = %#v, want visible framebuffer update after tree input/resize",
					reportScenario.Frames,
				)
			}
			raw, err := json.Marshal(
				buildReport(
					smokeOptions{
						Mode:       mode,
						SourcePath: "examples/surface/toolkit/surface_tree_app.tetra",
					},
					"linux-x64",
					componentTreeTestProcesses(mode),
					componentTreeTestArtifacts(mode),
					cleanArtifactScan(3),
					reportScenario,
				),
			)
			if err != nil {
				t.Fatalf("marshal component tree report: %v", err)
			}
			if err := surface.ValidateReport(raw); err != nil {
				t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
			}
		})
	}
}

func TestMinimalToolkitModesUseToolkitFormSource(t *testing.T) {
	for _, mode := range []string{
		"headless-minimal-toolkit",
		"linux-x64-real-window-minimal-toolkit",
		"wasm32-web-browser-canvas-minimal-toolkit",
	} {
		t.Run(mode, func(t *testing.T) {
			if err := validateSmokeMode(mode); err != nil {
				t.Fatalf("validateSmokeMode(%s) failed: %v", mode, err)
			}
			if got := defaultSurfaceSourcePath(
				smokeOptions{Mode: mode, SourcePath: "examples/surface/runtime/surface_counter.tetra"},
			); got != "examples/surface/toolkit/surface_toolkit_form.tetra" {
				t.Fatalf(
					"defaultSurfaceSourcePath(%s) = %q, want examples/surface/toolkit/surface_toolkit_form.tetra",
					mode,
					got,
				)
			}
		})
	}
}

func TestToolkitReuseModesUseToolkitSettingsSource(t *testing.T) {
	for _, mode := range []string{
		"headless-toolkit-reuse",
		"linux-x64-real-window-toolkit-reuse",
		"wasm32-web-browser-canvas-toolkit-reuse",
	} {
		t.Run(mode, func(t *testing.T) {
			if err := validateSmokeMode(mode); err != nil {
				t.Fatalf("validateSmokeMode(%s) failed: %v", mode, err)
			}
			if got := defaultSurfaceSourcePath(
				smokeOptions{Mode: mode, SourcePath: "examples/surface/runtime/surface_counter.tetra"},
			); got != "examples/surface/toolkit/surface_toolkit_settings.tetra" {
				t.Fatalf(
					("defaultSurfaceSourcePath(%s) = %q, want " +
						"examples/surface/toolkit/surface_toolkit_settings.tetra"),
					mode,
					got,
				)
			}
			scenario := runSurfaceScenario(mode)
			if scenario.Toolkit == nil {
				t.Fatalf("scenario.Toolkit is nil, want toolkit reuse evidence")
			}
			if scenario.Toolkit.ToolkitLevel != "toolkit-reuse-v1" {
				t.Fatalf("toolkit_level = %q, want toolkit-reuse-v1", scenario.Toolkit.ToolkitLevel)
			}
			textBoxes := 0
			buttons := 0
			for _, widget := range scenario.Toolkit.Widgets {
				switch widget.Kind {
				case "TextBox":
					textBoxes++
				case "Button":
					buttons++
				}
			}
			if textBoxes < 2 || buttons < 2 {
				t.Fatalf(
					"toolkit widgets = %#v, want at least two TextBoxes and two Buttons",
					scenario.Toolkit.Widgets,
				)
			}
			for _, want := range []string{
				"toolkit reuse second example evidence",
				"toolkit reuse multi TextBox routing",
				"toolkit reuse focused TextBox only mutates",
				"toolkit reuse StatusText updates",
				"toolkit reuse resize relayout",
			} {
				if !caseNamesContain(scenario.Cases, want) {
					t.Fatalf("cases = %#v, want %q", scenario.Cases, want)
				}
			}
		})
	}
}

func TestReleaseToolkitModesProduceProductionToolkitEvidence(t *testing.T) {
	for _, mode := range []string{
		"headless-release-toolkit",
		"linux-x64-release-toolkit",
		"wasm32-web-release-toolkit",
	} {
		t.Run(mode, func(t *testing.T) {
			if err := validateSmokeMode(mode); err != nil {
				t.Fatalf("validateSmokeMode(%s) failed: %v", mode, err)
			}
			if got := defaultSurfaceSourcePath(
				smokeOptions{Mode: mode, SourcePath: "examples/surface/runtime/surface_counter.tetra"},
			); got != "examples/surface/release/surface_release_form.tetra" {
				t.Fatalf(
					"defaultSurfaceSourcePath(%s) = %q, want examples/surface/release/surface_release_form.tetra",
					mode,
					got,
				)
			}
			scenario := runSurfaceScenario(mode)
			if scenario.Toolkit == nil {
				t.Fatalf("scenario.Toolkit is nil, want production toolkit evidence")
			}
			if scenario.Toolkit.ToolkitLevel != "production-widgets-v1" {
				t.Fatalf(
					"toolkit_level = %q, want production-widgets-v1",
					scenario.Toolkit.ToolkitLevel,
				)
			}
			if scenario.Toolkit.Experimental || !scenario.Toolkit.ProductionClaim {
				t.Fatalf(
					("toolkit release flags = experimental:%v production_claim:%v," +
						" want current production evidence"),
					scenario.Toolkit.Experimental,
					scenario.Toolkit.ProductionClaim,
				)
			}
			requiredKinds := map[string]bool{
				"Text": false, "Label": false, "StatusText": false, "Button": false,
				"TextBox": false, "Checkbox": false, "Row": false, "Column": false,
				"Panel": false, "Stack": false, "Scroll": false, "Spacer": false,
			}
			for _, widget := range scenario.Toolkit.Widgets {
				if _, ok := requiredKinds[widget.Kind]; ok {
					requiredKinds[widget.Kind] = true
				}
			}
			for kind, found := range requiredKinds {
				if !found {
					t.Fatalf(
						"toolkit widgets = %#v, missing required kind %s",
						scenario.Toolkit.Widgets,
						kind,
					)
				}
			}
			for _, want := range []string{
				"production toolkit required widget set",
				"production toolkit style module default theme",
				"production toolkit Checkbox toggle routed",
				"production toolkit Scroll offset routed",
				"production toolkit safe text storage",
			} {
				if !caseNamesContain(scenario.Cases, want) {
					t.Fatalf("cases = %#v, want %q", scenario.Cases, want)
				}
			}
		})
	}
}

func TestReleaseBrowserModeProducesBrowserCanvasReleaseEvidence(t *testing.T) {
	const mode = "wasm32-web-release-browser"
	if err := validateSmokeMode(mode); err != nil {
		t.Fatalf("validateSmokeMode(%s) failed: %v", mode, err)
	}
	if got := defaultSurfaceSourcePath(
		smokeOptions{Mode: mode, SourcePath: "examples/surface/runtime/surface_counter.tetra"},
	); got != "examples/surface/release/surface_release_form.tetra" {
		t.Fatalf(
			"defaultSurfaceSourcePath(%s) = %q, want examples/surface/release/surface_release_form.tetra",
			mode,
			got,
		)
	}
	scenario := runSurfaceScenario(mode)
	if scenario.Toolkit == nil {
		t.Fatalf("scenario.Toolkit is nil, want production toolkit evidence")
	}
	for _, want := range []string{
		"browser release Surface v1 schema",
		"browser release Chromium canvas readback",
		"browser release native pointer keyboard text resize",
		"browser release deterministic clipboard harness",
		"browser release composition trace",
		"browser release accessibility snapshot mirror",
		"browser release forbidden web sidecar rejection",
	} {
		if !caseNamesContain(scenario.Cases, want) {
			t.Fatalf("cases = %#v, want %q", scenario.Cases, want)
		}
	}
	scenario.Frames = append(scenario.Frames, releaseBrowserTestFrames()...)
	raw, err := json.Marshal(buildReport(
		smokeOptions{Mode: mode, SourcePath: "examples/surface/release/surface_release_form.tetra"},
		"linux-x64",
		releaseBrowserTestProcesses(),
		releaseBrowserTestArtifacts(),
		cleanArtifactScan(3),
		scenario,
	))
	if err != nil {
		t.Fatalf("marshal release browser report: %v", err)
	}
	var report surface.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode release browser report: %v", err)
	}
	if report.Target != "wasm32-web" {
		t.Fatalf("target = %q, want wasm32-web", report.Target)
	}
	if report.HostEvidence.Level != "wasm32-web-browser-canvas-release-v1" ||
		report.HostEvidence.Backend != "browser-canvas-rgba-accessible" ||
		!report.HostEvidence.BrowserCanvas ||
		!report.HostEvidence.BrowserInput ||
		!report.HostEvidence.BrowserClipboard ||
		report.HostEvidence.BrowserClipboardHarness != "deterministic-browser-clipboard-v1" ||
		!report.HostEvidence.BrowserComposition ||
		!report.HostEvidence.BrowserAccessibilitySnapshot ||
		!report.HostEvidence.BrowserAccessibilityMirror {
		t.Fatalf("host evidence = %#v, want strict browser release evidence", report.HostEvidence)
	}
	if report.BrowserSurface == nil ||
		report.BrowserSurface.Schema != surface.BrowserSurfaceSchemaV1 ||
		report.BrowserSurface.BrowserSurfaceLevel != "browser-canvas-release-v1" ||
		!report.BrowserSurface.DOMHostCanvasOnly ||
		!report.BrowserSurface.NegativeGuards.NoDOMAppUITree ||
		!report.BrowserSurface.NegativeGuards.NoUserJSAppLogic ||
		!report.BrowserSurface.NegativeGuards.NoNodeOnlyPromotion {
		t.Fatalf(
			"browser surface evidence = %#v, want strict browser_surface P13 evidence",
			report.BrowserSurface,
		)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed for release browser: %v\n%s", err, raw)
	}
}

func TestLinuxX64ReleaseWindowModeProducesReleaseEvidence(t *testing.T) {
	const mode = "linux-x64-release-window"
	if err := validateSmokeMode(mode); err != nil {
		t.Fatalf("validateSmokeMode(%s) failed: %v", mode, err)
	}
	if got := defaultSurfaceSourcePath(
		smokeOptions{Mode: mode, SourcePath: "examples/surface/runtime/surface_counter.tetra"},
	); got != "examples/surface/release/surface_release_form.tetra" {
		t.Fatalf(
			"defaultSurfaceSourcePath(%s) = %q, want examples/surface/release/surface_release_form.tetra",
			mode,
			got,
		)
	}
	scenario := runSurfaceScenario(mode)
	if scenario.Toolkit == nil {
		t.Fatalf("scenario.Toolkit is nil, want production toolkit evidence")
	}
	if scenario.AccessibilityTree == nil {
		t.Fatalf(
			"scenario.AccessibilityTree is nil, want linux release accessibility bridge evidence",
		)
	}
	for _, want := range []string{
		"linux release window v1 schema",
		"linux release real window presented frame",
		"linux release native pointer key text resize close",
		"linux release clipboard harness",
		"linux release composition harness",
		"linux release accessibility bridge probe",
		"linux release forbids memfd starter promotion",
	} {
		if !caseNamesContain(scenario.Cases, want) {
			t.Fatalf("cases = %#v, want %q", scenario.Cases, want)
		}
	}
	scenario.Frames = releaseWindowTestFrames()
	raw, err := json.Marshal(buildReport(
		smokeOptions{Mode: mode, SourcePath: "examples/surface/release/surface_release_form.tetra"},
		"linux-x64",
		releaseWindowTestProcesses(),
		releaseWindowTestArtifacts(),
		cleanArtifactScan(3),
		scenario,
	))
	if err != nil {
		t.Fatalf("marshal release window report: %v", err)
	}
	var report surface.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("decode release window report: %v", err)
	}
	if report.Target != "linux-x64" {
		t.Fatalf("target = %q, want linux-x64", report.Target)
	}
	if report.HostEvidence.Level != "linux-x64-release-window-v1" ||
		report.HostEvidence.Backend != "wayland-shm-rgba-release-v1" ||
		!report.HostEvidence.Framebuffer ||
		!report.HostEvidence.RealWindow ||
		!report.HostEvidence.NativeInput ||
		!report.HostEvidence.TextInput ||
		!report.HostEvidence.Clipboard ||
		!report.HostEvidence.Composition ||
		!report.HostEvidence.AccessibilityBridge {
		t.Fatalf(
			"host evidence = %#v, want strict linux release window evidence",
			report.HostEvidence,
		)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed for release window: %v\n%s", err, raw)
	}
}

func TestReleaseAccessibilityModesProducePlatformBridgeEvidence(t *testing.T) {
	for _, tc := range []struct {
		mode       string
		wantTarget string
	}{
		{mode: "headless-release-accessibility", wantTarget: "headless"},
		{mode: "linux-x64-release-accessibility", wantTarget: "linux-x64"},
		{mode: "wasm32-web-release-accessibility", wantTarget: "wasm32-web"},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			if err := validateSmokeMode(tc.mode); err != nil {
				t.Fatalf("validateSmokeMode(%s) failed: %v", tc.mode, err)
			}
			if got := defaultSurfaceSourcePath(
				smokeOptions{Mode: tc.mode, SourcePath: "examples/surface/runtime/surface_counter.tetra"},
			); got != "examples/surface/release/surface_release_accessibility.tetra" {
				t.Fatalf(
					("defaultSurfaceSourcePath(%s) = %q, want " +
						"examples/surface/release/surface_release_accessibility.tetra"),
					tc.mode,
					got,
				)
			}
			scenario := runSurfaceScenario(tc.mode)
			if scenario.AccessibilityTree == nil {
				t.Fatalf("scenario.AccessibilityTree is nil, want platform bridge evidence")
			}
			tree := scenario.AccessibilityTree
			if tree.AccessibilityLevel != "platform-bridge-v1" {
				t.Fatalf(
					"accessibility_level = %q, want platform-bridge-v1",
					tree.AccessibilityLevel,
				)
			}
			if tree.ReleaseScope != "surface-v1-linux-web" {
				t.Fatalf("release_scope = %q, want surface-v1-linux-web", tree.ReleaseScope)
			}
			if tree.Experimental || !tree.ProductionClaim {
				t.Fatalf(
					("release accessibility flags = experimental:%v " +
						"production_claim:%v, want current production evidence"),
					tree.Experimental,
					tree.ProductionClaim,
				)
			}
			for _, want := range []string{
				"accessibility platform bridge v1 schema",
				"accessibility platform export from metadata tree",
				"accessibility release honest screen reader evidence",
			} {
				if !caseNamesContain(scenario.Cases, want) {
					t.Fatalf("cases = %#v, want %q", scenario.Cases, want)
				}
			}
			processes := releaseAccessibilityTestProcesses(tc.mode)
			artifacts := releaseAccessibilityTestArtifacts(tc.mode)
			scenario.Frames = append(scenario.Frames, releaseAccessibilityTestFrames(tc.mode)...)
			raw, err := json.Marshal(
				buildReport(
					smokeOptions{
						Mode:       tc.mode,
						SourcePath: "examples/surface/release/surface_release_accessibility.tetra",
					},
					"linux-x64",
					processes,
					artifacts,
					cleanArtifactScan(len(artifacts)),
					scenario,
				),
			)
			if err != nil {
				t.Fatalf("marshal release accessibility report: %v", err)
			}
			var envelope struct {
				Target string `json:"target"`
			}
			if err := json.Unmarshal(raw, &envelope); err != nil {
				t.Fatalf("decode release accessibility report: %v", err)
			}
			if envelope.Target != tc.wantTarget {
				t.Fatalf("target = %q, want %q", envelope.Target, tc.wantTarget)
			}
			if err := surface.ValidateReport(raw); err != nil {
				t.Fatalf("ValidateReport failed for %s: %v\n%s", tc.mode, err, raw)
			}
		})
	}
}

// ---- main_test.go ----

func componentAppArtifacts(path string) []surface.ArtifactReport {
	return []surface.ArtifactReport{
		{
			Kind:   "component-app",
			Path:   path,
			SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Size:   49172,
		},
	}
}

func headlessSurfaceArtifacts(path string) []surface.ArtifactReport {
	artifacts := componentAppArtifacts(path)
	return append(artifacts, surface.ArtifactReport{
		Kind:   "runner-trace",
		Path:   "/tmp/surface-artifacts/surface-runner-trace.json",
		SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		Size:   409,
	})
}

func wasmSurfaceArtifacts() []surface.ArtifactReport {
	return []surface.ArtifactReport{
		{
			Kind:   "component-app",
			Path:   "/tmp/surface-artifacts/surface-counter.wasm",
			SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Size:   7502,
		},
		{
			Kind:   "compiler-owned-loader",
			Path:   "/tmp/surface-artifacts/surface-counter.mjs",
			SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			Size:   4931,
		},
		{
			Kind:   "runner-trace",
			Path:   "/tmp/surface-artifacts/surface-runner-trace.json",
			SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			Size:   413,
		},
	}
}

func wasmBrowserCanvasSurfaceArtifacts() []surface.ArtifactReport {
	return []surface.ArtifactReport{
		{
			Kind:   "component-app",
			Path:   "/tmp/surface-artifacts/surface-browser-counter.wasm",
			SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Size:   8604,
		},
		{
			Kind:   "compiler-owned-loader",
			Path:   "/tmp/surface-artifacts/surface-browser-counter.mjs",
			SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			Size:   4939,
		},
		{
			Kind:   "runner-trace",
			Path:   "/tmp/surface-artifacts/surface-runner-trace.json",
			SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			Size:   1184,
		},
	}
}

func cleanArtifactScan(filesChecked int) surface.ArtifactScanReport {
	return surface.ArtifactScanReport{
		Root:           "/tmp/surface-artifacts",
		FilesChecked:   filesChecked,
		ForbiddenPaths: []string{},
		Pass:           true,
	}
}

func TestSurfaceComponentAppExpectedExitForGeneratedTemplateSource(t *testing.T) {
	source := ("reports/surface-electron-react-beauty-production/P21/templat" +
		"e-smoke/templates/command-palette/src/main.tetra")
	if got := surfaceComponentAppExpectedExitForSource("headless-block-system", source); got != 0 {
		t.Fatalf("surfaceComponentAppExpectedExitForSource(generated template) = %d, want 0", got)
	}
	if got := surfaceComponentAppExpectedExitForSource("headless-morph", (("examples/surface/" +
		"reference_core/surface_reference_command_pa") +
		"lette.tetra")); got != 0 {
		t.Fatalf("surfaceComponentAppExpectedExitForSource(reference app) = %d, want 0", got)
	}
	if got := surfaceComponentAppExpectedExitForSource(
		"headless-morph",
		filepath.Join(
			"/tmp",
			"repo",
			"examples",
			"surface",
			"reference_core",
			"surface_reference_settings.tetra",
		),
	); got != 0 {
		t.Fatalf(
			"surfaceComponentAppExpectedExitForSource(absolute reference app) = %d, want 0",
			got,
		)
	}
	if got := surfaceComponentAppExpectedExitForSource("headless-block-system", (("examples/surface/" +
		"migration/surface_migration_tetra_control_c") +
		"enter.tetra")); got != 5 {
		t.Fatalf(
			"surfaceComponentAppExpectedExitForSource(flagship Control Center) = %d, want 5",
			got,
		)
	}
	if got := surfaceComponentAppExpectedExitForSource(
		"headless-block-system",
		filepath.Join(
			"/tmp",
			"repo",
			"examples",
			"surface",
			"migration",
			"surface_migration_tetra_control_center.tetra",
		),
	); got != 5 {
		t.Fatalf(
			"surfaceComponentAppExpectedExitForSource(absolute flagship Control Center) = %d, want 5",
			got,
		)
	}
	if got := surfaceComponentAppExpectedExitForSource(
		"headless-block-system",
		"examples/surface/block_core/surface_block_system.tetra",
	); got != 1 {
		t.Fatalf(
			"surfaceComponentAppExpectedExitForSource(canonical block system) = %d, want 1",
			got,
		)
	}
}

func TestSurfaceTemplateScenarioRetargetsOnlyGeneratedSources(t *testing.T) {
	generated := smokeOptions{
		Mode: "headless-block-system",
		SourcePath: ("reports/surface-electron-react-beauty-production/P21/templat" +
			"e-smoke/templates/command-palette/src/main.tetra"),
	}
	if !shouldRetargetSurfaceTemplateScenario(generated) {
		t.Fatalf("generated template source should retarget block-system scenario")
	}
	canonical := smokeOptions{
		Mode:       "headless-block-system",
		SourcePath: "examples/surface/block_core/surface_block_system.tetra",
	}
	if shouldRetargetSurfaceTemplateScenario(canonical) {
		t.Fatalf("canonical block-system source must not retarget to generated template module")
	}
	counter := smokeOptions{
		Mode: "headless",
		SourcePath: ("reports/surface-electron-react-beauty-production/P21/templat" +
			"e-smoke/templates/command-palette/src/main.tetra"),
	}
	if shouldRetargetSurfaceTemplateScenario(counter) {
		t.Fatalf("non Block/Morph mode must not retarget generated template scenario")
	}
}

func TestSurfaceTemplateSmokeUsesCanonicalRuntimeSources(t *testing.T) {
	root, err := repoRootForCommands()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	raw, err := os.ReadFile(
		filepath.Join(root, "scripts", "release", "surface", "surface-template-smoke.sh"),
	)
	if err != nil {
		t.Fatalf("read surface-template-smoke.sh: %v", err)
	}
	script := string(raw)
	for _, want := range []string{
		("go run ./tools/cmd/surface-runtime-smoke --mode " +
			"headless-block-system --source " +
			"examples/surface/block_core/surface_block_system.tetra " +
			"--report \"$block_report\""),
		("go run ./tools/cmd/surface-runtime-smoke --mode " +
			"headless-morph --source " +
			"examples/surface/morph_core/surface_morph_command_palette.te" +
			"tra --report \"$morph_report\""),
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("surface-template-smoke.sh missing canonical runtime source command %q", want)
		}
	}
	for _, forbidden := range []string{
		`--mode headless-block-system --source "$first_source"`,
		`--mode headless-morph --source "$first_source"`,
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf(
				("surface-template-smoke.sh must not run template source " +
					"through synthetic runtime evidence: found %q"),
				forbidden,
			)
		}
	}
}

func TestMorphRenderedFlagshipSourcePresentsSurfaceFrames(t *testing.T) {
	root, err := repoRootForCommands()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	raw, err := os.ReadFile(
		filepath.Join(
			root,
			"examples",
			"surface",
			"morph_flagship",
			"surface_morph_rendered_studio_shell.tetra",
		),
	)
	if err != nil {
		t.Fatalf("read Morph rendered flagship source: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`surface.open("Tetra Studio Shell", 320, 200)`,
		"surface.begin_frame(win)",
		"morph.render_studio_shell_frame(false, before_frame)",
		"surface.present(before_frame)",
		"surface.poll_event(win)",
		"surface.poll_event_text_into(win, text_bytes)",
		"win.width = resize_event.width",
		"morph.render_studio_shell_frame(active, after_frame)",
		"surface.present(after_frame)",
		"surface.close(win)",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf(
				"Morph rendered flagship source missing target runtime evidence marker %q",
				want,
			)
		}
	}
	for _, forbidden := range []string{
		"import lib.core.draw",
		"draw_flagship_shell_scene",
		"draw.DrawContext",
		"draw.rect",
		"draw.clear",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf(
				"Morph rendered flagship source must stay Morph-authored; found forbidden marker %q",
				forbidden,
			)
		}
	}
}

func TestMorphGuestDashboardSourcePresentsSurfaceFrames(t *testing.T) {
	root, err := repoRootForCommands()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	raw, err := os.ReadFile(
		filepath.Join(
			root,
			"examples",
			"surface",
			"morph_flagship",
			"surface_morph_guest_dashboard.tetra",
		),
	)
	if err != nil {
		t.Fatalf("read Morph guest dashboard source: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`surface.open("Tetra Surface Guest Dashboard", 1760, 700)`,
		"surface.begin_frame(win)",
		"morph.render_guest_dashboard_frame(false, before_frame)",
		"surface.present(before_frame)",
		"morph.render_guest_dashboard_frame(true, after_frame)",
		"surface.present(after_frame)",
		"surface.close(win)",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Morph guest dashboard source missing target runtime evidence marker %q", want)
		}
	}
	for _, forbidden := range []string{
		"import lib.core.draw",
		"draw.DrawContext",
		"draw.rect",
		"draw.clear",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf(
				"Morph guest dashboard source must stay Morph-authored; found forbidden marker %q",
				forbidden,
			)
		}
	}
}

func TestMorphBrowserCanvasScenarioNameUsesGuestDashboardSource(t *testing.T) {
	if got := morphBrowserCanvasScenarioName((("examples/surface/morph_flagship/surface_" +
		"morph_guest_dashboar") +
		"d.tetra")); got != "guest-dashboard" {
		t.Fatalf("morphBrowserCanvasScenarioName(guest dashboard) = %q, want guest-dashboard", got)
	}
	if got := morphBrowserCanvasScenarioName((("/tmp/repo/examples/surface/morph_flagship/" +
		"surface_morph_gues") +
		"t_dashboard.tetra")); got != "guest-dashboard" {
		t.Fatalf(
			"morphBrowserCanvasScenarioName(abs guest dashboard) = %q, want guest-dashboard",
			got,
		)
	}
	if got := morphBrowserCanvasScenarioName((("examples/surface/morph_flagship/surface_" +
		"morph_rendered_studi") +
		"o_shell.tetra")); got != "studio-shell" {
		t.Fatalf("morphBrowserCanvasScenarioName(studio shell) = %q, want studio-shell", got)
	}
}

func TestRealWindowProbeSupportsHoldUntilClose(t *testing.T) {
	root, err := repoRootForCommands()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	rawMain, err := os.ReadFile(
		filepath.Join(root, "tools", "cmd", "surface-runtime-smoke", "surface_smoke_core.go"),
	)
	if err != nil {
		t.Fatalf("read surface_smoke_core.go: %v", err)
	}
	rawWayland, err := os.ReadFile(
		filepath.Join(root, "tools", "cmd", "surface-runtime-smoke", "wayland_linux.go"),
	)
	if err != nil {
		t.Fatalf("read wayland_linux.go: %v", err)
	}
	rawLinuxProbes, err := os.ReadFile(
		filepath.Join(root, "tools", "cmd", "surface-runtime-smoke", "surface_smoke_render.go"),
	)
	if err != nil {
		t.Fatalf("read surface_smoke_render.go: %v", err)
	}
	for _, want := range []string{
		"probe-hold-until-close",
		"&opt.ProbeHoldUntilClose",
	} {
		if !strings.Contains(string(rawMain), want) {
			t.Fatalf("real-window probe CLI missing hold-until-close marker %q", want)
		}
	}
	for _, want := range []string{
		"opt.ProbeHoldUntilClose",
		"presentRealWindowSurface(opt.ProbeTitle, frame, 350*time.Millisecond, opt.ProbeHoldUntilClose)",
	} {
		if !strings.Contains(string(rawLinuxProbes), want) {
			t.Fatalf("real-window probe runner missing hold-until-close marker %q", want)
		}
	}
	for _, want := range []string{
		"xdgToplevelID uint32",
		"holdUntilClose bool",
		"for holdUntilClose || time.Now().Before(deadline)",
		"c.closed",
	} {
		if !strings.Contains(string(rawWayland), want) {
			t.Fatalf("Wayland presenter missing close-loop marker %q", want)
		}
	}
}

func TestHeadlessCounterScenarioProducesValidSurfaceRuntimeEvidence(t *testing.T) {
	scenario := runHeadlessCounterScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless",
		SourcePath: "examples/surface/runtime/surface_counter.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/runtime/surface_counter.tetra -o " +
				"/tmp/surface-artifacts/surface-counter"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-counter",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(1),
			ExpectedExitCode: intPtr(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, headlessSurfaceArtifacts(
		"/tmp/surface-artifacts/surface-counter",
	), cleanArtifactScan(
		2,
	), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.HostEvidence.Level != "deterministic-headless" ||
		report.HostEvidence.Backend != "software-rgba" ||
		!report.HostEvidence.Framebuffer ||
		report.HostEvidence.RealWindow ||
		report.HostEvidence.NativeInput {
		t.Fatalf(
			("host evidence = %#v, want deterministic headless software " +
				"framebuffer without real-window/native-input claim"),
			report.HostEvidence,
		)
	}
	if !strings.Contains(string(raw), `"key":0`) ||
		!strings.Contains(string(raw), `"timestamp_ms":0`) {
		t.Fatalf("report JSON %s, want explicit zero-valued key and timestamp_ms event fields", raw)
	}
	if len(report.Artifacts) != 2 || report.Artifacts[0].Kind != "component-app" ||
		report.Artifacts[0].Path != "/tmp/surface-artifacts/surface-counter" ||
		report.Artifacts[0].SHA256 == "" ||
		report.Artifacts[1].Kind != "runner-trace" {
		t.Fatalf(
			"artifacts = %#v, want component app and runner trace hash evidence",
			report.Artifacts,
		)
	}
	if len(scenario.Frames) != 2 || scenario.Frames[0].Checksum == "" ||
		scenario.Frames[1].Checksum == "" {
		t.Fatalf("scenario pre/post frame checksums missing: %#v", scenario.Frames)
	}
	if scenario.Frames[0].Checksum == scenario.Frames[1].Checksum {
		t.Fatalf(
			"scenario pre/post frame checksums match, want redraw evidence across state change: %#v",
			scenario.Frames,
		)
	}
	if len(scenario.StateTransitions) != 2 || scenario.StateTransitions[0].Before != "0" ||
		scenario.StateTransitions[0].After != "1" ||
		scenario.StateTransitions[1].Field != "text_count" {
		t.Fatalf(
			"state transitions = %#v, want count 0->1 and text_count 0->1",
			scenario.StateTransitions,
		)
	}
	if len(scenario.Components) != 2 ||
		!componentAbilitiesContainAll(
			scenario.Components[0].Abilities,
			[]string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
		) ||
		!componentAbilitiesContainAll(
			scenario.Components[1].Abilities,
			[]string{"measure", "layout", "draw", "event", "focus", "text", "accessibility"},
		) {
		t.Fatalf(
			("component abilities = %#v, want ordinary struct " +
				"measure/layout/draw/event/focus/text/accessibility evidence"),
			scenario.Components,
		)
	}
	if scenario.Components[1].ID != "CounterButton" {
		t.Fatalf(
			"component hierarchy = %#v, want CounterButton child component evidence",
			scenario.Components,
		)
	}
	if scenario.Components[0].Bounds.W != 320 || scenario.Components[0].Bounds.H != 200 ||
		scenario.Components[1].Bounds.X != 32 ||
		scenario.Components[1].Bounds.Y != 80 ||
		scenario.Components[1].Bounds.W != 160 ||
		scenario.Components[1].Bounds.H != 48 {
		t.Fatalf(
			"component bounds = %#v, want measured/layout child bounds evidence",
			scenario.Components,
		)
	}
	if len(scenario.Events) < 2 || scenario.Events[1].TargetComponent != "CounterButton" ||
		scenario.Events[1].Width != 320 ||
		scenario.Events[1].Height != 200 ||
		!intSlicesEqual(scenario.Events[1].BufferSlots, []int{5, 48, 96, 1, 0, 320, 200, 0, 0}) {
		t.Fatalf(
			"events = %#v, want full host event buffer dispatched to child CounterButton",
			scenario.Events,
		)
	}
	if len(scenario.Events) < 2 ||
		!stringSlicesEqual(
			scenario.Events[1].DispatchPath,
			[]string{"CounterApp", "CounterButton"},
		) {
		t.Fatalf("events = %#v, want root-to-child dispatch path evidence", scenario.Events)
	}
	if len(scenario.Events) < 3 || scenario.Events[2].Kind != "text_input" ||
		scenario.Events[2].TargetComponent != "CounterButton" ||
		scenario.Events[2].TextLen != 2 ||
		scenario.Events[2].TextBytesHex != "4f4b" {
		t.Fatalf(
			"events = %#v, want host text payload dispatched to child CounterButton",
			scenario.Events,
		)
	}
	if !intSlicesEqual(scenario.Events[2].BufferSlots, []int{8, 0, 0, 0, 0, 320, 200, 1, 2}) {
		t.Fatalf(
			"events = %#v, want full host text event buffer dispatched to child CounterButton",
			scenario.Events,
		)
	}
	if !caseNamesContain(scenario.Cases, "component hierarchy dispatch") {
		t.Fatalf(
			"scenario cases = %#v, want static component hierarchy dispatch evidence",
			scenario.Cases,
		)
	}
	if !caseNamesContain(scenario.Cases, "host event buffer poll_event") {
		t.Fatalf("scenario cases = %#v, want host event buffer evidence", scenario.Cases)
	}
	if !caseNamesContain(scenario.Cases, "component text input scalar dispatch") {
		t.Fatalf(
			"scenario cases = %#v, want static component text input scalar evidence",
			scenario.Cases,
		)
	}
	if !caseNamesContain(scenario.Cases, "host text payload buffer") {
		t.Fatalf("scenario cases = %#v, want host text payload buffer evidence", scenario.Cases)
	}
	if !caseNamesContain(scenario.Cases, "component focus dispatch") {
		t.Fatalf(
			"scenario cases = %#v, want static component focus dispatch evidence",
			scenario.Cases,
		)
	}
	if !caseNamesContain(scenario.Cases, "component accessibility metadata") {
		t.Fatalf(
			"scenario cases = %#v, want static component accessibility metadata evidence",
			scenario.Cases,
		)
	}
}

func TestCollectHeadlessProcessEvidenceRecordsRunnerTrace(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	reportPath := filepath.Join(t.TempDir(), "surface-headless.json")
	source, err := resolveSurfaceSourcePath("examples/surface/runtime/surface_counter.tetra")
	if err != nil {
		t.Fatalf("resolve Surface source: %v", err)
	}
	evidence, err := collectSurfaceProcessEvidence(
		smokeOptions{Mode: "headless", ReportPath: reportPath, SourcePath: source},
	)
	if err != nil {
		t.Fatalf("collectSurfaceProcessEvidence(headless): %v", err)
	}
	traceArtifact := artifactByKind(evidence.Artifacts, "runner-trace")
	if traceArtifact == nil {
		t.Fatalf("artifacts = %#v, want headless runner trace artifact", evidence.Artifacts)
	}
	if evidence.ArtifactScan.FilesChecked < 2 {
		t.Fatalf(
			"artifact scan = %#v, want component app and runner trace checked",
			evidence.ArtifactScan,
		)
	}
	traceFrames, err := readHeadlessSurfaceTrace(traceArtifact.Path)
	if err != nil {
		t.Fatalf("readHeadlessSurfaceTrace: %v", err)
	}
	if len(traceFrames) != 2 {
		t.Fatalf("trace frames = %#v, want deterministic pre/post frames", traceFrames)
	}
	beforeFrame := renderCounterFrameRGBA(0, true)
	afterFrame := renderCounterFrameRGBA(1, true)
	if traceFrames[0].Checksum != checksumRGBA(beforeFrame.Pixels) ||
		traceFrames[1].Checksum != checksumRGBA(afterFrame.Pixels) {
		t.Fatalf("trace frames = %#v, want actual headless runner checksums", traceFrames)
	}
}

func TestCollectMorphGuestDashboardProcessEvidenceUsesGuestArtifactName(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	const source = "examples/surface/morph_flagship/surface_morph_guest_dashboard.tetra"
	reportPath := filepath.Join(t.TempDir(), "surface-headless-morph-guest-dashboard.json")
	evidence, err := collectSurfaceProcessEvidence(
		smokeOptions{Mode: "headless-morph", ReportPath: reportPath, SourcePath: source},
	)
	if err != nil {
		t.Fatalf("collectSurfaceProcessEvidence(headless-morph guest dashboard): %v", err)
	}
	artifact := artifactByKind(evidence.Artifacts, "component-app")
	if artifact == nil {
		t.Fatalf("artifacts = %#v, want guest dashboard component-app artifact", evidence.Artifacts)
	}
	if !strings.Contains(filepath.ToSlash(artifact.Path), "surface-morph-guest-dashboard") {
		t.Fatalf(
			"component artifact path = %q, want guest-dashboard-specific artifact name",
			artifact.Path,
		)
	}
	if strings.Contains(filepath.ToSlash(artifact.Path), "surface-morph-command-palette") {
		t.Fatalf(
			"component artifact path = %q, must not reuse command-palette artifact name",
			artifact.Path,
		)
	}
}

func TestSurfaceRuntimeSmokeMainGuestDashboardTraceMatchesReportFrames(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	root, err := repoRootForCommands()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	testBinary, err := os.Executable()
	if err != nil {
		t.Fatalf("test binary: %v", err)
	}
	reportPath := filepath.Join(t.TempDir(), "surface-headless-morph-guest-dashboard.json")
	cmd := exec.Command(testBinary,
		"-test.run=^TestSurfaceRuntimeSmokeMainHelperProcess$",
		"--",
		"--mode", "headless-morph",
		"--source", "examples/surface/morph_flagship/surface_morph_guest_dashboard.tetra",
		"--report", reportPath,
	)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "SURFACE_RUNTIME_SMOKE_MAIN_HELPER=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("surface-runtime-smoke main failed: %v\n%s", err, out)
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report surface.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("unmarshal report: %v\n%s", err, raw)
	}
	traceArtifact := artifactByKind(report.Artifacts, "runner-trace")
	if traceArtifact == nil {
		t.Fatalf("artifacts = %#v, want runner trace artifact", report.Artifacts)
	}
	traceFrames, err := readHeadlessSurfaceTrace(traceArtifact.Path)
	if err != nil {
		t.Fatalf("readHeadlessSurfaceTrace(%s): %v", traceArtifact.Path, err)
	}
	if !sameFrameEvidence(traceFrames, report.Frames) {
		t.Fatalf("runner trace frames = %#v, want report frames %#v", traceFrames, report.Frames)
	}
}

func TestBlockSystemRuntimeSmokeReportArtifactScanIncludesFrameArtifacts(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}
	root, err := repoRootForCommands()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	testBinary, err := os.Executable()
	if err != nil {
		t.Fatalf("test binary: %v", err)
	}
	reportPath := filepath.Join(t.TempDir(), "surface-headless-block-system.json")
	cmd := exec.Command(testBinary,
		"-test.run=^TestSurfaceRuntimeSmokeMainHelperProcess$",
		"--",
		"--mode", "headless-block-system",
		"--source", "examples/surface/block_core/surface_block_system.tetra",
		"--report", reportPath,
	)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "SURFACE_RUNTIME_SMOKE_MAIN_HELPER=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("surface-runtime-smoke main failed: %v\n%s", err, out)
	}
	raw, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report surface.Report
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("unmarshal report: %v\n%s", err, raw)
	}
	actualFiles, err := countRegularFilesUnder(report.ArtifactScan.Root)
	if err != nil {
		t.Fatalf("count files under artifact_scan.root: %v", err)
	}
	if report.ArtifactScan.FilesChecked != actualFiles {
		t.Fatalf(
			"artifact_scan.files_checked = %d, actual files under %s = %d",
			report.ArtifactScan.FilesChecked,
			report.ArtifactScan.Root,
			actualFiles,
		)
	}
	if report.ArtifactScan.FilesChecked < 5 {
		t.Fatalf(
			"artifact_scan.files_checked = %d, want component app, runner trace, and frame artifacts",
			report.ArtifactScan.FilesChecked,
		)
	}
}

func TestSurfaceRuntimeSmokeMainHelperProcess(t *testing.T) {
	if os.Getenv("SURFACE_RUNTIME_SMOKE_MAIN_HELPER") != "1" {
		return
	}
	args := []string{"surface-runtime-smoke"}
	for i, arg := range os.Args {
		if arg == "--" {
			args = append(args, os.Args[i+1:]...)
			break
		}
	}
	if len(args) == 1 {
		t.Fatal("missing surface-runtime-smoke helper arguments")
	}
	os.Args = args
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	main()
}

func countRegularFilesUnder(root string) (int, error) {
	count := 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		count++
		return nil
	})
	return count, err
}

func TestHeadlessCounterScenarioFrameChecksumIsDeterministic(t *testing.T) {
	first := runHeadlessCounterScenario()
	second := runHeadlessCounterScenario()
	if len(first.Frames) != len(second.Frames) {
		t.Fatalf("frame count changed: %d != %d", len(first.Frames), len(second.Frames))
	}
	for i := range first.Frames {
		if first.Frames[i].Checksum != second.Frames[i].Checksum {
			t.Fatalf(
				"checksum %d changed: %s != %s",
				i,
				first.Frames[i].Checksum,
				second.Frames[i].Checksum,
			)
		}
	}
}

func TestHeadlessCounterScenarioFrameChecksumComesFromRGBAFramebuffer(t *testing.T) {
	beforeFrame := renderCounterFrameRGBA(0, true)
	afterFrame := renderCounterFrameRGBA(1, true)
	scenario := runHeadlessCounterScenario()
	if len(scenario.Frames) != 2 {
		t.Fatalf("frames = %#v, want pre-event and post-event frames", scenario.Frames)
	}
	for i, frame := range []rgbaFrame{beforeFrame, afterFrame} {
		reported := scenario.Frames[i]
		if reported.Width != frame.Width || reported.Height != frame.Height ||
			reported.Stride != frame.Stride {
			t.Fatalf(
				"frame %d dimensions = %dx%d stride %d, want %dx%d stride %d",
				i+1,
				reported.Width,
				reported.Height,
				reported.Stride,
				frame.Width,
				frame.Height,
				frame.Stride,
			)
		}
		if len(frame.Pixels) != frame.Stride*frame.Height {
			t.Fatalf("pixel buffer len = %d, want %d", len(frame.Pixels), frame.Stride*frame.Height)
		}
		want := checksumRGBA(frame.Pixels)
		if reported.Checksum != want {
			t.Fatalf(
				"reported checksum %d = %s, want RGBA framebuffer checksum %s",
				i+1,
				reported.Checksum,
				want,
			)
		}
	}
	if scenario.Frames[0].Checksum == scenario.Frames[1].Checksum {
		t.Fatalf("pre/post frame checksums match: %#v", scenario.Frames)
	}
}

func TestBlockPaintScenarioProducesVisualPaintEvidence(t *testing.T) {
	if err := validateSmokeMode("headless-block-paint"); err != nil {
		t.Fatalf("validateSmokeMode(headless-block-paint) failed: %v", err)
	}
	scenario := runBlockPaintScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless-block-paint",
		SourcePath: "examples/surface/block_render/surface_block_paint_layers.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/block_render/surface_block_paint_layers.tet" +
				"ra -o /tmp/surface-artifacts/surface-block-paint"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-block-paint",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(0),
			ExpectedExitCode: intPtr(0),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-block-paint",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, headlessSurfaceArtifacts(
		"/tmp/surface-artifacts/surface-block-paint",
	), cleanArtifactScan(
		2,
	), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if len(report.PaintLayers) < 5 || len(report.PaintCommands) < 5 {
		t.Fatalf(
			"paint evidence = layers %#v commands %#v, want layered paint commands",
			report.PaintLayers,
			report.PaintCommands,
		)
	}
	if !caseNamesContain(scenario.Cases, "block paint deterministic command order") ||
		!caseNamesContain(scenario.Cases, "block paint unsupported blur rejected") {
		t.Fatalf(
			"scenario cases = %#v, want paint command order and unsupported blur evidence",
			scenario.Cases,
		)
	}
	if len(scenario.Frames) < 2 || scenario.Frames[0].Checksum == scenario.Frames[1].Checksum {
		t.Fatalf(
			"paint scenario frames = %#v, want visual checksum change across hover/pressed paint state",
			scenario.Frames,
		)
	}
}

func TestBlockTextScenarioProducesTextMeasurementEvidence(t *testing.T) {
	scenario := runBlockTextScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless-block-text",
		SourcePath: "examples/surface/block_render/surface_block_text.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/block_render/surface_block_text.tetra -o " +
				"/tmp/surface-artifacts/surface-block-text"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-block-text",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(1),
			ExpectedExitCode: intPtr(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-block-text",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, headlessSurfaceArtifacts(
		"/tmp/surface-artifacts/surface-block-text",
	), cleanArtifactScan(
		2,
	), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if len(report.TextMeasurements) < 2 || len(report.FontFallbacks) == 0 ||
		len(report.GlyphCaches) == 0 ||
		len(report.TextRenderCommands) < 2 {
		t.Fatalf(
			("text evidence = measurements %#v fallback %#v cache %#v " +
				"commands %#v, want text engine evidence"),
			report.TextMeasurements,
			report.FontFallbacks,
			report.GlyphCaches,
			report.TextRenderCommands,
		)
	}
	if !caseNamesContain(scenario.Cases, "block text wrap ellipsis layout") ||
		!caseNamesContain(scenario.Cases, "block text editable lifetime") {
		t.Fatalf(
			"scenario cases = %#v, want wrap/ellipsis and editable lifetime evidence",
			scenario.Cases,
		)
	}
	if len(scenario.Frames) < 2 || scenario.Frames[0].Checksum == scenario.Frames[1].Checksum {
		t.Fatalf("text scenario frames = %#v, want text render checksum change", scenario.Frames)
	}
}

func TestBlockLayoutScenarioProducesLayoutConstraintEvidence(t *testing.T) {
	scenario := runBlockLayoutScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless-block-layout",
		SourcePath: "examples/surface/block_core/surface_block_layout.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/block_core/surface_block_layout.tetra -o " +
				"/tmp/surface-artifacts/surface-block-layout"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-block-layout",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(1),
			ExpectedExitCode: intPtr(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-block-layout",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, headlessSurfaceArtifacts(
		"/tmp/surface-artifacts/surface-block-layout",
	), cleanArtifactScan(
		2,
	), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if len(report.LayoutConstraints) < 4 || len(report.LayoutPasses) < 8 ||
		len(report.LayoutScrolls) == 0 {
		t.Fatalf(
			"layout evidence = constraints %#v passes %#v scrolls %#v, want constraint resolver evidence",
			report.LayoutConstraints,
			report.LayoutPasses,
			report.LayoutScrolls,
		)
	}
	if !caseNamesContain(scenario.Cases, "block layout grid dock overlay scroll") ||
		!caseNamesContain(scenario.Cases, "block layout resize constraints") {
		t.Fatalf(
			"scenario cases = %#v, want grid/dock/overlay/scroll and resize evidence",
			scenario.Cases,
		)
	}
	if len(scenario.Frames) < 2 || scenario.Frames[0].Checksum == scenario.Frames[1].Checksum {
		t.Fatalf(
			"layout scenario frames = %#v, want responsive layout checksum change",
			scenario.Frames,
		)
	}
}

func TestBlockEventScenarioProducesEventFocusEvidence(t *testing.T) {
	scenario := runBlockEventScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless-block-events",
		SourcePath: "examples/surface/block_core/surface_block_events.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/block_core/surface_block_events.tetra -o " +
				"/tmp/surface-artifacts/surface-block-events"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-block-events",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(1),
			ExpectedExitCode: intPtr(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-block-events",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, headlessSurfaceArtifacts(
		"/tmp/surface-artifacts/surface-block-events",
	), cleanArtifactScan(
		2,
	), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if len(report.BlockEventRoutes) < 5 || len(report.BlockFocusTransitions) < 2 {
		t.Fatalf(
			"event evidence = routes %#v focus %#v, want routed Block event/focus evidence",
			report.BlockEventRoutes,
			report.BlockFocusTransitions,
		)
	}
	if !caseNamesContain(scenario.Cases, "block event disabled click rejected") ||
		!caseNamesContain(scenario.Cases, "block focus tab order graph-derived") {
		t.Fatalf(
			"scenario cases = %#v, want disabled click and graph-derived focus evidence",
			scenario.Cases,
		)
	}
}

func TestBlockStateScenarioProducesSelectorResolverEvidence(t *testing.T) {
	scenario := runBlockStateScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless-block-states",
		SourcePath: "examples/surface/block_core/surface_block_states.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/block_core/surface_block_states.tetra -o " +
				"/tmp/surface-artifacts/surface-block-states"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-block-states",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(1),
			ExpectedExitCode: intPtr(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-block-states",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, headlessSurfaceArtifacts(
		"/tmp/surface-artifacts/surface-block-states",
	), cleanArtifactScan(
		2,
	), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if len(report.BlockStateSelectors) < 7 || len(report.BlockStateResolutions) < 8 {
		t.Fatalf(
			"state evidence = selectors %#v resolutions %#v, want generic selector resolver evidence",
			report.BlockStateSelectors,
			report.BlockStateResolutions,
		)
	}
	if !caseNamesContain(scenario.Cases, "block state hover fill override") ||
		!caseNamesContain(scenario.Cases, "block state disabled error loading overrides") {
		t.Fatalf(
			"scenario cases = %#v, want hover and disabled/error/loading state evidence",
			scenario.Cases,
		)
	}
	if len(scenario.Frames) < 2 || scenario.Frames[0].Checksum == scenario.Frames[1].Checksum {
		t.Fatalf("state scenario frames = %#v, want state-driven checksum change", scenario.Frames)
	}
}

func TestBlockMotionScenarioProducesTransitionEvidence(t *testing.T) {
	scenario := runBlockMotionScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless-block-motion",
		SourcePath: "examples/surface/block_core/surface_block_motion.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/block_core/surface_block_motion.tetra -o " +
				"/tmp/surface-artifacts/surface-block-motion"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-block-motion",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(1),
			ExpectedExitCode: intPtr(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-block-motion",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, headlessSurfaceArtifacts(
		"/tmp/surface-artifacts/surface-block-motion",
	), cleanArtifactScan(
		2,
	), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if len(report.MotionFrames) < 4 {
		t.Fatalf(
			"motion frames = %#v, want deterministic transition and reduced-motion evidence",
			report.MotionFrames,
		)
	}
	last := report.MotionFrames[len(report.MotionFrames)-1]
	if !last.ReducedMotion || last.Scheduled || !last.Settled {
		t.Fatalf("last motion frame = %#v, want reduced motion settled without scheduling", last)
	}
	if !caseNamesContain(scenario.Cases, "block motion opacity color transform frames") ||
		!caseNamesContain(scenario.Cases, "block motion completion stops scheduling") {
		t.Fatalf("scenario cases = %#v, want transition and completion evidence", scenario.Cases)
	}
}

func TestBlockAssetScenarioProducesLocalAssetEvidence(t *testing.T) {
	scenario := runBlockAssetScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless-block-assets",
		SourcePath: "examples/surface/block_render/surface_block_assets.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/block_render/surface_block_assets.tetra -o " +
				"/tmp/surface-artifacts/surface-block-assets"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-block-assets",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(1),
			ExpectedExitCode: intPtr(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-block-assets",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, headlessSurfaceArtifacts(
		"/tmp/surface-artifacts/surface-block-assets",
	), cleanArtifactScan(
		2,
	), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.BlockAssetManifest == nil || len(report.BlockAssetManifest.Assets) < 3 {
		t.Fatalf(
			"asset manifest = %#v, want font/icon/image asset hashes",
			report.BlockAssetManifest,
		)
	}
	if report.BlockAssetNetworkFetchAllowed {
		t.Fatalf("network fetch allowed, want local/embedded-only asset evidence")
	}
	if !report.BlockAssetCache.Bounded || report.BlockAssetCache.UsedBytes <= 0 {
		t.Fatalf("asset cache = %#v, want bounded cache use evidence", report.BlockAssetCache)
	}
	if len(report.BlockAssetDiagnostics) == 0 {
		t.Fatalf(
			"asset diagnostics = %#v, want missing asset fallback and network rejection evidence",
			report.BlockAssetDiagnostics,
		)
	}
	for _, want := range []string{
		"block asset icon tint evidence",
		"block asset image scale evidence",
		"block asset network url rejected",
	} {
		if !caseNamesContain(scenario.Cases, want) {
			t.Fatalf("scenario cases = %#v, want %q", scenario.Cases, want)
		}
	}
	if len(scenario.Frames) < 2 || scenario.Frames[0].Checksum == scenario.Frames[1].Checksum {
		t.Fatalf("asset scenario frames = %#v, want asset-driven checksum change", scenario.Frames)
	}
}

func TestBlockAccessibilityScenarioProducesGraphDerivedMetadataEvidence(t *testing.T) {
	scenario := runBlockAccessibilityScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless-block-accessibility",
		SourcePath: "examples/surface/block_render/surface_block_accessibility.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/block_render/surface_block_accessibility.te" +
				"tra -o /tmp/surface-artifacts/surface-block-accessibility"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-block-accessibility",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(1),
			ExpectedExitCode: intPtr(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-block-accessibility",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, headlessSurfaceArtifacts(
		"/tmp/surface-artifacts/surface-block-accessibility",
	), cleanArtifactScan(
		2,
	), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.BlockAccessibilityTree == nil {
		t.Fatalf("block accessibility tree is nil, want Block-derived metadata evidence")
	}
	if report.BlockAccessibilityTree.PlatformHostIntegration ||
		report.BlockAccessibilityTree.ScreenReaderEvidence != false {
		t.Fatalf(
			"block accessibility claims = platform %t screen-reader %#v, want metadata-only scoped claims",
			report.BlockAccessibilityTree.PlatformHostIntegration,
			report.BlockAccessibilityTree.ScreenReaderEvidence,
		)
	}
	if !intSlicesEqual(
		report.BlockAccessibilityTree.ReadingOrder,
		report.BlockGraph.AccessibilityOrder,
	) {
		t.Fatalf(
			"reading order = %#v, want block graph accessibility order %#v",
			report.BlockAccessibilityTree.ReadingOrder,
			report.BlockGraph.AccessibilityOrder,
		)
	}
	for _, want := range []string{"block accessibility tree derived from block graph", (("block accessibility " +
		"screen-reader claim without platform ") +
		"proof rejected")} {
		if !caseNamesContain(scenario.Cases, want) {
			t.Fatalf("scenario cases = %#v, want %q", scenario.Cases, want)
		}
	}
}

func TestBlockSystemScenarioProducesHeadlessGoldenChecksumEvidence(t *testing.T) {
	if err := validateSmokeMode("headless-block-system"); err != nil {
		t.Fatalf("validateSmokeMode(headless-block-system) failed: %v", err)
	}
	if got := defaultSurfaceSourcePath(
		smokeOptions{
			Mode:       "headless-block-system",
			SourcePath: "examples/surface/runtime/surface_counter.tetra",
		},
	); got != "examples/surface/block_core/surface_block_system.tetra" {
		t.Fatalf(
			("defaultSurfaceSourcePath(headless-block-system) = %q, want " +
				"examples/surface/block_core/surface_block_system.tetra"),
			got,
		)
	}

	scenario := runBlockSystemScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless-block-system",
		SourcePath: "examples/surface/block_core/surface_block_system.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/block_core/surface_block_system.tetra -o " +
				"/tmp/surface-artifacts/surface-block-system"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-block-system",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(1),
			ExpectedExitCode: intPtr(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-block-system",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, headlessSurfaceArtifacts(
		"/tmp/surface-artifacts/surface-block-system",
	), cleanArtifactScan(
		2,
	), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.BlockSystem == nil {
		t.Fatalf("block_system is nil, want headless Block golden/checksum evidence")
	}
	if report.BlockSystem.Schema != "tetra.surface.block-system.v1" ||
		report.BlockSystem.QualityLevel != "deterministic-headless-block-system-v1" {
		t.Fatalf(
			"block_system = %#v, want deterministic headless Block system schema",
			report.BlockSystem,
		)
	}
	if len(report.BlockSystem.Frames) != len(report.Frames) {
		t.Fatalf(
			"block_system frames = %d, report frames = %d",
			len(report.BlockSystem.Frames),
			len(report.Frames),
		)
	}
	for i, frame := range report.BlockSystem.Frames {
		if frame.Checksum != report.Frames[i].Checksum ||
			frame.GoldenChecksum != report.Frames[i].Checksum {
			t.Fatalf(
				"block_system frame %d checksums = %#v, report frame = %#v",
				i,
				frame,
				report.Frames[i],
			)
		}
		if frame.RepeatChecksum != frame.Checksum {
			t.Fatalf(
				"block_system frame %d repeat checksum = %q, want %q",
				i,
				frame.RepeatChecksum,
				frame.Checksum,
			)
		}
		if !frame.PaintEvidence || !frame.LayoutEvidence || !frame.AccessibilityEvidence {
			t.Fatalf(
				"block_system frame %d = %#v, want paint/layout/accessibility evidence flags",
				i,
				frame,
			)
		}
	}
	for _, want := range []string{
		"block system headless golden checksums",
		"block system deterministic repeat checksum",
		"block system missing frame checksum rejected",
		"block system nondeterministic checksum rejected",
		"block system missing paint evidence rejected",
		"block system missing layout evidence rejected",
		"block system missing accessibility evidence rejected",
	} {
		if !caseNamesContain(scenario.Cases, want) {
			t.Fatalf("scenario cases = %#v, want %q", scenario.Cases, want)
		}
	}
}

func TestBlockSystemScenarioWritesFrameArtifacts(t *testing.T) {
	dir := t.TempDir()
	scenario := runBlockSystemScenario()
	opt := smokeOptions{
		Mode:       "headless-block-system",
		SourcePath: "examples/surface/block_core/surface_block_system.tetra",
		ReportPath: filepath.Join(dir, "surface-headless-block-system.json"),
	}
	if err := attachBlockSystemFrameArtifacts(opt, &scenario); err != nil {
		t.Fatalf("attachBlockSystemFrameArtifacts failed: %v", err)
	}
	for _, frame := range scenario.Frames {
		if frame.ArtifactPath == "" {
			t.Fatalf("frame %d missing artifact_path", frame.Order)
		}
		raw, err := os.ReadFile(frame.ArtifactPath)
		if err != nil {
			t.Fatalf("read frame artifact %s: %v", frame.ArtifactPath, err)
		}
		if got := checksumRGBA(raw); got != frame.Checksum {
			t.Fatalf(
				"frame artifact %s checksum = %s, want %s",
				frame.ArtifactPath,
				got,
				frame.Checksum,
			)
		}
	}
	for _, frame := range scenario.BlockSystem.Frames {
		if frame.ArtifactPath == "" {
			t.Fatalf("block_system frame %d missing artifact_path", frame.Order)
		}
	}
}

func TestWASM32WebBrowserCanvasBlockSystemUsesExistingFrameArtifacts(t *testing.T) {
	dir := t.TempDir()
	initialFrame := renderBlockSystemFrameSizedRGBA(320, 200, false)
	focusedFrame := renderBlockSystemFrameSizedRGBA(400, 240, true)
	initialPath := filepath.Join(dir, "surface-browser-canvas-frame-order-1.rgba")
	focusedPath := filepath.Join(dir, "surface-browser-canvas-frame-order-5.rgba")
	if err := os.WriteFile(initialPath, initialFrame.Pixels, 0o644); err != nil {
		t.Fatalf("write initial frame artifact: %v", err)
	}
	if err := os.WriteFile(focusedPath, focusedFrame.Pixels, 0o644); err != nil {
		t.Fatalf("write focused frame artifact: %v", err)
	}

	scenario := runWASM32WebBrowserCanvasBlockSystemScenario()
	scenario.Frames = []surface.FrameReport{
		{
			Order:        1,
			Width:        initialFrame.Width,
			Height:       initialFrame.Height,
			Stride:       initialFrame.Stride,
			Checksum:     checksumRGBA(initialFrame.Pixels),
			ArtifactPath: initialPath,
			Presented:    true,
		},
		{
			Order:        5,
			Width:        focusedFrame.Width,
			Height:       focusedFrame.Height,
			Stride:       focusedFrame.Stride,
			Checksum:     checksumRGBA(focusedFrame.Pixels),
			ArtifactPath: focusedPath,
			Presented:    true,
		},
	}
	scenario.BlockSystem = blockSystemReportForWASM32WebBrowserCanvasScenario(
		"examples/surface/block_core/surface_block_system.tetra",
		scenario.Frames,
	)
	opt := smokeOptions{
		Mode:       "wasm32-web-browser-canvas-block-system",
		SourcePath: "examples/surface/block_core/surface_block_system.tetra",
		ReportPath: filepath.Join(dir, "surface-wasm-block-system.json"),
	}
	if err := attachBlockSystemFrameArtifacts(opt, &scenario); err != nil {
		t.Fatalf("attachBlockSystemFrameArtifacts failed: %v", err)
	}

	for _, frame := range scenario.BlockSystem.Frames {
		switch frame.Order {
		case 1:
			if frame.ArtifactPath != initialPath {
				t.Fatalf(
					"block_system frame order 1 artifact_path = %q, want %q",
					frame.ArtifactPath,
					initialPath,
				)
			}
		case 5:
			if frame.ArtifactPath != focusedPath {
				t.Fatalf(
					"block_system frame order 5 artifact_path = %q, want %q",
					frame.ArtifactPath,
					focusedPath,
				)
			}
		default:
			t.Fatalf("unexpected block_system frame order %d", frame.Order)
		}
	}
	syntheticPath := filepath.Join(
		surfaceRuntimeArtifactDir(opt),
		"surface-block-system-frame-order-1-initial.rgba",
	)
	if _, err := os.Stat(syntheticPath); err == nil {
		t.Fatalf("synthetic wasm Block-system artifact exists at %s", syntheticPath)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat synthetic wasm Block-system artifact %s: %v", syntheticPath, err)
	}
}

func TestMorphScenarioProducesHeadlessCapsuleEvidence(t *testing.T) {
	const mode = "headless-morph"
	const source = "examples/surface/morph_core/surface_morph_command_palette.tetra"
	if err := validateSmokeMode(mode); err != nil {
		t.Fatalf("validateSmokeMode(%s) failed: %v", mode, err)
	}
	if got := defaultSurfaceSourcePath(
		smokeOptions{Mode: mode, SourcePath: "examples/surface/runtime/surface_counter.tetra"},
	); got != source {
		t.Fatalf("defaultSurfaceSourcePath(%s) = %q, want %s", mode, got, source)
	}
	scenario := runMorphScenario()
	report := buildReport(smokeOptions{
		Mode:       mode,
		SourcePath: source,
	}, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/morph_core/surface_morph_command_palette.te" +
				"tra -o /tmp/surface-artifacts/surface-morph-command-palette"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-morph-command-palette",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(1),
			ExpectedExitCode: intPtr(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-morph",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, headlessSurfaceArtifacts(
		"/tmp/surface-artifacts/surface-morph-command-palette",
	), cleanArtifactScan(
		2,
	), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.Morph == nil {
		t.Fatalf("morph is nil, want Morph capsule evidence")
	}
	if report.Morph.Schema != "tetra.surface.morph.v1" ||
		report.Morph.QualityLevel != "deterministic-headless-morph-capsule-v1" {
		t.Fatalf("morph = %#v, want Morph v1 deterministic headless capsule evidence", report.Morph)
	}
	if report.Morph.TokenGraph == nil || report.Morph.TokenGraph.SourceOfTruth != "capsule" ||
		!report.Morph.TokenGraph.ExplicitImports ||
		!report.Morph.TokenGraph.NoGlobalCascade {
		t.Fatalf(
			"morph token graph = %#v, want P07 capsule source-of-truth boundary evidence",
			report.Morph.TokenGraph,
		)
	}
	if len(report.Morph.TokenGraph.Tokens) < 22 || len(report.Morph.TokenGraph.DensityDPI) != 3 ||
		!report.Morph.TokenGraph.Diagnostics.CSSRuntimeRejected {
		t.Fatalf(
			"morph token graph = %#v, want P07 typed tokens, density mapping, and diagnostics",
			report.Morph.TokenGraph,
		)
	}
	if len(report.Morph.Recipes) != 19 || len(report.Morph.RecipeExpansions) != 19 ||
		len(report.Morph.RecipeApps) != 7 {
		t.Fatalf(
			"morph recipes=%d expansions=%d apps=%d, want P08 recipe authoring suite",
			len(
				report.Morph.Recipes,
			),
			len(report.Morph.RecipeExpansions),
			len(report.Morph.RecipeApps),
		)
	}
	if report.BlockSystem == nil || report.BlockSystem.Source != source ||
		report.BlockGraph.Source != source {
		t.Fatalf(
			"Block evidence sources = block_system %#v block_graph %#v, want Morph source %s",
			report.BlockSystem,
			report.BlockGraph,
			source,
		)
	}
	if !caseNamesContain(report.Cases, "morph recipes expand to Block graph") {
		t.Fatalf("cases = %#v, want Morph recipe expansion evidence", report.Cases)
	}
}

func TestMorphScenarioProducesBlockSceneSnapshotEvidence(t *testing.T) {
	const mode = "headless-morph"
	const source = "examples/surface/morph_core/surface_morph_command_palette.tetra"
	scenario := runMorphScenario()
	report := buildReport(smokeOptions{
		Mode:       mode,
		SourcePath: source,
	}, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/morph_core/surface_morph_command_palette.te" +
				"tra -o /tmp/surface-artifacts/surface-morph-command-palette"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-morph-command-palette",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(1),
			ExpectedExitCode: intPtr(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-morph",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, headlessSurfaceArtifacts(
		"/tmp/surface-artifacts/surface-morph-command-palette",
	), cleanArtifactScan(
		2,
	), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	snapshot := report.BlockSceneSnapshot
	if snapshot == nil {
		t.Fatalf("block_scene_snapshot is nil, want rich renderable Block scene snapshot evidence")
	}
	if snapshot.Source != source || snapshot.QualityLevel != "rich-renderable-block-scene-v1" {
		t.Fatalf(
			"block_scene_snapshot = %#v, want Morph source and rich renderable quality",
			snapshot,
		)
	}
	if snapshot.CompactPropsOnly || len(snapshot.Nodes) == 0 ||
		snapshot.NodeCount != len(snapshot.Nodes) {
		t.Fatalf(
			"block_scene_snapshot compact=%t node_count=%d len(nodes)=%d, want rich node evidence",
			snapshot.CompactPropsOnly,
			snapshot.NodeCount,
			len(snapshot.Nodes),
		)
	}
	coverage := snapshot.SpecCoverage
	if !coverage.Layout || !coverage.Paint || !coverage.Text || !coverage.Image ||
		!coverage.Input ||
		!coverage.Event ||
		!coverage.State ||
		!coverage.Motion ||
		!coverage.Accessibility {
		t.Fatalf(
			"block_scene_snapshot spec coverage = %#v, want all rich specs preserved",
			coverage,
		)
	}
	for _, want := range []string{
		"block scene snapshot preserves rich visual specs",
		"block scene compact BlockProps-only evidence rejected",
		"block scene non-Block core primitive rejected",
		"block scene missing rich spec coverage rejected",
	} {
		if !caseNamesContain(report.Cases, want) {
			t.Fatalf("cases = %#v, want %q", report.Cases, want)
		}
	}
}

func TestMorphScenarioProducesRenderCommandStreamEvidence(t *testing.T) {
	const mode = "headless-morph"
	const source = "examples/surface/morph_core/surface_morph_command_palette.tetra"
	scenario := runMorphScenario()
	report := buildReport(smokeOptions{
		Mode:       mode,
		SourcePath: source,
	}, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/morph_core/surface_morph_command_palette.te" +
				"tra -o /tmp/surface-artifacts/surface-morph-command-palette"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-morph-command-palette",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(1),
			ExpectedExitCode: intPtr(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-morph",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, headlessSurfaceArtifacts(
		"/tmp/surface-artifacts/surface-morph-command-palette",
	), cleanArtifactScan(
		2,
	), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	stream := report.RenderCommandStream
	if stream == nil {
		t.Fatalf(
			"render_command_stream is nil, want source-linked stream from Morph-authored Block scene",
		)
	}
	if report.BlockSceneSnapshot == nil {
		t.Fatalf("block_scene_snapshot is nil")
	}
	if stream.Source != source ||
		stream.BlockSceneHash != report.BlockSceneSnapshot.BlockSceneHash ||
		!stream.DerivedFromBlockSceneSnapshot {
		t.Fatalf(
			"render_command_stream = %#v, want source %s and block_scene_hash %s",
			stream,
			source,
			report.BlockSceneSnapshot.BlockSceneHash,
		)
	}
	if !stream.SourceLinked || stream.HandcraftedFixture ||
		stream.CommandCount != len(stream.Commands) ||
		stream.CommandCount < 10 {
		t.Fatalf(
			"render_command_stream = %#v, want source-linked non-fixture command evidence",
			stream,
		)
	}
	if len(report.Frames) == 0 || stream.FrameChecksum != report.Frames[0].Checksum {
		t.Fatalf(
			"render_command_stream frame_checksum = %q, want first frame checksum",
			stream.FrameChecksum,
		)
	}
	for _, command := range []string{
		"fill",
		"gradient",
		"image_fill",
		"border",
		"radius_clip",
		"shadow",
		"overlay",
		"outline",
		"text",
		"icon",
	} {
		if !renderCommandStreamHasCommand(stream.Commands, command) {
			t.Fatalf("render_command_stream commands = %#v, want %s", stream.Commands, command)
		}
	}
	for _, command := range stream.Commands {
		if command.Source != source || command.Recipe == "" || command.BlockID <= 0 {
			t.Fatalf("render command = %#v, want source, recipe, and block_id links", command)
		}
	}
}

func TestMorphScenarioBuildsRenderedBeautyReportFromRuntimeAndVisualEvidence(t *testing.T) {
	const mode = "headless-morph"
	const source = "examples/surface/morph_core/surface_morph_command_palette.tetra"
	scenario := runMorphScenario()
	opt := smokeOptions{
		Mode:       mode,
		SourcePath: source,
		ReportPath: filepath.Join(t.TempDir(), "surface-headless-morph.json"),
	}
	if err := attachMorphRenderedBeautyFrameArtifacts(opt, &scenario); err != nil {
		t.Fatalf("attachMorphRenderedBeautyFrameArtifacts: %v", err)
	}
	report := buildReport(opt, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/morph_core/surface_morph_command_palette.te" +
				"tra -o /tmp/surface-artifacts/surface-morph-command-palette"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-morph-command-palette",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(1),
			ExpectedExitCode: intPtr(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-morph",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, headlessSurfaceArtifacts(
		"/tmp/surface-artifacts/surface-morph-command-palette",
	), cleanArtifactScan(
		2,
	), scenario)
	visual := morphRenderedBeautyVisualReportForTest("reports/surface/morph-runtime.json", report)

	mrb, err := buildMorphRenderedBeautyReport(
		"reports/surface/morph-runtime.json",
		report,
		visual,
		"headless-morph",
	)
	if err != nil {
		t.Fatalf("buildMorphRenderedBeautyReport failed: %v", err)
	}
	if err := surface.ValidateMorphRenderedBeautyReportValue(mrb); err != nil {
		t.Fatalf("ValidateMorphRenderedBeautyReportValue failed: %v", err)
	}
	if report.Frames[0].ArtifactPath == "" || report.Frames[0].Producer != "app" ||
		report.Frames[0].EvidenceRole != "product_visual" {
		t.Fatalf(
			"morph frame provenance = %#v, want app-produced product_visual artifact",
			report.Frames[0],
		)
	}
	if mrb.Schema != surface.MorphRenderedBeautyReportSchemaV1 || mrb.Target != "headless" ||
		mrb.ScenarioName != "headless-morph" {
		t.Fatalf("MRB report identity = %#v, want schema, target, and scenario", mrb)
	}
	if mrb.MorphEvidence.Source != source || mrb.MorphEvidence.TokenCount == 0 ||
		len(mrb.MorphEvidence.RecipeNames) == 0 {
		t.Fatalf(
			"morph evidence = %#v, want source, token coverage, and recipe coverage",
			mrb.MorphEvidence,
		)
	}
	if mrb.BlockSceneSnapshot.BlockSceneHash != report.BlockSceneSnapshot.BlockSceneHash {
		t.Fatalf(
			"block scene hash = %q, want %q",
			mrb.BlockSceneSnapshot.BlockSceneHash,
			report.BlockSceneSnapshot.BlockSceneHash,
		)
	}
	if mrb.RenderCommandStream.CommandStreamHash != report.RenderCommandStream.CommandStreamHash {
		t.Fatalf(
			"render command stream hash = %q, want %q",
			mrb.RenderCommandStream.CommandStreamHash,
			report.RenderCommandStream.CommandStreamHash,
		)
	}
	if mrb.RendererStableProof.PixelOwner != "surface-renderer" ||
		!mrb.RendererStableProof.RendererOwned ||
		mrb.RendererStableProof.BridgeOwnedPixels ||
		!mrb.RendererStableProof.DerivedFromRenderCommandStream ||
		!mrb.RendererStableProof.StablePromotionEligible {
		t.Fatalf(
			"renderer stable proof = %#v, want renderer-owned command-stream-derived proof",
			mrb.RendererStableProof,
		)
	}
	if mrb.PixelEvidence.FrameProducer != "app" || mrb.PixelEvidence.AppSource != source ||
		mrb.PixelEvidence.RenderCommandStreamHash != report.RenderCommandStream.CommandStreamHash {
		t.Fatalf(
			"pixel evidence = %#v, want app-produced source-linked pixel chain",
			mrb.PixelEvidence,
		)
	}
}

func TestApplyMorphRenderedBeautyProductSignoffRequiresCleanRendererOwnedProof(t *testing.T) {
	const mode = "headless-morph"
	const source = "examples/surface/morph_core/surface_morph_command_palette.tetra"
	scenario := runMorphScenario()
	opt := smokeOptions{
		Mode:       mode,
		SourcePath: source,
		ReportPath: filepath.Join(t.TempDir(), "surface-headless-morph.json"),
	}
	if err := attachMorphRenderedBeautyFrameArtifacts(opt, &scenario); err != nil {
		t.Fatalf("attachMorphRenderedBeautyFrameArtifacts: %v", err)
	}
	report := buildReport(opt, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/morph_core/surface_morph_command_palette.te" +
				"tra -o /tmp/surface-artifacts/surface-morph-command-palette"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-morph-command-palette",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(1),
			ExpectedExitCode: intPtr(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-morph",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, headlessSurfaceArtifacts(
		"/tmp/surface-artifacts/surface-morph-command-palette",
	), cleanArtifactScan(
		2,
	), scenario)
	visual := morphRenderedBeautyVisualReportForTest("reports/surface/morph-runtime.json", report)

	mrb, err := buildMorphRenderedBeautyReport(
		"reports/surface/morph-runtime.json",
		report,
		visual,
		"headless-morph",
	)
	if err != nil {
		t.Fatalf("buildMorphRenderedBeautyReport failed: %v", err)
	}
	mrb.GitDirty = false
	if err := applyMorphRenderedBeautyProductSignoff(&mrb, true, true); err != nil {
		t.Fatalf(
			"applyMorphRenderedBeautyProductSignoff rejected clean renderer-owned proof: %v",
			err,
		)
	}
	if !mrb.ProductClaim || !mrb.FinalSignoff {
		t.Fatalf(
			"MRB signoff = product_claim:%t final_signoff:%t, want true/true",
			mrb.ProductClaim,
			mrb.FinalSignoff,
		)
	}
	if err := surface.ValidateMorphRenderedBeautyReportValue(mrb); err != nil {
		t.Fatalf("ValidateMorphRenderedBeautyReportValue rejected signed MRB report: %v", err)
	}

	mrb.GitDirty = true
	mrb.ProductClaim = false
	mrb.FinalSignoff = false
	err = applyMorphRenderedBeautyProductSignoff(&mrb, true, true)
	if err == nil {
		t.Fatalf("expected dirty product signoff to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "dirty") {
		t.Fatalf("error = %v, want dirty diagnostic", err)
	}
}

func TestMorphFlagshipScenarioProducesRenderedBeautyReport(t *testing.T) {
	const mode = "headless-morph"
	const source = "examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra"
	rawSource, err := os.ReadFile(filepath.Clean(filepath.Join("..", "..", "..", source)))
	if err != nil {
		t.Fatalf("read flagship Morph source: %v", err)
	}
	if strings.Contains(string(rawSource), "import lib.core.draw") ||
		strings.Contains(string(rawSource), "func draw(") {
		t.Fatalf("flagship Morph source must not use manual draw authoring")
	}

	scenario := runMorphScenarioForSource(source)
	opt := smokeOptions{
		Mode:       mode,
		SourcePath: source,
		ReportPath: filepath.Join(t.TempDir(), "surface-headless-morph-flagship.json"),
	}
	if err := attachMorphRenderedBeautyFrameArtifacts(opt, &scenario); err != nil {
		t.Fatalf("attachMorphRenderedBeautyFrameArtifacts: %v", err)
	}
	report := buildReport(opt, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/morph_flagship/surface_morph_rendered_studi" +
				"o_shell.tetra -o " +
				"/tmp/surface-artifacts/surface-morph-rendered-studio-shell"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-morph-rendered-studio-shell",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(0),
			ExpectedExitCode: intPtr(0),
		},
		{
			Name: "surface headless runtime",
			Kind: "runtime",
			Path: ("tools/cmd/surface-runtime-smoke --mode headless-morph " +
				"--source " +
				"examples/surface/morph_flagship/surface_morph_rendered_studi" +
				"o_shell.tetra"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, headlessSurfaceArtifacts(
		"/tmp/surface-artifacts/surface-morph-rendered-studio-shell",
	), cleanArtifactScan(
		2,
	), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.Source != source || report.Morph == nil || report.Morph.Source != source ||
		report.BlockSceneSnapshot.Source != source ||
		report.RenderCommandStream.Source != source {
		t.Fatalf(
			"flagship evidence sources = report %q morph %#v scene %#v stream %#v, want %s",
			report.Source,
			report.Morph,
			report.BlockSceneSnapshot,
			report.RenderCommandStream,
			source,
		)
	}
	if report.BlockSceneSnapshot.NodeCount < 18 || report.Morph == nil ||
		len(report.Morph.RecipeExpansions) < 16 {
		t.Fatalf(
			"flagship scene coverage nodes=%d expansions=%d, want contract-scale Morph flagship evidence",
			report.BlockSceneSnapshot.NodeCount,
			len(report.Morph.RecipeExpansions),
		)
	}
	for _, want := range []string{
		"app.shell@1",
		"nav.item@1",
		"toolbar@1",
		"split.pane@1",
		"status.bar@1",
		"command.item@1",
		"settings.form@1",
		"log.row@1",
		"metric.tile@1",
		"toast.notification@1",
		"dialog.panel@1",
		"empty.state@1",
		"error.panel@1",
	} {
		if !morphRecipeNamesContain(report.Morph.Recipes, want) {
			t.Fatalf("flagship Morph recipes missing %q in %#v", want, report.Morph.Recipes)
		}
	}
	for _, want := range []string{
		"DashboardShell",
		"ProfilesActions",
		"CommandPalette",
		"SettingsForm",
		"LogsOutput",
		"DiagnosticsError",
		"StatusBar",
		"BlockedDialog",
	} {
		if !blockSceneSnapshotHasNode(report.BlockSceneSnapshot, want) {
			t.Fatalf(
				"flagship Block scene missing %q in %#v",
				want,
				report.BlockSceneSnapshot.Nodes,
			)
		}
	}
	visual := morphRenderedBeautyVisualReportForTest(
		"reports/surface/flagship-morph-runtime.json",
		report,
	)
	mrb, err := buildMorphRenderedBeautyReport(
		"reports/surface/flagship-morph-runtime.json",
		report,
		visual,
		morphRenderedBeautyScenarioName(opt),
	)
	if err != nil {
		t.Fatalf("buildMorphRenderedBeautyReport failed: %v", err)
	}
	if err := surface.ValidateMorphRenderedBeautyReportValue(mrb); err != nil {
		t.Fatalf("ValidateMorphRenderedBeautyReportValue failed: %v", err)
	}
	if mrb.MorphEvidence.Source != source || mrb.ScenarioName != mode+":"+source ||
		mrb.PixelEvidence.AppSource != source {
		t.Fatalf(
			"flagship MRB source/scenario = %#v pixel %#v, want %s",
			mrb.MorphEvidence,
			mrb.PixelEvidence,
			source,
		)
	}
	if mrb.RendererStableProof.PixelOwner != "surface-renderer" ||
		!mrb.RendererStableProof.RendererOwned ||
		mrb.RendererStableProof.BridgeOwnedPixels ||
		!mrb.RendererStableProof.DerivedFromRenderCommandStream ||
		!mrb.RendererStableProof.StablePromotionEligible {
		t.Fatalf(
			"flagship renderer stable proof = %#v, want renderer-owned command-stream-derived proof",
			mrb.RendererStableProof,
		)
	}
}

func TestGuestDashboardScenarioProducesPersonalCabinetEmptyState(t *testing.T) {
	const mode = "headless-morph"
	const source = "examples/surface/morph_flagship/surface_morph_guest_dashboard.tetra"
	rawSource, err := os.ReadFile(filepath.Clean(filepath.Join("..", "..", "..", source)))
	if err != nil {
		t.Fatalf("read guest dashboard Morph source: %v", err)
	}
	if strings.Contains(string(rawSource), "import lib.core.draw") ||
		strings.Contains(string(rawSource), "func draw(") {
		t.Fatalf("guest dashboard Morph source must not use manual draw authoring")
	}

	scenario := runMorphScenarioForSource(source)
	opt := smokeOptions{
		Mode:       mode,
		SourcePath: source,
		ReportPath: filepath.Join(t.TempDir(), "surface-headless-morph-guest-dashboard.json"),
	}
	report := buildReport(opt, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/morph_flagship/surface_morph_guest_dashboar" +
				"d.tetra -o " +
				"/tmp/surface-artifacts/surface-morph-guest-dashboard"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-morph-guest-dashboard",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(0),
			ExpectedExitCode: intPtr(0),
		},
		{
			Name: "surface headless runtime",
			Kind: "runtime",
			Path: ("tools/cmd/surface-runtime-smoke --mode headless-morph " +
				"--source " +
				"examples/surface/morph_flagship/surface_morph_guest_dashboar" +
				"d.tetra"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, headlessSurfaceArtifacts(
		"/tmp/surface-artifacts/surface-morph-guest-dashboard",
	), cleanArtifactScan(
		2,
	), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.Source != source || report.Morph == nil || report.Morph.Source != source ||
		report.BlockSceneSnapshot.Source != source ||
		report.RenderCommandStream.Source != source {
		t.Fatalf(
			"guest dashboard evidence sources = report %q morph %#v scene %#v stream %#v, want %s",
			report.Source,
			report.Morph,
			report.BlockSceneSnapshot,
			report.RenderCommandStream,
			source,
		)
	}
	for _, want := range []string{
		"GuestDashboardPage",
		"PersonalCabinetTitle",
		"RecentCoursesPanel",
		"RecentCoursesEmptyState",
		"CourseOverviewPanel",
		"CourseOverviewDivider",
		"CourseOverviewEmptyState",
		"CourseOverviewHeadline",
		"CourseOverviewBody",
	} {
		if !blockSceneSnapshotHasNode(report.BlockSceneSnapshot, want) {
			t.Fatalf(
				"guest dashboard Block scene missing %q in %#v",
				want,
				report.BlockSceneSnapshot.Nodes,
			)
		}
	}
	for _, want := range []string{"region.panel@1", "empty.state@1", "app.shell@1"} {
		if !morphRecipeNamesContain(report.Morph.Recipes, want) {
			t.Fatalf("guest dashboard Morph recipes missing %q in %#v", want, report.Morph.Recipes)
		}
	}
	if report.BlockSceneSnapshot.NodeCount < 9 {
		t.Fatalf(
			"guest dashboard node_count = %d, want page title, two panels, divider, and empty-state content",
			report.BlockSceneSnapshot.NodeCount,
		)
	}
}

func TestWASM32WebBrowserCanvasMorphScenarioUsesBrowserCanvasPixelEvidence(t *testing.T) {
	const mode = "wasm32-web-browser-canvas-morph"
	const source = "examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra"
	if err := validateSmokeMode(mode); err != nil {
		t.Fatalf("validateSmokeMode(%s) failed: %v", mode, err)
	}

	dir := t.TempDir()
	initialFrame := renderMorphStudioShellFrameRGBA(320, 200, false)
	focusedFrame := renderMorphStudioShellFrameRGBA(320, 200, true)
	initialArtifact := filepath.Join(dir, "surface-browser-canvas-frame-order-1.rgba")
	focusedArtifact := filepath.Join(dir, "surface-browser-canvas-frame-order-5.rgba")
	if err := os.WriteFile(initialArtifact, initialFrame.Pixels, 0o644); err != nil {
		t.Fatalf("write initial browser canvas artifact: %v", err)
	}
	if err := os.WriteFile(focusedArtifact, focusedFrame.Pixels, 0o644); err != nil {
		t.Fatalf("write focused browser canvas artifact: %v", err)
	}
	initialChecksum := checksumRGBA(initialFrame.Pixels)
	focusedChecksum := checksumRGBA(focusedFrame.Pixels)

	scenario := runMorphScenarioForSource(source)
	opt := smokeOptions{
		Mode:       mode,
		SourcePath: source,
		ReportPath: filepath.Join(dir, "surface-wasm-browser-canvas-morph.json"),
	}
	browserFrames := []surface.FrameReport{
		{
			Order:        1,
			Width:        320,
			Height:       200,
			Stride:       1280,
			Checksum:     initialChecksum,
			ArtifactPath: initialArtifact,
			Presented:    true,
		},
		{
			Order:        5,
			Width:        320,
			Height:       200,
			Stride:       1280,
			Checksum:     focusedChecksum,
			ArtifactPath: focusedArtifact,
			Presented:    true,
		},
	}
	if err := applyMorphTargetRuntimeFrameEvidence(opt, &scenario, browserFrames); err != nil {
		t.Fatalf("applyMorphTargetRuntimeFrameEvidence: %v", err)
	}
	report := buildReport(opt, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target wasm32-web " +
				"examples/surface/morph_flagship/surface_morph_rendered_studi" +
				"o_shell.tetra -o " +
				"/tmp/surface-artifacts/surface-morph-rendered-studio-shell.w" +
				"asm"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name: "surface wasm32-web browser canvas component app",
			Kind: "app",
			Path: ("/usr/bin/chromium --headless " +
				"<surface-browser-canvas-runner> scenario=studio-shell " +
				"wasm=/tmp/surface-artifacts/surface-morph-rendered-studio-sh" +
				"ell.wasm"),
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(0),
			ExpectedExitCode: intPtr(0),
		},
		{
			Name: "surface wasm32-web import validator",
			Kind: "runtime",
			Path: ("go run ./tools/cmd/validate-wasm-imports --target " +
				"wasm32-web " +
				"/tmp/surface-artifacts/surface-morph-rendered-studio-shell.w" +
				"asm"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:     "surface wasm32-web browser canvas runtime",
			Kind:     "runtime",
			Path:     "Chromium studio-shell fixture",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name: "surface wasm32-web browser canvas trace",
			Kind: "runtime",
			Path: ("/usr/bin/chromium --headless --dump-dom " +
				"<surface-browser-canvas-file-runner scenario=studio-shell>"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, []surface.ArtifactReport{
		{
			Kind:   "component-app",
			Path:   "/tmp/surface-artifacts/surface-morph-rendered-studio-shell.wasm",
			SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Size:   8604,
		},
		{
			Kind:   "compiler-owned-loader",
			Path:   "/tmp/surface-artifacts/surface-morph-rendered-studio-shell.mjs",
			SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			Size:   4939,
		},
		{
			Kind:   "runner-trace",
			Path:   "/tmp/surface-artifacts/surface-runner-trace.json",
			SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			Size:   1184,
		},
	}, cleanArtifactScan(3), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.Target != "wasm32-web" || report.Runtime != "surface-wasm32-web" ||
		!report.HostEvidence.BrowserCanvas ||
		!report.HostEvidence.BrowserInput {
		t.Fatalf(
			"target host evidence = target %q runtime %q host %#v, want browser-canvas wasm Morph evidence",
			report.Target,
			report.Runtime,
			report.HostEvidence,
		)
	}
	if report.RenderCommandStream.Renderer != "browser-canvas-rgba" ||
		report.RenderCommandStream.FrameChecksum != initialChecksum {
		t.Fatalf(
			"render_command_stream = %#v, want browser renderer and first browser canvas checksum %s",
			report.RenderCommandStream,
			initialChecksum,
		)
	}
	if report.Frames[0].ArtifactPath != initialArtifact || report.Frames[0].Producer != "app" ||
		report.Frames[0].EvidenceRole != "product_visual" {
		t.Fatalf(
			"browser Morph frame provenance = %#v, want app-produced product visual browser canvas artifact",
			report.Frames[0],
		)
	}

	const runtimeReportPath = "reports/surface/wasm-browser-canvas-morph-runtime.json"
	visual := morphRenderedBeautyVisualReportForTest(runtimeReportPath, report)
	visual.RequiredTargets = []string{"wasm32-web-browser-canvas"}
	visual.Apps[0].Targets[0].Target = "wasm32-web-browser-canvas"
	visual.Apps[0].Targets[0].Renderer = "browser-canvas-rgba"
	mrb, err := buildMorphRenderedBeautyReport(
		runtimeReportPath,
		report,
		visual,
		morphRenderedBeautyScenarioName(opt),
	)
	if err != nil {
		t.Fatalf("buildMorphRenderedBeautyReport failed: %v", err)
	}
	if err := surface.ValidateMorphRenderedBeautyReportValue(mrb); err != nil {
		t.Fatalf("ValidateMorphRenderedBeautyReportValue failed: %v", err)
	}
	if mrb.Target != "wasm32-web-browser-canvas" ||
		mrb.PixelEvidence.FrameArtifact != initialArtifact ||
		mrb.PixelEvidence.FrameChecksum != normalizePrefixedSHA256(initialChecksum) {
		t.Fatalf(
			"MRB target/pixel evidence = target %q pixel %#v, want browser canvas Morph pixel evidence",
			mrb.Target,
			mrb.PixelEvidence,
		)
	}
	if mrb.RendererStableProof.PixelOwner != "surface-renderer" ||
		!mrb.RendererStableProof.RendererOwned ||
		mrb.RendererStableProof.BridgeOwnedPixels ||
		!mrb.RendererStableProof.DerivedFromRenderCommandStream ||
		!mrb.RendererStableProof.StablePromotionEligible {
		t.Fatalf(
			("browser canvas renderer stable proof = %#v, want " +
				"renderer-owned command-stream-derived target proof"),
			mrb.RendererStableProof,
		)
	}
}

func TestLinuxX64RealWindowMorphScenarioUsesAppProducedPixelEvidence(t *testing.T) {
	const mode = "linux-x64-real-window-morph"
	const source = "examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra"
	if err := validateSmokeMode(mode); err != nil {
		t.Fatalf("validateSmokeMode(%s) failed: %v", mode, err)
	}

	dir := t.TempDir()
	initialFrame := renderMorphStudioShellFrameRGBA(320, 200, false)
	activeFrame := renderMorphStudioShellFrameRGBA(400, 240, true)
	initialArtifact := filepath.Join(dir, "surface-morph-real-window-frame-order-1.rgba")
	if err := os.WriteFile(initialArtifact, initialFrame.Pixels, 0o644); err != nil {
		t.Fatalf("write initial linux Morph frame artifact: %v", err)
	}
	activeArtifact := filepath.Join(dir, "surface-morph-real-window-frame-order-5.rgba")
	if err := os.WriteFile(activeArtifact, activeFrame.Pixels, 0o644); err != nil {
		t.Fatalf("write linux Morph frame artifact: %v", err)
	}
	initialChecksum := checksumRGBA(initialFrame.Pixels)
	activeChecksum := checksumRGBA(activeFrame.Pixels)

	scenario := runMorphScenarioForSource(source)
	opt := smokeOptions{
		Mode:       mode,
		SourcePath: source,
		ReportPath: filepath.Join(dir, "surface-linux-real-window-morph.json"),
	}
	frames := []surface.FrameReport{
		{
			Order:        1,
			Width:        320,
			Height:       200,
			Stride:       1280,
			Checksum:     initialChecksum,
			ArtifactPath: initialArtifact,
			Presented:    true,
		},
		{
			Order:        5,
			Width:        400,
			Height:       240,
			Stride:       1600,
			Checksum:     activeChecksum,
			ArtifactPath: activeArtifact,
			Presented:    true,
		},
	}
	if err := applyMorphTargetRuntimeFrameEvidence(opt, &scenario, frames); err != nil {
		t.Fatalf("applyMorphTargetRuntimeFrameEvidence: %v", err)
	}
	report := buildReport(opt, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/morph_flagship/surface_morph_rendered_studi" +
				"o_shell.tetra -o " +
				"/tmp/surface-artifacts/surface-morph-rendered-studio-shell"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-morph-rendered-studio-shell",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(0),
			ExpectedExitCode: intPtr(0),
		},
		{
			Name:             "surface linux-x64 Morph app-presented frame probe",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-morph-rendered-studio-shell-present-probe",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(-1),
			ExpectedExitCode: intPtr(-1),
		},
		{
			Name:             "surface linux-x64 real-window probe",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-morph-rendered-studio-shell-real-window-probe",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(42),
			ExpectedExitCode: intPtr(42),
		},
		{
			Name:     "surface linux-x64 real-window runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode linux-x64-real-window-morph",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, componentAppArtifacts(
		"/tmp/surface-artifacts/surface-morph-rendered-studio-shell",
	), cleanArtifactScan(
		2,
	), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.Target != "linux-x64" || report.Runtime != "surface-linux-x64" {
		t.Fatalf(
			"target/runtime = %s/%s, want linux-x64/surface-linux-x64",
			report.Target,
			report.Runtime,
		)
	}
	if report.HostEvidence.Level != "linux-x64-real-window" ||
		report.HostEvidence.Backend != "wayland-shm-rgba" ||
		!report.HostEvidence.Framebuffer ||
		!report.HostEvidence.RealWindow ||
		!report.HostEvidence.NativeInput {
		t.Fatalf(
			"host evidence = %#v, want linux-x64 real-window native-input evidence",
			report.HostEvidence,
		)
	}
	if report.RenderCommandStream == nil ||
		report.RenderCommandStream.Renderer != "wayland-shm-rgba" {
		t.Fatalf("render_command_stream = %#v, want wayland-shm-rgba", report.RenderCommandStream)
	}
	if len(report.Frames) != 2 {
		t.Fatalf(
			"frames = %#v, want initial and active Linux real-window Morph product frames",
			report.Frames,
		)
	}
	frame := report.Frames[1]
	if frame.Producer != "app" || frame.EvidenceRole != "product_visual" || frame.Precomputed ||
		frame.ArtifactPath != activeArtifact ||
		frame.Checksum != activeChecksum {
		t.Fatalf(
			"frame provenance = %#v, want app product_visual non-precomputed artifact evidence",
			frame,
		)
	}
	if report.BlockSystem != nil {
		t.Fatalf(
			"block_system = %#v, want nil for target-owned Morph product frame evidence",
			report.BlockSystem,
		)
	}
	if !caseNamesContain(
		report.Cases,
		"linux-x64 real-window Morph rendered beauty app frame readback",
	) {
		t.Fatalf("cases = %#v, want Linux Morph app frame readback evidence", report.Cases)
	}

	const runtimeReportPath = "reports/surface/linux-real-window-morph-runtime.json"
	visual := morphRenderedBeautyVisualReportForTest(runtimeReportPath, report)
	visual.RequiredTargets = []string{"linux-x64-real-window"}
	visual.Apps[0].Targets[0].Target = "linux-x64-real-window"
	visual.Apps[0].Targets[0].Renderer = "wayland-shm-rgba"
	mrb, err := buildMorphRenderedBeautyReport(
		runtimeReportPath,
		report,
		visual,
		morphRenderedBeautyScenarioName(opt),
	)
	if err != nil {
		t.Fatalf("buildMorphRenderedBeautyReport failed: %v", err)
	}
	if err := surface.ValidateMorphRenderedBeautyReportValue(mrb); err != nil {
		t.Fatalf("ValidateMorphRenderedBeautyReportValue failed: %v", err)
	}
	if mrb.Target != "linux-x64-real-window" ||
		mrb.PixelEvidence.FrameArtifact != initialArtifact ||
		mrb.PixelEvidence.FrameChecksum != normalizePrefixedSHA256(initialChecksum) {
		t.Fatalf(
			"MRB target/pixel evidence = target %q pixel %#v, want Linux real-window Morph pixel evidence",
			mrb.Target,
			mrb.PixelEvidence,
		)
	}
	if mrb.RendererStableProof.PixelOwner != "surface-renderer" ||
		!mrb.RendererStableProof.RendererOwned ||
		mrb.RendererStableProof.BridgeOwnedPixels ||
		!mrb.RendererStableProof.DerivedFromRenderCommandStream ||
		!mrb.RendererStableProof.StablePromotionEligible {
		t.Fatalf(
			"Linux renderer stable proof = %#v, want renderer-owned command-stream-derived target proof",
			mrb.RendererStableProof,
		)
	}
}

func TestBuildMorphRenderedBeautyReportRejectsMissingPixelGoldenFrame(t *testing.T) {
	const source = "examples/surface/morph_core/surface_morph_command_palette.tetra"
	scenario := runMorphScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless-morph",
		SourcePath: source,
	}, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/morph_core/surface_morph_command_palette.te" +
				"tra -o /tmp/surface-artifacts/surface-morph-command-palette"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-morph-command-palette",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(1),
			ExpectedExitCode: intPtr(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-morph",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, headlessSurfaceArtifacts(
		"/tmp/surface-artifacts/surface-morph-command-palette",
	), cleanArtifactScan(
		2,
	), scenario)
	visual := morphRenderedBeautyVisualReportForTest("reports/surface/morph-runtime.json", report)
	visual.Apps[0].Targets[0].Frames = nil

	_, err := buildMorphRenderedBeautyReport(
		"reports/surface/morph-runtime.json",
		report,
		visual,
		"headless-morph",
	)
	if err == nil {
		t.Fatalf("expected missing pixel golden frame to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "pixel golden") {
		t.Fatalf("error = %v, want pixel golden diagnostic", err)
	}
}

func morphRenderedBeautyVisualReportForTest(
	runtimeReportPath string,
	report surface.Report,
) surface.VisualRegressionReport {
	frameChecksum := normalizePrefixedSHA256(report.RenderCommandStream.FrameChecksum)
	goldenChecksum := "sha256:" + strings.Repeat("9", 64)
	return surface.VisualRegressionReport{
		Schema:          surface.VisualRegressionSchemaV1,
		Status:          "pass",
		GitHead:         report.Morph.GitHead,
		GoldenSet:       "surface-morph-rendered-beauty-v1",
		GoldenHash:      "sha256:" + strings.Repeat("8", 64),
		RequiredTargets: []string{"headless"},
		RequiredSources: []string{report.Source},
		Apps: []surface.VisualRegressionAppReport{{
			Name:         "surface-morph-command-palette",
			Source:       report.Source,
			ReferenceApp: true,
			Targets: []surface.VisualRegressionTargetReport{{
				Target:                "headless",
				RuntimeReport:         runtimeReportPath,
				RuntimeSchema:         surface.SchemaV1,
				GitHead:               report.Morph.GitHead,
				GoldenGitHead:         report.Morph.GitHead,
				Renderer:              report.RenderCommandStream.Renderer,
				BlockGraphEvidence:    true,
				TokenThemeEvidence:    true,
				LayoutEvidence:        true,
				AccessibilityEvidence: true,
				PerformanceEvidence:   true,
				Frames: []surface.VisualRegressionFrameReport{{
					Order:                 1,
					Label:                 "initial",
					Width:                 report.Frames[0].Width,
					Height:                report.Frames[0].Height,
					Stride:                report.Frames[0].Stride,
					Checksum:              frameChecksum,
					GoldenChecksum:        goldenChecksum,
					ArtifactPath:          morphRenderedBeautyFrameArtifactPathForTest(report),
					ArtifactSHA256:        frameChecksum,
					ArtifactFormat:        "rgba",
					GoldenArtifactPath:    "reports/surface/morph-rendered-beauty/headless/golden.rgba",
					GoldenArtifactSHA256:  goldenChecksum,
					DiffPixels:            0,
					DiffRatioMilli:        0,
					MaxChannelDelta:       0,
					TolerancePixels:       4,
					ToleranceRatioMilli:   1,
					ToleranceChannelDelta: 1,
					Pass:                  true,
				}},
			}},
		}},
		NegativeGuards: surface.VisualRegressionNegativeGuardsReport{
			ScreenshotOnlyRejected:           true,
			StaleGoldenRejected:              true,
			MajorDriftRejected:               true,
			MissingBlockGraphRejected:        true,
			MissingLayoutRejected:            true,
			MissingAccessibilityRejected:     true,
			MissingPerformanceRejected:       true,
			SelfGoldenRejected:               true,
			MetadataChecksumRejected:         true,
			FixtureFrameOnlyRejected:         true,
			MissingPNGOrRGBAArtifactRejected: true,
		},
	}
}

func morphRenderedBeautyFrameArtifactPathForTest(report surface.Report) string {
	if len(report.Frames) > 0 && strings.TrimSpace(report.Frames[0].ArtifactPath) != "" {
		return report.Frames[0].ArtifactPath
	}
	return "reports/surface/morph-rendered-beauty/headless/current.rgba"
}

func morphRecipeNamesContain(recipes []surface.MorphRecipeReport, want string) bool {
	for _, recipe := range recipes {
		if recipe.Name == want {
			return true
		}
	}
	return false
}

func blockSceneSnapshotHasNode(snapshot *surface.BlockSceneSnapshotReport, want string) bool {
	if snapshot == nil {
		return false
	}
	for _, node := range snapshot.Nodes {
		if node.Name == want || node.Recipe == want {
			return true
		}
	}
	return false
}

func renderCommandStreamHasCommand(commands []surface.RenderCommandReport, want string) bool {
	for _, command := range commands {
		if command.Command == want {
			return true
		}
	}
	return false
}

// ---- release_test.go ----

func releaseCounterTestEvidence(mode string) ([]surface.ProcessReport, []surface.ArtifactReport) {
	switch mode {
	case "wasm32-web-browser-canvas":
		return []surface.ProcessReport{
			{
				Name: "tetra build",
				Kind: "build",
				Path: ("tetra build --target wasm32-web " +
					"examples/surface/release/surface_release_counter.tetra -o " +
					"/tmp/surface-artifacts/surface-browser-counter.wasm"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web browser canvas component app",
				Kind: "app",
				Path: ("/usr/bin/chromium --headless " +
					"<surface-browser-canvas-runner> " +
					"wasm=/tmp/surface-artifacts/surface-browser-counter.wasm"),
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(0),
				ExpectedExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web import validator",
				Kind: "runtime",
				Path: ("go run ./tools/cmd/validate-wasm-imports --target " +
					"wasm32-web " +
					"/tmp/surface-artifacts/surface-browser-counter.wasm"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name:     "surface wasm32-web browser canvas runtime",
				Kind:     "runtime",
				Path:     "Chromium test fixture",
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web browser canvas trace",
				Kind: "runtime",
				Path: ("/usr/bin/chromium --headless --dump-dom http://127.0.0.1:" +
					"1/surface-browser-canvas-runner"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		}, wasmBrowserCanvasSurfaceArtifacts()
	default:
		return []surface.ProcessReport{
			{
				Name: "tetra build",
				Kind: "build",
				Path: ("tetra build --target linux-x64 " +
					"examples/surface/release/surface_release_counter.tetra -o " +
					"/tmp/surface-artifacts/surface-window-counter"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name:             "surface component app",
				Kind:             "app",
				Path:             "/tmp/surface-artifacts/surface-window-counter",
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(1),
				ExpectedExitCode: intPtr(1),
			},
			{
				Name:             "surface linux-x64 real-window probe",
				Kind:             "app",
				Path:             "/tmp/surface-artifacts/surface-real-window-probe",
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(42),
				ExpectedExitCode: intPtr(42),
			},
			{
				Name:     "surface linux-x64 runtime",
				Kind:     "runtime",
				Path:     "tools/cmd/surface-runtime-smoke",
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		}, componentAppArtifacts("/tmp/surface-artifacts/surface-window-counter")
	}
}

func releaseWindowTestProcesses() []surface.ProcessReport {
	return []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/release/surface_release_form.tetra -o " +
				"/tmp/surface-artifacts/surface-release-form"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-release-form",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(1),
			ExpectedExitCode: intPtr(1),
		},
		{
			Name:             "surface linux-x64 real-window probe",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-release-window-probe",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(42),
			ExpectedExitCode: intPtr(42),
		},
		{
			Name:     "surface linux-x64 release clipboard harness",
			Kind:     "runtime",
			Path:     "/tmp/surface-artifacts/surface-linux-clipboard-harness.json",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:     "surface linux-x64 release composition harness",
			Kind:     "runtime",
			Path:     "/tmp/surface-artifacts/surface-linux-composition-harness.json",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:     "surface linux accessibility host bridge",
			Kind:     "runtime",
			Path:     "/tmp/surface-artifacts/surface-linux-accessibility-bridge.json",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:     "surface linux accessibility platform probe",
			Kind:     "runtime",
			Path:     "/tmp/surface-artifacts/surface-linux-accessibility-probe.json",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:     "surface linux-x64 runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode linux-x64-release-window",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}
}

func releaseWindowTestArtifacts() []surface.ArtifactReport {
	return []surface.ArtifactReport{
		{
			Kind:   "component-app",
			Path:   "/tmp/surface-artifacts/surface-release-form",
			SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Size:   90001,
		},
		{
			Kind:   "linux-accessibility-host-bridge",
			Path:   "/tmp/surface-artifacts/surface-linux-accessibility-bridge.json",
			SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			Size:   4096,
		},
		{
			Kind:   "linux-accessibility-platform-probe",
			Path:   "/tmp/surface-artifacts/surface-linux-accessibility-probe.json",
			SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			Size:   4096,
		},
	}
}

func releaseWindowTestFrames() []surface.FrameReport {
	before := renderReleaseToolkitFrameRGBA(0, 0, -1, 0, 0, 0, false, 0, 320, 240)
	after := renderReleaseToolkitFrameRGBA(0, 0, 7, 1, 1, 2, true, 16, 560, 420)
	return []surface.FrameReport{
		{
			Order:     1,
			Width:     before.Width,
			Height:    before.Height,
			Stride:    before.Stride,
			Checksum:  checksumRGBA(before.Pixels),
			Presented: true,
		},
		{
			Order:     5,
			Width:     after.Width,
			Height:    after.Height,
			Stride:    after.Stride,
			Checksum:  checksumRGBA(after.Pixels),
			Presented: true,
		},
	}
}

func releaseBrowserTestProcesses() []surface.ProcessReport {
	return []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target wasm32-web " +
				"examples/surface/release/surface_release_form.tetra -o " +
				"/tmp/surface-artifacts/surface-release-form.wasm"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name: "surface wasm32-web browser canvas component app",
			Kind: "app",
			Path: ("/usr/bin/chromium --headless " +
				"<surface-browser-canvas-runner> scenario=release-browser " +
				"wasm=/tmp/surface-artifacts/surface-release-form.wasm"),
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(0),
			ExpectedExitCode: intPtr(0),
		},
		{
			Name: "surface wasm32-web import validator",
			Kind: "runtime",
			Path: ("go run ./tools/cmd/validate-wasm-imports --target " +
				"wasm32-web /tmp/surface-artifacts/surface-release-form.wasm"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:     "surface wasm32-web browser canvas runtime",
			Kind:     "runtime",
			Path:     "Chromium release browser fixture",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name: "surface wasm32-web browser canvas trace",
			Kind: "runtime",
			Path: ("/usr/bin/chromium --headless --dump-dom http://127.0.0.1:" +
				"1/surface-browser-canvas-runner?scenario=release-browser"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}
}

func releaseBrowserTestArtifacts() []surface.ArtifactReport {
	return []surface.ArtifactReport{
		{
			Kind:   "component-app",
			Path:   "/tmp/surface-artifacts/surface-release-form.wasm",
			SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Size:   9604,
		},
		{
			Kind:   "compiler-owned-loader",
			Path:   "/tmp/surface-artifacts/surface-release-form.mjs",
			SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			Size:   4939,
		},
		{
			Kind:   "runner-trace",
			Path:   "/tmp/surface-artifacts/surface-runner-trace.json",
			SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			Size:   4096,
		},
	}
}

func releaseBrowserTestFrames() []surface.FrameReport {
	before := renderReleaseToolkitFrameRGBA(0, 0, -1, 0, 0, 0, false, 0, 320, 240)
	after := renderReleaseToolkitFrameRGBA(0, 0, 7, 1, 1, 2, true, 16, 560, 420)
	return []surface.FrameReport{
		{
			Order:     1,
			Width:     before.Width,
			Height:    before.Height,
			Stride:    before.Stride,
			Checksum:  checksumRGBA(before.Pixels),
			Presented: true,
		},
		{
			Order:     5,
			Width:     after.Width,
			Height:    after.Height,
			Stride:    after.Stride,
			Checksum:  checksumRGBA(after.Pixels),
			Presented: true,
		},
	}
}

func releaseAccessibilityTestFrames(mode string) []surface.FrameReport {
	if mode != "wasm32-web-release-accessibility" {
		return nil
	}
	before := renderAccessibilityMetadataFrameRGBA(0, 0, -1, 0, 0, 0, 320, 240)
	after := renderAccessibilityMetadataFrameRGBA(0, 0, 5, 1, 1, 2, 480, 320)
	return []surface.FrameReport{
		{
			Order:     1,
			Width:     before.Width,
			Height:    before.Height,
			Stride:    before.Stride,
			Checksum:  checksumRGBA(before.Pixels),
			Presented: true,
		},
		{
			Order:     5,
			Width:     after.Width,
			Height:    after.Height,
			Stride:    after.Stride,
			Checksum:  checksumRGBA(after.Pixels),
			Presented: true,
		},
	}
}

func TestCollectWASM32WebProcessEvidenceRecordsPresentedFrameTrace(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required for wasm32-web Surface runner evidence")
	}
	source, err := resolveSurfaceSourcePath("examples/surface/runtime/surface_counter.tetra")
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
		t.Fatalf(
			"artifact scan = %#v, want wasm, loader, and runner trace checked",
			evidence.ArtifactScan,
		)
	}
	after := evidence.Frames[len(evidence.Frames)-1]
	if after.Order != 4 || after.Width != 320 || after.Height != 200 || after.Stride != 1280 ||
		!after.Presented {
		t.Fatalf("last wasm frame = %#v, want order-4 320x200 presented frame", after)
	}
	wantFrame := renderCounterFrameRGBA(1, true)
	if want := checksumRGBA(wantFrame.Pixels); after.Checksum != want {
		t.Fatalf(
			"last wasm frame checksum = %s, want actual CounterApp checksum %s",
			after.Checksum,
			want,
		)
	}
}

func textFocusInputTestProcesses(mode string) []surface.ProcessReport {
	switch mode {
	case "linux-x64-real-window-text-focus-input":
		return []surface.ProcessReport{
			{
				Name: "tetra build",
				Kind: "build",
				Path: ("tetra build --target linux-x64 " +
					"examples/surface/runtime/surface_textbox_app.tetra -o " +
					"/tmp/surface-artifacts/surface-textbox-app"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name:             "surface component app",
				Kind:             "app",
				Path:             "/tmp/surface-artifacts/surface-textbox-app",
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(1),
				ExpectedExitCode: intPtr(1),
			},
			{
				Name:             "surface linux-x64 real-window probe",
				Kind:             "app",
				Path:             "/tmp/surface-artifacts/surface-real-window-probe",
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(42),
				ExpectedExitCode: intPtr(42),
			},
			{
				Name:     "surface linux-x64 runtime",
				Kind:     "runtime",
				Path:     "tools/cmd/surface-runtime-smoke",
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		}
	case "wasm32-web-browser-canvas-text-focus-input":
		return []surface.ProcessReport{
			{
				Name: "tetra build",
				Kind: "build",
				Path: ("tetra build --target wasm32-web " +
					"examples/surface/runtime/surface_textbox_app.tetra -o " +
					"/tmp/surface-artifacts/surface-textbox-app.wasm"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web browser canvas component app",
				Kind: "app",
				Path: ("/usr/bin/chromium --headless " +
					"<surface-browser-canvas-runner> " +
					"wasm=/tmp/surface-artifacts/surface-textbox-app.wasm"),
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(0),
				ExpectedExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web import validator",
				Kind: "runtime",
				Path: ("go run ./tools/cmd/validate-wasm-imports --target " +
					"wasm32-web /tmp/surface-artifacts/surface-textbox-app.wasm"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name:     "surface wasm32-web browser canvas runtime",
				Kind:     "runtime",
				Path:     "Chromium test fixture",
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web browser canvas trace",
				Kind: "runtime",
				Path: ("/usr/bin/chromium --headless --dump-dom http://127.0.0.1:" +
					"1/surface-browser-canvas-runner?scenario=text-focus-input"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		}
	default:
		return []surface.ProcessReport{
			{
				Name: "tetra build",
				Kind: "build",
				Path: ("tetra build --target linux-x64 " +
					"examples/surface/runtime/surface_textbox_app.tetra -o " +
					"/tmp/surface-artifacts/surface-textbox-app"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name:             "surface component app",
				Kind:             "app",
				Path:             "/tmp/surface-artifacts/surface-textbox-app",
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(1),
				ExpectedExitCode: intPtr(1),
			},
			{
				Name:     "surface headless runtime",
				Kind:     "runtime",
				Path:     "tools/cmd/surface-runtime-smoke",
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		}
	}
}

func textFocusInputTestArtifacts(mode string) []surface.ArtifactReport {
	switch mode {
	case "wasm32-web-browser-canvas-text-focus-input":
		return []surface.ArtifactReport{
			{
				Kind:   "component-app",
				Path:   "/tmp/surface-artifacts/surface-textbox-app.wasm",
				SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Size:   8604,
			},
			{
				Kind:   "compiler-owned-loader",
				Path:   "/tmp/surface-artifacts/surface-textbox-app.mjs",
				SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				Size:   4939,
			},
			{
				Kind:   "runner-trace",
				Path:   "/tmp/surface-artifacts/surface-runner-trace.json",
				SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
				Size:   1184,
			},
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
		return []surface.FrameReport{
			{
				Order:     5,
				Width:     frame.Width,
				Height:    frame.Height,
				Stride:    frame.Stride,
				Checksum:  checksumRGBA(frame.Pixels),
				Presented: true,
			},
		}
	case "wasm32-web-browser-canvas-text-focus-input":
		before := renderTextFocusInputFrameRGBA(0, 0, 0, 320, 200)
		after := renderTextFocusInputFrameRGBA(1, 1, 1, 400, 240)
		return []surface.FrameReport{
			{
				Order:     1,
				Width:     before.Width,
				Height:    before.Height,
				Stride:    before.Stride,
				Checksum:  checksumRGBA(before.Pixels),
				Presented: true,
			},
			{
				Order:     5,
				Width:     after.Width,
				Height:    after.Height,
				Stride:    after.Stride,
				Checksum:  checksumRGBA(after.Pixels),
				Presented: true,
			},
		}
	default:
		return nil
	}
}

func releaseTextInputTestProcesses(mode string) []surface.ProcessReport {
	switch mode {
	case "linux-x64-release-text-input":
		return []surface.ProcessReport{
			{
				Name: "tetra build",
				Kind: "build",
				Path: ("tetra build --target linux-x64 " +
					"examples/surface/release/surface_release_text_input.tetra " +
					"-o /tmp/surface-artifacts/surface-release-text-input"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name:             "surface component app",
				Kind:             "app",
				Path:             "/tmp/surface-artifacts/surface-release-text-input",
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(1),
				ExpectedExitCode: intPtr(1),
			},
			{
				Name:             "surface linux-x64 real-window probe",
				Kind:             "app",
				Path:             "/tmp/surface-artifacts/surface-real-window-probe",
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(42),
				ExpectedExitCode: intPtr(42),
			},
			{
				Name:     "surface linux-x64 runtime",
				Kind:     "runtime",
				Path:     "tools/cmd/surface-runtime-smoke --mode linux-x64-release-text-input",
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		}
	case "wasm32-web-release-text-input":
		return []surface.ProcessReport{
			{
				Name: "tetra build",
				Kind: "build",
				Path: ("tetra build --target wasm32-web " +
					"examples/surface/release/surface_release_text_input.tetra " +
					"-o /tmp/surface-artifacts/surface-release-text-input.wasm"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web browser canvas component app",
				Kind: "app",
				Path: ("/usr/bin/chromium --headless " +
					"<surface-browser-canvas-runner> " +
					"wasm=/tmp/surface-artifacts/surface-release-text-input.wasm"),
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(0),
				ExpectedExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web import validator",
				Kind: "runtime",
				Path: ("go run ./tools/cmd/validate-wasm-imports --target " +
					"wasm32-web " +
					"/tmp/surface-artifacts/surface-release-text-input.wasm"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name:     "surface wasm32-web browser canvas runtime",
				Kind:     "runtime",
				Path:     "tools/cmd/surface-runtime-smoke --mode wasm32-web-release-text-input",
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		}
	default:
		return []surface.ProcessReport{
			{
				Name: "tetra build",
				Kind: "build",
				Path: ("tetra build --target linux-x64 " +
					"examples/surface/release/surface_release_text_input.tetra " +
					"-o /tmp/surface-artifacts/surface-release-text-input"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name:             "surface component app",
				Kind:             "app",
				Path:             "/tmp/surface-artifacts/surface-release-text-input",
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(1),
				ExpectedExitCode: intPtr(1),
			},
			{
				Name:     "surface headless runtime",
				Kind:     "runtime",
				Path:     "tools/cmd/surface-runtime-smoke --mode headless-release-text-input",
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		}
	}
}

func releaseTextInputTestArtifacts(mode string) []surface.ArtifactReport {
	switch mode {
	case "wasm32-web-release-text-input":
		return []surface.ArtifactReport{
			{
				Kind:   "component-app",
				Path:   "/tmp/surface-artifacts/surface-release-text-input.wasm",
				SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Size:   8604,
			},
			{
				Kind:   "compiler-owned-loader",
				Path:   "/tmp/surface-artifacts/surface-release-text-input.mjs",
				SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				Size:   4939,
			},
			{
				Kind:   "runner-trace",
				Path:   "/tmp/surface-artifacts/surface-runner-trace.json",
				SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
				Size:   1184,
			},
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
			{
				Name: "tetra build",
				Kind: "build",
				Path: ("tetra build --target linux-x64 " +
					"examples/surface/release/surface_release_accessibility.tetra" +
					" -o /tmp/surface-artifacts/surface-release-accessibility"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name:             "surface component app",
				Kind:             "app",
				Path:             "/tmp/surface-artifacts/surface-release-accessibility",
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(1),
				ExpectedExitCode: intPtr(1),
			},
			{
				Name:             "surface linux-x64 real-window probe",
				Kind:             "app",
				Path:             "/tmp/surface-artifacts/surface-accessibility-real-window-probe",
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(42),
				ExpectedExitCode: intPtr(42),
			},
			{
				Name:     "surface linux-x64 runtime",
				Kind:     "runtime",
				Path:     "tools/cmd/surface-runtime-smoke --mode linux-x64-release-accessibility",
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name:     "surface linux accessibility host bridge",
				Kind:     "runtime",
				Path:     "/tmp/surface-artifacts/surface-linux-accessibility-bridge.json",
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name:     "surface linux accessibility platform probe",
				Kind:     "runtime",
				Path:     "/tmp/surface-artifacts/surface-linux-accessibility-probe.json",
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		}
	case "wasm32-web-release-accessibility":
		return []surface.ProcessReport{
			{
				Name: "tetra build",
				Kind: "build",
				Path: ("tetra build --target wasm32-web " +
					"examples/surface/release/surface_release_accessibility.tetra" +
					" -o " +
					"/tmp/surface-artifacts/surface-release-accessibility.wasm"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web browser canvas component app",
				Kind: "app",
				Path: ("/usr/bin/chromium --headless " +
					"<surface-browser-canvas-runner> " +
					"scenario=release-accessibility " +
					"wasm=/tmp/surface-artifacts/surface-release-accessibility.wa" +
					"sm"),
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(0),
				ExpectedExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web import validator",
				Kind: "runtime",
				Path: ("go run ./tools/cmd/validate-wasm-imports --target " +
					"wasm32-web " +
					"/tmp/surface-artifacts/surface-release-accessibility.wasm"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name:     "surface wasm32-web browser canvas runtime",
				Kind:     "runtime",
				Path:     "tools/cmd/surface-runtime-smoke --mode wasm32-web-release-accessibility",
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web browser canvas trace",
				Kind: "runtime",
				Path: ("/usr/bin/chromium --headless --dump-dom http://127.0.0.1:" +
					"1/surface-browser-canvas-runner?scenario=release-accessibili" +
					"ty"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		}
	default:
		return []surface.ProcessReport{
			{
				Name: "tetra build",
				Kind: "build",
				Path: ("tetra build --target linux-x64 " +
					"examples/surface/release/surface_release_accessibility.tetra" +
					" -o /tmp/surface-artifacts/surface-release-accessibility"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name:             "surface component app",
				Kind:             "app",
				Path:             "/tmp/surface-artifacts/surface-release-accessibility",
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(1),
				ExpectedExitCode: intPtr(1),
			},
			{
				Name:     "surface headless runtime",
				Kind:     "runtime",
				Path:     "tools/cmd/surface-runtime-smoke --mode headless-release-accessibility",
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		}
	}
}

func releaseAccessibilityTestArtifacts(mode string) []surface.ArtifactReport {
	switch mode {
	case "linux-x64-release-accessibility":
		artifacts := componentAppArtifacts("/tmp/surface-artifacts/surface-release-accessibility")
		return append(
			artifacts,
			surface.ArtifactReport{
				Kind:   "linux-accessibility-host-bridge",
				Path:   "/tmp/surface-artifacts/surface-linux-accessibility-bridge.json",
				SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				Size:   4096,
			},
			surface.ArtifactReport{
				Kind:   "linux-accessibility-platform-probe",
				Path:   "/tmp/surface-artifacts/surface-linux-accessibility-probe.json",
				SHA256: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
				Size:   4096,
			},
		)
	case "wasm32-web-release-accessibility":
		return []surface.ArtifactReport{
			{
				Kind:   "component-app",
				Path:   "/tmp/surface-artifacts/surface-release-accessibility.wasm",
				SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Size:   9004,
			},
			{
				Kind:   "compiler-owned-loader",
				Path:   "/tmp/surface-artifacts/surface-release-accessibility.mjs",
				SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				Size:   4939,
			},
			{
				Kind:   "runner-trace",
				Path:   "/tmp/surface-artifacts/surface-runner-trace.json",
				SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
				Size:   4096,
			},
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
		{
			Name:          "release text input invalid UTF-8 rejected",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "invalid utf8 rejected",
		},
		{Name: "release text input multiline storage", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input caret home end arrows", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input selection replacement", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "release text input selection clipboard transfer",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{Name: "release text input backspace delete", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "release text input clipboard owned copy transfer",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name: "release text input composition start update",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{Name: "release text input composition commit", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input composition cancel", Kind: "positive", Ran: true, Pass: true},
		{Name: "release text input shaping plan scoped", Kind: "positive", Ran: true, Pass: true},
		{Name: "settings reference text input trace", Kind: "positive", Ran: true, Pass: true},
		{Name: "editor reference text input trace", Kind: "positive", Ran: true, Pass: true},
		{
			Name: "release text input safe view lifetime checked",
			Kind: "positive",
			Ran:  true,
			Pass: true,
		},
		{
			Name:          "reject legacy UI evidence",
			Kind:          "negative",
			Ran:           true,
			Pass:          true,
			ExpectedError: "legacy UI evidence rejected",
		},
	}
}

func componentTreeTestProcesses(mode string) []surface.ProcessReport {
	switch mode {
	case "linux-x64-real-window-component-tree", "linux-x64-real-window-component-tree-api":
		return []surface.ProcessReport{
			{
				Name: "tetra build",
				Kind: "build",
				Path: ("tetra build --target linux-x64 " +
					"examples/surface/toolkit/surface_tree_app.tetra -o " +
					"/tmp/surface-artifacts/surface-tree-app"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name:             "surface component app",
				Kind:             "app",
				Path:             "/tmp/surface-artifacts/surface-tree-app",
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(1),
				ExpectedExitCode: intPtr(1),
			},
			{
				Name:             "surface linux-x64 real-window probe",
				Kind:             "app",
				Path:             "/tmp/surface-artifacts/surface-real-window-probe",
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(42),
				ExpectedExitCode: intPtr(42),
			},
			{
				Name:     "surface linux-x64 runtime",
				Kind:     "runtime",
				Path:     "tools/cmd/surface-runtime-smoke",
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		}
	case "wasm32-web-browser-canvas-component-tree", "wasm32-web-browser-canvas-component-tree-api":
		return []surface.ProcessReport{
			{
				Name: "tetra build",
				Kind: "build",
				Path: ("tetra build --target wasm32-web " +
					"examples/surface/toolkit/surface_tree_app.tetra -o " +
					"/tmp/surface-artifacts/surface-tree-app.wasm"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web browser canvas component app",
				Kind: "app",
				Path: ("/usr/bin/chromium --headless " +
					"<surface-browser-canvas-runner> scenario=component-tree " +
					"wasm=/tmp/surface-artifacts/surface-tree-app.wasm"),
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(0),
				ExpectedExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web import validator",
				Kind: "runtime",
				Path: ("go run ./tools/cmd/validate-wasm-imports --target " +
					"wasm32-web /tmp/surface-artifacts/surface-tree-app.wasm"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name:     "surface wasm32-web browser canvas runtime",
				Kind:     "runtime",
				Path:     "Chromium test fixture",
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name: "surface wasm32-web browser canvas trace",
				Kind: "runtime",
				Path: ("/usr/bin/chromium --headless --dump-dom http://127.0.0.1:" +
					"1/surface-browser-canvas-runner?scenario=component-tree"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		}
	default:
		return []surface.ProcessReport{
			{
				Name: "tetra build",
				Kind: "build",
				Path: ("tetra build --target linux-x64 " +
					"examples/surface/toolkit/surface_tree_app.tetra -o " +
					"/tmp/surface-artifacts/surface-tree-app"),
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
			{
				Name:             "surface component app",
				Kind:             "app",
				Path:             "/tmp/surface-artifacts/surface-tree-app",
				Ran:              true,
				Pass:             true,
				ExitCode:         intPtr(1),
				ExpectedExitCode: intPtr(1),
			},
			{
				Name:     "surface headless runtime",
				Kind:     "runtime",
				Path:     "tools/cmd/surface-runtime-smoke",
				Ran:      true,
				Pass:     true,
				ExitCode: intPtr(0),
			},
		}
	}
}

func componentTreeTestArtifacts(mode string) []surface.ArtifactReport {
	switch mode {
	case "wasm32-web-browser-canvas-component-tree", "wasm32-web-browser-canvas-component-tree-api":
		return []surface.ArtifactReport{
			{
				Kind:   "component-app",
				Path:   "/tmp/surface-artifacts/surface-tree-app.wasm",
				SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Size:   9004,
			},
			{
				Kind:   "compiler-owned-loader",
				Path:   "/tmp/surface-artifacts/surface-tree-app.mjs",
				SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				Size:   4939,
			},
			{
				Kind:   "runner-trace",
				Path:   "/tmp/surface-artifacts/surface-runner-trace.json",
				SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
				Size:   22184,
			},
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
		return []surface.FrameReport{
			{
				Order:     5,
				Width:     frame.Width,
				Height:    frame.Height,
				Stride:    frame.Stride,
				Checksum:  checksumRGBA(frame.Pixels),
				Presented: true,
			},
		}
	case "wasm32-web-browser-canvas-component-tree", "wasm32-web-browser-canvas-component-tree-api":
		before := renderComponentTreeFrameRGBA(0, 0, -1, 0, 0, 320, 200)
		after := renderComponentTreeFrameRGBA(0, 0, 6, 1, 1, 400, 240)
		return []surface.FrameReport{
			{
				Order:     1,
				Width:     before.Width,
				Height:    before.Height,
				Stride:    before.Stride,
				Checksum:  checksumRGBA(before.Pixels),
				Presented: true,
			},
			{
				Order:     5,
				Width:     after.Width,
				Height:    after.Height,
				Stride:    after.Stride,
				Checksum:  checksumRGBA(after.Pixels),
				Presented: true,
			},
		}
	default:
		return nil
	}
}

func componentTreeDispatchPathsContain(
	paths []surface.ComponentTreeDispatchPathReport,
	want []int,
) bool {
	for _, path := range paths {
		if intSlicesEqual(path.Path, want) {
			return true
		}
	}
	return false
}

// ---- target_modes_test.go ----

func TestBlockSystemScenarioProducesMemoryBudgetEvidence(t *testing.T) {
	scenario := runBlockSystemScenario()
	report := buildReport(smokeOptions{
		Mode:       "headless-block-system",
		SourcePath: "examples/surface/block_core/surface_block_system.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/block_core/surface_block_system.tetra -o " +
				"/tmp/surface-artifacts/surface-block-system"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-block-system",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(1),
			ExpectedExitCode: intPtr(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-block-system",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, headlessSurfaceArtifacts(
		"/tmp/surface-artifacts/surface-block-system",
	), cleanArtifactScan(
		2,
	), scenario)

	if report.BlockSystem == nil || report.BlockSystem.MemoryBudget == nil {
		t.Fatalf("block_system.memory_budget is nil, want bounded memory/cache evidence")
	}
	budget := report.BlockSystem.MemoryBudget
	if budget.BlockCount != len(report.Components) || budget.StressBlockCount < 128 {
		t.Fatalf("memory budget block counts = %#v, components=%d", budget, len(report.Components))
	}
	if budget.PeakFramebufferBytes != 256000 || budget.TotalFramebufferBytes != 768000 ||
		budget.FramebufferBudgetBytes < budget.PeakFramebufferBytes {
		t.Fatalf("memory budget framebuffer bytes = %#v", budget)
	}
	if !budget.BoundedCaches || !budget.UnboundedCacheRejected ||
		budget.PerformanceClaim != "none" {
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
	if got := defaultSurfaceSourcePath(
		smokeOptions{Mode: mode, SourcePath: "examples/surface/runtime/surface_counter.tetra"},
	); got != "examples/surface/block_core/surface_block_system.tetra" {
		t.Fatalf(
			"defaultSurfaceSourcePath(%s) = %q, want examples/surface/block_core/surface_block_system.tetra",
			mode,
			got,
		)
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
	markHostProbeOnlyFrameEvidence(
		&hostProbeFrame,
		"/tmp/surface-artifacts/surface-block-system-real-window-frame.rgba",
	)
	scenario.Frames = append(scenario.Frames, hostProbeFrame)
	scenario.BlockSystem = blockSystemReportForLinuxX64RealWindowScenario(
		"examples/surface/block_core/surface_block_system.tetra",
		scenario.Frames,
	)
	attachBlockSystemMemoryBudget(&scenario)
	report := buildReport(smokeOptions{
		Mode:       mode,
		SourcePath: "examples/surface/block_core/surface_block_system.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/block_core/surface_block_system.tetra -o " +
				"/tmp/surface-artifacts/surface-block-system"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-block-system",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(1),
			ExpectedExitCode: intPtr(1),
		},
		{
			Name:             "surface linux-x64 real-window probe",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-block-system-real-window-probe",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(42),
			ExpectedExitCode: intPtr(42),
		},
		{
			Name:     "surface linux-x64 runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode linux-x64-real-window-block-system",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, componentAppArtifacts(
		"/tmp/surface-artifacts/surface-block-system",
	), cleanArtifactScan(
		1,
	), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.Target != "linux-x64" || report.Runtime != "surface-linux-x64" {
		t.Fatalf(
			"target/runtime = %s/%s, want linux-x64/surface-linux-x64",
			report.Target,
			report.Runtime,
		)
	}
	if report.HostEvidence.Level != "linux-x64-real-window" || !report.HostEvidence.RealWindow ||
		!report.HostEvidence.NativeInput {
		t.Fatalf(
			"host evidence = %#v, want linux-x64 real-window native-input evidence",
			report.HostEvidence,
		)
	}
	if report.BlockSystem == nil {
		t.Fatalf("block_system is nil, want linux-x64 real-window Block evidence")
	}
	if report.BlockSystem.QualityLevel != "linux-x64-real-window-block-system-v1" ||
		report.BlockSystem.Renderer != "wayland-shm-rgba" {
		t.Fatalf(
			"block_system = %#v, want linux-x64 real-window quality and renderer",
			report.BlockSystem,
		)
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
	if blockWindowFrame == nil || runtimeWindowFrame == nil ||
		blockWindowFrame.Checksum != runtimeWindowFrame.Checksum {
		t.Fatalf(
			"block_system frames = %#v, report frames = %#v",
			report.BlockSystem.Frames,
			report.Frames,
		)
	}
	if !runtimeWindowFrame.Precomputed || runtimeWindowFrame.EvidenceRole != "host_probe_only" ||
		runtimeWindowFrame.Producer != "host_probe" {
		t.Fatalf(
			("runtime real-window frame provenance = %#v, want " +
				"host_probe_only precomputed infrastructure evidence"),
			runtimeWindowFrame,
		)
	}
	if !blockWindowFrame.Precomputed ||
		blockWindowFrame.EvidenceRole != runtimeWindowFrame.EvidenceRole ||
		blockWindowFrame.Producer != runtimeWindowFrame.Producer {
		t.Fatalf(
			"block_system real-window frame provenance = %#v, want runtime provenance %#v",
			blockWindowFrame,
			runtimeWindowFrame,
		)
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
		SourcePath: "examples/surface/runtime/surface_counter.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/runtime/surface_counter.tetra -o " +
				"/tmp/surface-artifacts/surface-counter"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-counter",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(1),
			ExpectedExitCode: intPtr(1),
		},
		{
			Name:     "surface linux-x64 host probe build",
			Kind:     "build",
			Path:     "/tmp/tetra build probe",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface linux-x64 host probe",
			Kind:             "app",
			Path:             "/tmp/surface-host-probe",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(42),
			ExpectedExitCode: intPtr(42),
		},
		{
			Name:     "surface linux-x64 event sequence probe build",
			Kind:     "build",
			Path:     "/tmp/tetra build event sequence probe",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface linux-x64 event sequence probe",
			Kind:             "app",
			Path:             "/tmp/surface-event-sequence-probe",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(42),
			ExpectedExitCode: intPtr(42),
		},
		{
			Name:             "surface linux-x64 counter app presented frame probe",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-counter-present-probe",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(-1),
			ExpectedExitCode: intPtr(-1),
		},
		{
			Name:     "surface linux-x64 runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, componentAppArtifacts("/tmp/surface-artifacts/surface-counter"), cleanArtifactScan(1), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.HostEvidence.Level != "linux-x64-memfd-starter" ||
		report.HostEvidence.Backend != "memfd-rgba" ||
		!report.HostEvidence.Framebuffer ||
		report.HostEvidence.RealWindow ||
		report.HostEvidence.NativeInput {
		t.Fatalf(
			("linux-x64 host evidence = %#v, want explicit memfd starter " +
				"evidence without real-window/native-input claim"),
			report.HostEvidence,
		)
	}
	if !caseNamesContain(scenario.Cases, "linux-x64 Surface Host ABI") {
		t.Fatalf(
			"linux-x64 scenario cases = %#v, want target-specific Surface Host ABI evidence",
			scenario.Cases,
		)
	}
	if !caseNamesContain(scenario.Cases, "no legacy UI sidecar artifacts") {
		t.Fatalf("linux-x64 scenario cases = %#v, want no legacy sidecar evidence", scenario.Cases)
	}
	if !caseNamesContain(scenario.Cases, "linux-x64 app-presented RGBA checksum") {
		t.Fatalf(
			"linux-x64 scenario cases = %#v, want app-presented frame checksum evidence",
			scenario.Cases,
		)
	}
	if !caseNamesContain(scenario.Cases, "linux-x64 host event sequence") {
		t.Fatalf(
			"linux-x64 scenario cases = %#v, want executable event sequence evidence",
			scenario.Cases,
		)
	}
	if !caseNamesContain(scenario.Cases, "linux-x64 counter component app-presented frame") {
		t.Fatalf(
			"linux-x64 scenario cases = %#v, want counter component app-presented frame evidence",
			scenario.Cases,
		)
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
		SourcePath: "examples/surface/runtime/surface_window_counter.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/runtime/surface_window_counter.tetra -o " +
				"/tmp/surface-artifacts/surface-window-counter"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-window-counter",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(1),
			ExpectedExitCode: intPtr(1),
		},
		{
			Name:             "surface linux-x64 real-window probe",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-real-window-probe",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(42),
			ExpectedExitCode: intPtr(42),
		},
		{
			Name:     "surface linux-x64 runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, componentAppArtifacts(
		"/tmp/surface-artifacts/surface-window-counter",
	), cleanArtifactScan(
		1,
	), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.HostEvidence.Level != "linux-x64-real-window" ||
		report.HostEvidence.Backend != "wayland-shm-rgba" ||
		!report.HostEvidence.Framebuffer ||
		!report.HostEvidence.RealWindow ||
		!report.HostEvidence.NativeInput {
		t.Fatalf(
			"linux-x64 real-window host evidence = %#v, want explicit real-window/native-input evidence",
			report.HostEvidence,
		)
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
			t.Fatalf(
				"linux-x64 real-window events = %#v, want %s event evidence",
				scenario.Events,
				want,
			)
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
	scenario.Frames = append(
		scenario.Frames,
		surface.FrameReport{
			Order:     3,
			Width:     beforeFrame.Width,
			Height:    beforeFrame.Height,
			Stride:    beforeFrame.Stride,
			Checksum:  checksumRGBA(beforeFrame.Pixels),
			Presented: true,
		},
		surface.FrameReport{
			Order:     4,
			Width:     afterFrame.Width,
			Height:    afterFrame.Height,
			Stride:    afterFrame.Stride,
			Checksum:  checksumRGBA(afterFrame.Pixels),
			Presented: true,
		},
	)
	report := buildReport(smokeOptions{
		Mode:       "wasm32-web",
		SourcePath: "examples/surface/runtime/surface_counter.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target wasm32-web " +
				"examples/surface/runtime/surface_counter.tetra -o " +
				"/tmp/surface-artifacts/surface-counter.wasm"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name: "surface wasm32-web component app",
			Kind: "app",
			Path: ("node scripts/tools/web_run_module.mjs " +
				"/tmp/surface-artifacts/surface-counter.wasm"),
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(1),
			ExpectedExitCode: intPtr(1),
		},
		{
			Name: "surface wasm32-web import validator",
			Kind: "runtime",
			Path: ("go run ./tools/cmd/validate-wasm-imports --target " +
				"wasm32-web /tmp/surface-artifacts/surface-counter.wasm"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:     "surface wasm32-web runtime",
			Kind:     "runtime",
			Path:     "node --version",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, wasmSurfaceArtifacts(), cleanArtifactScan(3), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.HostEvidence.Level != "wasm32-web-compiler-owned-loader" ||
		report.HostEvidence.Backend != "node-surface-host" ||
		!report.HostEvidence.Framebuffer ||
		report.HostEvidence.RealWindow ||
		report.HostEvidence.NativeInput {
		t.Fatalf(
			("wasm32-web host evidence = %#v, want compiler-owned loader " +
				"evidence without production browser native-input claim"),
			report.HostEvidence,
		)
	}
	if !caseNamesContain(scenario.Cases, "wasm32-web Surface Host ABI imports") {
		t.Fatalf(
			"wasm32-web scenario cases = %#v, want Surface Host ABI import evidence",
			scenario.Cases,
		)
	}
	if !caseNamesContain(scenario.Cases, "compiler-owned wasm Surface loader") {
		t.Fatalf(
			"wasm32-web scenario cases = %#v, want compiler-owned loader evidence",
			scenario.Cases,
		)
	}
	if !caseNamesContain(scenario.Cases, "wasm32-web actual presented frame trace") {
		t.Fatalf(
			"wasm32-web scenario cases = %#v, want actual presented frame trace evidence",
			scenario.Cases,
		)
	}
	if caseNamesContain(scenario.Cases, "headless") {
		t.Fatalf(
			"wasm32-web scenario cases = %#v, must not reuse headless case names",
			scenario.Cases,
		)
	}
}

func TestWASM32WebBrowserCanvasModeProducesTargetSpecificRuntimeEvidence(t *testing.T) {
	if err := validateSmokeMode("wasm32-web-browser-canvas"); err != nil {
		t.Fatalf("validateSmokeMode(wasm32-web-browser-canvas) failed: %v", err)
	}
	scenario := runWASM32WebBrowserCanvasCounterScenario()
	beforeFrame := renderBrowserCounterFrameRGBA(0, 0, 320, 200, true)
	afterFrame := renderBrowserCounterFrameRGBA(2, 1, 400, 240, true)
	scenario.Frames = append(
		scenario.Frames,
		surface.FrameReport{
			Order:     1,
			Width:     beforeFrame.Width,
			Height:    beforeFrame.Height,
			Stride:    beforeFrame.Stride,
			Checksum:  checksumRGBA(beforeFrame.Pixels),
			Presented: true,
		},
		surface.FrameReport{
			Order:     5,
			Width:     afterFrame.Width,
			Height:    afterFrame.Height,
			Stride:    afterFrame.Stride,
			Checksum:  checksumRGBA(afterFrame.Pixels),
			Presented: true,
		},
	)
	report := buildReport(smokeOptions{
		Mode:       "wasm32-web-browser-canvas",
		SourcePath: "examples/surface/runtime/surface_browser_counter.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target wasm32-web " +
				"examples/surface/runtime/surface_browser_counter.tetra -o " +
				"/tmp/surface-artifacts/surface-browser-counter.wasm"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name: "surface wasm32-web browser canvas component app",
			Kind: "app",
			Path: ("/usr/bin/chromium --headless " +
				"<surface-browser-canvas-runner> " +
				"wasm=/tmp/surface-artifacts/surface-browser-counter.wasm"),
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(0),
			ExpectedExitCode: intPtr(0),
		},
		{
			Name: "surface wasm32-web import validator",
			Kind: "runtime",
			Path: ("go run ./tools/cmd/validate-wasm-imports --target " +
				"wasm32-web " +
				"/tmp/surface-artifacts/surface-browser-counter.wasm"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:     "surface wasm32-web browser canvas runtime",
			Kind:     "runtime",
			Path:     "Chromium test fixture",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name: "surface wasm32-web browser canvas trace",
			Kind: "runtime",
			Path: ("/usr/bin/chromium --headless --dump-dom http://127.0.0.1:" +
				"1/surface-browser-canvas-runner"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, wasmBrowserCanvasSurfaceArtifacts(), cleanArtifactScan(3), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.HostEvidence.Level != "wasm32-web-browser-canvas-input" ||
		report.HostEvidence.Backend != "browser-canvas-rgba" ||
		!report.HostEvidence.Framebuffer ||
		!report.HostEvidence.NativeInput ||
		report.HostEvidence.RealWindow {
		t.Fatalf(
			("browser canvas host evidence = %#v, want browser canvas " +
				"framebuffer/native-input without OS real-window claim"),
			report.HostEvidence,
		)
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
		t.Fatalf(
			"browser canvas state transitions = %#v, want click/key/resize/text transitions",
			scenario.StateTransitions,
		)
	}
}

func TestWASM32WebBrowserCanvasBlockSystemScenarioProducesBrowserEvidence(t *testing.T) {
	const mode = "wasm32-web-browser-canvas-block-system"
	if err := validateSmokeMode(mode); err != nil {
		t.Fatalf("validateSmokeMode(%s) failed: %v", mode, err)
	}
	if got := defaultSurfaceSourcePath(
		smokeOptions{Mode: mode, SourcePath: "examples/surface/runtime/surface_counter.tetra"},
	); got != "examples/surface/block_core/surface_block_system.tetra" {
		t.Fatalf(
			"defaultSurfaceSourcePath(%s) = %q, want examples/surface/block_core/surface_block_system.tetra",
			mode,
			got,
		)
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
	scenario.BlockSystem = blockSystemReportForWASM32WebBrowserCanvasScenario(
		"examples/surface/block_core/surface_block_system.tetra",
		scenario.Frames,
	)
	attachBlockSystemMemoryBudget(&scenario)
	report := buildReport(smokeOptions{
		Mode:       mode,
		SourcePath: "examples/surface/block_core/surface_block_system.tetra",
	}, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target wasm32-web " +
				"examples/surface/block_core/surface_block_system.tetra -o " +
				"/tmp/surface-artifacts/surface-block-system.wasm"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name: "surface wasm32-web browser canvas component app",
			Kind: "app",
			Path: ("/usr/bin/chromium --headless " +
				"<surface-browser-canvas-runner> scenario=block-system " +
				"wasm=/tmp/surface-artifacts/surface-block-system.wasm"),
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(0),
			ExpectedExitCode: intPtr(0),
		},
		{
			Name: "surface wasm32-web import validator",
			Kind: "runtime",
			Path: ("go run ./tools/cmd/validate-wasm-imports --target " +
				"wasm32-web /tmp/surface-artifacts/surface-block-system.wasm"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:     "surface wasm32-web browser canvas runtime",
			Kind:     "runtime",
			Path:     "Chromium Block-system fixture",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name: "surface wasm32-web browser canvas trace",
			Kind: "runtime",
			Path: ("/usr/bin/chromium --headless --dump-dom " +
				"<surface-browser-canvas-file-runner scenario=block-system>"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, []surface.ArtifactReport{
		{
			Kind:   "component-app",
			Path:   "/tmp/surface-artifacts/surface-block-system.wasm",
			SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Size:   8604,
		},
		{
			Kind:   "compiler-owned-loader",
			Path:   "/tmp/surface-artifacts/surface-block-system.mjs",
			SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			Size:   4939,
		},
		{
			Kind:   "runner-trace",
			Path:   "/tmp/surface-artifacts/surface-runner-trace.json",
			SHA256: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			Size:   1184,
		},
	}, cleanArtifactScan(3), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := surface.ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
	if report.BlockSystem == nil ||
		report.BlockSystem.QualityLevel != "wasm32-web-browser-canvas-block-system-v1" {
		t.Fatalf(
			"block_system = %#v, want wasm32-web browser-canvas Block system evidence",
			report.BlockSystem,
		)
	}
	if report.HostEvidence.Level != "wasm32-web-browser-canvas-input" ||
		!report.HostEvidence.BrowserCanvas ||
		!report.HostEvidence.BrowserInput ||
		report.HostEvidence.RealWindow {
		t.Fatalf(
			"host evidence = %#v, want browser-canvas input without OS real-window claim",
			report.HostEvidence,
		)
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
			if got := defaultSurfaceSourcePath(
				smokeOptions{Mode: mode, SourcePath: "examples/surface/runtime/surface_counter.tetra"},
			); got != "examples/surface/runtime/surface_textbox_app.tetra" {
				t.Fatalf(
					"defaultSurfaceSourcePath(%s) = %q, want examples/surface/runtime/surface_textbox_app.tetra",
					mode,
					got,
				)
			}
			scenario := runTextFocusInputScenario(mode)
			if len(scenario.Components) != 3 {
				t.Fatalf(
					"components = %#v, want app, TextBox, and Button evidence",
					scenario.Components,
				)
			}
			if scenario.Components[1].ID != "TextBox" ||
				scenario.Components[2].ID != "SubmitButton" {
				t.Fatalf(
					"components = %#v, want TextBox and SubmitButton child components",
					scenario.Components,
				)
			}
			if scenario.Components[1].State["buffer"] != "Z" ||
				scenario.Components[1].State["caret"] != "1" ||
				scenario.Components[2].State["press_count"] != "1" {
				t.Fatalf(
					"component state = %#v, want edited TextBox buffer/caret and focused Button press evidence",
					scenario.Components,
				)
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
			if len(reportScenario.Frames) < 2 ||
				reportScenario.Frames[0].Checksum == reportScenario.Frames[len(
					reportScenario.Frames,
				)-1].Checksum {
				t.Fatalf(
					"frames = %#v, want visible framebuffer update after text/focus changes",
					reportScenario.Frames,
				)
			}
			raw, err := json.Marshal(
				buildReport(
					smokeOptions{
						Mode:       mode,
						SourcePath: "examples/surface/runtime/surface_textbox_app.tetra",
					},
					"linux-x64",
					textFocusInputTestProcesses(mode),
					textFocusInputTestArtifacts(mode),
					cleanArtifactScan(3),
					reportScenario,
				),
			)
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
			opt := smokeOptions{
				Mode:       tc.mode,
				SourcePath: "examples/surface/runtime/surface_counter.tetra",
			}
			if got := defaultSurfaceSourcePath(
				opt,
			); got != "examples/surface/release/surface_release_text_input.tetra" {
				t.Fatalf(
					("defaultSurfaceSourcePath(%s) = %q, want " +
						"examples/surface/release/surface_release_text_input.tetra"),
					tc.mode,
					got,
				)
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
	opt := smokeOptions{Mode: mode, SourcePath: "examples/surface/runtime/surface_counter.tetra"}
	if got := defaultSurfaceSourcePath(
		opt,
	); got != "examples/surface/toolkit/surface_app_model.tetra" {
		t.Fatalf(
			"defaultSurfaceSourcePath(%s) = %q, want examples/surface/toolkit/surface_app_model.tetra",
			mode,
			got,
		)
	}
	scenario := runSurfaceScenario(mode)
	if scenario.AppModel == nil {
		t.Fatalf("scenario.AppModel is nil, want app-model command/reducer evidence")
	}
	if scenario.AppModel.Schema != "tetra.surface.app-model.v1" ||
		scenario.AppModel.AppModelLevel != "explicit-command-reducer-v1" {
		t.Fatalf(
			"app_model = %#v, want app-model v1 explicit command reducer evidence",
			scenario.AppModel,
		)
	}
	report := buildReport(opt, "linux-x64", []surface.ProcessReport{
		{
			Name: "tetra build",
			Kind: "build",
			Path: ("tetra build --target linux-x64 " +
				"examples/surface/toolkit/surface_app_model.tetra -o " +
				"/tmp/surface-artifacts/surface-app-model"),
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
		{
			Name:             "surface component app",
			Kind:             "app",
			Path:             "/tmp/surface-artifacts/surface-app-model",
			Ran:              true,
			Pass:             true,
			ExitCode:         intPtr(1),
			ExpectedExitCode: intPtr(1),
		},
		{
			Name:     "surface headless runtime",
			Kind:     "runtime",
			Path:     "tools/cmd/surface-runtime-smoke --mode headless-app-model",
			Ran:      true,
			Pass:     true,
			ExitCode: intPtr(0),
		},
	}, headlessSurfaceArtifacts(
		"/tmp/surface-artifacts/surface-app-model",
	), cleanArtifactScan(
		2,
	), scenario)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal app-model report: %v", err)
	}
	if report.Target != "headless" || report.Runtime != "surface-headless" {
		t.Fatalf(
			"target/runtime = %q/%q, want headless/surface-headless",
			report.Target,
			report.Runtime,
		)
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
			opt := smokeOptions{
				Mode:       tc.mode,
				SourcePath: "examples/surface/release/surface_release_counter.tetra",
			}
			if got := defaultSurfaceSourcePath(
				opt,
			); got != "examples/surface/release/surface_release_counter.tetra" {
				t.Fatalf(
					("defaultSurfaceSourcePath(%s) = %q, want " +
						"examples/surface/release/surface_release_counter.tetra"),
					tc.mode,
					got,
				)
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
				scenario.Frames = append(
					scenario.Frames,
					surface.FrameReport{
						Order:     1,
						Width:     beforeFrame.Width,
						Height:    beforeFrame.Height,
						Stride:    beforeFrame.Stride,
						Checksum:  checksumRGBA(beforeFrame.Pixels),
						Presented: true,
					},
					surface.FrameReport{
						Order:     5,
						Width:     afterFrame.Width,
						Height:    afterFrame.Height,
						Stride:    afterFrame.Stride,
						Checksum:  checksumRGBA(afterFrame.Pixels),
						Presented: true,
					},
				)
			}
			for _, component := range scenario.Components {
				if strings.HasPrefix(component.Type, "examples.surface_") &&
					!strings.HasPrefix(
						component.Type,
						"examples.surface.release.surface_release_counter.",
					) {
					t.Fatalf("component type = %q, want release counter module", component.Type)
				}
			}
			processes, artifacts := releaseCounterTestEvidence(tc.mode)
			raw, err := json.Marshal(
				buildReport(
					opt,
					"linux-x64",
					processes,
					artifacts,
					cleanArtifactScan(len(artifacts)),
					scenario,
				),
			)
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
			if report.Source != "examples/surface/release/surface_release_counter.tetra" {
				t.Fatalf(
					"source = %q, want examples/surface/release/surface_release_counter.tetra",
					report.Source,
				)
			}
			if err := surface.ValidateReport(raw); err != nil {
				t.Fatalf("ValidateReport failed for release counter %s: %v\n%s", tc.mode, err, raw)
			}
		})
	}
}
