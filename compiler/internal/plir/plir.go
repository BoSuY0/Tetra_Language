package plir

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"tetra_language/compiler/internal/frontend"
	"tetra_language/compiler/internal/layoutopt"
	"tetra_language/compiler/internal/rangeproof"
	"tetra_language/compiler/internal/runtimeabi"
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
			b.addFact(Fact{Kind: FactProvenanceKnown, ValueID: id, Reason: "checked parameter/local memory value"})
			b.addFact(Fact{Kind: FactRegionAlive, ValueID: id, Region: value.Region, Reason: "function region is alive"})
			if kind == ValueParam {
				b.addFact(Fact{Kind: FactLenStable, ValueID: id, Reason: "safe code cannot mutate slice/string metadata"})
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
		if value.Kind != ValueParam || value.Borrow != BorrowMut || value.Provenance.Kind != ProvenanceParam {
			continue
		}
		if value.Lifetime.Birth == "" || value.Lifetime.Death == "" || value.Lifetime.Owner == "" {
			continue
		}
		if b.hasFactForValue(FactProvenanceUnknown, id) || !b.hasFactForValue(FactBorrowedMut, id) || !b.hasFactForValue(FactRegionAlive, id) {
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
		ReturnRegionUnknown:   sig.ReturnRegionParam < -1 && len(sig.ReturnRegionSummary) == 0,
		ReturnRegionSummary:   cloneIntMap(sig.ReturnRegionSummary),
		ReturnResourceUnknown: sig.ReturnResourceParam < -1 && len(sig.ReturnResourceSummary) == 0,
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
	if len(sig.ReturnRegionSummary) > 0 || (sig.ReturnRegionParam >= 0 && sig.ReturnRegionParam < len(sig.ParamTypes)) {
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

type rangeProof struct {
	ID             string
	IndexName      string
	IndexValueID   string
	Base           string
	Condition      string
	Source         string
	RangeText      string
	Lower          Bound
	Upper          Bound
	InclusiveLower bool
	InclusiveUpper bool
	Reason         string
	Derivation     []string
}

func (b *builder) walkStmt(stmt frontend.Stmt) {
	switch s := stmt.(type) {
	case *frontend.LetStmt:
		b.walkExpr(s.Value, s.Name)
		if !exprStoresDirectlyIntoTarget(s.Value) {
			b.recordLocalAssignment(s.Name, s.Value, s.At)
		}
		b.rememberLocalProofMetadata(s.Name, s.Value)
		b.rememberAliasMetadata(s.Name, s.Value)
	case *frontend.AssignStmt:
		assignmentWalked := false
		if id, ok := s.Target.(*frontend.IdentExpr); ok {
			if b.isGlobalName(id.Name) {
				b.walkExpr(s.Value, "")
			} else {
				b.clearRawPointerMetadata(id.Name)
				b.walkExpr(s.Value, id.Name)
			}
			assignmentWalked = true
		}
		if !assignmentWalked {
			b.walkExpr(s.Value, "")
		}
		if idx, ok := s.Target.(*frontend.IndexExpr); ok {
			b.walkExpr(idx.Base, "")
			b.walkExpr(idx.Index, "")
			b.addOperation(Operation{
				Kind:   OpIndexStore,
				Source: sourceString(s.At),
				Inputs: []string{exprPath(idx.Base), exprPath(idx.Index)},
			})
		}
		if id, ok := s.Target.(*frontend.IdentExpr); ok {
			if b.isGlobalName(id.Name) {
				b.recordGlobalStore(id.Name, s.Value, s.At)
			} else {
				b.recordLocalAssignment(id.Name, s.Value, s.At)
				b.rememberLocalProofMetadata(id.Name, s.Value)
				b.rememberAliasMetadata(id.Name, s.Value)
				b.invalidateActiveProofForLocal(id.Name)
			}
		}
	case *frontend.ReturnStmt:
		b.walkExpr(s.Value, "$return")
		b.addOperation(Operation{
			Kind:   OpReturn,
			Source: sourceString(s.At),
			Inputs: []string{exprPath(s.Value)},
		})
	case *frontend.ThrowStmt:
		b.walkExpr(s.Value, "")
	case *frontend.PrintStmt:
		b.walkExpr(s.Value, "")
		b.addOperation(Operation{
			Kind:   OpPrint,
			Source: sourceString(s.At),
			Inputs: []string{exprPath(s.Value)},
		})
	case *frontend.ExprStmt:
		b.walkExpr(s.Expr, "")
	case *frontend.IfStmt:
		b.walkIfStmt(s)
	case *frontend.IfLetStmt:
		b.walkExpr(s.Value, s.ValueLocal)
		b.walkBlock(s.Then)
		b.walkBlock(s.Else)
	case *frontend.WhileStmt:
		b.walkWhileStmt(s)
	case *frontend.ForRangeStmt:
		b.walkForRangeStmt(s)
	case *frontend.MatchStmt:
		b.walkExpr(s.Value, s.ScrutineeLocal)
		for _, c := range s.Cases {
			b.walkExpr(c.Guard, "")
			b.walkBlock(c.Body)
		}
	case *frontend.IslandStmt:
		b.walkExpr(s.Size, "")
		b.walkBlock(s.Body)
	case *frontend.UnsafeStmt:
		b.addOperation(Operation{Kind: OpUnsafe, Source: sourceString(s.At), Note: "unsafe block requires conservative provenance/escape assumptions"})
		b.walkBlock(s.Body)
	case *frontend.DeferStmt:
		b.walkBlock(s.Body)
	}
}

func (b *builder) walkBlock(stmts []frontend.Stmt) {
	for _, stmt := range stmts {
		b.walkStmt(stmt)
	}
}

func (b *builder) walkIfStmt(s *frontend.IfStmt) {
	b.walkExpr(s.Cond, "")
	condOp := b.addOperation(Operation{Kind: OpGuard, Source: sourceString(s.At), Inputs: []string{exprPath(s.Cond)}, Note: exprPath(s.Cond)})
	condBlock := b.current
	thenBlock := b.newBlock("if_then", s.At, false)
	elseBlock := ""
	joinBlock := b.newBlock("if_join", s.At, false)
	b.addEdge(condBlock, thenBlock)
	if len(s.Else) > 0 {
		elseBlock = b.newBlock("if_else", s.At, false)
		b.addEdge(condBlock, elseBlock)
	} else {
		b.addEdge(condBlock, joinBlock)
	}
	var proof *rangeProof
	if candidate, ok := b.ifRangeProof(s); ok {
		b.addRangeProof(candidate, thenBlock, condOp.ID)
		proof = &candidate
	}
	b.current = thenBlock
	if proof != nil {
		b.pushActiveProof(*proof)
	}
	branchState := b.snapshotLocalProofState()
	b.walkBlock(s.Then)
	if proof != nil {
		b.popActiveProof()
	}
	thenEnd := b.current
	thenState := b.snapshotLocalProofState()
	b.addEdge(thenEnd, joinBlock)
	elseState := branchState
	if elseBlock != "" {
		b.restoreLocalProofState(branchState)
		b.current = elseBlock
		b.walkBlock(s.Else)
		elseEnd := b.current
		elseState = b.snapshotLocalProofState()
		b.addEdge(elseEnd, joinBlock)
	}
	b.current = joinBlock
	b.mergeLocalProofState(thenState, elseState)
}

func (b *builder) walkWhileStmt(s *frontend.WhileStmt) {
	preheader := b.current
	header := b.newBlock("while_header", s.At, false)
	body := b.newBlock("while_body", s.At, false)
	after := b.newBlock("while_after", s.At, false)
	b.addEdge(preheader, header)
	b.current = header
	b.walkExpr(s.Cond, "")
	condOp := b.addOperation(Operation{Kind: OpGuard, Source: sourceString(s.At), Inputs: []string{exprPath(s.Cond)}, Note: exprPath(s.Cond)})
	b.addEdge(header, body)
	b.addEdge(header, after)

	var proof *rangeProof
	if candidate, ok := b.whileRangeProof(s); ok {
		b.addRangeProof(candidate, body, condOp.ID)
		proof = &candidate
	}
	b.current = body
	if proof != nil {
		b.pushActiveProof(*proof)
	}
	b.walkBlock(s.Body)
	if proof != nil {
		b.popActiveProof()
	}
	b.addEdge(b.current, header)
	b.current = after
	if proof != nil {
		b.zeroLocals[proof.IndexName] = false
	}
}

func (b *builder) walkForRangeStmt(s *frontend.ForRangeStmt) {
	if s.Iterable == nil {
		b.walkExpr(s.Start, "")
		b.walkExpr(s.End, s.EndLocal)
		b.walkBlock(s.Body)
		return
	}
	b.walkExpr(s.Iterable, s.IterableLocal)
	base := exprPath(s.Iterable)
	if base == "" {
		base = s.IterableLocal
	}
	iterID := b.ensureViewValue(s.IterableLocal, base, s.Iterable.Pos())
	indexID := b.addLoopIndex(s)
	preheader := b.current
	header := b.newBlock("for_header", s.At, false)
	body := b.newBlock("for_body", s.At, false)
	after := b.newBlock("for_after", s.At, false)
	b.addEdge(preheader, header)
	b.current = header
	proofID := forCollectionProofID(s)
	op := b.addOperation(Operation{
		Kind:    OpForSlice,
		Source:  sourceString(s.At),
		Inputs:  []string{iterID, indexID},
		Outputs: []string{valueID(ValueLocal, s.Name)},
		Note:    "range: 0.." + base + ".len",
	})
	b.addEdge(header, body)
	b.addEdge(header, after)
	if b.collectionIterableProofAllowed(s.Iterable) {
		latticeRange := rangeproof.LessThanLen(s.IndexLocal, base)
		b.addFact(Fact{
			Kind:    FactIndexInRange,
			ValueID: indexID,
			Range:   "0.." + base + ".len",
			ProofID: proofID,
			Source:  sourceString(s.At),
			Reason:  "for collection loop index is dominated by index < iterable.len guard",
			Uses:    []string{iterID},
		})
		b.addFact(Fact{Kind: FactLenStable, ValueID: iterID, Reason: "for collection iterable is copied into hidden slice header"})
		b.proofGuards = append(b.proofGuards, ProofGuard{
			ID:        proofID,
			Kind:      "range",
			Block:     body,
			OpID:      op.ID,
			Condition: s.IndexLocal + " < " + base + ".len",
			Reason:    "for loop range proof",
		})
		b.proofUses = append(b.proofUses, ProofUse{
			ProofID: proofID,
			Block:   body,
			OpID:    op.ID,
			UseKind: "bounds_check",
			Source:  sourceString(s.At),
		})
		b.addBoundsProofTerm(rangeProof{
			ID:             proofID,
			IndexName:      s.IndexLocal,
			IndexValueID:   indexID,
			Base:           base,
			Condition:      s.IndexLocal + " < " + base + ".len",
			Source:         sourceString(s.At),
			RangeText:      "0.." + base + ".len",
			Lower:          plirBoundFromRangeBound(latticeRange.Lower),
			Upper:          plirBoundFromRangeBound(latticeRange.Upper),
			InclusiveLower: latticeRange.InclusiveLower,
			InclusiveUpper: latticeRange.InclusiveUpper,
			Reason:         "for collection loop index is dominated by index < iterable.len guard",
			Derivation:     append([]string(nil), latticeRange.Derivation...),
		})
		b.rangeFacts = append(b.rangeFacts, RangeFact{
			Value:          indexID,
			Lower:          plirBoundFromRangeBound(latticeRange.Lower),
			Upper:          plirBoundFromRangeBound(latticeRange.Upper),
			InclusiveLower: latticeRange.InclusiveLower,
			InclusiveUpper: latticeRange.InclusiveUpper,
			Source:         sourceString(s.At),
			ProofID:        proofID,
			Reason:         "for loop range proof",
			Derivation:     append([]string(nil), latticeRange.Derivation...),
		})
	}
	b.current = body
	b.walkBlock(s.Body)
	b.addEdge(b.current, header)
	b.current = after
}

func (b *builder) walkExpr(expr frontend.Expr, targetName string) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *frontend.CallExpr:
		for _, arg := range e.Args {
			b.walkExpr(arg, "")
		}
		name := e.Name
		if builtin, ok := semantics.ResolveBuiltinAlias(name); ok {
			name = builtin
		}
		ownership := b.callParamOwnership(name)
		b.invalidateActiveProofsForMutableCallArgs(e.Args, ownership)
		note := b.callSummaryNote(name)
		if boundary := b.callAliasBoundaryKind(name); boundary != "" && callHasInoutArgument(e.Args, ownership) {
			b.invalidateNoAliasForMutableCallArgs(e.Args, ownership, boundary)
			note = appendOperationNote(note, "alias_boundary:"+boundary)
		}
		if b.callSummaryUnknown(name) {
			b.invalidateNoAliasForCallInputs(e.Args, "unknown_external_call")
			note = appendOperationNote(note, "alias_boundary:unknown_external_call")
		}
		if name != e.Name {
			if b.recordBuiltinCall(name, e, targetName) {
				return
			}
		} else if b.recordBuiltinCall(name, e, targetName) {
			return
		}
		b.addOperation(Operation{
			Kind:   OpCall,
			Source: sourceString(e.At),
			Inputs: callInputs(e.Args),
			Note:   note,
		})
	case *frontend.BinaryExpr:
		b.walkExpr(e.Left, "")
		b.walkExpr(e.Right, "")
	case *frontend.UnaryExpr:
		b.walkExpr(e.X, "")
	case *frontend.TryExpr:
		b.walkExpr(e.X, "")
	case *frontend.AwaitExpr:
		b.walkExpr(e.X, "")
	case *frontend.StructLitExpr:
		inputs := make([]string, 0, len(e.Fields))
		for _, field := range e.Fields {
			b.walkExpr(field.Value, "")
			if input := exprPath(field.Value); input != "" {
				inputs = append(inputs, input)
			}
		}
		if targetName != "" && len(inputs) > 0 {
			b.addOperation(Operation{Kind: OpAggregate, Source: sourceString(e.At), Inputs: inputs, Outputs: []string{targetName}, Note: "struct aggregate"})
		}
	case *frontend.IndexExpr:
		b.walkExpr(e.Base, "")
		b.walkExpr(e.Index, "")
		op := b.addOperation(Operation{
			Kind:   OpIndexLoad,
			Source: sourceString(e.At),
			Inputs: []string{exprPath(e.Base), exprPath(e.Index)},
		})
		if proof, ok := b.activeProofForIndex(e); ok {
			b.proofUses = append(b.proofUses, ProofUse{
				ProofID: proof.ID,
				Block:   op.Block,
				OpID:    op.ID,
				UseKind: "bounds_check",
				Source:  sourceString(e.At),
			})
		}
	case *frontend.FieldAccessExpr:
		b.walkExpr(e.Base, "")
	case *frontend.MatchExpr:
		b.walkExpr(e.Value, e.ScrutineeLocal)
		for _, c := range e.Cases {
			b.walkExpr(c.Guard, "")
			b.walkExpr(c.Value, "")
		}
	case *frontend.CatchExpr:
		b.walkExpr(e.Call, "")
		for _, c := range e.Cases {
			b.walkExpr(c.Guard, "")
			b.walkExpr(c.Value, "")
		}
	case *frontend.ClosureExpr:
		inputs := make([]string, 0, len(e.Captures))
		for _, capture := range e.Captures {
			if capture.Name != "" {
				inputs = append(inputs, capture.Name)
			}
		}
		if len(inputs) > 0 {
			outputs := []string(nil)
			if targetName != "" {
				outputs = []string{targetName}
			}
			b.addOperation(Operation{Kind: OpClosure, Source: sourceString(e.At), Inputs: inputs, Outputs: outputs, Note: "closure captures environment"})
		}
	}
}

func (b *builder) recordLocalAssignment(name string, expr frontend.Expr, pos frontend.Position) {
	if name == "" {
		return
	}
	input := exprPath(expr)
	if input == "" || input == name {
		return
	}
	b.addOperation(Operation{Kind: OpAssign, Source: sourceString(pos), Inputs: []string{input}, Outputs: []string{name}, Note: "local assignment"})
}

func (b *builder) recordGlobalStore(name string, expr frontend.Expr, pos frontend.Position) {
	if name == "" {
		return
	}
	input := exprPath(expr)
	if input == "" {
		return
	}
	b.addOperation(Operation{Kind: OpGlobalStore, Source: sourceString(pos), Inputs: []string{input}, Outputs: []string{name}, Note: "global store"})
}

func (b *builder) isGlobalName(name string) bool {
	if name == "" || b.globals == nil {
		return false
	}
	_, ok := b.globals[name]
	return ok
}

func (b *builder) recordBuiltinCall(name string, call *frontend.CallExpr, targetName string) bool {
	if name == "core.alloc_bytes" {
		b.recordAllocBytesCall(name, call, targetName)
		return true
	}
	if name == "core.ptr_add" {
		b.recordRawPtrAddCall(name, call, targetName)
		return true
	}
	if rawMemoryAccessBuiltin(name) {
		b.recordRawMemoryAccessCall(name, call, targetName)
		return true
	}
	elem, ok := makeSliceElem(name)
	if ok {
		b.recordMakeSliceCall(name, elem, call, targetName)
		return true
	}
	elem, ok = rawSliceElem(name)
	if ok {
		b.recordRawSliceCall(name, elem, call, targetName)
		return true
	}
	elem, ok = sliceBorrowElem(name)
	if ok {
		b.recordBorrowCall(name, "[]"+elem, call, targetName)
		return true
	}
	if stringBorrowBuiltin(name) {
		b.recordBorrowCall(name, "str", call, targetName)
		return true
	}
	elem, ok = sliceCopyElem(name)
	if ok {
		b.recordCopyCall(name, "[]"+elem, elem, call, targetName)
		return true
	}
	if stringCopyBuiltin(name) {
		b.recordCopyCall(name, "str", "u8", call, targetName)
		return true
	}
	if name == "core.send_typed" {
		b.recordActorSendCall(name, call, targetName)
		return true
	}
	if name == "core.island_reset" {
		b.recordIslandResetCall(name, call, targetName)
		return true
	}
	if sliceCopyIntoBuiltin(name) || stringCopyIntoBuiltin(name) {
		b.recordCopyIntoCall(name, call)
		return true
	}
	elem, method, ok := sliceViewElem(name)
	if ok {
		b.recordSliceViewCall(name, "[]"+elem, method, call, targetName)
		return true
	}
	valueType, method, ok := stringViewBuiltin(name)
	if ok {
		b.recordSliceViewCall(name, valueType, method, call, targetName)
		return true
	}
	return false
}

func (b *builder) recordCopyIntoCall(name string, call *frontend.CallExpr) {
	source := callArgPath(call, 0)
	destination := callArgPath(call, 1)
	overlap := b.copyIntoOverlapStatus(source, destination)
	note := fmt.Sprintf("%s copies into caller-owned destination without allocation source:%s destination:%s dest_capacity_check:normal_build overlap:%s", name, source, destination, overlap)
	op := b.addOperation(Operation{Kind: OpCall, Source: sourceString(call.At), Inputs: callInputs(call.Args), Note: note})
	b.addCopyLoopRangeProof(name, call, op)
}

func (b *builder) recordActorSendCall(name string, call *frontend.CallExpr, targetName string) {
	op := Operation{
		Kind:   OpActorSend,
		Source: sourceString(call.At),
		Inputs: callInputs(call.Args),
		Note:   name + " typed actor ownership transfer",
	}
	if targetName != "" {
		op.Outputs = []string{targetName}
	}
	b.addOperation(op)
	if len(call.Args) < 2 {
		return
	}
	b.recordTypedActorMovedFacts(call.Args[1], call.At)
}

func (b *builder) recordIslandResetCall(name string, call *frontend.CallExpr, targetName string) {
	inputs := callInputs(call.Args)
	outputs := []string(nil)
	if targetName != "" {
		outputs = []string{valueID(ValueLocal, targetName)}
	}
	b.addOperation(Operation{
		Kind:    OpCall,
		Source:  sourceString(call.At),
		Inputs:  inputs,
		Outputs: outputs,
		Note:    name + " advances island token epoch and consumes the source token",
	})
	if len(call.Args) == 0 {
		return
	}
	source := callArgPath(call, 0)
	if source == "" || source == "?" {
		source = "island"
	}
	sourceToken := b.islandTokenForPath(source)
	nextToken := sourceToken
	nextToken.Epoch++
	if nextToken.Epoch <= 1 {
		nextToken.Epoch = 2
	}
	if targetName != "" {
		b.rememberIslandToken(targetName, nextToken)
	}
	if sourceID, ok := b.localOrParamValueIDForExpr(call.Args[0]); ok {
		b.addFact(Fact{
			Kind:    FactMoved,
			ValueID: sourceID,
			Region:  sourceToken.IslandID,
			Source:  name + " " + sourceString(call.At),
			Reason:  "island reset consumes the source token",
		})
	}
	b.addFact(Fact{
		Kind:     FactIslandEpochAdvanced,
		IslandID: sourceToken.IslandID,
		Epoch:    nextToken.Epoch,
		BaseID:   sourceToken.BaseID,
		Source:   name + " " + sourceString(call.At),
		Reason:   "island reset advances epoch and invalidates previous references",
	})
}

func (b *builder) rememberIslandToken(name string, token islandTokenState) {
	if name == "" || token.IslandID == "" {
		return
	}
	if token.Epoch <= 0 {
		token.Epoch = 1
	}
	if token.BaseID == "" {
		token.BaseID = "token:" + islandTokenRoot(token.IslandID)
	}
	b.islandTokens[name] = token
}

func (b *builder) rememberIslandTokenAlias(name string, expr frontend.Expr) {
	if name == "" {
		return
	}
	path := exprPath(expr)
	if path == "" {
		return
	}
	token, ok := b.islandTokens[path]
	if !ok {
		return
	}
	b.rememberIslandToken(name, token)
}

func (b *builder) islandTokenForPath(path string) islandTokenState {
	if token, ok := b.islandTokens[path]; ok && token.IslandID != "" {
		return token
	}
	if path == "" || path == "?" {
		path = "island"
	}
	return islandTokenState{
		IslandID: "island:" + path,
		Epoch:    1,
		BaseID:   "token:" + path,
	}
}

func islandTokenRoot(islandID string) string {
	return strings.TrimPrefix(islandID, "island:")
}

func (b *builder) recordTypedActorMovedFacts(expr frontend.Expr, pos frontend.Position) {
	msgCall, ok := expr.(*frontend.CallExpr)
	if !ok || msgCall == nil {
		return
	}
	_, caseInfo, ok := plirEnumCaseConstructor(msgCall, b.types)
	if !ok {
		return
	}
	ownerIDs := b.actorTransferOwnerValueIDs(msgCall, caseInfo)
	for i, payloadType := range caseInfo.PayloadTypes {
		if i >= len(msgCall.Args) {
			break
		}
		b.recordTypedActorMovedFactsForPayload(msgCall.Args[i], payloadType, ownerIDs, pos)
	}
}

func (b *builder) actorTransferOwnerValueIDs(call *frontend.CallExpr, caseInfo semantics.EnumCaseInfo) []string {
	var owners []string
	for i, payloadType := range caseInfo.PayloadTypes {
		if i >= len(call.Args) {
			break
		}
		if plirTypeKind(payloadType, b.types) != semantics.TypeIsland {
			continue
		}
		if id, ok := b.localOrParamValueIDForExpr(call.Args[i]); ok {
			owners = append(owners, id)
		}
	}
	return owners
}

func (b *builder) recordTypedActorMovedFactsForPayload(expr frontend.Expr, typeName string, ownerIDs []string, pos frontend.Position) {
	switch plirTypeKind(typeName, b.types) {
	case semantics.TypeIsland:
		if id, ok := b.localOrParamValueIDForExpr(expr); ok {
			b.addTypedActorMovedFact(id, nil, pos)
		}
	case semantics.TypeSlice:
		if len(ownerIDs) == 0 || plirExprIsExplicitCopy(expr) {
			return
		}
		if id, ok := b.localOrParamValueIDForExpr(expr); ok {
			b.addTypedActorMovedFact(id, ownerIDs, pos)
		}
	case semantics.TypeStruct:
		lit, ok := expr.(*frontend.StructLitExpr)
		if !ok {
			return
		}
		info, ok := b.types[typeName]
		if !ok {
			return
		}
		byName := make(map[string]frontend.Expr, len(lit.Fields))
		for _, field := range lit.Fields {
			byName[field.Name] = field.Value
		}
		for _, field := range info.Fields {
			if value := byName[field.Name]; value != nil {
				b.recordTypedActorMovedFactsForPayload(value, field.TypeName, ownerIDs, pos)
			}
		}
	case semantics.TypeEnum:
		call, ok := expr.(*frontend.CallExpr)
		if !ok || call == nil {
			return
		}
		_, caseInfo, ok := plirEnumCaseConstructor(call, b.types)
		if !ok {
			return
		}
		for i, payloadType := range caseInfo.PayloadTypes {
			if i >= len(call.Args) {
				break
			}
			b.recordTypedActorMovedFactsForPayload(call.Args[i], payloadType, ownerIDs, pos)
		}
	}
}

func (b *builder) localOrParamValueIDForExpr(expr frontend.Expr) (string, bool) {
	path := exprPath(expr)
	if path == "" {
		return "", false
	}
	for _, kind := range []ValueKind{ValueLocal, ValueParam} {
		id := valueID(kind, path)
		if _, ok := b.values[id]; ok {
			return id, true
		}
	}
	return "", false
}

func (b *builder) addTypedActorMovedFact(valueID string, uses []string, pos frontend.Position) {
	if valueID == "" || b.hasFactForValue(FactMoved, valueID) {
		return
	}
	b.addFact(Fact{
		Kind:    FactMoved,
		ValueID: valueID,
		Region:  "actor_transfer",
		Source:  "core.send_typed " + sourceString(pos),
		Reason:  "typed actor ownership transfer moved payload",
		Uses:    append([]string(nil), uses...),
	})
}

func plirEnumCaseConstructor(call *frontend.CallExpr, types map[string]*semantics.TypeInfo) (string, semantics.EnumCaseInfo, bool) {
	if call == nil {
		return "", semantics.EnumCaseInfo{}, false
	}
	caseName := plirCallCaseName(call.Name)
	if call.ResolvedType != "" {
		if info, ok := types[call.ResolvedType]; ok && info.Kind == semantics.TypeEnum {
			if caseInfo, ok := info.CaseMap[caseName]; ok {
				return call.ResolvedType, caseInfo, true
			}
		}
	}
	for typeName, info := range types {
		if info == nil || info.Kind != semantics.TypeEnum {
			continue
		}
		if caseInfo, ok := info.CaseMap[caseName]; ok {
			return typeName, caseInfo, true
		}
	}
	return "", semantics.EnumCaseInfo{}, false
}

func plirCallCaseName(name string) string {
	if dot := strings.LastIndex(name, "."); dot >= 0 {
		return name[dot+1:]
	}
	return name
}

func plirTypeKind(typeName string, types map[string]*semantics.TypeInfo) semantics.TypeKind {
	if info, ok := types[typeName]; ok && info != nil {
		return info.Kind
	}
	return semantics.TypeKind(-1)
}

func plirExprIsExplicitCopy(expr frontend.Expr) bool {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false
	}
	name := call.Name
	if target, ok := semantics.ResolveBuiltinAlias(name); ok {
		name = target
	}
	return copyBuiltin(name)
}

