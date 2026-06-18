package compiler

import (
	"tetra_language/compiler/internal/buildruntime"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/semantics"
)

func validateNativeRuntimeBeforeCodegen(checked *semantics.CheckedProgram, target string) error {
	if checked == nil || target != "linux-x86" {
		return nil
	}
	typedTasksUsed, typedTaskMaxSlots := collectTypedTaskRuntimeUsage(checked)
	if !typedTasksUsed {
		return nil
	}
	_, tasksPos := collectTaskRuntimeUsagePosition(checked)
	caps := nativeRuntimeCapabilitiesForTarget(target)
	if caps.maxTypedTaskSlots > 0 && typedTaskMaxSlots > caps.maxTypedTaskSlots {
		return targetRuntimeDiagnostic(tasksPos, target, "staged typed task")
	}
	return nil
}

func collectDistributedActorRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	return buildruntime.CollectDistributedActorRuntimeUsagePosition(checked)
}

func collectTimeRuntimeUsage(checked *semantics.CheckedProgram) bool {
	return buildruntime.CollectTimeRuntimeUsage(checked)
}

func collectTimeRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	return buildruntime.CollectTimeRuntimeUsagePosition(checked)
}

func collectFilesystemRuntimeUsage(checked *semantics.CheckedProgram) bool {
	return buildruntime.CollectFilesystemRuntimeUsage(checked)
}

func collectFilesystemRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	return buildruntime.CollectFilesystemRuntimeUsagePosition(checked)
}

func collectNetRuntimeUsage(checked *semantics.CheckedProgram) bool {
	return buildruntime.CollectNetRuntimeUsage(checked)
}

func collectNetRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	return buildruntime.CollectNetRuntimeUsagePosition(checked)
}

type netRuntimeUsageProfile struct {
	used            bool
	firstPos        frontend.Position
	symbolPositions map[string]frontend.Position
}

func (u netRuntimeUsageProfile) requiredSymbols() []string {
	return u.buildRuntimeProfile().RequiredSymbols()
}

func (u netRuntimeUsageProfile) buildRuntimeProfile() buildruntime.NetRuntimeUsageProfile {
	return buildruntime.NetRuntimeUsageProfile{
		Used:            u.used,
		FirstPos:        u.firstPos,
		SymbolPositions: u.symbolPositions,
	}
}

func collectNetRuntimeUsageProfile(checked *semantics.CheckedProgram) netRuntimeUsageProfile {
	usage := buildruntime.CollectNetRuntimeUsageProfile(checked)
	return netRuntimeUsageProfile{
		used:            usage.Used,
		firstPos:        usage.FirstPos,
		symbolPositions: usage.SymbolPositions,
	}
}

func netRuntimeSymbolForBuiltin(name string) (string, bool) {
	return buildruntime.NetRuntimeSymbolForBuiltin(name)
}

func collectSurfaceRuntimeUsage(checked *semantics.CheckedProgram) bool {
	return buildruntime.CollectSurfaceRuntimeUsage(checked)
}

func collectSurfaceRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	return buildruntime.CollectSurfaceRuntimeUsagePosition(checked)
}
