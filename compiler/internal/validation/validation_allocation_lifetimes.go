package validation

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/lower"
)

func ValidateAllocationPlan(plan *allocplan.Plan) error {
	if err := allocplan.VerifyPlan(plan); err != nil {
		return fmt.Errorf("allocation validation: %w", err)
	}
	return nil
}

func ValidateAllocationLowering(plan *allocplan.Plan, prog *ir.IRProgram) error {
	return validateAllocationLowering(plan, prog, prog)
}

func ValidateAllocationLoweringWithSummaryProgram(plan *allocplan.Plan, prog *ir.IRProgram, summaryProg *ir.IRProgram) error {
	return validateAllocationLowering(plan, prog, summaryProg)
}

func validateAllocationLowering(plan *allocplan.Plan, prog *ir.IRProgram, summaryProg *ir.IRProgram) error {
	if err := ValidateAllocationPlan(plan); err != nil {
		return err
	}
	if prog == nil {
		return fmt.Errorf("allocation lowering validation: missing IR program")
	}
	if summaryProg == nil {
		summaryProg = prog
	}
	if prog.MainName == "" {
		for _, fn := range prog.Funcs {
			if err := lower.VerifyFunc(fn); err != nil {
				return fmt.Errorf("allocation lowering validation: IR invalid: %w", err)
			}
		}
	} else if err := lower.VerifyProgram(prog); err != nil {
		return fmt.Errorf("allocation lowering validation: IR invalid: %w", err)
	}
	if summaryProg != prog {
		if summaryProg.MainName == "" {
			for _, fn := range summaryProg.Funcs {
				if err := lower.VerifyFunc(fn); err != nil {
					return fmt.Errorf("allocation lowering validation: summary IR invalid: %w", err)
				}
			}
		} else if err := lower.VerifyProgram(summaryProg); err != nil {
			return fmt.Errorf("allocation lowering validation: summary IR invalid: %w", err)
		}
	}
	expected := map[string]map[string]allocplan.StorageClass{}
	stackAllocs := map[string]map[string]bool{}
	expectedRegion := map[string]map[string]ir.IRInstrKind{}
	regionAllocs := map[string]map[string]bool{}
	expectedIsland := map[string]map[string]islandExpectation{}
	islandAllocs := map[string]map[string]bool{}
	for _, fn := range plan.Functions {
		for _, alloc := range fn.Allocations {
			switch alloc.ActualLoweringStorage {
			case allocplan.StorageStack:
				if expected[fn.Name] == nil {
					expected[fn.Name] = map[string]allocplan.StorageClass{}
				}
				expected[fn.Name][alloc.ID] = alloc.ActualLoweringStorage
				if stackAllocs[fn.Name] == nil {
					stackAllocs[fn.Name] = map[string]bool{}
				}
				stackAllocs[fn.Name][alloc.ID] = true
			case allocplan.StorageEliminated:
				if alloc.LoweringStatus == "eliminated_no_backing_storage" {
					if expected[fn.Name] == nil {
						expected[fn.Name] = map[string]allocplan.StorageClass{}
					}
					expected[fn.Name][alloc.ID] = alloc.ActualLoweringStorage
				}
			case allocplan.StorageFunctionTempRegion:
				kind, ok := regionIRKind(alloc)
				if !ok {
					return fmt.Errorf("allocation lowering validation: %s allocation %q reports actual FunctionTempRegion for unsupported builtin %q", fn.Name, alloc.ID, alloc.Builtin)
				}
				if expectedRegion[fn.Name] == nil {
					expectedRegion[fn.Name] = map[string]ir.IRInstrKind{}
				}
				expectedRegion[fn.Name][alloc.ID] = kind
				if regionAllocs[fn.Name] == nil {
					regionAllocs[fn.Name] = map[string]bool{}
				}
				regionAllocs[fn.Name][alloc.ID] = true
			case allocplan.StorageExplicitIsland:
				kind, ok := explicitIslandIRKind(alloc)
				if !ok {
					return fmt.Errorf("allocation lowering validation: %s allocation %q reports actual ExplicitIsland for unsupported builtin %q", fn.Name, alloc.ID, alloc.Builtin)
				}
				if strings.TrimSpace(alloc.RegionID) == "" {
					return fmt.Errorf("allocation lowering validation: %s allocation %q reports actual ExplicitIsland without region id", fn.Name, alloc.ID)
				}
				if strings.TrimSpace(alloc.Lifetime) == "" {
					return fmt.Errorf("allocation lowering validation: %s allocation %q reports actual ExplicitIsland without lifetime", fn.Name, alloc.ID)
				}
				if expectedIsland[fn.Name] == nil {
					expectedIsland[fn.Name] = map[string]islandExpectation{}
				}
				expectedIsland[fn.Name][alloc.ID] = islandExpectation{
					kind:                 kind,
					regionID:             alloc.RegionID,
					lifetime:             alloc.Lifetime,
					handleParamSlotKnown: alloc.ExplicitIslandHandleParamSlotKnown,
					handleParamSlot:      alloc.ExplicitIslandHandleParamSlot,
				}
				if islandAllocs[fn.Name] == nil {
					islandAllocs[fn.Name] = map[string]bool{}
				}
				islandAllocs[fn.Name][alloc.ID] = true
			}
		}
	}
	seen := map[string]map[string]bool{}
	seenRegion := map[string]map[string]bool{}
	seenIsland := map[string]map[string]bool{}
	for _, fn := range prog.Funcs {
		for i, instr := range fn.Instrs {
			switch instr.Kind {
			case ir.IRStackSliceU8, ir.IRStackSliceU16, ir.IRStackSliceI32:
				want, ok := expected[fn.Name][instr.Name]
				if !ok {
					return fmt.Errorf("allocation lowering validation: %s instruction %d stack-lowers %q without matching allocation plan", fn.Name, i, instr.Name)
				}
				if want == allocplan.StorageEliminated && instr.ArgSlots != 0 {
					return fmt.Errorf("allocation lowering validation: %s eliminated allocation %q has %d stack backing slots", fn.Name, instr.Name, instr.ArgSlots)
				}
				if want == allocplan.StorageStack && instr.ArgSlots <= 0 {
					return fmt.Errorf("allocation lowering validation: %s stack allocation %q has no backing slots", fn.Name, instr.Name)
				}
				if seen[fn.Name] == nil {
					seen[fn.Name] = map[string]bool{}
				}
				seen[fn.Name][instr.Name] = true
			case ir.IRRegionMakeSliceU8, ir.IRRegionMakeSliceU16, ir.IRRegionMakeSliceI32:
				want, ok := expectedRegion[fn.Name][instr.Name]
				if !ok {
					return fmt.Errorf("allocation lowering validation: %s instruction %d emits %s for %q without matching function-temp region allocation plan", fn.Name, i, regionIRKindName(instr.Kind), instr.Name)
				}
				if want != instr.Kind {
					return fmt.Errorf("allocation lowering validation: %s allocation %q reports actual FunctionTempRegion for %s but IR emitted %s", fn.Name, instr.Name, regionIRKindName(want), regionIRKindName(instr.Kind))
				}
				if seenRegion[fn.Name] == nil {
					seenRegion[fn.Name] = map[string]bool{}
				}
				if seenRegion[fn.Name][instr.Name] {
					return fmt.Errorf("allocation lowering validation: %s emits duplicate function-temp region slice for %q", fn.Name, instr.Name)
				}
				seenRegion[fn.Name][instr.Name] = true
			case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32:
				if strings.TrimSpace(instr.Name) == "" {
					return fmt.Errorf("allocation lowering validation: %s instruction %d emits %s without allocation name; no matching IR island slice", fn.Name, i, islandIRKindName(instr.Kind))
				}
				want, ok := expectedIsland[fn.Name][instr.Name]
				if !ok {
					return fmt.Errorf("allocation lowering validation: %s instruction %d emits %s for %q without matching explicit island allocation plan", fn.Name, i, islandIRKindName(instr.Kind), instr.Name)
				}
				if want.kind != instr.Kind {
					return fmt.Errorf("allocation lowering validation: %s allocation %q reports actual ExplicitIsland for %s but IR emitted %s", fn.Name, instr.Name, islandIRKindName(want.kind), islandIRKindName(instr.Kind))
				}
				if seenIsland[fn.Name] == nil {
					seenIsland[fn.Name] = map[string]bool{}
				}
				if seenIsland[fn.Name][instr.Name] {
					return fmt.Errorf("allocation lowering validation: %s emits duplicate explicit island slice for %q", fn.Name, instr.Name)
				}
				seenIsland[fn.Name][instr.Name] = true
			}
		}
	}
	stackCallSummaries := buildStackCallSummaries(summaryProg)
	if err := validateStackAllocationsDoNotEscape(prog, stackAllocs, stackCallSummaries); err != nil {
		return err
	}
	if err := validateStackAllocationsDoNotEscape(prog, regionAllocs, stackCallSummaries); err != nil {
		return err
	}
	if err := validateFunctionTempRegionResets(prog, regionAllocs); err != nil {
		return err
	}
	if err := validateExplicitIslandLifetimes(prog, summaryProg, expectedIsland); err != nil {
		return err
	}
	for fn, allocs := range expected {
		for id, storage := range allocs {
			if !seen[fn][id] {
				return fmt.Errorf("allocation lowering validation: %s allocation %q reports actual %s but no matching IR stack slice was emitted", fn, id, storage)
			}
		}
	}
	for fn, allocs := range expectedRegion {
		for id, kind := range allocs {
			if !seenRegion[fn][id] {
				return fmt.Errorf("allocation lowering validation: %s allocation %q reports actual FunctionTempRegion for %s but no matching IR function-temp region slice was emitted", fn, id, regionIRKindName(kind))
			}
		}
	}
	for fn, allocs := range expectedIsland {
		for id, want := range allocs {
			if !seenIsland[fn][id] {
				return fmt.Errorf("allocation lowering validation: %s allocation %q reports actual ExplicitIsland for %s but no matching IR island slice was emitted", fn, id, islandIRKindName(want.kind))
			}
		}
	}
	return nil
}

