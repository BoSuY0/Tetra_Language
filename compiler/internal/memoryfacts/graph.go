package memoryfacts

import (
	"fmt"
	"sort"
	"strings"
)

type Graph struct {
	programID string
	facts     map[FactID]Fact
	order     []FactID
}

func NewGraph(programID string) *Graph {
	return &Graph{
		programID: programID,
		facts:     map[FactID]Fact{},
	}
}

func (g *Graph) ProgramID() string {
	if g == nil {
		return ""
	}
	return g.programID
}

func (g *Graph) AddFact(f Fact) (FactID, error) {
	if g == nil {
		return "", fmt.Errorf("memoryfacts: nil graph")
	}
	if f.ID == "" {
		return "", fmt.Errorf("memoryfacts: fact_id is required")
	}
	if _, exists := g.facts[f.ID]; exists {
		return "", fmt.Errorf("memoryfacts: duplicate fact_id %q", f.ID)
	}
	if f.ProgramID == "" {
		f.ProgramID = g.programID
	}
	if f.ValidationState == "" {
		f.ValidationState = ValidationNotRun
	}
	inferredCostClass := f.CostClass == ""
	if inferredCostClass {
		f.CostClass = inferCostClass(f)
	}
	if inferredCostClass && f.CostClass == CostDynamicCheckRequired {
		f.NormalBuildCheck = true
	}
	if err := g.validateFact(f); err != nil {
		return "", err
	}
	g.facts[f.ID] = cloneFact(f)
	g.order = append(g.order, f.ID)
	return f.ID, nil
}

func (g *Graph) DeriveFact(parentID FactID, f Fact) (FactID, error) {
	if g == nil {
		return "", fmt.Errorf("memoryfacts: nil graph")
	}
	if parentID == "" {
		return "", fmt.Errorf("memoryfacts: parent fact_id is required for derived fact")
	}
	parent, ok := g.facts[parentID]
	if !ok {
		return "", fmt.Errorf("memoryfacts: parent fact_id %q does not exist", parentID)
	}
	f.ParentFactID = parentID
	if isUnsafeUnknown(parent) && isSafeProvenance(f.ProvenanceClass) {
		return "", fmt.Errorf("memoryfacts: unsafe_unknown fact %q cannot derive %s", parentID, f.ProvenanceClass)
	}
	id, err := g.AddFact(f)
	if err != nil {
		return "", err
	}
	parent = g.facts[parentID]
	parent.DerivedFactIDs = append(parent.DerivedFactIDs, id)
	g.facts[parentID] = parent
	return id, nil
}

func (g *Graph) InvalidateFact(factID FactID, reason string) error {
	if g == nil {
		return fmt.Errorf("memoryfacts: nil graph")
	}
	f, ok := g.facts[factID]
	if !ok {
		return fmt.Errorf("memoryfacts: fact_id %q does not exist", factID)
	}
	f.ValidationState = ValidationInvalidated
	if reason != "" {
		f.Reason = reason
	}
	g.facts[factID] = f
	return nil
}

func (g *Graph) AttachLoweredArtifact(factID FactID, artifactID string) error {
	if g == nil {
		return fmt.Errorf("memoryfacts: nil graph")
	}
	if strings.TrimSpace(artifactID) == "" {
		return fmt.Errorf("memoryfacts: lowered_artifact_id is required")
	}
	f, ok := g.facts[factID]
	if !ok {
		return fmt.Errorf("memoryfacts: fact_id %q does not exist", factID)
	}
	f.LoweredArtifactID = artifactID
	g.facts[factID] = f
	return nil
}

func (g *Graph) MarkValidated(factID FactID, validatorName string) error {
	if g == nil {
		return fmt.Errorf("memoryfacts: nil graph")
	}
	f, ok := g.facts[factID]
	if !ok {
		return fmt.Errorf("memoryfacts: fact_id %q does not exist", factID)
	}
	f.ValidationState = ValidationPass
	f.ValidatorName = validatorName
	if f.CostClass == CostInstrumentationOnly && zeroCostValidationRequiredClaim(f.Claim) {
		f.CostClass = inferCostClass(f)
	}
	if err := g.validateFact(f); err != nil {
		return err
	}
	g.facts[factID] = f
	return nil
}

func (g *Graph) Fact(factID FactID) (Fact, bool) {
	if g == nil {
		return Fact{}, false
	}
	f, ok := g.facts[factID]
	return cloneFact(f), ok
}

func (g *Graph) Facts() []Fact {
	if g == nil {
		return nil
	}
	out := make([]Fact, 0, len(g.order))
	for _, id := range g.order {
		out = append(out, cloneFact(g.facts[id]))
	}
	return out
}

func (g *Graph) Validate() error {
	if g == nil {
		return fmt.Errorf("memoryfacts: nil graph")
	}
	for _, id := range g.order {
		f := g.facts[id]
		if err := g.validateFact(f); err != nil {
			return err
		}
	}
	return nil
}

