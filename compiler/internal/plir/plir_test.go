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

func TestVerifyFunctionAcceptsResourceAggregateReturnWithoutBorrowedRegionOwnership(t *testing.T) {
	err := VerifyFunction(Function{
		Name:   "pass",
		Module: "lib.resources",
		Summary: &FunctionSummary{
			ParamNames: []string{"msg"},
			ParamTypes: []string{"lib.resources.MoveMsg"},
			ReturnType: "lib.resources.MoveMsg",
			ReturnResourceSummary: map[string][]ResourceProvenance{
				"": {{ParamIndex: 0}},
			},
		},
		Values: []Value{
			{
				ID:         "param:msg",
				Kind:       ValueParam,
				Type:       "lib.resources.MoveMsg",
				Source:     "resources.tetra:16:11",
				Provenance: Provenance{Kind: ProvenanceParam, Root: "msg"},
				Lifetime:   Lifetime{Birth: "entry", Death: "return", Owner: "msg"},
				Escape:     EscapeReturn,
			},
		},
		Ops:   []Operation{{ID: "return_msg", Kind: OpReturn, Source: "resources.tetra:17:5", Inputs: []string{"msg"}}},
		Facts: []Fact{{ID: "prov_msg", Kind: FactProvenanceKnown, ValueID: "param:msg", Source: "resources.tetra:16:11", Reason: "parameter provenance"}},
	})
	if err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
}

func TestVerifyFunctionAcceptsOwnedSliceParamReturnWithoutBorrowedRegionOwnership(t *testing.T) {
	err := VerifyFunction(Function{
		Name:   "pass",
		Module: "lib.async_slice_memory",
		Summary: &FunctionSummary{
			ParamNames:          []string{"bytes"},
			ParamTypes:          []string{"[]u8"},
			ParamOwnership:      []string{""},
			ReturnType:          "[]u8",
			ReturnRegionSummary: map[string]int{"": 0},
		},
		Values: []Value{
			{
				ID:         "param:bytes",
				Kind:       ValueParam,
				Type:       "[]u8",
				Source:     "async_slice_memory.tetra:9:17",
				Region:     "fn:pass",
				Provenance: Provenance{Kind: ProvenanceParam, Root: "bytes"},
				Lifetime:   Lifetime{Birth: "entry", Death: "return", Owner: "bytes"},
				Borrow:     BorrowImm,
				Escape:     EscapeReturn,
			},
		},
		Ops:   []Operation{{ID: "return_bytes", Kind: OpReturn, Source: "async_slice_memory.tetra:10:5", Inputs: []string{"bytes"}}},
		Facts: []Fact{{ID: "prov_bytes", Kind: FactProvenanceKnown, ValueID: "param:bytes", Source: "async_slice_memory.tetra:9:17", Reason: "parameter provenance"}},
	})
	if err != nil {
		t.Fatalf("VerifyFunction: %v", err)
	}
}

