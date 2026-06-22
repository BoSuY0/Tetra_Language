package heaptelemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileAcceptsValidLinuxX64Sidecar(t *testing.T) {
	dir := t.TempDir()
	path := writeSidecar(t, dir, "heap.json", validSampleMap())

	sample, err := ReadFile(path, dir)
	if err != nil {
		t.Fatalf("ReadFile valid sidecar: %v", err)
	}
	if sample.Schema != Schema {
		t.Fatalf("schema = %q, want %q", sample.Schema, Schema)
	}
	if sample.Method != MethodLinuxX64HeapTelemetryV1 {
		t.Fatalf("method = %q, want %q", sample.Method, MethodLinuxX64HeapTelemetryV1)
	}
	if sample.HeapPeakBytes != 64 || sample.HeapTotalAllocBytes != 96 ||
		sample.HeapAllocationCount != 2 {
		t.Fatalf("unexpected heap counters: %+v", sample)
	}
}

func TestReadFileParsesDomainBytes(t *testing.T) {
	dir := t.TempDir()
	raw := validSampleMap()
	raw["domain_bytes"] = []map[string]any{{
		"domain_id":             "actor:pong",
		"kind":                  "actor",
		"requested_bytes":       256,
		"reserved_bytes":        512,
		"committed_bytes":       384,
		"current_bytes":         128,
		"peak_bytes":            192,
		"bytes_copied":          64,
		"mailbox_current_bytes": 64,
		"mailbox_peak_bytes":    96,
		"stack_live_bytes":      64,
		"stack_reserved_bytes":  96,
		"stack_retained_bytes":  0,
		"stack_released_bytes":  0,
		"byte_budget":           1024,
		"over_budget_count":     3,
		"backpressure_events":   4,
	}}
	path := writeSidecar(t, dir, "heap.json", raw)

	sample, err := ReadFile(path, dir)
	if err != nil {
		t.Fatalf("ReadFile domain sidecar: %v", err)
	}
	if len(sample.DomainBytes) != 1 {
		t.Fatalf("domain bytes = %#v, want one actor domain", sample.DomainBytes)
	}
	domain := sample.DomainBytes[0]
	if domain.DomainID != "actor:pong" || domain.Kind != "actor" ||
		domain.RequestedBytes != 256 || domain.ReservedBytes != 512 ||
		domain.CommittedBytes != 384 || domain.CurrentBytes != 128 ||
		domain.PeakBytes != 192 || domain.BytesCopied != 64 ||
		domain.MailboxCurrentBytes != 64 || domain.MailboxPeakBytes != 96 ||
		domain.StackLiveBytes != 64 || domain.StackReservedBytes != 96 ||
		domain.StackRetainedBytes != 0 || domain.StackReleasedBytes != 0 ||
		domain.ByteBudget != 1024 || domain.OverBudgetCount != 3 ||
		domain.BackpressureEvents != 4 {
		t.Fatalf("domain bytes = %+v, want all sidecar fields preserved", domain)
	}
}

func TestReadFileRejectsActorDomainMissingBudgetBackpressureFields(t *testing.T) {
	dir := t.TempDir()
	raw := validSampleMap()
	raw["domain_bytes"] = []map[string]any{{
		"domain_id":     "actor:pong",
		"kind":          "actor",
		"current_bytes": 128,
		"peak_bytes":    192,
		"bytes_copied":  64,
	}}
	path := writeSidecar(t, dir, "heap.json", raw)

	_, err := ReadFile(path, dir)
	if err == nil || !strings.Contains(err.Error(), "mailbox_current_bytes") {
		t.Fatalf("ReadFile actor domain without budget fields = %v, want mailbox field rejection", err)
	}
}

