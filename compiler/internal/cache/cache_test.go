package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"tetra_language/compiler/internal/version"
)

func TestCacheKeyIncludesCompilerVersion(t *testing.T) {
	srcHash := sha256.Sum256([]byte("source"))
	depHash := sha256.Sum256([]byte("deps"))

	got := cacheKey("app.game", "linux-x64", "release-opt", srcHash, depHash)

	want := expectedCacheKey("app.game", "linux-x64", "release-opt", srcHash, depHash)
	if got != want {
		t.Fatalf("cacheKey mismatch: got %s want %s", got, want)
	}
}

func TestCacheKeyIncludesTargetAndBuildTag(t *testing.T) {
	srcHash := sha256.Sum256([]byte("source"))
	depHash := sha256.Sum256([]byte("deps"))

	base := cacheKey("app.game", "linux-x64", "release-opt", srcHash, depHash)
	if got := cacheKey("app.game", "windows-x64", "release-opt", srcHash, depHash); got == base {
		t.Fatalf("cache key did not change when target changed: %s", got)
	}
	if got := cacheKey("app.game", "linux-x64", "debug-info", srcHash, depHash); got == base {
		t.Fatalf("cache key did not change when build tag changed: %s", got)
	}
}

func TestLoadCachedObjectTreatsCorruptEntryAsMissAndRemovesIt(t *testing.T) {
	root := t.TempDir()
	srcHash := sha256.Sum256([]byte("source"))
	depHash := sha256.Sum256([]byte("deps"))
	path := cachePath(root, "linux-x64", "release-opt", "app.game", srcHash, depHash)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir cache dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("not-a-valid-tobj"), 0o644); err != nil {
		t.Fatalf("write corrupt cache entry: %v", err)
	}

	obj, hit, err := LoadCachedObject(root, "linux-x64", "release-opt", "app.game", srcHash, depHash)
	if err != nil {
		t.Fatalf("load corrupt cache entry: %v", err)
	}
	if hit || obj != nil {
		t.Fatalf("corrupt cache entry should be a miss, hit=%v obj=%#v", hit, obj)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("corrupt cache entry should be removed, stat err=%v", err)
	}
}

func expectedCacheKey(module, target, buildTag string, srcHash, depHash [32]byte) string {
	h := sha256.New()
	h.Write([]byte(module))
	h.Write([]byte{0})
	h.Write([]byte(target))
	h.Write([]byte{0})
	h.Write([]byte(buildTag))
	h.Write([]byte{0})
	h.Write([]byte(version.CompilerVersion))
	h.Write([]byte{0})
	h.Write(srcHash[:])
	h.Write(depHash[:])
	return hex.EncodeToString(h.Sum(nil))
}
