package semantics

import (
	"sort"
)

type BuiltinManifest struct {
	Name          string   `json:"name"`
	Aliases       []string `json:"aliases,omitempty"`
	ParamTypes    []string `json:"param_types,omitempty"`
	ReturnType    string   `json:"return_type"`
	Effects       []string `json:"effects,omitempty"`
	UnsafePolicy  string   `json:"unsafe_policy"`            // never | always | conditional
	UnsafeDetails string   `json:"unsafe_details,omitempty"` // human-readable condition
}

// DescribeBuiltins returns a stable, sorted snapshot of builtin signatures and safety policies.
func DescribeBuiltins() ([]BuiltinManifest, error) {
	types := baseTypes()
	sigs, err := builtinFuncSigs(types)
	if err != nil {
		return nil, err
	}

	aliasesByTarget := make(map[string][]string)
	for _, alias := range []string{
		"alloc_bytes",
		"make_u8",
		"make_i32",
		"island_new",
		"island_make_u8",
		"island_make_i32",
		"load_ptr",
		"store_ptr",
		"sym_addr",
		"ctx_switch",
		"actor_dispatch",
		"actor_main_entry_id",
	} {
		if target, ok := ResolveBuiltinAlias(alias); ok {
			aliasesByTarget[target] = append(aliasesByTarget[target], alias)
		}
	}
	for name, list := range aliasesByTarget {
		sort.Strings(list)
		aliasesByTarget[name] = list
	}

	out := make([]BuiltinManifest, 0, len(sigs))
	for name, sig := range sigs {
		effects := builtinEffects(name)
		unsafePolicy := "never"
		unsafeDetails := ""
		switch name {
		case "core.island_make_u8", "core.island_make_i32":
			unsafePolicy = "conditional"
			unsafeDetails = "requires unsafe when the island argument is not a scoped island variable"
		default:
			if builtinNeedsUnsafe(name, nil) {
				unsafePolicy = "always"
			}
		}
		out = append(out, BuiltinManifest{
			Name:          name,
			Aliases:       aliasesByTarget[name],
			ParamTypes:    append([]string(nil), sig.ParamTypes...),
			ReturnType:    sig.ReturnType,
			Effects:       effects,
			UnsafePolicy:  unsafePolicy,
			UnsafeDetails: unsafeDetails,
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func builtinEffects(name string) []string {
	var effects []string
	switch name {
	case "core.alloc_bytes":
		effects = []string{"alloc", "mem"}
	case "core.make_u8", "core.make_i32":
		effects = []string{"alloc", "mem"}
	case "core.island_new":
		effects = []string{"alloc", "islands", "mem"}
	case "core.island_make_u8", "core.island_make_i32":
		effects = []string{"alloc", "islands", "mem"}
	case "core.cap_io":
		effects = []string{"capability", "io"}
	case "core.cap_mem":
		effects = []string{"capability", "mem"}
	case "core.load_i32", "core.store_i32",
		"core.load_u8", "core.store_u8",
		"core.load_ptr", "core.store_ptr",
		"core.ptr_add":
		effects = []string{"mem"}
	case "core.mmio_read_i32", "core.mmio_write_i32":
		effects = []string{"io", "mmio"}
	case "core.sym_addr":
		effects = []string{"link"}
	case "core.ctx_switch":
		effects = []string{"control", "runtime"}
	case "core.actor_dispatch", "core.actor_main_entry_id",
		"core.spawn", "core.send", "core.recv", "core.self", "core.sender":
		effects = []string{"actors"}
	}
	sort.Strings(effects)
	return effects
}
