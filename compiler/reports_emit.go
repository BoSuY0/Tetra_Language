package compiler

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"tetra_language/compiler/internal/allocplan"
	"tetra_language/compiler/internal/lower"
	"tetra_language/compiler/internal/memoryfacts"
	"tetra_language/compiler/internal/plir"
	"tetra_language/compiler/internal/ramcontract"
	"tetra_language/compiler/internal/semantics"
	"tetra_language/compiler/internal/validation"
)

func emitExplainReports(outputPath string, target string, checked *semantics.CheckedProgram, opt BuildOptions) error {
	if !opt.Explain && !opt.EmitPLIR && !opt.EmitProof && !opt.EmitBoundsReport && !opt.EmitAllocReport && !opt.EmitMemoryReport && !ramContractRequested(opt) {
		return nil
	}
	plirProg, err := plir.FromCheckedProgram(checked)
	if err != nil {
		return err
	}
	if err := plir.VerifyProgram(plirProg); err != nil {
		return err
	}
	allocPlan, err := allocplan.FromPLIRWithOptions(plirProg, allocationPlanOptionsForTarget(target))
	if err != nil {
		return err
	}
	irProg, err := lower.LowerWithOptions(checked, lowerOptionsForTarget(target))
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
		if err := writeReport(outputPath+".plir.json", plirProg); err != nil {
			return err
		}
		if err := os.WriteFile(outputPath+".plir.txt", []byte(plir.FormatText(plirProg)), 0o644); err != nil {
			return err
		}
	}
	if opt.Explain || opt.EmitProof {
		if err := writeReport(outputPath+".proof.json", buildProofReport(plirProg, bounds, target)); err != nil {
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
		if err := os.WriteFile(outputPath+".alloc.txt", []byte(allocplan.FormatText(allocPlan)), 0o644); err != nil {
			return err
		}
	}
	if opt.EmitMemoryReport {
		graph, err := memoryfacts.FromPLIRAndAllocPlan(target, plirProg, allocPlan)
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
		ramReport := ramcontract.BuildReportFromAllocPlan(allocPlan, target, reportGitHead(), "tetra-compiler")
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
		pipeline := ramcontract.BuildPipelineCoverage(target, ramReport.GitHead, "tetra-compiler", buildEntrypointForOptions(opt), outputPath, []string{
			"plir.VerifyProgram",
			"allocplan.VerifyPlan",
			"validation.ValidateAllocationLowering",
			"validation.CheckBoundsProofsWithPLIR",
			"ramcontract.ValidateReport",
		})
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
			if err := os.WriteFile(outputPath+".ram-contract.txt", []byte(formatRAMContractText(ramReport)), 0o644); err != nil {
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
		if err := os.WriteFile(outputPath+".explain.txt", []byte(formatExplainText(target, bounds, allocPlan, plirProg)), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func ramContractRequested(opt BuildOptions) bool {
	return opt.EmitRAMContractReport || ramContractEnforcementRequested(opt)
}

func ramContractEnforcementRequested(opt BuildOptions) bool {
	return opt.FailIfHeap || opt.FailIfCopy || opt.FailIfUnbounded || opt.MemoryBudgetBytes > 0 || strings.TrimSpace(opt.RAMContractFile) != ""
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
	fmt.Fprintf(&b, "rows: %d heap: %d copy: %d unbounded: %d budget_bytes: %d\n",
		report.Summary.RowCount, report.Summary.HeapRows, report.Summary.CopyRows, report.Summary.UnboundedRows, report.Summary.BudgetBytes)
	for _, row := range report.Rows {
		fmt.Fprintf(&b, "%s %s %s grade=%s placement=%s", row.Function, row.SiteID, row.Intent, row.ContractGrade, row.Placement)
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
