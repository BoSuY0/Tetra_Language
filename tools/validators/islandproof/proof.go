package islandproof

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	SchemaV1      = "tetra.island.proof.v1"
	ValidatorName = "validate-island-proof"
)

type Options struct {
	MemoryReport      []byte
	Manifest          []byte
	CurrentGitHead    string
	RequireSameCommit bool
}

type Report struct {
	Schema          string  `json:"schema"`
	Producer        string  `json:"producer"`
	ProducerCommand string  `json:"producer_command"`
	GitHead         string  `json:"git_head"`
	GeneratedAt     string  `json:"generated_at,omitempty"`
	Proofs          []Proof `json:"proofs"`
}

type Proof struct {
	ProofID               string   `json:"proof_id"`
	Operation             string   `json:"operation"`
	ProofKind             string   `json:"proof_kind"`
	SubjectBaseID         string   `json:"subject_base_id"`
	IslandID              string   `json:"island_id"`
	Epoch                 int      `json:"epoch"`
	SourceFactID          string   `json:"source_fact_id"`
	Claim                 string   `json:"claim"`
	ValidatorName         string   `json:"validator_name"`
	ValidatorStatus       string   `json:"validator_status"`
	ProvenanceClass       string   `json:"provenance_class,omitempty"`
	UnsafeClass           string   `json:"unsafe_class,omitempty"`
	AliasState            string   `json:"alias_state,omitempty"`
	PlannedStorage        string   `json:"planned_storage,omitempty"`
	ActualLoweringStorage string   `json:"actual_lowering_storage,omitempty"`
	Dominance             string   `json:"dominance,omitempty"`
	DistinctLiveIslands   []string `json:"distinct_live_islands,omitempty"`
}

type memoryReport struct {
	SchemaVersion string            `json:"schema_version"`
	Rows          []memoryReportRow `json:"rows"`
}

type memoryReportRow struct {
	SourceFactID          string `json:"source_fact_id"`
	Claim                 string `json:"claim"`
	ClaimLevel            string `json:"claim_level"`
	ValidatorName         string `json:"validator_name"`
	ValidatorStatus       string `json:"validator_status"`
	ProvenanceClass       string `json:"provenance_class"`
	UnsafeClass           string `json:"unsafe_class"`
	AliasState            string `json:"alias_state"`
	IslandID              string `json:"island_id"`
	Epoch                 int    `json:"epoch"`
	BaseID                string `json:"base_id"`
	ProofID               string `json:"proof_id"`
	ProofKind             string `json:"proof_kind"`
	ProofSubjectBaseID    string `json:"proof_subject_base_id"`
	ProofOperation        string `json:"proof_operation"`
	PlannedStorage        string `json:"planned_storage"`
	ActualLoweringStorage string `json:"actual_lowering_storage"`
}

type releaseManifest struct {
	Schema       string            `json:"schema"`
	Target       string            `json:"target,omitempty"`
	GitHead      string            `json:"git_head"`
	GeneratedAt  string            `json:"generated_at,omitempty"`
	ReportDir    string            `json:"report_dir,omitempty"`
	HashManifest string            `json:"hash_manifest,omitempty"`
	Commands     []releaseCommand  `json:"commands"`
	Artifacts    []releaseArtifact `json:"artifacts"`
}

type releaseCommand struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

type releaseArtifact struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Schema  string `json:"schema,omitempty"`
	Target  string `json:"target,omitempty"`
	Command string `json:"command"`
}

func Validate(raw []byte, opts Options) error {
	var report Report
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}
	var memory memoryReport
	if len(bytes.TrimSpace(opts.MemoryReport)) == 0 {
		return errors.New("memory-report is required")
	}
	if err := decodeMemoryReport(opts.MemoryReport, &memory); err != nil {
		return fmt.Errorf("memory-report: %w", err)
	}
	var manifest *releaseManifest
	if len(bytes.TrimSpace(opts.Manifest)) > 0 {
		var parsed releaseManifest
		if err := decodeReleaseManifest(opts.Manifest, &parsed); err != nil {
			return fmt.Errorf("manifest: %w", err)
		}
		manifest = &parsed
	}
	return ValidateReport(report, memory, manifest, opts)
}

