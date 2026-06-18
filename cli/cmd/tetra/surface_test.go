package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSurfaceInspectWritesSnapshot(t *testing.T) {
	dir := t.TempDir()
	reportPath := filepath.Join(dir, "surface-runtime.json")
	outPath := filepath.Join(dir, "surface-inspector.json")
	if err := os.WriteFile(reportPath, []byte(surfaceInspectorCLIReportJSON), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"surface", "inspect", "--report", reportPath, "--out", outPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "surface inspector snapshot") {
		t.Fatalf("stdout = %q, want snapshot write confirmation", stdout.String())
	}

	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	var snapshot struct {
		Schema       string `json:"schema"`
		Level        string `json:"level"`
		Source       string `json:"source"`
		ReleaseScope string `json:"release_scope"`
		Summary      struct {
			ComponentCount          int  `json:"component_count"`
			LayoutBoxCount          int  `json:"layout_box_count"`
			PaintLayerCount         int  `json:"paint_layer_count"`
			AccessibilityNodeCount  int  `json:"accessibility_node_count"`
			PerformanceCounterCount int  `json:"performance_counter_count"`
			DocsOnly                bool `json:"docs_only"`
		} `json:"summary"`
		SourceLocations []struct {
			ID   string `json:"id"`
			Path string `json:"path"`
			Line int    `json:"line"`
		} `json:"source_locations"`
	}
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatalf("unmarshal snapshot: %v\n%s", err, raw)
	}
	if snapshot.Schema != "tetra.surface.inspector-snapshot.v1" || snapshot.Level != "surface-inspector-json-mvp-v1" {
		t.Fatalf("snapshot schema/level = %s/%s", snapshot.Schema, snapshot.Level)
	}
	if snapshot.Source != "examples/surface_counter.tetra" || snapshot.ReleaseScope != "surface-v1-linux-web" {
		t.Fatalf("snapshot source/scope = %s/%s", snapshot.Source, snapshot.ReleaseScope)
	}
	if snapshot.Summary.ComponentCount != 2 || snapshot.Summary.LayoutBoxCount == 0 || snapshot.Summary.PaintLayerCount == 0 || snapshot.Summary.AccessibilityNodeCount == 0 || snapshot.Summary.PerformanceCounterCount == 0 || snapshot.Summary.DocsOnly {
		t.Fatalf("snapshot summary = %#v, want real inspector views", snapshot.Summary)
	}
	if len(snapshot.SourceLocations) == 0 || snapshot.SourceLocations[0].Path == "" || snapshot.SourceLocations[0].Line <= 0 {
		t.Fatalf("source locations = %#v, want source mapping evidence", snapshot.SourceLocations)
	}
}

const surfaceInspectorCLIReportJSON = `{
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
    {"id":"CounterButton","type":"examples.surface_counter.CounterButton","parent":"CounterApp","bounds":{"x":32,"y":80,"w":160,"h":48},"abilities":["measure","layout","draw","event","focus","text","accessibility"],"state":{"pressed":"false","focused":"true","text_len_seen":"2","accessibility_role":"button","label":"Counter"}}
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
}`