func TestVerifyFunctionAcceptsIslandResourceReturnWithoutBorrowedRegionOwnership(t *testing.T) {
	err := VerifyFunction(Function{
		Name:   "alias_region",
		Module: "examples.microservices.memory_island_alias_region_service",
		Summary: &FunctionSummary{
			ParamNames: []string{"region"},
			ParamTypes: []string{"island"},
			ReturnType: "island",
			ReturnResourceSummary: map[string][]ResourceProvenance{
				"": {{ParamIndex: 0}},
			},
		},
		Values: []Value{
			{
				ID:         "param:region",
				Kind:       ValueParam,
				Type:       "island",
				Source:     "service.tetra:7:19",
				Provenance: Provenance{Kind: ProvenanceParam, Root: "region"},
				Lifetime:   Lifetime{Birth: "entry", Death: "return", Owner: "region"},
				Escape:     EscapeReturn,
			},
		},
		Ops:   []Operation{{ID: "return_region", Kind: OpReturn, Source: "service.tetra:8:5", Inputs: []string{"region"}}},
		Facts: []Fact{{ID: "prov_region", Kind: FactProvenanceKnown, ValueID: "param:region", Source: "service.tetra:7:19", Reason: "parameter provenance"}},
	})
	if err != nil {
		t.Fatalf("VerifyFunction: %v", err)
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

func findFunction(t *testing.T, prog *Program, name string) Function {
	t.Helper()
	for _, candidate := range prog.Funcs {
		if candidate.Name == name {
			return candidate
		}
	}
	t.Fatalf("missing PLIR function %s: %#v", name, prog.Funcs)
	return Function{}
}

func findValue(t *testing.T, fn Function, valueID string) Value {
	t.Helper()
	for _, value := range fn.Values {
		if value.ID == valueID {
			return value
		}
	}
	t.Fatalf("missing value %s in %s: %#v", valueID, fn.Name, fn.Values)
	return Value{}
}

func hasFactForValue(fn Function, kind FactKind, valueID string) bool {
	for _, fact := range fn.Facts {
		if fact.Kind == kind && fact.ValueID == valueID {
			return true
		}
	}
	return false
}

func findFactForValue(fn Function, kind FactKind, valueID string) (Fact, bool) {
	for _, fact := range fn.Facts {
		if fact.Kind == kind && fact.ValueID == valueID {
			return fact, true
		}
	}
	return Fact{}, false
}

func hasOperationKind(fn Function, kind OperationKind) bool {
	for _, op := range fn.Ops {
		if op.Kind == kind {
			return true
		}
	}
	return false
}

func TestVerifierRejectsContradictoryBorrowAndOwnershipFacts(t *testing.T) {
	prog := &Program{Funcs: []Function{{
		Name: "bad",
		Values: []Value{{
			ID:         "v0",
			Type:       "[]u8",
			Borrow:     BorrowImm,
			Provenance: Provenance{Kind: ProvenanceParam, Root: "xs"},
		}},
		Facts: []Fact{{
			ID:      "f0",
			Kind:    FactOwned,
			ValueID: "v0",
		}},
	}}}
	err := VerifyProgram(prog)
	if err == nil || !strings.Contains(err.Error(), "owned contradicts borrowed value") {
		t.Fatalf("VerifyProgram error = %v", err)
	}
}

func TestVerifierRejectsNoEscapeOnEscapingValue(t *testing.T) {
	prog := &Program{Funcs: []Function{{
		Name: "bad",
		Values: []Value{{
			ID:         "v0",
			Type:       "[]u8",
			Escape:     EscapeReturn,
			Provenance: Provenance{Kind: ProvenanceParam, Root: "xs"},
		}},
		Facts: []Fact{{
			ID:      "f0",
			Kind:    FactNoEscape,
			ValueID: "v0",
		}},
	}}}
	err := VerifyProgram(prog)
	if err == nil || !strings.Contains(err.Error(), "no_escape contradicts escaping value") {
		t.Fatalf("VerifyProgram error = %v", err)
	}
}

func TestVerifierRejectsBorrowedFactWithoutNoEscape(t *testing.T) {
	prog := &Program{Funcs: []Function{{
		Name: "bad",
		Values: []Value{{
			ID:         "v0",
			Type:       "[]u8",
			Borrow:     BorrowImm,
			Provenance: Provenance{Kind: ProvenanceParam, Root: "xs"},
		}},
		Facts: []Fact{{
			ID:      "borrowed",
			Kind:    FactBorrowedImm,
			ValueID: "v0",
		}},
	}}}
	err := VerifyProgram(prog)
	if err == nil || !strings.Contains(err.Error(), "borrowed_imm requires no_escape fact") {
		t.Fatalf("VerifyProgram error = %v", err)
	}
}

func TestVerifierRejectsDerivedWindowWithoutSource(t *testing.T) {
	prog := &Program{Funcs: []Function{{
		Name: "bad",
		Values: []Value{{
			ID:         "view",
			Kind:       ValueView,
			Type:       "[]u8",
			Borrow:     BorrowImm,
			Escape:     EscapeNoEscape,
			Provenance: Provenance{Kind: ProvenanceParam, Root: "xs"},
		}},
		Facts: []Fact{
			{ID: "window", Kind: FactDerivedWindow, ValueID: "view", Range: "0..1"},
			{ID: "borrowed", Kind: FactBorrowedImm, ValueID: "view"},
			{ID: "no_escape", Kind: FactNoEscape, ValueID: "view"},
		},
	}}}
	err := VerifyProgram(prog)
	if err == nil || !strings.Contains(err.Error(), "derived_window requires source") {
		t.Fatalf("VerifyProgram error = %v", err)
	}
}

func TestVerifierRejectsContradictoryProvenanceFacts(t *testing.T) {
	prog := &Program{Funcs: []Function{{
		Name: "bad",
		Values: []Value{{
			ID:         "v0",
			Type:       "[]u8",
			Provenance: Provenance{Kind: ProvenanceExternal, Root: "ffi"},
		}},
		Facts: []Fact{
			{ID: "known", Kind: FactProvenanceKnown, ValueID: "v0"},
			{ID: "unknown", Kind: FactProvenanceUnknown, ValueID: "v0"},
		},
	}}}
	err := VerifyProgram(prog)
	if err == nil || !strings.Contains(err.Error(), "provenance_known contradicts provenance_unknown") {
		t.Fatalf("VerifyProgram error = %v", err)
	}
}

func TestVerifierRejectsNoHeapAllocationOnAllocationIntent(t *testing.T) {
	prog := &Program{Funcs: []Function{{
		Name: "bad",
		Values: []Value{{
			ID:   "alloc",
			Kind: ValueAllocIntent,
			Type: "[]u8",
			Alloc: &AllocIntent{
				ElementType:         "u8",
				ElementSize:         1,
				LengthExpr:          "n",
				ZeroGuardStatus:     "checked",
				NegativeGuardStatus: "checked",
				OverflowGuardStatus: "checked",
			},
			Provenance: Provenance{Kind: ProvenanceAllocation, Root: "alloc"},
		}},
		Facts: []Fact{{
			ID:      "f0",
			Kind:    FactNoHeapAllocation,
			ValueID: "alloc",
		}},
	}}}
	err := VerifyProgram(prog)
	if err == nil || !strings.Contains(err.Error(), "no_heap_allocation contradicts allocation intent") {
		t.Fatalf("VerifyProgram error = %v", err)
	}
}

func TestVerifierRejectsCopyAllocationWithoutOwnedFact(t *testing.T) {
	prog := &Program{Funcs: []Function{{
		Name: "bad",
		Values: []Value{{
			ID:   "copy",
			Kind: ValueAllocIntent,
			Type: "[]u8",
			Alloc: &AllocIntent{
				ElementType:         "u8",
				ElementSize:         1,
				LengthExpr:          "xs.len",
				ZeroGuardStatus:     "checked",
				NegativeGuardStatus: "checked",
				OverflowGuardStatus: "checked",
				Builtin:             "core.slice_copy_u8",
			},
			Provenance: Provenance{Kind: ProvenanceAllocation, Root: "copy"},
		}},
		Facts: []Fact{{ID: "known", Kind: FactProvenanceKnown, ValueID: "copy"}},
	}}}
	err := VerifyProgram(prog)
	if err == nil || !strings.Contains(err.Error(), "copy allocation intent requires owned fact") {
		t.Fatalf("VerifyProgram error = %v", err)
	}
}

func TestFromCheckedProgramRecordsRawSliceExternalProvenance(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, capability, mem:
    var xs: []u8 = make_u8(1)
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let view: []u8 = core.raw_slice_u8_from_parts(xs.ptr, xs.len, mem)
        return view.len
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}

	var mainFn Function
	for _, candidate := range prog.Funcs {
		if candidate.Name == "main" {
			mainFn = candidate
			break
		}
	}
	if mainFn.Name == "" {
		t.Fatalf("missing PLIR function main")
	}
	var sawRawView bool
	for _, value := range mainFn.Values {
		if value.ID == "view:view" {
			sawRawView = true
			if value.Provenance.Kind != ProvenanceExternal {
				t.Fatalf("raw view provenance = %s, want external", value.Provenance.Kind)
			}
			if value.UnsafeClass != UnsafeUnknown {
				t.Fatalf("raw view unsafe class = %q, want %q", value.UnsafeClass, UnsafeUnknown)
			}
		}
	}
	if !sawRawView {
		t.Fatalf("missing raw view in PLIR values: %#v", mainFn.Values)
	}
	var sawRawSliceOp bool
	for _, op := range mainFn.Ops {
		if op.Kind == OpUnsafe && strings.Contains(op.Note, "external-provenance view") {
			sawRawSliceOp = true
			if op.UnsafeClass != UnsafeUnknown {
				t.Fatalf("raw slice op unsafe class = %q, want %q: %+v", op.UnsafeClass, UnsafeUnknown, op)
			}
		}
	}
	if !sawRawSliceOp {
		t.Fatalf("missing raw slice unsafe operation:\n%s", FormatText(prog))
	}
	if !mainFn.HasFact(FactProvenanceUnknown) {
		t.Fatalf("raw view should record conservative unknown provenance fact: %#v", mainFn.Facts)
	}
	if mainFn.HasFact(FactLenStable) {
		for _, fact := range mainFn.Facts {
			if fact.ValueID == "view:view" && fact.Kind == FactLenStable {
				t.Fatalf("raw view unexpectedly received len_stable fact: %#v", fact)
			}
		}
	}
}

func TestFromCheckedProgramRecordsVerifiedRootRawSliceBoundsEvidence(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let view: []u8 = core.raw_slice_u8_from_parts(p, 4, mem)
        return view.len
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "main")
	view := findPLIRValue(t, fn, "view:view")
	if view.Provenance.Kind == ProvenanceAllocation || view.Provenance.Kind == ProvenanceParam || view.Provenance.Kind == ProvenanceStack {
		t.Fatalf("verified-root raw slice must not become safe provenance: %+v", view)
	}
	if view.UnsafeClass != UnsafeChecked {
		t.Fatalf("verified-root raw slice unsafe class = %q, want %q\n%s", view.UnsafeClass, UnsafeChecked, FormatText(prog))
	}
	assertUnsafeNoteContains(t, fn, "core.raw_slice_u8_from_parts", "raw_slice_bounds", "verified_allocation_root", "base:p", "length_bytes:4")
	for _, fact := range fn.Facts {
		if fact.ValueID == "view:view" && (fact.Kind == FactProvenanceKnown || fact.Kind == FactLenStable || fact.Kind == FactIndexInRange || fact.Kind == FactNoAlias) {
			t.Fatalf("verified-root raw slice gained safe proof fact: %+v\n%s", fact, FormatText(prog))
		}
	}
}

