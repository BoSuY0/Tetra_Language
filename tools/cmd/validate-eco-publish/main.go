package main

import (
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
	Schema  string         `json:"schema"`
	Channel string         `json:"channel"`
	Hub     string         `json:"hub"`
	Capsule publishCapsule `json:"capsule"`
	Package publishPackage `json:"package"`
	Trust   *publishTrust  `json:"trust,omitempty"`
}

type publishCapsule struct {
	ID      string   `json:"id"`
	Version string   `json:"version"`
	Target  string   `json:"target"`
	Targets []string `json:"targets,omitempty"`
}

type publishPackage struct {
	File   string `json:"file"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type publishTrust struct {
	SnapshotHash string `json:"snapshot_sha256"`
}

func main() {
	var registry string
	var id string
	var version string
	var target string
	flag.StringVar(&registry, "registry", "", "path to beta registry root")
	flag.StringVar(&id, "id", "", "capsule id")
	flag.StringVar(&version, "version", "", "capsule version")
	flag.StringVar(&target, "target", "", "target triple")
	flag.Parse()

	if registry == "" || id == "" || version == "" {
		fmt.Fprintln(os.Stderr, "error: --registry, --id, and --version are required")
		os.Exit(2)
	}
	if err := validatePublishedPackage(registry, id, version, target); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validatePublishedPackage(registry string, id string, version string, target string) error {
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
	if err := json.Unmarshal(rawMeta, &meta); err != nil {
		return err
	}
	if meta.Schema != "tetra.eco.publish.v1beta" {
		return fmt.Errorf("unsupported schema %q", meta.Schema)
	}
	if meta.Channel != "beta" {
		return fmt.Errorf("unsupported channel %q", meta.Channel)
	}
	if meta.Capsule.ID != id {
		return fmt.Errorf("capsule id mismatch: metadata has %s", meta.Capsule.ID)
	}
	if meta.Capsule.Version != version {
		return fmt.Errorf("capsule version mismatch: metadata has %s", meta.Capsule.Version)
	}
	if meta.Capsule.Target != target {
		return fmt.Errorf("capsule target mismatch: metadata has %s", meta.Capsule.Target)
	}
	if meta.Trust != nil {
		if _, err := parseSHA256Hash(meta.Trust.SnapshotHash); err != nil {
			return err
		}
	}
	pkgPath := filepath.Join(targetDir, meta.Package.File)
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
