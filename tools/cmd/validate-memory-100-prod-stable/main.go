package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tetra_language/tools/internal/ramvalidate"
	"tetra_language/tools/validators/islandproof"
	"tetra_language/tools/validators/memoryprod"
	"tetra_language/tools/validators/surface"
)

const (
	memory100ManifestSchema   = "tetra.memory-100.prod-stable.v1"
	memory100HashSchema       = "tetra.release-artifact-hashes.v1alpha1"
	memory100ScopedReadyLocal = "MEMORY100_SCOPED_READY_LOCAL"
	memory100ScopedReadyDirty = "MEMORY100_SCOPED_READY_DIRTY"
)

type memory100Manifest struct {
	Schema       string              `json:"schema"`
	Status       string              `json:"status"`
	Verdict      string              `json:"verdict"`
	GitHead      string              `json:"git_head"`
	GitDirty     *bool               `json:"git_dirty"`
	GitStatus    []string            `json:"git_status_short_branch"`
	GeneratedAt  string              `json:"generated_at"`
	TargetMatrix []string            `json:"target_matrix"`
	HashManifest string              `json:"hash_manifest"`
	Claims       []string            `json:"claims"`
	NonClaims    []string            `json:"non_claims"`
	Commands     []memory100Command  `json:"commands"`
	Artifacts    []memory100Artifact `json:"artifacts"`
}

type memory100Command struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

type memory100Artifact struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Schema string `json:"schema,omitempty"`
}

type memory100RequiredArtifact struct {
	Path   string
	Kind   string
	Schema string
}

type memory100HashManifest struct {
	Schema    string                  `json:"schema"`
	Root      string                  `json:"root"`
	Artifacts []memory100HashArtifact `json:"artifacts"`
}

type memory100HashArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Schema string `json:"schema,omitempty"`
}

type memory100SchemaEnvelope struct {
	Schema        string `json:"schema"`
	SchemaVersion string `json:"schema_version"`
	GitHead       string `json:"git_head"`
}

type memory100MemoryFuzzSummary struct {
	SchemaVersion           string                       `json:"schema_version"`
	Kind                    string                       `json:"kind"`
	Tier                    string                       `json:"tier"`
	Status                  string                       `json:"status"`
	ObservedFailures        *int                         `json:"observed_failures"`
	ClassifiedFailures      *int                         `json:"classified_failures"`
	UnclassifiedFailures    *int                         `json:"unclassified_failures"`
	ReleaseBlockingFailures *int                         `json:"release_blocking_failures"`
	ReproducibilitySeeds    []string                     `json:"reproducibility_seeds"`
	Artifacts               map[string]string            `json:"artifacts"`
	Commands                []memory100MemoryFuzzCommand `json:"commands"`
	Policies                []string                     `json:"policies,omitempty"`
	NonClaims               []string                     `json:"non_claims,omitempty"`
}

type memory100MemoryFuzzCommand struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Status  string `json:"status"`
}

type memory100MemoryReleaseManifest struct {
	Schema       string                              `json:"schema"`
	Target       string                              `json:"target"`
	GitHead      string                              `json:"git_head"`
	GeneratedAt  string                              `json:"generated_at"`
	ReportDir    string                              `json:"report_dir"`
	HashManifest string                              `json:"hash_manifest"`
	Commands     []memory100Command                  `json:"commands"`
	Artifacts    []memory100MemoryReleaseArtifactRef `json:"artifacts"`
}

type memory100MemoryReleaseArtifactRef struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Schema  string `json:"schema,omitempty"`
	Target  string `json:"target"`
	Command string `json:"command"`
}

type memory100RequiredMemoryReleaseArtifactRef struct {
	Path            string
	Kind            string
	Schema          string
	CommandFragment string
}

type memory100RAMContractReleaseManifest struct {
	Schema       string                                   `json:"schema"`
	Status       string                                   `json:"status"`
	Target       string                                   `json:"target"`
	GitHead      string                                   `json:"git_head"`
	GeneratedAt  string                                   `json:"generated_at"`
	ReportDir    string                                   `json:"report_dir"`
	HashManifest string                                   `json:"hash_manifest"`
	Commands     []memory100Command                       `json:"commands"`
	Artifacts    []memory100RAMContractReleaseArtifactRef `json:"artifacts"`
	NonClaims    []string                                 `json:"non_claims"`
}

type memory100RAMContractReleaseArtifactRef struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Schema string `json:"schema,omitempty"`
}

type memory100RequiredRAMContractReleaseArtifactRef struct {
	Path   string
	Kind   string
	Schema string
}

type memory100IntegratedManifest struct {
	Schema       string                           `json:"schema"`
	Status       string                           `json:"status"`
	GitHead      string                           `json:"git_head"`
	GeneratedAt  string                           `json:"generated_at"`
	ReportDir    string                           `json:"report_dir"`
	HashManifest string                           `json:"hash_manifest"`
	Commands     []memory100Command               `json:"commands"`
	Artifacts    []memory100IntegratedArtifactRef `json:"artifacts"`
}

type memory100IntegratedArtifactRef struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Schema string `json:"schema,omitempty"`
}

type memory100RequiredIntegratedArtifactRef struct {
	Path   string
	Kind   string
	Schema string
}

type memory100AllocationLoweringReport struct {
	Status    string                                `json:"status"`
	GitHead   string                                `json:"git_head"`
	Decisions []memory100AllocationLoweringDecision `json:"decisions"`
}

type memory100AllocationLoweringDecision struct {
	Name                  string   `json:"name"`
	Status                string   `json:"status"`
	PlannedStorage        string   `json:"planned_storage"`
	ActualLoweringStorage string   `json:"actual_lowering_storage"`
	ProofArtifact         string   `json:"proof_artifact"`
	BlockerArtifact       string   `json:"blocker_artifact"`
	BlockerReason         string   `json:"blocker_reason"`
	BudgetImpact          string   `json:"budget_impact"`
	GradeImpact           string   `json:"grade_impact"`
	ValidatorCoverage     []string `json:"validator_coverage"`
	SourceArtifacts       []string `json:"source_artifacts"`
	CoveredSiteIDs        []string `json:"covered_site_ids"`
}

type memory100IslandProofFuzzSummary struct {
	SchemaVersion string `json:"schema_version"`
	Status        string `json:"status"`
	Corpus        string `json:"corpus,omitempty"`
	Total         int    `json:"total"`
	Rejected      int    `json:"rejected"`
	Accepted      int    `json:"accepted"`
	Cases         []struct {
		Name              string `json:"name"`
		Status            string `json:"status"`
		Mutation          string `json:"mutation,omitempty"`
		Error             string `json:"error,omitempty"`
		ExpectedRejection string `json:"expected_rejection,omitempty"`
	} `json:"cases"`
	NonClaims []string `json:"non_claims,omitempty"`
}

type memory100ProofTransitionReport struct {
	Schema    string                        `json:"schema"`
	Status    string                        `json:"status"`
	GitHead   string                        `json:"git_head"`
	Rows      []memory100ProofTransitionRow `json:"rows"`
	NonClaims []string                      `json:"non_claims"`
}

type memory100ProofTransitionRow struct {
	Name            string   `json:"name"`
	Transition      string   `json:"transition"`
	Evidence        string   `json:"evidence"`
	BeforeArtifact  string   `json:"before_artifact,omitempty"`
	AfterArtifact   string   `json:"after_artifact,omitempty"`
	ConsumerAction  string   `json:"consumer_action,omitempty"`
	SourceArtifacts []string `json:"source_artifacts"`
	Tests           []string `json:"tests"`
}

type memory100RuntimeMemoryContract struct {
	Schema    string                      `json:"schema"`
	Status    string                      `json:"status"`
	GitHead   string                      `json:"git_head"`
	Rows      []memory100RuntimeMemoryRow `json:"rows"`
	NonClaims []string                    `json:"non_claims"`
}

type memory100RuntimeMemoryRow struct {
	Target                          string   `json:"target"`
	IncludedInMemory100TargetMatrix bool     `json:"included_in_memory100_target_matrix"`
	RuntimeStatus                   string   `json:"runtime_status"`
	MemoryRun                       string   `json:"memory_run"`
	MemoryClaimLevel                string   `json:"memory_claim_level"`
	Evidence                        string   `json:"evidence"`
	ExcludedReason                  string   `json:"excluded_reason,omitempty"`
	SourceArtifacts                 []string `json:"source_artifacts"`
	Tests                           []string `json:"tests"`
	NonClaims                       []string `json:"non_claims"`
}

type memory100TargetsReport struct {
	Supported []string                    `json:"supported"`
	BuildOnly []string                    `json:"build_only"`
	Planned   []string                    `json:"planned"`
	Targets   []memory100TargetsReportRow `json:"targets"`
}

type memory100TargetsReportRow struct {
	Triple                   string   `json:"triple"`
	Status                   string   `json:"status"`
	OS                       string   `json:"os"`
	Arch                     string   `json:"arch"`
	ABI                      string   `json:"abi"`
	DataModel                string   `json:"data_model"`
	Format                   string   `json:"format"`
	ExeExt                   string   `json:"exe_ext"`
	BuildOnly                bool     `json:"build_only"`
	RunMode                  string   `json:"run_mode"`
	RunRunner                string   `json:"run_runner,omitempty"`
	RunSupported             bool     `json:"run_supported"`
	RunUnsupportedReason     string   `json:"run_unsupported_reason,omitempty"`
	UIRuntimeContract        string   `json:"ui_runtime_contract,omitempty"`
	UIRuntimeStatus          string   `json:"ui_runtime_status,omitempty"`
	UIRuntimeEvidence        string   `json:"ui_runtime_evidence,omitempty"`
	PointerWidthBits         int      `json:"pointer_width_bits"`
	RegisterWidthBits        int      `json:"register_width_bits"`
	NativeIntWidthBits       int      `json:"native_int_width_bits"`
	Endian                   string   `json:"endian"`
	StackAlignmentBytes      int      `json:"stack_alignment_bytes"`
	MaxAtomicWidthBits       int      `json:"max_atomic_width_bits"`
	AtomicWidthBits          []int    `json:"atomic_width_bits"`
	AtomicPointerWidthBits   int      `json:"atomic_pointer_width_bits"`
	UnsupportedReason        string   `json:"unsupported_reason,omitempty"`
	RuntimeStatus            string   `json:"runtime_status,omitempty"`
	StdlibStatus             string   `json:"stdlib_status,omitempty"`
	FFIStatus                string   `json:"ffi_status,omitempty"`
	MemoryBuild              string   `json:"memory_build"`
	MemoryLower              string   `json:"memory_lower"`
	MemoryRun                string   `json:"memory_run"`
	MemoryRawDiagnostics     string   `json:"memory_raw_diagnostics"`
	MemoryRegionLowering     string   `json:"memory_region_lowering"`
	MemoryAlignmentSemantics string   `json:"memory_alignment_semantics"`
	MemoryClaimLevel         string   `json:"memory_claim_level"`
	RunnerProbeCommand       string   `json:"runner_probe_command,omitempty"`
	ReleaseGate              string   `json:"release_gate,omitempty"`
	EvidenceArtifacts        []string `json:"evidence_artifacts,omitempty"`
	SyscallInstruction       string   `json:"syscall_instruction,omitempty"`
	SyscallNumbering         string   `json:"syscall_numbering,omitempty"`
	SyscallArgRegisters      []string `json:"syscall_arg_registers,omitempty"`
	SyscallErrorRange        string   `json:"syscall_error_range,omitempty"`
	SupportsDebugInfo        bool     `json:"supports_debug_info"`
	SupportsReleaseOptimize  bool     `json:"supports_release_optimize"`
}

var requiredMemory100Artifacts = []memory100RequiredArtifact{
	{Path: "memory-production/memory-production-linux-x64.json", Kind: "memory_production_report", Schema: "tetra.memory.production.v1"},
	{Path: "memory-production/memory-release-manifest.json", Kind: "memory_release_manifest", Schema: "tetra.memory.release-manifest.v1"},
	{Path: "memory-production/artifact-hashes.json", Kind: "memory_production_hash_manifest", Schema: memory100HashSchema},
	{Path: "ram-contract/ram-contract-release-manifest.json", Kind: "ram_contract_release_manifest", Schema: "tetra.ram-contract.release-manifest.v1"},
	{Path: "ram-contract/ram-contract-report.json", Kind: "ram_contract_report", Schema: "tetra.ram-contract-report.v1"},
	{Path: "ram-contract/memory-grade-report.json", Kind: "ram_memory_grade_report", Schema: "tetra.memory-grade-report.v1"},
	{Path: "ram-contract/proof-store-summary.json", Kind: "ram_proof_store_summary", Schema: "tetra.proof-store-summary.v1"},
	{Path: "ram-contract/validation-pipeline-coverage.json", Kind: "ram_validation_pipeline_coverage", Schema: "tetra.validation-pipeline-coverage.v1"},
	{Path: "ram-contract/heap-blockers.json", Kind: "ram_heap_blockers", Schema: "tetra.ram-blockers.v1"},
	{Path: "ram-contract/copy-blockers.json", Kind: "ram_copy_blockers", Schema: "tetra.ram-blockers.v1"},
	{Path: "ram-contract/fuzz/ram-contract-fuzz-oracle.json", Kind: "ram_contract_fuzz_oracle", Schema: "tetra.ram-contract-fuzz-oracle.v1"},
	{Path: "ram-contract/artifact-hashes.json", Kind: "ram_contract_hash_manifest", Schema: memory100HashSchema},
	{Path: "raw-memory-contract/raw-memory-contract.json", Kind: "raw_memory_contract_report", Schema: "tetra.raw-memory-contract.v1"},
	{Path: "allocation-lowering/allocation-lowering-report.json", Kind: "allocation_lowering_report", Schema: "tetra.allocation-lowering.v1"},
	{Path: "proof-store/proof-store-summary.json", Kind: "proof_store_summary", Schema: "tetra.proof-store-summary.v1"},
	{Path: "proof-transition/proof-transition-report.json", Kind: "proof_transition_report", Schema: "tetra.proof-transition-report.v1"},
	{Path: "runtime-memory/runtime-memory-contract.json", Kind: "runtime_memory_contract", Schema: "tetra.runtime-memory-contract.v1"},
	{Path: "memory-fuzz/memory-fuzz-oracle.json", Kind: "memory_fuzz_oracle_report", Schema: "tetra.memory-fuzz.oracle.v1"},
	{Path: "memory-fuzz/artifact-hashes.json", Kind: "memory_fuzz_hash_manifest", Schema: memory100HashSchema},
	{Path: "semantic-safety/memory-semantic-safety-matrix.json", Kind: "memory_semantic_safety_matrix", Schema: "tetra.memory-semantic-safety-matrix.v1"},
	{Path: "leak-resource/leak-resource-report.json", Kind: "leak_resource_report", Schema: "tetra.leak-resource.v1"},
	{Path: "integrated/memory-islands-surface-production-manifest.json", Kind: "integrated_memory_islands_surface_manifest", Schema: "tetra.memory-islands-surface.production-gate.v1"},
	{Path: "integrated/artifact-hashes.json", Kind: "integrated_hash_manifest", Schema: memory100HashSchema},
	{Path: "docs-manifest/claim-policy.json", Kind: "docs_claim_policy", Schema: "tetra.memory-100.claim-policy.v1"},
}

var requiredMemory100Commands = map[string]string{
	"memory-production-gate": "memory-production-linux-x64-smoke.sh",
	"ram-contract-gate":      "ram-contract-linux-x64-smoke.sh",
	"integrated-gate":        "memory-islands-surface-production-gate.sh",
	"memory-fuzz-short":      "go run ./tools/cmd/memory-fuzz-short",
	"memory-fuzz-validator":  "go run ./tools/cmd/validate-memory-fuzz-oracle",
	"docs-claim-policy":      "verify-docs",
	"artifact-hashes-write":  "validate-artifact-hashes --write",
	"memory-100-validator":   "validate-memory-100-prod-stable",
}

var requiredMemory100MemoryReleaseCommands = map[string]string{
	"memory-production-smoke":     "go run ./tools/cmd/memory-production-smoke",
	"target-report":               "go run ./cli/cmd/tetra targets",
	"validate-targets":            "go run ./tools/cmd/validate-targets",
	"memory-fuzz-short":           "go run ./tools/cmd/memory-fuzz-short",
	"validate-memory-fuzz-oracle": "go run ./tools/cmd/validate-memory-fuzz-oracle",
	"ram-contract-gate":           "ram-contract-linux-x64-smoke.sh",
	"island-proof-verifier":       "go run ./tools/cmd/validate-island-proof",
	"artifact-hashes-write":       "go run ./tools/cmd/validate-artifact-hashes --write",
	"artifact-hashes-validate":    "go run ./tools/cmd/validate-artifact-hashes --manifest",
}

var requiredMemory100MemoryReleaseArtifacts = []memory100RequiredMemoryReleaseArtifactRef{
	{Path: "memory-production-linux-x64.json", Kind: "memory_production_report", Schema: "tetra.memory.production.v1", CommandFragment: "go run ./tools/cmd/memory-production-smoke"},
	{Path: "targets.json", Kind: "target_report", CommandFragment: "go run ./cli/cmd/tetra targets"},
	{Path: "memory-fuzz-tier1/memory-fuzz-oracle.json", Kind: "memory_fuzz_oracle_report", Schema: "tetra.memory-fuzz.oracle.v1", CommandFragment: "go run ./tools/cmd/memory-fuzz-short"},
	{Path: "memory-fuzz-tier1/summary.json", Kind: "memory_fuzz_summary", Schema: "tetra.memory-fuzz-short.summary.v1", CommandFragment: "go run ./tools/cmd/memory-fuzz-short"},
	{Path: "memory-fuzz-tier1/island-proof-fuzz-summary.json", Kind: "memory_fuzz_island_proof_summary", Schema: "tetra.island-proof-fuzz-summary.v1", CommandFragment: "go run ./tools/cmd/memory-fuzz-short"},
	{Path: "ram-contract/ram-contract-release-manifest.json", Kind: "ram_contract_release_manifest", Schema: "tetra.ram-contract.release-manifest.v1", CommandFragment: "ram-contract-linux-x64-smoke.sh"},
	{Path: "ram-contract/ram-contract-report.json", Kind: "ram_contract_report", Schema: "tetra.ram-contract-report.v1", CommandFragment: "ram-contract-linux-x64-smoke.sh"},
	{Path: "ram-contract/memory-grade-report.json", Kind: "ram_memory_grade_report", Schema: "tetra.memory-grade-report.v1", CommandFragment: "ram-contract-linux-x64-smoke.sh"},
	{Path: "ram-contract/proof-store-summary.json", Kind: "ram_proof_store_summary", Schema: "tetra.proof-store-summary.v1", CommandFragment: "ram-contract-linux-x64-smoke.sh"},
	{Path: "ram-contract/validation-pipeline-coverage.json", Kind: "ram_validation_pipeline_coverage", Schema: "tetra.validation-pipeline-coverage.v1", CommandFragment: "ram-contract-linux-x64-smoke.sh"},
	{Path: "ram-contract/heap-blockers.json", Kind: "ram_heap_blockers", Schema: "tetra.ram-blockers.v1", CommandFragment: "ram-contract-linux-x64-smoke.sh"},
	{Path: "ram-contract/copy-blockers.json", Kind: "ram_copy_blockers", Schema: "tetra.ram-blockers.v1", CommandFragment: "ram-contract-linux-x64-smoke.sh"},
	{Path: "ram-contract/fuzz/ram-contract-fuzz-oracle.json", Kind: "ram_contract_fuzz_oracle", Schema: "tetra.ram-contract-fuzz-oracle.v1", CommandFragment: "ram-contract-linux-x64-smoke.sh"},
	{Path: "ram-contract/artifact-hashes.json", Kind: "ram_contract_hash_manifest", Schema: memory100HashSchema, CommandFragment: "ram-contract-linux-x64-smoke.sh"},
	{Path: "island-proof-verifier.json", Kind: "island_proof_verifier_report", Schema: "tetra.island.proof.v1", CommandFragment: "go run ./tools/cmd/validate-island-proof"},
	{Path: "island-proof-memory-report.json", Kind: "island_proof_memory_report", Schema: "tetra.memory-report.v1", CommandFragment: "go run ./tools/cmd/validate-island-proof"},
	{Path: "artifact-hashes.json", Kind: "artifact_hash_manifest", Schema: memory100HashSchema, CommandFragment: "go run ./tools/cmd/validate-artifact-hashes --write"},
}

var requiredMemory100RAMContractReleaseCommands = map[string]string{
	"ram-contract-build":                    "go run ./cli/cmd/tetra build",
	"validate-ram-contract-report":          "go run ./tools/cmd/validate-ram-contract-report",
	"validate-memory-grade-report":          "go run ./tools/cmd/validate-memory-grade-report",
	"validate-proof-store-summary":          "go run ./tools/cmd/validate-proof-store-summary",
	"validate-validation-pipeline-coverage": "go run ./tools/cmd/validate-validation-pipeline-coverage",
	"validate-heap-blockers":                "go run ./tools/cmd/validate-heap-blockers",
	"validate-copy-blockers":                "go run ./tools/cmd/validate-copy-blockers",
	"ram-contract-fuzz-short":               "go run ./tools/cmd/ram-contract-fuzz-short",
	"validate-ram-contract-fuzz-oracle":     "go run ./tools/cmd/validate-ram-contract-fuzz-oracle",
	"artifact-hashes-write":                 "go run ./tools/cmd/validate-artifact-hashes --write",
	"artifact-hashes-validate":              "go run ./tools/cmd/validate-artifact-hashes --manifest",
	"ram-contract-release-validator":        "go run ./tools/cmd/validate-ram-contract-release",
}

