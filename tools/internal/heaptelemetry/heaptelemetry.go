package heaptelemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	Schema                        = "tetra.runtime.heap_telemetry.v1"
	SchemaV2                      = "tetra.runtime.heap_telemetry.v2"
	MethodLinuxX64HeapTelemetryV1 = "tetra_linux_x64_heap_telemetry_v1"
	MethodLinuxX64HeapTelemetryV2 = "tetra_linux_x64_heap_telemetry_v2"
	TargetLinuxX64                = "linux-x64"
)

type Sample struct {
	Schema              string            `json:"schema"`
	Target              string            `json:"target"`
	Method              string            `json:"method"`
	Program             string            `json:"program"`
	PID                 int               `json:"pid,omitempty"`
	StartedUnixNano     int64             `json:"started_unix_nano,omitempty"`
	FinishedUnixNano    int64             `json:"finished_unix_nano,omitempty"`
	ExitStatus          int               `json:"exit_status"`
	HeapCurrentBytes    uint64            `json:"heap_current_bytes"`
	HeapPeakBytes       uint64            `json:"heap_peak_bytes"`
	HeapTotalAllocBytes uint64            `json:"heap_total_alloc_bytes"`
	HeapAllocationCount uint64            `json:"heap_allocation_count"`
	BytesRequested      uint64            `json:"bytes_requested"`
	BytesReserved       uint64            `json:"bytes_reserved"`
	AllocationPaths     map[string]uint64 `json:"allocation_paths,omitempty"`
	DomainBytes         []DomainBytes     `json:"domain_bytes,omitempty"`
	Notes               []string          `json:"notes,omitempty"`

	AllocatorMode                    string            `json:"allocator_mode,omitempty"`
	AllocatorStateScope              string            `json:"allocator_state_scope,omitempty"`
	AllocatorClaims                  []string          `json:"allocator_claims,omitempty"`
	SuccessfulAllocPayloadBytes      *uint64           `json:"successful_alloc_payload_bytes,omitempty"`
	SuccessfulDropPayloadBytes       *uint64           `json:"successful_drop_payload_bytes,omitempty"`
	PayloadTransferCurrentDeltaBytes int64             `json:"payload_transfer_current_delta_bytes,omitempty"`
	PayloadLiveCurrentBytes          *uint64           `json:"payload_live_current_bytes,omitempty"`
	FreeCount                        uint64            `json:"free_count,omitempty"`
	ReuseCount                       uint64            `json:"reuse_count,omitempty"`
	ReleasedTotalBytes               *uint64           `json:"released_total_bytes,omitempty"`
	OSReleaseAttemptCount            *uint64           `json:"os_release_attempt_count,omitempty"`
	OSReleaseSuccessCount            *uint64           `json:"os_release_success_count,omitempty"`
	OSReleaseSuccessBytes            *uint64           `json:"os_release_success_bytes,omitempty"`
	MetricSources                    map[string]string `json:"metric_sources,omitempty"`
	UnsupportedMetrics               []string          `json:"unsupported_metrics,omitempty"`
	NotSampledMetrics                []string          `json:"not_sampled_metrics,omitempty"`
}

type DomainBytes struct {
	DomainID             string `json:"domain_id"`
	Kind                 string `json:"kind"`
	RequestedBytes       uint64 `json:"requested_bytes,omitempty"`
	ReservedBytes        uint64 `json:"reserved_bytes,omitempty"`
	CommittedBytes       uint64 `json:"committed_bytes,omitempty"`
	CurrentBytes         uint64 `json:"current_bytes,omitempty"`
	PeakBytes            uint64 `json:"peak_bytes,omitempty"`
	BytesCopied          uint64 `json:"bytes_copied,omitempty"`
	MailboxCurrentBytes  uint64 `json:"mailbox_current_bytes,omitempty"`
	MailboxPeakBytes     uint64 `json:"mailbox_peak_bytes,omitempty"`
	StackLiveBytes       uint64 `json:"stack_live_bytes,omitempty"`
	StackReservedBytes   uint64 `json:"stack_reserved_bytes,omitempty"`
	StackRetainedBytes   uint64 `json:"stack_retained_bytes,omitempty"`
	StackReleasedBytes   uint64 `json:"stack_released_bytes,omitempty"`
	ByteBudget           uint64 `json:"byte_budget,omitempty"`
	OverBudgetCount      uint64 `json:"over_budget_count,omitempty"`
	BackpressureEvents   uint64 `json:"backpressure_events,omitempty"`
	ActorDomainFieldsSet bool   `json:"-"`
}

