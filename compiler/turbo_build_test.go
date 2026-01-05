package compiler

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func TestBuildDeterministicAcrossJobs(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"engine/core.tetra": "module engine.core\nfun inc(x: i32): i32 {\n  return x + 1\n}\n",
		"engine/math.tetra": "module engine.math\nimport engine.core as core\nfun add(a: i32, b: i32): i32 {\n  return core.inc(a) + b\n}\n",
		"app/game.tetra":    "module app.game\nimport engine.math as m\nfun main(): i32 {\n  return m.add(20, 22)\n}\n",
	}

	tmp1 := t.TempDir()
	writeTestFiles(t, tmp1, files)
	entry1 := filepath.Join(tmp1, filepath.FromSlash("app/game.tetra"))
	out1 := filepath.Join(tmp1, "out", "app1")
	if err := os.MkdirAll(filepath.Dir(out1), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, err := BuildFileWithStatsOpt(entry1, out1, "linux-x64", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("build jobs=1: %v", err)
	}
	if err := verifyELF(out1); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	hash1 := sha256File(t, out1)

	tmp2 := t.TempDir()
	writeTestFiles(t, tmp2, files)
	entry2 := filepath.Join(tmp2, filepath.FromSlash("app/game.tetra"))
	out2 := filepath.Join(tmp2, "out", "app2")
	if err := os.MkdirAll(filepath.Dir(out2), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, err := BuildFileWithStatsOpt(entry2, out2, "linux-x64", BuildOptions{Jobs: 8}); err != nil {
		t.Fatalf("build jobs=8: %v", err)
	}
	if err := verifyELF(out2); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	hash2 := sha256File(t, out2)

	if hash1 != hash2 {
		t.Fatalf("ELF hash mismatch: jobs=1 %x jobs=8 %x", hash1, hash2)
	}
}

func TestConcurrentBuildsSameCache(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"engine/core.tetra": "module engine.core\nfun inc(x: i32): i32 {\n  return x + 1\n}\n",
		"engine/math.tetra": "module engine.math\nimport engine.core as core\nfun add(a: i32, b: i32): i32 {\n  return core.inc(a) + b\n}\n",
		"app/game.tetra":    "module app.game\nimport engine.math as m\nfun main(): i32 {\n  return m.add(20, 22)\n}\n",
	}

	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/game.tetra"))
	out1 := filepath.Join(tmp, "out", "app1")
	out2 := filepath.Join(tmp, "out", "app2")
	if err := os.MkdirAll(filepath.Dir(out1), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, err := BuildFileWithStatsOpt(entry, out1, "linux-x64", BuildOptions{Jobs: 4})
		errs <- err
	}()
	go func() {
		defer wg.Done()
		_, err := BuildFileWithStatsOpt(entry, out2, "linux-x64", BuildOptions{Jobs: 4})
		errs <- err
	}()
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent build error: %v", err)
		}
	}

	if err := verifyELF(out1); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	if err := verifyELF(out2); err != nil {
		t.Fatalf("verify ELF: %v", err)
	}
	hash1 := sha256File(t, out1)
	hash2 := sha256File(t, out2)
	if hash1 != hash2 {
		t.Fatalf("ELF hash mismatch: %x vs %x", hash1, hash2)
	}
}

func TestBuildCacheHitNoLowering(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	files := map[string]string{
		"engine/render.tetra": "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/game.tetra":      "module app.game\nimport engine.render as r\nfun main(): i32 {\n  return r.add_one(41)\n}\n",
	}
	tmp := t.TempDir()
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/game.tetra"))
	out := filepath.Join(tmp, "out", "app")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	stats1, err := BuildFileWithStatsOpt(entry, out, "linux-x64", BuildOptions{Jobs: 2})
	if err != nil {
		t.Fatalf("build1: %v", err)
	}
	if len(stats1.LoweredModules) == 0 {
		t.Fatalf("expected lowering on first build")
	}

	stats2, err := BuildFileWithStatsOpt(entry, out, "linux-x64", BuildOptions{Jobs: 2})
	if err != nil {
		t.Fatalf("build2: %v", err)
	}
	if len(stats2.LoweredModules) != 0 {
		t.Fatalf("expected no lowering on cache hit")
	}
	if len(stats2.CompiledModules) != 0 {
		t.Fatalf("expected no compiled modules on cache hit")
	}
}

func sha256File(t *testing.T, path string) [32]byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	return sha256.Sum256(data)
}
