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
	"strings"

	ctarget "tetra_language/compiler/target"
)

const (
	materializationSchemaV1 = "tetra.eco.materialization.v1"
	sha256Prefix            = "sha256:"
)

type materializationReport struct {
	Schema          string  `json:"schema"`
	Target          *string `json:"target"`
	PackagePath     string  `json:"package_path"`
	MaterializedDir string  `json:"materialized_dir"`
	TrustSnapshot   *string `json:"trust_snapshot,omitempty"`
	LockSHA256      *string `json:"lock_sha256,omitempty"`
}

func main() {
	var materializationPath string
	flag.StringVar(&materializationPath, "materialization", "", "path to tetra.eco.materialization.v1 JSON report")
	flag.Parse()

	if materializationPath == "" {
		fmt.Fprintln(os.Stderr, "error: --materialization is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(materializationPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateEcoMaterialization(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateEcoMaterialization(raw []byte) error {
	var report materializationReport
	if err := decodeStrictJSON(raw, &report); err != nil {
		return err
	}
	if report.Schema == "" {
		return fmt.Errorf("schema is required")
	}
	if report.Schema != materializationSchemaV1 {
		return fmt.Errorf("unsupported materialization schema %q", report.Schema)
	}
	if report.Target == nil {
		return fmt.Errorf("target is required")
	}
	if err := validateMaterializationTarget(*report.Target); err != nil {
		return err
	}
	if err := validateMaterializationPath(report.PackagePath, "package_path"); err != nil {
		return err
	}
	if err := validateMaterializationPath(report.MaterializedDir, "materialized_dir"); err != nil {
		return err
	}
	if report.TrustSnapshot != nil {
		if err := validateMaterializationPath(*report.TrustSnapshot, "trust_snapshot"); err != nil {
			if *report.TrustSnapshot == "" {
				return fmt.Errorf("trust_snapshot is required when present")
			}
			return err
		}
	}
	if report.LockSHA256 != nil {
		if *report.LockSHA256 == "" {
			return fmt.Errorf("lock_sha256 is required when present")
		}
		if _, err := parseSHA256Hash(*report.LockSHA256); err != nil {
			return fmt.Errorf("invalid lock_sha256: %w", err)
		}
	}
	return nil
}

func decodeStrictJSON(raw []byte, out any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err != nil {
			return err
		}
		return fmt.Errorf("multiple JSON values")
	}
	return nil
}

func validateMaterializationTarget(target string) error {
	if target == "" {
		return nil
	}
	if !supportedTargets()[target] {
		return fmt.Errorf("unsupported target %s", target)
	}
	return nil
}

func validateMaterializationPath(path string, field string) error {
	if path == "" {
		return fmt.Errorf("%s is required", field)
	}
	clean := filepath.Clean(path)
	if clean == "." {
		return fmt.Errorf("%s is required", field)
	}
	if clean != path {
		return fmt.Errorf("%s path is not normalized: %s", field, path)
	}
	return nil
}

func supportedTargets() map[string]bool {
	out := map[string]bool{}
	for _, triple := range ctarget.SupportedTriples() {
		out[triple] = true
	}
	for _, triple := range ctarget.BuildOnlyTriples() {
		out[triple] = true
	}
	return out
}

func parseSHA256Hash(hash string) (string, error) {
	if !strings.HasPrefix(hash, sha256Prefix) {
		return "", fmt.Errorf("invalid sha256 hash %s", hash)
	}
	hexHash := strings.TrimPrefix(hash, sha256Prefix)
	if len(hexHash) != sha256.Size*2 {
		return "", fmt.Errorf("invalid sha256 hash %s", hash)
	}
	if _, err := hex.DecodeString(hexHash); err != nil {
		return "", fmt.Errorf("invalid sha256 hash %s", hash)
	}
	return hexHash, nil
}
