package main

import (
	"archive/tar"
	"bytes"
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

	"tetra_language/compiler"
	ctarget "tetra_language/compiler/target"
)

type capsuleManifest struct {
	ManifestSchema string
	Name           string
	ID             string
	Version        string
	Path           string
	Entry          string
	SourceRoots    []string
	Targets        []string
	Effects        []string
	Permissions    []string
	Dependencies   []capsuleDependency
	Artifacts      []capsuleArtifact
	Policy         map[string]string
}

type capsuleDependency struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Path    string `json:"path,omitempty"`
}

type capsuleArtifact struct {
	Kind   string `json:"kind"`
	Target string `json:"target,omitempty"`
	Path   string `json:"path"`
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
	ecoPackageSchemaV1       = "tetra.eco.package.v1"
	ecoPackageMetadataPath   = "tetra.package.json"
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

func runEco(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: tetra eco <verify|artifacts|pack|unpack|vault|seed|needmap|trust|materialize|publish|download|tetrahub> [options]")
		return 2
	}
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra eco <verify|artifacts|pack|unpack|vault|seed|needmap|trust|materialize|publish|download|tetrahub> [options]")
		fmt.Fprintln(stdout, "lock generation/validation is available through workflows such as: tetra eco verify --lock <path> <Capsule.t4>")
		return 0
	}
	switch args[0] {
	case "verify":
		return runEcoVerify(args[1:], stdout, stderr)
	case "artifacts":
		return runEcoArtifacts(args[1:], stdout, stderr)
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
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra eco vault <add|list|verify> [options]")
		return 0
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
	manifests, err := parseCapsuleGraphArgs(fs.Args())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
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

func runEcoArtifacts(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: tetra eco artifacts <build|check> [options]")
		return 2
	}
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra eco artifacts <build|check> [options]")
		return 0
	}
	switch args[0] {
	case "build":
		return runEcoArtifactsBuild(args[1:], stdout, stderr)
	case "check":
		return runEcoArtifactsCheck(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown eco artifacts command %q\n", args[0])
		return 2
	}
}

func runEcoArtifactsBuild(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco artifacts build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	targetFlag := fs.String("target", "", "native target triple for generated .tobj artifacts")
	lockPath := fs.String("lock", "", "path to write semantic lock; defaults to project Tetra.lock")
	checkOnly := fs.Bool("check", false, "dry-run and report pending artifact changes without writing files")
	allTargets := fs.Bool("all-targets", false, "build artifacts for every native target listed in Capsule.t4")
	jobs := fs.Int("jobs", 1, "parallel module build jobs")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "eco artifacts build accepts at most one Capsule.t4 path")
		return 2
	}
	capsulePath := defaultCapsulePath()
	if fs.NArg() == 1 {
		capsulePath = fs.Arg(0)
	}
	if *checkOnly {
		issues, err := checkCapsuleArtifacts(capsulePath, *targetFlag, *lockPath, *allTargets)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if len(issues) > 0 {
			writeArtifactIssues(stdout, issues, true)
			return 1
		}
		fmt.Fprintf(stdout, "Artifacts current: %s\n", capsulePath)
		return 0
	}
	if err := buildCapsuleArtifacts(capsulePath, capsuleArtifactBuildOptions{
		Target:     *targetFlag,
		LockPath:   *lockPath,
		Jobs:       *jobs,
		AllTargets: *allTargets,
	}); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Artifacts built: %s\n", capsulePath)
	return 0
}

func runEcoArtifactsCheck(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco artifacts check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	targetFlag := fs.String("target", "", "native target triple for object freshness checks")
	lockPath := fs.String("lock", "", "path to semantic lock; defaults to project Tetra.lock")
	allTargets := fs.Bool("all-targets", false, "check artifacts for every native target listed in Capsule.t4")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, "eco artifacts check accepts at most one Capsule.t4 path")
		return 2
	}
	capsulePath := defaultCapsulePath()
	if fs.NArg() == 1 {
		capsulePath = fs.Arg(0)
	}
	issues, err := checkCapsuleArtifacts(capsulePath, *targetFlag, *lockPath, *allTargets)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if len(issues) > 0 {
		writeArtifactIssues(stderr, issues, false)
		return 1
	}
	fmt.Fprintf(stdout, "Artifacts current: %s\n", capsulePath)
	return 0
}

type generatedCapsuleArtifact struct {
	Artifact capsuleArtifact
	Module   string
	Source   string
}

type capsuleSourceModule struct {
	Module string
	Path   string
}

type capsuleArtifactBuildOptions struct {
	Target     string
	LockPath   string
	Jobs       int
	AllTargets bool
}

type capsuleArtifactPlan struct {
	RootManifest        capsuleManifest
	Manifests           []capsuleManifest
	DependencyManifests []capsuleManifest
	Root                string
	Targets             []string
	Expected            []generatedCapsuleArtifact
}

type artifactIssue struct {
	Kind   string
	Module string
	Path   string
	Detail string
	Repair string
}

