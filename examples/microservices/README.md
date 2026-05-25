# Tetra Microservices

These examples model small production-style services using the current Tetra
v0.4.0 Linux-x64 service surface.

The current stable service transport in this repository is actor messaging and
the Linux-x64 `actor-net` broker. Tetra does not currently expose a stable HTTP
server API in the language, so these examples keep the business handlers
deterministic and executable while using the same actor/task runtime contracts
that back distributed actor release evidence.

Examples:

- `inventory_service.tetra`: checks stock requests through a tagged actor
  message.
- `payments_service.tetra`: authorizes a bounded payment request through a
  tagged actor message.
- `orders_gateway.tetra`: composes inventory, payment, and audit task handlers
  as a small gateway workflow.
- `memory_cache_service.tetra`: uses capability-bound memory helpers to seed,
  copy, and read a small cache line.
- `parallel_fanout_service.tetra`: spawns and joins independent task workers as
  a deterministic fanout service.
- `compiler_pipeline_service.tetra`: models parser/checker/codegen stages with
  enum payloads, generic identity, and static protocol dispatch.
- `island_cache_pool_service.tetra`: keeps two scoped cache regions in one
  service struct and verifies the checksum before the regions expire.
- `parallel_task_pool_service.tetra`: repeatedly schedules shard workers through
  the task runtime and joins every result deterministically.
- `compiler_artifact_router_service.tetra`: routes compile artifacts through
  enum payload states, static protocol dispatch, and generic helpers.
- `memory_journal_service.tetra`: stores and loads journal cells through
  capability-bound pointer arithmetic.
- `task_group_service.tetra`: schedules two grouped workers, joins typed result
  records, and closes the task group deterministically.
- `typed_task_error_service.tetra`: catches typed task error payloads from a
  worker through the typed task ABI.
- `task_group_cancel_service.tetra`: verifies grouped typed task cancellation
  maps to the stopped error path.
- `wait_select_service.tetra`: probes a pending task and completes it through a
  deadline-aware select path.
- `memory_bounds_probe_service.tetra`: writes a bounded cap.mem journal through
  positive `ptr_add` offsets and reads it back.
- `callable_router_service.tetra`: routes a captured callable through a struct
  field, enum payload, and callback parameter.
- `compiler_modular_gateway/app/main.tetra`: imports a cross-module compiler
  pipeline and runs enum payload plus generic helper dispatch.
- `island_slice_matrix_service.tetra`: verifies multiple typed island-backed
  slice lanes in one scoped region.
- `generic_optional_router_service.tetra`: exercises generic optional payload
  inference through a small status router.
- `actor_deadline_router_service.tetra`: combines tagged actor routing with an
  empty deadline receive and a deadline-aware reply wait.
- `typed_task_success_service.tetra`: exercises the successful path of the
  typed task ABI while preserving the typed catch route.
- `memory_byte_window_service.tetra`: writes byte-sized journal cells through
  cap.mem pointer offsets.
- `callable_return_router_service.tetra`: constructs a captured callable through
  a function-typed return and dispatches it through a callback parameter.
- `compiler_callable_pack/app/main.tetra`: imports a callable-bearing struct
  from another module and dispatches its payload in the app module.
- `actor_tagged_loop_service.tetra`: stress-tests tagged actor request/reply
  loops with deterministic drift checking.
- `task_group_lifecycle_service.tetra`: covers grouped spawn/join, close
  status, and cancel status in one service.
- `memory_negative_guard_service.tetra`: validates rejected negative memory
  helper lengths as an invalid-input guard path.
- `callable_identity_router_service.tetra`: routes a function-typed value
  through identity and callback helpers.
- `compiler_throwing_callable_pack/app/main.tetra`: imports a throwing callable
  producer and callback helper across modules.
- `actor_poll_timeout_service.tetra`: validates empty actor polling and
  deadline-aware tagged receive recovery.
- `task_timeout_recovery_service.tetra`: observes a task wait timeout and then
  joins the same task successfully.
- `memory_u16_lane_service.tetra`: checks island-backed `u16` slice lanes.
- `generic_struct_router_service.tetra`: routes a status through a generic
  struct and generic wrapper in one module.
- `compiler_generic_box_pack/app/main.tetra`: imports a generic struct helper
  across module boundaries.
- `task_group_payload_service.tetra`: catches typed payload errors from grouped
  tasks.
- `actor_sender_snapshot_service.tetra`: snapshots `core.sender()` and replies
  after a delayed actor step.
- `memory_copy_window_service.tetra`: copies a byte window between two raw
  allocations and verifies the checksum.
- `protocol_bound_generic_service.tetra`: exercises same-file static
  protocol-bound generic routing.
- `compiler_protocol_pack/app/main.tetra`: imports protocol-bound generic
  routing across module boundaries.
- `actor_state_counter_service.tetra`: reads scalar actor state across a tagged
  message loop while keeping the mutable counter local.
- `task_group_self_cancel_service.tetra`: cancels the current task group from a
  worker and checkpoints the cancellation path.
- `generic_typed_error_service.tetra`: monomorphizes a generic typed-error
  helper over an enum payload.
- `compiler_generic_error_pack/app/main.tetra`: imports generic typed-error
  dispatch across module boundaries.
- `task_group_current_status_service.tetra`: reads the current task-group
  handle inside a grouped worker and reports its status.
- `actor_dual_mailbox_service.tetra`: fans out to two actors and checks the
  combined deadline-aware mailbox replies without depending on delivery order.
- `memory_memset_stride_service.tetra`: fills a raw allocation with
  `memset_u8` and checks offset reads.
- `island_bool_flags_service.tetra`: stores service flags in an island-backed
  boolean slice.
- `compiler_generic_pair_pack/app/main.tetra`: imports a two-parameter generic
  struct helper across module boundaries.
- `actor_dual_value_mailbox_service.tetra`: fans out value messages to two
  actors and checks deadline-aware replies.
- `task_dual_deadline_service.tetra`: combines task timeout recovery with an
  independent fast task join.
- `memory_zero_copy_service.tetra`: verifies zero-length `memcpy_u8` leaves the
  target buffer untouched.
- `compiler_optional_box_pack/app/main.tetra`: imports a generic optional field
  helper across module boundaries.
- `actor_timeout_retry_service.tetra`: retries a delayed actor receive after an
  initial timeout.
- `task_poll_deadline_matrix_service.tetra`: combines task poll, deadline join,
  timeout, and final join paths.
- `memory_ptr_table_service.tetra`: stores and reloads an allocation-base
  pointer through raw memory.
- `optional_enum_router_service.tetra`: routes an optional enum payload through
  match and if-let.
- `optional_field_update_service.tetra`: updates an optional struct field
  through an explicitly typed optional local.
- `actor_chain_reply_service.tetra`: chains a router actor through a worker
  actor and forwards the reply.
- `task_group_poll_service.tetra`: polls a grouped task before joining and
  closing its group.
- `memory_i32_stride_service.tetra`: verifies four i32 cells at raw memory
  stride offsets.
- `actor_value_chain_service.tetra`: forwards a plain actor value request
  through a router and worker chain.
- `task_group_typed_success_service.tetra`: joins a typed grouped task through
  the success path and closes the group.
- `memory_chained_ptr_stride_service.tetra`: verifies pointer arithmetic
  provenance across chained `ptr_add` offsets.
