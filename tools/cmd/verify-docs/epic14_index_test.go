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
		"examples/hello.tetra",
		"examples/flow_hello.tetra",
		"examples/bool_smoke.tetra",
		"examples/for_range_smoke.tetra",
		"examples/for_collection_smoke.tetra",
		"examples/loop_control_smoke.tetra",
		"examples/const_smoke.tetra",
		"examples/const_bool_smoke.tetra",
		"examples/local_const_smoke.tetra",
		"examples/compound_assignment_smoke.tetra",
		"examples/enum_match_smoke.tetra",
		"examples/enum_exhaustive_match_smoke.tetra",
		"examples/optional_smoke.tetra",
		"examples/optional_match_smoke.tetra",
		"examples/typed_errors_smoke.tetra",
		"examples/generic_smoke.tetra",
		"examples/protocol_impl_smoke.tetra",
		"examples/extension_smoke.tetra",
		"examples/ownership_smoke.tetra",
		"examples/async_smoke.tetra",
		"examples/task_smoke.tetra",
		"examples/actors_pingpong.tetra",
		"examples/islands_hello.tetra",
		"examples/islands_i32.tetra",
		"examples/islands_overflow.tetra",
		"examples/cap_mem_smoke.tetra",
		"examples/mmio_smoke.tetra",
		"examples/memset_smoke.tetra",
		"examples/ui_web_smoke.tetra",
		"examples/ui_native_shell_smoke.tetra",
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
	if !strings.Contains(err.Error(), "examples/generic_struct_smoke.tetra") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyEpic14ExampleIndexRejectsMissingPrimaryT4ProjectEntry(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "examples_index.md")
	examples := []string{
		"examples/hello.tetra",
		"examples/flow_hello.tetra",
		"examples/bool_smoke.tetra",
		"examples/for_range_smoke.tetra",
		"examples/for_collection_smoke.tetra",
		"examples/loop_control_smoke.tetra",
		"examples/const_smoke.tetra",
		"examples/const_bool_smoke.tetra",
		"examples/local_const_smoke.tetra",
		"examples/compound_assignment_smoke.tetra",
		"examples/enum_match_smoke.tetra",
		"examples/enum_exhaustive_match_smoke.tetra",
		"examples/optional_smoke.tetra",
		"examples/optional_match_smoke.tetra",
		"examples/typed_errors_smoke.tetra",
		"examples/generic_smoke.tetra",
		"examples/generic_struct_smoke.tetra",
		"examples/protocol_impl_smoke.tetra",
		"examples/extension_smoke.tetra",
		"examples/ownership_smoke.tetra",
		"examples/async_smoke.tetra",
		"examples/task_smoke.tetra",
		"examples/actors_pingpong.tetra",
		"examples/islands_hello.tetra",
		"examples/islands_i32.tetra",
		"examples/islands_overflow.tetra",
		"examples/cap_mem_smoke.tetra",
		"examples/mmio_smoke.tetra",
		"examples/memset_smoke.tetra",
		"examples/ui_web_smoke.tetra",
		"examples/ui_native_shell_smoke.tetra",
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
		"| `examples/flow_hello.tetra` | test entry | native | exits 0 |",
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
	if !strings.Contains(err.Error(), "examples/hello.tetra") {
		t.Fatalf("unexpected error: %v", err)
	}
}
