package compiler

import (
	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/buildreports"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/lower"
	"tetra_language/compiler/internal/machine"
)

func buildBackendReport(target string, irProg *ir.IRProgram) backendReport {
	return buildreports.BuildBackendReport(target, irProg)
}

func buildMachineBackendFunctionReport(fn machine.Function, path string, callerSaved []machine.PhysReg, ssaVerified bool) (machineBackendFunctionReport, bool) {
	return buildreports.BuildMachineBackendFunctionReport(fn, path, callerSaved, ssaVerified)
}

func allocationPlanOptionsForTarget(target string) allocplan.Options {
	return allocplan.Options{
		EnableStackLowering:    targetSupportsStackAllocationLowering(target),
		EnableSmallHeapRuntime: target == "linux-x64",
		EnableRegionPlanning:   target == "linux-x64",
		EnableRegionLowering:   target == "linux-x64",
	}
}

func lowerOptionsForTarget(target string) lower.Options {
	return lower.Options{
		StackAllocationLowering:    targetSupportsStackAllocationLowering(target),
		FunctionTempRegionLowering: target == "linux-x64",
	}
}

func targetSupportsStackAllocationLowering(target string) bool {
	switch target {
	case "linux-x64", "macos-x64", "windows-x64":
		return true
	default:
		return false
	}
}
