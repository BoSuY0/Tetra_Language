package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSurfaceClaimsRejectsFullElectronReplacement(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(t, root, "docs/spec/current_supported_surface.md", `# Fake Surface Claim

Surface is a full Electron replacement for production desktop applications.
`)

	err := validateSurfaceClaims(surfaceClaimOptions{Root: root})
	if err == nil {
		t.Fatalf("validateSurfaceClaims accepted a full Electron replacement claim")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "electron") {
		t.Fatalf("error = %v, want Electron diagnostic", err)
	}
}

func TestValidateSurfaceClaimsRejectsReactAndCSSReplacement(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(t, root, "docs/user/surface_guide.md", `# Fake Surface Claim

Surface is a React replacement and CSS replacement for production app UI.
`)

	err := validateSurfaceClaims(surfaceClaimOptions{Root: root})
	if err == nil {
		t.Fatalf("validateSurfaceClaims accepted React/CSS replacement claims")
	}
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "react") || !strings.Contains(lower, "css") {
		t.Fatalf("error = %v, want React and CSS diagnostics", err)
	}
}

func TestValidateSurfaceClaimsRejectsProductionMorphExperimentalReport(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(t, root, "reports/surface-morph/headless/surface-headless-morph.json", `{
  "schema": "tetra.surface.runtime.v1",
  "target": "headless",
  "claim": "production Morph is ready",
  "experimental": true
}
`)

	err := validateSurfaceClaims(surfaceClaimOptions{
		Root:       root,
		ReportDirs: []string{filepath.Join(root, "reports")},
	})
	if err == nil {
		t.Fatalf("validateSurfaceClaims accepted a production Morph claim with experimental=true")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "morph") {
		t.Fatalf("error = %v, want Morph diagnostic", err)
	}
}

func TestValidateSurfaceClaimsAllowsProductionWordInMorphArtifactPaths(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(t, root, "reports/surface-electron-react-beauty-production/P07/morph-gate/headless/surface-headless-morph.json", `{
  "schema": "tetra.surface.runtime.v1",
  "path": "/repo/reports/surface-electron-react-beauty-production/P07/morph-gate/headless/surface-morph-command-palette",
  "root": "/repo/reports/surface-electron-react-beauty-production/P07/morph-gate/headless/surface-headless-morph-artifacts",
  "command_line": "bash scripts/release/surface/morph-gate.sh --report-dir reports/surface-electron-react-beauty-production/P07/morph-gate",
  "morph": {
    "experimental": true,
    "production_claim": false
  }
}
`)

	err := validateSurfaceClaims(surfaceClaimOptions{
		Root:       root,
		ReportDirs: []string{filepath.Join(root, "reports")},
	})
	if err != nil {
		t.Fatalf("validateSurfaceClaims rejected artifact paths as Morph production claims: %v", err)
	}
}

func TestValidateSurfaceClaimsRejectsUnsupportedGPUProductionClaim(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(t, root, "docs/release/surface_v1_release_notes.md", `# Fake Surface Claim

Surface GPU rendering is production supported for the current release.
`)

	err := validateSurfaceClaims(surfaceClaimOptions{Root: root})
	if err == nil {
		t.Fatalf("validateSurfaceClaims accepted unsupported GPU production claim")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "gpu") {
		t.Fatalf("error = %v, want GPU diagnostic", err)
	}
}

func TestValidateSurfaceClaimsRejectsMixedGPUProductionWithoutEvidenceClause(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(t, root, "docs/release/surface_v1_release_notes.md", `# Fake Surface Claim

Surface GPU rendering is production supported without additional target-host evidence.
`)

	err := validateSurfaceClaims(surfaceClaimOptions{Root: root})
	if err == nil {
		t.Fatalf("validateSurfaceClaims accepted mixed GPU production claim with without-evidence wording")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "gpu") {
		t.Fatalf("error = %v, want GPU diagnostic", err)
	}
}

func TestValidateSurfaceClaimsRejectsStaleProductionEvidence(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(t, root, "reports/surface-release-v1/stale-summary.json", `{
  "schema": "tetra.surface.release.v1",
  "release_scope": "surface-v1-linux-web",
  "production_claim": true,
  "same_commit_validated": false
}
`)

	err := validateSurfaceClaims(surfaceClaimOptions{
		Root:       root,
		ReportDirs: []string{filepath.Join(root, "reports")},
	})
	if err == nil {
		t.Fatalf("validateSurfaceClaims accepted stale production evidence")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "stale") {
		t.Fatalf("error = %v, want stale evidence diagnostic", err)
	}
}

func TestValidateSurfaceClaimsRejectsDocsOnlyProductionClaim(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(t, root, "docs/spec/surface_v1.md", `# Fake Surface Claim

Surface production support is proven by docs-only evidence.
`)

	err := validateSurfaceClaims(surfaceClaimOptions{Root: root})
	if err == nil {
		t.Fatalf("validateSurfaceClaims accepted docs-only production claim")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "docs-only") {
		t.Fatalf("error = %v, want docs-only diagnostic", err)
	}
}

func TestValidateSurfaceClaimsRejectsWindowsMacOSProductionWithoutTargetHostEvidence(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(t, root, "docs/release/surface_v1_release_notes.md", `# Fake Surface Claim

Windows Surface and macOS Surface are production supported real-window targets.
`)

	err := validateSurfaceClaims(surfaceClaimOptions{Root: root})
	if err == nil {
		t.Fatalf("validateSurfaceClaims accepted Windows/macOS production support without target-host evidence")
	}
	lower := strings.ToLower(err.Error())
	if !strings.Contains(lower, "windows") || !strings.Contains(lower, "macos") {
		t.Fatalf("error = %v, want Windows and macOS diagnostics", err)
	}
}

func TestValidateSurfaceClaimsAllowsScopedNonClaims(t *testing.T) {
	root := t.TempDir()
	writeSurfaceClaimFixture(t, root, "README.md", `# Honest Surface Scope

Surface v1 is PROD_STABLE_SCOPED for surface-v1-linux-web.
Surface is not an Electron replacement, not a React replacement, and no CSS replacement claim is made.
Morph remains EXPERIMENTAL; no production Morph claim is made.
Windows Surface and macOS Surface are UNSUPPORTED until BETA_TARGET_HOST reports exist.
`)

	err := validateSurfaceClaims(surfaceClaimOptions{Root: root})
	if err != nil {
		t.Fatalf("validateSurfaceClaims rejected scoped nonclaims: %v", err)
	}
}

func writeSurfaceClaimFixture(t *testing.T, root string, rel string, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
