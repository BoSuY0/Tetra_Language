package compiler

import "tetra_language/compiler/internal/buildruntime"

func buildLinuxX32BasicNetRuntimeObject() *Object {
	return buildruntime.BuildLinuxX32BasicNetRuntimeObject()
}

func appendLinuxX32BasicNetRuntimeObject(rt *Object) error {
	return buildruntime.AppendLinuxX32BasicNetRuntimeObject(rt)
}
