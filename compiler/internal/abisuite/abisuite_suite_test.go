package abisuite

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
	ctarget "tetra_language/compiler/target"
)

// ---- checks_test.go ----

func TestRunChecksRecordsErrorsAndContinues(t *testing.T) {
	var order []string
	checks := RunChecks([]Case{
		{Name: "first", Run: func() error {
			order = append(order, "first")
			return nil
		}},
		{Name: "second", Run: func() error {
			order = append(order, "second")
			return errors.New("boom")
		}},
	})

	if len(checks) != 2 {
		t.Fatalf("checks len = %d, want 2", len(checks))
	}
	if checks[0] != (Check{Name: "first"}) {
		t.Fatalf("first check = %#v, want success", checks[0])
	}
	if checks[1].Name != "second" || checks[1].Error != "boom" {
		t.Fatalf("second check = %#v, want recorded error", checks[1])
	}
	if strings.Join(order, ",") != "first,second" {
		t.Fatalf("run order = %q, want first,second", strings.Join(order, ","))
	}
}

func TestUnsupportedTargetError(t *testing.T) {
	err := UnsupportedTargetError("plan9-x64")
	if err == nil ||
		!strings.Contains(err.Error(), "ABI suite for target plan9-x64 is not implemented") {
		t.Fatalf("UnsupportedTargetError = %v", err)
	}
}

// ---- ctx_switch_test.go ----

func TestCtxSwitchObjectSmokesUseBackendCallbacks(t *testing.T) {
	var calls []string
	deps := CtxSwitchDeps{
		BuildX86Object: func(funcs []ir.IRFunc) (CtxSwitchObject, error) {
			requireCtxSwitchSmokeIR(t, funcs, "__tetra_x86_ctx_switch_smoke")
			calls = append(calls, "x86:"+funcs[0].Name)
			code := []byte{0x90}
			code = append(code, ctxSwitchI386Stub()...)
			code = append(code, []byte{0x31, 0xC0, 0x50}...)
			return CtxSwitchObject{Target: "linux-x86", Code: code}, nil
		},
		BuildX32Object: func(funcs []ir.IRFunc) (CtxSwitchObject, error) {
			requireCtxSwitchSmokeIR(t, funcs, "__tetra_x32_ctx_switch_smoke")
			calls = append(calls, "x32:"+funcs[0].Name)
			code := []byte{0x90}
			code = append(code, ctxSwitchX32SysVStub()...)
			code = append(code, []byte{0x31, 0xC0, 0x50}...)
			return CtxSwitchObject{Target: "linux-x32", Code: code}, nil
		},
	}

	if err := CheckX86CtxSwitchObjectSmoke(deps); err != nil {
		t.Fatalf("CheckX86CtxSwitchObjectSmoke: %v", err)
	}
	if err := CheckX32CtxSwitchObjectSmoke(deps); err != nil {
		t.Fatalf("CheckX32CtxSwitchObjectSmoke: %v", err)
	}
	wantCalls := []string{"x86:__tetra_x86_ctx_switch_smoke", "x32:__tetra_x32_ctx_switch_smoke"}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("backend calls = %#v, want %#v", calls, wantCalls)
	}
}

func requireCtxSwitchSmokeIR(t *testing.T, funcs []ir.IRFunc, name string) {
	t.Helper()
	if len(funcs) != 1 {
		t.Fatalf("func count = %d, want 1", len(funcs))
	}
	fn := funcs[0]
	if fn.Name != name {
		t.Fatalf("func name = %q, want %q", fn.Name, name)
	}
	if fn.ReturnSlots != 1 {
		t.Fatalf("return slots = %d, want 1", fn.ReturnSlots)
	}
	gotKinds := make([]ir.IRInstrKind, 0, len(fn.Instrs))
	for _, instr := range fn.Instrs {
		gotKinds = append(gotKinds, instr.Kind)
	}
	wantKinds := []ir.IRInstrKind{
		ir.IRConstI32,
		ir.IRConstI32,
		ir.IRConstI32,
		ir.IRCtxSwitch,
		ir.IRReturn,
	}
	if !reflect.DeepEqual(gotKinds, wantKinds) {
		t.Fatalf("instruction kinds = %#v, want %#v", gotKinds, wantKinds)
	}
}

// ---- ffi_test.go ----

