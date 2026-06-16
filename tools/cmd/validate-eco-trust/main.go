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
	"strings"

	"tetra_language/tools/internal/reportdecode"
)

const (
	trustSnapshotSchemaV1 = "tetra.eco.trust-snapshot.v1"
	sha256Prefix          = "sha256:"
)

type trustSnapshotReport struct {
	Schema        string              `json:"schema"`
	GeneratedUnix *int64              `json:"generated_at_unix"`
	LockSHA256    string              `json:"lock_sha256,omitempty"`
	VaultSHA256   string              `json:"vault_sha256,omitempty"`
	RecordCount   *int                `json:"record_count"`
	CapsulesRaw   json.RawMessage     `json:"capsules"`
	Capsules      []trustSnapshotItem `json:"-"`
}

type trustSnapshotItem struct {
	ID           string   `json:"id"`
	Version      string   `json:"version"`
	Permissions  []string `json:"permissions,omitempty"`
	TrustTier    string   `json:"trust_tier"`
	TrustScore   *int     `json:"trust_score"`
	TrustReasons []string `json:"trust_reasons,omitempty"`
}

var knownCapsulePermissions = map[string]string{
	"actors":                "actors",
	"alloc":                 "alloc",
	"cap.io":                "io",
	"cap.mem":               "mem",
	"capability":            "capability",
	"control":               "control",
	"fs.read":               "fs.read",
	"fs.readWrite.userData": "fs.readWrite.userData",
	"fs.write":              "fs.write",
	"io":                    "io",
	"io.read":               "io",
	"io.write":              "io",
	"islands":               "islands",
	"link":                  "link",
	"mem":                   "mem",
	"mem.read":              "mem",
	"mem.write":             "mem",
	"mmio":                  "mmio",
	"runtime":               "runtime",
	"runtime.exec":          "runtime",
	"ui":                    "ui",
}

