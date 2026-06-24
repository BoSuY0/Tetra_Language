package memorypipeline

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/memoryfacts"
	"tetra_language/compiler/internal/plir"
)

const memoryPipelineSchemaVersion = "tetra.memory-core-v2.state.v1"

func programIDForPLIR(
	target string,
	opt allocplan.Options,
	prog *plir.Program,
) (string, error) {
	if prog == nil {
		return "", fmt.Errorf("memorypipeline: missing PLIR program")
	}
	payload := struct {
		Schema    string            `json:"schema"`
		Target    string            `json:"target"`
		AllocPlan allocplan.Options `json:"alloc_plan"`
		PLIR      plir.Program      `json:"plir"`
	}{
		Schema:    memoryPipelineSchemaVersion,
		Target:    target,
		AllocPlan: opt,
		PLIR:      normalizePLIR(prog),
	}
	sum, err := digestPayload(payload)
	if err != nil {
		return "", err
	}
	return "program:sha256:" + sum, nil
}

func (s *State) ModulePlanDigest(module string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("memorypipeline: nil state")
	}
	if err := s.requirePhaseAtLeast(PhasePlanned); err != nil {
		return "", err
	}
	if s.PLIR == nil {
		return "", fmt.Errorf("memorypipeline: missing PLIR")
	}
	if s.Plan == nil {
		return "", fmt.Errorf("memorypipeline: missing allocation plan")
	}
	normalized := normalizePLIR(s.PLIR)
	functions, names := moduleFunctions(normalized, module)
	snapshot, err := s.Snapshot()
	if err != nil {
		return "", err
	}
	payload := struct {
		Schema        string                   `json:"schema"`
		Target        string                   `json:"target"`
		AllocPlan     allocplan.Options        `json:"alloc_plan"`
		Module        string                   `json:"module"`
		Functions     []plir.Function          `json:"functions"`
		Allocations   []allocplan.FunctionPlan `json:"allocations"`
		SourceFactIDs []string                 `json:"source_fact_ids"`
	}{
		Schema:        memoryPipelineSchemaVersion,
		Target:        s.Target,
		AllocPlan:     s.allocOptions,
		Module:        module,
		Functions:     functions,
		Allocations:   moduleAllocations(s.Plan, names),
		SourceFactIDs: moduleSourceFactIDs(snapshot, names),
	}
	sum, err := digestPayload(payload)
	if err != nil {
		return "", err
	}
	return "memory-plan:sha256:" + sum, nil
}

