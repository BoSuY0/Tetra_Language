package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSurfaceBlockContractAcceptsContract(t *testing.T) {
	path := writeBlockContractFixture(t, validBlockContractFixture(t))
	if err := validateSurfaceBlockContract(path); err != nil {
		t.Fatalf("validateSurfaceBlockContract failed: %v", err)
	}
}

func TestValidateSurfaceBlockContractRejectsCorePrimitiveButton(t *testing.T) {
	fixture := validBlockContractFixture(t)
	fixture["core_primitives"] = []any{"Block", "Button"}
	path := writeBlockContractFixture(t, fixture)

	err := validateSurfaceBlockContract(path)
	if err == nil {
		t.Fatalf("expected Button core primitive to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "button") {
		t.Fatalf("error = %v, want Button diagnostic", err)
	}
}

func TestValidateSurfaceBlockContractRejectsMissingBlockPrimitive(t *testing.T) {
	fixture := validBlockContractFixture(t)
	fixture["core_primitives"] = []any{}
	path := writeBlockContractFixture(t, fixture)

	err := validateSurfaceBlockContract(path)
	if err == nil {
		t.Fatalf("expected missing Block primitive to fail")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "block") {
		t.Fatalf("error = %v, want Block diagnostic", err)
	}
}

func TestValidateSurfaceBlockContractRejectsGraphWithoutRendererEvidence(t *testing.T) {
	report := map[string]any{
		"schema":          "tetra.surface.runtime.v1",
		"core_primitives": []any{"Block"},
		"block_graph": map[string]any{
			"schema": "tetra.surface.block-graph.v1",
			"nodes":  []any{},
		},
	}
	path := writeBlockReportFixture(t, report)

	err := validateSurfaceBlockContractReport(path)
	if err == nil {
		t.Fatalf("expected block_graph without paint/layout/a11y evidence to fail")
	}
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "paint") || !strings.Contains(lower, "layout") || !strings.Contains(lower, "accessibility") {
		t.Fatalf("error = %v, want paint/layout/accessibility diagnostics", err)
	}
}

func TestValidateSurfaceBlockContractAcceptsIndependentReportShape(t *testing.T) {
	report := map[string]any{
		"schema":          "tetra.surface.runtime.v1",
		"target":          "headless",
		"source":          "examples/unrelated_block_contract_fixture.tetra",
		"core_primitives": []any{"Block"},
		"block_graph": map[string]any{
			"schema": "tetra.surface.block-graph.v1",
			"nodes":  []any{map[string]any{"id": 1, "parent_id": -1}},
		},
		"paint_commands": []any{map[string]any{"kind": "fill"}},
		"layout_passes":  []any{map[string]any{"block_id": 1}},
		"block_accessibility_tree": map[string]any{
			"schema": "tetra.surface.block-accessibility-tree.v1",
			"nodes":  []any{map[string]any{"id": 1}},
		},
	}
	path := writeBlockReportFixture(t, report)

	if err := validateSurfaceBlockContractReport(path); err != nil {
		t.Fatalf("validateSurfaceBlockContractReport failed: %v", err)
	}
}

func validBlockContractFixture(t *testing.T) map[string]any {
	t.Helper()
	raw := []byte(`{
  "schema": "tetra.surface.block.contract.v1",
  "status": "contract-freeze",
  "surface_scope": "surface-block-system-linux-web",
  "core_primitives": ["Block"],
  "forbidden_core_primitives": ["Button", "Card", "TextField", "TextBox", "Sidebar", "Modal"],
  "abi": {
    "schema": "tetra.surface.block.abi.v1",
    "max_return_slots": 10,
    "block_slots": ["id", "parent_id", "props"],
    "block_props_slots": ["layout_mode", "paint_layers", "text_len", "visual_asset", "interaction_flags", "state_flags", "motion_ms", "accessibility_role"]
  },
  "report_schemas": {
    "block_graph": "tetra.surface.block-graph.v1",
    "resolved_block": "tetra.surface.resolved-block.v1",
    "paint_command": "tetra.surface.paint-command.v1",
    "layout_pass": "tetra.surface.layout-pass.v1",
    "accessibility_node": "tetra.surface.block-accessibility-node.v1"
  },
  "renderer_contract": {
    "software_renderer": true,
    "allowed_renderers": ["software-rgba-headless", "wayland-shm-rgba", "browser-canvas-rgba"],
    "required_report_sections": ["block_graph", "paint_commands", "layout_passes", "block_accessibility_tree"]
  },
  "compatibility_wrappers": ["Button", "Card", "TextField", "TextBox", "Sidebar", "Modal"],
  "nonclaims": ["no core Button primitive", "no CSS layout parity", "no GPU renderer production claim"]
}`)
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	return out
}

func writeBlockContractFixture(t *testing.T, value map[string]any) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "surface-block-contract.json")
	writeJSONFixture(t, path, value)
	return path
}

func writeBlockReportFixture(t *testing.T, value map[string]any) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "surface-block-report.json")
	writeJSONFixture(t, path, value)
	return path
}

func writeJSONFixture(t *testing.T, path string, value map[string]any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
