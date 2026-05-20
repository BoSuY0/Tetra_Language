package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"tetra_language/compiler"
)

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

func parseCapsule(path string) (capsuleManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return capsuleManifest{}, err
	}
	manifest := capsuleManifest{
		Path:           path,
		ManifestSchema: capsuleManifestSchemaV1,
		Policy:         map[string]string{},
	}
	var (
		sawManifest bool
		sawName     bool
		sawID       bool
		sawVersion  bool
		section     string
	)
	for i, line := range strings.Split(string(raw), "\n") {
		content := strings.TrimSpace(line)
		if content == "" || strings.HasPrefix(content, "//") || strings.HasPrefix(content, "#") {
			continue
		}
		if nextSection, ok := capsuleSectionHeader(content); ok {
			section = nextSection
			continue
		}
		if strings.HasPrefix(content, "manifest ") {
			section = ""
			if sawManifest {
				return capsuleManifest{}, fmt.Errorf("%s:%d: duplicate manifest field", path, i+1)
			}
			value, err := parseCapsuleString(path, i+1, strings.TrimSpace(strings.TrimPrefix(content, "manifest ")))
			if err != nil {
				return capsuleManifest{}, err
			}
			if value != capsuleManifestSchemaV1 {
				return capsuleManifest{}, fmt.Errorf("%s:%d: unsupported manifest schema %s", path, i+1, value)
			}
			manifest.ManifestSchema = value
			sawManifest = true
			continue
		}
		if strings.HasPrefix(content, "capsule ") {
			section = ""
			if sawName {
				return capsuleManifest{}, fmt.Errorf("%s:%d: duplicate capsule declaration", path, i+1)
			}
			name := strings.TrimSpace(strings.TrimPrefix(content, "capsule "))
			name = strings.TrimSuffix(name, ":")
			if name == "" {
				return capsuleManifest{}, fmt.Errorf("%s:%d: capsule name is required", path, i+1)
			}
			manifest.Name = name
			sawName = true
			continue
		}
		if strings.HasPrefix(content, "id ") {
			section = ""
			if sawID {
				return capsuleManifest{}, fmt.Errorf("%s:%d: duplicate id field", path, i+1)
			}
			value, err := parseCapsuleString(path, i+1, strings.TrimSpace(strings.TrimPrefix(content, "id ")))
			if err != nil {
				return capsuleManifest{}, err
			}
			if !strings.HasPrefix(value, "tetra://") {
				return capsuleManifest{}, fmt.Errorf("%s:%d: capsule id must use tetra:// prefix", path, i+1)
			}
			manifest.ID = value
			sawID = true
			continue
		}
		if strings.HasPrefix(content, "version ") {
			section = ""
			if sawVersion {
				return capsuleManifest{}, fmt.Errorf("%s:%d: duplicate version field", path, i+1)
			}
			value, err := parseCapsuleString(path, i+1, strings.TrimSpace(strings.TrimPrefix(content, "version ")))
			if err != nil {
				return capsuleManifest{}, err
			}
			if !isCapsuleSemver(value) {
				return capsuleManifest{}, fmt.Errorf("%s:%d: capsule version must use semver x.y.z", path, i+1)
			}
			manifest.Version = value
			sawVersion = true
			continue
		}
		if strings.HasPrefix(content, "entry ") {
			section = ""
			value, err := parseCapsuleBareOrQuoted(path, i+1, strings.TrimSpace(strings.TrimPrefix(content, "entry ")))
			if err != nil {
				return capsuleManifest{}, err
			}
			manifest.Entry = filepath.ToSlash(filepath.Clean(value))
			continue
		}
		if strings.HasPrefix(content, "source ") {
			section = ""
			value, err := parseCapsuleBareOrQuoted(path, i+1, strings.TrimSpace(strings.TrimPrefix(content, "source ")))
			if err != nil {
				return capsuleManifest{}, err
			}
			manifest.SourceRoots = appendCapsuleSourceRoot(manifest.SourceRoots, value)
			continue
		}
		if strings.HasPrefix(content, "target ") {
			section = ""
			value, err := parseCapsuleBareOrQuoted(path, i+1, strings.TrimSpace(strings.TrimPrefix(content, "target ")))
			if err != nil {
				return capsuleManifest{}, err
			}
			if err := appendCapsuleTarget(path, i+1, &manifest, value); err != nil {
				return capsuleManifest{}, err
			}
			continue
		}
		if strings.HasPrefix(content, "effect ") {
			section = ""
			value, err := parseCapsuleString(path, i+1, strings.TrimSpace(strings.TrimPrefix(content, "effect ")))
			if err != nil {
				return capsuleManifest{}, err
			}
			normalized, err := normalizeCapsuleEffect(value)
			if err != nil {
				return capsuleManifest{}, fmt.Errorf("%s:%d: %v", path, i+1, err)
			}
			if containsString(manifest.Effects, normalized) {
				return capsuleManifest{}, fmt.Errorf("%s:%d: duplicate effect %s", path, i+1, normalized)
			}
			manifest.Effects = append(manifest.Effects, normalized)
			manifest.Permissions = appendUniqueString(manifest.Permissions, normalized)
			continue
		}
		if strings.HasPrefix(content, "permission ") {
			section = ""
			value, err := parseCapsuleString(path, i+1, strings.TrimSpace(strings.TrimPrefix(content, "permission ")))
			if err != nil {
				return capsuleManifest{}, err
			}
			normalized, err := normalizeCapsulePermission(value)
			if err != nil {
				return capsuleManifest{}, fmt.Errorf("%s:%d: %v", path, i+1, err)
			}
			if containsString(manifest.Permissions, normalized) {
				return capsuleManifest{}, fmt.Errorf("%s:%d: duplicate permission %s", path, i+1, normalized)
			}
			manifest.Permissions = append(manifest.Permissions, normalized)
			continue
		}
		if strings.HasPrefix(content, "dependency ") {
			section = ""
			dep, err := parseCapsuleDependency(path, i+1, strings.TrimSpace(strings.TrimPrefix(content, "dependency ")))
			if err != nil {
				return capsuleManifest{}, err
			}
			manifest.Dependencies = append(manifest.Dependencies, dep)
			continue
		}
		if strings.HasPrefix(content, "artifact ") {
			section = ""
			artifact, err := parseCapsuleArtifact(path, i+1, strings.TrimSpace(strings.TrimPrefix(content, "artifact ")))
			if err != nil {
				return capsuleManifest{}, err
			}
			if err := appendCapsuleArtifact(path, i+1, &manifest, artifact); err != nil {
				return capsuleManifest{}, err
			}
			continue
		}
		if section != "" {
			if err := parseCapsuleSectionLine(path, i+1, section, content, &manifest); err != nil {
				return capsuleManifest{}, err
			}
			continue
		}
		return capsuleManifest{}, fmt.Errorf("%s:%d: unknown capsule field", path, i+1)
	}
	if manifest.Name == "" {
		return capsuleManifest{}, fmt.Errorf("%s: missing capsule declaration", path)
	}
	if manifest.ID == "" {
		return capsuleManifest{}, fmt.Errorf("%s: missing capsule id", path)
	}
	if manifest.Version == "" {
		return capsuleManifest{}, fmt.Errorf("%s: missing capsule version", path)
	}
	for _, effect := range manifest.Effects {
		manifest.Permissions = appendUniqueString(manifest.Permissions, effect)
	}
	sort.Strings(manifest.Permissions)
	sort.Strings(manifest.Effects)
	sort.Strings(manifest.SourceRoots)
	return manifest, nil
}

