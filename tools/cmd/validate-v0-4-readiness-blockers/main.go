package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const readinessBlockersSchema = "tetra.release.v0_4_0.readiness-blockers.v1"
const readinessBlockersArtifact = "readiness-blockers.json"
const defaultExpectedVersion = "v0.4.0"

type readinessBlockers struct {
	Schema         string             `json:"schema"`
	ReleaseVersion string             `json:"release_version"`
	Artifact       string             `json:"artifact"`
	SourceLog      string             `json:"source_log"`
	Blockers       []readinessBlocker `json:"blockers"`
}

type readinessBlocker struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Summary string `json:"summary"`
	Detail  string `json:"detail"`
}

func main() {
	var artifactPath string
	var reportDir string
	var expectedVersion string
	flag.StringVar(&artifactPath, "artifact", "", "path to readiness-blockers.json")
	flag.StringVar(&reportDir, "report-dir", "", "release gate report directory")
	flag.StringVar(&expectedVersion, "expected-version", defaultExpectedVersion, "expected release version")
	flag.Parse()

	if artifactPath == "" {
		fmt.Fprintln(os.Stderr, "error: --artifact is required")
		os.Exit(2)
	}
	if err := validateReadinessBlockersFile(artifactPath, reportDir, expectedVersion); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateReadinessBlockersFile(artifactPath, reportDir, expectedVersion string) error {
	raw, err := os.ReadFile(artifactPath)
	if err != nil {
		return err
	}
	if reportDir == "" {
		reportDir = filepath.Dir(filepath.Dir(artifactPath))
	}
	return validateReadinessBlockers(raw, reportDir, expectedVersion)
}

func validateReadinessBlockers(raw []byte, reportDir, expectedVersion string) error {
	var artifact readinessBlockers
	if err := decodeStrictJSON(raw, &artifact); err != nil {
		return err
	}
	if artifact.Schema != readinessBlockersSchema {
		return fmt.Errorf("schema = %q, want %q", artifact.Schema, readinessBlockersSchema)
	}
	if artifact.ReleaseVersion != expectedVersion {
		return fmt.Errorf("release_version = %q, want %q", artifact.ReleaseVersion, expectedVersion)
	}
	if artifact.Artifact != readinessBlockersArtifact {
		return fmt.Errorf("artifact = %q, want %q", artifact.Artifact, readinessBlockersArtifact)
	}
	if err := validateSourceLog(artifact.SourceLog, reportDir); err != nil {
		return err
	}
	if len(artifact.Blockers) == 0 {
		return fmt.Errorf("blockers must not be empty")
	}
	seen := make(map[string]bool, len(artifact.Blockers))
	for i, blocker := range artifact.Blockers {
		if err := validateBlocker(blocker, i); err != nil {
			return err
		}
		if seen[blocker.ID] {
			return fmt.Errorf("duplicate blocker id %q", blocker.ID)
		}
		seen[blocker.ID] = true
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
		return fmt.Errorf("readiness-blockers.json must contain a single JSON document")
	}
	return nil
}

func validateSourceLog(sourceLog string, reportDir string) error {
	if strings.TrimSpace(sourceLog) == "" {
		return fmt.Errorf("source_log is required")
	}
	if filepath.IsAbs(sourceLog) || strings.Contains(sourceLog, "..") || !strings.HasPrefix(filepath.ToSlash(sourceLog), "logs/") {
		return fmt.Errorf("unsafe source_log %q", sourceLog)
	}
	logPath := filepath.Join(reportDir, filepath.FromSlash(sourceLog))
	info, err := os.Stat(logPath)
	if err != nil {
		return fmt.Errorf("source_log %s is not readable: %w", sourceLog, err)
	}
	if info.IsDir() {
		return fmt.Errorf("source_log %s is a directory", sourceLog)
	}
	return nil
}

func validateBlocker(blocker readinessBlocker, index int) error {
	id := strings.TrimSpace(blocker.ID)
	if id == "" {
		return fmt.Errorf("blocker at index %d missing id", index)
	}
	if strings.TrimSpace(blocker.Status) != "blocked" {
		return fmt.Errorf("blocker %s status = %q, want blocked", id, blocker.Status)
	}
	if strings.TrimSpace(blocker.Summary) == "" {
		return fmt.Errorf("blocker %s missing summary", id)
	}
	if strings.TrimSpace(blocker.Detail) == "" {
		return fmt.Errorf("blocker %s missing detail", id)
	}
	return nil
}
