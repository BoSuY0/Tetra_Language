package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tetra_language/compiler"
)

type ecoLock struct {
	Schema           string           `json:"schema,omitempty"`
	ManifestSchema   string           `json:"manifest_schema,omitempty"`
	PermissionsModel string           `json:"permissions_model,omitempty"`
	GeneratedUnix    int64            `json:"generated_at_unix,omitempty"`
	GraphSHA256      string           `json:"graph_sha256,omitempty"`
	Capsules         []ecoLockCapsule `json:"capsules"`
}

type ecoLockCapsule struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Version      string              `json:"version"`
	Path         string              `json:"path"`
	Targets      []string            `json:"targets,omitempty"`
	Effects      []string            `json:"effects,omitempty"`
	Permissions  []string            `json:"permissions,omitempty"`
	Dependencies []capsuleDependency `json:"dependencies,omitempty"`
	Artifacts    []ecoLockArtifact   `json:"artifacts,omitempty"`
	Policy       map[string]string   `json:"policy,omitempty"`
}

type ecoLockArtifact struct {
	Kind          string `json:"kind"`
	Target        string `json:"target,omitempty"`
	Module        string `json:"module,omitempty"`
	PublicAPIHash string `json:"public_api_hash,omitempty"`
	Path          string `json:"path"`
	SHA256        string `json:"sha256,omitempty"`
}

func writeEcoLock(path string, manifests []capsuleManifest) error {
	lock, err := buildEcoLockWithArtifactHashes(manifests)
	if err != nil {
		return err
	}
	raw, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func buildEcoLockWithArtifactHashes(manifests []capsuleManifest) (ecoLock, error) {
	lock := buildEcoLock(manifests)
	if err := hydrateLockArtifactHashes(&lock, manifests); err != nil {
		return ecoLock{}, err
	}
	setEcoLockGraphHash(&lock)
	return lock, nil
}

func buildEcoLock(manifests []capsuleManifest) ecoLock {
	items := make([]ecoLockCapsule, 0, len(manifests))
	for _, manifest := range manifests {
		item := ecoLockCapsule{
			ID:           manifest.ID,
			Name:         manifest.Name,
			Version:      manifest.Version,
			Path:         filepath.Clean(manifest.Path),
			Targets:      sortedStrings(manifest.Targets),
			Effects:      sortedStrings(manifest.Effects),
			Permissions:  sortedStrings(manifest.Permissions),
			Dependencies: append([]capsuleDependency(nil), manifest.Dependencies...),
			Artifacts:    lockArtifactsFromCapsuleArtifacts(manifest.Artifacts),
			Policy:       copySortedPolicy(manifest.Policy),
		}
		sort.Slice(item.Dependencies, func(i, j int) bool {
			if item.Dependencies[i].ID == item.Dependencies[j].ID {
				return item.Dependencies[i].Version < item.Dependencies[j].Version
			}
			return item.Dependencies[i].ID < item.Dependencies[j].ID
		})
		sortEcoLockArtifacts(item.Artifacts)
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	lock := ecoLock{
		Schema:           ecoLockSchemaV1,
		ManifestSchema:   capsuleManifestSchemaV1,
		PermissionsModel: ecoPermissionsModelV1,
		GeneratedUnix:    0,
		Capsules:         items,
	}
	setEcoLockGraphHash(&lock)
	return lock
}

func lockArtifactsFromCapsuleArtifacts(artifacts []capsuleArtifact) []ecoLockArtifact {
	out := make([]ecoLockArtifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		out = append(out, ecoLockArtifact{Kind: artifact.Kind, Target: artifact.Target, Path: artifact.Path})
	}
	sortEcoLockArtifacts(out)
	return out
}

func hydrateLockArtifactHashes(lock *ecoLock, manifests []capsuleManifest) error {
	rootsByID := make(map[string]string, len(manifests))
	for _, manifest := range manifests {
		if manifest.ID == "" || manifest.Path == "" {
			continue
		}
		rootsByID[manifest.ID] = filepath.Dir(manifest.Path)
	}
	for i := range lock.Capsules {
		root := rootsByID[lock.Capsules[i].ID]
		if root == "" {
			continue
		}
		for j := range lock.Capsules[i].Artifacts {
			path := filepath.Join(root, filepath.FromSlash(lock.Capsules[i].Artifacts[j].Path))
			raw, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("%s: read artifact %s: %w", lock.Capsules[i].ID, lock.Capsules[i].Artifacts[j].Path, err)
			}
			sum := sha256.Sum256(raw)
			lock.Capsules[i].Artifacts[j].SHA256 = "sha256:" + hex.EncodeToString(sum[:])
			switch lock.Capsules[i].Artifacts[j].Kind {
			case "interface":
				moduleName := interfaceArtifactModuleName(raw)
				if moduleName == "" {
					return fmt.Errorf("%s: artifact %s missing module declaration", lock.Capsules[i].ID, lock.Capsules[i].Artifacts[j].Path)
				}
				hash, err := compiler.InterfaceFingerprintFromT4I(raw)
				if err != nil {
					return fmt.Errorf("%s: artifact %s: %w", lock.Capsules[i].ID, lock.Capsules[i].Artifacts[j].Path, err)
				}
				lock.Capsules[i].Artifacts[j].Module = moduleName
				lock.Capsules[i].Artifacts[j].PublicAPIHash = hash
			case "object":
				obj, err := compiler.ReadObject(path)
				if err != nil {
					return fmt.Errorf("%s: artifact %s: %w", lock.Capsules[i].ID, lock.Capsules[i].Artifacts[j].Path, err)
				}
				if lock.Capsules[i].Artifacts[j].Target != "" && obj.Target != "" && lock.Capsules[i].Artifacts[j].Target != obj.Target {
					return fmt.Errorf("%s: artifact %s target mismatch: manifest %s, object %s", lock.Capsules[i].ID, lock.Capsules[i].Artifacts[j].Path, lock.Capsules[i].Artifacts[j].Target, obj.Target)
				}
				if obj.Target != "" {
					lock.Capsules[i].Artifacts[j].Target = obj.Target
				}
				lock.Capsules[i].Artifacts[j].Module = obj.Module
				lock.Capsules[i].Artifacts[j].PublicAPIHash = obj.PublicAPIHash
			}
		}
		sortEcoLockArtifacts(lock.Capsules[i].Artifacts)
	}
	return nil
}

func setEcoLockGraphHash(lock *ecoLock) {
	if lock == nil {
		return
	}
	sum := sha256.Sum256([]byte(lockGraphFingerprint(lock.Capsules)))
	lock.GraphSHA256 = "sha256:" + hex.EncodeToString(sum[:])
}

func decodeEcoLock(raw []byte) (ecoLock, error) {
	var lock ecoLock
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&lock); err != nil {
		return ecoLock{}, err
	}
	normalizeLock(&lock)
	if err := validateEcoLockModel(lock); err != nil {
		return ecoLock{}, err
	}
	return lock, nil
}

