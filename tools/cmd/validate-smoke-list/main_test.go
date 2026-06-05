package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func nativeSmokeListForTest(omit string) []byte {
	cases := []struct {
		name         string
		srcPath      string
		targetGroup  string
		expectedExit int
	}{
		{"flow_hello", "examples/flow_hello.tetra", "native", 0},
		{"actors_pingpong", "examples/actors_pingpong.tetra", "native", 0},
		{"enum_match_smoke", "examples/enum_match_smoke.tetra", "native", 42},
		{"effects_io_smoke", "examples/effects_io_smoke.tetra", "native", 0},
		{"typed_errors_smoke", "examples/typed_errors_smoke.tetra", "native", 42},
		{"protocol_impl_smoke", "examples/protocol_impl_smoke.tetra", "native", 42},
		{"for_collection_smoke", "examples/for_collection_smoke.tetra", "native", 42},
		{"core_async_smoke", "examples/core_async_smoke.tetra", "native", 42},
		{"core_capability_smoke", "examples/core_capability_smoke.tetra", "native", 42},
		{"core_collections_smoke", "examples/core_collections_smoke.tetra", "native", 42},
		{"core_component_smoke", "examples/core_component_smoke.tetra", "native", 42},
		{"core_crypto_smoke", "examples/core_crypto_smoke.tetra", "native", 42},
		{"core_filesystem_smoke", "examples/core_filesystem_smoke.tetra", "native", 42},
		{"core_io_smoke", "examples/core_io_smoke.tetra", "native", 42},
		{"core_math_smoke", "examples/core_math_smoke.tetra", "native", 42},
		{"core_memory_smoke", "examples/core_memory_smoke.tetra", "native", 42},
		{"core_networking_smoke", "examples/core_networking_smoke.tetra", "native", 42},
		{"core_serialization_smoke", "examples/core_serialization_smoke.tetra", "native", 42},
		{"core_slices_smoke", "examples/core_slices_smoke.tetra", "native", 42},
		{"core_strings_smoke", "examples/core_strings_smoke.tetra", "native", 42},
		{"core_sync_smoke", "examples/core_sync_smoke.tetra", "native", 42},
		{"core_testing_smoke", "examples/core_testing_smoke.tetra", "native", 42},
		{"core_time_smoke", "examples/core_time_smoke.tetra", "native", 42},
		{"surface_counter", "examples/surface_counter.tetra", "native", 1},
		{"surface_text_input", "examples/surface_text_input.tetra", "native", 42},
		{"surface_migration_ui_web_smoke", "examples/surface_migration_ui_web_smoke.tetra", "native", 2},
		{"surface_migration_ui_native_shell_smoke", "examples/surface_migration_ui_native_shell_smoke.tetra", "native", 11},
		{"surface_migration_dogfood_web_ui", "examples/surface_migration_dogfood_web_ui.tetra", "native", 3},
		{"surface_migration_tetra_control_center", "examples/surface_migration_tetra_control_center.tetra", "native", 5},
	}
	for i := 1; len(cases) < 40; i++ {
		name := fmt.Sprintf("filler_case_%02d", i)
		cases = append(cases, struct {
			name         string
			srcPath      string
			targetGroup  string
			expectedExit int
		}{name: name, srcPath: fmt.Sprintf("examples/%s.tetra", name), targetGroup: "native", expectedExit: 0})
	}
	var b strings.Builder
	b.WriteString(`{"target":"linux-x64","total":`)
	total := len(cases)
	if omit != "" {
		total--
	}
	b.WriteString(fmt.Sprint(total))
	b.WriteString(`,"islands_debug":false,"cases":[`)
	first := true
	for _, c := range cases {
		if c.name == omit {
			continue
		}
		if !first {
			b.WriteByte(',')
		}
		first = false
		fmt.Fprintf(&b, `{"name":%q,"src_path":%q,"target_group":%q,"expected_exit":%d}`, c.name, c.srcPath, c.targetGroup, c.expectedExit)
	}
	b.WriteString(`]}`)
	return []byte(b.String())
}

