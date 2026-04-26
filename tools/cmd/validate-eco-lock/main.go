package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	ctarget "tetra_language/compiler/target"
)

type ecoLockEnvelope struct {
	CapsulesRaw json.RawMessage `json:"capsules"`
	Capsules    []ecoLockCapsule
}

type ecoLockCapsule struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Version      string              `json:"version"`
	Path         string              `json:"path"`
	Targets      []string            `json:"targets"`
	Effects      []string            `json:"effects,omitempty"`
	Dependencies []ecoLockDependency `json:"dependencies,omitempty"`
}

type ecoLockDependency struct {
	ID      string `json:"id"`
	Version string `json:"version"`
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

func main() {
	var lockPath string
	flag.StringVar(&lockPath, "lock", "", "path to tetra eco lock JSON")
	flag.Parse()

	if lockPath == "" {
		fmt.Fprintln(os.Stderr, "error: --lock is required")
		os.Exit(2)
	}
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := validateEcoLock(raw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateEcoLock(raw []byte) error {
	var lock ecoLockEnvelope
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&lock); err != nil {
		return err
	}
	if err := unmarshalCapsules(lock.CapsulesRaw, &lock.Capsules); err != nil {
		return err
	}
	if len(lock.Capsules) == 0 {
		return fmt.Errorf("capsules must not be empty")
	}
	byID := map[string]ecoLockCapsule{}
	normalized := make([]ecoLockCapsule, 0, len(lock.Capsules))
	for _, capsule := range lock.Capsules {
		checked, err := validateCapsule(capsule)
		if err != nil {
			return err
		}
		if _, exists := byID[checked.ID]; exists {
			return fmt.Errorf("duplicate capsule id %s", checked.ID)
		}
		byID[checked.ID] = checked
		normalized = append(normalized, checked)
	}
	for _, capsule := range normalized {
		seenDeps := map[string]bool{}
		for _, dep := range capsule.Dependencies {
			if dep.ID == "" {
				return fmt.Errorf("capsule %s has dependency with empty id", capsule.ID)
			}
			if !strings.HasPrefix(dep.ID, "tetra://") {
				return fmt.Errorf("capsule %s dependency %s must use tetra:// prefix", capsule.ID, dep.ID)
			}
			if dep.Version == "" || !isCapsuleSemver(dep.Version) {
				return fmt.Errorf("capsule %s dependency %s has invalid semver version %s", capsule.ID, dep.ID, dep.Version)
			}
			if dep.ID == capsule.ID {
				return fmt.Errorf("capsule %s cannot depend on itself", capsule.ID)
			}
			if seenDeps[dep.ID] {
				return fmt.Errorf("capsule %s has duplicate dependency %s", capsule.ID, dep.ID)
			}
			seenDeps[dep.ID] = true
			resolved, ok := byID[dep.ID]
			if !ok {
				return fmt.Errorf("capsule %s references unknown dependency %s", capsule.ID, dep.ID)
			}
			if resolved.Version != dep.Version {
				return fmt.Errorf("capsule %s dependency %s version mismatch: wants %s, lock has %s", capsule.ID, dep.ID, dep.Version, resolved.Version)
			}
			for _, effect := range resolved.Effects {
				if !containsString(capsule.Effects, effect) {
					return fmt.Errorf("capsule %s missing required effect %s for dependency %s", capsule.ID, effect, dep.ID)
				}
			}
		}
	}
	return nil
}

func unmarshalCapsules(raw json.RawMessage, out *[]ecoLockCapsule) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return fmt.Errorf("capsules must be an array")
	}
	if bytes.Equal(trimmed, []byte("null")) || trimmed[0] != '[' {
		return fmt.Errorf("capsules must be an array, not null")
	}
	var capsuleItems []json.RawMessage
	if err := json.Unmarshal(trimmed, &capsuleItems); err != nil {
		return fmt.Errorf("capsules: %w", err)
	}
	capsules := make([]ecoLockCapsule, 0, len(capsuleItems))
	for i, item := range capsuleItems {
		decoder := json.NewDecoder(bytes.NewReader(item))
		decoder.DisallowUnknownFields()
		var capsule ecoLockCapsule
		if err := decoder.Decode(&capsule); err != nil {
			return fmt.Errorf("capsules[%d]: %w", i, err)
		}
		capsules = append(capsules, capsule)
	}
	*out = capsules
	return nil
}

func validateCapsule(capsule ecoLockCapsule) (ecoLockCapsule, error) {
	if capsule.ID == "" {
		return ecoLockCapsule{}, fmt.Errorf("capsule missing id")
	}
	if !strings.HasPrefix(capsule.ID, "tetra://") {
		return ecoLockCapsule{}, fmt.Errorf("capsule %s id must use tetra:// prefix", capsule.ID)
	}
	if capsule.Name == "" {
		return ecoLockCapsule{}, fmt.Errorf("capsule %s missing name", capsule.ID)
	}
	if capsule.Version == "" || !isCapsuleSemver(capsule.Version) {
		return ecoLockCapsule{}, fmt.Errorf("capsule %s version must use semver x.y.z", capsule.ID)
	}
	if capsule.Path == "" {
		return ecoLockCapsule{}, fmt.Errorf("capsule %s missing path", capsule.ID)
	}
	if len(capsule.Targets) == 0 {
		return ecoLockCapsule{}, fmt.Errorf("capsule %s missing targets", capsule.ID)
	}
	seenTargets := map[string]bool{}
	normalizedEffects := make([]string, 0, len(capsule.Effects))
	supportedTargets := map[string]bool{}
	for _, triple := range ctarget.SupportedTriples() {
		supportedTargets[triple] = true
	}
	for _, target := range capsule.Targets {
		if target == "" {
			return ecoLockCapsule{}, fmt.Errorf("capsule %s has empty target", capsule.ID)
		}
		if !supportedTargets[target] {
			return ecoLockCapsule{}, fmt.Errorf("capsule %s has unsupported target %s", capsule.ID, target)
		}
		if seenTargets[target] {
			return ecoLockCapsule{}, fmt.Errorf("capsule %s has duplicate target %s", capsule.ID, target)
		}
		seenTargets[target] = true
	}
	seenEffects := map[string]bool{}
	for _, effect := range capsule.Effects {
		normalized, err := normalizeCapsuleEffect(effect)
		if err != nil {
			return ecoLockCapsule{}, fmt.Errorf("capsule %s %v", capsule.ID, err)
		}
		if seenEffects[normalized] {
			return ecoLockCapsule{}, fmt.Errorf("capsule %s has duplicate effect %s", capsule.ID, normalized)
		}
		seenEffects[normalized] = true
		normalizedEffects = append(normalizedEffects, normalized)
	}
	capsule.Effects = normalizedEffects
	return capsule, nil
}

func normalizeCapsuleEffect(name string) (string, error) {
	normalized, ok := knownCapsuleEffects[name]
	if !ok {
		return "", fmt.Errorf("has unknown effect %s", name)
	}
	return normalized, nil
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
