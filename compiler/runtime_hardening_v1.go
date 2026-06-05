package compiler

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"tetra_language/compiler/internal/actorsrt"
	"tetra_language/compiler/internal/httprt"
	"tetra_language/compiler/internal/parallelrt"
	"tetra_language/compiler/internal/pgrt"
	"tetra_language/compiler/internal/runtimeabi"
)

const (
	runtimeHardeningV1Schema    = "tetra.runtime.hardening.v1"
	runtimeHardeningV1ScopeP241 = "p24.1_runtime_hardening"

	p24RuntimeHardeningTrapWitnessID       = "deterministic_trap_surface"
	p24RuntimeHardeningAllocationWitnessID = "allocation_failure_surface"
	p24RuntimeHardeningStackWitnessID      = "stack_overflow_boundary"
	p24RuntimeHardeningOverflowWitnessID   = "integer_overflow_semantics"
	p24RuntimeHardeningCorruptionWitnessID = "allocator_corruption_instrumentation"
	p24RuntimeHardeningRegionWitnessID     = "region_lifetime_instrumentation"
	p24RuntimeHardeningMailboxWitnessID    = "actor_mailbox_overflow_policy"
	p24RuntimeHardeningParserWitnessID     = "network_parser_limits"
	p24RuntimeHardeningArtifactsWitnessID  = "runtime_hardening_artifacts"
)

type RuntimeHardeningV1ID string

const (
	RuntimeHardeningDeterministicTraps                 RuntimeHardeningV1ID = "deterministic_traps"
	RuntimeHardeningOOMPolicy                          RuntimeHardeningV1ID = "oom_policy"
	RuntimeHardeningStackOverflowGuard                 RuntimeHardeningV1ID = "stack_overflow_guard"
	RuntimeHardeningIntegerOverflowSemantics           RuntimeHardeningV1ID = "integer_overflow_semantics_audit"
	RuntimeHardeningAllocatorCorruptionInstrumentation RuntimeHardeningV1ID = "allocator_corruption_detection"
	RuntimeHardeningRegionUseAfterFreeInstrumentation  RuntimeHardeningV1ID = "region_double_free_use_after_free"
	RuntimeHardeningActorMailboxOverflowPolicy         RuntimeHardeningV1ID = "actor_mailbox_overflow_policy"
	RuntimeHardeningNetworkParserLimits                RuntimeHardeningV1ID = "network_parser_limits"
)

type RuntimeHardeningV1Report struct {
	SchemaVersion string                      `json:"schema_version"`
	Scope         string                      `json:"scope"`
	Rows          []RuntimeHardeningV1Row     `json:"rows"`
	Witnesses     []RuntimeHardeningV1Witness `json:"witnesses"`
	Artifacts     []RuntimeHardeningArtifact  `json:"artifacts"`
	NonClaims     []string                    `json:"non_claims"`

	DeterministicTrapsReviewed                 bool `json:"deterministic_traps_reviewed"`
	OOMPolicyReviewed                          bool `json:"oom_policy_reviewed"`
	StackOverflowGuardReviewed                 bool `json:"stack_overflow_guard_reviewed"`
	IntegerOverflowSemanticsAudited            bool `json:"integer_overflow_semantics_audited"`
	AllocatorCorruptionInstrumentationReviewed bool `json:"allocator_corruption_instrumentation_reviewed"`
	RegionDoubleFreeUseAfterFreeReviewed       bool `json:"region_double_free_use_after_free_reviewed"`
	ActorMailboxOverflowPolicyReviewed         bool `json:"actor_mailbox_overflow_policy_reviewed"`
	NetworkParserLimitsReviewed                bool `json:"network_parser_limits_reviewed"`
	RuntimeHardeningArtifactPresent            bool `json:"runtime_hardening_artifact_present"`
	RuntimeHardeningDesignArtifactPresent      bool `json:"runtime_hardening_design_artifact_present"`
	FullRuntimeHardeningClaimed                bool `json:"full_runtime_hardening_claimed"`
	FullStackOverflowProtectionClaimed         bool `json:"full_stack_overflow_protection_claimed"`
	FullOOMRecoveryClaimed                     bool `json:"full_oom_recovery_claimed"`
	FullAllocatorCorruptionDetectionClaimed    bool `json:"full_allocator_corruption_detection_claimed"`
	ProductionActorMailboxClaimed              bool `json:"production_actor_mailbox_claimed"`
	RuntimeBehaviorChanged                     bool `json:"runtime_behavior_changed"`
	SafeSemanticsChanged                       bool `json:"safe_semantics_changed"`
	PerformanceClaimed                         bool `json:"performance_claimed"`
}

type RuntimeHardeningV1Row struct {
	ID         RuntimeHardeningV1ID `json:"id"`
	Name       string               `json:"name"`
	Status     string               `json:"status"`
	Evidence   []string             `json:"evidence"`
	Tests      []string             `json:"tests"`
	Boundaries []string             `json:"boundaries"`
	WitnessIDs []string             `json:"witness_ids"`
}

type RuntimeHardeningArtifact struct {
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Present bool   `json:"present"`
}

