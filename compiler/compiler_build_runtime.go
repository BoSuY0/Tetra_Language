package compiler

import (
	"crypto/sha256"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"tetra_language/compiler/internal/abisuite"
	"tetra_language/compiler/internal/actorsrt"
	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/backend/linux_x32"
	"tetra_language/compiler/internal/backend/linux_x64"
	"tetra_language/compiler/internal/backend/linux_x86"
	"tetra_language/compiler/internal/backend/macos_x64"
	"tetra_language/compiler/internal/backend/wasm32_wasi"
	"tetra_language/compiler/internal/backend/wasm32_web"
	"tetra_language/compiler/internal/backend/windows_x64"
	buildapi "tetra_language/compiler/internal/buildapi"
	"tetra_language/compiler/internal/buildlink"
	"tetra_language/compiler/internal/buildnative"
	"tetra_language/compiler/internal/buildplan"
	"tetra_language/compiler/internal/buildruntime"
	"tetra_language/compiler/internal/buildwasm"
	"tetra_language/compiler/internal/cache"
	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
	"tetra_language/compiler/internal/version"
	ctarget "tetra_language/compiler/target"
)

// ---- buildapi_aliases.go ----

type EmitMode = buildapi.EmitMode

const (
	EmitExe     = buildapi.EmitExe
	EmitObject  = buildapi.EmitObject
	EmitLibrary = buildapi.EmitLibrary
)

type RuntimeMode = buildapi.RuntimeMode

const (
	RuntimeAuto     = buildapi.RuntimeAuto
	RuntimeSelfHost = buildapi.RuntimeSelfHost
	RuntimeBuiltin  = buildapi.RuntimeBuiltin
)

type BuildOptions = buildapi.BuildOptions

type BuildStats = buildapi.BuildStats

// ---- compiler_actor_dispatch.go ----

func buildActorDispatchFunc(entries []string, checked *semantics.CheckedProgram) (IRFunc, error) {
	return buildruntime.BuildActorDispatchFunc(entries, checked)
}

func buildActorMainEntryIDFunc(mainName string) (IRFunc, error) {
	return buildruntime.BuildActorMainEntryIDFunc(mainName)
}

// ---- compiler_actor_usage.go ----

func collectActorEntries(checked *semantics.CheckedProgram) (bool, []string, int, error) {
	return buildruntime.CollectActorEntries(checked)
}

func collectActorStateRuntimeUsage(checked *semantics.CheckedProgram) bool {
	return buildruntime.CollectActorStateRuntimeUsage(checked)
}

func collectActorStateRuntimeUsagePosition(
	checked *semantics.CheckedProgram,
) (bool, frontend.Position) {
	return buildruntime.CollectActorStateRuntimeUsagePosition(checked)
}

func collectActorRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	return buildruntime.CollectActorRuntimeUsagePosition(checked)
}

func collectActorSystemReceiveRuntimeUsagePosition(
	checked *semantics.CheckedProgram,
) (bool, frontend.Position) {
	return buildruntime.CollectActorSystemReceiveRuntimeUsagePosition(checked)
}

func collectTaskRuntimeUsage(checked *semantics.CheckedProgram) bool {
	return buildruntime.CollectTaskRuntimeUsage(checked)
}

func collectTaskRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	return buildruntime.CollectTaskRuntimeUsagePosition(checked)
}

func collectTaskGroupRuntimeUsage(checked *semantics.CheckedProgram) bool {
	return buildruntime.CollectTaskGroupRuntimeUsage(checked)
}

func collectTypedTaskRuntimeUsage(checked *semantics.CheckedProgram) (bool, int) {
	return buildruntime.CollectTypedTaskRuntimeUsage(checked)
}

// ---- compiler_link_objects.go ----

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
	return fmt.Errorf(
		("interface-only module '%s' cannot be linked; use --interface-" +
			"only or provide source/object implementation"),
		modules[0],
	)
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
	if err := buildlink.ValidateLinkedObjectSymbols(
		buildLinkedObject(current),
		buildSeen,
	); err != nil {
		return err
	}
	for name, linked := range buildSeen {
		if _, exists := seen[name]; !exists {
			seen[name] = rootLinkedObject(linked)
		}
	}
	return nil
}

func validateInterfaceImplementationProviders(
	world *World,
	checked *semantics.CheckedProgram,
	linked []linkedObject,
) error {
	return buildlink.ValidateInterfaceImplementationProviders(
		world,
		checked,
		buildLinkedObjects(linked),
	)
}

