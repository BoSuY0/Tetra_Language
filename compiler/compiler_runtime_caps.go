package compiler

import (
	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/buildruntime"
	"tetra_language/compiler/internal/frontend"
)

type runtimeUsageProfile struct {
	actorStateUsed        bool
	tasksUsed             bool
	taskGroupsUsed        bool
	typedTasksUsed        bool
	typedTaskMaxSlots     int
	timeRuntimeUsed       bool
	filesystemUsed        bool
	netUsed               bool
	netRuntimeSymbols     []string
	surfaceUsed           bool
	distributedActorsUsed bool
	actorSpawnCount       int
}

func (u runtimeUsageProfile) buildRuntimeUsage() buildruntime.UsageProfile {
	return buildruntime.UsageProfile{
		ActorStateUsed:        u.actorStateUsed,
		TasksUsed:             u.tasksUsed,
		TaskGroupsUsed:        u.taskGroupsUsed,
		TypedTasksUsed:        u.typedTasksUsed,
		TypedTaskMaxSlots:     u.typedTaskMaxSlots,
		TimeRuntimeUsed:       u.timeRuntimeUsed,
		FilesystemUsed:        u.filesystemUsed,
		NetUsed:               u.netUsed,
		NetRuntimeSymbols:     u.netRuntimeSymbols,
		SurfaceUsed:           u.surfaceUsed,
		DistributedActorsUsed: u.distributedActorsUsed,
		ActorSpawnCount:       u.actorSpawnCount,
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

func runtimeModeForNativeTarget(target string, requested RuntimeMode, selected RuntimeMode, usage runtimeUsageProfile) (RuntimeMode, error) {
	return buildruntime.RuntimeModeForNativeTarget(target, requested, selected, usage.buildRuntimeUsage())
}

func selectRuntimeModeForNativeTarget(target string, requested RuntimeMode, usage runtimeUsageProfile) (RuntimeMode, error) {
	return buildruntime.SelectRuntimeModeForNativeTarget(target, requested, usage.buildRuntimeUsage())
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

func unsupportedNetRuntimeUsagePosition(target string, usage netRuntimeUsageProfile) (frontend.Position, bool) {
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
	return runtimeObjectSlotSignature{paramSlots: sig.ParamSlots, returnSlots: sig.ReturnSlots}, true
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
