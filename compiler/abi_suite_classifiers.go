package compiler

import (
	"tetra_language/compiler/internal/abisuite"
	ctarget "tetra_language/compiler/target"
)

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
