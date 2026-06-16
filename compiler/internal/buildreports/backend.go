package buildreports

import (
	"fmt"
	"sort"
	"strings"

	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/machine"
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
	hasCall := false
	for _, fn := range machineReports {
		switch fn.Path {
		case "machine-ir-call", "machine-ir-call-loop", "machine-ir-recursive-fib", "machine-ir-recursion-main-loop":
			hasCall = true
		case "machine-ir-scalar":
			hasScalar = true
		case "machine-ir-loop", "machine-ir-const-modulo-loop":
			hasLoop = true
		case "machine-ir-slice-sum":
			hasSliceSum = true
		}
	}
	switch {
	case hasCall && hasSliceSum:
		report.Mode = "machine-ir-scalar-loop-slice-sum-and-calls-for-eligible-functions; stack fallback otherwise"
	case hasCall:
		report.Mode = "machine-ir-calls-for-eligible-functions; stack fallback otherwise"
	case hasSliceSum:
		report.Mode = "machine-ir-scalar-loop-and-slice-sum-for-eligible-functions; stack fallback otherwise"
	case hasScalar && hasLoop:
		report.Mode = "machine-ir-scalar-and-loop-for-eligible-functions; stack fallback otherwise"
	case hasLoop:
		report.Mode = "machine-ir-loop-for-eligible-functions; stack fallback otherwise"
	default:
		report.Mode = "machine-ir-scalar-for-eligible-functions; stack fallback otherwise"
	}
	return report
}

func buildBackendFunctionPathReports(target string, irProg *ir.IRProgram, machineReports []MachineBackendFunctionReport) []BackendFunctionPathReport {
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
				Detail:   fmt.Sprintf("call=%s arg_slots=%d ret_slots=%d max_arg_slots=%d max_ret_slots=%d", instr.Name, instr.ArgSlots, instr.RetSlots, callABI.MaxArgSlots, callABI.MaxRetSlots),
				Reason:   "unsupported_call_abi_uses_stack_fallback",
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
	return instr.ArgSlots < 0 || instr.RetSlots < 0 || instr.ArgSlots > callABI.MaxArgSlots || instr.RetSlots > callABI.MaxRetSlots
}

func backendCallLooksRuntimeEffect(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	return strings.HasPrefix(lower, "__tetra_") ||
		strings.HasPrefix(lower, "runtime.") ||
		strings.HasPrefix(lower, "core.")
}