func TestValidateSmokeListAcceptsExpectedShape(t *testing.T) {
	raw := nativeSmokeListForTest("")
	if err := validateSmokeList(raw); err != nil {
		t.Fatalf("validate smoke list: %v", err)
	}
}

func TestValidateSmokeListAcceptsWASMBuildOnlyProfile(t *testing.T) {
	raw := []byte(`{
  "target": "wasm32-web",
  "build_only": true,
  "run_supported": false,
  "total": 15,
  "islands_debug": false,
  "cases": [
    {"name":"legacy_hello","src_path":"examples/hello.tetra","target_group":"wasm","expected_exit":0},
    {"name":"effects_io_smoke","src_path":"examples/effects_io_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"ui_web_smoke","src_path":"examples/ui_web_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"core_slices_smoke","src_path":"examples/core_slices_smoke.tetra","target_group":"wasm","expected_exit":42},
    {"name":"wasm_globals_smoke","src_path":"examples/wasm_globals_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"surface_counter","src_path":"examples/surface_counter.tetra","target_group":"wasm","expected_exit":1},
    {"name":"surface_text_input","src_path":"examples/surface_text_input.tetra","target_group":"wasm","expected_exit":42},
    {"name":"wasm_multi_return_2_smoke","src_path":"examples/wasm_multi_return_2_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"wasm_multi_return_3_smoke","src_path":"examples/wasm_multi_return_3_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"wasm_multi_return_4_smoke","src_path":"examples/wasm_multi_return_4_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"dogfood_wasi","src_path":"examples/projects/dogfood_wasi/src/main.tetra","target_group":"wasm","expected_exit":0},
    {"name":"dogfood_web_ui","src_path":"examples/projects/dogfood_web_ui/src/main.tetra","target_group":"wasm","expected_exit":0},
    {"name":"time_sleep_smoke","src_path":"examples/time_sleep_smoke.tetra","target_group":"wasm","expected_exit":0,"unsupported":true,"expected_diagnostic":"runtime not supported on wasm32"},
    {"name":"task_smoke","src_path":"examples/task_smoke.tetra","target_group":"wasm","expected_exit":42,"unsupported":true,"expected_diagnostic":"runtime not supported on wasm32"},
    {"name":"actors_pingpong","src_path":"examples/actors_pingpong.tetra","target_group":"wasm","expected_exit":0,"unsupported":true,"expected_diagnostic":"runtime not supported on wasm32"}
  ]
}`)
	if err := validateSmokeList(raw); err != nil {
		t.Fatalf("validate wasm smoke list: %v", err)
	}
}

func TestValidateSmokeListRejectsUnknownFields(t *testing.T) {
	raw := []byte(`{
  "target": "wasm32-web",
  "build_only": true,
  "run_supported": false,
  "total": 14,
  "islands_debug": false,
  "cases": [
    {"name":"legacy_hello","src_path":"examples/hello.tetra","target_group":"wasm","expected_exit":0},
    {"name":"effects_io_smoke","src_path":"examples/effects_io_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"ui_web_smoke","src_path":"examples/ui_web_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"dogfood_wasi","src_path":"examples/projects/dogfood_wasi/src/main.tetra","target_group":"wasm","expected_exit":0},
    {"name":"dogfood_web_ui","src_path":"examples/projects/dogfood_web_ui/src/main.tetra","target_group":"wasm","expected_exit":0}
  ],
  "extra": true
}`)
	if err := validateSmokeList(raw); err == nil {
		t.Fatalf("expected unknown top-level field failure")
	}
	raw = []byte(`{
  "target": "wasm32-web",
  "build_only": true,
  "run_supported": false,
  "total": 14,
  "islands_debug": false,
  "cases": [
    {"name":"legacy_hello","src_path":"examples/hello.tetra","target_group":"wasm","expected_exit":0,"extra":true},
    {"name":"effects_io_smoke","src_path":"examples/effects_io_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"ui_web_smoke","src_path":"examples/ui_web_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"dogfood_wasi","src_path":"examples/projects/dogfood_wasi/src/main.tetra","target_group":"wasm","expected_exit":0},
    {"name":"dogfood_web_ui","src_path":"examples/projects/dogfood_web_ui/src/main.tetra","target_group":"wasm","expected_exit":0}
  ]
}`)
	if err := validateSmokeList(raw); err == nil {
		t.Fatalf("expected unknown nested field failure")
	}
}

