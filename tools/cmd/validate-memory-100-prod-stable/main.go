package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"tetra_language/tools/internal/memory100validate"
)

const memory100HashSchema = memory100validate.HashSchema

type memory100HashManifest = memory100validate.HashManifest
type memory100HashArtifact = memory100validate.HashArtifact

var requiredMemory100MemoryReleaseArtifacts = memory100validate.RequiredMemoryReleaseArtifacts

var requiredMemory100RAMContractReleaseArtifacts = memory100validate.RequiredRAMContractReleaseArtifacts
var requiredMemory100IntegratedArtifacts = memory100validate.RequiredIntegratedArtifacts

func main() {
	reportDir := flag.String("report-dir", "", "Memory100 aggregate report directory")
	currentGitHead := flag.String("current-git-head", "", "optional current git HEAD to require")
	flag.Parse()
	if strings.TrimSpace(*reportDir) == "" {
		fmt.Fprintln(os.Stderr, "error: --report-dir is required")
		os.Exit(2)
	}
	if err := validateMemory100ReportDir(*reportDir, *currentGitHead); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateMemory100ReportDir(reportDir string, currentGitHead string) error {
	return memory100validate.ValidateReportDir(reportDir, currentGitHead)
}