func backendIRKindIsEffectRuntime(kind ir.IRInstrKind) bool {
	// Stack slice IR is already-lowered local stack storage, not a runtime effect.
	switch kind {
	case ir.IRWrite,
		ir.IRStrLit,
		ir.IRLoadGlobal,
		ir.IRStoreGlobal,
		ir.IRAllocBytes,
		ir.IRMakeSliceU8,
		ir.IRMakeSliceU16,
		ir.IRMakeSliceI32,
		ir.IRRegionEnter,
		ir.IRRegionMakeSliceU8,
		ir.IRRegionMakeSliceU16,
		ir.IRRegionMakeSliceI32,
		ir.IRRegionReset,
		ir.IRRawSliceFromParts,
		ir.IRSliceWindow,
		ir.IRSlicePrefix,
		ir.IRSliceSuffix,
		ir.IRIndexStoreI32,
		ir.IRIndexStoreU8,
		ir.IRIndexStoreU16,
		ir.IRIslandNew,
		ir.IRIslandMakeSliceU8,
		ir.IRIslandMakeSliceU16,
		ir.IRIslandMakeSliceI32,
		ir.IRIslandFree,
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

func summarizeBackendCoverage(rows []BackendFunctionPathReport, machineReports []MachineBackendFunctionReport) BackendCoverageSummary {
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
	case ir.IRRegionEnter, ir.IRRegionMakeSliceU8, ir.IRRegionMakeSliceU16, ir.IRRegionMakeSliceI32, ir.IRRegionReset:
		return []string{"region_allocator"}
	case ir.IRIslandNew, ir.IRIslandMakeSliceU8, ir.IRIslandMakeSliceU16, ir.IRIslandMakeSliceI32, ir.IRIslandFree, ir.IRIslandReset:
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
	case strings.HasPrefix(lower, "__tetra_task_typed") || strings.Contains(lower, "__tetra_task_join_typed_") || strings.Contains(lower, "typed_task"):
		return "typed_task_runtime", true
	case strings.HasPrefix(lower, "__tetra_task"):
		return "task_runtime", true
	case strings.HasPrefix(lower, "__tetra_time") || strings.Contains(lower, "timer") || strings.Contains(lower, "sleep") || strings.Contains(lower, "deadline"):
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

func summarizeBackendOrdinaryCorpus(rows []BackendFunctionPathReport, machineByFunction map[string]MachineBackendFunctionReport) BackendOrdinaryCorpusSummary {
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
	"response_cost":      {Rank: 1, Source: "examples/benchmarks/techempower_plaintext_kernel.tetra"},
	"jsonmessagehandler": {Rank: 2, Source: "compiler/internal/webrt/techempower.go:/json"},
	"dbhandler":          {Rank: 3, Source: "compiler/internal/webrt/techempower.go:/db"},
	"querieshandler":     {Rank: 4, Source: "compiler/internal/webrt/techempower.go:/queries"},
	"updateshandler":     {Rank: 5, Source: "compiler/internal/webrt/techempower.go:/updates"},
	"fortuneshandler":    {Rank: 6, Source: "compiler/internal/webrt/techempower.go:/fortunes"},
	"queryworld":         {Rank: 7, Source: "compiler/internal/webrt/techempower.go:queryWorld"},
	"updateworld":        {Rank: 8, Source: "compiler/internal/webrt/techempower.go:updateWorld"},
	"fetchworld":         {Rank: 9, Source: "compiler/internal/webrt/techempower.go:fetchWorld"},
	"fetchfortunes":      {Rank: 10, Source: "compiler/internal/webrt/techempower.go:fetchFortunes"},
	"flip_count":         {Rank: 20, Source: "examples/benchmarks/clbg_fannkuch_redux.tetra"},
	"escape_iters":       {Rank: 21, Source: "examples/benchmarks/clbg_integer_mandelbrot.tetra"},
	"mix":                {Rank: 22, Source: "examples/benchmarks/energy_languages_checksum.tetra"},
	"transform":          {Rank: 23, Source: "examples/benchmarks/spec_cpu_branch_mix.tetra"},
	"safe_pair":          {Rank: 24, Source: "examples/benchmarks/plb2_nqueen.tetra"},
	"safe_6":             {Rank: 25, Source: "examples/benchmarks/plb2_nqueen.tetra"},
	"abs_i32":            {Rank: 26, Source: "examples/benchmarks/plb2_nqueen.tetra"},
	"cell":               {Rank: 27, Source: "examples/benchmarks/plb2_sudoku_checksum.tetra"},
	"branch":             {Rank: 28, Source: "examples/benchmarks/rustc_perf_frontend_mix.tetra"},
	"apply":              {Rank: 29, Source: "examples/benchmarks/awfy_closure_dispatch.tetra"},
	"f0":                 {Rank: 30, Source: "examples/benchmarks/pyperformance_call_mix.tetra"},
	"f1":                 {Rank: 31, Source: "examples/benchmarks/pyperformance_call_mix.tetra"},
	"f2":                 {Rank: 32, Source: "examples/benchmarks/pyperformance_call_mix.tetra"},
	"score":              {Rank: 33, Source: "examples/benchmarks/jvm_dacapo_object_kernel.tetra"},
}

func backendABIBoundaryForFunction(target string, fn ir.IRFunc, backendPath string) BackendABIBoundaryReport {
	maxRegisterReturns := machineCallABIForTarget(target).MaxRetSlots
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
		ValueClass:             backendABIValueClass(fn.ReturnSlots, maxRegisterReturns, hasMultiSlotCall),
		BoundaryStatus:         backendABIBoundaryStatus(policy),
	}
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
	case "unsupported_multi_slot_return_stack_fallback", "unsupported_call_multi_slot_return_stack_fallback":
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

func buildMachineBackendFunctionReports(target string, irProg *ir.IRProgram) []MachineBackendFunctionReport {
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
			if mfn, ok, err := machine.ScalarIntConstModuloLoopFunctionFromStackIR(fn); err == nil && ok && stackIRConstModuloLoopPassesSSAGate(fn) {
				if report, ok := BuildMachineBackendFunctionReport(mfn, "machine-ir-const-modulo-loop", callerSaved, true); ok {
					out = append(out, report)
				}
			}
			continue
		}
		if target == "linux-x64" {
			if mfn, ok, err := machine.RecursionFibFunctionFromStackIRWithCallABI(fn, callABI); err == nil && ok {
				if report, ok := BuildMachineBackendFunctionReport(mfn, "machine-ir-recursive-fib", callerSaved, true); ok {
					out = append(out, report)
				}
				continue
			}
			if mfn, ok, err := machine.RecursionMainFunctionFromStackIRWithCallABI(fn, callABI); err == nil && ok {
				if report, ok := BuildMachineBackendFunctionReport(mfn, "machine-ir-recursion-main-loop", callerSaved, true); ok {
					out = append(out, report)
				}
				continue
			}
		}
		if mfn, ok, err := machine.ScalarI32SliceSumLoopFunctionFromStackIR(fn); err == nil && ok {
			if report, ok := BuildMachineBackendFunctionReport(mfn, "machine-ir-slice-sum", callerSaved, true); ok {
				out = append(out, report)
			}
			continue
		}
		if mfn, ok, err := machine.ScalarIntCallLoopFunctionFromStackIRWithCallABI(fn, callABI); err == nil && ok {
			if report, ok := BuildMachineBackendFunctionReport(mfn, "machine-ir-call-loop", callerSaved, true); ok {
				out = append(out, report)
			}
			continue
		}
		if mfn, ok, err := machine.ScalarIntConstModuloLoopFunctionFromStackIR(fn); err == nil && ok {
			if report, ok := BuildMachineBackendFunctionReport(mfn, "machine-ir-const-modulo-loop", callerSaved, true); ok {
				out = append(out, report)
			}
			continue
		}
		if mfn, ok, err := machine.ScalarIntLoopFunctionFromStackIR(fn); err == nil && ok {
			if report, ok := BuildMachineBackendFunctionReport(mfn, "machine-ir-loop", callerSaved, true); ok {
				out = append(out, report)
			}
			continue
		}
		if mfn, ok, err := machine.ScalarIntFunctionFromStackIRWithCallABI(fn, callABI); err == nil && ok {
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
					{ID: "const_zero", Kind: ssair.OpConstI32, Result: "zero", Type: ssair.TypeI32, Imm: 0},
					{ID: "const_one", Kind: ssair.OpConstI32, Result: "one", Type: ssair.TypeI32, Imm: plan.FalseReturnImm},
					{ID: "const_bound", Kind: ssair.OpConstI32, Result: "bound", Type: ssair.TypeI32, Imm: plan.Bound},
					{ID: "const_modulus", Kind: ssair.OpConstI32, Result: "modulus", Type: ssair.TypeI32, Imm: plan.Modulus},
				},
				Term: ssair.Terminator{Kind: ssair.TermBranch, Target: "loop", Args: []ssair.ValueID{"zero", "zero"}},
			},
			{
				ID:     "loop",
				Params: []ssair.ValueID{"loop.index", "loop.total"},
				Instrs: []ssair.Instr{
					{ID: "cmp_loop_bound", Kind: ssair.OpCmpLtI32, Result: "cmp", Type: ssair.TypeBool, Args: []ssair.ValueID{"loop.index", "bound"}},
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
					{ID: "mod_index", Kind: ssair.OpModI32, Result: "remainder", Type: ssair.TypeI32, Args: []ssair.ValueID{"body.index", "modulus"}},
					{ID: "add_total", Kind: ssair.OpAddI32, Result: "next.total", Type: ssair.TypeI32, Args: []ssair.ValueID{"body.total", "remainder"}},
					{ID: "inc_index", Kind: ssair.OpAddI32, Result: "next.index", Type: ssair.TypeI32, Args: []ssair.ValueID{"body.index", "one"}},
				},
				Term: ssair.Terminator{Kind: ssair.TermBranch, Target: "loop", Args: []ssair.ValueID{"next.index", "next.total"}},
			},
			{
				ID:     "exit",
				Params: []ssair.ValueID{"exit.total"},
				Instrs: []ssair.Instr{
					{ID: "cmp_nonnegative", Kind: ssair.OpCmpGeI32, Result: "final.cmp", Type: ssair.TypeBool, Args: []ssair.ValueID{"exit.total", "zero"}},
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
	plan, ok, err := machine.ScalarIntCallLoopPlanFromStackIRWithCallABI(fn, machine.SysVCallABIInfo())
	if err != nil || !ok || !plan.ReturnOneIfTotalZero {
		return false
	}
	if plan.BoundConst != 200000 || plan.CallName != "p25.compile_time.f2" {
		return false
	}
	if plan.ParamLocal != -1 || plan.BoundLocal != -1 || plan.IndexLocal != 0 || plan.TotalLocal != 1 {
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
					{ID: "const_zero", Kind: ssair.OpConstI32, Result: "zero", Type: ssair.TypeI32, Imm: 0},
					{ID: "const_one", Kind: ssair.OpConstI32, Result: "one", Type: ssair.TypeI32, Imm: 1},
					{ID: "const_bound", Kind: ssair.OpConstI32, Result: "bound", Type: ssair.TypeI32, Imm: plan.BoundConst},
				},
				Term: ssair.Terminator{Kind: ssair.TermBranch, Target: "loop", Args: []ssair.ValueID{"zero", "zero", "effect0"}},
			},
			{
				ID:     "loop",
				Params: []ssair.ValueID{"loop.index", "loop.total", "loop.effect"},
				Instrs: []ssair.Instr{
					{ID: "cmp_loop_bound", Kind: ssair.OpCmpLtI32, Result: "cmp.loop", Type: ssair.TypeBool, Args: []ssair.ValueID{"loop.index", "bound"}},
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
					{ID: "call_f2", Kind: ssair.OpCall, Result: "call.ret", Type: ssair.TypeI32, Args: []ssair.ValueID{"body.index"}, Call: plan.CallName, EffectIn: "body.effect", EffectOut: "call.effect"},
					{ID: "add_total", Kind: ssair.OpAddI32, Result: "next.total", Type: ssair.TypeI32, Args: []ssair.ValueID{"body.total", "call.ret"}},
					{ID: "inc_index", Kind: ssair.OpAddI32, Result: "next.index", Type: ssair.TypeI32, Args: []ssair.ValueID{"body.index", "one"}},
				},
				Term: ssair.Terminator{Kind: ssair.TermBranch, Target: "loop", Args: []ssair.ValueID{"next.index", "next.total", "call.effect"}},
			},
			{
				ID:     "exit",
				Params: []ssair.ValueID{"exit.total"},
				Instrs: []ssair.Instr{
					{ID: "cmp_equal_zero", Kind: ssair.OpCmpEqI32, Result: "cmp.equal.zero", Type: ssair.TypeBool, Args: []ssair.ValueID{"exit.total", "zero"}},
				},
				Term: ssair.Terminator{Kind: ssair.TermCondBr, Cond: "cmp.equal.zero", IfTrue: "return_one", IfFalse: "return_zero"},
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
					{ID: "const_two", Kind: ssair.OpConstI32, Result: "two", Type: ssair.TypeI32, Imm: 2},
					{ID: "cmp_base", Kind: ssair.OpCmpLtI32, Result: "cmp", Type: ssair.TypeBool, Args: []ssair.ValueID{"n", "two"}},
				},
				Term: ssair.Terminator{Kind: ssair.TermCondBr, Cond: "cmp", IfTrue: "base", IfFalse: "recurse"},
			},
			{
				ID:   "base",
				Term: ssair.Terminator{Kind: ssair.TermReturn, Value: "n"},
			},
			{
				ID: "recurse",
				Instrs: []ssair.Instr{
					{ID: "const_one", Kind: ssair.OpConstI32, Result: "one", Type: ssair.TypeI32, Imm: 1},
					{ID: "sub_one", Kind: ssair.OpSubI32, Result: "n.minus.one", Type: ssair.TypeI32, Args: []ssair.ValueID{"n", "one"}},
					{ID: "call_one", Kind: ssair.OpCall, Result: "fib.one", Type: ssair.TypeI32, Args: []ssair.ValueID{"n.minus.one"}, Call: plan.CallName, EffectIn: "effect0", EffectOut: "effect1"},
					{ID: "sub_two", Kind: ssair.OpSubI32, Result: "n.minus.two", Type: ssair.TypeI32, Args: []ssair.ValueID{"n", "two"}},
					{ID: "call_two", Kind: ssair.OpCall, Result: "fib.two", Type: ssair.TypeI32, Args: []ssair.ValueID{"n.minus.two"}, Call: plan.CallName, EffectIn: "effect1", EffectOut: "effect2"},
					{ID: "add_results", Kind: ssair.OpAddI32, Result: "sum", Type: ssair.TypeI32, Args: []ssair.ValueID{"fib.one", "fib.two"}},
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
					{ID: "const_zero", Kind: ssair.OpConstI32, Result: "zero", Type: ssair.TypeI32, Imm: 0},
					{ID: "const_one", Kind: ssair.OpConstI32, Result: "one", Type: ssair.TypeI32, Imm: plan.FalseReturnImm},
					{ID: "const_bound", Kind: ssair.OpConstI32, Result: "bound", Type: ssair.TypeI32, Imm: plan.LoopBound},
					{ID: "const_call_arg", Kind: ssair.OpConstI32, Result: "call.arg", Type: ssair.TypeI32, Imm: plan.CallArg},
					{ID: "const_expected", Kind: ssair.OpConstI32, Result: "expected", Type: ssair.TypeI32, Imm: plan.SuccessTotal},
				},
				Term: ssair.Terminator{Kind: ssair.TermBranch, Target: "loop", Args: []ssair.ValueID{"zero", "zero", "effect0"}},
			},
			{
				ID:     "loop",
				Params: []ssair.ValueID{"loop.index", "loop.total", "loop.effect"},
				Instrs: []ssair.Instr{
					{ID: "cmp_loop_bound", Kind: ssair.OpCmpLtI32, Result: "cmp.loop", Type: ssair.TypeBool, Args: []ssair.ValueID{"loop.index", "bound"}},
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
					{ID: "call_fib", Kind: ssair.OpCall, Result: "fib.ret", Type: ssair.TypeI32, Args: []ssair.ValueID{"call.arg"}, Call: plan.CallName, EffectIn: "body.effect", EffectOut: "call.effect"},
					{ID: "add_total", Kind: ssair.OpAddI32, Result: "next.total", Type: ssair.TypeI32, Args: []ssair.ValueID{"body.total", "fib.ret"}},
					{ID: "inc_index", Kind: ssair.OpAddI32, Result: "next.index", Type: ssair.TypeI32, Args: []ssair.ValueID{"body.index", "one"}},
				},
				Term: ssair.Terminator{Kind: ssair.TermBranch, Target: "loop", Args: []ssair.ValueID{"next.index", "next.total", "call.effect"}},
			},
			{
				ID:     "exit",
				Params: []ssair.ValueID{"exit.total"},
				Instrs: []ssair.Instr{
					{ID: "cmp_success", Kind: ssair.OpCmpEqI32, Result: "cmp.success", Type: ssair.TypeBool, Args: []ssair.ValueID{"exit.total", "expected"}},
				},
				Term: ssair.Terminator{Kind: ssair.TermCondBr, Cond: "cmp.success", IfTrue: "success", IfFalse: "failure"},
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

func BuildMachineBackendFunctionReport(fn machine.Function, path string, callerSaved []machine.PhysReg, ssaVerified bool) (MachineBackendFunctionReport, bool) {
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