var requiredMemory100RAMContractReleaseArtifacts = []memory100RequiredRAMContractReleaseArtifactRef{
	{Path: "ram-contract-report.json", Kind: "ram_contract_report", Schema: "tetra.ram-contract-report.v1"},
	{Path: "memory-grade-report.json", Kind: "memory_grade_report", Schema: "tetra.memory-grade-report.v1"},
	{Path: "proof-store-summary.json", Kind: "proof_store_summary", Schema: "tetra.proof-store-summary.v1"},
	{Path: "validation-pipeline-coverage.json", Kind: "validation_pipeline_coverage", Schema: "tetra.validation-pipeline-coverage.v1"},
	{Path: "heap-blockers.json", Kind: "heap_blockers", Schema: "tetra.ram-blockers.v1"},
	{Path: "copy-blockers.json", Kind: "copy_blockers", Schema: "tetra.ram-blockers.v1"},
	{Path: "fuzz/ram-contract-fuzz-oracle.json", Kind: "ram_contract_fuzz_oracle", Schema: "tetra.ram-contract-fuzz-oracle.v1"},
	{Path: "artifact-hashes.json", Kind: "artifact_hash_manifest", Schema: memory100HashSchema},
}

var requiredMemory100IntegratedCommands = map[string]string{
	"memory-production-gate":               "memory-production-linux-x64-smoke.sh",
	"islands-debug-smoke":                  "go run ./cli/cmd/tetra smoke --target linux-x64 --run=true --islands-debug",
	"validate-islands-debug-smoke":         "go run ./tools/cmd/smoke-report-to-checklist --validate-only",
	"surface-release-gate":                 "scripts/release/surface/release-gate.sh",
	"surface-experimental-regression-gate": "scripts/release/surface/gate.sh",
	"safe-view-lifetime-gate":              "scripts/release/safe-view-lifetime/gate.sh",
	"surface-api-stability-gate":           "scripts/release/surface/api-stability-gate.sh",
	"validate-manifest":                    "go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json",
	"verify-docs":                          "go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
	"artifact-hashes-write":                "go run ./tools/cmd/validate-artifact-hashes --write",
	"artifact-hashes-validate":             "go run ./tools/cmd/validate-artifact-hashes --manifest",
	"integrated-release-validator":         "go run ./tools/cmd/validate-memory-islands-surface-production --report-dir",
}

var requiredMemory100IntegratedArtifacts = []memory100RequiredIntegratedArtifactRef{
	{Path: "memory/memory-production-linux-x64.json", Kind: "memory_production_report", Schema: "tetra.memory.production.v1"},
	{Path: "memory/memory-release-manifest.json", Kind: "memory_release_manifest", Schema: "tetra.memory.release-manifest.v1"},
	{Path: "memory/artifact-hashes.json", Kind: "memory_hash_manifest", Schema: memory100HashSchema},
	{Path: "memory/island-proof-verifier.json", Kind: "island_proof_verifier_report", Schema: "tetra.island.proof.v1"},
	{Path: "memory/island-proof-memory-report.json", Kind: "island_proof_memory_report", Schema: "tetra.memory-report.v1"},
	{Path: "memory/memory-fuzz-tier1/island-proof-fuzz-summary.json", Kind: "island_proof_fuzz_summary", Schema: "tetra.island-proof-fuzz-summary.v1"},
	{Path: "memory/ram-contract/ram-contract-release-manifest.json", Kind: "ram_contract_release_manifest", Schema: "tetra.ram-contract.release-manifest.v1"},
	{Path: "memory/ram-contract/ram-contract-report.json", Kind: "ram_contract_report", Schema: "tetra.ram-contract-report.v1"},
	{Path: "memory/ram-contract/memory-grade-report.json", Kind: "ram_memory_grade_report", Schema: "tetra.memory-grade-report.v1"},
	{Path: "memory/ram-contract/proof-store-summary.json", Kind: "ram_proof_store_summary", Schema: "tetra.proof-store-summary.v1"},
	{Path: "memory/ram-contract/validation-pipeline-coverage.json", Kind: "ram_validation_pipeline_coverage", Schema: "tetra.validation-pipeline-coverage.v1"},
	{Path: "memory/ram-contract/heap-blockers.json", Kind: "ram_heap_blockers", Schema: "tetra.ram-blockers.v1"},
	{Path: "memory/ram-contract/copy-blockers.json", Kind: "ram_copy_blockers", Schema: "tetra.ram-blockers.v1"},
	{Path: "memory/ram-contract/fuzz/ram-contract-fuzz-oracle.json", Kind: "ram_contract_fuzz_oracle", Schema: "tetra.ram-contract-fuzz-oracle.v1"},
	{Path: "memory/ram-contract/artifact-hashes.json", Kind: "ram_contract_hash_manifest", Schema: memory100HashSchema},
	{Path: "islands-debug-smoke.json", Kind: "islands_debug_smoke_report", Schema: "tetra.release.v0_2_0.smoke-report.v1"},
	{Path: "surface-release-v1/surface-release-summary.json", Kind: "surface_release_summary", Schema: "tetra.surface.release.v1"},
	{Path: "surface-release-v1/artifact-hashes.json", Kind: "surface_release_hash_manifest", Schema: memory100HashSchema},
	{Path: "surface-experimental-regression/artifact-hashes.json", Kind: "surface_experimental_hash_manifest", Schema: memory100HashSchema},
	{Path: "safe-view-lifetime/safe-view-lifetime-summary.json", Kind: "safe_view_lifetime_summary", Schema: "tetra.safe-view-lifetime.gate.v1"},
	{Path: "surface-api-stability-v1/surface-api-stability-summary.json", Kind: "surface_api_stability_summary", Schema: "tetra.surface.api-stability.v1"},
	{Path: "artifact-hashes.json", Kind: "integrated_hash_manifest", Schema: memory100HashSchema},
}

var requiredMemory100IntegratedHashPaths = []string{
	"memory-islands-surface-production-manifest.json",
	"memory/memory-production-linux-x64.json",
	"memory/memory-release-manifest.json",
	"memory/artifact-hashes.json",
	"memory/island-proof-verifier.json",
	"memory/island-proof-memory-report.json",
	"memory/memory-fuzz-tier1/island-proof-fuzz-summary.json",
	"memory/ram-contract/artifact-hashes.json",
	"memory/ram-contract/copy-blockers.json",
	"memory/ram-contract/fuzz/ram-contract-fuzz-oracle.json",
	"memory/ram-contract/heap-blockers.json",
	"memory/ram-contract/memory-grade-report.json",
	"memory/ram-contract/proof-store-summary.json",
	"memory/ram-contract/ram-contract-release-manifest.json",
	"memory/ram-contract/ram-contract-report.json",
	"memory/ram-contract/validation-pipeline-coverage.json",
	"islands-debug-smoke.json",
	"surface-release-v1/surface-release-summary.json",
	"surface-release-v1/artifact-hashes.json",
	"surface-experimental-regression/artifact-hashes.json",
	"safe-view-lifetime/safe-view-lifetime-summary.json",
	"surface-api-stability-v1/surface-api-stability-summary.json",
}

func main() {
	reportDir := flag.String("report-dir", "", "Memory100 aggregate report directory")
	currentGitHead := flag.String("current-git-head", "", "optional current git HEAD to require")
	flag.Parse()
	if strings.TrimSpace(*reportDir) == "" {
		fmt.Fprintln(os.Stderr, "error: --report-dir is required")
		os.Exit(2)
	}
	if err := validateMemory100ReportDir(*reportDir, *currentGitHead); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateMemory100ReportDir(reportDir string, currentGitHead string) error {
	reportDir = filepath.Clean(reportDir)
	info, err := os.Lstat(reportDir)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("Memory100 report dir must not be a symlink: %s", reportDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("Memory100 report dir is not a directory: %s", reportDir)
	}

	manifestPath := filepath.Join(reportDir, "memory-100-prod-stable-manifest.json")
	var manifest memory100Manifest
	if err := readMemory100StrictJSON(manifestPath, &manifest); err != nil {
		return fmt.Errorf("Memory100 manifest: %w", err)
	}

	var issues []string
	issues = append(issues, validateMemory100ManifestEnvelope(manifest, currentGitHead)...)
	issues = append(issues, validateMemory100GeneratedAtFreshness(reportDir, manifest.GeneratedAt)...)
	issues = append(issues, validateMemory100Commands(manifest.Commands, reportDir, manifest.GitHead)...)
	issues = append(issues, validateMemory100Claims("claims", manifest.Claims, false)...)
	issues = append(issues, validateMemory100Claims("non_claims", manifest.NonClaims, true)...)
	issues = append(issues, validateMemory100Artifacts(reportDir, manifest.Artifacts, manifest.GitHead)...)
	issues = append(issues, validateMemory100RuntimeMemoryContractTargetMatrix(filepath.Join(reportDir, "runtime-memory", "runtime-memory-contract.json"), manifest.GitHead, manifest.TargetMatrix)...)
	issues = append(issues, validateMemory100RAMContractBundle(reportDir, manifest.GitHead)...)
	issues = append(issues, validateMemory100AllocationLoweringRAMConsistency(reportDir)...)
	issues = append(issues, validateMemory100HashManifest(filepath.Join(reportDir, manifest.HashManifest), reportDir)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateMemory100ManifestEnvelope(manifest memory100Manifest, currentGitHead string) []string {
	var issues []string
	if manifest.Schema != memory100ManifestSchema {
		issues = append(issues, fmt.Sprintf("Memory100 manifest schema is %q, want %s", manifest.Schema, memory100ManifestSchema))
	}
	if manifest.Status != "pass" {
		issues = append(issues, fmt.Sprintf("Memory100 manifest status is %q, want pass", manifest.Status))
	}
	if strings.TrimSpace(manifest.Verdict) == "" {
		issues = append(issues, "Memory100 manifest verdict is required")
	}
	if !isMemory100GitHead(manifest.GitHead) {
		issues = append(issues, "Memory100 manifest git_head must be a 40-character lowercase hex commit")
	}
	currentGitHead = strings.TrimSpace(currentGitHead)
	if currentGitHead != "" && manifest.GitHead != currentGitHead {
		issues = append(issues, fmt.Sprintf("Memory100 manifest git_head %s does not match current git head %s", manifest.GitHead, currentGitHead))
	}
	if _, err := time.Parse(time.RFC3339, manifest.GeneratedAt); err != nil {
		issues = append(issues, fmt.Sprintf("Memory100 manifest generated_at must be RFC3339: %v", err))
	}
	if manifest.GitDirty == nil {
		issues = append(issues, "Memory100 manifest git_dirty is required")
	} else {
		statusLines := nonEmptyMemory100Strings(manifest.GitStatus)
		if len(statusLines) == 0 {
			issues = append(issues, "Memory100 manifest git_status_short_branch must not be empty")
		} else {
			statusDirty := memory100GitStatusSnapshotDirty(statusLines)
			if *manifest.GitDirty != statusDirty {
				issues = append(issues, fmt.Sprintf("Memory100 manifest git_dirty is %v but git_status_short_branch dirty state is %v", *manifest.GitDirty, statusDirty))
			}
			if *manifest.GitDirty && memory100VerdictClaimsClean(manifest.Verdict) {
				issues = append(issues, fmt.Sprintf("Memory100 manifest verdict %q claims clean/release-candidate status on a dirty checkout", manifest.Verdict))
			}
			issues = append(issues, validateMemory100VerdictDirtyTier(manifest.Verdict, *manifest.GitDirty)...)
		}
	}
	if len(manifest.TargetMatrix) == 0 {
		issues = append(issues, "Memory100 manifest target_matrix must not be empty")
	}
	if manifest.HashManifest != "artifact-hashes.json" {
		issues = append(issues, fmt.Sprintf("Memory100 manifest hash_manifest is %q, want artifact-hashes.json", manifest.HashManifest))
	}
	if len(manifest.Claims) == 0 {
		issues = append(issues, "Memory100 manifest claims must not be empty")
	}
	if len(manifest.NonClaims) == 0 {
		issues = append(issues, "Memory100 manifest non_claims must not be empty")
	}
	return issues
}

func validateMemory100GeneratedAtFreshness(reportDir string, aggregateGeneratedAt string) []string {
	return validateMemory100GeneratedAtFreshnessWithin(reportDir, "memory-100-prod-stable-manifest.json", aggregateGeneratedAt, "Memory100 manifest")
}

func validateMemory100GeneratedAtFreshnessWithin(rootDir string, parentRel string, parentGeneratedAt string, label string) []string {
	parentAt, err := time.Parse(time.RFC3339, parentGeneratedAt)
	if err != nil {
		return nil
	}
	parentRel = filepath.ToSlash(filepath.Clean(filepath.FromSlash(parentRel)))
	var issues []string
	err = filepath.WalkDir(rootDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			issues = append(issues, fmt.Sprintf("%s generated_at freshness walk %s: %v", label, filepath.ToSlash(path), walkErr))
			return nil
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s generated_at freshness read %s: %v", label, filepath.ToSlash(path), err))
			return nil
		}
		var obj map[string]any
		if err := json.Unmarshal(raw, &obj); err != nil {
			return nil
		}
		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			rel = path
		}
		rel = filepath.ToSlash(filepath.Clean(rel))
		if rel == parentRel {
			return nil
		}
		for _, key := range []string{"generated_at", "generated_at_utc"} {
			value, ok := obj[key].(string)
			if !ok || strings.TrimSpace(value) == "" {
				continue
			}
			childAt, err := time.Parse(time.RFC3339, value)
			if err != nil {
				issues = append(issues, fmt.Sprintf("%s child evidence %s %s must be RFC3339: %v", label, filepath.ToSlash(rel), key, err))
				continue
			}
			if childAt.After(parentAt) {
				issues = append(issues, fmt.Sprintf("%s generated_at %s is older than child evidence %s %s %s", label, parentGeneratedAt, filepath.ToSlash(rel), key, value))
			}
		}
		return nil
	})
	if err != nil {
		issues = append(issues, fmt.Sprintf("%s generated_at freshness walk failed: %v", label, err))
	}
	return issues
}

func validateMemory100Commands(commands []memory100Command, reportDir string, gitHead string) []string {
	seen := map[string]string{}
	var issues []string
	for _, command := range commands {
		name := strings.TrimSpace(command.Name)
		text := strings.TrimSpace(command.Command)
		if name == "" {
			issues = append(issues, "Memory100 command name is required")
			continue
		}
		if _, ok := seen[name]; ok {
			issues = append(issues, fmt.Sprintf("duplicate Memory100 command %s", name))
		}
		seen[name] = text
		if text == "" {
			issues = append(issues, fmt.Sprintf("Memory100 command %s command is required", name))
			continue
		}
		if strings.Contains(text, "|| true") || strings.Contains(text, "continue-on-error") || strings.Contains(text, "set +e") {
			issues = append(issues, fmt.Sprintf("Memory100 command %s contains bypass marker", name))
		}
	}
	for name, fragment := range requiredMemory100Commands {
		text, ok := seen[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing Memory100 command %s containing %q", name, fragment))
			continue
		}
		if !strings.Contains(text, fragment) {
			issues = append(issues, fmt.Sprintf("Memory100 command %s must contain %q", name, fragment))
		}
	}
	issues = append(issues, validateMemory100CommandProvenance(seen, reportDir, gitHead)...)
	return issues
}

func validateMemory100CommandProvenance(commands map[string]string, reportDir string, gitHead string) []string {
	type pathRequirement struct {
		name string
		flag string
		rel  string
	}
	pathRequirements := []pathRequirement{
		{name: "memory-production-gate", flag: "--report-dir", rel: "memory-production"},
		{name: "ram-contract-gate", flag: "--report-dir", rel: "ram-contract"},
		{name: "integrated-gate", flag: "--report-dir", rel: "integrated"},
		{name: "memory-fuzz-short", flag: "--report-dir", rel: "memory-fuzz"},
		{name: "memory-fuzz-validator", flag: "--report", rel: "memory-fuzz/memory-fuzz-oracle.json"},
		{name: "memory-fuzz-validator", flag: "--artifact-dir", rel: "memory-fuzz"},
		{name: "artifact-hashes-write", flag: "--root", rel: ""},
		{name: "artifact-hashes-write", flag: "--out", rel: "artifact-hashes.json"},
		{name: "memory-100-validator", flag: "--report-dir", rel: ""},
	}
	var issues []string
	for _, requirement := range pathRequirements {
		text := strings.TrimSpace(commands[requirement.name])
		if text == "" {
			continue
		}
		wantPath := reportDir
		if requirement.rel != "" {
			wantPath = filepath.Join(reportDir, filepath.FromSlash(requirement.rel))
		}
		if !memory100CommandContainsAnyPath(text, requirement.flag, memory100EquivalentPathForms(wantPath)) {
			issues = append(issues, fmt.Sprintf("Memory100 command %s must use %s under the current report dir for %s", requirement.name, requirement.flag, requirement.rel))
		}
	}
	for _, requirement := range []struct {
		name string
		flag string
	}{
		{name: "memory-fuzz-short", flag: "--git-head"},
		{name: "memory-fuzz-validator", flag: "--current-git-head"},
		{name: "memory-100-validator", flag: "--current-git-head"},
	} {
		text := strings.TrimSpace(commands[requirement.name])
		if text == "" || strings.TrimSpace(gitHead) == "" {
			continue
		}
		if !strings.Contains(text, requirement.flag+" "+gitHead) {
			issues = append(issues, fmt.Sprintf("Memory100 command %s must use %s %s", requirement.name, requirement.flag, gitHead))
		}
	}
	return issues
}

func validateMemory100Artifacts(reportDir string, artifacts []memory100Artifact, gitHead string) []string {
	byKind := map[string]memory100Artifact{}
	seenPath := map[string]bool{}
	requiredKinds := map[string]bool{}
	for _, required := range requiredMemory100Artifacts {
		requiredKinds[required.Kind] = true
	}
	var issues []string
	for _, artifact := range artifacts {
		if err := validateMemory100SafeRel(artifact.Path); err != nil {
			issues = append(issues, fmt.Sprintf("Memory100 artifact path %q is invalid: %v", artifact.Path, err))
			continue
		}
		if strings.TrimSpace(artifact.Kind) == "" {
			issues = append(issues, fmt.Sprintf("Memory100 artifact %s kind is required", artifact.Path))
			continue
		}
		if !requiredKinds[artifact.Kind] {
			issues = append(issues, fmt.Sprintf("unexpected Memory100 artifact kind %s at %s", artifact.Kind, artifact.Path))
		}
		if _, ok := byKind[artifact.Kind]; ok {
			issues = append(issues, fmt.Sprintf("duplicate Memory100 artifact kind %s", artifact.Kind))
		}
		byKind[artifact.Kind] = artifact
		if seenPath[artifact.Path] {
			issues = append(issues, fmt.Sprintf("duplicate Memory100 artifact path %s", artifact.Path))
		}
		seenPath[artifact.Path] = true
	}
	for _, required := range requiredMemory100Artifacts {
		artifact, ok := byKind[required.Kind]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing Memory100 artifact %s", required.Kind))
			continue
		}
		if artifact.Path != required.Path {
			issues = append(issues, fmt.Sprintf("Memory100 artifact %s path is %q, want %s", required.Kind, artifact.Path, required.Path))
		}
		if artifact.Schema != required.Schema {
			issues = append(issues, fmt.Sprintf("Memory100 artifact %s schema is %q, want %s", required.Kind, artifact.Schema, required.Schema))
		}
		issues = append(issues, validateMemory100ArtifactFile(reportDir, required, gitHead)...)
	}
	return issues
}

func validateMemory100ArtifactFile(reportDir string, required memory100RequiredArtifact, gitHead string) []string {
	path := filepath.Join(reportDir, filepath.FromSlash(required.Path))
	info, err := os.Lstat(path)
	if err != nil {
		return []string{fmt.Sprintf("%s artifact %s is missing: %v", required.Kind, required.Path, err)}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return []string{fmt.Sprintf("%s artifact %s must not be a symlink", required.Kind, required.Path)}
	}
	if !info.Mode().IsRegular() {
		return []string{fmt.Sprintf("%s artifact %s is not a regular file", required.Kind, required.Path)}
	}
	if info.Size() == 0 {
		return []string{fmt.Sprintf("%s artifact %s is empty", required.Kind, required.Path)}
	}
	var envelope memory100SchemaEnvelope
	if err := readMemory100JSON(path, &envelope); err != nil {
		return []string{fmt.Sprintf("%s artifact %s is invalid JSON: %v", required.Kind, required.Path, err)}
	}
	var issues []string
	if memory100SchemaOf(envelope) != required.Schema {
		issues = append(issues, fmt.Sprintf("%s artifact schema is %q, want %s", required.Kind, memory100SchemaOf(envelope), required.Schema))
	}
	if required.Schema != memory100HashSchema {
		if !isMemory100GitHead(envelope.GitHead) {
			issues = append(issues, fmt.Sprintf("%s artifact git_head must be a 40-character lowercase hex commit", required.Kind))
		} else if gitHead != "" && envelope.GitHead != gitHead {
			issues = append(issues, fmt.Sprintf("%s artifact git_head %s does not match Memory100 git_head %s", required.Kind, envelope.GitHead, gitHead))
		}
	}
	issues = append(issues, validateMemory100ArtifactContent(path, required.Kind, gitHead)...)
	return issues
}

