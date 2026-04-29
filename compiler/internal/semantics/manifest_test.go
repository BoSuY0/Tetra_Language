package semantics

import "testing"

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
