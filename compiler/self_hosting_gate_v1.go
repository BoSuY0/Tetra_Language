package compiler

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/differential"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/runtimeabi"
	"tetra_language/compiler/internal/selfhostgate"
	"tetra_language/compiler/internal/stdlibrt"
)

const (
	selfHostingGateV1Schema    = "tetra.self_hosting.gate.v1"
	selfHostingGateV1ScopeP233 = "p23.3_self_hosting_gate"

	p23SelfHostingSubsetWitnessID            = "self_host_subset_definition"
	p23SelfHostingBootstrapBlockersWitnessID = "self_host_bootstrap_blockers"
	p23SelfHostingRegisterBackendWitnessID   = "register_backend_stability"
	p23SelfHostingOptimizerWitnessID         = "optimizer_validation_maturity"
	p23SelfHostingAllocatorRuntimeWitnessID  = "allocator_runtime_stability"
	p23SelfHostingStdlibWitnessID            = "stdlib_sufficiency"
)

type SelfHostingGateV1ID string

const (
	SelfHostingGateSubsetDefinition       SelfHostingGateV1ID = "self_host_subset_definition"
	SelfHostingGateSmallComponentCompile  SelfHostingGateV1ID = "small_compiler_component_compile"
	SelfHostingGateOutputComparison       SelfHostingGateV1ID = "go_vs_tetra_output_comparison"
	SelfHostingGateRegisterBackend        SelfHostingGateV1ID = "register_backend_stability"
	SelfHostingGateOptimizerValidation    SelfHostingGateV1ID = "optimizer_validation_maturity"
	SelfHostingGateAllocatorRuntime       SelfHostingGateV1ID = "allocator_runtime_stability"
	SelfHostingGateStdlibSufficiency      SelfHostingGateV1ID = "stdlib_sufficiency"
	SelfHostingGateDeterministicBootstrap SelfHostingGateV1ID = "deterministic_bootstrap_chain"
	SelfHostingGateCrossPlatformBootstrap SelfHostingGateV1ID = "cross_platform_bootstrap_story"
	SelfHostingGateNoSelfHostingClaim     SelfHostingGateV1ID = "no_self_hosting_claim"
)

type SelfHostingGateV1Report struct {
	SchemaVersion                      string                     `json:"schema_version"`
	Scope                              string                     `json:"scope"`
	Rows                               []SelfHostingGateV1Row     `json:"rows"`
	Witnesses                          []SelfHostingGateV1Witness `json:"witnesses"`
	NonClaims                          []string                   `json:"non_claims"`
	GateDecision                       selfhostgate.Decision      `json:"gate_decision"`
	CompilerSubsetDefined              bool                       `json:"compiler_subset_defined"`
	SubsetName                         string                     `json:"subset_name"`
	SmallCompilerComponentCompiled     bool                       `json:"small_compiler_component_compiled"`
	GoVsTetraOutputCompared            bool                       `json:"go_vs_tetra_output_compared"`
	RegisterBackendEvidencePresent     bool                       `json:"register_backend_evidence_present"`
	OptimizerValidationEvidencePresent bool                       `json:"optimizer_validation_evidence_present"`
	AllocatorRuntimeEvidencePresent    bool                       `json:"allocator_runtime_evidence_present"`
	StdlibEvidencePresent              bool                       `json:"stdlib_evidence_present"`
	DeterministicBootstrapChain        bool                       `json:"deterministic_bootstrap_chain"`
	CrossPlatformBootstrapStory        bool                       `json:"cross_platform_bootstrap_story"`
	SelfHostingClaimed                 bool                       `json:"self_hosting_claimed"`
	RuntimeBehaviorChanged             bool                       `json:"runtime_behavior_changed"`
	SafeSemanticsChanged               bool                       `json:"safe_semantics_changed"`
	PerformanceClaimed                 bool                       `json:"performance_claimed"`
}

type SelfHostingGateV1Row struct {
	ID         SelfHostingGateV1ID `json:"id"`
	Name       string              `json:"name"`
	Status     string              `json:"status"`
	Evidence   []string            `json:"evidence"`
	Tests      []string            `json:"tests"`
	Boundaries []string            `json:"boundaries"`
	WitnessIDs []string            `json:"witness_ids"`
}