func TestValidateSmokeListRejectsMissingRequiredCase(t *testing.T) {
	raw := []byte(`{"total":1,"cases":[{"name":"flow_hello","src_path":"examples/flow_hello.tetra"}]}`)
	if err := validateSmokeList(raw); err == nil {
		t.Fatalf("expected missing required case failure")
	}
}

func TestValidateSmokeListRejectsMissingCoreStdlibCase(t *testing.T) {
	raw := nativeSmokeListForTest("core_crypto_smoke")
	err := validateSmokeList(raw)
	if err == nil {
		t.Fatalf("expected missing core stdlib case failure")
	}
	if !strings.Contains(err.Error(), "core_crypto_smoke") {
		t.Fatalf("missing core stdlib error = %v", err)
	}
}

func TestValidateSmokeListRejectsMissingNativeSurfaceCounter(t *testing.T) {
	raw := nativeSmokeListForTest("surface_counter")
	err := validateSmokeList(raw)
	if err == nil {
		t.Fatalf("expected missing native Surface counter case failure")
	}
	if !strings.Contains(err.Error(), "surface_counter") {
		t.Fatalf("missing native Surface counter error = %v", err)
	}
}

func TestValidateSmokeListRejectsMissingNativeSurfaceTextInput(t *testing.T) {
	raw := nativeSmokeListForTest("surface_text_input")
	err := validateSmokeList(raw)
	if err == nil {
		t.Fatalf("expected missing native Surface text input case failure")
	}
	if !strings.Contains(err.Error(), "surface_text_input") {
		t.Fatalf("missing native Surface text input error = %v", err)
	}
}

func TestValidateSmokeListRejectsMissingWASMSurfaceTextInput(t *testing.T) {
	raw := []byte(`{
  "target": "wasm32-web",
  "build_only": true,
  "run_supported": false,
  "total": 14,
  "islands_debug": false,
  "cases": [
    {"name":"legacy_hello","src_path":"examples/hello.tetra","target_group":"wasm","expected_exit":0},
    {"name":"effects_io_smoke","src_path":"examples/effects_io_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"ui_web_smoke","src_path":"examples/ui_web_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"core_slices_smoke","src_path":"examples/core_slices_smoke.tetra","target_group":"wasm","expected_exit":42},
    {"name":"wasm_globals_smoke","src_path":"examples/wasm_globals_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"surface_counter","src_path":"examples/surface_counter.tetra","target_group":"wasm","expected_exit":1},
    {"name":"wasm_multi_return_2_smoke","src_path":"examples/wasm_multi_return_2_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"wasm_multi_return_3_smoke","src_path":"examples/wasm_multi_return_3_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"wasm_multi_return_4_smoke","src_path":"examples/wasm_multi_return_4_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"dogfood_wasi","src_path":"examples/projects/dogfood_wasi/src/main.tetra","target_group":"wasm","expected_exit":0},
    {"name":"dogfood_web_ui","src_path":"examples/projects/dogfood_web_ui/src/main.tetra","target_group":"wasm","expected_exit":0},
    {"name":"time_sleep_smoke","src_path":"examples/time_sleep_smoke.tetra","target_group":"wasm","expected_exit":0,"unsupported":true,"expected_diagnostic":"runtime not supported on wasm32"},
    {"name":"task_smoke","src_path":"examples/task_smoke.tetra","target_group":"wasm","expected_exit":42,"unsupported":true,"expected_diagnostic":"runtime not supported on wasm32"},
    {"name":"actors_pingpong","src_path":"examples/actors_pingpong.tetra","target_group":"wasm","expected_exit":0,"unsupported":true,"expected_diagnostic":"runtime not supported on wasm32"}
  ]
}`)
	err := validateSmokeList(raw)
	if err == nil {
		t.Fatalf("expected missing wasm Surface text input case failure")
	}
	if !strings.Contains(err.Error(), "surface_text_input") {
		t.Fatalf("missing wasm Surface text input error = %v", err)
	}
}

