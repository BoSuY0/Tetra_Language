package plir

import (
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/semantics"
)

func checkedProgram(t *testing.T, src string) *semantics.CheckedProgram {
	t.Helper()
	prog, err := frontend.Parse([]byte(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checked, err := semantics.Check(prog)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return checked
}

func TestSummaryReturnOwnershipInfersBorrowForRegionParamReturn(t *testing.T) {
	got := summaryReturnOwnership(semantics.FuncSig{
		ReturnType:          "[]u8",
		ReturnRegionSummary: semantics.ReturnRegionSummary{"": 0},
	})
	if got != "borrow" {
		t.Fatalf("summaryReturnOwnership = %q, want borrow", got)
	}
}

func TestSummaryReturnOwnershipPreservesExplicitOwnership(t *testing.T) {
	got := summaryReturnOwnership(semantics.FuncSig{
		ReturnType:        "[]u8",
		ReturnOwnership:   "consume",
		ReturnRegionParam: 0,
	})
	if got != "consume" {
		t.Fatalf("summaryReturnOwnership = %q, want explicit consume", got)
	}
}

func TestSummaryReturnOwnershipDoesNotInferBorrowForOwnedSliceReturn(t *testing.T) {
	got := summaryReturnOwnership(semantics.FuncSig{ReturnType: "[]u8"})
	if got != "" {
		t.Fatalf("summaryReturnOwnership = %q, want empty ownership", got)
	}
}

func TestFromCheckedProgramRecordsSliceLoopFacts(t *testing.T) {
	checked := checkedProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    var total = 0
    for x in xs:
        total = total + x
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum(xs)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}

	var fn Function
	for _, candidate := range prog.Funcs {
		if candidate.Name == "sum" {
			fn = candidate
			break
		}
	}
	if fn.Name == "" {
		t.Fatalf("missing PLIR function sum: %#v", prog.Funcs)
	}
	for _, fact := range []FactKind{FactProvenanceKnown, FactLenStable, FactIndexInRange, FactRegionAlive, FactBorrowedImm} {
		if !fn.HasFact(fact) {
			t.Fatalf("PLIR function facts missing %s: %#v", fact, fn.Facts)
		}
	}
	if len(fn.RangeFacts) != 1 || !containsString(fn.RangeFacts[0].Derivation, "less_than_len") {
		t.Fatalf("for collection range derivation = %#v", fn.RangeFacts)
	}
	dump := FormatText(prog)
	for _, want := range []string{
		"func sum",
		"provenance: param:xs",
		"fact index_in_range",
		"range: 0..xs.len",
	} {
		if !strings.Contains(dump, want) {
			t.Fatalf("PLIR dump missing %q:\n%s", want, dump)
		}
	}
}

func TestFromCheckedProgramRecordsNoAliasForExclusiveInoutSliceParam(t *testing.T) {
	checked := checkedProgram(t, `
func mutate(xs: inout []u8) -> Int
uses mem:
    xs[0] = 1
    return xs[0]

func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(1)
    return mutate(xs)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findFunction(t, prog, "mutate")
	param := findValue(t, fn, "param:xs")
	if param.Borrow != BorrowMut || param.Provenance.Kind != ProvenanceParam {
		t.Fatalf("param:xs = %+v, want mutable param provenance", param)
	}
	if !hasFactForValue(fn, FactBorrowedMut, "param:xs") {
		t.Fatalf("mutate missing borrowed_mut for param:xs: %#v", fn.Facts)
	}
	if !hasFactForValue(fn, FactNoAlias, "param:xs") {
		t.Fatalf("mutate missing no_alias for exclusive inout param:\n%s", FormatText(prog))
	}
}

func TestFromCheckedProgramDoesNotClaimNoAliasAfterRawInoutExposure(t *testing.T) {
	checked := checkedProgram(t, `
func expose(xs: inout []u8) -> Int
uses capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        var raw: []u8 = core.raw_slice_u8_from_parts(xs.ptr, xs.len, mem)
        raw[0] = 1
        return raw[0]
    return 0

func main() -> Int
uses alloc, capability, mem:
    var xs: []u8 = make_u8(1)
    return expose(xs)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findFunction(t, prog, "expose")
	if !hasFactForValue(fn, FactBorrowedMut, "param:xs") {
		t.Fatalf("expose missing borrowed_mut for inout param: %#v", fn.Facts)
	}
	if hasFactForValue(fn, FactNoAlias, "param:xs") {
		t.Fatalf("raw pointer exposure must kill no_alias for param:xs:\n%s", FormatText(prog))
	}
	if !hasFactForValue(fn, FactProvenanceUnknown, "view:raw") {
		t.Fatalf("raw gateway missing conservative provenance_unknown fact:\n%s", FormatText(prog))
	}
}

func TestFromCheckedProgramDoesNotClaimNoAliasAfterCallbackInoutBoundary(t *testing.T) {
	checked := checkedProgram(t, `
func touch(view: inout []i32) -> Int
uses mem:
    return view.len

func sum_callback(view: inout []i32, cb: fn(inout []i32) -> Int uses mem) -> Int
uses mem:
    cb(view)
    return view.len

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    xs[0] = 1
    return sum_callback(xs, touch)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findFunction(t, prog, "sum_callback")
	if !hasFactForValue(fn, FactBorrowedMut, "param:view") {
		t.Fatalf("sum_callback missing borrowed_mut for inout param: %#v", fn.Facts)
	}
	if hasFactForValue(fn, FactNoAlias, "param:view") {
		t.Fatalf("callback inout boundary must kill no_alias for param:view:\n%s", FormatText(prog))
	}
	foundBoundaryNote := false
	for _, op := range fn.Ops {
		if op.Kind == OpCall && strings.Contains(op.Note, "alias_boundary:function_typed_inout") {
			foundBoundaryNote = true
			break
		}
	}
	if !foundBoundaryNote {
		t.Fatalf("callback inout call missing alias boundary note:\n%s", FormatText(prog))
	}
}

func TestFromCheckedProgramTiesIslandAllocationFactsToIslandHandle(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, islands, mem:
    island(64) as isl:
        var xs: []u8 = core.island_make_u8(isl, 3)
        xs[0] = 1
        return xs[0]
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}
	fn := findFunction(t, prog, "main")

	var alloc Value
	for _, candidate := range fn.Values {
		if candidate.ID == "alloc_intent:xs" {
			alloc = candidate
			break
		}
	}
	if alloc.ID == "" {
		t.Fatalf("missing island allocation value: %#v", fn.Values)
	}
	if alloc.Provenance.Kind != ProvenanceIsland || alloc.Provenance.Root != "isl" {
		t.Fatalf("island allocation provenance = %+v, want island rooted at isl", alloc.Provenance)
	}
	if alloc.Region != "island:isl" {
		t.Fatalf("island allocation region = %q, want island:isl", alloc.Region)
	}

	var sawAllocOp bool
	var sawAlignedFact bool
	var sawIslandMemoryRefFact bool
	for _, op := range fn.Ops {
		if op.Kind == OpAllocIntent && containsString(op.Outputs, "alloc_intent:xs") {
			sawAllocOp = true
			if !containsString(op.Inputs, "isl") || !containsString(op.Inputs, "3") {
				t.Fatalf("island alloc op inputs = %#v, want island handle and length", op.Inputs)
			}
		}
	}
	for _, fact := range fn.Facts {
		if fact.Kind == FactAligned && fact.ValueID == "alloc_intent:xs" && fact.Region == "island:isl" {
			sawAlignedFact = true
		}
		if fact.ValueID == "alloc_intent:xs" && fact.IslandID == "island:isl" && fact.Epoch == 1 && fact.BaseID == "alloc_intent:xs" {
			sawIslandMemoryRefFact = true
		}
	}
	if !sawAllocOp {
		t.Fatalf("missing alloc_intent operation for xs: %#v", fn.Ops)
	}
	if !sawAlignedFact {
		t.Fatalf("missing island allocation alignment fact: %#v", fn.Facts)
	}
	if !sawIslandMemoryRefFact {
		t.Fatalf("missing island memory ref identity on allocation facts: %#v", fn.Facts)
	}
}

func TestFromCheckedProgramRecordsIslandResetEpochFact(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, islands, mem:
    unsafe {
        let isl: island = core.island_new(16)
        let next: island = core.island_reset(isl)
        let xs: []u8 = core.island_make_u8(next, 1)
        free(next)
        return xs.len
    }
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findFunction(t, prog, "main")
	found := false
	foundResetAllocation := false
	for _, fact := range fn.Facts {
		if fact.ValueID == "alloc_intent:xs" && fact.IslandID == "island:isl" && fact.Epoch == 2 && fact.BaseID == "alloc_intent:xs" {
			foundResetAllocation = true
		}
		if fact.Kind != FactIslandEpochAdvanced {
			continue
		}
		found = true
		if fact.IslandID != "island:isl" || fact.Epoch != 2 || fact.BaseID == "" {
			t.Fatalf("island reset fact = %+v, want island:isl epoch 2 with base_id\n%s", fact, FormatText(prog))
		}
	}
	if !found {
		t.Fatalf("main missing island_epoch_advanced fact:\n%s", FormatText(prog))
	}
	if !foundResetAllocation {
		t.Fatalf("reset allocation xs missing island:isl epoch 2 identity:\n%s", FormatText(prog))
	}
}

func TestFromCheckedProgramRecordsTypedActorMovedFacts(t *testing.T) {
	checked := checkedProgram(t, `
enum MoveMsg:
    case region(island, []i32)

func main() -> Int
uses actors, alloc, islands, mem:
    unsafe:
        var region: island = core.island_new(128)
        var xs: []i32 = core.island_make_i32(region, 2)
        let sent: Int = core.send_typed(core.self(), MoveMsg.region(region, xs))
        return sent
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findFunction(t, prog, "main")
	if !hasOperationKind(fn, OpActorSend) {
		t.Fatalf("main missing actor_send operation:\n%s", FormatText(prog))
	}
	for _, valueID := range []string{"local:region", "local:xs"} {
		fact, ok := findFactForValue(fn, FactMoved, valueID)
		if !ok {
			t.Fatalf("main missing moved fact for %s:\n%s", valueID, FormatText(prog))
		}
		for _, want := range []string{"core.send_typed", "typed actor ownership transfer"} {
			if !strings.Contains(fact.Source+" "+fact.Reason, want) {
				t.Fatalf("moved fact for %s missing %q: %#v", valueID, want, fact)
			}
		}
	}
	sliceMove, _ := findFactForValue(fn, FactMoved, "local:xs")
	if !containsString(sliceMove.Uses, "local:region") {
		t.Fatalf("slice moved fact should cite region owner use: %#v", sliceMove)
	}
}

func TestVerifyProgramRejectsFakeActorMovedFactClaims(t *testing.T) {
	err := VerifyFunction(Function{
		Name: "fake",
		Values: []Value{{
			ID:         "local:view",
			Kind:       ValueLocal,
			Type:       "[]u8",
			Provenance: Provenance{Kind: ProvenanceStack, Root: "local:view"},
			Borrow:     BorrowImm,
		}},
		Facts: []Fact{{
			ID:      "fake-move",
			Kind:    FactMoved,
			ValueID: "local:view",
			Source:  "core.send_typed",
			Reason:  "typed actor ownership transfer",
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "moved") || !strings.Contains(err.Error(), "borrowed") {
		t.Fatalf("fake moved borrowed fact error = %v, want moved/borrowed rejection", err)
	}
}

func TestVerifyFunctionRejectsModuleBoundaryBorrowedReturnWithoutRegionSummary(t *testing.T) {
	err := VerifyFunction(Function{
		Name:   "borrowView",
		Module: "math",
		Summary: &FunctionSummary{
			Public:          true,
			ParamNames:      []string{"xs"},
			ParamTypes:      []string{"[]u8"},
			ParamOwnership:  []string{"borrow"},
			ReturnType:      "[]u8",
			ReturnOwnership: "borrow",
		},
		Values: []Value{
			{
				ID:         "param:xs",
				Kind:       ValueParam,
				Type:       "[]u8",
				Source:     "math.tetra:2:17",
				Region:     "fn:borrowView",
				Provenance: Provenance{Kind: ProvenanceParam, Root: "xs"},
				Lifetime:   Lifetime{Birth: "entry", Death: "return", Owner: "xs"},
				Borrow:     BorrowImm,
				Escape:     EscapeReturn,
			},
		},
		Ops:   []Operation{{ID: "return_xs", Kind: OpReturn, Source: "math.tetra:3:5", Inputs: []string{"xs"}}},
		Facts: []Fact{{ID: "prov_xs", Kind: FactProvenanceKnown, ValueID: "param:xs", Source: "math.tetra:2:17", Reason: "parameter provenance"}},
	})
	if err == nil || !strings.Contains(err.Error(), "summary completeness") || !strings.Contains(err.Error(), "return_region_summary") {
		t.Fatalf("VerifyFunction error = %v, want borrowed return summary completeness rejection", err)
	}
}

func TestVerifyFunctionAllowsResourceModuleValueReturnWithoutBorrowRegionSummary(t *testing.T) {
	err := VerifyFunction(Function{
		Name:   "pass",
		Module: "lib.resources",
		Summary: &FunctionSummary{
			Public:     true,
			ParamNames: []string{"msg"},
			ParamTypes: []string{"resources.MoveMsg"},
			ReturnType: "resources.MoveMsg",
		},
		Values: []Value{{
			ID:         "param:msg",
			Kind:       ValueParam,
			Type:       "resources.MoveMsg",
			Source:     "resources.tetra:16:11",
			Region:     "fn:pass",
			Provenance: Provenance{Kind: ProvenanceParam, Root: "msg"},
			Lifetime:   Lifetime{Birth: "entry", Death: "return", Owner: "msg"},
			Borrow:     BorrowImm,
			Escape:     EscapeReturn,
		}},
		Ops: []Operation{{ID: "return_msg", Kind: OpReturn, Source: "resources.tetra:17:5", Inputs: []string{"msg"}}},
	})
	if err != nil {
		t.Fatalf("VerifyFunction rejected value return from resources module as borrowed region return: %v", err)
	}
}

func TestVerifyFunctionAllowsOwnedIslandReturnWithResourceSummary(t *testing.T) {
	err := VerifyFunction(Function{
		Name:   "alias_region",
		Module: "examples.microservices.memory_island_alias_region_service",
		Summary: &FunctionSummary{
			Public:     true,
			ParamNames: []string{"region"},
			ParamTypes: []string{"island"},
			ReturnType: "island",
			ReturnResourceSummary: map[string][]ResourceProvenance{
				"": {{ParamIndex: 0}},
			},
		},
		Values: []Value{{
			ID:         "param:region",
			Kind:       ValueParam,
			Type:       "island",
			Source:     "memory_island_alias_region_service.tetra:7:19",
			Region:     "fn:alias_region",
			Provenance: Provenance{Kind: ProvenanceParam, Root: "region"},
			Lifetime:   Lifetime{Birth: "entry", Death: "return", Owner: "region"},
			Borrow:     BorrowImm,
			Escape:     EscapeReturn,
		}},
		Ops: []Operation{{ID: "return_region", Kind: OpReturn, Source: "memory_island_alias_region_service.tetra:8:5", Inputs: []string{"region"}}},
	})
	if err != nil {
		t.Fatalf("VerifyFunction rejected owned island resource return as borrowed region return: %v", err)
	}
}

func TestFromCheckedProgramRecordsWhileRangeCFGAndProofUse(t *testing.T) {
	checked := checkedProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < xs.len:
        total = total + xs[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum(xs)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}

	fn := findFunction(t, prog, "sum")
	if len(fn.Blocks) == 0 {
		t.Fatalf("sum missing CFG blocks: %#v", fn)
	}
	if len(fn.Dominators) == 0 {
		t.Fatalf("sum missing dominance rows: %#v", fn)
	}
	if len(fn.ProofGuards) != 1 {
		t.Fatalf("sum proof guards = %#v, want one while guard", fn.ProofGuards)
	}
	if len(fn.ProofUses) != 1 {
		t.Fatalf("sum proof uses = %#v, want one index load use", fn.ProofUses)
	}
	guard := fn.ProofGuards[0]
	use := fn.ProofUses[0]
	if guard.Kind != "range" || guard.Condition != "i < xs.len" {
		t.Fatalf("guard = %+v, want i < xs.len range guard", guard)
	}
	if !Dominates(fn, guard.Block, use.Block) {
		t.Fatalf("guard block %s should dominate use block %s in %+v", guard.Block, use.Block, fn.Dominators)
	}
	if len(fn.RangeFacts) != 1 {
		t.Fatalf("range facts = %#v, want one while range fact", fn.RangeFacts)
	}
	if fn.RangeFacts[0].ProofID != guard.ID || fn.RangeFacts[0].Reason != "while loop range proof" {
		t.Fatalf("range fact = %+v, guard = %+v", fn.RangeFacts[0], guard)
	}
	if !containsString(fn.RangeFacts[0].Derivation, "non_negative") || !containsString(fn.RangeFacts[0].Derivation, "less_than_len") {
		t.Fatalf("range derivation = %#v, want non_negative and less_than_len", fn.RangeFacts[0].Derivation)
	}
	if len(fn.ProofTerms) != 1 {
		t.Fatalf("sum proof terms = %#v, want one typed proof term", fn.ProofTerms)
	}
	term := fn.ProofTerms[0]
	if term.ID != guard.ID ||
		term.Kind != "bounds_check" ||
		term.SubjectBaseID != "xs" ||
		term.IndexValueID != "local:i" ||
		term.Operation != "index_load" ||
		term.Range != "i in [0, xs.len)" {
		t.Fatalf("proof term = %+v, guard = %+v", term, guard)
	}
}

func TestFromCheckedProgramRecordsCommutedWhileRangeProof(t *testing.T) {
	checked := checkedProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < xs.len:
        total = total + xs[i]
        i = 1 + i
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum(xs)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}

	fn := findFunction(t, prog, "sum")
	if len(fn.ProofGuards) != 1 || fn.ProofGuards[0].Condition != "i < xs.len" {
		t.Fatalf("commuted increment proof guards = %#v, want one while guard", fn.ProofGuards)
	}
	if len(fn.ProofUses) != 1 {
		t.Fatalf("commuted increment proof uses = %#v, want one index load use", fn.ProofUses)
	}
}

func TestFromCheckedProgramInvalidatesWhileRangeProofAfterBaseReassignment(t *testing.T) {
	checked := checkedProgram(t, `
func sum_reassign(xs: []i32, ys: []i32) -> Int
uses mem:
    var view: []i32 = xs
    var total = 0
    var i = 0
    while i < view.len:
        view = ys
        total = total + view[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    var ys: []i32 = make_i32(1)
    xs[0] = 1
    ys[0] = 2
    return sum_reassign(xs, ys)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findFunction(t, prog, "sum_reassign")
	if len(fn.ProofUses) != 0 {
		t.Fatalf("base reassignment should invalidate while proof uses: %#v\n%s", fn.ProofUses, FormatText(prog))
	}
}

func TestFromCheckedProgramInvalidatesWhileRangeProofAfterInoutCall(t *testing.T) {
	checked := checkedProgram(t, `
func touch(view: inout []i32) -> Int
uses mem:
    return view.len

func sum_inout(view: inout []i32) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < view.len:
        touch(view)
        total = total + view[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    xs[0] = 1
    return sum_inout(xs)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findFunction(t, prog, "sum_inout")
	if len(fn.ProofUses) != 0 {
		t.Fatalf("inout call should invalidate while proof uses: %#v\n%s", fn.ProofUses, FormatText(prog))
	}
}

func TestFromCheckedProgramInvalidatesWhileRangeProofAfterCallbackInoutCall(t *testing.T) {
	checked := checkedProgram(t, `
func touch(view: inout []i32) -> Int
uses mem:
    return view.len

func sum_callback(view: inout []i32, cb: fn(inout []i32) -> Int uses mem) -> Int
uses mem:
    var total = 0
    var i = 0
    while i < view.len:
        cb(view)
        total = total + view[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(1)
    xs[0] = 1
    return sum_callback(xs, touch)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findFunction(t, prog, "sum_callback")
	if len(fn.ProofUses) != 0 {
		t.Fatalf("callback inout call should invalidate while proof uses: %#v\n%s", fn.ProofUses, FormatText(prog))
	}
}

func TestFromCheckedProgramRecordsLessEqualLenMinusOneRangeDerivation(t *testing.T) {
	checked := checkedProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    var total = 0
    var i = 0
    while i <= xs.len - 1:
        total = total + xs[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum(xs)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}

	fn := findFunction(t, prog, "sum")
	if len(fn.RangeFacts) != 1 {
		t.Fatalf("range facts = %#v, want one while range fact", fn.RangeFacts)
	}
	if !containsString(fn.RangeFacts[0].Derivation, "less_equal_len_minus_one") {
		t.Fatalf("range derivation = %#v, want less_equal_len_minus_one", fn.RangeFacts[0].Derivation)
	}
}

func TestFromCheckedProgramRecordsConstStepWhileRangeProof(t *testing.T) {
	checked := checkedProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    let step: Int = 1
    var total = 0
    var i = 0
    while i < xs.len:
        total = total + xs[i]
        i = i + step
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum(xs)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}

	fn := findFunction(t, prog, "sum")
	if len(fn.ProofGuards) != 1 || fn.ProofGuards[0].Condition != "i < xs.len" {
		t.Fatalf("const step proof guards = %#v, want one while guard", fn.ProofGuards)
	}
	if len(fn.ProofUses) != 1 {
		t.Fatalf("const step proof uses = %#v, want one index load use", fn.ProofUses)
	}
}

func TestFromCheckedProgramRecordsNotEqualLenWhileRangeProof(t *testing.T) {
	checked := checkedProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    var total = 0
    var i = 0
    while i != xs.len:
        total = total + xs[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum(xs)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}

	fn := findFunction(t, prog, "sum")
	if len(fn.ProofGuards) != 1 || fn.ProofGuards[0].Condition != "i != xs.len" {
		t.Fatalf("!= len proof guards = %#v, want one while guard", fn.ProofGuards)
	}
	if len(fn.ProofUses) != 1 {
		t.Fatalf("!= len proof uses = %#v, want one index load use", fn.ProofUses)
	}
	if len(fn.RangeFacts) != 1 || fn.RangeFacts[0].Upper.Symbol != "xs.len" || fn.RangeFacts[0].InclusiveUpper {
		t.Fatalf("!= len range facts = %#v, want exclusive upper xs.len", fn.RangeFacts)
	}
}

func TestFromCheckedProgramRecordsStartEndAliasWhileRangeProof(t *testing.T) {
	checked := checkedProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    let start: Int = 0
    let end: Int = xs.len
    var total = 0
    var i = start
    while i < end:
        total = total + xs[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum(xs)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}

	fn := findFunction(t, prog, "sum")
	if len(fn.ProofGuards) != 1 || fn.ProofGuards[0].Condition != "i < end" {
		t.Fatalf("start/end alias proof guards = %#v, want one i < end while guard", fn.ProofGuards)
	}
	if len(fn.ProofUses) != 1 {
		t.Fatalf("start/end alias proof uses = %#v, want one index load use", fn.ProofUses)
	}
	if len(fn.RangeFacts) != 1 || fn.RangeFacts[0].Upper.Symbol != "xs.len" || fn.RangeFacts[0].InclusiveUpper {
		t.Fatalf("start/end alias range facts = %#v, want exclusive upper xs.len", fn.RangeFacts)
	}
}

func TestFromCheckedProgramRecordsAllocationLengthAliasWhileRangeProof(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, mem:
    let n: Int = 4
    var xs: []i32 = make_i32(n)
    var i = 0
    while i < n:
        xs[i] = i
        i = i + 1
    var total = 0
    i = 0
    while i < n:
        total = total + xs[i]
        i = i + 1
    return total
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}

	fn := findFunction(t, prog, "main")
	if len(fn.ProofGuards) != 2 || fn.ProofGuards[0].Condition != "i < n" || fn.ProofGuards[1].Condition != "i < n" {
		t.Fatalf("allocation length alias proof guards = %#v, want two i < n while guards", fn.ProofGuards)
	}
	if len(fn.ProofUses) != 1 {
		t.Fatalf("allocation length alias proof uses = %#v, want one index load use", fn.ProofUses)
	}
	if len(fn.RangeFacts) != 2 {
		t.Fatalf("allocation length alias range facts = %#v, want two range facts", fn.RangeFacts)
	}
	for _, fact := range fn.RangeFacts {
		if fact.Upper.Symbol != "xs.len" || fact.InclusiveUpper {
			t.Fatalf("allocation length alias range fact = %#v, want exclusive upper xs.len", fact)
		}
	}
}

func TestFromCheckedProgramPropagatesImmutableLocalConstAllocationLength(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, mem:
    let n: Int = 256
    var xs: []i32 = make_i32(n)
    return xs.len
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}

	fn := findFunction(t, prog, "main")
	xs := findValue(t, fn, "alloc_intent:xs")
	if xs.Alloc == nil {
		t.Fatalf("xs alloc intent missing: %#v", xs)
	}
	if !xs.Alloc.LengthConstKnown || xs.Alloc.LengthConst != 256 {
		t.Fatalf("xs length const = known:%v value:%d, want known:256", xs.Alloc.LengthConstKnown, xs.Alloc.LengthConst)
	}
	if xs.Alloc.LengthExpr != "n" {
		t.Fatalf("xs length expr = %q, want n", xs.Alloc.LengthExpr)
	}
}

func TestFromCheckedProgramDoesNotRecordMutableAllocationLengthAliasWhileRangeProof(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, mem:
    var n: Int = 4
    var xs: []i32 = make_i32(n)
    var total = 0
    var i = 0
    while i < n:
        total = total + xs[i]
        i = i + 1
    return total
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}

	fn := findFunction(t, prog, "main")
	if len(fn.ProofGuards) != 0 || len(fn.ProofUses) != 0 {
		t.Fatalf("mutable allocation length alias should not receive proof guards/uses: %#v %#v", fn.ProofGuards, fn.ProofUses)
	}
	for _, fact := range fn.RangeFacts {
		if fact.Upper.Symbol == "xs.len" {
			t.Fatalf("mutable allocation length alias received xs.len range fact: %#v", fact)
		}
	}
}

func TestFromCheckedProgramKeepsMutableLocalAllocationLengthRuntimeGuarded(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, mem:
    var n: Int = 256
    var xs: []i32 = make_i32(n)
    return xs.len
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}

	fn := findFunction(t, prog, "main")
	xs := findValue(t, fn, "alloc_intent:xs")
	if xs.Alloc == nil {
		t.Fatalf("xs alloc intent missing: %#v", xs)
	}
	if xs.Alloc.LengthConstKnown {
		t.Fatalf("mutable xs length const = known:%v value:%d, want runtime guarded", xs.Alloc.LengthConstKnown, xs.Alloc.LengthConst)
	}
}

func TestAllocationLengthAliasRejectsMutableMakeLengthBuiltins(t *testing.T) {
	for _, name := range []string{
		"make_u8", "make_u16", "make_i32", "make_bool",
		"core.make_u8", "core.make_u16", "core.make_i32", "core.make_bool",
	} {
		t.Run(name, func(t *testing.T) {
			b := &builder{fn: semantics.CheckedFunc{Locals: map[string]semantics.LocalInfo{
				"n": {Mutable: true},
			}}}
			_, ok := b.allocationLengthBoundLocal(&frontend.CallExpr{
				Name: name,
				Args: []frontend.Expr{&frontend.IdentExpr{Name: "n"}},
			})
			if ok {
				t.Fatalf("allocationLengthBoundLocal(%s, mutable n) accepted mutable length", name)
			}
		})
	}
}

func TestFromCheckedProgramRecordsViewEndAliasWhileRangeProof(t *testing.T) {
	checked := checkedProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    let view: []i32 = xs.prefix(2)
    let end: Int = view.len
    var total = 0
    var i = 0
    while i < end:
        total = total + view[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(4)
    xs[0] = 1
    xs[1] = 2
    return sum(xs)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}

	fn := findFunction(t, prog, "sum")
	if len(fn.ProofGuards) != 1 || fn.ProofGuards[0].Condition != "i < end" {
		t.Fatalf("view end alias proof guards = %#v, want one i < end while guard", fn.ProofGuards)
	}
	if len(fn.ProofUses) != 1 {
		t.Fatalf("view end alias proof uses = %#v, want one index load use", fn.ProofUses)
	}
	if len(fn.RangeFacts) != 1 || fn.RangeFacts[0].Upper.Symbol != "view.len" || fn.RangeFacts[0].InclusiveUpper {
		t.Fatalf("view end alias range facts = %#v, want exclusive upper view.len", fn.RangeFacts)
	}
}

func TestFromCheckedProgramRecordsAliasWhileRangeProof(t *testing.T) {
	checked := checkedProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    let ys: []i32 = xs
    var total = 0
    var i = 0
    while i < ys.len:
        total = total + ys[i]
        i = i + 1
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(2)
    xs[0] = 1
    xs[1] = 2
    return sum(xs)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}

	fn := findFunction(t, prog, "sum")
	if len(fn.ProofGuards) != 1 || fn.ProofGuards[0].Condition != "i < ys.len" {
		t.Fatalf("alias proof guards = %#v, want one ys.len range guard", fn.ProofGuards)
	}
	if len(fn.ProofUses) != 1 {
		t.Fatalf("alias proof uses = %#v, want one index load use", fn.ProofUses)
	}
	if len(fn.RangeFacts) != 1 || fn.RangeFacts[0].Upper.Symbol != "ys.len" {
		t.Fatalf("alias range facts = %#v, want upper ys.len", fn.RangeFacts)
	}
}

func TestRawSliceAliasWhileLoopDoesNotReceiveIndexRangeFact(t *testing.T) {
	checked := checkedProgram(t, `
func sum_raw(xs: []u8) -> Int
uses capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let view: []u8 = core.raw_slice_u8_from_parts(xs.ptr, xs.len, mem)
        let alias: []u8 = view
        var total = 0
        var i = 0
        while i < alias.len:
            total = total + alias[i]
            i = i + 1
        return total
    return 0

func main() -> Int
uses alloc, capability, mem:
    var xs: []u8 = make_u8(1)
    xs[0] = 1
    return sum_raw(xs)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}
	fn := findFunction(t, prog, "sum_raw")
	unknownLocals := map[string]bool{}
	for _, fact := range fn.Facts {
		if fact.Kind == FactIndexInRange {
			t.Fatalf("raw alias loop must not receive index_in_range fact: %#v", fn.Facts)
		}
		if fact.Kind == FactProvenanceKnown && (fact.ValueID == "local:view" || fact.ValueID == "local:alias") {
			t.Fatalf("raw alias local must not receive provenance_known fact: %#v", fact)
		}
		if fact.Kind == FactProvenanceUnknown && (fact.ValueID == "local:view" || fact.ValueID == "local:alias") {
			unknownLocals[fact.ValueID] = true
		}
	}
	if len(fn.ProofGuards) != 0 || len(fn.RangeFacts) != 0 {
		t.Fatalf("raw alias loop must not receive proof metadata: guards=%#v ranges=%#v", fn.ProofGuards, fn.RangeFacts)
	}
	for _, valueID := range []string{"local:view", "local:alias"} {
		if !unknownLocals[valueID] {
			t.Fatalf("raw alias local %s should record conservative unknown provenance: %#v", valueID, fn.Facts)
		}
	}
}

func TestBranchJoinInvalidAliasDoesNotReceiveWhileRangeProof(t *testing.T) {
	checked := checkedProgram(t, `
func sum_join(flag: Int) -> Int
uses mem:
    var alias: String = "abc"
    if flag:
        alias = core.string_window("abc", 4, 0)
    else:
        alias = "abc"
    var total = 0
    var i = 0
    while i < alias.len:
        total = total + alias[i]
        i = i + 1
    return total

func main() -> Int
uses mem:
    return sum_join(1)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}
	fn := findFunction(t, prog, "sum_join")
	for _, fact := range fn.Facts {
		if fact.Kind == FactIndexInRange {
			t.Fatalf("branch-joined invalid alias must not receive index_in_range fact: %#v", fn.Facts)
		}
	}
	if len(fn.ProofGuards) != 0 || len(fn.RangeFacts) != 0 {
		t.Fatalf("branch-joined invalid alias must not receive proof metadata: guards=%#v ranges=%#v", fn.ProofGuards, fn.RangeFacts)
	}
}

func TestVerifierRejectsUnknownProofUse(t *testing.T) {
	prog := &Program{Funcs: []Function{{
		Name: "bad",
		Blocks: []BasicBlock{
			{ID: "entry", Kind: "entry", Entry: true, Succs: []string{"body"}, Ops: []string{"op0"}},
			{ID: "body", Kind: "body", Preds: []string{"entry"}, Ops: []string{"op1"}},
		},
		Ops: []Operation{
			{ID: "op0", Kind: OpGuard, Block: "entry"},
			{ID: "op1", Kind: OpIndexLoad, Block: "body"},
		},
		ProofUses: []ProofUse{{
			ProofID: "proof:missing",
			Block:   "body",
			OpID:    "op1",
			UseKind: "bounds_check",
		}},
	}}}
	err := VerifyProgram(prog)
	if err == nil || !strings.Contains(err.Error(), "unknown proof id") {
		t.Fatalf("VerifyProgram error = %v, want unknown proof id", err)
	}
}

func TestVerifierRejectsNonDominatingProofUse(t *testing.T) {
	prog := &Program{Funcs: []Function{{
		Name: "bad",
		Values: []Value{{
			ID:         "local:i",
			Type:       "i32",
			Provenance: Provenance{Kind: ProvenanceStack, Root: "i"},
		}},
		Blocks: []BasicBlock{
			{ID: "entry", Kind: "entry", Entry: true, Succs: []string{"then", "sibling"}},
			{ID: "then", Kind: "then", Preds: []string{"entry"}, Ops: []string{"op0"}, Succs: []string{"join"}},
			{ID: "sibling", Kind: "else", Preds: []string{"entry"}, Ops: []string{"op1"}, Succs: []string{"join"}},
			{ID: "join", Kind: "join", Preds: []string{"then", "sibling"}, Exit: true},
		},
		Ops: []Operation{
			{ID: "op0", Kind: OpGuard, Block: "then"},
			{ID: "op1", Kind: OpIndexLoad, Block: "sibling"},
		},
		Facts: []Fact{{
			ID:      "f0",
			Kind:    FactIndexInRange,
			ValueID: "local:i",
			Range:   "0..xs.len",
			ProofID: "proof:branch",
			Source:  "test:1:1",
		}},
		ProofGuards: []ProofGuard{{
			ID:        "proof:branch",
			Kind:      "range",
			Block:     "then",
			OpID:      "op0",
			Condition: "i < xs.len",
		}},
		ProofUses: []ProofUse{{
			ProofID: "proof:branch",
			Block:   "sibling",
			OpID:    "op1",
			UseKind: "bounds_check",
		}},
	}}}
	err := VerifyProgram(prog)
	if err == nil || !strings.Contains(err.Error(), "does not dominate") {
		t.Fatalf("VerifyProgram error = %v, want non-dominating proof rejection", err)
	}
}

func TestVerifierRejectsInvertedRangeFact(t *testing.T) {
	prog := &Program{Funcs: []Function{{
		Name: "bad",
		RangeFacts: []RangeFact{{
			Value:          "i",
			Lower:          Bound{Kind: BoundConst, Const: 2},
			Upper:          Bound{Kind: BoundConst, Const: 1},
			InclusiveLower: true,
			InclusiveUpper: true,
			Source:         "test:1:1",
			ProofID:        "proof:range",
		}},
	}}}
	err := VerifyProgram(prog)
	if err == nil || !strings.Contains(err.Error(), "lower bound exceeds upper bound") {
		t.Fatalf("VerifyProgram error = %v, want inverted range rejection", err)
	}
}

func TestInvalidStringWindowLoopDoesNotReceiveIndexRangeFact(t *testing.T) {
	checked := checkedProgram(t, `
func sum_bad() -> Int:
    var total = 0
    for ch in core.string_window("abc", 4, 0):
        total = total + ch
    return total

func main() -> Int:
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	fn := findFunction(t, prog, "sum_bad")
	for _, fact := range fn.Facts {
		if fact.Kind == FactIndexInRange {
			t.Fatalf("invalid view loop must not receive index_in_range fact: %#v", fn.Facts)
		}
	}
}

func TestVerifierRejectsLenStableWithoutKnownProvenance(t *testing.T) {
	prog := &Program{Funcs: []Function{{
		Name: "bad",
		Values: []Value{{
			ID:         "v0",
			Type:       "[]u8",
			Region:     "r0",
			Provenance: Provenance{Kind: ProvenanceUnknown},
		}},
		Facts: []Fact{{
			ID:      "f0",
			Kind:    FactLenStable,
			ValueID: "v0",
		}},
	}}}
	err := VerifyProgram(prog)
	if err == nil || !strings.Contains(err.Error(), "len_stable requires known provenance") {
		t.Fatalf("VerifyProgram error = %v", err)
	}
}

func TestVerifierRejectsNoAliasWithoutExclusiveMutableBorrow(t *testing.T) {
	prog := &Program{Funcs: []Function{{
		Name: "bad",
		Values: []Value{{
			ID:         "param:xs",
			Kind:       ValueParam,
			Type:       "[]u8",
			Provenance: Provenance{Kind: ProvenanceParam, Root: "param:xs"},
			Lifetime:   Lifetime{Birth: "entry", Death: "return", Owner: "xs"},
			Borrow:     BorrowImm,
		}},
		Facts: []Fact{{
			ID:      "f0",
			Kind:    FactNoAlias,
			ValueID: "param:xs",
			Reason:  "forged immutable alias claim",
		}},
	}}}
	err := VerifyProgram(prog)
	if err == nil || !strings.Contains(err.Error(), "no_alias requires mutable borrow") {
		t.Fatalf("VerifyProgram error = %v, want no_alias mutable-borrow rejection", err)
	}
}

func TestVerifierRejectsNoAliasForExternalProvenance(t *testing.T) {
	prog := &Program{Funcs: []Function{{
		Name: "bad",
		Values: []Value{{
			ID:         "param:xs",
			Kind:       ValueParam,
			Type:       "[]u8",
			Provenance: Provenance{Kind: ProvenanceExternal, Root: "raw_parts"},
			Lifetime:   Lifetime{Birth: "entry", Death: "return", Owner: "xs"},
			Borrow:     BorrowMut,
		}},
		Facts: []Fact{
			{ID: "f0", Kind: FactBorrowedMut, ValueID: "param:xs"},
			{ID: "f1", Kind: FactRegionAlive, ValueID: "param:xs", Region: "fn:bad"},
			{ID: "f2", Kind: FactNoAlias, ValueID: "param:xs", Reason: "forged external alias claim"},
		},
	}}}
	err := VerifyProgram(prog)
	if err == nil || !strings.Contains(err.Error(), "no_alias requires parameter provenance") {
		t.Fatalf("VerifyProgram error = %v, want no_alias provenance rejection", err)
	}
}