func buildCapsuleArtifacts(capsulePath string, opt capsuleArtifactBuildOptions) error {
	manifests, err := parseCapsuleGraphArgs([]string{capsulePath})
	if err != nil {
		return err
	}
	if len(manifests) == 0 {
		return fmt.Errorf("%s: no capsule graph", capsulePath)
	}
	rootManifest := manifests[0]
	root := filepath.Dir(rootManifest.Path)
	targets, err := resolveArtifactBuildTargets(rootManifest, opt.Target, opt.AllTargets)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return fmt.Errorf("eco artifacts build requires at least one native object target")
	}
	for _, target := range targets {
		if err := validateCapsuleGraph(manifests, target); err != nil {
			return err
		}
	}
	lockPath := opt.LockPath
	if lockPath == "" {
		lockPath = filepath.Join(root, compiler.SemanticLockFileName)
	}
	jobs := opt.Jobs
	if jobs < 1 {
		jobs = 1
	}

	var generated []generatedCapsuleArtifact
	dependencyManifests := manifests[1:]
	for _, depManifest := range dependencyManifests {
		depRoot := filepath.Dir(depManifest.Path)
		modules, err := capsuleSourceModules(depManifest)
		if err != nil {
			return err
		}
		buildOpt, err := dependencyArtifactBuildOptions(depRoot, depManifest, jobs)
		if err != nil {
			return err
		}
		for _, module := range modules {
			ifaceRel := interfaceArtifactRelPath(module.Module)
			generated = append(generated, generatedCapsuleArtifact{
				Artifact: capsuleArtifact{Kind: "interface", Path: ifaceRel},
				Module:   module.Module,
				Source:   module.Path,
			})
			ifacePath := filepath.Join(root, filepath.FromSlash(ifaceRel))
			if err := os.MkdirAll(filepath.Dir(ifacePath), 0o755); err != nil {
				return err
			}
			iface, err := compiler.GenerateInterfaceFile(module.Path)
			if err != nil {
				return err
			}
			if err := os.WriteFile(ifacePath, iface, 0o644); err != nil {
				return err
			}

			for _, target := range targets {
				objectRel := objectArtifactRelPath(module.Module, target)
				objectPath := filepath.Join(root, filepath.FromSlash(objectRel))
				if err := os.MkdirAll(filepath.Dir(objectPath), 0o755); err != nil {
					return err
				}
				if _, err := compiler.BuildFileWithStatsOpt(module.Path, objectPath, target, buildOpt); err != nil {
					return err
				}
				generated = append(generated, generatedCapsuleArtifact{
					Artifact: capsuleArtifact{Kind: "object", Target: target, Path: objectRel},
					Module:   module.Module,
					Source:   module.Path,
				})
			}
		}
	}
	if len(dependencyManifests) > 0 {
		seedRel := seedArtifactRelPath(rootManifest.Name)
		seedPath := filepath.Join(root, filepath.FromSlash(seedRel))
		if err := os.MkdirAll(filepath.Dir(seedPath), 0o755); err != nil {
			return err
		}
		if err := writeDependencySeed(seedPath, dependencyManifests); err != nil {
			return err
		}
		generated = append(generated, generatedCapsuleArtifact{
			Artifact: capsuleArtifact{Kind: "seed", Path: seedRel},
		})
	}
	if err := appendGeneratedArtifactsToCapsule(rootManifest.Path, rootManifest.Artifacts, generated); err != nil {
		return err
	}
	updatedManifests, err := parseCapsuleGraphArgs([]string{rootManifest.Path})
	if err != nil {
		return err
	}
	for _, target := range targets {
		if err := validateCapsuleGraph(updatedManifests, target); err != nil {
			return err
		}
	}
	return writeEcoLock(lockPath, updatedManifests)
}

func resolveArtifactBuildTargets(manifest capsuleManifest, targetFlag string, allTargets bool) ([]string, error) {
	if targetFlag != "" {
		target, err := normalizeCapsuleTarget(targetFlag)
		if err != nil {
			return nil, err
		}
		if ctarget.IsBuildOnlyTarget(target) {
			return nil, fmt.Errorf("eco artifacts build requires a native object target, got %s", target)
		}
		return []string{target}, nil
	}
	if allTargets {
		seen := map[string]struct{}{}
		var targets []string
		for _, raw := range manifest.Targets {
			target, err := normalizeCapsuleTarget(raw)
			if err != nil {
				return nil, err
			}
			if ctarget.IsBuildOnlyTarget(target) {
				continue
			}
			if _, ok := seen[target]; ok {
				continue
			}
			seen[target] = struct{}{}
			targets = append(targets, target)
		}
		sort.Strings(targets)
		if len(targets) == 0 {
			return nil, fmt.Errorf("eco artifacts build --all-targets found no native targets in Capsule.t4")
		}
		return targets, nil
	}
	if len(manifest.Targets) > 0 {
		target, err := normalizeCapsuleTarget(manifest.Targets[0])
		if err != nil {
			return nil, err
		}
		if ctarget.IsBuildOnlyTarget(target) {
			return nil, fmt.Errorf("eco artifacts build requires a native object target, got %s", target)
		}
		return []string{target}, nil
	}
	host, ok := hostTarget()
	if !ok {
		return nil, fmt.Errorf("host target unsupported; pass --target")
	}
	return []string{host}, nil
}

