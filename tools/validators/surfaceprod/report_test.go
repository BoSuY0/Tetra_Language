package surfaceprod

import (
	"strings"
	"testing"
)

func TestValidateClaimAcceptsScopedLinuxWebProductionClaim(t *testing.T) {
	raw := []byte(validSurfaceProdClaim())
	if err := ValidateClaim(raw); err != nil {
		t.Fatalf("ValidateClaim failed: %v\n%s", err, raw)
	}
}

func TestValidateClaimRejectsFakeElectronReactCSSReplacement(t *testing.T) {
	raw := strings.Replace(validSurfaceProdClaim(),
		`"summary":"Scoped Linux/web Surface production claim with explicit Electron, React, CSS, GPU, accessibility, and cross-platform nonclaims."`,
		`"summary":"Surface fully replaces Electron, React, and CSS for all production UI."`,
		1)
	err := ValidateClaim([]byte(raw))
	requireIssue(t, err, "electron")
	requireIssue(t, err, "all production ui")
}

func TestValidateClaimRejectsFakeCrossPlatformSupport(t *testing.T) {
	raw := strings.Replace(validSurfaceProdClaim(),
		`{"target":"wasm32-web","support_level":"production","evidence":"reports/surface-release-v1/surface-wasm32-web-release-browser.json"}`,
		`{"target":"wasm32-web","support_level":"production","evidence":"reports/surface-release-v1/surface-wasm32-web-release-browser.json"},
    {"target":"windows-x64","support_level":"production","evidence":"docs-only planned Windows support"}`,
		1)
	err := ValidateClaim([]byte(raw))
	requireIssue(t, err, "windows-x64")
	requireIssue(t, err, "unsupported")
}

func TestValidateClaimRejectsFakeGPUProductionClaim(t *testing.T) {
	raw := strings.Replace(validSurfaceProdClaim(),
		`"gpu_production":false`,
		`"gpu_production":true`,
		1)
	err := ValidateClaim([]byte(raw))
	requireIssue(t, err, "gpu")
}

func TestValidateClaimRejectsFakeFullAccessibilityClaim(t *testing.T) {
	raw := strings.Replace(validSurfaceProdClaim(),
		`"full_accessibility_parity":false`,
		`"full_accessibility_parity":true`,
		1)
	err := ValidateClaim([]byte(raw))
	requireIssue(t, err, "accessibility")
}

func TestValidateClaimRejectsMissingTargetHostEvidence(t *testing.T) {
	raw := strings.Replace(validSurfaceProdClaim(),
		`  "target_host_evidence":[
    {"target":"linux-x64","host":"linux-x64","level":"target-host","real_window":true,"native_input":true,"browser_canvas":false,"same_commit":true,"report":"reports/surface-release-v1/surface-linux-x64-release-window.json"},
    {"target":"wasm32-web","host":"chromium-linux","level":"browser-canvas","real_window":false,"native_input":false,"browser_canvas":true,"same_commit":true,"report":"reports/surface-release-v1/surface-wasm32-web-release-browser.json"}
  ],
`,
		`  "target_host_evidence":[],
`,
		1)
	err := ValidateClaim([]byte(raw))
	requireIssue(t, err, "target-host")
	requireIssue(t, err, "linux-x64")
	requireIssue(t, err, "wasm32-web")
}

func requireIssue(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected issue containing %q, got nil", want)
	}
	if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(want)) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func validSurfaceProdClaim() string {
	return `{
  "schema":"tetra.surface.prod-claim.v1",
  "status":"pass",
  "claim_tier":"PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
  "scope":"surface-prod-scoped-linux-web",
  "summary":"Scoped Linux/web Surface production claim with explicit Electron, React, CSS, GPU, accessibility, and cross-platform nonclaims.",
  "producer":"tools/cmd/validate-surface-prod-claim",
  "git_head":"0123456789abcdef0123456789abcdef01234567",
  "git_dirty":false,
  "runtime_dependency_policy":{
    "electron":false,
    "chromium_desktop_shell":false,
    "react_runtime":false,
    "dom_ui":false,
    "css_runtime":false,
    "user_js_app_logic":false,
    "platform_widgets":false
  },
  "capabilities":{
    "renderer":"software-rgba",
    "gpu_production":false,
    "cross_platform_desktop_parity":false,
    "accessibility_level":"scoped-platform-bridge-v1",
    "full_accessibility_parity":false
  },
  "supported_targets":[
    {"target":"headless","support_level":"test-evidence","evidence":"reports/surface-release-v1/surface-headless-release-smoke.json"},
    {"target":"linux-x64","support_level":"production","evidence":"reports/surface-release-v1/surface-linux-x64-release-window.json"},
    {"target":"wasm32-web","support_level":"production","evidence":"reports/surface-release-v1/surface-wasm32-web-release-browser.json"}
  ],
  "unsupported_targets":["macos-x64","windows-x64","wasm32-wasi"],
  "nonclaims":[
    "not a broad Electron replacement",
    "not cross-platform desktop parity",
    "not GPU production rendering",
    "not full accessibility parity",
    "not a CSS cascade runtime"
  ],
  "target_host_evidence":[
    {"target":"linux-x64","host":"linux-x64","level":"target-host","real_window":true,"native_input":true,"browser_canvas":false,"same_commit":true,"report":"reports/surface-release-v1/surface-linux-x64-release-window.json"},
    {"target":"wasm32-web","host":"chromium-linux","level":"browser-canvas","real_window":false,"native_input":false,"browser_canvas":true,"same_commit":true,"report":"reports/surface-release-v1/surface-wasm32-web-release-browser.json"}
  ],
  "gate_evidence":[
    {"name":"surface release state","status":"pass","evidence":"scripts/release/surface/release-gate.sh"},
    {"name":"renderer backend decision gate","status":"pass","evidence":"tools/cmd/validate-surface-renderer-report"},
    {"name":"claim taxonomy negative fixtures","status":"pass","evidence":"tools/validators/surfaceprod/report_test.go"}
  ],
  "cases":[
    {"name":"fake electron/react/css replacement rejected","kind":"negative","ran":true,"pass":true},
    {"name":"fake cross-platform support rejected","kind":"negative","ran":true,"pass":true},
    {"name":"fake gpu production claim rejected","kind":"negative","ran":true,"pass":true},
    {"name":"gpu production without target-host backend reports rejected","kind":"negative","ran":true,"pass":true},
    {"name":"fake full accessibility parity rejected","kind":"negative","ran":true,"pass":true},
    {"name":"missing target-host evidence rejected","kind":"negative","ran":true,"pass":true}
  ]
}
`
}
