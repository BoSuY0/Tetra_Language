package compiler

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestBuildCacheReuse(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"engine/render.tetra": "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/game.tetra":      "module app.game\nimport engine.render as r\nfun main(): i32 {\n  return r.add_one(41)\n}\n",
	}
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/game.tetra"))
	outPath := filepath.Join(tmp, "out", "app")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	stats1, err := BuildFileWithStats(entry, outPath, "linux-x64")
	if err != nil {
		t.Fatalf("build1: %v", err)
	}
	assertModules(t, stats1.CompiledModules, []string{"app.game", "engine.render"})
	if len(stats1.CacheHits) != 0 {
		t.Fatalf("unexpected cache hits on first build")
	}

	stats2, err := BuildFileWithStats(entry, outPath, "linux-x64")
	if err != nil {
		t.Fatalf("build2: %v", err)
	}
	if len(stats2.CompiledModules) != 0 {
		t.Fatalf("expected no compiled modules on cache hit")
	}
	assertModules(t, stats2.CacheHits, []string{"app.game", "engine.render"})

	updated := "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 2\n}\n"
	renderPath := filepath.Join(tmp, filepath.FromSlash("engine/render.tetra"))
	if err := os.WriteFile(renderPath, []byte(updated), 0o644); err != nil {
		t.Fatalf("update module: %v", err)
	}

	stats3, err := BuildFileWithStats(entry, outPath, "linux-x64")
	if err != nil {
		t.Fatalf("build3: %v", err)
	}
	assertModules(t, stats3.CompiledModules, []string{"engine.render"})
	assertModules(t, stats3.CacheHits, []string{"app.game"})
}

func TestBuildCacheUnrelatedSignatureChangeDoesNotRebuild(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"engine/render.tetra": "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"engine/audio.tetra":  "module engine.audio\nfun beep(x: i32): i32 {\n  return x\n}\n",
		"app/game.tetra":      "module app.game\nimport engine.render as r\nimport engine.audio as a\nfun main(): i32 {\n  return r.add_one(41)\n}\n",
	}
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/game.tetra"))
	outPath := filepath.Join(tmp, "out", "app")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	stats1, err := BuildFileWithStats(entry, outPath, "linux-x64")
	if err != nil {
		t.Fatalf("build1: %v", err)
	}
	assertModules(t, stats1.CompiledModules, []string{"app.game", "engine.audio", "engine.render"})
	if len(stats1.CacheHits) != 0 {
		t.Fatalf("unexpected cache hits on first build")
	}

	stats2, err := BuildFileWithStats(entry, outPath, "linux-x64")
	if err != nil {
		t.Fatalf("build2: %v", err)
	}
	if len(stats2.CompiledModules) != 0 {
		t.Fatalf("expected no compiled modules on cache hit")
	}
	assertModules(t, stats2.CacheHits, []string{"app.game", "engine.audio", "engine.render"})

	updatedAudio := "module engine.audio\nfun beep(x: i32, y: i32): i32 {\n  return x\n}\n"
	audioPath := filepath.Join(tmp, filepath.FromSlash("engine/audio.tetra"))
	if err := os.WriteFile(audioPath, []byte(updatedAudio), 0o644); err != nil {
		t.Fatalf("update module: %v", err)
	}

	stats3, err := BuildFileWithStats(entry, outPath, "linux-x64")
	if err != nil {
		t.Fatalf("build3: %v", err)
	}
	assertModules(t, stats3.CompiledModules, []string{"engine.audio"})
	assertModules(t, stats3.CacheHits, []string{"app.game", "engine.render"})
}

