package semantics

import (
	"fmt"
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
		"make_u16",
		"make_i32",
		"make_bool",
		"island_new",
		"island_make_u8",
		"island_make_u16",
		"island_make_i32",
		"island_make_bool",
		"load_ptr",
		"store_ptr",
		"sym_addr",
		"ctx_switch",
		"time_now_ms",
		"sleep_ms",
		"sleep_until",
		"deadline_ms",
		"timer_ready",
		"yield",
		"task_group_open",
		"task_group_close",
		"task_group_cancel",
		"task_group_current",
		"task_group_status",
		"task_is_canceled",
		"task_checkpoint",
		"task_spawn_i32",
		"task_spawn_i32_typed",
		"task_spawn_group_i32",
		"task_spawn_group_i32_typed",
		"task_join_i32",
		"task_join_i32_typed",
		"task_join_group_i32_typed",
		"task_join_result_i32",
		"task_join_until_i32",
		"task_poll_i32",
		"select2_i32",
		"send_msg",
		"recv_msg",
		"recv_poll",
		"recv_until",
		"recv_msg_until",
		"send_typed",
		"recv_typed",
		"actor_dispatch",
		"actor_main_entry_id",
		"consent_token",
		"secret_seal_i32",
		"secret_unseal_i32",
	} {
		if target, ok := ResolveBuiltinAlias(alias); ok {
			aliasesByTarget[target] = append(aliasesByTarget[target], alias)
			continue
		}
		return nil, fmt.Errorf("builtin alias '%s' has no target", alias)
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
		case "core.island_make_u8", "core.island_make_u16", "core.island_make_i32", "core.island_make_bool":
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
		if err := validateBuiltinManifestEntry(out[len(out)-1]); err != nil {
			return nil, err
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func validateBuiltinManifestEntry(entry BuiltinManifest) error {
	if entry.Name == "" {
		return fmt.Errorf("manifest entry: missing name")
	}
	if entry.ReturnType == "" {
		return fmt.Errorf("manifest entry '%s': missing return type", entry.Name)
	}
	switch entry.UnsafePolicy {
	case "never", "always", "conditional":
		// valid
	default:
		return fmt.Errorf("manifest entry '%s': unknown unsafe policy '%s'", entry.Name, entry.UnsafePolicy)
	}
	for i := 1; i < len(entry.Aliases); i++ {
		if entry.Aliases[i-1] >= entry.Aliases[i] {
			return fmt.Errorf("manifest entry '%s': aliases must be sorted and unique", entry.Name)
		}
	}
	for i := 1; i < len(entry.Effects); i++ {
		if entry.Effects[i-1] >= entry.Effects[i] {
			return fmt.Errorf("manifest entry '%s': effects must be sorted and unique", entry.Name)
		}
	}
	return nil
}

func builtinEffects(name string) []string {
	var effects []string
	switch name {
	case "core.alloc_bytes":
		effects = []string{"alloc", "mem"}
	case "core.make_u8", "core.make_u16", "core.make_i32", "core.make_bool":
		effects = []string{"alloc", "mem"}
	case "core.island_new":
		effects = []string{"alloc", "islands", "mem"}
	case "core.island_make_u8", "core.island_make_u16", "core.island_make_i32", "core.island_make_bool":
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
	case "core.time_now_ms", "core.sleep_ms", "core.sleep_until", "core.deadline_ms", "core.timer_ready":
		effects = []string{"runtime"}
	case "core.yield":
		effects = []string{"actors", "runtime"}
	case "core.task_group_open", "core.task_group_close", "core.task_group_cancel", "core.task_group_current", "core.task_group_status",
		"core.task_is_canceled", "core.task_checkpoint",
		"core.task_spawn_i32", "core.task_spawn_i32_typed", "core.task_spawn_group_i32", "core.task_spawn_group_i32_typed",
		"core.task_join_i32", "core.task_join_i32_typed", "core.task_join_group_i32_typed", "core.task_join_result_i32", "core.task_join_until_i32",
		"core.task_poll_i32", "core.select2_i32":
		effects = []string{"runtime"}
	case "core.recv_until", "core.recv_msg_until":
		effects = []string{"actors", "runtime"}
	case "core.recv_poll":
		effects = []string{"actors"}
	case "core.actor_dispatch", "core.actor_main_entry_id",
		"core.spawn", "core.send", "core.send_msg", "core.recv", "core.recv_msg", "core.send_typed", "core.recv_typed", "core.self", "core.sender":
		effects = []string{"actors"}
	case "core.consent_token", "core.secret_seal_i32", "core.secret_unseal_i32":
		effects = []string{"privacy"}
	}
	sort.Strings(effects)
	return effects
}
