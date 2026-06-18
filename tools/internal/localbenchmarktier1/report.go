package localbenchmarktier1

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
	b.WriteString(
		("Status: local measured evidence only. No fastest-language, official benchmark, " +
			"cross-machine, TechEmpower, or production claim is made.\n\n"),
	)
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

func writeAudit(path string, report tier1Report, outDir string) error {
	var b strings.Builder
	b.WriteString("# Local Benchmark Tier 1 V1 Audit\n\n")
	b.WriteString("Status: P25.0 local benchmark evidence artifact.\n\n")
	b.WriteString("This audit records a local-only execution of the P20 matrix.\n\n")
	b.WriteString("It does not claim:\n\n")
	b.WriteString("- Tetra is the fastest language;\n")
	b.WriteString("- an official benchmark result;\n")
	b.WriteString("- cross-machine reproduction;\n")
	b.WriteString("- TechEmpower publication;\n")
	b.WriteString("- production readiness.\n\n")
	b.WriteString("Primary artifact:\n")
	writeArtifactRecord(&b, outDir, "report.json")
	b.WriteString("Summary artifact:\n")
	writeArtifactRecord(&b, outDir, "summary.md")
	b.WriteString("## Classifications\n\n")
	for _, result := range report.Results {
		fmt.Fprintf(&b, "### %s\n\n", result.Category)
		fmt.Fprintf(&b, "- Classification: `%s`.\n", result.Classification)
		writeWrappedMarkdownLine(&b, "- Reason: ", "  ", result.ClassificationReason, 100)
		b.WriteString("\n")
	}
	b.WriteString("\n## Required Verification\n\n")
	b.WriteString("```bash\n")
	writeReportDirShell(&b, outDir)
	b.WriteString("go run ./tools/cmd/validate-local-benchmark-tier1 \\\n")
	b.WriteString("  --report \"$report_dir/report.json\"\n")
	b.WriteString("go run ./tools/cmd/verify-docs \\\n")
	b.WriteString("  --manifest docs/generated/manifest.json\n")
	b.WriteString("go run ./tools/cmd/validate-manifest \\\n")
	b.WriteString("  --manifest docs/generated/manifest.json\n")
	b.WriteString("git diff --check\n")
	b.WriteString("graphify update .\n")
	b.WriteString("go test ./compiler/... ./cli/... ./tools/... -count=1\n")
	b.WriteString("```\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeArtifactRecord(b *strings.Builder, dir string, file string) {
	root := filepath.Dir(dir)
	run := filepath.Base(dir)
	if root == "." || root == dir {
		fmt.Fprintf(b, "- Dir: `%s`\n", dir)
	} else {
		fmt.Fprintf(b, "- Root: `%s`\n", root)
		fmt.Fprintf(b, "- Run: `%s`\n", run)
	}
	fmt.Fprintf(b, "- File: `%s`\n\n", file)
}

func writeReportDirShell(b *strings.Builder, dir string) {
	root := filepath.Dir(dir)
	run := filepath.Base(dir)
	if root == "." || root == dir {
		fmt.Fprintf(b, "report_dir=%q\n", dir)
		return
	}
	fmt.Fprintf(b, "report_root=%q\n", root)
	fmt.Fprintf(b, "report_dir=\"$report_root/%s\"\n", run)
}

func writeWrappedMarkdownLine(
	b *strings.Builder,
	firstPrefix string,
	nextPrefix string,
	text string,
	maxWidth int,
) {
	words := strings.Fields(text)
	if len(words) == 0 {
		b.WriteString(firstPrefix + "\n")
		return
	}
	prefix := firstPrefix
	line := prefix
	for _, word := range words {
		if line == prefix {
			line += word
			continue
		}
		if len(line)+1+len(word) > maxWidth {
			b.WriteString(line + "\n")
			prefix = nextPrefix
			line = prefix + word
			continue
		}
		line += " " + word
	}
	b.WriteString(line + "\n")
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
	case "hash table",
		"allocation",
		"region/island allocation",
		"JSON parse/stringify",
		"HTTP plaintext/json",
		"PostgreSQL single/multiple/update":
		return true
	default:
		return false
	}
}

func boundsSensitiveCategory(category string) bool {
	switch category {
	case "slice sum",
		"bounds-check loops",
		"matrix multiply",
		"JSON parse/stringify",
		"HTTP plaintext/json",
		"PostgreSQL single/multiple/update":
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
	return "deterministic P25.0 local Tier 1 " + category + (" workload with identical " +
		"intent across Tetra, C, C++, and Rust")
}
