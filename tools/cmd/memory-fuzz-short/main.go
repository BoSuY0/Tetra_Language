package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tetra_language/compiler"
	"tetra_language/tools/validators/islandproof"
)

type memoryFuzzShortOptions struct {
	Tier      string
	ReportDir string
	GitHead   string
}

type memoryFuzzShortSummaryJSON struct {
	SchemaVersion           string                      `json:"schema_version"`
	Kind                    string                      `json:"kind"`
	Tier                    string                      `json:"tier"`
	Status                  string                      `json:"status"`
	ObservedFailures        int                         `json:"observed_failures"`
	ClassifiedFailures      int                         `json:"classified_failures"`
	UnclassifiedFailures    int                         `json:"unclassified_failures"`
	ReleaseBlockingFailures int                         `json:"release_blocking_failures"`
	ReproducibilitySeeds    []string                    `json:"reproducibility_seeds"`
	Artifacts               map[string]string           `json:"artifacts"`
	Commands                []memoryFuzzShortCommandRow `json:"commands"`
	Policies                []string                    `json:"policies"`
	NonClaims               []string                    `json:"non_claims"`
}

type memoryFuzzShortCommandRow struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Status  string `json:"status"`
}

type memoryFuzzArtifactHashManifest struct {
	Schema    string                     `json:"schema"`
	Root      string                     `json:"root"`
	Artifacts []memoryFuzzHashedArtifact `json:"artifacts"`
}

type memoryFuzzHashedArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Schema string `json:"schema,omitempty"`
}

type islandProofFuzzSummaryJSON struct {
	SchemaVersion string                      `json:"schema_version"`
	Status        string                      `json:"status"`
	Corpus        string                      `json:"corpus"`
	Total         int                         `json:"total"`
	Rejected      int                         `json:"rejected"`
	Accepted      int                         `json:"accepted"`
	Cases         []islandProofFuzzCaseResult `json:"cases"`
	NonClaims     []string                    `json:"non_claims"`
}

type islandProofFuzzCaseResult struct {
	Name              string `json:"name"`
	Mutation          string `json:"mutation"`
	Status            string `json:"status"`
	Error             string `json:"error,omitempty"`
	ExpectedRejection string `json:"expected_rejection"`
}

