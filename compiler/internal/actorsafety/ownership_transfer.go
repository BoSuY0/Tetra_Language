package actorsafety

import (
	"fmt"
	"strings"
)

type TypedActorOwnershipTransferID string

const (
	TypedActorOwnershipBorrowedViewCopyBoundary     TypedActorOwnershipTransferID = "borrowed_view_copy_boundary"
	TypedActorOwnershipOwnedRegionMove              TypedActorOwnershipTransferID = "owned_region_move"
	TypedActorOwnershipSenderUseAfterMove           TypedActorOwnershipTransferID = "sender_use_after_move"
	TypedActorOwnershipReceiverOwnsMovedRegion      TypedActorOwnershipTransferID = "receiver_owns_moved_region"
	TypedActorOwnershipExplicitCopyFallback         TypedActorOwnershipTransferID = "explicit_copy_fallback"
	TypedActorOwnershipUnsafeSendContract           TypedActorOwnershipTransferID = "unsafe_send_contract"
	TypedActorOwnershipSemanticsChecker             TypedActorOwnershipTransferID = "semantics_transfer_checker"
	TypedActorOwnershipPLIRMovedFacts               TypedActorOwnershipTransferID = "plir_moved_facts"
	TypedActorOwnershipRuntimeMailboxRepresentation TypedActorOwnershipTransferID = "runtime_mailbox_representation"
	TypedActorOwnershipActorTransferReport          TypedActorOwnershipTransferID = "actor_transfer_report"
	TypedActorOwnershipStressDiagnostics            TypedActorOwnershipTransferID = "stress_use_after_move_diagnostics"
)

type TypedActorOwnershipTransferStatus string

const (
	TypedActorOwnershipImplementedNarrow TypedActorOwnershipTransferStatus = "implemented_narrow"
)

type TypedActorOwnershipTransferReport struct {
	SchemaVersion              string                           `json:"schema_version"`
	Rows                       []TypedActorOwnershipTransferRow `json:"rows"`
	NonClaims                  []string                         `json:"non_claims"`
	DistributedZeroCopyClaimed bool                             `json:"distributed_zero_copy_claimed"`
	RuntimeBehaviorChanged     bool                             `json:"runtime_behavior_changed"`
}

type TypedActorOwnershipTransferRow struct {
	ID                          TypedActorOwnershipTransferID     `json:"id"`
	Name                        string                            `json:"name"`
	Status                      TypedActorOwnershipTransferStatus `json:"status"`
	RequiredFacts               []string                          `json:"required_facts,omitempty"`
	Evidence                    string                            `json:"evidence"`
	Boundary                    string                            `json:"boundary"`
	ClaimsDistributedZeroCopy   bool                              `json:"claims_distributed_zero_copy,omitempty"`
	ClaimsRuntimeBehaviorChange bool                              `json:"claims_runtime_behavior_change,omitempty"`
}

func TypedActorOwnershipTransferCoverage() TypedActorOwnershipTransferReport {
	return TypedActorOwnershipTransferReport{
		SchemaVersion: "tetra.actors.ownership_transfer.v1",
		Rows: []TypedActorOwnershipTransferRow{
			borrowedViewCopyBoundaryRow(),
			ownedRegionMoveRow(),
			senderUseAfterMoveRow(),
			receiverOwnsMovedRegionRow(),
			explicitCopyFallbackRow(),
			unsafeSendContractRow(),
			semanticsTransferCheckerRow(),
			plirMovedFactsRow(),
			runtimeMailboxRepresentationRow(),
			actorTransferReportRow(),
			stressDiagnosticsRow(),
		},
		NonClaims: []string{
			"distributed pointer or region zero-copy is not claimed",
			"P18.1 does not change actor runtime behavior",
			"safe typed actor raw pointer payloads remain rejected; unsafe send contracts are checker-model evidence only",
			"full production actor runtime is not claimed",
		},
		DistributedZeroCopyClaimed: false,
		RuntimeBehaviorChanged:     false,
	}
}

