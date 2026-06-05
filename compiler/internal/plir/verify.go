package plir

import (
	"fmt"
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
				return fmt.Errorf("plir verifier: %s alloc intent %q has nil alloc metadata", fn.Name, value.ID)
			}
			if value.Alloc.ElementType == "" || value.Alloc.ElementSize <= 0 {
				return fmt.Errorf("plir verifier: %s alloc intent %q has invalid element metadata", fn.Name, value.ID)
			}
			if value.Alloc.LengthExpr == "" {
				return fmt.Errorf("plir verifier: %s alloc intent %q has empty length expression", fn.Name, value.ID)
			}
			if value.Alloc.ZeroGuardStatus == "" || value.Alloc.NegativeGuardStatus == "" || value.Alloc.OverflowGuardStatus == "" {
				return fmt.Errorf("plir verifier: %s alloc intent %q missing allocation length guard status", fn.Name, value.ID)
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
	rangeProofs := map[string]bool{}
	for _, fact := range fn.RangeFacts {
		if fact.Value == "" {
			return fmt.Errorf("plir verifier: %s range fact has empty value", fn.Name)
		}
		if fact.Source == "" {
			return fmt.Errorf("plir verifier: %s range fact for %q requires source", fn.Name, fact.Value)
		}
		if fact.Lower.Kind == BoundConst && fact.Upper.Kind == BoundConst && fact.Lower.Const > fact.Upper.Const {
			return fmt.Errorf("plir verifier: %s range fact for %q lower bound exceeds upper bound", fn.Name, fact.Value)
		}
		if fact.ProofID != "" {
			if _, ok := proofGuards[fact.ProofID]; !ok {
				return fmt.Errorf("plir verifier: %s range fact for %q references unknown proof id %q", fn.Name, fact.Value, fact.ProofID)
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
			return fmt.Errorf("plir verifier: %s fact %q references unknown value %q", fn.Name, fact.ID, fact.ValueID)
		}
		switch fact.Kind {
		case FactLenStable:
			if !hasValue {
				return fmt.Errorf("plir verifier: %s fact %q len_stable requires a value", fn.Name, fact.ID)
			}
			if value.Provenance.Kind == ProvenanceUnknown || value.Provenance.Kind == "" {
				return fmt.Errorf("plir verifier: %s fact %q len_stable requires known provenance for %q", fn.Name, fact.ID, fact.ValueID)
			}
		case FactIndexInRange:
			if !hasValue {
				return fmt.Errorf("plir verifier: %s fact %q index_in_range requires an index value", fn.Name, fact.ID)
			}
			if fact.Range == "" {
				return fmt.Errorf("plir verifier: %s fact %q index_in_range requires a range", fn.Name, fact.ID)
			}
			if fact.ProofID == "" {
				return fmt.Errorf("plir verifier: %s fact %q index_in_range requires proof id", fn.Name, fact.ID)
			}
			if _, ok := proofGuards[fact.ProofID]; !ok {
				return fmt.Errorf("plir verifier: %s fact %q index_in_range references unknown proof id %q", fn.Name, fact.ID, fact.ProofID)
			}
			rangeProofs[fact.ProofID] = true
		case FactProvenanceKnown:
			if !hasValue {
				return fmt.Errorf("plir verifier: %s fact %q provenance_known requires a value", fn.Name, fact.ID)
			}
			if value.Provenance.Kind == ProvenanceUnknown || value.Provenance.Kind == "" {
				return fmt.Errorf("plir verifier: %s fact %q provenance_known contradicts unknown provenance for %q", fn.Name, fact.ID, fact.ValueID)
			}
		case FactProvenanceUnknown:
			if !hasValue {
				return fmt.Errorf("plir verifier: %s fact %q provenance_unknown requires a value", fn.Name, fact.ID)
			}
			if value.Provenance.Kind != ProvenanceUnknown && value.Provenance.Kind != ProvenanceExternal {
				return fmt.Errorf("plir verifier: %s fact %q provenance_unknown contradicts known provenance for %q", fn.Name, fact.ID, fact.ValueID)
			}
		case FactNoAlias:
			if !hasValue {
				return fmt.Errorf("plir verifier: %s fact %q no_alias requires a value", fn.Name, fact.ID)
			}
			if value.Borrow != BorrowMut {
				return fmt.Errorf("plir verifier: %s fact %q no_alias requires mutable borrow for %q", fn.Name, fact.ID, fact.ValueID)
			}
			if value.Kind != ValueParam || value.Provenance.Kind != ProvenanceParam {
				return fmt.Errorf("plir verifier: %s fact %q no_alias requires parameter provenance for %q", fn.Name, fact.ID, fact.ValueID)
			}
			if value.Lifetime.Birth == "" || value.Lifetime.Death == "" || value.Lifetime.Owner == "" {
				return fmt.Errorf("plir verifier: %s fact %q no_alias requires bounded lifetime for %q", fn.Name, fact.ID, fact.ValueID)
			}
		case FactDerivedWindow:
			if !hasValue {
				return fmt.Errorf("plir verifier: %s fact %q derived_window requires a value", fn.Name, fact.ID)
			}
			if fact.Range == "" {
				return fmt.Errorf("plir verifier: %s fact %q derived_window requires a range", fn.Name, fact.ID)
			}
			if fact.Source == "" {
				return fmt.Errorf("plir verifier: %s fact %q derived_window requires source", fn.Name, fact.ID)
			}
		case FactMoved:
			if !hasValue {
				return fmt.Errorf("plir verifier: %s fact %q moved requires a value", fn.Name, fact.ID)
			}
			if strings.TrimSpace(fact.Source) == "" {
				return fmt.Errorf("plir verifier: %s fact %q moved requires source", fn.Name, fact.ID)
			}
			if strings.TrimSpace(fact.Reason) == "" {
				return fmt.Errorf("plir verifier: %s fact %q moved requires reason", fn.Name, fact.ID)
			}
			if value.Borrow == BorrowImm || value.Borrow == BorrowMut {
				return fmt.Errorf("plir verifier: %s fact %q moved contradicts borrowed value %q", fn.Name, fact.ID, fact.ValueID)
			}
		}
	}
	for _, use := range fn.ProofUses {
		if !rangeProofs[use.ProofID] {
			return fmt.Errorf("plir verifier: %s proof use %q has no explicit range proof fact", fn.Name, use.ProofID)
		}
	}
	if err := verifyFunctionFactConsistency(fn, values); err != nil {
		return err
	}
	return nil
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
				return nil, nil, fmt.Errorf("plir verifier: %s operation %q references unknown block %q", fn.Name, op.ID, op.Block)
			}
		}
		ops[op.ID] = op
	}
	for _, block := range fn.Blocks {
		for _, pred := range block.Preds {
			if _, ok := blocks[pred]; !ok {
				return nil, nil, fmt.Errorf("plir verifier: %s block %q references unknown predecessor %q", fn.Name, block.ID, pred)
			}
		}
		for _, succ := range block.Succs {
			if _, ok := blocks[succ]; !ok {
				return nil, nil, fmt.Errorf("plir verifier: %s block %q references unknown successor %q", fn.Name, block.ID, succ)
			}
		}
		for _, opID := range block.Ops {
			op, ok := ops[opID]
			if !ok {
				return nil, nil, fmt.Errorf("plir verifier: %s block %q references unknown operation %q", fn.Name, block.ID, opID)
			}
			if op.Block != "" && op.Block != block.ID {
				return nil, nil, fmt.Errorf("plir verifier: %s block %q contains operation %q assigned to block %q", fn.Name, block.ID, opID, op.Block)
			}
		}
	}
	return blocks, ops, nil
}

