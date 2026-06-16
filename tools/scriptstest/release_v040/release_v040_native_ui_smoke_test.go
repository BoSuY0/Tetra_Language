package release_v040

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseV040NativeUISmokeScriptRunsExecutableValidator(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "scripts", "release", "v0_4_0", "native-ui-linux-x64-smoke.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read native UI smoke script: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"Usage: bash scripts/release/v0_4_0/native-ui-linux-x64-smoke.sh [--report-dir DIR]",
		`native-ui-linux-x64.json`,
		`go run ./tools/cmd/native-ui-runtime-smoke`,
		`go run ./tools/cmd/validate-native-ui-runtime`,
		`tetra.ui.native-runtime.v1`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("native UI smoke script missing %q", want)
		}
	}
}