func ValidateTypedActorOwnershipTransferCoverage(report TypedActorOwnershipTransferReport) error {
	if report.SchemaVersion != "tetra.actors.ownership_transfer.v1" {
		return fmt.Errorf("typed actor ownership transfer: schema = %q", report.SchemaVersion)
	}
	if report.DistributedZeroCopyClaimed {
		return fmt.Errorf("typed actor ownership transfer: distributed zero-copy claim is forbidden for P18.1")
	}
	if report.RuntimeBehaviorChanged {
		return fmt.Errorf("typed actor ownership transfer: runtime behavior change claim is forbidden for P18.1")
	}
	if !containsOwnershipTransferText(report.NonClaims, "distributed pointer or region zero-copy is not claimed") {
		return fmt.Errorf("typed actor ownership transfer: missing distributed zero-copy non-claim")
	}
	if !containsOwnershipTransferText(report.NonClaims, "does not change actor runtime behavior") {
		return fmt.Errorf("typed actor ownership transfer: missing runtime behavior non-claim")
	}

	expected := map[TypedActorOwnershipTransferID]bool{
		TypedActorOwnershipBorrowedViewCopyBoundary:     false,
		TypedActorOwnershipOwnedRegionMove:              false,
		TypedActorOwnershipSenderUseAfterMove:           false,
		TypedActorOwnershipReceiverOwnsMovedRegion:      false,
		TypedActorOwnershipExplicitCopyFallback:         false,
		TypedActorOwnershipUnsafeSendContract:           false,
		TypedActorOwnershipSemanticsChecker:             false,
		TypedActorOwnershipPLIRMovedFacts:               false,
		TypedActorOwnershipRuntimeMailboxRepresentation: false,
		TypedActorOwnershipActorTransferReport:          false,
		TypedActorOwnershipStressDiagnostics:            false,
	}
	if len(report.Rows) != len(expected) {
		return fmt.Errorf("typed actor ownership transfer: row count = %d, want %d", len(report.Rows), len(expected))
	}
	rows := map[TypedActorOwnershipTransferID]TypedActorOwnershipTransferRow{}
	for _, row := range report.Rows {
		if row.ID == "" {
			return fmt.Errorf("typed actor ownership transfer: row missing id")
		}
		if _, ok := expected[row.ID]; !ok {
			return fmt.Errorf("typed actor ownership transfer: unexpected row %q", row.ID)
		}
		if expected[row.ID] {
			return fmt.Errorf("typed actor ownership transfer: duplicate row %q", row.ID)
		}
		expected[row.ID] = true
		rows[row.ID] = row
		if row.Status != TypedActorOwnershipImplementedNarrow {
			return fmt.Errorf("typed actor ownership transfer: row %q status = %q", row.ID, row.Status)
		}
		if strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Evidence) == "" || strings.TrimSpace(row.Boundary) == "" {
			return fmt.Errorf("typed actor ownership transfer: row %q missing evidence or boundary", row.ID)
		}
		if len(row.RequiredFacts) == 0 {
			return fmt.Errorf("typed actor ownership transfer: row %q (%s) missing required facts", row.ID, row.Name)
		}
		if row.ClaimsDistributedZeroCopy {
			return fmt.Errorf("typed actor ownership transfer: row %q claims distributed zero-copy", row.ID)
		}
		if row.ClaimsRuntimeBehaviorChange {
			return fmt.Errorf("typed actor ownership transfer: row %q claims runtime behavior change", row.ID)
		}
	}
	for id, seen := range expected {
		if !seen {
			return fmt.Errorf("typed actor ownership transfer: missing row %q", id)
		}
	}
	if err := requireOwnershipTransferFacts(rows[TypedActorOwnershipBorrowedViewCopyBoundary], "borrowed view rejected", "explicit copy accepted", ".copy()"); err != nil {
		return err
	}
	if err := requireOwnershipTransferFacts(rows[TypedActorOwnershipOwnedRegionMove], "owned region can move", "bytes_copied=0", "zero_copy_move"); err != nil {
		return err
	}
	if err := requireOwnershipTransferFacts(rows[TypedActorOwnershipSenderUseAfterMove], "cannot use consumed value", "cannot use moved region after send"); err != nil {
		return err
	}
	if err := requireOwnershipTransferFacts(rows[TypedActorOwnershipReceiverOwnsMovedRegion], "receiver owns moved region", "RegionOwner(frame) = actor1", "recv_typed"); err != nil {
		return err
	}
	if err := requireOwnershipTransferFacts(rows[TypedActorOwnershipExplicitCopyFallback], "copy fallback explicit", ".copy()", "TransferCopy"); err != nil {
		return err
	}
	if err := requireOwnershipTransferFacts(rows[TypedActorOwnershipUnsafeSendContract], "unsafe pointers require explicit unsafe send contract", "audited unsafe send contract", "safe typed actor ptr payloads rejected"); err != nil {
		return err
	}
	if err := requireOwnershipTransferFacts(rows[TypedActorOwnershipSemanticsChecker], "validateActorBoundaryPayloadExpr", "consumeTypedActorTransferPayloads", "checkBorrowedEscape"); err != nil {
		return err
	}
	if err := requireOwnershipTransferFacts(rows[TypedActorOwnershipPLIRMovedFacts], "PLIR moved facts", "FactMoved", "OpActorSend", "core.send_typed"); err != nil {
		return fmt.Errorf("typed actor ownership transfer: PLIR moved facts row invalid: %w", err)
	}
	if err := requireOwnershipTransferFacts(rows[TypedActorOwnershipRuntimeMailboxRepresentation], "TypedMailbox", "OwnershipMetadata", "zero_copy_move", "blocking_recv_yield"); err != nil {
		return err
	}
	if err := requireOwnershipTransferFacts(rows[TypedActorOwnershipActorTransferReport], "actor-transfer.json", "ownership", "transfer_mode", "bytes_copied"); err != nil {
		return err
	}
	if err := requireOwnershipTransferFacts(rows[TypedActorOwnershipStressDiagnostics], "stress diagnostics", "use-after-move", "cannot use consumed value"); err != nil {
		return fmt.Errorf("typed actor ownership transfer: stress diagnostics row invalid: %w", err)
	}
	return nil
}