type RuntimeHardeningV1Witness struct {
	ID                                         string   `json:"id"`
	Kind                                       string   `json:"kind"`
	Paths                                      []string `json:"paths,omitempty"`
	TrapPolicyReviewed                         bool     `json:"trap_policy_reviewed,omitempty"`
	WasmTrapEmitters                           int      `json:"wasm_trap_emitters,omitempty"`
	PanicImportPresent                         bool     `json:"panic_import_present,omitempty"`
	AllocationContracts                        int      `json:"allocation_contracts,omitempty"`
	ContractsWithFailureBehavior               int      `json:"contracts_with_failure_behavior,omitempty"`
	ContractsWithOverflowGuards                int      `json:"contracts_with_overflow_guards,omitempty"`
	OOMPolicyReviewed                          bool     `json:"oom_policy_reviewed,omitempty"`
	StackDepthChecks                           int      `json:"stack_depth_checks,omitempty"`
	StackOverflowGuardReviewed                 bool     `json:"stack_overflow_guard_reviewed,omitempty"`
	FullStackOverflowProtectionClaimed         bool     `json:"full_stack_overflow_protection_claimed,omitempty"`
	CheckedNegI32Present                       bool     `json:"checked_neg_i32_present,omitempty"`
	FoldConstBinaryI32Present                  bool     `json:"fold_const_binary_i32_present,omitempty"`
	ConstOverflowDiagnosticPresent             bool     `json:"const_overflow_diagnostic_present,omitempty"`
	AllocationOverflowGuards                   int      `json:"allocation_overflow_guards,omitempty"`
	IntegerOverflowSemanticsAudited            bool     `json:"integer_overflow_semantics_audited,omitempty"`
	BoundsHeaderContracts                      int      `json:"bounds_header_contracts,omitempty"`
	SmallHeapDoubleFreeRejected                bool     `json:"small_heap_double_free_rejected,omitempty"`
	RawPointerBoundsMetadataVersion            string   `json:"raw_pointer_bounds_metadata_version,omitempty"`
	AllocatorCorruptionInstrumentationReviewed bool     `json:"allocator_corruption_instrumentation_reviewed,omitempty"`
	RegionDebugHeaderBytes                     int32    `json:"region_debug_header_bytes,omitempty"`
	RegionUseAfterFreeContracts                int      `json:"region_use_after_free_contracts,omitempty"`
	RegionDoubleFreeContracts                  int      `json:"region_double_free_contracts,omitempty"`
	RegionResetContracts                       int      `json:"region_reset_contracts,omitempty"`
	RegionDoubleFreeUseAfterFreeReviewed       bool     `json:"region_double_free_use_after_free_reviewed,omitempty"`
	MailboxCapacity                            int      `json:"mailbox_capacity,omitempty"`
	BackpressureMode                           string   `json:"backpressure_mode,omitempty"`
	MailboxOverflowRejected                    bool     `json:"mailbox_overflow_rejected,omitempty"`
	MailboxFIFOReceive                         bool     `json:"mailbox_fifo_receive,omitempty"`
	ActorBoundaryRows                          int      `json:"actor_boundary_rows,omitempty"`
	BuiltinMessagePoolOverflowChecked          bool     `json:"builtin_message_pool_overflow_checked,omitempty"`
	ActorMailboxOverflowPolicyReviewed         bool     `json:"actor_mailbox_overflow_policy_reviewed,omitempty"`
	HTTPParserLimitsReviewed                   bool     `json:"http_parser_limits_reviewed,omitempty"`
	HTTPRequestViewLimitsReviewed              bool     `json:"http_request_view_limits_reviewed,omitempty"`
	PostgresFrameLimitsReviewed                bool     `json:"postgres_frame_limits_reviewed,omitempty"`
	NetworkParserLimitsReviewed                bool     `json:"network_parser_limits_reviewed,omitempty"`
	RuntimeHardeningArtifactPresent            bool     `json:"runtime_hardening_artifact_present,omitempty"`
	RuntimeHardeningDesignArtifactPresent      bool     `json:"runtime_hardening_design_artifact_present,omitempty"`
}