func TestValidateSmokeListRejectsDebugOnlyWithoutFlag(t *testing.T) {
	raw := []byte(`{
  "total": 39,
  "cases": [
    {"name":"flow_hello","src_path":"examples/flow_hello.tetra"},
    {"name":"actors_pingpong","src_path":"examples/actors_pingpong.tetra"},
    {"name":"enum_match_smoke","src_path":"examples/enum_match_smoke.tetra"},
    {"name":"effects_io_smoke","src_path":"examples/effects_io_smoke.tetra"},
    {"name":"typed_errors_smoke","src_path":"examples/typed_errors_smoke.tetra"},
    {"name":"protocol_impl_smoke","src_path":"examples/protocol_impl_smoke.tetra"},
    {"name":"core_memory_smoke","src_path":"examples/core_memory_smoke.tetra"},
    {"name":"for_collection_smoke","src_path":"examples/for_collection_smoke.tetra"},
    {"name":"islands_double_free","src_path":"examples/islands_double_free.tetra","debug_only":true},
    {"name":"case10","src_path":"examples/10.tetra"},
    {"name":"case11","src_path":"examples/11.tetra"},
    {"name":"case12","src_path":"examples/12.tetra"},
    {"name":"case13","src_path":"examples/13.tetra"},
    {"name":"case14","src_path":"examples/14.tetra"},
    {"name":"case15","src_path":"examples/15.tetra"},
    {"name":"case16","src_path":"examples/16.tetra"},
    {"name":"case17","src_path":"examples/17.tetra"},
    {"name":"case18","src_path":"examples/18.tetra"},
    {"name":"case19","src_path":"examples/19.tetra"},
    {"name":"case20","src_path":"examples/20.tetra"},
    {"name":"case21","src_path":"examples/21.tetra"},
    {"name":"case22","src_path":"examples/22.tetra"},
    {"name":"case23","src_path":"examples/23.tetra"},
    {"name":"case24","src_path":"examples/24.tetra"},
    {"name":"case25","src_path":"examples/25.tetra"},
    {"name":"case26","src_path":"examples/26.tetra"},
    {"name":"case27","src_path":"examples/27.tetra"},
    {"name":"case28","src_path":"examples/28.tetra"},
    {"name":"case29","src_path":"examples/29.tetra"},
    {"name":"case30","src_path":"examples/30.tetra"},
    {"name":"case31","src_path":"examples/31.tetra"},
    {"name":"case32","src_path":"examples/32.tetra"},
    {"name":"case33","src_path":"examples/33.tetra"},
    {"name":"case34","src_path":"examples/34.tetra"},
    {"name":"case35","src_path":"examples/35.tetra"},
    {"name":"case36","src_path":"examples/36.tetra"},
    {"name":"case37","src_path":"examples/37.tetra"},
    {"name":"case38","src_path":"examples/38.tetra"},
    {"name":"case39","src_path":"examples/39.tetra"}
  ]
}`)
	if err := validateSmokeList(raw); err == nil {
		t.Fatalf("expected debug-only failure")
	}
}

