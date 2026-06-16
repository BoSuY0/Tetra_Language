package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tetra_language/internal/outputformat"
)

type ecoMaterialization struct {
	Schema          string `json:"schema"`
	Target          string `json:"target"`
	PackagePath     string `json:"package_path"`
	MaterializedDir string `json:"materialized_dir"`
	TrustSnapshot   string `json:"trust_snapshot,omitempty"`
	LockSHA256      string `json:"lock_sha256,omitempty"`
}

func runEcoMaterialize(args []string, stdout io.Writer, stderr io.Writer) int {
	pkgPath, target, trustPath, outDir, metadataFormat, err := parseEcoMaterializeArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if err := unpackCapsule(pkgPath, outDir); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	capsulePath, err := findCapsulePath(outDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	manifest, err := parseCapsule(capsulePath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if target != "" && len(manifest.Targets) > 0 && !containsString(manifest.Targets, target) {
		fmt.Fprintf(stderr, "target mismatch for %s: does not support %s\n", manifest.ID, target)
		return 1
	}
	meta := ecoMaterialization{
		Schema:          ecoMaterializerSchemaV1,
		Target:          target,
		PackagePath:     filepath.Clean(pkgPath),
		MaterializedDir: filepath.Clean(outDir),
	}
	if trustPath != "" {
		raw, err := os.ReadFile(trustPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		var snapshot ecoTrustSnapshot
		if err := decodeEcoStructured(raw, outputformat.Auto, &snapshot); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if snapshot.Schema != "" && snapshot.Schema != ecoTrustSnapshotSchemaV1 {
			fmt.Fprintf(stderr, "unsupported trust snapshot schema %q\n", snapshot.Schema)
			return 1
		}
		meta.TrustSnapshot = filepath.Clean(trustPath)
		meta.LockSHA256 = snapshot.LockSHA256
	}
	if _, err := writeEcoStructuredFile(filepath.Join(outDir, "tetra.materialization.json"), metadataFormat, meta); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Materialized: %s\n", outDir)
	return 0
}

func parseEcoMaterializeArgs(args []string) (pkgPath string, target string, trustPath string, outDir string, metadataFormat string, err error) {
	outDir = "."
	metadataFormat = outputformat.JSON
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--target":
			i++
			if i >= len(args) {
				return "", "", "", "", "", fmt.Errorf("--target requires a value")
			}
			target = args[i]
		case "--trust":
			i++
			if i >= len(args) {
				return "", "", "", "", "", fmt.Errorf("--trust requires a value")
			}
			trustPath = args[i]
		case "-C", "--dir":
			i++
			if i >= len(args) {
				return "", "", "", "", "", fmt.Errorf("%s requires a value", args[i-1])
			}
			outDir = args[i]
		case "--metadata-format":
			i++
			if i >= len(args) {
				return "", "", "", "", "", fmt.Errorf("--metadata-format requires a value")
			}
			metadataFormat = args[i]
		default:
			if strings.HasPrefix(args[i], "-") {
				return "", "", "", "", "", fmt.Errorf("unknown option %s", args[i])
			}
			if pkgPath != "" {
				return "", "", "", "", "", fmt.Errorf("eco materialize requires one package path")
			}
			pkgPath = args[i]
		}
	}
	if pkgPath == "" {
		return "", "", "", "", "", fmt.Errorf("eco materialize requires one package path")
	}
	return pkgPath, target, trustPath, outDir, metadataFormat, nil
}
