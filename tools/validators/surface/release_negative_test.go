package surface

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type releaseNegativeFixture struct {
	Validator     string `json:"validator"`
	Mutation      string `json:"mutation"`
	ExpectedError string `json:"expected_error"`
}

func TestSurfaceReleaseNegativeFixturesRejectFakeClaims(t *testing.T) {
	fixtureDir := filepath.Join("testdata", "release_negative")
	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Fatalf("read release negative fixtures: %v", err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	wantNames := []string{
		"browser_dom_ui_claims_surface.json",
		"browser_node_only_claims_release.json",
		"browser_user_js_claims_surface.json",
		"clipboard_claim_without_transfer.json",
		"legacy_sidecars.json",
		"linux_memfd_claims_release_window.json",
		"linux_real_window_missing_accessibility.json",
		"linux_real_window_missing_clipboard.json",
		"metadata_only_accessibility_claims_production.json",
		"platform_claim_without_probe.json",
		"release_report_experimental_true.json",
		"release_report_production_false.json",
		"release_summary_missing_unsupported_targets.json",
		"screen_reader_claim_without_artifact.json",
		"stale_artifact_hash.json",
		"text_input_missing_composition_trace.json",
		"text_input_missing_utf8_validation.json",
		"toolkit_manual_bookkeeping_true.json",
		"toolkit_missing_checkbox.json",
		"toolkit_missing_scroll.json",
		"toolkit_single_example_claims_production.json",
	}
	if strings.Join(names, "\n") != strings.Join(wantNames, "\n") {
		t.Fatalf("release negative fixture names = %#v, want %#v", names, wantNames)
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(fixtureDir, name))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			var fixture releaseNegativeFixture
			if err := json.Unmarshal(raw, &fixture); err != nil {
				t.Fatalf("decode fixture: %v", err)
			}
			err = runReleaseNegativeFixture(t, fixture)
			if err == nil {
				t.Fatalf("fixture %s unexpectedly passed", name)
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(fixture.ExpectedError)) {
				t.Fatalf("fixture %s error = %v, want %q", name, err, fixture.ExpectedError)
			}
		})
	}
}

func TestSurfaceReleaseRejectsExperimentalTrue(t *testing.T) {
	requireReleaseNegativeFixture(t, "release_report_experimental_true.json")
}

func TestSurfaceReleaseRejectsProductionClaimFalse(t *testing.T) {
	requireReleaseNegativeFixture(t, "release_report_production_false.json")
}

func TestSurfaceReleaseRejectsMetadataOnlyAccessibility(t *testing.T) {
	requireReleaseNegativeFixture(t, "metadata_only_accessibility_claims_production.json")
}

func TestSurfaceReleaseRejectsPlatformClaimWithoutProbe(t *testing.T) {
	requireReleaseNegativeFixture(t, "platform_claim_without_probe.json")
}

func TestSurfaceReleaseRejectsNodeOnlyBrowserRelease(t *testing.T) {
	requireReleaseNegativeFixture(t, "browser_node_only_claims_release.json")
}

func TestSurfaceReleaseRejectsMemfdLinuxRelease(t *testing.T) {
	requireReleaseNegativeFixture(t, "linux_memfd_claims_release_window.json")
}

func TestSurfaceReleaseRejectsMissingClipboard(t *testing.T) {
	requireReleaseNegativeFixture(t, "linux_real_window_missing_clipboard.json")
}

func TestSurfaceReleaseRejectsMissingComposition(t *testing.T) {
	requireReleaseNegativeFixture(t, "text_input_missing_composition_trace.json")
}

func TestSurfaceReleaseRejectsToolkitWithoutScroll(t *testing.T) {
	requireReleaseNegativeFixture(t, "toolkit_missing_scroll.json")
}

func TestSurfaceReleaseRejectsToolkitWithoutCheckbox(t *testing.T) {
	requireReleaseNegativeFixture(t, "toolkit_missing_checkbox.json")
}

func TestSurfaceReleaseRejectsToolkitSingleExampleClaim(t *testing.T) {
	requireReleaseNegativeFixture(t, "toolkit_single_example_claims_production.json")
}

func TestSurfaceReleaseRejectsUserJS(t *testing.T) {
	requireReleaseNegativeFixture(t, "browser_user_js_claims_surface.json")
}

func TestSurfaceReleaseRejectsDOMUI(t *testing.T) {
	requireReleaseNegativeFixture(t, "browser_dom_ui_claims_surface.json")
}

func TestSurfaceReleaseRejectsLegacySidecars(t *testing.T) {
	requireReleaseNegativeFixture(t, "legacy_sidecars.json")
}

func requireReleaseNegativeFixture(t *testing.T, name string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "release_negative", name))
	if err != nil {
		t.Fatalf("read release negative fixture %s: %v", name, err)
	}
	var fixture releaseNegativeFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("decode release negative fixture %s: %v", name, err)
	}
	err = runReleaseNegativeFixture(t, fixture)
	if err == nil {
		t.Fatalf("fixture %s unexpectedly passed", name)
	}
	if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(fixture.ExpectedError)) {
		t.Fatalf("fixture %s error = %v, want %q", name, err, fixture.ExpectedError)
	}
}

