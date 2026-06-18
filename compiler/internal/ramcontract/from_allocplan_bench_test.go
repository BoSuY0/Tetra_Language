package ramcontract

import (
	"fmt"
	"testing"
)

var benchmarkBlockerReportSink BlockerReport

func BenchmarkBuildHeapBlockerReport(b *testing.B) {
	report := benchmarkRAMBlockerReport(4096, "heap")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkBlockerReportSink = BuildHeapBlockerReport(report)
	}
}

func BenchmarkBuildCopyBlockerReport(b *testing.B) {
	report := benchmarkRAMBlockerReport(4096, "copy")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkBlockerReportSink = BuildCopyBlockerReport(report)
	}
}

func benchmarkRAMBlockerReport(rows int, kind string) Report {
	report := Report{
		SchemaVersion: ReportSchemaV1,
		GitHead:       "e2c19b8ee276158f8eb2c54cf61e11bd84952893",
		Target:        "linux-x64",
		GeneratedBy:   "benchmark",
		NonClaims:     DefaultNonClaims(),
		Rows:          make([]Row, 0, rows),
	}
	for i := 0; i < rows; i++ {
		row := Row{
			SiteID:           fmt.Sprintf("site:bench:%04d", i),
			ValueID:          fmt.Sprintf("value:%04d", i),
			Function:         "bench",
			Intent:           IntentHeapFallback,
			RequestedBytes:   64,
			Bounded:          false,
			Owner:            "function:bench",
			Lifetime:         "function:bench",
			EscapeStatus:     EscapeUnknown,
			Placement:        PlacementHeapUnbounded,
			ContractGrade:    GradeM5,
			ValidationStatus: ValidationConservative,
			SourceFactID:     fmt.Sprintf("fact:ram:bench:%04d", i),
		}
		if kind == "copy" {
			row.Intent = IntentCopyHeapBounded
			row.Bounded = true
			row.EscapeStatus = EscapeNoEscape
			row.Placement = PlacementHeapBounded
			row.CopyReason = "copy_requires_bounded_heap_fallback"
			row.ContractGrade = GradeM4
		}
		report.Rows = append(report.Rows, row)
	}
	report.Summary = SummarizeRows(report.Rows)
	report.Functions = SummarizeFunctions(report.Rows)
	return report
}
