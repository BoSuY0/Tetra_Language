package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	ctarget "tetra_language/compiler/target"
)

type capsuleManifest struct {
	ManifestSchema string
	Name           string
	ID             string
	Version        string
	Path           string
	Targets        []string
	Effects        []string
	Permissions    []string
	Dependencies   []capsuleDependency
}

type capsuleDependency struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

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
}

type ecoPackageMetadata struct {
	Schema           string                   `json:"schema"`
	Compression      string                   `json:"compression"`
	MTimeUnix        int64                    `json:"mtime_unix"`
	Reproducible     bool                     `json:"reproducible,omitempty"`
	BuildInputsSHA   string                   `json:"build_inputs_sha256,omitempty"`
	ManifestSchema   string                   `json:"manifest_schema,omitempty"`
	PermissionsModel string                   `json:"permissions_model,omitempty"`
	FileCount        int                      `json:"file_count"`
	Files            []ecoPackageMetadataFile `json:"files"`
}

type ecoPackageMetadataFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type vaultRecord struct {
	Hash   string `json:"hash"`
	Kind   string `json:"kind"`
	Source string `json:"source"`
	Size   int64  `json:"size"`
}

type vaultIndex struct {
	Records []vaultRecord `json:"records"`
}

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

type ecoMaterialization struct {
	Schema          string `json:"schema"`
	Target          string `json:"target"`
	PackagePath     string `json:"package_path"`
	MaterializedDir string `json:"materialized_dir"`
	TrustSnapshot   string `json:"trust_snapshot,omitempty"`
	LockSHA256      string `json:"lock_sha256,omitempty"`
}

type ecoPublishMetadata struct {
	Schema        string                 `json:"schema"`
	Channel       string                 `json:"channel"`
	Hub           string                 `json:"hub"`
	PublishedUnix int64                  `json:"published_at_unix"`
	Capsule       ecoPublishCapsule      `json:"capsule"`
	Package       ecoPublishPackage      `json:"package"`
	Trust         *ecoPublishTrust       `json:"trust,omitempty"`
	Downloads     []ecoPublishDownload   `json:"downloads,omitempty"`
	Extra         map[string]interface{} `json:"extra,omitempty"`
}

type ecoPublishCapsule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Target      string   `json:"target"`
	Targets     []string `json:"targets,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

type ecoPublishPackage struct {
	File   string `json:"file"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type ecoPublishTrust struct {
	SnapshotFile string `json:"snapshot_file"`
	SnapshotHash string `json:"snapshot_sha256"`
	TrustTier    string `json:"trust_tier"`
}

type ecoPublishDownload struct {
	Target string `json:"target"`
	Path   string `json:"path"`
}

const (
	capsuleManifestSchemaV1  = "tetra.capsule.v1"
	ecoPermissionsModelV1    = "tetra.eco.permissions.v1"
	ecoLockSchemaV1          = "tetra.eco.lock.v1"
	ecoSeedSchemaV1          = "tetra.eco.seed.v1"
	ecoNeedMapSchemaV1       = "tetra.eco.needmap.v1"
	ecoTrustSnapshotSchemaV1 = "tetra.eco.trust-snapshot.v1"
	ecoMaterializerSchemaV1  = "tetra.eco.materialization.v1"
	ecoPublishSchemaV1Beta   = "tetra.eco.publish.v1beta"
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
	"actors":       "actors",
	"alloc":        "alloc",
	"cap.io":       "io",
	"cap.mem":      "mem",
	"capability":   "capability",
	"control":      "control",
	"io":           "io",
	"io.read":      "io",
	"io.write":     "io",
	"islands":      "islands",
	"link":         "link",
	"mem":          "mem",
	"mem.read":     "mem",
	"mem.write":    "mem",
	"mmio":         "mmio",
	"runtime":      "runtime",
	"runtime.exec": "runtime",
}

func runEco(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: tetra eco <verify|pack|unpack|vault|seed|needmap|trust|materialize|publish|download|tetrahub> [options]")
		return 2
	}
	switch args[0] {
	case "verify":
		return runEcoVerify(args[1:], stdout, stderr)
	case "pack":
		return runEcoPack(args[1:], stdout, stderr)
	case "unpack":
		return runEcoUnpack(args[1:], stdout, stderr)
	case "vault":
		return runEcoVault(args[1:], stdout, stderr)
	case "seed":
		return runEcoSeed(args[1:], stdout, stderr)
	case "needmap":
		return runEcoNeedMap(args[1:], stdout, stderr)
	case "trust":
		return runEcoTrust(args[1:], stdout, stderr)
	case "materialize":
		return runEcoMaterialize(args[1:], stdout, stderr)
	case "publish":
		return runEcoPublish(args[1:], stdout, stderr)
	case "download":
		return runEcoDownload(args[1:], stdout, stderr)
	case "tetrahub":
		return runEcoTetraHub(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown eco command %q\n", args[0])
		return 2
	}
}