func (b *builder) recordAllocBytesCall(name string, call *frontend.CallExpr, targetName string) {
	if targetName == "" {
		targetName = b.syntheticTargetName("raw_alloc", call)
	}
	b.clearRawPointerMetadata(targetName)
	lengthArg := allocationLengthArg(name, call)
	lengthExpr := exprPath(lengthArg)
	if lengthExpr == "" {
		lengthExpr = "expr"
	}
	lengthConst, lengthConstKnown := evalConstInt64(lengthArg)
	rawBaseBytes := int64(0)
	if lengthConstKnown && lengthConst > 0 {
		rawBaseBytes = lengthConst
	}
	id := valueID(ValueAllocIntent, targetName)
	value := Value{
		ID:     id,
		Kind:   ValueAllocIntent,
		Type:   "ptr",
		Source: sourceString(call.At),
		Region: "raw_allocation:" + targetName,
		Alloc: &AllocIntent{
			ElementType:            "raw_bytes",
			ElementSize:            1,
			LengthExpr:             lengthExpr,
			LengthConstKnown:       lengthConstKnown,
			LengthConst:            lengthConst,
			ZeroGuardStatus:        "invalid_precondition",
			NegativeGuardStatus:    "reject_before_allocation",
			OverflowGuardStatus:    "reject_before_allocation",
			Builtin:                name,
			Source:                 sourceString(call.At),
			RawPointerBoundsStatus: string(runtimeabi.RawPointerBoundsAllocationBase),
			RawPointerBaseID:       targetName,
			RawPointerBaseBytes:    rawBaseBytes,
			RawPointerOffsetBytes:  0,
			RawSlicePolicy:         string(runtimeabi.RawSliceBoundsExternalUnknown),
		},
		Provenance:  Provenance{Kind: ProvenanceAllocation, Root: targetName},
		UnsafeClass: UnsafeVerifiedRoot,
		Lifetime:    Lifetime{Birth: sourceString(call.At), Owner: targetName},
		Mutable:     true,
		Escape:      EscapeConservative,
	}
	b.addValue(value)
	b.addOperation(Operation{
		Kind:        OpAllocIntent,
		Source:      sourceString(call.At),
		Inputs:      callInputs(call.Args),
		Outputs:     []string{id},
		UnsafeClass: UnsafeVerifiedRoot,
		Note:        "alloc_bytes raw allocation-base metadata: zero invalid, negative and overflow reject before allocation",
	})
	b.rawPointerRoots[targetName] = targetName
	b.rawPointerOffsets[targetName] = 0
	if rawBaseBytes > 0 {
		b.rawPointerBytes[targetName] = rawBaseBytes
	}
}

