package plir

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tetra_language/compiler/internal/layoutopt"
	corerangeproof "tetra_language/compiler/internal/rangeproof"
	"tetra_language/compiler/internal/semantics"
)

type Program struct {
	Funcs []Function `json:"funcs"`
}

type Function struct {
	Name        string           `json:"name"`
	Module      string           `json:"module,omitempty"`
	Summary     *FunctionSummary `json:"summary,omitempty"`
	Values      []Value          `json:"values,omitempty"`
	Ops         []Operation      `json:"ops,omitempty"`
	Facts       []Fact           `json:"facts,omitempty"`
	Blocks      []BasicBlock     `json:"blocks,omitempty"`
	Dominators  []DominatorRow   `json:"dominators,omitempty"`
	ProofGuards []ProofGuard     `json:"proof_guards,omitempty"`
	ProofUses   []ProofUse       `json:"proof_uses,omitempty"`
	ProofTerms  []ProofTerm      `json:"proof_terms,omitempty"`
	RangeFacts  []RangeFact      `json:"range_facts,omitempty"`
}

type FunctionSummary struct {
	Generic               bool                            `json:"generic,omitempty"`
	Public                bool                            `json:"public,omitempty"`
	Async                 bool                            `json:"async,omitempty"`
	ParamNames            []string                        `json:"param_names,omitempty"`
	ParamTypes            []string                        `json:"param_types,omitempty"`
	ParamOwnership        []string                        `json:"param_ownership,omitempty"`
	ReturnType            string                          `json:"return_type,omitempty"`
	ReturnOwnership       string                          `json:"return_ownership,omitempty"`
	ThrowsType            string                          `json:"throws_type,omitempty"`
	Effects               []string                        `json:"effects,omitempty"`
	TouchesMutableGlobals bool                            `json:"touches_mutable_globals,omitempty"`
	ReturnRegionUnknown   bool                            `json:"return_region_unknown,omitempty"`
	ReturnRegionSummary   map[string]int                  `json:"return_region_summary,omitempty"`
	ReturnResourceUnknown bool                            `json:"return_resource_unknown,omitempty"`
	ReturnResourceSummary map[string][]ResourceProvenance `json:"return_resource_summary,omitempty"`
	ThrowResourceSummary  map[string][]ResourceProvenance `json:"throw_resource_summary,omitempty"`
}

type ResourceProvenance struct {
	ParamIndex int    `json:"param_index"`
	ParamPath  string `json:"param_path,omitempty"`
}

func (f Function) HasFact(kind FactKind) bool {
	for _, fact := range f.Facts {
		if fact.Kind == kind {
			return true
		}
	}
	return false
}

type Value struct {
	ID          string       `json:"id"`
	Kind        ValueKind    `json:"kind,omitempty"`
	Type        string       `json:"type"`
	Source      string       `json:"source,omitempty"`
	Region      string       `json:"region,omitempty"`
	Alloc       *AllocIntent `json:"alloc,omitempty"`
	Provenance  Provenance   `json:"provenance"`
	UnsafeClass UnsafeClass  `json:"unsafe_class,omitempty"`
	Lifetime    Lifetime     `json:"lifetime,omitempty"`
	Borrow      BorrowKind   `json:"borrow,omitempty"`
	Mutable     bool         `json:"mutable,omitempty"`
	Escape      EscapeState  `json:"escape,omitempty"`
}

type AllocIntent struct {
	ElementType            string `json:"element_type"`
	ElementSize            int    `json:"element_size"`
	LengthExpr             string `json:"length_expr,omitempty"`
	LengthConstKnown       bool   `json:"length_const_known,omitempty"`
	LengthConst            int64  `json:"length_const,omitempty"`
	ZeroGuardStatus        string `json:"zero_guard_status,omitempty"`
	NegativeGuardStatus    string `json:"negative_guard_status,omitempty"`
	OverflowGuardStatus    string `json:"overflow_guard_status,omitempty"`
	Builtin                string `json:"builtin,omitempty"`
	Source                 string `json:"source,omitempty"`
	RawPointerBoundsStatus string `json:"raw_pointer_bounds_status,omitempty"`
	RawPointerBaseID       string `json:"raw_pointer_base_id,omitempty"`
	RawPointerBaseBytes    int64  `json:"raw_pointer_base_bytes,omitempty"`
	RawPointerOffsetBytes  int64  `json:"raw_pointer_offset_bytes,omitempty"`
	RawSlicePolicy         string `json:"raw_slice_policy,omitempty"`
}