func runEcoVault(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: tetra eco vault <add|list|verify> [options]")
		return 2
	}
	switch args[0] {
	case "add":
		return runEcoVaultAdd(args[1:], stdout, stderr)
	case "list":
		return runEcoVaultList(args[1:], stdout, stderr)
	case "verify":
		return runEcoVaultVerify(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown eco vault command %q\n", args[0])
		return 2
	}
}

func runEcoVerify(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", "", "validate capsule target compatibility")
	lockPath := fs.String("lock", "", "write dependency lock/provenance JSON")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	paths := fs.Args()
	if len(paths) == 0 {
		paths = []string{"Tetra.capsule"}
	}
	manifests := make([]capsuleManifest, 0, len(paths))
	for _, path := range paths {
		manifest, err := parseCapsule(path)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		manifests = append(manifests, manifest)
	}
	if err := validateCapsuleGraph(manifests, *target); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *lockPath != "" {
		if err := writeEcoLock(*lockPath, manifests); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	if len(manifests) == 1 {
		manifest := manifests[0]
		fmt.Fprintf(stdout, "Capsule OK: %s %s (%s)\n", manifest.Name, manifest.Version, manifest.ID)
		return 0
	}
	fmt.Fprintf(stdout, "Capsule graph OK: %d capsules\n", len(manifests))
	return 0
}

func runEcoPack(args []string, stdout io.Writer, stderr io.Writer) int {
	capsulePath, outPath, project, err := parseEcoPackArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	manifest, err := parseCapsule(capsulePath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if outPath == "" {
		outPath = manifest.Name + ".todex"
	}
	if project {
		if err := packCapsuleProject(manifest.Path, outPath); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	} else if err := packCapsule(manifest.Path, outPath); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Packed: %s\n", outPath)
	return 0
}

func runEcoUnpack(args []string, stdout io.Writer, stderr io.Writer) int {
	pkgPath, outDir, err := parseEcoUnpackArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if outDir == "" {
		outDir = "."
	}
	if err := unpackCapsule(pkgPath, outDir); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Unpacked: %s\n", outDir)
	return 0
}

func runEcoVaultAdd(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco vault add", flag.ContinueOnError)
	fs.SetOutput(stderr)
	store := fs.String("store", ".tetra/todex-vault", "local Todex vault directory")
	kind := fs.String("kind", "source", "record kind: source, interface, build, or test")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "eco vault add requires one file path")
		return 2
	}
	if !validVaultKind(*kind) {
		fmt.Fprintf(stderr, "unsupported vault kind %q\n", *kind)
		return 2
	}
	record, err := addVaultRecord(*store, fs.Arg(0), *kind)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Vault added: %s %s %s\n", record.Hash, record.Kind, record.Source)
	return 0
}

func runEcoVaultList(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco vault list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	store := fs.String("store", ".tetra/todex-vault", "local Todex vault directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "eco vault list does not accept positional arguments")
		return 2
	}
	index, err := readVaultIndex(*store)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	sortVaultRecords(index.Records)
	for _, record := range index.Records {
		fmt.Fprintf(stdout, "%s %s %s %d\n", record.Hash, record.Kind, record.Source, record.Size)
	}
	return 0
}

