package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"tetra_language/compiler"
	ctarget "tetra_language/compiler/target"
)

func runTest(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", defaultTarget(), "target triple ("+supportedTargetsHelp+")")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text or json")
	reportFormat := fs.String("report", "text", "report format: text or json")
	format := fs.String("format", "", "output format alias for --report: text or json")
	allTargets := fs.Bool("all-targets", false, "run the required x86/x64/x32 target matrix")
	brutal := fs.Bool("brutal", false, "run the full brutal target matrix")
	abiSuite := fs.Bool("abi", false, "run ABI torture tests for the target")
	atomicStress := fs.Bool("atomic-stress", false, "run atomic stress tests for the target")
	fuzzSuite := fs.Bool("fuzz", false, "run fuzz/property tests for the target")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	reportFormatValue, ok := resolveTestReportFormat(fs, *reportFormat, *format, *diagnostics, stderr)
	if !ok {
		return 2
	}
	if reportFormatValue != "text" && reportFormatValue != "json" {
		writeValidationDiagnostic(stderr, *diagnostics, "unsupported --report format")
		return 2
	}
	if *allTargets || *brutal {
		return runAllTargetsSuite(*allTargets, *brutal, *diagnostics, reportFormatValue, stdout, stderr)
	}
	tgt, ok := parseBuildTargetOrReport(*target, *diagnostics, stderr)
	if !ok {
		return 2
	}
	if *abiSuite && !*atomicStress && !*fuzzSuite {
		return runTargetABISuite(*target, *diagnostics, reportFormatValue, stdout, stderr)
	}
	if *atomicStress && !*abiSuite && !*fuzzSuite {
		return runTargetAtomicStressSuite(*target, *diagnostics, reportFormatValue, stdout, stderr)
	}
	if *fuzzSuite && !*abiSuite && !*atomicStress {
		return runTargetFuzzSuite(*target, *diagnostics, reportFormatValue, stdout, stderr)
	}
	if *abiSuite || *atomicStress || *fuzzSuite {
		writeDiagnostic(stderr, *diagnostics, unsupportedTargetTestSuiteDiagnostic(tgt.Triple, *abiSuite, *atomicStress, *fuzzSuite))
		return 2
	}
	paths := fs.Args()
	explicitPaths := len(paths) > 0
	explicitTarget := testArgsIncludeTargetFlag(args)
	explicitSingleFileInput := false
	if len(paths) == 1 {
		if info, err := os.Stat(paths[0]); err == nil && !info.IsDir() {
			explicitSingleFileInput = true
		}
	}
	var projectCtx *cliProjectContext
	var worldOpt compiler.WorldOptions
	if len(paths) == 0 {
		ctx, err := discoverCLIProject(".")
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		if ctx != nil && ctx.Found {
			projectCtx = ctx
			worldOpt = compiler.WorldOptions{
				Root:            ctx.Root,
				SourceRoots:     append([]string(nil), ctx.SourceRoots...),
				DependencyRoots: append([]compiler.ModuleRoot(nil), ctx.DependencyRoots...),
			}
			paths = existingProjectSourcePaths(ctx)
		}
		if len(paths) == 0 {
			if explicitTarget && isRequiredTargetSuiteTriple(tgt.Triple) {
				return runTargetDefaultSuite(tgt, reportFormatValue, stdout, stderr)
			}
			paths = []string{"."}
		}
	} else if len(paths) == 1 {
		resolved, resolvedWorldOpt, ctx, err := resolveCLIInput(paths[0])
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		if ctx != nil && ctx.Found && isProjectReference(paths[0], ctx) {
			projectCtx = ctx
			worldOpt = resolvedWorldOpt
			paths = existingProjectSourcePaths(ctx)
			if len(paths) == 0 {
				paths = []string{resolved}
			}
		}
	}
	if isWASMTargetTriple(tgt.Triple) {
		writeTargetRuntimeDiagnostic(stderr, *diagnostics, fmt.Sprintf("cannot run tests for target %s: WASM test runner is not part of the current production runtime contract; use smoke/runtime reports for WASM execution evidence", tgt.Triple))
		return 2
	}
	if ctarget.IsBuildOnlyTarget(tgt.Triple) && !canRunBuildOnlyNativeTargetOnHost(tgt) {
		reason := buildOnlyNativeRunUnsupportedReason(tgt)
		writeTargetRuntimeDiagnostic(stderr, *diagnostics, fmt.Sprintf("cannot run tests for target %s: %s", tgt.Triple, reason))
		return 2
	}
	if !canRunNativeExecutableTargetOnHost(tgt) {
		writeTargetRuntimeDiagnostic(stderr, *diagnostics, fmt.Sprintf("cannot run tests for target %s on host %s/%s", tgt.Triple, runtime.GOOS, runtime.GOARCH))
		return 2
	}
	if !explicitPaths && explicitTarget && projectCtx == nil && isRequiredTargetSuiteTriple(tgt.Triple) {
		return runTargetDefaultSuite(tgt, reportFormatValue, stdout, stderr)
	}
	if err := validateDiscoveredProjectLock(projectCtx, tgt.Triple); err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	targetLinkObjects, err := projectLinkObjects(projectCtx, tgt.Triple, nil)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	files, err := collectTetraFiles(paths)
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	tmpDir, err := os.MkdirTemp("", "tetra-test-*")
	if err != nil {
		writeDiagnostic(stderr, *diagnostics, err)
		return 1
	}
	defer os.RemoveAll(tmpDir)
	total := 0
	passed := 0
	var results []compiler.TestRunnerResult
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		runners, err := compiler.TestRunnerSources(raw, file)
		if err != nil {
			writeDiagnostic(stderr, *diagnostics, err)
			return 1
		}
		for i, runner := range runners {
			total++
			start := time.Now()
			srcPath := filepath.Join(tmpDir, fmt.Sprintf("test_%d.t4", total))
			runnerSource := runner.Source
			sourceModule := modulePathFromSource(runner.Source)
			if sourceModule != "" {
				var err error
				srcPath, runnerSource, err = runnerSourcePathForModuleFile(file, runner.Source, total)
				if err != nil {
					writeDiagnostic(stderr, *diagnostics, err)
					return 1
				}
				defer os.Remove(srcPath)
			}
			outPath := filepath.Join(tmpDir, fmt.Sprintf("test_%d%s", total, tgt.ExeExt))
			if err := os.MkdirAll(filepath.Dir(srcPath), 0o755); err != nil {
				writeDiagnostic(stderr, *diagnostics, err)
				return 1
			}
			if err := os.WriteFile(srcPath, runnerSource, 0o644); err != nil {
				writeDiagnostic(stderr, *diagnostics, err)
				return 1
			}
			if modulePathFromSource(runnerSource) != "" && projectCtx != nil && projectCtx.Found {
				runnerProjectCtx := projectCtx
				runnerWorldOpt := worldOpt
				runnerLinkObjects := targetLinkObjects
				if runnerProjectCtx == nil {
					var err error
					runnerProjectCtx, runnerWorldOpt, runnerLinkObjects, err = testProjectContextForFile(file, sourceModule, tgt.Triple)
					if err != nil {
						writeDiagnostic(stderr, *diagnostics, err)
						return 1
					}
				}
				if runnerProjectCtx != nil && runnerProjectCtx.Found {
					opt := compiler.BuildOptions{
						Jobs:            1,
						ProjectRoot:     runnerWorldOpt.Root,
						SourceRoots:     append([]string(nil), runnerWorldOpt.SourceRoots...),
						DependencyRoots: append([]compiler.ModuleRoot(nil), runnerWorldOpt.DependencyRoots...),
						LinkObjectPaths: append([]string(nil), runnerLinkObjects...),
					}
					if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, opt); err != nil {
						writeDiagnostic(stderr, *diagnostics, err)
						return 1
					}
				} else if err := compiler.BuildFile(srcPath, outPath, tgt.Triple); err != nil {
					writeDiagnostic(stderr, *diagnostics, err)
					return 1
				}
			} else if modulePathFromSource(runnerSource) != "" {
				var runnerProjectCtx *cliProjectContext
				var runnerWorldOpt compiler.WorldOptions
				var runnerLinkObjects []string
				if !explicitSingleFileInput {
					var err error
					runnerProjectCtx, runnerWorldOpt, runnerLinkObjects, err = testProjectContextForFile(file, sourceModule, tgt.Triple)
					if err != nil {
						writeDiagnostic(stderr, *diagnostics, err)
						return 1
					}
				}
				if runnerProjectCtx != nil && runnerProjectCtx.Found {
					opt := compiler.BuildOptions{
						Jobs:            1,
						ProjectRoot:     runnerWorldOpt.Root,
						SourceRoots:     append([]string(nil), runnerWorldOpt.SourceRoots...),
						DependencyRoots: append([]compiler.ModuleRoot(nil), runnerWorldOpt.DependencyRoots...),
						LinkObjectPaths: append([]string(nil), runnerLinkObjects...),
					}
					if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, opt); err != nil {
						writeDiagnostic(stderr, *diagnostics, err)
						return 1
					}
				} else if err := compiler.BuildFile(srcPath, outPath, tgt.Triple); err != nil {
					writeDiagnostic(stderr, *diagnostics, err)
					return 1
				}
			} else {
				if err := compiler.BuildFile(srcPath, outPath, tgt.Triple); err != nil {
					writeDiagnostic(stderr, *diagnostics, err)
					return 1
				}
			}
			code := execNativeProgram(outPath, io.Discard, io.Discard)
			name := runner.Name
			if name == "" {
				name = fmt.Sprintf("%s#%d", file, i+1)
			}
			result := runner.ResultWithDuration(code, nil, elapsedMillis(time.Since(start)))
			results = append(results, result)
			if code == 0 {
				passed++
				if reportFormatValue == "text" {
					fmt.Fprintf(stdout, "PASS %s\n", name)
				}
			} else {
				if reportFormatValue == "text" {
					if result.Error != "" {
						fmt.Fprintf(stdout, "FAIL %s (%s)\n", name, result.Error)
					} else {
						fmt.Fprintf(stdout, "FAIL %s\n", name)
					}
				}
			}
		}
	}
	if reportFormatValue == "json" {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(compiler.NewTestRunnerReportForTarget(results, tgt.Triple)); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	} else {
		fmt.Fprintf(stdout, "Tetra tests: %d/%d passed\n", passed, total)
	}
	if passed != total {
		return 1
	}
	return 0
}