func validateInterfaceImplementationSymbols(
	world *World,
	checked *semantics.CheckedProgram,
	module string,
	obj *Object,
	path string,
) error {
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

// ---- compiler_native_link.go ----

func linkNativeExecutable(
	outputPath string,
	native nativeBuildTarget,
	opt BuildOptions,
	checked *semantics.CheckedProgram,
	objects []*Object,
	linkedObjects []linkedObject,
) error {
	actorsUsed, actorEntries, actorSpawnCount, err := collectActorEntries(checked)
	if err != nil {
		return err
	}
	actorStateUsed, actorStatePos := collectActorStateRuntimeUsagePosition(checked)
	actorRuntimeUsed, actorRuntimePos := collectActorRuntimeUsagePosition(checked)
	actorSystemReceiveUsed, actorSystemReceivePos := collectActorSystemReceiveRuntimeUsagePosition(
		checked,
	)
	tasksUsed, tasksPos := collectTaskRuntimeUsagePosition(checked)
	taskGroupsUsed := collectTaskGroupRuntimeUsage(checked)
	typedTasksUsed, typedTaskMaxSlots := collectTypedTaskRuntimeUsage(checked)
	timeRuntimeUsed, timeRuntimePos := collectTimeRuntimeUsagePosition(checked)
	filesystemRuntimeUsed, filesystemRuntimePos := collectFilesystemRuntimeUsagePosition(checked)
	netRuntimeUsage := collectNetRuntimeUsageProfile(checked)
	netRuntimeUsed := netRuntimeUsage.used
	surfaceRuntimeUsed, surfaceRuntimePos := collectSurfaceRuntimeUsagePosition(checked)
	distributedActorsUsed, distributedActorsPos := collectDistributedActorRuntimeUsagePosition(
		checked,
	)
	runtimeCaps := nativeRuntimeCapabilitiesForTarget(native.triple)
	runtimeObjectPlan := buildruntime.DecideRuntimeObjectPlan(
		native.triple,
		opt.RuntimeObjectPath != "",
		buildruntime.CapabilitiesForTarget(native.triple),
		buildruntime.RuntimeObjectPlanUsage{
			ActorsUsed:             actorsUsed,
			ActorRuntimeUsed:       actorRuntimeUsed,
			ActorSystemReceiveUsed: actorSystemReceiveUsed,
			ActorStateUsed:         actorStateUsed,
			TasksUsed:              tasksUsed,
			TaskGroupsUsed:         taskGroupsUsed,
			TypedTasksUsed:         typedTasksUsed,
			TimeRuntimeUsed:        timeRuntimeUsed,
			FilesystemRuntimeUsed:  filesystemRuntimeUsed,
			NetRuntimeUsed:         netRuntimeUsed,
			NetRuntimeSupported:    targetSupportsNetRuntimeUsage(native.triple, netRuntimeUsage),
			SurfaceRuntimeUsed:     surfaceRuntimeUsed,
			DistributedActorsUsed:  distributedActorsUsed,
		},
	)
	timeOnlyRuntime := runtimeObjectPlan.TimeOnlyRuntime
	linuxMinimalRuntime := runtimeObjectPlan.LinuxMinimalRuntime
	if netRuntimeUsed {
		if pos, unsupported := unsupportedNetRuntimeUsagePosition(
			native.triple,
			netRuntimeUsage,
		); unsupported {
			return targetRuntimeDiagnostic(pos, native.triple, "networking")
		}
	}
	if filesystemRuntimeUsed && !runtimeCaps.filesystem {
		return targetRuntimeDiagnostic(filesystemRuntimePos, native.triple, "filesystem")
	}
	if surfaceRuntimeUsed && !runtimeCaps.surface {
		return targetRuntimeDiagnostic(surfaceRuntimePos, native.triple, "surface")
	}
	if opt.SurfaceHostRequired {
		if native.triple != "linux-x64" {
			return fmt.Errorf(
				"surface host runtime requires linux-x64 target, got %s",
				native.triple,
			)
		}
		if strings.TrimSpace(opt.SurfaceHostDriver) != "wayland" {
			return fmt.Errorf(
				"surface host runtime requires backend wayland, got %q",
				opt.SurfaceHostDriver,
			)
		}
		if strings.TrimSpace(opt.SurfaceHostProtocol) != "tetra.surface.host-ipc.v1" {
			return fmt.Errorf(
				"surface host runtime requires protocol tetra.surface.host-ipc.v1, got %q",
				opt.SurfaceHostProtocol,
			)
		}
		if strings.TrimSpace(opt.SurfaceHostSocketPath) == "" {
			return fmt.Errorf("surface host runtime requires socket path")
		}
	}
	if distributedActorsUsed && !runtimeCaps.distributedActors {
		return targetRuntimeDiagnostic(distributedActorsPos, native.triple, "distributed actors")
	}
	if timeRuntimeUsed && !runtimeCaps.time && !timeOnlyRuntime {
		return targetRuntimeDiagnostic(timeRuntimePos, native.triple, "time")
	}
	if tasksUsed && !runtimeCaps.tasks {
		return targetRuntimeDiagnostic(tasksPos, native.triple, "task")
	}
	if actorRuntimeUsed && !runtimeCaps.actors {
		return targetRuntimeDiagnostic(actorRuntimePos, native.triple, "actors")
	}
	if actorSystemReceiveUsed && !runtimeCaps.actors {
		return targetRuntimeDiagnostic(actorSystemReceivePos, native.triple, "actor system-message receive")
	}
	if actorStateUsed && !runtimeCaps.actorState {
		return targetRuntimeDiagnostic(actorStatePos, native.triple, "actors")
	}
	if runtimeCaps.actors && runtimeCaps.maxActorSpawns != unlimitedActorSpawns &&
		actorSpawnCount > runtimeCaps.maxActorSpawns {
		return targetRuntimeDiagnostic(
			actorRuntimePos,
			native.triple,
			fmt.Sprintf("actor fanout above %d", runtimeCaps.maxActorSpawns),
		)
	}
	if taskGroupsUsed && !runtimeCaps.taskGroups {
		return targetRuntimeDiagnostic(tasksPos, native.triple, "task group")
	}
	if typedTasksUsed && !runtimeCaps.typedTasks {
		return targetRuntimeDiagnostic(tasksPos, native.triple, "typed task")
	}
	if typedTasksUsed && runtimeCaps.maxTypedTaskSlots > 0 &&
		typedTaskMaxSlots > runtimeCaps.maxTypedTaskSlots {
		return targetRuntimeDiagnostic(tasksPos, native.triple, "staged typed task")
	}
	runtimeUsed := runtimeObjectPlan.RuntimeUsed
	if runtimeUsed && len(actorEntries) == 0 {
		actorEntries = []string{checked.MainName}
	}
	mainName := checked.MainName
	if opt.RuntimeObjectPath != "" && !runtimeUsed {
		return fmt.Errorf(
			("runtime object override requires runtime usage (no actor/task/" +
				"time/filesystem/networking/surface/distributed actor builtins found)"),
		)
	}
	if runtimeUsed {
		runtimeObjectHandled := false
		if timeOnlyRuntime {
			rt, err := buildEmbeddedSelfHostTimeRuntimeObject(native.triple, native.codegen)
			if err != nil {
				return err
			}
			annotateRuntimeObjectSignatures(rt)
			if err := validateTimeRuntimeObject(rt); err != nil {
				return err
			}
			rt.Target = native.triple
			rt.Module = "__selfhosttime"
			objects = append(objects, rt)
			runtimeObjectHandled = true
		}
		if linuxMinimalRuntime {
			var rt *Object
			switch native.triple {
			case "linux-x86":
				if filesystemRuntimeUsed {
					rt = buildLinuxX86FilesystemRuntimeObject()
				} else {
					rt = buildLinuxX86BasicNetRuntimeObject()
				}
				if filesystemRuntimeUsed && netRuntimeUsed {
					if err := appendLinuxX86BasicNetRuntimeObject(rt); err != nil {
						return err
					}
				}
			case "linux-x32":
				if filesystemRuntimeUsed {
					rt = buildLinuxX32FilesystemRuntimeObject()
				} else {
					rt = buildLinuxX32BasicNetRuntimeObject()
				}
				if filesystemRuntimeUsed && netRuntimeUsed {
					if err := appendLinuxX32BasicNetRuntimeObject(rt); err != nil {
						return err
					}
				}
			}
			annotateRuntimeObjectSignatures(rt)
			if filesystemRuntimeUsed {
				if err := validateFilesystemRuntimeObject(rt); err != nil {
					return err
				}
			}
			if netRuntimeUsed {
				if runtimeCaps.networking {
					err = validateNetRuntimeObject(rt)
				} else {
					err = validateNetRuntimeObjectForUsage(rt, netRuntimeUsage)
				}
				if err != nil {
					return err
				}
			}
			rt.Target = native.triple
			if filesystemRuntimeUsed && netRuntimeUsed {
				if native.triple == "linux-x86" {
					rt.Module = "__linux_x86_minrt"
				} else {
					rt.Module = "__linux_x32_minrt"
				}
			}
			objects = append(objects, rt)
			runtimeObjectHandled = true
		}
		if !runtimeObjectHandled {
			usage := runtimeUsageProfile{
				actorStateUsed:         actorStateUsed,
				actorSystemReceiveUsed: actorSystemReceiveUsed,
				tasksUsed:              tasksUsed,
				taskGroupsUsed:         taskGroupsUsed,
				typedTasksUsed:         typedTasksUsed,
				typedTaskMaxSlots:      typedTaskMaxSlots,
				timeRuntimeUsed:        timeRuntimeUsed,
				filesystemUsed:         filesystemRuntimeUsed,
				netUsed:                netRuntimeUsed,
				netRuntimeSymbols:      netRuntimeUsage.requiredSymbols(),
				surfaceUsed:            surfaceRuntimeUsed,
				distributedActorsUsed:  distributedActorsUsed,
				actorSpawnCount:        actorSpawnCount,
			}
			runtimeMode, err := selectRuntimeModeForNativeTarget(native.triple, opt.Runtime, usage)
			if err != nil {
				return err
			}
			if native.triple == "linux-x32" && opt.RuntimeObjectPath == "" &&
				runtimeMode == RuntimeBuiltin {
				return fmt.Errorf(
					("builtin runtime is not supported on target linux-x32; use " +
						"runtime=selfhost for supported self-host runtime builds or remove " +
						"runtime builtins"),
				)
			}
			var rt *Object
			if opt.RuntimeObjectPath != "" {
				rt, err = ReadObject(opt.RuntimeObjectPath)
				if err != nil {
					return fmt.Errorf("read runtime object: %w", err)
				}
				if rt.Target == "" {
					return fmt.Errorf("runtime object has no target: %s", opt.RuntimeObjectPath)
				}
				if rt.Target != native.triple {
					return fmt.Errorf(
						"runtime object target mismatch: got=%s want=%s",
						rt.Target,
						native.triple,
					)
				}
			} else {
				switch runtimeMode {
				case RuntimeSelfHost:
					rt, err = buildEmbeddedSelfHostActorsRuntimeObject(native.triple, native.codegen)
				case RuntimeBuiltin:
					if native.backend.actorRuntime == nil {
						return fmt.Errorf("actors runtime is not supported on target %s", native.triple)
					}
					if opt.SurfaceHostRequired {
						rt, err = actorsrt.BuildLinuxX64WithSurfaceHostIPC(actorEntries, actorsrt.SurfaceHostIPCOptions{
							SocketPath: opt.SurfaceHostSocketPath,
						})
					} else {
						rt, err = native.backend.actorRuntime(actorEntries)
					}
				}
				if err != nil {
					return err
				}
				annotateRuntimeObjectSignatures(rt)
				if native.triple == "linux-x86" && filesystemRuntimeUsed {
					if err := appendLinuxX86FilesystemRuntimeObject(rt); err != nil {
						return err
					}
				}
				if native.triple == "linux-x32" && filesystemRuntimeUsed {
					if err := appendLinuxX32FilesystemRuntimeObject(rt); err != nil {
						return err
					}
				}
				if native.triple == "linux-x86" && netRuntimeUsed {
					if err := appendLinuxX86BasicNetRuntimeObject(rt); err != nil {
						return err
					}
				}
				if native.triple == "linux-x32" && netRuntimeUsed {
					if err := appendLinuxX32BasicNetRuntimeObject(rt); err != nil {
						return err
					}
				}
			}
			if err := validateActorRuntimeObject(rt); err != nil {
				return err
			}
			if actorSystemReceiveUsed {
				if err := validateActorSystemReceiveRuntimeObject(rt); err != nil {
					return err
				}
			}
			if opt.RuntimeHeapTelemetryActorDomains {
				if err := validateActorTelemetryRuntimeObject(rt); err != nil {
					return err
				}
			}
			if actorStateUsed {
				if err := validateActorStateRuntimeObject(rt); err != nil {
					return err
				}
			}
			if tasksUsed {
				if err := validateTaskRuntimeObject(rt); err != nil {
					return err
				}
			}
			if taskGroupsUsed {
				if err := validateTaskGroupRuntimeObject(rt); err != nil {
					return err
				}
			}
			if typedTasksUsed {
				if err := validateTypedTaskRuntimeObject(rt, typedTaskMaxSlots); err != nil {
					return err
				}
			}
			if timeRuntimeUsed {
				if err := validateTimeRuntimeObject(rt); err != nil {
					return err
				}
			}
			if filesystemRuntimeUsed {
				if err := validateFilesystemRuntimeObject(rt); err != nil {
					return err
				}
			}
			if netRuntimeUsed {
				if runtimeCaps.networking {
					if err := validateNetRuntimeObject(rt); err != nil {
						return err
					}
				} else if err := validateNetRuntimeObjectForUsage(rt, netRuntimeUsage); err != nil {
					return err
				}
			}
			if surfaceRuntimeUsed {
				if err := validateSurfaceRuntimeObject(rt); err != nil {
					return err
				}
			}
			if distributedActorsUsed {
				if err := validateDistributedActorRuntimeObject(rt); err != nil {
					return err
				}
			}

			glueObj, builtGlue, err := buildruntime.BuildActorGlueObject(
				rt,
				native.triple,
				actorEntries,
				checked,
				native.codegen,
			)
			if err != nil {
				return err
			}
			if builtGlue {
				objects = append(objects, glueObj)
			}
			rt.Target = native.triple
			switch {
			case opt.RuntimeObjectPath != "":
				rt.Module = "__runtime"
			case runtimeMode == RuntimeBuiltin:
				rt.Module = "__actorsrt"
			default:
				rt.Module = "__selfhostrt"
			}
			objects = append(objects, rt)
			mainName = "__tetra_entry"
		}
	}

	objects = buildnative.AppendLinkedObjects(objects, linkedObjectObjects(linkedObjects))
	return buildnative.LinkExecutable(
		outputPath,
		native.triple,
		buildNativeExecutableBackend(native.backend),
		objects,
		mainName,
	)
}

// ---- compiler_runtime_caps.go ----

type runtimeUsageProfile struct {
	actorSystemReceiveUsed bool
	actorStateUsed         bool
	tasksUsed              bool
	taskGroupsUsed         bool
	typedTasksUsed         bool
	typedTaskMaxSlots      int
	timeRuntimeUsed        bool
	filesystemUsed         bool
	netUsed                bool
	netRuntimeSymbols      []string
	surfaceUsed            bool
	distributedActorsUsed  bool
	actorSpawnCount        int
}

func (u runtimeUsageProfile) buildRuntimeUsage() buildruntime.UsageProfile {
	return buildruntime.UsageProfile{
		ActorSystemReceiveUsed: u.actorSystemReceiveUsed,
		ActorStateUsed:         u.actorStateUsed,
		TasksUsed:              u.tasksUsed,
		TaskGroupsUsed:         u.taskGroupsUsed,
		TypedTasksUsed:         u.typedTasksUsed,
		TypedTaskMaxSlots:      u.typedTaskMaxSlots,
		TimeRuntimeUsed:        u.timeRuntimeUsed,
		FilesystemUsed:         u.filesystemUsed,
		NetUsed:                u.netUsed,
		NetRuntimeSymbols:      u.netRuntimeSymbols,
		SurfaceUsed:            u.surfaceUsed,
		DistributedActorsUsed:  u.distributedActorsUsed,
		ActorSpawnCount:        u.actorSpawnCount,
	}
}

const unlimitedActorSpawns = buildruntime.UnlimitedActorSpawns

type nativeRuntimeCapabilities struct {
	actors                   bool
	actorState               bool
	tasks                    bool
	taskGroups               bool
	typedTasks               bool
	time                     bool
	timeOnlyWithoutScheduler bool
	filesystem               bool
	networking               bool
	surface                  bool
	distributedActors        bool
	maxActorSpawns           int
	maxTypedTaskSlots        int
	builtinRuntime           bool
	selfHostActorsRuntime    bool
	selfHostTimeRuntime      bool
}

func nativeRuntimeCapabilitiesForTarget(target string) nativeRuntimeCapabilities {
	caps := buildruntime.CapabilitiesForTarget(target)
	return nativeRuntimeCapabilities{
		actors:                   caps.Actors,
		actorState:               caps.ActorState,
		tasks:                    caps.Tasks,
		taskGroups:               caps.TaskGroups,
		typedTasks:               caps.TypedTasks,
		time:                     caps.Time,
		timeOnlyWithoutScheduler: caps.TimeOnlyWithoutScheduler,
		filesystem:               caps.Filesystem,
		networking:               caps.Networking,
		surface:                  caps.Surface,
		distributedActors:        caps.DistributedActors,
		maxActorSpawns:           caps.MaxActorSpawns,
		maxTypedTaskSlots:        caps.MaxTypedTaskSlots,
		builtinRuntime:           caps.BuiltinRuntime,
		selfHostActorsRuntime:    caps.SelfHostActorsRuntime,
		selfHostTimeRuntime:      caps.SelfHostTimeRuntime,
	}
}

func allocationPlanForIRFuncs(plan *allocplan.Plan, funcs []IRFunc) *allocplan.Plan {
	if plan == nil {
		return nil
	}
	names := map[string]bool{}
	for _, fn := range funcs {
		names[fn.Name] = true
	}
	out := &allocplan.Plan{Totals: plan.Totals}
	for _, fn := range plan.Functions {
		if names[fn.Name] {
			out.Functions = append(out.Functions, fn)
		}
	}
	return out
}

func selectRuntimeMode(requested RuntimeMode, usage runtimeUsageProfile) (RuntimeMode, error) {
	return buildruntime.SelectRuntimeMode(requested, usage.buildRuntimeUsage())
}

func runtimeModeForNativeTarget(
	target string,
	requested RuntimeMode,
	selected RuntimeMode,
	usage runtimeUsageProfile,
) (RuntimeMode, error) {
	return buildruntime.RuntimeModeForNativeTarget(
		target,
		requested,
		selected,
		usage.buildRuntimeUsage(),
	)
}

func selectRuntimeModeForNativeTarget(
	target string,
	requested RuntimeMode,
	usage runtimeUsageProfile,
) (RuntimeMode, error) {
	return buildruntime.SelectRuntimeModeForNativeTarget(
		target,
		requested,
		usage.buildRuntimeUsage(),
	)
}

func selfHostRuntimeSupportsNativeUsage(target string, usage runtimeUsageProfile) bool {
	return buildruntime.SelfHostRuntimeSupportsNativeUsage(target, usage.buildRuntimeUsage())
}

func requiredActorRuntimeSymbols() []string {
	return buildruntime.RequiredActorRuntimeSymbols()
}

func requiredActorStateRuntimeSymbols() []string {
	return buildruntime.RequiredActorStateRuntimeSymbols()
}

func requiredDistributedActorRuntimeSymbols() []string {
	return buildruntime.RequiredDistributedActorRuntimeSymbols()
}

func requiredTaskRuntimeSymbols() []string {
	return buildruntime.RequiredTaskRuntimeSymbols()
}

func requiredTaskGroupRuntimeSymbols() []string {
	return buildruntime.RequiredTaskGroupRuntimeSymbols()
}

func requiredTypedTaskRuntimeSymbols(maxSlots int) []string {
	return buildruntime.RequiredTypedTaskRuntimeSymbols(maxSlots)
}

func requiredTimeRuntimeSymbols() []string {
	return buildruntime.RequiredTimeRuntimeSymbols()
}

func requiredFilesystemRuntimeSymbols() []string {
	return buildruntime.RequiredFilesystemRuntimeSymbols()
}

func requiredNetRuntimeSymbols() []string {
	return buildruntime.RequiredNetRuntimeSymbols()
}

func targetSupportsNetRuntimeUsage(target string, usage netRuntimeUsageProfile) bool {
	return buildruntime.TargetSupportsNetRuntimeSymbols(target, usage.requiredSymbols())
}

func targetSupportsNetRuntimeSymbols(target string, symbols []string) bool {
	return buildruntime.TargetSupportsNetRuntimeSymbols(target, symbols)
}

func unsupportedNetRuntimeUsagePosition(
	target string,
	usage netRuntimeUsageProfile,
) (frontend.Position, bool) {
	supported := supportedNetRuntimeSymbolsForTarget(target)
	for _, symbol := range usage.requiredSymbols() {
		if _, ok := supported[symbol]; ok {
			continue
		}
		if pos, ok := usage.symbolPositions[symbol]; ok {
			return pos, true
		}
		return usage.firstPos, true
	}
	return frontend.Position{}, false
}

func supportedNetRuntimeSymbolsForTarget(target string) map[string]struct{} {
	return buildruntime.SupportedNetRuntimeSymbolsForTarget(target)
}

func requiredSurfaceRuntimeSymbols() []string {
	return buildruntime.RequiredSurfaceRuntimeSymbols()
}

type runtimeObjectSlotSignature struct {
	paramSlots  int
	returnSlots int
}

func runtimeObjectSignature(name string) (runtimeObjectSlotSignature, bool) {
	sig, ok := buildruntime.RuntimeObjectSignature(name)
	if !ok {
		return runtimeObjectSlotSignature{}, false
	}
	return runtimeObjectSlotSignature{
		paramSlots:  sig.ParamSlots,
		returnSlots: sig.ReturnSlots,
	}, true
}

func annotateRuntimeObjectSignatures(rt *Object) {
	buildruntime.AnnotateRuntimeObjectSignatures(rt)
}

func validateRuntimeObjectSymbols(rt *Object, missingObject string, required []string) error {
	return buildruntime.ValidateRuntimeObjectSymbols(rt, missingObject, required)
}

func validateActorRuntimeObject(rt *Object) error {
	return buildruntime.ValidateActorRuntimeObject(rt)
}

func validateActorSystemReceiveRuntimeObject(rt *Object) error {
	return buildruntime.ValidateActorSystemReceiveRuntimeObject(rt)
}

func validateActorTelemetryRuntimeObject(rt *Object) error {
	return buildruntime.ValidateActorTelemetryRuntimeObject(rt)
}

func validateActorStateRuntimeObject(rt *Object) error {
	return buildruntime.ValidateActorStateRuntimeObject(rt)
}

func validateDistributedActorRuntimeObject(rt *Object) error {
	return buildruntime.ValidateDistributedActorRuntimeObject(rt)
}

func validateTimeRuntimeObject(rt *Object) error {
	return buildruntime.ValidateTimeRuntimeObject(rt)
}

func validateFilesystemRuntimeObject(rt *Object) error {
	return buildruntime.ValidateFilesystemRuntimeObject(rt)
}

func validateNetRuntimeObject(rt *Object) error {
	return buildruntime.ValidateNetRuntimeObject(rt)
}

func validateNetRuntimeObjectForUsage(rt *Object, usage netRuntimeUsageProfile) error {
	return buildruntime.ValidateNetRuntimeObjectForSymbols(rt, usage.requiredSymbols())
}

func validateSurfaceRuntimeObject(rt *Object) error {
	return buildruntime.ValidateSurfaceRuntimeObject(rt)
}

func validateTypedTaskRuntimeObject(rt *Object, maxSlots int) error {
	return buildruntime.ValidateTypedTaskRuntimeObject(rt, maxSlots)
}

func validateTaskRuntimeObject(rt *Object) error {
	return buildruntime.ValidateTaskRuntimeObject(rt)
}

func validateTaskGroupRuntimeObject(rt *Object) error {
	return buildruntime.ValidateTaskGroupRuntimeObject(rt)
}

// ---- compiler_runtime_usage.go ----

func validateNativeRuntimeBeforeCodegen(checked *semantics.CheckedProgram, target string) error {
	if checked == nil || target != "linux-x86" {
		return nil
	}
	typedTasksUsed, typedTaskMaxSlots := collectTypedTaskRuntimeUsage(checked)
	if !typedTasksUsed {
		return nil
	}
	_, tasksPos := collectTaskRuntimeUsagePosition(checked)
	caps := nativeRuntimeCapabilitiesForTarget(target)
	if caps.maxTypedTaskSlots > 0 && typedTaskMaxSlots > caps.maxTypedTaskSlots {
		return targetRuntimeDiagnostic(tasksPos, target, "staged typed task")
	}
	return nil
}

func collectDistributedActorRuntimeUsagePosition(
	checked *semantics.CheckedProgram,
) (bool, frontend.Position) {
	return buildruntime.CollectDistributedActorRuntimeUsagePosition(checked)
}

func collectTimeRuntimeUsage(checked *semantics.CheckedProgram) bool {
	return buildruntime.CollectTimeRuntimeUsage(checked)
}

func collectTimeRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	return buildruntime.CollectTimeRuntimeUsagePosition(checked)
}

