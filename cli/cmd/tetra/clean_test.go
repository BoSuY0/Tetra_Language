package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanCommandRemovesCacheDirectories(t *testing.T) {
	dir := t.TempDir()
	for _, path := range []string{".tetra_cache", "tetra_cache"} {
		if err := os.MkdirAll(filepath.Join(dir, path, "nested"), 0o755); err != nil {
			t.Fatalf("mkdir cache dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, path, "nested", "entry"), []byte("cache"), 0o644); err != nil {
			t.Fatalf("write cache entry: %v", err)
		}
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	var stdout bytes.Buffer
	code := runCLI([]string{"clean"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("clean exit code = %d, stdout=%q", code, stdout.String())
	}
	for _, path := range []string{".tetra_cache", "tetra_cache"} {
		if _, err := os.Stat(filepath.Join(dir, path)); !os.IsNotExist(err) {
			t.Fatalf("cache dir %s still exists or stat failed with non-missing error: %v", path, err)
		}
	}
	if !strings.Contains(stdout.String(), "Cleaned Tetra cache") {
		t.Fatalf("clean stdout = %q", stdout.String())
	}
}

func TestCleanCommandTargetRemovesOnlyRequestedTargetCache(t *testing.T) {
	dir := t.TempDir()
	for _, path := range []string{
		filepath.Join(".tetra_cache", "linux-x64", "entry"),
		filepath.Join(".tetra_cache", "windows-x64", "entry"),
		filepath.Join("tetra_cache", "linux-x64", "entry"),
		filepath.Join("tetra_cache", "windows-x64", "entry"),
	} {
		if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(path)), 0o755); err != nil {
			t.Fatalf("mkdir cache dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, path), []byte("cache"), 0o644); err != nil {
			t.Fatalf("write cache entry: %v", err)
		}
	}
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	var stdout, stderr bytes.Buffer
	code := runCLI([]string{"clean", "--target", "linux-x64"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("clean --target exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	for _, path := range []string{filepath.Join(".tetra_cache", "linux-x64"), filepath.Join("tetra_cache", "linux-x64")} {
		if _, err := os.Stat(filepath.Join(dir, path)); !os.IsNotExist(err) {
			t.Fatalf("target cache dir %s still exists or stat failed with non-missing error: %v", path, err)
		}
	}
	for _, path := range []string{filepath.Join(".tetra_cache", "windows-x64", "entry"), filepath.Join("tetra_cache", "windows-x64", "entry")} {
		if _, err := os.Stat(filepath.Join(dir, path)); err != nil {
			t.Fatalf("non-target cache entry %s should remain: %v", path, err)
		}
	}
	if !strings.Contains(stdout.String(), "linux-x64") {
		t.Fatalf("clean stdout should name target: %q", stdout.String())
	}
}
