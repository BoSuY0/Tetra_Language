package semantics

func builtinFuncSigs(types map[string]*TypeInfo) (map[string]FuncSig, error) {
	_, err := ensureTypeInfo("[]u8", types)
	if err != nil {
		return nil, err
	}
	_, err = ensureTypeInfo("[]i32", types)
	if err != nil {
		return nil, err
	}
	actorInfo, err := ensureTypeInfo("actor", types)
	if err != nil {
		return nil, err
	}
	ptrInfo, err := ensureTypeInfo("ptr", types)
	if err != nil {
		return nil, err
	}
	sliceU8, err := ensureTypeInfo("[]u8", types)
	if err != nil {
		return nil, err
	}
	sliceI32, err := ensureTypeInfo("[]i32", types)
	if err != nil {
		return nil, err
	}

	islandInfo, err := ensureTypeInfo("island", types)
	if err != nil {
		return nil, err
	}
	capIO, err := ensureTypeInfo("cap.io", types)
	if err != nil {
		return nil, err
	}
	capMem, err := ensureTypeInfo("cap.mem", types)
	if err != nil {
		return nil, err
	}

	sigs := map[string]FuncSig{
		"core.alloc_bytes":         {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "ptr", ReturnSlots: ptrInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.make_u8":             {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: sliceU8.Name, ReturnSlots: sliceU8.SlotCount, ReturnRegionParam: regionNone},
		"core.make_i32":            {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: sliceI32.Name, ReturnSlots: sliceI32.SlotCount, ReturnRegionParam: regionNone},
		"core.island_new":          {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "island", ReturnSlots: islandInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.island_make_u8":      {ParamTypes: []string{"island", "i32"}, ParamSlots: 2, ReturnType: sliceU8.Name, ReturnSlots: sliceU8.SlotCount, ReturnRegionParam: 0},
		"core.island_make_i32":     {ParamTypes: []string{"island", "i32"}, ParamSlots: 2, ReturnType: sliceI32.Name, ReturnSlots: sliceI32.SlotCount, ReturnRegionParam: 0},
		"core.cap_io":              {ParamTypes: nil, ParamSlots: 0, ReturnType: capIO.Name, ReturnSlots: capIO.SlotCount, ReturnRegionParam: regionNone},
		"core.cap_mem":             {ParamTypes: nil, ParamSlots: 0, ReturnType: capMem.Name, ReturnSlots: capMem.SlotCount, ReturnRegionParam: regionNone},
		"core.load_i32":            {ParamTypes: []string{"ptr", capMem.Name}, ParamSlots: 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.store_i32":           {ParamTypes: []string{"ptr", "i32", capMem.Name}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.load_u8":             {ParamTypes: []string{"ptr", capMem.Name}, ParamSlots: 2, ReturnType: "u8", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.store_u8":            {ParamTypes: []string{"ptr", "u8", capMem.Name}, ParamSlots: 3, ReturnType: "u8", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.load_ptr":            {ParamTypes: []string{"ptr", capMem.Name}, ParamSlots: 2, ReturnType: "ptr", ReturnSlots: ptrInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.store_ptr":           {ParamTypes: []string{"ptr", "ptr", capMem.Name}, ParamSlots: 3, ReturnType: "ptr", ReturnSlots: ptrInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.ptr_add":             {ParamTypes: []string{"ptr", "i32", capMem.Name}, ParamSlots: 3, ReturnType: "ptr", ReturnSlots: ptrInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.mmio_read_i32":       {ParamTypes: []string{"ptr", capIO.Name}, ParamSlots: 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.mmio_write_i32":      {ParamTypes: []string{"ptr", "i32", capIO.Name}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.sym_addr":            {ParamTypes: []string{"str"}, ParamSlots: 2, ReturnType: "ptr", ReturnSlots: ptrInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.ctx_switch":          {ParamTypes: []string{"ptr", "ptr", capMem.Name}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.task_spawn_i32":      {ParamTypes: []string{"str"}, ParamSlots: 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.task_join_i32":       {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.actor_dispatch":      {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.actor_main_entry_id": {ParamTypes: nil, ParamSlots: 0, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.spawn":               {ParamTypes: []string{"str"}, ParamSlots: 2, ReturnType: actorInfo.Name, ReturnSlots: actorInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.send":                {ParamTypes: []string{"actor", "i32"}, ParamSlots: 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.recv":                {ParamTypes: nil, ParamSlots: 0, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.self":                {ParamTypes: nil, ParamSlots: 0, ReturnType: actorInfo.Name, ReturnSlots: actorInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.sender":              {ParamTypes: nil, ParamSlots: 0, ReturnType: actorInfo.Name, ReturnSlots: actorInfo.SlotCount, ReturnRegionParam: regionNone},
	}
	for name, sig := range sigs {
		sig.Effects = builtinEffects(name)
		sigs[name] = sig
	}
	return sigs, nil
}

func builtinNeedsUnsafe(name string, argRegions []int) bool {
	switch name {
	case "core.alloc_bytes", "core.island_new", "core.cap_io", "core.cap_mem",
		"core.load_i32", "core.store_i32",
		"core.load_u8", "core.store_u8",
		"core.load_ptr", "core.store_ptr",
		"core.ptr_add",
		"core.mmio_read_i32", "core.mmio_write_i32",
		"core.sym_addr", "core.ctx_switch":
		return true
	case "core.island_make_u8", "core.island_make_i32":
		if len(argRegions) == 0 {
			return true
		}
		return argRegions[0] == regionNone
	default:
		return false
	}
}

func builtinCapsulePermission(name string) (permission string, attenuatedEffect string) {
	switch name {
	case "core.cap_io", "core.mmio_read_i32", "core.mmio_write_i32":
		return "capsule.io", "io"
	case "core.cap_mem",
		"core.load_i32", "core.store_i32",
		"core.load_u8", "core.store_u8",
		"core.load_ptr", "core.store_ptr",
		"core.ptr_add", "core.ctx_switch":
		return "capsule.mem", "mem"
	default:
		return "", ""
	}
}

func ResolveBuiltinAlias(name string) (string, bool) {
	switch name {
	case "alloc_bytes":
		return "core.alloc_bytes", true
	case "make_u8":
		return "core.make_u8", true
	case "make_i32":
		return "core.make_i32", true
	case "island_new":
		return "core.island_new", true
	case "island_make_u8":
		return "core.island_make_u8", true
	case "island_make_i32":
		return "core.island_make_i32", true
	case "load_ptr":
		return "core.load_ptr", true
	case "store_ptr":
		return "core.store_ptr", true
	case "sym_addr":
		return "core.sym_addr", true
	case "ctx_switch":
		return "core.ctx_switch", true
	case "task_spawn_i32":
		return "core.task_spawn_i32", true
	case "task_join_i32":
		return "core.task_join_i32", true
	case "actor_dispatch":
		return "core.actor_dispatch", true
	case "actor_main_entry_id":
		return "core.actor_main_entry_id", true
	case "core.alloc_bytes", "core.make_u8", "core.make_i32",
		"core.island_new", "core.island_make_u8", "core.island_make_i32",
		"core.load_ptr", "core.store_ptr", "core.sym_addr", "core.ctx_switch",
		"core.task_spawn_i32", "core.task_join_i32",
		"core.actor_dispatch", "core.actor_main_entry_id":
		return name, true
	default:
		return "", false
	}
}