func collectFilesystemRuntimeUsage(checked *semantics.CheckedProgram) bool {
	return buildruntime.CollectFilesystemRuntimeUsage(checked)
}

func collectFilesystemRuntimeUsagePosition(
	checked *semantics.CheckedProgram,
) (bool, frontend.Position) {
	return buildruntime.CollectFilesystemRuntimeUsagePosition(checked)
}

func collectNetRuntimeUsage(checked *semantics.CheckedProgram) bool {
	return buildruntime.CollectNetRuntimeUsage(checked)
}

func collectNetRuntimeUsagePosition(checked *semantics.CheckedProgram) (bool, frontend.Position) {
	return buildruntime.CollectNetRuntimeUsagePosition(checked)
}

type netRuntimeUsageProfile struct {
	used            bool
	firstPos        frontend.Position
	symbolPositions map[string]frontend.Position
}

func (u netRuntimeUsageProfile) requiredSymbols() []string {
	return u.buildRuntimeProfile().RequiredSymbols()
}

func (u netRuntimeUsageProfile) buildRuntimeProfile() buildruntime.NetRuntimeUsageProfile {
	return buildruntime.NetRuntimeUsageProfile{
		Used:            u.used,
		FirstPos:        u.firstPos,
		SymbolPositions: u.symbolPositions,
	}
}

