package compiler

import "tetra_language/compiler/internal/buildruntime"

func buildLinuxX32FilesystemRuntimeObject() *Object {
	return buildruntime.BuildLinuxX32FilesystemRuntimeObject()
}

func appendLinuxX32FilesystemRuntimeObject(rt *Object) error {
	return buildruntime.AppendLinuxX32FilesystemRuntimeObject(rt)
}