func TestValidateSmokeListRejectsDuplicateSourcePath(t *testing.T) {
	raw := []byte(`{
  "total": 39,
  "islands_debug": false,
  "cases": [
    {"name":"flow_hello","src_path":"examples/flow_hello.tetra","expected_exit":0},
    {"name":"actors_pingpong","src_path":"examples/flow_hello.tetra","expected_exit":0},
    {"name":"enum_match_smoke","src_path":"examples/enum_match_smoke.tetra","expected_exit":42},
    {"name":"effects_io_smoke","src_path":"examples/effects_io_smoke.tetra","expected_exit":0},
    {"name":"typed_errors_smoke","src_path":"examples/typed_errors_smoke.tetra","expected_exit":42},
    {"name":"protocol_impl_smoke","src_path":"examples/protocol_impl_smoke.tetra","expected_exit":42},
    {"name":"core_memory_smoke","src_path":"examples/core_memory_smoke.tetra","expected_exit":42},
    {"name":"for_collection_smoke","src_path":"examples/for_collection_smoke.tetra","expected_exit":42},
    {"name":"case09","src_path":"examples/09.tetra","expected_exit":0},
    {"name":"case10","src_path":"examples/10.tetra","expected_exit":0},
    {"name":"case11","src_path":"examples/11.tetra","expected_exit":0},
    {"name":"case12","src_path":"examples/12.tetra","expected_exit":0},
    {"name":"case13","src_path":"examples/13.tetra","expected_exit":0},
    {"name":"case14","src_path":"examples/14.tetra","expected_exit":0},
    {"name":"case15","src_path":"examples/15.tetra","expected_exit":0},
    {"name":"case16","src_path":"examples/16.tetra","expected_exit":0},
    {"name":"case17","src_path":"examples/17.tetra","expected_exit":0},
    {"name":"case18","src_path":"examples/18.tetra","expected_exit":0},
    {"name":"case19","src_path":"examples/19.tetra","expected_exit":0},
    {"name":"case20","src_path":"examples/20.tetra","expected_exit":0},
    {"name":"case21","src_path":"examples/21.tetra","expected_exit":0},
    {"name":"case22","src_path":"examples/22.tetra","expected_exit":0},
    {"name":"case23","src_path":"examples/23.tetra","expected_exit":0},
    {"name":"case24","src_path":"examples/24.tetra","expected_exit":0},
    {"name":"case25","src_path":"examples/25.tetra","expected_exit":0},
    {"name":"case26","src_path":"examples/26.tetra","expected_exit":0},
    {"name":"case27","src_path":"examples/27.tetra","expected_exit":0},
    {"name":"case28","src_path":"examples/28.tetra","expected_exit":0},
    {"name":"case29","src_path":"examples/29.tetra","expected_exit":0},
    {"name":"case30","src_path":"examples/30.tetra","expected_exit":0},
    {"name":"case31","src_path":"examples/31.tetra","expected_exit":0},
    {"name":"case32","src_path":"examples/32.tetra","expected_exit":0},
    {"name":"case33","src_path":"examples/33.tetra","expected_exit":0},
    {"name":"case34","src_path":"examples/34.tetra","expected_exit":0},
    {"name":"case35","src_path":"examples/35.tetra","expected_exit":0},
    {"name":"case36","src_path":"examples/36.tetra","expected_exit":0},
    {"name":"case37","src_path":"examples/37.tetra","expected_exit":0},
    {"name":"case38","src_path":"examples/38.tetra","expected_exit":0},
    {"name":"case39","src_path":"examples/39.tetra","expected_exit":0}
  ]
}`)
	if err := validateSmokeList(raw); err == nil {
		t.Fatalf("expected duplicate source failure")
	}
}