func BuildP24RuntimeHardeningV1Report() (RuntimeHardeningV1Report, error) {
	trapWitness := buildP24RuntimeHardeningTrapWitness()
	allocationWitness, err := buildP24RuntimeHardeningAllocationWitness()
	if err != nil {
		return RuntimeHardeningV1Report{}, err
	}
	stackWitness := buildP24RuntimeHardeningStackWitness()
	overflowWitness, err := buildP24RuntimeHardeningOverflowWitness()
	if err != nil {
		return RuntimeHardeningV1Report{}, err
	}
	corruptionWitness, err := buildP24RuntimeHardeningCorruptionWitness()
	if err != nil {
		return RuntimeHardeningV1Report{}, err
	}
	regionWitness, err := buildP24RuntimeHardeningRegionWitness()
	if err != nil {
		return RuntimeHardeningV1Report{}, err
	}
	mailboxWitness, err := buildP24RuntimeHardeningMailboxWitness()
	if err != nil {
		return RuntimeHardeningV1Report{}, err
	}
	parserWitness := buildP24RuntimeHardeningParserWitness()
	artifacts := p24RuntimeHardeningArtifacts()
	artifactWitness := buildP24RuntimeHardeningArtifactsWitness(artifacts)

	report := RuntimeHardeningV1Report{
		SchemaVersion: runtimeHardeningV1Schema,
		Scope:         runtimeHardeningV1ScopeP241,
		Witnesses: []RuntimeHardeningV1Witness{
			trapWitness,
			allocationWitness,
			stackWitness,
			overflowWitness,
			corruptionWitness,
			regionWitness,
			mailboxWitness,
			parserWitness,
			artifactWitness,
		},
		Artifacts: artifacts,
		Rows: []RuntimeHardeningV1Row{
			p24RuntimeHardeningRow(RuntimeHardeningDeterministicTraps, "Deterministic traps", "reviewed_current_surface",
				[]string{
					"runtimeabi allocation contracts use trap_or_stable_status for allocation failure behavior and reject invalid sizes before allocator access.",
					"wasm32-wasi and wasm32-web backends contain emitWasmTrapIf deterministic trap emitters, and the web panic import formats tetra panic diagnostics deterministically.",
				},
				[]string{
					"go test ./compiler -run 'P24RuntimeHardening' -count=1",
					"go test ./compiler/internal/runtimeabi -run 'Allocation' -count=1",
					"go test ./compiler/tests/lowering -run 'Wasm|ABI' -count=1",
				},
				[]string{
					"trap review is bounded to current backend/runtime ABI surfaces and does not claim a full trap taxonomy for every target",
					"stable trap/status behavior is policy evidence, not a runtime behavior change",
				},
				[]string{p24RuntimeHardeningTrapWitnessID, p24RuntimeHardeningAllocationWitnessID}),
			p24RuntimeHardeningRow(RuntimeHardeningOOMPolicy, "OOM policy", "reviewed_runtime_contracts",
				[]string{
					"runtimeabi.AllocationFailureTrapOrStatus is required on every RuntimeAllocationContract, including core.alloc_bytes, make_* slices, explicit islands, and region.temp.",
					"negative and overflow lengths reject before allocator access, so OOM policy does not mask invalid preconditions as allocator failure.",
				},
				[]string{
					"go test ./compiler/internal/runtimeabi -run 'RuntimeAllocationContract' -count=1",
					"go test ./compiler -run 'P24RuntimeHardening' -count=1",
				},
				[]string{
					"OOM recovery guarantee is not claimed; the current policy is stable trap/status handling",
					"allocator contracts do not prove every platform-specific OOM path has identical process-level behavior",
				},
				[]string{p24RuntimeHardeningAllocationWitnessID}),
			p24RuntimeHardeningRow(RuntimeHardeningStackOverflowGuard, "Stack overflow guard", "reviewed_boundary_with_blocker",
				[]string{
					"backend stack-depth consistency checks reject malformed wasm/x64 lowering shapes before emitting invalid function bodies.",
					"current evidence records stack-depth consistency, while guard-page or recursion-depth runtime stack overflow protection remains an explicit boundary.",
				},
				[]string{
					"go test ./compiler/internal/backend/x64abi -run 'Stack|ABI' -count=1",
					"go test ./compiler/tests/lowering -run 'ABI|Wasm' -count=1",
					"go test ./compiler -run 'P24RuntimeHardening' -count=1",
				},
				[]string{
					"full stack-overflow protection is not claimed",
					"no guard-page or recursion-depth runtime proof is promoted by this report",
				},
				[]string{p24RuntimeHardeningStackWitnessID}),
			p24RuntimeHardeningRow(RuntimeHardeningIntegerOverflowSemantics, "Integer overflow semantics audit", "reviewed_optimizer_and_allocator_boundary",
				[]string{
					"optimizer coverage keeps overflow-sensitive checkedNegI32 and foldConstBinaryI32 cases unoptimized when the fold would change i32 semantics.",
					"allocation contracts and allocplan evidence reject byte-size overflow before allocation, and global const diagnostics reject overflow in global const expression.",
				},
				[]string{
					"go test ./compiler/internal/opt -run 'CoreOptimization|Scalar|Mem2Reg' -count=1",
					"go test ./compiler/internal/allocplan ./compiler/internal/runtimeabi -run 'Overflow|Allocation' -count=1",
					"go test ./compiler/tests/semantics -run 'Const|FeatureRegistry' -count=1",
				},
				[]string{
					"this is a current optimizer/allocation audit, not a full integer-overflow proof for the whole language",
					"overflow-sensitive rewrites remain rejected instead of normalized into a new runtime behavior",
				},
				[]string{p24RuntimeHardeningOverflowWitnessID, p24RuntimeHardeningAllocationWitnessID}),
			p24RuntimeHardeningRow(RuntimeHardeningAllocatorCorruptionInstrumentation, "Allocator corruption detection instrumentation", "reviewed_runtime_instrumentation",
				[]string{
					"runtimeabi contracts expose bounds_header debug instrumentation for heap allocation roots and raw-pointer-bounds-v1 metadata for checked raw pointer derivation.",
					"runtimeabi.PerCoreSmallHeapAllocator rejects stale or double free handles and records reuse metadata through PerCoreSmallHeapAllocator reports.",
				},
				[]string{
					"go test ./compiler/internal/runtimeabi -run 'RawPointer|SmallHeap|Allocation' -count=1",
					"go test ./compiler -run 'P24RuntimeHardening' -count=1",
				},
				[]string{
					"full allocator-corruption detection proof is not claimed",
					"debug instrumentation evidence is bounded to current runtime ABI models and small-heap stale-handle checks",
				},
				[]string{p24RuntimeHardeningCorruptionWitnessID}),
			p24RuntimeHardeningRow(RuntimeHardeningRegionUseAfterFreeInstrumentation, "Region double-free/use-after-free instrumentation", "reviewed_runtime_instrumentation",
				[]string{
					"RuntimeAllocationContracts include AllocationDebugDoubleFree and AllocationDebugUseAfterFree for explicit island paths, and region.temp includes AllocationDebugUseAfterFree plus AllocationDebugRegionReset instrumentation.",
					"RuntimeRegionAllocatorConfig(true) reserves a debug header and AlignRegionBytes rejects negative and overflow sizes for region payloads.",
				},
				[]string{
					"go test ./compiler/internal/runtimeabi -run 'Region|Allocation' -count=1",
					"go test ./compiler -run 'P24RuntimeHardening' -count=1",
				},
				[]string{
					"region instrumentation does not claim a complete temporal-memory-safety proof",
					"future region double-free runtime execution evidence must remain separate from current ABI instrumentation evidence",
				},
				[]string{p24RuntimeHardeningRegionWitnessID, p24RuntimeHardeningAllocationWitnessID}),
			p24RuntimeHardeningRow(RuntimeHardeningActorMailboxOverflowPolicy, "Actor mailbox overflow policy", "reviewed_boundary_with_blocker",
				[]string{
					"parallelrt.NewTypedMailbox records bounded capacity, blocking_recv_yield backpressure metadata, FIFO receive, and recoverable ErrMailboxFull when the typed mailbox model is full.",
					"actorsrt.ActorRuntimeProductionBoundaryAudit records that built-in message pool overflow is not a checked runtime error, preserving the production actor-runtime blocker.",
				},
				[]string{
					"go test ./compiler/internal/parallelrt ./compiler/internal/actorsrt -run 'Mailbox|ProductionBoundary|SchedulerModel' -count=1",
					"go test ./compiler -run 'P24RuntimeHardening' -count=1",
				},
				[]string{
					"production actor-mailbox promotion is not claimed",
					"typed mailbox model policy is evidence for bounded prototype behavior and does not promote the built-in actor runtime message pool",
				},
				[]string{p24RuntimeHardeningMailboxWitnessID}),
			p24RuntimeHardeningRow(RuntimeHardeningNetworkParserLimits, "Network parser limits", "reviewed_parser_limits",
				[]string{
					"httprt.ParseRequest and ParseRequestView return deterministic ErrHeaderTooLarge, ErrTooManyHeaders, ErrBodyTooLarge, malformed request/header, unsupported version, and unsupported transfer-encoding errors.",
					"pgrt.ReadFrame rejects malformed frame lengths with ErrMalformedFrame and oversized payloads with ErrFrameTooLarge before allocating payload buffers.",
				},
				[]string{
					"go test ./compiler/internal/httprt ./compiler/internal/pgrt -run 'ParseRequest|ReadFrame|RequestView' -count=1",
					"go test ./compiler -run 'P24RuntimeHardening' -count=1",
				},
				[]string{
					"network parser limits are local HTTP/PostgreSQL parser evidence, not a full production network-stack hardening proof",
					"TLS, channel binding, remote deployment, and all-protocol parser hardening remain outside this report",
				},
				[]string{p24RuntimeHardeningParserWitnessID}),
		},
		NonClaims: []string{
			"full runtime-hardening proof is not claimed",
			"full stack-overflow protection is not claimed",
			"OOM recovery guarantee is not claimed",
			"full allocator-corruption detection proof is not claimed",
			"production actor-mailbox promotion is not claimed",
			"runtime behavior does not change",
			"safe-program semantics do not change",
			"no performance claim is made",
		},
		DeterministicTrapsReviewed:                 trapWitness.TrapPolicyReviewed && allocationWitness.OOMPolicyReviewed,
		OOMPolicyReviewed:                          allocationWitness.OOMPolicyReviewed,
		StackOverflowGuardReviewed:                 stackWitness.StackOverflowGuardReviewed,
		IntegerOverflowSemanticsAudited:            overflowWitness.IntegerOverflowSemanticsAudited,
		AllocatorCorruptionInstrumentationReviewed: corruptionWitness.AllocatorCorruptionInstrumentationReviewed,
		RegionDoubleFreeUseAfterFreeReviewed:       regionWitness.RegionDoubleFreeUseAfterFreeReviewed,
		ActorMailboxOverflowPolicyReviewed:         mailboxWitness.ActorMailboxOverflowPolicyReviewed,
		NetworkParserLimitsReviewed:                parserWitness.NetworkParserLimitsReviewed,
		RuntimeHardeningArtifactPresent:            artifactWitness.RuntimeHardeningArtifactPresent,
		RuntimeHardeningDesignArtifactPresent:      artifactWitness.RuntimeHardeningDesignArtifactPresent,
	}
	if err := ValidateP24RuntimeHardeningV1Report(report); err != nil {
		return RuntimeHardeningV1Report{}, err
	}
	return report, nil
}