func ValidateReport(report Report, memory memoryReport, manifest *releaseManifest, opts Options) error {
	var issues []string
	if report.Schema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, SchemaV1))
	}
	if strings.TrimSpace(report.Producer) == "" {
		issues = append(issues, "producer is required")
	}
	if strings.TrimSpace(report.ProducerCommand) == "" {
		issues = append(issues, "producer_command is required")
	}
	if !isGitHead(report.GitHead) {
		issues = append(issues, "git_head must be a 40-character lowercase hex commit")
	}
	if opts.RequireSameCommit && report.GitHead != strings.TrimSpace(opts.CurrentGitHead) {
		issues = append(issues, fmt.Sprintf("git_head %q does not match current commit %q", report.GitHead, strings.TrimSpace(opts.CurrentGitHead)))
	}
	if len(report.Proofs) == 0 {
		issues = append(issues, "proofs are required")
	}
	if memory.SchemaVersion != "tetra.memory-report.v1" {
		issues = append(issues, fmt.Sprintf("memory report schema_version is %q, want tetra.memory-report.v1", memory.SchemaVersion))
	}
	rows := memoryRowsBySourceFactID(memory.Rows)
	for i, proof := range report.Proofs {
		issues = append(issues, validateProof(i, proof, rows)...)
	}
	if manifest != nil {
		issues = append(issues, validateManifestAgainstReport(report, *manifest)...)
	}
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateProof(index int, proof Proof, rows map[string]memoryReportRow) []string {
	prefix := fmt.Sprintf("proof %d", index)
	var issues []string
	if strings.TrimSpace(proof.ProofID) == "" {
		issues = append(issues, prefix+": proof_id is required")
	}
	if strings.TrimSpace(proof.Operation) == "" {
		issues = append(issues, prefix+": operation is required")
	}
	if strings.TrimSpace(proof.ProofKind) == "" {
		issues = append(issues, prefix+": proof_kind is required")
	}
	if strings.TrimSpace(proof.SubjectBaseID) == "" {
		issues = append(issues, prefix+": subject_base_id is required")
	}
	if strings.TrimSpace(proof.IslandID) == "" {
		issues = append(issues, prefix+": island_id is required")
	}
	if proof.Epoch <= 0 {
		issues = append(issues, prefix+": epoch must be positive")
	}
	if strings.TrimSpace(proof.SourceFactID) == "" {
		issues = append(issues, prefix+": source_fact_id is required")
	}
	if strings.TrimSpace(proof.ProvenanceClass) == "" {
		issues = append(issues, prefix+": provenance_class is required")
	} else if !knownProvenanceClass(proof.ProvenanceClass) {
		issues = append(issues, fmt.Sprintf("%s: unknown provenance_class %q", prefix, proof.ProvenanceClass))
	}
	if strings.TrimSpace(proof.UnsafeClass) == "" {
		issues = append(issues, prefix+": unsafe_class is required")
	} else if !knownUnsafeClass(proof.UnsafeClass) {
		issues = append(issues, fmt.Sprintf("%s: unknown unsafe_class %q", prefix, proof.UnsafeClass))
	}
	if proof.ValidatorName != ValidatorName {
		issues = append(issues, fmt.Sprintf("%s: validator_name is %q, want %q", prefix, proof.ValidatorName, ValidatorName))
	}
	if proof.ValidatorStatus != "pass" {
		issues = append(issues, fmt.Sprintf("%s: validator_status is %q, want pass", prefix, proof.ValidatorStatus))
	}
	if unsafeUnknown(proof.ProvenanceClass, proof.UnsafeClass) {
		issues = append(issues, prefix+": unsafe_unknown cannot be promoted by island proof")
	}
	if noAliasProof(proof) && len(uniqueNonEmpty(proof.DistinctLiveIslands)) < 2 {
		issues = append(issues, prefix+": noalias proof requires distinct live islands")
	}
	if boundsProof(proof) && strings.TrimSpace(proof.Dominance) == "" {
		issues = append(issues, prefix+": bounds proof requires dominance evidence")
	}
	if storageProof(proof) && (strings.TrimSpace(proof.PlannedStorage) == "" || strings.TrimSpace(proof.ActualLoweringStorage) == "") {
		issues = append(issues, prefix+": storage proof requires planned_storage and actual_lowering_storage")
	}
	if storageProof(proof) && explicitIslandStorageFallback(proof.PlannedStorage, proof.ActualLoweringStorage) {
		issues = append(issues, fmt.Sprintf("%s: storage proof planned %q but actual_lowering_storage is %q", prefix, proof.PlannedStorage, proof.ActualLoweringStorage))
	}
	row, ok := rows[strings.TrimSpace(proof.SourceFactID)]
	if !ok {
		issues = append(issues, fmt.Sprintf("%s: source_fact_id %q not found in memory facts report", prefix, proof.SourceFactID))
		return issues
	}
	issues = append(issues, validateProofAgainstMemoryRow(prefix, proof, row)...)
	return issues
}