func TestValidateSmokeListRejectsInvalidExitCode(t *testing.T) {
	raw := []byte(`{
  "total": 39,
  "islands_debug": false,
  "cases": [
    {"name":"flow_hello","src_path":"examples/flow_hello.tetra","expected_exit":300},
    {"name":"actors_pingpong","src_path":"examples/actors_pingpong.tetra","expected_exit":0},
    {"name":"enum_match_smoke","src_path":"examples/enum_match_smoke.tetra","expected_exit":42},
    {"name":"effects_io_smoke","src_path":"examples/effects_io_smoke.tetra","expected_exit":0},
    {"name":"typed_errors_smoke","src_path":"examples/typed_errors_smoke.tetra","expected_exit":42},
    {"name":"protocol_impl_smoke","src_path":"examples/protocol_impl_smoke.tetra","expected_exit":42},
    {"name":"core_memory_smoke","src_path":"examples/core_memory_smoke.tetra","expected_exit":42},
    {"name":"for_collection_smoke","src_path":"examples/for_collection_smoke.tetra","expected_exit":42},
    {"name":"case09","src_path":"examples/09.tetra","expected_exit":0},
    {"name":"case10","src_path":"examples/10.tetra","expected_exit":0},
    {"name":"case11","src_path":"examples/11.tetra","expected_exit":0},
    {"name":"case12","src_path":"examples/12.tetra","expected_exit":0},
    {"name":"case13","src_path":"examples/13.tetra","expected_exit":0},
    {"name":"case14","src_path":"examples/14.tetra","expected_exit":0},
    {"name":"case15","src_path":"examples/15.tetra","expected_exit":0},
    {"name":"case16","src_path":"examples/16.tetra","expected_exit":0},
    {"name":"case17","src_path":"examples/17.tetra","expected_exit":0},
    {"name":"case18","src_path":"examples/18.tetra","expected_exit":0},
    {"name":"case19","src_path":"examples/19.tetra","expected_exit":0},
    {"name":"case20","src_path":"examples/20.tetra","expected_exit":0},
    {"name":"case21","src_path":"examples/21.tetra","expected_exit":0},
    {"name":"case22","src_path":"examples/22.tetra","expected_exit":0},
    {"name":"case23","src_path":"examples/23.tetra","expected_exit":0},
    {"name":"case24","src_path":"examples/24.tetra","expected_exit":0},
    {"name":"case25","src_path":"examples/25.tetra","expected_exit":0},
    {"name":"case26","src_path":"examples/26.tetra","expected_exit":0},
    {"name":"case27","src_path":"examples/27.tetra","expected_exit":0},
    {"name":"case28","src_path":"examples/28.tetra","expected_exit":0},
    {"name":"case29","src_path":"examples/29.tetra","expected_exit":0},
    {"name":"case30","src_path":"examples/30.tetra","expected_exit":0},
    {"name":"case31","src_path":"examples/31.tetra","expected_exit":0},
    {"name":"case32","src_path":"examples/32.tetra","expected_exit":0},
    {"name":"case33","src_path":"examples/33.tetra","expected_exit":0},
    {"name":"case34","src_path":"examples/34.tetra","expected_exit":0},
    {"name":"case35","src_path":"examples/35.tetra","expected_exit":0},
    {"name":"case36","src_path":"examples/36.tetra","expected_exit":0},
    {"name":"case37","src_path":"examples/37.tetra","expected_exit":0},
    {"name":"case38","src_path":"examples/38.tetra","expected_exit":0},
    {"name":"case39","src_path":"examples/39.tetra","expected_exit":0}
  ]
}`)
	if err := validateSmokeList(raw); err == nil {
		t.Fatalf("expected invalid exit failure")
	}
}

