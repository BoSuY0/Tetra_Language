package compiler_test

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestFunctionTypedCallableTestsAreSplitByDomain(t *testing.T) {
	expected := map[string][]string{
		".": {
			"function_typed_callable_structure_test.go",
		},
		"core": {
			"function_typed_callable_direct_symbol_test.go",
			"function_typed_callable_local_return_alias_test.go",
			"function_typed_callable_parameter_field_return_test.go",
			"function_typed_callable_parameter_return_capture_test.go",
			"function_typed_callable_parameter_return_reassignment_test.go",
			"function_typed_callable_test.go",
		},
		"captures": {
			"function_typed_callable_captured_closure_test.go",
			"function_typed_callable_full_capture_test.go",
			"function_typed_callable_returned_struct_enum_payload_test.go",
		},
		"globals": {
			"function_typed_callable_enum_payload_test.go",
			"function_typed_callable_global_value_test.go",
			"function_typed_callable_imported_enum_payload_test.go",
			"function_typed_callable_imported_global_test.go",
			"function_typed_callable_mutable_global_storage_test.go",
			"function_typed_callable_mutable_global_test.go",
		},
		"reassignment": {
			"function_typed_callable_parameter_return_enum_reassignment_test.go",
			"function_typed_callable_reassignment_generic_test.go",
		},
		"cross_module": {
			"function_typed_callable_cross_module_callback_test.go",
			"function_typed_callable_cross_module_direct_storage_test.go",
			"function_typed_callable_cross_module_multi_target_test.go",
			"function_typed_callable_cross_module_return_test.go",
		},
		"throwing": {
			"function_typed_callable_throwing_test.go",
		},
		"unsupported": {
			"function_typed_callable_unsupported_diagnostics_test.go",
		},
	}

	if _, err := os.Stat("README.md"); err != nil {
		t.Fatalf("README.md must remain in callables root: %v", err)
	}
	for dir, want := range expected {
		requireGoFiles(t, dir, want)
		for _, name := range want {
			requireCompilerTestPackage(t, filepath.Join(dir, name))
		}
	}
	if _, err := os.Stat("runtime_helpers_test.go"); !os.IsNotExist(err) {
		t.Fatalf("runtime_helpers_test.go must move out of callables root; stat err=%v", err)
	}
}

func requireGoFiles(t *testing.T, dir string, want []string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	var got []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		got = append(got, entry.Name())
	}
	sort.Strings(got)
	want = append([]string(nil), want...)
	sort.Strings(want)

	if len(got) > 6 {
		t.Fatalf("%s has %d Go files, want <= 6: %v", dir, len(got), got)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s Go files mismatch:\n got: %v\nwant: %v", dir, got, want)
	}
}

func requireCompilerTestPackage(t *testing.T, path string) {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.HasPrefix(string(raw), "package compiler_test\n") {
		t.Fatalf("%s must use package compiler_test", path)
	}
}
