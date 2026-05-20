package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"tetra_language/compiler"
)

type ecoSeed struct {
	Schema        string        `json:"schema"`
	GeneratedUnix int64         `json:"generated_at_unix"`
	Lock          ecoLock       `json:"lock"`
	Capsules      []ecoSeedItem `json:"capsules"`
}

type ecoSeedItem struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Version     string              `json:"version"`
	Targets     []string            `json:"targets,omitempty"`
	Effects     []string            `json:"effects,omitempty"`
	Permissions []string            `json:"permissions,omitempty"`
	DependsOn   []capsuleDependency `json:"depends_on,omitempty"`
}

func runEcoSeed(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: tetra eco seed <export|import> [options]")
		return 2
	}
	switch args[0] {
	case "export":
		return runEcoSeedExport(args[1:], stdout, stderr)
	case "import":
		return runEcoSeedImport(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown eco seed command %q\n", args[0])
		return 2
	}
}

func runEcoSeedExport(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco seed export", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", "", "validate capsule target compatibility")
	outPath := fs.String("out", compiler.DefaultSeedFileName, "path to T4 seed output")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	manifests, err := parseCapsuleArgs(fs.Args())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := validateCapsuleGraph(manifests, *target); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	lock := buildEcoLock(manifests)
	seed := ecoSeed{
		Schema:        ecoSeedSchemaV1,
		GeneratedUnix: 0,
		Lock:          lock,
		Capsules:      make([]ecoSeedItem, 0, len(lock.Capsules)),
	}
	for _, capsule := range lock.Capsules {
		seed.Capsules = append(seed.Capsules, ecoSeedItem{
			ID:          capsule.ID,
			Name:        capsule.Name,
			Version:     capsule.Version,
			Targets:     append([]string(nil), capsule.Targets...),
			Effects:     append([]string(nil), capsule.Effects...),
			Permissions: append([]string(nil), capsule.Permissions...),
			DependsOn:   append([]capsuleDependency(nil), capsule.Dependencies...),
		})
	}
	if err := writeJSONFile(*outPath, seed); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Seed exported: %s\n", *outPath)
	return 0
}

func runEcoSeedImport(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco seed import", flag.ContinueOnError)
	fs.SetOutput(stderr)
	seedPath := fs.String("seed", "", "path to seed JSON input")
	lockPath := fs.String("lock", "", "path to write lock JSON")
	capsulesDir := fs.String("capsules-dir", "", "directory to write capsule manifests")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *seedPath == "" {
		if fs.NArg() == 1 {
			*seedPath = fs.Arg(0)
		} else {
			fmt.Fprintln(stderr, "eco seed import requires --seed")
			return 2
		}
	}
	raw, err := os.ReadFile(*seedPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	var seed ecoSeed
	if err := json.Unmarshal(raw, &seed); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if seed.Schema != "" && seed.Schema != ecoSeedSchemaV1 {
		fmt.Fprintf(stderr, "unsupported seed schema %q\n", seed.Schema)
		return 1
	}
	lock := seed.Lock
	if len(lock.Capsules) == 0 && len(seed.Capsules) > 0 {
		lock = ecoLock{
			Schema:           ecoLockSchemaV1,
			ManifestSchema:   capsuleManifestSchemaV1,
			PermissionsModel: ecoPermissionsModelV1,
			GeneratedUnix:    0,
			Capsules:         make([]ecoLockCapsule, 0, len(seed.Capsules)),
		}
		for _, item := range seed.Capsules {
			lock.Capsules = append(lock.Capsules, ecoLockCapsule{
				ID:           item.ID,
				Name:         item.Name,
				Version:      item.Version,
				Path:         filepath.Clean(item.Name + compiler.T4SourceExtension),
				Targets:      sortedStrings(item.Targets),
				Effects:      sortedStrings(item.Effects),
				Permissions:  sortedStrings(item.Permissions),
				Dependencies: append([]capsuleDependency(nil), item.DependsOn...),
			})
		}
	}
	normalizeLock(&lock)
	if len(lock.Capsules) == 0 {
		fmt.Fprintln(stderr, "seed contains no capsules")
		return 1
	}
	if err := validateEcoLockModel(lock); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *lockPath == "" && *capsulesDir == "" {
		fmt.Fprintln(stderr, "eco seed import requires --lock and/or --capsules-dir")
		return 2
	}
	if *lockPath != "" {
		if err := writeJSONFile(*lockPath, lock); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	if *capsulesDir != "" {
		if err := writeCapsuleManifestsFromLock(*capsulesDir, lock); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	fmt.Fprintf(stdout, "Seed imported: %s\n", *seedPath)
	return 0
}
