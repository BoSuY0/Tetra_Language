package lower

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/loweringevidence"
	corerangeproof "tetra_language/compiler/internal/rangeproof"
	"tetra_language/compiler/internal/semantics"
)

type ProgramResult struct {
	Program     *ir.IRProgram
	Evidence    LoweringEvidence
	moduleIndex map[string][]int
}

type LoweringEvidence = loweringevidence.Evidence
type AllocationLoweringEvidence = loweringevidence.Allocation

func LowerPlannedProgram(
	checked *semantics.CheckedProgram,
	plan *allocplan.Plan,
	opt Options,
) (*ProgramResult, error) {
	if checked == nil {
		return nil, fmt.Errorf("missing checked program")
	}
	if len(checked.Funcs) == 0 {
		return nil, fmt.Errorf("expected at least one function")
	}
	if plan == nil {
		return nil, fmt.Errorf("lower: missing allocation plan")
	}
	if err := allocplan.VerifyPlanned(plan); err != nil {
		return nil, err
	}
	return lowerWithVerifiedPlan(checked, plan, opt)
}

func lowerWithVerifiedPlan(
	checked *semantics.CheckedProgram,
	plan *allocplan.Plan,
	opt Options,
) (*ProgramResult, error) {
	allocationsByFunction := allocationPlanByFunction(plan)
	callBoundaryProofs := corerangeproof.CollectHashLookupCallBoundaryLenProofs(checked)
	helperSummaryProofs := corerangeproof.CollectHelperSummaryProofs(checked)
	helperOffsetProofs := corerangeproof.CollectHelperOffsetProofs(checked)
	ownedReturnSummaries := collectOwnedReturnSummaries(checked, opt)
	ownedThrowSummaries := collectOwnedThrowSummaries(checked, opt, ownedReturnSummaries)

	prog := ir.IRProgram{MainIndex: checked.MainIndex, MainName: checked.MainName}
	moduleIndex := map[string][]int{}
	recordFunc := func(module string, irFunc ir.IRFunc) {
		index := len(prog.Funcs)
		prog.Funcs = append(prog.Funcs, irFunc)
		moduleIndex[module] = append(moduleIndex[module], index)
	}

	wrappers := collectTypedTaskWrappers(checked, "")
	stagedTargets := collectStagedTypedTaskTargets(wrappers)
	callableTargets := collectFunctionTypedParamTargets(checked, "")
	for _, fn := range checked.Funcs {
		irFunc, err := lowerCheckedFuncWithOptions(
			fn,
			checked.Types,
			checked.FuncSigs,
			checked.GlobalsByModule[fn.Module],
			stagedTargets[fn.Name],
			callableTargets[fn.Name],
			ownedReturnSummaries,
			ownedThrowSummaries,
			opt,
			allocationsByFunction[fn.Name],
			callBoundaryProofs[fn.Name],
			helperSummaryProofs[fn.Name],
			helperOffsetProofs[fn.Name],
		)
		if err != nil {
			return nil, err
		}
		if err := VerifyFunc(irFunc); err != nil {
			return nil, err
		}
		recordFunc(fn.Module, irFunc)
	}
	for _, wrapper := range wrappers {
		irFunc, err := lowerTypedTaskWrapper(wrapper)
		if err != nil {
			return nil, err
		}
		if err := VerifyFunc(irFunc); err != nil {
			return nil, err
		}
		recordFunc(wrapper.Module, irFunc)
	}
	if err := VerifyProgram(&prog); err != nil {
		return nil, err
	}
	evidence, err := loweringEvidenceFromProgram(plan, &prog)
	if err != nil {
		return nil, err
	}
	return &ProgramResult{
		Program:     &prog,
		Evidence:    evidence,
		moduleIndex: moduleIndex,
	}, nil
}

func (r *ProgramResult) ModuleFuncs(module string) ([]ir.IRFunc, error) {
	if r == nil || r.Program == nil {
		return nil, fmt.Errorf("lower: nil program result")
	}
	indexes, ok := r.moduleIndex[module]
	if !ok {
		return nil, nil
	}
	out := make([]ir.IRFunc, 0, len(indexes))
	for _, index := range indexes {
		if index < 0 || index >= len(r.Program.Funcs) {
			return nil, fmt.Errorf("lower: invalid module index %d for module %q", index, module)
		}
		out = append(out, cloneIRFunc(r.Program.Funcs[index]))
	}
	return out, nil
}

