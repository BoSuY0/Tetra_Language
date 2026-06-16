package abisuite

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRuntimeSelfHostSmokesUseCallbacks(t *testing.T) {
	var built []string
	deps := RuntimeSmokeDeps{
		BuildExecutable: func(srcPath string, outPath string, target string) error {
			if _, err := os.Stat(srcPath); err != nil {
				t.Fatalf("source file was not written before build callback: %v", err)
			}
			built = append(built, filepath.Base(outPath)+":"+target)
			return os.WriteFile(outPath, runtimeSmokeTestELF(target), 0o755)
		},
	}

	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{name: "x32 task", run: func() error { return CheckX32SingleTaskSelfHostRuntimeSmoke(deps) }},
		{name: "x32 typed task", run: func() error { return CheckX32TypedTaskSelfHostRuntimeSmoke(deps) }},
		{name: "x32 staged typed task", run: func() error { return CheckX32StagedTypedTaskSelfHostRuntimeSmoke(deps) }},
		{name: "x32 task group", run: func() error { return CheckX32TaskGroupSelfHostRuntimeSmoke(deps) }},
		{name: "x32 typed task group", run: func() error { return CheckX32TypedTaskGroupSelfHostRuntimeSmoke(deps) }},
		{name: "x32 actor", run: func() error { return CheckX32SingleActorSelfHostRuntimeSmoke(deps) }},
		{name: "x32 actor state", run: func() error { return CheckX32ActorStateSelfHostRuntimeSmoke(deps) }},
		{name: "x86 task", run: func() error { return CheckX86SingleTaskSelfHostRuntimeSmoke(deps) }},
		{name: "x86 typed task", run: func() error { return CheckX86TypedTaskSelfHostRuntimeSmoke(deps) }},
		{name: "x86 staged typed task", run: func() error { return CheckX86StagedTypedTaskSelfHostRuntimeSmoke(deps) }},
		{name: "x86 task group", run: func() error { return CheckX86TaskGroupSelfHostRuntimeSmoke(deps) }},
		{name: "x86 typed task group", run: func() error { return CheckX86TypedTaskGroupSelfHostRuntimeSmoke(deps) }},
		{name: "x86 actor", run: func() error { return CheckX86SingleActorSelfHostRuntimeSmoke(deps) }},
		{name: "x86 actor state", run: func() error { return CheckX86ActorStateSelfHostRuntimeSmoke(deps) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err != nil {
				t.Fatalf("self-host runtime smoke: %v", err)
			}
		})
	}

	wantBuilt := []string{
		"x32-task-runtime:linux-x32",
		"x32-typed-task-runtime:linux-x32",
		"x32-staged-typed-task-runtime:linux-x32",
		"x32-task-group-runtime:linux-x32",
		"x32-typed-task-group-runtime:linux-x32",
		"x32-actor-runtime:linux-x32",
		"x32-actor-state-runtime:linux-x32",
		"x86-task-runtime:linux-x86",
		"x86-typed-task-runtime:linux-x86",
		"x86-staged-typed-task-runtime:linux-x86",
		"x86-task-group-runtime:linux-x86",
		"x86-typed-task-group-runtime:linux-x86",
		"x86-actor-runtime:linux-x86",
		"x86-actor-state-runtime:linux-x86",
	}
	if !reflect.DeepEqual(built, wantBuilt) {
		t.Fatalf("built = %#v, want %#v", built, wantBuilt)
	}
}