func borrowedViewCopyBoundaryRow() TypedActorOwnershipTransferRow {
	return TypedActorOwnershipTransferRow{
		ID:     TypedActorOwnershipBorrowedViewCopyBoundary,
		Name:   "Borrowed view copy boundary",
		Status: TypedActorOwnershipImplementedNarrow,
		RequiredFacts: []string{
			"borrowed view rejected at actor boundary",
			"explicit copy accepted for borrowed view payloads",
			".copy() creates owned storage before send",
		},
		Evidence: "compiler/internal/actorsafety/sendability_test.go::TestBorrowedSliceAcrossActorBoundaryRejectsUnlessCopied; compiler/tests/semantics/borrow_copy_test.go::TestBorrowedActorSendRejectedUnlessCopied; compiler/internal/semantics/exprs.go::validateActorBoundaryPayloadExpr",
		Boundary: "covers checked typed actor message expressions; borrowed views do not get a zero-copy actor path unless the expression is explicitly copied or an owned-region slice moves with its owner",
	}
}

func ownedRegionMoveRow() TypedActorOwnershipTransferRow {
	return TypedActorOwnershipTransferRow{
		ID:     TypedActorOwnershipOwnedRegionMove,
		Name:   "Owned region move",
		Status: TypedActorOwnershipImplementedNarrow,
		RequiredFacts: []string{
			"owned region can move across typed actor mailbox",
			"zero_copy_move for local owned-region slice payloads",
			"bytes_copied=0 for owned-region transfer rows",
		},
		Evidence: "compiler/actors_test.go::TestActorsTypedMessagesAllowIslandTransferCheckAndLower; compiler/actors_test.go::TestActorsTypedMessagesOwnedRegionSliceMoveBuildAndRun; compiler/internal/parallelrt/scheduler_model_test.go::TestOwnedRegionMessageMovesZeroCopyAndBorrowedPayloadRequiresCopy",
		Boundary: "covers local typed actor payloads with an island owner in the same checked message; it is not distributed pointer or region zero-copy",
	}
}

func senderUseAfterMoveRow() TypedActorOwnershipTransferRow {
	return TypedActorOwnershipTransferRow{
		ID:     TypedActorOwnershipSenderUseAfterMove,
		Name:   "Sender loses access after move",
		Status: TypedActorOwnershipImplementedNarrow,
		RequiredFacts: []string{
			"cannot use consumed value after typed actor transfer",
			"cannot use moved region after send in actorsafety checker model",
			"sender use-after-move diagnostics stay stable",
		},
		Evidence: "compiler/actors_test.go::TestActorsTypedMessagesIslandTransferConsumesSource; compiler/actors_test.go::TestActorsTypedMessagesOwnedRegionSliceMoveConsumesSenderSlice; compiler/internal/actorsafety/sendability_test.go::TestOwnedRegionMustMoveAndSenderUseAfterMoveRejects",
		Boundary: "covers checker-enforced consumed resource state and the actorsafety model; it does not prove a full cross-thread race-safety model",
	}
}

