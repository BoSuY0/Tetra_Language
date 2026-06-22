package compiler

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"tetra_language/compiler/internal/allocplan"
	buildreports "tetra_language/compiler/internal/buildreports"
	"tetra_language/compiler/internal/buildruntime"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/lower"
	"tetra_language/compiler/internal/machine"
	"tetra_language/compiler/internal/memoryfacts"
	"tetra_language/compiler/internal/memoryfacts/fromplir"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/ramcontract"
	"tetra_language/compiler/internal/semantics"
	"tetra_language/compiler/internal/validation"
)

// ---- reports.go ----

type reportEnvelope = buildreports.ReportEnvelope

type boundsReport = buildreports.BoundsReport
type boundsTotals = buildreports.BoundsTotals
type boundsFunctionRow = buildreports.BoundsFunctionRow
type boundsCheckSite = buildreports.BoundsCheckSite

type proofReport = buildreports.ProofReport
type proofEvidence = buildreports.ProofEvidence

type allocReport = buildreports.AllocReport
type allocTotals = buildreports.AllocTotals
type allocFunctionRow = buildreports.AllocFunctionRow
type allocationDecision = buildreports.AllocationDecision
type allocationPlanReport = buildreports.AllocationPlanReport

type backendReport = buildreports.BackendReport
type backendCoverageSummary = buildreports.BackendCoverageSummary
type backendOrdinaryCorpusSummary = buildreports.BackendOrdinaryCorpusSummary
type backendABIBoundarySummary = buildreports.BackendABIBoundarySummary
type backendRuntimeObjectPlan = buildreports.BackendRuntimeObjectPlan
type backendFunctionPathReport = buildreports.BackendFunctionPathReport
type backendABIBoundaryReport = buildreports.BackendABIBoundaryReport
type machineBackendFunctionReport = buildreports.MachineBackendFunctionReport
type machineAllocationReport = buildreports.MachineAllocationReport
type machineValidationReport = buildreports.MachineValidationReport

type layoutReport = buildreports.LayoutReport
type layoutSummary = buildreports.LayoutSummary
type layoutDecisionRow = buildreports.LayoutDecisionRow
type layoutFieldRow = buildreports.LayoutFieldRow

type perfReport = buildreports.PerfReport
type performanceBlockerRow = buildreports.PerformanceBlockerRow
type performanceBenchmarkExplanation = buildreports.PerformanceBenchmarkExplanation

type actorTransferReport = buildreports.ActorTransferReport
type actorTransferTotals = buildreports.ActorTransferTotals
type actorTransferRow = buildreports.ActorTransferRow
type actorMailboxRow = buildreports.ActorMailboxRow

// ---- reports_actor_helpers.go ----

func buildAllocReport(prog *ir.IRProgram, target string) allocReport {
	return buildreports.BuildAllocReport(prog, target)
}

func buildActorTransferReport(
	checked *semantics.CheckedProgram,
	target string,
) actorTransferReport {
	return buildreports.BuildActorTransferReport(checked, target)
}

