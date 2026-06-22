package validation

import (
	"fmt"
	"sort"
	"strings"

	"tetra_language/compiler/internal/ir"
)

type ownedStackKind int

const (
	ownedStackUnknown ownedStackKind = iota
	ownedStackValue
	ownedStackReleaseToken
	ownedStackConditionalErrorValue
)

type ownedStackItem struct {
	kind ownedStackKind
	id   string
	meta ownedReleaseMetadata
}

type ownedAllocationState struct {
	dropped     bool
	released    bool
	transferred bool
}

type ownedReleaseMetadata struct {
	layoutID string
	domain   ir.IROwnershipDomain
	release  ir.IRReleaseKind
}

type ownedValidationState struct {
	idx    int
	stack  []ownedStackItem
	locals []ownedStackItem
	allocs map[string]ownedAllocationState
}

func ValidateOwnedDropReleaseIR(fn ir.IRFunc) error {
	labels, err := ownedLabelMap(fn)
	if err != nil {
		return err
	}
	initial, err := initialOwnedValidationState(fn)
	if err != nil {
		return err
	}
	work := []ownedValidationState{initial}
	seen := map[string]bool{}
	for len(work) > 0 {
		cur := work[len(work)-1]
		work = work[:len(work)-1]
		if cur.idx > len(fn.Instrs) {
			return fmt.Errorf(
				"owned drop/release validation: %s instruction index %d out of range",
				fn.Name,
				cur.idx,
			)
		}
		key := ownedValidationStateKey(cur)
		if seen[key] {
			continue
		}
		seen[key] = true
		if cur.idx == len(fn.Instrs) {
			if err := validateOwnedReturnState(fn, cur, len(fn.Instrs)); err != nil {
				return err
			}
			continue
		}
		next, err := stepOwnedDropReleaseState(fn, cur, labels)
		if err != nil {
			return err
		}
		work = append(work, next...)
	}
	return nil
}

func initialOwnedValidationState(fn ir.IRFunc) (ownedValidationState, error) {
	state := ownedValidationState{
		idx:    0,
		locals: make([]ownedStackItem, fn.LocalSlots),
		allocs: map[string]ownedAllocationState{},
	}
	for i, param := range fn.OwnedParams {
		if param.Local < 0 || param.Local >= fn.LocalSlots {
			return ownedValidationState{}, fmt.Errorf(
				"owned drop/release validation: %s owned param %d local %d out of range",
				fn.Name,
				i,
				param.Local,
			)
		}
		meta, err := typedOwnedParamMetadata(param)
		if err != nil {
			return ownedValidationState{}, fmt.Errorf(
				"owned drop/release validation: %s owned param %d %w",
				fn.Name,
				i,
				err,
			)
		}
		if state.locals[param.Local].kind != ownedStackUnknown {
			return ownedValidationState{}, fmt.Errorf(
				"owned drop/release validation: %s owned param %d duplicates local %d",
				fn.Name,
				i,
				param.Local,
			)
		}
		id := fmt.Sprintf("%s:param:%d", fn.Name, param.Local)
		state.locals[param.Local] = ownedStackItem{kind: ownedStackValue, id: id, meta: meta}
		state.allocs[id] = ownedAllocationState{}
	}
	return state, nil
}

