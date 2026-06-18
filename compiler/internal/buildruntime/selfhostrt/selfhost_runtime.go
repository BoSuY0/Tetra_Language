package selfhostrt

import (
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/lower"
	"tetra_language/compiler/internal/module"
	"tetra_language/compiler/internal/semantics"
)

func BuildEmbeddedSelfHostRuntimeObject(
	target string,
	src []byte,
	filename string,
	codegen func([]ir.IRFunc, [][]byte) (*tobj.Object, error),
) (*tobj.Object, error) {
	file, err := frontend.ParseFile(src, filename)
	if err != nil {
		return nil, err
	}
	file.Path = filename

	world := &module.World{
		EntryPath:   filename,
		EntryModule: file.Module,
		Root:        "<embedded>",
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{file.Module: file},
	}

	checked, err := semantics.CheckWorldOpt(world, semantics.CheckOptions{RequireMain: false})
	if err != nil {
		return nil, err
	}

	funcs, err := lower.LowerModule(checked, world.EntryModule)
	if err != nil {
		return nil, err
	}

	dataPrefix := checked.GlobalDataByModule[world.EntryModule]
	obj, err := codegen(funcs, dataPrefix)
	if err != nil {
		return nil, err
	}
	obj.Target = target
	obj.Module = "__selfhostrt"
	return obj, nil
}