func TestFFIChecksUseBuildAndObjectCallbacks(t *testing.T) {
	var lastTarget string
	deps := FFICheckDeps{
		BuildLibrary: func(srcPath string, outPath string, target string) error {
			lastTarget = target
			switch {
			case strings.Contains(srcPath, "ffi_ref_null_return"):
				return fmt.Errorf("type mismatch: expected 'ref', got 'i32'")
			case strings.Contains(srcPath, "fnptr_param"):
				boundary := "i386"
				if target == "linux-x32" {
					boundary = "x32"
				}
				return fmt.Errorf(
					("exported function 'ffi_fnptr_param' parameter 'cb' type 'fnptr' " +
						"requires the %s pointer C ABI boundary; %s pointer C ABI boundary is " +
						"not verified on %s"),
					boundary,
					boundary,
					target,
				)
			case strings.Contains(srcPath, "fnptr_return"):
				boundary := "i386"
				if target == "linux-x32" {
					boundary = "x32"
				}
				return fmt.Errorf(
					("exported function 'ffi_fnptr_return' return type 'fnptr' " +
						"requires the %s pointer C ABI boundary; %s pointer C ABI boundary is " +
						"not verified on %s"),
					boundary,
					boundary,
					target,
				)
			default:
				return nil
			}
		},
		ReadObject: func(path string) (ObjectSummary, error) {
			return ObjectSummary{
				Target: lastTarget,
				Symbols: []ObjectSymbolSummary{
					{Name: "ffi_ptr_identity_c", HasSignature: true, ParamSlots: 1, ReturnSlots: 1},
					{
						Name:         "ffi_rawptr_identity_c",
						HasSignature: true,
						ParamSlots:   1,
						ReturnSlots:  1,
					},
					{
						Name:         "ffi_nullable_ptr_identity_c",
						HasSignature: true,
						ParamSlots:   1,
						ReturnSlots:  1,
					},
					{
						Name:         "ffi_nullable_ptr_null_c",
						HasSignature: true,
						ParamSlots:   0,
						ReturnSlots:  1,
					},
					{Name: "ffi_ref_identity_c", HasSignature: true, ParamSlots: 1, ReturnSlots: 1},
					{
						Name:         "ffi_c_int_identity_c",
						HasSignature: true,
						ParamSlots:   1,
						ReturnSlots:  1,
					},
					{
						Name:         "ffi_c_uint_identity_c",
						HasSignature: true,
						ParamSlots:   1,
						ReturnSlots:  1,
					},
					{
						Name:         "ffi_usize_identity_c",
						HasSignature: true,
						ParamSlots:   1,
						ReturnSlots:  1,
					},
					{
						Name:         "ffi_isize_identity_c",
						HasSignature: true,
						ParamSlots:   1,
						ReturnSlots:  1,
					},
					{
						Name:         "ffi_size_t_identity_c",
						HasSignature: true,
						ParamSlots:   1,
						ReturnSlots:  1,
					},
					{
						Name:         "ffi_ssize_t_identity_c",
						HasSignature: true,
						ParamSlots:   1,
						ReturnSlots:  1,
					},
					{
						Name:         "ffi_native_int_identity_c",
						HasSignature: true,
						ParamSlots:   1,
						ReturnSlots:  1,
					},
					{
						Name:         "ffi_native_uint_identity_c",
						HasSignature: true,
						ParamSlots:   1,
						ReturnSlots:  1,
					},
					{
						Name:         "ffi_c_long_identity_c",
						HasSignature: true,
						ParamSlots:   1,
						ReturnSlots:  1,
					},
					{
						Name:         "ffi_c_ulong_identity_c",
						HasSignature: true,
						ParamSlots:   1,
						ReturnSlots:  1,
					},
				},
			}, nil
		},
	}

	x86, err := ctarget.Parse("linux-x86")
	if err != nil {
		t.Fatalf("parse x86 target: %v", err)
	}
	x64, err := ctarget.Parse("linux-x64")
	if err != nil {
		t.Fatalf("parse x64 target: %v", err)
	}
	x32, err := ctarget.Parse("linux-x32")
	if err != nil {
		t.Fatalf("parse x32 target: %v", err)
	}

	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{name: "x86 pointer object", run: func() error { return CheckPointerFFIObjectSmoke(x86, deps) }},
		{name: "x32 pointer object", run: func() error { return CheckPointerFFIObjectSmoke(x32, deps) }},
		{name: "x64 c_int object", run: func() error { return CheckCIntFFIObjectSmoke(x64, deps) }},
		{name: "x86 c_uint object", run: func() error { return CheckCUIntFFIObjectSmoke(x86, deps) }},
		{name: "x32 native libc object", run: func() error {
			return CheckILP32NativeLibcFFIObjectSmoke(
				x32,
				deps,
			)
		}},
		{name: "x86 ref null diagnostic", run: func() error {
			return CheckRefFFINullReturnDiagnostics(
				"linux-x86",
				"x86",
				deps,
			)
		}},
		{name: "x32 fnptr diagnostic", run: func() error {
			return CheckFunctionPointerFFIDiagnostics(
				"linux-x32",
				"x32",
				"x32",
				deps,
			)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err != nil {
				t.Fatalf("FFI check: %v", err)
			}
		})
	}
}

// ---- native_classifiers_test.go ----

func TestNativeClassifierChecks(t *testing.T) {
	tests := []struct {
		name   string
		target string
		check  func(ctarget.Target) error
	}{
		{name: "x86 classifier", target: "linux-x86", check: CheckX86I386Classifier},
		{name: "x86 varargs and sret", target: "linux-x86", check: CheckX86VarargsAndSRet},
		{name: "x64 sysv classifier", target: "linux-x64", check: CheckX64Classifier},
		{
			name:   "x64 sysv varargs and aggregates",
			target: "linux-x64",
			check:  CheckX64VarargsAndAggregates,
		},
		{name: "x64 win64 classifier", target: "windows-x64", check: CheckX64Classifier},
		{
			name:   "x64 win64 varargs and aggregates",
			target: "windows-x64",
			check:  CheckX64VarargsAndAggregates,
		},
		{name: "x32 classifier", target: "linux-x32", check: CheckX32SysVClassifier},
		{
			name:   "x32 varargs and aggregates",
			target: "linux-x32",
			check:  CheckX32SysVVarargsAndAggregates,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tgt, err := ctarget.Parse(tt.target)
			if err != nil {
				t.Fatalf("parse target: %v", err)
			}
			if err := tt.check(tgt); err != nil {
				t.Fatalf("classifier check: %v", err)
			}
		})
	}
}

// ---- runtime_boundaries_test.go ----

type runtimeBoundaryTestError struct {
	message string
}

func (e runtimeBoundaryTestError) Error() string {
	return e.message
}

func TestRuntimeBoundaryDiagnosticsUseCallbacks(t *testing.T) {
	var built []string
	deps := RuntimeBoundaryDeps{
		TargetRuntimeDiagnosticCode: "TETRA3003",
		BuildExecutable: func(srcPath string, outPath string, target string) error {
			built = append(built, outPath)
			base := outPath[strings.LastIndex(outPath, string(os.PathSeparator))+1:]
			switch {
			case strings.HasPrefix(base, "filesystem-"):
				return runtimeBoundaryTestError{
					message: "filesystem runtime not supported on " + target,
				}
			case strings.HasPrefix(
				base,
				"networking-",
			) || strings.Contains(
				base,
				"socket_tcp4",
			) || strings.Contains(
				base,
				"epoll_create",
			):
				return runtimeBoundaryTestError{
					message: "networking runtime not supported on " + target,
				}
			case strings.Contains(base, "actor_fanout"):
				return runtimeBoundaryTestError{
					message: "actor fanout above 2 runtime not supported on " + target,
				}
			case strings.Contains(base, "surface"):
				return runtimeBoundaryTestError{
					message: "surface runtime not supported on " + target,
				}
			case strings.Contains(base, "distributed"):
				return runtimeBoundaryTestError{
					message: "distributed actors runtime not supported on " + target,
				}
			case target == "linux-x86":
				return runtimeBoundaryTestError{
					message: "networking runtime not supported on " + target,
				}
			default:
				return runtimeBoundaryTestError{
					message: fmt.Sprintf("unexpected build %s for %s", base, target),
				}
			}
		},
		DiagnosticFromError: func(err error) DiagnosticSummary {
			return DiagnosticSummary{
				Code:     "TETRA3003",
				Message:  err.Error(),
				Severity: "error",
				Hint:     "Build this source for linux-x64",
			}
		},
		TargetSupportsNetRuntimeSymbols: func(target string, symbols []string) bool {
			return false
		},
		RequiredNetRuntimeSymbols: func() []string {
			return []string{"__tetra_net_socket_tcp4"}
		},
		NetRuntimeSymbolForBuiltin: func(name string) (string, bool) {
			switch name {
			case "core.net_socket_tcp4":
				return "__tetra_net_socket_tcp4", true
			case "core.net_epoll_create":
				return "__tetra_net_epoll_create", true
			default:
				return "", false
			}
		},
	}

	wasm, err := ctarget.Parse("wasm32-wasi")
	if err != nil {
		t.Fatalf("parse wasm target: %v", err)
	}
	x86, err := ctarget.Parse("linux-x86")
	if err != nil {
		t.Fatalf("parse x86 target: %v", err)
	}

	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{name: "stdlib", run: func() error { return CheckStdlibRuntimeBoundaryDiagnostics(wasm, deps) }},
		{name: "target", run: func() error { return CheckTargetRuntimeBoundaryDiagnostics(x86, deps) }},
		{name: "surface", run: func() error {
			return CheckSurfaceDistributedRuntimeBoundaryDiagnostics(
				x86,
				deps,
			)
		}},
		{name: "networking", run: func() error {
			return CheckNetworkingRuntimeBoundaryDiagnostics(
				x86,
				deps,
			)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err != nil {
				t.Fatalf("runtime boundary check: %v", err)
			}
		})
	}
	if len(built) == 0 {
		t.Fatalf("runtime boundary checks did not call BuildExecutable")
	}
}