type SelfHostingGateV1Witness struct {
	ID                                 string   `json:"id"`
	Kind                               string   `json:"kind"`
	CompilerSubsetDefined              bool     `json:"compiler_subset_defined,omitempty"`
	SubsetName                         string   `json:"subset_name,omitempty"`
	SmallCompilerComponentCompiled     bool     `json:"small_compiler_component_compiled,omitempty"`
	GoVsTetraOutputCompared            bool     `json:"go_vs_tetra_output_compared,omitempty"`
	DeterministicBootstrapChain        bool     `json:"deterministic_bootstrap_chain,omitempty"`
	CrossPlatformBootstrapStory        bool     `json:"cross_platform_bootstrap_story,omitempty"`
	RegisterBackendEvidencePresent     bool     `json:"register_backend_evidence_present,omitempty"`
	BackendMatrixLanes                 int      `json:"backend_matrix_lanes,omitempty"`
	OptimizerValidationEvidencePresent bool     `json:"optimizer_validation_evidence_present,omitempty"`
	TranslationValidationRows          int      `json:"translation_validation_rows,omitempty"`
	TranslationValidationWitnesses     int      `json:"translation_validation_witnesses,omitempty"`
	AllocatorRuntimeEvidencePresent    bool     `json:"allocator_runtime_evidence_present,omitempty"`
	RuntimeAllocationContracts         int      `json:"runtime_allocation_contracts,omitempty"`
	RegionAllocatorAlignmentBytes      int32    `json:"region_allocator_alignment_bytes,omitempty"`
	PerCoreSmallHeapEvidencePresent    bool     `json:"per_core_small_heap_evidence_present,omitempty"`
	StdlibEvidencePresent              bool     `json:"stdlib_evidence_present,omitempty"`
	StdlibRows                         int      `json:"stdlib_rows,omitempty"`
	Blockers                           []string `json:"blockers,omitempty"`
}

