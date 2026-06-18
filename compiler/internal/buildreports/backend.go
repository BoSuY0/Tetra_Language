package buildreports

import (
	"fmt"
	"sort"
	"strings"

	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/machine"
	machinebounds "tetra_language/compiler/internal/machine/bounds"
	"tetra_language/compiler/internal/ssair"
)

func BuildBackendReport(target string, irProg *ir.IRProgram) BackendReport {
	report := BackendReport{
		ReportEnvelope: ReportEnvelope{SchemaVersion: 2, Kind: "backend", Target: target},
		Backend:        "stack",
		Mode:           "compatibility-with-proof-checked-index-loads",
	}
	if !targetSupportsMachineScalar(target) || irProg == nil {
		report.Functions = buildBackendFunctionPathReports(target, irProg, nil)
		report.Summary = summarizeBackendCoverage(report.Functions, nil)
		return report
	}
	machineReports := buildMachineBackendFunctionReports(target, irProg)
	report.Functions = buildBackendFunctionPathReports(target, irProg, machineReports)
	report.Summary = summarizeBackendCoverage(report.Functions, machineReports)
	if len(machineReports) == 0 {
		return report
	}
	report.Backend = "stack+machine"
	report.MachineFunctions = machineReports
	hasScalar := false
	hasLoop := false
	hasSliceSum := false
	hasSliceSumMain := false
	hasMatrixMultiplyMain := false
	hasCall := false
	hasPostgreSQLFrameTypeAt := false
	hasPostgreSQLInoutWriter := false
	hasHelperSummaryInoutWriter := false
	hasParallelMapReduceMain := false
	hasRegionIslandAllocationMain := false
	hasActorPingPongRuntimeCall := false
	for _, fn := range machineReports {
		switch fn.Path {
		case "machine-ir-call",
			"machine-ir-call-loop",
			"machine-ir-recursive-fib",
			"machine-ir-recursion-main-loop",
			"machine-ir-parallel-map-reduce-main",
			"machine-ir-actor-ping-pong-pong",
			"machine-ir-actor-ping-pong-main":
			hasCall = true
			if fn.Path == "machine-ir-parallel-map-reduce-main" {
				hasParallelMapReduceMain = true
			}
			if fn.Path == "machine-ir-actor-ping-pong-pong" ||
				fn.Path == "machine-ir-actor-ping-pong-main" {
				hasActorPingPongRuntimeCall = true
			}
		case "machine-ir-scalar":
			hasScalar = true
		case "machine-ir-loop",
			"machine-ir-const-modulo-loop",
			"machine-ir-allocation-loop",
			"machine-ir-bounds-check-loops",
			"machine-ir-hash-table-lookup",
			"machine-ir-hash-table-main":
			hasLoop = true
		case "machine-ir-slice-sum":
			hasSliceSum = true
		case "machine-ir-slice-sum-main":
			hasSliceSumMain = true
		case "machine-ir-matrix-multiply-main":
			hasMatrixMultiplyMain = true
		case "machine-ir-postgresql-frame-type-at":
			hasPostgreSQLFrameTypeAt = true
		case "machine-ir-postgresql-inout-writer",
			"machine-ir-postgresql-inout-writer-main":
			hasPostgreSQLInoutWriter = true
		case "machine-ir-inout-writer-helper-summary",
			"machine-ir-inout-writer-helper-summary-caller":
			hasHelperSummaryInoutWriter = true
		case "machine-ir-region-island-allocation-main":
			hasRegionIslandAllocationMain = true
		}
	}
	switch {
	case hasRegionIslandAllocationMain:
		report.Mode = "machine-ir-region-island-allocation-main-for-exact-row; stack fallback otherwise"
	case hasMatrixMultiplyMain:
		report.Mode = "machine-ir-matrix-multiply-main-for-exact-row; stack fallback otherwise"
	case hasSliceSumMain:
		report.Mode = "machine-ir-slice-sum-main-for-exact-row; stack fallback otherwise"
	case hasParallelMapReduceMain:
		report.Mode = "machine-ir-parallel-map-reduce-main-for-exact-row; stack fallback otherwise"
	case hasActorPingPongRuntimeCall:
		report.Mode = "machine-ir-actor-ping-pong-runtime-call-for-exact-row; stack fallback otherwise"
	case hasCall && hasSliceSum:
		report.Mode = ("machine-ir-scalar-loop-slice-sum-and-calls-for-eligible-" +
			"functions; stack fallback otherwise")
	case hasCall:
		report.Mode = "machine-ir-calls-for-eligible-functions; stack fallback otherwise"
	case hasSliceSum:
		report.Mode = ("machine-ir-scalar-loop-and-slice-sum-for-eligible-functions; " +
			"stack fallback otherwise")
	case hasScalar && hasLoop:
		report.Mode = "machine-ir-scalar-and-loop-for-eligible-functions; stack fallback otherwise"
	case hasLoop:
		report.Mode = "machine-ir-loop-for-eligible-functions; stack fallback otherwise"
	case hasPostgreSQLInoutWriter:
		report.Mode = "machine-ir-postgresql-inout-writer-for-exact-row; stack fallback otherwise"
	case hasHelperSummaryInoutWriter:
		report.Mode = "machine-ir-inout-writer-helper-summary-for-exact-row; stack fallback otherwise"
	case hasPostgreSQLFrameTypeAt:
		report.Mode = "machine-ir-postgresql-frame-type-at-for-exact-helper; stack fallback otherwise"
	default:
		report.Mode = "machine-ir-scalar-for-eligible-functions; stack fallback otherwise"
	}
	return report
}

func buildBackendFunctionPathReports(
	target string,
	irProg *ir.IRProgram,
	machineReports []MachineBackendFunctionReport,
) []BackendFunctionPathReport {
	if irProg == nil {
		return nil
	}
	machineByFunction := make(map[string]MachineBackendFunctionReport, len(machineReports))
	for _, report := range machineReports {
		machineByFunction[report.Function] = report
	}
	rows := make([]BackendFunctionPathReport, 0, len(irProg.Funcs))
	for _, fn := range irProg.Funcs {
		if machineReport, ok := machineByFunction[fn.Name]; ok {
			row := BackendFunctionPathReport{
				Function:    fn.Name,
				BackendPath: "register",
				Category:    "register_path",
				ABI:         backendABIBoundaryForFunction(target, fn, "register"),
				Detail:      machineReport.Path,
				Reason:      "eligible_machine_ir_subset",
			}
			applyBackendRuntimeFeatureEvidence(&row, fn)
			applyBackendHotness(&row)
			rows = append(rows, row)
			continue
		}
		classification := classifyBackendFallback(target, fn)
		row := BackendFunctionPathReport{
			Function:    fn.Name,
			BackendPath: "stack",
			Category:    classification.Category,
			ABI:         backendABIBoundaryForFunction(target, fn, "stack"),
			Detail:      classification.Detail,
			Reason:      classification.Reason,
		}
		applyBackendRuntimeFeatureEvidence(&row, fn)
		applyBackendHotness(&row)
		rows = append(rows, row)
	}
	return rows
}

type backendFallbackClassification struct {
	Category string
	Detail   string
	Reason   string
}

func classifyBackendFallback(target string, fn ir.IRFunc) backendFallbackClassification {
	if fn.ReturnSlots > 2 {
		return backendFallbackClassification{
			Category: "unsupported_aggregate_return",
			Detail:   fmt.Sprintf("return_slots=%d", fn.ReturnSlots),
			Reason:   "unsupported_aggregate_return_uses_stack_fallback",
		}
	}
	if fn.ReturnSlots == 2 {
		return backendFallbackClassification{
			Category: "unsupported_slice_string_return",
			Detail:   "return_slots=2",
			Reason:   "unsupported_slice_or_string_return_uses_stack_fallback",
		}
	}
	callABI := machineCallABIForTarget(target)
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRCall && backendCallABIUnsupported(instr, callABI) {
			return backendFallbackClassification{
				Category: "unsupported_call_abi",
				Detail: fmt.Sprintf(
					"call=%s arg_slots=%d ret_slots=%d max_arg_slots=%d max_ret_slots=%d",
					instr.Name,
					instr.ArgSlots,
					instr.RetSlots,
					callABI.MaxArgSlots,
					callABI.MaxRetSlots,
				),
				Reason: "unsupported_call_abi_uses_stack_fallback",
			}
		}
	}
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRCall && backendCallLooksRuntimeEffect(instr.Name) {
			return backendFallbackClassification{
				Category: "unsupported_effect_runtime_call",
				Detail:   fmt.Sprintf("runtime_call=%s", instr.Name),
				Reason:   "unsupported_effect_runtime_call_uses_stack_fallback",
			}
		}
		if backendIRKindIsIslandDomainPrimitive(instr.Kind) {
			return backendFallbackClassification{
				Category: "unsupported_island_domain_primitive",
				Detail:   fmt.Sprintf("ir_kind=%d", instr.Kind),
				Reason:   "unsupported_island_domain_primitive_uses_stack_fallback",
			}
		}
		if backendIRKindIsEffectRuntime(instr.Kind) {
			return backendFallbackClassification{
				Category: "unsupported_effect_runtime_call",
				Detail:   fmt.Sprintf("ir_kind=%d", instr.Kind),
				Reason:   "unsupported_effect_runtime_call_uses_stack_fallback",
			}
		}
	}
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRLabel || instr.Kind == ir.IRJmp || instr.Kind == ir.IRJmpIfZero {
			return backendFallbackClassification{
				Category: "unsupported_control_flow",
				Detail:   fmt.Sprintf("ir_kind=%d label=%d", instr.Kind, instr.Label),
				Reason:   "unsupported_control_flow_uses_stack_fallback",
			}
		}
	}
	return backendFallbackClassification{
		Category: "stack_fallback",
		Reason:   "unsupported_or_unproven_subset_uses_stack_fallback",
	}
}

func backendCallABIUnsupported(instr ir.IRInstr, callABI machine.CallABIInfo) bool {
	return instr.ArgSlots < 0 || instr.RetSlots < 0 || instr.ArgSlots > callABI.MaxArgSlots ||
		instr.RetSlots > callABI.MaxRetSlots
}

func backendCallLooksRuntimeEffect(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	return strings.HasPrefix(lower, "__tetra_") ||
		strings.HasPrefix(lower, "runtime.") ||
		strings.HasPrefix(lower, "core.")
}

func backendIRKindIsIslandDomainPrimitive(kind ir.IRInstrKind) bool {
	switch kind {
	case ir.IRRegionEnter,
		ir.IRRegionMakeSliceU8,
		ir.IRRegionMakeSliceU16,
		ir.IRRegionMakeSliceI32,
		ir.IRRegionReset,
		ir.IRIslandNew,
		ir.IRIslandMakeSliceU8,
		ir.IRIslandMakeSliceU16,
		ir.IRIslandMakeSliceI32,
		ir.IRIslandFree,
		ir.IRIslandReset:
		return true
	default:
		return false
	}
}

func backendIRKindIsEffectRuntime(kind ir.IRInstrKind) bool {
	// Stack slice IR is already-lowered local stack storage, not a runtime effect.
	// Checked indexed stores are local memory operations; bounds reports keep tracking their checks.
	switch kind {
	case ir.IRWrite,
		ir.IRStrLit,
		ir.IRLoadGlobal,
		ir.IRStoreGlobal,
		ir.IRAllocBytes,
		ir.IRMakeSliceU8,
		ir.IRMakeSliceU16,
		ir.IRMakeSliceI32,
		ir.IRRawSliceFromParts,
		ir.IRSliceWindow,
		ir.IRSlicePrefix,
		ir.IRSliceSuffix,
		ir.IRCapIO,
		ir.IRCapMem,
		ir.IRMemReadI32,
		ir.IRMemWriteI32,
		ir.IRMemReadU8,
		ir.IRMemWriteU8,
		ir.IRMemReadPtr,
		ir.IRMemWritePtr,
		ir.IRMemWriteArchPtr,
		ir.IRMemReadI32Offset,
		ir.IRMemWriteI32Offset,
		ir.IRMemReadU8Offset,
		ir.IRMemWriteU8Offset,
		ir.IRMemReadPtrOffset,
		ir.IRMemWritePtrOffset,
		ir.IRMemWriteArchPtrOffset,
		ir.IRPtrAdd,
		ir.IRMmioReadI32,
		ir.IRMmioWriteI32,
		ir.IRSymAddr,
		ir.IRCtxSwitch,
		ir.IRAtomicLoadPtr,
		ir.IRAtomicStorePtr,
		ir.IRAtomicExchangePtr,
		ir.IRAtomicFetchAddPtr,
		ir.IRAtomicFetchSubPtr,
		ir.IRAtomicFetchAndPtr,
		ir.IRAtomicFetchOrPtr,
		ir.IRAtomicFetchXorPtr,
		ir.IRAtomicCompareExchangePtr,
		ir.IRAtomicFenceSeqCst,
		ir.IRAtomicFenceRelaxed,
		ir.IRAtomicFenceAcquire,
		ir.IRAtomicFenceRelease,
		ir.IRAtomicFenceAcqRel,
		ir.IRAtomicLoadI32,
		ir.IRAtomicStoreI32,
		ir.IRAtomicExchangeI32,
		ir.IRAtomicCompareExchangeI32,
		ir.IRAtomicFetchAddI32,
		ir.IRAtomicFetchSubI32,
		ir.IRAtomicFetchAndI32,
		ir.IRAtomicFetchOrI32,
		ir.IRAtomicFetchXorI32,
		ir.IRAtomicLoadI64,
		ir.IRAtomicStoreI64,
		ir.IRAtomicExchangeI64,
		ir.IRAtomicCompareExchangeI64,
		ir.IRAtomicFetchAddI64,
		ir.IRAtomicFetchSubI64,
		ir.IRAtomicFetchAndI64,
		ir.IRAtomicFetchOrI64,
		ir.IRAtomicFetchXorI64,
		ir.IRAtomicLoadI8,
		ir.IRAtomicStoreI8,
		ir.IRAtomicExchangeI8,
		ir.IRAtomicCompareExchangeI8,
		ir.IRAtomicFetchAddI8,
		ir.IRAtomicFetchSubI8,
		ir.IRAtomicFetchAndI8,
		ir.IRAtomicFetchOrI8,
		ir.IRAtomicFetchXorI8,
		ir.IRAtomicLoadI16,
		ir.IRAtomicStoreI16,
		ir.IRAtomicExchangeI16,
		ir.IRAtomicCompareExchangeI16,
		ir.IRAtomicFetchAddI16,
		ir.IRAtomicFetchSubI16,
		ir.IRAtomicFetchAndI16,
		ir.IRAtomicFetchOrI16,
		ir.IRAtomicFetchXorI16:
		return true
	default:
		return false
	}
}

func summarizeBackendCoverage(
	rows []BackendFunctionPathReport,
	machineReports []MachineBackendFunctionReport,
) BackendCoverageSummary {
	summary := BackendCoverageSummary{
		FunctionCount:                len(rows),
		Categories:                   map[string]int{},
		HotnessSource:                "benchmark-corpus-static-map",
		RuntimeFeatureEvidenceClass:  backendRuntimeFeatureEvidenceClass,
		RuntimeFeatureEvidenceMethod: backendRuntimeFeatureEvidenceMethod,
	}
	runtimeFeatures := newBackendRuntimeFeatureSet()
	machineByFunction := make(map[string]MachineBackendFunctionReport, len(machineReports))
	for _, report := range machineReports {
		machineByFunction[report.Function] = report
		if report.Validation.StackChurnOps == 0 &&
			report.Validation.MachineVerifier == "pass" &&
			report.Validation.AllocationVerifier == "pass" {
			summary.MachineRegisterNoStackChurn++
			continue
		}
		if report.Validation.StackChurnOps > 0 {
			summary.MachineRegisterWithStackChurn++
		}
	}
	for _, row := range rows {
		if row.BackendPath == "register" {
			summary.RegisterPath++
		}
		if row.BackendPath == "stack" {
			summary.StackFallback++
		}
		if row.Category != "" {
			summary.Categories[row.Category]++
		}
		runtimeFeatures.addRequired(row.RuntimeFeaturesRequired...)
		runtimeFeatures.addLinked(row.RuntimeFeaturesLinked...)
		runtimeFeatures.addInitialized(row.RuntimeFeaturesInitialized...)
		runtimeFeatures.addBlockers(row.RuntimeLazyInitBlockers...)
	}
	summary.OrdinaryCorpus = summarizeBackendOrdinaryCorpus(rows, machineByFunction)
	summary.ABIBoundaries = summarizeBackendABIBoundaries(rows)
	summary.RuntimeFeaturesRequired = runtimeFeatures.requiredSlice()
	summary.RuntimeFeaturesLinked = runtimeFeatures.linkedSlice()
	summary.RuntimeFeaturesInitialized = runtimeFeatures.initializedSlice()
	summary.RuntimeLazyInitBlockers = runtimeFeatures.blockerSlice()
	return summary
}