func (d *DomainBytes) UnmarshalJSON(data []byte) error {
	type domainBytesAlias DomainBytes
	var alias domainBytesAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*d = DomainBytes(alias)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	_, hasMailboxCurrent := raw["mailbox_current_bytes"]
	_, hasMailboxPeak := raw["mailbox_peak_bytes"]
	_, hasStackLive := raw["stack_live_bytes"]
	_, hasStackReserved := raw["stack_reserved_bytes"]
	_, hasStackRetained := raw["stack_retained_bytes"]
	_, hasStackReleased := raw["stack_released_bytes"]
	_, hasByteBudget := raw["byte_budget"]
	_, hasOverBudget := raw["over_budget_count"]
	_, hasBackpressure := raw["backpressure_events"]
	d.ActorDomainFieldsSet = hasMailboxCurrent && hasMailboxPeak &&
		hasStackLive && hasStackReserved && hasStackRetained && hasStackReleased &&
		hasByteBudget && hasOverBudget && hasBackpressure
	return nil
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
		return Sample{}, fmt.Errorf("heap telemetry sidecar JSON: %w", err)
	}
	if err := Validate(sample); err != nil {
		return Sample{}, err
	}
	return sample, nil
}

func Validate(sample Sample) error {
	switch sample.Schema {
	case Schema:
		if sample.Method != MethodLinuxX64HeapTelemetryV1 {
			return fmt.Errorf(
				"heap telemetry method = %q, want %q",
				sample.Method,
				MethodLinuxX64HeapTelemetryV1,
			)
		}
	case SchemaV2:
		if sample.Method != MethodLinuxX64HeapTelemetryV2 {
			return fmt.Errorf(
				"heap telemetry method = %q, want %q",
				sample.Method,
				MethodLinuxX64HeapTelemetryV2,
			)
		}
	default:
		return fmt.Errorf(
			"heap telemetry schema = %q, want %q or %q",
			sample.Schema,
			Schema,
			SchemaV2,
		)
	}
	if sample.Target != TargetLinuxX64 {
		return fmt.Errorf("heap telemetry target = %q, want %q", sample.Target, TargetLinuxX64)
	}
	if strings.TrimSpace(sample.Program) == "" {
		return fmt.Errorf("heap telemetry program is required")
	}
	if sample.PID < 0 {
		return fmt.Errorf("heap telemetry pid = %d, want non-negative", sample.PID)
	}
	if sample.ExitStatus < 0 {
		return fmt.Errorf("heap telemetry exit_status = %d, want non-negative", sample.ExitStatus)
	}
	if sample.HeapPeakBytes < sample.HeapCurrentBytes {
		return fmt.Errorf(
			"heap telemetry heap_peak_bytes = %d below heap_current_bytes = %d",
			sample.HeapPeakBytes,
			sample.HeapCurrentBytes,
		)
	}
	if sample.HeapTotalAllocBytes < sample.HeapPeakBytes {
		return fmt.Errorf(
			"heap telemetry heap_total_alloc_bytes = %d below heap_peak_bytes = %d",
			sample.HeapTotalAllocBytes,
			sample.HeapPeakBytes,
		)
	}
	if sample.HeapAllocationCount == 0 &&
		(sample.HeapCurrentBytes != 0 || sample.HeapPeakBytes != 0 || sample.HeapTotalAllocBytes != 0) {
		return fmt.Errorf(
			"heap telemetry heap_allocation_count is zero but heap byte totals are non-zero",
		)
	}
	if sample.BytesReserved != 0 && sample.BytesReserved < sample.HeapPeakBytes {
		return fmt.Errorf(
			"heap telemetry bytes_reserved = %d below heap_peak_bytes = %d",
			sample.BytesReserved,
			sample.HeapPeakBytes,
		)
	}
	for path, count := range sample.AllocationPaths {
		if strings.TrimSpace(path) == "" {
			return fmt.Errorf("heap telemetry allocation_paths contains an empty path")
		}
		if count == 0 {
			return fmt.Errorf("heap telemetry allocation_paths[%q] has zero count", path)
		}
	}
	for i, domain := range sample.DomainBytes {
		if strings.TrimSpace(domain.DomainID) == "" {
			return fmt.Errorf("heap telemetry domain_bytes[%d] missing domain_id", i)
		}
		if strings.TrimSpace(domain.Kind) == "" {
			return fmt.Errorf("heap telemetry domain_bytes[%d] missing kind", i)
		}
		if domain.PeakBytes < domain.CurrentBytes {
			return fmt.Errorf(
				"heap telemetry domain_bytes[%d] peak_bytes = %d below current_bytes = %d",
				i,
				domain.PeakBytes,
				domain.CurrentBytes,
			)
		}
		if domain.Kind == "actor" {
			if !domain.ActorDomainFieldsSet {
				return fmt.Errorf(
					("heap telemetry domain_bytes[%d] actor domain missing " +
						"mailbox_current_bytes/mailbox_peak_bytes/stack_live_bytes/" +
						"stack_reserved_bytes/stack_retained_bytes/stack_released_bytes/" +
						"byte_budget/over_budget_count/backpressure_events"),
					i,
				)
			}
			if domain.MailboxPeakBytes < domain.MailboxCurrentBytes {
				return fmt.Errorf(
					("heap telemetry domain_bytes[%d] mailbox_peak_bytes = %d below " +
						"mailbox_current_bytes = %d"),
					i,
					domain.MailboxPeakBytes,
					domain.MailboxCurrentBytes,
				)
			}
			if domain.StackReservedBytes < domain.StackLiveBytes {
				return fmt.Errorf(
					("heap telemetry domain_bytes[%d] stack_reserved_bytes = %d below " +
						"stack_live_bytes = %d"),
					i,
					domain.StackReservedBytes,
					domain.StackLiveBytes,
				)
			}
			if domain.StackReservedBytes < domain.StackRetainedBytes {
				return fmt.Errorf(
					("heap telemetry domain_bytes[%d] stack_reserved_bytes = %d below " +
						"stack_retained_bytes = %d"),
					i,
					domain.StackReservedBytes,
					domain.StackRetainedBytes,
				)
			}
			stackAccounted, ok := checkedAddUint64(domain.StackLiveBytes, domain.StackRetainedBytes)
			if !ok || stackAccounted > domain.StackReservedBytes {
				return fmt.Errorf(
					("heap telemetry domain_bytes[%d] stack_live_bytes + " +
						"stack_retained_bytes exceeds stack_reserved_bytes"),
					i,
				)
			}
			expectedCurrent, ok := checkedAddUint64(
				domain.MailboxCurrentBytes,
				domain.StackLiveBytes,
			)
			if !ok || domain.CurrentBytes != expectedCurrent {
				return fmt.Errorf(
					("heap telemetry domain_bytes[%d] current_bytes = %d, want " +
						"mailbox_current_bytes + stack_live_bytes %d"),
					i,
					domain.CurrentBytes,
					expectedCurrent,
				)
			}
			minPeak, ok := checkedAddUint64(domain.MailboxPeakBytes, domain.StackReservedBytes)
			if !ok || domain.PeakBytes < minPeak {
				return fmt.Errorf(
					("heap telemetry domain_bytes[%d] peak_bytes = %d below " +
						"mailbox_peak_bytes + stack_reserved_bytes %d"),
					i,
					domain.PeakBytes,
					minPeak,
				)
			}
			if domain.ByteBudget == 0 {
				return fmt.Errorf("heap telemetry domain_bytes[%d] byte_budget is required", i)
			}
			if domain.ByteBudget < domain.StackReservedBytes {
				return fmt.Errorf(
					"heap telemetry domain_bytes[%d] byte_budget = %d below stack_reserved_bytes %d",
					i,
					domain.ByteBudget,
					domain.StackReservedBytes,
				)
			}
			if domain.OverBudgetCount > domain.BackpressureEvents {
				return fmt.Errorf(
					("heap telemetry domain_bytes[%d] over_budget_count = %d above " +
						"backpressure_events = %d"),
					i,
					domain.OverBudgetCount,
					domain.BackpressureEvents,
				)
			}
		}
	}
	if sample.Schema == SchemaV2 {
		if err := validateV2(sample); err != nil {
			return err
		}
	}
	return nil
}