func receiverOwnsMovedRegionRow() TypedActorOwnershipTransferRow {
	return TypedActorOwnershipTransferRow{
		ID:     TypedActorOwnershipReceiverOwnsMovedRegion,
		Name:   "Receiver owns moved region",
		Status: TypedActorOwnershipImplementedNarrow,
		RequiredFacts: []string{
			"receiver owns moved region after local scheduler model transfer",
			"RegionOwner(frame) = actor1 after Send",
			"recv_typed match can use moved_xs and moved_region in executable linux-x64 smoke",
		},
		Evidence: "compiler/internal/parallelrt/scheduler_model_test.go::TestOwnedRegionMessageMovesZeroCopyAndBorrowedPayloadRequiresCopy; compiler/actors_test.go::TestActorsTypedMessagesOwnedRegionSliceMoveBuildAndRun",
		Boundary: "receiver ownership evidence is local scheduler-model plus current linux-x64 typed mailbox execution; it is not a distributed ownership protocol",
	}
}

func explicitCopyFallbackRow() TypedActorOwnershipTransferRow {
	return TypedActorOwnershipTransferRow{
		ID:     TypedActorOwnershipExplicitCopyFallback,
		Name:   "Explicit copy fallback",
		Status: TypedActorOwnershipImplementedNarrow,
		RequiredFacts: []string{
			"copy fallback explicit through .copy()",
			"borrowed copied payloads report TransferCopy",
			"actor transfer report keeps explicit view copy rows",
		},
		Evidence: "compiler/tests/semantics/borrow_copy_test.go::TestBorrowedActorSendRejectedUnlessCopied; compiler/internal/parallelrt/scheduler_model_test.go::TestOwnedRegionMessageMovesZeroCopyAndBorrowedPayloadRequiresCopy; compiler/reports.go::actorTransferRowForPayload",
		Boundary: "copy fallback is opt-in at the expression/report level; no implicit copy-elision or hidden zero-copy promotion is claimed",
	}
}

func unsafeSendContractRow() TypedActorOwnershipTransferRow {
	return TypedActorOwnershipTransferRow{
		ID:     TypedActorOwnershipUnsafeSendContract,
		Name:   "Unsafe send contract",
		Status: TypedActorOwnershipImplementedNarrow,
		RequiredFacts: []string{
			"unsafe pointers require explicit unsafe send contract in checker model",
			"audited unsafe send contract is required before unsafe pointer sendability passes",
			"safe typed actor ptr payloads rejected by message type validation",
		},
		Evidence: "compiler/internal/actorsafety/sendability_test.go::TestUnsafePointerRequiresExplicitUnsafeSendContract; compiler/tests/safety/plan250_safety_runtime_test.go::TestPlan250SafetyRuntimeMatrix; compiler/internal/semantics/exprs.go::validateTypedActorMessageType",
		Boundary: "current source typed actor messages reject ptr payloads; the unsafe-contract path is internal checker-model evidence until a separate audited unsafe send surface exists",
	}
}

func semanticsTransferCheckerRow() TypedActorOwnershipTransferRow {
	return TypedActorOwnershipTransferRow{
		ID:     TypedActorOwnershipSemanticsChecker,
		Name:   "Semantics transfer checker",
		Status: TypedActorOwnershipImplementedNarrow,
		RequiredFacts: []string{
			"validateActorBoundaryPayloadExpr rejects borrowed payloads",
			"checkBorrowedEscape runs on typed actor message expressions",
			"consumeTypedActorTransferPayloads marks island and region-backed slice payloads consumed",
		},
		Evidence: "compiler/internal/semantics/exprs.go::checkTypedActorCallWithEffects; compiler/internal/semantics/exprs.go::validateActorBoundaryPayloadExpr; compiler/internal/semantics/exprs.go::consumeTypedActorTransferPayloads",
		Boundary: "semantics checker evidence is conservative and source-level; it does not by itself claim production runtime scheduling or distributed transfer behavior",
	}
}

