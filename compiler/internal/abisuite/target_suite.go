package abisuite

import (
	"fmt"

	ctarget "tetra_language/compiler/target"
)

type TargetCheckRunner func(tgt ctarget.Target) []Check

type TargetCheckRunners struct {
	X86  TargetCheckRunner
	X32  TargetCheckRunner
	X64  TargetCheckRunner
	WASM TargetCheckRunner
}

func RunTargetChecks(targetName string, runners TargetCheckRunners) ([]Check, error) {
	tgt, err := ctarget.Parse(targetName)
	if err != nil {
		return nil, err
	}
	switch {
	case tgt.Arch == ctarget.ArchX86 && tgt.ABI == ctarget.ABI386SysV:
		return runTargetCheckRunner("x86", runners.X86, tgt)
	case tgt.Arch == ctarget.ArchX64 && tgt.ABI == ctarget.ABIX32SysV:
		return runTargetCheckRunner("x32", runners.X32, tgt)
	case tgt.Arch == ctarget.ArchX64:
		return runTargetCheckRunner("x64", runners.X64, tgt)
	case tgt.Arch == ctarget.ArchWASM32:
		return runTargetCheckRunner("wasm", runners.WASM, tgt)
	default:
		return nil, UnsupportedTargetError(tgt.Triple)
	}
}

func X64CheckPrefix(tgt ctarget.Target) string {
	switch tgt.Triple {
	case "windows-x64", "macos-x64":
		return tgt.Triple
	default:
		return "x64"
	}
}

func runTargetCheckRunner(name string, runner TargetCheckRunner, tgt ctarget.Target) ([]Check, error) {
	if runner == nil {
		return nil, fmt.Errorf("missing ABI suite runner for %s", name)
	}
	return runner(tgt), nil
}
