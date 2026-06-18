package plir

import (
	"fmt"
	"sort"
	"strings"
)

func VerifyProgram(prog *Program) error {
	if prog == nil {
		return fmt.Errorf("plir verifier: missing program")
	}
	names := map[string]bool{}
	for _, fn := range prog.Funcs {
		if fn.Name == "" {
			return fmt.Errorf("plir verifier: function with empty name")
		}
		if names[fn.Name] {
			return fmt.Errorf("plir verifier: duplicate function %q", fn.Name)
		}
		names[fn.Name] = true
		if err := VerifyFunction(fn); err != nil {
			return err
		}
	}
	return nil
}

func VerifyFunction(fn Function) error {
	values := map[string]Value{}
	for _, value := range fn.Values {
		if value.ID == "" {
			return fmt.Errorf("plir verifier: %s has value with empty id", fn.Name)
		}
		if _, exists := values[value.ID]; exists {
			return fmt.Errorf("plir verifier: %s duplicate value %q", fn.Name, value.ID)
		}
		if value.Type == "" {
			return fmt.Errorf("plir verifier: %s value %q has empty type", fn.Name, value.ID)
		}
		if value.Kind == ValueAllocIntent {
			if value.Alloc == nil {
				return fmt.Errorf(
					"plir verifier: %s alloc intent %q has nil alloc metadata",
					fn.Name,
					value.ID,
				)
			}
			if value.Alloc.ElementType == "" || value.Alloc.ElementSize <= 0 {
				return fmt.Errorf(
					"plir verifier: %s alloc intent %q has invalid element metadata",
					fn.Name,
					value.ID,
				)
			}
			if value.Alloc.LengthExpr == "" {
				return fmt.Errorf(
					"plir verifier: %s alloc intent %q has empty length expression",
					fn.Name,
					value.ID,
				)
			}
			if value.Alloc.ZeroGuardStatus == "" || value.Alloc.NegativeGuardStatus == "" ||
				value.Alloc.OverflowGuardStatus == "" {
				return fmt.Errorf(
					"plir verifier: %s alloc intent %q missing allocation length guard status",
					fn.Name,
					value.ID,
				)
			}
		}
		values[value.ID] = value
	}
	blocks, ops, err := verifyCFG(fn)
	if err != nil {
		return err
	}
	proofGuards, err := verifyProofWiring(fn, blocks, ops)
	if err != nil {
		return err
	}
	proofTerms, err := verifyProofTerms(fn, proofGuards)
	if err != nil {
		return err
	}
	rangeProofs := map[string]bool{}
	indexFactsByProof := map[string][]Fact{}
	for _, fact := range fn.RangeFacts {
		if fact.Value == "" {
			return fmt.Errorf("plir verifier: %s range fact has empty value", fn.Name)
		}
		if fact.Source == "" {
			return fmt.Errorf(
				"plir verifier: %s range fact for %q requires source",
				fn.Name,
				fact.Value,
			)
		}
		if fact.Lower.Kind == BoundConst && fact.Upper.Kind == BoundConst &&
			fact.Lower.Const > fact.Upper.Const {
			return fmt.Errorf(
				"plir verifier: %s range fact for %q lower bound exceeds upper bound",
				fn.Name,
				fact.Value,
			)
		}
		if fact.ProofID != "" {
			if _, ok := proofGuards[fact.ProofID]; !ok {
				return fmt.Errorf(
					"plir verifier: %s range fact for %q references unknown proof id %q",
					fn.Name,
					fact.Value,
					fact.ProofID,
				)
			}
			rangeProofs[fact.ProofID] = true
		}
	}
	for _, fact := range fn.Facts {
		if fact.ID == "" {
			return fmt.Errorf("plir verifier: %s has fact with empty id", fn.Name)
		}
		value, hasValue := values[fact.ValueID]
		if fact.ValueID != "" && !hasValue {
			return fmt.Errorf(
				"plir verifier: %s fact %q references unknown value %q",
				fn.Name,
				fact.ID,
				fact.ValueID,
			)
		}
		if fact.IslandID != "" {
			if fact.Epoch <= 0 {
				return fmt.Errorf(
					"plir verifier: %s fact %q island_id requires positive epoch",
					fn.Name,
					fact.ID,
				)
			}
			if fact.BaseID == "" {
				return fmt.Errorf(
					"plir verifier: %s fact %q island_id requires base_id",
					fn.Name,
					fact.ID,
				)
			}
			if hasValue && value.Provenance.Kind != ProvenanceIsland {
				return fmt.Errorf(
					"plir verifier: %s fact %q island_id contradicts non-island value %q",
					fn.Name,
					fact.ID,
					fact.ValueID,
				)
			}
		}
		if fact.Epoch > 0 && fact.IslandID == "" {
			return fmt.Errorf(
				"plir verifier: %s fact %q epoch requires island_id",
				fn.Name,
				fact.ID,
			)
		}
		switch fact.Kind {
		case FactLenStable:
			if !hasValue {
				return fmt.Errorf(
					"plir verifier: %s fact %q len_stable requires a value",
					fn.Name,
					fact.ID,
				)
			}
			if value.Provenance.Kind == ProvenanceUnknown || value.Provenance.Kind == "" {
				return fmt.Errorf(
					"plir verifier: %s fact %q len_stable requires known provenance for %q",
					fn.Name,
					fact.ID,
					fact.ValueID,
				)
			}
		case FactIndexInRange:
			if !hasValue {
				return fmt.Errorf(
					"plir verifier: %s fact %q index_in_range requires an index value",
					fn.Name,
					fact.ID,
				)
			}
			if fact.Range == "" {
				return fmt.Errorf(
					"plir verifier: %s fact %q index_in_range requires a range",
					fn.Name,
					fact.ID,
				)
			}
			if fact.ProofID == "" {
				return fmt.Errorf(
					"plir verifier: %s fact %q index_in_range requires proof id",
					fn.Name,
					fact.ID,
				)
			}
			if _, ok := proofGuards[fact.ProofID]; !ok {
				return fmt.Errorf(
					"plir verifier: %s fact %q index_in_range references unknown proof id %q",
					fn.Name,
					fact.ID,
					fact.ProofID,
				)
			}
			rangeProofs[fact.ProofID] = true
			indexFactsByProof[fact.ProofID] = append(indexFactsByProof[fact.ProofID], fact)
		case FactProvenanceKnown:
			if !hasValue {
				return fmt.Errorf(
					"plir verifier: %s fact %q provenance_known requires a value",
					fn.Name,
					fact.ID,
				)
			}
			if value.Provenance.Kind == ProvenanceUnknown || value.Provenance.Kind == "" {
				return fmt.Errorf(
					"plir verifier: %s fact %q provenance_known contradicts unknown provenance for %q",
					fn.Name,
					fact.ID,
					fact.ValueID,
				)
			}
		case FactProvenanceUnknown:
			if !hasValue {
				return fmt.Errorf(
					"plir verifier: %s fact %q provenance_unknown requires a value",
					fn.Name,
					fact.ID,
				)
			}
			if value.Provenance.Kind != ProvenanceUnknown &&
				value.Provenance.Kind != ProvenanceExternal {
				return fmt.Errorf(
					"plir verifier: %s fact %q provenance_unknown contradicts known provenance for %q",
					fn.Name,
					fact.ID,
					fact.ValueID,
				)
			}
		case FactNoAlias:
			if !hasValue {
				return fmt.Errorf(
					"plir verifier: %s fact %q no_alias requires a value",
					fn.Name,
					fact.ID,
				)
			}
			if value.Borrow != BorrowMut {
				return fmt.Errorf(
					"plir verifier: %s fact %q no_alias requires mutable borrow for %q",
					fn.Name,
					fact.ID,
					fact.ValueID,
				)
			}
			if value.Kind != ValueParam || value.Provenance.Kind != ProvenanceParam {
				return fmt.Errorf(
					"plir verifier: %s fact %q no_alias requires parameter provenance for %q",
					fn.Name,
					fact.ID,
					fact.ValueID,
				)
			}
			if value.Lifetime.Birth == "" || value.Lifetime.Death == "" ||
				value.Lifetime.Owner == "" {
				return fmt.Errorf(
					"plir verifier: %s fact %q no_alias requires bounded lifetime for %q",
					fn.Name,
					fact.ID,
					fact.ValueID,
				)
			}
		case FactDerivedWindow:
			if !hasValue {
				return fmt.Errorf(
					"plir verifier: %s fact %q derived_window requires a value",
					fn.Name,
					fact.ID,
				)
			}
			if fact.Range == "" {
				return fmt.Errorf(
					"plir verifier: %s fact %q derived_window requires a range",
					fn.Name,
					fact.ID,
				)
			}
			if fact.Source == "" {
				return fmt.Errorf(
					"plir verifier: %s fact %q derived_window requires source",
					fn.Name,
					fact.ID,
				)
			}
		case FactMoved:
			if !hasValue {
				return fmt.Errorf(
					"plir verifier: %s fact %q moved requires a value",
					fn.Name,
					fact.ID,
				)
			}
			if strings.TrimSpace(fact.Source) == "" {
				return fmt.Errorf(
					"plir verifier: %s fact %q moved requires source",
					fn.Name,
					fact.ID,
				)
			}
			if strings.TrimSpace(fact.Reason) == "" {
				return fmt.Errorf(
					"plir verifier: %s fact %q moved requires reason",
					fn.Name,
					fact.ID,
				)
			}
			if value.Borrow == BorrowImm || value.Borrow == BorrowMut {
				return fmt.Errorf(
					"plir verifier: %s fact %q moved contradicts borrowed value %q",
					fn.Name,
					fact.ID,
					fact.ValueID,
				)
			}
		case FactIslandEpochAdvanced:
			if fact.IslandID == "" || fact.Epoch <= 0 || fact.BaseID == "" {
				return fmt.Errorf(
					("plir verifier: %s fact %q island_epoch_advanced requires island_" +
						"id, positive epoch, and base_id"),
					fn.Name,
					fact.ID,
				)
			}
			if strings.TrimSpace(fact.Source) == "" {
				return fmt.Errorf(
					"plir verifier: %s fact %q island_epoch_advanced requires source",
					fn.Name,
					fact.ID,
				)
			}
			if strings.TrimSpace(fact.Reason) == "" {
				return fmt.Errorf(
					"plir verifier: %s fact %q island_epoch_advanced requires reason",
					fn.Name,
					fact.ID,
				)
			}
		}
	}
	for _, use := range fn.ProofUses {
		if !rangeProofs[use.ProofID] {
			return fmt.Errorf(
				"plir verifier: %s proof use %q has no explicit range proof fact",
				fn.Name,
				use.ProofID,
			)
		}
		if _, ok := proofTerms[use.ProofID]; !ok {
			return fmt.Errorf(
				"plir verifier: %s proof use %q has no typed proof term",
				fn.Name,
				use.ProofID,
			)
		}
	}
	for _, term := range fn.ProofTerms {
		if term.Kind != "bounds_check" {
			continue
		}
		if !typedProofTermMatchesIndexFact(term, indexFactsByProof[term.ID]) {
			return fmt.Errorf(
				("plir verifier: %s typed proof term %q subject base/index/range " +
					"does not match index_in_range fact"),
				fn.Name,
				term.ID,
			)
		}
	}
	if err := verifyFunctionFactConsistency(fn, values); err != nil {
		return err
	}
	if err := verifyFunctionSummaryCompleteness(fn, values); err != nil {
		return err
	}
	return nil
}