func resolveDeclaredArtifactTargets(manifest capsuleManifest, targetFlag string, allTargets bool) ([]string, error) {
	if targetFlag != "" {
		target, err := normalizeCapsuleTarget(targetFlag)
		if err != nil {
			return nil, err
		}
		if ctarget.IsBuildOnlyTarget(target) {
			return nil, nil
		}
		return []string{target}, nil
	}
	if allTargets {
		seen := map[string]struct{}{}
		var targets []string
		for _, raw := range manifest.Targets {
			target, err := normalizeCapsuleTarget(raw)
			if err != nil {
				return nil, err
			}
			if ctarget.IsBuildOnlyTarget(target) {
				continue
			}
			if _, ok := seen[target]; ok {
				continue
			}
			seen[target] = struct{}{}
			targets = append(targets, target)
		}
		sort.Strings(targets)
		return targets, nil
	}
	if len(manifest.Targets) > 0 {
		target, err := normalizeCapsuleTarget(manifest.Targets[0])
		if err != nil {
			return nil, err
		}
		if ctarget.IsBuildOnlyTarget(target) {
			return nil, nil
		}
		return []string{target}, nil
	}
	host, ok := hostTarget()
	if !ok {
		return nil, nil
	}
	return []string{host}, nil
}

func planCapsuleArtifacts(capsulePath string, targetFlag string, allTargets bool) (capsuleArtifactPlan, error) {
	manifests, err := parseCapsuleGraphArgs([]string{capsulePath})
	if err != nil {
		return capsuleArtifactPlan{}, err
	}
	if len(manifests) == 0 {
		return capsuleArtifactPlan{}, fmt.Errorf("%s: no capsule graph", capsulePath)
	}
	rootManifest := manifests[0]
	targets, err := resolveArtifactBuildTargets(rootManifest, targetFlag, allTargets)
	if err != nil {
		return capsuleArtifactPlan{}, err
	}
	return planCapsuleArtifactsForTargets(manifests, targets)
}

func planDeclaredCapsuleArtifacts(capsulePath string, targetFlag string, allTargets bool) (capsuleArtifactPlan, error) {
	manifests, err := parseCapsuleGraphArgs([]string{capsulePath})
	if err != nil {
		return capsuleArtifactPlan{}, err
	}
	if len(manifests) == 0 {
		return capsuleArtifactPlan{}, fmt.Errorf("%s: no capsule graph", capsulePath)
	}
	targets, err := resolveDeclaredArtifactTargets(manifests[0], targetFlag, allTargets)
	if err != nil {
		return capsuleArtifactPlan{}, err
	}
	return planCapsuleArtifactsForTargets(manifests, targets)
}

func planCapsuleArtifactsForTargets(manifests []capsuleManifest, targets []string) (capsuleArtifactPlan, error) {
	if len(manifests) == 0 {
		return capsuleArtifactPlan{}, fmt.Errorf("no capsule graph")
	}
	rootManifest := manifests[0]
	root := filepath.Dir(rootManifest.Path)
	dependencyManifests := manifests[1:]
	var expected []generatedCapsuleArtifact
	seenInterfaces := map[string]struct{}{}
	for _, depManifest := range dependencyManifests {
		modules, err := capsuleSourceModules(depManifest)
		if err != nil {
			return capsuleArtifactPlan{}, err
		}
		for _, module := range modules {
			ifaceRel := interfaceArtifactRelPath(module.Module)
			if _, ok := seenInterfaces[ifaceRel]; !ok {
				seenInterfaces[ifaceRel] = struct{}{}
				expected = append(expected, generatedCapsuleArtifact{
					Artifact: capsuleArtifact{Kind: "interface", Path: ifaceRel},
					Module:   module.Module,
					Source:   module.Path,
				})
			}
			for _, target := range targets {
				expected = append(expected, generatedCapsuleArtifact{
					Artifact: capsuleArtifact{Kind: "object", Target: target, Path: objectArtifactRelPath(module.Module, target)},
					Module:   module.Module,
					Source:   module.Path,
				})
			}
		}
	}
	if len(dependencyManifests) > 0 {
		expected = append(expected, generatedCapsuleArtifact{
			Artifact: capsuleArtifact{Kind: "seed", Path: seedArtifactRelPath(rootManifest.Name)},
		})
	}
	return capsuleArtifactPlan{
		RootManifest:        rootManifest,
		Manifests:           manifests,
		DependencyManifests: dependencyManifests,
		Root:                root,
		Targets:             targets,
		Expected:            expected,
	}, nil
}

