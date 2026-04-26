package scriptstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV10GateDocumentsMandatoryTargets(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release_v1_0_gate.sh"))
	if err != nil {
		t.Fatalf("read v1.0 release gate: %v", err)
	}
	text := string(raw)
	for _, target := range []string{"linux-x64", "macos-x64", "windows-x64", "wasm32-wasi", "wasm32-web"} {
		if !strings.Contains(text, "--target "+target) {
			t.Fatalf("v1.0 release gate missing target %s", target)
		}
	}
}

func TestReleaseV10GateKeepsCurrentValidators(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "release_v1_0_gate.sh"))
	if err != nil {
		t.Fatalf("read v1.0 release gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"bash scripts/test_all.sh --full",
		"go run ./tools/cmd/validate-flow-only",
		"./tetra targets --format=json",
		"go run ./tools/cmd/validate-targets",
		"./tetra doctor --format=json",
		"go run ./tools/cmd/validate-doctor",
		"./tetra check examples/flow_hello.tetra",
		"./tetra doc examples",
		"./t version",
		"go run ./tools/cmd/validate-test-report",
		"go run ./tools/cmd/validate-manifest",
		"go run ./tools/cmd/verify-docs",
		"go run ./tools/cmd/smoke-report-to-checklist --validate-only",
		"./tetra smoke --list --format=json",
		"go run ./tools/cmd/validate-smoke-list",
		"go run ./tools/cmd/validate-api-docs",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("v1.0 release gate missing %q", want)
		}
	}
}

func TestRoadmapV10RecordsExplicitCompatibilityAndSafetyPolicy(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "docs", "roadmap_0_6_to_1_0.md"))
	if err != nil {
		t.Fatalf("read v1.0 roadmap: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Flow syntax is the only official 1.0 syntax",
		"`wasm32-wasi`",
		"`wasm32-web`",
		"no data races",
		"Network EcoNet/TetraHub",
		"explicitly labeled beta surface",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("v1.0 roadmap missing %q", want)
		}
	}
}

func TestBootstrapBuildsTetraAndTAlias(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "bootstrap.sh"))
	if err != nil {
		t.Fatalf("read bootstrap: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		`go build -o "./tetra${exe}" ./cli/cmd/tetra`,
		`cp "./tetra${exe}" "./t${exe}"`,
		`Built: ./tetra${exe} ./t${exe}`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("bootstrap missing %q", want)
		}
	}
}
