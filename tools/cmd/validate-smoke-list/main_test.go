package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSmokeListAcceptsExpectedShape(t *testing.T) {
	raw := []byte(`{
  "total": 39,
  "islands_debug": false,
  "cases": [
    {"name":"flow_hello","src_path":"examples/flow_hello.tetra","target_group":"native","expected_exit":0},
    {"name":"actors_pingpong","src_path":"examples/actors_pingpong.tetra","target_group":"native","expected_exit":0},
    {"name":"enum_match_smoke","src_path":"examples/enum_match_smoke.tetra","target_group":"native","expected_exit":42},
    {"name":"effects_io_smoke","src_path":"examples/effects_io_smoke.tetra","target_group":"native","expected_exit":0},
    {"name":"typed_errors_smoke","src_path":"examples/typed_errors_smoke.tetra","target_group":"native","expected_exit":42},
    {"name":"protocol_impl_smoke","src_path":"examples/protocol_impl_smoke.tetra","target_group":"native","expected_exit":42},
    {"name":"core_memory_smoke","src_path":"examples/core_memory_smoke.tetra","target_group":"native","expected_exit":42},
    {"name":"for_collection_smoke","src_path":"examples/for_collection_smoke.tetra","target_group":"native","expected_exit":42},
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
	if err := validateSmokeList(raw); err != nil {
		t.Fatalf("validate smoke list: %v", err)
	}
}

func TestValidateSmokeListAcceptsWASMBuildOnlyProfile(t *testing.T) {
	raw := []byte(`{
  "target": "wasm32-web",
  "build_only": true,
  "run_supported": false,
  "total": 5,
  "islands_debug": false,
  "cases": [
    {"name":"flow_hello","src_path":"examples/flow_hello.tetra","target_group":"wasm","expected_exit":0},
    {"name":"effects_io_smoke","src_path":"examples/effects_io_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"ui_web_smoke","src_path":"examples/ui_web_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"dogfood_wasi","src_path":"examples/projects/dogfood_wasi/src/main.tetra","target_group":"wasm","expected_exit":0},
    {"name":"dogfood_web_ui","src_path":"examples/projects/dogfood_web_ui/src/main.tetra","target_group":"wasm","expected_exit":0}
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
  "total": 5,
  "islands_debug": false,
  "cases": [
    {"name":"flow_hello","src_path":"examples/flow_hello.tetra","target_group":"wasm","expected_exit":0},
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
  "total": 5,
  "islands_debug": false,
  "cases": [
    {"name":"flow_hello","src_path":"examples/flow_hello.tetra","target_group":"wasm","expected_exit":0,"extra":true},
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
	writeExample("missing.tetra")

	raw := []byte(`{
  "target": "wasm32-web",
  "build_only": true,
  "total": 5,
  "islands_debug": false,
  "cases": [
    {"name":"flow_hello","src_path":"examples/flow_hello.tetra","target_group":"wasm","expected_exit":0},
    {"name":"effects_io_smoke","src_path":"examples/effects_io_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"ui_web_smoke","src_path":"examples/ui_web_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"dogfood_wasi","src_path":"examples/projects/dogfood_wasi/src/main.tetra","target_group":"wasm","expected_exit":0},
    {"name":"dogfood_web_ui","src_path":"examples/covered.tetra","target_group":"wasm","expected_exit":0}
  ]
}`)

	if err := validateSmokeListWithExamplesRoot(raw, root); err == nil {
		t.Fatalf("expected uncovered example failure")
	} else if !strings.Contains(err.Error(), "examples/missing.tetra") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSmokeListAcceptsDocumentedExampleExclusion(t *testing.T) {
	root := t.TempDir()
	full := filepath.Join(root, "excluded.tetra")
	if err := os.WriteFile(full, []byte("fun main(): i32 { return 0 }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	raw := []byte(`{
  "target": "wasm32-web",
  "build_only": true,
  "total": 5,
  "islands_debug": false,
  "cases": [
    {"name":"flow_hello","src_path":"examples/flow_hello.tetra","target_group":"wasm","expected_exit":0},
    {"name":"effects_io_smoke","src_path":"examples/effects_io_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"ui_web_smoke","src_path":"examples/ui_web_smoke.tetra","target_group":"wasm","expected_exit":0},
    {"name":"dogfood_wasi","src_path":"examples/projects/dogfood_wasi/src/main.tetra","target_group":"wasm","expected_exit":0},
    {"name":"dogfood_web_ui","src_path":"examples/projects/dogfood_web_ui/src/main.tetra","target_group":"wasm","expected_exit":0}
  ],
  "excluded_examples": [
    {"src_path":"examples/excluded.tetra","reason":"not part of wasm32-web smoke profile"}
  ]
}`)

	if err := validateSmokeListWithExamplesRoot(raw, root); err != nil {
		t.Fatalf("validate with exclusion: %v", err)
	}
}
