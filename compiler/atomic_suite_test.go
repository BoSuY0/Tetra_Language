package compiler

import "testing"

func TestRunTargetAtomicStressChecksCoversX86AndX64Family(t *testing.T) {
	tests := []struct {
		target string
		names  []string
	}{
		{
			target: "x86",
			names:  []string{"x86 atomic validation matrix", "x86 atomic object matrix", "x86 pointer atomic object width", "x86 atomic concurrency stress oracle", "x86 atomic diagnostics"},
		},
		{
			target: "x64",
			names:  []string{"x64 atomic validation matrix", "x64 atomic object matrix", "x64 pointer atomic object width", "x64 atomic concurrency stress oracle", "x64 atomic diagnostics"},
		},
		{
			target: "windows-x64",
			names:  []string{"windows-x64 atomic validation matrix", "windows-x64 atomic object matrix", "windows-x64 pointer atomic object width", "windows-x64 atomic concurrency stress oracle", "windows-x64 atomic diagnostics"},
		},
		{
			target: "macos-x64",
			names:  []string{"macos-x64 atomic validation matrix", "macos-x64 atomic object matrix", "macos-x64 pointer atomic object width", "macos-x64 atomic concurrency stress oracle", "macos-x64 atomic diagnostics"},
		},
		{
			target: "x32",
			names:  []string{"x32 atomic validation matrix", "x32 atomic object matrix", "x32 pointer atomic object width", "x32 atomic concurrency stress oracle", "x32 atomic diagnostics"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			checks, err := RunTargetAtomicStressChecks(tt.target)
			if err != nil {
				t.Fatalf("RunTargetAtomicStressChecks(%s): %v", tt.target, err)
			}
			if len(checks) != len(tt.names) {
				t.Fatalf("checks = %#v, want %d checks", checks, len(tt.names))
			}
			for i, want := range tt.names {
				if checks[i].Name != want {
					t.Fatalf("check[%d] = %#v, want name %q", i, checks[i], want)
				}
				if checks[i].Error != "" {
					t.Fatalf("check[%d] = %#v, want passing %q", i, checks[i], want)
				}
			}
		})
	}
}

func TestAtomicStressIterationsEnvOverride(t *testing.T) {
	t.Setenv("TETRA_ATOMIC_STRESS_ITERS", "7")
	got, err := atomicStressIterations()
	if err != nil {
		t.Fatalf("atomicStressIterations: %v", err)
	}
	if got != 7 {
		t.Fatalf("iterations = %d, want 7", got)
	}
}

func TestAtomicStressIterationsRejectsInvalidEnv(t *testing.T) {
	for _, raw := range []string{"0", "-1", "abc", "100001"} {
		t.Run(raw, func(t *testing.T) {
			t.Setenv("TETRA_ATOMIC_STRESS_ITERS", raw)
			if got, err := atomicStressIterations(); err == nil {
				t.Fatalf("atomicStressIterations() = %d, want error", got)
			}
		})
	}
}