func resolveTestReportFormat(fs *flag.FlagSet, reportValue string, formatValue string, diagnostics string, stderr io.Writer) (string, bool) {
	reportProvided := false
	formatProvided := false
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "report":
			reportProvided = true
		case "format":
			formatProvided = true
		}
	})
	if !formatProvided {
		return reportValue, true
	}
	if formatValue != "text" && formatValue != "json" {
		writeValidationDiagnostic(stderr, diagnostics, "unsupported --format")
		return "", false
	}
	if reportProvided && reportValue != formatValue {
		writeValidationDiagnostic(stderr, diagnostics, "--format and --report must match when both are provided")
		return "", false
	}
	return formatValue, true
}

func testProjectContextForFile(file string, module string, target string) (*cliProjectContext, compiler.WorldOptions, []string, error) {
	ctx, err := discoverCLIProject(filepath.Dir(file))
	if err != nil {
		return nil, compiler.WorldOptions{}, nil, err
	}
	if ctx == nil || !ctx.Found || !fileModuleMatchesProjectSourceRoots(file, module, ctx) {
		return nil, compiler.WorldOptions{}, nil, nil
	}
	if err := validateDiscoveredProjectLock(ctx, target); err != nil {
		return nil, compiler.WorldOptions{}, nil, err
	}
	linkObjects, err := projectLinkObjects(ctx, target, nil)
	if err != nil {
		return nil, compiler.WorldOptions{}, nil, err
	}
	opt := compiler.WorldOptions{
		Root:            ctx.Root,
		SourceRoots:     append([]string(nil), ctx.SourceRoots...),
		DependencyRoots: append([]compiler.ModuleRoot(nil), ctx.DependencyRoots...),
	}
	return ctx, opt, linkObjects, nil
}