func collectNetRuntimeUsageProfile(checked *semantics.CheckedProgram) netRuntimeUsageProfile {
	usage := buildruntime.CollectNetRuntimeUsageProfile(checked)
	return netRuntimeUsageProfile{
		used:            usage.Used,
		firstPos:        usage.FirstPos,
		symbolPositions: usage.SymbolPositions,
	}
}

func netRuntimeSymbolForBuiltin(name string) (string, bool) {
	return buildruntime.NetRuntimeSymbolForBuiltin(name)
}

func collectSurfaceRuntimeUsage(checked *semantics.CheckedProgram) bool {
	return buildruntime.CollectSurfaceRuntimeUsage(checked)
}

func collectSurfaceRuntimeUsagePosition(
	checked *semantics.CheckedProgram,
) (bool, frontend.Position) {
	return buildruntime.CollectSurfaceRuntimeUsagePosition(checked)
}

// ---- compiler_wasm_ui.go ----

func buildObjectFileWithStatsOpt(
	inputPath, outputPath string,
	tgt ctarget.Target,
	opt BuildOptions,
) (*BuildStats, error) {
	requireMain := opt.Emit == EmitObject && !opt.InterfaceOnly
	codegenOptions := nativeCodegenOptionsForTarget(tgt, opt)

	world, err := loadWorldForBuild(inputPath, opt)
	if err != nil {
		return nil, err
	}
	if err := validateTargetExportedFFIAST(world, tgt.Triple); err != nil {
		return nil, err
	}
	checked, err := semantics.CheckWorldOpt(
		world,
		semanticsCheckOptionsForTarget(requireMain, tgt.Triple),
	)
	if err != nil {
		return nil, translateTargetExportedFFISemanticError(err, tgt.Triple)
	}
	if opt.InterfaceOnly {
		return interfaceOnlyBuildStats(world), nil
	}
	if err := rejectInterfaceModulesForCodegen(world); err != nil {
		return nil, err
	}
	if err := validateTargetExportedFFIABI(checked, tgt.Triple); err != nil {
		return nil, err
	}

	memoryState, err := buildMemoryStateForTarget(checked, tgt.Triple)
	if err != nil {
		return nil, err
	}
	memoryPlanDigest, err := memoryState.ModulePlanDigest(world.EntryModule)
	if err != nil {
		return nil, err
	}
	loweringResult, err := lowerMemoryStateForBuild(checked, memoryState, tgt.Triple, opt, nil)
	if err != nil {
		return nil, err
	}
	funcs, err := loweringResult.ModuleFuncs(world.EntryModule)
	if err != nil {
		return nil, err
	}
	if err := validateTargetAtomicIR(funcs, tgt); err != nil {
		return nil, err
	}

	var obj *Object
	dataPrefix := checked.GlobalDataByModule[world.EntryModule]
	switch tgt.OS {
	case ctarget.OSLinux:
		switch tgt.Triple {
		case "linux-x86":
			obj, err = linux_x86.CodegenObjectLinuxX86WithOptionsAndDataPrefix(
				funcs,
				dataPrefix,
				codegenOptions,
			)
		case "linux-x64":
			obj, err = linux_x64.CodegenObjectLinuxX64WithOptionsAndDataPrefix(
				funcs,
				dataPrefix,
				codegenOptions,
			)
		case "linux-x32":
			obj, err = linux_x32.CodegenObjectLinuxX32WithOptionsAndDataPrefix(
				funcs,
				dataPrefix,
				codegenOptions,
			)
		default:
			return nil, fmt.Errorf(
				"target backend not implemented: %s (object codegen blocked)",
				tgt.Triple,
			)
		}
	case ctarget.OSWindows:
		obj, err = windows_x64.CodegenObjectWindowsX64WithOptionsAndDataPrefix(
			funcs,
			dataPrefix,
			codegenOptions,
		)
	case ctarget.OSMacOS:
		obj, err = macos_x64.CodegenObjectMacOSX64WithOptionsAndDataPrefix(
			funcs,
			dataPrefix,
			codegenOptions,
		)
	default:
		return nil, fmt.Errorf("unsupported target: %s", tgt.Triple)
	}
	if err != nil {
		return nil, err
	}

	obj.Target = tgt.Triple
	moduleName := world.EntryModule
	if moduleName == "" {
		moduleName = "__entry"
	}
	obj.Module = moduleName
	obj.CompilerVersion = version.CompilerVersion
	obj.MemoryPlanSchema = tobj.MemoryPlanSchemaV2
	obj.MemoryLoweringSchema = tobj.MemoryLoweringSchemaV2
	obj.MemoryPlanDigest = memoryPlanDigest
	obj.MemoryLoweringDigest, err = loweringResult.ModuleLoweringDigest(moduleName)
	if err != nil {
		return nil, err
	}
	file := world.ByModule[world.EntryModule]
	if file != nil {
		obj.SrcHash = sha256.Sum256(file.Src)
		hash, err := InterfaceFingerprintFromSource(file.Src, file.Path)
		if err != nil {
			return nil, err
		}
		obj.PublicAPIHash = hash
	}
	obj.WorldSigHash = cache.WorldSigHash(checked)

	if err := WriteObject(outputPath, obj); err != nil {
		return nil, err
	}
	return &BuildStats{
		CompiledModules: []string{moduleName},
		LoweredModules:  []string{moduleName},
	}, nil
}