func ValidateP24RuntimeHardeningV1Report(report RuntimeHardeningV1Report) error {
	if report.SchemaVersion != runtimeHardeningV1Schema {
		return fmt.Errorf("runtime hardening v1: schema_version is %q", report.SchemaVersion)
	}
	if report.Scope != runtimeHardeningV1ScopeP241 {
		return fmt.Errorf("runtime hardening v1: scope is %q", report.Scope)
	}
	if report.FullRuntimeHardeningClaimed {
		return fmt.Errorf("runtime hardening v1: full runtime-hardening claim is forbidden")
	}
	if report.FullStackOverflowProtectionClaimed {
		return fmt.Errorf("runtime hardening v1: full stack-overflow protection claim is forbidden")
	}
	if report.FullOOMRecoveryClaimed {
		return fmt.Errorf("runtime hardening v1: OOM recovery claim is forbidden")
	}
	if report.FullAllocatorCorruptionDetectionClaimed {
		return fmt.Errorf("runtime hardening v1: full allocator-corruption detection claim is forbidden")
	}
	if report.ProductionActorMailboxClaimed {
		return fmt.Errorf("runtime hardening v1: production actor-mailbox claim is forbidden")
	}
	if report.RuntimeBehaviorChanged {
		return fmt.Errorf("runtime hardening v1: runtime behavior change claim is forbidden")
	}
	if report.SafeSemanticsChanged {
		return fmt.Errorf("runtime hardening v1: safe semantics change claim is forbidden")
	}
	if report.PerformanceClaimed {
		return fmt.Errorf("runtime hardening v1: performance claim is forbidden")
	}
	if !report.DeterministicTrapsReviewed {
		return fmt.Errorf("runtime hardening v1: deterministic traps review missing")
	}
	if !report.OOMPolicyReviewed {
		return fmt.Errorf("runtime hardening v1: OOM policy review missing")
	}
	if !report.StackOverflowGuardReviewed {
		return fmt.Errorf("runtime hardening v1: stack overflow guard review missing")
	}
	if !report.IntegerOverflowSemanticsAudited {
		return fmt.Errorf("runtime hardening v1: integer overflow semantics audit missing")
	}
	if !report.AllocatorCorruptionInstrumentationReviewed {
		return fmt.Errorf("runtime hardening v1: allocator corruption instrumentation review missing")
	}
	if !report.RegionDoubleFreeUseAfterFreeReviewed {
		return fmt.Errorf("runtime hardening v1: region double-free/use-after-free review missing")
	}
	if !report.ActorMailboxOverflowPolicyReviewed {
		return fmt.Errorf("runtime hardening v1: actor mailbox overflow policy review missing")
	}
	if !report.NetworkParserLimitsReviewed {
		return fmt.Errorf("runtime hardening v1: network parser limits review missing")
	}
	for _, want := range []string{
		"full runtime-hardening proof is not claimed",
		"full stack-overflow protection is not claimed",
		"OOM recovery guarantee is not claimed",
		"full allocator-corruption detection proof is not claimed",
		"production actor-mailbox promotion is not claimed",
		"runtime behavior does not change",
		"safe-program semantics do not change",
		"no performance claim is made",
	} {
		if !p24RuntimeHardeningHasString(report.NonClaims, want) {
			return fmt.Errorf("runtime hardening v1: missing non-claim %q", want)
		}
	}
	if err := p24RuntimeHardeningValidateArtifacts(report); err != nil {
		return err
	}
	if err := p24RuntimeHardeningValidateRowsAndWitnesses(report.Rows, report.Witnesses); err != nil {
		return err
	}
	return nil
}

