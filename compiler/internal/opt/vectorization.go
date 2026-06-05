package opt

import (
	"fmt"

	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/machine"
)

type VectorizationID string

const (
	VectorizationSumI32       VectorizationID = "sum_i32"
	VectorizationCopyU8       VectorizationID = "copy_u8"
	VectorizationMemsetMemcpy VectorizationID = "memset_memcpy"
	VectorizationMapI32       VectorizationID = "map_i32"
)

type VectorizationStatus string

const (
	VectorizationCandidateGuarded       VectorizationStatus = "candidate_guarded"
	VectorizationBackendLoweringGuarded VectorizationStatus = "backend_lowering_guarded"
	VectorizationImplementedNarrow      VectorizationStatus = "implemented_narrow"
	VectorizationNotYetCovered          VectorizationStatus = "not_yet_covered"
)

type VectorizationDecision string

const (
	VectorizationVectorized    VectorizationDecision = "vectorized"
	VectorizationNotVectorized VectorizationDecision = "not_vectorized"
)

type VectorizationCoverageReport struct {
	SchemaVersion string                     `json:"schema_version"`
	Rows          []VectorizationCoverageRow `json:"rows"`
	NonClaims     []string                   `json:"non_claims"`
}

type VectorizationCoverageRow struct {
	ID            VectorizationID       `json:"id"`
	Name          string                `json:"name"`
	Status        VectorizationStatus   `json:"status"`
	Decision      VectorizationDecision `json:"decision"`
	Candidate     bool                  `json:"candidate,omitempty"`
	RangeProof    bool                  `json:"range_proof,omitempty"`
	ProofID       string                `json:"proof_id,omitempty"`
	MachineTarget string                `json:"machine_target,omitempty"`
	MachinePath   string                `json:"machine_path,omitempty"`
	RequiredFacts []string              `json:"required_facts,omitempty"`
	MissingFacts  []string              `json:"missing_facts,omitempty"`
	Reason        string                `json:"reason"`
	Evidence      string                `json:"evidence"`
	Boundary      string                `json:"boundary"`
}

func VectorizationCoverage() (VectorizationCoverageReport, error) {
	sum, err := vectorizationSumI32Row()
	if err != nil {
		return VectorizationCoverageReport{}, err
	}
	copyU8, err := vectorizationCopyU8Row()
	if err != nil {
		return VectorizationCoverageReport{}, err
	}
	memsetMemcpy, err := vectorizationMemsetMemcpyRow()
	if err != nil {
		return VectorizationCoverageReport{}, err
	}
	mapI32, err := vectorizationMapI32Row()
	if err != nil {
		return VectorizationCoverageReport{}, err
	}
	return VectorizationCoverageReport{
		SchemaVersion: "tetra.optimizer.vectorization.v1",
		Rows: []VectorizationCoverageRow{
			sum,
			copyU8,
			memsetMemcpy,
			mapI32,
		},
		NonClaims: []string{
			"no broad SIMD or auto-vectorization claim",
			"no throughput or C/Rust performance parity claim",
			"no vector path is selected without noalias/provenance and alignment or safe-unaligned evidence",
		},
	}, nil
}

