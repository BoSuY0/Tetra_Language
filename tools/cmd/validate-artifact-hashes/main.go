package main

import (
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
	if manifestPath == "" {
		fmt.Fprintln(os.Stderr, "error: --manifest is required unless --write is set")
		os.Exit(2)
	}
	if err := validateHashManifest(manifestPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return err
	}
	if manifest.Schema != hashManifestSchema {
		return fmt.Errorf("invalid schema %q", manifest.Schema)
	}
	if len(manifest.Artifacts) == 0 {
		return fmt.Errorf("artifacts must not be empty")
	}
	root := filepath.Dir(manifestPath)
	seen := map[string]bool{}
	for _, expected := range manifest.Artifacts {
		if expected.Path == "" {
			return fmt.Errorf("artifact missing path")
		}
		if filepath.IsAbs(expected.Path) || strings.Contains(expected.Path, "..") {
			return fmt.Errorf("unsafe artifact path %s", expected.Path)
		}
		if seen[expected.Path] {
			return fmt.Errorf("duplicate artifact path %s", expected.Path)
		}
		seen[expected.Path] = true
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