func buildWASM32WASIWithStatsOpt(
	inputPath, outputPath string,
	tgt ctarget.Target,
	opt BuildOptions,
) (*BuildStats, error) {
	if tgt.Triple != "wasm32-wasi" {
		return nil, fmt.Errorf("internal error: unexpected target for wasm backend: %s", tgt.Triple)
	}
	if opt.Emit != EmitExe {
		return nil, fmt.Errorf("wasm32-wasi supports only --emit=exe in this wave")
	}
	if opt.RuntimeObjectPath != "" {
		return nil, fmt.Errorf("wasm32-wasi does not support --runtime-object in this wave")
	}
	if len(opt.LinkObjectPaths) > 0 {
		return nil, fmt.Errorf("wasm32-wasi does not support --link-object in this wave")
	}

	world, err := loadWorldForBuild(inputPath, opt)
	if err != nil {
		return nil, err
	}
	checked, err := semantics.CheckWorldOpt(
		world,
		semantics.CheckOptions{RequireMain: !opt.InterfaceOnly},
	)
	if err != nil {
		return nil, err
	}
	if opt.InterfaceOnly {
		return interfaceOnlyBuildStats(world), nil
	}
	if err := rejectInterfaceModulesForCodegen(world); err != nil {
		return nil, err
	}

	modules := make([]string, 0, len(world.ByModule))
	for module := range world.ByModule {
		modules = append(modules, module)
	}
	sort.Strings(modules)

	var funcs []IRFunc
	var dataPrefix [][]byte
	globalOffset := 0
	stats := &BuildStats{
		CompiledModules: make([]string, 0, len(modules)),
		LoweredModules:  make([]string, 0, len(modules)),
	}
	memoryState, err := buildMemoryStateForTarget(checked, tgt.Triple)
	if err != nil {
		return nil, err
	}
	loweringResult, err := lowerMemoryStateForBuild(checked, memoryState, tgt.Triple, opt, nil)
	if err != nil {
		return nil, err
	}
	for _, module := range modules {
		moduleFuncs, err := loweringResult.ModuleFuncs(module)
		if err != nil {
			return nil, err
		}
		stats.LoweredModules = append(stats.LoweredModules, module)
		stats.CompiledModules = append(stats.CompiledModules, module)
		funcs = append(funcs, relocateWASMGlobalSlots(moduleFuncs, globalOffset)...)
		moduleData := checked.GlobalDataByModule[module]
		dataPrefix = append(dataPrefix, moduleData...)
		globalOffset += len(moduleData)
	}
	if err := validateWASMIRPolicy(tgt.Triple, funcs); err != nil {
		return nil, err
	}
	if err := rejectUnsupportedWASMRuntimeBuiltins(funcs, tgt.Triple); err != nil {
		return nil, err
	}

	obj, err := wasm32_wasi.CodegenObjectWithDataPrefix(funcs, checked.MainName, dataPrefix)
	if err != nil {
		return nil, err
	}
	wasmBytes, err := wasm32_wasi.LinkObject(obj)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(outputPath, wasmBytes, 0o755); err != nil {
		return nil, err
	}
	if err := emitUIArtifacts(outputPath, tgt.Triple, checked); err != nil {
		return nil, err
	}
	if err := emitExplainReports(
		outputPath,
		tgt.Triple,
		checked,
		opt,
		memoryState,
		loweringResult,
	); err != nil {
		return nil, err
	}
	return stats, nil
}

