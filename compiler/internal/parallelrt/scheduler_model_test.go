package parallelrt

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"tetra_language/compiler/internal/stdlibrt"
)

func TestSchedulerModelRunsSingleCoreFIFO(t *testing.T) {
	model, err := NewSchedulerModel(Config{Cores: 1, MailboxCapacity: 2})
	if err != nil {
		t.Fatal(err)
	}
	model.Enqueue(0, WorkItem{Name: "first"})
	model.Enqueue(0, WorkItem{Name: "second"})

	stats, err := model.RunUntilIdle()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := model.RunLog(), []string{"core0:first", "core0:second"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("RunLog() = %#v, want %#v", got, want)
	}
	if stats.Cores != 1 || stats.Steals != 0 || stats.Completed != 2 {
		t.Fatalf("stats = %#v, want single-core no-steal completion", stats)
	}
}

func TestSchedulerModelStealsWorkAcrossTwoCores(t *testing.T) {
	model, err := NewSchedulerModel(Config{Cores: 2, MailboxCapacity: 2})
	if err != nil {
		t.Fatal(err)
	}
	model.Enqueue(0, WorkItem{Name: "left"})
	model.Enqueue(0, WorkItem{Name: "right"})

	if ran, err := model.StepCore(1); err != nil || !ran {
		t.Fatalf("StepCore(1) = %v, %v; want stolen work", ran, err)
	}
	if got, want := model.RunLog(), []string{"core1:right"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("RunLog() = %#v, want %#v", got, want)
	}
	stats := model.Stats()
	if stats.Steals != 1 || stats.Completed != 1 || stats.MaxQueueDepth != 2 {
		t.Fatalf("stats = %#v, want one steal from two-item queue", stats)
	}
}

func TestTypedMailboxPreservesCapacityBackpressureAndOwnershipMetadata(t *testing.T) {
	box := NewTypedMailbox(MailboxConfig{
		Name:     "telemetry",
		Capacity: 1,
		Backpressure: BackpressurePolicy{
			Mode: "blocking_recv_yield",
		},
	})
	first := Message{
		Name: "reading",
		Payloads: []Payload{{
			Name:      "value",
			Kind:      PayloadScalar,
			SizeBytes: 8,
		}},
	}
	if report, err := box.Send(first); err != nil {
		t.Fatalf("first Send failed: %v", err)
	} else if report.TransferMode != TransferCopy || report.BytesCopied != 8 || report.ZeroCopy {
		t.Fatalf("first transfer report = %#v, want scalar copy metadata", report)
	}

	_, err := box.Send(first)
	if !errors.Is(err, ErrMailboxFull) {
		t.Fatalf("second Send error = %v, want ErrMailboxFull", err)
	}
	if box.Capacity() != 1 || box.Backpressure().Mode != "blocking_recv_yield" || !box.HasOwnershipMetadata() {
		t.Fatalf("mailbox metadata mismatch: capacity=%d backpressure=%#v ownership=%v", box.Capacity(), box.Backpressure(), box.HasOwnershipMetadata())
	}
}

func TestOwnedRegionMessageMovesZeroCopyAndBorrowedPayloadRequiresCopy(t *testing.T) {
	model, err := NewSchedulerModel(Config{Cores: 2, MailboxCapacity: 4})
	if err != nil {
		t.Fatal(err)
	}
	model.RegisterRegion(Region{Name: "frame", Owner: "sender"})

	zeroCopy, err := model.Send(0, 1, Message{
		Name: "Frame.region",
		Payloads: []Payload{{
			Name:       "bytes",
			Kind:       PayloadOwnedRegion,
			RegionName: "frame",
			Owner:      "sender",
			SizeBytes:  4096,
		}},
	})
	if err != nil {
		t.Fatalf("owned region Send failed: %v", err)
	}
	if zeroCopy.TransferMode != TransferZeroCopyMove || zeroCopy.BytesCopied != 0 || !zeroCopy.ZeroCopy {
		t.Fatalf("zero-copy transfer report = %#v", zeroCopy)
	}
	if got := model.RegionOwner("frame"); got != "actor1" {
		t.Fatalf("RegionOwner(frame) = %q, want actor1", got)
	}

	_, err = model.Send(1, 0, Message{
		Name: "Frame.view",
		Payloads: []Payload{{
			Name:      "view",
			Kind:      PayloadBorrowedView,
			SizeBytes: 128,
		}},
	})
	if !errors.Is(err, ErrBorrowedPayloadRequiresCopy) {
		t.Fatalf("borrowed Send error = %v, want ErrBorrowedPayloadRequiresCopy", err)
	}

	copied, err := model.Send(1, 0, Message{
		Name: "Frame.viewCopy",
		Payloads: []Payload{{
			Name:         "view",
			Kind:         PayloadBorrowedView,
			SizeBytes:    128,
			ExplicitCopy: true,
		}},
	})
	if err != nil {
		t.Fatalf("borrowed copied Send failed: %v", err)
	}
	if copied.TransferMode != TransferCopy || copied.BytesCopied != 128 || copied.ZeroCopy {
		t.Fatalf("borrowed copied transfer report = %#v, want copy", copied)
	}
}