func capsuleSectionHeader(content string) (string, bool) {
	switch strings.TrimSuffix(content, ":") {
	case "targets":
		return "targets", true
	case "deps":
		return "deps", true
	case "allow":
		return "allow", true
	case "policy":
		return "policy", true
	case "sources":
		return "sources", true
	case "artifacts":
		return "artifacts", true
	default:
		return "", false
	}
}

func parseCapsuleSectionLine(path string, line int, section string, content string, manifest *capsuleManifest) error {
	switch section {
	case "targets":
		return appendCapsuleTarget(path, line, manifest, content)
	case "deps":
		dep, err := parseCapsuleDependencyFields(path, line, strings.Fields(content))
		if err != nil {
			return err
		}
		manifest.Dependencies = append(manifest.Dependencies, dep)
		return nil
	case "allow":
		normalized, err := normalizeCapsulePermission(content)
		if err != nil {
			return fmt.Errorf("%s:%d: %v", path, line, err)
		}
		if containsString(manifest.Permissions, normalized) {
			return fmt.Errorf("%s:%d: duplicate permission %s", path, line, normalized)
		}
		manifest.Permissions = append(manifest.Permissions, normalized)
		return nil
	case "policy":
		fields := strings.Fields(content)
		if len(fields) != 2 {
			return fmt.Errorf("%s:%d: policy expects key and value", path, line)
		}
		return setCapsulePolicy(path, line, manifest, fields[0], fields[1])
	case "sources":
		manifest.SourceRoots = appendCapsuleSourceRoot(manifest.SourceRoots, content)
		return nil
	case "artifacts":
		artifact, err := parseCapsuleArtifact(path, line, content)
		if err != nil {
			return err
		}
		return appendCapsuleArtifact(path, line, manifest, artifact)
	default:
		return fmt.Errorf("%s:%d: unknown capsule section %s", path, line, section)
	}
}

