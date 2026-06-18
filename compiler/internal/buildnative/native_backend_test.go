package buildnative

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/buildapi"
	"tetra_language/compiler/internal/format/tobj"
	ctarget "tetra_language/compiler/target"
)

func mustTarget(t *testing.T, name string) ctarget.Target {
	t.Helper()
	tgt, err := ctarget.Parse(name)
	if err != nil {
		t.Fatalf("Parse(%q): %v", name, err)
	}
	return tgt
}

func TestExecutableBackendForTarget(t *testing.T) {
	linuxX64 := mustTarget(t, "linux-x64")
	backend, ok := ExecutableBackendForTarget(linuxX64)
	if !ok {
		t.Fatalf("ExecutableBackendForTarget(linux-x64) ok = false")
	}
	if backend.Name != "linux-x64" || backend.OS != ctarget.OSLinux ||
		backend.Format != ctarget.FormatELF {
		t.Fatalf(
			"linux-x64 backend = (%s, %s, %s), want linux-x64/linux/elf",
			backend.Name,
			backend.OS,
			backend.Format,
		)
	}
	if backend.Codegen == nil || backend.Link == nil || backend.ActorRuntime == nil {
		t.Fatalf("linux-x64 backend missing codegen/link/actor runtime")
	}

	linuxX86 := mustTarget(t, "linux-x86")
	backend, ok = ExecutableBackendForTarget(linuxX86)
	if !ok {
		t.Fatalf("ExecutableBackendForTarget(linux-x86) ok = false")
	}
	if backend.Name != "linux-x86" || backend.Codegen == nil || backend.Link == nil {
		t.Fatalf("linux-x86 backend is incomplete: %#v", backend)
	}

	wasm := mustTarget(t, "wasm32-wasi")
	if _, ok := ExecutableBackendForTarget(wasm); ok {
		t.Fatalf("ExecutableBackendForTarget(wasm32-wasi) ok = true, want false")
	}
}

func TestCodegenOptionsForTarget(t *testing.T) {
	tgt := mustTarget(t, "linux-x32")
	got := CodegenOptionsForTarget(tgt, buildapi.BuildOptions{
		IslandsDebug:                true,
		DebugInfo:                   true,
		ReleaseOptimize:             true,
		EmitRuntimeHeapTelemetry:    true,
		RuntimeHeapTelemetryDir:     "telemetry",
		RuntimeHeapTelemetryProgram: "prog",
		RuntimeHeapTelemetryMain:    "main",
	})

	if !got.IslandsDebug || !got.DebugInfo || !got.ReleaseOptimize ||
		!got.EmitRuntimeHeapTelemetry {
		t.Fatalf("CodegenOptionsForTarget did not preserve build flags: %#v", got)
	}
	if got.PointerWidthBits != tgt.PointerWidthBits ||
		got.NativeIntWidthBits != tgt.NativeIntWidthBits ||
		got.RegisterWidthBits != tgt.RegisterWidthBits {
		t.Fatalf("CodegenOptionsForTarget widths = (%d,%d,%d), want (%d,%d,%d)",
			got.PointerWidthBits, got.NativeIntWidthBits, got.RegisterWidthBits,
			tgt.PointerWidthBits, tgt.NativeIntWidthBits, tgt.RegisterWidthBits)
	}
	if got.RuntimeHeapTelemetryDir != "telemetry" || got.RuntimeHeapTelemetryProgram != "prog" ||
		got.RuntimeHeapTelemetryMain != "main" {
		t.Fatalf("CodegenOptionsForTarget telemetry fields = %#v", got)
	}
}

func TestValidateRuntimeHeapTelemetryBuildOptions(t *testing.T) {
	linuxX64 := mustTarget(t, "linux-x64")
	err := ValidateRuntimeHeapTelemetryBuildOptions(
		linuxX64,
		buildapi.BuildOptions{RuntimeHeapTelemetryDir: "telemetry"},
	)
	if err == nil || !strings.Contains(err.Error(), "requires EmitRuntimeHeapTelemetry") {
		t.Fatalf("telemetry dir without flag error = %v, want flag diagnostic", err)
	}

	err = ValidateRuntimeHeapTelemetryBuildOptions(linuxX64, buildapi.BuildOptions{
		EmitRuntimeHeapTelemetry: true,
		RuntimeHeapTelemetryDir:  "telemetry",
	})
	if err != nil {
		t.Fatalf("ValidateRuntimeHeapTelemetryBuildOptions(linux-x64) error = %v", err)
	}

	linuxX86 := mustTarget(t, "linux-x86")
	err = ValidateRuntimeHeapTelemetryBuildOptions(linuxX86, buildapi.BuildOptions{
		EmitRuntimeHeapTelemetry: true,
		RuntimeHeapTelemetryDir:  "telemetry",
	})
	if err == nil || !strings.Contains(err.Error(), "only supported for linux-x64") {
		t.Fatalf("telemetry on linux-x86 error = %v, want target diagnostic", err)
	}
}

func TestAppendLinkedObjectsPreservesOrder(t *testing.T) {
	base := []*tobj.Object{{Module: "main"}}
	linked := []*tobj.Object{{Module: "iface-a"}, {Module: "iface-b"}}

	got := AppendLinkedObjects(base, linked)
	if len(got) != 3 {
		t.Fatalf("AppendLinkedObjects len = %d, want 3", len(got))
	}
	wantModules := []string{"main", "iface-a", "iface-b"}
	for i, want := range wantModules {
		if got[i].Module != want {
			t.Fatalf("AppendLinkedObjects[%d].Module = %q, want %q", i, got[i].Module, want)
		}
	}
}

func TestLinkExecutableDispatchesBackend(t *testing.T) {
	objects := []*tobj.Object{{Module: "main"}}
	var gotPath, gotMain string
	var gotObjects []*tobj.Object
	backend := ExecutableBackend{
		Link: func(outputPath string, objects []*tobj.Object, mainName string) error {
			gotPath = outputPath
			gotObjects = objects
			gotMain = mainName
			return nil
		},
	}

	if err := LinkExecutable("out.bin", "linux-x64", backend, objects, "main"); err != nil {
		t.Fatalf("LinkExecutable error = %v", err)
	}
	if gotPath != "out.bin" || gotMain != "main" || len(gotObjects) != 1 ||
		gotObjects[0] != objects[0] {
		t.Fatalf(
			"LinkExecutable dispatch = path=%q main=%q objects=%v",
			gotPath,
			gotMain,
			gotObjects,
		)
	}

	err := LinkExecutable("out.bin", "linux-x64", ExecutableBackend{}, objects, "main")
	if err == nil || !strings.Contains(err.Error(), "target backend has no linker: linux-x64") {
		t.Fatalf("missing linker error = %v, want target diagnostic", err)
	}
}
