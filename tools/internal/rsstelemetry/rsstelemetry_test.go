package rsstelemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileAcceptsValidLinuxRSSSidecar(t *testing.T) {
	dir := t.TempDir()
	path := writeRSSSidecar(t, dir, "rss.json", validRSSSampleMap())

	sample, err := ReadFile(path, dir)
	if err != nil {
		t.Fatalf("ReadFile valid sidecar: %v", err)
	}
	if sample.Schema != Schema {
		t.Fatalf("schema = %q, want %q", sample.Schema, Schema)
	}
	if sample.Method != MethodLinuxProcfsWait4RSSSamplerV1 {
		t.Fatalf("method = %q, want %q", sample.Method, MethodLinuxProcfsWait4RSSSamplerV1)
	}
	if sample.RSSCurrentBytes != 4096 || sample.RSSPeakBytes != 8192 || sample.SampleCount != 2 {
		t.Fatalf("unexpected RSS counters: %+v", sample)
	}
}

func TestReadFileAcceptsPeakOnlySidecarWithoutCurrentSample(t *testing.T) {
	dir := t.TempDir()
	raw := validRSSSampleMap()
	raw["sample_count"] = 0
	raw["rss_current_bytes"] = 0
	raw["samples"] = []map[string]any{}
	path := writeRSSSidecar(t, dir, "rss.json", raw)

	sample, err := ReadFile(path, dir)
	if err != nil {
		t.Fatalf("ReadFile peak-only sidecar: %v", err)
	}
	if sample.SampleCount != 0 || sample.RSSPeakBytes == 0 {
		t.Fatalf("sample = %+v, want peak-only sample", sample)
	}
}

func TestReadFileRejectsInvalidMethodSchemaAndOS(t *testing.T) {
	for _, tc := range []struct {
		name  string
		field string
		value any
		want  string
	}{
		{name: "heap schema", field: "schema", value: "tetra.runtime.heap_telemetry.v1", want: "schema"},
		{name: "memstats method", field: "method", value: "MemStats", want: "method"},
		{name: "allocation method", field: "method", value: "allocation_report_summary", want: "method"},
		{
			name:  "heap method",
			field: "method",
			value: "tetra_linux_x64_heap_telemetry_v1",
			want:  "method",
		},
		{name: "target os", field: "target_os", value: "darwin", want: "target_os"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			raw := validRSSSampleMap()
			raw[tc.field] = tc.value
			path := writeRSSSidecar(t, dir, "rss.json", raw)

			_, err := ReadFile(path, dir)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ReadFile error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestReadFileRejectsImpossibleRSSInvariants(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(map[string]any)
		want string
	}{
		{
			name: "peak below current",
			edit: func(raw map[string]any) {
				raw["rss_current_bytes"] = 8192
				raw["rss_peak_bytes"] = 4096
			},
			want: "rss_peak_bytes",
		},
		{
			name: "current without sample",
			edit: func(raw map[string]any) {
				raw["sample_count"] = 0
				raw["rss_current_bytes"] = 4096
				raw["samples"] = []map[string]any{}
			},
			want: "sample_count",
		},
		{
			name: "finished before started",
			edit: func(raw map[string]any) {
				raw["started_unix_nano"] = int64(200)
				raw["finished_unix_nano"] = int64(100)
			},
			want: "finished_unix_nano",
		},
		{
			name: "zero sample rss",
			edit: func(raw map[string]any) {
				raw["samples"] = []map[string]any{{"unix_nano": int64(101), "rss_bytes": uint64(0)}}
			},
			want: "samples",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			raw := validRSSSampleMap()
			tc.edit(raw)
			path := writeRSSSidecar(t, dir, "rss.json", raw)

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
	path := writeRSSSidecar(t, outside, "rss.json", validRSSSampleMap())

	_, err := ReadFile(path, dir)
	if err == nil || !strings.Contains(err.Error(), "artifact root") {
		t.Fatalf("ReadFile outside root = %v, want artifact root rejection", err)
	}
}

func validRSSSampleMap() map[string]any {
	return map[string]any{
		"schema":                 Schema,
		"method":                 MethodLinuxProcfsWait4RSSSamplerV1,
		"program":                "allocation_tetra",
		"pid":                    1234,
		"target_os":              TargetOSLinux,
		"target_arch":            "amd64",
		"started_unix_nano":      int64(100),
		"finished_unix_nano":     int64(200),
		"exit_status":            0,
		"sample_interval_micros": uint64(500),
		"sample_count":           uint64(2),
		"rss_current_bytes":      uint64(4096),
		"rss_peak_bytes":         uint64(8192),
		"rss_peak_source":        PeakSourceWait4RusageMaxRSS,
		"ru_maxrss_raw":          uint64(8),
		"ru_maxrss_unit":         UnitKilobytes,
		"samples": []map[string]any{
			{"unix_nano": int64(110), "rss_bytes": uint64(4096)},
			{"unix_nano": int64(120), "rss_bytes": uint64(4096)},
		},
		"notes": []string{"test fixture"},
	}
}

func writeRSSSidecar(t *testing.T, dir string, name string, raw map[string]any) string {
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
