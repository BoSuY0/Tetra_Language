package compiler

import "tetra_language/compiler/internal/buildruntime"

func buildLinuxX86FilesystemRuntimeObject() *Object {
	return buildruntime.BuildLinuxX86FilesystemRuntimeObject()
}

func appendLinuxX86FilesystemRuntimeObject(rt *Object) error {
	return buildruntime.AppendLinuxX86FilesystemRuntimeObject(rt)
}