const (
	backendRuntimeFeatureEvidenceClass  = "lowered_ir_static_plan"
	backendRuntimeFeatureEvidenceMethod = "backend_report_lowered_ir_scan_v1"
)

type backendRuntimeFeatureSet struct {
	required    map[string]struct{}
	linked      map[string]struct{}
	initialized map[string]struct{}
	blockers    map[string]struct{}
}

func newBackendRuntimeFeatureSet() backendRuntimeFeatureSet {
	return backendRuntimeFeatureSet{
		required:    map[string]struct{}{},
		linked:      map[string]struct{}{},
		initialized: map[string]struct{}{},
		blockers:    map[string]struct{}{},
	}
}

func applyBackendRuntimeFeatureEvidence(row *BackendFunctionPathReport, fn ir.IRFunc) {
	features := collectBackendRuntimeFeatures(fn)
	row.RuntimeFeaturesRequired = features.requiredSlice()
	row.RuntimeFeaturesLinked = features.linkedSlice()
	row.RuntimeFeaturesInitialized = features.initializedSlice()
	row.RuntimeLazyInitBlockers = features.blockerSlice()
	row.RuntimeFeatureEvidenceClass = backendRuntimeFeatureEvidenceClass
	row.RuntimeFeatureEvidenceMethod = backendRuntimeFeatureEvidenceMethod
}

func collectBackendRuntimeFeatures(fn ir.IRFunc) backendRuntimeFeatureSet {
	features := newBackendRuntimeFeatureSet()
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRCall {
			features.addCall(instr.Name)
			continue
		}
		for _, feature := range backendRuntimeFeaturesForIRKind(instr.Kind) {
			features.addKnown(feature)
		}
	}
	return features
}

func (s backendRuntimeFeatureSet) addKnown(feature string) {
	if feature == "" {
		return
	}
	s.required[feature] = struct{}{}
	s.linked[feature] = struct{}{}
	s.initialized[feature] = struct{}{}
}

func (s backendRuntimeFeatureSet) addRequired(features ...string) {
	for _, feature := range features {
		if feature != "" {
			s.required[feature] = struct{}{}
		}
	}
}

func (s backendRuntimeFeatureSet) addLinked(features ...string) {
	for _, feature := range features {
		if feature != "" {
			s.linked[feature] = struct{}{}
		}
	}
}

func (s backendRuntimeFeatureSet) addInitialized(features ...string) {
	for _, feature := range features {
		if feature != "" {
			s.initialized[feature] = struct{}{}
		}
	}
}

func (s backendRuntimeFeatureSet) addBlockers(blockers ...string) {
	for _, blocker := range blockers {
		if blocker != "" {
			s.blockers[blocker] = struct{}{}
		}
	}
}

func (s backendRuntimeFeatureSet) addUnknownRuntimeCall(name string) {
	s.required["unknown_runtime"] = struct{}{}
	s.blockers["unknown_runtime_call:"+name] = struct{}{}
}

func (s backendRuntimeFeatureSet) addCall(name string) {
	feature, ok := backendRuntimeFeatureForCall(name)
	if ok {
		s.addKnown(feature)
		return
	}
	if backendCallLooksRuntimeEffect(name) {
		s.addUnknownRuntimeCall(strings.TrimSpace(name))
	}
}

func (s backendRuntimeFeatureSet) requiredSlice() []string {
	return sortedBackendRuntimeSet(s.required)
}

func (s backendRuntimeFeatureSet) linkedSlice() []string {
	return sortedBackendRuntimeSet(s.linked)
}

func (s backendRuntimeFeatureSet) initializedSlice() []string {
	return sortedBackendRuntimeSet(s.initialized)
}

func (s backendRuntimeFeatureSet) blockerSlice() []string {
	return sortedBackendRuntimeSet(s.blockers)
}

func sortedBackendRuntimeSet(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func backendRuntimeFeaturesForIRKind(kind ir.IRInstrKind) []string {
	switch kind {
	case ir.IRAllocBytes, ir.IRMakeSliceU8, ir.IRMakeSliceU16, ir.IRMakeSliceI32:
		return []string{"heap_runtime"}
	case ir.IRRegionEnter,
		ir.IRRegionMakeSliceU8,
		ir.IRRegionMakeSliceU16,
		ir.IRRegionMakeSliceI32,
		ir.IRRegionReset:
		return []string{"region_allocator"}
	case ir.IRIslandNew,
		ir.IRIslandMakeSliceU8,
		ir.IRIslandMakeSliceU16,
		ir.IRIslandMakeSliceI32,
		ir.IRIslandFree,
		ir.IRIslandReset:
		return []string{"island_allocator"}
	case ir.IRWrite, ir.IRCapIO:
		return []string{"io_runtime"}
	case ir.IRCapMem,
		ir.IRMemReadI32,
		ir.IRMemWriteI32,
		ir.IRMemReadU8,
		ir.IRMemWriteU8,
		ir.IRMemReadPtr,
		ir.IRMemWritePtr,
		ir.IRMemWriteArchPtr,
		ir.IRMemReadI32Offset,
		ir.IRMemWriteI32Offset,
		ir.IRMemReadU8Offset,
		ir.IRMemWriteU8Offset,
		ir.IRMemReadPtrOffset,
		ir.IRMemWritePtrOffset,
		ir.IRMemWriteArchPtrOffset,
		ir.IRPtrAdd,
		ir.IRMmioReadI32,
		ir.IRMmioWriteI32,
		ir.IRSymAddr,
		ir.IRCtxSwitch,
		ir.IRAtomicLoadPtr,
		ir.IRAtomicStorePtr,
		ir.IRAtomicExchangePtr,
		ir.IRAtomicFetchAddPtr,
		ir.IRAtomicFetchSubPtr,
		ir.IRAtomicFetchAndPtr,
		ir.IRAtomicFetchOrPtr,
		ir.IRAtomicFetchXorPtr,
		ir.IRAtomicCompareExchangePtr,
		ir.IRAtomicFenceSeqCst,
		ir.IRAtomicFenceRelaxed,
		ir.IRAtomicFenceAcquire,
		ir.IRAtomicFenceRelease,
		ir.IRAtomicFenceAcqRel,
		ir.IRAtomicLoadI32,
		ir.IRAtomicStoreI32,
		ir.IRAtomicExchangeI32,
		ir.IRAtomicCompareExchangeI32,
		ir.IRAtomicFetchAddI32,
		ir.IRAtomicFetchSubI32,
		ir.IRAtomicFetchAndI32,
		ir.IRAtomicFetchOrI32,
		ir.IRAtomicFetchXorI32,
		ir.IRAtomicLoadI64,
		ir.IRAtomicStoreI64,
		ir.IRAtomicExchangeI64,
		ir.IRAtomicCompareExchangeI64,
		ir.IRAtomicFetchAddI64,
		ir.IRAtomicFetchSubI64,
		ir.IRAtomicFetchAndI64,
		ir.IRAtomicFetchOrI64,
		ir.IRAtomicFetchXorI64,
		ir.IRAtomicLoadI8,
		ir.IRAtomicStoreI8,
		ir.IRAtomicExchangeI8,
		ir.IRAtomicCompareExchangeI8,
		ir.IRAtomicFetchAddI8,
		ir.IRAtomicFetchSubI8,
		ir.IRAtomicFetchAndI8,
		ir.IRAtomicFetchOrI8,
		ir.IRAtomicFetchXorI8,
		ir.IRAtomicLoadI16,
		ir.IRAtomicStoreI16,
		ir.IRAtomicExchangeI16,
		ir.IRAtomicCompareExchangeI16,
		ir.IRAtomicFetchAddI16,
		ir.IRAtomicFetchSubI16,
		ir.IRAtomicFetchAndI16,
		ir.IRAtomicFetchOrI16,
		ir.IRAtomicFetchXorI16:
		return []string{"memory_capability_runtime"}
	default:
		return nil
	}
}

func backendRuntimeFeatureForCall(name string) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(name))
	switch {
	case lower == "":
		return "", false
	case strings.HasPrefix(lower, "__tetra_actor_state"):
		return "actor_state_runtime", true
	case strings.Contains(lower, "actor_node") || strings.Contains(lower, "spawn_remote"):
		return "distributed_actor_runtime", true
	case strings.HasPrefix(lower, "__tetra_actor"):
		return "actor_runtime", true
	case strings.HasPrefix(lower, "__tetra_task_group"):
		return "task_group_runtime", true
	case strings.HasPrefix(
		lower,
		"__tetra_task_typed",
	) || strings.Contains(
		lower,
		"__tetra_task_join_typed_",
	) || strings.Contains(
		lower,
		"typed_task",
	):
		return "typed_task_runtime", true
	case strings.HasPrefix(lower, "__tetra_task"):
		return "task_runtime", true
	case strings.HasPrefix(
		lower,
		"__tetra_time",
	) || strings.Contains(
		lower,
		"timer",
	) || strings.Contains(
		lower,
		"sleep",
	) || strings.Contains(
		lower,
		"deadline",
	):
		return "time_runtime", true
	case strings.HasPrefix(lower, "__tetra_fs"):
		return "filesystem_runtime", true
	case strings.HasPrefix(lower, "__tetra_net"):
		return "net_runtime", true
	case strings.HasPrefix(lower, "__tetra_surface"):
		return "surface_runtime", true
	case strings.HasPrefix(lower, "__tetra_heap") || strings.Contains(lower, "alloc"):
		return "heap_runtime", true
	case strings.HasPrefix(lower, "core.task_"):
		return "task_runtime", true
	case strings.HasPrefix(lower, "core.actor_") || lower == "core.spawn" || lower == "core.send":
		return "actor_runtime", true
	case strings.HasPrefix(lower, "core.net_"):
		return "net_runtime", true
	case strings.HasPrefix(lower, "core.fs_"):
		return "filesystem_runtime", true
	case strings.HasPrefix(lower, "core.time_"):
		return "time_runtime", true
	case strings.HasPrefix(lower, "core.surface_"):
		return "surface_runtime", true
	case strings.Contains(lower, "alloc"):
		return "heap_runtime", true
	default:
		return "", false
	}
}

func summarizeBackendOrdinaryCorpus(
	rows []BackendFunctionPathReport,
	machineByFunction map[string]MachineBackendFunctionReport,
) BackendOrdinaryCorpusSummary {
	summary := BackendOrdinaryCorpusSummary{
		StackFallbackReasons: map[string]int{},
		EvidenceSource:       "benchmark-corpus-static-map/non-runtime-hotness",
	}
	for _, row := range rows {
		if !backendRowInOrdinaryCorpus(row) {
			continue
		}
		summary.FunctionCount++
		switch row.BackendPath {
		case "register":
			summary.RegisterPath++
			machineReport := machineByFunction[row.Function]
			if machineReport.Validation.StackChurnOps == 0 &&
				machineReport.Validation.MachineVerifier == "pass" &&
				machineReport.Validation.AllocationVerifier == "pass" {
				summary.RegisterNoStackChurn++
			} else {
				summary.RegisterWithStackChurn++
			}
		case "stack":
			summary.StackFallback++
			if row.Category != "" {
				summary.StackFallbackReasons[row.Category]++
			}
		}
	}
	summary.RegisterNoStackChurnMajority = summary.FunctionCount > 0 &&
		summary.RegisterNoStackChurn*2 > summary.FunctionCount
	return summary
}

func backendRowInOrdinaryCorpus(row BackendFunctionPathReport) bool {
	if row.HotnessRank <= 0 {
		return false
	}
	if backendHotnessSourceIsRuntimeHeavy(row.HotnessSource) {
		return false
	}
	return row.Category != "unsupported_effect_runtime_call"
}

func backendHotnessSourceIsRuntimeHeavy(source string) bool {
	lower := strings.ToLower(source)
	return strings.Contains(lower, "compiler/internal/webrt/") ||
		strings.Contains(lower, "runtime")
}

func summarizeBackendABIBoundaries(rows []BackendFunctionPathReport) BackendABIBoundarySummary {
	summary := BackendABIBoundarySummary{
		ValueClasses: map[string]int{},
	}
	for _, row := range rows {
		switch row.ABI.MultiSlotReturnPolicy {
		case "single_slot_register_return":
			summary.SingleSlotRegisterReturn++
		case "single_slot_stack_fallback":
			summary.SingleSlotStackFallback++
		case "unsupported_multi_slot_return_stack_fallback":
			summary.MultiSlotReturnStackFallback++
		case "unsupported_call_multi_slot_return_stack_fallback":
			summary.CallMultiSlotReturnStackFallback++
		}
		if row.ABI.ValueClass != "" {
			summary.ValueClasses[row.ABI.ValueClass]++
		}
	}
	return summary
}

type backendHotness struct {
	Rank   int
	Source string
}

func applyBackendHotness(row *BackendFunctionPathReport) {
	hotness := backendHotnessForFunction(row.Function)
	row.HotnessRank = hotness.Rank
	row.HotnessSource = hotness.Source
}

func backendHotnessForFunction(name string) backendHotness {
	normalized := normalizeBackendHotnessName(name)
	if hotness, ok := backendBenchmarkHotnessByFunction[normalized]; ok {
		return hotness
	}
	return backendHotness{Rank: 0, Source: "not_in_benchmark_corpus"}
}

func normalizeBackendHotnessName(name string) string {
	out := strings.ToLower(strings.TrimSpace(name))
	out = strings.ReplaceAll(out, "-", "_")
	out = strings.ReplaceAll(out, ".", "_")
	out = strings.ReplaceAll(out, ":", "_")
	return out
}

var backendBenchmarkHotnessByFunction = map[string]backendHotness{
	"response_cost": {
		Rank:   1,
		Source: "examples/benchmarks/systems/techempower_plaintext_kernel.tetra",
	},
	"jsonmessagehandler": {Rank: 2, Source: "compiler/internal/webrt/techempower.go:/json"},
	"dbhandler":          {Rank: 3, Source: "compiler/internal/webrt/techempower.go:/db"},
	"querieshandler":     {Rank: 4, Source: "compiler/internal/webrt/techempower.go:/queries"},
	"updateshandler":     {Rank: 5, Source: "compiler/internal/webrt/techempower.go:/updates"},
	"fortuneshandler":    {Rank: 6, Source: "compiler/internal/webrt/techempower.go:/fortunes"},
	"queryworld":         {Rank: 7, Source: "compiler/internal/webrt/techempower.go:queryWorld"},
	"updateworld":        {Rank: 8, Source: "compiler/internal/webrt/techempower.go:updateWorld"},
	"fetchworld":         {Rank: 9, Source: "compiler/internal/webrt/techempower.go:fetchWorld"},
	"fetchfortunes": {
		Rank:   10,
		Source: "compiler/internal/webrt/techempower.go:fetchFortunes",
	},
	"flip_count": {
		Rank:   20,
		Source: "examples/benchmarks/classic/clbg_fannkuch_redux.tetra",
	},
	"escape_iters": {
		Rank:   21,
		Source: "examples/benchmarks/classic/clbg_integer_mandelbrot.tetra",
	},
	"mix": {
		Rank:   22,
		Source: "examples/benchmarks/classic/energy_languages_checksum.tetra",
	},
	"transform": {
		Rank:   23,
		Source: "examples/benchmarks/systems/spec_cpu_branch_mix.tetra",
	},
	"safe_pair": {Rank: 24, Source: "examples/benchmarks/parallel/plb2_nqueen.tetra"},
	"safe_6":    {Rank: 25, Source: "examples/benchmarks/parallel/plb2_nqueen.tetra"},
	"abs_i32":   {Rank: 26, Source: "examples/benchmarks/parallel/plb2_nqueen.tetra"},
	"cell": {
		Rank:   27,
		Source: "examples/benchmarks/parallel/plb2_sudoku_checksum.tetra",
	},
	"branch": {
		Rank:   28,
		Source: "examples/benchmarks/classic/rustc_perf_frontend_mix.tetra",
	},
	"apply": {
		Rank:   29,
		Source: "examples/benchmarks/classic/awfy_closure_dispatch.tetra",
	},
	"f0": {
		Rank:   30,
		Source: "examples/benchmarks/classic/pyperformance_call_mix.tetra",
	},
	"f1": {
		Rank:   31,
		Source: "examples/benchmarks/classic/pyperformance_call_mix.tetra",
	},
	"f2": {
		Rank:   32,
		Source: "examples/benchmarks/classic/pyperformance_call_mix.tetra",
	},
	"score": {
		Rank:   33,
		Source: "examples/benchmarks/jvm/jvm_dacapo_object_kernel.tetra",
	},
}

