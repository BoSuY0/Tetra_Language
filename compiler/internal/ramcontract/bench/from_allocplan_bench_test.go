package bench

import (
	"fmt"
	"testing"

	rc "tetra_language/compiler/internal/ramcontract"
)

var benchmarkBlockerReportSink rc.BlockerReport

func BenchmarkBuildHeapBlockerReport(b *testing.B) {
	report := benchmarkRAMBlockerReport(4096, "heap")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkBlockerReportSink = rc.BuildHeapBlockerReport(report)
	}
}

func BenchmarkBuildCopyBlockerReport(b *testing.B) {
	report := benchmarkRAMBlockerReport(4096, "copy")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkBlockerReportSink = rc.BuildCopyBlockerReport(report)
	}
}

func benchmarkRAMBlockerReport(rows int, kind string) rc.Report {
	report := rc.Report{
		SchemaVersion: rc.ReportSchemaV1,
		GitHead:       "e2c19b8ee276158f8eb2c54cf61e11bd84952893",
		Target:        "linux-x64",
		GeneratedBy:   "benchmark",
		NonClaims:     rc.DefaultNonClaims(),
		Rows:          make([]rc.Row, 0, rows),
	}
	for i := 0; i < rows; i++ {
		row := rc.Row{
			SiteID:           fmt.Sprintf("site:bench:%04d", i),
			ValueID:          fmt.Sprintf("value:%04d", i),
			Function:         "bench",
			Intent:           rc.IntentHeapFallback,
			RequestedBytes:   64,
			Bounded:          false,
			Owner:            "function:bench",
			Lifetime:         "function:bench",
			EscapeStatus:     rc.EscapeUnknown,
			Placement:        rc.PlacementHeapUnbounded,
			ContractGrade:    rc.GradeM5,
			ValidationStatus: rc.ValidationConservative,
			SourceFactID:     fmt.Sprintf("fact:ram:bench:%04d", i),
		}
		if kind == "copy" {
			row.Intent = rc.IntentCopyHeapBounded
			row.Bounded = true
			row.EscapeStatus = rc.EscapeNoEscape
			row.Placement = rc.PlacementHeapBounded
			row.CopyReason = "copy_requires_bounded_heap_fallback"
			row.ContractGrade = rc.GradeM4
		}
		report.Rows = append(report.Rows, row)
	}
	report.Summary = rc.SummarizeRows(report.Rows)
	report.Functions = rc.SummarizeFunctions(report.Rows)
	return report
}