func (r *ProgramResult) ModuleLoweringDigest(module string) (string, error) {
	funcs, err := r.ModuleFuncs(module)
	if err != nil {
		return "", err
	}
	funcNames := map[string]bool{}
	for _, fn := range funcs {
		funcNames[fn.Name] = true
	}
	evidence := make([]AllocationLoweringEvidence, 0, len(r.Evidence.Allocations))
	for _, row := range r.Evidence.Allocations {
		if funcNames[row.Function] {
			evidence = append(evidence, row)
		}
	}
	sortLoweringEvidence(evidence)
	payload := struct {
		Schema   string                       `json:"schema"`
		Module   string                       `json:"module"`
		Funcs    []ir.IRFunc                  `json:"funcs"`
		Evidence []AllocationLoweringEvidence `json:"evidence"`
	}{
		Schema:   "tetra.lowering.module.v1",
		Module:   module,
		Funcs:    funcs,
		Evidence: evidence,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "lowering:sha256:" + hex.EncodeToString(sum[:]), nil
}

func loweringEvidenceFromProgram(plan *allocplan.Plan, prog *ir.IRProgram) (LoweringEvidence, error) {
	rowsByKey := map[string]AllocationLoweringEvidence{}
	planByFunction := allocationPlanByFunction(plan)
	for _, fn := range prog.Funcs {
		allocations := planByFunction[fn.Name]
		if len(allocations) == 0 {
			continue
		}
		for index, instr := range fn.Instrs {
			actual, ok := actualStorageForAllocationInstr(instr)
			if !ok || instr.Name == "" {
				continue
			}
			alloc, ok := allocations[instr.Name]
			if !ok {
				continue
			}
			row := loweringEvidenceRow(fn.Name, alloc, actual, index, index)
			key := loweringEvidenceKey(fn.Name, alloc.ID)
			if previous, exists := rowsByKey[key]; exists {
				return LoweringEvidence{}, fmt.Errorf(
					"lower: allocation %s/%s emitted multiple evidence rows: %q and %q",
					fn.Name,
					alloc.ID,
					previous.ArtifactID,
					row.ArtifactID,
				)
			}
			rowsByKey[key] = row
		}
	}

	rows := make([]AllocationLoweringEvidence, 0)
	for _, fn := range plan.Functions {
		for _, alloc := range fn.Allocations {
			key := loweringEvidenceKey(fn.Name, alloc.ID)
			if row, ok := rowsByKey[key]; ok {
				rows = append(rows, row)
				continue
			}
			actual := allocplan.StorageUnknownConservative
			first := -1
			last := -1
			if plannedStorageForEvidence(alloc) == allocplan.StorageEliminated {
				actual = allocplan.StorageEliminated
			}
			rows = append(rows, loweringEvidenceRow(fn.Name, alloc, actual, first, last))
		}
	}
	sortLoweringEvidence(rows)
	return LoweringEvidence{Allocations: rows}, nil
}

func loweringEvidenceRow(
	function string,
	alloc allocplan.Allocation,
	actual allocplan.StorageClass,
	first int,
	last int,
) AllocationLoweringEvidence {
	planned := plannedStorageForEvidence(alloc)
	artifactID := fmt.Sprintf("ir:%s:%d:%d:%s", function, first, last, alloc.ID)
	if actual == allocplan.StorageEliminated {
		artifactID = fmt.Sprintf("ir:%s:eliminated:%s", function, alloc.ID)
	}
	decisionCode := "lowering:emitted:" + string(actual)
	reason := "emitted branch matched planned storage"
	if actual != planned {
		decisionCode = fmt.Sprintf("lowering:fallback:%s->%s", planned, actual)
		reason = "emitted branch differed from planned storage"
		if actual == allocplan.StorageUnknownConservative {
			reason = "allocation branch did not emit a lowered artifact"
		}
	}
	if alloc.DecisionCode != "" {
		decisionCode = alloc.DecisionCode + "|" + decisionCode
	}
	return AllocationLoweringEvidence{
		Function:         function,
		AllocationID:     alloc.ID,
		ValueID:          alloc.ValueID,
		PlannedStorage:   planned,
		ActualStorage:    actual,
		ArtifactID:       artifactID,
		DecisionCode:     decisionCode,
		Reason:           reason,
		SourceFactIDs:    append([]string(nil), alloc.SourceFactIDs...),
		ProofIDs:         append([]string(nil), alloc.ProofIDs...),
		PlanDigest:       alloc.PlanDigest,
		FirstInstruction: first,
		LastInstruction:  last,
	}
}

func actualStorageForAllocationInstr(instr ir.IRInstr) (allocplan.StorageClass, bool) {
	switch instr.Kind {
	case ir.IRAllocBytes, ir.IRMakeSliceU8, ir.IRMakeSliceU16, ir.IRMakeSliceI32:
		return allocplan.StorageHeap, true
	case ir.IRStackSliceU8, ir.IRStackSliceU16, ir.IRStackSliceI32:
		if instr.Local < 0 && instr.ArgSlots == 0 && instr.Imm == 0 {
			return allocplan.StorageEliminated, true
		}
		return allocplan.StorageStack, true
	case ir.IRRegionMakeSliceU8, ir.IRRegionMakeSliceU16, ir.IRRegionMakeSliceI32:
		return allocplan.StorageFunctionTempRegion, true
	case ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32:
		return allocplan.StorageExplicitIsland, true
	default:
		return "", false
	}
}

func plannedStorageForEvidence(alloc allocplan.Allocation) allocplan.StorageClass {
	if alloc.PlannedStorage != "" {
		return alloc.PlannedStorage
	}
	if alloc.Storage != "" {
		return alloc.Storage
	}
	return allocplan.StorageUnknownConservative
}

func loweringEvidenceKey(function, allocationID string) string {
	return function + "\x00" + allocationID
}

func sortLoweringEvidence(rows []AllocationLoweringEvidence) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Function != rows[j].Function {
			return rows[i].Function < rows[j].Function
		}
		if rows[i].AllocationID != rows[j].AllocationID {
			return rows[i].AllocationID < rows[j].AllocationID
		}
		return rows[i].ArtifactID < rows[j].ArtifactID
	})
}

func cloneIRFunc(fn ir.IRFunc) ir.IRFunc {
	fn.OwnedParams = append([]ir.IROwnedParam(nil), fn.OwnedParams...)
	fn.Instrs = append([]ir.IRInstr(nil), fn.Instrs...)
	for i := range fn.Instrs {
		fn.Instrs[i].Str = append([]byte(nil), fn.Instrs[i].Str...)
	}
	return fn
}