type ValueKind string

const (
	ValueParam       ValueKind = "param"
	ValueLocal       ValueKind = "local"
	ValueLoopIndex   ValueKind = "loop_index"
	ValueAllocIntent ValueKind = "alloc_intent"
	ValueView        ValueKind = "view"
)

type Provenance struct {
	Kind ProvenanceKind `json:"kind"`
	Root string         `json:"root,omitempty"`
}

type ProvenanceKind string

const (
	ProvenanceAllocation    ProvenanceKind = "allocation"
	ProvenanceIsland        ProvenanceKind = "island"
	ProvenanceStack         ProvenanceKind = "stack"
	ProvenanceLiteral       ProvenanceKind = "literal"
	ProvenanceExternal      ProvenanceKind = "external"
	ProvenanceUnknown       ProvenanceKind = "unknown"
	ProvenanceActorTransfer ProvenanceKind = "actor_transfer"
	ProvenanceParam         ProvenanceKind = "param"
)

type UnsafeClass string

const (
	UnsafeSafe         UnsafeClass = "safe"
	UnsafeUnknown      UnsafeClass = "unsafe_unknown"
	UnsafeChecked      UnsafeClass = "unsafe_checked"
	UnsafeVerifiedRoot UnsafeClass = "unsafe_verified_root"
)

type Lifetime struct {
	Birth string `json:"birth,omitempty"`
	Death string `json:"death,omitempty"`
	Owner string `json:"owner,omitempty"`
}

type BorrowKind string

const (
	BorrowNone BorrowKind = ""
	BorrowImm  BorrowKind = "immutable"
	BorrowMut  BorrowKind = "mutable"
	BorrowMove BorrowKind = "moved"
)

type EscapeState string

const (
	EscapeUnknown      EscapeState = ""
	EscapeNoEscape     EscapeState = "no_escape"
	EscapeReturn       EscapeState = "escapes_return"
	EscapeGlobal       EscapeState = "escapes_global"
	EscapeCallUnknown  EscapeState = "escapes_call_unknown"
	EscapeActor        EscapeState = "escapes_actor"
	EscapeTask         EscapeState = "escapes_task"
	EscapeUnsafe       EscapeState = "escapes_unsafe"
	EscapeClosure      EscapeState = "escapes_closure"
	EscapeAggregate    EscapeState = "escapes_aggregate"
	EscapeConservative EscapeState = "unknown"
)

type Operation struct {
	ID          string        `json:"id"`
	Kind        OperationKind `json:"kind"`
	Block       string        `json:"block,omitempty"`
	Source      string        `json:"source,omitempty"`
	Inputs      []string      `json:"inputs,omitempty"`
	Outputs     []string      `json:"outputs,omitempty"`
	UnsafeClass UnsafeClass   `json:"unsafe_class,omitempty"`
	Note        string        `json:"note,omitempty"`
}

type BasicBlock struct {
	ID     string   `json:"id"`
	Kind   string   `json:"kind,omitempty"`
	Entry  bool     `json:"entry,omitempty"`
	Exit   bool     `json:"exit,omitempty"`
	Preds  []string `json:"preds,omitempty"`
	Succs  []string `json:"succs,omitempty"`
	Ops    []string `json:"ops,omitempty"`
	Source string   `json:"source,omitempty"`
}

type DominatorRow struct {
	Block      string   `json:"block"`
	Dominators []string `json:"dominators"`
}

type ProofGuard struct {
	ID        string     `json:"id"`
	Kind      string     `json:"kind"`
	Block     string     `json:"block"`
	OpID      string     `json:"op_id,omitempty"`
	Condition string     `json:"condition,omitempty"`
	Reason    string     `json:"reason,omitempty"`
	Dominates []ProofUse `json:"dominates,omitempty"`
}