// ---- runtime_build_smoke_test.go ----

func TestRuntimeBuildSmokesUseCallbacks(t *testing.T) {
	var built []string
	deps := RuntimeSmokeDeps{
		BuildExecutable: func(srcPath string, outPath string, target string) error {
			if _, err := os.Stat(srcPath); err != nil {
				t.Fatalf("source file was not written before build callback: %v", err)
			}
			built = append(built, filepath.Base(outPath)+":"+target)
			return os.WriteFile(outPath, runtimeSmokeTestELF(target), 0o755)
		},
	}

	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{name: "x86 time", run: func() error { return CheckX86TimeRuntimeSmoke(deps) }},
		{name: "x86 filesystem", run: func() error { return CheckX86FilesystemRuntimeSmoke(deps) }},
		{name: "x86 filesystem scheduler", run: func() error {
			return CheckX86FilesystemSchedulerCompositionSmoke(
				deps,
			)
		}},
		{name: "x32 time", run: func() error { return CheckX32TimeRuntimeSmoke(deps) }},
		{name: "x32 filesystem", run: func() error { return CheckX32FilesystemRuntimeSmoke(deps) }},
		{name: "x32 filesystem scheduler", run: func() error {
			return CheckX32FilesystemSchedulerCompositionSmoke(
				deps,
			)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err != nil {
				t.Fatalf("runtime build smoke: %v", err)
			}
		})
	}

	wantBuilt := []string{
		"x86-time-runtime:linux-x86",
		"x86-filesystem-runtime:linux-x86",
		"x86-filesystem-scheduler:linux-x86",
		"x32-time-runtime:linux-x32",
		"x32-filesystem-runtime:linux-x32",
		"x32-filesystem-scheduler-runtime:linux-x32",
	}
	if !reflect.DeepEqual(built, wantBuilt) {
		t.Fatalf("built = %#v, want %#v", built, wantBuilt)
	}
}

// ---- runtime_selfhost_smoke_test.go ----

func TestRuntimeSelfHostSmokesUseCallbacks(t *testing.T) {
	var built []string
	deps := RuntimeSmokeDeps{
		BuildExecutable: func(srcPath string, outPath string, target string) error {
			if _, err := os.Stat(srcPath); err != nil {
				t.Fatalf("source file was not written before build callback: %v", err)
			}
			built = append(built, filepath.Base(outPath)+":"+target)
			return os.WriteFile(outPath, runtimeSmokeTestELF(target), 0o755)
		},
	}

	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{name: "x32 task", run: func() error { return CheckX32SingleTaskSelfHostRuntimeSmoke(deps) }},
		{name: "x32 typed task", run: func() error {
			return CheckX32TypedTaskSelfHostRuntimeSmoke(
				deps,
			)
		}},
		{name: "x32 staged typed task", run: func() error {
			return CheckX32StagedTypedTaskSelfHostRuntimeSmoke(
				deps,
			)
		}},
		{name: "x32 task group", run: func() error {
			return CheckX32TaskGroupSelfHostRuntimeSmoke(
				deps,
			)
		}},
		{name: "x32 typed task group", run: func() error {
			return CheckX32TypedTaskGroupSelfHostRuntimeSmoke(
				deps,
			)
		}},
		{name: "x32 actor", run: func() error { return CheckX32SingleActorSelfHostRuntimeSmoke(deps) }},
		{name: "x32 actor state", run: func() error {
			return CheckX32ActorStateSelfHostRuntimeSmoke(
				deps,
			)
		}},
		{name: "x86 task", run: func() error { return CheckX86SingleTaskSelfHostRuntimeSmoke(deps) }},
		{name: "x86 typed task", run: func() error {
			return CheckX86TypedTaskSelfHostRuntimeSmoke(
				deps,
			)
		}},
		{name: "x86 staged typed task", run: func() error {
			return CheckX86StagedTypedTaskSelfHostRuntimeSmoke(
				deps,
			)
		}},
		{name: "x86 task group", run: func() error {
			return CheckX86TaskGroupSelfHostRuntimeSmoke(
				deps,
			)
		}},
		{name: "x86 typed task group", run: func() error {
			return CheckX86TypedTaskGroupSelfHostRuntimeSmoke(
				deps,
			)
		}},
		{name: "x86 actor", run: func() error { return CheckX86SingleActorSelfHostRuntimeSmoke(deps) }},
		{name: "x86 actor state", run: func() error {
			return CheckX86ActorStateSelfHostRuntimeSmoke(
				deps,
			)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err != nil {
				t.Fatalf("self-host runtime smoke: %v", err)
			}
		})
	}

	wantBuilt := []string{
		"x32-task-runtime:linux-x32",
		"x32-typed-task-runtime:linux-x32",
		"x32-staged-typed-task-runtime:linux-x32",
		"x32-task-group-runtime:linux-x32",
		"x32-typed-task-group-runtime:linux-x32",
		"x32-actor-runtime:linux-x32",
		"x32-actor-state-runtime:linux-x32",
		"x86-task-runtime:linux-x86",
		"x86-typed-task-runtime:linux-x86",
		"x86-staged-typed-task-runtime:linux-x86",
		"x86-task-group-runtime:linux-x86",
		"x86-typed-task-group-runtime:linux-x86",
		"x86-actor-runtime:linux-x86",
		"x86-actor-state-runtime:linux-x86",
	}
	if !reflect.DeepEqual(built, wantBuilt) {
		t.Fatalf("built = %#v, want %#v", built, wantBuilt)
	}
}