func fileModuleMatchesProjectSourceRoots(file string, module string, ctx *cliProjectContext) bool {
	if module == "" || ctx == nil || !ctx.Found {
		return false
	}
	abs, err := filepath.Abs(file)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(ctx.Root, abs)
	if err != nil {
		return false
	}
	cleanRel := filepath.Clean(rel)
	if cleanRel == "." || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) || filepath.IsAbs(cleanRel) {
		return false
	}
	for _, root := range ctx.SourceRoots {
		cleanRoot := filepath.Clean(filepath.FromSlash(root))
		moduleRel := cleanRel
		if root != "" && cleanRoot != "." {
			if cleanRel == cleanRoot || !strings.HasPrefix(cleanRel, cleanRoot+string(filepath.Separator)) {
				continue
			}
			moduleRel = strings.TrimPrefix(cleanRel, cleanRoot+string(filepath.Separator))
		}
		if cliModuleRelPathMatches(module, moduleRel) {
			return true
		}
	}
	return false
}

func testArgsIncludeTargetFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		if arg == "--target" || arg == "-target" || strings.HasPrefix(arg, "--target=") || strings.HasPrefix(arg, "-target=") {
			return true
		}
	}
	return false
}

func isRequiredTargetSuiteTriple(triple string) bool {
	switch triple {
	case "linux-x86", "linux-x64", "linux-x32":
		return true
	default:
		return false
	}
}

func runTargetDefaultSuite(tgt ctarget.Target, reportFormat string, stdout io.Writer, stderr io.Writer) int {
	return writeTargetSuiteReport(runABISuiteResults(tgt), tgt.Triple, reportFormat, stdout, stderr)
}

func unsupportedTargetTestSuiteDiagnostic(target string, abiSuite bool, atomicStress bool, fuzzSuite bool) error {
	var suites []string
	var flags []string
	if abiSuite {
		suites = append(suites, "ABI torture")
		flags = append(flags, "--abi")
	}
	if atomicStress {
		suites = append(suites, "atomic stress")
		flags = append(flags, "--atomic-stress")
	}
	if fuzzSuite {
		suites = append(suites, "fuzz")
		flags = append(flags, "--fuzz")
	}
	return fmt.Errorf("test suite %s (%s) for target %s is not implemented yet: no real target runner or oracle is wired; no fake or skipped tests will be emitted", strings.Join(suites, ", "), strings.Join(flags, ", "), target)
}

func runAllTargetsSuite(allTargets bool, brutal bool, diagnostics string, reportFormat string, stdout io.Writer, stderr io.Writer) int {
	targets := []string{"x86", "x64", "macos-x64", "windows-x64", "x32"}
	results := make([]compiler.TestRunnerResult, 0, 55)
	for _, raw := range targets {
		tgt, err := ctarget.Parse(raw)
		if err != nil {
			results = append(results, targetSuiteResult(0, "tetra:all-targets", raw+" target parse", 1, err))
			continue
		}
		results = append(results, runABISuiteResults(tgt)...)
	}
	if brutal {
		for _, raw := range targets {
			tgt, err := ctarget.Parse(raw)
			if err != nil {
				results = append(results, targetSuiteResult(0, "tetra:atomic-stress", raw+" atomic target parse", 1, err))
				continue
			}
			results = append(results, runAtomicStressSuiteResults(tgt)...)
		}
		for _, raw := range targets {
			tgt, err := ctarget.Parse(raw)
			if err != nil {
				results = append(results, targetSuiteResult(0, "tetra:fuzz", raw+" fuzz target parse", 1, err))
				continue
			}
			results = append(results, runFuzzSuiteResults(tgt)...)
		}
	}
	return writeTargetSuiteReport(results, "", reportFormat, stdout, stderr)
}

func runABISuiteResults(tgt ctarget.Target) []compiler.TestRunnerResult {
	if tgt.Triple == "linux-x86" {
		results, err := runX86ABISuite()
		if err != nil {
			return []compiler.TestRunnerResult{targetSuiteResult(0, "tetra:x86-abi", "x86 ABI suite", 1, err)}
		}
		return results
	}
	if tgt.Triple == "linux-x64" {
		results, err := runX64ABISuite()
		if err != nil {
			return []compiler.TestRunnerResult{targetSuiteResult(0, "tetra:x64-abi", "x64 ABI suite", 1, err)}
		}
		return results
	}
	if tgt.Triple == "linux-x32" {
		results, err := runX32ABISuite()
		if err != nil {
			return []compiler.TestRunnerResult{targetSuiteResult(0, "tetra:x32-abi", "x32 ABI suite", 1, err)}
		}
		return results
	}
	checks, err := compiler.RunTargetABIChecks(tgt.Triple)
	if err != nil {
		return []compiler.TestRunnerResult{targetSuiteResult(0, fmt.Sprintf("tetra:%s-abi", tgt.Arch), tgt.Arch.String()+" ABI suite", 1, err)}
	}
	return targetABICheckResults(tgt, checks)
}