- `compiler_optional_enum_pack/app/main.tetra`: imports an optional enum router
  and scores empty plus ready payloads across module boundaries.
- `actor_typed_payload_service.tetra`: sends enum payloads through the typed
  actor message ABI.
- `task_select_timeout_service.tetra`: observes a task select timeout and then
  joins the same task successfully.
- `memory_mixed_width_service.tetra`: combines byte and i32 raw-memory lanes in
  one allocation.
- `compiler_extension_pack/app/main.tetra`: imports an extension method and
  dispatches it as a static extension call.
- `actor_self_mailbox_service.tetra`: sends tagged messages to `core.self()`
  and verifies local mailbox ordering.
- `task_group_cancel_after_spawn_service.tetra`: cancels a grouped typed task
  after spawning and checks the stopped path.
- `memory_derived_copy_service.tetra`: copies a byte window between derived raw
  pointers with unrolled loads/stores and checks the copied checksum.
- `compiler_protocol_extension_pack/app/main.tetra`: imports a protocol-backed
  extension implementation and dispatches it through a static extension call.
- `actor_typed_chain_service.tetra`: routes a typed actor payload through a
  router actor and worker actor.
- `task_group_multi_cancel_service.tetra`: cancels two grouped typed tasks and
  checks both stopped joins.
- `memory_derived_ptr_table_service.tetra`: stores and reloads an allocation
  base pointer, then derives the target cell after loading the table entry.
- `compiler_generic_function_pack/app/main.tetra`: imports a generic identity
  helper across module boundaries and dispatches it with a typed local.
- `actor_self_typed_mailbox_service.tetra`: sends a typed enum payload to
  `core.self()` and receives it from the local typed mailbox.
- `actor_task_bridge_service.tetra`: joins a task worker from inside an actor
  and replies with the combined actor/task result.
- `memory_aggregate_ptr_service.tetra`: carries an allocation-base pointer
  through an enum aggregate before deriving raw-memory cells.
- `compiler_generic_extension_local_service.tetra`: confirms same-file generic
  extension static-call monomorphization.
- `actor_typed_task_bridge_service.tetra`: handles a typed actor request by
  joining a typed task worker and replying with a typed actor payload.
- `task_group_actor_fanout_service.tetra`: opens a task group from inside an
  actor and replies after joining both grouped workers.
- `memory_optional_ptr_service.tetra`: carries an allocation-base pointer
  through an optional payload before deriving a checked raw-memory cell.
- `compiler_callable_generic_route_service.tetra`: routes scalar work through a
  generic helper and dispatches a callable through a callback parameter.
- `task_actor_roundtrip_service.tetra`: spawns an actor from a task worker and
  returns the actor reply through the task runtime.
- `actor_typed_task_group_service.tetra`: opens a typed task group inside an
  actor and replies after catching the typed task result.
- `memory_function_ptr_service.tetra`: routes allocation-base pointers through
  function and generic helpers before deriving checked raw-memory cells.
- `task_typed_actor_roundtrip_service.tetra`: spawns a typed actor from inside
  a task worker and returns the typed reply through the task runtime.
- `actor_task_select_service.tetra`: selects a fast task and joins a slow task
  from inside an actor before replying.
- `compiler_generic_optional_route_service.tetra`: routes an optional scalar
  through a generic helper and dispatches the unwrapped value through a
  callback.
- `memory_global_state_service.tetra`: keeps global scalar service counters
  while raw pointers and pointer offsets stay local.
- `actor_typed_task_error_bridge_service.tetra`: forwards a typed task error
  payload through a typed actor reply.
- `actor_task_cancel_select_service.tetra`: observes a grouped task timeout,
  cancels it, and replies with the combined task status from an actor.
- `compiler_generic_optional_import_pack/app/main.tetra`: imports a generic
  optional identity helper across module boundaries.
- `memory_mutable_ptr_service.tetra`: keeps allocation-base pointers in mutable
  locals and derives raw-memory cells after selecting the active base.
- `memory_struct_offset_service.tetra`: stores pointer-window configuration in
  a struct and aliases the offset locally before raw pointer arithmetic.
- `actor_task_recovery_service.tetra`: catches a typed task failure inside an
  actor, runs a recovery task, and replies with the combined result.
- `compiler_generic_nested_optional_pack/app/main.tetra`: imports a generic
  optional-bearing struct helper across module boundaries.
- `memory_function_offset_service.tetra`: aliases function-returned and generic
  offsets into locals before raw pointer arithmetic.
- `memory_expression_offset_service.tetra`: computes arithmetic pointer offsets
  in locals before calling `core.ptr_add`.
- `actor_timer_task_matrix_service.tetra`: combines timer readiness, task
  polling, and final task join inside an actor.
- `compiler_generic_enum_import_pack/app/main.tetra`: imports a generic enum
  payload router across module boundaries.
- `memory_task_result_offset_service.tetra`: aliases task result fields into
  local offsets before raw pointer arithmetic.
- `memory_join_until_result_offset_service.tetra`: aliases a
  `task_join_until_i32` result value into a local offset before checked
  raw-memory access.
- `memory_poll_result_offset_service.tetra`: observes a pending poll, then
  aliases the completed `task_poll_i32` result value into a local offset before
  checked raw-memory access.
- `memory_select_result_offset_service.tetra`: aliases a successful
  `select2_i32` result value into a local offset before checked raw-memory
  access.
- `compiler_task_wait_memory_pack/app/main.tetra`: imports wait/poll/select
  task-result helpers and validates their local-offset memory routes under
  executable and `--interface-only` `Jobs: 4` compiler modes.
- `memory_join_until_error_offset_service.tetra`: observes a join-until
  timeout and uses the direct result error field as a byte offset before
  joining the task.
- `memory_poll_error_offset_service.tetra`: observes a pending poll and uses
  the direct poll error field as a byte offset before joining the task.
- `memory_select_error_offset_service.tetra`: observes a select timeout and
  uses the direct selected error field as a byte offset before joining the task.
- `compiler_task_wait_error_memory_pack/app/main.tetra`: imports
  wait/poll/select timeout error helpers and validates their byte-offset memory
  routes under executable and `--interface-only` `Jobs: 4` compiler modes.
- `memory_actor_message_offset_service.tetra`: aliases actor message fields into
  local offsets before raw pointer arithmetic.
- `memory_actor_recv_value_offset_service.tetra`: aliases a deadline-aware actor
  receive value into a local offset before raw pointer arithmetic.
- `memory_actor_poll_value_offset_service.tetra`: observes an empty actor poll,
  then aliases the completed poll value into a local offset before raw pointer
  arithmetic.
- `memory_actor_tag_offset_service.tetra`: aliases a tagged actor receive tag
  into a local offset before raw pointer arithmetic.
- `compiler_actor_wait_memory_pack/app/main.tetra`: imports actor receive, poll,
  and tagged-receive offset helpers and validates their memory routes under
  executable and `--interface-only` `Jobs: 4` compiler modes.
- `memory_actor_recv_error_offset_service.tetra`: observes a deadline-aware
  actor receive timeout and uses the direct error field as a byte offset before
  draining the delayed reply.
- `memory_actor_poll_error_offset_service.tetra`: observes an empty actor poll
  and uses the direct error field as a byte offset.