func TestValidateSmokeListRejectsMissingTargetGroup(t *testing.T) {
	raw := []byte(`{
  "target":"linux-x64",
  "total":39,
  "islands_debug":false,
  "cases":[
    {"name":"flow_hello","src_path":"examples/flow_hello.tetra","expected_exit":0},
    {"name":"actors_pingpong","src_path":"examples/actors_pingpong.tetra","target_group":"native","expected_exit":0},
    {"name":"enum_match_smoke","src_path":"examples/enum_match_smoke.tetra","target_group":"native","expected_exit":42},
    {"name":"effects_io_smoke","src_path":"examples/effects_io_smoke.tetra","target_group":"native","expected_exit":0},
    {"name":"typed_errors_smoke","src_path":"examples/typed_errors_smoke.tetra","target_group":"native","expected_exit":42},
    {"name":"protocol_impl_smoke","src_path":"examples/protocol_impl_smoke.tetra","target_group":"native","expected_exit":42},
    {"name":"core_memory_smoke","src_path":"examples/core_memory_smoke.tetra","target_group":"native","expected_exit":42},
    {"name":"for_collection_smoke","src_path":"examples/for_collection_smoke.tetra","target_group":"native","expected_exit":42},
    {"name":"case09","src_path":"examples/09.tetra","target_group":"native","expected_exit":0},
    {"name":"case10","src_path":"examples/10.tetra","target_group":"native","expected_exit":0},
    {"name":"case11","src_path":"examples/11.tetra","target_group":"native","expected_exit":0},
    {"name":"case12","src_path":"examples/12.tetra","target_group":"native","expected_exit":0},
    {"name":"case13","src_path":"examples/13.tetra","target_group":"native","expected_exit":0},
    {"name":"case14","src_path":"examples/14.tetra","target_group":"native","expected_exit":0},
    {"name":"case15","src_path":"examples/15.tetra","target_group":"native","expected_exit":0},
    {"name":"case16","src_path":"examples/16.tetra","target_group":"native","expected_exit":0},
    {"name":"case17","src_path":"examples/17.tetra","target_group":"native","expected_exit":0},
    {"name":"case18","src_path":"examples/18.tetra","target_group":"native","expected_exit":0},
    {"name":"case19","src_path":"examples/19.tetra","target_group":"native","expected_exit":0},
    {"name":"case20","src_path":"examples/20.tetra","target_group":"native","expected_exit":0},
    {"name":"case21","src_path":"examples/21.tetra","target_group":"native","expected_exit":0},
    {"name":"case22","src_path":"examples/22.tetra","target_group":"native","expected_exit":0},
    {"name":"case23","src_path":"examples/23.tetra","target_group":"native","expected_exit":0},
    {"name":"case24","src_path":"examples/24.tetra","target_group":"native","expected_exit":0},
    {"name":"case25","src_path":"examples/25.tetra","target_group":"native","expected_exit":0},
    {"name":"case26","src_path":"examples/26.tetra","target_group":"native","expected_exit":0},
    {"name":"case27","src_path":"examples/27.tetra","target_group":"native","expected_exit":0},
    {"name":"case28","src_path":"examples/28.tetra","target_group":"native","expected_exit":0},
    {"name":"case29","src_path":"examples/29.tetra","target_group":"native","expected_exit":0},
    {"name":"case30","src_path":"examples/30.tetra","target_group":"native","expected_exit":0},
    {"name":"case31","src_path":"examples/31.tetra","target_group":"native","expected_exit":0},
    {"name":"case32","src_path":"examples/32.tetra","target_group":"native","expected_exit":0},
    {"name":"case33","src_path":"examples/33.tetra","target_group":"native","expected_exit":0},
    {"name":"case34","src_path":"examples/34.tetra","target_group":"native","expected_exit":0},
    {"name":"case35","src_path":"examples/35.tetra","target_group":"native","expected_exit":0},
    {"name":"case36","src_path":"examples/36.tetra","target_group":"native","expected_exit":0},
    {"name":"case37","src_path":"examples/37.tetra","target_group":"native","expected_exit":0},
    {"name":"case38","src_path":"examples/38.tetra","target_group":"native","expected_exit":0},
    {"name":"case39","src_path":"examples/39.tetra","target_group":"native","expected_exit":0}
  ]
}`)
	if err := validateSmokeList(raw); err == nil {
		t.Fatalf("expected missing target_group failure")
	}
}