func (b *builder) recordRawPtrAddCall(name string, call *frontend.CallExpr, targetName string) {
	if targetName != "" {
		b.clearRawPointerMetadata(targetName)
	}
	inputs := callInputs(call.Args)
	outputs := []string(nil)
	if targetName != "" {
		outputs = []string{targetName}
	}
	base := callArgPath(call, 0)
	offset := callArgPath(call, 1)
	status := runtimeabi.RawPointerBoundsCheckedExternalUnknown
	unsafeClass := UnsafeUnknown
	note := fmt.Sprintf("%s raw_pointer_bounds: %s base:%s offset:%s", name, status, base, offset)
	if baseRoot := b.rawPointerBaseRoot(base); baseRoot != "" {
		status = runtimeabi.RawPointerBoundsDerivedOffset
		unsafeClass = UnsafeChecked
		validDerived := true
		offsetBytes, offsetKnown := evalConstInt64(callArg(call, 1))
		if !offsetKnown {
			status = runtimeabi.RawPointerBoundsCheckedExternalUnknown
			unsafeClass = UnsafeUnknown
			validDerived = false
			note = fmt.Sprintf("%s raw_pointer_bounds: %s base:%s offset:%s", name, status, baseRoot, offset)
		}
		if offsetKnown && offsetBytes < 0 {
			status = runtimeabi.RawPointerBoundsRejectedNegativeOffset
			validDerived = false
			note = fmt.Sprintf("%s raw_pointer_bounds: %s base:%s offset:%d width:%d", name, status, baseRoot, offsetBytes, int64(1))
		}
		baseOffset := int64(0)
		if prior, ok := b.rawPointerOffsetBytes(base); ok {
			baseOffset = prior
		}
		totalOffset := offsetBytes
		offsetSumOK := true
		if offsetKnown && offsetBytes >= 0 {
			totalOffset, offsetSumOK = checkedAddInt64(baseOffset, offsetBytes)
			if !offsetSumOK {
				status = runtimeabi.RawPointerBoundsRejectedAccessWidthOverflow
				validDerived = false
				note = fmt.Sprintf("%s raw_pointer_bounds: %s base:%s offset:%s", name, status, baseRoot, offset)
			}
		}
		if offsetSumOK && offsetKnown && offsetBytes >= 0 {
			if baseBytes, bytesKnown := b.rawPointerBaseByteSize(baseRoot); bytesKnown && offsetKnown {
				root, err := runtimeabi.NewRawAllocationBounds(baseRoot, baseBytes)
				if err == nil {
					derived, diag := runtimeabi.DeriveRawPointerBounds(root, totalOffset, 1)
					status = derived.Status
					validDerived = diag == nil
					note = fmt.Sprintf("%s raw_pointer_bounds: %s base:%s offset:%d width:%d", name, status, baseRoot, totalOffset, derived.AccessWidthBytes)
				}
			} else {
				note = fmt.Sprintf("%s raw_pointer_bounds: %s base:%s offset:%s", name, status, baseRoot, offset)
			}
		}
		if targetName != "" && validDerived {
			b.rawPointerRoots[targetName] = baseRoot
			if offsetKnown {
				b.rawPointerOffsets[targetName] = totalOffset
			}
		}
	}
	b.addOperation(Operation{Kind: OpUnsafe, Source: sourceString(call.At), Inputs: inputs, Outputs: outputs, UnsafeClass: unsafeClass, Note: note})
}

func (b *builder) recordRawMemoryAccessCall(name string, call *frontend.CallExpr, targetName string) {
	if targetName != "" {
		b.clearRawPointerMetadata(targetName)
	}
	inputs := callInputs(call.Args)
	outputs := []string(nil)
	if targetName != "" {
		outputs = []string{targetName}
	}
	ptr := callArgPath(call, 0)
	unsafeClass := b.rawPointerUnsafeClass(ptr)
	status := runtimeabi.RawPointerBoundsCheckedExternalUnknown
	if unsafeClass == UnsafeChecked {
		status = runtimeabi.RawPointerBoundsDerivedOffset
		if root := b.rawPointerBaseRoot(ptr); root != "" {
			if baseBytes, bytesKnown := b.rawPointerBaseByteSize(root); bytesKnown {
				if offsetBytes, offsetKnown := b.rawPointerOffsetBytes(ptr); offsetKnown {
					rootBounds, err := runtimeabi.NewRawAllocationBounds(root, baseBytes)
					if err == nil {
						derived, _ := runtimeabi.DeriveRawPointerBounds(rootBounds, offsetBytes, rawMemoryAccessWidthBytes(name))
						status = derived.Status
					}
				}
			}
		}
	}
	note := fmt.Sprintf("%s raw memory gateway: %s pointer:%s width:%d", name, status, ptr, rawMemoryAccessWidthBytes(name))
	if offsetBytes, offsetKnown := b.rawPointerOffsetBytes(ptr); offsetKnown {
		note = fmt.Sprintf("%s raw memory gateway: %s pointer:%s offset:%d width:%d", name, status, ptr, offsetBytes, rawMemoryAccessWidthBytes(name))
	}
	b.addOperation(Operation{Kind: OpUnsafe, Source: sourceString(call.At), Inputs: inputs, Outputs: outputs, UnsafeClass: unsafeClass, Note: note})
}

func (b *builder) rawPointerUnsafeClass(path string) UnsafeClass {
	if b.rawPointerBaseRoot(path) != "" {
		return UnsafeChecked
	}
	return UnsafeUnknown
}

func (b *builder) rawPointerBaseRoot(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || path == "?" || path == "expr" {
		return ""
	}
	if root, ok := b.rawPointerRoots[path]; ok {
		return root
	}
	if dot := strings.Index(path, "."); dot > 0 {
		if root, ok := b.rawPointerRoots[path[:dot]]; ok {
			return root
		}
	}
	return ""
}

func (b *builder) rawPointerBaseByteSize(root string) (int64, bool) {
	root = strings.TrimSpace(root)
	if root == "" {
		return 0, false
	}
	bytes, ok := b.rawPointerBytes[root]
	return bytes, ok && bytes > 0
}

func (b *builder) rawPointerOffsetBytes(path string) (int64, bool) {
	path = strings.TrimSpace(path)
	if path == "" {
		return 0, false
	}
	if offset, ok := b.rawPointerOffsets[path]; ok {
		return offset, true
	}
	if dot := strings.Index(path, "."); dot > 0 {
		if offset, ok := b.rawPointerOffsets[path[:dot]]; ok {
			return offset, true
		}
	}
	if root, ok := b.rawPointerRoots[path]; ok && root == path {
		return 0, true
	}
	return 0, false
}

func (b *builder) clearRawPointerMetadata(name string) {
	if name == "" {
		return
	}
	delete(b.rawPointerRoots, name)
	delete(b.rawPointerBytes, name)
	delete(b.rawPointerOffsets, name)
}

func (b *builder) recordMakeSliceCall(name string, elem string, call *frontend.CallExpr, targetName string) {
	if targetName == "" {
		targetName = b.syntheticTargetName("alloc", call)
	}
	id := valueID(ValueAllocIntent, targetName)
	prov := Provenance{Kind: ProvenanceAllocation, Root: targetName}
	region := "allocation:" + targetName
	zeroGuardStatus := "valid_empty_no_allocator"
	negativeGuardStatus := "reject_before_allocation"
	overflowGuardStatus := "reject_before_allocation"
	inputs := []string(nil)
	islandFactToken := islandTokenState{}
	if strings.HasPrefix(name, "core.island_make_") {
		islandRoot := callArgPath(call, 0)
		if islandRoot == "?" || islandRoot == "" {
			islandRoot = "island"
		}
		islandFactToken = b.islandTokenForPath(islandRoot)
		prov.Kind = ProvenanceIsland
		prov.Root = islandTokenRoot(islandFactToken.IslandID)
		region = islandFactToken.IslandID
		zeroGuardStatus = "valid_empty_no_metadata_access"
		negativeGuardStatus = "reject_before_metadata_access"
		overflowGuardStatus = "reject_before_metadata_access"
		inputs = callInputs(call.Args)
	}
	lengthArg := allocationLengthArg(name, call)
	lengthExpr := exprPath(lengthArg)
	if lengthExpr == "" {
		lengthExpr = "expr"
	}
	lengthConst, lengthConstKnown := evalConstInt64(lengthArg)
	value := Value{
		ID:     id,
		Kind:   ValueAllocIntent,
		Type:   "[]" + elem,
		Source: sourceString(call.At),
		Region: region,
		Alloc: &AllocIntent{
			ElementType:         elem,
			ElementSize:         elementSize(elem),
			LengthExpr:          lengthExpr,
			LengthConstKnown:    lengthConstKnown,
			LengthConst:         lengthConst,
			ZeroGuardStatus:     zeroGuardStatus,
			NegativeGuardStatus: negativeGuardStatus,
			OverflowGuardStatus: overflowGuardStatus,
			Builtin:             name,
			Source:              sourceString(call.At),
		},
		Provenance: prov,
		Lifetime:   Lifetime{Birth: sourceString(call.At), Owner: targetName},
		Mutable:    true,
		Escape:     EscapeConservative,
	}
	b.addValue(value)
	note := "make<" + elem + "> length contract: zero valid, negative and overflow reject before allocation"
	if prov.Kind == ProvenanceIsland {
		note = "island_make<" + elem + "> length contract: zero valid, negative and overflow reject before island metadata access"
	}
	b.addOperation(Operation{Kind: OpAllocIntent, Source: sourceString(call.At), Inputs: inputs, Outputs: []string{id}, Note: note})
	if prov.Kind == ProvenanceIsland {
		b.addFact(b.islandAllocationFact(FactProvenanceKnown, id, islandFactToken, "", "compiler-known allocation intent"))
		b.addFact(b.islandAllocationFact(FactLenStable, id, islandFactToken, "", "slice metadata is opaque in safe code"))
		b.addFact(b.islandAllocationFact(FactRegionAlive, id, islandFactToken, value.Region, ""))
		b.addFact(b.islandAllocationFact(FactAligned, id, islandFactToken, value.Region, "island region allocator returns 16-byte aligned payloads"))
		return
	}
	b.addFact(Fact{Kind: FactProvenanceKnown, ValueID: id, Reason: "compiler-known allocation intent"})
	b.addFact(Fact{Kind: FactLenStable, ValueID: id, Reason: "slice metadata is opaque in safe code"})
	b.addFact(Fact{Kind: FactRegionAlive, ValueID: id, Region: value.Region})
}

func (b *builder) islandAllocationFact(kind FactKind, valueID string, token islandTokenState, region string, reason string) Fact {
	return Fact{
		Kind:     kind,
		ValueID:  valueID,
		IslandID: token.IslandID,
		Epoch:    token.Epoch,
		Region:   region,
		Reason:   reason,
	}
}

func (b *builder) recordRawSliceCall(name string, elem string, call *frontend.CallExpr, targetName string) {
	if targetName == "" {
		targetName = b.syntheticTargetName("raw_view", call)
	}
	b.recordRawPointerExposure(call)
	id := valueID(ValueView, targetName)
	provenance := Provenance{Kind: ProvenanceExternal, Root: "raw_parts"}
	unsafeClass := UnsafeUnknown
	status := runtimeabi.RawSliceBoundsExternalUnknown
	note := name + " creates a conservative external-provenance view"
	ptr := callArgPath(call, 0)
	length := callArgPath(call, 1)
	if root := b.rawPointerBaseRoot(ptr); root != "" {
		if baseBytes, bytesKnown := b.rawPointerBaseByteSize(root); bytesKnown {
			if lengthConst, lengthKnown := evalConstInt64(callArg(call, 1)); lengthKnown {
				offsetBytes := int64(0)
				if offset, offsetKnown := b.rawPointerOffsetBytes(ptr); offsetKnown {
					offsetBytes = offset
				}
				ptrStatus := runtimeabi.RawPointerBoundsAllocationBase
				if offsetBytes != 0 {
					ptrStatus = runtimeabi.RawPointerBoundsDerivedOffset
				}
				sliceBounds := runtimeabi.RawSliceBoundsFromParts(runtimeabi.RawPointerBoundsMetadata{
					Status:                 ptrStatus,
					BaseID:                 root,
					BaseBytes:              baseBytes,
					OffsetBytes:            offsetBytes,
					VerifiedAllocationRoot: true,
				}, lengthConst, int64(elementSize(elem)))
				status = sliceBounds.Status
				switch status {
				case runtimeabi.RawSliceBoundsVerifiedAllocationRoot, runtimeabi.RawSliceBoundsRejectedNegativeLength, runtimeabi.RawSliceBoundsRejectedLengthOverflow:
					unsafeClass = UnsafeChecked
					provenance.Root = "raw_parts:" + root
					note = fmt.Sprintf("%s raw_slice_bounds: %s base:%s offset:%d length:%d length_bytes:%d elem_size:%d", name, status, root, offsetBytes, lengthConst, sliceBounds.LengthBytes, elementSize(elem))
				default:
					note = fmt.Sprintf("%s raw_slice_bounds: %s base:%s offset:%d length:%s elem_size:%d", name, status, root, offsetBytes, length, elementSize(elem))
				}
			}
		}
	}
	value := Value{
		ID:          id,
		Kind:        ValueView,
		Type:        "[]" + elem,
		Source:      sourceString(call.At),
		Region:      "external:" + targetName,
		Provenance:  provenance,
		UnsafeClass: unsafeClass,
		Lifetime:    Lifetime{Birth: sourceString(call.At), Owner: targetName},
		Mutable:     true,
		Escape:      EscapeConservative,
	}
	b.addValue(value)
	b.addOperation(Operation{Kind: OpUnsafe, Source: sourceString(call.At), Inputs: callInputs(call.Args), Outputs: []string{id}, UnsafeClass: unsafeClass, Note: note})
	if status == runtimeabi.RawSliceBoundsExternalUnknown {
		b.addFact(Fact{Kind: FactProvenanceUnknown, ValueID: id, Reason: "raw slice gateway has external provenance unless an unsafe proof supplies more facts"})
	}
	b.reclassifyMemoryBinding(targetName, provenance, "raw slice gateway has external provenance unless an unsafe proof supplies more facts")
}

func (b *builder) recordRawPointerExposure(call *frontend.CallExpr) {
	root := rawPointerRoot(callArgPath(call, 0))
	if root == "" {
		return
	}
	b.rawExposedRoots[root] = true
}