func runAtomicStressSuiteResults(tgt ctarget.Target) []compiler.TestRunnerResult {
	checks, err := compiler.RunTargetAtomicStressChecks(tgt.Triple)
	if err != nil {
		return []compiler.TestRunnerResult{targetSuiteResult(0, fmt.Sprintf("tetra:%s-atomic-stress", tgt.Arch), tgt.Arch.String()+" atomic stress", 1, err)}
	}
	return targetAtomicStressCheckResults(tgt, checks)
}

func runFuzzSuiteResults(tgt ctarget.Target) []compiler.TestRunnerResult {
	checks, err := compiler.RunTargetFuzzChecks(tgt.Triple)
	if err != nil {
		return []compiler.TestRunnerResult{targetSuiteResult(0, fmt.Sprintf("tetra:%s-fuzz", tgt.Arch), tgt.Arch.String()+" fuzz", 1, err)}
	}
	return targetFuzzCheckResults(tgt, checks)
}

func unsupportedMatrixResult(filename string, name string, message string) compiler.TestRunnerResult {
	return targetSuiteResult(0, filename, name, 1, fmt.Errorf("%s", message))
}

func runTargetABISuite(targetName string, diagnostics string, reportFormat string, stdout io.Writer, stderr io.Writer) int {
	tgt, ok := parseBuildTargetOrReport(targetName, diagnostics, stderr)
	if !ok {
		return 2
	}
	if tgt.Triple == "linux-x86" {
		results, err := runX86ABISuite()
		if err != nil {
			writeDiagnostic(stderr, diagnostics, err)
			return 1
		}
		return writeTargetSuiteReport(results, tgt.Triple, reportFormat, stdout, stderr)
	}
	if tgt.Triple == "linux-x64" {
		results, err := runX64ABISuite()
		if err != nil {
			writeDiagnostic(stderr, diagnostics, err)
			return 1
		}
		return writeTargetSuiteReport(results, tgt.Triple, reportFormat, stdout, stderr)
	}
	if tgt.Triple != "linux-x32" {
		checks, err := compiler.RunTargetABIChecks(tgt.Triple)
		if err != nil {
			writeDiagnostic(stderr, diagnostics, unsupportedTargetTestSuiteDiagnostic(tgt.Triple, true, false, false))
			return 2
		}
		return writeTargetSuiteReport(targetABICheckResults(tgt, checks), tgt.Triple, reportFormat, stdout, stderr)
	}
	results, err := runX32ABISuite()
	if err != nil {
		writeDiagnostic(stderr, diagnostics, err)
		return 1
	}
	return writeTargetSuiteReport(results, tgt.Triple, reportFormat, stdout, stderr)
}

func runTargetAtomicStressSuite(targetName string, diagnostics string, reportFormat string, stdout io.Writer, stderr io.Writer) int {
	tgt, ok := parseBuildTargetOrReport(targetName, diagnostics, stderr)
	if !ok {
		return 2
	}
	checks, err := compiler.RunTargetAtomicStressChecks(tgt.Triple)
	if err != nil {
		writeDiagnostic(stderr, diagnostics, unsupportedTargetTestSuiteDiagnostic(tgt.Triple, false, true, false))
		return 2
	}
	return writeTargetSuiteReport(targetAtomicStressCheckResults(tgt, checks), tgt.Triple, reportFormat, stdout, stderr)
}

func runTargetFuzzSuite(targetName string, diagnostics string, reportFormat string, stdout io.Writer, stderr io.Writer) int {
	tgt, ok := parseBuildTargetOrReport(targetName, diagnostics, stderr)
	if !ok {
		return 2
	}
	checks, err := compiler.RunTargetFuzzChecks(tgt.Triple)
	if err != nil {
		writeDiagnostic(stderr, diagnostics, unsupportedTargetTestSuiteDiagnostic(tgt.Triple, false, false, true))
		return 2
	}
	return writeTargetSuiteReport(targetFuzzCheckResults(tgt, checks), tgt.Triple, reportFormat, stdout, stderr)
}

func writeTargetSuiteReport(results []compiler.TestRunnerResult, target string, reportFormat string, stdout io.Writer, stderr io.Writer) int {
	passed := 0
	for _, result := range results {
		if result.Passed {
			passed++
			if reportFormat == "text" {
				fmt.Fprintf(stdout, "PASS %s\n", result.Name)
			}
			continue
		}
		if reportFormat == "text" {
			if result.Error != "" {
				fmt.Fprintf(stdout, "FAIL %s (%s)\n", result.Name, result.Error)
			} else {
				fmt.Fprintf(stdout, "FAIL %s\n", result.Name)
			}
		}
	}
	if reportFormat == "json" {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(compiler.NewTestRunnerReportForTarget(results, target)); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	} else {
		fmt.Fprintf(stdout, "Tetra tests: %d/%d passed\n", passed, len(results))
	}
	if passed != len(results) {
		return 1
	}
	return 0
}

