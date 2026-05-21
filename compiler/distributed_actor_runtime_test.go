package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/compiler/internal/semantics"
)

func TestCollectDistributedActorRuntimeUsage(t *testing.T) {
	checked := checkedDistributedActorProgram(t, `
func worker() -> Int:
    return 0

func main() -> Int
uses actors, runtime:
    let _connected: Int = core.actor_node_connect(2, 5010)
    let _peer: actor = core.spawn_remote(2, "worker")
    return core.actor_node_status(2)
`)

	used, _ := collectDistributedActorRuntimeUsagePosition(checked)
	if !used {
		t.Fatalf("distributed actor runtime usage was not detected")
	}

	actorsUsed, entries, _, err := collectActorEntries(checked)
	if err != nil {
		t.Fatalf("collectActorEntries: %v", err)
	}
	if !actorsUsed {
		t.Fatalf("actor runtime usage was not detected")
	}
	if !containsString(entries, "worker") {
		t.Fatalf("actor entries = %v, want remote spawn target worker", entries)
	}
}

func TestRequiredDistributedActorRuntimeSymbols(t *testing.T) {
	got := map[string]struct{}{}
	for _, name := range requiredDistributedActorRuntimeSymbols() {
		got[name] = struct{}{}
	}
	for _, name := range []string{
		"__tetra_actor_node_connect",
		"__tetra_actor_spawn_remote",
		"__tetra_actor_node_status",
	} {
		if _, ok := got[name]; !ok {
			t.Fatalf("required distributed actor runtime symbols missing %q", name)
		}
		sig, ok := runtimeObjectSignature(name)
		if !ok {
			t.Fatalf("missing runtime signature for %q", name)
		}
		if sig.returnSlots != 1 {
			t.Fatalf("%s return slots = %d, want 1", name, sig.returnSlots)
		}
	}
}

func TestRuntimeObjectValidationRejectsMissingDistributedActorSymbols(t *testing.T) {
	obj := &Object{Symbols: runtimeObjectSymbols(requiredActorRuntimeSymbols())}
	annotateRuntimeObjectSignatures(obj)
	err := validateDistributedActorRuntimeObject(obj)
	if err == nil {
		t.Fatalf("expected missing distributed actor runtime symbol error")
	}
	if !strings.Contains(err.Error(), "runtime object missing required symbol '__tetra_actor_node_connect'") {
		t.Fatalf("error = %v", err)
	}
}

func TestDistributedActorRuntimeRejectsUnsupportedNativeTargets(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "distributed_actor_status.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func main() -> Int
uses actors, runtime:
    return core.actor_node_status(2)
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	for _, target := range []string{"macos-x64", "windows-x64"} {
		t.Run(target, func(t *testing.T) {
			outPath := filepath.Join(tmp, "distributed-"+target)
			_, err := BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{Jobs: 1})
			if err == nil {
				t.Fatalf("expected unsupported distributed actors runtime diagnostic")
			}
			want := "distributed actors runtime not supported on " + target
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, want %q", err, want)
			}
		})
	}
}

func TestDistributedActorRuntimeBuildsWithLinuxBuiltinRuntime(t *testing.T) {
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "distributed_actor_linux.tetra")
	if err := os.WriteFile(srcPath, []byte(`
func worker() -> Int:
    return 0

func main() -> Int
uses actors, runtime:
    let _connected: Int = core.actor_node_connect(1, 4599)
    let _peer: actor = core.spawn_remote(2, "worker")
    return core.actor_node_status(2)
`), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	outPath := filepath.Join(tmp, "distributed-actor-linux")
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{Jobs: 1}); err != nil {
		t.Fatalf("linux distributed actor build: %v", err)
	}
}

func checkedDistributedActorProgram(t *testing.T, src string) *semantics.CheckedProgram {
	t.Helper()
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "main.tetra")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	world, err := LoadWorld(srcPath)
	if err != nil {
		t.Fatalf("LoadWorld: %v", err)
	}
	checked, err := CheckWorld(world)
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	return checked
}