type stackEscapeState struct {
	idx    int
	stack  []string
	locals []string
}

type stackCallSummary struct {
	retTags []string
}

type functionTempRegionResetState struct {
	idx     int
	entered bool
	active  bool
}

type islandExpectation struct {
	kind                 ir.IRInstrKind
	regionID             string
	lifetime             string
	handleParamSlotKnown bool
	handleParamSlot      int
}

type islandLifetimeState struct {
	idx    int
	stack  []string
	locals []string
	freed  map[string]bool
}

type islandReturnSummary struct {
	retTags []string
}

func validateStackAllocationsDoNotEscape(prog *ir.IRProgram, stackAllocs map[string]map[string]bool, callSummaries map[string]stackCallSummary) error {
	for _, fn := range prog.Funcs {
		tracked := stackAllocs[fn.Name]
		if len(tracked) == 0 {
			continue
		}
		if err := validateFunctionStackAllocationsDoNotEscape(fn, tracked, callSummaries); err != nil {
			return err
		}
	}
	return nil
}

func validateFunctionTempRegionResets(prog *ir.IRProgram, regionAllocs map[string]map[string]bool) error {
	for _, fn := range prog.Funcs {
		tracked := regionAllocs[fn.Name]
		if len(tracked) == 0 {
			continue
		}
		if err := validateFunctionTempRegionResetsInFunc(fn, tracked); err != nil {
			return err
		}
	}
	return nil
}

func validateFunctionTempRegionResetsInFunc(fn ir.IRFunc, tracked map[string]bool) error {
	labels := map[int]int{}
	for i, instr := range fn.Instrs {
		if instr.Kind == ir.IRLabel {
			labels[instr.Label] = i
		}
	}
	work := []functionTempRegionResetState{{idx: 0}}
	seen := map[functionTempRegionResetState]bool{}
	for len(work) > 0 {
		cur := work[len(work)-1]
		work = work[:len(work)-1]
		if cur.idx < 0 || cur.idx >= len(fn.Instrs) {
			if cur.active {
				return fmt.Errorf("allocation lowering validation: %s function-temp region reset does not dominate function exit", fn.Name)
			}
			continue
		}
		if seen[cur] {
			continue
		}
		seen[cur] = true
		next, err := stepFunctionTempRegionResetState(fn, cur, labels, tracked)
		if err != nil {
			return err
		}
		work = append(work, next...)
	}
	return nil
}