func verifyProofWiring(fn Function, blocks map[string]BasicBlock, ops map[string]Operation) (map[string]ProofGuard, error) {
	guards := map[string]ProofGuard{}
	for _, guard := range fn.ProofGuards {
		if guard.ID == "" {
			return nil, fmt.Errorf("plir verifier: %s has proof guard with empty id", fn.Name)
		}
		if _, exists := guards[guard.ID]; exists {
			return nil, fmt.Errorf("plir verifier: %s duplicate proof id %q", fn.Name, guard.ID)
		}
		if guard.Block == "" {
			return nil, fmt.Errorf("plir verifier: %s proof guard %q missing block", fn.Name, guard.ID)
		}
		if _, ok := blocks[guard.Block]; !ok {
			return nil, fmt.Errorf("plir verifier: %s proof guard %q references unknown block %q", fn.Name, guard.ID, guard.Block)
		}
		if guard.OpID != "" {
			if _, ok := ops[guard.OpID]; !ok {
				return nil, fmt.Errorf("plir verifier: %s proof guard %q references unknown operation %q", fn.Name, guard.ID, guard.OpID)
			}
		}
		guards[guard.ID] = guard
	}
	for _, use := range fn.ProofUses {
		guard, ok := guards[use.ProofID]
		if !ok {
			return nil, fmt.Errorf("plir verifier: %s proof use references unknown proof id %q", fn.Name, use.ProofID)
		}
		if use.Block == "" {
			return nil, fmt.Errorf("plir verifier: %s proof use %q missing block", fn.Name, use.ProofID)
		}
		if _, ok := blocks[use.Block]; !ok {
			return nil, fmt.Errorf("plir verifier: %s proof use %q references unknown block %q", fn.Name, use.ProofID, use.Block)
		}
		if use.OpID != "" {
			if _, ok := ops[use.OpID]; !ok {
				return nil, fmt.Errorf("plir verifier: %s proof use %q references unknown operation %q", fn.Name, use.ProofID, use.OpID)
			}
		}
		if !Dominates(fn, guard.Block, use.Block) {
			return nil, fmt.Errorf("plir verifier: %s proof id %q guard block %q does not dominate use block %q", fn.Name, use.ProofID, guard.Block, use.Block)
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
				return fmt.Errorf("plir verifier: %s fact %q owned contradicts borrowed value %q", fn.Name, fact.ID, fact.ValueID)
			}
		case FactBorrowedImm, FactBorrowedMut:
			borrowed[fact.ValueID] = fact
			if previous := moved[fact.ValueID]; previous != "" {
				return fmt.Errorf("plir verifier: %s fact %q %s contradicts moved fact %q for %q", fn.Name, fact.ID, fact.Kind.String(), previous, fact.ValueID)
			}
			if hasValue && value.Borrow == BorrowNone {
				return fmt.Errorf("plir verifier: %s fact %q %s contradicts non-borrowed value %q", fn.Name, fact.ID, fact.Kind.String(), fact.ValueID)
			}
		case FactNoEscape:
			noEscape[fact.ValueID] = fact.ID
			if previous := moved[fact.ValueID]; previous != "" {
				return fmt.Errorf("plir verifier: %s fact %q no_escape contradicts moved fact %q for %q", fn.Name, fact.ID, previous, fact.ValueID)
			}
			if hasValue && value.Escape != "" && value.Escape != EscapeNoEscape {
				return fmt.Errorf("plir verifier: %s fact %q no_escape contradicts escaping value %q", fn.Name, fact.ID, fact.ValueID)
			}
		case FactMoved:
			moved[fact.ValueID] = fact.ID
			if previous := borrowed[fact.ValueID]; previous.ID != "" {
				return fmt.Errorf("plir verifier: %s fact %q moved contradicts %s fact %q for %q", fn.Name, fact.ID, previous.Kind.String(), previous.ID, fact.ValueID)
			}
			if previous := noEscape[fact.ValueID]; previous != "" {
				return fmt.Errorf("plir verifier: %s fact %q moved contradicts no_escape fact %q for %q", fn.Name, fact.ID, previous, fact.ValueID)
			}
		case FactRegionAlive:
			regionAlive[fact.ValueID] = fact.ID
		case FactNoAlias:
			noAlias[fact.ValueID] = fact
		case FactNoHeapAllocation:
			if hasValue && (value.Kind == ValueAllocIntent || value.Alloc != nil) {
				return fmt.Errorf("plir verifier: %s fact %q no_heap_allocation contradicts allocation intent %q", fn.Name, fact.ID, fact.ValueID)
			}
		case FactProvenanceKnown:
			if previous := provenanceUnknown[fact.ValueID]; previous != "" {
				return fmt.Errorf("plir verifier: %s fact %q provenance_known contradicts provenance_unknown fact %q for %q", fn.Name, fact.ID, previous, fact.ValueID)
			}
			provenanceKnown[fact.ValueID] = fact.ID
		case FactProvenanceUnknown:
			if previous := provenanceKnown[fact.ValueID]; previous != "" {
				return fmt.Errorf("plir verifier: %s fact %q provenance_known contradicts provenance_unknown fact %q for %q", fn.Name, previous, fact.ID, fact.ValueID)
			}
			provenanceUnknown[fact.ValueID] = fact.ID
		}
	}
	for valueID, fact := range noAlias {
		if borrowed[valueID].Kind != FactBorrowedMut {
			return fmt.Errorf("plir verifier: %s fact %q no_alias requires borrowed_mut fact for %q", fn.Name, fact.ID, valueID)
		}
		if regionAlive[valueID] == "" {
			return fmt.Errorf("plir verifier: %s fact %q no_alias requires region_alive fact for %q", fn.Name, fact.ID, valueID)
		}
		if provenanceKnown[valueID] == "" {
			return fmt.Errorf("plir verifier: %s fact %q no_alias requires provenance_known fact for %q", fn.Name, fact.ID, valueID)
		}
	}
	for valueID, fact := range borrowed {
		if value := values[valueID]; value.Kind != ValueParam {
			if noEscape[valueID] == "" {
				return fmt.Errorf("plir verifier: %s fact %q %s requires no_escape fact for %q", fn.Name, fact.ID, fact.Kind.String(), valueID)
			}
		}
	}
	for _, value := range values {
		if value.Kind != ValueAllocIntent || value.Alloc == nil || !isCopyAllocBuiltin(value.Alloc.Builtin) {
			continue
		}
		if owned[value.ID] == "" {
			return fmt.Errorf("plir verifier: %s copy allocation intent requires owned fact for %q", fn.Name, value.ID)
		}
	}
	return nil
}

func isCopyAllocBuiltin(name string) bool {
	return name == "core.string_copy" ||
		(strings.HasPrefix(name, "core.slice_copy_") && !strings.HasPrefix(name, "core.slice_copy_into_"))
}