func (b *builder) recordSliceViewCall(name string, valueType string, method string, call *frontend.CallExpr, targetName string) {
	if targetName == "" {
		targetName = b.syntheticTargetName("slice_view", call)
	}
	id := valueID(ValueView, targetName)
	source := firstArgPath(call)
	prov, known := b.derivedProvenance(source)
	sourceInvalid := b.exprIsInvalidView(callArg(call, 0))
	invalidView := staticInvalidStringViewCall(name, call) || sourceInvalid
	if invalidView {
		prov = Provenance{Kind: ProvenanceUnknown}
		known = false
	}
	value := Value{
		ID:         id,
		Kind:       ValueView,
		Type:       valueType,
		Source:     sourceString(call.At),
		Region:     "fn:" + b.fn.Name,
		Provenance: prov,
		Lifetime:   Lifetime{Birth: sourceString(call.At), Owner: targetName},
		Borrow:     BorrowImm,
		Mutable:    true,
		Escape:     EscapeNoEscape,
	}
	b.addValue(value)
	if invalidView {
		reason := "statically invalid String view has no constructed header"
		if sourceInvalid {
			reason = "view source is invalid before construction"
		}
		b.addOperation(Operation{Kind: OpSliceWindow, Source: sourceString(call.At), Inputs: callInputs(call.Args), Outputs: []string{id}, Note: name + " invalid range is rejected before construction"})
		b.addFact(Fact{Kind: FactProvenanceUnknown, ValueID: id, Reason: reason})
		b.reclassifyMemoryBinding(targetName, Provenance{Kind: ProvenanceUnknown}, reason)
		return
	}
	windowRange := b.sliceViewRange(method, source, call)
	width, shift := sliceViewElementLayout(valueType)
	b.addOperation(Operation{Kind: OpSliceWindow, Source: sourceString(call.At), Inputs: callInputs(call.Args), Outputs: []string{id}, Note: fmt.Sprintf("%s range %s elem_width:%d elem_shift:%d bounds_check:normal_build", name, windowRange, width, shift)})
	b.addFact(Fact{Kind: FactDerivedWindow, ValueID: id, Range: windowRange, Source: sourceString(call.At), Reason: "safe slice view range is checked before construction"})
	b.addFact(Fact{Kind: FactRegionAlive, ValueID: id, Region: value.Region})
	b.addFact(Fact{Kind: FactBorrowedImm, ValueID: id})
	b.addFact(Fact{Kind: FactNoEscape, ValueID: id, Reason: "slice view may not escape its owner"})
	if known {
		b.addFact(Fact{Kind: FactProvenanceKnown, ValueID: id, Reason: "slice view provenance is derived from source slice"})
		b.addFact(Fact{Kind: FactLenStable, ValueID: id, Reason: "safe slice view metadata is constructed by checked compiler builtin"})
		return
	}
	b.addFact(Fact{Kind: FactProvenanceUnknown, ValueID: id, Reason: "slice view source provenance is external or unknown"})
	b.reclassifyMemoryBinding(targetName, prov, "slice view source provenance is external or unknown")
}

func (b *builder) recordBorrowCall(name string, valueType string, call *frontend.CallExpr, targetName string) {
	if targetName == "" {
		targetName = b.syntheticTargetName("borrow", call)
	}
	id := valueID(ValueView, targetName)
	source := firstArgPath(call)
	prov, known := b.derivedProvenance(source)
	value := Value{
		ID:         id,
		Kind:       ValueView,
		Type:       valueType,
		Source:     sourceString(call.At),
		Region:     "fn:" + b.fn.Name,
		Provenance: prov,
		Lifetime:   Lifetime{Birth: sourceString(call.At), Owner: targetName},
		Borrow:     BorrowImm,
		Mutable:    false,
		Escape:     EscapeNoEscape,
	}
	b.addValue(value)
	b.addOperation(Operation{Kind: OpCall, Source: sourceString(call.At), Inputs: callInputs(call.Args), Outputs: []string{id}, Note: name + " creates borrowed view without allocation"})
	b.addFact(Fact{Kind: FactBorrowedImm, ValueID: id, Reason: "explicit borrow view"})
	b.addFact(Fact{Kind: FactNoEscape, ValueID: id, Reason: "explicit borrowed view may not escape owner"})
	b.addFact(Fact{Kind: FactRegionAlive, ValueID: id, Region: value.Region})
	if known {
		b.addFact(Fact{Kind: FactProvenanceKnown, ValueID: id, Reason: "borrow preserves source provenance"})
		b.addFact(Fact{Kind: FactLenStable, ValueID: id, Reason: "borrowed view header is immutable in safe code"})
	} else {
		b.addFact(Fact{Kind: FactProvenanceUnknown, ValueID: id, Reason: "borrow source provenance is external or unknown"})
		b.reclassifyMemoryBinding(targetName, prov, "borrow source provenance is external or unknown")
	}
	b.copyDerivedWindowFacts(source, id, "borrow preserves derived window range")
}

func (b *builder) recordCopyCall(name string, valueType string, elem string, call *frontend.CallExpr, targetName string) {
	if targetName == "" {
		targetName = b.syntheticTargetName("copy", call)
	}
	id := valueID(ValueAllocIntent, targetName)
	source := firstArgPath(call)
	lengthExpr := "expr.len"
	if source != "" {
		lengthExpr = source + ".len"
	}
	lengthConst, lengthConstKnown := b.copyLengthConst(call)
	value := Value{
		ID:     id,
		Kind:   ValueAllocIntent,
		Type:   valueType,
		Source: sourceString(call.At),
		Region: "allocation:" + targetName,
		Alloc: &AllocIntent{
			ElementType:         elem,
			ElementSize:         elementSize(elem),
			LengthExpr:          lengthExpr,
			LengthConstKnown:    lengthConstKnown,
			LengthConst:         lengthConst,
			ZeroGuardStatus:     "valid_empty_no_allocator",
			NegativeGuardStatus: "reject_before_allocation",
			OverflowGuardStatus: "reject_before_allocation",
			Builtin:             name,
			Source:              sourceString(call.At),
		},
		Provenance: Provenance{Kind: ProvenanceAllocation, Root: targetName},
		Lifetime:   Lifetime{Birth: sourceString(call.At), Owner: targetName},
		Borrow:     BorrowNone,
		Mutable:    true,
		Escape:     EscapeConservative,
	}
	b.addValue(value)
	op := b.addOperation(Operation{Kind: OpAllocIntent, Source: sourceString(call.At), Inputs: callInputs(call.Args), Outputs: []string{id}, Note: name + " creates owned copy with new provenance"})
	b.addFact(Fact{Kind: FactOwned, ValueID: id, Reason: "copy result owns new storage"})
	b.addFact(Fact{Kind: FactProvenanceKnown, ValueID: id, Reason: "copy creates owned value with new provenance"})
	b.addFact(Fact{Kind: FactLenStable, ValueID: id, Reason: "copy result metadata is owned by the new allocation"})
	b.addFact(Fact{Kind: FactRegionAlive, ValueID: id, Region: value.Region})
	b.addCopyLoopRangeProof(name, call, op)
}

func (b *builder) addCopyLoopRangeProof(name string, call *frontend.CallExpr, op Operation) {
	if call == nil {
		return
	}
	proofID := copyLoopProofID(name, call.At)
	source := firstArgPath(call)
	if source == "" {
		source = "source"
	}
	indexName := fmt.Sprintf("copy:%s:%d:%d:index", proofNamePart(name, "copy"), call.At.Line, call.At.Col)
	indexID := valueID(ValueLoopIndex, indexName)
	b.addValue(Value{
		ID:         indexID,
		Kind:       ValueLoopIndex,
		Type:       "i32",
		Source:     sourceString(call.At),
		Region:     "fn:" + b.fn.Name,
		Provenance: Provenance{Kind: ProvenanceStack, Root: indexName},
		Lifetime:   Lifetime{Birth: "copy_loop:start", Death: "copy_loop:end", Owner: indexName},
		Escape:     EscapeNoEscape,
	})
	b.addFact(Fact{
		Kind:    FactIndexInRange,
		ValueID: indexID,
		Range:   "0.." + source + ".len",
		ProofID: proofID,
		Source:  sourceString(call.At),
		Reason:  "copy loop source index is dominated by index < source.len guard",
		Uses:    callInputs(call.Args),
	})
	b.proofGuards = append(b.proofGuards, ProofGuard{
		ID:        proofID,
		Kind:      "range",
		Block:     op.Block,
		OpID:      op.ID,
		Condition: indexName + " < " + source + ".len",
		Reason:    "copy loop range proof",
	})
	b.proofUses = append(b.proofUses, ProofUse{
		ProofID: proofID,
		Block:   op.Block,
		OpID:    op.ID,
		UseKind: "bounds_check",
		Source:  sourceString(call.At),
	})
	b.addBoundsProofTerm(rangeProof{
		ID:             proofID,
		IndexName:      indexName,
		IndexValueID:   indexID,
		Base:           source,
		Condition:      indexName + " < " + source + ".len",
		Source:         sourceString(call.At),
		RangeText:      "0.." + source + ".len",
		Lower:          Bound{Kind: BoundConst, Const: 0},
		Upper:          Bound{Kind: BoundSymbol, Symbol: source + ".len"},
		InclusiveLower: true,
		InclusiveUpper: false,
		Reason:         "copy loop range proof",
		Derivation:     []string{"non_negative", "less_than_len"},
	})
	b.rangeFacts = append(b.rangeFacts, RangeFact{
		Value:          indexID,
		Lower:          Bound{Kind: BoundConst, Const: 0},
		Upper:          Bound{Kind: BoundSymbol, Symbol: source + ".len"},
		InclusiveLower: true,
		InclusiveUpper: false,
		Source:         sourceString(call.At),
		ProofID:        proofID,
		Reason:         "copy loop range proof",
	})
}

func (b *builder) copyLengthConst(call *frontend.CallExpr) (int64, bool) {
	if call == nil || len(call.Args) == 0 {
		return 0, false
	}
	if n, ok := directViewLengthConst(call.Args[0]); ok {
		return n, true
	}
	source := firstArgPath(call)
	if source == "" {
		return 0, false
	}
	candidates := []string{
		valueID(ValueView, source),
		valueID(ValueAllocIntent, source),
		valueID(ValueLocal, source),
		valueID(ValueParam, source),
	}
	for _, candidate := range candidates {
		for _, fact := range b.facts {
			if fact.Kind != FactDerivedWindow || fact.ValueID != candidate {
				continue
			}
			if n, ok := derivedWindowLengthConst(fact.Range); ok {
				return n, true
			}
		}
	}
	return 0, false
}

func directViewLengthConst(expr frontend.Expr) (int64, bool) {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return 0, false
	}
	name := call.Name
	if target, aliasOK := semantics.ResolveBuiltinAlias(name); aliasOK {
		name = target
	}
	switch {
	case strings.HasPrefix(name, "core.slice_borrow_") || name == "core.string_borrow":
		if len(call.Args) != 1 {
			return 0, false
		}
		return directViewLengthConst(call.Args[0])
	case strings.HasPrefix(name, "core.slice_window_") || name == "core.string_window":
		if len(call.Args) != 3 {
			return 0, false
		}
		return evalConstInt64(call.Args[2])
	case strings.HasPrefix(name, "core.slice_prefix_") || name == "core.string_prefix":
		if len(call.Args) != 2 {
			return 0, false
		}
		return evalConstInt64(call.Args[1])
	default:
		return 0, false
	}
}

func derivedWindowLengthConst(rangeText string) (int64, bool) {
	start := strings.LastIndex(rangeText, "[")
	end := strings.LastIndex(rangeText, "]")
	if start < 0 || end <= start {
		return 0, false
	}
	parts := strings.Split(rangeText[start+1:end], "..")
	if len(parts) != 2 {
		return 0, false
	}
	lo := strings.TrimSpace(parts[0])
	hi := strings.TrimSpace(parts[1])
	if plus := strings.LastIndex(hi, "+"); plus >= 0 {
		prefix := strings.TrimSpace(hi[:plus])
		if prefix == lo {
			n, err := strconv.ParseInt(strings.TrimSpace(hi[plus+1:]), 10, 64)
			return n, err == nil
		}
	}
	if lo == "0" {
		n, err := strconv.ParseInt(hi, 10, 64)
		return n, err == nil
	}
	return 0, false
}

func (b *builder) copyDerivedWindowFacts(source string, dstValueID string, reason string) {
	if source == "" {
		return
	}
	candidates := []string{
		valueID(ValueView, source),
		valueID(ValueAllocIntent, source),
		valueID(ValueLocal, source),
		valueID(ValueParam, source),
	}
	for _, candidate := range candidates {
		for _, fact := range b.facts {
			if fact.Kind != FactDerivedWindow || fact.ValueID != candidate {
				continue
			}
			b.addFact(Fact{Kind: FactDerivedWindow, ValueID: dstValueID, Range: fact.Range, Source: fact.Source, Reason: reason, Uses: []string{candidate}})
			return
		}
	}
}

func (b *builder) derivedProvenance(source string) (Provenance, bool) {
	if source == "" {
		return Provenance{Kind: ProvenanceUnknown}, false
	}
	if strings.HasPrefix(source, "string:") {
		return Provenance{Kind: ProvenanceLiteral, Root: source}, true
	}
	for _, kind := range []ValueKind{ValueAllocIntent, ValueView, ValueParam, ValueLocal} {
		id := valueID(kind, source)
		if value, ok := b.values[id]; ok {
			switch value.Provenance.Kind {
			case ProvenanceUnknown, "":
				return Provenance{Kind: ProvenanceUnknown}, false
			case ProvenanceExternal:
				return Provenance{Kind: ProvenanceExternal, Root: "derived:" + value.Provenance.Root}, false
			default:
				return Provenance{Kind: value.Provenance.Kind, Root: "derived:" + value.Provenance.Root}, true
			}
		}
	}
	return Provenance{Kind: ProvenanceParam, Root: "derived:" + source}, true
}

func (b *builder) ensureViewValue(localName string, base string, pos frontend.Position) string {
	if localName == "" {
		localName = base
	}
	id := valueID(ValueView, localName)
	if _, ok := b.values[id]; ok {
		return id
	}
	typeName := ""
	if info, ok := b.fn.Locals[localName]; ok {
		typeName = info.TypeName
	}
	if typeName == "" && base != "" {
		if info, ok := b.fn.Locals[base]; ok {
			typeName = info.TypeName
		}
	}
	value := Value{
		ID:         id,
		Kind:       ValueView,
		Type:       typeName,
		Source:     sourceString(pos),
		Region:     "fn:" + b.fn.Name,
		Provenance: Provenance{Kind: ProvenanceParam, Root: "param:" + base},
		Lifetime:   Lifetime{Birth: sourceString(pos), Death: "loop:end", Owner: base},
		Borrow:     BorrowImm,
		Escape:     EscapeNoEscape,
	}
	b.addValue(value)
	b.addFact(Fact{Kind: FactProvenanceKnown, ValueID: id, Reason: "for collection iterable view preserves source provenance"})
	b.addFact(Fact{Kind: FactRegionAlive, ValueID: id, Region: value.Region})
	b.addFact(Fact{Kind: FactBorrowedImm, ValueID: id})
	b.addFact(Fact{Kind: FactNoEscape, ValueID: id, Reason: "for collection iterable view may not escape its owner"})
	return id
}

