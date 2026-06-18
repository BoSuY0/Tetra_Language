package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"tetra_language/tools/validators/surface"
)

type blockReportValidationOptions struct {
	SameCommit string
}

func main() {
	reportPath := flag.String(
		"report",
		"",
		"path to tetra.surface.runtime.v1 report with block_system evidence",
	)
	sameCommit := flag.String(
		"same-commit",
		"",
		"require the report to validate at this git commit",
	)
	flag.Parse()
	if *reportPath == "" {
		fmt.Fprintln(os.Stderr, "error: --report is required")
		os.Exit(2)
	}
	if err := validateSurfaceBlockReportWithOptions(
		*reportPath,
		blockReportValidationOptions{SameCommit: *sameCommit},
	); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateSurfaceBlockReport(path string) error {
	return validateSurfaceBlockReportWithOptions(path, blockReportValidationOptions{})
}

func validateSurfaceBlockReportWithOptions(
	path string,
	options blockReportValidationOptions,
) error {
	if err := validateReportPathSafety(path); err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := surface.ValidateReport(raw); err != nil {
		return err
	}
	var report surface.Report
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		return err
	}
	if report.BlockSystem == nil {
		return fmt.Errorf("surface Block report requires block_system evidence")
	}
	if report.BlockSystem.Schema != "tetra.surface.block-system.v1" {
		return fmt.Errorf(
			"block_system schema is %q, want tetra.surface.block-system.v1",
			report.BlockSystem.Schema,
		)
	}
	if err := validateBlockReportArtifactLocality(
		path,
		report.ArtifactScan,
		report.Artifacts,
	); err != nil {
		return err
	}
	if err := validateBlockReportArtifactFiles(filepath.Dir(path), report.Artifacts); err != nil {
		return err
	}
	if strings.TrimSpace(options.SameCommit) != "" {
		actual, err := currentGitCommit()
		if err != nil {
			return err
		}
		if err := validateSameCommit(options.SameCommit, actual); err != nil {
			return err
		}
	}
	return nil
}

func validateReportPathSafety(reportPath string) error {
	if strings.TrimSpace(reportPath) == "" {
		return fmt.Errorf("report path is required")
	}
	abs, err := filepath.Abs(reportPath)
	if err != nil {
		return err
	}
	if err := rejectSymlinkPath(abs); err != nil {
		return err
	}
	return rejectSymlinkPath(filepath.Dir(abs))
}

func rejectSymlinkPath(path string) error {
	clean := filepath.Clean(path)
	for {
		info, err := os.Lstat(clean)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("report path %s uses symlink component %s", path, clean)
		}
		parent := filepath.Dir(clean)
		if parent == clean {
			return nil
		}
		clean = parent
	}
}

func validateBlockReportArtifactLocality(
	reportPath string,
	scan surface.ArtifactScanReport,
	artifacts []surface.ArtifactReport,
) error {
	reportDir, err := filepath.Abs(filepath.Dir(reportPath))
	if err != nil {
		return err
	}
	if strings.TrimSpace(scan.Root) != "" {
		root, err := artifactPathAbs(scan.Root, reportDir)
		if err != nil {
			return err
		}
		if !pathWithin(root, reportDir) {
			return fmt.Errorf(
				"artifact_scan.root %s is outside report dir %s",
				scan.Root,
				reportDir,
			)
		}
	}
	for _, artifact := range artifacts {
		path, err := artifactPathAbs(artifact.Path, reportDir)
		if err != nil {
			return err
		}
		if !pathWithin(path, reportDir) {
			return fmt.Errorf(
				"artifact %s path %s is outside report dir %s",
				artifact.Kind,
				artifact.Path,
				reportDir,
			)
		}
	}
	return nil
}

func validateBlockReportArtifactFiles(reportDir string, artifacts []surface.ArtifactReport) error {
	absReportDir, err := filepath.Abs(reportDir)
	if err != nil {
		return err
	}
	for _, artifact := range artifacts {
		path, err := artifactPathAbs(artifact.Path, absReportDir)
		if err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf(
				"artifact %s path %s cannot be read: %w",
				artifact.Kind,
				artifact.Path,
				err,
			)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("artifact %s path %s is a symlink", artifact.Kind, artifact.Path)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf(
				"artifact %s path %s cannot be read: %w",
				artifact.Kind,
				artifact.Path,
				err,
			)
		}
		if int64(len(raw)) != artifact.Size {
			return fmt.Errorf(
				"artifact %s size = %d, want %d",
				artifact.Kind,
				len(raw),
				artifact.Size,
			)
		}
		sum := sha256.Sum256(raw)
		want := strings.TrimSpace(artifact.SHA256)
		got := fmt.Sprintf("sha256:%x", sum)
		if want != got {
			return fmt.Errorf("artifact %s sha256 = %s, want %s", artifact.Kind, got, want)
		}
	}
	return nil
}

func artifactPathAbs(path string, reportDir string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("artifact path is required")
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	if _, err := os.Stat(path); err == nil {
		return filepath.Abs(path)
	}
	return filepath.Abs(filepath.Join(reportDir, path))
}

func pathWithin(path string, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." ||
		(rel != "" && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != "..")
}

func validateSameCommit(expected string, actual string) error {
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)
	if expected == "" {
		return nil
	}
	if actual == "" {
		return fmt.Errorf("same-commit validation requires current git commit evidence")
	}
	if expected == actual || strings.HasPrefix(actual, expected) ||
		strings.HasPrefix(expected, actual) {
		return nil
	}
	return fmt.Errorf("same-commit mismatch: expected %s, got %s", expected, actual)
}

func currentGitCommit() (string, error) {
	raw, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("read current git commit: %w", err)
	}
	return strings.TrimSpace(string(raw)), nil
}
