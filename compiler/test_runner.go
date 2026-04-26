package compiler

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

type TestRunnerSource struct {
	Name         string
	Filename     string
	Index        int
	FunctionName string
	Source       []byte
}

type TestRunnerResult struct {
	Name         string `json:"name"`
	Filename     string `json:"filename"`
	Index        int    `json:"index"`
	FunctionName string `json:"function_name"`
	ExitCode     int    `json:"exit_code"`
	Passed       bool   `json:"passed"`
	DurationMS   int64  `json:"duration_ms"`
	Error        string `json:"error,omitempty"`
}

type TestRunnerFileReport struct {
	Filename   string `json:"filename"`
	Total      int    `json:"total"`
	Passed     int    `json:"passed"`
	Failed     int    `json:"failed"`
	DurationMS int64  `json:"duration_ms"`
}

type TestRunnerReport struct {
	Total      int                    `json:"total"`
	Passed     int                    `json:"passed"`
	Failed     int                    `json:"failed"`
	DurationMS int64                  `json:"duration_ms"`
	Files      []TestRunnerFileReport `json:"files"`
	Results    []TestRunnerResult     `json:"results"`
}

func TestRunnerSources(src []byte, filename string) ([]TestRunnerSource, error) {
	file, err := frontend.ParseFile(src, filename)
	if err != nil {
		return nil, err
	}
	baseSrc := testRunnerBaseSource(file)
	out := make([]TestRunnerSource, 0, len(file.Tests))
	for i, test := range file.Tests {
		var b strings.Builder
		if baseSrc != "" {
			b.WriteString(baseSrc)
			b.WriteString("\n\n")
		}
		fnName := fmt.Sprintf("__tetra_test_%d_%s", i, sanitizeTestName(test.Name))
		b.WriteString("\nfunc ")
		b.WriteString(fnName)
		b.WriteString("() -> Int\n")
		b.WriteString(testRunnerUsesClause())
		b.WriteString(":\n")
		for _, stmt := range test.Body {
			writeTestStmt(&b, stmt, 1)
		}
		b.WriteString("    return 0\n\n")
		b.WriteString("func main() -> Int\n")
		b.WriteString(testRunnerUsesClause())
		b.WriteString(":\n")
		b.WriteString("    return ")
		b.WriteString(fnName)
		b.WriteString("()\n")
		out = append(out, TestRunnerSource{
			Name:         test.Name,
			Filename:     filename,
			Index:        i,
			FunctionName: fnName,
			Source:       []byte(b.String()),
		})
	}
	return out, nil
}

func testRunnerBaseSource(file *frontend.FileAST) string {
	base := *file
	base.Module = ""
	base.Imports = nil
	base.Tests = nil
	var p sourcePrinter
	p.file(&base)
	return strings.TrimSpace(p.b.String())
}

func testRunnerUsesClause() string {
	return "uses actors, alloc, capability, control, islands, io, link, mem, mmio, runtime"
}

func (s TestRunnerSource) Result(exitCode int, runErr error) TestRunnerResult {
	return s.ResultWithDuration(exitCode, runErr, 0)
}

func (s TestRunnerSource) ResultWithDuration(exitCode int, runErr error, durationMS int64) TestRunnerResult {
	result := TestRunnerResult{
		Name:         s.Name,
		Filename:     s.Filename,
		Index:        s.Index,
		FunctionName: s.FunctionName,
		ExitCode:     exitCode,
		Passed:       exitCode == 0 && runErr == nil,
		DurationMS:   durationMS,
	}
	if runErr != nil {
		result.Error = runErr.Error()
	} else if exitCode != 0 {
		result.Error = fmt.Sprintf("exit code %d", exitCode)
	}
	return result
}

func NewTestRunnerReport(results []TestRunnerResult) TestRunnerReport {
	report := TestRunnerReport{
		Total:   len(results),
		Files:   []TestRunnerFileReport{},
		Results: append([]TestRunnerResult{}, results...),
	}
	byFile := map[string]*TestRunnerFileReport{}
	for _, result := range results {
		if result.Passed {
			report.Passed++
		} else {
			report.Failed++
		}
		report.DurationMS += result.DurationMS
		file := byFile[result.Filename]
		if file == nil {
			file = &TestRunnerFileReport{Filename: result.Filename}
			byFile[result.Filename] = file
		}
		file.Total++
		file.DurationMS += result.DurationMS
		if result.Passed {
			file.Passed++
		} else {
			file.Failed++
		}
	}
	filenames := make([]string, 0, len(byFile))
	for filename := range byFile {
		filenames = append(filenames, filename)
	}
	sort.Strings(filenames)
	for _, filename := range filenames {
		report.Files = append(report.Files, *byFile[filename])
	}
	return report
}

func writeTestStmt(b *strings.Builder, stmt frontend.Stmt, indent int) {
	prefix := strings.Repeat(" ", indent*4)
	switch s := stmt.(type) {
	case *frontend.ExpectStmt:
		b.WriteString(prefix)
		b.WriteString("if ")
		b.WriteString(formatTestExpr(s.Cond))
		b.WriteString(":\n")
		b.WriteString(prefix)
		b.WriteString("    let __ok: Int = 0\n")
		b.WriteString(prefix)
		b.WriteString("else:\n")
		b.WriteString(prefix)
		b.WriteString("    return 1\n")
	default:
		var p sourcePrinter
		p.stmt(stmt, indent)
		b.WriteString(p.b.String())
	}
}

func formatTestExpr(expr frontend.Expr) string {
	var p sourcePrinter
	return p.formatExpr(expr)
}

var nonTestNameChar = regexp.MustCompile(`[^A-Za-z0-9_]+`)

func sanitizeTestName(name string) string {
	clean := nonTestNameChar.ReplaceAllString(name, "_")
	clean = strings.Trim(clean, "_")
	if clean == "" {
		return "case"
	}
	if clean[0] >= '0' && clean[0] <= '9' {
		return "case_" + clean
	}
	return clean
}
