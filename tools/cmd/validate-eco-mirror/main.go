package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	ctarget "tetra_language/compiler/target"
	"tetra_language/tools/internal/reportdecode"
)

const (
	mirrorSchemaV1 = "tetra.eco.mirror.v1"
	sha256Prefix   = "sha256:"
)

type mirrorReport struct {
	Schema              string `json:"schema"`
	MirroredUnix        int64  `json:"mirrored_at_unix"`
	SourceStore         string `json:"source_store"`
	DestinationStore    string `json:"destination_store"`
	ID                  string `json:"id"`
	Version             string `json:"version"`
	Target              string `json:"target"`
	Channel             string `json:"channel"`
	Hub                 string `json:"hub"`
	PackagePath         string `json:"package_path"`
	PackageSHA256       string `json:"package_sha256"`
	MetadataPath        string `json:"metadata_path"`
	MetadataSHA256      string `json:"metadata_sha256"`
	TrustSnapshotPath   string `json:"trust_snapshot_path,omitempty"`
	TrustSnapshotSHA256 string `json:"trust_snapshot_sha256,omitempty"`
}

func main() {
	var mirrorPath string
	var reportFormat string
	flag.StringVar(&mirrorPath, "mirror", "", "path to tetra.eco.mirror.v1 JSON report")
	flag.StringVar(&reportFormat, "format", "auto", "report format: auto, json, or toon")
	flag.Parse()

	if mirrorPath == "" {
		fmt.Fprintln(os.Stderr, "error: --mirror is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(mirrorPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateEcoMirrorFormat(raw, reportFormat); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateEcoMirror(raw []byte) error {
	return validateEcoMirrorFormat(raw, "auto")
}

func validateEcoMirrorFormat(raw []byte, format string) error {
	var report mirrorReport
	if err := reportdecode.DecodeStrictFormat(raw, format, &report); err != nil {
		return err
	}
	if report.Schema == "" {
		return fmt.Errorf("schema is required")
	}
	if report.Schema != mirrorSchemaV1 {
		return fmt.Errorf("unsupported mirror schema %q", report.Schema)
	}
	if report.MirroredUnix < 0 {
		return fmt.Errorf("mirrored_at_unix must not be negative")
	}
	if err := validateStorePath(report.SourceStore, "source_store"); err != nil {
		return err
	}
	if err := validateStorePath(report.DestinationStore, "destination_store"); err != nil {
		return err
	}
	if report.ID == "" {
		return fmt.Errorf("id is required")
	}
	if report.Version == "" {
		return fmt.Errorf("version is required")
	}
	if err := validateMirrorTarget(report.Target); err != nil {
		return err
	}
	if !isSupportedMirrorChannel(report.Channel) {
		return fmt.Errorf("unsupported channel %q", report.Channel)
	}
	if report.Hub == "" {
		return fmt.Errorf("hub is required")
	}
	if err := validateMirrorRelativePath(report.PackagePath, "package_path"); err != nil {
		return err
	}
	if err := validateMirrorRelativePath(report.MetadataPath, "metadata_path"); err != nil {
		return err
	}
	if _, err := parseSHA256Hash(report.PackageSHA256); err != nil {
		return fmt.Errorf("invalid package_sha256: %w", err)
	}
	if _, err := parseSHA256Hash(report.MetadataSHA256); err != nil {
		return fmt.Errorf("invalid metadata_sha256: %w", err)
	}

	expectedBase := filepath.ToSlash(
		filepath.Join("packages", capsuleIDDirectory(report.ID), report.Version, report.Target),
	)
	if report.PackagePath != filepath.ToSlash(filepath.Join(expectedBase, "package.todex")) {
		return fmt.Errorf(
			"package_path mismatch: metadata has %s, expected %s",
			report.PackagePath,
			filepath.ToSlash(filepath.Join(expectedBase, "package.todex")),
		)
	}
	if report.MetadataPath != filepath.ToSlash(filepath.Join(expectedBase, "metadata.json")) {
		return fmt.Errorf(
			"metadata_path mismatch: metadata has %s, expected %s",
			report.MetadataPath,
			filepath.ToSlash(filepath.Join(expectedBase, "metadata.json")),
		)
	}
	if report.TrustSnapshotPath != "" || report.TrustSnapshotSHA256 != "" {
		if err := validateMirrorRelativePath(
			report.TrustSnapshotPath,
			"trust_snapshot_path",
		); err != nil {
			return err
		}
		if _, err := parseSHA256Hash(report.TrustSnapshotSHA256); err != nil {
			return fmt.Errorf("invalid trust_snapshot_sha256: %w", err)
		}
		expectedTrustPath := filepath.ToSlash(filepath.Join(expectedBase, "trust.snapshot.json"))
		if report.TrustSnapshotPath != expectedTrustPath {
			return fmt.Errorf(
				"trust_snapshot_path mismatch: metadata has %s, expected %s",
				report.TrustSnapshotPath,
				expectedTrustPath,
			)
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

func validateStorePath(path string, field string) error {
	if path == "" {
		return fmt.Errorf("%s is required", field)
	}
	if field == "source_store" && isHTTPStoreURL(path) {
		return nil
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

func isHTTPStoreURL(path string) bool {
	parsed, err := url.Parse(path)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	if parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	return strings.TrimRight(parsed.String(), "/") == strings.TrimRight(path, "/")
}

func validateMirrorRelativePath(path string, field string) error {
	if path == "" {
		return fmt.Errorf("%s is required", field)
	}
	if strings.Contains(path, "\\") {
		return fmt.Errorf("unsafe %s path %s", field, path)
	}
	clean := filepath.Clean(path)
	if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return fmt.Errorf("unsafe %s path %s", field, path)
	}
	if filepath.ToSlash(clean) != path {
		return fmt.Errorf("%s path is not normalized: %s", field, path)
	}
	return nil
}

func validateMirrorTarget(target string) error {
	if target == "" {
		return fmt.Errorf("target is required")
	}
	for _, triple := range ctarget.SupportedTriples() {
		if target == triple {
			return nil
		}
	}
	for _, triple := range ctarget.BuildOnlyTriples() {
		if target == triple {
			return nil
		}
	}
	return fmt.Errorf("unsupported target %s", target)
}

func isSupportedMirrorChannel(channel string) bool {
	switch channel {
	case "beta", "stable":
		return true
	default:
		return false
	}
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

func capsuleIDDirectory(id string) string {
	s := strings.TrimPrefix(id, "tetra://")
	if s == "" {
		s = "unknown"
	}
	var b strings.Builder
	b.WriteString("tetra_")
	for _, ch := range s {
		switch {
		case ch >= 'a' && ch <= 'z':
			b.WriteRune(ch)
		case ch >= 'A' && ch <= 'Z':
			b.WriteRune(ch + ('a' - 'A'))
		case ch >= '0' && ch <= '9':
			b.WriteRune(ch)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