func verifyProofTerms(fn Function, guards map[string]ProofGuard) (map[string]ProofTerm, error) {
	terms := map[string]ProofTerm{}
	for _, term := range fn.ProofTerms {
		if strings.TrimSpace(term.ID) == "" {
			return nil, fmt.Errorf("plir verifier: %s has typed proof term with empty id", fn.Name)
		}
		if _, exists := terms[term.ID]; exists {
			return nil, fmt.Errorf(
				"plir verifier: %s duplicate typed proof term %q",
				fn.Name,
				term.ID,
			)
		}
		if _, ok := guards[term.ID]; !ok {
			return nil, fmt.Errorf(
				"plir verifier: %s typed proof term %q references unknown proof guard",
				fn.Name,
				term.ID,
			)
		}
		switch term.Kind {
		case "bounds_check":
			if strings.TrimSpace(term.SubjectBaseID) == "" {
				return nil, fmt.Errorf(
					"plir verifier: %s typed proof term %q requires subject base",
					fn.Name,
					term.ID,
				)
			}
			if strings.TrimSpace(term.IndexValueID) == "" {
				return nil, fmt.Errorf(
					"plir verifier: %s typed proof term %q requires index value",
					fn.Name,
					term.ID,
				)
			}
			if strings.TrimSpace(term.Operation) == "" {
				return nil, fmt.Errorf(
					"plir verifier: %s typed proof term %q requires operation",
					fn.Name,
					term.ID,
				)
			}
			if strings.TrimSpace(term.Range) == "" {
				return nil, fmt.Errorf(
					"plir verifier: %s typed proof term %q requires range",
					fn.Name,
					term.ID,
				)
			}
		default:
			return nil, fmt.Errorf(
				"plir verifier: %s typed proof term %q has unknown kind %q",
				fn.Name,
				term.ID,
				term.Kind,
			)
		}
		if term.Epoch > 0 && strings.TrimSpace(term.IslandID) == "" {
			return nil, fmt.Errorf(
				"plir verifier: %s typed proof term %q epoch requires island_id",
				fn.Name,
				term.ID,
			)
		}
		if strings.TrimSpace(term.IslandID) != "" &&
			(term.Epoch <= 0 || strings.TrimSpace(term.BaseID) == "") {
			return nil, fmt.Errorf(
				"plir verifier: %s typed proof term %q island proof requires positive epoch and base_id",
				fn.Name,
				term.ID,
			)
		}
		terms[term.ID] = term
	}
	return terms, nil
}

