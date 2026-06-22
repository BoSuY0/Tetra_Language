package memoryfacts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type Snapshot struct {
	programID    string
	facts        []Fact
	byID         map[FactID]Fact
	byValue      map[ValueKey][]FactID
	byAllocation map[AllocationKey][]FactID
	byProof      map[ProofKey][]FactID
	byParent     map[FactID][]FactID
}

type AllocationEvidence struct {
	FunctionID         string
	ValueID            string
	AllocationSiteID   string
	SiteID             string
	SourceSpan         string
	TypeName           string
	ProvenanceClass    ProvenanceClass
	UnsafeClass        UnsafeClass
	BorrowState        BorrowState
	EscapeState        EscapeState
	AliasState         AliasState
	RegionID           string
	IslandID           string
	Epoch              int
	OwnerID            string
	DomainKind         DomainKind
	DomainID           string
	DomainOwnerID      string
	TransferKind       TransferKind
	TransferProofID    string
	SourceConsumed     bool
	LiveBorrowCrossing bool
	DestinationActive  bool
	LifetimeBirth      string
	LifetimeDeath      string
	LifetimeOwner      string
	NoEscapeProofID    string
	NoEscapeReason     string
	StorageProofID     string
	SourceFactIDs      []FactID
}

type ProofQuery struct {
	FunctionID    string
	ProofID       string
	Kind          ProofKind
	SubjectBaseID string
	Operation     string
	IslandID      string
	Epoch         int
}

type ProofEvidence struct {
	FactID        FactID
	ProofID       string
	Kind          ProofKind
	SubjectBaseID string
	Operation     string
	IslandID      string
	Epoch         int
	ValidatorName string
	SourceStage   SourceStage
}

type Invalidation struct {
	FactID FactID
	Reason string
}

type ArtifactAttachment struct {
	FactID     FactID
	ArtifactID string
}

type ValidationMark struct {
	FactID        FactID
	ValidatorName string
}

type Delta struct {
	Stage      SourceStage
	Add        []Fact
	Invalidate []Invalidation
	Attach     []ArtifactAttachment
	Validate   []ValidationMark
}

func (g *Graph) Snapshot() (Snapshot, error) {
	if g == nil {
		return Snapshot{}, fmt.Errorf("memoryfacts: nil graph")
	}
	if err := g.Validate(); err != nil {
		return Snapshot{}, err
	}
	s := Snapshot{
		programID:    g.programID,
		facts:        make([]Fact, 0, len(g.order)),
		byID:         map[FactID]Fact{},
		byValue:      map[ValueKey][]FactID{},
		byAllocation: map[AllocationKey][]FactID{},
		byProof:      map[ProofKey][]FactID{},
		byParent:     map[FactID][]FactID{},
	}
	for _, id := range g.order {
		f := cloneFact(g.facts[id])
		s.facts = append(s.facts, f)
		s.byID[f.ID] = f
		if f.FunctionID != "" && f.ValueID != "" {
			key := ValueKey{FunctionID: f.FunctionID, ValueID: f.ValueID}
			s.byValue[key] = append(s.byValue[key], f.ID)
		}
		if f.FunctionID != "" && f.AllocationSiteID != "" {
			key := AllocationKey{FunctionID: f.FunctionID, AllocationSiteID: f.AllocationSiteID}
			s.byAllocation[key] = append(s.byAllocation[key], f.ID)
		}
		if f.FunctionID != "" && f.ProofID != "" {
			key := ProofKey{FunctionID: f.FunctionID, ProofID: f.ProofID}
			s.byProof[key] = append(s.byProof[key], f.ID)
		}
		if f.ParentFactID != "" {
			s.byParent[f.ParentFactID] = append(s.byParent[f.ParentFactID], f.ID)
		}
	}
	return s, nil
}

func (s Snapshot) ProgramID() string {
	return s.programID
}

func (s Snapshot) Facts() []Fact {
	out := make([]Fact, len(s.facts))
	for i, f := range s.facts {
		out[i] = cloneFact(f)
	}
	return out
}

func (s Snapshot) Fact(id FactID) (Fact, bool) {
	f, ok := s.byID[id]
	return cloneFact(f), ok
}

func (s Snapshot) FactsForValue(key ValueKey) []Fact {
	return s.factsForIDs(s.byValue[key])
}

func (s Snapshot) FactsForAllocation(key AllocationKey) []Fact {
	return s.factsForIDs(s.byAllocation[key])
}

func (s Snapshot) FactsForProof(key ProofKey) []Fact {
	return s.factsForIDs(s.byProof[key])
}

func (s Snapshot) DerivedFacts(parent FactID) []Fact {
	return s.factsForIDs(s.byParent[parent])
}

