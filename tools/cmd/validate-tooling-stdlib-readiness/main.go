package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

type readinessEvidence struct {
	Features     []byte
	StdlibDocs   []byte
	CLIContracts []byte
}

type featuresReport struct {
	Schema   string         `json:"schema"`
	Version  string         `json:"version"`
	Features []featureEntry `json:"features"`
}

type featureEntry struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Status    string   `json:"status"`
	Since     string   `json:"since,omitempty"`
	Scope     string   `json:"scope"`
	Stability string   `json:"stability"`
	Docs      []string `json:"docs"`
}

var requiredStdlibModules = []string{
	"lib.core.collections",
	"lib.core.strings",
	"lib.core.slices",
	"lib.core.math",
	"lib.core.io",
	"lib.core.filesystem",
	"lib.core.networking",
	"lib.core.async",
	"lib.core.sync",
	"lib.core.testing",
	"lib.core.serialization",
	"lib.core.time",
	"lib.core.crypto",
}

var requiredCLICommands = []string{
	"check",
	"build",
	"run",
	"fmt",
	"test",
	"doc",
	"doctor",
	"project",
	"workspace",
	"lsp",
}

var forbiddenStdlibProductionPhrases = []string{
	"placeholder",
	"mock",
	"explicit placeholder",
	"stable placeholder",
	"placeholder interface",
	"placeholder modules",
	"placeholder semantics",
	"placeholder-level",
	"not production host",
	"not a host filesystem",
	"not a socket",
	"not cryptographic",
	"not production cryptography",
	"no socket or network",
}

var forbiddenCLIProductionPhrases = []string{
	"single-file textual MVP",
	"syntax-aware rename is not implemented",
	"not a syntax-aware rename",
}

func main() {
	featuresPath := flag.String("features", "", "features JSON produced by ./tetra features --format=json")
	stdlibDocsPath := flag.String("stdlib-docs", "docs/spec/stdlib.md", "stdlib production/readiness docs")
	cliContractsPath := flag.String("cli-contracts", "docs/spec/cli_contracts.md", "CLI contracts docs")
	flag.Parse()

	evidence, err := readReadinessEvidence(*featuresPath, *stdlibDocsPath, *cliContractsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "validate-tooling-stdlib-readiness: %v\n", err)
		os.Exit(2)
	}
	if err := validateToolingStdlibReadiness(evidence); err != nil {
		fmt.Fprintf(os.Stderr, "validate-tooling-stdlib-readiness: %v\n", err)
		os.Exit(1)
	}
}

func readReadinessEvidence(featuresPath, stdlibDocsPath, cliContractsPath string) (readinessEvidence, error) {
	readRequired := func(path, label string) ([]byte, error) {
		if strings.TrimSpace(path) == "" {
			return nil, fmt.Errorf("%s path is required", label)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", label, err)
		}
		return raw, nil
	}
	features, err := readRequired(featuresPath, "features")
	if err != nil {
		return readinessEvidence{}, err
	}
	stdlibDocs, err := readRequired(stdlibDocsPath, "stdlib docs")
	if err != nil {
		return readinessEvidence{}, err
	}
	cliContracts, err := readRequired(cliContractsPath, "CLI contracts")
	if err != nil {
		return readinessEvidence{}, err
	}
	return readinessEvidence{
		Features:     features,
		StdlibDocs:   stdlibDocs,
		CLIContracts: cliContracts,
	}, nil
}