type ProofUse struct {
	ProofID string `json:"proof_id"`
	Block   string `json:"block"`
	OpID    string `json:"op_id,omitempty"`
	UseKind string `json:"use_kind"`
	Source  string `json:"source,omitempty"`
}

type ProofTerm struct {
	ID            string   `json:"id"`
	Kind          string   `json:"kind"`
	SubjectBaseID string   `json:"subject_base_id,omitempty"`
	IndexValueID  string   `json:"index_value_id,omitempty"`
	Operation     string   `json:"operation,omitempty"`
	Range         string   `json:"range,omitempty"`
	IslandID      string   `json:"island_id,omitempty"`
	Epoch         int      `json:"epoch,omitempty"`
	BaseID        string   `json:"base_id,omitempty"`
	Source        string   `json:"source,omitempty"`
	FactsUsed     []string `json:"facts_used,omitempty"`
}

type RangeFact struct {
	Value          string   `json:"value"`
	Lower          Bound    `json:"lower"`
	Upper          Bound    `json:"upper"`
	InclusiveLower bool     `json:"inclusive_lower"`
	InclusiveUpper bool     `json:"inclusive_upper"`
	Source         string   `json:"source"`
	ProofID        string   `json:"proof_id,omitempty"`
	Reason         string   `json:"reason,omitempty"`
	Derivation     []string `json:"derivation,omitempty"`
}

type BoundKind string

const (
	BoundUnknown     BoundKind = "unknown"
	BoundConst       BoundKind = "const"
	BoundSymbol      BoundKind = "symbol"
	BoundSymbolMinus BoundKind = "symbol_minus"
)

type Bound struct {
	Kind   BoundKind `json:"kind"`
	Symbol string    `json:"symbol,omitempty"`
	Const  int64     `json:"const,omitempty"`
}

type OperationKind string

const (
	OpAllocIntent OperationKind = "alloc_intent"
	OpAggregate   OperationKind = "aggregate"
	OpAssign      OperationKind = "assign"
	OpActorSend   OperationKind = "actor_send"
	OpCall        OperationKind = "call"
	OpClosure     OperationKind = "closure_capture"
	OpForSlice    OperationKind = "for_slice"
	OpGlobalStore OperationKind = "global_store"
	OpGuard       OperationKind = "guard"
	OpIndexLoad   OperationKind = "index_load"
	OpIndexStore  OperationKind = "index_store"
	OpPrint       OperationKind = "print"
	OpReturn      OperationKind = "return"
	OpSliceWindow OperationKind = "slice_window"
	OpUnsafe      OperationKind = "unsafe_boundary"
)

type FactKind int

const (
	FactLenStable FactKind = iota
	FactIndexInRange
	FactRegionAlive
	FactNoEscape
	FactNoAlias
	FactNonNull
	FactMaybeNull
	FactAligned
	FactProvenanceKnown
	FactProvenanceUnknown
	FactOwned
	FactBorrowedImm
	FactBorrowedMut
	FactMoved
	FactPureCall
	FactNoHeapAllocation
	FactNoMemWrite
	FactNoActorSend
	FactNoUnknownEscape
	FactDerivedWindow
	FactIslandEpochAdvanced
)

func (k FactKind) String() string {
	switch k {
	case FactLenStable:
		return "len_stable"
	case FactIndexInRange:
		return "index_in_range"
	case FactRegionAlive:
		return "region_alive"
	case FactNoEscape:
		return "no_escape"
	case FactNoAlias:
		return "no_alias"
	case FactNonNull:
		return "non_null"
	case FactMaybeNull:
		return "maybe_null"
	case FactAligned:
		return "aligned"
	case FactProvenanceKnown:
		return "provenance_known"
	case FactProvenanceUnknown:
		return "provenance_unknown"
	case FactOwned:
		return "owned"
	case FactBorrowedImm:
		return "borrowed_imm"
	case FactBorrowedMut:
		return "borrowed_mut"
	case FactMoved:
		return "moved"
	case FactPureCall:
		return "pure_call"
	case FactNoHeapAllocation:
		return "no_heap_allocation"
	case FactNoMemWrite:
		return "no_mem_write"
	case FactNoActorSend:
		return "no_actor_send"
	case FactNoUnknownEscape:
		return "no_unknown_escape"
	case FactDerivedWindow:
		return "derived_window"
	case FactIslandEpochAdvanced:
		return "island_epoch_advanced"
	default:
		return fmt.Sprintf("fact_%d", int(k))
	}
}