func TestBuildCacheAddingUnusedExportDoesNotRebuildConsumer(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"engine/render.tetra": "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/game.tetra":      "module app.game\nimport engine.render as r\nfun main(): i32 {\n  return r.add_one(41)\n}\n",
	}
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/game.tetra"))
	outPath := filepath.Join(tmp, "out", "app")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	stats1, err := BuildFileWithStats(entry, outPath, "linux-x64")
	if err != nil {
		t.Fatalf("build1: %v", err)
	}
	assertModules(t, stats1.CompiledModules, []string{"app.game", "engine.render"})
	if len(stats1.CacheHits) != 0 {
		t.Fatalf("unexpected cache hits on first build")
	}

	updatedRender := "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\nfun add_two(x: i32): i32 {\n  return x + 2\n}\n"
	renderPath := filepath.Join(tmp, filepath.FromSlash("engine/render.tetra"))
	if err := os.WriteFile(renderPath, []byte(updatedRender), 0o644); err != nil {
		t.Fatalf("update module: %v", err)
	}

	stats2, err := BuildFileWithStats(entry, outPath, "linux-x64")
	if err != nil {
		t.Fatalf("build2: %v", err)
	}
	assertModules(t, stats2.CompiledModules, []string{"engine.render"})
	assertModules(t, stats2.CacheHits, []string{"app.game"})
}

func TestBuildCacheTransitiveDoesNotRebuildRoot(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"engine/core.tetra": "module engine.core\nfun inc(x: i32): i32 {\n  return x + 1\n}\n",
		"engine/math.tetra": "module engine.math\nimport engine.core as core\nfun sum2(a: i32, b: i32): i32 {\n  return core.inc(a) + b\n}\n",
		"app/game.tetra":    "module app.game\nimport engine.math as m\nfun main(): i32 {\n  return m.sum2(20, 22)\n}\n",
	}
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/game.tetra"))
	outPath := filepath.Join(tmp, "out", "app")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	stats1, err := BuildFileWithStats(entry, outPath, "linux-x64")
	if err != nil {
		t.Fatalf("build1: %v", err)
	}
	assertModules(t, stats1.CompiledModules, []string{"app.game", "engine.core", "engine.math"})
	if len(stats1.CacheHits) != 0 {
		t.Fatalf("unexpected cache hits on first build")
	}

	stats2, err := BuildFileWithStats(entry, outPath, "linux-x64")
	if err != nil {
		t.Fatalf("build2: %v", err)
	}
	if len(stats2.CompiledModules) != 0 {
		t.Fatalf("expected no compiled modules on cache hit")
	}
	assertModules(t, stats2.CacheHits, []string{"app.game", "engine.core", "engine.math"})

	updatedCore := "module engine.core\nfun inc(x: i32, y: i32): i32 {\n  return x + y\n}\n"
	corePath := filepath.Join(tmp, filepath.FromSlash("engine/core.tetra"))
	if err := os.WriteFile(corePath, []byte(updatedCore), 0o644); err != nil {
		t.Fatalf("update module: %v", err)
	}
	updatedMath := "module engine.math\nimport engine.core as core\nfun sum2(a: i32, b: i32): i32 {\n  return core.inc(a, 1) + b\n}\n"
	mathPath := filepath.Join(tmp, filepath.FromSlash("engine/math.tetra"))
	if err := os.WriteFile(mathPath, []byte(updatedMath), 0o644); err != nil {
		t.Fatalf("update module: %v", err)
	}

	stats3, err := BuildFileWithStats(entry, outPath, "linux-x64")
	if err != nil {
		t.Fatalf("build3: %v", err)
	}
	assertModules(t, stats3.CompiledModules, []string{"engine.core", "engine.math"})
	assertModules(t, stats3.CacheHits, []string{"app.game"})
}