func stepFunctionTempRegionResetState(fn ir.IRFunc, cur functionTempRegionResetState, labels map[int]int, tracked map[string]bool) ([]functionTempRegionResetState, error) {
	instr := fn.Instrs[cur.idx]
	entered := cur.entered
	active := cur.active
	switch instr.Kind {
	case ir.IRRegionEnter:
		entered = true
	case ir.IRRegionMakeSliceU8, ir.IRRegionMakeSliceU16, ir.IRRegionMakeSliceI32:
		if tracked[instr.Name] {
			if !entered {
				return nil, fmt.Errorf("allocation lowering validation: %s instruction %d function-temp region enter does not dominate make for %q", fn.Name, cur.idx, instr.Name)
			}
			active = true
		}
	case ir.IRRegionReset:
		active = false
	case ir.IRReturn:
		if active {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d function-temp region reset does not dominate return", fn.Name, cur.idx)
		}
		return nil, nil
	}

	next := functionTempRegionResetState{idx: cur.idx + 1, entered: entered, active: active}
	switch instr.Kind {
	case ir.IRJmp:
		labelIdx, ok := labels[instr.Label]
		if !ok {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d jumps to unknown label %d", fn.Name, cur.idx, instr.Label)
		}
		next.idx = labelIdx
		return []functionTempRegionResetState{next}, nil
	case ir.IRJmpIfZero:
		labelIdx, ok := labels[instr.Label]
		if !ok {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d jumps to unknown label %d", fn.Name, cur.idx, instr.Label)
		}
		branch := next
		branch.idx = labelIdx
		return []functionTempRegionResetState{next, branch}, nil
	default:
		return []functionTempRegionResetState{next}, nil
	}
}

func validateExplicitIslandLifetimes(prog *ir.IRProgram, summaryProg *ir.IRProgram, expected map[string]map[string]islandExpectation) error {
	summaries := inferIslandReturnSummaries(summaryProg)
	for _, fn := range prog.Funcs {
		tracked := expected[fn.Name]
		if len(tracked) == 0 {
			continue
		}
		locals, err := initialExplicitIslandLifetimeLocals(fn, tracked)
		if err != nil {
			return err
		}
		labels := map[int]int{}
		for i, instr := range fn.Instrs {
			if instr.Kind == ir.IRLabel {
				labels[instr.Label] = i
			}
		}
		work := []islandLifetimeState{{
			idx:    0,
			locals: locals,
			freed:  map[string]bool{},
		}}
		seen := map[string]bool{}
		for len(work) > 0 {
			cur := work[len(work)-1]
			work = work[:len(work)-1]
			if cur.idx < 0 || cur.idx >= len(fn.Instrs) {
				continue
			}
			key := islandLifetimeStateKey(cur)
			if seen[key] {
				continue
			}
			seen[key] = true
			next, err := stepExplicitIslandLifetimeState(fn, cur, labels, tracked, summaries)
			if err != nil {
				return err
			}
			work = append(work, next...)
		}
	}
	return nil
}

func initialExplicitIslandLifetimeLocals(fn ir.IRFunc, tracked map[string]islandExpectation) ([]string, error) {
	locals := make([]string, fn.LocalSlots)
	for alloc, expectation := range tracked {
		if !expectation.handleParamSlotKnown {
			continue
		}
		slot := expectation.handleParamSlot
		if slot < 0 || slot >= fn.ParamSlots || slot >= len(locals) {
			return nil, fmt.Errorf("allocation lowering validation: %s allocation %q explicit island handle parameter slot %d is outside params=%d locals=%d", fn.Name, alloc, slot, fn.ParamSlots, fn.LocalSlots)
		}
		locals[slot] = islandParamTag(slot)
	}
	return locals, nil
}

func stepExplicitIslandLifetimeState(fn ir.IRFunc, cur islandLifetimeState, labels map[int]int, expected map[string]islandExpectation, summaries map[string]islandReturnSummary) ([]islandLifetimeState, error) {
	instr := fn.Instrs[cur.idx]
	stack := append([]string(nil), cur.stack...)
	locals := append([]string(nil), cur.locals...)
	freed := cloneBoolMap(cur.freed)
	pop, push, ok := validationStackEffect(instr)
	if !ok {
		return nil, fmt.Errorf("allocation lowering validation: %s instruction %d has unknown IR kind %d", fn.Name, cur.idx, instr.Kind)
	}

	switch instr.Kind {
	case ir.IRReturn:
		if name := firstEscapingTrackedExplicitIslandSlice(stack, expected); name != "" {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island allocation %q escapes via return", fn.Name, cur.idx, name)
		}
		if tag := firstFreedIslandUse(stack, freed); tag != "" {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island use after free via return of %s", fn.Name, cur.idx, tag)
		}
		return nil, nil
	case ir.IRIslandNew:
		_, rest := popStackTags(stack, pop)
		tag := explicitIslandHandleTag(fn.Name, cur.idx)
		delete(freed, tag)
		stack = append(rest, tag)
	case ir.IRIslandFree:
		popped, rest := popStackTags(stack, pop)
		tag := firstExplicitIslandHandleTag(popped)
		if tag == "" {
			stack = rest
			break
		}
		if freed[tag] {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island double free for %s", fn.Name, cur.idx, tag)
		}
		freed[tag] = true
		stack = rest
	case ir.IRIslandReset:
		popped, rest := popStackTags(stack, pop)
		tag := firstExplicitIslandHandleTag(popped)
		if tag == "" {
			stack = pushEmptyTags(rest, push)
			break
		}
		if freed[tag] {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island reset after free for %s", fn.Name, cur.idx, tag)
		}
		if live := firstLiveIslandSliceForHandle(stack, locals, tag); live != "" {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island reset while live slice %s still references %s", fn.Name, cur.idx, live, tag)
		}
		freed[tag] = true
		stack = append(rest, explicitIslandHandleTag(fn.Name, cur.idx))
	case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32:
		popped, rest := popStackTags(stack, pop)
		if _, ok := expected[instr.Name]; ok {
			if tag := firstFreedIslandUse(popped, freed); tag != "" {
				return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island allocation %q use after free via operands of %s", fn.Name, cur.idx, instr.Name, tag)
			}
			tag := explicitIslandMakeHandleOperandTag(popped)
			if tag == "" {
				return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island allocation %q has no active island handle operand", fn.Name, cur.idx, instr.Name)
			}
			if freed[tag] {
				return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island allocation %q use after free for %s", fn.Name, cur.idx, instr.Name, tag)
			}
			stack = append(rest, explicitIslandSliceTag(instr.Name, tag), "")
		} else {
			stack = pushEmptyTags(rest, push)
		}
	case ir.IRStoreLocal:
		popped, rest := popStackTags(stack, pop)
		stack = rest
		if instr.Local >= 0 && instr.Local < len(locals) {
			locals[instr.Local] = firstExplicitIslandValueTag(popped)
		}
	case ir.IRLoadLocal:
		if instr.Local >= 0 && instr.Local < len(locals) {
			stack = append(stack, locals[instr.Local])
		} else {
			stack = append(stack, "")
		}
	case ir.IRSliceWindow, ir.IRSlicePrefix, ir.IRSliceSuffix:
		popped, rest := popStackTags(stack, pop)
		if tag := firstFreedIslandUse(popped, freed); tag != "" {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island use after free via slice view of %s", fn.Name, cur.idx, tag)
		}
		stack = append(rest, firstExplicitIslandValueTag(popped), "")
	case ir.IRCall:
		popped, rest := popStackTags(stack, pop)
		if tag := firstFreedIslandUse(popped, freed); tag != "" {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island use after free via %s", fn.Name, cur.idx, tag)
		}
		stack = append(rest, islandCallReturnTags(instr, popped, summaries)...)
	case ir.IRIndexLoadI32, ir.IRIndexLoadU8, ir.IRIndexLoadU16,
		ir.IRIndexLoadI32Unchecked, ir.IRIndexLoadU8Unchecked, ir.IRIndexLoadU16Unchecked,
		ir.IRIndexStoreI32, ir.IRIndexStoreU8, ir.IRIndexStoreU16,
		ir.IRStoreGlobal,
		ir.IRMemReadI32, ir.IRMemReadU8, ir.IRMemReadPtr,
		ir.IRMemWriteI32, ir.IRMemWriteU8, ir.IRMemWritePtr, ir.IRMemWriteArchPtr,
		ir.IRMemReadI32Offset, ir.IRMemReadU8Offset, ir.IRMemReadPtrOffset,
		ir.IRMemWriteI32Offset, ir.IRMemWriteU8Offset, ir.IRMemWritePtrOffset, ir.IRMemWriteArchPtrOffset:
		popped, rest := popStackTags(stack, pop)
		if tag := firstFreedIslandUse(popped, freed); tag != "" {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island use after free via %s", fn.Name, cur.idx, tag)
		}
		stack = pushEmptyTags(rest, push)
	default:
		popped, rest := popStackTags(stack, pop)
		if tag := firstFreedIslandUse(popped, freed); tag != "" {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d explicit island use after free via %s", fn.Name, cur.idx, tag)
		}
		stack = pushEmptyTags(rest, push)
	}

	next := islandLifetimeState{idx: cur.idx + 1, stack: stack, locals: locals, freed: freed}
	switch instr.Kind {
	case ir.IRJmp:
		next.idx = labels[instr.Label]
		return []islandLifetimeState{next}, nil
	case ir.IRJmpIfZero:
		branch := cloneIslandLifetimeState(next)
		branch.idx = labels[instr.Label]
		return []islandLifetimeState{next, branch}, nil
	default:
		return []islandLifetimeState{next}, nil
	}
}

