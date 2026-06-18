package surfacerenderer

import (
	"strings"
	"testing"
)

func TestValidateReportAcceptsSoftwareOnlyDecisionGate(t *testing.T) {
	raw := []byte(validSurfaceRendererReport())
	if err := ValidateReport(raw); err != nil {
		t.Fatalf("ValidateReport failed: %v\n%s", err, raw)
	}
}

func TestValidateReportRejectsGPUProductionWithoutBackendReports(t *testing.T) {
	raw := strings.Replace(validSurfaceRendererReport(),
		`"production_claim": false`,
		`"production_claim": true`,
		1)
	raw = strings.Replace(raw,
		`"status": "experimental"`,
		`"status": "production"`,
		1)
	raw = strings.Replace(raw,
		`"target_host_backend_reports": []`,
		`"target_host_backend_reports": []`,
		1)
	err := ValidateReport([]byte(raw))
	requireRendererIssue(t, err, "gpu")
	requireRendererIssue(t, err, "target-host")
}

func TestValidateReportRejectsMissingCompositorCapability(t *testing.T) {
	raw := strings.Replace(validSurfaceRendererReport(),
		`"texture_atlas"`,
		`"texture-cache"`,
		1)
	err := ValidateReport([]byte(raw))
	requireRendererIssue(t, err, "texture_atlas")
}

func TestValidateReportRejectsDocsGPUProductionOverclaim(t *testing.T) {
	raw := strings.Replace(validSurfaceRendererReport(),
		`"docs_gpu_production_rejected": true`,
		`"docs_gpu_production_rejected": false`,
		1)
	err := ValidateReport([]byte(raw))
	requireRendererIssue(t, err, "docs")
	requireRendererIssue(t, err, "gpu")
}

func requireRendererIssue(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected issue containing %q, got nil", want)
	}
	if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(want)) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func validSurfaceRendererReport() string {
	return `{
  "schema": "tetra.surface.renderer-backend.v1",
  "status": "pass",
  "decision": "software-only-prod-go-gpu-experimental",
  "scope": "surface-prod-scoped-linux-web",
  "producer": "tools/cmd/validate-surface-renderer-report",
  "git_head": "0123456789abcdef0123456789abcdef01234567",
  "software_baseline": {
    "backend": "software-rgba",
    "production_path": true,
    "evidence_schema": "tetra.surface.software-renderer.v1",
    "release_gate": "scripts/release/surface/release-gate.sh",
    "report": "reports/surface-release-v1-p06-rerun/block-system/headless/surface-headless-block-system.json"
  },
  "gpu_compositor": {
    "status": "experimental",
    "production_claim": false,
    "required_capabilities": [
      "layer_compositing",
      "transforms",
      "clipping",
      "texture_atlas",
      "vsync_frame_timing"
    ],
    "target_host_backend_reports": [],
    "fallback": "software-rgba",
    "same_scene_equivalence": false
  },
  "target_host_requirements": [
    "linux target-host GPU smoke",
    "web compositor/canvas evidence",
    "Windows/macOS target-host GPU evidence if claimed"
  ],
  "nonclaims": [
    "GPU renderer production",
    "GPU compositor production",
    "Windows/macOS GPU backend parity"
  ],
  "negative_guards": {
    "gpu_production_without_backend_reports_rejected": true,
    "docs_gpu_production_rejected": true,
    "software_only_prod_stable_allowed": true
  },
  "cases": [
    {"name":"software-only scoped production go decision","kind":"positive","ran":true,"pass":true},
    {"name":"gpu production without target-host backend reports rejected","kind":"negative","ran":true,"pass":true},
    {"name":"docs gpu renderer production overclaim rejected","kind":"negative","ran":true,"pass":true}
  ]
}
`
}