func validateToolingStdlibReadiness(evidence readinessEvidence) error {
	var issues []string

	report, err := decodeFeaturesReport(evidence.Features)
	if err != nil {
		return err
	}
	if report.Schema != "tetra.features.v1" {
		issues = append(issues, fmt.Sprintf("features schema = %q, want tetra.features.v1", report.Schema))
	}
	if strings.TrimSpace(report.Version) == "" {
		issues = append(issues, "features version is required")
	}
	featuresByID := map[string]featureEntry{}
	for _, feature := range report.Features {
		if feature.ID == "" {
			issues = append(issues, "feature missing id")
			continue
		}
		if _, exists := featuresByID[feature.ID]; exists {
			issues = append(issues, fmt.Sprintf("duplicate feature %s", feature.ID))
			continue
		}
		featuresByID[feature.ID] = feature
	}
	issues = append(issues, validateCurrentFeature(featuresByID, "cli.core")...)
	issues = append(issues, validateCurrentFeature(featuresByID, "stdlib.core-current")...)
	issues = append(issues, validateCurrentFeature(featuresByID, "stdlib.experimental-mirrors")...)
	issues = append(issues, validateStdlibFeature(featuresByID["stdlib.core-current"])...)
	issues = append(issues, validateStdlibDocs(evidence.StdlibDocs)...)
	issues = append(issues, validateCLIContracts(evidence.CLIContracts)...)

	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func decodeFeaturesReport(raw []byte) (featuresReport, error) {
	var report featuresReport
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&report); err != nil {
		return featuresReport{}, fmt.Errorf("invalid features JSON: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return featuresReport{}, fmt.Errorf("features JSON contains trailing document")
		}
		return featuresReport{}, fmt.Errorf("invalid features JSON: %w", err)
	}
	return report, nil
}

func validateCurrentFeature(featuresByID map[string]featureEntry, id string) []string {
	feature, ok := featuresByID[id]
	if !ok {
		return []string{fmt.Sprintf("feature %s missing from features report", id)}
	}
	var issues []string
	if feature.Status != "current" {
		issues = append(issues, fmt.Sprintf("feature %s status = %s, want current", id, feature.Status))
	}
	if strings.TrimSpace(feature.Since) == "" {
		issues = append(issues, fmt.Sprintf("feature %s missing since", id))
	}
	if strings.TrimSpace(feature.Scope) == "" || strings.TrimSpace(feature.Stability) == "" {
		issues = append(issues, fmt.Sprintf("feature %s missing scope or stability", id))
	}
	if len(feature.Docs) == 0 {
		issues = append(issues, fmt.Sprintf("feature %s missing docs", id))
	}
	return issues
}

func validateStdlibFeature(feature featureEntry) []string {
	if feature.ID == "" {
		return nil
	}
	var issues []string
	text := strings.ToLower(feature.Scope + "\n" + feature.Stability)
	for _, phrase := range forbiddenStdlibProductionPhrases {
		if strings.Contains(text, phrase) {
			issues = append(issues, fmt.Sprintf("feature stdlib.core-current contains production-blocking phrase %q", phrase))
		}
	}
	for _, module := range requiredStdlibModules {
		if !strings.Contains(feature.Scope, module) && !strings.Contains(feature.Stability, module) {
			// The full module list may live in docs rather than the short feature row, so this
			// remains a docs-level check below.
			continue
		}
	}
	return issues
}

func validateStdlibDocs(raw []byte) []string {
	text := string(raw)
	lower := strings.ToLower(text)
	var issues []string
	for _, module := range requiredStdlibModules {
		if !strings.Contains(text, module) {
			issues = append(issues, fmt.Sprintf("docs/spec/stdlib.md missing required module %s", module))
		}
	}
	for _, phrase := range forbiddenStdlibProductionPhrases {
		if strings.Contains(lower, phrase) {
			issues = append(issues, fmt.Sprintf("docs/spec/stdlib.md contains production-blocking phrase %q", phrase))
		}
	}
	return issues
}

func validateCLIContracts(raw []byte) []string {
	text := string(raw)
	lower := strings.ToLower(text)
	var issues []string
	for _, command := range requiredCLICommands {
		if !strings.Contains(text, command) {
			issues = append(issues, fmt.Sprintf("docs/spec/cli_contracts.md missing CLI command %s", command))
		}
	}
	for _, phrase := range []string{
		"future v1 required tooling contract",
		"current public profile for this branch are `v0.3.0`",
		"current public profile for this branch is `v0.3.0`",
		"not current release readiness",
	} {
		if strings.Contains(lower, phrase) {
			issues = append(issues, fmt.Sprintf("docs/spec/cli_contracts.md contains stale tooling-readiness phrase %q", phrase))
		}
	}
	for _, phrase := range forbiddenCLIProductionPhrases {
		if strings.Contains(lower, strings.ToLower(phrase)) {
			issues = append(issues, fmt.Sprintf("docs/spec/cli_contracts.md contains production-blocking phrase %q", phrase))
		}
	}
	return issues
}
