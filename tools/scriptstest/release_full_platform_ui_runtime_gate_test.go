package scriptstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseFullPlatformUIRuntimeGateRunsMandatoryEvidence(t *testing.T) {
	path := filepath.Join(repoRoot(t), "scripts", "release", "full_platform", "ui-runtime-gate.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read full-platform UI gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/full_platform/ui-runtime-gate.sh [--report-dir DIR]",
		"prepare_report_dir",
		"go test ./compiler/... ./cli/... ./tools/... -count=1",
		"go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json",
		"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
		"go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json",
		"go run ./tools/cmd/validate-targets",
		`bash scripts/release/v0_4_0/native-ui-linux-x64-smoke.sh --report-dir "$report_dir"`,
		`go run ./tools/cmd/validate-native-ui-runtime --report "$report_dir/native-ui-linux-x64.json"`,
		`bash scripts/release/post_v0_4/ui-production-runtime-linux-x64-smoke.sh --report-dir "$report_dir"`,
		`go run ./tools/cmd/validate-ui-production-runtime --report "$report_dir/ui-production-runtime-linux-x64.json"`,
		`bash scripts/release/full_platform/windows-ui-runtime-smoke.sh --report-dir "$report_dir"`,
		`go run ./tools/cmd/validate-windows-ui-runtime --report "$report_dir/windows-ui-runtime.json"`,
		`bash scripts/release/full_platform/macos-ui-runtime-smoke.sh --report-dir "$report_dir"`,
		`go run ./tools/cmd/validate-macos-ui-runtime --report "$report_dir/macos-ui-runtime.json"`,
		`bash scripts/release/v1_0/web-smoke.sh --report "$report_dir/web-smoke.json"`,
		`go run ./tools/cmd/validate-web-ui-smoke --report "$report_dir/web-smoke.json"`,
		"go run ./tools/cmd/validate-cross-platform-ui-runtime",
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
		"tetra.release.full_platform.ui_runtime.production-gate.v1",
		"TETRA_WINDOWS_UI_RUNTIME_REPORT",
		"TETRA_MACOS_UI_RUNTIME_REPORT",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("full-platform UI gate missing %q", want)
		}
	}
}

func TestReleaseFullPlatformSmokeScriptsExist(t *testing.T) {
	for _, rel := range []string{
		"scripts/release/full_platform/README.md",
		"scripts/release/full_platform/windows-ui-runtime-smoke.sh",
		"scripts/release/full_platform/macos-ui-runtime-smoke.sh",
	} {
		info, err := os.Stat(filepath.Join(repoRoot(t), filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("%s must exist: %v", rel, err)
		}
		if info.IsDir() || info.Size() == 0 {
			t.Fatalf("%s must be a non-empty file", rel)
		}
	}
}

func TestReleaseFullPlatformSmokeScriptsAcceptValidatedExternalEvidence(t *testing.T) {
	for _, rel := range []struct {
		path      string
		env       string
		validator string
	}{
		{
			path:      "scripts/release/full_platform/windows-ui-runtime-smoke.sh",
			env:       "TETRA_WINDOWS_UI_RUNTIME_REPORT",
			validator: `go run ./tools/cmd/validate-windows-ui-runtime --report "$report_path"`,
		},
		{
			path:      "scripts/release/full_platform/macos-ui-runtime-smoke.sh",
			env:       "TETRA_MACOS_UI_RUNTIME_REPORT",
			validator: `go run ./tools/cmd/validate-macos-ui-runtime --report "$report_path"`,
		},
	} {
		raw, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(rel.path)))
		if err != nil {
			t.Fatalf("read %s: %v", rel.path, err)
		}
		text := string(raw)
		for _, want := range []string{rel.env, `cp -- "$external_report" "$report_path"`, rel.validator} {
			if !strings.Contains(text, want) {
				t.Fatalf("%s missing %q", rel.path, want)
			}
		}
	}
}
