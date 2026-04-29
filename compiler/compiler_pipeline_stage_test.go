package compiler

import (
	"crypto/sha256"
	"path/filepath"
	"strings"
	"testing"

	ctarget "tetra_language/compiler/target"
)

func TestPipelineResolveNativeTargetStage(t *testing.T) {
	native, handled, stats, err := resolveExecutableBuildTarget("missing.tetra", "out", "linux-x64", BuildOptions{Jobs: 1})
	if err != nil {
		t.Fatalf("resolve native target: %v", err)
	}
	if handled {
		t.Fatalf("linux-x64 executable build should continue through native pipeline")
	}
	if stats != nil {
		t.Fatalf("native resolve should not produce stats before pipeline execution")
	}
	if native.triple != "linux-x64" || native.codegen == nil {
		t.Fatalf("native target = %#v, want linux-x64 with codegen", native)
	}
	if native.target.Format != ctarget.FormatELF {
		t.Fatalf("native target format = %s, want elf", native.target.Format)
	}
	if native.backend.name != "linux-x64" || native.backend.link == nil || native.backend.actorRuntime == nil {
		t.Fatalf("native backend = %#v, want linux-x64 backend with linker and actor runtime", native.backend)
	}

	for _, triple := range []string{"windows-x64", "macos-x64"} {
		tgt, err := ctarget.Parse(triple)
		if err != nil {
			t.Fatalf("parse %s: %v", triple, err)
		}
		backend, ok := nativeExecutableBackendForTarget(tgt)
		if !ok {
			t.Fatalf("missing native backend for %s", triple)
		}
		if backend.link == nil || backend.codegen == nil || backend.actorRuntime == nil {
			t.Fatalf("incomplete backend for %s: %#v", triple, backend)
		}
	}

	_, handled, _, err = resolveExecutableBuildTarget("missing.tetra", "out.wasm", "wasm32-wasi", BuildOptions{Jobs: 1, DebugInfo: true})
	if err == nil || !strings.Contains(err.Error(), "target does not support debug info: wasm32-wasi") {
		t.Fatalf("wasm debug-info rejection error = %v", err)
	}
	if handled {
		t.Fatalf("capability rejection should fail before wasm build dispatch")
	}

	for _, triple := range []string{"wasm32-wasi", "wasm32-web"} {
		_, handled, stats, err = resolveExecutableBuildTarget("missing.tetra", "out.wasm", triple, BuildOptions{Jobs: 1, Emit: EmitObject})
		if err == nil || !strings.Contains(err.Error(), "supports only --emit=exe") {
			t.Fatalf("%s object emit error = %v, want build-only emit rejection", triple, err)
		}
		if !handled {
			t.Fatalf("%s should be handled by build-only WASM pipeline before native emit/object dispatch", triple)
		}
		if stats != nil {
			t.Fatalf("%s failed dispatch returned stats %#v", triple, stats)
		}
	}

	_, _, _, err = resolveExecutableBuildTarget("missing.tetra", "out", "unknown-target", BuildOptions{Jobs: 1})
	if err == nil || !strings.Contains(err.Error(), "unsupported target: unknown-target") {
		t.Fatalf("unknown target error = %v", err)
	}
}

func TestBuildTagFromOptionsIncludesLinkedObjectContentDeterministically(t *testing.T) {
	firstHash := sha256.Sum256([]byte("first-object"))
	secondHash := sha256.Sum256([]byte("second-object"))
	changedHash := sha256.Sum256([]byte("second-object-changed"))

	base := BuildOptions{DebugInfo: true}
	firstOrder := []linkedObject{
		{path: "b.tobj", obj: &Object{Module: "dep.b"}, contentHash: secondHash},
		{path: "a.tobj", obj: &Object{Module: "dep.a"}, contentHash: firstHash},
	}
	secondOrder := []linkedObject{
		{path: "a.tobj", obj: &Object{Module: "dep.a"}, contentHash: firstHash},
		{path: "b.tobj", obj: &Object{Module: "dep.b"}, contentHash: secondHash},
	}

	got := buildTagFromOptions(base, firstOrder)
	if got == "" || !strings.Contains(got, "debug-info") || !strings.Contains(got, "link=") {
		t.Fatalf("build tag = %q, want debug/link components", got)
	}
	if reordered := buildTagFromOptions(base, secondOrder); reordered != got {
		t.Fatalf("linked object build tag should be order independent: %q vs %q", got, reordered)
	}

	changed := buildTagFromOptions(base, []linkedObject{
		{path: "a.tobj", obj: &Object{Module: "dep.a"}, contentHash: firstHash},
		{path: "b.tobj", obj: &Object{Module: "dep.b"}, contentHash: changedHash},
	})
	if changed == got {
		t.Fatalf("linked object build tag did not change after content hash changed: %q", got)
	}
}

