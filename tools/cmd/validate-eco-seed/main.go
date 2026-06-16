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

	ctarget "tetra_language/compiler/target"
	"tetra_language/tools/internal/reportdecode"
)

const (
	seedSchemaV1       = "tetra.eco.seed.v1"
	lockSchemaV1       = "tetra.eco.lock.v1"
	manifestSchemaV1   = "tetra.capsule.v1"
	permissionsModelV1 = "tetra.eco.permissions.v1"
	sha256Prefix       = "sha256:"
	t4InterfaceExt     = ".t4i"
	t4ObjectExt        = ".tobj"
	t4SeedExt          = ".t4s"
)

type seedReport struct {
	Schema        string          `json:"schema"`
	GeneratedUnix *int64          `json:"generated_at_unix"`
	LockRaw       json.RawMessage `json:"lock"`
	CapsulesRaw   json.RawMessage `json:"capsules"`
	Lock          ecoLock         `json:"-"`
	Capsules      []ecoSeedItem   `json:"-"`
}

type ecoLock struct {
	Schema           string           `json:"schema"`
	ManifestSchema   string           `json:"manifest_schema"`
	PermissionsModel string           `json:"permissions_model"`
	GeneratedUnix    int64            `json:"generated_at_unix,omitempty"`
	GraphSHA256      string           `json:"graph_sha256,omitempty"`
	CapsulesRaw      json.RawMessage  `json:"capsules"`
	Capsules         []ecoLockCapsule `json:"-"`
}

type ecoLockCapsule struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Path         string            `json:"path"`
	Targets      []string          `json:"targets"`
	Effects      []string          `json:"effects,omitempty"`
	Permissions  []string          `json:"permissions,omitempty"`
	Dependencies []seedDependency  `json:"dependencies,omitempty"`
	Artifacts    []ecoLockArtifact `json:"artifacts,omitempty"`
	Policy       map[string]string `json:"policy,omitempty"`
}

type ecoLockArtifact struct {
	Kind          string `json:"kind"`
	Target        string `json:"target,omitempty"`
	Module        string `json:"module,omitempty"`
	PublicAPIHash string `json:"public_api_hash,omitempty"`
	Path          string `json:"path"`
	SHA256        string `json:"sha256,omitempty"`
}

type ecoSeedItem struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Version     string           `json:"version"`
	Targets     []string         `json:"targets,omitempty"`
	Effects     []string         `json:"effects,omitempty"`
	Permissions []string         `json:"permissions,omitempty"`
	DependsOn   []seedDependency `json:"depends_on,omitempty"`
}

type seedDependency struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Path    string `json:"path,omitempty"`
	PathSet bool   `json:"-"`
}

func (dep *seedDependency) UnmarshalJSON(raw []byte) error {
	var payload struct {
		ID      string  `json:"id"`
		Version string  `json:"version"`
		Path    *string `json:"path,omitempty"`
	}
	if err := decodeStrictJSON(raw, &payload); err != nil {
		return err
	}
	dep.ID = payload.ID
	dep.Version = payload.Version
	dep.Path = ""
	dep.PathSet = false
	if payload.Path != nil {
		dep.Path = *payload.Path
		dep.PathSet = true
	}
	return nil
}

