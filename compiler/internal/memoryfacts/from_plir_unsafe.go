package memoryfacts

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/plir"
)

func provenanceClassForPLIRFact(fact plir.Fact, value plir.Value) ProvenanceClass {
	if value.Alloc != nil && value.Alloc.Builtin == "core.alloc_bytes" {
		return ProvenanceUnsafeVerifiedRoot
	}
	switch fact.Kind {
	case plir.FactProvenanceUnknown:
		return ProvenanceUnsafeUnknown
	case plir.FactBorrowedImm, plir.FactBorrowedMut:
		if value.Provenance.Kind == plir.ProvenanceExternal || value.Provenance.Kind == plir.ProvenanceUnknown {
			return ProvenanceUnsafeUnknown
		}
		return ProvenanceSafeBorrowed
	case plir.FactOwned:
		return ProvenanceSafeOwned
	default:
		if value.Provenance.Kind == plir.ProvenanceExternal || value.Provenance.Kind == plir.ProvenanceUnknown {
			return ProvenanceUnsafeUnknown
		}
		return ProvenanceSafeKnown
	}
}

func unsafeClassForPLIRFact(fact plir.Fact, value plir.Value) UnsafeClass {
	if value.Alloc != nil && value.Alloc.Builtin == "core.alloc_bytes" {
		return UnsafeVerifiedRoot
	}
	switch fact.Kind {
	case plir.FactProvenanceUnknown:
		return UnsafeUnknown
	default:
		if value.Provenance.Kind == plir.ProvenanceExternal || value.Provenance.Kind == plir.ProvenanceUnknown {
			return UnsafeUnknown
		}
		return UnsafeSafe
	}
}

func borrowStateForPLIRFact(fact plir.Fact) BorrowState {
	switch fact.Kind {
	case plir.FactBorrowedImm:
		return BorrowImmutable
	case plir.FactBorrowedMut:
		return BorrowMutable
	case plir.FactMoved:
		return BorrowMoved
	default:
		return BorrowNone
	}
}

func aliasStateForPLIRFact(fact plir.Fact) AliasState {
	switch fact.Kind {
	case plir.FactNoAlias:
		return AliasMutableExclusive
	default:
		return AliasUnknown
	}
}

func escapeStateForPLIRFact(fact plir.Fact, value plir.Value) EscapeState {
	if fact.Kind == plir.FactNoEscape {
		return EscapeNoEscape
	}
	switch value.Escape {
	case plir.EscapeNoEscape:
		return EscapeNoEscape
	case plir.EscapeReturn:
		return EscapeReturn
	case plir.EscapeGlobal:
		return EscapeGlobal
	case plir.EscapeActor:
		return EscapeActor
	case plir.EscapeTask:
		return EscapeTask
	case plir.EscapeUnsafe:
		return EscapeUnsafe
	case plir.EscapeConservative:
		return EscapeConservative
	default:
		return EscapeUnknown
	}
}

func allocationSiteIDForPLIRValue(value plir.Value) string {
	if value.Alloc == nil {
		return ""
	}
	return nonEmpty(value.Alloc.Builtin, value.Provenance.Root, value.ID)
}

func ownerForPLIRValue(value plir.Value) string {
	return normalizeOwnerID(nonEmpty(value.Provenance.Root, value.Lifetime.Owner))
}

func islandIDForPLIRFact(fact plir.Fact, value plir.Value) string {
	if fact.IslandID != "" {
		return fact.IslandID
	}
	if value.Provenance.Kind != plir.ProvenanceIsland {
		return ""
	}
	root := value.Provenance.Root
	if root == "" {
		root = "unknown"
	}
	return "island:" + root
}

func epochForPLIRFact(fact plir.Fact, value plir.Value) int {
	if fact.Epoch != 0 {
		return fact.Epoch
	}
	if islandIDForPLIRFact(fact, value) != "" {
		return 1
	}
	return 0
}

func baseIDForPLIRFact(fact plir.Fact, value plir.Value) string {
	if fact.BaseID != "" {
		return fact.BaseID
	}
	if islandIDForPLIRFact(fact, value) != "" {
		return value.ID
	}
	return ""
}

