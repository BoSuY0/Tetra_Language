package allocplan

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tetra_language/compiler/internal/islandkernel"
	"tetra_language/compiler/internal/memoryfacts"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/runtimeabi"
)

type Input struct {
	Program  *plir.Program
	Snapshot memoryfacts.Snapshot
	Options  Options
}

func Build(input Input) (*Plan, error) {
	if input.Program == nil {
		return nil, fmt.Errorf("allocplan: missing PLIR program")
	}
	plan := &Plan{}
	functions := append([]plir.Function(nil), input.Program.Funcs...)
	sort.Slice(functions, func(i, j int) bool { return functions[i].Name < functions[j].Name })
	for _, fn := range functions {
		row := FunctionPlan{Name: fn.Name}
		values := append([]plir.Value(nil), fn.Values...)
		sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
		functionTempRegionUsed := false
		for _, value := range values {
			if value.Kind != plir.ValueAllocIntent || value.Alloc == nil {
				continue
			}
			evidence, err := resolvePlannerAllocation(
				input.Snapshot,
				memoryfacts.ValueKey{FunctionID: fn.Name, ValueID: value.ID},
			)
			if err != nil {
				return nil, fmt.Errorf(
					"allocplan: allocation evidence for %s/%s: %w",
					fn.Name,
					value.ID,
					err,
				)
			}
			if err := verifyAllocationEvidenceMatchesPLIR(fn, value, evidence); err != nil {
				return nil, err
			}
			valueOpt := input.Options
			if functionTempRegionUsed {
				valueOpt.EnableRegionPlanning = false
				valueOpt.EnableRegionLowering = false
			}
			alloc := planAllocationFromEvidence(fn, value, valueOpt, evidence)
			if alloc.Storage == StorageFunctionTempRegion {
				functionTempRegionUsed = true
			}
			row.Allocations = append(row.Allocations, alloc)
			plan.Totals.add(alloc.Storage)
		}
		if len(row.Allocations) > 0 {
			plan.Functions = append(plan.Functions, row)
		}
	}
	assignPlanDigests(plan, input.Options)
	if err := VerifyPlanned(plan); err != nil {
		return nil, err
	}
	return plan, nil
}

