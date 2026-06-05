package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeArgsAlwaysDumpsWholeProject(t *testing.T) {
	root := t.TempDir()
	dumpDir := filepath.Join(root, "dumps")

	forward, outputPath, err := sanitizeArgs(root, dumpDir, nil)
	if err != nil {
		t.Fatalf("sanitize args: %v", err)
	}

	if !hasArg(forward, "--all") {
		t.Fatalf("forward args = %#v, want automatic --all", forward)
	}
	if !hasArg(forward, "--no-summary") {
		t.Fatalf("forward args = %#v, want internal --no-summary", forward)
	}
	if outputPath == "" || filepath.Dir(outputPath) != dumpDir {
		t.Fatalf("output path = %q, want file inside dumps", outputPath)
	}
}

func TestSanitizeArgsRejectsDumpModeFlags(t *testing.T) {
	root := t.TempDir()
	dumpDir := filepath.Join(root, "dumps")

	for _, args := range [][]string{
		{"--all"},
		{"--only", "compiler"},
		{"--exclude-prefix", "reports"},
		{"--max-file-bytes", "42"},
		{"--no-summary"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			if _, _, err := sanitizeArgs(root, dumpDir, args); err == nil {
				t.Fatalf("sanitizeArgs(%#v) succeeded, want rejection", args)
			}
		})
	}
}

func hasArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

func TestNormalizeOutputPathForcesMarkdownInDumps(t *testing.T) {
	root := t.TempDir()
	dumpDir := filepath.Join(root, "dumps")

	out, err := normalizeOutputPath(root, dumpDir, "project_dump.txt")
	if err != nil {
		t.Fatalf("normalize output path: %v", err)
	}

	want := filepath.Join(dumpDir, "project_dump.md")
	if out != want {
		t.Fatalf("output path = %q, want %q", out, want)
	}
}

func TestSplitDumpFileKeepsChunksUnderLimitAndMarkdown(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "dumps", "project_dump.md")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatalf("mkdir dumps: %v", err)
	}
	data := strings.Repeat("a", 25)
	if err := os.WriteFile(source, []byte(data), 0o644); err != nil {
		t.Fatalf("write source dump: %v", err)
	}

	paths, err := splitDumpFile(source, 10)
	if err != nil {
		t.Fatalf("split dump: %v", err)
	}
	if len(paths) != 3 {
		t.Fatalf("chunks = %d, want 3", len(paths))
	}
	for _, path := range paths {
		if filepath.Ext(path) != ".md" {
			t.Fatalf("chunk path %q does not use .md", path)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat chunk %q: %v", path, err)
		}
		if info.Size() > 10 {
			t.Fatalf("chunk %q size = %d, want <= 10", path, info.Size())
		}
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source dump should be replaced by chunks, stat error = %v", err)
	}
}

func TestRemovePreviousDumpFilesDeletesTopLevelFilesOnly(t *testing.T) {
	root := t.TempDir()
	dumpDir := filepath.Join(root, "dumps")
	if err := os.MkdirAll(filepath.Join(dumpDir, "kept"), 0o755); err != nil {
		t.Fatalf("mkdir dumps: %v", err)
	}
	oldFiles := []string{
		filepath.Join(dumpDir, "project_dump_20260520_225713Z_part_001.md"),
		filepath.Join(dumpDir, "project_dump_20260520_225713Z_part_002.md"),
		filepath.Join(dumpDir, "project_dump_20260520_225713Z.txt"),
	}
	for _, path := range oldFiles {
		if err := os.WriteFile(path, []byte("old dump"), 0o644); err != nil {
			t.Fatalf("write old dump %q: %v", path, err)
		}
	}
	nested := filepath.Join(dumpDir, "kept", "note.md")
	if err := os.WriteFile(nested, []byte("not a top-level dump"), 0o644); err != nil {
		t.Fatalf("write nested file: %v", err)
	}

	removed, err := removePreviousDumpFiles(dumpDir)
	if err != nil {
		t.Fatalf("remove previous dump files: %v", err)
	}
	if removed != len(oldFiles) {
		t.Fatalf("removed = %d, want %d", removed, len(oldFiles))
	}
	for _, path := range oldFiles {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("old dump %q should be removed, stat error = %v", path, err)
		}
	}
	if _, err := os.Stat(nested); err != nil {
		t.Fatalf("nested file should be preserved: %v", err)
	}
}