func TestValidateSmokeListRejectsUnassignedExampleWhenRootProvided(t *testing.T) {
	root := t.TempDir()
	writeExample := func(path string) {
		t.Helper()
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("fun main(): i32 { return 0 }\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeExample("covered.tetra")
	writeExample("missing.t4")

	raw := []byte(`{
  "target": "wasm32-web",
  "build_only": true,
  "total": 15,
  "islands_debug": false,
  "cases": [
    {"name":"legacy_hello","src_path":"examples/hello.tetra","target_group":"wasm","expected_exit":0},
    {"name":"effects_io_smoke","src_path":"examples/effects_io_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"ui_web_smoke","src_path":"examples/ui_web_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"core_slices_smoke","src_path":"examples/core_slices_smoke.tetra","target_group":"wasm","expected_exit":42},
    {"name":"wasm_globals_smoke","src_path":"examples/wasm_globals_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"surface_counter","src_path":"examples/surface_counter.tetra","target_group":"wasm","expected_exit":1},
    {"name":"surface_text_input","src_path":"examples/surface_text_input.tetra","target_group":"wasm","expected_exit":42},
    {"name":"wasm_multi_return_2_smoke","src_path":"examples/wasm_multi_return_2_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"wasm_multi_return_3_smoke","src_path":"examples/wasm_multi_return_3_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"wasm_multi_return_4_smoke","src_path":"examples/wasm_multi_return_4_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"dogfood_wasi","src_path":"examples/projects/dogfood_wasi/src/main.tetra","target_group":"wasm","expected_exit":0},
    {"name":"dogfood_web_ui","src_path":"examples/covered.tetra","target_group":"wasm","expected_exit":0},
    {"name":"time_sleep_smoke","src_path":"examples/time_sleep_smoke.tetra","target_group":"wasm","expected_exit":0,"unsupported":true,"expected_diagnostic":"runtime not supported on wasm32"},
    {"name":"task_smoke","src_path":"examples/task_smoke.tetra","target_group":"wasm","expected_exit":42,"unsupported":true,"expected_diagnostic":"runtime not supported on wasm32"},
    {"name":"actors_pingpong","src_path":"examples/actors_pingpong.tetra","target_group":"wasm","expected_exit":0,"unsupported":true,"expected_diagnostic":"runtime not supported on wasm32"}
  ]
}`)

	if err := validateSmokeListWithExamplesRoot(raw, root); err == nil {
		t.Fatalf("expected uncovered example failure")
	} else if !strings.Contains(err.Error(), "examples/missing.t4") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSmokeListAcceptsDocumentedExampleExclusion(t *testing.T) {
	root := t.TempDir()
	full := filepath.Join(root, "excluded.t4")
	if err := os.WriteFile(full, []byte("func main() -> Int:\n    return 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	raw := []byte(`{
  "target": "wasm32-web",
  "build_only": true,
  "total": 15,
  "islands_debug": false,
  "cases": [
    {"name":"legacy_hello","src_path":"examples/hello.tetra","target_group":"wasm","expected_exit":0},
    {"name":"effects_io_smoke","src_path":"examples/effects_io_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"ui_web_smoke","src_path":"examples/ui_web_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"core_slices_smoke","src_path":"examples/core_slices_smoke.tetra","target_group":"wasm","expected_exit":42},
    {"name":"wasm_globals_smoke","src_path":"examples/wasm_globals_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"surface_counter","src_path":"examples/surface_counter.tetra","target_group":"wasm","expected_exit":1},
    {"name":"surface_text_input","src_path":"examples/surface_text_input.tetra","target_group":"wasm","expected_exit":42},
    {"name":"wasm_multi_return_2_smoke","src_path":"examples/wasm_multi_return_2_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"wasm_multi_return_3_smoke","src_path":"examples/wasm_multi_return_3_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"wasm_multi_return_4_smoke","src_path":"examples/wasm_multi_return_4_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"dogfood_wasi","src_path":"examples/projects/dogfood_wasi/src/main.tetra","target_group":"wasm","expected_exit":0},
    {"name":"dogfood_web_ui","src_path":"examples/projects/dogfood_web_ui/src/main.tetra","target_group":"wasm","expected_exit":0},
    {"name":"time_sleep_smoke","src_path":"examples/time_sleep_smoke.tetra","target_group":"wasm","expected_exit":0,"unsupported":true,"expected_diagnostic":"runtime not supported on wasm32"},
    {"name":"task_smoke","src_path":"examples/task_smoke.tetra","target_group":"wasm","expected_exit":42,"unsupported":true,"expected_diagnostic":"runtime not supported on wasm32"},
    {"name":"actors_pingpong","src_path":"examples/actors_pingpong.tetra","target_group":"wasm","expected_exit":0,"unsupported":true,"expected_diagnostic":"runtime not supported on wasm32"}
  ],
  "excluded_examples": [
    {"src_path":"examples/excluded.t4","reason":"not part of wasm32-web smoke profile"}
  ]
}`)

	if err := validateSmokeListWithExamplesRoot(raw, root); err != nil {
		t.Fatalf("validate with exclusion: %v", err)
	}
}