func TestFromCheckedProgramRejectsVerifiedRootRawSliceNegativeLength(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let view: []u8 = core.raw_slice_u8_from_parts(p, 0 - 1, mem)
        return view.len
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "main")
	assertUnsafeNoteContains(t, fn, "core.raw_slice_u8_from_parts", "rejected_negative_length")
	view := findPLIRValue(t, fn, "view:view")
	if view.UnsafeClass != UnsafeChecked {
		t.Fatalf("rejected verified-root raw slice unsafe class = %q, want %q", view.UnsafeClass, UnsafeChecked)
	}
}

func TestFromCheckedProgramRecordsRawSliceElementWidthAndOverflowEvidence(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(64)
        let bytes: []u8 = core.raw_slice_u8_from_parts(p, 8, mem)
        let words: []u16 = core.raw_slice_u16_from_parts(p, 8, mem)
        let ints: []i32 = core.raw_slice_i32_from_parts(p, 8, mem)
        let flags: []bool = core.raw_slice_bool_from_parts(p, 8, mem)
        let overflow: []i32 = core.raw_slice_i32_from_parts(p, 536870912, mem)
        return bytes.len + words.len + ints.len + flags.len + overflow.len
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "main")
	for _, tc := range []struct {
		name        string
		elemSize    string
		lengthBytes string
	}{
		{name: "core.raw_slice_u8_from_parts", elemSize: "elem_size:1", lengthBytes: "length_bytes:8"},
		{name: "core.raw_slice_u16_from_parts", elemSize: "elem_size:2", lengthBytes: "length_bytes:16"},
		{name: "core.raw_slice_i32_from_parts", elemSize: "elem_size:4", lengthBytes: "length_bytes:32"},
		{name: "core.raw_slice_bool_from_parts", elemSize: "elem_size:4", lengthBytes: "length_bytes:32"},
	} {
		assertUnsafeNoteContains(t, fn, tc.name, "raw_slice_bounds", "verified_allocation_root", tc.elemSize, tc.lengthBytes)
	}
	assertUnsafeNoteContains(t, fn, "overflow", "raw_slice_bounds", "rejected_length_overflow", "elem_size:4")
}