func validateMemory100ArtifactContent(path string, kind string, gitHead string) []string {
	switch kind {
	case "memory_production_report":
		return validateMemory100MemoryProductionReport(path)
	case "memory_release_manifest":
		return validateMemory100MemoryReleaseManifest(path, gitHead)
	case "ram_contract_release_manifest":
		return validateMemory100RAMContractReleaseManifest(path, gitHead)
	case "memory_production_hash_manifest":
		return validateMemory100NestedHashManifestWithRequired(path, filepath.Dir(path), "memory production artifact-hashes.json", memory100MemoryReleaseRequiredHashPaths())
	case "ram_contract_hash_manifest":
		return validateMemory100NestedHashManifest(path, filepath.Dir(path), "ram contract artifact-hashes.json")
	case "ram_contract_fuzz_oracle":
		return validateMemory100RAMContractFuzzOracle(path, gitHead)
	case "raw_memory_contract_report":
		return validateMemory100RawMemoryContract(path, gitHead)
	case "allocation_lowering_report":
		return validateMemory100AllocationLowering(path, gitHead)
	case "proof_store_summary":
		return validateMemory100ProofStoreSummary(path)
	case "memory_fuzz_oracle_report":
		return validateMemory100MemoryFuzzBundle(filepath.Dir(path), gitHead)
	case "proof_transition_report":
		return validateMemory100ProofTransitionReport(path, gitHead)
	case "runtime_memory_contract":
		return validateMemory100RuntimeMemoryContract(path, gitHead, nil)
	case "memory_semantic_safety_matrix":
		return validateMemory100SemanticSafetyMatrix(path, gitHead)
	case "leak_resource_report":
		return validateMemory100LeakResource(path, gitHead)
	case "integrated_memory_islands_surface_manifest":
		return validateMemory100IntegratedManifest(path, gitHead)
	case "integrated_hash_manifest":
		issues := validateMemory100NestedHashManifestWithRequired(path, filepath.Dir(path), "integrated artifact-hashes.json", requiredMemory100IntegratedHashPaths)
		issues = append(issues, validateMemory100IntegratedNestedMemory(filepath.Dir(path), gitHead)...)
		issues = append(issues, validateMemory100IntegratedNestedRAMContract(filepath.Dir(path), gitHead)...)
		issues = append(issues, validateMemory100IntegratedIslandsDebugSmoke(filepath.Join(filepath.Dir(path), "islands-debug-smoke.json"), gitHead)...)
		issues = append(issues, validateMemory100IntegratedSurfaceEvidence(filepath.Dir(path), gitHead)...)
		return issues
	case "docs_claim_policy":
		return validateMemory100ClaimPolicyArtifact(path, gitHead)
	default:
		return nil
	}
}

func validateMemory100RAMContractFuzzOracle(path string, gitHead string) []string {
	var report struct {
		SchemaVersion string `json:"schema_version"`
		GitHead       string `json:"git_head"`
		GeneratedAt   string `json:"generated_at"`
		Observations  []struct {
			Mutation         string `json:"mutation"`
			Rejected         bool   `json:"rejected"`
			Validator        string `json:"validator"`
			ValidatorCommand string `json:"validator_command"`
			ExitCode         *int   `json:"exit_code"`
			OutputExcerpt    string `json:"output_excerpt"`
			MutatedFile      string `json:"mutated_file"`
			Reason           string `json:"reason"`
		} `json:"observations"`
		Summary struct {
			Mutations int `json:"mutations"`
			Rejected  int `json:"rejected"`
		} `json:"summary"`
		NonClaims []string `json:"non_claims"`
	}
	if err := readMemory100StrictJSON(path, &report); err != nil {
		return []string{fmt.Sprintf("RAM contract fuzz oracle invalid: %v", err)}
	}
	var issues []string
	if report.SchemaVersion != "tetra.ram-contract-fuzz-oracle.v1" {
		issues = append(issues, fmt.Sprintf("RAM contract fuzz oracle schema_version is %q, want tetra.ram-contract-fuzz-oracle.v1", report.SchemaVersion))
	}
	if gitHead != "" && report.GitHead != gitHead {
		issues = append(issues, fmt.Sprintf("RAM contract fuzz oracle git_head %s does not match Memory100 git_head %s", report.GitHead, gitHead))
	}
	if _, err := time.Parse(time.RFC3339, report.GeneratedAt); err != nil {
		issues = append(issues, fmt.Sprintf("RAM contract fuzz oracle generated_at must be RFC3339: %v", err))
	}
	if len(nonEmptyMemory100Strings(report.NonClaims)) == 0 {
		issues = append(issues, "RAM contract fuzz oracle non_claims must not be empty")
	}
	issues = append(issues, validateMemory100Claims("RAM contract fuzz oracle non_claims", report.NonClaims, true)...)

	required := map[string]bool{
		"mutated_proof_id":        false,
		"widened_grade":           false,
		"missing_blocker":         false,
		"budget_drift":            false,
		"artifact_hash_drift":     false,
		"forbidden_nonclaim_text": false,
	}
	rejected := 0
	for i, obs := range report.Observations {
		mutation := strings.TrimSpace(obs.Mutation)
		if mutation == "" {
			issues = append(issues, fmt.Sprintf("RAM contract fuzz observation %d missing mutation", i))
			continue
		}
		if _, ok := required[mutation]; ok {
			required[mutation] = true
		}
		if !obs.Rejected {
			issues = append(issues, fmt.Sprintf("RAM contract fuzz mutation %s was not rejected", mutation))
		} else {
			rejected++
		}
		if strings.TrimSpace(obs.Validator) == "" || strings.TrimSpace(obs.ValidatorCommand) == "" || strings.TrimSpace(obs.Reason) == "" {
			issues = append(issues, fmt.Sprintf("RAM contract fuzz mutation %s missing validator command/reason evidence", mutation))
		}
		if obs.ExitCode == nil || *obs.ExitCode == 0 {
			issues = append(issues, fmt.Sprintf("RAM contract fuzz mutation %s missing nonzero exit evidence", mutation))
		}
		if strings.TrimSpace(obs.OutputExcerpt) == "" || strings.TrimSpace(obs.MutatedFile) == "" {
			issues = append(issues, fmt.Sprintf("RAM contract fuzz mutation %s missing output or mutated_file evidence", mutation))
		}
	}
	for mutation, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("RAM contract fuzz oracle missing mutation class %s", mutation))
		}
	}
	if report.Summary.Mutations != len(report.Observations) || report.Summary.Rejected != rejected {
		issues = append(issues, fmt.Sprintf("RAM contract fuzz oracle summary mismatch: mutations=%d rejected=%d observations=%d counted_rejected=%d", report.Summary.Mutations, report.Summary.Rejected, len(report.Observations), rejected))
	}
	return issues
}

func validateMemory100ProofStoreSummary(path string) []string {
	if err := ramvalidate.ValidateProofStoreSummaryFile(path); err != nil {
		return []string{fmt.Sprintf("proof store summary: %v", err)}
	}
	return nil
}

func validateMemory100MemoryProductionReport(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("memory production report unreadable: %v", err)}
	}
	if err := memoryprod.ValidateReport(raw); err != nil {
		return []string{fmt.Sprintf("memory production report: %v", err)}
	}
	return nil
}

func validateMemory100MemoryReleaseManifest(path string, gitHead string) []string {
	var manifest memory100MemoryReleaseManifest
	if err := readMemory100StrictJSON(path, &manifest); err != nil {
		return []string{fmt.Sprintf("memory release manifest invalid: %v", err)}
	}
	var issues []string
	if manifest.Schema != "tetra.memory.release-manifest.v1" {
		issues = append(issues, fmt.Sprintf("memory release manifest schema is %q, want tetra.memory.release-manifest.v1", manifest.Schema))
	}
	if manifest.Target != "linux-x64" {
		issues = append(issues, fmt.Sprintf("memory release manifest target is %q, want linux-x64", manifest.Target))
	}
	if gitHead != "" && manifest.GitHead != gitHead {
		issues = append(issues, fmt.Sprintf("memory release manifest git_head %s does not match Memory100 git_head %s", manifest.GitHead, gitHead))
	}
	if _, err := time.Parse(time.RFC3339, manifest.GeneratedAt); err != nil {
		issues = append(issues, fmt.Sprintf("memory release manifest generated_at must be RFC3339: %v", err))
	}
	if manifest.ReportDir != "." {
		issues = append(issues, fmt.Sprintf("memory release manifest report_dir is %q, want .", manifest.ReportDir))
	}
	if manifest.HashManifest != "artifact-hashes.json" {
		issues = append(issues, fmt.Sprintf("memory release manifest hash_manifest is %q, want artifact-hashes.json", manifest.HashManifest))
	}
	memoryDir := filepath.Dir(path)
	issues = append(issues, validateMemory100GeneratedAtFreshnessWithin(memoryDir, "memory-release-manifest.json", manifest.GeneratedAt, "memory release manifest")...)
	issues = append(issues, validateMemory100MemoryReleaseCommands(manifest.Commands, memoryDir, gitHead)...)
	issues = append(issues, validateMemory100MemoryReleaseArtifactRefs(manifest.Artifacts, memoryDir, gitHead)...)
	issues = append(issues, validateMemory100MemoryReleaseNestedEvidence(memoryDir, gitHead)...)
	return issues
}

func validateMemory100MemoryReleaseNestedEvidence(memoryDir string, gitHead string) []string {
	var issues []string
	issues = append(issues, validateMemory100TargetsReport(filepath.Join(memoryDir, "targets.json"), "memory release targets.json")...)
	for _, issue := range validateMemory100MemoryFuzzBundle(filepath.Join(memoryDir, "memory-fuzz-tier1"), gitHead) {
		issues = append(issues, "memory release "+issue)
	}
	issues = append(issues, validateMemory100IslandProofEvidence(memoryDir, gitHead, "memory release island proof")...)
	issues = append(issues, validateMemory100NestedRAMContract(filepath.Join(memoryDir, "ram-contract"), gitHead, "memory release")...)
	return issues
}

func validateMemory100TargetsReport(path string, label string) []string {
	var report memory100TargetsReport
	if err := readMemory100StrictJSON(path, &report); err != nil {
		return []string{fmt.Sprintf("%s missing or invalid: %v", label, err)}
	}
	var issues []string
	issues = append(issues, validateMemory100StringSequence(label+" supported", report.Supported, []string{"linux-x64", "windows-x64", "macos-x64", "wasm32-wasi", "wasm32-web"})...)
	issues = append(issues, validateMemory100StringSequence(label+" build_only", report.BuildOnly, []string{"linux-x86", "linux-x32"})...)
	issues = append(issues, validateMemory100StringSequence(label+" planned", report.Planned, nil)...)

	expected := []struct {
		triple                  string
		status                  string
		os                      string
		arch                    string
		abi                     string
		format                  string
		exeExt                  string
		buildOnly               bool
		runMode                 string
		supportsDebugInfo       bool
		supportsReleaseOptimize bool
	}{
		{triple: "linux-x64", status: "supported", os: "linux", arch: "x64", abi: "sysv", format: "elf", exeExt: "", buildOnly: false, runMode: "host_native", supportsDebugInfo: true, supportsReleaseOptimize: true},
		{triple: "windows-x64", status: "supported", os: "windows", arch: "x64", abi: "win64", format: "pe", exeExt: ".exe", buildOnly: false, runMode: "host_native", supportsDebugInfo: true, supportsReleaseOptimize: true},
		{triple: "macos-x64", status: "supported", os: "macos", arch: "x64", abi: "sysv", format: "macho", exeExt: "", buildOnly: false, runMode: "host_native", supportsDebugInfo: true, supportsReleaseOptimize: true},
		{triple: "wasm32-wasi", status: "supported", os: "wasi", arch: "wasm32", abi: "wasi", format: "wasm", exeExt: ".wasm", buildOnly: false, runMode: "wasi_runner", supportsDebugInfo: false, supportsReleaseOptimize: true},
		{triple: "wasm32-web", status: "supported", os: "web", arch: "wasm32", abi: "web", format: "wasm", exeExt: ".wasm", buildOnly: false, runMode: "web_runner", supportsDebugInfo: false, supportsReleaseOptimize: true},
		{triple: "linux-x86", status: "build_only", os: "linux", arch: "x86", abi: "i386-sysv", format: "elf", exeExt: "", buildOnly: true, runMode: "host_probed", supportsDebugInfo: false, supportsReleaseOptimize: false},
		{triple: "linux-x32", status: "build_only", os: "linux", arch: "x64", abi: "x32-sysv", format: "elf", exeExt: "", buildOnly: true, runMode: "host_probed", supportsDebugInfo: false, supportsReleaseOptimize: false},
	}
	if len(report.Targets) != len(expected) {
		issues = append(issues, fmt.Sprintf("%s target metadata count = %d, want %d", label, len(report.Targets), len(expected)))
	}
	seen := map[string]bool{}
	for i, want := range expected {
		if i >= len(report.Targets) {
			break
		}
		row := report.Targets[i]
		if seen[row.Triple] {
			issues = append(issues, fmt.Sprintf("%s target metadata %q is duplicated", label, row.Triple))
		}
		seen[row.Triple] = true
		if row.Triple != want.triple {
			issues = append(issues, fmt.Sprintf("%s target metadata[%d].triple = %q, want %q", label, i, row.Triple, want.triple))
			continue
		}
		if row.Status != want.status {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].status = %q, want %q", label, row.Triple, row.Status, want.status))
		}
		if row.OS != want.os || row.Arch != want.arch || row.ABI != want.abi || row.Format != want.format {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s] platform = os:%s arch:%s abi:%s format:%s, want os:%s arch:%s abi:%s format:%s", label, row.Triple, row.OS, row.Arch, row.ABI, row.Format, want.os, want.arch, want.abi, want.format))
		}
		if row.ExeExt != want.exeExt {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].exe_ext = %q, want %q", label, row.Triple, row.ExeExt, want.exeExt))
		}
		if row.BuildOnly != want.buildOnly {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].build_only = %v, want %v", label, row.Triple, row.BuildOnly, want.buildOnly))
		}
		if row.RunMode != want.runMode {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].run_mode = %q, want %q", label, row.Triple, row.RunMode, want.runMode))
		}
		if row.SupportsDebugInfo != want.supportsDebugInfo {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].supports_debug_info = %v, want %v", label, row.Triple, row.SupportsDebugInfo, want.supportsDebugInfo))
		}
		if row.SupportsReleaseOptimize != want.supportsReleaseOptimize {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].supports_release_optimize = %v, want %v", label, row.Triple, row.SupportsReleaseOptimize, want.supportsReleaseOptimize))
		}
		issues = append(issues, validateMemory100TargetMemoryClaims(label, row)...)
	}
	return issues
}

func validateMemory100StringSequence(label string, got []string, want []string) []string {
	if len(got) != len(want) {
		return []string{fmt.Sprintf("%s count = %d, want %d", label, len(got), len(want))}
	}
	var issues []string
	seen := map[string]bool{}
	for i := range want {
		if got[i] != want[i] {
			issues = append(issues, fmt.Sprintf("%s[%d] = %q, want %q", label, i, got[i], want[i]))
		}
		if seen[got[i]] {
			issues = append(issues, fmt.Sprintf("%s %q is duplicated", label, got[i]))
		}
		seen[got[i]] = true
	}
	return issues
}

func validateMemory100TargetMemoryClaims(label string, row memory100TargetsReportRow) []string {
	var issues []string
	if row.MemoryBuild != "yes" {
		issues = append(issues, fmt.Sprintf("%s target metadata[%s].memory_build = %q, want yes", label, row.Triple, row.MemoryBuild))
	}
	if row.MemoryLower != "yes" {
		issues = append(issues, fmt.Sprintf("%s target metadata[%s].memory_lower = %q, want yes", label, row.Triple, row.MemoryLower))
	}
	switch row.Triple {
	case "linux-x64":
		if row.RuntimeStatus != "production" {
			issues = append(issues, fmt.Sprintf("%s target metadata[linux-x64].runtime_status = %q, want production", label, row.RuntimeStatus))
		}
		if row.StdlibStatus != "production" {
			issues = append(issues, fmt.Sprintf("%s target metadata[linux-x64].stdlib_status = %q, want production", label, row.StdlibStatus))
		}
		if row.MemoryRun != "yes" {
			issues = append(issues, fmt.Sprintf("%s target metadata[linux-x64].memory_run = %q, want yes", label, row.MemoryRun))
		}
		if row.MemoryRawDiagnostics != "yes" {
			issues = append(issues, fmt.Sprintf("%s target metadata[linux-x64].memory_raw_diagnostics = %q, want yes", label, row.MemoryRawDiagnostics))
		}
		if row.MemoryRegionLowering != "yes/partial" {
			issues = append(issues, fmt.Sprintf("%s target metadata[linux-x64].memory_region_lowering = %q, want yes/partial", label, row.MemoryRegionLowering))
		}
		if row.MemoryAlignmentSemantics != "yes" {
			issues = append(issues, fmt.Sprintf("%s target metadata[linux-x64].memory_alignment_semantics = %q, want yes", label, row.MemoryAlignmentSemantics))
		}
		if row.MemoryClaimLevel != "production/host_runtime" {
			issues = append(issues, fmt.Sprintf("%s target metadata[linux-x64].memory_claim_level = %q, want production/host_runtime", label, row.MemoryClaimLevel))
		}
		for _, required := range []string{"targets.json", "linux-x64-runner.json", "linux-x64-abi.json"} {
			if !memory100StringSliceHas(row.EvidenceArtifacts, required) {
				issues = append(issues, fmt.Sprintf("%s target metadata[linux-x64].evidence_artifacts missing %s", label, required))
			}
		}
	case "linux-x86", "linux-x32":
		if row.RuntimeStatus != "partial_build_only" {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].runtime_status = %q, want partial_build_only", label, row.Triple, row.RuntimeStatus))
		}
		if row.StdlibStatus != "partial_build_only" {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].stdlib_status = %q, want partial_build_only", label, row.Triple, row.StdlibStatus))
		}
		if row.MemoryRun != "no/host-dependent" {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].memory_run = %q, want no/host-dependent", label, row.Triple, row.MemoryRun))
		}
		if row.MemoryClaimLevel != "build_lower_only" {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].memory_claim_level = %q, want build_lower_only", label, row.Triple, row.MemoryClaimLevel))
		}
		if row.RunnerProbeCommand == "" {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].runner_probe_command is required", label, row.Triple))
		}
		if row.ReleaseGate == "" {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].release_gate is required", label, row.Triple))
		}
		for _, required := range []string{"targets.json", row.Triple + "-runner.json", row.Triple + "-abi.json"} {
			if !memory100StringSliceHas(row.EvidenceArtifacts, required) {
				issues = append(issues, fmt.Sprintf("%s target metadata[%s].evidence_artifacts missing %s", label, row.Triple, required))
			}
		}
	default:
		if row.MemoryRun == "yes" {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].memory_run must not claim yes without target-host Memory100 evidence", label, row.Triple))
		}
		if row.MemoryClaimLevel == "production/host_runtime" {
			issues = append(issues, fmt.Sprintf("%s target metadata[%s].memory_claim_level must not claim production/host_runtime without target-host Memory100 evidence", label, row.Triple))
		}
	}
	return issues
}

func memory100StringSliceHas(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func validateMemory100MemoryReleaseCommands(commands []memory100Command, memoryDir string, gitHead string) []string {
	seen := map[string]string{}
	var issues []string
	for _, command := range commands {
		name := strings.TrimSpace(command.Name)
		text := strings.TrimSpace(command.Command)
		if name == "" {
			issues = append(issues, "memory release manifest command name is required")
			continue
		}
		if _, ok := seen[name]; ok {
			issues = append(issues, fmt.Sprintf("duplicate memory release manifest command %s", name))
		}
		seen[name] = text
		if text == "" {
			issues = append(issues, fmt.Sprintf("memory release manifest command %s command is required", name))
		}
		if strings.Contains(text, "|| true") || strings.Contains(text, "continue-on-error") || strings.Contains(text, "set +e") {
			issues = append(issues, fmt.Sprintf("memory release manifest command %s contains bypass marker", name))
		}
	}
	for name, fragment := range requiredMemory100MemoryReleaseCommands {
		text, ok := seen[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing memory release manifest command %s containing %q", name, fragment))
			continue
		}
		if !strings.Contains(text, fragment) {
			issues = append(issues, fmt.Sprintf("memory release manifest command %s must contain %q", name, fragment))
		}
	}
	issues = append(issues, validateMemory100MemoryReleaseCommandProvenance(seen, memoryDir, gitHead)...)
	return issues
}

func validateMemory100MemoryReleaseCommandProvenance(commands map[string]string, memoryDir string, gitHead string) []string {
	type pathRequirement struct {
		name string
		flag string
		rel  string
	}
	pathRequirements := []pathRequirement{
		{name: "memory-production-smoke", flag: "--report", rel: "memory-production-linux-x64.json"},
		{name: "target-report", flag: ">", rel: "targets.json"},
		{name: "validate-targets", flag: "--report", rel: "targets.json"},
		{name: "memory-fuzz-short", flag: "--report-dir", rel: "memory-fuzz-tier1"},
		{name: "validate-memory-fuzz-oracle", flag: "--report", rel: "memory-fuzz-tier1/memory-fuzz-oracle.json"},
		{name: "validate-memory-fuzz-oracle", flag: "--artifact-dir", rel: "memory-fuzz-tier1"},
		{name: "ram-contract-gate", flag: "--report-dir", rel: "ram-contract"},
		{name: "island-proof-verifier", flag: "--proof", rel: "island-proof-verifier.json"},
		{name: "island-proof-verifier", flag: "--memory-report", rel: "island-proof-memory-report.json"},
		{name: "artifact-hashes-write", flag: "--root", rel: ""},
		{name: "artifact-hashes-write", flag: "--out", rel: "artifact-hashes.json"},
		{name: "artifact-hashes-validate", flag: "--manifest", rel: "artifact-hashes.json"},
	}
	var issues []string
	for _, requirement := range pathRequirements {
		text := strings.TrimSpace(commands[requirement.name])
		if text == "" {
			continue
		}
		wantPath := memoryDir
		if requirement.rel != "" {
			wantPath = filepath.Join(memoryDir, filepath.FromSlash(requirement.rel))
		}
		if !memory100CommandContainsAnyPath(text, requirement.flag, memory100EquivalentPathForms(wantPath)) {
			issues = append(issues, fmt.Sprintf("memory release manifest command %s must use %s under the current memory production report dir for %s", requirement.name, requirement.flag, requirement.rel))
		}
	}
	for _, requirement := range []struct {
		name string
		flag string
	}{
		{name: "memory-production-smoke", flag: "--git-head"},
		{name: "memory-fuzz-short", flag: "--git-head"},
		{name: "validate-memory-fuzz-oracle", flag: "--current-git-head"},
		{name: "island-proof-verifier", flag: "--current-git-head"},
	} {
		text := strings.TrimSpace(commands[requirement.name])
		if text == "" || strings.TrimSpace(gitHead) == "" {
			continue
		}
		if !strings.Contains(text, requirement.flag+" "+gitHead) {
			issues = append(issues, fmt.Sprintf("memory release manifest command %s must use %s %s", requirement.name, requirement.flag, gitHead))
		}
	}
	if text := strings.TrimSpace(commands["island-proof-verifier"]); text != "" && !strings.Contains(text, "--require-same-commit") {
		issues = append(issues, "memory release manifest command island-proof-verifier must require same-commit validation")
	}
	return issues
}

func validateMemory100MemoryReleaseArtifactRefs(artifacts []memory100MemoryReleaseArtifactRef, memoryDir string, gitHead string) []string {
	byKind := map[string]memory100MemoryReleaseArtifactRef{}
	seenPath := map[string]bool{}
	requiredKinds := map[string]bool{}
	for _, required := range requiredMemory100MemoryReleaseArtifacts {
		requiredKinds[required.Kind] = true
	}
	var issues []string
	for _, artifact := range artifacts {
		if err := validateMemory100SafeRel(artifact.Path); err != nil {
			issues = append(issues, fmt.Sprintf("memory release manifest artifact path %q is invalid: %v", artifact.Path, err))
			continue
		}
		if strings.TrimSpace(artifact.Kind) == "" {
			issues = append(issues, fmt.Sprintf("memory release manifest artifact %s kind is required", artifact.Path))
			continue
		}
		if !requiredKinds[artifact.Kind] {
			issues = append(issues, fmt.Sprintf("unexpected memory release manifest artifact kind %s at %s", artifact.Kind, artifact.Path))
		}
		if _, ok := byKind[artifact.Kind]; ok {
			issues = append(issues, fmt.Sprintf("duplicate memory release manifest artifact kind %s", artifact.Kind))
		}
		byKind[artifact.Kind] = artifact
		if seenPath[artifact.Path] {
			issues = append(issues, fmt.Sprintf("duplicate memory release manifest artifact path %s", artifact.Path))
		}
		seenPath[artifact.Path] = true
		if artifact.Target != "linux-x64" {
			issues = append(issues, fmt.Sprintf("memory release manifest artifact %s target is %q, want linux-x64", artifact.Kind, artifact.Target))
		}
		if strings.TrimSpace(artifact.Command) == "" {
			issues = append(issues, fmt.Sprintf("memory release manifest artifact %s command is required", artifact.Kind))
		}
		issues = append(issues, validateMemory100MemoryReleaseArtifactCommand(artifact, memoryDir, gitHead)...)
	}
	for _, required := range requiredMemory100MemoryReleaseArtifacts {
		artifact, ok := byKind[required.Kind]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing memory release manifest artifact %s", required.Kind))
			continue
		}
		if artifact.Path != required.Path {
			issues = append(issues, fmt.Sprintf("memory release manifest artifact %s path is %q, want %s", required.Kind, artifact.Path, required.Path))
		}
		if required.Schema != "" && artifact.Schema != required.Schema {
			issues = append(issues, fmt.Sprintf("memory release manifest artifact %s schema is %q, want %s", required.Kind, artifact.Schema, required.Schema))
		}
		if required.CommandFragment != "" && !strings.Contains(artifact.Command, required.CommandFragment) {
			issues = append(issues, fmt.Sprintf("memory release manifest artifact %s command must contain %q", required.Kind, required.CommandFragment))
		}
	}
	return issues
}