- `memory_actor_recv_msg_error_offset_service.tetra`: observes a tagged actor
  receive timeout and uses the direct error field as a byte offset before
  draining the delayed tagged reply.
- `compiler_actor_error_memory_pack/app/main.tetra`: imports actor receive,
  poll, and tagged-receive timeout error helpers and validates their byte-offset
  memory routes under executable and `--interface-only` `Jobs: 4` compiler
  modes.
- `actor_task_group_error_recovery_service.tetra`: catches a typed grouped-task
  error inside an actor and runs a recovery task before replying.
- `compiler_generic_struct_field_pack/app/main.tetra`: imports a generic struct
  field wrapper across module boundaries.
- `memory_indexed_metadata_offset_service.tetra`: aliases slice-index and string
  metadata offset reads into locals before raw pointer arithmetic.
- `parallel_typed_task_payload_handle_service.tetra`: routes payload typed-task
  and grouped-task errors while keeping typed handles inferred.
- `actor_typed_dual_mailbox_service.tetra`: fans out typed actor requests to
  two actors and combines both blocking typed replies.
- `task_group_nested_service.tetra`: opens a child task group from inside a
  grouped parent worker and closes both groups deterministically.
- `compiler_generic_optional_struct_pack/app/main.tetra`: imports a generic
  optional struct wrapper and unwraps it across module boundaries.
- `memory_direct_base_offset_service.tetra`: aliases direct allocation,
  function-returned, and loaded pointer bases before raw pointer arithmetic.
- `parallel_typed_task_wide_payload_service.tetra`: exercises wide payload
  typed-task and grouped typed-task catch paths near the current slot boundary.
- `actor_typed_wide_payload_service.tetra`: sends and receives an eight-slot
  typed actor enum payload.
- `compiler_cross_module_runtime_pack/app/main.tetra`: imports cross-module
  task and typed actor workers and spawns them by module alias entrypoint.
- `actor_typed_envelope_service.tetra`: keeps typed actor traffic on one shared
  enum envelope to avoid cross-enum mailbox reinterpretation.
- `parallel_time_window_service.tetra`: checks logical sleep, immediate
  negative deadlines, and deadline-aware task join behavior.
- `actor_state_status_service.tetra`: mixes supported actor state slot types
  and replies with a deterministic state checksum.
- `memory_inline_ptradd_window_service.tetra`: uses inline allocation-base
  `ptr_add` expressions directly inside raw load/store calls.
- `parallel_typed_task_struct_payload_service.tetra`: sends structured
  value-only error payloads through plain and grouped typed-task joins.
- `actor_typed_struct_payload_service.tetra`: sends and receives structured
  value-only payloads through a typed actor envelope.
- `memory_callable_ptr_base_service.tetra`: routes allocation-base pointers
  through function-typed callables before deriving raw-memory offsets.
- `memory_callable_optional_ptr_service.tetra`: routes allocation-base pointers
  through optional-pointer callable parameters, struct fields, and enum
  payloads before deriving raw-memory offsets after unwrapping.
- `compiler_match_ptr_base_service.tetra`: extracts allocation-base pointers
  from enum match expressions before raw pointer arithmetic.
- `memory_typed_error_ptr_base_service.tetra`: carries allocation-base pointers
  through synchronous typed-error catch paths before deriving offsets.
- `parallel_join_until_rejoin_service.tetra`: checks the current
  non-consuming `task_join_until_i32` handle model after a successful wait.
- `actor_task_result_window_service.tetra`: polls and joins a task from inside
  an actor, then replies through a deadline-aware mailbox receive.
- `compiler_inout_return_service.tetra`: uses scalar `inout` by consuming the
  returned value explicitly as the stable writeback workaround.
- `memory_dynamic_base_offset_service.tetra`: keeps dynamic offset loops
  anchored at allocation-base pointers before raw memory loads.
- `parallel_group_close_before_join_service.tetra`: closes a task group before
  joining its scheduled child and verifies the result remains available.
- `parallel_group_cancel_after_join_service.tetra`: joins a grouped child, then
  cancels and closes the group to verify post-join lifecycle behavior.
- `compiler_parallel_jobs_pack/app/main.tetra`: imports a packet router and
  task worker and is built by the bug ledger gate with `Jobs: 4`.
- `parallel_cross_module_typed_task_pack/app/main.tetra`: catches imported
  typed-task errors from plain and grouped cross-module workers.
- `parallel_selfhost_deadline_service.tetra`: exercises the explicit selfhost
  runtime's supported untyped deadline/poll task surface.
- `memory_struct_base_dynamic_service.tetra`: carries allocation-base pointers
  through struct fields before dynamic raw-memory loops.
- `memory_enum_base_dynamic_service.tetra`: carries allocation-base pointers
  through enum payloads before dynamic raw-memory loops.
- `memory_typed_error_base_dynamic_service.tetra`: recovers allocation-base
  pointers from typed errors before dynamic raw-memory loops.
- `parallel_select_recovery_service.tetra`: recovers from a `select2_i32`
  timeout through poll and blocking join.
- `compiler_pattern_binding_unique_service.tetra`: exercises match, if-let, and
  typed catch patterns with whole-function-unique binding names.
- `memory_base_dynamic_copy_service.tetra`: copies a byte subwindow with
  dynamic source/destination offsets anchored at allocation bases.
- `parallel_select_rejoin_service.tetra`: checks that a successful
  `select2_i32` wait leaves the task handle available for a full join.
- `parallel_group_cancel_select_service.tetra`: verifies `select2_i32` observes
  a canceled grouped task without waiting until the deadline.
- `compiler_interface_jobs_pack/app/main.tetra`: imports a packet router and is
  built by the bug ledger gate in both executable and `--interface-only`
  `Jobs: 4` compiler modes.
- `memory_zero_length_derived_helper_service.tetra`: calls `memcpy_u8` and
  `memset_u8` with zero length over derived windows to confirm the helpers do
  not touch memory in that case.
- `parallel_group_spawn_after_cancel_service.tetra`: spawns into a canceled task
  group and checks the join observes the stopped result.
- `parallel_join_until_poll_service.tetra`: observes a successful
  `task_join_until_i32` result again through a later non-blocking poll.
- `compiler_interface_control_pack/app/main.tetra`: imports a route enum and is
  built by the bug ledger gate in executable and `--interface-only` `Jobs: 4`
  compiler modes.
- `memory_zero_length_base_helper_service.tetra`: calls `memcpy_u8` and
  `memset_u8` with zero length over allocation bases and checks both cells stay
  unchanged.
- `parallel_yield_join_window_service.tetra`: polls a pending task, yields to
  the cooperative runtime, then completes through `task_join_until_i32`.
- `parallel_group_status_roundtrip_service.tetra`: checks parent and child
  visibility of task-group status before and after close.
- `memory_group_status_direct_offset_service.tetra`: uses direct
  `task_group_status` builtin results as byte offsets for open, canceled, and
  closed task-group states.
- `memory_group_current_status_offset_service.tetra`: returns the current
  task-group status from a grouped worker and uses it as a memory offset after
  joining.
- `parallel_group_cancel_close_direct_service.tetra`: closes the task-group
  handle returned directly from `task_group_cancel`.
- `compiler_group_status_memory_pack/app/main.tetra`: imports task-group status
  helpers and validates their byte-offset memory routes under executable and
  `--interface-only` `Jobs: 4` compiler modes.
