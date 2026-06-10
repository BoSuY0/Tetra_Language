package ramcontract

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"tetra_language/compiler/internal/allocplan"
)

func BuildReportFromAllocPlan(plan *allocplan.Plan, target string, gitHead string, generatedBy string) Report {
	report := Report{
		SchemaVersion: ReportSchemaV1,
		GitHead:       defaultString(gitHead, "unknown"),
		Target:        target,
		GeneratedBy:   defaultString(generatedBy, "tetra-compiler"),
		GeneratedAt:   nowRFC3339(),
		NonClaims:     DefaultNonClaims(),
	}
	if plan == nil {
		report.Summary = SummarizeRows(nil)
		return report
	}
	proofs := map[string]ProofSummary{}
	for _, fn := range plan.Functions {
		for _, alloc := range fn.Allocations {
			row, proof := rowFromAllocation(fn.Name, alloc)
			if proof.ProofID != "" {
				proofs[proof.ProofID] = proof
			}
			report.Rows = append(report.Rows, row)
		}
	}
	if len(report.Rows) == 0 {
		return EmptyM0Report(target, gitHead, generatedBy)
	}
	sort.Slice(report.Rows, func(i, j int) bool {
		if report.Rows[i].Function != report.Rows[j].Function {
			return report.Rows[i].Function < report.Rows[j].Function
		}
		if report.Rows[i].SiteID != report.Rows[j].SiteID {
			return report.Rows[i].SiteID < report.Rows[j].SiteID
		}
		return report.Rows[i].ValueID < report.Rows[j].ValueID
	})
	for _, proof := range proofs {
		report.Proofs = append(report.Proofs, proof)
	}
	sort.Slice(report.Proofs, func(i, j int) bool { return report.Proofs[i].ProofID < report.Proofs[j].ProofID })
	report.Summary = SummarizeRows(report.Rows)
	report.Functions = SummarizeFunctions(report.Rows)
	return report
}

func rowFromAllocation(function string, alloc allocplan.Allocation) (Row, ProofSummary) {
	placement := placementFromAllocation(alloc)
	intent := intentFromAllocation(alloc, placement)
	row := Row{
		SiteID:           alloc.SiteID,
		ValueID:          alloc.ValueID,
		Function:         function,
		SourceSpan:       alloc.Source,
		Intent:           intent,
		RequestedBytes:   int64(allocationBytes(alloc)),
		Bounded:          allocationBounded(alloc, placement),
		Owner:            "function:" + function,
		Lifetime:         allocationLifetime(alloc, function),
		EscapeStatus:     escapeStatusFromAllocPlan(alloc.Escape),
		Placement:        placement,
		Blockers:         blockersFromAllocation(alloc, placement),
		CopyReason:       copyReasonFromAllocation(alloc, intent, placement),
		FreePoint:        freePointFromAllocation(alloc, function),
		ContractGrade:    GradeForPlacement(placement),
		ValidationStatus: validationStatusFromAllocation(alloc, placement),
		SourceFactID:     "fact:ram:" + sanitizeID(alloc.SiteID),
	}
	if trustedPlacement(placement) && row.ValidationStatus == ValidationValidated {
		proof := proofForAllocation(function, alloc, row)
		row.ProofIDs = []string{proof.ProofID}
		return row, proof
	}
	return row, ProofSummary{}
}

func placementFromAllocation(alloc allocplan.Allocation) Placement {
	switch alloc.ActualLoweringStorage {
	case allocplan.StorageEliminated:
		return PlacementEliminated
	case allocplan.StorageRegister:
		return PlacementRegister
	case allocplan.StorageStack:
		return PlacementStack
	case allocplan.StorageRegion, allocplan.StorageFunctionTempRegion, allocplan.StorageTaskRegion, allocplan.StorageActorMoveRegion:
		return PlacementRegion
	case allocplan.StorageExplicitIsland:
		return PlacementIsland
	case allocplan.StorageExternal:
		return PlacementExternal
	case allocplan.StorageHeap, allocplan.StorageLargeMmap:
		if allocationBounded(alloc, PlacementHeapBounded) {
			return PlacementHeapBounded
		}
		return PlacementHeapUnbounded
	default:
		return PlacementRejected
	}
}

