package compiler

import "tetra_language/compiler/internal/buildruntime"

func buildLinuxX86BasicNetRuntimeObject() *Object {
	return buildruntime.BuildLinuxX86BasicNetRuntimeObject()
}

func appendLinuxX86BasicNetRuntimeObject(rt *Object) error {
	return buildruntime.AppendLinuxX86BasicNetRuntimeObject(rt)
}
