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

	"tetra_language/tools/internal/rambaseline"
)

func main() {
	outDir := flag.String("out-dir", "", "output artifact directory")
	iterations := flag.Int("iterations", 5, "steady-state churn rounds")
	workBytes := flag.Int("work-bytes", 4*1024*1024, "bytes touched per live/scratch round")
	flag.Parse()
	outDirProvided := flagWasProvided("out-dir")

	gitHead := gitOutput("rev-parse", "HEAD")
	gitStatus := gitOutput("status", "--short")
	if strings.TrimSpace(*outDir) == "" {
		*outDir = defaultOutDir(gitHead)
	}
	result, err := rambaseline.Run(rambaseline.Options{
		OutDir:         *outDir,
		Iterations:     *iterations,
		WorkBytes:      *workBytes,
		GitHead:        gitHead,
		GitStatusShort: gitStatus,
		Command:        reproducibleCommand(*outDir, outDirProvided, *iterations, *workBytes),
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

func reproducibleCommand(outDir string, outDirProvided bool, iterations int, workBytes int) []string {
	command := []string{"go", "run", "./tools/cmd/ram-p0-baseline"}
	if outDirProvided {
		command = append(command, "--out-dir", outDir)
	}
	return append(
		command,
		"--iterations", strconv.Itoa(iterations),
		"--work-bytes", strconv.Itoa(workBytes),
	)
}

func defaultOutDir(gitHead string) string {
	head := strings.TrimSpace(gitHead)
	if len(head) > 12 {
		head = head[:12]
	}
	if head == "" {
		head = "unknown"
	}
	return filepath.Join("reports", "stabilization", "tetra-ram-p0-baseline-"+head)
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
