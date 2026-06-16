package postv04

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleasePostV04UIToolkitCoreGateRunsProductionEvidence(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "post_v0_4", "ui-toolkit-core-production-gate.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read UI Toolkit Core gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/post_v0_4/ui-toolkit-core-production-gate.sh [--report-dir DIR]",
		`prepare_report_dir`,
		`find "$find_report_dir" -mindepth 1 -maxdepth 1 -exec rm -rf -- {} +`,
		`go test ./compiler/... ./cli/... ./tools/... -count=1`,
		`go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json`,
		`go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`,
		`go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`,
		`go run ./tools/cmd/validate-targets`,
		`./tetra features --format=json > "$report_dir/features.json"`,
		`./tetra targets --format=json > "$report_dir/targets.json"`,
		`go run ./tools/cmd/ui-toolkit-core-smoke --report "$report_dir/ui-toolkit-core.json"`,
		`go run ./tools/cmd/validate-ui-toolkit-core --report "$report_dir/ui-toolkit-core.json"`,
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
		`go run ./tools/cmd/validate-artifact-hashes --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
		`tetra.release.post_v0_4.ui_toolkit_core.production-gate.v1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("UI Toolkit Core gate missing %q", want)
		}
	}
	if strings.Index(text, "ui-toolkit-core-smoke") > strings.Index(text, "validate-ui-toolkit-core") {
		t.Fatalf("toolkit smoke must run before toolkit validator")
	}
}

func TestReleasePostV04READMEAdvertisesUIToolkitCoreGate(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "post_v0_4", "README.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read post-v0.4 README: %v", err)
	}
	for _, want := range []string{
		"ui-toolkit-core-production-gate.sh",
		"tetra.release.post_v0_4.ui_toolkit_core.production-gate.v1",
	} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("post-v0.4 README missing %q", want)
		}
	}
}