- `compiler_import_alias_pack/app/main.tetra`: imports a math helper module
  through a stable alias and is built by the bug ledger gate in executable and
  `--interface-only` `Jobs: 4` compiler modes.
- `memory_heap_u16_slice_service.tetra`: allocates a heap-backed `[]u16` slice
  and verifies indexed read-after-write arithmetic.
- `memory_heap_bool_flags_service.tetra`: allocates a heap-backed `[]bool`
  slice and verifies boolean lane reads after writes.
- `parallel_actor_yield_mailbox_service.tetra`: starts a worker task, observes
  an empty parent mailbox, yields, then receives the worker payload.
- `parallel_group_current_cancel_status_service.tetra`: cancels a worker's
  current task group and verifies both worker and parent status observations.
- `compiler_cross_module_actor_pack/app/main.tetra`: imports a worker actor
  module, spawns the actor through the module alias, and is built by the bug
  ledger gate in executable and `--interface-only` `Jobs: 4` compiler modes.
- `memory_heap_i32_bool_slice_service.tetra`: combines heap-backed `[]i32` and
  `[]bool` slices and verifies cross-lane read-after-write behavior.
- `parallel_task_actor_deadline_service.tetra`: starts a task that replies
  through the actor mailbox, observes an initial deadline timeout, then joins
  and receives through deadline waits.
- `compiler_actor_resource_pack/app/main.tetra`: carries actor handles through
  imported struct fields and enum payloads, then sends through the recovered
  handles under executable and `--interface-only` `Jobs: 4` compiler modes.
- `memory_heap_u8_slice_service.tetra`: allocates a heap-backed `[]u8` slice
  and verifies indexed byte read-after-write arithmetic.
- `parallel_typed_group_cancel_status_service.tetra`: cancels a typed grouped
  task and verifies the typed stopped branch matches canceled group status.
- `compiler_callable_return_pack/app/main.tetra`: imports function-typed
  callback construction and application, then builds under executable and
  `--interface-only` `Jobs: 4` compiler modes.
- `compiler_callable_optional_ptr_pack/app/main.tetra`: imports
  optional-pointer callback helpers, routes allocation-base pointers through
  imported function-typed fields and enum payloads, and builds under executable
  and `--interface-only` `Jobs: 4` compiler modes.
- `compiler_async_interface_pack/app/main.tetra`: imports a module that exposes
  async helpers plus a synchronous fallback and builds under executable and
  `--interface-only` `Jobs: 4` compiler modes.
- `memory_slice_optional_service.tetra`: stores a heap-backed `[]u8` in an
  optional struct field and reads through the payload binding.
- `memory_slice_enum_service.tetra`: stores a heap-backed `[]u8` in an enum
  payload and reads through the match binding.
- `parallel_task_result_box_service.tetra`: stores `task.result_i32` in a
  struct field after a join and reads result fields through the wrapper.
- `parallel_task_result_enum_service.tetra`: stores `task.result_i32` in an
  enum payload after a join and reads result fields through the match binding.
- `compiler_test_command_service.tetra`: combines an executable entrypoint with
  a Tetra `test` declaration to exercise the CLI test runner surface.
- `parallel_task_test_command_service.tetra`: combines a runtime task join in
  `main` with a Tetra `test` declaration for the worker function.
- `generic_typed_result_payload_service.tetra`: wraps typed task results in a
  generic struct and validates both the joined payload and status fields.
- `memory_slice_struct_loop_service.tetra`: stores a heap-backed `[]u8` in a
  struct field and verifies looped reads after indexed writes.
- `memory_slice_generic_box_service.tetra`: carries a heap-backed `[]u8`
  through an inferred generic struct wrapper and reads it after writes.
- `parallel_task_result_optional_service.tetra`: stores `task.result_i32` in an
  optional struct field and unwraps it before checking the result.
- `parallel_nested_task_spawn_service.tetra`: starts a task whose worker starts
  and joins a child task before returning to the parent.
- `compiler_generic_slice_pack/app/main.tetra`: imports a generic slice helper
  module and builds under executable and `--interface-only` `Jobs: 4` compiler
  modes.
- `memory_slice_for_loop_service.tetra`: fills a heap-backed `[]u8` and sums it
  through collection-loop lowering.
- `memory_slice_inout_mutation_service.tetra`: mutates a heap-backed `[]u8`
  through an `inout` parameter and reads caller-visible cells afterwards.
- `parallel_task_handle_optional_join_service.tetra`: carries a task handle
  through an optional payload and joins only the recovered handle.
- `parallel_task_optional_enum_join_service.tetra`: carries a task handle
  through an enum payload wrapped in an optional and joins the matched handle.
- `compiler_optional_task_pack/app/main.tetra`: imports an optional task-handle
  helper module and builds under executable and `--interface-only` `Jobs: 4`
  compiler modes.
- `memory_bool_for_loop_service.tetra`: fills a heap-backed `[]bool` and sums
  true lanes through collection-loop lowering.
- `memory_i32_for_loop_service.tetra`: fills a heap-backed `[]i32` and sums it
  through collection-loop lowering.
- `parallel_actor_handle_optional_send_service.tetra`: carries an actor handle
  through an optional payload and sends only through the recovered handle.
- `parallel_actor_optional_enum_send_service.tetra`: carries an actor handle
  through an enum payload wrapped in an optional and sends through the matched
  handle.
- `compiler_optional_actor_pack/app/main.tetra`: imports an optional
  actor-handle helper module and builds under executable and `--interface-only`
  `Jobs: 4` compiler modes.
- `memory_u16_for_loop_service.tetra`: fills a heap-backed `[]u16` and sums it
  through collection-loop lowering.
- `parallel_group_optional_close_service.tetra`: carries a task group through
  an optional payload and closes only the recovered handle.
- `parallel_group_optional_cancel_service.tetra`: carries a task group through
  an optional payload, cancels it, checks canceled status, and closes it.
- `compiler_optional_group_pack/app/main.tetra`: imports an optional task-group
  helper module and builds under executable and `--interface-only` `Jobs: 4`
  compiler modes.
- `memory_bool_inout_toggle_service.tetra`: mutates a heap-backed `[]bool`
  through an `inout` parameter and verifies caller-visible lanes afterwards.
- `memory_i32_inout_fill_service.tetra`: mutates a heap-backed `[]i32` through
  an `inout` parameter and reads caller-visible cells afterwards.
- `parallel_group_optional_match_close_service.tetra`: carries a task group
  through an optional payload, recovers it via `match`, checks status, and
  closes the recovered handle.
- `parallel_group_optional_enum_close_service.tetra`: carries a task group
  through an enum payload wrapped in an optional and limits the matched handle
  to status/close operations.
- `compiler_optional_group_match_pack/app/main.tetra`: imports an optional
  task-group helper module, recovers the group through `match`, and builds under
  executable and `--interface-only` `Jobs: 4` compiler modes.
- `parallel_group_struct_spawn_service.tetra`: carries a task group through a
  struct field, spawns a grouped worker from the recovered handle, joins it, and
  closes the handle.
- `parallel_group_enum_spawn_service.tetra`: carries a task group through an
  enum payload, spawns a grouped worker from the matched handle, joins it, and
  closes the handle.
- `parallel_group_typed_struct_spawn_service.tetra`: carries a task group
  through a struct field and joins a typed grouped worker through the success
  path.
