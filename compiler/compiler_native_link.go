package compiler

import (
	"fmt"
	"tetra_language/compiler/internal/buildnative"
	"tetra_language/compiler/internal/buildruntime"
	"tetra_language/compiler/internal/semantics"
)

func linkNativeExecutable(outputPath string, native nativeBuildTarget, opt BuildOptions, checked *semantics.CheckedProgram, objects []*Object, linkedObjects []linkedObject) error {
	actorsUsed, actorEntries, actorSpawnCount, err := collectActorEntries(checked)
	if err != nil {
		return err
	}
	actorStateUsed, actorStatePos := collectActorStateRuntimeUsagePosition(checked)
	actorRuntimeUsed, actorRuntimePos := collectActorRuntimeUsagePosition(checked)
	tasksUsed, tasksPos := collectTaskRuntimeUsagePosition(checked)
	taskGroupsUsed := collectTaskGroupRuntimeUsage(checked)
	typedTasksUsed, typedTaskMaxSlots := collectTypedTaskRuntimeUsage(checked)
	timeRuntimeUsed, timeRuntimePos := collectTimeRuntimeUsagePosition(checked)
	filesystemRuntimeUsed, filesystemRuntimePos := collectFilesystemRuntimeUsagePosition(checked)
	netRuntimeUsage := collectNetRuntimeUsageProfile(checked)
	netRuntimeUsed := netRuntimeUsage.used
	surfaceRuntimeUsed, surfaceRuntimePos := collectSurfaceRuntimeUsagePosition(checked)
	distributedActorsUsed, distributedActorsPos := collectDistributedActorRuntimeUsagePosition(checked)
	runtimeCaps := nativeRuntimeCapabilitiesForTarget(native.triple)
	runtimeObjectPlan := buildruntime.DecideRuntimeObjectPlan(
		native.triple,
		opt.RuntimeObjectPath != "",
		buildruntime.CapabilitiesForTarget(native.triple),
		buildruntime.RuntimeObjectPlanUsage{
			ActorsUsed:            actorsUsed,
			ActorRuntimeUsed:      actorRuntimeUsed,
			ActorStateUsed:        actorStateUsed,
			TasksUsed:             tasksUsed,
			TaskGroupsUsed:        taskGroupsUsed,
			TypedTasksUsed:        typedTasksUsed,
			TimeRuntimeUsed:       timeRuntimeUsed,
			FilesystemRuntimeUsed: filesystemRuntimeUsed,
			NetRuntimeUsed:        netRuntimeUsed,
			NetRuntimeSupported:   targetSupportsNetRuntimeUsage(native.triple, netRuntimeUsage),
			SurfaceRuntimeUsed:    surfaceRuntimeUsed,
			DistributedActorsUsed: distributedActorsUsed,
		},
	)
	timeOnlyRuntime := runtimeObjectPlan.TimeOnlyRuntime
	linuxMinimalRuntime := runtimeObjectPlan.LinuxMinimalRuntime
	if netRuntimeUsed {
		if pos, unsupported := unsupportedNetRuntimeUsagePosition(native.triple, netRuntimeUsage); unsupported {
			return targetRuntimeDiagnostic(pos, native.triple, "networking")
		}
	}
	if filesystemRuntimeUsed && !runtimeCaps.filesystem {
		return targetRuntimeDiagnostic(filesystemRuntimePos, native.triple, "filesystem")
	}
	if surfaceRuntimeUsed && !runtimeCaps.surface {
		return targetRuntimeDiagnostic(surfaceRuntimePos, native.triple, "surface")
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
	if actorStateUsed && !runtimeCaps.actorState {
		return targetRuntimeDiagnostic(actorStatePos, native.triple, "actors")
	}
	if runtimeCaps.actors && runtimeCaps.maxActorSpawns != unlimitedActorSpawns && actorSpawnCount > runtimeCaps.maxActorSpawns {
		return targetRuntimeDiagnostic(actorRuntimePos, native.triple, fmt.Sprintf("actor fanout above %d", runtimeCaps.maxActorSpawns))
	}
	if taskGroupsUsed && !runtimeCaps.taskGroups {
		return targetRuntimeDiagnostic(tasksPos, native.triple, "task group")
	}
	if typedTasksUsed && !runtimeCaps.typedTasks {
		return targetRuntimeDiagnostic(tasksPos, native.triple, "typed task")
	}
	if typedTasksUsed && runtimeCaps.maxTypedTaskSlots > 0 && typedTaskMaxSlots > runtimeCaps.maxTypedTaskSlots {
		return targetRuntimeDiagnostic(tasksPos, native.triple, "staged typed task")
	}
	runtimeUsed := runtimeObjectPlan.RuntimeUsed
	if runtimeUsed && len(actorEntries) == 0 {
		actorEntries = []string{checked.MainName}
	}
	mainName := checked.MainName
	if opt.RuntimeObjectPath != "" && !runtimeUsed {
		return fmt.Errorf("runtime object override requires runtime usage (no actor/task/time/filesystem/networking/surface/distributed actor builtins found)")
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
				actorStateUsed:        actorStateUsed,
				tasksUsed:             tasksUsed,
				taskGroupsUsed:        taskGroupsUsed,
				typedTasksUsed:        typedTasksUsed,
				typedTaskMaxSlots:     typedTaskMaxSlots,
				timeRuntimeUsed:       timeRuntimeUsed,
				filesystemUsed:        filesystemRuntimeUsed,
				netUsed:               netRuntimeUsed,
				netRuntimeSymbols:     netRuntimeUsage.requiredSymbols(),
				surfaceUsed:           surfaceRuntimeUsed,
				distributedActorsUsed: distributedActorsUsed,
				actorSpawnCount:       actorSpawnCount,
			}
			runtimeMode, err := selectRuntimeModeForNativeTarget(native.triple, opt.Runtime, usage)
			if err != nil {
				return err
			}
			if native.triple == "linux-x32" && opt.RuntimeObjectPath == "" && runtimeMode == RuntimeBuiltin {
				return fmt.Errorf("builtin runtime is not supported on target linux-x32; use runtime=selfhost for supported self-host runtime builds or remove runtime builtins")
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
					return fmt.Errorf("runtime object target mismatch: got=%s want=%s", rt.Target, native.triple)
				}
			} else {
				switch runtimeMode {
				case RuntimeSelfHost:
					rt, err = buildEmbeddedSelfHostActorsRuntimeObject(native.triple, native.codegen)
				case RuntimeBuiltin:
					if native.backend.actorRuntime == nil {
						return fmt.Errorf("actors runtime is not supported on target %s", native.triple)
					}
					rt, err = native.backend.actorRuntime(actorEntries)
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

			glueObj, builtGlue, err := buildruntime.BuildActorGlueObject(rt, native.triple, actorEntries, checked, native.codegen)
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
	return buildnative.LinkExecutable(outputPath, native.triple, buildNativeExecutableBackend(native.backend), objects, mainName)
}
