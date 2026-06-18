package compiler_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	compiler "tetra_language/compiler"
)

func TestTestRunnerSourcesIncludeCaseMetadata(t *testing.T) {
	src := []byte(`test "math case":
    expect 40 + 2 == 42
`)
	cases, err := compiler.TestRunnerSources(src, "math_test.tetra")
	if err != nil {
		t.Fatalf("TestRunnerSources: %v", err)
	}
	if len(cases) != 1 {
		t.Fatalf("cases = %d, want 1", len(cases))
	}
	c := cases[0]
	if c.Name != "math case" || c.Filename != "math_test.tetra" || c.Index != 0 {
		t.Fatalf("metadata = %#v", c)
	}
	if c.FunctionName != "__tetra_test_0_math_case" {
		t.Fatalf("function name = %q", c.FunctionName)
	}
	if !strings.Contains(
		string(c.Source),
		"func __tetra_test_0_math_case() -> Int\nuses actors, alloc, capability, control, islands, io, link, mem, mmio, runtime:",
	) {
		t.Fatalf("generated source missing test function:\n%s", string(c.Source))
	}
}

func TestTestRunnerSourcesStripTestDeclsForTempBuilds(t *testing.T) {
	src := []byte(`module app.tests

import lib.core.math as math

func fortyTwo() -> Int:
    return math.add_i32(40, 2)

test "module case":
    expect fortyTwo() == 42
`)
	cases, err := compiler.TestRunnerSources(src, "app/tests.tetra")
	if err != nil {
		t.Fatalf("TestRunnerSources: %v", err)
	}
	if len(cases) != 1 {
		t.Fatalf("cases = %d, want 1", len(cases))
	}
	generated := string(cases[0].Source)
	if strings.Contains(generated, `test "module case"`) {
		t.Fatalf("generated source should not include test blocks:\n%s", generated)
	}
	if !strings.Contains(generated, "module app.tests") {
		t.Fatalf("generated source should preserve module declaration for imports:\n%s", generated)
	}
	if !strings.Contains(generated, "func fortyTwo() -> Int:") {
		t.Fatalf("generated source missing helper function:\n%s", generated)
	}
	if !strings.Contains(generated, "import lib.core.math as math") {
		t.Fatalf("generated source missing required imports:\n%s", generated)
	}
}

func TestTestRunnerSourcesBuildModuleDeclaredCase(t *testing.T) {
	src := []byte(`module app.tests

func fortyTwo() -> Int:
    return 42

test "module case":
    expect fortyTwo() == 42
`)
	cases, err := compiler.TestRunnerSources(src, "app/tests.tetra")
	if err != nil {
		t.Fatalf("TestRunnerSources: %v", err)
	}
	tmp := t.TempDir()
	srcPath := filepath.Join(tmp, "app", "tests.tetra")
	if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
		t.Fatalf("mkdir generated source dir: %v", err)
	}
	outPath := filepath.Join(tmp, "test_1")
	if err := os.WriteFile(srcPath, cases[0].Source, 0o644); err != nil {
		t.Fatalf("write generated source: %v", err)
	}
	if err := compiler.BuildFile(srcPath, outPath, "linux-x64"); err != nil {
		t.Fatalf("BuildFile generated test source: %v\n%s", err, string(cases[0].Source))
	}
}

func TestTestRunnerSourcesDeclareEffectsForSyntheticFunctions(t *testing.T) {
	src := []byte(`test "io":
    print("hi\n")
    expect true
`)
	cases, err := compiler.TestRunnerSources(src, "io_test.tetra")
	if err != nil {
		t.Fatalf("TestRunnerSources: %v", err)
	}
	generated := string(cases[0].Source)
	if !strings.Contains(
		generated,
		"func __tetra_test_0_io() -> Int\nuses actors, alloc, capability, control, islands, io, link, mem, mmio, runtime:",
	) {
		t.Fatalf("generated test function missing conservative uses:\n%s", generated)
	}
	if !strings.Contains(
		generated,
		"func main() -> Int\nuses actors, alloc, capability, control, islands, io, link, mem, mmio, runtime:",
	) {
		t.Fatalf("generated main missing conservative uses:\n%s", generated)
	}
}