func typedProofTermMatchesIndexFact(term ProofTerm, facts []Fact) bool {
	for _, fact := range facts {
		if fact.ValueID != term.IndexValueID {
			continue
		}
		if fact.Range != term.Range {
			continue
		}
		if rangeSubjectBase(fact.Range) != term.SubjectBaseID {
			continue
		}
		return true
	}
	return false
}

func rangeSubjectBase(rangeText string) string {
	text := strings.TrimSpace(rangeText)
	if parts := strings.Split(text, ".."); len(parts) == 2 {
		return baseFromRangeUpper(parts[1])
	}
	if comma := strings.LastIndex(text, ","); comma >= 0 {
		upper := strings.TrimRight(strings.TrimSpace(text[comma+1:]), ")] ")
		return baseFromRangeUpper(upper)
	}
	return ""
}

func baseFromRangeUpper(upper string) string {
	upper = strings.TrimSpace(upper)
	if minus := strings.Index(upper, " - "); minus >= 0 {
		upper = strings.TrimSpace(upper[:minus])
	}
	upper = strings.TrimSuffix(upper, ".len")
	return strings.TrimSpace(upper)
}

func VerifyFunctionSummaryCompleteness(fn Function) error {
	values := map[string]Value{}
	for _, value := range fn.Values {
		if value.ID == "" {
			continue
		}
		values[value.ID] = value
	}
	return verifyFunctionSummaryCompleteness(fn, values)
}

