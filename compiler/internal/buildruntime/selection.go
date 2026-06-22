package buildruntime

import (
	"fmt"

	"tetra_language/compiler/internal/buildapi"
	"tetra_language/compiler/internal/runtimeabi"
)

type UsageProfile struct {
	ActorSystemReceiveUsed bool
	ActorStateUsed         bool
	TasksUsed              bool
	TaskGroupsUsed         bool
	TypedTasksUsed         bool
	TypedTaskMaxSlots      int
	TimeRuntimeUsed        bool
	FilesystemUsed         bool
	NetUsed                bool
	NetRuntimeSymbols      []string
	SurfaceUsed            bool
	DistributedActorsUsed  bool
	ActorSpawnCount        int
}

func SelectRuntimeMode(
	requested buildapi.RuntimeMode,
	usage UsageProfile,
) (buildapi.RuntimeMode, error) {
	switch requested {
	case buildapi.RuntimeAuto:
		// Default to self-host runtime when its ABI can express the program surface.
		if usage.ActorSystemReceiveUsed ||
			usage.ActorStateUsed || usage.TasksUsed || usage.TaskGroupsUsed ||
			usage.TypedTasksUsed ||
			usage.TimeRuntimeUsed ||
			usage.FilesystemUsed ||
			usage.NetUsed ||
			usage.SurfaceUsed ||
			usage.DistributedActorsUsed ||
			usage.TypedTaskMaxSlots > 4 ||
			usage.ActorSpawnCount > 1 {
			return buildapi.RuntimeBuiltin, nil
		}
		return buildapi.RuntimeSelfHost, nil
	case buildapi.RuntimeSelfHost:
		if usage.ActorSystemReceiveUsed {
			return 0, fmt.Errorf(
				"self-host runtime does not support actor system-message receive; use runtime=auto or runtime=builtin",
			)
		}
		if usage.SurfaceUsed {
			return 0, fmt.Errorf(
				"self-host runtime does not support Tetra Surface; use runtime=auto or runtime=builtin",
			)
		}
		if usage.DistributedActorsUsed {
			return 0, fmt.Errorf(
				"self-host runtime does not support distributed actors; use runtime=auto or runtime=builtin",
			)
		}
		if usage.TaskGroupsUsed {
			return 0, fmt.Errorf(
				"self-host runtime does not support task groups; use runtime=auto or runtime=builtin",
			)
		}
		if usage.TypedTasksUsed {
			return 0, fmt.Errorf(
				"self-host runtime does not support typed task handles; use runtime=auto or runtime=builtin",
			)
		}
		if usage.ActorSpawnCount > 1 {
			return 0, fmt.Errorf(
				"self-host runtime supports at most one spawned actor; use runtime=auto or runtime=builtin",
			)
		}
		return buildapi.RuntimeSelfHost, nil
	case buildapi.RuntimeBuiltin:
		return buildapi.RuntimeBuiltin, nil
	default:
		return 0, fmt.Errorf("unsupported runtime mode: %d", requested)
	}
}

func RuntimeModeForNativeTarget(
	target string,
	requested buildapi.RuntimeMode,
	selected buildapi.RuntimeMode,
	usage UsageProfile,
) (buildapi.RuntimeMode, error) {
	caps := CapabilitiesForTarget(target)
	if !caps.SelfHostActorsRuntime || caps.BuiltinRuntime || requested != buildapi.RuntimeAuto ||
		selected != buildapi.RuntimeBuiltin {
		return selected, nil
	}
	if SelfHostRuntimeSupportsNativeUsage(target, usage) {
		return buildapi.RuntimeSelfHost, nil
	}
	return selected, nil
}

func SelectRuntimeModeForNativeTarget(
	target string,
	requested buildapi.RuntimeMode,
	usage UsageProfile,
) (buildapi.RuntimeMode, error) {
	selected, err := SelectRuntimeMode(requested, usage)
	if err != nil {
		if requested == buildapi.RuntimeSelfHost &&
			SelfHostRuntimeSupportsNativeUsage(target, usage) {
			return buildapi.RuntimeSelfHost, nil
		}
		return 0, err
	}
	return RuntimeModeForNativeTarget(target, requested, selected, usage)
}

func SelfHostRuntimeSupportsNativeUsage(target string, usage UsageProfile) bool {
	if usage.ActorSystemReceiveUsed || usage.SurfaceUsed || usage.DistributedActorsUsed {
		return false
	}
	if usage.NetUsed && !TargetSupportsNetRuntimeSymbols(target, usage.NetRuntimeSymbols) {
		return false
	}
	switch target {
	case "linux-x32":
		if usage.ActorSpawnCount > 2 {
			return false
		}
		return !usage.TypedTasksUsed || usage.TypedTaskMaxSlots <= 8
	case "linux-x86":
		if usage.ActorSpawnCount > 2 {
			return false
		}
		return !usage.TypedTasksUsed || usage.TypedTaskMaxSlots <= 8
	default:
		if usage.ActorSpawnCount > 1 {
			return false
		}
		if usage.TypedTasksUsed {
			return false
		}
		_, err := SelectRuntimeMode(buildapi.RuntimeSelfHost, usage)
		return err == nil
	}
}

func TargetSupportsNetRuntimeSymbols(target string, symbols []string) bool {
	if len(symbols) == 0 {
		return true
	}
	supported := SupportedNetRuntimeSymbolsForTarget(target)
	if len(supported) == 0 {
		return false
	}
	for _, symbol := range symbols {
		if _, ok := supported[symbol]; !ok {
			return false
		}
	}
	return true
}

func SupportedNetRuntimeSymbolsForTarget(target string) map[string]struct{} {
	if CapabilitiesForTarget(target).Networking {
		required := runtimeabi.RequiredNetSymbols()
		symbols := make(map[string]struct{}, len(required))
		for _, symbol := range required {
			symbols[symbol] = struct{}{}
		}
		return symbols
	}
	return nil
}
