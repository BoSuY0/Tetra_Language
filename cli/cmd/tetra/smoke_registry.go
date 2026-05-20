package main

import ctarget "tetra_language/compiler/target"

type smokeSourceSet string

const (
	smokeSourceSetNative            smokeSourceSet = "native"
	smokeSourceSetWasmBuildOnly     smokeSourceSet = "wasm-build-only"
	smokeSourceSetWasmWASIBuildOnly smokeSourceSet = "wasm-wasi-build-only"
)

var smokeCaseRegistry = map[smokeSourceSet][]smokeCase{
	smokeSourceSetNative: {
		{name: "islands_hello", srcPath: "examples/islands_hello.tetra", expectedExit: 0},
		{name: "islands_i32", srcPath: "examples/islands_i32.tetra", expectedExit: 55},
		{name: "islands_overflow", srcPath: "examples/islands_overflow.tetra", expectedExit: 1},
		{name: "mmio_smoke", srcPath: "examples/mmio_smoke.tetra", expectedExit: 123},
		{name: "cap_mem_smoke", srcPath: "examples/cap_mem_smoke.tetra", expectedExit: 77},
		{name: "memset_smoke", srcPath: "examples/memset_smoke.tetra", expectedExit: 88},
		{name: "actors_pingpong", srcPath: "examples/actors_pingpong.tetra", expectedExit: 0},
		{name: "actor_sleep_pingpong", srcPath: "examples/actor_sleep_pingpong.tetra", expectedExit: 0},
		{name: "flow_hello", srcPath: "examples/flow_hello.tetra", expectedExit: 0},
		{name: "flow_struct_smoke", srcPath: "examples/flow_struct_smoke.tetra", expectedExit: 42},
		{name: "flow_islands_smoke", srcPath: "examples/flow_islands_smoke.tetra", expectedExit: 0},
		{name: "flow_unsafe_cap_mem_smoke", srcPath: "examples/flow_unsafe_cap_mem_smoke.tetra", expectedExit: 42},
		{name: "ui_native_shell_smoke", srcPath: "examples/ui_native_shell_smoke.tetra", expectedExit: 0},
		{name: "bool_smoke", srcPath: "examples/bool_smoke.tetra", expectedExit: 42},
		{name: "for_range_smoke", srcPath: "examples/for_range_smoke.tetra", expectedExit: 55},
		{name: "for_collection_smoke", srcPath: "examples/for_collection_smoke.tetra", expectedExit: 42},
		{name: "for_collection_u8_smoke", srcPath: "examples/for_collection_u8_smoke.tetra", expectedExit: 42},
		{name: "loop_control_smoke", srcPath: "examples/loop_control_smoke.tetra", expectedExit: 42},
		{name: "complex_control_flow_smoke", srcPath: "examples/complex_control_flow_smoke.tetra", expectedExit: 42},
		{name: "unary_not_smoke", srcPath: "examples/unary_not_smoke.tetra", expectedExit: 42},
		{name: "const_smoke", srcPath: "examples/const_smoke.tetra", expectedExit: 42},
		{name: "const_bool_smoke", srcPath: "examples/const_bool_smoke.tetra", expectedExit: 42},
		{name: "local_const_smoke", srcPath: "examples/local_const_smoke.tetra", expectedExit: 42},
		{name: "compound_assignment_smoke", srcPath: "examples/compound_assignment_smoke.tetra", expectedExit: 42},
		{name: "else_if_smoke", srcPath: "examples/else_if_smoke.tetra", expectedExit: 42},
		{name: "enum_match_smoke", srcPath: "examples/enum_match_smoke.tetra", expectedExit: 42},
		{name: "enum_exhaustive_match_smoke", srcPath: "examples/enum_exhaustive_match_smoke.tetra", expectedExit: 42},
		{name: "effects_io_smoke", srcPath: "examples/effects_io_smoke.tetra", expectedExit: 0},
		{name: "effects_mem_smoke", srcPath: "examples/effects_mem_smoke.tetra", expectedExit: 17},
		{name: "effects_actors_smoke", srcPath: "examples/effects_actors_smoke.tetra", expectedExit: 0},
		{name: "optional_smoke", srcPath: "examples/optional_smoke.tetra", expectedExit: 42},
		{name: "optional_match_smoke", srcPath: "examples/optional_match_smoke.tetra", expectedExit: 42},
		{name: "optional_match_some_smoke", srcPath: "examples/optional_match_some_smoke.tetra", expectedExit: 42},
		{name: "ownership_smoke", srcPath: "examples/ownership_smoke.tetra", expectedExit: 42},
		{name: "typed_errors_smoke", srcPath: "examples/typed_errors_smoke.tetra", expectedExit: 42},
		{name: "async_smoke", srcPath: "examples/async_smoke.tetra", expectedExit: 42},
		{name: "task_smoke", srcPath: "examples/task_smoke.tetra", expectedExit: 42},
		{name: "time_sleep_smoke", srcPath: "examples/time_sleep_smoke.tetra", expectedExit: 0},
		{name: "task_sleep_deadline_smoke", srcPath: "examples/task_sleep_deadline_smoke.tetra", expectedExit: 0},
		{name: "task_join_wait_smoke", srcPath: "examples/task_join_wait_smoke.tetra", expectedExit: 5},
		{name: "task_group_cancel_smoke", srcPath: "examples/task_group_cancel_smoke.tetra", expectedExit: 1},
		{name: "task_group_lifecycle_smoke", srcPath: "examples/task_group_lifecycle_smoke.tetra", expectedExit: 42},
		{name: "deadline_aware_waits_smoke", srcPath: "examples/deadline_aware_waits_smoke.tetra", expectedExit: 0},
		{name: "wait_composition_smoke", srcPath: "examples/wait_composition_smoke.tetra", expectedExit: 0},
		{name: "core_math_smoke", srcPath: "examples/core_math_smoke.tetra", expectedExit: 42},
		{name: "core_memory_smoke", srcPath: "examples/core_memory_smoke.tetra", expectedExit: 42},
		{name: "core_strings_smoke", srcPath: "examples/core_strings_smoke.tetra", expectedExit: 42},
		{name: "core_slices_smoke", srcPath: "examples/core_slices_smoke.tetra", expectedExit: 42},
		{name: "core_io_smoke", srcPath: "examples/core_io_smoke.tetra", expectedExit: 42},
		{name: "core_testing_smoke", srcPath: "examples/core_testing_smoke.tetra", expectedExit: 42},
		{name: "core_collections_smoke", srcPath: "examples/core_collections_smoke.tetra", expectedExit: 42},
		{name: "core_serialization_smoke", srcPath: "examples/core_serialization_smoke.tetra", expectedExit: 42},
		{name: "core_filesystem_smoke", srcPath: "examples/core_filesystem_smoke.tetra", expectedExit: 42},
		{name: "core_networking_smoke", srcPath: "examples/core_networking_smoke.tetra", expectedExit: 42},
		{name: "core_async_smoke", srcPath: "examples/core_async_smoke.tetra", expectedExit: 42},
		{name: "core_sync_smoke", srcPath: "examples/core_sync_smoke.tetra", expectedExit: 42},
		{name: "core_time_smoke", srcPath: "examples/core_time_smoke.tetra", expectedExit: 42},
		{name: "core_crypto_smoke", srcPath: "examples/core_crypto_smoke.tetra", expectedExit: 42},
		{name: "core_capability_smoke", srcPath: "examples/core_capability_smoke.tetra", expectedExit: 42},
		{name: "extension_smoke", srcPath: "examples/extension_smoke.tetra", expectedExit: 42},
		{name: "generic_smoke", srcPath: "examples/generic_smoke.tetra", expectedExit: 42},
		{name: "protocol_impl_smoke", srcPath: "examples/protocol_impl_smoke.tetra", expectedExit: 42},
		{name: "dogfood_cli", srcPath: "examples/projects/dogfood_cli/src/main.tetra", expectedExit: 0},
		{name: "dogfood_actor_task", srcPath: "examples/projects/dogfood_actor_task/src/main.tetra", expectedExit: 0},
	},
	smokeSourceSetWasmBuildOnly: {
		{name: "legacy_hello", srcPath: "examples/hello.tetra", expectedExit: 0},
		{name: "effects_io_smoke", srcPath: "examples/effects_io_smoke.tetra", expectedExit: 0},
		{name: "ui_web_smoke", srcPath: "examples/ui_web_smoke.tetra", expectedExit: 0},
		{name: "core_slices_smoke", srcPath: "examples/core_slices_smoke.tetra", expectedExit: 0},
		{name: "wasm_globals_smoke", srcPath: "examples/wasm_globals_smoke.tetra", expectedExit: 0},
		{name: "wasm_multi_return_2_smoke", srcPath: "examples/wasm_multi_return_2_smoke.tetra", expectedExit: 0},
		{name: "wasm_multi_return_3_smoke", srcPath: "examples/wasm_multi_return_3_smoke.tetra", expectedExit: 0},
		{name: "wasm_multi_return_4_smoke", srcPath: "examples/wasm_multi_return_4_smoke.tetra", expectedExit: 0},
		{name: "dogfood_wasi", srcPath: "examples/projects/dogfood_wasi/src/main.tetra", expectedExit: 0},
		{name: "dogfood_web_ui", srcPath: "examples/projects/dogfood_web_ui/src/main.tetra", expectedExit: 0},
		{name: "time_sleep_smoke", srcPath: "examples/time_sleep_smoke.tetra", expectedExit: 0, expectedDiagnostic: "runtime not supported on wasm32"},
		{name: "task_smoke", srcPath: "examples/task_smoke.tetra", expectedExit: 42, expectedDiagnostic: "runtime not supported on wasm32"},
		{name: "actors_pingpong", srcPath: "examples/actors_pingpong.tetra", expectedExit: 0, expectedDiagnostic: "runtime not supported on wasm32"},
	},
	smokeSourceSetWasmWASIBuildOnly: {
		{name: "legacy_hello", srcPath: "examples/hello.tetra", expectedExit: 0},
		{name: "effects_io_smoke", srcPath: "examples/effects_io_smoke.tetra", expectedExit: 0},
		{name: "ui_web_smoke", srcPath: "examples/ui_web_smoke.tetra", expectedExit: 0},
		{name: "core_slices_smoke", srcPath: "examples/core_slices_smoke.tetra", expectedExit: 0},
		{name: "wasm_globals_smoke", srcPath: "examples/wasm_globals_smoke.tetra", expectedExit: 0},
		{name: "wasm_multi_return_2_smoke", srcPath: "examples/wasm_multi_return_2_smoke.tetra", expectedExit: 0},
		{name: "wasm_multi_return_3_smoke", srcPath: "examples/wasm_multi_return_3_smoke.tetra", expectedExit: 0},
		{name: "wasm_multi_return_4_smoke", srcPath: "examples/wasm_multi_return_4_smoke.tetra", expectedExit: 0},
		{name: "dogfood_wasi", srcPath: "examples/projects/dogfood_wasi/src/main.tetra", expectedExit: 0},
		{name: "dogfood_web_ui", srcPath: "examples/projects/dogfood_web_ui/src/main.tetra", expectedExit: 0},
		{name: "time_sleep_smoke", srcPath: "examples/time_sleep_smoke.tetra", expectedExit: 0, expectedDiagnostic: "runtime not supported on wasm32"},
		{name: "task_smoke", srcPath: "examples/task_smoke.tetra", expectedExit: 42, expectedDiagnostic: "runtime not supported on wasm32"},
		{name: "actors_pingpong", srcPath: "examples/actors_pingpong.tetra", expectedExit: 0, expectedDiagnostic: "runtime not supported on wasm32"},
	},
}

func smokeRegistryCases(set smokeSourceSet) []smokeCase {
	cases := smokeCaseRegistry[set]
	out := make([]smokeCase, len(cases))
	copy(out, cases)
	return out
}

func smokeCases(islandsDebug bool) []smokeCase {
	return smokeRegistryCases(smokeSourceSetNative)
}

func smokeCasesForTarget(islandsDebug bool, tgt ctarget.Target) []smokeCase {
	if tgt.Triple == "wasm32-wasi" {
		return smokeRegistryCases(smokeSourceSetWasmWASIBuildOnly)
	}
	if tgt.Triple == "wasm32-web" {
		return smokeRegistryCases(smokeSourceSetWasmBuildOnly)
	}
	return smokeCases(islandsDebug)
}
