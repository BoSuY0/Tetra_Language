package plir_test

import (
	"strconv"
	"strings"
	"testing"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/module"
	. "tetra_language/compiler/internal/plir"
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

func checkedFileProgram(t *testing.T, src string) *semantics.CheckedProgram {
	t.Helper()
	file, err := frontend.ParseFile([]byte(src), "p25/hash_table.tetra")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	world := &module.World{
		EntryModule: file.Module,
		Files:       []*frontend.FileAST{file},
		ByModule:    map[string]*frontend.FileAST{file.Module: file},
	}
	checked, err := semantics.CheckWorldOpt(world, semantics.CheckOptions{RequireMain: true})
	if err != nil {
		t.Fatalf("CheckWorld: %v", err)
	}
	return checked
}

func checkedPLIRProgram(t *testing.T, src string) *Program {
	t.Helper()
	prog, err := FromCheckedProgram(checkedProgram(t, src))
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	return prog
}

func checkedFilePLIRProgram(t *testing.T, src string) *Program {
	t.Helper()
	prog, err := FromCheckedProgram(checkedFileProgram(t, src))
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}
	return prog
}

func TestCopyLoopProofUsesAssignedCopyTargetName(t *testing.T) {
	prog := checkedPLIRProgram(t, `
func main() -> Int
uses alloc, mem:
    var xs: []i32 = make_i32(4)
    xs[0] = 10
    xs[1] = 20
    let ys: []i32 = xs.window(1, 2).copy()
    return ys[0]
`)
	var mainFn Function
	for _, fn := range prog.Funcs {
		if fn.Name == "main" {
			mainFn = fn
			break
		}
	}
	if mainFn.Name == "" {
		t.Fatalf("main PLIR function not found: %s", FormatText(prog))
	}
	for _, guard := range mainFn.ProofGuards {
		if strings.HasPrefix(guard.ID, "proof:copy-loop:ys:") {
			return
		}
	}
	t.Fatalf(
		"copy-loop proof guards = %#v, want proof:copy-loop:ys prefix\n%s",
		mainFn.ProofGuards,
		FormatText(prog),
	)
}

func TestFunctionSummaryRecordsBorrowReturnOwnership(t *testing.T) {
	prog := checkedPLIRProgram(t, `
func view_bytes(xs: borrow []u8) -> borrow []u8:
    return xs.borrow()

func main() -> Int:
    return 0
`)
	fn := findFunction(t, prog, "view_bytes")
	if fn.Summary == nil {
		t.Fatalf("view_bytes missing FunctionSummary")
	}
	if got := fn.Summary.ReturnOwnership; got != "borrow" {
		t.Fatalf("FunctionSummary.ReturnOwnership = %q, want borrow", got)
	}
	if got := fn.Summary.ReturnRegionSummary[""]; got != 0 {
		t.Fatalf(
			"FunctionSummary.ReturnRegionSummary = %#v, want return from param 0",
			fn.Summary.ReturnRegionSummary,
		)
	}
}