func TestFromCheckedProgramRecordsAllocBytesRawBoundsMetadata(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(16)
        let q: ptr = core.ptr_add(p, 4, mem)
        let _: UInt8 = core.store_u8(q, 7, mem)
        return core.load_u8(q, mem)
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}

	var mainFn Function
	for _, candidate := range prog.Funcs {
		if candidate.Name == "main" {
			mainFn = candidate
			break
		}
	}
	if mainFn.Name == "" {
		t.Fatalf("missing PLIR function main")
	}
	var rawAlloc Value
	for _, value := range mainFn.Values {
		if value.ID == "alloc_intent:p" {
			rawAlloc = value
			break
		}
	}
	if rawAlloc.ID == "" || rawAlloc.Alloc == nil {
		t.Fatalf("missing raw alloc_bytes allocation intent: %#v", mainFn.Values)
	}
	if rawAlloc.Alloc.Builtin != "core.alloc_bytes" || rawAlloc.Alloc.ElementType != "raw_bytes" || rawAlloc.Alloc.RawPointerBoundsStatus != "allocation_base_metadata" {
		t.Fatalf("raw allocation intent = %+v, want alloc_bytes raw allocation-base metadata", rawAlloc.Alloc)
	}
	if rawAlloc.UnsafeClass != UnsafeVerifiedRoot {
		t.Fatalf("raw allocation unsafe class = %q, want %q", rawAlloc.UnsafeClass, UnsafeVerifiedRoot)
	}
	var sawDerivedOffset bool
	var sawRawStore bool
	var sawRawLoad bool
	for _, op := range mainFn.Ops {
		if len(op.Outputs) == 1 && op.Outputs[0] == "q" && strings.Contains(op.Note, "derived_allocation_offset") && strings.Contains(op.Note, "base:p") {
			if op.UnsafeClass != UnsafeChecked {
				t.Fatalf("ptr_add unsafe class = %q, want %q: %+v", op.UnsafeClass, UnsafeChecked, op)
			}
			sawDerivedOffset = true
		}
		if op.Kind == OpUnsafe && strings.Contains(op.Note, "core.store_u8 raw memory gateway") {
			if op.UnsafeClass != UnsafeChecked {
				t.Fatalf("store_u8 unsafe class = %q, want %q: %+v", op.UnsafeClass, UnsafeChecked, op)
			}
			sawRawStore = true
		}
		if op.Kind == OpUnsafe && strings.Contains(op.Note, "core.load_u8 raw memory gateway") {
			if op.UnsafeClass != UnsafeChecked {
				t.Fatalf("load_u8 unsafe class = %q, want %q: %+v", op.UnsafeClass, UnsafeChecked, op)
			}
			sawRawLoad = true
		}
	}
	if !sawDerivedOffset {
		t.Fatalf("missing derived raw pointer offset operation:\n%s", FormatText(prog))
	}
	if !sawRawStore || !sawRawLoad {
		t.Fatalf("missing raw load/store unsafe gateway operations store=%v load=%v:\n%s", sawRawStore, sawRawLoad, FormatText(prog))
	}
}

func TestFromCheckedProgramRecordsVerifiedRootRawBoundsRejections(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let neg_base: ptr = core.alloc_bytes(8)
        let neg: ptr = core.ptr_add(neg_base, 0 - 1, mem)
        let neg_read: UInt8 = core.load_u8(neg, mem)
        let upper_base: ptr = core.alloc_bytes(8)
        let upper: ptr = core.ptr_add(upper_base, 8, mem)
        let upper_read: UInt8 = core.load_u8(upper, mem)
        let i32_base: ptr = core.alloc_bytes(8)
        let i32_ptr: ptr = core.ptr_add(i32_base, 5, mem)
        let i32_read: Int = core.load_i32(i32_ptr, mem)
        let ptr_base: ptr = core.alloc_bytes(4)
        let ptr_ptr: ptr = core.ptr_add(ptr_base, 1, mem)
        let ptr_write: ptr = core.store_ptr(ptr_ptr, ptr_base, mem)
        return 0
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "main")
	assertUnsafeNoteContains(t, fn, "neg", "rejected_negative_offset")
	assertUnsafeNoteContains(t, fn, "upper", "rejected_upper_bound")
	assertUnsafeNoteContains(t, fn, "core.load_i32 raw memory gateway", "rejected_access_width_overflow")
	assertUnsafeNoteContains(t, fn, "core.store_ptr raw memory gateway", "rejected_access_width_overflow")
}

func TestFromCheckedProgramKeepsUnknownRawPointerNegativeOffsetConservative(t *testing.T) {
	checked := checkedProgram(t, `
func external(raw: ptr) -> Int
uses capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let q: ptr = core.ptr_add(raw, 0 - 1, mem)
        let xs: []u8 = core.raw_slice_u8_from_parts(q, 1, mem)
        return xs.len
    return 0

func main() -> Int:
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "external")
	assertUnsafeNoteContains(t, fn, "core.ptr_add", "checked_external_unknown", "offset:0 - 1")
	assertUnsafeNoteContains(t, fn, "external-provenance view")
	for _, op := range fn.Ops {
		if op.Kind != OpUnsafe {
			continue
		}
		if strings.Contains(op.Note, "rejected_") || strings.Contains(op.Note, "derived_allocation_offset") {
			t.Fatalf("unknown raw pointer op gained checked-root bounds claim: %+v\n%s", op, FormatText(prog))
		}
	}
}

func TestFromCheckedProgramRejectsNestedNegativePtrAddDelta(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(16)
        let q: ptr = core.ptr_add(p, 8, mem)
        let r: ptr = core.ptr_add(q, 0 - 1, mem)
        let read: UInt8 = core.load_u8(r, mem)
        return 0
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "main")
	assertUnsafeNoteContains(t, fn, "r", "rejected_negative_offset")
	for _, op := range fn.Ops {
		if op.Kind == OpUnsafe && containsString(op.Outputs, "r") && strings.Contains(op.Note, "derived_allocation_offset") {
			t.Fatalf("nested negative ptr_add delta was accepted as derived offset: %+v\n%s", op, FormatText(prog))
		}
	}
}

func TestFromCheckedProgramClearsRawPointerMetadataOnAssignment(t *testing.T) {
	checked := checkedProgram(t, `
func external(raw: ptr) -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        var q: ptr = core.ptr_add(p, 4, mem)
        q = core.ptr_add(p, 0 - 1, mem)
        let neg_read: UInt8 = core.load_u8(q, mem)
        q = core.ptr_add(raw, 0, mem)
        let unknown_read: UInt8 = core.load_u8(q, mem)
        return 0
    return 0

func main() -> Int:
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "external")
	assertUnsafeNoteContains(t, fn, "q", "rejected_negative_offset")
	assertUnsafeNoteContains(t, fn, "core.ptr_add", "checked_external_unknown", "base:raw")
	for _, op := range fn.Ops {
		if op.Kind != OpUnsafe || !strings.Contains(op.Note, "core.load_u8 raw memory gateway") {
			continue
		}
		if strings.Contains(op.Note, "derived_allocation_offset") {
			t.Fatalf("load after reassignment retained stale verified-root metadata: %+v\n%s", op, FormatText(prog))
		}
	}
}

