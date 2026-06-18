package buildplan

import (
	"fmt"
	"sort"
	"strings"
	"tetra_language/compiler/internal/buildapi"
	"tetra_language/compiler/internal/buildlink"
	"tetra_language/compiler/internal/cache"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/module"
	"tetra_language/compiler/internal/semantics"
)

type ModuleBuildJob struct {
	Module  string
	SrcHash [32]byte
	DepHash [32]byte
}

type ModuleBuildPlan struct {
	Modules           []string
	PublicAPIHashes   map[string]string
	BuildTag          string
	ObjectsByModule   map[string]*tobj.Object
	ObjectlessModules map[string]bool
	ToCompile         []ModuleBuildJob
}

func SourceModules(world *module.World) []string {
	if world == nil || world.ByModule == nil {
		return nil
	}
	modules := make([]string, 0, len(world.ByModule))
	for moduleName := range world.ByModule {
		if world.InterfaceModules[moduleName] {
			continue
		}
		modules = append(modules, moduleName)
	}
	sort.Strings(modules)
	return modules
}

func BuildTagFromOptions(opt buildapi.BuildOptions, linkedObjects []buildlink.LinkedObject) string {
	var tags []string
	if opt.IslandsDebug {
		tags = append(tags, "islands-debug")
	}
	if opt.DebugInfo {
		tags = append(tags, "debug-info")
	}
	if opt.ReleaseOptimize {
		tags = append(tags, "release-opt")
	}
	if opt.InterfaceOnly {
		tags = append(tags, "interface-only")
	}
	if opt.EmitRuntimeHeapTelemetry {
		tags = append(tags, "runtime-heap-telemetry")
	}
	if opt.SurfaceHostRequired {
		tags = append(
			tags,
			"surface-host="+opt.SurfaceHostDriver+":"+opt.SurfaceHostProtocol+":"+opt.SurfaceHostSocketPath,
		)
	}
	if len(linkedObjects) > 0 {
		entries := make([]string, 0, len(linkedObjects))
		for _, linked := range linkedObjects {
			moduleName := ""
			if linked.Object != nil {
				moduleName = linked.Object.Module
			}
			entries = append(entries, fmt.Sprintf("%s:%x", moduleName, linked.ContentHash))
		}
		sort.Strings(entries)
		tags = append(tags, "link="+strings.Join(entries, ","))
	}
	return strings.Join(tags, "+")
}

func WithStackAllocationBuildTag(buildTag string) string {
	if buildTag == "" {
		return "alloc-stack-v1"
	}
	return buildTag + "+alloc-stack-v1"
}

func ModuleLocalFunctionSigDeps(moduleName string, sigMap map[string]semantics.FuncSig) []string {
	names := make([]string, 0)
	for name := range sigMap {
		if cache.ModuleOf(name) == moduleName {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func ModuleLocalTypeSigDeps(moduleName string, typeSigMap map[string]string) []string {
	names := make([]string, 0)
	for name := range typeSigMap {
		if cache.ModuleOf(name) == moduleName {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func SortStats(stats *buildapi.BuildStats) {
	if stats == nil {
		return
	}
	sort.Strings(stats.CacheHits)
	sort.Strings(stats.CompiledModules)
	sort.Strings(stats.LoweredModules)
}

func ObjectsFromModulePlan(plan ModuleBuildPlan) ([]*tobj.Object, error) {
	objects := make([]*tobj.Object, 0, len(plan.Modules))
	for _, moduleName := range plan.Modules {
		obj := plan.ObjectsByModule[moduleName]
		if obj == nil {
			if plan.ObjectlessModules[moduleName] {
				continue
			}
			return nil, fmt.Errorf("missing object for module '%s'", moduleName)
		}
		objects = append(objects, obj)
	}
	return objects, nil
}