func appendCapsuleTarget(path string, line int, manifest *capsuleManifest, value string) error {
	normalized, err := normalizeCapsuleTarget(value)
	if err != nil {
		return fmt.Errorf("%s:%d: %v", path, line, err)
	}
	if containsString(manifest.Targets, normalized) {
		return fmt.Errorf("%s:%d: duplicate target %s", path, line, normalized)
	}
	manifest.Targets = append(manifest.Targets, normalized)
	return nil
}

func appendCapsuleSourceRoot(roots []string, value string) []string {
	clean := filepath.ToSlash(filepath.Clean(value))
	if clean == "." {
		clean = ""
	}
	if clean == "" || strings.HasPrefix(clean, "../") || clean == ".." || filepath.IsAbs(clean) {
		return roots
	}
	return appendUniqueString(roots, clean)
}

func setCapsulePolicy(path string, line int, manifest *capsuleManifest, key string, value string) error {
	if manifest.Policy == nil {
		manifest.Policy = map[string]string{}
	}
	if err := validateCapsulePolicyValue(key, value); err != nil {
		return fmt.Errorf("%s:%d: %v", path, line, err)
	}
	if _, exists := manifest.Policy[key]; exists {
		return fmt.Errorf("%s:%d: duplicate policy %s", path, line, key)
	}
	manifest.Policy[key] = value
	return nil
}

func parseCapsuleDependency(path string, line int, value string) (capsuleDependency, error) {
	fields, err := splitQuotedFields(value)
	if err != nil {
		return capsuleDependency{}, fmt.Errorf("%s:%d: %v", path, line, err)
	}
	return parseCapsuleDependencyFields(path, line, fields)
}

func parseCapsuleDependencyFields(path string, line int, fields []string) (capsuleDependency, error) {
	if len(fields) != 2 && len(fields) != 3 {
		return capsuleDependency{}, fmt.Errorf("%s:%d: dependency expects id, version, and optional path", path, line)
	}
	id := fields[0]
	if !strings.HasPrefix(id, "tetra://") {
		id = "tetra://" + id
	}
	if !isCapsuleSemver(fields[1]) {
		return capsuleDependency{}, fmt.Errorf("%s:%d: dependency version must use semver x.y.z", path, line)
	}
	dep := capsuleDependency{ID: id, Version: fields[1]}
	if len(fields) == 3 {
		dep.Path = filepath.ToSlash(filepath.Clean(fields[2]))
	}
	return dep, nil
}

func parseCapsuleArtifact(path string, line int, value string) (capsuleArtifact, error) {
	fields, err := parseCapsuleArtifactFields(value)
	if err != nil {
		return capsuleArtifact{}, fmt.Errorf("%s:%d: %v", path, line, err)
	}
	if len(fields) != 2 && len(fields) != 3 {
		return capsuleArtifact{}, fmt.Errorf("%s:%d: artifact expects kind, optional target, and path", path, line)
	}
	kind, err := normalizeCapsuleArtifactKind(fields[0])
	if err != nil {
		return capsuleArtifact{}, fmt.Errorf("%s:%d: %v", path, line, err)
	}
	target := ""
	pathField := fields[1]
	if len(fields) == 3 {
		if kind != "object" {
			return capsuleArtifact{}, fmt.Errorf("%s:%d: only object artifacts accept a target", path, line)
		}
		target, err = normalizeCapsuleTarget(fields[1])
		if err != nil {
			return capsuleArtifact{}, fmt.Errorf("%s:%d: %v", path, line, err)
		}
		pathField = fields[2]
	}
	rel, err := cleanCapsuleArtifactPath(pathField)
	if err != nil {
		return capsuleArtifact{}, fmt.Errorf("%s:%d: %v", path, line, err)
	}
	if err := validateCapsuleArtifactExtension(kind, rel); err != nil {
		return capsuleArtifact{}, fmt.Errorf("%s:%d: %v", path, line, err)
	}
	return capsuleArtifact{Kind: kind, Target: target, Path: rel}, nil
}