func validateEcoLockModel(lock ecoLock) error {
	if lock.Schema != ecoLockSchemaV1 {
		return fmt.Errorf("unsupported lock schema %q", lock.Schema)
	}
	if lock.ManifestSchema != capsuleManifestSchemaV1 {
		return fmt.Errorf("unsupported lock manifest schema %q", lock.ManifestSchema)
	}
	if lock.PermissionsModel != ecoPermissionsModelV1 {
		return fmt.Errorf("unsupported lock permissions model %q", lock.PermissionsModel)
	}
	if len(lock.Capsules) == 0 {
		return fmt.Errorf("lock contains no capsules")
	}
	byID := make(map[string]ecoLockCapsule, len(lock.Capsules))
	for _, capsule := range lock.Capsules {
		if capsule.ID == "" {
			return fmt.Errorf("lock capsule missing id")
		}
		if !strings.HasPrefix(capsule.ID, "tetra://") {
			return fmt.Errorf("lock capsule %s id must use tetra:// prefix", capsule.ID)
		}
		if _, exists := byID[capsule.ID]; exists {
			return fmt.Errorf("duplicate lock capsule id %s", capsule.ID)
		}
		if capsule.Name == "" {
			return fmt.Errorf("lock capsule %s missing name", capsule.ID)
		}
		if !isCapsuleSemver(capsule.Version) {
			return fmt.Errorf("lock capsule %s version must use semver x.y.z", capsule.ID)
		}
		for _, effect := range capsule.Effects {
			if _, err := normalizeCapsuleEffect(effect); err != nil {
				return fmt.Errorf("lock capsule %s: %v", capsule.ID, err)
			}
		}
		for _, permission := range capsule.Permissions {
			if _, err := normalizeCapsulePermission(permission); err != nil {
				return fmt.Errorf("lock capsule %s: %v", capsule.ID, err)
			}
		}
		if err := validateCapsulePolicy(capsule.Policy); err != nil {
			return fmt.Errorf("lock capsule %s: %v", capsule.ID, err)
		}
		seenDeps := map[string]struct{}{}
		for _, dep := range capsule.Dependencies {
			if dep.ID == "" {
				return fmt.Errorf("lock capsule %s has dependency with empty id", capsule.ID)
			}
			if !strings.HasPrefix(dep.ID, "tetra://") {
				return fmt.Errorf("lock capsule %s dependency %s must use tetra:// prefix", capsule.ID, dep.ID)
			}
			if !isCapsuleSemver(dep.Version) {
				return fmt.Errorf("lock capsule %s dependency %s has invalid semver %s", capsule.ID, dep.ID, dep.Version)
			}
			if dep.Path != "" && strings.Contains(dep.Path, "\\") {
				return fmt.Errorf("lock capsule %s dependency %s has non-normalized path %s", capsule.ID, dep.ID, dep.Path)
			}
			key := dep.ID + "\x00" + dep.Version
			if _, exists := seenDeps[key]; exists {
				return fmt.Errorf("lock capsule %s has duplicate dependency %s %s", capsule.ID, dep.ID, dep.Version)
			}
			seenDeps[key] = struct{}{}
		}
		seenArtifacts := map[string]struct{}{}
		for _, artifact := range capsule.Artifacts {
			kind, err := normalizeCapsuleArtifactKind(artifact.Kind)
			if err != nil {
				return fmt.Errorf("lock capsule %s: %v", capsule.ID, err)
			}
			if artifact.Target != "" {
				if _, err := normalizeCapsuleTarget(artifact.Target); err != nil {
					return fmt.Errorf("lock capsule %s: %v", capsule.ID, err)
				}
				if kind != "object" {
					return fmt.Errorf("lock capsule %s: only object artifacts accept a target", capsule.ID)
				}
			}
			if _, err := cleanCapsuleArtifactPath(artifact.Path); err != nil {
				return fmt.Errorf("lock capsule %s: %v", capsule.ID, err)
			}
			if err := validateCapsuleArtifactExtension(kind, artifact.Path); err != nil {
				return fmt.Errorf("lock capsule %s: %v", capsule.ID, err)
			}
			if artifact.SHA256 != "" {
				if _, err := ecoPublishPackageHashHex(artifact.SHA256); err != nil {
					return fmt.Errorf("lock capsule %s artifact %s has invalid sha256: %w", capsule.ID, artifact.Path, err)
				}
			}
			if artifact.PublicAPIHash != "" {
				if _, err := ecoPublishPackageHashHex(artifact.PublicAPIHash); err != nil {
					return fmt.Errorf("lock capsule %s artifact %s has invalid public_api_hash: %w", capsule.ID, artifact.Path, err)
				}
			}
			key := kind + "\x00" + artifact.Target + "\x00" + artifact.Path
			if _, exists := seenArtifacts[key]; exists {
				return fmt.Errorf("lock capsule %s has duplicate artifact %s %s", capsule.ID, artifact.Kind, artifact.Path)
			}
			seenArtifacts[key] = struct{}{}
		}
		byID[capsule.ID] = capsule
	}
	for _, capsule := range lock.Capsules {
		for _, dep := range capsule.Dependencies {
			found, ok := byID[dep.ID]
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
	if lock.GraphSHA256 != "" {
		hashHex, err := ecoPublishPackageHashHex(lock.GraphSHA256)
		if err != nil {
			return fmt.Errorf("invalid lock graph_sha256: %w", err)
		}
		sum := sha256.Sum256([]byte(lockGraphFingerprint(lock.Capsules)))
		if hex.EncodeToString(sum[:]) != hashHex {
			return fmt.Errorf("lock graph_sha256 mismatch")
		}
	}
	return nil
}

func normalizeLock(lock *ecoLock) {
	if lock.Schema == "" {
		lock.Schema = ecoLockSchemaV1
	}
	if lock.ManifestSchema == "" {
		lock.ManifestSchema = capsuleManifestSchemaV1
	}
	if lock.PermissionsModel == "" {
		lock.PermissionsModel = ecoPermissionsModelV1
	}
	for i := range lock.Capsules {
		item := &lock.Capsules[i]
		if item.Path == "" {
			item.Path = filepath.Clean(item.Name + compiler.T4SourceExtension)
		}
		for _, effect := range item.Effects {
			item.Permissions = appendUniqueString(item.Permissions, effect)
		}
		sort.Strings(item.Effects)
		sort.Strings(item.Permissions)
		sort.Strings(item.Targets)
		item.Policy = copySortedPolicy(item.Policy)
		sort.Slice(item.Dependencies, func(i, j int) bool {
			if item.Dependencies[i].ID == item.Dependencies[j].ID {
				return item.Dependencies[i].Version < item.Dependencies[j].Version
			}
			return item.Dependencies[i].ID < item.Dependencies[j].ID
		})
		sortEcoLockArtifacts(item.Artifacts)
	}
	sort.Slice(lock.Capsules, func(i, j int) bool { return lock.Capsules[i].ID < lock.Capsules[j].ID })
	if lock.GraphSHA256 == "" {
		setEcoLockGraphHash(lock)
	}
}

func readLockOrBuild(paths []string, lockPath string) (ecoLock, []byte, error) {
	if lockPath != "" {
		raw, err := os.ReadFile(lockPath)
		if err != nil {
			return ecoLock{}, nil, err
		}
		lock, err := decodeEcoLock(raw)
		if err != nil {
			return ecoLock{}, nil, err
		}
		return lock, raw, nil
	}
	manifests, err := parseCapsuleArgs(paths)
	if err != nil {
		return ecoLock{}, nil, err
	}
	if err := validateCapsuleGraph(manifests, ""); err != nil {
		return ecoLock{}, nil, err
	}
	lock, err := buildEcoLockWithArtifactHashes(manifests)
	if err != nil {
		return ecoLock{}, nil, err
	}
	raw, err := json.Marshal(lock)
	if err != nil {
		return ecoLock{}, nil, err
	}
	return lock, raw, nil
}

func writeCapsuleManifestsFromLock(dir string, lock ecoLock) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, capsule := range lock.Capsules {
		lines := []string{
			fmt.Sprintf("manifest %q", lock.ManifestSchema),
			fmt.Sprintf("capsule %s:", capsule.Name),
			fmt.Sprintf("    id %q", capsule.ID),
			fmt.Sprintf("    version %q", capsule.Version),
		}
		for _, target := range capsule.Targets {
			lines = append(lines, fmt.Sprintf("    target %q", target))
		}
		for _, permission := range capsule.Permissions {
			lines = append(lines, fmt.Sprintf("    permission %q", permission))
		}
		if len(capsule.Policy) > 0 {
			lines = append(lines, "    policy:")
			for _, key := range sortedPolicyKeys(capsule.Policy) {
				lines = append(lines, fmt.Sprintf("        %s %s", key, capsule.Policy[key]))
			}
		}
		for _, dep := range capsule.Dependencies {
			if dep.Path != "" {
				lines = append(lines, fmt.Sprintf("    dependency %q %q %q", dep.ID, dep.Version, dep.Path))
			} else {
				lines = append(lines, fmt.Sprintf("    dependency %q %q", dep.ID, dep.Version))
			}
		}
		lines = append(lines, "")
		path := filepath.Join(dir, capsule.Name+compiler.T4SourceExtension)
		if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func lockGraphFingerprint(items []ecoLockCapsule) string {
	var b strings.Builder
	for _, item := range items {
		b.WriteString(item.ID)
		b.WriteByte('|')
		b.WriteString(item.Version)
		b.WriteByte('|')
		b.WriteString(strings.Join(item.Targets, ","))
		b.WriteByte('|')
		b.WriteString(strings.Join(item.Permissions, ","))
		b.WriteByte('|')
		b.WriteString(policyFingerprint(item.Policy))
		b.WriteByte('|')
		for _, dep := range item.Dependencies {
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
		for _, artifact := range item.Artifacts {
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

func sortEcoLockArtifacts(artifacts []ecoLockArtifact) {
	sort.Slice(artifacts, func(i, j int) bool {
		if artifacts[i].Kind == artifacts[j].Kind {
			if artifacts[i].Target == artifacts[j].Target {
				return artifacts[i].Path < artifacts[j].Path
			}
			return artifacts[i].Target < artifacts[j].Target
		}
		return artifacts[i].Kind < artifacts[j].Kind
	})
}

func copySortedPolicy(policy map[string]string) map[string]string {
	if len(policy) == 0 {
		return nil
	}
	out := make(map[string]string, len(policy))
	for _, key := range sortedPolicyKeys(policy) {
		out[key] = policy[key]
	}
	return out
}

func sortedPolicyKeys(policy map[string]string) []string {
	keys := make([]string, 0, len(policy))
	for key := range policy {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func policyFingerprint(policy map[string]string) string {
	if len(policy) == 0 {
		return ""
	}
	var b strings.Builder
	for _, key := range sortedPolicyKeys(policy) {
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteString(policy[key])
		b.WriteByte(',')
	}
	return b.String()
}
