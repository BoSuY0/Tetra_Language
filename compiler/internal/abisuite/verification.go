package abisuite

import (
	"fmt"
	"strings"
)

const (
	VerificationSchemaV1  = "tetra.abi.verification.v1"
	VerificationScopeP211 = "p21.1_abi_verification"
)

const (
	VerificationTaskCorpus           = "abi_test_corpus"
	VerificationTaskAggregateReturns = "struct_enum_slice_string_return_validation"
	VerificationTaskCallBoundary     = "call_boundary_validation"
	VerificationTaskFFIReprC         = "ffi_repr_c_tests"
)

type VerificationReport struct {
	Schema    string
	Scope     string
	Claims    []string
	NonClaims []string
	Targets   []VerificationTargetRow
	Tasks     []VerificationTaskRow
}

type VerificationTargetRow struct {
	Target       string
	ABI          string
	Status       string
	TaskCoverage []string
	Evidence     []string
	Claims       []string
}

type VerificationTaskRow struct {
	ID       string
	Name     string
	Targets  []string
	Evidence []string
}

func BuildP21VerificationReport() VerificationReport {
	targets := P21VerificationTargets()
	tasks := P21VerificationTaskIDs()
	return VerificationReport{
		Schema: VerificationSchemaV1,
		Scope:  VerificationScopeP211,
		Claims: []string{
			"ABI verification v1 covers declared target metadata, classifier/layout rows, backend call-boundary metadata, and repr(C) aggregate export gates",
			"wasm32 targets use compiler-owned i32 slot ABI metadata validation",
			"native exported aggregate FFI boundaries require explicit repr(C)",
		},
		NonClaims: []string{
			"no runtime execution claim for build-only or wasm targets",
			"no C ABI claim for default structs",
			"no native C aggregate ABI claim for wasm targets",
			"no performance claim",
			"no safe-program semantics change",
		},
		Targets: []VerificationTargetRow{
			{
				Target:       "linux-x64",
				ABI:          "SysV x86_64",
				Status:       "supported_native",
				TaskCoverage: append([]string{}, tasks...),
				Evidence: []string{
					"compiler/abi_suite.go: x64 target model, SysV classifier, varargs and aggregates, c_int/c_uint FFI object smokes",
					"compiler/ffi_target_diagnostics_test.go: native exported aggregate repr(C) diagnostics",
				},
				Claims: []string{"linux-x64 SysV classifier/layout/call-boundary evidence is covered by the ABI suite"},
			},
			{
				Target:       "linux-x86",
				ABI:          "i386 SysV",
				Status:       "build_only",
				TaskCoverage: append([]string{}, tasks...),
				Evidence: []string{
					"compiler/abi_suite.go: x86 target model, i386 SysV classifier, varargs/sret, pointer/c_int/c_uint/ILP32 FFI object smokes",
					"compiler/ffi_target_diagnostics_test.go: x86 pointer and repr(C) aggregate diagnostics",
				},
				Claims: []string{"linux-x86 i386 SysV classifier/layout/call-boundary evidence is covered by compile and object checks"},
			},
			{
				Target:       "linux-x32",
				ABI:          "x32 SysV",
				Status:       "build_only",
				TaskCoverage: append([]string{}, tasks...),
				Evidence: []string{
					"compiler/abi_suite.go: x32 target model, x32 SysV classifier, varargs/aggregates, pointer/c_int/c_uint/ILP32 FFI object smokes",
					"compiler/ffi_target_diagnostics_test.go: x32 pointer and repr(C) aggregate diagnostics",
				},
				Claims: []string{"linux-x32 SysV classifier/layout/call-boundary evidence is covered by compile and object checks"},
			},
			{
				Target:       "macos-x64",
				ABI:          "SysV x86_64 Mach-O",
				Status:       "supported_native",
				TaskCoverage: append([]string{}, tasks...),
				Evidence: []string{
					"compiler/abi_suite.go: macos-x64 target model, SysV classifier, varargs/aggregates, object ABI smoke",
					"compiler/ffi_target_diagnostics_test.go: native exported aggregate repr(C) diagnostics",
				},
				Claims: []string{"macos-x64 SysV classifier/layout/call-boundary evidence is covered by object ABI smoke and classifier rows"},
			},
			{
				Target:       "windows-x64",
				ABI:          "Win64",
				Status:       "supported_native",
				TaskCoverage: append([]string{}, tasks...),
				Evidence: []string{
					"compiler/abi_suite.go: windows-x64 target model, Win64 classifier, varargs/aggregates, object ABI smoke",
					"compiler/ffi_target_diagnostics_test.go: native exported aggregate repr(C) diagnostics",
				},
				Claims: []string{"windows-x64 Win64 classifier/layout/call-boundary evidence is covered by object ABI smoke and classifier rows"},
			},
			{
				Target:       "wasm32-wasi",
				ABI:          "WASI i32 slot ABI",
				Status:       "supported_wasm_artifact",
				TaskCoverage: append([]string{}, tasks...),
				Evidence: []string{
					"compiler/abi_wasm.go: wasm32-wasi target model, slot ABI metadata, aggregate return layout, and call boundary validation",
					"compiler/internal/backend/wasm32_wasi/codegen.go: IRCall arg/return slot metadata validation",
				},
				Claims: []string{"wasm32-wasi uses compiler-owned i32 slot ABI metadata for aggregate returns and call boundaries"},
			},
			{
				Target:       "wasm32-web",
				ABI:          "Web i32 slot ABI",
				Status:       "supported_wasm_artifact",
				TaskCoverage: append([]string{}, tasks...),
				Evidence: []string{
					"compiler/abi_wasm.go: wasm32-web target model, slot ABI metadata, aggregate return layout, and call boundary validation",
					"compiler/internal/backend/wasm32_web/codegen.go: IRCall arg/return slot metadata validation including Surface imports",
				},
				Claims: []string{"wasm32-web uses compiler-owned i32 slot ABI metadata for aggregate returns and call boundaries"},
			},
		},
		Tasks: []VerificationTaskRow{
			{
				ID:       VerificationTaskCorpus,
				Name:     "ABI test corpus",
				Targets:  append([]string{}, targets...),
				Evidence: []string{"compiler/abi_suite.go", "compiler/abi_suite_test.go"},
			},
			{
				ID:       VerificationTaskAggregateReturns,
				Name:     "Struct/enum/slice/String return validation",
				Targets:  append([]string{}, targets...),
				Evidence: []string{"compiler/abi_suite.go native aggregate classifier checks", "compiler/abi_wasm.go wasm aggregate return layout checks"},
			},
			{
				ID:       VerificationTaskCallBoundary,
				Name:     "Call boundary validation",
				Targets:  append([]string{}, targets...),
				Evidence: []string{"compiler/internal/backend/wasm32_wasi/codegen.go", "compiler/internal/backend/wasm32_web/codegen.go", "compiler/internal/backend/x64abi/classifier.go", "compiler/internal/backend/x86abi/classifier.go"},
			},
			{
				ID:       VerificationTaskFFIReprC,
				Name:     "FFI repr(C) tests",
				Targets:  append([]string{}, targets...),
				Evidence: []string{"compiler/ffi_target.go", "compiler/ffi_target_diagnostics_test.go", "compiler/internal/semantics/layout_repr_test.go"},
			},
		},
	}
}