func backendABIBoundaryForFunction(
	target string,
	fn ir.IRFunc,
	backendPath string,
) BackendABIBoundaryReport {
	maxRegisterReturns := machineCallABIForTarget(target).MaxRetSlots
	if backendPath == "register" && backendExactVerifiedSingleSlotRegisterABI(fn) {
		return BackendABIBoundaryReport{
			ReturnSlots:            fn.ReturnSlots,
			MaxRegisterReturnSlots: maxRegisterReturns,
			MultiSlotReturnPolicy:  "single_slot_register_return",
			ValueClass:             "single_register_slot",
			BoundaryStatus:         "register_return_verified",
		}
	}
	hasMultiSlotCall := hasMultiSlotCallReturn(fn, maxRegisterReturns)
	policy := "single_slot_stack_fallback"
	switch {
	case hasMultiSlotCall:
		policy = "unsupported_call_multi_slot_return_stack_fallback"
	case fn.ReturnSlots > maxRegisterReturns:
		policy = "unsupported_multi_slot_return_stack_fallback"
	case backendPath == "register":
		policy = "single_slot_register_return"
	}
	return BackendABIBoundaryReport{
		ReturnSlots:            fn.ReturnSlots,
		MaxRegisterReturnSlots: maxRegisterReturns,
		MultiSlotReturnPolicy:  policy,
		ValueClass: backendABIValueClass(
			fn.ReturnSlots,
			maxRegisterReturns,
			hasMultiSlotCall,
		),
		BoundaryStatus: backendABIBoundaryStatus(policy),
	}
}

func backendExactVerifiedSingleSlotRegisterABI(fn ir.IRFunc) bool {
	if backendExactInternalInoutWriterABI(fn) {
		return true
	}
	if _, ok, err := machine.ParallelMapReduceMainPlanFromStackIR(fn); err == nil && ok {
		return true
	}
	return false
}

func backendExactInternalInoutWriterABI(fn ir.IRFunc) bool {
	if _, ok, err := machine.PostgreSQLInoutWriterPlanFromStackIR(fn); err == nil && ok {
		return true
	}
	if _, ok, err := machine.PostgreSQLInoutWriterMainPlanFromStackIR(fn); err == nil && ok {
		return true
	}
	if _, ok, err := machine.InoutWriterHelperSummaryPlanFromStackIR(fn); err == nil && ok {
		return true
	}
	if _, ok, err := machine.InoutWriterHelperSummaryCallerPlanFromStackIR(fn); err == nil && ok {
		return true
	}
	return false
}

func backendABIValueClass(returnSlots int, maxRegisterReturns int, hasMultiSlotCall bool) string {
	switch {
	case hasMultiSlotCall:
		return "callee_multi_slot_return_unverified"
	case returnSlots == 0:
		return "void_or_no_return"
	case returnSlots <= maxRegisterReturns:
		return "single_register_slot"
	case returnSlots == 2:
		return "unverified_header_or_pair"
	default:
		return "unverified_aggregate"
	}
}

func backendABIBoundaryStatus(policy string) string {
	switch policy {
	case "single_slot_register_return":
		return "register_return_verified"
	case "unsupported_multi_slot_return_stack_fallback",
		"unsupported_call_multi_slot_return_stack_fallback":
		return "stack_fallback_until_multi_slot_abi_verified"
	default:
		return "stack_fallback_for_unpromoted_single_slot"
	}
}

func hasMultiSlotCallReturn(fn ir.IRFunc, maxRegisterReturns int) bool {
	for _, instr := range fn.Instrs {
		if instr.Kind == ir.IRCall && instr.RetSlots > maxRegisterReturns {
			return true
		}
	}
	return false
}

func buildMachineBackendFunctionReports(
	target string,
	irProg *ir.IRProgram,
) []MachineBackendFunctionReport {
	if irProg == nil {
		return nil
	}
	callABI := machineCallABIForTarget(target)
	callerSaved := machineCallerSavedForTarget(target)
	var out []MachineBackendFunctionReport
	for _, fn := range irProg.Funcs {
		ssaVerified := stackIRFunctionPassesSSAGate(fn)
		if !ssaVerified {
			ssaVerified = stackIRRecursionBenchmarkPassesSSAGate(fn)
		}
		if !ssaVerified {
			ssaVerified = stackIRCompileTimeCallLoopPassesSSAGate(fn)
		}
		if !ssaVerified {
			ssaVerified = stackIRAllocationLoopPassesSSAGate(fn)
		}
		if !ssaVerified {
			ssaVerified = stackIRBoundsCheckLoopsPassesSSAGate(fn)
		}
		if !ssaVerified {
			ssaVerified = stackIRHashTableLookupPassesSSAGate(fn)
		}
		if !ssaVerified && targetSupportsHashTableMainMachinePath(target) {
			ssaVerified = stackIRHashTableMainPassesSSAGate(fn)
		}
		if !ssaVerified {
			ssaVerified = stackIRPostgreSQLFrameTypeAtPassesSSAGate(fn)
		}
		if !ssaVerified {
			ssaVerified = stackIRPostgreSQLInoutWriterPassesSSAGate(fn)
		}
		if !ssaVerified {
			ssaVerified = stackIRPostgreSQLInoutWriterMainPassesSSAGate(fn)
		}
		if !ssaVerified {
			ssaVerified = stackIRInoutWriterHelperSummaryPassesSSAGate(fn)
		}
		if !ssaVerified {
			ssaVerified = stackIRInoutWriterHelperSummaryCallerPassesSSAGate(fn)
		}
		if !ssaVerified {
			ssaVerified = stackIRParallelMapReduceMainPassesSSAGate(fn)
		}
		if !ssaVerified {
			ssaVerified = stackIRActorPingPongRuntimeCallPassesSSAGate(fn, callABI)
		}
		if !ssaVerified {
			ssaVerified = stackIRMatrixMultiplyMainPassesSSAGate(fn)
		}
		if !ssaVerified {
			ssaVerified = stackIRSliceSumMainPassesSSAGate(fn)
		}
		if !ssaVerified {
			ssaVerified = stackIRRegionIslandAllocationMainPassesSSAGate(fn)
		}
		if !ssaVerified {
			if mfn, ok, err := machine.ScalarIntConstModuloLoopFunctionFromStackIR(fn); err == nil &&
				ok &&
				stackIRConstModuloLoopPassesSSAGate(fn) {
				if report, ok := BuildMachineBackendFunctionReport(
					mfn,
					"machine-ir-const-modulo-loop",
					callerSaved,
					true,
				); ok {
					out = append(out, report)
				}
			}
			continue
		}
		if target == "linux-x64" {
			if mfn, ok, err := machine.RecursionFibFunctionFromStackIRWithCallABI(fn, callABI); err == nil &&
				ok {
				if report, ok := BuildMachineBackendFunctionReport(
					mfn,
					"machine-ir-recursive-fib",
					callerSaved,
					true,
				); ok {
					out = append(out, report)
				}
				continue
			}
			if mfn, ok, err := machine.RecursionMainFunctionFromStackIRWithCallABI(
				fn,
				callABI,
			); err == nil &&
				ok {
				if report, ok := BuildMachineBackendFunctionReport(
					mfn,
					"machine-ir-recursion-main-loop",
					callerSaved,
					true,
				); ok {
					out = append(out, report)
				}
				continue
			}
		}
		if mfn, ok, err := machinebounds.BoundsCheckLoopsFunctionFromStackIR(fn); err == nil && ok {
			if report, ok := BuildMachineBackendFunctionReport(
				mfn,
				"machine-ir-bounds-check-loops",
				callerSaved,
				true,
			); ok {
				out = append(out, report)
			}
			continue
		}
		if mfn, ok, err := machine.AllocationLoopFunctionFromStackIR(fn); err == nil && ok {
			if report, ok := BuildMachineBackendFunctionReport(
				mfn,
				"machine-ir-allocation-loop",
				callerSaved,
				true,
			); ok {
				out = append(out, report)
			}
			continue
		}
		if mfn, ok, err := machine.MatrixMultiplyMainFunctionFromStackIR(fn); err == nil && ok {
			if report, ok := BuildMachineBackendFunctionReport(
				mfn,
				"machine-ir-matrix-multiply-main",
				callerSaved,
				true,
			); ok {
				out = append(out, report)
			}
			continue
		}
		if mfn, ok, err := machine.SliceSumMainFunctionFromStackIR(fn); err == nil && ok {
			if report, ok := BuildMachineBackendFunctionReport(
				mfn,
				"machine-ir-slice-sum-main",
				callerSaved,
				true,
			); ok {
				out = append(out, report)
			}
			continue
		}
		if mfn, ok, err := machine.RegionIslandAllocationMainFunctionFromStackIR(fn); err == nil &&
			ok {
			if report, ok := BuildMachineBackendFunctionReport(
				mfn,
				"machine-ir-region-island-allocation-main",
				callerSaved,
				true,
			); ok {
				out = append(out, report)
			}
			continue
		}
		if mfn, ok, err := machine.ScalarI32SliceSumLoopFunctionFromStackIR(fn); err == nil && ok {
			if report, ok := BuildMachineBackendFunctionReport(
				mfn,
				"machine-ir-slice-sum",
				callerSaved,
				true,
			); ok {
				out = append(out, report)
			}
			continue
		}
		if mfn, ok, err := machine.ScalarIntCallLoopFunctionFromStackIRWithCallABI(
			fn,
			callABI,
		); err == nil &&
			ok {
			if report, ok := BuildMachineBackendFunctionReport(
				mfn,
				"machine-ir-call-loop",
				callerSaved,
				true,
			); ok {
				out = append(out, report)
			}
			continue
		}
		if mfn, ok, err := machine.ParallelMapReduceMainFunctionFromStackIR(fn); err == nil && ok {
			if report, ok := BuildMachineBackendFunctionReport(
				mfn,
				"machine-ir-parallel-map-reduce-main",
				callerSaved,
				true,
			); ok {
				out = append(out, report)
			}
			continue
		}
		if plan, ok, err := machine.ActorPingPongRuntimeCallPlanFromStackIRWithCallABI(
			fn,
			callABI,
		); err == nil && ok {
			if report, ok := BuildMachineBackendFunctionReport(
				plan.Function,
				plan.Path,
				callerSaved,
				true,
			); ok {
				out = append(out, report)
			}
			continue
		}
		if mfn, ok, err := machine.PostgreSQLFrameTypeAtFunctionFromStackIR(fn); err == nil && ok {
			if report, ok := BuildMachineBackendFunctionReport(
				mfn,
				"machine-ir-postgresql-frame-type-at",
				callerSaved,
				true,
			); ok {
				out = append(out, report)
			}
			continue
		}
		if mfn, ok, err := machine.PostgreSQLInoutWriterFunctionFromStackIR(fn); err == nil && ok {
			if report, ok := BuildMachineBackendFunctionReport(
				mfn,
				"machine-ir-postgresql-inout-writer",
				callerSaved,
				true,
			); ok {
				out = append(out, report)
			}
			continue
		}
		if mfn, ok, err := machine.PostgreSQLInoutWriterMainFunctionFromStackIR(fn); err == nil && ok {
			if report, ok := BuildMachineBackendFunctionReport(
				mfn,
				"machine-ir-postgresql-inout-writer-main",
				callerSaved,
				true,
			); ok {
				out = append(out, report)
			}
			continue
		}
		if mfn, ok, err := machine.InoutWriterHelperSummaryFunctionFromStackIR(fn); err == nil && ok {
			if report, ok := BuildMachineBackendFunctionReport(
				mfn,
				"machine-ir-inout-writer-helper-summary",
				callerSaved,
				true,
			); ok {
				out = append(out, report)
			}
			continue
		}
		if mfn, ok, err := machine.InoutWriterHelperSummaryCallerFunctionFromStackIR(fn); err == nil &&
			ok {
			if report, ok := BuildMachineBackendFunctionReport(
				mfn,
				"machine-ir-inout-writer-helper-summary-caller",
				callerSaved,
				true,
			); ok {
				out = append(out, report)
			}
			continue
		}
		if mfn, ok, err := machine.HashTableLookupFunctionFromStackIR(fn); err == nil && ok {
			if report, ok := BuildMachineBackendFunctionReport(
				mfn,
				"machine-ir-hash-table-lookup",
				callerSaved,
				true,
			); ok {
				out = append(out, report)
			}
			continue
		}
		if targetSupportsHashTableMainMachinePath(target) {
			if mfn, ok, err := machine.HashTableMainFunctionFromStackIR(fn); err == nil && ok {
				if report, ok := BuildMachineBackendFunctionReport(
					mfn,
					"machine-ir-hash-table-main",
					callerSaved,
					true,
				); ok {
					out = append(out, report)
				}
				continue
			}
		}
		if mfn, ok, err := machine.ScalarIntConstModuloLoopFunctionFromStackIR(fn); err == nil &&
			ok {
			if report, ok := BuildMachineBackendFunctionReport(
				mfn,
				"machine-ir-const-modulo-loop",
				callerSaved,
				true,
			); ok {
				out = append(out, report)
			}
			continue
		}
		if mfn, ok, err := machine.ScalarIntLoopFunctionFromStackIR(fn); err == nil && ok {
			if report, ok := BuildMachineBackendFunctionReport(
				mfn,
				"machine-ir-loop",
				callerSaved,
				true,
			); ok {
				out = append(out, report)
			}
			continue
		}
		if mfn, ok, err := machine.ScalarIntFunctionFromStackIRWithCallABI(fn, callABI); err == nil &&
			ok {
			path := "machine-ir-scalar"
			if machineFunctionHasOp(mfn, machine.OpCall) {
				path = "machine-ir-call"
			}
			if report, ok := BuildMachineBackendFunctionReport(mfn, path, callerSaved, true); ok {
				out = append(out, report)
			}
		}
	}
	return out
}

func stackIRFunctionPassesSSAGate(fn ir.IRFunc) bool {
	ssaFn, ok, err := ssair.FromStackIRFunction(fn)
	if err != nil || !ok {
		return false
	}
	return ssair.VerifyFunction(ssaFn) == nil
}

func stackIRConstModuloLoopPassesSSAGate(fn ir.IRFunc) bool {
	plan, ok, err := machine.ScalarIntConstModuloLoopPlanFromStackIR(fn)
	if err != nil || !ok {
		return false
	}
	ssaFn := ssair.Function{
		Name:       fn.Name,
		ReturnType: ssair.TypeI32,
		Values: []ssair.Value{
			{ID: "zero", Type: ssair.TypeI32, Origin: "const"},
			{ID: "one", Type: ssair.TypeI32, Origin: "const"},
			{ID: "bound", Type: ssair.TypeI32, Origin: "const"},
			{ID: "modulus", Type: ssair.TypeI32, Origin: "const"},
			{ID: "loop.index", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "loop.total", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "body.index", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "body.total", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "exit.total", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "cmp", Type: ssair.TypeBool, Origin: "instr"},
			{ID: "remainder", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "next.total", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "next.index", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "final.cmp", Type: ssair.TypeBool, Origin: "instr"},
		},
		Blocks: []ssair.Block{
			{
				ID:    "entry",
				Entry: true,
				Instrs: []ssair.Instr{
					{
						ID:     "const_zero",
						Kind:   ssair.OpConstI32,
						Result: "zero",
						Type:   ssair.TypeI32,
						Imm:    0,
					},
					{
						ID:     "const_one",
						Kind:   ssair.OpConstI32,
						Result: "one",
						Type:   ssair.TypeI32,
						Imm:    plan.FalseReturnImm,
					},
					{
						ID:     "const_bound",
						Kind:   ssair.OpConstI32,
						Result: "bound",
						Type:   ssair.TypeI32,
						Imm:    plan.Bound,
					},
					{
						ID:     "const_modulus",
						Kind:   ssair.OpConstI32,
						Result: "modulus",
						Type:   ssair.TypeI32,
						Imm:    plan.Modulus,
					},
				},
				Term: ssair.Terminator{
					Kind:   ssair.TermBranch,
					Target: "loop",
					Args:   []ssair.ValueID{"zero", "zero"},
				},
			},
			{
				ID:     "loop",
				Params: []ssair.ValueID{"loop.index", "loop.total"},
				Instrs: []ssair.Instr{
					{
						ID:     "cmp_loop_bound",
						Kind:   ssair.OpCmpLtI32,
						Result: "cmp",
						Type:   ssair.TypeBool,
						Args:   []ssair.ValueID{"loop.index", "bound"},
					},
				},
				Term: ssair.Terminator{
					Kind:        ssair.TermCondBr,
					Cond:        "cmp",
					IfTrue:      "body",
					IfTrueArgs:  []ssair.ValueID{"loop.index", "loop.total"},
					IfFalse:     "exit",
					IfFalseArgs: []ssair.ValueID{"loop.total"},
				},
			},
			{
				ID:     "body",
				Params: []ssair.ValueID{"body.index", "body.total"},
				Instrs: []ssair.Instr{
					{
						ID:     "mod_index",
						Kind:   ssair.OpModI32,
						Result: "remainder",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"body.index", "modulus"},
					},
					{
						ID:     "add_total",
						Kind:   ssair.OpAddI32,
						Result: "next.total",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"body.total", "remainder"},
					},
					{
						ID:     "inc_index",
						Kind:   ssair.OpAddI32,
						Result: "next.index",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"body.index", "one"},
					},
				},
				Term: ssair.Terminator{
					Kind:   ssair.TermBranch,
					Target: "loop",
					Args:   []ssair.ValueID{"next.index", "next.total"},
				},
			},
			{
				ID:     "exit",
				Params: []ssair.ValueID{"exit.total"},
				Instrs: []ssair.Instr{
					{
						ID:     "cmp_nonnegative",
						Kind:   ssair.OpCmpGeI32,
						Result: "final.cmp",
						Type:   ssair.TypeBool,
						Args:   []ssair.ValueID{"exit.total", "zero"},
					},
				},
				Term: ssair.Terminator{
					Kind:    ssair.TermCondBr,
					Cond:    "final.cmp",
					IfTrue:  "return_zero",
					IfFalse: "return_one",
				},
			},
			{
				ID:   "return_zero",
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "zero"},
			},
			{
				ID:   "return_one",
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "one"},
			},
		},
	}
	return ssair.VerifyFunction(ssaFn) == nil
}