func BuildP23SelfHostingGateV1Report() (SelfHostingGateV1Report, error) {
	subset := buildP23SelfHostingSubsetWitness()
	blockers := buildP23SelfHostingBootstrapBlockersWitness()
	backend, err := buildP23SelfHostingRegisterBackendWitness()
	if err != nil {
		return SelfHostingGateV1Report{}, err
	}
	optimizer, err := buildP23SelfHostingOptimizerWitness()
	if err != nil {
		return SelfHostingGateV1Report{}, err
	}
	allocator, err := buildP23SelfHostingAllocatorRuntimeWitness()
	if err != nil {
		return SelfHostingGateV1Report{}, err
	}
	stdlib, err := buildP23SelfHostingStdlibWitness()
	if err != nil {
		return SelfHostingGateV1Report{}, err
	}

	decision := selfhostgate.Evaluate(selfhostgate.Evidence{
		CompilerSubsetDefined:       subset.CompilerSubsetDefined,
		RegisterBackendStable:       backend.RegisterBackendEvidencePresent,
		OptimizerValidated:          optimizer.OptimizerValidationEvidencePresent,
		AllocatorStable:             allocator.AllocatorRuntimeEvidencePresent,
		StdlibStrongEnough:          stdlib.StdlibEvidencePresent,
		SmallCompilerComponentBuilt: blockers.SmallCompilerComponentCompiled,
		GoVsTetraOutputCompared:     blockers.GoVsTetraOutputCompared,
		DeterministicBootstrapChain: blockers.DeterministicBootstrapChain,
		CrossPlatformBootstrapStory: blockers.CrossPlatformBootstrapStory,
	})

	report := SelfHostingGateV1Report{
		SchemaVersion: selfHostingGateV1Schema,
		Scope:         selfHostingGateV1ScopeP233,
		Witnesses: []SelfHostingGateV1Witness{
			subset,
			blockers,
			backend,
			optimizer,
			allocator,
			stdlib,
		},
		Rows: []SelfHostingGateV1Row{
			p23SelfHostingGateRow(SelfHostingGateSubsetDefinition, "Self-host subset definition", "defined_gate_subset",
				[]string{
					"P23.3 defines a verified subset gate for evidence-bearing compiler slices; this is not self-hosting and not a claim that Tetra compiles its compiler.",
					"The subset is limited to parser/checker/PLIR/lowering witnesses, scalar i32 backend witnesses, optimizer validation, allocator/runtime contracts, and region-aware stdlib evidence.",
				},
				[]string{
					"go test ./compiler -run 'P23SelfHostingGate'",
					"go test ./compiler/internal/selfhostgate -run 'SelfHosting'",
				},
				[]string{
					"verified subset is an evidence gate, not a Tetra-authored compiler subset that compiles itself",
					"no full compiler source migration to Tetra is claimed",
				},
				[]string{p23SelfHostingSubsetWitnessID}),
			p23SelfHostingGateRow(SelfHostingGateSmallComponentCompile, "Small compiler component compile boundary", "blocked_missing_bootstrap_evidence",
				[]string{
					"No small compiler component is currently claimed to compile as Tetra-authored compiler source.",
					"The small compiler component compile task remains blocked until a real Tetra compiler component source and deterministic build artifact exist.",
				},
				[]string{
					"go test ./compiler -run 'P23SelfHostingGate'",
				},
				[]string{
					"blocked rather than treated as Go compiler evidence",
					"Go implementation tests do not count as a Tetra small compiler component compile",
				},
				[]string{p23SelfHostingBootstrapBlockersWitnessID}),
			p23SelfHostingGateRow(SelfHostingGateOutputComparison, "Go output vs Tetra-compiled output comparison boundary", "blocked_missing_bootstrap_evidence",
				[]string{
					"No Go compiler output vs Tetra-compiled output comparison is claimed yet.",
					"The comparison row remains blocked until both the current Go compiler output and Tetra-compiled output for the same compiler subset are produced and compared deterministically.",
				},
				[]string{
					"go test ./compiler -run 'P23SelfHostingGate'",
				},
				[]string{
					"blocked until same-input Go compiler output and Tetra-compiled output artifacts exist",
					"no output equivalence, byte equivalence, or semantic equivalence claim is made",
				},
				[]string{p23SelfHostingBootstrapBlockersWitnessID}),
			p23SelfHostingGateRow(SelfHostingGateRegisterBackend, "Register backend stability gate", "current_evidence_present",
				[]string{
					"differential.CheckBackendMatrix covers source, Stack IR, optimized Stack IR, SSA, and Machine IR lanes for the register backend stability witness.",
					"Machine IR evidence is current internal backend evidence and does not make the register backend a public self-host backend selector.",
				},
				[]string{
					"go test ./compiler/internal/differential -run 'CheckBackendMatrix' -count=1",
					"go test ./compiler -run 'P23SelfHostingGate'",
				},
				[]string{
					"register backend stability is current supported-subset evidence only",
					"broader compiler self-hosting remains blocked by bootstrap evidence",
				},
				[]string{p23SelfHostingRegisterBackendWitnessID}),
			p23SelfHostingGateRow(SelfHostingGateOptimizerValidation, "Optimizer validation maturity gate", "current_evidence_present",
				[]string{
					"P23.0 translation validation v2 records optimizer validation maturity through registered pass coverage, symbolic scalar equivalence, memory equivalence, proof preservation, allocation preservation, and sha256 before/after metadata.",
					"BuildP23TranslationValidationV2 and ValidateP23TranslationValidationV2 are reused as the live optimizer gate witness.",
				},
				[]string{
					"go test ./compiler -run 'P23TranslationValidationV2|P23SelfHostingGate' -count=1",
				},
				[]string{
					"translation validation v2 is supported-subset evidence, not exhaustive optimizer completeness",
					"optimizer maturity alone does not allow self-hosting",
				},
				[]string{p23SelfHostingOptimizerWitnessID}),
			p23SelfHostingGateRow(SelfHostingGateAllocatorRuntime, "Allocator/runtime stability gate", "current_evidence_present",
				[]string{
					"runtimeabi.RuntimeAllocationContracts validates allocation APIs, guard behavior, failure behavior, debug instrumentation, and report hooks.",
					"runtimeabi.RuntimeRegionAllocatorConfig, AlignRegionBytes, and RuntimePerCoreSmallHeapABI provide allocator/runtime stability evidence for the current gate.",
				},
				[]string{
					"go test ./compiler/internal/runtimeabi -run 'Allocation|Region|SmallHeap' -count=1",
					"go test ./compiler -run 'P23SelfHostingGate'",
				},
				[]string{
					"allocator/runtime evidence is internal runtime ABI evidence, not a complete self-host runtime",
					"cross-platform bootstrap and Tetra compiler component evidence remain blocked",
				},
				[]string{p23SelfHostingAllocatorRuntimeWitnessID}),
			p23SelfHostingGateRow(SelfHostingGateStdlibSufficiency, "Stdlib sufficiency gate", "current_evidence_present",
				[]string{
					"stdlibrt.RegionAwareStdlibCoverage and ValidateRegionAwareStdlibCoverage record current region-aware stdlib evidence for StringBuilder, VecBytes, HashMapBytes, buffers, borrowed JSON/HTTP views, PostgreSQL helpers, and production boundaries.",
					"The stdlib witness is sufficient for this gate evidence layer but not sufficient for a full self-hosting claim.",
				},
				[]string{
					"go test ./compiler/internal/stdlibrt -run 'RegionAwareStdlibCoverage' -count=1",
					"go test ./compiler -run 'P23SelfHostingGate'",
				},
				[]string{
					"stdlib sufficiency is evidence for the current gate only",
					"full compiler stdlib needs and cross-platform bootstrap remain unpromoted",
				},
				[]string{p23SelfHostingStdlibWitnessID}),
			p23SelfHostingGateRow(SelfHostingGateDeterministicBootstrap, "Deterministic bootstrap chain gate", "blocked_missing_bootstrap_evidence",
				[]string{
					"No deterministic bootstrap chain is claimed yet.",
					"The bootstrap chain remains blocked until a staged Go-to-Tetra-to-Tetra compiler build emits deterministic artifacts with stable hashes.",
				},
				[]string{
					"go test ./compiler -run 'P23SelfHostingGate'",
				},
				[]string{
					"blocked until staged bootstrap artifacts and hashes exist",
					"scripts/dev/bootstrap.sh refreshes Go-built binaries and does not count as a self-host chain",
				},
				[]string{p23SelfHostingBootstrapBlockersWitnessID}),
			p23SelfHostingGateRow(SelfHostingGateCrossPlatformBootstrap, "Cross-platform bootstrap story gate", "blocked_missing_bootstrap_evidence",
				[]string{
					"No cross-platform bootstrap story is claimed yet.",
					"The cross-platform bootstrap row remains blocked until Linux, macOS, Windows, and build-only target bootstrap evidence has matching artifacts and no host fallback.",
				},
				[]string{
					"go test ./compiler -run 'P23SelfHostingGate'",
				},
				[]string{
					"blocked until platform-specific bootstrap evidence exists",
					"current native target evidence is not a cross-platform self-host bootstrap story",
				},
				[]string{p23SelfHostingBootstrapBlockersWitnessID}),
			p23SelfHostingGateRow(SelfHostingGateNoSelfHostingClaim, "No self-hosting claim", "blocked_no_claim",
				[]string{
					"SelfHostingClaimed=false and GateDecision.Allowed=false are required for the current P23.3 report.",
					"selfhostgate.Evaluate records missing small compiler component, Go-vs-Tetra output comparison, deterministic bootstrap chain, and cross-platform bootstrap story evidence.",
				},
				[]string{
					"go test ./compiler/internal/selfhostgate -run 'SelfHosting' -count=1",
					"go test ./compiler -run 'P23SelfHostingGate'",
				},
				[]string{
					"no self-hosting claim is made",
					"future self-hosting promotion must replace blocker rows with real evidence and keep GateDecision honest",
				},
				[]string{p23SelfHostingBootstrapBlockersWitnessID}),
		},
		NonClaims: []string{
			"Tetra is not self-hosting",
			"no Tetra compiler component is claimed to compile itself yet",
			"no Go compiler output vs Tetra-compiled output equivalence is claimed yet",
			"no deterministic bootstrap chain is claimed yet",
			"no cross-platform bootstrap story is claimed yet",
			"runtime behavior does not change",
			"safe-program semantics do not change",
			"no performance claim is made",
		},
		GateDecision:                       decision,
		CompilerSubsetDefined:              subset.CompilerSubsetDefined,
		SubsetName:                         subset.SubsetName,
		SmallCompilerComponentCompiled:     blockers.SmallCompilerComponentCompiled,
		GoVsTetraOutputCompared:            blockers.GoVsTetraOutputCompared,
		RegisterBackendEvidencePresent:     backend.RegisterBackendEvidencePresent,
		OptimizerValidationEvidencePresent: optimizer.OptimizerValidationEvidencePresent,
		AllocatorRuntimeEvidencePresent:    allocator.AllocatorRuntimeEvidencePresent,
		StdlibEvidencePresent:              stdlib.StdlibEvidencePresent,
		DeterministicBootstrapChain:        blockers.DeterministicBootstrapChain,
		CrossPlatformBootstrapStory:        blockers.CrossPlatformBootstrapStory,
		SelfHostingClaimed:                 false,
		RuntimeBehaviorChanged:             false,
		SafeSemanticsChanged:               false,
		PerformanceClaimed:                 false,
	}
	if err := ValidateP23SelfHostingGateV1Report(report); err != nil {
		return SelfHostingGateV1Report{}, err
	}
	return report, nil
}

