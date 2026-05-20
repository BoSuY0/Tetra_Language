package compiler

import (
	"reflect"
	"strings"
	"testing"
)

func TestManifestRuntimeABIIncludesFullRequiredSymbolSets(t *testing.T) {
	manifest, err := GetManifest()
	if err != nil {
		t.Fatalf("GetManifest: %v", err)
	}

	assertSymbolSequence(t, "actors_required_symbols", manifest.RuntimeABI.ActorsRequiredSymbols, requiredActorRuntimeSymbols())
	assertSymbolSequence(t, "actor_state_required_symbols", manifest.RuntimeABI.ActorStateRequiredSymbols, requiredActorStateRuntimeSymbols())
	assertSymbolSequence(t, "task_required_symbols", manifest.RuntimeABI.TaskRequiredSymbols, requiredTaskRuntimeSymbols())
	assertSymbolSequence(t, "task_group_required_symbols", manifest.RuntimeABI.TaskGroupRequiredSymbols, requiredTaskGroupRuntimeSymbols())
	assertSymbolSequence(t, "typed_task_required_symbols", manifest.RuntimeABI.TypedTaskRequiredSymbols, requiredTypedTaskRuntimeSymbols(8))
	assertSymbolSequence(t, "time_required_symbols", manifest.RuntimeABI.TimeRequiredSymbols, requiredTimeRuntimeSymbols())
	assertSymbolSequence(t, "filesystem_required_symbols", manifest.RuntimeABI.FilesystemRequiredSymbols, requiredFilesystemRuntimeSymbols())
}

func assertSymbolSequence(t *testing.T, field string, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s = %#v, want %#v", field, got, want)
	}
}

func TestManifestBuiltinsExposeStableUnsafePoliciesForPublicSurface(t *testing.T) {
	manifest, err := GetManifest()
	if err != nil {
		t.Fatalf("GetManifest: %v", err)
	}

	byName := map[string]BuiltinManifest{}
	for _, builtin := range manifest.Builtins {
		byName[builtin.Name] = builtin
	}

	for name, wantEffects := range map[string]string{
		"core.cap_io":         "capability,io",
		"core.cap_mem":        "capability,mem",
		"core.load_i32":       "mem",
		"core.store_i32":      "mem",
		"core.mmio_read_i32":  "io,mmio",
		"core.mmio_write_i32": "io,mmio",
	} {
		got, ok := byName[name]
		if !ok {
			t.Fatalf("manifest missing builtin %s", name)
		}
		if got.UnsafePolicy != "always" {
			t.Fatalf("%s unsafe_policy = %q, want always", name, got.UnsafePolicy)
		}
		if got.UnsafeDetails != "" {
			t.Fatalf("%s unsafe_details = %q, want empty", name, got.UnsafeDetails)
		}
		if strings.Join(got.Effects, ",") != wantEffects {
			t.Fatalf("%s effects = %q, want %q", name, strings.Join(got.Effects, ","), wantEffects)
		}
	}

	const wantConditionalUnsafeDetails = "requires unsafe when the island argument is not a scoped island variable"
	for _, name := range []string{
		"core.island_make_u8",
		"core.island_make_u16",
		"core.island_make_i32",
		"core.island_make_bool",
	} {
		got, ok := byName[name]
		if !ok {
			t.Fatalf("manifest missing builtin %s", name)
		}
		if got.UnsafePolicy != "conditional" {
			t.Fatalf("%s unsafe_policy = %q, want conditional", name, got.UnsafePolicy)
		}
		if got.UnsafeDetails != wantConditionalUnsafeDetails {
			t.Fatalf("%s unsafe_details = %q, want %q", name, got.UnsafeDetails, wantConditionalUnsafeDetails)
		}
	}
}