func verifyCFG(fn Function) (map[string]BasicBlock, map[string]Operation, error) {
	blocks := map[string]BasicBlock{}
	for _, block := range fn.Blocks {
		if block.ID == "" {
			return nil, nil, fmt.Errorf("plir verifier: %s has block with empty id", fn.Name)
		}
		if _, exists := blocks[block.ID]; exists {
			return nil, nil, fmt.Errorf("plir verifier: %s duplicate block %q", fn.Name, block.ID)
		}
		blocks[block.ID] = block
	}
	ops := map[string]Operation{}
	for _, op := range fn.Ops {
		if op.ID == "" {
			return nil, nil, fmt.Errorf("plir verifier: %s has operation with empty id", fn.Name)
		}
		if _, exists := ops[op.ID]; exists {
			return nil, nil, fmt.Errorf("plir verifier: %s duplicate operation %q", fn.Name, op.ID)
		}
		if op.Block != "" {
			if _, ok := blocks[op.Block]; !ok {
				return nil, nil, fmt.Errorf(
					"plir verifier: %s operation %q references unknown block %q",
					fn.Name,
					op.ID,
					op.Block,
				)
			}
		}
		ops[op.ID] = op
	}
	for _, block := range fn.Blocks {
		for _, pred := range block.Preds {
			if _, ok := blocks[pred]; !ok {
				return nil, nil, fmt.Errorf(
					"plir verifier: %s block %q references unknown predecessor %q",
					fn.Name,
					block.ID,
					pred,
				)
			}
		}
		for _, succ := range block.Succs {
			if _, ok := blocks[succ]; !ok {
				return nil, nil, fmt.Errorf(
					"plir verifier: %s block %q references unknown successor %q",
					fn.Name,
					block.ID,
					succ,
				)
			}
		}
		for _, opID := range block.Ops {
			op, ok := ops[opID]
			if !ok {
				return nil, nil, fmt.Errorf(
					"plir verifier: %s block %q references unknown operation %q",
					fn.Name,
					block.ID,
					opID,
				)
			}
			if op.Block != "" && op.Block != block.ID {
				return nil, nil, fmt.Errorf(
					"plir verifier: %s block %q contains operation %q assigned to block %q",
					fn.Name,
					block.ID,
					opID,
					op.Block,
				)
			}
		}
	}
	return blocks, ops, nil
}

