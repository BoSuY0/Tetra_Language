package compiler

import (
	"fmt"
	"tetra_language/compiler/internal/buildlink"
	"tetra_language/compiler/internal/buildplan"
	"tetra_language/compiler/internal/semantics"
)

func loadWorldForBuild(inputPath string, opt BuildOptions) (*World, error) {
	if opt.ProjectRoot == "" && len(opt.SourceRoots) == 0 && len(opt.DependencyRoots) == 0 {
		return LoadWorld(inputPath)
	}
	return LoadWorldOpt(inputPath, WorldOptions{
		Root:            opt.ProjectRoot,
		SourceRoots:     opt.SourceRoots,
		DependencyRoots: opt.DependencyRoots,
	})
}

func rejectInterfaceModulesForCodegen(world *World) error {
	modules := sortedInterfaceModules(world)
	if len(modules) == 0 {
		return nil
	}
	return fmt.Errorf("interface-only module '%s' cannot be linked; use --interface-only or provide source/object implementation", modules[0])
}

func readLinkObjects(paths []string, target string) ([]linkedObject, error) {
	linked, err := buildlink.ReadLinkObjects(paths, target)
	if err != nil {
		return nil, err
	}
	return rootLinkedObjects(linked), nil
}

func validateLinkedObjectSymbols(current linkedObject, seen map[string]linkedObject) error {
	buildSeen := make(map[string]buildlink.LinkedObject, len(seen))
	for name, linked := range seen {
		buildSeen[name] = buildLinkedObject(linked)
	}
	if err := buildlink.ValidateLinkedObjectSymbols(buildLinkedObject(current), buildSeen); err != nil {
		return err
	}
	for name, linked := range buildSeen {
		if _, exists := seen[name]; !exists {
			seen[name] = rootLinkedObject(linked)
		}
	}
	return nil
}

func validateInterfaceImplementationProviders(world *World, checked *semantics.CheckedProgram, linked []linkedObject) error {
	return buildlink.ValidateInterfaceImplementationProviders(world, checked, buildLinkedObjects(linked))
}

func validateInterfaceImplementationSymbols(world *World, checked *semantics.CheckedProgram, module string, obj *Object, path string) error {
	return buildlink.ValidateInterfaceImplementationSymbols(world, checked, module, obj, path)
}

func unsupportedInterfaceModuleGenericSymbols(world *World, module string) []string {
	return buildlink.UnsupportedInterfaceModuleGenericSymbols(world, module)
}

func expectedInterfaceModuleSymbols(world *World, module string) []string {
	return buildlink.ExpectedInterfaceModuleSymbols(world, module)
}

func qualifyObjectSymbol(module, name string) string {
	return buildlink.QualifyObjectSymbol(module, name)
}

func interfaceOnlyBuildStats(world *World) *BuildStats {
	return &BuildStats{InterfaceModules: sortedInterfaceModules(world)}
}

func sortedInterfaceModules(world *World) []string {
	return buildlink.SortedInterfaceModules(world)
}

func buildLinkedObject(linked linkedObject) buildlink.LinkedObject {
	return buildlink.LinkedObject{
		Path:        linked.path,
		Object:      linked.obj,
		ContentHash: linked.contentHash,
	}
}

func rootLinkedObject(linked buildlink.LinkedObject) linkedObject {
	return linkedObject{
		path:        linked.Path,
		obj:         linked.Object,
		contentHash: linked.ContentHash,
	}
}

func buildLinkedObjects(linked []linkedObject) []buildlink.LinkedObject {
	if len(linked) == 0 {
		return nil
	}
	out := make([]buildlink.LinkedObject, 0, len(linked))
	for _, item := range linked {
		out = append(out, buildLinkedObject(item))
	}
	return out
}

func rootLinkedObjects(linked []buildlink.LinkedObject) []linkedObject {
	if len(linked) == 0 {
		return nil
	}
	out := make([]linkedObject, 0, len(linked))
	for _, item := range linked {
		out = append(out, rootLinkedObject(item))
	}
	return out
}

func linkedObjectObjects(linked []linkedObject) []*Object {
	if len(linked) == 0 {
		return nil
	}
	out := make([]*Object, 0, len(linked))
	for _, item := range linked {
		out = append(out, item.obj)
	}
	return out
}

func buildTagFromOptions(opt BuildOptions, linkedObjects []linkedObject) string {
	return buildplan.BuildTagFromOptions(opt, buildLinkedObjects(linkedObjects))
}