func TestFromCheckedProgramKeepsDynamicRawPointerOffsetConservative(t *testing.T) {
	checked := checkedProgram(t, `
func read_at(n: Int) -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let q: ptr = core.ptr_add(p, n, mem)
        let read: UInt8 = core.load_u8(q, mem)
        return 0
    return 0

func main() -> Int:
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "read_at")
	assertUnsafeNoteContains(t, fn, "core.ptr_add", "checked_external_unknown", "offset:n")
	for _, op := range fn.Ops {
		if op.Kind != OpUnsafe {
			continue
		}
		if strings.Contains(op.Note, "derived_allocation_offset") || strings.Contains(op.Note, "rejected_") {
			t.Fatalf("dynamic raw offset received static bounds claim: %+v\n%s", op, FormatText(prog))
		}
	}
}

func TestFromCheckedProgramRecordsAllocationLengthContract(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, islands, mem:
    var bytes: []u8 = make_u8(0)
    var flags: []bool = make_bool(536870912)
    island(64) as isl:
        var words: []u16 = core.island_make_u16(isl, 3)
        return bytes.len + flags.len + words.len
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}

	mainFn := findPLIRFunction(t, prog, "main")
	bytes := findPLIRAllocValue(t, mainFn, "bytes")
	if bytes.Alloc.ElementType != "u8" || bytes.Alloc.ElementSize != 1 || bytes.Alloc.LengthExpr != "0" {
		t.Fatalf("bytes allocation intent = %+v", bytes.Alloc)
	}
	if bytes.Alloc.ZeroGuardStatus != "valid_empty_no_allocator" ||
		bytes.Alloc.NegativeGuardStatus != "reject_before_allocation" ||
		bytes.Alloc.OverflowGuardStatus != "reject_before_allocation" {
		t.Fatalf("bytes allocation guards = %+v", bytes.Alloc)
	}

	flags := findPLIRAllocValue(t, mainFn, "flags")
	if flags.Alloc.ElementType != "bool" || flags.Alloc.ElementSize != 4 || flags.Alloc.LengthExpr != "536870912" {
		t.Fatalf("flags allocation intent = %+v", flags.Alloc)
	}
	if !flags.Alloc.LengthConstKnown || flags.Alloc.LengthConst != 536870912 {
		t.Fatalf("flags length const = known:%v value:%d", flags.Alloc.LengthConstKnown, flags.Alloc.LengthConst)
	}

	words := findPLIRAllocValue(t, mainFn, "words")
	if words.Alloc.ElementType != "u16" || words.Alloc.ElementSize != 2 || words.Alloc.LengthExpr != "3" {
		t.Fatalf("island words allocation intent = %+v", words.Alloc)
	}
	if words.Provenance.Kind != ProvenanceIsland {
		t.Fatalf("island words provenance = %s, want island", words.Provenance.Kind)
	}
}

func TestFromCheckedProgramRecordsSliceWindowProvenanceAndRange(t *testing.T) {
	checked := checkedProgram(t, `
func sum(xs: []i32) -> Int
uses mem:
    var total = 0
    for x in xs.window(1, 2):
        total = total + x
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(3)
    xs[0] = 1
    xs[1] = 2
    xs[2] = 3
    return sum(xs)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}
	dump := FormatText(prog)
	for _, want := range []string{
		"func sum",
		"fact derived_window",
		"range: xs[1..3]",
		"fact len_stable",
		"fact index_in_range",
	} {
		if !strings.Contains(dump, want) {
			t.Fatalf("PLIR dump missing %q:\n%s", want, dump)
		}
	}
}