// ---- runtime_smoke_test.go ----

func TestRuntimeStdoutStderrSmokesUseCallbacks(t *testing.T) {
	var built []string
	deps := RuntimeSmokeDeps{
		BuildExecutable: func(srcPath string, outPath string, target string) error {
			built = append(built, filepath.Base(outPath)+":"+target)
			data := runtimeSmokeTestELF(target)
			if strings.Contains(outPath, "stdout") {
				if target == "linux-x86" {
					data = append(data, []byte("x86 stdout\n")...)
					data = append(data, []byte{0xB8, 0x04, 0x00, 0x00, 0x00, 0xCD, 0x80}...)
				} else {
					data = append(data, []byte("x32 stdout\n")...)
					data = append(data, []byte{0xB8, 0x01, 0x00, 0x00, 0x40, 0x0F, 0x05}...)
				}
			} else {
				data = append(data, []byte{0xB8, 0x02, 0x00, 0x00, 0x00, 0x50}...)
				if target == "linux-x86" {
					data = append(data, []byte{0x8B, 0x5D, 0x08, 0x8B, 0x4D, 0x0C, 0x03, 0x4D, 0x14}...)
					data = append(data, []byte{0xB8, 0x04, 0x00, 0x00, 0x00, 0xCD, 0x80}...)
				} else {
					data = append(data, []byte{0x48, 0x63, 0xC9, 0x48, 0x01, 0xCE, 0x4C, 0x89, 0xC2}...)
					data = append(data, []byte{0xB8, 0x01, 0x00, 0x00, 0x40, 0x0F, 0x05}...)
				}
			}
			return os.WriteFile(outPath, data, 0o755)
		},
	}

	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{name: "x86 stdout", run: func() error { return CheckX86StdoutExecutableSmoke(deps) }},
		{name: "x32 stdout", run: func() error { return CheckX32StdoutExecutableSmoke(deps) }},
		{name: "x86 stderr", run: func() error { return CheckX86StderrFDRuntimeSmoke(deps) }},
		{name: "x32 stderr", run: func() error { return CheckX32StderrFDRuntimeSmoke(deps) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err != nil {
				t.Fatalf("runtime smoke: %v", err)
			}
		})
	}
	if len(built) != 4 {
		t.Fatalf("built %d executables, want 4: %#v", len(built), built)
	}
}

func TestRuntimeAllocatorSmokesUseCallbacks(t *testing.T) {
	var built []string
	deps := RuntimeSmokeDeps{
		BuildExecutable: func(srcPath string, outPath string, target string) error {
			built = append(built, filepath.Base(outPath)+":"+target)
			data := runtimeSmokeTestELF(target)
			data = appendAllocatorSmokeBytes(data, target, strings.Contains(outPath, "failure"))
			return os.WriteFile(outPath, data, 0o755)
		},
	}

	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{name: "x86 allocator", run: func() error { return CheckX86AllocatorExecutableSmoke(deps) }},
		{name: "x86 allocator failure", run: func() error {
			return CheckX86AllocatorFailureExecutableSmoke(
				deps,
			)
		}},
		{name: "x32 allocator", run: func() error { return CheckX32AllocatorExecutableSmoke(deps) }},
		{name: "x32 allocator failure", run: func() error {
			return CheckX32AllocatorFailureExecutableSmoke(
				deps,
			)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err != nil {
				t.Fatalf("allocator runtime smoke: %v", err)
			}
		})
	}
	if len(built) != 4 {
		t.Fatalf("built %d executables, want 4: %#v", len(built), built)
	}
}

func TestRuntimeRawMemoryPointerSmokesUseCallbacks(t *testing.T) {
	var built []string
	deps := RuntimeSmokeDeps{
		BuildExecutable: func(srcPath string, outPath string, target string) error {
			built = append(built, filepath.Base(outPath)+":"+target)
			data := runtimeSmokeTestELF(target)
			data = appendRawMemoryPointerSmokeBytes(data, target, outPath)
			return os.WriteFile(outPath, data, 0o755)
		},
	}

	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{name: "x86 raw memory", run: func() error {
			return CheckX86RawMemoryBoundsExecutableSmoke(
				deps,
			)
		}},
		{name: "x32 raw memory", run: func() error {
			return CheckX32RawMemoryBoundsExecutableSmoke(
				deps,
			)
		}},
		{name: "x86 raw pointer slot", run: func() error {
			return CheckX86RawPointerSlotExecutableSmoke(
				deps,
			)
		}},
		{name: "x32 raw pointer slot", run: func() error {
			return CheckX32RawPointerSlotExecutableSmoke(
				deps,
			)
		}},
		{name: "x86 raw pointer offset slot", run: func() error {
			return CheckX86RawPointerOffsetSlotExecutableSmoke(
				deps,
			)
		}},
		{name: "x32 raw pointer offset slot", run: func() error {
			return CheckX32RawPointerOffsetSlotExecutableSmoke(
				deps,
			)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err != nil {
				t.Fatalf("raw memory/pointer runtime smoke: %v", err)
			}
		})
	}
	if len(built) != 6 {
		t.Fatalf("built %d executables, want 6: %#v", len(built), built)
	}
}

func TestRuntimeNetworkingLifecycleSmokesUseCallbacks(t *testing.T) {
	var built []string
	deps := RuntimeSmokeDeps{
		BuildExecutable: func(srcPath string, outPath string, target string) error {
			built = append(built, filepath.Base(outPath)+":"+target)
			data := runtimeSmokeTestELF(target)
			data = appendNetworkingLifecycleSmokeBytes(data, target)
			return os.WriteFile(outPath, data, 0o755)
		},
	}

	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{name: "x86 networking lifecycle", run: func() error {
			return CheckX86NetworkingLifecycleRuntimeSmoke(
				deps,
			)
		}},
		{name: "x32 networking lifecycle", run: func() error {
			return CheckX32NetworkingLifecycleRuntimeSmoke(
				deps,
			)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err != nil {
				t.Fatalf("networking lifecycle runtime smoke: %v", err)
			}
		})
	}
	if len(built) != 2 {
		t.Fatalf("built %d executables, want 2: %#v", len(built), built)
	}
}

