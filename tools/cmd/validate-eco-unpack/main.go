package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tetra_language/compiler"
)

type ecoPackageMetadata struct {
	Schema           string                   `json:"schema"`
	Compression      string                   `json:"compression"`
	MTimeUnix        int64                    `json:"mtime_unix"`
	Reproducible     bool                     `json:"reproducible,omitempty"`
	BuildInputsSHA   string                   `json:"build_inputs_sha256,omitempty"`
	ManifestSchema   string                   `json:"manifest_schema,omitempty"`
	PermissionsModel string                   `json:"permissions_model,omitempty"`
	FileCount        int                      `json:"file_count"`
	Files            []ecoPackageMetadataFile `json:"files"`
}

type ecoPackageMetadataFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type capsuleUnpackManifest struct {
	SourceRoots []string
}

func main() {
	var dir string
	flag.StringVar(&dir, "dir", "", "path to unpacked Eco project bundle")
	flag.Parse()

	if dir == "" {
		fmt.Fprintln(os.Stderr, "error: --dir is required")
		os.Exit(2)
	}
	if err := validateEcoUnpack(dir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateEcoUnpack(dir string) error {
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symlink", dir)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}
	manifestPath, err := findCapsuleManifest(dir)
	if err != nil {
		return err
	}
	raw, err := readRegularUnpackedFile(manifestPath, "manifest")
	if err != nil {
		return err
	}
	manifest, err := validateManifestText(string(raw))
	if err != nil {
		return err
	}
	if len(manifest.SourceRoots) == 0 {
		manifest.SourceRoots = []string{"src"}
	}
	hasSource, err := validateSourceRoots(dir, manifest.SourceRoots)
	if err != nil {
		return err
	}
	if !hasSource {
		return fmt.Errorf("missing T4 sources under %s", strings.Join(manifest.SourceRoots, ", "))
	}
	if err := validatePackageMetadata(dir); err != nil {
		return err
	}
	return nil
}

func validateSourceRoots(dir string, roots []string) (bool, error) {
	hasSource := false
	for _, root := range roots {
		sourceDir := filepath.Join(dir, filepath.FromSlash(root))
		if info, err := os.Lstat(sourceDir); err != nil {
			if os.IsNotExist(err) {
				return false, fmt.Errorf("missing T4 sources under %s", root)
			}
			return false, err
		} else if info.Mode()&os.ModeSymlink != 0 {
			return false, fmt.Errorf("source root %s is a symlink", root)
		} else if !info.IsDir() {
			return false, fmt.Errorf("%s is not a directory", root)
		}
		if err := filepath.WalkDir(sourceDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.Type()&os.ModeSymlink != 0 {
				rel, relErr := filepath.Rel(dir, path)
				if relErr != nil {
					rel = path
				}
				return fmt.Errorf("unpacked package contains symlink %s", filepath.ToSlash(rel))
			}
			if d.IsDir() {
				return nil
			}
			if compiler.IsSourceFile(path) {
				hasSource = true
				raw, err := readRegularUnpackedFile(path, "source file")
				if err != nil {
					return err
				}
				if _, err := compiler.ParseFile(raw, filepath.ToSlash(path)); err != nil {
					return fmt.Errorf("%s: parse failed: %w", path, err)
				}
			}
			return nil
		}); err != nil {
			return false, err
		}
	}
	return hasSource, nil
}

func findCapsuleManifest(dir string) (string, error) {
	for _, name := range []string{compiler.CapsuleFileName, compiler.LegacyCapsuleFileName} {
		path := filepath.Join(dir, name)
		if info, err := os.Lstat(path); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return "", fmt.Errorf("%s is a symlink", name)
			}
			if info.IsDir() {
				return "", fmt.Errorf("%s is a directory", name)
			}
			return path, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", fmt.Errorf(
		"missing %s (or legacy %s)",
		compiler.CapsuleFileName,
		compiler.LegacyCapsuleFileName,
	)
}

func validateManifestText(text string) (capsuleUnpackManifest, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return capsuleUnpackManifest{}, fmt.Errorf("manifest is empty")
	}
	if !strings.Contains(trimmed, "capsule ") {
		return capsuleUnpackManifest{}, fmt.Errorf("manifest missing capsule declaration")
	}
	var (
		manifest capsuleUnpackManifest
		hasID    bool
		hasVer   bool
		hasTgt   bool
		section  string
	)
	for _, line := range strings.Split(trimmed, "\n") {
		content := strings.TrimSpace(line)
		if content == "" || strings.HasPrefix(content, "//") || strings.HasPrefix(content, "#") {
			continue
		}
		if nextSection, ok := unpackManifestSectionHeader(content); ok {
			section = nextSection
			continue
		}
		switch {
		case strings.HasPrefix(content, "id "):
			section = ""
			hasID = true
		case strings.HasPrefix(content, "version "):
			section = ""
			hasVer = true
		case strings.HasPrefix(content, "target "):
			section = ""
			hasTgt = true
		case strings.HasPrefix(content, "source "):
			section = ""
			value := strings.TrimSpace(strings.TrimPrefix(content, "source "))
			manifest.SourceRoots = appendUnpackSourceRoot(manifest.SourceRoots, value)
		case section == "targets":
			hasTgt = true
		case section == "sources":
			manifest.SourceRoots = appendUnpackSourceRoot(manifest.SourceRoots, content)
		}
	}
	if !hasID {
		return capsuleUnpackManifest{}, fmt.Errorf("manifest missing id")
	}
	if !hasVer {
		return capsuleUnpackManifest{}, fmt.Errorf("manifest missing version")
	}
	if !hasTgt {
		return capsuleUnpackManifest{}, fmt.Errorf("manifest missing target")
	}
	return manifest, nil
}