func verifyProofWiring(
	fn Function,
	blocks map[string]BasicBlock,
	ops map[string]Operation,
) (map[string]ProofGuard, error) {
	guards := map[string]ProofGuard{}
	for _, guard := range fn.ProofGuards {
		if guard.ID == "" {
			return nil, fmt.Errorf("plir verifier: %s has proof guard with empty id", fn.Name)
		}
		if _, exists := guards[guard.ID]; exists {
			return nil, fmt.Errorf("plir verifier: %s duplicate proof id %q", fn.Name, guard.ID)
		}
		if guard.Block == "" {
			return nil, fmt.Errorf(
				"plir verifier: %s proof guard %q missing block",
				fn.Name,
				guard.ID,
			)
		}
		if _, ok := blocks[guard.Block]; !ok {
			return nil, fmt.Errorf(
				"plir verifier: %s proof guard %q references unknown block %q",
				fn.Name,
				guard.ID,
				guard.Block,
			)
		}
		if guard.OpID != "" {
			if _, ok := ops[guard.OpID]; !ok {
				return nil, fmt.Errorf(
					"plir verifier: %s proof guard %q references unknown operation %q",
					fn.Name,
					guard.ID,
					guard.OpID,
				)
			}
		}
		guards[guard.ID] = guard
	}
	for _, use := range fn.ProofUses {
		guard, ok := guards[use.ProofID]
		if !ok {
			return nil, fmt.Errorf(
				"plir verifier: %s proof use references unknown proof id %q",
				fn.Name,
				use.ProofID,
			)
		}
		if use.Block == "" {
			return nil, fmt.Errorf(
				"plir verifier: %s proof use %q missing block",
				fn.Name,
				use.ProofID,
			)
		}
		if _, ok := blocks[use.Block]; !ok {
			return nil, fmt.Errorf(
				"plir verifier: %s proof use %q references unknown block %q",
				fn.Name,
				use.ProofID,
				use.Block,
			)
		}
		if use.OpID != "" {
			if _, ok := ops[use.OpID]; !ok {
				return nil, fmt.Errorf(
					"plir verifier: %s proof use %q references unknown operation %q",
					fn.Name,
					use.ProofID,
					use.OpID,
				)
			}
		}
		if !Dominates(fn, guard.Block, use.Block) {
			return nil, fmt.Errorf(
				"plir verifier: %s proof id %q guard block %q does not dominate use block %q",
				fn.Name,
				use.ProofID,
				guard.Block,
				use.Block,
			)
		}
	}
	return guards, nil
}