func checkedAddUint64(a uint64, b uint64) (uint64, bool) {
	if a > ^uint64(0)-b {
		return 0, false
	}
	return a + b, true
}

func validateV2(sample Sample) error {
	if strings.TrimSpace(sample.AllocatorMode) == "" {
		return fmt.Errorf("heap telemetry v2 allocator_mode is required")
	}
	if strings.TrimSpace(sample.AllocatorStateScope) == "" {
		return fmt.Errorf("heap telemetry v2 allocator_state_scope is required")
	}
	if perCoreClaimed(sample) && sample.AllocatorStateScope == "process" {
		return fmt.Errorf(
			"heap telemetry v2 per_core allocator claim contradicts process allocator_state_scope",
		)
	}
	if err := validateMetricAbsence(sample); err != nil {
		return err
	}
	if err := requireRuntimeMeasuredSource(sample, "payload_live_current_bytes"); err != nil {
		return err
	}
	if err := requireRuntimeMeasuredSource(sample, "released_total_bytes"); err != nil {
		return err
	}
	if sample.PayloadLiveCurrentBytes == nil {
		return fmt.Errorf("heap telemetry v2 payload_live_current_bytes is required")
	}
	if sample.SuccessfulAllocPayloadBytes == nil {
		return fmt.Errorf("heap telemetry v2 successful_alloc_payload_bytes is required")
	}
	if sample.SuccessfulDropPayloadBytes == nil {
		return fmt.Errorf("heap telemetry v2 successful_drop_payload_bytes is required")
	}
	expectedLive := int64(*sample.SuccessfulAllocPayloadBytes) -
		int64(*sample.SuccessfulDropPayloadBytes) +
		sample.PayloadTransferCurrentDeltaBytes
	if expectedLive < 0 {
		return fmt.Errorf("heap telemetry v2 payload lifecycle counters reconcile below zero")
	}
	if *sample.PayloadLiveCurrentBytes != uint64(expectedLive) {
		return fmt.Errorf(
			("heap telemetry v2 payload_live_current_bytes = %d, want successful_alloc_payload_bytes " +
				"- successful_drop_payload_bytes + payload_transfer_current_delta_bytes = %d"),
			*sample.PayloadLiveCurrentBytes,
			expectedLive,
		)
	}
	if sample.ReleasedTotalBytes != nil && *sample.ReleasedTotalBytes > 0 {
		if sample.OSReleaseSuccessCount == nil || *sample.OSReleaseSuccessCount == 0 {
			return fmt.Errorf(
				"heap telemetry v2 released_total_bytes = %d but os_release_success_count is zero or absent",
				*sample.ReleasedTotalBytes,
			)
		}
		if sample.OSReleaseSuccessBytes == nil || *sample.OSReleaseSuccessBytes < *sample.ReleasedTotalBytes {
			return fmt.Errorf(
				"heap telemetry v2 released_total_bytes = %d exceeds os_release_success_bytes",
				*sample.ReleasedTotalBytes,
			)
		}
	}
	if sample.OSReleaseAttemptCount != nil && sample.OSReleaseSuccessCount != nil &&
		*sample.OSReleaseSuccessCount > *sample.OSReleaseAttemptCount {
		return fmt.Errorf(
			"heap telemetry v2 os_release_success_count = %d above os_release_attempt_count = %d",
			*sample.OSReleaseSuccessCount,
			*sample.OSReleaseAttemptCount,
		)
	}
	if sample.FreeCount > sample.HeapAllocationCount {
		return fmt.Errorf(
			"heap telemetry v2 free_count = %d above heap_allocation_count = %d",
			sample.FreeCount,
			sample.HeapAllocationCount,
		)
	}
	return nil
}

