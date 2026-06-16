package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"tetra_language/compiler"
	"tetra_language/internal/outputformat"
)

type ecoNeedMap struct {
	Schema     string           `json:"schema"`
	LockSHA256 string           `json:"lock_sha256,omitempty"`
	Capsules   []ecoNeedMapNode `json:"capsules"`
	Edges      []ecoNeedMapEdge `json:"edges,omitempty"`
	Targets    []string         `json:"targets,omitempty"`
}

type ecoNeedMapNode struct {
	ID                string   `json:"id"`
	Version           string   `json:"version"`
	Targets           []string `json:"targets,omitempty"`
	Permissions       []string `json:"permissions,omitempty"`
	TransitiveNeedIDs []string `json:"transitive_need_ids,omitempty"`
}

type ecoNeedMapEdge struct {
	FromID  string `json:"from_id"`
	ToID    string `json:"to_id"`
	Version string `json:"version"`
}

type ecoTrustSnapshot struct {
	Schema        string                    `json:"schema"`
	GeneratedUnix int64                     `json:"generated_at_unix"`
	LockSHA256    string                    `json:"lock_sha256,omitempty"`
	VaultSHA256   string                    `json:"vault_sha256,omitempty"`
	RecordCount   int                       `json:"record_count"`
	Capsules      []ecoTrustSnapshotCapsule `json:"capsules"`
}

type ecoTrustSnapshotCapsule struct {
	ID           string   `json:"id"`
	Version      string   `json:"version"`
	Permissions  []string `json:"permissions,omitempty"`
	TrustTier    string   `json:"trust_tier"`
	TrustScore   int      `json:"trust_score"`
	TrustReasons []string `json:"trust_reasons,omitempty"`
}

func runEcoNeedMap(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco needmap", flag.ContinueOnError)
	fs.SetOutput(stderr)
	lockPath := fs.String("lock", "", "path to lock JSON input")
	outPath := fs.String("o", compiler.DefaultNeedMapName, "path to NeedMap output")
	format := fs.String("format", outputformat.JSON, "NeedMap output format: json, toon, or both")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	lock, rawLock, err := readLockOrBuild(fs.Args(), *lockPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	needMap := buildNeedMap(lock, rawLock)
	if _, err := writeEcoStructuredFile(*outPath, *format, needMap); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "NeedMap written: %s\n", *outPath)
	return 0
}

func runEcoTrust(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: tetra eco trust <snapshot> [options]")
		return 2
	}
	switch args[0] {
	case "snapshot":
		return runEcoTrustSnapshot(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown eco trust command %q\n", args[0])
		return 2
	}
}