func runReleaseNegativeFixture(t *testing.T, fixture releaseNegativeFixture) error {
	t.Helper()
	switch fixture.Validator {
	case "release-summary":
		return ValidateReleaseSummary(mutateReleaseSummaryFixture(t, fixture.Mutation))
	case "text-input":
		return ValidateTextInputReport(mutateTextInputFixture(t, fixture.Mutation))
	case "linux-release-window":
		return ValidateReport(validLinuxReleaseWindowSurfaceReportJSON(t, func(report map[string]any) {
			mutateLinuxReleaseWindowFixture(t, report, fixture.Mutation)
		}))
	case "browser-release":
		return ValidateReport(validWASM32WebReleaseBrowserSurfaceReportJSON(t, func(report map[string]any) {
			mutateBrowserReleaseFixture(t, report, fixture.Mutation)
		}))
	case "accessibility-release":
		return ValidateReport(validLinuxReleaseAccessibilitySurfaceReportJSON(t, func(report map[string]any) {
			mutateAccessibilityReleaseFixture(t, report, fixture.Mutation)
		}))
	case "accessibility-metadata":
		return ValidateReport(validHeadlessAccessibilityMetadataSurfaceReportJSON(t, func(report map[string]any) {
			tree := report["accessibility_tree"].(map[string]any)
			tree["experimental"] = false
			tree["production_claim"] = true
		}))
	case "toolkit":
		return ValidateReport(validHeadlessProductionToolkitSurfaceReportJSON(t, func(report map[string]any) {
			mutateToolkitFixture(t, report, fixture.Mutation)
		}))
	default:
		t.Fatalf("unknown fixture validator %q", fixture.Validator)
	}
	return nil
}

func mutateReleaseSummaryFixture(t *testing.T, mutation string) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validSurfaceReleaseSummaryJSON(), &report); err != nil {
		t.Fatalf("decode release summary fixture: %v", err)
	}
	switch mutation {
	case "missing_unsupported_targets":
		report["unsupported_targets"] = []any{"macos-x64", "windows-x64"}
	case "experimental_true":
		report["experimental"] = true
	case "production_false":
		report["production_claim"] = false
	case "stale_artifact_hash":
		report["artifact_hashes_validated"] = false
	default:
		t.Fatalf("unknown release summary mutation %q", mutation)
	}
	return mustMarshalFixture(t, report)
}

func mutateTextInputFixture(t *testing.T, mutation string) []byte {
	t.Helper()
	var report map[string]any
	if err := json.Unmarshal(validSurfaceTextInputReportJSON(), &report); err != nil {
		t.Fatalf("decode text input fixture: %v", err)
	}
	switch mutation {
	case "missing_utf8_validation":
		report["utf8_validation"] = false
	case "missing_composition_trace":
		report["composition_trace"] = map[string]any{"start": true, "update": true, "commit": false, "cancel": true}
	case "clipboard_without_transfer":
		report["clipboard_owned_copy"] = false
	default:
		t.Fatalf("unknown text input mutation %q", mutation)
	}
	return mustMarshalFixture(t, report)
}

func mutateLinuxReleaseWindowFixture(t *testing.T, report map[string]any, mutation string) {
	t.Helper()
	host := report["host_evidence"].(map[string]any)
	switch mutation {
	case "memfd_claims_release":
		host["level"] = "linux-x64-memfd-starter"
	case "missing_clipboard":
		host["clipboard"] = false
	case "missing_accessibility":
		host["accessibility_bridge"] = false
	default:
		t.Fatalf("unknown linux release window mutation %q", mutation)
	}
}

func mutateBrowserReleaseFixture(t *testing.T, report map[string]any, mutation string) {
	t.Helper()
	host := report["host_evidence"].(map[string]any)
	switch mutation {
	case "node_only_claims_release":
		processes := report["processes"].([]any)
		app := processes[1].(map[string]any)
		app["path"] = "node scripts/tools/web_run_module.mjs surface-release-form.wasm"
	case "dom_ui_claims_surface":
		toolkit := report["toolkit"].(map[string]any)
		toolkit["no_dom_ui"] = false
	case "user_js_claims_surface":
		toolkit := report["toolkit"].(map[string]any)
		toolkit["no_user_js"] = false
	case "legacy_sidecars":
		artifacts := report["artifacts"].([]any)
		report["artifacts"] = append(artifacts, map[string]any{
			"kind":   "legacy-ui-sidecar",
			"path":   "/tmp/surface-artifacts/surface-release-form.ui.json",
			"sha256": "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
			"size":   float64(128),
		})
		report["artifact_scan"].(map[string]any)["files_checked"] = float64(4)
	default:
		t.Fatalf("unknown browser release mutation %q", mutation)
	}
	host["level"] = "wasm32-web-browser-canvas-release-v1"
}

func mutateAccessibilityReleaseFixture(t *testing.T, report map[string]any, mutation string) {
	t.Helper()
	tree := report["accessibility_tree"].(map[string]any)
	switch mutation {
	case "platform_claim_without_probe":
		tree["linux_platform_probe"] = false
	case "screen_reader_without_artifact":
		tree["linux_probe_artifact"] = ""
	default:
		t.Fatalf("unknown accessibility release mutation %q", mutation)
	}
}

func mutateToolkitFixture(t *testing.T, report map[string]any, mutation string) {
	t.Helper()
	toolkit := report["toolkit"].(map[string]any)
	switch mutation {
	case "single_example_claims_production":
		toolkit["example_count"] = float64(0)
		toolkit["sources"] = []any{"examples/surface_toolkit_form.tetra"}
	case "missing_scroll":
		toolkit["widget_set"] = removeStringAny(toolkit["widget_set"].([]any), "Scroll")
	case "missing_checkbox":
		toolkit["widget_set"] = removeStringAny(toolkit["widget_set"].([]any), "Checkbox")
	case "manual_bookkeeping_true":
		toolkit["manual_bookkeeping"] = true
	default:
		t.Fatalf("unknown toolkit mutation %q", mutation)
	}
}

func removeStringAny(values []any, remove string) []any {
	var out []any
	for _, value := range values {
		if value == remove {
			continue
		}
		out = append(out, value)
	}
	return out
}

func mustMarshalFixture(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return raw
}