func parseCapsuleArtifactFields(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("artifact expects kind and path")
	}
	if strings.Contains(value, "\"") {
		return splitQuotedFields(value)
	}
	return strings.Fields(value), nil
}

func appendCapsuleArtifact(path string, line int, manifest *capsuleManifest, artifact capsuleArtifact) error {
	for _, existing := range manifest.Artifacts {
		if existing.Kind == artifact.Kind && existing.Target == artifact.Target && existing.Path == artifact.Path {
			return fmt.Errorf("%s:%d: duplicate artifact %s %s", path, line, artifact.Kind, artifact.Path)
		}
	}
	manifest.Artifacts = append(manifest.Artifacts, artifact)
	return nil
}

func splitQuotedFields(value string) ([]string, error) {
	var out []string
	rest := strings.TrimSpace(value)
	for rest != "" {
		if !strings.HasPrefix(rest, "\"") {
			return nil, fmt.Errorf("expected quoted string")
		}
		end := 1
		escaped := false
		for ; end < len(rest); end++ {
			ch := rest[end]
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				break
			}
		}
		if end >= len(rest) {
			return nil, fmt.Errorf("unterminated quoted string")
		}
		parsed, err := strconv.Unquote(rest[:end+1])
		if err != nil {
			return nil, fmt.Errorf("expected quoted string")
		}
		if parsed == "" {
			return nil, fmt.Errorf("string must not be empty")
		}
		out = append(out, parsed)
		rest = strings.TrimSpace(rest[end+1:])
	}
	return out, nil
}

func validateCapsuleGraph(manifests []capsuleManifest, target string) error {
	byID := make(map[string]capsuleManifest, len(manifests))
	for _, manifest := range manifests {
		if _, exists := byID[manifest.ID]; exists {
			return fmt.Errorf("duplicate capsule id %q", manifest.ID)
		}
		if manifest.ManifestSchema != capsuleManifestSchemaV1 {
			return fmt.Errorf("%s: unsupported manifest schema %s", manifest.Path, manifest.ManifestSchema)
		}
		if target != "" && len(manifest.Targets) > 0 && !containsString(manifest.Targets, target) {
			return fmt.Errorf("%s: target mismatch for %s: does not support %s", manifest.Path, manifest.ID, target)
		}
		seenEffects := map[string]struct{}{}
		for _, effect := range manifest.Effects {
			if _, exists := seenEffects[effect]; exists {
				return fmt.Errorf("%s: duplicate effect %s", manifest.Path, effect)
			}
			seenEffects[effect] = struct{}{}
		}
		seenPermissions := map[string]struct{}{}
		for _, permission := range manifest.Permissions {
			if _, exists := seenPermissions[permission]; exists {
				return fmt.Errorf("%s: duplicate permission %s", manifest.Path, permission)
			}
			seenPermissions[permission] = struct{}{}
		}
		if err := validateCapsulePolicy(manifest.Policy); err != nil {
			return fmt.Errorf("%s: %v", manifest.Path, err)
		}
		seenDeps := map[string]struct{}{}
		for _, dep := range manifest.Dependencies {
			key := dep.ID + "\x00" + dep.Version
			if _, exists := seenDeps[key]; exists {
				return fmt.Errorf("%s: duplicate dependency %s %s", manifest.Path, dep.ID, dep.Version)
			}
			seenDeps[key] = struct{}{}
		}
		byID[manifest.ID] = manifest
	}
	for _, manifest := range manifests {
		for _, dep := range manifest.Dependencies {
			found, ok := byID[dep.ID]
			if !ok {
				return fmt.Errorf("%s: missing dependency %s %s", manifest.Path, dep.ID, dep.Version)
			}
			if found.Version != dep.Version {
				return fmt.Errorf("%s: dependency %s version mismatch: want %s, got %s", manifest.Path, dep.ID, dep.Version, found.Version)
			}
			for _, effect := range found.Effects {
				if !containsString(manifest.Effects, effect) {
					return fmt.Errorf("%s: missing required effect %s for dependency %s", manifest.Path, effect, dep.ID)
				}
			}
			for _, permission := range found.Permissions {
				if !containsString(manifest.Permissions, permission) {
					return fmt.Errorf("%s: missing required permission %s for dependency %s", manifest.Path, permission, dep.ID)
				}
			}
		}
	}
	return nil
}

func parseCapsuleArgs(paths []string) ([]capsuleManifest, error) {
	paths = defaultCapsuleArgs(paths)
	manifests := make([]capsuleManifest, 0, len(paths))
	for _, path := range paths {
		manifest, err := parseCapsule(path)
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, manifest)
	}
	return manifests, nil
}