func intentFromAllocation(alloc allocplan.Allocation, placement Placement) Intent {
	if isCopyBuiltin(alloc.Builtin) {
		switch placement {
		case PlacementEliminated:
			return IntentCopyEliminated
		case PlacementStack, PlacementRegister:
			return IntentCopyStackBacked
		case PlacementHeapBounded:
			return IntentCopyHeapBounded
		case PlacementHeapUnbounded:
			return IntentCopyHeapUnbounded
		default:
			return IntentCopy
		}
	}
	if isHeapPlacement(placement) {
		return IntentHeapFallback
	}
	if placement == PlacementRegion || placement == PlacementIsland {
		return IntentRegionAlloc
	}
	return IntentAllocation
}

func escapeStatusFromAllocPlan(escape allocplan.EscapeClass) EscapeStatus {
	switch escape {
	case allocplan.EscapeNoEscape:
		return EscapeNoEscape
	case allocplan.EscapeReturn:
		return EscapeReturn
	case allocplan.EscapeCallUnknown:
		return EscapeCall
	case allocplan.EscapeActor:
		return EscapeActorCrossing
	case allocplan.EscapeTask:
		return EscapeTaskCrossing
	case allocplan.EscapeUnsafe:
		return EscapeUnsafe
	default:
		return EscapeUnknown
	}
}

func validationStatusFromAllocation(alloc allocplan.Allocation, placement Placement) ValidationStatus {
	if strings.HasPrefix(alloc.ValidationStatus, "invalid") || placement == PlacementRejected {
		return ValidationRejected
	}
	if strings.HasPrefix(alloc.ValidationStatus, "validated") {
		if placement == PlacementHeapUnbounded || alloc.Escape == allocplan.EscapeUnknown {
			return ValidationConservative
		}
		return ValidationValidated
	}
	if placement == PlacementHeapUnbounded {
		return ValidationConservative
	}
	return ValidationUnknown
}

func blockersFromAllocation(alloc allocplan.Allocation, placement Placement) []string {
	blockers := map[string]bool{}
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			blockers[value] = true
		}
	}
	if isHeapPlacement(placement) || placement == PlacementRejected {
		switch alloc.Escape {
		case allocplan.EscapeReturn:
			add("escapes_return")
		case allocplan.EscapeCallUnknown:
			add("unknown_call")
		case allocplan.EscapeActor:
			add("actor_crossing")
		case allocplan.EscapeTask:
			add("task_crossing")
		case allocplan.EscapeUnsafe:
			add("unsafe_exposure")
		case allocplan.EscapeClosure:
			add("closure_escape")
		case allocplan.EscapeAggregate:
			add("aggregate_escape")
		case allocplan.EscapeUnknown:
			add("unknown_escape")
		}
		if allocationBytes(alloc) == 0 && alloc.LengthStatus != allocplan.LengthStatusValidEmpty {
			add("unknown_size")
		}
		if alloc.ActualLoweringStorage == allocplan.StorageHeap && alloc.Storage != allocplan.StorageHeap {
			add("backend_conservative_heap_fallback")
		}
		if alloc.Reason != "" {
			add(normalizeBlocker(alloc.Reason))
		}
	}
	out := make([]string, 0, len(blockers))
	for blocker := range blockers {
		out = append(out, blocker)
	}
	sort.Strings(out)
	return out
}

func proofForAllocation(function string, alloc allocplan.Allocation, row Row) ProofSummary {
	id := "proof:ram:" + sanitizeID(row.SiteID)
	subject := function + "/" + alloc.ValueID
	stable := stableProofHash(id, "allocation_placement", subject, alloc.ValidationStatus, string(row.Placement))
	return ProofSummary{
		ProofID:    id,
		Kind:       "allocation_placement",
		Subject:    subject,
		StableHash: stable,
		Status:     "proven",
	}
}