func runX32ABISuite() ([]compiler.TestRunnerResult, error) {
	tmpDir, err := os.MkdirTemp("", "tetra-x32-abi-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	tgt, err := ctarget.Parse("x32")
	if err != nil {
		return nil, err
	}
	checks, err := compiler.RunTargetABIChecks(tgt.Triple)
	if err != nil {
		return nil, err
	}
	results := targetABICheckResults(tgt, checks)
	cases := []struct {
		name string
		run  func(string) error
	}{
		{name: "x32 object ABI smoke", run: runX32ObjectABISmoke},
		{name: "x32 atomic ABI object", run: runX32AtomicABIObject},
		{name: "x32 executable matrix smoke", run: runX32ExecutableMatrixSmoke},
	}
	for i, tc := range cases {
		start := time.Now()
		err := tc.run(tmpDir)
		results = append(results, targetSuiteResult(len(checks)+i, "tetra:x32-abi", tc.name, elapsedMillis(time.Since(start)), err))
	}
	return results, nil
}

func targetABICheckResults(tgt ctarget.Target, checks []compiler.ABICheck) []compiler.TestRunnerResult {
	filename := fmt.Sprintf("tetra:%s-abi", targetSuiteFilenameStem(tgt))
	results := make([]compiler.TestRunnerResult, 0, len(checks))
	for i, check := range checks {
		err := error(nil)
		if check.Error != "" {
			err = fmt.Errorf("%s", check.Error)
		}
		results = append(results, targetSuiteResult(i, filename, check.Name, 1, err))
	}
	return results
}

func targetAtomicStressCheckResults(tgt ctarget.Target, checks []compiler.AtomicStressCheck) []compiler.TestRunnerResult {
	filename := fmt.Sprintf("tetra:%s-atomic-stress", targetSuiteFilenameStem(tgt))
	results := make([]compiler.TestRunnerResult, 0, len(checks))
	for i, check := range checks {
		err := error(nil)
		if check.Error != "" {
			err = fmt.Errorf("%s", check.Error)
		}
		results = append(results, targetSuiteResult(i, filename, check.Name, 1, err))
	}
	return results
}

func targetFuzzCheckResults(tgt ctarget.Target, checks []compiler.FuzzCheck) []compiler.TestRunnerResult {
	filename := fmt.Sprintf("tetra:%s-fuzz", targetSuiteFilenameStem(tgt))
	results := make([]compiler.TestRunnerResult, 0, len(checks))
	for i, check := range checks {
		err := error(nil)
		if check.Error != "" {
			err = fmt.Errorf("%s", check.Error)
		}
		results = append(results, targetSuiteResult(i, filename, check.Name, 1, err))
	}
	return results
}

func targetSuiteFilenameStem(tgt ctarget.Target) string {
	switch tgt.Triple {
	case "linux-x86":
		return "x86"
	case "linux-x64":
		return "x64"
	case "linux-x32":
		return "x32"
	default:
		return tgt.Triple
	}
}

func targetSuiteResult(index int, filename string, name string, durationMS int64, err error) compiler.TestRunnerResult {
	result := compiler.TestRunnerResult{
		Name:         name,
		Filename:     filename,
		Index:        index,
		FunctionName: targetSuiteFunctionName(name),
		Passed:       err == nil,
		DurationMS:   durationMS,
	}
	if err != nil {
		result.ExitCode = 1
		result.Error = err.Error()
	}
	return result
}

func targetSuiteFunctionName(name string) string {
	replacer := strings.NewReplacer(" ", "_", "-", "_", "/", "_", ":", "_")
	return "__tetra_test_" + replacer.Replace(strings.ToLower(strings.TrimSpace(name)))
}

func runX86ABISuite() ([]compiler.TestRunnerResult, error) {
	tmpDir, err := os.MkdirTemp("", "tetra-x86-abi-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	tgt, err := ctarget.Parse("x86")
	if err != nil {
		return nil, err
	}
	checks, err := compiler.RunTargetABIChecks(tgt.Triple)
	if err != nil {
		return nil, err
	}
	results := targetABICheckResults(tgt, checks)
	cases := []struct {
		name string
		run  func(string) error
	}{
		{name: "x86 object ABI smoke", run: runX86ObjectABISmoke},
		{name: "x86 atomic ABI object", run: func(string) error { return runAtomicABIObjectCheck("x86", "x86") }},
		{name: "x86 executable matrix smoke", run: runX86ExecutableMatrixSmoke},
	}
	for i, tc := range cases {
		start := time.Now()
		err := tc.run(tmpDir)
		results = append(results, targetSuiteResult(len(checks)+i, "tetra:x86-abi", tc.name, elapsedMillis(time.Since(start)), err))
	}
	return results, nil
}

func runX64ABISuite() ([]compiler.TestRunnerResult, error) {
	tmpDir, err := os.MkdirTemp("", "tetra-x64-abi-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	tgt, err := ctarget.Parse("x64")
	if err != nil {
		return nil, err
	}
	checks, err := compiler.RunTargetABIChecks(tgt.Triple)
	if err != nil {
		return nil, err
	}
	results := targetABICheckResults(tgt, checks)
	cases := []struct {
		name string
		run  func(string) error
	}{
		{name: "x64 object ABI smoke", run: runX64ObjectABISmoke},
		{name: "x64 atomic ABI object", run: func(string) error { return runAtomicABIObjectCheck("x64", "x64") }},
		{name: "x64 executable matrix smoke", run: runX64ExecutableMatrixSmoke},
	}
	for i, tc := range cases {
		start := time.Now()
		err := tc.run(tmpDir)
		results = append(results, targetSuiteResult(len(checks)+i, "tetra:x64-abi", tc.name, elapsedMillis(time.Since(start)), err))
	}
	return results, nil
}

func runAtomicABIObjectCheck(targetName string, prefix string) error {
	checks, err := compiler.RunTargetAtomicStressChecks(targetName)
	if err != nil {
		return err
	}
	want := prefix + " atomic object matrix"
	for _, check := range checks {
		if check.Name != want {
			continue
		}
		if check.Error != "" {
			return fmt.Errorf("%s: %s", want, check.Error)
		}
		return nil
	}
	return fmt.Errorf("atomic object matrix check %q was not produced for %s", want, targetName)
}

func runX64ObjectABISmoke(tmpDir string) error {
	srcPath := filepath.Join(tmpDir, "x64_abi_smoke.tetra")
	outPath := filepath.Join(tmpDir, "x64_abi_smoke.tobj")
	src := "@export(\"ffi_say_i32\")\nfun say(): i32 uses io {\n  print(\"x64 abi\\n\")\n  return 0\n}\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "x64", compiler.BuildOptions{Emit: compiler.EmitLibrary}); err != nil {
		return err
	}
	obj, err := compiler.ReadObject(outPath)
	if err != nil {
		return err
	}
	if obj.Target != "linux-x64" {
		return fmt.Errorf("target mismatch: got %q want linux-x64", obj.Target)
	}
	if !abiObjectHasSymbolSignature(obj, "ffi_say_i32", 0, 1) {
		return fmt.Errorf("object missing scalar exported ffi_say_i32 symbol: %#v", obj.Symbols)
	}
	if !containsMovEaxImm32(obj.Code, 1) || !bytes.Contains(obj.Code, []byte{0x0F, 0x05}) {
		return fmt.Errorf("missing linux-x64 write syscall in object code")
	}
	if containsMovEaxImm32(obj.Code, 0x40000001) {
		return fmt.Errorf("linux-x64 object emitted x32 write syscall number")
	}
	for _, reloc := range obj.Relocs {
		if reloc.Kind == compiler.RelocIATDisp32 {
			return fmt.Errorf("linux-x64 object unexpectedly has Windows IAT reloc: %#v", obj.Relocs)
		}
	}
	return nil
}

func runX64ExecutableMatrixSmoke(tmpDir string) error {
	cases := []struct {
		name string
		src  string
	}{
		{
			name: "control",
			src:  "func add(a: Int, b: Int) -> Int:\n    return a + b\n\nfunc main() -> Int:\n    var i: Int = 0\n    var acc: Int = 0\n    while i < 3:\n        acc = acc + i\n        i = i + 1\n    return add(acc, 39)\n",
		},
		{
			name: "aggregates",
			src:  "struct Pair:\n    left: Int\n    right: Int\n\nenum Msg:\n    case value(Pair)\n    case empty\n\nfunc pick() -> Msg:\n    return Msg.value(Pair(left: 40, right: 2))\n\nfunc main() -> Int:\n    match pick():\n    case Msg.value(pair):\n        return pair.left + pair.right\n    case Msg.empty:\n        return 0\n",
		},
		{
			name: "memory",
			src:  "fun main(): i32 uses alloc, mem {\n  var bytes: []u8 = make_u8(2)\n  var words: []u16 = make_u16(2)\n  var flags: []bool = make_bool(1)\n  bytes[0] = 40\n  bytes[1] = 1\n  words[0] = bytes[0] + bytes[1]\n  flags[0] = true\n  if flags[0] {\n    return words[0] + 1\n  }\n  return 0\n}\n",
		},
	}
	for _, tc := range cases {
		srcPath := filepath.Join(tmpDir, "x64_matrix_"+tc.name+".tetra")
		outPath := filepath.Join(tmpDir, "x64_matrix_"+tc.name)
		if err := os.WriteFile(srcPath, []byte(tc.src), 0o644); err != nil {
			return err
		}
		if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "x64", compiler.BuildOptions{Jobs: 1}); err != nil {
			return fmt.Errorf("%s: %w", tc.name, err)
		}
		if err := validateX64Executable(outPath); err != nil {
			return fmt.Errorf("%s: %w", tc.name, err)
		}
	}
	return nil
}

func validateX64Executable(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) < 64 {
		return fmt.Errorf("x64 executable too small: %d bytes", len(data))
	}
	if string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("x64 executable missing ELF magic: % x", data[:4])
	}
	if data[4] != 2 {
		return fmt.Errorf("x64 executable class = %d, want ELFCLASS64", data[4])
	}
	if machine := binary.LittleEndian.Uint16(data[18:20]); machine != 0x3e {
		return fmt.Errorf("x64 executable machine = %#x, want EM_X86_64", machine)
	}
	if !containsMovEaxImm32(data, 60) || !bytes.Contains(data, []byte{0x0F, 0x05}) {
		return fmt.Errorf("x64 executable missing x64 exit syscall")
	}
	if containsMovEaxImm32(data, 0x4000003c) {
		return fmt.Errorf("x64 executable emitted x32 exit syscall number")
	}
	if bytes.Contains(data, []byte{0xCD, 0x80}) {
		return fmt.Errorf("x64 executable emitted i386 int 0x80 syscall")
	}
	return nil
}

