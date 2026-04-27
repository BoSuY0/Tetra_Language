package compiler

import (
	"path/filepath"
	"runtime"
	"testing"
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
			path:     filepath.Join("..", "examples", "task_bounded_stress.tetra"),
			wantExit: 42,
		},
		{
			name:     "actor task dogfood",
			path:     filepath.Join("..", "examples", "projects", "dogfood_actor_task", "src", "main.tetra"),
			wantExit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, exitCode := buildAndRunFile(t, tt.path)
			if stdout != "" {
				t.Fatalf("stdout mismatch: %q", stdout)
			}
			if exitCode != tt.wantExit {
				t.Fatalf("exit code = %d, want %d", exitCode, tt.wantExit)
			}
		})
	}
}