func validateMemory100MemoryReleaseArtifactCommand(artifact memory100MemoryReleaseArtifactRef, memoryDir string, gitHead string) []string {
	type pathRequirement struct {
		flag string
		rel  string
	}
	requirementsByKind := map[string][]pathRequirement{
		"memory_production_report":         {{flag: "--report", rel: "memory-production-linux-x64.json"}},
		"target_report":                    {{flag: ">", rel: "targets.json"}},
		"memory_fuzz_oracle_report":        {{flag: "--report-dir", rel: "memory-fuzz-tier1"}},
		"memory_fuzz_summary":              {{flag: "--report-dir", rel: "memory-fuzz-tier1"}},
		"memory_fuzz_island_proof_summary": {{flag: "--report-dir", rel: "memory-fuzz-tier1"}},
		"ram_contract_release_manifest":    {{flag: "--report-dir", rel: "ram-contract"}},
		"ram_contract_report":              {{flag: "--report-dir", rel: "ram-contract"}},
		"ram_memory_grade_report":          {{flag: "--report-dir", rel: "ram-contract"}},
		"ram_proof_store_summary":          {{flag: "--report-dir", rel: "ram-contract"}},
		"ram_validation_pipeline_coverage": {{flag: "--report-dir", rel: "ram-contract"}},
		"ram_heap_blockers":                {{flag: "--report-dir", rel: "ram-contract"}},
		"ram_copy_blockers":                {{flag: "--report-dir", rel: "ram-contract"}},
		"ram_contract_fuzz_oracle":         {{flag: "--report-dir", rel: "ram-contract"}},
		"ram_contract_hash_manifest":       {{flag: "--report-dir", rel: "ram-contract"}},
		"island_proof_verifier_report":     {{flag: "--proof", rel: "island-proof-verifier.json"}, {flag: "--memory-report", rel: "island-proof-memory-report.json"}},
		"island_proof_memory_report":       {{flag: "--proof", rel: "island-proof-verifier.json"}, {flag: "--memory-report", rel: "island-proof-memory-report.json"}},
		"artifact_hash_manifest":           {{flag: "--root", rel: ""}, {flag: "--out", rel: "artifact-hashes.json"}},
	}
	var issues []string
	requirements := requirementsByKind[artifact.Kind]
	for _, requirement := range requirements {
		wantPath := memoryDir
		if requirement.rel != "" {
			wantPath = filepath.Join(memoryDir, filepath.FromSlash(requirement.rel))
		}
		if !memory100CommandContainsAnyPath(artifact.Command, requirement.flag, memory100EquivalentPathForms(wantPath)) {
			issues = append(issues, fmt.Sprintf("memory release manifest artifact %s command must use %s under the current memory production report dir for %s", artifact.Kind, requirement.flag, requirement.rel))
		}
	}
	if strings.TrimSpace(gitHead) != "" {
		switch artifact.Kind {
		case "memory_production_report", "memory_fuzz_oracle_report", "memory_fuzz_summary", "memory_fuzz_island_proof_summary", "island_proof_verifier_report", "island_proof_memory_report":
			if !strings.Contains(artifact.Command, gitHead) {
				issues = append(issues, fmt.Sprintf("memory release manifest artifact %s command must include git head %s", artifact.Kind, gitHead))
			}
		}
	}
	if (artifact.Kind == "island_proof_verifier_report" || artifact.Kind == "island_proof_memory_report") && !strings.Contains(artifact.Command, "--require-same-commit") {
		issues = append(issues, fmt.Sprintf("memory release manifest artifact %s command must require same-commit validation", artifact.Kind))
	}
	return issues
}

func validateMemory100RAMContractReleaseManifest(path string, gitHead string) []string {
	var manifest memory100RAMContractReleaseManifest
	if err := readMemory100StrictJSON(path, &manifest); err != nil {
		return []string{fmt.Sprintf("RAM contract release manifest invalid: %v", err)}
	}
	var issues []string
	if manifest.Schema != "tetra.ram-contract.release-manifest.v1" {
		issues = append(issues, fmt.Sprintf("RAM contract release manifest schema is %q, want tetra.ram-contract.release-manifest.v1", manifest.Schema))
	}
	if manifest.Status != "pass" {
		issues = append(issues, fmt.Sprintf("RAM contract release manifest status is %q, want pass", manifest.Status))
	}
	if manifest.Target != "linux-x64" {
		issues = append(issues, fmt.Sprintf("RAM contract release manifest target is %q, want linux-x64", manifest.Target))
	}
	if gitHead != "" && manifest.GitHead != gitHead {
		issues = append(issues, fmt.Sprintf("RAM contract release manifest git_head %s does not match Memory100 git_head %s", manifest.GitHead, gitHead))
	}
	if _, err := time.Parse(time.RFC3339, manifest.GeneratedAt); err != nil {
		issues = append(issues, fmt.Sprintf("RAM contract release manifest generated_at must be RFC3339: %v", err))
	}
	if manifest.ReportDir != "." {
		issues = append(issues, fmt.Sprintf("RAM contract release manifest report_dir is %q, want .", manifest.ReportDir))
	}
	if manifest.HashManifest != "artifact-hashes.json" {
		issues = append(issues, fmt.Sprintf("RAM contract release manifest hash_manifest is %q, want artifact-hashes.json", manifest.HashManifest))
	}
	if len(nonEmptyMemory100Strings(manifest.NonClaims)) == 0 {
		issues = append(issues, "RAM contract release manifest non_claims must not be empty")
	}
	issues = append(issues, validateMemory100Claims("RAM contract release manifest non_claims", manifest.NonClaims, true)...)
	issues = append(issues, validateMemory100RAMContractReleaseCommands(manifest.Commands, filepath.Dir(path), gitHead)...)
	issues = append(issues, validateMemory100RAMContractReleaseArtifactRefs(manifest.Artifacts)...)
	return issues
}

func validateMemory100RAMContractReleaseCommands(commands []memory100Command, ramDir string, gitHead string) []string {
	seen := map[string]string{}
	var issues []string
	for _, command := range commands {
		name := strings.TrimSpace(command.Name)
		text := strings.TrimSpace(command.Command)
		if name == "" {
			issues = append(issues, "RAM contract release manifest command name is required")
			continue
		}
		if _, ok := seen[name]; ok {
			issues = append(issues, fmt.Sprintf("duplicate RAM contract release manifest command %s", name))
		}
		seen[name] = text
		if text == "" {
			issues = append(issues, fmt.Sprintf("RAM contract release manifest command %s command is required", name))
			continue
		}
		if strings.Contains(text, "|| true") || strings.Contains(text, "continue-on-error") || strings.Contains(text, "set +e") {
			issues = append(issues, fmt.Sprintf("RAM contract release manifest command %s contains bypass marker", name))
		}
	}
	for name, fragment := range requiredMemory100RAMContractReleaseCommands {
		text, ok := seen[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing RAM contract release manifest command %s containing %q", name, fragment))
			continue
		}
		if !strings.Contains(text, fragment) {
			issues = append(issues, fmt.Sprintf("RAM contract release manifest command %s must contain %q", name, fragment))
		}
	}
	type pathRequirement struct {
		name string
		flag string
		rel  string
	}
	for _, requirement := range []pathRequirement{
		{name: "validate-ram-contract-report", flag: "--report", rel: "ram-contract-report.json"},
		{name: "validate-memory-grade-report", flag: "--report", rel: "memory-grade-report.json"},
		{name: "validate-proof-store-summary", flag: "--report", rel: "proof-store-summary.json"},
		{name: "validate-validation-pipeline-coverage", flag: "--report", rel: "validation-pipeline-coverage.json"},
		{name: "validate-heap-blockers", flag: "--report", rel: "heap-blockers.json"},
		{name: "validate-copy-blockers", flag: "--report", rel: "copy-blockers.json"},
		{name: "ram-contract-fuzz-short", flag: "--report-dir", rel: "fuzz"},
		{name: "validate-ram-contract-fuzz-oracle", flag: "--report", rel: "fuzz/ram-contract-fuzz-oracle.json"},
		{name: "artifact-hashes-write", flag: "--root", rel: ""},
		{name: "artifact-hashes-write", flag: "--out", rel: "artifact-hashes.json"},
		{name: "artifact-hashes-validate", flag: "--manifest", rel: "artifact-hashes.json"},
		{name: "ram-contract-release-validator", flag: "--report-dir", rel: ""},
	} {
		text := strings.TrimSpace(seen[requirement.name])
		if text == "" {
			continue
		}
		wantPath := ramDir
		if requirement.rel != "" {
			wantPath = filepath.Join(ramDir, filepath.FromSlash(requirement.rel))
		}
		if !memory100CommandContainsAnyPath(text, requirement.flag, memory100EquivalentPathForms(wantPath)) {
			issues = append(issues, fmt.Sprintf("RAM contract release manifest command %s must use %s under the current RAM contract report dir for %s", requirement.name, requirement.flag, requirement.rel))
		}
	}
	for _, requirement := range []struct {
		name string
		flag string
	}{
		{name: "ram-contract-fuzz-short", flag: "--git-head"},
		{name: "validate-ram-contract-fuzz-oracle", flag: "--current-git-head"},
		{name: "ram-contract-release-validator", flag: "--current-git-head"},
	} {
		text := strings.TrimSpace(seen[requirement.name])
		if text == "" || strings.TrimSpace(gitHead) == "" {
			continue
		}
		if !strings.Contains(text, requirement.flag+" "+gitHead) {
			issues = append(issues, fmt.Sprintf("RAM contract release manifest command %s must use %s %s", requirement.name, requirement.flag, gitHead))
		}
	}
	return issues
}

func validateMemory100RAMContractReleaseArtifactRefs(artifacts []memory100RAMContractReleaseArtifactRef) []string {
	byKind := map[string]memory100RAMContractReleaseArtifactRef{}
	seenPath := map[string]bool{}
	requiredKinds := map[string]bool{}
	for _, required := range requiredMemory100RAMContractReleaseArtifacts {
		requiredKinds[required.Kind] = true
	}
	var issues []string
	for _, artifact := range artifacts {
		if err := validateMemory100SafeRel(artifact.Path); err != nil {
			issues = append(issues, fmt.Sprintf("RAM contract release manifest artifact path %q is invalid: %v", artifact.Path, err))
			continue
		}
		if strings.TrimSpace(artifact.Kind) == "" {
			issues = append(issues, fmt.Sprintf("RAM contract release manifest artifact %s kind is required", artifact.Path))
			continue
		}
		if !requiredKinds[artifact.Kind] {
			issues = append(issues, fmt.Sprintf("unexpected RAM contract release manifest artifact kind %s at %s", artifact.Kind, artifact.Path))
		}
		if _, ok := byKind[artifact.Kind]; ok {
			issues = append(issues, fmt.Sprintf("duplicate RAM contract release manifest artifact kind %s", artifact.Kind))
		}
		byKind[artifact.Kind] = artifact
		if seenPath[artifact.Path] {
			issues = append(issues, fmt.Sprintf("duplicate RAM contract release manifest artifact path %s", artifact.Path))
		}
		seenPath[artifact.Path] = true
	}
	for _, required := range requiredMemory100RAMContractReleaseArtifacts {
		artifact, ok := byKind[required.Kind]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing RAM contract release manifest artifact %s", required.Kind))
			continue
		}
		if artifact.Path != required.Path {
			issues = append(issues, fmt.Sprintf("RAM contract release manifest artifact %s path is %q, want %s", required.Kind, artifact.Path, required.Path))
		}
		if artifact.Schema != required.Schema {
			issues = append(issues, fmt.Sprintf("RAM contract release manifest artifact %s schema is %q, want %s", required.Kind, artifact.Schema, required.Schema))
		}
	}
	return issues
}

func validateMemory100IntegratedSurfaceEvidence(integratedDir string, gitHead string) []string {
	surfaceDir := filepath.Join(integratedDir, "surface-release-v1")
	var issues []string
	issues = append(issues, validateMemory100IntegratedSurfaceReleaseSummary(filepath.Join(surfaceDir, "surface-release-summary.json"), gitHead)...)
	issues = append(issues, validateMemory100NestedHashManifest(filepath.Join(surfaceDir, "artifact-hashes.json"), surfaceDir, "integrated surface release artifact-hashes.json")...)
	issues = append(issues, validateMemory100IntegratedSurfaceExperimentalRegression(filepath.Join(integratedDir, "surface-experimental-regression"))...)
	issues = append(issues, validateMemory100IntegratedSafeViewLifetimeSummary(filepath.Join(integratedDir, "safe-view-lifetime", "safe-view-lifetime-summary.json"))...)
	issues = append(issues, validateMemory100IntegratedSurfaceAPIStabilitySummary(filepath.Join(integratedDir, "surface-api-stability-v1", "surface-api-stability-summary.json"))...)
	return issues
}

func validateMemory100IntegratedSurfaceExperimentalRegression(dir string) []string {
	issues := validateMemory100NestedHashManifest(filepath.Join(dir, "artifact-hashes.json"), dir, "integrated surface experimental artifact-hashes.json")
	paths, err := listMemory100ArtifactPaths(dir)
	if err != nil {
		return append(issues, fmt.Sprintf("integrated surface experimental artifacts: %v", err))
	}
	for _, rel := range paths {
		if rel == "artifact-hashes.json" || !strings.HasSuffix(rel, ".json") {
			continue
		}
		path := filepath.Join(dir, filepath.FromSlash(rel))
		var envelope memory100SchemaEnvelope
		if err := readMemory100JSON(path, &envelope); err != nil {
			issues = append(issues, fmt.Sprintf("integrated surface experimental JSON %s invalid: %v", rel, err))
			continue
		}
		if memory100SchemaOf(envelope) != surface.SchemaV1 {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			issues = append(issues, fmt.Sprintf("integrated surface experimental runtime report %s unreadable: %v", rel, err))
			continue
		}
		if err := surface.ValidateReport(raw); err != nil {
			issues = append(issues, fmt.Sprintf("integrated surface experimental runtime report %s invalid: %v", rel, err))
		}
	}
	return issues
}

func validateMemory100IntegratedSurfaceAPIStabilitySummary(path string) []string {
	var summary struct {
		Schema                string `json:"schema"`
		Status                string `json:"status"`
		ReleaseScope          string `json:"release_scope"`
		DocsManifestValidated bool   `json:"docs_manifest_validated"`
	}
	if err := readMemory100JSON(path, &summary); err != nil {
		return []string{fmt.Sprintf("integrated surface API stability summary missing or invalid: %v", err)}
	}
	var issues []string
	if summary.Schema != "tetra.surface.api-stability.v1" {
		issues = append(issues, fmt.Sprintf("integrated surface API stability summary schema is %q, want tetra.surface.api-stability.v1", summary.Schema))
	}
	if summary.Status != "pass" {
		issues = append(issues, fmt.Sprintf("integrated surface API stability summary status is %q, want pass", summary.Status))
	}
	if summary.ReleaseScope != "surface-v1-linux-web" {
		issues = append(issues, fmt.Sprintf("integrated surface API stability summary release_scope is %q, want surface-v1-linux-web", summary.ReleaseScope))
	}
	if !summary.DocsManifestValidated {
		issues = append(issues, "integrated surface API stability summary docs_manifest_validated must be true")
	}
	return issues
}

func validateMemory100IntegratedSafeViewLifetimeSummary(path string) []string {
	var summary struct {
		Schema          string `json:"schema"`
		Status          string `json:"status"`
		Bounded         bool   `json:"bounded"`
		ReleaseBlocking bool   `json:"release_blocking"`
	}
	if err := readMemory100JSON(path, &summary); err != nil {
		return []string{fmt.Sprintf("integrated safe-view lifetime summary missing or invalid: %v", err)}
	}
	var issues []string
	if summary.Schema != "tetra.safe-view-lifetime.gate.v1" {
		issues = append(issues, fmt.Sprintf("integrated safe-view lifetime summary schema is %q, want tetra.safe-view-lifetime.gate.v1", summary.Schema))
	}
	if summary.Status != "pass" {
		issues = append(issues, fmt.Sprintf("integrated safe-view lifetime summary status is %q, want pass", summary.Status))
	}
	if !summary.Bounded {
		issues = append(issues, "integrated safe-view lifetime summary bounded must be true")
	}
	if !summary.ReleaseBlocking {
		issues = append(issues, "integrated safe-view lifetime summary release_blocking must be true")
	}
	return issues
}

func validateMemory100IntegratedSurfaceReleaseSummary(path string, gitHead string) []string {
	var summary struct {
		Schema       string `json:"schema"`
		Status       string `json:"status"`
		GitHead      string `json:"git_head"`
		ReleaseScope string `json:"release_scope"`
	}
	if err := readMemory100JSON(path, &summary); err != nil {
		return []string{fmt.Sprintf("integrated surface release summary missing or invalid: %v", err)}
	}
	var issues []string
	if summary.Schema != "tetra.surface.release.v1" {
		issues = append(issues, fmt.Sprintf("integrated surface release summary schema is %q, want tetra.surface.release.v1", summary.Schema))
	}
	if summary.Status != "current" {
		issues = append(issues, fmt.Sprintf("integrated surface release summary status is %q, want current", summary.Status))
	}
	if gitHead != "" && summary.GitHead != gitHead {
		issues = append(issues, fmt.Sprintf("integrated surface release summary git_head %s does not match Memory100 git_head %s", summary.GitHead, gitHead))
	}
	if summary.ReleaseScope != "surface-v1-linux-web" {
		issues = append(issues, fmt.Sprintf("integrated surface release summary release_scope is %q, want surface-v1-linux-web", summary.ReleaseScope))
	}
	return issues
}

func validateMemory100IntegratedNestedMemory(integratedDir string, gitHead string) []string {
	memoryDir := filepath.Join(integratedDir, "memory")
	productionPath := filepath.Join(memoryDir, "memory-production-linux-x64.json")
	var issues []string
	for _, issue := range validateMemory100MemoryProductionReport(productionPath) {
		issues = append(issues, "integrated "+issue)
	}
	var envelope memory100SchemaEnvelope
	if err := readMemory100JSON(productionPath, &envelope); err != nil {
		issues = append(issues, fmt.Sprintf("integrated memory production report envelope: %v", err))
	} else {
		if memory100SchemaOf(envelope) != "tetra.memory.production.v1" {
			issues = append(issues, fmt.Sprintf("integrated memory production report schema is %q, want tetra.memory.production.v1", memory100SchemaOf(envelope)))
		}
		if gitHead != "" && envelope.GitHead != gitHead {
			issues = append(issues, fmt.Sprintf("integrated memory production report git_head %s does not match Memory100 git_head %s", envelope.GitHead, gitHead))
		}
	}
	for _, issue := range validateMemory100MemoryReleaseManifest(filepath.Join(memoryDir, "memory-release-manifest.json"), gitHead) {
		issues = append(issues, "integrated "+issue)
	}
	issues = append(issues, validateMemory100IntegratedIslandProofEvidence(memoryDir, gitHead)...)
	for _, issue := range validateMemory100MemoryFuzzBundle(filepath.Join(memoryDir, "memory-fuzz-tier1"), gitHead) {
		issues = append(issues, "integrated "+issue)
	}
	issues = append(issues, validateMemory100NestedHashManifestWithRequired(filepath.Join(memoryDir, "artifact-hashes.json"), memoryDir, "integrated memory artifact-hashes.json", memory100MemoryReleaseRequiredHashPaths())...)
	return issues
}