func ValidateP23SelfHostingGateV1Report(report SelfHostingGateV1Report) error {
	if report.SchemaVersion != selfHostingGateV1Schema {
		return fmt.Errorf("self-hosting gate v1: schema_version is %q", report.SchemaVersion)
	}
	if report.Scope != selfHostingGateV1ScopeP233 {
		return fmt.Errorf("self-hosting gate v1: scope is %q", report.Scope)
	}
	if report.SelfHostingClaimed {
		return fmt.Errorf("self-hosting gate v1: self-hosting claim is forbidden")
	}
	if report.RuntimeBehaviorChanged {
		return fmt.Errorf("self-hosting gate v1: runtime behavior change claim is forbidden")
	}
	if report.SafeSemanticsChanged {
		return fmt.Errorf("self-hosting gate v1: safe semantics change claim is forbidden")
	}
	if report.PerformanceClaimed {
		return fmt.Errorf("self-hosting gate v1: performance claim is forbidden")
	}
	if !report.CompilerSubsetDefined || strings.TrimSpace(report.SubsetName) == "" {
		return fmt.Errorf("self-hosting gate v1: compiler subset evidence missing")
	}
	if !report.RegisterBackendEvidencePresent {
		return fmt.Errorf("self-hosting gate v1: register backend evidence missing")
	}
	if !report.OptimizerValidationEvidencePresent {
		return fmt.Errorf("self-hosting gate v1: optimizer validation evidence missing")
	}
	if !report.AllocatorRuntimeEvidencePresent {
		return fmt.Errorf("self-hosting gate v1: allocator/runtime evidence missing")
	}
	if !report.StdlibEvidencePresent {
		return fmt.Errorf("self-hosting gate v1: stdlib evidence missing")
	}
	if report.SmallCompilerComponentCompiled {
		return fmt.Errorf("self-hosting gate v1: small compiler component compile claim is forbidden without Tetra component evidence")
	}
	if report.GoVsTetraOutputCompared {
		return fmt.Errorf("self-hosting gate v1: output comparison claim is forbidden without Go and Tetra artifacts")
	}
	if report.DeterministicBootstrapChain {
		return fmt.Errorf("self-hosting gate v1: deterministic bootstrap claim is forbidden without staged hashes")
	}
	if report.CrossPlatformBootstrapStory {
		return fmt.Errorf("self-hosting gate v1: cross-platform bootstrap claim is forbidden without platform artifacts")
	}
	if err := p23SelfHostingValidateDecision(report.GateDecision); err != nil {
		return err
	}
	for _, want := range []string{
		"Tetra is not self-hosting",
		"no Tetra compiler component is claimed to compile itself yet",
		"no Go compiler output vs Tetra-compiled output equivalence is claimed yet",
		"no deterministic bootstrap chain is claimed yet",
		"no cross-platform bootstrap story is claimed yet",
		"runtime behavior does not change",
		"safe-program semantics do not change",
		"no performance claim is made",
	} {
		if !p23SelfHostingGateHasString(report.NonClaims, want) {
			return fmt.Errorf("self-hosting gate v1: missing non-claim %q", want)
		}
	}
	if err := p23SelfHostingGateValidateRowsAndWitnesses(report.Rows, report.Witnesses); err != nil {
		return err
	}
	return nil
}

