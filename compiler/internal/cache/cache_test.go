package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"tetra_language/compiler/internal/version"
)

func TestCacheKeyIncludesCompilerVersion(t *testing.T) {
	srcHash := sha256.Sum256([]byte("source"))
	depHash := sha256.Sum256([]byte("deps"))

	got := cacheKey("app.game", "linux-x64", "release-opt", srcHash, depHash)

	h := sha256.New()
	h.Write([]byte("app.game"))
	h.Write([]byte{0})
	h.Write([]byte("linux-x64"))
	h.Write([]byte{0})
	h.Write([]byte("release-opt"))
	h.Write([]byte{0})
	h.Write([]byte(version.CompilerVersion))
	h.Write([]byte{0})
	h.Write(srcHash[:])
	h.Write(depHash[:])
	want := hex.EncodeToString(h.Sum(nil))
	if got != want {
		t.Fatalf("cacheKey mismatch: got %s want %s", got, want)
	}
}