func (b *builder) addLoopIndex(s *frontend.ForRangeStmt) string {
	name := s.IndexLocal
	if name == "" {
		name = s.Name + ":index"
	}
	id := valueID(ValueLoopIndex, name)
	value := Value{
		ID:         id,
		Kind:       ValueLoopIndex,
		Type:       "i32",
		Source:     sourceString(s.At),
		Region:     "fn:" + b.fn.Name,
		Provenance: Provenance{Kind: ProvenanceStack, Root: name},
		Lifetime:   Lifetime{Birth: "loop:start", Death: "loop:end", Owner: name},
		Escape:     EscapeNoEscape,
	}
	b.addValue(value)
	return id
}

func (b *builder) addValue(value Value) {
	if value.ID == "" {
		value.ID = fmt.Sprintf("v%d", b.valueSeq)
		b.valueSeq++
	}
	b.values[value.ID] = value
}

func (b *builder) syntheticTargetName(prefix string, call *frontend.CallExpr) string {
	if path := callResultPath(call); path != "" {
		return path
	}
	if call != nil && (call.At.Line != 0 || call.At.Col != 0) {
		return syntheticCallPath(prefix, call)
	}
	name := fmt.Sprintf("%s_%d", prefix, b.valueSeq)
	b.valueSeq++
	return name
}

func (b *builder) addFact(fact Fact) {
	if fact.ID == "" {
		fact.ID = fmt.Sprintf("f%d", b.factSeq)
		b.factSeq++
	}
	if fact.ValueID != "" {
		if value, ok := b.values[fact.ValueID]; ok && value.Provenance.Kind == ProvenanceIsland {
			root := value.Provenance.Root
			if root == "" {
				root = "unknown"
			}
			if fact.IslandID == "" {
				fact.IslandID = "island:" + root
			}
			if fact.Epoch == 0 {
				fact.Epoch = 1
			}
			if fact.BaseID == "" {
				fact.BaseID = value.ID
			}
		}
	}
	b.facts = append(b.facts, fact)
}

func (b *builder) addOperation(op Operation) Operation {
	if op.ID == "" {
		op.ID = fmt.Sprintf("op%d", b.opSeq)
		b.opSeq++
	}
	if op.Block == "" {
		op.Block = b.current
	}
	b.ops = append(b.ops, op)
	if op.Block != "" {
		b.appendBlockOp(op.Block, op.ID)
	}
	return op
}

func (b *builder) newBlock(kind string, pos frontend.Position, entry bool) string {
	id := kind
	if entry {
		id = "entry"
	} else {
		id = fmt.Sprintf("%s:%d", kind, b.blockSeq)
		b.blockSeq++
	}
	block := BasicBlock{ID: id, Kind: kind, Entry: entry, Source: sourceString(pos)}
	b.blockIndex[id] = len(b.blocks)
	b.blocks = append(b.blocks, block)
	return id
}

func (b *builder) appendBlockOp(blockID string, opID string) {
	idx, ok := b.blockIndex[blockID]
	if !ok {
		return
	}
	for _, existing := range b.blocks[idx].Ops {
		if existing == opID {
			return
		}
	}
	b.blocks[idx].Ops = append(b.blocks[idx].Ops, opID)
}

func (b *builder) addEdge(from string, to string) {
	if from == "" || to == "" || from == to {
		return
	}
	fromIdx, fromOK := b.blockIndex[from]
	toIdx, toOK := b.blockIndex[to]
	if !fromOK || !toOK {
		return
	}
	if !containsString(b.blocks[fromIdx].Succs, to) {
		b.blocks[fromIdx].Succs = append(b.blocks[fromIdx].Succs, to)
		sort.Strings(b.blocks[fromIdx].Succs)
	}
	if !containsString(b.blocks[toIdx].Preds, from) {
		b.blocks[toIdx].Preds = append(b.blocks[toIdx].Preds, from)
		sort.Strings(b.blocks[toIdx].Preds)
	}
}

func (b *builder) markExit(blockID string) {
	if idx, ok := b.blockIndex[blockID]; ok {
		b.blocks[idx].Exit = true
	}
}

func (b *builder) attachProofUses(fn *Function) {
	if len(fn.ProofGuards) == 0 || len(fn.ProofUses) == 0 {
		return
	}
	for i := range fn.ProofGuards {
		for _, use := range fn.ProofUses {
			if use.ProofID == fn.ProofGuards[i].ID {
				fn.ProofGuards[i].Dominates = append(fn.ProofGuards[i].Dominates, use)
			}
		}
	}
}

type localProofState struct {
	zero     map[string]bool
	constInt map[string]int64
	lenBound map[string]string
	external map[string]bool
	invalid  map[string]bool
}

func (b *builder) snapshotLocalProofState() localProofState {
	return localProofState{
		zero:     cloneBoolMap(b.zeroLocals),
		constInt: cloneInt64Map(b.constIntLocals),
		lenBound: cloneStringMap(b.lenBoundLocals),
		external: cloneBoolMap(b.externalLocals),
		invalid:  cloneBoolMap(b.invalidLocals),
	}
}

func (b *builder) restoreLocalProofState(state localProofState) {
	b.zeroLocals = cloneBoolMap(state.zero)
	b.constIntLocals = cloneInt64Map(state.constInt)
	b.lenBoundLocals = cloneStringMap(state.lenBound)
	b.externalLocals = cloneBoolMap(state.external)
	b.invalidLocals = cloneBoolMap(state.invalid)
}

func (b *builder) mergeLocalProofState(thenState localProofState, elseState localProofState) {
	keys := map[string]bool{}
	for key := range thenState.zero {
		keys[key] = true
	}
	for key := range elseState.zero {
		keys[key] = true
	}
	for key := range thenState.constInt {
		keys[key] = true
	}
	for key := range elseState.constInt {
		keys[key] = true
	}
	for key := range thenState.lenBound {
		keys[key] = true
	}
	for key := range elseState.lenBound {
		keys[key] = true
	}
	for key := range thenState.external {
		keys[key] = true
	}
	for key := range elseState.external {
		keys[key] = true
	}
	for key := range thenState.invalid {
		keys[key] = true
	}
	for key := range elseState.invalid {
		keys[key] = true
	}
	for key := range keys {
		b.zeroLocals[key] = thenState.zero[key] && elseState.zero[key]
		if thenValue, thenOK := thenState.constInt[key]; thenOK {
			if elseValue, elseOK := elseState.constInt[key]; elseOK && thenValue == elseValue {
				b.constIntLocals[key] = thenValue
			} else {
				delete(b.constIntLocals, key)
			}
		} else {
			delete(b.constIntLocals, key)
		}
		if thenValue, thenOK := thenState.lenBound[key]; thenOK {
			if elseValue, elseOK := elseState.lenBound[key]; elseOK && thenValue == elseValue {
				b.lenBoundLocals[key] = thenValue
			} else {
				delete(b.lenBoundLocals, key)
			}
		} else {
			delete(b.lenBoundLocals, key)
		}
		b.externalLocals[key] = thenState.external[key] || elseState.external[key]
		b.invalidLocals[key] = thenState.invalid[key] || elseState.invalid[key]
	}
}