func TestTestRunnerSourcesStripUserMainBeforeGeneratingRunnerMain(t *testing.T) {
	src := []byte(`func helper() -> Int:
    return 41

func main() -> Int:
    return helper()

test "helper + 1":
    expect helper() + 1 == 42
`)
	cases, err := compiler.TestRunnerSources(src, "main_with_test.tetra")
	if err != nil {
		t.Fatalf("TestRunnerSources: %v", err)
	}
	if len(cases) != 1 {
		t.Fatalf("cases = %d, want 1", len(cases))
	}
	generated := string(cases[0].Source)
	if strings.Count(generated, "func main() -> Int") != 1 {
		t.Fatalf("generated source should contain exactly one main:\n%s", generated)
	}
	if !strings.Contains(generated, "func helper() -> Int:") {
		t.Fatalf("generated source missing helper function:\n%s", generated)
	}
}

func TestTestRunnerReportCountsPassFailMetadata(t *testing.T) {
	pass := compiler.TestRunnerSource{
		Name:         "passes",
		Filename:     "suite.tetra",
		Index:        0,
		FunctionName: "__tetra_test_0_passes",
	}.ResultWithDuration(0, nil, 12)
	fail := compiler.TestRunnerSource{
		Name:         "fails",
		Filename:     "suite.tetra",
		Index:        1,
		FunctionName: "__tetra_test_1_fails",
	}.ResultWithDuration(1, errors.New("exit status 1"), 30)

	report := compiler.NewTestRunnerReport([]compiler.TestRunnerResult{pass, fail})
	if report.Total != 2 || report.Passed != 1 || report.Failed != 1 {
		t.Fatalf(
			"report counts = total:%d passed:%d failed:%d",
			report.Total,
			report.Passed,
			report.Failed,
		)
	}
	if report.DurationMS != 42 {
		t.Fatalf("report duration = %d, want 42", report.DurationMS)
	}
	if len(report.Files) != 1 || report.Files[0].Filename != "suite.tetra" ||
		report.Files[0].Total != 2 ||
		report.Files[0].Passed != 1 ||
		report.Files[0].Failed != 1 ||
		report.Files[0].DurationMS != 42 {
		t.Fatalf("file report = %#v", report.Files)
	}
	if !report.Results[0].Passed || report.Results[0].ExitCode != 0 ||
		report.Results[0].DurationMS != 12 {
		t.Fatalf("pass result = %#v", report.Results[0])
	}
	if report.Results[1].Passed || report.Results[1].ExitCode != 1 ||
		report.Results[1].Error != "exit status 1" ||
		report.Results[1].DurationMS != 30 {
		t.Fatalf("fail result = %#v", report.Results[1])
	}
}

func TestTestRunnerResultReportsNonZeroExitWithoutRunError(t *testing.T) {
	fail := compiler.TestRunnerSource{
		Name:         "fails",
		Filename:     "suite.tetra",
		Index:        0,
		FunctionName: "__tetra_test_0_fails",
	}.ResultWithDuration(1, nil, 9)

	if fail.Passed || fail.ExitCode != 1 || fail.Error != "exit code 1" || fail.DurationMS != 9 {
		t.Fatalf("fail result = %#v", fail)
	}
}

func TestTestRunnerReportAggregatesFilesDeterministically(t *testing.T) {
	results := []compiler.TestRunnerResult{
		{Name: "b1", Filename: "b.tetra", Passed: true, DurationMS: 5},
		{Name: "a1", Filename: "a.tetra", Passed: false, DurationMS: 7},
		{Name: "a2", Filename: "a.tetra", Passed: true, DurationMS: 11},
	}
	report := compiler.NewTestRunnerReport(results)
	if len(report.Files) != 2 {
		t.Fatalf("files = %#v", report.Files)
	}
	if report.DurationMS != 23 {
		t.Fatalf("report duration = %d, want 23", report.DurationMS)
	}
	if report.Files[0].Filename != "a.tetra" || report.Files[0].Total != 2 ||
		report.Files[0].Passed != 1 ||
		report.Files[0].Failed != 1 ||
		report.Files[0].DurationMS != 18 {
		t.Fatalf("files[0] = %#v", report.Files[0])
	}
	if report.Files[1].Filename != "b.tetra" || report.Files[1].Total != 1 ||
		report.Files[1].Passed != 1 ||
		report.Files[1].Failed != 0 ||
		report.Files[1].DurationMS != 5 {
		t.Fatalf("files[1] = %#v", report.Files[1])
	}
}
