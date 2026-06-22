package heaptelemetry

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	compiler "tetra_language/compiler"
	"tetra_language/tools/internal/rsstelemetry"
)

func TestCompiledOwnedAllocDropWritesV2ReleaseCountersAndRSSSamples(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("linux-x64 runtime heap telemetry release smoke requires linux/amd64 host")
	}

	dir := t.TempDir()
	srcPath := filepath.Join(dir, "main.tetra")
	outPath := filepath.Join(dir, "owned-release")
	telemetryDir := filepath.Join(dir, "heap-telemetry")
	if err := os.MkdirAll(telemetryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := `
func release_once() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(16)
        let _stored: Int = core.store_i32(p, 7, mem)
    return 0

func main() -> Int
uses alloc, capability, mem:
    let _released: Int = release_once()
    var i: Int = 0
    var acc: Int = 0
    while i < 20000000:
        acc = acc + 1
        i = i + 1
    if acc < 0:
        return 1
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "linux-x64", compiler.BuildOptions{
		Jobs:                     1,
		EmitRuntimeHeapTelemetry: true,
		RuntimeHeapTelemetryDir:  telemetryDir,
		OwnedAllocDropLowering:   true,
	}); err != nil {
		t.Fatalf("BuildFileWithStatsOpt telemetry: %v", err)
	}
	rssSample := runWithRSSSamples(t, outPath)
	if rssSample.SampleCount == 0 {
		t.Fatalf("RSS sample_count = 0, want OS-side process samples")
	}
	if rssSample.MappingCount == nil || *rssSample.MappingCount == 0 {
		t.Fatalf("RSS mapping_count = %v, want linux procfs mapping count", rssSample.MappingCount)
	}
	if err := rsstelemetry.Validate(rssSample); err != nil {
		t.Fatalf("Validate generated RSS sample: %v\n%+v", err, rssSample)
	}

	matches, err := filepath.Glob(filepath.Join(telemetryDir, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("heap telemetry sidecars = %d (%v), want 1", len(matches), matches)
	}
	sample, err := ReadFile(matches[0], telemetryDir)
	if err != nil {
		t.Fatalf("ReadFile generated v2 release sidecar: %v", err)
	}
	if sample.Schema != SchemaV2 || sample.Method != MethodLinuxX64HeapTelemetryV2 {
		t.Fatalf("sidecar identity = %s/%s, want v2", sample.Schema, sample.Method)
	}
	if sample.HeapAllocationCount != 1 || sample.FreeCount != 1 {
		t.Fatalf(
			"allocation/free counters = %d/%d, want 1/1",
			sample.HeapAllocationCount,
			sample.FreeCount,
		)
	}
	if sample.SuccessfulAllocPayloadBytes == nil || *sample.SuccessfulAllocPayloadBytes != 16 {
		t.Fatalf("successful_alloc_payload_bytes = %v, want 16", sample.SuccessfulAllocPayloadBytes)
	}
	if sample.SuccessfulDropPayloadBytes == nil || *sample.SuccessfulDropPayloadBytes != 16 {
		t.Fatalf("successful_drop_payload_bytes = %v, want 16", sample.SuccessfulDropPayloadBytes)
	}
	if sample.PayloadLiveCurrentBytes == nil || *sample.PayloadLiveCurrentBytes != 0 {
		t.Fatalf("payload_live_current_bytes = %v, want 0", sample.PayloadLiveCurrentBytes)
	}
	if sample.OSReleaseAttemptCount == nil || *sample.OSReleaseAttemptCount != 1 {
		t.Fatalf("os_release_attempt_count = %v, want 1", sample.OSReleaseAttemptCount)
	}
	if sample.OSReleaseSuccessCount == nil || *sample.OSReleaseSuccessCount != 1 {
		t.Fatalf("os_release_success_count = %v, want 1", sample.OSReleaseSuccessCount)
	}
	if sample.ReleasedTotalBytes == nil || *sample.ReleasedTotalBytes == 0 {
		t.Fatalf("released_total_bytes = %v, want non-zero", sample.ReleasedTotalBytes)
	}
	if sample.OSReleaseSuccessBytes == nil ||
		*sample.OSReleaseSuccessBytes < *sample.ReleasedTotalBytes {
		t.Fatalf(
			"os_release_success_bytes = %v below released_total_bytes %v",
			sample.OSReleaseSuccessBytes,
			sample.ReleasedTotalBytes,
		)
	}
}

func runWithRSSSamples(t *testing.T, outPath string) rsstelemetry.Sample {
	t.Helper()
	cmd := exec.Command(outPath)
	started := time.Now()
	if err := cmd.Start(); err != nil {
		t.Fatalf("start telemetry binary: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	sample := rsstelemetry.Sample{
		Schema:               rsstelemetry.SchemaV2,
		Method:               rsstelemetry.MethodLinuxProcfsPhaseRSSSamplerV2,
		Program:              filepath.Base(outPath),
		PID:                  cmd.Process.Pid,
		TargetOS:             runtime.GOOS,
		TargetArch:           runtime.GOARCH,
		StartedUnixNano:      started.UnixNano(),
		WorkloadKind:         "owned_alloc_release",
		SampleIntervalMicros: 500,
		RSSPeakSource:        "procfs_phase_samples_max_v2",
		Notes: []string{
			"OS-side samples are captured while main spins after release_once returns",
		},
	}
	ticker := time.NewTicker(500 * time.Microsecond)
	defer ticker.Stop()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	phase := 0
	for {
		select {
		case err := <-done:
			sample.FinishedUnixNano = time.Now().UnixNano()
			if err != nil {
				t.Fatalf("run telemetry binary: %v", err)
			}
			return sample
		case <-timeout.C:
			_ = cmd.Process.Kill()
			t.Fatalf("run telemetry binary timed out")
		case now := <-ticker.C:
			rssBytes, ok := rsstelemetry.ReadProcessRSSBytes(cmd.Process.Pid)
			if !ok || rssBytes == 0 {
				continue
			}
			mappingCount, ok := rsstelemetry.ReadProcessMappingCount(cmd.Process.Pid)
			if !ok || mappingCount == 0 {
				continue
			}
			phase++
			sample.SampleCount++
			sample.RSSCurrentBytes = rssBytes
			sample.MappingCount = &mappingCount
			if sample.RSSPeakBytes < rssBytes {
				sample.RSSPeakBytes = rssBytes
			}
			sample.Samples = append(sample.Samples, rsstelemetry.RSSSample{
				Phase:        "post_release_spin",
				UnixNano:     now.UnixNano() + int64(phase),
				RSSBytes:     rssBytes,
				MappingCount: &mappingCount,
			})
		}
	}
}