func (s Snapshot) ResolveAllocation(key ValueKey) (AllocationEvidence, error) {
	facts := s.FactsForValue(key)
	if len(facts) == 0 {
		return AllocationEvidence{}, fmt.Errorf("memoryfacts: no allocation facts for %s/%s", key.FunctionID, key.ValueID)
	}
	var out AllocationEvidence
	out.FunctionID = key.FunctionID
	out.ValueID = key.ValueID
	seen := map[FactID]struct{}{}
	for _, f := range facts {
		if f.ID != "" {
			seen[f.ID] = struct{}{}
		}
		if err := mergeString(&out.AllocationSiteID, f.AllocationSiteID, "allocation_site_id"); err != nil {
			return AllocationEvidence{}, err
		}
		mergeFirstString(&out.SiteID, f.SiteID)
		mergeFirstString(&out.SourceSpan, f.SourceSpan)
		if err := mergeString(&out.TypeName, f.TypeName, "type_name"); err != nil {
			return AllocationEvidence{}, err
		}
		if err := mergeProvenance(&out.ProvenanceClass, f.ProvenanceClass); err != nil {
			return AllocationEvidence{}, err
		}
		if err := mergeUnsafe(&out.UnsafeClass, f.UnsafeClass); err != nil {
			return AllocationEvidence{}, err
		}
		if err := mergeBorrow(&out.BorrowState, f.BorrowState); err != nil {
			return AllocationEvidence{}, err
		}
		if err := mergeEscape(&out.EscapeState, f.EscapeState); err != nil {
			return AllocationEvidence{}, err
		}
		if err := mergeAlias(&out.AliasState, f.AliasState); err != nil {
			return AllocationEvidence{}, err
		}
		if err := mergeString(&out.RegionID, f.RegionID, "region_id"); err != nil {
			return AllocationEvidence{}, err
		}
		if err := mergeString(&out.IslandID, f.IslandID, "island_id"); err != nil {
			return AllocationEvidence{}, err
		}
		if err := mergeInt(&out.Epoch, f.Epoch, "epoch"); err != nil {
			return AllocationEvidence{}, err
		}
		if err := mergeString(&out.OwnerID, f.OwnerID, "owner"); err != nil {
			return AllocationEvidence{}, err
		}
		if err := mergeDomainKind(&out.DomainKind, f.DomainKind); err != nil {
			return AllocationEvidence{}, err
		}
		if err := mergeString(&out.DomainID, f.DomainID, "domain_id"); err != nil {
			return AllocationEvidence{}, err
		}
		if err := mergeString(&out.DomainOwnerID, f.DomainOwnerID, "domain_owner_id"); err != nil {
			return AllocationEvidence{}, err
		}
		if err := mergeTransferKind(&out.TransferKind, f.TransferKind); err != nil {
			return AllocationEvidence{}, err
		}
		if err := mergeString(&out.TransferProofID, f.TransferProofID, "transfer_proof_id"); err != nil {
			return AllocationEvidence{}, err
		}
		if f.ProofKind == ProofDomainMove && f.ProofID != "" {
			if err := mergeString(&out.TransferProofID, f.ProofID, "transfer_proof_id"); err != nil {
				return AllocationEvidence{}, err
			}
		}
		out.SourceConsumed = out.SourceConsumed || f.SourceConsumed
		out.LiveBorrowCrossing = out.LiveBorrowCrossing || f.LiveBorrowCrossing
		out.DestinationActive = out.DestinationActive || f.DestinationActive
		if err := mergeString(&out.LifetimeBirth, f.LifetimeBirth, "lifetime_birth"); err != nil {
			return AllocationEvidence{}, err
		}
		if err := mergeString(&out.LifetimeDeath, f.LifetimeDeath, "lifetime_death"); err != nil {
			return AllocationEvidence{}, err
		}
		if err := mergeString(&out.LifetimeOwner, f.LifetimeOwner, "lifetime_owner"); err != nil {
			return AllocationEvidence{}, err
		}
		if f.Claim == ClaimNoEscape && f.ProofID != "" {
			out.NoEscapeProofID = f.ProofID
		}
		if f.Claim == ClaimTrustedStorage && f.ProofID != "" {
			out.StorageProofID = f.ProofID
		}
	}
	if out.AllocationSiteID == "" {
		return AllocationEvidence{}, fmt.Errorf("memoryfacts: allocation_site_id is required for %s/%s", key.FunctionID, key.ValueID)
	}
	for id := range seen {
		out.SourceFactIDs = append(out.SourceFactIDs, id)
	}
	sort.Slice(out.SourceFactIDs, func(i, j int) bool { return out.SourceFactIDs[i] < out.SourceFactIDs[j] })
	return out, nil
}