func memory100MemoryReleaseRequiredHashPaths() []string {
	var paths []string
	for _, required := range requiredMemory100MemoryReleaseArtifacts {
		if required.Path == "artifact-hashes.json" {
			continue
		}
		paths = append(paths, required.Path)
	}
	return paths
}

func validateMemory100IntegratedIslandProofEvidence(memoryDir string, gitHead string) []string {
	return validateMemory100IslandProofEvidence(memoryDir, gitHead, "integrated island proof")
}

func validateMemory100IslandProofEvidence(memoryDir string, gitHead string, label string) []string {
	proofPath := filepath.Join(memoryDir, "island-proof-verifier.json")
	memoryPath := filepath.Join(memoryDir, "island-proof-memory-report.json")
	manifestPath := filepath.Join(memoryDir, "memory-release-manifest.json")
	proofRaw, err := os.ReadFile(proofPath)
	if err != nil {
		return []string{fmt.Sprintf("%s verifier unreadable: %v", label, err)}
	}
	memoryRaw, err := os.ReadFile(memoryPath)
	if err != nil {
		return []string{fmt.Sprintf("%s memory report unreadable: %v", label, err)}
	}
	manifestRaw, err := os.ReadFile(manifestPath)
	if err != nil {
		return []string{fmt.Sprintf("%s release manifest unreadable: %v", label, err)}
	}
	if err := islandproof.Validate(proofRaw, islandproof.Options{
		MemoryReport:      memoryRaw,
		Manifest:          manifestRaw,
		CurrentGitHead:    gitHead,
		RequireSameCommit: gitHead != "",
	}); err != nil {
		return []string{fmt.Sprintf("%s verifier: %v", label, err)}
	}
	return nil
}

func validateMemory100IntegratedIslandsDebugSmoke(path string, gitHead string) []string {
	var report struct {
		Target  string `json:"target"`
		GitHead string `json:"git_head"`
		Islands bool   `json:"islands_debug"`
		Total   *int   `json:"total"`
		Passed  *int   `json:"passed"`
		Failed  *int   `json:"failed"`
		Cases   []struct {
			Name         string `json:"name"`
			SrcPath      string `json:"src_path"`
			ExpectedExit int    `json:"expected_exit"`
			ActualExit   *int   `json:"actual_exit"`
			Ran          bool   `json:"ran"`
			Pass         bool   `json:"pass"`
		} `json:"cases"`
	}
	if err := readMemory100JSON(path, &report); err != nil {
		return []string{fmt.Sprintf("integrated islands debug smoke missing or invalid: %v", err)}
	}
	var issues []string
	if report.Target != "linux-x64" {
		issues = append(issues, fmt.Sprintf("integrated islands debug smoke target is %q, want linux-x64", report.Target))
	}
	if report.GitHead != "" && gitHead != "" && !memory100SameGitHead(report.GitHead, gitHead) {
		issues = append(issues, fmt.Sprintf("integrated islands debug smoke git_head %s does not match Memory100 git_head %s", report.GitHead, gitHead))
	}
	if !report.Islands {
		issues = append(issues, "integrated islands debug smoke islands_debug must be true")
	}
	if report.Total == nil || report.Passed == nil || report.Failed == nil {
		issues = append(issues, "integrated islands debug smoke counts are required")
	} else {
		passed := 0
		for _, c := range report.Cases {
			if c.Pass {
				passed++
			}
		}
		failed := len(report.Cases) - passed
		if *report.Total != len(report.Cases) || *report.Passed != passed || *report.Failed != failed {
			issues = append(issues, fmt.Sprintf("integrated islands debug smoke counts mismatch: got total=%d passed=%d failed=%d computed total=%d passed=%d failed=%d", *report.Total, *report.Passed, *report.Failed, len(report.Cases), passed, failed))
		}
	}
	foundTrap := false
	for _, c := range report.Cases {
		if c.Name != "islands_overflow" {
			continue
		}
		foundTrap = true
		if c.SrcPath != "examples/islands_overflow.tetra" {
			issues = append(issues, fmt.Sprintf("integrated islands debug smoke trap src_path is %q, want examples/islands_overflow.tetra", c.SrcPath))
		}
		if c.ExpectedExit == 0 {
			issues = append(issues, "integrated islands debug smoke trap expected_exit must be non-zero")
		}
		if c.ActualExit == nil {
			issues = append(issues, "integrated islands debug smoke trap missing actual_exit")
		} else if *c.ActualExit != c.ExpectedExit {
			issues = append(issues, fmt.Sprintf("integrated islands debug smoke trap actual_exit=%d, want %d", *c.ActualExit, c.ExpectedExit))
		}
		if !c.Ran || !c.Pass {
			issues = append(issues, "integrated islands debug smoke trap must run and pass")
		}
	}
	if !foundTrap {
		issues = append(issues, "integrated islands debug smoke missing islands_overflow trap")
	}
	return issues
}

func memory100SameGitHead(got string, want string) bool {
	got = strings.TrimSpace(got)
	want = strings.TrimSpace(want)
	if got == want {
		return true
	}
	if len(got) >= 7 && len(want) == 40 && strings.HasPrefix(want, got) {
		return true
	}
	if len(want) >= 7 && len(got) == 40 && strings.HasPrefix(got, want) {
		return true
	}
	return false
}

func validateMemory100IntegratedNestedRAMContract(integratedDir string, gitHead string) []string {
	ramDir := filepath.Join(integratedDir, "memory", "ram-contract")
	return validateMemory100NestedRAMContract(ramDir, gitHead, "integrated")
}

func validateMemory100NestedRAMContract(ramDir string, gitHead string, label string) []string {
	var issues []string
	for _, issue := range validateMemory100RAMContractBundleAt(ramDir, gitHead) {
		issues = append(issues, label+" "+issue)
	}
	for _, issue := range validateMemory100RAMContractReleaseManifest(filepath.Join(ramDir, "ram-contract-release-manifest.json"), gitHead) {
		issues = append(issues, label+" "+issue)
	}
	issues = append(issues, validateMemory100NestedHashManifest(filepath.Join(ramDir, "artifact-hashes.json"), ramDir, label+" RAM contract artifact-hashes.json")...)
	for _, issue := range validateMemory100RAMContractFuzzOracle(filepath.Join(ramDir, "fuzz", "ram-contract-fuzz-oracle.json"), gitHead) {
		issues = append(issues, label+" "+issue)
	}
	return issues
}

func validateMemory100NestedHashManifest(hashPath string, dir string, label string) []string {
	return validateMemory100NestedHashManifestWithRequired(hashPath, dir, label, nil)
}

func validateMemory100NestedHashManifestWithRequired(hashPath string, dir string, label string, requiredPaths []string) []string {
	var manifest memory100HashManifest
	if err := readMemory100StrictJSON(hashPath, &manifest); err != nil {
		return []string{fmt.Sprintf("%s missing or invalid: %v", label, err)}
	}
	var issues []string
	if manifest.Schema != memory100HashSchema {
		issues = append(issues, fmt.Sprintf("%s schema is %q, want %s", label, manifest.Schema, memory100HashSchema))
	}
	if manifest.Root != "." {
		issues = append(issues, fmt.Sprintf("%s root is %q, want .", label, manifest.Root))
	}
	if len(manifest.Artifacts) == 0 {
		issues = append(issues, fmt.Sprintf("%s artifacts must not be empty", label))
	}
	seen := map[string]memory100HashArtifact{}
	lastPath := ""
	for _, artifact := range manifest.Artifacts {
		if err := validateMemory100SafeRel(artifact.Path); err != nil {
			issues = append(issues, fmt.Sprintf("%s path %q is invalid: %v", label, artifact.Path, err))
			continue
		}
		if artifact.Path == "artifact-hashes.json" {
			issues = append(issues, fmt.Sprintf("%s must not list itself", label))
		}
		if lastPath != "" && artifact.Path < lastPath {
			issues = append(issues, fmt.Sprintf("%s artifacts must be sorted by path", label))
		}
		lastPath = artifact.Path
		if _, ok := seen[artifact.Path]; ok {
			issues = append(issues, fmt.Sprintf("duplicate %s entry for %s", label, artifact.Path))
		}
		seen[artifact.Path] = artifact
		if err := validateMemory100SHA256(artifact.SHA256, artifact.Path); err != nil {
			issues = append(issues, err.Error())
		}
		actual, err := hashMemory100File(dir, artifact.Path)
		if err != nil {
			issues = append(issues, fmt.Sprintf("hash %s artifact %s: %v", label, artifact.Path, err))
			continue
		}
		if actual.Size != artifact.Size {
			issues = append(issues, fmt.Sprintf("%s size mismatch for %s: got %d want %d", label, artifact.Path, actual.Size, artifact.Size))
		}
		if actual.SHA256 != artifact.SHA256 {
			issues = append(issues, fmt.Sprintf("%s sha256 mismatch for %s: got %s want %s", label, artifact.Path, actual.SHA256, artifact.SHA256))
		}
		if actual.Schema != artifact.Schema {
			issues = append(issues, fmt.Sprintf("%s schema mismatch for %s: got %q want %q", label, artifact.Path, actual.Schema, artifact.Schema))
		}
	}
	for _, rel := range requiredPaths {
		if _, ok := seen[rel]; !ok {
			issues = append(issues, fmt.Sprintf("missing %s entry for %s", label, rel))
		}
	}
	actualPaths, err := listMemory100ArtifactPaths(dir)
	if err != nil {
		issues = append(issues, fmt.Sprintf("list %s artifacts: %v", label, err))
	} else {
		for _, rel := range actualPaths {
			if _, ok := seen[rel]; !ok {
				issues = append(issues, fmt.Sprintf("unlisted %s artifact %s", label, rel))
			}
		}
	}
	return issues
}

func validateMemory100IntegratedManifest(path string, gitHead string) []string {
	var manifest memory100IntegratedManifest
	if err := readMemory100StrictJSON(path, &manifest); err != nil {
		return []string{fmt.Sprintf("integrated manifest invalid: %v", err)}
	}
	var issues []string
	if manifest.Schema != "tetra.memory-islands-surface.production-gate.v1" {
		issues = append(issues, fmt.Sprintf("integrated manifest schema is %q, want tetra.memory-islands-surface.production-gate.v1", manifest.Schema))
	}
	if manifest.Status != "pass" {
		issues = append(issues, fmt.Sprintf("integrated manifest status is %q, want pass", manifest.Status))
	}
	if gitHead != "" && manifest.GitHead != gitHead {
		issues = append(issues, fmt.Sprintf("integrated manifest git_head %s does not match Memory100 git_head %s", manifest.GitHead, gitHead))
	}
	if _, err := time.Parse(time.RFC3339, manifest.GeneratedAt); err != nil {
		issues = append(issues, fmt.Sprintf("integrated manifest generated_at must be RFC3339: %v", err))
	}
	if manifest.ReportDir != "." {
		issues = append(issues, fmt.Sprintf("integrated manifest report_dir is %q, want .", manifest.ReportDir))
	}
	if manifest.HashManifest != "artifact-hashes.json" {
		issues = append(issues, fmt.Sprintf("integrated manifest hash_manifest is %q, want artifact-hashes.json", manifest.HashManifest))
	}
	integratedDir := filepath.Dir(path)
	issues = append(issues, validateMemory100GeneratedAtFreshnessWithin(integratedDir, "memory-islands-surface-production-manifest.json", manifest.GeneratedAt, "integrated manifest")...)
	issues = append(issues, validateMemory100IntegratedCommands(manifest.Commands, integratedDir, gitHead)...)
	issues = append(issues, validateMemory100IntegratedArtifactRefs(manifest.Artifacts)...)
	return issues
}

func validateMemory100IntegratedCommands(commands []memory100Command, integratedDir string, gitHead string) []string {
	seen := map[string]string{}
	var issues []string
	for _, command := range commands {
		name := strings.TrimSpace(command.Name)
		text := strings.TrimSpace(command.Command)
		if name == "" {
			issues = append(issues, "integrated manifest command name is required")
			continue
		}
		if _, ok := seen[name]; ok {
			issues = append(issues, fmt.Sprintf("duplicate integrated manifest command %s", name))
		}
		seen[name] = text
		if text == "" {
			issues = append(issues, fmt.Sprintf("integrated manifest command %s command is required", name))
		}
		if strings.Contains(text, "|| true") || strings.Contains(text, "continue-on-error") || strings.Contains(text, "set +e") {
			issues = append(issues, fmt.Sprintf("integrated manifest command %s contains bypass marker", name))
		}
	}
	for name, fragment := range requiredMemory100IntegratedCommands {
		text, ok := seen[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing integrated manifest command %s containing %q", name, fragment))
			continue
		}
		if !strings.Contains(text, fragment) {
			issues = append(issues, fmt.Sprintf("integrated manifest command %s must contain %q", name, fragment))
		}
	}
	issues = append(issues, validateMemory100IntegratedCommandProvenance(seen, integratedDir, gitHead)...)
	return issues
}

func validateMemory100IntegratedCommandProvenance(commands map[string]string, integratedDir string, gitHead string) []string {
	type pathRequirement struct {
		name string
		flag string
		rel  string
	}
	pathRequirements := []pathRequirement{
		{name: "memory-production-gate", flag: "--report-dir", rel: "memory"},
		{name: "islands-debug-smoke", flag: "--report", rel: "islands-debug-smoke.json"},
		{name: "validate-islands-debug-smoke", flag: "--report", rel: "islands-debug-smoke.json"},
		{name: "surface-release-gate", flag: "--report-dir", rel: "surface-release-v1"},
		{name: "surface-experimental-regression-gate", flag: "--report-dir", rel: "surface-experimental-regression"},
		{name: "safe-view-lifetime-gate", flag: "--report-dir", rel: "safe-view-lifetime"},
		{name: "surface-api-stability-gate", flag: "--report-dir", rel: "surface-api-stability-v1"},
		{name: "artifact-hashes-write", flag: "--root", rel: ""},
		{name: "artifact-hashes-write", flag: "--out", rel: "artifact-hashes.json"},
		{name: "artifact-hashes-validate", flag: "--manifest", rel: "artifact-hashes.json"},
		{name: "integrated-release-validator", flag: "--report-dir", rel: ""},
	}
	var issues []string
	for _, requirement := range pathRequirements {
		text := strings.TrimSpace(commands[requirement.name])
		if text == "" {
			continue
		}
		wantPath := integratedDir
		if requirement.rel != "" {
			wantPath = filepath.Join(integratedDir, filepath.FromSlash(requirement.rel))
		}
		if !memory100CommandContainsAnyPath(text, requirement.flag, memory100EquivalentPathForms(wantPath)) {
			issues = append(issues, fmt.Sprintf("integrated manifest command %s must use %s under the current integrated report dir for %s", requirement.name, requirement.flag, requirement.rel))
		}
	}
	if text := strings.TrimSpace(commands["integrated-release-validator"]); text != "" && strings.TrimSpace(gitHead) != "" {
		if !strings.Contains(text, "--current-git-head "+gitHead) {
			issues = append(issues, fmt.Sprintf("integrated manifest command integrated-release-validator must use --current-git-head %s", gitHead))
		}
	}
	return issues
}

func validateMemory100IntegratedArtifactRefs(artifacts []memory100IntegratedArtifactRef) []string {
	byKind := map[string]memory100IntegratedArtifactRef{}
	seenPath := map[string]bool{}
	requiredKinds := map[string]bool{}
	for _, required := range requiredMemory100IntegratedArtifacts {
		requiredKinds[required.Kind] = true
	}
	var issues []string
	for _, artifact := range artifacts {
		if err := validateMemory100SafeRel(artifact.Path); err != nil {
			issues = append(issues, fmt.Sprintf("integrated manifest artifact path %q is invalid: %v", artifact.Path, err))
			continue
		}
		if strings.TrimSpace(artifact.Kind) == "" {
			issues = append(issues, fmt.Sprintf("integrated manifest artifact %s kind is required", artifact.Path))
			continue
		}
		if !requiredKinds[artifact.Kind] {
			issues = append(issues, fmt.Sprintf("unexpected integrated manifest artifact kind %s at %s", artifact.Kind, artifact.Path))
		}
		if _, ok := byKind[artifact.Kind]; ok {
			issues = append(issues, fmt.Sprintf("duplicate integrated manifest artifact kind %s", artifact.Kind))
		}
		byKind[artifact.Kind] = artifact
		if seenPath[artifact.Path] {
			issues = append(issues, fmt.Sprintf("duplicate integrated manifest artifact path %s", artifact.Path))
		}
		seenPath[artifact.Path] = true
	}
	for _, required := range requiredMemory100IntegratedArtifacts {
		artifact, ok := byKind[required.Kind]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing integrated manifest artifact %s", required.Kind))
			continue
		}
		if artifact.Path != required.Path {
			issues = append(issues, fmt.Sprintf("integrated manifest artifact %s path is %q, want %s", required.Kind, artifact.Path, required.Path))
		}
		if required.Schema != "" && artifact.Schema != required.Schema {
			issues = append(issues, fmt.Sprintf("integrated manifest artifact %s schema is %q, want %s", required.Kind, artifact.Schema, required.Schema))
		}
	}
	return issues
}

func validateMemory100RAMContractBundle(reportDir string, gitHead string) []string {
	ramDir := filepath.Join(reportDir, "ram-contract")
	return validateMemory100RAMContractBundleAt(ramDir, gitHead)
}

func validateMemory100RAMContractBundleAt(ramDir string, gitHead string) []string {
	reportPath := filepath.Join(ramDir, "ram-contract-report.json")
	gradePath := filepath.Join(ramDir, "memory-grade-report.json")
	proofPath := filepath.Join(ramDir, "proof-store-summary.json")
	pipelinePath := filepath.Join(ramDir, "validation-pipeline-coverage.json")
	heapPath := filepath.Join(ramDir, "heap-blockers.json")
	copyPath := filepath.Join(ramDir, "copy-blockers.json")

	var issues []string
	var report ramvalidate.Report
	reportOK := false
	if err := ramvalidate.ReadStrictJSONFile(reportPath, &report); err != nil {
		issues = append(issues, fmt.Sprintf("ram-contract-report.json: %v", err))
	} else {
		reportOK = true
		if err := ramvalidate.ValidateReport(report); err != nil {
			issues = append(issues, fmt.Sprintf("ram-contract-report.json: %v", err))
		}
		if gitHead != "" && report.GitHead != gitHead {
			issues = append(issues, fmt.Sprintf("ram-contract-report.json git_head %s does not match Memory100 git_head %s", report.GitHead, gitHead))
		}
	}
	if err := ramvalidate.ValidateGradeReportFile(gradePath); err != nil {
		issues = append(issues, fmt.Sprintf("memory-grade-report.json: %v", err))
	}
	if err := ramvalidate.ValidateProofStoreSummaryFile(proofPath); err != nil {
		issues = append(issues, fmt.Sprintf("proof-store-summary.json: %v", err))
	}
	if err := ramvalidate.ValidatePipelineCoverageFile(pipelinePath); err != nil {
		issues = append(issues, fmt.Sprintf("validation-pipeline-coverage.json: %v", err))
	}
	if err := ramvalidate.ValidateBlockerReportFile(heapPath, "heap"); err != nil {
		issues = append(issues, fmt.Sprintf("heap-blockers.json: %v", err))
	}
	if err := ramvalidate.ValidateBlockerReportFile(copyPath, "copy"); err != nil {
		issues = append(issues, fmt.Sprintf("copy-blockers.json: %v", err))
	}

	var heapBlockers ramvalidate.BlockerReport
	heapOK := false
	if err := ramvalidate.ReadStrictJSONFile(heapPath, &heapBlockers); err != nil {
		issues = append(issues, fmt.Sprintf("heap-blockers.json: %v", err))
	} else {
		heapOK = true
	}
	var copyBlockers ramvalidate.BlockerReport
	copyOK := false
	if err := ramvalidate.ReadStrictJSONFile(copyPath, &copyBlockers); err != nil {
		issues = append(issues, fmt.Sprintf("copy-blockers.json: %v", err))
	} else {
		copyOK = true
	}
	if reportOK && heapOK && copyOK {
		issues = append(issues, validateMemory100RAMHeapCopyClassification(report, heapBlockers, copyBlockers)...)
	}
	return issues
}

