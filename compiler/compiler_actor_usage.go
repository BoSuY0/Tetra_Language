package compiler

import (
	"tetra_language/compiler/internal/buildruntime"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/semantics"
)

func collectActorEntries(checked *semantics.CheckedProgram) (bool, []string, int, error) {
	return buildruntime.CollectActorEntries(checked)
}

func collectActorStateRuntimeUsage(checked *semantics.CheckedProgram) bool {
	return buildruntime.CollectActorStateRuntimeUsage(checked)
}

func collectActorStateRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	return buildruntime.CollectActorStateRuntimeUsagePosition(checked)
}

func collectActorRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	return buildruntime.CollectActorRuntimeUsagePosition(checked)
}

func collectTaskRuntimeUsage(checked *semantics.CheckedProgram) bool {
	return buildruntime.CollectTaskRuntimeUsage(checked)
}

func collectTaskRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	return buildruntime.CollectTaskRuntimeUsagePosition(checked)
}

func collectTaskGroupRuntimeUsage(checked *semantics.CheckedProgram) bool {
	return buildruntime.CollectTaskGroupRuntimeUsage(checked)
}

func collectTypedTaskRuntimeUsage(checked *semantics.CheckedProgram) (bool, int) {
	return buildruntime.CollectTypedTaskRuntimeUsage(checked)
}
