package surface

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateReportAcceptsHeadlessSurfaceRuntimeEvidence(t *testing.T) {
	raw := validHeadlessSurfaceReportJSON()
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportAcceptsLinuxX64SurfaceRuntimeEvidence(t *testing.T) {
	raw := validLinuxX64SurfaceReportJSON()
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportAcceptsLinuxX64RealWindowSurfaceRuntimeEvidence(t *testing.T) {
	raw := validLinuxX64RealWindowSurfaceReportJSON()
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportAcceptsWASM32WebSurfaceRuntimeEvidence(t *testing.T) {
	raw := validWASM32WebSurfaceReportJSON()
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestValidateReportAcceptsWASM32WebBrowserCanvasSurfaceRuntimeEvidence(t *testing.T) {
	raw := validWASM32WebBrowserCanvasSurfaceReportJSON()
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}
func TestHeadlessReleaseRequiresBuiltBinary(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `    {"name":"tetra build","kind":"build","path":"tetra build --target linux-x64 examples/surface_counter.tetra -o /tmp/surface-artifacts/surface-counter","ran":true,"pass":true,"exit_code":0},
`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected headless report without build process to fail")
	}
	if !strings.Contains(err.Error(), "build process") {
		t.Fatalf("error = %v, want build process diagnostic", err)
	}
}
func TestHeadlessRunnerTraceMatchesReport(t *testing.T) {
	raw := mutateHeadlessSurfaceReport(t, func(report map[string]any) {
		frames := report["frames"].([]any)
		first := frames[0].(map[string]any)
		second := frames[1].(map[string]any)
		second["checksum"] = first["checksum"]
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected unchanged pre/post headless frame checksum evidence to fail")
	}
	if !strings.Contains(err.Error(), "pre/post") {
		t.Fatalf("error = %v, want pre/post frame diagnostic", err)
	}
}
func TestHeadlessRejectsMetadataOnlyFrame(t *testing.T) {
	raw := mutateHeadlessSurfaceReport(t, func(report map[string]any) {
		frames := report["frames"].([]any)
		first := frames[0].(map[string]any)
		first["checksum"] = ""
	})
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected metadata-only headless frame to fail")
	}
	if !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("error = %v, want checksum diagnostic", err)
	}
}
func TestHeadlessNoLegacySidecars(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()),
		`    {"name":"no legacy UI sidecar artifacts","kind":"positive","ran":true,"pass":true},
`,
		``,
		1,
	)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing no-legacy sidecar case to fail")
	}
	if !strings.Contains(err.Error(), "no legacy UI sidecar") {
		t.Fatalf("error = %v, want no legacy sidecar diagnostic", err)
	}
}
func mutateHeadlessSurfaceReport(t *testing.T, mutate func(map[string]any)) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validHeadlessSurfaceReportJSON(), &report); err != nil {
		t.Fatalf("decode headless report: %v", err)
	}
	mutate(report)
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal headless report: %v", err)
	}
	return raw
}
func TestValidateReportRejectsMissingHostEvidence(t *testing.T) {
	raw := []byte(strings.Replace(string(validHeadlessSurfaceReportJSON()), `  "host_evidence": {"level":"deterministic-headless","backend":"software-rgba","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false},
`, ``, 1))
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected report without explicit host_evidence to fail")
	}
	if !strings.Contains(err.Error(), "host_evidence") {
		t.Fatalf("error = %v, want host_evidence diagnostic", err)
	}
}
func TestValidateReportRejectsLinuxX64ReportWithHeadlessRuntimeEvidence(t *testing.T) {
	raw := []byte(strings.Replace(string(validHeadlessSurfaceReportJSON()), `"target": "headless"`, `"target": "linux-x64"`, 1))
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected linux-x64 report with headless runtime evidence to fail")
	}
	if !strings.Contains(err.Error(), "linux-x64") || !strings.Contains(err.Error(), "surface-linux-x64") {
		t.Fatalf("error = %v, want linux-x64 runtime evidence diagnostic", err)
	}
}
func TestValidateReportRejectsLinuxX64MemfdStarterClaimingRealWindow(t *testing.T) {
	raw := strings.Replace(string(validLinuxX64SurfaceReportJSON()), `"real_window":false,"native_input":false`, `"real_window":true,"native_input":true`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected linux-x64 memfd starter real-window claim to fail")
	}
	if !strings.Contains(err.Error(), "memfd starter") || !strings.Contains(err.Error(), "real_window") {
		t.Fatalf("error = %v, want memfd starter real_window diagnostic", err)
	}
}
func TestValidateReportRejectsLinuxX64RealWindowWithoutRealWindowProbeEvidence(t *testing.T) {
	raw := strings.Replace(string(validLinuxX64SurfaceReportJSON()),
		`"host_evidence": {"level":"linux-x64-memfd-starter","backend":"memfd-rgba","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false}`,
		`"host_evidence": {"level":"linux-x64-real-window","backend":"x11-rgba","framebuffer":true,"real_window":true,"native_input":true,"user_facing_platform_widgets":false}`,
		1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected linux-x64 real-window claim without real-window process/case evidence to fail")
	}
	for _, want := range []string{"real-window", "native input"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsWASM32WebReportWithHeadlessRuntimeEvidence(t *testing.T) {
	raw := []byte(strings.Replace(string(validHeadlessSurfaceReportJSON()), `"target": "headless"`, `"target": "wasm32-web"`, 1))
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected wasm32-web report with headless runtime evidence to fail")
	}
	if !strings.Contains(err.Error(), "wasm32-web") || !strings.Contains(err.Error(), "surface-wasm32-web") {
		t.Fatalf("error = %v, want wasm32-web runtime evidence diagnostic", err)
	}
}
func TestValidateReportRejectsWASM32WebReportMissingCompilerOwnedLoaderEvidence(t *testing.T) {
	raw := strings.Replace(string(validWASM32WebSurfaceReportJSON()), `,
    {"name":"compiler-owned wasm Surface loader","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected wasm32-web report without compiler-owned loader evidence to fail")
	}
	if !strings.Contains(err.Error(), "compiler-owned wasm Surface loader") {
		t.Fatalf("error = %v, want compiler-owned loader evidence diagnostic", err)
	}
}
func TestValidateReportRejectsWASM32WebReportMissingActualPresentedFrameTraceEvidence(t *testing.T) {
	raw := strings.Replace(string(validWASM32WebSurfaceReportJSON()), `,
    {"order":4,"width":320,"height":200,"stride":1280,"checksum":"4444444444444444444444444444444444444444444444444444444444444444","presented":true}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"wasm32-web actual presented frame trace","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected wasm32-web report without actual presented frame trace evidence to fail")
	}
	if !strings.Contains(err.Error(), "actual presented frame trace") {
		t.Fatalf("error = %v, want actual presented frame trace evidence diagnostic", err)
	}
}
func TestValidateReportRejectsWASM32WebReportMissingImportValidatorProcess(t *testing.T) {
	raw := strings.Replace(string(validWASM32WebSurfaceReportJSON()), `,
    {"name":"surface wasm32-web import validator","kind":"runtime","path":"go run ./tools/cmd/validate-wasm-imports --target wasm32-web /tmp/surface-artifacts/surface-counter.wasm","ran":true,"pass":true,"exit_code":0}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected wasm32-web report without validate-wasm-imports process evidence to fail")
	}
	for _, want := range []string{"wasm32-web", "validate-wasm-imports"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsWASM32WebBrowserCanvasWithoutBrowserProcess(t *testing.T) {
	raw := strings.Replace(string(validWASM32WebBrowserCanvasSurfaceReportJSON()), `/usr/bin/chromium --headless <surface-browser-canvas-runner> wasm=/tmp/surface-artifacts/surface-browser-counter.wasm`, `node scripts/tools/web_run_module.mjs /tmp/surface-artifacts/surface-browser-counter.wasm`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected browser canvas report without Chromium process evidence to fail")
	}
	if !strings.Contains(err.Error(), "Chromium-compatible browser") {
		t.Fatalf("error = %v, want Chromium-compatible browser process diagnostic", err)
	}
}
func TestValidateReportRejectsWASM32WebBrowserCanvasMissingInputEvidence(t *testing.T) {
	raw := strings.Replace(string(validWASM32WebBrowserCanvasSurfaceReportJSON()), `,
    {"order":2,"kind":"key_down","target_component":"CounterApp","dispatch_path":["CounterApp"],"handled":true,"pass":true,"x":0,"y":0,"key":32,"width":320,"height":200,"timestamp_ms":1,"buffer_slots":[6,0,0,0,32,320,200,1,0],"before_state":{"CounterApp.count":"1","CounterApp.key_count":"0"},"after_state":{"CounterApp.count":"2","CounterApp.key_count":"1"}}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"wasm32-web browser canvas keyboard input","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected browser canvas report without keyboard input evidence to fail")
	}
	for _, want := range []string{"keyboard input", "key_down"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsWASM32WebReportMissingRunnerTraceArtifact(t *testing.T) {
	raw := strings.Replace(string(validWASM32WebSurfaceReportJSON()), `,
    {"kind":"runner-trace","path":"/tmp/surface-artifacts/surface-runner-trace.json","sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","size":413}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected wasm32-web report without runner trace artifact to fail")
	}
	if !strings.Contains(err.Error(), "runner trace artifact") {
		t.Fatalf("error = %v, want runner trace artifact diagnostic", err)
	}
}
func TestValidateReportRejectsHeadlessReportMissingRunnerTraceEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `,
    {"kind":"runner-trace","path":"/tmp/surface-artifacts/surface-runner-trace.json","sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","size":409}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"headless actual runner trace","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected headless report without runner trace evidence to fail")
	}
	if !strings.Contains(err.Error(), "headless actual runner trace") {
		t.Fatalf("error = %v, want headless runner trace evidence diagnostic", err)
	}
}
func TestValidateReportRejectsLinuxX64ReportMissingAppPresentedFrameEvidence(t *testing.T) {
	raw := strings.Replace(string(validLinuxX64SurfaceReportJSON()), `,
    {"order":3,"width":2,"height":2,"stride":8,"checksum":"3333333333333333333333333333333333333333333333333333333333333333","presented":true}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"linux-x64 app-presented RGBA checksum","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected linux-x64 report without app-presented frame evidence to fail")
	}
	if !strings.Contains(err.Error(), "app-presented RGBA checksum") {
		t.Fatalf("error = %v, want app-presented frame evidence diagnostic", err)
	}
}
func TestValidateReportRejectsLinuxX64ReportMissingCounterComponentAppPresentedFrameEvidence(t *testing.T) {
	raw := strings.Replace(string(validLinuxX64SurfaceReportJSON()), `,
    {"order":4,"width":320,"height":200,"stride":1280,"checksum":"4444444444444444444444444444444444444444444444444444444444444444","presented":true}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"surface linux-x64 counter app presented frame probe","kind":"app","path":"/tmp/surface-artifacts/surface-counter-present-probe","ran":true,"pass":true,"exit_code":-1,"expected_exit_code":-1}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"linux-x64 counter component app-presented frame","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected linux-x64 report without counter component app-presented frame evidence to fail")
	}
	if !strings.Contains(err.Error(), "counter component app-presented frame") {
		t.Fatalf("error = %v, want counter component app-presented frame evidence diagnostic", err)
	}
}
func TestValidateReportRejectsLinuxX64ReportMissingEventSequenceProbeEvidence(t *testing.T) {
	raw := strings.Replace(string(validLinuxX64SurfaceReportJSON()), `,
    {"name":"surface linux-x64 event sequence probe build","kind":"build","path":"/tmp/tetra build event sequence probe","ran":true,"pass":true,"exit_code":0},
    {"name":"surface linux-x64 event sequence probe","kind":"app","path":"/tmp/surface-event-sequence-probe","ran":true,"pass":true,"exit_code":42,"expected_exit_code":42}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"linux-x64 host event sequence","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected linux-x64 report without event sequence probe evidence to fail")
	}
	if !strings.Contains(err.Error(), "event sequence") {
		t.Fatalf("error = %v, want event sequence probe evidence diagnostic", err)
	}
}
func TestValidateReportRejectsMissingPrePostEventFrameSequence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `,
    {"order":2,"width":320,"height":200,"stride":1280,"checksum":"2222222222222222222222222222222222222222222222222222222222222222","presented":true}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"pre/post event frame sequence","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without pre/post frame evidence to fail")
	}
	if !strings.Contains(err.Error(), "pre/post event frame sequence") {
		t.Fatalf("error = %v, want pre/post frame sequence diagnostic", err)
	}
}
func TestValidateReportRejectsLegacyMetadataEvidence(t *testing.T) {
	raw := []byte(`{"schema":"tetra.ui.v1","status":"pass","source":"examples/ui_web_smoke.tetra"}`)
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected legacy metadata report to fail")
	}
	if !strings.Contains(err.Error(), SchemaV1) {
		t.Fatalf("error = %v, want Surface runtime schema rejection", err)
	}
}
func TestValidateReportRejectsDocsOnlyMarkers(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"source": "examples/surface_counter.tetra"`, `"source": "docs-only surface note"`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected docs-only marker to fail")
	}
	if !strings.Contains(err.Error(), "docs-only") {
		t.Fatalf("error = %v, want docs-only rejection", err)
	}
}
func TestValidateReportRejectsForbiddenEvidenceMarkers(t *testing.T) {
	for _, tc := range []struct {
		source string
		want   string
	}{
		{source: "web-only", want: "web-only"},
		{source: "metadata-only", want: "metadata-only"},
		{source: "node-only", want: "node-only"},
		{source: "dom-only", want: "dom-only"},
		{source: "build-only", want: "build-only"},
		{source: "surface fake evidence", want: "fake"},
		{source: "surface stale evidence", want: "stale"},
		{source: "surface mock evidence", want: "mock"},
		{source: "placeholder", want: "placeholder"},
	} {
		t.Run(tc.source, func(t *testing.T) {
			raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"source": "examples/surface_counter.tetra"`, `"source": "`+tc.source+`"`, 1)
			err := ValidateReport([]byte(raw))
			if err == nil {
				t.Fatalf("expected marker %q to fail", tc.source)
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("error = %v, want marker rejection for %q", err, tc.want)
			}
		})
	}
}
func TestValidateReportRejectsLegacyUISidecarMarkers(t *testing.T) {
	for _, tc := range []struct {
		name string
		want string
	}{
		{name: "generated .ui.html sidecar", want: ".ui.html"},
		{name: "generated .ui.web.mjs sidecar", want: ".ui.web.mjs"},
		{name: "generated .ui.json sidecar", want: ".ui.json"},
		{name: "DOM UI surface", want: "dom ui"},
		{name: "user JavaScript bridge", want: "user javascript"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"pure Tetra component app"`, `"`+tc.name+`"`, 1)
			err := ValidateReport([]byte(raw))
			if err == nil {
				t.Fatalf("expected legacy UI marker %q to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("error = %v, want marker rejection for %q", err, tc.want)
			}
		})
	}
}
func TestValidateReportRejectsUserFacingPlatformWidgetMarkers(t *testing.T) {
	for _, tc := range []struct {
		name string
		want string
	}{
		{name: "React component surface", want: "react"},
		{name: "GTK widget surface", want: "gtk widget"},
		{name: "Qt widget surface", want: "qt widget"},
		{name: "WinUI widget surface", want: "winui"},
		{name: "Cocoa widget surface", want: "cocoa"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"pure Tetra component app"`, `"`+tc.name+`"`, 1)
			err := ValidateReport([]byte(raw))
			if err == nil {
				t.Fatalf("expected platform widget marker %q to fail", tc.name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("error = %v, want marker rejection for %q", err, tc.want)
			}
		})
	}
}
func TestValidateReportRejectsMissingNoLegacyUISidecarEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `,
    {"name":"no legacy UI sidecar artifacts","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing no-sidecar evidence to fail")
	}
	if !strings.Contains(err.Error(), "no legacy UI sidecar artifacts") {
		t.Fatalf("error = %v, want no legacy UI sidecar evidence diagnostic", err)
	}
}
func TestValidateReportRejectsMissingArtifactScanEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `,
  "artifact_scan": {"root":"/tmp/surface-artifacts","files_checked":2,"forbidden_paths":[],"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without artifact_scan evidence to fail")
	}
	for _, want := range []string{"artifact_scan"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsArtifactOutsideArtifactScanRoot(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"artifact_scan": {"root":"/tmp/surface-artifacts"`, `"artifact_scan": {"root":"/tmp/other-surface-artifacts"`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report whose artifacts are outside artifact_scan.root to fail")
	}
	for _, want := range []string{"artifact_scan.root", "outside"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsArtifactScanCheckingFewerFilesThanReportedArtifacts(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"files_checked":2`, `"files_checked":1`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report whose artifact_scan checked fewer files than reported artifacts to fail")
	}
	for _, want := range []string{"artifact_scan.files_checked", "reported artifacts"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMissingHostProvidedPointerEventEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `,
    {"name":"host-provided pointer event dispatch","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing host-provided pointer event evidence to fail")
	}
	if !strings.Contains(err.Error(), "host-provided pointer event dispatch") {
		t.Fatalf("error = %v, want host-provided pointer event evidence diagnostic", err)
	}
}
func TestValidateReportRejectsComponentMissingMeasureLayoutAbilities(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"abilities":["measure","layout","draw","event","focus","text","accessibility"]`, `"abilities":["draw","event","focus","text","accessibility"]`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected component without measure/layout abilities to fail")
	}
	for _, want := range []string{"measure ability", "layout ability"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMissingFocusAbilityAndEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"abilities":["measure","layout","draw","event","focus","text","accessibility"]`, `"abilities":["measure","layout","draw","event","text","accessibility"]`, 1)
	raw = strings.Replace(raw, `,
    {"name":"component focus dispatch","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without focus ability and case evidence to fail")
	}
	for _, want := range []string{"focus ability", "component focus dispatch"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMissingAccessibilityAbilityAndEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"abilities":["measure","layout","draw","event","focus","text","accessibility"]`, `"abilities":["measure","layout","draw","event","focus","text"]`, 1)
	raw = strings.Replace(raw, `,
    {"name":"component accessibility metadata","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without accessibility ability and case evidence to fail")
	}
	for _, want := range []string{"accessibility ability", "component accessibility metadata"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMissingTextAbilityAndEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"abilities":["measure","layout","draw","event","focus","text","accessibility"]`, `"abilities":["measure","layout","draw","event","focus","accessibility"]`, 1)
	raw = strings.Replace(raw, `,
    {"order":3,"kind":"text_input","target_component":"CounterButton","handled":true,"pass":true,"x":0,"y":0,"text_len":2,"text_bytes_hex":"4f4b","before_state":{"CounterApp.text_count":"0","CounterButton.text_len_seen":"0"},"after_state":{"CounterApp.text_count":"1","CounterButton.text_len_seen":"2"}}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"order":2,"component":"CounterApp","field":"text_count","before":"0","after":"1","cause":"text_input"}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"component text input scalar dispatch","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without text ability and scalar text-input evidence to fail")
	}
	for _, want := range []string{"text ability", "component text input scalar dispatch"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMissingHostTextPayloadBufferEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"text_len":2,"text_bytes_hex":"4f4b",`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"host text payload buffer","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without host text payload buffer evidence to fail")
	}
	for _, want := range []string{"text payload", "host text payload buffer"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMissingHostEventBufferEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"key":0,"width":320,"height":200,"timestamp_ms":0,"buffer_slots":[5,48,96,1,0,320,200,0,0],`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"host event buffer poll_event","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without host event buffer evidence to fail")
	}
	for _, want := range []string{"event buffer", "host event buffer poll_event"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMissingHostEventBufferSequenceEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()),
		`"timestamp_ms":1,"text_len":2,"text_bytes_hex":"4f4b","buffer_slots":[8,0,0,0,0,320,200,1,2]`,
		`"timestamp_ms":0,"text_len":2,"text_bytes_hex":"4f4b","buffer_slots":[8,0,0,0,0,320,200,0,2]`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without host event buffer pointer/text sequence to fail")
	}
	if !strings.Contains(err.Error(), "event buffer pointer/text sequence") {
		t.Fatalf("error = %v, want host event buffer pointer/text sequence diagnostic", err)
	}
}
func TestValidateReportRejectsMissingComponentHierarchyEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `,
    {"id":"CounterButton","type":"examples.surface_counter.CounterButton","parent":"CounterApp","bounds":{"x":32,"y":80,"w":160,"h":48},"abilities":["measure","layout","draw","event","focus","text","accessibility"],"state":{"pressed":"false","focused":"true","text_len_seen":"2","accessibility_role":"button"}}`, ``, 1)
	raw = strings.Replace(raw, `"target_component":"CounterButton"`, `"target_component":"CounterApp"`, 1)
	raw = strings.Replace(raw, `,
    {"name":"component hierarchy dispatch","kind":"positive","ran":true,"pass":true}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without child component hierarchy evidence to fail")
	}
	for _, want := range []string{"component hierarchy", "component hierarchy dispatch"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMissingComponentLayoutBoundsEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"bounds":{"x":32,"y":80,"w":160,"h":48},`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without child component bounds evidence to fail")
	}
	if !strings.Contains(err.Error(), "layout bounds") {
		t.Fatalf("error = %v, want layout bounds diagnostic", err)
	}
}
func TestValidateReportRejectsMissingEventDispatchPathEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"dispatch_path":["CounterApp","CounterButton"],`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without child dispatch_path evidence to fail")
	}
	if !strings.Contains(err.Error(), "dispatch_path") {
		t.Fatalf("error = %v, want dispatch_path diagnostic", err)
	}
}
func TestValidateReportRejectsDispatchPathSkippingParent(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"dispatch_path":["CounterApp","CounterButton"]`, `"dispatch_path":["CounterButton"]`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report with dispatch_path skipping parent to fail")
	}
	if !strings.Contains(err.Error(), "dispatch_path") || !strings.Contains(err.Error(), "parent") {
		t.Fatalf("error = %v, want dispatch_path parent diagnostic", err)
	}
}
func TestValidateReportRejectsPointerDispatchOutsideTargetBounds(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"x":48,"y":96,"key":0,"width":320,"height":200,"timestamp_ms":0,"buffer_slots":[5,48,96,1,0,320,200,0,0]`, `"x":4,"y":4,"key":0,"width":320,"height":200,"timestamp_ms":0,"buffer_slots":[5,4,4,1,0,320,200,0,0]`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected pointer dispatch outside target bounds to fail")
	}
	if !strings.Contains(err.Error(), "target bounds") {
		t.Fatalf("error = %v, want target bounds diagnostic", err)
	}
}
func TestValidateReportRejectsSourcePathAsExecutableAppProcess(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"kind":"app","path":"/tmp/surface-artifacts/surface-counter"`, `"kind":"app","path":"examples/surface_counter.tetra"`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected app process source path to fail")
	}
	if !strings.Contains(err.Error(), "executable Surface app process") {
		t.Fatalf("error = %v, want executable app path diagnostic", err)
	}
}
func TestValidateReportRejectsBuildProcessMissingReportedSource(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"path":"tetra build --target linux-x64 examples/surface_counter.tetra -o /tmp/surface-artifacts/surface-counter"`, `"path":"tetra build --target linux-x64 examples/other_surface.tetra -o /tmp/surface-artifacts/surface-counter"`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected build process without reported source to fail")
	}
	for _, want := range []string{"build process", "source"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsMissingSurfaceComponentAppProcess(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"name":"surface component app"`, `"name":"surface auxiliary app"`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected missing Surface component app process to fail")
	}
	if !strings.Contains(err.Error(), "Surface component app process") {
		t.Fatalf("error = %v, want Surface component app process diagnostic", err)
	}
}
func TestValidateReportRejectsMissingComponentAppArtifactEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `    {"kind":"component-app","path":"/tmp/surface-artifacts/surface-counter","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":49172},
`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected report without Surface component app artifact hash evidence to fail")
	}
	for _, want := range []string{"artifact", "Surface component app"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestSurfaceProjectTemplateSourceAcceptsFinalProductReportPath(t *testing.T) {
	source := "reports/surface-product-v1/templates/command-palette/src/main.tetra"
	if !isSurfaceProjectTemplateSource(source) {
		t.Fatalf("final product report template source was rejected: %s", source)
	}
	if !isSurfaceBlockAccessibilitySource(source) {
		t.Fatalf("final product report template source was rejected for Block accessibility evidence: %s", source)
	}
}
func TestValidateReportRejectsWASM32WebMissingCompilerOwnedLoaderArtifact(t *testing.T) {
	raw := strings.Replace(string(validWASM32WebSurfaceReportJSON()), `,
    {"kind":"compiler-owned-loader","path":"/tmp/surface-artifacts/surface-counter.mjs","sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","size":4931}`, ``, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected wasm32-web report without compiler-owned loader artifact to fail")
	}
	for _, want := range []string{"compiler-owned loader artifact", "wasm32-web"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestValidateReportRejectsGeneratedHTMLArtifactEvidence(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `{"kind":"component-app","path":"/tmp/surface-artifacts/surface-counter"`, `{"kind":"component-app","path":"/tmp/surface-artifacts/surface-counter.html"`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected generated HTML artifact evidence to fail")
	}
	if !strings.Contains(err.Error(), "generated HTML UI") {
		t.Fatalf("error = %v, want generated HTML UI diagnostic", err)
	}
}
func TestValidateReportRejectsPlatformWidgetArtifactEvidence(t *testing.T) {
	for _, tc := range []struct {
		suffix string
		want   string
	}{
		{suffix: ".jsx", want: "react"},
		{suffix: ".tsx", want: "react"},
		{suffix: ".qml", want: "qt"},
		{suffix: ".xaml", want: "winui"},
		{suffix: ".xib", want: "cocoa"},
		{suffix: ".storyboard", want: "cocoa"},
		{suffix: ".glade", want: "gtk"},
	} {
		t.Run(tc.suffix, func(t *testing.T) {
			raw := strings.ReplaceAll(string(validHeadlessSurfaceReportJSON()), `/tmp/surface-artifacts/surface-counter`, `/tmp/surface-artifacts/surface-counter`+tc.suffix)
			err := ValidateReport([]byte(raw))
			if err == nil {
				t.Fatalf("expected platform widget artifact suffix %q to fail", tc.suffix)
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.want) {
				t.Fatalf("error = %v, want platform artifact rejection for %q", err, tc.want)
			}
		})
	}
}
func TestValidateReportRejectsSourceComponentModuleMismatch(t *testing.T) {
	raw := strings.Replace(string(validHeadlessSurfaceReportJSON()), `"source": "examples/surface_counter.tetra"`, `"source": "examples/other_surface.tetra"`, 1)
	err := ValidateReport([]byte(raw))
	if err == nil {
		t.Fatalf("expected source/component module mismatch to fail")
	}
	for _, want := range []string{"source module", "component type"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q diagnostic", err, want)
		}
	}
}
func TestSurfaceProjectTemplateSourceAcceptsExitZeroComponentAppProcess(t *testing.T) {
	for _, source := range []string{
		"reports/surface-electron-react-beauty-production/P21/template-smoke/templates/command-palette/src/main.tetra",
		"reports/surface-electron-react-beauty-production/P21/release-gate/templates/command-palette/src/main.tetra",
		"reports/surface/mrb10-template-smoke/templates/studio-shell/src/main.tetra",
		"reports/surface-product-v1-final-clean-20260613-0926/templates/command-palette/src/main.tetra",
		"reports/contract-refactor-pr-hardening/surface-release-final/templates/studio-shell/src/main.tetra",
	} {
		t.Run(source, func(t *testing.T) {
			exit := 0
			expected := 0
			process := ProcessReport{
				Name:             "surface component app",
				Kind:             "app",
				Path:             "reports/surface-electron-react-beauty-production/P21/template-smoke/template-runtime/command-palette-linux-x64",
				Ran:              true,
				Pass:             true,
				ExitCode:         &exit,
				ExpectedExitCode: &expected,
			}
			if !isSurfaceComponentAppProcess(source, process) {
				t.Fatalf("generated Surface template source should accept exit-zero component app process")
			}
			if !isSurfaceBlockAccessibilitySource(source) {
				t.Fatalf("generated Surface template source should accept Block accessibility evidence")
			}
			if !isSurfaceMorphReportSource(source) {
				t.Fatalf("generated Surface template source should accept Morph evidence")
			}
		})
	}
}

func TestSurfaceProjectTemplateSourceRejectsNonReportMainTetra(t *testing.T) {
	for _, source := range []string{
		"templates/studio-shell/src/main.tetra",
		"examples/studio-shell/src/main.tetra",
		"src/main.tetra",
	} {
		t.Run(source, func(t *testing.T) {
			if isSurfaceProjectTemplateSource(source) {
				t.Fatalf("non-report main.tetra source was accepted: %s", source)
			}
		})
	}
}

func TestSurfaceReferenceAppSourceAcceptsExitZeroMorphEvidence(t *testing.T) {
	exit := 0
	expected := 0
	process := ProcessReport{
		Name:             "surface component app",
		Kind:             "app",
		Path:             "reports/surface/reference-apps/reference-runtime/surface-headless-morph-artifacts/surface-morph-command-palette",
		Ran:              true,
		Pass:             true,
		ExitCode:         &exit,
		ExpectedExitCode: &expected,
	}
	for _, source := range []string{
		"examples/surface_reference_command_palette.tetra",
		"/repo/examples/surface_reference_settings.tetra",
	} {
		t.Run(source, func(t *testing.T) {
			if !isSurfaceReferenceAppSource(source) {
				t.Fatalf("reference Surface app source was rejected: %s", source)
			}
			if !isSurfaceComponentAppProcess(source, process) {
				t.Fatalf("reference Surface app source should accept exit-zero component app process")
			}
			if !isSurfaceBlockAccessibilitySource(source) {
				t.Fatalf("reference Surface app source should accept Block accessibility evidence")
			}
			if !isSurfaceMorphReportSource(source) {
				t.Fatalf("reference Surface app source should accept Morph evidence")
			}
		})
	}
}
func TestSurfaceFlagshipControlCenterSourceAcceptsAppStateExitFive(t *testing.T) {
	exit := 5
	expected := 5
	process := ProcessReport{
		Name:             "surface component app",
		Kind:             "app",
		Path:             "reports/surface-product-slice/product-gate/flagship/surface-headless-block-system-artifacts/surface-block-system",
		Ran:              true,
		Pass:             true,
		ExitCode:         &exit,
		ExpectedExitCode: &expected,
	}
	for _, source := range []string{
		"examples/surface_migration_tetra_control_center.tetra",
		"/repo/examples/surface_migration_tetra_control_center.tetra",
	} {
		if !isSurfaceComponentAppProcess(source, process) {
			t.Fatalf("flagship Surface source should accept app-state exit 5 for %s", source)
		}
	}
}
func TestValidateReportRejectsMissingFrameChecksumAndStateTransition(t *testing.T) {
	raw := []byte(`{
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
    {"name":"tetra build","kind":"build","path":"tetra build --target linux-x64 examples/surface_counter.tetra -o /tmp/surface-artifacts/surface-counter","ran":true,"pass":true,"exit_code":0},
    {"name":"surface component app","kind":"app","path":"/tmp/surface-artifacts/surface-counter","ran":true,"pass":true,"exit_code":1,"expected_exit_code":1},
    {"name":"surface headless runtime","kind":"runtime","path":"tools/cmd/surface-runtime-smoke","ran":true,"pass":true,"exit_code":0}
  ],
  "artifacts": [
    {"kind":"component-app","path":"/tmp/surface-artifacts/surface-counter","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":49172},
    {"kind":"runner-trace","path":"/tmp/surface-artifacts/surface-runner-trace.json","sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","size":409}
  ],
  "artifact_scan": {"root":"/tmp/surface-artifacts","files_checked":2,"forbidden_paths":[],"pass":true},
  "components": [
    {"id":"CounterApp","type":"examples.surface_counter.CounterApp","bounds":{"x":0,"y":0,"w":320,"h":200},"abilities":["measure","layout","draw","event","focus","text","accessibility"],"state":{"count":"1","text_count":"1","accessibility_role":"button"}},
    {"id":"CounterButton","type":"examples.surface_counter.CounterButton","parent":"CounterApp","bounds":{"x":32,"y":80,"w":160,"h":48},"abilities":["measure","layout","draw","event","focus","text","accessibility"],"state":{"pressed":"false","focused":"true","text_len_seen":"2","accessibility_role":"button"}}
  ],
  "events": [
    {"order":1,"kind":"mouse_up","target_component":"CounterApp","dispatch_path":["CounterApp"],"handled":true,"pass":true,"x":48,"y":96,"before_state":{"CounterApp.count":"0"},"after_state":{"CounterApp.count":"1"}}
  ],
  "frames": [
    {"order":1,"width":320,"height":200,"stride":1280,"checksum":"","presented":true}
  ],
  "cases": [
    {"name":"pure Tetra component app","kind":"positive","ran":true,"pass":true},
    {"name":"headless event dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"headless framebuffer checksum","kind":"positive","ran":true,"pass":true},
    {"name":"headless actual runner trace","kind":"positive","ran":true,"pass":true},
    {"name":"host-provided pointer event dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component hierarchy dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component focus dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component accessibility metadata","kind":"positive","ran":true,"pass":true},
    {"name":"no legacy UI sidecar artifacts","kind":"positive","ran":true,"pass":true},
    {"name":"state transition","kind":"positive","ran":true,"pass":true},
    {"name":"reject legacy UI evidence","kind":"negative","ran":true,"pass":true,"expected_error":"legacy UI evidence rejected"}
  ]
}`)
	err := ValidateReport(raw)
	if err == nil {
		t.Fatalf("expected missing checksum and transition to fail")
	}
	for _, want := range []string{"checksum", "state transition"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error missing %q:\n%v", want, err)
		}
	}
}
func validHeadlessSurfaceReportJSON() []byte {
	return []byte(`{
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
    {"name":"tetra build","kind":"build","path":"tetra build --target linux-x64 examples/surface_counter.tetra -o /tmp/surface-artifacts/surface-counter","ran":true,"pass":true,"exit_code":0},
    {"name":"surface component app","kind":"app","path":"/tmp/surface-artifacts/surface-counter","ran":true,"pass":true,"exit_code":1,"expected_exit_code":1},
    {"name":"surface headless runtime","kind":"runtime","path":"tools/cmd/surface-runtime-smoke","ran":true,"pass":true,"exit_code":0}
  ],
  "artifacts": [
    {"kind":"component-app","path":"/tmp/surface-artifacts/surface-counter","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":49172},
    {"kind":"runner-trace","path":"/tmp/surface-artifacts/surface-runner-trace.json","sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","size":409}
  ],
  "artifact_scan": {"root":"/tmp/surface-artifacts","files_checked":2,"forbidden_paths":[],"pass":true},
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
    {"order":1,"width":320,"height":200,"stride":1280,"checksum":"1111111111111111111111111111111111111111111111111111111111111111","presented":true},
    {"order":2,"width":320,"height":200,"stride":1280,"checksum":"2222222222222222222222222222222222222222222222222222222222222222","presented":true}
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
	}`)
}
func validLinuxX64SurfaceReportJSON() []byte {
	raw := string(validHeadlessSurfaceReportJSON())
	replacements := []struct {
		old string
		new string
	}{
		{old: `"target": "headless"`, new: `"target": "linux-x64"`},
		{old: `"runtime": "surface-headless"`, new: `"runtime": "surface-linux-x64"`},
		{old: `"host_evidence": {"level":"deterministic-headless","backend":"software-rgba","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false}`,
			new: `"host_evidence": {"level":"linux-x64-memfd-starter","backend":"memfd-rgba","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false}`},
		{old: `{"name":"surface component app","kind":"app","path":"/tmp/surface-artifacts/surface-counter","ran":true,"pass":true,"exit_code":1,"expected_exit_code":1}`,
			new: `{"name":"surface component app","kind":"app","path":"/tmp/surface-artifacts/surface-counter","ran":true,"pass":true,"exit_code":1,"expected_exit_code":1},
    {"name":"surface linux-x64 host probe build","kind":"build","path":"/tmp/tetra build probe","ran":true,"pass":true,"exit_code":0},
    {"name":"surface linux-x64 host probe","kind":"app","path":"/tmp/surface-host-probe","ran":true,"pass":true,"exit_code":42,"expected_exit_code":42},
    {"name":"surface linux-x64 event sequence probe build","kind":"build","path":"/tmp/tetra build event sequence probe","ran":true,"pass":true,"exit_code":0},
    {"name":"surface linux-x64 event sequence probe","kind":"app","path":"/tmp/surface-event-sequence-probe","ran":true,"pass":true,"exit_code":42,"expected_exit_code":42},
    {"name":"surface linux-x64 counter app presented frame probe","kind":"app","path":"/tmp/surface-artifacts/surface-counter-present-probe","ran":true,"pass":true,"exit_code":-1,"expected_exit_code":-1}`},
		{old: `"surface headless runtime"`, new: `"surface linux-x64 runtime"`},
		{old: `"headless event dispatch"`, new: `"linux-x64 Surface Host ABI open/present/close"`},
		{old: `"headless framebuffer checksum"`, new: `"linux-x64 framebuffer present evidence"`},
	}
	for _, repl := range replacements {
		raw = strings.Replace(raw, repl.old, repl.new, 1)
	}
	raw = strings.Replace(raw, `,
    {"kind":"runner-trace","path":"/tmp/surface-artifacts/surface-runner-trace.json","sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","size":409}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"headless actual runner trace","kind":"positive","ran":true,"pass":true}`, ``, 1)
	raw = strings.Replace(raw, `{"order":2,"width":320,"height":200,"stride":1280,"checksum":"2222222222222222222222222222222222222222222222222222222222222222","presented":true}`,
		`{"order":2,"width":320,"height":200,"stride":1280,"checksum":"2222222222222222222222222222222222222222222222222222222222222222","presented":true},
    {"order":3,"width":2,"height":2,"stride":8,"checksum":"3333333333333333333333333333333333333333333333333333333333333333","presented":true},
    {"order":4,"width":320,"height":200,"stride":1280,"checksum":"4444444444444444444444444444444444444444444444444444444444444444","presented":true}`, 1)
	raw = strings.Replace(raw, `{"name":"linux-x64 framebuffer present evidence","kind":"positive","ran":true,"pass":true}`,
		`{"name":"linux-x64 framebuffer present evidence","kind":"positive","ran":true,"pass":true},
    {"name":"linux-x64 host event sequence","kind":"positive","ran":true,"pass":true},
    {"name":"linux-x64 app-presented RGBA checksum","kind":"positive","ran":true,"pass":true},
    {"name":"linux-x64 counter component app-presented frame","kind":"positive","ran":true,"pass":true}`, 1)
	return []byte(raw)
}
func validLinuxX64RealWindowSurfaceReportJSON() []byte {
	raw := string(validLinuxX64SurfaceReportJSON())
	replacements := []struct {
		old string
		new string
	}{
		{old: `"source": "examples/surface_counter.tetra"`, new: `"source": "examples/surface_window_counter.tetra"`},
		{old: `"host_evidence": {"level":"linux-x64-memfd-starter","backend":"memfd-rgba","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false}`,
			new: `"host_evidence": {"level":"linux-x64-real-window","backend":"wayland-shm-rgba","framebuffer":true,"real_window":true,"native_input":true,"user_facing_platform_widgets":false}`},
		{old: `examples/surface_counter.tetra`, new: `examples/surface_window_counter.tetra`},
		{old: `/tmp/surface-artifacts/surface-counter`, new: `/tmp/surface-artifacts/surface-window-counter`},
		{old: `examples.surface_counter.CounterApp`, new: `examples.surface_window_counter.CounterApp`},
		{old: `examples.surface_counter.CounterButton`, new: `examples.surface_window_counter.CounterButton`},
	}
	for _, repl := range replacements {
		raw = strings.ReplaceAll(raw, repl.old, repl.new)
	}
	raw = strings.Replace(raw, `,
    {"name":"surface linux-x64 event sequence probe build","kind":"build","path":"/tmp/tetra build event sequence probe","ran":true,"pass":true,"exit_code":0},
    {"name":"surface linux-x64 event sequence probe","kind":"app","path":"/tmp/surface-event-sequence-probe","ran":true,"pass":true,"exit_code":42,"expected_exit_code":42},
    {"name":"surface linux-x64 counter app presented frame probe","kind":"app","path":"/tmp/surface-artifacts/surface-window-counter-present-probe","ran":true,"pass":true,"exit_code":-1,"expected_exit_code":-1}`,
		`,
    {"name":"surface linux-x64 real-window probe","kind":"app","path":"/tmp/surface-artifacts/surface-real-window-probe","ran":true,"pass":true,"exit_code":42,"expected_exit_code":42}`,
		1)
	raw = strings.Replace(raw, `{"order":4,"width":320,"height":200,"stride":1280,"checksum":"4444444444444444444444444444444444444444444444444444444444444444","presented":true}`,
		`{"order":4,"width":320,"height":200,"stride":1280,"checksum":"4444444444444444444444444444444444444444444444444444444444444444","presented":true},
    {"order":5,"width":400,"height":240,"stride":1600,"checksum":"5555555555555555555555555555555555555555555555555555555555555555","presented":true}`, 1)
	raw = strings.Replace(raw, `{"order":3,"kind":"text_input","target_component":"CounterButton","dispatch_path":["CounterApp","CounterButton"],"handled":true,"pass":true,"x":0,"y":0,"key":0,"width":320,"height":200,"timestamp_ms":1,"text_len":2,"text_bytes_hex":"4f4b","buffer_slots":[8,0,0,0,0,320,200,1,2],"before_state":{"CounterApp.text_count":"0","CounterButton.text_len_seen":"0"},"after_state":{"CounterApp.text_count":"1","CounterButton.text_len_seen":"2"}}`,
		`{"order":3,"kind":"key_down","target_component":"CounterApp","dispatch_path":["CounterApp"],"handled":true,"pass":true,"x":0,"y":0,"key":32,"width":320,"height":200,"timestamp_ms":1,"buffer_slots":[6,0,0,0,32,320,200,1,0],"before_state":{"CounterApp.key_count":"0"},"after_state":{"CounterApp.key_count":"1"}},
    {"order":4,"kind":"resize","target_component":"CounterApp","dispatch_path":["CounterApp"],"handled":true,"pass":true,"x":0,"y":0,"key":0,"width":400,"height":240,"timestamp_ms":2,"buffer_slots":[2,0,0,0,0,400,240,2,0],"before_state":{"CounterApp.width":"320","CounterApp.height":"200"},"after_state":{"CounterApp.width":"400","CounterApp.height":"240"}},
    {"order":5,"kind":"text_input","target_component":"CounterButton","dispatch_path":["CounterApp","CounterButton"],"handled":true,"pass":true,"x":0,"y":0,"key":0,"width":400,"height":240,"timestamp_ms":3,"text_len":2,"text_bytes_hex":"4f4b","buffer_slots":[8,0,0,0,0,400,240,3,2],"before_state":{"CounterApp.text_count":"0","CounterButton.text_len_seen":"0"},"after_state":{"CounterApp.text_count":"1","CounterButton.text_len_seen":"2"}},
    {"order":6,"kind":"close","target_component":"CounterApp","dispatch_path":["CounterApp"],"handled":true,"pass":true,"x":0,"y":0,"key":0,"width":400,"height":240,"timestamp_ms":4,"buffer_slots":[1,0,0,0,0,400,240,4,0],"before_state":{"CounterApp.closed":"false"},"after_state":{"CounterApp.closed":"true"}}`,
		1)
	raw = strings.Replace(raw, `{"order":2,"component":"CounterApp","field":"text_count","before":"0","after":"1","cause":"text_input"}`,
		`{"order":2,"component":"CounterApp","field":"key_count","before":"0","after":"1","cause":"key_down"},
    {"order":3,"component":"CounterApp","field":"width","before":"320","after":"400","cause":"resize"},
    {"order":4,"component":"CounterApp","field":"text_count","before":"0","after":"1","cause":"text_input"},
    {"order":5,"component":"CounterApp","field":"closed","before":"false","after":"true","cause":"close"}`, 1)
	raw = strings.Replace(raw, `{"name":"linux-x64 counter component app-presented frame","kind":"positive","ran":true,"pass":true}`,
		`{"name":"linux-x64 counter component app-presented frame","kind":"positive","ran":true,"pass":true},
    {"name":"linux-x64 real-window surface","kind":"positive","ran":true,"pass":true},
    {"name":"linux-x64 native input event pump","kind":"positive","ran":true,"pass":true},
    {"name":"linux-x64 real-window resize event","kind":"positive","ran":true,"pass":true},
    {"name":"linux-x64 real-window close event","kind":"positive","ran":true,"pass":true}`, 1)
	return []byte(raw)
}
func validWASM32WebSurfaceReportJSON() []byte {
	raw := string(validHeadlessSurfaceReportJSON())
	replacements := []struct {
		old string
		new string
	}{
		{old: `"target": "headless"`, new: `"target": "wasm32-web"`},
		{old: `"runtime": "surface-headless"`, new: `"runtime": "surface-wasm32-web"`},
		{old: `"host_evidence": {"level":"deterministic-headless","backend":"software-rgba","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false}`,
			new: `"host_evidence": {"level":"wasm32-web-compiler-owned-loader","backend":"node-surface-host","framebuffer":true,"real_window":false,"native_input":false,"user_facing_platform_widgets":false}`},
		{old: `{"name":"surface component app","kind":"app","path":"/tmp/surface-artifacts/surface-counter","ran":true,"pass":true,"exit_code":1,"expected_exit_code":1}`,
			new: `{"name":"surface wasm32-web component app","kind":"app","path":"node scripts/tools/web_run_module.mjs /tmp/surface-artifacts/surface-counter.wasm","ran":true,"pass":true,"exit_code":1,"expected_exit_code":1},
    {"name":"surface wasm32-web import validator","kind":"runtime","path":"go run ./tools/cmd/validate-wasm-imports --target wasm32-web /tmp/surface-artifacts/surface-counter.wasm","ran":true,"pass":true,"exit_code":0}`},
		{old: `{"kind":"component-app","path":"/tmp/surface-artifacts/surface-counter","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":49172}`,
			new: `{"kind":"component-app","path":"/tmp/surface-artifacts/surface-counter.wasm","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":7502}`},
		{old: `"surface headless runtime"`, new: `"surface wasm32-web runtime"`},
		{old: `"headless event dispatch"`, new: `"wasm32-web Surface Host ABI imports"`},
		{old: `"headless framebuffer checksum"`, new: `"wasm32-web framebuffer checksum evidence"`},
		{old: `"artifact_scan": {"root":"/tmp/surface-artifacts","files_checked":2`, new: `"artifact_scan": {"root":"/tmp/surface-artifacts","files_checked":3`},
	}
	for _, repl := range replacements {
		raw = strings.Replace(raw, repl.old, repl.new, 1)
	}
	raw = strings.Replace(raw, `,
    {"kind":"runner-trace","path":"/tmp/surface-artifacts/surface-runner-trace.json","sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","size":409}`, ``, 1)
	raw = strings.Replace(raw, `,
    {"name":"headless actual runner trace","kind":"positive","ran":true,"pass":true}`, ``, 1)
	raw = strings.Replace(raw, `"artifacts": [
    {"kind":"component-app","path":"/tmp/surface-artifacts/surface-counter.wasm","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":7502}
  ]`, `"artifacts": [
    {"kind":"component-app","path":"/tmp/surface-artifacts/surface-counter.wasm","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":7502},
    {"kind":"compiler-owned-loader","path":"/tmp/surface-artifacts/surface-counter.mjs","sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","size":4931},
    {"kind":"runner-trace","path":"/tmp/surface-artifacts/surface-runner-trace.json","sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","size":413}
  ]`, 1)
	raw = strings.Replace(raw, `{"order":2,"width":320,"height":200,"stride":1280,"checksum":"2222222222222222222222222222222222222222222222222222222222222222","presented":true}`,
		`{"order":2,"width":320,"height":200,"stride":1280,"checksum":"2222222222222222222222222222222222222222222222222222222222222222","presented":true},
    {"order":3,"width":320,"height":200,"stride":1280,"checksum":"3333333333333333333333333333333333333333333333333333333333333333","presented":true},
    {"order":4,"width":320,"height":200,"stride":1280,"checksum":"4444444444444444444444444444444444444444444444444444444444444444","presented":true}`, 1)
	raw = strings.Replace(raw, `{"name":"wasm32-web framebuffer checksum evidence","kind":"positive","ran":true,"pass":true}`,
		`{"name":"wasm32-web framebuffer checksum evidence","kind":"positive","ran":true,"pass":true},
    {"name":"compiler-owned wasm Surface loader","kind":"positive","ran":true,"pass":true},
    {"name":"wasm32-web actual presented frame trace","kind":"positive","ran":true,"pass":true}`, 1)
	return []byte(raw)
}
func validWASM32WebBrowserCanvasSurfaceReportJSON() []byte {
	return []byte(`{
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
    {"name":"tetra build","kind":"build","path":"tetra build --target wasm32-web examples/surface_browser_counter.tetra -o /tmp/surface-artifacts/surface-browser-counter.wasm","ran":true,"pass":true,"exit_code":0},
    {"name":"surface wasm32-web browser canvas component app","kind":"app","path":"/usr/bin/chromium --headless <surface-browser-canvas-runner> wasm=/tmp/surface-artifacts/surface-browser-counter.wasm","ran":true,"pass":true,"exit_code":0,"expected_exit_code":0},
    {"name":"surface wasm32-web import validator","kind":"runtime","path":"go run ./tools/cmd/validate-wasm-imports --target wasm32-web /tmp/surface-artifacts/surface-browser-counter.wasm","ran":true,"pass":true,"exit_code":0},
    {"name":"surface wasm32-web browser canvas runtime","kind":"runtime","path":"Chromium fixture","ran":true,"pass":true,"exit_code":0}
  ],
  "artifacts": [
    {"kind":"component-app","path":"/tmp/surface-artifacts/surface-browser-counter.wasm","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":8604},
    {"kind":"compiler-owned-loader","path":"/tmp/surface-artifacts/surface-browser-counter.mjs","sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","size":4939},
    {"kind":"runner-trace","path":"/tmp/surface-artifacts/surface-runner-trace.json","sha256":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","size":1184}
  ],
  "artifact_scan": {"root":"/tmp/surface-artifacts","files_checked":3,"forbidden_paths":[],"pass":true},
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
}`)
}