func cloneBoolMap(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneInt64Map(in map[string]int64) map[string]int64 {
	out := make(map[string]int64, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func (b *builder) pushActiveProof(proof rangeProof) {
	b.activeProof = append(b.activeProof, proof)
}

func (b *builder) popActiveProof() {
	b.activeProof = b.activeProof[:len(b.activeProof)-1]
}

func (b *builder) invalidateActiveProofForLocal(name string) {
	for i := range b.activeProof {
		if proofPathMatchesMutation(b.activeProof[i].IndexName, name) || proofPathMatchesMutation(b.activeProof[i].Base, name) {
			b.activeProof[i].ID = ""
		}
	}
}

func (b *builder) invalidateActiveProofsForMutableCallArgs(args []frontend.Expr, ownership []string) {
	if len(args) == 0 || len(ownership) == 0 {
		return
	}
	for i, owner := range ownership {
		if owner != "inout" {
			continue
		}
		if i >= len(args) {
			break
		}
		path := exprPath(args[i])
		if path == "" {
			continue
		}
		b.invalidateActiveProofForLocal(path)
	}
}

func (b *builder) invalidateNoAliasForMutableCallArgs(args []frontend.Expr, ownership []string, reason string) {
	if len(args) == 0 || len(ownership) == 0 {
		return
	}
	for i, owner := range ownership {
		if owner != "inout" {
			continue
		}
		if i >= len(args) {
			break
		}
		b.invalidateNoAliasForPath(exprPath(args[i]), reason)
	}
}

func (b *builder) invalidateNoAliasForCallInputs(args []frontend.Expr, reason string) {
	for _, arg := range args {
		b.invalidateNoAliasForPath(exprPath(arg), reason)
	}
}

func (b *builder) invalidateNoAliasForPath(path string, reason string) {
	root := rootPath(path)
	if root == "" {
		return
	}
	b.noAliasInvalidatedRoots[root] = reason
}

func rootPath(path string) string {
	if path == "" {
		return ""
	}
	if idx := strings.IndexByte(path, '.'); idx >= 0 {
		return path[:idx]
	}
	return path
}

func callHasInoutArgument(args []frontend.Expr, ownership []string) bool {
	if len(args) == 0 || len(ownership) == 0 {
		return false
	}
	for i, owner := range ownership {
		if i >= len(args) {
			return false
		}
		if owner == "inout" && exprPath(args[i]) != "" {
			return true
		}
	}
	return false
}

func (b *builder) callParamOwnership(name string) []string {
	if name == "" {
		return nil
	}
	if b.funcs != nil {
		if sig, ok := b.funcs[name]; ok {
			return sig.ParamOwnership
		}
	}
	if local, ok := b.fn.Locals[name]; ok && local.FunctionTypeValue {
		return local.FunctionParamOwnership
	}
	if b.globals != nil {
		if global, ok := b.globals[name]; ok && global.FunctionTypeValue {
			return global.FunctionParamOwnership
		}
	}
	return nil
}

func (b *builder) callAliasBoundaryKind(name string) string {
	if name == "" {
		return ""
	}
	if local, ok := b.fn.Locals[name]; ok && local.FunctionTypeValue {
		return "function_typed_inout"
	}
	if b.globals != nil {
		if global, ok := b.globals[name]; ok && global.FunctionTypeValue {
			return "function_typed_inout"
		}
	}
	return ""
}

func (b *builder) callSummaryNote(name string) string {
	if !b.callSummaryUnknown(name) {
		return name
	}
	if name == "" {
		return "unknown external call"
	}
	return name + " unknown external call"
}

func (b *builder) callSummaryUnknown(name string) bool {
	if name == "" {
		return true
	}
	if strings.HasPrefix(name, "core.") {
		return false
	}
	if b.funcs != nil {
		if _, ok := b.funcs[name]; ok {
			return false
		}
	}
	if local, ok := b.fn.Locals[name]; ok && local.FunctionTypeValue {
		return false
	}
	if b.globals != nil {
		if global, ok := b.globals[name]; ok && global.FunctionTypeValue {
			return false
		}
	}
	return true
}

func appendOperationNote(note string, part string) string {
	if strings.TrimSpace(part) == "" {
		return note
	}
	if strings.Contains(note, part) {
		return note
	}
	if strings.TrimSpace(note) == "" {
		return part
	}
	return note + " " + part
}

func proofPathMatchesMutation(proofPath string, mutatedPath string) bool {
	if proofPath == "" || mutatedPath == "" {
		return false
	}
	return proofPath == mutatedPath || strings.HasPrefix(proofPath, mutatedPath+".")
}

func (b *builder) activeProofForIndex(index *frontend.IndexExpr) (rangeProof, bool) {
	base := exprPath(index.Base)
	idx := exprPath(index.Index)
	if base == "" || idx == "" {
		return rangeProof{}, false
	}
	for i := len(b.activeProof) - 1; i >= 0; i-- {
		proof := b.activeProof[i]
		if proof.ID != "" && proof.Base == base && proof.IndexName == idx {
			return proof, true
		}
	}
	return rangeProof{}, false
}

func (b *builder) addRangeProof(proof rangeProof, truthBlock string, opID string) {
	if proof.ID == "" {
		return
	}
	b.addFact(Fact{
		Kind:    FactIndexInRange,
		ValueID: proof.IndexValueID,
		Range:   proof.RangeText,
		ProofID: proof.ID,
		Source:  sourceStringFromProofSource(proof),
		Reason:  proof.Reason,
	})
	b.proofGuards = append(b.proofGuards, ProofGuard{
		ID:        proof.ID,
		Kind:      "range",
		Block:     truthBlock,
		OpID:      opID,
		Condition: proof.Condition,
		Reason:    proof.Reason,
	})
	b.rangeFacts = append(b.rangeFacts, RangeFact{
		Value:          proof.IndexValueID,
		Lower:          proof.Lower,
		Upper:          proof.Upper,
		InclusiveLower: proof.InclusiveLower,
		InclusiveUpper: proof.InclusiveUpper,
		Source:         sourceStringFromProofSource(proof),
		ProofID:        proof.ID,
		Reason:         proof.Reason,
		Derivation:     append([]string(nil), proof.Derivation...),
	})
	b.addBoundsProofTerm(proof)
}

func (b *builder) addBoundsProofTerm(proof rangeProof) {
	if proof.ID == "" {
		return
	}
	for _, term := range b.proofTerms {
		if term.ID == proof.ID {
			return
		}
	}
	term := ProofTerm{
		ID:            proof.ID,
		Kind:          "bounds_check",
		SubjectBaseID: proof.Base,
		IndexValueID:  proof.IndexValueID,
		Operation:     "index_load",
		Range:         proof.RangeText,
		Source:        sourceStringFromProofSource(proof),
		FactsUsed:     append([]string(nil), proof.Derivation...),
	}
	if ref, ok := b.proofMemoryRefForBase(proof.Base); ok {
		term.IslandID = ref.IslandID
		term.Epoch = ref.Epoch
		term.BaseID = ref.BaseID
	}
	b.proofTerms = append(b.proofTerms, term)
}

func (b *builder) proofMemoryRefForBase(base string) (islandTokenState, bool) {
	for _, id := range valueIDsForPath(base) {
		for _, fact := range b.facts {
			if fact.ValueID != id || fact.IslandID == "" || fact.Epoch <= 0 {
				continue
			}
			return islandTokenState{IslandID: fact.IslandID, Epoch: fact.Epoch, BaseID: fact.BaseID}, true
		}
	}
	return islandTokenState{}, false
}

func sourceStringFromProofSource(proof rangeProof) string {
	if proof.Source != "" {
		return proof.Source
	}
	return proof.ID
}

func (b *builder) rememberAliasMetadata(name string, expr frontend.Expr) {
	if name == "" {
		return
	}
	b.externalLocals[name] = b.exprHasExternalProvenance(expr)
	b.invalidLocals[name] = b.exprIsInvalidView(expr)
	b.rememberIslandTokenAlias(name, expr)
	b.recordViewAlias(name, expr)
	if b.invalidLocals[name] {
		b.reclassifyMemoryBinding(name, Provenance{Kind: ProvenanceUnknown}, "alias source is invalid before construction")
		return
	}
	if b.externalLocals[name] {
		b.reclassifyMemoryBinding(name, b.conservativeProvenanceFromExpr(expr), "alias source has external or unknown provenance")
	}
}

func (b *builder) recordViewAlias(name string, expr frontend.Expr) {
	id, ok := expr.(*frontend.IdentExpr)
	if !ok || id == nil {
		return
	}
	sourceViewID := valueID(ValueView, id.Name)
	sourceView, ok := b.values[sourceViewID]
	if !ok {
		return
	}
	aliasID := valueID(ValueView, name)
	if _, exists := b.values[aliasID]; !exists {
		alias := sourceView
		alias.ID = aliasID
		alias.Source = sourceString(expr.Pos())
		alias.Lifetime = Lifetime{Birth: sourceString(expr.Pos()), Owner: name}
		b.addValue(alias)
	}
	b.copyDerivedWindowFacts(id.Name, aliasID, "alias preserves derived window range")
	if sourceView.Provenance.Kind == ProvenanceExternal || sourceView.Provenance.Kind == ProvenanceUnknown {
		b.externalLocals[name] = true
		if !b.hasFactForValue(FactProvenanceUnknown, aliasID) {
			b.addFact(Fact{Kind: FactProvenanceUnknown, ValueID: aliasID, Reason: "alias source has external or unknown provenance"})
		}
	}
}

func (b *builder) reclassifyMemoryBinding(name string, provenance Provenance, reason string) {
	if name == "" {
		return
	}
	if provenance.Kind == "" {
		provenance = Provenance{Kind: ProvenanceUnknown}
	}
	if provenance.Kind == ProvenanceExternal && provenance.Root == "" {
		provenance.Root = "external:" + name
	}
	for _, kind := range []ValueKind{ValueLocal, ValueParam} {
		id := valueID(kind, name)
		value, ok := b.values[id]
		if !ok || !isMemoryBackedType(value.Type) {
			continue
		}
		value.Provenance = provenance
		b.values[id] = value
		b.removeFactsForValue(id, FactProvenanceKnown, FactLenStable)
		if !b.hasFactForValue(FactProvenanceUnknown, id) {
			b.addFact(Fact{Kind: FactProvenanceUnknown, ValueID: id, Reason: reason})
		}
	}
}

func (b *builder) removeFactsForValue(valueID string, kinds ...FactKind) {
	if valueID == "" || len(kinds) == 0 {
		return
	}
	remove := map[FactKind]bool{}
	for _, kind := range kinds {
		remove[kind] = true
	}
	filtered := b.facts[:0]
	for _, fact := range b.facts {
		if fact.ValueID == valueID && remove[fact.Kind] {
			continue
		}
		filtered = append(filtered, fact)
	}
	b.facts = filtered
}

func (b *builder) hasFactForValue(kind FactKind, valueID string) bool {
	for _, fact := range b.facts {
		if fact.Kind == kind && fact.ValueID == valueID {
			return true
		}
	}
	return false
}

func (b *builder) conservativeProvenanceFromExpr(expr frontend.Expr) Provenance {
	if b.exprIsInvalidView(expr) {
		return Provenance{Kind: ProvenanceUnknown}
	}
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		for _, kind := range []ValueKind{ValueView, ValueLocal, ValueParam, ValueAllocIntent} {
			value, ok := b.values[valueID(kind, e.Name)]
			if !ok {
				continue
			}
			switch value.Provenance.Kind {
			case ProvenanceExternal:
				root := value.Provenance.Root
				if root == "" {
					root = e.Name
				}
				return Provenance{Kind: ProvenanceExternal, Root: "derived:" + root}
			case ProvenanceUnknown, "":
				return Provenance{Kind: ProvenanceUnknown}
			}
		}
		if b.externalLocals[e.Name] {
			return Provenance{Kind: ProvenanceExternal, Root: "alias:" + e.Name}
		}
	case *frontend.CallExpr:
		name := e.Name
		if target, ok := semantics.ResolveBuiltinAlias(name); ok {
			name = target
		}
		if rawSliceBuiltin(name) {
			return Provenance{Kind: ProvenanceExternal, Root: "raw_parts"}
		}
		if borrowOrViewBuiltin(name) {
			if len(e.Args) == 0 {
				return Provenance{Kind: ProvenanceUnknown}
			}
			return b.conservativeProvenanceFromExpr(e.Args[0])
		}
	}
	return Provenance{Kind: ProvenanceUnknown}
}

func (b *builder) exprHasExternalProvenance(expr frontend.Expr) bool {
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		if b.externalLocals[e.Name] {
			return true
		}
		if value, ok := b.values[valueID(ValueView, e.Name)]; ok {
			return value.Provenance.Kind == ProvenanceExternal || value.Provenance.Kind == ProvenanceUnknown
		}
		return false
	case *frontend.CallExpr:
		name := e.Name
		if target, ok := semantics.ResolveBuiltinAlias(name); ok {
			name = target
		}
		if rawSliceBuiltin(name) {
			return true
		}
		if copyBuiltin(name) {
			return false
		}
		if borrowOrViewBuiltin(name) {
			return len(e.Args) == 0 || b.exprHasExternalProvenance(e.Args[0])
		}
	}
	return false
}

func (b *builder) exprIsInvalidView(expr frontend.Expr) bool {
	if staticInvalidIterableExpr(expr) {
		return true
	}
	switch e := expr.(type) {
	case *frontend.IdentExpr:
		return b.invalidLocals[e.Name]
	case *frontend.CallExpr:
		name := e.Name
		if target, ok := semantics.ResolveBuiltinAlias(name); ok {
			name = target
		}
		if copyBuiltin(name) {
			return false
		}
		if borrowOrViewBuiltin(name) {
			return len(e.Args) > 0 && b.exprIsInvalidView(e.Args[0])
		}
	}
	return false
}

func (b *builder) collectionIterableProofAllowed(expr frontend.Expr) bool {
	if expr == nil {
		return false
	}
	return !b.exprHasExternalProvenance(expr) && !b.exprIsInvalidView(expr)
}

func (b *builder) whileRangeProof(s *frontend.WhileStmt) (rangeProof, bool) {
	proof, ok := b.rangeProofFromCondition(s.Cond, s.At)
	if !ok {
		return rangeProof{}, false
	}
	if b.externalLocals[proof.Base] || b.invalidLocals[proof.Base] {
		return rangeProof{}, false
	}
	if !b.zeroLocals[proof.IndexName] {
		return rangeProof{}, false
	}
	if !b.bodyHasUnitIncrement(s.Body, proof.IndexName) {
		return rangeProof{}, false
	}
	proof.ID = proofIDForRange("while", proof.IndexName, proof.Base, s.At)
	proof.Reason = "while loop range proof"
	return proof, true
}

func (b *builder) ifRangeProof(s *frontend.IfStmt) (rangeProof, bool) {
	proof, ok := b.branchRangeProofFromCondition(s.Cond, s.At)
	if !ok {
		proof, ok = b.rangeProofFromCondition(s.Cond, s.At)
		if !ok || !b.zeroLocals[proof.IndexName] {
			return rangeProof{}, false
		}
	}
	if b.externalLocals[proof.Base] || b.invalidLocals[proof.Base] {
		return rangeProof{}, false
	}
	proof.ID = proofIDForRange("if", proof.IndexName, proof.Base, s.At)
	proof.Reason = "if branch range proof"
	return proof, true
}

func (b *builder) rangeProofFromCondition(cond frontend.Expr, pos frontend.Position) (rangeProof, bool) {
	bin, ok := cond.(*frontend.BinaryExpr)
	if !ok || bin == nil {
		return rangeProof{}, false
	}
	index, ok := bin.Left.(*frontend.IdentExpr)
	if !ok || index == nil {
		return rangeProof{}, false
	}
	base, latticeRange, ok := b.rangeFromCondition(index.Name, bin.Op, bin.Right)
	if !ok || base == "" || !latticeRange.Known {
		return rangeProof{}, false
	}
	indexID := b.valueIDForName(index.Name)
	return rangeProof{
		IndexName:      index.Name,
		IndexValueID:   indexID,
		Base:           base,
		Condition:      exprPath(cond),
		Source:         sourceString(pos),
		RangeText:      rangeTextFromLattice(latticeRange),
		Lower:          plirBoundFromRangeBound(latticeRange.Lower),
		Upper:          plirBoundFromRangeBound(latticeRange.Upper),
		InclusiveLower: latticeRange.InclusiveLower,
		InclusiveUpper: latticeRange.InclusiveUpper,
		Reason:         "range guard proof",
		Derivation:     append([]string(nil), latticeRange.Derivation...),
	}, true
}

func (b *builder) branchRangeProofFromCondition(cond frontend.Expr, pos frontend.Position) (rangeProof, bool) {
	bin, ok := cond.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenAmpAmp {
		return rangeProof{}, false
	}
	if proof, ok := b.branchRangeProofParts(bin.Left, bin.Right, pos); ok {
		return proof, true
	}
	return b.branchRangeProofParts(bin.Right, bin.Left, pos)
}

func (b *builder) branchRangeProofParts(lower frontend.Expr, upper frontend.Expr, pos frontend.Position) (rangeProof, bool) {
	lowerIndex, ok := nonNegativeGuardIndex(lower)
	if !ok {
		return rangeProof{}, false
	}
	proof, ok := b.rangeProofFromCondition(upper, pos)
	if !ok || proof.IndexName != lowerIndex {
		return rangeProof{}, false
	}
	proof.Condition = exprPath(lower) + " && " + exprPath(upper)
	return proof, true
}

func nonNegativeGuardIndex(expr frontend.Expr) (string, bool) {
	bin, ok := expr.(*frontend.BinaryExpr)
	if !ok || bin == nil {
		return "", false
	}
	if left, ok := bin.Left.(*frontend.IdentExpr); ok && left != nil && bin.Op == frontend.TokenGreaterEq && isZeroNumber(bin.Right) {
		return left.Name, true
	}
	if right, ok := bin.Right.(*frontend.IdentExpr); ok && right != nil && bin.Op == frontend.TokenLessEq && isZeroNumber(bin.Left) {
		return right.Name, true
	}
	return "", false
}

func isZeroNumber(expr frontend.Expr) bool {
	num, ok := expr.(*frontend.NumberExpr)
	return ok && num != nil && num.Value == 0
}

func (b *builder) valueIDForName(name string) string {
	for _, kind := range []ValueKind{ValueLocal, ValueLoopIndex, ValueParam, ValueView, ValueAllocIntent} {
		id := valueID(kind, name)
		if _, ok := b.values[id]; ok {
			return id
		}
	}
	return valueID(ValueLocal, name)
}

func (b *builder) rangeUpperFromCondition(op frontend.TokenType, right frontend.Expr) (string, Bound, bool, bool) {
	switch op {
	case frontend.TokenLess, frontend.TokenBangEq:
		base := b.lenBoundBase(right)
		if base == "" {
			return "", Bound{}, false, false
		}
		return base, Bound{Kind: BoundSymbol, Symbol: base + ".len"}, false, true
	case frontend.TokenLessEq:
		base := lenMinusOneBase(right)
		if base == "" {
			return "", Bound{}, false, false
		}
		return base, Bound{Kind: BoundSymbolMinus, Symbol: base + ".len", Const: 1}, true, true
	default:
		return "", Bound{}, false, false
	}
}

func (b *builder) rangeFromCondition(indexName string, op frontend.TokenType, right frontend.Expr) (string, rangeproof.Range, bool) {
	switch op {
	case frontend.TokenLess, frontend.TokenBangEq:
		base := b.lenBoundBase(right)
		if base == "" {
			return "", rangeproof.Range{}, false
		}
		return base, rangeproof.LessThanLen(indexName, base), true
	case frontend.TokenLessEq:
		base := lenMinusOneBase(right)
		if base == "" {
			return "", rangeproof.Range{}, false
		}
		return base, rangeproof.LessEqualLenMinusOne(indexName, base), true
	default:
		return "", rangeproof.Range{}, false
	}
}

func (b *builder) lenBoundBase(expr frontend.Expr) string {
	if base := lenFieldBase(expr); base != "" {
		return base
	}
	id, ok := expr.(*frontend.IdentExpr)
	if !ok || id == nil {
		return ""
	}
	return b.lenBoundLocals[id.Name]
}

func rangeUpperFromCondition(op frontend.TokenType, right frontend.Expr) (string, Bound, bool, bool) {
	switch op {
	case frontend.TokenLess, frontend.TokenBangEq:
		base := lenFieldBase(right)
		if base == "" {
			return "", Bound{}, false, false
		}
		return base, Bound{Kind: BoundSymbol, Symbol: base + ".len"}, false, true
	case frontend.TokenLessEq:
		base := lenMinusOneBase(right)
		if base == "" {
			return "", Bound{}, false, false
		}
		return base, Bound{Kind: BoundSymbolMinus, Symbol: base + ".len", Const: 1}, true, true
	default:
		return "", Bound{}, false, false
	}
}

func lenFieldBase(expr frontend.Expr) string {
	field, ok := expr.(*frontend.FieldAccessExpr)
	if !ok || field == nil || field.Field != "len" {
		return ""
	}
	return exprPath(field.Base)
}

func lenMinusOneBase(expr frontend.Expr) string {
	bin, ok := expr.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenMinus {
		return ""
	}
	num, ok := bin.Right.(*frontend.NumberExpr)
	if !ok || num == nil || num.Value != 1 {
		return ""
	}
	return lenFieldBase(bin.Left)
}

func (b *builder) rememberLocalProofMetadata(name string, expr frontend.Expr) {
	if value, ok := b.proofConstIntValue(expr); ok {
		b.zeroLocals[name] = value == 0
		b.constIntLocals[name] = value
	} else {
		b.zeroLocals[name] = isZeroExpr(expr)
		delete(b.constIntLocals, name)
	}
	if base := lenFieldBase(expr); base != "" {
		b.lenBoundLocals[name] = base
	} else {
		delete(b.lenBoundLocals, name)
	}
}

func (b *builder) bodyHasUnitIncrement(stmts []frontend.Stmt, indexName string) bool {
	for _, stmt := range stmts {
		assign, ok := stmt.(*frontend.AssignStmt)
		if !ok || assign == nil {
			continue
		}
		target, ok := assign.Target.(*frontend.IdentExpr)
		if !ok || target.Name != indexName {
			continue
		}
		if b.isUnitIncrement(assign.Value, indexName) {
			return true
		}
	}
	return false
}

func (b *builder) isUnitIncrement(expr frontend.Expr, indexName string) bool {
	bin, ok := expr.(*frontend.BinaryExpr)
	if !ok || bin == nil || bin.Op != frontend.TokenPlus {
		return false
	}
	if left, ok := bin.Left.(*frontend.IdentExpr); ok && left.Name == indexName {
		return b.isUnitStepExpr(bin.Right)
	}
	if right, ok := bin.Right.(*frontend.IdentExpr); ok && right.Name == indexName {
		return b.isUnitStepExpr(bin.Left)
	}
	return false
}

func (b *builder) isUnitStepExpr(expr frontend.Expr) bool {
	if num, ok := expr.(*frontend.NumberExpr); ok && num != nil {
		return num.Value == 1
	}
	id, ok := expr.(*frontend.IdentExpr)
	if !ok || id == nil {
		return false
	}
	info, ok := b.fn.Locals[id.Name]
	if !ok || info.Mutable {
		return false
	}
	value, ok := b.constIntLocals[id.Name]
	return ok && value == 1
}

func (b *builder) proofConstIntValue(expr frontend.Expr) (int64, bool) {
	if value, ok := evalConstInt64(expr); ok {
		return value, true
	}
	id, ok := expr.(*frontend.IdentExpr)
	if !ok || id == nil {
		return 0, false
	}
	info, ok := b.fn.Locals[id.Name]
	if !ok || info.Mutable {
		return 0, false
	}
	value, ok := b.constIntLocals[id.Name]
	return value, ok
}

func isZeroExpr(expr frontend.Expr) bool {
	num, ok := expr.(*frontend.NumberExpr)
	return ok && num != nil && num.Value == 0
}

func rangeText(indexName string, base string, upper Bound, inclusiveUpper bool) string {
	if upper.Kind == BoundSymbolMinus {
		return fmt.Sprintf("%s in [0, %s - %d]", indexName, upper.Symbol, upper.Const)
	}
	if inclusiveUpper {
		return fmt.Sprintf("%s in [0, %s]", indexName, base+".len")
	}
	return fmt.Sprintf("%s in [0, %s)", indexName, base+".len")
}

func rangeTextFromLattice(r rangeproof.Range) string {
	upper := plirBoundFromRangeBound(r.Upper)
	base := ""
	switch r.Upper.Kind {
	case rangeproof.BoundSymbol, rangeproof.BoundSymbolMinus, rangeproof.BoundSymbolPlus:
		base = strings.TrimSuffix(r.Upper.Symbol, ".len")
	}
	return rangeText(r.Value, base, upper, r.InclusiveUpper)
}

func plirBoundFromRangeBound(bound rangeproof.Bound) Bound {
	switch bound.Kind {
	case rangeproof.BoundConst:
		return Bound{Kind: BoundConst, Const: bound.Const}
	case rangeproof.BoundSymbol:
		return Bound{Kind: BoundSymbol, Symbol: bound.Symbol}
	case rangeproof.BoundSymbolMinus:
		return Bound{Kind: BoundSymbolMinus, Symbol: bound.Symbol, Const: bound.Const}
	default:
		return Bound{Kind: BoundUnknown}
	}
}

func proofIDForRange(kind string, indexName string, base string, pos frontend.Position) string {
	base = proofNamePart(base, "value")
	return fmt.Sprintf("proof:%s:%s:%s:%d:%d", kind, indexName, base, pos.Line, pos.Col)
}

func copyLoopProofID(name string, pos frontend.Position) string {
	return fmt.Sprintf("proof:copy-loop:%s:%d:%d", proofNamePart(name, "copy"), pos.Line, pos.Col)
}

func proofNamePart(name string, fallback string) string {
	name = strings.NewReplacer(".", "_", " ", "_").Replace(name)
	if name == "" {
		return fallback
	}
	return name
}

func forCollectionProofID(stmt *frontend.ForRangeStmt) string {
	kind := "for-collection"
	if isViewIterable(stmt.Iterable) {
		kind = "for-collection-view"
	}
	return fmt.Sprintf("proof:%s:%s:%d:%d", kind, stmt.Name, stmt.At.Line, stmt.At.Col)
}

func isViewIterable(expr frontend.Expr) bool {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false
	}
	name := call.Name
	if builtin, ok := semantics.ResolveBuiltinAlias(name); ok {
		name = builtin
	}
	return strings.HasPrefix(name, "core.slice_window_") ||
		strings.HasPrefix(name, "core.slice_prefix_") ||
		strings.HasPrefix(name, "core.slice_suffix_") ||
		name == "core.string_window" ||
		name == "core.string_prefix" ||
		name == "core.string_suffix"
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func valueID(kind ValueKind, name string) string {
	if name == "" {
		name = "anon"
	}
	return string(kind) + ":" + name
}

func valueIDsForPath(path string) []string {
	return []string{
		valueID(ValueView, path),
		valueID(ValueAllocIntent, path),
		valueID(ValueLocal, path),
		valueID(ValueParam, path),
	}
}

func makeSliceElem(name string) (string, bool) {
	switch name {
	case "core.make_u8", "core.island_make_u8":
		return "u8", true
	case "core.make_u16", "core.island_make_u16":
		return "u16", true
	case "core.make_i32", "core.island_make_i32":
		return "i32", true
	case "core.make_bool", "core.island_make_bool":
		return "bool", true
	default:
		return "", false
	}
}

func rawSliceElem(name string) (string, bool) {
	switch name {
	case "core.raw_slice_u8_from_parts":
		return "u8", true
	case "core.raw_slice_u16_from_parts":
		return "u16", true
	case "core.raw_slice_i32_from_parts":
		return "i32", true
	case "core.raw_slice_bool_from_parts":
		return "bool", true
	default:
		return "", false
	}
}

func rawSliceBuiltin(name string) bool {
	_, ok := rawSliceElem(name)
	return ok
}

func rawMemoryAccessBuiltin(name string) bool {
	switch name {
	case "core.load_i32", "core.store_i32",
		"core.load_u8", "core.store_u8",
		"core.load_ptr", "core.store_ptr", "core.store_arch_ptr":
		return true
	default:
		return false
	}
}

func rawMemoryAccessWidthBytes(name string) int64 {
	switch name {
	case "core.load_i32", "core.store_i32":
		return 4
	case "core.load_ptr", "core.store_ptr", "core.store_arch_ptr":
		// MPC-8 runtime evidence is linux-x64; other targets stay build/lower scoped.
		return 8
	default:
		return 1
	}
}

func rawPointerRoot(path string) string {
	if !strings.HasSuffix(path, ".ptr") {
		return ""
	}
	rootPath := strings.TrimSuffix(path, ".ptr")
	if rootPath == "" || rootPath == "?" || rootPath == "expr" {
		return ""
	}
	if dot := strings.Index(rootPath, "."); dot >= 0 {
		return rootPath[:dot]
	}
	return rootPath
}

func sliceBorrowElem(name string) (string, bool) {
	if !strings.HasPrefix(name, "core.slice_borrow_") {
		return "", false
	}
	elem := strings.TrimPrefix(name, "core.slice_borrow_")
	switch elem {
	case "u8", "u16", "i32", "bool":
		return elem, true
	default:
		return "", false
	}
}

func sliceCopyElem(name string) (string, bool) {
	if !strings.HasPrefix(name, "core.slice_copy_") || strings.HasPrefix(name, "core.slice_copy_into_") {
		return "", false
	}
	elem := strings.TrimPrefix(name, "core.slice_copy_")
	switch elem {
	case "u8", "u16", "i32", "bool":
		return elem, true
	default:
		return "", false
	}
}

func sliceCopyIntoBuiltin(name string) bool {
	if !strings.HasPrefix(name, "core.slice_copy_into_") {
		return false
	}
	switch strings.TrimPrefix(name, "core.slice_copy_into_") {
	case "u8", "u16", "i32", "bool":
		return true
	default:
		return false
	}
}

func stringBorrowBuiltin(name string) bool {
	return name == "core.string_borrow"
}

func stringCopyBuiltin(name string) bool {
	return name == "core.string_copy"
}

func stringCopyIntoBuiltin(name string) bool {
	return name == "core.string_copy_into"
}

func copyBuiltin(name string) bool {
	if stringCopyBuiltin(name) || sliceCopyElemName(name) || stringCopyIntoBuiltin(name) || sliceCopyIntoBuiltin(name) {
		return true
	}
	return false
}

func sliceCopyElemName(name string) bool {
	_, ok := sliceCopyElem(name)
	return ok
}

func borrowOrViewBuiltin(name string) bool {
	if stringBorrowBuiltin(name) {
		return true
	}
	if _, ok := sliceBorrowElem(name); ok {
		return true
	}
	if _, _, ok := sliceViewElem(name); ok {
		return true
	}
	if _, _, ok := stringViewBuiltin(name); ok {
		return true
	}
	return false
}

func sliceViewElem(name string) (elem string, method string, ok bool) {
	if !strings.HasPrefix(name, "core.slice_") {
		return "", "", false
	}
	rest := strings.TrimPrefix(name, "core.slice_")
	for _, candidate := range []string{"window", "prefix", "suffix"} {
		prefix := candidate + "_"
		if !strings.HasPrefix(rest, prefix) {
			continue
		}
		elem = strings.TrimPrefix(rest, prefix)
		switch elem {
		case "u8", "u16", "i32", "bool":
			return elem, candidate, true
		default:
			return "", "", false
		}
	}
	return "", "", false
}

func stringViewBuiltin(name string) (valueType string, method string, ok bool) {
	if !strings.HasPrefix(name, "core.string_") {
		return "", "", false
	}
	method = strings.TrimPrefix(name, "core.string_")
	switch method {
	case "window", "prefix", "suffix":
		return "str", method, true
	default:
		return "", "", false
	}
}

type derivedWindowRange struct {
	base  string
	start string
	end   string
}

func (b *builder) sliceViewRange(method string, source string, call *frontend.CallExpr) string {
	if parent, ok := b.derivedWindowRangeForSource(source); ok {
		return composeSliceViewRange(parent, method, call)
	}
	return baseSliceViewRange(method, source, call)
}

func (b *builder) derivedWindowRangeForSource(source string) (derivedWindowRange, bool) {
	if source == "" {
		return derivedWindowRange{}, false
	}
	candidates := []string{
		valueID(ValueView, source),
		valueID(ValueAllocIntent, source),
		valueID(ValueLocal, source),
		valueID(ValueParam, source),
	}
	for _, candidate := range candidates {
		for _, fact := range b.facts {
			if fact.Kind != FactDerivedWindow || fact.ValueID != candidate {
				continue
			}
			return parseDerivedWindowRange(fact.Range)
		}
	}
	return derivedWindowRange{}, false
}

func parseDerivedWindowRange(text string) (derivedWindowRange, bool) {
	start := strings.LastIndex(text, "[")
	end := strings.LastIndex(text, "]")
	if start < 0 || end <= start {
		return derivedWindowRange{}, false
	}
	parts := strings.Split(text[start+1:end], "..")
	if len(parts) != 2 {
		return derivedWindowRange{}, false
	}
	base := strings.TrimSpace(text[:start])
	lo := strings.TrimSpace(parts[0])
	hi := strings.TrimSpace(parts[1])
	if base == "" || lo == "" || hi == "" {
		return derivedWindowRange{}, false
	}
	return derivedWindowRange{base: base, start: lo, end: hi}, true
}

func (b *builder) copyIntoOverlapStatus(source string, destination string) string {
	sourceRange, sourceRangeOK := b.derivedWindowRangeForSource(source)
	destinationRange, destinationRangeOK := b.derivedWindowRangeForSource(destination)
	if sourceRangeOK && destinationRangeOK {
		sourceBase := normalizeOverlapRoot(sourceRange.base)
		destinationBase := normalizeOverlapRoot(destinationRange.base)
		if sourceBase == "" || destinationBase == "" {
			return "unknown_conservative"
		}
		if sourceBase == destinationBase {
			if overlap, ok := staticDerivedRangesOverlap(sourceRange, destinationRange); ok {
				if overlap {
					return "known_overlap"
				}
				return "known_disjoint"
			}
			return "unknown_conservative"
		}
		return "distinct_roots"
	}
	sourceRoot, sourceKnown := b.copyIntoKnownRoot(source)
	destinationRoot, destinationKnown := b.copyIntoKnownRoot(destination)
	if !sourceKnown || !destinationKnown || sourceRoot == "" || destinationRoot == "" {
		return "unknown_conservative"
	}
	if sourceRoot == destinationRoot {
		return "unknown_conservative"
	}
	return "distinct_roots"
}

func (b *builder) copyIntoKnownRoot(path string) (string, bool) {
	provenance, known := b.derivedProvenance(path)
	if !known {
		return "", false
	}
	root := normalizeOverlapRoot(provenance.Root)
	if root == "" {
		root = normalizeOverlapRoot(path)
	}
	if root == "" || root == "?" || root == "expr" {
		return "", false
	}
	return root, true
}

func staticDerivedRangesOverlap(source derivedWindowRange, destination derivedWindowRange) (bool, bool) {
	sourceStart, ok := parseRangeConst(source.start)
	if !ok {
		return false, false
	}
	sourceEnd, ok := parseRangeConst(source.end)
	if !ok {
		return false, false
	}
	destinationStart, ok := parseRangeConst(destination.start)
	if !ok {
		return false, false
	}
	destinationEnd, ok := parseRangeConst(destination.end)
	if !ok {
		return false, false
	}
	return sourceStart < destinationEnd && destinationStart < sourceEnd, true
}

func parseRangeConst(text string) (int64, bool) {
	value, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	return value, err == nil
}

func normalizeOverlapRoot(root string) string {
	root = strings.TrimSpace(root)
	for strings.HasPrefix(root, "derived:") {
		root = strings.TrimPrefix(root, "derived:")
	}
	for _, prefix := range []string{"param:", "local:", "view:", "alloc_intent:"} {
		root = strings.TrimPrefix(root, prefix)
	}
	if dot := strings.Index(root, "."); dot > 0 {
		root = root[:dot]
	}
	return root
}

func composeSliceViewRange(parent derivedWindowRange, method string, call *frontend.CallExpr) string {
	switch method {
	case "window":
		start := callArgPath(call, 1)
		count := callArgPath(call, 2)
		lo := addRangeExpr(parent.start, start)
		return fmt.Sprintf("%s[%s..%s]", parent.base, lo, addRangeExpr(lo, count))
	case "prefix":
		count := callArgPath(call, 1)
		return fmt.Sprintf("%s[%s..%s]", parent.base, parent.start, addRangeExpr(parent.start, count))
	case "suffix":
		start := callArgPath(call, 1)
		return fmt.Sprintf("%s[%s..%s]", parent.base, addRangeExpr(parent.start, start), parent.end)
	default:
		return fmt.Sprintf("%s[%s..%s]", parent.base, parent.start, parent.end)
	}
}

func addRangeExpr(left string, right string) string {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" || left == "0" {
		return right
	}
	if right == "" || right == "0" {
		return left
	}
	var constSum int64
	terms := []string{}
	addTerm := func(term string) {
		term = strings.TrimSpace(term)
		if term == "" || term == "0" {
			return
		}
		if value, err := strconv.ParseInt(term, 10, 64); err == nil {
			constSum += value
			return
		}
		terms = append(terms, term)
	}
	for _, part := range strings.Split(left, "+") {
		addTerm(part)
	}
	for _, part := range strings.Split(right, "+") {
		addTerm(part)
	}
	out := []string{}
	if constSum != 0 {
		out = append(out, strconv.FormatInt(constSum, 10))
	}
	out = append(out, terms...)
	if len(out) == 0 {
		return "0"
	}
	return strings.Join(out, "+")
}

