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
	Schema      string                   `json:"schema"`
	Compression string                   `json:"compression"`
	MTimeUnix   int64                    `json:"mtime_unix"`
	FileCount   int                      `json:"file_count"`
	Files       []ecoPackageMetadataFile `json:"files"`
}

type ecoPackageMetadataFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
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
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}
	manifestPath := filepath.Join(dir, "Tetra.capsule")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("missing Tetra.capsule")
		}
		return err
	}
	if err := validateManifestText(string(raw)); err != nil {
		return err
	}
	srcDir := filepath.Join(dir, "src")
	if info, err := os.Stat(srcDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("missing .tetra sources under src")
		}
		return err
	} else if !info.IsDir() {
		return fmt.Errorf("src is not a directory")
	}
	hasSource := false
	if err := filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".tetra" {
			hasSource = true
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if _, err := compiler.ParseFile(raw, filepath.ToSlash(path)); err != nil {
				return fmt.Errorf("%s: parse failed: %w", path, err)
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if !hasSource {
		return fmt.Errorf("missing .tetra sources under src")
	}
	if err := validatePackageMetadata(dir); err != nil {
		return err
	}
	return nil
}

func validateManifestText(text string) error {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return fmt.Errorf("manifest is empty")
	}
	if !strings.Contains(trimmed, "capsule ") {
		return fmt.Errorf("manifest missing capsule declaration")
	}
	if !hasManifestField(trimmed, "id") {
		return fmt.Errorf("manifest missing id")
	}
	if !hasManifestField(trimmed, "version") {
		return fmt.Errorf("manifest missing version")
	}
	if !hasManifestField(trimmed, "target") {
		return fmt.Errorf("manifest missing target")
	}
	return nil
}

func hasManifestField(text string, field string) bool {
	prefix := field + " "
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			return true
		}
	}
	return false
}

func validatePackageMetadata(dir string) error {
	const metadataFile = "tetra.package.json"
	const metadataSchema = "tetra.eco.package.v1"
	raw, err := os.ReadFile(filepath.Join(dir, metadataFile))
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
	if metadata.FileCount != len(metadata.Files) {
		return fmt.Errorf("package metadata file_count mismatch: expected %d, got %d", len(metadata.Files), metadata.FileCount)
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
		fileRaw, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(normalizedPath)))
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
	if _, ok := seen["Tetra.capsule"]; !ok {
		return fmt.Errorf("package metadata missing Tetra.capsule entry")
	}
	return nil
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
