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
	Dependencies []ecoLockDependency `json:"dependencies,omitempty"`
}

type ecoLockDependency struct {
	ID      string `json:"id"`
	Version string `json:"version"`
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
	if err := json.Unmarshal(raw, &lock); err != nil {
		return err
	}
	if err := unmarshalCapsules(lock.CapsulesRaw, &lock.Capsules); err != nil {
		return err
	}
	if len(lock.Capsules) == 0 {
		return fmt.Errorf("capsules must not be empty")
	}
	byID := map[string]ecoLockCapsule{}
	for _, capsule := range lock.Capsules {
		if err := validateCapsule(capsule); err != nil {
			return err
		}
		if _, exists := byID[capsule.ID]; exists {
			return fmt.Errorf("duplicate capsule id %s", capsule.ID)
		}
		byID[capsule.ID] = capsule
	}
	for _, capsule := range lock.Capsules {
		seenDeps := map[string]bool{}
		for _, dep := range capsule.Dependencies {
			if dep.ID == "" {
				return fmt.Errorf("capsule %s has dependency with empty id", capsule.ID)
			}
			if dep.Version == "" {
				return fmt.Errorf("capsule %s dependency %s has empty version", capsule.ID, dep.ID)
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
	if err := json.Unmarshal(trimmed, out); err != nil {
		return fmt.Errorf("capsules: %w", err)
	}
	return nil
}

func validateCapsule(capsule ecoLockCapsule) error {
	if capsule.ID == "" {
		return fmt.Errorf("capsule missing id")
	}
	if !strings.HasPrefix(capsule.ID, "tetra://") {
		return fmt.Errorf("capsule %s id must use tetra:// prefix", capsule.ID)
	}
	if capsule.Name == "" {
		return fmt.Errorf("capsule %s missing name", capsule.ID)
	}
	if capsule.Version == "" {
		return fmt.Errorf("capsule %s missing version", capsule.ID)
	}
	if capsule.Path == "" {
		return fmt.Errorf("capsule %s missing path", capsule.ID)
	}
	if len(capsule.Targets) == 0 {
		return fmt.Errorf("capsule %s missing targets", capsule.ID)
	}
	seenTargets := map[string]bool{}
	supportedTargets := map[string]bool{}
	for _, triple := range ctarget.SupportedTriples() {
		supportedTargets[triple] = true
	}
	for _, target := range capsule.Targets {
		if target == "" {
			return fmt.Errorf("capsule %s has empty target", capsule.ID)
		}
		if !supportedTargets[target] {
			return fmt.Errorf("capsule %s has unsupported target %s", capsule.ID, target)
		}
		if seenTargets[target] {
			return fmt.Errorf("capsule %s has duplicate target %s", capsule.ID, target)
		}
		seenTargets[target] = true
	}
	return nil
}