func buildP23SelfHostingSubsetWitness() SelfHostingGateV1Witness {
	return SelfHostingGateV1Witness{
		ID:                    p23SelfHostingSubsetWitnessID,
		Kind:                  "self_host_subset_definition",
		CompilerSubsetDefined: true,
		SubsetName:            "p23.3_verified_subset_gate_not_self_hosted",
	}
}

func buildP23SelfHostingBootstrapBlockersWitness() SelfHostingGateV1Witness {
	return SelfHostingGateV1Witness{
		ID:                             p23SelfHostingBootstrapBlockersWitnessID,
		Kind:                           "self_host_bootstrap_blockers",
		SmallCompilerComponentCompiled: false,
		GoVsTetraOutputCompared:        false,
		DeterministicBootstrapChain:    false,
		CrossPlatformBootstrapStory:    false,
		Blockers: []string{
			"small_compiler_component_compiled",
			"go_vs_tetra_output_compared",
			"deterministic_bootstrap_chain",
			"cross_platform_bootstrap_story",
		},
	}
}

func buildP23SelfHostingRegisterBackendWitness() (SelfHostingGateV1Witness, error) {
	matrix, err := differential.CheckBackendMatrix(differential.BackendMatrixCase{
		Name:      "p23.3-self-host-register-backend-loop",
		Functions: []ir.IRFunc{p23LoopSumFunc()},
		Entry:     "sum_n",
		Samples: []differential.MatrixSample{
			{Name: "zero", Args: []int32{0}},
			{Name: "seven", Args: []int32{7}},
		},
		Source: func(sample differential.MatrixSample) (int32, bool) {
			n := sample.Args[0]
			var total int32
			for i := int32(0); i < n; i++ {
				total += i
			}
			return total, true
		},
	})
	if err != nil {
		return SelfHostingGateV1Witness{}, err
	}
	return SelfHostingGateV1Witness{
		ID:                             p23SelfHostingRegisterBackendWitnessID,
		Kind:                           "register_backend_stability",
		RegisterBackendEvidencePresent: matrix.HasLane(differential.LaneMachineIRInterpreter),
		BackendMatrixLanes:             len(matrix.Lanes),
	}, nil
}

