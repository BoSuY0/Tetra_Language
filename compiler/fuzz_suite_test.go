package compiler

import "testing"

func TestRunTargetFuzzChecksCoversX86AndX64Family(t *testing.T) {
	tests := []struct {
		target string
		names  []string
	}{
		{
			target: "x86",
			names:  []string{"x86 layout fuzz", "x86 object signature fuzz", "x86 target alias fuzz"},
		},
		{
			target: "x64",
			names:  []string{"x64 layout fuzz", "x64 object signature fuzz", "x64 target alias fuzz"},
		},
		{
			target: "windows-x64",
			names:  []string{"windows-x64 layout fuzz", "windows-x64 object signature fuzz", "windows-x64 target alias fuzz"},
		},
		{
			target: "macos-x64",
			names:  []string{"macos-x64 layout fuzz", "macos-x64 object signature fuzz", "macos-x64 target alias fuzz"},
		},
		{
			target: "x32",
			names:  []string{"x32 layout fuzz", "x32 object signature fuzz", "x32 target alias fuzz"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			checks, err := RunTargetFuzzChecks(tt.target)
			if err != nil {
				t.Fatalf("RunTargetFuzzChecks(%s): %v", tt.target, err)
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