func validateProofAgainstMemoryRow(prefix string, proof Proof, row memoryReportRow) []string {
	var issues []string
	if row.Claim != "island_proof_verified" {
		issues = append(issues, fmt.Sprintf("%s: memory row claim is %q, want island_proof_verified", prefix, row.Claim))
	}
	if row.ClaimLevel != "validated" || row.ValidatorStatus != "pass" {
		issues = append(issues, fmt.Sprintf("%s: memory row must be validated/pass, got %q/%q", prefix, row.ClaimLevel, row.ValidatorStatus))
	}
	if row.ValidatorName != ValidatorName {
		issues = append(issues, fmt.Sprintf("%s: memory row validator_name is %q, want %q", prefix, row.ValidatorName, ValidatorName))
	}
	if missing := missingMemoryProofMetadata(row); len(missing) > 0 {
		issues = append(issues, fmt.Sprintf("%s: memory row lost proof metadata: %s", prefix, strings.Join(missing, ", ")))
	}
	if row.ProofID != "" && row.ProofID != proof.ProofID {
		issues = append(issues, fmt.Sprintf("%s: proof_id mismatch memory=%q proof=%q", prefix, row.ProofID, proof.ProofID))
	}
	if row.ProofOperation != "" && row.ProofOperation != proof.Operation {
		issues = append(issues, fmt.Sprintf("%s: operation mismatch memory=%q proof=%q", prefix, row.ProofOperation, proof.Operation))
	}
	if row.ProofKind != "" && row.ProofKind != proof.ProofKind {
		issues = append(issues, fmt.Sprintf("%s: proof_kind mismatch memory=%q proof=%q", prefix, row.ProofKind, proof.ProofKind))
	}
	if row.IslandID != proof.IslandID {
		issues = append(issues, fmt.Sprintf("%s: island_id mismatch memory=%q proof=%q", prefix, row.IslandID, proof.IslandID))
	}
	if row.Epoch != proof.Epoch {
		issues = append(issues, fmt.Sprintf("%s: epoch mismatch memory=%d proof=%d", prefix, row.Epoch, proof.Epoch))
	}
	if row.BaseID != proof.SubjectBaseID {
		issues = append(issues, fmt.Sprintf("%s: base_id mismatch memory=%q proof=%q", prefix, row.BaseID, proof.SubjectBaseID))
	}
	if row.ProofSubjectBaseID != proof.SubjectBaseID {
		issues = append(issues, fmt.Sprintf("%s: proof_subject_base_id mismatch memory=%q proof=%q", prefix, row.ProofSubjectBaseID, proof.SubjectBaseID))
	}
	if row.ProvenanceClass != proof.ProvenanceClass {
		issues = append(issues, fmt.Sprintf("%s: provenance_class mismatch memory=%q proof=%q", prefix, row.ProvenanceClass, proof.ProvenanceClass))
	}
	if row.UnsafeClass != proof.UnsafeClass {
		issues = append(issues, fmt.Sprintf("%s: unsafe_class mismatch memory=%q proof=%q", prefix, row.UnsafeClass, proof.UnsafeClass))
	}
	if unsafeUnknown(row.ProvenanceClass, row.UnsafeClass) {
		issues = append(issues, prefix+": memory row unsafe_unknown cannot support verified island proof")
	}
	if noAliasProof(proof) && !validatedNoAliasState(row.AliasState) {
		issues = append(issues, fmt.Sprintf("%s: noalias proof requires unique/mutable_exclusive memory alias_state, got %q", prefix, row.AliasState))
	}
	if storageProof(proof) {
		if row.PlannedStorage != proof.PlannedStorage || row.ActualLoweringStorage != proof.ActualLoweringStorage {
			issues = append(issues, fmt.Sprintf("%s: storage mismatch memory=%q/%q proof=%q/%q", prefix, row.PlannedStorage, row.ActualLoweringStorage, proof.PlannedStorage, proof.ActualLoweringStorage))
		}
	}
	return issues
}

