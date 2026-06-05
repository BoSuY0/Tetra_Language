package compiler

import (
	"fmt"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/module"
	"tetra_language/compiler/internal/semantics"
)

func embeddedSelfHostActorsRuntimeSource(target string) ([]byte, string, error) {
	switch target {
	case "linux-x64", "macos-x64", "linux-x32":
		return embeddedActorsSysV, "<embedded selfhostrt actors_sysv>", nil
	case "linux-x86":
		return embeddedActorsI386, "<embedded selfhostrt actors_i386>", nil
	case "windows-x64":
		return embeddedActorsWin64, "<embedded selfhostrt actors_win64>", nil
	default:
		return nil, "", fmt.Errorf("self-host runtime not available for target %s", target)
	}
}

func embeddedSelfHostTimeRuntimeSource(target string) ([]byte, string, error) {
	switch target {
	case "linux-x86":
		return embeddedTimeILP32, "<embedded selfhostrt time_ilp32>", nil
	default:
		return nil, "", fmt.Errorf("self-host time runtime not available for target %s", target)
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
	return buildEmbeddedSelfHostRuntimeObject(target, src, filename, codegen)
}

func buildEmbeddedSelfHostTimeRuntimeObject(
	target string,
	codegen func([]IRFunc, [][]byte) (*Object, error),
) (*Object, error) {
	src, filename, err := embeddedSelfHostTimeRuntimeSource(target)
	if err != nil {
		return nil, err
	}
	return buildEmbeddedSelfHostRuntimeObject(target, src, filename, codegen)
}

func buildEmbeddedSelfHostRuntimeObject(
	target string,
	src []byte,
	filename string,
	codegen func([]IRFunc, [][]byte) (*Object, error),
) (*Object, error) {
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