func planAllocationFromEvidence(
	fn plir.Function,
	value plir.Value,
	opt Options,
	evidence memoryfacts.AllocationEvidence,
) Allocation {
	id := allocationName(value.ID)
	escape := escapeClassFromEvidence(evidence)
	storage, storageReason := chooseStorageFromEvidence(fn, value, escape, opt, evidence)
	lengthStatus := classifyLengthStatus(value.Alloc)
	byteSize := constantByteSize(value.Alloc)
	if lengthStatus == LengthStatusValidEmpty {
		storageReason = storageReason + ("; valid empty allocation has no allocator access where the " +
			"backend implements the contract")
	}
	if lengthStatus == LengthStatusRejectedNegative {
		storageReason = storageReason + "; negative length is rejected before allocation"
	}
	if lengthStatus == LengthStatusRejectedOverflow {
		storageReason = storageReason + "; byte-size overflow is rejected before allocation"
	}
	alloc := Allocation{
		ID:                     id,
		SiteID:                 allocationSiteID(fn.Name, id, value.Source),
		ValueID:                value.ID,
		Source:                 value.Source,
		Builtin:                value.Alloc.Builtin,
		ElementType:            value.Alloc.ElementType,
		ElementSize:            value.Alloc.ElementSize,
		LengthExpr:             value.Alloc.LengthExpr,
		LengthStatus:           lengthStatus,
		ZeroGuardStatus:        value.Alloc.ZeroGuardStatus,
		NegativeGuardStatus:    value.Alloc.NegativeGuardStatus,
		OverflowGuardStatus:    value.Alloc.OverflowGuardStatus,
		ByteSize:               byteSize,
		Escape:                 escape,
		Storage:                storage,
		PlannedStorage:         storage,
		ActualLoweringStorage:  StorageUnknownConservative,
		Reason:                 storageReason,
		ValidationStatus:       "planned",
		LoweringStatus:         "pending",
		RawPointerBoundsStatus: value.Alloc.RawPointerBoundsStatus,
		RawPointerBaseID:       value.Alloc.RawPointerBaseID,
		RawPointerBaseBytes:    value.Alloc.RawPointerBaseBytes,
		RawPointerOffsetBytes:  value.Alloc.RawPointerOffsetBytes,
		RawSlicePolicy:         value.Alloc.RawSlicePolicy,
		SourceFactIDs:          factIDsToStrings(evidence.SourceFactIDs),
		ProofIDs:               proofIDsFromEvidence(evidence),
		DecisionCode:           decisionCode(escape, storage),
	}
	if escape == EscapeNoEscape && opt.EnableStackLowering {
		if scalarReplacement, scalarReason := isScalarReplacementCandidate(fn, value, id); scalarReplacement {
			alloc.Storage = StorageEliminated
			alloc.PlannedStorage = StorageEliminated
				alloc.Reason = scalarReason
				alloc.ValidationStatus = "validated_no_escape"
				alloc.DecisionCode = decisionCodeWithDetail(alloc.Escape, alloc.Storage, "scalar_replacement")
			}
		}
	if opt.EnableStackLowering && isUnusedCopyAllocation(fn, value, id) {
		alloc.Escape = EscapeNoEscape
		alloc.Storage = StorageEliminated
		alloc.PlannedStorage = StorageEliminated
		alloc.Reason = "copy result is unused in the supported typed PLIR scan; allocation intent can be elided"
		alloc.ValidationStatus = "validated_no_escape"
		alloc.DecisionCode = decisionCodeWithDetail(alloc.Escape, alloc.Storage, "unused_copy")
	}
	if alloc.Storage == StorageFunctionTempRegion {
		applyPlannedRegionAllocatorEvidence(&alloc, fn, byteSize)
	}
	applyBoundaryStorageValidationStatus(&alloc)
	if alloc.Storage == StorageExplicitIsland {
		alloc.RegionID = nonEmptyString(evidence.IslandID, value.Region)
		alloc.Lifetime = allocationLifetime(value)
		alloc.Domain = domainForEvidence(alloc, evidence)
		if slot, ok := explicitIslandHandleParamSlot(fn, value); ok {
			alloc.ExplicitIslandHandleParamSlotKnown = true
			alloc.ExplicitIslandHandleParamSlot = slot
		}
	}
	if alloc.Domain == nil {
		alloc.Domain = domainForEvidence(alloc, evidence)
	}
	alloc.ReasonCodes = appendReasonCodes(
		alloc.ReasonCodes,
		islandKernelPlannerReasonCodes(value, evidence)...,
	)
	applyDefaultAllocationReportHooks(&alloc)
	applyHeapReasonCodeEvidence(&alloc)
	return alloc
}

func verifyAllocationEvidenceMatchesPLIR(
	fn plir.Function,
	value plir.Value,
	evidence memoryfacts.AllocationEvidence,
) error {
	if evidence.TypeName != "" && evidence.TypeName != value.Type {
		return fmt.Errorf(
			"allocplan: PLIR/evidence mismatch for %s/%s: type %q != %q",
			fn.Name,
			value.ID,
			value.Type,
			evidence.TypeName,
		)
	}
	if evidence.AllocationSiteID != "" && value.Alloc != nil &&
		evidence.AllocationSiteID != value.Alloc.Builtin {
		return fmt.Errorf(
			"allocplan: PLIR/evidence mismatch for %s/%s: allocation site %q != builtin %q",
			fn.Name,
			value.ID,
			evidence.AllocationSiteID,
			value.Alloc.Builtin,
		)
	}
	if value.Provenance.Kind == plir.ProvenanceIsland {
		wantIsland, wantEpoch, err := islandIdentityFromProgramIR(fn, value)
		if err != nil {
			return err
		}
		if evidence.IslandID != "" && wantIsland != "" && evidence.IslandID != wantIsland {
			return fmt.Errorf(
				"allocplan: PLIR/evidence mismatch for %s/%s: island %q != %q",
				fn.Name,
				value.ID,
				evidence.IslandID,
				wantIsland,
			)
		}
		if evidence.Epoch != 0 && wantEpoch != 0 && evidence.Epoch != wantEpoch {
			return fmt.Errorf(
				"allocplan: PLIR/evidence mismatch for %s/%s: island epoch %d != %d",
				fn.Name,
				value.ID,
				evidence.Epoch,
				wantEpoch,
			)
		}
	}
	return nil
}

