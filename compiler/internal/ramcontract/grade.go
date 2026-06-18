package ramcontract

import "sort"

func GradeForPlacement(placement Placement) MemoryGrade {
	switch placement {
	case PlacementEliminated:
		return GradeM0
	case PlacementRegister, PlacementStack:
		return GradeM1
	case PlacementStatic, PlacementInterned:
		return GradeM2
	case PlacementIsland, PlacementRegion:
		return GradeM3
	case PlacementHeapBounded:
		return GradeM4
	case PlacementHeapUnbounded:
		return GradeM5
	default:
		return GradeM6
	}
}

func MaxGrade(a, b MemoryGrade) MemoryGrade {
	if gradeRank(b) > gradeRank(a) {
		return b
	}
	return a
}

func SummarizeRows(rows []Row) Summary {
	summary := Summary{ArtifactGrade: GradeM0}
	domains := map[string]MemoryDomainSummary{}
	for _, row := range rows {
		summary.RowCount++
		summary.ArtifactGrade = MaxGrade(summary.ArtifactGrade, row.ContractGrade)
		if isHeapPlacement(row.Placement) {
			summary.HeapRows++
		}
		if isCopyIntent(row.Intent) {
			summary.CopyRows++
		}
		if row.Placement == PlacementHeapUnbounded || row.ContractGrade == GradeM5 ||
			row.ContractGrade == GradeM6 {
			summary.UnboundedRows++
		}
		if row.RequestedBytes > 0 {
			summary.BudgetBytes += row.RequestedBytes
		}
		if row.Domain != nil {
			key := memoryDomainSummaryKey(*row.Domain)
			domain := domains[key]
			if domain.DomainID == "" {
				domain.DomainID = row.Domain.DomainID
				domain.ParentDomainID = row.Domain.ParentDomainID
				domain.Kind = row.Domain.Kind
				domain.OwnerKind = row.Domain.OwnerKind
				domain.OwnerID = row.Domain.OwnerID
				domain.Lifetime = row.Domain.Lifetime
			}
			domain.RowCount++
			domain.BudgetBytes += row.Domain.BudgetBytes
			domain.RequestedBytes += row.Domain.RequestedBytes
			domain.ReservedBytes += row.Domain.ReservedBytes
			domain.CommittedBytes += row.Domain.CommittedBytes
			domain.ReleasedBytes += row.Domain.ReleasedBytes
			domain.CurrentBytes += row.Domain.CurrentBytes
			if row.Domain.PeakBytes > domain.PeakBytes {
				domain.PeakBytes = row.Domain.PeakBytes
			}
			domain.CopyCount += row.Domain.CopyCount
			domain.BytesCopied += row.Domain.BytesCopied
			domains[key] = domain
		}
	}
	if len(domains) > 0 {
		keys := make([]string, 0, len(domains))
		for key := range domains {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		summary.Domains = make([]MemoryDomainSummary, 0, len(keys))
		for _, key := range keys {
			summary.Domains = append(summary.Domains, domains[key])
		}
	}
	return summary
}

func memoryDomainSummaryKey(domain MemoryDomain) string {
	return string(
		domain.Kind,
	) + "\x00" + domain.DomainID + "\x00" + domain.ParentDomainID + "\x00" + domain.OwnerKind + "\x00" + domain.OwnerID + "\x00" + domain.Lifetime
}

func SummarizeFunctions(rows []Row) []FunctionRow {
	byName := map[string]FunctionRow{}
	for _, row := range rows {
		name := row.Function
		if name == "" {
			name = "<unknown>"
		}
		fn := byName[name]
		fn.Function = name
		if fn.Grade == "" {
			fn.Grade = GradeM0
		}
		fn.Grade = MaxGrade(fn.Grade, row.ContractGrade)
		fn.RowCount++
		if isHeapPlacement(row.Placement) {
			fn.HeapRows++
		}
		if isCopyIntent(row.Intent) {
			fn.CopyRows++
		}
		if row.RequestedBytes > 0 {
			fn.BudgetBytes += row.RequestedBytes
		}
		byName[name] = fn
	}
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]FunctionRow, 0, len(names))
	for _, name := range names {
		out = append(out, byName[name])
	}
	return out
}

func BuildGradeReport(report Report) GradeReport {
	return GradeReport{
		SchemaVersion: GradeReportSchemaV1,
		GitHead:       report.GitHead,
		Target:        report.Target,
		GeneratedBy:   report.GeneratedBy,
		ArtifactGrade: report.Summary.ArtifactGrade,
		Functions:     SummarizeFunctions(report.Rows),
		Summary:       report.Summary,
		NonClaims:     append([]string(nil), report.NonClaims...),
	}
}

func isCopyIntent(intent Intent) bool {
	switch intent {
	case IntentCopy,
		IntentCopyEliminated,
		IntentCopyStackBacked,
		IntentCopyHeapBounded,
		IntentCopyHeapUnbounded,
		IntentCopyRequiredBoundary,
		IntentCopyRequiredMutableAlias,
		IntentCopyIntoNoAllocation:
		return true
	default:
		return false
	}
}

func isHeapPlacement(placement Placement) bool {
	return placement == PlacementHeapBounded || placement == PlacementHeapUnbounded
}

func gradeRank(grade MemoryGrade) int {
	switch grade {
	case GradeM0:
		return 0
	case GradeM1:
		return 1
	case GradeM2:
		return 2
	case GradeM3:
		return 3
	case GradeM4:
		return 4
	case GradeM5:
		return 5
	case GradeM6:
		return 6
	default:
		return 7
	}
}
