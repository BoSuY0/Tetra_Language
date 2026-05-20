package semantics

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestManifestDescribeBuiltinsSortedAndAliasStable(t *testing.T) {
	got, err := DescribeBuiltins()
	if err != nil {
		t.Fatalf("DescribeBuiltins: %v", err)
	}
	if len(got) == 0 {
		t.Fatalf("expected non-empty builtin manifest")
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].Name >= got[i].Name {
			t.Fatalf("manifest names not sorted: %q then %q", got[i-1].Name, got[i].Name)
		}
	}

	foundMakeU8 := false
	foundMakeU16 := false
	foundMakeBool := false
	foundIslandMakeBool := false
	for _, entry := range got {
		switch entry.Name {
		case "core.make_u8":
			foundMakeU8 = true
			if len(entry.Aliases) == 0 || entry.Aliases[0] != "make_u8" {
				t.Fatalf("core.make_u8 aliases = %#v, want leading make_u8", entry.Aliases)
			}
			if entry.ReturnType != "[]u8" {
				t.Fatalf("core.make_u8 return type = %q, want []u8", entry.ReturnType)
			}
		case "core.make_u16":
			foundMakeU16 = true
			if len(entry.Aliases) == 0 || entry.Aliases[0] != "make_u16" {
				t.Fatalf("core.make_u16 aliases = %#v, want leading make_u16", entry.Aliases)
			}
			if entry.ReturnType != "[]u16" {
				t.Fatalf("core.make_u16 return type = %q, want []u16", entry.ReturnType)
			}
		case "core.make_bool":
			foundMakeBool = true
			if len(entry.Aliases) == 0 || entry.Aliases[0] != "make_bool" {
				t.Fatalf("core.make_bool aliases = %#v, want leading make_bool", entry.Aliases)
			}
			if entry.ReturnType != "[]bool" {
				t.Fatalf("core.make_bool return type = %q, want []bool", entry.ReturnType)
			}
		case "core.island_make_bool":
			foundIslandMakeBool = true
			if len(entry.Aliases) == 0 || entry.Aliases[0] != "island_make_bool" {
				t.Fatalf("core.island_make_bool aliases = %#v, want leading island_make_bool", entry.Aliases)
			}
			if entry.ReturnType != "[]bool" {
				t.Fatalf("core.island_make_bool return type = %q, want []bool", entry.ReturnType)
			}
			if entry.UnsafePolicy != "conditional" {
				t.Fatalf("core.island_make_bool unsafe policy = %q, want conditional", entry.UnsafePolicy)
			}
		}
	}
	if !foundMakeU8 {
		t.Fatalf("missing core.make_u8 in manifest output")
	}
	if !foundMakeU16 {
		t.Fatalf("missing core.make_u16 in manifest output")
	}
	if !foundMakeBool {
		t.Fatalf("missing core.make_bool in manifest output")
	}
	if !foundIslandMakeBool {
		t.Fatalf("missing core.island_make_bool in manifest output")
	}
}

func TestManifestValidationRejectsInvalidUnsafePolicy(t *testing.T) {
	err := validateBuiltinManifestEntry(BuiltinManifest{
		Name:         "core.fake",
		ReturnType:   "i32",
		UnsafePolicy: "sometimes",
	})
	if err == nil {
		t.Fatalf("expected unsafe policy validation error")
	}
}

func TestManifestValidationRejectsUnsortedEffectsOrAliases(t *testing.T) {
	tests := []BuiltinManifest{
		{
			Name:         "core.fake.aliases",
			ReturnType:   "i32",
			UnsafePolicy: "never",
			Aliases:      []string{"z", "a"},
		},
		{
			Name:         "core.fake.effects",
			ReturnType:   "i32",
			UnsafePolicy: "never",
			Effects:      []string{"runtime", "actors"},
		},
	}
	for _, tc := range tests {
		if err := validateBuiltinManifestEntry(tc); err == nil {
			t.Fatalf("expected validation error for %#v", tc)
		}
	}
}

func TestManifestValidationAcceptsWellFormedEntry(t *testing.T) {
	err := validateBuiltinManifestEntry(BuiltinManifest{
		Name:         "core.fake",
		ReturnType:   "i32",
		UnsafePolicy: "conditional",
		Aliases:      []string{"fake"},
		Effects:      []string{"mem", "runtime"},
	})
	if err != nil {
		t.Fatalf("validateBuiltinManifestEntry: %v", err)
	}
}

