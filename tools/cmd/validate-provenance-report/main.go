package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type provenanceReport struct {
	Schema                string                   `json:"schema"`
	GeneratedAt           string                   `json:"generated_at"`
	Corpus                provenanceCorpus         `json:"corpus"`
	Totals                provenanceStats          `json:"totals"`
	Artifacts             []provenanceArtifact     `json:"artifacts"`
	UnknownReasonBuckets  []provenanceReasonBucket `json:"unknown_reason_buckets"`
	ExternalReasonBuckets []provenanceReasonBucket `json:"external_reason_buckets"`
	FalseKnownRiskPoints  []provenanceRiskPoint    `json:"false_known_risk_points"`
	NonClaims             []string                 `json:"non_claims"`
}

type provenanceCorpus struct {
	PathGlob         string   `json:"path_glob"`
	ProofFiles       int      `json:"proof_files"`
	BoundsFiles      int      `json:"bounds_files"`
	EvidenceCommands []string `json:"evidence_commands"`
}

type provenanceStats struct {
	Artifacts              int            `json:"artifacts,omitempty"`
	Functions              int            `json:"functions"`
	Values                 int            `json:"values"`
	Facts                  int            `json:"facts"`
	ProofUses              int            `json:"proof_uses"`
	ProvenanceCounts       map[string]int `json:"provenance_counts"`
	ProvenanceUnknownFacts int            `json:"provenance_unknown_facts"`
}

type provenanceArtifact struct {
	Name                   string         `json:"name"`
	Source                 string         `json:"source"`
	Proof                  string         `json:"proof"`
	Bounds                 string         `json:"bounds"`
	Functions              int            `json:"functions"`
	Values                 int            `json:"values"`
	Facts                  int            `json:"facts"`
	ProofUses              int            `json:"proof_uses"`
	ProvenanceCounts       map[string]int `json:"provenance_counts"`
	ProvenanceUnknownFacts int            `json:"provenance_unknown_facts"`
}

type provenanceReasonBucket struct {
	Reason     string   `json:"reason"`
	ValueCount int      `json:"value_count"`
	FactCount  int      `json:"fact_count"`
	Artifacts  []string `json:"artifacts"`
	Note       string   `json:"note"`
}

type provenanceRiskPoint struct {
	ID               string                   `json:"id"`
	Status           string                   `json:"status"`
	Summary          string                   `json:"summary"`
	CorpusEvidence   string                   `json:"corpus_evidence"`
	CodeEvidence     []map[string]interface{} `json:"code_evidence"`
	ArtifactEvidence []string                 `json:"artifact_evidence"`
	Mitigation       string                   `json:"mitigation"`
}

type proofDocument struct {
	SchemaVersion int               `json:"schema_version,omitempty"`
	Kind          string            `json:"kind,omitempty"`
	Target        string            `json:"target,omitempty"`
	Bounds        json.RawMessage   `json:"bounds,omitempty"`
	Proofs        []json.RawMessage `json:"proofs,omitempty"`
	PLIR          struct {
		Funcs []proofFunction `json:"funcs"`
	} `json:"plir"`
}

type proofFunction struct {
	Name      string        `json:"name"`
	Values    []proofValue  `json:"values"`
	Facts     []proofFact   `json:"facts"`
	ProofUses []interface{} `json:"proof_uses"`
}

type proofValue struct {
	ID         string `json:"id"`
	Provenance struct {
		Kind string `json:"kind"`
		Root string `json:"root"`
	} `json:"provenance"`
}

type proofFact struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	ValueID string `json:"value_id"`
	Reason  string `json:"reason"`
}

type corpusEvidence struct {
	ProofFiles      []string
	BoundsFiles     []string
	Totals          provenanceStats
	Artifacts       map[string]provenanceStats
	UnknownBuckets  map[string]reasonEvidence
	ExternalBuckets map[string]reasonEvidence
}

type reasonEvidence struct {
	Values    map[string]bool
	Facts     int
	Artifacts map[string]bool
}