func TestFromCheckedProgramRecordsStringWindowProvenanceAndRange(t *testing.T) {
	checked := checkedProgram(t, `
func sum(text: String) -> Int
uses mem:
    var total = 0
    for ch in text.window(1, 3):
        total = total + ch
    return total

func main() -> Int
uses mem:
    let text: String = "abcdef"
    return sum(text)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}
	dump := FormatText(prog)
	for _, want := range []string{
		"func sum",
		"value view:",
		": str",
		"fact derived_window",
		"range: text[1..4]",
		"fact len_stable",
		"fact index_in_range",
	} {
		if !strings.Contains(dump, want) {
			t.Fatalf("PLIR dump missing %q:\n%s", want, dump)
		}
	}
}

func TestFromCheckedProgramRecordsSliceViewByteWidthAndNormalBuildBoundsChecks(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, mem:
    var bytes: []u8 = make_u8(4)
    var words: []u16 = make_u16(4)
    var nums: []i32 = make_i32(4)
    var flags: []bool = make_bool(4)
    let b: []u8 = bytes.window(1, 2)
    let w: []u16 = words.prefix(2)
    let n: []i32 = nums.suffix(1)
    let f: []bool = flags.window(0, 1)
    let s: String = "abcdef".window(1, 3)
    let sp: String = s.prefix(2)
    return b.len + w.len + n.len + f.len + sp.len
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "main")
	tests := []struct {
		output string
		want   []string
	}{
		{output: "view:b", want: []string{"core.slice_window_u8", "elem_width:1", "elem_shift:0", "bounds_check:normal_build"}},
		{output: "view:w", want: []string{"core.slice_prefix_u16", "elem_width:2", "elem_shift:1", "bounds_check:normal_build"}},
		{output: "view:n", want: []string{"core.slice_suffix_i32", "elem_width:4", "elem_shift:2", "bounds_check:normal_build"}},
		{output: "view:f", want: []string{"core.slice_window_bool", "elem_width:4", "elem_shift:2", "bounds_check:normal_build"}},
		{output: "view:s", want: []string{"core.string_window", "elem_width:1", "elem_shift:0", "bounds_check:normal_build"}},
		{output: "view:sp", want: []string{"core.string_prefix", "elem_width:1", "elem_shift:0", "bounds_check:normal_build"}},
	}
	for _, tc := range tests {
		op, ok := findOperationForOutput(fn, tc.output)
		if !ok {
			t.Fatalf("missing slice view operation for %s:\n%s", tc.output, FormatText(prog))
		}
		for _, want := range tc.want {
			if !strings.Contains(op.Note, want) {
				t.Fatalf("operation for %s note %q missing %q", tc.output, op.Note, want)
			}
		}
	}
}

func TestFromCheckedProgramRecordsCopyIntoOverlapAndCapacityContract(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, mem:
    var xs: []u8 = make_u8(4)
    let src: []u8 = xs.window(0, 3)
    var dst: []u8 = xs.window(1, 3)
    return src.copy_into(dst)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "main")
	op, ok := findOperationNoteContaining(fn, "core.slice_copy_into_u8")
	if !ok {
		t.Fatalf("missing copy_into operation:\n%s", FormatText(prog))
	}
	for _, want := range []string{
		"source:src",
		"destination:dst",
		"dest_capacity_check:normal_build",
		"overlap:known_overlap",
	} {
		if !strings.Contains(op.Note, want) {
			t.Fatalf("copy_into note %q missing %q\n%s", op.Note, want, FormatText(prog))
		}
	}
	for _, valueID := range []string{"view:src", "view:dst"} {
		if hasFactForValue(fn, FactNoAlias, valueID) {
			t.Fatalf("copy_into overlap must not create no_alias for %s:\n%s", valueID, FormatText(prog))
		}
	}
}

func TestFromCheckedProgramRecordsBorrowCopyFacts(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(3)
    xs[0] = 1
    xs[1] = 2
    xs[2] = 3
    let view: []i32 = xs.window(1, 2)
    let borrowed: []i32 = view.borrow()
    let copied: []i32 = borrowed.copy()
    return copied.len
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}
	fn := findPLIRFunction(t, prog, "main")
	borrowed := findPLIRValue(t, fn, "view:borrowed")
	if borrowed.Borrow != BorrowImm || borrowed.Escape != EscapeNoEscape {
		t.Fatalf("borrowed value = %+v, want immutable no_escape borrow", borrowed)
	}
	copied := findPLIRAllocValue(t, fn, "copied")
	if copied.Borrow != BorrowNone || copied.Provenance.Kind != ProvenanceAllocation {
		t.Fatalf("copied value = %+v, want owned allocation provenance", copied)
	}
	for _, op := range fn.Ops {
		if op.Kind == OpCall && (op.Note == "core.slice_window_i32" || op.Note == "core.slice_copy_i32") {
			t.Fatalf("known slice builtin was also recorded as unknown call: %#v\n%s", op, FormatText(prog))
		}
	}
	dump := FormatText(prog)
	for _, want := range []string{
		"fact borrowed_imm value: view:borrowed",
		"fact no_escape value: view:borrowed",
		"fact owned value: alloc_intent:copied",
		"fact provenance_known value: alloc_intent:copied",
		"fact derived_window value: view:borrowed",
	} {
		if !strings.Contains(dump, want) {
			t.Fatalf("PLIR dump missing %q:\n%s", want, dump)
		}
	}
}

func TestFromCheckedProgramPreservesIslandViewAndOwnedCopyFacts(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, islands, mem:
    island(64) as isl:
        var xs: []u8 = core.island_make_u8(isl, 4)
        let view: []u8 = xs.window(0, 2)
        let borrowed: []u8 = view.borrow()
        let copied: []u8 = borrowed.copy()
        return copied.len
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}
	fn := findPLIRFunction(t, prog, "main")
	xs := findPLIRAllocValue(t, fn, "xs")
	if xs.Provenance.Kind != ProvenanceIsland {
		t.Fatalf("island allocation provenance = %+v, want island", xs.Provenance)
	}
	view := findPLIRValue(t, fn, "view:view")
	if view.Provenance.Kind != ProvenanceIsland || view.Escape != EscapeNoEscape {
		t.Fatalf("island view = %+v, want island no_escape view", view)
	}
	borrowed := findPLIRValue(t, fn, "view:borrowed")
	if borrowed.Provenance.Kind != ProvenanceIsland || borrowed.Borrow != BorrowImm || borrowed.Escape != EscapeNoEscape {
		t.Fatalf("borrowed island view = %+v, want borrowed island no_escape view", borrowed)
	}
	copied := findPLIRAllocValue(t, fn, "copied")
	if copied.Provenance.Kind != ProvenanceAllocation || copied.Borrow != BorrowNone {
		t.Fatalf("copy from island = %+v, want owned allocation provenance", copied)
	}
	dump := FormatText(prog)
	for _, want := range []string{
		"fact provenance_known value: alloc_intent:xs",
		"fact borrowed_imm value: view:view",
		"fact borrowed_imm value: view:borrowed",
		"fact owned value: alloc_intent:copied",
	} {
		if !strings.Contains(dump, want) {
			t.Fatalf("PLIR dump missing %q:\n%s", want, dump)
		}
	}
}

func TestFromCheckedProgramRecordsIndexStoreFacts(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, io, mem:
    var xs: []u8 = make_u8(2)
    xs[1] = 42
    print(xs)
    return xs[1]
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}
	fn := findPLIRFunction(t, prog, "main")
	var sawStore bool
	var sawPrint bool
	for _, op := range fn.Ops {
		if op.Kind == OpIndexStore {
			sawStore = true
			if len(op.Inputs) != 2 || op.Inputs[0] != "xs" || op.Inputs[1] != "1" {
				t.Fatalf("index store inputs = %#v, want xs/1\n%s", op.Inputs, FormatText(prog))
			}
		}
		if op.Kind == OpPrint {
			sawPrint = true
			if len(op.Inputs) != 1 || op.Inputs[0] != "xs" {
				t.Fatalf("print inputs = %#v, want xs\n%s", op.Inputs, FormatText(prog))
			}
		}
	}
	if !sawStore {
		t.Fatalf("PLIR dump missing index_store:\n%s", FormatText(prog))
	}
	if !sawPrint {
		t.Fatalf("PLIR dump missing print:\n%s", FormatText(prog))
	}
}

func TestFromCheckedProgramRecordsEffectOptimizationFacts(t *testing.T) {
	checked := checkedProgram(t, `
func add(x: Int, y: Int) -> Int:
    return x + y

func log(xs: []u8) -> Int
uses io:
    print(xs)
    return 0

func main() -> Int
uses alloc, io, mem:
    var xs: []u8 = make_u8(1)
    log(xs)
    return add(20, 22)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}
	add := findPLIRFunction(t, prog, "add")
	for _, want := range []FactKind{FactPureCall, FactNoHeapAllocation, FactNoMemWrite, FactNoActorSend, FactNoUnknownEscape} {
		if !add.HasFact(want) {
			t.Fatalf("add facts missing %s: %#v\n%s", want, add.Facts, FormatText(prog))
		}
	}
	log := findPLIRFunction(t, prog, "log")
	for _, want := range []FactKind{FactNoHeapAllocation, FactNoMemWrite, FactNoActorSend} {
		if !log.HasFact(want) {
			t.Fatalf("log facts missing %s: %#v\n%s", want, log.Facts, FormatText(prog))
		}
	}
	for _, forbidden := range []FactKind{FactPureCall, FactNoUnknownEscape} {
		if log.HasFact(forbidden) {
			t.Fatalf("log unexpectedly has %s: %#v\n%s", forbidden, log.Facts, FormatText(prog))
		}
	}
	mainFn := findPLIRFunction(t, prog, "main")
	for _, forbidden := range []FactKind{FactNoHeapAllocation, FactNoMemWrite, FactPureCall} {
		if mainFn.HasFact(forbidden) {
			t.Fatalf("main unexpectedly has %s: %#v\n%s", forbidden, mainFn.Facts, FormatText(prog))
		}
	}
}