func perCoreClaimed(sample Sample) bool {
	if strings.Contains(strings.ToLower(sample.AllocatorMode), "per_core") {
		return true
	}
	for _, claim := range sample.AllocatorClaims {
		if strings.Contains(strings.ToLower(claim), "per_core") {
			return true
		}
	}
	return false
}

func validateMetricAbsence(sample Sample) error {
	for _, metric := range append(sample.UnsupportedMetrics, sample.NotSampledMetrics...) {
		if v2MetricPresent(sample, metric) {
			return fmt.Errorf(
				"heap telemetry v2 metric %q is marked unsupported/not sampled but has a numeric value",
				metric,
			)
		}
	}
	return nil
}

func v2MetricPresent(sample Sample, metric string) bool {
	switch metric {
	case "payload_live_current_bytes":
		return sample.PayloadLiveCurrentBytes != nil
	case "successful_alloc_payload_bytes":
		return sample.SuccessfulAllocPayloadBytes != nil
	case "successful_drop_payload_bytes":
		return sample.SuccessfulDropPayloadBytes != nil
	case "released_total_bytes":
		return sample.ReleasedTotalBytes != nil
	case "os_release_attempt_count":
		return sample.OSReleaseAttemptCount != nil
	case "os_release_success_count":
		return sample.OSReleaseSuccessCount != nil
	case "os_release_success_bytes":
		return sample.OSReleaseSuccessBytes != nil
	default:
		return false
	}
}