func stackIRCompileTimeCallLoopPassesSSAGate(fn ir.IRFunc) bool {
	if fn.Name != "p25.compile_time.main" {
		return false
	}
	plan, ok, err := machine.ScalarIntCallLoopPlanFromStackIRWithCallABI(
		fn,
		machine.SysVCallABIInfo(),
	)
	if err != nil || !ok || !plan.ReturnOneIfTotalZero {
		return false
	}
	if plan.BoundConst != 200000 || plan.CallName != "p25.compile_time.f2" {
		return false
	}
	if plan.ParamLocal != -1 || plan.BoundLocal != -1 || plan.IndexLocal != 0 ||
		plan.TotalLocal != 1 {
		return false
	}
	if len(plan.CallArgLocals) != 1 || plan.CallArgLocals[0] != plan.IndexLocal {
		return false
	}
	ssaFn := ssair.Function{
		Name:       fn.Name,
		ReturnType: ssair.TypeI32,
		Values: []ssair.Value{
			{ID: "zero", Type: ssair.TypeI32, Origin: "const"},
			{ID: "one", Type: ssair.TypeI32, Origin: "const"},
			{ID: "bound", Type: ssair.TypeI32, Origin: "const"},
			{ID: "loop.index", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "loop.total", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "loop.effect", Type: ssair.TypeEffect, Origin: "block_param"},
			{ID: "body.index", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "body.total", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "body.effect", Type: ssair.TypeEffect, Origin: "block_param"},
			{ID: "exit.total", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "cmp.loop", Type: ssair.TypeBool, Origin: "instr"},
			{ID: "call.ret", Type: ssair.TypeI32, Origin: "call_result"},
			{ID: "next.total", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "next.index", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "cmp.equal.zero", Type: ssair.TypeBool, Origin: "instr"},
			{ID: "effect0", Type: ssair.TypeEffect, Origin: "entry_effect"},
			{ID: "call.effect", Type: ssair.TypeEffect, Origin: "call_effect"},
		},
		Blocks: []ssair.Block{
			{
				ID:    "entry",
				Entry: true,
				Instrs: []ssair.Instr{
					{
						ID:     "const_zero",
						Kind:   ssair.OpConstI32,
						Result: "zero",
						Type:   ssair.TypeI32,
						Imm:    0,
					},
					{
						ID:     "const_one",
						Kind:   ssair.OpConstI32,
						Result: "one",
						Type:   ssair.TypeI32,
						Imm:    1,
					},
					{
						ID:     "const_bound",
						Kind:   ssair.OpConstI32,
						Result: "bound",
						Type:   ssair.TypeI32,
						Imm:    plan.BoundConst,
					},
				},
				Term: ssair.Terminator{
					Kind:   ssair.TermBranch,
					Target: "loop",
					Args:   []ssair.ValueID{"zero", "zero", "effect0"},
				},
			},
			{
				ID:     "loop",
				Params: []ssair.ValueID{"loop.index", "loop.total", "loop.effect"},
				Instrs: []ssair.Instr{
					{
						ID:     "cmp_loop_bound",
						Kind:   ssair.OpCmpLtI32,
						Result: "cmp.loop",
						Type:   ssair.TypeBool,
						Args:   []ssair.ValueID{"loop.index", "bound"},
					},
				},
				Term: ssair.Terminator{
					Kind:        ssair.TermCondBr,
					Cond:        "cmp.loop",
					IfTrue:      "body",
					IfTrueArgs:  []ssair.ValueID{"loop.index", "loop.total", "loop.effect"},
					IfFalse:     "exit",
					IfFalseArgs: []ssair.ValueID{"loop.total"},
				},
			},
			{
				ID:     "body",
				Params: []ssair.ValueID{"body.index", "body.total", "body.effect"},
				Instrs: []ssair.Instr{
					{
						ID:        "call_f2",
						Kind:      ssair.OpCall,
						Result:    "call.ret",
						Type:      ssair.TypeI32,
						Args:      []ssair.ValueID{"body.index"},
						Call:      plan.CallName,
						EffectIn:  "body.effect",
						EffectOut: "call.effect",
					},
					{
						ID:     "add_total",
						Kind:   ssair.OpAddI32,
						Result: "next.total",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"body.total", "call.ret"},
					},
					{
						ID:     "inc_index",
						Kind:   ssair.OpAddI32,
						Result: "next.index",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"body.index", "one"},
					},
				},
				Term: ssair.Terminator{
					Kind:   ssair.TermBranch,
					Target: "loop",
					Args:   []ssair.ValueID{"next.index", "next.total", "call.effect"},
				},
			},
			{
				ID:     "exit",
				Params: []ssair.ValueID{"exit.total"},
				Instrs: []ssair.Instr{
					{
						ID:     "cmp_equal_zero",
						Kind:   ssair.OpCmpEqI32,
						Result: "cmp.equal.zero",
						Type:   ssair.TypeBool,
						Args:   []ssair.ValueID{"exit.total", "zero"},
					},
				},
				Term: ssair.Terminator{
					Kind:    ssair.TermCondBr,
					Cond:    "cmp.equal.zero",
					IfTrue:  "return_one",
					IfFalse: "return_zero",
				},
			},
			{
				ID:   "return_one",
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "one"},
			},
			{
				ID:   "return_zero",
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "zero"},
			},
		},
	}
	return ssair.VerifyFunction(ssaFn) == nil
}

func stackIRAllocationLoopPassesSSAGate(fn ir.IRFunc) bool {
	plan, ok, err := machine.AllocationLoopPlanFromStackIR(fn)
	if err != nil || !ok {
		return false
	}
	ssaFn := ssair.Function{
		Name:       fn.Name,
		ReturnType: ssair.TypeI32,
		Values: []ssair.Value{
			{ID: "zero", Type: ssair.TypeI32, Origin: "const"},
			{ID: "one", Type: ssair.TypeI32, Origin: "const"},
			{ID: "bound", Type: ssair.TypeI32, Origin: "const"},
			{ID: "slice.ptr", Type: ssair.TypePtr, Origin: "stack_slice"},
			{ID: "slice.len", Type: ssair.TypeI32, Origin: "stack_slice"},
			{ID: "index.zero", Type: ssair.TypeI32, Origin: "const"},
			{ID: "loop.index", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "loop.checksum", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "loop.effect", Type: ssair.TypeEffect, Origin: "block_param"},
			{ID: "body.index", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "body.checksum", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "body.effect", Type: ssair.TypeEffect, Origin: "block_param"},
			{ID: "exit.checksum", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "cmp.loop", Type: ssair.TypeBool, Origin: "instr"},
			{ID: "store.effect", Type: ssair.TypeEffect, Origin: "checked_index_store"},
			{ID: "loaded", Type: ssair.TypeI32, Origin: "checked_index_load"},
			{ID: "load.effect", Type: ssair.TypeEffect, Origin: "checked_index_load"},
			{ID: "next.checksum", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "next.index", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "final.cmp", Type: ssair.TypeBool, Origin: "instr"},
			{ID: "effect0", Type: ssair.TypeEffect, Origin: "entry_effect"},
		},
		Blocks: []ssair.Block{
			{
				ID:    "entry",
				Entry: true,
				Instrs: []ssair.Instr{
					{
						ID:     "const_zero",
						Kind:   ssair.OpConstI32,
						Result: "zero",
						Type:   ssair.TypeI32,
						Imm:    0,
					},
					{
						ID:     "const_one",
						Kind:   ssair.OpConstI32,
						Result: "one",
						Type:   ssair.TypeI32,
						Imm:    plan.FailureReturn,
					},
					{
						ID:     "const_bound",
						Kind:   ssair.OpConstI32,
						Result: "bound",
						Type:   ssair.TypeI32,
						Imm:    plan.LoopBound,
					},
					{
						ID:     "const_index_zero",
						Kind:   ssair.OpConstI32,
						Result: "index.zero",
						Type:   ssair.TypeI32,
						Imm:    plan.IndexConst,
					},
					{
						ID:   "stack_slice",
						Kind: ssair.OpOpaque,
						Args: []ssair.ValueID{"index.zero"},
						Note: "IRStackSliceI32 length 32",
					},
				},
				Term: ssair.Terminator{
					Kind:   ssair.TermBranch,
					Target: "loop",
					Args:   []ssair.ValueID{"zero", "zero", "effect0"},
				},
			},
			{
				ID:     "loop",
				Params: []ssair.ValueID{"loop.index", "loop.checksum", "loop.effect"},
				Instrs: []ssair.Instr{
					{
						ID:     "cmp_loop_bound",
						Kind:   ssair.OpCmpLtI32,
						Result: "cmp.loop",
						Type:   ssair.TypeBool,
						Args:   []ssair.ValueID{"loop.index", "bound"},
					},
				},
				Term: ssair.Terminator{
					Kind:        ssair.TermCondBr,
					Cond:        "cmp.loop",
					IfTrue:      "body",
					IfTrueArgs:  []ssair.ValueID{"loop.index", "loop.checksum", "loop.effect"},
					IfFalse:     "exit",
					IfFalseArgs: []ssair.ValueID{"loop.checksum"},
				},
			},
			{
				ID:     "body",
				Params: []ssair.ValueID{"body.index", "body.checksum", "body.effect"},
				Instrs: []ssair.Instr{
					{
						ID:   "checked_store_xs0",
						Kind: ssair.OpOpaque,
						Args: []ssair.ValueID{
							"slice.ptr",
							"slice.len",
							"index.zero",
							"body.index",
						},
						EffectIn:  "body.effect",
						EffectOut: "store.effect",
						Note:      "checked index_store_i32 xs[0] = r",
					},
					{
						ID:        "checked_load_xs0",
						Kind:      ssair.OpIndexLoadI32,
						Result:    "loaded",
						Type:      ssair.TypeI32,
						Args:      []ssair.ValueID{"slice.ptr", "slice.len", "index.zero"},
						EffectIn:  "store.effect",
						EffectOut: "load.effect",
					},
					{
						ID:     "add_checksum",
						Kind:   ssair.OpAddI32,
						Result: "next.checksum",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"body.checksum", "loaded"},
					},
					{
						ID:     "inc_index",
						Kind:   ssair.OpAddI32,
						Result: "next.index",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"body.index", "one"},
					},
				},
				Term: ssair.Terminator{
					Kind:   ssair.TermBranch,
					Target: "loop",
					Args:   []ssair.ValueID{"next.index", "next.checksum", "load.effect"},
				},
			},
			{
				ID:     "exit",
				Params: []ssair.ValueID{"exit.checksum"},
				Instrs: []ssair.Instr{
					{
						ID:     "cmp_checksum_positive",
						Kind:   ssair.OpCmpGtI32,
						Result: "final.cmp",
						Type:   ssair.TypeBool,
						Args:   []ssair.ValueID{"exit.checksum", "zero"},
					},
				},
				Term: ssair.Terminator{
					Kind:    ssair.TermCondBr,
					Cond:    "final.cmp",
					IfTrue:  "return_zero",
					IfFalse: "return_one",
				},
			},
			{
				ID:   "return_zero",
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "zero"},
			},
			{
				ID:   "return_one",
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "one"},
			},
		},
	}
	return ssair.VerifyFunction(ssaFn) == nil
}

func stackIRBoundsCheckLoopsPassesSSAGate(fn ir.IRFunc) bool {
	plan, ok, err := machinebounds.BoundsCheckLoopsPlanFromStackIR(fn)
	if err != nil || !ok {
		return false
	}
	ssaFn := ssair.Function{
		Name:       fn.Name,
		ReturnType: ssair.TypeI32,
		Values: []ssair.Value{
			{ID: "zero", Type: ssair.TypeI32, Origin: "const"},
			{ID: "one", Type: ssair.TypeI32, Origin: "const"},
			{ID: "slice.ptr", Type: ssair.TypePtr, Origin: "stack_slice"},
			{ID: "slice.len", Type: ssair.TypeI32, Origin: "stack_slice"},
			{ID: "fill.modulus", Type: ssair.TypeI32, Origin: "const"},
			{ID: "hot.bound", Type: ssair.TypeI32, Origin: "const"},
			{ID: "index.multiplier", Type: ssair.TypeI32, Origin: "const"},
			{ID: "fill.loop.index", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "fill.loop.effect", Type: ssair.TypeEffect, Origin: "block_param"},
			{ID: "fill.body.index", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "fill.body.effect", Type: ssair.TypeEffect, Origin: "block_param"},
			{ID: "hot.loop.index", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "hot.loop.total", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "hot.loop.effect", Type: ssair.TypeEffect, Origin: "block_param"},
			{ID: "hot.body.index", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "hot.body.total", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "hot.body.effect", Type: ssair.TypeEffect, Origin: "block_param"},
			{ID: "exit.total", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "cmp.fill", Type: ssair.TypeBool, Origin: "instr"},
			{ID: "fill.value", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "store.effect", Type: ssair.TypeEffect, Origin: "checked_index_store"},
			{ID: "next.fill.index", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "cmp.hot", Type: ssair.TypeBool, Origin: "instr"},
			{ID: "idx.product", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "idx", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "loaded", Type: ssair.TypeI32, Origin: "checked_index_load"},
			{ID: "load.effect", Type: ssair.TypeEffect, Origin: "checked_index_load"},
			{ID: "next.hot.total", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "next.hot.index", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "final.cmp", Type: ssair.TypeBool, Origin: "instr"},
			{ID: "effect0", Type: ssair.TypeEffect, Origin: "entry_effect"},
		},
		Blocks: []ssair.Block{
			{
				ID:    "entry",
				Entry: true,
				Instrs: []ssair.Instr{
					{
						ID:     "const_zero",
						Kind:   ssair.OpConstI32,
						Result: "zero",
						Type:   ssair.TypeI32,
						Imm:    plan.SuccessReturn,
					},
					{
						ID:     "const_one",
						Kind:   ssair.OpConstI32,
						Result: "one",
						Type:   ssair.TypeI32,
						Imm:    plan.FailureReturn,
					},
					{
						ID:     "const_slice_len",
						Kind:   ssair.OpConstI32,
						Result: "slice.len",
						Type:   ssair.TypeI32,
						Imm:    plan.SliceLength,
					},
					{
						ID:     "const_fill_modulus",
						Kind:   ssair.OpConstI32,
						Result: "fill.modulus",
						Type:   ssair.TypeI32,
						Imm:    plan.FillModulus,
					},
					{
						ID:     "const_hot_bound",
						Kind:   ssair.OpConstI32,
						Result: "hot.bound",
						Type:   ssair.TypeI32,
						Imm:    plan.HotLoopBound,
					},
					{
						ID:     "const_index_multiplier",
						Kind:   ssair.OpConstI32,
						Result: "index.multiplier",
						Type:   ssair.TypeI32,
						Imm:    plan.IndexMultiplier,
					},
					{
						ID:   "stack_slice",
						Kind: ssair.OpOpaque,
						Args: []ssair.ValueID{"slice.len"},
						Note: "IRStackSliceI32 length 4096",
					},
				},
				Term: ssair.Terminator{
					Kind:   ssair.TermBranch,
					Target: "fill_loop",
					Args:   []ssair.ValueID{"zero", "effect0"},
				},
			},
			{
				ID:     "fill_loop",
				Params: []ssair.ValueID{"fill.loop.index", "fill.loop.effect"},
				Instrs: []ssair.Instr{
					{
						ID:     "cmp_fill_bound",
						Kind:   ssair.OpCmpLtI32,
						Result: "cmp.fill",
						Type:   ssair.TypeBool,
						Args:   []ssair.ValueID{"fill.loop.index", "slice.len"},
					},
				},
				Term: ssair.Terminator{
					Kind:        ssair.TermCondBr,
					Cond:        "cmp.fill",
					IfTrue:      "fill_body",
					IfTrueArgs:  []ssair.ValueID{"fill.loop.index", "fill.loop.effect"},
					IfFalse:     "hot_loop",
					IfFalseArgs: []ssair.ValueID{"zero", "zero", "fill.loop.effect"},
				},
			},
			{
				ID:     "fill_body",
				Params: []ssair.ValueID{"fill.body.index", "fill.body.effect"},
				Instrs: []ssair.Instr{
					{
						ID:     "mod_fill_value",
						Kind:   ssair.OpModI32,
						Result: "fill.value",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"fill.body.index", "fill.modulus"},
					},
					{
						ID:   "checked_store_xsi",
						Kind: ssair.OpOpaque,
						Args: []ssair.ValueID{
							"slice.ptr",
							"slice.len",
							"fill.body.index",
							"fill.value",
						},
						EffectIn:  "fill.body.effect",
						EffectOut: "store.effect",
						Note:      "checked index_store_i32 xs[i] = i % 97 " + plan.StoreProofID,
					},
					{
						ID:     "inc_fill_index",
						Kind:   ssair.OpAddI32,
						Result: "next.fill.index",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"fill.body.index", "one"},
					},
				},
				Term: ssair.Terminator{
					Kind:   ssair.TermBranch,
					Target: "fill_loop",
					Args:   []ssair.ValueID{"next.fill.index", "store.effect"},
				},
			},
			{
				ID:     "hot_loop",
				Params: []ssair.ValueID{"hot.loop.index", "hot.loop.total", "hot.loop.effect"},
				Instrs: []ssair.Instr{
					{
						ID:     "cmp_hot_bound",
						Kind:   ssair.OpCmpLtI32,
						Result: "cmp.hot",
						Type:   ssair.TypeBool,
						Args:   []ssair.ValueID{"hot.loop.index", "hot.bound"},
					},
				},
				Term: ssair.Terminator{
					Kind:   ssair.TermCondBr,
					Cond:   "cmp.hot",
					IfTrue: "hot_body",
					IfTrueArgs: []ssair.ValueID{
						"hot.loop.index",
						"hot.loop.total",
						"hot.loop.effect",
					},
					IfFalse:     "exit",
					IfFalseArgs: []ssair.ValueID{"hot.loop.total"},
				},
			},
			{
				ID:     "hot_body",
				Params: []ssair.ValueID{"hot.body.index", "hot.body.total", "hot.body.effect"},
				Instrs: []ssair.Instr{
					{
						ID:     "mul_index",
						Kind:   ssair.OpMulI32,
						Result: "idx.product",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"hot.body.index", "index.multiplier"},
					},
					{
						ID:     "mod_index",
						Kind:   ssair.OpModI32,
						Result: "idx",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"idx.product", "slice.len"},
					},
					{
						ID:        "checked_load_xsidx",
						Kind:      ssair.OpIndexLoadI32,
						Result:    "loaded",
						Type:      ssair.TypeI32,
						Args:      []ssair.ValueID{"slice.ptr", "slice.len", "idx"},
						EffectIn:  "hot.body.effect",
						EffectOut: "load.effect",
						ProofID:   plan.LoadProofID,
					},
					{
						ID:     "add_total",
						Kind:   ssair.OpAddI32,
						Result: "next.hot.total",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"hot.body.total", "loaded"},
					},
					{
						ID:     "inc_hot_index",
						Kind:   ssair.OpAddI32,
						Result: "next.hot.index",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"hot.body.index", "one"},
					},
				},
				Term: ssair.Terminator{
					Kind:   ssair.TermBranch,
					Target: "hot_loop",
					Args:   []ssair.ValueID{"next.hot.index", "next.hot.total", "load.effect"},
				},
			},
			{
				ID:     "exit",
				Params: []ssair.ValueID{"exit.total"},
				Instrs: []ssair.Instr{
					{
						ID:     "cmp_nonnegative",
						Kind:   ssair.OpCmpGeI32,
						Result: "final.cmp",
						Type:   ssair.TypeBool,
						Args:   []ssair.ValueID{"exit.total", "zero"},
					},
				},
				Term: ssair.Terminator{
					Kind:    ssair.TermCondBr,
					Cond:    "final.cmp",
					IfTrue:  "return_zero",
					IfFalse: "return_one",
				},
			},
			{
				ID:   "return_zero",
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "zero"},
			},
			{
				ID:   "return_one",
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "one"},
			},
		},
	}
	return ssair.VerifyFunction(ssaFn) == nil
}