func buildP23SelfHostingOptimizerWitness() (SelfHostingGateV1Witness, error) {
	report, err := BuildP23TranslationValidationV2()
	if err != nil {
		return SelfHostingGateV1Witness{}, err
	}
	if err := ValidateP23TranslationValidationV2(report); err != nil {
		return SelfHostingGateV1Witness{}, err
	}
	return SelfHostingGateV1Witness{
		ID:   p23SelfHostingOptimizerWitnessID,
		Kind: "optimizer_validation_maturity",
		OptimizerValidationEvidencePresent: report.RegisteredPassCoverageComplete &&
			report.SymbolicScalarEquivalenceSamples > 0 &&
			report.MemoryEquivalenceSamples > 0 &&
			report.BoundsProofsPreserved &&
			report.AllocationPlanValidated &&
			report.BeforeAfterHashesMachineCheckable,
		TranslationValidationRows:      len(report.Rows),
		TranslationValidationWitnesses: len(report.Witnesses),
	}, nil
}

func buildP23SelfHostingAllocatorRuntimeWitness() (SelfHostingGateV1Witness, error) {
	contracts := runtimeabi.RuntimeAllocationContracts()
	for _, contract := range contracts {
		if err := runtimeabi.ValidateRuntimeAllocationContract(contract); err != nil {
			return SelfHostingGateV1Witness{}, err
		}
	}
	region := runtimeabi.RuntimeRegionAllocatorConfig(false)
	aligned, alignedOK := runtimeabi.AlignRegionBytes(33)
	_, invalidRejected := runtimeabi.AlignRegionBytes(-1)
	allocator, err := runtimeabi.NewPerCoreSmallHeapAllocator(runtimeabi.RuntimePerCoreSmallHeapABI(2))
	if err != nil {
		return SelfHostingGateV1Witness{}, err
	}
	handle, err := allocator.Alloc(0, 32)
	if err != nil {
		return SelfHostingGateV1Witness{}, err
	}
	if err := allocator.Free(handle); err != nil {
		return SelfHostingGateV1Witness{}, err
	}
	if _, err := allocator.Alloc(0, 32); err != nil {
		return SelfHostingGateV1Witness{}, err
	}
	smallHeap := allocator.Report()
	return SelfHostingGateV1Witness{
		ID:   p23SelfHostingAllocatorRuntimeWitnessID,
		Kind: "allocator_runtime_stability",
		AllocatorRuntimeEvidencePresent: len(contracts) >= 5 &&
			region.AlignmentBytes == runtimeabi.RegionAllocatorAlignmentBytes &&
			alignedOK &&
			aligned == 48 &&
			!invalidRejected &&
			smallHeap.TotalReuses > 0 &&
			!smallHeap.EstimatedMmapPerAllocation,
		RuntimeAllocationContracts:      len(contracts),
		RegionAllocatorAlignmentBytes:   region.AlignmentBytes,
		PerCoreSmallHeapEvidencePresent: smallHeap.TotalReuses > 0 && !smallHeap.EstimatedMmapPerAllocation,
	}, nil
}