func (k FactKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.String())
}

type Fact struct {
	ID       string   `json:"id"`
	Kind     FactKind `json:"kind"`
	ValueID  string   `json:"value_id,omitempty"`
	IslandID string   `json:"island_id,omitempty"`
	Epoch    int      `json:"epoch,omitempty"`
	BaseID   string   `json:"base_id,omitempty"`
	Region   string   `json:"region,omitempty"`
	Range    string   `json:"range,omitempty"`
	ProofID  string   `json:"proof_id,omitempty"`
	Source   string   `json:"source,omitempty"`
	Reason   string   `json:"reason,omitempty"`
	Uses     []string `json:"uses,omitempty"`
}

func FromCheckedProgram(checked *semantics.CheckedProgram) (*Program, error) {
	if checked == nil {
		return nil, fmt.Errorf("plir: missing checked program")
	}
	out := &Program{Funcs: make([]Function, 0, len(checked.Funcs))}
	callBoundaryProofs := corerangeproof.CollectHashLookupCallBoundaryLenProofs(checked)
	helperSummaryProofs := corerangeproof.CollectHelperSummaryProofs(checked)
	helperOffsetProofs := corerangeproof.CollectHelperOffsetProofs(checked)
	for _, fn := range checked.Funcs {
		l := &builder{
			fn:                      fn,
			funcs:                   checked.FuncSigs,
			globals:                 checked.GlobalsByModule[fn.Module],
			types:                   checked.Types,
			values:                  map[string]Value{},
			valueSeq:                0,
			factSeq:                 0,
			opSeq:                   0,
			blockIndex:              map[string]int{},
			zeroLocals:              map[string]bool{},
			constIntLocals:          map[string]int64{},
			lenBoundLocals:          map[string]string{},
			externalLocals:          map[string]bool{},
			invalidLocals:           map[string]bool{},
			rawExposedRoots:         map[string]bool{},
			noAliasInvalidatedRoots: map[string]string{},
			islandTokens:            map[string]islandTokenState{},
			rawPointerRoots:         map[string]string{},
			rawPointerBytes:         map[string]int64{},
			rawPointerOffsets:       map[string]int64{},
			callBoundaryLenProof:    callBoundaryProofs[fn.Name],
			helperSummaryProof:      helperSummaryProofs[fn.Name],
			helperOffsetProof:       helperOffsetProofs[fn.Name],
			activeProof:             nil,
		}
		if err := l.build(); err != nil {
			return nil, err
		}
		out.Funcs = append(out.Funcs, l.function())
	}
	return out, nil
}

type builder struct {
	fn                      semantics.CheckedFunc
	funcs                   map[string]semantics.FuncSig
	globals                 map[string]semantics.GlobalInfo
	types                   map[string]*semantics.TypeInfo
	values                  map[string]Value
	valueSeq                int
	factSeq                 int
	opSeq                   int
	blockSeq                int
	blockIndex              map[string]int
	current                 string
	ops                     []Operation
	facts                   []Fact
	blocks                  []BasicBlock
	proofGuards             []ProofGuard
	proofUses               []ProofUse
	proofTerms              []ProofTerm
	rangeFacts              []RangeFact
	activeProof             []rangeProof
	zeroLocals              map[string]bool
	constIntLocals          map[string]int64
	lenBoundLocals          map[string]string
	externalLocals          map[string]bool
	invalidLocals           map[string]bool
	rawExposedRoots         map[string]bool
	noAliasInvalidatedRoots map[string]string
	islandTokens            map[string]islandTokenState
	rawPointerRoots         map[string]string
	rawPointerBytes         map[string]int64
	rawPointerOffsets       map[string]int64
	callBoundaryLenProof    corerangeproof.CallBoundaryLenProof
	helperSummaryProof      corerangeproof.HelperSummaryProof
	helperOffsetProof       corerangeproof.HelperOffsetProof
}

