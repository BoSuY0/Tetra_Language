package scriptstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseFullPlatformSmokeScriptsExistAndNameValidators(t *testing.T) {
	root := repoRoot(t)
	for _, rel := range []string{
		"scripts/release/full_platform/windows-ui-runtime-smoke.sh",
		"scripts/release/full_platform/macos-ui-runtime-smoke.sh",
		"scripts/release/full_platform/ui-runtime-gate.sh",
	} {
		raw, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
		text := string(raw)
		for _, want := range []string{
			"reports/full-platform-ui-runtime",
			"validate-artifact-hashes",
		} {
			if !strings.Contains(text, want) {
				t.Fatalf("%s missing %q", rel, want)
			}
		}
	}
}

func TestReleaseFullPlatformGateRunsMandatoryEvidence(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts/release/full_platform/ui-runtime-gate.sh"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{
		"go test ./compiler/... ./cli/... ./tools/... -count=1",
		"go run ./tools/cmd/gen-manifest -o docs/generated/manifest.json",
		"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
		"go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json",
		"go run ./tools/cmd/validate-targets",
		"native-ui-linux-x64-smoke.sh --report-dir",
		"validate-native-ui-runtime --report",
		"ui-production-runtime-linux-x64-smoke.sh --report-dir",
		"validate-ui-production-runtime --report",
		"windows-ui-runtime-smoke.sh --report-dir",
		"validate-windows-ui-runtime --report",
		"macos-ui-runtime-smoke.sh --report-dir",
		"validate-macos-ui-runtime --report",
		"web-smoke.sh --report",
		"validate-web-ui-smoke --report",
		"validate-cross-platform-ui-runtime",
		"validate-artifact-hashes --write",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("ui-runtime-gate.sh missing mandatory step %q", want)
		}
	}
}

func TestReleaseFullPlatformGateRequiresFreshReportDir(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts/release/full_platform/ui-runtime-gate.sh"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{
		"require_fresh_report_dir()",
		"find \"$report_dir\" -mindepth 1 -maxdepth 1",
		"requires a fresh empty report directory",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("ui-runtime-gate.sh must reject reused report dirs; missing %q", want)
		}
	}
}

func TestReleaseFullPlatformGateCollectsPlatformBlockersBeforeFailing(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts/release/full_platform/ui-runtime-gate.sh"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{
		"failures=()",
		"record_failure()",
		"run_required_step",
		`run_required_step "windows UI runtime smoke"`,
		`run_required_step "macOS UI runtime smoke"`,
		`run_required_step "web UI runtime smoke"`,
		"full-platform UI runtime gate failed:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("ui-runtime-gate.sh must collect all platform blockers before exiting; missing %q", want)
		}
	}
	if strings.Contains(text, "\nbash scripts/release/full_platform/windows-ui-runtime-smoke.sh --report-dir \"$report_dir\"\n") {
		t.Fatalf("ui-runtime-gate.sh runs Windows smoke directly under set -e; use run_required_step so macOS/Web evidence still runs")
	}
}

func TestReleaseFullPlatformSmokeScriptsTreatExternalEvidenceAsOnlyPassPath(t *testing.T) {
	root := repoRoot(t)
	for _, rel := range []string{
		"scripts/release/full_platform/windows-ui-runtime-smoke.sh",
		"scripts/release/full_platform/macos-ui-runtime-smoke.sh",
	} {
		raw, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		text := string(raw)
		for _, want := range []string{
			"--evidence",
			"generated_at",
			"target-host-runtime",
			"blocked",
			"cannot collect production UI runtime evidence on this host",
		} {
			if !strings.Contains(text, want) {
				t.Fatalf("%s missing external evidence/blocker contract %q", rel, want)
			}
		}
	}
}

func TestReleaseFullPlatformBlockedReportsUseDetectedHostTriple(t *testing.T) {
	root := repoRoot(t)
	for _, rel := range []string{
		"scripts/release/full_platform/windows-ui-runtime-smoke.sh",
		"scripts/release/full_platform/macos-ui-runtime-smoke.sh",
	} {
		raw, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		text := string(raw)
		for _, want := range []string{
			"host_triple()",
			"host_triple=\"$(host_triple)\"",
			`"host": "${host_triple}"`,
		} {
			if !strings.Contains(text, want) {
				t.Fatalf("%s blocked report must use detected host triple; missing %q", rel, want)
			}
		}
		if strings.Contains(text, `"host": "linux-x64"`) {
			t.Fatalf("%s blocked report hardcodes linux-x64 host", rel)
		}
	}
}
