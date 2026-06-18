package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSurfaceRendererReportAcceptsValidReportFile(t *testing.T) {
	path := writeRendererReportFixture(t, validSurfaceRendererReportJSON())
	if err := validateSurfaceRendererReport(path); err != nil {
		t.Fatalf("validateSurfaceRendererReport failed: %v", err)
	}
}

func TestValidateSurfaceRendererReportRejectsGPUProductionWithoutBackendReports(t *testing.T) {
	raw := strings.Replace(validSurfaceRendererReportJSON(),
		`"production_claim": false`,
		`"production_claim": true`,
		1)
	raw = strings.Replace(raw,
		`"status": "experimental"`,
		`"status": "production"`,
		1)
	path := writeRendererReportFixture(t, raw)

	err := validateSurfaceRendererReport(path)
	if err == nil {
		t.Fatalf("expected GPU production claim without target-host backend reports to fail")
	}
	for _, want := range []string{"gpu", "target-host"} {
		if !strings.Contains(strings.ToLower(err.Error()), want) {
			t.Fatalf("error = %v, want %q", err, want)
		}
	}
}

func writeRendererReportFixture(t *testing.T, raw string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "surface-renderer-backend.json")
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func validSurfaceRendererReportJSON() string {
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
