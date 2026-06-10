package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tetra_language/compiler"
	ctarget "tetra_language/compiler/target"
)

type smokeCaseReport struct {
	Name               string `json:"name"`
	SrcPath            string `json:"src_path"`
	OutPath            string `json:"out_path"`
	ExpectedExit       int    `json:"expected_exit"`
	Unsupported        bool   `json:"unsupported,omitempty"`
	ExpectedDiagnostic string `json:"expected_diagnostic,omitempty"`
	Diagnostic         string `json:"diagnostic,omitempty"`
	ActualExit         *int   `json:"actual_exit,omitempty"`
	Ran                bool   `json:"ran"`
	Pass               bool   `json:"pass"`
	Error              string `json:"error,omitempty"`
}

type islandsDebugScopeRow struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	CaseName string `json:"case_name,omitempty"`
	SrcPath  string `json:"src_path,omitempty"`
	Evidence string `json:"evidence"`
	Reason   string `json:"reason"`
}

type smokeReport struct {
	Timestamp         string                 `json:"timestamp"`
	Target            string                 `json:"target"`
	BuildOnly         bool                   `json:"build_only"`
	Runner            string                 `json:"runner,omitempty"`
	Host              string                 `json:"host"`
	Version           string                 `json:"version"`
	GitHead           string                 `json:"git_head,omitempty"`
	IslandsDebug      bool                   `json:"islands_debug"`
	IslandsDebugScope []islandsDebugScopeRow `json:"islands_debug_scope,omitempty"`
	Total             int                    `json:"total"`
	Passed            int                    `json:"passed"`
	Failed            int                    `json:"failed"`
	Cases             []smokeCaseReport      `json:"cases"`
}

type smokeCase struct {
	name               string
	srcPath            string
	expectedExit       int
	debugOnly          bool
	expectedDiagnostic string
}

type smokeListCase struct {
	Name               string `json:"name"`
	SrcPath            string `json:"src_path"`
	TargetGroup        string `json:"target_group"`
	ExpectedExit       int    `json:"expected_exit"`
	Unsupported        bool   `json:"unsupported,omitempty"`
	ExpectedDiagnostic string `json:"expected_diagnostic,omitempty"`
	DebugOnly          bool   `json:"debug_only,omitempty"`
}

type smokeExcludedExample struct {
	SrcPath string `json:"src_path"`
	Reason  string `json:"reason"`
}

type smokeListReport struct {
	Target           string                 `json:"target"`
	BuildOnly        bool                   `json:"build_only"`
	RunSupported     bool                   `json:"run_supported"`
	Total            int                    `json:"total"`
	IslandsDebug     bool                   `json:"islands_debug"`
	Cases            []smokeListCase        `json:"cases"`
	ExcludedExamples []smokeExcludedExample `json:"excluded_examples,omitempty"`
}