func ValidateP21VerificationReport(report VerificationReport) error {
	if report.Schema != VerificationSchemaV1 {
		return fmt.Errorf("ABI verification report schema = %q, want %q", report.Schema, VerificationSchemaV1)
	}
	if report.Scope != VerificationScopeP211 {
		return fmt.Errorf("ABI verification report scope = %q, want %q", report.Scope, VerificationScopeP211)
	}
	if err := validateStrings("claim", report.Claims, true); err != nil {
		return err
	}
	targetRows := map[string]VerificationTargetRow{}
	for _, row := range report.Targets {
		if strings.TrimSpace(row.Target) == "" || strings.TrimSpace(row.ABI) == "" || strings.TrimSpace(row.Status) == "" {
			return fmt.Errorf("ABI verification target row missing required metadata: %#v", row)
		}
		if _, exists := targetRows[row.Target]; exists {
			return fmt.Errorf("duplicate ABI verification target %s", row.Target)
		}
		if err := validateStrings("target "+row.Target+" evidence", row.Evidence, false); err != nil {
			return err
		}
		if err := validateStrings("target "+row.Target+" claim", row.Claims, true); err != nil {
			return err
		}
		for _, task := range P21VerificationTaskIDs() {
			if !stringSliceHas(row.TaskCoverage, task) {
				return fmt.Errorf("target %s missing task %s coverage", row.Target, task)
			}
		}
		targetRows[row.Target] = row
	}
	for _, target := range P21VerificationTargets() {
		if _, ok := targetRows[target]; !ok {
			return fmt.Errorf("missing target %s in ABI verification report", target)
		}
	}
	taskRows := map[string]VerificationTaskRow{}
	for _, row := range report.Tasks {
		if strings.TrimSpace(row.ID) == "" || strings.TrimSpace(row.Name) == "" {
			return fmt.Errorf("ABI verification task row missing required metadata: %#v", row)
		}
		if _, exists := taskRows[row.ID]; exists {
			return fmt.Errorf("duplicate ABI verification task %s", row.ID)
		}
		if err := validateStrings("task "+row.ID+" evidence", row.Evidence, false); err != nil {
			return err
		}
		for _, target := range P21VerificationTargets() {
			if !stringSliceHas(row.Targets, target) {
				return fmt.Errorf("task %s missing target %s coverage", row.ID, target)
			}
		}
		taskRows[row.ID] = row
	}
	for _, task := range P21VerificationTaskIDs() {
		if _, ok := taskRows[task]; !ok {
			return fmt.Errorf("missing task %s in ABI verification report", task)
		}
	}
	for _, nonClaim := range P21VerificationNonClaims() {
		if !stringSliceHas(report.NonClaims, nonClaim) {
			return fmt.Errorf("ABI verification report missing non-claim %q", nonClaim)
		}
	}
	if err := validateStrings("non-claim", report.NonClaims, false); err != nil {
		return err
	}
	return nil
}