func checkCapsuleArtifacts(capsulePath string, targetFlag string, lockPath string, allTargets bool) ([]artifactIssue, error) {
	return checkCapsuleArtifactsOpt(capsulePath, targetFlag, lockPath, allTargets, true)
}

func checkDeclaredCapsuleArtifacts(capsulePath string, targetFlag string, lockPath string, allTargets bool) ([]artifactIssue, error) {
	return checkCapsuleArtifactsOpt(capsulePath, targetFlag, lockPath, allTargets, false)
}

func checkCapsuleArtifactsOpt(capsulePath string, targetFlag string, lockPath string, allTargets bool, requireExpected bool) ([]artifactIssue, error) {
	var plan capsuleArtifactPlan
	var err error
	if requireExpected {
		plan, err = planCapsuleArtifacts(capsulePath, targetFlag, allTargets)
	} else {
		plan, err = planDeclaredCapsuleArtifacts(capsulePath, targetFlag, allTargets)
	}
	if err != nil {
		return nil, err
	}
	if lockPath == "" {
		lockPath = filepath.Join(plan.Root, compiler.SemanticLockFileName)
	}
	repair := artifactRepairCommand(plan.RootManifest.Path, plan.Targets, allTargets)
	var issues []artifactIssue
	declared := map[string]capsuleArtifact{}
	for _, artifact := range plan.RootManifest.Artifacts {
		declared[capsuleArtifactKey(artifact)] = artifact
	}
	for _, item := range plan.Expected {
		key := capsuleArtifactKey(item.Artifact)
		if _, ok := declared[key]; !ok {
			if requireExpected {
				issues = append(issues, artifactIssue{
					Kind:   "missing " + item.Artifact.Kind + " artifact",
					Module: item.Module,
					Path:   item.Artifact.Path,
					Detail: "Capsule.t4 does not declare this generated artifact",
					Repair: repair,
				})
			}
			continue
		}
		path := filepath.Join(plan.Root, filepath.FromSlash(item.Artifact.Path))
		raw, err := os.ReadFile(path)
		if err != nil {
			issues = append(issues, artifactIssue{
				Kind:   "missing " + item.Artifact.Kind + " artifact",
				Module: item.Module,
				Path:   item.Artifact.Path,
				Detail: err.Error(),
				Repair: repair,
			})
			continue
		}
		switch item.Artifact.Kind {
		case "interface":
			expected, err := sourcePublicAPIHash(item.Source)
			if err != nil {
				return nil, err
			}
			actual, err := compiler.InterfaceFingerprintFromT4I(raw)
			if err != nil {
				issues = append(issues, artifactIssue{Kind: "invalid interface artifact", Module: item.Module, Path: item.Artifact.Path, Detail: err.Error(), Repair: repair})
				continue
			}
			if actual != expected {
				issues = append(issues, artifactIssue{
					Kind:   "stale interface artifact",
					Module: item.Module,
					Path:   item.Artifact.Path,
					Detail: fmt.Sprintf("expected public API %s, artifact has %s", expected, actual),
					Repair: repair,
				})
			}
		case "object":
			obj, err := compiler.ReadObject(path)
			if err != nil {
				issues = append(issues, artifactIssue{Kind: "invalid object artifact", Module: item.Module, Path: item.Artifact.Path, Detail: err.Error(), Repair: repair})
				continue
			}
			if obj.Target != "" && obj.Target != item.Artifact.Target {
				issues = append(issues, artifactIssue{
					Kind:   "wrong-target object artifact",
					Module: item.Module,
					Path:   item.Artifact.Path,
					Detail: fmt.Sprintf("expected target %s, object has %s", item.Artifact.Target, obj.Target),
					Repair: repair,
				})
			}
			expectedAPI, err := sourcePublicAPIHash(item.Source)
			if err != nil {
				return nil, err
			}
			if obj.PublicAPIHash != expectedAPI {
				issues = append(issues, artifactIssue{
					Kind:   "stale object artifact",
					Module: item.Module,
					Path:   item.Artifact.Path,
					Detail: fmt.Sprintf("expected public API %s, object has %s", expectedAPI, obj.PublicAPIHash),
					Repair: repair,
				})
			}
			expectedSrc, err := sourceSHA256(item.Source)
			if err != nil {
				return nil, err
			}
			if obj.SrcHash != expectedSrc {
				issues = append(issues, artifactIssue{
					Kind:   "stale object artifact",
					Module: item.Module,
					Path:   item.Artifact.Path,
					Detail: "source hash changed since object build",
					Repair: repair,
				})
			}
		case "seed":
			var seed ecoSeed
			decoder := json.NewDecoder(bytes.NewReader(raw))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&seed); err != nil {
				issues = append(issues, artifactIssue{Kind: "invalid seed artifact", Path: item.Artifact.Path, Detail: err.Error(), Repair: repair})
				continue
			}
			currentLock, err := buildEcoLockWithArtifactHashes(plan.DependencyManifests)
			if err != nil {
				return nil, err
			}
			normalizeLock(&seed.Lock)
			if seed.Lock.GraphSHA256 != currentLock.GraphSHA256 {
				issues = append(issues, artifactIssue{
					Kind:   "stale seed artifact",
					Path:   item.Artifact.Path,
					Detail: fmt.Sprintf("expected dependency graph %s, seed has %s", currentLock.GraphSHA256, seed.Lock.GraphSHA256),
					Repair: repair,
				})
			}
		}
	}
	if _, err := os.Stat(lockPath); err != nil {
		if os.IsNotExist(err) {
			if requireExpected {
				issues = append(issues, artifactIssue{Kind: "missing lock", Path: filepath.ToSlash(lockPath), Detail: "Tetra.lock is required for locked project builds", Repair: repair})
			}
		} else {
			return nil, err
		}
	} else {
		raw, err := os.ReadFile(lockPath)
		if err != nil {
			return nil, err
		}
		lock, err := decodeEcoLock(raw)
		if err != nil {
			issues = append(issues, artifactIssue{Kind: "invalid lock", Path: filepath.ToSlash(lockPath), Detail: err.Error(), Repair: repair})
		} else {
			current, err := buildEcoLockWithArtifactHashes(plan.Manifests)
			if err != nil {
				return nil, err
			}
			if lock.GraphSHA256 != current.GraphSHA256 {
				issues = append(issues, artifactIssue{
					Kind:   "stale lock",
					Path:   filepath.ToSlash(lockPath),
					Detail: fmt.Sprintf("expected graph %s, lock has %s", current.GraphSHA256, lock.GraphSHA256),
					Repair: repair,
				})
			}
		}
	}
	return dedupeArtifactIssues(issues), nil
}

