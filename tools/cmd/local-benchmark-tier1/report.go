package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func writeJSON(path string, data any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func writeSummary(path string, report tier1Report) error {
	var b strings.Builder
	b.WriteString("# Local Benchmark Tier 1 V1\n\n")
	b.WriteString("Status: local measured evidence only. No fastest-language, official benchmark, cross-machine, TechEmpower, or production claim is made.\n\n")
	b.WriteString("| Category | Classification | Primary metric | Tetra | C | C++ | Rust |\n")
	b.WriteString("| --- | --- | --- | ---: | ---: | ---: | ---: |\n")
	for _, result := range report.Results {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s |\n",
			result.Category,
			result.Classification,
			primaryMetricName(result.Category),
			formatPrimaryMetric(result.Category, result.Rows, "tetra"),
			formatPrimaryMetric(result.Category, result.Rows, "c"),
			formatPrimaryMetric(result.Category, result.Rows, "cpp"),
			formatPrimaryMetric(result.Category, result.Rows, "rust"),
		)
	}
	b.WriteString("\n## Non-Claims\n\n")
	for _, claim := range report.NonClaims {
		fmt.Fprintf(&b, "- %s\n", claim)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeAudit(path string, report tier1Report) error {
	var b strings.Builder
	b.WriteString("# Local Benchmark Tier 1 V1 Audit\n\n")
	b.WriteString("Status: P25.0 local benchmark evidence artifact.\n\n")
	b.WriteString("This audit records a local-only execution of the P20 matrix. It does not claim Tetra is the fastest language, does not claim an official benchmark result, does not claim cross-machine reproduction, does not claim TechEmpower publication, and does not claim production readiness.\n\n")
	b.WriteString("Primary artifact: `reports/local-benchmark-tier1-v1/report.json`.\n\n")
	b.WriteString("Summary artifact: `reports/local-benchmark-tier1-v1/summary.md`.\n\n")
	b.WriteString("## Classifications\n\n")
	for _, result := range report.Results {
		fmt.Fprintf(&b, "- `%s`: `%s` — %s\n", result.Category, result.Classification, result.ClassificationReason)
	}
	b.WriteString("\n## Required Verification\n\n")
	b.WriteString("- `go run ./tools/cmd/validate-local-benchmark-tier1 --report reports/local-benchmark-tier1-v1/report.json`\n")
	b.WriteString("- `go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json`\n")
	b.WriteString("- `go run ./tools/cmd/validate-manifest --manifest docs/generated/manifest.json`\n")
	b.WriteString("- `git diff --check`\n")
	b.WriteString("- `graphify update .`\n")
	b.WriteString("- `go test ./compiler/... ./cli/... ./tools/... -count=1`\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func formatMedian(rows []benchmarkRow, language string) string {
	row, ok := rowForLanguage(rows, language)
	if !ok || row.MedianRuntimeMS <= 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.3f", row.MedianRuntimeMS)
}

func primaryMetricName(category string) string {
	switch category {
	case "binary size":
		return "binary_size_bytes"
	case "compile time":
		return "compile_time_ms"
	default:
		return "median_runtime_ms"
	}
}

func formatPrimaryMetric(category string, rows []benchmarkRow, language string) string {
	row, ok := rowForLanguage(rows, language)
	if !ok {
		return "n/a"
	}
	switch category {
	case "binary size":
		if row.BinarySizeBytes <= 0 {
			return "n/a"
		}
		return fmt.Sprintf("%d", row.BinarySizeBytes)
	case "compile time":
		if row.CompileTimeMS <= 0 {
			return "n/a"
		}
		return fmt.Sprintf("%.3f", row.CompileTimeMS)
	default:
		if row.MedianRuntimeMS <= 0 {
			return "n/a"
		}
		return fmt.Sprintf("%.3f", row.MedianRuntimeMS)
	}
}

func rowForLanguage(rows []benchmarkRow, language string) (benchmarkRow, bool) {
	for _, row := range rows {
		if row.Language == language {
			return row, true
		}
	}
	return benchmarkRow{}, false
}

func measuredCompetitorMedians(rows []benchmarkRow) []float64 {
	var out []float64
	for _, language := range []string{"c", "cpp", "rust"} {
		row, ok := rowForLanguage(rows, language)
		if ok && row.Status == "measured" && row.MedianRuntimeMS > 0 {
			out = append(out, row.MedianRuntimeMS)
		}
	}
	return out
}

func heapSensitiveCategory(category string) bool {
	switch category {
	case "hash table", "allocation", "region/island allocation", "JSON parse/stringify", "HTTP plaintext/json", "PostgreSQL single/multiple/update":
		return true
	default:
		return false
	}
}

func boundsSensitiveCategory(category string) bool {
	switch category {
	case "slice sum", "bounds-check loops", "matrix multiply", "JSON parse/stringify", "HTTP plaintext/json", "PostgreSQL single/multiple/update":
		return true
	default:
		return false
	}
}

func languageOrder(language string) int {
	for i, supported := range requiredLanguages {
		if language == supported {
			return i
		}
	}
	return len(requiredLanguages)
}

func extensionFor(language string) string {
	switch language {
	case "tetra":
		return ".tetra"
	case "c":
		return ".c"
	case "cpp":
		return ".cpp"
	case "rust":
		return ".rs"
	default:
		return ".txt"
	}
}

func ensureRawRunArtifacts(stdoutPath string, stderrPath string, message string) {
	_ = os.WriteFile(stdoutPath, []byte(message), 0o644)
	_ = os.WriteFile(stderrPath, nil, 0o644)
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

func millis(duration time.Duration) float64 {
	return math.Round(duration.Seconds()*1000000) / 1000
}

func slug(value string) string {
	replacer := strings.NewReplacer("/", "_", "-", "_")
	return strings.Join(strings.Fields(replacer.Replace(strings.ToLower(value))), "_")
}

func inputDescription(category string) string {
	return "deterministic P25.0 local Tier 1 " + category + " workload with identical intent across Tetra, C, C++, and Rust"
}
