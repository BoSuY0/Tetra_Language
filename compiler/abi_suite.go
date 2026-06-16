package compiler

import (
	"tetra_language/compiler/internal/abisuite"
	ctarget "tetra_language/compiler/target"
)

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
		{name: "x86 i386 SysV classifier", run: func() error { return checkX86I386Classifier(tgt) }},
		{name: "x86 varargs and sret ABI", run: func() error { return checkX86VarargsAndSRet(tgt) }},
		{name: "x86 pointer FFI object smoke", run: func() error { return checkPointerFFIObjectSmoke(tgt) }},
		{name: "x86 c_int FFI object smoke", run: func() error { return checkCIntFFIObjectSmoke(tgt) }},
		{name: "x86 c_uint FFI object smoke", run: func() error { return checkCUIntFFIObjectSmoke(tgt) }},
		{name: "x86 ILP32 native/libc FFI object smoke", run: func() error { return checkILP32NativeLibcFFIObjectSmoke(tgt) }},
		{name: "x86 ref FFI null-return diagnostics", run: checkX86RefFFINullReturnDiagnostics},
		{name: "x86 function-pointer FFI diagnostics", run: checkX86FunctionPointerFFIDiagnostics},
		{name: "x86 source native scalar diagnostics", run: func() error { return checkSourceNativeScalarDiagnostics(tgt) }},
		{name: "x86 stdout executable smoke", run: checkX86StdoutExecutableSmoke},
		{name: "x86 stderr fd runtime smoke", run: checkX86StderrFDRuntimeSmoke},
		{name: "x86 allocator executable smoke", run: checkX86AllocatorExecutableSmoke},
		{name: "x86 allocator failure executable smoke", run: checkX86AllocatorFailureExecutableSmoke},
		{name: "x86 raw memory bounds executable smoke", run: checkX86RawMemoryBoundsExecutableSmoke},
		{name: "x86 raw pointer slot executable smoke", run: checkX86RawPointerSlotExecutableSmoke},
		{name: "x86 raw pointer offset slot executable smoke", run: checkX86RawPointerOffsetSlotExecutableSmoke},
		{name: "x86 island free executable smoke", run: checkX86IslandFreeExecutableSmoke},
		{name: "x86 stdlib runtime boundary diagnostics", run: func() error { return checkStdlibRuntimeBoundaryDiagnostics(tgt) }},
		{name: "x86 filesystem runtime smoke", run: checkX86FilesystemRuntimeSmoke},
		{name: "x86 filesystem scheduler composition smoke", run: checkX86FilesystemSchedulerCompositionSmoke},
		{name: "x86 time runtime smoke", run: checkX86TimeRuntimeSmoke},
		{name: "x86 single-actor self-host runtime smoke", run: checkX86SingleActorSelfHostRuntimeSmoke},
		{name: "x86 single-task self-host runtime smoke", run: checkX86SingleTaskSelfHostRuntimeSmoke},
		{name: "x86 typed-task self-host runtime smoke", run: checkX86TypedTaskSelfHostRuntimeSmoke},
		{name: "x86 staged typed-task self-host runtime smoke", run: checkX86StagedTypedTaskSelfHostRuntimeSmoke},
		{name: "x86 task-group self-host runtime smoke", run: checkX86TaskGroupSelfHostRuntimeSmoke},
		{name: "x86 typed-task-group self-host runtime smoke", run: checkX86TypedTaskGroupSelfHostRuntimeSmoke},
		{name: "x86 actor-state self-host runtime smoke", run: checkX86ActorStateSelfHostRuntimeSmoke},
		{name: "x86 ctx_switch object smoke", run: checkX86CtxSwitchObjectSmoke},
		{name: "x86 target runtime boundary diagnostics", run: func() error { return checkTargetRuntimeBoundaryDiagnostics(tgt) }},
		{name: "x86 networking runtime boundary diagnostics", run: func() error { return checkNetworkingRuntimeBoundaryDiagnostics(tgt) }},
		{name: "x86 networking lifecycle runtime smoke", run: checkX86NetworkingLifecycleRuntimeSmoke},
		{name: "x86 surface/distributed runtime boundary diagnostics", run: func() error { return checkSurfaceDistributedRuntimeBoundaryDiagnostics(tgt) }},
		{name: "x86 pointer atomic ABI width", run: func() error { return checkAtomicPointerObjectWidth(tgt) }},
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
		{name: prefix + " " + abiName + " classifier", run: func() error { return checkX64Classifier(tgt) }},
		{name: prefix + " " + abiName + " varargs and aggregates", run: func() error { return checkX64VarargsAndAggregates(tgt) }},
	}
	if tgt.Triple == "macos-x64" || tgt.Triple == "windows-x64" {
		checks = append(checks, struct {
			name string
			run  func() error
		}{name: prefix + " object ABI smoke", run: func() error { return checkX64PlatformObjectABISmoke(tgt) }})
	}
	checks = append(checks, struct {
		name string
		run  func() error
	}{name: prefix + " source native scalar diagnostics", run: func() error { return checkSourceNativeScalarDiagnostics(tgt) }})
	if tgt.Triple == "linux-x64" {
		checks = append(checks, struct {
			name string
			run  func() error
		}{name: prefix + " pointer FFI regression smoke", run: checkX64PointerFFIRegressionSmoke})
		checks = append(checks, struct {
			name string
			run  func() error
		}{name: prefix + " c_int FFI object smoke", run: func() error { return checkCIntFFIObjectSmoke(tgt) }})
		checks = append(checks, struct {
			name string
			run  func() error
		}{name: prefix + " c_uint FFI object smoke", run: func() error { return checkCUIntFFIObjectSmoke(tgt) }})
		checks = append(checks, struct {
			name string
			run  func() error
		}{name: prefix + " filesystem scheduler composition smoke", run: checkX64FilesystemSchedulerCompositionSmoke})
		checks = append(checks, struct {
			name string
			run  func() error
		}{name: prefix + " networking runtime smoke", run: checkX64NetworkingRuntimeSmoke})
		checks = append(checks, struct {
			name string
			run  func() error
		}{name: prefix + " scheduler restriction regression smoke", run: checkX64SchedulerRestrictionRegressionSmoke})
	}
	checks = append(checks, struct {
		name string
		run  func() error
	}{name: prefix + " pointer atomic ABI width", run: func() error { return checkAtomicPointerObjectWidth(tgt) }})
	return runABIChecks(checks)
}

