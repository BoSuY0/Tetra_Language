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

func TestManifestDescribeBuiltinsIncludesAtomicSurface(t *testing.T) {
	got, err := DescribeBuiltins()
	if err != nil {
		t.Fatalf("DescribeBuiltins: %v", err)
	}
	byName := map[string]BuiltinManifest{}
	for _, entry := range got {
		byName[entry.Name] = entry
	}

	tests := []struct {
		name       string
		params     []string
		returnType string
	}{
		{name: "core.atomic_load_i32_acquire", params: []string{"ptr", "cap.mem"}, returnType: "i32"},
		{name: "core.atomic_store_i32_release", params: []string{"ptr", "i32", "cap.mem"}, returnType: "i32"},
		{name: "core.atomic_compare_exchange_i32_acq_rel", params: []string{"ptr", "i32", "i32", "cap.mem"}, returnType: "i32"},
		{name: "core.atomic_compare_exchange_weak_i32_seq_cst", params: []string{"ptr", "i32", "i32", "cap.mem"}, returnType: "i32"},
		{name: "core.atomic_load_i64_acquire", params: []string{"ptr", "cap.mem"}, returnType: "i64"},
		{name: "core.atomic_compare_exchange_weak_i64_seq_cst", params: []string{"ptr", "i64", "i64", "cap.mem"}, returnType: "i64"},
		{name: "core.atomic_exchange_u8_seq_cst", params: []string{"ptr", "u8", "cap.mem"}, returnType: "u8"},
		{name: "core.atomic_exchange_u16_seq_cst", params: []string{"ptr", "u16", "cap.mem"}, returnType: "u16"},
		{name: "core.atomic_fetch_add_ptr_relaxed", params: []string{"ptr", "ptr", "cap.mem"}, returnType: "ptr"},
		{name: "core.atomic_fence_seq_cst", params: []string{"cap.mem"}, returnType: "i32"},
	}
	for _, tt := range tests {
		entry, ok := byName[tt.name]
		if !ok {
			t.Fatalf("manifest missing %s", tt.name)
		}
		if !reflect.DeepEqual(entry.ParamTypes, tt.params) {
			t.Fatalf("%s param types = %#v, want %#v", tt.name, entry.ParamTypes, tt.params)
		}
		if entry.ReturnType != tt.returnType {
			t.Fatalf("%s return type = %q, want %q", tt.name, entry.ReturnType, tt.returnType)
		}
		if strings.Join(entry.Effects, ",") != "mem" {
			t.Fatalf("%s effects = %q, want mem", tt.name, strings.Join(entry.Effects, ","))
		}
		if entry.UnsafePolicy != "always" {
			t.Fatalf("%s unsafe policy = %q, want always", tt.name, entry.UnsafePolicy)
		}
	}
}

func TestManifestDescribeBuiltinsIncludesNetSocketLifecycle(t *testing.T) {
	got, err := DescribeBuiltins()
	if err != nil {
		t.Fatalf("DescribeBuiltins: %v", err)
	}
	byName := map[string]BuiltinManifest{}
	for _, entry := range got {
		byName[entry.Name] = entry
	}
	tests := []struct {
		name       string
		params     []string
		returnType string
		effects    string
	}{
		{name: "core.net_socket_tcp4", params: []string{"cap.io"}, returnType: "i32", effects: "io"},
		{name: "core.net_bind_tcp4_loopback", params: []string{"i32", "i32", "cap.io"}, returnType: "i32", effects: "io"},
		{name: "core.net_connect_tcp4_loopback", params: []string{"i32", "i32", "cap.io"}, returnType: "i32", effects: "io"},
		{name: "core.net_listen", params: []string{"i32", "i32", "cap.io"}, returnType: "i32", effects: "io"},
		{name: "core.net_accept4", params: []string{"i32", "i32", "cap.io"}, returnType: "i32", effects: "io"},
		{name: "core.net_read", params: []string{"i32", "[]u8", "i32", "i32", "cap.io"}, returnType: "i32", effects: "io,mem"},
		{name: "core.net_recv", params: []string{"i32", "[]u8", "i32", "i32", "cap.io"}, returnType: "i32", effects: "io,mem"},
		{name: "core.net_write", params: []string{"i32", "[]u8", "i32", "i32", "cap.io"}, returnType: "i32", effects: "io,mem"},
		{name: "core.net_send", params: []string{"i32", "[]u8", "i32", "i32", "cap.io"}, returnType: "i32", effects: "io,mem"},
		{name: "core.net_epoll_create", params: []string{"cap.io"}, returnType: "i32", effects: "io"},
		{name: "core.net_epoll_ctl_add_read", params: []string{"i32", "i32", "cap.io"}, returnType: "i32", effects: "io"},
		{name: "core.net_epoll_ctl_add_read_write", params: []string{"i32", "i32", "cap.io"}, returnType: "i32", effects: "io"},
		{name: "core.net_epoll_ctl_mod_read", params: []string{"i32", "i32", "cap.io"}, returnType: "i32", effects: "io"},
		{name: "core.net_epoll_ctl_mod_read_write", params: []string{"i32", "i32", "cap.io"}, returnType: "i32", effects: "io"},
		{name: "core.net_epoll_ctl_delete", params: []string{"i32", "i32", "cap.io"}, returnType: "i32", effects: "io"},
		{name: "core.net_epoll_wait_one", params: []string{"i32", "i32", "cap.io"}, returnType: "i32", effects: "io"},
		{name: "core.net_epoll_wait_one_into", params: []string{"i32", "[]i32", "i32", "cap.io"}, returnType: "i32", effects: "io,mem"},
		{name: "core.net_set_nonblocking", params: []string{"i32", "cap.io"}, returnType: "i32", effects: "io"},
		{name: "core.net_set_reuseport", params: []string{"i32", "cap.io"}, returnType: "i32", effects: "io"},
		{name: "core.net_set_tcp_nodelay", params: []string{"i32", "cap.io"}, returnType: "i32", effects: "io"},
		{name: "core.net_close", params: []string{"i32", "cap.io"}, returnType: "i32", effects: "io"},
	}
	for _, tt := range tests {
		entry, ok := byName[tt.name]
		if !ok {
			t.Fatalf("manifest missing %s", tt.name)
		}
		if !reflect.DeepEqual(entry.ParamTypes, tt.params) {
			t.Fatalf("%s param types = %#v, want %#v", tt.name, entry.ParamTypes, tt.params)
		}
		if entry.ReturnType != tt.returnType {
			t.Fatalf("%s return type = %q, want %q", tt.name, entry.ReturnType, tt.returnType)
		}
		if strings.Join(entry.Effects, ",") != tt.effects {
			t.Fatalf("%s effects = %q, want %q", tt.name, strings.Join(entry.Effects, ","), tt.effects)
		}
		if entry.UnsafePolicy != "never" {
			t.Fatalf("%s unsafe policy = %q, want never", tt.name, entry.UnsafePolicy)
		}
	}
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