func validateMemory100RAMHeapCopyClassification(report ramvalidate.Report, heapBlockers ramvalidate.BlockerReport, copyBlockers ramvalidate.BlockerReport) []string {
	rowsBySite := map[string]ramvalidate.Row{}
	heapRows := map[string]ramvalidate.Row{}
	copyRows := map[string]ramvalidate.Row{}
	var issues []string
	for i, row := range report.Rows {
		if strings.TrimSpace(row.SiteID) == "" {
			continue
		}
		rowsBySite[row.SiteID] = row
		if memory100RAMRowIsHeap(row) {
			heapRows[row.SiteID] = row
			if memory100RAMUnclassified(row.ValidationStatus) {
				issues = append(issues, fmt.Sprintf("ram-contract-report.json heap row %d site_id %q has unclassified validation_status %q", i, row.SiteID, row.ValidationStatus))
			}
			if len(nonEmptyMemory100Strings(row.Blockers)) == 0 {
				issues = append(issues, fmt.Sprintf("ram-contract-report.json heap row %d site_id %q has no classified blockers", i, row.SiteID))
			}
			for _, blocker := range row.Blockers {
				if memory100RAMUnclassified(blocker) {
					issues = append(issues, fmt.Sprintf("ram-contract-report.json heap row %d site_id %q has unclassified blocker %q", i, row.SiteID, blocker))
				}
			}
		}
		if memory100RAMRowIsCopy(row) {
			copyRows[row.SiteID] = row
			if memory100RAMUnclassified(row.ValidationStatus) {
				issues = append(issues, fmt.Sprintf("ram-contract-report.json copy row %d site_id %q has unclassified validation_status %q", i, row.SiteID, row.ValidationStatus))
			}
			if memory100RAMUnclassified(row.CopyReason) {
				issues = append(issues, fmt.Sprintf("ram-contract-report.json copy row %d site_id %q has unclassified copy_reason %q", i, row.SiteID, row.CopyReason))
			}
		}
	}

	heapBlockerSites := map[string]ramvalidate.BlockerRow{}
	for i, row := range heapBlockers.Rows {
		if memory100RAMUnclassified(strings.Join(row.Blockers, "\n")) {
			issues = append(issues, fmt.Sprintf("heap-blockers.json row %d site_id %q has unclassified blockers", i, row.SiteID))
		}
		heapBlockerSites[row.SiteID] = row
		ramRow, ok := rowsBySite[row.SiteID]
		if !ok {
			issues = append(issues, fmt.Sprintf("heap-blockers.json row %d site_id %q missing from ram-contract-report.json", i, row.SiteID))
			continue
		}
		if !memory100RAMRowIsHeap(ramRow) {
			issues = append(issues, fmt.Sprintf("heap-blockers.json row %d site_id %q is not a heap RAM report row", i, row.SiteID))
		}
	}
	for siteID := range heapRows {
		if _, ok := heapBlockerSites[siteID]; !ok {
			issues = append(issues, fmt.Sprintf("ram-contract-report.json heap row site_id %q missing from heap-blockers.json", siteID))
		}
	}

	copyBlockerSites := map[string]ramvalidate.BlockerRow{}
	for i, row := range copyBlockers.Rows {
		if memory100RAMUnclassified(row.CopyReason) {
			issues = append(issues, fmt.Sprintf("copy-blockers.json row %d site_id %q has unclassified copy_reason", i, row.SiteID))
		}
		copyBlockerSites[row.SiteID] = row
		ramRow, ok := rowsBySite[row.SiteID]
		if !ok {
			issues = append(issues, fmt.Sprintf("copy-blockers.json row %d site_id %q missing from ram-contract-report.json", i, row.SiteID))
			continue
		}
		if !memory100RAMRowIsCopy(ramRow) {
			issues = append(issues, fmt.Sprintf("copy-blockers.json row %d site_id %q is not a copy RAM report row", i, row.SiteID))
		}
	}
	for siteID := range copyRows {
		if _, ok := copyBlockerSites[siteID]; !ok {
			issues = append(issues, fmt.Sprintf("ram-contract-report.json copy row site_id %q missing from copy-blockers.json", siteID))
		}
	}
	return issues
}

func validateMemory100AllocationLoweringRAMConsistency(reportDir string) []string {
	allocationPath := filepath.Join(reportDir, "allocation-lowering", "allocation-lowering-report.json")
	ramPath := filepath.Join(reportDir, "ram-contract", "ram-contract-report.json")
	heapPath := filepath.Join(reportDir, "ram-contract", "heap-blockers.json")
	copyPath := filepath.Join(reportDir, "ram-contract", "copy-blockers.json")
	var allocation memory100AllocationLoweringReport
	if err := readMemory100JSON(allocationPath, &allocation); err != nil {
		return []string{fmt.Sprintf("allocation lowering invalid: %v", err)}
	}
	var report ramvalidate.Report
	if err := ramvalidate.ReadStrictJSONFile(ramPath, &report); err != nil {
		return []string{fmt.Sprintf("ram-contract-report.json: %v", err)}
	}
	var heapBlockers ramvalidate.BlockerReport
	if err := ramvalidate.ReadStrictJSONFile(heapPath, &heapBlockers); err != nil {
		return []string{fmt.Sprintf("heap-blockers.json: %v", err)}
	}
	var copyBlockers ramvalidate.BlockerReport
	if err := ramvalidate.ReadStrictJSONFile(copyPath, &copyBlockers); err != nil {
		return []string{fmt.Sprintf("copy-blockers.json: %v", err)}
	}

	proofBackedTrustedRows := 0
	rowsBySite := map[string]ramvalidate.Row{}
	proofBackedTrustedSites := map[string]struct{}{}
	heapSites := map[string]struct{}{}
	copySites := map[string]struct{}{}
	for _, row := range report.Rows {
		if strings.TrimSpace(row.SiteID) == "" {
			continue
		}
		rowsBySite[row.SiteID] = row
		if memory100RAMRowIsHeap(row) {
			heapSites[row.SiteID] = struct{}{}
		}
		if memory100RAMRowIsCopy(row) {
			copySites[row.SiteID] = struct{}{}
		}
		if memory100RAMTrustedPlacement(row.Placement) &&
			row.EscapeStatus == "no_escape" &&
			row.ValidationStatus == "validated" &&
			len(nonEmptyMemory100Strings(row.ProofIDs)) > 0 {
			proofBackedTrustedRows++
			proofBackedTrustedSites[row.SiteID] = struct{}{}
		}
	}
	heapBlockerSites := map[string]struct{}{}
	for _, row := range heapBlockers.Rows {
		if strings.TrimSpace(row.SiteID) != "" {
			heapBlockerSites[row.SiteID] = struct{}{}
		}
	}
	copyBlockerSites := map[string]struct{}{}
	for _, row := range copyBlockers.Rows {
		if strings.TrimSpace(row.SiteID) != "" {
			copyBlockerSites[row.SiteID] = struct{}{}
		}
	}

	var issues []string
	proofCoveredSites := map[string]struct{}{}
	for _, decision := range allocation.Decisions {
		status := memory100AllocationDecisionStatus(decision)
		coveredSites := memory100StringSet(decision.CoveredSiteIDs)
		for siteID := range coveredSites {
			row, ok := rowsBySite[siteID]
			if !ok {
				issues = append(issues, fmt.Sprintf("allocation lowering decision %s covered_site_ids site_id %q missing from ram-contract-report.json", decision.Name, siteID))
				continue
			}
			if !memory100AllocationActualStorageMatchesRAMRow(decision, row) {
				issues = append(issues, fmt.Sprintf("allocation lowering decision %s actual_lowering_storage %q contradicts ram-contract-report.json site_id %q placement %q", decision.Name, decision.ActualLoweringStorage, siteID, row.Placement))
			}
		}
		if status == "not_observed" {
			switch decision.Name {
			case "heap_fallback_blocker":
				if len(heapSites) > 0 {
					issues = append(issues, fmt.Sprintf("allocation lowering decision %s is not_observed but ram-contract-report.json has heap rows %v", decision.Name, memory100SortedSetKeys(heapSites)))
				}
			case "copy_blocker":
				if len(copySites) > 0 {
					issues = append(issues, fmt.Sprintf("allocation lowering decision %s is not_observed but ram-contract-report.json has copy rows %v", decision.Name, memory100SortedSetKeys(copySites)))
				}
			}
			continue
		}
		if strings.TrimSpace(decision.ProofArtifact) == "" {
			switch decision.Name {
			case "heap_fallback_blocker":
				for siteID := range coveredSites {
					if _, ok := heapSites[siteID]; !ok {
						issues = append(issues, fmt.Sprintf("allocation lowering decision %s covered_site_ids site_id %q is not a heap RAM row", decision.Name, siteID))
					}
					if _, ok := heapBlockerSites[siteID]; !ok {
						issues = append(issues, fmt.Sprintf("allocation lowering decision %s covered_site_ids site_id %q missing from heap-blockers.json", decision.Name, siteID))
					}
				}
				for _, siteID := range memory100MissingSetKeys(heapSites, coveredSites) {
					issues = append(issues, fmt.Sprintf("allocation lowering decision %s missing heap RAM row site_id %q in covered_site_ids", decision.Name, siteID))
				}
			case "copy_blocker":
				for siteID := range coveredSites {
					if _, ok := copySites[siteID]; !ok {
						issues = append(issues, fmt.Sprintf("allocation lowering decision %s covered_site_ids site_id %q is not a copy RAM row", decision.Name, siteID))
					}
					if _, ok := copyBlockerSites[siteID]; !ok {
						issues = append(issues, fmt.Sprintf("allocation lowering decision %s covered_site_ids site_id %q missing from copy-blockers.json", decision.Name, siteID))
					}
				}
				for _, siteID := range memory100MissingSetKeys(copySites, coveredSites) {
					issues = append(issues, fmt.Sprintf("allocation lowering decision %s missing copy RAM row site_id %q in covered_site_ids", decision.Name, siteID))
				}
			}
			continue
		}
		if proofBackedTrustedRows == 0 {
			issues = append(issues, fmt.Sprintf("allocation lowering decision %s is proof-backed but ram-contract-report.json has no proof-backed trusted rows", decision.Name))
		}
		for siteID := range coveredSites {
			if _, ok := proofBackedTrustedSites[siteID]; !ok {
				issues = append(issues, fmt.Sprintf("allocation lowering decision %s covered_site_ids site_id %q is not a proof-backed trusted RAM row", decision.Name, siteID))
				continue
			}
			if status == "proven" {
				proofCoveredSites[siteID] = struct{}{}
			}
		}
	}
	for _, siteID := range memory100MissingSetKeys(proofBackedTrustedSites, proofCoveredSites) {
		issues = append(issues, fmt.Sprintf("allocation lowering missing proof-backed trusted RAM row site_id %q in proven covered_site_ids", siteID))
	}
	return issues
}

func memory100StringSet(values []string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	return set
}

func memory100MissingSetKeys(want map[string]struct{}, got map[string]struct{}) []string {
	var missing []string
	for value := range want {
		if _, ok := got[value]; !ok {
			missing = append(missing, value)
		}
	}
	sort.Strings(missing)
	return missing
}

func memory100SortedSetKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func memory100RAMRowIsHeap(row ramvalidate.Row) bool {
	return row.Placement == "heap_bounded" || row.Placement == "heap_unbounded"
}

func memory100RAMRowIsCopy(row ramvalidate.Row) bool {
	return strings.HasPrefix(row.Intent, "copy")
}

func memory100RAMUnclassified(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "", "unknown", "unclassified", "todo", "tbd", "none", "n/a":
		return true
	default:
		return false
	}
}

func memory100RAMTrustedPlacement(placement string) bool {
	switch placement {
	case "eliminated", "register", "stack", "static", "interned", "island", "region":
		return true
	default:
		return false
	}
}

func memory100AllocationActualStorageMatchesRAMRow(decision memory100AllocationLoweringDecision, row ramvalidate.Row) bool {
	actual := strings.TrimSpace(decision.ActualLoweringStorage)
	switch actual {
	case "Copy":
		return memory100RAMRowIsCopy(row)
	case "Eliminated":
		return row.Placement == "eliminated"
	case "Register":
		return row.Placement == "register"
	case "Stack":
		return row.Placement == "stack"
	case "Static":
		return row.Placement == "static"
	case "Interned":
		return row.Placement == "interned"
	case "Region", "FunctionTempRegion", "TaskRegion", "ActorMoveRegion":
		return row.Placement == "region"
	case "ExplicitIsland":
		return row.Placement == "island"
	case "Heap", "LargeMmap":
		return memory100RAMRowIsHeap(row)
	case "External":
		return row.Placement == "external"
	case "Rejected":
		return row.Placement == "rejected"
	default:
		return actual == ""
	}
}

func validateMemory100RawMemoryContract(path string, gitHead string) []string {
	var report struct {
		Status     string `json:"status"`
		GitHead    string `json:"git_head"`
		Operations []struct {
			Name            string   `json:"name"`
			SourceArtifacts []string `json:"source_artifacts"`
			PositiveTests   []string `json:"positive_tests"`
			NegativeTests   []string `json:"negative_tests"`
			NonClaims       []string `json:"non_claims"`
		} `json:"operations"`
	}
	if err := readMemory100JSON(path, &report); err != nil {
		return []string{fmt.Sprintf("raw memory contract invalid: %v", err)}
	}
	var issues []string
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("raw memory contract status is %q, want pass", report.Status))
	}
	if gitHead != "" && report.GitHead != gitHead {
		issues = append(issues, fmt.Sprintf("raw memory contract git_head %s does not match Memory100 git_head %s", report.GitHead, gitHead))
	}
	byName := map[string]struct {
		SourceArtifacts []string
		PositiveTests   []string
		NegativeTests   []string
		NonClaims       []string
	}{}
	for _, operation := range report.Operations {
		byName[operation.Name] = struct {
			SourceArtifacts []string
			PositiveTests   []string
			NegativeTests   []string
			NonClaims       []string
		}{
			SourceArtifacts: operation.SourceArtifacts,
			PositiveTests:   operation.PositiveTests,
			NegativeTests:   operation.NegativeTests,
			NonClaims:       operation.NonClaims,
		}
	}
	for _, name := range []string{"core.alloc_bytes", "core.ptr_add", "raw_slice_from_parts", "raw_load_store_metadata", "memcpy_u8", "memset_u8", "cap.mem"} {
		operation, ok := byName[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("raw memory contract missing operation %s", name))
			continue
		}
		if len(nonEmptyMemory100Strings(operation.SourceArtifacts)) == 0 {
			issues = append(issues, fmt.Sprintf("raw memory contract operation %s missing source_artifacts", name))
		}
		if len(nonEmptyMemory100Strings(operation.PositiveTests))+len(nonEmptyMemory100Strings(operation.NegativeTests)) == 0 {
			issues = append(issues, fmt.Sprintf("raw memory contract operation %s missing positive_tests or negative_tests", name))
		}
		if name == "cap.mem" && len(nonEmptyMemory100Strings(operation.NonClaims)) == 0 {
			issues = append(issues, "raw memory contract operation cap.mem missing non_claims")
		}
		switch name {
		case "core.alloc_bytes":
			issues = append(issues, requireMemory100RawEvidence(name, "source_artifacts", operation.SourceArtifacts, "compiler/internal/runtimeabi/raw_pointer_bounds_test.go")...)
			issues = append(issues, requireMemory100RawEvidence(name, "positive_tests", operation.PositiveTests, "allocation-base metadata")...)
		case "core.ptr_add":
			issues = append(issues, requireMemory100RawEvidence(name, "source_artifacts", operation.SourceArtifacts, "compiler/internal/runtimeabi/raw_pointer_bounds_test.go")...)
			issues = append(issues, requireMemory100RawEvidence(name, "negative_tests", operation.NegativeTests, "negative offset", "upper bound", "access-width overflow")...)
		case "raw_slice_from_parts":
			issues = append(issues, requireMemory100RawEvidence(name, "source_artifacts", operation.SourceArtifacts, "compiler/internal/runtimeabi/raw_pointer_bounds_test.go", "compiler/tests/semantics/memory_ideal_v5_raw_pointer_test.go")...)
			issues = append(issues, requireMemory100RawEvidence(name, "negative_tests", operation.NegativeTests, "outside unsafe", "negative length", "i32 byte overflow")...)
		case "raw_load_store_metadata":
			issues = append(issues, requireMemory100RawEvidence(name, "source_artifacts", operation.SourceArtifacts, "compiler/internal/plir/plir_test.go", "compiler/internal/lower/raw_memory_test.go", "compiler/internal/memoryfacts/from_plir_test.go")...)
			issues = append(issues, requireMemory100RawEvidence(name, "positive_tests", operation.PositiveTests, "IRMemWriteI32Offset", "IRMemReadI32Offset", "raw memory gateway", "UnsafeChecked")...)
			issues = append(issues, requireMemory100RawEvidence(name, "negative_tests", operation.NegativeTests, "checked_external_unknown", "rejected_access_width_overflow")...)
		case "memcpy_u8":
			issues = append(issues, requireMemory100RawEvidence(name, "source_artifacts", operation.SourceArtifacts, "lib/core/memory.tetra")...)
			issues = append(issues, requireMemory100RawEvidence(name, "negative_tests", operation.NegativeTests, "negative length", "access-width overflow")...)
			issues = append(issues, requireMemory100RawEvidence(name, "non_claims", operation.NonClaims, "overlapping memcpy")...)
		case "memset_u8":
			issues = append(issues, requireMemory100RawEvidence(name, "source_artifacts", operation.SourceArtifacts, "lib/core/memory.tetra")...)
			issues = append(issues, requireMemory100RawEvidence(name, "negative_tests", operation.NegativeTests, "negative length", "access-width overflow")...)
		case "cap.mem":
			issues = append(issues, requireMemory100RawEvidence(name, "source_artifacts", operation.SourceArtifacts, "lib/core/capability.tetra")...)
			issues = append(issues, requireMemory100RawEvidence(name, "negative_tests", operation.NegativeTests, "unsafe_unknown", "overclaim")...)
			issues = append(issues, requireMemory100RawEvidence(name, "non_claims", operation.NonClaims, "no arbitrary external pointer safety claim")...)
		}
	}
	return issues
}

func requireMemory100RawEvidence(operation string, field string, values []string, wants ...string) []string {
	joined := strings.ToLower(strings.Join(nonEmptyMemory100Strings(values), "\n"))
	var issues []string
	for _, want := range wants {
		if !strings.Contains(joined, strings.ToLower(want)) {
			issues = append(issues, fmt.Sprintf("raw memory contract operation %s %s missing %q", operation, field, want))
		}
	}
	return issues
}

func validateMemory100AllocationLowering(path string, gitHead string) []string {
	var report memory100AllocationLoweringReport
	if err := readMemory100JSON(path, &report); err != nil {
		return []string{fmt.Sprintf("allocation lowering invalid: %v", err)}
	}
	var issues []string
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("allocation lowering status is %q, want pass", report.Status))
	}
	if gitHead != "" && report.GitHead != gitHead {
		issues = append(issues, fmt.Sprintf("allocation lowering git_head %s does not match Memory100 git_head %s", report.GitHead, gitHead))
	}
	byName := map[string]struct {
		Status                string
		PlannedStorage        string
		ActualLoweringStorage string
		ProofArtifact         string
		BlockerArtifact       string
		BlockerReason         string
		BudgetImpact          string
		GradeImpact           string
		ValidatorCoverage     []string
		SourceArtifacts       []string
		CoveredSiteIDs        []string
	}{}
	for _, decision := range report.Decisions {
		byName[decision.Name] = struct {
			Status                string
			PlannedStorage        string
			ActualLoweringStorage string
			ProofArtifact         string
			BlockerArtifact       string
			BlockerReason         string
			BudgetImpact          string
			GradeImpact           string
			ValidatorCoverage     []string
			SourceArtifacts       []string
			CoveredSiteIDs        []string
		}{
			Status:                decision.Status,
			PlannedStorage:        decision.PlannedStorage,
			ActualLoweringStorage: decision.ActualLoweringStorage,
			ProofArtifact:         decision.ProofArtifact,
			BlockerArtifact:       decision.BlockerArtifact,
			BlockerReason:         decision.BlockerReason,
			BudgetImpact:          decision.BudgetImpact,
			GradeImpact:           decision.GradeImpact,
			ValidatorCoverage:     decision.ValidatorCoverage,
			SourceArtifacts:       decision.SourceArtifacts,
			CoveredSiteIDs:        decision.CoveredSiteIDs,
		}
	}
	for _, name := range []string{"stack_trusted_no_escape", "heap_fallback_blocker", "copy_blocker", "lowering_storage_match"} {
		decision, ok := byName[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("allocation lowering missing decision %s", name))
			continue
		}
		if strings.TrimSpace(decision.PlannedStorage) == "" || strings.TrimSpace(decision.ActualLoweringStorage) == "" {
			issues = append(issues, fmt.Sprintf("allocation lowering decision %s missing planned/actual storage", name))
		}
		if len(nonEmptyMemory100Strings(decision.SourceArtifacts)) == 0 {
			issues = append(issues, fmt.Sprintf("allocation lowering decision %s missing source_artifacts", name))
		}
		status := memory100AllocationDecisionStatus(memory100AllocationLoweringDecision{
			Status:          decision.Status,
			ProofArtifact:   decision.ProofArtifact,
			BlockerArtifact: decision.BlockerArtifact,
		})
		switch status {
		case "", "proven", "blocked", "not_observed":
		default:
			issues = append(issues, fmt.Sprintf("allocation lowering decision %s has unknown status %q", name, decision.Status))
		}
		planned := strings.TrimSpace(decision.PlannedStorage)
		actual := strings.TrimSpace(decision.ActualLoweringStorage)
		if planned != "" && actual != "" && !strings.EqualFold(planned, actual) {
			if strings.TrimSpace(decision.BlockerArtifact) == "" {
				issues = append(issues, fmt.Sprintf("allocation lowering decision %s planned/actual mismatch %s -> %s requires blocker_artifact", name, planned, actual))
			}
			issues = append(issues, requireMemory100AllocationField(name, "blocker_reason", decision.BlockerReason)...)
			issues = append(issues, requireMemory100AllocationField(name, "budget_impact", decision.BudgetImpact)...)
			issues = append(issues, requireMemory100AllocationField(name, "grade_impact", decision.GradeImpact)...)
		}
		if status == "not_observed" {
			if strings.TrimSpace(decision.ProofArtifact) != "" || strings.TrimSpace(decision.BlockerArtifact) != "" {
				issues = append(issues, fmt.Sprintf("allocation lowering decision %s is not_observed but carries proof/blocker artifact", name))
			}
			if len(nonEmptyMemory100Strings(decision.CoveredSiteIDs)) > 0 {
				issues = append(issues, fmt.Sprintf("allocation lowering decision %s is not_observed but carries covered_site_ids", name))
			}
			continue
		}
		if len(nonEmptyMemory100Strings(decision.CoveredSiteIDs)) == 0 {
			issues = append(issues, fmt.Sprintf("allocation lowering decision %s missing covered_site_ids", name))
		}
		if status == "proven" && strings.TrimSpace(decision.ProofArtifact) == "" {
			issues = append(issues, fmt.Sprintf("allocation lowering decision %s status proven requires proof_artifact", name))
		}
		if status == "blocked" && strings.TrimSpace(decision.BlockerArtifact) == "" {
			issues = append(issues, fmt.Sprintf("allocation lowering decision %s status blocked requires blocker_artifact", name))
		}
		if strings.TrimSpace(decision.ProofArtifact) == "" && strings.TrimSpace(decision.BlockerArtifact) == "" {
			issues = append(issues, fmt.Sprintf("allocation lowering decision %s missing proof_artifact or blocker_artifact", name))
		}
		switch name {
		case "heap_fallback_blocker":
			if !strings.Contains(filepath.ToSlash(decision.BlockerArtifact), "ram-contract/heap-blockers.json") {
				issues = append(issues, fmt.Sprintf("allocation lowering decision %s blocker_artifact must reference ram-contract/heap-blockers.json", name))
			}
			issues = append(issues, requireMemory100AllocationField(name, "blocker_reason", decision.BlockerReason)...)
			issues = append(issues, requireMemory100AllocationField(name, "budget_impact", decision.BudgetImpact)...)
			issues = append(issues, requireMemory100AllocationField(name, "grade_impact", decision.GradeImpact)...)
			issues = append(issues, requireMemory100AllocationCoverage(name, decision.ValidatorCoverage, "validate-heap-blockers", "validate-ram-contract-release")...)
		case "copy_blocker":
			if !strings.Contains(filepath.ToSlash(decision.BlockerArtifact), "ram-contract/copy-blockers.json") {
				issues = append(issues, fmt.Sprintf("allocation lowering decision %s blocker_artifact must reference ram-contract/copy-blockers.json", name))
			}
			issues = append(issues, requireMemory100AllocationField(name, "blocker_reason", decision.BlockerReason)...)
			issues = append(issues, requireMemory100AllocationField(name, "budget_impact", decision.BudgetImpact)...)
			issues = append(issues, requireMemory100AllocationField(name, "grade_impact", decision.GradeImpact)...)
			issues = append(issues, requireMemory100AllocationCoverage(name, decision.ValidatorCoverage, "validate-copy-blockers", "validate-ram-contract-release")...)
		}
	}
	return issues
}