func requireRuntimeMeasuredSource(sample Sample, metric string) error {
	if !v2MetricPresent(sample, metric) {
		return nil
	}
	source := strings.TrimSpace(sample.MetricSources[metric])
	switch source {
	case "runtime_measured", "os_measured":
		return nil
	case "":
		return fmt.Errorf("heap telemetry v2 metric_sources[%q] is required", metric)
	default:
		return fmt.Errorf(
			"heap telemetry v2 metric_sources[%q] = %q, want runtime_measured or os_measured",
			metric,
			source,
		)
	}
}

func requirePathInsideRoot(path string, artifactRoot string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("heap telemetry sidecar path is required")
	}
	if strings.TrimSpace(artifactRoot) == "" {
		return nil
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("heap telemetry sidecar path: %w", err)
	}
	absRoot, err := filepath.Abs(artifactRoot)
	if err != nil {
		return fmt.Errorf("heap telemetry artifact root: %w", err)
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return fmt.Errorf("heap telemetry artifact root: %w", err)
	}
	if rel == "." || rel == "" {
		return nil
	}
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." ||
		filepath.IsAbs(rel) {
		return fmt.Errorf(
			"heap telemetry sidecar %s is outside artifact root %s",
			path,
			artifactRoot,
		)
	}
	return nil
}
