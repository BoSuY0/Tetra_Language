package proof

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type Kind string

const (
	KindAllocationPlacement Kind = "allocation_placement"
	KindBounds              Kind = "bounds"
	KindCopyNecessity       Kind = "copy_necessity"
	KindTranslationWitness  Kind = "translation_witness"
)

type Status string

const (
	StatusProven       Status = "proven"
	StatusConservative Status = "conservative"
	StatusRejected     Status = "rejected"
	StatusUnknown      Status = "unknown"
)

type Subject struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type Term struct {
	ID                 string   `json:"id"`
	Kind               Kind     `json:"kind"`
	Subject            Subject  `json:"subject"`
	Assumptions        []string `json:"assumptions,omitempty"`
	DerivationRule     string   `json:"derivation_rule"`
	SourceSpan         string   `json:"source_span,omitempty"`
	ASTID              string   `json:"ast_id,omitempty"`
	PLIROpID           string   `json:"plir_op_id,omitempty"`
	IROpID             string   `json:"ir_op_id,omitempty"`
	DominanceScope     string   `json:"dominance_scope,omitempty"`
	LifetimeScope      string   `json:"lifetime_scope,omitempty"`
	MutationEpoch      string   `json:"mutation_epoch,omitempty"`
	AliasEpoch         string   `json:"alias_epoch,omitempty"`
	InvalidationPolicy string   `json:"invalidation_policy,omitempty"`
	StableHash         string   `json:"stable_hash"`
	ProducerPass       string   `json:"producer_pass"`
	ConsumerPasses     []string `json:"consumer_passes,omitempty"`
	Status             Status   `json:"status"`
}

type Reference struct {
	ID         string  `json:"id"`
	Subject    Subject `json:"subject,omitempty"`
	StableHash string  `json:"stable_hash,omitempty"`
}

type Store struct {
	Terms []Term `json:"terms"`
}

func NewTerm(term Term) Term {
	term.StableHash = StableHash(term)
	return term
}

func NewStore(terms ...Term) Store {
	return Store{Terms: append([]Term(nil), terms...)}
}

func (s Store) Validate() error {
	var issues []string
	seen := map[string]bool{}
	for i, term := range s.Terms {
		prefix := fmt.Sprintf("term %d", i)
		if strings.TrimSpace(term.ID) == "" {
			issues = append(issues, prefix+": missing proof id")
			continue
		}
		if seen[term.ID] {
			issues = append(issues, prefix+": duplicate proof id "+term.ID)
		}
		seen[term.ID] = true
		if strings.TrimSpace(string(term.Kind)) == "" {
			issues = append(issues, prefix+": missing proof kind")
		}
		if strings.TrimSpace(term.Subject.Kind) == "" || strings.TrimSpace(term.Subject.ID) == "" {
			issues = append(issues, prefix+": missing proof subject")
		}
		if strings.TrimSpace(term.ProducerPass) == "" {
			issues = append(issues, prefix+": missing producer_pass")
		}
		if !knownStatus(term.Status) {
			issues = append(issues, fmt.Sprintf("%s: unknown status %q", prefix, term.Status))
		}
		if term.StableHash == "" {
			issues = append(issues, prefix+": missing stable_hash")
		} else if want := StableHash(term); term.StableHash != want {
			issues = append(issues, fmt.Sprintf("%s: stale stable_hash for %s: got %s want %s", prefix, term.ID, term.StableHash, want))
		}
		if term.Status == StatusProven && termContainsUnsafeUnknown(term) {
			issues = append(issues, fmt.Sprintf("%s: unsafe_unknown cannot be promoted to proven proof %s", prefix, term.ID))
		}
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func (s Store) ValidateReferences(refs []Reference) error {
	var issues []string
	if err := s.Validate(); err != nil {
		issues = append(issues, err.Error())
	}
	byID := map[string]Term{}
	for _, term := range s.Terms {
		byID[term.ID] = term
	}
	for i, ref := range refs {
		if strings.TrimSpace(ref.ID) == "" {
			issues = append(issues, fmt.Sprintf("reference %d: missing proof id", i))
			continue
		}
		term, ok := byID[ref.ID]
		if !ok {
			issues = append(issues, fmt.Sprintf("reference %d: missing proof id %q", i, ref.ID))
			continue
		}
		if ref.Subject.Kind != "" || ref.Subject.ID != "" {
			if ref.Subject != term.Subject {
				issues = append(issues, fmt.Sprintf("reference %d: subject mismatch for %s: got %+v want %+v", i, ref.ID, ref.Subject, term.Subject))
			}
		}
		if ref.StableHash != "" && ref.StableHash != term.StableHash {
			issues = append(issues, fmt.Sprintf("reference %d: stable hash mismatch for %s", i, ref.ID))
		}
		if term.Status == StatusRejected || term.Status == StatusUnknown {
			issues = append(issues, fmt.Sprintf("reference %d: proof %s has unusable status %s", i, ref.ID, term.Status))
		}
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func StableHash(term Term) string {
	canonical := struct {
		ID             string   `json:"id"`
		Kind           Kind     `json:"kind"`
		Subject        Subject  `json:"subject"`
		Assumptions    []string `json:"assumptions,omitempty"`
		DerivationRule string   `json:"derivation_rule"`
		SourceSpan     string   `json:"source_span,omitempty"`
		ASTID          string   `json:"ast_id,omitempty"`
		PLIROpID       string   `json:"plir_op_id,omitempty"`
		IROpID         string   `json:"ir_op_id,omitempty"`
		ProducerPass   string   `json:"producer_pass"`
		Status         Status   `json:"status"`
	}{
		ID:             term.ID,
		Kind:           term.Kind,
		Subject:        term.Subject,
		Assumptions:    sortedCopy(term.Assumptions),
		DerivationRule: term.DerivationRule,
		SourceSpan:     term.SourceSpan,
		ASTID:          term.ASTID,
		PLIROpID:       term.PLIROpID,
		IROpID:         term.IROpID,
		ProducerPass:   term.ProducerPass,
		Status:         term.Status,
	}
	raw, _ := json.Marshal(canonical)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func knownStatus(status Status) bool {
	switch status {
	case StatusProven, StatusConservative, StatusRejected, StatusUnknown:
		return true
	default:
		return false
	}
}

func termContainsUnsafeUnknown(term Term) bool {
	for _, value := range append(append([]string(nil), term.Assumptions...), term.DerivationRule) {
		if strings.Contains(strings.ToLower(value), "unsafe_unknown") {
			return true
		}
	}
	return false
}