- `parallel_group_typed_enum_spawn_service.tetra`: carries a task group through
  an enum payload and joins a typed grouped worker through the success path.
- `memory_u16_inout_stride_service.tetra`: mutates a heap-backed `[]u16`
  through an `inout` parameter and verifies dependent indexed writes from the
  caller afterwards.
- `compiler_group_aggregate_pack/app/main.tetra`: imports task-group aggregate
  helper types, spawns untyped and typed grouped workers through imported
  struct/enum payload routes, and builds under executable and
  `--interface-only` `Jobs: 4` compiler modes.
- `parallel_group_alias_spawn_service.tetra`: sends a task group through a
  non-generic identity helper, spawns a grouped worker from the returned handle,
  joins it, and closes the handle.
- `parallel_group_generic_box_spawn_service.tetra`: carries a task group
  through a monomorphized generic `Box<task.group>`, spawns a grouped worker
  from the boxed handle, joins it, and closes the handle.
- `memory_optional_generic_u16_box_service.tetra`: carries a heap-backed
  `[]u16` through a generic box wrapped in an optional payload and verifies
  indexed reads after unwrapping.
- `compiler_group_generic_pack/app/main.tetra`: imports generic task-group
  box helpers, spawns grouped workers through imported generic-box and
  non-generic alias routes, and builds under executable and `--interface-only`
  `Jobs: 4` compiler modes.
- `parallel_task_alias_join_service.tetra`: sends a task handle through a
  non-generic identity helper and joins the returned handle.
- `parallel_task_generic_box_join_service.tetra`: carries a task handle through
  a monomorphized generic `Box<task.i32>` and joins the boxed handle.
- `parallel_task_optional_struct_box_join_service.tetra`: carries a task-handle
  struct wrapper through an optional payload and joins the unwrapped field.
- `parallel_task_optional_generic_box_join_service.tetra`: carries a generic
  task-handle box through an optional payload and joins the unwrapped field.
- `memory_optional_generic_bool_box_service.tetra`: carries a heap-backed
  `[]bool` through a generic box wrapped in an optional payload and verifies
  indexed reads after unwrapping.
- `compiler_task_generic_pack/app/main.tetra`: imports generic task-handle box
  helpers, joins imported generic-box and non-generic alias task handles, and
  builds under executable and `--interface-only` `Jobs: 4` compiler modes.
- `parallel_actor_alias_send_service.tetra`: sends an actor handle through a
  non-generic identity helper and sends a request through the returned handle.
- `parallel_actor_generic_box_send_service.tetra`: carries an actor handle
  through a monomorphized generic `Box<actor>` and sends through the boxed
  handle.
- `parallel_actor_optional_struct_box_send_service.tetra`: carries an actor
  struct wrapper through an optional payload and sends through the unwrapped
  field.
- `parallel_actor_optional_generic_box_send_service.tetra`: carries a generic
  actor box through an optional payload and sends through the unwrapped field.
- `memory_optional_generic_i32_box_service.tetra`: carries a heap-backed `[]i32`
  through a generic box wrapped in an optional payload and verifies indexed
  reads after unwrapping.
- `compiler_actor_generic_pack/app/main.tetra`: imports generic actor box
  helpers, sends through imported generic-box and non-generic alias actor
  handles, and builds under executable and `--interface-only` `Jobs: 4`
  compiler modes.
- `memory_island_alias_region_service.tetra`: sends an island handle through a
  non-generic identity helper, allocates a region-backed slice from the returned
  handle, and frees it.
- `memory_island_generic_box_region_service.tetra`: carries an island handle
  through a monomorphized generic `Box<island>`, allocates from the boxed
  handle, and frees it.
- `memory_island_optional_struct_box_service.tetra`: carries an island struct
  wrapper through an optional payload, allocates from the unwrapped handle, and
  frees it.
- `memory_island_optional_generic_box_service.tetra`: carries a generic island
  box through an optional payload, allocates from the unwrapped generic field,
  and frees it.
- `compiler_island_generic_pack/app/main.tetra`: imports generic island box
  helpers, allocates through imported generic-box and non-generic alias island
  handles, and builds under executable and `--interface-only` `Jobs: 4`
  compiler modes.
- `memory_ptr_alias_base_service.tetra`: sends an allocation-base pointer
  through a non-generic identity helper and derives the checked cell from the
  returned base.
- `memory_ptr_generic_identity_base_service.tetra`: sends an allocation-base
  pointer through a generic identity helper and derives the checked cell from
  the returned base.
- `compiler_ptr_generic_pack/app/main.tetra`: imports pointer alias and generic
  identity helpers, derives checked cells from returned allocation-base
  pointers, and builds under executable and `--interface-only` `Jobs: 4`
  compiler modes.
- `memory_task_result_optional_offset_service.tetra`: unwraps optional
  `task.result_i32` payloads, aliases joined values into local offsets, and
  uses those offsets for checked raw-memory reads.
- `memory_task_result_optional_enum_offset_service.tetra`: unwraps a
  `task.result_i32` enum payload carried inside an optional, aliases the joined
  value into a raw-memory offset, and verifies checked reads.
- `parallel_task_result_generic_box_service.tetra`: carries a joined task
  result through a monomorphized generic box and aliases the selected field
  before generic identity routing.
- `compiler_task_result_generic_pack/app/main.tetra`: imports task-result
  generic box helpers, aliases selected result fields before imported generic
  identity routing, and builds under executable and `--interface-only`
  `Jobs: 4` compiler modes.
- `compiler_async_optional_pack/app/main.tetra`: imports async helpers with
  optional and throwing optional return signatures plus a synchronous fallback,
  then builds under executable and `--interface-only` `Jobs: 4` compiler modes.
- `compiler_async_memory_pack/app/main.tetra`: imports async helpers that carry
  allocation-base `ptr`, `cap.mem`, and scalar `inout` parameters through
  checked memory operations, plus a synchronous fallback for executable mode.
- `compiler_async_resource_pack/app/main.tetra`: imports async helpers that
  carry optional task, actor, and task-group handles through direct awaited
  returns, plus a synchronous task/group memory fallback for executable mode.
- `compiler_async_throw_resource_pack/app/main.tetra`: imports throwing async
  helpers that carry optional task, actor, and task-group handles through
  direct `try await` returns, plus a synchronous task/group memory fallback.
- `compiler_async_resource_wrapper_pack/app/main.tetra`: imports async helpers
  that carry task, actor, and task-group handles through struct and enum
  aggregate wrappers with direct awaited returns.
- `compiler_async_throw_resource_wrapper_pack/app/main.tetra`: imports
  throwing async helpers that carry task, actor, and task-group handles through
  struct and enum aggregate wrappers with direct `try await` returns.
- `compiler_async_generic_resource_pack/app/main.tetra`: imports generic async
  box helpers that carry scalar, task, actor, and task-group values with direct
  awaited resource returns plus a scalar awaited-local control.
- `compiler_async_throw_generic_resource_pack/app/main.tetra`: imports
  throwing generic async box helpers that carry scalar, task, actor, and
  task-group values with direct `try await` resource returns plus a scalar
  `try await` local control.
- `compiler_async_throw_memory_ptr_pack/app/main.tetra`: imports throwing async
  memory-pointer helpers that validate `try await` local pointer usage and a
  scalar aggregate direct-return control, plus a synchronous memory fallback.