func buildP23SelfHostingStdlibWitness() (SelfHostingGateV1Witness, error) {
	report, err := stdlibrt.RegionAwareStdlibCoverage()
	if err != nil {
		return SelfHostingGateV1Witness{}, err
	}
	if err := stdlibrt.ValidateRegionAwareStdlibCoverage(report); err != nil {
		return SelfHostingGateV1Witness{}, err
	}
	return SelfHostingGateV1Witness{
		ID:                    p23SelfHostingStdlibWitnessID,
		Kind:                  "stdlib_sufficiency",
		StdlibEvidencePresent: len(report.Rows) >= 10,
		StdlibRows:            len(report.Rows),
	}, nil
}

func p23SelfHostingValidateDecision(decision selfhostgate.Decision) error {
	if decision.Allowed {
		return fmt.Errorf("self-hosting gate v1: gate decision unexpectedly allowed self-hosting")
	}
	if !strings.Contains(decision.Reason, "blocked") {
		return fmt.Errorf("self-hosting gate v1: gate decision reason must remain blocked")
	}
	for _, missing := range []string{
		"small_compiler_component_compiled",
		"go_vs_tetra_output_compared",
		"deterministic_bootstrap_chain",
		"cross_platform_bootstrap_story",
	} {
		if !decision.Missing(missing) {
			return fmt.Errorf("self-hosting gate v1: gate decision missing blocker %s", missing)
		}
	}
	return nil
}

