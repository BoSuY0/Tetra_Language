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

func main() {
	var path string
	flag.StringVar(&path, "report", "", "path to tetra features --format=json output")
	flag.Parse()
	if path == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateFeaturesReport(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateFeaturesReport(raw []byte) error {
	var report featuresReport
	if err := decodeStrictJSON(raw, &report); err != nil {
		return fmt.Errorf("invalid features JSON: %w", err)
	}
	if report.Schema != "tetra.features.v1" {
		return fmt.Errorf("features schema = %q, want tetra.features.v1", report.Schema)
	}
	if report.Version == "" {
		return fmt.Errorf("features version is required")
	}
	return validateFeatures(report.Features)
}

func decodeStrictJSON(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON value")
		}
		return err
	}
	return nil
}

func validateFeatures(features []featureEntry) error {
	if len(features) == 0 {
		return fmt.Errorf("features must not be empty")
	}
	allowedStatus := map[string]bool{
		"current":      true,
		"experimental": true,
		"planned":      true,
		"post-v1":      true,
	}
	requiredStatuses := []string{"current", "planned", "post-v1"}
	seenStatus := map[string]bool{}
	seenID := map[string]string{}
	for _, feature := range features {
		if feature.ID == "" {
			return fmt.Errorf("feature missing id")
		}
		if feature.Name == "" || feature.Scope == "" || feature.Stability == "" {
			return fmt.Errorf("feature %s missing name, scope, or stability", feature.ID)
		}
		if !allowedStatus[feature.Status] {
			return fmt.Errorf("feature %s invalid status %q", feature.ID, feature.Status)
		}
		if previousStatus, ok := seenID[feature.ID]; ok {
			return fmt.Errorf("duplicate feature %s (%s and %s)", feature.ID, previousStatus, feature.Status)
		}
		seenID[feature.ID] = feature.Status
		seenStatus[feature.Status] = true
		if feature.Status == "current" && feature.Since == "" {
			return fmt.Errorf("current feature %s missing since", feature.ID)
		}
		if err := validateFeatureDocs(feature); err != nil {
			return err
		}
	}
	for _, status := range requiredStatuses {
		if !seenStatus[status] {
			return fmt.Errorf("features missing %s status", status)
		}
	}
	return nil
}

func validateFeatureDocs(feature featureEntry) error {
	if len(feature.Docs) == 0 {
		return fmt.Errorf("feature %s missing docs", feature.ID)
	}
	seenDocs := map[string]bool{}
	for _, doc := range feature.Docs {
		docPath := filepath.ToSlash(doc)
		if doc == "" {
			return fmt.Errorf("feature %s has empty doc reference", feature.ID)
		}
		if filepath.IsAbs(doc) || strings.Contains(docPath, "..") {
			return fmt.Errorf("feature %s has unsafe doc reference %q", feature.ID, doc)
		}
		if !strings.HasPrefix(docPath, "docs/") || !strings.HasSuffix(docPath, ".md") {
			return fmt.Errorf("feature %s doc reference %q must point at docs/*.md", feature.ID, doc)
		}
		if seenDocs[docPath] {
			return fmt.Errorf("feature %s doc reference %q is duplicated", feature.ID, doc)
		}
		seenDocs[docPath] = true
		if _, err := statFromRepoRoot(docPath); err != nil {
			return fmt.Errorf("feature %s doc reference %q is not readable: %v", feature.ID, doc, err)
		}
	}
	return nil
}

func statFromRepoRoot(path string) (os.FileInfo, error) {
	root, err := repoRoot()
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("is a directory")
	}
	return info, nil
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if fileExists(filepath.Join(dir, "go.mod")) &&
			fileExists(filepath.Join(dir, "docs")) &&
			fileExists(filepath.Join(dir, "compiler")) {
			return dir, nil
		}
		next := filepath.Dir(dir)
		if next == dir {
			return "", fmt.Errorf("could not find repo root from %s", dir)
		}
		dir = next
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
