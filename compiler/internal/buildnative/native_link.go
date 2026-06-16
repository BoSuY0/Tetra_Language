package buildnative

import (
	"fmt"

	"tetra_language/compiler/internal/format/tobj"
)

func AppendLinkedObjects(objects []*tobj.Object, linked []*tobj.Object) []*tobj.Object {
	for _, obj := range linked {
		objects = append(objects, obj)
	}
	return objects
}

func LinkExecutable(outputPath string, target string, backend ExecutableBackend, objects []*tobj.Object, mainName string) error {
	if backend.Link == nil {
		return fmt.Errorf("target backend has no linker: %s", target)
	}
	return backend.Link(outputPath, objects, mainName)
}
