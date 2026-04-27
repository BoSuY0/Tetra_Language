package scriptstest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShellScriptsUsePortableBashSafetyHeader(t *testing.T) {
	root := repoRoot(t)
	entries, err := filepath.Glob(filepath.Join(root, "scripts", "*.sh"))
	if err != nil {
		t.Fatalf("glob scripts: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no shell scripts found")
	}
	for _, path := range entries {
		t.Run(filepath.Base(path), func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read script: %v", err)
			}
			text := string(raw)
			lines := strings.Split(text, "\n")
			if len(lines) < 2 || lines[0] != "#!/usr/bin/env bash" {
				t.Fatalf("%s missing portable bash shebang", path)
			}
			if lines[1] != "set -euo pipefail" {
				t.Fatalf("%s missing set -euo pipefail immediately after shebang", path)
			}
			if strings.Contains(text, "mktemp -d") && !strings.Contains(text, `trap 'rm -rf "$tmp_dir"' EXIT`) && !strings.Contains(text, `rm -rf "$tmp_dir"`) {
				t.Fatalf("%s creates a temp dir without quoted cleanup trap", path)
			}
			for _, needle := range []string{
				"gofmt -w $(",
				"rm -rf $",
				"cp ./tetra",
			} {
				if strings.Contains(text, needle) {
					t.Fatalf("%s contains path-space-unsafe shell pattern %q", path, needle)
				}
			}
		})
	}
}
