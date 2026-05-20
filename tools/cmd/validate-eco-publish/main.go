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
	"sort"
	"strings"
)

type publishMetadata struct {
	Schema        string            `json:"schema"`
	Channel       string            `json:"channel"`
	Hub           string            `json:"hub"`
	PublishedUnix int64             `json:"published_at_unix"`
	Capsule       publishCapsule    `json:"capsule"`
	Package       publishPackage    `json:"package"`
	Trust         *publishTrust     `json:"trust,omitempty"`
	Downloads     []publishDownload `json:"downloads,omitempty"`
}

type publishCapsule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Target      string   `json:"target"`
	Targets     []string `json:"targets,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

type publishPackage struct {
	File   string `json:"file"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type publishTrust struct {
	SnapshotFile string `json:"snapshot_file"`
	SnapshotHash string `json:"snapshot_sha256"`
	TrustTier    string `json:"trust_tier"`
}

type publishDownload struct {
	Target string `json:"target"`
	Path   string `json:"path"`
}

func main() {
	var registry string
	var id string
	var version string
	var target string
	var channel string
	flag.StringVar(&registry, "registry", "", "path to beta registry root")
	flag.StringVar(&id, "id", "", "capsule id")
	flag.StringVar(&version, "version", "", "capsule version")
	flag.StringVar(&target, "target", "", "target triple")
	flag.StringVar(&channel, "channel", "beta", "publish channel: beta or stable")
	flag.Parse()

	if registry == "" || id == "" || version == "" {
		fmt.Fprintln(os.Stderr, "error: --registry, --id, and --version are required")
		os.Exit(2)
	}
	if err := validatePublishedPackage(registry, id, version, target, channel); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validatePublishedPackage(registry string, id string, version string, target string, channel string) error {
	if !isSupportedPublishChannel(channel) {
		return fmt.Errorf("unsupported channel %q", channel)
	}
	base := filepath.Join(registry, "packages", capsuleIDDirectory(id), version)
	if target == "" {
		entries, err := os.ReadDir(base)
		if err != nil {
			return err
		}
		var dirs []string
		for _, entry := range entries {
			if entry.IsDir() {
				dirs = append(dirs, entry.Name())
			}
		}
		sort.Strings(dirs)
		if len(dirs) == 0 {
			return fmt.Errorf("no target entries under %s", base)
		}
		target = dirs[0]
	}
	targetDir := filepath.Join(base, target)
	rawMeta, err := os.ReadFile(filepath.Join(targetDir, "metadata.json"))
	if err != nil {
		return err
	}
	var meta publishMetadata
	decoder := json.NewDecoder(bytes.NewReader(rawMeta))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&meta); err != nil {
		return err
	}
	if meta.Schema != publishSchemaForChannel(channel) {
		return fmt.Errorf("unsupported schema %q", meta.Schema)
	}
	if meta.Channel != channel {
		return fmt.Errorf("unsupported channel %q", meta.Channel)
	}
	if meta.Hub == "" {
		return fmt.Errorf("hub is required")
	}
	if meta.PublishedUnix < 0 {
		return fmt.Errorf("published_at_unix must not be negative")
	}
	if meta.Capsule.ID != id {
		return fmt.Errorf("capsule id mismatch: metadata has %s", meta.Capsule.ID)
	}
	if meta.Capsule.Name == "" {
		return fmt.Errorf("capsule name is required")
	}
	if meta.Capsule.Version != version {
		return fmt.Errorf("capsule version mismatch: metadata has %s", meta.Capsule.Version)
	}
	if meta.Capsule.Target != target {
		return fmt.Errorf("capsule target mismatch: metadata has %s", meta.Capsule.Target)
	}
	if len(meta.Capsule.Targets) > 0 && !containsString(meta.Capsule.Targets, target) {
		return fmt.Errorf("capsule targets missing selected target %s", target)
	}
	if meta.Trust != nil {
		if err := validateRelativeMetadataPath(meta.Trust.SnapshotFile, "trust snapshot file"); err != nil {
			return err
		}
		hexHash, err := parseSHA256Hash(meta.Trust.SnapshotHash)
		if err != nil {
			return err
		}
		if meta.Trust.TrustTier == "" {
			return fmt.Errorf("trust tier is required")
		}
		snapshotPath := filepath.Join(targetDir, filepath.FromSlash(meta.Trust.SnapshotFile))
		rawSnapshot, err := os.ReadFile(snapshotPath)
		if err != nil {
			return err
		}
		snapshotSum := sha256.Sum256(rawSnapshot)
		if hex.EncodeToString(snapshotSum[:]) != hexHash {
			return fmt.Errorf("trust snapshot hash mismatch for %s", snapshotPath)
		}
	}
	if err := validateRelativeMetadataPath(meta.Package.File, "package file"); err != nil {
		return err
	}
	if meta.Package.Size < 0 {
		return fmt.Errorf("package size must not be negative")
	}
	expectedDownloadPath := filepath.ToSlash(filepath.Join("packages", capsuleIDDirectory(id), version, target, meta.Package.File))
	if len(meta.Downloads) == 0 {
		return fmt.Errorf("downloads must not be empty")
	}
	for _, download := range meta.Downloads {
		if download.Target != target {
			return fmt.Errorf("download target mismatch: metadata has %s", download.Target)
		}
		if err := validateRelativeMetadataPath(download.Path, "download path"); err != nil {
			return err
		}
		if download.Path != expectedDownloadPath {
			return fmt.Errorf("download path mismatch: metadata has %s, expected %s", download.Path, expectedDownloadPath)
		}
	}
	pkgPath := filepath.Join(targetDir, filepath.FromSlash(meta.Package.File))
	rawPkg, err := os.ReadFile(pkgPath)
	if err != nil {
		return err
	}
	if int64(len(rawPkg)) != meta.Package.Size {
		return fmt.Errorf("package size mismatch: metadata=%d actual=%d", meta.Package.Size, len(rawPkg))
	}
	hexHash, err := parseSHA256Hash(meta.Package.SHA256)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(rawPkg)
	if hex.EncodeToString(sum[:]) != hexHash {
		return fmt.Errorf("package hash mismatch for %s", pkgPath)
	}
	return nil
}

func isSupportedPublishChannel(channel string) bool {
	switch channel {
	case "beta", "stable":
		return true
	default:
		return false
	}
}

func publishSchemaForChannel(channel string) string {
	if channel == "stable" {
		return "tetra.eco.publish.v1"
	}
	return "tetra.eco.publish.v1beta"
}

func validateRelativeMetadataPath(path string, label string) error {
	if path == "" {
		return fmt.Errorf("%s is required", label)
	}
	if strings.Contains(path, "\\") {
		return fmt.Errorf("unsafe %s path %s", label, path)
	}
	clean := filepath.Clean(path)
	if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return fmt.Errorf("unsafe %s path %s", label, path)
	}
	if filepath.ToSlash(clean) != path {
		return fmt.Errorf("%s path %s is not normalized", label, path)
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

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