func vectorizationMemsetMemcpyRow() (VectorizationCoverageRow, error) {
	memsetFn := vectorizationMemsetZeroU8Func(true)
	memsetPlan, memsetOK, err := machine.VectorU8x16MemsetZeroPlanFromStackIR(memsetFn)
	if err != nil {
		return VectorizationCoverageRow{}, err
	}
	if !memsetOK {
		return VectorizationCoverageRow{}, fmt.Errorf("vectorization coverage: proof-tagged memset_zero_u8 helper did not match vector-u8x16-memset-zero machine shape")
	}
	copyFn := vectorizationCopyU8Func(true)
	copyPlan, copyOK, err := machine.VectorU8x16CopyLoopPlanFromStackIR(copyFn)
	if err != nil {
		return VectorizationCoverageRow{}, err
	}
	if !copyOK {
		return VectorizationCoverageRow{}, fmt.Errorf("vectorization coverage: memcpy helper evidence did not match existing vector-u8x16-copy machine shape")
	}
	if copyPlan.ProofID == "" {
		return VectorizationCoverageRow{}, fmt.Errorf("vectorization coverage: memcpy helper copy_u8 evidence missing proof id")
	}
	return VectorizationCoverageRow{
		ID:            VectorizationMemsetMemcpy,
		Name:          "memset/memcpy helpers",
		Status:        VectorizationImplementedNarrow,
		Decision:      VectorizationVectorized,
		Candidate:     true,
		RangeProof:    true,
		ProofID:       memsetPlan.ProofID,
		MachineTarget: memsetPlan.Function.Target,
		MachinePath:   "linux-x64-native-simd-vector-u8x16-memset-zero-plus-copy-u8-helper",
		RequiredFacts: []string{
			"range_proof",
			"noalias_not_required_single_mutable_slice_zero_fill",
			"memcpy_helper_reuses_copy_u8_noalias_source_dest_disjoint",
			"safe_unaligned_vector_path",
			"vector_backend_lowering",
			"tail_handling",
			"scalar_fallback",
			"native_simd_codegen",
			"translation_differential_validation",
		},
		Reason:   "vectorized:proof_tagged_memset_loop_zero_fill_linux_x64_native_simd_and_memcpy_helper_via_copy_u8_with_stack_fallback_differential_validation",
		Evidence: "compiler/internal/machine/vector_memset_u8.go::VectorU8x16MemsetZeroPlanFromStackIR; compiler/internal/machine/ir_test.go::TestVectorU8x16MemsetZeroHelperFromStackIRRequiresRangeSafeUnalignedTailAndFallback; compiler/internal/backend/x64core/vector_memset_u8_register.go::emitVectorMemsetZeroU8RegisterFunction; compiler/internal/backend/linux_x64/codegen_test.go::TestCodegenObjectLinuxX64UsesVectorMemsetZeroU8PathForProofHelper; compiler/internal/backend/linux_x64/codegen_test.go::TestCodegenObjectLinuxX64VectorMemsetZeroU8MatchesStackFallbackWithTail; memcpy helper via copy []u8 evidence reuses compiler/internal/machine/vector_copy_u8.go::VectorU8x16CopyLoopPlanFromStackIR and compiler/internal/backend/linux_x64/codegen_test.go::TestCodegenObjectLinuxX64VectorCopyU8MatchesStackFallbackWithTail",
		Boundary: "proof-tagged memset/memcpy helpers are implemented narrowly: memset coverage is limited to a proof-tagged zero-fill helper with memset-loop range proof, single mutable slice zero-fill noalias-not-required evidence, safe unaligned u8x16 vector backend lowering, scalar tail handling, scalar-u8-memset-zero fallback through vector-u8x16-memset-zero-plan, linux-x64 native SIMD codegen, and translation/differential validation against stack fallback; memcpy helper via copy []u8 is limited to the existing proof-tagged source/dest disjoint copy_u8 linux-x64 native SIMD evidence; no arbitrary non-zero memset, overlapping memcpy, libc/runtime helper lowering, checked/no-proof helper, throughput, or C/Rust parity claim is made",
	}, nil
}

func vectorizationMapI32Row() (VectorizationCoverageRow, error) {
	fn := vectorizationMapI32Func(true)
	vectorPlan, vectorOK, err := machine.VectorI32x4MapAddConstPlanFromStackIR(fn)
	if err != nil {
		return VectorizationCoverageRow{}, err
	}
	if !vectorOK {
		return VectorizationCoverageRow{}, fmt.Errorf("vectorization coverage: proof-tagged map_i32 candidate did not match vector-i32x4-map-add-const machine shape")
	}
	return VectorizationCoverageRow{
		ID:            VectorizationMapI32,
		Name:          "simple map over []i32",
		Status:        VectorizationImplementedNarrow,
		Decision:      VectorizationVectorized,
		Candidate:     true,
		RangeProof:    true,
		ProofID:       vectorPlan.ProofID,
		MachineTarget: vectorPlan.Function.Target,
		MachinePath:   "linux-x64-native-simd-vector-i32x4-map-add-const",
		RequiredFacts: []string{
			"range_proof",
			"noalias_not_required_single_mutable_slice_in_place",
			"safe_unaligned_vector_path",
			"vector_backend_lowering",
			"tail_handling",
			"scalar_fallback",
			"native_simd_codegen",
			"translation_differential_validation",
		},
		Reason:   "vectorized:proof_tagged_map_i32_add_const_linux_x64_native_simd_with_stack_fallback_differential_validation",
		Evidence: "compiler/internal/machine/vector_map_i32.go::VectorI32x4MapAddConstPlanFromStackIR; compiler/internal/machine/ir_test.go::TestVectorI32x4MapAddConstFromStackIRRequiresRangeSafeUnalignedTailAndFallback; compiler/internal/backend/x64core/vector_map_i32_register.go::emitVectorMapI32AddConstRegisterFunction; compiler/internal/backend/linux_x64/codegen_test.go::TestCodegenObjectLinuxX64UsesVectorMapI32AddConstPathForProofLoop; compiler/internal/backend/linux_x64/codegen_test.go::TestCodegenObjectLinuxX64VectorMapI32AddConstMatchesStackFallbackWithTail",
		Boundary: "proof-tagged simple map over []i32 has map-loop range proof, noalias not required evidence for a single mutable slice in-place map, safe unaligned i32x4 vector backend lowering, scalar tail handling, scalar-i32-map fallback through vector-i32x4-map-add-const-plan, linux-x64 native SIMD codegen, and translation/differential validation against stack fallback; vectorized scope is limited to proof-tagged in-place add-constant-1 map []i32 on linux-x64 machine paths and does not claim checked/no-proof map, broader map shapes, memset/memcpy, throughput, or C/Rust parity",
	}, nil
}

