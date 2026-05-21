package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const hashManifestSchema = "tetra.release-artifact-hashes.v1alpha1"
const hashManifestArtifact = "tetra.release.v0_2_0.artifact-hashes.v1"

type hashManifest struct {
	Schema    string           `json:"schema"`
	Root      string           `json:"root"`
	Artifacts []hashedArtifact `json:"artifacts"`
}

type hashedArtifact struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Schema string `json:"schema,omitempty"`
}

func main() {
	var write bool
	var root string
	var out string
	var manifestPath string
	flag.BoolVar(&write, "write", false, "write a hash manifest")
	flag.StringVar(&root, "root", "", "artifact root to hash")
	flag.StringVar(&out, "out", "", "hash manifest output path")
	flag.StringVar(&manifestPath, "manifest", "", "hash manifest to validate")
	flag.Parse()

	if write {
		if root == "" || out == "" {
			fmt.Fprintln(os.Stderr, "error: --write requires --root and --out")
			os.Exit(2)
		}
		manifest, err := buildHashManifest(root, filepath.Base(out))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		raw, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		raw = append(raw, '\n')
		if err := os.WriteFile(out, raw, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	manifestPath, err := resolveHashManifestPath(manifestPath, root, out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := validateHashManifest(manifestPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func resolveHashManifestPath(manifestPath string, root string, out string) (string, error) {
	if manifestPath != "" {
		return manifestPath, nil
	}
	if root != "" && out != "" {
		return out, nil
	}
	return "", fmt.Errorf("error: --manifest is required unless --write is set; --root and --out may be used as the validation manifest path")
}

func buildHashManifest(root string, manifestName string) (hashManifest, error) {
	root = filepath.Clean(root)
	var artifacts []hashedArtifact
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == manifestName {
			return nil
		}
		artifact, err := hashFile(root, rel)
		if err != nil {
			return err
		}
		artifacts = append(artifacts, artifact)
		return nil
	})
	if err != nil {
		return hashManifest{}, err
	}
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	return hashManifest{
		Schema:    hashManifestSchema,
		Root:      ".",
		Artifacts: artifacts,
	}, nil
}

func validateHashManifest(manifestPath string) error {
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var manifest hashManifest
	if err := decodeStrictJSON(raw, &manifest); err != nil {
		return err
	}
	if manifest.Schema != hashManifestSchema {
		return fmt.Errorf("invalid schema %q", manifest.Schema)
	}
	if manifest.Root == "" {
		return fmt.Errorf("root must not be empty")
	}
	if filepath.IsAbs(manifest.Root) || strings.Contains(manifest.Root, "..") {
		return fmt.Errorf("unsafe root %q", manifest.Root)
	}
	if len(manifest.Artifacts) == 0 {
		return fmt.Errorf("artifacts must not be empty")
	}
	root := filepath.Join(filepath.Dir(manifestPath), filepath.FromSlash(manifest.Root))
	manifestRel, err := filepath.Rel(root, manifestPath)
	if err != nil {
		return err
	}
	manifestRel = filepath.ToSlash(manifestRel)
	seen := map[string]bool{}
	lastPath := ""
	for _, expected := range manifest.Artifacts {
		if expected.Path == "" {
			return fmt.Errorf("artifact missing path")
		}
		if filepath.IsAbs(expected.Path) || strings.Contains(expected.Path, "..") {
			return fmt.Errorf("unsafe artifact path %s", expected.Path)
		}
		if lastPath != "" && expected.Path < lastPath {
			return fmt.Errorf("artifacts must be sorted by path: %s appears before %s", expected.Path, lastPath)
		}
		lastPath = expected.Path
		if seen[expected.Path] {
			return fmt.Errorf("duplicate artifact path %s", expected.Path)
		}
		seen[expected.Path] = true
		if expected.Size < 0 {
			return fmt.Errorf("artifact %s has negative size", expected.Path)
		}
		if err := validateSHA256(expected.SHA256, expected.Path); err != nil {
			return err
		}
		actual, err := hashFile(root, expected.Path)
		if err != nil {
			return err
		}
		if actual.Size != expected.Size {
			return fmt.Errorf("size mismatch for %s: got %d want %d", expected.Path, actual.Size, expected.Size)
		}
		if actual.SHA256 != expected.SHA256 {
			return fmt.Errorf("sha256 mismatch for %s: got %s want %s", expected.Path, actual.SHA256, expected.SHA256)
		}
		if actual.Schema != expected.Schema {
			return fmt.Errorf("schema mismatch for %s: got %q want %q", expected.Path, actual.Schema, expected.Schema)
		}
	}
	actualPaths, err := listArtifactPaths(root, manifestRel)
	if err != nil {
		return err
	}
	for _, path := range actualPaths {
		if !seen[path] {
			return fmt.Errorf("unlisted artifact %s", path)
		}
	}
	return nil
}

func decodeStrictJSON(raw []byte, out any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	return dec.Decode(out)
}

func validateSHA256(value string, path string) error {
	if !strings.HasPrefix(value, "sha256:") {
		return fmt.Errorf("artifact %s has invalid sha256 format %q", path, value)
	}
	hexPart := strings.TrimPrefix(value, "sha256:")
	if len(hexPart) != 64 {
		return fmt.Errorf("artifact %s sha256 must contain 64 hex chars", path)
	}
	for _, ch := range hexPart {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return fmt.Errorf("artifact %s sha256 has non-hex characters", path)
		}
	}
	return nil
}

func hashFile(root string, rel string) (hashedArtifact, error) {
	path := filepath.Join(root, filepath.FromSlash(rel))
	file, err := os.Open(path)
	if err != nil {
		return hashedArtifact{}, err
	}
	defer file.Close()
	h := sha256.New()
	size, err := io.Copy(h, file)
	if err != nil {
		return hashedArtifact{}, err
	}
	return hashedArtifact{
		Path:   filepath.ToSlash(rel),
		SHA256: "sha256:" + hex.EncodeToString(h.Sum(nil)),
		Size:   size,
		Schema: detectJSONSchema(path),
	}, nil
}

func listArtifactPaths(root string, manifestName string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == manifestName {
			return nil
		}
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func detectJSONSchema(path string) string {
	if filepath.Ext(path) != ".json" {
		return ""
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var envelope struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return ""
	}
	return envelope.Schema
}