func runEcoVaultVerify(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco vault verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	store := fs.String("store", ".tetra/todex-vault", "local Todex vault directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "eco vault verify does not accept positional arguments")
		return 2
	}
	index, err := readVaultIndex(*store)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	for _, record := range index.Records {
		if err := verifyVaultRecord(*store, record); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	fmt.Fprintf(stdout, "Vault OK: %d records\n", len(index.Records))
	return 0
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
	outPath := fs.String("out", "tetra.seed.json", "path to seed JSON output")
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
				Path:         filepath.Clean(item.Name + ".capsule"),
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

func runEcoNeedMap(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco needmap", flag.ContinueOnError)
	fs.SetOutput(stderr)
	lockPath := fs.String("lock", "", "path to lock JSON input")
	outPath := fs.String("o", "tetra.needmap.json", "path to NeedMap JSON output")
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
	if err := writeJSONFile(*outPath, needMap); err != nil {
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
	index, _ := readVaultIndex(*store)
	vaultRaw := []byte("{}")
	if raw, err := os.ReadFile(vaultIndexPath(*store)); err == nil {
		vaultRaw = raw
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
	if err := writeJSONFile(*outPath, snapshot); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Trust snapshot written: %s\n", *outPath)
	return 0
}

func runEcoMaterialize(args []string, stdout io.Writer, stderr io.Writer) int {
	pkgPath, target, trustPath, outDir, err := parseEcoMaterializeArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if err := unpackCapsule(pkgPath, outDir); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	manifest, err := parseCapsule(filepath.Join(outDir, "Tetra.capsule"))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if target != "" && len(manifest.Targets) > 0 && !containsString(manifest.Targets, target) {
		fmt.Fprintf(stderr, "target mismatch for %s: does not support %s\n", manifest.ID, target)
		return 1
	}
	meta := ecoMaterialization{
		Schema:          ecoMaterializerSchemaV1,
		Target:          target,
		PackagePath:     filepath.Clean(pkgPath),
		MaterializedDir: filepath.Clean(outDir),
	}
	if trustPath != "" {
		raw, err := os.ReadFile(trustPath)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		var snapshot ecoTrustSnapshot
		if err := json.Unmarshal(raw, &snapshot); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if snapshot.Schema != "" && snapshot.Schema != ecoTrustSnapshotSchemaV1 {
			fmt.Fprintf(stderr, "unsupported trust snapshot schema %q\n", snapshot.Schema)
			return 1
		}
		meta.TrustSnapshot = filepath.Clean(trustPath)
		meta.LockSHA256 = snapshot.LockSHA256
	}
	if err := writeJSONFile(filepath.Join(outDir, "tetra.materialization.json"), meta); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Materialized: %s\n", outDir)
	return 0
}

func runEcoPublish(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco publish", flag.ContinueOnError)
	fs.SetOutput(stderr)
	pkgPath := fs.String("package", "", "path to .todex package")
	registry := fs.String("registry", ".tetra/registry-beta", "path to local beta registry")
	target := fs.String("target", "", "target triple to publish")
	trustPath := fs.String("trust", "", "optional trust snapshot file")
	hub := fs.String("hub", "local-beta", "hub routing label")
	channel := fs.String("channel", "beta", "publishing channel")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *pkgPath == "" {
		if fs.NArg() != 1 {
			fmt.Fprintln(stderr, "eco publish requires --package or one package path")
			return 2
		}
		*pkgPath = fs.Arg(0)
	}
	if *channel != "beta" {
		fmt.Fprintf(stderr, "eco publish currently supports beta channel only, got %q\n", *channel)
		return 2
	}
	metaPath, err := publishPackage(*pkgPath, *registry, *target, *trustPath, *hub, *channel)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Published (beta): %s\n", metaPath)
	return 0
}

func runEcoDownload(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco download", flag.ContinueOnError)
	fs.SetOutput(stderr)
	id := fs.String("id", "", "capsule id")
	version := fs.String("version", "", "capsule version")
	target := fs.String("target", "", "target triple")
	registry := fs.String("registry", ".tetra/registry-beta", "path to local beta registry")
	outPath := fs.String("o", "", "output package path")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *id == "" || *version == "" {
		fmt.Fprintln(stderr, "eco download requires --id and --version")
		return 2
	}
	path, err := downloadPackage(*registry, *id, *version, *target, *outPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Downloaded: %s\n", path)
	return 0
}

func runEcoTetraHub(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: tetra eco tetrahub <publish|download> [options]")
		return 2
	}
	switch args[0] {
	case "publish":
		return runEcoTetraHubPublish(args[1:], stdout, stderr)
	case "download":
		return runEcoTetraHubDownload(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown eco tetrahub command %q\n", args[0])
		return 2
	}
}

func runEcoTetraHubPublish(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco tetrahub publish", flag.ContinueOnError)
	fs.SetOutput(stderr)
	pkgPath := fs.String("package", "", "path to .todex package")
	store := fs.String("store", ".tetra/tetrahub-beta", "path to local TetraHub beta store")
	target := fs.String("target", "", "target triple to publish")
	trustPath := fs.String("trust", "", "optional trust snapshot file")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *pkgPath == "" {
		if fs.NArg() != 1 {
			fmt.Fprintln(stderr, "eco tetrahub publish requires --package or one package path")
			return 2
		}
		*pkgPath = fs.Arg(0)
	}
	metaPath, err := publishPackage(*pkgPath, *store, *target, *trustPath, "tetrahub-beta", "beta")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "TetraHub beta published: %s\n", metaPath)
	return 0
}

func runEcoTetraHubDownload(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco tetrahub download", flag.ContinueOnError)
	fs.SetOutput(stderr)
	store := fs.String("store", ".tetra/tetrahub-beta", "path to local TetraHub beta store")
	id := fs.String("id", "", "capsule id")
	version := fs.String("version", "", "capsule version")
	target := fs.String("target", "", "target triple")
	outPath := fs.String("o", "", "output package path")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *id == "" || *version == "" {
		fmt.Fprintln(stderr, "eco tetrahub download requires --id and --version")
		return 2
	}
	path, err := downloadPackage(*store, *id, *version, *target, *outPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "TetraHub beta downloaded: %s\n", path)
	return 0
}

func parseCapsule(path string) (capsuleManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return capsuleManifest{}, err
	}
	manifest := capsuleManifest{
		Path:           path,
		ManifestSchema: capsuleManifestSchemaV1,
	}
	var (
		sawManifest bool
		sawName     bool
		sawID       bool
		sawVersion  bool
	)
	for i, line := range strings.Split(string(raw), "\n") {
		content := strings.TrimSpace(line)
		if content == "" || strings.HasPrefix(content, "//") || strings.HasPrefix(content, "#") {
			continue
		}
		if strings.HasPrefix(content, "manifest ") {
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
		if strings.HasPrefix(content, "target ") {
			value, err := parseCapsuleString(path, i+1, strings.TrimSpace(strings.TrimPrefix(content, "target ")))
			if err != nil {
				return capsuleManifest{}, err
			}
			if !isSupportedCapsuleTarget(value) {
				return capsuleManifest{}, fmt.Errorf("%s:%d: unsupported target %s", path, i+1, value)
			}
			if containsString(manifest.Targets, value) {
				return capsuleManifest{}, fmt.Errorf("%s:%d: duplicate target %s", path, i+1, value)
			}
			manifest.Targets = append(manifest.Targets, value)
			continue
		}
		if strings.HasPrefix(content, "effect ") {
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
			dep, err := parseCapsuleDependency(path, i+1, strings.TrimSpace(strings.TrimPrefix(content, "dependency ")))
			if err != nil {
				return capsuleManifest{}, err
			}
			manifest.Dependencies = append(manifest.Dependencies, dep)
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
	return manifest, nil
}

func parseCapsuleDependency(path string, line int, value string) (capsuleDependency, error) {
	fields, err := splitQuotedFields(value)
	if err != nil {
		return capsuleDependency{}, fmt.Errorf("%s:%d: %v", path, line, err)
	}
	if len(fields) != 2 {
		return capsuleDependency{}, fmt.Errorf("%s:%d: dependency expects quoted id and version", path, line)
	}
	if !strings.HasPrefix(fields[0], "tetra://") {
		return capsuleDependency{}, fmt.Errorf("%s:%d: dependency id must use tetra:// prefix", path, line)
	}
	if !isCapsuleSemver(fields[1]) {
		return capsuleDependency{}, fmt.Errorf("%s:%d: dependency version must use semver x.y.z", path, line)
	}
	return capsuleDependency{ID: fields[0], Version: fields[1]}, nil
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

func writeEcoLock(path string, manifests []capsuleManifest) error {
	lock := buildEcoLock(manifests)
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
		}
		sort.Slice(item.Dependencies, func(i, j int) bool {
			if item.Dependencies[i].ID == item.Dependencies[j].ID {
				return item.Dependencies[i].Version < item.Dependencies[j].Version
			}
			return item.Dependencies[i].ID < item.Dependencies[j].ID
		})
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
	sum := sha256.Sum256([]byte(lockGraphFingerprint(items)))
	lock.GraphSHA256 = "sha256:" + hex.EncodeToString(sum[:])
	return lock
}

func parseCapsuleArgs(paths []string) ([]capsuleManifest, error) {
	if len(paths) == 0 {
		paths = []string{"Tetra.capsule"}
	}
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

func decodeEcoLock(raw []byte) (ecoLock, error) {
	var lock ecoLock
	if err := json.Unmarshal(raw, &lock); err != nil {
		return ecoLock{}, err
	}
	normalizeLock(&lock)
	return lock, nil
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
			item.Path = filepath.Clean(item.Name + ".capsule")
		}
		for _, effect := range item.Effects {
			item.Permissions = appendUniqueString(item.Permissions, effect)
		}
		sort.Strings(item.Effects)
		sort.Strings(item.Permissions)
		sort.Strings(item.Targets)
		sort.Slice(item.Dependencies, func(i, j int) bool {
			if item.Dependencies[i].ID == item.Dependencies[j].ID {
				return item.Dependencies[i].Version < item.Dependencies[j].Version
			}
			return item.Dependencies[i].ID < item.Dependencies[j].ID
		})
	}
	sort.Slice(lock.Capsules, func(i, j int) bool { return lock.Capsules[i].ID < lock.Capsules[j].ID })
	if lock.GraphSHA256 == "" {
		sum := sha256.Sum256([]byte(lockGraphFingerprint(lock.Capsules)))
		lock.GraphSHA256 = "sha256:" + hex.EncodeToString(sum[:])
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
	lock := buildEcoLock(manifests)
	raw, err := json.Marshal(lock)
	if err != nil {
		return ecoLock{}, nil, err
	}
	return lock, raw, nil
}

func writeJSONFile(path string, v interface{}) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
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
		for _, dep := range capsule.Dependencies {
			lines = append(lines, fmt.Sprintf("    dependency %q %q", dep.ID, dep.Version))
		}
		lines = append(lines, "")
		path := filepath.Join(dir, capsule.Name+".capsule")
		if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
			return err
		}
	}
	return nil
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

func publishPackage(pkgPath string, registry string, target string, trustPath string, hub string, channel string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "tetra-eco-publish-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)
	if err := unpackCapsule(pkgPath, tmpDir); err != nil {
		return "", err
	}
	manifest, err := parseCapsule(filepath.Join(tmpDir, "Tetra.capsule"))
	if err != nil {
		return "", err
	}
	if target == "" {
		if len(manifest.Targets) > 0 {
			target = manifest.Targets[0]
		} else {
			target = "any"
		}
	}
	if len(manifest.Targets) > 0 && !containsString(manifest.Targets, target) {
		return "", fmt.Errorf("target mismatch for %s: does not support %s", manifest.ID, target)
	}
	pkgRaw, err := os.ReadFile(pkgPath)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(pkgRaw)
	targetDir := filepath.Join(registry, "packages", capsuleIDDirectory(manifest.ID), manifest.Version, target)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}
	pkgOutPath := filepath.Join(targetDir, "package.todex")
	if err := os.WriteFile(pkgOutPath, pkgRaw, 0o644); err != nil {
		return "", err
	}
	meta := ecoPublishMetadata{
		Schema:        ecoPublishSchemaV1Beta,
		Channel:       channel,
		Hub:           hub,
		PublishedUnix: 0,
		Capsule: ecoPublishCapsule{
			ID:          manifest.ID,
			Name:        manifest.Name,
			Version:     manifest.Version,
			Target:      target,
			Targets:     append([]string(nil), manifest.Targets...),
			Permissions: append([]string(nil), manifest.Permissions...),
		},
		Package: ecoPublishPackage{
			File:   "package.todex",
			Size:   int64(len(pkgRaw)),
			SHA256: "sha256:" + hex.EncodeToString(sum[:]),
		},
		Downloads: []ecoPublishDownload{
			{Target: target, Path: filepath.ToSlash(filepath.Join("packages", capsuleIDDirectory(manifest.ID), manifest.Version, target, "package.todex"))},
		},
	}
	if trustPath != "" {
		raw, err := os.ReadFile(trustPath)
		if err != nil {
			return "", err
		}
		hash := sha256.Sum256(raw)
		tier := "unknown"
		var snapshot ecoTrustSnapshot
		if err := json.Unmarshal(raw, &snapshot); err == nil {
			for _, capsule := range snapshot.Capsules {
				if capsule.ID == manifest.ID && capsule.Version == manifest.Version {
					tier = capsule.TrustTier
					break
				}
			}
		}
		meta.Trust = &ecoPublishTrust{
			SnapshotFile: filepath.Clean(trustPath),
			SnapshotHash: "sha256:" + hex.EncodeToString(hash[:]),
			TrustTier:    tier,
		}
	}
	metaPath := filepath.Join(targetDir, "metadata.json")
	if err := writeJSONFile(metaPath, meta); err != nil {
		return "", err
	}
	return metaPath, nil
}

func downloadPackage(registry string, id string, version string, target string, outPath string) (string, error) {
	baseDir := filepath.Join(registry, "packages", capsuleIDDirectory(id), version)
	if target == "" {
		entries, err := os.ReadDir(baseDir)
		if err != nil {
			return "", err
		}
		var candidates []string
		for _, entry := range entries {
			if entry.IsDir() {
				candidates = append(candidates, entry.Name())
			}
		}
		sort.Strings(candidates)
		if len(candidates) == 0 {
			return "", fmt.Errorf("no targets available for %s %s", id, version)
		}
		target = candidates[0]
	}
	targetDir := filepath.Join(baseDir, target)
	metaPath := filepath.Join(targetDir, "metadata.json")
	rawMeta, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			entries, readErr := os.ReadDir(baseDir)
			if readErr != nil {
				return "", err
			}
			var available []string
			for _, entry := range entries {
				if entry.IsDir() {
					available = append(available, entry.Name())
				}
			}
			sort.Strings(available)
			return "", fmt.Errorf("target %s not available for %s %s (available: %s)", target, id, version, strings.Join(available, ", "))
		}
		return "", err
	}
	var meta ecoPublishMetadata
	if err := json.Unmarshal(rawMeta, &meta); err != nil {
		return "", err
	}
	if meta.Schema != ecoPublishSchemaV1Beta || meta.Channel != "beta" {
		return "", fmt.Errorf("unsupported publish metadata in %s", metaPath)
	}
	pkgPath := filepath.Join(targetDir, meta.Package.File)
	if outPath == "" {
		outPath = fmt.Sprintf("%s-%s-%s.todex", capsuleIDDirectory(id), version, target)
	}
	raw, err := os.ReadFile(pkgPath)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(outPath, raw, 0o644); err != nil {
		return "", err
	}
	return outPath, nil
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
		for _, dep := range item.Dependencies {
			b.WriteString(dep.ID)
			b.WriteByte('@')
			b.WriteString(dep.Version)
			b.WriteByte(',')
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func packageMetadataFingerprint(files []ecoPackageMetadataFile) string {
	var b strings.Builder
	for _, file := range files {
		b.WriteString(file.Path)
		b.WriteByte('|')
		b.WriteString(file.SHA256)
		b.WriteByte('|')
		b.WriteString(strconv.FormatInt(file.Size, 10))
		b.WriteByte('\n')
	}
	return b.String()
}

func dedupeStrings(values []string) []string {
	if len(values) <= 1 {
		return values
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func validVaultKind(kind string) bool {
	switch kind {
	case "source", "interface", "build", "test":
		return true
	default:
		return false
	}
}

func addVaultRecord(store string, path string, kind string) (vaultRecord, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return vaultRecord{}, err
	}
	sum := sha256.Sum256(raw)
	hashHex := fmt.Sprintf("%x", sum[:])
	objectPath := vaultObjectPath(store, hashHex)
	if err := os.MkdirAll(filepath.Dir(objectPath), 0o755); err != nil {
		return vaultRecord{}, err
	}
	if _, err := os.Stat(objectPath); err != nil {
		if !os.IsNotExist(err) {
			return vaultRecord{}, err
		}
		if err := os.WriteFile(objectPath, raw, 0o644); err != nil {
			return vaultRecord{}, err
		}
	}
	record := vaultRecord{
		Hash:   "sha256:" + hashHex,
		Kind:   kind,
		Source: filepath.Clean(path),
		Size:   int64(len(raw)),
	}
	index, err := readVaultIndex(store)
	if err != nil {
		return vaultRecord{}, err
	}
	index.Records = upsertVaultRecord(index.Records, record)
	if err := writeVaultIndex(store, index); err != nil {
		return vaultRecord{}, err
	}
	return record, nil
}

func upsertVaultRecord(records []vaultRecord, record vaultRecord) []vaultRecord {
	for i, existing := range records {
		if existing.Hash == record.Hash && existing.Kind == record.Kind && existing.Source == record.Source {
			records[i] = record
			sortVaultRecords(records)
			return records
		}
	}
	records = append(records, record)
	sortVaultRecords(records)
	return records
}

func readVaultIndex(store string) (vaultIndex, error) {
	path := vaultIndexPath(store)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return vaultIndex{}, nil
		}
		return vaultIndex{}, err
	}
	var index vaultIndex
	if err := json.Unmarshal(raw, &index); err != nil {
		return vaultIndex{}, err
	}
	sortVaultRecords(index.Records)
	return index, nil
}

func writeVaultIndex(store string, index vaultIndex) error {
	sortVaultRecords(index.Records)
	raw, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	path := vaultIndexPath(store)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func verifyVaultRecord(store string, record vaultRecord) error {
	const prefix = "sha256:"
	if !strings.HasPrefix(record.Hash, prefix) {
		return fmt.Errorf("vault record %s has unsupported hash", record.Source)
	}
	hashHex := strings.TrimPrefix(record.Hash, prefix)
	raw, err := os.ReadFile(vaultObjectPath(store, hashHex))
	if err != nil {
		return err
	}
	sum := sha256.Sum256(raw)
	actual := fmt.Sprintf("%x", sum[:])
	if actual != hashHex {
		return fmt.Errorf("vault object mismatch for %s", record.Source)
	}
	if int64(len(raw)) != record.Size {
		return fmt.Errorf("vault object size mismatch for %s", record.Source)
	}
	return nil
}

func vaultIndexPath(store string) string {
	return filepath.Join(store, "records.json")
}

func vaultObjectPath(store string, hashHex string) string {
	return filepath.Join(store, "objects", "sha256", hashHex)
}

func sortVaultRecords(records []vaultRecord) {
	sort.Slice(records, func(i, j int) bool {
		if records[i].Hash == records[j].Hash {
			if records[i].Kind == records[j].Kind {
				return records[i].Source < records[j].Source
			}
			return records[i].Kind < records[j].Kind
		}
		return records[i].Hash < records[j].Hash
	})
}

func appendUniqueString(values []string, value string) []string {
	if containsString(values, value) {
		return values
	}
	return append(values, value)
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func sortedStrings(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
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

func isSupportedCapsuleTarget(target string) bool {
	for _, triple := range ctarget.SupportedTriples() {
		if triple == target {
			return true
		}
	}
	return false
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

func parseEcoPackArgs(args []string) (capsulePath string, outPath string, project bool, err error) {
	capsulePath = "Tetra.capsule"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--project":
			project = true
		case "-o", "--out":
			i++
			if i >= len(args) {
				return "", "", false, fmt.Errorf("%s requires a value", args[i-1])
			}
			outPath = args[i]
		default:
			if strings.HasPrefix(args[i], "-") {
				return "", "", false, fmt.Errorf("unknown option %s", args[i])
			}
			if capsulePath != "Tetra.capsule" {
				return "", "", false, fmt.Errorf("eco pack accepts one capsule path")
			}
			capsulePath = args[i]
		}
	}
	return capsulePath, outPath, project, nil
}

func parseEcoUnpackArgs(args []string) (pkgPath string, outDir string, err error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-C", "--dir":
			i++
			if i >= len(args) {
				return "", "", fmt.Errorf("%s requires a value", args[i-1])
			}
			outDir = args[i]
		default:
			if strings.HasPrefix(args[i], "-") {
				return "", "", fmt.Errorf("unknown option %s", args[i])
			}
			if pkgPath != "" {
				return "", "", fmt.Errorf("eco unpack accepts one package path")
			}
			pkgPath = args[i]
		}
	}
	if pkgPath == "" {
		return "", "", fmt.Errorf("eco unpack requires a package path")
	}
	return pkgPath, outDir, nil
}

func parseEcoMaterializeArgs(args []string) (pkgPath string, target string, trustPath string, outDir string, err error) {
	outDir = "."
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--target":
			i++
			if i >= len(args) {
				return "", "", "", "", fmt.Errorf("--target requires a value")
			}
			target = args[i]
		case "--trust":
			i++
			if i >= len(args) {
				return "", "", "", "", fmt.Errorf("--trust requires a value")
			}
			trustPath = args[i]
		case "-C", "--dir":
			i++
			if i >= len(args) {
				return "", "", "", "", fmt.Errorf("%s requires a value", args[i-1])
			}
			outDir = args[i]
		default:
			if strings.HasPrefix(args[i], "-") {
				return "", "", "", "", fmt.Errorf("unknown option %s", args[i])
			}
			if pkgPath != "" {
				return "", "", "", "", fmt.Errorf("eco materialize requires one package path")
			}
			pkgPath = args[i]
		}
	}
	if pkgPath == "" {
		return "", "", "", "", fmt.Errorf("eco materialize requires one package path")
	}
	return pkgPath, target, trustPath, outDir, nil
}