type islandTokenState struct {
	IslandID string
	Epoch    int
	BaseID   string
}

func (b *builder) build() error {
	b.current = b.newBlock("entry", b.fn.Decl.Pos, true)
	b.addEffectOptimizationFacts()
	localNames := make([]string, 0, len(b.fn.Locals))
	for name := range b.fn.Locals {
		localNames = append(localNames, name)
	}
	sort.Strings(localNames)
	for _, name := range localNames {
		info := b.fn.Locals[name]
		kind := ValueLocal
		prov := Provenance{Kind: ProvenanceStack, Root: "local:" + name}
		borrow := BorrowNone
		for _, param := range b.fn.Decl.Params {
			if param.Name != name {
				continue
			}
			kind = ValueParam
			prov = Provenance{Kind: ProvenanceParam, Root: "param:" + name}
			switch param.Ownership {
			case "inout":
				borrow = BorrowMut
			case "consume":
				borrow = BorrowMove
			default:
				borrow = BorrowImm
			}
			break
		}
		id := valueID(kind, name)
		value := Value{
			ID:         id,
			Kind:       kind,
			Type:       info.TypeName,
			Source:     sourceString(b.fn.Decl.Pos),
			Region:     "fn:" + b.fn.Name,
			Provenance: prov,
			Lifetime:   Lifetime{Birth: "entry", Death: "return", Owner: name},
			Borrow:     borrow,
			Mutable:    info.Mutable,
			Escape:     EscapeConservative,
		}
		b.addValue(value)
		if info.TypeName == "island" {
			b.rememberIslandToken(name, islandTokenState{
				IslandID: "island:" + name,
				Epoch:    1,
				BaseID:   "token:" + name,
			})
		}
		if isMemoryBackedType(info.TypeName) {
			b.addFact(
				Fact{
					Kind:    FactProvenanceKnown,
					ValueID: id,
					Reason:  "checked parameter/local memory value",
				},
			)
			b.addFact(
				Fact{
					Kind:    FactRegionAlive,
					ValueID: id,
					Region:  value.Region,
					Reason:  "function region is alive",
				},
			)
			if kind == ValueParam {
				b.addFact(
					Fact{
						Kind:    FactLenStable,
						ValueID: id,
						Reason:  "safe code cannot mutate slice/string metadata",
					},
				)
			}
			if borrow == BorrowImm {
				b.addFact(Fact{Kind: FactBorrowedImm, ValueID: id})
			}
			if borrow == BorrowMut {
				b.addFact(Fact{Kind: FactBorrowedMut, ValueID: id})
			}
		}
	}
	for _, stmt := range b.fn.Decl.Body {
		b.walkStmt(stmt)
	}
	b.addExclusiveInoutNoAliasFacts()
	b.markExit(b.current)
	return nil
}

func (b *builder) addExclusiveInoutNoAliasFacts() {
	for _, param := range b.fn.Decl.Params {
		if param.Ownership != "inout" {
			continue
		}
		id := valueID(ValueParam, param.Name)
		value, ok := b.values[id]
		if !ok || !isMemoryBackedType(value.Type) {
			continue
		}
		if b.rawExposedRoots[param.Name] {
			continue
		}
		if b.noAliasInvalidatedRoots[param.Name] != "" {
			continue
		}
		if value.Kind != ValueParam || value.Borrow != BorrowMut ||
			value.Provenance.Kind != ProvenanceParam {
			continue
		}
		if value.Lifetime.Birth == "" || value.Lifetime.Death == "" || value.Lifetime.Owner == "" {
			continue
		}
		if b.hasFactForValue(FactProvenanceUnknown, id) ||
			!b.hasFactForValue(FactBorrowedMut, id) ||
			!b.hasFactForValue(FactRegionAlive, id) {
			continue
		}
		if b.hasFactForValue(FactNoAlias, id) {
			continue
		}
		b.addFact(Fact{
			Kind:    FactNoAlias,
			ValueID: id,
			Region:  value.Region,
			Reason:  "inout parameter has exclusive mutable access for call duration",
			Uses:    []string{"active_borrow_graph", "lifetime:return", "provenance:param"},
		})
	}
}