func TestRuntimeIslandFreeSmokesUseBuildOptionsCallback(t *testing.T) {
	var built []string
	deps := RuntimeSmokeDeps{
		BuildExecutableWithOptions: func(srcPath string, outPath string, target string, opts RuntimeBuildOptions) error {
			mode := "normal"
			if opts.IslandsDebug {
				mode = "debug"
			}
			built = append(built, filepath.Base(outPath)+":"+target+":"+mode)
			data := runtimeSmokeTestELF(target)
			data = appendIslandFreeSmokeBytes(data, target, opts.IslandsDebug)
			return os.WriteFile(outPath, data, 0o755)
		},
	}

	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{name: "x86 island free", run: func() error { return CheckX86IslandFreeExecutableSmoke(deps) }},
		{name: "x32 island free", run: func() error { return CheckX32IslandFreeExecutableSmoke(deps) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err != nil {
				t.Fatalf("island/free runtime smoke: %v", err)
			}
		})
	}
	if len(built) != 4 {
		t.Fatalf("built %d executables, want 4: %#v", len(built), built)
	}
}

func runtimeSmokeTestELF(target string) []byte {
	data := make([]byte, 20)
	copy(data[:4], "\x7fELF")
	data[4] = 1
	if target == "linux-x86" {
		data[18] = 0x03
	} else {
		data[18] = 0x3e
	}
	return data
}

func appendIslandFreeSmokeBytes(data []byte, target string, debug bool) []byte {
	if target == "linux-x86" {
		if debug {
			data = append(data, []byte{0xC7, 0x00, 0x00, 0x10, 0x00, 0x00}...)
			data = append(data, []byte{0x8B, 0x43, 0x0C, 0x85, 0xC0, 0x0F, 0x84}...)
			data = append(
				data,
				[]byte{0xBB, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x01, 0x00, 0x00, 0x00, 0xCD, 0x80}...)
			data = append(data, []byte{0xC7, 0x43, 0x0C, 0x01, 0x00, 0x00, 0x00}...)
			return append(data, []byte{0xB8, 0x7D, 0x00, 0x00, 0x00, 0xCD, 0x80}...)
		}
		data = append(data, []byte{0xB8, 0xC0, 0x00, 0x00, 0x00, 0xCD, 0x80}...)
		data = append(data, []byte{0xC7, 0x00, 0x10, 0x00, 0x00, 0x00}...)
		return append(data, []byte{0x8B, 0x4B, 0x08, 0xB8, 0x5B, 0x00, 0x00, 0x00, 0xCD, 0x80}...)
	}
	if debug {
		data = append(data, []byte{0xC7, 0x00, 0x00, 0x10, 0x00, 0x00}...)
		data = append(data, []byte{0x8B, 0x47, 0x0C, 0x85, 0xC0, 0x0F, 0x84}...)
		data = append(
			data,
			[]byte{0xBF, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x3C, 0x00, 0x00, 0x40, 0x0F, 0x05}...)
		data = append(data, []byte{0x48, 0x89, 0xF8, 0xC7, 0x40, 0x0C, 0x01, 0x00, 0x00, 0x00}...)
		data = append(
			data,
			[]byte{0x8B, 0x47, 0x08, 0x2D, 0x00, 0x10, 0x00, 0x00, 0x48, 0x89, 0xC6}...)
		return append(data, []byte{0xB8, 0x0A, 0x00, 0x00, 0x40, 0x0F, 0x05}...)
	}
	data = append(data, []byte{0xB8, 0x09, 0x00, 0x00, 0x40, 0x0F, 0x05}...)
	data = append(data, []byte{0xC7, 0x00, 0x10, 0x00, 0x00, 0x00}...)
	return append(data, []byte{0x8B, 0x77, 0x08, 0xB8, 0x0B, 0x00, 0x00, 0x40, 0x0F, 0x05}...)
}

func appendAllocatorSmokeBytes(data []byte, target string, failure bool) []byte {
	if target == "linux-x86" {
		if failure {
			data = append(data, []byte{0x83, 0xF9, 0x01, 0x0F, 0x8D}...)
			data = append(
				data,
				[]byte{0xBB, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x01, 0x00, 0x00, 0x00, 0xCD, 0x80}...)
			return append(data, []byte{0xB8, 0xC0, 0x00, 0x00, 0x00, 0xCD, 0x80}...)
		}
		data = append(data, []byte{0xB8, 0xC0, 0x00, 0x00, 0x00, 0xCD, 0x80}...)
		data = append(data, []byte{0x3D, 0x01, 0xF0, 0xFF, 0xFF, 0x0F, 0x83}...)
		data = append(
			data,
			[]byte{0xBB, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x01, 0x00, 0x00, 0x00, 0xCD, 0x80}...)
		return append(data, []byte{0x89, 0x08, 0x83, 0xC0, 0x08}...)
	}
	if failure {
		data = append(data, []byte{0x89, 0xF0, 0x3D, 0x01, 0x00, 0x00, 0x00, 0x0F, 0x8D}...)
		data = append(
			data,
			[]byte{0xBF, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x3C, 0x00, 0x00, 0x40, 0x0F, 0x05}...)
		return append(data, []byte{0xB8, 0x09, 0x00, 0x00, 0x40, 0x0F, 0x05}...)
	}
	data = append(data, []byte{0xB8, 0x09, 0x00, 0x00, 0x40, 0x0F, 0x05}...)
	data = append(data, []byte{0x48, 0x3D, 0x01, 0xF0, 0xFF, 0xFF, 0x0F, 0x83}...)
	data = append(
		data,
		[]byte{0xBF, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x3C, 0x00, 0x00, 0x40, 0x0F, 0x05}...)
	return append(data, []byte{0x89, 0x30, 0x48, 0x05, 0x08, 0x00, 0x00, 0x00}...)
}