func main() {
	reportPath := flag.String("report", "", "path to provenance coverage JSON report")
	corpusDir := flag.String("corpus-dir", "", "directory containing proof/bounds corpus artifacts")
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(*reportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *corpusDir == "" {
		*corpusDir = filepath.Dir(*reportPath)
	}
	if err := validateProvenanceReport(raw, *corpusDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateProvenanceReport(raw []byte, corpusDir string) error {
	var report provenanceReport
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		return err
	}
	if report.Schema != "tetra.proof-coverage.provenance.v1" {
		return fmt.Errorf("unsupported schema %q", report.Schema)
	}
	if strings.TrimSpace(report.GeneratedAt) == "" {
		return fmt.Errorf("generated_at is required")
	}
	if len(report.Corpus.EvidenceCommands) == 0 {
		return fmt.Errorf("corpus.evidence_commands must not be empty")
	}
	if len(report.Artifacts) == 0 {
		return fmt.Errorf("artifacts must not be empty")
	}
	if len(report.NonClaims) == 0 {
		return fmt.Errorf("non_claims must not be empty")
	}
	if len(report.FalseKnownRiskPoints) == 0 {
		return fmt.Errorf("false_known_risk_points must not be empty")
	}

	evidence, err := collectCorpusEvidence(corpusDir)
	if err != nil {
		return err
	}
	if report.Corpus.ProofFiles != len(evidence.ProofFiles) {
		return fmt.Errorf(
			"corpus.proof_files = %d, want %d",
			report.Corpus.ProofFiles,
			len(evidence.ProofFiles),
		)
	}
	if report.Corpus.BoundsFiles != len(evidence.BoundsFiles) {
		return fmt.Errorf(
			"corpus.bounds_files = %d, want %d",
			report.Corpus.BoundsFiles,
			len(evidence.BoundsFiles),
		)
	}
	if err := compareStats("totals", report.Totals, evidence.Totals, true); err != nil {
		return err
	}
	if report.Totals.Artifacts != len(evidence.ProofFiles) {
		return fmt.Errorf(
			"totals.artifacts = %d, want %d",
			report.Totals.Artifacts,
			len(evidence.ProofFiles),
		)
	}
	if len(report.Artifacts) != len(evidence.Artifacts) {
		return fmt.Errorf(
			"artifacts length = %d, want %d",
			len(report.Artifacts),
			len(evidence.Artifacts),
		)
	}
	for _, artifact := range report.Artifacts {
		if strings.TrimSpace(artifact.Proof) == "" {
			return fmt.Errorf("artifact %q proof is required", artifact.Name)
		}
		want, ok := evidence.Artifacts[filepath.Base(artifact.Proof)]
		if !ok {
			return fmt.Errorf("artifact proof %q not found in corpus", artifact.Proof)
		}
		got := provenanceStats{
			Functions:              artifact.Functions,
			Values:                 artifact.Values,
			Facts:                  artifact.Facts,
			ProofUses:              artifact.ProofUses,
			ProvenanceCounts:       artifact.ProvenanceCounts,
			ProvenanceUnknownFacts: artifact.ProvenanceUnknownFacts,
		}
		if err := compareStats("artifact "+artifact.Proof, got, want, false); err != nil {
			return err
		}
	}
	if err := compareReasonBuckets(
		"unknown_reason_buckets",
		report.UnknownReasonBuckets,
		evidence.UnknownBuckets,
	); err != nil {
		return err
	}
	if err := compareReasonBuckets(
		"external_reason_buckets",
		report.ExternalReasonBuckets,
		evidence.ExternalBuckets,
	); err != nil {
		return err
	}
	return nil
}

func collectCorpusEvidence(corpusDir string) (corpusEvidence, error) {
	proofFiles, err := filepath.Glob(filepath.Join(corpusDir, "*.proof.json"))
	if err != nil {
		return corpusEvidence{}, err
	}
	boundsFiles, err := filepath.Glob(filepath.Join(corpusDir, "*.bounds.json"))
	if err != nil {
		return corpusEvidence{}, err
	}
	sort.Strings(proofFiles)
	sort.Strings(boundsFiles)
	if len(proofFiles) == 0 {
		return corpusEvidence{}, fmt.Errorf("corpus has no *.proof.json files in %s", corpusDir)
	}
	evidence := corpusEvidence{
		ProofFiles:      proofFiles,
		BoundsFiles:     boundsFiles,
		Artifacts:       map[string]provenanceStats{},
		UnknownBuckets:  map[string]reasonEvidence{},
		ExternalBuckets: map[string]reasonEvidence{},
	}
	evidence.Totals.ProvenanceCounts = map[string]int{}
	for _, path := range proofFiles {
		stats, unknown, external, err := scanProof(path)
		if err != nil {
			return corpusEvidence{}, err
		}
		name := filepath.Base(path)
		evidence.Artifacts[name] = stats
		addStats(&evidence.Totals, stats)
		mergeReasonBuckets(evidence.UnknownBuckets, unknown)
		mergeReasonBuckets(evidence.ExternalBuckets, external)
	}
	evidence.Totals.Artifacts = len(proofFiles)
	return evidence, nil
}

func scanProof(
	path string,
) (provenanceStats, map[string]reasonEvidence, map[string]reasonEvidence, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return provenanceStats{}, nil, nil, err
	}
	var doc proofDocument
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&doc); err != nil {
		return provenanceStats{}, nil, nil, fmt.Errorf("%s: %w", path, err)
	}
	stats := provenanceStats{ProvenanceCounts: map[string]int{}}
	unknown := map[string]reasonEvidence{}
	external := map[string]reasonEvidence{}
	artifact := filepath.Base(path)
	for _, fn := range doc.PLIR.Funcs {
		stats.Functions++
		stats.Values += len(fn.Values)
		stats.Facts += len(fn.Facts)
		stats.ProofUses += len(fn.ProofUses)
		valueKinds := map[string]string{}
		for _, value := range fn.Values {
			kind := strings.TrimSpace(value.Provenance.Kind)
			if kind == "" {
				kind = "missing"
			}
			stats.ProvenanceCounts[kind]++
			valueKinds[value.ID] = kind
		}
		for _, fact := range fn.Facts {
			if fact.Kind != "provenance_unknown" {
				continue
			}
			stats.ProvenanceUnknownFacts++
			reason := strings.TrimSpace(fact.Reason)
			if reason == "" {
				reason = "unspecified provenance unknown reason"
			}
			addReasonEvidence(unknown, reason, fact.ValueID, artifact)
			if valueKinds[fact.ValueID] == "external" {
				addReasonEvidence(external, reason, fact.ValueID, artifact)
			}
		}
	}
	return stats, unknown, external, nil
}

