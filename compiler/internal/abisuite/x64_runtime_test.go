package abisuite

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ctarget "tetra_language/compiler/target"
)

func TestX64AndSourceNativeChecksUseFFICallbacks(t *testing.T) {
	var lastTarget string
	deps := FFICheckDeps{
		BuildLibrary: func(srcPath string, outPath string, target string) error {
			lastTarget = target
			if strings.Contains(srcPath, "u32_param") ||
				strings.Contains(srcPath, "u64_param") ||
				strings.Contains(srcPath, "f64_return") ||
				strings.Contains(srcPath, "usize_param") ||
				strings.Contains(srcPath, "size_t_param") ||
				strings.Contains(srcPath, "native_int_return") ||
				strings.Contains(srcPath, "c_long_return") {
				return fmt.Errorf("target-layout scalar type is not supported in source-level Tetra yet; native-int/codegen support is required")
			}
			return nil
		},
		ReadObject: func(path string) (ObjectSummary, error) {
			obj := ObjectSummary{
				Target: lastTarget,
				Data:   []byte(lastTarget + " abi\n"),
				Symbols: []ObjectSymbolSummary{
					{Name: "ffi_say_i32", HasSignature: true, ParamSlots: 0, ReturnSlots: 1},
					{Name: "ffi_ptr_param_c", HasSignature: true, ParamSlots: 1, ReturnSlots: 1},
				},
				Relocs: []ObjectRelocSummary{
					{Kind: ObjectRelocDataDisp32},
				},
			}
			if lastTarget == "windows-x64" {
				obj.Relocs = append(obj.Relocs,
					ObjectRelocSummary{Kind: ObjectRelocIATDisp32, Name: "kernel32.GetStdHandle"},
					ObjectRelocSummary{Kind: ObjectRelocIATDisp32, Name: "kernel32.WriteFile"},
				)
			}
			return obj, nil
		},
	}

	for _, targetName := range []string{"linux-x86", "windows-x64"} {
		t.Run("source native scalar "+targetName, func(t *testing.T) {
			tgt, err := ctarget.Parse(targetName)
			if err != nil {
				t.Fatalf("parse target: %v", err)
			}
			if err := CheckSourceNativeScalarDiagnostics(tgt, deps); err != nil {
				t.Fatalf("source native scalar diagnostics: %v", err)
			}
		})
	}

	for _, targetName := range []string{"macos-x64", "windows-x64"} {
		t.Run("platform object "+targetName, func(t *testing.T) {
			tgt, err := ctarget.Parse(targetName)
			if err != nil {
				t.Fatalf("parse target: %v", err)
			}
			if err := CheckX64PlatformObjectABISmoke(tgt, deps); err != nil {
				t.Fatalf("x64 platform object smoke: %v", err)
			}
		})
	}

	t.Run("linux x64 pointer FFI regression", func(t *testing.T) {
		if err := CheckX64PointerFFIRegressionSmoke(deps); err != nil {
			t.Fatalf("x64 pointer FFI regression: %v", err)
		}
	})
}

func TestX64RuntimeExecutionSmokesUseCallbacks(t *testing.T) {
	var built []string
	var ran []string
	deps := RuntimeSmokeDeps{
		HostGOOS:   "linux",
		HostGOARCH: "amd64",
		BuildExecutable: func(srcPath string, outPath string, target string) error {
			built = append(built, filepath.Base(outPath)+":"+target)
			data := make([]byte, 20)
			copy(data[:4], "\x7fELF")
			data[4] = 2
			data[18] = 0x3e
			return os.WriteFile(outPath, data, 0o755)
		},
		RunExecutable: func(path string) RuntimeRunResult {
			ran = append(ran, filepath.Base(path))
			if strings.Contains(path, "scheduler-regression") {
				return RuntimeRunResult{ExitCode: 42}
			}
			return RuntimeRunResult{}
		},
	}

	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{name: "filesystem scheduler", run: func() error { return CheckX64FilesystemSchedulerCompositionSmoke(deps) }},
		{name: "networking", run: func() error { return CheckX64NetworkingRuntimeSmoke(deps) }},
		{name: "scheduler restriction", run: func() error { return CheckX64SchedulerRestrictionRegressionSmoke(deps) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err != nil {
				t.Fatalf("x64 runtime smoke: %v", err)
			}
		})
	}
	if len(built) != 3 {
		t.Fatalf("built %d executables, want 3: %#v", len(built), built)
	}
	if len(ran) != 2 {
		t.Fatalf("ran %d executables, want 2: %#v", len(ran), ran)
	}
}