func ownerFromOperationInput(op plir.Operation, index int) string {
	if index < 0 || index >= len(op.Inputs) {
		return ""
	}
	return normalizeOwnerID(op.Inputs[index])
}

func normalizeOwnerID(owner string) string {
	owner = strings.TrimSpace(owner)
	for strings.HasPrefix(owner, "derived:") {
		owner = strings.TrimPrefix(owner, "derived:")
	}
	owner = strings.TrimPrefix(owner, "param:")
	owner = strings.TrimPrefix(owner, "local:")
	owner = strings.TrimPrefix(owner, "view:")
	owner = strings.TrimPrefix(owner, "alloc_intent:")
	if dot := strings.Index(owner, "."); dot > 0 {
		owner = owner[:dot]
	}
	return owner
}

func isCopyAllocationValue(value plir.Value) bool {
	return value.Kind == plir.ValueAllocIntent && value.Alloc != nil && copyAllocationBuiltin(value.Alloc.Builtin)
}

func copyAllocationBuiltin(name string) bool {
	return name == "core.string_copy" || (strings.HasPrefix(name, "core.slice_copy_") && !strings.HasPrefix(name, "core.slice_copy_into_"))
}

func isCopyIntoOperation(op plir.Operation) bool {
	return op.Kind == plir.OpCall && strings.Contains(op.Note, "copy_into")
}

func sourceFactIDForPath(path string, factIDs map[plirFactKey]FactID, values map[string]plir.Value) FactID {
	for _, valueID := range candidateValueIDs(path, values) {
		for _, kind := range []plir.FactKind{plir.FactBorrowedImm, plir.FactBorrowedMut, plir.FactProvenanceKnown, plir.FactOwned, plir.FactNoEscape} {
			if id := factIDs[plirFactKey{kind: kind, valueID: valueID}]; id != "" {
				return id
			}
		}
	}
	return ""
}

func valueIDForPath(path string, values map[string]plir.Value) string {
	for _, valueID := range candidateValueIDs(path, values) {
		if _, ok := values[valueID]; ok {
			return valueID
		}
	}
	return ""
}

func candidateValueIDs(path string, values map[string]plir.Value) []string {
	owner := normalizeOwnerID(path)
	if owner == "" {
		return nil
	}
	candidates := []string{owner}
	for _, prefix := range []string{"view:", "alloc_intent:", "local:", "param:"} {
		candidates = append(candidates, prefix+owner)
	}
	if path != owner {
		candidates = append(candidates, path)
		for _, prefix := range []string{"view:", "alloc_intent:", "local:", "param:"} {
			candidates = append(candidates, prefix+path)
		}
	}
	out := candidates[:0]
	seen := map[string]bool{}
	for _, candidate := range candidates {
		if candidate == "" || seen[candidate] {
			continue
		}
		if len(values) > 0 {
			if _, ok := values[candidate]; !ok && !strings.Contains(candidate, ":") {
				continue
			}
		}
		seen[candidate] = true
		out = append(out, candidate)
	}
	return out
}

func ffiDerivedFactID(parentID FactID, opID string, suffix string) FactID {
	return FactID(fmt.Sprintf("%s:%s:%s", parentID, safeFactIDPart(opID), suffix))
}

func aliasBoundaryDerivedFactID(parentID FactID, opID string, suffix string) FactID {
	return FactID(fmt.Sprintf("%s:%s:%s", parentID, safeFactIDPart(opID), suffix))
}

func claimForPLIRFact(fact plir.Fact) string {
	if fact.Kind == plir.FactProvenanceUnknown && strings.Contains(strings.ToLower(fact.Reason), "raw slice") {
		return "external_unknown"
	}
	return fact.Kind.String()
}

func isSafeWrapperPromotionOperation(op plir.Operation) bool {
	if op.Kind != plir.OpUnsafe {
		return false
	}
	note := strings.ToLower(strings.TrimSpace(op.Note))
	return strings.Contains(note, "safe wrapper") && (strings.Contains(note, "external") || strings.Contains(note, "unsafe_unknown") || strings.Contains(note, "raw"))
}

