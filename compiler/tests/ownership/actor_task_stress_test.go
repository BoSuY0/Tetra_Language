package compiler_test

import (
	"path/filepath"
	"runtime"
	"testing"

	compiler "tetra_language/compiler"
	"tetra_language/compiler/internal/testkit"
)

func TestActorTaskBoundedStressExamples(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux/amd64 only")
	}

	tests := []struct {
		name     string
		path     string
		wantExit int
	}{
		{
			name:     "task bounded stress",
			path:     testkit.RepoPath(t, "examples", "task_bounded_stress.tetra"),
			wantExit: 42,
		},
		{
			name:     "task group cancel smoke",
			path:     testkit.RepoPath(t, "examples", "task_group_cancel_smoke.tetra"),
			wantExit: 1,
		},
		{
			name:     "task group lifecycle smoke",
			path:     testkit.RepoPath(t, "examples", "task_group_lifecycle_smoke.tetra"),
			wantExit: 42,
		},
		{
			name:     "actor task dogfood",
			path:     testkit.RepoPath(t, "examples", "projects", "dogfood_actor_task", "src", "main.tetra"),
			wantExit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, exitCode := buildActorTaskStressFile(t, tt.path)
			if stdout != "" {
				t.Fatalf("stdout mismatch: %q", stdout)
			}
			if exitCode != tt.wantExit {
				t.Fatalf("exit code = %d, want %d", exitCode, tt.wantExit)
			}
		})
	}
}

func buildActorTaskStressFile(t *testing.T, srcPath string) (string, int) {
	t.Helper()
	outPath := filepath.Join(t.TempDir(), "app")
	if err := compiler.BuildFile(srcPath, outPath, "linux-x64"); err != nil {
		t.Fatalf("build %s: %v", srcPath, err)
	}
	return testkit.RunBinary(t, outPath)
}