func verifyFunctionFactConsistency(fn Function, values map[string]Value) error {
	provenanceKnown := map[string]string{}
	provenanceUnknown := map[string]string{}
	noEscape := map[string]string{}
	regionAlive := map[string]string{}
	owned := map[string]string{}
	moved := map[string]string{}
	borrowed := map[string]Fact{}
	noAlias := map[string]Fact{}
	for _, fact := range fn.Facts {
		value, hasValue := values[fact.ValueID]
		switch fact.Kind {
		case FactOwned:
			owned[fact.ValueID] = fact.ID
			if hasValue && value.Borrow != BorrowNone {
				return fmt.Errorf(
					"plir verifier: %s fact %q owned contradicts borrowed value %q",
					fn.Name,
					fact.ID,
					fact.ValueID,
				)
			}
		case FactBorrowedImm, FactBorrowedMut:
			borrowed[fact.ValueID] = fact
			if previous := moved[fact.ValueID]; previous != "" {
				return fmt.Errorf(
					"plir verifier: %s fact %q %s contradicts moved fact %q for %q",
					fn.Name,
					fact.ID,
					fact.Kind.String(),
					previous,
					fact.ValueID,
				)
			}
			if hasValue && value.Borrow == BorrowNone {
				return fmt.Errorf(
					"plir verifier: %s fact %q %s contradicts non-borrowed value %q",
					fn.Name,
					fact.ID,
					fact.Kind.String(),
					fact.ValueID,
				)
			}
		case FactNoEscape:
			noEscape[fact.ValueID] = fact.ID
			if previous := moved[fact.ValueID]; previous != "" {
				return fmt.Errorf(
					"plir verifier: %s fact %q no_escape contradicts moved fact %q for %q",
					fn.Name,
					fact.ID,
					previous,
					fact.ValueID,
				)
			}
			if hasValue && value.Escape != "" && value.Escape != EscapeNoEscape {
				return fmt.Errorf(
					"plir verifier: %s fact %q no_escape contradicts escaping value %q",
					fn.Name,
					fact.ID,
					fact.ValueID,
				)
			}
		case FactMoved:
			moved[fact.ValueID] = fact.ID
			if previous := borrowed[fact.ValueID]; previous.ID != "" {
				return fmt.Errorf(
					"plir verifier: %s fact %q moved contradicts %s fact %q for %q",
					fn.Name,
					fact.ID,
					previous.Kind.String(),
					previous.ID,
					fact.ValueID,
				)
			}
			if previous := noEscape[fact.ValueID]; previous != "" {
				return fmt.Errorf(
					"plir verifier: %s fact %q moved contradicts no_escape fact %q for %q",
					fn.Name,
					fact.ID,
					previous,
					fact.ValueID,
				)
			}
		case FactRegionAlive:
			regionAlive[fact.ValueID] = fact.ID
		case FactNoAlias:
			noAlias[fact.ValueID] = fact
		case FactNoHeapAllocation:
			if hasValue && (value.Kind == ValueAllocIntent || value.Alloc != nil) {
				return fmt.Errorf(
					"plir verifier: %s fact %q no_heap_allocation contradicts allocation intent %q",
					fn.Name,
					fact.ID,
					fact.ValueID,
				)
			}
		case FactProvenanceKnown:
			if previous := provenanceUnknown[fact.ValueID]; previous != "" {
				return fmt.Errorf(
					"plir verifier: %s fact %q provenance_known contradicts provenance_unknown fact %q for %q",
					fn.Name,
					fact.ID,
					previous,
					fact.ValueID,
				)
			}
			provenanceKnown[fact.ValueID] = fact.ID
		case FactProvenanceUnknown:
			if previous := provenanceKnown[fact.ValueID]; previous != "" {
				return fmt.Errorf(
					"plir verifier: %s fact %q provenance_known contradicts provenance_unknown fact %q for %q",
					fn.Name,
					previous,
					fact.ID,
					fact.ValueID,
				)
			}
			provenanceUnknown[fact.ValueID] = fact.ID
		}
	}
	for valueID, fact := range noAlias {
		if borrowed[valueID].Kind != FactBorrowedMut {
			return fmt.Errorf(
				"plir verifier: %s fact %q no_alias requires borrowed_mut fact for %q",
				fn.Name,
				fact.ID,
				valueID,
			)
		}
		if regionAlive[valueID] == "" {
			return fmt.Errorf(
				"plir verifier: %s fact %q no_alias requires region_alive fact for %q",
				fn.Name,
				fact.ID,
				valueID,
			)
		}
		if provenanceKnown[valueID] == "" {
			return fmt.Errorf(
				"plir verifier: %s fact %q no_alias requires provenance_known fact for %q",
				fn.Name,
				fact.ID,
				valueID,
			)
		}
	}
	for valueID, fact := range borrowed {
		if value := values[valueID]; value.Kind != ValueParam {
			if noEscape[valueID] == "" {
				return fmt.Errorf(
					"plir verifier: %s fact %q %s requires no_escape fact for %q",
					fn.Name,
					fact.ID,
					fact.Kind.String(),
					valueID,
				)
			}
		}
	}
	for _, value := range values {
		if value.Kind != ValueAllocIntent || value.Alloc == nil ||
			!isCopyAllocBuiltin(value.Alloc.Builtin) {
			continue
		}
		if owned[value.ID] == "" {
			return fmt.Errorf(
				"plir verifier: %s copy allocation intent requires owned fact for %q",
				fn.Name,
				value.ID,
			)
		}
	}
	return nil
}

func isCopyAllocBuiltin(name string) bool {
	return name == "core.string_copy" ||
		(strings.HasPrefix(name, "core.slice_copy_") && !strings.HasPrefix(name, "core.slice_copy_into_"))
}

func verifyFunctionSummaryCompleteness(fn Function, values map[string]Value) error {
	moduleBoundary := strings.TrimSpace(fn.Module) != "" || (fn.Summary != nil && fn.Summary.Public)
	if !moduleBoundary {
		return nil
	}
	if fn.Summary == nil {
		if functionHasSummaryObligation(fn, values) {
			return fmt.Errorf(
				("plir verifier: %s summary completeness: module-boundary memory-" +
					"bearing function requires FunctionSummary"),
				fn.Name,
			)
		}
		return nil
	}
	if err := verifySummaryParameterMetadata(fn, values); err != nil {
		return err
	}
	if err := verifySummaryReturnMetadata(fn, values); err != nil {
		return err
	}
	return nil
}