func unsafeOperationClaim(op plir.Operation) (string, ProvenanceClass, UnsafeClass, bool) {
	note := strings.ToLower(op.Note)
	switch {
	case unsafeStaticContractNote(note):
		return "unsafe_contract_static_untrusted", ProvenanceUnsafeUnknown, UnsafeUnknown, true
	case unsafeRuntimeCheckableContractNote(note):
		return "unsafe_contract_runtime_checkable", ProvenanceUnsafeChecked, UnsafeChecked, true
	case strings.Contains(note, "rejected_negative_offset"):
		return "rejected_negative_offset", ProvenanceUnsafeChecked, UnsafeChecked, true
	case strings.Contains(note, "rejected_upper_bound"):
		return "rejected_upper_bound", ProvenanceUnsafeChecked, UnsafeChecked, true
	case strings.Contains(note, "rejected_access_width_overflow"):
		return "rejected_access_width_overflow", ProvenanceUnsafeChecked, UnsafeChecked, true
	case strings.Contains(note, "rejected_negative_length"):
		return "rejected_negative_length", ProvenanceUnsafeChecked, UnsafeChecked, true
	case strings.Contains(note, "rejected_length_overflow"):
		return "rejected_length_overflow", ProvenanceUnsafeChecked, UnsafeChecked, true
	case strings.Contains(note, "raw_slice_bounds") && strings.Contains(note, "verified_allocation_root"):
		return "raw_slice_verified_allocation_root", ProvenanceUnsafeChecked, UnsafeChecked, true
	case strings.Contains(note, "raw memory gateway"):
		if op.UnsafeClass == plir.UnsafeChecked {
			return "raw_memory_access_checked", ProvenanceUnsafeChecked, UnsafeChecked, true
		}
		return "raw_memory_access_unknown", ProvenanceUnsafeUnknown, UnsafeUnknown, true
	case strings.Contains(note, "derived_allocation_offset"):
		return "derived_allocation_offset", ProvenanceUnsafeChecked, UnsafeChecked, true
	case strings.Contains(note, "checked_external_unknown"):
		return "checked_external_unknown", ProvenanceUnsafeUnknown, UnsafeUnknown, true
	case strings.Contains(note, "external-provenance view"):
		return "external_unknown", ProvenanceUnsafeUnknown, UnsafeUnknown, true
	default:
		return "", "", "", false
	}
}

func unsafeRuntimeCheckableContractNote(note string) bool {
	if !strings.Contains(note, "unsafe contract") || !strings.Contains(note, "runtime_checkable") {
		return false
	}
	return strings.Contains(note, "nonnull") ||
		strings.Contains(note, "non_null") ||
		strings.Contains(note, "alignment") ||
		strings.Contains(note, "aligned") ||
		strings.Contains(note, "length") ||
		strings.Contains(note, "bounds")
}

func unsafeStaticContractNote(note string) bool {
	if !strings.Contains(note, "unsafe contract") {
		return false
	}
	return strings.Contains(note, "static_untrusted") ||
		strings.Contains(note, "noalias") ||
		strings.Contains(note, "no_alias") ||
		strings.Contains(note, "lifetime") ||
		strings.Contains(note, "region")
}

func finalizeUnsafeOperationFact(graph *Graph, id FactID, claim string, provenance ProvenanceClass, unsafeClass UnsafeClass) error {
	switch claim {
	case "unsafe_contract_runtime_checkable":
		return graph.MarkValidated(id, "unsafe_runtime_contract_validator")
	}
	if rawBoundsRuntimeCheckClaim(claim) && provenance == ProvenanceUnsafeChecked && unsafeClass == UnsafeChecked {
		parent, ok := graph.Fact(id)
		if !ok {
			return fmt.Errorf("memoryfacts: unsafe operation fact %q was not recorded", id)
		}
		if err := addRawBoundsRuntimeCheckFact(graph, parent); err != nil {
			return err
		}
	}
	if provenance == ProvenanceUnsafeUnknown || unsafeClass == UnsafeUnknown {
		parent, ok := graph.Fact(id)
		if !ok {
			return fmt.Errorf("memoryfacts: unsafe operation fact %q was not recorded", id)
		}
		if claim == "unsafe_unknown_rejected_safe_facts" {
			return nil
		}
		return addUnsafeUnknownRejectedSafeFacts(graph, parent)
	}
	return nil
}