func stepOwnedDropReleaseState(
	fn ir.IRFunc,
	cur ownedValidationState,
	labels map[int]int,
) ([]ownedValidationState, error) {
	idx := cur.idx
	instr := fn.Instrs[idx]
	stack := append([]ownedStackItem(nil), cur.stack...)
	locals := append([]ownedStackItem(nil), cur.locals...)
	allocs := cloneOwnedAllocationStates(cur.allocs)

	switch instr.Kind {
	case ir.IRConstI32, ir.IRStrLit, ir.IRLoadGlobal, ir.IRCapIO, ir.IRCapMem, ir.IRSymAddr:
		_, push, _ := validationStackEffect(instr)
		stack = pushOwnedUnknown(stack, push)
	case ir.IRAllocBytes:
		var err error
		stack, err = popOwnedStack(fn.Name, idx, stack, 1)
		if err != nil {
			return nil, err
		}
		id := fmt.Sprintf("%s:alloc:%d", fn.Name, idx)
		if instr.Name != "" {
			id = fmt.Sprintf("%s:%s", id, instr.Name)
		}
		allocs[id] = ownedAllocationState{}
		stack = append(stack, ownedStackItem{kind: ownedStackValue, id: id})
	case ir.IRStoreLocal:
		popped, rest, err := popOwnedStackItemsAllowConditional(fn.Name, idx, stack, 1, true)
		if err != nil {
			return nil, err
		}
		if instr.Local < 0 || instr.Local >= len(locals) {
			return nil, fmt.Errorf(
				"owned drop/release validation: %s instruction %d local %d out of range",
				fn.Name,
				idx,
				instr.Local,
			)
		}
		locals[instr.Local] = popped[0]
		stack = rest
	case ir.IRStoreGlobal:
		popped, rest, err := popOwnedStackItems(fn.Name, idx, stack, 1)
		if err != nil {
			return nil, err
		}
		item := popped[0]
		switch item.kind {
		case ownedStackUnknown:
			// Unknown values can still be stored globally; they carry no owned obligation.
		case ownedStackValue:
			state := allocs[item.id]
			if state.transferred {
				return nil, fmt.Errorf(
					"owned drop/release validation: %s instruction %d use after transfer for %s",
					fn.Name,
					idx,
					item.id,
				)
			}
			if state.dropped {
				return nil, fmt.Errorf(
					"owned drop/release validation: %s instruction %d use after drop for %s",
					fn.Name,
					idx,
					item.id,
				)
			}
			state.transferred = true
			allocs[item.id] = state
		case ownedStackReleaseToken:
			return nil, fmt.Errorf(
				"owned drop/release validation: %s instruction %d transfer of release token for %s",
				fn.Name,
				idx,
				item.id,
			)
		}
		stack = rest
	case ir.IRLoadLocal:
		if instr.Local < 0 || instr.Local >= len(locals) {
			return nil, fmt.Errorf(
				"owned drop/release validation: %s instruction %d local %d out of range",
				fn.Name,
				idx,
				instr.Local,
			)
		}
		stack = append(stack, locals[instr.Local])
	case ir.IRDropOwned:
		meta, err := typedOwnedMetadata(instr, "drop")
		if err != nil {
			return nil, fmt.Errorf(
				"owned drop/release validation: %s instruction %d %w",
				fn.Name,
				idx,
				err,
			)
		}
		popped, rest, err := popOwnedStackItems(fn.Name, idx, stack, 1)
		if err != nil {
			return nil, err
		}
		item := popped[0]
		if item.kind != ownedStackValue {
			return nil, fmt.Errorf(
				"owned drop/release validation: %s instruction %d drop of non-owned value",
				fn.Name,
				idx,
			)
		}
		state := allocs[item.id]
		if state.transferred {
			return nil, fmt.Errorf(
				"owned drop/release validation: %s instruction %d use after transfer for %s",
				fn.Name,
				idx,
				item.id,
			)
		}
		if state.dropped {
			return nil, fmt.Errorf(
				"owned drop/release validation: %s instruction %d use after drop for %s",
				fn.Name,
				idx,
				item.id,
			)
		}
		state.dropped = true
		allocs[item.id] = state
		stack = append(rest, ownedStackItem{
			kind: ownedStackReleaseToken,
			id:   item.id,
			meta: meta,
		})
	case ir.IRReleaseAllocation:
		meta, err := typedOwnedMetadata(instr, "release")
		if err != nil {
			return nil, fmt.Errorf(
				"owned drop/release validation: %s instruction %d %w",
				fn.Name,
				idx,
				err,
			)
		}
		popped, rest, err := popOwnedStackItems(fn.Name, idx, stack, 1)
		if err != nil {
			return nil, err
		}
		item := popped[0]
		if item.kind != ownedStackReleaseToken {
			return nil, fmt.Errorf(
				"owned drop/release validation: %s instruction %d release without drop",
				fn.Name,
				idx,
			)
		}
		if item.meta != meta {
			return nil, fmt.Errorf(
				"owned drop/release validation: %s instruction %d release metadata does not match drop metadata",
				fn.Name,
				idx,
			)
		}
		state := allocs[item.id]
		if state.released {
			return nil, fmt.Errorf(
				"owned drop/release validation: %s instruction %d double release for %s",
				fn.Name,
				idx,
				item.id,
			)
		}
		state.released = true
		allocs[item.id] = state
		stack = rest
	case ir.IRCall:
		var err error
		stack, err = popOwnedStack(fn.Name, idx, stack, instr.ArgSlots)
		if err != nil {
			return nil, err
		}
		if instr.OwnsErrorSlot {
			if instr.RetSlots <= 0 {
				return nil, fmt.Errorf(
					"owned drop/release validation: %s instruction %d owned error metadata requires return slots",
					fn.Name,
					idx,
				)
			}
			if instr.OwnedErrorSlot < 0 || instr.OwnedErrorSlot >= instr.RetSlots-1 {
				return nil, fmt.Errorf(
					"owned drop/release validation: %s instruction %d owned error slot %d out of range for %d return slots",
					fn.Name,
					idx,
					instr.OwnedErrorSlot,
					instr.RetSlots,
				)
			}
			meta, err := typedOwnedMetadata(instr, "call error")
			if err != nil {
				return nil, fmt.Errorf(
					"owned drop/release validation: %s instruction %d %w",
					fn.Name,
					idx,
					err,
				)
			}
			id := fmt.Sprintf("%s:call-error:%d:%s", fn.Name, idx, instr.Name)
			allocs[id] = ownedAllocationState{}
			for slot := 0; slot < instr.RetSlots; slot++ {
				if slot == instr.OwnedErrorSlot {
					stack = append(stack, ownedStackItem{
						kind: ownedStackConditionalErrorValue,
						id:   id,
						meta: meta,
					})
					continue
				}
				stack = append(stack, ownedStackItem{kind: ownedStackUnknown})
			}
			break
		}
		if hasOwnedCallReturnMetadata(instr) {
			if instr.RetSlots <= 0 {
				return nil, fmt.Errorf(
					"owned drop/release validation: %s instruction %d owned call return metadata requires return slots",
					fn.Name,
					idx,
				)
			}
			if instr.OwnedReturnSlot < 0 || instr.OwnedReturnSlot >= instr.RetSlots {
				return nil, fmt.Errorf(
					"owned drop/release validation: %s instruction %d owned call return slot %d out of range for %d return slots",
					fn.Name,
					idx,
					instr.OwnedReturnSlot,
					instr.RetSlots,
				)
			}
			meta, err := typedOwnedMetadata(instr, "call return")
			if err != nil {
				return nil, fmt.Errorf(
					"owned drop/release validation: %s instruction %d %w",
					fn.Name,
					idx,
					err,
				)
			}
			id := fmt.Sprintf("%s:call:%d:%s", fn.Name, idx, instr.Name)
			allocs[id] = ownedAllocationState{}
			for slot := 0; slot < instr.RetSlots; slot++ {
				if slot == instr.OwnedReturnSlot {
					kind := ownedStackValue
					if instr.OwnedReturnConditional {
						kind = ownedStackConditionalErrorValue
					}
					stack = append(stack, ownedStackItem{kind: kind, id: id, meta: meta})
					continue
				}
				stack = append(stack, ownedStackItem{kind: ownedStackUnknown})
			}
			break
		}
		stack = pushOwnedUnknown(stack, instr.RetSlots)
	case ir.IRReturn:
		return nil, validateOwnedReturnState(
			fn,
			ownedValidationState{idx: idx, stack: stack, locals: locals, allocs: allocs},
			idx,
		)
	default:
		pop, push, ok := validationStackEffect(instr)
		if !ok {
			return nil, fmt.Errorf(
				"owned drop/release validation: %s instruction %d has unknown IR kind %d",
				fn.Name,
				idx,
				instr.Kind,
			)
		}
		var err error
		stack, err = popOwnedStack(fn.Name, idx, stack, pop)
		if err != nil {
			return nil, err
		}
		stack = pushOwnedUnknown(stack, push)
	}

	next := ownedValidationState{idx: idx + 1, stack: stack, locals: locals, allocs: allocs}
	switch instr.Kind {
	case ir.IRJmp:
		target, ok := labels[instr.Label]
		if !ok {
			return nil, fmt.Errorf(
				"owned drop/release validation: %s instruction %d unknown label %d",
				fn.Name,
				idx,
				instr.Label,
			)
		}
		next.idx = target
		return []ownedValidationState{next}, nil
	case ir.IRJmpIfZero:
		target, ok := labels[instr.Label]
		if !ok {
			return nil, fmt.Errorf(
				"owned drop/release validation: %s instruction %d unknown label %d",
				fn.Name,
				idx,
				instr.Label,
			)
		}
		branch := resolveConditionalErrorValues(cloneOwnedValidationState(next), false)
		next = resolveConditionalErrorValues(next, true)
		branch.idx = target
		return []ownedValidationState{next, branch}, nil
	default:
		return []ownedValidationState{next}, nil
	}
}

