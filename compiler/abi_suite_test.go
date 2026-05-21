package compiler

import "testing"

func TestRunTargetABIChecksCoversX86X64AndRejectsWASM(t *testing.T) {
	tests := []struct {
		target string
		names  []string
	}{
		{
			target: "x86",
			names:  []string{"x86 target model", "x86 i386 SysV classifier", "x86 varargs and sret ABI", "x86 pointer/native-libc FFI diagnostics", "x86 source native scalar diagnostics", "x86 stdlib runtime boundary diagnostics", "x86 target runtime boundary diagnostics", "x86 pointer atomic ABI width"},
		},
		{
			target: "x64",
			names:  []string{"x64 target model", "x64 SysV classifier", "x64 SysV varargs and aggregates", "x64 source native scalar diagnostics", "x64 pointer atomic ABI width"},
		},
		{
			target: "windows-x64",
			names:  []string{"windows-x64 target model", "windows-x64 Win64 classifier", "windows-x64 Win64 varargs and aggregates", "windows-x64 object ABI smoke", "windows-x64 source native scalar diagnostics", "windows-x64 pointer atomic ABI width"},
		},
		{
			target: "macos-x64",
			names:  []string{"macos-x64 target model", "macos-x64 SysV classifier", "macos-x64 SysV varargs and aggregates", "macos-x64 object ABI smoke", "macos-x64 source native scalar diagnostics", "macos-x64 pointer atomic ABI width"},
		},
		{
			target: "x32",
			names:  []string{"x32 target model", "x32 SysV classifier", "x32 SysV varargs and aggregates", "x32 pointer/native-libc FFI diagnostics", "x32 source native scalar diagnostics", "x32 stdlib runtime boundary diagnostics", "x32 target runtime boundary diagnostics", "x32 pointer atomic ABI width"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			checks, err := RunTargetABIChecks(tt.target)
			if err != nil {
				t.Fatalf("RunTargetABIChecks(%s): %v", tt.target, err)
			}
			if len(checks) != len(tt.names) {
				t.Fatalf("checks = %#v, want %d checks", checks, len(tt.names))
			}
			for i, want := range tt.names {
				if checks[i].Name != want || checks[i].Error != "" {
					t.Fatalf("check[%d] = %#v, want passing %q", i, checks[i], want)
				}
			}
		})
	}

	if checks, err := RunTargetABIChecks("wasm32-wasi"); err == nil {
		t.Fatalf("RunTargetABIChecks(wasm32-wasi) = %#v, want unsupported error", checks)
	}
}