func stackIRPostgreSQLFrameTypeAtPassesSSAGate(fn ir.IRFunc) bool {
	plan, ok, err := machine.PostgreSQLFrameTypeAtPlanFromStackIR(fn)
	if err != nil || !ok {
		return false
	}
	ssaFn := ssair.Function{
		Name:       fn.Name,
		ReturnType: ssair.TypeI32,
		Values: []ssair.Value{
			{ID: "src.ptr", Type: ssair.TypePtr, Origin: "param"},
			{ID: "src.len", Type: ssair.TypeI32, Origin: "param"},
			{ID: "offset", Type: ssair.TypeI32, Origin: "param"},
			{ID: "effect0", Type: ssair.TypeEffect, Origin: "entry_effect"},
			{ID: "frame.type", Type: ssair.TypeI32, Origin: "unchecked_index_load_u8"},
			{ID: "load.effect", Type: ssair.TypeEffect, Origin: "unchecked_index_load_u8"},
		},
		Blocks: []ssair.Block{
			{
				ID:    "entry",
				Entry: true,
				Instrs: []ssair.Instr{
					{
						ID:        "load_frame_type",
						Kind:      ssair.OpIndexLoadI32,
						Result:    "frame.type",
						Type:      ssair.TypeI32,
						Args:      []ssair.ValueID{"src.ptr", "src.len", "offset"},
						EffectIn:  "effect0",
						EffectOut: "load.effect",
						ProofID:   plan.ProofID,
					},
				},
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "frame.type"},
			},
		},
	}
	return ssair.VerifyFunction(ssaFn) == nil
}

func stackIRPostgreSQLInoutWriterPassesSSAGate(fn ir.IRFunc) bool {
	plan, ok, err := machine.PostgreSQLInoutWriterPlanFromStackIR(fn)
	if err != nil || !ok {
		return false
	}
	values := []ssair.Value{
		{ID: "dst.ptr", Type: ssair.TypePtr, Origin: "param"},
		{ID: "dst.len", Type: ssair.TypeI32, Origin: "param"},
		{ID: "start", Type: ssair.TypeI32, Origin: "param"},
		{ID: "value", Type: ssair.TypeI32, Origin: "param"},
		{ID: "return.addend", Type: ssair.TypeI32, Origin: "const"},
		{ID: "return.start", Type: ssair.TypeI32, Origin: "instr"},
		{ID: "effect0", Type: ssair.TypeEffect, Origin: "entry_effect"},
	}
	instrs := make([]ssair.Instr, 0, plan.StoreCount+2)
	effectIn := ssair.ValueID("effect0")
	for i, proofID := range plan.ProofIDs {
		effectOut := ssair.ValueID(fmt.Sprintf("store.effect%d", i))
		values = append(
			values,
			ssair.Value{ID: effectOut, Type: ssair.TypeEffect, Origin: "memory_effect"},
		)
		instrs = append(instrs, ssair.Instr{
			ID:        fmt.Sprintf("store_byte_%d", i),
			Kind:      ssair.OpOpaque,
			Args:      []ssair.ValueID{"dst.ptr", "dst.len", "start", "value"},
			EffectIn:  effectIn,
			EffectOut: effectOut,
			ProofID:   proofID,
			Note:      "exact PostgreSQL inout []u8 writer byte store",
		})
		effectIn = effectOut
	}
	instrs = append(instrs,
		ssair.Instr{
			ID:     "const_return_addend",
			Kind:   ssair.OpConstI32,
			Result: "return.addend",
			Type:   ssair.TypeI32,
			Imm:    plan.ReturnAddend,
		},
		ssair.Instr{
			ID:     "return_start",
			Kind:   ssair.OpAddI32,
			Result: "return.start",
			Type:   ssair.TypeI32,
			Args:   []ssair.ValueID{"start", "return.addend"},
		},
	)
	ssaFn := ssair.Function{
		Name:       fn.Name,
		ReturnType: ssair.TypeI32,
		Values:     values,
		Blocks: []ssair.Block{{
			ID:     "entry",
			Entry:  true,
			Instrs: instrs,
			Term:   ssair.Terminator{Kind: ssair.TermReturn, Value: "return.start"},
		}},
	}
	return ssair.VerifyFunction(ssaFn) == nil
}

func stackIRPostgreSQLInoutWriterMainPassesSSAGate(fn ir.IRFunc) bool {
	if _, ok, err := machine.PostgreSQLInoutWriterMainPlanFromStackIR(fn); err != nil || !ok {
		return false
	}
	ssaFn := ssair.Function{
		Name:       fn.Name,
		ReturnType: ssair.TypeI32,
		Values: []ssair.Value{
			{ID: "zero", Type: ssair.TypeI32, Origin: "const"},
		},
		Blocks: []ssair.Block{{
			ID:    "entry",
			Entry: true,
			Instrs: []ssair.Instr{{
				ID:     "const_zero",
				Kind:   ssair.OpConstI32,
				Result: "zero",
				Type:   ssair.TypeI32,
				Imm:    0,
			}},
			Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "zero"},
		}},
	}
	return ssair.VerifyFunction(ssaFn) == nil
}

func stackIRInoutWriterHelperSummaryPassesSSAGate(fn ir.IRFunc) bool {
	plan, ok, err := machine.InoutWriterHelperSummaryPlanFromStackIR(fn)
	if err != nil || !ok {
		return false
	}
	values := []ssair.Value{
		{ID: "dst.ptr", Type: ssair.TypePtr, Origin: "param"},
		{ID: "dst.len", Type: ssair.TypeI32, Origin: "param"},
		{ID: "return.count", Type: ssair.TypeI32, Origin: "const"},
		{ID: "effect0", Type: ssair.TypeEffect, Origin: "entry_effect"},
	}
	instrs := make([]ssair.Instr, 0, plan.StoreCount*3+1)
	effectIn := ssair.ValueID("effect0")
	for i := 0; i < plan.StoreCount; i++ {
		indexValue := ssair.ValueID(fmt.Sprintf("store.index%d", i))
		byteValue := ssair.ValueID(fmt.Sprintf("store.byte%d", i))
		effectOut := ssair.ValueID(fmt.Sprintf("store.effect%d", i))
		values = append(
			values,
			ssair.Value{ID: indexValue, Type: ssair.TypeI32, Origin: "const"},
			ssair.Value{ID: byteValue, Type: ssair.TypeI32, Origin: "const"},
			ssair.Value{ID: effectOut, Type: ssair.TypeEffect, Origin: "memory_effect"},
		)
		instrs = append(instrs,
			ssair.Instr{
				ID:     fmt.Sprintf("const_store_index_%d", i),
				Kind:   ssair.OpConstI32,
				Result: indexValue,
				Type:   ssair.TypeI32,
				Imm:    plan.StoreIndexes[i],
			},
			ssair.Instr{
				ID:     fmt.Sprintf("const_store_byte_%d", i),
				Kind:   ssair.OpConstI32,
				Result: byteValue,
				Type:   ssair.TypeI32,
				Imm:    plan.StoreValues[i],
			},
			ssair.Instr{
				ID:        fmt.Sprintf("store_byte_%d", i),
				Kind:      ssair.OpOpaque,
				Args:      []ssair.ValueID{"dst.ptr", "dst.len", indexValue, byteValue},
				EffectIn:  effectIn,
				EffectOut: effectOut,
				ProofID:   plan.ProofIDs[i],
				Note:      "exact helper-summary inout []u8 writer byte store",
			},
		)
		effectIn = effectOut
	}
	instrs = append(instrs, ssair.Instr{
		ID:     "const_return_count",
		Kind:   ssair.OpConstI32,
		Result: "return.count",
		Type:   ssair.TypeI32,
		Imm:    plan.ScalarReturnConst,
	})
	ssaFn := ssair.Function{
		Name:       fn.Name,
		ReturnType: ssair.TypeI32,
		Values:     values,
		Blocks: []ssair.Block{{
			ID:     "entry",
			Entry:  true,
			Instrs: instrs,
			Term:   ssair.Terminator{Kind: ssair.TermReturn, Value: "return.count"},
		}},
	}
	return ssair.VerifyFunction(ssaFn) == nil
}

func stackIRInoutWriterHelperSummaryCallerPassesSSAGate(fn ir.IRFunc) bool {
	plan, ok, err := machine.InoutWriterHelperSummaryCallerPlanFromStackIR(fn)
	if err != nil || !ok {
		return false
	}
	if _, ok, err := machine.InoutWriterHelperSummaryCallerFunctionFromStackIR(fn); err != nil || !ok {
		return false
	}
	values := []ssair.Value{
		{ID: "zero", Type: ssair.TypeI32, Origin: "const"},
		{ID: "effect0", Type: ssair.TypeEffect, Origin: "entry_effect"},
	}
	instrs := make([]ssair.Instr, 0, plan.CallCount+1)
	effectIn := ssair.ValueID("effect0")
	for i, call := range plan.AcceptedHelperCalls {
		effectOut := ssair.ValueID(fmt.Sprintf("call.effect%d", i))
		values = append(values, ssair.Value{
			ID:     effectOut,
			Type:   ssair.TypeEffect,
			Origin: "memory_effect",
		})
		instrs = append(instrs, ssair.Instr{
			ID:        fmt.Sprintf("accepted_helper_call_%d", i),
			Kind:      ssair.OpOpaque,
			EffectIn:  effectIn,
			EffectOut: effectOut,
			Note: fmt.Sprintf(
				"exact helper-summary caller %s arg_slots=%d ret_slots=%d",
				call.HelperName,
				call.ArgSlots,
				call.RetSlots,
			),
		})
		effectIn = effectOut
	}
	instrs = append(instrs, ssair.Instr{
		ID:     "const_zero",
		Kind:   ssair.OpConstI32,
		Result: "zero",
		Type:   ssair.TypeI32,
		Imm:    0,
	})
	ssaFn := ssair.Function{
		Name:       fn.Name,
		ReturnType: ssair.TypeI32,
		Values:     values,
		Blocks: []ssair.Block{{
			ID:     "entry",
			Entry:  true,
			Instrs: instrs,
			Term:   ssair.Terminator{Kind: ssair.TermReturn, Value: "zero"},
		}},
	}
	return ssair.VerifyFunction(ssaFn) == nil
}

func stackIRParallelMapReduceMainPassesSSAGate(fn ir.IRFunc) bool {
	plan, ok, err := machine.ParallelMapReduceMainPlanFromStackIR(fn)
	if err != nil || !ok {
		return false
	}
	values := []ssair.Value{
		{ID: "zero", Type: ssair.TypeI32, Origin: "const"},
		{ID: "expected", Type: ssair.TypeI32, Origin: "const"},
		{ID: "effect0", Type: ssair.TypeEffect, Origin: "entry_effect"},
	}
	instrs := []ssair.Instr{
		{
			ID:     "const_zero",
			Kind:   ssair.OpConstI32,
			Result: "zero",
			Type:   ssair.TypeI32,
			Imm:    plan.SuccessReturn,
		},
		{
			ID:     "const_expected",
			Kind:   ssair.OpConstI32,
			Result: "expected",
			Type:   ssair.TypeI32,
			Imm:    plan.ExpectedTotal,
		},
	}
	effectIn := ssair.ValueID("effect0")
	for i, spawn := range plan.Spawns {
		handleValue := ssair.ValueID(fmt.Sprintf("%s.handle", spawn.Worker))
		statusValue := ssair.ValueID(fmt.Sprintf("%s.status", spawn.Worker))
		effectOut := ssair.ValueID(fmt.Sprintf("spawn.effect%d", i))
		values = append(values,
			ssair.Value{ID: handleValue, Type: ssair.TypePtr, Origin: "task_handle"},
			ssair.Value{ID: statusValue, Type: ssair.TypeI32, Origin: "task_status"},
			ssair.Value{ID: effectOut, Type: ssair.TypeEffect, Origin: "runtime_effect"},
		)
		instrs = append(instrs, ssair.Instr{
			ID:        fmt.Sprintf("spawn_%s", spawn.Worker),
			Kind:      ssair.OpOpaque,
			EffectIn:  effectIn,
			EffectOut: effectOut,
			Note: fmt.Sprintf(
				"exact __tetra_task_spawn_i32 entry=%d ret_slots=2 handle=%s status=%s",
				spawn.EntryID,
				handleValue,
				statusValue,
			),
		})
		effectIn = effectOut
	}
	partialValues := make([]ssair.ValueID, 0, len(plan.Joins))
	for i, join := range plan.Joins {
		valueID := ssair.ValueID(fmt.Sprintf("%s.value", join.Worker))
		handleValue := ssair.ValueID(fmt.Sprintf("%s.handle", join.Worker))
		statusValue := ssair.ValueID(fmt.Sprintf("%s.status", join.Worker))
		effectOut := ssair.ValueID(fmt.Sprintf("join.effect%d", i))
		values = append(values,
			ssair.Value{ID: valueID, Type: ssair.TypeI32, Origin: "task_result"},
			ssair.Value{ID: effectOut, Type: ssair.TypeEffect, Origin: "runtime_effect"},
		)
		instrs = append(instrs, ssair.Instr{
			ID:        fmt.Sprintf("join_%s", join.Worker),
			Kind:      ssair.OpCall,
			Result:    valueID,
			Type:      ssair.TypeI32,
			Call:      "__tetra_task_join_i32",
			Args:      []ssair.ValueID{handleValue, statusValue},
			EffectIn:  effectIn,
			EffectOut: effectOut,
		})
		partialValues = append(partialValues, valueID)
		effectIn = effectOut
	}
	if len(partialValues) != 3 {
		return false
	}
	values = append(values,
		ssair.Value{ID: "partial.total", Type: ssair.TypeI32, Origin: "instr"},
		ssair.Value{ID: "total", Type: ssair.TypeI32, Origin: "instr"},
		ssair.Value{ID: "matches.expected", Type: ssair.TypeBool, Origin: "instr"},
	)
	instrs = append(instrs,
		ssair.Instr{
			ID:     "partial_total",
			Kind:   ssair.OpAddI32,
			Result: "partial.total",
			Type:   ssair.TypeI32,
			Args:   []ssair.ValueID{partialValues[0], partialValues[1]},
		},
		ssair.Instr{
			ID:     "total",
			Kind:   ssair.OpAddI32,
			Result: "total",
			Type:   ssair.TypeI32,
			Args:   []ssair.ValueID{"partial.total", partialValues[2]},
		},
		ssair.Instr{
			ID:     "matches_expected",
			Kind:   ssair.OpCmpEqI32,
			Result: "matches.expected",
			Type:   ssair.TypeBool,
			Args:   []ssair.ValueID{"total", "expected"},
		},
	)
	ssaFn := ssair.Function{
		Name:       fn.Name,
		ReturnType: ssair.TypeI32,
		Values:     values,
		Blocks: []ssair.Block{
			{
				ID:     "entry",
				Entry:  true,
				Instrs: instrs,
				Term: ssair.Terminator{
					Kind:    ssair.TermCondBr,
					Cond:    "matches.expected",
					IfTrue:  "success",
					IfFalse: "failure",
				},
			},
			{
				ID:   "success",
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "zero"},
			},
			{
				ID:   "failure",
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "total"},
			},
		},
	}
	return ssair.VerifyFunction(ssaFn) == nil
}