func islandIdentityFromProgramIR(fn plir.Function, value plir.Value) (string, int, error) {
	islandID := ""
	if value.Provenance.Kind == plir.ProvenanceIsland {
		root := strings.TrimSpace(value.Provenance.Root)
		if root == "" {
			root = "unknown"
		}
		islandID = "island:" + root
	}
	epoch := 0
	if islandID != "" {
		epoch = 1
	}
	explicitEpoch := false
	for _, fact := range fn.Facts {
		if fact.ValueID != value.ID {
			continue
		}
		if fact.IslandID != "" {
			if islandID != "" && fact.IslandID != islandID {
				return "", 0, fmt.Errorf(
					"allocplan: PLIR island evidence conflict for %s/%s: island %q != %q",
					fn.Name,
					value.ID,
					fact.IslandID,
					islandID,
				)
			}
			islandID = fact.IslandID
		}
		if fact.Epoch != 0 {
			if explicitEpoch && fact.Epoch != epoch {
				return "", 0, fmt.Errorf(
					"allocplan: PLIR island evidence conflict for %s/%s: epoch %d != %d",
					fn.Name,
					value.ID,
					fact.Epoch,
					epoch,
				)
			}
			epoch = fact.Epoch
			explicitEpoch = true
		}
	}
	return islandID, epoch, nil
}

func resolvePlannerAllocation(
	snapshot memoryfacts.Snapshot,
	key memoryfacts.ValueKey,
) (memoryfacts.AllocationEvidence, error) {
	facts := snapshot.FactsForValue(key)
	if len(facts) == 0 {
		return memoryfacts.AllocationEvidence{}, fmt.Errorf(
			"no allocation facts for %s/%s",
			key.FunctionID,
			key.ValueID,
		)
	}
	out := memoryfacts.AllocationEvidence{
		FunctionID: key.FunctionID,
		ValueID:    key.ValueID,
	}
	seen := map[memoryfacts.FactID]struct{}{}
	for _, fact := range facts {
		if fact.ValidationState == memoryfacts.ValidationInvalidated {
			continue
		}
		if fact.ID != "" {
			seen[fact.ID] = struct{}{}
		}
		firstString(&out.AllocationSiteID, fact.AllocationSiteID)
		firstString(&out.SiteID, fact.SiteID)
		firstString(&out.SourceSpan, fact.SourceSpan)
		firstString(&out.TypeName, fact.TypeName)
		out.ProvenanceClass = mergePlannerProvenance(out.ProvenanceClass, fact.ProvenanceClass)
		out.UnsafeClass = mergePlannerUnsafe(out.UnsafeClass, fact.UnsafeClass)
		firstString(&out.RegionID, fact.RegionID)
		firstString(&out.IslandID, fact.IslandID)
		if out.Epoch == 0 {
			out.Epoch = fact.Epoch
		}
		firstString(&out.OwnerID, fact.OwnerID)
		firstDomainKind(&out.DomainKind, fact.DomainKind)
		firstString(&out.DomainID, fact.DomainID)
		firstString(&out.DomainOwnerID, fact.DomainOwnerID)
		firstTransferKind(&out.TransferKind, fact.TransferKind)
		firstString(&out.TransferProofID, fact.TransferProofID)
		if fact.ProofKind == memoryfacts.ProofDomainMove {
			firstString(&out.TransferProofID, fact.ProofID)
		}
		out.SourceConsumed = out.SourceConsumed || fact.SourceConsumed
		out.LiveBorrowCrossing = out.LiveBorrowCrossing || fact.LiveBorrowCrossing
		out.DestinationActive = out.DestinationActive || fact.DestinationActive
		firstString(&out.LifetimeBirth, fact.LifetimeBirth)
		firstString(&out.LifetimeDeath, fact.LifetimeDeath)
		firstString(&out.LifetimeOwner, fact.LifetimeOwner)
		if plannerEscapeFact(fact) {
			out.EscapeState = mergePlannerEscape(out.EscapeState, fact.EscapeState)
		}
		if fact.Claim == memoryfacts.ClaimNoEscape {
			out.NoEscapeProofID = nonEmptyString(fact.ProofID, string(fact.ID))
			firstString(&out.NoEscapeReason, fact.Reason)
		}
		if fact.Claim == memoryfacts.ClaimTrustedStorage {
			out.StorageProofID = nonEmptyString(fact.ProofID, string(fact.ID))
		}
	}
	if out.AllocationSiteID == "" {
		return memoryfacts.AllocationEvidence{}, fmt.Errorf(
			"allocation_site_id is required for %s/%s",
			key.FunctionID,
			key.ValueID,
		)
	}
	for id := range seen {
		out.SourceFactIDs = append(out.SourceFactIDs, id)
	}
	sort.Slice(out.SourceFactIDs, func(i, j int) bool {
		return out.SourceFactIDs[i] < out.SourceFactIDs[j]
	})
	return out, nil
}

