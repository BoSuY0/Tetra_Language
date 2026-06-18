package artifacts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const HashManifestName = "artifact-hashes.json"
const HashManifestSchema = "tetra.release-artifact-hashes.v1alpha1"

type CommandPlan struct {
	Name string
	Args []string
}

type HashCommandPlan struct {
	ManifestPath string
	Write        CommandPlan
	Validate     CommandPlan
}

func ValidateRequiredReports(reportRoot string, reportPaths []string) ([]string, error) {
	validated := make([]string, 0, len(reportPaths))
	for _, reportPath := range reportPaths {
		path, err := ValidateRequiredReport(reportRoot, reportPath)
		if err != nil {
			return nil, err
		}
		validated = append(validated, path)
	}
	return validated, nil
}

func ValidateRequiredReport(reportRoot, reportPath string) (string, error) {
	root, err := cleanReportRoot(reportRoot)
	if err != nil {
		return "", err
	}
	parts, err := reportPathParts(reportPath)
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, filepath.Join(parts...))
	if err := rejectSymlinkComponents(root, parts, reportPath); err != nil {
		return "", err
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("missing required report file: %s", reportPath)
	}
	if err != nil {
		return "", fmt.Errorf("inspect required report %q: %w", reportPath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("symlink required report file is not allowed: %s", reportPath)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("required report is not a regular file: %s", reportPath)
	}
	if info.Size() == 0 {
		return "", fmt.Errorf("empty file is not allowed for required report: %s", reportPath)
	}
	return path, nil
}

func NewHashCommandPlan(reportRoot string) (HashCommandPlan, error) {
	if strings.TrimSpace(reportRoot) == "" {
		return HashCommandPlan{}, fmt.Errorf("empty report root")
	}
	root := filepath.Clean(reportRoot)
	manifestPath := filepath.Join(root, HashManifestName)
	return HashCommandPlan{
		ManifestPath: manifestPath,
		Write: CommandPlan{
			Name: "artifact-hashes-write",
			Args: []string{
				"go", "run", "./tools/cmd/validate-artifact-hashes",
				"--write", "--root", root,
				"--out", manifestPath,
			},
		},
		Validate: CommandPlan{
			Name: "artifact-hashes-validate",
			Args: []string{
				"go", "run", "./tools/cmd/validate-artifact-hashes",
				"--manifest", manifestPath,
			},
		},
	}, nil
}

func cleanReportRoot(reportRoot string) (string, error) {
	if reportRoot == "" {
		return "", fmt.Errorf("empty report root")
	}
	root, err := filepath.Abs(reportRoot)
	if err != nil {
		return "", fmt.Errorf("resolve report root %q: %w", reportRoot, err)
	}
	root = filepath.Clean(root)
	info, err := os.Lstat(root)
	if err != nil {
		return "", fmt.Errorf("inspect report root %q: %w", reportRoot, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("symlink report root is not allowed: %s", reportRoot)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("report root is not a directory: %s", reportRoot)
	}
	return root, nil
}

func reportPathParts(reportPath string) ([]string, error) {
	if reportPath == "" {
		return nil, fmt.Errorf("empty required report path")
	}
	if filepath.IsAbs(reportPath) {
		return nil, fmt.Errorf("absolute required report path is not allowed: %s", reportPath)
	}
	rawParts := strings.Split(filepath.ToSlash(reportPath), "/")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			return nil, fmt.Errorf(
				"parent traversal is not allowed in required report path: %s",
				reportPath,
			)
		}
		parts = append(parts, part)
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty required report path")
	}
	return parts, nil
}

func rejectSymlinkComponents(root string, parts []string, reportPath string) error {
	current := root
	for i, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspect required report component %q: %w", reportPath, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			if i == len(parts)-1 {
				return fmt.Errorf("symlink required report file is not allowed: %s", reportPath)
			}
			return fmt.Errorf(
				"symlink required report path component is not allowed: %s",
				reportPath,
			)
		}
	}
	return nil
}