func functionHasSummaryObligation(fn Function, values map[string]Value) bool {
	for _, op := range fn.Ops {
		switch op.Kind {
		case OpReturn:
			for _, input := range op.Inputs {
				for _, value := range summaryValuesForPath(input, values) {
					if memoryBearingSummaryType(value.Type) {
						return true
					}
				}
			}
		case OpGlobalStore, OpActorSend, OpClosure:
			if operationHasMemoryBearingInput(op, values) {
				return true
			}
		case OpCall:
			if isTaskSummaryOperation(op) || isUnknownExternalSummaryOperation(op) {
				if operationHasMemoryBearingInput(op, values) {
					return true
				}
			}
		}
	}
	for _, fact := range fn.Facts {
		value, ok := values[fact.ValueID]
		if !ok || !memoryBearingSummaryType(value.Type) {
			continue
		}
		switch fact.Kind {
		case FactMoved, FactBorrowedMut, FactNoAlias, FactProvenanceUnknown:
			return true
		}
	}
	return false
}

func verifySummaryParameterMetadata(fn Function, values map[string]Value) error {
	for _, value := range values {
		if value.Kind != ValueParam || !memoryBearingSummaryType(value.Type) {
			continue
		}
		name := summaryParamNameForValue(value)
		index := indexOfString(fn.Summary.ParamNames, name)
		if index < 0 {
			return fmt.Errorf(
				"plir verifier: %s summary completeness: param_names missing memory-bearing parameter %q",
				fn.Name,
				name,
			)
		}
		if index >= len(fn.Summary.ParamTypes) ||
			strings.TrimSpace(fn.Summary.ParamTypes[index]) == "" {
			return fmt.Errorf(
				"plir verifier: %s summary completeness: param_types missing memory-bearing parameter %q",
				fn.Name,
				name,
			)
		}
		if (value.Borrow == BorrowMut || value.Borrow == BorrowMove) &&
			(index >= len(
				fn.Summary.ParamOwnership,
			) || strings.TrimSpace(
				fn.Summary.ParamOwnership[index],
			) == "") {
			return fmt.Errorf(
				("plir verifier: %s summary completeness: param_ownership missing " +
					"borrowed/moved memory-bearing parameter %q"),
				fn.Name,
				name,
			)
		}
	}
	return nil
}

func verifySummaryReturnMetadata(fn Function, values map[string]Value) error {
	for _, op := range fn.Ops {
		if op.Kind != OpReturn {
			continue
		}
		for _, input := range op.Inputs {
			for _, value := range summaryValuesForPath(input, values) {
				if !borrowedRegionSummaryType(value.Type) || !summaryReturnIsBorrowed(value) {
					continue
				}
				if fn.Summary.ReturnOwnership != "borrow" {
					return fmt.Errorf(
						("plir verifier: %s summary completeness: borrowed memory return " +
							"%q requires return_ownership borrow"),
						fn.Name,
						value.ID,
					)
				}
				if len(fn.Summary.ReturnRegionSummary) == 0 && !fn.Summary.ReturnRegionUnknown {
					return fmt.Errorf(
						("plir verifier: %s summary completeness: borrowed memory return " +
							"%q requires return_region_summary or return_region_unknown"),
						fn.Name,
						value.ID,
					)
				}
			}
		}
	}
	return nil
}

func operationHasMemoryBearingInput(op Operation, values map[string]Value) bool {
	for _, input := range op.Inputs {
		for _, value := range summaryValuesForPath(input, values) {
			if memoryBearingSummaryType(value.Type) {
				return true
			}
		}
	}
	return false
}

func summaryValuesForPath(path string, values map[string]Value) []Value {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if value, ok := values[path]; ok {
		return []Value{value}
	}
	owner := summaryOwnerPath(path)
	candidates := []string{owner}
	for _, prefix := range []string{"view:", "alloc_intent:", "local:", "param:"} {
		candidates = append(candidates, prefix+owner)
	}
	if owner != path {
		candidates = append(candidates, path)
		for _, prefix := range []string{"view:", "alloc_intent:", "local:", "param:"} {
			candidates = append(candidates, prefix+path)
		}
	}
	out := []Value(nil)
	seen := map[string]bool{}
	for _, candidate := range candidates {
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		if value, ok := values[candidate]; ok {
			out = append(out, value)
		}
	}
	return out
}

func summaryOwnerPath(path string) string {
	path = strings.TrimSpace(path)
	for _, prefix := range []string{"view:", "alloc_intent:", "local:", "param:"} {
		path = strings.TrimPrefix(path, prefix)
	}
	for strings.HasPrefix(path, "derived:") {
		path = strings.TrimPrefix(path, "derived:")
	}
	return path
}