func TestBuildCacheStructDependencyChangeRebuildsConsumer(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"engine/math.tetra": "module engine.math\nstruct Vec2 { x: i32, y: i32 }\nfun sum(v: Vec2): i32 {\n  return v.x + v.y\n}\n",
		"app/game.tetra":    "module app.game\nimport engine.math as m\nfun accept(v: m.Vec2): i32 {\n  return 1\n}\nfun main(): i32 {\n  return 0\n}\n",
	}
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/game.tetra"))
	outPath := filepath.Join(tmp, "out", "app")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	stats1, err := BuildFileWithStats(entry, outPath, "linux-x64")
	if err != nil {
		t.Fatalf("build1: %v", err)
	}
	assertModules(t, stats1.CompiledModules, []string{"app.game", "engine.math"})
	if len(stats1.CacheHits) != 0 {
		t.Fatalf("unexpected cache hits on first build")
	}

	stats2, err := BuildFileWithStats(entry, outPath, "linux-x64")
	if err != nil {
		t.Fatalf("build2: %v", err)
	}
	if len(stats2.CompiledModules) != 0 {
		t.Fatalf("expected no compiled modules on cache hit")
	}
	assertModules(t, stats2.CacheHits, []string{"app.game", "engine.math"})

	updatedMath := "module engine.math\nstruct Vec2 { x: i32, y: i32, z: i32 }\nfun sum(v: Vec2): i32 {\n  return v.x + v.y + v.z\n}\n"
	mathPath := filepath.Join(tmp, filepath.FromSlash("engine/math.tetra"))
	if err := os.WriteFile(mathPath, []byte(updatedMath), 0o644); err != nil {
		t.Fatalf("update module: %v", err)
	}

	stats3, err := BuildFileWithStats(entry, outPath, "linux-x64")
	if err != nil {
		t.Fatalf("build3: %v", err)
	}
	assertModules(t, stats3.CompiledModules, []string{"app.game", "engine.math"})
	if len(stats3.CacheHits) != 0 {
		t.Fatalf("unexpected cache hits after struct change")
	}
}

func TestBuildCacheSliceModuleChangeRebuildsOnlyProducer(t *testing.T) {
	tmp := t.TempDir()
	files := map[string]string{
		"engine/arr.tetra": "module engine.arr\nfun sum3(): i32 {\n  var xs: []i32 = make_i32(3)\n  xs[0] = 1\n  xs[1] = 2\n  xs[2] = xs[0] + xs[1]\n  return xs[2]\n}\n",
		"app/game.tetra":   "module app.game\nimport engine.arr as a\nfun main(): i32 {\n  return a.sum3()\n}\n",
	}
	writeTestFiles(t, tmp, files)
	entry := filepath.Join(tmp, filepath.FromSlash("app/game.tetra"))
	outPath := filepath.Join(tmp, "out", "app")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	stats1, err := BuildFileWithStats(entry, outPath, "linux-x64")
	if err != nil {
		t.Fatalf("build1: %v", err)
	}
	assertModules(t, stats1.CompiledModules, []string{"app.game", "engine.arr"})

	stats2, err := BuildFileWithStats(entry, outPath, "linux-x64")
	if err != nil {
		t.Fatalf("build2: %v", err)
	}
	if len(stats2.CompiledModules) != 0 {
		t.Fatalf("expected no compiled modules on cache hit")
	}
	assertModules(t, stats2.CacheHits, []string{"app.game", "engine.arr"})

	updated := "module engine.arr\nfun sum3(): i32 {\n  var xs: []i32 = make_i32(2)\n  xs[0] = 3\n  xs[1] = 4\n  return xs[0] + xs[1]\n}\n"
	arrPath := filepath.Join(tmp, filepath.FromSlash("engine/arr.tetra"))
	if err := os.WriteFile(arrPath, []byte(updated), 0o644); err != nil {
		t.Fatalf("update module: %v", err)
	}

	stats3, err := BuildFileWithStats(entry, outPath, "linux-x64")
	if err != nil {
		t.Fatalf("build3: %v", err)
	}
	assertModules(t, stats3.CompiledModules, []string{"engine.arr"})
	assertModules(t, stats3.CacheHits, []string{"app.game"})
}

func assertModules(t *testing.T, got []string, want []string) {
	t.Helper()
	gotSorted := append([]string(nil), got...)
	wantSorted := append([]string(nil), want...)
	sort.Strings(gotSorted)
	sort.Strings(wantSorted)
	if len(gotSorted) != len(wantSorted) {
		t.Fatalf("module list mismatch: got %v want %v", gotSorted, wantSorted)
	}
	for i := range gotSorted {
		if gotSorted[i] != wantSorted[i] {
			t.Fatalf("module list mismatch: got %v want %v", gotSorted, wantSorted)
		}
	}
}