func runSmoke(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("smoke", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", defaultTarget(), "target triple ("+supportedTargetsHelp+")")
	runBuilt := fs.Bool("run", true, "run built binaries when host matches target")
	reportPath := fs.String("report", "", "write JSON smoke report")
	listCases := fs.Bool("list", false, "list smoke cases without building")
	listFormat := fs.String("format", "text", "smoke list format: text or json")
	islandsDebug := fs.Bool("islands-debug", false, "enable islands debug runtime checks")
	runtimeMode := fs.String("runtime", "auto", "actors runtime: auto, selfhost, or builtin")
	jobs := fs.Int("jobs", 1, "parallel module build jobs")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "smoke does not accept positional arguments")
		return 2
	}
	if *listCases {
		tgt, ok := parseBuildTargetOrReport(*target, "text", stderr)
		if !ok {
			return 2
		}
		return writeSmokeList(stdout, stderr, smokeCasesForTarget(*islandsDebug, tgt), *islandsDebug, *listFormat, tgt)
	}
	if *listFormat != "text" {
		fmt.Fprintln(stderr, "--format is only supported with --list")
		return 2
	}
	tgt, ok := parseBuildTargetOrReport(*target, "text", stderr)
	if !ok {
		return 2
	}
	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	tmpDir, err := os.MkdirTemp("", "tetra-smoke-*")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer os.RemoveAll(tmpDir)
	outputDir := tmpDir
	if tgt.Arch == ctarget.ArchWASM32 && *reportPath != "" {
		outputDir = smokeArtifactDir(*reportPath)
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}

	host := ""
	hostTriple, hostOK := hostTarget()
	if hostOK {
		host = hostTriple
	}
	cases := smokeCasesForTarget(*islandsDebug, tgt)
	shouldRun := *runBuilt && hostOK && hostTriple == tgt.Triple
	runWASI := false
	var wasiRunner wasiRunner
	runWeb := false
	var webRunner string
	if *runBuilt && tgt.Triple == "wasm32-wasi" {
		runner, err := discoverWASIRunner(repoRoot)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		wasiRunner = runner
		runWASI = true
		shouldRun = true
	}
	if *runBuilt && tgt.Triple == "wasm32-web" {
		runner, err := discoverWebRunner()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		webRunner = runner
		runWeb = true
		shouldRun = true
	}
	opt, err := buildOptions("exe", *runtimeMode, *islandsDebug, "", nil, *jobs)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	report := smokeReport{
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Target:       tgt.Triple,
		BuildOnly:    ctarget.IsBuildOnlyTarget(tgt.Triple),
		Runner:       runnerName(wasiRunner.Name, webRunner),
		Host:         host,
		Version:      compiler.Version(),
		GitHead:      gitHead(repoRoot),
		IslandsDebug: *islandsDebug,
	}
	if *islandsDebug {
		report.IslandsDebugScope = islandsDebugScopeRows()
	}
	for _, c := range cases {
		outPath := filepath.Join(outputDir, c.name+tgt.ExeExt)
		srcAbs := filepath.Join(repoRoot, filepath.FromSlash(c.srcPath))
		caseReport := smokeCaseReport{
			Name:               c.name,
			SrcPath:            c.srcPath,
			OutPath:            outPath,
			ExpectedExit:       c.expectedExit,
			Unsupported:        c.expectedDiagnostic != "",
			ExpectedDiagnostic: c.expectedDiagnostic,
		}
		if _, err := compiler.BuildFileWithStatsOpt(srcAbs, outPath, tgt.Triple, opt); err != nil {
			if c.expectedDiagnostic != "" {
				caseReport.OutPath = ""
				caseReport.Diagnostic = err.Error()
				if strings.Contains(err.Error(), c.expectedDiagnostic) {
					caseReport.Pass = true
				} else {
					caseReport.Error = "build diagnostic mismatch: " + err.Error()
				}
				report.Cases = append(report.Cases, caseReport)
				continue
			}
			caseReport.Error = "build: " + err.Error()
			report.Cases = append(report.Cases, caseReport)
			continue
		}
		if c.expectedDiagnostic != "" {
			caseReport.Error = "build succeeded, want diagnostic containing " + c.expectedDiagnostic
			report.Cases = append(report.Cases, caseReport)
			continue
		}
		if shouldRun {
			caseReport.Ran = true
			var actual int
			if runWASI {
				actual, err = execWASMProgramWithRunner(outPath, wasiRunner, io.Discard, io.Discard)
				if err != nil {
					caseReport.Error = "run: " + err.Error()
					caseReport.Pass = false
					report.Cases = append(report.Cases, caseReport)
					continue
				}
			} else if runWeb {
				actual, err = execWebProgramWithBrowserRunner(outPath, webRunner, io.Discard, io.Discard)
				if err != nil {
					caseReport.Error = "run: " + err.Error()
					caseReport.Pass = false
					report.Cases = append(report.Cases, caseReport)
					continue
				}
			} else {
				actual = execProgram(outPath, io.Discard, io.Discard)
			}
			caseReport.ActualExit = &actual
			caseReport.Pass = actual == c.expectedExit
		} else {
			caseReport.Pass = true
		}
		report.Cases = append(report.Cases, caseReport)
	}

	passed := 0
	for _, c := range report.Cases {
		if c.Pass {
			passed++
		}
	}
	report.Total = len(report.Cases)
	report.Passed = passed
	report.Failed = report.Total - report.Passed
	fmt.Fprintf(stdout, "Smoke %s: %d/%d passed\n", tgt.Triple, passed, len(report.Cases))
	if *reportPath != "" {
		if err := writeJSON(*reportPath, report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	if passed != len(report.Cases) {
		return 1
	}
	return 0
}

func islandsDebugScopeRows() []islandsDebugScopeRow {
	return []islandsDebugScopeRow{
		{
			Name:     "overflow_trap",
			Status:   "live_trap",
			CaseName: "islands_overflow",
			SrcPath:  "examples/islands_overflow.tetra",
			Evidence: "tetra smoke --islands-debug executes islands_overflow and observes non-zero trap exit",
			Reason:   "live sanitizer trap row for bounded island allocation overflow",
		},
		{
			Name:     "double_free",
			Status:   "static_only_nonclaim",
			CaseName: "islands_double_free",
			SrcPath:  "examples/islands_double_free.tetra",
			Evidence: "compiler/tests/runtime/resource_finalization_test.go; compiler/compiler_test.go; compiler/internal/backend/x64abi/abi_test.go",
			Reason:   "static semantics reject double-free before runtime; backend freed-marker trap is covered, but no live double-free bypass is claimed",
		},
		{
			Name:     "use_after_free",
			Status:   "static_only_nonclaim",
			Evidence: "compiler/internal/validation/validation_test.go; compiler/tests/runtime/resource_finalization_test.go",
			Reason:   "static validation rejects island use-after-free before runtime; no live UAF sanitizer row is claimed",
		},
		{
			Name:     "stale_epoch",
			Status:   "static_only_nonclaim",
			Evidence: "compiler/tests/runtime/resource_finalization_test.go; compiler/internal/islandkernel/kernel_test.go; compiler/internal/memoryfacts/report_test.go",
			Reason:   "reset/stale-epoch misuse is covered by static/kernel/report validators; no live stale-epoch sanitizer row is claimed",
		},
		{
			Name:     "wrong_island",
			Status:   "static_only_nonclaim",
			Evidence: "compiler/internal/islandkernel/kernel_test.go; tools/validators/islandproof/proof_test.go",
			Reason:   "wrong-island proof/report misuse is covered by static verifier evidence; no live wrong-island sanitizer row is claimed",
		},
	}
}

func runnerName(names ...string) string {
	for _, name := range names {
		if name != "" {
			return name
		}
	}
	return ""
}

func smokeArtifactDir(reportPath string) string {
	base := filepath.Base(reportPath)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	if stem == "" || stem == "." {
		stem = "smoke"
	}
	return filepath.Join(filepath.Dir(reportPath), stem+"-artifacts")
}

func writeSmokeList(stdout io.Writer, stderr io.Writer, cases []smokeCase, islandsDebug bool, format string, tgt ctarget.Target) int {
	host, hostOK := hostTarget()
	runSupported, _, _ := targetRunSupport(tgt, host, hostOK)
	report := smokeListReport{
		Target:       tgt.Triple,
		BuildOnly:    ctarget.IsBuildOnlyTarget(tgt.Triple),
		RunSupported: runSupported,
		Total:        len(cases),
		IslandsDebug: islandsDebug,
		Cases:        make([]smokeListCase, 0, len(cases)),
	}
	for _, c := range cases {
		report.Cases = append(report.Cases, smokeListCase{
			Name:               c.name,
			SrcPath:            c.srcPath,
			TargetGroup:        smokeTargetGroup(tgt.Triple),
			ExpectedExit:       c.expectedExit,
			Unsupported:        c.expectedDiagnostic != "",
			ExpectedDiagnostic: c.expectedDiagnostic,
			DebugOnly:          c.debugOnly,
		})
	}
	if repoRoot, err := findRepoRoot(); err == nil {
		report.ExcludedExamples = smokeExampleExclusions(repoRoot, cases, tgt)
	}
	switch format {
	case "", "text":
		for _, c := range report.Cases {
			if c.DebugOnly {
				fmt.Fprintf(stdout, "%s %s exit=%d debug-only\n", c.Name, c.SrcPath, c.ExpectedExit)
			} else {
				fmt.Fprintf(stdout, "%s %s exit=%d\n", c.Name, c.SrcPath, c.ExpectedExit)
			}
		}
		return 0
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	default:
		fmt.Fprintln(stderr, "unsupported --format")
		return 2
	}
}

func smokeExampleExclusions(repoRoot string, cases []smokeCase, tgt ctarget.Target) []smokeExcludedExample {
	covered := map[string]bool{}
	for _, c := range cases {
		covered[filepath.ToSlash(filepath.Clean(c.srcPath))] = true
	}

	examplesRoot := filepath.Join(repoRoot, "examples")
	var out []smokeExcludedExample
	walkErr := filepath.WalkDir(examplesRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !compiler.IsSourceFile(path) {
			return nil
		}
		rel, err := filepath.Rel(examplesRoot, path)
		if err != nil {
			return nil
		}
		srcPath := "examples/" + filepath.ToSlash(rel)
		if covered[srcPath] {
			return nil
		}
		out = append(out, smokeExcludedExample{
			SrcPath: srcPath,
			Reason:  fmt.Sprintf("not part of %s smoke profile", tgt.Triple),
		})
		return nil
	})
	if walkErr != nil && len(out) == 0 {
		return out
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SrcPath < out[j].SrcPath })
	return out
}

func smokeTargetGroup(target string) string {
	if target == "wasm32-wasi" || target == "wasm32-web" {
		return "wasm"
	}
	return "native"
}
