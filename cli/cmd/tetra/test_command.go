package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"tetra_language/compiler"
)

func runTest(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", defaultTarget(), "target triple ("+supportedTargetsHelp+")")
	diagnostics := fs.String("diagnostics", "text", "diagnostics format: text or json")
	reportFormat := fs.String("report", "text", "report format: text or json")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if !validateDiagnosticsMode(stderr, *diagnostics) {
		return 2
	}
	if *reportFormat != "text" && *reportFormat != "json" {
		writeValidationDiagnostic(stderr, *diagnostics, "unsupported --report format")
		return 2
	}
	paths := fs.Args()
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
	tgt, ok := parseBuildTargetOrReport(*target, *diagnostics, stderr)
	if !ok {
		return 2
	}
	if isWASMTargetTriple(tgt.Triple) {
		writeDiagnostic(stderr, *diagnostics, fmt.Errorf("cannot run tests for target %s: WASM test runner is not part of the current production runtime contract; use smoke/runtime reports for WASM execution evidence", tgt.Triple))
		return 2
	}
	host, ok := hostTarget()
	if !ok || host != tgt.Triple {
		writeDiagnostic(stderr, *diagnostics, fmt.Errorf("cannot run tests for target %s on host %s/%s", tgt.Triple, runtime.GOOS, runtime.GOARCH))
		return 2
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
			if modulePath := modulePathFromSource(runner.Source); modulePath != "" {
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
				opt := compiler.BuildOptions{
					Jobs:            1,
					ProjectRoot:     worldOpt.Root,
					SourceRoots:     append([]string(nil), worldOpt.SourceRoots...),
					DependencyRoots: append([]compiler.ModuleRoot(nil), worldOpt.DependencyRoots...),
					LinkObjectPaths: append([]string(nil), targetLinkObjects...),
				}
				if _, err := compiler.BuildFileWithStatsOpt(srcPath, outPath, tgt.Triple, opt); err != nil {
					writeDiagnostic(stderr, *diagnostics, err)
					return 1
				}
			} else {
				if err := compiler.BuildFile(srcPath, outPath, tgt.Triple); err != nil {
					writeDiagnostic(stderr, *diagnostics, err)
					return 1
				}
			}
			code := execProgram(outPath, io.Discard, io.Discard)
			name := runner.Name
			if name == "" {
				name = fmt.Sprintf("%s#%d", file, i+1)
			}
			result := runner.ResultWithDuration(code, nil, elapsedMillis(time.Since(start)))
			results = append(results, result)
			if code == 0 {
				passed++
				if *reportFormat == "text" {
					fmt.Fprintf(stdout, "PASS %s\n", name)
				}
			} else {
				if *reportFormat == "text" {
					if result.Error != "" {
						fmt.Fprintf(stdout, "FAIL %s (%s)\n", name, result.Error)
					} else {
						fmt.Fprintf(stdout, "FAIL %s\n", name)
					}
				}
			}
		}
	}
	if *reportFormat == "json" {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(compiler.NewTestRunnerReport(results)); err != nil {
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