func TestBorrowFromSimpleAliasPreservesDerivedWindowFact(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(3)
    xs[0] = 1
    xs[1] = 2
    xs[2] = 3
    let view: []i32 = xs.window(1, 2)
    let alias: []i32 = view
    let borrowed: []i32 = alias.borrow()
    return borrowed.len
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}
	dump := FormatText(prog)
	for _, want := range []string{
		"fact derived_window value: view:alias",
		"fact derived_window value: view:borrowed",
		"fact no_escape value: view:borrowed",
	} {
		if !strings.Contains(dump, want) {
			t.Fatalf("PLIR dump missing %q:\n%s", want, dump)
		}
	}
}

func TestFromCheckedProgramRecordsLocalViewChainForRangeProof(t *testing.T) {
	checked := checkedProgram(t, `
func sum_chain(xs: []i32) -> Int
uses mem:
    let view: []i32 = xs.prefix(4).suffix(1)
    var total = 0
    for x in view:
        total = total + x
    return total

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(4)
    return sum_chain(xs)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "sum_chain")
	dump := FormatText(prog)
	for _, want := range []string{
		"fact derived_window value: view:view range: xs[1..4]",
		"fact index_in_range",
		"proof proof:for-collection:x:",
	} {
		if !strings.Contains(dump, want) {
			t.Fatalf("PLIR dump missing %q:\n%s", want, dump)
		}
	}
	if len(fn.ProofGuards) != 1 || len(fn.ProofUses) != 1 {
		t.Fatalf("view-chain loop proof guards/uses = %d/%d, want 1/1\n%s", len(fn.ProofGuards), len(fn.ProofUses), dump)
	}
	if fn.ProofGuards[0].ID != fn.ProofUses[0].ProofID {
		t.Fatalf("view-chain proof guard/use mismatch: %#v vs %#v", fn.ProofGuards[0], fn.ProofUses[0])
	}
}

func TestFromCheckedProgramComposesViewChainDerivedWindowRange(t *testing.T) {
	checked := checkedProgram(t, `
func chain_range(xs: []i32) -> Int
uses mem:
    let a: []i32 = xs.window(1, 5)
    let b: []i32 = a.prefix(4)
    let c: []i32 = b.suffix(1)
    return c.len

func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(8)
    return chain_range(xs)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	dump := FormatText(prog)
	for _, want := range []string{
		"fact derived_window value: view:a range: xs[1..6]",
		"fact derived_window value: view:b range: xs[1..5]",
		"fact derived_window value: view:c range: xs[2..5]",
	} {
		if !strings.Contains(dump, want) {
			t.Fatalf("PLIR dump missing composed range %q:\n%s", want, dump)
		}
	}
}

func TestFromCheckedProgramDoesNotProveInvalidStringViewLoop(t *testing.T) {
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
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v", err)
	}
	fn := findPLIRFunction(t, prog, "sum_bad")
	for _, fact := range fn.Facts {
		if fact.Kind == FactIndexInRange || fact.Kind == FactDerivedWindow || fact.Kind == FactLenStable {
			t.Fatalf("invalid String view loop received false fact: %#v\n%s", fact, FormatText(prog))
		}
	}
}