func memory100AllocationDecisionStatus(decision memory100AllocationLoweringDecision) string {
	status := strings.TrimSpace(decision.Status)
	if status != "" {
		return status
	}
	if strings.TrimSpace(decision.ProofArtifact) != "" {
		return "proven"
	}
	if strings.TrimSpace(decision.BlockerArtifact) != "" {
		return "blocked"
	}
	return ""
}

func requireMemory100AllocationField(decision string, field string, value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{fmt.Sprintf("allocation lowering decision %s missing %s", decision, field)}
	}
	return nil
}

func requireMemory100AllocationCoverage(decision string, values []string, wants ...string) []string {
	joined := strings.ToLower(strings.Join(nonEmptyMemory100Strings(values), "\n"))
	var issues []string
	for _, want := range wants {
		if !strings.Contains(joined, strings.ToLower(want)) {
			issues = append(issues, fmt.Sprintf("allocation lowering decision %s validator_coverage missing %q", decision, want))
		}
	}
	return issues
}

func validateMemory100MemoryFuzzBundle(dir string, gitHead string) []string {
	var issues []string
	for _, rel := range []string{"memory-fuzz-oracle.json", "summary.md", "summary.json", "island-proof-fuzz-summary.json", "artifact-hashes.json"} {
		issues = append(issues, requireMemory100MemoryFuzzFile(dir, rel)...)
	}
	for _, rel := range []string{"reproducers/compiler-crash", "reproducers/miscompile", "reducers/miscompile"} {
		issues = append(issues, requireMemory100MemoryFuzzDir(dir, rel)...)
	}
	if len(issues) > 0 {
		return issues
	}

	summaryMD, err := os.ReadFile(filepath.Join(dir, "summary.md"))
	if err != nil {
		issues = append(issues, fmt.Sprintf("memory fuzz summary.md missing or unreadable: %v", err))
	} else {
		text := string(summaryMD)
		for _, want := range []string{"Memory Fuzz Short Summary", "Tier 1", "memory-fuzz-oracle.json"} {
			if !strings.Contains(text, want) {
				issues = append(issues, fmt.Sprintf("memory fuzz summary.md missing %q", want))
			}
		}
	}

	var summary memory100MemoryFuzzSummary
	if err := readMemory100StrictJSON(filepath.Join(dir, "summary.json"), &summary); err != nil {
		issues = append(issues, fmt.Sprintf("memory fuzz summary.json invalid: %v", err))
	} else {
		issues = append(issues, validateMemory100MemoryFuzzSummary(summary, gitHead, dir)...)
	}

	var proofSummary memory100IslandProofFuzzSummary
	if err := readMemory100StrictJSON(filepath.Join(dir, "island-proof-fuzz-summary.json"), &proofSummary); err != nil {
		issues = append(issues, fmt.Sprintf("memory fuzz island-proof-fuzz-summary.json invalid: %v", err))
	} else {
		issues = append(issues, validateMemory100IslandProofFuzzSummary(proofSummary)...)
	}

	issues = append(issues, validateMemory100MemoryFuzzHashManifest(filepath.Join(dir, "artifact-hashes.json"), dir)...)
	return issues
}

func requireMemory100MemoryFuzzFile(dir string, rel string) []string {
	path := filepath.Join(dir, filepath.FromSlash(rel))
	info, err := os.Lstat(path)
	if err != nil {
		return []string{fmt.Sprintf("memory fuzz artifact %s is missing: %v", rel, err)}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return []string{fmt.Sprintf("memory fuzz artifact %s must not be a symlink", rel)}
	}
	if !info.Mode().IsRegular() {
		return []string{fmt.Sprintf("memory fuzz artifact %s is not a regular file", rel)}
	}
	if info.Size() == 0 {
		return []string{fmt.Sprintf("memory fuzz artifact %s is empty", rel)}
	}
	return nil
}

func requireMemory100MemoryFuzzDir(dir string, rel string) []string {
	if err := validateMemory100SafeRel(rel); err != nil {
		return []string{fmt.Sprintf("memory fuzz artifact dir %s path invalid: %v", rel, err)}
	}
	path := filepath.Join(dir, filepath.FromSlash(rel))
	info, err := os.Lstat(path)
	if err != nil {
		return []string{fmt.Sprintf("memory fuzz artifact dir %s is missing: %v", rel, err)}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return []string{fmt.Sprintf("memory fuzz artifact dir %s must not be a symlink", rel)}
	}
	if !info.IsDir() {
		return []string{fmt.Sprintf("memory fuzz artifact dir %s is not a directory", rel)}
	}
	return nil
}

func validateMemory100MemoryFuzzSummary(summary memory100MemoryFuzzSummary, gitHead string, artifactDir string) []string {
	var issues []string
	artifactDirs := memory100EquivalentPathForms(artifactDir)
	artifactDirLabel := artifactDirs[0]
	var oraclePaths []string
	for _, dir := range artifactDirs {
		oraclePaths = append(oraclePaths, dir+"/memory-fuzz-oracle.json")
	}
	if summary.SchemaVersion != "tetra.memory-fuzz-short.summary.v1" {
		issues = append(issues, fmt.Sprintf("memory fuzz summary.json schema_version is %q, want tetra.memory-fuzz-short.summary.v1", summary.SchemaVersion))
	}
	if summary.Kind != "tier1_short_ci_smoke" || summary.Tier != "tier1_short_ci_smoke" || summary.Status != "pass" {
		issues = append(issues, fmt.Sprintf("memory fuzz summary.json identity/status must record passing Tier 1 short CI smoke, got kind=%q tier=%q status=%q", summary.Kind, summary.Tier, summary.Status))
	}
	issues = append(issues, validateMemory100MemoryFuzzFailureClassification(summary)...)
	issues = append(issues, validateMemory100MemoryFuzzReproducibilitySeeds(summary.ReproducibilitySeeds)...)
	for key, want := range map[string]string{
		"artifact_hashes":           "artifact-hashes.json",
		"island_proof_fuzz_summary": "island-proof-fuzz-summary.json",
		"oracle_report":             "memory-fuzz-oracle.json",
		"summary_md":                "summary.md",
		"summary_json":              "summary.json",
	} {
		got := summary.Artifacts[key]
		if got != want {
			issues = append(issues, fmt.Sprintf("memory fuzz summary.json artifact %s is %q, want %q", key, got, want))
		}
		if strings.TrimSpace(got) != "" {
			if err := validateMemory100SafeRel(got); err != nil {
				issues = append(issues, fmt.Sprintf("memory fuzz summary.json artifact %s path invalid: %v", key, err))
			}
		}
	}
	var sawRunner, sawValidator bool
	for _, command := range summary.Commands {
		if command.Status != "pass" {
			issues = append(issues, fmt.Sprintf("memory fuzz summary.json command %s status is %q, want pass", command.Name, command.Status))
		}
		switch command.Name {
		case "memory-fuzz-short":
			if strings.Contains(command.Command, "go run ./tools/cmd/memory-fuzz-short") && memory100CommandContainsAnyPath(command.Command, "--report-dir", artifactDirs) && strings.Contains(command.Command, "--git-head "+gitHead) {
				sawRunner = true
			}
		case "validate-memory-fuzz-oracle":
			if strings.Contains(command.Command, "go run ./tools/cmd/validate-memory-fuzz-oracle") && memory100CommandContainsAnyPath(command.Command, "--report", oraclePaths) && memory100CommandContainsAnyPath(command.Command, "--artifact-dir", artifactDirs) && strings.Contains(command.Command, "--current-git-head "+gitHead) {
				sawValidator = true
			}
		}
	}
	if !sawRunner {
		issues = append(issues, fmt.Sprintf("memory fuzz summary.json missing memory-fuzz-short same-commit command provenance for current artifact dir %s", artifactDirLabel))
	}
	if !sawValidator {
		issues = append(issues, fmt.Sprintf("memory fuzz summary.json missing validate-memory-fuzz-oracle same-commit command provenance for current artifact dir %s", artifactDirLabel))
	}
	return issues
}

func validateMemory100MemoryFuzzReproducibilitySeeds(seeds []string) []string {
	if len(seeds) == 0 {
		return []string{"memory fuzz summary.json reproducibility_seeds are required"}
	}
	var issues []string
	if len(seeds) < 12 {
		issues = append(issues, fmt.Sprintf("memory fuzz summary.json reproducibility_seeds has %d entries, want at least 12 for v0-v11", len(seeds)))
	}
	seen := map[string]bool{}
	for _, seed := range seeds {
		text := strings.TrimSpace(seed)
		if text == "" {
			issues = append(issues, "memory fuzz summary.json reproducibility_seeds contains empty seed")
			continue
		}
		lower := strings.ToLower(text)
		for _, forbidden := range []string{"todo", "placeholder", "fake", "mock"} {
			if strings.Contains(lower, forbidden) {
				issues = append(issues, fmt.Sprintf("memory fuzz summary.json reproducibility_seeds contains forbidden marker %q", forbidden))
			}
		}
		if seen[text] {
			issues = append(issues, fmt.Sprintf("memory fuzz summary.json reproducibility_seeds duplicate seed %q", text))
		}
		seen[text] = true
	}
	joined := "\n" + strings.Join(seeds, "\n") + "\n"
	for i := 0; i < 12; i++ {
		if !strings.Contains(joined, fmt.Sprintf(":v%d:", i)) {
			issues = append(issues, fmt.Sprintf("memory fuzz summary.json reproducibility_seeds missing v%d seed", i))
		}
	}
	return issues
}

func validateMemory100MemoryFuzzFailureClassification(summary memory100MemoryFuzzSummary) []string {
	var issues []string
	counts := []struct {
		name  string
		value *int
	}{
		{name: "observed_failures", value: summary.ObservedFailures},
		{name: "classified_failures", value: summary.ClassifiedFailures},
		{name: "unclassified_failures", value: summary.UnclassifiedFailures},
		{name: "release_blocking_failures", value: summary.ReleaseBlockingFailures},
	}
	values := map[string]int{}
	for _, count := range counts {
		if count.value == nil {
			issues = append(issues, fmt.Sprintf("memory fuzz summary.json %s is required", count.name))
			continue
		}
		if *count.value < 0 {
			issues = append(issues, fmt.Sprintf("memory fuzz summary.json %s is %d, want non-negative", count.name, *count.value))
		}
		values[count.name] = *count.value
	}
	if len(issues) > 0 {
		return issues
	}
	if values["classified_failures"]+values["unclassified_failures"] != values["observed_failures"] {
		issues = append(issues, "memory fuzz summary.json classified_failures + unclassified_failures must equal observed_failures")
	}
	if values["release_blocking_failures"] > values["observed_failures"] {
		issues = append(issues, fmt.Sprintf("memory fuzz summary.json release_blocking_failures is %d, exceeds observed_failures %d", values["release_blocking_failures"], values["observed_failures"]))
	}
	if values["unclassified_failures"] != 0 {
		issues = append(issues, fmt.Sprintf("memory fuzz summary.json unclassified_failures is %d, want 0", values["unclassified_failures"]))
	}
	if summary.Status == "pass" && (values["observed_failures"] != 0 || values["classified_failures"] != 0 || values["release_blocking_failures"] != 0) {
		issues = append(issues, fmt.Sprintf("memory fuzz summary.json passing Tier 1 summary must record zero observed/classified/release_blocking failures, got observed=%d classified=%d release_blocking=%d", values["observed_failures"], values["classified_failures"], values["release_blocking_failures"]))
	}
	return issues
}

func memory100EquivalentPathForms(path string) []string {
	seen := map[string]bool{}
	add := func(value string, out *[]string) {
		value = filepath.ToSlash(filepath.Clean(value))
		if value != "" && !seen[value] {
			seen[value] = true
			*out = append(*out, value)
		}
	}
	var out []string
	add(path, &out)
	if abs, err := filepath.Abs(path); err == nil {
		add(abs, &out)
		if cwd, err := os.Getwd(); err == nil {
			if rel, err := filepath.Rel(cwd, abs); err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
				add(rel, &out)
			}
		}
	}
	return out
}

func memory100CommandContainsAnyPath(command string, flag string, paths []string) bool {
	for _, path := range paths {
		if strings.Contains(command, flag+" "+path) {
			return true
		}
	}
	return false
}

func validateMemory100IslandProofFuzzSummary(summary memory100IslandProofFuzzSummary) []string {
	var issues []string
	if summary.SchemaVersion != "tetra.island-proof-fuzz-summary.v1" {
		issues = append(issues, fmt.Sprintf("memory fuzz island-proof-fuzz-summary.json schema_version is %q, want tetra.island-proof-fuzz-summary.v1", summary.SchemaVersion))
	}
	if summary.Status != "pass" {
		issues = append(issues, fmt.Sprintf("memory fuzz island-proof-fuzz-summary.json status is %q, want pass", summary.Status))
	}
	if summary.Total < 10 {
		issues = append(issues, fmt.Sprintf("memory fuzz island-proof-fuzz-summary.json total is %d, want at least 10", summary.Total))
	}
	if summary.Accepted != 0 || summary.Rejected != summary.Total {
		issues = append(issues, fmt.Sprintf("memory fuzz island-proof-fuzz-summary.json counts total=%d rejected=%d accepted=%d, want all rejected", summary.Total, summary.Rejected, summary.Accepted))
	}
	seen := map[string]bool{}
	for _, c := range summary.Cases {
		if c.Status != "rejected" {
			issues = append(issues, fmt.Sprintf("memory fuzz island proof fuzz case %s status is %q, want rejected", c.Name, c.Status))
		}
		seen[c.Name] = true
	}
	for _, name := range []string{
		"malformed_proof_json",
		"stale_epoch",
		"mismatched_island_id",
		"wrong_base_allocation",
		"broken_dominance",
		"missing_proof_id",
		"wrong_operation",
		"unsafe_unknown_promotion",
		"noalias_broad_proof",
		"storage_heap_fallback",
		"transform_lost_metadata",
	} {
		if !seen[name] {
			issues = append(issues, fmt.Sprintf("memory fuzz island-proof-fuzz-summary.json missing mutation case %s", name))
		}
	}
	return issues
}

func validateMemory100MemoryFuzzHashManifest(hashPath string, dir string) []string {
	var manifest memory100HashManifest
	if err := readMemory100StrictJSON(hashPath, &manifest); err != nil {
		return []string{fmt.Sprintf("memory fuzz artifact-hashes.json missing or invalid: %v", err)}
	}
	var issues []string
	if manifest.Schema != memory100HashSchema {
		issues = append(issues, fmt.Sprintf("memory fuzz artifact-hashes.json schema is %q, want %s", manifest.Schema, memory100HashSchema))
	}
	if manifest.Root != "." {
		issues = append(issues, fmt.Sprintf("memory fuzz artifact-hashes.json root is %q, want .", manifest.Root))
	}
	if len(manifest.Artifacts) == 0 {
		issues = append(issues, "memory fuzz artifact-hashes.json artifacts must not be empty")
	}
	seen := map[string]memory100HashArtifact{}
	lastPath := ""
	for _, artifact := range manifest.Artifacts {
		if err := validateMemory100SafeRel(artifact.Path); err != nil {
			issues = append(issues, fmt.Sprintf("memory fuzz artifact-hashes.json path %q is invalid: %v", artifact.Path, err))
			continue
		}
		if artifact.Path == "artifact-hashes.json" {
			issues = append(issues, "memory fuzz artifact-hashes.json must not list itself")
		}
		if lastPath != "" && artifact.Path < lastPath {
			issues = append(issues, "memory fuzz artifact-hashes.json artifacts must be sorted by path")
		}
		lastPath = artifact.Path
		if _, ok := seen[artifact.Path]; ok {
			issues = append(issues, fmt.Sprintf("duplicate memory fuzz hash entry for %s", artifact.Path))
		}
		seen[artifact.Path] = artifact
		if err := validateMemory100SHA256(artifact.SHA256, artifact.Path); err != nil {
			issues = append(issues, err.Error())
		}
		actual, err := hashMemory100File(dir, artifact.Path)
		if err != nil {
			issues = append(issues, fmt.Sprintf("hash memory fuzz artifact %s: %v", artifact.Path, err))
			continue
		}
		if actual.Size != artifact.Size {
			issues = append(issues, fmt.Sprintf("memory fuzz size mismatch for %s: got %d want %d", artifact.Path, actual.Size, artifact.Size))
		}
		if actual.SHA256 != artifact.SHA256 {
			issues = append(issues, fmt.Sprintf("memory fuzz sha256 mismatch for %s: got %s want %s", artifact.Path, actual.SHA256, artifact.SHA256))
		}
		if actual.Schema != artifact.Schema {
			issues = append(issues, fmt.Sprintf("memory fuzz schema mismatch for %s: got %q want %q", artifact.Path, actual.Schema, artifact.Schema))
		}
	}
	for _, rel := range []string{"memory-fuzz-oracle.json", "summary.md", "summary.json", "island-proof-fuzz-summary.json"} {
		if _, ok := seen[rel]; !ok {
			issues = append(issues, fmt.Sprintf("missing memory fuzz hash manifest entry for %s", rel))
		}
	}
	actualPaths, err := listMemory100ArtifactPaths(dir)
	if err != nil {
		issues = append(issues, fmt.Sprintf("list memory fuzz artifacts: %v", err))
	} else {
		for _, rel := range actualPaths {
			if _, ok := seen[rel]; !ok {
				issues = append(issues, fmt.Sprintf("unlisted memory fuzz artifact %s", rel))
			}
		}
	}
	return issues
}

func validateMemory100ProofTransitionReport(path string, gitHead string) []string {
	var report memory100ProofTransitionReport
	if err := readMemory100JSON(path, &report); err != nil {
		return []string{fmt.Sprintf("proof transition report invalid: %v", err)}
	}
	var issues []string
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("proof transition report status is %q, want pass", report.Status))
	}
	if gitHead != "" && report.GitHead != gitHead {
		issues = append(issues, fmt.Sprintf("proof transition report git_head %s does not match Memory100 git_head %s", report.GitHead, gitHead))
	}
	if len(nonEmptyMemory100Strings(report.NonClaims)) == 0 {
		issues = append(issues, "proof transition report non_claims must not be empty")
	}

	required := map[string]string{
		"stable_hash_semantic_fields":                "invalidated",
		"bounds_proof_preserved_through_translation": "preserved",
		"translation_missing_proof_requires_recheck": "requires_recheck",
		"optimization_invalidates_bounds_proofs":     "invalidated",
		"lowering_refines_bounds_proof_use":          "refined",
		"new_proof_requires_store_reference":         "new",
	}
	seenTransitions := map[string]bool{}
	seenRows := map[string]memory100ProofTransitionRow{}
	for i, row := range report.Rows {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			issues = append(issues, fmt.Sprintf("proof transition row %d missing name", i))
			continue
		}
		if _, ok := seenRows[name]; ok {
			issues = append(issues, fmt.Sprintf("duplicate proof transition row %s", name))
		}
		seenRows[name] = row
		transition := strings.TrimSpace(row.Transition)
		if !memory100KnownProofTransition(transition) {
			issues = append(issues, fmt.Sprintf("proof transition row %s has unknown transition %q", name, row.Transition))
		} else {
			seenTransitions[transition] = true
		}
		if strings.TrimSpace(row.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("proof transition row %s missing evidence", name))
		}
		if len(nonEmptyMemory100Strings(row.SourceArtifacts)) == 0 {
			issues = append(issues, fmt.Sprintf("proof transition row %s missing source_artifacts", name))
		}
		if len(nonEmptyMemory100Strings(row.Tests)) == 0 {
			issues = append(issues, fmt.Sprintf("proof transition row %s missing tests", name))
		}
		switch transition {
		case "preserved", "refined":
			if strings.TrimSpace(row.BeforeArtifact) == "" || strings.TrimSpace(row.AfterArtifact) == "" {
				issues = append(issues, fmt.Sprintf("proof transition row %s transition %s requires before_artifact and after_artifact", name, transition))
			}
		case "invalidated", "requires_recheck":
			action := strings.ToLower(strings.TrimSpace(row.ConsumerAction))
			if !strings.Contains(action, "recheck") && !strings.Contains(action, "block") {
				issues = append(issues, fmt.Sprintf("proof transition row %s transition %s requires consumer_action with recheck or block", name, transition))
			}
		case "new":
			if strings.TrimSpace(row.AfterArtifact) == "" {
				issues = append(issues, fmt.Sprintf("proof transition row %s transition new requires after_artifact", name))
			}
		}
	}
	for _, transition := range []string{"preserved", "refined", "invalidated", "new", "requires_recheck"} {
		if !seenTransitions[transition] {
			issues = append(issues, fmt.Sprintf("proof transition report missing transition %s", transition))
		}
	}
	for name, wantTransition := range required {
		row, ok := seenRows[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("proof transition report missing row %s", name))
			continue
		}
		if row.Transition != wantTransition {
			issues = append(issues, fmt.Sprintf("proof transition row %s transition is %q, want %q", name, row.Transition, wantTransition))
		}
		issues = append(issues, validateMemory100ProofTransitionEvidence(row)...)
	}
	return issues
}

