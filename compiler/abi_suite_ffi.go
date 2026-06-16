package compiler

import (
	"tetra_language/compiler/internal/abisuite"
	ctarget "tetra_language/compiler/target"
)

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
	return abisuite.CheckFunctionPointerFFIDiagnostics(targetName, boundaryName, stem, abiSuiteFFICheckDeps())
}

func abiSuiteFFICheckDeps() abisuite.FFICheckDeps {
	return abisuite.FFICheckDeps{
		BuildLibrary: func(srcPath string, outPath string, target string) error {
			_, err := BuildFileWithStatsOpt(srcPath, outPath, target, BuildOptions{Emit: EmitLibrary, Jobs: 1})
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