func buildP24RuntimeHardeningTrapWitness() RuntimeHardeningV1Witness {
	paths := []string{
		"compiler/internal/backend/wasm32_wasi/codegen.go",
		"compiler/internal/backend/wasm32_web/codegen.go",
		"docs/spec/current_supported_surface.md",
	}
	wasmTrapEmitters := 0
	for _, path := range paths[:2] {
		if p24RuntimeHardeningFileContains(path, "emitWasmTrapIf") {
			wasmTrapEmitters++
		}
	}
	panicImport := p24RuntimeHardeningFileContains("compiler/internal/backend/wasm32_web/codegen.go", "tetra panic")
	return RuntimeHardeningV1Witness{
		ID:                 p24RuntimeHardeningTrapWitnessID,
		Kind:               "deterministic_trap_surface",
		Paths:              paths,
		TrapPolicyReviewed: p24AllRepoPathsExist(paths) && wasmTrapEmitters >= 2 && panicImport,
		WasmTrapEmitters:   wasmTrapEmitters,
		PanicImportPresent: panicImport,
	}
}

func buildP24RuntimeHardeningAllocationWitness() (RuntimeHardeningV1Witness, error) {
	contracts := runtimeabi.RuntimeAllocationContracts()
	var failureBehaviors int
	var overflowGuards int
	for _, contract := range contracts {
		if err := runtimeabi.ValidateRuntimeAllocationContract(contract); err != nil {
			return RuntimeHardeningV1Witness{}, err
		}
		if contract.FailureBehavior == runtimeabi.AllocationFailureTrapOrStatus {
			failureBehaviors++
		}
		if contract.OverflowBehavior != "" {
			overflowGuards++
		}
	}
	return RuntimeHardeningV1Witness{
		ID:                           p24RuntimeHardeningAllocationWitnessID,
		Kind:                         "allocation_failure_surface",
		Paths:                        []string{"compiler/internal/runtimeabi/allocation_contract.go", "docs/design/runtime_allocation_contract.md"},
		AllocationContracts:          len(contracts),
		ContractsWithFailureBehavior: failureBehaviors,
		ContractsWithOverflowGuards:  overflowGuards,
		OOMPolicyReviewed:            len(contracts) >= 5 && failureBehaviors == len(contracts) && overflowGuards == len(contracts),
	}, nil
}

func buildP24RuntimeHardeningStackWitness() RuntimeHardeningV1Witness {
	paths := []string{
		"compiler/internal/backend/wasm32_wasi/codegen.go",
		"compiler/internal/backend/wasm32_web/codegen.go",
		"compiler/internal/backend/x64abi/abi_test.go",
		"compiler/tests/lowering/x64_abi_test.go",
	}
	stackDepthChecks := 0
	for _, path := range paths {
		if p24RuntimeHardeningFileContains(path, "stack depth") {
			stackDepthChecks++
		}
	}
	return RuntimeHardeningV1Witness{
		ID:                                 p24RuntimeHardeningStackWitnessID,
		Kind:                               "stack_overflow_boundary",
		Paths:                              paths,
		StackDepthChecks:                   stackDepthChecks,
		StackOverflowGuardReviewed:         p24AllRepoPathsExist(paths) && stackDepthChecks >= 2,
		FullStackOverflowProtectionClaimed: false,
	}
}

func buildP24RuntimeHardeningOverflowWitness() (RuntimeHardeningV1Witness, error) {
	allocation, err := buildP24RuntimeHardeningAllocationWitness()
	if err != nil {
		return RuntimeHardeningV1Witness{}, err
	}
	checkedNeg := p24RuntimeHardeningFileContains("compiler/internal/opt/scalar.go", "checkedNegI32")
	foldConst := p24RuntimeHardeningFileContains("compiler/internal/opt/scalar.go", "foldConstBinaryI32")
	constOverflow := p24RuntimeHardeningFileContains("compiler/internal/semantics/checker.go", "overflow in global const expression")
	return RuntimeHardeningV1Witness{
		ID:                              p24RuntimeHardeningOverflowWitnessID,
		Kind:                            "integer_overflow_semantics",
		Paths:                           []string{"compiler/internal/opt/scalar.go", "compiler/internal/opt/coverage.go", "compiler/internal/semantics/checker.go", "compiler/internal/allocplan/plan.go", "compiler/internal/runtimeabi/allocation_contract.go"},
		CheckedNegI32Present:            checkedNeg,
		FoldConstBinaryI32Present:       foldConst,
		ConstOverflowDiagnosticPresent:  constOverflow,
		AllocationOverflowGuards:        allocation.ContractsWithOverflowGuards,
		IntegerOverflowSemanticsAudited: checkedNeg && foldConst && constOverflow && allocation.ContractsWithOverflowGuards >= 5,
	}, nil
}