func packCapsule(capsulePath string, outPath string) error {
	return packFiles(filepath.Dir(capsulePath), []string{filepath.Base(capsulePath)}, outPath)
}

func packCapsuleProject(capsulePath string, outPath string) error {
	root := filepath.Dir(capsulePath)
	var relPaths []string
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == ".tetra_cache" || name == "tetra_cache" {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if filepath.Clean(path) == filepath.Clean(outPath) || strings.HasSuffix(rel, ".todex") {
			return nil
		}
		relPaths = append(relPaths, rel)
		return nil
	}); err != nil {
		return err
	}
	sort.Strings(relPaths)
	return packFiles(root, relPaths, outPath)
}

func packFiles(root string, relPaths []string, outPath string) error {
	const packageMetadataFile = "tetra.package.json"
	zeroTime := time.Unix(0, 0).UTC()
	cleanRelPaths := append([]string(nil), relPaths...)
	sort.Strings(cleanRelPaths)
	for _, rel := range cleanRelPaths {
		if filepath.ToSlash(rel) == packageMetadataFile {
			return fmt.Errorf("project already contains reserved file %s", packageMetadataFile)
		}
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()
	gz := gzip.NewWriter(out)
	gz.ModTime = zeroTime
	gz.OS = 255
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	metadata := ecoPackageMetadata{
		Schema:           "tetra.eco.package.v1",
		Compression:      "gzip",
		MTimeUnix:        0,
		Reproducible:     true,
		ManifestSchema:   capsuleManifestSchemaV1,
		PermissionsModel: ecoPermissionsModelV1,
		Files:            make([]ecoPackageMetadataFile, 0, len(cleanRelPaths)),
	}
	for _, rel := range cleanRelPaths {
		if rel == "" || strings.HasPrefix(filepath.Clean(rel), "..") || filepath.IsAbs(rel) {
			return fmt.Errorf("unsafe archive path %q", rel)
		}
		path := filepath.Join(root, rel)
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		info, err := in.Stat()
		if err != nil {
			_ = in.Close()
			return err
		}
		header := &tar.Header{
			Name:       filepath.ToSlash(rel),
			Mode:       0o644,
			Size:       info.Size(),
			Format:     tar.FormatPAX,
			ModTime:    zeroTime,
			AccessTime: zeroTime,
			ChangeTime: zeroTime,
			Uid:        0,
			Gid:        0,
			Uname:      "",
			Gname:      "",
		}
		if err := tw.WriteHeader(header); err != nil {
			_ = in.Close()
			return err
		}
		hash := sha256.New()
		if _, err := io.Copy(tw, io.TeeReader(in, hash)); err != nil {
			_ = in.Close()
			return err
		}
		if err := in.Close(); err != nil {
			return err
		}
		metadata.Files = append(metadata.Files, ecoPackageMetadataFile{
			Path:   filepath.ToSlash(rel),
			SHA256: "sha256:" + hex.EncodeToString(hash.Sum(nil)),
			Size:   info.Size(),
		})
	}
	metadata.FileCount = len(metadata.Files)
	sum := sha256.Sum256([]byte(packageMetadataFingerprint(metadata.Files)))
	metadata.BuildInputsSHA = "sha256:" + hex.EncodeToString(sum[:])
	rawMetadata, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	rawMetadata = append(rawMetadata, '\n')
	if err := tw.WriteHeader(&tar.Header{
		Name:       packageMetadataFile,
		Mode:       0o644,
		Size:       int64(len(rawMetadata)),
		Format:     tar.FormatPAX,
		ModTime:    zeroTime,
		AccessTime: zeroTime,
		ChangeTime: zeroTime,
		Uid:        0,
		Gid:        0,
		Uname:      "",
		Gname:      "",
	}); err != nil {
		return err
	}
	if _, err := tw.Write(rawMetadata); err != nil {
		return err
	}
	return nil
}

func unpackCapsule(pkgPath string, outDir string) error {
	in, err := os.Open(pkgPath)
	if err != nil {
		return err
	}
	defer in.Close()
	gz, err := gzip.NewReader(in)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(header.Name)
		if name == "." || strings.HasPrefix(name, "..") || filepath.IsAbs(name) {
			return fmt.Errorf("unsafe archive path %q", header.Name)
		}
		outPath := filepath.Join(outDir, name)
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			_ = out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
	}
}