func vectorizationCopyU8Row() (VectorizationCoverageRow, error) {
	fn := vectorizationCopyU8Func(true)
	vectorPlan, vectorOK, err := machine.VectorU8x16CopyLoopPlanFromStackIR(fn)
	if err != nil {
		return VectorizationCoverageRow{}, err
	}
	if !vectorOK {
		return VectorizationCoverageRow{}, fmt.Errorf("vectorization coverage: proof-tagged copy_u8 candidate did not match vector-u8x16-copy machine shape")
	}
	return VectorizationCoverageRow{
		ID:            VectorizationCopyU8,
		Name:          "copy []u8",
		Status:        VectorizationImplementedNarrow,
		Decision:      VectorizationVectorized,
		Candidate:     true,
		RangeProof:    true,
		ProofID:       vectorPlan.ProofID,
		MachineTarget: vectorPlan.Function.Target,
		MachinePath:   "machine-ir-vector-u8x16-copy-plan",
		RequiredFacts: []string{
			"range_proof",
			"noalias_source_dest_disjoint_owned_copy_result",
			"safe_unaligned_vector_path",
			"vector_backend_lowering",
			"tail_handling",
			"scalar_fallback",
			"native_simd_codegen",
			"translation_differential_validation",
		},
		Reason:   "vectorized:proof_tagged_copy_u8_linux_x64_native_simd_with_stack_fallback_differential_validation",
		Evidence: "compiler/internal/plir/plir.go::addCopyLoopRangeProof; compiler/internal/plir/plir_test.go::TestFromCheckedProgramRecordsBorrowCopyFacts; compiler/internal/machine/vector_copy_u8.go::VectorU8x16CopyLoopPlanFromStackIR; compiler/internal/machine/ir_test.go::TestVectorU8x16CopyLoopFromStackIRRequiresRangeNoAliasSafeUnalignedTailAndFallback; compiler/internal/backend/x64core/vector_copy_u8_register.go::emitVectorCopyU8RegisterFunction; compiler/internal/backend/linux_x64/codegen_test.go::TestCodegenObjectLinuxX64UsesVectorCopyU8PathForProofLoop; compiler/internal/backend/linux_x64/codegen_test.go::TestCodegenObjectLinuxX64VectorCopyU8MatchesStackFallbackWithTail",
		Boundary: "proof-tagged copy []u8 has copy-loop range proof, noalias required evidence through source/dest disjoint owned copy result, safe unaligned u8x16 vector backend lowering, scalar tail handling, scalar-u8-copy fallback through vector-u8x16-copy-plan, linux-x64 native SIMD codegen, and translation/differential validation against stack fallback; vectorized scope is limited to proof-tagged copy []u8 on linux-x64 machine paths and does not claim checked/no-proof copy, overlapping slices, memset/memcpy, map []i32, throughput, or C/Rust parity",
	}, nil
}