func (b *builder) addEffectOptimizationFacts() {
	facts := layoutopt.EffectFactsFromEnforcedEffectsOpt(b.fn.Effects, layoutopt.EffectFactOptions{
		TouchesMutableGlobals: b.fn.TouchesMutableGlobals,
	})
	for _, name := range facts {
		kind, ok := effectOptimizationFactKind(name)
		if !ok {
			continue
		}
		b.addFact(Fact{
			Kind:   kind,
			Source: sourceString(b.fn.Decl.Pos),
			Reason: "semantics checker enforced declared effects",
			Uses:   append([]string(nil), b.fn.Effects...),
		})
	}
}

func effectOptimizationFactKind(name string) (FactKind, bool) {
	switch name {
	case "pure_call":
		return FactPureCall, true
	case "no_heap_allocation":
		return FactNoHeapAllocation, true
	case "no_mem_write":
		return FactNoMemWrite, true
	case "no_actor_send":
		return FactNoActorSend, true
	case "no_unknown_escape":
		return FactNoUnknownEscape, true
	default:
		return 0, false
	}
}

func (b *builder) function() Function {
	values := make([]Value, 0, len(b.values))
	for _, value := range b.values {
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	fn := Function{
		Name:        b.fn.Name,
		Module:      b.fn.Module,
		Summary:     b.functionSummary(),
		Values:      values,
		Ops:         b.ops,
		Facts:       b.facts,
		Blocks:      b.blocks,
		ProofGuards: b.proofGuards,
		ProofUses:   b.proofUses,
		ProofTerms:  b.proofTerms,
		RangeFacts:  b.rangeFacts,
	}
	b.attachProofUses(&fn)
	fn.Dominators = DominatorRows(fn)
	return fn
}

func (b *builder) functionSummary() *FunctionSummary {
	if b == nil || b.funcs == nil {
		return nil
	}
	sig, ok := b.funcs[b.fn.Name]
	if !ok {
		return nil
	}
	summary := &FunctionSummary{
		Generic:               sig.Generic,
		Public:                sig.Public,
		Async:                 sig.Async,
		ParamNames:            append([]string(nil), sig.ParamNames...),
		ParamTypes:            append([]string(nil), sig.ParamTypes...),
		ParamOwnership:        append([]string(nil), sig.ParamOwnership...),
		ReturnType:            sig.ReturnType,
		ReturnOwnership:       summaryReturnOwnership(sig),
		ThrowsType:            sig.ThrowsType,
		Effects:               append([]string(nil), sig.Effects...),
		TouchesMutableGlobals: sig.TouchesMutableGlobals,
		ReturnRegionUnknown:   sig.ReturnRegionParam < semantics.SummaryParamNone && len(sig.ReturnRegionSummary) == 0,
		ReturnRegionSummary:   cloneIntMap(sig.ReturnRegionSummary),
		ReturnResourceUnknown: sig.ReturnResourceParam < semantics.SummaryParamNone && len(sig.ReturnResourceSummary) == 0,
		ReturnResourceSummary: cloneResourceSummary(sig.ReturnResourceSummary),
		ThrowResourceSummary:  cloneResourceSummary(sig.ThrowResourceSummary),
	}
	return summary
}

func summaryReturnOwnership(sig semantics.FuncSig) string {
	if strings.TrimSpace(sig.ReturnOwnership) != "" {
		return sig.ReturnOwnership
	}
	if !borrowedRegionSummaryType(sig.ReturnType) {
		return ""
	}
	if len(sig.ReturnRegionSummary) > 0 ||
		(sig.ReturnRegionParam >= 0 && sig.ReturnRegionParam < len(sig.ParamTypes)) {
		return "borrow"
	}
	return ""
}

func cloneIntMap(in map[string]int) map[string]int {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneResourceSummary(in semantics.ReturnResourceSummary) map[string][]ResourceProvenance {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string][]ResourceProvenance, len(in))
	for leaf, provenances := range in {
		for _, provenance := range provenances {
			out[leaf] = append(out[leaf], ResourceProvenance{
				ParamIndex: provenance.ParamIndex,
				ParamPath:  provenance.ParamPath,
			})
		}
	}
	return out
}