func ownedLabelMap(fn ir.IRFunc) (map[int]int, error) {
	labels := map[int]int{}
	for i, instr := range fn.Instrs {
		if instr.Kind != ir.IRLabel {
			continue
		}
		if previous, ok := labels[instr.Label]; ok {
			return nil, fmt.Errorf(
				"owned drop/release validation: %s duplicate label %d at instructions %d and %d",
				fn.Name,
				instr.Label,
				previous,
				i,
			)
		}
		labels[instr.Label] = i
	}
	return labels, nil
}

func validateOwnedReturnState(fn ir.IRFunc, state ownedValidationState, idx int) error {
	if token := firstOwnedReleaseToken(append(
		append([]ownedStackItem(nil), state.stack...),
		state.locals...,
	)); token != "" {
		return fmt.Errorf(
			"owned drop/release validation: %s instruction %d release token %s reaches return",
			fn.Name,
			idx,
			token,
		)
	}
	returned := returnedOwnedIDs(state.stack, fn.ReturnSlots)
	for id, alloc := range state.allocs {
		if alloc.transferred {
			if returned[id] {
				return fmt.Errorf(
					"owned drop/release validation: %s instruction %d use after transfer for %s",
					fn.Name,
					idx,
					id,
				)
			}
			continue
		}
		if alloc.dropped && !alloc.released {
			return fmt.Errorf(
				"owned drop/release validation: release token for %s was not released",
				id,
			)
		}
		if !alloc.dropped && !returned[id] {
			return fmt.Errorf(
				"owned drop/release validation: %s instruction %d allocation %s not dropped on return",
				fn.Name,
				idx,
				id,
			)
		}
	}
	return nil
}