func (s Snapshot) ResolveProof(query ProofQuery) (ProofEvidence, bool) {
	ids := s.candidateProofIDs(query)
	for _, id := range ids {
		f, ok := s.byID[id]
		if !ok || f.ValidationState != ValidationPass || strings.TrimSpace(f.ValidatorName) == "" {
			continue
		}
		if f.ProofKind != query.Kind || f.ProofKind == "" {
			continue
		}
		if f.ValidationState == ValidationInvalidated {
			continue
		}
		if query.FunctionID != "" && f.FunctionID != query.FunctionID {
			continue
		}
		if query.ProofID != "" && f.ProofID != query.ProofID {
			continue
		}
		if query.SubjectBaseID != "" && f.ProofSubjectBaseID != query.SubjectBaseID {
			continue
		}
		if query.Operation != "" && f.ProofOperation != query.Operation {
			continue
		}
		if query.IslandID != "" && f.IslandID != query.IslandID {
			continue
		}
		if query.Epoch != 0 && f.Epoch != query.Epoch {
			continue
		}
		if proofKindAuthorizesOptimization(f.ProofKind) && isUnsafeUnknown(f) {
			continue
		}
		return ProofEvidence{
			FactID:        f.ID,
			ProofID:       f.ProofID,
			Kind:          f.ProofKind,
			SubjectBaseID: f.ProofSubjectBaseID,
			Operation:     f.ProofOperation,
			IslandID:      f.IslandID,
			Epoch:         f.Epoch,
			ValidatorName: f.ValidatorName,
			SourceStage:   f.SourceStage,
		}, true
	}
	return ProofEvidence{}, false
}