func main() {
	var trustPath string
	var reportFormat string
	flag.StringVar(&trustPath, "trust", "", "path to tetra.eco.trust-snapshot.v1 JSON report")
	flag.StringVar(&reportFormat, "format", "auto", "report format: auto, json, or toon")
	flag.Parse()

	if trustPath == "" {
		fmt.Fprintln(os.Stderr, "error: --trust is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(trustPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateEcoTrustSnapshotFormat(raw, reportFormat); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateEcoTrustSnapshot(raw []byte) error {
	return validateEcoTrustSnapshotFormat(raw, "auto")
}

func validateEcoTrustSnapshotFormat(raw []byte, format string) error {
	var report trustSnapshotReport
	if err := reportdecode.DecodeStrictFormat(raw, format, &report); err != nil {
		return err
	}
	if report.Schema == "" {
		return fmt.Errorf("schema is required")
	}
	if report.Schema != trustSnapshotSchemaV1 {
		return fmt.Errorf("unsupported trust snapshot schema %q", report.Schema)
	}
	if report.GeneratedUnix == nil {
		return fmt.Errorf("generated_at_unix is required")
	}
	if *report.GeneratedUnix < 0 {
		return fmt.Errorf("generated_at_unix must not be negative")
	}
	if report.LockSHA256 != "" {
		if _, err := parseSHA256Hash(report.LockSHA256); err != nil {
			return fmt.Errorf("invalid lock_sha256: %w", err)
		}
	}
	if report.VaultSHA256 != "" {
		if _, err := parseSHA256Hash(report.VaultSHA256); err != nil {
			return fmt.Errorf("invalid vault_sha256: %w", err)
		}
	}
	if report.RecordCount == nil {
		return fmt.Errorf("record_count is required")
	}
	if *report.RecordCount < 0 {
		return fmt.Errorf("record_count must not be negative")
	}
	if err := unmarshalRequiredArray(report.CapsulesRaw, "capsules", &report.Capsules); err != nil {
		return err
	}
	if *report.RecordCount != len(report.Capsules) {
		return fmt.Errorf("record_count mismatch: got %d, capsules has %d", *report.RecordCount, len(report.Capsules))
	}
	if len(report.Capsules) == 0 {
		return fmt.Errorf("capsules must not be empty")
	}
	return validateTrustSnapshotCapsules(report.Capsules)
}

func unmarshalRequiredArray[T any](raw json.RawMessage, field string, out *[]T) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return fmt.Errorf("%s is required", field)
	}
	if bytes.Equal(trimmed, []byte("null")) || trimmed[0] != '[' {
		return fmt.Errorf("%s must be an array, not null", field)
	}
	if err := decodeStrictJSON(trimmed, out); err != nil {
		return fmt.Errorf("%s: %w", field, err)
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

func validateTrustSnapshotCapsules(capsules []trustSnapshotItem) error {
	seen := map[string]bool{}
	for _, capsule := range capsules {
		if capsule.ID == "" {
			return fmt.Errorf("capsule missing id")
		}
		if !strings.HasPrefix(capsule.ID, "tetra://") {
			return fmt.Errorf("capsule %s id must use tetra:// prefix", capsule.ID)
		}
		if seen[capsule.ID] {
			return fmt.Errorf("duplicate capsule id %s", capsule.ID)
		}
		seen[capsule.ID] = true
		if capsule.Version == "" || !isCapsuleSemver(capsule.Version) {
			return fmt.Errorf("capsule %s version must use semver x.y.z", capsule.ID)
		}
		if err := validatePermissions(capsule.ID, capsule.Permissions); err != nil {
			return err
		}
		if capsule.TrustTier == "" {
			return fmt.Errorf("capsule %s trust_tier is required", capsule.ID)
		}
		if capsule.TrustScore == nil {
			return fmt.Errorf("capsule %s trust_score is required", capsule.ID)
		}
		if *capsule.TrustScore < 10 || *capsule.TrustScore > 100 {
			return fmt.Errorf("capsule %s trust_score must be between 10 and 100", capsule.ID)
		}
		expectedTier := trustTierForScore(*capsule.TrustScore)
		if capsule.TrustTier != expectedTier {
			return fmt.Errorf("capsule %s trust_tier mismatch: score %d expects %s, got %s", capsule.ID, *capsule.TrustScore, expectedTier, capsule.TrustTier)
		}
		expectedScore := trustScoreForPermissions(capsule.Permissions)
		if *capsule.TrustScore != expectedScore {
			return fmt.Errorf("capsule %s trust_score mismatch: permissions expect %d, got %d", capsule.ID, expectedScore, *capsule.TrustScore)
		}
		if len(capsule.TrustReasons) == 0 {
			return fmt.Errorf("capsule %s trust_reasons must not be empty", capsule.ID)
		}
		expectedReason := "permissions=" + strings.Join(capsule.Permissions, ",")
		if !containsString(capsule.TrustReasons, expectedReason) {
			return fmt.Errorf("capsule %s trust_reasons missing %q", capsule.ID, expectedReason)
		}
	}
	return nil
}

func validatePermissions(id string, permissions []string) error {
	seen := map[string]bool{}
	for _, permission := range permissions {
		if permission == "" {
			return fmt.Errorf("capsule %s has empty permission", id)
		}
		if _, ok := knownCapsulePermissions[permission]; !ok {
			return fmt.Errorf("capsule %s has unknown permission %s", id, permission)
		}
		if seen[permission] {
			return fmt.Errorf("capsule %s has duplicate permission %s", id, permission)
		}
		seen[permission] = true
	}
	return nil
}

func trustScoreForPermissions(permissions []string) int {
	score := 100
	for _, permission := range permissions {
		score -= 5
		switch permission {
		case "mem", "mmio", "capability":
			score -= 10
		}
	}
	if score < 10 {
		score = 10
	}
	return score
}

func trustTierForScore(score int) string {
	switch {
	case score >= 80:
		return "high"
	case score >= 60:
		return "medium"
	default:
		return "low"
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

func isCapsuleSemver(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, ch := range part {
			if ch < '0' || ch > '9' {
				return false
			}
		}
	}
	return true
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