func baseSliceViewRange(method string, source string, call *frontend.CallExpr) string {
	if source == "" {
		source = "source"
	}
	switch method {
	case "window":
		start := callArgPath(call, 1)
		count := callArgPath(call, 2)
		return fmt.Sprintf("%s[%s..%s]", source, start, addRangeExpr(start, count))
	case "prefix":
		count := callArgPath(call, 1)
		return fmt.Sprintf("%s[0..%s]", source, count)
	case "suffix":
		start := callArgPath(call, 1)
		return fmt.Sprintf("%s[%s..len]", source, start)
	default:
		return source + "[view]"
	}
}

func staticInvalidStringViewCall(name string, call *frontend.CallExpr) bool {
	if call == nil {
		return false
	}
	if target, aliasOK := semantics.ResolveBuiltinAlias(name); aliasOK {
		name = target
	}
	if !strings.HasPrefix(name, "core.string_") {
		return false
	}
	sourceLen, knownLen := staticStringByteLen(callArg(call, 0))
	if !knownLen {
		return false
	}
	switch name {
	case "core.string_window":
		if len(call.Args) != 3 {
			return false
		}
		start, startKnown := evalConstInt64(call.Args[1])
		count, countKnown := evalConstInt64(call.Args[2])
		if !startKnown || !countKnown {
			return false
		}
		return start < 0 || count < 0 || start > sourceLen || count > sourceLen-start
	case "core.string_prefix":
		if len(call.Args) != 2 {
			return false
		}
		count, known := evalConstInt64(call.Args[1])
		if !known {
			return false
		}
		return count < 0 || count > sourceLen
	case "core.string_suffix":
		if len(call.Args) != 2 {
			return false
		}
		start, known := evalConstInt64(call.Args[1])
		if !known {
			return false
		}
		return start < 0 || start > sourceLen
	default:
		return false
	}
}

