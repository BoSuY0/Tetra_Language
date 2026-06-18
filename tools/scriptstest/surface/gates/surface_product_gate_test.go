package surface_gates

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSurfaceProductGateRunsScopedProductEvidence(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "surface", "product-gate.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Surface product gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/product-gate.sh [--report-dir DIR]",
		`source "$script_dir/report-dir-guard.sh"`,
		`surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "surface_product_gate:"`,
		`bash scripts/release/surface/release-gate.sh --report-dir "$report_dir_arg"`,
		`go run ./tools/cmd/validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-surface-product-summary --report-dir "$report_dir"`,
		`go run ./tools/cmd/validate-surface-claims --root "$repo_root" --report-dir "$report_dir"`,
		`go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`,
		`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`,
		`summary_path="$report_dir/surface-product-gate-summary.json"`,
		`product_summary_path="$report_dir/product-summary.json"`,
		`"schema": "tetra.surface.product-gate.v1"`,
		`"schema": "tetra.surface.product-summary.v1"`,
		`"release_gate_report": "surface-release-summary.json"`,
		`"artifact_hash_manifest": "artifact-hashes.json"`,
		`"final_verdict_owner": "SURFACE-BEAUTY-P29"`,
		`"final_verdict": $(json_string "$final_verdict")`,
		`"canonical_final_readiness_report": true`,
		`"inner_release_summary_role": "prerequisite_evidence_not_final_signoff"`,
		`"release_gate_report_final_signoff": false`,
		`"visual": "visual/visual-summary.json"`,
		`"accessibility": "accessibility/accessibility-summary.json"`,
		`"performance": "performance/performance-budget.json"`,
		`"app_shell": "app-shell/app-shell-summary.json"`,
		`"package": "package/package-summary.json"`,
		`"reference_apps": "reference-apps/reference-apps-summary.json"`,
		`"claim_governance": "claim-governance/claims-summary.json"`,
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Surface product gate missing %q", want)
		}
	}
	assertOrderedFragments(t, text,
		`surface_release_require_fresh_report_dir "$report_dir_arg"`,
		`release-gate.sh --report-dir "$report_dir_arg"`,
		`validate-artifact-hashes --manifest "$report_dir/artifact-hashes.json"`,
		`validate-surface-claims --root "$repo_root" --report-dir "$report_dir"`,
		`validate-manifest --manifest docs/generated/manifest.json`,
		`verify-docs --manifest docs/generated/manifest.json`,
		`cat > "$summary_path" <<JSON`,
		`cat > "$product_summary_path" <<JSON`,
		`write_product_category_summary \`,
		`validate-artifact-hashes --write --root "$report_dir"`,
		`validate-surface-product-summary --report-dir "$report_dir"`,
	)
	for _, forbidden := range []string{
		"continue-on-error",
		"|| true",
		"set +e",
		"GOCACHE=/tmp",
		"GOTMPDIR=/tmp",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf(
				"Surface product gate must not contain bypass or tmpfs cache marker %q",
				forbidden,
			)
		}
	}
}

func TestSurfaceProductGateWritesP29RequiredSummaryAliases(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "surface", "product-gate.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Surface product gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`final_verdict="BLOCKED_DIRTY_CHECKOUT"`,
		`product_summary_status="product_gate_passed_clean_same_commit_blocked"`,
		`write_product_category_summary \`,
		`"$report_dir/visual/visual-summary.json"`,
		`"$report_dir/accessibility/accessibility-summary.json"`,
		`"$report_dir/performance/performance-budget.json"`,
		`"$report_dir/app-shell/app-shell-summary.json"`,
		`"$report_dir/package/package-summary.json"`,
		`"$report_dir/reference-apps/reference-apps-summary.json"`,
		`"$report_dir/claim-governance/claims-summary.json"`,
		`"nonclaim-macos-surface-production-support"`,
		`"nonclaim-windows-surface-production-support"`,
		`"wasm32-wasi-surface-ui-runtime"`,
		`"dom-authored-application-ui"`,
		`"user-javascript-application-logic"`,
		`Surface product summary: $product_summary_path`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Surface product gate missing P29 alias detail %q", want)
		}
	}
	assertOrderedFragments(t, text,
		`cat > "$product_summary_path" <<JSON`,
		`write_product_category_summary \`,
		`validate-artifact-hashes --write --root "$report_dir"`,
	)
}

func TestSurfaceProductSliceGateRunsFlagshipEvidence(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "surface", "surface-product-slice-gate.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Surface product-slice gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/surface-product-slice-gate.sh [--report-dir DIR]",
		`surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "surface_product_slice_gate:"`,
		`flagship_source="examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra"`,
		`go run ./tools/cmd/surface-runtime-smoke --mode headless-block-system --source "$flagship_source"`,
		`go run ./tools/cmd/surface-runtime-smoke --mode linux-x64-real-window-block-system --source "$flagship_source"`,
		`go run ./tools/cmd/surface-runtime-smoke --mode wasm32-web-browser-canvas-block-system --source "$flagship_source"`,
		`go run ./cli/cmd/tetra surface dev \`,
		`--change-file "token:lib/core/morph/morph.tetra"`,
		`--change-file "recipe:lib/core/morph/morph.tetra"`,
		`bash scripts/release/surface/surface-package-smoke.sh \`,
		`--app-id studio-shell`,
		`--expected-exit-code 0`,
		`bash scripts/release/surface/morph-rendered-beauty-gate.sh --report-dir "$mrb_gate_dir"`,
		`go run ./tools/cmd/validate-surface-claims --root "$repo_root" --report-dir "$report_dir"`,
		`go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`,
		`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`,
		`"schema": "tetra.surface.product-slice-summary.v1"`,
		`"flagship_source": $(json_string "$flagship_source")`,
		`"morph_rendered_beauty": "validated"`,
		`"morph_rendered_beauty_gate": "morph-rendered-beauty/morph-rendered-beauty-gate-summary.json"`,
		`"category_flagship_runtime": "categories/flagship-runtime.json"`,
		`"category_morph_beauty": "categories/morph-rendered-beauty.json"`,
		`"no-electron-api-compatibility"`,
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-surface-product-slice --report-dir "$report_dir"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Surface product-slice gate missing %q", want)
		}
	}
	assertOrderedFragments(t, text,
		`surface_release_require_fresh_report_dir "$report_dir_arg"`,
		`surface-runtime-smoke --mode headless-block-system`,
		`surface-runtime-smoke --mode linux-x64-real-window-block-system`,
		`surface-runtime-smoke --mode wasm32-web-browser-canvas-block-system`,
		`go run ./cli/cmd/tetra surface dev`,
		`surface-package-smoke.sh`,
		`morph-rendered-beauty-gate.sh --report-dir "$mrb_gate_dir"`,
		`validate-surface-claims --root "$repo_root" --report-dir "$report_dir"`,
		`validate-manifest --manifest docs/generated/manifest.json`,
		`verify-docs --manifest docs/generated/manifest.json`,
		`cat > "$summary_path" <<JSON`,
		`validate-artifact-hashes --write --root "$report_dir"`,
		`validate-surface-product-slice --report-dir "$report_dir"`,
	)
	for _, forbidden := range []string{
		"continue-on-error",
		"|| true",
		"GOCACHE=/tmp",
		"GOTMPDIR=/tmp",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf(
				"Surface product-slice gate must not contain bypass or tmpfs cache marker %q",
				forbidden,
			)
		}
	}
}

func TestSurfaceMorphRenderedBeautyGateRunsIntegratedEvidence(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "surface", "morph-rendered-beauty-gate.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Surface Morph rendered beauty gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/surface/morph-rendered-beauty-gate.sh [--report-dir DIR]",
		`surface_release_require_fresh_report_dir "$report_dir_arg" "$repo_root" "surface_morph_rendered_beauty_gate:"`,
		`source_path="examples/surface/morph_flagship/surface_morph_rendered_studio_shell.tetra"`,
		`go run ./tools/cmd/validate-surface-morph-rendered-beauty \`,
		`--contract docs/spec/surface/morph/surface_morph_rendered_beauty_contract.json`,
		`go run ./tools/cmd/validate-surface-block-contract \`,
		`--contract docs/spec/surface/surface_block_contract.json`,
		`go run ./tools/cmd/surface-runtime-smoke \`,
		`--mode headless-morph`,
		`go run ./tools/cmd/surface-visual-diff \`,
		`--morph-rendered-beauty-report "$mrb_report"`,
		`--morph-to-pixels-chain-out "$morph_to_pixels_report"`,
		`bash scripts/release/surface/surface-dev-workflow-smoke.sh --report-dir "$dev_dir"`,
		`bash scripts/release/surface/surface-inspector-smoke.sh --report-dir "$inspector_dir"`,
		`bash scripts/release/surface/surface-template-smoke.sh --report-dir "$template_dir"`,
		`bash scripts/release/surface/surface-reference-apps-smoke.sh --report-dir "$reference_dir"`,
		`bash scripts/release/surface/surface-docs-claims-gate.sh --report-dir "$report_dir"`,
		`"schema": "tetra.surface.morph-rendered-beauty.gate.v1"`,
		`gate_status="validated_with_target_blockers"`,
		`"status": $(json_string "$gate_status")`,
		`"product_claim": $product_claim`,
		`"final_signoff": $final_signoff`,
		`"target_blockers": []`,
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Surface Morph rendered beauty gate missing %q", want)
		}
	}
	assertOrderedFragments(t, text,
		`validate-surface-morph-rendered-beauty \`,
		`validate-surface-block-contract \`,
		`surface-runtime-smoke \`,
		`surface-visual-diff \`,
		`--morph-rendered-beauty-report "$mrb_report"`,
		`surface-dev-workflow-smoke.sh`,
		`surface-inspector-smoke.sh`,
		`surface-template-smoke.sh`,
		`surface-reference-apps-smoke.sh`,
		`surface-docs-claims-gate.sh`,
		`cat > "$summary_path" <<JSON`,
		`validate-artifact-hashes --write --root "$report_dir"`,
	)
	for _, forbidden := range []string{
		"continue-on-error",
		"|| true",
		"set +e",
		"GOCACHE=/tmp",
		"GOTMPDIR=/tmp",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf(
				"Surface Morph rendered beauty gate must not contain bypass or tmpfs cache marker %q",
				forbidden,
			)
		}
	}
}

func TestSurfaceProductGateRejectsStaleReportDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash release script test")
	}

	root := surfaceProductGateFakeRoot(t)
	reportRel := filepath.ToSlash(filepath.Join("reports", "surface-product"))
	reportPath := filepath.Join(root, filepath.FromSlash(reportRel))
	if err := os.MkdirAll(reportPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(reportPath, "stale.json"),
		[]byte("{}\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	out, err := runSurfaceProductGate(t, root, "--report-dir", reportRel)
	if err == nil {
		t.Fatalf("expected stale report-dir rejection\n%s", out)
	}
	for _, want := range []string{
		"surface_product_gate: refusing to reuse non-empty report directory: " + reportRel,
		"surface_product_gate: choose a fresh --report-dir so stale reports cannot be reused",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("stale product report-dir output missing %q:\n%s", want, out)
		}
	}
	for _, forbidden := range []string{
		"release-gate.sh",
		"go run",
		"mkdir:",
		"find:",
		"No such file or directory",
	} {
		if strings.Contains(string(out), forbidden) {
			t.Fatalf(
				"product gate should reject report dir before sub-gates or raw shell side effects:\n%s",
				out,
			)
		}
	}
}

func surfaceProductGateFakeRoot(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	srcDir := filepath.Join(repoRoot(t), "scripts", "release", "surface")
	dstDir := filepath.Join(root, "scripts", "release", "surface")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"product-gate.sh", "report-dir-guard.sh"} {
		if err := copyFile(filepath.Join(srcDir, name), filepath.Join(dstDir, name), 0o755); err != nil {
			t.Fatalf("copy %s: %v", name, err)
		}
	}
	return root
}

func runSurfaceProductGate(t *testing.T, root string, args ...string) ([]byte, error) {
	t.Helper()

	cmdArgs := append([]string{"scripts/release/surface/product-gate.sh"}, args...)
	cmd := exec.Command("bash", cmdArgs...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GOTELEMETRY=off",
		"GOCACHE="+filepath.Join(root, ".cache", "go-build-surface-product-gate-scriptstest"),
		"GOTMPDIR="+filepath.Join(root, ".cache", "go-tmp-surface-product-gate-scriptstest"),
	)
	return cmd.CombinedOutput()
}