func TestPipelineLoadCheckedBuildWorldRequireMainStage(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"math/core.t4": "module math.core\npub func add(a: Int, b: Int) -> Int:\n    return a + b\n",
	})
	entry := filepath.Join(tmp, filepath.FromSlash("math/core.t4"))

	build, err := loadCheckedBuildWorld(entry, BuildOptions{Jobs: 1}, false)
	if err != nil {
		t.Fatalf("load checked world without main requirement: %v", err)
	}
	if build.world == nil || build.checked == nil {
		t.Fatalf("checked build world has nil fields: %#v", build)
	}
	if len(build.world.ByModule) != 1 || build.world.ByModule["math.core"] == nil {
		t.Fatalf("world modules = %#v, want math.core", build.world.ByModule)
	}

	_, err = loadCheckedBuildWorld(entry, BuildOptions{Jobs: 1}, true)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "main") {
		t.Fatalf("require-main error = %v", err)
	}
}

func TestPipelineNativeModulePlanInvalidatesWhenLinkedObjectContentChanges(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"app/main.t4": "module app.main\nfun main(): i32 {\n  return 42\n}\n",
	})
	entry := filepath.Join(tmp, filepath.FromSlash("app/main.t4"))
	opt := BuildOptions{Jobs: 1}

	build, err := loadCheckedBuildWorld(entry, opt, true)
	if err != nil {
		t.Fatalf("load checked world: %v", err)
	}
	target := "linux-x64"
	linkedA := []linkedObject{{path: "dep.tobj", obj: &Object{Module: "dep.lib"}, contentHash: sha256.Sum256([]byte("dep-v1"))}}
	plan1, stats1, err := planNativeModuleBuild(build.world, build.checked, target, opt, linkedA)
	if err != nil {
		t.Fatalf("plan first build: %v", err)
	}
	if len(plan1.toCompile) != 1 {
		t.Fatalf("first plan toCompile = %#v, want app.main", plan1.toCompile)
	}
	tgt, err := ctarget.Parse(target)
	if err != nil {
		t.Fatalf("parse target: %v", err)
	}
	codegen, err := nativeCodegenForTarget(tgt, opt)
	if err != nil {
		t.Fatalf("native codegen: %v", err)
	}
	backend, ok := nativeExecutableBackendForTarget(tgt)
	if !ok {
		t.Fatalf("native backend: %s", tgt.Triple)
	}
	native := nativeBuildTarget{target: tgt, triple: tgt.Triple, backend: backend, codegen: codegen}
	if err := compileNativeModulePlan(build.world, build.checked, native, opt, plan1, stats1); err != nil {
		t.Fatalf("compile first plan: %v", err)
	}

	plan2, stats2, err := planNativeModuleBuild(build.world, build.checked, target, opt, linkedA)
	if err != nil {
		t.Fatalf("plan cached build: %v", err)
	}
	if len(plan2.toCompile) != 0 || len(stats2.CacheHits) != 1 {
		t.Fatalf("cached plan toCompile=%#v cacheHits=%#v, want one cache hit", plan2.toCompile, stats2.CacheHits)
	}

	linkedB := []linkedObject{{path: "dep.tobj", obj: &Object{Module: "dep.lib"}, contentHash: sha256.Sum256([]byte("dep-v2"))}}
	plan3, stats3, err := planNativeModuleBuild(build.world, build.checked, target, opt, linkedB)
	if err != nil {
		t.Fatalf("plan changed link object build: %v", err)
	}
	if len(plan3.toCompile) != 1 {
		t.Fatalf("changed link object plan toCompile=%#v, want rebuild", plan3.toCompile)
	}
	if len(stats3.CacheHits) != 0 {
		t.Fatalf("changed link object cache hits=%#v, want none", stats3.CacheHits)
	}
}

