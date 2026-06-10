package semantics

import (
	"fmt"
	"strings"
)

type MemoryBoundaryHandoffID string

const (
	MemoryBoundaryActorBorrowRejected       MemoryBoundaryHandoffID = "actor_borrow_rejected"
	MemoryBoundaryTaskBorrowRejected        MemoryBoundaryHandoffID = "task_borrow_rejected"
	MemoryBoundaryRequestRegionScoped       MemoryBoundaryHandoffID = "request_region_scoped"
	MemoryBoundaryUnsafeSafeMessageRejected MemoryBoundaryHandoffID = "unsafe_safe_message_rejected"
	MemoryBoundaryStaleEpochRejected        MemoryBoundaryHandoffID = "stale_epoch_rejected"
	MemoryBoundaryIslandMoveLinear          MemoryBoundaryHandoffID = "island_move_linear"
	MemoryBoundaryActorRuntimeNonClaim      MemoryBoundaryHandoffID = "actor_runtime_nonclaim"
)

type MemoryBoundaryHandoffStatus string

const (
	MemoryBoundaryImplementedNarrow MemoryBoundaryHandoffStatus = "implemented_narrow"
)

type MemoryBoundaryHandoffReport struct {
	SchemaVersion           string                     `json:"schema_version"`
	Rows                    []MemoryBoundaryHandoffRow `json:"rows"`
	NonClaims               []string                   `json:"non_claims"`
	FullActorRuntimeClaimed bool                       `json:"full_actor_runtime_claimed"`
}

type MemoryBoundaryHandoffRow struct {
	ID            MemoryBoundaryHandoffID     `json:"id"`
	Name          string                      `json:"name"`
	Status        MemoryBoundaryHandoffStatus `json:"status"`
	RequiredFacts []string                    `json:"required_facts"`
	Evidence      string                      `json:"evidence"`
	Boundary      string                      `json:"boundary"`
}

func MemoryBoundaryHandoffAudit() MemoryBoundaryHandoffReport {
	return MemoryBoundaryHandoffReport{
		SchemaVersion: "tetra.memory.boundary_handoff.v1",
		Rows: []MemoryBoundaryHandoffRow{
			actorBorrowRejectedRow(),
			taskBorrowRejectedRow(),
			requestRegionScopedRow(),
			unsafeSafeMessageRejectedRow(),
			staleEpochRejectedRow(),
			islandMoveLinearRow(),
			actorRuntimeNonClaimRow(),
		},
		NonClaims: []string{
			"full production actor runtime is not claimed",
			"request/task region reset evidence is local runtime scope evidence, not global escape analysis completeness",
			"unsafe send contracts remain checker-model evidence; safe actor/task messages reject raw unsafe payloads",
			"no distributed ownership protocol or benchmark superiority is claimed",
		},
		FullActorRuntimeClaimed: false,
	}
}

