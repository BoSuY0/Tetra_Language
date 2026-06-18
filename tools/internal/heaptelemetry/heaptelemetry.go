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
	MethodLinuxX64HeapTelemetryV1 = "tetra_linux_x64_heap_telemetry_v1"
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
	_, hasByteBudget := raw["byte_budget"]
	_, hasOverBudget := raw["over_budget_count"]
	_, hasBackpressure := raw["backpressure_events"]
	d.ActorDomainFieldsSet = hasMailboxCurrent && hasMailboxPeak &&
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
	if sample.Schema != Schema {
		return fmt.Errorf("heap telemetry schema = %q, want %q", sample.Schema, Schema)
	}
	if sample.Target != TargetLinuxX64 {
		return fmt.Errorf("heap telemetry target = %q, want %q", sample.Target, TargetLinuxX64)
	}
	if sample.Method != MethodLinuxX64HeapTelemetryV1 {
		return fmt.Errorf(
			"heap telemetry method = %q, want %q",
			sample.Method,
			MethodLinuxX64HeapTelemetryV1,
		)
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
						"mailbox_current_bytes/mailbox_peak_bytes/byte_budget/" +
						"over_budget_count/backpressure_events"),
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
			if domain.CurrentBytes != domain.MailboxCurrentBytes {
				return fmt.Errorf(
					"heap telemetry domain_bytes[%d] current_bytes = %d, want mailbox_current_bytes %d",
					i,
					domain.CurrentBytes,
					domain.MailboxCurrentBytes,
				)
			}
			if domain.PeakBytes != domain.MailboxPeakBytes {
				return fmt.Errorf(
					"heap telemetry domain_bytes[%d] peak_bytes = %d, want mailbox_peak_bytes %d",
					i,
					domain.PeakBytes,
					domain.MailboxPeakBytes,
				)
			}
			if domain.ByteBudget == 0 {
				return fmt.Errorf("heap telemetry domain_bytes[%d] byte_budget is required", i)
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
	return nil
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