func appendRawMemoryPointerSmokeBytes(data []byte, target string, outPath string) []byte {
	if target == "linux-x86" {
		data = append(data, []byte{0xB8, 0xC0, 0x00, 0x00, 0x00, 0xCD, 0x80}...)
		data = append(data, []byte{0x3D, 0x01, 0xF0, 0xFF, 0xFF, 0x0F, 0x83}...)
		data = append(
			data,
			[]byte{0xBB, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x01, 0x00, 0x00, 0x00, 0xCD, 0x80}...)
			if !strings.Contains(outPath, "raw_pointer_offset_slot") {
				data = append(
					data,
					[]byte{
						0xB8, 0x00, 0x00, 0x00, 0x00, 0x50, 0x8B, 0x85,
						0xF8, 0xFF, 0xFF, 0xFF, 0x50, 0x59, 0x5A, 0x58,
						0x83, 0xFA, 0x00, 0x0F, 0x8D,
					}...)
			}
		data = append(data, []byte{0x81, 0xE7, 0x00, 0xF0, 0xFF, 0xFF}...)
		data = append(data, []byte{0x8B, 0x0F, 0x83, 0xC7, 0x08}...)
		data = append(data, []byte{0x01, 0xC2, 0x83, 0xC2, 0x04, 0x39, 0xCA}...)
		if strings.Contains(outPath, "raw_memory_bounds") {
			data = append(data, []byte{0x01, 0xC2, 0x83, 0xC2, 0x01, 0x39, 0xCA}...)
			data = append(data, []byte{0x88, 0x18, 0x53}...)
			return append(data, []byte{0x0F, 0xB6, 0x00, 0x50}...)
		}
		data = append(data, []byte{0x89, 0x18, 0x53}...)
		return append(data, []byte{0x8B, 0x00, 0x50}...)
	}
	data = append(data, []byte{0xB8, 0x09, 0x00, 0x00, 0x40, 0x0F, 0x05}...)
	data = append(data, []byte{0x48, 0x3D, 0x01, 0xF0, 0xFF, 0xFF, 0x0F, 0x83}...)
	data = append(
		data,
		[]byte{0xBF, 0x02, 0x00, 0x00, 0x00, 0xB8, 0x3C, 0x00, 0x00, 0x40, 0x0F, 0x05}...)
		if !strings.Contains(outPath, "raw_pointer_offset_slot") {
			data = append(
				data,
				[]byte{
					0xB8, 0x00, 0x00, 0x00, 0x00, 0x50, 0x48, 0x8B,
					0x85, 0xF8, 0xFF, 0xFF, 0xFF, 0x50, 0x59, 0x5A,
					0x58, 0x81, 0xFA, 0x00, 0x00, 0x00, 0x00, 0x0F,
					0x8D,
				}...)
		}
	data = append(data, []byte{0x48, 0x89, 0xC7, 0x48, 0x81, 0xE7, 0x00, 0xF0, 0xFF, 0xFF}...)
	data = append(
		data,
		[]byte{0x8B, 0x8F, 0x00, 0x00, 0x00, 0x00, 0x48, 0x81, 0xC7, 0x08, 0x00, 0x00, 0x00}...)
	data = append(
		data,
		[]byte{
			0x48,
			0x29,
			0xF8,
			0x48,
			0x01,
			0xC2,
			0x81,
			0xC2,
			0x04,
			0x00,
			0x00,
			0x00,
			0x39,
			0xCA,
		}...)
	data = append(data, []byte{0x48, 0x63, 0xD2, 0x48, 0x89, 0xF8, 0x48, 0x01, 0xD0}...)
	if strings.Contains(outPath, "raw_memory_bounds") {
		data = append(
			data,
			[]byte{
				0x48,
				0x29,
				0xF8,
				0x48,
				0x01,
				0xC2,
				0x81,
				0xC2,
				0x01,
				0x00,
				0x00,
				0x00,
				0x39,
				0xCA,
			}...)
		data = append(data, []byte{0x44, 0x88, 0x00, 0x41, 0x50}...)
		return append(data, []byte{0x0F, 0xB6, 0x00, 0x50}...)
	}
	data = append(data, []byte{0x48, 0x89, 0xC7, 0x45, 0x89, 0xC0, 0x44, 0x89, 0x07, 0x41, 0x50}...)
	return append(data, []byte{0x8B, 0x00, 0x50}...)
}

func appendNetworkingLifecycleSmokeBytes(data []byte, target string) []byte {
	if target == "linux-x86" {
		for _, seq := range [][]byte{
			{0xB8, 0x66, 0x00, 0x00, 0x00},
			{0xBB, 0x01, 0x00, 0x00, 0x00},
			{0xBB, 0x02, 0x00, 0x00, 0x00},
			{0xBB, 0x03, 0x00, 0x00, 0x00},
			{0xBB, 0x04, 0x00, 0x00, 0x00},
			{0xBB, 0x09, 0x00, 0x00, 0x00},
			{0xBB, 0x0A, 0x00, 0x00, 0x00},
			{0xBB, 0x0E, 0x00, 0x00, 0x00},
			{0xBB, 0x12, 0x00, 0x00, 0x00},
			{0xB8, 0x03, 0x00, 0x00, 0x00},
			{0xB8, 0x04, 0x00, 0x00, 0x00},
			{0xB8, 0x49, 0x01, 0x00, 0x00},
			{0xB8, 0xFF, 0x00, 0x00, 0x00},
			{0xB8, 0x00, 0x01, 0x00, 0x00},
			{0xB8, 0x37, 0x00, 0x00, 0x00},
			{0x0D, 0x00, 0x08, 0x00, 0x00},
			{0xB8, 0x06, 0x00, 0x00, 0x00},
			{0xCD, 0x80},
		} {
			data = append(data, seq...)
		}
		return data
	}
	for _, seq := range [][]byte{
		{0xB8, 0x29, 0x00, 0x00, 0x40},
		{0xB8, 0x31, 0x00, 0x00, 0x40},
		{0xB8, 0x2A, 0x00, 0x00, 0x40},
		{0xB8, 0x32, 0x00, 0x00, 0x40},
		{0xB8, 0x20, 0x01, 0x00, 0x40},
		{0xB8, 0x00, 0x00, 0x00, 0x40},
		{0xB8, 0x01, 0x00, 0x00, 0x40},
		{0xB8, 0x2C, 0x00, 0x00, 0x40},
		{0xB8, 0x05, 0x02, 0x00, 0x40},
		{0xB8, 0x1D, 0x02, 0x00, 0x40},
		{0xB8, 0xE8, 0x00, 0x00, 0x40},
		{0xB8, 0xE9, 0x00, 0x00, 0x40},
		{0xB8, 0x23, 0x01, 0x00, 0x40},
		{0xB8, 0x48, 0x00, 0x00, 0x40},
		{0x0D, 0x00, 0x08, 0x00, 0x00},
		{0xB8, 0x03, 0x00, 0x00, 0x40},
		{0x0F, 0x05},
	} {
		data = append(data, seq...)
	}
	return data
}