func memoryRowsBySourceFactID(rows []memoryReportRow) map[string]memoryReportRow {
	out := map[string]memoryReportRow{}
	for _, row := range rows {
		id := strings.TrimSpace(row.SourceFactID)
		if id == "" {
			continue
		}
		if _, exists := out[id]; !exists {
			out[id] = row
		}
	}
	return out
}

func missingMemoryProofMetadata(row memoryReportRow) []string {
	var missing []string
	if strings.TrimSpace(row.ProofID) == "" {
		missing = append(missing, "proof_id")
	}
	if strings.TrimSpace(row.ProofKind) == "" {
		missing = append(missing, "proof_kind")
	}
	if strings.TrimSpace(row.ProofSubjectBaseID) == "" {
		missing = append(missing, "proof_subject_base_id")
	}
	if strings.TrimSpace(row.ProofOperation) == "" {
		missing = append(missing, "proof_operation")
	}
	return missing
}

func decodeStrict(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("JSON must contain a single document")
		}
		return fmt.Errorf("trailing data after JSON: %w", err)
	}
	return nil
}

func decodeMemoryReport(raw []byte, out *memoryReport) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(out); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("memory report JSON must contain a single document")
		}
		return fmt.Errorf("trailing data after memory report JSON: %w", err)
	}
	return nil
}

func decodeReleaseManifest(raw []byte, out *releaseManifest) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("release manifest JSON must contain a single document")
		}
		return fmt.Errorf("trailing data after release manifest JSON: %w", err)
	}
	return nil
}

func validateManifestAgainstReport(report Report, manifest releaseManifest) []string {
	var issues []string
	if strings.TrimSpace(manifest.Schema) == "" {
		issues = append(issues, "manifest schema is required")
	}
	if strings.TrimSpace(manifest.GitHead) != "" && manifest.GitHead != report.GitHead {
		issues = append(issues, fmt.Sprintf("manifest git_head %q does not match proof git_head %q", manifest.GitHead, report.GitHead))
	}

	commands := map[string]string{}
	for _, command := range manifest.Commands {
		name := strings.TrimSpace(command.Name)
		text := strings.TrimSpace(command.Command)
		if name == "" {
			issues = append(issues, "manifest command name is required")
			continue
		}
		if _, exists := commands[name]; exists {
			issues = append(issues, fmt.Sprintf("duplicate manifest command %q", name))
		}
		commands[name] = text
	}
	verifierCommand := commands["island-proof-verifier"]
	if strings.TrimSpace(verifierCommand) == "" {
		issues = append(issues, "missing manifest command island-proof-verifier")
	} else {
		issues = append(issues, validateVerifierCommand("manifest command island-proof-verifier", verifierCommand, report.ProducerCommand)...)
	}

	artifacts := map[string]releaseArtifact{}
	for _, artifact := range manifest.Artifacts {
		kind := strings.TrimSpace(artifact.Kind)
		if kind == "" {
			issues = append(issues, "manifest artifact kind is required")
			continue
		}
		if _, exists := artifacts[kind]; exists {
			issues = append(issues, fmt.Sprintf("duplicate manifest artifact kind %q", kind))
		}
		artifacts[kind] = artifact
	}
	issues = append(issues, validateManifestArtifact(artifacts, "island_proof_verifier_report", SchemaV1, report.ProducerCommand)...)
	issues = append(issues, validateManifestArtifact(artifacts, "island_proof_memory_report", "tetra.memory-report.v1", report.ProducerCommand)...)
	return issues
}

