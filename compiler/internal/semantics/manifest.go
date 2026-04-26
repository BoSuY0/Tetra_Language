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
		"task_group_open",
		"task_group_close",
		"task_group_cancel",
		"task_spawn_i32",
		"task_spawn_group_i32",
		"task_join_i32",
		"task_join_result_i32",
		"send_msg",
		"recv_msg",
		"actor_dispatch",
		"actor_main_entry_id",
		"consent_token",
		"secret_seal_i32",
		"secret_unseal_i32",
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
	case "core.task_group_open", "core.task_group_close", "core.task_group_cancel",
		"core.task_spawn_i32", "core.task_spawn_group_i32",
		"core.task_join_i32", "core.task_join_result_i32":
		effects = []string{"runtime"}
	case "core.actor_dispatch", "core.actor_main_entry_id",
		"core.spawn", "core.send", "core.send_msg", "core.recv", "core.recv_msg", "core.self", "core.sender":
		effects = []string{"actors"}
	case "core.consent_token", "core.secret_seal_i32", "core.secret_unseal_i32":
		effects = []string{"privacy"}
	}
	sort.Strings(effects)
	return effects
}