func plannerEscapeFact(fact memoryfacts.Fact) bool {
	if fact.EscapeState == "" {
		return false
	}
	id := string(fact.ID)
	if strings.Contains(id, ":summary:alloc_escape") {
		return true
	}
	if !strings.HasPrefix(id, "plir:") {
		return true
	}
	return false
}

func mergePlannerProvenance(
	current memoryfacts.ProvenanceClass,
	next memoryfacts.ProvenanceClass,
) memoryfacts.ProvenanceClass {
	if next == "" || next == memoryfacts.ProvenanceSafeKnown {
		return current
	}
	if current == "" || current == memoryfacts.ProvenanceSafeKnown {
		return next
	}
	if current == memoryfacts.ProvenanceUnsafeUnknown || next == memoryfacts.ProvenanceUnsafeUnknown {
		return memoryfacts.ProvenanceUnsafeUnknown
	}
	if current == memoryfacts.ProvenanceUnsafeChecked || next == memoryfacts.ProvenanceUnsafeChecked {
		return memoryfacts.ProvenanceUnsafeChecked
	}
	if current == memoryfacts.ProvenanceUnsafeVerifiedRoot || next == memoryfacts.ProvenanceUnsafeVerifiedRoot {
		return memoryfacts.ProvenanceUnsafeVerifiedRoot
	}
	if current == memoryfacts.ProvenanceSafeOwned || next == memoryfacts.ProvenanceSafeOwned {
		return memoryfacts.ProvenanceSafeOwned
	}
	if current == memoryfacts.ProvenanceSafeBorrowed || next == memoryfacts.ProvenanceSafeBorrowed {
		return memoryfacts.ProvenanceSafeBorrowed
	}
	return current
}

func mergePlannerUnsafe(
	current memoryfacts.UnsafeClass,
	next memoryfacts.UnsafeClass,
) memoryfacts.UnsafeClass {
	if next == "" || next == memoryfacts.UnsafeSafe {
		return current
	}
	if current == "" || current == memoryfacts.UnsafeSafe {
		return next
	}
	if current == memoryfacts.UnsafeUnknown || next == memoryfacts.UnsafeUnknown {
		return memoryfacts.UnsafeUnknown
	}
	if current == memoryfacts.UnsafeChecked || next == memoryfacts.UnsafeChecked {
		return memoryfacts.UnsafeChecked
	}
	if current == memoryfacts.UnsafeVerifiedRoot || next == memoryfacts.UnsafeVerifiedRoot {
		return memoryfacts.UnsafeVerifiedRoot
	}
	return current
}

func mergePlannerEscape(
	current memoryfacts.EscapeState,
	next memoryfacts.EscapeState,
) memoryfacts.EscapeState {
	if next == "" || next == memoryfacts.EscapeConservative {
		return current
	}
	if current == "" || current == memoryfacts.EscapeConservative {
		return next
	}
	if current == next {
		return current
	}
	return memoryfacts.EscapeConservative
}

func firstString(dst *string, value string) {
	if *dst == "" && strings.TrimSpace(value) != "" {
		*dst = value
	}
}

func firstDomainKind(dst *memoryfacts.DomainKind, value memoryfacts.DomainKind) {
	if *dst == "" && value != "" {
		*dst = value
	}
}

func firstTransferKind(dst *memoryfacts.TransferKind, value memoryfacts.TransferKind) {
	if *dst == "" && value != "" {
		*dst = value
	}
}

