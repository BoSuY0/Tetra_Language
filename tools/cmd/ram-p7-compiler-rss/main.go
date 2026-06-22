package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"tetra_language/tools/internal/ramcompilerrss"
)

func main() {
	outDir := flag.String("out-dir", "", "output artifact directory")
	samples := flag.Int("samples", 1, "valid samples per compiler RSS scenario")
	matrix := flag.String("matrix", ramcompilerrss.ScenarioMatrixDefault, "compiler RSS scenario matrix: default, p7_5, representative, or full_repo")
	compareBaselineDir := flag.String("compare-baseline-dir", "", "baseline compiler RSS bundle directory")
	compareCandidateDir := flag.String("compare-candidate-dir", "", "candidate compiler RSS bundle directory")
	compareOut := flag.String("compare-out", "", "baseline/candidate comparison output JSON path")
	flag.Parse()
	outDirProvided := flagWasProvided("out-dir")
	samplesProvided := flagWasProvided("samples")
	matrixProvided := flagWasProvided("matrix")
	compareBaselineProvided := flagWasProvided("compare-baseline-dir")
	compareCandidateProvided := flagWasProvided("compare-candidate-dir")
	compareOutProvided := flagWasProvided("compare-out")
	if compareBaselineProvided || compareCandidateProvided {
		if !compareBaselineProvided || !compareCandidateProvided {
			fmt.Fprintln(os.Stderr, "--compare-baseline-dir and --compare-candidate-dir must be provided together")
			os.Exit(2)
		}
		outPath := comparisonOutPath(*compareCandidateDir, *compareOut)
		commandOutPath := ""
		if compareOutProvided {
			commandOutPath = outPath
		}
		result, err := ramcompilerrss.CompareBundles(ramcompilerrss.CompareOptions{
			BaselineDir:  *compareBaselineDir,
			CandidateDir: *compareCandidateDir,
			OutPath:      outPath,
			Command:      reproducibleComparisonCommand(*compareBaselineDir, *compareCandidateDir, commandOutPath),
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(result.OutPath)
		return
	}

	gitHead := gitOutput("rev-parse", "HEAD")
	gitStatus := gitOutput("status", "--short")
	if strings.TrimSpace(*outDir) == "" {
		*outDir = defaultOutDir(gitHead)
	}
	result, err := ramcompilerrss.Run(ramcompilerrss.Options{
		OutDir:         *outDir,
		GitHead:        gitHead,
		GitStatusShort: gitStatus,
		Command:        reproducibleCommand(*outDir, outDirProvided, *samples, samplesProvided, *matrix, matrixProvided),
		Samples:        *samples,
		Matrix:         *matrix,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(result.ManifestPath)
}

func flagWasProvided(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func reproducibleCommand(outDir string, outDirProvided bool, samples int, samplesProvided bool, matrix string, matrixProvided bool) []string {
	command := []string{"go", "run", "./tools/cmd/ram-p7-compiler-rss"}
	if outDirProvided {
		command = append(command, "--out-dir", outDir)
	}
	if samplesProvided {
		command = append(command, "--samples", strconv.Itoa(samples))
	}
	if matrixProvided {
		command = append(command, "--matrix", matrix)
	}
	return command
}

func comparisonOutPath(candidateDir string, explicitOut string) string {
	if strings.TrimSpace(explicitOut) != "" {
		return explicitOut
	}
	return filepath.Join(candidateDir, "baseline-candidate-comparison.json")
}

func reproducibleComparisonCommand(baselineDir string, candidateDir string, outPath string) []string {
	command := []string{
		"go",
		"run",
		"./tools/cmd/ram-p7-compiler-rss",
		"--compare-baseline-dir",
		baselineDir,
		"--compare-candidate-dir",
		candidateDir,
	}
	if strings.TrimSpace(outPath) != "" {
		command = append(command, "--compare-out", outPath)
	}
	return command
}

func defaultOutDir(gitHead string) string {
	head := strings.TrimSpace(gitHead)
	if len(head) > 12 {
		head = head[:12]
	}
	if head == "" {
		head = "unknown"
	}
	return filepath.Join("reports", "stabilization", "tetra-ram-p7-compiler-rss-"+head)
}

func gitOutput(args ...string) string {
	cmd := exec.Command("git", args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return ""
	}
	return strings.TrimSpace(stdout.String())
}