// ---- target_models_test.go ----

func TestTargetModelChecks(t *testing.T) {
	tests := []struct {
		name   string
		target string
		check  func(ctarget.Target) error
	}{
		{name: "x86", target: "linux-x86", check: CheckX86TargetModel},
		{name: "x64", target: "linux-x64", check: CheckX64TargetModel},
		{name: "x32", target: "linux-x32", check: CheckX32TargetModel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tgt, err := ctarget.Parse(tt.target)
			if err != nil {
				t.Fatalf("parse target: %v", err)
			}
			if err := tt.check(tgt); err != nil {
				t.Fatalf("target model check: %v", err)
			}
		})
	}
}

func TestX86TargetModelRejectsWrongTarget(t *testing.T) {
	tgt, err := ctarget.Parse("linux-x64")
	if err != nil {
		t.Fatalf("parse target: %v", err)
	}
	err = CheckX86TargetModel(tgt)
	if err == nil || !strings.Contains(err.Error(), "want linux-x86/linux/x86/i386-sysv") {
		t.Fatalf("CheckX86TargetModel(linux-x64) = %v", err)
	}
}

// ---- target_suite_test.go ----

func TestRunTargetChecksRoutesByParsedTarget(t *testing.T) {
	var routed []string
	runners := TargetCheckRunners{
		X86:  targetRoutingRunner("x86", &routed),
		X32:  targetRoutingRunner("x32", &routed),
		X64:  targetRoutingRunner("x64", &routed),
		WASM: targetRoutingRunner("wasm", &routed),
	}

	for _, tc := range []struct {
		target    string
		wantRoute string
		wantCheck string
	}{
		{target: "x86", wantRoute: "x86:linux-x86", wantCheck: "x86 check"},
		{target: "x32", wantRoute: "x32:linux-x32", wantCheck: "x32 check"},
		{target: "linux-x64", wantRoute: "x64:linux-x64", wantCheck: "x64 check"},
		{target: "wasm32-wasi", wantRoute: "wasm:wasm32-wasi", wantCheck: "wasm check"},
	} {
		t.Run(tc.target, func(t *testing.T) {
			routed = nil
			checks, err := RunTargetChecks(tc.target, runners)
			if err != nil {
				t.Fatalf("RunTargetChecks(%s): %v", tc.target, err)
			}
			if strings.Join(routed, ",") != tc.wantRoute {
				t.Fatalf("route = %#v, want %q", routed, tc.wantRoute)
			}
			if len(checks) != 1 || checks[0].Name != tc.wantCheck {
				t.Fatalf("checks = %#v, want %q", checks, tc.wantCheck)
			}
		})
	}
}

func TestRunTargetChecksRejectsMissingRunner(t *testing.T) {
	_, err := RunTargetChecks("x86", TargetCheckRunners{})
	if err == nil || !strings.Contains(err.Error(), "missing ABI suite runner for x86") {
		t.Fatalf("RunTargetChecks missing runner error = %v", err)
	}
}

func TestX64CheckPrefix(t *testing.T) {
	for _, tc := range []struct {
		target string
		want   string
	}{
		{target: "linux-x64", want: "x64"},
		{target: "windows-x64", want: "windows-x64"},
		{target: "macos-x64", want: "macos-x64"},
	} {
		t.Run(tc.target, func(t *testing.T) {
			tgt, err := ctarget.Parse(tc.target)
			if err != nil {
				t.Fatalf("Parse(%s): %v", tc.target, err)
			}
			if got := X64CheckPrefix(tgt); got != tc.want {
				t.Fatalf("X64CheckPrefix(%s) = %q, want %q", tc.target, got, tc.want)
			}
		})
	}
}

func targetRoutingRunner(name string, routed *[]string) TargetCheckRunner {
	return func(tgt ctarget.Target) []Check {
		*routed = append(*routed, name+":"+tgt.Triple)
		return []Check{{Name: name + " check"}}
	}
}

// ---- verification_test.go ----

func TestBuildP21VerificationReportCoversTargetsTasksAndNonClaims(t *testing.T) {
	report := BuildP21VerificationReport()
	if report.Schema != VerificationSchemaV1 {
		t.Fatalf("schema = %q, want %q", report.Schema, VerificationSchemaV1)
	}
	if report.Scope != VerificationScopeP211 {
		t.Fatalf("scope = %q, want %q", report.Scope, VerificationScopeP211)
	}
	if err := ValidateP21VerificationReport(report); err != nil {
		t.Fatalf("ValidateP21VerificationReport: %v", err)
	}
	if len(report.Targets) != len(P21VerificationTargets()) {
		t.Fatalf("targets len = %d, want %d", len(report.Targets), len(P21VerificationTargets()))
	}
	if len(report.Tasks) != len(P21VerificationTaskIDs()) {
		t.Fatalf("tasks len = %d, want %d", len(report.Tasks), len(P21VerificationTaskIDs()))
	}
	for _, nonClaim := range P21VerificationNonClaims() {
		if !stringSliceHas(report.NonClaims, nonClaim) {
			t.Fatalf("missing non-claim %q", nonClaim)
		}
	}
}

func TestValidateP21VerificationReportRejectsFakeClaims(t *testing.T) {
	report := BuildP21VerificationReport()
	report.Claims = append(report.Claims, "full runtime execution verified")
	err := ValidateP21VerificationReport(report)
	if err == nil || !strings.Contains(err.Error(), "runtime execution") {
		t.Fatalf("ValidateP21VerificationReport fake claim err = %v", err)
	}
}

// ---- wasm_test.go ----

func TestWASMABIChecks(t *testing.T) {
	for _, targetName := range []string{"wasm32-wasi", "wasm32-web"} {
		t.Run(targetName, func(t *testing.T) {
			tgt, err := ctarget.Parse(targetName)
			if err != nil {
				t.Fatalf("parse target: %v", err)
			}
			for _, check := range []struct {
				name string
				run  func(ctarget.Target) error
			}{
				{name: "target model", run: CheckWASMTargetModel},
				{name: "slot ABI metadata", run: CheckWASMSlotABIMetadata},
				{name: "aggregate return layouts", run: CheckWASMAggregateReturnLayouts},
				{name: "call boundary validation", run: CheckWASMCallBoundaryValidation},
				{name: "FFI repr(C) policy", run: CheckWASMFFIReprCBoundaryPolicy},
			} {
				if err := check.run(tgt); err != nil {
					t.Fatalf("%s: %v", check.name, err)
				}
			}
		})
	}
}