func plirMovedFactsRow() TypedActorOwnershipTransferRow {
	return TypedActorOwnershipTransferRow{
		ID:     TypedActorOwnershipPLIRMovedFacts,
		Name:   "PLIR moved facts",
		Status: TypedActorOwnershipImplementedNarrow,
		RequiredFacts: []string{
			"PLIR moved facts emitted for direct typed actor ownership transfers",
			"FactMoved rows cite core.send_typed typed actor ownership transfer",
			"OpActorSend operation records typed actor send boundary",
		},
		Evidence: "compiler/internal/plir/plir.go::recordActorSendCall; compiler/internal/plir/plir_test.go::TestFromCheckedProgramRecordsTypedActorMovedFacts; compiler/internal/plir/plir_test.go::TestVerifyProgramRejectsFakeActorMovedFactClaims",
		Boundary: "PLIR moved facts cover checked direct constructor send_typed payloads with local values; alias/interprocedural forms remain represented by the semantics consumed-resource state unless separately promoted",
	}
}

func runtimeMailboxRepresentationRow() TypedActorOwnershipTransferRow {
	return TypedActorOwnershipTransferRow{
		ID:     TypedActorOwnershipRuntimeMailboxRepresentation,
		Name:   "Runtime mailbox representation",
		Status: TypedActorOwnershipImplementedNarrow,
		RequiredFacts: []string{
			"TypedMailbox preserves OwnershipMetadata",
			"bounded typed mailbox uses blocking_recv_yield backpressure",
			"zero_copy_move reports bytes_copied=0 for owned-region model transfer",
		},
		Evidence: "compiler/internal/parallelrt/scheduler_model.go::TypedMailbox; compiler/internal/parallelrt/scheduler_model_test.go::TestTypedMailboxPreservesCapacityBackpressureAndOwnershipMetadata; compiler/internal/parallelrt/scheduler_model_test.go::TestOwnedRegionMessageMovesZeroCopyAndBorrowedPayloadRequiresCopy",
		Boundary: "runtime mailbox evidence is local bounded model/report evidence and does not promote P18.0 actor runtime production blockers",
	}
}

func actorTransferReportRow() TypedActorOwnershipTransferRow {
	return TypedActorOwnershipTransferRow{
		ID:     TypedActorOwnershipActorTransferReport,
		Name:   "Actor transfer report",
		Status: TypedActorOwnershipImplementedNarrow,
		RequiredFacts: []string{
			"actor-transfer.json records ownership",
			"actor-transfer.json records transfer_mode",
			"actor-transfer.json records bytes_copied",
		},
		Evidence: "compiler/reports.go::buildActorTransferReport; compiler/reports.go::actorTransferRowForPayload; compiler/actors_test.go::TestActorsTypedMessagesOwnedRegionSliceMoveExplainReport; compiler/actors_test.go::TestActorsTypedMailboxExplainReportIncludesMetadataAndCopyMove",
		Boundary: "report rows are evidence only; reports do not toggle safe semantics or runtime behavior",
	}
}

func stressDiagnosticsRow() TypedActorOwnershipTransferRow {
	return TypedActorOwnershipTransferRow{
		ID:     TypedActorOwnershipStressDiagnostics,
		Name:   "Use-after-move stress diagnostics",
		Status: TypedActorOwnershipImplementedNarrow,
		RequiredFacts: []string{
			"stress diagnostics cover actor/task ownership transfer cases",
			"use-after-move diagnostics remain stable",
			"cannot use consumed value after typed actor transfer",
		},
		Evidence: "compiler/tests/ownership/actor_task_ownership_test.go; compiler/tests/ownership/actor_task_stress_test.go::TestActorTaskBoundedStressExamples; compiler/actors_test.go::TestActorsTypedMessagesIslandTransferConsumesSource; compiler/actors_test.go::TestActorsTypedMessagesOwnedRegionSliceMoveConsumesSenderSlice",
		Boundary: "stress evidence covers bounded compiler diagnostics and linux-x64 stress examples; it is not a full race-safety proof",
	}
}

func requireOwnershipTransferFacts(row TypedActorOwnershipTransferRow, wants ...string) error {
	for _, want := range wants {
		if !containsOwnershipTransferText(row.RequiredFacts, want) {
			return fmt.Errorf("row %q missing fact %q", row.ID, want)
		}
	}
	return nil
}

func containsOwnershipTransferText(items []string, want string) bool {
	for _, item := range items {
		if strings.Contains(item, want) {
			return true
		}
	}
	return false
}