func TestFromCheckedProgramDoesNotProveInvalidIntermediateViewChain(t *testing.T) {
	checked := checkedProgram(t, `
func sum_bad_chain() -> Int:
    let view: String = core.string_suffix(core.string_window("abc", 4, 0), 0)
    var total = 0
    for ch in view:
        total = total + ch
    return total

func main() -> Int:
    return 0
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "sum_bad_chain")
	for _, fact := range fn.Facts {
		if fact.Kind == FactIndexInRange || fact.Kind == FactDerivedWindow || fact.Kind == FactLenStable {
			t.Fatalf("invalid intermediate view chain received false fact: %#v\n%s", fact, FormatText(prog))
		}
	}
	if len(fn.ProofGuards) != 0 || len(fn.ProofUses) != 0 {
		t.Fatalf("invalid intermediate view chain received proof guards/uses: %#v %#v\n%s", fn.ProofGuards, fn.ProofUses, FormatText(prog))
	}
}

func TestFromCheckedProgramDoesNotProveRawDerivedViewChain(t *testing.T) {
	checked := checkedProgram(t, `
func sum_raw_chain(xs: []u8) -> Int
uses capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let raw: []u8 = core.raw_slice_u8_from_parts(xs.ptr, xs.len, mem)
        let view: []u8 = raw.prefix(1).suffix(0)
        var total = 0
        for x in view:
            total = total + x
        return total
    return 0

func main() -> Int
uses alloc, capability, mem:
    var xs: []u8 = make_u8(1)
    return sum_raw_chain(xs)
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	fn := findPLIRFunction(t, prog, "sum_raw_chain")
	for _, fact := range fn.Facts {
		if fact.Kind == FactIndexInRange {
			t.Fatalf("raw-derived view chain received range/len proof fact: %#v\n%s", fact, FormatText(prog))
		}
		if fact.Kind == FactLenStable && (strings.HasPrefix(fact.ValueID, "view:") || fact.ValueID == "local:view" || fact.ValueID == "local:raw") {
			t.Fatalf("raw-derived view chain received view len_stable fact: %#v\n%s", fact, FormatText(prog))
		}
	}
	if len(fn.ProofGuards) != 0 || len(fn.ProofUses) != 0 {
		t.Fatalf("raw-derived view chain received proof guards/uses: %#v %#v\n%s", fn.ProofGuards, fn.ProofUses, FormatText(prog))
	}
}

func findPLIRFunction(t *testing.T, prog *Program, name string) Function {
	t.Helper()
	for _, fn := range prog.Funcs {
		if fn.Name == name {
			return fn
		}
	}
	t.Fatalf("missing PLIR function %s: %#v", name, prog.Funcs)
	return Function{}
}

func findPLIRAllocValue(t *testing.T, fn Function, name string) Value {
	t.Helper()
	want := valueID(ValueAllocIntent, name)
	for _, value := range fn.Values {
		if value.ID == want {
			if value.Alloc == nil {
				t.Fatalf("%s has nil allocation intent", want)
			}
			return value
		}
	}
	t.Fatalf("missing PLIR alloc value %s in %s: %#v", want, fn.Name, fn.Values)
	return Value{}
}

func findPLIRValue(t *testing.T, fn Function, id string) Value {
	t.Helper()
	for _, value := range fn.Values {
		if value.ID == id {
			return value
		}
	}
	t.Fatalf("missing PLIR value %s in %s: %#v", id, fn.Name, fn.Values)
	return Value{}
}

func findOperationForOutput(fn Function, output string) (Operation, bool) {
	for _, op := range fn.Ops {
		if containsString(op.Outputs, output) {
			return op, true
		}
	}
	return Operation{}, false
}

func findOperationNoteContaining(fn Function, needle string) (Operation, bool) {
	for _, op := range fn.Ops {
		if strings.Contains(op.Note, needle) {
			return op, true
		}
	}
	return Operation{}, false
}

func assertUnsafeNoteContains(t *testing.T, fn Function, needles ...string) {
	t.Helper()
	for _, op := range fn.Ops {
		if op.Kind != OpUnsafe {
			continue
		}
		haystack := op.Note + " " + strings.Join(op.Inputs, " ") + " " + strings.Join(op.Outputs, " ")
		matches := true
		for _, needle := range needles {
			if !strings.Contains(haystack, needle) {
				matches = false
				break
			}
		}
		if matches {
			return
		}
	}
	t.Fatalf("missing unsafe op note containing %v:\n%s", needles, FormatText(&Program{Funcs: []Function{fn}}))
}
