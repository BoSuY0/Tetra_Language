package release_v040

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBootstrapBuildsTetraAndTAlias(t *testing.T) {
	assertLegacyFileRemoved(t, "scripts/bootstrap.sh", "scripts/dev/bootstrap.sh")
	devRaw, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", "dev", "bootstrap.sh"))
	if err != nil {
		t.Fatalf("read dev bootstrap: %v", err)
	}
	devText := string(devRaw)
	assertNoLegacyMention(t, devText, "scripts/bootstrap.sh", "scripts/dev/bootstrap.sh help")
	for _, want := range []string{
		`release_artifact="tetra.release.v0_4_0.bootstrap-binaries.v1"`,
		`go build -o "./tetra${exe}" ./cli/cmd/tetra`,
		`cp "./tetra${exe}" "./t${exe}"`,
		`Built: ./tetra${exe} ./t${exe}`,
	} {
		if !strings.Contains(devText, want) {
			t.Fatalf("dev bootstrap missing %q", want)
		}
	}
}
