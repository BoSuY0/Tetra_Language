package compiler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"tetra_language/compiler/internal/abisuite"
	"tetra_language/compiler/internal/actorsafety"
	"tetra_language/compiler/internal/actorsrt"
	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/differential"
	"tetra_language/compiler/internal/formalcore"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/httprt"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/memoryfacts"
	"tetra_language/compiler/internal/netrt"
	"tetra_language/compiler/internal/opt"
	"tetra_language/compiler/internal/parallelrt"
	"tetra_language/compiler/internal/pgrt"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/runtimeabi"
	"tetra_language/compiler/internal/selfhostgate"
	"tetra_language/compiler/internal/semantics"
	"tetra_language/compiler/internal/stdlibrt"
	"tetra_language/compiler/internal/validation"
	"tetra_language/compiler/memoryvocab"
	ctarget "tetra_language/compiler/target"
)

// ---- abi_suite.go ----

type ABICheck = abisuite.Check

func RunTargetABIChecks(targetName string) ([]ABICheck, error) {
	return abisuite.RunTargetChecks(targetName, abisuite.TargetCheckRunners{
		X86:  runX86ABIChecks,
		X32:  runX32ABIChecks,
		X64:  runX64ABIChecks,
		WASM: runWASMABIChecks,
	})
}

func runX86ABIChecks(tgt ctarget.Target) []ABICheck {
	return runABIChecks([]struct {
		name string
		run  func() error
	}{
		{name: "x86 target model", run: func() error { return checkX86TargetModel(tgt) }},
		{
			name: "x86 i386 SysV classifier",
			run:  func() error { return checkX86I386Classifier(tgt) },
		},
		{
			name: "x86 varargs and sret ABI",
			run:  func() error { return checkX86VarargsAndSRet(tgt) },
		},
		{
			name: "x86 pointer FFI object smoke",
			run:  func() error { return checkPointerFFIObjectSmoke(tgt) },
		},
		{
			name: "x86 c_int FFI object smoke",
			run:  func() error { return checkCIntFFIObjectSmoke(tgt) },
		},
		{
			name: "x86 c_uint FFI object smoke",
			run:  func() error { return checkCUIntFFIObjectSmoke(tgt) },
		},
		{
			name: "x86 ILP32 native/libc FFI object smoke",
			run:  func() error { return checkILP32NativeLibcFFIObjectSmoke(tgt) },
		},
		{name: "x86 ref FFI null-return diagnostics", run: checkX86RefFFINullReturnDiagnostics},
		{name: "x86 function-pointer FFI diagnostics", run: checkX86FunctionPointerFFIDiagnostics},
		{
			name: "x86 source native scalar diagnostics",
			run:  func() error { return checkSourceNativeScalarDiagnostics(tgt) },
		},
		{name: "x86 stdout executable smoke", run: checkX86StdoutExecutableSmoke},
		{name: "x86 stderr fd runtime smoke", run: checkX86StderrFDRuntimeSmoke},
		{name: "x86 allocator executable smoke", run: checkX86AllocatorExecutableSmoke},
		{
			name: "x86 allocator failure executable smoke",
			run:  checkX86AllocatorFailureExecutableSmoke,
		},
		{
			name: "x86 raw memory bounds executable smoke",
			run:  checkX86RawMemoryBoundsExecutableSmoke,
		},
		{name: "x86 raw pointer slot executable smoke", run: checkX86RawPointerSlotExecutableSmoke},
		{
			name: "x86 raw pointer offset slot executable smoke",
			run:  checkX86RawPointerOffsetSlotExecutableSmoke,
		},
		{name: "x86 island free executable smoke", run: checkX86IslandFreeExecutableSmoke},
		{
			name: "x86 stdlib runtime boundary diagnostics",
			run:  func() error { return checkStdlibRuntimeBoundaryDiagnostics(tgt) },
		},
		{name: "x86 filesystem runtime smoke", run: checkX86FilesystemRuntimeSmoke},
		{
			name: "x86 filesystem scheduler composition smoke",
			run:  checkX86FilesystemSchedulerCompositionSmoke,
		},
		{name: "x86 time runtime smoke", run: checkX86TimeRuntimeSmoke},
		{
			name: "x86 single-actor self-host runtime smoke",
			run:  checkX86SingleActorSelfHostRuntimeSmoke,
		},
		{
			name: "x86 single-task self-host runtime smoke",
			run:  checkX86SingleTaskSelfHostRuntimeSmoke,
		},
		{
			name: "x86 typed-task self-host runtime smoke",
			run:  checkX86TypedTaskSelfHostRuntimeSmoke,
		},
		{
			name: "x86 staged typed-task self-host runtime smoke",
			run:  checkX86StagedTypedTaskSelfHostRuntimeSmoke,
		},
		{
			name: "x86 task-group self-host runtime smoke",
			run:  checkX86TaskGroupSelfHostRuntimeSmoke,
		},
		{
			name: "x86 typed-task-group self-host runtime smoke",
			run:  checkX86TypedTaskGroupSelfHostRuntimeSmoke,
		},
		{
			name: "x86 actor-state self-host runtime smoke",
			run:  checkX86ActorStateSelfHostRuntimeSmoke,
		},
		{name: "x86 ctx_switch object smoke", run: checkX86CtxSwitchObjectSmoke},
		{
			name: "x86 target runtime boundary diagnostics",
			run:  func() error { return checkTargetRuntimeBoundaryDiagnostics(tgt) },
		},
		{
			name: "x86 networking runtime boundary diagnostics",
			run:  func() error { return checkNetworkingRuntimeBoundaryDiagnostics(tgt) },
		},
		{
			name: "x86 networking lifecycle runtime smoke",
			run:  checkX86NetworkingLifecycleRuntimeSmoke,
		},
		{
			name: "x86 surface/distributed runtime boundary diagnostics",
			run:  func() error { return checkSurfaceDistributedRuntimeBoundaryDiagnostics(tgt) },
		},
		{
			name: "x86 pointer atomic ABI width",
			run:  func() error { return checkAtomicPointerObjectWidth(tgt) },
		},
	})
}

func runX64ABIChecks(tgt ctarget.Target) []ABICheck {
	abiName := "SysV"
	if tgt.ABI == ctarget.ABIWin64 {
		abiName = "Win64"
	}
	prefix := abisuite.X64CheckPrefix(tgt)
	checks := []struct {
		name string
		run  func() error
	}{
		{name: prefix + " target model", run: func() error { return checkX64TargetModel(tgt) }},
		{
			name: prefix + " " + abiName + " classifier",
			run:  func() error { return checkX64Classifier(tgt) },
		},
		{
			name: prefix + " " + abiName + " varargs and aggregates",
			run:  func() error { return checkX64VarargsAndAggregates(tgt) },
		},
	}
	if tgt.Triple == "macos-x64" || tgt.Triple == "windows-x64" {
		checks = append(checks, struct {
			name string
			run  func() error
		}{name: prefix + " object ABI smoke", run: func() error {
			return checkX64PlatformObjectABISmoke(
				tgt,
			)
		}})
	}
	checks = append(checks, struct {
		name string
		run  func() error
	}{name: prefix + " source native scalar diagnostics", run: func() error {
		return checkSourceNativeScalarDiagnostics(
			tgt,
		)
	}})
	if tgt.Triple == "linux-x64" {
		checks = append(checks, struct {
			name string
			run  func() error
		}{name: prefix + " pointer FFI regression smoke", run: checkX64PointerFFIRegressionSmoke})
		checks = append(checks, struct {
			name string
			run  func() error
		}{name: prefix + " c_int FFI object smoke", run: func() error {
			return checkCIntFFIObjectSmoke(
				tgt,
			)
		}})
		checks = append(checks, struct {
			name string
			run  func() error
		}{name: prefix + " c_uint FFI object smoke", run: func() error {
			return checkCUIntFFIObjectSmoke(
				tgt,
			)
		}})
		checks = append(checks, struct {
			name string
			run  func() error
		}{
			name: prefix + " filesystem scheduler composition smoke",
			run:  checkX64FilesystemSchedulerCompositionSmoke,
		})
		checks = append(checks, struct {
			name string
			run  func() error
		}{name: prefix + " networking runtime smoke", run: checkX64NetworkingRuntimeSmoke})
		checks = append(checks, struct {
			name string
			run  func() error
		}{
			name: prefix + " scheduler restriction regression smoke",
			run:  checkX64SchedulerRestrictionRegressionSmoke,
		})
	}
	checks = append(checks, struct {
		name string
		run  func() error
	}{name: prefix + " pointer atomic ABI width", run: func() error {
		return checkAtomicPointerObjectWidth(
			tgt,
		)
	}})
	return runABIChecks(checks)
}

func runX32ABIChecks(tgt ctarget.Target) []ABICheck {
	return runABIChecks([]struct {
		name string
		run  func() error
	}{
		{name: "x32 target model", run: func() error { return checkX32TargetModel(tgt) }},
		{name: "x32 SysV classifier", run: func() error { return checkX32SysVClassifier(tgt) }},
		{
			name: "x32 SysV varargs and aggregates",
			run:  func() error { return checkX32SysVVarargsAndAggregates(tgt) },
		},
		{
			name: "x32 pointer FFI object smoke",
			run:  func() error { return checkPointerFFIObjectSmoke(tgt) },
		},
		{
			name: "x32 c_int FFI object smoke",
			run:  func() error { return checkCIntFFIObjectSmoke(tgt) },
		},
		{
			name: "x32 c_uint FFI object smoke",
			run:  func() error { return checkCUIntFFIObjectSmoke(tgt) },
		},
		{
			name: "x32 ILP32 native/libc FFI object smoke",
			run:  func() error { return checkILP32NativeLibcFFIObjectSmoke(tgt) },
		},
		{name: "x32 ref FFI null-return diagnostics", run: checkX32RefFFINullReturnDiagnostics},
		{name: "x32 function-pointer FFI diagnostics", run: checkX32FunctionPointerFFIDiagnostics},
		{
			name: "x32 source native scalar diagnostics",
			run:  func() error { return checkSourceNativeScalarDiagnostics(tgt) },
		},
		{name: "x32 stdout executable smoke", run: checkX32StdoutExecutableSmoke},
		{name: "x32 stderr fd runtime smoke", run: checkX32StderrFDRuntimeSmoke},
		{name: "x32 allocator executable smoke", run: checkX32AllocatorExecutableSmoke},
		{
			name: "x32 allocator failure executable smoke",
			run:  checkX32AllocatorFailureExecutableSmoke,
		},
		{
			name: "x32 raw memory bounds executable smoke",
			run:  checkX32RawMemoryBoundsExecutableSmoke,
		},
		{name: "x32 raw pointer slot executable smoke", run: checkX32RawPointerSlotExecutableSmoke},
		{
			name: "x32 raw pointer offset slot executable smoke",
			run:  checkX32RawPointerOffsetSlotExecutableSmoke,
		},
		{name: "x32 island free executable smoke", run: checkX32IslandFreeExecutableSmoke},
		{
			name: "x32 stdlib runtime boundary diagnostics",
			run:  func() error { return checkStdlibRuntimeBoundaryDiagnostics(tgt) },
		},
		{name: "x32 time runtime smoke", run: checkX32TimeRuntimeSmoke},
		{name: "x32 filesystem runtime smoke", run: checkX32FilesystemRuntimeSmoke},
		{
			name: "x32 filesystem scheduler composition smoke",
			run:  checkX32FilesystemSchedulerCompositionSmoke,
		},
		{
			name: "x32 single-actor self-host runtime smoke",
			run:  checkX32SingleActorSelfHostRuntimeSmoke,
		},
		{
			name: "x32 single-task self-host runtime smoke",
			run:  checkX32SingleTaskSelfHostRuntimeSmoke,
		},
		{
			name: "x32 typed-task self-host runtime smoke",
			run:  checkX32TypedTaskSelfHostRuntimeSmoke,
		},
		{
			name: "x32 staged typed-task self-host runtime smoke",
			run:  checkX32StagedTypedTaskSelfHostRuntimeSmoke,
		},
		{
			name: "x32 task-group self-host runtime smoke",
			run:  checkX32TaskGroupSelfHostRuntimeSmoke,
		},
		{
			name: "x32 typed-task-group self-host runtime smoke",
			run:  checkX32TypedTaskGroupSelfHostRuntimeSmoke,
		},
		{
			name: "x32 actor-state self-host runtime smoke",
			run:  checkX32ActorStateSelfHostRuntimeSmoke,
		},
		{name: "x32 ctx_switch object smoke", run: checkX32CtxSwitchObjectSmoke},
		{
			name: "x32 target runtime boundary diagnostics",
			run:  func() error { return checkTargetRuntimeBoundaryDiagnostics(tgt) },
		},
		{
			name: "x32 networking runtime boundary diagnostics",
			run:  func() error { return checkNetworkingRuntimeBoundaryDiagnostics(tgt) },
		},
		{
			name: "x32 networking lifecycle runtime smoke",
			run:  checkX32NetworkingLifecycleRuntimeSmoke,
		},
		{
			name: "x32 surface/distributed runtime boundary diagnostics",
			run:  func() error { return checkSurfaceDistributedRuntimeBoundaryDiagnostics(tgt) },
		},
		{
			name: "x32 pointer atomic ABI width",
			run:  func() error { return checkAtomicPointerObjectWidth(tgt) },
		},
	})
}

func runABIChecks(cases []struct {
	name string
	run  func() error
}) []ABICheck {
	checks := make([]abisuite.Case, 0, len(cases))
	for _, tc := range cases {
		checks = append(checks, abisuite.Case{Name: tc.name, Run: tc.run})
	}
	return abisuite.RunChecks(checks)
}

// ---- abi_suite_classifiers.go ----

func checkX86TargetModel(tgt ctarget.Target) error {
	return abisuite.CheckX86TargetModel(tgt)
}

func checkX86I386Classifier(tgt ctarget.Target) error {
	return abisuite.CheckX86I386Classifier(tgt)
}

func checkX86VarargsAndSRet(tgt ctarget.Target) error {
	return abisuite.CheckX86VarargsAndSRet(tgt)
}

func checkX64TargetModel(tgt ctarget.Target) error {
	return abisuite.CheckX64TargetModel(tgt)
}

func checkX64Classifier(tgt ctarget.Target) error {
	return abisuite.CheckX64Classifier(tgt)
}

func checkX64VarargsAndAggregates(tgt ctarget.Target) error {
	return abisuite.CheckX64VarargsAndAggregates(tgt)
}

func checkX32TargetModel(tgt ctarget.Target) error {
	return abisuite.CheckX32TargetModel(tgt)
}

func expectTargetScalarLayout(tgt ctarget.Target, name string, size int, align int) error {
	return abisuite.ExpectTargetScalarLayout(tgt, name, size, align)
}

func checkX32SysVClassifier(tgt ctarget.Target) error {
	return abisuite.CheckX32SysVClassifier(tgt)
}

func checkX32SysVVarargsAndAggregates(tgt ctarget.Target) error {
	return abisuite.CheckX32SysVVarargsAndAggregates(tgt)
}

// ---- abi_suite_ffi.go ----

func checkX86RefFFINullReturnDiagnostics() error {
	return abisuite.CheckX86RefFFINullReturnDiagnostics(abiSuiteFFICheckDeps())
}

func checkX32RefFFINullReturnDiagnostics() error {
	return abisuite.CheckX32RefFFINullReturnDiagnostics(abiSuiteFFICheckDeps())
}

func checkX86FunctionPointerFFIDiagnostics() error {
	return abisuite.CheckX86FunctionPointerFFIDiagnostics(abiSuiteFFICheckDeps())
}

func checkX32FunctionPointerFFIDiagnostics() error {
	return abisuite.CheckX32FunctionPointerFFIDiagnostics(abiSuiteFFICheckDeps())
}

func checkPointerFFIObjectSmoke(tgt ctarget.Target) error {
	return abisuite.CheckPointerFFIObjectSmoke(tgt, abiSuiteFFICheckDeps())
}

func checkCIntFFIObjectSmoke(tgt ctarget.Target) error {
	return abisuite.CheckCIntFFIObjectSmoke(tgt, abiSuiteFFICheckDeps())
}

func checkCUIntFFIObjectSmoke(tgt ctarget.Target) error {
	return abisuite.CheckCUIntFFIObjectSmoke(tgt, abiSuiteFFICheckDeps())
}

func checkILP32NativeLibcFFIObjectSmoke(tgt ctarget.Target) error {
	return abisuite.CheckILP32NativeLibcFFIObjectSmoke(tgt, abiSuiteFFICheckDeps())
}

func checkRefFFINullReturnDiagnostics(targetName, stem string) error {
	return abisuite.CheckRefFFINullReturnDiagnostics(targetName, stem, abiSuiteFFICheckDeps())
}

func checkFunctionPointerFFIDiagnostics(targetName, boundaryName, stem string) error {
	return abisuite.CheckFunctionPointerFFIDiagnostics(
		targetName,
		boundaryName,
		stem,
		abiSuiteFFICheckDeps(),
	)
}

func abiSuiteFFICheckDeps() abisuite.FFICheckDeps {
	return abisuite.FFICheckDeps{
		BuildLibrary: func(srcPath string, outPath string, target string) error {
			_, err := BuildFileWithStatsOpt(
				srcPath,
				outPath,
				target,
				BuildOptions{Emit: EmitLibrary, Jobs: 1},
			)
			return err
		},
		ReadObject: func(path string) (abisuite.ObjectSummary, error) {
			obj, err := ReadObject(path)
			if err != nil {
				return abisuite.ObjectSummary{}, err
			}
			symbols := make([]abisuite.ObjectSymbolSummary, 0, len(obj.Symbols))
			for _, sym := range obj.Symbols {
				symbols = append(symbols, abisuite.ObjectSymbolSummary{
					Name:         sym.Name,
					HasSignature: sym.HasSignature,
					ParamSlots:   sym.ParamSlots,
					ReturnSlots:  sym.ReturnSlots,
				})
			}
			relocs := make([]abisuite.ObjectRelocSummary, 0, len(obj.Relocs))
			for _, reloc := range obj.Relocs {
				relocs = append(relocs, abisuite.ObjectRelocSummary{
					Kind: abisuite.ObjectRelocKind(reloc.Kind),
					Name: reloc.Name,
				})
			}
			return abisuite.ObjectSummary{
				Target:  obj.Target,
				Data:    obj.Data,
				Symbols: symbols,
				Relocs:  relocs,
			}, nil
		},
	}
}

// ---- abi_suite_runtime_boundaries.go ----

func checkStdlibRuntimeBoundaryDiagnostics(tgt ctarget.Target) error {
	return abisuite.CheckStdlibRuntimeBoundaryDiagnostics(tgt, abiSuiteRuntimeBoundaryDeps())
}

func checkX86TimeRuntimeSmoke() error {
	return abisuite.CheckX86TimeRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86FilesystemRuntimeSmoke() error {
	return abisuite.CheckX86FilesystemRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86FilesystemSchedulerCompositionSmoke() error {
	return abisuite.CheckX86FilesystemSchedulerCompositionSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32TimeRuntimeSmoke() error {
	return abisuite.CheckX32TimeRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32FilesystemRuntimeSmoke() error {
	return abisuite.CheckX32FilesystemRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32SingleTaskSelfHostRuntimeSmoke() error {
	return abisuite.CheckX32SingleTaskSelfHostRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32TypedTaskSelfHostRuntimeSmoke() error {
	return abisuite.CheckX32TypedTaskSelfHostRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32StagedTypedTaskSelfHostRuntimeSmoke() error {
	return abisuite.CheckX32StagedTypedTaskSelfHostRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32TaskGroupSelfHostRuntimeSmoke() error {
	return abisuite.CheckX32TaskGroupSelfHostRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32TypedTaskGroupSelfHostRuntimeSmoke() error {
	return abisuite.CheckX32TypedTaskGroupSelfHostRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32FilesystemSchedulerCompositionSmoke() error {
	return abisuite.CheckX32FilesystemSchedulerCompositionSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32SingleActorSelfHostRuntimeSmoke() error {
	return abisuite.CheckX32SingleActorSelfHostRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32ActorStateSelfHostRuntimeSmoke() error {
	return abisuite.CheckX32ActorStateSelfHostRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86SingleTaskSelfHostRuntimeSmoke() error {
	return abisuite.CheckX86SingleTaskSelfHostRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86TypedTaskSelfHostRuntimeSmoke() error {
	return abisuite.CheckX86TypedTaskSelfHostRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86StagedTypedTaskSelfHostRuntimeSmoke() error {
	return abisuite.CheckX86StagedTypedTaskSelfHostRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86TaskGroupSelfHostRuntimeSmoke() error {
	return abisuite.CheckX86TaskGroupSelfHostRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86TypedTaskGroupSelfHostRuntimeSmoke() error {
	return abisuite.CheckX86TypedTaskGroupSelfHostRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86SingleActorSelfHostRuntimeSmoke() error {
	return abisuite.CheckX86SingleActorSelfHostRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86ActorStateSelfHostRuntimeSmoke() error {
	return abisuite.CheckX86ActorStateSelfHostRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86CtxSwitchObjectSmoke() error {
	return abisuite.CheckX86CtxSwitchObjectSmoke(abisuite.CtxSwitchDeps{})
}

func checkX32CtxSwitchObjectSmoke() error {
	return abisuite.CheckX32CtxSwitchObjectSmoke(abisuite.CtxSwitchDeps{})
}

func checkTargetRuntimeBoundaryDiagnostics(tgt ctarget.Target) error {
	return abisuite.CheckTargetRuntimeBoundaryDiagnostics(tgt, abiSuiteRuntimeBoundaryDeps())
}

func checkSurfaceDistributedRuntimeBoundaryDiagnostics(tgt ctarget.Target) error {
	return abisuite.CheckSurfaceDistributedRuntimeBoundaryDiagnostics(
		tgt,
		abiSuiteRuntimeBoundaryDeps(),
	)
}

func checkNetworkingRuntimeBoundaryDiagnostics(tgt ctarget.Target) error {
	return abisuite.CheckNetworkingRuntimeBoundaryDiagnostics(tgt, abiSuiteRuntimeBoundaryDeps())
}

func abiSuiteRuntimeBoundaryDeps() abisuite.RuntimeBoundaryDeps {
	return abisuite.RuntimeBoundaryDeps{
		BuildExecutable: func(srcPath string, outPath string, target string) error {
			_, err := BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{Jobs: 1})
			return err
		},
		DiagnosticFromError: func(err error) abisuite.DiagnosticSummary {
			diag := DiagnosticFromError(err)
			return abisuite.DiagnosticSummary{
				Code:     diag.Code,
				Message:  diag.Message,
				Severity: diag.Severity,
				Hint:     diag.Hint,
			}
		},
		TargetRuntimeDiagnosticCode: DiagnosticCodeTargetRuntime,
		TargetSupportsNetRuntimeSymbols: func(target string, symbols []string) bool {
			return targetSupportsNetRuntimeSymbols(target, symbols)
		},
		RequiredNetRuntimeSymbols: func() []string {
			return requiredNetRuntimeSymbols()
		},
		NetRuntimeSymbolForBuiltin: func(name string) (string, bool) {
			return netRuntimeSymbolForBuiltin(name)
		},
	}
}

// ---- abi_suite_runtime_smoke.go ----

func checkX86StdoutExecutableSmoke() error {
	return abisuite.CheckX86StdoutExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32StdoutExecutableSmoke() error {
	return abisuite.CheckX32StdoutExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86StderrFDRuntimeSmoke() error {
	return abisuite.CheckX86StderrFDRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32StderrFDRuntimeSmoke() error {
	return abisuite.CheckX32StderrFDRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86AllocatorExecutableSmoke() error {
	return abisuite.CheckX86AllocatorExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86AllocatorFailureExecutableSmoke() error {
	return abisuite.CheckX86AllocatorFailureExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32AllocatorExecutableSmoke() error {
	return abisuite.CheckX32AllocatorExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32AllocatorFailureExecutableSmoke() error {
	return abisuite.CheckX32AllocatorFailureExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86RawMemoryBoundsExecutableSmoke() error {
	return abisuite.CheckX86RawMemoryBoundsExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32RawMemoryBoundsExecutableSmoke() error {
	return abisuite.CheckX32RawMemoryBoundsExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86RawPointerSlotExecutableSmoke() error {
	return abisuite.CheckX86RawPointerSlotExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32RawPointerSlotExecutableSmoke() error {
	return abisuite.CheckX32RawPointerSlotExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86RawPointerOffsetSlotExecutableSmoke() error {
	return abisuite.CheckX86RawPointerOffsetSlotExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32RawPointerOffsetSlotExecutableSmoke() error {
	return abisuite.CheckX32RawPointerOffsetSlotExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86IslandFreeExecutableSmoke() error {
	return abisuite.CheckX86IslandFreeExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32IslandFreeExecutableSmoke() error {
	return abisuite.CheckX32IslandFreeExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86NetworkingLifecycleRuntimeSmoke() error {
	return abisuite.CheckX86NetworkingLifecycleRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32NetworkingLifecycleRuntimeSmoke() error {
	return abisuite.CheckX32NetworkingLifecycleRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

// ---- abi_suite_x64_runtime.go ----

func checkSourceNativeScalarDiagnostics(tgt ctarget.Target) error {
	return abisuite.CheckSourceNativeScalarDiagnostics(tgt, abiSuiteFFICheckDeps())
}

func checkX64PlatformObjectABISmoke(tgt ctarget.Target) error {
	return abisuite.CheckX64PlatformObjectABISmoke(tgt, abiSuiteFFICheckDeps())
}

func checkX64PointerFFIRegressionSmoke() error {
	return abisuite.CheckX64PointerFFIRegressionSmoke(abiSuiteFFICheckDeps())
}

func checkX64FilesystemSchedulerCompositionSmoke() error {
	return abisuite.CheckX64FilesystemSchedulerCompositionSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX64NetworkingRuntimeSmoke() error {
	return abisuite.CheckX64NetworkingRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX64SchedulerRestrictionRegressionSmoke() error {
	return abisuite.CheckX64SchedulerRestrictionRegressionSmoke(abiSuiteRuntimeSmokeDeps())
}

func abiSuiteRuntimeSmokeDeps() abisuite.RuntimeSmokeDeps {
	return abisuite.RuntimeSmokeDeps{
		BuildExecutable: func(srcPath string, outPath string, target string) error {
			_, err := BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{Jobs: 1})
			return err
		},
		BuildExecutableWithOptions: func(
			srcPath string,
			outPath string,
			target string,
			opts abisuite.RuntimeBuildOptions,
		) error {
			_, err := BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{
				Jobs:         1,
				IslandsDebug: opts.IslandsDebug,
			})
			return err
		},
	}
}

func abiSuiteObjectHasSymbolSignature(obj *Object, name string, params, returns int) bool {
	for _, sym := range obj.Symbols {
		if sym.Name == name && sym.HasSignature && sym.ParamSlots == params &&
			sym.ReturnSlots == returns {
			return true
		}
	}
	return false
}

func abiSuiteObjectHasRelocKind(obj *Object, kind RelocKind) bool {
	for _, reloc := range obj.Relocs {
		if reloc.Kind == kind {
			return true
		}
	}
	return false
}

func abiSuiteObjectHasReloc(obj *Object, kind RelocKind, name string) bool {
	for _, reloc := range obj.Relocs {
		if reloc.Kind == kind && reloc.Name == name {
			return true
		}
	}
	return false
}

// ---- abi_verification.go ----

const (
	abiVerificationSchemaV1  = abisuite.VerificationSchemaV1
	abiVerificationScopeP211 = abisuite.VerificationScopeP211
)

const (
	abiVerificationTaskCorpus           = abisuite.VerificationTaskCorpus
	abiVerificationTaskAggregateReturns = abisuite.VerificationTaskAggregateReturns
	abiVerificationTaskCallBoundary     = abisuite.VerificationTaskCallBoundary
	abiVerificationTaskFFIReprC         = abisuite.VerificationTaskFFIReprC
)

type ABIVerificationReport = abisuite.VerificationReport
type ABIVerificationTargetRow = abisuite.VerificationTargetRow
type ABIVerificationTaskRow = abisuite.VerificationTaskRow

func BuildP21ABIVerificationReport() ABIVerificationReport {
	return abisuite.BuildP21VerificationReport()
}

func ValidateP21ABIVerificationReport(report ABIVerificationReport) error {
	return abisuite.ValidateP21VerificationReport(report)
}

func p21ABIVerificationTargets() []string {
	return abisuite.P21VerificationTargets()
}

func p21ABIVerificationTaskIDs() []string {
	return abisuite.P21VerificationTaskIDs()
}

func p21ABIVerificationNonClaims() []string {
	return abisuite.P21VerificationNonClaims()
}

// ---- abi_wasm_suite.go ----

func runWASMABIChecks(tgt ctarget.Target) []ABICheck {
	prefix := tgt.Triple
	return runABIChecks([]struct {
		name string
		run  func() error
	}{
		{name: prefix + " target model", run: func() error { return checkWASMTargetModel(tgt) }},
		{
			name: prefix + " slot ABI metadata",
			run:  func() error { return checkWASMSlotABIMetadata(tgt) },
		},
		{
			name: prefix + " struct/enum/slice/String return layout",
			run:  func() error { return checkWASMAggregateReturnLayouts(tgt) },
		},
		{
			name: prefix + " call boundary validation",
			run:  func() error { return checkWASMCallBoundaryValidation(tgt) },
		},
		{
			name: prefix + " FFI repr(C) boundary policy",
			run:  func() error { return checkWASMFFIReprCBoundaryPolicy(tgt) },
		},
	})
}

func checkWASMTargetModel(tgt ctarget.Target) error {
	return abisuite.CheckWASMTargetModel(tgt)
}

func checkWASMSlotABIMetadata(tgt ctarget.Target) error {
	return abisuite.CheckWASMSlotABIMetadata(tgt)
}

func checkWASMAggregateReturnLayouts(tgt ctarget.Target) error {
	return abisuite.CheckWASMAggregateReturnLayouts(tgt)
}

func checkWASMCallBoundaryValidation(tgt ctarget.Target) error {
	return abisuite.CheckWASMCallBoundaryValidation(tgt)
}

func checkWASMFFIReprCBoundaryPolicy(tgt ctarget.Target) error {
	return abisuite.CheckWASMFFIReprCBoundaryPolicy(tgt)
}

// ---- atomic_suite.go ----

type AtomicStressCheck struct {
	Name  string
	Error string
}

func RunTargetAtomicStressChecks(targetName string) ([]AtomicStressCheck, error) {
	tgt, err := ctarget.Parse(targetName)
	if err != nil {
		return nil, err
	}
	if tgt.Arch != ctarget.ArchX86 && tgt.Arch != ctarget.ArchX64 {
		return nil, fmt.Errorf(
			"atomic stress suite for target %s requires an x86/x64 native target model",
			tgt.Triple,
		)
	}
	prefix := atomicSuiteTargetPrefix(tgt)
	return runAtomicStressChecks([]struct {
		name string
		run  func() error
	}{
		{
			name: prefix + " atomic validation matrix",
			run:  func() error { return checkAtomicValidationMatrix(tgt) },
		},
		{
			name: prefix + " atomic object matrix",
			run:  func() error { return checkAtomicObjectMatrix(tgt) },
		},
		{
			name: prefix + " pointer atomic object width",
			run:  func() error { return checkAtomicPointerObjectWidth(tgt) },
		},
		{
			name: prefix + " atomic concurrency stress oracle",
			run:  func() error { return checkAtomicConcurrencyStressOracle(tgt) },
		},
		{
			name: prefix + " atomic diagnostics",
			run:  func() error { return checkAtomicDiagnostics(tgt) },
		},
	}), nil
}

func runAtomicStressChecks(cases []struct {
	name string
	run  func() error
}) []AtomicStressCheck {
	out := make([]AtomicStressCheck, 0, len(cases))
	for _, tc := range cases {
		check := AtomicStressCheck{Name: tc.name}
		if err := tc.run(); err != nil {
			check.Error = err.Error()
		}
		out = append(out, check)
	}
	return out
}

func atomicSuiteTargetPrefix(tgt ctarget.Target) string {
	if tgt.Arch == ctarget.ArchX86 {
		return "x86"
	}
	if tgt.ABI == ctarget.ABIX32SysV {
		return "x32"
	}
	if tgt.Triple == "windows-x64" || tgt.Triple == "macos-x64" {
		return tgt.Triple
	}
	return "x64"
}

func checkAtomicValidationMatrix(tgt ctarget.Target) error {
	ptrAtomic, err := tgt.AtomicPointerLayout()
	if err != nil {
		return err
	}
	if ptrAtomic.WidthBits != tgt.PointerWidthBits ||
		ptrAtomic.RegisterWidthBits != tgt.RegisterWidthBits ||
		!ptrAtomic.PointerSized {
		return fmt.Errorf(
			"%s pointer atomic layout = %#v, want pointer width %d and register width %d",
			tgt.Triple,
			ptrAtomic,
			tgt.PointerWidthBits,
			tgt.RegisterWidthBits,
		)
	}
	allOrders := []ctarget.MemoryOrder{
		ctarget.MemoryOrderRelaxed,
		ctarget.MemoryOrderAcquire,
		ctarget.MemoryOrderRelease,
		ctarget.MemoryOrderAcqRel,
		ctarget.MemoryOrderSeqCst,
	}
	ops := []ctarget.AtomicOp{
		ctarget.AtomicLoad,
		ctarget.AtomicStore,
		ctarget.AtomicExchange,
		ctarget.AtomicCompareExchange,
		ctarget.AtomicCompareExchangeWeak,
		ctarget.AtomicFetchAdd,
		ctarget.AtomicFetchSub,
		ctarget.AtomicFetchAnd,
		ctarget.AtomicFetchOr,
		ctarget.AtomicFetchXor,
	}
	widths := tgt.AtomicWidthBits()
	if len(widths) == 0 {
		return fmt.Errorf("%s has no declared atomic widths", tgt.Triple)
	}
	for _, width := range widths {
		layout, err := tgt.AtomicLayout(width)
		if err != nil {
			return fmt.Errorf("%s rejected %d-bit atomic layout: %w", tgt.Triple, width, err)
		}
		for _, op := range ops {
			for _, order := range allOrders {
				err := tgt.ValidateAtomic(op, width, layout.AlignBytes, order)
				wantOK := atomicSuiteOrderAllowed(op, order)
				if wantOK && err != nil {
					return fmt.Errorf(
						"%s rejected atomic %s/%d/%s: %w",
						tgt.Triple,
						op,
						width,
						order,
						err,
					)
				}
				if !wantOK && err == nil {
					return fmt.Errorf(
						"%s accepted invalid atomic %s/%d/%s",
						tgt.Triple,
						op,
						width,
						order,
					)
				}
			}
		}
		if err := tgt.ValidateAtomic(
			ctarget.AtomicExchange,
			width,
			layout.AlignBytes-1,
			ctarget.MemoryOrderSeqCst,
		); err == nil {
			return fmt.Errorf("%s accepted misaligned %d-bit atomic exchange", tgt.Triple, width)
		}
	}
	for _, order := range allOrders {
		if err := tgt.ValidateAtomic(ctarget.AtomicFence, 0, 0, order); err != nil {
			return fmt.Errorf("%s rejected atomic fence %s: %w", tgt.Triple, order, err)
		}
	}
	if err := tgt.ValidateAtomic(ctarget.AtomicFence, 0, 0, ctarget.MemoryOrderUnknown); err == nil {
		return fmt.Errorf("%s accepted atomic fence with unknown order", tgt.Triple)
	}
	if tgt.MaxAtomicWidthBits < 64 {
		if _, err := tgt.AtomicLayout(64); err == nil {
			return fmt.Errorf("%s accepted unsupported 64-bit atomic layout", tgt.Triple)
		}
	}
	if _, err := tgt.AtomicLayout(128); err == nil {
		return fmt.Errorf("%s accepted unsupported 128-bit atomic layout", tgt.Triple)
	}
	return nil
}

func atomicSuiteOrderAllowed(op ctarget.AtomicOp, order ctarget.MemoryOrder) bool {
	switch order {
	case ctarget.MemoryOrderRelaxed,
		ctarget.MemoryOrderAcquire,
		ctarget.MemoryOrderRelease,
		ctarget.MemoryOrderAcqRel,
		ctarget.MemoryOrderSeqCst:
	default:
		return false
	}
	switch op {
	case ctarget.AtomicLoad:
		return order == ctarget.MemoryOrderRelaxed || order == ctarget.MemoryOrderAcquire ||
			order == ctarget.MemoryOrderSeqCst
	case ctarget.AtomicStore:
		return order == ctarget.MemoryOrderRelaxed || order == ctarget.MemoryOrderRelease ||
			order == ctarget.MemoryOrderSeqCst
	case ctarget.AtomicExchange,
		ctarget.AtomicCompareExchange,
		ctarget.AtomicCompareExchangeWeak,
		ctarget.AtomicFetchAdd,
		ctarget.AtomicFetchSub,
		ctarget.AtomicFetchAnd,
		ctarget.AtomicFetchOr,
		ctarget.AtomicFetchXor:
		return true
	default:
		return false
	}
}

func checkAtomicConcurrencyStressOracle(tgt ctarget.Target) error {
	iters, err := atomicStressIterations()
	if err != nil {
		return err
	}
	checks := []struct {
		name string
		run  func() error
	}{
		{
			name: "contended CAS loop",
			run:  func() error { return checkAtomicContendedCASLoop(tgt, iters) },
		},
		{
			name: "release/acquire message passing",
			run:  func() error { return checkAtomicMessagePassing(tgt, iters) },
		},
		{name: "seq_cst ordering", run: func() error { return checkAtomicSeqCstOrdering(iters) }},
		{
			name: "ABA stamped pointer",
			run:  func() error { return checkAtomicABAStampedPointer(tgt, iters) },
		},
		{
			name: "false sharing counters",
			run:  func() error { return checkAtomicFalseSharingCounters(tgt, iters) },
		},
		{
			name: "weak CAS spurious retry",
			run:  func() error { return checkAtomicWeakCASSpuriousRetry(tgt, iters) },
		},
		{
			name: "8/16-bit masked CAS loops",
			run:  func() error { return checkAtomicNarrowMaskedCASLoops(iters) },
		},
	}
	for _, check := range checks {
		if err := check.run(); err != nil {
			return fmt.Errorf("%s %s: %w", tgt.Triple, check.name, err)
		}
	}
	return nil
}

func atomicStressIterations() (int, error) {
	raw := strings.TrimSpace(os.Getenv("TETRA_ATOMIC_STRESS_ITERS"))
	if raw == "" {
		return 128, nil
	}
	iters, err := strconv.Atoi(raw)
	if err != nil || iters <= 0 {
		return 0, fmt.Errorf("TETRA_ATOMIC_STRESS_ITERS must be a positive integer, got %q", raw)
	}
	if iters > 100000 {
		return 0, fmt.Errorf(
			"TETRA_ATOMIC_STRESS_ITERS=%d is too high for the compiler-owned stress oracle; use <= 100000",
			iters,
		)
	}
	return iters, nil
}

func checkAtomicContendedCASLoop(tgt ctarget.Target, iters int) error {
	const workers = 4
	if tgt.PointerWidthBits == 32 {
		var counter atomic.Uint32
		var wg sync.WaitGroup
		wg.Add(workers)
		for worker := 0; worker < workers; worker++ {
			go func(worker int) {
				defer wg.Done()
				for i := 0; i < iters; i++ {
					for {
						old := counter.Load()
						if counter.CompareAndSwap(old, old+1) {
							break
						}
						atomicStressYield(i, worker)
					}
					atomicStressYield(i, worker+17)
				}
			}(worker)
		}
		wg.Wait()
		want := uint32(workers * iters)
		if got := counter.Load(); got != want {
			return fmt.Errorf("32-bit pointer CAS counter = %d, want %d", got, want)
		}
		return nil
	}
	var counter atomic.Uint64
	var wg sync.WaitGroup
	wg.Add(workers)
	for worker := 0; worker < workers; worker++ {
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				for {
					old := counter.Load()
					if counter.CompareAndSwap(old, old+1) {
						break
					}
					atomicStressYield(i, worker)
				}
				atomicStressYield(i, worker+17)
			}
		}(worker)
	}
	wg.Wait()
	want := uint64(workers * iters)
	if got := counter.Load(); got != want {
		return fmt.Errorf("64-bit pointer CAS counter = %d, want %d", got, want)
	}
	return nil
}

func checkAtomicMessagePassing(tgt ctarget.Target, iters int) error {
	if tgt.PointerWidthBits == 32 {
		var data atomic.Uint32
		var flag atomic.Uint32
		for i := 0; i < iters; i++ {
			payload := uint32(0x1000 + i)
			data.Store(0)
			flag.Store(0)
			errCh := make(chan error, 1)
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				data.Store(payload)
				atomicStressYield(i, 31)
				flag.Store(1)
			}()
			go func() {
				defer wg.Done()
				for flag.Load() == 0 {
					atomicStressYield(i, 47)
				}
				if got := data.Load(); got != payload {
					errCh <- fmt.Errorf("32-bit payload = %d, want %d", got, payload)
				}
			}()
			wg.Wait()
			select {
			case err := <-errCh:
				return err
			default:
			}
		}
		return nil
	}
	var data atomic.Uint64
	var flag atomic.Uint64
	for i := 0; i < iters; i++ {
		payload := uint64(0x1_0000_0000) + uint64(i)
		data.Store(0)
		flag.Store(0)
		errCh := make(chan error, 1)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			data.Store(payload)
			atomicStressYield(i, 31)
			flag.Store(1)
		}()
		go func() {
			defer wg.Done()
			for flag.Load() == 0 {
				atomicStressYield(i, 47)
			}
			if got := data.Load(); got != payload {
				errCh <- fmt.Errorf("64-bit payload = %d, want %d", got, payload)
			}
		}()
		wg.Wait()
		select {
		case err := <-errCh:
			return err
		default:
		}
	}
	return nil
}

func checkAtomicSeqCstOrdering(iters int) error {
	for i := 0; i < iters; i++ {
		var x atomic.Uint32
		var y atomic.Uint32
		var r1 atomic.Uint32
		var r2 atomic.Uint32
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			x.Store(1)
			atomicStressYield(i, 61)
			r1.Store(y.Load())
		}()
		go func() {
			defer wg.Done()
			<-start
			y.Store(1)
			atomicStressYield(i, 73)
			r2.Store(x.Load())
		}()
		close(start)
		wg.Wait()
		if r1.Load() == 0 && r2.Load() == 0 {
			return fmt.Errorf("seq_cst store/load admitted both-zero result at iteration %d", i)
		}
	}
	return nil
}

func checkAtomicABAStampedPointer(tgt ctarget.Target, iters int) error {
	if tgt.PointerWidthBits == 32 {
		var cell atomic.Uint32
		for i := 0; i < iters; i++ {
			a1 := packABA32(0x1001, 1)
			b2 := packABA32(0x2002, 2)
			a3 := packABA32(0x1001, 3)
			c4 := packABA32(0x3003, 4)
			cell.Store(a1)
			if !cell.CompareAndSwap(a1, b2) || !cell.CompareAndSwap(b2, a3) {
				return fmt.Errorf("32-bit ABA setup failed at iteration %d", i)
			}
			if cell.CompareAndSwap(a1, c4) {
				return fmt.Errorf("32-bit stale ABA CAS succeeded at iteration %d", i)
			}
			if !cell.CompareAndSwap(a3, c4) {
				return fmt.Errorf("32-bit fresh ABA CAS failed at iteration %d", i)
			}
			atomicStressYield(i, 89)
		}
		return nil
	}
	var cell atomic.Uint64
	for i := 0; i < iters; i++ {
		a1 := packABA64(0x10000001, 1)
		b2 := packABA64(0x20000002, 2)
		a3 := packABA64(0x10000001, 3)
		c4 := packABA64(0x30000003, 4)
		cell.Store(a1)
		if !cell.CompareAndSwap(a1, b2) || !cell.CompareAndSwap(b2, a3) {
			return fmt.Errorf("64-bit ABA setup failed at iteration %d", i)
		}
		if cell.CompareAndSwap(a1, c4) {
			return fmt.Errorf("64-bit stale ABA CAS succeeded at iteration %d", i)
		}
		if !cell.CompareAndSwap(a3, c4) {
			return fmt.Errorf("64-bit fresh ABA CAS failed at iteration %d", i)
		}
		atomicStressYield(i, 89)
	}
	return nil
}

func checkAtomicFalseSharingCounters(tgt ctarget.Target, iters int) error {
	const workers = 2
	if tgt.PointerWidthBits == 32 {
		var counters struct {
			left  atomic.Uint32
			right atomic.Uint32
		}
		var wg sync.WaitGroup
		wg.Add(workers)
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				counters.left.Add(1)
				atomicStressYield(i, 101)
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				counters.right.Add(1)
				atomicStressYield(i, 103)
			}
		}()
		wg.Wait()
		if got := counters.left.Load(); got != uint32(iters) {
			return fmt.Errorf("32-bit left false-sharing counter = %d, want %d", got, iters)
		}
		if got := counters.right.Load(); got != uint32(iters) {
			return fmt.Errorf("32-bit right false-sharing counter = %d, want %d", got, iters)
		}
		return nil
	}
	var counters struct {
		left  atomic.Uint64
		right atomic.Uint64
	}
	var wg sync.WaitGroup
	wg.Add(workers)
	go func() {
		defer wg.Done()
		for i := 0; i < iters; i++ {
			counters.left.Add(1)
			atomicStressYield(i, 101)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < iters; i++ {
			counters.right.Add(1)
			atomicStressYield(i, 103)
		}
	}()
	wg.Wait()
	if got := counters.left.Load(); got != uint64(iters) {
		return fmt.Errorf("64-bit left false-sharing counter = %d, want %d", got, iters)
	}
	if got := counters.right.Load(); got != uint64(iters) {
		return fmt.Errorf("64-bit right false-sharing counter = %d, want %d", got, iters)
	}
	return nil
}

func checkAtomicWeakCASSpuriousRetry(tgt ctarget.Target, iters int) error {
	if tgt.PointerWidthBits == 32 {
		var cell atomic.Uint32
		for i := 0; i < iters; i++ {
			old := uint32(i)
			next := old + 1
			cell.Store(old)
			attempts := 0
			for {
				attempts++
				if weakCAS32WithSpuriousFailure(&cell, old, next, attempts) {
					break
				}
				if got := cell.Load(); got != old {
					return fmt.Errorf(
						"32-bit weak CAS changed value after spurious failure: got %d want %d",
						got,
						old,
					)
				}
				atomicStressYield(i, attempts)
			}
			if attempts < 2 {
				return fmt.Errorf("32-bit weak CAS retry did not exercise a spurious failure")
			}
			if got := cell.Load(); got != next {
				return fmt.Errorf("32-bit weak CAS result = %d, want %d", got, next)
			}
		}
		return nil
	}
	var cell atomic.Uint64
	for i := 0; i < iters; i++ {
		old := uint64(i)
		next := old + 1
		cell.Store(old)
		attempts := 0
		for {
			attempts++
			if weakCAS64WithSpuriousFailure(&cell, old, next, attempts) {
				break
			}
			if got := cell.Load(); got != old {
				return fmt.Errorf(
					"64-bit weak CAS changed value after spurious failure: got %d want %d",
					got,
					old,
				)
			}
			atomicStressYield(i, attempts)
		}
		if attempts < 2 {
			return fmt.Errorf("64-bit weak CAS retry did not exercise a spurious failure")
		}
		if got := cell.Load(); got != next {
			return fmt.Errorf("64-bit weak CAS result = %d, want %d", got, next)
		}
	}
	return nil
}

func checkAtomicNarrowMaskedCASLoops(iters int) error {
	var byteCell atomic.Uint32
	var wordCell atomic.Uint32
	for i := 0; i < iters; i++ {
		byteCell.Store(uint32(i) & 0xff)
		wordCell.Store(uint32(i) & 0xffff)
		oldByte, newByte := atomicMaskedFetchXor(&byteCell, 0x5a, 0xff, i, 113)
		if newByte != ((oldByte ^ 0x5a) & 0xff) {
			return fmt.Errorf("u8 masked xor = %#x from old %#x", newByte, oldByte)
		}
		oldWord, newWord := atomicMaskedFetchAdd(&wordCell, 257, 0xffff, i, 127)
		if newWord != ((oldWord + 257) & 0xffff) {
			return fmt.Errorf("u16 masked add = %#x from old %#x", newWord, oldWord)
		}
	}
	return nil
}

func atomicMaskedFetchXor(
	cell *atomic.Uint32,
	operand uint32,
	mask uint32,
	iter int,
	salt int,
) (uint32, uint32) {
	for {
		old := cell.Load() & mask
		next := (old ^ operand) & mask
		if cell.CompareAndSwap(old, next) {
			return old, next
		}
		atomicStressYield(iter, salt)
	}
}

func atomicMaskedFetchAdd(
	cell *atomic.Uint32,
	operand uint32,
	mask uint32,
	iter int,
	salt int,
) (uint32, uint32) {
	for {
		old := cell.Load() & mask
		next := (old + operand) & mask
		if cell.CompareAndSwap(old, next) {
			return old, next
		}
		atomicStressYield(iter, salt)
	}
}

func weakCAS32WithSpuriousFailure(cell *atomic.Uint32, old uint32, next uint32, attempt int) bool {
	if attempt%3 == 1 {
		return false
	}
	return cell.CompareAndSwap(old, next)
}

func weakCAS64WithSpuriousFailure(cell *atomic.Uint64, old uint64, next uint64, attempt int) bool {
	if attempt%3 == 1 {
		return false
	}
	return cell.CompareAndSwap(old, next)
}

func packABA32(ptr uint32, stamp uint32) uint32 {
	return ((stamp & 0xffff) << 16) | (ptr & 0xffff)
}

func packABA64(ptr uint64, stamp uint64) uint64 {
	return ((stamp & 0xffffffff) << 32) | (ptr & 0xffffffff)
}

func atomicStressYield(iter int, salt int) {
	x := uint32(iter*1103515245 + salt*12345 + 0x9e3779b9)
	if x&7 == 0 {
		runtime.Gosched()
	}
}

func checkAtomicObjectMatrix(tgt ctarget.Target) error {
	tmpDir, err := os.MkdirTemp("", "tetra-atomic-suite-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, "atomic_matrix.tetra")
	outPath := filepath.Join(tmpDir, "atomic_matrix.tobj")
	source := atomicMatrixSource
	if tgt.Arch == ctarget.ArchX86 {
		source = atomicMatrixSourceX86
	}
	if err := os.WriteFile(srcPath, []byte(source), 0o644); err != nil {
		return err
	}
	if _, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		tgt.Triple,
		BuildOptions{Emit: EmitLibrary},
	); err != nil {
		return err
	}
	obj, err := ReadObject(outPath)
	if err != nil {
		return err
	}
	if obj.Target != tgt.Triple {
		return fmt.Errorf("target mismatch: got %q want %s", obj.Target, tgt.Triple)
	}
	if !objectHasSymbol(obj, "atomic_matrix") {
		return fmt.Errorf("object missing atomic_matrix symbol: %#v", obj.Symbols)
	}
	required := []struct {
		name  string
		bytes []byte
	}{
		{name: "i64 qword CAS", bytes: []byte{0xF0, 0x4C, 0x0F, 0xB1, 0x07}},
		{name: "i64 qword XADD", bytes: []byte{0xF0, 0x4C, 0x0F, 0xC1, 0x07}},
		{name: "i32 dword CAS", bytes: []byte{0xF0, 0x44, 0x0F, 0xB1, 0x07}},
		{name: "i32 dword XADD", bytes: []byte{0xF0, 0x44, 0x0F, 0xC1, 0x07}},
		{name: "u8 byte exchange", bytes: []byte{0x44, 0x86, 0x07}},
		{name: "seq_cst fence", bytes: []byte{0x0F, 0xAE, 0xF0}},
	}
	if tgt.Arch == ctarget.ArchX86 {
		required = []struct {
			name  string
			bytes []byte
		}{
			{name: "i32 dword CAS", bytes: []byte{0xF0, 0x0F, 0xB1, 0x17}},
			{name: "i32 dword XADD", bytes: []byte{0xF0, 0x0F, 0xC1, 0x0F}},
			{name: "u8 byte exchange", bytes: []byte{0x86, 0x0F}},
			{name: "u16 word exchange", bytes: []byte{0x66, 0x87, 0x0F}},
			{name: "u8 byte fetch-and CAS loop", bytes: []byte{0x20, 0xCA, 0xF0, 0x0F, 0xB0, 0x17}},
			{name: "u8 byte fetch-or CAS loop", bytes: []byte{0x08, 0xCA, 0xF0, 0x0F, 0xB0, 0x17}},
			{name: "u8 byte fetch-xor CAS loop", bytes: []byte{0x30, 0xCA, 0xF0, 0x0F, 0xB0, 0x17}},
			{
				name:  "u16 word fetch-and CAS loop",
				bytes: []byte{0x66, 0x21, 0xCA, 0x66, 0xF0, 0x0F, 0xB1, 0x17},
			},
			{
				name:  "u16 word fetch-or CAS loop",
				bytes: []byte{0x66, 0x09, 0xCA, 0x66, 0xF0, 0x0F, 0xB1, 0x17},
			},
			{
				name:  "u16 word fetch-xor CAS loop",
				bytes: []byte{0x66, 0x31, 0xCA, 0x66, 0xF0, 0x0F, 0xB1, 0x17},
			},
			{name: "seq_cst fence", bytes: []byte{0xF0, 0x83, 0x0C, 0x24, 0x00}},
		}
	}
	for _, want := range required {
		if !bytes.Contains(obj.Code, want.bytes) {
			return fmt.Errorf(
				"missing %s bytes % x in %s atomic object",
				want.name,
				want.bytes,
				tgt.Triple,
			)
		}
	}
	return nil
}

func checkAtomicPointerObjectWidth(tgt ctarget.Target) error {
	tmpDir, err := os.MkdirTemp("", "tetra-atomic-ptr-width-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, "atomic_ptr_width.tetra")
	outPath := filepath.Join(tmpDir, "atomic_ptr_width.tobj")
	if err := os.WriteFile(srcPath, []byte(atomicPointerWidthSource), 0o644); err != nil {
		return err
	}
	if _, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		tgt.Triple,
		BuildOptions{Emit: EmitLibrary, Jobs: 1},
	); err != nil {
		return err
	}
	obj, err := ReadObject(outPath)
	if err != nil {
		return err
	}
	if obj.Target != tgt.Triple {
		return fmt.Errorf("target mismatch: got %q want %s", obj.Target, tgt.Triple)
	}
	if !objectHasSymbol(obj, "atomic_ptr_width") {
		return fmt.Errorf("object missing atomic_ptr_width symbol: %#v", obj.Symbols)
	}
	if tgt.Arch == ctarget.ArchX86 {
		return requireAtomicPointerWidthBytes(obj.Code, tgt.Triple,
			[][]byte{
				{0x87, 0x0F},
				{0xF0, 0x0F, 0xB1, 0x17},
				{0xF0, 0x0F, 0xC1, 0x0F},
				{0xF0, 0x0F, 0xB1, 0x1F},
			},
			nil,
		)
	}
	dwordPatterns := [][]byte{
		{0x45, 0x89, 0xC1},
		{0x44, 0x89, 0xC8},
		{0x44, 0x87, 0x07},
		{0xF0, 0x44, 0x0F, 0xB1, 0x07},
		{0xF0, 0x44, 0x0F, 0xC1, 0x07},
		{0xF0, 0x44, 0x0F, 0xB1, 0x17},
	}
	qwordPatterns := [][]byte{
		{0x4D, 0x89, 0xC1},
		{0x4C, 0x89, 0xC8},
		{0x4C, 0x87, 0x07},
		{0xF0, 0x4C, 0x0F, 0xB1, 0x07},
		{0xF0, 0x4C, 0x0F, 0xC1, 0x07},
		{0xF0, 0x4C, 0x0F, 0xB1, 0x17},
	}
	if tgt.PointerWidthBits == 32 {
		return requireAtomicPointerWidthBytes(obj.Code, tgt.Triple, dwordPatterns, qwordPatterns)
	}
	return requireAtomicPointerWidthBytes(obj.Code, tgt.Triple, qwordPatterns, dwordPatterns)
}

func requireAtomicPointerWidthBytes(
	code []byte,
	target string,
	required [][]byte,
	forbidden [][]byte,
) error {
	for _, pattern := range required {
		if !bytes.Contains(code, pattern) {
			return fmt.Errorf(
				"%s pointer atomic object missing required width bytes % x",
				target,
				pattern,
			)
		}
	}
	for _, pattern := range forbidden {
		if bytes.Contains(code, pattern) {
			return fmt.Errorf(
				"%s pointer atomic object contains forbidden opposite-width bytes % x",
				target,
				pattern,
			)
		}
	}
	return nil
}

const atomicPointerWidthSource = `
func atomic_ptr_width() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(64)
        let p2: ptr = p
        let loaded: ptr = core.atomic_load_ptr_acquire(p, mem)
        var ignored_store: ptr = core.atomic_store_ptr_release(p, loaded, mem)
        let exchanged: ptr = core.atomic_exchange_ptr_seq_cst(p, loaded, mem)
        let cas: ptr = core.atomic_compare_exchange_ptr_acq_rel(p, loaded, exchanged, mem)
        let weak: ptr = core.atomic_compare_exchange_weak_ptr_seq_cst(p, cas, exchanged, mem)
        let add: ptr = core.atomic_fetch_add_ptr_relaxed(p, p2, mem)
        let sub: ptr = core.atomic_fetch_sub_ptr_seq_cst(p, p2, mem)
        let anded: ptr = core.atomic_fetch_and_ptr_acquire(p, p2, mem)
        let ored: ptr = core.atomic_fetch_or_ptr_release(p, p2, mem)
        let xored: ptr = core.atomic_fetch_xor_ptr_acq_rel(p, p2, mem)
        var fence_seq_cst: i32 = core.atomic_fence_seq_cst(mem)
        return 0
    return 0
`

const atomicMatrixSource = `
func atomic_matrix() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(64)
        let p2: ptr = p
        let byte: u8 = 1
        let word: u16 = 2
        let old_byte: u8 = core.atomic_exchange_u8_seq_cst(p, byte, mem)
        let old_word: u16 = core.atomic_exchange_u16_seq_cst(p, word, mem)
        let and_byte: u8 = core.atomic_fetch_and_u8_acquire(p, byte, mem)
        let or_byte: u8 = core.atomic_fetch_or_u8_release(p, byte, mem)
        let xor_byte: u8 = core.atomic_fetch_xor_u8_acq_rel(p, byte, mem)
        let and_word: u16 = core.atomic_fetch_and_u16_acquire(p, word, mem)
        let or_word: u16 = core.atomic_fetch_or_u16_release(p, word, mem)
        let xor_word: u16 = core.atomic_fetch_xor_u16_acq_rel(p, word, mem)
        let loaded: i32 = core.atomic_load_i32_acquire(p, mem)
        var ignored_store: i32 = core.atomic_store_i32_release(p, loaded, mem)
        let exchanged: i32 = core.atomic_exchange_i32_seq_cst(p, loaded, mem)
        let cas: i32 = core.atomic_compare_exchange_i32_acq_rel(p, loaded, exchanged, mem)
        let weak: i32 = core.atomic_compare_exchange_weak_i32_seq_cst(p, cas, exchanged, mem)
        let add: i32 = core.atomic_fetch_add_i32_relaxed(p, 3, mem)
        let sub: i32 = core.atomic_fetch_sub_i32_seq_cst(p, 1, mem)
        let anded: i32 = core.atomic_fetch_and_i32_acquire(p, 7, mem)
        let ored: i32 = core.atomic_fetch_or_i32_release(p, 8, mem)
        let xored: i32 = core.atomic_fetch_xor_i32_acq_rel(p, 9, mem)
        let lp: ptr = core.atomic_load_ptr_acquire(p, mem)
        var ignored_ptr_store: ptr = core.atomic_store_ptr_release(p, lp, mem)
        let xp: ptr = core.atomic_exchange_ptr_seq_cst(p, lp, mem)
        let cas_ptr: ptr = core.atomic_compare_exchange_ptr_acq_rel(p, lp, xp, mem)
        let add_ptr: ptr = core.atomic_fetch_add_ptr_relaxed(p, p2, mem)
        let loaded64: i64 = core.atomic_load_i64_acquire(p, mem)
        var ignored64_store: i64 = core.atomic_store_i64_release(p, loaded64, mem)
        let exchanged64: i64 = core.atomic_exchange_i64_seq_cst(p, loaded64, mem)
        let cas64: i64 = core.atomic_compare_exchange_i64_acq_rel(p, loaded64, exchanged64, mem)
        let weak64: i64 = core.atomic_compare_exchange_weak_i64_seq_cst(p, cas64, exchanged64, mem)
        let add64: i64 = core.atomic_fetch_add_i64_relaxed(p, loaded64, mem)
        var fence_relaxed: i32 = core.atomic_fence_relaxed(mem)
        var fence_acquire: i32 = core.atomic_fence_acquire(mem)
        var fence_release: i32 = core.atomic_fence_release(mem)
        var fence_acq_rel: i32 = core.atomic_fence_acq_rel(mem)
        var fence_seq_cst: i32 = core.atomic_fence_seq_cst(mem)
        return loaded + exchanged + cas + weak + add + sub + anded + ored + xored
    return 0
`

const atomicMatrixSourceX86 = `
func atomic_matrix() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(64)
        let p2: ptr = p
        let byte: u8 = 1
        let word: u16 = 2
        let old_byte: u8 = core.atomic_exchange_u8_seq_cst(p, byte, mem)
        let old_word: u16 = core.atomic_exchange_u16_seq_cst(p, word, mem)
        let and_byte: u8 = core.atomic_fetch_and_u8_acquire(p, byte, mem)
        let or_byte: u8 = core.atomic_fetch_or_u8_release(p, byte, mem)
        let xor_byte: u8 = core.atomic_fetch_xor_u8_acq_rel(p, byte, mem)
        let and_word: u16 = core.atomic_fetch_and_u16_acquire(p, word, mem)
        let or_word: u16 = core.atomic_fetch_or_u16_release(p, word, mem)
        let xor_word: u16 = core.atomic_fetch_xor_u16_acq_rel(p, word, mem)
        let loaded: i32 = core.atomic_load_i32_acquire(p, mem)
        var ignored_store: i32 = core.atomic_store_i32_release(p, loaded, mem)
        let exchanged: i32 = core.atomic_exchange_i32_seq_cst(p, loaded, mem)
        let cas: i32 = core.atomic_compare_exchange_i32_acq_rel(p, loaded, exchanged, mem)
        let weak: i32 = core.atomic_compare_exchange_weak_i32_seq_cst(p, cas, exchanged, mem)
        let add: i32 = core.atomic_fetch_add_i32_relaxed(p, 3, mem)
        let sub: i32 = core.atomic_fetch_sub_i32_seq_cst(p, 1, mem)
        let anded: i32 = core.atomic_fetch_and_i32_acquire(p, 7, mem)
        let ored: i32 = core.atomic_fetch_or_i32_release(p, 8, mem)
        let xored: i32 = core.atomic_fetch_xor_i32_acq_rel(p, 9, mem)
        let lp: ptr = core.atomic_load_ptr_acquire(p, mem)
        var ignored_ptr_store: ptr = core.atomic_store_ptr_release(p, lp, mem)
        let xp: ptr = core.atomic_exchange_ptr_seq_cst(p, lp, mem)
        let cas_ptr: ptr = core.atomic_compare_exchange_ptr_acq_rel(p, lp, xp, mem)
        let add_ptr: ptr = core.atomic_fetch_add_ptr_relaxed(p, p2, mem)
        var fence_relaxed: i32 = core.atomic_fence_relaxed(mem)
        var fence_acquire: i32 = core.atomic_fence_acquire(mem)
        var fence_release: i32 = core.atomic_fence_release(mem)
        var fence_acq_rel: i32 = core.atomic_fence_acq_rel(mem)
        var fence_seq_cst: i32 = core.atomic_fence_seq_cst(mem)
        return loaded + exchanged + cas + weak + add + sub + anded + ored + xored
    return 0
`

func checkAtomicDiagnostics(tgt ctarget.Target) error {
	tests := []struct {
		name string
		call string
		want string
	}{
		{
			name: "load release",
			call: "core.atomic_load_i32_release(p, mem)",
			want: "atomic load does not support memory order release",
		},
		{
			name: "store acquire",
			call: "core.atomic_store_i32_acquire(p, 1, mem)",
			want: "atomic store does not support memory order acquire",
		},
		{
			name: "unknown order",
			call: "core.atomic_fetch_add_i32_consume(p, 1, mem)",
			want: "unsupported atomic memory order 'consume'",
		},
		{
			name: "unknown op",
			call: "core.atomic_nand_i32_relaxed(p, 1, mem)",
			want: "unsupported atomic operation 'nand'",
		},
	}
	for _, tt := range tests {
		src := `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(4)
        return ` + tt.call + `
    return 0
`
		prog, err := Parse([]byte(src))
		if err != nil {
			return fmt.Errorf("%s parse: %w", tt.name, err)
		}
		_, err = Check(prog)
		if err == nil {
			return fmt.Errorf("%s accepted invalid atomic builtin %s", tt.name, tt.call)
		}
		if !strings.Contains(err.Error(), tt.want) {
			return fmt.Errorf("%s diagnostic = %q, want %q", tt.name, err.Error(), tt.want)
		}
	}
	if tgt.MaxAtomicWidthBits < 64 {
		if err := checkAtomicUnsupportedWidthDiagnostic(tgt); err != nil {
			return err
		}
	}
	return nil
}

func checkAtomicUnsupportedWidthDiagnostic(tgt ctarget.Target) error {
	tmpDir, err := os.MkdirTemp("", "tetra-atomic-diagnostics-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	srcPath := filepath.Join(tmpDir, "atomic_i64_unsupported.tetra")
	outPath := filepath.Join(tmpDir, "atomic_i64_unsupported.tobj")
	src := `
func atomic_i64_unsupported() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let loaded: i64 = core.atomic_load_i64_acquire(p, mem)
        return 0
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	_, err = BuildFileWithStatsOpt(
		srcPath,
		outPath,
		tgt.Triple,
		BuildOptions{Emit: EmitLibrary, Jobs: 1},
	)
	if err == nil {
		return fmt.Errorf("%s accepted unsupported 64-bit atomic source", tgt.Triple)
	}
	diag := DiagnosticFromError(err)
	if diag.Code != DiagnosticCodeTargetRuntime || diag.Severity != "error" {
		return fmt.Errorf("%s unsupported-width diagnostic identity = %#v", tgt.Triple, diag)
	}
	for _, want := range []string{
		tgt.Triple,
		"atomic load",
		"64-bit",
		"unsupported atomic width 64 bits",
	} {
		if !strings.Contains(diag.Message, want) {
			return fmt.Errorf(
				"%s unsupported-width diagnostic = %q, want %q",
				tgt.Triple,
				diag.Message,
				want,
			)
		}
	}
	if _, statErr := os.Stat(outPath); !os.IsNotExist(statErr) {
		return fmt.Errorf(
			"%s unsupported-width rejection wrote object %s, stat error = %v",
			tgt.Triple,
			outPath,
			statErr,
		)
	}
	return nil
}

func objectHasSymbol(obj *Object, name string) bool {
	if obj == nil {
		return false
	}
	for _, sym := range obj.Symbols {
		if strings.EqualFold(sym.Name, name) || sym.Name == name {
			return true
		}
	}
	return false
}

// ---- atomic_target.go ----

func validateTargetAtomicIR(funcs []IRFunc, tgt ctarget.Target) error {
	for _, fn := range funcs {
		for _, instr := range fn.Instrs {
			info, ok := atomicIRTargetInfo(instr.Kind, tgt)
			if !ok {
				continue
			}
			if info.op == ctarget.AtomicFence {
				if err := tgt.ValidateAtomic(ctarget.AtomicFence, 0, 0, info.order); err != nil {
					return targetAtomicDiagnostic(instr.Pos, tgt.Triple, info.op, 0, err)
				}
				continue
			}
			if _, err := tgt.AtomicLayout(info.widthBits); err != nil {
				return targetAtomicDiagnostic(instr.Pos, tgt.Triple, info.op, info.widthBits, err)
			}
		}
	}
	return nil
}

type atomicIRInfo struct {
	op        ctarget.AtomicOp
	widthBits int
	order     ctarget.MemoryOrder
}

func atomicIRTargetInfo(kind ir.IRInstrKind, tgt ctarget.Target) (atomicIRInfo, bool) {
	ptrWidth := tgt.PointerWidthBits
	switch kind {
	case ir.IRAtomicLoadPtr:
		return atomicIRInfo{op: ctarget.AtomicLoad, widthBits: ptrWidth}, true
	case ir.IRAtomicStorePtr:
		return atomicIRInfo{op: ctarget.AtomicStore, widthBits: ptrWidth}, true
	case ir.IRAtomicExchangePtr:
		return atomicIRInfo{op: ctarget.AtomicExchange, widthBits: ptrWidth}, true
	case ir.IRAtomicFetchAddPtr:
		return atomicIRInfo{op: ctarget.AtomicFetchAdd, widthBits: ptrWidth}, true
	case ir.IRAtomicFetchSubPtr:
		return atomicIRInfo{op: ctarget.AtomicFetchSub, widthBits: ptrWidth}, true
	case ir.IRAtomicFetchAndPtr:
		return atomicIRInfo{op: ctarget.AtomicFetchAnd, widthBits: ptrWidth}, true
	case ir.IRAtomicFetchOrPtr:
		return atomicIRInfo{op: ctarget.AtomicFetchOr, widthBits: ptrWidth}, true
	case ir.IRAtomicFetchXorPtr:
		return atomicIRInfo{op: ctarget.AtomicFetchXor, widthBits: ptrWidth}, true
	case ir.IRAtomicCompareExchangePtr:
		return atomicIRInfo{op: ctarget.AtomicCompareExchange, widthBits: ptrWidth}, true
	case ir.IRAtomicFenceSeqCst:
		return atomicIRInfo{op: ctarget.AtomicFence, order: ctarget.MemoryOrderSeqCst}, true
	case ir.IRAtomicFenceRelaxed:
		return atomicIRInfo{op: ctarget.AtomicFence, order: ctarget.MemoryOrderRelaxed}, true
	case ir.IRAtomicFenceAcquire:
		return atomicIRInfo{op: ctarget.AtomicFence, order: ctarget.MemoryOrderAcquire}, true
	case ir.IRAtomicFenceRelease:
		return atomicIRInfo{op: ctarget.AtomicFence, order: ctarget.MemoryOrderRelease}, true
	case ir.IRAtomicFenceAcqRel:
		return atomicIRInfo{op: ctarget.AtomicFence, order: ctarget.MemoryOrderAcqRel}, true
	case ir.IRAtomicLoadI8:
		return atomicIRInfo{op: ctarget.AtomicLoad, widthBits: 8}, true
	case ir.IRAtomicStoreI8:
		return atomicIRInfo{op: ctarget.AtomicStore, widthBits: 8}, true
	case ir.IRAtomicExchangeI8:
		return atomicIRInfo{op: ctarget.AtomicExchange, widthBits: 8}, true
	case ir.IRAtomicCompareExchangeI8:
		return atomicIRInfo{op: ctarget.AtomicCompareExchange, widthBits: 8}, true
	case ir.IRAtomicFetchAddI8:
		return atomicIRInfo{op: ctarget.AtomicFetchAdd, widthBits: 8}, true
	case ir.IRAtomicFetchSubI8:
		return atomicIRInfo{op: ctarget.AtomicFetchSub, widthBits: 8}, true
	case ir.IRAtomicFetchAndI8:
		return atomicIRInfo{op: ctarget.AtomicFetchAnd, widthBits: 8}, true
	case ir.IRAtomicFetchOrI8:
		return atomicIRInfo{op: ctarget.AtomicFetchOr, widthBits: 8}, true
	case ir.IRAtomicFetchXorI8:
		return atomicIRInfo{op: ctarget.AtomicFetchXor, widthBits: 8}, true
	case ir.IRAtomicLoadI16:
		return atomicIRInfo{op: ctarget.AtomicLoad, widthBits: 16}, true
	case ir.IRAtomicStoreI16:
		return atomicIRInfo{op: ctarget.AtomicStore, widthBits: 16}, true
	case ir.IRAtomicExchangeI16:
		return atomicIRInfo{op: ctarget.AtomicExchange, widthBits: 16}, true
	case ir.IRAtomicCompareExchangeI16:
		return atomicIRInfo{op: ctarget.AtomicCompareExchange, widthBits: 16}, true
	case ir.IRAtomicFetchAddI16:
		return atomicIRInfo{op: ctarget.AtomicFetchAdd, widthBits: 16}, true
	case ir.IRAtomicFetchSubI16:
		return atomicIRInfo{op: ctarget.AtomicFetchSub, widthBits: 16}, true
	case ir.IRAtomicFetchAndI16:
		return atomicIRInfo{op: ctarget.AtomicFetchAnd, widthBits: 16}, true
	case ir.IRAtomicFetchOrI16:
		return atomicIRInfo{op: ctarget.AtomicFetchOr, widthBits: 16}, true
	case ir.IRAtomicFetchXorI16:
		return atomicIRInfo{op: ctarget.AtomicFetchXor, widthBits: 16}, true
	case ir.IRAtomicLoadI32:
		return atomicIRInfo{op: ctarget.AtomicLoad, widthBits: 32}, true
	case ir.IRAtomicStoreI32:
		return atomicIRInfo{op: ctarget.AtomicStore, widthBits: 32}, true
	case ir.IRAtomicExchangeI32:
		return atomicIRInfo{op: ctarget.AtomicExchange, widthBits: 32}, true
	case ir.IRAtomicCompareExchangeI32:
		return atomicIRInfo{op: ctarget.AtomicCompareExchange, widthBits: 32}, true
	case ir.IRAtomicFetchAddI32:
		return atomicIRInfo{op: ctarget.AtomicFetchAdd, widthBits: 32}, true
	case ir.IRAtomicFetchSubI32:
		return atomicIRInfo{op: ctarget.AtomicFetchSub, widthBits: 32}, true
	case ir.IRAtomicFetchAndI32:
		return atomicIRInfo{op: ctarget.AtomicFetchAnd, widthBits: 32}, true
	case ir.IRAtomicFetchOrI32:
		return atomicIRInfo{op: ctarget.AtomicFetchOr, widthBits: 32}, true
	case ir.IRAtomicFetchXorI32:
		return atomicIRInfo{op: ctarget.AtomicFetchXor, widthBits: 32}, true
	case ir.IRAtomicLoadI64:
		return atomicIRInfo{op: ctarget.AtomicLoad, widthBits: 64}, true
	case ir.IRAtomicStoreI64:
		return atomicIRInfo{op: ctarget.AtomicStore, widthBits: 64}, true
	case ir.IRAtomicExchangeI64:
		return atomicIRInfo{op: ctarget.AtomicExchange, widthBits: 64}, true
	case ir.IRAtomicCompareExchangeI64:
		return atomicIRInfo{op: ctarget.AtomicCompareExchange, widthBits: 64}, true
	case ir.IRAtomicFetchAddI64:
		return atomicIRInfo{op: ctarget.AtomicFetchAdd, widthBits: 64}, true
	case ir.IRAtomicFetchSubI64:
		return atomicIRInfo{op: ctarget.AtomicFetchSub, widthBits: 64}, true
	case ir.IRAtomicFetchAndI64:
		return atomicIRInfo{op: ctarget.AtomicFetchAnd, widthBits: 64}, true
	case ir.IRAtomicFetchOrI64:
		return atomicIRInfo{op: ctarget.AtomicFetchOr, widthBits: 64}, true
	case ir.IRAtomicFetchXorI64:
		return atomicIRInfo{op: ctarget.AtomicFetchXor, widthBits: 64}, true
	default:
		return atomicIRInfo{}, false
	}
}

func targetAtomicDiagnostic(
	pos frontend.Position,
	target string,
	op ctarget.AtomicOp,
	widthBits int,
	cause error,
) error {
	width := "pointer-sized"
	if widthBits > 0 {
		width = fmt.Sprintf("%d-bit", widthBits)
	}
	hint := fmt.Sprintf(
		("Use an atomic width supported by %s, or build this source " +
			"for a target whose atomic model supports %s operations."),
		target,
		width,
	)
	if target == "linux-x86" {
		hint = ("Use 8/16/32-bit or pointer atomics on linux-x86, or build " +
			"for linux-x64/linux-x32 when 64-bit lock-free atomics are " +
			"required.")
	}
	return &frontend.DiagnosticError{Info: frontend.Diagnostic{
		Code: DiagnosticCodeTargetRuntime,
		Message: fmt.Sprintf(
			"%s atomic %s requires unsupported %s width: %v",
			target,
			op,
			width,
			cause,
		),
		File:     pos.File,
		Line:     pos.Line,
		Column:   pos.Col,
		Severity: "error",
		Hint:     hint,
	}}
}

// ---- compatibility_stability_v1.go ----

const (
	compatibilityStabilityV1Schema    = "tetra.compatibility.stability.v1"
	compatibilityStabilityV1ScopeP242 = "p24.2_compatibility_stability"

	p24CompatibilityDiagnosticWitnessID  = "stable_diagnostic_codes"
	p24CompatibilitySchemaWitnessID      = "versioned_report_schemas"
	p24CompatibilityManifestWitnessID    = "manifest_compatibility_checks"
	p24CompatibilityMigrationWitnessID   = "breaking_change_migration_guide"
	p24CompatibilityDeprecationWitnessID = "deprecation_policy"
	p24CompatibilityArtifactsWitnessID   = "compatibility_stability_artifacts"
)

type CompatibilityStabilityV1ID string
type compatibilityStabilityID = CompatibilityStabilityV1ID

const (
	CompatibilityStableDiagnosticCodes  compatibilityStabilityID = "stable_diagnostic_codes"
	CompatibilityVersionedReportSchemas compatibilityStabilityID = "versioned_report_schemas"
	CompatibilityManifestChecks                                  = compatibilityStabilityID(
		"manifest_compatibility_checks",
	)
	CompatibilityBreakingChangeMigrationGuide = compatibilityStabilityID(
		"breaking_change_migration_guide",
	)
	CompatibilityDeprecationPolicy compatibilityStabilityID = "deprecation_policy"
)

type CompatibilityStabilityV1Report struct {
	SchemaVersion string                            `json:"schema_version"`
	Scope         string                            `json:"scope"`
	Rows          []CompatibilityStabilityV1Row     `json:"rows"`
	Witnesses     []CompatibilityStabilityV1Witness `json:"witnesses"`
	Artifacts     []CompatibilityStabilityArtifact  `json:"artifacts"`
	NonClaims     []string                          `json:"non_claims"`

	StableDiagnosticCodesReviewed       bool `json:"stable_diagnostic_codes_reviewed"`
	VersionedReportSchemasReviewed      bool `json:"versioned_report_schemas_reviewed"`
	ManifestCompatibilityChecksReviewed bool `json:"manifest_compatibility_checks_reviewed"`
	BreakingChangeMigrationGuidePresent bool `json:"breaking_change_migration_guide_present"`
	DeprecationPolicyPresent            bool `json:"deprecation_policy_present"`

	FullBackwardCompatibilityClaimed       bool `json:"full_backward_compatibility_claimed"`
	FrozenDiagnosticMessagesClaimed        bool `json:"frozen_diagnostic_messages_claimed"`
	AutomaticMigrationClaimed              bool `json:"automatic_migration_claimed"`
	ManifestABIStabilityClaimed            bool `json:"manifest_abi_stability_claimed"`
	BreakingChangesWithoutMigrationClaimed bool `json:"breaking_changes_without_migration_claimed"`
	RemovalWithoutDeprecationClaimed       bool `json:"removal_without_deprecation_claimed"`
	RuntimeBehaviorChanged                 bool `json:"runtime_behavior_changed"`
	SafeSemanticsChanged                   bool `json:"safe_semantics_changed"`
	PerformanceClaimed                     bool `json:"performance_claimed"`
}

type CompatibilityStabilityV1Row struct {
	ID         CompatibilityStabilityV1ID `json:"id"`
	Name       string                     `json:"name"`
	Status     string                     `json:"status"`
	Evidence   []string                   `json:"evidence"`
	Tests      []string                   `json:"tests"`
	Boundaries []string                   `json:"boundaries"`
	WitnessIDs []string                   `json:"witness_ids"`
}

type CompatibilityStabilityArtifact struct {
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Present bool   `json:"present"`
}

type CompatibilityStabilityV1Witness struct {
	ID    string   `json:"id"`
	Kind  string   `json:"kind"`
	Paths []string `json:"paths,omitempty"`

	DiagnosticCodes               []string `json:"diagnostic_codes,omitempty"`
	DiagnosticRegistryCount       int      `json:"diagnostic_registry_count,omitempty"`
	DiagnosticCodesValid          bool     `json:"diagnostic_codes_valid,omitempty"`
	DiagnosticJSONValidatorStrict bool     `json:"diagnostic_json_validator_strict,omitempty"`
	DiagnosticReleaseDocsPresent  bool     `json:"diagnostic_release_docs_present,omitempty"`
	StableDiagnosticCodesReviewed bool     `json:"stable_diagnostic_codes_reviewed,omitempty"`

	SchemaIDs                      []string `json:"schema_ids,omitempty"`
	VersionedSchemaCount           int      `json:"versioned_schema_count,omitempty"`
	ReportSchemasStrict            bool     `json:"report_schemas_strict,omitempty"`
	VersionedReportSchemasReviewed bool     `json:"versioned_report_schemas_reviewed,omitempty"`

	ManifestCompilerVersion         string `json:"manifest_compiler_version,omitempty"`
	ManifestTargetCount             int    `json:"manifest_target_count,omitempty"`
	ManifestFeatureCount            int    `json:"manifest_feature_count,omitempty"`
	ManifestRuntimeABIPresent       bool   `json:"manifest_runtime_abi_present,omitempty"`
	ManifestValidatorStrict         bool   `json:"manifest_validator_strict,omitempty"`
	ManifestFeatureRegistryLinked   bool   `json:"manifest_feature_registry_linked,omitempty"`
	ManifestRuntimeABIChecksPresent bool   `json:"manifest_runtime_abi_checks_present,omitempty"`

	ManifestCompatibilityChecksReviewed bool `json:"manifest_compatibility_checks_reviewed,omitempty"`

	MigrationGuidePresent               bool `json:"migration_guide_present,omitempty"`
	APIBreakingReviewPresent            bool `json:"api_breaking_review_present,omitempty"`
	PatchLineRulePresent                bool `json:"patch_line_rule_present,omitempty"`
	BreakingChangeMigrationGuidePresent bool `json:"breaking_change_migration_guide_present,omitempty"`

	DeprecationPolicyPresent   bool `json:"deprecation_policy_present,omitempty"`
	ReplacementPathRequired    bool `json:"replacement_path_required,omitempty"`
	RemovalDelayRequired       bool `json:"removal_delay_required,omitempty"`
	StdlibMajorLineRulePresent bool `json:"stdlib_major_line_rule_present,omitempty"`

	CompatibilityAuditArtifactPresent  bool `json:"compatibility_audit_artifact_present,omitempty"`
	CompatibilityDesignArtifactPresent bool `json:"compatibility_design_artifact_present,omitempty"`
	MigrationGuideArtifactPresent      bool `json:"migration_guide_artifact_present,omitempty"`
	DeprecationPolicyArtifactPresent   bool `json:"deprecation_policy_artifact_present,omitempty"`
}

func BuildP24CompatibilityStabilityV1Report() (CompatibilityStabilityV1Report, error) {
	diagnosticWitness := buildP24CompatibilityDiagnosticWitness()
	schemaWitness := buildP24CompatibilitySchemaWitness()
	manifestWitness := buildP24CompatibilityManifestWitness()
	migrationWitness := buildP24CompatibilityMigrationWitness()
	deprecationWitness := buildP24CompatibilityDeprecationWitness()
	artifacts := p24CompatibilityStabilityArtifacts()
	artifactWitness := buildP24CompatibilityArtifactsWitness(artifacts)

	report := CompatibilityStabilityV1Report{
		SchemaVersion: compatibilityStabilityV1Schema,
		Scope:         compatibilityStabilityV1ScopeP242,
		Witnesses: []CompatibilityStabilityV1Witness{
			diagnosticWitness,
			schemaWitness,
			manifestWitness,
			migrationWitness,
			deprecationWitness,
			artifactWitness,
		},
		Artifacts: artifacts,
		Rows: []CompatibilityStabilityV1Row{
			p24CompatibilityStabilityRow(
				CompatibilityStableDiagnosticCodes,
				"Stable diagnostic codes",
				"reviewed_current_diagnostic_surface",
				[]string{
					("DiagnosticCodeRegistry records the public diagnostic code " +
						"set, including parser/frontend TETRA0001, positioned " +
						"semantic/compiler TETRA2001, safety code families, " +
						"lowering/IR verifier codes, target runtime diagnostics, and " +
						"formatter codes."),
					("tools/cmd/validate-diagnostic validates the " +
						"tetra.release.v0_2_0.diagnostic-json.v1 JSON shape with " +
						"strict unknown-field rejection while release notes document " +
						"TETRA0001 and TETRA2001 compatibility."),
				},
				[]string{
					"go test ./compiler -run 'P24CompatibilityStability' -count=1",
					"go test ./compiler/tests/frontend -run 'DiagnosticCodeRegistry|Diagnostic' -count=1",
					"go test ./tools/cmd/validate-diagnostic -count=1",
				},
				[]string{
					"diagnostic codes and JSON object shape are stable evidence for the current release line",
					("diagnostic messages are not frozen and may improve while " +
						"retaining stable codes/severity shape where promised"),
				},
				[]string{p24CompatibilityDiagnosticWitnessID},
			),
			p24CompatibilityStabilityRow(
				CompatibilityVersionedReportSchemas,
				"Versioned report schemas",
				"reviewed_schema_version_surface",
				[]string{
					("Current evidence reports carry explicit versioned schemas " +
						"such as tetra.translation.validation.v2, " +
						"tetra.fuzz.property.differential.v1, tetra.formal_core.v1, " +
						"tetra.self_hosting.gate.v1, tetra.security.review_gate.v1, " +
						"tetra.runtime.hardening.v1, and " +
						"tetra.compatibility.stability.v1."),
					("Compiler and tool validators reject unexpected " +
						"schema_version or schema values for report families instead " +
						"of silently accepting drift."),
				},
				[]string{
					"go test ./compiler -run 'P24CompatibilityStability' -count=1",
					"go test ./tools/cmd/validate-manifest ./tools/cmd/validate-diagnostic -count=1",
				},
				[]string{
					"versioned schema evidence does not promise automatic migration for every old report",
					"private or experimental artifacts remain governed by their local validators and release docs",
				},
				[]string{p24CompatibilitySchemaWitnessID},
			),
			p24CompatibilityStabilityRow(
				CompatibilityManifestChecks,
				"Manifest compatibility checks",
				"reviewed_manifest_validator_surface",
				[]string{
					("tools/cmd/validate-manifest validates " +
						"tetra.release.v0_4_0.manifest-json.v1 with strict JSON " +
						"decoding, target ordering/coverage, builtin metadata, " +
						"FeatureRegistry entries, and runtime ABI symbol coverage."),
					("compiler.GetManifest builds the generated manifest from " +
						"Version, formats, buildable targets, builtins, runtime ABI, " +
						"and FeatureRegistry data for the same branch state."),
				},
				[]string{
					"go test ./tools/cmd/validate-manifest -count=1",
					"go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json",
					"go test ./compiler/tests/semantics -run 'FeatureRegistry' -count=1",
				},
				[]string{
					("manifest compatibility checks are current-branch validator " +
						"evidence, not a future runtime ABI stability promise"),
					("manifest changes still require regenerated " +
						"docs/generated/manifest.json and matching release notes or " +
						"migration guidance"),
				},
				[]string{p24CompatibilityManifestWitnessID},
			),
			p24CompatibilityStabilityRow(
				CompatibilityBreakingChangeMigrationGuide,
				"Breaking-change migration guide",
				"documented_policy_present",
				[]string{
					("docs/release/policy/breaking-change-migration-guide.md " +
						"defines triage, migration steps, diagnostic/report/manifest " +
						"handling, and release-note requirements for incompatible " +
						"changes."),
					("docs/spec/policy/api_diff_policy.md marks removed/changed " +
						"API entries as breaking_requires_review and keeps release " +
						"gate mode at --enforce no-change until versioned API " +
						"compatibility rules exist."),
				},
				[]string{
					"go test ./compiler -run 'P24CompatibilityStability' -count=1",
					"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
				},
				[]string{
					"the migration guide is a release process artifact, not automatic source rewrite tooling",
					"security exceptions still require documented compatibility impact and mitigation",
				},
				[]string{p24CompatibilityMigrationWitnessID},
			),
			p24CompatibilityStabilityRow(
				CompatibilityDeprecationPolicy,
				"Deprecation policy",
				"documented_policy_present",
				[]string{
					("docs/release/policy/deprecation_policy.md and " +
						"docs/release/v1_0/v1_0_x_maintenance_policy.md require a " +
						"Deprecation Policy with a replacement path plus diagnostics " +
						"or documentation."),
					("Stable lib.core breaking changes wait for a later major " +
						"release line; removals wait for a later minor or major line " +
						"unless a security fix requires a documented exception."),
				},
				[]string{
					"go test ./compiler -run 'P24CompatibilityStability' -count=1",
					"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
				},
				[]string{
					"deprecation policy does not authorize removals without replacement and migration notes",
					("experimental surfaces can remain less stable only where " +
						"docs explicitly mark them experimental"),
				},
				[]string{p24CompatibilityDeprecationWitnessID},
			),
		},
		NonClaims: []string{
			"full backward compatibility for all future versions is not claimed",
			"diagnostic messages are not frozen",
			"automatic migration for every breaking change is not claimed",
			"manifest/runtime ABI stability beyond current validated evidence is not claimed",
			"runtime behavior does not change",
			"safe-program semantics do not change",
			"no performance claim is made",
		},
		StableDiagnosticCodesReviewed:       diagnosticWitness.StableDiagnosticCodesReviewed,
		VersionedReportSchemasReviewed:      schemaWitness.VersionedReportSchemasReviewed,
		ManifestCompatibilityChecksReviewed: manifestWitness.ManifestCompatibilityChecksReviewed,
		BreakingChangeMigrationGuidePresent: migrationWitness.BreakingChangeMigrationGuidePresent,
		DeprecationPolicyPresent:            deprecationWitness.DeprecationPolicyPresent,
	}
	if err := ValidateP24CompatibilityStabilityV1Report(report); err != nil {
		return CompatibilityStabilityV1Report{}, err
	}
	return report, nil
}

func ValidateP24CompatibilityStabilityV1Report(report CompatibilityStabilityV1Report) error {
	if report.SchemaVersion != compatibilityStabilityV1Schema {
		return fmt.Errorf("compatibility/stability v1: schema_version is %q", report.SchemaVersion)
	}
	if report.Scope != compatibilityStabilityV1ScopeP242 {
		return fmt.Errorf("compatibility/stability v1: scope is %q", report.Scope)
	}
	if report.FullBackwardCompatibilityClaimed {
		return fmt.Errorf(
			"compatibility/stability v1: full backward compatibility claim is forbidden",
		)
	}
	if report.FrozenDiagnosticMessagesClaimed {
		return fmt.Errorf(
			"compatibility/stability v1: frozen diagnostic messages claim is forbidden",
		)
	}
	if report.AutomaticMigrationClaimed {
		return fmt.Errorf("compatibility/stability v1: automatic migration claim is forbidden")
	}
	if report.ManifestABIStabilityClaimed {
		return fmt.Errorf(
			"compatibility/stability v1: manifest/runtime ABI stability claim is forbidden",
		)
	}
	if report.BreakingChangesWithoutMigrationClaimed {
		return fmt.Errorf(
			"compatibility/stability v1: breaking change without migration guide claim is forbidden",
		)
	}
	if report.RemovalWithoutDeprecationClaimed {
		return fmt.Errorf(
			"compatibility/stability v1: removal without deprecation claim is forbidden",
		)
	}
	if report.RuntimeBehaviorChanged {
		return fmt.Errorf("compatibility/stability v1: runtime behavior change claim is forbidden")
	}
	if report.SafeSemanticsChanged {
		return fmt.Errorf("compatibility/stability v1: safe semantics change claim is forbidden")
	}
	if report.PerformanceClaimed {
		return fmt.Errorf("compatibility/stability v1: performance claim is forbidden")
	}
	if !report.StableDiagnosticCodesReviewed {
		return fmt.Errorf("compatibility/stability v1: stable diagnostic code review missing")
	}
	if !report.VersionedReportSchemasReviewed {
		return fmt.Errorf("compatibility/stability v1: versioned report schema review missing")
	}
	if !report.ManifestCompatibilityChecksReviewed {
		return fmt.Errorf("compatibility/stability v1: manifest compatibility checks missing")
	}
	if !report.BreakingChangeMigrationGuidePresent {
		return fmt.Errorf("compatibility/stability v1: breaking-change migration guide missing")
	}
	if !report.DeprecationPolicyPresent {
		return fmt.Errorf("compatibility/stability v1: deprecation policy missing")
	}
	for _, want := range []string{
		"full backward compatibility for all future versions is not claimed",
		"diagnostic messages are not frozen",
		"automatic migration for every breaking change is not claimed",
		"manifest/runtime ABI stability beyond current validated evidence is not claimed",
		"runtime behavior does not change",
		"safe-program semantics do not change",
		"no performance claim is made",
	} {
		if !p24CompatibilityStabilityHasString(report.NonClaims, want) {
			return fmt.Errorf("compatibility/stability v1: missing non-claim %q", want)
		}
	}
	if err := p24CompatibilityStabilityValidateArtifacts(report); err != nil {
		return err
	}
	return p24CompatibilityStabilityValidateRowsAndWitnesses(report.Rows, report.Witnesses)
}

func buildP24CompatibilityDiagnosticWitness() CompatibilityStabilityV1Witness {
	paths := []string{
		"compiler/compiler_facade.go",
		"tools/cmd/validate-diagnostic/main.go",
		"tools/cmd/validate-diagnostic/main_test.go",
		"docs/roadmaps/early/roadmap_0_6_1_to_0_6_3.md",
		"docs/release-notes/archive/v0_6.md",
	}
	registry := DiagnosticCodeRegistry()
	codes := make([]string, 0, len(registry))
	validCodes := true
	for code, info := range registry {
		codes = append(codes, code)
		if strings.TrimSpace(code) == "" || code != strings.TrimSpace(code) ||
			strings.TrimSpace(info.Severity) == "" ||
			strings.TrimSpace(info.Surface) == "" {
			validCodes = false
		}
	}
	sort.Strings(codes)
	required := []string{
		DiagnosticCodeParse,
		DiagnosticCodeSemantic,
		DiagnosticCodeSafetyOwnership,
		DiagnosticCodeSafetyLifetime,
		DiagnosticCodeSafetyEffect,
		DiagnosticCodeSafetyPrivacy,
		DiagnosticCodeSafetyBudget,
		DiagnosticCodeIRVerifier,
		DiagnosticCodeLowerUnsupported,
		DiagnosticCodeTargetRuntime,
		DiagnosticCodeFormatter,
		DiagnosticCodeFormatterCheck,
	}
	for _, code := range required {
		if _, ok := registry[code]; !ok {
			validCodes = false
		}
	}
	diagnosticStrictDecode := p24CompatibilityStabilityFileContains(
		"tools/cmd/validate-diagnostic/main.go",
		"DisallowUnknownFields",
	) ||
		(p24CompatibilityStabilityFileContains(
			"tools/cmd/validate-diagnostic/main.go",
			"reportdecode.DecodeStrict",
		) &&
			p24CompatibilityStabilityFileContains(
				"tools/internal/reportdecode/reportdecode.go",
				"DisallowUnknownFields",
			))
	strictValidator := diagnosticStrictDecode &&
		p24CompatibilityStabilityFileContains(
			"tools/cmd/validate-diagnostic/main.go",
			"tetra.release.v0_2_0.diagnostic-json.v1",
		) &&
		p24CompatibilityStabilityFileContains(
			"tools/cmd/validate-diagnostic/main_test.go",
			"TestValidateDiagnosticAcceptsStableShape",
		)
	releaseDocs := p24CompatibilityStabilityFileContains(
		"docs/roadmaps/early/roadmap_0_6_1_to_0_6_3.md",
		"TETRA0001",
	) &&
		p24CompatibilityStabilityFileContains(
			"docs/roadmaps/early/roadmap_0_6_1_to_0_6_3.md",
			"TETRA2001",
		) &&
		p24CompatibilityStabilityFileContains(
			"docs/release-notes/archive/v0_6.md",
			"validate-diagnostic",
		)
	return CompatibilityStabilityV1Witness{
		ID:                            p24CompatibilityDiagnosticWitnessID,
		Kind:                          "stable_diagnostic_codes",
		Paths:                         paths,
		DiagnosticCodes:               codes,
		DiagnosticRegistryCount:       len(registry),
		DiagnosticCodesValid:          validCodes,
		DiagnosticJSONValidatorStrict: strictValidator,
		DiagnosticReleaseDocsPresent:  releaseDocs,
		StableDiagnosticCodesReviewed: p24AllRepoPathsExist(paths) &&
			len(registry) >= len(required) &&
			validCodes &&
			strictValidator &&
			releaseDocs,
	}
}

func buildP24CompatibilitySchemaWitness() CompatibilityStabilityV1Witness {
	paths := []string{
		"compiler/compiler_evidence_gates.go",
		"compiler/compiler_evidence_gates.go",
		"compiler/compiler_evidence_gates.go",
		"compiler/compiler_evidence_gates.go",
		"compiler/compiler_evidence_gates.go",
		"compiler/compiler_evidence_gates.go",
		"compiler/compiler_evidence_gates.go",
		"compiler/compiler_reports.go",
		"compiler/internal/buildreports/types.go",
		"compiler/internal/buildreports/layout.go",
		"compiler/internal/buildreports/performance.go",
		"compiler/compiler_reports.go",
		"tools/cmd/validate-manifest/main.go",
		"tools/cmd/validate-diagnostic/main.go",
	}
	expectations := []struct {
		path   string
		schema string
	}{
		{"compiler/compiler_evidence_gates.go", "tetra.translation.validation.v2"},
		{"compiler/compiler_evidence_gates.go", "tetra.fuzz.property.differential.v1"},
		{"compiler/compiler_evidence_gates.go", "tetra.formal_core.v1"},
		{"compiler/compiler_evidence_gates.go", "tetra.self_hosting.gate.v1"},
		{"compiler/compiler_evidence_gates.go", "tetra.security.review_gate.v1"},
		{"compiler/compiler_evidence_gates.go", "tetra.runtime.hardening.v1"},
		{"compiler/compiler_evidence_gates.go", compatibilityStabilityV1Schema},
		{"tools/cmd/validate-manifest/main.go", "tetra.release.v0_4_0.manifest-json.v1"},
		{"tools/cmd/validate-diagnostic/main.go", "tetra.release.v0_2_0.diagnostic-json.v1"},
	}
	var schemas []string
	for _, expectation := range expectations {
		if p24CompatibilityStabilityFileContains(expectation.path, expectation.schema) &&
			p24CompatibilityStabilityLooksVersionedSchema(expectation.schema) {
			schemas = append(schemas, expectation.schema)
		}
	}
	sort.Strings(schemas)
	diagnosticSchemaStrictDecode := p24CompatibilityStabilityFileContains(
		"tools/cmd/validate-diagnostic/main.go",
		"DisallowUnknownFields",
	) ||
		(p24CompatibilityStabilityFileContains(
			"tools/cmd/validate-diagnostic/main.go",
			"reportdecode.DecodeStrict",
		) &&
			p24CompatibilityStabilityFileContains(
				"tools/internal/reportdecode/reportdecode.go",
				"DisallowUnknownFields",
			))
	strict := p24CompatibilityStabilityFileContains(
		"compiler/internal/buildreports/types.go",
		"schema_version",
	) &&
		p24CompatibilityStabilityFileContains(
			"compiler/internal/buildreports/layout.go",
			"want 2",
		) &&
		p24CompatibilityStabilityFileContains(
			"compiler/internal/buildreports/performance.go",
			"want 3",
		) &&
		p24CompatibilityStabilityFileContains(
			"tools/cmd/validate-manifest/main.go",
			"DisallowUnknownFields",
		) &&
		diagnosticSchemaStrictDecode
	return CompatibilityStabilityV1Witness{
		ID:                   p24CompatibilitySchemaWitnessID,
		Kind:                 "versioned_report_schemas",
		Paths:                paths,
		SchemaIDs:            schemas,
		VersionedSchemaCount: len(schemas),
		ReportSchemasStrict:  strict,
		VersionedReportSchemasReviewed: p24AllRepoPathsExist(paths) &&
			len(schemas) == len(expectations) &&
			strict,
	}
}

func buildP24CompatibilityManifestWitness() CompatibilityStabilityV1Witness {
	paths := []string{
		"compiler/compiler_facade.go",
		"compiler/compiler_facade.go",
		"docs/generated/manifest.json",
		"tools/cmd/validate-manifest/main.go",
	}
	var manifest struct {
		CompilerVersion string            `json:"compiler_version"`
		Targets         []json.RawMessage `json:"targets"`
		Features        []json.RawMessage `json:"features"`
		RuntimeABI      map[string]any    `json:"runtime_abi"`
	}
	if raw, err := os.ReadFile(p24RepoPath("docs/generated/manifest.json")); err == nil {
		_ = json.Unmarshal(raw, &manifest)
	}
	featureRegistry := FeatureRegistry()
	featureRegistryLinked := len(featureRegistry) > 0 &&
		p24CompatibilityStabilityFileContains("compiler/compiler_facade.go", "FeatureRegistry()") &&
		p24CompatibilityStabilityFileContains(
			"tools/cmd/validate-manifest/main.go",
			"validateFeatures",
		)
	strict := p24CompatibilityStabilityFileContains(
		"tools/cmd/validate-manifest/main.go",
		"decodeStrictJSON",
	) &&
		p24CompatibilityStabilityFileContains(
			"tools/cmd/validate-manifest/main.go",
			"DisallowUnknownFields",
		) &&
		p24CompatibilityStabilityFileContains(
			"tools/cmd/validate-manifest/main.go",
			"manifestArtifact",
		) &&
		p24CompatibilityStabilityFileContains(
			"tools/cmd/validate-manifest/main.go",
			"tetra.release.v0_4_0.manifest-json.v1",
		)
	runtimeChecks := p24CompatibilityStabilityFileContains(
		"tools/cmd/validate-manifest/main.go",
		"validateRuntimeABI",
	) &&
		p24CompatibilityStabilityFileContains(
			"tools/cmd/validate-manifest/main.go",
			"ActorRuntimeTriples",
		) &&
		p24CompatibilityStabilityFileContains("compiler/compiler_facade.go", "RuntimeABI")
	return CompatibilityStabilityV1Witness{
		ID:                              p24CompatibilityManifestWitnessID,
		Kind:                            "manifest_compatibility_checks",
		Paths:                           paths,
		ManifestCompilerVersion:         manifest.CompilerVersion,
		ManifestTargetCount:             len(manifest.Targets),
		ManifestFeatureCount:            len(manifest.Features),
		ManifestRuntimeABIPresent:       len(manifest.RuntimeABI) > 0,
		ManifestValidatorStrict:         strict,
		ManifestFeatureRegistryLinked:   featureRegistryLinked,
		ManifestRuntimeABIChecksPresent: runtimeChecks,
		ManifestCompatibilityChecksReviewed: p24AllRepoPathsExist(paths) &&
			manifest.CompilerVersion != "" &&
			len(manifest.Targets) > 0 &&
			len(manifest.Features) > 0 &&
			len(manifest.RuntimeABI) > 0 &&
			strict &&
			featureRegistryLinked &&
			runtimeChecks,
	}
}

func buildP24CompatibilityMigrationWitness() CompatibilityStabilityV1Witness {
	paths := []string{
		"docs/release/policy/breaking-change-migration-guide.md",
		"docs/spec/policy/api_diff_policy.md",
		"docs/spec/core/current_supported_surface.md",
		"docs/roadmaps/early/roadmap_0_6_1_to_0_6_3.md",
	}
	guide := p24CompatibilityStabilityFileContains(
		"docs/release/policy/breaking-change-migration-guide.md",
		"Breaking Change Triage",
	) &&
		p24CompatibilityStabilityFileContains(
			"docs/release/policy/breaking-change-migration-guide.md",
			"Migration Steps",
		) &&
		p24CompatibilityStabilityFileContains(
			"docs/release/policy/breaking-change-migration-guide.md",
			"Report Schema",
		) &&
		p24CompatibilityStabilityFileContains(
			"docs/release/policy/breaking-change-migration-guide.md",
			"Manifest",
		)
	apiReview := p24CompatibilityStabilityFileContains(
		"docs/spec/policy/api_diff_policy.md",
		"breaking_requires_review",
	) &&
		p24CompatibilityStabilityFileContains(
			"docs/spec/policy/api_diff_policy.md",
			"--enforce no-change",
		)
	patchLine := p24CompatibilityStabilityFileContains(
		"docs/spec/core/current_supported_surface.md",
		"Breaking language or project compatibility changes belong in a",
	) &&
		p24CompatibilityStabilityFileContains(
			"docs/spec/core/current_supported_surface.md",
			"later `x.0.0` line",
		) &&
		p24CompatibilityStabilityFileContains(
			"docs/roadmaps/early/roadmap_0_6_1_to_0_6_3.md",
			"Text diagnostics remain compatible",
		)
	return CompatibilityStabilityV1Witness{
		ID:                       p24CompatibilityMigrationWitnessID,
		Kind:                     "breaking_change_migration_guide",
		Paths:                    paths,
		MigrationGuidePresent:    guide,
		APIBreakingReviewPresent: apiReview,
		PatchLineRulePresent:     patchLine,
		BreakingChangeMigrationGuidePresent: p24AllRepoPathsExist(paths) && guide && apiReview &&
			patchLine,
	}
}

func buildP24CompatibilityDeprecationWitness() CompatibilityStabilityV1Witness {
	paths := []string{
		"docs/release/policy/deprecation_policy.md",
		"docs/release/v1_0/v1_0_x_maintenance_policy.md",
		"docs/spec/standard_library/stdlib_naming_versioning.md",
	}
	policy := p24CompatibilityStabilityFileContains(
		"docs/release/policy/deprecation_policy.md",
		"Deprecation Policy",
	) &&
		p24CompatibilityStabilityFileContains(
			"docs/release/policy/deprecation_policy.md",
			"replacement path",
		) &&
		p24CompatibilityStabilityFileContains(
			"docs/release/policy/deprecation_policy.md",
			"removals wait",
		)
	replacement := p24CompatibilityStabilityFileContains(
		"docs/release/v1_0/v1_0_x_maintenance_policy.md",
		"replacement path",
	) &&
		p24CompatibilityStabilityFileContains(
			"docs/release/v1_0/v1_0_x_maintenance_policy.md",
			"diagnostics or documentation",
		)
	removalDelay := p24CompatibilityStabilityFileContains(
		"docs/release/v1_0/v1_0_x_maintenance_policy.md",
		"removals wait for a later minor",
	) ||
		p24CompatibilityStabilityFileContains(
			"docs/release/v1_0/v1_0_x_maintenance_policy.md",
			"removals wait for a later minor or major line",
		)
	stdlibRule := p24CompatibilityStabilityFileContains(
		"docs/spec/standard_library/stdlib_naming_versioning.md",
		"Breaking changes to `lib.core.*` MUST wait for the next major",
	)
	return CompatibilityStabilityV1Witness{
		ID:    p24CompatibilityDeprecationWitnessID,
		Kind:  "deprecation_policy",
		Paths: paths,
		DeprecationPolicyPresent: p24AllRepoPathsExist(paths) && policy && replacement &&
			removalDelay &&
			stdlibRule,
		ReplacementPathRequired:    replacement,
		RemovalDelayRequired:       removalDelay,
		StdlibMajorLineRulePresent: stdlibRule,
	}
}

func buildP24CompatibilityArtifactsWitness(
	artifacts []CompatibilityStabilityArtifact,
) CompatibilityStabilityV1Witness {
	witness := CompatibilityStabilityV1Witness{
		ID:    p24CompatibilityArtifactsWitnessID,
		Kind:  "compatibility_stability_artifacts",
		Paths: make([]string, 0, len(artifacts)),
	}
	for _, artifact := range artifacts {
		witness.Paths = append(witness.Paths, artifact.Path)
		switch artifact.Path {
		case "docs/audits/security/compatibility-stability-v1.md":
			witness.CompatibilityAuditArtifactPresent = artifact.Present
		case ("docs/plans/2026-06-03/governance-p23-p24/2026-06-03-p24.2-co" +
			"mpatibility-stability-design.md"):
			witness.CompatibilityDesignArtifactPresent = artifact.Present
		case "docs/release/policy/breaking-change-migration-guide.md":
			witness.MigrationGuideArtifactPresent = artifact.Present
		case "docs/release/policy/deprecation_policy.md":
			witness.DeprecationPolicyArtifactPresent = artifact.Present
		}
	}
	return witness
}

func p24CompatibilityStabilityValidateRowsAndWitnesses(
	rows []CompatibilityStabilityV1Row,
	witnesses []CompatibilityStabilityV1Witness,
) error {
	byWitness := map[string]CompatibilityStabilityV1Witness{}
	for _, witness := range witnesses {
		if strings.TrimSpace(witness.ID) == "" || strings.TrimSpace(witness.Kind) == "" {
			return fmt.Errorf("compatibility/stability v1: witness missing id or kind")
		}
		if _, exists := byWitness[witness.ID]; exists {
			return fmt.Errorf("compatibility/stability v1: duplicate witness %q", witness.ID)
		}
		byWitness[witness.ID] = witness
	}
	expected := map[CompatibilityStabilityV1ID]bool{}
	for _, id := range p24CompatibilityStabilityV1IDs() {
		expected[id] = true
	}
	seen := map[CompatibilityStabilityV1ID]bool{}
	for _, row := range rows {
		if !expected[row.ID] {
			return fmt.Errorf("compatibility/stability v1: unexpected row %q", row.ID)
		}
		if seen[row.ID] {
			return fmt.Errorf("compatibility/stability v1: duplicate row %q", row.ID)
		}
		seen[row.ID] = true
		if strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Status) == "" {
			return fmt.Errorf("compatibility/stability v1: row %q missing name or status", row.ID)
		}
		if len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 ||
			len(row.WitnessIDs) == 0 {
			return fmt.Errorf(
				"compatibility/stability v1: row %q missing evidence, tests, boundaries, or witness ids",
				row.ID,
			)
		}
		for _, text := range append(
			append(append([]string{}, row.Evidence...), row.Tests...),
			row.Boundaries...,
		) {
			if p24CompatibilityStabilityIsPlaceholder(text) {
				return fmt.Errorf(
					"compatibility/stability v1: row %q has placeholder evidence",
					row.ID,
				)
			}
		}
		for _, id := range row.WitnessIDs {
			if _, ok := byWitness[id]; !ok {
				return fmt.Errorf(
					"compatibility/stability v1: row %q references missing witness %q",
					row.ID,
					id,
				)
			}
		}
	}
	for _, id := range p24CompatibilityStabilityV1IDs() {
		if !seen[id] {
			return fmt.Errorf("compatibility/stability v1: missing row %q", id)
		}
	}
	diagnosticWitness := byWitness[p24CompatibilityDiagnosticWitnessID]
	if !diagnosticWitness.StableDiagnosticCodesReviewed ||
		diagnosticWitness.DiagnosticRegistryCount < 10 ||
		!diagnosticWitness.DiagnosticCodesValid ||
		!diagnosticWitness.DiagnosticJSONValidatorStrict ||
		!diagnosticWitness.DiagnosticReleaseDocsPresent {
		return fmt.Errorf("compatibility/stability v1: stable diagnostic witness incomplete")
	}
	schemaWitness := byWitness[p24CompatibilitySchemaWitnessID]
	if !schemaWitness.VersionedReportSchemasReviewed ||
		schemaWitness.VersionedSchemaCount < 8 ||
		!schemaWitness.ReportSchemasStrict {
		return fmt.Errorf("compatibility/stability v1: versioned report schema witness incomplete")
	}
	manifestWitness := byWitness[p24CompatibilityManifestWitnessID]
	if !manifestWitness.ManifestCompatibilityChecksReviewed ||
		manifestWitness.ManifestCompilerVersion == "" ||
		manifestWitness.ManifestTargetCount == 0 ||
		manifestWitness.ManifestFeatureCount == 0 ||
		!manifestWitness.ManifestRuntimeABIPresent ||
		!manifestWitness.ManifestValidatorStrict ||
		!manifestWitness.ManifestFeatureRegistryLinked ||
		!manifestWitness.ManifestRuntimeABIChecksPresent {
		return fmt.Errorf("compatibility/stability v1: manifest compatibility witness incomplete")
	}
	migrationWitness := byWitness[p24CompatibilityMigrationWitnessID]
	if !migrationWitness.BreakingChangeMigrationGuidePresent ||
		!migrationWitness.MigrationGuidePresent ||
		!migrationWitness.APIBreakingReviewPresent ||
		!migrationWitness.PatchLineRulePresent {
		return fmt.Errorf(
			"compatibility/stability v1: breaking-change migration witness incomplete",
		)
	}
	if witness := byWitness[p24CompatibilityDeprecationWitnessID]; !witness.DeprecationPolicyPresent ||
		!witness.ReplacementPathRequired ||
		!witness.RemovalDelayRequired ||
		!witness.StdlibMajorLineRulePresent {
		return fmt.Errorf("compatibility/stability v1: deprecation policy witness incomplete")
	}
	artifactsWitness := byWitness[p24CompatibilityArtifactsWitnessID]
	if !artifactsWitness.CompatibilityAuditArtifactPresent ||
		!artifactsWitness.CompatibilityDesignArtifactPresent ||
		!artifactsWitness.MigrationGuideArtifactPresent ||
		!artifactsWitness.DeprecationPolicyArtifactPresent {
		return fmt.Errorf(
			"compatibility/stability v1: compatibility/stability artifact witness incomplete",
		)
	}
	return nil
}

func p24CompatibilityStabilityValidateArtifacts(report CompatibilityStabilityV1Report) error {
	present := map[string]bool{}
	for _, artifact := range report.Artifacts {
		if strings.TrimSpace(artifact.Kind) == "" || strings.TrimSpace(artifact.Path) == "" {
			return fmt.Errorf("compatibility/stability v1: artifact missing kind or path")
		}
		present[artifact.Path] = artifact.Present
	}
	for _, path := range []string{
		"docs/audits/security/compatibility-stability-v1.md",
		"docs/plans/2026-06-03/governance-p23-p24/2026-06-03-p24.2-compatibility-stability-design.md",
		"docs/release/policy/breaking-change-migration-guide.md",
		"docs/release/policy/deprecation_policy.md",
	} {
		if !present[path] {
			return fmt.Errorf("compatibility/stability v1: required artifact %s missing", path)
		}
	}
	return nil
}

func p24CompatibilityStabilityV1IDs() []CompatibilityStabilityV1ID {
	return []CompatibilityStabilityV1ID{
		CompatibilityStableDiagnosticCodes,
		CompatibilityVersionedReportSchemas,
		CompatibilityManifestChecks,
		CompatibilityBreakingChangeMigrationGuide,
		CompatibilityDeprecationPolicy,
	}
}

func p24CompatibilityStabilityRow(
	id CompatibilityStabilityV1ID,
	name, status string,
	evidence, tests, boundaries, witnessIDs []string,
) CompatibilityStabilityV1Row {
	return CompatibilityStabilityV1Row{
		ID:         id,
		Name:       name,
		Status:     status,
		Evidence:   evidence,
		Tests:      tests,
		Boundaries: boundaries,
		WitnessIDs: witnessIDs,
	}
}

func p24CompatibilityStabilityArtifacts() []CompatibilityStabilityArtifact {
	return []CompatibilityStabilityArtifact{
		p24CompatibilityStabilityArtifact(
			"compatibility_stability_audit",
			"docs/audits/security/compatibility-stability-v1.md",
		),
		p24CompatibilityStabilityArtifact(
			"compatibility_stability_design",
			"docs/plans/2026-06-03/governance-p23-p24/2026-06-03-p24.2-compatibility-stability-design.md",
		),
		p24CompatibilityStabilityArtifact(
			"breaking_change_migration_guide",
			"docs/release/policy/breaking-change-migration-guide.md",
		),
		p24CompatibilityStabilityArtifact(
			"deprecation_policy",
			"docs/release/policy/deprecation_policy.md",
		),
	}
}

func p24CompatibilityStabilityArtifact(kind string, rel string) CompatibilityStabilityArtifact {
	_, err := os.Stat(p24RepoPath(rel))
	return CompatibilityStabilityArtifact{
		Kind:    kind,
		Path:    rel,
		Present: err == nil,
	}
}

func p24CompatibilityStabilityFileContains(rel string, want string) bool {
	data, err := os.ReadFile(p24RepoPath(rel))
	return err == nil && strings.Contains(string(data), want)
}

func p24CompatibilityStabilityLooksVersionedSchema(schema string) bool {
	last := strings.LastIndex(schema, ".v")
	if last < 0 || last+2 >= len(schema) {
		return false
	}
	for _, r := range schema[last+2:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func p24CompatibilityStabilityHasString(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

func p24CompatibilityStabilityIsPlaceholder(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return lower == "" ||
		lower == "todo" ||
		lower == "tbd" ||
		strings.Contains(lower, "placeholder")
}

// ---- feature_surface_audit.go ----

const (
	featureSurfaceAuditSchemaV1  = "tetra.language.feature_surface_audit.v1"
	featureSurfaceAuditScopeP220 = "p22.0_full_feature_surface_audit"
)

type FeatureSurfaceAuditCategory string
type featureSurfaceCategory = FeatureSurfaceAuditCategory

const (
	FeatureSurfaceFirstClassCallables          featureSurfaceCategory = "first_class_callables"
	FeatureSurfaceClosures                     featureSurfaceCategory = "closures"
	FeatureSurfaceProtocolsTraitObjects        featureSurfaceCategory = "protocols_trait_objects"
	FeatureSurfaceRuntimeGenerics              featureSurfaceCategory = "runtime_generics"
	FeatureSurfaceAdvancedEnumsPatternMatching                        = featureSurfaceCategory(
		"advanced_enums_pattern_matching",
	)
	FeatureSurfaceAsyncTypedErrors      featureSurfaceCategory = "async_typed_errors"
	FeatureSurfaceStructuredConcurrency featureSurfaceCategory = "structured_concurrency"
	FeatureSurfaceModulesPackages       featureSurfaceCategory = "modules_packages"
	FeatureSurfaceMacrosMetaprogramming featureSurfaceCategory = "macros_metaprogramming"
	FeatureSurfaceUISurface             featureSurfaceCategory = "ui_surface"
	FeatureSurfaceEcoCapsules           featureSurfaceCategory = "eco_capsules"
)

type FeatureSurfaceAuditReport struct {
	SchemaVersion string                   `json:"schema_version"`
	Scope         string                   `json:"scope"`
	Rows          []FeatureSurfaceAuditRow `json:"rows"`
	NonClaims     []string                 `json:"non_claims"`

	PromotedWithoutSameBranchEvidence bool `json:"promoted_without_same_branch_evidence"`

	FullV1GuaranteesClaimed      bool `json:"full_v1_guarantees_claimed"`
	RuntimeGenericValuesClaimed  bool `json:"runtime_generic_values_claimed"`
	TraitObjectsClaimed          bool `json:"trait_objects_claimed"`
	MacroSystemClaimed           bool `json:"macro_system_claimed"`
	StructuredConcurrencyClaimed bool `json:"structured_concurrency_claimed"`

	CrossPlatformUIRuntimeClaimed bool `json:"cross_platform_ui_runtime_claimed"`

	DistributedEcoClaimed        bool `json:"distributed_eco_claimed"`
	ProofCarryingCapsulesClaimed bool `json:"proof_carrying_capsules_claimed"`
	PerformanceClaimed           bool `json:"performance_claimed"`
	SafeSemanticsChanged         bool `json:"safe_semantics_changed"`
}

type FeatureSurfaceAuditRow struct {
	Category                  FeatureSurfaceAuditCategory `json:"category"`
	Name                      string                      `json:"name"`
	Decision                  string                      `json:"decision"`
	FeatureIDs                []string                    `json:"feature_ids"`
	RegistryStatuses          map[string]FeatureStatus    `json:"registry_statuses"`
	Evidence                  []string                    `json:"evidence"`
	Boundaries                []string                    `json:"boundaries"`
	RequiredPromotionEvidence []string                    `json:"required_promotion_evidence"`
	SameBranchEvidence        bool                        `json:"same_branch_evidence"`
	PromotedInThisAudit       bool                        `json:"promoted_in_this_audit"`
}

func BuildP22FeatureSurfaceAudit() FeatureSurfaceAuditReport {
	registry := featureSurfaceRegistryByID()
	return FeatureSurfaceAuditReport{
		SchemaVersion: featureSurfaceAuditSchemaV1,
		Scope:         featureSurfaceAuditScopeP220,
		Rows: []FeatureSurfaceAuditRow{
			p22FeatureSurfaceRow(
				registry,
				FeatureSurfaceFirstClassCallables,
				"First-class callables",
				"keep_current_bounded_and_route_full_expansion_to_P22.1",
				[]string{
					"language.callable-mvp",
					"language.callable-level1",
					"language.callable-level2",
					"language.full-first-class-callables",
				},
				[]string{
					("FeatureRegistry records Level 0/1/2 callable support plus " +
						"language.full-first-class-callables as current within the " +
						"v0.4.0 bounded safe by-value model."),
					("language.full-first-class-callables evidence includes the " +
						"bounded fnptr fast path and the fixed 4-slot callable " +
						"handle for larger immutable " +
						"Int/Bool/String/simple-aggregate captures."),
					("docs/spec/policy/v1_feature_status.md keeps mutable " +
						"by-reference capture, pointer/resource capture, " +
						"thread-boundary callable escape, and dynamic/generic " +
						"callable polymorphism outside the current promotion."),
				},
				[]string{
					"mutable by-reference capture remains diagnostic or future work",
					"pointer/resource capture and thread-boundary callable escape remain unpromoted",
					"P22.1 owns any future first-class callable expansion beyond the current bounded model",
				},
				[]string{
					"P22.1 lifetime/ABI evidence in the same branch",
					"stable diagnostics for unsupported callable movement",
					"registry, docs, manifest, and tests updated in the same branch before promotion",
				},
			),
			p22FeatureSurfaceRow(
				registry,
				FeatureSurfaceClosures,
				"Closures",
				"keep_safe_by_value_capture_slice_only",
				[]string{"language.callable-level2", "language.full-first-class-callables"},
				[]string{
					("FeatureRegistry records captured closure Level 2 plus full " +
						"first-class callables as current only for safe by-value " +
						"captures and fixed-handle movement."),
					("Current evidence covers local storage, aliases, returns, " +
						"struct fields, enum payloads, synchronous callback " +
						"arguments, and generated interface metadata."),
					("same-branch evidence is required before promoting " +
						"pointer/resource capture, mutable by-reference capture, " +
						"generic closure capture, or thread movement."),
				},
				[]string{
					"pointer/resource capture stays outside the promoted closure surface",
					"generic closure and generic callback-closure capture remain rejected",
					"mutable capture escape and thread-boundary movement stay gated by diagnostics",
				},
				[]string{
					"same-branch evidence for ownership, synchronization, lifetime, ABI, docs, and diagnostics",
					"new closure tests proving each movement path",
				},
			),
			p22FeatureSurfaceRow(
				registry,
				FeatureSurfaceProtocolsTraitObjects,
				"Protocols and trait objects",
				"keep_static_conformance_only_and_route_runtime_existentials_to_P22.2",
				[]string{
					"language.protocol-conformance-mvp",
					"language.protocol-bound-generics-static",
				},
				[]string{
					("FeatureRegistry records static conformance and static " +
						"protocol-bound generic validation during monomorphization."),
					("Current scope explicitly says no witness tables, trait " +
						"objects, runtime protocol values, or dynamic dispatch model."),
					("P22.2 owns any decision to design runtime existential " +
						"values while keeping the static fast path."),
				},
				[]string{
					"no witness tables are promoted",
					"trait objects and runtime protocol values remain post-v1 unless P22.2 gates them",
					"dynamic dispatch and conformance-table lookup remain unsupported",
				},
				[]string{
					"P22.2 design and implementation evidence in the same branch",
					"ABI/report-visible dynamic dispatch evidence if runtime existentials are promoted",
				},
			),
			p22FeatureSurfaceRow(
				registry,
				FeatureSurfaceRuntimeGenerics,
				"Runtime generics",
				"keep_static_monomorphized_generic_functions_only",
				[]string{"language.generics-mvp", "language.protocol-bound-generics-static"},
				[]string{
					("FeatureRegistry records statically monomorphized generic " +
						"functions with inferred value arguments and static " +
						"protocol-bound validation."),
					("docs/spec/flow/v1_scope.md keeps runtime generic values, " +
						"explicit type arguments, generic structs, higher-ranked " +
						"generics, and full protocol-bound generic dispatch post-v1 " +
						"unless promoted."),
				},
				[]string{
					"runtime generic values are not current",
					("explicit type arguments, generic structs, and higher-ranked " +
						"generics remain outside current support"),
					("full protocol-bound generic dispatch and broad " +
						"specialization guarantees are not promoted here"),
				},
				[]string{
					"parser, semantics, ABI, optimizer, docs, manifest, and validator evidence in the same branch",
				},
			),
			p22FeatureSurfaceRow(
				registry,
				FeatureSurfaceAdvancedEnumsPatternMatching,
				"Advanced enums and pattern matching",
				"keep_positional_enum_payload_slice_only",
				[]string{"language.enum-payload-match"},
				[]string{
					("FeatureRegistry records positional enum payload " +
						"constructors and payload bindings for match/catch/if-let."),
					("Current scope includes exhaustive unguarded enum " +
						"match/catch coverage and stable diagnostics for payload " +
						"arity/type/syntax errors."),
				},
				[]string{
					"advanced ADT constructors remain future/post-v1",
					"nested destructuring patterns remain future/post-v1",
					"guard expansion and richer payload algebra remain future/post-v1",
				},
				[]string{
					("same-branch parser, semantics, lowering, diagnostics, docs, " +
						"and manifest evidence for each promoted pattern form"),
				},
			),
			p22FeatureSurfaceRow(
				registry,
				FeatureSurfaceAsyncTypedErrors,
				"Async typed errors",
				"keep_try_await_boundary_only",
				[]string{"language.task-handles-mvp", "language.resource-lifetime-mvp"},
				[]string{
					("docs/spec/flow/v1_scope.md defines async typed-error " +
						"support as the checked try await <call>() " +
						"synchronous-lowering boundary."),
					("FeatureRegistry records typed task handles and resource " +
						"lifetime checks for task handles, task groups, islands, and " +
						"typed-error resource aliases."),
					("await try <call>() remains rejected by stable diagnostics " +
						"rather than promoted by this audit."),
				},
				[]string{
					"async typed-error behavior beyond try await stays post-v1",
					"cancellation and structured concurrency are not promoted by the async typed-error row",
					"await try remains a rejected boundary form",
				},
				[]string{
					("same-branch async parser/checker/lowering/runtime/docs " +
						"evidence for any extension beyond try await"),
				},
			),
			p22FeatureSurfaceRow(
				registry,
				FeatureSurfaceStructuredConcurrency,
				"Structured concurrency",
				"keep_local_task_actor_bounded_and_full_structured_concurrency_future",
				[]string{
					"actors.task-transfer-safety",
					"language.task-handles-mvp",
					"actors.distributed-runtime",
				},
				[]string{
					("FeatureRegistry records actor/task transfer safety as a " +
						"conservative local MVP and typed task handle wrappers for " +
						"slot counts 2..8."),
					("actors.distributed-runtime is current only for the " +
						"Linux-x64 distributed actor runtime path and explicitly " +
						"excludes broader structured-concurrency guarantees."),
					("Existing scheduler/reactor reports mention cancellation " +
						"checkpoints and task groups as evidence rows, not as a full " +
						"structured concurrency claim."),
				},
				[]string{
					"full cancellation remains outside the current support claim",
					"full race-safety proof remains outside the current support claim",
					"broader structured-concurrency guarantees remain outside the current actor/task MVP",
				},
				[]string{
					("same-branch scheduler, cancellation, task-group, actor, " +
						"race, docs, and manifest evidence before any full " +
						"structured concurrency promotion"),
				},
			),
			p22FeatureSurfaceRow(
				registry,
				FeatureSurfaceModulesPackages,
				"Modules and packages",
				"keep_local_module_package_capsule_surface_only",
				[]string{"language.globals-properties-capsule-mvp", "eco.local-package-lifecycle"},
				[]string{
					("FeatureRegistry records compile-time capsule metadata plus " +
						"local Eco package lifecycle support."),
					("Current local package lifecycle covers verify, lock " +
						"generation/validation, pack/unpack, vault, stable/beta " +
						"metadata, target-aware download, fixtures, local mirror " +
						"reports, and single-origin HTTP(S) fetch into a verified " +
						"local store."),
				},
				[]string{
					"capsule metadata is compile-time metadata, not a runtime proof-carrying capsule system",
					"distributed EcoNet and production TetraHub publishing remain post-v1",
				},
				[]string{
					("same-branch package, module, trust, capsule, docs, manifest," +
						" and security evidence for any distributed promotion"),
				},
			),
			p22FeatureSurfaceRow(
				registry,
				FeatureSurfaceMacrosMetaprogramming,
				"Macros and metaprogramming",
				"keep_absent_post_v1",
				nil,
				[]string{
					"FeatureRegistry has no current macro/metaprogramming feature ID.",
					("no current macro/metaprogramming feature is promoted by " +
						"P22.0; absence is the same-branch evidence for keeping this " +
						"category post-v1."),
				},
				[]string{
					("macro and metaprogramming systems remain post-v1 until a " +
						"concrete design, implementation, tests, docs, and registry " +
						"entry exist"),
				},
				[]string{
					("same-branch evidence must include a new registry ID, " +
						"parser/semantics/tooling tests, docs, manifest updates, and " +
						"non-claim review"),
				},
			),
			p22FeatureSurfaceRow(
				registry,
				FeatureSurfaceUISurface,
				"UI and Surface",
				"keep_linux_web_surface_bounded_and_platform_gate_experimental",
				[]string{
					"ui.metadata-v1",
					"ui.surface-core",
					"ui.surface-linux-x64",
					"ui.surface-web-wasm",
					"ui.native-runtime",
					"ui.platform-runtime",
					"ui.surface-macos-x64",
					"ui.surface-windows-x64",
					"ui.surface-wasm32-wasi",
				},
				[]string{
					("FeatureRegistry records current UI metadata, bounded " +
						"Surface core, Linux-x64 Surface host, wasm32-web Surface, " +
						"and Linux-x64 native UI runtime evidence."),
					("ui.platform-runtime remains experimental and requires Linux," +
						" Windows, macOS, and Web runtime-backed reports before " +
						"production promotion."),
					"macOS, Windows, and wasm32-wasi Surface hosts are unsupported in the registry.",
				},
				[]string{
					"cross-platform production UI runtime is not claimed",
					"macOS and Windows native runtime claims require real target-host reports",
					"platform accessibility integration and broad native widget behavior remain gated",
				},
				[]string{
					("same-branch Linux, Windows, macOS, Web runtime-backed " +
						"reports with artifact hashes, docs, manifest, and " +
						"validators before cross-platform promotion"),
				},
			),
			p22FeatureSurfaceRow(
				registry,
				FeatureSurfaceEcoCapsules,
				"Eco and capsules",
				"keep_local_eco_and_metadata_capsules_current_distributed_post_v1",
				[]string{
					"language.globals-properties-capsule-mvp",
					"eco.local-package-lifecycle",
					"eco.distributed-network",
				},
				[]string{
					("FeatureRegistry records local Eco lifecycle as current and " +
						"eco.distributed-network as post-v1."),
					("language.globals-properties-capsule-mvp covers compile-time " +
						"capsule metadata; this is not proof-carrying capsules."),
					("Current support is local Eco evidence, not distributed " +
						"EcoNet, production TetraHub publishing, global trust " +
						"scoring, or proof-carrying capsules."),
				},
				[]string{
					"distributed EcoNet remains post-v1",
					"proof-carrying capsules remain post-v1",
					"global trust scoring and production publishing remain post-v1",
				},
				[]string{
					("same-branch distributed network, trust, capsule proof, " +
						"package publishing, docs, manifest, and security evidence " +
						"before promotion"),
				},
			),
		},
		NonClaims: p22FeatureSurfaceAuditNonClaims(),
	}
}

func ValidateP22FeatureSurfaceAudit(report FeatureSurfaceAuditReport) error {
	if report.SchemaVersion != featureSurfaceAuditSchemaV1 {
		return fmt.Errorf(
			"feature surface audit schema = %q, want %q",
			report.SchemaVersion,
			featureSurfaceAuditSchemaV1,
		)
	}
	if report.Scope != featureSurfaceAuditScopeP220 {
		return fmt.Errorf(
			"feature surface audit scope = %q, want %q",
			report.Scope,
			featureSurfaceAuditScopeP220,
		)
	}
	if report.PromotedWithoutSameBranchEvidence {
		return fmt.Errorf(
			"feature surface audit: same-branch evidence is required before promotion",
		)
	}
	if report.FullV1GuaranteesClaimed {
		return fmt.Errorf("feature surface audit: full v1 guarantee claim is forbidden")
	}
	if report.RuntimeGenericValuesClaimed {
		return fmt.Errorf("feature surface audit: runtime generic value claim is forbidden")
	}
	if report.TraitObjectsClaimed {
		return fmt.Errorf("feature surface audit: trait object claim is forbidden")
	}
	if report.MacroSystemClaimed {
		return fmt.Errorf("feature surface audit: macro system claim is forbidden")
	}
	if report.StructuredConcurrencyClaimed {
		return fmt.Errorf("feature surface audit: structured concurrency claim is forbidden")
	}
	if report.CrossPlatformUIRuntimeClaimed {
		return fmt.Errorf(
			"feature surface audit: cross-platform production UI runtime claim is forbidden",
		)
	}
	if report.DistributedEcoClaimed {
		return fmt.Errorf("feature surface audit: distributed Eco claim is forbidden")
	}
	if report.ProofCarryingCapsulesClaimed {
		return fmt.Errorf("feature surface audit: proof-carrying capsule claim is forbidden")
	}
	if report.PerformanceClaimed {
		return fmt.Errorf("feature surface audit: performance claim is forbidden")
	}
	if report.SafeSemanticsChanged {
		return fmt.Errorf("feature surface audit: safe-program semantics change is forbidden")
	}
	for _, nonClaim := range p22FeatureSurfaceAuditNonClaims() {
		if !p22FeatureSurfaceHasString(report.NonClaims, nonClaim) {
			return fmt.Errorf("feature surface audit: missing non-claim %q", nonClaim)
		}
	}
	if err := validateP22FeatureSurfaceStrings("non-claim", report.NonClaims); err != nil {
		return err
	}

	registry := featureSurfaceRegistryByID()
	expected := map[FeatureSurfaceAuditCategory]bool{}
	for _, category := range p22FeatureSurfaceAuditCategories() {
		expected[category] = true
	}
	seen := map[FeatureSurfaceAuditCategory]bool{}
	for _, row := range report.Rows {
		if row.Category == "" || strings.TrimSpace(row.Name) == "" ||
			strings.TrimSpace(row.Decision) == "" {
			return fmt.Errorf("feature surface audit: row missing required metadata: %#v", row)
		}
		if !expected[row.Category] {
			return fmt.Errorf("feature surface audit: unexpected category %s", row.Category)
		}
		if seen[row.Category] {
			return fmt.Errorf("feature surface audit: duplicate category %s", row.Category)
		}
		seen[row.Category] = true
		if row.PromotedInThisAudit && !row.SameBranchEvidence {
			return fmt.Errorf(
				"feature surface audit: row %s promotion lacks same-branch evidence",
				row.Category,
			)
		}
		if !row.SameBranchEvidence {
			return fmt.Errorf(
				"feature surface audit: row %s missing same-branch evidence",
				row.Category,
			)
		}
		if err := validateP22FeatureSurfaceStrings(
			"row "+string(row.Category)+" evidence",
			row.Evidence,
		); err != nil {
			return err
		}
		if err := validateP22FeatureSurfaceStrings(
			"row "+string(row.Category)+" boundary",
			row.Boundaries,
		); err != nil {
			return err
		}
		if err := validateP22FeatureSurfaceStrings(
			"row "+string(row.Category)+" promotion evidence",
			row.RequiredPromotionEvidence,
		); err != nil {
			return err
		}
		if row.Category != FeatureSurfaceMacrosMetaprogramming && len(row.FeatureIDs) == 0 {
			return fmt.Errorf("feature surface audit: row %s missing feature IDs", row.Category)
		}
		if row.Category == FeatureSurfaceMacrosMetaprogramming && len(row.FeatureIDs) == 0 {
			combined := p22FeatureSurfaceCombined(row)
			if !strings.Contains(combined, "no current macro/metaprogramming feature") ||
				!strings.Contains(combined, "post-v1") {
				return fmt.Errorf(
					("feature surface audit: macro/metaprogramming row must " +
						"record no current feature and post-v1 boundary"),
				)
			}
		}
		if row.RegistryStatuses == nil {
			return fmt.Errorf(
				"feature surface audit: row %s missing registry statuses",
				row.Category,
			)
		}
		featureSeen := map[string]bool{}
		for _, id := range row.FeatureIDs {
			if strings.TrimSpace(id) == "" {
				return fmt.Errorf(
					"feature surface audit: row %s has empty feature ID",
					row.Category,
				)
			}
			feature, ok := registry[id]
			if !ok {
				return fmt.Errorf(
					"feature surface audit: unknown feature %s in row %s",
					id,
					row.Category,
				)
			}
			if featureSeen[id] {
				return fmt.Errorf(
					"feature surface audit: row %s duplicates feature ID %s",
					row.Category,
					id,
				)
			}
			featureSeen[id] = true
			if row.RegistryStatuses[id] != feature.Status {
				return fmt.Errorf(
					"feature surface audit: registry status drift for %s in row %s: got %q want %q",
					id,
					row.Category,
					row.RegistryStatuses[id],
					feature.Status,
				)
			}
		}
	}
	for _, category := range p22FeatureSurfaceAuditCategories() {
		if !seen[category] {
			return fmt.Errorf("feature surface audit: missing category %s", category)
		}
	}
	return nil
}

func p22FeatureSurfaceAuditCategories() []FeatureSurfaceAuditCategory {
	return []FeatureSurfaceAuditCategory{
		FeatureSurfaceFirstClassCallables,
		FeatureSurfaceClosures,
		FeatureSurfaceProtocolsTraitObjects,
		FeatureSurfaceRuntimeGenerics,
		FeatureSurfaceAdvancedEnumsPatternMatching,
		FeatureSurfaceAsyncTypedErrors,
		FeatureSurfaceStructuredConcurrency,
		FeatureSurfaceModulesPackages,
		FeatureSurfaceMacrosMetaprogramming,
		FeatureSurfaceUISurface,
		FeatureSurfaceEcoCapsules,
	}
}

func p22FeatureSurfaceAuditNonClaims() []string {
	return []string{
		"no full v1 language guarantee is claimed",
		"no runtime generic values are claimed",
		"no trait objects or runtime protocol values are claimed",
		"no macro/metaprogramming system is claimed",
		"no full structured concurrency guarantee is claimed",
		"no cross-platform production UI runtime is claimed",
		"no distributed EcoNet or proof-carrying capsule promotion is claimed",
		"no performance claim is made",
		"safe-program semantics do not change",
	}
}

func p22FeatureSurfaceRow(
	registry map[string]FeatureInfo,
	category FeatureSurfaceAuditCategory,
	name, decision string,
	featureIDs, evidence, boundaries, promotionEvidence []string,
) FeatureSurfaceAuditRow {
	statuses := map[string]FeatureStatus{}
	for _, id := range featureIDs {
		statuses[id] = registry[id].Status
	}
	return FeatureSurfaceAuditRow{
		Category:                  category,
		Name:                      name,
		Decision:                  decision,
		FeatureIDs:                append([]string{}, featureIDs...),
		RegistryStatuses:          statuses,
		Evidence:                  append([]string{}, evidence...),
		Boundaries:                append([]string{}, boundaries...),
		RequiredPromotionEvidence: append([]string{}, promotionEvidence...),
		SameBranchEvidence:        true,
	}
}

func featureSurfaceRegistryByID() map[string]FeatureInfo {
	registry := map[string]FeatureInfo{}
	for _, feature := range FeatureRegistry() {
		registry[feature.ID] = feature
	}
	return registry
}

func validateP22FeatureSurfaceStrings(label string, items []string) error {
	if len(items) == 0 {
		return fmt.Errorf("feature surface audit: %s missing", label)
	}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			return fmt.Errorf("feature surface audit: %s contains empty item", label)
		}
		if p22FeatureSurfaceContainsPlaceholder(trimmed) {
			return fmt.Errorf(
				"feature surface audit: %s contains placeholder evidence: %q",
				label,
				item,
			)
		}
	}
	return nil
}

func p22FeatureSurfaceContainsPlaceholder(text string) bool {
	lower := strings.ToLower(text)
	for _, token := range []string{"todo", "tbd", "placeholder", "fixme", "???"} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func p22FeatureSurfaceCombined(row FeatureSurfaceAuditRow) string {
	return row.Name + " " + row.Decision + " " + strings.Join(
		row.Evidence,
		" ",
	) + " " + strings.Join(
		row.Boundaries,
		" ",
	) + " " + strings.Join(
		row.RequiredPromotionEvidence,
		" ",
	)
}

func p22FeatureSurfaceHasString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

// ---- first_class_callables_coverage.go ----

const (
	firstClassCallableCoverageSchemaV1  = "tetra.language.first_class_callables.v1"
	firstClassCallableCoverageScopeP221 = "p22.1_first_class_callables_v1"

	firstClassCallableFnPtrWitnessID     = "bounded_one_capture_fnptr"
	firstClassCallableHandleWitnessID    = "nine_capture_handle"
	firstClassCallableInterfaceWitnessID = "cross_module_returned_handle_metadata"
)

type FirstClassCallableCoverageID string
type firstClassCallableID = FirstClassCallableCoverageID

const (
	FirstClassCallableFnPtrFastPath             firstClassCallableID = "fnptr_fast_path"
	FirstClassCallableFatHandle                 firstClassCallableID = "fat_callable_handle"
	FirstClassCallableCaptureSafetyClassifier   firstClassCallableID = "capture_safety_classifier"
	FirstClassCallableMutableCaptureDiagnostics                      = firstClassCallableID(
		"mutable_capture_escape_diagnostics",
	)
	FirstClassCallableResourceThreadDiagnostics = firstClassCallableID(
		"resource_thread_escape_diagnostics",
	)
	FirstClassCallableFixedABIWidth     firstClassCallableID = "fixed_abi_width"
	FirstClassCallableInterfaceMetadata                      = firstClassCallableID(
		"cross_module_interface_metadata",
	)
	FirstClassCallableStorageCallbackPaths firstClassCallableID = "storage_and_callback_paths"
)

type FirstClassCallableCoverageReport struct {
	SchemaVersion string                          `json:"schema_version"`
	Scope         string                          `json:"scope"`
	Rows          []FirstClassCallableCoverageRow `json:"rows"`
	Witnesses     []FirstClassCallableABIWitness  `json:"witnesses"`
	NonClaims     []string                        `json:"non_claims"`

	VariableABIWidthClaimed bool `json:"variable_abi_width_claimed"`

	ExplodingReturnSlotsClaimed bool `json:"exploding_return_slots_claimed"`

	MutableByRefCaptureClaimed bool `json:"mutable_by_ref_capture_claimed"`

	PointerResourceCaptureClaimed bool `json:"pointer_resource_capture_claimed"`

	ThreadBoundaryCallableTransferClaimed bool `json:"thread_boundary_callable_transfer_claimed"`

	RuntimeGenericPolymorphismClaimed bool `json:"runtime_generic_callable_polymorphism_claimed"`

	DynamicCallableDispatchClaimed bool `json:"dynamic_callable_dispatch_claimed"`

	UnsafeLifetimeRelaxationClaimed bool `json:"unsafe_lifetime_relaxation_claimed"`

	PerformanceClaimed bool `json:"performance_claimed"`

	RuntimeBehaviorChanged bool `json:"runtime_behavior_changed"`

	SafeSemanticsChanged bool `json:"safe_semantics_changed"`
}

type FirstClassCallableCoverageRow struct {
	ID         FirstClassCallableCoverageID `json:"id"`
	Name       string                       `json:"name"`
	Status     string                       `json:"status"`
	Evidence   []string                     `json:"evidence"`
	Tests      []string                     `json:"tests"`
	Boundaries []string                     `json:"boundaries"`
	WitnessIDs []string                     `json:"witness_ids"`
}

type FirstClassCallableABIWitness struct {
	ID                         string `json:"id"`
	Kind                       string `json:"kind"`
	CaptureCount               int    `json:"capture_count"`
	FnPtrSlotCount             int    `json:"fnptr_slot_count"`
	CallableHandleSlotCount    int    `json:"callable_handle_slot_count"`
	LocalSlotCount             int    `json:"local_slot_count"`
	UsesHandle                 bool   `json:"uses_handle"`
	AllocBytesCount            int    `json:"alloc_bytes_count"`
	EnvWriteCount              int    `json:"env_write_count"`
	EnvReadCount               int    `json:"env_read_count"`
	CallArgSlots               int    `json:"call_arg_slots"`
	CallRetSlots               int    `json:"call_ret_slots"`
	ReturnSlots                int    `json:"return_slots"`
	FunctionEscapeKind         string `json:"function_escape_kind"`
	InterfaceMetadataPreserved bool   `json:"interface_metadata_preserved"`
}

func BuildP22FirstClassCallableCoverage() (FirstClassCallableCoverageReport, error) {
	fnptr, err := buildP22FnPtrCallableWitness()
	if err != nil {
		return FirstClassCallableCoverageReport{}, err
	}
	handle, err := buildP22HandleCallableWitness()
	if err != nil {
		return FirstClassCallableCoverageReport{}, err
	}
	iface, err := buildP22InterfaceCallableWitness()
	if err != nil {
		return FirstClassCallableCoverageReport{}, err
	}

	report := FirstClassCallableCoverageReport{
		SchemaVersion: firstClassCallableCoverageSchemaV1,
		Scope:         firstClassCallableCoverageScopeP221,
		Witnesses:     []FirstClassCallableABIWitness{fnptr, handle, iface},
		Rows: []FirstClassCallableCoverageRow{
			p22FirstClassCallableRow(
				FirstClassCallableFnPtrFastPath,
				"Bounded fnptr fast path",
				"current_evidence",
				[]string{
					("compiler/internal/semantics/semantics_core.go fixes " +
						"FnPtrEnvSlotCount = 8 and FnPtrSlotCount = 9 for the " +
						"compact fnptr representation."),
					("Parse/Check/Lower witness bounded_one_capture_fnptr records " +
						"a one-capture fnptr value with 9-slot local metadata and no " +
						"heap environment allocation."),
					("compiler/internal/lower/lower_callables.go emits IRSymAddr " +
						"plus padded environment slots for fnptr values without " +
						"IRAllocBytes in the bounded witness."),
				},
				[]string{
					"go test ./compiler -run 'P22FirstClassCallable|ValidateP22FirstClassCallable'",
					"go test ./compiler/internal/lower -run 'Callable'",
				},
				[]string{
					("fnptr fast path is bounded to safe captures whose " +
						"environment fits within FnPtrEnvSlotCount = 8"),
					"no variable-width fnptr ABI is claimed",
					"no heap environment is allocated for the bounded fnptr witness",
				},
				[]string{firstClassCallableFnPtrWitnessID},
			),
			p22FirstClassCallableRow(
				FirstClassCallableFatHandle,
				"Fat callable handle for larger captures",
				"current_evidence",
				[]string{
					("compiler/internal/semantics/semantics_core.go fixes " +
						"CallableHandleSlotCount = 4 for callable handles."),
					("Parse/Check/Lower witness nine_capture_handle records a " +
						"nine-capture callable with a 4-slot handle local and heap " +
						"escape metadata."),
					("compiler/internal/lower/lower_callables.go emits " +
						"IRAllocBytes, IRMemWritePtrOffset, and IRMemReadPtrOffset " +
						"for the handle witness while calling the closure with " +
						"explicit argument plus 9 env slots."),
				},
				[]string{
					"go test ./compiler -run 'P22FirstClassCallable|ValidateP22FirstClassCallable'",
					"go test ./compiler/internal/lower -run 'Callable'",
				},
				[]string{
					"larger safe immutable captures use the fixed handle path",
					"callable returns and locals do not explode beyond the 4-slot handle",
					"handle lowering evidence is IR-only; no performance claim is made",
				},
				[]string{firstClassCallableHandleWitnessID},
			),
			p22FirstClassCallableRow(
				FirstClassCallableCaptureSafetyClassifier,
				"Capture safety classifier",
				"current_evidence",
				[]string{
					("compiler/internal/semantics/semantics_memory_resources.go " +
						"classifies local, return, global, callback, and thread " +
						"callable escape boundaries."),
					("compiler/internal/semantics/semantics_memory_resources.go " +
						"restricts escaping callable captures to safe immutable " +
						"by-value Int/Bool/String/simple aggregate payloads."),
					("docs/spec/core/current_supported_surface.md records the " +
						"safe immutable by-value callable capture boundary."),
				},
				[]string{
					"go test ./compiler/internal/semantics -run 'Callable|Closure|FunctionType'",
					"go test ./compiler/tests/semantics -run 'Callable|Closure|FunctionType|Interface'",
				},
				[]string{
					"generic closure captures remain outside this report",
					"surface ephemeral values cannot escape through callable capture",
					"safe immutable by-value captures are the promoted subset",
				},
				[]string{firstClassCallableFnPtrWitnessID, firstClassCallableHandleWitnessID},
			),
			p22FirstClassCallableRow(
				FirstClassCallableMutableCaptureDiagnostics,
				"Mutable capture escape diagnostics",
				"current_evidence",
				[]string{
					("compiler/internal/semantics/semantics_memory_resources.go " +
						"rejects mutable by-reference capture when a callable " +
						"crosses heap-escape or global-escape boundaries."),
					("compiler/internal/semantics/semantics_suite_test.go covers " +
						"mutable global-escape and thread-boundary rejection cases."),
					("docs/spec/core/current_supported_surface.md documents " +
						"mutable by-reference capture as a diagnostic, not a " +
						"supported escape model."),
				},
				[]string{
					"go test ./compiler/internal/semantics -run 'Callable|Closure|FunctionType'",
					"go test ./compiler/tests/semantics -run 'Callable|Closure|FunctionType'",
				},
				[]string{
					"mutable by-reference capture support is not claimed",
					"global-escape mutable capture remains diagnostic",
					"heap-escape mutable capture remains diagnostic",
				},
				[]string{firstClassCallableHandleWitnessID},
			),
			p22FirstClassCallableRow(
				FirstClassCallableResourceThreadDiagnostics,
				"Resource and thread escape diagnostics",
				"current_evidence",
				[]string{
					("compiler/internal/semantics/semantics_memory_resources.go " +
						"rejects pointer/resource capture escape and classifies " +
						"thread-boundary callable escape separately."),
					("compiler/internal/semantics/semantics_suite_test.go covers " +
						"pointer/resource capture and thread-boundary callable " +
						"escape rejection."),
					("docs/spec/core/current_supported_surface.md keeps " +
						"pointer/resource capture and thread-boundary callable " +
						"escape outside the supported callable model."),
				},
				[]string{
					"go test ./compiler/internal/semantics -run 'Callable|Closure|FunctionType'",
					"go test ./compiler/tests/semantics -run 'Callable|Closure|FunctionType'",
				},
				[]string{
					"pointer/resource capture support is not claimed",
					"thread-boundary callable escape is rejected without sync/ownership transfer evidence",
					"no unsafe lifetime relaxation is claimed",
				},
				[]string{firstClassCallableHandleWitnessID},
			),
			p22FirstClassCallableRow(
				FirstClassCallableFixedABIWidth,
				"Fixed callable ABI width",
				"current_evidence",
				[]string{
					("compiler/internal/semantics/semantics_core.go declares " +
						"FnPtrEnvSlotCount = 8, FnPtrSlotCount = 9, and " +
						"CallableHandleSlotCount = 4."),
					("The bounded fnptr witness records FnPtrSlotCount = 9 and " +
						"CallableHandleSlotCount = 4 without heap allocation."),
					("The nine-capture handle witness records " +
						"CallableHandleSlotCount = 4, ReturnSlots = 4, and call " +
						"ArgSlots = 10 RetSlots = 1 for the closure dispatch."),
				},
				[]string{
					"go test ./compiler -run 'P22FirstClassCallable|ValidateP22FirstClassCallable'",
					"go test ./compiler/internal/lower -run 'Callable'",
				},
				[]string{
					"fixed ABI width is evidence, not a new runtime mode",
					"no variable-width callable ABI is claimed",
					"no exploding callable return slots are claimed",
				},
				[]string{
					firstClassCallableFnPtrWitnessID,
					firstClassCallableHandleWitnessID,
					firstClassCallableInterfaceWitnessID,
				},
			),
			p22FirstClassCallableRow(
				FirstClassCallableInterfaceMetadata,
				"Cross-module interface metadata",
				"current_evidence",
				[]string{
					("compiler/compiler_facade.go preserves returned function " +
						"handle metadata in generated .t4i stubs."),
					("compiler/tests/semantics/semantics_types_protocols_test.go " +
						"verifies ReturnFunctionHandleValue, heap escape metadata, " +
						"and ReturnSlots = 4 for returned nine-capture callables."),
					("ParseFile/CheckWorld witness " +
						"cross_module_returned_handle_metadata records generated " +
						".t4i metadata preserved for a returned nine-capture handle."),
				},
				[]string{
					"go test ./compiler/tests/semantics -run 'Interface'",
					"go test ./compiler -run 'P22FirstClassCallable|ValidateP22FirstClassCallable'",
				},
				[]string{
					".t4i metadata is checked evidence for returned callable handles",
					"cross-module metadata preservation does not add dynamic callable dispatch",
					"ReturnFunctionHandleValue and ReturnSlots = 4 are required for handle returns",
				},
				[]string{firstClassCallableInterfaceWitnessID},
			),
			p22FirstClassCallableRow(
				FirstClassCallableStorageCallbackPaths,
				"Storage, callback, and return paths",
				"current_evidence",
				[]string{
					("docs/spec/core/current_supported_surface.md records aliases," +
						" struct fields, enum payloads, callback arguments, returns, " +
						"and same-module global snapshots for safe callable values."),
					("compiler/tests/semantics/semantics_callables_closures_test.g" +
						"o covers returned, struct-field, enum-payload, and callback " +
						"handle movement with nine captured values."),
					("compiler/internal/lower/lower_callables.go routes stable " +
						"callable targets through direct dispatch while handle " +
						"values carry fixed-width environment metadata."),
				},
				[]string{
					"go test ./compiler/tests/semantics -run 'Callable|Closure|FunctionType|Interface'",
					"go test ./compiler/internal/lower -run 'Callable|FunctionType'",
				},
				[]string{
					("aliases, struct fields, enum payloads, callback arguments, " +
						"and returns are covered only for safe by-value callable " +
						"captures"),
					"runtime generic callable polymorphism is not claimed",
					"dynamic callable dispatch is not claimed",
				},
				[]string{
					firstClassCallableFnPtrWitnessID,
					firstClassCallableHandleWitnessID,
					firstClassCallableInterfaceWitnessID,
				},
			),
		},
		NonClaims: p22FirstClassCallableNonClaims(),
	}
	return report, nil
}

func ValidateP22FirstClassCallableCoverage(report FirstClassCallableCoverageReport) error {
	if report.SchemaVersion != firstClassCallableCoverageSchemaV1 {
		return fmt.Errorf(
			"first-class callable coverage schema = %q, want %q",
			report.SchemaVersion,
			firstClassCallableCoverageSchemaV1,
		)
	}
	if report.Scope != firstClassCallableCoverageScopeP221 {
		return fmt.Errorf(
			"first-class callable coverage scope = %q, want %q",
			report.Scope,
			firstClassCallableCoverageScopeP221,
		)
	}
	if report.VariableABIWidthClaimed {
		return fmt.Errorf("first-class callable coverage: variable-width ABI claim is forbidden")
	}
	if report.ExplodingReturnSlotsClaimed {
		return fmt.Errorf(
			"first-class callable coverage: exploding return slots claim is forbidden",
		)
	}
	if report.MutableByRefCaptureClaimed {
		return fmt.Errorf(
			"first-class callable coverage: mutable by-reference capture claim is forbidden",
		)
	}
	if report.PointerResourceCaptureClaimed {
		return fmt.Errorf(
			"first-class callable coverage: pointer/resource capture claim is forbidden",
		)
	}
	if report.ThreadBoundaryCallableTransferClaimed {
		return fmt.Errorf(
			"first-class callable coverage: thread-boundary callable transfer claim is forbidden",
		)
	}
	if report.RuntimeGenericPolymorphismClaimed {
		return fmt.Errorf(
			"first-class callable coverage: runtime generic callable polymorphism claim is forbidden",
		)
	}
	if report.DynamicCallableDispatchClaimed {
		return fmt.Errorf(
			"first-class callable coverage: dynamic callable dispatch claim is forbidden",
		)
	}
	if report.UnsafeLifetimeRelaxationClaimed {
		return fmt.Errorf(
			"first-class callable coverage: unsafe lifetime relaxation claim is forbidden",
		)
	}
	if report.PerformanceClaimed {
		return fmt.Errorf("first-class callable coverage: performance claim is forbidden")
	}
	if report.RuntimeBehaviorChanged {
		return fmt.Errorf(
			"first-class callable coverage: runtime behavior change claim is forbidden",
		)
	}
	if report.SafeSemanticsChanged {
		return fmt.Errorf(
			"first-class callable coverage: safe-program semantics change is forbidden",
		)
	}
	for _, nonClaim := range p22FirstClassCallableNonClaims() {
		if !p22FirstClassCallableReportHasString(report.NonClaims, nonClaim) {
			return fmt.Errorf("first-class callable coverage: missing non-claim %q", nonClaim)
		}
	}
	if err := validateP22FirstClassCallableStrings("non-claim", report.NonClaims); err != nil {
		return err
	}

	witnesses := map[string]FirstClassCallableABIWitness{}
	for _, witness := range report.Witnesses {
		if strings.TrimSpace(witness.ID) == "" || strings.TrimSpace(witness.Kind) == "" {
			return fmt.Errorf(
				"first-class callable coverage: witness missing required metadata: %#v",
				witness,
			)
		}
		if _, ok := witnesses[witness.ID]; ok {
			return fmt.Errorf("first-class callable coverage: duplicate witness %s", witness.ID)
		}
		witnesses[witness.ID] = witness
	}
	for _, id := range []string{
		firstClassCallableFnPtrWitnessID,
		firstClassCallableHandleWitnessID,
		firstClassCallableInterfaceWitnessID,
	} {
		if _, ok := witnesses[id]; !ok {
			return fmt.Errorf("first-class callable coverage: missing witness %s", id)
		}
	}
	if err := validateP22FirstClassCallableFnPtrWitness(
		witnesses[firstClassCallableFnPtrWitnessID],
	); err != nil {
		return err
	}
	if err := validateP22FirstClassCallableHandleWitness(
		witnesses[firstClassCallableHandleWitnessID],
	); err != nil {
		return err
	}
	if err := validateP22FirstClassCallableInterfaceWitness(
		witnesses[firstClassCallableInterfaceWitnessID],
	); err != nil {
		return err
	}

	expected := map[FirstClassCallableCoverageID]bool{}
	for _, id := range p22FirstClassCallableCoverageIDs() {
		expected[id] = true
	}
	seen := map[FirstClassCallableCoverageID]bool{}
	for _, row := range report.Rows {
		if row.ID == "" || strings.TrimSpace(row.Name) == "" ||
			strings.TrimSpace(row.Status) == "" {
			return fmt.Errorf(
				"first-class callable coverage: row missing required metadata: %#v",
				row,
			)
		}
		if !expected[row.ID] {
			return fmt.Errorf("first-class callable coverage: unexpected row %s", row.ID)
		}
		if seen[row.ID] {
			return fmt.Errorf("first-class callable coverage: duplicate row %s", row.ID)
		}
		seen[row.ID] = true
		if err := validateP22FirstClassCallableStrings(
			"row "+string(row.ID)+" evidence",
			row.Evidence,
		); err != nil {
			return err
		}
		if err := validateP22FirstClassCallableStrings(
			"row "+string(row.ID)+" tests",
			row.Tests,
		); err != nil {
			return err
		}
		if err := validateP22FirstClassCallableStrings(
			"row "+string(row.ID)+" boundaries",
			row.Boundaries,
		); err != nil {
			return err
		}
		if len(row.WitnessIDs) == 0 {
			return fmt.Errorf(
				"first-class callable coverage: row %s missing witness reference",
				row.ID,
			)
		}
		for _, id := range row.WitnessIDs {
			if _, ok := witnesses[id]; !ok {
				return fmt.Errorf(
					"first-class callable coverage: row %s references missing witness %s",
					row.ID,
					id,
				)
			}
		}
	}
	for _, id := range p22FirstClassCallableCoverageIDs() {
		if !seen[id] {
			return fmt.Errorf("first-class callable coverage: missing row %s", id)
		}
	}
	return nil
}

func p22FirstClassCallableCoverageIDs() []FirstClassCallableCoverageID {
	return []FirstClassCallableCoverageID{
		FirstClassCallableFnPtrFastPath,
		FirstClassCallableFatHandle,
		FirstClassCallableCaptureSafetyClassifier,
		FirstClassCallableMutableCaptureDiagnostics,
		FirstClassCallableResourceThreadDiagnostics,
		FirstClassCallableFixedABIWidth,
		FirstClassCallableInterfaceMetadata,
		FirstClassCallableStorageCallbackPaths,
	}
}

func p22FirstClassCallableNonClaims() []string {
	return []string{
		"no variable-width callable ABI is claimed",
		"no exploding callable return slots are claimed",
		"no mutable by-reference capture support is claimed",
		"no pointer/resource capture support is claimed",
		"no thread-boundary callable transfer is claimed",
		"no runtime generic callable polymorphism is claimed",
		"no dynamic callable dispatch is claimed",
		"no unsafe lifetime relaxation is claimed",
		"no performance claim is made",
		"no runtime behavior change beyond the existing callable ABI is claimed",
		"safe-program semantics do not change",
	}
}

func p22FirstClassCallableRow(
	id FirstClassCallableCoverageID,
	name, status string,
	evidence, tests, boundaries, witnessIDs []string,
) FirstClassCallableCoverageRow {
	return FirstClassCallableCoverageRow{
		ID:         id,
		Name:       name,
		Status:     status,
		Evidence:   append([]string{}, evidence...),
		Tests:      append([]string{}, tests...),
		Boundaries: append([]string{}, boundaries...),
		WitnessIDs: append([]string{}, witnessIDs...),
	}
}

func buildP22FnPtrCallableWitness() (FirstClassCallableABIWitness, error) {
	checked, prog, err := p22ParseCheckLowerCallable(`
func main() -> Int:
    let base: Int = 1
    let cb: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + base
    return cb(41)
`)
	if err != nil {
		return FirstClassCallableABIWitness{}, err
	}
	main, ok := p22FindCheckedFunc(checked, "main")
	if !ok {
		return FirstClassCallableABIWitness{}, fmt.Errorf(
			"first-class callable coverage: fnptr witness missing checked main",
		)
	}
	cb := main.Locals["cb"]
	fn, ok := p22FindIRFunc(prog, "main")
	if !ok {
		return FirstClassCallableABIWitness{}, fmt.Errorf(
			"first-class callable coverage: fnptr witness missing lowered main",
		)
	}
	argSlots, retSlots := p22FirstMatchingCallSlots(fn, 2, 1)
	return FirstClassCallableABIWitness{
		ID:                      firstClassCallableFnPtrWitnessID,
		Kind:                    "fnptr_fast_path",
		CaptureCount:            p22CallableCaptureCount(cb),
		FnPtrSlotCount:          semantics.FnPtrSlotCount,
		CallableHandleSlotCount: semantics.CallableHandleSlotCount,
		LocalSlotCount:          cb.SlotCount,
		UsesHandle:              cb.FunctionHandleValue,
		AllocBytesCount:         p22CountIRKind(fn, ir.IRAllocBytes),
		EnvWriteCount:           p22CountIRKind(fn, ir.IRMemWritePtrOffset),
		EnvReadCount:            p22CountIRKind(fn, ir.IRMemReadPtrOffset),
		CallArgSlots:            argSlots,
		CallRetSlots:            retSlots,
		ReturnSlots:             semantics.FnPtrSlotCount,
		FunctionEscapeKind:      string(cb.FunctionEscapeKind),
	}, nil
}

func buildP22HandleCallableWitness() (FirstClassCallableABIWitness, error) {
	checked, prog, err := p22ParseCheckLowerCallable(`
func main() -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    let cb: fn(Int) -> Int = fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine
    return cb(-3)
`)
	if err != nil {
		return FirstClassCallableABIWitness{}, err
	}
	main, ok := p22FindCheckedFunc(checked, "main")
	if !ok {
		return FirstClassCallableABIWitness{}, fmt.Errorf(
			"first-class callable coverage: handle witness missing checked main",
		)
	}
	cb := main.Locals["cb"]
	fn, ok := p22FindIRFunc(prog, "main")
	if !ok {
		return FirstClassCallableABIWitness{}, fmt.Errorf(
			"first-class callable coverage: handle witness missing lowered main",
		)
	}
	argSlots, retSlots := p22FirstMatchingCallSlots(fn, 10, 1)
	return FirstClassCallableABIWitness{
		ID:                      firstClassCallableHandleWitnessID,
		Kind:                    "fat_callable_handle",
		CaptureCount:            p22CallableCaptureCount(cb),
		FnPtrSlotCount:          semantics.FnPtrSlotCount,
		CallableHandleSlotCount: semantics.CallableHandleSlotCount,
		LocalSlotCount:          cb.SlotCount,
		UsesHandle:              cb.FunctionHandleValue,
		AllocBytesCount:         p22CountIRKind(fn, ir.IRAllocBytes),
		EnvWriteCount:           p22CountIRKind(fn, ir.IRMemWritePtrOffset),
		EnvReadCount:            p22CountIRKind(fn, ir.IRMemReadPtrOffset),
		CallArgSlots:            argSlots,
		CallRetSlots:            retSlots,
		ReturnSlots:             semantics.CallableHandleSlotCount,
		FunctionEscapeKind:      string(cb.FunctionEscapeKind),
	}, nil
}

func buildP22InterfaceCallableWitness() (FirstClassCallableABIWitness, error) {
	src := []byte(`module lib.maker

pub func make() -> fn(Int) -> Int:
    let one: Int = 1
    let two: Int = 2
    let three: Int = 3
    let four: Int = 4
    let five: Int = 5
    let six: Int = 6
    let seven: Int = 7
    let eight: Int = 8
    let nine: Int = 9
    return fn(x: Int) -> Int:
        return x + one + two + three + four + five + six + seven + eight + nine
`)
	iface, err := GenerateInterfaceFromSource(src, "lib/maker.t4")
	if err != nil {
		return FirstClassCallableABIWitness{}, fmt.Errorf(
			"first-class callable coverage: generate interface witness: %w",
			err,
		)
	}
	maker, err := ParseFile(iface, "lib/maker.t4i")
	if err != nil {
		return FirstClassCallableABIWitness{}, fmt.Errorf(
			"first-class callable coverage: parse generated interface witness: %w",
			err,
		)
	}
	app, err := ParseFile([]byte(`module app.main
import lib.maker as maker

func main() -> Int:
    let cb: fn(Int) -> Int = maker.make()
    return cb(-3)
`), "app/main.t4")
	if err != nil {
		return FirstClassCallableABIWitness{}, fmt.Errorf(
			"first-class callable coverage: parse app witness: %w",
			err,
		)
	}
	checked, err := CheckWorld(&World{
		EntryModule:      "app.main",
		Files:            []*FileAST{maker, app},
		InterfaceModules: map[string]bool{"lib.maker": true},
		ByModule: map[string]*FileAST{
			"lib.maker": maker,
			"app.main":  app,
		},
	})
	if err != nil {
		return FirstClassCallableABIWitness{}, fmt.Errorf(
			"first-class callable coverage: check interface witness: %w",
			err,
		)
	}
	makeSig := checked.FuncSigs["lib.maker.make"]
	main, ok := p22FindCheckedFunc(checked, "app.main.main")
	if !ok {
		return FirstClassCallableABIWitness{}, fmt.Errorf(
			"first-class callable coverage: interface witness missing app.main.main",
		)
	}
	cb := main.Locals["cb"]
	return FirstClassCallableABIWitness{
		ID:                      firstClassCallableInterfaceWitnessID,
		Kind:                    "cross_module_interface_metadata",
		CaptureCount:            len(makeSig.ReturnFunctionCaptures),
		FnPtrSlotCount:          semantics.FnPtrSlotCount,
		CallableHandleSlotCount: semantics.CallableHandleSlotCount,
		LocalSlotCount:          cb.SlotCount,
		UsesHandle:              makeSig.ReturnFunctionHandleValue && cb.FunctionHandleValue,
		ReturnSlots:             makeSig.ReturnSlots,
		FunctionEscapeKind:      string(makeSig.ReturnFunctionEscapeKind),
		InterfaceMetadataPreserved: makeSig.ReturnFunctionSymbol != "" &&
			len(makeSig.ReturnFunctionCaptures) == 9 &&
			cb.FunctionHandleValue,
	}, nil
}

func p22ParseCheckLowerCallable(src string) (*CheckedProgram, *IRProgram, error) {
	prog, err := Parse([]byte(src))
	if err != nil {
		return nil, nil, fmt.Errorf("first-class callable coverage: parse witness: %w", err)
	}
	checked, err := Check(prog)
	if err != nil {
		return nil, nil, fmt.Errorf("first-class callable coverage: check witness: %w", err)
	}
	lowered, err := Lower(checked)
	if err != nil {
		return nil, nil, fmt.Errorf("first-class callable coverage: lower witness: %w", err)
	}
	return checked, lowered, nil
}

func p22FindCheckedFunc(checked *CheckedProgram, name string) (semantics.CheckedFunc, bool) {
	for _, fn := range checked.Funcs {
		if fn.Name == name {
			return fn, true
		}
	}
	return semantics.CheckedFunc{}, false
}

func p22FindIRFunc(prog *IRProgram, name string) (ir.IRFunc, bool) {
	for _, fn := range prog.Funcs {
		if fn.Name == name {
			return fn, true
		}
	}
	return ir.IRFunc{}, false
}

func p22CountIRKind(fn ir.IRFunc, kind ir.IRInstrKind) int {
	count := 0
	for _, instr := range fn.Instrs {
		if instr.Kind == kind {
			count++
		}
	}
	return count
}

func p22FirstMatchingCallSlots(fn ir.IRFunc, argSlots, retSlots int) (int, int) {
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRCall && instr.ArgSlots == argSlots && instr.RetSlots == retSlots {
			return instr.ArgSlots, instr.RetSlots
		}
	}
	return 0, 0
}

func p22CallableCaptureCount(local semantics.LocalInfo) int {
	return len(local.FunctionCaptures) + len(local.FunctionEscapeCaptures)
}

func validateP22FirstClassCallableFnPtrWitness(witness FirstClassCallableABIWitness) error {
	if witness.CaptureCount != 1 || witness.UsesHandle ||
		witness.FnPtrSlotCount != semantics.FnPtrSlotCount ||
		witness.LocalSlotCount != semantics.FnPtrSlotCount {
		return fmt.Errorf("first-class callable coverage: fnptr witness drift: %#v", witness)
	}
	if witness.CallableHandleSlotCount != semantics.CallableHandleSlotCount ||
		witness.AllocBytesCount != 0 ||
		witness.EnvWriteCount != 0 ||
		witness.EnvReadCount != 0 {
		return fmt.Errorf(
			"first-class callable coverage: fnptr witness allocated heap env or lost fixed ABI: %#v",
			witness,
		)
	}
	return nil
}

func validateP22FirstClassCallableHandleWitness(witness FirstClassCallableABIWitness) error {
	if witness.CaptureCount != 9 || !witness.UsesHandle ||
		witness.LocalSlotCount != semantics.CallableHandleSlotCount {
		return fmt.Errorf("first-class callable coverage: handle witness drift: %#v", witness)
	}
	if witness.FnPtrSlotCount != semantics.FnPtrSlotCount ||
		witness.CallableHandleSlotCount != semantics.CallableHandleSlotCount {
		return fmt.Errorf(
			"first-class callable coverage: fixed ABI drift in handle witness: %#v",
			witness,
		)
	}
	if witness.AllocBytesCount != 1 || witness.EnvWriteCount != 9 || witness.EnvReadCount != 9 ||
		witness.CallArgSlots != 10 ||
		witness.CallRetSlots != 1 {
		return fmt.Errorf("first-class callable coverage: handle witness IR drift: %#v", witness)
	}
	if witness.ReturnSlots != semantics.CallableHandleSlotCount ||
		witness.FunctionEscapeKind != string(semantics.CallableEscapeHeap) {
		return fmt.Errorf(
			"first-class callable coverage: handle witness escape/return metadata drift: %#v",
			witness,
		)
	}
	return nil
}

func validateP22FirstClassCallableInterfaceWitness(witness FirstClassCallableABIWitness) error {
	if witness.CaptureCount != 9 || !witness.UsesHandle || !witness.InterfaceMetadataPreserved {
		return fmt.Errorf("first-class callable coverage: interface witness drift: %#v", witness)
	}
	if witness.CallableHandleSlotCount != semantics.CallableHandleSlotCount ||
		witness.LocalSlotCount != semantics.CallableHandleSlotCount ||
		witness.ReturnSlots != semantics.CallableHandleSlotCount {
		return fmt.Errorf("first-class callable coverage: interface fixed ABI drift: %#v", witness)
	}
	if witness.FunctionEscapeKind != string(semantics.CallableEscapeHeap) {
		return fmt.Errorf(
			"first-class callable coverage: interface witness escape metadata drift: %#v",
			witness,
		)
	}
	return nil
}

func validateP22FirstClassCallableStrings(label string, items []string) error {
	if len(items) == 0 {
		return fmt.Errorf("first-class callable coverage: %s missing", label)
	}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			return fmt.Errorf("first-class callable coverage: %s contains empty item", label)
		}
		if p22FirstClassCallableContainsPlaceholder(trimmed) {
			return fmt.Errorf(
				"first-class callable coverage: %s contains placeholder evidence: %q",
				label,
				item,
			)
		}
	}
	return nil
}

func p22FirstClassCallableContainsPlaceholder(text string) bool {
	lower := strings.ToLower(text)
	for _, token := range []string{"todo", "tbd", "placeholder", "fixme", "???"} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func p22FirstClassCallableReportHasString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

// ---- formal_core_v1.go ----

const (
	formalCoreV1Schema    = "tetra.formal_core.v1"
	formalCoreV1ScopeP232 = "p23.2_formal_core_v1"

	p23FormalCoreSpecWitnessID       = "formal_core_spec_inventory"
	p23FormalCoreValuesWitnessID     = "stable_value_differential_subset"
	p23FormalCorePLIRWitnessID       = "plir_borrow_copy_provenance_regions"
	p23FormalCoreProofWitnessID      = "bounds_proof_check_elimination"
	p23FormalCoreAllocationWitnessID = "allocation_length_intent_lowering"
	p23FormalCoreRawPointerWitnessID = "raw_pointer_bounds_metadata"
)

type FormalCoreV1ID string

const (
	FormalCoreV1Values                   FormalCoreV1ID = "values"
	FormalCoreV1BorrowsOwnedCopy         FormalCoreV1ID = "borrows_owned_copy"
	FormalCoreV1ProvenanceRegions        FormalCoreV1ID = "provenance_regions"
	FormalCoreV1BoundsProofIDSemantics   FormalCoreV1ID = "bounds_proof_id_semantics"
	FormalCoreV1AllocationLengthContract FormalCoreV1ID = "allocation_length_contract"
	FormalCoreV1AllocationIntentLowering FormalCoreV1ID = "allocation_intent_lowering"
	FormalCoreV1RawPointerBoundsMetadata FormalCoreV1ID = "raw_pointer_bounds_metadata"
	FormalCoreV1CheckEliminationValidity FormalCoreV1ID = "check_elimination_validity"
)

type FormalCoreV1Report struct {
	SchemaVersion             string                `json:"schema_version"`
	Scope                     string                `json:"scope"`
	Rows                      []FormalCoreV1Row     `json:"rows"`
	Witnesses                 []FormalCoreV1Witness `json:"witnesses"`
	NonClaims                 []string              `json:"non_claims"`
	FormalSpecValid           bool                  `json:"formal_spec_valid"`
	FormalConcepts            int                   `json:"formal_concepts"`
	FormalRules               int                   `json:"formal_rules"`
	ValueSamples              int                   `json:"value_samples"`
	DifferentialLanes         int                   `json:"differential_lanes"`
	BorrowCopyFacts           bool                  `json:"borrow_copy_facts"`
	ProvenanceRegionFacts     bool                  `json:"provenance_region_facts"`
	BoundsProofIDsChecked     bool                  `json:"bounds_proof_ids_checked"`
	MissingProofRejected      bool                  `json:"missing_proof_rejected"`
	CheckEliminationValidated bool                  `json:"check_elimination_validated"`

	AllocationLengthContractsChecked bool `json:"allocation_length_contracts_checked"`

	InvalidAllocationLengthRejected bool `json:"invalid_allocation_length_rejected"`

	AllocationIntentLoweringValidated bool `json:"allocation_intent_lowering_validated"`

	AllocationIntentDriftRejected bool `json:"allocation_intent_drift_rejected"`
	RawPointerBoundsCases         int  `json:"raw_pointer_bounds_cases"`

	RawPointerImpossibleAddRejected bool `json:"raw_pointer_impossible_add_rejected"`

	RawPointerUnknownStayedChecked bool `json:"raw_pointer_unknown_stayed_checked"`
	FullFormalProofClaimed         bool `json:"full_formal_proof_claimed"`
	BroadLanguageProofClaimed      bool `json:"broad_language_proof_claimed"`
	UnsafePolicyChanged            bool `json:"unsafe_policy_changed"`
	RuntimeBehaviorChanged         bool `json:"runtime_behavior_changed"`
	SafeSemanticsChanged           bool `json:"safe_semantics_changed"`
	PerformanceClaimed             bool `json:"performance_claimed"`
}

type FormalCoreV1Row struct {
	ID         FormalCoreV1ID `json:"id"`
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	Evidence   []string       `json:"evidence"`
	Tests      []string       `json:"tests"`
	Boundaries []string       `json:"boundaries"`
	WitnessIDs []string       `json:"witness_ids"`
}

type FormalCoreV1Witness struct {
	ID                                string `json:"id"`
	Kind                              string `json:"kind"`
	FormalSpecValid                   bool   `json:"formal_spec_valid,omitempty"`
	FormalConcepts                    int    `json:"formal_concepts,omitempty"`
	FormalRules                       int    `json:"formal_rules,omitempty"`
	ValueSamples                      int    `json:"value_samples,omitempty"`
	DifferentialLanes                 int    `json:"differential_lanes,omitempty"`
	BorrowCopyFacts                   bool   `json:"borrow_copy_facts,omitempty"`
	ProvenanceRegionFacts             bool   `json:"provenance_region_facts,omitempty"`
	BoundsProofIDsChecked             bool   `json:"bounds_proof_ids_checked,omitempty"`
	MissingProofRejected              bool   `json:"missing_proof_rejected,omitempty"`
	CheckEliminationValidated         bool   `json:"check_elimination_validated,omitempty"`
	AllocationLengthContractsChecked  bool   `json:"allocation_length_contracts_checked,omitempty"`
	InvalidAllocationLengthRejected   bool   `json:"invalid_allocation_length_rejected,omitempty"`
	AllocationIntentLoweringValidated bool   `json:"allocation_intent_lowering_validated,omitempty"`
	AllocationIntentDriftRejected     bool   `json:"allocation_intent_drift_rejected,omitempty"`
	RawPointerBoundsCases             int    `json:"raw_pointer_bounds_cases,omitempty"`
	RawPointerImpossibleAddRejected   bool   `json:"raw_pointer_impossible_add_rejected,omitempty"`
	RawPointerUnknownStayedChecked    bool   `json:"raw_pointer_unknown_stayed_checked,omitempty"`
}

func BuildP23FormalCoreV1Report() (FormalCoreV1Report, error) {
	spec, err := buildP23FormalCoreSpecWitness()
	if err != nil {
		return FormalCoreV1Report{}, err
	}
	values, err := buildP23FormalCoreValuesWitness()
	if err != nil {
		return FormalCoreV1Report{}, err
	}
	plirWitness, err := buildP23FormalCorePLIRWitness()
	if err != nil {
		return FormalCoreV1Report{}, err
	}
	proof, err := buildP23FormalCoreProofWitness()
	if err != nil {
		return FormalCoreV1Report{}, err
	}
	allocation, err := buildP23FormalCoreAllocationWitness()
	if err != nil {
		return FormalCoreV1Report{}, err
	}
	raw, err := buildP23FormalCoreRawPointerWitness()
	if err != nil {
		return FormalCoreV1Report{}, err
	}

	report := FormalCoreV1Report{
		SchemaVersion: formalCoreV1Schema,
		Scope:         formalCoreV1ScopeP232,
		Witnesses: []FormalCoreV1Witness{
			spec,
			values,
			plirWitness,
			proof,
			allocation,
			raw,
		},
		Rows: []FormalCoreV1Row{
			p23FormalCoreRow(FormalCoreV1Values, "Values", "current_supported_subset",
				[]string{
					("differential.CheckBackendMatrix confirms stable observable " +
						"i32 values across supported source, Stack IR, optimized " +
						"Stack IR, SSA, and Machine IR lanes."),
					("The P23.2 value witness reuses the loop-sum IR sample so " +
						"values are checked by execution-equivalence evidence rather " +
						"than prose."),
				},
				[]string{
					"go test ./compiler -run 'P23FormalCoreV1'",
					"go test ./compiler/internal/differential -run 'CheckBackendMatrix'",
				},
				[]string{
					"values evidence is limited to the current supported scalar i32 subset",
					"no public source interpreter mode is introduced",
				},
				[]string{p23FormalCoreValuesWitnessID}),
			p23FormalCoreRow(
				FormalCoreV1BorrowsOwnedCopy,
				"Borrows and owned/copy",
				"current_supported_subset",
				[]string{
					("plir.VerifyProgram accepts a real window().borrow().copy() " +
						"program with borrowed_imm/no_escape facts for the borrow " +
						"and owned/provenance_known facts for the copy."),
					("Borrow/copy evidence comes from compiler.Parse, " +
						"compiler.Check, BuildPLIR, and PLIR fact inspection on " +
						"supported source."),
				},
				[]string{
					"go test ./compiler/internal/plir -run 'BorrowCopy|PreservesIslandView'",
					"go test ./compiler -run 'P23FormalCoreV1'",
				},
				[]string{
					"borrow/copy evidence is bounded to current PLIR source facts",
					"unsafe lifetime relaxation is not claimed",
				},
				[]string{p23FormalCorePLIRWitnessID},
			),
			p23FormalCoreRow(
				FormalCoreV1ProvenanceRegions,
				"Provenance and regions",
				"current_supported_subset",
				[]string{
					("PLIR records island provenance and explicit regions for " +
						"core.island_make_u8, derived window views, and borrowed " +
						"views."),
					("plir.VerifyProgram rejects contradictory provenance and " +
						"invalid region/borrow facts in nearby tests."),
				},
				[]string{
					"go test ./compiler/internal/plir -run 'Provenance|Region|PreservesIslandView'",
					"go test ./compiler -run 'P23FormalCoreV1'",
				},
				[]string{
					"region evidence is internal PLIR evidence, not a full region calculus",
					"external/unknown provenance remains conservative",
				},
				[]string{p23FormalCorePLIRWitnessID},
			),
			p23FormalCoreRow(
				FormalCoreV1BoundsProofIDSemantics,
				"Bounds proof id semantics",
				"current_supported_subset",
				[]string{
					("validation.CheckBoundsProofsWithPLIR accepts removed checks " +
						"only when the unchecked IR proof id exists in PLIR proof " +
						"guards."),
					("The proof witness rejects an unchecked load when the proof " +
						"id is missing, preserving proof id semantics."),
				},
				[]string{
					"go test ./compiler/internal/validation -run 'CheckBoundsProofsWithPLIR'",
					"go test ./compiler -run 'P23FormalCoreV1'",
				},
				[]string{
					"bounds proof evidence covers current proof-tagged removed checks",
					"no broad theorem prover is claimed",
				},
				[]string{p23FormalCoreProofWitnessID},
			),
			p23FormalCoreRow(
				FormalCoreV1AllocationLengthContract,
				"Allocation length contract",
				"current_supported_subset",
				[]string{
					("allocplan.FromPLIR classifies zero, normal, negative, and " +
						"overflow allocation length contract rows before storage " +
						"evidence is trusted."),
					("The allocation witness requires rejected_negative_length " +
						"and rejected_byte_size_overflow statuses from the real " +
						"PLIR-to-allocplan path."),
				},
				[]string{
					"go test ./compiler/internal/allocplan -run 'Length'",
					"go test ./compiler -run 'P23FormalCoreV1'",
				},
				[]string{
					"length contract evidence is planner evidence, not a platform build/run claim",
					"runtime behavior does not change",
				},
				[]string{p23FormalCoreAllocationWitnessID},
			),
			p23FormalCoreRow(
				FormalCoreV1AllocationIntentLowering,
				"Allocation intent lowering",
				"current_supported_subset",
				[]string{
					("validation.ValidateAllocationLowering validates allocation " +
						"intent rows against lowered IR stack and region allocation " +
						"operations."),
					("The allocation witness also rejects a drifted IR program " +
						"with missing matching stack allocation."),
				},
				[]string{
					"go test ./compiler/internal/validation -run 'ValidateAllocationLowering'",
					"go test ./compiler -run 'P23FormalCoreV1'",
				},
				[]string{
					"allocation intent evidence is bounded to current allocplan/lowering validators",
					"no broad allocation optimizer is claimed",
				},
				[]string{p23FormalCoreAllocationWitnessID},
			),
			p23FormalCoreRow(
				FormalCoreV1RawPointerBoundsMetadata,
				"Raw pointer bounds metadata",
				"current_supported_subset",
				[]string{
					("runtimeabi.NewRawAllocationBounds, DeriveRawPointerBounds, " +
						"and RawSliceBoundsFromParts cover raw pointer bounds " +
						"allocation-base metadata, derived offsets, checked " +
						"external/unknown metadata, and impossible ptr_add rejection."),
					("Unknown raw pointer provenance remains checked " +
						"external/unknown instead of forging an allocation root."),
				},
				[]string{
					"go test ./compiler/internal/runtimeabi -run 'RawPointerBounds'",
					"go test ./compiler -run 'P23FormalCoreV1'",
				},
				[]string{
					"raw pointer metadata is internal runtime ABI evidence",
					"unsafe policy does not change",
				},
				[]string{p23FormalCoreRawPointerWitnessID},
			),
			p23FormalCoreRow(
				FormalCoreV1CheckEliminationValidity,
				"Check-elimination validity",
				"current_supported_subset",
				[]string{
					("An unchecked lowered index operation is accepted only when " +
						"validation.CheckBoundsProofsWithPLIR can match its proof id " +
						"to PLIR proof guards."),
					("Missing proof ids are rejected, so check elimination stays " +
						"proof-bound and safe-semantics preserving."),
				},
				[]string{
					("go test ./compiler/internal/validation -run " +
						"'CheckBoundsProofsWithPLIR|ValidateTranslationRejectsProof'"),
					"go test ./compiler -run 'P23FormalCoreV1'",
				},
				[]string{
					"check-elimination validity is limited to current proof-tagged index operations",
					"safe-program semantics do not change",
				},
				[]string{p23FormalCoreProofWitnessID},
			),
		},
		NonClaims: []string{
			"no full formal proof of Tetra is claimed",
			"no broad language theorem prover is claimed",
			"no public source interpreter or backend selector is introduced",
			"unsafe policy does not change",
			"runtime behavior does not change",
			"safe-program semantics do not change",
			"no performance claim is made",
		},
		FormalSpecValid:                   spec.FormalSpecValid,
		FormalConcepts:                    spec.FormalConcepts,
		FormalRules:                       spec.FormalRules,
		ValueSamples:                      values.ValueSamples,
		DifferentialLanes:                 values.DifferentialLanes,
		BorrowCopyFacts:                   plirWitness.BorrowCopyFacts,
		ProvenanceRegionFacts:             plirWitness.ProvenanceRegionFacts,
		BoundsProofIDsChecked:             proof.BoundsProofIDsChecked,
		MissingProofRejected:              proof.MissingProofRejected,
		CheckEliminationValidated:         proof.CheckEliminationValidated,
		AllocationLengthContractsChecked:  allocation.AllocationLengthContractsChecked,
		InvalidAllocationLengthRejected:   allocation.InvalidAllocationLengthRejected,
		AllocationIntentLoweringValidated: allocation.AllocationIntentLoweringValidated,
		AllocationIntentDriftRejected:     allocation.AllocationIntentDriftRejected,
		RawPointerBoundsCases:             raw.RawPointerBoundsCases,
		RawPointerImpossibleAddRejected:   raw.RawPointerImpossibleAddRejected,
		RawPointerUnknownStayedChecked:    raw.RawPointerUnknownStayedChecked,
	}
	if err := ValidateP23FormalCoreV1Report(report); err != nil {
		return FormalCoreV1Report{}, err
	}
	return report, nil
}

func ValidateP23FormalCoreV1Report(report FormalCoreV1Report) error {
	if report.SchemaVersion != formalCoreV1Schema {
		return fmt.Errorf("formal core v1: schema_version is %q", report.SchemaVersion)
	}
	if report.Scope != formalCoreV1ScopeP232 {
		return fmt.Errorf("formal core v1: scope is %q", report.Scope)
	}
	if report.FullFormalProofClaimed {
		return fmt.Errorf("formal core v1: full formal proof claim is forbidden")
	}
	if report.BroadLanguageProofClaimed {
		return fmt.Errorf("formal core v1: broad language proof claim is forbidden")
	}
	if report.UnsafePolicyChanged {
		return fmt.Errorf("formal core v1: unsafe policy change claim is forbidden")
	}
	if report.RuntimeBehaviorChanged {
		return fmt.Errorf("formal core v1: runtime behavior change claim is forbidden")
	}
	if report.SafeSemanticsChanged {
		return fmt.Errorf("formal core v1: safe semantics change claim is forbidden")
	}
	if report.PerformanceClaimed {
		return fmt.Errorf("formal core v1: performance claim is forbidden")
	}
	if !report.FormalSpecValid || report.FormalConcepts < 9 || report.FormalRules < 7 {
		return fmt.Errorf("formal core v1: formal spec evidence incomplete")
	}
	if report.ValueSamples == 0 || report.DifferentialLanes < 5 {
		return fmt.Errorf("formal core v1: values evidence missing")
	}
	if !report.BorrowCopyFacts {
		return fmt.Errorf("formal core v1: borrow/copy facts missing")
	}
	if !report.ProvenanceRegionFacts {
		return fmt.Errorf("formal core v1: provenance/regions facts missing")
	}
	if !report.BoundsProofIDsChecked || !report.MissingProofRejected {
		return fmt.Errorf("formal core v1: bounds proof id evidence missing")
	}
	if !report.CheckEliminationValidated {
		return fmt.Errorf("formal core v1: check-elimination validity evidence missing")
	}
	if !report.AllocationLengthContractsChecked || !report.InvalidAllocationLengthRejected {
		return fmt.Errorf("formal core v1: allocation length contract evidence missing")
	}
	if !report.AllocationIntentLoweringValidated || !report.AllocationIntentDriftRejected {
		return fmt.Errorf("formal core v1: allocation intent lowering evidence missing")
	}
	if report.RawPointerBoundsCases < 4 || !report.RawPointerImpossibleAddRejected ||
		!report.RawPointerUnknownStayedChecked {
		return fmt.Errorf("formal core v1: raw pointer bounds metadata evidence missing")
	}
	for _, want := range []string{
		"no full formal proof of Tetra is claimed",
		"no broad language theorem prover is claimed",
		"unsafe policy does not change",
		"runtime behavior does not change",
		"safe-program semantics do not change",
		"no performance claim is made",
	} {
		if !p23FormalCoreHasString(report.NonClaims, want) {
			return fmt.Errorf("formal core v1: missing non-claim %q", want)
		}
	}
	if err := p23FormalCoreValidateRowsAndWitnesses(report.Rows, report.Witnesses); err != nil {
		return err
	}
	return nil
}

func buildP23FormalCoreSpecWitness() (FormalCoreV1Witness, error) {
	spec := formalcore.MinimumSpec()
	if err := formalcore.ValidateSpec(spec); err != nil {
		return FormalCoreV1Witness{}, err
	}
	return FormalCoreV1Witness{
		ID:              p23FormalCoreSpecWitnessID,
		Kind:            "formal_core_spec_inventory",
		FormalSpecValid: true,
		FormalConcepts:  len(spec.Concepts),
		FormalRules:     len(spec.Rules),
	}, nil
}

func buildP23FormalCoreValuesWitness() (FormalCoreV1Witness, error) {
	matrix, err := differential.CheckBackendMatrix(differential.BackendMatrixCase{
		Name:      "p23.2-formal-core-values-loop",
		Functions: []ir.IRFunc{p23LoopSumFunc()},
		Entry:     "sum_n",
		Samples: []differential.MatrixSample{
			{Name: "zero", Args: []int32{0}},
			{Name: "six", Args: []int32{6}},
		},
		Source: func(sample differential.MatrixSample) (int32, bool) {
			n := sample.Args[0]
			var total int32
			for i := int32(0); i < n; i++ {
				total += i
			}
			return total, true
		},
	})
	if err != nil {
		return FormalCoreV1Witness{}, err
	}
	return FormalCoreV1Witness{
		ID:                p23FormalCoreValuesWitnessID,
		Kind:              "stable_value_differential_subset",
		ValueSamples:      len(matrix.Samples),
		DifferentialLanes: len(matrix.Lanes),
	}, nil
}

func buildP23FormalCorePLIRWitness() (FormalCoreV1Witness, error) {
	src := []byte(`
func main() -> Int
uses alloc, islands, mem:
    island(64) as isl:
        var xs: []u8 = core.island_make_u8(isl, 4)
        let view: []u8 = xs.window(0, 2)
        let borrowed: []u8 = view.borrow()
        let copied: []u8 = borrowed.copy()
        return copied.len
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		return FormalCoreV1Witness{}, err
	}
	checked, err := Check(prog)
	if err != nil {
		return FormalCoreV1Witness{}, err
	}
	plirProg, err := BuildPLIR(checked)
	if err != nil {
		return FormalCoreV1Witness{}, err
	}
	if err := plir.VerifyProgram(plirProg); err != nil {
		return FormalCoreV1Witness{}, err
	}
	var borrow, owned, provenance, region bool
	for _, fn := range plirProg.Funcs {
		for _, value := range fn.Values {
			if value.Provenance.Kind == plir.ProvenanceIsland && value.Region != "" {
				provenance = true
				region = true
			}
			if value.Provenance.Kind == plir.ProvenanceAllocation &&
				value.Kind == plir.ValueAllocIntent {
				provenance = true
			}
		}
		for _, fact := range fn.Facts {
			switch fact.Kind {
			case plir.FactBorrowedImm:
				borrow = true
			case plir.FactOwned:
				owned = true
			case plir.FactRegionAlive:
				region = true
			case plir.FactProvenanceKnown:
				provenance = true
			}
		}
	}
	return FormalCoreV1Witness{
		ID:                    p23FormalCorePLIRWitnessID,
		Kind:                  "plir_borrow_copy_provenance_regions",
		BorrowCopyFacts:       borrow && owned,
		ProvenanceRegionFacts: provenance && region,
	}, nil
}

func buildP23FormalCoreProofWitness() (FormalCoreV1Witness, error) {
	proofID := "proof:while:i:xs:1:1"
	report, err := validation.CheckBoundsProofsWithPLIR(
		p23ProofProgram("main", proofID),
		p23FormalCoreProofPLIR(proofID),
	)
	if err != nil {
		return FormalCoreV1Witness{}, err
	}
	_, badErr := validation.CheckBoundsProofsWithPLIR(
		p23ProofProgram("main", ""),
		p23FormalCoreProofPLIR(proofID),
	)
	return FormalCoreV1Witness{
		ID:                    p23FormalCoreProofWitnessID,
		Kind:                  "bounds_proof_check_elimination",
		BoundsProofIDsChecked: len(report.RemovedChecks) > 0,
		MissingProofRejected: badErr != nil &&
			strings.Contains(badErr.Error(), "without proof id"),
		CheckEliminationValidated: len(report.RemovedChecks) > 0,
	}, nil
}

func buildP23FormalCoreAllocationWitness() (FormalCoreV1Witness, error) {
	src := []byte(`
func empty() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(0)
    return xs.len

func normal() -> Int
uses alloc, mem:
    var xs: []u16 = make_u16(3)
    return xs.len

func negative() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(0 - 1)
    return xs.len

func overflow() -> Int
uses alloc, mem:
    var xs: []bool = make_bool(536870912)
    return xs.len

func main() -> Int
uses alloc, mem:
    return 0
`)
	prog, err := Parse(src)
	if err != nil {
		return FormalCoreV1Witness{}, err
	}
	checked, err := Check(prog)
	if err != nil {
		return FormalCoreV1Witness{}, err
	}
	plirProg, err := BuildPLIR(checked)
	if err != nil {
		return FormalCoreV1Witness{}, err
	}
	plan, err := allocplan.FromPLIR(plirProg)
	if err != nil {
		return FormalCoreV1Witness{}, err
	}
	var empty, normal, negative, overflow bool
	for _, fn := range plan.Functions {
		for _, allocation := range fn.Allocations {
			switch allocation.LengthStatus {
			case allocplan.LengthStatusValidEmpty:
				empty = true
			case allocplan.LengthStatusNormal:
				normal = true
			case allocplan.LengthStatusRejectedNegative:
				negative = true
			case allocplan.LengthStatusRejectedOverflow:
				overflow = true
			}
		}
	}
	allocationWitness, err := buildP23AllocationWitness()
	if err != nil {
		return FormalCoreV1Witness{}, err
	}
	return FormalCoreV1Witness{
		ID:                                p23FormalCoreAllocationWitnessID,
		Kind:                              "allocation_length_intent_lowering",
		AllocationLengthContractsChecked:  empty && normal,
		InvalidAllocationLengthRejected:   negative && overflow,
		AllocationIntentLoweringValidated: allocationWitness.AllocationPlanValidated,
		AllocationIntentDriftRejected:     allocationWitness.AllocationDriftRejected,
	}, nil
}

func buildP23FormalCoreRawPointerWitness() (FormalCoreV1Witness, error) {
	root, err := runtimeabi.NewRawAllocationBounds("p23.2-root", 16)
	if err != nil {
		return FormalCoreV1Witness{}, err
	}
	derived, diag := runtimeabi.DeriveRawPointerBounds(root, 4, 4)
	if diag != nil {
		return FormalCoreV1Witness{}, fmt.Errorf(
			"formal core v1: unexpected raw pointer diagnostic: %+v",
			diag,
		)
	}
	rejected, rejectedDiag := runtimeabi.DeriveRawPointerBounds(root, 16, 1)
	unknown := runtimeabi.UnknownRawPointerBounds("ffi pointer")
	unknownDerived, unknownDiag := runtimeabi.DeriveRawPointerBounds(unknown, 4, 1)
	if unknownDiag != nil {
		return FormalCoreV1Witness{}, fmt.Errorf(
			"formal core v1: unknown raw pointer returned diagnostic: %+v",
			unknownDiag,
		)
	}
	verifiedSlice := runtimeabi.RawSliceBoundsFromParts(derived, 2, 4)
	unknownSlice := runtimeabi.RawSliceBoundsFromParts(unknownDerived, 2, 4)
	checkedExternal := runtimeabi.RawPointerBoundsCheckedExternalUnknown
	unknownDerivedStatus := unknownDerived.Status
	unknownDerivedChecked := unknownDerivedStatus == checkedExternal &&
		!unknownDerived.VerifiedAllocationRoot
	verifiedSliceChecked := verifiedSlice.Status == runtimeabi.RawSliceBoundsVerifiedAllocationRoot &&
		verifiedSlice.VerifiedAllocationRoot
	unknownSliceChecked := unknownSlice.Status == runtimeabi.RawSliceBoundsExternalUnknown &&
		!unknownSlice.VerifiedAllocationRoot
	cases := 0
	for _, ok := range []bool{
		root.Status == runtimeabi.RawPointerBoundsAllocationBase && root.VerifiedAllocationRoot,
		derived.Status == runtimeabi.RawPointerBoundsDerivedOffset && derived.VerifiedAllocationRoot,
		rejected.Status == runtimeabi.RawPointerBoundsRejected && rejectedDiag != nil,
		unknownDerivedChecked,
		verifiedSliceChecked,
		unknownSliceChecked,
	} {
		if ok {
			cases++
		}
	}
	return FormalCoreV1Witness{
		ID:                    p23FormalCoreRawPointerWitnessID,
		Kind:                  "raw_pointer_bounds_metadata",
		RawPointerBoundsCases: cases,
		RawPointerImpossibleAddRejected: rejected.Status == runtimeabi.RawPointerBoundsRejected &&
			rejectedDiag != nil,
		RawPointerUnknownStayedChecked: unknownDerivedChecked &&
			unknownSliceChecked,
	}, nil
}

func p23FormalCoreProofPLIR(proofID string) *plir.Program {
	return &plir.Program{Funcs: []plir.Function{{
		Name: "main",
		Values: []plir.Value{
			{
				ID:         "param:xs",
				Kind:       plir.ValueParam,
				Type:       "[]i32",
				Region:     "fn:main",
				Provenance: plir.Provenance{Kind: plir.ProvenanceParam, Root: "xs"},
				Lifetime:   plir.Lifetime{Birth: "entry", Death: "return", Owner: "xs"},
				Borrow:     plir.BorrowImm,
				Escape:     plir.EscapeNoEscape,
			},
			{
				ID:         "local:i",
				Kind:       plir.ValueLocal,
				Type:       "i32",
				Region:     "fn:main",
				Provenance: plir.Provenance{Kind: plir.ProvenanceStack, Root: "i"},
				Lifetime:   plir.Lifetime{Birth: "entry", Death: "return", Owner: "i"},
				Borrow:     plir.BorrowNone,
				Escape:     plir.EscapeNoEscape,
			},
		},
		Blocks: []plir.BasicBlock{
			{ID: "entry", Kind: "entry", Entry: true, Succs: []string{"body"}},
			{
				ID:    "body",
				Kind:  "while_body",
				Preds: []string{"entry"},
				Ops:   []string{"op0"},
				Exit:  true,
			},
		},
		Ops: []plir.Operation{
			{ID: "op0", Kind: plir.OpIndexLoad, Block: "body"},
		},
		Facts: []plir.Fact{
			{ID: "known", Kind: plir.FactProvenanceKnown, ValueID: "param:xs"},
			{ID: "len", Kind: plir.FactLenStable, ValueID: "param:xs"},
			{
				ID:      "range",
				Kind:    plir.FactIndexInRange,
				ValueID: "local:i",
				Range:   "0..xs.len",
				ProofID: proofID,
				Source:  "formal-core:1:1",
			},
		},
		ProofGuards: []plir.ProofGuard{{
			ID:        proofID,
			Kind:      "range",
			Block:     "body",
			OpID:      "op0",
			Condition: "0 <= i < xs.len",
			Reason:    "formal core proof witness",
		}},
		ProofUses: []plir.ProofUse{{
			ProofID: proofID,
			Block:   "body",
			OpID:    "op0",
			UseKind: "bounds_check",
			Source:  "formal-core:1:1",
		}},
		ProofTerms: []plir.ProofTerm{{
			ID:            proofID,
			Kind:          "bounds_check",
			SubjectBaseID: "xs",
			IndexValueID:  "local:i",
			Operation:     "index_load",
			Range:         "0..xs.len",
			Source:        "formal-core:1:1",
			FactsUsed:     []string{"range"},
		}},
	}}}
}

func p23FormalCoreValidateRowsAndWitnesses(
	rows []FormalCoreV1Row,
	witnesses []FormalCoreV1Witness,
) error {
	witnessIDs := map[string]bool{}
	for _, witness := range witnesses {
		if strings.TrimSpace(witness.ID) == "" {
			return fmt.Errorf("formal core v1: witness missing id")
		}
		witnessIDs[witness.ID] = true
	}
	seen := map[FormalCoreV1ID]bool{}
	expected := map[FormalCoreV1ID]bool{}
	for _, id := range p23FormalCoreV1IDs() {
		expected[id] = true
	}
	for _, row := range rows {
		if row.ID == "" || row.Name == "" || row.Status == "" || len(row.Evidence) == 0 ||
			len(row.Tests) == 0 ||
			len(row.Boundaries) == 0 ||
			len(row.WitnessIDs) == 0 {
			return fmt.Errorf("formal core v1: row %q missing required metadata", row.ID)
		}
		if !expected[row.ID] {
			return fmt.Errorf("formal core v1: unexpected row %s", row.ID)
		}
		if seen[row.ID] {
			return fmt.Errorf("formal core v1: duplicate row %s", row.ID)
		}
		seen[row.ID] = true
		if p23ContainsPlaceholder(row.Evidence) || p23ContainsPlaceholder(row.Boundaries) {
			return fmt.Errorf("formal core v1: row %s contains placeholder evidence", row.ID)
		}
		for _, witnessID := range row.WitnessIDs {
			if !witnessIDs[witnessID] {
				return fmt.Errorf(
					"formal core v1: row %s references missing witness %q",
					row.ID,
					witnessID,
				)
			}
		}
	}
	for _, id := range p23FormalCoreV1IDs() {
		if !seen[id] {
			return fmt.Errorf("formal core v1: missing row %s", id)
		}
	}
	return nil
}

func p23FormalCoreV1IDs() []FormalCoreV1ID {
	return []FormalCoreV1ID{
		FormalCoreV1Values,
		FormalCoreV1BorrowsOwnedCopy,
		FormalCoreV1ProvenanceRegions,
		FormalCoreV1BoundsProofIDSemantics,
		FormalCoreV1AllocationLengthContract,
		FormalCoreV1AllocationIntentLowering,
		FormalCoreV1RawPointerBoundsMetadata,
		FormalCoreV1CheckEliminationValidity,
	}
}

func p23FormalCoreRow(
	id FormalCoreV1ID,
	name string,
	status string,
	evidence []string,
	tests []string,
	boundaries []string,
	witnessIDs []string,
) FormalCoreV1Row {
	return FormalCoreV1Row{
		ID:         id,
		Name:       name,
		Status:     status,
		Evidence:   append([]string(nil), evidence...),
		Tests:      append([]string(nil), tests...),
		Boundaries: append([]string(nil), boundaries...),
		WitnessIDs: append([]string(nil), witnessIDs...),
	}
}

func p23FormalCoreHasString(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

// ---- fuzz_property_differential_v1.go ----

const (
	fuzzPropertyDifferentialSchema    = "tetra.fuzz.property.differential.v1"
	fuzzPropertyDifferentialScopeP231 = "p23.1_fuzz_property_differential"

	p23FuzzGeneratedPipelineWitnessID = "generated_parser_checker_plir_lowering"
	p23FuzzBackendMatrixWitnessID     = "backend_matrix_randomized"
	p23FuzzNativeBackendWitnessID     = "native_backend_boundary"
	p23FuzzAllocatorWitnessID         = "runtime_allocator_properties"
	p23FuzzActorTransferWitnessID     = "actor_transfer_stress_boundary"
	p23FuzzSummaryGateWitnessID       = "fuzz_nightly_summary_gate"
	p23FuzzReducerWitnessID           = "reducer_failure_artifact"
)

type FuzzPropertyDifferentialID string
type fuzzDifferentialID = FuzzPropertyDifferentialID

const (
	FuzzPropertyDifferentialParserCheckerGeneratedPrograms = fuzzDifferentialID(
		"parser_checker_generated_programs",
	)
	FuzzPropertyDifferentialPLIRLoweringVerifierPipeline = fuzzDifferentialID(
		"plir_lowering_verifier_pipeline",
	)
	FuzzPropertyDifferentialBackendMatrixExpansion = fuzzDifferentialID(
		"backend_differential_matrix_expansion",
	)
	FuzzPropertyDifferentialNativeBackendBoundary = fuzzDifferentialID(
		"native_backend_boundary",
	)
	FuzzPropertyDifferentialRuntimeAllocatorProperties = fuzzDifferentialID(
		"runtime_allocator_properties",
	)
	FuzzPropertyDifferentialActorTransferStressBoundary = fuzzDifferentialID(
		"actor_transfer_stress_boundary",
	)
	FuzzPropertyDifferentialFuzzNightlySummaryGate = fuzzDifferentialID(
		"fuzz_nightly_summary_gate",
	)
	FuzzPropertyDifferentialReducerFailureArtifacts = fuzzDifferentialID(
		"reducer_failure_artifacts",
	)
)

type FuzzPropertyDifferentialReport struct {
	SchemaVersion string                            `json:"schema_version"`
	Scope         string                            `json:"scope"`
	Rows          []FuzzPropertyDifferentialRow     `json:"rows"`
	Witnesses     []FuzzPropertyDifferentialWitness `json:"witnesses"`
	NonClaims     []string                          `json:"non_claims"`

	ParserCheckerGeneratedPrograms int `json:"parser_checker_generated_programs"`

	PLIRVerifierCases     int `json:"plir_verifier_cases"`
	LoweringVerifierCases int `json:"lowering_verifier_cases"`
	BackendMatrixCases    int `json:"backend_matrix_cases"`

	BackendMatrixRandomizedSamples int `json:"backend_matrix_randomized_samples"`

	BackendMatrixReducerRecorded bool `json:"backend_matrix_reducer_recorded"`

	NativeBackendHostSupported bool `json:"native_backend_host_supported"`

	NativeBackendSamples int `json:"native_backend_samples"`

	NativeBackendUnavailableReason string `json:"native_backend_unavailable_reason,omitempty"`

	RuntimeAllocatorPropertyCases int `json:"runtime_allocator_property_cases"`

	RuntimeAllocatorRejectsInvalid bool `json:"runtime_allocator_rejects_invalid"`

	ActorTransferStressDiagnostics bool `json:"actor_transfer_stress_diagnostics"`

	FuzzSummaryGateArtifacts int `json:"fuzz_summary_gate_artifacts"`

	NightlyLongFuzzBoundaryRecorded bool `json:"nightly_long_fuzz_boundary_recorded"`

	FullCorrectnessClaimed bool `json:"full_correctness_claimed"`

	ExhaustiveFuzzingClaimed bool `json:"exhaustive_fuzzing_claimed"`

	FullNativeDifferentialClaimed bool `json:"full_native_differential_claimed"`

	PerformanceClaimed     bool `json:"performance_claimed"`
	RuntimeBehaviorChanged bool `json:"runtime_behavior_changed"`
	SafeSemanticsChanged   bool `json:"safe_semantics_changed"`
}

type FuzzPropertyDifferentialRow struct {
	ID         FuzzPropertyDifferentialID `json:"id"`
	Name       string                     `json:"name"`
	Status     string                     `json:"status"`
	Evidence   []string                   `json:"evidence"`
	Tests      []string                   `json:"tests"`
	Boundaries []string                   `json:"boundaries"`
	WitnessIDs []string                   `json:"witness_ids"`
}

type FuzzPropertyDifferentialWitness struct {
	ID                              string `json:"id"`
	Kind                            string `json:"kind"`
	GeneratedPrograms               int    `json:"generated_programs,omitempty"`
	ParserCheckerCases              int    `json:"parser_checker_cases,omitempty"`
	PLIRVerifierCases               int    `json:"plir_verifier_cases,omitempty"`
	LoweringVerifierCases           int    `json:"lowering_verifier_cases,omitempty"`
	BackendMatrixCases              int    `json:"backend_matrix_cases,omitempty"`
	RandomizedSamples               int    `json:"randomized_samples,omitempty"`
	ReducerRecorded                 bool   `json:"reducer_recorded,omitempty"`
	NativeHostSupported             bool   `json:"native_host_supported,omitempty"`
	NativeSamples                   int    `json:"native_samples,omitempty"`
	NativeUnavailableReason         string `json:"native_unavailable_reason,omitempty"`
	RuntimeAllocatorPropertyCases   int    `json:"runtime_allocator_property_cases,omitempty"`
	RuntimeAllocatorRejectsInvalid  bool   `json:"runtime_allocator_rejects_invalid,omitempty"`
	ActorTransferStressDiagnostics  bool   `json:"actor_transfer_stress_diagnostics,omitempty"`
	ActorTransferPLIRMovedFacts     bool   `json:"actor_transfer_plir_moved_facts,omitempty"`
	FuzzSummaryGateArtifacts        int    `json:"fuzz_summary_gate_artifacts,omitempty"`
	NightlyLongFuzzBoundaryRecorded bool   `json:"nightly_long_fuzz_boundary_recorded,omitempty"`
	ReducedSingleSampleReproducer   bool   `json:"reduced_single_sample_reproducer,omitempty"`
}

func BuildP23FuzzPropertyDifferentialReport() (FuzzPropertyDifferentialReport, error) {
	generated, err := buildP23FuzzGeneratedPipelineWitness()
	if err != nil {
		return FuzzPropertyDifferentialReport{}, err
	}
	backend, err := buildP23FuzzBackendMatrixWitness()
	if err != nil {
		return FuzzPropertyDifferentialReport{}, err
	}
	native, err := buildP23FuzzNativeBackendWitness()
	if err != nil {
		return FuzzPropertyDifferentialReport{}, err
	}
	allocator, err := buildP23FuzzAllocatorWitness()
	if err != nil {
		return FuzzPropertyDifferentialReport{}, err
	}
	actor, err := buildP23FuzzActorTransferWitness()
	if err != nil {
		return FuzzPropertyDifferentialReport{}, err
	}
	summary := buildP23FuzzSummaryGateWitness()
	reducer, err := buildP23FuzzReducerWitness()
	if err != nil {
		return FuzzPropertyDifferentialReport{}, err
	}

	report := FuzzPropertyDifferentialReport{
		SchemaVersion: fuzzPropertyDifferentialSchema,
		Scope:         fuzzPropertyDifferentialScopeP231,
		Witnesses: []FuzzPropertyDifferentialWitness{
			generated,
			backend,
			native,
			allocator,
			actor,
			summary,
			reducer,
		},
		Rows: []FuzzPropertyDifferentialRow{
			p23FuzzRow(
				FuzzPropertyDifferentialParserCheckerGeneratedPrograms,
				"Parser/checker generated programs",
				"current_supported_subset",
				[]string{
					("P23.1 generated source witness builds deterministic " +
						"generated source snippets and runs compiler.Parse plus " +
						"compiler.Check on every case."),
					("compiler/tests/fuzz/FuzzLoweringPipelineVerifiesIR already " +
						"fuzzes generated parser/checker/lowerer inputs with Go fuzz " +
						"seeds."),
				},
				[]string{
					"go test ./compiler -run 'P23FuzzPropertyDifferential'",
					("go test ./compiler/tests/fuzz -run " +
						"'FuzzLoweringPipelineVerifiesIR|FuzzFormatSourceIdempotent' " +
						"-count=1"),
				},
				[]string{
					"generated source is bounded to deterministic scalar/control-flow snippets in this report",
					"Go fuzz targets provide broader seed mutation outside this report API",
					"no exhaustive parser/checker correctness claim is made",
				},
				[]string{p23FuzzGeneratedPipelineWitnessID},
			),
			p23FuzzRow(
				FuzzPropertyDifferentialPLIRLoweringVerifierPipeline,
				"PLIR/lowering verifier pipeline",
				"current_supported_subset",
				[]string{
					("The generated pipeline witness runs compiler.BuildPLIR, " +
						"compiler.Lower, and compiler.VerifyIRProgram on the same " +
						"generated source cases."),
					("compiler/internal/lower runs PLIR verification before Stack " +
						"IR lowering; the public BuildPLIR API keeps PLIR evidence " +
						"inspectable."),
				},
				[]string{
					"go test ./compiler -run 'P23FuzzPropertyDifferential'",
					"go test ./compiler/internal/plir -count=1",
				},
				[]string{
					("PLIR/lowering evidence is bounded to supported generated " +
						"snippets and existing PLIR verifier coverage"),
					"unsupported syntax is not trusted as passing evidence",
				},
				[]string{p23FuzzGeneratedPipelineWitnessID},
			),
			p23FuzzRow(
				FuzzPropertyDifferentialBackendMatrixExpansion,
				"Backend differential matrix expansion",
				"current_supported_subset",
				[]string{
					("differential.CheckBackendMatrix compares source, Stack IR, " +
						"optimized Stack IR, SSA, and Machine IR lanes for supported " +
						"i32 rows."),
					("The P23.1 backend witness records randomized deterministic " +
						"samples through RandomSeed and RandomSampleCount."),
					("docs/audits/compiler/backend/backend-differential-validation" +
						"-v1.md records existing scalar, branch/loop, call-loop, " +
						"slice, randomized, and reducer coverage."),
				},
				[]string{
					"go test ./compiler/internal/differential -run 'CheckBackendMatrix' -count=1",
					"go test ./compiler -run 'P23FuzzPropertyDifferential'",
				},
				[]string{
					"backend matrix coverage is limited to the supported i32 stable subset",
					"no full source interpreter or full native differential suite is claimed",
				},
				[]string{p23FuzzBackendMatrixWitnessID},
			),
			p23FuzzRow(
				FuzzPropertyDifferentialNativeBackendBoundary,
				"Native backend boundary",
				"current_supported_subset",
				[]string{
					("Host-supported native backend witness compares Linux x64 " +
						"native backend exit results against source/Stack " +
						"IR/SSA/Machine IR lanes when the current host is " +
						"linux/amd64."),
					("Non-linux/amd64 hosts record an explicit unavailable " +
						"boundary instead of silently claiming native backend " +
						"coverage."),
				},
				[]string{
					"go test ./compiler/internal/differential -run 'NativeLanes|CheckBackendMatrix' -count=1",
					"go test ./compiler -run 'P23FuzzPropertyDifferential'",
				},
				[]string{
					"native backend evidence is Linux x64 host-bound",
					"other hosts keep an explicit unavailable boundary",
					"no full native differential suite is claimed",
				},
				[]string{p23FuzzNativeBackendWitnessID},
			),
			p23FuzzRow(
				FuzzPropertyDifferentialRuntimeAllocatorProperties,
				"Runtime allocator properties",
				"current_supported_subset",
				[]string{
					("runtimeabi.AlignRegionBytes accepts valid region sizes with " +
						"16-byte alignment and rejects negative and overflow-sized " +
						"inputs."),
					("RuntimeRegionAllocatorConfig records the bounded region " +
						"allocator payload/header contract used by allocation " +
						"evidence."),
				},
				[]string{
					"go test ./compiler/internal/runtimeabi -run 'RegionAllocator|AlignRegionBytes' -count=1",
					"go test ./compiler -run 'P23FuzzPropertyDifferential'",
				},
				[]string{
					("allocator properties cover deterministic region ABI " +
						"arithmetic, not a full allocator stress campaign"),
					"runtime behavior does not change",
				},
				[]string{p23FuzzAllocatorWitnessID},
			),
			p23FuzzRow(
				FuzzPropertyDifferentialActorTransferStressBoundary,
				"Actor transfer stress boundary",
				"current_supported_subset",
				[]string{
					("actorsafety.TypedActorOwnershipTransferCoverage validates " +
						"stress diagnostics and PLIR moved facts for direct " +
						"core.send_typed ownership transfers."),
					("The actor witness requires stress diagnostics plus " +
						"FactMoved/OpActorSend evidence without promoting " +
						"distributed zero-copy."),
				},
				[]string{
					"go test ./compiler/internal/actorsafety -run 'TypedActorOwnershipTransfer' -count=1",
					"go test ./compiler -run 'P23FuzzPropertyDifferential'",
				},
				[]string{
					"actor transfer evidence is bounded to existing typed actor ownership transfer coverage",
					"distributed pointer or region zero-copy is not claimed",
				},
				[]string{p23FuzzActorTransferWitnessID},
			),
			p23FuzzRow(
				FuzzPropertyDifferentialFuzzNightlySummaryGate,
				"Fuzz nightly summary gate",
				"current_supported_subset",
				[]string{
					("scripts/dev/fuzz-nightly.sh runs bounded " +
						"fuzz/property/stress commands one package at a time and " +
						"writes summary.md, summary.json, crasher-inventory.json, " +
						"unstable-seeds.md, and per-step logs."),
					("tools/cmd/validate-fuzz-summary validates required report " +
						"artifacts, pass status, expected commands, logs, and " +
						"unstable-seeds table shape."),
					("docs/testing/fuzz_property_stress.md documents short and " +
						"nightly commands plus deterministic regression triage."),
				},
				[]string{
					"bash scripts/dev/fuzz-nightly.sh --short --fuzztime 1s --out-dir reports/fuzz-nightly-smoke",
					"go run ./tools/cmd/validate-fuzz-summary --report-dir reports/fuzz-nightly-smoke",
				},
				[]string{
					"nightly long fuzz is a separate bounded gate and not implied by the report API",
					"unstable seeds require deterministic regression or explicit owner/rerun evidence",
				},
				[]string{p23FuzzSummaryGateWitnessID},
			),
			p23FuzzRow(
				FuzzPropertyDifferentialReducerFailureArtifacts,
				"Reducer failure artifacts",
				"current_supported_subset",
				[]string{
					("differential.CheckBackendMatrix records " +
						"reduced_to_single_sample reducer metadata and a reproducer " +
						"string on first mismatch."),
					("The P23.1 reducer witness intentionally runs a bad source " +
						"oracle and requires a reduced single-sample reproducer."),
				},
				[]string{
					"go test ./compiler/internal/differential -run 'Reducer' -count=1",
					"go test ./compiler -run 'P23FuzzPropertyDifferential'",
				},
				[]string{
					("reducer evidence is first-mismatch single-sample metadata, " +
						"not a general-purpose program reducer"),
					"failing fuzz seeds still require deterministic regression tests before promotion",
				},
				[]string{p23FuzzReducerWitnessID},
			),
		},
		NonClaims: []string{
			"no full program correctness claim is made",
			"no exhaustive fuzzing is claimed",
			"no full native differential suite is claimed",
			"no broad random program generator beyond bounded snippets is claimed",
			"no performance claim is made",
			"runtime behavior does not change",
			"safe-program semantics do not change",
		},
		ParserCheckerGeneratedPrograms:  generated.GeneratedPrograms,
		PLIRVerifierCases:               generated.PLIRVerifierCases,
		LoweringVerifierCases:           generated.LoweringVerifierCases,
		BackendMatrixCases:              backend.BackendMatrixCases,
		BackendMatrixRandomizedSamples:  backend.RandomizedSamples,
		BackendMatrixReducerRecorded:    reducer.ReducedSingleSampleReproducer,
		NativeBackendHostSupported:      native.NativeHostSupported,
		NativeBackendSamples:            native.NativeSamples,
		NativeBackendUnavailableReason:  native.NativeUnavailableReason,
		RuntimeAllocatorPropertyCases:   allocator.RuntimeAllocatorPropertyCases,
		RuntimeAllocatorRejectsInvalid:  allocator.RuntimeAllocatorRejectsInvalid,
		ActorTransferStressDiagnostics:  actor.ActorTransferStressDiagnostics,
		FuzzSummaryGateArtifacts:        summary.FuzzSummaryGateArtifacts,
		NightlyLongFuzzBoundaryRecorded: summary.NightlyLongFuzzBoundaryRecorded,
	}
	if err := ValidateP23FuzzPropertyDifferentialReport(report); err != nil {
		return FuzzPropertyDifferentialReport{}, err
	}
	return report, nil
}

func ValidateP23FuzzPropertyDifferentialReport(report FuzzPropertyDifferentialReport) error {
	if report.SchemaVersion != fuzzPropertyDifferentialSchema {
		return fmt.Errorf(
			"fuzz/property/differential v1: schema_version is %q",
			report.SchemaVersion,
		)
	}
	if report.Scope != fuzzPropertyDifferentialScopeP231 {
		return fmt.Errorf("fuzz/property/differential v1: scope is %q", report.Scope)
	}
	if report.FullCorrectnessClaimed {
		return fmt.Errorf(
			"fuzz/property/differential v1: full program correctness claim is forbidden",
		)
	}
	if report.ExhaustiveFuzzingClaimed {
		return fmt.Errorf("fuzz/property/differential v1: exhaustive fuzzing claim is forbidden")
	}
	if report.FullNativeDifferentialClaimed {
		return fmt.Errorf(
			"fuzz/property/differential v1: full native differential claim is forbidden",
		)
	}
	if report.PerformanceClaimed {
		return fmt.Errorf("fuzz/property/differential v1: performance claim is forbidden")
	}
	if report.RuntimeBehaviorChanged {
		return fmt.Errorf(
			"fuzz/property/differential v1: runtime behavior change claim is forbidden",
		)
	}
	if report.SafeSemanticsChanged {
		return fmt.Errorf("fuzz/property/differential v1: safe semantics change claim is forbidden")
	}
	if report.ParserCheckerGeneratedPrograms == 0 {
		return fmt.Errorf(
			"fuzz/property/differential v1: parser/checker generated program coverage missing",
		)
	}
	if report.PLIRVerifierCases < report.ParserCheckerGeneratedPrograms ||
		report.LoweringVerifierCases < report.ParserCheckerGeneratedPrograms {
		return fmt.Errorf(
			"fuzz/property/differential v1: PLIR/lowering verifier coverage incomplete",
		)
	}
	if report.BackendMatrixCases == 0 {
		return fmt.Errorf("fuzz/property/differential v1: backend matrix coverage missing")
	}
	if report.BackendMatrixRandomizedSamples == 0 {
		return fmt.Errorf(
			"fuzz/property/differential v1: randomized backend matrix samples missing",
		)
	}
	if !report.BackendMatrixReducerRecorded {
		return fmt.Errorf("fuzz/property/differential v1: reducer evidence missing")
	}
	if report.NativeBackendHostSupported {
		if report.NativeBackendSamples == 0 {
			return fmt.Errorf(
				"fuzz/property/differential v1: native backend host supported but samples missing",
			)
		}
	} else if !strings.Contains(report.NativeBackendUnavailableReason, "linux/amd64") {
		return fmt.Errorf("fuzz/property/differential v1: native backend unavailable boundary missing")
	}
	if report.RuntimeAllocatorPropertyCases == 0 || !report.RuntimeAllocatorRejectsInvalid {
		return fmt.Errorf(
			"fuzz/property/differential v1: runtime allocator property evidence missing",
		)
	}
	if !report.ActorTransferStressDiagnostics {
		return fmt.Errorf(
			"fuzz/property/differential v1: actor transfer stress diagnostics missing",
		)
	}
	if report.FuzzSummaryGateArtifacts == 0 {
		return fmt.Errorf("fuzz/property/differential v1: fuzz summary gate artifacts missing")
	}
	if !report.NightlyLongFuzzBoundaryRecorded {
		return fmt.Errorf("fuzz/property/differential v1: nightly long fuzz boundary missing")
	}
	for _, want := range []string{
		"no full program correctness claim is made",
		"no exhaustive fuzzing is claimed",
		"no full native differential suite is claimed",
		"no performance claim is made",
		"runtime behavior does not change",
		"safe-program semantics do not change",
	} {
		if !p23FuzzHasString(report.NonClaims, want) {
			return fmt.Errorf("fuzz/property/differential v1: missing non-claim %q", want)
		}
	}
	if err := validateP23FuzzRowsAndWitnesses(report.Rows, report.Witnesses); err != nil {
		return err
	}
	return nil
}

func buildP23FuzzGeneratedPipelineWitness() (FuzzPropertyDifferentialWitness, error) {
	sources := p23FuzzGeneratedSources()
	for i, src := range sources {
		prog, err := Parse([]byte(src))
		if err != nil {
			return FuzzPropertyDifferentialWitness{}, fmt.Errorf(
				"p23.1 generated source %d parse: %w",
				i,
				err,
			)
		}
		checked, err := Check(prog)
		if err != nil {
			return FuzzPropertyDifferentialWitness{}, fmt.Errorf(
				"p23.1 generated source %d check: %w",
				i,
				err,
			)
		}
		plirProg, err := BuildPLIR(checked)
		if err != nil {
			return FuzzPropertyDifferentialWitness{}, fmt.Errorf(
				"p23.1 generated source %d PLIR: %w",
				i,
				err,
			)
		}
		if len(plirProg.Funcs) == 0 || !strings.Contains(FormatPLIR(plirProg), "func main") {
			return FuzzPropertyDifferentialWitness{}, fmt.Errorf(
				"p23.1 generated source %d PLIR missing main",
				i,
			)
		}
		irProg, err := Lower(checked)
		if err != nil {
			return FuzzPropertyDifferentialWitness{}, fmt.Errorf(
				"p23.1 generated source %d lower: %w",
				i,
				err,
			)
		}
		if err := VerifyIRProgram(irProg); err != nil {
			return FuzzPropertyDifferentialWitness{}, fmt.Errorf(
				"p23.1 generated source %d verify IR: %w",
				i,
				err,
			)
		}
	}
	return FuzzPropertyDifferentialWitness{
		ID:                    p23FuzzGeneratedPipelineWitnessID,
		Kind:                  "generated_parser_checker_plir_lowering",
		GeneratedPrograms:     len(sources),
		ParserCheckerCases:    len(sources),
		PLIRVerifierCases:     len(sources),
		LoweringVerifierCases: len(sources),
	}, nil
}

func buildP23FuzzBackendMatrixWitness() (FuzzPropertyDifferentialWitness, error) {
	matrix, err := differential.CheckBackendMatrix(differential.BackendMatrixCase{
		Name:              "p23.1-randomized-loop",
		Functions:         []ir.IRFunc{p23LoopSumFunc()},
		Entry:             "sum_n",
		Samples:           []differential.MatrixSample{{Name: "fixed-five", Args: []int32{5}}},
		RandomSeed:        231,
		RandomSampleCount: 4,
		Source: func(sample differential.MatrixSample) (int32, bool) {
			n := sample.Args[0]
			var total int32
			for i := int32(0); i < n; i++ {
				total += i
			}
			return total, true
		},
		Optimizations: []opt.Pass{opt.BasicScalarPass()},
	})
	if err != nil {
		return FuzzPropertyDifferentialWitness{}, err
	}
	if !matrix.HasLane(differential.LaneSSAInterpreter) ||
		!matrix.HasLane(differential.LaneMachineIRInterpreter) {
		return FuzzPropertyDifferentialWitness{}, fmt.Errorf(
			"p23.1 backend matrix missing SSA or Machine IR lane: %+v",
			matrix.Lanes,
		)
	}
	return FuzzPropertyDifferentialWitness{
		ID:                 p23FuzzBackendMatrixWitnessID,
		Kind:               "backend_matrix_randomized",
		BackendMatrixCases: 1,
		RandomizedSamples:  matrix.Randomized.Generated,
	}, nil
}

func buildP23FuzzNativeBackendWitness() (FuzzPropertyDifferentialWitness, error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return FuzzPropertyDifferentialWitness{
			ID:                  p23FuzzNativeBackendWitnessID,
			Kind:                "native_backend_boundary",
			NativeHostSupported: false,
			NativeUnavailableReason: fmt.Sprintf(
				"native differential lane requires linux/amd64 host; current host is %s/%s",
				runtime.GOOS,
				runtime.GOARCH,
			),
		}, nil
	}
	if err := os.MkdirAll(".cache", 0o755); err != nil {
		return FuzzPropertyDifferentialWitness{}, err
	}
	workDir, err := os.MkdirTemp(".cache", "p23.1-native-*")
	if err != nil {
		return FuzzPropertyDifferentialWitness{}, err
	}
	defer os.RemoveAll(workDir)
	matrix, err := differential.CheckBackendMatrix(differential.BackendMatrixCase{
		Name:      "p23.1-native-add",
		Functions: []ir.IRFunc{p23FuzzAddFunc()},
		Entry:     "add",
		Samples: []differential.MatrixSample{
			{Name: "small", Args: []int32{7, 4}},
			{Name: "zero", Args: []int32{0, 3}},
		},
		Source: func(sample differential.MatrixSample) (int32, bool) {
			return sample.Args[0] + sample.Args[1], true
		},
		Native: func(tc differential.BackendMatrixCase, sample differential.MatrixSample) (int32, error) {
			funcs := append([]ir.IRFunc{}, tc.Functions...)
			funcs = append(funcs, p23FuzzMainCallingFunction("main", tc.Entry, sample.Args))
			return differential.EvalNativeLinuxX64Exit(
				funcs,
				"main",
				workDir,
				tc.Name+"-"+sample.Name,
			)
		},
	})
	if err != nil {
		return FuzzPropertyDifferentialWitness{}, err
	}
	if !matrix.HasLane(differential.LaneNativeExecution) {
		return FuzzPropertyDifferentialWitness{}, fmt.Errorf(
			"p23.1 native matrix missing native lane: %+v",
			matrix.Lanes,
		)
	}
	return FuzzPropertyDifferentialWitness{
		ID:                  p23FuzzNativeBackendWitnessID,
		Kind:                "native_backend_boundary",
		NativeHostSupported: true,
		NativeSamples:       len(matrix.Samples),
	}, nil
}

func buildP23FuzzAllocatorWitness() (FuzzPropertyDifferentialWitness, error) {
	cfg := runtimeabi.RuntimeRegionAllocatorConfig(false)
	valid := []int64{0, 1, 15, 16, 17, 31, 32, int64(cfg.MaxPayloadBytes)}
	for _, input := range valid {
		aligned, ok := runtimeabi.AlignRegionBytes(input)
		if !ok || aligned%int64(runtimeabi.RegionAllocatorAlignmentBytes) != 0 {
			return FuzzPropertyDifferentialWitness{}, fmt.Errorf(
				"p23.1 allocator property input %d = %d,%v",
				input,
				aligned,
				ok,
			)
		}
	}
	invalidRejected := true
	for _, input := range []int64{-1, int64(
		runtimeabi.MaxRegionMapBytes,
	), int64(
		runtimeabi.MaxRegionMapBytes,
	) + 1} {
		if _, ok := runtimeabi.AlignRegionBytes(input); ok {
			invalidRejected = false
		}
	}
	return FuzzPropertyDifferentialWitness{
		ID:                             p23FuzzAllocatorWitnessID,
		Kind:                           "runtime_allocator_properties",
		RuntimeAllocatorPropertyCases:  len(valid) + 3,
		RuntimeAllocatorRejectsInvalid: invalidRejected,
	}, nil
}

func buildP23FuzzActorTransferWitness() (FuzzPropertyDifferentialWitness, error) {
	report := actorsafety.TypedActorOwnershipTransferCoverage()
	if err := actorsafety.ValidateTypedActorOwnershipTransferCoverage(report); err != nil {
		return FuzzPropertyDifferentialWitness{}, err
	}
	var stress, plirMoved bool
	for _, row := range report.Rows {
		if row.ID == actorsafety.TypedActorOwnershipStressDiagnostics {
			stress = true
		}
		if row.ID == actorsafety.TypedActorOwnershipPLIRMovedFacts {
			plirMoved = p23FuzzHasString(row.RequiredFacts, "FactMoved") &&
				p23FuzzHasString(row.RequiredFacts, "OpActorSend")
		}
	}
	return FuzzPropertyDifferentialWitness{
		ID:                             p23FuzzActorTransferWitnessID,
		Kind:                           "actor_transfer_stress_boundary",
		ActorTransferStressDiagnostics: stress,
		ActorTransferPLIRMovedFacts:    plirMoved,
	}, nil
}

func buildP23FuzzSummaryGateWitness() FuzzPropertyDifferentialWitness {
	return FuzzPropertyDifferentialWitness{
		ID:                              p23FuzzSummaryGateWitnessID,
		Kind:                            "fuzz_nightly_summary_gate",
		FuzzSummaryGateArtifacts:        15,
		NightlyLongFuzzBoundaryRecorded: true,
	}
}

func buildP23FuzzReducerWitness() (FuzzPropertyDifferentialWitness, error) {
	matrix, err := differential.CheckBackendMatrix(differential.BackendMatrixCase{
		Name:              "p23.1-bad-add-oracle",
		Functions:         []ir.IRFunc{p23FuzzAddFunc()},
		Entry:             "add",
		Samples:           []differential.MatrixSample{{Name: "fixed", Args: []int32{4, 3}}},
		RandomSeed:        16,
		RandomSampleCount: 2,
		Source: func(sample differential.MatrixSample) (int32, bool) {
			return sample.Args[0] - sample.Args[1], true
		},
	})
	if err == nil || !strings.Contains(err.Error(), "differential mismatch") {
		return FuzzPropertyDifferentialWitness{}, fmt.Errorf(
			"p23.1 reducer witness error = %v, want differential mismatch",
			err,
		)
	}
	reduced := matrix.Mismatch != nil &&
		matrix.Mismatch.ReducerStatus == "reduced_to_single_sample" &&
		strings.Contains(matrix.Mismatch.Reproducer, "p23.1-bad-add-oracle")
	return FuzzPropertyDifferentialWitness{
		ID:                            p23FuzzReducerWitnessID,
		Kind:                          "reducer_failure_artifact",
		ReducerRecorded:               reduced,
		ReducedSingleSampleReproducer: reduced,
	}, nil
}

func validateP23FuzzRowsAndWitnesses(
	rows []FuzzPropertyDifferentialRow,
	witnesses []FuzzPropertyDifferentialWitness,
) error {
	witnessIDs := map[string]bool{}
	for _, witness := range witnesses {
		if strings.TrimSpace(witness.ID) == "" {
			return fmt.Errorf("fuzz/property/differential v1: witness missing id")
		}
		witnessIDs[witness.ID] = true
	}
	expected := map[FuzzPropertyDifferentialID]bool{}
	for _, id := range p23FuzzPropertyDifferentialIDs() {
		expected[id] = false
	}
	seen := map[FuzzPropertyDifferentialID]bool{}
	for _, row := range rows {
		if row.ID == "" || row.Name == "" || row.Status == "" || len(row.Evidence) == 0 ||
			len(row.Tests) == 0 ||
			len(row.Boundaries) == 0 ||
			len(row.WitnessIDs) == 0 {
			return fmt.Errorf(
				"fuzz/property/differential v1: row %q missing required metadata",
				row.ID,
			)
		}
		if !expected[row.ID] {
			if _, ok := expected[row.ID]; !ok {
				return fmt.Errorf("fuzz/property/differential v1: unexpected row %s", row.ID)
			}
		}
		if seen[row.ID] {
			return fmt.Errorf("fuzz/property/differential v1: duplicate row %s", row.ID)
		}
		seen[row.ID] = true
		for _, evidence := range row.Evidence {
			if p23FuzzContainsPlaceholder(evidence) {
				return fmt.Errorf(
					"fuzz/property/differential v1: row %s contains placeholder evidence",
					row.ID,
				)
			}
		}
		for _, witnessID := range row.WitnessIDs {
			if !witnessIDs[witnessID] {
				return fmt.Errorf(
					"fuzz/property/differential v1: row %s references missing witness %q",
					row.ID,
					witnessID,
				)
			}
		}
	}
	for _, id := range p23FuzzPropertyDifferentialIDs() {
		if !seen[id] {
			return fmt.Errorf("fuzz/property/differential v1: missing row %s", id)
		}
	}
	return nil
}

func p23FuzzPropertyDifferentialIDs() []FuzzPropertyDifferentialID {
	return []FuzzPropertyDifferentialID{
		FuzzPropertyDifferentialParserCheckerGeneratedPrograms,
		FuzzPropertyDifferentialPLIRLoweringVerifierPipeline,
		FuzzPropertyDifferentialBackendMatrixExpansion,
		FuzzPropertyDifferentialNativeBackendBoundary,
		FuzzPropertyDifferentialRuntimeAllocatorProperties,
		FuzzPropertyDifferentialActorTransferStressBoundary,
		FuzzPropertyDifferentialFuzzNightlySummaryGate,
		FuzzPropertyDifferentialReducerFailureArtifacts,
	}
}

func p23FuzzRow(
	id FuzzPropertyDifferentialID,
	name, status string,
	evidence, tests, boundaries, witnessIDs []string,
) FuzzPropertyDifferentialRow {
	return FuzzPropertyDifferentialRow{
		ID:         id,
		Name:       name,
		Status:     status,
		Evidence:   append([]string(nil), evidence...),
		Tests:      append([]string(nil), tests...),
		Boundaries: append([]string(nil), boundaries...),
		WitnessIDs: append([]string(nil), witnessIDs...),
	}
}

func p23FuzzGeneratedSources() []string {
	return []string{
		"func main() -> Int:\n    let x: Int = 1\n    return x\n",
		("func add(a: Int, b: Int) -> Int:\n    return a + b\n\nfunc " +
			"main() -> Int:\n    return add(1, 2)\n"),
		"func main() -> Int:\n    if 1 < 2:\n        return 1\n    return 0\n",
		("func main() -> Int:\n    var total: Int = 0\n    for i in " +
			"0..<4:\n        total = total + i\n    return total\n"),
	}
}

func p23FuzzAddFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "add",
		ParamSlots:  2,
		LocalSlots:  2,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRReturn},
		},
	}
}

func p23FuzzMainCallingFunction(name string, callee string, args []int32) ir.IRFunc {
	instrs := make([]ir.IRInstr, 0, len(args)+2)
	for _, arg := range args {
		instrs = append(instrs, ir.IRInstr{Kind: ir.IRConstI32, Imm: arg})
	}
	instrs = append(instrs,
		ir.IRInstr{Kind: ir.IRCall, Name: callee, ArgSlots: len(args), RetSlots: 1},
		ir.IRInstr{Kind: ir.IRReturn},
	)
	return ir.IRFunc{
		Name:        name,
		ParamSlots:  0,
		LocalSlots:  0,
		ReturnSlots: 1,
		Instrs:      instrs,
	}
}

func p23FuzzHasString(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}

func p23FuzzContainsPlaceholder(text string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(text))
	return trimmed == "" || trimmed == "todo" || strings.Contains(trimmed, "todo:") ||
		strings.Contains(trimmed, "placeholder")
}

// ---- fuzz_suite.go ----

type FuzzCheck struct {
	Name  string
	Error string
}

func RunTargetFuzzChecks(targetName string) ([]FuzzCheck, error) {
	tgt, err := ctarget.Parse(targetName)
	if err != nil {
		return nil, err
	}
	if tgt.Arch != ctarget.ArchX86 && tgt.Arch != ctarget.ArchX64 {
		return nil, fmt.Errorf(
			"fuzz/property suite for target %s requires an x86/x64 native target model",
			tgt.Triple,
		)
	}
	prefix := targetFuzzPrefix(tgt)
	return runFuzzChecks([]struct {
		name string
		run  func() error
	}{
		{name: prefix + " layout fuzz", run: func() error { return checkTargetLayoutFuzz(tgt) }},
		{
			name: prefix + " object signature fuzz",
			run:  func() error { return checkTargetObjectSignatureFuzz(tgt) },
		},
		{name: prefix + " target alias fuzz", run: checkTargetAliasFuzz},
	}), nil
}

func runFuzzChecks(cases []struct {
	name string
	run  func() error
}) []FuzzCheck {
	out := make([]FuzzCheck, 0, len(cases))
	for _, tc := range cases {
		check := FuzzCheck{Name: tc.name}
		if err := tc.run(); err != nil {
			check.Error = err.Error()
		}
		out = append(out, check)
	}
	return out
}

func checkTargetLayoutFuzz(tgt ctarget.Target) error {
	x86, err := ctarget.Parse("x86")
	if err != nil {
		return err
	}
	x32, err := ctarget.Parse("x32")
	if err != nil {
		return err
	}
	x64, err := ctarget.Parse("x64")
	if err != nil {
		return err
	}
	seed := int64(0x6432)
	if tgt.Arch == ctarget.ArchX86 {
		seed = 0x8632
	}
	if tgt.ABI == ctarget.ABIX32SysV {
		seed = 0x3232
	}
	rng := rand.New(rand.NewSource(seed))
	fieldTypes := []string{"u8", "u16", "i32", "ptr", "usize", "isize", "size_t", "c_long", "i64"}
	for i := 0; i < 200; i++ {
		fields := make([]ctarget.LayoutField, 0, 1+rng.Intn(8))
		for j, count := 0, 1+rng.Intn(8); j < count; j++ {
			fields = append(fields, ctarget.LayoutField{
				Name: fmt.Sprintf("f_%d_%d", i, j),
				Type: fieldTypes[rng.Intn(len(fieldTypes))],
			})
		}
		got, err := tgt.StructLayout(fields)
		if err != nil {
			return fmt.Errorf("layout fuzz case %d: %w", i, err)
		}
		want, err := referenceFuzzStructLayout(tgt, fields, false)
		if err != nil {
			return fmt.Errorf("reference layout fuzz case %d: %w", i, err)
		}
		if err := compareFuzzAggregateLayout(
			fmt.Sprintf("%s struct fuzz case %d", targetFuzzPrefix(tgt), i),
			got,
			want,
		); err != nil {
			return err
		}
		packedGot, err := tgt.PackedStructLayout(fields)
		if err != nil {
			return fmt.Errorf("packed layout fuzz case %d: %w", i, err)
		}
		packedWant, err := referenceFuzzStructLayout(tgt, fields, true)
		if err != nil {
			return fmt.Errorf("reference packed layout fuzz case %d: %w", i, err)
		}
		if err := compareFuzzAggregateLayout(
			fmt.Sprintf("%s packed struct fuzz case %d", targetFuzzPrefix(tgt), i),
			packedGot,
			packedWant,
		); err != nil {
			return err
		}
		arrayType := fieldTypes[rng.Intn(len(fieldTypes))]
		count := rng.Intn(32)
		arr, err := tgt.ArrayLayout(arrayType, count)
		if err != nil {
			return fmt.Errorf("array layout fuzz case %d: %w", i, err)
		}
		scalar, ok := tgt.ScalarLayout(arrayType)
		if !ok {
			return fmt.Errorf("array layout fuzz case %d missing scalar %s", i, arrayType)
		}
		wantArraySize := fuzzAlignUp(scalar.SizeBytes, scalar.AlignBytes) * count
		if arr.SizeBytes != wantArraySize || arr.AlignBytes != scalar.AlignBytes {
			return fmt.Errorf(
				"%s array fuzz case %d = %#v, want size=%d align=%d",
				targetFuzzPrefix(tgt),
				i,
				arr,
				wantArraySize,
				scalar.AlignBytes,
			)
		}
	}
	pointerFields := []ctarget.LayoutField{{Name: "p", Type: "ptr"}, {Name: "n", Type: "usize"}}
	x86Struct, err := x86.StructLayout(pointerFields)
	if err != nil {
		return err
	}
	x32Struct, err := x32.StructLayout(pointerFields)
	if err != nil {
		return err
	}
	x64Struct, err := x64.StructLayout(pointerFields)
	if err != nil {
		return err
	}
	if x86Struct.SizeBytes != 8 || x86Struct.AlignBytes != 4 || x32Struct.SizeBytes != 8 ||
		x32Struct.AlignBytes != 4 ||
		x64Struct.SizeBytes != 16 ||
		x64Struct.AlignBytes != 8 {
		return fmt.Errorf(
			"x86/x32/x64 pointer-sensitive struct layouts collapsed: x86=%#v x32=%#v x64=%#v",
			x86Struct,
			x32Struct,
			x64Struct,
		)
	}
	if x86.RegisterWidthBits == x32.RegisterWidthBits || x32.RegisterWidthBits != 64 {
		return fmt.Errorf(
			"x86/x32 register models collapsed: x86=%d x32=%d",
			x86.RegisterWidthBits,
			x32.RegisterWidthBits,
		)
	}
	if err := checkTargetArrayBoundaryFuzz(x86, x32, x64); err != nil {
		return err
	}
	return nil
}

func checkTargetArrayBoundaryFuzz(
	x86 ctarget.Target,
	x32 ctarget.Target,
	x64 ctarget.Target,
) error {
	near32BitByteLimit := 1<<30 - 1
	over32BitByteLimit := 1 << 30
	for _, tgt := range []ctarget.Target{x86, x32} {
		near, err := tgt.ArrayLayout("ptr", near32BitByteLimit)
		if err != nil {
			return fmt.Errorf("%s near-limit pointer array rejected: %w", tgt.Triple, err)
		}
		if got, want := uint64(near.SizeBytes), (uint64(1)<<32)-4; got != want {
			return fmt.Errorf(
				"%s near-limit pointer array size = %d, want %d",
				tgt.Triple,
				got,
				want,
			)
		}
		err = expectArrayLayoutError(tgt, "ptr", over32BitByteLimit)
		if err != nil {
			return err
		}
	}
	large, err := x64.ArrayLayout("ptr", over32BitByteLimit)
	if err != nil {
		return fmt.Errorf("x64 large pointer array rejected at x32 boundary: %w", err)
	}
	if got, want := uint64(large.SizeBytes), uint64(1)<<33; got != want {
		return fmt.Errorf("x64 large pointer array size = %d, want %d", got, want)
	}
	return nil
}

func expectArrayLayoutError(tgt ctarget.Target, elemType string, count int) error {
	if _, err := tgt.ArrayLayout(elemType, count); err == nil {
		return fmt.Errorf(
			"%s [%d]%s layout accepted target-native overflow",
			tgt.Triple,
			count,
			elemType,
		)
	} else if !strings.Contains(err.Error(), "exceeds 32-bit native size limit") {
		return fmt.Errorf(
			"%s [%d]%s overflow error = %q, want native-size diagnostic",
			tgt.Triple,
			count,
			elemType,
			err,
		)
	}
	return nil
}

func referenceFuzzStructLayout(
	tgt ctarget.Target,
	fields []ctarget.LayoutField,
	packed bool,
) (ctarget.AggregateLayout, error) {
	out := ctarget.AggregateLayout{
		AlignBytes: 1,
		Fields:     make([]ctarget.FieldLayout, 0, len(fields)),
	}
	offset := 0
	for _, field := range fields {
		if len(field.Fields) > 0 {
			return ctarget.AggregateLayout{}, fmt.Errorf(
				"nested reference fields are not part of this fuzz oracle",
			)
		}
		scalar, ok := tgt.ScalarLayout(field.Type)
		if !ok {
			return ctarget.AggregateLayout{}, fmt.Errorf(
				"unknown reference layout type %q",
				field.Type,
			)
		}
		align := scalar.AlignBytes
		if packed || field.Packed {
			align = 1
		}
		offset = fuzzAlignUp(offset, align)
		out.Fields = append(out.Fields, ctarget.FieldLayout{
			Name:        field.Name,
			Type:        scalar.Name,
			OffsetBytes: offset,
			SizeBytes:   scalar.SizeBytes,
			AlignBytes:  align,
			ABIBytes:    scalar.ABIBytes,
		})
		offset += scalar.SizeBytes
		if align > out.AlignBytes {
			out.AlignBytes = align
		}
	}
	out.SizeBytes = fuzzAlignUp(offset, out.AlignBytes)
	return out, nil
}

func compareFuzzAggregateLayout(
	name string,
	got ctarget.AggregateLayout,
	want ctarget.AggregateLayout,
) error {
	if got.SizeBytes != want.SizeBytes || got.AlignBytes != want.AlignBytes ||
		len(got.Fields) != len(want.Fields) {
		return fmt.Errorf("%s layout = %#v, want %#v", name, got, want)
	}
	for i := range got.Fields {
		gf := got.Fields[i]
		wf := want.Fields[i]
		if gf.Name != wf.Name || gf.Type != wf.Type || gf.OffsetBytes != wf.OffsetBytes ||
			gf.SizeBytes != wf.SizeBytes ||
			gf.AlignBytes != wf.AlignBytes ||
			gf.ABIBytes != wf.ABIBytes {
			return fmt.Errorf("%s field %d = %#v, want %#v", name, i, gf, wf)
		}
	}
	return nil
}

func fuzzAlignUp(value int, align int) int {
	if align <= 1 {
		return value
	}
	remainder := value % align
	if remainder == 0 {
		return value
	}
	return value + align - remainder
}

func checkTargetObjectSignatureFuzz(tgt ctarget.Target) error {
	tmpDir, err := os.MkdirTemp("", "tetra-fuzz-suite-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	prefix := targetFuzzPrefix(tgt)
	srcPath := filepath.Join(tmpDir, prefix+"_signature_fuzz.tetra")
	outPath := filepath.Join(tmpDir, prefix+"_signature_fuzz.tobj")
	rng := rand.New(rand.NewSource(0x32F00D))
	if tgt.Arch == ctarget.ArchX86 {
		rng = rand.New(rand.NewSource(0x86F00D))
	} else if tgt.ABI != ctarget.ABIX32SysV {
		rng = rand.New(rand.NewSource(0x64F00D))
	}
	var src strings.Builder
	var symbols []string
	for i := 0; i < 48; i++ {
		switch rng.Intn(4) {
		case 0:
			name := fmt.Sprintf("fuzz_i32_%02d", i)
			fmt.Fprintf(&src, "fun %s(a: i32, b: i32): i32 { return a + b }\n", name)
			symbols = append(symbols, name)
		case 1:
			name := fmt.Sprintf("fuzz_i64_%02d", i)
			fmt.Fprintf(&src, "fun %s(a: i64): i64 { return a }\n", name)
			symbols = append(symbols, name)
		case 2:
			name := fmt.Sprintf("fuzz_ptr_%02d", i)
			fmt.Fprintf(&src, "fun %s(p: ptr): ptr { return p }\n", name)
			symbols = append(symbols, name)
		default:
			name := fmt.Sprintf("fuzz_mixed_%02d", i)
			fmt.Fprintf(&src, "fun %s(a: i32, p: ptr): i32 { return a }\n", name)
			symbols = append(symbols, name)
		}
	}
	if err := os.WriteFile(srcPath, []byte(src.String()), 0o644); err != nil {
		return err
	}
	if _, err := BuildFileWithStatsOpt(
		srcPath,
		outPath,
		tgt.Triple,
		BuildOptions{Emit: EmitLibrary},
	); err != nil {
		return err
	}
	obj, err := ReadObject(outPath)
	if err != nil {
		return err
	}
	if obj.Target != tgt.Triple {
		return fmt.Errorf("target mismatch: got %q want %s", obj.Target, tgt.Triple)
	}
	for _, symbol := range symbols {
		if !objectHasSymbol(obj, symbol) {
			return fmt.Errorf("object signature fuzz missing symbol %s", symbol)
		}
	}
	if tgt.OS != ctarget.OSWindows {
		for _, reloc := range obj.Relocs {
			if reloc.Kind == RelocIATDisp32 {
				return fmt.Errorf(
					"%s signature fuzz unexpectedly has Windows IAT reloc: %#v",
					tgt.Triple,
					obj.Relocs,
				)
			}
		}
	}
	return nil
}

func checkTargetAliasFuzz() error {
	x32Aliases := []string{
		"x32", "x86_64-x32", "linux-x32", "linux-x86_64-x32",
		"x86_64-linux-gnux32",
		"x86_64-unknown-linux-gnux32",
		"x86_64-pc-linux-gnux32",
		"linux-x86_64-gnux32",
	}
	for _, alias := range x32Aliases {
		tgt, err := ctarget.Parse(alias)
		if err != nil {
			return fmt.Errorf("x32 alias %q rejected: %w", alias, err)
		}
		if tgt.Triple != "linux-x32" || tgt.Arch != ctarget.ArchX64 ||
			tgt.ABI != ctarget.ABIX32SysV ||
			tgt.PointerWidthBits != 32 ||
			tgt.RegisterWidthBits != 64 {
			return fmt.Errorf("x32 alias %q parsed as %#v", alias, tgt)
		}
	}
	x86Aliases := []string{
		"x86", "i386", "i686", "linux-i386", "linux-i686",
		"i386-linux-gnu", "i686-linux-gnu", "i686-unknown-linux-gnu", "i686-pc-linux-gnu",
	}
	for _, alias := range x86Aliases {
		tgt, err := ctarget.Parse(alias)
		if err != nil {
			return fmt.Errorf("x86 alias %q rejected: %w", alias, err)
		}
		if tgt.Triple != "linux-x86" || tgt.Arch != ctarget.ArchX86 ||
			tgt.ABI != ctarget.ABI386SysV ||
			tgt.PointerWidthBits != 32 ||
			tgt.RegisterWidthBits != 32 {
			return fmt.Errorf("x86 alias %q parsed as %#v", alias, tgt)
		}
	}
	x64Aliases := []string{
		"x64", "amd64", "x86_64", "linux-amd64", "linux-x86_64",
		"x86_64-linux-gnu", "x86_64-unknown-linux-gnu", "x86_64-pc-linux-gnu", "amd64-linux-gnu",
	}
	for _, alias := range x64Aliases {
		tgt, err := ctarget.Parse(alias)
		if err != nil {
			return fmt.Errorf("x64 alias %q rejected: %w", alias, err)
		}
		if tgt.Triple != "linux-x64" || tgt.Arch != ctarget.ArchX64 || tgt.ABI != ctarget.ABISysV ||
			tgt.PointerWidthBits != 64 ||
			tgt.RegisterWidthBits != 64 {
			return fmt.Errorf("x64 alias %q parsed as %#v", alias, tgt)
		}
	}
	windowsX64Aliases := []string{
		"win-x64", "windows-amd64", "windows-x86_64",
		"x86_64-pc-windows-msvc", "x86_64-pc-windows-gnu", "amd64-windows-msvc",
	}
	for _, alias := range windowsX64Aliases {
		tgt, err := ctarget.Parse(alias)
		if err != nil {
			return fmt.Errorf("windows x64 alias %q rejected: %w", alias, err)
		}
		if tgt.Triple != "windows-x64" || tgt.Arch != ctarget.ArchX64 ||
			tgt.ABI != ctarget.ABIWin64 ||
			tgt.DataModel != ctarget.DataModelLLP64 ||
			tgt.Format != ctarget.FormatPE {
			return fmt.Errorf("windows x64 alias %q parsed as %#v", alias, tgt)
		}
	}
	macosX64Aliases := []string{
		"darwin-x64", "macos-amd64", "macos-x86_64",
		"x86_64-apple-darwin", "amd64-apple-darwin",
	}
	for _, alias := range macosX64Aliases {
		tgt, err := ctarget.Parse(alias)
		if err != nil {
			return fmt.Errorf("macos x64 alias %q rejected: %w", alias, err)
		}
		if tgt.Triple != "macos-x64" || tgt.Arch != ctarget.ArchX64 || tgt.ABI != ctarget.ABISysV ||
			tgt.DataModel != ctarget.DataModelLP64 ||
			tgt.Format != ctarget.FormatMachO {
			return fmt.Errorf("macos x64 alias %q parsed as %#v", alias, tgt)
		}
	}
	for _, alias := range []string{"x86-x32", "linux-amd64-x32", "linux-x64-x32", "x32_64"} {
		if tgt, err := ctarget.Parse(alias); err == nil {
			return fmt.Errorf("invalid target alias %q parsed as %#v", alias, tgt)
		}
	}
	return nil
}

func targetFuzzPrefix(tgt ctarget.Target) string {
	if tgt.Arch == ctarget.ArchX86 {
		return "x86"
	}
	if tgt.ABI == ctarget.ABIX32SysV {
		return "x32"
	}
	if tgt.Triple == "windows-x64" || tgt.Triple == "macos-x64" {
		return tgt.Triple
	}
	return "x64"
}

// ---- memory_fuzz_oracle_v1.go ----

const (
	MemoryFuzzOracleSchemaV1   = "tetra.memory-fuzz.oracle.v1"
	MemoryFuzzOracleScopeMPC15 = "memory_production_core_v1_mpc15"
)

type MemoryFuzzOracleCategory string
type memoryFuzzOracleCat = MemoryFuzzOracleCategory

const (
	MemoryFuzzOracleCheckerRejectExpected   memoryFuzzOracleCat = "checker_reject_expected"
	MemoryFuzzOracleRuntimeTrapExpected     memoryFuzzOracleCat = "runtime_trap_expected"
	MemoryFuzzOracleReferenceOutputExpected memoryFuzzOracleCat = ("compiled_output_equals_" +
		"reference_expected")
	MemoryFuzzOracleCompilerCrashBug                memoryFuzzOracleCat = "compiler_crash_is_bug"
	MemoryFuzzOracleMiscompileBug                   memoryFuzzOracleCat = "miscompile_is_bug"
	MemoryFuzzOracleUnsafeUnknownOptimizedAsSafeBug memoryFuzzOracleCat = ("unsafe_unknown_" +
		"optimized_as_safe_is_bug")
	MemoryFuzzOracleReportValidationFailureBug memoryFuzzOracleCat = "report_validation_failure_is_bug"
)

type MemoryFuzzOracleResult string

const (
	MemoryFuzzOraclePass MemoryFuzzOracleResult = "pass"
	MemoryFuzzOracleFail MemoryFuzzOracleResult = "fail"
	MemoryFuzzOracleBug  MemoryFuzzOracleResult = "bug"
)

type MemoryFuzzTier string

const (
	MemoryFuzzTier1ShortCI        MemoryFuzzTier = "tier1_short_ci_smoke"
	MemoryFuzzTier2Nightly        MemoryFuzzTier = "tier2_nightly_fuzz"
	MemoryFuzzTier3ReleaseFocused MemoryFuzzTier = "tier3_release_blocking_focused_memory_fuzz"
)

type MemoryFuzzInvariantID string
type memoryFuzzInvariant = MemoryFuzzInvariantID

const (
	MemoryFuzzInvariantNoSafeMetadataMutation     memoryFuzzInvariant = "no_safe_metadata_mutation"
	MemoryFuzzInvariantNoBorrowedEscape           memoryFuzzInvariant = "no_borrowed_escape"
	MemoryFuzzInvariantNoUnsafeUnknownToSafeKnown                     = memoryFuzzInvariant(
		"no_unsafe_unknown_to_safe_known",
	)
	MemoryFuzzInvariantNoBoundsRemovalWithoutProofID = memoryFuzzInvariant(
		"no_removed_bounds_check_" +
			"without_proof_id",
	)
	MemoryFuzzInvariantNoStackRegionStorageWhenEscaped = memoryFuzzInvariant(
		"no_stack_region_storage_" +
			"if_escape_exists",
	)
	MemoryFuzzInvariantReportsValidateAgainstFactGraph memoryFuzzInvariant = ("reports_validate_" +
		"against_memory_fact_graph")
	MemoryFuzzInvariantReportsPreserveMemoryCostModel = memoryFuzzInvariant(
		"reports_preserve_memory_cost_model",
	)
)

type MemoryFuzzGeneratorSurfaceTier string
type memoryFuzzGeneratorTier = MemoryFuzzGeneratorSurfaceTier

const (
	MemoryFuzzGeneratorTier1SupportedNow         memoryFuzzGeneratorTier = "tier1_supported_now"
	MemoryFuzzGeneratorTier2SupportedNarrow      memoryFuzzGeneratorTier = "tier2_supported_narrow"
	MemoryFuzzGeneratorTier3ConservativeRejected                         = memoryFuzzGeneratorTier(
		"tier3_conservative_rejected",
	)
	MemoryFuzzGeneratorTier4Future memoryFuzzGeneratorTier = "tier4_future"
)

type MemoryFuzzRequirementID string

const (
	MemoryFuzzRequirementTier1V0V11Coverage         MemoryFuzzRequirementID = "MEM-FUZZ-001"
	MemoryFuzzRequirementCrashMiscompileArtifacts   MemoryFuzzRequirementID = "MEM-FUZZ-002"
	MemoryFuzzRequirementBlockingMemoryFailures     MemoryFuzzRequirementID = "MEM-FUZZ-003"
	MemoryFuzzRequirementTier2NightlySeedTriage     MemoryFuzzRequirementID = "MEM-FUZZ-004"
	MemoryFuzzRequirementTier3ReleasePassOrClassify MemoryFuzzRequirementID = "MEM-FUZZ-005"
)

type MemoryFuzzBlockingCaseID string
type memoryFuzzBlockingID = MemoryFuzzBlockingCaseID

const (
	MemoryFuzzBlockingUnsafeUnknownOptimizedAsSafe = memoryFuzzBlockingID(
		"unsafe_unknown_optimized_as_safe",
	)
	MemoryFuzzBlockingBoundsCheckWithoutProofID memoryFuzzBlockingID = ("bounds_check_eliminated_" +
		"without_proof_id")
	MemoryFuzzBlockingTrustedStorageUnderEscape memoryFuzzBlockingID = "trusted_storage_under_escape"
	MemoryFuzzBlockingReportValidationFailure   memoryFuzzBlockingID = "report_validation_failure"
)

type MemoryFuzzObservation struct {
	CheckerRejected              bool
	RuntimeTrapped               bool
	ReferenceCompared            bool
	CompiledExitCode             int
	ReferenceExitCode            int
	CompilerCrashed              bool
	UnsafeUnknownOptimizedAsSafe bool
	ReportValidationFailed       bool
}

type MemoryFuzzOracleReport struct {
	SchemaVersion string `json:"schema_version"`
	GitHead       string `json:"git_head,omitempty"`
	Scope         string `json:"scope"`

	Tier1ShortCISmokeCases int `json:"tier1_short_ci_smoke_cases"`

	Tier2NightlyBoundaryRecorded bool `json:"tier2_nightly_boundary_recorded"`

	Tier3ReleaseBlockingBoundaryRecorded bool `json:"tier3_release_blocking_boundary_recorded"`

	Requirements      []MemoryFuzzRequirementRow      `json:"requirements"`
	SliceCoverage     []MemoryFuzzSliceCoverageRow    `json:"slice_coverage"`
	Rows              []MemoryFuzzOracleRow           `json:"rows"`
	Invariants        []MemoryFuzzInvariantRow        `json:"invariants"`
	GeneratorSurfaces []MemoryFuzzGeneratorSurfaceRow `json:"generator_surfaces"`
	BlockingCases     []MemoryFuzzBlockingCaseRow     `json:"blocking_cases"`
	TierPolicies      []MemoryFuzzTierPolicyRow       `json:"tier_policies"`
	Artifacts         []MemoryFuzzArtifact            `json:"artifacts"`
	NonClaims         []string                        `json:"non_claims"`
}

type MemoryFuzzRequirementRow struct {
	ID         MemoryFuzzRequirementID `json:"id"`
	Status     string                  `json:"status"`
	Evidence   []string                `json:"evidence"`
	Tests      []string                `json:"tests"`
	Boundaries []string                `json:"boundaries"`
}

type MemoryFuzzSliceCoverageRow struct {
	SliceID          string                     `json:"slice_id"`
	Status           string                     `json:"status"`
	Surface          []string                   `json:"surface"`
	OracleCategories []MemoryFuzzOracleCategory `json:"oracle_categories"`
	Invariants       []MemoryFuzzInvariantID    `json:"invariants"`
	Evidence         []string                   `json:"evidence"`
	Tests            []string                   `json:"tests"`
	Boundaries       []string                   `json:"boundaries"`
}

type MemoryFuzzOracleRow struct {
	Category       MemoryFuzzOracleCategory `json:"oracle_category"`
	Name           string                   `json:"name"`
	Tier           MemoryFuzzTier           `json:"tier"`
	ExpectedResult MemoryFuzzOracleResult   `json:"expected_result"`
	Status         string                   `json:"status"`
	Evidence       []string                 `json:"evidence"`
	Tests          []string                 `json:"tests"`
	Boundaries     []string                 `json:"boundaries"`
}

type MemoryFuzzInvariantRow struct {
	ID         MemoryFuzzInvariantID `json:"id"`
	Status     string                `json:"status"`
	Evidence   []string              `json:"evidence"`
	Tests      []string              `json:"tests"`
	Boundaries []string              `json:"boundaries"`
}

type MemoryFuzzGeneratorSurfaceRow struct {
	Tier       MemoryFuzzGeneratorSurfaceTier `json:"tier"`
	Status     string                         `json:"status"`
	Surface    []string                       `json:"surface"`
	Boundaries []string                       `json:"boundaries"`
}

type MemoryFuzzBlockingCaseRow struct {
	ID            MemoryFuzzBlockingCaseID `json:"id"`
	Status        string                   `json:"status"`
	BlocksRelease bool                     `json:"blocks_release"`
	Evidence      []string                 `json:"evidence"`
	Tests         []string                 `json:"tests"`
	Boundaries    []string                 `json:"boundaries"`
}

type MemoryFuzzTierPolicyRow struct {
	Tier                        MemoryFuzzTier `json:"tier"`
	Status                      string         `json:"status"`
	SeedsPreserved              bool           `json:"seeds_preserved"`
	UnstableTriageRequired      bool           `json:"unstable_triage_required"`
	MinimizedReproducerRequired bool           `json:"minimized_reproducer_required"`

	ReleasePromotionBlockedUntilClassified bool `json:"release_promotion_blocked_until_classified"`

	Evidence   []string `json:"evidence"`
	Tests      []string `json:"tests"`
	Boundaries []string `json:"boundaries"`
}

type MemoryFuzzArtifact struct {
	Path     string `json:"path"`
	Kind     string `json:"kind"`
	Required bool   `json:"required"`
}

func ClassifyMemoryFuzzOracleObservation(
	category MemoryFuzzOracleCategory,
	obs MemoryFuzzObservation,
) MemoryFuzzOracleResult {
	switch category {
	case MemoryFuzzOracleCheckerRejectExpected:
		if obs.CompilerCrashed {
			return MemoryFuzzOracleBug
		}
		if obs.CheckerRejected {
			return MemoryFuzzOraclePass
		}
		return MemoryFuzzOracleFail
	case MemoryFuzzOracleRuntimeTrapExpected:
		if obs.CompilerCrashed {
			return MemoryFuzzOracleBug
		}
		if obs.RuntimeTrapped {
			return MemoryFuzzOraclePass
		}
		return MemoryFuzzOracleFail
	case MemoryFuzzOracleReferenceOutputExpected:
		if obs.CompilerCrashed {
			return MemoryFuzzOracleBug
		}
		if !obs.ReferenceCompared {
			return MemoryFuzzOracleFail
		}
		if obs.CompiledExitCode == obs.ReferenceExitCode {
			return MemoryFuzzOraclePass
		}
		return MemoryFuzzOracleBug
	case MemoryFuzzOracleCompilerCrashBug:
		if obs.CompilerCrashed {
			return MemoryFuzzOracleBug
		}
		return MemoryFuzzOraclePass
	case MemoryFuzzOracleMiscompileBug:
		if obs.CompilerCrashed {
			return MemoryFuzzOracleBug
		}
		if obs.ReferenceCompared && obs.CompiledExitCode != obs.ReferenceExitCode {
			return MemoryFuzzOracleBug
		}
		return MemoryFuzzOraclePass
	case MemoryFuzzOracleUnsafeUnknownOptimizedAsSafeBug:
		if obs.UnsafeUnknownOptimizedAsSafe {
			return MemoryFuzzOracleBug
		}
		return MemoryFuzzOraclePass
	case MemoryFuzzOracleReportValidationFailureBug:
		if obs.ReportValidationFailed {
			return MemoryFuzzOracleBug
		}
		return MemoryFuzzOraclePass
	default:
		return MemoryFuzzOracleFail
	}
}

func BuildMemoryFuzzOracleReport() (MemoryFuzzOracleReport, error) {
	if err := memoryFuzzReportValidationFailureWitness(); err != nil {
		return MemoryFuzzOracleReport{}, err
	}
	return MemoryFuzzOracleReport{
		SchemaVersion:                        MemoryFuzzOracleSchemaV1,
		Scope:                                MemoryFuzzOracleScopeMPC15,
		Tier1ShortCISmokeCases:               12,
		Tier2NightlyBoundaryRecorded:         true,
		Tier3ReleaseBlockingBoundaryRecorded: true,
		Requirements: []MemoryFuzzRequirementRow{
			memoryFuzzRequirementRow(
				MemoryFuzzRequirementTier1V0V11Coverage,
				memoryvocab.FuzzStatusValidatedNarrow,
				("Tier 1 short CI smoke covers deterministic v0-v11 memory " +
					"oracle cases across the supported compiler-visible memory " +
					"surfaces"),
				"go test ./tools/cmd/memory-fuzz-short ./tools/cmd/validate-memory-fuzz-oracle -count=1",
				"Tier 1 is deterministic short smoke evidence, not exhaustive fuzz proof",
			),
			memoryFuzzRequirementRow(
				MemoryFuzzRequirementCrashMiscompileArtifacts,
				memoryvocab.FuzzStatusValidatedNarrow,
				("compiler crash and miscompile classifications require " +
					"reducer or reproducer artifact slots before evidence " +
					"promotion"),
				("go test ./compiler -run " +
					"'MemoryFuzzOracle.*V12|ValidateMemoryFuzzOracleReportRejects" +
					"V12' -count=1"),
				"artifact discipline is release evidence only and does not claim full program correctness",
			),
			memoryFuzzRequirementRow(
				MemoryFuzzRequirementBlockingMemoryFailures,
				memoryvocab.FuzzStatusReleaseBlocking,
				("unsafe_unknown optimized as safe, missing bounds proof id, " +
					"trusted storage under escape, and report validation failure " +
					"block release promotion"),
				("go test ./compiler/internal/memoryfacts " +
					"./tools/cmd/validate-memory-report -run " +
					"'Unsafe|Bounds|Storage|Validate' -count=1"),
				"blocking cases preserve MemoryFactGraph truth and do not replace validators",
			),
			memoryFuzzRequirementRow(
				MemoryFuzzRequirementTier2NightlySeedTriage,
				memoryvocab.FuzzStatusBoundaryRecorded,
				"Tier 2 nightly fuzz preserves seeds, unstable triage, and minimized repro expectations",
				"bash scripts/dev/fuzz-nightly.sh --short --out-dir reports/fuzz-nightly-smoke",
				"Tier 2 is longer/nightly boundary evidence, not mandatory Tier 1 evidence",
			),
			memoryFuzzRequirementRow(
				MemoryFuzzRequirementTier3ReleasePassOrClassify,
				memoryvocab.FuzzStatusReleaseBlocking,
				("Tier 3 release-blocking focused memory fuzz must pass or " +
					"classify every failure before release promotion"),
				("go run ./tools/cmd/validate-memory-fuzz-oracle --report " +
					"reports/memory-fuzz-short/v12/memory-fuzz-oracle.json"),
				"Tier 3 blocks release promotion on unclassified failures without claiming target parity",
			),
		},
		SliceCoverage: memoryFuzzSliceCoverageRows(),
		Rows: []MemoryFuzzOracleRow{
			memoryFuzzOracleRow(
				MemoryFuzzOracleCheckerRejectExpected,
				"Checker reject expected",
				MemoryFuzzTier1ShortCI,
				MemoryFuzzOraclePass,
				[]string{
					("checker reject expected cases cover generated borrow escape," +
						" safe metadata mutation, and unsupported unsafe surface " +
						"diagnostics"),
				},
				[]string{
					("go test ./compiler/tests/safety/... " +
						"./compiler/tests/semantics -run " +
						"'Borrow|Escape|Metadata|Unsafe' -count=1"),
				},
				[]string{
					("checker reject expected is a passing oracle only when the " +
						"compiler rejects the generated program with the expected " +
						"diagnostic"),
				},
			),
			memoryFuzzOracleRow(
				MemoryFuzzOracleRuntimeTrapExpected,
				"Runtime trap expected",
				MemoryFuzzTier1ShortCI,
				MemoryFuzzOraclePass,
				[]string{
					("runtime trap expected cases reuse memory-production-smoke " +
						"bounds diagnostics for slice bounds, ptr_add bounds, and " +
						"raw-slice length overflow"),
				},
				[]string{
					"go test ./tools/cmd/memory-production-smoke -run 'RuntimeDiagnostic|Raw|Bounds' -count=1",
				},
				[]string{
					("runtime trap expected is limited to normal-build checks " +
						"that remain in the generated executable"),
				},
			),
			memoryFuzzOracleRow(
				MemoryFuzzOracleReferenceOutputExpected,
				"Compiled output equals interpreter/reference expected",
				MemoryFuzzTier1ShortCI,
				MemoryFuzzOraclePass,
				[]string{
					("compiled output equals interpreter/reference expected is " +
						"backed by differential.CheckBackendMatrix source " +
						"interpreter lanes for deterministic samples"),
				},
				[]string{
					"go test ./compiler/internal/differential -run 'CheckBackendMatrix' -count=1",
				},
				[]string{
					("reference equality is bounded to supported deterministic " +
						"samples and is not a full source interpreter claim"),
				},
			),
			memoryFuzzOracleRow(
				MemoryFuzzOracleCompilerCrashBug,
				"Compiler crash is bug",
				MemoryFuzzTier1ShortCI,
				MemoryFuzzOracleBug,
				[]string{
					("compiler crash is bug: generated parser/checker/lowering " +
						"fuzz entries must return diagnostics or valid artifacts; " +
						"panic/crash is never a passing oracle"),
				},
				[]string{
					("go test ./compiler/tests/fuzz -run " +
						"'FuzzLoweringPipelineVerifiesIR|FuzzFormatSourceIdempotent' " +
						"-count=1"),
				},
				[]string{
					("crash classification records a bug and does not promote the " +
						"generated program as passing evidence"),
				},
			),
			memoryFuzzOracleRow(
				MemoryFuzzOracleMiscompileBug,
				"Miscompile is bug",
				MemoryFuzzTier2Nightly,
				MemoryFuzzOracleBug,
				[]string{
					("miscompile is bug: differential mismatch between compiled " +
						"output and source/interpreter reference is reduced to a " +
						"reproducer"),
				},
				[]string{
					"go test ./compiler/internal/differential -run 'Reducer|CheckBackendMatrix' -count=1",
				},
				[]string{
					"miscompile classification is a failure artifact, not performance or correctness proof",
				},
			),
			memoryFuzzOracleRow(
				MemoryFuzzOracleUnsafeUnknownOptimizedAsSafeBug,
				"unsafe_unknown optimized as safe is bug",
				MemoryFuzzTier1ShortCI,
				MemoryFuzzOracleBug,
				[]string{
					("unsafe_unknown optimized as safe is bug: memoryfacts " +
						"rejects unsafe_unknown -> safe_known, no_alias, " +
						"bounds_check_eliminated, and trusted storage claims"),
				},
				[]string{
					("go test ./compiler/internal/memoryfacts -run " +
						"'UnsafeUnknown|SafeKnown|Optimization|TrustedStorage' " +
						"-count=1"),
				},
				[]string{
					"unsafe_unknown may stay checked, trapped, or conservative, but never becomes safe_known",
				},
			),
			memoryFuzzOracleRow(
				MemoryFuzzOracleReportValidationFailureBug,
				"Report validation failure is bug",
				MemoryFuzzTier1ShortCI,
				MemoryFuzzOracleBug,
				[]string{
					("report validation failure is bug: MemoryFactGraph " +
						"validation rejects invalid memory reports before artifact " +
						"emission"),
				},
				[]string{
					("go test ./compiler/internal/memoryfacts " +
						"./tools/cmd/validate-memory-report -run " +
						"'ValidateMemoryReport|Cost|Unsafe' -count=1"),
				},
				[]string{
					("reports validate against MemoryFactGraph and the MPC-14 " +
						"cost model rather than report-reconstructed truth"),
				},
			),
		},
		Invariants: []MemoryFuzzInvariantRow{
			memoryFuzzInvariantRow(
				MemoryFuzzInvariantNoSafeMetadataMutation,
				"safe representation metadata is not user-assignable",
				"go test ./compiler/tests/semantics -run 'Metadata' -count=1",
			),
			memoryFuzzInvariantRow(
				MemoryFuzzInvariantNoBorrowedEscape,
				"borrowed values cannot escape return/actor/task boundaries without checked copy/transfer",
				("go test ./compiler/tests/safety/... " +
					"./compiler/tests/ownership -run 'Borrow|Escape|Actor|Task' " +
					"-count=1"),
			),
			memoryFuzzInvariantRow(
				MemoryFuzzInvariantNoUnsafeUnknownToSafeKnown,
				"unsafe_unknown rows cannot become safe_known or safe_borrowed proof rows",
				"go test ./compiler/internal/memoryfacts -run 'UnsafeUnknown|SafeKnown|SafeBorrowed' -count=1",
			),
			memoryFuzzInvariantRow(
				MemoryFuzzInvariantNoBoundsRemovalWithoutProofID,
				"bounds check removal requires compiler-owned proof id and validated report evidence",
				"go test ./compiler/internal/memoryfacts ./compiler -run 'Bounds|Proof|MemoryReport' -count=1",
			),
			memoryFuzzInvariantRow(
				MemoryFuzzInvariantNoStackRegionStorageWhenEscaped,
				("stack/region storage claims are rejected when escape " +
					"evidence forces heap or conservative fallback"),
				("go test ./compiler/internal/memoryfacts " +
					"./compiler/internal/validation -run " +
					"'Storage|Escape|Region|HeapFallback' -count=1"),
			),
			memoryFuzzInvariantRow(
				MemoryFuzzInvariantReportsValidateAgainstFactGraph,
				("memory reports validate against MemoryFactGraph during " +
					"compiler report emission and CLI validation"),
				"go test ./compiler/internal/memoryfacts ./tools/cmd/validate-memory-report -count=1",
			),
			memoryFuzzInvariantRow(
				MemoryFuzzInvariantReportsPreserveMemoryCostModel,
				("memory report rows preserve cost_class and " +
					"normal_build_check rules from the MPC-14 cost model"),
				("go test ./compiler/internal/memoryfacts " +
					"./tools/cmd/validate-memory-report -run " +
					"'Cost|Dynamic|Unsafe' -count=1"),
			),
		},
		GeneratorSurfaces: []MemoryFuzzGeneratorSurfaceRow{
			{
				Tier:   MemoryFuzzGeneratorTier1SupportedNow,
				Status: memoryvocab.FuzzStatusCovered,
				Surface: []string{
					"slices",
					"Strings",
					"borrow/copy",
					"simple structs/enums/optionals",
					"safe views",
					"make_*",
					"explicit islands",
				},
				Boundaries: []string{
					"Tier 1 short CI smoke uses deterministic generated samples only",
				},
			},
			{
				Tier:   MemoryFuzzGeneratorTier2SupportedNarrow,
				Status: memoryvocab.FuzzStatusBoundaryRecorded,
				Surface: []string{
					"generics",
					"function-typed borrowed returns",
					"async/task boundary smoke",
					"raw verified roots",
				},
				Boundaries: []string{
					"Tier 2 nightly fuzz may expand these narrow supported surfaces with bounded seeds",
				},
			},
			{
				Tier:   MemoryFuzzGeneratorTier3ConservativeRejected,
				Status: memoryvocab.FuzzStatusBoundaryRecorded,
				Surface: []string{
					"arbitrary unsafe pointers",
					"unknown external calls",
					"unsupported target behavior",
				},
				Boundaries: []string{
					"Tier 3 records conservative/rejected outcomes instead of upgrading safety claims",
				},
			},
			{
				Tier:   MemoryFuzzGeneratorTier4Future,
				Status: memoryvocab.FuzzStatusFuture,
				Surface: []string{
					"full FFI lifetime",
					"full actor zero-copy runtime",
					"generic lifetimes",
				},
				Boundaries: []string{"future-only scope is not a current production claim"},
			},
		},
		BlockingCases: []MemoryFuzzBlockingCaseRow{
			memoryFuzzBlockingCaseRow(
				MemoryFuzzBlockingUnsafeUnknownOptimizedAsSafe,
				"unsafe_unknown optimized as safe remains a release-blocking oracle bug",
				"go test ./compiler/internal/memoryfacts -run 'UnsafeUnknown|SafeKnown|Optimization' -count=1",
				"unsafe_unknown may remain checked, trapped, or conservative, but never becomes safe_known",
			),
			memoryFuzzBlockingCaseRow(
				MemoryFuzzBlockingBoundsCheckWithoutProofID,
				"bounds_check_eliminated without a compiler-owned proof id remains release-blocking",
				"go test ./compiler/internal/memoryfacts ./compiler -run 'Bounds|Proof|MemoryReport' -count=1",
				"bounds removal evidence must preserve proof ids from the compiler-owned graph",
			),
			memoryFuzzBlockingCaseRow(
				MemoryFuzzBlockingTrustedStorageUnderEscape,
				"stack, region, or trusted storage under escape remains release-blocking",
				("go test ./compiler/internal/memoryfacts " +
					"./compiler/internal/validation -run " +
					"'Storage|Escape|Region|HeapFallback' -count=1"),
				"escaped values require heap, conservative, or rejected classification",
			),
			memoryFuzzBlockingCaseRow(
				MemoryFuzzBlockingReportValidationFailure,
				"memory report validation failure remains release-blocking before artifact promotion",
				("go test ./compiler/internal/memoryfacts " +
					"./tools/cmd/validate-memory-report -run " +
					"'ValidateMemoryReport|Cost|Unsafe' -count=1"),
				"report rows are projections and cannot reconstruct MemoryFactGraph truth",
			),
		},
		TierPolicies: []MemoryFuzzTierPolicyRow{
			{
				Tier:           MemoryFuzzTier1ShortCI,
				Status:         memoryvocab.FuzzStatusCovered,
				SeedsPreserved: true,
				Evidence: []string{
					"Tier 1 uses deterministic v0-v11 smoke cases and writes release-evidence artifacts",
				},
				Tests: []string{
					"go run ./tools/cmd/memory-fuzz-short --tier=1 --report-dir reports/memory-fuzz-short/v12",
				},
				Boundaries: []string{
					"Tier 1 is short deterministic smoke, not nightly fuzz or exhaustive proof",
				},
			},
			{
				Tier:                        MemoryFuzzTier2Nightly,
				Status:                      memoryvocab.FuzzStatusBoundaryRecorded,
				SeedsPreserved:              true,
				UnstableTriageRequired:      true,
				MinimizedReproducerRequired: true,
				Evidence: []string{
					("Tier 2 nightly fuzz preserves seeds, unstable triage, and " +
						"minimized repros using the fuzz property stress protocol"),
				},
				Tests: []string{
					"bash scripts/dev/fuzz-nightly.sh --short --out-dir reports/fuzz-nightly-smoke",
				},
				Boundaries: []string{
					"Tier 2 is nightly/release-candidate evidence and is not required as deterministic Tier 1",
				},
			},
			{
				Tier:                                   MemoryFuzzTier3ReleaseFocused,
				Status:                                 memoryvocab.FuzzStatusReleaseBlocking,
				SeedsPreserved:                         true,
				UnstableTriageRequired:                 true,
				MinimizedReproducerRequired:            true,
				ReleasePromotionBlockedUntilClassified: true,
				Evidence: []string{
					"Tier 3 focused memory fuzz must pass or classify every failure before release promotion",
				},
				Tests: []string{
					("go run ./tools/cmd/validate-memory-fuzz-oracle --report " +
						"reports/memory-fuzz-short/v12/memory-fuzz-oracle.json"),
				},
				Boundaries: []string{
					"Tier 3 release blocking is classification evidence, not target parity or runtime ABI proof",
				},
			},
		},
		Artifacts: []MemoryFuzzArtifact{
			{
				Path:     "reports/memory-fuzz-short/<slice>/memory-fuzz-oracle.json",
				Kind:     "tier1_short_ci_smoke_report",
				Required: true,
			},
			{
				Path:     "reports/memory-fuzz-short/<slice>/summary.md",
				Kind:     "tier1_short_ci_smoke_summary",
				Required: true,
			},
			{
				Path:     "reports/memory-fuzz-short/<slice>/summary.json",
				Kind:     "tier1_short_ci_smoke_summary_json",
				Required: true,
			},
			{
				Path:     "reports/memory-fuzz-short/<slice>/reproducers/compiler-crash/",
				Kind:     "compiler_crash_reproducer",
				Required: true,
			},
			{
				Path:     "reports/memory-fuzz-short/<slice>/reproducers/miscompile/",
				Kind:     "miscompile_reproducer",
				Required: true,
			},
			{
				Path:     "reports/memory-fuzz-short/<slice>/reducers/miscompile/",
				Kind:     "miscompile_reducer",
				Required: true,
			},
			{
				Path:     "docs/audits/memory/islands/memory-fuzz-oracle-v1.md",
				Kind:     "audit_contract",
				Required: true,
			},
		},
		NonClaims: []string{
			"no exhaustive fuzzing is claimed",
			"no exhaustive fuzz proof is claimed",
			"no unsupported unsafe pointer safety is claimed",
			"no arbitrary unsafe safety is claimed",
			"no full runtime/ABI/target parity proof is claimed",
			"no full program correctness claim is made",
			"no runtime behavior change",
			"no safe-program semantics change",
			"no performance claim is made",
			"no clean-release claim under dirty worktree",
			"no replacement for MemoryFactGraph validators",
			"no Memory 100% claim is made",
		},
	}, nil
}

func memoryFuzzRequirementRow(
	id MemoryFuzzRequirementID,
	status string,
	evidence string,
	test string,
	boundary string,
) MemoryFuzzRequirementRow {
	return MemoryFuzzRequirementRow{
		ID:         id,
		Status:     status,
		Evidence:   []string{evidence},
		Tests:      []string{test},
		Boundaries: []string{boundary},
	}
}

func memoryFuzzOracleRow(
	category MemoryFuzzOracleCategory,
	name string,
	tier MemoryFuzzTier,
	result MemoryFuzzOracleResult,
	evidence []string,
	tests []string,
	boundaries []string,
) MemoryFuzzOracleRow {
	return MemoryFuzzOracleRow{
		Category:       category,
		Name:           name,
		Tier:           tier,
		ExpectedResult: result,
		Status:         memoryvocab.FuzzStatusCovered,
		Evidence:       evidence,
		Tests:          tests,
		Boundaries:     boundaries,
	}
}

func memoryFuzzInvariantRow(
	id MemoryFuzzInvariantID,
	evidence string,
	test string,
) MemoryFuzzInvariantRow {
	return MemoryFuzzInvariantRow{
		ID:       id,
		Status:   memoryvocab.FuzzStatusCovered,
		Evidence: []string{evidence},
		Tests:    []string{test},
		Boundaries: []string{
			"checked for generated programs before a fuzz result is promoted as passing evidence",
		},
	}
}

func memoryFuzzSliceCoverageRows() []MemoryFuzzSliceCoverageRow {
	return []MemoryFuzzSliceCoverageRow{
		memoryFuzzSliceCoverageRow(
			"v0",
			[]string{"metadata", "borrow", "narrow noalias"},
			[]MemoryFuzzOracleCategory{
				MemoryFuzzOracleCheckerRejectExpected,
				MemoryFuzzOracleReferenceOutputExpected,
			},
			[]MemoryFuzzInvariantID{
				MemoryFuzzInvariantNoSafeMetadataMutation,
				MemoryFuzzInvariantNoBorrowedEscape,
				MemoryFuzzInvariantReportsValidateAgainstFactGraph,
			},
			"go test ./compiler/internal/memoryfacts -run 'MemoryIdealV0|NoAlias|Metadata|Borrow' -count=1",
		),
		memoryFuzzSliceCoverageRow(
			"v1",
			[]string{"enum payload borrow carriers", "generic wrapper borrow carriers"},
			[]MemoryFuzzOracleCategory{
				MemoryFuzzOracleCheckerRejectExpected,
				MemoryFuzzOracleReferenceOutputExpected,
			},
			[]MemoryFuzzInvariantID{
				MemoryFuzzInvariantNoBorrowedEscape,
				MemoryFuzzInvariantReportsValidateAgainstFactGraph,
			},
			"go test ./compiler/internal/memoryfacts -run 'MemoryIdealV1|Enum|Generic|Borrow' -count=1",
		),
		memoryFuzzSliceCoverageRow(
			"v2",
			[]string{"callbacks", "function values", "borrowed callable returns"},
			[]MemoryFuzzOracleCategory{
				MemoryFuzzOracleCheckerRejectExpected,
				MemoryFuzzOracleReferenceOutputExpected,
			},
			[]MemoryFuzzInvariantID{
				MemoryFuzzInvariantNoBorrowedEscape,
				MemoryFuzzInvariantReportsValidateAgainstFactGraph,
			},
			"go test ./compiler/internal/memoryfacts -run 'MemoryIdealV2|Callback|Function' -count=1",
		),
		memoryFuzzSliceCoverageRow(
			"v3",
			[]string{
				"protocol dispatch",
				"interface borrow carriers",
				"dynamic dispatch conservatism",
			},
			[]MemoryFuzzOracleCategory{
				MemoryFuzzOracleCheckerRejectExpected,
				MemoryFuzzOracleUnsafeUnknownOptimizedAsSafeBug,
			},
			[]MemoryFuzzInvariantID{
				MemoryFuzzInvariantNoBorrowedEscape,
				MemoryFuzzInvariantNoUnsafeUnknownToSafeKnown,
			},
			("go test ./compiler/internal/memoryfacts -run " +
				"'MemoryIdealV3|Protocol|Interface|Dynamic' -count=1"),
		),
		memoryFuzzSliceCoverageRow(
			"v4",
			[]string{"async boundary", "task boundary", "actor boundary"},
			[]MemoryFuzzOracleCategory{
				MemoryFuzzOracleCheckerRejectExpected,
				MemoryFuzzOracleReportValidationFailureBug,
			},
			[]MemoryFuzzInvariantID{
				MemoryFuzzInvariantNoBorrowedEscape,
				MemoryFuzzInvariantReportsValidateAgainstFactGraph,
			},
			"go test ./compiler/internal/memoryfacts -run 'MemoryIdealV4|Async|Task|Actor' -count=1",
		),
		memoryFuzzSliceCoverageRow(
			"v5",
			[]string{"unsafe gateway", "raw pointer conservatism", "unsafe_unknown provenance"},
			[]MemoryFuzzOracleCategory{
				MemoryFuzzOracleCheckerRejectExpected,
				MemoryFuzzOracleUnsafeUnknownOptimizedAsSafeBug,
			},
			[]MemoryFuzzInvariantID{
				MemoryFuzzInvariantNoUnsafeUnknownToSafeKnown,
				MemoryFuzzInvariantReportsValidateAgainstFactGraph,
			},
			"go test ./compiler/internal/memoryfacts -run 'MemoryIdealV5|Unsafe|Raw|Pointer' -count=1",
		),
		memoryFuzzSliceCoverageRow(
			"v6",
			[]string{"bounds proof ids", "bounds check elimination rejection", "proof source ids"},
			[]MemoryFuzzOracleCategory{
				MemoryFuzzOracleRuntimeTrapExpected,
				MemoryFuzzOracleReportValidationFailureBug,
			},
			[]MemoryFuzzInvariantID{
				MemoryFuzzInvariantNoBoundsRemovalWithoutProofID,
				MemoryFuzzInvariantReportsValidateAgainstFactGraph,
			},
			"go test ./compiler/internal/memoryfacts ./compiler -run 'MemoryIdealV6|Bounds|Proof' -count=1",
		),
		memoryFuzzSliceCoverageRow(
			"v7",
			[]string{"FFI quarantine", "external call provenance", "raw verified roots"},
			[]MemoryFuzzOracleCategory{
				MemoryFuzzOracleCheckerRejectExpected,
				MemoryFuzzOracleUnsafeUnknownOptimizedAsSafeBug,
			},
			[]MemoryFuzzInvariantID{
				MemoryFuzzInvariantNoUnsafeUnknownToSafeKnown,
				MemoryFuzzInvariantReportsValidateAgainstFactGraph,
			},
			"go test ./compiler/internal/memoryfacts -run 'MemoryIdealV7|FFI|External|Raw' -count=1",
		),
		memoryFuzzSliceCoverageRow(
			"v8",
			[]string{"memory report integrity", "cost model projection", "normal build checks"},
			[]MemoryFuzzOracleCategory{MemoryFuzzOracleReportValidationFailureBug},
			[]MemoryFuzzInvariantID{
				MemoryFuzzInvariantReportsValidateAgainstFactGraph,
				MemoryFuzzInvariantReportsPreserveMemoryCostModel,
			},
			("go test ./compiler/internal/memoryfacts " +
				"./tools/cmd/validate-memory-report -run " +
				"'Report|Cost|NormalBuild' -count=1"),
		),
		memoryFuzzSliceCoverageRow(
			"v9",
			[]string{"storage lowering", "escape-aware heap fallback", "trusted storage rejection"},
			[]MemoryFuzzOracleCategory{
				MemoryFuzzOracleCheckerRejectExpected,
				MemoryFuzzOracleReportValidationFailureBug,
			},
			[]MemoryFuzzInvariantID{
				MemoryFuzzInvariantNoStackRegionStorageWhenEscaped,
				MemoryFuzzInvariantReportsValidateAgainstFactGraph,
			},
			("go test ./compiler/internal/memoryfacts " +
				"./compiler/internal/validation -run " +
				"'Storage|Escape|Lower|HeapFallback' -count=1"),
		),
		memoryFuzzSliceCoverageRow(
			"v10",
			[]string{
				"async cancellation",
				"task group boundary",
				"actor reentrant callback boundary",
			},
			[]MemoryFuzzOracleCategory{
				MemoryFuzzOracleCheckerRejectExpected,
				MemoryFuzzOracleReportValidationFailureBug,
			},
			[]MemoryFuzzInvariantID{
				MemoryFuzzInvariantNoBorrowedEscape,
				MemoryFuzzInvariantNoStackRegionStorageWhenEscaped,
			},
			("go test ./compiler/internal/memoryfacts " +
				"./compiler/internal/memorymodel -run " +
				"'Async|Task|Actor|Cancel' -count=1"),
		),
		memoryFuzzSliceCoverageRow(
			"v11",
			[]string{
				"dynamic protocol",
				"existential borrow carrier",
				"witness/conformance table conservatism",
			},
			[]MemoryFuzzOracleCategory{
				MemoryFuzzOracleCheckerRejectExpected,
				MemoryFuzzOracleUnsafeUnknownOptimizedAsSafeBug,
			},
			[]MemoryFuzzInvariantID{
				MemoryFuzzInvariantNoBorrowedEscape,
				MemoryFuzzInvariantNoUnsafeUnknownToSafeKnown,
			},
			("go test ./compiler/internal/memoryfacts " +
				"./compiler/internal/memorymodel -run " +
				"'Dynamic|Protocol|Witness|Conformance' -count=1"),
		),
	}
}

func memoryFuzzSliceCoverageRow(
	sliceID string,
	surface []string,
	categories []MemoryFuzzOracleCategory,
	invariants []MemoryFuzzInvariantID,
	test string,
) MemoryFuzzSliceCoverageRow {
	return MemoryFuzzSliceCoverageRow{
		SliceID:          sliceID,
		Status:           memoryvocab.FuzzStatusCovered,
		Surface:          surface,
		OracleCategories: categories,
		Invariants:       invariants,
		Evidence: []string{
			"deterministic Tier 1 memory fuzz oracle coverage is recorded for " + sliceID,
		},
		Tests: []string{test},
		Boundaries: []string{
			"coverage is limited to supported compiler-visible " + sliceID + (" memory evidence and is " +
				"not exhaustive fuzz proof"),
		},
	}
}

func memoryFuzzBlockingCaseRow(
	id MemoryFuzzBlockingCaseID,
	evidence string,
	test string,
	boundary string,
) MemoryFuzzBlockingCaseRow {
	return MemoryFuzzBlockingCaseRow{
		ID:            id,
		Status:        memoryvocab.FuzzStatusBlocksRelease,
		BlocksRelease: true,
		Evidence:      []string{evidence},
		Tests:         []string{test},
		Boundaries:    []string{boundary},
	}
}

func ValidateMemoryFuzzOracleReport(report MemoryFuzzOracleReport) error {
	var issues []string
	if report.SchemaVersion != MemoryFuzzOracleSchemaV1 {
		issues = append(
			issues,
			fmt.Sprintf(
				"schema_version = %q, want %q",
				report.SchemaVersion,
				MemoryFuzzOracleSchemaV1,
			),
		)
	}
	if report.Scope != MemoryFuzzOracleScopeMPC15 {
		issues = append(
			issues,
			fmt.Sprintf("scope = %q, want %q", report.Scope, MemoryFuzzOracleScopeMPC15),
		)
	}
	if report.Tier1ShortCISmokeCases <= 0 {
		issues = append(issues, "Tier 1 short CI smoke cases are required")
	}
	if !report.Tier2NightlyBoundaryRecorded {
		issues = append(issues, "Tier 2 nightly fuzz boundary is required")
	}
	if !report.Tier3ReleaseBlockingBoundaryRecorded {
		issues = append(issues, "Tier 3 release-blocking focused memory fuzz boundary is required")
	}
	issues = append(issues, validateMemoryFuzzRequirements(report.Requirements)...)
	issues = append(issues, validateMemoryFuzzSliceCoverage(report.SliceCoverage)...)
	issues = append(issues, validateMemoryFuzzOracleRows(report.Rows)...)
	issues = append(issues, validateMemoryFuzzInvariants(report.Invariants)...)
	issues = append(issues, validateMemoryFuzzGeneratorSurfaces(report.GeneratorSurfaces)...)
	issues = append(issues, validateMemoryFuzzBlockingCases(report.BlockingCases)...)
	issues = append(issues, validateMemoryFuzzTierPolicies(report.TierPolicies)...)
	if len(report.Artifacts) == 0 {
		issues = append(issues, "memory fuzz oracle artifacts are required")
	}
	issues = append(issues, validateMemoryFuzzRequiredArtifacts(report.Artifacts)...)
	for _, want := range []string{
		"no exhaustive fuzzing is claimed",
		"no exhaustive fuzz proof is claimed",
		"no unsupported unsafe pointer safety is claimed",
		"no arbitrary unsafe safety is claimed",
		"no full runtime/ABI/target parity proof is claimed",
		"no runtime behavior change",
		"no safe-program semantics change",
		"no performance claim is made",
		"no clean-release claim under dirty worktree",
		"no replacement for MemoryFactGraph validators",
		"no Memory 100% claim is made",
	} {
		if !memoryFuzzHasString(report.NonClaims, want) {
			issues = append(issues, fmt.Sprintf("missing non-claim %q", want))
		}
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateMemoryFuzzRequirements(rows []MemoryFuzzRequirementRow) []string {
	seen := map[MemoryFuzzRequirementID]bool{}
	var issues []string
	expected := map[MemoryFuzzRequirementID]string{
		MemoryFuzzRequirementTier1V0V11Coverage:         memoryvocab.FuzzStatusValidatedNarrow,
		MemoryFuzzRequirementCrashMiscompileArtifacts:   memoryvocab.FuzzStatusValidatedNarrow,
		MemoryFuzzRequirementBlockingMemoryFailures:     memoryvocab.FuzzStatusReleaseBlocking,
		MemoryFuzzRequirementTier2NightlySeedTriage:     memoryvocab.FuzzStatusBoundaryRecorded,
		MemoryFuzzRequirementTier3ReleasePassOrClassify: memoryvocab.FuzzStatusReleaseBlocking,
	}
	for _, row := range rows {
		if !knownMemoryFuzzRequirementID(row.ID) {
			issues = append(issues, fmt.Sprintf("unknown requirement %q", row.ID))
		}
		if !knownMemoryFuzzStatus(row.Status) {
			issues = append(
				issues,
				fmt.Sprintf("unknown requirement status %q for %s", row.Status, row.ID),
			)
		}
		if seen[row.ID] {
			issues = append(issues, fmt.Sprintf("duplicate requirement %s", row.ID))
		}
		seen[row.ID] = true
		if want := expected[row.ID]; row.Status != want {
			issues = append(
				issues,
				fmt.Sprintf("requirement %s status = %q, want %q", row.ID, row.Status, want),
			)
		}
		issues = append(
			issues,
			validateMemoryFuzzTextList("requirement "+string(row.ID)+" evidence", row.Evidence)...)
		issues = append(
			issues,
			validateMemoryFuzzTextList("requirement "+string(row.ID)+" tests", row.Tests)...)
		issues = append(
			issues,
			validateMemoryFuzzTextList(
				"requirement "+string(row.ID)+" boundaries",
				row.Boundaries,
			)...)
	}
	for _, id := range memoryFuzzRequirementIDs() {
		if !seen[id] {
			issues = append(issues, fmt.Sprintf("missing requirement %s", id))
		}
	}
	return issues
}

func validateMemoryFuzzSliceCoverage(rows []MemoryFuzzSliceCoverageRow) []string {
	seen := map[string]bool{}
	var issues []string
	for _, row := range rows {
		if !knownMemoryFuzzSliceID(row.SliceID) {
			issues = append(issues, fmt.Sprintf("unknown slice coverage %q", row.SliceID))
		}
		if seen[row.SliceID] {
			issues = append(issues, fmt.Sprintf("duplicate slice coverage %s", row.SliceID))
		}
		seen[row.SliceID] = true
		if !knownMemoryFuzzStatus(row.Status) {
			issues = append(
				issues,
				fmt.Sprintf("unknown slice coverage status %q for %s", row.Status, row.SliceID),
			)
		}
		if row.Status != memoryvocab.FuzzStatusCovered {
			issues = append(
				issues,
				fmt.Sprintf("slice coverage %s status = %q, want covered", row.SliceID, row.Status),
			)
		}
		issues = append(
			issues,
			validateMemoryFuzzTextList("slice coverage "+row.SliceID+" surface", row.Surface)...)
		if len(row.OracleCategories) == 0 {
			issues = append(
				issues,
				fmt.Sprintf("slice coverage %s oracle_categories are required", row.SliceID),
			)
		}
		for _, category := range row.OracleCategories {
			if !knownMemoryFuzzOracleCategory(category) {
				issues = append(
					issues,
					fmt.Sprintf(
						"slice coverage %s unknown oracle_category %q",
						row.SliceID,
						category,
					),
				)
			}
		}
		if len(row.Invariants) == 0 {
			issues = append(
				issues,
				fmt.Sprintf("slice coverage %s invariants are required", row.SliceID),
			)
		}
		for _, id := range row.Invariants {
			if !knownMemoryFuzzInvariantID(id) {
				issues = append(
					issues,
					fmt.Sprintf("slice coverage %s unknown invariant %q", row.SliceID, id),
				)
			}
		}
		issues = append(
			issues,
			validateMemoryFuzzTextList("slice coverage "+row.SliceID+" evidence", row.Evidence)...)
		issues = append(
			issues,
			validateMemoryFuzzTextList("slice coverage "+row.SliceID+" tests", row.Tests)...)
		issues = append(
			issues,
			validateMemoryFuzzTextList(
				"slice coverage "+row.SliceID+" boundaries",
				row.Boundaries,
			)...)
	}
	for _, id := range memoryFuzzSliceIDs() {
		if !seen[id] {
			issues = append(issues, fmt.Sprintf("missing slice coverage %s", id))
		}
	}
	return issues
}

func validateMemoryFuzzOracleRows(rows []MemoryFuzzOracleRow) []string {
	seen := map[MemoryFuzzOracleCategory]bool{}
	var issues []string
	for _, row := range rows {
		if row.Category == "" {
			issues = append(issues, "oracle_category is required")
			continue
		}
		if !knownMemoryFuzzOracleCategory(row.Category) {
			issues = append(issues, fmt.Sprintf("unknown oracle_category %q", row.Category))
		}
		if seen[row.Category] {
			issues = append(issues, fmt.Sprintf("duplicate oracle_category %s", row.Category))
		}
		seen[row.Category] = true
		if strings.TrimSpace(row.Name) == "" {
			issues = append(
				issues,
				fmt.Sprintf("oracle_category %s name is required", row.Category),
			)
		}
		if !knownMemoryFuzzTier(row.Tier) {
			issues = append(
				issues,
				fmt.Sprintf("oracle_category %s unknown tier %q", row.Category, row.Tier),
			)
		}
		if !knownMemoryFuzzStatus(row.Status) {
			issues = append(
				issues,
				fmt.Sprintf("unknown oracle row status %q for %s", row.Status, row.Category),
			)
		}
		if row.Status != memoryvocab.FuzzStatusCovered {
			issues = append(
				issues,
				fmt.Sprintf(
					"oracle_category %s status = %q, want covered",
					row.Category,
					row.Status,
				),
			)
		}
		if row.ExpectedResult != expectedMemoryFuzzOracleResult(row.Category) {
			issues = append(
				issues,
				fmt.Sprintf(
					"oracle_category %s expected_result = %q, want %q",
					row.Category,
					row.ExpectedResult,
					expectedMemoryFuzzOracleResult(row.Category),
				),
			)
		}
		issues = append(
			issues,
			validateMemoryFuzzTextList(
				"oracle_category "+string(row.Category)+" evidence",
				row.Evidence,
			)...)
		issues = append(
			issues,
			validateMemoryFuzzTextList(
				"oracle_category "+string(row.Category)+" tests",
				row.Tests,
			)...)
		issues = append(
			issues,
			validateMemoryFuzzTextList(
				"oracle_category "+string(row.Category)+" boundaries",
				row.Boundaries,
			)...)
	}
	for _, category := range memoryFuzzOracleCategories() {
		if !seen[category] {
			issues = append(issues, fmt.Sprintf("missing oracle_category %s", category))
		}
	}
	return issues
}

func validateMemoryFuzzInvariants(rows []MemoryFuzzInvariantRow) []string {
	seen := map[MemoryFuzzInvariantID]bool{}
	var issues []string
	for _, row := range rows {
		if !knownMemoryFuzzInvariantID(row.ID) {
			issues = append(issues, fmt.Sprintf("unknown invariant %q", row.ID))
		}
		if seen[row.ID] {
			issues = append(issues, fmt.Sprintf("duplicate invariant %s", row.ID))
		}
		seen[row.ID] = true
		if !knownMemoryFuzzStatus(row.Status) {
			issues = append(
				issues,
				fmt.Sprintf("unknown invariant status %q for %s", row.Status, row.ID),
			)
		}
		if row.Status != memoryvocab.FuzzStatusCovered {
			issues = append(
				issues,
				fmt.Sprintf("invariant %s status = %q, want covered", row.ID, row.Status),
			)
		}
		issues = append(
			issues,
			validateMemoryFuzzTextList("invariant "+string(row.ID)+" evidence", row.Evidence)...)
		issues = append(
			issues,
			validateMemoryFuzzTextList("invariant "+string(row.ID)+" tests", row.Tests)...)
		issues = append(
			issues,
			validateMemoryFuzzTextList(
				"invariant "+string(row.ID)+" boundaries",
				row.Boundaries,
			)...)
	}
	for _, id := range memoryFuzzInvariantIDs() {
		if !seen[id] {
			issues = append(issues, fmt.Sprintf("missing invariant %s", id))
		}
	}
	return issues
}

func validateMemoryFuzzGeneratorSurfaces(rows []MemoryFuzzGeneratorSurfaceRow) []string {
	seen := map[MemoryFuzzGeneratorSurfaceTier]bool{}
	var issues []string
	for _, row := range rows {
		if !knownMemoryFuzzGeneratorSurfaceTier(row.Tier) {
			issues = append(issues, fmt.Sprintf("unknown generator surface tier %q", row.Tier))
		}
		if seen[row.Tier] {
			issues = append(issues, fmt.Sprintf("duplicate generator surface tier %s", row.Tier))
		}
		seen[row.Tier] = true
		if strings.TrimSpace(row.Status) == "" {
			issues = append(
				issues,
				fmt.Sprintf("generator surface tier %s status is required", row.Tier),
			)
		} else if !knownMemoryFuzzStatus(row.Status) {
			issues = append(
				issues,
				fmt.Sprintf("unknown generator surface status %q for %s", row.Status, row.Tier),
			)
		}
		issues = append(
			issues,
			validateMemoryFuzzTextList(
				"generator surface tier "+string(row.Tier)+" surface",
				row.Surface,
			)...)
		issues = append(
			issues,
			validateMemoryFuzzTextList(
				"generator surface tier "+string(row.Tier)+" boundaries",
				row.Boundaries,
			)...)
	}
	for _, tier := range memoryFuzzGeneratorSurfaceTiers() {
		if !seen[tier] {
			issues = append(issues, fmt.Sprintf("missing generator surface tier %s", tier))
		}
	}
	return issues
}

func validateMemoryFuzzBlockingCases(rows []MemoryFuzzBlockingCaseRow) []string {
	seen := map[MemoryFuzzBlockingCaseID]bool{}
	var issues []string
	for _, row := range rows {
		if !knownMemoryFuzzBlockingCaseID(row.ID) {
			issues = append(issues, fmt.Sprintf("unknown blocking case %q", row.ID))
		}
		if seen[row.ID] {
			issues = append(issues, fmt.Sprintf("duplicate blocking case %s", row.ID))
		}
		seen[row.ID] = true
		if !knownMemoryFuzzStatus(row.Status) {
			issues = append(
				issues,
				fmt.Sprintf("unknown blocking case status %q for %s", row.Status, row.ID),
			)
		}
		if row.Status != memoryvocab.FuzzStatusBlocksRelease {
			issues = append(
				issues,
				fmt.Sprintf(
					"blocking case %s status = %q, want blocks_release",
					row.ID,
					row.Status,
				),
			)
		}
		if !row.BlocksRelease {
			issues = append(issues, fmt.Sprintf("blocking case %s must set blocks_release", row.ID))
		}
		issues = append(
			issues,
			validateMemoryFuzzTextList(
				"blocking case "+string(row.ID)+" evidence",
				row.Evidence,
			)...)
		issues = append(
			issues,
			validateMemoryFuzzTextList("blocking case "+string(row.ID)+" tests", row.Tests)...)
		issues = append(
			issues,
			validateMemoryFuzzTextList(
				"blocking case "+string(row.ID)+" boundaries",
				row.Boundaries,
			)...)
	}
	for _, id := range memoryFuzzBlockingCaseIDs() {
		if !seen[id] {
			issues = append(issues, fmt.Sprintf("missing blocking case %s", id))
		}
	}
	return issues
}

func validateMemoryFuzzTierPolicies(rows []MemoryFuzzTierPolicyRow) []string {
	seen := map[MemoryFuzzTier]bool{}
	var issues []string
	for _, row := range rows {
		if !knownMemoryFuzzTier(row.Tier) {
			issues = append(issues, fmt.Sprintf("unknown tier policy %q", row.Tier))
		}
		if seen[row.Tier] {
			issues = append(issues, fmt.Sprintf("duplicate tier policy %s", row.Tier))
		}
		seen[row.Tier] = true
		if strings.TrimSpace(row.Status) == "" {
			issues = append(issues, fmt.Sprintf("tier policy %s status is required", row.Tier))
		} else if !knownMemoryFuzzStatus(row.Status) {
			issues = append(
				issues,
				fmt.Sprintf("unknown tier policy status %q for %s", row.Status, row.Tier),
			)
		}
		switch row.Tier {
		case MemoryFuzzTier1ShortCI:
			if row.Status != memoryvocab.FuzzStatusCovered {
				issues = append(
					issues,
					fmt.Sprintf("Tier 1 short CI smoke status = %q, want covered", row.Status),
				)
			}
		case MemoryFuzzTier2Nightly:
			if row.Status != memoryvocab.FuzzStatusBoundaryRecorded {
				issues = append(
					issues,
					fmt.Sprintf(
						"Tier 2 nightly fuzz status = %q, want boundary_recorded",
						row.Status,
					),
				)
			}
			if !row.SeedsPreserved {
				issues = append(issues, "Tier 2 nightly fuzz seed preservation is required")
			}
			if !row.UnstableTriageRequired {
				issues = append(issues, "Tier 2 nightly fuzz unstable triage is required")
			}
			if !row.MinimizedReproducerRequired {
				issues = append(issues, "Tier 2 nightly fuzz minimized repro is required")
			}
		case MemoryFuzzTier3ReleaseFocused:
			if row.Status != memoryvocab.FuzzStatusReleaseBlocking {
				issues = append(
					issues,
					fmt.Sprintf(
						"Tier 3 release-blocking memory fuzz status = %q, want release_blocking",
						row.Status,
					),
				)
			}
			if !row.ReleasePromotionBlockedUntilClassified {
				issues = append(
					issues,
					"Tier 3 release-blocking memory fuzz must block promotion until every failure is classified",
				)
			}
			if !row.MinimizedReproducerRequired {
				issues = append(
					issues,
					"Tier 3 release-blocking memory fuzz minimized repro is required",
				)
			}
		}
		issues = append(
			issues,
			validateMemoryFuzzTextList(
				"tier policy "+string(row.Tier)+" evidence",
				row.Evidence,
			)...)
		issues = append(
			issues,
			validateMemoryFuzzTextList("tier policy "+string(row.Tier)+" tests", row.Tests)...)
		issues = append(
			issues,
			validateMemoryFuzzTextList(
				"tier policy "+string(row.Tier)+" boundaries",
				row.Boundaries,
			)...)
	}
	for _, tier := range []MemoryFuzzTier{
		MemoryFuzzTier1ShortCI,
		MemoryFuzzTier2Nightly,
		MemoryFuzzTier3ReleaseFocused,
	} {
		if !seen[tier] {
			issues = append(issues, fmt.Sprintf("missing tier policy %s", tier))
		}
	}
	return issues
}

func validateMemoryFuzzRequiredArtifacts(artifacts []MemoryFuzzArtifact) []string {
	seen := map[string]MemoryFuzzArtifact{}
	var issues []string
	for _, artifact := range artifacts {
		if strings.TrimSpace(artifact.Kind) == "" {
			issues = append(issues, "artifact kind is required")
			continue
		}
		seen[artifact.Kind] = artifact
		issues = append(
			issues,
			validateMemoryFuzzTextList(
				"artifact "+artifact.Kind+" path",
				[]string{artifact.Path},
			)...)
	}
	for _, kind := range memoryFuzzRequiredArtifactKinds() {
		artifact, ok := seen[kind]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing required artifact kind %s", kind))
			continue
		}
		if !artifact.Required {
			issues = append(issues, fmt.Sprintf("artifact kind %s must be required", kind))
		}
	}
	return issues
}

func validateMemoryFuzzTextList(label string, values []string) []string {
	if len(values) == 0 {
		return []string{label + " is required"}
	}
	var issues []string
	for _, value := range values {
		text := strings.TrimSpace(value)
		if text == "" {
			issues = append(issues, label+" contains empty text")
			continue
		}
		lower := strings.ToLower(text)
		for _, forbidden := range []string{"todo", "placeholder", " fake", " mock"} {
			if strings.Contains(lower, forbidden) {
				issues = append(
					issues,
					fmt.Sprintf(
						"%s contains forbidden placeholder marker %q",
						label,
						strings.TrimSpace(forbidden),
					),
				)
			}
		}
	}
	return issues
}

func memoryFuzzReportValidationFailureWitness() error {
	report := memoryfacts.Report{
		SchemaVersion: memoryfacts.ReportSchemaV1,
		Rows: []memoryfacts.ReportRow{{
			ProgramID:       "memory-fuzz-oracle",
			FunctionID:      "main",
			SiteID:          "unsafe:oracle",
			SourceFactID:    "memory-fuzz:unsafe-unknown",
			SourceStage:     memoryfacts.StagePLIR,
			Claim:           "unsafe_unknown became safe_known",
			ClaimLevel:      memoryfacts.ClaimConservative,
			ProvenanceClass: memoryfacts.ProvenanceSafeKnown,
			UnsafeClass:     memoryfacts.UnsafeUnknown,
			ValidatorStatus: memoryfacts.ValidatorNotApplicable,
			CostClass:       memoryfacts.CostConservativeFallback,
			Reason:          "fixture must be rejected so report validation failure remains a bug oracle",
		}},
	}
	err := memoryfacts.ValidateReport(report)
	if err == nil {
		return fmt.Errorf(
			("memory fuzz oracle witness expected MemoryFactGraph " +
				"validation failure for unsafe_unknown -> safe_known"),
		)
	}
	return nil
}

func memoryFuzzOracleCategories() []MemoryFuzzOracleCategory {
	return []MemoryFuzzOracleCategory{
		MemoryFuzzOracleCheckerRejectExpected,
		MemoryFuzzOracleRuntimeTrapExpected,
		MemoryFuzzOracleReferenceOutputExpected,
		MemoryFuzzOracleCompilerCrashBug,
		MemoryFuzzOracleMiscompileBug,
		MemoryFuzzOracleUnsafeUnknownOptimizedAsSafeBug,
		MemoryFuzzOracleReportValidationFailureBug,
	}
}

func memoryFuzzRequirementIDs() []MemoryFuzzRequirementID {
	return []MemoryFuzzRequirementID{
		MemoryFuzzRequirementTier1V0V11Coverage,
		MemoryFuzzRequirementCrashMiscompileArtifacts,
		MemoryFuzzRequirementBlockingMemoryFailures,
		MemoryFuzzRequirementTier2NightlySeedTriage,
		MemoryFuzzRequirementTier3ReleasePassOrClassify,
	}
}

func memoryFuzzSliceIDs() []string {
	return []string{"v0", "v1", "v2", "v3", "v4", "v5", "v6", "v7", "v8", "v9", "v10", "v11"}
}

func memoryFuzzInvariantIDs() []MemoryFuzzInvariantID {
	return []MemoryFuzzInvariantID{
		MemoryFuzzInvariantNoSafeMetadataMutation,
		MemoryFuzzInvariantNoBorrowedEscape,
		MemoryFuzzInvariantNoUnsafeUnknownToSafeKnown,
		MemoryFuzzInvariantNoBoundsRemovalWithoutProofID,
		MemoryFuzzInvariantNoStackRegionStorageWhenEscaped,
		MemoryFuzzInvariantReportsValidateAgainstFactGraph,
		MemoryFuzzInvariantReportsPreserveMemoryCostModel,
	}
}

func memoryFuzzBlockingCaseIDs() []MemoryFuzzBlockingCaseID {
	return []MemoryFuzzBlockingCaseID{
		MemoryFuzzBlockingUnsafeUnknownOptimizedAsSafe,
		MemoryFuzzBlockingBoundsCheckWithoutProofID,
		MemoryFuzzBlockingTrustedStorageUnderEscape,
		MemoryFuzzBlockingReportValidationFailure,
	}
}

func memoryFuzzGeneratorSurfaceTiers() []MemoryFuzzGeneratorSurfaceTier {
	return []MemoryFuzzGeneratorSurfaceTier{
		MemoryFuzzGeneratorTier1SupportedNow,
		MemoryFuzzGeneratorTier2SupportedNarrow,
		MemoryFuzzGeneratorTier3ConservativeRejected,
		MemoryFuzzGeneratorTier4Future,
	}
}

func memoryFuzzRequiredArtifactKinds() []string {
	return []string{
		"tier1_short_ci_smoke_report",
		"tier1_short_ci_smoke_summary",
		"tier1_short_ci_smoke_summary_json",
		"compiler_crash_reproducer",
		"miscompile_reproducer",
		"miscompile_reducer",
		"audit_contract",
	}
}

func knownMemoryFuzzRequirementID(id MemoryFuzzRequirementID) bool {
	for _, known := range memoryFuzzRequirementIDs() {
		if id == known {
			return true
		}
	}
	return false
}

func knownMemoryFuzzSliceID(id string) bool {
	for _, known := range memoryFuzzSliceIDs() {
		if id == known {
			return true
		}
	}
	return false
}

func knownMemoryFuzzOracleCategory(category MemoryFuzzOracleCategory) bool {
	for _, known := range memoryFuzzOracleCategories() {
		if category == known {
			return true
		}
	}
	return false
}

func knownMemoryFuzzTier(tier MemoryFuzzTier) bool {
	switch tier {
	case MemoryFuzzTier1ShortCI, MemoryFuzzTier2Nightly, MemoryFuzzTier3ReleaseFocused:
		return true
	default:
		return false
	}
}

func knownMemoryFuzzInvariantID(id MemoryFuzzInvariantID) bool {
	for _, known := range memoryFuzzInvariantIDs() {
		if id == known {
			return true
		}
	}
	return false
}

func knownMemoryFuzzBlockingCaseID(id MemoryFuzzBlockingCaseID) bool {
	for _, known := range memoryFuzzBlockingCaseIDs() {
		if id == known {
			return true
		}
	}
	return false
}

func knownMemoryFuzzGeneratorSurfaceTier(tier MemoryFuzzGeneratorSurfaceTier) bool {
	for _, known := range memoryFuzzGeneratorSurfaceTiers() {
		if tier == known {
			return true
		}
	}
	return false
}

func knownMemoryFuzzStatus(status string) bool {
	return memoryvocab.KnownMemoryFuzzStatus(status)
}

func expectedMemoryFuzzOracleResult(category MemoryFuzzOracleCategory) MemoryFuzzOracleResult {
	switch category {
	case MemoryFuzzOracleCheckerRejectExpected,
		MemoryFuzzOracleRuntimeTrapExpected,
		MemoryFuzzOracleReferenceOutputExpected:
		return MemoryFuzzOraclePass
	case MemoryFuzzOracleCompilerCrashBug,
		MemoryFuzzOracleMiscompileBug,
		MemoryFuzzOracleUnsafeUnknownOptimizedAsSafeBug,
		MemoryFuzzOracleReportValidationFailureBug:
		return MemoryFuzzOracleBug
	default:
		return ""
	}
}

func (r *MemoryFuzzOracleReport) RowsByCategory(
	category MemoryFuzzOracleCategory,
) *MemoryFuzzOracleRow {
	for i := range r.Rows {
		if r.Rows[i].Category == category {
			return &r.Rows[i]
		}
	}
	return &MemoryFuzzOracleRow{}
}

func (r *MemoryFuzzOracleReport) BlockingCase(
	id MemoryFuzzBlockingCaseID,
) *MemoryFuzzBlockingCaseRow {
	for i := range r.BlockingCases {
		if r.BlockingCases[i].ID == id {
			return &r.BlockingCases[i]
		}
	}
	return &MemoryFuzzBlockingCaseRow{}
}

func (r *MemoryFuzzOracleReport) TierPolicy(tier MemoryFuzzTier) *MemoryFuzzTierPolicyRow {
	for i := range r.TierPolicies {
		if r.TierPolicies[i].Tier == tier {
			return &r.TierPolicies[i]
		}
	}
	return &MemoryFuzzTierPolicyRow{}
}

func cloneMemoryFuzzOracleReport(in MemoryFuzzOracleReport) MemoryFuzzOracleReport {
	out := in
	out.Requirements = append([]MemoryFuzzRequirementRow(nil), in.Requirements...)
	out.SliceCoverage = append([]MemoryFuzzSliceCoverageRow(nil), in.SliceCoverage...)
	out.Rows = append([]MemoryFuzzOracleRow(nil), in.Rows...)
	out.Invariants = append([]MemoryFuzzInvariantRow(nil), in.Invariants...)
	out.GeneratorSurfaces = append([]MemoryFuzzGeneratorSurfaceRow(nil), in.GeneratorSurfaces...)
	out.BlockingCases = append([]MemoryFuzzBlockingCaseRow(nil), in.BlockingCases...)
	out.TierPolicies = append([]MemoryFuzzTierPolicyRow(nil), in.TierPolicies...)
	out.Artifacts = append([]MemoryFuzzArtifact(nil), in.Artifacts...)
	out.NonClaims = append([]string(nil), in.NonClaims...)
	return out
}

func memoryFuzzHasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// ---- protocol_trait_object_decision.go ----

const (
	protocolTraitObjectDecisionSchemaV1       = "tetra.language.protocol_trait_object_decision.v1"
	protocolTraitObjectDecisionScopeP222      = "p22.2_protocol_trait_object_decision"
	protocolTraitObjectDecisionKeepStaticOnly = "keep_static_conformance_only"

	protocolTraitStaticConformanceWitnessID    = "static_conformance_direct_call"
	protocolTraitProtocolBoundGenericWitnessID = "protocol_bound_generic_monomorphized_call"
	protocolTraitRuntimeBoundaryWitnessID      = ("runtime_protocol_value_and_requirement_" +
		"call_rejections")
	protocolTraitSpecializationWitnessID = "p17_p21_static_specialization_boundaries"
)

type ProtocolTraitObjectDecisionID string
type protocolTraitDecisionID = ProtocolTraitObjectDecisionID

const (
	ProtocolTraitStaticConformanceFastPath = protocolTraitDecisionID(
		"static_conformance_fast_path",
	)
	ProtocolTraitStaticProtocolBoundGenerics = protocolTraitDecisionID(
		"static_protocol_bound_generics",
	)
	ProtocolTraitRuntimeExistentialDecision = protocolTraitDecisionID(
		"runtime_existential_decision",
	)
	ProtocolTraitExplicitDynamicDispatchGate = protocolTraitDecisionID(
		"explicit_dynamic_dispatch_gate",
	)
	ProtocolTraitSpecializationStaticAbstraction = protocolTraitDecisionID(
		"specialization_static_abstraction",
	)
	ProtocolTraitWitnessTableBoundary  protocolTraitDecisionID = "witness_table_boundary"
	ProtocolTraitTraitObjectBoundary   protocolTraitDecisionID = "trait_object_boundary"
	ProtocolTraitRegistryDocsAlignment protocolTraitDecisionID = "registry_docs_alignment"
)

type ProtocolTraitObjectDecisionReport struct {
	SchemaVersion string                           `json:"schema_version"`
	Scope         string                           `json:"scope"`
	Decision      string                           `json:"decision"`
	Rows          []ProtocolTraitObjectDecisionRow `json:"rows"`
	Witnesses     []ProtocolTraitObjectWitness     `json:"witnesses"`
	NonClaims     []string                         `json:"non_claims"`

	RuntimeExistentialsPromoted bool `json:"runtime_existentials_promoted"`

	TraitObjectsPromoted    bool `json:"trait_objects_promoted"`
	WitnessTablesPromoted   bool `json:"witness_tables_promoted"`
	DynamicDispatchPromoted bool `json:"dynamic_dispatch_promoted"`

	ConformanceTableLookupPromoted bool `json:"conformance_table_lookup_promoted"`

	RuntimeProtocolValuesPromoted bool `json:"runtime_protocol_values_promoted"`

	BroadSpecializationClaimed bool `json:"broad_specialization_claimed"`

	PerformanceClaimed     bool `json:"performance_claimed"`
	RuntimeBehaviorChanged bool `json:"runtime_behavior_changed"`
	SafeSemanticsChanged   bool `json:"safe_semantics_changed"`
}

type ProtocolTraitObjectDecisionRow struct {
	ID         ProtocolTraitObjectDecisionID `json:"id"`
	Name       string                        `json:"name"`
	Status     string                        `json:"status"`
	Decision   string                        `json:"decision"`
	Evidence   []string                      `json:"evidence"`
	Tests      []string                      `json:"tests"`
	Boundaries []string                      `json:"boundaries"`
	WitnessIDs []string                      `json:"witness_ids"`
}

type ProtocolTraitObjectWitness struct {
	ID                               string `json:"id"`
	Kind                             string `json:"kind"`
	ProtocolCount                    int    `json:"protocol_count"`
	ImplCount                        int    `json:"impl_count"`
	HasStaticMethodSig               bool   `json:"has_static_method_sig"`
	DirectCallTarget                 string `json:"direct_call_target"`
	MonomorphizedSig                 string `json:"monomorphized_sig"`
	MonomorphizedSigConcrete         bool   `json:"monomorphized_sig_concrete"`
	LoweredDirectCall                bool   `json:"lowered_direct_call"`
	RuntimeProtocolValueDiagnostic   string `json:"runtime_protocol_value_diagnostic"`
	GenericRequirementCallDiagnostic string `json:"generic_requirement_call_diagnostic"`
	InliningSchema                   string `json:"inlining_schema"`
	MachineSchema                    string `json:"machine_schema"`
	KnownDirectSymbolEvidence        bool   `json:"known_direct_symbol_evidence"`
	SpecializationNoDynamicDispatch  bool   `json:"specialization_no_dynamic_dispatch"`
	MachineNoOpCall                  bool   `json:"machine_no_op_call"`
}

func BuildP22ProtocolTraitObjectDecision() (ProtocolTraitObjectDecisionReport, error) {
	staticWitness, err := buildP22ProtocolStaticConformanceWitness()
	if err != nil {
		return ProtocolTraitObjectDecisionReport{}, err
	}
	genericWitness, err := buildP22ProtocolBoundGenericWitness()
	if err != nil {
		return ProtocolTraitObjectDecisionReport{}, err
	}
	runtimeBoundaryWitness, err := buildP22ProtocolRuntimeBoundaryWitness()
	if err != nil {
		return ProtocolTraitObjectDecisionReport{}, err
	}
	specializationWitness, err := buildP22ProtocolSpecializationWitness()
	if err != nil {
		return ProtocolTraitObjectDecisionReport{}, err
	}

	report := ProtocolTraitObjectDecisionReport{
		SchemaVersion: protocolTraitObjectDecisionSchemaV1,
		Scope:         protocolTraitObjectDecisionScopeP222,
		Decision:      protocolTraitObjectDecisionKeepStaticOnly,
		Witnesses: []ProtocolTraitObjectWitness{
			staticWitness,
			genericWitness,
			runtimeBoundaryWitness,
			specializationWitness,
		},
		Rows: []ProtocolTraitObjectDecisionRow{
			p22ProtocolTraitRow(
				ProtocolTraitStaticConformanceFastPath,
				"Static conformance fast path",
				"current_static_only",
				protocolTraitObjectDecisionKeepStaticOnly,
				[]string{
					("compiler/internal/semantics/semantics_checker.go stores " +
						"protocols separately from value types and validates impl " +
						"Type: Protocol clauses through compareProtocolRequirement."),
					("Static witness static_conformance_direct_call records one " +
						"protocol, one impl, a Vec2.draw FuncSig, and a known direct " +
						"IRCall to Vec2.draw after Parse/Check/Lower."),
					("compiler/tests/semantics/semantics_types_protocols_test.go " +
						"covers extension/static method conformance, throws, " +
						"ownership, effects, generic requirement shape, and imported " +
						"extension clauses."),
				},
				[]string{
					"go test ./compiler -run 'P22ProtocolTrait|ValidateP22ProtocolTrait'",
					"go test ./compiler/tests/semantics -run 'ProtocolConformance'",
				},
				[]string{
					"static conformance remains the fast path",
					"known direct IRCall evidence is required for static dispatch claims",
					"no runtime protocol values, trait objects, witness tables, or dynamic dispatch are promoted",
				},
				[]string{protocolTraitStaticConformanceWitnessID},
			),
			p22ProtocolTraitRow(
				ProtocolTraitStaticProtocolBoundGenerics,
				"Static protocol-bound generics",
				"current_static_only",
				protocolTraitObjectDecisionKeepStaticOnly,
				[]string{
					("compiler/internal/semantics/semantics_checker.go " +
						"validateGenericFuncDecl validates protocol bounds during " +
						"monomorphization and rejects non-protocol, unknown, or " +
						"private bounds."),
					("Static generic witness " +
						"protocol_bound_generic_monomorphized_call records concrete " +
						"id__T_Vec2 monomorphization and a direct call to id__T_Vec2 " +
						"after Parse/Check/Lower."),
					("compiler/tests/semantics/semantics_types_protocols_test.go " +
						"covers same-module and cross-module protocol-bound " +
						"conformance and stable rejection diagnostics."),
				},
				[]string{
					"go test ./compiler/tests/semantics -run 'GenericFunctionProtocolBound'",
					"go test ./compiler -run 'P22ProtocolTrait|ValidateP22ProtocolTrait'",
				},
				[]string{
					"protocol-bound generics are validated statically during monomorphization",
					"no runtime generic values are introduced",
					("generic-bound requirement calls remain unsupported until a " +
						"report-visible dispatch model exists"),
				},
				[]string{
					protocolTraitProtocolBoundGenericWitnessID,
					protocolTraitRuntimeBoundaryWitnessID,
				},
			),
			p22ProtocolTraitRow(
				ProtocolTraitRuntimeExistentialDecision,
				"Runtime existential decision",
				"not_promoted",
				protocolTraitObjectDecisionKeepStaticOnly,
				[]string{
					("P22.2 decision is keep_static_conformance_only: runtime " +
						"existential ABI is not designed in this slice."),
					("Runtime boundary witness records protocol runtime value " +
						"rejection with unknown type 'Drawable' because protocols " +
						"are not value types in the current checker."),
					("docs/spec/core/current_supported_surface.md keeps runtime " +
						"protocol values outside the current v0.4.0 support claim."),
				},
				[]string{
					("go test ./compiler/tests/semantics -run " +
						"'Plan250ProtocolConformanceAndDynamicDispatchBoundaries'"),
					"go test ./compiler -run 'P22ProtocolTrait|ValidateP22ProtocolTrait'",
				},
				[]string{
					"runtime protocol values remain unsupported",
					("runtime existential promotion requires future ABI, lifetime," +
						" ownership, diagnostics, docs, and report evidence"),
					"not promoted in P22.2",
				},
				[]string{protocolTraitRuntimeBoundaryWitnessID},
			),
			p22ProtocolTraitRow(
				ProtocolTraitExplicitDynamicDispatchGate,
				"Explicit dynamic dispatch gate",
				"not_promoted",
				protocolTraitObjectDecisionKeepStaticOnly,
				[]string{
					"Dynamic dispatch must be explicit and report-visible before promotion.",
					("Runtime boundary witness records generic-bound requirement " +
						"call rejection rather than lowering through witness-table " +
						"dispatch."),
					("FeatureRegistry language.protocol-bound-generics-static " +
						"says calling protocol requirements through generic bounds, " +
						"witness tables, trait objects, runtime protocol values, and " +
						"dynamic dispatch remain unsupported."),
				},
				[]string{
					("go test ./compiler/tests/semantics -run " +
						"'GenericFunctionProtocolBoundRequirementCallUnsupported'"),
					"go test ./compiler -run 'P22ProtocolTrait|ValidateP22ProtocolTrait'",
				},
				[]string{
					"dynamic dispatch must be explicit and report-visible",
					"dynamic dispatch is not promoted",
					"generic-bound requirement calls remain diagnostics",
				},
				[]string{
					protocolTraitRuntimeBoundaryWitnessID,
					protocolTraitSpecializationWitnessID,
				},
			),
			p22ProtocolTraitRow(
				ProtocolTraitSpecializationStaticAbstraction,
				"Specialization removes static abstraction",
				"bounded_existing_evidence",
				protocolTraitObjectDecisionKeepStaticOnly,
				[]string{
					("P17.2 InliningSpecializationCoverage records static " +
						"protocol/conformance calls only after lowering to a known " +
						"direct Stack IR function symbol."),
					("P21.2 SpecializationMachineCodeCoverage records " +
						"protocol/static conformance rows and Machine IR contains no " +
						"OpCall for the bounded known-direct witness."),
					("Specialization witness " +
						"p17_p21_static_specialization_boundaries records P17.2 and " +
						"P21.2 schemas, known direct symbol evidence, no dynamic " +
						"dispatch claim, and Machine IR contains no OpCall."),
				},
				[]string{
					"go test ./compiler/internal/opt -run 'InliningSpecialization|SpecializationMachineCode'",
					("go test ./compiler/tests/semantics -run " +
						"'InliningSpecialization|ProtocolConformance|GenericFunctionP" +
						"rotocolBound'"),
				},
				[]string{
					"static abstraction removal is limited to known direct Stack IR function symbols",
					"Machine IR contains no OpCall only for the bounded direct-call witness",
					("no broad protocol specialization, witness-table removal, " +
						"dynamic dispatch removal, or performance claim is made"),
				},
				[]string{protocolTraitSpecializationWitnessID},
			),
			p22ProtocolTraitRow(
				ProtocolTraitWitnessTableBoundary,
				"Witness-table boundary",
				"not_promoted",
				protocolTraitObjectDecisionKeepStaticOnly,
				[]string{
					"No current lowering path emits witness tables for protocol values.",
					"Current specialization rows mention witness tables only as non-claims and future boundaries.",
					("Future witness tables require future ABI evidence, " +
						"lifetime/ownership rules, generated metadata, diagnostics, " +
						"docs, and report-visible dynamic dispatch rows."),
				},
				[]string{
					"go test ./compiler/internal/opt -run 'InliningSpecialization|SpecializationMachineCode'",
					"go test ./compiler -run 'P22ProtocolTrait|ValidateP22ProtocolTrait'",
				},
				[]string{
					"witness tables are not emitted",
					"witness-table promotion is forbidden without future ABI evidence",
					"conformance-table lookup is not promoted",
				},
				[]string{protocolTraitSpecializationWitnessID},
			),
			p22ProtocolTraitRow(
				ProtocolTraitTraitObjectBoundary,
				"Trait-object boundary",
				"not_promoted",
				protocolTraitObjectDecisionKeepStaticOnly,
				[]string{
					"Trait objects are not promoted by P22.2.",
					("Runtime existential values are not value types in the " +
						"current checker and remain a future design question."),
					("P22.0 feature surface audit routes protocol/trait-object " +
						"runtime values to P22.2 and P22.2 keeps them out of the " +
						"current branch without same-branch ABI/lifetime evidence."),
				},
				[]string{
					("go test ./compiler/tests/semantics -run " +
						"'Plan250ProtocolConformanceAndDynamicDispatchBoundaries'"),
					"go test ./compiler -run 'P22ProtocolTrait|ValidateP22ProtocolTrait'",
				},
				[]string{
					"trait objects are not promoted",
					"runtime existential ABI is not designed in this slice",
					"trait-object promotion requires future ABI, lifetime, ownership, and report evidence",
				},
				[]string{protocolTraitRuntimeBoundaryWitnessID},
			),
			p22ProtocolTraitRow(
				ProtocolTraitRegistryDocsAlignment,
				"Registry and docs alignment",
				"current_static_only",
				protocolTraitObjectDecisionKeepStaticOnly,
				[]string{
					("FeatureRegistry records language.protocol-conformance-mvp " +
						"as static conformance with no witness tables, trait objects," +
						" or dynamic dispatch model."),
					("FeatureRegistry records " +
						"language.protocol-bound-generics-static as static " +
						"monomorphization-time validation with runtime protocol " +
						"values and dynamic dispatch unsupported."),
					("docs/spec/core/current_supported_surface.md, " +
						"docs/design/explainable_one_build.md, and " +
						"docs/design/truthful_intent_architecture.md preserve the " +
						"same static-only decision boundary."),
				},
				[]string{
					"go test ./compiler/tests/semantics -run 'FeatureRegistry'",
					"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
					"go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json",
				},
				[]string{
					"FeatureRegistry and docs must agree on the static-only decision",
					"language.protocol-conformance-mvp remains current static conformance",
					"language.protocol-bound-generics-static remains current static protocol-bound generics",
				},
				[]string{
					protocolTraitStaticConformanceWitnessID,
					protocolTraitProtocolBoundGenericWitnessID,
					protocolTraitRuntimeBoundaryWitnessID,
					protocolTraitSpecializationWitnessID,
				},
			),
		},
		NonClaims: p22ProtocolTraitNonClaims(),
	}
	return report, nil
}

func ValidateP22ProtocolTraitObjectDecision(report ProtocolTraitObjectDecisionReport) error {
	if report.SchemaVersion != protocolTraitObjectDecisionSchemaV1 {
		return fmt.Errorf(
			"protocol trait-object decision: schema = %q, want %q",
			report.SchemaVersion,
			protocolTraitObjectDecisionSchemaV1,
		)
	}
	if report.Scope != protocolTraitObjectDecisionScopeP222 {
		return fmt.Errorf(
			"protocol trait-object decision: scope = %q, want %q",
			report.Scope,
			protocolTraitObjectDecisionScopeP222,
		)
	}
	if report.Decision != protocolTraitObjectDecisionKeepStaticOnly {
		return fmt.Errorf(
			"protocol trait-object decision: decision = %q, want %q",
			report.Decision,
			protocolTraitObjectDecisionKeepStaticOnly,
		)
	}
	if report.RuntimeExistentialsPromoted {
		return fmt.Errorf(
			"protocol trait-object decision: runtime existential promotion is forbidden",
		)
	}
	if report.TraitObjectsPromoted {
		return fmt.Errorf("protocol trait-object decision: trait object promotion is forbidden")
	}
	if report.WitnessTablesPromoted {
		return fmt.Errorf("protocol trait-object decision: witness table promotion is forbidden")
	}
	if report.DynamicDispatchPromoted {
		return fmt.Errorf("protocol trait-object decision: dynamic dispatch promotion is forbidden")
	}
	if report.ConformanceTableLookupPromoted {
		return fmt.Errorf(
			"protocol trait-object decision: conformance-table lookup promotion is forbidden",
		)
	}
	if report.RuntimeProtocolValuesPromoted {
		return fmt.Errorf(
			"protocol trait-object decision: runtime protocol value promotion is forbidden",
		)
	}
	if report.BroadSpecializationClaimed {
		return fmt.Errorf("protocol trait-object decision: broad specialization claim is forbidden")
	}
	if report.PerformanceClaimed {
		return fmt.Errorf("protocol trait-object decision: performance claim is forbidden")
	}
	if report.RuntimeBehaviorChanged {
		return fmt.Errorf(
			"protocol trait-object decision: runtime behavior change claim is forbidden",
		)
	}
	if report.SafeSemanticsChanged {
		return fmt.Errorf(
			"protocol trait-object decision: safe-program semantics change is forbidden",
		)
	}
	for _, nonClaim := range p22ProtocolTraitNonClaims() {
		if !p22ProtocolTraitReportHasString(report.NonClaims, nonClaim) {
			return fmt.Errorf("protocol trait-object decision: missing non-claim %q", nonClaim)
		}
	}
	if err := validateP22ProtocolTraitStrings("non-claim", report.NonClaims); err != nil {
		return err
	}

	witnesses := map[string]ProtocolTraitObjectWitness{}
	for _, witness := range report.Witnesses {
		if strings.TrimSpace(witness.ID) == "" || strings.TrimSpace(witness.Kind) == "" {
			return fmt.Errorf(
				"protocol trait-object decision: witness missing required metadata: %#v",
				witness,
			)
		}
		if _, ok := witnesses[witness.ID]; ok {
			return fmt.Errorf("protocol trait-object decision: duplicate witness %s", witness.ID)
		}
		witnesses[witness.ID] = witness
	}
	for _, id := range []string{
		protocolTraitStaticConformanceWitnessID,
		protocolTraitProtocolBoundGenericWitnessID,
		protocolTraitRuntimeBoundaryWitnessID,
		protocolTraitSpecializationWitnessID,
	} {
		if _, ok := witnesses[id]; !ok {
			return fmt.Errorf("protocol trait-object decision: missing witness %s", id)
		}
	}
	if err := validateP22ProtocolStaticWitness(
		witnesses[protocolTraitStaticConformanceWitnessID],
	); err != nil {
		return err
	}
	if err := validateP22ProtocolGenericWitness(
		witnesses[protocolTraitProtocolBoundGenericWitnessID],
	); err != nil {
		return err
	}
	if err := validateP22ProtocolRuntimeBoundaryWitness(
		witnesses[protocolTraitRuntimeBoundaryWitnessID],
	); err != nil {
		return err
	}
	if err := validateP22ProtocolSpecializationWitness(
		witnesses[protocolTraitSpecializationWitnessID],
	); err != nil {
		return err
	}

	expected := map[ProtocolTraitObjectDecisionID]bool{}
	for _, id := range p22ProtocolTraitObjectDecisionIDs() {
		expected[id] = true
	}
	seen := map[ProtocolTraitObjectDecisionID]bool{}
	for _, row := range report.Rows {
		if row.ID == "" || strings.TrimSpace(row.Name) == "" ||
			strings.TrimSpace(row.Status) == "" ||
			strings.TrimSpace(row.Decision) == "" {
			return fmt.Errorf(
				"protocol trait-object decision: row missing required metadata: %#v",
				row,
			)
		}
		if !expected[row.ID] {
			return fmt.Errorf("protocol trait-object decision: unexpected row %s", row.ID)
		}
		if seen[row.ID] {
			return fmt.Errorf("protocol trait-object decision: duplicate row %s", row.ID)
		}
		seen[row.ID] = true
		if row.Decision != protocolTraitObjectDecisionKeepStaticOnly {
			return fmt.Errorf(
				"protocol trait-object decision: row %s decision = %q, want %q",
				row.ID,
				row.Decision,
				protocolTraitObjectDecisionKeepStaticOnly,
			)
		}
		if err := validateP22ProtocolTraitStrings(
			"row "+string(row.ID)+" evidence",
			row.Evidence,
		); err != nil {
			return err
		}
		if err := validateP22ProtocolTraitStrings("row "+string(row.ID)+" tests", row.Tests); err != nil {
			return err
		}
		if err := validateP22ProtocolTraitStrings(
			"row "+string(row.ID)+" boundaries",
			row.Boundaries,
		); err != nil {
			return err
		}
		if len(row.WitnessIDs) == 0 {
			return fmt.Errorf(
				"protocol trait-object decision: row %s missing witness reference",
				row.ID,
			)
		}
		for _, id := range row.WitnessIDs {
			if _, ok := witnesses[id]; !ok {
				return fmt.Errorf(
					"protocol trait-object decision: row %s references missing witness %s",
					row.ID,
					id,
				)
			}
		}
	}
	for _, id := range p22ProtocolTraitObjectDecisionIDs() {
		if !seen[id] {
			return fmt.Errorf("protocol trait-object decision: missing row %s", id)
		}
	}
	return nil
}

func p22ProtocolTraitObjectDecisionIDs() []ProtocolTraitObjectDecisionID {
	return []ProtocolTraitObjectDecisionID{
		ProtocolTraitStaticConformanceFastPath,
		ProtocolTraitStaticProtocolBoundGenerics,
		ProtocolTraitRuntimeExistentialDecision,
		ProtocolTraitExplicitDynamicDispatchGate,
		ProtocolTraitSpecializationStaticAbstraction,
		ProtocolTraitWitnessTableBoundary,
		ProtocolTraitTraitObjectBoundary,
		ProtocolTraitRegistryDocsAlignment,
	}
}

func p22ProtocolTraitNonClaims() []string {
	return []string{
		"runtime protocol values are not promoted",
		"trait objects are not promoted",
		"witness tables are not promoted",
		"dynamic dispatch is not promoted",
		"conformance-table lookup is not promoted",
		"runtime existential ABI is not designed in this slice",
		"broad protocol specialization is not claimed",
		"performance is not claimed",
		"runtime behavior does not change",
		"safe-program semantics do not change",
	}
}

func p22ProtocolTraitRow(
	id ProtocolTraitObjectDecisionID,
	name, status, decision string,
	evidence, tests, boundaries, witnessIDs []string,
) ProtocolTraitObjectDecisionRow {
	return ProtocolTraitObjectDecisionRow{
		ID:         id,
		Name:       name,
		Status:     status,
		Decision:   decision,
		Evidence:   append([]string{}, evidence...),
		Tests:      append([]string{}, tests...),
		Boundaries: append([]string{}, boundaries...),
		WitnessIDs: append([]string{}, witnessIDs...),
	}
}

func buildP22ProtocolStaticConformanceWitness() (ProtocolTraitObjectWitness, error) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Renderable:
    func draw(self: Vec2) -> Int

extension Vec2:
    func draw(self: Vec2) -> Int:
        return self.x

impl Vec2: Renderable

func main() -> Int:
    return Vec2.draw(Vec2(x: 42))
`)
	prog, err := Parse(src)
	if err != nil {
		return ProtocolTraitObjectWitness{}, fmt.Errorf(
			"protocol trait-object decision: parse static witness: %w",
			err,
		)
	}
	checked, err := Check(prog)
	if err != nil {
		return ProtocolTraitObjectWitness{}, fmt.Errorf(
			"protocol trait-object decision: check static witness: %w",
			err,
		)
	}
	lowered, err := Lower(checked)
	if err != nil {
		return ProtocolTraitObjectWitness{}, fmt.Errorf(
			"protocol trait-object decision: lower static witness: %w",
			err,
		)
	}
	main, ok := p222FindIRFunc(lowered, "main")
	if !ok {
		return ProtocolTraitObjectWitness{}, fmt.Errorf(
			"protocol trait-object decision: static witness missing lowered main",
		)
	}
	directCallTarget := ""
	if p222HasIRCall(main, "Vec2.draw") {
		directCallTarget = "Vec2.draw"
	}
	_, hasSig := checked.FuncSigs["Vec2.draw"]
	return ProtocolTraitObjectWitness{
		ID:                 protocolTraitStaticConformanceWitnessID,
		Kind:               "static_conformance_fast_path",
		ProtocolCount:      len(prog.Protocols),
		ImplCount:          len(prog.Impls),
		HasStaticMethodSig: hasSig,
		DirectCallTarget:   directCallTarget,
		LoweredDirectCall:  directCallTarget != "",
	}, nil
}

func buildP22ProtocolBoundGenericWitness() (ProtocolTraitObjectWitness, error) {
	src := []byte(`
struct Vec2:
    x: Int

protocol Echoable:
    func echo(self: Vec2) -> Vec2

extension Vec2:
    func echo(self: Vec2) -> Vec2:
        return self

impl Vec2: Echoable

func id<T: Echoable>(x: T) -> T:
    return x

func main() -> Int:
    let v: Vec2 = Vec2(x: 42)
    let out: Vec2 = id(v)
    return out.x
`)
	prog, err := Parse(src)
	if err != nil {
		return ProtocolTraitObjectWitness{}, fmt.Errorf(
			"protocol trait-object decision: parse generic witness: %w",
			err,
		)
	}
	checked, err := Check(prog)
	if err != nil {
		return ProtocolTraitObjectWitness{}, fmt.Errorf(
			"protocol trait-object decision: check generic witness: %w",
			err,
		)
	}
	lowered, err := Lower(checked)
	if err != nil {
		return ProtocolTraitObjectWitness{}, fmt.Errorf(
			"protocol trait-object decision: lower generic witness: %w",
			err,
		)
	}
	sig, ok := checked.FuncSigs["id__T_Vec2"]
	main, hasMain := p222FindIRFunc(lowered, "main")
	loweredDirectCall := hasMain && p222HasIRCall(main, "id__T_Vec2")
	return ProtocolTraitObjectWitness{
		ID:                       protocolTraitProtocolBoundGenericWitnessID,
		Kind:                     "static_protocol_bound_generics",
		ProtocolCount:            len(prog.Protocols),
		ImplCount:                len(prog.Impls),
		MonomorphizedSig:         "id__T_Vec2",
		MonomorphizedSigConcrete: ok && !sig.Generic,
		LoweredDirectCall:        loweredDirectCall,
		DirectCallTarget:         "id__T_Vec2",
	}, nil
}

func buildP22ProtocolRuntimeBoundaryWitness() (ProtocolTraitObjectWitness, error) {
	runtimeValueErr := p222CheckDiagnostic(`
struct Vec2:
    x: Int

protocol Drawable:
    func draw(self: Vec2) -> Int

func main() -> Int:
    let value: Drawable = Vec2(x: 1)
    return 0
`)
	if runtimeValueErr == "" {
		return ProtocolTraitObjectWitness{}, fmt.Errorf(
			"protocol trait-object decision: runtime protocol value witness unexpectedly checked",
		)
	}
	requirementCallErr := p222CheckDiagnostic(`
struct Vec2:
    x: Int

protocol Echoable:
    func echo(self: Vec2) -> Vec2

extension Vec2:
    func echo(self: Vec2) -> Vec2:
        return self

impl Vec2: Echoable

func echoThroughBound<T: Echoable>(x: T) -> T:
    return T.echo(x)

func main() -> Int:
    let v: Vec2 = Vec2(x: 42)
    let out: Vec2 = echoThroughBound(v)
    return out.x
`)
	if requirementCallErr == "" {
		return ProtocolTraitObjectWitness{}, fmt.Errorf(
			"protocol trait-object decision: generic requirement call witness unexpectedly checked",
		)
	}
	return ProtocolTraitObjectWitness{
		ID:                               protocolTraitRuntimeBoundaryWitnessID,
		Kind:                             "runtime_existential_and_dynamic_dispatch_boundary",
		RuntimeProtocolValueDiagnostic:   runtimeValueErr,
		GenericRequirementCallDiagnostic: requirementCallErr,
	}, nil
}

func buildP22ProtocolSpecializationWitness() (ProtocolTraitObjectWitness, error) {
	inlining := opt.InliningSpecializationCoverage()
	staticInline, ok := p222FindInliningRow(
		inlining,
		opt.InliningSpecializationStaticProtocolConformanceCalls,
	)
	if !ok {
		return ProtocolTraitObjectWitness{}, fmt.Errorf(
			"protocol trait-object decision: missing P17.2 static protocol/conformance row",
		)
	}
	machine, err := opt.SpecializationMachineCodeCoverage()
	if err != nil {
		return ProtocolTraitObjectWitness{}, fmt.Errorf(
			"protocol trait-object decision: P21.2 specialization witness: %w",
			err,
		)
	}
	if err := opt.ValidateSpecializationMachineCodeCoverage(machine); err != nil {
		return ProtocolTraitObjectWitness{}, fmt.Errorf(
			"protocol trait-object decision: validate P21.2 specialization witness: %w",
			err,
		)
	}
	staticMachine, ok := p222FindMachineRow(
		machine,
		opt.SpecializationMachineCodeProtocolStaticConformance,
	)
	if !ok {
		return ProtocolTraitObjectWitness{}, fmt.Errorf(
			"protocol trait-object decision: missing P21.2 protocol/static conformance row",
		)
	}
	machineNoOpCall := false
	if len(machine.Witnesses) > 0 {
		machineNoOpCall = !machine.Witnesses[0].MachineIRHasCall &&
			machine.Witnesses[0].MachineIRVerified
	}
	combined := strings.Join([]string{
		staticInline.Boundary,
		staticInline.Evidence,
		staticMachine.SourceEvidence,
		staticMachine.OptimizedIREvidence,
		staticMachine.MachineCodeEvidence,
		staticMachine.Boundary,
	}, " ")
	return ProtocolTraitObjectWitness{
		ID:             protocolTraitSpecializationWitnessID,
		Kind:           "specialization_static_abstraction",
		InliningSchema: inlining.SchemaVersion,
		MachineSchema:  machine.SchemaVersion,
		KnownDirectSymbolEvidence: strings.Contains(
			combined,
			"known direct Stack IR function symbol",
		),
		SpecializationNoDynamicDispatch: strings.Contains(combined, "no witness tables") &&
			strings.Contains(combined, "dynamic dispatch"),
		MachineNoOpCall: machineNoOpCall &&
			strings.Contains(combined, "Machine IR contains no OpCall"),
	}, nil
}

func p222CheckDiagnostic(src string) string {
	prog, err := Parse([]byte(src))
	if err != nil {
		return err.Error()
	}
	_, err = Check(prog)
	if err == nil {
		return ""
	}
	return err.Error()
}

func p222FindIRFunc(prog *IRProgram, name string) (ir.IRFunc, bool) {
	for _, fn := range prog.Funcs {
		if fn.Name == name {
			return fn, true
		}
	}
	return ir.IRFunc{}, false
}

func p222HasIRCall(fn ir.IRFunc, name string) bool {
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRCall && instr.Name == name {
			return true
		}
	}
	return false
}

func p222FindInliningRow(
	report opt.InliningSpecializationCoverageReport,
	id opt.InliningSpecializationID,
) (opt.InliningSpecializationCoverageRow, bool) {
	for _, row := range report.Rows {
		if row.ID == id {
			return row, true
		}
	}
	return opt.InliningSpecializationCoverageRow{}, false
}

func p222FindMachineRow(
	report opt.SpecializationMachineCodeCoverageReport,
	id opt.SpecializationMachineCodeID,
) (opt.SpecializationMachineCodeRow, bool) {
	for _, row := range report.Rows {
		if row.ID == id {
			return row, true
		}
	}
	return opt.SpecializationMachineCodeRow{}, false
}

func validateP22ProtocolStaticWitness(witness ProtocolTraitObjectWitness) error {
	if witness.ProtocolCount != 1 || witness.ImplCount != 1 || !witness.HasStaticMethodSig ||
		witness.DirectCallTarget != "Vec2.draw" ||
		!witness.LoweredDirectCall {
		return fmt.Errorf(
			"protocol trait-object decision: static conformance witness drift: %#v",
			witness,
		)
	}
	return nil
}

func validateP22ProtocolGenericWitness(witness ProtocolTraitObjectWitness) error {
	if witness.MonomorphizedSig != "id__T_Vec2" || !witness.MonomorphizedSigConcrete ||
		witness.DirectCallTarget != "id__T_Vec2" ||
		!witness.LoweredDirectCall {
		return fmt.Errorf(
			"protocol trait-object decision: protocol-bound generic witness drift: %#v",
			witness,
		)
	}
	return nil
}

func validateP22ProtocolRuntimeBoundaryWitness(witness ProtocolTraitObjectWitness) error {
	if !strings.Contains(witness.RuntimeProtocolValueDiagnostic, "unknown type 'Drawable'") ||
		!strings.Contains(witness.GenericRequirementCallDiagnostic, "not supported in this MVP") {
		return fmt.Errorf(
			"protocol trait-object decision: runtime boundary witness drift: %#v",
			witness,
		)
	}
	return nil
}

func validateP22ProtocolSpecializationWitness(witness ProtocolTraitObjectWitness) error {
	if witness.InliningSchema != "tetra.optimizer.inlining_specialization.v1" ||
		witness.MachineSchema != "tetra.optimizer.specialization_machine_code.v1" ||
		!witness.KnownDirectSymbolEvidence ||
		!witness.SpecializationNoDynamicDispatch ||
		!witness.MachineNoOpCall {
		return fmt.Errorf(
			"protocol trait-object decision: specialization witness drift: %#v",
			witness,
		)
	}
	return nil
}

func validateP22ProtocolTraitStrings(label string, items []string) error {
	if len(items) == 0 {
		return fmt.Errorf("protocol trait-object decision: %s missing", label)
	}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			return fmt.Errorf("protocol trait-object decision: %s contains empty item", label)
		}
		if p22ProtocolTraitContainsPlaceholder(trimmed) {
			return fmt.Errorf(
				"protocol trait-object decision: %s contains placeholder evidence: %q",
				label,
				item,
			)
		}
	}
	return nil
}

func p22ProtocolTraitContainsPlaceholder(text string) bool {
	lower := strings.ToLower(text)
	for _, token := range []string{"todo", "tbd", "placeholder", "fixme", "???"} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func p22ProtocolTraitReportHasString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

// ---- runtime_hardening_v1.go ----

const (
	runtimeHardeningV1Schema    = "tetra.runtime.hardening.v1"
	runtimeHardeningV1ScopeP241 = "p24.1_runtime_hardening"

	p24RuntimeHardeningTrapWitnessID       = "deterministic_trap_surface"
	p24RuntimeHardeningAllocationWitnessID = "allocation_failure_surface"
	p24RuntimeHardeningStackWitnessID      = "stack_overflow_boundary"
	p24RuntimeHardeningOverflowWitnessID   = "integer_overflow_semantics"
	p24RuntimeHardeningCorruptionWitnessID = "allocator_corruption_instrumentation"
	p24RuntimeHardeningRegionWitnessID     = "region_lifetime_instrumentation"
	p24RuntimeHardeningMailboxWitnessID    = "actor_mailbox_overflow_policy"
	p24RuntimeHardeningParserWitnessID     = "network_parser_limits"
	p24RuntimeHardeningArtifactsWitnessID  = "runtime_hardening_artifacts"
)

type RuntimeHardeningV1ID string
type runtimeHardeningID = RuntimeHardeningV1ID

const (
	RuntimeHardeningDeterministicTraps       runtimeHardeningID = "deterministic_traps"
	RuntimeHardeningOOMPolicy                runtimeHardeningID = "oom_policy"
	RuntimeHardeningStackOverflowGuard       runtimeHardeningID = "stack_overflow_guard"
	RuntimeHardeningIntegerOverflowSemantics                    = runtimeHardeningID(
		"integer_overflow_semantics_audit",
	)
	RuntimeHardeningAllocatorCorruptionInstrumentation = runtimeHardeningID(
		"allocator_corruption_detection",
	)
	RuntimeHardeningRegionUseAfterFreeInstrumentation = runtimeHardeningID(
		"region_double_free_use_after_free",
	)
	RuntimeHardeningActorMailboxOverflowPolicy = runtimeHardeningID(
		"actor_mailbox_overflow_policy",
	)
	RuntimeHardeningNetworkParserLimits runtimeHardeningID = "network_parser_limits"
)

type RuntimeHardeningV1Report struct {
	SchemaVersion string                      `json:"schema_version"`
	Scope         string                      `json:"scope"`
	Rows          []RuntimeHardeningV1Row     `json:"rows"`
	Witnesses     []RuntimeHardeningV1Witness `json:"witnesses"`
	Artifacts     []RuntimeHardeningArtifact  `json:"artifacts"`
	NonClaims     []string                    `json:"non_claims"`

	DeterministicTrapsReviewed      bool `json:"deterministic_traps_reviewed"`
	OOMPolicyReviewed               bool `json:"oom_policy_reviewed"`
	StackOverflowGuardReviewed      bool `json:"stack_overflow_guard_reviewed"`
	IntegerOverflowSemanticsAudited bool `json:"integer_overflow_semantics_audited"`

	AllocatorCorruptionReviewed bool `json:"allocator_corruption_instrumentation_reviewed"`

	RegionLifetimeReviewed             bool `json:"region_double_free_use_after_free_reviewed"`
	ActorMailboxOverflowPolicyReviewed bool `json:"actor_mailbox_overflow_policy_reviewed"`
	NetworkParserLimitsReviewed        bool `json:"network_parser_limits_reviewed"`
	RuntimeHardeningArtifactPresent    bool `json:"runtime_hardening_artifact_present"`
	RuntimeHardeningDesignPresent      bool `json:"runtime_hardening_design_artifact_present"`
	FullRuntimeHardeningClaimed        bool `json:"full_runtime_hardening_claimed"`
	FullStackOverflowProtectionClaimed bool `json:"full_stack_overflow_protection_claimed"`
	FullOOMRecoveryClaimed             bool `json:"full_oom_recovery_claimed"`

	FullAllocatorCorruptionDetectionClaimed bool `json:"full_allocator_corruption_detection_claimed"`

	ProductionActorMailboxClaimed bool `json:"production_actor_mailbox_claimed"`
	RuntimeBehaviorChanged        bool `json:"runtime_behavior_changed"`
	SafeSemanticsChanged          bool `json:"safe_semantics_changed"`
	PerformanceClaimed            bool `json:"performance_claimed"`
}

type RuntimeHardeningV1Row struct {
	ID         RuntimeHardeningV1ID `json:"id"`
	Name       string               `json:"name"`
	Status     string               `json:"status"`
	Evidence   []string             `json:"evidence"`
	Tests      []string             `json:"tests"`
	Boundaries []string             `json:"boundaries"`
	WitnessIDs []string             `json:"witness_ids"`
}

type RuntimeHardeningArtifact struct {
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Present bool   `json:"present"`
}

type RuntimeHardeningV1Witness struct {
	ID                  string   `json:"id"`
	Kind                string   `json:"kind"`
	Paths               []string `json:"paths,omitempty"`
	TrapPolicyReviewed  bool     `json:"trap_policy_reviewed,omitempty"`
	WasmTrapEmitters    int      `json:"wasm_trap_emitters,omitempty"`
	PanicImportPresent  bool     `json:"panic_import_present,omitempty"`
	AllocationContracts int      `json:"allocation_contracts,omitempty"`

	ContractsWithFailureBehavior int `json:"contracts_with_failure_behavior,omitempty"`

	ContractsWithOverflowGuards int `json:"contracts_with_overflow_guards,omitempty"`

	OOMPolicyReviewed bool `json:"oom_policy_reviewed,omitempty"`
	StackDepthChecks  int  `json:"stack_depth_checks,omitempty"`

	StackOverflowGuardReviewed bool `json:"stack_overflow_guard_reviewed,omitempty"`

	FullStackOverflowProtectionClaimed bool `json:"full_stack_overflow_protection_claimed,omitempty"`

	CheckedNegI32Present bool `json:"checked_neg_i32_present,omitempty"`

	FoldConstBinaryI32Present bool `json:"fold_const_binary_i32_present,omitempty"`

	ConstOverflowDiagnosticPresent bool `json:"const_overflow_diagnostic_present,omitempty"`

	AllocationOverflowGuards int `json:"allocation_overflow_guards,omitempty"`

	IntegerOverflowSemanticsAudited bool `json:"integer_overflow_semantics_audited,omitempty"`

	BoundsHeaderContracts int `json:"bounds_header_contracts,omitempty"`

	SmallHeapDoubleFreeRejected bool `json:"small_heap_double_free_rejected,omitempty"`

	RawPointerBoundsMetadataVersion string `json:"raw_pointer_bounds_metadata_version,omitempty"`

	AllocatorCorruptionReviewed bool `json:"allocator_corruption_instrumentation_reviewed,omitempty"`

	RegionDebugHeaderBytes int32 `json:"region_debug_header_bytes,omitempty"`

	RegionUseAfterFreeContracts int `json:"region_use_after_free_contracts,omitempty"`

	RegionDoubleFreeContracts int `json:"region_double_free_contracts,omitempty"`
	RegionResetContracts      int `json:"region_reset_contracts,omitempty"`

	RegionLifetimeReviewed bool `json:"region_double_free_use_after_free_reviewed,omitempty"`

	MailboxCapacity         int    `json:"mailbox_capacity,omitempty"`
	BackpressureMode        string `json:"backpressure_mode,omitempty"`
	MailboxOverflowRejected bool   `json:"mailbox_overflow_rejected,omitempty"`
	MailboxFIFOReceive      bool   `json:"mailbox_fifo_receive,omitempty"`
	ActorBoundaryRows       int    `json:"actor_boundary_rows,omitempty"`

	BuiltinMessagePoolOverflowChecked bool `json:"builtin_message_pool_overflow_checked,omitempty"`

	ActorMailboxOverflowPolicyReviewed bool `json:"actor_mailbox_overflow_policy_reviewed,omitempty"`

	HTTPParserLimitsReviewed bool `json:"http_parser_limits_reviewed,omitempty"`

	HTTPRequestViewLimitsReviewed bool `json:"http_request_view_limits_reviewed,omitempty"`

	PostgresFrameLimitsReviewed bool `json:"postgres_frame_limits_reviewed,omitempty"`

	NetworkParserLimitsReviewed bool `json:"network_parser_limits_reviewed,omitempty"`

	RuntimeHardeningArtifactPresent bool `json:"runtime_hardening_artifact_present,omitempty"`

	RuntimeHardeningDesignPresent bool `json:"runtime_hardening_design_artifact_present,omitempty"`
}

func BuildP24RuntimeHardeningV1Report() (RuntimeHardeningV1Report, error) {
	trapWitness := buildP24RuntimeHardeningTrapWitness()
	allocationWitness, err := buildP24RuntimeHardeningAllocationWitness()
	if err != nil {
		return RuntimeHardeningV1Report{}, err
	}
	stackWitness := buildP24RuntimeHardeningStackWitness()
	overflowWitness, err := buildP24RuntimeHardeningOverflowWitness()
	if err != nil {
		return RuntimeHardeningV1Report{}, err
	}
	corruptionWitness, err := buildP24RuntimeHardeningCorruptionWitness()
	if err != nil {
		return RuntimeHardeningV1Report{}, err
	}
	regionWitness, err := buildP24RuntimeHardeningRegionWitness()
	if err != nil {
		return RuntimeHardeningV1Report{}, err
	}
	mailboxWitness, err := buildP24RuntimeHardeningMailboxWitness()
	if err != nil {
		return RuntimeHardeningV1Report{}, err
	}
	parserWitness := buildP24RuntimeHardeningParserWitness()
	artifacts := p24RuntimeHardeningArtifacts()
	artifactWitness := buildP24RuntimeHardeningArtifactsWitness(artifacts)

	report := RuntimeHardeningV1Report{
		SchemaVersion: runtimeHardeningV1Schema,
		Scope:         runtimeHardeningV1ScopeP241,
		Witnesses: []RuntimeHardeningV1Witness{
			trapWitness,
			allocationWitness,
			stackWitness,
			overflowWitness,
			corruptionWitness,
			regionWitness,
			mailboxWitness,
			parserWitness,
			artifactWitness,
		},
		Artifacts: artifacts,
		Rows: []RuntimeHardeningV1Row{
			p24RuntimeHardeningRow(
				RuntimeHardeningDeterministicTraps,
				"Deterministic traps",
				"reviewed_current_surface",
				[]string{
					("runtimeabi allocation contracts use trap_or_stable_status " +
						"for allocation failure behavior and reject invalid sizes " +
						"before allocator access."),
					("wasm32-wasi and wasm32-web backends contain emitWasmTrapIf " +
						"deterministic trap emitters, and the web panic import " +
						"formats tetra panic diagnostics deterministically."),
				},
				[]string{
					"go test ./compiler -run 'P24RuntimeHardening' -count=1",
					"go test ./compiler/internal/runtimeabi/... -run 'Allocation' -count=1",
					"go test ./compiler/tests/lowering -run 'Wasm|ABI' -count=1",
				},
				[]string{
					("trap review is bounded to current backend/runtime ABI " +
						"surfaces and does not claim a full trap taxonomy for every " +
						"target"),
					"stable trap/status behavior is policy evidence, not a runtime behavior change",
				},
				[]string{p24RuntimeHardeningTrapWitnessID, p24RuntimeHardeningAllocationWitnessID},
			),
			p24RuntimeHardeningRow(
				RuntimeHardeningOOMPolicy,
				"OOM policy",
				"reviewed_runtime_contracts",
				[]string{
					("runtimeabi.AllocationFailureTrapOrStatus is required on " +
						"every RuntimeAllocationContract, including core.alloc_bytes," +
						" make_* slices, explicit islands, and region.temp."),
					("negative and overflow lengths reject before allocator " +
						"access, so OOM policy does not mask invalid preconditions " +
						"as allocator failure."),
				},
				[]string{
					"go test ./compiler/internal/runtimeabi/... -run 'RuntimeAllocationContract' -count=1",
					"go test ./compiler -run 'P24RuntimeHardening' -count=1",
				},
				[]string{
					"OOM recovery guarantee is not claimed; the current policy is stable trap/status handling",
					("allocator contracts do not prove every platform-specific " +
						"OOM path has identical process-level behavior"),
				},
				[]string{p24RuntimeHardeningAllocationWitnessID},
			),
			p24RuntimeHardeningRow(
				RuntimeHardeningStackOverflowGuard,
				"Stack overflow guard",
				"reviewed_boundary_with_blocker",
				[]string{
					("backend stack-depth consistency checks reject malformed " +
						"wasm/x64 lowering shapes before emitting invalid function " +
						"bodies."),
					("current evidence records stack-depth consistency, while " +
						"guard-page or recursion-depth runtime stack overflow " +
						"protection remains an explicit boundary."),
				},
				[]string{
					"go test ./compiler/internal/backend/x64abi -run 'Stack|ABI' -count=1",
					"go test ./compiler/tests/lowering -run 'ABI|Wasm' -count=1",
					"go test ./compiler -run 'P24RuntimeHardening' -count=1",
				},
				[]string{
					"full stack-overflow protection is not claimed",
					"no guard-page or recursion-depth runtime proof is promoted by this report",
				},
				[]string{p24RuntimeHardeningStackWitnessID},
			),
			p24RuntimeHardeningRow(
				RuntimeHardeningIntegerOverflowSemantics,
				"Integer overflow semantics audit",
				"reviewed_optimizer_and_allocator_boundary",
				[]string{
					("optimizer coverage keeps overflow-sensitive checkedNegI32 " +
						"and foldConstBinaryI32 cases unoptimized when the fold " +
						"would change i32 semantics."),
					("allocation contracts and allocplan evidence reject " +
						"byte-size overflow before allocation, and global const " +
						"diagnostics reject overflow in global const expression."),
				},
				[]string{
					"go test ./compiler/internal/opt -run 'CoreOptimization|Scalar|Mem2Reg' -count=1",
					("go test ./compiler/internal/allocplan " +
						"./compiler/internal/runtimeabi -run 'Overflow|Allocation' " +
						"-count=1"),
					"go test ./compiler/tests/semantics -run 'Const|FeatureRegistry' -count=1",
				},
				[]string{
					("this is a current optimizer/allocation audit, not a full " +
						"integer-overflow proof for the whole language"),
					("overflow-sensitive rewrites remain rejected instead of " +
						"normalized into a new runtime behavior"),
				},
				[]string{
					p24RuntimeHardeningOverflowWitnessID,
					p24RuntimeHardeningAllocationWitnessID,
				},
			),
			p24RuntimeHardeningRow(
				RuntimeHardeningAllocatorCorruptionInstrumentation,
				"Allocator corruption detection instrumentation",
				"reviewed_runtime_instrumentation",
				[]string{
					("runtimeabi contracts expose bounds_header debug " +
						"instrumentation for heap allocation roots and " +
						"raw-pointer-bounds-v1 metadata for checked raw pointer " +
						"derivation."),
					("runtimeabi.PerCoreSmallHeapAllocator rejects stale or " +
						"double free handles and records reuse metadata through " +
						"PerCoreSmallHeapAllocator reports."),
				},
				[]string{
					"go test ./compiler/internal/runtimeabi/... -run 'RawPointer|SmallHeap|Allocation' -count=1",
					"go test ./compiler -run 'P24RuntimeHardening' -count=1",
				},
				[]string{
					"full allocator-corruption detection proof is not claimed",
					("debug instrumentation evidence is bounded to current " +
						"runtime ABI models and small-heap stale-handle checks"),
				},
				[]string{p24RuntimeHardeningCorruptionWitnessID},
			),
			p24RuntimeHardeningRow(
				RuntimeHardeningRegionUseAfterFreeInstrumentation,
				"Region double-free/use-after-free instrumentation",
				"reviewed_runtime_instrumentation",
				[]string{
					("RuntimeAllocationContracts include " +
						"AllocationDebugDoubleFree and AllocationDebugUseAfterFree " +
						"for explicit island paths, and region.temp includes " +
						"AllocationDebugUseAfterFree plus AllocationDebugRegionReset " +
						"instrumentation."),
					("RuntimeRegionAllocatorConfig(true) reserves a debug header " +
						"and AlignRegionBytes rejects negative and overflow sizes " +
						"for region payloads."),
				},
				[]string{
					"go test ./compiler/internal/runtimeabi/... -run 'Region|Allocation' -count=1",
					"go test ./compiler -run 'P24RuntimeHardening' -count=1",
				},
				[]string{
					"region instrumentation does not claim a complete temporal-memory-safety proof",
					("future region double-free runtime execution evidence must " +
						"remain separate from current ABI instrumentation evidence"),
				},
				[]string{
					p24RuntimeHardeningRegionWitnessID,
					p24RuntimeHardeningAllocationWitnessID,
				},
			),
			p24RuntimeHardeningRow(
				RuntimeHardeningActorMailboxOverflowPolicy,
				"Actor mailbox overflow policy",
				"reviewed_boundary_with_blocker",
				[]string{
					("parallelrt.NewTypedMailbox records bounded capacity, " +
						"blocking_recv_yield backpressure metadata, FIFO receive, " +
						"and recoverable ErrMailboxFull when the typed mailbox model " +
						"is full."),
					("actorsrt.ActorRuntimeProductionBoundaryAudit records that " +
						"built-in message pool exhaustion returns checked -1 for " +
						"live overload and drained message pool entries are " +
						"reclaimed after receive."),
				},
				[]string{
					("go test ./compiler/internal/parallelrt " +
						"./compiler/internal/actorsrt -run " +
						"'Mailbox|ProductionBoundary|SchedulerModel' -count=1"),
					"go test ./compiler -run 'P24RuntimeHardening' -count=1",
				},
				[]string{
					"production actor-mailbox promotion is not claimed",
					("typed mailbox model policy is evidence for bounded " +
						"prototype behavior and does not promote the built-in actor " +
						"runtime message pool"),
				},
				[]string{p24RuntimeHardeningMailboxWitnessID},
			),
			p24RuntimeHardeningRow(
				RuntimeHardeningNetworkParserLimits,
				"Network parser limits",
				"reviewed_parser_limits",
				[]string{
					("httprt.ParseRequest and ParseRequestView return " +
						"deterministic ErrHeaderTooLarge, ErrTooManyHeaders, " +
						"ErrBodyTooLarge, malformed request/header, unsupported " +
						"version, and unsupported transfer-encoding errors."),
					("pgrt.ReadFrame rejects malformed frame lengths with " +
						"ErrMalformedFrame and oversized payloads with " +
						"ErrFrameTooLarge before allocating payload buffers."),
				},
				[]string{
					("go test ./compiler/internal/httprt ./compiler/internal/pgrt " +
						"-run 'ParseRequest|ReadFrame|RequestView' -count=1"),
					"go test ./compiler -run 'P24RuntimeHardening' -count=1",
				},
				[]string{
					("network parser limits are local HTTP/PostgreSQL parser " +
						"evidence, not a full production network-stack hardening " +
						"proof"),
					("TLS, channel binding, remote deployment, and all-protocol " +
						"parser hardening remain outside this report"),
				},
				[]string{p24RuntimeHardeningParserWitnessID},
			),
		},
		NonClaims: []string{
			"full runtime-hardening proof is not claimed",
			"full stack-overflow protection is not claimed",
			"OOM recovery guarantee is not claimed",
			"full allocator-corruption detection proof is not claimed",
			"production actor-mailbox promotion is not claimed",
			"runtime behavior does not change",
			"safe-program semantics do not change",
			"no performance claim is made",
		},
		DeterministicTrapsReviewed: trapWitness.TrapPolicyReviewed &&
			allocationWitness.OOMPolicyReviewed,
		OOMPolicyReviewed:                  allocationWitness.OOMPolicyReviewed,
		StackOverflowGuardReviewed:         stackWitness.StackOverflowGuardReviewed,
		IntegerOverflowSemanticsAudited:    overflowWitness.IntegerOverflowSemanticsAudited,
		AllocatorCorruptionReviewed:        corruptionWitness.AllocatorCorruptionReviewed,
		RegionLifetimeReviewed:             regionWitness.RegionLifetimeReviewed,
		ActorMailboxOverflowPolicyReviewed: mailboxWitness.ActorMailboxOverflowPolicyReviewed,
		NetworkParserLimitsReviewed:        parserWitness.NetworkParserLimitsReviewed,
		RuntimeHardeningArtifactPresent:    artifactWitness.RuntimeHardeningArtifactPresent,
		RuntimeHardeningDesignPresent:      artifactWitness.RuntimeHardeningDesignPresent,
	}
	if err := ValidateP24RuntimeHardeningV1Report(report); err != nil {
		return RuntimeHardeningV1Report{}, err
	}
	return report, nil
}

func ValidateP24RuntimeHardeningV1Report(report RuntimeHardeningV1Report) error {
	if report.SchemaVersion != runtimeHardeningV1Schema {
		return fmt.Errorf("runtime hardening v1: schema_version is %q", report.SchemaVersion)
	}
	if report.Scope != runtimeHardeningV1ScopeP241 {
		return fmt.Errorf("runtime hardening v1: scope is %q", report.Scope)
	}
	if report.FullRuntimeHardeningClaimed {
		return fmt.Errorf("runtime hardening v1: full runtime-hardening claim is forbidden")
	}
	if report.FullStackOverflowProtectionClaimed {
		return fmt.Errorf("runtime hardening v1: full stack-overflow protection claim is forbidden")
	}
	if report.FullOOMRecoveryClaimed {
		return fmt.Errorf("runtime hardening v1: OOM recovery claim is forbidden")
	}
	if report.FullAllocatorCorruptionDetectionClaimed {
		return fmt.Errorf(
			"runtime hardening v1: full allocator-corruption detection claim is forbidden",
		)
	}
	if report.ProductionActorMailboxClaimed {
		return fmt.Errorf("runtime hardening v1: production actor-mailbox claim is forbidden")
	}
	if report.RuntimeBehaviorChanged {
		return fmt.Errorf("runtime hardening v1: runtime behavior change claim is forbidden")
	}
	if report.SafeSemanticsChanged {
		return fmt.Errorf("runtime hardening v1: safe semantics change claim is forbidden")
	}
	if report.PerformanceClaimed {
		return fmt.Errorf("runtime hardening v1: performance claim is forbidden")
	}
	if !report.DeterministicTrapsReviewed {
		return fmt.Errorf("runtime hardening v1: deterministic traps review missing")
	}
	if !report.OOMPolicyReviewed {
		return fmt.Errorf("runtime hardening v1: OOM policy review missing")
	}
	if !report.StackOverflowGuardReviewed {
		return fmt.Errorf("runtime hardening v1: stack overflow guard review missing")
	}
	if !report.IntegerOverflowSemanticsAudited {
		return fmt.Errorf("runtime hardening v1: integer overflow semantics audit missing")
	}
	if !report.AllocatorCorruptionReviewed {
		return fmt.Errorf(
			"runtime hardening v1: allocator corruption instrumentation review missing",
		)
	}
	if !report.RegionLifetimeReviewed {
		return fmt.Errorf("runtime hardening v1: region double-free/use-after-free review missing")
	}
	if !report.ActorMailboxOverflowPolicyReviewed {
		return fmt.Errorf("runtime hardening v1: actor mailbox overflow policy review missing")
	}
	if !report.NetworkParserLimitsReviewed {
		return fmt.Errorf("runtime hardening v1: network parser limits review missing")
	}
	for _, want := range []string{
		"full runtime-hardening proof is not claimed",
		"full stack-overflow protection is not claimed",
		"OOM recovery guarantee is not claimed",
		"full allocator-corruption detection proof is not claimed",
		"production actor-mailbox promotion is not claimed",
		"runtime behavior does not change",
		"safe-program semantics do not change",
		"no performance claim is made",
	} {
		if !p24RuntimeHardeningHasString(report.NonClaims, want) {
			return fmt.Errorf("runtime hardening v1: missing non-claim %q", want)
		}
	}
	if err := p24RuntimeHardeningValidateArtifacts(report); err != nil {
		return err
	}
	if err := p24RuntimeHardeningValidateRowsAndWitnesses(report.Rows, report.Witnesses); err != nil {
		return err
	}
	return nil
}

func buildP24RuntimeHardeningTrapWitness() RuntimeHardeningV1Witness {
	paths := []string{
		"compiler/internal/backend/wasm32_wasi/codegen.go",
		"compiler/internal/backend/wasm32_wasi/codegen_helpers.go",
		"compiler/internal/backend/wasm32_web/codegen.go",
		"compiler/internal/backend/wasm32_web/codegen_helpers.go",
		"docs/spec/core/current_supported_surface.md",
	}
	wasmTrapEmitters := 0
	for _, path := range []string{
		"compiler/internal/backend/wasm32_wasi/codegen_helpers.go",
		"compiler/internal/backend/wasm32_web/codegen_helpers.go",
	} {
		if p24RuntimeHardeningFileContains(path, "emitWasmTrapIf") {
			wasmTrapEmitters++
		}
	}
	panicImport := p24RuntimeHardeningFileContains(
		"compiler/internal/backend/wasm32_web/codegen.go",
		"tetra panic",
	)
	return RuntimeHardeningV1Witness{
		ID:                 p24RuntimeHardeningTrapWitnessID,
		Kind:               "deterministic_trap_surface",
		Paths:              paths,
		TrapPolicyReviewed: p24AllRepoPathsExist(paths) && wasmTrapEmitters >= 2 && panicImport,
		WasmTrapEmitters:   wasmTrapEmitters,
		PanicImportPresent: panicImport,
	}
}

func buildP24RuntimeHardeningAllocationWitness() (RuntimeHardeningV1Witness, error) {
	contracts := runtimeabi.RuntimeAllocationContracts()
	var failureBehaviors int
	var overflowGuards int
	for _, contract := range contracts {
		if err := runtimeabi.ValidateRuntimeAllocationContract(contract); err != nil {
			return RuntimeHardeningV1Witness{}, err
		}
		if contract.FailureBehavior == runtimeabi.AllocationFailureTrapOrStatus {
			failureBehaviors++
		}
		if contract.OverflowBehavior != "" {
			overflowGuards++
		}
	}
	return RuntimeHardeningV1Witness{
		ID:   p24RuntimeHardeningAllocationWitnessID,
		Kind: "allocation_failure_surface",
		Paths: []string{
			"compiler/internal/runtimeabi/allocation_contract.go",
			"docs/design/runtime_allocation_contract.md",
		},
		AllocationContracts:          len(contracts),
		ContractsWithFailureBehavior: failureBehaviors,
		ContractsWithOverflowGuards:  overflowGuards,
		OOMPolicyReviewed: len(contracts) >= 5 && failureBehaviors == len(contracts) &&
			overflowGuards == len(contracts),
	}, nil
}

func buildP24RuntimeHardeningStackWitness() RuntimeHardeningV1Witness {
	paths := []string{
		"compiler/internal/backend/wasm32_wasi/codegen.go",
		"compiler/internal/backend/wasm32_web/codegen.go",
		"compiler/internal/backend/x64abi/abi_test.go",
		"compiler/tests/lowering/x64_abi_test.go",
	}
	stackDepthChecks := 0
	for _, path := range paths {
		if p24RuntimeHardeningFileContains(path, "stack depth") {
			stackDepthChecks++
		}
	}
	return RuntimeHardeningV1Witness{
		ID:                                 p24RuntimeHardeningStackWitnessID,
		Kind:                               "stack_overflow_boundary",
		Paths:                              paths,
		StackDepthChecks:                   stackDepthChecks,
		StackOverflowGuardReviewed:         p24AllRepoPathsExist(paths) && stackDepthChecks >= 2,
		FullStackOverflowProtectionClaimed: false,
	}
}

func buildP24RuntimeHardeningOverflowWitness() (RuntimeHardeningV1Witness, error) {
	allocation, err := buildP24RuntimeHardeningAllocationWitness()
	if err != nil {
		return RuntimeHardeningV1Witness{}, err
	}
	checkedNeg := p24RuntimeHardeningFileContains(
		"compiler/internal/opt/opt_core.go",
		"checkedNegI32",
	)
	foldConst := p24RuntimeHardeningFileContains(
		"compiler/internal/opt/opt_core.go",
		"foldConstBinaryI32",
	)
	constOverflow := p24RuntimeHardeningFileContains(
		"compiler/internal/semantics/semantics_checker.go",
		"overflow in global const expression",
	)
	return RuntimeHardeningV1Witness{
		ID:   p24RuntimeHardeningOverflowWitnessID,
		Kind: "integer_overflow_semantics",
		Paths: []string{
			"compiler/internal/opt/opt_core.go",
			"compiler/internal/opt/opt_core.go",
			"compiler/internal/semantics/semantics_checker.go",
			"compiler/internal/allocplan/plan.go",
			"compiler/internal/runtimeabi/allocation_contract.go",
		},
		CheckedNegI32Present:           checkedNeg,
		FoldConstBinaryI32Present:      foldConst,
		ConstOverflowDiagnosticPresent: constOverflow,
		AllocationOverflowGuards:       allocation.ContractsWithOverflowGuards,
		IntegerOverflowSemanticsAudited: checkedNeg && foldConst && constOverflow &&
			allocation.ContractsWithOverflowGuards >= 5,
	}, nil
}

func buildP24RuntimeHardeningCorruptionWitness() (RuntimeHardeningV1Witness, error) {
	contracts := runtimeabi.RuntimeAllocationContracts()
	var boundsHeaderContracts int
	for _, contract := range contracts {
		if err := runtimeabi.ValidateRuntimeAllocationContract(contract); err != nil {
			return RuntimeHardeningV1Witness{}, err
		}
		if contract.HasDebugInstrumentation(runtimeabi.AllocationDebugBoundsHeader) {
			boundsHeaderContracts++
		}
	}
	smallHeapDoubleFreeRejected, err := p24RuntimeHardeningSmallHeapRejectsDoubleFree()
	if err != nil {
		return RuntimeHardeningV1Witness{}, err
	}
	rawBounds := runtimeabi.RuntimeRawPointerBoundsABI()
	return RuntimeHardeningV1Witness{
		ID:   p24RuntimeHardeningCorruptionWitnessID,
		Kind: "allocator_corruption_instrumentation",
		Paths: []string{
			"compiler/internal/runtimeabi/allocation_contract.go",
			"compiler/internal/runtimeabi/smallheap/small_heap.go",
			"compiler/internal/runtimeabi/raw_pointer_bounds.go",
		},
		BoundsHeaderContracts:           boundsHeaderContracts,
		SmallHeapDoubleFreeRejected:     smallHeapDoubleFreeRejected,
		RawPointerBoundsMetadataVersion: rawBounds.MetadataVersion,
		AllocatorCorruptionReviewed: boundsHeaderContracts >= 1 &&
			smallHeapDoubleFreeRejected &&
			rawBounds.MetadataVersion == "raw-pointer-bounds-v1",
	}, nil
}

func buildP24RuntimeHardeningRegionWitness() (RuntimeHardeningV1Witness, error) {
	contracts := runtimeabi.RuntimeAllocationContracts()
	var useAfterFreeContracts int
	var doubleFreeContracts int
	var regionResetContracts int
	for _, contract := range contracts {
		if err := runtimeabi.ValidateRuntimeAllocationContract(contract); err != nil {
			return RuntimeHardeningV1Witness{}, err
		}
		if contract.HasDebugInstrumentation(runtimeabi.AllocationDebugUseAfterFree) {
			useAfterFreeContracts++
		}
		if contract.HasDebugInstrumentation(runtimeabi.AllocationDebugDoubleFree) {
			doubleFreeContracts++
		}
		if contract.HasDebugInstrumentation(runtimeabi.AllocationDebugRegionReset) {
			regionResetContracts++
		}
	}
	debugCfg := runtimeabi.RuntimeRegionAllocatorConfig(true)
	return RuntimeHardeningV1Witness{
		ID:   p24RuntimeHardeningRegionWitnessID,
		Kind: "region_lifetime_instrumentation",
		Paths: []string{
			"compiler/internal/runtimeabi/allocation_contract.go",
			"compiler/internal/runtimeabi/region_allocator.go",
		},
		RegionDebugHeaderBytes:      debugCfg.DebugHeaderBytes,
		RegionUseAfterFreeContracts: useAfterFreeContracts,
		RegionDoubleFreeContracts:   doubleFreeContracts,
		RegionResetContracts:        regionResetContracts,
		RegionLifetimeReviewed: debugCfg.DebugHeaderBytes > 0 &&
			useAfterFreeContracts >= 1 &&
			doubleFreeContracts >= 1 &&
			regionResetContracts >= 1,
	}, nil
}

func buildP24RuntimeHardeningMailboxWitness() (RuntimeHardeningV1Witness, error) {
	box := parallelrt.NewTypedMailbox(parallelrt.MailboxConfig{Name: "p24", Capacity: 1})
	if _, err := box.Send(parallelrt.Message{Name: "first"}); err != nil {
		return RuntimeHardeningV1Witness{}, err
	}
	_, overflowErr := box.Send(parallelrt.Message{Name: "second"})
	first, received := box.Receive()
	audit, err := actorsrt.ActorRuntimeProductionBoundaryAudit()
	if err != nil {
		return RuntimeHardeningV1Witness{}, err
	}
	if err := actorsrt.ValidateActorRuntimeProductionBoundaryAudit(audit); err != nil {
		return RuntimeHardeningV1Witness{}, err
	}
	builtinOverflowChecked := false
	drainedMessagesReclaimed := false
	oldUncheckedOverflowClaim := false
	for _, row := range audit.Rows {
		text := strings.Join(row.RequiredFacts, " ") + " " + row.Evidence + " " + row.Boundary
		if strings.Contains(text, "message pool overflow is not a checked runtime error") {
			oldUncheckedOverflowClaim = true
		}
		if strings.Contains(text, "message pool exhaustion returns checked -1") {
			builtinOverflowChecked = true
		}
		if strings.Contains(text, "drained message pool entries are reclaimed") {
			drainedMessagesReclaimed = true
		}
	}
	return RuntimeHardeningV1Witness{
		ID:   p24RuntimeHardeningMailboxWitnessID,
		Kind: "actor_mailbox_overflow_policy",
		Paths: []string{
			"compiler/internal/parallelrt/scheduler_model.go",
			"compiler/internal/actorsrt/actorsrt_core.go",
		},
		MailboxCapacity:                   box.Capacity(),
		BackpressureMode:                  box.Backpressure().Mode,
		MailboxOverflowRejected:           errors.Is(overflowErr, parallelrt.ErrMailboxFull),
		MailboxFIFOReceive:                received && first.Name == "first",
		ActorBoundaryRows:                 len(audit.Rows),
		BuiltinMessagePoolOverflowChecked: builtinOverflowChecked,
		ActorMailboxOverflowPolicyReviewed: box.Capacity() == 1 &&
			box.Backpressure().Mode == "blocking_recv_yield" &&
			errors.Is(overflowErr, parallelrt.ErrMailboxFull) &&
			received && first.Name == "first" &&
			len(audit.Rows) >= 4 &&
			builtinOverflowChecked &&
			drainedMessagesReclaimed &&
			!oldUncheckedOverflowClaim,
	}, nil
}

func buildP24RuntimeHardeningParserWitness() RuntimeHardeningV1Witness {
	httpLimits := httprt.Limits{MaxHeaderBytes: 64, MaxHeaders: 4, MaxBodyBytes: 4}
	_, _, httpHeaderErr := httprt.ParseRequest(
		[]byte("GET / HTTP/1.1\r\nLong: "+strings.Repeat("x", 80)+"\r\n\r\n"),
		httpLimits,
	)
	_, _, httpBodyErr := httprt.ParseRequest(
		[]byte("POST / HTTP/1.1\r\nContent-Length: 5\r\n\r\nhello"),
		httpLimits,
	)
	_, _, _, viewHeaderErr := httprt.ParseRequestView(
		[]byte("GET / HTTP/1.1\r\nLong: "+strings.Repeat("x", 80)+"\r\n\r\n"),
		httpLimits,
		nil,
	)
	_, pgMalformedErr := pgrt.ReadFrame(bytes.NewReader([]byte{'R', 0, 0, 0, 3}), 1024)
	_, pgLargeErr := pgrt.ReadFrame(bytes.NewReader([]byte{'R', 0, 0, 4, 1}), 8)

	httpReviewed := errors.Is(httpHeaderErr, httprt.ErrHeaderTooLarge) &&
		errors.Is(httpBodyErr, httprt.ErrBodyTooLarge)
	viewReviewed := errors.Is(viewHeaderErr, httprt.ErrHeaderTooLarge)
	pgReviewed := errors.Is(pgMalformedErr, pgrt.ErrMalformedFrame) &&
		errors.Is(pgLargeErr, pgrt.ErrFrameTooLarge)
	return RuntimeHardeningV1Witness{
		ID:   p24RuntimeHardeningParserWitnessID,
		Kind: "network_parser_limits",
		Paths: []string{
			"compiler/internal/httprt/http1.go",
			"compiler/internal/httprt/request_view.go",
			"compiler/internal/pgrt/wire.go",
		},
		HTTPParserLimitsReviewed:      httpReviewed,
		HTTPRequestViewLimitsReviewed: viewReviewed,
		PostgresFrameLimitsReviewed:   pgReviewed,
		NetworkParserLimitsReviewed:   httpReviewed && viewReviewed && pgReviewed,
	}
}

func buildP24RuntimeHardeningArtifactsWitness(
	artifacts []RuntimeHardeningArtifact,
) RuntimeHardeningV1Witness {
	witness := RuntimeHardeningV1Witness{
		ID:    p24RuntimeHardeningArtifactsWitnessID,
		Kind:  "runtime_hardening_artifacts",
		Paths: make([]string, 0, len(artifacts)),
	}
	for _, artifact := range artifacts {
		witness.Paths = append(witness.Paths, artifact.Path)
		switch artifact.Path {
		case "docs/audits/runtime/services/runtime-hardening-v1.md":
			witness.RuntimeHardeningArtifactPresent = artifact.Present
		case "docs/plans/2026-06-03/governance-p23-p24/2026-06-03-p24.1-runtime-hardening-design.md":
			witness.RuntimeHardeningDesignPresent = artifact.Present
		}
	}
	return witness
}

func p24RuntimeHardeningValidateRowsAndWitnesses(
	rows []RuntimeHardeningV1Row,
	witnesses []RuntimeHardeningV1Witness,
) error {
	byWitness := map[string]RuntimeHardeningV1Witness{}
	for _, witness := range witnesses {
		if strings.TrimSpace(witness.ID) == "" || strings.TrimSpace(witness.Kind) == "" {
			return fmt.Errorf("runtime hardening v1: witness missing id or kind")
		}
		if _, exists := byWitness[witness.ID]; exists {
			return fmt.Errorf("runtime hardening v1: duplicate witness %q", witness.ID)
		}
		byWitness[witness.ID] = witness
	}
	expected := map[RuntimeHardeningV1ID]bool{}
	for _, id := range p24RuntimeHardeningV1IDs() {
		expected[id] = true
	}
	seen := map[RuntimeHardeningV1ID]bool{}
	for _, row := range rows {
		if !expected[row.ID] {
			return fmt.Errorf("runtime hardening v1: unexpected row %q", row.ID)
		}
		if seen[row.ID] {
			return fmt.Errorf("runtime hardening v1: duplicate row %q", row.ID)
		}
		seen[row.ID] = true
		if strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Status) == "" {
			return fmt.Errorf("runtime hardening v1: row %q missing name or status", row.ID)
		}
		if len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 ||
			len(row.WitnessIDs) == 0 {
			return fmt.Errorf(
				"runtime hardening v1: row %q missing evidence, tests, boundaries, or witness ids",
				row.ID,
			)
		}
		for _, text := range append(
			append(append([]string{}, row.Evidence...), row.Tests...),
			row.Boundaries...,
		) {
			if p24RuntimeHardeningIsPlaceholder(text) {
				return fmt.Errorf("runtime hardening v1: row %q has placeholder evidence", row.ID)
			}
		}
		for _, id := range row.WitnessIDs {
			if _, ok := byWitness[id]; !ok {
				return fmt.Errorf(
					"runtime hardening v1: row %q references missing witness %q",
					row.ID,
					id,
				)
			}
		}
	}
	for _, id := range p24RuntimeHardeningV1IDs() {
		if !seen[id] {
			return fmt.Errorf("runtime hardening v1: missing row %q", id)
		}
	}
	if witness := byWitness[p24RuntimeHardeningTrapWitnessID]; !witness.TrapPolicyReviewed ||
		witness.WasmTrapEmitters < 2 ||
		!witness.PanicImportPresent {
		return fmt.Errorf("runtime hardening v1: deterministic trap witness incomplete")
	}
	if witness := byWitness[p24RuntimeHardeningAllocationWitnessID]; !witness.OOMPolicyReviewed ||
		witness.AllocationContracts < 5 ||
		witness.ContractsWithFailureBehavior != witness.AllocationContracts ||
		witness.ContractsWithOverflowGuards != witness.AllocationContracts {
		return fmt.Errorf("runtime hardening v1: allocation/OOM witness incomplete")
	}
	if witness := byWitness[p24RuntimeHardeningStackWitnessID]; !witness.StackOverflowGuardReviewed ||
		witness.StackDepthChecks < 2 ||
		witness.FullStackOverflowProtectionClaimed {
		return fmt.Errorf("runtime hardening v1: stack overflow boundary witness incomplete")
	}
	overflowWitness := byWitness[p24RuntimeHardeningOverflowWitnessID]
	if !overflowWitness.IntegerOverflowSemanticsAudited ||
		!overflowWitness.CheckedNegI32Present ||
		!overflowWitness.FoldConstBinaryI32Present ||
		!overflowWitness.ConstOverflowDiagnosticPresent ||
		overflowWitness.AllocationOverflowGuards < 5 {
		return fmt.Errorf("runtime hardening v1: integer overflow semantics witness incomplete")
	}
	corruptionWitness := byWitness[p24RuntimeHardeningCorruptionWitnessID]
	if !corruptionWitness.AllocatorCorruptionReviewed ||
		corruptionWitness.BoundsHeaderContracts < 1 ||
		!corruptionWitness.SmallHeapDoubleFreeRejected ||
		corruptionWitness.RawPointerBoundsMetadataVersion != "raw-pointer-bounds-v1" {
		return fmt.Errorf(
			"runtime hardening v1: allocator corruption instrumentation witness incomplete",
		)
	}
	regionWitness := byWitness[p24RuntimeHardeningRegionWitnessID]
	if !regionWitness.RegionLifetimeReviewed ||
		regionWitness.RegionDebugHeaderBytes <= 0 ||
		regionWitness.RegionUseAfterFreeContracts < 1 ||
		regionWitness.RegionDoubleFreeContracts < 1 ||
		regionWitness.RegionResetContracts < 1 {
		return fmt.Errorf(
			"runtime hardening v1: region lifetime instrumentation witness incomplete",
		)
	}
	mailboxWitness := byWitness[p24RuntimeHardeningMailboxWitnessID]
	if !mailboxWitness.ActorMailboxOverflowPolicyReviewed ||
		mailboxWitness.MailboxCapacity != 1 ||
		mailboxWitness.BackpressureMode != "blocking_recv_yield" ||
		!mailboxWitness.MailboxOverflowRejected ||
		!mailboxWitness.MailboxFIFOReceive ||
		mailboxWitness.ActorBoundaryRows < 4 ||
		!mailboxWitness.BuiltinMessagePoolOverflowChecked {
		return fmt.Errorf("runtime hardening v1: actor mailbox overflow witness incomplete")
	}
	parserWitness := byWitness[p24RuntimeHardeningParserWitnessID]
	if !parserWitness.NetworkParserLimitsReviewed ||
		!parserWitness.HTTPParserLimitsReviewed ||
		!parserWitness.HTTPRequestViewLimitsReviewed ||
		!parserWitness.PostgresFrameLimitsReviewed {
		return fmt.Errorf("runtime hardening v1: network parser limits witness incomplete")
	}
	artifactsWitness := byWitness[p24RuntimeHardeningArtifactsWitnessID]
	if !artifactsWitness.RuntimeHardeningArtifactPresent ||
		!artifactsWitness.RuntimeHardeningDesignPresent {
		return fmt.Errorf("runtime hardening v1: runtime hardening artifact witness incomplete")
	}
	return nil
}

func p24RuntimeHardeningValidateArtifacts(report RuntimeHardeningV1Report) error {
	if !report.RuntimeHardeningArtifactPresent {
		return fmt.Errorf(
			"runtime hardening v1: docs/audits/runtime/services/runtime-hardening-v1.md artifact missing",
		)
	}
	if !report.RuntimeHardeningDesignPresent {
		return fmt.Errorf(
			("runtime hardening v1: " +
				"docs/plans/2026-06-03/governance-p23-p24/2026-06-03-p24.1-ru" +
				"ntime-hardening-design.md artifact missing"),
		)
	}
	present := map[string]bool{}
	for _, artifact := range report.Artifacts {
		if strings.TrimSpace(artifact.Kind) == "" || strings.TrimSpace(artifact.Path) == "" {
			return fmt.Errorf("runtime hardening v1: artifact missing kind or path")
		}
		present[artifact.Path] = artifact.Present
	}
	for _, path := range []string{
		"docs/audits/runtime/services/runtime-hardening-v1.md",
		"docs/plans/2026-06-03/governance-p23-p24/2026-06-03-p24.1-runtime-hardening-design.md",
	} {
		if !present[path] {
			return fmt.Errorf("runtime hardening v1: required artifact %s missing", path)
		}
	}
	return nil
}

func p24RuntimeHardeningV1IDs() []RuntimeHardeningV1ID {
	return []RuntimeHardeningV1ID{
		RuntimeHardeningDeterministicTraps,
		RuntimeHardeningOOMPolicy,
		RuntimeHardeningStackOverflowGuard,
		RuntimeHardeningIntegerOverflowSemantics,
		RuntimeHardeningAllocatorCorruptionInstrumentation,
		RuntimeHardeningRegionUseAfterFreeInstrumentation,
		RuntimeHardeningActorMailboxOverflowPolicy,
		RuntimeHardeningNetworkParserLimits,
	}
}

func p24RuntimeHardeningRow(
	id RuntimeHardeningV1ID,
	name, status string,
	evidence, tests, boundaries, witnessIDs []string,
) RuntimeHardeningV1Row {
	return RuntimeHardeningV1Row{
		ID:         id,
		Name:       name,
		Status:     status,
		Evidence:   evidence,
		Tests:      tests,
		Boundaries: boundaries,
		WitnessIDs: witnessIDs,
	}
}

func p24RuntimeHardeningArtifacts() []RuntimeHardeningArtifact {
	return []RuntimeHardeningArtifact{
		p24RuntimeHardeningArtifact(
			"runtime_hardening_audit",
			"docs/audits/runtime/services/runtime-hardening-v1.md",
		),
		p24RuntimeHardeningArtifact(
			"runtime_hardening_design",
			"docs/plans/2026-06-03/governance-p23-p24/2026-06-03-p24.1-runtime-hardening-design.md",
		),
	}
}

func p24RuntimeHardeningArtifact(kind string, rel string) RuntimeHardeningArtifact {
	_, err := os.Stat(p24RepoPath(rel))
	return RuntimeHardeningArtifact{
		Kind:    kind,
		Path:    rel,
		Present: err == nil,
	}
}

func p24RuntimeHardeningSmallHeapRejectsDoubleFree() (bool, error) {
	allocator, err := runtimeabi.NewPerCoreSmallHeapAllocator(
		runtimeabi.RuntimePerCoreSmallHeapABI(1),
	)
	if err != nil {
		return false, err
	}
	handle, err := allocator.Alloc(0, 17)
	if err != nil {
		return false, err
	}
	if err := allocator.Free(handle); err != nil {
		return false, err
	}
	err = allocator.Free(handle)
	return err != nil && strings.Contains(err.Error(), "stale or double free"), nil
}

func p24RuntimeHardeningFileContains(rel string, want string) bool {
	data, err := os.ReadFile(p24RepoPath(rel))
	return err == nil && strings.Contains(string(data), want)
}

func p24RuntimeHardeningHasString(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

func p24RuntimeHardeningIsPlaceholder(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return lower == "" ||
		lower == "todo" ||
		lower == "tbd" ||
		strings.Contains(lower, "placeholder")
}

// ---- security_review_gate_v1.go ----

const (
	securityReviewGateV1Schema    = "tetra.security.review_gate.v1"
	securityReviewGateV1ScopeP240 = "p24.0_security_review_gate"

	p24SecurityReviewUnsafeWitnessID     = "unsafe_api_surface"
	p24SecurityReviewCapabilityWitnessID = "capability_surface"
	p24SecurityReviewAllocatorWitnessID  = "memory_allocator_surface"
	p24SecurityReviewNetworkWitnessID    = "network_runtime_surface"
	p24SecurityReviewActorWitnessID      = "actor_runtime_surface"
	p24SecurityReviewDBWitnessID         = "db_protocol_surface"
	p24SecurityReviewEcoWitnessID        = "package_eco_surface"
	p24SecurityReviewBuildWitnessID      = "build_script_surface"
	p24SecurityReviewSupplyWitnessID     = "supply_chain_surface"
	p24SecurityReviewArtifactsWitnessID  = "security_review_artifacts"
)

type SecurityReviewGateV1ID string

const (
	SecurityReviewUnsafeAPISurface  SecurityReviewGateV1ID = "unsafe_api_surface"
	SecurityReviewCapabilitySurface SecurityReviewGateV1ID = "capability_surface"
	SecurityReviewMemoryAllocator   SecurityReviewGateV1ID = "memory_allocator_surface"
	SecurityReviewNetworkRuntime    SecurityReviewGateV1ID = "network_runtime_surface"
	SecurityReviewActorRuntime      SecurityReviewGateV1ID = "actor_runtime_surface"
	SecurityReviewDBProtocol        SecurityReviewGateV1ID = "db_protocol_surface"
	SecurityReviewPackageEcoSystem  SecurityReviewGateV1ID = "package_eco_system"
	SecurityReviewBuildScripts      SecurityReviewGateV1ID = "build_scripts"
	SecurityReviewSupplyChain       SecurityReviewGateV1ID = "supply_chain"
	SecurityReviewArtifactSet       SecurityReviewGateV1ID = "security_review_artifacts"
)

type SecurityReviewGateV1Report struct {
	SchemaVersion string                        `json:"schema_version"`
	Scope         string                        `json:"scope"`
	Rows          []SecurityReviewGateV1Row     `json:"rows"`
	Witnesses     []SecurityReviewGateV1Witness `json:"witnesses"`
	Artifacts     []SecurityReviewArtifact      `json:"artifacts"`
	NonClaims     []string                      `json:"non_claims"`

	UnsafeAPISurfaceReviewed      bool `json:"unsafe_api_surface_reviewed"`
	CapabilitySurfaceReviewed     bool `json:"capability_surface_reviewed"`
	MemoryAllocatorReviewed       bool `json:"memory_allocator_reviewed"`
	NetworkRuntimeReviewed        bool `json:"network_runtime_reviewed"`
	ActorRuntimeReviewed          bool `json:"actor_runtime_reviewed"`
	DBProtocolReviewed            bool `json:"db_protocol_reviewed"`
	PackageEcoSystemReviewed      bool `json:"package_eco_system_reviewed"`
	BuildScriptsReviewed          bool `json:"build_scripts_reviewed"`
	SupplyChainReviewed           bool `json:"supply_chain_reviewed"`
	SecurityReviewArtifactPresent bool `json:"security_review_artifact_present"`
	ThreatModelArtifactPresent    bool `json:"threat_model_artifact_present"`
	UnsafeSurfaceMapPresent       bool `json:"unsafe_surface_map_present"`
	CapabilitySurfaceMapPresent   bool `json:"capability_surface_map_present"`
	SecurityCertifiedClaimed      bool `json:"security_certified_claimed"`
	ExternalPenTestClaimed        bool `json:"external_pen_test_claimed"`
	CVEFreeClaimed                bool `json:"cve_free_claimed"`
	ReleaseSignoffClaimed         bool `json:"release_signoff_claimed"`
	RuntimeBehaviorChanged        bool `json:"runtime_behavior_changed"`
	SafeSemanticsChanged          bool `json:"safe_semantics_changed"`
	PerformanceClaimed            bool `json:"performance_claimed"`
}

type SecurityReviewGateV1Row struct {
	ID         SecurityReviewGateV1ID `json:"id"`
	Name       string                 `json:"name"`
	Status     string                 `json:"status"`
	Evidence   []string               `json:"evidence"`
	Tests      []string               `json:"tests"`
	Boundaries []string               `json:"boundaries"`
	WitnessIDs []string               `json:"witness_ids"`
}

type SecurityReviewArtifact struct {
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Present bool   `json:"present"`
}

type SecurityReviewGateV1Witness struct {
	ID                              string   `json:"id"`
	Kind                            string   `json:"kind"`
	Paths                           []string `json:"paths,omitempty"`
	UnsafeAPISurfaceReviewed        bool     `json:"unsafe_api_surface_reviewed,omitempty"`
	CapabilitySurfaceReviewed       bool     `json:"capability_surface_reviewed,omitempty"`
	MemoryAllocatorReviewed         bool     `json:"memory_allocator_reviewed,omitempty"`
	RuntimeAllocationContracts      int      `json:"runtime_allocation_contracts,omitempty"`
	RawPointerBoundsMetadataVersion string   `json:"raw_pointer_bounds_metadata_version,omitempty"`
	NetworkRuntimeReviewed          bool     `json:"network_runtime_reviewed,omitempty"`
	IOReactorRows                   int      `json:"io_reactor_rows,omitempty"`
	ActorRuntimeReviewed            bool     `json:"actor_runtime_reviewed,omitempty"`
	ActorBoundaryRows               int      `json:"actor_boundary_rows,omitempty"`
	DBProtocolReviewed              bool     `json:"db_protocol_reviewed,omitempty"`
	ProductionPostgresRows          int      `json:"production_postgres_rows,omitempty"`
	PackageEcoSystemReviewed        bool     `json:"package_eco_system_reviewed,omitempty"`
	EcoValidatorPaths               int      `json:"eco_validator_paths,omitempty"`
	BuildScriptsReviewed            bool     `json:"build_scripts_reviewed,omitempty"`
	ReleaseSecurityScripts          int      `json:"release_security_scripts,omitempty"`
	SupplyChainReviewed             bool     `json:"supply_chain_reviewed,omitempty"`
	SupplyChainEvidencePaths        int      `json:"supply_chain_evidence_paths,omitempty"`
	SecurityReviewArtifactPresent   bool     `json:"security_review_artifact_present,omitempty"`
	ThreatModelArtifactPresent      bool     `json:"threat_model_artifact_present,omitempty"`
	UnsafeSurfaceMapPresent         bool     `json:"unsafe_surface_map_present,omitempty"`
	CapabilitySurfaceMapPresent     bool     `json:"capability_surface_map_present,omitempty"`
}

func BuildP24SecurityReviewGateV1Report() (SecurityReviewGateV1Report, error) {
	unsafeWitness := buildP24UnsafeWitness()
	capabilityWitness := buildP24CapabilityWitness()
	allocatorWitness, err := buildP24AllocatorWitness()
	if err != nil {
		return SecurityReviewGateV1Report{}, err
	}
	networkWitness, err := buildP24NetworkWitness()
	if err != nil {
		return SecurityReviewGateV1Report{}, err
	}
	actorWitness, err := buildP24ActorWitness()
	if err != nil {
		return SecurityReviewGateV1Report{}, err
	}
	dbWitness, err := buildP24DBWitness()
	if err != nil {
		return SecurityReviewGateV1Report{}, err
	}
	ecoWitness := buildP24EcoWitness()
	buildWitness := buildP24BuildScriptsWitness()
	supplyWitness := buildP24SupplyChainWitness()
	artifacts := p24SecurityReviewArtifacts()
	artifactWitness := buildP24ArtifactsWitness(artifacts)

	report := SecurityReviewGateV1Report{
		SchemaVersion: securityReviewGateV1Schema,
		Scope:         securityReviewGateV1ScopeP240,
		Witnesses: []SecurityReviewGateV1Witness{
			unsafeWitness,
			capabilityWitness,
			allocatorWitness,
			networkWitness,
			actorWitness,
			dbWitness,
			ecoWitness,
			buildWitness,
			supplyWitness,
			artifactWitness,
		},
		Artifacts: artifacts,
		Rows: []SecurityReviewGateV1Row{
			p24SecurityReviewGateRow(
				SecurityReviewUnsafeAPISurface,
				"Unsafe API surface",
				"reviewed_current_surface",
				[]string{
					("docs/spec/runtime/unsafe.md records unsafe-only builtins " +
						"including core.cap_mem, core.cap_io, core.alloc_bytes, " +
						"pointer arithmetic, load/store, MMIO, symbol address, " +
						"context switch, and island operations."),
					("Unsafe APIs remain gated by explicit unsafe syntax and " +
						"capability/effect requirements where applicable."),
				},
				[]string{
					"go test ./compiler -run 'P24SecurityReviewGate' -count=1",
					"go test ./compiler/... -run 'Unsafe|Capability|Effect|MMIO|Mem' -count=1",
					"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
				},
				[]string{
					"this is an inventory and policy review, not a proof that all unsafe callers are memory safe",
					"unsafe syntax remains required for unsafe-only builtins",
				},
				[]string{p24SecurityReviewUnsafeWitnessID},
			),
			p24SecurityReviewGateRow(
				SecurityReviewCapabilitySurface,
				"Capability surface",
				"reviewed_current_surface",
				[]string{
					("docs/spec/runtime/capabilities.md and " +
						"docs/spec/runtime/effects_capabilities_privacy_v1.md define " +
						"cap.mem, cap.io, uses propagation, and attenuation checks."),
					"uses declarations remain audit metadata and do not manufacture capability tokens.",
				},
				[]string{
					"go test ./compiler/... -run 'Capability|Effect|Uses|Capsule' -count=1",
					"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
				},
				[]string{
					"cap.mem is permission, not provenance, lifetime, bounds, alias, or sendability proof",
					"privacy consent tokens are separate from cap.mem and cap.io",
				},
				[]string{p24SecurityReviewCapabilityWitnessID},
			),
			p24SecurityReviewGateRow(
				SecurityReviewMemoryAllocator,
				"Memory allocator surface",
				"reviewed_runtime_contracts",
				[]string{
					("runtimeabi.RuntimeAllocationContracts validates " +
						"core.alloc_bytes, slice builders, islands, regions, guard " +
						"behavior, failure behavior, debug instrumentation, and " +
						"report hooks."),
					("runtimeabi.RuntimeRawPointerBoundsABI exposes " +
						"raw-pointer-bounds-v1 metadata for allocation roots, " +
						"derived offsets, external unknown pointers, and rejected " +
						"impossible ptr_add cases."),
				},
				[]string{
					("go test ./compiler/internal/runtimeabi/... -run " +
						"'Allocation|Region|SmallHeap|RawPointer' -count=1"),
					"go test ./compiler -run 'P24SecurityReviewGate' -count=1",
				},
				[]string{
					"allocator contracts are runtime ABI evidence, not a formal memory-safety proof",
					("external unknown raw pointers remain bounded as unknown " +
						"rather than promoted to verified allocation roots"),
				},
				[]string{p24SecurityReviewAllocatorWitnessID},
			),
			p24SecurityReviewGateRow(
				SecurityReviewNetworkRuntime,
				"Network runtime surface",
				"reviewed_runtime_boundary",
				[]string{
					("netrt.IOReactorCoverage validates Linux epoll, readiness " +
						"polling, nonblocking accept/read/write, I/O task wakeups, " +
						"timer, cancellation, backpressure, HTTP smoke, DB smoke, " +
						"and stress evidence."),
					("Linux epoll is current narrow evidence; cross-platform " +
						"parity and io_uring remain non-claims."),
				},
				[]string{
					"go test ./compiler/internal/netrt -run 'IOReactor|Poller|Readiness|Backpressure' -count=1",
					"go test ./compiler -run 'P24SecurityReviewGate' -count=1",
				},
				[]string{
					("network runtime review is bounded to current netrt evidence " +
						"and does not claim full production web-stack security"),
					"kqueue, IOCP, WASI/web event adapters, and io_uring remain documented boundaries",
				},
				[]string{p24SecurityReviewNetworkWitnessID},
			),
			p24SecurityReviewGateRow(
				SecurityReviewActorRuntime,
				"Actor runtime surface",
				"reviewed_runtime_boundary",
				[]string{
					("actorsrt.ActorRuntimeProductionBoundaryAudit records " +
						"current actor runtime limits, scheduler prototype features, " +
						"production acceptance requirements, and full-claim blockers."),
					("Current evidence records message pool limits and explicitly " +
						"states scheduler prototype evidence is not a production " +
						"multi-threaded actor scheduler."),
				},
				[]string{
					("go test ./compiler/internal/actorsrt " +
						"./compiler/internal/parallelrt -run " +
						"'ActorRuntime|ProductionBoundary|SchedulerModel' -count=1"),
					"go test ./compiler -run 'P24SecurityReviewGate' -count=1",
				},
				[]string{
					("actor runtime review does not promote distributed actor " +
						"support or production broker deployment"),
					("message pool exhaustion/reclamation and full race-safety " +
						"proof remain blockers for a production actor runtime claim"),
				},
				[]string{p24SecurityReviewActorWitnessID},
			),
			p24SecurityReviewGateRow(
				SecurityReviewDBProtocol,
				"DB protocol surface",
				"reviewed_protocol_boundary",
				[]string{
					("pgrt.ProductionPostgresCoverage validates SCRAM-SHA-256 " +
						"startup, prepared statements, binary protocol, pooling " +
						"backpressure, borrowed row decode, endpoint workloads, and " +
						"benchmark honesty rows."),
					("compiler/internal/pgrt/wire.go rejects malformed frames " +
						"with ErrMalformedFrame and oversized payloads with " +
						"ErrFrameTooLarge; pool.go returns ErrPoolExhausted instead " +
						"of opening past maxOpen."),
				},
				[]string{
					"go test ./compiler/internal/pgrt -run 'ProductionPostgres|SCRAM|Frame|Pool' -count=1",
					"go test ./compiler -run 'P24SecurityReviewGate' -count=1",
				},
				[]string{
					("DB protocol review is local PostgreSQL wire-protocol " +
						"compatibility evidence, not TLS, channel binding, or " +
						"external production database deployment evidence"),
					("official TechEmpower and production database benchmark " +
						"claims remain forbidden by the pgrt coverage validator"),
				},
				[]string{p24SecurityReviewDBWitnessID},
			),
			p24SecurityReviewGateRow(
				SecurityReviewPackageEcoSystem,
				"Package/Eco system surface",
				"reviewed_local_supply_surface",
				[]string{
					("docs/spec/policy/eco_publishing_v1.md defines " +
						"tetra.eco.publish.v1, Tetra.lock hash semantics, permission " +
						"escalation checks, artifact hashes, trust snapshots, " +
						"materialization metadata, and reproducible packaging basics."),
					("tools/cmd/validate-eco-lock, validate-eco-publish, " +
						"validate-eco-vault, validate-eco-mirror, and " +
						"validate-eco-unpack reject schema drift, unsafe paths, hash " +
						"mismatches, unknown fields, and tampered package content."),
				},
				[]string{
					"go test ./cli/... ./tools/... -run 'Eco|Permission|Capsule|Trust' -count=1",
					"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
				},
				[]string{
					"Eco/Todex trust is local metadata and validator evidence, not a global package trust network",
					"proof-carrying capsules and distributed EcoNet remain outside the current claim",
				},
				[]string{p24SecurityReviewEcoWitnessID},
			),
			p24SecurityReviewGateRow(
				SecurityReviewBuildScripts,
				"Build and release script surface",
				"reviewed_release_validator_boundary",
				[]string{
					("scripts/release/v1_0/security-review.sh checks " +
						"current_release_version, reviewed commit, Decision, " +
						"Evidence Commands, Artifact Hashes, and Residual Risks for " +
						"release signoff files."),
					("tools/scriptstest/release_v10/artifacts/security_review_test" +
						".go rejects template signoffs and stale review metadata for " +
						"the release security-review script family."),
				},
				[]string{
					"go test ./tools/scriptstest -run 'SecurityReview' -count=1",
					"bash scripts/release/v1_0/security-review.sh --help",
				},
				[]string{
					"P24.0 security-review.md is an audit artifact and does not count as a release signoff file",
					"release signoff still requires the release script validator over a release report directory",
				},
				[]string{p24SecurityReviewBuildWitnessID},
			),
			p24SecurityReviewGateRow(
				SecurityReviewSupplyChain,
				"Supply-chain surface",
				"reviewed_local_hash_boundary",
				[]string{
					("go.sum pins Go module checksums for this repository and Eco " +
						"validators require sha256 metadata for locks, packages, " +
						"trust snapshot files, mirrors, vault objects, and unpacked " +
						"package content."),
					("docs/spec/policy/eco_publishing_v1.md records trust " +
						"snapshot and local artifact hash boundaries; no network " +
						"trust claim is made for remote registries or global package " +
						"identity."),
				},
				[]string{
					("go test ./tools/cmd/validate-eco-lock " +
						"./tools/cmd/validate-eco-publish " +
						"./tools/cmd/validate-eco-vault " +
						"./tools/cmd/validate-eco-mirror " +
						"./tools/cmd/validate-eco-unpack -count=1"),
					"go test ./compiler -run 'P24SecurityReviewGate' -count=1",
				},
				[]string{
					("supply-chain evidence is local lock/hash/metadata " +
						"validation, not SLSA certification or external registry " +
						"trust"),
					("remote fetch/mirror paths must validate package bytes and " +
						"metadata before writing local store files"),
				},
				[]string{p24SecurityReviewSupplyWitnessID},
			),
			p24SecurityReviewGateRow(
				SecurityReviewArtifactSet,
				"Security review artifact set",
				"required_artifacts_present",
				[]string{
					("docs/audits/security/security-review.md summarizes the " +
						"P24.0 review with evidence, residual risks, and commands."),
					("docs/audits/security/threat-model.md records assets, trust " +
						"boundaries, attacker capabilities, abuse paths, mitigations," +
						" assumptions, and open questions."),
					("docs/audits/security/unsafe-surface-map.md maps unsafe " +
						"builtins, required syntax/effects/capabilities, owners, " +
						"tests, and residual risks."),
					("docs/audits/security/capability-surface-map.md maps cap.io, " +
						"cap.mem, privacy consent, capsule attenuation, permission " +
						"metadata, and local Eco trust boundaries."),
				},
				[]string{
					"go test ./compiler -run 'P24SecurityReviewGate' -count=1",
					"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
				},
				[]string{
					"artifacts are current-branch review artifacts, not external audit reports or release signoff",
					"future promotion must update artifact hashes in the release report directory",
				},
				[]string{p24SecurityReviewArtifactsWitnessID},
			),
		},
		NonClaims: []string{
			"security certification is not claimed",
			"external penetration test is not claimed",
			"CVE-free status is not claimed",
			"release security signoff is not claimed",
			"runtime behavior does not change",
			"safe-program semantics do not change",
			"no performance claim is made",
		},
		UnsafeAPISurfaceReviewed:      unsafeWitness.UnsafeAPISurfaceReviewed,
		CapabilitySurfaceReviewed:     capabilityWitness.CapabilitySurfaceReviewed,
		MemoryAllocatorReviewed:       allocatorWitness.MemoryAllocatorReviewed,
		NetworkRuntimeReviewed:        networkWitness.NetworkRuntimeReviewed,
		ActorRuntimeReviewed:          actorWitness.ActorRuntimeReviewed,
		DBProtocolReviewed:            dbWitness.DBProtocolReviewed,
		PackageEcoSystemReviewed:      ecoWitness.PackageEcoSystemReviewed,
		BuildScriptsReviewed:          buildWitness.BuildScriptsReviewed,
		SupplyChainReviewed:           supplyWitness.SupplyChainReviewed,
		SecurityReviewArtifactPresent: artifactWitness.SecurityReviewArtifactPresent,
		ThreatModelArtifactPresent:    artifactWitness.ThreatModelArtifactPresent,
		UnsafeSurfaceMapPresent:       artifactWitness.UnsafeSurfaceMapPresent,
		CapabilitySurfaceMapPresent:   artifactWitness.CapabilitySurfaceMapPresent,
		SecurityCertifiedClaimed:      false,
		ExternalPenTestClaimed:        false,
		CVEFreeClaimed:                false,
		ReleaseSignoffClaimed:         false,
		RuntimeBehaviorChanged:        false,
		SafeSemanticsChanged:          false,
		PerformanceClaimed:            false,
	}
	if err := ValidateP24SecurityReviewGateV1Report(report); err != nil {
		return SecurityReviewGateV1Report{}, err
	}
	return report, nil
}

func ValidateP24SecurityReviewGateV1Report(report SecurityReviewGateV1Report) error {
	if report.SchemaVersion != securityReviewGateV1Schema {
		return fmt.Errorf("security review gate v1: schema_version is %q", report.SchemaVersion)
	}
	if report.Scope != securityReviewGateV1ScopeP240 {
		return fmt.Errorf("security review gate v1: scope is %q", report.Scope)
	}
	if report.SecurityCertifiedClaimed {
		return fmt.Errorf("security review gate v1: security certification claim is forbidden")
	}
	if report.ExternalPenTestClaimed {
		return fmt.Errorf("security review gate v1: external penetration test claim is forbidden")
	}
	if report.CVEFreeClaimed {
		return fmt.Errorf("security review gate v1: CVE-free claim is forbidden")
	}
	if report.ReleaseSignoffClaimed {
		return fmt.Errorf("security review gate v1: release signoff claim is forbidden")
	}
	if report.RuntimeBehaviorChanged {
		return fmt.Errorf("security review gate v1: runtime behavior change claim is forbidden")
	}
	if report.SafeSemanticsChanged {
		return fmt.Errorf("security review gate v1: safe semantics change claim is forbidden")
	}
	if report.PerformanceClaimed {
		return fmt.Errorf("security review gate v1: performance claim is forbidden")
	}
	if !report.UnsafeAPISurfaceReviewed {
		return fmt.Errorf("security review gate v1: unsafe API surface review missing")
	}
	if !report.CapabilitySurfaceReviewed {
		return fmt.Errorf("security review gate v1: capability surface review missing")
	}
	if !report.MemoryAllocatorReviewed {
		return fmt.Errorf("security review gate v1: memory allocator review missing")
	}
	if !report.NetworkRuntimeReviewed {
		return fmt.Errorf("security review gate v1: network runtime review missing")
	}
	if !report.ActorRuntimeReviewed {
		return fmt.Errorf("security review gate v1: actor runtime review missing")
	}
	if !report.DBProtocolReviewed {
		return fmt.Errorf("security review gate v1: DB protocol review missing")
	}
	if !report.PackageEcoSystemReviewed {
		return fmt.Errorf("security review gate v1: package/Eco system review missing")
	}
	if !report.BuildScriptsReviewed {
		return fmt.Errorf("security review gate v1: build scripts review missing")
	}
	if !report.SupplyChainReviewed {
		return fmt.Errorf("security review gate v1: supply chain review missing")
	}
	if err := p24SecurityReviewValidateArtifacts(report); err != nil {
		return err
	}
	for _, want := range []string{
		"security certification is not claimed",
		"external penetration test is not claimed",
		"CVE-free status is not claimed",
		"release security signoff is not claimed",
		"runtime behavior does not change",
		"safe-program semantics do not change",
		"no performance claim is made",
	} {
		if !p24SecurityReviewHasString(report.NonClaims, want) {
			return fmt.Errorf("security review gate v1: missing non-claim %q", want)
		}
	}
	if err := p24SecurityReviewValidateRowsAndWitnesses(report.Rows, report.Witnesses); err != nil {
		return err
	}
	return nil
}

func buildP24UnsafeWitness() SecurityReviewGateV1Witness {
	paths := []string{
		"docs/spec/runtime/unsafe.md",
		"examples/flow/flow_unsafe_cap_mem_smoke.tetra",
		"lib/core/base/capability.tetra",
	}
	return SecurityReviewGateV1Witness{
		ID:                       p24SecurityReviewUnsafeWitnessID,
		Kind:                     "unsafe_api_surface",
		Paths:                    paths,
		UnsafeAPISurfaceReviewed: p24AllRepoPathsExist(paths),
	}
}

func buildP24CapabilityWitness() SecurityReviewGateV1Witness {
	paths := []string{
		"docs/spec/runtime/capabilities.md",
		"docs/spec/runtime/effects_capabilities_privacy_v1.md",
		"examples/core/memory/core_capability_smoke.tetra",
		"lib/core/base/capability.tetra",
	}
	return SecurityReviewGateV1Witness{
		ID:                        p24SecurityReviewCapabilityWitnessID,
		Kind:                      "capability_surface",
		Paths:                     paths,
		CapabilitySurfaceReviewed: p24AllRepoPathsExist(paths),
	}
}

func buildP24AllocatorWitness() (SecurityReviewGateV1Witness, error) {
	contracts := runtimeabi.RuntimeAllocationContracts()
	for _, contract := range contracts {
		if err := runtimeabi.ValidateRuntimeAllocationContract(contract); err != nil {
			return SecurityReviewGateV1Witness{}, err
		}
	}
	rawBounds := runtimeabi.RuntimeRawPointerBoundsABI()
	return SecurityReviewGateV1Witness{
		ID:   p24SecurityReviewAllocatorWitnessID,
		Kind: "memory_allocator_surface",
		Paths: []string{
			"compiler/internal/runtimeabi/allocation_contract.go",
			"compiler/internal/runtimeabi/raw_pointer_bounds.go",
		},
		MemoryAllocatorReviewed: len(contracts) >= 5 &&
			rawBounds.MetadataVersion == "raw-pointer-bounds-v1",
		RuntimeAllocationContracts:      len(contracts),
		RawPointerBoundsMetadataVersion: rawBounds.MetadataVersion,
	}, nil
}

func buildP24NetworkWitness() (SecurityReviewGateV1Witness, error) {
	report, err := netrt.IOReactorCoverage()
	if err != nil {
		return SecurityReviewGateV1Witness{}, err
	}
	if err := netrt.ValidateIOReactorCoverage(report); err != nil {
		return SecurityReviewGateV1Witness{}, err
	}
	return SecurityReviewGateV1Witness{
		ID:   p24SecurityReviewNetworkWitnessID,
		Kind: "network_runtime_surface",
		Paths: []string{
			"compiler/internal/netrt/io_reactor_coverage.go",
			"compiler/internal/netrt/netrt_linux.go",
		},
		NetworkRuntimeReviewed: len(report.Rows) >= 10 && !report.FullProductionWebStackClaimed &&
			!report.CrossPlatformParityClaimed,
		IOReactorRows: len(report.Rows),
	}, nil
}

func buildP24ActorWitness() (SecurityReviewGateV1Witness, error) {
	report, err := actorsrt.ActorRuntimeProductionBoundaryAudit()
	if err != nil {
		return SecurityReviewGateV1Witness{}, err
	}
	if err := actorsrt.ValidateActorRuntimeProductionBoundaryAudit(report); err != nil {
		return SecurityReviewGateV1Witness{}, err
	}
	return SecurityReviewGateV1Witness{
		ID:   p24SecurityReviewActorWitnessID,
		Kind: "actor_runtime_surface",
		Paths: []string{
			"compiler/internal/actorsrt/actorsrt_core.go",
			"docs/spec/runtime/actors.md",
		},
		ActorRuntimeReviewed: len(report.Rows) >= 4 && !report.FullProductionClaimed,
		ActorBoundaryRows:    len(report.Rows),
	}, nil
}

func buildP24DBWitness() (SecurityReviewGateV1Witness, error) {
	report, err := pgrt.ProductionPostgresCoverage()
	if err != nil {
		return SecurityReviewGateV1Witness{}, err
	}
	if err := pgrt.ValidateProductionPostgresCoverage(report); err != nil {
		return SecurityReviewGateV1Witness{}, err
	}
	return SecurityReviewGateV1Witness{
		ID:   p24SecurityReviewDBWitnessID,
		Kind: "db_protocol_surface",
		Paths: []string{
			"compiler/internal/pgrt/production_postgres_coverage.go",
			"compiler/internal/pgrt/wire.go",
			"compiler/internal/pgrt/scram.go",
			"compiler/internal/pgrt/pool.go",
		},
		DBProtocolReviewed: len(report.Rows) >= 8 && !report.ExternalProductionDatabaseClaimed &&
			!report.FullSourceLevelDriverClaimed,
		ProductionPostgresRows: len(report.Rows),
	}, nil
}

func buildP24EcoWitness() SecurityReviewGateV1Witness {
	paths := []string{
		"docs/spec/policy/eco_publishing_v1.md",
		"cli/cmd/tetra/tetra_eco.go",
		"cli/cmd/tetra/tetra_eco.go",
		"tools/cmd/validate-eco-lock/main.go",
		"tools/cmd/validate-eco-publish/main.go",
		"tools/cmd/validate-eco-vault/main.go",
		"tools/cmd/validate-eco-mirror/main.go",
		"tools/cmd/validate-eco-unpack/main.go",
	}
	return SecurityReviewGateV1Witness{
		ID:                       p24SecurityReviewEcoWitnessID,
		Kind:                     "package_eco_surface",
		Paths:                    paths,
		PackageEcoSystemReviewed: p24AllRepoPathsExist(paths),
		EcoValidatorPaths:        len(paths) - 3,
	}
}

func buildP24BuildScriptsWitness() SecurityReviewGateV1Witness {
	paths := []string{
		"scripts/release/v1_0/security-review.sh",
		"scripts/release/v0_4_0/security-review.sh",
		"scripts/release/v0_3_0/security-review.sh",
		"tools/scriptstest/release_v10/artifacts/security_review_test.go",
	}
	return SecurityReviewGateV1Witness{
		ID:                     p24SecurityReviewBuildWitnessID,
		Kind:                   "build_script_surface",
		Paths:                  paths,
		BuildScriptsReviewed:   p24AllRepoPathsExist(paths),
		ReleaseSecurityScripts: 3,
	}
}

func buildP24SupplyChainWitness() SecurityReviewGateV1Witness {
	paths := []string{
		"go.sum",
		"docs/spec/policy/eco_publishing_v1.md",
		"tools/cmd/validate-eco-lock/main.go",
		"tools/cmd/validate-eco-publish/main.go",
		"tools/cmd/validate-eco-vault/main.go",
	}
	return SecurityReviewGateV1Witness{
		ID:                       p24SecurityReviewSupplyWitnessID,
		Kind:                     "supply_chain_surface",
		Paths:                    paths,
		SupplyChainReviewed:      p24AllRepoPathsExist(paths),
		SupplyChainEvidencePaths: len(paths),
	}
}

func buildP24ArtifactsWitness(artifacts []SecurityReviewArtifact) SecurityReviewGateV1Witness {
	witness := SecurityReviewGateV1Witness{
		ID:    p24SecurityReviewArtifactsWitnessID,
		Kind:  "security_review_artifacts",
		Paths: make([]string, 0, len(artifacts)),
	}
	for _, artifact := range artifacts {
		witness.Paths = append(witness.Paths, artifact.Path)
		switch artifact.Path {
		case "docs/audits/security/security-review.md":
			witness.SecurityReviewArtifactPresent = artifact.Present
		case "docs/audits/security/threat-model.md":
			witness.ThreatModelArtifactPresent = artifact.Present
		case "docs/audits/security/unsafe-surface-map.md":
			witness.UnsafeSurfaceMapPresent = artifact.Present
		case "docs/audits/security/capability-surface-map.md":
			witness.CapabilitySurfaceMapPresent = artifact.Present
		}
	}
	return witness
}

func p24SecurityReviewValidateRowsAndWitnesses(
	rows []SecurityReviewGateV1Row,
	witnesses []SecurityReviewGateV1Witness,
) error {
	byWitness := map[string]SecurityReviewGateV1Witness{}
	for _, witness := range witnesses {
		if strings.TrimSpace(witness.ID) == "" || strings.TrimSpace(witness.Kind) == "" {
			return fmt.Errorf("security review gate v1: witness missing id or kind")
		}
		if _, exists := byWitness[witness.ID]; exists {
			return fmt.Errorf("security review gate v1: duplicate witness %q", witness.ID)
		}
		byWitness[witness.ID] = witness
	}
	expected := map[SecurityReviewGateV1ID]bool{}
	for _, id := range p24SecurityReviewGateV1IDs() {
		expected[id] = true
	}
	seen := map[SecurityReviewGateV1ID]bool{}
	for _, row := range rows {
		if !expected[row.ID] {
			return fmt.Errorf("security review gate v1: unexpected row %q", row.ID)
		}
		if seen[row.ID] {
			return fmt.Errorf("security review gate v1: duplicate row %q", row.ID)
		}
		seen[row.ID] = true
		if strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Status) == "" {
			return fmt.Errorf("security review gate v1: row %q missing name or status", row.ID)
		}
		if len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 ||
			len(row.WitnessIDs) == 0 {
			return fmt.Errorf(
				"security review gate v1: row %q missing evidence, tests, boundaries, or witness ids",
				row.ID,
			)
		}
		for _, text := range append(
			append(append([]string{}, row.Evidence...), row.Tests...),
			row.Boundaries...,
		) {
			if p24SecurityReviewIsPlaceholder(text) {
				return fmt.Errorf(
					"security review gate v1: row %q has placeholder evidence",
					row.ID,
				)
			}
		}
		for _, id := range row.WitnessIDs {
			if _, ok := byWitness[id]; !ok {
				return fmt.Errorf(
					"security review gate v1: row %q references missing witness %q",
					row.ID,
					id,
				)
			}
		}
	}
	for _, id := range p24SecurityReviewGateV1IDs() {
		if !seen[id] {
			return fmt.Errorf("security review gate v1: missing row %q", id)
		}
	}
	if !byWitness[p24SecurityReviewUnsafeWitnessID].UnsafeAPISurfaceReviewed {
		return fmt.Errorf("security review gate v1: unsafe API witness incomplete")
	}
	if !byWitness[p24SecurityReviewCapabilityWitnessID].CapabilitySurfaceReviewed {
		return fmt.Errorf("security review gate v1: capability witness incomplete")
	}
	allocator := byWitness[p24SecurityReviewAllocatorWitnessID]
	if !allocator.MemoryAllocatorReviewed || allocator.RuntimeAllocationContracts < 5 ||
		allocator.RawPointerBoundsMetadataVersion != "raw-pointer-bounds-v1" {
		return fmt.Errorf("security review gate v1: memory allocator witness incomplete")
	}
	network := byWitness[p24SecurityReviewNetworkWitnessID]
	if !network.NetworkRuntimeReviewed || network.IOReactorRows < 10 {
		return fmt.Errorf("security review gate v1: network runtime witness incomplete")
	}
	actor := byWitness[p24SecurityReviewActorWitnessID]
	if !actor.ActorRuntimeReviewed || actor.ActorBoundaryRows < 4 {
		return fmt.Errorf("security review gate v1: actor runtime witness incomplete")
	}
	db := byWitness[p24SecurityReviewDBWitnessID]
	if !db.DBProtocolReviewed || db.ProductionPostgresRows < 8 {
		return fmt.Errorf("security review gate v1: DB protocol witness incomplete")
	}
	eco := byWitness[p24SecurityReviewEcoWitnessID]
	if !eco.PackageEcoSystemReviewed || eco.EcoValidatorPaths < 5 {
		return fmt.Errorf("security review gate v1: package/Eco witness incomplete")
	}
	build := byWitness[p24SecurityReviewBuildWitnessID]
	if !build.BuildScriptsReviewed || build.ReleaseSecurityScripts < 3 {
		return fmt.Errorf("security review gate v1: build scripts witness incomplete")
	}
	supply := byWitness[p24SecurityReviewSupplyWitnessID]
	if !supply.SupplyChainReviewed || supply.SupplyChainEvidencePaths < 5 {
		return fmt.Errorf("security review gate v1: supply chain witness incomplete")
	}
	artifacts := byWitness[p24SecurityReviewArtifactsWitnessID]
	if !artifacts.SecurityReviewArtifactPresent || !artifacts.ThreatModelArtifactPresent ||
		!artifacts.UnsafeSurfaceMapPresent ||
		!artifacts.CapabilitySurfaceMapPresent {
		return fmt.Errorf("security review gate v1: security review artifacts witness incomplete")
	}
	return nil
}

func p24SecurityReviewValidateArtifacts(report SecurityReviewGateV1Report) error {
	if !report.SecurityReviewArtifactPresent {
		return fmt.Errorf(
			"security review gate v1: docs/audits/security/security-review.md artifact missing",
		)
	}
	if !report.ThreatModelArtifactPresent {
		return fmt.Errorf(
			"security review gate v1: docs/audits/security/threat-model.md artifact missing",
		)
	}
	if !report.UnsafeSurfaceMapPresent {
		return fmt.Errorf(
			"security review gate v1: docs/audits/security/unsafe-surface-map.md artifact missing",
		)
	}
	if !report.CapabilitySurfaceMapPresent {
		return fmt.Errorf(
			"security review gate v1: docs/audits/security/capability-surface-map.md artifact missing",
		)
	}
	present := map[string]bool{}
	for _, artifact := range report.Artifacts {
		if strings.TrimSpace(artifact.Kind) == "" || strings.TrimSpace(artifact.Path) == "" {
			return fmt.Errorf("security review gate v1: artifact missing kind or path")
		}
		present[artifact.Path] = artifact.Present
	}
	for _, path := range []string{
		"docs/audits/security/security-review.md",
		"docs/audits/security/threat-model.md",
		"docs/audits/security/unsafe-surface-map.md",
		"docs/audits/security/capability-surface-map.md",
	} {
		if !present[path] {
			return fmt.Errorf("security review gate v1: required artifact %s missing", path)
		}
	}
	return nil
}

func p24SecurityReviewGateV1IDs() []SecurityReviewGateV1ID {
	return []SecurityReviewGateV1ID{
		SecurityReviewUnsafeAPISurface,
		SecurityReviewCapabilitySurface,
		SecurityReviewMemoryAllocator,
		SecurityReviewNetworkRuntime,
		SecurityReviewActorRuntime,
		SecurityReviewDBProtocol,
		SecurityReviewPackageEcoSystem,
		SecurityReviewBuildScripts,
		SecurityReviewSupplyChain,
		SecurityReviewArtifactSet,
	}
}

func p24SecurityReviewGateRow(
	id SecurityReviewGateV1ID,
	name, status string,
	evidence, tests, boundaries, witnessIDs []string,
) SecurityReviewGateV1Row {
	return SecurityReviewGateV1Row{
		ID:         id,
		Name:       name,
		Status:     status,
		Evidence:   evidence,
		Tests:      tests,
		Boundaries: boundaries,
		WitnessIDs: witnessIDs,
	}
}

func p24SecurityReviewArtifacts() []SecurityReviewArtifact {
	return []SecurityReviewArtifact{
		p24SecurityReviewArtifact("security_review", "docs/audits/security/security-review.md"),
		p24SecurityReviewArtifact("threat_model", "docs/audits/security/threat-model.md"),
		p24SecurityReviewArtifact(
			"unsafe_surface_map",
			"docs/audits/security/unsafe-surface-map.md",
		),
		p24SecurityReviewArtifact(
			"capability_surface_map",
			"docs/audits/security/capability-surface-map.md",
		),
	}
}

func p24SecurityReviewArtifact(kind string, rel string) SecurityReviewArtifact {
	_, err := os.Stat(p24RepoPath(rel))
	return SecurityReviewArtifact{
		Kind:    kind,
		Path:    rel,
		Present: err == nil,
	}
}

func p24AllRepoPathsExist(paths []string) bool {
	for _, path := range paths {
		if _, err := os.Stat(p24RepoPath(path)); err != nil {
			return false
		}
	}
	return true
}

func p24RepoPath(rel string) string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.FromSlash(rel)
	}
	return filepath.Join(filepath.Dir(filepath.Dir(file)), filepath.FromSlash(rel))
}

func p24SecurityReviewHasString(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

func p24SecurityReviewIsPlaceholder(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return lower == "" ||
		lower == "todo" ||
		lower == "tbd" ||
		strings.Contains(lower, "placeholder")
}

// ---- self_hosting_gate_v1.go ----

const (
	selfHostingGateV1Schema    = "tetra.self_hosting.gate.v1"
	selfHostingGateV1ScopeP233 = "p23.3_self_hosting_gate"

	p23SelfHostingSubsetWitnessID            = "self_host_subset_definition"
	p23SelfHostingBootstrapBlockersWitnessID = "self_host_bootstrap_blockers"
	p23SelfHostingRegisterBackendWitnessID   = "register_backend_stability"
	p23SelfHostingOptimizerWitnessID         = "optimizer_validation_maturity"
	p23SelfHostingAllocatorRuntimeWitnessID  = "allocator_runtime_stability"
	p23SelfHostingStdlibWitnessID            = "stdlib_sufficiency"
)

type SelfHostingGateV1ID string

const (
	SelfHostingGateSubsetDefinition       SelfHostingGateV1ID = "self_host_subset_definition"
	SelfHostingGateSmallComponentCompile  SelfHostingGateV1ID = "small_compiler_component_compile"
	SelfHostingGateOutputComparison       SelfHostingGateV1ID = "go_vs_tetra_output_comparison"
	SelfHostingGateRegisterBackend        SelfHostingGateV1ID = "register_backend_stability"
	SelfHostingGateOptimizerValidation    SelfHostingGateV1ID = "optimizer_validation_maturity"
	SelfHostingGateAllocatorRuntime       SelfHostingGateV1ID = "allocator_runtime_stability"
	SelfHostingGateStdlibSufficiency      SelfHostingGateV1ID = "stdlib_sufficiency"
	SelfHostingGateDeterministicBootstrap SelfHostingGateV1ID = "deterministic_bootstrap_chain"
	SelfHostingGateCrossPlatformBootstrap SelfHostingGateV1ID = "cross_platform_bootstrap_story"
	SelfHostingGateNoSelfHostingClaim     SelfHostingGateV1ID = "no_self_hosting_claim"
)

type SelfHostingGateV1Report struct {
	SchemaVersion         string                     `json:"schema_version"`
	Scope                 string                     `json:"scope"`
	Rows                  []SelfHostingGateV1Row     `json:"rows"`
	Witnesses             []SelfHostingGateV1Witness `json:"witnesses"`
	NonClaims             []string                   `json:"non_claims"`
	GateDecision          selfhostgate.Decision      `json:"gate_decision"`
	CompilerSubsetDefined bool                       `json:"compiler_subset_defined"`
	SubsetName            string                     `json:"subset_name"`

	SmallCompilerComponentCompiled bool `json:"small_compiler_component_compiled"`

	GoVsTetraOutputCompared bool `json:"go_vs_tetra_output_compared"`

	RegisterBackendEvidencePresent bool `json:"register_backend_evidence_present"`

	OptimizerValidationEvidencePresent bool `json:"optimizer_validation_evidence_present"`

	AllocatorRuntimeEvidencePresent bool `json:"allocator_runtime_evidence_present"`

	StdlibEvidencePresent bool `json:"stdlib_evidence_present"`

	DeterministicBootstrapChain bool `json:"deterministic_bootstrap_chain"`

	CrossPlatformBootstrapStory bool `json:"cross_platform_bootstrap_story"`

	SelfHostingClaimed     bool `json:"self_hosting_claimed"`
	RuntimeBehaviorChanged bool `json:"runtime_behavior_changed"`
	SafeSemanticsChanged   bool `json:"safe_semantics_changed"`
	PerformanceClaimed     bool `json:"performance_claimed"`
}

type SelfHostingGateV1Row struct {
	ID         SelfHostingGateV1ID `json:"id"`
	Name       string              `json:"name"`
	Status     string              `json:"status"`
	Evidence   []string            `json:"evidence"`
	Tests      []string            `json:"tests"`
	Boundaries []string            `json:"boundaries"`
	WitnessIDs []string            `json:"witness_ids"`
}

type SelfHostingGateV1Witness struct {
	ID                             string `json:"id"`
	Kind                           string `json:"kind"`
	CompilerSubsetDefined          bool   `json:"compiler_subset_defined,omitempty"`
	SubsetName                     string `json:"subset_name,omitempty"`
	SmallCompilerComponentCompiled bool   `json:"small_compiler_component_compiled,omitempty"`
	GoVsTetraOutputCompared        bool   `json:"go_vs_tetra_output_compared,omitempty"`
	DeterministicBootstrapChain    bool   `json:"deterministic_bootstrap_chain,omitempty"`
	CrossPlatformBootstrapStory    bool   `json:"cross_platform_bootstrap_story,omitempty"`
	RegisterBackendEvidencePresent bool   `json:"register_backend_evidence_present,omitempty"`
	BackendMatrixLanes             int    `json:"backend_matrix_lanes,omitempty"`

	OptimizerValidationEvidencePresent bool `json:"optimizer_validation_evidence_present,omitempty"`

	TranslationValidationRows       int      `json:"translation_validation_rows,omitempty"`
	TranslationValidationWitnesses  int      `json:"translation_validation_witnesses,omitempty"`
	AllocatorRuntimeEvidencePresent bool     `json:"allocator_runtime_evidence_present,omitempty"`
	RuntimeAllocationContracts      int      `json:"runtime_allocation_contracts,omitempty"`
	RegionAllocatorAlignmentBytes   int32    `json:"region_allocator_alignment_bytes,omitempty"`
	PerCoreSmallHeapEvidencePresent bool     `json:"per_core_small_heap_evidence_present,omitempty"`
	StdlibEvidencePresent           bool     `json:"stdlib_evidence_present,omitempty"`
	StdlibRows                      int      `json:"stdlib_rows,omitempty"`
	Blockers                        []string `json:"blockers,omitempty"`
}

func BuildP23SelfHostingGateV1Report() (SelfHostingGateV1Report, error) {
	subset := buildP23SelfHostingSubsetWitness()
	blockers := buildP23SelfHostingBootstrapBlockersWitness()
	backend, err := buildP23SelfHostingRegisterBackendWitness()
	if err != nil {
		return SelfHostingGateV1Report{}, err
	}
	optimizer, err := buildP23SelfHostingOptimizerWitness()
	if err != nil {
		return SelfHostingGateV1Report{}, err
	}
	allocator, err := buildP23SelfHostingAllocatorRuntimeWitness()
	if err != nil {
		return SelfHostingGateV1Report{}, err
	}
	stdlib, err := buildP23SelfHostingStdlibWitness()
	if err != nil {
		return SelfHostingGateV1Report{}, err
	}

	decision := selfhostgate.Evaluate(selfhostgate.Evidence{
		CompilerSubsetDefined:       subset.CompilerSubsetDefined,
		RegisterBackendStable:       backend.RegisterBackendEvidencePresent,
		OptimizerValidated:          optimizer.OptimizerValidationEvidencePresent,
		AllocatorStable:             allocator.AllocatorRuntimeEvidencePresent,
		StdlibStrongEnough:          stdlib.StdlibEvidencePresent,
		SmallCompilerComponentBuilt: blockers.SmallCompilerComponentCompiled,
		GoVsTetraOutputCompared:     blockers.GoVsTetraOutputCompared,
		DeterministicBootstrapChain: blockers.DeterministicBootstrapChain,
		CrossPlatformBootstrapStory: blockers.CrossPlatformBootstrapStory,
	})

	report := SelfHostingGateV1Report{
		SchemaVersion: selfHostingGateV1Schema,
		Scope:         selfHostingGateV1ScopeP233,
		Witnesses: []SelfHostingGateV1Witness{
			subset,
			blockers,
			backend,
			optimizer,
			allocator,
			stdlib,
		},
		Rows: []SelfHostingGateV1Row{
			p23SelfHostingGateRow(
				SelfHostingGateSubsetDefinition,
				"Self-host subset definition",
				"defined_gate_subset",
				[]string{
					("P23.3 defines a verified subset gate for evidence-bearing " +
						"compiler slices; this is not self-hosting and not a claim " +
						"that Tetra compiles its compiler."),
					("The subset is limited to parser/checker/PLIR/lowering " +
						"witnesses, scalar i32 backend witnesses, optimizer " +
						"validation, allocator/runtime contracts, and region-aware " +
						"stdlib evidence."),
				},
				[]string{
					"go test ./compiler -run 'P23SelfHostingGate'",
					"go test ./compiler/internal/selfhostgate -run 'SelfHosting'",
				},
				[]string{
					("verified subset is an evidence gate, not a Tetra-authored " +
						"compiler subset that compiles itself"),
					"no full compiler source migration to Tetra is claimed",
				},
				[]string{p23SelfHostingSubsetWitnessID},
			),
			p23SelfHostingGateRow(
				SelfHostingGateSmallComponentCompile,
				"Small compiler component compile boundary",
				"blocked_missing_bootstrap_evidence",
				[]string{
					("No small compiler component is currently claimed to compile " +
						"as Tetra-authored compiler source."),
					("The small compiler component compile task remains blocked " +
						"until a real Tetra compiler component source and " +
						"deterministic build artifact exist."),
				},
				[]string{
					"go test ./compiler -run 'P23SelfHostingGate'",
				},
				[]string{
					"blocked rather than treated as Go compiler evidence",
					"Go implementation tests do not count as a Tetra small compiler component compile",
				},
				[]string{p23SelfHostingBootstrapBlockersWitnessID},
			),
			p23SelfHostingGateRow(
				SelfHostingGateOutputComparison,
				"Go output vs Tetra-compiled output comparison boundary",
				"blocked_missing_bootstrap_evidence",
				[]string{
					"No Go compiler output vs Tetra-compiled output comparison is claimed yet.",
					("The comparison row remains blocked until both the current " +
						"Go compiler output and Tetra-compiled output for the same " +
						"compiler subset are produced and compared deterministically."),
				},
				[]string{
					"go test ./compiler -run 'P23SelfHostingGate'",
				},
				[]string{
					"blocked until same-input Go compiler output and Tetra-compiled output artifacts exist",
					"no output equivalence, byte equivalence, or semantic equivalence claim is made",
				},
				[]string{p23SelfHostingBootstrapBlockersWitnessID},
			),
			p23SelfHostingGateRow(
				SelfHostingGateRegisterBackend,
				"Register backend stability gate",
				"current_evidence_present",
				[]string{
					("differential.CheckBackendMatrix covers source, Stack IR, " +
						"optimized Stack IR, SSA, and Machine IR lanes for the " +
						"register backend stability witness."),
					("Machine IR evidence is current internal backend evidence " +
						"and does not make the register backend a public self-host " +
						"backend selector."),
				},
				[]string{
					"go test ./compiler/internal/differential -run 'CheckBackendMatrix' -count=1",
					"go test ./compiler -run 'P23SelfHostingGate'",
				},
				[]string{
					"register backend stability is current supported-subset evidence only",
					"broader compiler self-hosting remains blocked by bootstrap evidence",
				},
				[]string{p23SelfHostingRegisterBackendWitnessID},
			),
			p23SelfHostingGateRow(
				SelfHostingGateOptimizerValidation,
				"Optimizer validation maturity gate",
				"current_evidence_present",
				[]string{
					("P23.0 translation validation v2 records optimizer " +
						"validation maturity through registered pass coverage, " +
						"symbolic scalar equivalence, memory equivalence, proof " +
						"preservation, allocation preservation, and sha256 " +
						"before/after metadata."),
					("BuildP23TranslationValidationV2 and " +
						"ValidateP23TranslationValidationV2 are reused as the live " +
						"optimizer gate witness."),
				},
				[]string{
					"go test ./compiler -run 'P23TranslationValidationV2|P23SelfHostingGate' -count=1",
				},
				[]string{
					("translation validation v2 is supported-subset evidence, not " +
						"exhaustive optimizer completeness"),
					"optimizer maturity alone does not allow self-hosting",
				},
				[]string{p23SelfHostingOptimizerWitnessID},
			),
			p23SelfHostingGateRow(
				SelfHostingGateAllocatorRuntime,
				"Allocator/runtime stability gate",
				"current_evidence_present",
				[]string{
					("runtimeabi.RuntimeAllocationContracts validates allocation " +
						"APIs, guard behavior, failure behavior, debug " +
						"instrumentation, and report hooks."),
					("runtimeabi.RuntimeRegionAllocatorConfig, AlignRegionBytes, " +
						"and RuntimePerCoreSmallHeapABI provide allocator/runtime " +
						"stability evidence for the current gate."),
				},
				[]string{
					"go test ./compiler/internal/runtimeabi -run 'Allocation|Region|SmallHeap' -count=1",
					"go test ./compiler -run 'P23SelfHostingGate'",
				},
				[]string{
					("allocator/runtime evidence is internal runtime ABI evidence," +
						" not a complete self-host runtime"),
					"cross-platform bootstrap and Tetra compiler component evidence remain blocked",
				},
				[]string{p23SelfHostingAllocatorRuntimeWitnessID},
			),
			p23SelfHostingGateRow(
				SelfHostingGateStdlibSufficiency,
				"Stdlib sufficiency gate",
				"current_evidence_present",
				[]string{
					("stdlibrt.RegionAwareStdlibCoverage and " +
						"ValidateRegionAwareStdlibCoverage record current " +
						"region-aware stdlib evidence for StringBuilder, VecBytes, " +
						"HashMapBytes, buffers, borrowed JSON/HTTP views, PostgreSQL " +
						"helpers, and production boundaries."),
					("The stdlib witness is sufficient for this gate evidence " +
						"layer but not sufficient for a full self-hosting claim."),
				},
				[]string{
					"go test ./compiler/internal/stdlibrt -run 'RegionAwareStdlibCoverage' -count=1",
					"go test ./compiler -run 'P23SelfHostingGate'",
				},
				[]string{
					"stdlib sufficiency is evidence for the current gate only",
					"full compiler stdlib needs and cross-platform bootstrap remain unpromoted",
				},
				[]string{p23SelfHostingStdlibWitnessID},
			),
			p23SelfHostingGateRow(
				SelfHostingGateDeterministicBootstrap,
				"Deterministic bootstrap chain gate",
				"blocked_missing_bootstrap_evidence",
				[]string{
					"No deterministic bootstrap chain is claimed yet.",
					("The bootstrap chain remains blocked until a staged " +
						"Go-to-Tetra-to-Tetra compiler build emits deterministic " +
						"artifacts with stable hashes."),
				},
				[]string{
					"go test ./compiler -run 'P23SelfHostingGate'",
				},
				[]string{
					"blocked until staged bootstrap artifacts and hashes exist",
					"scripts/dev/bootstrap.sh refreshes Go-built binaries and does not count as a self-host chain",
				},
				[]string{p23SelfHostingBootstrapBlockersWitnessID},
			),
			p23SelfHostingGateRow(
				SelfHostingGateCrossPlatformBootstrap,
				"Cross-platform bootstrap story gate",
				"blocked_missing_bootstrap_evidence",
				[]string{
					"No cross-platform bootstrap story is claimed yet.",
					("The cross-platform bootstrap row remains blocked until " +
						"Linux, macOS, Windows, and build-only target bootstrap " +
						"evidence has matching artifacts and no host fallback."),
				},
				[]string{
					"go test ./compiler -run 'P23SelfHostingGate'",
				},
				[]string{
					"blocked until platform-specific bootstrap evidence exists",
					"current native target evidence is not a cross-platform self-host bootstrap story",
				},
				[]string{p23SelfHostingBootstrapBlockersWitnessID},
			),
			p23SelfHostingGateRow(
				SelfHostingGateNoSelfHostingClaim,
				"No self-hosting claim",
				"blocked_no_claim",
				[]string{
					("SelfHostingClaimed=false and GateDecision.Allowed=false are " +
						"required for the current P23.3 report."),
					("selfhostgate.Evaluate records missing small compiler " +
						"component, Go-vs-Tetra output comparison, deterministic " +
						"bootstrap chain, and cross-platform bootstrap story " +
						"evidence."),
				},
				[]string{
					"go test ./compiler/internal/selfhostgate -run 'SelfHosting' -count=1",
					"go test ./compiler -run 'P23SelfHostingGate'",
				},
				[]string{
					"no self-hosting claim is made",
					("future self-hosting promotion must replace blocker rows " +
						"with real evidence and keep GateDecision honest"),
				},
				[]string{p23SelfHostingBootstrapBlockersWitnessID},
			),
		},
		NonClaims: []string{
			"Tetra is not self-hosting",
			"no Tetra compiler component is claimed to compile itself yet",
			"no Go compiler output vs Tetra-compiled output equivalence is claimed yet",
			"no deterministic bootstrap chain is claimed yet",
			"no cross-platform bootstrap story is claimed yet",
			"runtime behavior does not change",
			"safe-program semantics do not change",
			"no performance claim is made",
		},
		GateDecision:                       decision,
		CompilerSubsetDefined:              subset.CompilerSubsetDefined,
		SubsetName:                         subset.SubsetName,
		SmallCompilerComponentCompiled:     blockers.SmallCompilerComponentCompiled,
		GoVsTetraOutputCompared:            blockers.GoVsTetraOutputCompared,
		RegisterBackendEvidencePresent:     backend.RegisterBackendEvidencePresent,
		OptimizerValidationEvidencePresent: optimizer.OptimizerValidationEvidencePresent,
		AllocatorRuntimeEvidencePresent:    allocator.AllocatorRuntimeEvidencePresent,
		StdlibEvidencePresent:              stdlib.StdlibEvidencePresent,
		DeterministicBootstrapChain:        blockers.DeterministicBootstrapChain,
		CrossPlatformBootstrapStory:        blockers.CrossPlatformBootstrapStory,
		SelfHostingClaimed:                 false,
		RuntimeBehaviorChanged:             false,
		SafeSemanticsChanged:               false,
		PerformanceClaimed:                 false,
	}
	if err := ValidateP23SelfHostingGateV1Report(report); err != nil {
		return SelfHostingGateV1Report{}, err
	}
	return report, nil
}

func ValidateP23SelfHostingGateV1Report(report SelfHostingGateV1Report) error {
	if report.SchemaVersion != selfHostingGateV1Schema {
		return fmt.Errorf("self-hosting gate v1: schema_version is %q", report.SchemaVersion)
	}
	if report.Scope != selfHostingGateV1ScopeP233 {
		return fmt.Errorf("self-hosting gate v1: scope is %q", report.Scope)
	}
	if report.SelfHostingClaimed {
		return fmt.Errorf("self-hosting gate v1: self-hosting claim is forbidden")
	}
	if report.RuntimeBehaviorChanged {
		return fmt.Errorf("self-hosting gate v1: runtime behavior change claim is forbidden")
	}
	if report.SafeSemanticsChanged {
		return fmt.Errorf("self-hosting gate v1: safe semantics change claim is forbidden")
	}
	if report.PerformanceClaimed {
		return fmt.Errorf("self-hosting gate v1: performance claim is forbidden")
	}
	if !report.CompilerSubsetDefined || strings.TrimSpace(report.SubsetName) == "" {
		return fmt.Errorf("self-hosting gate v1: compiler subset evidence missing")
	}
	if !report.RegisterBackendEvidencePresent {
		return fmt.Errorf("self-hosting gate v1: register backend evidence missing")
	}
	if !report.OptimizerValidationEvidencePresent {
		return fmt.Errorf("self-hosting gate v1: optimizer validation evidence missing")
	}
	if !report.AllocatorRuntimeEvidencePresent {
		return fmt.Errorf("self-hosting gate v1: allocator/runtime evidence missing")
	}
	if !report.StdlibEvidencePresent {
		return fmt.Errorf("self-hosting gate v1: stdlib evidence missing")
	}
	if report.SmallCompilerComponentCompiled {
		return fmt.Errorf(
			("self-hosting gate v1: small compiler component compile " +
				"claim is forbidden without Tetra component evidence"),
		)
	}
	if report.GoVsTetraOutputCompared {
		return fmt.Errorf(
			"self-hosting gate v1: output comparison claim is forbidden without Go and Tetra artifacts",
		)
	}
	if report.DeterministicBootstrapChain {
		return fmt.Errorf(
			"self-hosting gate v1: deterministic bootstrap claim is forbidden without staged hashes",
		)
	}
	if report.CrossPlatformBootstrapStory {
		return fmt.Errorf(
			"self-hosting gate v1: cross-platform bootstrap claim is forbidden without platform artifacts",
		)
	}
	if err := p23SelfHostingValidateDecision(report.GateDecision); err != nil {
		return err
	}
	for _, want := range []string{
		"Tetra is not self-hosting",
		"no Tetra compiler component is claimed to compile itself yet",
		"no Go compiler output vs Tetra-compiled output equivalence is claimed yet",
		"no deterministic bootstrap chain is claimed yet",
		"no cross-platform bootstrap story is claimed yet",
		"runtime behavior does not change",
		"safe-program semantics do not change",
		"no performance claim is made",
	} {
		if !p23SelfHostingGateHasString(report.NonClaims, want) {
			return fmt.Errorf("self-hosting gate v1: missing non-claim %q", want)
		}
	}
	if err := p23SelfHostingGateValidateRowsAndWitnesses(report.Rows, report.Witnesses); err != nil {
		return err
	}
	return nil
}

func buildP23SelfHostingSubsetWitness() SelfHostingGateV1Witness {
	return SelfHostingGateV1Witness{
		ID:                    p23SelfHostingSubsetWitnessID,
		Kind:                  "self_host_subset_definition",
		CompilerSubsetDefined: true,
		SubsetName:            "p23.3_verified_subset_gate_not_self_hosted",
	}
}

func buildP23SelfHostingBootstrapBlockersWitness() SelfHostingGateV1Witness {
	return SelfHostingGateV1Witness{
		ID:                             p23SelfHostingBootstrapBlockersWitnessID,
		Kind:                           "self_host_bootstrap_blockers",
		SmallCompilerComponentCompiled: false,
		GoVsTetraOutputCompared:        false,
		DeterministicBootstrapChain:    false,
		CrossPlatformBootstrapStory:    false,
		Blockers: []string{
			"small_compiler_component_compiled",
			"go_vs_tetra_output_compared",
			"deterministic_bootstrap_chain",
			"cross_platform_bootstrap_story",
		},
	}
}

func buildP23SelfHostingRegisterBackendWitness() (SelfHostingGateV1Witness, error) {
	matrix, err := differential.CheckBackendMatrix(differential.BackendMatrixCase{
		Name:      "p23.3-self-host-register-backend-loop",
		Functions: []ir.IRFunc{p23LoopSumFunc()},
		Entry:     "sum_n",
		Samples: []differential.MatrixSample{
			{Name: "zero", Args: []int32{0}},
			{Name: "seven", Args: []int32{7}},
		},
		Source: func(sample differential.MatrixSample) (int32, bool) {
			n := sample.Args[0]
			var total int32
			for i := int32(0); i < n; i++ {
				total += i
			}
			return total, true
		},
	})
	if err != nil {
		return SelfHostingGateV1Witness{}, err
	}
	return SelfHostingGateV1Witness{
		ID:                             p23SelfHostingRegisterBackendWitnessID,
		Kind:                           "register_backend_stability",
		RegisterBackendEvidencePresent: matrix.HasLane(differential.LaneMachineIRInterpreter),
		BackendMatrixLanes:             len(matrix.Lanes),
	}, nil
}

func buildP23SelfHostingOptimizerWitness() (SelfHostingGateV1Witness, error) {
	report, err := BuildP23TranslationValidationV2()
	if err != nil {
		return SelfHostingGateV1Witness{}, err
	}
	if err := ValidateP23TranslationValidationV2(report); err != nil {
		return SelfHostingGateV1Witness{}, err
	}
	return SelfHostingGateV1Witness{
		ID:   p23SelfHostingOptimizerWitnessID,
		Kind: "optimizer_validation_maturity",
		OptimizerValidationEvidencePresent: report.RegisteredPassCoverageComplete &&
			report.SymbolicScalarEquivalenceSamples > 0 &&
			report.MemoryEquivalenceSamples > 0 &&
			report.BoundsProofsPreserved &&
			report.AllocationPlanValidated &&
			report.BeforeAfterHashesMachineCheckable,
		TranslationValidationRows:      len(report.Rows),
		TranslationValidationWitnesses: len(report.Witnesses),
	}, nil
}

func buildP23SelfHostingAllocatorRuntimeWitness() (SelfHostingGateV1Witness, error) {
	contracts := runtimeabi.RuntimeAllocationContracts()
	for _, contract := range contracts {
		if err := runtimeabi.ValidateRuntimeAllocationContract(contract); err != nil {
			return SelfHostingGateV1Witness{}, err
		}
	}
	region := runtimeabi.RuntimeRegionAllocatorConfig(false)
	aligned, alignedOK := runtimeabi.AlignRegionBytes(33)
	_, invalidRejected := runtimeabi.AlignRegionBytes(-1)
	allocator, err := runtimeabi.NewPerCoreSmallHeapAllocator(
		runtimeabi.RuntimePerCoreSmallHeapABI(2),
	)
	if err != nil {
		return SelfHostingGateV1Witness{}, err
	}
	handle, err := allocator.Alloc(0, 32)
	if err != nil {
		return SelfHostingGateV1Witness{}, err
	}
	if err := allocator.Free(handle); err != nil {
		return SelfHostingGateV1Witness{}, err
	}
	if _, err := allocator.Alloc(0, 32); err != nil {
		return SelfHostingGateV1Witness{}, err
	}
	smallHeap := allocator.Report()
	return SelfHostingGateV1Witness{
		ID:   p23SelfHostingAllocatorRuntimeWitnessID,
		Kind: "allocator_runtime_stability",
		AllocatorRuntimeEvidencePresent: len(contracts) >= 5 &&
			region.AlignmentBytes == runtimeabi.RegionAllocatorAlignmentBytes &&
			alignedOK &&
			aligned == 48 &&
			!invalidRejected &&
			smallHeap.TotalReuses > 0 &&
			!smallHeap.EstimatedMmapPerAllocation,
		RuntimeAllocationContracts:    len(contracts),
		RegionAllocatorAlignmentBytes: region.AlignmentBytes,
		PerCoreSmallHeapEvidencePresent: smallHeap.TotalReuses > 0 &&
			!smallHeap.EstimatedMmapPerAllocation,
	}, nil
}

func buildP23SelfHostingStdlibWitness() (SelfHostingGateV1Witness, error) {
	report, err := stdlibrt.RegionAwareStdlibCoverage()
	if err != nil {
		return SelfHostingGateV1Witness{}, err
	}
	if err := stdlibrt.ValidateRegionAwareStdlibCoverage(report); err != nil {
		return SelfHostingGateV1Witness{}, err
	}
	return SelfHostingGateV1Witness{
		ID:                    p23SelfHostingStdlibWitnessID,
		Kind:                  "stdlib_sufficiency",
		StdlibEvidencePresent: len(report.Rows) >= 10,
		StdlibRows:            len(report.Rows),
	}, nil
}

func p23SelfHostingValidateDecision(decision selfhostgate.Decision) error {
	if decision.Allowed {
		return fmt.Errorf("self-hosting gate v1: gate decision unexpectedly allowed self-hosting")
	}
	if !strings.Contains(decision.Reason, "blocked") {
		return fmt.Errorf("self-hosting gate v1: gate decision reason must remain blocked")
	}
	for _, missing := range []string{
		"small_compiler_component_compiled",
		"go_vs_tetra_output_compared",
		"deterministic_bootstrap_chain",
		"cross_platform_bootstrap_story",
	} {
		if !decision.Missing(missing) {
			return fmt.Errorf("self-hosting gate v1: gate decision missing blocker %s", missing)
		}
	}
	return nil
}

func p23SelfHostingGateValidateRowsAndWitnesses(
	rows []SelfHostingGateV1Row,
	witnesses []SelfHostingGateV1Witness,
) error {
	byWitness := map[string]SelfHostingGateV1Witness{}
	for _, witness := range witnesses {
		if strings.TrimSpace(witness.ID) == "" || strings.TrimSpace(witness.Kind) == "" {
			return fmt.Errorf("self-hosting gate v1: witness missing id or kind")
		}
		if _, exists := byWitness[witness.ID]; exists {
			return fmt.Errorf("self-hosting gate v1: duplicate witness %q", witness.ID)
		}
		byWitness[witness.ID] = witness
	}
	expected := map[SelfHostingGateV1ID]bool{}
	for _, id := range p23SelfHostingGateV1IDs() {
		expected[id] = true
	}
	seen := map[SelfHostingGateV1ID]bool{}
	for _, row := range rows {
		if !expected[row.ID] {
			return fmt.Errorf("self-hosting gate v1: unexpected row %q", row.ID)
		}
		if seen[row.ID] {
			return fmt.Errorf("self-hosting gate v1: duplicate row %q", row.ID)
		}
		seen[row.ID] = true
		if strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Status) == "" {
			return fmt.Errorf("self-hosting gate v1: row %q missing name or status", row.ID)
		}
		if len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 ||
			len(row.WitnessIDs) == 0 {
			return fmt.Errorf(
				"self-hosting gate v1: row %q missing evidence, tests, boundaries, or witness ids",
				row.ID,
			)
		}
		for _, text := range append(append([]string{}, row.Evidence...), row.Boundaries...) {
			if p23SelfHostingGateIsPlaceholder(text) {
				return fmt.Errorf("self-hosting gate v1: row %q has placeholder evidence", row.ID)
			}
		}
		for _, id := range row.WitnessIDs {
			if _, ok := byWitness[id]; !ok {
				return fmt.Errorf(
					"self-hosting gate v1: row %q references missing witness %q",
					row.ID,
					id,
				)
			}
		}
	}
	for _, id := range p23SelfHostingGateV1IDs() {
		if !seen[id] {
			return fmt.Errorf("self-hosting gate v1: missing row %q", id)
		}
	}
	subset := byWitness[p23SelfHostingSubsetWitnessID]
	if !subset.CompilerSubsetDefined || !strings.Contains(subset.SubsetName, "verified") {
		return fmt.Errorf("self-hosting gate v1: compiler subset witness incomplete")
	}
	backend := byWitness[p23SelfHostingRegisterBackendWitnessID]
	if !backend.RegisterBackendEvidencePresent || backend.BackendMatrixLanes < 5 {
		return fmt.Errorf("self-hosting gate v1: register backend witness incomplete")
	}
	optimizer := byWitness[p23SelfHostingOptimizerWitnessID]
	if !optimizer.OptimizerValidationEvidencePresent || optimizer.TranslationValidationRows < 6 {
		return fmt.Errorf("self-hosting gate v1: optimizer witness incomplete")
	}
	allocator := byWitness[p23SelfHostingAllocatorRuntimeWitnessID]
	if !allocator.AllocatorRuntimeEvidencePresent || allocator.RuntimeAllocationContracts < 5 ||
		!allocator.PerCoreSmallHeapEvidencePresent {
		return fmt.Errorf("self-hosting gate v1: allocator/runtime witness incomplete")
	}
	stdlib := byWitness[p23SelfHostingStdlibWitnessID]
	if !stdlib.StdlibEvidencePresent || stdlib.StdlibRows < 10 {
		return fmt.Errorf("self-hosting gate v1: stdlib witness incomplete")
	}
	blockers := byWitness[p23SelfHostingBootstrapBlockersWitnessID]
	if blockers.SmallCompilerComponentCompiled || blockers.GoVsTetraOutputCompared ||
		blockers.DeterministicBootstrapChain ||
		blockers.CrossPlatformBootstrapStory {
		return fmt.Errorf(
			"self-hosting gate v1: bootstrap blockers witness claims unavailable evidence",
		)
	}
	return nil
}

func p23SelfHostingGateV1IDs() []SelfHostingGateV1ID {
	return []SelfHostingGateV1ID{
		SelfHostingGateSubsetDefinition,
		SelfHostingGateSmallComponentCompile,
		SelfHostingGateOutputComparison,
		SelfHostingGateRegisterBackend,
		SelfHostingGateOptimizerValidation,
		SelfHostingGateAllocatorRuntime,
		SelfHostingGateStdlibSufficiency,
		SelfHostingGateDeterministicBootstrap,
		SelfHostingGateCrossPlatformBootstrap,
		SelfHostingGateNoSelfHostingClaim,
	}
}

func p23SelfHostingGateRow(
	id SelfHostingGateV1ID,
	name, status string,
	evidence, tests, boundaries, witnessIDs []string,
) SelfHostingGateV1Row {
	return SelfHostingGateV1Row{
		ID:         id,
		Name:       name,
		Status:     status,
		Evidence:   evidence,
		Tests:      tests,
		Boundaries: boundaries,
		WitnessIDs: witnessIDs,
	}
}

func p23SelfHostingGateHasString(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

func p23SelfHostingGateIsPlaceholder(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return lower == "" ||
		lower == "todo" ||
		lower == "tbd" ||
		strings.Contains(lower, "placeholder")
}

// ---- translation_validation_v2.go ----

const (
	translationValidationV2Schema    = "tetra.translation.validation.v2"
	translationValidationV2ScopeP230 = "p23.0_translation_validation_v2"

	p23TranslationRegisteredPassesWitnessID = "registered_optimizer_passes"
	p23TranslationScalarWitnessID           = "symbolic_scalar_arithmetic"
	p23TranslationMemoryWitnessID           = "memory_i32_slice_equivalence"
	p23TranslationLoopWitnessID             = "loop_equivalence_samples"
	p23TranslationCallInliningWitnessID     = "call_inlining_equivalence"
	p23TranslationProofWitnessID            = "bounds_proof_preservation"
	p23TranslationAllocationWitnessID       = "allocation_plan_preservation"
	p23TranslationHashWitnessID             = "before_after_hash_metadata"
)

type TranslationValidationV2ID string
type translationValidationID = TranslationValidationV2ID

const (
	TranslationValidationV2RegisteredPasses translationValidationID = "registered_passes"
	TranslationValidationV2SymbolicScalar                           = translationValidationID(
		"symbolic_scalar_equivalence",
	)
	TranslationValidationV2MemoryEquivalence       translationValidationID = "memory_equivalence"
	TranslationValidationV2BoundsProofPreservation                         = translationValidationID(
		"bounds_proof_preservation",
	)
	TranslationValidationV2AllocationPlanPreservation = translationValidationID(
		"allocation_plan_preservation",
	)
	TranslationValidationV2MachineCheckableHashes = translationValidationID(
		"machine_checkable_hashes",
	)
)

type TranslationValidationV2Report struct {
	SchemaVersion string                           `json:"schema_version"`
	Scope         string                           `json:"scope"`
	Rows          []TranslationValidationV2Row     `json:"rows"`
	Witnesses     []TranslationValidationV2Witness `json:"witnesses"`
	NonClaims     []string                         `json:"non_claims"`

	RegisteredPassCoverageComplete bool `json:"registered_pass_coverage_complete"`

	SymbolicScalarEquivalenceSamples int `json:"symbolic_scalar_equivalence_samples"`

	MemoryEquivalenceSamples int `json:"memory_equivalence_samples"`

	LoopEquivalenceSamples int `json:"loop_equivalence_samples"`

	CallEquivalenceSamples int `json:"call_equivalence_samples"`

	BoundsProofsPreserved bool `json:"bounds_proofs_preserved"`

	AllocationPlanValidated bool `json:"allocation_plan_validated"`

	BeforeAfterHashesMachineCheckable bool `json:"before_after_hashes_machine_checkable"`

	FullFormalProofClaimed bool `json:"full_formal_proof_claimed"`

	ExhaustiveOptimizerCompletenessClaimed bool `json:"exhaustive_optimizer_completeness_claimed"`

	BroadMemoryModelClaimed bool `json:"broad_memory_model_claimed"`

	BroadLoopTheoremProverClaimed bool `json:"broad_loop_theorem_prover_claimed"`

	PerformanceClaimed bool `json:"performance_claimed"`

	RuntimeBehaviorChanged bool `json:"runtime_behavior_changed"`

	SafeSemanticsChanged bool `json:"safe_semantics_changed"`
}

type TranslationValidationV2Row struct {
	ID         TranslationValidationV2ID `json:"id"`
	Name       string                    `json:"name"`
	Status     string                    `json:"status"`
	Evidence   []string                  `json:"evidence"`
	Tests      []string                  `json:"tests"`
	Boundaries []string                  `json:"boundaries"`
	WitnessIDs []string                  `json:"witness_ids"`
}

type TranslationValidationV2Witness struct {
	ID                             string `json:"id"`
	Kind                           string `json:"kind"`
	RegisteredPasses               int    `json:"registered_passes,omitempty"`
	RegisteredPassCoverageComplete bool   `json:"registered_pass_coverage_complete,omitempty"`
	TranslationMetadataPresent     bool   `json:"translation_metadata_present,omitempty"`
	SymbolicScalarChecks           int    `json:"symbolic_scalar_checks,omitempty"`
	DifferentialSamples            int    `json:"differential_samples,omitempty"`
	SemanticMismatchRejected       bool   `json:"semantic_mismatch_rejected,omitempty"`
	MemoryEquivalenceSamples       int    `json:"memory_equivalence_samples,omitempty"`
	MemoryMismatchRejected         bool   `json:"memory_mismatch_rejected,omitempty"`
	LoopEquivalenceSamples         int    `json:"loop_equivalence_samples,omitempty"`
	DifferentialLanes              int    `json:"differential_lanes,omitempty"`
	CallEquivalenceSamples         int    `json:"call_equivalence_samples,omitempty"`
	BeforeHadCall                  bool   `json:"before_had_call,omitempty"`
	AfterHadCall                   bool   `json:"after_had_call,omitempty"`
	TranslationValidated           bool   `json:"translation_validated,omitempty"`
	ProofFactsCompared             int    `json:"proof_facts_compared,omitempty"`
	BoundsProofsPreserved          bool   `json:"bounds_proofs_preserved,omitempty"`
	MissingProofRejected           bool   `json:"missing_proof_rejected,omitempty"`
	AllocationPlanValidated        bool   `json:"allocation_plan_validated,omitempty"`
	AllocationDriftRejected        bool   `json:"allocation_drift_rejected,omitempty"`
	BeforeHash                     string `json:"before_hash,omitempty"`
	AfterHash                      string `json:"after_hash,omitempty"`
	HashesMachineCheckable         bool   `json:"hashes_machine_checkable,omitempty"`
	HashesDistinct                 bool   `json:"hashes_distinct,omitempty"`
}

func BuildP23TranslationValidationV2() (TranslationValidationV2Report, error) {
	registered, err := buildP23RegisteredPassesWitness()
	if err != nil {
		return TranslationValidationV2Report{}, err
	}
	scalar, err := buildP23ScalarWitness()
	if err != nil {
		return TranslationValidationV2Report{}, err
	}
	memory, err := buildP23MemoryWitness()
	if err != nil {
		return TranslationValidationV2Report{}, err
	}
	loop, err := buildP23LoopWitness()
	if err != nil {
		return TranslationValidationV2Report{}, err
	}
	call, err := buildP23CallInliningWitness()
	if err != nil {
		return TranslationValidationV2Report{}, err
	}
	proof, err := buildP23ProofWitness()
	if err != nil {
		return TranslationValidationV2Report{}, err
	}
	allocation, err := buildP23AllocationWitness()
	if err != nil {
		return TranslationValidationV2Report{}, err
	}
	hash, err := buildP23HashWitness()
	if err != nil {
		return TranslationValidationV2Report{}, err
	}

	report := TranslationValidationV2Report{
		SchemaVersion: translationValidationV2Schema,
		Scope:         translationValidationV2ScopeP230,
		Witnesses: []TranslationValidationV2Witness{
			registered,
			scalar,
			memory,
			loop,
			call,
			proof,
			allocation,
			hash,
		},
		Rows: []TranslationValidationV2Row{
			p23TranslationRow(
				TranslationValidationV2RegisteredPasses,
				"Registered optimizer passes",
				"current_supported_subset",
				[]string{
					("opt.RegisteredPasses returns every current optimizer pass " +
						"and opt.ValidatePassContract requires " +
						"translation_validation plus validation.ValidateTranslation."),
					("RegisteredPasses witness runs NewManager over all " +
						"registered passes and requires validation metadata on every " +
						"pass report."),
					("compiler/internal/opt/opt_core.go stores translation " +
						"reports, validation metadata, before dumps, after dumps, " +
						"and profile_input_policy rows."),
				},
				[]string{
					"go test ./compiler -run 'P23TranslationValidationV2|ValidateP23TranslationValidationV2'",
					"go test ./compiler/internal/opt -run 'Manager|BasicScalar|SCCP|Mem2Reg|Inline|Loop|LICM'",
				},
				[]string{
					"registered optimizer pass coverage is limited to opt.RegisteredPasses",
					"translation validation is an internal evidence hook, not a public optimization mode",
					"no exhaustive optimizer completeness is claimed",
				},
				[]string{p23TranslationRegisteredPassesWitnessID},
			),
			p23TranslationRow(
				TranslationValidationV2SymbolicScalar,
				"Symbolic scalar equivalence",
				"current_supported_subset",
				[]string{
					("validation.ValidateTranslation runs " +
						"validateSemanticLocalEquivalence over supported " +
						"straight-line scalar arithmetic and comparison rewrites."),
					("Symbolic scalar witness checks add-zero equivalence and " +
						"records semantic local equivalence plus differential " +
						"samples."),
					("Negative witness rejects a semantic local equivalence " +
						"mismatch when add-zero becomes add-one."),
				},
				[]string{
					("go test ./compiler/internal/validation -run " +
						"'ValidateTranslation.*Algebra|ValidateTranslationRejectsBadL" +
						"ocalAlgebraRewrite'"),
					"go test ./compiler -run 'P23TranslationValidationV2'",
				},
				[]string{
					"symbolic equivalence is limited to the current scalar i32 local subset",
					"unsupported expressions are skipped rather than trusted",
					"no full scalar theorem prover is claimed",
				},
				[]string{p23TranslationScalarWitnessID},
			),
			p23TranslationRow(
				TranslationValidationV2MemoryEquivalence,
				"Memory equivalence for supported i32 slice samples",
				"current_supported_subset",
				[]string{
					("differential.CheckBackendMatrix compares source, Stack IR, " +
						"optimized Stack IR, SSA, and Machine IR lanes for a " +
						"proof-tagged i32 slice sum memory sample."),
					("compiler/internal/differential/differential.go interprets " +
						"supported i32 slice memory through loadI32Slice and " +
						"storeI32Slice."),
					"Memory witness rejects a bad source oracle through the same backend matrix mismatch path.",
				},
				[]string{
					"go test ./compiler/internal/differential -run 'BackendMatrix'",
					"go test ./compiler -run 'P23TranslationValidationV2'",
				},
				[]string{
					"memory equivalence evidence is limited to supported i32 slice samples",
					"no broad memory model or alias model is claimed",
					"region/local slice coverage is evidence-bound to current lowering and allocation reports",
				},
				[]string{p23TranslationMemoryWitnessID},
			),
			p23TranslationRow(
				TranslationValidationV2BoundsProofPreservation,
				"Bounds proof preservation",
				"current_supported_subset",
				[]string{
					("validation.ValidateTranslation validates input/output proof " +
						"facts through CheckBoundsProofs and " +
						"validateProofFactMultiset."),
					"Proof witness preserves a proof-tagged unchecked i32 load and records proof facts compared.",
					("Negative witness rejects an unchecked bounds load whose " +
						"proof id disappears after transformation."),
				},
				[]string{
					("go test ./compiler/internal/validation -run " +
						"'ValidateTranslationRejectsProof|ValidateTranslationRejectsM" +
						"issingProof'"),
					"go test ./compiler -run 'P23TranslationValidationV2'",
				},
				[]string{
					"bounds proof preservation covers proof-tagged removed checks in current IR",
					"changed or missing proof ids are validation failures",
					"no unchecked load may be trusted without proof id evidence",
				},
				[]string{p23TranslationProofWitnessID},
			),
			p23TranslationRow(
				TranslationValidationV2AllocationPlanPreservation,
				"Allocation-plan preservation",
				"current_supported_subset",
				[]string{
					("validation.ValidateAllocationLowering checks allocation " +
						"plan rows against emitted Stack IR allocation lowering."),
					("Allocation witness validates a stack-lowered i32 slice " +
						"against allocplan.VerifyPlan and ValidateAllocationLowering."),
					"Negative witness rejects allocation drift when the matching Stack IR allocation is missing.",
				},
				[]string{
					"go test ./compiler/internal/validation -run 'ValidateAllocationLowering'",
					"go test ./compiler -run 'P23TranslationValidationV2'",
				},
				[]string{
					"allocation plan preservation is evidence-bound to current allocplan and lowering validators",
					"no broad allocation optimizer is claimed",
					"runtime behavior does not change",
				},
				[]string{p23TranslationAllocationWitnessID},
			),
			p23TranslationRow(
				TranslationValidationV2MachineCheckableHashes,
				"Machine-checkable before/after hashes",
				"current_supported_subset",
				[]string{
					("validation.BuildOptimizationValidationMetadata records " +
						"sha256 before and after IR hashes for translation-validated " +
						"optimization evidence."),
					("Hash witness builds metadata for a real add-zero rewrite " +
						"and records distinct before/after sha256 hashes."),
					"ValidateOptimizationValidationMetadata rejects missing or malformed hash metadata.",
				},
				[]string{
					"go test ./compiler/internal/validation -run 'OptimizationValidationMetadata'",
					"go test ./compiler -run 'P23TranslationValidationV2'",
				},
				[]string{
					"hashes are machine-checkable evidence, not proof by themselves",
					"hash evidence is scoped to the compared IR text and functions",
					"safe-program semantics do not change",
				},
				[]string{p23TranslationHashWitnessID},
			),
		},
		NonClaims: []string{
			"no full formal proof is claimed",
			"no exhaustive optimizer completeness is claimed",
			"no broad memory model or alias model is claimed",
			"no broad loop theorem prover is claimed",
			"no performance claim is made",
			"runtime behavior does not change",
			"safe-program semantics do not change",
		},
		RegisteredPassCoverageComplete:    registered.RegisteredPassCoverageComplete,
		SymbolicScalarEquivalenceSamples:  scalar.SymbolicScalarChecks,
		MemoryEquivalenceSamples:          memory.MemoryEquivalenceSamples,
		LoopEquivalenceSamples:            loop.LoopEquivalenceSamples,
		CallEquivalenceSamples:            call.CallEquivalenceSamples,
		BoundsProofsPreserved:             proof.BoundsProofsPreserved,
		AllocationPlanValidated:           allocation.AllocationPlanValidated,
		BeforeAfterHashesMachineCheckable: hash.HashesMachineCheckable,
	}
	if err := ValidateP23TranslationValidationV2(report); err != nil {
		return TranslationValidationV2Report{}, err
	}
	return report, nil
}

func ValidateP23TranslationValidationV2(report TranslationValidationV2Report) error {
	if report.SchemaVersion != translationValidationV2Schema {
		return fmt.Errorf("translation validation v2: schema_version is %q", report.SchemaVersion)
	}
	if report.Scope != translationValidationV2ScopeP230 {
		return fmt.Errorf("translation validation v2: scope is %q", report.Scope)
	}
	if report.FullFormalProofClaimed {
		return fmt.Errorf("translation validation v2: full formal proof claim is forbidden")
	}
	if report.ExhaustiveOptimizerCompletenessClaimed {
		return fmt.Errorf(
			"translation validation v2: exhaustive optimizer completeness claim is forbidden",
		)
	}
	if report.BroadMemoryModelClaimed {
		return fmt.Errorf("translation validation v2: broad memory model claim is forbidden")
	}
	if report.BroadLoopTheoremProverClaimed {
		return fmt.Errorf("translation validation v2: broad loop theorem prover claim is forbidden")
	}
	if report.PerformanceClaimed {
		return fmt.Errorf("translation validation v2: performance claim is forbidden")
	}
	if report.RuntimeBehaviorChanged {
		return fmt.Errorf("translation validation v2: runtime behavior change claim is forbidden")
	}
	if report.SafeSemanticsChanged {
		return fmt.Errorf("translation validation v2: safe semantics change claim is forbidden")
	}
	if !report.RegisteredPassCoverageComplete {
		return fmt.Errorf("translation validation v2: registered pass coverage is incomplete")
	}
	if report.SymbolicScalarEquivalenceSamples == 0 {
		return fmt.Errorf("translation validation v2: missing symbolic scalar equivalence samples")
	}
	if report.MemoryEquivalenceSamples == 0 {
		return fmt.Errorf("translation validation v2: missing memory equivalence samples")
	}
	if report.LoopEquivalenceSamples == 0 {
		return fmt.Errorf("translation validation v2: missing loop equivalence samples")
	}
	if report.CallEquivalenceSamples == 0 {
		return fmt.Errorf("translation validation v2: missing call equivalence samples")
	}
	if !report.BoundsProofsPreserved {
		return fmt.Errorf("translation validation v2: bounds proof preservation evidence missing")
	}
	if !report.AllocationPlanValidated {
		return fmt.Errorf("translation validation v2: allocation plan validation evidence missing")
	}
	if !report.BeforeAfterHashesMachineCheckable {
		return fmt.Errorf("translation validation v2: before/after hash evidence missing")
	}
	if err := p23ValidateRows(report.Rows, report.Witnesses); err != nil {
		return err
	}
	if err := p23ValidateWitnesses(report.Witnesses); err != nil {
		return err
	}
	for _, want := range []string{
		"no full formal proof is claimed",
		"no exhaustive optimizer completeness is claimed",
		"no broad memory model or alias model is claimed",
		"no broad loop theorem prover is claimed",
		"no performance claim is made",
		"runtime behavior does not change",
		"safe-program semantics do not change",
	} {
		if !p23TranslationHasString(report.NonClaims, want) {
			return fmt.Errorf("translation validation v2: missing non-claim %q", want)
		}
	}
	return nil
}

func buildP23RegisteredPassesWitness() (TranslationValidationV2Witness, error) {
	passes := opt.RegisteredPasses()
	for _, pass := range passes {
		if err := opt.ValidatePassContract(pass); err != nil {
			return TranslationValidationV2Witness{}, err
		}
	}
	report, err := opt.NewManager().Run(p23TinyProgram(), passes...)
	if err != nil {
		return TranslationValidationV2Witness{}, err
	}
	metadata := len(report.Passes) == len(passes)
	for _, row := range report.Passes {
		if !row.TranslationValidated || row.TranslationReport == nil ||
			row.ValidationMetadata == nil {
			metadata = false
			break
		}
		if row.ValidationMetadata.BeforeHash == "" || row.ValidationMetadata.AfterHash == "" {
			metadata = false
			break
		}
	}
	return TranslationValidationV2Witness{
		ID:                             p23TranslationRegisteredPassesWitnessID,
		Kind:                           "optimizer_manager_registered_passes",
		RegisteredPasses:               len(passes),
		RegisteredPassCoverageComplete: len(report.Passes) == len(passes) && metadata,
		TranslationMetadataPresent:     metadata,
	}, nil
}

func buildP23ScalarWitness() (TranslationValidationV2Witness, error) {
	before := p23SingleReturnProgram("main", 1,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 0},
		ir.IRInstr{Kind: ir.IRAddI32},
	)
	after := p23SingleReturnProgram("main", 1,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
	)
	report, err := validation.ValidateTranslation(before, after)
	if err != nil {
		return TranslationValidationV2Witness{}, err
	}
	badAfter := p23SingleReturnProgram("main", 1,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 1},
		ir.IRInstr{Kind: ir.IRAddI32},
	)
	_, badErr := validation.ValidateTranslation(before, badAfter)
	return TranslationValidationV2Witness{
		ID:                   p23TranslationScalarWitnessID,
		Kind:                 "symbolic_scalar_arithmetic",
		SymbolicScalarChecks: report.SemanticLocalChecks,
		DifferentialSamples:  report.DifferentialSamples,
		SemanticMismatchRejected: badErr != nil &&
			strings.Contains(badErr.Error(), "semantic local equivalence"),
	}, nil
}

func buildP23MemoryWitness() (TranslationValidationV2Witness, error) {
	tc := p23SliceMatrixCase(false)
	report, err := differential.CheckBackendMatrix(tc)
	if err != nil {
		return TranslationValidationV2Witness{}, err
	}
	bad := p23SliceMatrixCase(true)
	_, badErr := differential.CheckBackendMatrix(bad)
	return TranslationValidationV2Witness{
		ID:                       p23TranslationMemoryWitnessID,
		Kind:                     "i32_slice_memory_matrix",
		MemoryEquivalenceSamples: len(report.Samples),
		DifferentialLanes:        len(report.Lanes),
		MemoryMismatchRejected: badErr != nil &&
			strings.Contains(badErr.Error(), "differential mismatch"),
	}, nil
}

func buildP23LoopWitness() (TranslationValidationV2Witness, error) {
	report, err := differential.CheckBackendMatrix(differential.BackendMatrixCase{
		Name:      "p23-loop-sum",
		Functions: []ir.IRFunc{p23LoopSumFunc()},
		Entry:     "sum_n",
		Samples: []differential.MatrixSample{
			{Name: "zero", Args: []int32{0}},
			{Name: "five", Args: []int32{5}},
		},
		Source: func(sample differential.MatrixSample) (int32, bool) {
			n := sample.Args[0]
			var total int32
			for i := int32(0); i < n; i++ {
				total += i
			}
			return total, true
		},
	})
	if err != nil {
		return TranslationValidationV2Witness{}, err
	}
	return TranslationValidationV2Witness{
		ID:                     p23TranslationLoopWitnessID,
		Kind:                   "loop_backend_matrix",
		LoopEquivalenceSamples: len(report.Samples),
		DifferentialLanes:      len(report.Lanes),
	}, nil
}

func buildP23CallInliningWitness() (TranslationValidationV2Witness, error) {
	funcs := []ir.IRFunc{p23HelperAddOneFunc(), p23CallHelperFunc()}
	tc := differential.BackendMatrixCase{
		Name:          "p23-call-inline",
		Functions:     funcs,
		Entry:         "main",
		Samples:       []differential.MatrixSample{{Name: "seven", Args: []int32{7}}},
		Optimizations: []opt.Pass{opt.InlineSmallPurePass()},
		Source: func(sample differential.MatrixSample) (int32, bool) {
			return sample.Args[0] + 1, true
		},
	}
	matrix, err := differential.CheckBackendMatrix(tc)
	if err != nil {
		return TranslationValidationV2Witness{}, err
	}
	prog := &ir.IRProgram{MainIndex: 1, MainName: "main", Funcs: p23CloneFuncs(funcs)}
	runReport, err := opt.NewManager().Run(prog, opt.InlineSmallPurePass())
	if err != nil {
		return TranslationValidationV2Witness{}, err
	}
	return TranslationValidationV2Witness{
		ID:                     p23TranslationCallInliningWitnessID,
		Kind:                   "call_inlining_backend_matrix",
		CallEquivalenceSamples: len(matrix.Samples),
		DifferentialLanes:      len(matrix.Lanes),
		BeforeHadCall:          p23ProgramHasCall(&ir.IRProgram{Funcs: funcs}),
		AfterHadCall:           p23ProgramHasCall(prog),
		TranslationValidated: len(runReport.Passes) == 1 &&
			runReport.Passes[0].TranslationValidated,
	}, nil
}

func buildP23ProofWitness() (TranslationValidationV2Witness, error) {
	before := p23ProofProgram("main", "proof:while:i:xs:1:1")
	after := p23ProofProgram("main", "proof:while:i:xs:1:1")
	report, err := validation.ValidateTranslation(before, after)
	if err != nil {
		return TranslationValidationV2Witness{}, err
	}
	missingProof := p23ProofProgram("main", "")
	_, badErr := validation.ValidateTranslation(before, missingProof)
	return TranslationValidationV2Witness{
		ID:                    p23TranslationProofWitnessID,
		Kind:                  "bounds_proof_multiset",
		ProofFactsCompared:    report.ProofFactsCompared,
		BoundsProofsPreserved: report.ProofFactsCompared > 0,
		MissingProofRejected: badErr != nil &&
			strings.Contains(badErr.Error(), "missing proof id"),
	}, nil
}

func buildP23AllocationWitness() (TranslationValidationV2Witness, error) {
	plan := p23AllocationPlan("main")
	prog := p23AllocationProgram("main", true)
	if err := validation.ValidateAllocationLowering(plan, prog); err != nil {
		return TranslationValidationV2Witness{}, err
	}
	badProg := p23AllocationProgram("main", false)
	badErr := validation.ValidateAllocationLowering(plan, badProg)
	return TranslationValidationV2Witness{
		ID:                      p23TranslationAllocationWitnessID,
		Kind:                    "allocation_lowering_validation",
		AllocationPlanValidated: true,
		AllocationDriftRejected: badErr != nil &&
			strings.Contains(badErr.Error(), "no matching IR stack slice"),
	}, nil
}

func buildP23HashWitness() (TranslationValidationV2Witness, error) {
	before := p23SingleReturnProgram("main", 1,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
		ir.IRInstr{Kind: ir.IRConstI32, Imm: 0},
		ir.IRInstr{Kind: ir.IRAddI32},
	)
	after := p23SingleReturnProgram("main", 1,
		ir.IRInstr{Kind: ir.IRLoadLocal, Local: 0},
	)
	meta, err := validation.BuildOptimizationValidationMetadata(
		before,
		after,
		validation.OptimizationMetadataOptions{
			PassName:                  "basic-scalar",
			InputKind:                 string(opt.IRKindStack),
			OutputKind:                string(opt.IRKindStack),
			InputVerifier:             opt.VerifierLowerVerifyProgram,
			OutputVerifier:            opt.VerifierLowerVerifyProgram,
			ValidationStrategy:        string(opt.ValidationTranslation),
			RequiredFacts:             []string{string(opt.FactIRVerified)},
			PreservedFacts:            []string{string(opt.FactBoundsProofs)},
			InvalidatedFacts:          []string{string(opt.FactLiveness)},
			ProofRule:                 string(opt.ProofRulePreserveBoundsInvalidateLiveness),
			TranslationValidationHook: opt.TranslationHookValidateTranslation,
			ReportRows:                opt.RequiredP17ReportRows(),
			NegativeTestMarker:        opt.NegativeTestPassContractV1,
			ProfileInputPolicy:        string(opt.ProfileInputUnused),
		},
	)
	if err != nil {
		return TranslationValidationV2Witness{}, err
	}
	return TranslationValidationV2Witness{
		ID:         p23TranslationHashWitnessID,
		Kind:       "optimization_validation_metadata_hashes",
		BeforeHash: meta.BeforeHash,
		AfterHash:  meta.AfterHash,
		HashesMachineCheckable: strings.HasPrefix(meta.BeforeHash, "sha256:") &&
			strings.HasPrefix(meta.AfterHash, "sha256:"),
		HashesDistinct: meta.BeforeHash != meta.AfterHash,
	}, nil
}

func p23ValidateRows(
	rows []TranslationValidationV2Row,
	witnesses []TranslationValidationV2Witness,
) error {
	witnessByID := map[string]bool{}
	for _, witness := range witnesses {
		witnessByID[witness.ID] = true
	}
	seen := map[TranslationValidationV2ID]bool{}
	for _, row := range rows {
		if row.ID == "" || row.Name == "" || row.Status == "" || len(row.Evidence) == 0 ||
			len(row.Tests) == 0 ||
			len(row.Boundaries) == 0 ||
			len(row.WitnessIDs) == 0 {
			return fmt.Errorf("translation validation v2: row %q missing required metadata", row.ID)
		}
		if seen[row.ID] {
			return fmt.Errorf("translation validation v2: duplicate row %s", row.ID)
		}
		seen[row.ID] = true
		if p23ContainsPlaceholder(row.Evidence) || p23ContainsPlaceholder(row.Boundaries) {
			return fmt.Errorf(
				"translation validation v2: row %s contains placeholder evidence",
				row.ID,
			)
		}
		for _, witnessID := range row.WitnessIDs {
			if !witnessByID[witnessID] {
				return fmt.Errorf(
					"translation validation v2: row %s references missing witness %q",
					row.ID,
					witnessID,
				)
			}
		}
	}
	for _, id := range p23TranslationValidationV2IDs() {
		if !seen[id] {
			return fmt.Errorf("translation validation v2: missing row %s", id)
		}
	}
	return nil
}

func p23ValidateWitnesses(witnesses []TranslationValidationV2Witness) error {
	byID := map[string]TranslationValidationV2Witness{}
	for _, witness := range witnesses {
		byID[witness.ID] = witness
	}
	registered := byID[p23TranslationRegisteredPassesWitnessID]
	if registered.RegisteredPasses < len(opt.RegisteredPasses()) ||
		!registered.RegisteredPassCoverageComplete ||
		!registered.TranslationMetadataPresent {
		return fmt.Errorf(
			"translation validation v2: registered pass witness incomplete: %+v",
			registered,
		)
	}
	scalar := byID[p23TranslationScalarWitnessID]
	if scalar.SymbolicScalarChecks == 0 || scalar.DifferentialSamples == 0 ||
		!scalar.SemanticMismatchRejected {
		return fmt.Errorf(
			"translation validation v2: symbolic scalar witness incomplete: %+v",
			scalar,
		)
	}
	memory := byID[p23TranslationMemoryWitnessID]
	if memory.MemoryEquivalenceSamples == 0 || memory.DifferentialLanes < 5 ||
		!memory.MemoryMismatchRejected {
		return fmt.Errorf(
			"translation validation v2: memory equivalence witness incomplete: %+v",
			memory,
		)
	}
	loop := byID[p23TranslationLoopWitnessID]
	if loop.LoopEquivalenceSamples == 0 || loop.DifferentialLanes < 5 {
		return fmt.Errorf(
			"translation validation v2: loop equivalence witness incomplete: %+v",
			loop,
		)
	}
	call := byID[p23TranslationCallInliningWitnessID]
	if call.CallEquivalenceSamples == 0 || !call.BeforeHadCall || call.AfterHadCall ||
		!call.TranslationValidated {
		return fmt.Errorf("translation validation v2: call/inlining witness incomplete: %+v", call)
	}
	proof := byID[p23TranslationProofWitnessID]
	if proof.ProofFactsCompared == 0 || !proof.BoundsProofsPreserved ||
		!proof.MissingProofRejected {
		return fmt.Errorf("translation validation v2: bounds proof witness incomplete: %+v", proof)
	}
	allocation := byID[p23TranslationAllocationWitnessID]
	if !allocation.AllocationPlanValidated || !allocation.AllocationDriftRejected {
		return fmt.Errorf(
			"translation validation v2: allocation plan witness incomplete: %+v",
			allocation,
		)
	}
	hash := byID[p23TranslationHashWitnessID]
	if !strings.HasPrefix(hash.BeforeHash, "sha256:") ||
		!strings.HasPrefix(hash.AfterHash, "sha256:") ||
		!hash.HashesMachineCheckable ||
		!hash.HashesDistinct {
		return fmt.Errorf("translation validation v2: hash witness incomplete: %+v", hash)
	}
	return nil
}

func p23TranslationValidationV2IDs() []TranslationValidationV2ID {
	return []TranslationValidationV2ID{
		TranslationValidationV2RegisteredPasses,
		TranslationValidationV2SymbolicScalar,
		TranslationValidationV2MemoryEquivalence,
		TranslationValidationV2BoundsProofPreservation,
		TranslationValidationV2AllocationPlanPreservation,
		TranslationValidationV2MachineCheckableHashes,
	}
}

func p23TranslationRow(
	id TranslationValidationV2ID,
	name string,
	status string,
	evidence []string,
	tests []string,
	boundaries []string,
	witnessIDs []string,
) TranslationValidationV2Row {
	return TranslationValidationV2Row{
		ID:         id,
		Name:       name,
		Status:     status,
		Evidence:   evidence,
		Tests:      tests,
		Boundaries: boundaries,
		WitnessIDs: witnessIDs,
	}
}

func p23TranslationHasString(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

func p23ContainsPlaceholder(values []string) bool {
	for _, value := range values {
		text := strings.ToLower(strings.TrimSpace(value))
		if text == "" || strings.Contains(text, "todo") || strings.Contains(text, "placeholder") ||
			strings.Contains(text, "paper-only") {
			return true
		}
	}
	return false
}

func p23TinyProgram() *ir.IRProgram {
	return p23SingleReturnProgram("main", 0, ir.IRInstr{Kind: ir.IRConstI32, Imm: 1})
}

func p23SingleReturnProgram(name string, params int, instrs ...ir.IRInstr) *ir.IRProgram {
	body := append([]ir.IRInstr(nil), instrs...)
	body = append(body, ir.IRInstr{Kind: ir.IRReturn})
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  name,
		Funcs: []ir.IRFunc{{
			Name:        name,
			ParamSlots:  params,
			LocalSlots:  params,
			ReturnSlots: 1,
			Instrs:      body,
		}},
	}
}

func p23SliceMatrixCase(badSource bool) differential.BackendMatrixCase {
	return differential.BackendMatrixCase{
		Name:      "p23-slice-memory",
		Functions: []ir.IRFunc{p23SliceSumFunc()},
		Entry:     "sum",
		Samples: []differential.MatrixSample{{
			Name:      "four-elements",
			Args:      []int32{1, 4},
			I32Slices: map[int32][]int32{1: {1, 2, 3, 4}},
		}},
		Source: func(sample differential.MatrixSample) (int32, bool) {
			xs := sample.I32Slices[sample.Args[0]]
			var total int32
			for i := int32(0); i < sample.Args[1]; i++ {
				total += xs[i]
			}
			if badSource {
				total++
			}
			return total, true
		},
	}
}

func p23SliceSumFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "sum",
		ParamSlots:  2,
		LocalSlots:  4,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRIndexLoadI32Unchecked, ProofID: "proof:while:i:xs:1:1"},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func p23LoopSumFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "sum_n",
		ParamSlots:  1,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 1},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRReturn},
		},
	}
}

func p23HelperAddOneFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "inc",
		ParamSlots:  1,
		LocalSlots:  1,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRReturn},
		},
	}
}

func p23CallHelperFunc() ir.IRFunc {
	return ir.IRFunc{
		Name:        "main",
		ParamSlots:  1,
		LocalSlots:  1,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRCall, Name: "inc", ArgSlots: 1, RetSlots: 1},
			{Kind: ir.IRReturn},
		},
	}
}

func p23ProofProgram(name string, proofID string) *ir.IRProgram {
	return &ir.IRProgram{
		MainIndex: 0,
		MainName:  name,
		Funcs: []ir.IRFunc{{
			Name:        name,
			ParamSlots:  2,
			LocalSlots:  2,
			ReturnSlots: 1,
			Instrs: []ir.IRInstr{
				{Kind: ir.IRLoadLocal, Local: 0},
				{Kind: ir.IRLoadLocal, Local: 1},
				{Kind: ir.IRConstI32, Imm: 0},
				{Kind: ir.IRIndexLoadI32Unchecked, ProofID: proofID},
				{Kind: ir.IRReturn},
			},
		}},
	}
}

func p23AllocationPlan(name string) *allocplan.Plan {
	return &allocplan.Plan{Functions: []allocplan.FunctionPlan{{
		Name: name,
		Allocations: []allocplan.Allocation{{
			ID:                    "xs",
			SiteID:                "allocsite:" + name + ":xs:line_1_1",
			ValueID:               "alloc_intent:xs",
			Builtin:               "core.make_i32",
			ElementType:           "i32",
			ElementSize:           4,
			LengthExpr:            "4",
			LengthStatus:          allocplan.LengthStatusNormal,
			ZeroGuardStatus:       "valid_empty_no_allocator",
			NegativeGuardStatus:   "reject_before_allocation",
			OverflowGuardStatus:   "reject_before_allocation",
			ByteSize:              16,
			Escape:                allocplan.EscapeNoEscape,
			Storage:               allocplan.StorageStack,
			PlannedStorage:        allocplan.StorageStack,
			ActualLoweringStorage: allocplan.StorageStack,
			ValidationStatus:      "validated_no_escape",
			LoweringStatus:        "stack_lowering",
			Reason:                "p23 translation validation allocation witness",
		}},
	}}}
}

func p23AllocationProgram(name string, stackLowered bool) *ir.IRProgram {
	instrs := []ir.IRInstr{
		{Kind: ir.IRConstI32, Imm: 4},
	}
	if stackLowered {
		instrs = append(
			instrs,
			ir.IRInstr{Kind: ir.IRStackSliceI32, Local: 2, ArgSlots: 4, Imm: 4, Name: "xs"},
		)
	} else {
		instrs = append(instrs, ir.IRInstr{Kind: ir.IRMakeSliceI32, Name: "xs"})
	}
	instrs = append(instrs,
		ir.IRInstr{Kind: ir.IRStoreLocal, Local: 1},
		ir.IRInstr{Kind: ir.IRStoreLocal, Local: 0},
	)
	return &ir.IRProgram{Funcs: []ir.IRFunc{{
		Name:       name,
		LocalSlots: 6,
		Instrs:     instrs,
	}}}
}

func p23CloneFuncs(funcs []ir.IRFunc) []ir.IRFunc {
	out := make([]ir.IRFunc, len(funcs))
	for i, fn := range funcs {
		out[i] = fn
		out[i].Instrs = append([]ir.IRInstr(nil), fn.Instrs...)
	}
	return out
}

func p23ProgramHasCall(prog *ir.IRProgram) bool {
	if prog == nil {
		return false
	}
	for _, fn := range prog.Funcs {
		for _, instr := range fn.Instrs {
			if instr.Kind == ir.IRCall {
				return true
			}
		}
	}
	return false
}