func runX32ABIChecks(tgt ctarget.Target) []ABICheck {
	return runABIChecks([]struct {
		name string
		run  func() error
	}{
		{name: "x32 target model", run: func() error { return checkX32TargetModel(tgt) }},
		{name: "x32 SysV classifier", run: func() error { return checkX32SysVClassifier(tgt) }},
		{name: "x32 SysV varargs and aggregates", run: func() error { return checkX32SysVVarargsAndAggregates(tgt) }},
		{name: "x32 pointer FFI object smoke", run: func() error { return checkPointerFFIObjectSmoke(tgt) }},
		{name: "x32 c_int FFI object smoke", run: func() error { return checkCIntFFIObjectSmoke(tgt) }},
		{name: "x32 c_uint FFI object smoke", run: func() error { return checkCUIntFFIObjectSmoke(tgt) }},
		{name: "x32 ILP32 native/libc FFI object smoke", run: func() error { return checkILP32NativeLibcFFIObjectSmoke(tgt) }},
		{name: "x32 ref FFI null-return diagnostics", run: checkX32RefFFINullReturnDiagnostics},
		{name: "x32 function-pointer FFI diagnostics", run: checkX32FunctionPointerFFIDiagnostics},
		{name: "x32 source native scalar diagnostics", run: func() error { return checkSourceNativeScalarDiagnostics(tgt) }},
		{name: "x32 stdout executable smoke", run: checkX32StdoutExecutableSmoke},
		{name: "x32 stderr fd runtime smoke", run: checkX32StderrFDRuntimeSmoke},
		{name: "x32 allocator executable smoke", run: checkX32AllocatorExecutableSmoke},
		{name: "x32 allocator failure executable smoke", run: checkX32AllocatorFailureExecutableSmoke},
		{name: "x32 raw memory bounds executable smoke", run: checkX32RawMemoryBoundsExecutableSmoke},
		{name: "x32 raw pointer slot executable smoke", run: checkX32RawPointerSlotExecutableSmoke},
		{name: "x32 raw pointer offset slot executable smoke", run: checkX32RawPointerOffsetSlotExecutableSmoke},
		{name: "x32 island free executable smoke", run: checkX32IslandFreeExecutableSmoke},
		{name: "x32 stdlib runtime boundary diagnostics", run: func() error { return checkStdlibRuntimeBoundaryDiagnostics(tgt) }},
		{name: "x32 time runtime smoke", run: checkX32TimeRuntimeSmoke},
		{name: "x32 filesystem runtime smoke", run: checkX32FilesystemRuntimeSmoke},
		{name: "x32 filesystem scheduler composition smoke", run: checkX32FilesystemSchedulerCompositionSmoke},
		{name: "x32 single-actor self-host runtime smoke", run: checkX32SingleActorSelfHostRuntimeSmoke},
		{name: "x32 single-task self-host runtime smoke", run: checkX32SingleTaskSelfHostRuntimeSmoke},
		{name: "x32 typed-task self-host runtime smoke", run: checkX32TypedTaskSelfHostRuntimeSmoke},
		{name: "x32 staged typed-task self-host runtime smoke", run: checkX32StagedTypedTaskSelfHostRuntimeSmoke},
		{name: "x32 task-group self-host runtime smoke", run: checkX32TaskGroupSelfHostRuntimeSmoke},
		{name: "x32 typed-task-group self-host runtime smoke", run: checkX32TypedTaskGroupSelfHostRuntimeSmoke},
		{name: "x32 actor-state self-host runtime smoke", run: checkX32ActorStateSelfHostRuntimeSmoke},
		{name: "x32 ctx_switch object smoke", run: checkX32CtxSwitchObjectSmoke},
		{name: "x32 target runtime boundary diagnostics", run: func() error { return checkTargetRuntimeBoundaryDiagnostics(tgt) }},
		{name: "x32 networking runtime boundary diagnostics", run: func() error { return checkNetworkingRuntimeBoundaryDiagnostics(tgt) }},
		{name: "x32 networking lifecycle runtime smoke", run: checkX32NetworkingLifecycleRuntimeSmoke},
		{name: "x32 surface/distributed runtime boundary diagnostics", run: func() error { return checkSurfaceDistributedRuntimeBoundaryDiagnostics(tgt) }},
		{name: "x32 pointer atomic ABI width", run: func() error { return checkAtomicPointerObjectWidth(tgt) }},
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