func buildWASM32WEBWithStatsOpt(
	inputPath, outputPath string,
	tgt ctarget.Target,
	opt BuildOptions,
) (*BuildStats, error) {
	if tgt.Triple != "wasm32-web" {
		return nil, fmt.Errorf("internal error: unexpected target for wasm backend: %s", tgt.Triple)
	}
	if opt.Emit != EmitExe {
		return nil, fmt.Errorf("wasm32-web supports only --emit=exe in this wave")
	}
	if opt.RuntimeObjectPath != "" {
		return nil, fmt.Errorf("wasm32-web does not support --runtime-object in this wave")
	}
	if len(opt.LinkObjectPaths) > 0 {
		return nil, fmt.Errorf("wasm32-web does not support --link-object in this wave")
	}

	world, err := loadWorldForBuild(inputPath, opt)
	if err != nil {
		return nil, err
	}
	checked, err := semantics.CheckWorldOpt(
		world,
		semantics.CheckOptions{RequireMain: !opt.InterfaceOnly},
	)
	if err != nil {
		return nil, err
	}
	if opt.InterfaceOnly {
		return interfaceOnlyBuildStats(world), nil
	}
	if err := rejectInterfaceModulesForCodegen(world); err != nil {
		return nil, err
	}

	modules := make([]string, 0, len(world.ByModule))
	for module := range world.ByModule {
		modules = append(modules, module)
	}
	sort.Strings(modules)

	var funcs []IRFunc
	var dataPrefix [][]byte
	globalOffset := 0
	stats := &BuildStats{
		CompiledModules: make([]string, 0, len(modules)),
		LoweredModules:  make([]string, 0, len(modules)),
	}
	memoryState, err := buildMemoryStateForTarget(checked, tgt.Triple)
	if err != nil {
		return nil, err
	}
	loweringResult, err := lowerMemoryStateForBuild(checked, memoryState, tgt.Triple, opt, nil)
	if err != nil {
		return nil, err
	}
	for _, module := range modules {
		moduleFuncs, err := loweringResult.ModuleFuncs(module)
		if err != nil {
			return nil, err
		}
		stats.LoweredModules = append(stats.LoweredModules, module)
		stats.CompiledModules = append(stats.CompiledModules, module)
		funcs = append(funcs, relocateWASMGlobalSlots(moduleFuncs, globalOffset)...)
		moduleData := checked.GlobalDataByModule[module]
		dataPrefix = append(dataPrefix, moduleData...)
		globalOffset += len(moduleData)
	}
	if err := validateWASMIRPolicy(tgt.Triple, funcs); err != nil {
		return nil, err
	}
	if err := rejectUnsupportedWASMRuntimeBuiltins(funcs, tgt.Triple); err != nil {
		return nil, err
	}

	obj, err := wasm32_web.CodegenObjectWithDataPrefix(funcs, checked.MainName, dataPrefix)
	if err != nil {
		return nil, err
	}
	wasmBytes, err := wasm32_web.LinkObject(obj)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(outputPath, wasmBytes, 0o755); err != nil {
		return nil, err
	}

	loaderPath := wasmWebLoaderPath(outputPath)
	loader := wasm32_web.LoaderModule(filepath.Base(outputPath))
	if err := os.WriteFile(loaderPath, loader, 0o644); err != nil {
		return nil, err
	}
	if err := emitUIArtifacts(outputPath, tgt.Triple, checked); err != nil {
		return nil, err
	}
	if err := emitExplainReports(
		outputPath,
		tgt.Triple,
		checked,
		opt,
		memoryState,
		loweringResult,
	); err != nil {
		return nil, err
	}
	return stats, nil
}

func wasmWebLoaderPath(outputPath string) string {
	return buildwasm.WebLoaderPath(outputPath)
}

func relocateWASMGlobalSlots(funcs []IRFunc, offset int) []IRFunc {
	return buildwasm.RelocateGlobalSlots(funcs, offset)
}

func rejectUnsupportedWASMRuntimeBuiltins(funcs []IRFunc, target string) error {
	pos, runtimeName, unsupported := buildwasm.FirstUnsupportedRuntimeBuiltin(funcs, target)
	if !unsupported {
		return nil
	}
	return targetRuntimeDiagnostic(pos, target, runtimeName)
}