func writeReport(path string, data any) error {
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := f.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	enc := json.NewEncoder(f)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func formatExplainText(
	target string,
	bounds boundsReport,
	alloc *allocplan.Plan,
	plirProg *plir.Program,
) string {
	var b strings.Builder
	fmt.Fprintf(&b, "target: %s\n", target)
	fmt.Fprintf(&b, "bounds checks removed: %d\n", bounds.Totals.Removed)
	fmt.Fprintf(&b, "bounds checks left: %d\n", bounds.Totals.Left)
	if alloc != nil {
		fmt.Fprintf(&b, "planned heap allocations: %d\n", alloc.Totals.Heap)
		fmt.Fprintf(&b, "planned stack allocations: %d\n", alloc.Totals.Stack)
		fmt.Fprintf(&b, "explicit island allocations: %d\n\n", alloc.Totals.ExplicitIsland)
		b.WriteString(allocplan.FormatText(alloc))
		b.WriteString("\n")
	}
	b.WriteString(plir.FormatText(plirProg))
	return b.String()
}

// ---- reports_backend.go ----

func buildBackendReport(target string, irProg *ir.IRProgram) backendReport {
	return buildreports.BuildBackendReport(target, irProg)
}

func buildMachineBackendFunctionReport(
	fn machine.Function,
	path string,
	callerSaved []machine.PhysReg,
	ssaVerified bool,
) (machineBackendFunctionReport, bool) {
	return buildreports.BuildMachineBackendFunctionReport(fn, path, callerSaved, ssaVerified)
}

func allocationPlanOptionsForTarget(target string) allocplan.Options {
	return allocplan.Options{
		EnableStackLowering:    targetSupportsStackAllocationLowering(target),
		EnableSmallHeapRuntime: target == "linux-x64",
		EnableRegionPlanning:   target == "linux-x64",
		EnableRegionLowering:   target == "linux-x64",
	}
}

func lowerOptionsForTarget(target string) lower.Options {
	return lower.Options{
		StackAllocationLowering:    targetSupportsStackAllocationLowering(target),
		FunctionTempRegionLowering: target == "linux-x64",
	}
}

func lowerOptionsForBuild(target string, opt BuildOptions) lower.Options {
	out := lowerOptionsForTarget(target)
	out.OwnedAllocDropLowering = opt.OwnedAllocDropLowering
	return out
}

func targetSupportsStackAllocationLowering(target string) bool {
	switch target {
	case "linux-x64", "macos-x64", "windows-x64":
		return true
	default:
		return false
	}
}

// ---- reports_bounds.go ----

func buildProofReport(plirProg *plir.Program, bounds boundsReport, target string) proofReport {
	return buildreports.BuildProofReport(plirProg, bounds, target)
}

func wrapAllocationPlanReport(plan *allocplan.Plan, target string) allocationPlanReport {
	return buildreports.WrapAllocationPlanReport(plan, target)
}

func validateAllocationPlanReport(plan *allocplan.Plan, report allocationPlanReport) error {
	return buildreports.ValidateAllocationPlanReport(plan, report)
}

func allocationPlanTargetStorageScope(triple string) (string, string, error) {
	return buildreports.AllocationPlanTargetStorageScope(triple)
}

func buildBoundsReport(
	prog *ir.IRProgram,
	checked *semantics.CheckedProgram,
	target string,
) boundsReport {
	return buildreports.BuildBoundsReport(prog, checked, target)
}

// ---- reports_emit.go ----

func writePLIRReports(outputPath string, plirProg *plir.Program) error {
	if err := writeReport(outputPath+".plir.json", plirProg); err != nil {
		return err
	}
	if err := os.WriteFile(
		outputPath+".plir.txt",
		[]byte(plir.FormatText(plirProg)),
		0o644,
	); err != nil {
		return err
	}
	return nil
}

func emitExplainReports(
	outputPath string,
	target string,
	checked *semantics.CheckedProgram,
	opt BuildOptions,
) error {
	if !opt.Explain && !opt.EmitPLIR && !opt.EmitProof && !opt.EmitBoundsReport &&
		!opt.EmitAllocReport &&
		!opt.EmitMemoryReport &&
		!ramContractRequested(opt) {
		return nil
	}
	plirOnly := opt.EmitPLIR && !opt.Explain && !opt.EmitProof && !opt.EmitBoundsReport &&
		!opt.EmitAllocReport &&
		!opt.EmitMemoryReport &&
		!ramContractRequested(opt)
	plirProg, err := plir.FromCheckedProgram(checked)
	if err != nil {
		return err
	}
	if err := plir.VerifyProgram(plirProg); err != nil {
		return err
	}
	if plirOnly {
		return writePLIRReports(outputPath, plirProg)
	}
	allocPlan, err := allocplan.FromPLIRWithOptions(
		plirProg,
		allocationPlanOptionsForTarget(target),
	)
	if err != nil {
		return err
	}
	irProg, err := lower.LowerWithOptions(checked, lowerOptionsForBuild(target, opt))
	if err != nil {
		return err
	}
	if err := validation.ValidateAllocationLowering(allocPlan, irProg); err != nil {
		return err
	}
	if _, err := validation.CheckBoundsProofsWithPLIR(irProg, plirProg); err != nil {
		return err
	}
	bounds := buildBoundsReport(irProg, checked, target)
	if opt.Explain || opt.EmitPLIR {
		if err := writePLIRReports(outputPath, plirProg); err != nil {
			return err
		}
	}
	if opt.Explain || opt.EmitProof {
		if err := writeReport(
			outputPath+".proof.json",
			buildProofReport(plirProg, bounds, target),
		); err != nil {
			return err
		}
	}
	if opt.Explain || opt.EmitBoundsReport {
		if err := writeReport(outputPath+".bounds.json", bounds); err != nil {
			return err
		}
	}
	if opt.Explain || opt.EmitAllocReport {
		allocReport := wrapAllocationPlanReport(allocPlan, target)
		if err := validateAllocationPlanReport(allocPlan, allocReport); err != nil {
			return err
		}
		if err := writeReport(outputPath+".alloc.json", allocReport); err != nil {
			return err
		}
		if err := os.WriteFile(
			outputPath+".alloc.txt",
			[]byte(allocplan.FormatText(allocPlan)),
			0o644,
		); err != nil {
			return err
		}
	}
	if opt.EmitMemoryReport {
		graph, err := fromplir.FromPLIRAndAllocPlan(target, plirProg, allocPlan)
		if err != nil {
			return err
		}
		report := memoryfacts.BuildReportFromGraph(graph)
		if err := validateMemoryReportForEmission(graph, report); err != nil {
			return err
		}
		if err := writeReport(outputPath+".memory.json", report); err != nil {
			return err
		}
	}
	if opt.Explain || opt.EmitRAMContractReport || ramContractEnforcementRequested(opt) {
		ramReport := ramcontract.BuildReportFromAllocPlan(
			allocPlan,
			target,
			reportGitHead(),
			"tetra-compiler",
		)
		if err := ramcontract.ValidateReport(ramReport); err != nil {
			return fmt.Errorf("validate RAM contract report: %w", err)
		}
		gradeReport := ramcontract.BuildGradeReport(ramReport)
		if err := ramcontract.ValidateGradeReport(gradeReport); err != nil {
			return fmt.Errorf("validate memory grade report: %w", err)
		}
		proofStore := ramcontract.BuildProofStoreSummary(ramReport)
		if err := ramcontract.ValidateProofStoreSummary(proofStore); err != nil {
			return fmt.Errorf("validate proof store summary: %w", err)
		}
		pipeline := ramcontract.BuildPipelineCoverage(
			target,
			ramReport.GitHead,
			"tetra-compiler",
			buildEntrypointForOptions(opt),
			outputPath,
			[]string{
				"plir.VerifyProgram",
				"allocplan.VerifyPlan",
				"validation.ValidateAllocationLowering",
				"validation.CheckBoundsProofsWithPLIR",
				"ramcontract.ValidateReport",
			},
		)
		if err := ramcontract.ValidatePipelineCoverage(pipeline); err != nil {
			return fmt.Errorf("validate pipeline coverage: %w", err)
		}
		heapBlockers := ramcontract.BuildHeapBlockerReport(ramReport)
		if err := ramcontract.ValidateBlockerReport(heapBlockers, "heap"); err != nil {
			return fmt.Errorf("validate heap blocker report: %w", err)
		}
		copyBlockers := ramcontract.BuildCopyBlockerReport(ramReport)
		if err := ramcontract.ValidateBlockerReport(copyBlockers, "copy"); err != nil {
			return fmt.Errorf("validate copy blocker report: %w", err)
		}
		if opt.Explain || opt.EmitRAMContractReport {
			for _, report := range []struct {
				path string
				data any
			}{
				{path: outputPath + ".ram-contract.json", data: ramReport},
				{path: outputPath + ".memory-grade.json", data: gradeReport},
				{path: outputPath + ".proof-store-summary.json", data: proofStore},
				{path: outputPath + ".validation-pipeline-coverage.json", data: pipeline},
				{path: outputPath + ".heap-blockers.json", data: heapBlockers},
				{path: outputPath + ".copy-blockers.json", data: copyBlockers},
			} {
				if err := writeReport(report.path, report.data); err != nil {
					return err
				}
			}
			if err := os.WriteFile(
				outputPath+".ram-contract.txt",
				[]byte(formatRAMContractText(ramReport)),
				0o644,
			); err != nil {
				return err
			}
		}
		if err := ramcontract.Enforce(ramReport, ramcontract.EnforcementOptions{
			FailIfHeap:        opt.FailIfHeap,
			FailIfCopy:        opt.FailIfCopy,
			FailIfUnbounded:   opt.FailIfUnbounded,
			MemoryBudgetBytes: opt.MemoryBudgetBytes,
			ContractFile:      opt.RAMContractFile,
		}); err != nil {
			return err
		}
	}
	if opt.Explain {
		backend := buildBackendReport(target, irProg)
		if err := annotateBackendReportRuntimeObjectPlan(&backend, target, checked, opt); err != nil {
			return err
		}
		actorTransfers := buildActorTransferReport(checked, target)
		layout := buildLayoutReport(target, checked)
		if err := ValidateLayoutReport(layout); err != nil {
			return err
		}
		for _, report := range []struct {
			path string
			data any
		}{
			{path: outputPath + ".backend.json", data: backend},
			{path: outputPath + ".actor-transfer.json", data: actorTransfers},
			{path: outputPath + ".layout.json", data: layout},
			{path: outputPath + ".perf.json", data: buildPerformanceReport(target)},
		} {
			if err := writeReport(report.path, report.data); err != nil {
				return err
			}
		}
		if err := os.WriteFile(
			outputPath+".explain.txt",
			[]byte(formatExplainText(target, bounds, allocPlan, plirProg)),
			0o644,
		); err != nil {
			return err
		}
	}
	return nil
}

func ramContractRequested(opt BuildOptions) bool {
	return opt.EmitRAMContractReport || ramContractEnforcementRequested(opt)
}

func ramContractEnforcementRequested(opt BuildOptions) bool {
	return opt.FailIfHeap || opt.FailIfCopy || opt.FailIfUnbounded || opt.MemoryBudgetBytes > 0 ||
		strings.TrimSpace(opt.RAMContractFile) != ""
}

func buildEntrypointForOptions(opt BuildOptions) string {
	if opt.InterfaceOnly {
		return "InterfaceOnly"
	}
	switch opt.Emit {
	case EmitObject:
		return "buildObjectFileWithStatsOpt"
	case EmitLibrary:
		return "buildLibraryObjectWithStatsOpt"
	default:
		return "BuildFileWithStatsOpt"
	}
}

func reportGitHead() string {
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	head := strings.TrimSpace(string(out))
	if head == "" {
		return "unknown"
	}
	return head
}

func formatRAMContractText(report ramcontract.Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "RAM contract report\n")
	fmt.Fprintf(&b, "target: %s\n", report.Target)
	fmt.Fprintf(&b, "artifact grade: %s\n", report.Summary.ArtifactGrade)
	fmt.Fprintf(
		&b,
		"rows: %d heap: %d copy: %d unbounded: %d budget_bytes: %d\n",
		report.Summary.RowCount,
		report.Summary.HeapRows,
		report.Summary.CopyRows,
		report.Summary.UnboundedRows,
		report.Summary.BudgetBytes,
	)
	for _, row := range report.Rows {
		fmt.Fprintf(
			&b,
			"%s %s %s grade=%s placement=%s",
			row.Function,
			row.SiteID,
			row.Intent,
			row.ContractGrade,
			row.Placement,
		)
		if len(row.ProofIDs) > 0 {
			fmt.Fprintf(&b, " proof_ids=%s", strings.Join(row.ProofIDs, ","))
		}
		if len(row.Blockers) > 0 {
			fmt.Fprintf(&b, " blockers=%s", strings.Join(row.Blockers, ","))
		}
		if row.CopyReason != "" {
			fmt.Fprintf(&b, " copy_reason=%s", row.CopyReason)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func validateMemoryReportForEmission(graph *memoryfacts.Graph, report memoryfacts.Report) error {
	if err := memoryfacts.ValidateReportProjection(graph, report); err != nil {
		return fmt.Errorf("validate memory report projection: %w", err)
	}
	return nil
}

// ---- reports_layout_perf.go ----

const p21LayoutPolicy = buildreports.P21LayoutPolicy

func buildLayoutReport(target string, checked *semantics.CheckedProgram) layoutReport {
	return buildreports.BuildLayoutReport(target, checked)
}

func ValidateLayoutReport(report layoutReport) error {
	return buildreports.ValidateLayoutReport(report)
}

func buildPerformanceReport(target string) perfReport {
	return buildreports.BuildPerformanceReport(target)
}

func ValidatePerformanceBlockerReport(report perfReport) error {
	return buildreports.ValidatePerformanceBlockerReport(report)
}

// ---- reports_runtime_object.go ----

const (
	runtimeObjectPlanEvidenceClass  = "native_runtime_object_plan"
	runtimeObjectPlanEvidenceMethod = "native_link_runtime_object_plan_v1"
)

func annotateBackendReportRuntimeObjectPlan(
	report *backendReport,
	target string,
	checked *CheckedProgram,
	opt BuildOptions,
) error {
	if report == nil {
		return nil
	}
	plan, err := buildBackendRuntimeObjectPlan(target, opt.RuntimeObjectPath != "", checked)
	if err != nil {
		return err
	}
	report.Summary.RuntimeObjectPlan = plan
	return nil
}

func buildBackendRuntimeObjectPlan(
	target string,
	runtimeObjectOverride bool,
	checked *CheckedProgram,
) (backendRuntimeObjectPlan, error) {
	actorsUsed, _, _, err := collectActorEntries(checked)
	if err != nil {
		return backendRuntimeObjectPlan{}, err
	}
	actorStateUsed, _ := collectActorStateRuntimeUsagePosition(checked)
	actorRuntimeUsed, _ := collectActorRuntimeUsagePosition(checked)
	actorSystemReceiveUsed, _ := collectActorSystemReceiveRuntimeUsagePosition(checked)
	tasksUsed, _ := collectTaskRuntimeUsagePosition(checked)
	taskGroupsUsed := collectTaskGroupRuntimeUsage(checked)
	typedTasksUsed, _ := collectTypedTaskRuntimeUsage(checked)
	timeRuntimeUsed, _ := collectTimeRuntimeUsagePosition(checked)
	filesystemRuntimeUsed, _ := collectFilesystemRuntimeUsagePosition(checked)
	netRuntimeUsage := collectNetRuntimeUsageProfile(checked)
	surfaceRuntimeUsed, _ := collectSurfaceRuntimeUsagePosition(checked)
	distributedActorsUsed, _ := collectDistributedActorRuntimeUsagePosition(checked)

	usage := buildruntime.RuntimeObjectPlanUsage{
		ActorsUsed:             actorsUsed,
		ActorRuntimeUsed:       actorRuntimeUsed,
		ActorSystemReceiveUsed: actorSystemReceiveUsed,
		ActorStateUsed:         actorStateUsed,
		TasksUsed:              tasksUsed,
		TaskGroupsUsed:         taskGroupsUsed,
		TypedTasksUsed:         typedTasksUsed,
		TimeRuntimeUsed:        timeRuntimeUsed,
		FilesystemRuntimeUsed:  filesystemRuntimeUsed,
		NetRuntimeUsed:         netRuntimeUsage.used,
		NetRuntimeSupported:    targetSupportsNetRuntimeUsage(target, netRuntimeUsage),
		SurfaceRuntimeUsed:     surfaceRuntimeUsed,
		DistributedActorsUsed:  distributedActorsUsed,
	}
	decision := buildruntime.DecideRuntimeObjectPlan(
		target,
		runtimeObjectOverride,
		buildruntime.CapabilitiesForTarget(target),
		usage,
	)
	required := runtimeObjectFeaturesForUsage(usage)
	linked := []string{}
	initialized := []string{}
	if decision.RuntimeUsed {
		linked = append([]string(nil), required...)
		initialized = append([]string(nil), required...)
	}
	return backendRuntimeObjectPlan{
		EvidenceClass:                    runtimeObjectPlanEvidenceClass,
		EvidenceMethod:                   runtimeObjectPlanEvidenceMethod,
		RuntimeUsed:                      decision.RuntimeUsed,
		RuntimeObjectLinked:              decision.RuntimeUsed,
		RuntimeObjectInitialized:         decision.RuntimeUsed,
		RuntimeObjectOverride:            runtimeObjectOverride,
		TimeOnlyRuntime:                  decision.TimeOnlyRuntime,
		LinuxMinimalRuntime:              decision.LinuxMinimalRuntime,
		RuntimeObjectFeaturesRequired:    required,
		RuntimeObjectFeaturesLinked:      linked,
		RuntimeObjectFeaturesInitialized: initialized,
		RuntimeObjectLazyInitBlockers:    []string{},
	}, nil
}

func runtimeObjectFeaturesForUsage(usage buildruntime.RuntimeObjectPlanUsage) []string {
	features := map[string]struct{}{}
	if usage.ActorRuntimeUsed {
		features["actor_runtime"] = struct{}{}
	}
	if usage.ActorSystemReceiveUsed {
		features["actor_system_receive_runtime"] = struct{}{}
	}
	if usage.ActorStateUsed {
		features["actor_state_runtime"] = struct{}{}
	}
	if usage.TasksUsed {
		features["task_runtime"] = struct{}{}
	}
	if usage.TaskGroupsUsed {
		features["task_group_runtime"] = struct{}{}
	}
	if usage.TypedTasksUsed {
		features["typed_task_runtime"] = struct{}{}
	}
	if usage.TimeRuntimeUsed {
		features["time_runtime"] = struct{}{}
	}
	if usage.FilesystemRuntimeUsed {
		features["filesystem_runtime"] = struct{}{}
	}
	if usage.NetRuntimeUsed {
		features["net_runtime"] = struct{}{}
	}
	if usage.SurfaceRuntimeUsed {
		features["surface_runtime"] = struct{}{}
	}
	if usage.DistributedActorsUsed {
		features["distributed_actor_runtime"] = struct{}{}
	}
	out := make([]string, 0, len(features))
	for feature := range features {
		out = append(out, feature)
	}
	sort.Strings(out)
	return out
}
