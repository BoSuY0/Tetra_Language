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

	"tetra_language/tools/validators/surface"
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

func TestValidateSurfaceReleaseStateRejectsMissingMorphReport(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceReleaseStateFixture(t, dir)
	if err := os.Remove(filepath.Join(dir, "morph", "headless", "surface-headless-morph.json")); err != nil {
		t.Fatalf("remove morph report: %v", err)
	}
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
	if err == nil {
		t.Fatalf("expected missing morph report to fail")
	}
	if !strings.Contains(err.Error(), "surface-headless-morph.json") {
		t.Fatalf("error = %v, want missing morph report diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateAcceptsProdScopeWithProdGateReport(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceReleaseStateFixture(t, dir)
	writeSurfaceProdGateFixture(t, dir, validSurfaceProdGateReportJSON())
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(surfaceReleaseStateManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
		ManifestPath:   manifestPath,
	})
	if err != nil {
		t.Fatalf("validateSurfaceReleaseState prod scope failed: %v", err)
	}
}

func TestValidateSurfaceReleaseStateRequiresProdGateReportForProdClaim(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceReleaseStateFixture(t, dir)
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(surfaceReleaseStateManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected missing prod gate report to fail")
	}
	if !strings.Contains(err.Error(), "surface-prod-gate-report.json") {
		t.Fatalf("error = %v, want missing prod gate report diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsSkippedTargetCountedAsPass(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceReleaseStateFixture(t, dir)
	raw := strings.Replace(validSurfaceProdGateReportJSON(),
		`{"target":"linux-x64","tier":"prod","ran":true,"pass":true,"skipped":false}`,
		`{"target":"linux-x64","tier":"prod","ran":true,"pass":true,"skipped":true}`,
		1)
	writeSurfaceProdGateFixture(t, dir, raw)
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(surfaceReleaseStateManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected skipped target counted as pass to fail")
	}
	if !strings.Contains(err.Error(), "skipped target") {
		t.Fatalf("error = %v, want skipped target diagnostic", err)
	}
}

func TestValidateSurfaceReleaseStateRejectsMissingProdArtifactHashManifest(t *testing.T) {
	dir := t.TempDir()
	writeSurfaceReleaseStateFixture(t, dir)
	raw := strings.Replace(validSurfaceProdGateReportJSON(),
		`"artifact_hash_manifest":"surface-release-v1/artifact-hashes.json"`,
		`"artifact_hash_manifest":""`,
		1)
	writeSurfaceProdGateFixture(t, dir, raw)
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifestPath, []byte(surfaceReleaseStateManifestJSON()), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	err := validateSurfaceReleaseState(surfaceReleaseStateOptions{
		ReportDir:      dir,
		ExpectedStatus: "current",
		Scope:          "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
		ManifestPath:   manifestPath,
	})
	if err == nil {
		t.Fatalf("expected missing prod artifact hash manifest to fail")
	}
	if !strings.Contains(err.Error(), "artifact hash manifest") {
		t.Fatalf("error = %v, want artifact hash diagnostic", err)
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
    {"id":"ui.surface-block-system","status":"experimental"},
    {"id":"ui.surface-morph-capsule","status":"experimental"},
    {"id":"ui.surface-gpu","status":"experimental"},
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

func writeSurfaceProdGateFixture(t *testing.T, dir string, raw string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "surface-prod-gate-report.json"), []byte(raw+"\n"), 0o644); err != nil {
		t.Fatalf("write surface-prod-gate-report.json: %v", err)
	}
}

func validSurfaceProdGateReportJSON() string {
	return `{
  "schema":"tetra.surface.prod-gate-report.v1",
  "status":"pass",
  "level":"surface-production-ci-release-gate-v1",
  "scope":"PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
  "release_scope":"surface-v1-linux-web",
  "producer":"scripts/release/surface/prod-gate.sh",
  "git_head":"0123456789abcdef0123456789abcdef01234567",
  "same_commit":true,
  "ci_jobs":[
    {"workflow":".github/workflows/release-packages.yml","job":"release-packages","required":true,"continue_on_error":false,"command":"bash scripts/release/surface/prod-gate.sh","artifact_upload":"surface-production-final"}
  ],
  "gates":[
    {"name":"surface-release","report_dir":"surface-release-v1","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-release-v1/artifact-hashes.json"},
    {"name":"block-system","report_dir":"surface-release-v1/block-system","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-release-v1/artifact-hashes.json"},
    {"name":"morph","report_dir":"surface-release-v1/morph","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-release-v1/artifact-hashes.json"},
    {"name":"visual","report_dir":"surface-visual-regression","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-visual-regression/artifact-hashes.json"},
    {"name":"package","report_dir":"surface-package-distribution","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-package-distribution/artifact-hashes.json"},
    {"name":"security","report_dir":"surface-security-sandbox","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-security-sandbox/artifact-hashes.json"},
    {"name":"ipc-lifecycle","report_dir":"surface-ipc-lifecycle","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-ipc-lifecycle/artifact-hashes.json"},
    {"name":"crash-diagnostics","report_dir":"surface-crash-diagnostics","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-crash-diagnostics/artifact-hashes.json"},
    {"name":"i18n-localization","report_dir":"surface-i18n-localization","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-i18n-localization/artifact-hashes.json"},
    {"name":"performance-memory","report_dir":"surface-performance-memory","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-performance-memory/artifact-hashes.json"},
    {"name":"widget-migration","report_dir":"surface-widget-migration","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-widget-migration/artifact-hashes.json"},
    {"name":"example-suite","report_dir":"surface-example-suite","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-example-suite/artifact-hashes.json"},
    {"name":"api-stability","report_dir":"surface-api-stability-v1","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-api-stability-v1/artifact-hashes.json"},
    {"name":"electron-comparison","report_dir":"surface-electron-comparison","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-electron-comparison/artifact-hashes.json"},
    {"name":"prod-claim","report_dir":"surface-prod-claim","ran":true,"pass":true,"skipped":false,"artifact_hash_manifest":"surface-prod-claim/artifact-hashes.json"}
  ],
  "targets":[
    {"target":"linux-x64","tier":"prod","ran":true,"pass":true,"skipped":false},
    {"target":"wasm32-web","tier":"prod","ran":true,"pass":true,"skipped":false},
    {"target":"windows-x64","tier":"beta","ran":false,"pass":false,"skipped":true},
    {"target":"macos-x64","tier":"beta","ran":false,"pass":false,"skipped":true}
  ],
  "artifact_hashes_validated":true,
  "negative_guards":{
    "missing_job_rejected":true,
    "continue_on_error_rejected":true,
    "skipped_target_as_pass_rejected":true,
    "missing_artifact_hash_manifest_rejected":true
  },
  "cases":[
    {"name":"release-packages production gate job required","kind":"positive","ran":true,"pass":true},
    {"name":"no continue-on-error production jobs","kind":"negative","ran":true,"pass":true},
    {"name":"skipped target counted as pass rejected","kind":"negative","ran":true,"pass":true},
    {"name":"artifact hash manifest missing rejected","kind":"negative","ran":true,"pass":true}
  ]
}`
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
  "producer": "scripts/release/surface/release-gate.sh",
  "git_head": "0123456789abcdef0123456789abcdef01234567",
  "version": "tetra_language",
  "git_dirty": false,
  "host_os": "linux",
  "host_arch": "amd64",
  "generated_at_utc": "2026-06-08T16:00:00Z",
  "command_line": "bash scripts/release/surface/release-gate.sh --report-dir reports/surface-release-v1",
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
  "block_system": "block-system",
  "block_system_gate": "tetra.surface.block-system.gate.v1",
  "morph": "morph-capsule",
  "morph_gate": "tetra.surface.morph.gate.v1",
  "artifact_hashes_validated": true,
  "legacy_sidecars": false,
  "dom_ui": false,
  "user_js": false,
  "platform_widgets": false
}`,
		"morph/surface-morph-gate-summary.json": `{
  "schema": "tetra.surface.morph.gate.v1",
  "status": "current",
  "release_scope": "surface-morph-experimental-linux-web",
  "producer": "scripts/release/surface/morph-gate.sh",
  "source": "examples/surface_morph_command_palette.tetra",
  "module": "lib.core.morph",
  "schema_under_test": "tetra.surface.morph.v1",
  "dependency_gate": "tetra.surface.block-system.gate.v1",
  "same_commit_validated": true,
  "headless_report": "headless/surface-headless-morph.json",
  "target_evidence": ["headless"],
  "core_primitives": ["Block"],
  "forbidden_core_primitives": ["Button", "Card", "TextField", "TextBox", "Sidebar", "Modal"],
  "artifact_hashes_validated": true
}`,
		"morph/headless/surface-headless-morph.json": `{
  "schema": "tetra.surface.runtime.v1",
  "status": "pass",
  "target": "headless",
  "source": "examples/surface_morph_command_palette.tetra",
  "morph": {
    "schema": "tetra.surface.morph.v1",
    "quality_level": "deterministic-headless-morph-capsule-v1",
    "source": "examples/surface_morph_command_palette.tetra",
    "module": "lib.core.morph",
    "surface_scope": "surface-morph-experimental-linux-web",
    "production_claim": false
  }
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
  "text_pipeline": {
    "schema": "tetra.surface.text-pipeline.v1",
    "level": "scoped-latin-utf8-text-pipeline-v1",
    "engine": "deterministic-tetra-text-shaper",
    "platform_widget_text_controls": false,
    "font_manifest": [
      {"id":"tetra-ui-regular","family":"Tetra UI","style":"normal","weight":400,"source":"embedded:tetra-ui-regular","sha256":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","bytes":32768},
      {"id":"noto-sans-fallback","family":"Noto Sans","style":"normal","weight":400,"source":"system:fontconfig/noto-sans","sha256":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","bytes":65536}
    ],
    "font_fallbacks": [
      {"id":"release-fallback","requested_family":"Tetra UI","resolved_family":"Noto Sans","chain":["Tetra UI","Noto Sans","monospace"],"missing_glyphs":0,"coverage":"latin-plus-basic-utf8-smoke"}
    ],
    "glyph_runs": [
      {"id":"latin-run","font_family":"Tetra UI","script":"Latin","direction":"ltr","shaping":"tier1-latin-simple","text_len":5,"byte_start":0,"byte_end":5,"scalar_start":0,"scalar_end":5,"glyph_count":5,"glyph_ids":[36,69,70,32,71],"advances":[8,8,8,4,8],"clusters":[0,1,2,3,4],"baseline":14,"checksum":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
      {"id":"fallback-run","font_family":"Noto Sans","script":"Common","direction":"ltr","shaping":"tier1-fallback-simple","text_len":1,"byte_start":5,"byte_end":7,"scalar_start":5,"scalar_end":6,"glyph_count":1,"glyph_ids":[9731],"advances":[9],"clusters":[5],"baseline":14,"checksum":"sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"}
    ],
    "glyph_caches": [
      {"id":"release-glyph-cache","strategy":"bounded-lru","budget_bytes":65536,"used_bytes":8192,"entry_count":24,"eviction":"lru","bounded":true}
    ],
    "cache_budget_bytes": 65536,
    "glyph_cache_budget_bytes": 65536,
    "glyph_cache_used_bytes": 8192,
    "bounded_caches": true,
    "cache_eviction": "lru",
    "unicode_boundaries": {
      "utf8_storage": true,
      "scalar_boundaries": true,
      "cluster_boundaries": true,
      "latin_tier": true,
      "combining_marks": false,
      "bidi": false,
      "unsupported_scripts": ["Arabic","Devanagari","Thai"],
      "boundary_cases": ["ASCII insertion","UTF-8 scalar insertion","cluster caret clamp"]
    },
    "shaping_scope": {
      "tier": "tier1-latin-utf8",
      "supported_scripts": ["Latin","Common"],
      "unsupported_scripts": ["Arabic","Devanagari","Thai"],
      "engine_decision": "deterministic embedded shaper until HarfBuzz-class evidence exists",
      "full_unicode_editor_semantics": false,
      "bidi": false,
      "combining_marks": false,
      "system_library_integration": "not required for Tier 1; future HarfBuzz-class gate",
      "platform_widgets": false
    },
    "measurements": [
      {"id":"release-label-measure","block_id":1,"text_len":18,"font_family":"Tetra UI","font_weight":400,"font_size":14,"line_height":18,"max_width":120,"measured":{"w":108,"h":18},"line_count":1,"wrap":"none","overflow":"clip","ellipsis":false,"ellipsized_text_len":18,"align":"start","quality":"deterministic-metrics-v1","checksum":"sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},
      {"id":"release-ellipsis-measure","block_id":2,"text_len":32,"font_family":"Tetra UI","font_weight":400,"font_size":14,"line_height":18,"max_width":96,"measured":{"w":96,"h":36},"line_count":2,"wrap":"word","overflow":"ellipsis","ellipsis":true,"ellipsized_text_len":20,"align":"start","quality":"deterministic-metrics-v1","checksum":"sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"}
    ],
    "measurement_consistency": {
      "same_input_same_metrics": true,
      "target_independent_baseline": true,
      "max_delta_px": 0,
      "cases": ["latin measurement repeat","fallback measurement repeat","ellipsis measurement repeat"]
    },
    "layout": {"wrap":true,"ellipsis":true,"alignment":["start","center","end"],"baseline":true,"line_height":true},
    "caret_rects": [{"x":32,"y":64,"w":2,"h":18}],
    "selection_rects": [{"x":32,"y":64,"w":18,"h":18}],
    "ime_composition_spans": [
      {"kind":"composition","byte_start":0,"byte_end":3,"scalar_start":0,"scalar_end":3,"rect":{"x":32,"y":64,"w":24,"h":18}}
    ],
    "nonclaims": [
      "full Unicode editor semantics",
      "bidi production shaping",
      "complex script shaping without HarfBuzz-class evidence",
      "platform widget text controls"
    ],
    "negative_guards": {
      "full_unicode_editor_without_tests_rejected": true,
      "missing_font_fallback_rejected": true,
      "unbounded_glyph_cache_rejected": true,
      "platform_widget_text_controls_rejected": true
    }
  },
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
	files["surface-headless-release-text-input.json"] = string(surfaceReleaseStateTextInputReportJSON(t, files["surface-headless-release-text-input.json"]))
	for name, raw := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(raw+"\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	writeSurfaceReleaseArtifactHashes(t, dir, files)
}

func surfaceReleaseStateTextInputReportJSON(t *testing.T, raw string) []byte {
	t.Helper()
	var report surface.TextInputReport
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		t.Fatalf("decode text input fixture: %v", err)
	}
	report.TextEditing = surfaceReleaseStateTextEditingReport(report.Target)
	report.Cases = append(report.Cases,
		surface.CaseReport{Name: "release text editing target IME trace", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "release text editing clipboard owned copies", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "release text editing undo unit boundaries", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "release text editing validation diagnostics", Kind: "positive", Ran: true, Pass: true},
		surface.CaseReport{Name: "release text editing rich text nonclaim", Kind: "positive", Ran: true, Pass: true},
	)
	out, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal text input fixture: %v", err)
	}
	return out
}

func surfaceReleaseStateTextEditingReport(target string) surface.TextEditingReport {
	return surface.TextEditingReport{
		Schema:   surface.TextEditingSchemaV1,
		Level:    "production-editing-basics-v1",
		Target:   target,
		Producer: "tools/cmd/surface-runtime-smoke",
		EditableBlocks: []surface.EditableTextBlockReport{
			{ID: "ReleaseTextBox", Kind: "TextBox", Storage: "owned-utf8-byte-buffer", FormsSafe: true, CommandPaletteSearchSafe: true, MaxBytes: 1024, UTF8Validation: true},
		},
		EditOperations: []surface.TextEditOperationReport{
			{Order: 1, Action: "insert_text", Target: "ReleaseTextBox", BeforeTextLen: 0, AfterTextLen: 3, BeforeCaret: 0, AfterCaret: 3, SelectionBefore: surface.TextSelectionRangeReport{Anchor: 0, Focus: 0}, SelectionAfter: surface.TextSelectionRangeReport{Anchor: 3, Focus: 3}, UndoUnitID: "insert-ada", Checksum: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
			{Order: 2, Action: "move_caret_left", Target: "ReleaseTextBox", BeforeTextLen: 3, AfterTextLen: 3, BeforeCaret: 3, AfterCaret: 2, SelectionBefore: surface.TextSelectionRangeReport{Anchor: 3, Focus: 3}, SelectionAfter: surface.TextSelectionRangeReport{Anchor: 2, Focus: 2}, UndoUnitID: "navigation-left", Checksum: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
			{Order: 3, Action: "replace_selection", Target: "ReleaseTextBox", BeforeTextLen: 5, AfterTextLen: 4, BeforeCaret: 1, AfterCaret: 2, SelectionBefore: surface.TextSelectionRangeReport{Anchor: 1, Focus: 4}, SelectionAfter: surface.TextSelectionRangeReport{Anchor: 2, Focus: 2}, UndoUnitID: "replace-selection", Checksum: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
			{Order: 4, Action: "composition_commit", Target: "ReleaseTextBox", BeforeTextLen: 4, AfterTextLen: 5, BeforeCaret: 4, AfterCaret: 5, SelectionBefore: surface.TextSelectionRangeReport{Anchor: 4, Focus: 4}, SelectionAfter: surface.TextSelectionRangeReport{Anchor: 5, Focus: 5}, UndoUnitID: "ime-commit", Checksum: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
			{Order: 5, Action: "clipboard_write", Target: "ReleaseTextBox", BeforeTextLen: 5, AfterTextLen: 5, BeforeCaret: 5, AfterCaret: 5, SelectionBefore: surface.TextSelectionRangeReport{Anchor: 0, Focus: 5}, SelectionAfter: surface.TextSelectionRangeReport{Anchor: 0, Focus: 5}, UndoUnitID: "clipboard-copy", Checksum: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},
			{Order: 6, Action: "clipboard_read", Target: "ReleaseTextBox", BeforeTextLen: 0, AfterTextLen: 5, BeforeCaret: 0, AfterCaret: 5, SelectionBefore: surface.TextSelectionRangeReport{Anchor: 0, Focus: 0}, SelectionAfter: surface.TextSelectionRangeReport{Anchor: 5, Focus: 5}, UndoUnitID: "clipboard-paste", Checksum: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		},
		SelectionModel: surface.TextSelectionModelReport{
			CaretMovement:        []string{"left", "right", "home", "end"},
			SelectionReplacement: true,
			ScalarBoundaryClamp:  true,
			CaretRects:           []surface.RectReport{{X: 32, Y: 64, W: 2, H: 18}},
			SelectionRects:       []surface.RectReport{{X: 32, Y: 64, W: 24, H: 18}},
		},
		IMETraces: []surface.TextIMETraceReport{
			{
				Target:     target,
				Start:      true,
				Update:     true,
				Commit:     true,
				Cancel:     true,
				EventCount: 4,
				CompositionSpan: surface.TextCompositionSpanReport{
					Kind:        "composition",
					ByteStart:   0,
					ByteEnd:     3,
					ScalarStart: 0,
					ScalarEnd:   3,
					Rect:        surface.RectReport{X: 32, Y: 64, W: 24, H: 18},
				},
				CommittedTextOwnedCopy: true,
			},
		},
		ClipboardTransfers: []surface.TextClipboardTransferReport{
			{Direction: "write", HostABI: "__tetra_surface_clipboard_write_text", Bytes: 5, UTF8Valid: true, OwnedCopy: true, BorrowedView: false, Checksum: "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
			{Direction: "read", HostABI: "__tetra_surface_clipboard_read_text_into", Bytes: 5, UTF8Valid: true, OwnedCopy: true, BorrowedView: false, Checksum: "sha256:2222222222222222222222222222222222222222222222222222222222222222"},
		},
		UndoUnits: []surface.TextUndoUnitReport{
			{ID: "insert-ada", OperationOrders: []int{1}, Boundary: "text-input-operation", Reversible: true, Coalesced: false},
			{ID: "navigation-left", OperationOrders: []int{2}, Boundary: "caret-navigation", Reversible: true, Coalesced: false},
			{ID: "replace-selection", OperationOrders: []int{3}, Boundary: "selection-replacement", Reversible: true, Coalesced: false},
			{ID: "ime-commit", OperationOrders: []int{4}, Boundary: "composition-commit", Reversible: true, Coalesced: false},
			{ID: "clipboard-copy", OperationOrders: []int{5}, Boundary: "clipboard-copy", Reversible: true, Coalesced: false},
			{ID: "clipboard-paste", OperationOrders: []int{6}, Boundary: "clipboard-paste", Reversible: true, Coalesced: false},
		},
		ValidationDiagnostics: []surface.TextEditingDiagnosticReport{
			{Name: "invalid UTF-8 rejected", Ran: true, Pass: true},
			{Name: "borrowed text buffer rejected at host boundary", Ran: true, Pass: true},
			{Name: "IME claim without target trace rejected", Ran: true, Pass: true},
			{Name: "rich text claim rejected", Ran: true, Pass: true},
		},
		HostBoundary: surface.TextEditingHostBoundaryReport{
			CopySafe:                      true,
			ClipboardOwnedCopy:            true,
			CompositionOwnedCopy:          true,
			BorrowedTextBufferCrossesHost: false,
		},
		FormsSafe:                true,
		CommandPaletteSearchSafe: true,
		RichText:                 false,
		NonClaims: []string{
			"rich text",
			"full editor-grade text semantics",
			"native platform text controls",
		},
		NegativeGuards: surface.TextEditingNegativeGuardsReport{
			IMEWithoutTargetTraceRejected: true,
			BorrowedTextBufferRejected:    true,
			RichTextClaimRejected:         true,
			UnsafeClipboardAliasRejected:  true,
			InvalidUTF8Rejected:           true,
		},
	}
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
