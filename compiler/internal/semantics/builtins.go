package semantics

import "strings"

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
	strInfo, err := ensureTypeInfo("str", types)
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
		"core.alloc_bytes":                      {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "ptr", ReturnSlots: ptrInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.make_u8":                          {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: sliceU8.Name, ReturnSlots: sliceU8.SlotCount, ReturnRegionParam: regionNone},
		"core.make_u16":                         {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: sliceU16.Name, ReturnSlots: sliceU16.SlotCount, ReturnRegionParam: regionNone},
		"core.make_i32":                         {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: sliceI32.Name, ReturnSlots: sliceI32.SlotCount, ReturnRegionParam: regionNone},
		"core.make_bool":                        {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: sliceBool.Name, ReturnSlots: sliceBool.SlotCount, ReturnRegionParam: regionNone},
		"core.raw_slice_u8_from_parts":          {ParamTypes: []string{"ptr", "i32", capMem.Name}, ParamSlots: 3, ReturnType: sliceU8.Name, ReturnSlots: sliceU8.SlotCount, ReturnRegionParam: regionNone},
		"core.raw_slice_u16_from_parts":         {ParamTypes: []string{"ptr", "i32", capMem.Name}, ParamSlots: 3, ReturnType: sliceU16.Name, ReturnSlots: sliceU16.SlotCount, ReturnRegionParam: regionNone},
		"core.raw_slice_i32_from_parts":         {ParamTypes: []string{"ptr", "i32", capMem.Name}, ParamSlots: 3, ReturnType: sliceI32.Name, ReturnSlots: sliceI32.SlotCount, ReturnRegionParam: regionNone},
		"core.raw_slice_bool_from_parts":        {ParamTypes: []string{"ptr", "i32", capMem.Name}, ParamSlots: 3, ReturnType: sliceBool.Name, ReturnSlots: sliceBool.SlotCount, ReturnRegionParam: regionNone},
		"core.slice_window_u8":                  {ParamTypes: []string{sliceU8.Name, "i32", "i32"}, ParamOwnership: []string{"borrow"}, ParamSlots: sliceU8.SlotCount + 2, ReturnType: sliceU8.Name, ReturnSlots: sliceU8.SlotCount, ReturnRegionParam: 0},
		"core.slice_window_u16":                 {ParamTypes: []string{sliceU16.Name, "i32", "i32"}, ParamOwnership: []string{"borrow"}, ParamSlots: sliceU16.SlotCount + 2, ReturnType: sliceU16.Name, ReturnSlots: sliceU16.SlotCount, ReturnRegionParam: 0},
		"core.slice_window_i32":                 {ParamTypes: []string{sliceI32.Name, "i32", "i32"}, ParamOwnership: []string{"borrow"}, ParamSlots: sliceI32.SlotCount + 2, ReturnType: sliceI32.Name, ReturnSlots: sliceI32.SlotCount, ReturnRegionParam: 0},
		"core.slice_window_bool":                {ParamTypes: []string{sliceBool.Name, "i32", "i32"}, ParamOwnership: []string{"borrow"}, ParamSlots: sliceBool.SlotCount + 2, ReturnType: sliceBool.Name, ReturnSlots: sliceBool.SlotCount, ReturnRegionParam: 0},
		"core.slice_prefix_u8":                  {ParamTypes: []string{sliceU8.Name, "i32"}, ParamOwnership: []string{"borrow"}, ParamSlots: sliceU8.SlotCount + 1, ReturnType: sliceU8.Name, ReturnSlots: sliceU8.SlotCount, ReturnRegionParam: 0},
		"core.slice_prefix_u16":                 {ParamTypes: []string{sliceU16.Name, "i32"}, ParamOwnership: []string{"borrow"}, ParamSlots: sliceU16.SlotCount + 1, ReturnType: sliceU16.Name, ReturnSlots: sliceU16.SlotCount, ReturnRegionParam: 0},
		"core.slice_prefix_i32":                 {ParamTypes: []string{sliceI32.Name, "i32"}, ParamOwnership: []string{"borrow"}, ParamSlots: sliceI32.SlotCount + 1, ReturnType: sliceI32.Name, ReturnSlots: sliceI32.SlotCount, ReturnRegionParam: 0},
		"core.slice_prefix_bool":                {ParamTypes: []string{sliceBool.Name, "i32"}, ParamOwnership: []string{"borrow"}, ParamSlots: sliceBool.SlotCount + 1, ReturnType: sliceBool.Name, ReturnSlots: sliceBool.SlotCount, ReturnRegionParam: 0},
		"core.slice_suffix_u8":                  {ParamTypes: []string{sliceU8.Name, "i32"}, ParamOwnership: []string{"borrow"}, ParamSlots: sliceU8.SlotCount + 1, ReturnType: sliceU8.Name, ReturnSlots: sliceU8.SlotCount, ReturnRegionParam: 0},
		"core.slice_suffix_u16":                 {ParamTypes: []string{sliceU16.Name, "i32"}, ParamOwnership: []string{"borrow"}, ParamSlots: sliceU16.SlotCount + 1, ReturnType: sliceU16.Name, ReturnSlots: sliceU16.SlotCount, ReturnRegionParam: 0},
		"core.slice_suffix_i32":                 {ParamTypes: []string{sliceI32.Name, "i32"}, ParamOwnership: []string{"borrow"}, ParamSlots: sliceI32.SlotCount + 1, ReturnType: sliceI32.Name, ReturnSlots: sliceI32.SlotCount, ReturnRegionParam: 0},
		"core.slice_suffix_bool":                {ParamTypes: []string{sliceBool.Name, "i32"}, ParamOwnership: []string{"borrow"}, ParamSlots: sliceBool.SlotCount + 1, ReturnType: sliceBool.Name, ReturnSlots: sliceBool.SlotCount, ReturnRegionParam: 0},
		"core.slice_borrow_u8":                  {ParamTypes: []string{sliceU8.Name}, ParamOwnership: []string{"borrow"}, ParamSlots: sliceU8.SlotCount, ReturnType: sliceU8.Name, ReturnSlots: sliceU8.SlotCount, ReturnRegionParam: 0},
		"core.slice_borrow_u16":                 {ParamTypes: []string{sliceU16.Name}, ParamOwnership: []string{"borrow"}, ParamSlots: sliceU16.SlotCount, ReturnType: sliceU16.Name, ReturnSlots: sliceU16.SlotCount, ReturnRegionParam: 0},
		"core.slice_borrow_i32":                 {ParamTypes: []string{sliceI32.Name}, ParamOwnership: []string{"borrow"}, ParamSlots: sliceI32.SlotCount, ReturnType: sliceI32.Name, ReturnSlots: sliceI32.SlotCount, ReturnRegionParam: 0},
		"core.slice_borrow_bool":                {ParamTypes: []string{sliceBool.Name}, ParamOwnership: []string{"borrow"}, ParamSlots: sliceBool.SlotCount, ReturnType: sliceBool.Name, ReturnSlots: sliceBool.SlotCount, ReturnRegionParam: 0},
		"core.slice_copy_u8":                    {ParamTypes: []string{sliceU8.Name}, ParamOwnership: []string{"borrow"}, ParamSlots: sliceU8.SlotCount, ReturnType: sliceU8.Name, ReturnSlots: sliceU8.SlotCount, ReturnRegionParam: regionNone},
		"core.slice_copy_u16":                   {ParamTypes: []string{sliceU16.Name}, ParamOwnership: []string{"borrow"}, ParamSlots: sliceU16.SlotCount, ReturnType: sliceU16.Name, ReturnSlots: sliceU16.SlotCount, ReturnRegionParam: regionNone},
		"core.slice_copy_i32":                   {ParamTypes: []string{sliceI32.Name}, ParamOwnership: []string{"borrow"}, ParamSlots: sliceI32.SlotCount, ReturnType: sliceI32.Name, ReturnSlots: sliceI32.SlotCount, ReturnRegionParam: regionNone},
		"core.slice_copy_bool":                  {ParamTypes: []string{sliceBool.Name}, ParamOwnership: []string{"borrow"}, ParamSlots: sliceBool.SlotCount, ReturnType: sliceBool.Name, ReturnSlots: sliceBool.SlotCount, ReturnRegionParam: regionNone},
		"core.slice_copy_into_u8":               {ParamTypes: []string{sliceU8.Name, sliceU8.Name}, ParamOwnership: []string{"borrow", "inout"}, ParamSlots: sliceU8.SlotCount * 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.slice_copy_into_u16":              {ParamTypes: []string{sliceU16.Name, sliceU16.Name}, ParamOwnership: []string{"borrow", "inout"}, ParamSlots: sliceU16.SlotCount * 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.slice_copy_into_i32":              {ParamTypes: []string{sliceI32.Name, sliceI32.Name}, ParamOwnership: []string{"borrow", "inout"}, ParamSlots: sliceI32.SlotCount * 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.slice_copy_into_bool":             {ParamTypes: []string{sliceBool.Name, sliceBool.Name}, ParamOwnership: []string{"borrow", "inout"}, ParamSlots: sliceBool.SlotCount * 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.string_window":                    {ParamTypes: []string{strInfo.Name, "i32", "i32"}, ParamOwnership: []string{"borrow"}, ParamSlots: strInfo.SlotCount + 2, ReturnType: strInfo.Name, ReturnSlots: strInfo.SlotCount, ReturnRegionParam: 0},
		"core.string_prefix":                    {ParamTypes: []string{strInfo.Name, "i32"}, ParamOwnership: []string{"borrow"}, ParamSlots: strInfo.SlotCount + 1, ReturnType: strInfo.Name, ReturnSlots: strInfo.SlotCount, ReturnRegionParam: 0},
		"core.string_suffix":                    {ParamTypes: []string{strInfo.Name, "i32"}, ParamOwnership: []string{"borrow"}, ParamSlots: strInfo.SlotCount + 1, ReturnType: strInfo.Name, ReturnSlots: strInfo.SlotCount, ReturnRegionParam: 0},
		"core.string_borrow":                    {ParamTypes: []string{strInfo.Name}, ParamOwnership: []string{"borrow"}, ParamSlots: strInfo.SlotCount, ReturnType: strInfo.Name, ReturnSlots: strInfo.SlotCount, ReturnRegionParam: 0},
		"core.string_copy":                      {ParamTypes: []string{strInfo.Name}, ParamOwnership: []string{"borrow"}, ParamSlots: strInfo.SlotCount, ReturnType: strInfo.Name, ReturnSlots: strInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.string_copy_into":                 {ParamTypes: []string{strInfo.Name, sliceU8.Name}, ParamOwnership: []string{"borrow", "inout"}, ParamSlots: strInfo.SlotCount + sliceU8.SlotCount, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.island_new":                       {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "island", ReturnSlots: islandInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.island_make_u8":                   {ParamTypes: []string{"island", "i32"}, ParamSlots: 2, ReturnType: sliceU8.Name, ReturnSlots: sliceU8.SlotCount, ReturnRegionParam: 0},
		"core.island_make_u16":                  {ParamTypes: []string{"island", "i32"}, ParamSlots: 2, ReturnType: sliceU16.Name, ReturnSlots: sliceU16.SlotCount, ReturnRegionParam: 0},
		"core.island_make_i32":                  {ParamTypes: []string{"island", "i32"}, ParamSlots: 2, ReturnType: sliceI32.Name, ReturnSlots: sliceI32.SlotCount, ReturnRegionParam: 0},
		"core.island_make_bool":                 {ParamTypes: []string{"island", "i32"}, ParamSlots: 2, ReturnType: sliceBool.Name, ReturnSlots: sliceBool.SlotCount, ReturnRegionParam: 0},
		"core.island_reset":                     {ParamTypes: []string{"island"}, ParamOwnership: []string{"consume"}, ParamSlots: 1, ReturnType: "island", ReturnSlots: islandInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.cap_io":                           {ParamTypes: nil, ParamSlots: 0, ReturnType: capIO.Name, ReturnSlots: capIO.SlotCount, ReturnRegionParam: regionNone},
		"core.cap_mem":                          {ParamTypes: nil, ParamSlots: 0, ReturnType: capMem.Name, ReturnSlots: capMem.SlotCount, ReturnRegionParam: regionNone},
		"core.load_i32":                         {ParamTypes: []string{"ptr", capMem.Name}, ParamSlots: 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.store_i32":                        {ParamTypes: []string{"ptr", "i32", capMem.Name}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.load_u8":                          {ParamTypes: []string{"ptr", capMem.Name}, ParamSlots: 2, ReturnType: "u8", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.store_u8":                         {ParamTypes: []string{"ptr", "u8", capMem.Name}, ParamSlots: 3, ReturnType: "u8", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.load_ptr":                         {ParamTypes: []string{"ptr", capMem.Name}, ParamSlots: 2, ReturnType: "ptr", ReturnSlots: ptrInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.store_ptr":                        {ParamTypes: []string{"ptr", "ptr", capMem.Name}, ParamSlots: 3, ReturnType: "ptr", ReturnSlots: ptrInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.store_arch_ptr":                   {ParamTypes: []string{"ptr", "ptr", capMem.Name}, ParamSlots: 3, ReturnType: "ptr", ReturnSlots: ptrInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.ptr_add":                          {ParamTypes: []string{"ptr", "i32", capMem.Name}, ParamSlots: 3, ReturnType: "ptr", ReturnSlots: ptrInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.mmio_read_i32":                    {ParamTypes: []string{"ptr", capIO.Name}, ParamSlots: 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.mmio_write_i32":                   {ParamTypes: []string{"ptr", "i32", capIO.Name}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.fs_exists":                        {ParamTypes: []string{"str", capIO.Name}, ParamSlots: 3, ReturnType: "bool", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.net_socket_tcp4":                  {ParamTypes: []string{capIO.Name}, ParamSlots: 1, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.net_bind_tcp4_loopback":           {ParamTypes: []string{"i32", "i32", capIO.Name}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.net_connect_tcp4_loopback":        {ParamTypes: []string{"i32", "i32", capIO.Name}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.net_listen":                       {ParamTypes: []string{"i32", "i32", capIO.Name}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.net_accept4":                      {ParamTypes: []string{"i32", "i32", capIO.Name}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.net_read":                         {ParamTypes: []string{"i32", sliceU8.Name, "i32", "i32", capIO.Name}, ParamSlots: 6, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.net_recv":                         {ParamTypes: []string{"i32", sliceU8.Name, "i32", "i32", capIO.Name}, ParamSlots: 6, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.net_write":                        {ParamTypes: []string{"i32", sliceU8.Name, "i32", "i32", capIO.Name}, ParamSlots: 6, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.net_send":                         {ParamTypes: []string{"i32", sliceU8.Name, "i32", "i32", capIO.Name}, ParamSlots: 6, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.net_epoll_create":                 {ParamTypes: []string{capIO.Name}, ParamSlots: 1, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.net_epoll_ctl_add_read":           {ParamTypes: []string{"i32", "i32", capIO.Name}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.net_epoll_ctl_add_read_write":     {ParamTypes: []string{"i32", "i32", capIO.Name}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.net_epoll_ctl_mod_read":           {ParamTypes: []string{"i32", "i32", capIO.Name}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.net_epoll_ctl_mod_read_write":     {ParamTypes: []string{"i32", "i32", capIO.Name}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.net_epoll_ctl_delete":             {ParamTypes: []string{"i32", "i32", capIO.Name}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.net_epoll_wait_one":               {ParamTypes: []string{"i32", "i32", capIO.Name}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.net_epoll_wait_one_into":          {ParamTypes: []string{"i32", sliceI32.Name, "i32", capIO.Name}, ParamSlots: 5, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.net_set_nonblocking":              {ParamTypes: []string{"i32", capIO.Name}, ParamSlots: 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.net_set_reuseport":                {ParamTypes: []string{"i32", capIO.Name}, ParamSlots: 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.net_set_tcp_nodelay":              {ParamTypes: []string{"i32", capIO.Name}, ParamSlots: 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.net_close":                        {ParamTypes: []string{"i32", capIO.Name}, ParamSlots: 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.sym_addr":                         {ParamTypes: []string{"str"}, ParamSlots: 2, ReturnType: "ptr", ReturnSlots: ptrInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.ctx_switch":                       {ParamTypes: []string{"ptr", "ptr", capMem.Name}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.surface_open":                     {ParamTypes: []string{"str", "i32", "i32"}, ParamSlots: 4, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.surface_close":                    {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.surface_poll_event_kind":          {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.surface_poll_event_x":             {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.surface_poll_event_y":             {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.surface_poll_event_button":        {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.surface_poll_event_into":          {ParamTypes: []string{"i32", sliceI32.Name}, ParamSlots: 1 + sliceI32.SlotCount, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.surface_poll_event_text_len":      {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.surface_poll_event_text_into":     {ParamTypes: []string{"i32", sliceU8.Name}, ParamSlots: 1 + sliceU8.SlotCount, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.surface_clipboard_write_text":     {ParamTypes: []string{"i32", sliceU8.Name}, ParamSlots: 1 + sliceU8.SlotCount, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.surface_clipboard_read_text_into": {ParamTypes: []string{"i32", sliceU8.Name}, ParamSlots: 1 + sliceU8.SlotCount, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.surface_poll_composition_into":    {ParamTypes: []string{"i32", sliceI32.Name}, ParamSlots: 1 + sliceI32.SlotCount, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.surface_begin_frame":              {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.surface_present_rgba":             {ParamTypes: []string{"i32", sliceU8.Name, "i32", "i32", "i32"}, ParamSlots: 1 + sliceU8.SlotCount + 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.surface_now_ms":                   {ParamTypes: nil, ParamSlots: 0, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.surface_request_redraw":           {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.time_now_ms":                      {ParamTypes: nil, ParamSlots: 0, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.sleep_ms":                         {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.sleep_until":                      {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.deadline_ms":                      {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.timer_ready":                      {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "bool", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.yield":                            {ParamTypes: nil, ParamSlots: 0, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.task_group_open":                  {ParamTypes: nil, ParamSlots: 0, ReturnType: taskGroupInfo.Name, ReturnSlots: taskGroupInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.task_group_close":                 {ParamTypes: []string{"task.group"}, ParamSlots: taskGroupInfo.SlotCount, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.task_group_cancel":                {ParamTypes: []string{"task.group"}, ParamSlots: taskGroupInfo.SlotCount, ReturnType: "task.group", ReturnSlots: taskGroupInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.task_group_current":               {ParamTypes: nil, ParamSlots: 0, ReturnType: taskGroupInfo.Name, ReturnSlots: taskGroupInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.task_group_status":                {ParamTypes: []string{"task.group"}, ParamSlots: taskGroupInfo.SlotCount, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.task_is_canceled":                 {ParamTypes: nil, ParamSlots: 0, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.task_checkpoint":                  {ParamTypes: nil, ParamSlots: 0, ReturnType: taskErrorInfo.Name, ReturnSlots: taskErrorInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.task_spawn_i32":                   {ParamTypes: []string{"str"}, ParamSlots: 2, ReturnType: taskHandleI32.Name, ReturnSlots: taskHandleI32.SlotCount, ReturnRegionParam: regionNone},
		"core.task_spawn_i32_typed":             {ParamTypes: []string{"str"}, ParamSlots: 2, ReturnType: taskHandleI32.Name, ReturnSlots: taskHandleI32.SlotCount, ReturnRegionParam: regionNone},
		"core.task_spawn_group_i32":             {ParamTypes: []string{"task.group", "str"}, ParamSlots: taskGroupInfo.SlotCount + 2, ReturnType: taskHandleI32.Name, ReturnSlots: taskHandleI32.SlotCount, ReturnRegionParam: regionNone},
		"core.task_spawn_group_i32_typed":       {ParamTypes: []string{"task.group", "str"}, ParamSlots: taskGroupInfo.SlotCount + 2, ReturnType: taskHandleI32.Name, ReturnSlots: taskHandleI32.SlotCount, ReturnRegionParam: regionNone},
		"core.task_join_i32":                    {ParamTypes: []string{"task.i32"}, ParamSlots: taskHandleI32.SlotCount, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.task_join_i32_typed":              {ParamTypes: []string{"task.i32"}, ParamSlots: taskHandleI32.SlotCount, ReturnType: "i32", ThrowsType: "enum", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.task_join_group_i32_typed":        {ParamTypes: []string{"task.i32"}, ParamSlots: taskHandleI32.SlotCount, ReturnType: "i32", ThrowsType: "enum", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.task_join_result_i32":             {ParamTypes: []string{"task.i32"}, ParamSlots: taskHandleI32.SlotCount, ReturnType: taskResultI32.Name, ReturnSlots: taskResultI32.SlotCount, ReturnRegionParam: regionNone},
		"core.task_join_until_i32":              {ParamTypes: []string{"task.i32", "i32"}, ParamSlots: taskHandleI32.SlotCount + 1, ReturnType: taskResultI32.Name, ReturnSlots: taskResultI32.SlotCount, ReturnRegionParam: regionNone},
		"core.task_poll_i32":                    {ParamTypes: []string{"task.i32"}, ParamSlots: taskHandleI32.SlotCount, ReturnType: taskResultI32.Name, ReturnSlots: taskResultI32.SlotCount, ReturnRegionParam: regionNone},
		"core.select2_i32":                      {ParamTypes: []string{"task.i32", "i32"}, ParamSlots: taskHandleI32.SlotCount + 1, ReturnType: taskResultI32.Name, ReturnSlots: taskResultI32.SlotCount, ReturnRegionParam: regionNone},
		"core.actor_dispatch":                   {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.actor_main_entry_id":              {ParamTypes: nil, ParamSlots: 0, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.actor_node_connect":               {ParamTypes: []string{"i32", "i32"}, ParamSlots: 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.actor_node_status":                {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.spawn":                            {ParamTypes: []string{"str"}, ParamSlots: 2, ReturnType: actorInfo.Name, ReturnSlots: actorInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.spawn_remote":                     {ParamTypes: []string{"i32", "str"}, ParamSlots: 3, ReturnType: actorInfo.Name, ReturnSlots: actorInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.send":                             {ParamTypes: []string{"actor", "i32"}, ParamSlots: 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.send_msg":                         {ParamTypes: []string{"actor", "i32", "i32"}, ParamSlots: 3, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.send_typed":                       {ParamTypes: []string{"actor", "enum"}, ParamSlots: 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.recv":                             {ParamTypes: nil, ParamSlots: 0, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.recv_msg":                         {ParamTypes: nil, ParamSlots: 0, ReturnType: actorMsgInfo.Name, ReturnSlots: actorMsgInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.recv_poll":                        {ParamTypes: nil, ParamSlots: 0, ReturnType: actorRecvResultI32.Name, ReturnSlots: actorRecvResultI32.SlotCount, ReturnRegionParam: regionNone},
		"core.recv_until":                       {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: actorRecvResultI32.Name, ReturnSlots: actorRecvResultI32.SlotCount, ReturnRegionParam: regionNone},
		"core.recv_msg_until":                   {ParamTypes: []string{"i32"}, ParamSlots: 1, ReturnType: actorRecvMsgResult.Name, ReturnSlots: actorRecvMsgResult.SlotCount, ReturnRegionParam: regionNone},
		"core.recv_typed":                       {ParamTypes: nil, ParamSlots: 0, ReturnType: "enum", ReturnSlots: 1, ReturnRegionParam: regionNone},
		"core.self":                             {ParamTypes: nil, ParamSlots: 0, ReturnType: actorInfo.Name, ReturnSlots: actorInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.sender":                           {ParamTypes: nil, ParamSlots: 0, ReturnType: actorInfo.Name, ReturnSlots: actorInfo.SlotCount, ReturnRegionParam: regionNone},
		"core.consent_token":                    {ParamTypes: nil, ParamSlots: 0, ReturnType: consentToken.Name, ReturnSlots: consentToken.SlotCount, ReturnRegionParam: regionNone},
		"core.secret_seal_i32":                  {ParamTypes: []string{"i32", consentToken.Name}, ParamSlots: 2, ReturnType: secretI32.Name, ReturnSlots: secretI32.SlotCount, ReturnRegionParam: regionNone},
		"core.secret_unseal_i32":                {ParamTypes: []string{secretI32.Name, consentToken.Name}, ParamSlots: 2, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone},
	}
	addAtomicBuiltinSigs(sigs, capMem.Name)
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

type atomicBuiltinValueType struct {
	Suffix   string
	TypeName string
}

var atomicBuiltinValueTypes = []atomicBuiltinValueType{
	{Suffix: "u8", TypeName: "u8"},
	{Suffix: "u16", TypeName: "u16"},
	{Suffix: "i32", TypeName: "i32"},
	{Suffix: "i64", TypeName: "i64"},
	{Suffix: "ptr", TypeName: "ptr"},
}

var atomicBuiltinLoadOrders = []string{"relaxed", "acquire", "seq_cst"}
var atomicBuiltinStoreOrders = []string{"relaxed", "release", "seq_cst"}
var atomicBuiltinReadModifyWriteOrders = []string{"relaxed", "acquire", "release", "acq_rel", "seq_cst"}
var atomicBuiltinFenceOrders = []string{"relaxed", "acquire", "release", "acq_rel", "seq_cst"}

func addAtomicBuiltinSigs(sigs map[string]FuncSig, capMem string) {
	for _, valueType := range atomicBuiltinValueTypes {
		for _, order := range atomicBuiltinLoadOrders {
			name := "core.atomic_load_" + valueType.Suffix + "_" + order
			sigs[name] = FuncSig{ParamTypes: []string{"ptr", capMem}, ParamSlots: 2, ReturnType: valueType.TypeName, ReturnSlots: 1, ReturnRegionParam: regionNone}
		}
		for _, order := range atomicBuiltinStoreOrders {
			name := "core.atomic_store_" + valueType.Suffix + "_" + order
			sigs[name] = FuncSig{ParamTypes: []string{"ptr", valueType.TypeName, capMem}, ParamSlots: 3, ReturnType: valueType.TypeName, ReturnSlots: 1, ReturnRegionParam: regionNone}
		}
		for _, op := range []string{"exchange", "fetch_add", "fetch_sub", "fetch_and", "fetch_or", "fetch_xor"} {
			for _, order := range atomicBuiltinReadModifyWriteOrders {
				name := "core.atomic_" + op + "_" + valueType.Suffix + "_" + order
				sigs[name] = FuncSig{ParamTypes: []string{"ptr", valueType.TypeName, capMem}, ParamSlots: 3, ReturnType: valueType.TypeName, ReturnSlots: 1, ReturnRegionParam: regionNone}
			}
		}
		for _, order := range atomicBuiltinReadModifyWriteOrders {
			name := "core.atomic_compare_exchange_" + valueType.Suffix + "_" + order
			sigs[name] = FuncSig{ParamTypes: []string{"ptr", valueType.TypeName, valueType.TypeName, capMem}, ParamSlots: 4, ReturnType: valueType.TypeName, ReturnSlots: 1, ReturnRegionParam: regionNone}
		}
		for _, order := range atomicBuiltinReadModifyWriteOrders {
			name := "core.atomic_compare_exchange_weak_" + valueType.Suffix + "_" + order
			sigs[name] = FuncSig{ParamTypes: []string{"ptr", valueType.TypeName, valueType.TypeName, capMem}, ParamSlots: 4, ReturnType: valueType.TypeName, ReturnSlots: 1, ReturnRegionParam: regionNone}
		}
	}
	for _, order := range atomicBuiltinFenceOrders {
		name := "core.atomic_fence_" + order
		sigs[name] = FuncSig{ParamTypes: []string{capMem}, ParamSlots: 1, ReturnType: "i32", ReturnSlots: 1, ReturnRegionParam: regionNone}
	}
}

func isCoreAtomicBuiltin(name string) bool {
	return strings.HasPrefix(name, "core.atomic_")
}

func atomicBuiltinDiagnostic(name string) (string, bool) {
	const prefix = "core.atomic_"
	if !strings.HasPrefix(name, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(name, prefix)
	if strings.HasPrefix(rest, "fence_") {
		order := strings.TrimPrefix(rest, "fence_")
		if !atomicBuiltinOrderKnown(order) {
			return "unsupported atomic memory order '" + order + "'", true
		}
		return "", false
	}
	for _, op := range atomicBuiltinOpPrefixes() {
		if !strings.HasPrefix(rest, op.Prefix) {
			continue
		}
		tail := strings.TrimPrefix(rest, op.Prefix)
		width, order, ok := splitAtomicBuiltinWidthOrder(tail)
		if !ok {
			width, order, ok = splitAtomicBuiltinKnownWidthUnknownOrder(tail)
			if ok {
				return "unsupported atomic memory order '" + order + "'", true
			}
			return "unsupported atomic value width '" + atomicBuiltinDiagnosticWidth(tail) + "'", true
		}
		if !atomicBuiltinWidthKnown(width) {
			return "unsupported atomic value width '" + width + "'", true
		}
		if !atomicBuiltinOrderAllowed(op.Name, order) {
			return "atomic " + op.Display + " does not support memory order " + order, true
		}
		return "", false
	}
	if op, ok := atomicBuiltinUnknownOp(rest); ok {
		return "unsupported atomic operation '" + op + "'", true
	}
	return "unsupported atomic builtin '" + name + "'", true
}

type atomicBuiltinOpPrefix struct {
	Prefix  string
	Name    string
	Display string
}

func atomicBuiltinOpPrefixes() []atomicBuiltinOpPrefix {
	return []atomicBuiltinOpPrefix{
		{Prefix: "compare_exchange_weak_", Name: "compare_exchange_weak", Display: "compare_exchange_weak"},
		{Prefix: "compare_exchange_", Name: "compare_exchange", Display: "compare_exchange"},
		{Prefix: "fetch_add_", Name: "fetch_add", Display: "fetch_add"},
		{Prefix: "fetch_sub_", Name: "fetch_sub", Display: "fetch_sub"},
		{Prefix: "fetch_and_", Name: "fetch_and", Display: "fetch_and"},
		{Prefix: "fetch_or_", Name: "fetch_or", Display: "fetch_or"},
		{Prefix: "fetch_xor_", Name: "fetch_xor", Display: "fetch_xor"},
		{Prefix: "exchange_", Name: "exchange", Display: "exchange"},
		{Prefix: "store_", Name: "store", Display: "store"},
		{Prefix: "load_", Name: "load", Display: "load"},
	}
}

func splitAtomicBuiltinWidthOrder(tail string) (width string, order string, ok bool) {
	for _, candidate := range atomicBuiltinAllOrders() {
		suffix := "_" + candidate
		if strings.HasSuffix(tail, suffix) {
			return strings.TrimSuffix(tail, suffix), candidate, true
		}
	}
	return "", "", false
}

func splitAtomicBuiltinKnownWidthUnknownOrder(tail string) (width string, order string, ok bool) {
	for _, valueType := range atomicBuiltinValueTypes {
		prefix := valueType.Suffix + "_"
		if strings.HasPrefix(tail, prefix) {
			return valueType.Suffix, strings.TrimPrefix(tail, prefix), true
		}
	}
	return "", "", false
}

func atomicBuiltinDiagnosticWidth(tail string) string {
	if before, _, ok := strings.Cut(tail, "_"); ok {
		return before
	}
	return tail
}

func atomicBuiltinUnknownOp(rest string) (string, bool) {
	for _, order := range atomicBuiltinAllOrders() {
		beforeOrder := strings.TrimSuffix(rest, "_"+order)
		if beforeOrder == rest {
			continue
		}
		for _, valueType := range atomicBuiltinValueTypes {
			widthSuffix := "_" + valueType.Suffix
			if strings.HasSuffix(beforeOrder, widthSuffix) {
				op := strings.TrimSuffix(beforeOrder, widthSuffix)
				if op != "" {
					return op, true
				}
			}
		}
	}
	return "", false
}

func atomicBuiltinAllOrders() []string {
	return []string{"relaxed", "acquire", "release", "acq_rel", "seq_cst"}
}

func atomicBuiltinWidthKnown(width string) bool {
	for _, valueType := range atomicBuiltinValueTypes {
		if width == valueType.Suffix {
			return true
		}
	}
	return false
}

func atomicBuiltinOrderKnown(order string) bool {
	for _, candidate := range atomicBuiltinAllOrders() {
		if order == candidate {
			return true
		}
	}
	return false
}

func atomicBuiltinOrderAllowed(op string, order string) bool {
	switch op {
	case "load":
		return order == "relaxed" || order == "acquire" || order == "seq_cst"
	case "store":
		return order == "relaxed" || order == "release" || order == "seq_cst"
	case "exchange", "compare_exchange", "compare_exchange_weak",
		"fetch_add", "fetch_sub", "fetch_and", "fetch_or", "fetch_xor":
		return atomicBuiltinOrderKnown(order)
	default:
		return false
	}
}

func builtinNeedsUnsafe(name string, argRegions []int) bool {
	if isCoreAtomicBuiltin(name) {
		return true
	}
	switch name {
	case "core.alloc_bytes", "core.island_new", "core.island_reset", "core.cap_io", "core.cap_mem",
		"core.raw_slice_u8_from_parts", "core.raw_slice_u16_from_parts",
		"core.raw_slice_i32_from_parts", "core.raw_slice_bool_from_parts",
		"core.load_i32", "core.store_i32",
		"core.load_u8", "core.store_u8",
		"core.load_ptr", "core.store_ptr", "core.store_arch_ptr",
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
	if isCoreAtomicBuiltin(name) {
		return "capsule.mem", "mem"
	}
	switch name {
	case "core.cap_io", "core.mmio_read_i32", "core.mmio_write_i32":
		return "capsule.io", "io"
	case "core.cap_mem",
		"core.raw_slice_u8_from_parts", "core.raw_slice_u16_from_parts",
		"core.raw_slice_i32_from_parts", "core.raw_slice_bool_from_parts",
		"core.load_i32", "core.store_i32",
		"core.load_u8", "core.store_u8",
		"core.load_ptr", "core.store_ptr", "core.store_arch_ptr",
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
	case "island_reset":
		return "core.island_reset", true
	case "load_ptr":
		return "core.load_ptr", true
	case "store_ptr":
		return "core.store_ptr", true
	case "store_arch_ptr":
		return "core.store_arch_ptr", true
	case "sym_addr":
		return "core.sym_addr", true
	case "ctx_switch":
		return "core.ctx_switch", true
	case "surface_open":
		return "core.surface_open", true
	case "surface_close":
		return "core.surface_close", true
	case "surface_poll_event_kind":
		return "core.surface_poll_event_kind", true
	case "surface_poll_event_x":
		return "core.surface_poll_event_x", true
	case "surface_poll_event_y":
		return "core.surface_poll_event_y", true
	case "surface_poll_event_button":
		return "core.surface_poll_event_button", true
	case "surface_poll_event_into":
		return "core.surface_poll_event_into", true
	case "surface_poll_event_text_len":
		return "core.surface_poll_event_text_len", true
	case "surface_poll_event_text_into":
		return "core.surface_poll_event_text_into", true
	case "surface_clipboard_write_text":
		return "core.surface_clipboard_write_text", true
	case "surface_clipboard_read_text_into":
		return "core.surface_clipboard_read_text_into", true
	case "surface_poll_composition_into":
		return "core.surface_poll_composition_into", true
	case "surface_begin_frame":
		return "core.surface_begin_frame", true
	case "surface_present_rgba":
		return "core.surface_present_rgba", true
	case "surface_now_ms":
		return "core.surface_now_ms", true
	case "surface_request_redraw":
		return "core.surface_request_redraw", true
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
		"core.raw_slice_u8_from_parts", "core.raw_slice_u16_from_parts",
		"core.raw_slice_i32_from_parts", "core.raw_slice_bool_from_parts",
		"core.island_new", "core.island_make_u8", "core.island_make_u16", "core.island_make_i32", "core.island_make_bool", "core.island_reset",
		"core.load_ptr", "core.store_ptr", "core.sym_addr", "core.ctx_switch",
		"core.surface_open", "core.surface_close", "core.surface_poll_event_kind", "core.surface_poll_event_x",
		"core.surface_poll_event_y", "core.surface_poll_event_button", "core.surface_poll_event_into", "core.surface_poll_event_text_len", "core.surface_poll_event_text_into",
		"core.surface_clipboard_write_text", "core.surface_clipboard_read_text_into", "core.surface_poll_composition_into", "core.surface_begin_frame",
		"core.surface_present_rgba", "core.surface_now_ms", "core.surface_request_redraw",
		"core.time_now_ms", "core.sleep_ms", "core.sleep_until", "core.deadline_ms", "core.timer_ready", "core.yield",
		"core.task_group_open", "core.task_group_close", "core.task_group_cancel", "core.task_group_current", "core.task_group_status",
		"core.task_is_canceled", "core.task_checkpoint",
		"core.task_spawn_i32", "core.task_spawn_i32_typed", "core.task_spawn_group_i32", "core.task_spawn_group_i32_typed",
		"core.task_join_i32", "core.task_join_i32_typed", "core.task_join_group_i32_typed", "core.task_join_result_i32", "core.task_join_until_i32",
		"core.task_poll_i32", "core.select2_i32",
		"core.send_msg", "core.recv_msg", "core.recv_poll", "core.recv_until", "core.recv_msg_until", "core.send_typed", "core.recv_typed",
		"core.actor_dispatch", "core.actor_main_entry_id", "core.actor_node_connect", "core.actor_node_status", "core.spawn_remote",
		"core.consent_token", "core.secret_seal_i32", "core.secret_unseal_i32":
		return name, true
	default:
		return "", false
	}
}