func inferIslandReturnSummaries(prog *ir.IRProgram) map[string]islandReturnSummary {
	summaries := map[string]islandReturnSummary{}
	for _, fn := range prog.Funcs {
		summary, ok := inferIslandReturnSummary(fn)
		if ok {
			summaries[fn.Name] = summary
		}
	}
	return summaries
}

func inferIslandReturnSummary(fn ir.IRFunc) (islandReturnSummary, bool) {
	if fn.ParamSlots <= 0 || fn.ReturnSlots <= 0 || fn.LocalSlots <= 0 {
		return islandReturnSummary{}, false
	}
	labels := map[int]int{}
	for i, instr := range fn.Instrs {
		if instr.Kind == ir.IRLabel {
			labels[instr.Label] = i
		}
	}
	locals := make([]string, fn.LocalSlots)
	for i := 0; i < fn.ParamSlots && i < len(locals); i++ {
		locals[i] = islandParamTag(i)
	}
	work := []stackEscapeState{{idx: 0, locals: locals}}
	seen := map[string]bool{}
	var merged []string
	for len(work) > 0 {
		cur := work[len(work)-1]
		work = work[:len(work)-1]
		if cur.idx < 0 || cur.idx >= len(fn.Instrs) {
			continue
		}
		key := stackEscapeStateKey(cur)
		if seen[key] {
			continue
		}
		seen[key] = true
		next, ret, ok := stepIslandReturnSummaryState(fn, cur, labels)
		if !ok {
			return islandReturnSummary{}, false
		}
		if ret != nil {
			merged = mergeIslandReturnSources(merged, ret)
			continue
		}
		work = append(work, next...)
	}
	if merged == nil {
		return islandReturnSummary{}, false
	}
	return islandReturnSummary{retTags: merged}, true
}

func stepIslandReturnSummaryState(fn ir.IRFunc, cur stackEscapeState, labels map[int]int) ([]stackEscapeState, []string, bool) {
	instr := fn.Instrs[cur.idx]
	stack := append([]string(nil), cur.stack...)
	locals := append([]string(nil), cur.locals...)
	pop, push, ok := validationStackEffect(instr)
	if !ok {
		return nil, nil, false
	}

	switch instr.Kind {
	case ir.IRReturn:
		return nil, islandReturnTags(stack, fn.ReturnSlots), true
	case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32:
		popped, rest := popStackTags(stack, pop)
		if tag := explicitIslandMakeHandleOperandTag(popped); strings.HasPrefix(tag, "island-param:") {
			stack = append(rest, explicitIslandSliceTag(instr.Name, tag), "")
		} else {
			stack = pushEmptyTags(rest, push)
		}
	case ir.IRStoreLocal:
		popped, rest := popStackTags(stack, pop)
		stack = rest
		if instr.Local >= 0 && instr.Local < len(locals) {
			locals[instr.Local] = firstIslandParamTag(popped)
		}
	case ir.IRLoadLocal:
		if instr.Local >= 0 && instr.Local < len(locals) {
			stack = append(stack, locals[instr.Local])
		} else {
			stack = append(stack, "")
		}
	default:
		_, rest := popStackTags(stack, pop)
		stack = pushEmptyTags(rest, push)
	}

	next := stackEscapeState{idx: cur.idx + 1, stack: stack, locals: locals}
	switch instr.Kind {
	case ir.IRJmp:
		next.idx = labels[instr.Label]
		return []stackEscapeState{next}, nil, true
	case ir.IRJmpIfZero:
		branch := cloneStackEscapeState(next)
		branch.idx = labels[instr.Label]
		return []stackEscapeState{next, branch}, nil, true
	default:
		return []stackEscapeState{next}, nil, true
	}
}

