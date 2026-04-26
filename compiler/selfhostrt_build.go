package compiler

import (
	"fmt"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/module"
	"tetra_language/compiler/internal/semantics"
)

func embeddedSelfHostActorsRuntimeSource(target string) ([]byte, string, error) {
	switch target {
	case "linux-x64", "macos-x64":
		return embeddedActorsSysV, "<embedded selfhostrt actors_sysv>", nil
	case "windows-x64":
		return embeddedActorsWin64, "<embedded selfhostrt actors_win64>", nil
	default:
		return nil, "", fmt.Errorf("self-host runtime not available for target %s", target)
	}
}

func buildEmbeddedSelfHostActorsRuntimeObject(
	target string,
	codegen func([]IRFunc, [][]byte) (*Object, error),
) (*Object, error) {
	src, filename, err := embeddedSelfHostActorsRuntimeSource(target)
	if err != nil {
		return nil, err
	}
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

	funcs, err := LowerModule(checked, world.EntryModule)
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
