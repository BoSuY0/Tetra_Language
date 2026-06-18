package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const productSliceTestGitCommit = "95bfd4a887bab5032437cb22494d034e82ae6d35"

func TestValidateProductSliceAcceptsCompleteSummary(t *testing.T) {
	dir := writeProductSliceFixture(t)
	if err := validateProductSlice(productSliceOptions{ReportDir: dir}); err != nil {
		t.Fatalf("validateProductSlice failed: %v", err)
	}
}

func TestValidateProductSliceRejectsMissingFlagshipRuntime(t *testing.T) {
	dir := writeProductSliceFixture(t)
	if err := os.Remove(
		filepath.Join(dir, "flagship", "linux-x64-real-window-block-system.json"),
	); err != nil {
		t.Fatal(err)
	}
	err := validateProductSlice(productSliceOptions{ReportDir: dir})
	if err == nil {
		t.Fatalf("expected missing flagship runtime report to fail")
	}
	if !strings.Contains(err.Error(), "flagship/linux-x64-real-window-block-system.json") {
		t.Fatalf("error = %v, want flagship runtime diagnostic", err)
	}
}

func TestValidateProductSliceRejectsMissingMorphRenderedBeautyGate(t *testing.T) {
	dir := writeProductSliceFixture(t)
	if err := os.Remove(
		filepath.Join(dir, "morph-rendered-beauty", "morph-rendered-beauty-gate-summary.json"),
	); err != nil {
		t.Fatal(err)
	}
	err := validateProductSlice(productSliceOptions{ReportDir: dir})
	if err == nil {
		t.Fatalf("expected missing Morph rendered beauty gate summary to fail")
	}
	if !strings.Contains(
		err.Error(),
		"morph-rendered-beauty/morph-rendered-beauty-gate-summary.json",
	) {
		t.Fatalf("error = %v, want Morph rendered beauty gate diagnostic", err)
	}
}