func islandReturnTags(stack []string, slots int) []string {
	tags := make([]string, slots)
	if slots <= 0 || len(stack) < slots {
		return tags
	}
	start := len(stack) - slots
	for i := 0; i < slots; i++ {
		if islandParamSlotFromValueTag(stack[start+i]) >= 0 {
			tags[i] = stack[start+i]
		}
	}
	return tags
}

func mergeIslandReturnSources(merged []string, ret []string) []string {
	if merged == nil {
		return append([]string(nil), ret...)
	}
	for i := range merged {
		if i >= len(ret) || merged[i] != ret[i] {
			merged[i] = ""
		}
	}
	return merged
}

func islandCallReturnTags(instr ir.IRInstr, popped []string, summaries map[string]islandReturnSummary) []string {
	tags := make([]string, instr.RetSlots)
	summary, ok := summaries[instr.Name]
	if !ok {
		return tags
	}
	for i := range tags {
		if i >= len(summary.retTags) {
			continue
		}
		tags[i] = substituteIslandSummaryTag(summary.retTags[i], instr.ArgSlots, popped)
	}
	return tags
}

func substituteIslandSummaryTag(tag string, argSlots int, popped []string) string {
	if tag == "" {
		return ""
	}
	if slot := islandParamSlot(tag); slot >= 0 {
		return islandCallArgTag(slot, argSlots, popped)
	}
	name, handleTag, ok := islandSliceTagParts(tag)
	if !ok {
		return ""
	}
	slot := islandParamSlot(handleTag)
	if slot < 0 {
		return ""
	}
	replacement := islandCallArgTag(slot, argSlots, popped)
	if replacement == "" {
		return ""
	}
	return explicitIslandSliceTag(name, replacement)
}

func islandCallArgTag(slot int, argSlots int, popped []string) string {
	if slot < 0 || slot >= argSlots || slot >= len(popped) {
		return ""
	}
	return popped[len(popped)-1-slot]
}

func islandParamTag(slot int) string {
	return fmt.Sprintf("island-param:%d", slot)
}

func islandParamSlot(tag string) int {
	const prefix = "island-param:"
	if !strings.HasPrefix(tag, prefix) {
		return -1
	}
	slot, err := strconv.Atoi(strings.TrimPrefix(tag, prefix))
	if err != nil {
		return -1
	}
	return slot
}

func firstIslandParamTag(tags []string) string {
	for _, tag := range tags {
		if islandParamSlotFromValueTag(tag) >= 0 {
			return tag
		}
	}
	return ""
}

func islandParamSlotFromValueTag(tag string) int {
	if slot := islandParamSlot(tag); slot >= 0 {
		return slot
	}
	_, handleTag, ok := islandSliceTagParts(tag)
	if !ok {
		return -1
	}
	return islandParamSlot(handleTag)
}

func explicitIslandHandleTag(function string, idx int) string {
	return fmt.Sprintf("island:%s:%d", function, idx)
}

func explicitIslandSliceTag(name string, islandTag string) string {
	return "island-slice:" + name + "@" + islandTag
}

func firstExplicitIslandHandleTag(tags []string) string {
	for _, tag := range tags {
		if strings.HasPrefix(tag, "island:") || strings.HasPrefix(tag, "island-param:") {
			return tag
		}
	}
	return ""
}

func explicitIslandMakeHandleOperandTag(popped []string) string {
	if len(popped) < 2 {
		return ""
	}
	tag := popped[1]
	if strings.HasPrefix(tag, "island:") || strings.HasPrefix(tag, "island-param:") {
		return tag
	}
	return ""
}

func firstExplicitIslandValueTag(tags []string) string {
	for _, tag := range tags {
		if strings.HasPrefix(tag, "island:") || strings.HasPrefix(tag, "island-param:") || strings.HasPrefix(tag, "island-slice:") {
			return tag
		}
	}
	return ""
}

func firstFreedIslandUse(tags []string, freed map[string]bool) string {
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		if strings.HasPrefix(tag, "island:") || strings.HasPrefix(tag, "island-param:") {
			if freed[tag] {
				return tag
			}
			continue
		}
		if _, islandTag, ok := islandSliceTagParts(tag); ok {
			if freed[islandTag] {
				return tag
			}
		}
	}
	return ""
}

func firstEscapingTrackedExplicitIslandSlice(tags []string, expected map[string]islandExpectation) string {
	for _, tag := range tags {
		name, handleTag, ok := islandSliceTagParts(tag)
		if !ok {
			continue
		}
		expectation, ok := expected[name]
		if !ok {
			continue
		}
		if expectation.handleParamSlotKnown && islandParamSlot(handleTag) == expectation.handleParamSlot {
			continue
		}
		return name
	}
	return ""
}

func firstLiveIslandSliceForHandle(stack []string, locals []string, handleTag string) string {
	for _, tags := range [][]string{stack, locals} {
		for _, tag := range tags {
			name, tagHandle, ok := islandSliceTagParts(tag)
			if ok && tagHandle == handleTag {
				return name
			}
		}
	}
	return ""
}