func vectorizationSumI32Row() (VectorizationCoverageRow, error) {
	fn := hotLoopSliceSumFunc(true)
	ssaVerified, err := hotLoopSSAVerified(fn)
	if err != nil {
		return VectorizationCoverageRow{}, err
	}
	plan, ok, err := machine.ScalarI32SliceSumLoopPlanFromStackIR(fn)
	if err != nil {
		return VectorizationCoverageRow{}, err
	}
	if !ok {
		return VectorizationCoverageRow{}, fmt.Errorf("vectorization coverage: proof-tagged sum_i32 candidate did not match scalar-i32-slice-sum machine shape")
	}
	vectorPlan, vectorOK, err := machine.VectorI32x4SliceSumLoopPlanFromStackIR(fn)
	if err != nil {
		return VectorizationCoverageRow{}, err
	}
	if !vectorOK {
		return VectorizationCoverageRow{}, fmt.Errorf("vectorization coverage: proof-tagged sum_i32 candidate did not match vector-i32x4-slice-sum machine shape")
	}
	if !ssaVerified || plan.ProofID == "" {
		return VectorizationCoverageRow{}, fmt.Errorf("vectorization coverage: proof-tagged sum_i32 candidate missing SSA/range proof evidence")
	}
	return VectorizationCoverageRow{
		ID:            VectorizationSumI32,
		Name:          "sum []i32",
		Status:        VectorizationImplementedNarrow,
		Decision:      VectorizationVectorized,
		Candidate:     true,
		RangeProof:    true,
		ProofID:       vectorPlan.ProofID,
		MachineTarget: vectorPlan.Function.Target,
		MachinePath:   "linux-x64-native-simd-vector-i32x4-slice-sum",
		RequiredFacts: []string{
			"range_proof",
			"noalias_not_required_read_only_reduction",
			"safe_unaligned_vector_path",
			"vector_backend_lowering",
			"tail_handling",
			"scalar_fallback",
			"native_simd_codegen",
			"translation_differential_validation",
		},
		Reason:   "vectorized:proof_sum_i32_safe_unaligned_i32x4_native_simd_validated",
		Evidence: "compiler/internal/opt/hotloop.go::CoreHotLoopShapeEvidence; compiler/internal/machine/scalar_slice_sum.go::ScalarI32SliceSumLoopPlanFromStackIR; compiler/internal/machine/vector_slice_sum.go::VectorI32x4SliceSumLoopPlanFromStackIR; compiler/internal/machine/ir_test.go::TestVectorI32x4SliceSumLoopFromStackIRUsesSafeUnalignedTailAndScalarFallback; compiler/internal/backend/x64core/vector_slice_sum_register.go::emitVectorSliceSumRegisterFunction; compiler/internal/backend/linux_x64/codegen_test.go::TestCodegenObjectLinuxX64UsesVectorSliceSumPathForProofLoop; compiler/internal/backend/linux_x64/codegen_test.go::TestCodegenObjectLinuxX64VectorSliceSumMatchesStackFallbackWithTail; docs/audits/noalias-mutable-borrow-v1.md::read-only reduction means noalias not required for sum []i32",
		Boundary: "proof-tagged sum []i32 has range proof, noalias not required because this is a read-only reduction with no slice memory stores, safe unaligned i32x4 vector backend lowering, scalar tail handling, scalar-i32-slice-sum fallback through vector-i32x4-slice-sum-plan, linux-x64 native SIMD codegen, and translation/differential validation against stack fallback; vectorized scope is limited to proof-tagged step=1 sum []i32 on linux-x64 machine paths and does not claim checked/no-proof loops, constant stride, copy []u8, memset/memcpy, map []i32, throughput, or C/Rust parity",
	}, nil
}

func vectorizationNotYetCoveredRow(id VectorizationID, name string, reason string) VectorizationCoverageRow {
	return VectorizationCoverageRow{
		ID:       id,
		Name:     name,
		Status:   VectorizationNotYetCovered,
		Decision: VectorizationNotVectorized,
		Reason:   "not_vectorized:" + reason,
		Evidence: "P17.3 master-plan initial target row",
		Boundary: reason + "; no vector candidate, no range proof, no noalias/alignment facts, no vector backend lowering, and no SIMD/performance claim",
	}
}

func vectorizationCopyU8Func(proof bool) ir.IRFunc {
	loadKind := ir.IRIndexLoadU8
	proofID := ""
	if proof {
		loadKind = ir.IRIndexLoadU8Unchecked
		proofID = "proof:copy-loop:u8:1:1"
	}
	return ir.IRFunc{
		Name:        "copy_u8",
		ParamSlots:  3,
		LocalSlots:  4,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: loadKind, ProofID: proofID},
			{Kind: ir.IRIndexStoreU8},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
}

func vectorizationMapI32Func(proof bool) ir.IRFunc {
	loadKind := ir.IRIndexLoadI32
	proofID := ""
	if proof {
		loadKind = ir.IRIndexLoadI32Unchecked
		proofID = "proof:map-loop:i32:1:1"
	}
	return ir.IRFunc{
		Name:        "map_i32_add1",
		ParamSlots:  2,
		LocalSlots:  4,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: loadKind, ProofID: proofID},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 3},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 3},
			{Kind: ir.IRIndexStoreI32},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
}

func vectorizationMemsetZeroU8Func(proof bool) ir.IRFunc {
	proofID := ""
	if proof {
		proofID = "proof:memset-loop:u8:zero:1:1"
	}
	return ir.IRFunc{
		Name:        "memset_zero_u8",
		ParamSlots:  2,
		LocalSlots:  3,
		ReturnSlots: 1,
		Instrs: []ir.IRInstr{
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRLabel, Label: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRCmpLtI32},
			{Kind: ir.IRJmpIfZero, Label: 2},
			{Kind: ir.IRLoadLocal, Local: 0},
			{Kind: ir.IRLoadLocal, Local: 1},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRIndexStoreU8, ProofID: proofID},
			{Kind: ir.IRLoadLocal, Local: 2},
			{Kind: ir.IRConstI32, Imm: 1},
			{Kind: ir.IRAddI32},
			{Kind: ir.IRStoreLocal, Local: 2},
			{Kind: ir.IRJmp, Label: 1},
			{Kind: ir.IRLabel, Label: 2},
			{Kind: ir.IRConstI32, Imm: 0},
			{Kind: ir.IRReturn},
		},
	}
}
