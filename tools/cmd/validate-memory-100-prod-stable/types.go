package main

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
