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
	_, err = ensureTypeInfo("[]u16", types)
	if err != nil {
		return nil, err
	}
	_, err = ensureTypeInfo("[]bool", types)
	if err != nil {
		return nil, err
	}
	actorInfo, err := ensureTypeInfo("actor", types)
	if err != nil {
		return nil, err
	}
	actorMsgInfo, err := ensureTypeInfo("actor.msg", types)
	if err != nil {
		return nil, err
	}
	actorRecvResultI32, err := ensureTypeInfo("actor.recv_result_i32", types)
	if err != nil {
		return nil, err
	}
	actorRecvMsgResult, err := ensureTypeInfo("actor.recv_msg_result", types)
	if err != nil {
		return nil, err
	}
	ptrInfo, err := ensureTypeInfo("ptr", types)
	if err != nil {
		return nil, err
	}
	taskGroupInfo, err := ensureTypeInfo("task.group", types)
	if err != nil {
		return nil, err
	}
	taskErrorInfo, err := ensureTypeInfo("task.error", types)
	if err != nil {
		return nil, err
	}
	taskHandleI32, err := ensureTypeInfo("task.i32", types)
	if err != nil {
		return nil, err
	}
	taskResultI32, err := ensureTypeInfo("task.result_i32", types)
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
	sliceU16, err := ensureTypeInfo("[]u16", types)
	if err != nil {
		return nil, err
	}
	sliceBool, err := ensureTypeInfo("[]bool", types)
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
	consentToken, err := ensureTypeInfo("consent.token", types)
	if err != nil {
		return nil, err
	}
	secretI32, err := ensureTypeInfo("secret.i32", types)
	if err != nil {
		return nil, err
	}

	sigs := map[string]FuncSig{
		"core.alloc_bytes":                {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "ptr", ReturnSlots: ptrInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.make_u8":                    {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: sliceU8.Name, ReturnSlots: sliceU8.SlotCount, ReturnRegionParam: regionNone},
		"core.make_u16":                   {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: sliceU16.Name, ReturnSlots: sliceU16.SlotCount, ReturnRegionParam: regionNone},
		"core.make_i32":                   {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: sliceI32.Name, ReturnSlots: sliceI32.SlotCount, ReturnRegionParam: regionNone},
		"core.make_bool":                  {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: sliceBool.Name, ReturnSlots: sliceBool.SlotCount, ReturnRegionParam: regionNone},
		"core.island_new":                 {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "island", ReturnSlots: islandInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.island_make_u8":             {ParamTypes: []string{"island", "i32"}, ParamSlots: 2, ReturnType: sliceU8.Name, ReturnSlots: sliceU8.SlotCount, ReturnRegionParam: 0},
		"core.island_make_u16":            {ParamTypes: []string{"island", "i32"}, ParamSlots: 2, ReturnType: sliceU16.Name, ReturnSlots: sliceU16.SlotCount, ReturnRegionParam: 0},
		"core.island_make_i32":            {ParamTypes: []string{"island", "i32"}, ParamSlots: 2, ReturnType: sliceI32.Name, ReturnSlots: sliceI32.SlotCount, ReturnRegionParam: 0},
		"core.island_make_bool":           {ParamTypes: []string{"island", "i32"}, ParamSlots: 2, ReturnType: sliceBool.Name, ReturnSlots: sliceBool.SlotCount, ReturnRegionParam: 0},
		"core.cap_io":                     {ParamTypes: nil, ParamSlots: 0, ReturnType: capIO.Name, ReturnSlots: capIO.SlotCount, ReturnRegionParam: regionNone},
		"core.cap_mem":                    {ParamTypes: nil, ParamSlots: 0, ReturnType: capMem.Name, ReturnSlots: capMem.SlotCount, ReturnRegionParam: regionNone},
		"core.load_i32":                   {ParamTypes: []string{"ptr", capMem.Name}, ParamSlots: 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.store_i32":                  {ParamTypes: []string{"ptr", "i32", capMem.Name}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.load_u8":                    {ParamTypes: []string{"ptr", capMem.Name}, ParamSlots: 2, ReturnType: "u8", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.store_u8":                   {ParamTypes: []string{"ptr", "u8", capMem.Name}, ParamSlots: 3, ReturnType: "u8", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.load_ptr":                   {ParamTypes: []string{"ptr", capMem.Name}, ParamSlots: 2, ReturnType: "ptr", ReturnSlots: ptrInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.store_ptr":                  {ParamTypes: []string{"ptr", "ptr", capMem.Name}, ParamSlots: 3, ReturnType: "ptr", ReturnSlots: ptrInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.ptr_add":                    {ParamTypes: []string{"ptr", "i32", capMem.Name}, ParamSlots: 3, ReturnType: "ptr", ReturnSlots: ptrInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.mmio_read_i32":              {ParamTypes: []string{"ptr", capIO.Name}, ParamSlots: 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.mmio_write_i32":             {ParamTypes: []string{"ptr", "i32", capIO.Name}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.sym_addr":                   {ParamTypes: []string{"str"}, ParamSlots: 2, ReturnType: "ptr", ReturnSlots: ptrInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.ctx_switch":                 {ParamTypes: []string{"ptr", "ptr", capMem.Name}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.time_now_ms":                {ParamTypes: nil, ParamSlots: 0, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.sleep_ms":                   {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.sleep_until":                {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.deadline_ms":                {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.timer_ready":                {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "bool", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.yield":                      {ParamTypes: nil, ParamSlots: 0, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.task_group_open":            {ParamTypes: nil, ParamSlots: 0, ReturnType: taskGroupInfo.Name, ReturnSlots: taskGroupInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.task_group_close":           {ParamTypes: []string{"task.group"}, ParamSlots: taskGroupInfo.SlotCount, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.task_group_cancel":          {ParamTypes: []string{"task.group"}, ParamSlots: taskGroupInfo.SlotCount, ReturnType: "task.group", ReturnSlots: taskGroupInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.task_group_current":         {ParamTypes: nil, ParamSlots: 0, ReturnType: taskGroupInfo.Name, ReturnSlots: taskGroupInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.task_group_status":          {ParamTypes: []string{"task.group"}, ParamSlots: taskGroupInfo.SlotCount, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.task_is_canceled":           {ParamTypes: nil, ParamSlots: 0, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.task_checkpoint":            {ParamTypes: nil, ParamSlots: 0, ReturnType: taskErrorInfo.Name, ReturnSlots: taskErrorInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.task_spawn_i32":             {ParamTypes: []string{"str"}, ParamSlots: 2, ReturnType: taskHandleI32.Name, ReturnSlots: taskHandleI32.SlotCount, ReturnRegionParam: regionNone},
		"core.task_spawn_i32_typed":       {ParamTypes: []string{"str"}, ParamSlots: 2, ReturnType: taskHandleI32.Name, ReturnSlots: taskHandleI32.SlotCount, ReturnRegionParam: regionNone},
		"core.task_spawn_group_i32":       {ParamTypes: []string{"task.group", "str"}, ParamSlots: taskGroupInfo.SlotCount + 2, ReturnType: taskHandleI32.Name, ReturnSlots: taskHandleI32.SlotCount, ReturnRegionParam: regionNone},
		"core.task_spawn_group_i32_typed": {ParamTypes: []string{"task.group", "str"}, ParamSlots: taskGroupInfo.SlotCount + 2, ReturnType: taskHandleI32.Name, ReturnSlots: taskHandleI32.SlotCount, ReturnRegionParam: regionNone},
		"core.task_join_i32":              {ParamTypes: []string{"task.i32"}, ParamSlots: taskHandleI32.SlotCount, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.task_join_i32_typed":        {ParamTypes: []string{"task.i32"}, ParamSlots: taskHandleI32.SlotCount, ReturnType: "i32", ThrowsType: "enum", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.task_join_group_i32_typed":  {ParamTypes: []string{"task.i32"}, ParamSlots: taskHandleI32.SlotCount, ReturnType: "i32", ThrowsType: "enum", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.task_join_result_i32":       {ParamTypes: []string{"task.i32"}, ParamSlots: taskHandleI32.SlotCount, ReturnType: taskResultI32.Name, ReturnSlots: taskResultI32.SlotCount, ReturnRegionParam: regionNone},
		"core.task_join_until_i32":        {ParamTypes: []string{"task.i32", "i32"}, ParamSlots: taskHandleI32.SlotCount + 1, ReturnType: taskResultI32.Name, ReturnSlots: taskResultI32.SlotCount, ReturnRegionParam: regionNone},
		"core.task_poll_i32":              {ParamTypes: []string{"task.i32"}, ParamSlots: taskHandleI32.SlotCount, ReturnType: taskResultI32.Name, ReturnSlots: taskResultI32.SlotCount, ReturnRegionParam: regionNone},
		"core.select2_i32":                {ParamTypes: []string{"task.i32", "i32"}, ParamSlots: taskHandleI32.SlotCount + 1, ReturnType: taskResultI32.Name, ReturnSlots: taskResultI32.SlotCount, ReturnRegionParam: regionNone},
		"core.actor_dispatch":             {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.actor_main_entry_id":        {ParamTypes: nil, ParamSlots: 0, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.spawn":                      {ParamTypes: []string{"str"}, ParamSlots: 2, ReturnType: actorInfo.Name, ReturnSlots: actorInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.send":                       {ParamTypes: []string{"actor", "i32"}, ParamSlots: 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.send_msg":                   {ParamTypes: []string{"actor", "i32", "i32"}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.send_typed":                 {ParamTypes: []string{"actor", "enum"}, ParamSlots: 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.recv":                       {ParamTypes: nil, ParamSlots: 0, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.recv_msg":                   {ParamTypes: nil, ParamSlots: 0, ReturnType: actorMsgInfo.Name, ReturnSlots: actorMsgInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.recv_poll":                  {ParamTypes: nil, ParamSlots: 0, ReturnType: actorRecvResultI32.Name, ReturnSlots: actorRecvResultI32.SlotCount, ReturnRegionParam: regionNone},
		"core.recv_until":                 {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: actorRecvResultI32.Name, ReturnSlots: actorRecvResultI32.SlotCount, ReturnRegionParam: regionNone},
		"core.recv_msg_until":             {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: actorRecvMsgResult.Name, ReturnSlots: actorRecvMsgResult.SlotCount, ReturnRegionParam: regionNone},
		"core.recv_typed":                 {ParamTypes: nil, ParamSlots: 0, ReturnType: "enum", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.self":                       {ParamTypes: nil, ParamSlots: 0, ReturnType: actorInfo.Name, ReturnSlots: actorInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.sender":                     {ParamTypes: nil, ParamSlots: 0, ReturnType: actorInfo.Name, ReturnSlots: actorInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.consent_token":              {ParamTypes: nil, ParamSlots: 0, ReturnType: consentToken.Name, ReturnSlots: consentToken.SlotCount, ReturnRegionParam: regionNone},
		"core.secret_seal_i32":            {ParamTypes: []string{"i32", consentToken.Name}, ParamSlots: 2, ReturnType: secretI32.Name, ReturnSlots: secretI32.SlotCount, ReturnRegionParam: regionNone},
		"core.secret_unseal_i32":          {ParamTypes: []string{secretI32.Name, consentToken.Name}, ParamSlots: 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
	}
	for name, sig := range sigs {
		sig.ReturnResourceParam = regionNone
		if name == "core.task_group_cancel" {
			sig.ReturnResourceParam = 0
		}
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
	case "core.island_make_u8", "core.island_make_u16", "core.island_make_i32", "core.island_make_bool":
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
	case "make_u16":
		return "core.make_u16", true
	case "make_i32":
		return "core.make_i32", true
	case "make_bool":
		return "core.make_bool", true
	case "island_new":
		return "core.island_new", true
	case "island_make_u8":
		return "core.island_make_u8", true
	case "island_make_u16":
		return "core.island_make_u16", true
	case "island_make_i32":
		return "core.island_make_i32", true
	case "island_make_bool":
		return "core.island_make_bool", true
	case "load_ptr":
		return "core.load_ptr", true
	case "store_ptr":
		return "core.store_ptr", true
	case "sym_addr":
		return "core.sym_addr", true
	case "ctx_switch":
		return "core.ctx_switch", true
	case "time_now_ms":
		return "core.time_now_ms", true
	case "sleep_ms":
		return "core.sleep_ms", true
	case "deadline_ms":
		return "core.deadline_ms", true
	case "sleep_until":
		return "core.sleep_until", true
	case "timer_ready":
		return "core.timer_ready", true
	case "yield":
		return "core.yield", true
	case "task_spawn_i32":
		return "core.task_spawn_i32", true
	case "task_spawn_i32_typed":
		return "core.task_spawn_i32_typed", true
	case "task_spawn_group_i32":
		return "core.task_spawn_group_i32", true
	case "task_spawn_group_i32_typed":
		return "core.task_spawn_group_i32_typed", true
	case "task_join_i32":
		return "core.task_join_i32", true
	case "task_join_i32_typed":
		return "core.task_join_i32_typed", true
	case "task_join_group_i32_typed":
		return "core.task_join_group_i32_typed", true
	case "task_join_result_i32":
		return "core.task_join_result_i32", true
	case "task_join_until_i32":
		return "core.task_join_until_i32", true
	case "task_poll_i32":
		return "core.task_poll_i32", true
	case "select2_i32":
		return "core.select2_i32", true
	case "task_group_open":
		return "core.task_group_open", true
	case "task_group_close":
		return "core.task_group_close", true
	case "task_group_cancel":
		return "core.task_group_cancel", true
	case "task_group_current":
		return "core.task_group_current", true
	case "task_group_status":
		return "core.task_group_status", true
	case "task_is_canceled":
		return "core.task_is_canceled", true
	case "task_checkpoint":
		return "core.task_checkpoint", true
	case "send_msg":
		return "core.send_msg", true
	case "recv_msg":
		return "core.recv_msg", true
	case "recv_poll":
		return "core.recv_poll", true
	case "recv_until":
		return "core.recv_until", true
	case "recv_msg_until":
		return "core.recv_msg_until", true
	case "send_typed":
		return "core.send_typed", true
	case "recv_typed":
		return "core.recv_typed", true
	case "actor_dispatch":
		return "core.actor_dispatch", true
	case "actor_main_entry_id":
		return "core.actor_main_entry_id", true
	case "consent_token":
		return "core.consent_token", true
	case "secret_seal_i32":
		return "core.secret_seal_i32", true
	case "secret_unseal_i32":
		return "core.secret_unseal_i32", true
	case "core.alloc_bytes", "core.make_u8", "core.make_u16", "core.make_i32", "core.make_bool",
		"core.island_new", "core.island_make_u8", "core.island_make_u16", "core.island_make_i32", "core.island_make_bool",
		"core.load_ptr", "core.store_ptr", "core.sym_addr", "core.ctx_switch",
		"core.time_now_ms", "core.sleep_ms", "core.sleep_until", "core.deadline_ms", "core.timer_ready", "core.yield",
		"core.task_group_open", "core.task_group_close", "core.task_group_cancel", "core.task_group_current", "core.task_group_status",
		"core.task_is_canceled", "core.task_checkpoint",
		"core.task_spawn_i32", "core.task_spawn_i32_typed", "core.task_spawn_group_i32", "core.task_spawn_group_i32_typed",
		"core.task_join_i32", "core.task_join_i32_typed", "core.task_join_group_i32_typed", "core.task_join_result_i32", "core.task_join_until_i32",
		"core.task_poll_i32", "core.select2_i32",
		"core.send_msg", "core.recv_msg", "core.recv_poll", "core.recv_until", "core.recv_msg_until", "core.send_typed", "core.recv_typed",
		"core.actor_dispatch", "core.actor_main_entry_id",
		"core.consent_token", "core.secret_seal_i32", "core.secret_unseal_i32":
		return name, true
	default:
		return "", false
	}
}
