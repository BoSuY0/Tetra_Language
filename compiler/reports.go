package compiler

import buildreports "tetra_language/compiler/internal/buildreports"

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