func wasmRuntimeNameForBuiltin(name string, target string) (string, bool) {
	return buildwasm.RuntimeNameForBuiltin(name, target)
}

func targetRuntimeDiagnostic(pos frontend.Position, target string, runtimeName string) error {
	hint := ("Build this source for a native x64 target or remove the runtime " +
		"builtin for this WASM target.")
	if !strings.HasPrefix(target, "wasm32-") {
		hint = fmt.Sprintf(
			"Build this source for linux-x64 or remove the %s runtime builtin for this target.",
			runtimeName,
		)
	}
	return &frontend.DiagnosticError{Info: frontend.Diagnostic{
		Code:     DiagnosticCodeTargetRuntime,
		Message:  fmt.Sprintf("%s runtime not supported on %s", runtimeName, target),
		File:     pos.File,
		Line:     pos.Line,
		Column:   pos.Col,
		Severity: "error",
		Hint:     hint,
	}}
}

type wasmIRPolicy struct {
	builtin  string
	category string
}

func validateWASMIRPolicy(target string, funcs []IRFunc) error {
	pos, policy, blocked := buildwasm.FirstBlockedIRPolicy(target, funcs)
	if !blocked {
		return nil
	}
	return targetWASMPolicyDiagnostic(pos, target, rootWASMIRPolicy(policy))
}

func blockedWASMIRPolicy(kind ir.IRInstrKind) (wasmIRPolicy, bool) {
	policy, blocked := buildwasm.BlockedIRPolicy(kind)
	if !blocked {
		return wasmIRPolicy{}, false
	}
	return rootWASMIRPolicy(policy), true
}

func rootWASMIRPolicy(policy buildwasm.IRPolicy) wasmIRPolicy {
	return wasmIRPolicy{builtin: policy.Builtin, category: policy.Category}
}

func targetWASMPolicyDiagnostic(pos frontend.Position, target string, policy wasmIRPolicy) error {
	return &frontend.DiagnosticError{Info: frontend.Diagnostic{
		Code: DiagnosticCodeTargetRuntime,
		Message: fmt.Sprintf(
			"%s target does not support %s (%s); unsupported on WASM targets by policy",
			target,
			policy.builtin,
			policy.category,
		),
		File:     pos.File,
		Line:     pos.Line,
		Column:   pos.Col,
		Severity: "error",
		Hint: ("Build this unsafe/capability memory path for a native x64 " +
			"target, or replace it with the supported WASM-safe slice/island surface."),
	}}
}

func emitUIArtifacts(outputPath string, target string, checked *semantics.CheckedProgram) error {
	return buildwasm.EmitUIArtifacts(outputPath, target, checked)
}

func uiArtifactBasePath(outputPath string) string {
	return buildwasm.UIArtifactBasePath(outputPath)
}

// ---- ffi_target.go ----