func parseCapsuleGraphArgs(paths []string) ([]capsuleManifest, error) {
	paths = defaultCapsuleArgs(paths)
	if len(paths) != 1 {
		return parseCapsuleArgs(paths)
	}
	manifest, err := parseCapsule(paths[0])
	if err != nil {
		return nil, err
	}
	root := filepath.Dir(paths[0])
	root, err = filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	_, depManifests, err := projectDependencyGraph(root, manifest, map[string]int{root: projectDependencyVisiting}, []string{root})
	if err != nil {
		return nil, err
	}
	manifests := append([]capsuleManifest{manifest}, depManifests...)
	return manifests, nil
}

func defaultCapsuleArgs(paths []string) []string {
	if len(paths) == 0 {
		return []string{defaultCapsulePath()}
	}
	return paths
}

func validateCapsulePolicy(policy map[string]string) error {
	for key, value := range policy {
		if err := validateCapsulePolicyValue(key, value); err != nil {
			return err
		}
	}
	return nil
}

func validateCapsulePolicyValue(key string, value string) error {
	switch key {
	case "unsafe":
		if value != "deny" && value != "allow" {
			return fmt.Errorf("unsafe policy must be deny or allow")
		}
	case "reproducible":
		if value != "required" && value != "preferred" && value != "off" {
			return fmt.Errorf("reproducible policy must be required, preferred, or off")
		}
	default:
		return fmt.Errorf("unknown policy %s", key)
	}
	return nil
}

func normalizeCapsuleEffect(name string) (string, error) {
	normalized, ok := knownCapsuleEffects[name]
	if !ok {
		return "", fmt.Errorf("unknown effect %q", name)
	}
	return normalized, nil
}

func normalizeCapsulePermission(name string) (string, error) {
	normalized, ok := knownCapsulePermissions[name]
	if !ok {
		return "", fmt.Errorf("unknown permission %q", name)
	}
	return normalized, nil
}

func normalizeCapsuleArtifactKind(kind string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "interface", "t4i":
		return "interface", nil
	case "object", "tobj":
		return "object", nil
	case "seed", "t4s":
		return "seed", nil
	default:
		return "", fmt.Errorf("unknown artifact kind %q", kind)
	}
}

func cleanCapsuleArtifactPath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("artifact path is empty")
	}
	if strings.Contains(value, "\\") {
		return "", fmt.Errorf("artifact path must use forward slashes")
	}
	if filepath.IsAbs(value) {
		return "", fmt.Errorf("artifact path must be relative")
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("artifact path must stay inside capsule root")
	}
	return clean, nil
}

func validateCapsuleArtifactExtension(kind string, path string) error {
	switch kind {
	case "interface":
		if filepath.Ext(path) != compiler.T4InterfaceExtension {
			return fmt.Errorf("interface artifact must use %s", compiler.T4InterfaceExtension)
		}
	case "object":
		if filepath.Ext(path) != ".tobj" {
			return fmt.Errorf("object artifact must use .tobj")
		}
	case "seed":
		if filepath.Ext(path) != compiler.T4SeedExtension {
			return fmt.Errorf("seed artifact must use %s", compiler.T4SeedExtension)
		}
	default:
		return fmt.Errorf("unknown artifact kind %q", kind)
	}
	return nil
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

func normalizeCapsuleTarget(target string) (string, error) {
	switch strings.ToLower(target) {
	case "linux":
		target = "linux-x64"
	case "windows":
		target = "windows-x64"
	case "macos", "macosx":
		target = "macos-x64"
	case "web":
		target = "wasm32-web"
	case "wasi":
		target = "wasm32-wasi"
	}
	if !isSupportedCapsuleTarget(target) {
		return "", fmt.Errorf("unsupported target %s", target)
	}
	return target, nil
}

func parseCapsuleString(path string, line int, value string) (string, error) {
	out, err := strconv.Unquote(value)
	if err != nil {
		return "", fmt.Errorf("%s:%d: expected quoted string", path, line)
	}
	if out == "" {
		return "", fmt.Errorf("%s:%d: string must not be empty", path, line)
	}
	return out, nil
}

func parseCapsuleBareOrQuoted(path string, line int, value string) (string, error) {
	if strings.HasPrefix(value, "\"") {
		return parseCapsuleString(path, line, value)
	}
	if value == "" {
		return "", fmt.Errorf("%s:%d: string must not be empty", path, line)
	}
	return value, nil
}