func chooseStorageFromEvidence(
	fn plir.Function,
	value plir.Value,
	escape EscapeClass,
	opt Options,
	evidence memoryfacts.AllocationEvidence,
) (StorageClass, string) {
	if classifyLengthStatus(value.Alloc) == LengthStatusValidEmpty {
		return StorageEliminated, "zero-length allocation intent needs no backing storage"
	}
	if evidence.ProvenanceClass == memoryfacts.ProvenanceUnsafeUnknown ||
		evidence.UnsafeClass == memoryfacts.UnsafeUnknown {
		decision := islandkernel.CanPromoteUnsafeRoot(
			islandkernel.UnsafeRequest{Ref: islandKernelMemoryRef(value, evidence)},
		)
		return StorageHeap, decision.Reason.Message
	}
	if evidence.ProvenanceClass == memoryfacts.ProvenanceUnsafeVerifiedRoot ||
		evidence.ProvenanceClass == memoryfacts.ProvenanceUnsafeChecked {
		return StorageHeap, "unsafe provenance requires conservative heap fallback before typed runtime proof"
	}
	if value.Provenance.Kind == plir.ProvenanceIsland {
		decision := islandKernelPlanDecision(value, evidence)
		if decision.Decision == islandkernel.Accept {
			return StorageExplicitIsland, decision.Reason.Message
		}
		return StorageHeap, decision.Reason.Message
	}
	if evidence.DomainKind == memoryfacts.DomainRequest && !requestDomainProofValid(evidence) {
		return StorageHeap, HeapReasonRequestOwnerUnproven + ": request domain owner lacks typed proof"
	}
	switch escape {
	case EscapeNoEscape:
		if opt.EnableRegionPlanning && isFunctionTempRegionCandidate(value) {
			return StorageFunctionTempRegion, (("function-local temporary copy has bounded " +
				"lifetime and can ") +
				"be planned for a temp region")
		}
		if !opt.EnableStackLowering {
			return StorageHeap, "target allocation plan disables stack lowering; heap fallback keeps no-escape allocation portable"
		}
		bytes := constantByteSize(value.Alloc)
		stackLimit := stackAllocationLimitBytes(value.Alloc)
		if bytes > 0 && bytes <= stackLimit {
			if strings.Contains(evidence.NoEscapeReason, "typed read-only local call") {
				return StorageStack, fmt.Sprintf(
					("fixed_small_read_only_local_call_no_escape: fixed-size no-escape " +
						"allocation crosses only proven read-only local call summaries and is " +
						"%d bytes, within stack threshold; allocation is passed only to " +
						"proven read-only local call summary parameters and does not escape " +
						"in the supported v0 scan"),
					bytes,
				)
			}
			if stackLimit > smallStackAllocationBytes {
				return StorageStack, fmt.Sprintf(
					("fixed_i32_no_escape: fixed-size no-escape i32 allocation is " +
						"%d bytes, within i32 stack threshold"),
					bytes,
				)
			}
			return StorageStack, fmt.Sprintf(
				"fixed_small_no_escape: fixed-size no-escape allocation is %d bytes, within stack threshold",
				bytes,
			)
		}
		return StorageHeap, "no-escape allocation has non-constant or large size; planner keeps heap fallback"
	case EscapeActor:
		if boundaryMoveProofValid(evidence, memoryfacts.DomainActor) {
			return StorageActorMoveRegion, "proof-carrying actor ownership move may use ActorMoveRegion"
		}
		return StorageHeap, HeapReasonActorMoveUnproven + ": typed actor escape lacks validated move proof, consumed source, active destination, or no-live-borrow evidence"
	case EscapeTask:
		if boundaryMoveProofValid(evidence, memoryfacts.DomainTask) {
			return StorageTaskRegion, "proof-carrying task ownership move may use TaskRegion"
		}
		return StorageHeap, HeapReasonTaskMoveUnproven + ": typed task escape lacks validated move proof, consumed source, active destination, or no-live-borrow evidence"
	case EscapeUnsafe:
		return StorageHeap, "typed unsafe escape requires conservative heap fallback"
	case EscapeReturn:
		return StorageHeap, "returned allocation needs caller-owned region support before it can avoid heap fallback"
	case EscapeGlobal:
		return StorageHeap, "global escape requires conservative heap fallback"
	case EscapeClosure:
		return StorageHeap, "closure environment escape requires conservative heap fallback"
	case EscapeAggregate:
		return StorageHeap, "aggregate escape requires conservative heap fallback until field-sensitive storage planning"
	case EscapeCallUnknown:
		return StorageHeap, "unknown call escape requires conservative heap fallback"
	default:
		return StorageHeap, "unknown escape state requires conservative heap fallback"
	}
}

func applyBoundaryStorageValidationStatus(alloc *Allocation) {
	if alloc == nil {
		return
	}
	switch alloc.Storage {
	case StorageTaskRegion:
		alloc.ValidationStatus = "validated_task_region_scope"
	case StorageActorMoveRegion:
		alloc.ValidationStatus = "validated_actor_move_region_scope"
	}
}

func boundaryMoveProofValid(
	evidence memoryfacts.AllocationEvidence,
	want memoryfacts.DomainKind,
) bool {
	return evidence.DomainKind == want &&
		evidence.TransferKind == memoryfacts.TransferMove &&
		strings.TrimSpace(evidence.DomainOwnerID) != "" &&
		strings.TrimSpace(evidence.TransferProofID) != "" &&
		evidence.SourceConsumed &&
		!evidence.LiveBorrowCrossing &&
		evidence.DestinationActive
}