func TestWASMFFIReprCBoundaryPolicyPredicates(t *testing.T) {
	for _, targetName := range []string{"wasm32-wasi", "wasm32-web"} {
		if TargetRequiresExplicitAggregateExportGate(targetName) {
			t.Fatalf("%s unexpectedly requires native aggregate C ABI gate", targetName)
		}
		if TargetRequiresExplicitPointerExportGate(targetName) {
			t.Fatalf("%s unexpectedly requires native pointer C ABI gate", targetName)
		}
	}
	for _, targetName := range []string{
		"linux-x86",
		"linux-x64",
		"linux-x32",
		"macos-x64",
		"windows-x64",
	} {
		if !TargetRequiresExplicitAggregateExportGate(targetName) {
			t.Fatalf("%s lost native aggregate C ABI gate", targetName)
		}
	}

	types := map[string]*semantics.TypeInfo{
		"Pair":   {Name: "Pair", Kind: semantics.TypeStruct},
		"Bytes":  {Name: "Bytes", Kind: semantics.TypeSlice},
		"String": {Name: "String", Kind: semantics.TypeStr},
		"Choice": {Name: "Choice", Kind: semantics.TypeEnum},
	}
	for _, typeName := range []string{"Pair", "Bytes", "String", "Choice"} {
		if !TargetExportedFFIRequiresAggregateABI(typeName, types) {
			t.Fatalf("aggregate FFI detector did not recognize %s", typeName)
		}
	}
	if TargetExportedFFIRequiresAggregateABI("i32", types) {
		t.Fatalf("aggregate FFI detector should not recognize missing scalar i32 as aggregate")
	}
}

// ---- x64_runtime_test.go ----

func TestX64AndSourceNativeChecksUseFFICallbacks(t *testing.T) {
	var lastTarget string
	deps := FFICheckDeps{
		BuildLibrary: func(srcPath string, outPath string, target string) error {
			lastTarget = target
			if strings.Contains(srcPath, "u32_param") ||
				strings.Contains(srcPath, "u64_param") ||
				strings.Contains(srcPath, "f64_return") ||
				strings.Contains(srcPath, "usize_param") ||
				strings.Contains(srcPath, "size_t_param") ||
				strings.Contains(srcPath, "native_int_return") ||
				strings.Contains(srcPath, "c_long_return") {
				return fmt.Errorf(
					("target-layout scalar type is not supported in source-level " +
						"Tetra yet; native-int/codegen support is required"),
				)
			}
			return nil
		},
		ReadObject: func(path string) (ObjectSummary, error) {
			obj := ObjectSummary{
				Target: lastTarget,
				Data:   []byte(lastTarget + " abi\n"),
				Symbols: []ObjectSymbolSummary{
					{Name: "ffi_say_i32", HasSignature: true, ParamSlots: 0, ReturnSlots: 1},
					{Name: "ffi_ptr_param_c", HasSignature: true, ParamSlots: 1, ReturnSlots: 1},
				},
				Relocs: []ObjectRelocSummary{
					{Kind: ObjectRelocDataDisp32},
				},
			}
			if lastTarget == "windows-x64" {
				obj.Relocs = append(obj.Relocs,
					ObjectRelocSummary{Kind: ObjectRelocIATDisp32, Name: "kernel32.GetStdHandle"},
					ObjectRelocSummary{Kind: ObjectRelocIATDisp32, Name: "kernel32.WriteFile"},
				)
			}
			return obj, nil
		},
	}

	for _, targetName := range []string{"linux-x86", "windows-x64"} {
		t.Run("source native scalar "+targetName, func(t *testing.T) {
			tgt, err := ctarget.Parse(targetName)
			if err != nil {
				t.Fatalf("parse target: %v", err)
			}
			if err := CheckSourceNativeScalarDiagnostics(tgt, deps); err != nil {
				t.Fatalf("source native scalar diagnostics: %v", err)
			}
		})
	}

	for _, targetName := range []string{"macos-x64", "windows-x64"} {
		t.Run("platform object "+targetName, func(t *testing.T) {
			tgt, err := ctarget.Parse(targetName)
			if err != nil {
				t.Fatalf("parse target: %v", err)
			}
			if err := CheckX64PlatformObjectABISmoke(tgt, deps); err != nil {
				t.Fatalf("x64 platform object smoke: %v", err)
			}
		})
	}

	t.Run("linux x64 pointer FFI regression", func(t *testing.T) {
		if err := CheckX64PointerFFIRegressionSmoke(deps); err != nil {
			t.Fatalf("x64 pointer FFI regression: %v", err)
		}
	})
}

func TestX64RuntimeExecutionSmokesUseCallbacks(t *testing.T) {
	var built []string
	var ran []string
	deps := RuntimeSmokeDeps{
		HostGOOS:   "linux",
		HostGOARCH: "amd64",
		BuildExecutable: func(srcPath string, outPath string, target string) error {
			built = append(built, filepath.Base(outPath)+":"+target)
			data := make([]byte, 20)
			copy(data[:4], "\x7fELF")
			data[4] = 2
			data[18] = 0x3e
			return os.WriteFile(outPath, data, 0o755)
		},
		RunExecutable: func(path string) RuntimeRunResult {
			ran = append(ran, filepath.Base(path))
			if strings.Contains(path, "scheduler-regression") {
				return RuntimeRunResult{ExitCode: 42}
			}
			return RuntimeRunResult{}
		},
	}

	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{name: "filesystem scheduler", run: func() error {
			return CheckX64FilesystemSchedulerCompositionSmoke(
				deps,
			)
		}},
		{name: "networking", run: func() error { return CheckX64NetworkingRuntimeSmoke(deps) }},
		{name: "scheduler restriction", run: func() error {
			return CheckX64SchedulerRestrictionRegressionSmoke(
				deps,
			)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err != nil {
				t.Fatalf("x64 runtime smoke: %v", err)
			}
		})
	}
	if len(built) != 3 {
		t.Fatalf("built %d executables, want 3: %#v", len(built), built)
	}
	if len(ran) != 2 {
		t.Fatalf("ran %d executables, want 2: %#v", len(ran), ran)
	}
}
