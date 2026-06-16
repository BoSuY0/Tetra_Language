package rsstelemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	Schema                             = "tetra.local_benchmark.process_rss_telemetry.v1"
	MethodLinuxProcfsWait4RSSSamplerV1 = "linux_procfs_wait4_rss_sampler_v1"
	MethodLinuxProcfsStatusVmRSSV1     = "linux_procfs_status_vmrss_v1"
	MethodLinuxWait4RusageMaxRSSV1     = "linux_wait4_rusage_maxrss_v1"
	TargetOSLinux                      = "linux"
	PeakSourceWait4RusageMaxRSS        = "wait4_rusage_maxrss"
	UnitKilobytes                      = "kilobytes"
)

type Sample struct {
	Schema               string      `json:"schema"`
	Method               string      `json:"method"`
	Program              string      `json:"program"`
	PID                  int         `json:"pid,omitempty"`
	TargetOS             string      `json:"target_os"`
	TargetArch           string      `json:"target_arch,omitempty"`
	StartedUnixNano      int64       `json:"started_unix_nano,omitempty"`
	FinishedUnixNano     int64       `json:"finished_unix_nano,omitempty"`
	ExitStatus           int         `json:"exit_status"`
	SampleIntervalMicros uint64      `json:"sample_interval_micros,omitempty"`
	SampleCount          uint64      `json:"sample_count"`
	RSSCurrentBytes      uint64      `json:"rss_current_bytes"`
	RSSPeakBytes         uint64      `json:"rss_peak_bytes"`
	RSSPeakSource        string      `json:"rss_peak_source,omitempty"`
	RUMaxRSSRaw          uint64      `json:"ru_maxrss_raw,omitempty"`
	RUMaxRSSUnit         string      `json:"ru_maxrss_unit,omitempty"`
	Samples              []RSSSample `json:"samples,omitempty"`
	Notes                []string    `json:"notes,omitempty"`
}

type RSSSample struct {
	UnixNano int64  `json:"unix_nano,omitempty"`
	RSSBytes uint64 `json:"rss_bytes"`
}

func ReadFile(path string, artifactRoot string) (Sample, error) {
	if err := requirePathInsideRoot(path, artifactRoot); err != nil {
		return Sample{}, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return Sample{}, err
	}
	var sample Sample
	if err := json.Unmarshal(raw, &sample); err != nil {
		return Sample{}, fmt.Errorf("RSS telemetry sidecar JSON: %w", err)
	}
	if err := Validate(sample); err != nil {
		return Sample{}, err
	}
	return sample, nil
}

func Validate(sample Sample) error {
	if sample.Schema != Schema {
		return fmt.Errorf("RSS telemetry schema = %q, want %q", sample.Schema, Schema)
	}
	if sample.Method != MethodLinuxProcfsWait4RSSSamplerV1 {
		return fmt.Errorf("RSS telemetry method = %q, want %q", sample.Method, MethodLinuxProcfsWait4RSSSamplerV1)
	}
	if sample.TargetOS != TargetOSLinux {
		return fmt.Errorf("RSS telemetry target_os = %q, want %q", sample.TargetOS, TargetOSLinux)
	}
	if strings.TrimSpace(sample.Program) == "" {
		return fmt.Errorf("RSS telemetry program is required")
	}
	if sample.PID < 0 {
		return fmt.Errorf("RSS telemetry pid = %d, want non-negative", sample.PID)
	}
	if sample.ExitStatus < 0 {
		return fmt.Errorf("RSS telemetry exit_status = %d, want non-negative", sample.ExitStatus)
	}
	if sample.StartedUnixNano != 0 && sample.FinishedUnixNano != 0 && sample.FinishedUnixNano < sample.StartedUnixNano {
		return fmt.Errorf("RSS telemetry finished_unix_nano = %d before started_unix_nano = %d", sample.FinishedUnixNano, sample.StartedUnixNano)
	}
	if sample.SampleCount == 0 && sample.RSSCurrentBytes != 0 {
		return fmt.Errorf("RSS telemetry sample_count is zero but rss_current_bytes = %d", sample.RSSCurrentBytes)
	}
	if sample.SampleCount > 0 && sample.RSSCurrentBytes == 0 {
		return fmt.Errorf("RSS telemetry sample_count = %d but rss_current_bytes is zero", sample.SampleCount)
	}
	if sample.SampleCount > 0 && sample.RSSPeakBytes < sample.RSSCurrentBytes {
		return fmt.Errorf("RSS telemetry rss_peak_bytes = %d below rss_current_bytes = %d", sample.RSSPeakBytes, sample.RSSCurrentBytes)
	}
	if sample.RSSPeakBytes > 0 {
		if strings.TrimSpace(sample.RSSPeakSource) == "" {
			return fmt.Errorf("RSS telemetry rss_peak_source is required when rss_peak_bytes is non-zero")
		}
		if sample.RUMaxRSSRaw > 0 && sample.RUMaxRSSUnit != UnitKilobytes {
			return fmt.Errorf("RSS telemetry ru_maxrss_unit = %q, want %q", sample.RUMaxRSSUnit, UnitKilobytes)
		}
	}
	for i, point := range sample.Samples {
		if point.RSSBytes == 0 {
			return fmt.Errorf("RSS telemetry samples[%d] rss_bytes is zero", i)
		}
	}
	return nil
}

func requirePathInsideRoot(path string, artifactRoot string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("RSS telemetry sidecar path is required")
	}
	if strings.TrimSpace(artifactRoot) == "" {
		return nil
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("RSS telemetry sidecar path: %w", err)
	}
	absRoot, err := filepath.Abs(artifactRoot)
	if err != nil {
		return fmt.Errorf("RSS telemetry artifact root: %w", err)
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return fmt.Errorf("RSS telemetry artifact root: %w", err)
	}
	if rel == "." || rel == "" {
		return nil
	}
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
		return fmt.Errorf("RSS telemetry sidecar %s is outside artifact root %s", path, artifactRoot)
	}
	return nil
}
