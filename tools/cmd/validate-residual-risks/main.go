package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

const residualRisksSchema = "tetra.release.residual-risks.v1"
const residualRisksArtifact = "residual-risks.json"

type residualRisks struct {
	Schema         string         `json:"schema"`
	ReleaseVersion string         `json:"release_version"`
	Artifact       string         `json:"artifact"`
	Risks          []residualRisk `json:"risks"`
}

type residualRisk struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Owner    string `json:"owner"`
	Status   string `json:"status"`
	Summary  string `json:"summary,omitempty"`
	Evidence string `json:"evidence,omitempty"`
}

func main() {
	var path string
	var expectedVersion string
	flag.StringVar(&path, "artifact", "", "path to residual-risks.json")
	flag.StringVar(&expectedVersion, "expected-version", "", "expected release version")
	flag.Parse()

	if path == "" {
		fmt.Fprintln(os.Stderr, "error: --artifact is required")
		os.Exit(2)
	}
	if expectedVersion == "" {
		fmt.Fprintln(os.Stderr, "error: --expected-version is required")
		os.Exit(2)
	}
	if err := validateResidualRisksFile(path, expectedVersion); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateResidualRisksFile(path string, expectedVersion string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return validateResidualRisks(raw, expectedVersion)
}

func validateResidualRisks(raw []byte, expectedVersion string) error {
	if !risksFieldIsArray(raw) {
		return fmt.Errorf("risks array required")
	}

	var artifact residualRisks
	if err := decodeStrictJSON(raw, &artifact); err != nil {
		return err
	}
	if artifact.Schema != residualRisksSchema {
		return fmt.Errorf("schema = %q, want %q", artifact.Schema, residualRisksSchema)
	}
	if artifact.ReleaseVersion != expectedVersion {
		return fmt.Errorf("release_version = %q, want %q", artifact.ReleaseVersion, expectedVersion)
	}
	if artifact.Artifact != residualRisksArtifact {
		return fmt.Errorf("artifact = %q, want %q", artifact.Artifact, residualRisksArtifact)
	}

	seen := make(map[string]bool, len(artifact.Risks))
	for i, risk := range artifact.Risks {
		if err := validateResidualRisk(risk, i); err != nil {
			return err
		}
		if seen[risk.ID] {
			return fmt.Errorf("duplicate residual risk id %q", risk.ID)
		}
		seen[risk.ID] = true
	}
	return nil
}

func decodeStrictJSON(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("residual-risks.json must contain a single JSON document")
	}
	return nil
}

func risksFieldIsArray(raw []byte) bool {
	var root map[string]json.RawMessage
	dec := json.NewDecoder(bytes.NewReader(raw))
	if err := dec.Decode(&root); err != nil {
		return true
	}
	risksRaw, ok := root["risks"]
	if !ok {
		return false
	}
	trimmed := bytes.TrimSpace(risksRaw)
	return len(trimmed) > 0 && trimmed[0] == '['
}

func validateResidualRisk(risk residualRisk, index int) error {
	id := strings.TrimSpace(risk.ID)
	if id == "" {
		return fmt.Errorf("residual risk at index %d missing id", index)
	}
	severity := strings.ToLower(strings.TrimSpace(risk.Severity))
	if severity == "" {
		return fmt.Errorf("residual risk %s missing severity", id)
	}
	if !knownSeverity(severity) {
		return fmt.Errorf("residual risk %s has unknown severity %s", id, severity)
	}
	owner := strings.TrimSpace(risk.Owner)
	status := strings.TrimSpace(risk.Status)
	if severity == "medium" || severity == "high" || severity == "critical" {
		if missingOrUnknown(owner) || missingOrUnknown(status) {
			return fmt.Errorf(
				"%s residual risk %s requires known status and owner (owner=%s, status=%s)",
				severity,
				id,
				owner,
				status,
			)
		}
	}
	return nil
}

func knownSeverity(severity string) bool {
	switch severity {
	case "none", "low", "medium", "high", "critical":
		return true
	default:
		return false
	}
}

func missingOrUnknown(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "unknown", "unowned", "todo", "tbd":
		return true
	default:
		return false
	}
}