func addStats(total *provenanceStats, stats provenanceStats) {
	total.Functions += stats.Functions
	total.Values += stats.Values
	total.Facts += stats.Facts
	total.ProofUses += stats.ProofUses
	total.ProvenanceUnknownFacts += stats.ProvenanceUnknownFacts
	if total.ProvenanceCounts == nil {
		total.ProvenanceCounts = map[string]int{}
	}
	for kind, count := range stats.ProvenanceCounts {
		total.ProvenanceCounts[kind] += count
	}
}

func addReasonEvidence(
	buckets map[string]reasonEvidence,
	reason string,
	valueID string,
	artifact string,
) {
	bucket := buckets[reason]
	if bucket.Values == nil {
		bucket.Values = map[string]bool{}
	}
	if bucket.Artifacts == nil {
		bucket.Artifacts = map[string]bool{}
	}
	if valueID != "" {
		bucket.Values[valueID] = true
	}
	bucket.Facts++
	bucket.Artifacts[artifact] = true
	buckets[reason] = bucket
}

func mergeReasonBuckets(dst map[string]reasonEvidence, src map[string]reasonEvidence) {
	for reason, bucket := range src {
		for value := range bucket.Values {
			addReasonEvidence(dst, reason, value, "")
		}
		merged := dst[reason]
		merged.Facts += bucket.Facts - len(bucket.Values)
		for artifact := range bucket.Artifacts {
			if artifact != "" {
				merged.Artifacts[artifact] = true
			}
		}
		dst[reason] = merged
	}
}

func compareStats(
	label string,
	got provenanceStats,
	want provenanceStats,
	includeArtifacts bool,
) error {
	if includeArtifacts && got.Artifacts != want.Artifacts {
		return fmt.Errorf("%s.artifacts = %d, want %d", label, got.Artifacts, want.Artifacts)
	}
	for field, pair := range map[string][2]int{
		"functions":                {got.Functions, want.Functions},
		"values":                   {got.Values, want.Values},
		"facts":                    {got.Facts, want.Facts},
		"proof_uses":               {got.ProofUses, want.ProofUses},
		"provenance_unknown_facts": {got.ProvenanceUnknownFacts, want.ProvenanceUnknownFacts},
	} {
		if pair[0] != pair[1] {
			return fmt.Errorf("%s.%s = %d, want %d", label, field, pair[0], pair[1])
		}
	}
	if err := compareCountMap(
		label+".provenance_counts",
		got.ProvenanceCounts,
		want.ProvenanceCounts,
	); err != nil {
		return err
	}
	return nil
}

func compareCountMap(label string, got map[string]int, want map[string]int) error {
	for key, wantValue := range want {
		if got[key] != wantValue {
			return fmt.Errorf("%s.%s = %d, want %d", label, key, got[key], wantValue)
		}
	}
	for key, gotValue := range got {
		if want[key] == 0 && gotValue != 0 {
			return fmt.Errorf("%s.%s = %d, want 0", label, key, gotValue)
		}
	}
	return nil
}

func compareReasonBuckets(
	label string,
	got []provenanceReasonBucket,
	want map[string]reasonEvidence,
) error {
	if len(want) > 0 && len(got) == 0 {
		return fmt.Errorf("%s must include %d reason bucket(s)", label, len(want))
	}
	seen := map[string]bool{}
	for _, bucket := range got {
		if strings.TrimSpace(bucket.Reason) == "" {
			return fmt.Errorf("%s contains bucket with empty reason", label)
		}
		wantBucket, ok := want[bucket.Reason]
		if !ok {
			return fmt.Errorf("%s contains unexpected reason %q", label, bucket.Reason)
		}
		seen[bucket.Reason] = true
		if bucket.ValueCount != len(wantBucket.Values) {
			return fmt.Errorf(
				"%s[%q].value_count = %d, want %d",
				label,
				bucket.Reason,
				bucket.ValueCount,
				len(wantBucket.Values),
			)
		}
		if bucket.FactCount != wantBucket.Facts {
			return fmt.Errorf(
				"%s[%q].fact_count = %d, want %d",
				label,
				bucket.Reason,
				bucket.FactCount,
				wantBucket.Facts,
			)
		}
		if len(bucket.Artifacts) == 0 {
			return fmt.Errorf("%s[%q].artifacts must not be empty", label, bucket.Reason)
		}
	}
	for reason := range want {
		if !seen[reason] {
			return fmt.Errorf("%s missing reason %q", label, reason)
		}
	}
	return nil
}