func ValidateMemoryBoundaryHandoffAudit(report MemoryBoundaryHandoffReport) error {
	if report.SchemaVersion != "tetra.memory.boundary_handoff.v1" {
		return fmt.Errorf("memory boundary handoff audit: schema = %q", report.SchemaVersion)
	}
	if report.FullActorRuntimeClaimed {
		return fmt.Errorf("memory boundary handoff audit: full production actor runtime claim is forbidden for P10")
	}
	if !containsMemoryBoundaryText(report.NonClaims, "full production actor runtime is not claimed") {
		return fmt.Errorf("memory boundary handoff audit: missing full production actor runtime nonclaim")
	}

	expected := map[MemoryBoundaryHandoffID]bool{
		MemoryBoundaryActorBorrowRejected:       false,
		MemoryBoundaryTaskBorrowRejected:        false,
		MemoryBoundaryRequestRegionScoped:       false,
		MemoryBoundaryUnsafeSafeMessageRejected: false,
		MemoryBoundaryStaleEpochRejected:        false,
		MemoryBoundaryIslandMoveLinear:          false,
		MemoryBoundaryActorRuntimeNonClaim:      false,
	}
	rows := map[MemoryBoundaryHandoffID]MemoryBoundaryHandoffRow{}
	for _, row := range report.Rows {
		if row.ID == "" {
			return fmt.Errorf("memory boundary handoff audit: row missing id")
		}
		if _, ok := expected[row.ID]; !ok {
			return fmt.Errorf("memory boundary handoff audit: unexpected row %q", row.ID)
		}
		if expected[row.ID] {
			return fmt.Errorf("memory boundary handoff audit: duplicate row %q", row.ID)
		}
		expected[row.ID] = true
		rows[row.ID] = row
		if row.Status != MemoryBoundaryImplementedNarrow {
			return fmt.Errorf("memory boundary handoff audit: row %q status = %q", row.ID, row.Status)
		}
		if strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Evidence) == "" || strings.TrimSpace(row.Boundary) == "" {
			return fmt.Errorf("memory boundary handoff audit: row %q missing evidence or boundary", row.ID)
		}
		if len(row.RequiredFacts) == 0 {
			return fmt.Errorf("memory boundary handoff audit: row %q missing required facts", row.ID)
		}
	}
	for id, seen := range expected {
		if !seen {
			return fmt.Errorf("memory boundary handoff audit: missing row %q", id)
		}
	}
	if len(report.Rows) != len(expected) {
		return fmt.Errorf("memory boundary handoff audit: row count = %d, want %d", len(report.Rows), len(expected))
	}

	if err := requireMemoryBoundaryFacts(rows[MemoryBoundaryActorBorrowRejected], "cannot send borrowed view across actor boundary", ".copy()"); err != nil {
		return err
	}
	if err := requireMemoryBoundaryFacts(rows[MemoryBoundaryTaskBorrowRejected], "typed task error payload must be sendable across task boundary"); err != nil {
		return err
	}
	if err := requireMemoryBoundaryFacts(rows[MemoryBoundaryRequestRegionScoped], "RequestRegionScope", "TaskRegionScope", "reset"); err != nil {
		return err
	}
	if err := requireMemoryBoundaryFacts(rows[MemoryBoundaryUnsafeSafeMessageRejected], "ptr", "cap.mem", "typed actor message payload must be value-only"); err != nil {
		return err
	}
	if err := requireMemoryBoundaryFacts(rows[MemoryBoundaryStaleEpochRejected], "core.island_reset", "cannot use consumed value"); err != nil {
		return err
	}
	if err := requireMemoryBoundaryFacts(rows[MemoryBoundaryIslandMoveLinear], "core.send_typed", "cannot use consumed value", "island"); err != nil {
		return err
	}
	if err := requireMemoryBoundaryFacts(rows[MemoryBoundaryActorRuntimeNonClaim], "not a production actor runtime"); err != nil {
		return err
	}
	return nil
}

func actorBorrowRejectedRow() MemoryBoundaryHandoffRow {
	return MemoryBoundaryHandoffRow{
		ID:     MemoryBoundaryActorBorrowRejected,
		Name:   "Actor borrowed payload rejection",
		Status: MemoryBoundaryImplementedNarrow,
		RequiredFacts: []string{
			"cannot send borrowed view across actor boundary",
			".copy() is required for borrowed slice/String actor payloads",
			"borrowed aggregate actor payloads reject unless explicitly copied",
		},
		Evidence: "compiler/tests/semantics/borrow_copy_test.go::TestBorrowedActorSendRejectedUnlessCopied; compiler/tests/semantics/memory_ideal_v4_boundary_test.go::TestMemoryIdealV4ActorBoundaryCopyAndBorrowDiagnostics; compiler/internal/semantics/exprs.go::validateActorBoundaryPayloadExpr",
		Boundary: "source semantics reject borrowed actor message payloads or require explicit copy; this is checker evidence, not production actor runtime evidence",
	}
}

func taskBorrowRejectedRow() MemoryBoundaryHandoffRow {
	return MemoryBoundaryHandoffRow{
		ID:     MemoryBoundaryTaskBorrowRejected,
		Name:   "Task borrowed payload rejection",
		Status: MemoryBoundaryImplementedNarrow,
		RequiredFacts: []string{
			"typed task error payload must be sendable across task boundary",
			"borrowed slice/String task error payloads reject",
			"copy before typed task boundary is accepted",
		},
		Evidence: "compiler/tests/semantics/borrow_copy_test.go::TestBorrowedTaskBoundaryTypedErrorPayloadRejected; compiler/tests/semantics/memory_ideal_v4_boundary_test.go::TestMemoryIdealV4TaskBoundaryCurrentSurfaceDiagnostics",
		Boundary: "current typed task surface has no arbitrary payload spawn API; evidence covers typed task error payloads and rejects reference-shaped task boundary payloads",
	}
}

func requestRegionScopedRow() MemoryBoundaryHandoffRow {
	return MemoryBoundaryHandoffRow{
		ID:     MemoryBoundaryRequestRegionScoped,
		Name:   "Request/task region scope reset",
		Status: MemoryBoundaryImplementedNarrow,
		RequiredFacts: []string{
			"RequestRegionScope injects and resets request region storage",
			"TaskRegionScope injects and resets task region storage",
			"reset prevents request/task region data from becoming a safe cross-boundary message by default",
		},
		Evidence: "docs/audits/request-task-region-v1.md; compiler/internal/httprt/request_region.go::RequestRegionScope; compiler/internal/parallelrt/task_region.go::TaskRegionScope; compiler/internal/httprt/request_view_test.go::TestRequestRegionScopeInjectsRegionForHTTPJSONAndResetsAfterWrite; compiler/internal/parallelrt/scheduler_model_test.go::TestTaskRegionScopeInjectsRegionAndResetsAfterTask",
		Boundary: "request/task region evidence is scoped runtime entry behavior and reset reporting; it is not a claim that arbitrary region-backed data may cross actor/task/request boundaries safely",
	}
}