func unpackManifestSectionHeader(content string) (string, bool) {
	switch strings.TrimSuffix(content, ":") {
	case "sources":
		return "sources", true
	case "targets":
		return "targets", true
	case "deps", "allow", "policy", "artifacts":
		return "other", true
	default:
		return "", false
	}
}

func appendUnpackSourceRoot(roots []string, value string) []string {
	value = strings.Trim(value, `"`)
	clean := filepath.ToSlash(filepath.Clean(value))
	if clean == "." || clean == "" || clean == ".." || strings.HasPrefix(clean, "../") ||
		filepath.IsAbs(clean) {
		return roots
	}
	for _, root := range roots {
		if root == clean {
			return roots
		}
	}
	return append(roots, clean)
}

func validatePackageMetadata(dir string) error {
	const metadataFile = "tetra.package.json"
	const metadataSchema = "tetra.eco.package.v1"
	raw, err := readRegularUnpackedFile(filepath.Join(dir, metadataFile), metadataFile)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("missing tetra.package.json")
		}
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var metadata ecoPackageMetadata
	if err := decoder.Decode(&metadata); err != nil {
		return fmt.Errorf("invalid tetra.package.json: %w", err)
	}
	if metadata.Schema != metadataSchema {
		return fmt.Errorf("unsupported package metadata schema %q", metadata.Schema)
	}
	if metadata.Compression != "gzip" {
		return fmt.Errorf("package metadata compression must be gzip")
	}
	if metadata.MTimeUnix != 0 {
		return fmt.Errorf("package metadata mtime_unix must be 0")
	}
	if metadata.ManifestSchema != "" && metadata.ManifestSchema != "tetra.capsule.v1" {
		return fmt.Errorf(
			"unsupported package metadata manifest_schema %q",
			metadata.ManifestSchema,
		)
	}
	if metadata.PermissionsModel != "" && metadata.PermissionsModel != "tetra.eco.permissions.v1" {
		return fmt.Errorf(
			"unsupported package metadata permissions_model %q",
			metadata.PermissionsModel,
		)
	}
	if metadata.BuildInputsSHA != "" {
		if _, err := parseSHA256Hash(metadata.BuildInputsSHA); err != nil {
			return fmt.Errorf("package metadata build_inputs_sha256: %w", err)
		}
	}
	if metadata.FileCount != len(metadata.Files) {
		return fmt.Errorf(
			"package metadata file_count mismatch: expected %d, got %d",
			len(metadata.Files),
			metadata.FileCount,
		)
	}
	if metadata.FileCount <= 0 {
		return fmt.Errorf("package metadata file_count must be positive")
	}
	seen := map[string]struct{}{}
	lastPath := ""
	for _, entry := range metadata.Files {
		if entry.Path == "" {
			return fmt.Errorf("package metadata has empty path")
		}
		if entry.Size < 0 {
			return fmt.Errorf("package metadata has negative size for %s", entry.Path)
		}
		cleanPath := filepath.Clean(entry.Path)
		if cleanPath == "." || strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) {
			return fmt.Errorf("package metadata has unsafe path %s", entry.Path)
		}
		normalizedPath := filepath.ToSlash(cleanPath)
		if normalizedPath != entry.Path {
			return fmt.Errorf("package metadata path %s is not normalized", entry.Path)
		}
		if normalizedPath == metadataFile {
			return fmt.Errorf("package metadata must not self-reference %s", metadataFile)
		}
		if normalizedPath <= lastPath {
			return fmt.Errorf("package metadata files must be strictly sorted by path")
		}
		lastPath = normalizedPath
		if _, exists := seen[normalizedPath]; exists {
			return fmt.Errorf("package metadata has duplicate file path %s", normalizedPath)
		}
		seen[normalizedPath] = struct{}{}
		hashHex, err := parseSHA256Hash(entry.SHA256)
		if err != nil {
			return fmt.Errorf("package metadata %s: %w", normalizedPath, err)
		}
		fileRaw, err := readRegularUnpackedFile(
			filepath.Join(dir, filepath.FromSlash(normalizedPath)),
			normalizedPath,
		)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("package metadata references missing file %s", normalizedPath)
			}
			return err
		}
		if int64(len(fileRaw)) != entry.Size {
			return fmt.Errorf("metadata size mismatch for %s", normalizedPath)
		}
		sum := sha256.Sum256(fileRaw)
		actual := hex.EncodeToString(sum[:])
		if actual != hashHex {
			return fmt.Errorf("metadata hash mismatch for %s", normalizedPath)
		}
	}
	if _, ok := seen[compiler.CapsuleFileName]; !ok {
		if _, legacyOK := seen[compiler.LegacyCapsuleFileName]; !legacyOK {
			return fmt.Errorf("package metadata missing %s entry", compiler.CapsuleFileName)
		}
	}
	return nil
}

func readRegularUnpackedFile(path string, label string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%s is a symlink: %s", label, path)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file: %s", label, path)
	}
	return os.ReadFile(path)
}

func parseSHA256Hash(hash string) (string, error) {
	const prefix = "sha256:"
	if !strings.HasPrefix(hash, prefix) {
		return "", fmt.Errorf("invalid sha256 hash %s", hash)
	}
	hexHash := strings.TrimPrefix(hash, prefix)
	if len(hexHash) != sha256.Size*2 {
		return "", fmt.Errorf("invalid sha256 hash %s", hash)
	}
	if _, err := hex.DecodeString(hexHash); err != nil {
		return "", fmt.Errorf("invalid sha256 hash %s", hash)
	}
	return hexHash, nil
}