func buildP24RuntimeHardeningCorruptionWitness() (RuntimeHardeningV1Witness, error) {
	contracts := runtimeabi.RuntimeAllocationContracts()
	var boundsHeaderContracts int
	for _, contract := range contracts {
		if err := runtimeabi.ValidateRuntimeAllocationContract(contract); err != nil {
			return RuntimeHardeningV1Witness{}, err
		}
		if contract.HasDebugInstrumentation(runtimeabi.AllocationDebugBoundsHeader) {
			boundsHeaderContracts++
		}
	}
	smallHeapDoubleFreeRejected, err := p24RuntimeHardeningSmallHeapRejectsDoubleFree()
	if err != nil {
		return RuntimeHardeningV1Witness{}, err
	}
	rawBounds := runtimeabi.RuntimeRawPointerBoundsABI()
	return RuntimeHardeningV1Witness{
		ID:                              p24RuntimeHardeningCorruptionWitnessID,
		Kind:                            "allocator_corruption_instrumentation",
		Paths:                           []string{"compiler/internal/runtimeabi/allocation_contract.go", "compiler/internal/runtimeabi/small_heap.go", "compiler/internal/runtimeabi/raw_pointer_bounds.go"},
		BoundsHeaderContracts:           boundsHeaderContracts,
		SmallHeapDoubleFreeRejected:     smallHeapDoubleFreeRejected,
		RawPointerBoundsMetadataVersion: rawBounds.MetadataVersion,
		AllocatorCorruptionInstrumentationReviewed: boundsHeaderContracts >= 1 && smallHeapDoubleFreeRejected && rawBounds.MetadataVersion == "raw-pointer-bounds-v1",
	}, nil
}

func buildP24RuntimeHardeningRegionWitness() (RuntimeHardeningV1Witness, error) {
	contracts := runtimeabi.RuntimeAllocationContracts()
	var useAfterFreeContracts int
	var doubleFreeContracts int
	var regionResetContracts int
	for _, contract := range contracts {
		if err := runtimeabi.ValidateRuntimeAllocationContract(contract); err != nil {
			return RuntimeHardeningV1Witness{}, err
		}
		if contract.HasDebugInstrumentation(runtimeabi.AllocationDebugUseAfterFree) {
			useAfterFreeContracts++
		}
		if contract.HasDebugInstrumentation(runtimeabi.AllocationDebugDoubleFree) {
			doubleFreeContracts++
		}
		if contract.HasDebugInstrumentation(runtimeabi.AllocationDebugRegionReset) {
			regionResetContracts++
		}
	}
	debugCfg := runtimeabi.RuntimeRegionAllocatorConfig(true)
	return RuntimeHardeningV1Witness{
		ID:                                   p24RuntimeHardeningRegionWitnessID,
		Kind:                                 "region_lifetime_instrumentation",
		Paths:                                []string{"compiler/internal/runtimeabi/allocation_contract.go", "compiler/internal/runtimeabi/region_allocator.go"},
		RegionDebugHeaderBytes:               debugCfg.DebugHeaderBytes,
		RegionUseAfterFreeContracts:          useAfterFreeContracts,
		RegionDoubleFreeContracts:            doubleFreeContracts,
		RegionResetContracts:                 regionResetContracts,
		RegionDoubleFreeUseAfterFreeReviewed: debugCfg.DebugHeaderBytes > 0 && useAfterFreeContracts >= 1 && doubleFreeContracts >= 1 && regionResetContracts >= 1,
	}, nil
}

func buildP24RuntimeHardeningMailboxWitness() (RuntimeHardeningV1Witness, error) {
	box := parallelrt.NewTypedMailbox(parallelrt.MailboxConfig{Name: "p24", Capacity: 1})
	if _, err := box.Send(parallelrt.Message{Name: "first"}); err != nil {
		return RuntimeHardeningV1Witness{}, err
	}
	_, overflowErr := box.Send(parallelrt.Message{Name: "second"})
	first, received := box.Receive()
	audit, err := actorsrt.ActorRuntimeProductionBoundaryAudit()
	if err != nil {
		return RuntimeHardeningV1Witness{}, err
	}
	if err := actorsrt.ValidateActorRuntimeProductionBoundaryAudit(audit); err != nil {
		return RuntimeHardeningV1Witness{}, err
	}
	builtinOverflowChecked := true
	for _, row := range audit.Rows {
		text := strings.Join(row.RequiredFacts, " ") + " " + row.Evidence + " " + row.Boundary
		if strings.Contains(text, "message pool overflow is not a checked runtime error") {
			builtinOverflowChecked = false
			break
		}
	}
	return RuntimeHardeningV1Witness{
		ID:                                p24RuntimeHardeningMailboxWitnessID,
		Kind:                              "actor_mailbox_overflow_policy",
		Paths:                             []string{"compiler/internal/parallelrt/scheduler_model.go", "compiler/internal/actorsrt/production_boundary.go"},
		MailboxCapacity:                   box.Capacity(),
		BackpressureMode:                  box.Backpressure().Mode,
		MailboxOverflowRejected:           errors.Is(overflowErr, parallelrt.ErrMailboxFull),
		MailboxFIFOReceive:                received && first.Name == "first",
		ActorBoundaryRows:                 len(audit.Rows),
		BuiltinMessagePoolOverflowChecked: builtinOverflowChecked,
		ActorMailboxOverflowPolicyReviewed: box.Capacity() == 1 &&
			box.Backpressure().Mode == "blocking_recv_yield" &&
			errors.Is(overflowErr, parallelrt.ErrMailboxFull) &&
			received && first.Name == "first" &&
			len(audit.Rows) >= 4 &&
			!builtinOverflowChecked,
	}, nil
}

