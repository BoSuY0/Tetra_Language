package compiler

import (
	"tetra_language/compiler/internal/buildruntime"
	"tetra_language/compiler/internal/semantics"
)

func buildActorDispatchFunc(entries []string, checked *semantics.CheckedProgram) (IRFunc, error) {
	return buildruntime.BuildActorDispatchFunc(entries, checked)
}

func buildActorMainEntryIDFunc(mainName string) (IRFunc, error) {
	return buildruntime.BuildActorMainEntryIDFunc(mainName)
}
