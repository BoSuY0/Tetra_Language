package main

import (
	"bytes"
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

func compactFixtureJSON(raw string) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(raw)); err != nil {
		panic("compact fixture JSON: " + err.Error())
	}
	return buf.String()
}

func fixturePositiveCaseLine(name string) string {
	return `    {"name":` + jsonString(name) +
		`,"kind":"positive","ran":true,"pass":true}`
}

func fixtureNegativeCaseLine(name string, expectedError string) string {
	return `    {"name":` + jsonString(name) +
		`,"kind":"negative","ran":true,"pass":true,"expected_error":` +
		jsonString(expectedError) + `}`
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
	return writeNamedSurfaceArtifactFixture(
		t,
		dir,
		"surface-counter",
		[]byte("surface component artifact fixture\n"),
		0o755,
	)
}

func writeNamedSurfaceArtifactFixture(
	t *testing.T,
	dir string,
	name string,
	contents []byte,
	perm os.FileMode,
) (string, string, int64) {
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
	return writeSurfaceTraceFixtureWithSourceAndFrames(
		t,
		dir,
		"examples/surface/runtime/surface_counter.tetra",
		[]surfaceTraceFrameFixture{
			{
				Order:     1,
				Width:     320,
				Height:    200,
				Stride:    1280,
				Checksum:  "8ff0b13bb6026d76c7f61b202436f30e703d45d96730033ee6d171bcf5802f81",
				Presented: true,
			},
			{
				Order:     2,
				Width:     320,
				Height:    200,
				Stride:    1280,
				Checksum:  "9ff0b13bb6026d76c7f61b202436f30e703d45d96730033ee6d171bcf5802f82",
				Presented: true,
			},
		},
	)
}

type surfaceTraceFrameFixture struct {
	Order     int    `json:"order"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Stride    int    `json:"stride"`
	Checksum  string `json:"checksum"`
	Presented bool   `json:"presented"`
}

func writeSurfaceTraceFixtureWithFrames(
	t *testing.T,
	dir string,
	frames []surfaceTraceFrameFixture,
) (string, string, int64) {
	t.Helper()
	return writeSurfaceTraceFixtureWithSourceAndFrames(
		t,
		dir,
		"examples/surface/runtime/surface_counter.tetra",
		frames,
	)
}

func writeSurfaceTraceFixtureWithSourceAndFrames(
	t *testing.T,
	dir string,
	source string,
	frames []surfaceTraceFrameFixture,
) (string, string, int64) {
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

func writeWASMTraceFixtureWithFrames(
	t *testing.T,
	dir string,
	frames []wasmTraceFrameFixture,
) (string, string, int64) {
	t.Helper()
	return writeWASMTraceFixtureWithWASMAndFrames(
		t,
		dir,
		filepath.Join(dir, "surface-counter.wasm"),
		frames,
	)
}

func writeWASMTraceFixtureWithWASMAndFrames(
	t *testing.T,
	dir string,
	wasmPath string,
	frames []wasmTraceFrameFixture,
) (string, string, int64) {
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

func writeBrowserCanvasTraceFixture(
	t *testing.T,
	dir string,
	wasmPath string,
) (string, string, int64) {
	t.Helper()
	return writeBrowserCanvasTraceFixtureWithChecksums(t, dir, wasmPath,
		"1111111111111111111111111111111111111111111111111111111111111111",
		"5555555555555555555555555555555555555555555555555555555555555555",
		"5555555555555555555555555555555555555555555555555555555555555555",
	)
}

func writeBrowserCanvasTraceFixtureWithChecksums(
	t *testing.T,
	dir string,
	wasmPath string,
	firstChecksum string,
	secondSourceChecksum string,
	secondCanvasChecksum string,
) (string, string, int64) {
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
			{
				Order:          1,
				Width:          320,
				Height:         200,
				Stride:         1280,
				PixelsLen:      256000,
				SourceChecksum: firstChecksum,
				CanvasChecksum: firstChecksum,
				Checksum:       firstChecksum,
				Presented:      true,
			},
			{
				Order:          5,
				Width:          400,
				Height:         240,
				Stride:         1600,
				PixelsLen:      384000,
				SourceChecksum: secondSourceChecksum,
				CanvasChecksum: secondCanvasChecksum,
				Checksum:       secondCanvasChecksum,
				Presented:      true,
			},
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
	wasmPath, wasmSHA, wasmSize := writeNamedSurfaceArtifactFixture(
		t,
		artifactDir,
		"surface-release-form.wasm",
		[]byte("\x00asm\x01\x00\x00\x00surface browser release wasm fixture\n"),
		0o755,
	)
	loaderPath, loaderSHA, loaderSize := writeNamedSurfaceArtifactFixture(
		t,
		artifactDir,
		"surface-release-form.mjs",
		[]byte(`function createSurfaceHost(instanceRef) {
  return { __tetra_surface_present_rgba() { return 0; } };
}
const imports = { tetra_surface_host_v1: createSurfaceHost({ instance: null }) };
`),
		0o644,
	)
	tracePath, traceSHA, traceSize := writeBrowserReleaseTraceFixture(t, artifactDir, wasmPath)
	reportPath := filepath.Join(dir, "surface-wasm32-web-release-browser.json")
	raw := string(
		validWASM32WebBrowserCanvasSurfaceRuntimeReportJSON(
			wasmPath,
			wasmSHA,
			wasmSize,
			loaderPath,
			loaderSHA,
			loaderSize,
			tracePath,
			traceSHA,
			traceSize,
		),
	)
	raw = strings.Replace(
		raw,
		`"source": "examples/surface/runtime/surface_browser_counter.tetra"`,
		`"source": "examples/surface/release/surface_release_form.tetra"`,
		1,
	)
	raw = strings.Replace(
		raw,
		("\"host_evidence\": {\"level\":\"wasm32-web-browser-canvas-input\"," +
			"\"backend\":\"browser-canvas-rgba\",\"framebuffer\":true," +
			"\"real_window\":false,\"native_input\":true," +
			"\"user_facing_platform_widgets\":false}"),
		("\"host_evidence\": {\"level\":" +
			"\"wasm32-web-browser-canvas-release-v1\",\"backend\":" +
			"\"browser-canvas-rgba-accessible\",\"framebuffer\":true," +
			"\"real_window\":false,\"native_input\":true,\"browser_canvas\":" +
			"true,\"browser_input\":true,\"browser_clipboard\":true," +
			"\"browser_clipboard_harness\":" +
			"\"deterministic-browser-clipboard-v1\",\"browser_composition\":" +
			"true,\"browser_accessibility_snapshot\":true," +
			"\"browser_accessibility_mirror\":true," +
			"\"user_facing_platform_widgets\":false}"),
		1,
	)
	raw = strings.Replace(
		raw,
		`examples/surface/runtime/surface_browser_counter.tetra`,
		`examples/surface/release/surface_release_form.tetra`,
		1,
	)
	raw = strings.Replace(
		raw,
		`<surface-browser-canvas-runner> wasm=`,
		`<surface-browser-canvas-runner> scenario=release-browser wasm=`,
		1,
	)
	var report map[string]any
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		t.Fatalf("decode browser release report fixture: %v", err)
	}
	report["browser_surface"] = browserSurfaceEvidenceMapForTest(tracePath)
	mutatedRaw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal browser release report fixture: %v", err)
	}
	if err := os.WriteFile(reportPath, mutatedRaw, 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	return reportPath
}

func browserSurfaceEvidenceMapForTest(tracePath string) map[string]any {
	return map[string]any{
		"schema":                "tetra.surface.browser-surface.v1",
		"browser_surface_level": "browser-canvas-release-v1",
		"release_scope":         "surface-v1-linux-web",
		"source":                "examples/surface/release/surface_release_form.tetra",
		"host_adapter":          "compiler-owned-browser-canvas-host",
		"production_claim":      true,
		"experimental":          false,
		"compiler_owned_boot":   true,
		"dom_host_canvas_only":  true,
		"canvas": map[string]any{
			"opened":        true,
			"readback":      true,
			"width":         560,
			"height":        420,
			"frame_order":   5,
			"artifact_kind": "runner-trace",
			"pass":          true,
		},
		"input": map[string]any{
			"pointer":       true,
			"keyboard":      true,
			"text":          true,
			"resize":        true,
			"host_trace":    true,
			"native_events": []any{"pointerup", "keydown", "beforeinput", "resize"},
			"pass":          true,
		},
		"clipboard": map[string]any{
			"harness":    "deterministic-browser-clipboard-v1",
			"read":       true,
			"write":      true,
			"owned_copy": true,
			"bytes":      13,
			"pass":       true,
		},
		"composition": map[string]any{
			"start":  true,
			"update": true,
			"commit": true,
			"cancel": true,
			"pass":   true,
		},
		"accessibility": map[string]any{
			"snapshot":       true,
			"mirror":         true,
			"compiler_owned": true,
			"bounds":         true,
			"focus":          true,
			"roles":          []any{"root", "textbox", "checkbox", "button", "status"},
			"dom_visual_ui":  false,
			"user_js":        false,
			"pass":           true,
		},
		"host_traces": []any{
			map[string]any{
				"name":          "browser-canvas",
				"artifact_kind": "runner-trace",
				"path":          tracePath,
				"pass":          true,
			},
		},
		"negative_guards": map[string]any{
			"no_dom_app_ui_tree":     true,
			"no_user_js_app_logic":   true,
			"no_node_only_promotion": true,
			"no_legacy_sidecars":     true,
			"no_react_runtime":       true,
			"no_platform_widgets":    true,
		},
	}
}

func writeBrowserReleaseTraceFixture(
	t *testing.T,
	dir string,
	wasmPath string,
) (string, string, int64) {
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
			{
				Order:          1,
				Width:          320,
				Height:         200,
				Stride:         1280,
				PixelsLen:      256000,
				SourceChecksum: "1111111111111111111111111111111111111111111111111111111111111111",
				CanvasChecksum: "1111111111111111111111111111111111111111111111111111111111111111",
				Checksum:       "1111111111111111111111111111111111111111111111111111111111111111",
				Presented:      true,
			},
			{
				Order:          5,
				Width:          400,
				Height:         240,
				Stride:         1600,
				PixelsLen:      384000,
				SourceChecksum: "5555555555555555555555555555555555555555555555555555555555555555",
				CanvasChecksum: "5555555555555555555555555555555555555555555555555555555555555555",
				Checksum:       "5555555555555555555555555555555555555555555555555555555555555555",
				Presented:      true,
			},
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

func validSurfaceRuntimeReportJSON(
	artifactPath string,
	artifactSHA string,
	artifactSize int64,
	tracePath string,
	traceSHA string,
	traceSize int64,
) []byte {
	buildPath := ("tetra build --target linux-x64 " +
		"examples/surface/runtime/surface_counter.tetra -o ") + artifactPath
	artifactScanRoot := filepath.Dir(artifactPath)
	raw := strings.Join([]string{
		"{",
		`  "schema": "tetra.surface.runtime.v1",`,
		`  "status": "pass",`,
		`  "target": "headless",`,
		`  "host": "linux-x64",`,
		`  "runtime": "surface-headless",`,
		`  "surface_schema": "tetra.surface.v1",`,
		`  "host_abi": "tetra.surface.host-abi.v1",`,
		`  "host_evidence": ` + compactFixtureJSON(`{
			"level": "deterministic-headless",
			"backend": "software-rgba",
			"framebuffer": true,
			"real_window": false,
			"native_input": false,
			"user_facing_platform_widgets": false
		}`) + `,`,
		`  "source": "examples/surface/runtime/surface_counter.tetra",`,
		`  "processes": [`,
		`    ` + compactFixtureJSON(`{
			"name": "tetra build",
			"kind": "build",
			"path": "__BUILD_PROCESS_PATH__",
			"ran": true,
			"pass": true,
			"exit_code": 0
		}`) + `,`,
		`    ` + compactFixtureJSON(`{
			"name": "surface component app",
			"kind": "app",
			"path": "__ARTIFACT_PATH__",
			"ran": true,
			"pass": true,
			"exit_code": 1,
			"expected_exit_code": 1
		}`) + `,`,
		`    ` + compactFixtureJSON(`{
			"name": "surface headless runtime",
			"kind": "runtime",
			"path": "tools/cmd/surface-runtime-smoke",
			"ran": true,
			"pass": true,
			"exit_code": 0
		}`),
		`  ],`,
		`  "artifacts": [`,
		`    ` + compactFixtureJSON(`{
			"kind": "component-app",
			"path": "__ARTIFACT_PATH__",
			"sha256": "__ARTIFACT_SHA__",
			"size": "__ARTIFACT_SIZE__"
		}`) + `,`,
		`    ` + compactFixtureJSON(`{
			"kind": "runner-trace",
			"path": "__TRACE_PATH__",
			"sha256": "__TRACE_SHA__",
			"size": "__TRACE_SIZE__"
		}`),
		`  ],`,
		`  "artifact_scan": ` + compactFixtureJSON(`{
			"root": "__ARTIFACT_SCAN_ROOT__",
			"files_checked": 2,
			"forbidden_paths": [],
			"pass": true
		}`) + `,`,
		`  "components": [`,
		`    ` + compactFixtureJSON(`{
			"id": "CounterApp",
			"type": "examples.surface.runtime.surface_counter.CounterApp",
			"bounds": {"x":0,"y":0,"w":320,"h":200},
			"abilities": ["measure","layout","draw","event","focus","text","accessibility"],
			"state": {
				"count": "1",
				"text_count": "1",
				"accessibility_role": "button"
			}
		}`) + `,`,
		`    ` + compactFixtureJSON(`{
			"id": "CounterButton",
			"type": "examples.surface.runtime.surface_counter.CounterButton",
			"parent": "CounterApp",
			"bounds": {"x":32,"y":80,"w":160,"h":48},
			"abilities": ["measure","layout","draw","event","focus","text","accessibility"],
			"state": {
				"pressed": "false",
				"focused": "true",
				"text_len_seen": "2",
				"accessibility_role": "button"
			}
		}`),
		`  ],`,
		`  "events": [`,
		`    ` + compactFixtureJSON(`{
			"order": 1,
			"kind": "none",
			"target_component": "CounterApp",
			"dispatch_path": ["CounterApp"],
			"handled": false,
			"pass": true,
			"x": 0,
			"y": 0,
			"before_state": {"CounterApp.count":"0"},
			"after_state": {"CounterApp.count":"0"}
		}`) + `,`,
		`    ` + compactFixtureJSON(`{
			"order": 2,
			"kind": "mouse_up",
			"target_component": "CounterButton",
			"dispatch_path": ["CounterApp","CounterButton"],
			"handled": true,
			"pass": true,
			"x": 48,
			"y": 96,
			"key": 0,
			"width": 320,
			"height": 200,
			"timestamp_ms": 0,
			"buffer_slots": [5,48,96,1,0,320,200,0,0],
			"before_state": {"CounterApp.count":"0","CounterButton.pressed":"false"},
			"after_state": {"CounterApp.count":"1","CounterButton.pressed":"false"}
		}`) + `,`,
		`    ` + compactFixtureJSON(`{
			"order": 3,
			"kind": "text_input",
			"target_component": "CounterButton",
			"dispatch_path": ["CounterApp","CounterButton"],
			"handled": true,
			"pass": true,
			"x": 0,
			"y": 0,
			"key": 0,
			"width": 320,
			"height": 200,
			"timestamp_ms": 1,
			"text_len": 2,
			"text_bytes_hex": "4f4b",
			"buffer_slots": [8,0,0,0,0,320,200,1,2],
			"before_state": {
				"CounterApp.text_count":"0",
				"CounterButton.text_len_seen":"0"
			},
			"after_state": {
				"CounterApp.text_count":"1",
				"CounterButton.text_len_seen":"2"
			}
		}`),
		`  ],`,
		`  "frames": [`,
		`    ` + compactFixtureJSON(`{
			"order": 1,
			"width": 320,
			"height": 200,
			"stride": 1280,
			"checksum": "8ff0b13bb6026d76c7f61b202436f30e703d45d96730033ee6d171bcf5802f81",
			"presented": true
		}`) + `,`,
		`    ` + compactFixtureJSON(`{
			"order": 2,
			"width": 320,
			"height": 200,
			"stride": 1280,
			"checksum": "9ff0b13bb6026d76c7f61b202436f30e703d45d96730033ee6d171bcf5802f82",
			"presented": true
		}`),
		`  ],`,
		`  "state_transitions": [`,
		`    ` + compactFixtureJSON(`{
			"order": 1,
			"component": "CounterApp",
			"field": "count",
			"before": "0",
			"after": "1",
			"cause": "mouse_up"
		}`) + `,`,
		`    ` + compactFixtureJSON(`{
			"order": 2,
			"component": "CounterApp",
			"field": "text_count",
			"before": "0",
			"after": "1",
			"cause": "text_input"
		}`),
		`  ],`,
		`  "cases": [`,
		`    {"name":"pure Tetra component app","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"headless event dispatch","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"headless framebuffer checksum","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"headless actual runner trace","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"host-provided pointer event dispatch","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"host event buffer poll_event","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"pre/post event frame sequence","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"component hierarchy dispatch","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"component text input scalar dispatch","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"host text payload buffer","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"component focus dispatch","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"component accessibility metadata","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"no legacy UI sidecar artifacts","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"state transition","kind":"positive","ran":true,"pass":true},`,
		fixtureNegativeCaseLine("reject legacy UI evidence", "legacy UI evidence rejected"),
		`  ]`,
		`}`,
	}, "\n")
	raw = strings.NewReplacer(
		`"__BUILD_PROCESS_PATH__"`, jsonString(buildPath),
		`"__ARTIFACT_PATH__"`, jsonString(artifactPath),
		`"__ARTIFACT_SHA__"`, jsonString(artifactSHA),
		`"__ARTIFACT_SIZE__"`, strconv.FormatInt(artifactSize, 10),
		`"__TRACE_PATH__"`, jsonString(tracePath),
		`"__TRACE_SHA__"`, jsonString(traceSHA),
		`"__TRACE_SIZE__"`, strconv.FormatInt(traceSize, 10),
		`"__ARTIFACT_SCAN_ROOT__"`, jsonString(artifactScanRoot),
	).Replace(raw)
	return []byte(raw)
}