var knownCapsuleEffects = map[string]string{
	"actors":     "actors",
	"alloc":      "alloc",
	"cap.io":     "io",
	"cap.mem":    "mem",
	"capability": "capability",
	"control":    "control",
	"io":         "io",
	"islands":    "islands",
	"link":       "link",
	"mem":        "mem",
	"mmio":       "mmio",
	"runtime":    "runtime",
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
	var seedPath string
	var reportFormat string
	flag.StringVar(&seedPath, "seed", "", "path to tetra.eco.seed.v1 JSON report")
	flag.StringVar(&reportFormat, "format", "auto", "report format: auto, json, or toon")
	flag.Parse()

	if seedPath == "" {
		fmt.Fprintln(os.Stderr, "error: --seed is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(seedPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateEcoSeedFormat(raw, reportFormat); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateEcoSeed(raw []byte) error {
	return validateEcoSeedFormat(raw, "auto")
}

func validateEcoSeedFormat(raw []byte, format string) error {
	var report seedReport
	if err := reportdecode.DecodeStrictFormat(raw, format, &report); err != nil {
		return err
	}
	if report.Schema == "" {
		return fmt.Errorf("schema is required")
	}
	if report.Schema != seedSchemaV1 {
		return fmt.Errorf("unsupported seed schema %q", report.Schema)
	}
	if report.GeneratedUnix == nil {
		return fmt.Errorf("generated_at_unix is required")
	}
	if *report.GeneratedUnix < 0 {
		return fmt.Errorf("generated_at_unix must not be negative")
	}
	if len(bytes.TrimSpace(report.LockRaw)) == 0 || bytes.Equal(bytes.TrimSpace(report.LockRaw), []byte("null")) {
		return fmt.Errorf("lock is required")
	}
	if err := decodeEcoLock(report.LockRaw, &report.Lock); err != nil {
		return err
	}
	if err := unmarshalArray(report.CapsulesRaw, "capsules", &report.Capsules); err != nil {
		return err
	}
	if len(report.Capsules) == 0 {
		return fmt.Errorf("capsules must not be empty")
	}
	if err := validateSeedCapsules(report.Capsules); err != nil {
		return err
	}
	if err := validateSeedMatchesLock(report.Capsules, report.Lock.Capsules); err != nil {
		return err
	}
	return nil
}

func decodeEcoLock(raw json.RawMessage, lock *ecoLock) error {
	if err := decodeStrictJSON(raw, lock); err != nil {
		return fmt.Errorf("lock: %w", err)
	}
	if lock.Schema == "" {
		return fmt.Errorf("lock schema is required")
	}
	if lock.Schema != lockSchemaV1 {
		return fmt.Errorf("unsupported lock schema %q", lock.Schema)
	}
	if lock.ManifestSchema == "" {
		return fmt.Errorf("lock manifest_schema is required")
	}
	if lock.ManifestSchema != manifestSchemaV1 {
		return fmt.Errorf("unsupported lock manifest_schema %q", lock.ManifestSchema)
	}
	if lock.PermissionsModel == "" {
		return fmt.Errorf("lock permissions_model is required")
	}
	if lock.PermissionsModel != permissionsModelV1 {
		return fmt.Errorf("unsupported lock permissions_model %q", lock.PermissionsModel)
	}
	if lock.GeneratedUnix < 0 {
		return fmt.Errorf("lock generated_at_unix must not be negative")
	}
	if lock.GraphSHA256 != "" {
		if _, err := parseSHA256Hash(lock.GraphSHA256); err != nil {
			return fmt.Errorf("invalid lock graph_sha256: %w", err)
		}
	}
	if err := unmarshalArray(lock.CapsulesRaw, "lock capsules", &lock.Capsules); err != nil {
		return err
	}
	if len(lock.Capsules) == 0 {
		return fmt.Errorf("lock capsules must not be empty")
	}
	if err := validateLockCapsules(lock.Capsules); err != nil {
		return err
	}
	if lock.GraphSHA256 != "" {
		sum := sha256.Sum256([]byte(lockGraphFingerprint(lock.Capsules)))
		expected := sha256Prefix + hex.EncodeToString(sum[:])
		if lock.GraphSHA256 != expected {
			return fmt.Errorf("lock graph_sha256 mismatch: metadata has %s, computed %s", lock.GraphSHA256, expected)
		}
	}
	return nil
}

func unmarshalArray[T any](raw json.RawMessage, field string, out *[]T) error {
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

func validateSeedCapsules(capsules []ecoSeedItem) error {
	seen := map[string]bool{}
	for _, capsule := range capsules {
		if err := validateSeedCapsule(capsule); err != nil {
			return err
		}
		if seen[capsule.ID] {
			return fmt.Errorf("duplicate seed capsule id %s", capsule.ID)
		}
		seen[capsule.ID] = true
	}
	return nil
}

func validateSeedCapsule(capsule ecoSeedItem) error {
	if err := validateCapsuleIdentity("seed capsule", capsule.ID, capsule.Name, capsule.Version); err != nil {
		return err
	}
	if err := validateTargets("seed capsule "+capsule.ID, capsule.Targets); err != nil {
		return err
	}
	if err := validateEffectsAndPermissions("seed capsule "+capsule.ID, capsule.Effects, capsule.Permissions); err != nil {
		return err
	}
	return validateDependencies("seed capsule "+capsule.ID, capsule.ID, capsule.DependsOn)
}

func validateLockCapsules(capsules []ecoLockCapsule) error {
	seen := map[string]ecoLockCapsule{}
	for _, capsule := range capsules {
		if err := validateLockCapsule(capsule); err != nil {
			return err
		}
		if _, exists := seen[capsule.ID]; exists {
			return fmt.Errorf("duplicate lock capsule id %s", capsule.ID)
		}
		seen[capsule.ID] = capsule
	}
	for _, capsule := range capsules {
		for _, dep := range capsule.Dependencies {
			found, ok := seen[dep.ID]
			if !ok {
				return fmt.Errorf("lock capsule %s references unknown dependency %s", capsule.ID, dep.ID)
			}
			if found.Version != dep.Version {
				return fmt.Errorf("lock capsule %s dependency %s version mismatch: wants %s, lock has %s", capsule.ID, dep.ID, dep.Version, found.Version)
			}
			for _, effect := range found.Effects {
				if !containsString(capsule.Effects, effect) {
					return fmt.Errorf("lock capsule %s missing required effect %s for dependency %s", capsule.ID, effect, dep.ID)
				}
			}
			for _, permission := range found.Permissions {
				if !containsString(capsule.Permissions, permission) {
					return fmt.Errorf("lock capsule %s missing required permission %s for dependency %s", capsule.ID, permission, dep.ID)
				}
			}
		}
	}
	return nil
}

func validateLockCapsule(capsule ecoLockCapsule) error {
	if err := validateCapsuleIdentity("lock capsule", capsule.ID, capsule.Name, capsule.Version); err != nil {
		return err
	}
	if _, err := validatePortableRelativePath(capsule.Path, "lock capsule "+capsule.ID); err != nil {
		return err
	}
	if err := validateTargets("lock capsule "+capsule.ID, capsule.Targets); err != nil {
		return err
	}
	if err := validateEffectsAndPermissions("lock capsule "+capsule.ID, capsule.Effects, capsule.Permissions); err != nil {
		return err
	}
	if err := validateDependencies("lock capsule "+capsule.ID, capsule.ID, capsule.Dependencies); err != nil {
		return err
	}
	if err := validateCapsulePolicy(capsule.ID, capsule.Policy); err != nil {
		return err
	}
	return validateArtifacts(capsule.ID, capsule.Artifacts)
}

func validateSeedMatchesLock(seedCapsules []ecoSeedItem, lockCapsules []ecoLockCapsule) error {
	lockByID := map[string]ecoLockCapsule{}
	for _, capsule := range lockCapsules {
		lockByID[capsule.ID] = capsule
	}
	if len(seedCapsules) != len(lockCapsules) {
		return fmt.Errorf("seed capsule count %d does not match lock capsule count %d", len(seedCapsules), len(lockCapsules))
	}
	for _, seed := range seedCapsules {
		lock, ok := lockByID[seed.ID]
		if !ok {
			return fmt.Errorf("seed capsule %s missing from lock", seed.ID)
		}
		if seed.Name != lock.Name {
			return fmt.Errorf("seed capsule %s name mismatch: seed has %s, lock has %s", seed.ID, seed.Name, lock.Name)
		}
		if seed.Version != lock.Version {
			return fmt.Errorf("seed capsule %s version mismatch: seed has %s, lock has %s", seed.ID, seed.Version, lock.Version)
		}
		if !sameStringSet(seed.Targets, lock.Targets) {
			return fmt.Errorf("seed capsule %s targets mismatch", seed.ID)
		}
		if !sameStringSet(seed.Effects, lock.Effects) {
			return fmt.Errorf("seed capsule %s effects mismatch", seed.ID)
		}
		if !sameStringSet(seed.Permissions, lock.Permissions) {
			return fmt.Errorf("seed capsule %s permissions mismatch", seed.ID)
		}
		if !sameDependencies(seed.DependsOn, lock.Dependencies) {
			return fmt.Errorf("seed capsule %s dependencies mismatch", seed.ID)
		}
	}
	return nil
}

func validateCapsuleIdentity(label string, id string, name string, version string) error {
	if id == "" {
		return fmt.Errorf("%s missing id", label)
	}
	if !strings.HasPrefix(id, "tetra://") {
		return fmt.Errorf("%s %s id must use tetra:// prefix", label, id)
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%s %s missing name", label, id)
	}
	if version == "" || !isCapsuleSemver(version) {
		return fmt.Errorf("%s %s version must use semver x.y.z", label, id)
	}
	return nil
}

func validateTargets(label string, targets []string) error {
	if len(targets) == 0 {
		return fmt.Errorf("%s missing targets", label)
	}
	supported := supportedTargets()
	seen := map[string]bool{}
	for _, target := range targets {
		if target == "" {
			return fmt.Errorf("%s has empty target", label)
		}
		if !supported[target] {
			return fmt.Errorf("%s has unsupported target %s", label, target)
		}
		if seen[target] {
			return fmt.Errorf("%s has duplicate target %s", label, target)
		}
		seen[target] = true
	}
	return nil
}

func validateEffectsAndPermissions(label string, effects []string, permissions []string) error {
	seenEffects := map[string]bool{}
	for _, effect := range effects {
		normalized, ok := knownCapsuleEffects[effect]
		if !ok {
			return fmt.Errorf("%s has unknown effect %s", label, effect)
		}
		if seenEffects[normalized] {
			return fmt.Errorf("%s has duplicate effect %s", label, normalized)
		}
		seenEffects[normalized] = true
	}
	seenPermissions := map[string]bool{}
	for _, permission := range permissions {
		normalized, ok := knownCapsulePermissions[permission]
		if !ok {
			return fmt.Errorf("%s has unknown permission %s", label, permission)
		}
		if seenPermissions[normalized] {
			return fmt.Errorf("%s has duplicate permission %s", label, normalized)
		}
		seenPermissions[normalized] = true
	}
	return nil
}

func validateDependencies(label string, capsuleID string, deps []seedDependency) error {
	seen := map[string]bool{}
	for _, dep := range deps {
		if dep.ID == "" {
			return fmt.Errorf("%s has dependency with empty id", label)
		}
		if !strings.HasPrefix(dep.ID, "tetra://") {
			return fmt.Errorf("%s dependency %s must use tetra:// prefix", label, dep.ID)
		}
		if dep.Version == "" || !isCapsuleSemver(dep.Version) {
			return fmt.Errorf("%s dependency %s has invalid semver version %s", label, dep.ID, dep.Version)
		}
		if dep.ID == capsuleID {
			return fmt.Errorf("%s cannot depend on itself", label)
		}
		if dep.PathSet {
			if _, err := validatePortableRelativePath(dep.Path, label+" dependency "+dep.ID); err != nil {
				return err
			}
		}
		if seen[dep.ID] {
			return fmt.Errorf("%s has duplicate dependency %s", label, dep.ID)
		}
		seen[dep.ID] = true
	}
	return nil
}

func validatePortableRelativePath(value string, label string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s path must not be empty", label)
	}
	if strings.Contains(value, "\\") {
		return "", fmt.Errorf("%s path must use forward slashes", label)
	}
	if isPortableAbsolutePath(value) {
		return "", fmt.Errorf("%s path must be relative", label)
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == ".." {
			return "", fmt.Errorf("%s path must not contain ..", label)
		}
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if clean == "." {
		return "", fmt.Errorf("%s path must not normalize to empty", label)
	}
	if clean != value {
		return "", fmt.Errorf("%s path must already be normalized", label)
	}
	return clean, nil
}

func isPortableAbsolutePath(value string) bool {
	if filepath.IsAbs(value) || strings.HasPrefix(value, "/") {
		return true
	}
	if len(value) >= 3 && value[1] == ':' && value[2] == '/' {
		ch := value[0]
		return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
	}
	return false
}

func validateCapsulePolicy(capsuleID string, policy map[string]string) error {
	for key, value := range policy {
		switch key {
		case "unsafe":
			if value != "deny" && value != "allow" {
				return fmt.Errorf("lock capsule %s unsafe policy must be deny or allow", capsuleID)
			}
		case "reproducible":
			if value != "required" && value != "preferred" && value != "off" {
				return fmt.Errorf("lock capsule %s reproducible policy must be required, preferred, or off", capsuleID)
			}
		default:
			return fmt.Errorf("lock capsule %s unknown policy %s", capsuleID, key)
		}
	}
	return nil
}

func validateArtifacts(capsuleID string, artifacts []ecoLockArtifact) error {
	seen := map[string]bool{}
	for _, artifact := range artifacts {
		kind, err := normalizeArtifactKind(artifact.Kind)
		if err != nil {
			return fmt.Errorf("lock capsule %s %v", capsuleID, err)
		}
		if artifact.Target != "" {
			if !supportedTargets()[artifact.Target] {
				return fmt.Errorf("lock capsule %s artifact %s has unsupported target %s", capsuleID, artifact.Path, artifact.Target)
			}
			if kind != "object" {
				return fmt.Errorf("lock capsule %s only object artifacts accept a target", capsuleID)
			}
		}
		cleanPath, err := cleanArtifactPath(artifact.Path)
		if err != nil {
			return fmt.Errorf("lock capsule %s %v", capsuleID, err)
		}
		if err := validateArtifactExtension(kind, cleanPath); err != nil {
			return fmt.Errorf("lock capsule %s %v", capsuleID, err)
		}
		if artifact.SHA256 != "" {
			if _, err := parseSHA256Hash(artifact.SHA256); err != nil {
				return fmt.Errorf("lock capsule %s artifact %s has invalid sha256: %w", capsuleID, cleanPath, err)
			}
		}
		if artifact.PublicAPIHash != "" {
			if _, err := parseSHA256Hash(artifact.PublicAPIHash); err != nil {
				return fmt.Errorf("lock capsule %s artifact %s has invalid public_api_hash: %w", capsuleID, cleanPath, err)
			}
		}
		key := kind + "\x00" + artifact.Target + "\x00" + cleanPath
		if seen[key] {
			return fmt.Errorf("lock capsule %s has duplicate artifact %s %s", capsuleID, kind, cleanPath)
		}
		seen[key] = true
	}
	return nil
}

func normalizeArtifactKind(kind string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "interface", "t4i":
		return "interface", nil
	case "object", "tobj":
		return "object", nil
	case "seed", "t4s":
		return "seed", nil
	default:
		return "", fmt.Errorf("has unknown artifact kind %s", kind)
	}
}

func cleanArtifactPath(value string) (string, error) {
	return validatePortableRelativePath(value, "artifact")
}

func validateArtifactExtension(kind string, path string) error {
	switch kind {
	case "interface":
		if filepath.Ext(path) != t4InterfaceExt {
			return fmt.Errorf("interface artifact must use %s", t4InterfaceExt)
		}
	case "object":
		if filepath.Ext(path) != t4ObjectExt {
			return fmt.Errorf("object artifact must use %s", t4ObjectExt)
		}
	case "seed":
		if filepath.Ext(path) != t4SeedExt {
			return fmt.Errorf("seed artifact must use %s", t4SeedExt)
		}
	default:
		return fmt.Errorf("has unknown artifact kind %s", kind)
	}
	return nil
}

func lockGraphFingerprint(items []ecoLockCapsule) string {
	normalized := append([]ecoLockCapsule(nil), items...)
	sort.Slice(normalized, func(i, j int) bool { return normalized[i].ID < normalized[j].ID })
	var b strings.Builder
	for _, item := range normalized {
		b.WriteString(item.ID)
		b.WriteByte('|')
		b.WriteString(item.Version)
		b.WriteByte('|')
		b.WriteString(strings.Join(sortedStringCopy(item.Targets), ","))
		b.WriteByte('|')
		b.WriteString(strings.Join(sortedStringCopy(item.Permissions), ","))
		b.WriteByte('|')
		b.WriteString(policyFingerprint(item.Policy))
		b.WriteByte('|')
		for _, dep := range sortedDependencyCopy(item.Dependencies) {
			b.WriteString(dep.ID)
			b.WriteByte('@')
			b.WriteString(dep.Version)
			if dep.Path != "" {
				b.WriteByte(':')
				b.WriteString(dep.Path)
			}
			b.WriteByte(',')
		}
		b.WriteByte('|')
		for _, artifact := range sortedArtifactCopy(item.Artifacts) {
			b.WriteString(artifact.Kind)
			b.WriteByte(':')
			if artifact.Target != "" {
				b.WriteString(artifact.Target)
			}
			b.WriteByte(':')
			if artifact.Module != "" {
				b.WriteString(artifact.Module)
			}
			b.WriteByte(':')
			if artifact.PublicAPIHash != "" {
				b.WriteString(artifact.PublicAPIHash)
			}
			b.WriteByte(':')
			b.WriteString(artifact.Path)
			if artifact.SHA256 != "" {
				b.WriteByte('@')
				b.WriteString(artifact.SHA256)
			}
			b.WriteByte(',')
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func policyFingerprint(policy map[string]string) string {
	if len(policy) == 0 {
		return ""
	}
	keys := make([]string, 0, len(policy))
	for key := range policy {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, key := range keys {
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(policy[key])
		b.WriteByte(',')
	}
	return b.String()
}

func sameStringSet(left []string, right []string) bool {
	leftSorted := sortedStringCopy(left)
	rightSorted := sortedStringCopy(right)
	if len(leftSorted) != len(rightSorted) {
		return false
	}
	for i := range leftSorted {
		if leftSorted[i] != rightSorted[i] {
			return false
		}
	}
	return true
}

func sameDependencies(left []seedDependency, right []seedDependency) bool {
	leftSorted := sortedDependencyCopy(left)
	rightSorted := sortedDependencyCopy(right)
	if len(leftSorted) != len(rightSorted) {
		return false
	}
	for i := range leftSorted {
		if leftSorted[i] != rightSorted[i] {
			return false
		}
	}
	return true
}

func sortedStringCopy(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func sortedDependencyCopy(values []seedDependency) []seedDependency {
	out := append([]seedDependency(nil), values...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].ID == out[j].ID {
			if out[i].Version == out[j].Version {
				return out[i].Path < out[j].Path
			}
			return out[i].Version < out[j].Version
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func sortedArtifactCopy(values []ecoLockArtifact) []ecoLockArtifact {
	out := append([]ecoLockArtifact(nil), values...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind == out[j].Kind {
			if out[i].Target == out[j].Target {
				return out[i].Path < out[j].Path
			}
			return out[i].Target < out[j].Target
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}

func supportedTargets() map[string]bool {
	targets := map[string]bool{}
	for _, triple := range ctarget.SupportedTriples() {
		targets[triple] = true
	}
	for _, triple := range ctarget.BuildOnlyTriples() {
		targets[triple] = true
	}
	return targets
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func isCapsuleSemver(version string) bool {
	if version == "" {
		return false
	}
	main := version
	if idx := strings.IndexAny(version, "-+"); idx >= 0 {
		main = version[:idx]
	}
	parts := strings.Split(main, ".")
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
