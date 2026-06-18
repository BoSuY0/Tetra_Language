package shell

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestShellScriptsUsePortableBashSafetyHeader(t *testing.T) {
	root := repoRoot(t)
	var entries []string
	err := filepath.WalkDir(
		filepath.Join(root, "scripts"),
		func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.Type().IsRegular() && strings.HasSuffix(entry.Name(), ".sh") {
				entries = append(entries, path)
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("walk scripts: %v", err)
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
			if strings.Contains(text, "mktemp -d") &&
				!strings.Contains(text, `trap 'rm -rf "$tmp_dir"' EXIT`) &&
				!strings.Contains(text, `rm -rf "$tmp_dir"`) {
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

func TestShellScriptsDoNotDefaultTetraArtifactsToTmpfs(t *testing.T) {
	root := repoRoot(t)
	bannedPrefixes := []string{
		"/tmp/" + "tetra",
		"/tmp/" + "release-",
	}
	var entries []string
	err := filepath.WalkDir(
		filepath.Join(root, "scripts"),
		func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.Type().IsRegular() && strings.HasSuffix(entry.Name(), ".sh") {
				entries = append(entries, path)
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("walk scripts: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no shell scripts found")
	}
	for _, path := range entries {
		t.Run(
			filepath.ToSlash(strings.TrimPrefix(path, root+string(os.PathSeparator))),
			func(t *testing.T) {
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read script: %v", err)
				}
				text := string(raw)
				for _, bannedPrefix := range bannedPrefixes {
					if strings.Contains(text, bannedPrefix) {
						t.Fatalf(
							"%s must not default Tetra cache/report artifacts under tmpfs prefix %q",
							path,
							bannedPrefix,
						)
					}
				}
			},
		)
	}
}

func TestDevScriptsHaveNoLegacyRootEntrypoints(t *testing.T) {
	for rel, canonical := range map[string]string{
		"scripts/format.sh": "scripts/dev/format.sh",
		"scripts/dump.sh":   "scripts/dev/dump-project.sh",
	} {
		assertLegacyFileRemoved(t, rel, canonical)
	}
}

func TestCanonicalTestScriptDoesNotMutateFormatting(t *testing.T) {
	root := repoRoot(t)
	ciPath := filepath.Join(root, "scripts", "ci", "test.sh")
	assertLegacyFileRemoved(t, "scripts/test.sh", "scripts/ci/test.sh")
	ciRaw, err := os.ReadFile(ciPath)
	if err != nil {
		t.Fatalf("read ci test script: %v", err)
	}
	text := string(ciRaw)
	assertNoLegacyMention(t, text, "scripts/test.sh", "scripts/ci/test.sh help")
	for _, needle := range []string{"gofmt -w", "gofmt --write"} {
		if strings.Contains(text, needle) {
			t.Fatalf("%s must check formatting without mutating files; found %q", ciPath, needle)
		}
	}
	if !strings.Contains(text, "gofmt -l") {
		t.Fatalf("%s must run gofmt -l as a non-mutating formatting gate", ciPath)
	}
	for _, want := range []string{"--cached", "--others", "--exclude-standard"} {
		if !strings.Contains(text, want) {
			t.Fatalf(
				"%s must include %s so untracked Go files are checked before commit",
				ciPath,
				want,
			)
		}
	}
}

func TestFormattingWorkflowSeparatesCheckAndWrite(t *testing.T) {
	root := repoRoot(t)
	testRaw, err := os.ReadFile(filepath.Join(root, "scripts", "ci", "test.sh"))
	if err != nil {
		t.Fatalf("read ci test script: %v", err)
	}
	devFormatRaw, err := os.ReadFile(filepath.Join(root, "scripts", "dev", "format.sh"))
	if err != nil {
		t.Fatalf("read dev format script: %v", err)
	}
	testText := string(testRaw)
	devFormatText := string(devFormatRaw)
	if !strings.Contains(testText, "gofmt -l") || strings.Contains(testText, "gofmt -w") {
		t.Fatalf("scripts/ci/test.sh must check formatting without writes")
	}
	if !strings.Contains(devFormatText, "gofmt -w") {
		t.Fatalf("scripts/dev/format.sh must be the explicit mutating formatter")
	}
	assertNoLegacyMention(t, devFormatText, "scripts/format.sh", "scripts/dev/format.sh help")
	for _, want := range []string{"--cached", "--others", "--exclude-standard"} {
		if !strings.Contains(devFormatText, want) {
			t.Fatalf(
				"scripts/dev/format.sh must include %s so untracked Go files are formatted",
				want,
			)
		}
	}
}

func TestDumpProjectWorkflowLivesInDevScript(t *testing.T) {
	root := repoRoot(t)
	devPath := filepath.Join(root, "scripts", "dev", "dump-project.sh")
	devRaw, err := os.ReadFile(devPath)
	if err != nil {
		t.Fatalf("read dev dump script: %v", err)
	}
	devText := string(devRaw)
	for _, want := range []string{
		`release_artifact="tetra.release.v0_3_0.project-dump.v1"`,
		`repo_root="$(cd "$script_dir/../.." && pwd)"`,
		`go run ./tools/cmd/dump-project "$@"`,
	} {
		if !strings.Contains(devText, want) {
			t.Fatalf("scripts/dev/dump-project.sh missing %q", want)
		}
	}
	assertNoLegacyMention(t, devText, "scripts/dump.sh", "scripts/dev/dump-project.sh help")

	cmd := exec.Command("bash", devPath, "--help")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("dump help failed: %v\n%s", err, out)
	}
	for _, want := range []string{
		"Usage: bash scripts/dev/dump-project.sh [dump-project flags...]",
		"go run ./tools/cmd/dump-project",
	} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("dump help missing %q:\n%s", want, out)
		}
	}
	assertNoLegacyMention(t, string(out), "scripts/dump.sh", "dump help")
}
