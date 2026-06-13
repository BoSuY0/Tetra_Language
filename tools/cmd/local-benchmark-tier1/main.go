package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"
)

func main() {
	outDir := flag.String("out-dir", "reports/local-benchmark-tier1-v1", "output artifact directory")
	iterations := flag.Int("iterations", 3, "run iterations per benchmark row")
	timeout := flag.Duration("timeout", 20*time.Second, "timeout per build/run command")
	flag.Parse()
	if err := run(options{OutDir: *outDir, Iterations: *iterations, Timeout: *timeout}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(opt options) error {
	if opt.Iterations <= 0 {
		return fmt.Errorf("iterations must be positive")
	}
	if err := os.MkdirAll(opt.OutDir, 0o755); err != nil {
		return err
	}
	for _, rel := range []string{"artifacts", "report.json", "summary.md"} {
		if err := os.RemoveAll(filepath.Join(opt.OutDir, rel)); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Join(opt.OutDir, "artifacts", "src"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(opt.OutDir, "artifacts", "bin"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(opt.OutDir, "artifacts", "raw"), 0o755); err != nil {
		return err
	}

	root, err := os.Getwd()
	if err != nil {
		return err
	}
	env := commandEnv(root)
	tetraTool := filepath.Join(opt.OutDir, "artifacts", "bin", "tetra")
	tetraBuildStdout := filepath.Join(opt.OutDir, "artifacts", "raw", "tetra_cli_build.stdout.txt")
	tetraBuildStderr := filepath.Join(opt.OutDir, "artifacts", "raw", "tetra_cli_build.stderr.txt")
	if _, _, err := runCaptured(opt.Timeout, []string{"go", "build", "-o", tetraTool, "./cli/cmd/tetra"}, env, tetraBuildStdout, tetraBuildStderr); err != nil {
		return fmt.Errorf("build local tetra CLI: %w", err)
	}

	versions := compilerVersions(opt.Timeout, env, tetraTool)
	optimizerArtifact, err := writeOptimizerArtifact(opt.OutDir)
	if err != nil {
		return err
	}
	report := tier1Report{
		Schema:      schemaLocalBenchmarkTier1,
		Scope:       scopeP25RealLocalBenchmark,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Host: tier1Host{
			GOOS:      runtime.GOOS,
			GOARCH:    runtime.GOARCH,
			CPUs:      runtime.NumCPU(),
			TargetCPU: detectTargetCPU(),
			GitCommit: gitCommit(opt.Timeout, env),
		},
		Policy: tier1Policy{
			Tier:                "tier1_local_benchmark_evidence",
			ComparableThreshold: 0.20,
			Iterations:          opt.Iterations,
		},
		NonClaims: []string{
			"no fastest-language claim",
			"no official benchmark claim",
			"no cross-machine claim",
			"no TechEmpower claim",
			"no production claim",
		},
		OptimizerValidation: optimizerValidation{Status: "current_supported_subset", Artifact: optimizerArtifact},
	}

	rowsByCategory := map[string][]benchmarkRow{}
	for _, spec := range buildBenchmarkSpecs(opt.OutDir) {
		row := executeSpec(spec, opt, env, versions, tetraTool, optimizerArtifact)
		rowsByCategory[spec.Category] = append(rowsByCategory[spec.Category], row)
	}
	for _, category := range requiredP20Categories {
		rows := rowsByCategory[category]
		sort.Slice(rows, func(i, j int) bool {
			return languageOrder(rows[i].Language) < languageOrder(rows[j].Language)
		})
		classification, reason := classifyCategory(category, rows, report.Policy.ComparableThreshold)
		report.Results = append(report.Results, categoryResult{
			Category:             category,
			AlgorithmID:          "p25.0." + slug(category),
			InputDescription:     inputDescription(category),
			Classification:       classification,
			ClassificationReason: reason,
			Rows:                 rows,
		})
	}

	if err := writeJSON(filepath.Join(opt.OutDir, "report.json"), report); err != nil {
		return err
	}
	if err := writeSummary(filepath.Join(opt.OutDir, "summary.md"), report); err != nil {
		return err
	}
	if err := writeAudit(filepath.Join("docs", "audits", "local-benchmark-tier1-v1.md"), report); err != nil {
		return err
	}
	return nil
}