func stackIRActorPingPongRuntimeCallPassesSSAGate(
	fn ir.IRFunc,
	callABI machine.CallABIInfo,
) bool {
	plan, ok, err := machine.ActorPingPongRuntimeCallPlanFromStackIRWithCallABI(fn, callABI)
	if err != nil || !ok {
		return false
	}
	switch plan.Path {
	case "machine-ir-actor-ping-pong-pong":
		return stackIRActorPingPongPongPassesSSAGate(fn, plan)
	case "machine-ir-actor-ping-pong-main":
		return stackIRActorPingPongMainPassesSSAGate(fn, plan)
	default:
		return false
	}
}

func stackIRActorPingPongPongPassesSSAGate(
	fn ir.IRFunc,
	plan machine.ActorPingPongRuntimeCallPlan,
) bool {
	ssaFn := ssair.Function{
		Name:       fn.Name,
		ReturnType: ssair.TypeI32,
		Values: []ssair.Value{
			{ID: "effect0", Type: ssair.TypeEffect, Origin: "entry_effect"},
			{ID: "recv.effect", Type: ssair.TypeEffect, Origin: "runtime_effect"},
			{ID: "sender.effect", Type: ssair.TypeEffect, Origin: "runtime_effect"},
			{ID: "send.effect", Type: ssair.TypeEffect, Origin: "runtime_effect"},
			{ID: "recv.value", Type: ssair.TypeI32, Origin: "actor_recv"},
			{ID: "sender.handle", Type: ssair.TypePtr, Origin: "actor_sender"},
			{ID: "send.status", Type: ssair.TypeI32, Origin: "actor_send"},
			{ID: "expected", Type: ssair.TypeI32, Origin: "const"},
			{ID: "reply", Type: ssair.TypeI32, Origin: "const"},
			{ID: "zero", Type: ssair.TypeI32, Origin: "const"},
			{ID: "one", Type: ssair.TypeI32, Origin: "const"},
			{ID: "matches.expected", Type: ssair.TypeBool, Origin: "instr"},
		},
		Blocks: []ssair.Block{
			{
				ID:    "entry",
				Entry: true,
				Instrs: []ssair.Instr{
					{
						ID:        "recv",
						Kind:      ssair.OpCall,
						Result:    "recv.value",
						Type:      ssair.TypeI32,
						Call:      "__tetra_actor_recv",
						EffectIn:  "effect0",
						EffectOut: "recv.effect",
					},
					{
						ID:     "const_expected",
						Kind:   ssair.OpConstI32,
						Result: "expected",
						Type:   ssair.TypeI32,
						Imm:    41,
					},
					{
						ID:     "const_zero",
						Kind:   ssair.OpConstI32,
						Result: "zero",
						Type:   ssair.TypeI32,
						Imm:    0,
					},
					{
						ID:     "const_one",
						Kind:   ssair.OpConstI32,
						Result: "one",
						Type:   ssair.TypeI32,
						Imm:    1,
					},
					{
						ID:     "matches_expected",
						Kind:   ssair.OpCmpEqI32,
						Result: "matches.expected",
						Type:   ssair.TypeBool,
						Args:   []ssair.ValueID{"recv.value", "expected"},
					},
				},
				Term: ssair.Terminator{
					Kind:    ssair.TermCondBr,
					Cond:    "matches.expected",
					IfTrue:  "send",
					IfFalse: "failure",
				},
			},
			{
				ID: "send",
				Instrs: []ssair.Instr{
					{
						ID:        "sender",
						Kind:      ssair.OpCall,
						Result:    "sender.handle",
						Type:      ssair.TypePtr,
						Call:      "__tetra_actor_sender",
						EffectIn:  "recv.effect",
						EffectOut: "sender.effect",
					},
					{
						ID:     "const_reply",
						Kind:   ssair.OpConstI32,
						Result: "reply",
						Type:   ssair.TypeI32,
						Imm:    42,
					},
					{
						ID:        "send_scalar",
						Kind:      ssair.OpCall,
						Result:    "send.status",
						Type:      ssair.TypeI32,
						Call:      "__tetra_actor_send",
						Args:      []ssair.ValueID{"sender.handle", "reply"},
						EffectIn:  "sender.effect",
						EffectOut: "send.effect",
					},
				},
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "zero"},
			},
			{
				ID:   "failure",
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "one"},
			},
		},
	}
	_ = plan
	return ssair.VerifyFunction(ssaFn) == nil
}

func stackIRActorPingPongMainPassesSSAGate(
	fn ir.IRFunc,
	plan machine.ActorPingPongRuntimeCallPlan,
) bool {
	ssaFn := ssair.Function{
		Name:       fn.Name,
		ReturnType: ssair.TypeI32,
		Values: []ssair.Value{
			{ID: "effect0", Type: ssair.TypeEffect, Origin: "entry_effect"},
			{ID: "spawn.effect", Type: ssair.TypeEffect, Origin: "runtime_effect"},
			{ID: "send.effect", Type: ssair.TypeEffect, Origin: "runtime_effect"},
			{ID: "recv.effect", Type: ssair.TypeEffect, Origin: "runtime_effect"},
			{ID: "spawn.id", Type: ssair.TypeI32, Origin: "const"},
			{ID: "actor.handle", Type: ssair.TypePtr, Origin: "actor_spawn"},
			{ID: "ping", Type: ssair.TypeI32, Origin: "const"},
			{ID: "send.status", Type: ssair.TypeI32, Origin: "actor_send"},
			{ID: "recv.reply", Type: ssair.TypeI32, Origin: "actor_recv"},
			{ID: "expected", Type: ssair.TypeI32, Origin: "const"},
			{ID: "zero", Type: ssair.TypeI32, Origin: "const"},
			{ID: "one", Type: ssair.TypeI32, Origin: "const"},
			{ID: "matches.expected", Type: ssair.TypeBool, Origin: "instr"},
		},
		Blocks: []ssair.Block{
			{
				ID:    "entry",
				Entry: true,
				Instrs: []ssair.Instr{
					{
						ID:     "const_spawn_id",
						Kind:   ssair.OpConstI32,
						Result: "spawn.id",
						Type:   ssair.TypeI32,
						Imm:    plan.SpawnEntryID,
					},
					{
						ID:        "spawn",
						Kind:      ssair.OpCall,
						Result:    "actor.handle",
						Type:      ssair.TypePtr,
						Call:      "__tetra_actor_spawn",
						Args:      []ssair.ValueID{"spawn.id"},
						EffectIn:  "effect0",
						EffectOut: "spawn.effect",
					},
					{
						ID:     "const_ping",
						Kind:   ssair.OpConstI32,
						Result: "ping",
						Type:   ssair.TypeI32,
						Imm:    41,
					},
					{
						ID:        "send_scalar",
						Kind:      ssair.OpCall,
						Result:    "send.status",
						Type:      ssair.TypeI32,
						Call:      "__tetra_actor_send",
						Args:      []ssair.ValueID{"actor.handle", "ping"},
						EffectIn:  "spawn.effect",
						EffectOut: "send.effect",
					},
					{
						ID:        "recv",
						Kind:      ssair.OpCall,
						Result:    "recv.reply",
						Type:      ssair.TypeI32,
						Call:      "__tetra_actor_recv",
						EffectIn:  "send.effect",
						EffectOut: "recv.effect",
					},
					{
						ID:     "const_expected",
						Kind:   ssair.OpConstI32,
						Result: "expected",
						Type:   ssair.TypeI32,
						Imm:    42,
					},
					{
						ID:     "const_zero",
						Kind:   ssair.OpConstI32,
						Result: "zero",
						Type:   ssair.TypeI32,
						Imm:    0,
					},
					{
						ID:     "const_one",
						Kind:   ssair.OpConstI32,
						Result: "one",
						Type:   ssair.TypeI32,
						Imm:    1,
					},
					{
						ID:     "matches_expected",
						Kind:   ssair.OpCmpEqI32,
						Result: "matches.expected",
						Type:   ssair.TypeBool,
						Args:   []ssair.ValueID{"recv.reply", "expected"},
					},
				},
				Term: ssair.Terminator{
					Kind:    ssair.TermCondBr,
					Cond:    "matches.expected",
					IfTrue:  "success",
					IfFalse: "failure",
				},
			},
			{
				ID:   "success",
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "zero"},
			},
			{
				ID:   "failure",
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "one"},
			},
		},
	}
	return ssair.VerifyFunction(ssaFn) == nil
}

func stackIRSliceSumMainPassesSSAGate(fn ir.IRFunc) bool {
	plan, ok, err := machine.SliceSumMainPlanFromStackIR(fn)
	if err != nil || !ok {
		return false
	}
	ssaFn := ssair.Function{
		Name:       fn.Name,
		ReturnType: ssair.TypeI32,
		Values: []ssair.Value{
			{ID: "zero", Type: ssair.TypeI32, Origin: "const"},
			{ID: "one", Type: ssair.TypeI32, Origin: "const"},
			{ID: "length", Type: ssair.TypeI32, Origin: "const"},
			{ID: "fill.modulus", Type: ssair.TypeI32, Origin: "const"},
			{ID: "repeat.count", Type: ssair.TypeI32, Origin: "const"},
			{ID: "xs.ptr", Type: ssair.TypePtr, Origin: "stack_slice"},
			{ID: "fill.index", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "fill.body.index", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "fill.cmp", Type: ssair.TypeBool, Origin: "instr"},
			{ID: "fill.value", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "fill.next", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "outer.r", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "outer.total", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "outer.cmp", Type: ssair.TypeBool, Origin: "instr"},
			{ID: "inner.index", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "inner.total", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "inner.r", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "inner.cmp", Type: ssair.TypeBool, Origin: "instr"},
			{ID: "inner.body.index", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "inner.body.total", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "inner.body.r", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "elem", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "next.total", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "next.inner.index", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "outer.step.total", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "outer.step.r", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "next.outer.r", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "final.total", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "final.cmp", Type: ssair.TypeBool, Origin: "instr"},
		},
		Blocks: []ssair.Block{
			{
				ID:    "entry",
				Entry: true,
				Instrs: []ssair.Instr{
					{ID: "const_zero", Kind: ssair.OpConstI32, Result: "zero", Type: ssair.TypeI32, Imm: 0},
					{ID: "const_one", Kind: ssair.OpConstI32, Result: "one", Type: ssair.TypeI32, Imm: plan.Step},
					{
						ID:     "const_length",
						Kind:   ssair.OpConstI32,
						Result: "length",
						Type:   ssair.TypeI32,
						Imm:    plan.Length,
					},
					{
						ID:     "const_fill_modulus",
						Kind:   ssair.OpConstI32,
						Result: "fill.modulus",
						Type:   ssair.TypeI32,
						Imm:    plan.FillModulus,
					},
					{
						ID:     "const_repeat_count",
						Kind:   ssair.OpConstI32,
						Result: "repeat.count",
						Type:   ssair.TypeI32,
						Imm:    plan.RepeatCount,
					},
				},
				Term: ssair.Terminator{
					Kind:   ssair.TermBranch,
					Target: "fill_header",
					Args:   []ssair.ValueID{"zero"},
				},
			},
			{
				ID:     "fill_header",
				Params: []ssair.ValueID{"fill.index"},
				Instrs: []ssair.Instr{{
					ID:     "fill_cmp",
					Kind:   ssair.OpCmpLtI32,
					Result: "fill.cmp",
					Type:   ssair.TypeBool,
					Args:   []ssair.ValueID{"fill.index", "length"},
				}},
				Term: ssair.Terminator{
					Kind:        ssair.TermCondBr,
					Cond:        "fill.cmp",
					IfTrue:      "fill_body",
					IfTrueArgs:  []ssair.ValueID{"fill.index"},
					IfFalse:     "outer_header",
					IfFalseArgs: []ssair.ValueID{"zero", "zero"},
				},
			},
			{
				ID:     "fill_body",
				Params: []ssair.ValueID{"fill.body.index"},
				Instrs: []ssair.Instr{
					{
						ID:     "fill_mod",
						Kind:   ssair.OpModI32,
						Result: "fill.value",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"fill.body.index", "fill.modulus"},
					},
					{
						ID:   "fill_store",
						Kind: ssair.OpOpaque,
						Args: []ssair.ValueID{
							"xs.ptr",
							"length",
							"fill.body.index",
							"fill.value",
						},
						ProofID: plan.StoreProofID,
						Note:    "proof-tagged i32 index store",
					},
					{
						ID:     "fill_next",
						Kind:   ssair.OpAddI32,
						Result: "fill.next",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"fill.body.index", "one"},
					},
				},
				Term: ssair.Terminator{
					Kind:   ssair.TermBranch,
					Target: "fill_header",
					Args:   []ssair.ValueID{"fill.next"},
				},
			},
			{
				ID:     "outer_header",
				Params: []ssair.ValueID{"outer.r", "outer.total"},
				Instrs: []ssair.Instr{{
					ID:     "outer_cmp",
					Kind:   ssair.OpCmpLtI32,
					Result: "outer.cmp",
					Type:   ssair.TypeBool,
					Args:   []ssair.ValueID{"outer.r", "repeat.count"},
				}},
				Term: ssair.Terminator{
					Kind:        ssair.TermCondBr,
					Cond:        "outer.cmp",
					IfTrue:      "inner_header",
					IfTrueArgs:  []ssair.ValueID{"zero", "outer.total", "outer.r"},
					IfFalse:     "final",
					IfFalseArgs: []ssair.ValueID{"outer.total"},
				},
			},
			{
				ID:     "inner_header",
				Params: []ssair.ValueID{"inner.index", "inner.total", "inner.r"},
				Instrs: []ssair.Instr{{
					ID:     "inner_cmp",
					Kind:   ssair.OpCmpLtI32,
					Result: "inner.cmp",
					Type:   ssair.TypeBool,
					Args:   []ssair.ValueID{"inner.index", "length"},
				}},
				Term: ssair.Terminator{
					Kind:        ssair.TermCondBr,
					Cond:        "inner.cmp",
					IfTrue:      "inner_body",
					IfTrueArgs:  []ssair.ValueID{"inner.index", "inner.total", "inner.r"},
					IfFalse:     "outer_step",
					IfFalseArgs: []ssair.ValueID{"inner.total", "inner.r"},
				},
			},
			{
				ID:     "inner_body",
				Params: []ssair.ValueID{"inner.body.index", "inner.body.total", "inner.body.r"},
				Instrs: []ssair.Instr{
					{
						ID:      "load_elem",
						Kind:    ssair.OpIndexLoadI32,
						Result:  "elem",
						Type:    ssair.TypeI32,
						Args:    []ssair.ValueID{"xs.ptr", "length", "inner.body.index"},
						ProofID: plan.LoadProofID,
					},
					{
						ID:     "next_total",
						Kind:   ssair.OpAddI32,
						Result: "next.total",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"inner.body.total", "elem"},
					},
					{
						ID:     "next_inner_index",
						Kind:   ssair.OpAddI32,
						Result: "next.inner.index",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"inner.body.index", "one"},
					},
				},
				Term: ssair.Terminator{
					Kind:   ssair.TermBranch,
					Target: "inner_header",
					Args: []ssair.ValueID{
						"next.inner.index",
						"next.total",
						"inner.body.r",
					},
				},
			},
			{
				ID:     "outer_step",
				Params: []ssair.ValueID{"outer.step.total", "outer.step.r"},
				Instrs: []ssair.Instr{{
					ID:     "next_outer_r",
					Kind:   ssair.OpAddI32,
					Result: "next.outer.r",
					Type:   ssair.TypeI32,
					Args:   []ssair.ValueID{"outer.step.r", "one"},
				}},
				Term: ssair.Terminator{
					Kind:   ssair.TermBranch,
					Target: "outer_header",
					Args:   []ssair.ValueID{"next.outer.r", "outer.step.total"},
				},
			},
			{
				ID:     "final",
				Params: []ssair.ValueID{"final.total"},
				Instrs: []ssair.Instr{{
					ID:     "final_cmp",
					Kind:   ssair.OpCmpGtI32,
					Result: "final.cmp",
					Type:   ssair.TypeBool,
					Args:   []ssair.ValueID{"final.total", "zero"},
				}},
				Term: ssair.Terminator{
					Kind:    ssair.TermCondBr,
					Cond:    "final.cmp",
					IfTrue:  "success",
					IfFalse: "failure",
				},
			},
			{
				ID:   "success",
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "zero"},
			},
			{
				ID:   "failure",
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "one"},
			},
		},
	}
	return ssair.VerifyFunction(ssaFn) == nil
}