func TestReadFileRejectsActorDomainMissingStackFields(t *testing.T) {
	dir := t.TempDir()
	raw := validSampleMap()
	raw["domain_bytes"] = []map[string]any{{
		"domain_id":             "actor:pong",
		"kind":                  "actor",
		"current_bytes":         128,
		"peak_bytes":            192,
		"bytes_copied":          64,
		"mailbox_current_bytes": 128,
		"mailbox_peak_bytes":    192,
		"byte_budget":           1024,
		"over_budget_count":     3,
		"backpressure_events":   4,
	}}
	path := writeSidecar(t, dir, "heap.json", raw)

	_, err := ReadFile(path, dir)
	if err == nil || !strings.Contains(err.Error(), "stack_live_bytes") {
		t.Fatalf("ReadFile actor domain without stack fields = %v, want stack field rejection", err)
	}
}

func TestReadFileRejectsInvalidMethodAndTarget(t *testing.T) {
	for _, tc := range []struct {
		name  string
		field string
		value any
		want  string
	}{
		{name: "memstats", field: "method", value: "MemStats", want: "method"},
		{name: "allocation report", field: "method", value: "allocation_report_summary", want: "method"},
		{name: "rss", field: "method", value: "linux_proc_status", want: "method"},
		{name: "target", field: "target", value: "wasm32-wasi", want: "target"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			raw := validSampleMap()
			raw[tc.field] = tc.value
			path := writeSidecar(t, dir, "heap.json", raw)

			_, err := ReadFile(path, dir)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ReadFile error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestReadFileRejectsImpossibleByteInvariants(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(map[string]any)
		want string
	}{
		{
			name: "peak below current",
			edit: func(raw map[string]any) {
				raw["heap_current_bytes"] = 128
				raw["heap_peak_bytes"] = 64
			},
			want: "heap_peak_bytes",
		},
		{
			name: "total below peak",
			edit: func(raw map[string]any) {
				raw["heap_peak_bytes"] = 128
				raw["heap_total_alloc_bytes"] = 64
			},
			want: "heap_total_alloc_bytes",
		},
		{
			name: "heap bytes without allocation count",
			edit: func(raw map[string]any) {
				raw["heap_current_bytes"] = 0
				raw["heap_peak_bytes"] = 64
				raw["heap_total_alloc_bytes"] = 64
				raw["heap_allocation_count"] = 0
			},
			want: "heap_allocation_count",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			raw := validSampleMap()
			tc.edit(raw)
			path := writeSidecar(t, dir, "heap.json", raw)

			_, err := ReadFile(path, dir)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ReadFile error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestReadFileV2RejectsReleaseWithoutSuccessfulOSRelease(t *testing.T) {
	dir := t.TempDir()
	raw := validV2SampleMap()
	raw["released_total_bytes"] = uint64(4096)
	raw["os_release_success_count"] = uint64(0)
	raw["os_release_success_bytes"] = uint64(0)
	path := writeSidecar(t, dir, "heap-v2.json", raw)

	_, err := ReadFile(path, dir)
	if err == nil || !strings.Contains(err.Error(), "released_total_bytes") {
		t.Fatalf("ReadFile v2 release without OS release = %v, want release rejection", err)
	}
}

func TestReadFileV2RejectsLivePayloadThatDoesNotReconcile(t *testing.T) {
	dir := t.TempDir()
	raw := validV2SampleMap()
	raw["successful_alloc_payload_bytes"] = uint64(1024)
	raw["successful_drop_payload_bytes"] = uint64(256)
	raw["payload_live_current_bytes"] = uint64(1024)
	path := writeSidecar(t, dir, "heap-v2.json", raw)

	_, err := ReadFile(path, dir)
	if err == nil || !strings.Contains(err.Error(), "payload_live_current_bytes") {
		t.Fatalf("ReadFile v2 unreconciled live payload = %v, want current/live rejection", err)
	}
}

func TestReadFileV2RejectsPerCoreClaimForProcessGlobalAllocator(t *testing.T) {
	dir := t.TempDir()
	raw := validV2SampleMap()
	raw["allocator_claims"] = []string{"per_core", "free_list_reuse"}
	raw["allocator_state_scope"] = "process"
	path := writeSidecar(t, dir, "heap-v2.json", raw)

	_, err := ReadFile(path, dir)
	if err == nil || !strings.Contains(err.Error(), "per_core") {
		t.Fatalf("ReadFile v2 process-global per-core claim = %v, want per_core rejection", err)
	}
}

func TestReadFileV2RejectsMeasuredFieldsFromEstimates(t *testing.T) {
	dir := t.TempDir()
	raw := validV2SampleMap()
	raw["metric_sources"] = map[string]string{
		"payload_live_current_bytes": "allocation_plan_estimate",
		"released_total_bytes":       "runtime_measured",
	}
	path := writeSidecar(t, dir, "heap-v2.json", raw)

	_, err := ReadFile(path, dir)
	if err == nil || !strings.Contains(err.Error(), "allocation_plan_estimate") {
		t.Fatalf("ReadFile v2 estimated measured field = %v, want provenance rejection", err)
	}
}

func TestReadFileV2AcceptsUnsupportedMetricWithoutNumericZero(t *testing.T) {
	dir := t.TempDir()
	raw := validV2SampleMap()
	raw["unsupported_metrics"] = []string{"os_release_success_bytes"}
	delete(raw, "os_release_success_bytes")
	delete(raw, "os_release_success_count")
	delete(raw, "released_total_bytes")
	path := writeSidecar(t, dir, "heap-v2.json", raw)

	sample, err := ReadFile(path, dir)
	if err != nil {
		t.Fatalf("ReadFile v2 unsupported metric without numeric zero: %v", err)
	}
	if sample.Schema != SchemaV2 {
		t.Fatalf("schema = %q, want %q", sample.Schema, SchemaV2)
	}
}

func TestReadFileRejectsArtifactOutsideRoot(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	path := writeSidecar(t, outside, "heap.json", validSampleMap())

	_, err := ReadFile(path, dir)
	if err == nil || !strings.Contains(err.Error(), "artifact root") {
		t.Fatalf("ReadFile outside root = %v, want artifact root rejection", err)
	}
}

func validV2SampleMap() map[string]any {
	return map[string]any{
		"schema":                         SchemaV2,
		"target":                         TargetLinuxX64,
		"method":                         MethodLinuxX64HeapTelemetryV2,
		"program":                        "allocation_tetra",
		"pid":                            1234,
		"exit_status":                    0,
		"allocator_mode":                 "process_bump_small_heap_v0",
		"allocator_state_scope":          "process",
		"successful_alloc_payload_bytes": uint64(1024),
		"successful_drop_payload_bytes":  uint64(256),
		"payload_live_current_bytes":     uint64(768),
		"heap_allocation_count":          uint64(4),
		"free_count":                     uint64(1),
		"reuse_count":                    uint64(0),
		"released_total_bytes":           uint64(0),
		"os_release_attempt_count":       uint64(0),
		"os_release_success_count":       uint64(0),
		"os_release_success_bytes":       uint64(0),
		"bytes_requested":                uint64(1024),
		"bytes_reserved":                 uint64(4096),
		"metric_sources": map[string]string{
			"payload_live_current_bytes": "runtime_measured",
			"released_total_bytes":       "runtime_measured",
			"bytes_reserved":             "runtime_measured",
		},
		"allocation_paths": map[string]uint64{"small_heap_bump": 4},
		"notes":            []string{"test fixture"},
	}
}

func validSampleMap() map[string]any {
	return map[string]any{
		"schema":                 Schema,
		"target":                 "linux-x64",
		"method":                 MethodLinuxX64HeapTelemetryV1,
		"program":                "allocation_tetra",
		"pid":                    1234,
		"exit_status":            0,
		"heap_current_bytes":     32,
		"heap_peak_bytes":        64,
		"heap_total_alloc_bytes": 96,
		"heap_allocation_count":  2,
		"bytes_requested":        96,
		"bytes_reserved":         4096,
		"allocation_paths":       map[string]uint64{"small_heap_bump": 1, "large_mmap": 1},
		"domain_bytes": []map[string]any{
			{
				"domain_id":     "domain:process",
				"kind":          "process",
				"current_bytes": 32,
				"peak_bytes":    64,
			},
		},
		"notes": []string{"test fixture"},
	}
}

func writeSidecar(t *testing.T, dir string, name string, raw map[string]any) string {
	t.Helper()
	path := filepath.Join(dir, name)
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal sidecar: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}
	return path
}