func runX86ObjectABISmoke(tmpDir string) error {
	srcPath := filepath.Join(tmpDir, "x86_abi_smoke.tetra")
	outPath := filepath.Join(tmpDir, "x86_abi_smoke.tobj")
	src := "@export(\"ffi_say_i32\")\nfun say(): i32 uses io {\n  print(\"x86 abi\\n\")\n  return 0\n}\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "x86", compiler.BuildOptions{Emit: compiler.EmitLibrary}); err != nil {
		return err
	}
	obj, err := compiler.ReadObject(outPath)
	if err != nil {
		return err
	}
	if obj.Target != "linux-x86" {
		return fmt.Errorf("target mismatch: got %q want linux-x86", obj.Target)
	}
	if !abiObjectHasSymbolSignature(obj, "ffi_say_i32", 0, 1) {
		return fmt.Errorf("object missing scalar exported ffi_say_i32 symbol: %#v", obj.Symbols)
	}
	if !bytes.Contains(obj.Code, []byte{0xB8, 0x04, 0x00, 0x00, 0x00, 0xCD, 0x80}) {
		return fmt.Errorf("missing i386 write syscall in object code")
	}
	for _, reloc := range obj.Relocs {
		if reloc.Kind == compiler.RelocIATDisp32 {
			return fmt.Errorf("linux-x86 object unexpectedly has Windows IAT reloc: %#v", obj.Relocs)
		}
	}
	return nil
}

type executableMatrixCase struct {
	name string
	src  string
}