func memory100KnownProofTransition(transition string) bool {
	switch transition {
	case "preserved", "refined", "invalidated", "new", "requires_recheck":
		return true
	default:
		return false
	}
}

func validateMemory100ProofTransitionEvidence(row memory100ProofTransitionRow) []string {
	text := strings.ToLower(strings.Join(append(append([]string{row.Evidence, row.BeforeArtifact, row.AfterArtifact, row.ConsumerAction}, row.SourceArtifacts...), row.Tests...), "\n"))
	wants := map[string][]string{
		"stable_hash_semantic_fields":                {"stablehash", "semantic"},
		"bounds_proof_preserved_through_translation": {"bounds", "proof", "translation"},
		"translation_missing_proof_requires_recheck": {"missing proof", "recheck"},
		"optimization_invalidates_bounds_proofs":     {"invalidat", "bounds"},
		"lowering_refines_bounds_proof_use":          {"lower", "proof"},
		"new_proof_requires_store_reference":         {"proof", "store"},
	}
	var issues []string
	for _, want := range wants[row.Name] {
		if !strings.Contains(text, want) {
			issues = append(issues, fmt.Sprintf("proof transition row %s evidence missing %q", row.Name, want))
		}
	}
	return issues
}

func validateMemory100RuntimeMemoryContractTargetMatrix(path string, gitHead string, targetMatrix []string) []string {
	return validateMemory100RuntimeMemoryContract(path, gitHead, nonEmptyMemory100Strings(targetMatrix))
}

func validateMemory100RuntimeMemoryContract(path string, gitHead string, targetMatrix []string) []string {
	var report memory100RuntimeMemoryContract
	if err := readMemory100JSON(path, &report); err != nil {
		return []string{fmt.Sprintf("runtime memory contract invalid: %v", err)}
	}
	var issues []string
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("runtime memory contract status is %q, want pass", report.Status))
	}
	if gitHead != "" && report.GitHead != gitHead {
		issues = append(issues, fmt.Sprintf("runtime memory contract git_head %s does not match Memory100 git_head %s", report.GitHead, gitHead))
	}
	for _, want := range []string{
		"no all-target memory parity claim",
		"OOM recovery guarantee is not claimed",
		"full stack-overflow protection is not claimed",
		"full allocator-corruption detection proof is not claimed",
		"production actor runtime is not claimed",
	} {
		if !memory100ContainsFold(report.NonClaims, want) {
			issues = append(issues, fmt.Sprintf("runtime memory contract missing non_claim %q", want))
		}
	}

	required := map[string]string{
		"linux-x64":   "production_host_runtime",
		"windows-x64": "host_required_nonclaim",
		"macos-x64":   "host_required_nonclaim",
		"wasm32-wasi": "artifact_runtime_tiered_nonclaim",
		"wasm32-web":  "artifact_runtime_tiered_nonclaim",
		"linux-x86":   "build_lower_only_nonclaim",
		"linux-x32":   "build_lower_only_nonclaim",
	}
	rowsByTarget := map[string]memory100RuntimeMemoryRow{}
	for i, row := range report.Rows {
		target := strings.TrimSpace(row.Target)
		if target == "" {
			issues = append(issues, fmt.Sprintf("runtime memory row %d missing target", i))
			continue
		}
		if _, ok := rowsByTarget[target]; ok {
			issues = append(issues, fmt.Sprintf("duplicate runtime memory target row %s", target))
		}
		rowsByTarget[target] = row
		if strings.TrimSpace(row.RuntimeStatus) == "" || strings.TrimSpace(row.MemoryRun) == "" || strings.TrimSpace(row.MemoryClaimLevel) == "" {
			issues = append(issues, fmt.Sprintf("runtime memory row %s missing runtime_status, memory_run, or memory_claim_level", target))
		}
		if strings.TrimSpace(row.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("runtime memory row %s missing evidence", target))
		}
		if len(nonEmptyMemory100Strings(row.SourceArtifacts)) == 0 {
			issues = append(issues, fmt.Sprintf("runtime memory row %s missing source_artifacts", target))
		}
		if len(nonEmptyMemory100Strings(row.Tests)) == 0 {
			issues = append(issues, fmt.Sprintf("runtime memory row %s missing tests", target))
		}
		if len(nonEmptyMemory100Strings(row.NonClaims)) == 0 {
			issues = append(issues, fmt.Sprintf("runtime memory row %s missing non_claims", target))
		}
		if row.IncludedInMemory100TargetMatrix {
			if row.MemoryClaimLevel != "production_host_runtime" {
				issues = append(issues, fmt.Sprintf("runtime memory row %s is included but claim level is %q, want production_host_runtime", target, row.MemoryClaimLevel))
			}
			if row.RuntimeStatus != "production" || row.MemoryRun != "yes" {
				issues = append(issues, fmt.Sprintf("runtime memory row %s is included but runtime_status=%q memory_run=%q, want production/yes", target, row.RuntimeStatus, row.MemoryRun))
			}
			if !memory100ContainsFold(append(append([]string{row.Evidence}, row.SourceArtifacts...), row.Tests...), "runtime hardening") {
				issues = append(issues, fmt.Sprintf("runtime memory row %s included production evidence must mention runtime hardening", target))
			}
			if !memory100ContainsFold(append(append([]string{row.Evidence}, row.SourceArtifacts...), row.Tests...), "runtimeabi") {
				issues = append(issues, fmt.Sprintf("runtime memory row %s included production evidence must mention runtimeabi", target))
			}
		} else {
			if strings.TrimSpace(row.ExcludedReason) == "" {
				issues = append(issues, fmt.Sprintf("runtime memory row %s is excluded but missing excluded_reason", target))
			}
			if row.MemoryClaimLevel == "production_host_runtime" {
				issues = append(issues, fmt.Sprintf("runtime memory row %s is excluded but claims production_host_runtime", target))
			}
		}
	}
	for target, wantClaim := range required {
		row, ok := rowsByTarget[target]
		if !ok {
			issues = append(issues, fmt.Sprintf("runtime memory contract missing target row %s", target))
			continue
		}
		if row.MemoryClaimLevel != wantClaim {
			issues = append(issues, fmt.Sprintf("runtime memory row %s claim level is %q, want %q", target, row.MemoryClaimLevel, wantClaim))
		}
		if target == "linux-x64" && !row.IncludedInMemory100TargetMatrix {
			issues = append(issues, "runtime memory row linux-x64 must be included in Memory100 target matrix")
		}
		if target != "linux-x64" && row.IncludedInMemory100TargetMatrix {
			issues = append(issues, fmt.Sprintf("runtime memory row %s must not be included in Memory100 target matrix without target-host evidence", target))
		}
	}
	if len(targetMatrix) > 0 {
		included := map[string]struct{}{}
		for _, row := range rowsByTarget {
			if row.IncludedInMemory100TargetMatrix {
				included[row.Target] = struct{}{}
			}
		}
		matrixSet := memory100StringSet(targetMatrix)
		for _, missing := range memory100MissingSetKeys(matrixSet, included) {
			issues = append(issues, fmt.Sprintf("Memory100 target_matrix target %s missing included runtime memory row", missing))
		}
		for _, extra := range memory100MissingSetKeys(included, matrixSet) {
			issues = append(issues, fmt.Sprintf("runtime memory included target %s missing from Memory100 target_matrix", extra))
		}
	}
	return issues
}

func memory100ContainsFold(values []string, want string) bool {
	want = strings.ToLower(want)
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), want) {
			return true
		}
	}
	return false
}

func validateMemory100LeakResource(path string, gitHead string) []string {
	var report struct {
		Status  string `json:"status"`
		GitHead string `json:"git_head"`
		Checks  []struct {
			Name            string   `json:"name"`
			Kind            string   `json:"kind"`
			Evidence        string   `json:"evidence"`
			SourceArtifacts []string `json:"source_artifacts"`
		} `json:"checks"`
	}
	if err := readMemory100JSON(path, &report); err != nil {
		return []string{fmt.Sprintf("leak/resource report invalid: %v", err)}
	}
	var issues []string
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("leak/resource report status is %q, want pass", report.Status))
	}
	if gitHead != "" && report.GitHead != gitHead {
		issues = append(issues, fmt.Sprintf("leak/resource report git_head %s does not match Memory100 git_head %s", report.GitHead, gitHead))
	}
	byName := map[string]struct {
		Kind            string
		Evidence        string
		SourceArtifacts []string
	}{}
	for _, check := range report.Checks {
		byName[check.Name] = struct {
			Kind            string
			Evidence        string
			SourceArtifacts []string
		}{
			Kind:            check.Kind,
			Evidence:        check.Evidence,
			SourceArtifacts: check.SourceArtifacts,
		}
	}
	for _, name := range []string{"actornet_close_without_cancel", "compiler_resource_finalization", "surface_frame_escape", "actor_task_transfer"} {
		check, ok := byName[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("leak/resource report missing check %s", name))
			continue
		}
		if strings.TrimSpace(check.Kind) == "" || strings.TrimSpace(check.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("leak/resource report check %s missing kind or evidence", name))
		}
		if len(nonEmptyMemory100Strings(check.SourceArtifacts)) == 0 {
			issues = append(issues, fmt.Sprintf("leak/resource report check %s missing source_artifacts", name))
		}
	}
	return issues
}

func validateMemory100SemanticSafetyMatrix(path string, gitHead string) []string {
	var report struct {
		Status  string `json:"status"`
		GitHead string `json:"git_head"`
		Rows    []struct {
			Name            string   `json:"name"`
			Kind            string   `json:"kind"`
			Evidence        string   `json:"evidence"`
			SourceArtifacts []string `json:"source_artifacts"`
			Tests           []string `json:"tests"`
		} `json:"rows"`
		NonClaims []string `json:"non_claims"`
	}
	if err := readMemory100JSON(path, &report); err != nil {
		return []string{fmt.Sprintf("semantic safety matrix invalid: %v", err)}
	}
	var issues []string
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("semantic safety matrix status is %q, want pass", report.Status))
	}
	if gitHead != "" && report.GitHead != gitHead {
		issues = append(issues, fmt.Sprintf("semantic safety matrix git_head %s does not match Memory100 git_head %s", report.GitHead, gitHead))
	}
	byName := map[string]struct {
		Kind            string
		Evidence        string
		SourceArtifacts []string
		Tests           []string
	}{}
	for _, row := range report.Rows {
		byName[row.Name] = struct {
			Kind            string
			Evidence        string
			SourceArtifacts []string
			Tests           []string
		}{
			Kind:            row.Kind,
			Evidence:        row.Evidence,
			SourceArtifacts: row.SourceArtifacts,
			Tests:           row.Tests,
		}
	}
	required := []string{
		"borrowed_view_return_escape",
		"borrowed_view_owned_aggregate_escape",
		"borrowed_text_host_boundary_copy",
		"inout_alias_escape",
		"surface_frame_escape",
		"use_after_present_close",
		"resource_finalizer_double_close",
		"actor_task_non_sendable_transfer",
	}
	for _, name := range required {
		row, ok := byName[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("semantic safety matrix missing row %s", name))
			continue
		}
		if strings.TrimSpace(row.Kind) == "" || strings.TrimSpace(row.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("semantic safety matrix row %s missing kind or evidence", name))
		}
		if len(nonEmptyMemory100Strings(row.SourceArtifacts)) == 0 {
			issues = append(issues, fmt.Sprintf("semantic safety matrix row %s missing source_artifacts", name))
		}
		if len(nonEmptyMemory100Strings(row.Tests)) == 0 {
			issues = append(issues, fmt.Sprintf("semantic safety matrix row %s missing tests", name))
		}
	}
	if len(nonEmptyMemory100Strings(report.NonClaims)) == 0 {
		issues = append(issues, "semantic safety matrix missing non_claims")
	}
	return issues
}

func validateMemory100ClaimPolicyArtifact(path string, gitHead string) []string {
	var policy struct {
		Status          string   `json:"status"`
		GitHead         string   `json:"git_head"`
		AllowedClaims   []string `json:"allowed_claims"`
		ForbiddenClaims []string `json:"forbidden_claims"`
		NonClaims       []string `json:"non_claims"`
	}
	if err := readMemory100JSON(path, &policy); err != nil {
		return []string{fmt.Sprintf("claim policy invalid: %v", err)}
	}
	var issues []string
	if policy.Status != "pass" {
		issues = append(issues, fmt.Sprintf("claim policy status is %q, want pass", policy.Status))
	}
	if gitHead != "" && policy.GitHead != gitHead {
		issues = append(issues, fmt.Sprintf("claim policy git_head %s does not match Memory100 git_head %s", policy.GitHead, gitHead))
	}
	if len(nonEmptyMemory100Strings(policy.AllowedClaims)) == 0 {
		issues = append(issues, "claim policy allowed_claims must not be empty")
	}
	forbidden := nonEmptyMemory100Strings(policy.ForbiddenClaims)
	if len(forbidden) == 0 {
		issues = append(issues, "claim policy forbidden_claims must not be empty")
	}
	for _, want := range []string{
		"Memory is 100% ready",
		"fully proven memory safety",
		"full formal proof of memory safety",
		"all targets memory-stable",
		"all-target memory parity",
		"unsafe/raw memory is safe",
		"no leaks",
	} {
		if !memory100StringSetContains(forbidden, want) {
			issues = append(issues, fmt.Sprintf("claim policy forbidden_claims missing %q", want))
		}
	}
	if len(nonEmptyMemory100Strings(policy.NonClaims)) == 0 {
		issues = append(issues, "claim policy non_claims must not be empty")
	}
	return issues
}

func nonEmptyMemory100Strings(values []string) []string {
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func memory100StringSetContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func memory100GitStatusSnapshotDirty(lines []string) bool {
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "## ") {
			continue
		}
		return true
	}
	return false
}

func memory100VerdictClaimsClean(verdict string) bool {
	upper := strings.ToUpper(strings.TrimSpace(verdict))
	for _, marker := range []string{
		"CLEAN",
		"RELEASE_CANDIDATE",
		"PROD_READY_PROVEN",
		"RAW_ACCEPTED_PROVEN_PROD_STABLE_100_PERC",
	} {
		if strings.Contains(upper, marker) {
			return true
		}
	}
	return false
}

func validateMemory100HashManifest(hashPath string, reportDir string) []string {
	var manifest memory100HashManifest
	if err := readMemory100StrictJSON(hashPath, &manifest); err != nil {
		return []string{fmt.Sprintf("Memory100 hash manifest missing or invalid: %v", err)}
	}
	var issues []string
	if manifest.Schema != memory100HashSchema {
		issues = append(issues, fmt.Sprintf("Memory100 hash manifest schema is %q, want %s", manifest.Schema, memory100HashSchema))
	}
	if manifest.Root != "." {
		issues = append(issues, fmt.Sprintf("Memory100 hash manifest root is %q, want .", manifest.Root))
	}
	if len(manifest.Artifacts) == 0 {
		issues = append(issues, "Memory100 hash manifest artifacts must not be empty")
	}
	seen := map[string]memory100HashArtifact{}
	lastPath := ""
	for _, artifact := range manifest.Artifacts {
		if err := validateMemory100SafeRel(artifact.Path); err != nil {
			issues = append(issues, fmt.Sprintf("Memory100 hash path %q is invalid: %v", artifact.Path, err))
			continue
		}
		if artifact.Path == "artifact-hashes.json" {
			issues = append(issues, "Memory100 hash manifest must not list itself")
			continue
		}
		if lastPath != "" && artifact.Path < lastPath {
			issues = append(issues, "Memory100 hash manifest artifacts must be sorted by path")
		}
		lastPath = artifact.Path
		if _, ok := seen[artifact.Path]; ok {
			issues = append(issues, fmt.Sprintf("duplicate Memory100 hash entry for %s", artifact.Path))
		}
		seen[artifact.Path] = artifact
		if err := validateMemory100SHA256(artifact.SHA256, artifact.Path); err != nil {
			issues = append(issues, err.Error())
		}
		actual, err := hashMemory100File(reportDir, artifact.Path)
		if err != nil {
			issues = append(issues, fmt.Sprintf("hash Memory100 artifact %s: %v", artifact.Path, err))
			continue
		}
		if actual.Size != artifact.Size {
			issues = append(issues, fmt.Sprintf("size mismatch for %s: got %d want %d", artifact.Path, actual.Size, artifact.Size))
		}
		if actual.SHA256 != artifact.SHA256 {
			issues = append(issues, fmt.Sprintf("sha256 mismatch for %s: got %s want %s", artifact.Path, actual.SHA256, artifact.SHA256))
		}
		if actual.Schema != artifact.Schema {
			issues = append(issues, fmt.Sprintf("schema mismatch for %s: got %q want %q", artifact.Path, actual.Schema, artifact.Schema))
		}
	}
	requiredHashPaths := map[string]bool{"memory-100-prod-stable-manifest.json": true}
	for _, required := range requiredMemory100Artifacts {
		requiredHashPaths[required.Path] = true
	}
	for _, rel := range sortedMemory100Keys(requiredHashPaths) {
		if _, ok := seen[rel]; !ok {
			issues = append(issues, fmt.Sprintf("missing Memory100 hash manifest entry for %s", rel))
		}
	}
	actualPaths, err := listMemory100ArtifactPaths(reportDir)
	if err != nil {
		issues = append(issues, fmt.Sprintf("list Memory100 artifacts: %v", err))
	} else {
		for _, rel := range actualPaths {
			if _, ok := seen[rel]; !ok {
				issues = append(issues, fmt.Sprintf("unlisted Memory100 artifact %s", rel))
			}
		}
	}
	return issues
}

func validateMemory100Claims(label string, values []string, allowNegated bool) []string {
	var issues []string
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			issues = append(issues, fmt.Sprintf("%s contains empty claim", label))
			continue
		}
		if memory100ContainsForbiddenClaim(value, allowNegated) {
			issues = append(issues, fmt.Sprintf("%s contains forbidden Memory100 claim: %q", label, value))
		}
	}
	return issues
}

func memory100ContainsForbiddenClaim(value string, allowNegated bool) bool {
	lower := strings.ToLower(value)
	if allowNegated && memory100HasNegation(lower) {
		return false
	}
	for _, phrase := range []string{
		"memory is 100% ready",
		"memory 100% ready",
		"memory is perfect",
		"fully proven memory safety",
		"full formal proof",
		"all targets memory-stable",
		"all targets memory stable",
		"unsafe/raw memory is safe",
		"unsafe memory is safe",
		"raw memory is safe",
		"zero heap for all programs",
		"zero-copy for all programs",
		"zero copy for all programs",
		"production actor runtime",
		"c/rust parity",
		"faster than c",
		"faster than rust",
		"official benchmark result",
		"release accepted",
	} {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

func memory100HasNegation(lower string) bool {
	for _, marker := range []string{"no ", "not ", "does not ", "without ", "nonclaim", "non-claim"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func validateMemory100VerdictDirtyTier(verdict string, dirty bool) []string {
	verdict = strings.TrimSpace(verdict)
	if dirty {
		if verdict != memory100ScopedReadyDirty {
			return []string{fmt.Sprintf("dirty Memory100 manifest verdict is %q, want %s", verdict, memory100ScopedReadyDirty)}
		}
		return nil
	}
	if verdict == memory100ScopedReadyDirty {
		return []string{fmt.Sprintf("clean Memory100 manifest verdict is %q, want %s or a higher clean evidence tier", verdict, memory100ScopedReadyLocal)}
	}
	return nil
}

func readMemory100StrictJSON(path string, out any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("%s must contain a single JSON document", path)
	}
	return nil
}

func readMemory100JSON(path string, out any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func validateMemory100SafeRel(rel string) error {
	if strings.TrimSpace(rel) == "" {
		return fmt.Errorf("path is required")
	}
	if filepath.IsAbs(rel) {
		return fmt.Errorf("absolute paths are not allowed")
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))
	if clean == "." || clean != rel || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return fmt.Errorf("path must be clean and stay under report root")
	}
	return nil
}

func validateMemory100SHA256(value string, path string) error {
	if !strings.HasPrefix(value, "sha256:") {
		return fmt.Errorf("Memory100 artifact %s has invalid sha256 format %q", path, value)
	}
	hexPart := strings.TrimPrefix(value, "sha256:")
	if len(hexPart) != 64 {
		return fmt.Errorf("Memory100 artifact %s sha256 must contain 64 hex chars", path)
	}
	for _, ch := range hexPart {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return fmt.Errorf("Memory100 artifact %s sha256 has non-hex character %q", path, ch)
		}
	}
	return nil
}

func hashMemory100File(root string, rel string) (memory100HashArtifact, error) {
	if err := validateMemory100SafeRel(rel); err != nil {
		return memory100HashArtifact{}, err
	}
	path := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Lstat(path)
	if err != nil {
		return memory100HashArtifact{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return memory100HashArtifact{}, fmt.Errorf("symlink artifact is not allowed")
	}
	if !info.Mode().IsRegular() {
		return memory100HashArtifact{}, fmt.Errorf("artifact is not a regular file")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return memory100HashArtifact{}, err
	}
	sum := sha256.Sum256(raw)
	return memory100HashArtifact{
		Path:   rel,
		SHA256: "sha256:" + hex.EncodeToString(sum[:]),
		Size:   int64(len(raw)),
		Schema: detectMemory100JSONSchema(raw),
	}, nil
}

func detectMemory100JSONSchema(raw []byte) string {
	var envelope memory100SchemaEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return ""
	}
	return memory100SchemaOf(envelope)
}

func memory100SchemaOf(envelope memory100SchemaEnvelope) string {
	if envelope.Schema != "" {
		return envelope.Schema
	}
	return envelope.SchemaVersion
}

func isMemory100GitHead(value string) bool {
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

func sortedMemory100Keys(values map[string]bool) []string {
	var out []string
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func listMemory100ArtifactPaths(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink artifact %s is not allowed", path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "artifact-hashes.json" {
			return nil
		}
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}