func digestPayload(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func normalizePLIR(prog *plir.Program) plir.Program {
	if prog == nil {
		return plir.Program{}
	}
	out := plir.Program{
		Funcs: make([]plir.Function, len(prog.Funcs)),
	}
	for i, fn := range prog.Funcs {
		out.Funcs[i] = cloneFunction(fn)
	}
	sort.Slice(out.Funcs, func(i, j int) bool {
		return functionKey(out.Funcs[i]) < functionKey(out.Funcs[j])
	})
	for i := range out.Funcs {
		normalizeFunction(&out.Funcs[i])
	}
	return out
}

func cloneFunction(fn plir.Function) plir.Function {
	out := fn
	out.Summary = cloneSummary(fn.Summary)
	out.Values = append([]plir.Value(nil), fn.Values...)
	out.Ops = make([]plir.Operation, len(fn.Ops))
	for i, op := range fn.Ops {
		out.Ops[i] = op
		out.Ops[i].Inputs = append([]string(nil), op.Inputs...)
		out.Ops[i].Outputs = append([]string(nil), op.Outputs...)
	}
	out.Facts = make([]plir.Fact, len(fn.Facts))
	for i, fact := range fn.Facts {
		out.Facts[i] = fact
		out.Facts[i].Uses = append([]string(nil), fact.Uses...)
	}
	out.Blocks = make([]plir.BasicBlock, len(fn.Blocks))
	for i, block := range fn.Blocks {
		out.Blocks[i] = block
		out.Blocks[i].Preds = append([]string(nil), block.Preds...)
		out.Blocks[i].Succs = append([]string(nil), block.Succs...)
		out.Blocks[i].Ops = append([]string(nil), block.Ops...)
	}
	out.Dominators = make([]plir.DominatorRow, len(fn.Dominators))
	for i, row := range fn.Dominators {
		out.Dominators[i] = row
		out.Dominators[i].Dominators = append([]string(nil), row.Dominators...)
	}
	out.ProofGuards = make([]plir.ProofGuard, len(fn.ProofGuards))
	for i, guard := range fn.ProofGuards {
		out.ProofGuards[i] = guard
		out.ProofGuards[i].Dominates = append([]plir.ProofUse(nil), guard.Dominates...)
	}
	out.ProofUses = append([]plir.ProofUse(nil), fn.ProofUses...)
	out.ProofTerms = make([]plir.ProofTerm, len(fn.ProofTerms))
	for i, term := range fn.ProofTerms {
		out.ProofTerms[i] = term
		out.ProofTerms[i].FactsUsed = append([]string(nil), term.FactsUsed...)
	}
	out.RangeFacts = make([]plir.RangeFact, len(fn.RangeFacts))
	for i, fact := range fn.RangeFacts {
		out.RangeFacts[i] = fact
		out.RangeFacts[i].Derivation = append([]string(nil), fact.Derivation...)
	}
	return out
}

func cloneSummary(summary *plir.FunctionSummary) *plir.FunctionSummary {
	if summary == nil {
		return nil
	}
	out := *summary
	out.ParamNames = append([]string(nil), summary.ParamNames...)
	out.ParamTypes = append([]string(nil), summary.ParamTypes...)
	out.ParamOwnership = append([]string(nil), summary.ParamOwnership...)
	out.Effects = append([]string(nil), summary.Effects...)
	out.ReturnRegionSummary = cloneStringIntMap(summary.ReturnRegionSummary)
	out.ReturnResourceSummary = cloneResourceSummary(summary.ReturnResourceSummary)
	out.ThrowResourceSummary = cloneResourceSummary(summary.ThrowResourceSummary)
	return &out
}

func cloneStringIntMap(values map[string]int) map[string]int {
	if values == nil {
		return nil
	}
	out := make(map[string]int, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func cloneResourceSummary(
	values map[string][]plir.ResourceProvenance,
) map[string][]plir.ResourceProvenance {
	if values == nil {
		return nil
	}
	out := make(map[string][]plir.ResourceProvenance, len(values))
	for key, value := range values {
		out[key] = append([]plir.ResourceProvenance(nil), value...)
	}
	return out
}

func normalizeFunction(fn *plir.Function) {
	if fn == nil {
		return
	}
	normalizeSummary(fn.Summary)
	sort.Slice(fn.Values, func(i, j int) bool {
		return fn.Values[i].ID < fn.Values[j].ID
	})
	sort.Slice(fn.Ops, func(i, j int) bool {
		return operationKey(fn.Ops[i]) < operationKey(fn.Ops[j])
	})
	sort.Slice(fn.Facts, func(i, j int) bool {
		return factKey(fn.Facts[i]) < factKey(fn.Facts[j])
	})
	for i := range fn.Facts {
		sort.Strings(fn.Facts[i].Uses)
	}
	sort.Slice(fn.Blocks, func(i, j int) bool {
		return fn.Blocks[i].ID < fn.Blocks[j].ID
	})
	for i := range fn.Blocks {
		sort.Strings(fn.Blocks[i].Preds)
		sort.Strings(fn.Blocks[i].Succs)
		sort.Strings(fn.Blocks[i].Ops)
	}
	sort.Slice(fn.Dominators, func(i, j int) bool {
		return fn.Dominators[i].Block < fn.Dominators[j].Block
	})
	for i := range fn.Dominators {
		sort.Strings(fn.Dominators[i].Dominators)
	}
	sort.Slice(fn.ProofGuards, func(i, j int) bool {
		return proofGuardKey(fn.ProofGuards[i]) < proofGuardKey(fn.ProofGuards[j])
	})
	for i := range fn.ProofGuards {
		sort.Slice(fn.ProofGuards[i].Dominates, func(a, b int) bool {
			return proofUseKey(fn.ProofGuards[i].Dominates[a]) <
				proofUseKey(fn.ProofGuards[i].Dominates[b])
		})
	}
	sort.Slice(fn.ProofUses, func(i, j int) bool {
		return proofUseKey(fn.ProofUses[i]) < proofUseKey(fn.ProofUses[j])
	})
	sort.Slice(fn.ProofTerms, func(i, j int) bool {
		return proofTermKey(fn.ProofTerms[i]) < proofTermKey(fn.ProofTerms[j])
	})
	for i := range fn.ProofTerms {
		sort.Strings(fn.ProofTerms[i].FactsUsed)
	}
	sort.Slice(fn.RangeFacts, func(i, j int) bool {
		return rangeFactKey(fn.RangeFacts[i]) < rangeFactKey(fn.RangeFacts[j])
	})
	for i := range fn.RangeFacts {
		sort.Strings(fn.RangeFacts[i].Derivation)
	}
}

func normalizeSummary(summary *plir.FunctionSummary) {
	if summary == nil {
		return
	}
	sort.Strings(summary.Effects)
	normalizeResourceSummary(summary.ReturnResourceSummary)
	normalizeResourceSummary(summary.ThrowResourceSummary)
}

func normalizeResourceSummary(values map[string][]plir.ResourceProvenance) {
	for key := range values {
		sort.Slice(values[key], func(i, j int) bool {
			if values[key][i].ParamIndex != values[key][j].ParamIndex {
				return values[key][i].ParamIndex < values[key][j].ParamIndex
			}
			return values[key][i].ParamPath < values[key][j].ParamPath
		})
	}
}

func moduleFunctions(prog plir.Program, module string) ([]plir.Function, map[string]struct{}) {
	var functions []plir.Function
	names := map[string]struct{}{}
	for _, fn := range prog.Funcs {
		if fn.Module != module {
			continue
		}
		functions = append(functions, fn)
		names[fn.Name] = struct{}{}
	}
	return functions, names
}

func moduleAllocations(
	plan *allocplan.Plan,
	functions map[string]struct{},
) []allocplan.FunctionPlan {
	if plan == nil {
		return nil
	}
	var out []allocplan.FunctionPlan
	for _, fn := range plan.Functions {
		if _, ok := functions[fn.Name]; !ok {
			continue
		}
		row := allocplan.FunctionPlan{
			Name:        fn.Name,
			Allocations: append([]allocplan.Allocation(nil), fn.Allocations...),
		}
		sort.Slice(row.Allocations, func(i, j int) bool {
			return allocationKey(row.Allocations[i]) < allocationKey(row.Allocations[j])
		})
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func moduleSourceFactIDs(
	snapshot memoryfacts.Snapshot,
	functions map[string]struct{},
) []string {
	var ids []string
	seen := map[string]struct{}{}
	for _, fact := range snapshot.Facts() {
		if _, ok := functions[fact.FunctionID]; !ok {
			continue
		}
		id := string(fact.ID)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func functionKey(fn plir.Function) string {
	return strings.Join([]string{fn.Module, fn.Name}, "\x00")
}

func operationKey(op plir.Operation) string {
	return strings.Join([]string{op.ID, string(op.Kind), op.Block, op.Source}, "\x00")
}

func factKey(fact plir.Fact) string {
	return strings.Join([]string{
		fact.Kind.String(),
		fact.ValueID,
		"",
		fact.Source,
		fact.ID,
	}, "\x00")
}

func proofGuardKey(guard plir.ProofGuard) string {
	return strings.Join([]string{
		guard.ID,
		guard.OpID,
		guard.Kind,
		guard.Block,
		guard.Condition,
		guard.Reason,
	}, "\x00")
}

func proofUseKey(use plir.ProofUse) string {
	return strings.Join([]string{
		use.ProofID,
		use.OpID,
		use.Block,
		use.UseKind,
		use.Source,
	}, "\x00")
}

func proofTermKey(term plir.ProofTerm) string {
	return strings.Join([]string{
		term.ID,
		term.Operation,
		term.Kind,
		term.SubjectBaseID,
		term.IndexValueID,
		term.Range,
		term.IslandID,
		fmt.Sprintf("%d", term.Epoch),
		term.BaseID,
		term.Source,
	}, "\x00")
}

func rangeFactKey(fact plir.RangeFact) string {
	copy := fact
	sort.Strings(copy.Derivation)
	raw, _ := json.Marshal(copy)
	return string(raw)
}

func allocationKey(alloc allocplan.Allocation) string {
	return strings.Join([]string{
		alloc.ID,
		alloc.SiteID,
		alloc.ValueID,
		alloc.Source,
		alloc.Builtin,
	}, "\x00")
}