func x86FamilyExecutableMatrixCases() []executableMatrixCase {
	return []executableMatrixCase{
		{
			name: "recursion",
			src:  "func fact(n: Int) -> Int:\n    if n <= 1:\n        return 1\n    return n * fact(n - 1)\n\nfunc main() -> Int:\n    return fact(5)\n",
		},
		{
			name: "globals_strings",
			src:  "val greeting: String = \"hello\"\nvar answer: Int = 1\n\nfunc main() -> Int:\n    let local: String = \"abc\"\n    answer = greeting.len + local.len + 34\n    return answer\n",
		},
		{
			name: "direct_callback",
			src:  "func add1(x: Int) -> Int:\n    return x + 1\n\nfunc apply(cb: fn(Int) -> Int, x: Int) -> Int:\n    return cb(x)\n\nfunc main() -> Int:\n    return apply(add1, 41)\n",
		},
		{
			name: "control",
			src:  "func add(a: Int, b: Int) -> Int:\n    return a + b\n\nfunc main() -> Int:\n    var i: Int = 0\n    var acc: Int = 0\n    while i < 3:\n        acc = acc + i\n        i = i + 1\n    return add(acc, 39)\n",
		},
		{
			name: "aggregates",
			src:  "struct Pair:\n    left: Int\n    right: Int\n\nenum Msg:\n    case value(Pair)\n    case empty\n\nfunc pick() -> Msg:\n    return Msg.value(Pair(left: 40, right: 2))\n\nfunc main() -> Int:\n    match pick():\n    case Msg.value(pair):\n        return pair.left + pair.right\n    case Msg.empty:\n        return 0\n",
		},
		{
			name: "memory",
			src:  "fun main(): i32 uses alloc, mem {\n  var bytes: []u8 = make_u8(2)\n  var words: []u16 = make_u16(2)\n  var flags: []bool = make_bool(1)\n  bytes[0] = 40\n  bytes[1] = 1\n  words[0] = bytes[0] + bytes[1]\n  flags[0] = true\n  if flags[0] {\n    return words[0] + 1\n  }\n  return 0\n}\n",
		},
		{
			name: "raw_memory",
			src:  "func main() -> Int\nuses alloc, capability, mem:\n    unsafe:\n        let mem: cap.mem = core.cap_mem()\n        let p: ptr = core.alloc_bytes(16)\n        let stored: Int = core.store_i32(core.ptr_add(p, 4, mem), 42, mem)\n        let value: Int = core.load_i32(core.ptr_add(p, 4, mem), mem)\n        let stored_ptr: ptr = core.store_ptr(core.ptr_add(p, 8, mem), p, mem)\n        let loaded_ptr: ptr = core.load_ptr(core.ptr_add(p, 8, mem), mem)\n        if value == 42:\n            return 0\n    return 1\n",
		},
		{
			name: "scoped_island",
			src:  "fun main(): i32 uses alloc, islands, mem {\n  var out: i32 = 0\n  island(64) as isl {\n    var xs: []u16 = core.island_make_u16(isl, 2)\n    xs[0] = 40\n    xs[1] = 2\n    out = xs[0] + xs[1]\n  }\n  return out\n}\n",
		},
		{
			name: "mmio",
			src:  "func main() -> Int\nuses alloc, capability, io, mem, mmio:\n    unsafe:\n        let mem: cap.mem = core.cap_mem()\n        let io_cap: cap.io = core.cap_io()\n        let p: ptr = core.alloc_bytes(4)\n        let stored: Int = core.store_i32(p, 41, mem)\n        let written: Int = core.mmio_write_i32(p, 42, io_cap)\n        return core.mmio_read_i32(p, io_cap)\n    return 0\n",
		},
	}
}

func runX86ExecutableMatrixSmoke(tmpDir string) error {
	for _, tc := range x86FamilyExecutableMatrixCases() {
		srcPath := filepath.Join(tmpDir, "x86_matrix_"+tc.name+".tetra")
		outPath := filepath.Join(tmpDir, "x86_matrix_"+tc.name)
		if err := os.WriteFile(srcPath, []byte(tc.src), 0o644); err != nil {
			return err
		}
		if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "x86", compiler.BuildOptions{Jobs: 1}); err != nil {
			return fmt.Errorf("%s: %w", tc.name, err)
		}
		if err := validateX86Executable(outPath); err != nil {
			return fmt.Errorf("%s: %w", tc.name, err)
		}
	}
	return nil
}

func validateX86Executable(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) < 52 {
		return fmt.Errorf("x86 executable too small: %d bytes", len(data))
	}
	if string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("x86 executable missing ELF magic: % x", data[:4])
	}
	if data[4] != 1 {
		return fmt.Errorf("x86 executable class = %d, want ELFCLASS32", data[4])
	}
	if machine := binary.LittleEndian.Uint16(data[18:20]); machine != 3 {
		return fmt.Errorf("x86 executable machine = %#x, want EM_386", machine)
	}
	if !bytes.Contains(data, []byte{0x89, 0xC3, 0xB8, 0x01, 0x00, 0x00, 0x00, 0xCD, 0x80}) {
		return fmt.Errorf("x86 executable missing i386 exit int 0x80 stub")
	}
	if containsMovEaxImm32(data, 60) {
		return fmt.Errorf("x86 executable emitted x64 exit syscall number")
	}
	return nil
}

