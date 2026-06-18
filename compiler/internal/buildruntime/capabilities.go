package buildruntime

const UnlimitedActorSpawns = -1

type Capabilities struct {
	Actors                   bool
	ActorState               bool
	Tasks                    bool
	TaskGroups               bool
	TypedTasks               bool
	Time                     bool
	TimeOnlyWithoutScheduler bool
	Filesystem               bool
	Networking               bool
	Surface                  bool
	DistributedActors        bool
	MaxActorSpawns           int
	MaxTypedTaskSlots        int
	BuiltinRuntime           bool
	SelfHostActorsRuntime    bool
	SelfHostTimeRuntime      bool
}

func CapabilitiesForTarget(target string) Capabilities {
	switch target {
	case "linux-x64":
		return Capabilities{
			Actors:                true,
			ActorState:            true,
			Tasks:                 true,
			TaskGroups:            true,
			TypedTasks:            true,
			Time:                  true,
			Filesystem:            true,
			Networking:            true,
			Surface:               true,
			DistributedActors:     true,
			MaxActorSpawns:        UnlimitedActorSpawns,
			MaxTypedTaskSlots:     8,
			BuiltinRuntime:        true,
			SelfHostActorsRuntime: true,
		}
	case "linux-x32":
		return Capabilities{
			Actors:                true,
			ActorState:            true,
			Tasks:                 true,
			TaskGroups:            true,
			TypedTasks:            true,
			Time:                  true,
			Filesystem:            true,
			Networking:            true,
			MaxActorSpawns:        2,
			MaxTypedTaskSlots:     8,
			SelfHostActorsRuntime: true,
		}
	case "linux-x86":
		return Capabilities{
			Actors:                   true,
			ActorState:               true,
			Tasks:                    true,
			TaskGroups:               true,
			TypedTasks:               true,
			Time:                     true,
			TimeOnlyWithoutScheduler: true,
			Filesystem:               true,
			Networking:               true,
			MaxActorSpawns:           2,
			MaxTypedTaskSlots:        8,
			SelfHostActorsRuntime:    true,
			SelfHostTimeRuntime:      true,
		}
	case "macos-x64", "windows-x64":
		return Capabilities{
			Actors:                true,
			ActorState:            true,
			Tasks:                 true,
			TaskGroups:            true,
			TypedTasks:            true,
			Time:                  true,
			MaxActorSpawns:        UnlimitedActorSpawns,
			MaxTypedTaskSlots:     8,
			BuiltinRuntime:        true,
			SelfHostActorsRuntime: true,
		}
	default:
		return Capabilities{MaxActorSpawns: 0}
	}
}