func returnedOwnedIDs(stack []ownedStackItem, slots int) map[string]bool {
	returned := map[string]bool{}
	if slots <= 0 || len(stack) < slots {
		return returned
	}
	start := len(stack) - slots
	for _, item := range stack[start:] {
		if item.kind == ownedStackValue || item.kind == ownedStackConditionalErrorValue {
			returned[item.id] = true
		}
	}
	return returned
}

func cloneOwnedValidationState(state ownedValidationState) ownedValidationState {
	return ownedValidationState{
		idx:    state.idx,
		stack:  append([]ownedStackItem(nil), state.stack...),
		locals: append([]ownedStackItem(nil), state.locals...),
		allocs: cloneOwnedAllocationStates(state.allocs),
	}
}

func cloneOwnedAllocationStates(
	in map[string]ownedAllocationState,
) map[string]ownedAllocationState {
	out := make(map[string]ownedAllocationState, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func resolveConditionalErrorValues(state ownedValidationState, active bool) ownedValidationState {
	resolve := func(items []ownedStackItem) {
		for i := range items {
			if items[i].kind != ownedStackConditionalErrorValue {
				continue
			}
			if active {
				items[i].kind = ownedStackValue
				continue
			}
			delete(state.allocs, items[i].id)
			items[i] = ownedStackItem{}
		}
	}
	resolve(state.stack)
	resolve(state.locals)
	return state
}

func ownedValidationStateKey(state ownedValidationState) string {
	var b strings.Builder
	fmt.Fprintf(&b, "idx=%d|stack=", state.idx)
	writeOwnedStackKey(&b, state.stack)
	b.WriteString("|locals=")
	writeOwnedStackKey(&b, state.locals)
	b.WriteString("|allocs=")
	keys := make([]string, 0, len(state.allocs))
	for key := range state.allocs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := state.allocs[key]
		fmt.Fprintf(&b, "%s:%t:%t:%t;", key, value.dropped, value.released, value.transferred)
	}
	return b.String()
}

func writeOwnedStackKey(b *strings.Builder, stack []ownedStackItem) {
	for _, item := range stack {
		fmt.Fprintf(
			b,
			"%d:%s:%s:%s:%s;",
			item.kind,
			item.id,
			item.meta.layoutID,
			item.meta.domain,
			item.meta.release,
		)
	}
}

func typedOwnedMetadata(instr ir.IRInstr, phase string) (ownedReleaseMetadata, error) {
	if instr.LayoutID == "" || instr.OwnershipDomain == "" || instr.ReleaseKind == "" {
		return ownedReleaseMetadata{}, fmt.Errorf("missing typed %s metadata", phase)
	}
	return ownedReleaseMetadata{
		layoutID: instr.LayoutID,
		domain:   instr.OwnershipDomain,
		release:  instr.ReleaseKind,
	}, nil
}

func hasOwnedCallReturnMetadata(instr ir.IRInstr) bool {
	return instr.LayoutID != "" || instr.OwnershipDomain != "" || instr.ReleaseKind != ""
}

func typedOwnedParamMetadata(param ir.IROwnedParam) (ownedReleaseMetadata, error) {
	if param.LayoutID == "" || param.OwnershipDomain == "" || param.ReleaseKind == "" {
		return ownedReleaseMetadata{}, fmt.Errorf("missing typed param metadata")
	}
	return ownedReleaseMetadata{
		layoutID: param.LayoutID,
		domain:   param.OwnershipDomain,
		release:  param.ReleaseKind,
	}, nil
}

func popOwnedStack(
	fn string,
	idx int,
	stack []ownedStackItem,
	count int,
) ([]ownedStackItem, error) {
	_, rest, err := popOwnedStackItems(fn, idx, stack, count)
	return rest, err
}

func popOwnedStackItems(
	fn string,
	idx int,
	stack []ownedStackItem,
	count int,
) ([]ownedStackItem, []ownedStackItem, error) {
	return popOwnedStackItemsAllowConditional(fn, idx, stack, count, false)
}

func popOwnedStackItemsAllowConditional(
	fn string,
	idx int,
	stack []ownedStackItem,
	count int,
	allowConditional bool,
) ([]ownedStackItem, []ownedStackItem, error) {
	if count < 0 {
		return nil, nil, fmt.Errorf("owned drop/release validation: invalid pop count %d", count)
	}
	if len(stack) < count {
		return nil, nil, fmt.Errorf(
			"owned drop/release validation: %s instruction %d stack underflow",
			fn,
			idx,
		)
	}
	cut := len(stack) - count
	popped := append([]ownedStackItem(nil), stack[cut:]...)
	for _, item := range popped {
		if item.kind == ownedStackConditionalErrorValue && !allowConditional {
			return nil, nil, fmt.Errorf(
				"owned drop/release validation: %s instruction %d conditional owned error slot used before status check",
				fn,
				idx,
			)
		}
	}
	rest := append([]ownedStackItem(nil), stack[:cut]...)
	return popped, rest, nil
}

func pushOwnedUnknown(stack []ownedStackItem, count int) []ownedStackItem {
	for i := 0; i < count; i++ {
		stack = append(stack, ownedStackItem{})
	}
	return stack
}

func firstOwnedReleaseToken(stack []ownedStackItem) string {
	for _, item := range stack {
		if item.kind == ownedStackReleaseToken {
			return item.id
		}
	}
	return ""
}
