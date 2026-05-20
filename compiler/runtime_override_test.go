package compiler

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tetra_language/compiler/internal/actorsrt"
	"tetra_language/compiler/target"
)

func TestRuntimeObjectOverrideActorsPingPong(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	srcPath := filepath.Join("..", "examples", "actors_pingpong.tetra")
	if _, err := os.Stat(srcPath); err != nil {
		t.Fatalf("missing example: %v", err)
	}

	world, err := LoadWorld(srcPath)
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	actorsUsed, actorEntries, err := collectActorEntries(checked)
	if err != nil {
		t.Fatalf("collect actor entries: %v", err)
	}
	if !actorsUsed || len(actorEntries) == 0 {
		t.Fatalf("expected actors usage")
	}

	var rt *Object
	switch tgt.Triple {
	case "linux-x64":
		rt, err = actorsrt.BuildLinuxX64(actorEntries)
	case "macos-x64":
		rt, err = actorsrt.BuildMacOSX64(actorEntries)
	case "windows-x64":
		rt, err = actorsrt.BuildWindowsX64(actorEntries)
	default:
		t.Fatalf("unsupported target: %s", tgt.Triple)
	}
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	rt.Target = tgt.Triple
	rt.Module = "__runtime"

	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime.tobj")
	if err := WriteObject(rtPath, rt); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	outPath := filepath.Join(tmp, "actors_pingpong"+tgt.ExeExt)
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{RuntimeObjectPath: rtPath}); err != nil {
		t.Fatalf("build with runtime override: %v", err)
	}
	stdout, exitCode := runBinary(t, outPath)
	if stdout != "" {
		t.Fatalf("stdout mismatch: %q", stdout)
	}
	if exitCode != 0 {
		t.Fatalf("exit code mismatch: %d", exitCode)
	}
}

func TestRuntimeObjectOverrideRelinksWhenRuntimeObjectChanges(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	srcPath := filepath.Join("..", "examples", "actors_pingpong.tetra")
	world, err := LoadWorld(srcPath)
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	actorsUsed, actorEntries, err := collectActorEntries(checked)
	if err != nil {
		t.Fatalf("collect actor entries: %v", err)
	}
	if !actorsUsed || len(actorEntries) == 0 {
		t.Fatalf("expected actors usage")
	}

	rt, err := buildHostRuntimeObject(tgt.Triple, actorEntries)
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime.tobj")
	rt.Target = tgt.Triple
	rt.Module = "__runtime"
	if err := WriteObject(rtPath, rt); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	outPath := filepath.Join(tmp, "actors_pingpong"+tgt.ExeExt)
	stats1, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{RuntimeObjectPath: rtPath})
	if err != nil {
		t.Fatalf("build1 with runtime override: %v", err)
	}
	first, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read build1 output: %v", err)
	}

	rt.Code = append(rt.Code, 0x90)
	if err := WriteObject(rtPath, rt); err != nil {
		t.Fatalf("rewrite runtime object: %v", err)
	}
	stats2, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{RuntimeObjectPath: rtPath})
	if err != nil {
		t.Fatalf("build2 with changed runtime override: %v", err)
	}
	second, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read build2 output: %v", err)
	}
	if sha256.Sum256(first) == sha256.Sum256(second) {
		t.Fatalf("output did not change after runtime object changed")
	}
	if len(stats1.CompiledModules) == 0 && len(stats1.CacheHits) == 0 {
		t.Fatalf("first build had no module activity")
	}
	if len(stats2.CacheHits) == 0 {
		t.Fatalf("second build should still be able to reuse program module cache while relinking runtime")
	}
}

func buildHostRuntimeObject(triple string, actorEntries []string) (*Object, error) {
	switch triple {
	case "linux-x64":
		return actorsrt.BuildLinuxX64(actorEntries)
	case "macos-x64":
		return actorsrt.BuildMacOSX64(actorEntries)
	case "windows-x64":
		return actorsrt.BuildWindowsX64(actorEntries)
	default:
		return nil, target.UnsupportedTargetError{Triple: triple}
	}
}

