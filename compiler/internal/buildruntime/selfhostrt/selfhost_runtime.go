package selfhostrt

import (
	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/lower"
	"tetra_language/compiler/internal/memorypipeline"
	"tetra_language/compiler/internal/module"
	"tetra_language/compiler/internal/semantics"
	"tetra_language/compiler/internal/validation"
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

	state, err := memorypipeline.Build(checked, memorypipeline.Options{
		Target:    target,
		AllocPlan: selfHostAllocPlanOptions(target),
	})
	if err != nil {
		return nil, err
	}
	lowering, err := lower.LowerPlannedProgram(
		checked,
		state.Plan,
		selfHostLowerOptions(target),
	)
	if err != nil {
		return nil, err
	}
	if err := state.ApplyLowering(lowering.Program, lowering.Evidence); err != nil {
		return nil, err
	}
	if err := validation.ValidateAllocationLowering(state.Plan, lowering.Program); err != nil {
		return nil, err
	}
	funcs, err := lowering.ModuleFuncs(world.EntryModule)
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

func selfHostAllocPlanOptions(target string) allocplan.Options {
	return allocplan.Options{
		EnableStackLowering:    selfHostTargetSupportsStackAllocationLowering(target),
		EnableSmallHeapRuntime: target == "linux-x64",
		EnableRegionPlanning:   target == "linux-x64",
		EnableRegionLowering:   target == "linux-x64",
	}
}

func selfHostLowerOptions(target string) lower.Options {
	return lower.Options{
		StackAllocationLowering:    selfHostTargetSupportsStackAllocationLowering(target),
		FunctionTempRegionLowering: target == "linux-x64",
	}
}

func selfHostTargetSupportsStackAllocationLowering(target string) bool {
	switch target {
	case "linux-x64", "macos-x64", "windows-x64":
		return true
	default:
		return false
	}
}
