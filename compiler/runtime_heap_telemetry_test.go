package compiler

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLinuxX64RuntimeHeapTelemetrySidecarFromCompiledBinary(t *testing.T) {
	sidecar := buildRunReadHeapTelemetrySidecar(t, `func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return 0
`, "heap-smoke")
	requireHeapTelemetrySidecarIdentity(t, sidecar, "heap-smoke")
	if _, ok := sidecar["heap_allocation_count"].(float64); !ok {
		t.Fatalf("sidecar heap_allocation_count = %#v, want numeric", sidecar["heap_allocation_count"])
	}
	if _, ok := sidecar["heap_peak_bytes"].(float64); !ok {
		t.Fatalf("sidecar heap_peak_bytes = %#v, want numeric", sidecar["heap_peak_bytes"])
	}
}

func TestLinuxX64RuntimeHeapTelemetryReportsZeroForStackMakeSlice(t *testing.T) {
	sidecar := buildRunReadHeapTelemetrySidecar(t, `func main() -> Int
uses alloc, mem:
    let n: Int = 4
    var xs: []i32 = core.make_i32(n)
    var i: Int = 0
    while i < n:
        xs[i] = i + 1
        i = i + 1
    if xs[0] == 1:
        return 0
    return 1
`, "heap-stack-smoke")
	requireHeapTelemetrySidecarIdentity(t, sidecar, "heap-stack-smoke")
	if got, ok := sidecar["heap_allocation_count"].(float64); !ok || got != 0 {
		t.Fatalf("sidecar heap_allocation_count = %#v, want 0 for stack-backed allocation", sidecar["heap_allocation_count"])
	}
	if got, ok := sidecar["heap_peak_bytes"].(float64); !ok || got != 0 {
		t.Fatalf("sidecar heap_peak_bytes = %#v, want 0 for stack-backed allocation", sidecar["heap_peak_bytes"])
	}
}

func buildRunReadHeapTelemetrySidecar(t *testing.T, src string, outputName string) map[string]any {
	t.Helper()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux-x64 runtime heap telemetry smoke requires linux/amd64 host")
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	outPath := filepath.Join(dir, outputName)
	telemetryDir := filepath.Join(dir, "heap-telemetry")
	if err := os.MkdirAll(telemetryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", BuildOptions{
		Jobs:                     1,
		EmitRuntimeHeapTelemetry: true,
		RuntimeHeapTelemetryDir:  telemetryDir,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt telemetry: %v", err)
	}
	if out, err := exec.Command(outPath).CombinedOutput(); err != nil {
		t.Fatalf("run telemetry binary: %v\n%s", err, string(out))
	}
	matches, err := filepath.Glob(filepath.Join(telemetryDir, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("heap telemetry sidecars = %d (%v), want 1", len(matches), matches)
	}
	raw, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("read sidecar: %v", err)
	}
	var sidecar map[string]any
	if err := json.Unmarshal(raw, &sidecar); err != nil {
		t.Fatalf("sidecar JSON: %v\n%s", err, string(raw))
	}
	return sidecar
}

func requireHeapTelemetrySidecarIdentity(t *testing.T, sidecar map[string]any, program string) {
	t.Helper()
	if sidecar["schema"] != "tetra.runtime.heap_telemetry.v1" {
		t.Fatalf("sidecar schema = %#v", sidecar["schema"])
	}
	if sidecar["method"] != "tetra_linux_x64_heap_telemetry_v1" {
		t.Fatalf("sidecar method = %#v", sidecar["method"])
	}
	if sidecar["program"] != program {
		t.Fatalf("sidecar program = %#v, want %q", sidecar["program"], program)
	}
}