- `compiler_async_optional_memory_ptr_pack/app/main.tetra`: imports async and
  throwing async optional-pointer helpers that validate awaited local unwrapping
  plus scalar optional direct-return controls.
- `compiler_async_generic_memory_ptr_pack/app/main.tetra`: imports generic
  async pointer helpers that validate awaited generic local pointer usage plus
  scalar generic direct-return controls.
- `compiler_async_enum_memory_ptr_pack/app/main.tetra`: imports async enum
  pointer-route helpers that validate awaited local enum unwrapping plus scalar
  enum direct-return controls.
- `compiler_async_slice_memory_pack/app/main.tetra`: imports async slice helpers
  that validate awaited local `[]u8` usage plus scalar direct-return controls.
- `compiler_async_slice_lane_memory_pack/app/main.tetra`: imports async slice
  lane helpers that validate awaited local `[]i32`, `[]bool`, `[]u16`, and
  slice-box usage plus scalar direct-return controls.
- `compiler_async_slice_shape_pack/app/main.tetra`: imports async optional,
  generic, generic-box, and enum-payload `[]i32` helpers with awaited local
  and `try await` local controls.
- `compiler_async_string_memory_pack/app/main.tetra`: imports async string
  helpers that validate awaited local `String` and `StringBox` usage plus
  scalar direct-return controls.
- `compiler_async_optional_generic_string_pack/app/main.tetra`: imports async
  optional and generic `String` helpers that validate awaited local unwrapping
  and generic boxes plus scalar generic direct-return controls.
- `memory_ptr_generic_optional_field_service.tetra`: carries an
  allocation-base pointer through a generic optional field before deriving the
  checked cell after unwrapping.
- `parallel_task_result_generic_optional_field_service.tetra`: carries a joined
  task result through a generic optional field and unwraps it before checking
  the result value.
- `compiler_generic_optional_field_pack/app/main.tetra`: imports generic
  optional-field helpers for pointer bases and task results, combines the
  unwrapped task result with checked raw-memory access, and builds under
  executable and `--interface-only` `Jobs: 4` compiler modes.
- `memory_ptr_generic_optional_call_service.tetra`: routes an allocation-base
  pointer through generic optional-pointer identity and consume helpers before
  deriving the checked cell.
- `memory_optional_ptr_inout_return_service.tetra`: updates an optional pointer
  through an `inout` helper but consumes the explicit returned value as the
  stable writeback workaround.
- `compiler_ptr_optional_generic_call_pack/app/main.tetra`: imports
  optional-pointer generic identity, consume, and inout-return helpers, derives
  checked cells from the returned optional pointer, and builds under executable
  and `--interface-only` `Jobs: 4` compiler modes.
- `parallel_actor_optional_alias_send_service.tetra`: unwraps an optional actor
  handle, routes it through a non-generic alias helper, and sends through the
  returned handle.
- `parallel_task_optional_alias_join_service.tetra`: unwraps an optional task
  handle, routes it through a non-generic alias helper, and joins the returned
  handle.
- `parallel_group_optional_alias_close_service.tetra`: unwraps an optional task
  group, routes it through a non-generic alias helper, then checks status and
  closes the returned handle.
- `compiler_optional_alias_resource_pack/app/main.tetra`: imports optional
  actor, task, and task-group alias helpers and builds under executable and
  `--interface-only` `Jobs: 4` compiler modes.
- `compiler_optional_enum_resource_pack/app/main.tetra`: imports optional enum
  wrappers for actor, task, and task-result payloads, then validates actor
  sends, task joins, and memory offsets under executable and `--interface-only`
  `Jobs: 4` compiler modes.
- `parallel_actor_typed_optional_alias_send_service.tetra`: unwraps an optional
  actor handle, routes it through a non-generic alias helper, and sends a typed
  enum request through the returned handle.
- `parallel_typed_group_optional_alias_spawn_service.tetra`: unwraps an
  optional task group, routes it through a non-generic alias helper, and runs a
  typed grouped task through the returned handle.
- `compiler_typed_optional_alias_resource_pack/app/main.tetra`: imports typed
  actor and typed grouped-task optional alias helpers and builds under
  executable and `--interface-only` `Jobs: 4` compiler modes.
- `parallel_typed_task_match_catch_service.tetra`: routes through a match
  expression, then joins a staged typed task on the inference-only handle path.
- `memory_typed_task_error_offset_service.tetra`: catches typed task error
  payloads, aliases the scalar payloads into local offsets, and uses those
  offsets for checked raw-memory writes.
- `compiler_typed_task_match_pack/app/main.tetra`: imports typed task error
  payload workers, combines match routing with direct and grouped typed task
  joins, and builds under executable and `--interface-only` `Jobs: 4` compiler
  modes.
- `memory_typed_task_error_struct_offset_service.tetra`: catches a typed task
  error carrying a struct payload, aliases the struct field into a local
  offset, and uses it for checked raw-memory access.
- `memory_typed_task_error_nested_enum_offset_service.tetra`: catches a typed
  task error carrying a nested enum payload, resolves it through a helper, and
  uses the result for checked raw-memory access.
- `memory_typed_task_error_optional_offset_service.tetra`: catches a typed task
  error carrying an optional scalar payload, unwraps it through a helper, and
  uses the recovered value as a checked raw-memory offset.
- `memory_typed_task_error_guarded_offset_service.tetra`: catches a typed task
  error through a guarded payload case and uses the selected scalar as a
  checked raw-memory offset.
- `compiler_typed_error_payload_memory_pack/app/main.tetra`: imports typed
  error struct, nested enum, optional, and guarded payload workers, then
  validates their recovered offsets under executable and `--interface-only`
  `Jobs: 4` compiler modes.
- `parallel_defer_group_close_service.tetra`: defers task-group close while a
  grouped worker is spawned and joined through the still-open handle.
- `memory_defer_store_service.tetra`: defers a checked raw-memory write inside
  a nested block and verifies the cell after the block exits.
- `parallel_defer_group_cancel_checkpoint_service.tetra`: cancels a grouped
  worker's current group and runs a checkpoint from a defer cleanup before the
  parent join path observes the worker result.
- `memory_defer_task_result_offset_service.tetra`: joins a task, aliases the
  result value into a local offset, and performs a deferred checked raw-memory
  write through the derived cell.
- `compiler_defer_cleanup_pack/app/main.tetra`: imports deferred cleanup
  workers and validates grouped close, cancel/checkpoint cleanup, and
  task-result memory offsets under executable and `--interface-only` `Jobs: 4`
  compiler modes.
- `memory_defer_throw_base_store_service.tetra`: passes an allocation base into
  a throwing helper and verifies a deferred raw-memory write runs during
  typed-error unwind.
- `memory_defer_return_base_store_service.tetra`: passes an allocation base into
  a helper and verifies a deferred raw-memory write runs before an early return.
- `parallel_typed_task_defer_actor_reply_service.tetra`: sends an actor reply
  from a typed task worker's defer cleanup while the worker throws a typed
  error.
- `compiler_defer_unwind_pack/app/main.tetra`: imports typed-error unwind,
  return cleanup, and typed-task actor cleanup helpers and validates their
  memory/actor routes under executable and `--interface-only` `Jobs: 4`
  compiler modes.