func runEcoTrustSnapshot(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco trust snapshot", flag.ContinueOnError)
	fs.SetOutput(stderr)
	lockPath := fs.String("lock", "", "path to lock JSON input")
	store := fs.String("store", ".tetra/todex-vault", "path to local Todex vault store")
	outPath := fs.String("o", "tetra.trust-snapshot.json", "path to trust snapshot output")
	format := fs.String("format", outputformat.JSON, "trust snapshot output format: json, toon, or both")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *lockPath == "" {
		fmt.Fprintln(stderr, "eco trust snapshot requires --lock")
		return 2
	}
	lockRaw, err := os.ReadFile(*lockPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	lock, err := decodeEcoLock(lockRaw)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	lockHash := sha256.Sum256(lockRaw)
	index, err := readVaultIndex(*store)
	if err != nil {
		fmt.Fprintf(stderr, "read vault store: %v\n", err)
		return 1
	}
	vaultRaw, err := os.ReadFile(vaultIndexPath(*store))
	if err != nil {
		fmt.Fprintf(stderr, "read vault store: %v\n", err)
		return 1
	}
	vaultHash := sha256.Sum256(vaultRaw)
	snapshot := ecoTrustSnapshot{
		Schema:        ecoTrustSnapshotSchemaV1,
		GeneratedUnix: 0,
		LockSHA256:    "sha256:" + hex.EncodeToString(lockHash[:]),
		VaultSHA256:   "sha256:" + hex.EncodeToString(vaultHash[:]),
		RecordCount:   len(index.Records),
		Capsules:      make([]ecoTrustSnapshotCapsule, 0, len(lock.Capsules)),
	}
	for _, capsule := range lock.Capsules {
		score, tier, reasons := scoreCapsuleTrust(capsule.Permissions)
		snapshot.Capsules = append(snapshot.Capsules, ecoTrustSnapshotCapsule{
			ID:           capsule.ID,
			Version:      capsule.Version,
			Permissions:  append([]string(nil), capsule.Permissions...),
			TrustTier:    tier,
			TrustScore:   score,
			TrustReasons: reasons,
		})
	}
	sort.Slice(snapshot.Capsules, func(i, j int) bool { return snapshot.Capsules[i].ID < snapshot.Capsules[j].ID })
	if _, err := writeEcoStructuredFile(*outPath, *format, snapshot); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Trust snapshot written: %s\n", *outPath)
	return 0
}

func buildNeedMap(lock ecoLock, rawLock []byte) ecoNeedMap {
	needMap := ecoNeedMap{
		Schema:   ecoNeedMapSchemaV1,
		Capsules: make([]ecoNeedMapNode, 0, len(lock.Capsules)),
		Edges:    []ecoNeedMapEdge{},
	}
	if len(rawLock) > 0 {
		sum := sha256.Sum256(rawLock)
		needMap.LockSHA256 = "sha256:" + hex.EncodeToString(sum[:])
	}
	byID := map[string]ecoLockCapsule{}
	targetSet := map[string]struct{}{}
	for _, capsule := range lock.Capsules {
		byID[capsule.ID] = capsule
		for _, target := range capsule.Targets {
			targetSet[target] = struct{}{}
		}
	}
	for _, capsule := range lock.Capsules {
		transitive := collectTransitiveNeeds(capsule.ID, byID, map[string]bool{})
		sort.Strings(transitive)
		node := ecoNeedMapNode{
			ID:                capsule.ID,
			Version:           capsule.Version,
			Targets:           append([]string(nil), capsule.Targets...),
			Permissions:       append([]string(nil), capsule.Permissions...),
			TransitiveNeedIDs: transitive,
		}
		needMap.Capsules = append(needMap.Capsules, node)
		for _, dep := range capsule.Dependencies {
			needMap.Edges = append(needMap.Edges, ecoNeedMapEdge{
				FromID:  capsule.ID,
				ToID:    dep.ID,
				Version: dep.Version,
			})
		}
	}
	sort.Slice(needMap.Capsules, func(i, j int) bool { return needMap.Capsules[i].ID < needMap.Capsules[j].ID })
	sort.Slice(needMap.Edges, func(i, j int) bool {
		if needMap.Edges[i].FromID == needMap.Edges[j].FromID {
			if needMap.Edges[i].ToID == needMap.Edges[j].ToID {
				return needMap.Edges[i].Version < needMap.Edges[j].Version
			}
			return needMap.Edges[i].ToID < needMap.Edges[j].ToID
		}
		return needMap.Edges[i].FromID < needMap.Edges[j].FromID
	})
	for target := range targetSet {
		needMap.Targets = append(needMap.Targets, target)
	}
	sort.Strings(needMap.Targets)
	return needMap
}

func collectTransitiveNeeds(id string, byID map[string]ecoLockCapsule, seen map[string]bool) []string {
	capsule, ok := byID[id]
	if !ok {
		return nil
	}
	var out []string
	for _, dep := range capsule.Dependencies {
		if seen[dep.ID] {
			continue
		}
		seen[dep.ID] = true
		out = append(out, dep.ID)
		out = append(out, collectTransitiveNeeds(dep.ID, byID, seen)...)
	}
	return dedupeStrings(out)
}

func scoreCapsuleTrust(permissions []string) (score int, tier string, reasons []string) {
	score = 100
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
	switch {
	case score >= 80:
		tier = "high"
	case score >= 60:
		tier = "medium"
	default:
		tier = "low"
	}
	reasons = append(reasons, fmt.Sprintf("permissions=%s", strings.Join(permissions, ",")))
	return score, tier, reasons
}