func sourcePublicAPIHash(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return compiler.InterfaceFingerprintFromSource(raw, path)
}

func sourceSHA256(path string) ([32]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(raw), nil
}

func artifactRepairCommand(capsulePath string, targets []string, allTargets bool) string {
	if allTargets {
		return fmt.Sprintf("tetra eco artifacts build --all-targets --lock %s %s", compiler.SemanticLockFileName, filepath.ToSlash(capsulePath))
	}
	target := ""
	if len(targets) > 0 {
		target = targets[0]
	}
	if target != "" {
		return fmt.Sprintf("tetra eco artifacts build --target %s --lock %s %s", target, compiler.SemanticLockFileName, filepath.ToSlash(capsulePath))
	}
	return fmt.Sprintf("tetra eco artifacts build --lock %s %s", compiler.SemanticLockFileName, filepath.ToSlash(capsulePath))
}

func writeArtifactIssues(w io.Writer, issues []artifactIssue, dryRun bool) {
	for _, issue := range issues {
		prefix := issue.Kind
		if dryRun && strings.HasPrefix(issue.Kind, "missing ") {
			prefix = "would generate " + strings.TrimPrefix(issue.Kind, "missing ")
		}
		if issue.Module != "" {
			fmt.Fprintf(w, "%s: %s (%s)", prefix, issue.Path, issue.Module)
		} else {
			fmt.Fprintf(w, "%s: %s", prefix, issue.Path)
		}
		if issue.Detail != "" {
			fmt.Fprintf(w, ": %s", issue.Detail)
		}
		if issue.Repair != "" {
			fmt.Fprintf(w, "\nrepair: %s", issue.Repair)
		}
		fmt.Fprintln(w)
	}
}

func dedupeArtifactIssues(issues []artifactIssue) []artifactIssue {
	seen := map[string]struct{}{}
	var out []artifactIssue
	for _, issue := range issues {
		key := issue.Kind + "\x00" + issue.Module + "\x00" + issue.Path + "\x00" + issue.Detail
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, issue)
	}
	return out
}

func dependencyArtifactBuildOptions(root string, manifest capsuleManifest, jobs int) (compiler.BuildOptions, error) {
	depRoots, _, err := projectDependencyGraph(root, manifest, map[string]int{root: projectDependencyVisiting}, []string{root})
	if err != nil {
		return compiler.BuildOptions{}, err
	}
	artifactRoots, err := projectArtifactInterfaceRoots(root, manifest.Artifacts)
	if err != nil {
		return compiler.BuildOptions{}, err
	}
	depRoots = append(depRoots, artifactRoots...)
	return compiler.BuildOptions{
		Jobs:            jobs,
		Emit:            compiler.EmitLibrary,
		ProjectRoot:     root,
		SourceRoots:     projectSourceRoots(manifest),
		DependencyRoots: depRoots,
	}, nil
}