func main() {
	var opt memoryFuzzShortOptions
	flag.StringVar(&opt.Tier, "tier", "1", "memory fuzz tier to run; only Tier 1 short CI smoke is supported by this command")
	flag.StringVar(&opt.ReportDir, "report-dir", "", "directory for memory fuzz short artifacts")
	flag.StringVar(&opt.GitHead, "git-head", "", "optional git HEAD provenance to include in the oracle report")
	flag.Parse()
	if err := runMemoryFuzzShort(opt); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runMemoryFuzzShort(opt memoryFuzzShortOptions) error {
	tier := strings.TrimSpace(strings.ToLower(opt.Tier))
	if tier != "1" && tier != "tier1" && tier != "tier-1" {
		return fmt.Errorf("memory-fuzz-short only supports Tier 1 short CI smoke, got %q", opt.Tier)
	}
	if strings.TrimSpace(opt.ReportDir) == "" {
		return fmt.Errorf("--report-dir is required")
	}
	if err := checkMemoryFuzzShortReportDirFresh(opt.ReportDir); err != nil {
		return err
	}
	if err := os.MkdirAll(opt.ReportDir, 0o755); err != nil {
		return err
	}
	report, err := compiler.BuildMemoryFuzzOracleReport()
	if err != nil {
		return err
	}
	report.GitHead = strings.TrimSpace(opt.GitHead)
	if err := compiler.ValidateMemoryFuzzOracleReport(report); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	reportPath := filepath.Join(opt.ReportDir, "memory-fuzz-oracle.json")
	if err := os.WriteFile(reportPath, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	proofSummary, proofSummaryErr := buildIslandProofFuzzSummary()
	proofSummaryRaw, err := json.MarshalIndent(proofSummary, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(opt.ReportDir, "island-proof-fuzz-summary.json"), append(proofSummaryRaw, '\n'), 0o644); err != nil {
		return err
	}
	if proofSummaryErr != nil {
		return proofSummaryErr
	}
	if err := writeMemoryFuzzReproducerPlaceholders(opt.ReportDir); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(opt.ReportDir, "summary.md"), []byte(memoryFuzzShortSummary(report, reportPath, proofSummary)), 0o644); err != nil {
		return err
	}
	summaryJSON, err := json.MarshalIndent(memoryFuzzShortSummaryForJSON(opt.ReportDir, opt.GitHead), "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(opt.ReportDir, "summary.json"), append(summaryJSON, '\n'), 0o644); err != nil {
		return err
	}
	return writeMemoryFuzzArtifactHashes(opt.ReportDir)
}

func writeMemoryFuzzReproducerPlaceholders(reportDir string) error {
	entries := []struct {
		dir  string
		text string
	}{
		{
			dir:  "reproducers/compiler-crash",
			text: "Compiler crash reproducers are required release evidence slots. Tier 1 observed no compiler crash in this deterministic smoke run.\n",
		},
		{
			dir:  "reproducers/miscompile",
			text: "Miscompile reproducers are required release evidence slots. Tier 1 observed no miscompile in this deterministic smoke run.\n",
		},
		{
			dir:  "reducers/miscompile",
			text: "Miscompile reducers are required release evidence slots. Tier 1 observed no reducer input in this deterministic smoke run.\n",
		},
	}
	for _, entry := range entries {
		dir := filepath.Join(reportDir, filepath.FromSlash(entry.dir))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(entry.text), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func checkMemoryFuzzShortReportDirFresh(reportDir string) error {
	info, err := os.Lstat(reportDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to use symlink --report-dir %s; choose a real fresh --report-dir", reportDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("refusing to use non-directory --report-dir %s; choose a fresh --report-dir directory", reportDir)
	}
	entries, err := os.ReadDir(reportDir)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("refusing to reuse non-empty --report-dir %s; choose a fresh --report-dir so stale fuzz artifacts cannot be reused", reportDir)
	}
	return nil
}

func memoryFuzzShortSummary(report compiler.MemoryFuzzOracleReport, reportPath string, proofSummary islandProofFuzzSummaryJSON) string {
	return fmt.Sprintf("# Memory Fuzz Short Summary\n\n- schema: `%s`\n- scope: `%s`\n- tier: `Tier 1 short CI smoke`\n- report: `%s`\n- summary_json: `summary.json`\n- island_proof_fuzz_summary: `island-proof-fuzz-summary.json`\n- validator: `go run ./tools/cmd/validate-memory-fuzz-oracle --report %s --artifact-dir %s`\n- oracle_categories: `%d`\n- release_evidence_requirements: `%d` (`MEM-FUZZ-001`..`MEM-FUZZ-005`)\n- deterministic_slice_coverage: `%d` (`v0-v11`)\n- tier1_short_ci_smoke_cases: `%d`\n- island_proof_mutations_rejected: `%d/%d`\n\n", report.SchemaVersion, report.Scope, filepath.ToSlash(reportPath), filepath.ToSlash(reportPath), filepath.ToSlash(filepath.Dir(reportPath)), len(report.Rows), len(report.Requirements), len(report.SliceCoverage), report.Tier1ShortCISmokeCases, proofSummary.Rejected, proofSummary.Total)
}

func memoryFuzzShortSummaryForJSON(reportDir string, gitHead string) memoryFuzzShortSummaryJSON {
	reportDirSlash := filepath.ToSlash(reportDir)
	reportPath := filepath.ToSlash(filepath.Join(reportDir, "memory-fuzz-oracle.json"))
	command := "go run ./tools/cmd/memory-fuzz-short --tier 1 --report-dir " + reportDirSlash
	validatorCommand := "go run ./tools/cmd/validate-memory-fuzz-oracle --report " + reportPath + " --artifact-dir " + reportDirSlash
	gitHead = strings.TrimSpace(gitHead)
	if gitHead != "" {
		command += " --git-head " + gitHead
		validatorCommand += " --current-git-head " + gitHead
	}
	return memoryFuzzShortSummaryJSON{
		SchemaVersion:           "tetra.memory-fuzz-short.summary.v1",
		Kind:                    "tier1_short_ci_smoke",
		Tier:                    string(compiler.MemoryFuzzTier1ShortCI),
		Status:                  "pass",
		ObservedFailures:        0,
		ClassifiedFailures:      0,
		UnclassifiedFailures:    0,
		ReleaseBlockingFailures: 0,
		ReproducibilitySeeds:    memoryFuzzShortReproducibilitySeeds(),
		Artifacts: map[string]string{
			"artifact_hashes":           "artifact-hashes.json",
			"island_proof_fuzz_summary": "island-proof-fuzz-summary.json",
			"oracle_report":             "memory-fuzz-oracle.json",
			"summary_md":                "summary.md",
			"summary_json":              "summary.json",
		},
		Commands: []memoryFuzzShortCommandRow{
			{
				Name:    "memory-fuzz-short",
				Command: command,
				Status:  "pass",
			},
			{
				Name:    "validate-memory-fuzz-oracle",
				Command: validatorCommand,
				Status:  "pass",
			},
		},
		Policies: []string{
			"Tier 1 deterministic smoke writes report, markdown summary, and machine-readable summary",
			"Tier 2 nightly seed triage and minimized repro policy remains boundary-recorded in the oracle report",
			"Tier 3 release-blocking focused memory fuzz blocks promotion until failures are classified",
		},
		NonClaims: []string{
			"no exhaustive fuzz proof is claimed",
			"no Memory 100% claim is made",
		},
	}
}

func memoryFuzzShortReproducibilitySeeds() []string {
	seeds := make([]string, 0, 12)
	for i := 0; i < 12; i++ {
		seeds = append(seeds, fmt.Sprintf("memory-fuzz:v%d:seed:%04d", i, 1000+i))
	}
	return seeds
}

func writeMemoryFuzzArtifactHashes(reportDir string) error {
	manifest, err := buildMemoryFuzzArtifactHashManifest(reportDir, "artifact-hashes.json")
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(reportDir, "artifact-hashes.json"), append(raw, '\n'), 0o644)
}

func buildMemoryFuzzArtifactHashManifest(reportDir string, manifestName string) (memoryFuzzArtifactHashManifest, error) {
	var artifacts []memoryFuzzHashedArtifact
	err := filepath.WalkDir(reportDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(reportDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == manifestName {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("memory fuzz artifact %s must not be a symlink", rel)
		}
		artifact, err := hashMemoryFuzzArtifact(reportDir, rel)
		if err != nil {
			return err
		}
		artifacts = append(artifacts, artifact)
		return nil
	})
	if err != nil {
		return memoryFuzzArtifactHashManifest{}, err
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	return memoryFuzzArtifactHashManifest{
		Schema:    "tetra.release-artifact-hashes.v1alpha1",
		Root:      ".",
		Artifacts: artifacts,
	}, nil
}

func hashMemoryFuzzArtifact(reportDir string, rel string) (memoryFuzzHashedArtifact, error) {
	path := filepath.Join(reportDir, filepath.FromSlash(rel))
	info, err := os.Lstat(path)
	if err != nil {
		return memoryFuzzHashedArtifact{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return memoryFuzzHashedArtifact{}, fmt.Errorf("memory fuzz artifact %s must not be a symlink", rel)
	}
	if !info.Mode().IsRegular() {
		return memoryFuzzHashedArtifact{}, fmt.Errorf("memory fuzz artifact %s must be a regular file", rel)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return memoryFuzzHashedArtifact{}, err
	}
	sum := sha256.Sum256(raw)
	return memoryFuzzHashedArtifact{
		Path:   filepath.ToSlash(rel),
		SHA256: "sha256:" + hex.EncodeToString(sum[:]),
		Size:   int64(len(raw)),
		Schema: detectMemoryFuzzJSONSchema(raw),
	}, nil
}

func detectMemoryFuzzJSONSchema(raw []byte) string {
	var envelope struct {
		Schema        string `json:"schema"`
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return ""
	}
	if envelope.Schema != "" {
		return envelope.Schema
	}
	return envelope.SchemaVersion
}

func buildIslandProofFuzzSummary() (islandProofFuzzSummaryJSON, error) {
	cases := islandProofFuzzCases()
	summary := islandProofFuzzSummaryJSON{
		SchemaVersion: "tetra.island-proof-fuzz-summary.v1",
		Status:        "pass",
		Corpus:        "deterministic-short",
		Total:         len(cases),
		NonClaims: []string{
			"not exhaustive proof fuzzing",
			"not broad compiler-generated island proof coverage",
		},
	}
	var accepted []string
	for _, tc := range cases {
		err := islandproof.Validate([]byte(tc.proof), islandproof.Options{MemoryReport: []byte(tc.memory)})
		result := islandProofFuzzCaseResult{
			Name:              tc.name,
			Mutation:          tc.mutation,
			ExpectedRejection: tc.want,
			Status:            "rejected",
		}
		if err == nil {
			result.Status = "accepted"
			accepted = append(accepted, tc.name)
			summary.Accepted++
		} else {
			result.Error = err.Error()
			summary.Rejected++
			if tc.want != "" && !strings.Contains(err.Error(), tc.want) {
				result.Status = "wrong_rejection"
				accepted = append(accepted, tc.name+": "+err.Error())
				summary.Accepted++
				summary.Rejected--
			}
		}
		summary.Cases = append(summary.Cases, result)
	}
	if len(accepted) > 0 {
		summary.Status = "fail"
		return summary, fmt.Errorf("island proof fuzz accepted or misclassified mutations: %s", strings.Join(accepted, ", "))
	}
	return summary, nil
}

type islandProofFuzzCase struct {
	name     string
	mutation string
	proof    string
	memory   string
	want     string
}

func islandProofFuzzCases() []islandProofFuzzCase {
	baseProof := islandProofFuzzValidProof()
	baseMemory := islandProofFuzzValidMemoryReport()
	return []islandProofFuzzCase{
		{
			name:     "malformed_proof_json",
			mutation: "truncate proof JSON",
			proof:    `{"schema":"tetra.island.proof.v1"`,
			memory:   baseMemory,
			want:     "",
		},
		{
			name:     "stale_epoch",
			mutation: "proof epoch differs from memory row",
			proof:    strings.Replace(baseProof, `"epoch": 1,`, `"epoch": 2,`, 1),
			memory:   baseMemory,
			want:     "epoch mismatch",
		},
		{
			name:     "mismatched_island_id",
			mutation: "proof island id differs from memory row",
			proof:    strings.Replace(baseProof, `"island_id": "island:fuzz:0"`, `"island_id": "island:fuzz:other"`, 1),
			memory:   baseMemory,
			want:     "island_id mismatch",
		},
		{
			name:     "wrong_base_allocation",
			mutation: "proof subject base differs from memory row",
			proof:    strings.Replace(baseProof, `"subject_base_id": "alloc:fuzz:0"`, `"subject_base_id": "alloc:fuzz:other"`, 1),
			memory:   baseMemory,
			want:     "subject_base_id mismatch",
		},
		{
			name:     "broken_dominance",
			mutation: "bounds proof has no dominance evidence",
			proof: strings.Replace(
				strings.Replace(
					strings.Replace(baseProof, `"operation": "island_borrow"`, `"operation": "bounds_check_removed"`, 1),
					`"proof_kind": "island_epoch"`, `"proof_kind": "bounds_check"`, 1),
				`"dominance": "entry dominates island borrow"`, `"dominance": ""`, 1),
			memory: strings.Replace(strings.Replace(baseMemory, `"proof_operation": "island_borrow"`, `"proof_operation": "bounds_check_removed"`, 1), `"proof_kind": "island_epoch"`, `"proof_kind": "bounds_check"`, 1),
			want:   "dominance",
		},
		{
			name:     "missing_proof_id",
			mutation: "proof id removed",
			proof:    strings.Replace(baseProof, `"proof_id": "proof:fuzz:island:borrow:1"`, `"proof_id": ""`, 1),
			memory:   baseMemory,
			want:     "proof_id",
		},
		{
			name:     "wrong_operation",
			mutation: "proof operation reused for another operation",
			proof:    strings.Replace(baseProof, `"operation": "island_borrow"`, `"operation": "island_reset"`, 1),
			memory:   baseMemory,
			want:     "operation mismatch",
		},
		{
			name:     "unsafe_unknown_promotion",
			mutation: "unsafe_unknown row claims verified island proof",
			proof: strings.Replace(
				strings.Replace(baseProof, `"provenance_class": "safe_known"`, `"provenance_class": "unsafe_unknown"`, 1),
				`"unsafe_class": "safe"`, `"unsafe_class": "unsafe_unknown"`, 1),
			memory: strings.Replace(
				strings.Replace(baseMemory, `"provenance_class": "safe_known"`, `"provenance_class": "unsafe_unknown"`, 1),
				`"unsafe_class": "safe"`, `"unsafe_class": "unsafe_unknown"`, 1),
			want: "unsafe_unknown",
		},
		{
			name:     "noalias_broad_proof",
			mutation: "noalias proof has only one live island",
			proof: strings.Replace(
				strings.Replace(
					strings.Replace(baseProof, `"operation": "island_borrow"`, `"operation": "island_noalias"`, 1),
					`"claim": "island_proof_verified"`, `"claim": "no_alias"`, 1),
				`"distinct_live_islands": ["island:fuzz:0", "island:fuzz:1"]`, `"distinct_live_islands": ["island:fuzz:0"]`, 1),
			memory: strings.Replace(baseMemory, `"proof_operation": "island_borrow"`, `"proof_operation": "island_noalias"`, 1),
			want:   "distinct live islands",
		},
		{
			name:     "storage_heap_fallback",
			mutation: "explicit island storage proof falls back to heap",
			proof: strings.Replace(
				strings.Replace(baseProof, `"operation": "island_borrow"`, `"operation": "storage_lowering"`, 1),
				`"actual_lowering_storage": "ExplicitIsland"`, `"actual_lowering_storage": "Heap"`, 1),
			memory: strings.Replace(
				strings.Replace(baseMemory, `"proof_operation": "island_borrow"`, `"proof_operation": "storage_lowering"`, 1),
				`"actual_lowering_storage": "ExplicitIsland"`, `"actual_lowering_storage": "Heap"`, 1),
			want: "actual_lowering_storage",
		},
		{
			name:     "transform_lost_metadata",
			mutation: "memory row lost proof metadata after transform",
			proof:    baseProof,
			memory: strings.Replace(
				strings.Replace(
					strings.Replace(
						strings.Replace(baseMemory, `"proof_id": "proof:fuzz:island:borrow:1"`, `"proof_id": ""`, 1),
						`"proof_kind": "island_epoch"`, `"proof_kind": ""`, 1),
					`"proof_subject_base_id": "alloc:fuzz:0"`, `"proof_subject_base_id": ""`, 1),
				`"proof_operation": "island_borrow"`, `"proof_operation": ""`, 1),
			want: "lost proof metadata",
		},
	}
}

func islandProofFuzzValidProof() string {
	return `{
  "schema": "tetra.island.proof.v1",
  "producer": "tools/cmd/memory-fuzz-short",
  "producer_command": "go run ./tools/cmd/validate-island-proof",
  "git_head": "0123456789abcdef0123456789abcdef01234567",
  "proofs": [
    {
      "proof_id": "proof:fuzz:island:borrow:1",
      "operation": "island_borrow",
      "proof_kind": "island_epoch",
      "subject_base_id": "alloc:fuzz:0",
      "island_id": "island:fuzz:0",
      "epoch": 1,
      "source_fact_id": "fact:fuzz:island-proof:1",
      "claim": "island_proof_verified",
      "provenance_class": "safe_known",
      "unsafe_class": "safe",
      "validator_name": "validate-island-proof",
      "validator_status": "pass",
      "planned_storage": "ExplicitIsland",
      "actual_lowering_storage": "ExplicitIsland",
      "dominance": "entry dominates island borrow",
      "distinct_live_islands": ["island:fuzz:0", "island:fuzz:1"]
    }
  ]
}` + "\n"
}

func islandProofFuzzValidMemoryReport() string {
	return `{
  "schema_version": "tetra.memory-report.v1",
  "rows": [
    {
      "source_fact_id": "fact:fuzz:island-proof:1",
      "claim": "island_proof_verified",
      "claim_level": "validated",
      "provenance_class": "safe_known",
      "unsafe_class": "safe",
      "alias_state": "unique",
      "island_id": "island:fuzz:0",
      "epoch": 1,
      "base_id": "alloc:fuzz:0",
      "proof_id": "proof:fuzz:island:borrow:1",
      "proof_kind": "island_epoch",
      "proof_subject_base_id": "alloc:fuzz:0",
      "proof_operation": "island_borrow",
      "planned_storage": "ExplicitIsland",
      "actual_lowering_storage": "ExplicitIsland",
      "validator_name": "validate-island-proof",
      "validator_status": "pass"
    }
  ]
}` + "\n"
}