func buildP24RuntimeHardeningParserWitness() RuntimeHardeningV1Witness {
	httpLimits := httprt.Limits{MaxHeaderBytes: 64, MaxHeaders: 4, MaxBodyBytes: 4}
	_, _, httpHeaderErr := httprt.ParseRequest([]byte("GET / HTTP/1.1\r\nLong: "+strings.Repeat("x", 80)+"\r\n\r\n"), httpLimits)
	_, _, httpBodyErr := httprt.ParseRequest([]byte("POST / HTTP/1.1\r\nContent-Length: 5\r\n\r\nhello"), httpLimits)
	_, _, _, viewHeaderErr := httprt.ParseRequestView([]byte("GET / HTTP/1.1\r\nLong: "+strings.Repeat("x", 80)+"\r\n\r\n"), httpLimits, nil)
	_, pgMalformedErr := pgrt.ReadFrame(bytes.NewReader([]byte{'R', 0, 0, 0, 3}), 1024)
	_, pgLargeErr := pgrt.ReadFrame(bytes.NewReader([]byte{'R', 0, 0, 4, 1}), 8)

	httpReviewed := errors.Is(httpHeaderErr, httprt.ErrHeaderTooLarge) && errors.Is(httpBodyErr, httprt.ErrBodyTooLarge)
	viewReviewed := errors.Is(viewHeaderErr, httprt.ErrHeaderTooLarge)
	pgReviewed := errors.Is(pgMalformedErr, pgrt.ErrMalformedFrame) && errors.Is(pgLargeErr, pgrt.ErrFrameTooLarge)
	return RuntimeHardeningV1Witness{
		ID:                            p24RuntimeHardeningParserWitnessID,
		Kind:                          "network_parser_limits",
		Paths:                         []string{"compiler/internal/httprt/http1.go", "compiler/internal/httprt/request_view.go", "compiler/internal/pgrt/wire.go"},
		HTTPParserLimitsReviewed:      httpReviewed,
		HTTPRequestViewLimitsReviewed: viewReviewed,
		PostgresFrameLimitsReviewed:   pgReviewed,
		NetworkParserLimitsReviewed:   httpReviewed && viewReviewed && pgReviewed,
	}
}

func buildP24RuntimeHardeningArtifactsWitness(artifacts []RuntimeHardeningArtifact) RuntimeHardeningV1Witness {
	witness := RuntimeHardeningV1Witness{
		ID:    p24RuntimeHardeningArtifactsWitnessID,
		Kind:  "runtime_hardening_artifacts",
		Paths: make([]string, 0, len(artifacts)),
	}
	for _, artifact := range artifacts {
		witness.Paths = append(witness.Paths, artifact.Path)
		switch artifact.Path {
		case "docs/audits/runtime-hardening-v1.md":
			witness.RuntimeHardeningArtifactPresent = artifact.Present
		case "docs/plans/2026-06-03-p24.1-runtime-hardening-design.md":
			witness.RuntimeHardeningDesignArtifactPresent = artifact.Present
		}
	}
	return witness
}

func p24RuntimeHardeningValidateRowsAndWitnesses(rows []RuntimeHardeningV1Row, witnesses []RuntimeHardeningV1Witness) error {
	byWitness := map[string]RuntimeHardeningV1Witness{}
	for _, witness := range witnesses {
		if strings.TrimSpace(witness.ID) == "" || strings.TrimSpace(witness.Kind) == "" {
			return fmt.Errorf("runtime hardening v1: witness missing id or kind")
		}
		if _, exists := byWitness[witness.ID]; exists {
			return fmt.Errorf("runtime hardening v1: duplicate witness %q", witness.ID)
		}
		byWitness[witness.ID] = witness
	}
	expected := map[RuntimeHardeningV1ID]bool{}
	for _, id := range p24RuntimeHardeningV1IDs() {
		expected[id] = true
	}
	seen := map[RuntimeHardeningV1ID]bool{}
	for _, row := range rows {
		if !expected[row.ID] {
			return fmt.Errorf("runtime hardening v1: unexpected row %q", row.ID)
		}
		if seen[row.ID] {
			return fmt.Errorf("runtime hardening v1: duplicate row %q", row.ID)
		}
		seen[row.ID] = true
		if strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Status) == "" {
			return fmt.Errorf("runtime hardening v1: row %q missing name or status", row.ID)
		}
		if len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 || len(row.WitnessIDs) == 0 {
			return fmt.Errorf("runtime hardening v1: row %q missing evidence, tests, boundaries, or witness ids", row.ID)
		}
		for _, text := range append(append(append([]string{}, row.Evidence...), row.Tests...), row.Boundaries...) {
			if p24RuntimeHardeningIsPlaceholder(text) {
				return fmt.Errorf("runtime hardening v1: row %q has placeholder evidence", row.ID)
			}
		}
		for _, id := range row.WitnessIDs {
			if _, ok := byWitness[id]; !ok {
				return fmt.Errorf("runtime hardening v1: row %q references missing witness %q", row.ID, id)
			}
		}
	}
	for _, id := range p24RuntimeHardeningV1IDs() {
		if !seen[id] {
			return fmt.Errorf("runtime hardening v1: missing row %q", id)
		}
	}
	if witness := byWitness[p24RuntimeHardeningTrapWitnessID]; !witness.TrapPolicyReviewed || witness.WasmTrapEmitters < 2 || !witness.PanicImportPresent {
		return fmt.Errorf("runtime hardening v1: deterministic trap witness incomplete")
	}
	if witness := byWitness[p24RuntimeHardeningAllocationWitnessID]; !witness.OOMPolicyReviewed || witness.AllocationContracts < 5 || witness.ContractsWithFailureBehavior != witness.AllocationContracts || witness.ContractsWithOverflowGuards != witness.AllocationContracts {
		return fmt.Errorf("runtime hardening v1: allocation/OOM witness incomplete")
	}
	if witness := byWitness[p24RuntimeHardeningStackWitnessID]; !witness.StackOverflowGuardReviewed || witness.StackDepthChecks < 2 || witness.FullStackOverflowProtectionClaimed {
		return fmt.Errorf("runtime hardening v1: stack overflow boundary witness incomplete")
	}
	if witness := byWitness[p24RuntimeHardeningOverflowWitnessID]; !witness.IntegerOverflowSemanticsAudited || !witness.CheckedNegI32Present || !witness.FoldConstBinaryI32Present || !witness.ConstOverflowDiagnosticPresent || witness.AllocationOverflowGuards < 5 {
		return fmt.Errorf("runtime hardening v1: integer overflow semantics witness incomplete")
	}
	if witness := byWitness[p24RuntimeHardeningCorruptionWitnessID]; !witness.AllocatorCorruptionInstrumentationReviewed || witness.BoundsHeaderContracts < 1 || !witness.SmallHeapDoubleFreeRejected || witness.RawPointerBoundsMetadataVersion != "raw-pointer-bounds-v1" {
		return fmt.Errorf("runtime hardening v1: allocator corruption instrumentation witness incomplete")
	}
	if witness := byWitness[p24RuntimeHardeningRegionWitnessID]; !witness.RegionDoubleFreeUseAfterFreeReviewed || witness.RegionDebugHeaderBytes <= 0 || witness.RegionUseAfterFreeContracts < 1 || witness.RegionDoubleFreeContracts < 1 || witness.RegionResetContracts < 1 {
		return fmt.Errorf("runtime hardening v1: region lifetime instrumentation witness incomplete")
	}
	if witness := byWitness[p24RuntimeHardeningMailboxWitnessID]; !witness.ActorMailboxOverflowPolicyReviewed || witness.MailboxCapacity != 1 || witness.BackpressureMode != "blocking_recv_yield" || !witness.MailboxOverflowRejected || !witness.MailboxFIFOReceive || witness.ActorBoundaryRows < 4 || witness.BuiltinMessagePoolOverflowChecked {
		return fmt.Errorf("runtime hardening v1: actor mailbox overflow witness incomplete")
	}
	if witness := byWitness[p24RuntimeHardeningParserWitnessID]; !witness.NetworkParserLimitsReviewed || !witness.HTTPParserLimitsReviewed || !witness.HTTPRequestViewLimitsReviewed || !witness.PostgresFrameLimitsReviewed {
		return fmt.Errorf("runtime hardening v1: network parser limits witness incomplete")
	}
	if witness := byWitness[p24RuntimeHardeningArtifactsWitnessID]; !witness.RuntimeHardeningArtifactPresent || !witness.RuntimeHardeningDesignArtifactPresent {
		return fmt.Errorf("runtime hardening v1: runtime hardening artifact witness incomplete")
	}
	return nil
}