func TestRuntimeObjectOverrideRejectsWithoutRuntimeUsage(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime_unused.tobj")
	if err := WriteObject(rtPath, &Object{
		Target:  tgt.Triple,
		Module:  "__runtime_unused",
		Code:    []byte{0xC3},
		Symbols: []Symbol{{Name: "__tetra_entry", Offset: 0}},
	}); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	srcPath := filepath.Join(tmp, "plain_main.t4")
	if err := os.WriteFile(srcPath, []byte(`func main() -> Int:
    return 0
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	outPath := filepath.Join(tmp, "plain_main"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{RuntimeObjectPath: rtPath})
	if err == nil {
		t.Fatalf("expected runtime object override without runtime usage to fail")
	}
	if !strings.Contains(err.Error(), "runtime object override requires runtime usage") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeObjectOverrideRejectsMissingRequiredSymbols(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime_missing_symbols.tobj")
	if err := WriteObject(rtPath, &Object{
		Target:  tgt.Triple,
		Module:  "__runtime_missing",
		Code:    []byte{0xC3},
		Symbols: []Symbol{{Name: "__tetra_entry", Offset: 0}},
	}); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	outPath := filepath.Join(tmp, "actors_pingpong"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(filepath.Join("..", "examples", "actors_pingpong.tetra"), outPath, tgt.Triple, BuildOptions{RuntimeObjectPath: rtPath})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "runtime object missing required symbol") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeObjectOverrideRejectsSignatureMismatch(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	srcPath := filepath.Join("..", "examples", "actors_pingpong.tetra")
	world, err := LoadWorld(srcPath)
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	actorsUsed, actorEntries, err := collectActorEntries(checked)
	if err != nil {
		t.Fatalf("collect actor entries: %v", err)
	}
	if !actorsUsed || len(actorEntries) == 0 {
		t.Fatalf("expected actors usage")
	}

	rt, err := buildHostRuntimeObject(tgt.Triple, actorEntries)
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	rt.Target = tgt.Triple
	rt.Module = "__runtime"
	annotateRuntimeObjectSignatures(rt)
	replaceRuntimeSymbolSignature(rt, "__tetra_actor_spawn", 2, 1)

	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime_wrong_signature.tobj")
	if err := WriteObject(rtPath, rt); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	outPath := filepath.Join(tmp, "actors_pingpong"+tgt.ExeExt)
	_, err = BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{RuntimeObjectPath: rtPath})
	if err == nil {
		t.Fatalf("expected signature mismatch error, got nil")
	}
	if !strings.Contains(err.Error(), "runtime object symbol '__tetra_actor_spawn' signature mismatch") ||
		!strings.Contains(err.Error(), "params=2 want=1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeObjectOverrideRejectsMissingTaggedMessageSymbols(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	symbols := make([]Symbol, 0, len(requiredActorRuntimeSymbols()))
	for _, name := range []string{
		"__tetra_entry",
		"__tetra_actor_spawn",
		"__tetra_actor_send",
		"__tetra_actor_recv",
		"__tetra_actor_self",
		"__tetra_actor_sender",
	} {
		symbols = append(symbols, Symbol{Name: name, Offset: 0})
	}

	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime_missing_tagged_msg_symbols.tobj")
	if err := WriteObject(rtPath, &Object{
		Target:  tgt.Triple,
		Module:  "__runtime_missing_tagged_msg",
		Code:    []byte{0xC3},
		Symbols: symbols,
	}); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	outPath := filepath.Join(tmp, "actors_tagged_stress"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(filepath.Join("..", "examples", "actors_tagged_stress.tetra"), outPath, tgt.Triple, BuildOptions{RuntimeObjectPath: rtPath})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "runtime object missing required symbol '__tetra_actor_send_msg'") &&
		!strings.Contains(err.Error(), "runtime object missing required symbol '__tetra_actor_recv_msg'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeObjectOverrideRejectsMissingActorStateSymbols(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	symbols := make([]Symbol, 0, len(requiredActorRuntimeSymbols()))
	for _, name := range requiredActorRuntimeSymbols() {
		symbols = append(symbols, Symbol{Name: name, Offset: 0})
	}

	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime_missing_actor_state_symbols.tobj")
	if err := WriteObject(rtPath, &Object{
		Target:  tgt.Triple,
		Module:  "__runtime_missing_actor_state",
		Code:    []byte{0xC3},
		Symbols: symbols,
	}); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	srcPath := filepath.Join(tmp, "actor_state_main.tetra")
	src := `actor Counter:
    var count: Int = 0
    func run() -> Int:
        count = count + 1
        return count

func main() -> Int
uses actors:
    let _peer: actor = core.spawn("Counter.run")
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	outPath := filepath.Join(tmp, "actor_state_main"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{RuntimeObjectPath: rtPath})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "runtime object missing required symbol '__tetra_actor_state_load'") &&
		!strings.Contains(err.Error(), "runtime object missing required symbol '__tetra_actor_state_store'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeObjectOverrideRejectsMissingTimeSymbols(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	symbols := make([]Symbol, 0, len(requiredActorRuntimeSymbols()))
	for _, name := range requiredActorRuntimeSymbols() {
		symbols = append(symbols, Symbol{Name: name, Offset: 0})
	}

	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime_missing_time_symbols.tobj")
	if err := WriteObject(rtPath, &Object{
		Target:  tgt.Triple,
		Module:  "__runtime_missing_time",
		Code:    []byte{0xC3},
		Symbols: symbols,
	}); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	srcPath := filepath.Join(tmp, "main.tetra")
	outPath := filepath.Join(tmp, "time"+tgt.ExeExt)
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses runtime:
    return core.time_now_ms()
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	_, err := BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, BuildOptions{RuntimeObjectPath: rtPath})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "runtime object missing required symbol '__tetra_time_now_ms'") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeObjectOverrideRejectsTargetMismatch(t *testing.T) {
	tgt, ok := target.Host()
	if !ok {
		t.Skipf("unsupported host: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("amd64 only")
	}

	other := "windows-x64"
	if tgt.Triple == "windows-x64" {
		other = "linux-x64"
	}

	tmp := t.TempDir()
	rtPath := filepath.Join(tmp, "runtime_wrong_target.tobj")
	if err := WriteObject(rtPath, &Object{
		Target:  other,
		Module:  "__runtime_wrong_target",
		Code:    []byte{0xC3},
		Symbols: []Symbol{{Name: "__tetra_entry", Offset: 0}},
	}); err != nil {
		t.Fatalf("write runtime object: %v", err)
	}

	outPath := filepath.Join(tmp, "actors_pingpong"+tgt.ExeExt)
	_, err := BuildFileWithStatsOpt(filepath.Join("..", "examples", "actors_pingpong.tetra"), outPath, tgt.Triple, BuildOptions{RuntimeObjectPath: rtPath})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "runtime object target mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeObjectOverrideBuildsForAllX64Targets(t *testing.T) {
	srcPath := filepath.Join("..", "examples", "actors_pingpong.tetra")
	world, err := LoadWorld(srcPath)
	if err != nil {
		t.Fatalf("load world: %v", err)
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	actorsUsed, actorEntries, err := collectActorEntries(checked)
	if err != nil {
		t.Fatalf("collect actor entries: %v", err)
	}
	if !actorsUsed || len(actorEntries) == 0 {
		t.Fatalf("expected actors usage")
	}

	for _, triple := range []string{"linux-x64", "macos-x64", "windows-x64"} {
		t.Run(triple, func(t *testing.T) {
			tgt, err := target.Parse(triple)
			if err != nil {
				t.Fatalf("parse target: %v", err)
			}
			rt, err := buildHostRuntimeObject(triple, actorEntries)
			if err != nil {
				t.Fatalf("build runtime: %v", err)
			}
			rt.Target = triple
			rt.Module = "__runtime"
			annotateRuntimeObjectSignatures(rt)

			tmp := t.TempDir()
			rtPath := filepath.Join(tmp, "runtime.tobj")
			if err := WriteObject(rtPath, rt); err != nil {
				t.Fatalf("write runtime object: %v", err)
			}
			outPath := filepath.Join(tmp, "actors_pingpong"+tgt.ExeExt)
			if _, err := BuildFileWithStatsOpt(srcPath, outPath, triple, BuildOptions{RuntimeObjectPath: rtPath}); err != nil {
				t.Fatalf("build with runtime override: %v", err)
			}
			if _, err := os.Stat(outPath); err != nil {
				t.Fatalf("missing output: %v", err)
			}
		})
	}
}