func requestDomainProofValid(evidence memoryfacts.AllocationEvidence) bool {
	return evidence.DomainKind == memoryfacts.DomainRequest &&
		strings.TrimSpace(evidence.DomainOwnerID) != "" &&
		strings.TrimSpace(evidence.TransferProofID) != "" &&
		evidence.DestinationActive &&
		!evidence.LiveBorrowCrossing &&
		evidence.UnsafeClass == memoryfacts.UnsafeSafe
}

func islandKernelPlannerReasonCodes(
	value plir.Value,
	evidence memoryfacts.AllocationEvidence,
) []string {
	switch {
	case value.Provenance.Kind == plir.ProvenanceIsland:
		if code := islandKernelPlanDecision(value, evidence).Reason.Code; code != "" {
			return []string{code}
		}
	case evidence.ProvenanceClass == memoryfacts.ProvenanceUnsafeUnknown ||
		evidence.UnsafeClass == memoryfacts.UnsafeUnknown:
		if code := islandkernel.CanPromoteUnsafeRoot(
			islandkernel.UnsafeRequest{Ref: islandKernelMemoryRef(value, evidence)},
		).Reason.Code; code != "" {
			return []string{code}
		}
	}
	return nil
}

func islandKernelPlanDecision(
	value plir.Value,
	evidence memoryfacts.AllocationEvidence,
) islandkernel.Result {
	return islandkernel.CanPlanExplicitIsland(islandkernel.StoragePlanRequest{
		Ref:          islandKernelMemoryRef(value, evidence),
		Escape:       evidence.EscapeState,
		StorageProof: islandKernelStorageProof(evidence),
	})
}

func islandKernelStorageProof(evidence memoryfacts.AllocationEvidence) islandkernel.Proof {
	proofID := nonEmptyString(evidence.StorageProofID, evidence.NoEscapeProofID)
	return islandkernel.Proof{
		ID:            proofID,
		Kind:          memoryfacts.ProofStorage,
		SubjectBaseID: evidence.ValueID,
		IslandID:      evidence.IslandID,
		Epoch:         uint64(nonNegativeInt(evidence.Epoch)),
		Operation:     islandkernel.OperationExplicitIslandStorage,
		Verified:      proofID != "",
	}
}

func islandKernelMemoryRef(
	value plir.Value,
	evidence memoryfacts.AllocationEvidence,
) islandkernel.MemoryRef {
	return islandkernel.MemoryRef{
		BaseID:      evidence.ValueID,
		IslandID:    evidence.IslandID,
		Epoch:       uint64(nonNegativeInt(evidence.Epoch)),
		OwnerID:     nonEmptyString(evidence.OwnerID, value.Provenance.Root),
		Provenance:  evidence.ProvenanceClass,
		UnsafeClass: evidence.UnsafeClass,
	}
}

