package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyEpic14ExampleIndexRejectsMissingGenericStructEntry(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "examples_index.md")
	examples := []string{
		"examples/smoke/basic/hello.tetra",
		"examples/flow/flow_hello.tetra",
		"examples/smoke/scalars/bool_smoke.tetra",
		"examples/smoke/control/for_range_smoke.tetra",
		"examples/smoke/control/for_collection_smoke.tetra",
		"examples/smoke/control/loop_control_smoke.tetra",
		"examples/smoke/scalars/const_smoke.tetra",
		"examples/smoke/scalars/const_bool_smoke.tetra",
		"examples/smoke/scalars/local_const_smoke.tetra",
		"examples/smoke/scalars/compound_assignment_smoke.tetra",
		"examples/smoke/types/enum_match_smoke.tetra",
		"examples/smoke/types/enum_exhaustive_match_smoke.tetra",
		"examples/smoke/types/optional_smoke.tetra",
		"examples/smoke/types/optional_match_smoke.tetra",
		"examples/smoke/errors/typed_errors_smoke.tetra",
		"examples/smoke/language/generic_smoke.tetra",
		"examples/smoke/language/protocol_impl_smoke.tetra",
		"examples/smoke/language/extension_smoke.tetra",
		"examples/memory/ownership/ownership_smoke.tetra",
		"examples/async/async_smoke.tetra",
		"examples/tasks/task_smoke.tetra",
		"examples/actors/actors_pingpong.tetra",
		"examples/memory/islands/islands_hello.tetra",
		"examples/memory/islands/islands_i32.tetra",
		"examples/memory/islands/islands_overflow.tetra",
		"examples/memory/raw/cap_mem_smoke.tetra",
		"examples/memory/raw/mmio_smoke.tetra",
		"examples/memory/raw/memset_smoke.tetra",
		"examples/ui/ui_web_smoke.tetra",
		"examples/ui/ui_native_shell_smoke.tetra",
		"examples/projects/hello_t4/src/main.t4",
		"examples/projects/dogfood_wasi/src/main.tetra",
		"examples/projects/dogfood_web_ui/src/main.tetra",
		"examples/projects/dogfood_cli/src/main.tetra",
		"examples/projects/dogfood_actor_task/src/main.tetra",
		"examples/projects/eco_dogfood/src/main.tetra",
	}
	headings := []string{
		"## Epic 14 Verification Commands",
		"## Troubleshooting Notes (Epic 14)",
		"### Basic language examples (`V020-0701..0705`)",
		"### Control-flow examples (`V020-0706..0710`)",
		"### Const and assignment examples (`V020-0711..0715`)",
		"### Enum/match examples (`V020-0716..0720`)",
		"### Optional/error examples (`V020-0721..0725`)",
		"### Generic/protocol/extension examples (`V020-0726..0730`)",
		"### Safety/runtime examples (`V020-0731..0735`)",
		"### Memory/capability examples (`V020-0736..0740`)",
		"### UI/WASM examples (`V020-0741..0745`)",
		"### Project dogfood examples (`V020-0746..0750`)",
	}

	lines := []string{
		"# Examples Index",
		"",
		"| Example | Purpose | Target group | Expected behavior |",
		"| --- | --- | --- | --- |",
	}
	for _, example := range examples {
		lines = append(lines, "| `"+example+"` | test entry | native | exits 0 |")
	}
	for _, heading := range headings {
		lines = append(lines, "", heading, "", "unsupported profile note", "regression note")
	}

	if err := os.WriteFile(indexPath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	err := verifyEpic14ExampleIndex(indexPath)
	if err == nil {
		t.Fatalf("expected missing generic struct coverage failure")
	}
	if !strings.Contains(err.Error(), "examples/smoke/language/generic_struct_smoke.tetra") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyEpic14ExampleIndexRejectsMissingPrimaryT4ProjectEntry(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "examples_index.md")
	examples := []string{
		"examples/smoke/basic/hello.tetra",
		"examples/flow/flow_hello.tetra",
		"examples/smoke/scalars/bool_smoke.tetra",
		"examples/smoke/control/for_range_smoke.tetra",
		"examples/smoke/control/for_collection_smoke.tetra",
		"examples/smoke/control/loop_control_smoke.tetra",
		"examples/smoke/scalars/const_smoke.tetra",
		"examples/smoke/scalars/const_bool_smoke.tetra",
		"examples/smoke/scalars/local_const_smoke.tetra",
		"examples/smoke/scalars/compound_assignment_smoke.tetra",
		"examples/smoke/types/enum_match_smoke.tetra",
		"examples/smoke/types/enum_exhaustive_match_smoke.tetra",
		"examples/smoke/types/optional_smoke.tetra",
		"examples/smoke/types/optional_match_smoke.tetra",
		"examples/smoke/errors/typed_errors_smoke.tetra",
		"examples/smoke/language/generic_smoke.tetra",
		"examples/smoke/language/generic_struct_smoke.tetra",
		"examples/smoke/language/protocol_impl_smoke.tetra",
		"examples/smoke/language/extension_smoke.tetra",
		"examples/memory/ownership/ownership_smoke.tetra",
		"examples/async/async_smoke.tetra",
		"examples/tasks/task_smoke.tetra",
		"examples/actors/actors_pingpong.tetra",
		"examples/memory/islands/islands_hello.tetra",
		"examples/memory/islands/islands_i32.tetra",
		"examples/memory/islands/islands_overflow.tetra",
		"examples/memory/raw/cap_mem_smoke.tetra",
		"examples/memory/raw/mmio_smoke.tetra",
		"examples/memory/raw/memset_smoke.tetra",
		"examples/ui/ui_web_smoke.tetra",
		"examples/ui/ui_native_shell_smoke.tetra",
		"examples/projects/dogfood_wasi/src/main.tetra",
		"examples/projects/dogfood_web_ui/src/main.tetra",
		"examples/projects/dogfood_cli/src/main.tetra",
		"examples/projects/dogfood_actor_task/src/main.tetra",
		"examples/projects/eco_dogfood/src/main.tetra",
	}
	headings := []string{
		"## Epic 14 Verification Commands",
		"## Troubleshooting Notes (Epic 14)",
		"### Basic language examples (`V020-0701..0705`)",
		"### Control-flow examples (`V020-0706..0710`)",
		"### Const and assignment examples (`V020-0711..0715`)",
		"### Enum/match examples (`V020-0716..0720`)",
		"### Optional/error examples (`V020-0721..0725`)",
		"### Generic/protocol/extension examples (`V020-0726..0730`)",
		"### Safety/runtime examples (`V020-0731..0735`)",
		"### Memory/capability examples (`V020-0736..0740`)",
		"### UI/WASM examples (`V020-0741..0745`)",
		"### Project dogfood examples (`V020-0746..0750`)",
	}
	lines := []string{
		"# Examples Index",
		"",
		"| Example | Purpose | Target group | Expected behavior |",
		"| --- | --- | --- | --- |",
	}
	for _, example := range examples {
		lines = append(lines, "| `"+example+"` | test entry | native | exits 0 |")
	}
	for _, heading := range headings {
		lines = append(lines, "", heading, "", "unsupported profile note", "regression note")
	}
	if err := os.WriteFile(indexPath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	err := verifyEpic14ExampleIndex(indexPath)
	if err == nil {
		t.Fatalf("expected missing primary .t4 project coverage failure")
	}
	if !strings.Contains(err.Error(), "examples/projects/hello_t4/src/main.t4") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyEpic14ExampleIndexRejectsMissingEntry(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "examples_index.md")
	body := strings.Join([]string{
		"# Examples Index",
		"",
		"| Example | Purpose | Target group | Expected behavior |",
		"| --- | --- | --- | --- |",
		"| `examples/flow/flow_hello.tetra` | test entry | native | exits 0 |",
		"## Epic 14 Verification Commands",
		"## Troubleshooting Notes (Epic 14)",
		"### Basic language examples (`V020-0701..0705`)",
		"unsupported regression note",
		"### Control-flow examples (`V020-0706..0710`)",
		"unsupported regression note",
		"### Const and assignment examples (`V020-0711..0715`)",
		"unsupported regression note",
		"### Enum/match examples (`V020-0716..0720`)",
		"unsupported regression note",
		"### Optional/error examples (`V020-0721..0725`)",
		"unsupported regression note",
		"### Generic/protocol/extension examples (`V020-0726..0730`)",
		"unsupported regression note",
		"### Safety/runtime examples (`V020-0731..0735`)",
		"unsupported regression note",
		"### Memory/capability examples (`V020-0736..0740`)",
		"unsupported regression note",
		"### UI/WASM examples (`V020-0741..0745`)",
		"unsupported regression note",
		"### Project dogfood examples (`V020-0746..0750`)",
		"unsupported regression note",
	}, "\n")
	if err := os.WriteFile(indexPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	err := verifyEpic14ExampleIndex(indexPath)
	if err == nil {
		t.Fatalf("expected Epic 14 missing coverage failure")
	}
	if !strings.Contains(err.Error(), "examples/smoke/basic/hello.tetra") {
		t.Fatalf("unexpected error: %v", err)
	}
}
