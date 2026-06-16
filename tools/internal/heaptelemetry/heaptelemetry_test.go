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
	if sample.HeapPeakBytes != 64 || sample.HeapTotalAllocBytes != 96 || sample.HeapAllocationCount != 2 {
		t.Fatalf("unexpected heap counters: %+v", sample)
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

func TestReadFileRejectsArtifactOutsideRoot(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	path := writeSidecar(t, outside, "heap.json", validSampleMap())

	_, err := ReadFile(path, dir)
	if err == nil || !strings.Contains(err.Error(), "artifact root") {
		t.Fatalf("ReadFile outside root = %v, want artifact root rejection", err)
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
		"domain_bytes":           []map[string]any{{"domain_id": "domain:process", "kind": "process", "current_bytes": 32, "peak_bytes": 64}},
		"notes":                  []string{"test fixture"},
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
