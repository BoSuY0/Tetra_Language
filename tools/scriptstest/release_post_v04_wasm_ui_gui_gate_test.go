package scriptstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleasePostV04WASMUIGUIGateRunsProductionEvidence(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "post_v0_4", "wasm-ui-gui-production-gate.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read post-v0.4 WASM/UI/GUI gate: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/post_v0_4/wasm-ui-gui-production-gate.sh [--report-dir DIR]",
		`prepare_report_dir`,
		`find "$find_report_dir" -mindepth 1 -maxdepth 1 -exec rm -rf -- {} +`,
		`./tetra smoke --target wasm32-wasi --run=false --report "$report_dir/wasi-artifact.json"`,
		`go run ./tools/cmd/validate-wasm-imports --target wasm32-wasi --report "$report_dir/wasi-artifact.json"`,
		`./tetra smoke --target wasm32-wasi --run=true --report "$report_dir/wasi-runtime.json"`,
		`./tetra smoke --target wasm32-web --run=false --report "$report_dir/web-artifact.json"`,
		`go run ./tools/cmd/validate-wasm-imports --target wasm32-web --report "$report_dir/web-artifact.json"`,
		`./tetra smoke --target wasm32-web --run=true --report "$report_dir/web-runtime.json"`,
		`bash scripts/release/v1_0/wasi-smoke.sh --report "$report_dir/wasi-smoke.json"`,
		`bash scripts/release/v1_0/web-smoke.sh --report "$report_dir/web-smoke.json"`,
		`go run ./tools/cmd/validate-web-ui-smoke --report "$report_dir/web-smoke.json"`,
		`bash scripts/release/v0_4_0/native-ui-linux-x64-smoke.sh --report-dir "$report_dir"`,
		`go run ./tools/cmd/validate-native-ui-runtime --report "$report_dir/native-ui-linux-x64.json"`,
		`bash scripts/release/post_v0_4/ui-production-runtime-linux-x64-smoke.sh --report-dir "$report_dir"`,
		`go run ./tools/cmd/validate-ui-production-runtime --report "$report_dir/ui-production-runtime-linux-x64.json"`,
		`go run ./tools/cmd/validate-artifact-hashes --write --root "$report_dir" --out "$report_dir/artifact-hashes.json"`,
		`tetra.release.post_v0_4.wasm_ui_gui.production-gate.v1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("post-v0.4 WASM/UI/GUI gate missing %q", want)
		}
	}
	if strings.Index(text, "wasm32-wasi --run=false") > strings.Index(text, "wasm32-wasi --run=true") {
		t.Fatalf("WASI artifact smoke must run before WASI runtime smoke")
	}
	if strings.Index(text, "wasm32-web --run=false") > strings.Index(text, "wasm32-web --run=true") {
		t.Fatalf("web artifact smoke must run before web runtime smoke")
	}
	if strings.Index(text, "web-smoke.sh") > strings.Index(text, "validate-web-ui-smoke") {
		t.Fatalf("web UI smoke must run before web UI validator")
	}
}

func TestReleasePostV04READMEAdvertisesWASMUIGUIGate(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "post_v0_4", "README.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read post-v0.4 README: %v", err)
	}
	for _, want := range []string{
		"wasm-ui-gui-production-gate.sh",
		"tetra.release.post_v0_4.wasm_ui_gui.production-gate.v1",
	} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("post-v0.4 README missing %q", want)
		}
	}
}

func TestReleaseV0400GateDelegatesToPostV04WASMUIGUIGate(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "v0.4.0_0", "gate.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read v0.4.0_0 gate wrapper: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"post_v0_4/wasm-ui-gui-production-gate.sh",
		`exec bash`,
		`"$@"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("v0.4.0_0 gate wrapper missing %q", want)
		}
	}
}
