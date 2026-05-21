package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
