package reportdir

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func PrepareFresh(repoRoot, reportDir string) (string, error) {
	reportPath, err := ValidateFresh(repoRoot, reportDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(reportPath, 0o755); err != nil {
		return "", fmt.Errorf("create fresh report directory %q: %w", reportDir, err)
	}
	return ValidateFresh(repoRoot, reportDir)
}

func ValidateFresh(repoRoot, reportDir string) (string, error) {
	root, err := cleanRepoRoot(repoRoot)
	if err != nil {
		return "", err
	}
	if reportDir == "" {
		return "", fmt.Errorf("empty report directory")
	}
	if filepath.IsAbs(reportDir) {
		return "", fmt.Errorf("absolute report directory is not allowed: %s", reportDir)
	}

	parts, err := reportDirParts(reportDir)
	if err != nil {
		return "", err
	}
	reportPath := filepath.Join(root, filepath.Join(parts...))
	if err := rejectSymlinkComponents(root, parts, reportDir); err != nil {
		return "", err
	}

	info, err := os.Lstat(reportPath)
	if os.IsNotExist(err) {
		return reportPath, nil
	}
	if err != nil {
		return "", fmt.Errorf("inspect report directory %q: %w", reportDir, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("symlink report directory is not allowed: %s", reportDir)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("non-directory report path is not allowed: %s", reportDir)
	}
	empty, err := isEmptyDir(reportPath)
	if err != nil {
		return "", fmt.Errorf("inspect report directory entries %q: %w", reportDir, err)
	}
	if !empty {
		return "", fmt.Errorf("non-empty report directory is not allowed: %s", reportDir)
	}
	return reportPath, nil
}

func cleanRepoRoot(repoRoot string) (string, error) {
	if repoRoot == "" {
		return "", fmt.Errorf("empty repo root")
	}
	root, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", fmt.Errorf("resolve repo root %q: %w", repoRoot, err)
	}
	root = filepath.Clean(root)
	info, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("inspect repo root %q: %w", repoRoot, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("repo root is not a directory: %s", repoRoot)
	}
	return root, nil
}

func reportDirParts(reportDir string) ([]string, error) {
	rawParts := strings.Split(filepath.ToSlash(reportDir), "/")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			return nil, fmt.Errorf(
				"parent traversal is not allowed in report directory: %s",
				reportDir,
			)
		}
		if len(parts) == 0 && strings.HasPrefix(part, "-") {
			return nil, fmt.Errorf("dash-prefixed report directory is not allowed: %s", reportDir)
		}
		parts = append(parts, part)
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("repo root is not a report directory")
	}
	return parts, nil
}

func rejectSymlinkComponents(root string, parts []string, reportDir string) error {
	current := root
	for _, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspect report directory component %q: %w", reportDir, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink report directory component is not allowed: %s", reportDir)
		}
	}
	return nil
}

func isEmptyDir(path string) (bool, error) {
	dir, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer dir.Close()
	_, err = dir.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return false, nil
}