func capsuleSourceModules(manifest capsuleManifest) ([]capsuleSourceModule, error) {
	root := filepath.Dir(manifest.Path)
	roots := projectSourceRoots(manifest)
	seenModules := map[string]string{}
	var modules []capsuleSourceModule
	for _, sourceRoot := range roots {
		dir := root
		if sourceRoot != "" {
			dir = filepath.Join(root, filepath.FromSlash(sourceRoot))
		}
		info, err := os.Stat(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if !info.IsDir() {
			continue
		}
		err = filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				name := d.Name()
				if name == ".tetra" || name == ".tetra_cache" || name == "tetra_cache" {
					return filepath.SkipDir
				}
				return nil
			}
			if !compiler.IsSourceFile(path) {
				return nil
			}
			base := filepath.Base(path)
			if base == compiler.CapsuleFileName || base == compiler.LegacyCapsuleFileName {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			file, err := compiler.ParseFile(raw, path)
			if err != nil {
				return err
			}
			if file.Module == "" {
				return nil
			}
			if first, ok := seenModules[file.Module]; ok {
				return fmt.Errorf("%s: duplicate dependency module %s (first %s)", path, file.Module, first)
			}
			seenModules[file.Module] = path
			modules = append(modules, capsuleSourceModule{Module: file.Module, Path: path})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(modules, func(i, j int) bool { return modules[i].Module < modules[j].Module })
	return modules, nil
}

func interfaceArtifactRelPath(module string) string {
	return filepath.ToSlash(filepath.Join("interfaces", strings.ReplaceAll(module, ".", "/")+compiler.T4InterfaceExtension))
}

func objectArtifactRelPath(module string, target string) string {
	return filepath.ToSlash(filepath.Join("artifacts", strings.ReplaceAll(module, ".", "/")+"."+target+".tobj"))
}

func seedArtifactRelPath(name string) string {
	slug := artifactSlug(name)
	if slug == "" {
		slug = "capsule"
	}
	return filepath.ToSlash(filepath.Join("seeds", slug+"-deps"+compiler.T4SeedExtension))
}

func artifactSlug(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastDash := false
	for _, ch := range name {
		ok := (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
		if ok {
			b.WriteRune(ch)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func writeDependencySeed(path string, manifests []capsuleManifest) error {
	lock, err := buildEcoLockWithArtifactHashes(manifests)
	if err != nil {
		return err
	}
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
	return writeJSONFile(path, seed)
}

func appendGeneratedArtifactsToCapsule(path string, existing []capsuleArtifact, generated []generatedCapsuleArtifact) error {
	var missing []capsuleArtifact
	seen := map[string]struct{}{}
	for _, artifact := range existing {
		seen[capsuleArtifactKey(artifact)] = struct{}{}
	}
	for _, item := range generated {
		key := capsuleArtifactKey(item.Artifact)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		missing = append(missing, item.Artifact)
	}
	if len(missing) == 0 {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := strings.TrimRight(string(raw), "\n")
	var b strings.Builder
	b.WriteString(text)
	b.WriteString("\n\n    artifacts:\n")
	for _, artifact := range missing {
		b.WriteString("        ")
		b.WriteString(formatCapsuleArtifactLine(artifact))
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func capsuleArtifactKey(artifact capsuleArtifact) string {
	return artifact.Kind + "\x00" + artifact.Target + "\x00" + artifact.Path
}

func formatCapsuleArtifactLine(artifact capsuleArtifact) string {
	if artifact.Target != "" {
		return artifact.Kind + " " + artifact.Target + " " + artifact.Path
	}
	return artifact.Kind + " " + artifact.Path
}

func runEcoPack(args []string, stdout io.Writer, stderr io.Writer) int {
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra eco pack [--project] [-o PATH] [Capsule.t4]")
		return 0
	}
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
		outPath = manifest.Name + compiler.TodexFragmentExtension
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
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra eco unpack PACKAGE [-C DIR]")
		return 0
	}
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

func runEcoNeedMap(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco needmap", flag.ContinueOnError)
	fs.SetOutput(stderr)
	lockPath := fs.String("lock", "", "path to lock JSON input")
	outPath := fs.String("o", compiler.DefaultNeedMapName, "path to NeedMap output")
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
	capsulePath, err := findCapsulePath(outDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	manifest, err := parseCapsule(capsulePath)
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
	pkgPath := fs.String("package", "", "path to .tdx/.todex package")
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
	pkgPath := fs.String("package", "", "path to .tdx/.todex package")
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
	switch key {
	case "unsafe":
		if value != "deny" && value != "allow" {
			return fmt.Errorf("%s:%d: unsafe policy must be deny or allow", path, line)
		}
	case "reproducible":
		if value != "required" && value != "preferred" && value != "off" {
			return fmt.Errorf("%s:%d: reproducible policy must be required, preferred, or off", path, line)
		}
	default:
		return fmt.Errorf("%s:%d: unknown policy %s", path, line, key)
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

func parseCapsuleArgs(paths []string) ([]capsuleManifest, error) {
	if len(paths) == 0 {
		paths = []string{defaultCapsulePath()}
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

func parseCapsuleGraphArgs(paths []string) ([]capsuleManifest, error) {
	if len(paths) == 0 {
		paths = []string{defaultCapsulePath()}
	}
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
	depRoots, depManifests, err := projectDependencyGraph(root, manifest, map[string]int{root: projectDependencyVisiting}, []string{root})
	if err != nil {
		return nil, err
	}
	_ = depRoots
	manifests := append([]capsuleManifest{manifest}, depManifests...)
	return manifests, nil
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

func defaultCapsulePath() string {
	if fileExists(compiler.CapsuleFileName) {
		return compiler.CapsuleFileName
	}
	if fileExists(compiler.LegacyCapsuleFileName) {
		return compiler.LegacyCapsuleFileName
	}
	return compiler.CapsuleFileName
}

func findCapsulePath(dir string) (string, error) {
	for _, name := range []string{compiler.CapsuleFileName, compiler.LegacyCapsuleFileName} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("missing %s (or legacy %s)", compiler.CapsuleFileName, compiler.LegacyCapsuleFileName)
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
	capsulePath, err := findCapsulePath(tmpDir)
	if err != nil {
		return "", err
	}
	manifest, err := parseCapsule(capsulePath)
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
		trustFile := "trust.snapshot.json"
		if err := os.WriteFile(filepath.Join(targetDir, trustFile), raw, 0o644); err != nil {
			return "", err
		}
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
			SnapshotFile: trustFile,
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
	if err := validateEcoPublishPackagePath(meta.Package.File); err != nil {
		return "", err
	}
	pkgPath := filepath.Join(targetDir, filepath.FromSlash(meta.Package.File))
	if outPath == "" {
		outPath = fmt.Sprintf("%s-%s-%s%s", capsuleIDDirectory(id), version, target, compiler.TodexFragmentExtension)
	}
	raw, err := os.ReadFile(pkgPath)
	if err != nil {
		return "", err
	}
	if int64(len(raw)) != meta.Package.Size {
		return "", fmt.Errorf("package size mismatch for %s: metadata=%d actual=%d", pkgPath, meta.Package.Size, len(raw))
	}
	hashHex, err := ecoPublishPackageHashHex(meta.Package.SHA256)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != hashHex {
		return "", fmt.Errorf("package hash mismatch for %s", pkgPath)
	}
	if err := os.WriteFile(outPath, raw, 0o644); err != nil {
		return "", err
	}
	return outPath, nil
}

func validateEcoPublishPackagePath(path string) error {
	if path == "" {
		return fmt.Errorf("package file is required")
	}
	if strings.Contains(path, "\\") {
		return fmt.Errorf("unsafe package file path %s", path)
	}
	clean := filepath.Clean(path)
	if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return fmt.Errorf("unsafe package file path %s", path)
	}
	if filepath.ToSlash(clean) != path {
		return fmt.Errorf("package file path %s is not normalized", path)
	}
	return nil
}

func ecoPublishPackageHashHex(hash string) (string, error) {
	const prefix = "sha256:"
	if !strings.HasPrefix(hash, prefix) {
		return "", fmt.Errorf("invalid package sha256 hash %s", hash)
	}
	hexHash := strings.TrimPrefix(hash, prefix)
	if len(hexHash) != sha256.Size*2 {
		return "", fmt.Errorf("invalid package sha256 hash %s", hash)
	}
	if _, err := hex.DecodeString(hexHash); err != nil {
		return "", fmt.Errorf("invalid package sha256 hash %s", hash)
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

func validateCapsulePolicy(policy map[string]string) error {
	for key, value := range policy {
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

func isSupportedCapsuleTarget(target string) bool {
	for _, triple := range ctarget.SupportedTriples() {
		if triple == target {
			return true
		}
	}
	for _, triple := range ctarget.BuildOnlyTriples() {
		if triple == target {
			return true
		}
	}
	return false
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

func parseEcoPackArgs(args []string) (capsulePath string, outPath string, project bool, err error) {
	capsulePath = defaultCapsulePath()
	sawCapsulePath := false
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
			if sawCapsulePath {
				return "", "", false, fmt.Errorf("eco pack accepts one capsule path")
			}
			sawCapsulePath = true
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
		if filepath.Clean(path) == filepath.Clean(outPath) || strings.HasSuffix(rel, ".todex") || strings.HasSuffix(rel, compiler.TodexFragmentExtension) {
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
	zeroTime := time.Unix(0, 0).UTC()
	cleanRelPaths := append([]string(nil), relPaths...)
	sort.Strings(cleanRelPaths)
	for _, rel := range cleanRelPaths {
		if filepath.ToSlash(rel) == ecoPackageMetadataPath {
			return fmt.Errorf("project already contains reserved file %s", ecoPackageMetadataPath)
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
		Schema:           ecoPackageSchemaV1,
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
		Name:       ecoPackageMetadataPath,
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
	entries := map[string][]byte{}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := filepath.Clean(header.Name)
		if name == "." || strings.HasPrefix(name, "..") || filepath.IsAbs(name) {
			return fmt.Errorf("unsafe archive path %q", header.Name)
		}
		normalizedName := filepath.ToSlash(name)
		if normalizedName != header.Name {
			return fmt.Errorf("archive path %q is not normalized", header.Name)
		}
		if _, exists := entries[normalizedName]; exists {
			return fmt.Errorf("duplicate archive path %q", header.Name)
		}
		raw, err := io.ReadAll(tr)
		if err != nil {
			return err
		}
		if header.Size >= 0 && int64(len(raw)) != header.Size {
			return fmt.Errorf("archive size mismatch for %s", header.Name)
		}
		entries[normalizedName] = raw
	}
	if err := validateEcoPackageEntries(entries); err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	var names []string
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		outPath := filepath.Join(outDir, name)
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		if _, err := out.Write(entries[name]); err != nil {
			_ = out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
	}
	return nil
}

func validateEcoPackageEntries(entries map[string][]byte) error {
	rawMetadata, ok := entries[ecoPackageMetadataPath]
	if !ok {
		return fmt.Errorf("missing %s", ecoPackageMetadataPath)
	}
	decoder := json.NewDecoder(bytes.NewReader(rawMetadata))
	decoder.DisallowUnknownFields()
	var metadata ecoPackageMetadata
	if err := decoder.Decode(&metadata); err != nil {
		return fmt.Errorf("invalid %s: %w", ecoPackageMetadataPath, err)
	}
	if metadata.Schema != ecoPackageSchemaV1 {
		return fmt.Errorf("unsupported package metadata schema %q", metadata.Schema)
	}
	if metadata.Compression != "gzip" {
		return fmt.Errorf("package metadata compression must be gzip")
	}
	if metadata.MTimeUnix != 0 {
		return fmt.Errorf("package metadata mtime_unix must be 0")
	}
	if metadata.ManifestSchema != "" && metadata.ManifestSchema != capsuleManifestSchemaV1 {
		return fmt.Errorf("unsupported package metadata manifest_schema %q", metadata.ManifestSchema)
	}
	if metadata.PermissionsModel != "" && metadata.PermissionsModel != ecoPermissionsModelV1 {
		return fmt.Errorf("unsupported package metadata permissions_model %q", metadata.PermissionsModel)
	}
	if metadata.FileCount != len(metadata.Files) {
		return fmt.Errorf("package metadata file_count mismatch: expected %d, got %d", len(metadata.Files), metadata.FileCount)
	}
	if metadata.FileCount <= 0 {
		return fmt.Errorf("package metadata file_count must be positive")
	}
	declared := map[string]struct{}{ecoPackageMetadataPath: {}}
	lastPath := ""
	for _, file := range metadata.Files {
		if file.Path == "" {
			return fmt.Errorf("package metadata has empty path")
		}
		cleanPath := filepath.Clean(file.Path)
		if cleanPath == "." || strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) {
			return fmt.Errorf("package metadata has unsafe path %s", file.Path)
		}
		normalizedPath := filepath.ToSlash(cleanPath)
		if normalizedPath != file.Path {
			return fmt.Errorf("package metadata path %s is not normalized", file.Path)
		}
		if normalizedPath == ecoPackageMetadataPath {
			return fmt.Errorf("package metadata must not self-reference %s", ecoPackageMetadataPath)
		}
		if normalizedPath <= lastPath {
			return fmt.Errorf("package metadata files must be strictly sorted by path")
		}
		lastPath = normalizedPath
		if _, exists := declared[normalizedPath]; exists {
			return fmt.Errorf("package metadata has duplicate file path %s", normalizedPath)
		}
		raw, ok := entries[normalizedPath]
		if !ok {
			return fmt.Errorf("package metadata references missing file %s", normalizedPath)
		}
		if int64(len(raw)) != file.Size {
			return fmt.Errorf("package metadata size mismatch for %s", normalizedPath)
		}
		hashHex, err := ecoPublishPackageHashHex(file.SHA256)
		if err != nil {
			return fmt.Errorf("package metadata %s: %w", normalizedPath, err)
		}
		sum := sha256.Sum256(raw)
		if hex.EncodeToString(sum[:]) != hashHex {
			return fmt.Errorf("package metadata hash mismatch for %s", normalizedPath)
		}
		declared[normalizedPath] = struct{}{}
	}
	if _, ok := declared[compiler.CapsuleFileName]; !ok {
		if _, legacyOK := declared[compiler.LegacyCapsuleFileName]; !legacyOK {
			return fmt.Errorf("package metadata missing %s entry", compiler.CapsuleFileName)
		}
	}
	for name := range entries {
		if _, ok := declared[name]; !ok {
			return fmt.Errorf("archive contains undeclared file %s", name)
		}
	}
	if metadata.BuildInputsSHA != "" {
		hashHex, err := ecoPublishPackageHashHex(metadata.BuildInputsSHA)
		if err != nil {
			return fmt.Errorf("package metadata build_inputs_sha256: %w", err)
		}
		sum := sha256.Sum256([]byte(packageMetadataFingerprint(metadata.Files)))
		if hex.EncodeToString(sum[:]) != hashHex {
			return fmt.Errorf("package metadata build_inputs_sha256 mismatch")
		}
	}
	return nil
}