func validNativeSurfaceHostReleaseRuntimeReportJSON(
	t *testing.T,
	artifactDir string,
	mutate func(map[string]any),
) []byte {
	t.Helper()
	source := "examples/surface/runtime/surface_window_counter.tetra"
	componentPath, componentSHA, componentSize := writeNamedSurfaceArtifactFixture(
		t,
		artifactDir,
		"surface-window-counter",
		[]byte("compiled linux-x64 Tetra Surface counter fixture\n"),
		0o755,
	)
	hostPath, hostSHA, hostSize := writeNamedSurfaceArtifactFixture(
		t,
		artifactDir,
		"tetra-surface-host-wayland",
		[]byte("official Tetra Surface Wayland host fixture\n"),
		0o755,
	)
	hostReportPath, hostReportSHA, hostReportSize := writeNamedSurfaceArtifactFixture(
		t,
		artifactDir,
		"tetra-surface-host-report.json",
		[]byte(
			("{\"schema\":\"tetra.surface.host-report.v1\",\"protocol\":"+
				"\"tetra.surface.host-ipc.v1\",\"presented_frame_count\":2,"+
				"\"real_pointer_event_count\":1,\"real_key_event_count\":1,"+
				"\"real_close_event\":true}")+"\n",
		),
		0o644,
	)
	hostProcessPath := hostPath +
		" --socket /run/user/1000/tetra-surface-host.sock --report " +
		hostReportPath
	report := map[string]any{
		"schema":         "tetra.surface.runtime.v1",
		"status":         "pass",
		"target":         "linux-x64",
		"host":           "linux-x64",
		"runtime":        "surface-linux-x64",
		"surface_schema": "tetra.surface.v1",
		"host_abi":       "tetra.surface.host-abi.v1",
		"host_evidence": map[string]any{
			"level":                        surface.NativeSurfaceHostLevelLinuxX64,
			"backend":                      surface.NativeSurfaceHostBackendWayland,
			"framebuffer":                  true,
			"real_window":                  true,
			"native_input":                 true,
			"user_facing_platform_widgets": false,
		},
		"source": source,
		"processes": []any{
			map[string]any{
				"name":      "tetra build native surface host app",
				"kind":      "build",
				"path":      "tetra build --target linux-x64 " + source + " -o " + componentPath,
				"ran":       true,
				"pass":      true,
				"exit_code": 0,
			},
			map[string]any{
				"name":               "surface component app",
				"kind":               "app",
				"path":               componentPath + " --surface-host wayland",
				"ran":                true,
				"pass":               true,
				"exit_code":          0,
				"expected_exit_code": 0,
			},
			map[string]any{
				"name":      "surface linux-x64 native surface host wayland",
				"kind":      "runtime",
				"path":      hostProcessPath,
				"ran":       true,
				"pass":      true,
				"exit_code": 0,
			},
		},
		"artifacts": []any{
			map[string]any{
				"kind":   "component-app",
				"path":   componentPath,
				"sha256": componentSHA,
				"size":   componentSize,
			},
			map[string]any{
				"kind":   "surface-host",
				"path":   hostPath,
				"sha256": hostSHA,
				"size":   hostSize,
			},
			map[string]any{
				"kind":   "native-surface-host-report",
				"path":   hostReportPath,
				"sha256": hostReportSHA,
				"size":   hostReportSize,
			},
		},
		"artifact_scan": map[string]any{
			"root":            artifactDir,
			"files_checked":   3,
			"forbidden_paths": []any{},
			"pass":            true,
		},
		"components": []any{
			map[string]any{
				"id":     "CounterApp",
				"type":   "examples.surface.runtime.surface_window_counter.CounterApp",
				"bounds": map[string]any{"x": 0, "y": 0, "w": 320, "h": 200},
				"abilities": []any{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				"state": map[string]any{
					"count":              "1",
					"text_count":         "1",
					"key_count":          "1",
					"closed":             "true",
					"accessibility_role": "button",
				},
			},
			map[string]any{
				"id":     "CounterButton",
				"type":   "examples.surface.runtime.surface_window_counter.CounterButton",
				"parent": "CounterApp",
				"bounds": map[string]any{"x": 32, "y": 80, "w": 160, "h": 48},
				"abilities": []any{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				"state": map[string]any{
					"pressed":            "false",
					"focused":            "true",
					"text_len_seen":      "2",
					"accessibility_role": "button",
				},
			},
		},
		"events": []any{
			map[string]any{
				"order":            1,
				"kind":             "mouse_up",
				"target_component": "CounterButton",
				"dispatch_path":    []any{"CounterApp", "CounterButton"},
				"handled":          true,
				"pass":             true,
				"x":                48,
				"y":                96,
				"key":              0,
				"width":            320,
				"height":           200,
				"timestamp_ms":     0,
				"buffer_slots":     []any{5, 48, 96, 1, 0, 320, 200, 0, 0},
				"before_state": map[string]any{
					"CounterApp.count":      "0",
					"CounterButton.pressed": "false",
				},
				"after_state": map[string]any{
					"CounterApp.count":      "1",
					"CounterButton.pressed": "false",
				},
			},
			map[string]any{
				"order":            2,
				"kind":             "text_input",
				"target_component": "CounterButton",
				"dispatch_path":    []any{"CounterApp", "CounterButton"},
				"handled":          true,
				"pass":             true,
				"x":                0,
				"y":                0,
				"key":              0,
				"width":            320,
				"height":           200,
				"timestamp_ms":     1,
				"text_len":         2,
				"text_bytes_hex":   "4f4b",
				"buffer_slots":     []any{8, 0, 0, 0, 0, 320, 200, 1, 2},
				"before_state": map[string]any{
					"CounterApp.text_count":       "0",
					"CounterButton.text_len_seen": "0",
				},
				"after_state": map[string]any{
					"CounterApp.text_count":       "1",
					"CounterButton.text_len_seen": "2",
				},
			},
			map[string]any{
				"order":            3,
				"kind":             "key_down",
				"target_component": "CounterApp",
				"dispatch_path":    []any{"CounterApp"},
				"handled":          true,
				"pass":             true,
				"x":                0,
				"y":                0,
				"key":              32,
				"width":            320,
				"height":           200,
				"timestamp_ms":     2,
				"buffer_slots":     []any{6, 0, 0, 0, 32, 320, 200, 2, 0},
				"before_state":     map[string]any{"CounterApp.key_count": "0"},
				"after_state":      map[string]any{"CounterApp.key_count": "1"},
			},
			map[string]any{
				"order":            4,
				"kind":             "close",
				"target_component": "CounterApp",
				"dispatch_path":    []any{"CounterApp"},
				"handled":          true,
				"pass":             true,
				"x":                0,
				"y":                0,
				"key":              0,
				"width":            320,
				"height":           200,
				"timestamp_ms":     3,
				"buffer_slots":     []any{1, 0, 0, 0, 0, 320, 200, 3, 0},
				"before_state":     map[string]any{"CounterApp.closed": "false"},
				"after_state":      map[string]any{"CounterApp.closed": "true"},
			},
		},
		"frames": []any{
			map[string]any{
				"order":         1,
				"width":         320,
				"height":        200,
				"stride":        1280,
				"checksum":      "1111111111111111111111111111111111111111111111111111111111111111",
				"producer":      "app",
				"evidence_role": "runtime_smoke",
				"app_source":    source,
				"precomputed":   false,
				"presented":     true,
			},
			map[string]any{
				"order":         2,
				"width":         320,
				"height":        200,
				"stride":        1280,
				"checksum":      "2222222222222222222222222222222222222222222222222222222222222222",
				"producer":      "app",
				"evidence_role": "runtime_smoke",
				"app_source":    source,
				"precomputed":   false,
				"presented":     true,
			},
		},
		"state_transitions": []any{
			map[string]any{
				"order":     1,
				"component": "CounterApp",
				"field":     "count",
				"before":    "0",
				"after":     "1",
				"cause":     "mouse_up",
			},
			map[string]any{
				"order":     2,
				"component": "CounterApp",
				"field":     "text_count",
				"before":    "0",
				"after":     "1",
				"cause":     "text_input",
			},
			map[string]any{
				"order":     3,
				"component": "CounterApp",
				"field":     "key_count",
				"before":    "0",
				"after":     "1",
				"cause":     "key_down",
			},
			map[string]any{
				"order":     4,
				"component": "CounterApp",
				"field":     "closed",
				"before":    "false",
				"after":     "true",
				"cause":     "close",
			},
		},
		"cases": []any{
			map[string]any{
				"name": "pure Tetra component app",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "native Surface host Wayland live window",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "native Surface host app loop observed",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "native Surface host close event",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "native Surface host pointer input",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "native Surface host keyboard input",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "native Surface host frame presented by running app",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "host-provided pointer event dispatch",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "host event buffer poll_event",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "pre/post event frame sequence",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "component hierarchy dispatch",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "component text input scalar dispatch",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "host text payload buffer",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "component focus dispatch",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "component accessibility metadata",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "no legacy UI sidecar artifacts",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "state transition",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name":           "native Surface host rejects pre-rendered frame source",
				"kind":           "negative",
				"ran":            true,
				"pass":           true,
				"expected_error": "pre-rendered frame source rejected",
			},
			map[string]any{
				"name":           "native Surface host rejects viewer substitution",
				"kind":           "negative",
				"ran":            true,
				"pass":           true,
				"expected_error": "viewer substitution rejected",
			},
			map[string]any{
				"name":           "native Surface host rejects probe-frame substitution",
				"kind":           "negative",
				"ran":            true,
				"pass":           true,
				"expected_error": "probe-frame substitution rejected",
			},
			map[string]any{
				"name":           "reject legacy UI evidence",
				"kind":           "negative",
				"ran":            true,
				"pass":           true,
				"expected_error": "legacy UI evidence rejected",
			},
		},
		"native_surface_host": map[string]any{
			"schema":                    surface.NativeSurfaceHostSchemaV1,
			"host":                      "wayland",
			"protocol":                  surface.NativeSurfaceHostProtocolV1,
			"app_process_kind":          "compiled-linux-x64-tetra-app",
			"host_process_kind":         "tetra-surface-host-wayland",
			"app_pid":                   4242,
			"host_pid":                  4243,
			"surface_open_from_app":     true,
			"poll_event_from_host":      true,
			"present_from_app_rgba":     true,
			"app_loop_observed":         true,
			"real_window":               true,
			"real_close_event":          true,
			"real_pointer_event_count":  1,
			"real_key_event_count":      1,
			"presented_frame_count":     2,
			"pre_rendered_frame_source": false,
			"delivery_path":             "compiled-tetra-app-to-wayland-surface",
		},
	}
	if mutate != nil {
		mutate(report)
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal native Surface host release report: %v", err)
	}
	return raw
}

func validProductionTextInputReportJSON(t *testing.T) []byte {
	t.Helper()
	intRef := func(v int) *int { return &v }
	report := surface.TextInputReport{
		Schema:                     surface.TextInputSchemaV1,
		Target:                     "headless",
		Source:                     "examples/surface/release/surface_release_text_input.tetra",
		Level:                      "production-text-input-v1",
		Experimental:               false,
		ProductionClaim:            true,
		Storage:                    "owned-utf8-byte-buffer",
		UTF8Validation:             true,
		InvalidUTF8Rejected:        true,
		Caret:                      true,
		Selection:                  true,
		SelectionClipboardTransfer: true,
		Multiline:                  true,
		Backspace:                  true,
		Delete:                     true,
		HomeEnd:                    true,
		ArrowLeftRight:             true,
		CompositionEvents:          true,
		CompositionCommit:          true,
		CompositionCancel:          true,
		ClipboardRead:              true,
		ClipboardWrite:             true,
		ClipboardHostABI:           true,
		ClipboardOwnedCopy:         true,
		TargetHostCompositionTrace: true,
		CompositionTrace: surface.CompositionTraceReport{
			Start:  true,
			Update: true,
			Commit: true,
			Cancel: true,
		},
		TextShapingPlan: surface.TextShapingPlanReport{
			QualityLevel:       "scoped-text-shaping-plan-v1",
			FallbackFonts:      true,
			GraphemeBoundaries: "byte-offset-codepoint-v1",
			LineBreaking:       "newline-storage-plus-wrap-plan-v1",
			Bidi:               "nonclaim-full-bidi-v1",
			RichText:           "nonclaim-rich-text-editor-v1",
		},
		ReferenceTraces: []surface.TextInputReferenceTraceReport{
			{
				Source:      "examples/surface/morph_core/surface_morph_settings.tetra",
				Trace:       "settings text field trace",
				Focus:       true,
				Selection:   true,
				Clipboard:   true,
				Composition: true,
				Multiline:   true,
				Pass:        true,
			},
			{
				Source:      "examples/surface/morph_core/surface_morph_editor_shell.tetra",
				Trace:       "editor shell text area trace",
				Focus:       true,
				Selection:   true,
				Clipboard:   true,
				Composition: true,
				Multiline:   true,
				Pass:        true,
			},
		},
		UnsupportedClaims: []string{
			"full-rich-text-editor",
			"full-bidi-shaping",
			"grapheme-cluster-caret",
			"ide-grade-editor",
		},
		RichTextProductionClaim:   false,
		BidiProductionClaim:       false,
		FullEditorProductionClaim: false,
		BorrowedViewStorage:       false,
		SafeViewLifetimeChecked:   true,
		Processes: []surface.ProcessReport{
			{
				Name: "tetra build",
				Kind: "build",
				Path: ("tetra build --target linux-x64 " +
					"examples/surface/release/surface_release_text_input.tetra " +
					"-o /tmp/surface-artifacts/surface-release-text-input"),
				Ran:      true,
				Pass:     true,
				ExitCode: intRef(0),
			},
			{
				Name:             "surface component app",
				Kind:             "app",
				Path:             "/tmp/surface-artifacts/surface-release-text-input",
				Ran:              true,
				Pass:             true,
				ExitCode:         intRef(1),
				ExpectedExitCode: intRef(1),
			},
			{
				Name:     "surface headless runtime",
				Kind:     "runtime",
				Path:     "tools/cmd/surface-runtime-smoke --mode headless-release-text-input",
				Ran:      true,
				Pass:     true,
				ExitCode: intRef(0),
			},
		},
		Artifacts: []surface.ArtifactReport{
			{
				Kind:   "component-app",
				Path:   "/tmp/surface-artifacts/surface-release-text-input",
				SHA256: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Size:   4096,
			},
			{
				Kind:   "runner-trace",
				Path:   "/tmp/surface-artifacts/surface-runner-trace.json",
				SHA256: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				Size:   2048,
			},
		},
		ArtifactScan: surface.ArtifactScanReport{
			Root:           "/tmp/surface-artifacts",
			FilesChecked:   2,
			ForbiddenPaths: []string{},
			Pass:           true,
		},
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
			{
				Name:          "release text input invalid UTF-8 rejected",
				Kind:          "negative",
				Ran:           true,
				Pass:          true,
				ExpectedError: "invalid utf8 rejected",
			},
			{Name: "release text input multiline storage", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "release text input caret home end arrows",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "release text input selection replacement",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
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
			{
				Name: "release text input composition commit",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "release text input composition cancel",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "release text input shaping plan scoped",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
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
		},
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal text input report: %v", err)
	}
	return raw
}

func validAppModelReleaseRuntimeReportJSON(
	t *testing.T,
	artifactPath string,
	artifactSHA string,
	artifactSize int64,
	tracePath string,
	traceSHA string,
	traceSize int64,
) []byte {
	t.Helper()
	intRef := func(v int) *int { return &v }
	report := surface.Report{
		Schema:        surface.SchemaV1,
		Status:        "pass",
		Target:        "headless",
		Host:          "linux-x64",
		Runtime:       "surface-headless",
		SurfaceSchema: "tetra.surface.v1",
		HostABI:       "tetra.surface.host-abi.v1",
		HostEvidence: surface.HostEvidenceReport{
			Level:       "deterministic-headless",
			Backend:     "software-rgba",
			Framebuffer: true,
		},
		Source: "examples/surface/toolkit/surface_app_model.tetra",
		Processes: []surface.ProcessReport{
			{
				Name: "tetra build",
				Kind: "build",
				Path: ("tetra build --target linux-x64 " +
					"examples/surface/toolkit/surface_app_model.tetra -o ") + artifactPath,
				Ran:      true,
				Pass:     true,
				ExitCode: intRef(0),
			},
			{
				Name:             "surface component app",
				Kind:             "app",
				Path:             artifactPath,
				Ran:              true,
				Pass:             true,
				ExitCode:         intRef(1),
				ExpectedExitCode: intRef(1),
			},
			{
				Name:     "surface headless runtime",
				Kind:     "runtime",
				Path:     "tools/cmd/surface-runtime-smoke --mode headless-app-model",
				Ran:      true,
				Pass:     true,
				ExitCode: intRef(0),
			},
		},
		Artifacts: []surface.ArtifactReport{
			{Kind: "component-app", Path: artifactPath, SHA256: artifactSHA, Size: artifactSize},
			{Kind: "runner-trace", Path: tracePath, SHA256: traceSHA, Size: traceSize},
		},
		ArtifactScan: surface.ArtifactScanReport{
			Root:           filepath.Dir(artifactPath),
			FilesChecked:   2,
			ForbiddenPaths: []string{},
			Pass:           true,
		},
		Components: []surface.ComponentReport{
			{
				ID:     "AppModelApp",
				Type:   "examples.surface.toolkit.surface_app_model.AppModelApp",
				Bounds: surface.RectReport{X: 0, Y: 0, W: 480, H: 320},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"route":              "settings",
					"focused":            "NameField",
					"save_count":         "1",
					"pending_task":       "0",
					"history_depth":      "1",
					"redo_depth":         "0",
					"accessibility_role": "none",
				},
			},
			{
				ID:     "NameField",
				Type:   "examples.surface.toolkit.surface_app_model.NameField",
				Parent: "AppModelApp",
				Bounds: surface.RectReport{X: 32, Y: 80, W: 240, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "true",
					"buffer":             "Ada",
					"caret":              "3",
					"accessibility_role": "textbox",
				},
			},
			{
				ID:     "SaveButton",
				Type:   "examples.surface.toolkit.surface_app_model.SaveButton",
				Parent: "AppModelApp",
				Bounds: surface.RectReport{X: 32, Y: 144, W: 132, H: 44},
				Abilities: []string{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				State: map[string]string{
					"focused":            "false",
					"press_count":        "1",
					"action":             "save",
					"accessibility_role": "button",
				},
			},
		},
		Events: []surface.EventReport{
			{
				Order:           1,
				Kind:            "mouse_up",
				TargetComponent: "NameField",
				DispatchPath:    []string{"AppModelApp", "NameField"},
				Handled:         true,
				Pass:            true,
				X:               48,
				Y:               96,
				Width:           480,
				Height:          320,
				BufferSlots:     []int{5, 48, 96, 1, 0, 480, 320, 0, 0},
				BeforeState:     map[string]string{"AppModelApp.focused": ""},
				AfterState:      map[string]string{"AppModelApp.focused": "NameField"},
			},
			{
				Order:           2,
				Kind:            "text_input",
				TargetComponent: "NameField",
				DispatchPath:    []string{"AppModelApp", "NameField"},
				Handled:         true,
				Pass:            true,
				Width:           480,
				Height:          320,
				TimestampMS:     1,
				TextLen:         3,
				TextBytesHex:    "416461",
				BufferSlots:     []int{8, 0, 0, 0, 0, 480, 320, 1, 3},
				BeforeState:     map[string]string{"NameField.buffer": ""},
				AfterState:      map[string]string{"NameField.buffer": "Ada"},
			},
			{
				Order:           3,
				Kind:            "key_down",
				TargetComponent: "SaveButton",
				DispatchPath:    []string{"AppModelApp", "SaveButton"},
				Handled:         true,
				Pass:            true,
				Key:             13,
				Width:           480,
				Height:          320,
				TimestampMS:     2,
				BufferSlots:     []int{6, 0, 0, 0, 13, 480, 320, 2, 0},
				BeforeState:     map[string]string{"AppModelApp.save_count": "0"},
				AfterState:      map[string]string{"AppModelApp.save_count": "1"},
			},
		},
		Frames: []surface.FrameReport{
			{
				Order:     1,
				Width:     320,
				Height:    200,
				Stride:    1280,
				Checksum:  "1111111111111111111111111111111111111111111111111111111111111111",
				Presented: true,
			},
			{
				Order:     2,
				Width:     320,
				Height:    200,
				Stride:    1280,
				Checksum:  "2222222222222222222222222222222222222222222222222222222222222222",
				Presented: true,
			},
		},
		StateTransitions: []surface.StateTransitionReport{
			{
				Order:     1,
				Component: "AppModelApp",
				Field:     "focused",
				Before:    "",
				After:     "NameField",
				Cause:     "focus",
			},
			{
				Order:     2,
				Component: "NameField",
				Field:     "buffer",
				Before:    "",
				After:     "Ada",
				Cause:     "command.insert_text",
			},
			{
				Order:     3,
				Component: "AppModelApp",
				Field:     "route",
				Before:    "home",
				After:     "settings",
				Cause:     "command.navigate",
			},
			{
				Order:     4,
				Component: "AppModelApp",
				Field:     "pending_task",
				Before:    "1",
				After:     "0",
				Cause:     "command.async_complete",
			},
			{
				Order:     5,
				Component: "AppModelApp",
				Field:     "history_depth",
				Before:    "0",
				After:     "1",
				Cause:     "command.undoable",
			},
			{
				Order:     6,
				Component: "AppModelApp",
				Field:     "save_count",
				Before:    "0",
				After:     "1",
				Cause:     "command.save",
			},
		},
		AppModel: &surface.AppModelReport{
			Schema:                "tetra.surface.app-model.v1",
			AppModelLevel:         "explicit-command-reducer-v1",
			ReleaseScope:          "surface-v1-linux-web",
			Source:                "examples/surface/toolkit/surface_app_model.tetra",
			Module:                "lib.core.surface_app",
			UsesComponentTreeAPI:  true,
			CallerOwnedState:      true,
			ExplicitEventBindings: true,
			DeterministicReducer:  true,
			StateFields: []string{
				"route",
				"focused",
				"name_buffer",
				"save_count",
				"pending_task",
				"history_depth",
				"redo_depth",
			},
			CommandRegistry: []string{
				"focus.name",
				"text.insert",
				"nav.push.settings",
				"nav.back",
				"async.save.start",
				"async.save.complete",
				"async.save.cancel",
				"history.undo",
				"history.redo",
			},
			EventBindings: []surface.AppModelEventBindingReport{
				{
					Order:        1,
					EventOrder:   1,
					EventKind:    "mouse_up",
					Target:       "NameField",
					DispatchPath: []string{"AppModelApp", "NameField"},
					Command:      "focus.name",
					Explicit:     true,
				},
				{
					Order:        2,
					EventOrder:   2,
					EventKind:    "text_input",
					Target:       "NameField",
					DispatchPath: []string{"AppModelApp", "NameField"},
					Command:      "text.insert",
					Explicit:     true,
				},
				{
					Order:        3,
					EventOrder:   3,
					EventKind:    "key_down",
					Target:       "SaveButton",
					DispatchPath: []string{"AppModelApp", "SaveButton"},
					Command:      "async.save.start",
					Explicit:     true,
				},
			},
			CommandDispatches: []surface.AppModelCommandDispatchReport{
				{
					Order:       1,
					EventOrder:  1,
					Command:     "focus.name",
					Kind:        "focus",
					Target:      "NameField",
					Handled:     true,
					BeforeState: map[string]string{"focused": ""},
					AfterState:  map[string]string{"focused": "NameField"},
				},
				{
					Order:        2,
					EventOrder:   2,
					Command:      "text.insert",
					Kind:         "edit",
					Target:       "NameField",
					Handled:      true,
					BeforeState:  map[string]string{"name_buffer": ""},
					AfterState:   map[string]string{"name_buffer": "Ada"},
					Reversible:   true,
					HistoryIndex: 1,
				},
				{
					Order:       3,
					EventOrder:  3,
					Command:     "async.save.start",
					Kind:        "async_start",
					Target:      "SaveButton",
					Handled:     true,
					BeforeState: map[string]string{"pending_task": "0"},
					AfterState:  map[string]string{"pending_task": "1"},
					AsyncTaskID: "save-1",
				},
				{
					Order:       4,
					Command:     "async.save.complete",
					Kind:        "async_complete",
					Target:      "AppModelApp",
					Handled:     true,
					BeforeState: map[string]string{"pending_task": "1", "save_count": "0"},
					AfterState:  map[string]string{"pending_task": "0", "save_count": "1"},
					AsyncTaskID: "save-1",
				},
			},
			NavigationTransitions: []surface.AppModelNavigationReport{
				{
					Order:       1,
					Command:     "nav.push.settings",
					Operation:   "push",
					BeforeRoute: "home",
					AfterRoute:  "settings",
					StackBefore: []string{"home"},
					StackAfter:  []string{"home", "settings"},
				},
				{
					Order:       2,
					Command:     "nav.back",
					Operation:   "back",
					BeforeRoute: "settings",
					AfterRoute:  "home",
					StackBefore: []string{"home", "settings"},
					StackAfter:  []string{"home"},
				},
				{
					Order:             3,
					Command:           "nav.back",
					Operation:         "back",
					BeforeRoute:       "home",
					AfterRoute:        "home",
					StackBefore:       []string{"home"},
					StackAfter:        []string{"home"},
					UnderflowRejected: true,
				},
			},
			FocusScopeTransitions: []surface.AppModelFocusScopeReport{
				{Order: 1, Scope: "main", BeforeFocus: "", AfterFocus: "NameField"},
				{
					Order:       2,
					Scope:       "dialog",
					BeforeFocus: "DialogCancel",
					AfterFocus:  "DialogConfirm",
					Wrapped:     true,
					ModalTrap:   true,
				},
			},
			AsyncTasks: []surface.AppModelAsyncTaskReport{
				{
					ID:          "save-1",
					Command:     "async.save.start",
					Operation:   "start",
					Status:      "pending",
					BeforeState: map[string]string{"pending_task": "0"},
					AfterState:  map[string]string{"pending_task": "1"},
				},
				{
					ID:              "save-1",
					Command:         "async.save.complete",
					Operation:       "complete",
					Status:          "completed",
					BeforeState:     map[string]string{"pending_task": "1"},
					AfterState:      map[string]string{"pending_task": "0"},
					CompletionOrder: 4,
				},
				{
					ID:          "save-2",
					Command:     "async.save.cancel",
					Operation:   "cancel",
					Status:      "canceled",
					BeforeState: map[string]string{"pending_task": "1", "save_count": "1"},
					AfterState:  map[string]string{"pending_task": "0", "save_count": "1"},
					Canceled:    true,
				},
			},
			UndoRedoTransitions: []surface.AppModelUndoRedoReport{
				{
					Order:               1,
					Command:             "text.insert",
					HistoryIndex:        1,
					Operation:           "record",
					Before:              "",
					After:               "Ada",
					MatchedHistoryEntry: true,
					Applied:             true,
				},
				{
					Order:               2,
					Command:             "history.undo",
					HistoryIndex:        1,
					Operation:           "undo",
					Before:              "Ada",
					After:               "",
					MatchedHistoryEntry: true,
					Applied:             true,
				},
				{
					Order:               3,
					Command:             "history.redo",
					HistoryIndex:        1,
					Operation:           "redo",
					Before:              "",
					After:               "Ada",
					MatchedHistoryEntry: true,
					Applied:             true,
				},
			},
			NegativeGuards: surface.AppModelNegativeGuardsReport{
				NoHiddenAppState:              true,
				NoReactHooks:                  true,
				NoDOMEventModel:               true,
				NoUserJS:                      true,
				NoPlatformWidgets:             true,
				AsyncCancelNoMutation:         true,
				NavigationUnderflowRejected:   true,
				FocusScopeEscapeRejected:      true,
				UndoRedoRequiresHistory:       true,
				CommandWithoutBindingRejected: true,
			},
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
			{
				Name: "app model explicit event-to-command binding",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{
				Name: "app model deterministic command reducer",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{Name: "app model navigation stack", Kind: "positive", Ran: true, Pass: true},
			{Name: "app model focus scope modal trap", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "app model async completion cancellation boundary",
				Kind: "positive",
				Ran:  true,
				Pass: true,
			},
			{Name: "app model undo redo history", Kind: "positive", Ran: true, Pass: true},
			{
				Name: "app model no React hooks DOM event model hidden JS state",
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
		},
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal app-model report: %v", err)
	}
	return raw
}

func validLinuxAppShellReleaseRuntimeReportJSON(
	t *testing.T,
	artifactDir string,
	mutate func(map[string]any),
) []byte {
	t.Helper()
	componentPath, componentSHA, componentSize := writeNamedSurfaceArtifactFixture(
		t,
		artifactDir,
		"surface-linux-app-shell-notes",
		[]byte("surface linux app-shell notes fixture\n"),
		0o755,
	)
	hostTracePath, hostTraceSHA, hostTraceSize := writeNamedSurfaceArtifactFixture(
		t,
		artifactDir,
		"surface-linux-app-shell-host-trace.json",
		[]byte(
			("{\"schema\":\"tetra.surface.linux-app-shell-host-trace.v1\","+
				"\"source\":"+
				"\"examples/surface/toolkit/surface_linux_app_shell_notes.tetr"+
				"a\",\"lifecycle\":[\"open\",\"close\",\"reopen\"]}")+"\n",
		),
		0o644,
	)
	windowTracePath, windowTraceSHA, windowTraceSize := writeNamedSurfaceArtifactFixture(
		t,
		artifactDir,
		"surface-linux-app-shell-window-trace.json",
		[]byte(
			("{\"schema\":\"tetra.surface.linux-app-shell-window-trace.v1\","+
				"\"windows\":[\"notes-main\",\"notes-inspector\"],"+
				"\"dpi_scale_milli\":1250}")+"\n",
		),
		0o644,
	)
	accessibilityProbePath, accessibilityProbeSHA, accessibilityProbeSize :=
		writeNamedSurfaceArtifactFixture(
			t,
			artifactDir,
			"surface-linux-accessibility-probe.json",
			[]byte(
				`{"schema":"tetra.surface.linux-accessibility-platform-probe.v1","platform_export":true}`+"\n",
			),
			0o644,
		)

	report := map[string]any{
		"schema":         "tetra.surface.runtime.v1",
		"status":         "pass",
		"target":         "linux-x64",
		"host":           "linux-x64",
		"runtime":        "surface-linux-x64",
		"surface_schema": "tetra.surface.v1",
		"host_abi":       "tetra.surface.host-abi.v1",
		"host_evidence": map[string]any{
			"level":                        "linux-x64-release-window-v1",
			"backend":                      "wayland-shm-rgba-release-v1",
			"framebuffer":                  true,
			"real_window":                  true,
			"native_input":                 true,
			"text_input":                   true,
			"clipboard":                    true,
			"composition":                  true,
			"accessibility_bridge":         true,
			"user_facing_platform_widgets": false,
		},
		"source": "examples/surface/toolkit/surface_linux_app_shell_notes.tetra",
		"processes": []any{
			map[string]any{
				"name": "tetra build",
				"kind": "build",
				"path": ("tetra build --target linux-x64 " +
					"examples/surface/toolkit/surface_linux_app_shell_notes.tetra" +
					" -o ") + componentPath,
				"ran":       true,
				"pass":      true,
				"exit_code": 0,
			},
			map[string]any{
				"name":               "surface component app",
				"kind":               "app",
				"path":               componentPath,
				"ran":                true,
				"pass":               true,
				"exit_code":          1,
				"expected_exit_code": 1,
			},
			map[string]any{
				"name": "surface linux-x64 real-window probe",
				"kind": "app",
				"path": filepath.Join(
					artifactDir,
					"surface-linux-app-shell-window-probe",
				),
				"ran":                true,
				"pass":               true,
				"exit_code":          42,
				"expected_exit_code": 42,
			},
			map[string]any{
				"name":      "surface linux app-shell host trace",
				"kind":      "runtime",
				"path":      hostTracePath,
				"ran":       true,
				"pass":      true,
				"exit_code": 0,
			},
			map[string]any{
				"name":      "surface linux app-shell window trace",
				"kind":      "runtime",
				"path":      windowTracePath,
				"ran":       true,
				"pass":      true,
				"exit_code": 0,
			},
			map[string]any{
				"name":      "surface linux-x64 release clipboard harness",
				"kind":      "runtime",
				"path":      filepath.Join(artifactDir, "surface-linux-clipboard-harness.json"),
				"ran":       true,
				"pass":      true,
				"exit_code": 0,
			},
			map[string]any{
				"name":      "surface linux-x64 release composition harness",
				"kind":      "runtime",
				"path":      filepath.Join(artifactDir, "surface-linux-composition-harness.json"),
				"ran":       true,
				"pass":      true,
				"exit_code": 0,
			},
			map[string]any{
				"name":      "surface linux accessibility platform probe",
				"kind":      "runtime",
				"path":      accessibilityProbePath,
				"ran":       true,
				"pass":      true,
				"exit_code": 0,
			},
			map[string]any{
				"name":      "surface linux-x64 runtime",
				"kind":      "runtime",
				"path":      "tools/cmd/surface-runtime-smoke --mode linux-x64-release-app-shell",
				"ran":       true,
				"pass":      true,
				"exit_code": 0,
			},
		},
		"artifacts": []any{
			map[string]any{
				"kind":   "component-app",
				"path":   componentPath,
				"sha256": componentSHA,
				"size":   componentSize,
			},
			map[string]any{
				"kind":   "linux-app-shell-host-trace",
				"path":   hostTracePath,
				"sha256": hostTraceSHA,
				"size":   hostTraceSize,
			},
			map[string]any{
				"kind":   "linux-app-shell-window-trace",
				"path":   windowTracePath,
				"sha256": windowTraceSHA,
				"size":   windowTraceSize,
			},
			map[string]any{
				"kind":   "linux-accessibility-platform-probe",
				"path":   accessibilityProbePath,
				"sha256": accessibilityProbeSHA,
				"size":   accessibilityProbeSize,
			},
		},
		"artifact_scan": map[string]any{
			"root":            artifactDir,
			"files_checked":   4,
			"forbidden_paths": []any{},
			"pass":            true,
		},
		"components": []any{
			map[string]any{
				"id":     "NotesShellApp",
				"type":   "examples.surface.toolkit.surface_linux_app_shell_notes.NotesShellApp",
				"bounds": map[string]any{"x": 0, "y": 0, "w": 720, "h": 540},
				"abilities": []any{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				"state": map[string]any{
					"open_windows":       "2",
					"focused_window":     "notes-main",
					"accessibility_role": "application",
				},
			},
			map[string]any{
				"id":     "NotesMainWindow",
				"type":   "examples.surface.toolkit.surface_linux_app_shell_notes.NotesMainWindow",
				"parent": "NotesShellApp",
				"bounds": map[string]any{"x": 0, "y": 0, "w": 560, "h": 420},
				"abilities": []any{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				"state": map[string]any{
					"title":              "Notes",
					"lifecycle":          "reopened",
					"dpi_scale_milli":    "1250",
					"cursor":             "text",
					"accessibility_role": "document",
				},
			},
			map[string]any{
				"id":     "NotesInspectorWindow",
				"type":   "examples.surface.toolkit.surface_linux_app_shell_notes.NotesInspectorWindow",
				"parent": "NotesShellApp",
				"bounds": map[string]any{"x": 24, "y": 24, "w": 320, "h": 240},
				"abilities": []any{
					"measure",
					"layout",
					"draw",
					"event",
					"focus",
					"text",
					"accessibility",
				},
				"state": map[string]any{
					"title":              "Inspector",
					"lifecycle":          "open",
					"dpi_scale_milli":    "1000",
					"cursor":             "pointer",
					"accessibility_role": "panel",
				},
			},
		},
		"events": []any{
			map[string]any{
				"order":            1,
				"kind":             "mouse_up",
				"target_component": "NotesMainWindow",
				"dispatch_path":    []any{"NotesShellApp", "NotesMainWindow"},
				"handled":          true,
				"pass":             true,
				"x":                40,
				"y":                72,
				"key":              0,
				"width":            560,
				"height":           420,
				"timestamp_ms":     0,
				"buffer_slots":     []any{5, 40, 72, 1, 0, 560, 420, 0, 0},
				"before_state":     map[string]any{"NotesShellApp.focused_window": ""},
				"after_state":      map[string]any{"NotesShellApp.focused_window": "notes-main"},
			},
			map[string]any{
				"order":            2,
				"kind":             "key_down",
				"target_component": "NotesMainWindow",
				"dispatch_path":    []any{"NotesShellApp", "NotesMainWindow"},
				"handled":          true,
				"pass":             true,
				"x":                0,
				"y":                0,
				"key":              78,
				"width":            560,
				"height":           420,
				"timestamp_ms":     2,
				"buffer_slots":     []any{6, 0, 0, 1, 78, 560, 420, 2, 0},
				"before_state":     map[string]any{"NotesMainWindow.shortcut": ""},
				"after_state":      map[string]any{"NotesMainWindow.shortcut": "new-note"},
			},
			map[string]any{
				"order":            3,
				"kind":             "text_input",
				"target_component": "NotesMainWindow",
				"dispatch_path":    []any{"NotesShellApp", "NotesMainWindow"},
				"handled":          true,
				"pass":             true,
				"x":                0,
				"y":                0,
				"key":              0,
				"width":            560,
				"height":           420,
				"timestamp_ms":     3,
				"text_len":         5,
				"text_bytes_hex":   "4e6f746573",
				"buffer_slots":     []any{8, 0, 0, 0, 0, 560, 420, 3, 5},
				"before_state":     map[string]any{"NotesMainWindow.buffer": ""},
				"after_state":      map[string]any{"NotesMainWindow.buffer": "Notes"},
			},
			map[string]any{
				"order":            4,
				"kind":             "resize",
				"target_component": "NotesMainWindow",
				"dispatch_path":    []any{"NotesShellApp", "NotesMainWindow"},
				"handled":          true,
				"pass":             true,
				"width":            720,
				"height":           540,
				"timestamp_ms":     4,
				"buffer_slots":     []any{7, 0, 0, 0, 0, 720, 540, 4, 0},
				"before_state": map[string]any{
					"NotesMainWindow.size": "560x420",
					"NotesMainWindow.dpi":  "1000",
				},
				"after_state": map[string]any{
					"NotesMainWindow.size": "720x540",
					"NotesMainWindow.dpi":  "1250",
				},
			},
			map[string]any{
				"order":            5,
				"kind":             "close",
				"target_component": "NotesInspectorWindow",
				"dispatch_path":    []any{"NotesShellApp", "NotesInspectorWindow"},
				"handled":          true,
				"pass":             true,
				"width":            320,
				"height":           240,
				"timestamp_ms":     5,
				"buffer_slots":     []any{9, 0, 0, 0, 0, 320, 240, 5, 0},
				"before_state":     map[string]any{"NotesInspectorWindow.open": "true"},
				"after_state":      map[string]any{"NotesInspectorWindow.open": "false"},
			},
		},
		"frames": []any{
			map[string]any{
				"order":     1,
				"width":     400,
				"height":    240,
				"stride":    1600,
				"checksum":  "1111111111111111111111111111111111111111111111111111111111111111",
				"presented": true,
			},
			map[string]any{
				"order":     5,
				"width":     560,
				"height":    420,
				"stride":    2240,
				"checksum":  "2222222222222222222222222222222222222222222222222222222222222222",
				"presented": true,
			},
			map[string]any{
				"order":     6,
				"width":     720,
				"height":    540,
				"stride":    2880,
				"checksum":  "3333333333333333333333333333333333333333333333333333333333333333",
				"presented": true,
			},
		},
		"state_transitions": []any{
			map[string]any{
				"order":     1,
				"component": "NotesShellApp",
				"field":     "focused_window",
				"before":    "",
				"after":     "notes-main",
				"cause":     "lifecycle.open",
			},
			map[string]any{
				"order":     2,
				"component": "NotesInspectorWindow",
				"field":     "open",
				"before":    "true",
				"after":     "false",
				"cause":     "lifecycle.close",
			},
			map[string]any{
				"order":     3,
				"component": "NotesMainWindow",
				"field":     "size",
				"before":    "560x420",
				"after":     "720x540",
				"cause":     "resize",
			},
		},
		"cases": []any{
			map[string]any{
				"name": "pure Tetra component app",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "host-provided pointer event dispatch",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "host event buffer poll_event",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "pre/post event frame sequence",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "component hierarchy dispatch",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "component text input scalar dispatch",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "host text payload buffer",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "component focus dispatch",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "component accessibility metadata",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "no legacy UI sidecar artifacts",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "state transition",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name":           "reject legacy UI evidence",
				"kind":           "negative",
				"ran":            true,
				"pass":           true,
				"expected_error": "legacy UI evidence rejected",
			},
			map[string]any{
				"name": "linux-x64 real-window surface",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "linux-x64 native input event pump",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "linux-x64 real-window resize event",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "linux-x64 real-window close event",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "linux release real window presented frame",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "linux release accessibility bridge probe",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "linux app-shell v1 schema",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "linux app-shell lifecycle open close reopen",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "linux app-shell multi-window notes reference",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "linux app-shell resize dpi cursor trace",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "linux app-shell clipboard ime accessibility adapters",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "linux app-shell file dialog notification blocked-pass",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "linux app-shell electron feature ledger",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "linux app-shell dialog file picker tray blocked-pass",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "linux app-shell crash error report scoped adapters",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "surface security permission model default deny filesystem network",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "surface security app-shell feature policy enforcement",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "surface security IPC process boundary schema validation",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "surface security asset font image local hash policy",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name":           "surface security network asset fetch rejected",
				"kind":           "negative",
				"ran":            true,
				"pass":           true,
				"expected_error": "network asset fetch rejected",
			},
			map[string]any{
				"name": "surface security notification dialog permission nonclaims",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "surface performance budget startup first frame",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "surface performance budget frame p50 p95",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "surface performance budget memory cache framebuffer rss",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "surface performance budget binary size",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name": "surface performance budget cpu power proxy",
				"kind": "positive",
				"ran":  true,
				"pass": true,
			},
			map[string]any{
				"name":           "surface performance budget faster than electron nonclaim",
				"kind":           "negative",
				"ran":            true,
				"pass":           true,
				"expected_error": "unsupported faster than electron claim rejected",
			},
			map[string]any{
				"name":           "linux app-shell rejects GTK Qt native widget UI",
				"kind":           "negative",
				"ran":            true,
				"pass":           true,
				"expected_error": "native widget UI rejected",
			},
			map[string]any{
				"name":           "linux app-shell no Electron React DOM application scripting",
				"kind":           "negative",
				"ran":            true,
				"pass":           true,
				"expected_error": "runtime substitute rejected",
			},
		},
		"linux_app_shell": linuxAppShellEvidenceMap(
			hostTracePath,
			windowTracePath,
			accessibilityProbePath,
		),
		"security_permissions": linuxAppShellSecurityPermissionsMap(
			p16LinuxAppShellRuntimeFeaturesForTest(),
		),
		"surface_performance_budget": linuxAppShellPerformanceBudgetMap(
			"linux-x64",
			"surface-linux-x64",
			"examples/surface/toolkit/surface_linux_app_shell_notes.tetra",
			componentPath,
			componentSize,
		),
	}
	if mutate != nil {
		mutate(report)
	}
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal linux app-shell report: %v", err)
	}
	return raw
}

func linuxAppShellEvidenceMap(
	hostTracePath string,
	windowTracePath string,
	accessibilityProbePath string,
) map[string]any {
	return map[string]any{
		"schema":           "tetra.surface.linux-app-shell.v1",
		"app_shell_level":  "linux-app-shell-subset-v1",
		"release_scope":    "surface-v1-linux-web",
		"source":           "examples/surface/toolkit/surface_linux_app_shell_notes.tetra",
		"module":           "lib.core.surface_app_shell",
		"host_adapter":     "wayland-shm-rgba-release-v1",
		"production_claim": true,
		"experimental":     false,
		"window_lifecycle": []any{
			map[string]any{
				"order":      1,
				"window_id":  "notes-main",
				"operation":  "open",
				"host_trace": true,
				"pass":       true,
			},
			map[string]any{
				"order":      2,
				"window_id":  "notes-inspector",
				"operation":  "open",
				"host_trace": true,
				"pass":       true,
			},
			map[string]any{
				"order":      3,
				"window_id":  "notes-inspector",
				"operation":  "close",
				"host_trace": true,
				"pass":       true,
			},
			map[string]any{
				"order":      4,
				"window_id":  "notes-inspector",
				"operation":  "reopen",
				"host_trace": true,
				"pass":       true,
			},
		},
		"windows": []any{
			map[string]any{
				"id":              "notes-main",
				"title":           "Notes",
				"role":            "primary",
				"block_root":      "NotesMainWindow",
				"real_window":     true,
				"presented":       true,
				"width":           720,
				"height":          540,
				"dpi_scale_milli": 1250,
			},
			map[string]any{
				"id":              "notes-inspector",
				"title":           "Inspector",
				"role":            "secondary",
				"block_root":      "NotesInspectorWindow",
				"real_window":     true,
				"presented":       true,
				"width":           320,
				"height":          240,
				"dpi_scale_milli": 1000,
			},
		},
		"resize_dpi": []any{
			map[string]any{
				"window_id":       "notes-main",
				"operation":       "resize",
				"before_width":    560,
				"before_height":   420,
				"after_width":     720,
				"after_height":    540,
				"dpi_scale_milli": 1250,
				"host_trace":      true,
				"pass":            true,
			},
			map[string]any{
				"window_id":       "notes-main",
				"operation":       "dpi_scale",
				"before_width":    720,
				"before_height":   540,
				"after_width":     720,
				"after_height":    540,
				"dpi_scale_milli": 1250,
				"host_trace":      true,
				"pass":            true,
			},
		},
		"cursor_transitions": []any{
			map[string]any{
				"window_id":  "notes-main",
				"cursor":     "pointer",
				"target":     "NotesMainWindow",
				"host_trace": true,
				"pass":       true,
			},
			map[string]any{
				"window_id":  "notes-main",
				"cursor":     "text",
				"target":     "NotesMainWindow",
				"host_trace": true,
				"pass":       true,
			},
			map[string]any{
				"window_id":  "notes-main",
				"cursor":     "resize",
				"target":     "NotesMainWindow",
				"host_trace": true,
				"pass":       true,
			},
		},
		"clipboard": map[string]any{
			"level":         "clipboard-text-v1",
			"host_trace":    true,
			"artifact_kind": "linux-app-shell-host-trace",
			"read":          true,
			"write":         true,
			"pass":          true,
		},
		"ime": map[string]any{
			"level":         "composition-baseline-v1",
			"host_trace":    true,
			"artifact_kind": "linux-app-shell-host-trace",
			"start":         true,
			"update":        true,
			"commit":        true,
			"cancel":        true,
			"pass":          true,
		},
		"accessibility": map[string]any{
			"level":           "platform-bridge-v1",
			"host_trace":      true,
			"artifact_kind":   "linux-accessibility-platform-probe",
			"metadata_tree":   true,
			"platform_export": true,
			"pass":            true,
		},
		"shell_features": p16LinuxAppShellRuntimeFeaturesForTest(),
		"host_traces": []any{
			map[string]any{
				"name":          "lifecycle",
				"artifact_kind": "linux-app-shell-host-trace",
				"path":          hostTracePath,
				"pass":          true,
			},
			map[string]any{
				"name":          "windows",
				"artifact_kind": "linux-app-shell-window-trace",
				"path":          windowTracePath,
				"pass":          true,
			},
			map[string]any{
				"name":          "accessibility",
				"artifact_kind": "linux-accessibility-platform-probe",
				"path":          accessibilityProbePath,
				"pass":          true,
			},
		},
		"negative_guards": map[string]any{
			"no_gtk":              true,
			"no_qt":               true,
			"no_native_widgets":   true,
			"no_electron_runtime": true,
			"no_react_runtime":    true,
			"no_dom_ui":           true,
			"no_user_js":          true,
			"no_platform_widgets": true,
		},
	}
}

func p16LinuxAppShellRuntimeFeaturesForTest() []any {
	return []any{
		map[string]any{
			"name":                "app_menu",
			"status":              "scoped_adapter",
			"claimed":             true,
			"host_trace":          true,
			"blocked_reason":      "",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "window_lifecycle",
			"status":              "target_evidenced",
			"claimed":             true,
			"host_trace":          true,
			"blocked_reason":      "",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "multi_window",
			"status":              "target_evidenced",
			"claimed":             true,
			"host_trace":          true,
			"blocked_reason":      "",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "clipboard",
			"status":              "target_evidenced",
			"claimed":             true,
			"host_trace":          true,
			"blocked_reason":      "",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "ime",
			"status":              "target_evidenced",
			"claimed":             true,
			"host_trace":          true,
			"blocked_reason":      "",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "accessibility_bridge",
			"status":              "target_evidenced",
			"claimed":             true,
			"host_trace":          true,
			"blocked_reason":      "",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "crash_recovery",
			"status":              "scoped_adapter",
			"claimed":             true,
			"host_trace":          true,
			"blocked_reason":      "",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "error_report",
			"status":              "scoped_adapter",
			"claimed":             true,
			"host_trace":          true,
			"blocked_reason":      "",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "dialog",
			"status":              "blocked_pass",
			"claimed":             false,
			"host_trace":          true,
			"blocked_reason":      "target host dialog unavailable in CI",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "file_dialog",
			"status":              "blocked_pass",
			"claimed":             false,
			"host_trace":          true,
			"blocked_reason":      "target host file dialog unavailable in CI",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "file_picker",
			"status":              "blocked_pass",
			"claimed":             false,
			"host_trace":          true,
			"blocked_reason":      "target host file picker unavailable in CI",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "notification",
			"status":              "blocked_pass",
			"claimed":             false,
			"host_trace":          true,
			"blocked_reason":      "target host notification unavailable in CI",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "tray",
			"status":              "blocked_pass",
			"claimed":             false,
			"host_trace":          true,
			"blocked_reason":      "target host tray unavailable in CI",
			"no_native_widget_ui": true,
			"pass":                true,
		},
		map[string]any{
			"name":                "deep_link",
			"status":              "blocked_pass",
			"claimed":             false,
			"host_trace":          true,
			"blocked_reason":      "target host deep link unavailable in CI",
			"no_native_widget_ui": true,
			"pass":                true,
		},
	}
}

func linuxAppShellSecurityPermissionsMap(features []any) map[string]any {
	capabilities := make([]any, 0, len(features))
	for _, feature := range features {
		row := feature.(map[string]any)
		name := row["name"].(string)
		status, allowed := securityCapabilityStatusForRuntimeTest(row["status"].(string))
		blockedReason, _ := row["blocked_reason"].(string)
		capabilities = append(capabilities, map[string]any{
			"name":               name,
			"source_feature":     name,
			"status":             status,
			"allowed":            allowed,
			"capability_checked": true,
			"host_trace":         true,
			"policy":             "surface-app-shell-capability-policy-v1",
			"evidence":           "linux-app-shell-host-trace",
			"blocked_reason":     blockedReason,
			"pass":               true,
		})
	}
	return map[string]any{
		"schema":                        "tetra.surface.security-permission.v1",
		"model":                         "surface-security-permission-v1",
		"release_scope":                 "surface-v1-linux-web",
		"source":                        "examples/surface/toolkit/surface_linux_app_shell_notes.tetra",
		"app_shell_features":            "electron-feature-ledger-v1",
		"production_claim":              true,
		"experimental":                  false,
		"default_deny":                  true,
		"shell_feature_policy_enforced": true,
		"capabilities":                  capabilities,
		"permissions": []any{
			map[string]any{
				"name":               "filesystem",
				"status":             "denied",
				"allowed":            false,
				"capability_checked": true,
				"blocked_reason":     "ambient filesystem denied in default template",
				"evidence":           "default-deny-policy",
				"pass":               true,
			},
			map[string]any{
				"name":               "network",
				"status":             "denied",
				"allowed":            false,
				"capability_checked": true,
				"blocked_reason":     "ambient network denied in default template",
				"evidence":           "default-deny-policy",
				"pass":               true,
			},
			map[string]any{
				"name":               "clipboard",
				"status":             "allowed_with_policy",
				"allowed":            true,
				"capability_checked": true,
				"blocked_reason":     "",
				"evidence":           "linux-app-shell-host-trace",
				"pass":               true,
			},
			map[string]any{
				"name":               "notifications",
				"status":             "denied",
				"allowed":            false,
				"capability_checked": true,
				"blocked_reason":     "notification target evidence absent",
				"evidence":           "blocked-pass-nonclaim",
				"pass":               true,
			},
			map[string]any{
				"name":               "dialogs",
				"status":             "denied",
				"allowed":            false,
				"capability_checked": true,
				"blocked_reason":     "dialog target evidence absent",
				"evidence":           "blocked-pass-nonclaim",
				"pass":               true,
			},
			map[string]any{
				"name":               "shell_open_url",
				"status":             "denied",
				"allowed":            false,
				"capability_checked": true,
				"blocked_reason":     "shell open-url denied in default template",
				"evidence":           "default-deny-policy",
				"pass":               true,
			},
		},
		"process_boundaries": []any{
			map[string]any{
				"name":               "surface_app_to_host_abi",
				"schema_checked":     true,
				"capability_checked": true,
				"user_js":            false,
				"node_integration":   false,
				"electron_runtime":   false,
				"pass":               true,
			},
			map[string]any{
				"name":               "linux_app_shell_host_adapter",
				"schema_checked":     true,
				"capability_checked": true,
				"user_js":            false,
				"node_integration":   false,
				"electron_runtime":   false,
				"pass":               true,
			},
			map[string]any{
				"name":               "browser_canvas_host",
				"schema_checked":     true,
				"capability_checked": true,
				"user_js":            false,
				"node_integration":   false,
				"electron_runtime":   false,
				"pass":               true,
			},
		},
		"asset_safety": []any{
			map[string]any{
				"kind":                  "font",
				"local_only":            true,
				"sha256_required":       true,
				"size_limit_bytes":      1048576,
				"network_fetch_allowed": false,
				"parser":                "bounded-font-metadata-v1",
				"bounds_checked":        true,
				"pass":                  true,
			},
			map[string]any{
				"kind":                  "image",
				"local_only":            true,
				"sha256_required":       true,
				"size_limit_bytes":      2097152,
				"network_fetch_allowed": false,
				"parser":                "bounded-image-header-v1",
				"bounds_checked":        true,
				"pass":                  true,
			},
			map[string]any{
				"kind":                  "icon",
				"local_only":            true,
				"sha256_required":       true,
				"size_limit_bytes":      262144,
				"network_fetch_allowed": false,
				"parser":                "bounded-icon-header-v1",
				"bounds_checked":        true,
				"pass":                  true,
			},
		},
		"unsupported_claims": []any{
			"unrestricted-filesystem",
			"unrestricted-network",
			"native-permission-prompts",
			"production-notifications",
			"production-dialogs",
			"remote-asset-fetch",
			"electron-node-integration",
		},
		"negative_guards": map[string]any{
			"no_ambient_filesystem":                          true,
			"no_ambient_network":                             true,
			"no_shell_feature_bypass":                        true,
			"no_permissionless_clipboard":                    true,
			"no_notification_dialog_without_target_evidence": true,
			"no_network_asset_fetch":                         true,
			"no_untrusted_font_image_decode":                 true,
			"no_electron_node_integration":                   true,
			"no_user_js_app_logic":                           true,
			"no_dom_app_ui_tree":                             true,
		},
	}
}

func linuxAppShellPerformanceBudgetMap(
	target string,
	runtimeName string,
	source string,
	artifactPath string,
	artifactSize int64,
) map[string]any {
	return map[string]any{
		"schema":            "tetra.surface.performance-budget.v1",
		"model":             "surface-performance-budget-v1",
		"release_scope":     "surface-v1-linux-web",
		"source":            source,
		"target":            target,
		"runtime":           runtimeName,
		"production_claim":  true,
		"experimental":      false,
		"git_head":          "0123456789abcdef0123456789abcdef01234567",
		"performance_claim": "none",
		"startup": map[string]any{
			"launch_to_first_frame_ms": 18,
			"budget_ms":                250,
			"trace":                    "local-startup-trace-v1",
			"pass":                     true,
		},
		"frame": map[string]any{
			"frame_count":     3,
			"p50_build_ms":    4,
			"p95_build_ms":    7,
			"p50_present_ms":  3,
			"p95_present_ms":  6,
			"budget_ms":       16,
			"idle_loop_count": 24,
			"work_loop_count": 6,
			"pass":            true,
		},
		"scene": map[string]any{
			"block_count":            3,
			"recipe_expansion_count": 0,
			"paint_command_count":    10,
			"layout_pass_count":      4,
			"text_run_count":         2,
		},
		"memory": map[string]any{
			"glyph_cache_bytes":        4096,
			"asset_cache_bytes":        5376,
			"layout_cache_bytes":       4096,
			"paint_cache_bytes":        10240,
			"framebuffer_peak_bytes":   1555200,
			"framebuffer_total_bytes":  2880000,
			"rss_measured":             false,
			"peak_rss_bytes":           0,
			"allocation_count":         42,
			"allocation_bytes":         2903808,
			"bounded_caches":           true,
			"unbounded_cache_rejected": true,
			"pass":                     true,
		},
		"binary": map[string]any{
			"artifact_path": artifactPath,
			"size_bytes":    artifactSize,
			"budget_bytes":  16777216,
			"pass":          true,
		},
		"cpu_power_proxy": map[string]any{
			"idle_loop_count":     24,
			"work_loop_count":     6,
			"idle_frame_count":    2,
			"work_frame_count":    1,
			"real_power_measured": false,
			"pass":                true,
		},
		"cache": map[string]any{
			"glyph_cache_budget_bytes":  65536,
			"asset_cache_budget_bytes":  65536,
			"layout_cache_budget_bytes": 65536,
			"paint_cache_budget_bytes":  65536,
			"total_cache_bytes":         23808,
			"total_cache_budget_bytes":  262144,
			"eviction":                  "bounded-lru",
			"pass":                      true,
		},
		"methodology": map[string]any{
			"kind":                "local-deterministic-budget-v1",
			"electron_comparison": "none",
			"official_benchmark":  false,
			"cross_machine":       false,
			"fair_comparison_required_for_electron_claim": true,
		},
		"unsupported_claims": []any{
			"faster-than-electron",
			"lower-power-than-electron",
			"official-benchmark-result",
			"cross-machine-benchmark",
			"electron-parity-performance",
		},
		"negative_guards": map[string]any{
			"bounded_caches":                true,
			"unbounded_cache_rejected":      true,
			"stale_report_rejected":         true,
			"no_faster_than_electron_claim": true,
			"no_benchmark_parity_claim":     true,
			"peak_memory_field_required":    true,
			"no_official_benchmark_claim":   true,
		},
	}
}

func securityCapabilityStatusForRuntimeTest(featureStatus string) (string, bool) {
	switch featureStatus {
	case "target_evidenced", "scoped_adapter":
		return "allowed_with_policy", true
	case "blocked_pass":
		return "blocked_nonclaim", false
	default:
		return "unknown", false
	}
}

func withoutLinuxAppShellRuntimeFeature(features []any, name string) []any {
	filtered := make([]any, 0, len(features))
	for _, feature := range features {
		row := feature.(map[string]any)
		if row["name"] == name {
			continue
		}
		filtered = append(filtered, feature)
	}
	return filtered
}

func headlessReleaseRuntimeReportJSON(
	artifactPath string,
	artifactSHA string,
	artifactSize int64,
	tracePath string,
	traceSHA string,
	traceSize int64,
) []byte {
	return validSurfaceRuntimeReportJSON(
		artifactPath,
		artifactSHA,
		artifactSize,
		tracePath,
		traceSHA,
		traceSize,
	)
}

func validWASM32WebSurfaceRuntimeReportJSON(
	wasmPath string,
	wasmSHA string,
	wasmSize int64,
	loaderPath string,
	loaderSHA string,
	loaderSize int64,
	tracePath string,
	traceSHA string,
	traceSize int64,
) []byte {
	raw := string(
		validSurfaceRuntimeReportJSON(wasmPath, wasmSHA, wasmSize, tracePath, traceSHA, traceSize),
	)
	replacements := []struct {
		old string
		new string
	}{
		{old: `"target": "headless"`, new: `"target": "wasm32-web"`},
		{old: `"runtime": "surface-headless"`, new: `"runtime": "surface-wasm32-web"`},
		{
			old: ("\"host_evidence\": {\"level\":\"deterministic-headless\"," +
				"\"backend\":\"software-rgba\",\"framebuffer\":true,\"real_window\":" +
				"false,\"native_input\":false,\"user_facing_platform_widgets\":" +
				"false}"),
			new: ("\"host_evidence\": {\"level\":" +
				"\"wasm32-web-compiler-owned-loader\",\"backend\":" +
				"\"node-surface-host\",\"framebuffer\":true,\"real_window\":false," +
				"\"native_input\":false,\"user_facing_platform_widgets\":false}"),
		},
		{
			old: `"tetra build --target linux-x64 examples/surface/runtime/surface_counter.tetra -o `,
			new: `"tetra build --target wasm32-web examples/surface/runtime/surface_counter.tetra -o `,
		},
		{
			old: `"name":"surface component app","kind":"app","path":` + jsonString(
				wasmPath,
			) + `,"ran":true,"pass":true,"exit_code":1,"expected_exit_code":1}`,
			new: `"name":"surface wasm32-web component app","kind":"app","path":` + jsonString(
				"node scripts/tools/web_run_module.mjs --surface-trace "+tracePath+" "+wasmPath,
			) + `,"ran":true,"pass":true,"exit_code":1,"expected_exit_code":1},
    {"name":"surface wasm32-web import validator","kind":"runtime","path":` + jsonString(
				"go run ./tools/cmd/validate-wasm-imports --target wasm32-web "+wasmPath,
			) + `,"ran":true,"pass":true,"exit_code":0}`,
		},
		{old: `"surface headless runtime"`, new: `"surface wasm32-web runtime"`},
		{old: `"headless event dispatch"`, new: `"wasm32-web Surface Host ABI imports"`},
		{old: `"headless framebuffer checksum"`, new: `"wasm32-web framebuffer checksum evidence"`},
		{old: `"headless actual runner trace"`, new: `"wasm32-web runner trace"`},
		{
			old: `"artifact_scan": {"root":` + jsonString(
				filepath.Dir(wasmPath),
			) + `,"files_checked":2`,
			new: `"artifact_scan": {"root":` + jsonString(
				filepath.Dir(wasmPath),
			) + `,"files_checked":3`,
		},
	}
	for _, repl := range replacements {
		raw = strings.Replace(raw, repl.old, repl.new, 1)
	}
	raw = strings.Replace(
		raw,
		`{"kind":"component-app","path":`+jsonString(
			wasmPath,
		)+`,"sha256":`+jsonString(
			wasmSHA,
		)+`,"size":`+strconv.FormatInt(
			wasmSize,
			10,
		)+`},
    {"kind":"runner-trace","path":`+jsonString(
			tracePath,
		)+`,"sha256":`+jsonString(
			traceSHA,
		)+`,"size":`+strconv.FormatInt(
			traceSize,
			10,
		)+`}`,
		`{"kind":"component-app","path":`+jsonString(
			wasmPath,
		)+`,"sha256":`+jsonString(
			wasmSHA,
		)+`,"size":`+strconv.FormatInt(
			wasmSize,
			10,
		)+`},
    {"kind":"compiler-owned-loader","path":`+jsonString(
			loaderPath,
		)+`,"sha256":`+jsonString(
			loaderSHA,
		)+`,"size":`+strconv.FormatInt(
			loaderSize,
			10,
		)+`},
    {"kind":"runner-trace","path":`+jsonString(
			tracePath,
		)+`,"sha256":`+jsonString(
			traceSHA,
		)+`,"size":`+strconv.FormatInt(
			traceSize,
			10,
		)+`}`,
		1,
	)
	raw = strings.Replace(
		raw,
		("{\"order\":2,\"width\":320,\"height\":200,\"stride\":1280," +
			"\"checksum\":" +
			"\"9ff0b13bb6026d76c7f61b202436f30e703d45d96730033ee6d171bcf58" +
			"02f82\",\"presented\":true}"),
		("{\"order\":2,\"width\":320,\"height\":200,\"stride\":1280," +
			"\"checksum\":" +
			"\"9ff0b13bb6026d76c7f61b202436f30e703d45d96730033ee6d171bcf58" +
			"02f82\",\"presented\":true},\n    {\"order\":3,\"width\":320," +
			"\"height\":200,\"stride\":1280,\"checksum\":" +
			"\"33333333333333333333333333333333333333333333333333333333333" +
			"33333\",\"presented\":true},\n    {\"order\":4,\"width\":320," +
			"\"height\":200,\"stride\":1280,\"checksum\":" +
			"\"44444444444444444444444444444444444444444444444444444444444" +
			"44444\",\"presented\":true}"),
		1,
	)
	raw = strings.Replace(
		raw,
		`{"name":"wasm32-web framebuffer checksum evidence","kind":"positive","ran":true,"pass":true}`,
		`{"name":"wasm32-web framebuffer checksum evidence","kind":"positive","ran":true,"pass":true},
    {"name":"compiler-owned wasm Surface loader","kind":"positive","ran":true,"pass":true},
    {"name":"wasm32-web actual presented frame trace","kind":"positive","ran":true,"pass":true}`,
		1,
	)
	return []byte(raw)
}

func validWASM32WebBrowserCanvasSurfaceRuntimeReportJSON(
	wasmPath string,
	wasmSHA string,
	wasmSize int64,
	loaderPath string,
	loaderSHA string,
	loaderSize int64,
	tracePath string,
	traceSHA string,
	traceSize int64,
) []byte {
	raw := strings.Join([]string{
		"{",
		`  "schema": "tetra.surface.runtime.v1",`,
		`  "status": "pass",`,
		`  "target": "wasm32-web",`,
		`  "host": "linux-x64",`,
		`  "runtime": "surface-wasm32-web",`,
		`  "surface_schema": "tetra.surface.v1",`,
		`  "host_abi": "tetra.surface.host-abi.v1",`,
		`  "host_evidence": ` + compactFixtureJSON(`{
			"level": "wasm32-web-browser-canvas-input",
			"backend": "browser-canvas-rgba",
			"framebuffer": true,
			"real_window": false,
			"native_input": true,
			"user_facing_platform_widgets": false
		}`) + `,`,
		`  "source": "examples/surface/runtime/surface_browser_counter.tetra",`,
		`  "processes": [`,
		`    ` + compactFixtureJSON(`{
			"name": "tetra build",
			"kind": "build",
			"path": "__BUILD_PROCESS_PATH__",
			"ran": true,
			"pass": true,
			"exit_code": 0
		}`) + `,`,
		`    ` + compactFixtureJSON(`{
			"name": "surface wasm32-web browser canvas component app",
			"kind": "app",
			"path": "__BROWSER_PROCESS_PATH__",
			"ran": true,
			"pass": true,
			"exit_code": 0,
			"expected_exit_code": 0
		}`) + `,`,
		`    ` + compactFixtureJSON(`{
			"name": "surface wasm32-web import validator",
			"kind": "runtime",
			"path": "__IMPORT_VALIDATOR_PROCESS_PATH__",
			"ran": true,
			"pass": true,
			"exit_code": 0
		}`) + `,`,
		`    ` + compactFixtureJSON(`{
			"name": "surface wasm32-web browser canvas runtime",
			"kind": "runtime",
			"path": "Chromium fixture",
			"ran": true,
			"pass": true,
			"exit_code": 0
		}`),
		`  ],`,
		`  "artifacts": [`,
		`    ` + compactFixtureJSON(`{
			"kind": "component-app",
			"path": "__WASM_PATH__",
			"sha256": "__WASM_SHA__",
			"size": "__WASM_SIZE__"
		}`) + `,`,
		`    ` + compactFixtureJSON(`{
			"kind": "compiler-owned-loader",
			"path": "__LOADER_PATH__",
			"sha256": "__LOADER_SHA__",
			"size": "__LOADER_SIZE__"
		}`) + `,`,
		`    ` + compactFixtureJSON(`{
			"kind": "runner-trace",
			"path": "__TRACE_PATH__",
			"sha256": "__TRACE_SHA__",
			"size": "__TRACE_SIZE__"
		}`),
		`  ],`,
		`  "artifact_scan": ` + compactFixtureJSON(`{
			"root": "__ARTIFACT_SCAN_ROOT__",
			"files_checked": 3,
			"forbidden_paths": [],
			"pass": true
		}`) + `,`,
		`  "components": [`,
		`    ` + compactFixtureJSON(`{
			"id": "CounterApp",
			"type": "examples.surface.runtime.surface_browser_counter.CounterApp",
			"bounds": {"x":0,"y":0,"w":400,"h":240},
			"abilities": ["measure","layout","draw","event","focus","text","accessibility"],
			"state": {
				"count": "2",
				"key_count": "1",
				"width": "400",
				"accessibility_role": "button"
			}
		}`) + `,`,
		`    ` + compactFixtureJSON(`{
			"id": "CounterButton",
			"type": "examples.surface.runtime.surface_browser_counter.CounterButton",
			"parent": "CounterApp",
			"bounds": {"x":32,"y":88,"w":160,"h":48},
			"abilities": ["measure","layout","draw","event","focus","text","accessibility"],
			"state": {"focused":"true","text_len_seen":"2"}
		}`),
		`  ],`,
		`  "events": [`,
		`    ` + compactFixtureJSON(`{
			"order": 1,
			"kind": "mouse_up",
			"target_component": "CounterButton",
			"dispatch_path": ["CounterApp","CounterButton"],
			"handled": true,
			"pass": true,
			"x": 48,
			"y": 96,
			"key": 0,
			"width": 320,
			"height": 200,
			"timestamp_ms": 0,
			"buffer_slots": [5,48,96,1,0,320,200,0,0],
			"before_state": {"CounterApp.count":"0"},
			"after_state": {"CounterApp.count":"1"}
		}`) + `,`,
		`    ` + compactFixtureJSON(`{
			"order": 2,
			"kind": "key_down",
			"target_component": "CounterApp",
			"dispatch_path": ["CounterApp"],
			"handled": true,
			"pass": true,
			"x": 0,
			"y": 0,
			"key": 32,
			"width": 320,
			"height": 200,
			"timestamp_ms": 1,
			"buffer_slots": [6,0,0,0,32,320,200,1,0],
			"before_state": {"CounterApp.count":"1","CounterApp.key_count":"0"},
			"after_state": {"CounterApp.count":"2","CounterApp.key_count":"1"}
		}`) + `,`,
		`    ` + compactFixtureJSON(`{
			"order": 3,
			"kind": "resize",
			"target_component": "CounterApp",
			"dispatch_path": ["CounterApp"],
			"handled": true,
			"pass": true,
			"x": 0,
			"y": 0,
			"key": 0,
			"width": 400,
			"height": 240,
			"timestamp_ms": 2,
			"buffer_slots": [2,0,0,0,0,400,240,2,0],
			"before_state": {"CounterApp.width":"320"},
			"after_state": {"CounterApp.width":"400"}
		}`) + `,`,
		`    ` + compactFixtureJSON(`{
			"order": 4,
			"kind": "text_input",
			"target_component": "CounterButton",
			"dispatch_path": ["CounterApp","CounterButton"],
			"handled": true,
			"pass": true,
			"x": 0,
			"y": 0,
			"key": 0,
			"width": 400,
			"height": 240,
			"timestamp_ms": 3,
			"text_len": 2,
			"text_bytes_hex": "4f4b",
			"buffer_slots": [8,0,0,0,0,400,240,3,2],
			"before_state": {"CounterButton.text_len_seen":"0"},
			"after_state": {"CounterButton.text_len_seen":"2"}
		}`),
		`  ],`,
		`  "frames": [`,
		`    ` + compactFixtureJSON(`{
			"order": 1,
			"width": 320,
			"height": 200,
			"stride": 1280,
			"checksum": "1111111111111111111111111111111111111111111111111111111111111111",
			"presented": true
		}`) + `,`,
		`    ` + compactFixtureJSON(`{
			"order": 5,
			"width": 400,
			"height": 240,
			"stride": 1600,
			"checksum": "5555555555555555555555555555555555555555555555555555555555555555",
			"presented": true
		}`),
		`  ],`,
		`  "state_transitions": [`,
		`    ` + compactFixtureJSON(`{
			"order": 1,
			"component": "CounterApp",
			"field": "count",
			"before": "0",
			"after": "1",
			"cause": "mouse_up"
		}`) + `,`,
		`    ` + compactFixtureJSON(`{
			"order": 2,
			"component": "CounterApp",
			"field": "key_count",
			"before": "0",
			"after": "1",
			"cause": "key_down"
		}`) + `,`,
		`    ` + compactFixtureJSON(`{
			"order": 3,
			"component": "CounterApp",
			"field": "width",
			"before": "320",
			"after": "400",
			"cause": "resize"
		}`) + `,`,
		`    ` + compactFixtureJSON(`{
			"order": 4,
			"component": "CounterButton",
			"field": "text_len_seen",
			"before": "0",
			"after": "2",
			"cause": "text_input"
		}`),
		`  ],`,
		`  "cases": [`,
		`    {"name":"pure Tetra component app","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"wasm32-web Surface Host ABI imports","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"compiler-owned wasm Surface loader","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"wasm32-web browser canvas surface","kind":"positive","ran":true,"pass":true},`,
		fixturePositiveCaseLine("wasm32-web browser canvas RGBA readback") + `,`,
		fixturePositiveCaseLine("wasm32-web browser canvas pointer input") + `,`,
		fixturePositiveCaseLine("wasm32-web browser canvas keyboard input") + `,`,
		`    {"name":"wasm32-web browser canvas resize input","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"wasm32-web browser canvas text input","kind":"positive","ran":true,"pass":true},`,
		fixturePositiveCaseLine("compiler-owned browser canvas Surface host") + `,`,
		`    {"name":"host-provided pointer event dispatch","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"host event buffer poll_event","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"pre/post event frame sequence","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"component hierarchy dispatch","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"component text input scalar dispatch","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"host text payload buffer","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"component focus dispatch","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"component accessibility metadata","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"no legacy UI sidecar artifacts","kind":"positive","ran":true,"pass":true},`,
		`    {"name":"state transition","kind":"positive","ran":true,"pass":true},`,
		fixtureNegativeCaseLine("reject legacy UI evidence", "legacy UI evidence rejected"),
		`  ]`,
		`}`,
	}, "\n")
	raw = strings.NewReplacer(
		`"__BROWSER_PROCESS_PATH__"`,
		jsonString("/usr/bin/chromium --headless <surface-browser-canvas-runner> wasm="+wasmPath),
		`"__IMPORT_VALIDATOR_PROCESS_PATH__"`,
		jsonString("go run ./tools/cmd/validate-wasm-imports --target wasm32-web "+wasmPath),
		`"__BUILD_PROCESS_PATH__"`,
		jsonString(
			("tetra build --target wasm32-web "+
				"examples/surface/runtime/surface_browser_counter.tetra -o ")+wasmPath,
		),
		`"__WASM_PATH__"`,
		jsonString(wasmPath),
		`"__WASM_SHA__"`,
		jsonString(wasmSHA),
		`"__WASM_SIZE__"`,
		strconv.FormatInt(wasmSize, 10),
		`"__LOADER_PATH__"`,
		jsonString(loaderPath),
		`"__LOADER_SHA__"`,
		jsonString(loaderSHA),
		`"__LOADER_SIZE__"`,
		strconv.FormatInt(loaderSize, 10),
		`"__TRACE_PATH__"`,
		jsonString(tracePath),
		`"__TRACE_SHA__"`,
		jsonString(traceSHA),
		`"__TRACE_SIZE__"`,
		strconv.FormatInt(traceSize, 10),
		`"__ARTIFACT_SCAN_ROOT__"`,
		jsonString(filepath.Dir(wasmPath)),
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
  "command_line": "` + ("bash scripts/release/surface/release-gate.sh " +
		"--report-dir reports/surface-release-v1") + `",
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
	  "app_model": "explicit-command-reducer-v1",
	  "linux_app_shell": "linux-app-shell-subset-v1",
	  "app_shell_features": "electron-feature-ledger-v1",
	  "security_permissions": "surface-security-permission-v1",
	  "performance_budget": "surface-performance-budget-v1",
	  "developer_fast_loop": "surface-dev-workflow-v1",
	  "inspector": "surface-inspector-v1",
	  "project_templates": "surface-template-smoke-v1",
	  "reference_apps": "surface-reference-app-suite-v1",
	  "surface_package": "surface-package-v1",
	  "crash_reporting": "surface-crash-report-v1",
	  "i18n_localization": "surface-i18n-v1",
	  "widget_migration": "surface-widget-migration-v1",
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

func releaseToolkitSliceReportForTest(
	target string,
	host surface.HostEvidenceReport,
) surface.Report {
	return surface.Report{
		Schema:           surface.SchemaV1,
		Status:           "pass",
		Target:           target,
		Source:           "examples/surface/release/surface_release_form.tetra",
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

func releaseAccessibilitySliceReportForTest(
	target string,
	host surface.HostEvidenceReport,
) surface.Report {
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
		Source:            "examples/surface/release/surface_release_accessibility.tetra",
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