func summaryParamNameForValue(value Value) string {
	name := summaryOwnerPath(value.ID)
	if name == "" {
		name = summaryOwnerPath(value.Provenance.Root)
	}
	if name == "" {
		name = summaryOwnerPath(value.Lifetime.Owner)
	}
	return name
}

func summaryReturnIsBorrowed(value Value) bool {
	if value.Borrow != BorrowNone {
		return true
	}
	if value.Provenance.Kind == ProvenanceParam && strings.TrimSpace(value.Provenance.Root) != "" {
		return true
	}
	return strings.Contains(summaryOwnerPath(value.Provenance.Root), ".")
}

func memoryBearingSummaryType(typeName string) bool {
	typeName = strings.TrimSpace(typeName)
	baseName := summaryTypeBaseName(typeName)
	return typeName == "ptr" ||
		typeName == "String" ||
		typeName == "island" ||
		strings.HasPrefix(typeName, "[]") ||
		strings.Contains(strings.ToLower(baseName), "resource")
}

func borrowedRegionSummaryType(typeName string) bool {
	typeName = strings.TrimSpace(typeName)
	baseName := summaryTypeBaseName(typeName)
	return typeName == "String" ||
		strings.HasPrefix(typeName, "[]") ||
		strings.Contains(strings.ToLower(baseName), "resource")
}

func summaryTypeBaseName(typeName string) string {
	typeName = strings.TrimSpace(typeName)
	if generic := strings.Index(typeName, "<"); generic >= 0 {
		typeName = typeName[:generic]
	}
	if dot := strings.LastIndex(typeName, "."); dot >= 0 {
		typeName = typeName[dot+1:]
	}
	return typeName
}

func isTaskSummaryOperation(op Operation) bool {
	note := strings.ToLower(strings.TrimSpace(op.Note))
	return strings.Contains(note, "task_spawn") || strings.Contains(note, "task_group")
}

func isUnknownExternalSummaryOperation(op Operation) bool {
	note := strings.ToLower(strings.TrimSpace(op.Note))
	return strings.Contains(note, "unknown external") ||
		strings.Contains(note, "external call") ||
		strings.HasPrefix(note, "ffi.") ||
		strings.Contains(note, " extern")
}

func indexOfString(values []string, want string) int {
	for index, value := range values {
		if value == want {
			return index
		}
	}
	return -1
}

func DominatorRows(fn Function) []DominatorRow {
	sets := computeDominators(fn)
	rows := make([]DominatorRow, 0, len(sets))
	for _, block := range fn.Blocks {
		doms := make([]string, 0, len(sets[block.ID]))
		for dom := range sets[block.ID] {
			doms = append(doms, dom)
		}
		sort.Strings(doms)
		rows = append(rows, DominatorRow{Block: block.ID, Dominators: doms})
	}
	return rows
}

func Dominates(fn Function, dominator string, block string) bool {
	if dominator == "" || block == "" {
		return false
	}
	sets := computeDominators(fn)
	return sets[block][dominator]
}

func computeDominators(fn Function) map[string]map[string]bool {
	blocks := map[string]BasicBlock{}
	all := map[string]bool{}
	entry := ""
	for _, block := range fn.Blocks {
		blocks[block.ID] = block
		all[block.ID] = true
		if block.Entry && entry == "" {
			entry = block.ID
		}
	}
	if entry == "" && len(fn.Blocks) > 0 {
		entry = fn.Blocks[0].ID
	}
	doms := map[string]map[string]bool{}
	for id := range blocks {
		doms[id] = cloneStringSet(all)
	}
	if entry != "" {
		doms[entry] = map[string]bool{entry: true}
	}
	changed := true
	for changed {
		changed = false
		for _, block := range fn.Blocks {
			if block.ID == entry {
				continue
			}
			next := cloneStringSet(all)
			if len(block.Preds) == 0 {
				next = map[string]bool{}
			}
			for i, pred := range block.Preds {
				predSet, ok := doms[pred]
				if !ok {
					continue
				}
				if i == 0 {
					next = cloneStringSet(predSet)
					continue
				}
				next = intersectStringSets(next, predSet)
			}
			next[block.ID] = true
			if !equalStringSets(doms[block.ID], next) {
				doms[block.ID] = next
				changed = true
			}
		}
	}
	return doms
}

func cloneStringSet(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func intersectStringSets(left map[string]bool, right map[string]bool) map[string]bool {
	out := map[string]bool{}
	for key := range left {
		if right[key] {
			out[key] = true
		}
	}
	return out
}

func equalStringSets(left map[string]bool, right map[string]bool) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}
