package compiler

import (
	"sort"

	"tetra_language/compiler/internal/buildruntime"
)

const (
	runtimeObjectPlanEvidenceClass  = "native_runtime_object_plan"
	runtimeObjectPlanEvidenceMethod = "native_link_runtime_object_plan_v1"
)

func annotateBackendReportRuntimeObjectPlan(report *backendReport, target string, checked *CheckedProgram, opt BuildOptions) error {
	if report == nil {
		return nil
	}
	plan, err := buildBackendRuntimeObjectPlan(target, opt.RuntimeObjectPath != "", checked)
	if err != nil {
		return err
	}
	report.Summary.RuntimeObjectPlan = plan
	return nil
}

func buildBackendRuntimeObjectPlan(target string, runtimeObjectOverride bool, checked *CheckedProgram) (backendRuntimeObjectPlan, error) {
	actorsUsed, _, _, err := collectActorEntries(checked)
	if err != nil {
		return backendRuntimeObjectPlan{}, err
	}
	actorStateUsed, _ := collectActorStateRuntimeUsagePosition(checked)
	actorRuntimeUsed, _ := collectActorRuntimeUsagePosition(checked)
	tasksUsed, _ := collectTaskRuntimeUsagePosition(checked)
	taskGroupsUsed := collectTaskGroupRuntimeUsage(checked)
	typedTasksUsed, _ := collectTypedTaskRuntimeUsage(checked)
	timeRuntimeUsed, _ := collectTimeRuntimeUsagePosition(checked)
	filesystemRuntimeUsed, _ := collectFilesystemRuntimeUsagePosition(checked)
	netRuntimeUsage := collectNetRuntimeUsageProfile(checked)
	surfaceRuntimeUsed, _ := collectSurfaceRuntimeUsagePosition(checked)
	distributedActorsUsed, _ := collectDistributedActorRuntimeUsagePosition(checked)

	usage := buildruntime.RuntimeObjectPlanUsage{
		ActorsUsed:            actorsUsed,
		ActorRuntimeUsed:      actorRuntimeUsed,
		ActorStateUsed:        actorStateUsed,
		TasksUsed:             tasksUsed,
		TaskGroupsUsed:        taskGroupsUsed,
		TypedTasksUsed:        typedTasksUsed,
		TimeRuntimeUsed:       timeRuntimeUsed,
		FilesystemRuntimeUsed: filesystemRuntimeUsed,
		NetRuntimeUsed:        netRuntimeUsage.used,
		NetRuntimeSupported:   targetSupportsNetRuntimeUsage(target, netRuntimeUsage),
		SurfaceRuntimeUsed:    surfaceRuntimeUsed,
		DistributedActorsUsed: distributedActorsUsed,
	}
	decision := buildruntime.DecideRuntimeObjectPlan(target, runtimeObjectOverride, buildruntime.CapabilitiesForTarget(target), usage)
	required := runtimeObjectFeaturesForUsage(usage)
	linked := []string{}
	initialized := []string{}
	if decision.RuntimeUsed {
		linked = append([]string(nil), required...)
		initialized = append([]string(nil), required...)
	}
	return backendRuntimeObjectPlan{
		EvidenceClass:                    runtimeObjectPlanEvidenceClass,
		EvidenceMethod:                   runtimeObjectPlanEvidenceMethod,
		RuntimeUsed:                      decision.RuntimeUsed,
		RuntimeObjectLinked:              decision.RuntimeUsed,
		RuntimeObjectInitialized:         decision.RuntimeUsed,
		RuntimeObjectOverride:            runtimeObjectOverride,
		TimeOnlyRuntime:                  decision.TimeOnlyRuntime,
		LinuxMinimalRuntime:              decision.LinuxMinimalRuntime,
		RuntimeObjectFeaturesRequired:    required,
		RuntimeObjectFeaturesLinked:      linked,
		RuntimeObjectFeaturesInitialized: initialized,
		RuntimeObjectLazyInitBlockers:    []string{},
	}, nil
}

func runtimeObjectFeaturesForUsage(usage buildruntime.RuntimeObjectPlanUsage) []string {
	features := map[string]struct{}{}
	if usage.ActorRuntimeUsed {
		features["actor_runtime"] = struct{}{}
	}
	if usage.ActorStateUsed {
		features["actor_state_runtime"] = struct{}{}
	}
	if usage.TasksUsed {
		features["task_runtime"] = struct{}{}
	}
	if usage.TaskGroupsUsed {
		features["task_group_runtime"] = struct{}{}
	}
	if usage.TypedTasksUsed {
		features["typed_task_runtime"] = struct{}{}
	}
	if usage.TimeRuntimeUsed {
		features["time_runtime"] = struct{}{}
	}
	if usage.FilesystemRuntimeUsed {
		features["filesystem_runtime"] = struct{}{}
	}
	if usage.NetRuntimeUsed {
		features["net_runtime"] = struct{}{}
	}
	if usage.SurfaceRuntimeUsed {
		features["surface_runtime"] = struct{}{}
	}
	if usage.DistributedActorsUsed {
		features["distributed_actor_runtime"] = struct{}{}
	}
	out := make([]string, 0, len(features))
	for feature := range features {
		out = append(out, feature)
	}
	sort.Strings(out)
	return out
}