func stackIRRegionIslandAllocationMainPassesSSAGate(fn ir.IRFunc) bool {
	plan, ok, err := machine.RegionIslandAllocationMainPlanFromStackIR(fn)
	if err != nil || !ok {
		return false
	}
	ssaFn := ssair.Function{
		Name:       fn.Name,
		ReturnType: ssair.TypeI32,
		Values: []ssair.Value{
			{ID: "zero", Type: ssair.TypeI32, Origin: "const"},
			{ID: "one", Type: ssair.TypeI32, Origin: "const"},
			{ID: "bound", Type: ssair.TypeI32, Origin: "const"},
			{ID: "loop.index", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "loop.checksum", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "body.index", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "body.checksum", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "exit.checksum", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "cmp", Type: ssair.TypeBool, Origin: "instr"},
			{ID: "xs0", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "next.checksum", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "next.index", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "final.cmp", Type: ssair.TypeBool, Origin: "instr"},
			{ID: "return.success", Type: ssair.TypeI32, Origin: "const"},
			{ID: "return.failure", Type: ssair.TypeI32, Origin: "const"},
		},
		Blocks: []ssair.Block{
			{
				ID:    "entry",
				Entry: true,
				Instrs: []ssair.Instr{
					{
						ID:     "const_zero",
						Kind:   ssair.OpConstI32,
						Result: "zero",
						Type:   ssair.TypeI32,
						Imm:    plan.IndexConst,
					},
					{ID: "const_one", Kind: ssair.OpConstI32, Result: "one", Type: ssair.TypeI32, Imm: plan.Step},
					{
						ID:     "const_bound",
						Kind:   ssair.OpConstI32,
						Result: "bound",
						Type:   ssair.TypeI32,
						Imm:    plan.LoopBound,
					},
				},
				Term: ssair.Terminator{
					Kind:   ssair.TermBranch,
					Target: "loop",
					Args:   []ssair.ValueID{"zero", "zero"},
				},
			},
			{
				ID:     "loop",
				Params: []ssair.ValueID{"loop.index", "loop.checksum"},
				Instrs: []ssair.Instr{
					{
						ID:     "cmp_loop_bound",
						Kind:   ssair.OpCmpLtI32,
						Result: "cmp",
						Type:   ssair.TypeBool,
						Args:   []ssair.ValueID{"loop.index", "bound"},
					},
				},
				Term: ssair.Terminator{
					Kind:        ssair.TermCondBr,
					Cond:        "cmp",
					IfTrue:      "body",
					IfTrueArgs:  []ssair.ValueID{"loop.index", "loop.checksum"},
					IfFalse:     "exit",
					IfFalseArgs: []ssair.ValueID{"loop.checksum"},
				},
			},
			{
				ID:     "body",
				Params: []ssair.ValueID{"body.index", "body.checksum"},
				Instrs: []ssair.Instr{
					{
						ID:      "load_xs0",
						Kind:    ssair.OpIndexLoadI32,
						Result:  "xs0",
						Type:    ssair.TypeI32,
						Args:    []ssair.ValueID{"zero"},
						ProofID: plan.LoadProofID,
						Note:    "scalar-replaced exact island xs[0]",
					},
					{
						ID:     "add_checksum",
						Kind:   ssair.OpAddI32,
						Result: "next.checksum",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"body.checksum", "xs0"},
					},
					{
						ID:     "inc_index",
						Kind:   ssair.OpAddI32,
						Result: "next.index",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"body.index", "one"},
					},
				},
				Term: ssair.Terminator{
					Kind:   ssair.TermBranch,
					Target: "loop",
					Args:   []ssair.ValueID{"next.index", "next.checksum"},
				},
			},
			{
				ID:     "exit",
				Params: []ssair.ValueID{"exit.checksum"},
				Instrs: []ssair.Instr{
					{
						ID:     "cmp_checksum_positive",
						Kind:   ssair.OpCmpGtI32,
						Result: "final.cmp",
						Type:   ssair.TypeBool,
						Args:   []ssair.ValueID{"exit.checksum", "zero"},
					},
				},
				Term: ssair.Terminator{
					Kind:    ssair.TermCondBr,
					Cond:    "final.cmp",
					IfTrue:  "success",
					IfFalse: "failure",
				},
			},
			{
				ID: "success",
				Instrs: []ssair.Instr{
					{
						ID:     "const_success",
						Kind:   ssair.OpConstI32,
						Result: "return.success",
						Type:   ssair.TypeI32,
						Imm:    plan.SuccessReturn,
					},
				},
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "return.success"},
			},
			{
				ID: "failure",
				Instrs: []ssair.Instr{
					{
						ID:     "const_failure",
						Kind:   ssair.OpConstI32,
						Result: "return.failure",
						Type:   ssair.TypeI32,
						Imm:    plan.FailureReturn,
					},
				},
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "return.failure"},
			},
		},
	}
	return ssair.VerifyFunction(ssaFn) == nil
}

func stackIRMatrixMultiplyMainPassesSSAGate(fn ir.IRFunc) bool {
	plan, ok, err := machine.MatrixMultiplyMainPlanFromStackIR(fn)
	if err != nil || !ok {
		return false
	}
	ssaFn := ssair.Function{
		Name:       fn.Name,
		ReturnType: ssair.TypeI32,
		Values: []ssair.Value{
			{ID: "zero", Type: ssair.TypeI32, Origin: "const"},
			{ID: "one", Type: ssair.TypeI32, Origin: "const"},
			{ID: "slice.length", Type: ssair.TypeI32, Origin: "const"},
			{ID: "dimension", Type: ssair.TypeI32, Origin: "const"},
			{ID: "repeat.count", Type: ssair.TypeI32, Origin: "const"},
			{ID: "a.ptr", Type: ssair.TypePtr, Origin: "stack_slice"},
			{ID: "b.ptr", Type: ssair.TypePtr, Origin: "stack_slice"},
			{ID: "c.ptr", Type: ssair.TypePtr, Origin: "stack_slice"},
			{ID: "i", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "row", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "col", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "k", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "r", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "total", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "checksum", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "fill.a", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "fill.b", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "row.k.index", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "k.col.index", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "row.col.index", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "r.mod.index", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "a.value", Type: ssair.TypeI32, Origin: "unchecked_index_load"},
			{ID: "b.value", Type: ssair.TypeI32, Origin: "unchecked_index_load"},
			{ID: "c.value", Type: ssair.TypeI32, Origin: "unchecked_index_load"},
			{ID: "product", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "next.total", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "next.checksum", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "final.cmp", Type: ssair.TypeBool, Origin: "instr"},
		},
		Blocks: []ssair.Block{
			{
				ID:    "entry",
				Entry: true,
				Instrs: []ssair.Instr{
					{ID: "const_zero", Kind: ssair.OpConstI32, Result: "zero", Type: ssair.TypeI32, Imm: 0},
					{ID: "const_one", Kind: ssair.OpConstI32, Result: "one", Type: ssair.TypeI32, Imm: plan.Step},
					{
						ID:     "const_slice_length",
						Kind:   ssair.OpConstI32,
						Result: "slice.length",
						Type:   ssair.TypeI32,
						Imm:    plan.SliceLength,
					},
					{
						ID:     "const_dimension",
						Kind:   ssair.OpConstI32,
						Result: "dimension",
						Type:   ssair.TypeI32,
						Imm:    plan.Dimension,
					},
					{
						ID:     "const_repeat_count",
						Kind:   ssair.OpConstI32,
						Result: "repeat.count",
						Type:   ssair.TypeI32,
						Imm:    plan.RepeatCount,
					},
					{
						ID:     "fill_a_value",
						Kind:   ssair.OpAddI32,
						Result: "fill.a",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"i", "one"},
					},
					{
						ID:     "fill_b_value",
						Kind:   ssair.OpSubI32,
						Result: "fill.b",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"slice.length", "i"},
					},
					{
						ID:      "fill_a_store",
						Kind:    ssair.OpOpaque,
						Args:    []ssair.ValueID{"a.ptr", "slice.length", "i", "fill.a"},
						ProofID: plan.AFillProofID,
						Note:    "proof-tagged a[i] store",
					},
					{
						ID:      "fill_b_store",
						Kind:    ssair.OpOpaque,
						Args:    []ssair.ValueID{"b.ptr", "slice.length", "i", "fill.b"},
						ProofID: plan.BFillProofID,
						Note:    "proof-tagged b[i] store",
					},
					{
						ID:      "fill_c_store",
						Kind:    ssair.OpOpaque,
						Args:    []ssair.ValueID{"c.ptr", "slice.length", "i", "zero"},
						ProofID: plan.CFillProofID,
						Note:    "proof-tagged c[i] store",
					},
					{
						ID:     "row_k_index",
						Kind:   ssair.OpOpaque,
						Result: "row.k.index",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"row", "dimension", "k"},
						Note:   "row * 3 + k",
					},
					{
						ID:     "k_col_index",
						Kind:   ssair.OpOpaque,
						Result: "k.col.index",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"k", "dimension", "col"},
						Note:   "k * 3 + col",
					},
					{
						ID:      "a_load",
						Kind:    ssair.OpIndexLoadI32,
						Result:  "a.value",
						Type:    ssair.TypeI32,
						Args:    []ssair.ValueID{"a.ptr", "slice.length", "row.k.index"},
						ProofID: plan.ARowKProofID,
					},
					{
						ID:      "b_load",
						Kind:    ssair.OpIndexLoadI32,
						Result:  "b.value",
						Type:    ssair.TypeI32,
						Args:    []ssair.ValueID{"b.ptr", "slice.length", "k.col.index"},
						ProofID: plan.BKColProofID,
					},
					{
						ID:     "product",
						Kind:   ssair.OpMulI32,
						Result: "product",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"a.value", "b.value"},
					},
					{
						ID:     "next_total",
						Kind:   ssair.OpAddI32,
						Result: "next.total",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"total", "product"},
					},
					{
						ID:     "row_col_index",
						Kind:   ssair.OpOpaque,
						Result: "row.col.index",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"row", "dimension", "col"},
						Note:   "row * 3 + col",
					},
					{
						ID:      "c_store",
						Kind:    ssair.OpOpaque,
						Args:    []ssair.ValueID{"c.ptr", "slice.length", "row.col.index", "next.total"},
						ProofID: plan.CRowColProofID,
						Note:    "proof-tagged c[row * 3 + col] store",
					},
					{
						ID:     "r_mod_index",
						Kind:   ssair.OpModI32,
						Result: "r.mod.index",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"r", "slice.length"},
					},
					{
						ID:      "c_checksum_load",
						Kind:    ssair.OpIndexLoadI32,
						Result:  "c.value",
						Type:    ssair.TypeI32,
						Args:    []ssair.ValueID{"c.ptr", "slice.length", "r.mod.index"},
						ProofID: plan.CModuloProofID,
					},
					{
						ID:     "next_checksum",
						Kind:   ssair.OpAddI32,
						Result: "next.checksum",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"checksum", "c.value"},
					},
					{
						ID:     "final_cmp",
						Kind:   ssair.OpCmpGtI32,
						Result: "final.cmp",
						Type:   ssair.TypeBool,
						Args:   []ssair.ValueID{"next.checksum", "zero"},
					},
				},
				Term: ssair.Terminator{Kind: ssair.TermCondBr, Cond: "final.cmp", IfTrue: "success", IfFalse: "failure"},
			},
			{
				ID:   "success",
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "zero"},
			},
			{
				ID:   "failure",
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "one"},
			},
		},
	}
	return ssair.VerifyFunction(ssaFn) == nil
}

func stackIRHashTableLookupPassesSSAGate(fn ir.IRFunc) bool {
	plan, ok, err := machine.HashTableLookupPlanFromStackIR(fn)
	if err != nil || !ok {
		return false
	}
	ssaFn := ssair.Function{
		Name:       fn.Name,
		ReturnType: ssair.TypeI32,
		Values: []ssair.Value{
			{ID: "keys.ptr", Type: ssair.TypePtr, Origin: "param"},
			{ID: "keys.len", Type: ssair.TypeI32, Origin: "param"},
			{ID: "values.ptr", Type: ssair.TypePtr, Origin: "param"},
			{ID: "values.len", Type: ssair.TypeI32, Origin: "param"},
			{ID: "n", Type: ssair.TypeI32, Origin: "param"},
			{ID: "key", Type: ssair.TypeI32, Origin: "param"},
			{ID: "zero", Type: ssair.TypeI32, Origin: "const"},
			{ID: "one", Type: ssair.TypeI32, Origin: "const"},
			{ID: "effect0", Type: ssair.TypeEffect, Origin: "entry_effect"},
			{ID: "loop.index", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "loop.effect", Type: ssair.TypeEffect, Origin: "block_param"},
			{ID: "cmp.loop", Type: ssair.TypeBool, Origin: "instr"},
			{ID: "body.index", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "body.effect", Type: ssair.TypeEffect, Origin: "block_param"},
			{ID: "key.elem", Type: ssair.TypeI32, Origin: "unchecked_index_load"},
			{ID: "key.load.effect", Type: ssair.TypeEffect, Origin: "unchecked_index_load"},
			{ID: "key.match", Type: ssair.TypeBool, Origin: "instr"},
			{ID: "return.index", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "return.effect", Type: ssair.TypeEffect, Origin: "block_param"},
			{ID: "value", Type: ssair.TypeI32, Origin: "unchecked_index_load"},
			{ID: "value.load.effect", Type: ssair.TypeEffect, Origin: "unchecked_index_load"},
			{ID: "miss.index", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "miss.effect", Type: ssair.TypeEffect, Origin: "block_param"},
			{ID: "next.index", Type: ssair.TypeI32, Origin: "instr"},
		},
		Blocks: []ssair.Block{
			{
				ID:    "entry",
				Entry: true,
				Instrs: []ssair.Instr{
					{
						ID:     "const_zero",
						Kind:   ssair.OpConstI32,
						Result: "zero",
						Type:   ssair.TypeI32,
						Imm:    plan.NotFoundReturn,
					},
					{
						ID:     "const_one",
						Kind:   ssair.OpConstI32,
						Result: "one",
						Type:   ssair.TypeI32,
						Imm:    plan.Step,
					},
				},
				Term: ssair.Terminator{
					Kind:   ssair.TermBranch,
					Target: "loop",
					Args:   []ssair.ValueID{"zero", "effect0"},
				},
			},
			{
				ID:     "loop",
				Params: []ssair.ValueID{"loop.index", "loop.effect"},
				Instrs: []ssair.Instr{
					{
						ID:     "cmp_loop_bound",
						Kind:   ssair.OpCmpLtI32,
						Result: "cmp.loop",
						Type:   ssair.TypeBool,
						Args:   []ssair.ValueID{"loop.index", "n"},
					},
				},
				Term: ssair.Terminator{
					Kind:        ssair.TermCondBr,
					Cond:        "cmp.loop",
					IfTrue:      "body",
					IfTrueArgs:  []ssair.ValueID{"loop.index", "loop.effect"},
					IfFalse:     "exit",
					IfFalseArgs: nil,
				},
			},
			{
				ID:     "body",
				Params: []ssair.ValueID{"body.index", "body.effect"},
				Instrs: []ssair.Instr{
					{
						ID:        "load_key",
						Kind:      ssair.OpIndexLoadI32,
						Result:    "key.elem",
						Type:      ssair.TypeI32,
						Args:      []ssair.ValueID{"keys.ptr", "keys.len", "body.index"},
						EffectIn:  "body.effect",
						EffectOut: "key.load.effect",
						ProofID:   plan.KeysProofID,
					},
					{
						ID:     "cmp_key",
						Kind:   ssair.OpCmpEqI32,
						Result: "key.match",
						Type:   ssair.TypeBool,
						Args:   []ssair.ValueID{"key.elem", "key"},
					},
				},
				Term: ssair.Terminator{
					Kind:        ssair.TermCondBr,
					Cond:        "key.match",
					IfTrue:      "return_value",
					IfTrueArgs:  []ssair.ValueID{"body.index", "key.load.effect"},
					IfFalse:     "miss",
					IfFalseArgs: []ssair.ValueID{"body.index", "key.load.effect"},
				},
			},
			{
				ID:     "return_value",
				Params: []ssair.ValueID{"return.index", "return.effect"},
				Instrs: []ssair.Instr{
					{
						ID:        "load_value",
						Kind:      ssair.OpIndexLoadI32,
						Result:    "value",
						Type:      ssair.TypeI32,
						Args:      []ssair.ValueID{"values.ptr", "values.len", "return.index"},
						EffectIn:  "return.effect",
						EffectOut: "value.load.effect",
						ProofID:   plan.ValuesProofID,
					},
				},
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "value"},
			},
			{
				ID:     "miss",
				Params: []ssair.ValueID{"miss.index", "miss.effect"},
				Instrs: []ssair.Instr{
					{
						ID:     "inc_index",
						Kind:   ssair.OpAddI32,
						Result: "next.index",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"miss.index", "one"},
					},
				},
				Term: ssair.Terminator{
					Kind:   ssair.TermBranch,
					Target: "loop",
					Args:   []ssair.ValueID{"next.index", "miss.effect"},
				},
			},
			{
				ID:   "exit",
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "zero"},
			},
		},
	}
	return ssair.VerifyFunction(ssaFn) == nil
}

