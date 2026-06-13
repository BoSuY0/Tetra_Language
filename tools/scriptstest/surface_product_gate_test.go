package scriptstest

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
	for _, forbidden := range []string{"continue-on-error", "|| true", "set +e", "GOCACHE=/tmp", "GOTMPDIR=/tmp"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("Surface product gate must not contain bypass or tmpfs cache marker %q", forbidden)
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
		`"macos-surface-production-nonclaim"`,
		`"windows-surface-production-nonclaim"`,
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
	if err := os.WriteFile(filepath.Join(reportPath, "stale.json"), []byte("{}\n"), 0o644); err != nil {
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
			t.Fatalf("product gate should reject report dir before sub-gates or raw shell side effects:\n%s", out)
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