func nonNegativeInt(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func escapeClassFromEvidence(evidence memoryfacts.AllocationEvidence) EscapeClass {
	if evidence.ProvenanceClass == memoryfacts.ProvenanceUnsafeUnknown ||
		evidence.UnsafeClass == memoryfacts.UnsafeUnknown {
		return EscapeUnknown
	}
	switch evidence.EscapeState {
	case memoryfacts.EscapeNoEscape:
		return EscapeNoEscape
	case memoryfacts.EscapeReturn:
		return EscapeReturn
	case memoryfacts.EscapeGlobal:
		return EscapeGlobal
	case memoryfacts.EscapeActor:
		return EscapeActor
	case memoryfacts.EscapeTask:
		return EscapeTask
	case memoryfacts.EscapeUnsafe:
		return EscapeUnsafe
	case memoryfacts.EscapeConservative:
		return EscapeUnknown
	default:
		return EscapeUnknown
	}
}

func domainForEvidence(
	alloc Allocation,
	evidence memoryfacts.AllocationEvidence,
) *runtimeabi.MemoryDomain {
	requested := int64(allocationReportBytesRequested(alloc))
	reserved := int64(allocationReportBytesReserved(alloc))
	var domain runtimeabi.MemoryDomain
	switch {
	case alloc.PlannedStorage == StorageExplicitIsland && evidence.IslandID != "":
		domain = runtimeabi.IslandMemoryDomain(evidence.IslandID, allocationLifetimeFromEvidence(evidence), requested, reserved)
	case evidence.DomainKind == memoryfacts.DomainTask &&
		strings.TrimSpace(evidence.DomainOwnerID) != "" &&
		(boundaryMoveProofValid(evidence, memoryfacts.DomainTask) || evidence.TransferKind == memoryfacts.TransferCopy):
		domain = runtimeabi.TaskMemoryDomain(
			evidence.DomainOwnerID,
			"domain:process",
			allocationLifetimeFromEvidence(evidence),
			0,
		)
	case evidence.DomainKind == memoryfacts.DomainActor &&
		strings.TrimSpace(evidence.DomainOwnerID) != "" &&
		(boundaryMoveProofValid(evidence, memoryfacts.DomainActor) || evidence.TransferKind == memoryfacts.TransferCopy):
		domain = runtimeabi.ActorMemoryDomain(
			evidence.DomainOwnerID,
			"domain:process",
			allocationLifetimeFromEvidence(evidence),
			0,
		)
	case requestDomainProofValid(evidence):
		domain = runtimeabi.RequestMemoryDomain(
			evidence.DomainOwnerID,
			"domain:process",
			allocationLifetimeFromEvidence(evidence),
			0,
		)
	case alloc.Escape == EscapeUnsafe || evidence.ProvenanceClass == memoryfacts.ProvenanceUnsafeUnknown:
		domain = runtimeabi.ExternalMemoryDomain(nonEmptyString(evidence.OwnerID, alloc.ID), allocationLifetimeFromEvidence(evidence), requested, reserved)
	default:
		domain = runtimeabi.DefaultProcessMemoryDomain(0, 0)
	}
	domain.RequestedBytes = requested
	domain.ReservedBytes = reserved
	return &domain
}

func allocationLifetimeFromEvidence(evidence memoryfacts.AllocationEvidence) string {
	if evidence.LifetimeBirth != "" && evidence.LifetimeDeath != "" {
		return evidence.LifetimeBirth + ".." + evidence.LifetimeDeath
	}
	if evidence.LifetimeOwner != "" {
		return "owner:" + evidence.LifetimeOwner
	}
	if evidence.OwnerID != "" {
		return "owner:" + evidence.OwnerID
	}
	return "planned"
}

func factIDsToStrings(ids []memoryfacts.FactID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != "" {
			out = append(out, string(id))
		}
	}
	sort.Strings(out)
	return out
}

func proofIDsFromEvidence(evidence memoryfacts.AllocationEvidence) []string {
	var ids []string
	if evidence.NoEscapeProofID != "" {
		ids = append(ids, evidence.NoEscapeProofID)
	}
	if evidence.StorageProofID != "" {
		ids = append(ids, evidence.StorageProofID)
	}
	if evidence.TransferProofID != "" {
		ids = append(ids, evidence.TransferProofID)
	}
	sort.Strings(ids)
	return uniqueStrings(ids)
}

func decisionCode(escape EscapeClass, storage StorageClass) string {
	return "allocplan:" + string(escape) + ":" + string(storage)
}

func decisionCodeWithDetail(escape EscapeClass, storage StorageClass, detail string) string {
	if strings.TrimSpace(detail) == "" {
		return decisionCode(escape, storage)
	}
	return decisionCode(escape, storage) + ":" + strings.TrimSpace(detail)
}

func assignPlanDigests(plan *Plan, opt Options) {
	if plan == nil {
		return
	}
	for i := range plan.Functions {
		for j := range plan.Functions[i].Allocations {
			plan.Functions[i].Allocations[j].PlanDigest = allocationPlanDigest(
				plan.Functions[i].Name,
				plan.Functions[i].Allocations[j],
				opt,
			)
		}
	}
}

func allocationPlanDigest(function string, alloc Allocation, opt Options) string {
	alloc.PlanDigest = ""
	raw, _ := json.Marshal(struct {
		Schema   string     `json:"schema"`
		Function string     `json:"function"`
		Options  Options    `json:"options"`
		Alloc    Allocation `json:"allocation"`
	}{
		Schema:   "tetra.allocplan.v2",
		Function: function,
		Options:  opt,
		Alloc:    alloc,
	})
	sum := sha256.Sum256(raw)
	return "alloc-plan:sha256:" + hex.EncodeToString(sum[:])
}