func validateManifestArtifact(artifacts map[string]releaseArtifact, kind string, schema string, producerCommand string) []string {
	artifact, ok := artifacts[kind]
	if !ok {
		return []string{fmt.Sprintf("missing manifest artifact %s", kind)}
	}
	var issues []string
	if strings.TrimSpace(artifact.Path) == "" {
		issues = append(issues, fmt.Sprintf("manifest artifact %s path is required", kind))
	}
	if schema != "" && artifact.Schema != schema {
		issues = append(issues, fmt.Sprintf("manifest artifact %s schema is %q, want %s", kind, artifact.Schema, schema))
	}
	issues = append(issues, validateVerifierCommand("manifest artifact "+kind, artifact.Command, producerCommand)...)
	return issues
}

func validateVerifierCommand(label string, command string, producerCommand string) []string {
	command = strings.TrimSpace(command)
	var issues []string
	if command == "" {
		return []string{label + " command is required"}
	}
	for _, fragment := range []string{"go run ./tools/cmd/validate-island-proof", "--proof", "--memory-report"} {
		if !strings.Contains(command, fragment) {
			issues = append(issues, fmt.Sprintf("%s command must contain %q", label, fragment))
		}
	}
	producerCommand = strings.TrimSpace(producerCommand)
	if producerCommand != "" && !strings.Contains(command, producerCommand) {
		issues = append(issues, fmt.Sprintf("%s command does not include producer_command %q", label, producerCommand))
	}
	return issues
}

func isGitHead(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, ch := range value {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}
	return true
}

func unsafeUnknown(provenance string, unsafeClass string) bool {
	return provenance == "unsafe_unknown" || unsafeClass == "unsafe_unknown"
}

func noAliasProof(proof Proof) bool {
	return strings.Contains(proof.Operation, "noalias") || strings.Contains(proof.Operation, "no_alias") || strings.Contains(proof.Claim, "noalias") || strings.Contains(proof.Claim, "no_alias")
}

func boundsProof(proof Proof) bool {
	return strings.Contains(proof.Operation, "bounds") || strings.Contains(proof.ProofKind, "bounds") || strings.Contains(proof.Claim, "bounds")
}

func storageProof(proof Proof) bool {
	return strings.Contains(proof.Operation, "storage") || strings.Contains(proof.Claim, "storage") || proof.PlannedStorage != "" || proof.ActualLoweringStorage != ""
}

func explicitIslandStorageFallback(planned string, actual string) bool {
	return strings.TrimSpace(planned) == "ExplicitIsland" && strings.TrimSpace(actual) != "ExplicitIsland"
}

func uniqueNonEmpty(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func validatedNoAliasState(aliasState string) bool {
	return aliasState == "unique" || aliasState == "mutable_exclusive"
}

func knownProvenanceClass(value string) bool {
	switch value {
	case "safe_known", "safe_borrowed", "safe_owned", "unsafe_unknown", "unsafe_checked", "unsafe_verified_root":
		return true
	default:
		return false
	}
}

func knownUnsafeClass(value string) bool {
	switch value {
	case "safe", "unsafe_unknown", "unsafe_checked", "unsafe_verified_root":
		return true
	default:
		return false
	}
}