func p23SelfHostingGateValidateRowsAndWitnesses(rows []SelfHostingGateV1Row, witnesses []SelfHostingGateV1Witness) error {
	byWitness := map[string]SelfHostingGateV1Witness{}
	for _, witness := range witnesses {
		if strings.TrimSpace(witness.ID) == "" || strings.TrimSpace(witness.Kind) == "" {
			return fmt.Errorf("self-hosting gate v1: witness missing id or kind")
		}
		if _, exists := byWitness[witness.ID]; exists {
			return fmt.Errorf("self-hosting gate v1: duplicate witness %q", witness.ID)
		}
		byWitness[witness.ID] = witness
	}
	expected := map[SelfHostingGateV1ID]bool{}
	for _, id := range p23SelfHostingGateV1IDs() {
		expected[id] = true
	}
	seen := map[SelfHostingGateV1ID]bool{}
	for _, row := range rows {
		if !expected[row.ID] {
			return fmt.Errorf("self-hosting gate v1: unexpected row %q", row.ID)
		}
		if seen[row.ID] {
			return fmt.Errorf("self-hosting gate v1: duplicate row %q", row.ID)
		}
		seen[row.ID] = true
		if strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Status) == "" {
			return fmt.Errorf("self-hosting gate v1: row %q missing name or status", row.ID)
		}
		if len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 || len(row.WitnessIDs) == 0 {
			return fmt.Errorf("self-hosting gate v1: row %q missing evidence, tests, boundaries, or witness ids", row.ID)
		}
		for _, text := range append(append([]string{}, row.Evidence...), row.Boundaries...) {
			if p23SelfHostingGateIsPlaceholder(text) {
				return fmt.Errorf("self-hosting gate v1: row %q has placeholder evidence", row.ID)
			}
		}
		for _, id := range row.WitnessIDs {
			if _, ok := byWitness[id]; !ok {
				return fmt.Errorf("self-hosting gate v1: row %q references missing witness %q", row.ID, id)
			}
		}
	}
	for _, id := range p23SelfHostingGateV1IDs() {
		if !seen[id] {
			return fmt.Errorf("self-hosting gate v1: missing row %q", id)
		}
	}
	subset := byWitness[p23SelfHostingSubsetWitnessID]
	if !subset.CompilerSubsetDefined || !strings.Contains(subset.SubsetName, "verified") {
		return fmt.Errorf("self-hosting gate v1: compiler subset witness incomplete")
	}
	backend := byWitness[p23SelfHostingRegisterBackendWitnessID]
	if !backend.RegisterBackendEvidencePresent || backend.BackendMatrixLanes < 5 {
		return fmt.Errorf("self-hosting gate v1: register backend witness incomplete")
	}
	optimizer := byWitness[p23SelfHostingOptimizerWitnessID]
	if !optimizer.OptimizerValidationEvidencePresent || optimizer.TranslationValidationRows < 6 {
		return fmt.Errorf("self-hosting gate v1: optimizer witness incomplete")
	}
	allocator := byWitness[p23SelfHostingAllocatorRuntimeWitnessID]
	if !allocator.AllocatorRuntimeEvidencePresent || allocator.RuntimeAllocationContracts < 5 || !allocator.PerCoreSmallHeapEvidencePresent {
		return fmt.Errorf("self-hosting gate v1: allocator/runtime witness incomplete")
	}
	stdlib := byWitness[p23SelfHostingStdlibWitnessID]
	if !stdlib.StdlibEvidencePresent || stdlib.StdlibRows < 10 {
		return fmt.Errorf("self-hosting gate v1: stdlib witness incomplete")
	}
	blockers := byWitness[p23SelfHostingBootstrapBlockersWitnessID]
	if blockers.SmallCompilerComponentCompiled || blockers.GoVsTetraOutputCompared || blockers.DeterministicBootstrapChain || blockers.CrossPlatformBootstrapStory {
		return fmt.Errorf("self-hosting gate v1: bootstrap blockers witness claims unavailable evidence")
	}
	return nil
}

func p23SelfHostingGateV1IDs() []SelfHostingGateV1ID {
	return []SelfHostingGateV1ID{
		SelfHostingGateSubsetDefinition,
		SelfHostingGateSmallComponentCompile,
		SelfHostingGateOutputComparison,
		SelfHostingGateRegisterBackend,
		SelfHostingGateOptimizerValidation,
		SelfHostingGateAllocatorRuntime,
		SelfHostingGateStdlibSufficiency,
		SelfHostingGateDeterministicBootstrap,
		SelfHostingGateCrossPlatformBootstrap,
		SelfHostingGateNoSelfHostingClaim,
	}
}

func p23SelfHostingGateRow(id SelfHostingGateV1ID, name, status string, evidence, tests, boundaries, witnessIDs []string) SelfHostingGateV1Row {
	return SelfHostingGateV1Row{
		ID:         id,
		Name:       name,
		Status:     status,
		Evidence:   evidence,
		Tests:      tests,
		Boundaries: boundaries,
		WitnessIDs: witnessIDs,
	}
}

func p23SelfHostingGateHasString(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

func p23SelfHostingGateIsPlaceholder(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return lower == "" ||
		lower == "todo" ||
		lower == "tbd" ||
		strings.Contains(lower, "placeholder")
}
