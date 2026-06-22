package buildruntime

type RuntimeObjectPlanUsage struct {
	ActorsUsed             bool
	ActorRuntimeUsed       bool
	ActorSystemReceiveUsed bool
	ActorStateUsed         bool
	TasksUsed              bool
	TaskGroupsUsed         bool
	TypedTasksUsed         bool
	TimeRuntimeUsed        bool
	FilesystemRuntimeUsed  bool
	NetRuntimeUsed         bool
	NetRuntimeSupported    bool
	SurfaceRuntimeUsed     bool
	DistributedActorsUsed  bool
}

type RuntimeObjectPlanDecision struct {
	RuntimeUsed         bool
	TimeOnlyRuntime     bool
	LinuxMinimalRuntime bool
}

func DecideRuntimeObjectPlan(
	target string,
	runtimeObjectOverride bool,
	caps Capabilities,
	usage RuntimeObjectPlanUsage,
) RuntimeObjectPlanDecision {
	runtimeUsed := usage.ActorsUsed ||
		usage.ActorRuntimeUsed ||
		usage.ActorSystemReceiveUsed ||
		usage.ActorStateUsed ||
		usage.TasksUsed ||
		usage.TaskGroupsUsed ||
		usage.TypedTasksUsed ||
		usage.TimeRuntimeUsed ||
		usage.FilesystemRuntimeUsed ||
		usage.NetRuntimeUsed ||
		usage.SurfaceRuntimeUsed ||
		usage.DistributedActorsUsed

	timeOnlyRuntime := caps.TimeOnlyWithoutScheduler &&
		!runtimeObjectOverride &&
		usage.TimeRuntimeUsed &&
		!usage.ActorSystemReceiveUsed &&
		!usage.ActorRuntimeUsed &&
		!usage.ActorStateUsed &&
		!usage.TasksUsed &&
		!usage.TaskGroupsUsed &&
		!usage.TypedTasksUsed &&
		!usage.FilesystemRuntimeUsed &&
		!usage.NetRuntimeUsed &&
		!usage.SurfaceRuntimeUsed &&
		!usage.DistributedActorsUsed

	linuxMinimalRuntime := (target == "linux-x86" || target == "linux-x32") &&
		!runtimeObjectOverride &&
		(usage.FilesystemRuntimeUsed || usage.NetRuntimeUsed) &&
		usage.NetRuntimeSupported &&
		!usage.ActorsUsed &&
		!usage.ActorRuntimeUsed &&
		!usage.ActorSystemReceiveUsed &&
		!usage.ActorStateUsed &&
		!usage.TasksUsed &&
		!usage.TaskGroupsUsed &&
		!usage.TypedTasksUsed &&
		!usage.TimeRuntimeUsed &&
		!usage.SurfaceRuntimeUsed &&
		!usage.DistributedActorsUsed

	return RuntimeObjectPlanDecision{
		RuntimeUsed:         runtimeUsed,
		TimeOnlyRuntime:     timeOnlyRuntime,
		LinuxMinimalRuntime: linuxMinimalRuntime,
	}
}
