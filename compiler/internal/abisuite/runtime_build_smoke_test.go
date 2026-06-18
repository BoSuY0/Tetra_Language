package abisuite

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRuntimeBuildSmokesUseCallbacks(t *testing.T) {
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
		{name: "x86 time", run: func() error { return CheckX86TimeRuntimeSmoke(deps) }},
		{name: "x86 filesystem", run: func() error { return CheckX86FilesystemRuntimeSmoke(deps) }},
		{name: "x86 filesystem scheduler", run: func() error { return CheckX86FilesystemSchedulerCompositionSmoke(deps) }},
		{name: "x32 time", run: func() error { return CheckX32TimeRuntimeSmoke(deps) }},
		{name: "x32 filesystem", run: func() error { return CheckX32FilesystemRuntimeSmoke(deps) }},
		{name: "x32 filesystem scheduler", run: func() error { return CheckX32FilesystemSchedulerCompositionSmoke(deps) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err != nil {
				t.Fatalf("runtime build smoke: %v", err)
			}
		})
	}

	wantBuilt := []string{
		"x86-time-runtime:linux-x86",
		"x86-filesystem-runtime:linux-x86",
		"x86-filesystem-scheduler:linux-x86",
		"x32-time-runtime:linux-x32",
		"x32-filesystem-runtime:linux-x32",
		"x32-filesystem-scheduler-runtime:linux-x32",
	}
	if !reflect.DeepEqual(built, wantBuilt) {
		t.Fatalf("built = %#v, want %#v", built, wantBuilt)
	}
}