func (s Snapshot) Digest() string {
	facts := s.Facts()
	sort.Slice(facts, func(i, j int) bool { return deterministicFactKey(facts[i]) < deterministicFactKey(facts[j]) })
	for i := range facts {
		sort.Slice(facts[i].DerivedFactIDs, func(a, b int) bool {
			return facts[i].DerivedFactIDs[a] < facts[i].DerivedFactIDs[b]
		})
	}
	raw, _ := json.Marshal(struct {
		ProgramID string `json:"program_id"`
		Facts     []Fact `json:"facts"`
	}{ProgramID: s.programID, Facts: facts})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (g *Graph) Apply(delta Delta) error {
	if g == nil {
		return fmt.Errorf("memoryfacts: nil graph")
	}
	clone := g.clone()
	if delta.Stage != "" {
		if err := clone.AdvanceTo(delta.Stage); err != nil {
			return err
		}
	}
	for _, f := range delta.Add {
		if f.SourceStage == "" {
			f.SourceStage = delta.Stage
		}
		if clone.stageRegression(f.SourceStage) {
			return fmt.Errorf("memoryfacts: stage regression from %q to %q", clone.currentStage, f.SourceStage)
		}
		if _, err := clone.AddFact(f); err != nil {
			return err
		}
	}
	for _, invalidation := range delta.Invalidate {
		if err := clone.InvalidateFact(invalidation.FactID, invalidation.Reason); err != nil {
			return err
		}
	}
	for _, attach := range delta.Attach {
		if err := clone.AttachLoweredArtifact(attach.FactID, attach.ArtifactID); err != nil {
			return err
		}
	}
	for _, mark := range delta.Validate {
		if err := clone.MarkValidated(mark.FactID, mark.ValidatorName); err != nil {
			return err
		}
	}
	if err := clone.Validate(); err != nil {
		return err
	}
	*g = *clone
	return nil
}

func (g *Graph) AdvanceTo(stage SourceStage) error {
	if g == nil {
		return fmt.Errorf("memoryfacts: nil graph")
	}
	if !knownSourceStage(stage) {
		return fmt.Errorf("memoryfacts: unknown source_stage %q", stage)
	}
	if g.stageRegression(stage) {
		return fmt.Errorf("memoryfacts: stage regression from %q to %q", g.currentStage, stage)
	}
	g.currentStage = stage
	return nil
}

func (g *Graph) CurrentStage() SourceStage {
	if g == nil {
		return ""
	}
	return g.currentStage
}

func (s Snapshot) factsForIDs(ids []FactID) []Fact {
	out := make([]Fact, 0, len(ids))
	for _, id := range ids {
		if f, ok := s.byID[id]; ok {
			out = append(out, cloneFact(f))
		}
	}
	sort.Slice(out, func(i, j int) bool { return deterministicFactKey(out[i]) < deterministicFactKey(out[j]) })
	return out
}

func (s Snapshot) candidateProofIDs(query ProofQuery) []FactID {
	var ids []FactID
	if query.FunctionID != "" && query.ProofID != "" {
		ids = append(ids, s.byProof[ProofKey{FunctionID: query.FunctionID, ProofID: query.ProofID}]...)
	} else {
		for _, f := range s.facts {
			if f.ProofID != "" {
				ids = append(ids, f.ID)
			}
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

func deterministicFactKey(f Fact) string {
	return strings.Join([]string{
		string(f.ID),
		f.ProgramID,
		f.FunctionID,
		f.ValueID,
		f.AllocationSiteID,
		f.ProofID,
		string(f.Claim),
		string(f.SourceStage),
		string(f.ParentFactID),
		f.LoweredArtifactID,
	}, "\x00")
}

func (g *Graph) clone() *Graph {
	if g == nil {
		return nil
	}
	out := &Graph{
		programID:    g.programID,
		facts:        map[FactID]Fact{},
		order:        append([]FactID(nil), g.order...),
		currentStage: g.currentStage,
	}
	for id, fact := range g.facts {
		out.facts[id] = cloneFact(fact)
	}
	return out
}

func (g *Graph) stageRegression(stage SourceStage) bool {
	if stage == "" || g.currentStage == "" {
		return false
	}
	return stageRank(stage) < stageRank(g.currentStage)
}

func stageRank(stage SourceStage) int {
	for i, candidate := range []SourceStage{
		StageSemantics,
		StageUnsafeGatewayLowering,
		StagePLIR,
		StageAllocPlan,
		StageLowering,
		StageOptimization,
		StageValidation,
	} {
		if stage == candidate {
			return i
		}
	}
	return -1
}

func proofKindAuthorizesOptimization(kind ProofKind) bool {
	switch kind {
	case ProofBounds, ProofNoAlias, ProofStorage, ProofNoEscape, ProofRegionAlive, ProofDomainMove:
		return true
	default:
		return false
	}
}

func mergeString(dst *string, value string, name string) error {
	if value == "" {
		return nil
	}
	if *dst != "" && *dst != value {
		return fmt.Errorf("memoryfacts: conflicting %s %q vs %q", name, *dst, value)
	}
	*dst = value
	return nil
}

func mergeFirstString(dst *string, value string) {
	if *dst == "" && value != "" {
		*dst = value
	}
}

func mergeInt(dst *int, value int, name string) error {
	if value == 0 {
		return nil
	}
	if *dst != 0 && *dst != value {
		return fmt.Errorf("memoryfacts: conflicting %s %d vs %d", name, *dst, value)
	}
	*dst = value
	return nil
}

func mergeProvenance(dst *ProvenanceClass, value ProvenanceClass) error {
	if value == "" {
		return nil
	}
	if *dst == "" {
		*dst = value
		return nil
	}
	if *dst == value {
		return nil
	}
	if safeProvenanceClass(*dst) && safeProvenanceClass(value) {
		*dst = strongerSafeProvenance(*dst, value)
		return nil
	}
	return fmt.Errorf("memoryfacts: conflicting provenance %q vs %q", *dst, value)
}

func mergeUnsafe(dst *UnsafeClass, value UnsafeClass) error {
	return mergeTypedString((*string)(dst), string(value), "unsafe")
}

func mergeBorrow(dst *BorrowState, value BorrowState) error {
	return mergeTypedString((*string)(dst), string(value), "borrow")
}

func mergeEscape(dst *EscapeState, value EscapeState) error {
	if value == EscapeNoEscape && *dst != "" && *dst != EscapeNoEscape {
		return nil
	}
	if *dst == EscapeNoEscape && value != "" && value != EscapeNoEscape {
		*dst = value
		return nil
	}
	return mergeTypedString((*string)(dst), string(value), "escape")
}

func mergeAlias(dst *AliasState, value AliasState) error {
	return mergeTypedString((*string)(dst), string(value), "alias")
}

func mergeDomainKind(dst *DomainKind, value DomainKind) error {
	return mergeTypedString((*string)(dst), string(value), "domain_kind")
}

func mergeTransferKind(dst *TransferKind, value TransferKind) error {
	return mergeTypedString((*string)(dst), string(value), "transfer_kind")
}

func mergeTypedString(dst *string, value string, name string) error {
	return mergeString(dst, value, name)
}

func safeProvenanceClass(value ProvenanceClass) bool {
	switch value {
	case ProvenanceSafeKnown, ProvenanceSafeBorrowed, ProvenanceSafeOwned:
		return true
	default:
		return false
	}
}

func strongerSafeProvenance(a ProvenanceClass, b ProvenanceClass) ProvenanceClass {
	if a == ProvenanceSafeOwned || b == ProvenanceSafeOwned {
		return ProvenanceSafeOwned
	}
	if a == ProvenanceSafeBorrowed || b == ProvenanceSafeBorrowed {
		return ProvenanceSafeBorrowed
	}
	return ProvenanceSafeKnown
}