func islandSliceTagParts(tag string) (name string, handleTag string, ok bool) {
	const prefix = "island-slice:"
	if !strings.HasPrefix(tag, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(tag, prefix)
	at := strings.LastIndex(rest, "@")
	if at < 0 || at == 0 || at+1 >= len(rest) {
		return "", "", false
	}
	return rest[:at], rest[at+1:], true
}

func cloneBoolMap(in map[string]bool) map[string]bool {
	out := map[string]bool{}
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneIslandLifetimeState(state islandLifetimeState) islandLifetimeState {
	return islandLifetimeState{
		idx:    state.idx,
		stack:  append([]string(nil), state.stack...),
		locals: append([]string(nil), state.locals...),
		freed:  cloneBoolMap(state.freed),
	}
}

func islandLifetimeStateKey(state islandLifetimeState) string {
	var b strings.Builder
	b.WriteString(strconv.Itoa(state.idx))
	b.WriteByte('|')
	for _, tag := range state.stack {
		b.WriteString(tag)
		b.WriteByte(',')
	}
	b.WriteByte('|')
	for i, tag := range state.locals {
		if tag == "" {
			continue
		}
		b.WriteString(strconv.Itoa(i))
		b.WriteByte('=')
		b.WriteString(tag)
		b.WriteByte(',')
	}
	b.WriteByte('|')
	freed := make([]string, 0, len(state.freed))
	for tag, value := range state.freed {
		if value {
			freed = append(freed, tag)
		}
	}
	sort.Strings(freed)
	for _, tag := range freed {
		b.WriteString(tag)
		b.WriteByte(',')
	}
	return b.String()
}

func buildStackCallSummaries(prog *ir.IRProgram) map[string]stackCallSummary {
	if prog == nil {
		return nil
	}
	summaries := map[string]stackCallSummary{}
	for _, fn := range prog.Funcs {
		summary, ok := summarizeStackCall(fn)
		if ok {
			summaries[fn.Name] = summary
		}
	}
	return summaries
}

func summarizeStackCall(fn ir.IRFunc) (stackCallSummary, bool) {
	labels := map[int]int{}
	for i, instr := range fn.Instrs {
		if instr.Kind == ir.IRLabel {
			labels[instr.Label] = i
		}
	}
	locals := make([]string, fn.LocalSlots)
	for i := 0; i < fn.ParamSlots && i < len(locals); i++ {
		locals[i] = stackParamTag(i)
	}
	work := []stackEscapeState{{idx: 0, locals: locals}}
	seen := map[string]bool{}
	var merged []string
	for len(work) > 0 {
		cur := work[len(work)-1]
		work = work[:len(work)-1]
		if cur.idx < 0 || cur.idx >= len(fn.Instrs) {
			continue
		}
		key := stackEscapeStateKey(cur)
		if seen[key] {
			continue
		}
		seen[key] = true
		next, ret, ok := stepStackCallSummaryState(fn, cur, labels)
		if !ok {
			return stackCallSummary{}, false
		}
		if ret != nil {
			merged = mergeStackReturnTags(merged, ret)
			continue
		}
		work = append(work, next...)
	}
	if merged == nil {
		return stackCallSummary{}, false
	}
	return stackCallSummary{retTags: merged}, true
}

func stepStackCallSummaryState(fn ir.IRFunc, cur stackEscapeState, labels map[int]int) ([]stackEscapeState, []string, bool) {
	instr := fn.Instrs[cur.idx]
	stack := append([]string(nil), cur.stack...)
	locals := append([]string(nil), cur.locals...)
	pop, push, ok := validationStackEffect(instr)
	if !ok {
		return nil, nil, false
	}

	switch instr.Kind {
	case ir.IRReturn:
		return nil, stackReturnTags(stack, fn.ReturnSlots), true
	case ir.IRCall, ir.IRStoreGlobal:
		popped, rest := popStackTags(stack, pop)
		if firstStackParamTag(popped) != "" {
			return nil, nil, false
		}
		stack = pushEmptyTags(rest, push)
	case ir.IRStoreLocal:
		popped, rest := popStackTags(stack, pop)
		stack = rest
		if instr.Local >= 0 && instr.Local < len(locals) {
			locals[instr.Local] = firstStackParamTag(popped)
		}
	case ir.IRLoadLocal:
		if instr.Local >= 0 && instr.Local < len(locals) {
			stack = append(stack, locals[instr.Local])
		} else {
			stack = append(stack, "")
		}
	case ir.IRRawSliceFromParts, ir.IRSliceWindow, ir.IRSlicePrefix, ir.IRSliceSuffix:
		popped, rest := popStackTags(stack, pop)
		stack = append(rest, firstStackParamTag(popped), "")
	case ir.IRIndexStoreI32, ir.IRIndexStoreU8, ir.IRIndexStoreU16:
		popped, rest := popStackTags(stack, pop)
		if len(popped) > 0 && isStackParamTag(popped[0]) {
			return nil, nil, false
		}
		stack = rest
	case ir.IRMemWriteI32, ir.IRMemWriteU8, ir.IRMemWritePtr, ir.IRMemWriteArchPtr,
		ir.IRMemWriteI32Offset, ir.IRMemWriteU8Offset, ir.IRMemWritePtrOffset, ir.IRMemWriteArchPtrOffset,
		ir.IRMmioWriteI32, ir.IRAtomicStorePtr, ir.IRAtomicExchangePtr,
		ir.IRAtomicFetchAddPtr, ir.IRAtomicFetchSubPtr, ir.IRAtomicFetchAndPtr,
		ir.IRAtomicFetchOrPtr, ir.IRAtomicFetchXorPtr, ir.IRAtomicStoreI32,
		ir.IRAtomicExchangeI32, ir.IRAtomicFetchAddI32, ir.IRAtomicFetchSubI32,
		ir.IRAtomicFetchAndI32, ir.IRAtomicFetchOrI32, ir.IRAtomicFetchXorI32,
		ir.IRAtomicStoreI64, ir.IRAtomicExchangeI64, ir.IRAtomicFetchAddI64,
		ir.IRAtomicFetchSubI64, ir.IRAtomicFetchAndI64, ir.IRAtomicFetchOrI64,
		ir.IRAtomicFetchXorI64, ir.IRAtomicStoreI8, ir.IRAtomicExchangeI8,
		ir.IRAtomicFetchAddI8, ir.IRAtomicFetchSubI8, ir.IRAtomicFetchAndI8,
		ir.IRAtomicFetchOrI8, ir.IRAtomicFetchXorI8, ir.IRAtomicStoreI16,
		ir.IRAtomicExchangeI16, ir.IRAtomicFetchAddI16, ir.IRAtomicFetchSubI16,
		ir.IRAtomicFetchAndI16, ir.IRAtomicFetchOrI16, ir.IRAtomicFetchXorI16,
		ir.IRAtomicCompareExchangePtr, ir.IRAtomicCompareExchangeI32,
		ir.IRAtomicCompareExchangeI64, ir.IRAtomicCompareExchangeI8,
		ir.IRAtomicCompareExchangeI16:
		popped, rest := popStackTags(stack, pop)
		if len(popped) > 0 && isStackParamTag(popped[0]) {
			return nil, nil, false
		}
		stack = pushEmptyTags(rest, push)
	case ir.IRPtrAdd:
		popped, rest := popStackTags(stack, pop)
		stack = append(rest, firstStackParamTag(popped))
	default:
		_, rest := popStackTags(stack, pop)
		stack = pushEmptyTags(rest, push)
	}

	next := stackEscapeState{idx: cur.idx + 1, stack: stack, locals: locals}
	switch instr.Kind {
	case ir.IRJmp:
		next.idx = labels[instr.Label]
		return []stackEscapeState{next}, nil, true
	case ir.IRJmpIfZero:
		branch := cloneStackEscapeState(next)
		branch.idx = labels[instr.Label]
		return []stackEscapeState{next, branch}, nil, true
	default:
		return []stackEscapeState{next}, nil, true
	}
}

func validateFunctionStackAllocationsDoNotEscape(fn ir.IRFunc, tracked map[string]bool, callSummaries map[string]stackCallSummary) error {
	labels := map[int]int{}
	for i, instr := range fn.Instrs {
		if instr.Kind == ir.IRLabel {
			labels[instr.Label] = i
		}
	}
	work := []stackEscapeState{{
		idx:    0,
		locals: make([]string, fn.LocalSlots),
	}}
	seen := map[string]bool{}
	for len(work) > 0 {
		cur := work[len(work)-1]
		work = work[:len(work)-1]
		if cur.idx < 0 || cur.idx >= len(fn.Instrs) {
			continue
		}
		key := stackEscapeStateKey(cur)
		if seen[key] {
			continue
		}
		seen[key] = true
		next, err := stepStackEscapeState(fn, cur, labels, tracked, callSummaries)
		if err != nil {
			return err
		}
		work = append(work, next...)
	}
	return nil
}

func stepStackEscapeState(fn ir.IRFunc, cur stackEscapeState, labels map[int]int, tracked map[string]bool, callSummaries map[string]stackCallSummary) ([]stackEscapeState, error) {
	instr := fn.Instrs[cur.idx]
	stack := append([]string(nil), cur.stack...)
	locals := append([]string(nil), cur.locals...)
	pop, push, ok := validationStackEffect(instr)
	if !ok {
		return nil, fmt.Errorf("allocation lowering validation: %s instruction %d has unknown IR kind %d", fn.Name, cur.idx, instr.Kind)
	}

	switch instr.Kind {
	case ir.IRReturn:
		if tag := firstStackTag(stack); tag != "" {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d stack allocation %q escapes via return", fn.Name, cur.idx, tag)
		}
		return nil, nil
	case ir.IRCall:
		popped, rest := popStackTags(stack, pop)
		if tag := firstStackTag(popped); tag != "" {
			retTags, ok := stackCallReturnTags(instr, popped, callSummaries)
			if !ok {
				return nil, fmt.Errorf("allocation lowering validation: %s instruction %d stack allocation %q escapes via call %q", fn.Name, cur.idx, tag, instr.Name)
			}
			stack = append(rest, retTags...)
		} else {
			stack = pushEmptyTags(rest, push)
		}
	case ir.IRStoreGlobal:
		popped, rest := popStackTags(stack, pop)
		if tag := firstStackTag(popped); tag != "" {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d stack allocation %q escapes via global store", fn.Name, cur.idx, tag)
		}
		stack = rest
	case ir.IRStoreLocal:
		popped, rest := popStackTags(stack, pop)
		stack = rest
		if instr.Local >= 0 && instr.Local < len(locals) {
			locals[instr.Local] = firstStackTag(popped)
		}
	case ir.IRLoadLocal:
		if instr.Local >= 0 && instr.Local < len(locals) {
			stack = append(stack, locals[instr.Local])
		} else {
			stack = append(stack, "")
		}
	case ir.IRStackSliceU8, ir.IRStackSliceU16, ir.IRStackSliceI32,
		ir.IRRegionMakeSliceU8, ir.IRRegionMakeSliceU16, ir.IRRegionMakeSliceI32:
		_, rest := popStackTags(stack, pop)
		if tracked[instr.Name] {
			stack = append(rest, instr.Name, "")
		} else {
			stack = pushEmptyTags(rest, push)
		}
	case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32:
		_, rest := popStackTags(stack, pop)
		if tracked[instr.Name] {
			stack = append(rest, instr.Name, "")
		} else {
			stack = pushEmptyTags(rest, push)
		}
	case ir.IRRawSliceFromParts, ir.IRSliceWindow, ir.IRSlicePrefix, ir.IRSliceSuffix:
		popped, rest := popStackTags(stack, pop)
		tag := firstStackTag(popped)
		stack = append(rest, tag, "")
	case ir.IRIndexStoreI32, ir.IRIndexStoreU8, ir.IRIndexStoreU16:
		popped, rest := popStackTags(stack, pop)
		if len(popped) > 0 && popped[0] != "" {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d stack allocation %q escapes via indexed store value", fn.Name, cur.idx, popped[0])
		}
		stack = rest
	case ir.IRMemWriteI32, ir.IRMemWriteU8, ir.IRMemWritePtr, ir.IRMemWriteArchPtr,
		ir.IRMemWriteI32Offset, ir.IRMemWriteU8Offset, ir.IRMemWritePtrOffset, ir.IRMemWriteArchPtrOffset,
		ir.IRMmioWriteI32, ir.IRAtomicStorePtr, ir.IRAtomicExchangePtr,
		ir.IRAtomicFetchAddPtr, ir.IRAtomicFetchSubPtr, ir.IRAtomicFetchAndPtr,
		ir.IRAtomicFetchOrPtr, ir.IRAtomicFetchXorPtr, ir.IRAtomicStoreI32,
		ir.IRAtomicExchangeI32, ir.IRAtomicFetchAddI32, ir.IRAtomicFetchSubI32,
		ir.IRAtomicFetchAndI32, ir.IRAtomicFetchOrI32, ir.IRAtomicFetchXorI32,
		ir.IRAtomicStoreI64, ir.IRAtomicExchangeI64, ir.IRAtomicFetchAddI64,
		ir.IRAtomicFetchSubI64, ir.IRAtomicFetchAndI64, ir.IRAtomicFetchOrI64,
		ir.IRAtomicFetchXorI64, ir.IRAtomicStoreI8, ir.IRAtomicExchangeI8,
		ir.IRAtomicFetchAddI8, ir.IRAtomicFetchSubI8, ir.IRAtomicFetchAndI8,
		ir.IRAtomicFetchOrI8, ir.IRAtomicFetchXorI8, ir.IRAtomicStoreI16,
		ir.IRAtomicExchangeI16, ir.IRAtomicFetchAddI16, ir.IRAtomicFetchSubI16,
		ir.IRAtomicFetchAndI16, ir.IRAtomicFetchOrI16, ir.IRAtomicFetchXorI16,
		ir.IRAtomicCompareExchangePtr, ir.IRAtomicCompareExchangeI32,
		ir.IRAtomicCompareExchangeI64, ir.IRAtomicCompareExchangeI8,
		ir.IRAtomicCompareExchangeI16:
		popped, rest := popStackTags(stack, pop)
		if len(popped) > 0 && popped[0] != "" {
			return nil, fmt.Errorf("allocation lowering validation: %s instruction %d stack allocation %q escapes via memory store value", fn.Name, cur.idx, popped[0])
		}
		stack = pushEmptyTags(rest, push)
	case ir.IRPtrAdd:
		popped, rest := popStackTags(stack, pop)
		tag := firstStackTag(popped)
		stack = append(rest, tag)
	default:
		_, rest := popStackTags(stack, pop)
		stack = pushEmptyTags(rest, push)
	}

	next := stackEscapeState{idx: cur.idx + 1, stack: stack, locals: locals}
	switch instr.Kind {
	case ir.IRJmp:
		next.idx = labels[instr.Label]
		return []stackEscapeState{next}, nil
	case ir.IRJmpIfZero:
		branch := cloneStackEscapeState(next)
		branch.idx = labels[instr.Label]
		return []stackEscapeState{next, branch}, nil
	default:
		return []stackEscapeState{next}, nil
	}
}

func popStackTags(stack []string, count int) ([]string, []string) {
	if count <= 0 {
		return nil, stack
	}
	if count > len(stack) {
		count = len(stack)
	}
	popped := make([]string, count)
	for i := 0; i < count; i++ {
		popped[i] = stack[len(stack)-1-i]
	}
	return popped, stack[:len(stack)-count]
}

func stackReturnTags(stack []string, slots int) []string {
	tags := make([]string, slots)
	if slots <= 0 || len(stack) < slots {
		return tags
	}
	start := len(stack) - slots
	for i := 0; i < slots; i++ {
		if isStackParamTag(stack[start+i]) {
			tags[i] = stack[start+i]
		}
	}
	return tags
}

func mergeStackReturnTags(merged []string, ret []string) []string {
	if merged == nil {
		return append([]string(nil), ret...)
	}
	for i := range merged {
		if i >= len(ret) || merged[i] != ret[i] {
			merged[i] = ""
		}
	}
	return merged
}

func stackCallReturnTags(instr ir.IRInstr, popped []string, summaries map[string]stackCallSummary) ([]string, bool) {
	tags := make([]string, instr.RetSlots)
	summary, ok := summaries[instr.Name]
	if !ok {
		return nil, false
	}
	for i := range tags {
		if i >= len(summary.retTags) {
			continue
		}
		tags[i] = substituteStackSummaryTag(summary.retTags[i], instr.ArgSlots, popped)
	}
	return tags, true
}

func substituteStackSummaryTag(tag string, argSlots int, popped []string) string {
	slot := stackParamSlot(tag)
	if slot < 0 || slot >= argSlots || slot >= len(popped) {
		return ""
	}
	return popped[len(popped)-1-slot]
}

func stackParamTag(slot int) string {
	return fmt.Sprintf("stack-param:%d", slot)
}

func firstStackParamTag(tags []string) string {
	for _, tag := range tags {
		if isStackParamTag(tag) {
			return tag
		}
	}
	return ""
}

func isStackParamTag(tag string) bool {
	return stackParamSlot(tag) >= 0
}

func stackParamSlot(tag string) int {
	const prefix = "stack-param:"
	if !strings.HasPrefix(tag, prefix) {
		return -1
	}
	slot, err := strconv.Atoi(strings.TrimPrefix(tag, prefix))
	if err != nil {
		return -1
	}
	return slot
}

func pushEmptyTags(stack []string, count int) []string {
	for i := 0; i < count; i++ {
		stack = append(stack, "")
	}
	return stack
}

func firstStackTag(tags []string) string {
	for _, tag := range tags {
		if tag != "" {
			return tag
		}
	}
	return ""
}

func cloneStackEscapeState(state stackEscapeState) stackEscapeState {
	return stackEscapeState{
		idx:    state.idx,
		stack:  append([]string(nil), state.stack...),
		locals: append([]string(nil), state.locals...),
	}
}

func stackEscapeStateKey(state stackEscapeState) string {
	var b strings.Builder
	b.WriteString(strconv.Itoa(state.idx))
	b.WriteByte('|')
	for _, tag := range state.stack {
		b.WriteString(tag)
		b.WriteByte(',')
	}
	b.WriteByte('|')
	for i, tag := range state.locals {
		if tag == "" {
			continue
		}
		b.WriteString(strconv.Itoa(i))
		b.WriteByte('=')
		b.WriteString(tag)
		b.WriteByte(',')
	}
	return b.String()
}

func explicitIslandIRKind(alloc allocplan.Allocation) (ir.IRInstrKind, bool) {
	switch alloc.Builtin {
	case "core.island_make_u8":
		return ir.IRIslandMakeSliceU8, true
	case "core.island_make_u16":
		return ir.IRIslandMakeSliceU16, true
	case "core.island_make_i32", "core.island_make_bool":
		return ir.IRIslandMakeSliceI32, true
	default:
		return 0, false
	}
}

func islandIRKindName(kind ir.IRInstrKind) string {
	switch kind {
	case ir.IRIslandMakeSliceU8:
		return "IRIslandMakeSliceU8"
	case ir.IRIslandMakeSliceU16:
		return "IRIslandMakeSliceU16"
	case ir.IRIslandMakeSliceI32:
		return "IRIslandMakeSliceI32"
	case ir.IRIslandReset:
		return "IRIslandReset"
	default:
		return fmt.Sprintf("ir.%d", kind)
	}
}

func regionIRKind(alloc allocplan.Allocation) (ir.IRInstrKind, bool) {
	switch alloc.Builtin {
	case "core.slice_copy_u8", "core.string_copy":
		return ir.IRRegionMakeSliceU8, true
	case "core.slice_copy_u16":
		return ir.IRRegionMakeSliceU16, true
	case "core.slice_copy_i32", "core.slice_copy_bool":
		return ir.IRRegionMakeSliceI32, true
	default:
		return 0, false
	}
}

func regionIRKindName(kind ir.IRInstrKind) string {
	switch kind {
	case ir.IRRegionMakeSliceU8:
		return "IRRegionMakeSliceU8"
	case ir.IRRegionMakeSliceU16:
		return "IRRegionMakeSliceU16"
	case ir.IRRegionMakeSliceI32:
		return "IRRegionMakeSliceI32"
	default:
		return fmt.Sprintf("ir.%d", kind)
	}
}
