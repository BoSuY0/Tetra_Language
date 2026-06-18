package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"tetra_language/tools/validators/surface"
)

func TestRunValidatesSurfaceVisualReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "visual.json")
	if err := os.WriteFile(path, validVisualReportJSONForCLI(), 0o644); err != nil {
		t.Fatalf("write visual report: %v", err)
	}
	if err := run([]string{"--report", path}); err != nil {
		t.Fatalf("run validate-surface-visual-report: %v", err)
	}
}

func TestRunRejectsScreenshotOnlySurfaceVisualReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "visual.json")
	raw := screenshotOnlyVisualReportJSONForCLI(t)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write visual report: %v", err)
	}
	if err := run([]string{"--report", path}); err == nil {
		t.Fatalf("expected screenshot-only visual report to fail")
	}
}

func screenshotOnlyVisualReportJSONForCLI(t *testing.T) []byte {
	t.Helper()
	var report surface.VisualRegressionReport
	if err := json.Unmarshal(validVisualReportJSONForCLI(), &report); err != nil {
		t.Fatalf("decode valid visual report fixture: %v", err)
	}
	report.Apps[0].Targets[0].ScreenshotOnly = true
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("marshal screenshot-only visual report: %v", err)
	}
	return raw
}

func validVisualReportJSONForCLI() []byte {
	return []byte(`{
  "schema": "tetra.surface.visual-regression.v1",
  "status": "pass",
  "git_head": "c0258b63a636775b114d69d31cb7832fc3991b05",
  "golden_set": "surface-visual-regression-v1",
  "golden_hash": "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
  "required_targets": ["headless"],
  "required_sources": ["examples/surface/block_core/surface_block_system.tetra"],
  "apps": [{
    "name": "surface-block-system",
    "source": "examples/surface/block_core/surface_block_system.tetra",
    "reference_app": true,
    "targets": [{
      "target": "headless",
      "runtime_report": "reports/surface-visual/headless/surface-headless-block-system.json",
      "runtime_schema": "` + surface.SchemaV1 + `",
      "git_head": "c0258b63a636775b114d69d31cb7832fc3991b05",
      "golden_git_head": "c0258b63a636775b114d69d31cb7832fc3991b05",
      "renderer": "software-rgba",
      "block_graph_evidence": true,
      "token_theme_evidence": true,
      "layout_evidence": true,
      "accessibility_evidence": true,
      "performance_evidence": true,
      "frames": [{
        "order": 1,
        "label": "initial",
        "width": 320,
        "height": 200,
        "stride": 1280,
        "checksum": "sha256:1111111111111111111111111111111111111111111111111111111111111111",
        "golden_checksum": "sha256:1111111111111111111111111111111111111111111111111111111111111111",
        "artifact_path": "reports/surface-visual/headless/frames/initial.rgba",
        "artifact_sha256": "sha256:1111111111111111111111111111111111111111111111111111111111111111",
        "artifact_format": "rgba",
        "golden_artifact_path": "reports/surface/goldens/headless/initial.rgba",
        "golden_artifact_sha256": "sha256:1111111111111111111111111111111111111111111111111111111111111111",
        "tolerance_pixels": 4,
        "tolerance_ratio_milli": 1,
        "tolerance_channel_delta": 1,
        "pass": true
      }]
    }]
  }],
  "negative_guards": {
    "screenshot_only_rejected": true,
    "stale_golden_rejected": true,
    "major_drift_rejected": true,
    "missing_block_graph_rejected": true,
    "missing_layout_rejected": true,
    "missing_accessibility_rejected": true,
    "missing_performance_rejected": true,
    "self_golden_rejected": true,
    "metadata_checksum_rejected": true,
    "fixture_frame_only_rejected": true,
    "missing_png_or_rgba_artifact_rejected": true
  }
}`)
}