func TestValidateProductSliceRejectsMorphRenderedBeautyProductSignoff(t *testing.T) {
	dir := writeProductSliceFixture(t)
	path := filepath.Join(dir, "morph-rendered-beauty", "morph-rendered-beauty-gate-summary.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), `"product_claim": false`, `"product_claim": true`, 1))
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	err = validateProductSlice(productSliceOptions{ReportDir: dir})
	if err == nil {
		t.Fatalf("expected Morph rendered beauty product claim to fail for MRB-12 product-slice")
	}
	if !strings.Contains(err.Error(), "morph_rendered_beauty.product_claim") {
		t.Fatalf("error = %v, want Morph rendered beauty product claim diagnostic", err)
	}
}

func TestValidateProductSliceAcceptsPromotedCleanFinalSignoff(t *testing.T) {
	dir := writeProductSliceFixture(t)
	writePromotedProductSliceSummary(t, dir)
	writePromotedProductSliceMRBGate(t, dir)
	writeProductSliceHashManifest(t, dir)

	if err := validateProductSlice(productSliceOptions{ReportDir: dir}); err != nil {
		t.Fatalf("validateProductSlice rejected promoted clean final signoff: %v", err)
	}
}

func TestValidateProductSliceRejectsMissingGitCommit(t *testing.T) {
	dir := writeProductSliceFixture(t)
	path := filepath.Join(dir, "surface-product-slice-summary.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(
		strings.Replace(
			string(raw),
			"  \"git_commit\": \""+productSliceTestGitCommit+"\",\n",
			"",
			1,
		),
	)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	writeProductSliceHashManifest(t, dir)
	err = validateProductSlice(productSliceOptions{ReportDir: dir})
	if err == nil {
		t.Fatalf("expected missing product-slice git_commit to fail")
	}
	if !strings.Contains(err.Error(), "git_commit") {
		t.Fatalf("error = %v, want git_commit diagnostic", err)
	}
}

func TestValidateProductSliceRejectsMismatchedMRBGateGitCommit(t *testing.T) {
	dir := writeProductSliceFixture(t)
	path := filepath.Join(dir, "morph-rendered-beauty", "morph-rendered-beauty-gate-summary.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(
		strings.Replace(
			string(raw),
			`"git_commit": "`+productSliceTestGitCommit+`"`,
			`"git_commit": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`,
			1,
		),
	)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	writeProductSliceHashManifest(t, dir)
	err = validateProductSlice(productSliceOptions{ReportDir: dir})
	if err == nil {
		t.Fatalf("expected mismatched MRB gate git_commit to fail")
	}
	if !strings.Contains(
		err.Error(),
		"morph rendered beauty gate summary git_commit must match git_head",
	) {
		t.Fatalf("error = %v, want MRB gate git_commit mismatch diagnostic", err)
	}
}

func TestValidateProductSliceRejectsMissingNonclaim(t *testing.T) {
	dir := writeProductSliceFixture(t)
	path := filepath.Join(dir, "surface-product-slice-summary.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), `"no-electron-api-compatibility",`, "", 1))
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	writeProductSliceHashManifest(t, dir)
	err = validateProductSlice(productSliceOptions{ReportDir: dir})
	if err == nil {
		t.Fatalf("expected missing nonclaim to fail")
	}
	if !strings.Contains(err.Error(), "no-electron-api-compatibility") {
		t.Fatalf("error = %v, want missing nonclaim diagnostic", err)
	}
}

func writeProductSliceFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, path := range productSliceRequiredArtifacts() {
		if path == "artifact-hashes.json" {
			continue
		}
		full := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	summary := productSliceSummary{
		Schema:       productSliceSummarySchema,
		ReleaseScope: productSliceReleaseScope,
		Producer:     productSliceProducer,
		GitHead:      productSliceTestGitCommit,
		GitCommit:    productSliceTestGitCommit,
		GitDirty:     boolPtrProductSlice(true),
		CommandLine: ("bash scripts/release/surface/surface-product-slice-gate.sh --" +
			"report-dir reports/surface-product-slice/product-gate"),
		FlagshipSource:       "examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra",
		AppID:                "studio-shell",
		ArtifactHashManifest: "artifact-hashes.json",
		ClaimScanner:         "validated",
		Manifest:             "validated",
		Docs:                 "validated",
		MorphRenderedBeauty:  "validated",
		ProductClaim:         boolPtrProductSlice(false),
		FinalSignoff:         boolPtrProductSlice(false),
		Categories: []productSliceCategory{
			{
				Name:         "flagship-runtime",
				Status:       "validated",
				SourceReport: "flagship/flagship-runtime-summary.json",
				Evidence:     "flagship runtime",
				Pass:         true,
			},
			{
				Name:         "developer-loop",
				Status:       "validated",
				SourceReport: "dev-workflow/surface-dev-workflow.json",
				Evidence:     "developer loop",
				Pass:         true,
			},
			{
				Name:         "package-update",
				Status:       "validated",
				SourceReport: "package/surface-package.json",
				Evidence:     "package update",
				Pass:         true,
			},
			{
				Name:         "morph-rendered-beauty",
				Status:       "validated",
				SourceReport: "morph-rendered-beauty/morph-rendered-beauty-gate-summary.json",
				Evidence:     "Morph rendered beauty gate",
				Pass:         true,
			},
			{
				Name:         "claim-governance",
				Status:       "validated",
				SourceReport: "claims/claim-governance-summary.json",
				Evidence:     "claims",
				Pass:         true,
			},
			{
				Name:         "docs-manifest",
				Status:       "validated",
				SourceReport: "docs-manifest/docs-manifest-summary.json",
				Evidence:     "docs manifest",
				Pass:         true,
			},
		},
		RequiredArtifacts: productSliceRequiredArtifacts(),
		Nonclaims: []string{
			"no-electron-api-compatibility",
			"no-react-runtime-claim",
			"no-css-runtime-claim",
			"no-dom-authored-application-ui",
			"nonclaim-macos-surface-production-support",
			"nonclaim-windows-surface-production-support",
			"no-gpu-renderer-parity",
			"no-native-widget-parity",
			"no-signing-or-notarization-claim",
			"no-automatic-network-update-claim",
		},
		Validations: productSliceValidations{
			FlagshipRuntime:     "validated",
			DeveloperLoop:       "validated",
			Package:             "validated",
			MorphRenderedBeauty: "validated",
			Claims:              "validated",
			Manifest:            "validated",
			Docs:                "validated",
			ArtifactHashes:      "validated",
		},
		Pass: true,
	}
	raw, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "surface-product-slice-summary.json"),
		append(raw, '\n'),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	writeProductSliceMRBGateFixture(t, dir)
	writeProductSliceHashManifest(t, dir)
	return dir
}

func writeProductSliceMRBGateFixture(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, "morph-rendered-beauty", "morph-rendered-beauty-gate-summary.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{
  "schema": "tetra.surface.morph-rendered-beauty.gate.v1",
  "status": "validated_with_target_blockers",
  "producer": "scripts/release/surface/morph-rendered-beauty-gate.sh",
  "git_head": "95bfd4a887bab5032437cb22494d034e82ae6d35",
  "git_commit": "95bfd4a887bab5032437cb22494d034e82ae6d35",
  "morph_rendered_beauty_report": "morph-rendered-beauty.json",
  "morph_to_pixels_report": "morph-to-pixels.json",
  "product_claim": false,
  "final_signoff": false,
  "pass": true
}
`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writePromotedProductSliceSummary(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, "surface-product-slice-summary.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	text = strings.Replace(text, `"git_dirty": true`, `"git_dirty": false`, 1)
	text = strings.Replace(text, `"product_claim": false`, `"product_claim": true`, 1)
	text = strings.Replace(text, `"final_signoff": false`, `"final_signoff": true`, 1)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writePromotedProductSliceMRBGate(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, "morph-rendered-beauty", "morph-rendered-beauty-gate-summary.json")
	raw := []byte(`{
  "schema": "tetra.surface.morph-rendered-beauty.gate.v1",
  "status": "validated",
  "producer": "scripts/release/surface/morph-rendered-beauty-gate.sh",
  "git_head": "95bfd4a887bab5032437cb22494d034e82ae6d35",
  "git_commit": "95bfd4a887bab5032437cb22494d034e82ae6d35",
  "git_dirty": false,
  "morph_rendered_beauty_report": "morph-rendered-beauty.json",
  "morph_to_pixels_report": "morph-to-pixels.json",
  "target_matrix": [
    {"target": "headless", "status": "validated", "renderer_owned_stable_proof": true, "product_claim": true},
    {"target": "linux-x64-real-window", "status": "validated", "renderer_owned_stable_proof": true, "product_claim": true},
    {"target": "wasm32-web-browser-canvas", "status": "validated", "renderer_owned_stable_proof": true, "product_claim": true}
  ],
  "stable_promotion_blockers": [],
  "renderer_owned_stable_targets": ["headless", "linux-x64-real-window", "wasm32-web-browser-canvas"],
  "bridge_owned_stable_targets": [],
  "product_claim": true,
  "final_signoff": true,
  "pass": true
}
`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeProductSliceHashManifest(t *testing.T, dir string) {
	t.Helper()
	type artifact struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
		Size   int64  `json:"size_bytes"`
	}
	var artifacts []artifact
	for _, path := range productSliceRequiredArtifacts() {
		if path == "artifact-hashes.json" {
			continue
		}
		full := filepath.Join(dir, filepath.FromSlash(path))
		raw, err := os.ReadFile(full)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		artifacts = append(artifacts, artifact{
			Path:   path,
			SHA256: "sha256:" + hex.EncodeToString(sum[:]),
			Size:   int64(len(raw)),
		})
	}
	manifest := map[string]any{
		"schema":    productSliceHashSchema,
		"artifacts": artifacts,
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "artifact-hashes.json"),
		append(raw, '\n'),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
}

func boolPtrProductSlice(value bool) *bool {
	return &value
}