func TestFunctionSummaryKeepsOwnedSliceReturnOwnershipEmpty(t *testing.T) {
	prog := checkedPLIRProgram(t, `
func owned_bytes() -> []u8
uses alloc, mem:
    return make_u8(1)

func main() -> Int:
    return 0
`)
	fn := findFunction(t, prog, "owned_bytes")
	if fn.Summary == nil {
		t.Fatalf("owned_bytes missing FunctionSummary")
	}
	if got := fn.Summary.ReturnOwnership; got != "" {
		t.Fatalf("FunctionSummary.ReturnOwnership = %q, want empty ownership", got)
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
	for _, fact := range []FactKind{
		FactProvenanceKnown,
		FactLenStable,
		FactIndexInRange,
		FactRegionAlive,
		FactBorrowedImm,
	} {
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
		if fact.Kind == FactAligned && fact.ValueID == "alloc_intent:xs" &&
			fact.Region == "island:isl" {
			sawAlignedFact = true
		}
		if fact.ValueID == "alloc_intent:xs" && fact.IslandID == "island:isl" && fact.Epoch == 1 &&
			fact.BaseID == "alloc_intent:xs" {
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
		if fact.ValueID == "alloc_intent:xs" && fact.IslandID == "island:isl" && fact.Epoch == 2 &&
			fact.BaseID == "alloc_intent:xs" {
			foundResetAllocation = true
		}
		if fact.Kind != FactIslandEpochAdvanced {
			continue
		}
		found = true
		if fact.IslandID != "island:isl" || fact.Epoch != 2 || fact.BaseID == "" {
			t.Fatalf(
				"island reset fact = %+v, want island:isl epoch 2 with base_id\n%s",
				fact,
				FormatText(prog),
			)
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
	if err == nil || !strings.Contains(err.Error(), "moved") ||
		!strings.Contains(err.Error(), "borrowed") {
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
		Ops: []Operation{
			{ID: "return_xs", Kind: OpReturn, Source: "math.tetra:3:5", Inputs: []string{"xs"}},
		},
		Facts: []Fact{
			{
				ID:      "prov_xs",
				Kind:    FactProvenanceKnown,
				ValueID: "param:xs",
				Source:  "math.tetra:2:17",
				Reason:  "parameter provenance",
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "summary completeness") ||
		!strings.Contains(err.Error(), "return_region_summary") {
		t.Fatalf(
			"VerifyFunction error = %v, want borrowed return summary completeness rejection",
			err,
		)
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
		Ops: []Operation{
			{
				ID:     "return_msg",
				Kind:   OpReturn,
				Source: "resources.tetra:17:5",
				Inputs: []string{"msg"},
			},
		},
	})
	if err != nil {
		t.Fatalf(
			"VerifyFunction rejected value return from resources module as borrowed region return: %v",
			err,
		)
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
		Ops: []Operation{
			{
				ID:     "return_region",
				Kind:   OpReturn,
				Source: "memory_island_alias_region_service.tetra:8:5",
				Inputs: []string{"region"},
			},
		},
	})
	if err != nil {
		t.Fatalf(
			"VerifyFunction rejected owned island resource return as borrowed region return: %v",
			err,
		)
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
		t.Fatalf(
			"guard block %s should dominate use block %s in %+v",
			guard.Block,
			use.Block,
			fn.Dominators,
		)
	}
	if len(fn.RangeFacts) != 1 {
		t.Fatalf("range facts = %#v, want one while range fact", fn.RangeFacts)
	}
	if fn.RangeFacts[0].ProofID != guard.ID || fn.RangeFacts[0].Reason != "while loop range proof" {
		t.Fatalf("range fact = %+v, guard = %+v", fn.RangeFacts[0], guard)
	}
	if !containsString(fn.RangeFacts[0].Derivation, "non_negative") ||
		!containsString(fn.RangeFacts[0].Derivation, "less_than_len") {
		t.Fatalf(
			"range derivation = %#v, want non_negative and less_than_len",
			fn.RangeFacts[0].Derivation,
		)
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

func TestP50HashTableLookupRecordsCallBoundaryProofUses(t *testing.T) {
	prog := checkedFilePLIRProgram(t, `
module p25.hash_table

func lookup(keys: []i32, values: []i32, n: Int, key: Int) -> Int
uses mem:
    var i: Int = 0
    while i < n:
        if keys[i] == key:
            return values[i]
        i = i + 1
    return 0

func main() -> Int
uses alloc, mem:
    let n: Int = 256
    var keys: []i32 = core.make_i32(n)
    var values: []i32 = core.make_i32(n)
    return lookup(keys, values, n, 7)
`)

	fn := findFunction(t, prog, "p25.hash_table.lookup")
	termsByBase := map[string]ProofTerm{}
	for _, term := range fn.ProofTerms {
		if strings.HasPrefix(term.ID, "proof:call-boundary:i:") {
			termsByBase[term.SubjectBaseID] = term
		}
	}
	if len(termsByBase) != 2 {
		t.Fatalf(
			"lookup should have call-boundary proof terms for keys and values, got %#v\n%s",
			fn.ProofTerms,
			FormatText(prog),
		)
	}
	for _, base := range []string{"keys", "values"} {
		term := termsByBase[base]
		if term.SubjectBaseID != base ||
			term.IndexValueID != "local:i" ||
			term.Operation != "index_load" ||
			term.Range != "i in [0, "+base+".len)" ||
			!containsString(term.FactsUsed, "call_boundary_length") {
			t.Fatalf("call-boundary proof term for %s = %+v", base, term)
		}

		guard, ok := proofGuardForID(fn, term.ID)
		if !ok {
			t.Fatalf(
				"missing call-boundary proof guard for %s/%s: %#v",
				base,
				term.ID,
				fn.ProofGuards,
			)
		}
		if guard.Kind != "range" ||
			!strings.Contains(guard.Condition, "i < n") ||
			!strings.Contains(guard.Condition, "n <= "+base+".len") {
			t.Fatalf("call-boundary proof guard for %s = %+v", base, guard)
		}

		use, ok := proofUseForID(fn, term.ID)
		if !ok {
			t.Fatalf("missing call-boundary proof use for %s/%s: %#v", base, term.ID, fn.ProofUses)
		}
		if !Dominates(fn, guard.Block, use.Block) {
			t.Fatalf(
				"call-boundary guard block %s should dominate use block %s in %+v",
				guard.Block,
				use.Block,
				fn.Dominators,
			)
		}
		op, ok := operationForID(fn, use.OpID)
		if !ok || op.Kind != OpIndexLoad || len(op.Inputs) < 2 || op.Inputs[0] != base ||
			op.Inputs[1] != "i" {
			t.Fatalf(
				"call-boundary proof use for %s should target lookup index_load, use=%+v op=%+v",
				base,
				use,
				op,
			)
		}

		rangeFact, ok := rangeFactForProofID(fn, term.ID)
		if !ok {
			t.Fatalf(
				"missing call-boundary range fact for %s/%s: %#v",
				base,
				term.ID,
				fn.RangeFacts,
			)
		}
		if rangeFact.Value != "local:i" ||
			rangeFact.Lower != (Bound{Kind: BoundConst, Const: 0}) ||
			rangeFact.Upper != (Bound{Kind: BoundSymbol, Symbol: base + ".len"}) ||
			!rangeFact.InclusiveLower ||
			rangeFact.InclusiveUpper ||
			!containsString(rangeFact.Derivation, "call_boundary_length") {
			t.Fatalf("call-boundary range fact for %s = %+v", base, rangeFact)
		}
	}
}

func TestP50HashTableLookupRejectsUnsafeCallBoundaryProofUses(t *testing.T) {
	prog := checkedFilePLIRProgram(t, `
module p25.hash_table

func lookup(keys: []i32, values: []i32, n: Int, key: Int) -> Int
uses mem:
    var i: Int = 0
    while i < n:
        if keys[i] == key:
            return values[i]
        i = i + 1
    return 0

func main() -> Int
uses alloc, mem:
    let n: Int = 256
    let short: Int = 128
    var keys: []i32 = core.make_i32(n)
    var values: []i32 = core.make_i32(short)
    return lookup(keys, values, n, 7)
`)

	fn := findFunction(t, prog, "p25.hash_table.lookup")
	for _, term := range fn.ProofTerms {
		if strings.HasPrefix(term.ID, "proof:call-boundary:") {
			t.Fatalf(
				"unsafe lookup call unexpectedly received call-boundary proof term: %+v\n%s",
				term,
				FormatText(prog),
			)
		}
	}
	for _, use := range fn.ProofUses {
		if strings.HasPrefix(use.ProofID, "proof:call-boundary:") {
			t.Fatalf(
				"unsafe lookup call unexpectedly received call-boundary proof use: %+v\n%s",
				use,
				FormatText(prog),
			)
		}
	}
}

func TestFromCheckedProgramRecordsJsonWriteMessageObjectHelperSummaryProof(t *testing.T) {
	prog := checkedFilePLIRProgram(t, jsonParseStringifyPLIRHelperSummarySource)
	fn := findFunction(t, prog, "p25.json_parse_stringify.write_message_object")
	termsByIndex := map[string]ProofTerm{}
	for _, term := range fn.ProofTerms {
		if strings.HasPrefix(term.ID, "proof:helper-summary:") {
			termsByIndex[term.IndexValueID] = term
		}
	}
	if len(termsByIndex) != 27 {
		t.Fatalf(
			"helper-summary proof terms = %d, want 27; terms=%#v\n%s",
			len(termsByIndex),
			fn.ProofTerms,
			FormatText(prog),
		)
	}
	for i := 0; i < 27; i++ {
		index := strconv.Itoa(i)
		indexValueID := "local:" + index
		term, ok := termsByIndex[indexValueID]
		if !ok {
			t.Fatalf(
				"missing helper-summary proof term for index %s; terms=%#v",
				index,
				fn.ProofTerms,
			)
		}
		if term.Kind != "bounds_check" ||
			term.SubjectBaseID != "dst" ||
			term.Operation != "index_store" ||
			term.Range != index+" in [0, dst.len)" ||
			!containsString(term.FactsUsed, "helper_summary_local_call") ||
			!containsString(term.FactsUsed, "caller_known_length:128") {
			t.Fatalf("helper-summary proof term for index %s = %+v", index, term)
		}
		guard, ok := proofGuardForID(fn, term.ID)
		if !ok {
			t.Fatalf(
				"missing helper-summary proof guard for %s/%s: %#v",
				index,
				term.ID,
				fn.ProofGuards,
			)
		}
		if guard.Kind != "range" ||
			!strings.Contains(
				guard.Condition,
				"main -> p25.json_parse_stringify.write_message_object",
			) ||
			!strings.Contains(guard.Condition, index+" < 128") ||
			!strings.Contains(guard.Condition, "dst.len >= 128") {
			t.Fatalf("helper-summary proof guard for index %s = %+v", index, guard)
		}
		use, ok := proofUseForID(fn, term.ID)
		if !ok {
			t.Fatalf(
				"missing helper-summary proof use for %s/%s: %#v",
				index,
				term.ID,
				fn.ProofUses,
			)
		}
		if !Dominates(fn, guard.Block, use.Block) {
			t.Fatalf(
				"helper-summary guard block %s should dominate use block %s in %+v",
				guard.Block,
				use.Block,
				fn.Dominators,
			)
		}
		op, ok := operationForID(fn, use.OpID)
		if !ok || op.Kind != OpIndexStore || len(op.Inputs) < 2 || op.Inputs[0] != "dst" ||
			op.Inputs[1] != index {
			t.Fatalf(
				"helper-summary proof use for index %s should target dst index_store, use=%+v op=%+v",
				index,
				use,
				op,
			)
		}
		rangeFact, ok := rangeFactForProofID(fn, term.ID)
		if !ok {
			t.Fatalf(
				"missing helper-summary range fact for %s/%s: %#v",
				index,
				term.ID,
				fn.RangeFacts,
			)
		}
		if rangeFact.Value != indexValueID ||
			rangeFact.Lower != (Bound{Kind: BoundConst, Const: 0}) ||
			rangeFact.Upper != (Bound{Kind: BoundSymbol, Symbol: "dst.len"}) ||
			!rangeFact.InclusiveLower ||
			rangeFact.InclusiveUpper ||
			!containsString(rangeFact.Derivation, "helper_summary_local_call") {
			t.Fatalf("helper-summary range fact for index %s = %+v", index, rangeFact)
		}
	}
}

func TestFromCheckedProgramRecordsHTTPMultiHelperSummaryProofs(t *testing.T) {
	prog := checkedFilePLIRProgram(t, httpPlaintextJSONPLIRHelperSummarySource)
	tests := []struct {
		fnName string
		want   int
	}{
		{fnName: "p25.http_plaintext_json.write_plaintext_response", want: 24},
		{fnName: "p25.http_plaintext_json.write_json_response", want: 21},
	}
	for _, tt := range tests {
		t.Run(tt.fnName, func(t *testing.T) {
			fn := findFunction(t, prog, tt.fnName)
			termsByIndex := map[string]ProofTerm{}
			for _, term := range fn.ProofTerms {
				if strings.HasPrefix(term.ID, "proof:helper-offset:") {
					t.Fatalf(
						"%s accidentally received helper-offset proof term: %+v\n%s",
						tt.fnName,
						term,
						FormatText(prog),
					)
				}
				if strings.HasPrefix(term.ID, "proof:helper-summary:") {
					termsByIndex[term.IndexValueID] = term
				}
			}
			if len(termsByIndex) != tt.want {
				t.Fatalf(
					"%s helper-summary proof terms = %d, want %d; terms=%#v\n%s",
					tt.fnName,
					len(termsByIndex),
					tt.want,
					fn.ProofTerms,
					FormatText(prog),
				)
			}
			for i := 0; i < tt.want; i++ {
				index := strconv.Itoa(i)
				indexValueID := "local:" + index
				term, ok := termsByIndex[indexValueID]
				if !ok {
					t.Fatalf(
						"%s missing helper-summary proof term for index %s; terms=%#v",
						tt.fnName,
						index,
						fn.ProofTerms,
					)
				}
				if term.Kind != "bounds_check" ||
					term.SubjectBaseID != "dst" ||
					term.Operation != "index_store" ||
					term.Range != index+" in [0, dst.len)" ||
					!containsString(term.FactsUsed, "helper_summary_local_call") ||
					!containsString(term.FactsUsed, "caller_known_length:192") {
					t.Fatalf(
						"%s helper-summary proof term for index %s = %+v",
						tt.fnName,
						index,
						term,
					)
				}
				guard, ok := proofGuardForID(fn, term.ID)
				if !ok {
					t.Fatalf(
						"%s missing helper-summary proof guard for %s/%s: %#v",
						tt.fnName,
						index,
						term.ID,
						fn.ProofGuards,
					)
				}
				if guard.Kind != "range" ||
					!strings.Contains(guard.Condition, "main -> "+tt.fnName) ||
					!strings.Contains(guard.Condition, index+" < 192") ||
					!strings.Contains(guard.Condition, "dst.len >= 192") {
					t.Fatalf(
						"%s helper-summary proof guard for index %s = %+v",
						tt.fnName,
						index,
						guard,
					)
				}
				use, ok := proofUseForID(fn, term.ID)
				if !ok {
					t.Fatalf(
						"%s missing helper-summary proof use for %s/%s: %#v",
						tt.fnName,
						index,
						term.ID,
						fn.ProofUses,
					)
				}
				if !Dominates(fn, guard.Block, use.Block) {
					t.Fatalf(
						"%s helper-summary guard block %s should dominate use block %s in %+v",
						tt.fnName,
						guard.Block,
						use.Block,
						fn.Dominators,
					)
				}
				op, ok := operationForID(fn, use.OpID)
				if !ok || op.Kind != OpIndexStore || len(op.Inputs) < 2 || op.Inputs[0] != "dst" ||
					op.Inputs[1] != index {
					t.Fatalf(
						"%s helper-summary proof use for index %s should target dst index_store, use=%+v op=%+v",
						tt.fnName,
						index,
						use,
						op,
					)
				}
				rangeFact, ok := rangeFactForProofID(fn, term.ID)
				if !ok {
					t.Fatalf(
						"%s missing helper-summary range fact for %s/%s: %#v",
						tt.fnName,
						index,
						term.ID,
						fn.RangeFacts,
					)
				}
				if rangeFact.Value != indexValueID ||
					rangeFact.Lower != (Bound{Kind: BoundConst, Const: 0}) ||
					rangeFact.Upper != (Bound{Kind: BoundSymbol, Symbol: "dst.len"}) ||
					!rangeFact.InclusiveLower ||
					rangeFact.InclusiveUpper ||
					!containsString(rangeFact.Derivation, "helper_summary_local_call") {
					t.Fatalf(
						"%s helper-summary range fact for index %s = %+v",
						tt.fnName,
						index,
						rangeFact,
					)
				}
			}
			for _, use := range fn.ProofUses {
				if strings.HasPrefix(use.ProofID, "proof:helper-offset:") {
					t.Fatalf(
						"%s accidentally received helper-offset proof use: %+v\n%s",
						tt.fnName,
						use,
						FormatText(prog),
					)
				}
			}
		})
	}
}

func TestFromCheckedProgramRejectsUnsafeJsonHelperSummaryProofUses(t *testing.T) {
	prog := checkedFilePLIRProgram(t, `
module p25.json_parse_stringify

func write_message_object(dst: inout []u8) -> Int
uses mem:
    unsafe:
        dst[0] = 125
    return 1

func main() -> Int
uses alloc, mem:
    var buf: []u8 = core.make_u8(128)
    return write_message_object(buf)
`)
	fn := findFunction(t, prog, "p25.json_parse_stringify.write_message_object")
	for _, term := range fn.ProofTerms {
		if strings.HasPrefix(term.ID, "proof:helper-summary:") {
			t.Fatalf(
				"unsafe helper unexpectedly received helper-summary proof term: %+v\n%s",
				term,
				FormatText(prog),
			)
		}
	}
	for _, use := range fn.ProofUses {
		if strings.HasPrefix(use.ProofID, "proof:helper-summary:") {
			t.Fatalf(
				"unsafe helper unexpectedly received helper-summary proof use: %+v\n%s",
				use,
				FormatText(prog),
			)
		}
	}
}

func TestFromCheckedProgramRecordsPostgreSQLHelperOffsetProofs(t *testing.T) {
	prog := checkedFilePLIRProgram(t, postgresqlHelperOffsetPLIRSource)

	tests := []struct {
		fnName    string
		base      string
		operation string
		indexes   []string
	}{
		{
			fnName:    "p25.postgresql_single_multiple_update.frame_type_at",
			base:      "src",
			operation: "index_load",
			indexes:   []string{"0"},
		},
		{
			fnName:    "p25.postgresql_single_multiple_update.write_i32_be_at",
			base:      "dst",
			operation: "index_store",
			indexes:   []string{"1", "2", "3", "4"},
		},
		{
			fnName:    "p25.postgresql_single_multiple_update.write_i16_be_at",
			base:      "dst",
			operation: "index_store",
			indexes:   []string{"5", "6"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.fnName, func(t *testing.T) {
			fn := findFunction(t, prog, tt.fnName)
			termsByRange := map[string]ProofTerm{}
			for _, term := range fn.ProofTerms {
				if strings.HasPrefix(term.ID, "proof:helper-offset:") {
					termsByRange[term.Range] = term
				}
			}
			if len(termsByRange) != len(tt.indexes) {
				t.Fatalf(
					"%s helper-offset proof terms = %d, want %d; terms=%#v\n%s",
					tt.fnName,
					len(termsByRange),
					len(tt.indexes),
					fn.ProofTerms,
					FormatText(prog),
				)
			}
			for _, index := range tt.indexes {
				wantRange := index + " in [0, " + tt.base + ".len)"
				term, ok := termsByRange[wantRange]
				if !ok {
					t.Fatalf(
						"missing helper-offset proof term for range %s; terms=%#v",
						wantRange,
						fn.ProofTerms,
					)
				}
				if term.Kind != "bounds_check" ||
					term.SubjectBaseID != tt.base ||
					term.Operation != tt.operation ||
					!containsString(term.FactsUsed, "helper_offset_local_call") ||
					!containsString(term.FactsUsed, "caller_known_length:64") {
					t.Fatalf("helper-offset proof term for range %s = %+v", wantRange, term)
				}
				guard, ok := proofGuardForID(fn, term.ID)
				if !ok {
					t.Fatalf(
						"missing helper-offset proof guard for %s/%s: %#v",
						wantRange,
						term.ID,
						fn.ProofGuards,
					)
				}
				if guard.Kind != "range" ||
					!strings.Contains(guard.Condition, "main -> "+tt.fnName) ||
					!strings.Contains(guard.Condition, index+" < 64") ||
					!strings.Contains(guard.Condition, tt.base+".len >= 64") {
					t.Fatalf("helper-offset proof guard for range %s = %+v", wantRange, guard)
				}
				use, ok := proofUseForID(fn, term.ID)
				if !ok {
					t.Fatalf(
						"missing helper-offset proof use for %s/%s: %#v",
						wantRange,
						term.ID,
						fn.ProofUses,
					)
				}
				if !Dominates(fn, guard.Block, use.Block) {
					t.Fatalf(
						"helper-offset guard block %s should dominate use block %s in %+v",
						guard.Block,
						use.Block,
						fn.Dominators,
					)
				}
				op, ok := operationForID(fn, use.OpID)
				if !ok || string(op.Kind) != tt.operation || len(op.Inputs) < 1 ||
					op.Inputs[0] != tt.base {
					t.Fatalf(
						"helper-offset proof use for range %s should target %s %s, use=%+v op=%+v",
						wantRange,
						tt.base,
						tt.operation,
						use,
						op,
					)
				}
				rangeFact, ok := rangeFactForProofID(fn, term.ID)
				if !ok {
					t.Fatalf(
						"missing helper-offset range fact for %s/%s: %#v",
						wantRange,
						term.ID,
						fn.RangeFacts,
					)
				}
				if rangeFact.Lower != (Bound{Kind: BoundConst, Const: 0}) ||
					rangeFact.Upper != (Bound{Kind: BoundSymbol, Symbol: tt.base + ".len"}) ||
					!rangeFact.InclusiveLower ||
					rangeFact.InclusiveUpper ||
					!containsString(rangeFact.Derivation, "helper_offset_local_call") {
					t.Fatalf("helper-offset range fact for range %s = %+v", wantRange, rangeFact)
				}
			}
		})
	}
}

const postgresqlHelperOffsetPLIRSource = `
module p25.postgresql_single_multiple_update

func frame_data_row() -> Int:
    return 68

func frame_payload_start(offset: Int) -> Int:
    return offset + 5

func frame_type_at(src: []u8, offset: Int) -> Int
uses mem:
    return src[offset]

func write_i32_be_at(dst: inout []u8, start: Int, value: Int) -> Int
uses mem:
    dst[start] = (value / 16777216) % 256
    dst[start + 1] = (value / 65536) % 256
    dst[start + 2] = (value / 256) % 256
    dst[start + 3] = value % 256
    return start + 4

func write_i16_be_at(dst: inout []u8, start: Int, value: Int) -> Int
uses mem:
    dst[start] = (value / 256) % 256
    dst[start + 1] = value % 256
    return start + 2

func main() -> Int
uses alloc, mem:
    var frame: []u8 = core.make_u8(64)
    var i: Int = 0
    var total: Int = 0
    while i < 2048:
        frame[0] = frame_data_row()
        var pos: Int = write_i32_be_at(frame, 1, 12)
        pos = write_i16_be_at(frame, pos, 2)
        total = total + frame_type_at(frame, 0) + frame_payload_start(0)
        i = i + 1
    if total > 0:
        return 0
    return 1
`

const jsonParseStringifyPLIRHelperSummarySource = `
module p25.json_parse_stringify

func write_message_object(dst: inout []u8) -> Int
uses mem:
    dst[0] = 123
    dst[1] = 34
    dst[2] = 109
    dst[3] = 101
    dst[4] = 115
    dst[5] = 115
    dst[6] = 97
    dst[7] = 103
    dst[8] = 101
    dst[9] = 34
    dst[10] = 58
    dst[11] = 34
    dst[12] = 72
    dst[13] = 101
    dst[14] = 108
    dst[15] = 108
    dst[16] = 111
    dst[17] = 44
    dst[18] = 32
    dst[19] = 87
    dst[20] = 111
    dst[21] = 114
    dst[22] = 108
    dst[23] = 100
    dst[24] = 33
    dst[25] = 34
    dst[26] = 125
    return 27

func main() -> Int
uses alloc, mem:
    var buf: []u8 = core.make_u8(128)
    var i: Int = 0
    var total: Int = 0
    while i < 2048:
        total = total + write_message_object(buf)
        i = i + 1
    if total == 55296:
        return 0
    return 1
`

const httpPlaintextJSONPLIRHelperSummarySource = `
module p25.http_plaintext_json

func write_plaintext_response(dst: inout []u8) -> Int
uses mem:
    dst[0] = 72
    dst[1] = 84
    dst[2] = 84
    dst[3] = 80
    dst[4] = 47
    dst[5] = 49
    dst[6] = 46
    dst[7] = 49
    dst[8] = 32
    dst[9] = 50
    dst[10] = 48
    dst[11] = 48
    dst[12] = 32
    dst[13] = 79
    dst[14] = 75
    dst[15] = 13
    dst[16] = 10
    dst[17] = 13
    dst[18] = 10
    dst[19] = 72
    dst[20] = 101
    dst[21] = 108
    dst[22] = 108
    dst[23] = 111
    return 24

func write_json_response(dst: inout []u8) -> Int
uses mem:
    dst[0] = 72
    dst[1] = 84
    dst[2] = 84
    dst[3] = 80
    dst[4] = 47
    dst[5] = 49
    dst[6] = 46
    dst[7] = 49
    dst[8] = 32
    dst[9] = 50
    dst[10] = 48
    dst[11] = 48
    dst[12] = 32
    dst[13] = 79
    dst[14] = 75
    dst[15] = 13
    dst[16] = 10
    dst[17] = 13
    dst[18] = 10
    dst[19] = 123
    dst[20] = 125
    return 21

func main() -> Int
uses alloc, mem:
    var plain: []u8 = core.make_u8(192)
    var json_buf: []u8 = core.make_u8(192)
    var i: Int = 0
    var total: Int = 0
    while i < 1024:
        total = total + write_plaintext_response(plain)
        total = total + write_json_response(json_buf)
        i = i + 1
    if total > 0:
        return 0
    return 1
`

func TestP50UnrelatedPublicHelperRejectsCallBoundaryProofUses(t *testing.T) {
	prog := checkedFilePLIRProgram(t, `
module p25.hash_table

pub func probe(keys: []i32, values: []i32, n: Int, key: Int) -> Int
uses mem:
    var i: Int = 0
    while i < n:
        if keys[i] == key:
            return values[i]
        i = i + 1
    return 0

func main() -> Int
uses alloc, mem:
    let n: Int = 256
    var keys: []i32 = core.make_i32(n)
    var values: []i32 = core.make_i32(n)
    return probe(keys, values, n, 7)
`)

	fn := findFunction(t, prog, "p25.hash_table.probe")
	for _, term := range fn.ProofTerms {
		if strings.HasPrefix(term.ID, "proof:call-boundary:") {
			t.Fatalf(
				"unrelated public helper unexpectedly received call-boundary proof term: %+v\n%s",
				term,
				FormatText(prog),
			)
		}
	}
	for _, use := range fn.ProofUses {
		if strings.HasPrefix(use.ProofID, "proof:call-boundary:") {
			t.Fatalf(
				"unrelated public helper unexpectedly received call-boundary proof use: %+v\n%s",
				use,
				FormatText(prog),
			)
		}
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
		t.Fatalf(
			"base reassignment should invalidate while proof uses: %#v\n%s",
			fn.ProofUses,
			FormatText(prog),
		)
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
		t.Fatalf(
			"inout call should invalidate while proof uses: %#v\n%s",
			fn.ProofUses,
			FormatText(prog),
		)
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
		t.Fatalf(
			"callback inout call should invalidate while proof uses: %#v\n%s",
			fn.ProofUses,
			FormatText(prog),
		)
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
		t.Fatalf(
			"range derivation = %#v, want less_equal_len_minus_one",
			fn.RangeFacts[0].Derivation,
		)
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
	if len(fn.RangeFacts) != 1 || fn.RangeFacts[0].Upper.Symbol != "xs.len" ||
		fn.RangeFacts[0].InclusiveUpper {
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
	if len(fn.RangeFacts) != 1 || fn.RangeFacts[0].Upper.Symbol != "xs.len" ||
		fn.RangeFacts[0].InclusiveUpper {
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
	if len(fn.ProofGuards) != 2 || fn.ProofGuards[0].Condition != "i < n" ||
		fn.ProofGuards[1].Condition != "i < n" {
		t.Fatalf(
			"allocation length alias proof guards = %#v, want two i < n while guards",
			fn.ProofGuards,
		)
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

func TestFromCheckedProgramRecordsModuloAllocationLengthAliasProof(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, mem:
    let n: Int = 4096
    var xs: []i32 = core.make_i32(n)
    var i: Int = 0
    while i < n:
        xs[i] = i % 97
        i = i + 1
    var total: Int = 0
    i = 0
    while i < 200000:
        let idx: Int = (i * 17) % n
        total = total + xs[idx]
        i = i + 1
    if total >= 0:
        return 0
    return 1
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}

	fn := findFunction(t, prog, "main")
	var guard ProofGuard
	for _, candidate := range fn.ProofGuards {
		if strings.HasPrefix(candidate.ID, "proof:modulo:") {
			guard = candidate
			break
		}
	}
	if guard.ID == "" {
		t.Fatalf("missing modulo proof guard:\n%s", FormatText(prog))
	}
	if guard.Condition != "idx = i * 17 % n" || guard.Kind != "range" {
		t.Fatalf("modulo proof guard = %+v", guard)
	}

	var use ProofUse
	for _, candidate := range fn.ProofUses {
		if candidate.ProofID == guard.ID {
			use = candidate
			break
		}
	}
	if use.ProofID == "" {
		t.Fatalf("missing modulo proof use for %q: %#v", guard.ID, fn.ProofUses)
	}
	if !Dominates(fn, guard.Block, use.Block) {
		t.Fatalf(
			"modulo guard block %s should dominate use block %s in %+v",
			guard.Block,
			use.Block,
			fn.Dominators,
		)
	}

	var rangeFact RangeFact
	for _, candidate := range fn.RangeFacts {
		if candidate.ProofID == guard.ID {
			rangeFact = candidate
			break
		}
	}
	if rangeFact.ProofID == "" {
		t.Fatalf("missing modulo range fact for %q: %#v", guard.ID, fn.RangeFacts)
	}
	if rangeFact.Value != "local:idx" ||
		rangeFact.Lower != (Bound{Kind: BoundConst, Const: 0}) ||
		rangeFact.Upper != (Bound{Kind: BoundSymbol, Symbol: "xs.len"}) ||
		!rangeFact.InclusiveLower ||
		rangeFact.InclusiveUpper ||
		!containsString(rangeFact.Derivation, "modulo_allocation_length_alias") {
		t.Fatalf("modulo range fact = %+v", rangeFact)
	}

	var term ProofTerm
	for _, candidate := range fn.ProofTerms {
		if candidate.ID == guard.ID {
			term = candidate
			break
		}
	}
	if term.ID == "" {
		t.Fatalf("missing modulo proof term for %q: %#v", guard.ID, fn.ProofTerms)
	}
	if term.SubjectBaseID != "xs" ||
		term.IndexValueID != "local:idx" ||
		term.Operation != "index_load" ||
		term.Range != "idx in [0, xs.len)" ||
		!containsString(term.FactsUsed, "modulo_allocation_length_alias") {
		t.Fatalf("modulo proof term = %+v", term)
	}
}

func TestFromCheckedProgramRecordsMatrixModuloConstInlineProof(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, mem:
    var c: []i32 = core.make_i32(9)
    var r: Int = 0
    var checksum: Int = 0
    while r < 2000:
        checksum = checksum + c[r % 9]
        r = r + 1
    return checksum
`)

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}

	fn := findFunction(t, prog, "main")
	var guard ProofGuard
	for _, candidate := range fn.ProofGuards {
		if strings.HasPrefix(candidate.ID, "proof:modulo:") {
			guard = candidate
			break
		}
	}
	if guard.ID == "" {
		t.Fatalf("missing matrix modulo const proof guard:\n%s", FormatText(prog))
	}
	if guard.Kind != "range" || !strings.Contains(guard.Condition, "r % 9") {
		t.Fatalf("matrix modulo const proof guard = %+v", guard)
	}

	var use ProofUse
	for _, candidate := range fn.ProofUses {
		if candidate.ProofID == guard.ID {
			use = candidate
			break
		}
	}
	if use.ProofID == "" {
		t.Fatalf("missing matrix modulo const proof use for %q: %#v", guard.ID, fn.ProofUses)
	}
	if !Dominates(fn, guard.Block, use.Block) {
		t.Fatalf(
			"matrix modulo const guard block %s should dominate use block %s in %+v",
			guard.Block,
			use.Block,
			fn.Dominators,
		)
	}

	var rangeFact RangeFact
	for _, candidate := range fn.RangeFacts {
		if candidate.ProofID == guard.ID {
			rangeFact = candidate
			break
		}
	}
	if rangeFact.ProofID == "" {
		t.Fatalf("missing matrix modulo const range fact for %q: %#v", guard.ID, fn.RangeFacts)
	}
	if !strings.Contains(rangeFact.Value, "r % 9") ||
		rangeFact.Lower != (Bound{Kind: BoundConst, Const: 0}) ||
		rangeFact.Upper != (Bound{Kind: BoundSymbol, Symbol: "c.len"}) ||
		!rangeFact.InclusiveLower ||
		rangeFact.InclusiveUpper ||
		!containsString(rangeFact.Derivation, "modulo_const_allocation_length") {
		t.Fatalf("matrix modulo const range fact = %+v", rangeFact)
	}

	var term ProofTerm
	for _, candidate := range fn.ProofTerms {
		if candidate.ID == guard.ID {
			term = candidate
			break
		}
	}
	if term.ID == "" {
		t.Fatalf("missing matrix modulo const proof term for %q: %#v", guard.ID, fn.ProofTerms)
	}
	if term.SubjectBaseID != "c" ||
		term.IndexValueID != rangeFact.Value ||
		term.Operation != "index_load" ||
		term.Range != "r % 9 in [0, c.len)" ||
		!containsString(term.FactsUsed, "modulo_const_allocation_length") {
		t.Fatalf("matrix modulo const proof term = %+v", term)
	}
}

func TestFromCheckedProgramRecordsMatrixConstLoopSetupStoreProofs(t *testing.T) {
	checked := checkedProgram(t, `
func main() -> Int
uses alloc, mem:
    var a: []i32 = core.make_i32(9)
    var b: []i32 = core.make_i32(9)
    var c: []i32 = core.make_i32(9)
    var i: Int = 0
    while i < 9:
        a[i] = i + 1
        b[i] = 9 - i
        c[i] = 0
        i = i + 1
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
	termsByBase := map[string]ProofTerm{}
	for _, term := range fn.ProofTerms {
		if !containsString(term.FactsUsed, "const_loop_allocation_length") {
			continue
		}
		switch term.SubjectBaseID {
		case "a", "b", "c":
			termsByBase[term.SubjectBaseID] = term
		}
	}
	if len(termsByBase) != 3 {
		t.Fatalf(
			"setup stores should have one const-loop proof term per base, got %#v\n%s",
			fn.ProofTerms,
			FormatText(prog),
		)
	}

	for _, base := range []string{"a", "b", "c"} {
		term := termsByBase[base]
		if !strings.HasPrefix(term.ID, "proof:while-const:i:"+base+":") ||
			term.IndexValueID != "local:i" ||
			term.Operation != "index_store" ||
			term.Range != "i in [0, "+base+".len)" {
			t.Fatalf("setup store proof term for %s = %+v", base, term)
		}

		var guard ProofGuard
		for _, candidate := range fn.ProofGuards {
			if candidate.ID == term.ID {
				guard = candidate
				break
			}
		}
		if guard.ID == "" {
			t.Fatalf(
				"missing setup store proof guard for %s/%s: %#v",
				base,
				term.ID,
				fn.ProofGuards,
			)
		}
		if guard.Kind != "range" || !strings.Contains(guard.Condition, "i < 9") ||
			!strings.Contains(guard.Condition, base+".len == 9") {
			t.Fatalf("setup store proof guard for %s = %+v", base, guard)
		}

		var use ProofUse
		for _, candidate := range fn.ProofUses {
			if candidate.ProofID == term.ID {
				use = candidate
				break
			}
		}
		if use.ProofID == "" {
			t.Fatalf("missing setup store proof use for %s/%s: %#v", base, term.ID, fn.ProofUses)
		}
		if !Dominates(fn, guard.Block, use.Block) {
			t.Fatalf(
				"setup store guard block %s should dominate use block %s in %+v",
				guard.Block,
				use.Block,
				fn.Dominators,
			)
		}
		var op Operation
		for _, candidate := range fn.Ops {
			if candidate.ID == use.OpID {
				op = candidate
				break
			}
		}
		if op.ID == "" || op.Kind != OpIndexStore || len(op.Inputs) < 2 || op.Inputs[0] != base ||
			op.Inputs[1] != "i" {
			t.Fatalf(
				"setup proof use for %s should target its index_store op, use=%+v op=%+v",
				base,
				use,
				op,
			)
		}

		var rangeFact RangeFact
		for _, candidate := range fn.RangeFacts {
			if candidate.ProofID == term.ID {
				rangeFact = candidate
				break
			}
		}
		if rangeFact.ProofID == "" ||
			rangeFact.Value != "local:i" ||
			rangeFact.Lower != (Bound{Kind: BoundConst, Const: 0}) ||
			rangeFact.Upper != (Bound{Kind: BoundSymbol, Symbol: base + ".len"}) ||
			!rangeFact.InclusiveLower ||
			rangeFact.InclusiveUpper ||
			!containsString(rangeFact.Derivation, "const_loop_allocation_length") {
			t.Fatalf("setup store range fact for %s = %+v", base, rangeFact)
		}
	}
}

func TestFromCheckedProgramRecordsMatrixAffineConstStoreProof(t *testing.T) {
	checked := checkedProgram(t, matrixAffinePLIRProgram(
		"var c: []i32 = core.make_i32(9)",
		"row < 3",
		"col < 3",
		"row * 3 + col",
		"row = row + 1",
		"col = col + 1",
		"",
	))

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}

	fn := findFunction(t, prog, "main")
	term, ok := affineProofTermFor(fn, "c", "index_store")
	if !ok {
		t.Fatalf("missing c affine const store proof term:\n%s", FormatText(prog))
	}
	if term.SubjectBaseID != "c" ||
		term.IndexValueID != "local:row * 3 + col" ||
		term.Operation != "index_store" ||
		term.Range != "row * 3 + col in [0, c.len)" ||
		!containsString(term.FactsUsed, "affine_const_extent") {
		t.Fatalf("affine proof term = %+v", term)
	}

	guard, ok := proofGuardForID(fn, term.ID)
	if !ok {
		t.Fatalf("missing affine proof guard for %q: %#v", term.ID, fn.ProofGuards)
	}
	if guard.Kind != "range" ||
		!strings.Contains(guard.Condition, "row < 3") ||
		!strings.Contains(guard.Condition, "col < 3") ||
		!strings.Contains(guard.Condition, "c.len == 9") ||
		!strings.Contains(guard.Condition, "row * 3 + col") {
		t.Fatalf("affine proof guard = %+v", guard)
	}

	use, ok := proofUseForID(fn, term.ID)
	if !ok {
		t.Fatalf("missing affine proof use for %q: %#v", term.ID, fn.ProofUses)
	}
	if !Dominates(fn, guard.Block, use.Block) {
		t.Fatalf(
			"affine guard block %s should dominate use block %s in %+v",
			guard.Block,
			use.Block,
			fn.Dominators,
		)
	}
	op, ok := operationForID(fn, use.OpID)
	if !ok || op.Kind != OpIndexStore || len(op.Inputs) < 2 || op.Inputs[0] != "c" ||
		op.Inputs[1] != "row * 3 + col" {
		t.Fatalf("affine proof use should point at c index_store, use=%+v op=%+v", use, op)
	}

	rangeFact, ok := rangeFactForProofID(fn, term.ID)
	if !ok {
		t.Fatalf("missing affine range fact for %q: %#v", term.ID, fn.RangeFacts)
	}
	if rangeFact.Value != "local:row * 3 + col" ||
		rangeFact.Lower != (Bound{Kind: BoundConst, Const: 0}) ||
		rangeFact.Upper != (Bound{Kind: BoundSymbol, Symbol: "c.len"}) ||
		!rangeFact.InclusiveLower ||
		rangeFact.InclusiveUpper ||
		!containsString(rangeFact.Derivation, "affine_const_extent") {
		t.Fatalf("affine range fact = %+v", rangeFact)
	}
	if !strings.HasPrefix(term.ID, "proof:affine-const:") || !strings.Contains(term.ID, ":c:") {
		t.Fatalf("affine proof id = %q, want base-specific c id", term.ID)
	}
}

func TestFromCheckedProgramRecordsMatrixAffineConstALoadProof(t *testing.T) {
	checked := checkedProgram(t, matrixAffineLoadPLIRProgram(
		"var a: []i32 = core.make_i32(9)",
		"var c: []i32 = core.make_i32(9)",
		"row < 3",
		"k < 3",
		"row * 3 + k",
		"col < 3",
		"row * 3 + col",
		"row = row + 1",
		"k = k + 1",
		"col = col + 1",
		"",
	))

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}

	fn := findFunction(t, prog, "main")
	aTerms := affineProofTermsForBase(fn, "a")
	if len(aTerms) != 1 {
		t.Fatalf(
			"want exactly one affine proof term for base a, got %#v\n%s",
			aTerms,
			FormatText(prog),
		)
	}
	term := aTerms[0]
	if term.IndexValueID != "local:row * 3 + k" ||
		term.Operation != "index_load" ||
		term.Range != "row * 3 + k in [0, a.len)" ||
		!containsString(term.FactsUsed, "affine_const_extent") {
		t.Fatalf("a affine load proof term = %+v", term)
	}
	if !strings.HasPrefix(term.ID, "proof:affine-const:row_k:a:") {
		t.Fatalf("a affine proof id = %q, want stable base-specific row_k/a id", term.ID)
	}
	bTerms := affineProofTermsForBase(fn, "b")
	if len(bTerms) != 1 {
		t.Fatalf(
			"want exactly one affine proof term for base b, got %#v\n%s",
			bTerms,
			FormatText(prog),
		)
	}
	bTerm := bTerms[0]
	if bTerm.IndexValueID != "local:k * 3 + col" ||
		bTerm.Operation != "index_load" ||
		bTerm.Range != "k * 3 + col in [0, b.len)" ||
		!containsString(bTerm.FactsUsed, "affine_const_extent") {
		t.Fatalf("b affine load proof term = %+v", bTerm)
	}
	if bTerm.ID == term.ID || !strings.HasPrefix(bTerm.ID, "proof:affine-const:k_col:b:") {
		t.Fatalf(
			"b affine proof should stay distinct and base-specific to k_col/b: b=%+v a=%+v",
			bTerm,
			term,
		)
	}

	guard, ok := proofGuardForID(fn, term.ID)
	if !ok {
		t.Fatalf("missing affine load proof guard for %q: %#v", term.ID, fn.ProofGuards)
	}
	if guard.Kind != "range" ||
		!strings.Contains(guard.Condition, "row < 3") ||
		!strings.Contains(guard.Condition, "k < 3") ||
		!strings.Contains(guard.Condition, "a.len == 9") ||
		!strings.Contains(guard.Condition, "row * 3 + k") {
		t.Fatalf("a affine proof guard = %+v", guard)
	}

	use, ok := proofUseForID(fn, term.ID)
	if !ok {
		t.Fatalf("missing affine load proof use for %q: %#v", term.ID, fn.ProofUses)
	}
	if !Dominates(fn, guard.Block, use.Block) {
		t.Fatalf(
			"affine load guard block %s should dominate use block %s in %+v",
			guard.Block,
			use.Block,
			fn.Dominators,
		)
	}
	op, ok := operationForID(fn, use.OpID)
	if !ok || op.Kind != OpIndexLoad || len(op.Inputs) < 2 || op.Inputs[0] != "a" ||
		op.Inputs[1] != "row * 3 + k" {
		t.Fatalf("affine proof use should point at a index_load, use=%+v op=%+v", use, op)
	}

	rangeFact, ok := rangeFactForProofID(fn, term.ID)
	if !ok {
		t.Fatalf("missing affine load range fact for %q: %#v", term.ID, fn.RangeFacts)
	}
	if rangeFact.Value != "local:row * 3 + k" ||
		rangeFact.Lower != (Bound{Kind: BoundConst, Const: 0}) ||
		rangeFact.Upper != (Bound{Kind: BoundSymbol, Symbol: "a.len"}) ||
		!rangeFact.InclusiveLower ||
		rangeFact.InclusiveUpper ||
		!containsString(rangeFact.Derivation, "affine_const_extent") {
		t.Fatalf("a affine load range fact = %+v", rangeFact)
	}

	cTerm, ok := affineProofTermFor(fn, "c", "index_store")
	if !ok || !strings.HasPrefix(cTerm.ID, "proof:affine-const:row_col:c:") {
		t.Fatalf("P38 c store proof should remain intact, got %+v\n%s", cTerm, FormatText(prog))
	}
}

func TestFromCheckedProgramMatrixAffineProofIDsStayBaseSpecific(t *testing.T) {
	checked := checkedProgram(t, matrixAffinePLIRProgram(
		"var c: []i32 = core.make_i32(9)",
		"row < 3",
		"col < 3",
		"row * 3 + col",
		"row = row + 1",
		"col = col + 1",
		"a[row * 3 + col] = total",
	))

	prog, err := FromCheckedProgram(checked)
	if err != nil {
		t.Fatalf("FromCheckedProgram: %v", err)
	}
	if err := VerifyProgram(prog); err != nil {
		t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
	}

	fn := findFunction(t, prog, "main")
	term, ok := affineProofTermFor(fn, "c", "index_store")
	if !ok {
		t.Fatalf("expected affine proof term for c store:\n%s", FormatText(prog))
	}
	if term.SubjectBaseID != "c" || !strings.Contains(term.ID, ":c:") {
		t.Fatalf("affine proof should stay base-specific to c, got %+v", term)
	}
	for _, candidate := range fn.ProofTerms {
		if !strings.HasPrefix(candidate.ID, "proof:affine-const:") {
			continue
		}
		if candidate.SubjectBaseID == "" ||
			!strings.Contains(candidate.ID, ":"+candidate.SubjectBaseID+":") {
			t.Fatalf("affine proof id is not base-specific: %+v", candidate)
		}
	}
}

func TestFromCheckedProgramRejectsInvalidMatrixAffineConstALoadProofs(t *testing.T) {
	tests := []struct {
		name       string
		aDecl      string
		rowGuard   string
		kGuard     string
		aLoadIndex string
		rowInc     string
		kInc       string
		beforeLoad string
	}{
		{
			name:       "wrong_stride",
			aDecl:      "var a: []i32 = core.make_i32(9)",
			rowGuard:   "row < 3",
			kGuard:     "k < 3",
			aLoadIndex: "row * 4 + k",
			rowInc:     "row = row + 1",
			kInc:       "k = k + 1",
		},
		{
			name:       "mutable_allocation_length",
			aDecl:      "var n: Int = 9\n    var a: []i32 = core.make_i32(n)",
			rowGuard:   "row < 3",
			kGuard:     "k < 3",
			aLoadIndex: "row * 3 + k",
			rowInc:     "row = row + 1",
			kInc:       "k = k + 1",
		},
		{
			name:       "non_unit_k_increment",
			aDecl:      "var a: []i32 = core.make_i32(9)",
			rowGuard:   "row < 3",
			kGuard:     "k < 3",
			aLoadIndex: "row * 3 + k",
			rowInc:     "row = row + 1",
			kInc:       "k = k + 2",
		},
		{
			name:       "non_strict_k_guard",
			aDecl:      "var a: []i32 = core.make_i32(9)",
			rowGuard:   "row < 3",
			kGuard:     "k <= 2",
			aLoadIndex: "row * 3 + k",
			rowInc:     "row = row + 1",
			kInc:       "k = k + 1",
		},
		{
			name:       "base_reassignment_before_load",
			aDecl:      "var a: []i32 = core.make_i32(9)",
			rowGuard:   "row < 3",
			kGuard:     "k < 3",
			aLoadIndex: "row * 3 + k",
			rowInc:     "row = row + 1",
			kInc:       "k = k + 1",
			beforeLoad: "a = core.make_i32(9)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checked := checkedProgram(t, matrixAffineLoadPLIRProgram(
				tt.aDecl,
				"var c: []i32 = core.make_i32(9)",
				tt.rowGuard,
				tt.kGuard,
				tt.aLoadIndex,
				"col < 3",
				"row * 3 + col",
				tt.rowInc,
				tt.kInc,
				"col = col + 1",
				tt.beforeLoad,
			))
			prog, err := FromCheckedProgram(checked)
			if err != nil {
				t.Fatalf("FromCheckedProgram: %v", err)
			}
			if err := VerifyProgram(prog); err != nil {
				t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
			}
			fn := findFunction(t, prog, "main")
			if got := affineProofTermsForBase(fn, "a"); len(got) != 0 {
				t.Fatalf(
					"%s: invalid a load shape received affine load proof terms: %#v\n%s",
					tt.name,
					got,
					FormatText(prog),
				)
			}
			for _, use := range fn.ProofUses {
				if strings.HasPrefix(use.ProofID, "proof:affine-const:row_k:a:") {
					t.Fatalf(
						"%s: invalid a load shape received affine proof use: %+v\n%s",
						tt.name,
						use,
						FormatText(prog),
					)
				}
			}
			for _, candidate := range fn.ProofTerms {
				if strings.HasPrefix(candidate.ID, "proof:affine-const:") &&
					(candidate.SubjectBaseID == "" || !strings.Contains(
						candidate.ID,
						":"+candidate.SubjectBaseID+":",
					)) {
					t.Fatalf("%s: affine proof id is not base-specific: %+v", tt.name, candidate)
				}
			}
			bTerms := affineProofTermsForBase(fn, "b")
			if len(bTerms) > 1 {
				t.Fatalf(
					"%s: want at most one b affine proof, got %#v\n%s",
					tt.name,
					bTerms,
					FormatText(prog),
				)
			}
			if len(bTerms) == 1 {
				bTerm := bTerms[0]
				if bTerm.IndexValueID != "local:k * 3 + col" ||
					bTerm.Operation != "index_load" ||
					bTerm.Range != "k * 3 + col in [0, b.len)" ||
					!strings.HasPrefix(bTerm.ID, "proof:affine-const:k_col:b:") {
					t.Fatalf(
						"%s: b affine proof should be the valid k_col/b proof, got %+v",
						tt.name,
						bTerm,
					)
				}
			}
		})
	}
}

func TestFromCheckedProgramRejectsInvalidMatrixAffineConstStoreProofs(t *testing.T) {
	tests := []struct {
		name        string
		cDecl       string
		rowGuard    string
		colGuard    string
		storeIndex  string
		rowInc      string
		colInc      string
		beforeStore string
	}{
		{
			name:       "wrong_stride",
			cDecl:      "var c: []i32 = core.make_i32(9)",
			rowGuard:   "row < 3",
			colGuard:   "col < 3",
			storeIndex: "row * 4 + col",
			rowInc:     "row = row + 1",
			colInc:     "col = col + 1",
		},
		{
			name:       "mutable_allocation_length",
			cDecl:      "var n: Int = 9\n    var c: []i32 = core.make_i32(n)",
			rowGuard:   "row < 3",
			colGuard:   "col < 3",
			storeIndex: "row * 3 + col",
			rowInc:     "row = row + 1",
			colInc:     "col = col + 1",
		},
		{
			name:       "non_unit_col_increment",
			cDecl:      "var c: []i32 = core.make_i32(9)",
			rowGuard:   "row < 3",
			colGuard:   "col < 3",
			storeIndex: "row * 3 + col",
			rowInc:     "row = row + 1",
			colInc:     "col = col + 2",
		},
		{
			name:       "non_strict_col_guard",
			cDecl:      "var c: []i32 = core.make_i32(9)",
			rowGuard:   "row < 3",
			colGuard:   "col <= 2",
			storeIndex: "row * 3 + col",
			rowInc:     "row = row + 1",
			colInc:     "col = col + 1",
		},
		{
			name:        "base_reassignment_before_store",
			cDecl:       "var c: []i32 = core.make_i32(9)",
			rowGuard:    "row < 3",
			colGuard:    "col < 3",
			storeIndex:  "row * 3 + col",
			rowInc:      "row = row + 1",
			colInc:      "col = col + 1",
			beforeStore: "c = core.make_i32(9)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checked := checkedProgram(
				t,
				matrixAffinePLIRProgram(
					tt.cDecl,
					tt.rowGuard,
					tt.colGuard,
					tt.storeIndex,
					tt.rowInc,
					tt.colInc,
					tt.beforeStore,
				),
			)
			prog, err := FromCheckedProgram(checked)
			if err != nil {
				t.Fatalf("FromCheckedProgram: %v", err)
			}
			if err := VerifyProgram(prog); err != nil {
				t.Fatalf("VerifyProgram: %v\n%s", err, FormatText(prog))
			}
			fn := findFunction(t, prog, "main")
			if _, ok := affineProofTermFor(fn, "c", "index_store"); ok {
				t.Fatalf(
					"%s: invalid affine store shape received c store proof metadata:\n%s",
					tt.name,
					FormatText(prog),
				)
			}
		})
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
		t.Fatalf(
			"xs length const = known:%v value:%d, want known:256",
			xs.Alloc.LengthConstKnown,
			xs.Alloc.LengthConst,
		)
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
		t.Fatalf(
			"mutable allocation length alias should not receive proof guards/uses: %#v %#v",
			fn.ProofGuards,
			fn.ProofUses,
		)
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
		t.Fatalf(
			"mutable xs length const = known:%v value:%d, want runtime guarded",
			xs.Alloc.LengthConstKnown,
			xs.Alloc.LengthConst,
		)
	}
}

func TestAllocationLengthAliasRejectsMutableMakeLengthBuiltins(t *testing.T) {
	for _, tc := range []struct {
		name      string
		sliceType string
	}{
		{name: "make_u8", sliceType: "[]u8"},
		{name: "make_u16", sliceType: "[]u16"},
		{name: "make_i32", sliceType: "[]i32"},
		{name: "make_bool", sliceType: "[]bool"},
		{name: "core.make_u8", sliceType: "[]u8"},
		{name: "core.make_u16", sliceType: "[]u16"},
		{name: "core.make_i32", sliceType: "[]i32"},
		{name: "core.make_bool", sliceType: "[]bool"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checked := checkedProgram(t, strings.NewReplacer(
				"$TYPE", tc.sliceType,
				"$BUILTIN", tc.name,
			).Replace(`
func main() -> Int
uses alloc, mem:
    var n: Int = 256
    var xs: $TYPE = $BUILTIN(n)
    return xs.len
`))
			prog, err := FromCheckedProgram(checked)
			if err != nil {
				t.Fatalf("FromCheckedProgram: %v", err)
			}
			if err := VerifyProgram(prog); err != nil {
				t.Fatalf("VerifyProgram: %v", err)
			}
			xs := findValue(t, findFunction(t, prog, "main"), "alloc_intent:xs")
			if xs.Alloc == nil {
				t.Fatalf("xs alloc intent missing: %#v", xs)
			}
			if xs.Alloc.LengthConstKnown {
				t.Fatalf(
					"%s mutable length const = known:%v value:%d, want runtime guarded",
					tc.name,
					xs.Alloc.LengthConstKnown,
					xs.Alloc.LengthConst,
				)
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
	if len(fn.RangeFacts) != 1 || fn.RangeFacts[0].Upper.Symbol != "view.len" ||
		fn.RangeFacts[0].InclusiveUpper {
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
		if fact.Kind == FactProvenanceKnown &&
			(fact.ValueID == "local:view" || fact.ValueID == "local:alias") {
			t.Fatalf("raw alias local must not receive provenance_known fact: %#v", fact)
		}
		if fact.Kind == FactProvenanceUnknown &&
			(fact.ValueID == "local:view" || fact.ValueID == "local:alias") {
			unknownLocals[fact.ValueID] = true
		}
	}
	if len(fn.ProofGuards) != 0 || len(fn.RangeFacts) != 0 {
		t.Fatalf(
			"raw alias loop must not receive proof metadata: guards=%#v ranges=%#v",
			fn.ProofGuards,
			fn.RangeFacts,
		)
	}
	for _, valueID := range []string{"local:view", "local:alias"} {
		if !unknownLocals[valueID] {
			t.Fatalf(
				"raw alias local %s should record conservative unknown provenance: %#v",
				valueID,
				fn.Facts,
			)
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
			t.Fatalf(
				"branch-joined invalid alias must not receive index_in_range fact: %#v",
				fn.Facts,
			)
		}
	}
	if len(fn.ProofGuards) != 0 || len(fn.RangeFacts) != 0 {
		t.Fatalf(
			"branch-joined invalid alias must not receive proof metadata: guards=%#v ranges=%#v",
			fn.ProofGuards,
			fn.RangeFacts,
		)
	}
}

func TestVerifierRejectsUnknownProofUse(t *testing.T) {
	prog := &Program{Funcs: []Function{{
		Name: "bad",
		Blocks: []BasicBlock{
			{
				ID:    "entry",
				Kind:  "entry",
				Entry: true,
				Succs: []string{"body"},
				Ops:   []string{"op0"},
			},
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
			{
				ID:    "then",
				Kind:  "then",
				Preds: []string{"entry"},
				Ops:   []string{"op0"},
				Succs: []string{"join"},
			},
			{
				ID:    "sibling",
				Kind:  "else",
				Preds: []string{"entry"},
				Ops:   []string{"op1"},
				Succs: []string{"join"},
			},
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
			{
				ID:      "f2",
				Kind:    FactNoAlias,
				ValueID: "param:xs",
				Reason:  "forged external alias claim",
			},
		},
	}}}
	err := VerifyProgram(prog)
	if err == nil || !strings.Contains(err.Error(), "no_alias requires parameter provenance") {
		t.Fatalf("VerifyProgram error = %v, want no_alias provenance rejection", err)
	}
}

func matrixAffinePLIRProgram(
	cDecl string,
	rowGuard string,
	colGuard string,
	storeIndex string,
	rowInc string,
	colInc string,
	beforeStore string,
) string {
	if beforeStore != "" {
		beforeStore = "\n            " + beforeStore
	}
	return strings.NewReplacer(
		"$C_DECL", cDecl,
		"$ROW_GUARD", rowGuard,
		"$COL_GUARD", colGuard,
		"$STORE_INDEX", storeIndex,
		"$ROW_INC", rowInc,
		"$COL_INC", colInc,
		"$BEFORE_STORE", beforeStore,
	).Replace(`
func main() -> Int
uses alloc, mem:
    var a: []i32 = core.make_i32(9)
    var b: []i32 = core.make_i32(9)
    $C_DECL
    var row: Int = 0
    while $ROW_GUARD:
        var col: Int = 0
        while $COL_GUARD:
            var k: Int = 0
            var total: Int = 0
            while k < 3:
                total = total + a[row * 3 + k] * b[k * 3 + col]
                k = k + 1$BEFORE_STORE
            c[$STORE_INDEX] = total
            $COL_INC
        $ROW_INC
    return 0
`)
}

func matrixAffineLoadPLIRProgram(
	aDecl string,
	cDecl string,
	rowGuard string,
	kGuard string,
	aLoadIndex string,
	colGuard string,
	storeIndex string,
	rowInc string,
	kInc string,
	colInc string,
	beforeLoad string,
) string {
	if beforeLoad != "" {
		beforeLoad = "\n                " + beforeLoad
	}
	return strings.NewReplacer(
		"$A_DECL", aDecl,
		"$C_DECL", cDecl,
		"$ROW_GUARD", rowGuard,
		"$K_GUARD", kGuard,
		"$A_LOAD_INDEX", aLoadIndex,
		"$COL_GUARD", colGuard,
		"$STORE_INDEX", storeIndex,
		"$ROW_INC", rowInc,
		"$K_INC", kInc,
		"$COL_INC", colInc,
		"$BEFORE_LOAD", beforeLoad,
	).Replace(`
func main() -> Int
uses alloc, mem:
    $A_DECL
    var b: []i32 = core.make_i32(9)
    $C_DECL
    var row: Int = 0
    while $ROW_GUARD:
        var col: Int = 0
        while $COL_GUARD:
            var k: Int = 0
            var total: Int = 0
            while $K_GUARD:$BEFORE_LOAD
                total = total + a[$A_LOAD_INDEX] * b[k * 3 + col]
                $K_INC
            c[$STORE_INDEX] = total
            $COL_INC
        $ROW_INC
    return 0
`)
}

func singleAffineProofTerm(fn Function) (ProofTerm, bool) {
	var term ProofTerm
	count := 0
	for _, candidate := range fn.ProofTerms {
		if strings.HasPrefix(candidate.ID, "proof:affine-const:") {
			term = candidate
			count++
		}
	}
	return term, count == 1
}

func affineProofTermsForBase(fn Function, base string) []ProofTerm {
	var out []ProofTerm
	for _, candidate := range fn.ProofTerms {
		if strings.HasPrefix(candidate.ID, "proof:affine-const:") &&
			candidate.SubjectBaseID == base {
			out = append(out, candidate)
		}
	}
	return out
}

func affineProofTermFor(fn Function, base string, operation string) (ProofTerm, bool) {
	var term ProofTerm
	count := 0
	for _, candidate := range fn.ProofTerms {
		if strings.HasPrefix(candidate.ID, "proof:affine-const:") &&
			candidate.SubjectBaseID == base &&
			candidate.Operation == operation {
			term = candidate
			count++
		}
	}
	return term, count == 1
}

func hasAffineProofMetadata(fn Function) bool {
	for _, guard := range fn.ProofGuards {
		if strings.HasPrefix(guard.ID, "proof:affine-const:") {
			return true
		}
	}
	for _, use := range fn.ProofUses {
		if strings.HasPrefix(use.ProofID, "proof:affine-const:") {
			return true
		}
	}
	for _, term := range fn.ProofTerms {
		if strings.HasPrefix(term.ID, "proof:affine-const:") {
			return true
		}
	}
	for _, fact := range fn.RangeFacts {
		if strings.HasPrefix(fact.ProofID, "proof:affine-const:") {
			return true
		}
	}
	return false
}

func proofGuardForID(fn Function, proofID string) (ProofGuard, bool) {
	for _, guard := range fn.ProofGuards {
		if guard.ID == proofID {
			return guard, true
		}
	}
	return ProofGuard{}, false
}

func proofUseForID(fn Function, proofID string) (ProofUse, bool) {
	for _, use := range fn.ProofUses {
		if use.ProofID == proofID {
			return use, true
		}
	}
	return ProofUse{}, false
}

func operationForID(fn Function, opID string) (Operation, bool) {
	for _, op := range fn.Ops {
		if op.ID == opID {
			return op, true
		}
	}
	return Operation{}, false
}

func rangeFactForProofID(fn Function, proofID string) (RangeFact, bool) {
	for _, fact := range fn.RangeFacts {
		if fact.ProofID == proofID {
			return fact, true
		}
	}
	return RangeFact{}, false
}