- `memory_typed_error_optional_ptr_base_service.tetra`: carries an
  allocation-base pointer through an optional typed-error payload and derives
  the checked cell after unwrapping.
- `memory_typed_error_optional_ptr_dynamic_service.tetra`: unwraps an optional
  allocation-base pointer recovered from a typed error and uses it in a dynamic
  byte-window loop.
- `compiler_typed_error_optional_ptr_pack/app/main.tetra`: imports an optional
  pointer typed-error helper, recovers the allocation-base pointer, and builds
  under executable and `--interface-only` `Jobs: 4` compiler modes.
- `memory_actor_typed_payload_offset_service.tetra`: receives a scalar typed
  actor enum payload and uses the case binding as a checked raw-memory offset.
- `memory_actor_typed_struct_payload_offset_service.tetra`: receives a typed
  actor struct payload, aliases its scalar field into a local offset, and uses
  that local for checked raw-memory access.
- `compiler_typed_actor_payload_memory_pack/app/main.tetra`: imports typed
  actor struct-payload messages, aliases the received payload field into a
  memory offset, and builds under executable and `--interface-only` `Jobs: 4`
  compiler modes.
- `memory_actor_typed_enum_payload_offset_service.tetra`: receives a nested
  enum payload through a typed actor mailbox and uses the inner scalar case
  binding as a checked raw-memory offset.
- `memory_actor_typed_enum_struct_payload_offset_service.tetra`: receives a
  nested enum payload containing a struct, aliases the struct field into a
  local offset, and uses that local for checked memory.
- `compiler_typed_actor_enum_payload_memory_pack/app/main.tetra`: imports typed
  actor nested enum/struct payload messages, aliases the received field into a
  memory offset, and builds under executable and `--interface-only` `Jobs: 4`
  compiler modes.
- `backend_postgres_session_state_service.tetra`: builds PostgreSQL startup,
  simple query, describe, execute, sync, terminate, and ready-for-query
  session-state frames through the stable backend wire helpers.
- `backend_postgres_cstring_bounds_guard_service.tetra`: verifies PostgreSQL
  bounded C-string scanning returns `-1` for empty, reversed, and negative
  search ranges instead of reading outside the caller-owned payload.
- `backend_postgres_cstring_nul_guard_service.tetra`: verifies PostgreSQL
  C-string length and writer helpers reject embedded NUL bytes before writing
  startup, query, statement, portal, or low-level C-string fields.
- `backend_postgres_data_row_length_guard_service.tetra`: verifies PostgreSQL
  signed DataRow length helpers normalize malformed negative length fields to
  `-1` while preserving valid and NULL column controls.
- `backend_postgres_ascii_i32_bounds_guard_service.tetra`: verifies PostgreSQL
  bounded ASCII integer parsing returns `0` for negative or empty ranges while
  preserving signed, unsigned, and non-digit controls.
- `backend_postgres_ascii_i32_min_guard_service.tetra`: verifies PostgreSQL
  bounded ASCII integer parsing preserves `-2147483648` through both direct
  parser and DataRow integer paths while preserving the positive max control.
- `backend_postgres_ascii_i32_overflow_guard_service.tetra`: verifies
  PostgreSQL bounded ASCII integer parsing returns `0` for out-of-range
  positive and negative i32 text while preserving max/min boundary controls.
- `backend_postgres_command_tag_bounds_guard_service.tetra`: verifies
  PostgreSQL CommandComplete affected-row parsing returns `0` for negative or
  empty ranges while preserving `INSERT` and `UPDATE` count controls.
- `backend_postgres_command_tag_overflow_guard_service.tetra`: verifies
  PostgreSQL CommandComplete affected-row parsing returns `0` for out-of-range
  counts while preserving `2147483647` boundary controls.
- `backend_postgres_command_tag_trailing_guard_service.tetra`: verifies
  PostgreSQL CommandComplete affected-row parsing only returns a digit run
  that actually trails the bounded tag payload.
- `backend_postgres_parse_count_guard_service.tetra`: verifies PostgreSQL
  Parse frame sizing and writing reject parameter type counts above the signed
  i16 protocol range while preserving small valid Parse frames.
- `backend_postgres_row_description_bounds_guard_service.tetra`: verifies
  PostgreSQL RowDescription type-OID scanning returns `-1` for negative,
  empty, truncated, or out-of-range metadata requests.
- `backend_postgres_data_row_bounds_guard_service.tetra`: verifies PostgreSQL
  DataRow value length/start helpers return `-1` for negative starts and
  missing columns while preserving valid integer columns.
- `backend_postgres_data_row_truncated_value_guard_service.tetra`: verifies
  PostgreSQL DataRow value helpers reject advertised positive lengths whose
  value bytes are not physically present in the caller-owned buffer.
- `backend_postgres_parser_short_guard_service.tetra`: verifies PostgreSQL
  bounded C-string, ASCII integer, and CommandComplete parsers return sentinel
  values for overstated limits/counts while preserving valid parser controls.
- `backend_postgres_frame_header_bounds_guard_service.tetra`: verifies
  PostgreSQL frame header readers return `-1` for negative starts and
  malformed short length fields while preserving valid Sync frame controls.
- `backend_postgres_frame_signed_length_guard_service.tetra`: verifies
  PostgreSQL frame length readers normalize malformed negative signed length
  fields to `-1` while preserving valid Sync and short-length controls.
- `backend_postgres_frame_total_overflow_guard_service.tetra`: verifies
  PostgreSQL frame total-length readers reject signed maximum length fields
  whose total byte count would overflow `Int`.
- `backend_postgres_frame_short_guard_service.tetra`: verifies PostgreSQL
  frame header readers return `-1` for empty, tag-only, and truncated length
  buffers while preserving valid Sync frame controls.
- `backend_postgres_ready_status_bounds_guard_service.tetra`: verifies
  PostgreSQL ReadyForQuery status reads return `-1` for negative starts while
  preserving idle, in-transaction, and failed-transaction status bytes.
- `backend_postgres_ready_status_short_guard_service.tetra`: verifies
  PostgreSQL ReadyForQuery status reads return `-1` for empty and offset-short
  payloads while preserving idle, in-transaction, and failed-transaction status
  bytes.
- `backend_postgres_column_count_bounds_guard_service.tetra`: verifies
  PostgreSQL RowDescription and DataRow column-count readers return `-1` for
  negative starts while preserving valid count fields.
- `backend_postgres_column_count_signed_guard_service.tetra`: verifies
  PostgreSQL RowDescription and DataRow column-count readers reject high-bit
  signed i16 count fields while preserving valid one-column payloads.
- `backend_postgres_read_bounds_guard_service.tetra`: verifies PostgreSQL
  big-endian i32/i16 readers return `-1` for negative starts while preserving
  valid unsigned and signed read controls.
- `backend_postgres_read_short_guard_service.tetra`: verifies PostgreSQL
  big-endian i32/i16 readers return `-1` for empty and truncated buffers while
  preserving valid unsigned and signed read controls.
- `backend_postgres_high_bit_read_guard_service.tetra`: verifies PostgreSQL
  big-endian i32 readers return `-1` for high-bit/unrepresentable values while
  preserving the `2147483647` positive control.
- `backend_postgres_write_bounds_guard_service.tetra`: verifies PostgreSQL
  big-endian i32/i16 writers return `-1` for negative starts while preserving
  valid writes and existing buffer contents.