func TestPrototypeBenchmarksReportFanoutAndZeroCopyRows(t *testing.T) {
	rows, err := PrototypeBenchmarks()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 5 {
		t.Fatalf("PrototypeBenchmarks returned %d rows, want 5", len(rows))
	}
	byName := map[string]PrototypeBenchmark{}
	for _, row := range rows {
		byName[row.Name] = row
		if row.ClaimTier != "tier0_local_smoke_only" || row.Ran || !row.Pass {
			t.Fatalf("row %s tier/ran/pass = %q/%v/%v, want Tier 0 dry-run pass", row.Name, row.ClaimTier, row.Ran, row.Pass)
		}
		if row.ImprovementRatio != 0 || row.BaselineValue != 0 || row.MeasuredValue != 0 {
			t.Fatalf("row %s recorded measured values in Tier 0 prep: %#v", row.Name, row)
		}
		if len(row.RawOutputArtifacts) == 0 {
			t.Fatalf("row %s missing raw output artifact refs", row.Name)
		}
	}
	for _, want := range []string{
		"actor ping-pong benchmark prep",
		"actor fanout/fanin benchmark prep",
		"actor mailbox throughput benchmark prep",
		"actor backpressure latency benchmark prep",
		"zero_copy_move local typed mailbox benchmark prep",
	} {
		if _, ok := byName[want]; !ok {
			t.Fatalf("missing benchmark prep row %q: %#v", want, rows)
		}
	}
	fanout := byName["actor fanout/fanin benchmark prep"]
	if fanout.Kind != "actor_benchmark_prep" || !strings.Contains(fanout.Evidence, "work stealing") {
		t.Fatalf("fanout benchmark = %#v, want work-stealing Tier 0 prep row", fanout)
	}
	zeroCopy := byName["zero_copy_move local typed mailbox benchmark prep"]
	if zeroCopy.Kind != "actor_transfer_prep" || !strings.Contains(zeroCopy.Evidence, "zero_copy_move") || strings.Contains(strings.ToLower(zeroCopy.Claim), "production runtime claim") {
		t.Fatalf("zero-copy benchmark = %#v, want scoped zero_copy_move prep nonclaim", zeroCopy)
	}
}

func TestTaskRegionScopeInjectsRegionAndResetsAfterTask(t *testing.T) {
	scope := NewTaskRegionScope(TaskRegionOptions{RegionID: "task:decode", Capacity: 128})
	var report TaskRegionReport
	var err error

	allocs := testing.AllocsPerRun(1000, func() {
		report, err = scope.Run("decode", func(region *stdlibrt.Region) error {
			if region == nil || region.ID() != "task:decode" {
				return errors.New("missing task region injection")
			}
			_, allocErr := region.Alloc(32)
			return allocErr
		})
	})
	if err != nil {
		t.Fatalf("task region run: %v", err)
	}
	if allocs != 0 {
		t.Fatalf("task region allocations = %.2f, want 0", allocs)
	}
	if report.RegionID != "task:decode" || report.Lifetime != "task:decode" || !report.Reset {
		t.Fatalf("task region report = %#v, want task lifetime reset", report)
	}
	if report.BytesUsedBeforeReset != 32 || scope.RegionUsed() != 0 {
		t.Fatalf("task region reset evidence = used_before=%d used_after=%d", report.BytesUsedBeforeReset, scope.RegionUsed())
	}
}