func runX32ObjectABISmoke(tmpDir string) error {
	srcPath := filepath.Join(tmpDir, "x32_abi_smoke.tetra")
	outPath := filepath.Join(tmpDir, "x32_abi_smoke.tobj")
	src := "@export(\"ffi_say_i32\")\nfun say(): i32 uses io {\n  print(\"x32 abi\\n\")\n  return 0\n}\n"
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "x32", compiler.BuildOptions{Emit: compiler.EmitLibrary}); err != nil {
		return err
	}
	obj, err := compiler.ReadObject(outPath)
	if err != nil {
		return err
	}
	if obj.Target != "linux-x32" {
		return fmt.Errorf("target mismatch: got %q want linux-x32", obj.Target)
	}
	if !abiObjectHasSymbolSignature(obj, "ffi_say_i32", 0, 1) {
		return fmt.Errorf("object missing scalar exported ffi_say_i32 symbol: %#v", obj.Symbols)
	}
	if !containsMovEaxImm32(obj.Code, 0x40000001) {
		return fmt.Errorf("missing x32 write syscall number in object code")
	}
	if containsMovEaxImm32(obj.Code, 1) {
		return fmt.Errorf("linux-x32 object emitted plain x64 write syscall number")
	}
	for _, reloc := range obj.Relocs {
		if reloc.Kind == compiler.RelocIATDisp32 {
			return fmt.Errorf("linux-x32 object unexpectedly has Windows IAT reloc: %#v", obj.Relocs)
		}
	}
	return nil
}

func runX32AtomicABIObject(tmpDir string) error {
	srcPath := filepath.Join(tmpDir, "x32_atomic_abi.tetra")
	outPath := filepath.Join(tmpDir, "x32_atomic_abi.tobj")
	src := `
func atomic_probe() -> Int
uses alloc, capability, mem:
    unsafe:
        let mem: cap.mem = core.cap_mem()
        let p: ptr = core.alloc_bytes(8)
        let loaded: i64 = core.atomic_load_i64_acquire(p, mem)
        let exchanged: i64 = core.atomic_exchange_i64_seq_cst(p, loaded, mem)
        let weak64: i64 = core.atomic_compare_exchange_weak_i64_seq_cst(p, loaded, exchanged, mem)
        var ignored_store: i64 = core.atomic_store_i64_release(p, weak64, mem)
        return core.atomic_compare_exchange_weak_i32_seq_cst(p, 0, 1, mem)
    return 0
`
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		return err
	}
	if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "x32", compiler.BuildOptions{Emit: compiler.EmitLibrary}); err != nil {
		return err
	}
	obj, err := compiler.ReadObject(outPath)
	if err != nil {
		return err
	}
	if obj.Target != "linux-x32" {
		return fmt.Errorf("target mismatch: got %q want linux-x32", obj.Target)
	}
	if !abiObjectHasSymbol(obj, "atomic_probe") {
		return fmt.Errorf("object missing atomic_probe symbol: %#v", obj.Symbols)
	}
	if !bytes.Contains(obj.Code, []byte{0xF0, 0x4C, 0x0F, 0xB1, 0x07}) {
		return fmt.Errorf("missing qword weak-CAS codegen for i64 atomic on x32")
	}
	if !bytes.Contains(obj.Code, []byte{0xF0, 0x44, 0x0F, 0xB1, 0x07}) {
		return fmt.Errorf("missing dword weak-CAS codegen for i32 atomic on x32")
	}
	return nil
}

func runX32ExecutableMatrixSmoke(tmpDir string) error {
	for _, tc := range x86FamilyExecutableMatrixCases() {
		srcPath := filepath.Join(tmpDir, "x32_matrix_"+tc.name+".tetra")
		outPath := filepath.Join(tmpDir, "x32_matrix_"+tc.name)
		if err := os.WriteFile(srcPath, []byte(tc.src), 0o644); err != nil {
			return err
		}
		if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, "x32", compiler.BuildOptions{Jobs: 1}); err != nil {
			return fmt.Errorf("%s: %w", tc.name, err)
		}
		if err := validateX32Executable(outPath); err != nil {
			return fmt.Errorf("%s: %w", tc.name, err)
		}
	}
	return nil
}

func validateX32Executable(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(data) < 52 {
		return fmt.Errorf("x32 executable too small: %d bytes", len(data))
	}
	if string(data[:4]) != "\x7fELF" {
		return fmt.Errorf("x32 executable missing ELF magic: % x", data[:4])
	}
	if data[4] != 1 {
		return fmt.Errorf("x32 executable class = %d, want ELFCLASS32", data[4])
	}
	if machine := binary.LittleEndian.Uint16(data[18:20]); machine != 0x3e {
		return fmt.Errorf("x32 executable machine = %#x, want EM_X86_64", machine)
	}
	if !containsMovEaxImm32(data, 0x4000003c) {
		return fmt.Errorf("x32 executable missing x32 exit syscall number")
	}
	if containsMovEaxImm32(data, 60) {
		return fmt.Errorf("x32 executable emitted plain x64 exit syscall number")
	}
	if bytes.Contains(data, []byte{0xCD, 0x80}) {
		return fmt.Errorf("x32 executable emitted i386 int 0x80 syscall")
	}
	return nil
}

func abiObjectHasSymbol(obj *compiler.Object, name string) bool {
	if obj == nil {
		return false
	}
	for _, sym := range obj.Symbols {
		if strings.EqualFold(sym.Name, name) || sym.Name == name {
			return true
		}
	}
	return false
}

func abiObjectHasSymbolSignature(obj *compiler.Object, name string, params int, returns int) bool {
	if obj == nil {
		return false
	}
	for _, sym := range obj.Symbols {
		if !(strings.EqualFold(sym.Name, name) || sym.Name == name) {
			continue
		}
		return sym.HasSignature && sym.ParamSlots == params && sym.ReturnSlots == returns
	}
	return false
}

func containsMovEaxImm32(buf []byte, imm uint32) bool {
	for i := 0; i+5 <= len(buf); i++ {
		if buf[i] == 0xB8 && binary.LittleEndian.Uint32(buf[i+1:i+5]) == imm {
			return true
		}
	}
	return false
}

func elapsedMillis(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	ms := d.Milliseconds()
	if ms == 0 {
		return 1
	}
	return ms
}