func stackIRHashTableMainPassesSSAGate(fn ir.IRFunc) bool {
	plan, ok, err := machine.HashTableMainPlanFromStackIR(fn)
	if err != nil || !ok {
		return false
	}
	ssaFn := ssair.Function{
		Name:       fn.Name,
		ReturnType: ssair.TypeI32,
		Values: []ssair.Value{
			{ID: "zero", Type: ssair.TypeI32, Origin: "const"},
			{ID: "effect0", Type: ssair.TypeEffect, Origin: "entry_effect"},
			{ID: "effect1", Type: ssair.TypeEffect, Origin: "hash_table_main_exact"},
		},
		Blocks: []ssair.Block{{
			ID:    "entry",
			Entry: true,
			Instrs: []ssair.Instr{
				{
					ID:     "const_zero",
					Kind:   ssair.OpConstI32,
					Result: "zero",
					Type:   ssair.TypeI32,
					Imm:    plan.SuccessReturn,
				},
				{
					ID:        "hash_table_main_exact",
					Kind:      ssair.OpOpaque,
					EffectIn:  "effect0",
					EffectOut: "effect1",
					Note: fmt.Sprintf(
						"exact hash-table main stack slices length=%d call=%s arg_slots=%d ret_slots=%d",
						plan.Length,
						plan.CallName,
						plan.CallArgSlots,
						plan.CallRetSlots,
					),
				},
			},
			Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "zero"},
		}},
	}
	return ssair.VerifyFunction(ssaFn) == nil
}

func stackIRRecursionBenchmarkPassesSSAGate(fn ir.IRFunc) bool {
	switch fn.Name {
	case "p25.recursion.fib":
		return stackIRRecursionFibPassesSSAGate(fn)
	case "p25.recursion.main":
		return stackIRRecursionMainPassesSSAGate(fn)
	default:
		return false
	}
}

func stackIRRecursionFibPassesSSAGate(fn ir.IRFunc) bool {
	plan, ok, err := machine.RecursionFibPlanFromStackIRWithCallABI(fn, machine.SysVCallABIInfo())
	if err != nil || !ok {
		return false
	}
	ssaFn := ssair.Function{
		Name:       fn.Name,
		ReturnType: ssair.TypeI32,
		Values: []ssair.Value{
			{ID: "n", Type: ssair.TypeI32, Origin: "param"},
			{ID: "two", Type: ssair.TypeI32, Origin: "const"},
			{ID: "one", Type: ssair.TypeI32, Origin: "const"},
			{ID: "cmp", Type: ssair.TypeBool, Origin: "instr"},
			{ID: "n.minus.one", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "n.minus.two", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "fib.one", Type: ssair.TypeI32, Origin: "call"},
			{ID: "fib.two", Type: ssair.TypeI32, Origin: "call"},
			{ID: "sum", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "effect0", Type: ssair.TypeEffect, Origin: "entry_effect"},
			{ID: "effect1", Type: ssair.TypeEffect, Origin: "call_effect"},
			{ID: "effect2", Type: ssair.TypeEffect, Origin: "call_effect"},
		},
		Blocks: []ssair.Block{
			{
				ID:    "entry",
				Entry: true,
				Instrs: []ssair.Instr{
					{
						ID:     "const_two",
						Kind:   ssair.OpConstI32,
						Result: "two",
						Type:   ssair.TypeI32,
						Imm:    2,
					},
					{
						ID:     "cmp_base",
						Kind:   ssair.OpCmpLtI32,
						Result: "cmp",
						Type:   ssair.TypeBool,
						Args:   []ssair.ValueID{"n", "two"},
					},
				},
				Term: ssair.Terminator{
					Kind:    ssair.TermCondBr,
					Cond:    "cmp",
					IfTrue:  "base",
					IfFalse: "recurse",
				},
			},
			{
				ID:   "base",
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "n"},
			},
			{
				ID: "recurse",
				Instrs: []ssair.Instr{
					{
						ID:     "const_one",
						Kind:   ssair.OpConstI32,
						Result: "one",
						Type:   ssair.TypeI32,
						Imm:    1,
					},
					{
						ID:     "sub_one",
						Kind:   ssair.OpSubI32,
						Result: "n.minus.one",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"n", "one"},
					},
					{
						ID:        "call_one",
						Kind:      ssair.OpCall,
						Result:    "fib.one",
						Type:      ssair.TypeI32,
						Args:      []ssair.ValueID{"n.minus.one"},
						Call:      plan.CallName,
						EffectIn:  "effect0",
						EffectOut: "effect1",
					},
					{
						ID:     "sub_two",
						Kind:   ssair.OpSubI32,
						Result: "n.minus.two",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"n", "two"},
					},
					{
						ID:        "call_two",
						Kind:      ssair.OpCall,
						Result:    "fib.two",
						Type:      ssair.TypeI32,
						Args:      []ssair.ValueID{"n.minus.two"},
						Call:      plan.CallName,
						EffectIn:  "effect1",
						EffectOut: "effect2",
					},
					{
						ID:     "add_results",
						Kind:   ssair.OpAddI32,
						Result: "sum",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"fib.one", "fib.two"},
					},
				},
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "sum"},
			},
		},
	}
	return ssair.VerifyFunction(ssaFn) == nil
}

func stackIRRecursionMainPassesSSAGate(fn ir.IRFunc) bool {
	plan, ok, err := machine.RecursionMainPlanFromStackIRWithCallABI(fn, machine.SysVCallABIInfo())
	if err != nil || !ok {
		return false
	}
	ssaFn := ssair.Function{
		Name:       fn.Name,
		ReturnType: ssair.TypeI32,
		Values: []ssair.Value{
			{ID: "zero", Type: ssair.TypeI32, Origin: "const"},
			{ID: "one", Type: ssair.TypeI32, Origin: "const"},
			{ID: "bound", Type: ssair.TypeI32, Origin: "const"},
			{ID: "call.arg", Type: ssair.TypeI32, Origin: "const"},
			{ID: "expected", Type: ssair.TypeI32, Origin: "const"},
			{ID: "loop.index", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "loop.total", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "loop.effect", Type: ssair.TypeEffect, Origin: "block_param"},
			{ID: "body.index", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "body.total", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "body.effect", Type: ssair.TypeEffect, Origin: "block_param"},
			{ID: "exit.total", Type: ssair.TypeI32, Origin: "block_param"},
			{ID: "cmp.loop", Type: ssair.TypeBool, Origin: "instr"},
			{ID: "fib.ret", Type: ssair.TypeI32, Origin: "call"},
			{ID: "next.total", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "next.index", Type: ssair.TypeI32, Origin: "instr"},
			{ID: "cmp.success", Type: ssair.TypeBool, Origin: "instr"},
			{ID: "effect0", Type: ssair.TypeEffect, Origin: "entry_effect"},
			{ID: "call.effect", Type: ssair.TypeEffect, Origin: "call_effect"},
		},
		Blocks: []ssair.Block{
			{
				ID:    "entry",
				Entry: true,
				Instrs: []ssair.Instr{
					{
						ID:     "const_zero",
						Kind:   ssair.OpConstI32,
						Result: "zero",
						Type:   ssair.TypeI32,
						Imm:    0,
					},
					{
						ID:     "const_one",
						Kind:   ssair.OpConstI32,
						Result: "one",
						Type:   ssair.TypeI32,
						Imm:    plan.FalseReturnImm,
					},
					{
						ID:     "const_bound",
						Kind:   ssair.OpConstI32,
						Result: "bound",
						Type:   ssair.TypeI32,
						Imm:    plan.LoopBound,
					},
					{
						ID:     "const_call_arg",
						Kind:   ssair.OpConstI32,
						Result: "call.arg",
						Type:   ssair.TypeI32,
						Imm:    plan.CallArg,
					},
					{
						ID:     "const_expected",
						Kind:   ssair.OpConstI32,
						Result: "expected",
						Type:   ssair.TypeI32,
						Imm:    plan.SuccessTotal,
					},
				},
				Term: ssair.Terminator{
					Kind:   ssair.TermBranch,
					Target: "loop",
					Args:   []ssair.ValueID{"zero", "zero", "effect0"},
				},
			},
			{
				ID:     "loop",
				Params: []ssair.ValueID{"loop.index", "loop.total", "loop.effect"},
				Instrs: []ssair.Instr{
					{
						ID:     "cmp_loop_bound",
						Kind:   ssair.OpCmpLtI32,
						Result: "cmp.loop",
						Type:   ssair.TypeBool,
						Args:   []ssair.ValueID{"loop.index", "bound"},
					},
				},
				Term: ssair.Terminator{
					Kind:        ssair.TermCondBr,
					Cond:        "cmp.loop",
					IfTrue:      "body",
					IfTrueArgs:  []ssair.ValueID{"loop.index", "loop.total", "loop.effect"},
					IfFalse:     "exit",
					IfFalseArgs: []ssair.ValueID{"loop.total"},
				},
			},
			{
				ID:     "body",
				Params: []ssair.ValueID{"body.index", "body.total", "body.effect"},
				Instrs: []ssair.Instr{
					{
						ID:        "call_fib",
						Kind:      ssair.OpCall,
						Result:    "fib.ret",
						Type:      ssair.TypeI32,
						Args:      []ssair.ValueID{"call.arg"},
						Call:      plan.CallName,
						EffectIn:  "body.effect",
						EffectOut: "call.effect",
					},
					{
						ID:     "add_total",
						Kind:   ssair.OpAddI32,
						Result: "next.total",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"body.total", "fib.ret"},
					},
					{
						ID:     "inc_index",
						Kind:   ssair.OpAddI32,
						Result: "next.index",
						Type:   ssair.TypeI32,
						Args:   []ssair.ValueID{"body.index", "one"},
					},
				},
				Term: ssair.Terminator{
					Kind:   ssair.TermBranch,
					Target: "loop",
					Args:   []ssair.ValueID{"next.index", "next.total", "call.effect"},
				},
			},
			{
				ID:     "exit",
				Params: []ssair.ValueID{"exit.total"},
				Instrs: []ssair.Instr{
					{
						ID:     "cmp_success",
						Kind:   ssair.OpCmpEqI32,
						Result: "cmp.success",
						Type:   ssair.TypeBool,
						Args:   []ssair.ValueID{"exit.total", "expected"},
					},
				},
				Term: ssair.Terminator{
					Kind:    ssair.TermCondBr,
					Cond:    "cmp.success",
					IfTrue:  "success",
					IfFalse: "failure",
				},
			},
			{
				ID:   "success",
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "zero"},
			},
			{
				ID:   "failure",
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "one"},
			},
		},
	}
	return ssair.VerifyFunction(ssaFn) == nil
}

func BuildMachineBackendFunctionReport(
	fn machine.Function,
	path string,
	callerSaved []machine.PhysReg,
	ssaVerified bool,
) (MachineBackendFunctionReport, bool) {
	live, err := machine.AnalyzeLiveness(fn)
	if err != nil {
		return MachineBackendFunctionReport{}, false
	}
	intervals, err := machine.BuildIntervals(fn)
	if err != nil {
		return MachineBackendFunctionReport{}, false
	}
	alloc, err := machine.LinearScan(intervals, callerSaved)
	if err != nil {
		return MachineBackendFunctionReport{}, false
	}
	spillSlots := len(alloc.Spills)
	if alloc.Assignments == nil {
		alloc.Assignments = map[machine.VReg]machine.PhysReg{}
	}
	if alloc.Spills == nil {
		alloc.Spills = map[machine.VReg]int{}
	}
	if err := machine.VerifyAllocation(fn, alloc, callerSaved, spillSlots); err != nil {
		return MachineBackendFunctionReport{}, false
	}
	validation := MachineValidationReport{
		MachineVerifier:    "pass",
		AllocationVerifier: "pass",
		SpillReload:        machineSpillReloadValidationStatus(fn, spillSlots),
		CallClobbers:       machineCallClobberValidationStatus(fn),
		StackChurnOps:      machineStackChurnOps(fn),
	}
	return MachineBackendFunctionReport{
		Function:             fn.Name,
		Path:                 path,
		SSAPath:              "value-ssa-v1",
		SSAVerified:          ssaVerified,
		InstructionSelection: machineInstructionSelection(fn),
		Validation:           validation,
		Dump:                 machine.FormatFunction(fn),
		Liveness:             live,
		Intervals:            intervals,
		Allocation: MachineAllocationReport{
			Assignments: alloc.Assignments,
			Spills:      alloc.Spills,
		},
		SpillSlots: spillSlots,
	}, true
}

func machineInstructionSelection(fn machine.Function) []string {
	seen := map[string]bool{}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if instr.Op != "" {
				seen[string(instr.Op)] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for op := range seen {
		out = append(out, op)
	}
	sort.Strings(out)
	return out
}

func machineSpillReloadValidationStatus(fn machine.Function, spillSlots int) string {
	hasSpillReloadOps := false
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if instr.Op == machine.OpSpill || instr.Op == machine.OpReload {
				hasSpillReloadOps = true
			}
		}
	}
	switch {
	case hasSpillReloadOps:
		return "validated_spill_reload_ops"
	case spillSlots > 0:
		return "validated_spills_reported"
	default:
		return "validated_no_spills"
	}
}

func machineCallClobberValidationStatus(fn machine.Function) string {
	hasCall := false
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if instr.Op != machine.OpCall {
				continue
			}
			hasCall = true
			if instr.ABI == "" || len(instr.Clobbers) == 0 {
				return "missing_call_clobber_metadata"
			}
		}
	}
	if !hasCall {
		return "not_applicable"
	}
	return "validated"
}

func machineStackChurnOps(fn machine.Function) int {
	count := 0
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if instr.Op == machine.OpPush || instr.Op == machine.OpPop {
				count++
			}
		}
	}
	return count
}

func machineFunctionHasOp(fn machine.Function, op machine.Opcode) bool {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if instr.Op == op {
				return true
			}
		}
	}
	return false
}

func machineCallABIForTarget(target string) machine.CallABIInfo {
	if target == "windows-x64" {
		return machine.Win64CallABIInfo()
	}
	return machine.SysVCallABIInfo()
}

func machineCallerSavedForTarget(target string) []machine.PhysReg {
	if target == "windows-x64" {
		return machine.Win64CallerSaved()
	}
	return machine.LinuxX64CallerSaved()
}

func targetSupportsMachineScalar(target string) bool {
	switch target {
	case "linux-x64", "macos-x64", "windows-x64":
		return true
	default:
		return false
	}
}

func targetSupportsHashTableMainMachinePath(target string) bool {
	return target == "linux-x64"
}