- `backend_postgres_signed_write_guard_service.tetra`: verifies PostgreSQL
  big-endian i32/i16 writers emit two's-complement bytes for negative values
  while preserving positive write controls.
- `backend_postgres_write_short_guard_service.tetra`: verifies PostgreSQL
  big-endian i32/i16 writers return `-1` for empty and truncated buffers while
  preserving valid writes and existing buffer contents.
- `backend_postgres_text_write_bounds_guard_service.tetra`: verifies
  PostgreSQL ASCII and C-string writers return `-1` for negative starts while
  preserving valid text writes and existing buffer contents.
- `backend_postgres_text_write_short_guard_service.tetra`: verifies
  PostgreSQL ASCII and C-string writers return `-1` for empty and truncated
  buffers while preserving valid text writes and existing buffer contents.
- `backend_postgres_frame_writer_short_guard_service.tetra`: verifies
  PostgreSQL startup, simple-query, extended-query, Sync, and Terminate frame
  writers return `-1` for empty and truncated buffers while preserving valid
  frames and existing buffer contents.
- `backend_http_writer_bounds_guard_service.tetra`: verifies HTTP ASCII, CRLF,
  header, and decimal writers return `-1` for negative starts while preserving
  valid writes and existing buffer contents.
- `backend_http_writer_short_guard_service.tetra`: verifies HTTP ASCII, CRLF,
  header, and decimal writers return `-1` for empty and truncated buffers
  while preserving valid writes and existing buffer contents.
- `backend_http_response_writer_short_guard_service.tetra`: verifies HTTP
  response-head, plaintext-response, and JSON-response writers return `-1` for
  empty or truncated buffers while preserving valid responses and existing
  buffer contents.
- `backend_http_negative_content_length_guard_service.tetra`: verifies HTTP
  response-head sizing and writing reject negative `Content-Length` values
  while preserving zero and positive length controls.
- `backend_http_status_code_guard_service.tetra`: verifies HTTP response-head
  sizing and writing reject non-three-digit status codes while preserving
  `100`, `200`, and `999` controls.
- `backend_http_header_injection_guard_service.tetra`: verifies HTTP header and
  response-head writers reject CR/LF header injection in names, values, reason
  text, and response header fields.
- `backend_http_header_control_guard_service.tetra`: verifies HTTP header and
  response-head writers reject non-HTAB control bytes in values, reason text,
  and response header fields while preserving valid HTAB values.
- `backend_http_status_matrix_service.tetra`: checks HTTP status head
  serialization, decimal byte writes, route path character tables, and
  case-insensitive `Connection: close` keep-alive detection.
- `backend_http_json_i32_min_guard_service.tetra`: verifies HTTP and JSON i32
  decimal digit helpers plus HTTP decimal byte writes preserve `-2147483648`.
- `backend_http_header_whitespace_service.tetra`: verifies string and
  byte-buffer keep-alive detection for exact, multi-space, and tabbed
  `Connection: close` headers.
- `backend_http_connection_list_service.tetra`: verifies string and byte-buffer
  keep-alive detection for comma-separated `Connection` token lists such as
  `keep-alive, close` and `upgrade, Close`.
- `backend_http_connection_scope_service.tetra`: verifies `Connection: close`
  is scoped to the actual `Connection` header and does not trigger from
  `X-Connection`, `Proxy-Connection`, or `Connection-Mode`.
- `backend_http_connection_token_boundary_service.tetra`: verifies `close`
  only disables keep-alive when it is a complete `Connection` token, not a
  prefix inside `closex` or `close-upgrade`.
- `backend_http_version_scope_service.tetra`: verifies HTTP/1.1 detection is
  scoped to the request line and rejects header-only `HTTP/1.1` markers or
  `HTTP/1.10` version prefixes.
- `backend_http_request_target_guard_service.tetra`: verifies malformed empty
  or query-only request targets are bad requests while `/` and `/?query`
  remain valid not-found targets.
- `backend_http_request_target_char_guard_service.tetra`: verifies HTTP
  request-target scanning rejects tab/control target bytes as bad requests
  while preserving valid query-string target controls.
- `backend_http_request_line_token_guard_service.tetra`: verifies HTTP/1.1
  detection only accepts the version as the third request-line token and
  rejects extra tokens such as `GET /json debug HTTP/1.1`.
- `backend_http_request_crlf_guard_service.tetra`: verifies HTTP request-line
  detection rejects LF-only and bare-CR `HTTP/1.1` terminators while preserving
  valid CRLF request controls.
- `backend_http_request_short_guard_service.tetra`: verifies byte-buffer HTTP
  request scanners return sentinel values for empty, offset-short, and
  overstated request windows while preserving valid pipelined request controls.
- `backend_http_keep_alive_target_guard_service.tetra`: verifies keep-alive
  detection rejects malformed request targets such as `noslash` and
  query-only targets while preserving `/`, `/?query`, and `/json` controls.
- `backend_http_connection_body_scope_service.tetra`: verifies
  `Connection: close` only disables keep-alive while it appears in the header
  section, not in the body after the empty header terminator.
- `backend_http_keep_alive_method_guard_service.tetra`: verifies keep-alive
  detection rejects malformed method tokens containing separators or tabs
  while preserving valid `GET` and `POST` request-lines.
- `backend_json_control_matrix_service.tetra`: validates JSON string length and
  byte serialization for tab, carriage-return, newline, empty-message objects,
  signed integer length helpers, and lowercase hex digit helpers.
- `backend_json_hex_digit_guard_service.tetra`: verifies public lowercase hex
  digit helper bounds for negative and over-15 inputs while preserving message
  object serialization controls.
- `backend_json_writer_bounds_guard_service.tetra`: verifies JSON string and
  message-object writers return `-1` for negative starts while preserving valid
  escaped string and object serialization.
- `backend_json_writer_short_guard_service.tetra`: verifies JSON string and
  message-object writers return `-1` for empty and truncated buffers while
  preserving valid escaped string and object serialization.
- `backend_net_epoll_event_bounds_guard_service.tetra`: verifies epoll event
  fd/flag extractors return `-1` for missing slots while preserving valid
  event flag controls.
- `backend_net_port_bounds_guard_service.tetra`: verifies TCP loopback bind
  rejects negative and above-range ports while preserving ephemeral port `0`.
- `backend_network_backoff_overflow_guard_service.tetra`: verifies retry backoff
  caps are honored before i32 doubling overflow and ordinary retry controls stay
  stable.
- `backend_crypto_mix_min_guard_service.tetra`: verifies crypto seed mixing stays
  non-negative when `seed * 33 + value` reaches the i32 minimum value.
- `backend_filesystem_nul_exists_guard_service.tetra`: verifies filesystem
  existence checks reject embedded-NUL paths instead of checking only the host
  prefix before the NUL byte.
- `backend_time_overflow_guard_service.tetra`: verifies duration helpers
  saturate positive millisecond overflow to `Int` max and clamp negative
  duration underflow to `0`.
- `backend_time_negative_base_delta_guard_service.tetra`: verifies duration
  addition applies positive deltas to negative bases before clamping the summed
  result, while preserving underflow and overflow controls.

Each program is an independent Linux-x64 entrypoint and exits `0` when its
service contract passes.

Focused verification:

```sh
go test ./compiler -run TestMicroserviceExamplesAndBugLedger -count=1
```