func p24RuntimeHardeningValidateArtifacts(report RuntimeHardeningV1Report) error {
	if !report.RuntimeHardeningArtifactPresent {
		return fmt.Errorf("runtime hardening v1: docs/audits/runtime-hardening-v1.md artifact missing")
	}
	if !report.RuntimeHardeningDesignArtifactPresent {
		return fmt.Errorf("runtime hardening v1: docs/plans/2026-06-03-p24.1-runtime-hardening-design.md artifact missing")
	}
	present := map[string]bool{}
	for _, artifact := range report.Artifacts {
		if strings.TrimSpace(artifact.Kind) == "" || strings.TrimSpace(artifact.Path) == "" {
			return fmt.Errorf("runtime hardening v1: artifact missing kind or path")
		}
		present[artifact.Path] = artifact.Present
	}
	for _, path := range []string{
		"docs/audits/runtime-hardening-v1.md",
		"docs/plans/2026-06-03-p24.1-runtime-hardening-design.md",
	} {
		if !present[path] {
			return fmt.Errorf("runtime hardening v1: required artifact %s missing", path)
		}
	}
	return nil
}

func p24RuntimeHardeningV1IDs() []RuntimeHardeningV1ID {
	return []RuntimeHardeningV1ID{
		RuntimeHardeningDeterministicTraps,
		RuntimeHardeningOOMPolicy,
		RuntimeHardeningStackOverflowGuard,
		RuntimeHardeningIntegerOverflowSemantics,
		RuntimeHardeningAllocatorCorruptionInstrumentation,
		RuntimeHardeningRegionUseAfterFreeInstrumentation,
		RuntimeHardeningActorMailboxOverflowPolicy,
		RuntimeHardeningNetworkParserLimits,
	}
}

func p24RuntimeHardeningRow(id RuntimeHardeningV1ID, name, status string, evidence, tests, boundaries, witnessIDs []string) RuntimeHardeningV1Row {
	return RuntimeHardeningV1Row{
		ID:         id,
		Name:       name,
		Status:     status,
		Evidence:   evidence,
		Tests:      tests,
		Boundaries: boundaries,
		WitnessIDs: witnessIDs,
	}
}

func p24RuntimeHardeningArtifacts() []RuntimeHardeningArtifact {
	return []RuntimeHardeningArtifact{
		p24RuntimeHardeningArtifact("runtime_hardening_audit", "docs/audits/runtime-hardening-v1.md"),
		p24RuntimeHardeningArtifact("runtime_hardening_design", "docs/plans/2026-06-03-p24.1-runtime-hardening-design.md"),
	}
}

func p24RuntimeHardeningArtifact(kind string, rel string) RuntimeHardeningArtifact {
	_, err := os.Stat(p24RepoPath(rel))
	return RuntimeHardeningArtifact{
		Kind:    kind,
		Path:    rel,
		Present: err == nil,
	}
}

func p24RuntimeHardeningSmallHeapRejectsDoubleFree() (bool, error) {
	allocator, err := runtimeabi.NewPerCoreSmallHeapAllocator(runtimeabi.RuntimePerCoreSmallHeapABI(1))
	if err != nil {
		return false, err
	}
	handle, err := allocator.Alloc(0, 17)
	if err != nil {
		return false, err
	}
	if err := allocator.Free(handle); err != nil {
		return false, err
	}
	err = allocator.Free(handle)
	return err != nil && strings.Contains(err.Error(), "stale or double free"), nil
}

func p24RuntimeHardeningFileContains(rel string, want string) bool {
	data, err := os.ReadFile(p24RepoPath(rel))
	return err == nil && strings.Contains(string(data), want)
}

func p24RuntimeHardeningHasString(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

func p24RuntimeHardeningIsPlaceholder(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return lower == "" ||
		lower == "todo" ||
		lower == "tbd" ||
		strings.Contains(lower, "placeholder")
}