func staticStringByteLen(expr frontend.Expr) (int64, bool) {
	lit, ok := expr.(*frontend.StringLitExpr)
	if !ok || lit == nil {
		return 0, false
	}
	return int64(len(lit.Value)), true
}

func elementSize(elem string) int {
	switch elem {
	case "u8":
		return 1
	case "u16":
		return 2
	case "i32", "bool":
		return 4
	case "raw_bytes":
		return 1
	default:
		return 0
	}
}

func sliceViewElementLayout(valueType string) (int, int) {
	switch valueType {
	case "str", "String", "[]u8":
		return 1, 0
	case "[]u16":
		return 2, 1
	case "[]i32", "[]bool":
		return 4, 2
	default:
		return 0, 0
	}
}

func allocationLengthArg(name string, call *frontend.CallExpr) frontend.Expr {
	if call == nil {
		return nil
	}
	index := 0
	if strings.HasPrefix(name, "core.island_make_") {
		index = 1
	}
	if index < 0 || index >= len(call.Args) {
		return nil
	}
	return call.Args[index]
}

func callArg(call *frontend.CallExpr, index int) frontend.Expr {
	if call == nil || index < 0 || index >= len(call.Args) {
		return nil
	}
	return call.Args[index]
}

func callResultPath(call *frontend.CallExpr) string {
	if call == nil {
		return ""
	}
	name := call.Name
	if target, ok := semantics.ResolveBuiltinAlias(name); ok {
		name = target
	}
	if _, ok := makeSliceElem(name); ok {
		return syntheticCallPath("alloc", call)
	}
	if name == "core.alloc_bytes" {
		return syntheticCallPath("raw_alloc", call)
	}
	if rawSliceBuiltin(name) {
		return syntheticCallPath("raw_view", call)
	}
	if _, ok := sliceCopyElem(name); ok || stringCopyBuiltin(name) {
		return syntheticCallPath("copy", call)
	}
	if _, ok := sliceBorrowElem(name); ok || stringBorrowBuiltin(name) {
		return syntheticCallPath("borrow", call)
	}
	if _, _, ok := sliceViewElem(name); ok {
		return syntheticCallPath("slice_view", call)
	}
	if _, _, ok := stringViewBuiltin(name); ok {
		return syntheticCallPath("slice_view", call)
	}
	return ""
}

func exprStoresDirectlyIntoTarget(expr frontend.Expr) bool {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false
	}
	return callResultPath(call) != ""
}

func syntheticCallPath(prefix string, call *frontend.CallExpr) string {
	if call == nil || (call.At.Line == 0 && call.At.Col == 0) {
		return prefix
	}
	return fmt.Sprintf("%s_%d_%d", prefix, call.At.Line, call.At.Col)
}

func firstArgPath(call *frontend.CallExpr) string {
	if call == nil || len(call.Args) == 0 {
		return ""
	}
	return exprPath(call.Args[0])
}

func callArgPath(call *frontend.CallExpr, index int) string {
	if call == nil || index < 0 || index >= len(call.Args) {
		return "?"
	}
	path := exprPath(call.Args[index])
	if path == "" {
		return "expr"
	}
	return path
}

func callInputs(args []frontend.Expr) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		if input := exprPath(arg); input != "" {
			out = append(out, input)
		}
	}
	return out
}

func isMemoryBackedType(typeName string) bool {
	return strings.HasPrefix(typeName, "[]") || typeName == "str" || typeName == "String"
}

func sourceString(pos frontend.Position) string {
	if pos.Line == 0 && pos.Col == 0 && pos.File == "" {
		return ""
	}
	return frontend.FormatPos(pos)
}

func exprPath(expr frontend.Expr) string {
	switch e := expr.(type) {
	case nil:
		return ""
	case *frontend.IdentExpr:
		return e.Name
	case *frontend.FieldAccessExpr:
		base := exprPath(e.Base)
		if base == "" {
			return e.Field
		}
		return base + "." + e.Field
	case *frontend.NumberExpr:
		return fmt.Sprintf("%d", e.Value)
	case *frontend.StringLitExpr:
		return stringLiteralPath(string(e.Value))
	case *frontend.CallExpr:
		return callResultPath(e)
	case *frontend.UnaryExpr:
		x := exprPath(e.X)
		if x == "" {
			return ""
		}
		switch e.Op {
		case frontend.TokenMinus:
			return "-" + x
		default:
			return ""
		}
	case *frontend.BinaryExpr:
		left := exprPath(e.Left)
		right := exprPath(e.Right)
		if left == "" || right == "" {
			return ""
		}
		op := plirTokenString(e.Op)
		if op == "" {
			return ""
		}
		return left + " " + op + " " + right
	default:
		return ""
	}
}

func stringLiteralPath(value string) string {
	part := strings.NewReplacer(
		" ", "_",
		"\t", "_",
		"\n", "_",
		"\r", "_",
		".", "_",
		":", "_",
		"/", "_",
		"\\", "_",
		"\"", "_",
		"'", "_",
	).Replace(value)
	if part == "" {
		part = "empty"
	}
	if len(part) > 32 {
		part = part[:32]
	}
	return fmt.Sprintf("string:%d:%s", len(value), part)
}

func evalConstInt64(expr frontend.Expr) (int64, bool) {
	switch e := expr.(type) {
	case nil:
		return 0, false
	case *frontend.NumberExpr:
		return int64(e.Value), true
	case *frontend.UnaryExpr:
		v, ok := evalConstInt64(e.X)
		if !ok {
			return 0, false
		}
		switch e.Op {
		case frontend.TokenMinus:
			return -v, true
		default:
			return 0, false
		}
	case *frontend.BinaryExpr:
		left, ok := evalConstInt64(e.Left)
		if !ok {
			return 0, false
		}
		right, ok := evalConstInt64(e.Right)
		if !ok {
			return 0, false
		}
		switch e.Op {
		case frontend.TokenPlus:
			return left + right, true
		case frontend.TokenMinus:
			return left - right, true
		case frontend.TokenStar:
			return left * right, true
		case frontend.TokenSlash:
			if right == 0 {
				return 0, false
			}
			return left / right, true
		case frontend.TokenPercent:
			if right == 0 {
				return 0, false
			}
			return left % right, true
		default:
			return 0, false
		}
	default:
		return 0, false
	}
}

func checkedAddInt64(left int64, right int64) (int64, bool) {
	if right > 0 && left > math.MaxInt64-right {
		return 0, false
	}
	if right < 0 && left < math.MinInt64-right {
		return 0, false
	}
	return left + right, true
}

func plirTokenString(op frontend.TokenType) string {
	switch op {
	case frontend.TokenPlus:
		return "+"
	case frontend.TokenMinus:
		return "-"
	case frontend.TokenStar:
		return "*"
	case frontend.TokenSlash:
		return "/"
	case frontend.TokenPercent:
		return "%"
	case frontend.TokenLess:
		return "<"
	case frontend.TokenLessEq:
		return "<="
	case frontend.TokenGreater:
		return ">"
	case frontend.TokenGreaterEq:
		return ">="
	case frontend.TokenEqEq:
		return "=="
	case frontend.TokenBangEq:
		return "!="
	default:
		return ""
	}
}

func staticInvalidAllocationExpr(expr frontend.Expr) bool {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false
	}
	name := call.Name
	elem, ok := makeSliceElem(name)
	if !ok {
		if target, aliasOK := semantics.ResolveBuiltinAlias(name); aliasOK {
			name = target
			elem, ok = makeSliceElem(name)
		}
	}
	if !ok {
		return false
	}
	length, known := evalConstInt64(allocationLengthArg(name, call))
	if !known {
		return false
	}
	if length < 0 {
		return true
	}
	size := int64(elementSize(elem))
	return size > 0 && length*size > 2147483647
}

func staticInvalidIterableExpr(expr frontend.Expr) bool {
	return staticInvalidAllocationExpr(expr) || staticInvalidStringViewCallExpr(expr)
}

func staticInvalidStringViewCallExpr(expr frontend.Expr) bool {
	call, ok := expr.(*frontend.CallExpr)
	if !ok || call == nil {
		return false
	}
	return staticInvalidStringViewCall(call.Name, call)
}

func FormatText(prog *Program) string {
	if prog == nil {
		return ""
	}
	var b strings.Builder
	for _, fn := range prog.Funcs {
		fmt.Fprintf(&b, "func %s\n", fn.Name)
		for _, value := range fn.Values {
			fmt.Fprintf(&b, "  value %s: %s", value.ID, value.Type)
			if value.Provenance.Kind != "" {
				fmt.Fprintf(&b, " provenance: %s", value.Provenance.Root)
				if value.Provenance.Root == "" {
					fmt.Fprintf(&b, " provenance: %s", value.Provenance.Kind)
				}
			}
			if value.Region != "" {
				fmt.Fprintf(&b, " region: %s", value.Region)
			}
			fmt.Fprintln(&b)
		}
		for _, fact := range fn.Facts {
			fmt.Fprintf(&b, "  fact %s", fact.Kind)
			if fact.ValueID != "" {
				fmt.Fprintf(&b, " value: %s", fact.ValueID)
			}
			if fact.Range != "" {
				fmt.Fprintf(&b, " range: %s", fact.Range)
			}
			if fact.IslandID != "" {
				fmt.Fprintf(&b, " island: %s epoch: %d base: %s", fact.IslandID, fact.Epoch, fact.BaseID)
			}
			if fact.ProofID != "" {
				fmt.Fprintf(&b, " proof: %s", fact.ProofID)
			}
			if fact.Reason != "" {
				fmt.Fprintf(&b, " reason: %s", fact.Reason)
			}
			fmt.Fprintln(&b)
		}
		for _, rf := range fn.RangeFacts {
			fmt.Fprintf(&b, "  range %s lower: %s upper: %s proof: %s", rf.Value, formatBound(rf.Lower), formatBound(rf.Upper), rf.ProofID)
			if rf.Reason != "" {
				fmt.Fprintf(&b, " reason: %s", rf.Reason)
			}
			if len(rf.Derivation) > 0 {
				fmt.Fprintf(&b, " derivation: %s", strings.Join(rf.Derivation, ","))
			}
			fmt.Fprintln(&b)
		}
		for _, block := range fn.Blocks {
			fmt.Fprintf(&b, "  block %s", block.ID)
			if block.Entry {
				fmt.Fprintf(&b, " entry")
			}
			if block.Exit {
				fmt.Fprintf(&b, " exit")
			}
			if len(block.Succs) > 0 {
				fmt.Fprintf(&b, " succs: %s", strings.Join(block.Succs, ","))
			}
			fmt.Fprintln(&b)
		}
		for _, guard := range fn.ProofGuards {
			fmt.Fprintf(&b, "  proof %s kind: %s block: %s guard: %s", guard.ID, guard.Kind, guard.Block, guard.Condition)
			if guard.Reason != "" {
				fmt.Fprintf(&b, " reason: %s", guard.Reason)
			}
			fmt.Fprintln(&b)
		}
		for _, term := range fn.ProofTerms {
			fmt.Fprintf(&b, "  proof_term %s kind: %s subject: %s index: %s op: %s range: %s", term.ID, term.Kind, term.SubjectBaseID, term.IndexValueID, term.Operation, term.Range)
			if term.IslandID != "" {
				fmt.Fprintf(&b, " island: %s epoch: %d base: %s", term.IslandID, term.Epoch, term.BaseID)
			}
			fmt.Fprintln(&b)
		}
		for _, op := range fn.Ops {
			fmt.Fprintf(&b, "  op %s %s", op.ID, op.Kind)
			if op.Block != "" {
				fmt.Fprintf(&b, " block: %s", op.Block)
			}
			if op.Note != "" {
				fmt.Fprintf(&b, " %s", op.Note)
			}
			fmt.Fprintln(&b)
		}
	}
	return b.String()
}

func formatBound(bound Bound) string {
	switch bound.Kind {
	case BoundConst:
		return fmt.Sprintf("%d", bound.Const)
	case BoundSymbol:
		return bound.Symbol
	case BoundSymbolMinus:
		return fmt.Sprintf("%s-%d", bound.Symbol, bound.Const)
	default:
		return string(bound.Kind)
	}
}
