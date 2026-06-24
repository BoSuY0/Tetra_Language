package fromplir

import (
	"sort"
	"strings"

	"tetra_language/compiler/internal/plir"
)

type allocationReadOnlyCallSummary struct {
	Params            map[int]bool
	InoutWriterParams map[int]bool
}

func addAllocationIntentSummaryFacts(
	graph *Graph,
	fn plir.Function,
	values map[string]plir.Value,
	callSummaries map[string]allocationReadOnlyCallSummary,
) error {
	ordered := make([]plir.Value, 0, len(values))
	for _, value := range values {
		if value.Kind == plir.ValueAllocIntent && value.Alloc != nil {
			ordered = append(ordered, value)
		}
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	for _, value := range ordered {
		allocName := allocationNameForPLIRValue(value.ID)
		escape, reason := classifyAllocationEscapeForPLIR(fn, allocName, value, callSummaries)
		transfer := boundaryTransferEvidenceForAllocation(fn, allocName, value)
		fact := functionSummaryFactFor(
			fn,
			"alloc_escape:"+value.ID,
			claimForAllocationEscape(value, escape),
			nonEmpty(value.Source, summarySite(fn)),
			value,
			provenanceClassForAllocationIntent(value),
			unsafeClassForAllocationIntent(value),
			escape,
			AliasUnknown,
			ownerForPLIRValue(value),
			reason,
		)
		fact.ID = summaryFactID(fn.Name, "alloc_escape:"+value.ID, string(escape))
		fact.IslandID = islandIDForPLIRFact(plir.Fact{}, value)
		fact.Epoch = epochForPLIRFact(plir.Fact{}, value)
		fact.BaseID = baseIDForPLIRFact(plir.Fact{}, value)
		applyBoundaryTransferEvidence(&fact, transfer)
		if _, err := graph.AddFact(fact); err != nil {
			return err
		}
		if value.Provenance.Kind == plir.ProvenanceIsland {
			if _, err := graph.AddFact(explicitIslandTrustedStorageFact(fn.Name, value)); err != nil {
				return err
			}
		}
	}
	return nil
}

type boundaryTransferEvidence struct {
	DomainKind         DomainKind
	DomainOwnerID      string
	TransferKind       TransferKind
	TransferProofID    string
	SourceConsumed     bool
	LiveBorrowCrossing bool
	DestinationActive  bool
}

func applyBoundaryTransferEvidence(fact *Fact, evidence boundaryTransferEvidence) {
	if fact == nil || evidence.DomainKind == "" {
		return
	}
	fact.DomainKind = evidence.DomainKind
	fact.DomainOwnerID = evidence.DomainOwnerID
	fact.TransferKind = evidence.TransferKind
	fact.TransferProofID = evidence.TransferProofID
	fact.SourceConsumed = evidence.SourceConsumed
	fact.LiveBorrowCrossing = evidence.LiveBorrowCrossing
	fact.DestinationActive = evidence.DestinationActive
	if evidence.TransferProofID != "" {
		fact.ProofID = evidence.TransferProofID
		fact.ProofKind = ProofDomainMove
		fact.ProofSubjectBaseID = fact.ValueID
		fact.ProofOperation = "domain_move"
	}
}

func boundaryTransferEvidenceForAllocation(
	fn plir.Function,
	allocName string,
	value plir.Value,
) boundaryTransferEvidence {
	carriers := allocationCarriersForPLIR(fn, allocName, value.ID)
	moved := allocationCarrierHasMovedFact(fn, carriers)
	borrowed := allocationCarrierHasBorrowedFact(fn, carriers) || value.Borrow != plir.BorrowNone
	copied := isCopyAllocationValue(value)
	for _, op := range fn.Ops {
		if !opUsesAllocationCarrier(op, carriers) {
			continue
		}
		switch {
		case op.Kind == plir.OpActorSend:
			return boundaryTransferEvidenceForOperation(
				fn.Name,
				value.ID,
				DomainActor,
				ownerFromOperationInput(op, 0),
				op.ID,
				moved,
				borrowed,
				copied,
			)
		case op.Kind == plir.OpCall && isTaskEscapeOperation(op):
			return boundaryTransferEvidenceForOperation(
				fn.Name,
				value.ID,
				DomainTask,
				ownerFromTaskBoundaryOperation(op, carriers),
				op.ID,
				moved,
				borrowed,
				copied,
			)
		}
	}
	return boundaryTransferEvidence{}
}

func allocationHasTypedBoundaryOperation(fn plir.Function, allocName string, valueID string) bool {
	carriers := allocationCarriersForPLIR(fn, allocName, valueID)
	for _, op := range fn.Ops {
		if !opUsesAllocationCarrier(op, carriers) {
			continue
		}
		if op.Kind == plir.OpActorSend {
			return true
		}
		if op.Kind == plir.OpCall && isTaskEscapeOperation(op) {
			return true
		}
	}
	return false
}

func boundaryTransferEvidenceForOperation(
	functionName string,
	valueID string,
	domain DomainKind,
	owner string,
	opID string,
	moved bool,
	borrowed bool,
	copied bool,
) boundaryTransferEvidence {
	owner = strings.TrimSpace(owner)
	out := boundaryTransferEvidence{
		DomainKind:        domain,
		DomainOwnerID:     owner,
		DestinationActive: owner != "",
	}
	switch {
	case copied:
		out.TransferKind = TransferCopy
	case moved:
		out.TransferKind = TransferMove
		out.SourceConsumed = true
	default:
		out.TransferKind = TransferBorrowed
	}
	out.LiveBorrowCrossing = borrowed && out.TransferKind != TransferCopy
	if out.TransferKind == TransferMove && out.SourceConsumed && !out.LiveBorrowCrossing &&
		out.DestinationActive {
		out.TransferProofID = "proof:domain_move:" + functionName + ":" + valueID + ":" + opID
	}
	return out
}

func allocationCarrierHasMovedFact(fn plir.Function, carriers map[string]bool) bool {
	for _, fact := range fn.Facts {
		if fact.Kind == plir.FactMoved && inputCarriesAnyAllocationCarrier(fact.ValueID, carriers) {
			return true
		}
	}
	return false
}

func allocationCarrierHasBorrowedFact(fn plir.Function, carriers map[string]bool) bool {
	for _, fact := range fn.Facts {
		switch fact.Kind {
		case plir.FactBorrowedImm, plir.FactBorrowedMut:
			if inputCarriesAnyAllocationCarrier(fact.ValueID, carriers) {
				return true
			}
		}
	}
	return false
}

func ownerFromTaskBoundaryOperation(op plir.Operation, carriers map[string]bool) string {
	for _, input := range op.Inputs {
		if inputCarriesAnyAllocationCarrier(input, carriers) {
			continue
		}
		if !typedDomainOwnerInput(input, "task") {
			continue
		}
		if owner := normalizeOwnerID(input); owner != "" {
			return owner
		}
	}
	return ""
}

func explicitIslandTrustedStorageFact(functionName string, value plir.Value) Fact {
	baseID := baseIDForPLIRFact(plir.Fact{}, value)
	islandID := islandIDForPLIRFact(plir.Fact{}, value)
	epoch := epochForPLIRFact(plir.Fact{}, value)
	fact := functionSummaryFact(
		functionName,
		"explicit_island_storage:"+value.ID,
		ClaimTrustedStorage,
		nonEmpty(value.Source, functionName+":explicit_island_storage:"+value.ID),
		value,
		ProvenanceSafeOwned,
		UnsafeSafe,
		EscapeNoEscape,
		AliasUnique,
		ownerForPLIRValue(value),
		"explicit island allocation carries typed identity/epoch storage proof prerequisite",
	)
	fact.ID = summaryFactID(fact.FunctionID, "explicit_island_storage:"+fact.ValueID, "trusted")
	fact.IslandID = islandID
	fact.Epoch = epoch
	fact.BaseID = baseID
	fact.ProofID = "proof:storage:" + functionName + ":" + value.ID
	fact.ProofKind = ProofStorage
	fact.ProofSubjectBaseID = baseID
	fact.ProofOperation = "explicit_island_storage"
	return fact
}

func provenanceClassForAllocationIntent(value plir.Value) ProvenanceClass {
	if value.Alloc != nil && value.Alloc.Builtin == "core.alloc_bytes" {
		return ProvenanceUnsafeVerifiedRoot
	}
	switch value.Provenance.Kind {
	case plir.ProvenanceExternal, plir.ProvenanceUnknown:
		return ProvenanceUnsafeUnknown
	default:
		return ProvenanceSafeOwned
	}
}

func unsafeClassForAllocationIntent(value plir.Value) UnsafeClass {
	if value.Alloc != nil && value.Alloc.Builtin == "core.alloc_bytes" {
		return UnsafeVerifiedRoot
	}
	switch value.Provenance.Kind {
	case plir.ProvenanceExternal, plir.ProvenanceUnknown:
		return UnsafeUnknown
	default:
		return UnsafeSafe
	}
}

func claimForAllocationEscape(value plir.Value, escape EscapeState) string {
	if value.Alloc != nil && value.Alloc.Builtin == "core.alloc_bytes" {
		return "allocation_base_metadata"
	}
	switch escape {
	case EscapeNoEscape:
		return "no_escape"
	case EscapeReturn:
		return "returns_owned_new_allocation"
	case EscapeGlobal:
		return "may_store_global"
	case EscapeActor:
		return "may_escape_to_actor"
	case EscapeTask:
		return "may_escape_to_task"
	case EscapeUnsafe:
		return "may_retain_pointer"
	default:
		return "unknown_external_call_conservative"
	}
}

func classifyAllocationEscapeForPLIR(
	fn plir.Function,
	allocName string,
	value plir.Value,
	callSummaries map[string]allocationReadOnlyCallSummary,
) (EscapeState, string) {
	if value.Provenance.Kind == plir.ProvenanceIsland {
		return EscapeNoEscape, "explicit island allocation is bounded by typed island scope evidence"
	}
	if allocName == "$return" {
		return EscapeReturn, "allocation is returned directly"
	}
	if value.Escape == plir.EscapeNoEscape &&
		!allocationHasTypedBoundaryOperation(fn, allocName, value.ID) {
		return EscapeNoEscape, "PLIR value carries typed no-escape state"
	}
	if value.Escape != "" && value.Escape != plir.EscapeNoEscape &&
		value.Escape != plir.EscapeConservative {
		return escapeStateForPLIRValue(value), "PLIR value carries typed escape state"
	}
	carriers := allocationCarriersForPLIR(fn, allocName, value.ID)
	unsafeBoundary := false
	aggregateBoundary := false
	closureBoundary := false
	readOnlyCallSummaryProof := false
	inoutWriterCallSummaryProof := false
	for _, op := range fn.Ops {
		if opUsesAllocationCarrier(op, carriers) {
			switch op.Kind {
			case plir.OpReturn:
				return EscapeReturn, "allocation value is returned from the function"
			case plir.OpGlobalStore:
				return EscapeGlobal, "allocation value is stored in global state"
			case plir.OpClosure:
				closureBoundary = true
			case plir.OpAggregate:
				if allocationContainsString(op.Outputs, "$return") {
					return EscapeReturn, "allocation is embedded in a returned aggregate"
				}
				aggregateBoundary = true
			case plir.OpActorSend:
				return EscapeActor, "allocation crosses a typed actor/send boundary"
			case plir.OpCall:
				if isNonEscapingAllocationBuiltinCall(op.Note) {
					continue
				}
				if isTaskEscapeOperation(op) {
					return EscapeTask, "allocation crosses a typed task boundary"
				}
				if proof, ok := allocationCallInputsCoveredByLocalNoEscapeSummary(
					op,
					carriers,
					callSummaries,
				); ok {
					readOnlyCallSummaryProof = readOnlyCallSummaryProof || proof.readOnly
					inoutWriterCallSummaryProof = inoutWriterCallSummaryProof || proof.inoutWriter
					continue
				}
				return EscapeConservative, "allocation is passed to a call without typed interprocedural escape evidence"
			}
		}
		if op.Kind == plir.OpUnsafe {
			unsafeBoundary = true
		}
	}
	if closureBoundary {
		return EscapeConservative, "allocation is captured by a closure environment"
	}
	if aggregateBoundary {
		return EscapeConservative, "allocation is stored inside an aggregate value"
	}
	if unsafeBoundary {
		return EscapeUnsafe, "function contains unsafe boundary evidence"
	}
	if readOnlyCallSummaryProof && inoutWriterCallSummaryProof {
		return EscapeNoEscape, "allocation is covered by typed read-only and inout-writer local call summaries"
	}
	if inoutWriterCallSummaryProof {
		return EscapeNoEscape, "allocation is covered by typed inout-writer local call summaries"
	}
	if readOnlyCallSummaryProof {
		return EscapeNoEscape, "allocation is covered by typed read-only local call summaries"
	}
	return EscapeNoEscape, "allocation does not escape in the PLIR allocation evidence scan"
}

type allocationLocalCallSummaryProof struct {
	readOnly    bool
	inoutWriter bool
}

func allocationCallInputsCoveredByLocalNoEscapeSummary(
	op plir.Operation,
	carriers map[string]bool,
	summaries map[string]allocationReadOnlyCallSummary,
) (allocationLocalCallSummaryProof, bool) {
	callee := allocationLocalCallSummaryName(op.Note)
	if callee == "" {
		return allocationLocalCallSummaryProof{}, false
	}
	summary, ok := summaries[callee]
	if !ok || (len(summary.Params) == 0 && len(summary.InoutWriterParams) == 0) {
		return allocationLocalCallSummaryProof{}, false
	}
	proof := allocationLocalCallSummaryProof{}
	matched := false
	for i, input := range op.Inputs {
		if !inputCarriesAnyAllocationCarrier(input, carriers) {
			continue
		}
		matched = true
		if summary.Params[i] {
			proof.readOnly = true
			continue
		}
		if summary.InoutWriterParams[i] {
			proof.inoutWriter = true
			continue
		}
		return allocationLocalCallSummaryProof{}, false
	}
	return proof, matched
}

func buildAllocationReadOnlyCallSummaries(
	prog *plir.Program,
) map[string]allocationReadOnlyCallSummary {
	if prog == nil {
		return nil
	}
	out := map[string]allocationReadOnlyCallSummary{}
	for _, fn := range prog.Funcs {
		summary := allocationReadOnlyCallSummary{
			Params:            map[int]bool{},
			InoutWriterParams: map[int]bool{},
		}
		if fn.Summary == nil || fn.Summary.Async || fn.Summary.TouchesMutableGlobals {
			continue
		}
		for i, name := range fn.Summary.ParamNames {
			if !allocationMemoryBearingParamType(fn.Summary, i) {
				continue
			}
			if allocationParamHasReadOnlyNoEscapeUse(fn, name) {
				summary.Params[i] = true
			}
			if allocationParamHasInoutWriterNoEscapeUse(fn, i, name) {
				summary.InoutWriterParams[i] = true
			}
		}
		if len(summary.Params) > 0 || len(summary.InoutWriterParams) > 0 {
			out[fn.Name] = summary
		}
	}
	return out
}

func allocationParamHasReadOnlyNoEscapeUse(fn plir.Function, paramName string) bool {
	if strings.TrimSpace(paramName) == "" {
		return false
	}
	carriers := allocationCarriersForPLIR(fn, paramName, string(plir.ValueParam)+":"+paramName)
	for _, op := range fn.Ops {
		if op.Kind == plir.OpUnsafe {
			return false
		}
		if !opUsesAllocationCarrier(op, carriers) {
			continue
		}
		switch op.Kind {
		case plir.OpAssign, plir.OpGuard, plir.OpIndexLoad, plir.OpSliceWindow:
			continue
		case plir.OpCall:
			if isNonEscapingAllocationBuiltinCall(op.Note) {
				continue
			}
			return false
		default:
			return false
		}
	}
	return true
}

func allocationParamHasInoutWriterNoEscapeUse(
	fn plir.Function,
	index int,
	paramName string,
) bool {
	if fn.Summary == nil || !allocationMemoryBearingParamType(fn.Summary, index) ||
		!allocationSummaryParamOwnershipIs(fn.Summary, index, "inout") {
		return false
	}
	if strings.TrimSpace(paramName) == "" {
		return false
	}
	carriers := allocationCarriersForPLIR(fn, paramName, string(plir.ValueParam)+":"+paramName)
	sawCarrierUse := false
	for _, op := range fn.Ops {
		if op.Kind == plir.OpUnsafe {
			return false
		}
		if !opUsesAllocationCarrier(op, carriers) {
			continue
		}
		sawCarrierUse = true
		switch op.Kind {
		case plir.OpAssign, plir.OpGuard, plir.OpIndexLoad, plir.OpSliceWindow:
			continue
		case plir.OpIndexStore:
			if allocationIndexStoreUsesCarrierOnlyAsBase(op, carriers) {
				continue
			}
			return false
		case plir.OpCall:
			if isNonEscapingAllocationBuiltinCall(op.Note) {
				continue
			}
			return false
		default:
			return false
		}
	}
	return sawCarrierUse
}

func allocationCarriersForPLIR(
	fn plir.Function,
	allocName string,
	valueID string,
) map[string]bool {
	carriers := map[string]bool{}
	addAllocationCarrier(carriers, allocName)
	addAllocationCarrier(carriers, valueID)
	changed := true
	for changed {
		changed = false
		for _, op := range fn.Ops {
			switch op.Kind {
			case plir.OpAssign, plir.OpAggregate, plir.OpSliceWindow:
			case plir.OpCall:
				if !isAllocationBorrowViewOperation(op.Note) {
					continue
				}
			default:
				continue
			}
			if !allocationInputsUseCarrier(op.Inputs, carriers) {
				continue
			}
			for _, output := range op.Outputs {
				if addAllocationCarrier(carriers, output) {
					changed = true
				}
			}
		}
	}
	return carriers
}

func addAllocationCarrier(carriers map[string]bool, name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	changed := false
	for _, candidate := range allocationCarrierAliases(name) {
		if candidate == "" || carriers[candidate] {
			continue
		}
		carriers[candidate] = true
		changed = true
	}
	return changed
}

func allocationCarrierAliases(name string) []string {
	aliases := []string{name}
	if idx := strings.Index(name, ":"); idx >= 0 && idx+1 < len(name) {
		aliases = append(aliases, name[idx+1:])
	}
	return aliases
}

func opUsesAllocationCarrier(op plir.Operation, carriers map[string]bool) bool {
	return allocationInputsUseCarrier(op.Inputs, carriers)
}

func allocationInputsUseCarrier(inputs []string, carriers map[string]bool) bool {
	for _, input := range inputs {
		if inputCarriesAnyAllocationCarrier(input, carriers) {
			return true
		}
	}
	return false
}

func inputCarriesAnyAllocationCarrier(input string, carriers map[string]bool) bool {
	for carrier := range carriers {
		if allocationInputCarriesValue(input, carrier) {
			return true
		}
	}
	return false
}

func allocationInputCarriesValue(input string, carrier string) bool {
	if !allocationInputUses(input, carrier) {
		return false
	}
	return input != carrier+".len" && !strings.HasSuffix(input, ".len")
}

func allocationInputUses(input string, allocName string) bool {
	input = strings.TrimSpace(input)
	allocName = strings.TrimSpace(allocName)
	if input == "" || allocName == "" {
		return false
	}
	if input == allocName || input == "alloc_intent:"+allocName {
		return true
	}
	prefixes := []string{
		allocName + ".",
		"alloc_intent:" + allocName + ".",
		"view:" + allocName + ".",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(input, prefix) {
			return true
		}
	}
	return false
}

func allocationNameForPLIRValue(valueID string) string {
	valueID = strings.TrimSpace(valueID)
	if strings.HasPrefix(valueID, "alloc_intent:") {
		return strings.TrimPrefix(valueID, "alloc_intent:")
	}
	return valueID
}

func allocationLocalCallSummaryName(note string) string {
	note = strings.TrimSpace(note)
	if note == "" {
		return ""
	}
	lower := strings.ToLower(note)
	if strings.Contains(lower, "unknown external") || strings.Contains(lower, "external call") ||
		strings.Contains(lower, "alias_boundary:") {
		return ""
	}
	fields := strings.Fields(note)
	if len(fields) == 0 {
		return ""
	}
	name := fields[0]
	if strings.HasPrefix(name, "core.") || strings.HasPrefix(name, "ffi.") {
		return ""
	}
	return name
}

func isNonEscapingAllocationBuiltinCall(note string) bool {
	return isAllocationBorrowViewOperation(note) ||
		strings.Contains(note, "copies into caller-owned destination without allocation")
}

func isAllocationBorrowViewOperation(note string) bool {
	return strings.Contains(note, "creates borrowed view without allocation")
}

func allocationMemoryBearingParamType(summary *plir.FunctionSummary, index int) bool {
	if summary == nil || index < 0 || index >= len(summary.ParamTypes) {
		return false
	}
	typeName := strings.TrimSpace(summary.ParamTypes[index])
	return strings.HasPrefix(typeName, "[]") || typeName == "str" || typeName == "String"
}

func allocationSummaryParamOwnershipIs(
	summary *plir.FunctionSummary,
	index int,
	want string,
) bool {
	if summary == nil || index < 0 || index >= len(summary.ParamOwnership) {
		return false
	}
	return strings.TrimSpace(summary.ParamOwnership[index]) == want
}

func allocationIndexStoreUsesCarrierOnlyAsBase(op plir.Operation, carriers map[string]bool) bool {
	if len(op.Inputs) == 0 || !inputCarriesAnyAllocationCarrier(op.Inputs[0], carriers) {
		return false
	}
	for _, input := range op.Inputs[1:] {
		if inputCarriesAnyAllocationCarrier(input, carriers) {
			return false
		}
	}
	return true
}

func allocationContainsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