func validateTargetExportedFFIAST(world *World, target string) error {
	if world == nil || !targetRequiresExplicitPointerExportGate(target) {
		return nil
	}
	for _, file := range world.Files {
		if file == nil {
			continue
		}
		for _, fn := range file.Funcs {
			if err := validateTargetExportedFFIDeclAST(fn, file.Module, target); err != nil {
				return err
			}
		}
		for _, actor := range file.Actors {
			if actor == nil {
				continue
			}
			for _, method := range actor.Methods {
				if err := validateTargetExportedFFIDeclAST(method, file.Module, target); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateTargetExportedFFIDeclAST(fn *frontend.FuncDecl, module string, target string) error {
	if fn == nil || fn.ExportName == "" || isInternalRuntimeExportedSymbol(module, fn.ExportName) {
		return nil
	}
	for _, param := range fn.Params {
		typeName := targetExportedFFITypeRefName(param.Type)
		if targetExportedFFIRequiresPointerBoundaryGate(target, typeName) {
			return targetExportedFFIPointerParamDiagnostic(
				param.At,
				target,
				fn.Name,
				param.Name,
				typeName,
			)
		}
	}
	typeName := targetExportedFFITypeRefName(fn.ReturnType)
	if targetExportedFFIRequiresPointerBoundaryGate(target, typeName) {
		pos := fn.ReturnType.At
		if pos.Line == 0 || pos.Col == 0 {
			pos = fn.Pos
		}
		return targetExportedFFIPointerReturnDiagnostic(pos, target, fn.Name, typeName)
	}
	return nil
}

func targetExportedFFITypeRefName(ref frontend.TypeRef) string {
	switch ref.Kind {
	case frontend.TypeRefFunction:
		return "fnptr"
	case frontend.TypeRefNamed:
		return strings.TrimSpace(ref.Name)
	default:
		return strings.TrimSpace(ref.Name)
	}
}

func validateTargetExportedFFIABI(checked *semantics.CheckedProgram, target string) error {
	if checked == nil || !targetRequiresExplicitAggregateExportGate(target) {
		return nil
	}
	for _, fn := range checked.Funcs {
		decl := fn.Decl
		if decl == nil || decl.ExportName == "" ||
			isInternalRuntimeExportedSymbol(fn.Module, decl.ExportName) {
			continue
		}
		sig, ok := checked.FuncSigs[fn.Name]
		if !ok {
			continue
		}
		for i, typeName := range sig.ParamTypes {
			if targetExportedFFIRequiresPointerBoundaryGate(target, typeName) {
				pos := decl.Pos
				paramName := fmt.Sprintf("#%d", i+1)
				if i < len(decl.Params) {
					pos = decl.Params[i].At
					paramName = decl.Params[i].Name
				}
				return targetExportedFFIPointerParamDiagnostic(
					pos,
					target,
					decl.Name,
					paramName,
					typeName,
				)
			}
			if !targetExportedFFIRequiresAggregateABI(typeName, checked.Types) {
				continue
			}
			pos := decl.Pos
			paramName := fmt.Sprintf("#%d", i+1)
			if i < len(decl.Params) {
				pos = decl.Params[i].At
				paramName = decl.Params[i].Name
			}
			return targetExportedFFIAggregateParamDiagnostic(
				pos,
				target,
				decl.Name,
				paramName,
				typeName,
			)
		}
		if targetExportedFFIRequiresPointerBoundaryGate(target, sig.ReturnType) {
			pos := decl.ReturnType.At
			if pos.Line == 0 || pos.Col == 0 {
				pos = decl.Pos
			}
			return targetExportedFFIPointerReturnDiagnostic(pos, target, decl.Name, sig.ReturnType)
		}
		if targetExportedFFIRequiresAggregateABI(sig.ReturnType, checked.Types) {
			pos := decl.ReturnType.At
			if pos.Line == 0 || pos.Col == 0 {
				pos = decl.Pos
			}
			return targetExportedFFIAggregateReturnDiagnostic(
				pos,
				target,
				decl.Name,
				sig.ReturnType,
			)
		}
	}
	return nil
}

func targetRequiresExplicitAggregateExportGate(target string) bool {
	return abisuite.TargetRequiresExplicitAggregateExportGate(target)
}

func targetRequiresExplicitPointerExportGate(target string) bool {
	return abisuite.TargetRequiresExplicitPointerExportGate(target)
}

func targetExportedFFIRequiresX32PointerBoundaryGate(target, typeName string) bool {
	return abisuite.TargetExportedFFIRequiresX32PointerBoundaryGate(target, typeName)
}

func targetExportedFFIRequiresPointerBoundaryGate(target, typeName string) bool {
	return abisuite.TargetExportedFFIRequiresPointerBoundaryGate(target, typeName)
}

func translateTargetExportedFFISemanticError(err error, target string) error {
	if err == nil || !targetRequiresExplicitPointerExportGate(target) {
		return err
	}
	diag := DiagnosticFromError(err)
	if !strings.Contains(diag.Message, "cannot expose function-typed value 'fnptr'") {
		return err
	}
	fnName := quotedAfter(diag.Message, "exported function '")
	if fnName == "" {
		return err
	}
	pos := frontend.Position{File: diag.File, Line: diag.Line, Col: diag.Column}
	if strings.Contains(diag.Message, " in parameter '") {
		paramName := quotedAfter(diag.Message, " in parameter '")
		if paramName == "" {
			return err
		}
		return targetExportedFFIPointerParamDiagnostic(pos, target, fnName, paramName, "fnptr")
	}
	if strings.Contains(diag.Message, " in return type") {
		return targetExportedFFIPointerReturnDiagnostic(pos, target, fnName, "fnptr")
	}
	return err
}

func quotedAfter(s, prefix string) string {
	start := strings.Index(s, prefix)
	if start < 0 {
		return ""
	}
	rest := s[start+len(prefix):]
	end := strings.Index(rest, "'")
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func targetExportedFFIRequiresAggregateABI(
	typeName string,
	types map[string]*semantics.TypeInfo,
) bool {
	return abisuite.TargetExportedFFIRequiresAggregateABI(typeName, types)
}

func isInternalRuntimeExportedSymbol(module, exportName string) bool {
	return strings.HasPrefix(exportName, "__tetra_") &&
		(module == "__rt" || strings.HasPrefix(module, "__rt."))
}

func targetExportedFFIAggregateParamDiagnostic(
	pos frontend.Position,
	target, fnName, paramName, typeName string,
) error {
	return &frontend.DiagnosticError{Info: frontend.Diagnostic{
		Code: DiagnosticCodeTargetRuntime,
		Message: fmt.Sprintf(
			("exported function '%s' parameter '%s' type '%s' requires " +
				"aggregate C ABI; aggregate C ABI is not supported on %s"),
			fnName,
			paramName,
			typeName,
			target,
		),
		File:     pos.File,
		Line:     pos.Line,
		Column:   pos.Col,
		Severity: "error",
		Hint: ("Export a scalar FFI wrapper for this target, or keep the " +
			"aggregate behind a target-specific runtime object with a verified C ABI."),
	}}
}

func targetExportedFFIPointerParamDiagnostic(
	pos frontend.Position,
	target, fnName, paramName, typeName string,
) error {
	boundary := targetPointerCBoundaryName(target)
	return &frontend.DiagnosticError{Info: frontend.Diagnostic{
		Code: DiagnosticCodeTargetRuntime,
		Message: fmt.Sprintf(
			("exported function '%s' parameter '%s' type '%s' requires the %s " +
				"pointer C ABI boundary; %s pointer C ABI boundary is not verified on %s"),
			fnName,
			paramName,
			typeName,
			boundary,
			boundary,
			target,
		),
		File:     pos.File,
		Line:     pos.Line,
		Column:   pos.Col,
		Severity: "error",
		Hint: fmt.Sprintf(
			("Export an i32 handle wrapper for %s, or keep the pointer " +
				"boundary inside a verified target-specific runtime object."),
			target,
		),
	}}
}

func targetExportedFFIPointerReturnDiagnostic(
	pos frontend.Position,
	target, fnName, typeName string,
) error {
	boundary := targetPointerCBoundaryName(target)
	return &frontend.DiagnosticError{Info: frontend.Diagnostic{
		Code: DiagnosticCodeTargetRuntime,
		Message: fmt.Sprintf(
			("exported function '%s' return type '%s' requires the %s pointer " +
				"C ABI boundary; %s pointer C ABI boundary is not verified on %s"),
			fnName,
			typeName,
			boundary,
			boundary,
			target,
		),
		File:     pos.File,
		Line:     pos.Line,
		Column:   pos.Col,
		Severity: "error",
		Hint: fmt.Sprintf(
			("Export an i32 handle wrapper for %s, or keep the pointer " +
				"boundary inside a verified target-specific runtime object."),
			target,
		),
	}}
}

func targetPointerCBoundaryName(target string) string {
	switch target {
	case "linux-x86":
		return "i386"
	case "linux-x32":
		return "x32"
	default:
		return target
	}
}

func targetExportedFFIAggregateReturnDiagnostic(
	pos frontend.Position,
	target, fnName, typeName string,
) error {
	return &frontend.DiagnosticError{Info: frontend.Diagnostic{
		Code: DiagnosticCodeTargetRuntime,
		Message: fmt.Sprintf(
			("exported function '%s' return type '%s' requires aggregate C " +
				"ABI; aggregate C ABI is not supported on %s"),
			fnName,
			typeName,
			target,
		),
		File:     pos.File,
		Line:     pos.Line,
		Column:   pos.Col,
		Severity: "error",
		Hint: ("Export a scalar FFI wrapper for this target, or keep the " +
			"aggregate behind a target-specific runtime object with a verified C ABI."),
	}}
}

// ---- linuxx32_filesystem_runtime.go ----

func buildLinuxX32FilesystemRuntimeObject() *Object {
	return buildruntime.BuildLinuxX32FilesystemRuntimeObject()
}

func appendLinuxX32FilesystemRuntimeObject(rt *Object) error {
	return buildruntime.AppendLinuxX32FilesystemRuntimeObject(rt)
}

// ---- linuxx32_net_runtime.go ----

func buildLinuxX32BasicNetRuntimeObject() *Object {
	return buildruntime.BuildLinuxX32BasicNetRuntimeObject()
}

func appendLinuxX32BasicNetRuntimeObject(rt *Object) error {
	return buildruntime.AppendLinuxX32BasicNetRuntimeObject(rt)
}

// ---- selfhostrt_build.go ----

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
	return buildruntime.BuildEmbeddedSelfHostRuntimeObject(target, src, filename, codegen)
}

// ---- selfhostrt_embed.go ----

// Embedded self-host runtime sources.
//
// These are compiled into TOBJ objects on demand and linked when actors are used.

//go:embed selfhostrt/actors_sysv.tetra
var embeddedActorsSysV []byte

//go:embed selfhostrt/actors_win64.tetra
var embeddedActorsWin64 []byte

//go:embed selfhostrt/actors_i386.tetra
var embeddedActorsI386 []byte

//go:embed selfhostrt/time_ilp32.tetra
var embeddedTimeILP32 []byte

// ---- x86_filesystem_runtime.go ----

func buildLinuxX86FilesystemRuntimeObject() *Object {
	return buildruntime.BuildLinuxX86FilesystemRuntimeObject()
}

func appendLinuxX86FilesystemRuntimeObject(rt *Object) error {
	return buildruntime.AppendLinuxX86FilesystemRuntimeObject(rt)
}

// ---- x86_net_runtime.go ----

func buildLinuxX86BasicNetRuntimeObject() *Object {
	return buildruntime.BuildLinuxX86BasicNetRuntimeObject()
}

func appendLinuxX86BasicNetRuntimeObject(rt *Object) error {
	return buildruntime.AppendLinuxX86BasicNetRuntimeObject(rt)
}