func stableProofHash(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func BuildProofStoreSummary(report Report) ProofStoreSummary {
	out := ProofStoreSummary{
		SchemaVersion: ProofStoreSummarySchemaV1,
		GitHead:       report.GitHead,
		Target:        report.Target,
		GeneratedBy:   report.GeneratedBy,
		Proofs:        append([]ProofSummary(nil), report.Proofs...),
		NonClaims:     append([]string(nil), report.NonClaims...),
	}
	out.Summary.ProofCount = len(out.Proofs)
	for _, proof := range out.Proofs {
		switch proof.Status {
		case "proven":
			out.Summary.Proven++
		case "conservative":
			out.Summary.Conservative++
		case "rejected":
			out.Summary.Rejected++
		default:
			out.Summary.Unknown++
		}
	}
	return out
}

func BuildPipelineCoverage(target string, gitHead string, generatedBy string, entrypoint string, artifactPath string, validators []string) PipelineCoverageReport {
	return PipelineCoverageReport{
		SchemaVersion: PipelineCoverageSchemaV1,
		GitHead:       defaultString(gitHead, "unknown"),
		Target:        target,
		GeneratedBy:   defaultString(generatedBy, "tetra-compiler"),
		Entries: []PipelineEntry{{
			Entrypoint:   entrypoint,
			ArtifactPath: artifactPath,
			Status:       "validated_by_pipeline",
			Validators:   append([]string(nil), validators...),
		}},
		NonClaims: DefaultNonClaims(),
	}
}

func BuildHeapBlockerReport(report Report) BlockerReport {
	out := BlockerReport{
		SchemaVersion: BlockerReportSchemaV1,
		Kind:          "heap",
		GitHead:       report.GitHead,
		Target:        report.Target,
		GeneratedBy:   report.GeneratedBy,
		NonClaims:     append([]string(nil), report.NonClaims...),
	}
	for _, row := range report.Rows {
		if !isHeapPlacement(row.Placement) {
			continue
		}
		out.Rows = append(out.Rows, blockerRowFromRAMRow(row))
	}
	return out
}

func BuildCopyBlockerReport(report Report) BlockerReport {
	out := BlockerReport{
		SchemaVersion: BlockerReportSchemaV1,
		Kind:          "copy",
		GitHead:       report.GitHead,
		Target:        report.Target,
		GeneratedBy:   report.GeneratedBy,
		NonClaims:     append([]string(nil), report.NonClaims...),
	}
	for _, row := range report.Rows {
		if !isCopyIntent(row.Intent) {
			continue
		}
		out.Rows = append(out.Rows, blockerRowFromRAMRow(row))
	}
	return out
}

func blockerRowFromRAMRow(row Row) BlockerRow {
	return BlockerRow{
		SiteID:        row.SiteID,
		Function:      row.Function,
		Intent:        row.Intent,
		Placement:     row.Placement,
		Blockers:      append([]string(nil), row.Blockers...),
		CopyReason:    row.CopyReason,
		ContractGrade: row.ContractGrade,
	}
}

func allocationBytes(alloc allocplan.Allocation) int {
	if alloc.BytesReserved > 0 {
		return alloc.BytesReserved
	}
	if alloc.BytesRequested > 0 {
		return alloc.BytesRequested
	}
	if alloc.ByteSize > 0 {
		return alloc.ByteSize
	}
	return 0
}

func allocationBounded(alloc allocplan.Allocation, placement Placement) bool {
	if placement == PlacementHeapUnbounded {
		return false
	}
	if alloc.LengthStatus == allocplan.LengthStatusRejectedNegative || alloc.LengthStatus == allocplan.LengthStatusRejectedOverflow {
		return true
	}
	return allocationBytes(alloc) > 0 || alloc.LengthStatus == allocplan.LengthStatusValidEmpty
}

func allocationLifetime(alloc allocplan.Allocation, function string) string {
	if alloc.Lifetime != "" {
		return alloc.Lifetime
	}
	if alloc.RegionID != "" {
		return "region:" + alloc.RegionID
	}
	return "function:" + function
}

func freePointFromAllocation(alloc allocplan.Allocation, function string) string {
	switch alloc.ActualLoweringStorage {
	case allocplan.StorageRegion, allocplan.StorageFunctionTempRegion, allocplan.StorageExplicitIsland,
		allocplan.StorageTaskRegion, allocplan.StorageActorMoveRegion:
		if alloc.Lifetime != "" {
			return alloc.Lifetime + ":end"
		}
		return "function:" + function + ":end"
	case allocplan.StorageHeap:
		return "runtime_allocator"
	default:
		return "not_applicable"
	}
}

func copyReasonFromAllocation(alloc allocplan.Allocation, intent Intent, placement Placement) string {
	if !isCopyIntent(intent) {
		return ""
	}
	switch intent {
	case IntentCopyEliminated:
		return "copy_eliminated_by_supported_no-use_scan"
	case IntentCopyStackBacked:
		return "copy_result_stack_backed_by_no_escape_proof"
	case IntentCopyHeapBounded:
		return "copy_requires_bounded_heap_fallback"
	case IntentCopyHeapUnbounded:
		return "copy_requires_unbounded_heap_fallback"
	default:
		if isHeapPlacement(placement) {
			return "copy_requires_conservative_heap_fallback"
		}
		return "copy_necessity_recorded"
	}
}

func isCopyBuiltin(name string) bool {
	return name == "core.string_copy" || (strings.HasPrefix(name, "core.slice_copy_") && !strings.HasPrefix(name, "core.slice_copy_into_"))
}

func normalizeBlocker(reason string) string {
	reason = strings.ToLower(strings.TrimSpace(reason))
	replacer := strings.NewReplacer(" ", "_", ";", "", ":", "", ",", "", ".", "", "/", "_", "-", "_")
	reason = replacer.Replace(reason)
	if len(reason) > 96 {
		reason = reason[:96]
	}
	if reason == "" {
		return "conservative_fallback"
	}
	return reason
}

func sanitizeID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer(" ", "_", "/", "_", "\\", "_")
	return replacer.Replace(id)
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func EmptyM0Report(target string, gitHead string, generatedBy string) Report {
	row := Row{
		SiteID:           "site:artifact:no-allocation",
		ValueID:          "artifact:no-allocation",
		Function:         "<artifact>",
		Intent:           IntentAllocation,
		Bounded:          true,
		Owner:            "artifact",
		Lifetime:         "artifact",
		EscapeStatus:     EscapeNoEscape,
		Placement:        PlacementEliminated,
		ProofIDs:         nil,
		ContractGrade:    GradeM0,
		ValidationStatus: ValidationConservative,
		SourceFactID:     "fact:ram:artifact:no-allocation",
	}
	report := Report{
		SchemaVersion: ReportSchemaV1,
		GitHead:       defaultString(gitHead, "unknown"),
		Target:        target,
		GeneratedBy:   defaultString(generatedBy, "tetra-compiler"),
		GeneratedAt:   nowRFC3339(),
		Rows:          []Row{row},
		NonClaims:     DefaultNonClaims(),
	}
	report.Summary = SummarizeRows(report.Rows)
	report.Functions = SummarizeFunctions(report.Rows)
	return report
}

func DebugString(report Report) string {
	return fmt.Sprintf("RAM contract %s target=%s grade=%s rows=%d heap=%d copy=%d",
		report.SchemaVersion, report.Target, report.Summary.ArtifactGrade, report.Summary.RowCount, report.Summary.HeapRows, report.Summary.CopyRows)
}