func (g *Graph) validateFact(f Fact) error {
	if f.ID == "" {
		return fmt.Errorf("memoryfacts: fact_id is required")
	}
	if f.ParentFactID != "" {
		parent, ok := g.facts[f.ParentFactID]
		if !ok {
			return fmt.Errorf("memoryfacts: parent fact_id %q does not exist", f.ParentFactID)
		}
		if isUnsafeUnknown(parent) && isSafeProvenance(f.ProvenanceClass) {
			return fmt.Errorf("memoryfacts: unsafe_unknown fact %q cannot derive %s", f.ParentFactID, f.ProvenanceClass)
		}
	}
	if f.ValidationState == ValidationPass && requiresLoweredArtifact(f) && f.LoweredArtifactID == "" {
		return fmt.Errorf("memoryfacts: validated lowering/storage fact %q requires lowered_artifact_id", f.ID)
	}
	if f.ValidationState == ValidationPass && strings.TrimSpace(f.ValidatorName) == "" {
		return fmt.Errorf("memoryfacts: validated fact %q requires validator_name", f.ID)
	}
	if f.ValidationState == ValidationPass && boundsTypedProofClaim(f.Claim) {
		if missing := missingTypedProofFields(f.ProofID, f.ProofKind, f.ProofSubjectBaseID, f.ProofIndexValueID, f.ProofOperation, f.ProofRange); len(missing) > 0 {
			return fmt.Errorf("memoryfacts: validated bounds proof fact %q requires typed proof fields: %s", f.ID, strings.Join(missing, ", "))
		}
	}
	if !knownSourceStage(f.SourceStage) {
		return fmt.Errorf("memoryfacts: unknown source_stage %q", f.SourceStage)
	}
	if !knownProvenanceClass(f.ProvenanceClass) {
		return fmt.Errorf("memoryfacts: unknown provenance_class %q", f.ProvenanceClass)
	}
	if !knownUnsafeClass(f.UnsafeClass) {
		return fmt.Errorf("memoryfacts: unknown unsafe_class %q", f.UnsafeClass)
	}
	if !knownAliasState(f.AliasState) {
		return fmt.Errorf("memoryfacts: unknown alias_state %q", f.AliasState)
	}
	if !knownCostClass(f.CostClass) {
		return fmt.Errorf("memoryfacts: unknown cost_class %q", f.CostClass)
	}
	if islandBackedFact(f) && f.Epoch <= 0 {
		return fmt.Errorf("memoryfacts: island-backed fact %q requires positive epoch", f.ID)
	}
	if f.Epoch > 0 && strings.TrimSpace(f.IslandID) == "" {
		return fmt.Errorf("memoryfacts: fact %q epoch requires island_id", f.ID)
	}
	if strings.TrimSpace(f.IslandID) != "" && strings.TrimSpace(f.BaseID) == "" {
		return fmt.Errorf("memoryfacts: fact %q island_id requires base_id", f.ID)
	}
	if dynamicRawRuntimeCheckCostDisallowed(f.Claim, f.CostClass) {
		return fmt.Errorf("memoryfacts: dynamic raw check fact %q with claim %q must use %s, got %s", f.ID, f.Claim, CostDynamicCheckRequired, f.CostClass)
	}
	if zeroCostProvenClaimDisallowed(f) {
		return fmt.Errorf("memoryfacts: zero_cost_proven fact %q requires validated compiler-owned proof for claim %q", f.ID, f.Claim)
	}
	if f.CostClass == CostDynamicCheckRequired && !f.NormalBuildCheck {
		return fmt.Errorf("memoryfacts: dynamic_check_required fact %q requires normal_build_check", f.ID)
	}
	if hasSafeProvenanceFromUnsafeUnknown(f.ProvenanceClass, f.UnsafeClass) {
		return fmt.Errorf("memoryfacts: safe provenance %s cannot be sourced from unsafe_unknown", f.ProvenanceClass)
	}
	if isUnsafeUnknown(f) && unsafeUnknownOptimizationClaim(f.Claim, f.AliasState) {
		return fmt.Errorf("memoryfacts: unsafe_unknown fact %q cannot authorize optimization claim %q", f.ID, f.Claim)
	}
	if broadNoAliasClaim(f.Claim) {
		return fmt.Errorf("memoryfacts: broad noalias claim %q is outside Memory Ideal v0", f.Claim)
	}
	if f.ValidationState == ValidationPass && conservativeNoAliasBoundaryClaim(f.Claim) {
		return fmt.Errorf("memoryfacts: conservative noalias boundary fact %q cannot be validated", f.ID)
	}
	if claimRequiresParentFactID(f.Claim) && f.ParentFactID == "" {
		return fmt.Errorf("memoryfacts: derived claim %q requires parent_fact_id", f.Claim)
	}
	if f.Claim == "copy_owned" && f.ProvenanceClass != ProvenanceSafeOwned {
		return fmt.Errorf("memoryfacts: copy_owned fact %q requires safe_owned provenance", f.ID)
	}
	if isUnsafeUnknown(f) && f.CostClass == CostZeroCostProven {
		return fmt.Errorf("memoryfacts: unsafe_unknown fact %q cannot claim %s", f.ID, f.CostClass)
	}
	if f.CostClass == CostDynamicCheckRequired && memoryOptimizationClaim(f.Claim, f.AliasState) && !f.NormalBuildCheck {
		return fmt.Errorf("memoryfacts: dynamic_check_required optimization claim %q requires normal_build_check for fact %q", f.Claim, f.ID)
	}
	if bareBoundsCheckEliminatedClaim(f.Claim) {
		return fmt.Errorf("memoryfacts: bounds_check_eliminated fact %q requires compiler-owned proof id; use bounds_check_removed_with_proof_id", f.ID)
	}
	if unsafeVerifiedRootDisallowedClaim(f.ProvenanceClass, f.UnsafeClass, f.Claim) {
		return fmt.Errorf("memoryfacts: unsafe_verified_root fact %q cannot claim %q without bounded raw metadata", f.ID, f.Claim)
	}
	if unsafeCheckedDisallowedClaim(f.ProvenanceClass, f.UnsafeClass, f.Claim) {
		return fmt.Errorf("memoryfacts: unsafe_checked fact %q cannot claim %q outside checked runtime/bounds evidence", f.ID, f.Claim)
	}
	if capMemDisallowedProofClaim(f.Claim, f.ValidatorName, f.Reason) {
		return fmt.Errorf("memoryfacts: cap.mem authorization fact %q cannot claim %q; cap.mem authorizes raw operations only and does not prove pointer validity, bounds, ownership, noalias, or safe provenance", f.ID, f.Claim)
	}
	if f.ValidationState == ValidationPass && unsafeExternalRootTrustedStorage(f.ProvenanceClass, f.UnsafeClass, f.StoragePlan, f.ActualLoweringStorage) {
		return fmt.Errorf("memoryfacts: unsafe/external root %s/%s fact %q cannot validate trusted storage lowering %q/%q", f.ProvenanceClass, f.UnsafeClass, f.ID, f.StoragePlan, f.ActualLoweringStorage)
	}
	if f.ValidationState == ValidationPass && validatedTrustedStorageHeapFallback(f.StoragePlan, f.ActualLoweringStorage) {
		return fmt.Errorf("memoryfacts: validated %s claim cannot lower as Heap for fact %q", f.StoragePlan, f.ID)
	}
	if f.ValidationState == ValidationPass && runtimeProofRequiredStorage(f.StoragePlan, f.ActualLoweringStorage) {
		return fmt.Errorf("memoryfacts: validated runtime boundary storage %q/%q for fact %q requires production runtime proof", f.StoragePlan, f.ActualLoweringStorage, f.ID)
	}
	if f.ValidationState == ValidationPass && strings.Contains(f.Claim, "no_alias") && !validatedNoAliasState(f.AliasState) {
		return fmt.Errorf("memoryfacts: validated no_alias fact %q requires unique or mutable_exclusive alias state, got unknown alias %q", f.ID, f.AliasState)
	}
	if f.StoragePlan != "" && !knownStorageClass(f.StoragePlan) {
		return fmt.Errorf("memoryfacts: unknown storage_plan %q", f.StoragePlan)
	}
	if f.ActualLoweringStorage != "" && !knownStorageClass(f.ActualLoweringStorage) {
		return fmt.Errorf("memoryfacts: unknown actual_lowering_storage %q", f.ActualLoweringStorage)
	}
	if (f.StoragePlan == "") != (f.ActualLoweringStorage == "") {
		return fmt.Errorf("memoryfacts: fact %q requires planned_storage and actual_lowering_storage together", f.ID)
	}
	if storageFallbackRequiresReason(f.StoragePlan, f.ActualLoweringStorage, f.CostClass) && strings.TrimSpace(f.Reason) == "" {
		return fmt.Errorf("memoryfacts: storage/conservative fallback fact %q requires reason", f.ID)
	}
	if f.ParamIndex != nil && *f.ParamIndex < 0 {
		return fmt.Errorf("memoryfacts: fact %q has negative param_index %d", f.ID, *f.ParamIndex)
	}
	return nil
}

func islandBackedFact(f Fact) bool {
	return strings.TrimSpace(f.IslandID) != "" || f.StoragePlan == StorageExplicitIsland || f.ActualLoweringStorage == StorageExplicitIsland
}

func (g *Graph) SortedFactsForTest() []Fact {
	out := g.Facts()
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func cloneFact(f Fact) Fact {
	f.DerivedFactIDs = append([]FactID(nil), f.DerivedFactIDs...)
	if f.ParamIndex != nil {
		index := *f.ParamIndex
		f.ParamIndex = &index
	}
	return f
}
