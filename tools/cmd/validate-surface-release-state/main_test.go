package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestValidateSurfaceReleaseStateAcceptsCurrentLinuxWebScope(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceReleaseStateFixture(t, dir)
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(surfaceReleaseStateManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err != nil {
		t.Fatalf("validateSurfaceReleaseState failed: %v", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsManifestMissingSurfaceFeatureRegistry(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceReleaseStateFixture(t, dir)
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(`{
  "surface_release": {
    "scope": "surface-v1-linux-web",
    "status": "current"
  },
  "docs": [
    "docs/spec/surface_v1.md",
    "docs/user/surface_guide.md",
    "docs/user/examples_index.md"
  ]
}`+"\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected manifest without Surface release feature IDs to fail")
	}
	if !strings.Contains(err.Error(), "ui.surface-core") {
		t.Fatalf("error = %v, want missing Surface release feature diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsMissingLinuxReport(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceReleaseStateFixture(t, dir)
	if err := os.Remove(filepath.Join(dir, "surface-linux-x64-release-window.json")); err != nil {
		t.Fatalf("remove linux report: %v", err)
	}
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(`{"surface_release":{"scope":"surface-v1-linux-web","status":"current"},"docs":["docs/spec/surface_v1.md","docs/user/surface_guide.md","docs/user/examples_index.md"]}`+"\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "surface-v1-linux-web",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected missing linux release report to fail")
	}
	if !strings.Contains(err.Error(), "surface-linux-x64-release-window.json") {
		t.Fatalf("error = %v, want missing linux report diagnostic", err)
	}
}

func surfaceReleaseStateManifestJSON() string {
	return `{
  "surface_release": {
    "scope": "surface-v1-linux-web",
    "status": "current"
  },
  "docs": [
    "docs/spec/surface_v1.md",
    "docs/user/surface_guide.md",
    "docs/user/examples_index.md"
  ],
  "features": [
    {"id":"ui.surface-core","status":"current"},
    {"id":"ui.surface-headless","status":"current"},
    {"id":"ui.surface-linux-x64","status":"current"},
    {"id":"ui.surface-web-wasm","status":"current"},
    {"id":"ui.surface-component-model","status":"current"},
    {"id":"ui.surface-toolkit-v1","status":"current"},
    {"id":"ui.surface-text-input-v1","status":"current"},
    {"id":"ui.surface-accessibility-v1","status":"current"},
    {"id":"ui.surface-macos-x64","status":"unsupported"},
    {"id":"ui.surface-windows-x64","status":"unsupported"},
    {"id":"ui.surface-wasm32-wasi","status":"unsupported"}
  ]
}
`
}

func writeSurfaceReleaseStateFixture(t *testing.T, dir string) {
	t.Helper()
	files := map[string]string{
		"surface-release-summary.json": `{
  "schema": "tetra.surface.release.v1",
  "release_scope": "surface-v1-linux-web",
  "status": "current",
  "production_claim": true,
  "experimental": false,
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
  "browser_surface": "browser-canvas-release-v1",
  "linux_surface": "linux-x64-release-window-v1",
  "artifact_hashes_validated": true,
  "legacy_sidecars": false,
  "dom_ui": false,
  "user_js": false,
  "platform_widgets": false
}`,
		"surface-headless-release-text-input.json": `{
  "schema": "tetra.surface.text-input.v1",
  "target": "headless",
  "source": "examples/surface_release_text_input.tetra",
  "level": "production-text-input-v1",
  "experimental": false,
  "production_claim": true,
  "storage": "owned-utf8-byte-buffer",
  "utf8_validation": true,
  "caret": true,
  "selection": true,
  "backspace": true,
  "delete": true,
  "home_end": true,
  "arrow_left_right": true,
  "composition_events": true,
  "composition_commit": true,
  "composition_cancel": true,
  "clipboard_read": true,
  "clipboard_write": true,
  "clipboard_host_abi": true,
  "clipboard_owned_copy": true,
  "composition_trace": {"start":true,"update":true,"commit":true,"cancel":true},
  "borrowed_view_storage": false,
  "safe_view_lifetime_checked": true,
  "processes": [
    {"name":"tetra build","kind":"build","path":"tetra build --target linux-x64 examples/surface_release_text_input.tetra -o /tmp/surface-artifacts/surface-release-text-input","ran":true,"pass":true,"exit_code":0},
    {"name":"surface component app","kind":"app","path":"/tmp/surface-artifacts/surface-release-text-input","ran":true,"pass":true,"exit_code":1,"expected_exit_code":1},
    {"name":"surface headless runtime","kind":"runtime","path":"tools/cmd/surface-runtime-smoke --mode headless-release-text-input","ran":true,"pass":true,"exit_code":0}
  ],
  "artifacts": [
    {"kind":"component-app","path":"/tmp/surface-artifacts/surface-release-text-input","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":4096},
    {"kind":"runner-trace","path":"/tmp/surface-artifacts/surface-runner-trace.json","sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","size":2048}
  ],
  "artifact_scan": {"root":"/tmp/surface-artifacts","files_checked":2,"forbidden_paths":[],"pass":true},
  "cases": [
    {"name":"host-provided pointer event dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"host event buffer poll_event","kind":"positive","ran":true,"pass":true},
    {"name":"pre/post event frame sequence","kind":"positive","ran":true,"pass":true},
    {"name":"component hierarchy dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component text input scalar dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"host text payload buffer","kind":"positive","ran":true,"pass":true},
    {"name":"component focus dispatch","kind":"positive","ran":true,"pass":true},
    {"name":"component accessibility metadata","kind":"positive","ran":true,"pass":true},
    {"name":"no legacy UI sidecar artifacts","kind":"positive","ran":true,"pass":true},
    {"name":"release text input ASCII insertion","kind":"positive","ran":true,"pass":true},
    {"name":"release text input UTF-8 insertion","kind":"positive","ran":true,"pass":true},
    {"name":"release text input caret home end arrows","kind":"positive","ran":true,"pass":true},
    {"name":"release text input selection replacement","kind":"positive","ran":true,"pass":true},
    {"name":"release text input backspace delete","kind":"positive","ran":true,"pass":true},
    {"name":"release text input clipboard owned copy transfer","kind":"positive","ran":true,"pass":true},
    {"name":"release text input composition start update","kind":"positive","ran":true,"pass":true},
    {"name":"release text input composition commit","kind":"positive","ran":true,"pass":true},
    {"name":"release text input composition cancel","kind":"positive","ran":true,"pass":true},
    {"name":"release text input safe view lifetime checked","kind":"positive","ran":true,"pass":true},
    {"name":"reject legacy UI evidence","kind":"negative","ran":true,"pass":true,"expected_error":"legacy UI evidence rejected"}
  ]
}`,
		"surface-wasm32-web-release-browser.json": `{"schema":"tetra.surface.runtime.v1","status":"pass","target":"wasm32-web","host_evidence":{"level":"wasm32-web-browser-canvas-release-v1","backend":"browser-canvas-rgba-accessible","framebuffer":true,"browser_canvas":true,"browser_input":true,"browser_clipboard":true,"browser_clipboard_harness":"deterministic-browser-clipboard-v1","browser_composition":true,"browser_accessibility_snapshot":true,"browser_accessibility_mirror":true,"user_facing_platform_widgets":false},"source":"examples/surface_release_form.tetra"}`,
		"surface-linux-x64-release-window.json":   `{"schema":"tetra.surface.runtime.v1","status":"pass","target":"linux-x64","host_evidence":{"level":"linux-x64-release-window-v1","backend":"wayland-shm-rgba-release-v1","framebuffer":true,"real_window":true,"native_input":true,"text_input":true,"clipboard":true,"composition":true,"accessibility_bridge":true,"user_facing_platform_widgets":false},"source":"examples/surface_release_form.tetra"}`,
	}
	for name, raw := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(raw+"\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	writeSurfaceReleaseArtifactHashes(t, dir, files)
}

func writeSurfaceReleaseArtifactHashes(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	type artifact struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
		Size   int64  `json:"size"`
		Schema string `json:"schema,omitempty"`
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	artifacts := make([]artifact, 0, len(names))
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s for hash manifest: %v", name, err)
		}
		sum := sha256.Sum256(raw)
		entry := artifact{
			Path:   filepath.ToSlash(name),
			SHA256: "sha256:" + hex.EncodeToString(sum[:]),
			Size:   int64(len(raw)),
		}
		var envelope struct {
			Schema string `json:"schema"`
		}
		if err := json.Unmarshal(raw, &envelope); err == nil {
			entry.Schema = envelope.Schema
		}
		artifacts = append(artifacts, entry)
	}
	manifest := struct {
		Schema    string     `json:"schema"`
		Root      string     `json:"root"`
		Artifacts []artifact `json:"artifacts"`
	}{
		Schema:    "tetra.release-artifact-hashes.v1alpha1",
		Root:      ".",
		Artifacts: artifacts,
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("marshal artifact hashes: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "artifact-hashes.json"), append(raw, '\n'), 0o644); err != nil {
		t.Fatalf("write artifact hashes: %v", err)
	}
}
