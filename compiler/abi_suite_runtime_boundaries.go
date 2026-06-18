package compiler

import (
	"tetra_language/compiler/internal/abisuite"
	ctarget "tetra_language/compiler/target"
)

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
	return abisuite.CheckSurfaceDistributedRuntimeBoundaryDiagnostics(tgt, abiSuiteRuntimeBoundaryDeps())
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