func TestPipelineNativeModulePlanCacheStages(t *testing.T) {
	tmp := t.TempDir()
	writeTestFiles(t, tmp, map[string]string{
		"engine/render.t4": "module engine.render\nfun add_one(x: i32): i32 {\n  return x + 1\n}\n",
		"app/game.t4":      "module app.game\nimport engine.render as r\nfun main(): i32 {\n  return r.add_one(41)\n}\n",
	})
	entry := filepath.Join(tmp, filepath.FromSlash("app/game.t4"))
	opt := BuildOptions{Jobs: 1}

	build, err := loadCheckedBuildWorld(entry, opt, true)
	if err != nil {
		t.Fatalf("load checked world: %v", err)
	}
	tgt, err := ctarget.Parse("linux-x64")
	if err != nil {
		t.Fatalf("parse target: %v", err)
	}
	codegen, err := nativeCodegenForTarget(tgt, opt)
	if err != nil {
		t.Fatalf("native codegen: %v", err)
	}
	backend, ok := nativeExecutableBackendForTarget(tgt)
	if !ok {
		t.Fatalf("native backend: %s", tgt.Triple)
	}
	native := nativeBuildTarget{target: tgt, triple: tgt.Triple, backend: backend, codegen: codegen}

	plan1, stats1, err := planNativeModuleBuild(build.world, build.checked, native.triple, opt, nil)
	if err != nil {
		t.Fatalf("plan first build: %v", err)
	}
	assertModules(t, plan1.modules, []string{"app.game", "engine.render"})
	if len(plan1.toCompile) != 2 {
		t.Fatalf("first plan toCompile = %#v, want two modules", plan1.toCompile)
	}
	if len(stats1.CacheHits) != 0 {
		t.Fatalf("first plan cache hits = %#v, want none", stats1.CacheHits)
	}
	if err := compileNativeModulePlan(build.world, build.checked, native, opt, plan1, stats1); err != nil {
		t.Fatalf("compile first plan: %v", err)
	}
	assertModules(t, stats1.CompiledModules, []string{"app.game", "engine.render"})
	assertModules(t, stats1.LoweredModules, []string{"app.game", "engine.render"})
	objects, err := objectsFromModulePlan(plan1)
	if err != nil {
		t.Fatalf("objects from first plan: %v", err)
	}
	if len(objects) != 2 {
		t.Fatalf("objects len = %d, want 2", len(objects))
	}

	plan2, stats2, err := planNativeModuleBuild(build.world, build.checked, native.triple, opt, nil)
	if err != nil {
		t.Fatalf("plan cached build: %v", err)
	}
	if len(plan2.toCompile) != 0 {
		t.Fatalf("cached plan toCompile = %#v, want none", plan2.toCompile)
	}
	assertModules(t, stats2.CacheHits, []string{"app.game", "engine.render"})
	if err := compileNativeModulePlan(build.world, build.checked, native, opt, plan2, stats2); err != nil {
		t.Fatalf("compile cached plan: %v", err)
	}
	if len(stats2.CompiledModules) != 0 || len(stats2.LoweredModules) != 0 {
		t.Fatalf("cached stats compiled=%#v lowered=%#v, want none", stats2.CompiledModules, stats2.LoweredModules)
	}
	objects, err = objectsFromModulePlan(plan2)
	if err != nil {
		t.Fatalf("objects from cached plan: %v", err)
	}
	if len(objects) != 2 {
		t.Fatalf("cached objects len = %d, want 2", len(objects))
	}
}
