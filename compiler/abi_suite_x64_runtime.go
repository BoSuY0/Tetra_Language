package compiler

import (
	"tetra_language/compiler/internal/abisuite"
	ctarget "tetra_language/compiler/target"
)

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
		BuildExecutableWithOptions: func(srcPath string, outPath string, target string, opts abisuite.RuntimeBuildOptions) error {
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
		if sym.Name == name && sym.HasSignature && sym.ParamSlots == params && sym.ReturnSlots == returns {
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