func unsafeSafeMessageRejectedRow() MemoryBoundaryHandoffRow {
	return MemoryBoundaryHandoffRow{
		ID:     MemoryBoundaryUnsafeSafeMessageRejected,
		Name:   "Unsafe payload cannot become safe message",
		Status: MemoryBoundaryImplementedNarrow,
		RequiredFacts: []string{
			"ptr typed actor payload rejects with typed actor message payload must be value-only",
			"cap.mem typed actor payload rejects with typed actor message payload must be value-only",
			"unsafe send contracts are not safe typed actor message permission",
		},
		Evidence: "compiler/tests/safety/plan250_safety_runtime_test.go::TestPlan250SafetySendabilityAcrossModuleBoundaries; compiler/internal/semantics/exprs.go::validateTypedActorMessageType; compiler/internal/actorsafety/sendability_test.go::TestUnsafePointerRequiresExplicitUnsafeSendContract",
		Boundary: "raw unsafe payloads stay rejected for safe typed actor messages; internal unsafe-send contract model evidence does not expose a safe message surface",
	}
}

func staleEpochRejectedRow() MemoryBoundaryHandoffRow {
	return MemoryBoundaryHandoffRow{
		ID:     MemoryBoundaryStaleEpochRejected,
		Name:   "Stale epoch after reset rejection",
		Status: MemoryBoundaryImplementedNarrow,
		RequiredFacts: []string{
			"core.island_reset consumes the prior island handle",
			"cannot use consumed value after reset",
			"stale island handle cannot be sent across actor boundary after reset",
		},
		Evidence: "compiler/tests/runtime/resource_finalization_test.go::TestMemoryBoundaryHandoffRejectsStaleIslandAfterResetAcrossActorBoundary; compiler/tests/runtime/resource_finalization_test.go::TestIslandResetRejectsUseAfterReset; compiler/internal/semantics/exprs.go::consumeTypedActorTransferPayloads",
		Boundary: "stale epoch rejection is enforced as consumed-resource semantics before actor send; no live stale-epoch runtime sanitizer bypass is claimed",
	}
}

func islandMoveLinearRow() MemoryBoundaryHandoffRow {
	return MemoryBoundaryHandoffRow{
		ID:     MemoryBoundaryIslandMoveLinear,
		Name:   "Island move remains linear across actor boundary",
		Status: MemoryBoundaryImplementedNarrow,
		RequiredFacts: []string{
			"core.send_typed consumes island payloads",
			"cannot use consumed value after typed actor transfer",
			"island handle moved twice across actor boundary is rejected",
		},
		Evidence: "compiler/actors_test.go::TestActorsTypedMessagesIslandTransferConsumesSource; compiler/tests/runtime/resource_finalization_test.go::TestTypedActorTransferRejectsFieldAccessEnumPayloadAliasReuse; compiler/internal/semantics/exprs.go::consumeTypedActorTransferPayloads",
		Boundary: "linear transfer evidence covers current typed actor message payloads and source diagnostics; it is not a distributed ownership protocol or race-safety proof",
	}
}

func actorRuntimeNonClaimRow() MemoryBoundaryHandoffRow {
	return MemoryBoundaryHandoffRow{
		ID:     MemoryBoundaryActorRuntimeNonClaim,
		Name:   "Actor runtime production nonclaim",
		Status: MemoryBoundaryImplementedNarrow,
		RequiredFacts: []string{
			"not a production actor runtime",
			"full production actor runtime is not claimed",
			"actor/task/request boundary handoff does not start actor runtime implementation",
		},
		Evidence: "compiler/internal/actorsrt/production_boundary.go::ActorRuntimeProductionBoundaryAudit; docs/audits/actor-runtime-production-boundary-v1.md",
		Boundary: "P10 proves Memory/Islands boundary handoff only; actor production runtime remains a later plan with separate gates",
	}
}

func requireMemoryBoundaryFacts(row MemoryBoundaryHandoffRow, wants ...string) error {
	for _, want := range wants {
		if !containsMemoryBoundaryText(row.RequiredFacts, want) {
			return fmt.Errorf("memory boundary handoff audit: row %q missing fact %q", row.ID, want)
		}
	}
	return nil
}

func containsMemoryBoundaryText(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}