func rawBoundsRuntimeCheckClaim(claim string) bool {
	switch strings.ToLower(strings.TrimSpace(claim)) {
	case "raw_memory_access_checked", "rejected_access_width_overflow", "rejected_length_overflow":
		return true
	default:
		return false
	}
}

func addRawBoundsRuntimeCheckFact(graph *Graph, parent Fact) error {
	_, err := graph.DeriveFact(parent.ID, Fact{
		ID:               derivedFactID(parent.ID, "raw_bounds_runtime_check_normal_build"),
		FunctionID:       parent.FunctionID,
		ValueID:          parent.ValueID,
		SiteID:           parent.SiteID,
		SourceSpan:       parent.SourceSpan,
		TypeName:         parent.TypeName,
		SourceStage:      parent.SourceStage,
		ProvenanceClass:  ProvenanceUnsafeChecked,
		UnsafeClass:      UnsafeChecked,
		EscapeState:      parent.EscapeState,
		Claim:            "raw_bounds_runtime_check_normal_build",
		ValidationState:  ValidationPass,
		ValidatorName:    "raw_bounds_width_validator",
		CostClass:        CostDynamicCheckRequired,
		NormalBuildCheck: true,
		Reason:           "Memory Ideal v6 keeps raw bounds width/overflow uncertainty as a normal-build check or trap",
	})
	return err
}

func addUnsafeUnknownRejectedSafeFacts(graph *Graph, parent Fact) error {
	_, err := graph.DeriveFact(parent.ID, Fact{
		ID:              derivedFactID(parent.ID, "unsafe_unknown_rejected_safe_facts"),
		FunctionID:      parent.FunctionID,
		ValueID:         parent.ValueID,
		SiteID:          parent.SiteID,
		SourceSpan:      parent.SourceSpan,
		TypeName:        parent.TypeName,
		SourceStage:     parent.SourceStage,
		ProvenanceClass: ProvenanceUnsafeUnknown,
		UnsafeClass:     UnsafeUnknown,
		EscapeState:     EscapeConservative,
		Claim:           "unsafe_unknown_rejected_safe_facts",
		ValidationState: ValidationFail,
		ValidatorName:   "unsafe_unknown_fact_validator",
		CostClass:       CostUnsupportedRejected,
		Reason:          "Memory Ideal v5 rejects unsafe_unknown promotion to safe_known, provenance_known, or noalias facts",
	})
	return err
}

func costClassForUnsafeOperationClaim(claim string, provenance ProvenanceClass, unsafeClass UnsafeClass) CostClass {
	if provenance == ProvenanceUnsafeUnknown || unsafeClass == UnsafeUnknown {
		return CostConservativeFallback
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(claim)), "rejected_") {
		return CostUnsupportedRejected
	}
	switch claim {
	case "derived_allocation_offset", "raw_memory_access_checked", "raw_slice_verified_allocation_root", "unsafe_contract_runtime_checkable":
		return CostDynamicCheckRequired
	default:
		return CostInstrumentationOnly
	}
}

func costClassForAllocFact(claim string, alloc allocplan.Allocation) CostClass {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(claim)), "rejected_") {
		return CostUnsupportedRejected
	}
	if allocPlanRuntimeProofRequiredStorage(alloc.PlannedStorage, alloc.ActualLoweringStorage) {
		return CostConservativeFallback
	}
	if alloc.ActualLoweringStorage == allocplan.StorageHeap &&
		alloc.PlannedStorage != "" &&
		alloc.PlannedStorage != allocplan.StorageHeap {
		return CostConservativeFallback
	}
	if !allocPlanValidationPasses(alloc) {
		return CostInstrumentationOnly
	}
	if alloc.Builtin == "core.alloc_bytes" && claim == "allocation_base_metadata" {
		return CostZeroCostProven
	}
	switch claim {
	case "storage_lowering":
		return CostZeroCostProven
	default:
		return CostInstrumentationOnly
	}
}