func P21VerificationTargets() []string {
	return []string{"linux-x64", "linux-x86", "linux-x32", "macos-x64", "windows-x64", "wasm32-wasi", "wasm32-web"}
}

func P21VerificationTaskIDs() []string {
	return []string{
		VerificationTaskCorpus,
		VerificationTaskAggregateReturns,
		VerificationTaskCallBoundary,
		VerificationTaskFFIReprC,
	}
}

func P21VerificationNonClaims() []string {
	return []string{
		"no runtime execution claim for build-only or wasm targets",
		"no C ABI claim for default structs",
		"no native C aggregate ABI claim for wasm targets",
		"no performance claim",
		"no safe-program semantics change",
	}
}

func validateStrings(label string, values []string, rejectBroadClaims bool) error {
	if len(values) == 0 {
		return fmt.Errorf("%s list is empty", label)
	}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		lower := strings.ToLower(trimmed)
		if trimmed == "" {
			return fmt.Errorf("%s contains an empty entry", label)
		}
		for _, forbidden := range []string{"placeholder", "todo", "mock"} {
			if strings.Contains(lower, forbidden) {
				return fmt.Errorf("%s contains %s evidence/claim: %q", label, forbidden, value)
			}
		}
		if !rejectBroadClaims {
			continue
		}
		if strings.Contains(lower, "full runtime") || strings.Contains(lower, "runtime execution verified") {
			return fmt.Errorf("%s contains unsupported runtime execution claim: %q", label, value)
		}
		if strings.Contains(lower, "performance") {
			return fmt.Errorf("%s contains unsupported performance claim: %q", label, value)
		}
		if strings.Contains(lower, "default structs") && strings.Contains(lower, "c abi") {
			return fmt.Errorf("%s contains unsupported default structs C ABI claim: %q", label, value)
		}
		if strings.Contains(lower, "wasm") && strings.Contains(lower, "native c aggregate abi") {
			return fmt.Errorf("%s contains unsupported wasm native C aggregate ABI claim: %q", label, value)
		}
	}
	return nil
}

func stringSliceHas(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