func TestManifestDescribeBuiltinsIncludesFilesystemExists(t *testing.T) {
	got, err := DescribeBuiltins()
	if err != nil {
		t.Fatalf("DescribeBuiltins: %v", err)
	}
	for _, entry := range got {
		if entry.Name != "core.fs_exists" {
			continue
		}
		if !reflect.DeepEqual(entry.ParamTypes, []string{"str", "cap.io"}) {
			t.Fatalf("core.fs_exists param types = %#v, want str, cap.io", entry.ParamTypes)
		}
		if entry.ReturnType != "bool" {
			t.Fatalf("core.fs_exists return type = %q, want bool", entry.ReturnType)
		}
		if strings.Join(entry.Effects, ",") != "io" {
			t.Fatalf("core.fs_exists effects = %q, want io", strings.Join(entry.Effects, ","))
		}
		if entry.UnsafePolicy != "never" {
			t.Fatalf("core.fs_exists unsafe policy = %q, want never", entry.UnsafePolicy)
		}
		return
	}
	t.Fatalf("manifest missing core.fs_exists")
}

func TestManifestDriftProofAgainstBuiltinPolicySources(t *testing.T) {
	got, err := DescribeBuiltins()
	if err != nil {
		t.Fatalf("DescribeBuiltins: %v", err)
	}

	types := baseTypes()
	sigs, err := builtinFuncSigs(types)
	if err != nil {
		t.Fatalf("builtinFuncSigs: %v", err)
	}

	if len(got) != len(sigs) {
		t.Fatalf("manifest size = %d, builtin signatures = %d", len(got), len(sigs))
	}

	entriesByName := make(map[string]BuiltinManifest, len(got))
	for _, entry := range got {
		entriesByName[entry.Name] = entry
	}

	expectedAliasesByTarget := make(map[string][]string, len(sigs))
	for target := range sigs {
		short := strings.TrimPrefix(target, "core.")
		if resolved, ok := ResolveBuiltinAlias(short); ok {
			if resolved != target {
				t.Fatalf("ResolveBuiltinAlias(%q) = %q, want %q", short, resolved, target)
			}
			expectedAliasesByTarget[target] = append(expectedAliasesByTarget[target], short)
		}
	}
	for target, aliases := range expectedAliasesByTarget {
		sort.Strings(aliases)
		expectedAliasesByTarget[target] = aliases
	}

	for name, sig := range sigs {
		entry, ok := entriesByName[name]
		if !ok {
			t.Fatalf("manifest missing builtin %q", name)
		}
		if !reflect.DeepEqual(entry.ParamTypes, sig.ParamTypes) {
			t.Fatalf("%s param types = %#v, want %#v", name, entry.ParamTypes, sig.ParamTypes)
		}
		if entry.ReturnType != sig.ReturnType {
			t.Fatalf("%s return type = %q, want %q", name, entry.ReturnType, sig.ReturnType)
		}

		wantEffects := builtinEffects(name)
		if !reflect.DeepEqual(entry.Effects, wantEffects) {
			t.Fatalf("%s effects = %#v, want %#v", name, entry.Effects, wantEffects)
		}

		wantPolicy, wantDetails := expectedBuiltinUnsafePolicy(name)
		if entry.UnsafePolicy != wantPolicy {
			t.Fatalf("%s unsafe policy = %q, want %q", name, entry.UnsafePolicy, wantPolicy)
		}
		if entry.UnsafeDetails != wantDetails {
			t.Fatalf("%s unsafe details = %q, want %q", name, entry.UnsafeDetails, wantDetails)
		}

		wantAliases := expectedAliasesByTarget[name]
		if !reflect.DeepEqual(entry.Aliases, wantAliases) {
			t.Fatalf("%s aliases = %#v, want %#v", name, entry.Aliases, wantAliases)
		}
		for _, alias := range entry.Aliases {
			resolved, ok := ResolveBuiltinAlias(alias)
			if !ok || resolved != name {
				t.Fatalf("%s alias %q resolves to (%q, %v), want (%q, true)", name, alias, resolved, ok, name)
			}
		}
	}
}

func expectedBuiltinUnsafePolicy(name string) (policy string, details string) {
	switch name {
	case "core.island_make_u8", "core.island_make_u16", "core.island_make_i32", "core.island_make_bool":
		return "conditional", "requires unsafe when the island argument is not a scoped island variable"
	default:
		if builtinNeedsUnsafe(name, nil) {
			return "always", ""
		}
		return "never", ""
	}
}