func VerifyPlanned(plan *Plan) error {
	if plan == nil {
		return fmt.Errorf("allocplan verifier: missing plan")
	}
	if err := verifyPlanShape(plan); err != nil {
		return err
	}
	for _, fn := range plan.Functions {
		for _, alloc := range fn.Allocations {
			if alloc.ActualLoweringStorage != StorageUnknownConservative {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q planned state must keep actual lowering pending, got %q",
					fn.Name,
					alloc.ValueID,
					alloc.ActualLoweringStorage,
				)
			}
			if alloc.LoweringStatus != "pending" {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q planned state has lowering_status %q, want pending",
					fn.Name,
					alloc.ValueID,
					alloc.LoweringStatus,
				)
			}
			if alloc.Storage == StorageHeap && len(alloc.HeapReasonCodes) == 0 {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q heap storage missing heap reason code",
					fn.Name,
					alloc.ValueID,
				)
			}
			if alloc.PlanDigest == "" {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q missing plan digest",
					fn.Name,
					alloc.ValueID,
				)
			}
			if alloc.DecisionCode == "" {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q missing decision code",
					fn.Name,
					alloc.ValueID,
				)
			}
			if err := validateAllocationReasonCodes(fn.Name, alloc); err != nil {
				return err
			}
			if alloc.Domain != nil {
				if err := runtimeabi.ValidateMemoryDomain(*alloc.Domain); err != nil {
					return fmt.Errorf(
						"allocplan verifier: %s allocation %q invalid memory domain: %w",
						fn.Name,
						alloc.ValueID,
						err,
					)
				}
			}
		}
	}
	return nil
}

func VerifyLowered(plan *Plan) error {
	if plan == nil {
		return fmt.Errorf("allocplan verifier: missing plan")
	}
	if err := verifyPlanShape(plan); err != nil {
		return err
	}
	for _, fn := range plan.Functions {
		for _, alloc := range fn.Allocations {
			if alloc.ActualLoweringStorage == StorageUnknownConservative ||
				alloc.LoweringStatus == "pending" {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q lowering is still pending",
					fn.Name,
					alloc.ValueID,
				)
			}
			if strings.TrimSpace(alloc.LoweredArtifactID) == "" {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q missing lowered artifact",
					fn.Name,
					alloc.ValueID,
				)
			}
		}
	}
	return VerifyPlan(plan)
}

func verifyPlanShape(plan *Plan) error {
	seen := map[string]bool{}
	for _, fn := range plan.Functions {
		if fn.Name == "" {
			return fmt.Errorf("allocplan verifier: function with empty name")
		}
		for _, alloc := range fn.Allocations {
			if alloc.ValueID == "" {
				return fmt.Errorf("allocplan verifier: %s allocation with empty value id", fn.Name)
			}
			if alloc.ID == "" {
				return fmt.Errorf(
					"allocplan verifier: %s allocation with empty allocation id",
					fn.Name,
				)
			}
			if alloc.SiteID == "" {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q missing stable site id",
					fn.Name,
					alloc.ValueID,
				)
			}
			if alloc.Builtin == "" {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q missing builtin",
					fn.Name,
					alloc.ValueID,
				)
			}
			key := fn.Name + "\x00" + alloc.ValueID
			if seen[key] {
				return fmt.Errorf(
					"allocplan verifier: %s duplicate allocation %q",
					fn.Name,
					alloc.ValueID,
				)
			}
			seen[key] = true
			if alloc.Storage == "" {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q has empty storage",
					fn.Name,
					alloc.ValueID,
				)
			}
			if alloc.PlannedStorage == "" {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q has empty planned storage",
					fn.Name,
					alloc.ValueID,
				)
			}
			if alloc.PlannedStorage != alloc.Storage {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q planned storage %s does not match storage %s",
					fn.Name,
					alloc.ValueID,
					alloc.PlannedStorage,
					alloc.Storage,
				)
			}
			if alloc.LengthStatus == "" {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q has empty length status",
					fn.Name,
					alloc.ValueID,
				)
			}
			if strings.TrimSpace(alloc.Reason) == "" {
				return fmt.Errorf(
					"allocplan verifier: %s allocation %q missing storage reason",
					fn.Name,
					alloc.ValueID,
				)
			}
			if alloc.Escape != EscapeNoEscape &&
				trustedStorageRequiresNoEscape(alloc.Storage, alloc.LengthStatus) &&
				!trustedBoundaryStorageForEscape(alloc.Escape, alloc.Storage) {
				return fmt.Errorf(
					"allocplan verifier: %s escaping allocation %q cannot use %s storage",
					fn.Name,
					alloc.ValueID,
					alloc.Storage,
				)
			}
		}
	}
	return nil
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := values[:0]
	var prev string
	for i, value := range values {
		if i > 0 && value == prev {
			continue
		}
		out = append(out, value)
		prev = value
	}
	return out
}

func nonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func cleanDomainPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer(":", "_", "/", "_", "\\", "_", " ", "_")
	return replacer.Replace(value)
}
