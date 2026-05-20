package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tetra_language/compiler"
)

type ecoPublishMetadata struct {
	Schema        string               `json:"schema"`
	Channel       string               `json:"channel"`
	Hub           string               `json:"hub"`
	PublishedUnix int64                `json:"published_at_unix"`
	Capsule       ecoPublishCapsule    `json:"capsule"`
	Package       ecoPublishPackage    `json:"package"`
	Trust         *ecoPublishTrust     `json:"trust,omitempty"`
	Downloads     []ecoPublishDownload `json:"downloads,omitempty"`
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

type ecoMirrorReport struct {
	Schema              string `json:"schema"`
	MirroredUnix        int64  `json:"mirrored_at_unix"`
	SourceStore         string `json:"source_store"`
	DestinationStore    string `json:"destination_store"`
	ID                  string `json:"id"`
	Version             string `json:"version"`
	Target              string `json:"target"`
	Channel             string `json:"channel"`
	Hub                 string `json:"hub"`
	PackagePath         string `json:"package_path"`
	PackageSHA256       string `json:"package_sha256"`
	MetadataPath        string `json:"metadata_path"`
	MetadataSHA256      string `json:"metadata_sha256"`
	TrustSnapshotPath   string `json:"trust_snapshot_path,omitempty"`
	TrustSnapshotSHA256 string `json:"trust_snapshot_sha256,omitempty"`
}

const (
	ecoPublishedPackageFileName = "package.todex"
	ecoPublishMetadataFileName  = "metadata.json"
	ecoPublishTrustFileName     = "trust.snapshot.json"
)

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
	resolvedPkgPath, err := resolveEcoPackageArg(fs, *pkgPath, "eco publish")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if err := validateEcoPublishChannel("eco publish", *channel); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	metaPath, err := publishPackage(resolvedPkgPath, *registry, *target, *trustPath, *hub, *channel)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Published (%s): %s\n", *channel, metaPath)
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
	if err := validateEcoDownloadRequest("eco download", *id, *version); err != nil {
		fmt.Fprintln(stderr, err)
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
		fmt.Fprintln(stderr, "usage: tetra eco tetrahub <publish|download|mirror|fetch> [options]")
		return 2
	}
	switch args[0] {
	case "publish":
		return runEcoTetraHubPublish(args[1:], stdout, stderr)
	case "download":
		return runEcoTetraHubDownload(args[1:], stdout, stderr)
	case "mirror":
		return runEcoTetraHubMirror(args[1:], stdout, stderr)
	case "fetch":
		return runEcoTetraHubFetch(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown eco tetrahub command %q\n", args[0])
		return 2
	}
}

func runEcoTetraHubPublish(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco tetrahub publish", flag.ContinueOnError)
	fs.SetOutput(stderr)
	pkgPath := fs.String("package", "", "path to .tdx/.todex package")
	store := fs.String("store", ".tetra/tetrahub-beta", "path to local TetraHub store")
	target := fs.String("target", "", "target triple to publish")
	trustPath := fs.String("trust", "", "optional trust snapshot file")
	channel := fs.String("channel", "beta", "publishing channel")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	resolvedPkgPath, err := resolveEcoPackageArg(fs, *pkgPath, "eco tetrahub publish")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if err := validateEcoPublishChannel("eco tetrahub publish", *channel); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	metaPath, err := publishPackage(resolvedPkgPath, *store, *target, *trustPath, ecoTetraHubLabel(*channel), *channel)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "TetraHub %s published: %s\n", *channel, metaPath)
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
	if err := validateEcoDownloadRequest("eco tetrahub download", *id, *version); err != nil {
		fmt.Fprintln(stderr, err)
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

func runEcoTetraHubMirror(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco tetrahub mirror", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fromStore := fs.String("from", "", "source local TetraHub store")
	toStore := fs.String("to", "", "destination local TetraHub store")
	id := fs.String("id", "", "capsule id")
	version := fs.String("version", "", "capsule version")
	target := fs.String("target", "", "target triple")
	outPath := fs.String("o", "tetra.eco.mirror.json", "output mirror report path")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *fromStore == "" || *toStore == "" {
		fmt.Fprintln(stderr, "eco tetrahub mirror requires --from and --to")
		return 2
	}
	if err := validateEcoTargetedRequest("eco tetrahub mirror", *id, *version, *target); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	report, err := mirrorPublishedPackage(*fromStore, *toStore, *id, *version, *target)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := writeJSONFile(*outPath, report); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "TetraHub mirrored: %s\n", *outPath)
	return 0
}

func runEcoTetraHubFetch(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco tetrahub fetch", flag.ContinueOnError)
	fs.SetOutput(stderr)
	baseURL := fs.String("url", "", "source HTTP(S) TetraHub store URL")
	toStore := fs.String("to", "", "destination local TetraHub store")
	id := fs.String("id", "", "capsule id")
	version := fs.String("version", "", "capsule version")
	target := fs.String("target", "", "target triple")
	outPath := fs.String("o", "tetra.eco.mirror.json", "output mirror report path")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *baseURL == "" || *toStore == "" {
		fmt.Fprintln(stderr, "eco tetrahub fetch requires --url and --to")
		return 2
	}
	if err := validateEcoTargetedRequest("eco tetrahub fetch", *id, *version, *target); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	report, err := fetchPublishedPackage(*baseURL, *toStore, *id, *version, *target)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := writeJSONFile(*outPath, report); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "TetraHub fetched: %s\n", *outPath)
	return 0
}

func resolveEcoPackageArg(fs *flag.FlagSet, pkgPath string, command string) (string, error) {
	if pkgPath != "" {
		return pkgPath, nil
	}
	if fs.NArg() != 1 {
		return "", fmt.Errorf("%s requires --package or one package path", command)
	}
	return fs.Arg(0), nil
}

func validateEcoPublishChannel(command string, channel string) error {
	if isSupportedEcoPublishChannel(channel) {
		return nil
	}
	return fmt.Errorf("%s supports beta or stable channel, got %q", command, channel)
}

func validateEcoDownloadRequest(command string, id string, version string) error {
	if id == "" || version == "" {
		return fmt.Errorf("%s requires --id and --version", command)
	}
	return nil
}

func validateEcoTargetedRequest(command string, id string, version string, target string) error {
	if id == "" || version == "" || target == "" {
		return fmt.Errorf("%s requires --id, --version, and --target", command)
	}
	return nil
}

func ecoTetraHubLabel(channel string) string {
	if channel == "stable" {
		return "tetrahub-stable"
	}
	return "tetrahub-beta"
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
	target, err = resolvePublishedPackageTarget(manifest, target)
	if err != nil {
		return "", err
	}
	pkgRaw, err := os.ReadFile(pkgPath)
	if err != nil {
		return "", err
	}
	targetRel := ecoPublishTargetRel(manifest.ID, manifest.Version, target)
	pkgOutPath, err := writeEcoFileInsideRootNoSymlink(registry, ecoPublishTargetFileRel(targetRel, ecoPublishedPackageFileName), pkgRaw, 0o644, "publish package")
	if err != nil {
		return "", err
	}
	targetDir := filepath.Dir(pkgOutPath)
	meta := buildEcoPublishMetadata(manifest, target, hub, channel, pkgRaw)
	if trustPath != "" {
		if err := attachEcoPublishTrust(&meta, targetDir, trustPath, manifest); err != nil {
			return "", err
		}
	}
	metaPath, err := writeEcoJSONFileInsideRootNoSymlink(registry, ecoPublishTargetFileRel(targetRel, ecoPublishMetadataFileName), meta, "publish metadata")
	if err != nil {
		return "", err
	}
	return metaPath, nil
}

func buildEcoPublishMetadata(manifest capsuleManifest, target string, hub string, channel string, pkgRaw []byte) ecoPublishMetadata {
	sum := sha256.Sum256(pkgRaw)
	return ecoPublishMetadata{
		Schema:        ecoPublishSchemaForChannel(channel),
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
			File:   ecoPublishedPackageFileName,
			Size:   int64(len(pkgRaw)),
			SHA256: "sha256:" + hex.EncodeToString(sum[:]),
		},
		Downloads: []ecoPublishDownload{
			{Target: target, Path: ecoPublishDownloadPath(manifest.ID, manifest.Version, target, ecoPublishedPackageFileName)},
		},
	}
}

func attachEcoPublishTrust(meta *ecoPublishMetadata, targetDir string, trustPath string, manifest capsuleManifest) error {
	raw, err := os.ReadFile(trustPath)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(raw)
	trustFile := ecoPublishTrustFileName
	if _, err := writeEcoFileInsideRootNoSymlink(targetDir, trustFile, raw, 0o644, "publish trust snapshot"); err != nil {
		return err
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
	return nil
}

func resolvePublishedPackageTarget(manifest capsuleManifest, target string) (string, error) {
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
	return target, nil
}

func downloadPackage(registry string, id string, version string, target string, outPath string) (string, error) {
	baseDir := filepath.Join(registry, "packages", capsuleIDDirectory(id), version)
	if target == "" {
		candidates, err := availableEcoPackageTargets(baseDir)
		if err != nil {
			return "", err
		}
		if len(candidates) == 0 {
			return "", fmt.Errorf("no targets available for %s %s", id, version)
		}
		target = candidates[0]
	}
	targetDir := filepath.Join(baseDir, target)
	metaPath := filepath.Join(targetDir, ecoPublishMetadataFileName)
	rawMeta, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			available, readErr := availableEcoPackageTargets(baseDir)
			if readErr != nil {
				return "", err
			}
			return "", fmt.Errorf("target %s not available for %s %s (available: %s)", target, id, version, strings.Join(available, ", "))
		}
		return "", err
	}
	var meta ecoPublishMetadata
	if err := decodeEcoPublishMetadata(rawMeta, &meta); err != nil {
		return "", err
	}
	if err := validateEcoPublishMetadataForDownload(meta, id, version, target, targetDir); err != nil {
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
	if err := validateEcoPublishPackageBytes(meta, pkgPath, raw); err != nil {
		return "", err
	}
	if err := os.WriteFile(outPath, raw, 0o644); err != nil {
		return "", err
	}
	return outPath, nil
}

func availableEcoPackageTargets(baseDir string) ([]string, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}
	var targets []string
	for _, entry := range entries {
		if entry.IsDir() {
			targets = append(targets, entry.Name())
		}
	}
	sort.Strings(targets)
	return targets, nil
}

func mirrorPublishedPackage(fromStore string, toStore string, id string, version string, target string) (ecoMirrorReport, error) {
	targetRel := ecoPublishTargetRel(id, version, target)
	metadataRel := ecoPublishTargetFileRel(targetRel, ecoPublishMetadataFileName)
	sourceTargetDir := filepath.Join(fromStore, filepath.FromSlash(targetRel))
	sourceMetadataPath := filepath.Join(sourceTargetDir, ecoPublishMetadataFileName)
	rawMeta, err := os.ReadFile(sourceMetadataPath)
	if err != nil {
		return ecoMirrorReport{}, err
	}
	var meta ecoPublishMetadata
	if err := decodeEcoPublishMetadata(rawMeta, &meta); err != nil {
		return ecoMirrorReport{}, err
	}
	if err := validateEcoPublishMetadataForDownload(meta, id, version, target, sourceTargetDir); err != nil {
		return ecoMirrorReport{}, err
	}
	sourcePackagePath := filepath.Join(sourceTargetDir, filepath.FromSlash(meta.Package.File))
	rawPackage, err := os.ReadFile(sourcePackagePath)
	if err != nil {
		return ecoMirrorReport{}, err
	}
	if err := validateEcoPublishPackageBytes(meta, sourcePackagePath, rawPackage); err != nil {
		return ecoMirrorReport{}, err
	}
	packageRel := ecoPublishTargetFileRel(targetRel, meta.Package.File)

	var (
		trustRel string
		rawTrust []byte
	)
	if meta.Trust != nil {
		trustRel = meta.Trust.SnapshotFile
		sourceTrustPath := filepath.Join(sourceTargetDir, filepath.FromSlash(trustRel))
		rawTrust, err = os.ReadFile(sourceTrustPath)
		if err != nil {
			return ecoMirrorReport{}, err
		}
	}

	return writeMirroredPublishedPackage(ecoMirrorWriteOptions{
		SourceStore:      filepath.Clean(fromStore),
		DestinationStore: filepath.Clean(toStore),
		ID:               id,
		Version:          version,
		Target:           target,
		TargetRel:        targetRel,
		PackageRel:       packageRel,
		MetadataRel:      metadataRel,
		TrustRel:         trustRel,
		Meta:             meta,
		RawMetadata:      rawMeta,
		RawPackage:       rawPackage,
		RawTrust:         rawTrust,
	})
}

func fetchPublishedPackage(baseURL string, toStore string, id string, version string, target string) (ecoMirrorReport, error) {
	sourceURL, err := normalizeEcoHTTPStoreURL(baseURL)
	if err != nil {
		return ecoMirrorReport{}, err
	}
	targetRel := ecoPublishTargetRel(id, version, target)
	metadataRel := ecoPublishTargetFileRel(targetRel, ecoPublishMetadataFileName)
	rawMeta, err := fetchEcoHTTPStoreFile(sourceURL, metadataRel)
	if err != nil {
		return ecoMirrorReport{}, err
	}
	var meta ecoPublishMetadata
	if err := decodeEcoPublishMetadata(rawMeta, &meta); err != nil {
		return ecoMirrorReport{}, err
	}
	if err := validateEcoPublishMetadataEnvelope(meta, id, version, target); err != nil {
		return ecoMirrorReport{}, err
	}
	packageRel := ecoPublishTargetFileRel(targetRel, meta.Package.File)
	rawPackage, err := fetchEcoHTTPStoreFile(sourceURL, packageRel)
	if err != nil {
		return ecoMirrorReport{}, err
	}
	if err := validateEcoPublishPackageBytes(meta, sourceURL+"/"+packageRel, rawPackage); err != nil {
		return ecoMirrorReport{}, err
	}

	var (
		trustRel string
		rawTrust []byte
	)
	if meta.Trust != nil {
		trustRel = meta.Trust.SnapshotFile
		trustPathRel := ecoPublishTargetFileRel(targetRel, trustRel)
		rawTrust, err = fetchEcoHTTPStoreFile(sourceURL, trustPathRel)
		if err != nil {
			return ecoMirrorReport{}, err
		}
		trustHash, err := ecoPublishSHA256HashHex(meta.Trust.SnapshotHash)
		if err != nil {
			return ecoMirrorReport{}, err
		}
		trustSum := sha256.Sum256(rawTrust)
		if hex.EncodeToString(trustSum[:]) != trustHash {
			return ecoMirrorReport{}, fmt.Errorf("trust snapshot hash mismatch for %s/%s", sourceURL, trustPathRel)
		}
	}

	return writeMirroredPublishedPackage(ecoMirrorWriteOptions{
		SourceStore:      sourceURL,
		DestinationStore: filepath.Clean(toStore),
		ID:               id,
		Version:          version,
		Target:           target,
		TargetRel:        targetRel,
		PackageRel:       packageRel,
		MetadataRel:      metadataRel,
		TrustRel:         trustRel,
		Meta:             meta,
		RawMetadata:      rawMeta,
		RawPackage:       rawPackage,
		RawTrust:         rawTrust,
	})
}

type ecoMirrorWriteOptions struct {
	SourceStore      string
	DestinationStore string
	ID               string
	Version          string
	Target           string
	TargetRel        string
	PackageRel       string
	MetadataRel      string
	TrustRel         string
	Meta             ecoPublishMetadata
	RawMetadata      []byte
	RawPackage       []byte
	RawTrust         []byte
}

func writeMirroredPublishedPackage(opt ecoMirrorWriteOptions) (ecoMirrorReport, error) {
	destPackagePath, err := writeEcoFileInsideRootNoSymlink(opt.DestinationStore, opt.PackageRel, opt.RawPackage, 0o644, "mirror package")
	if err != nil {
		return ecoMirrorReport{}, err
	}
	destTargetDir := filepath.Dir(destPackagePath)
	if _, err := writeEcoFileInsideRootNoSymlink(opt.DestinationStore, opt.MetadataRel, opt.RawMetadata, 0o644, "mirror metadata"); err != nil {
		return ecoMirrorReport{}, err
	}
	if opt.Meta.Trust != nil {
		if _, err := writeEcoFileInsideRootNoSymlink(opt.DestinationStore, ecoPublishTargetFileRel(opt.TargetRel, opt.TrustRel), opt.RawTrust, 0o644, "mirror trust snapshot"); err != nil {
			return ecoMirrorReport{}, err
		}
	}
	if err := validateEcoPublishMetadataForDownload(opt.Meta, opt.ID, opt.Version, opt.Target, destTargetDir); err != nil {
		return ecoMirrorReport{}, err
	}
	if err := validateEcoPublishPackageBytes(opt.Meta, destPackagePath, opt.RawPackage); err != nil {
		return ecoMirrorReport{}, err
	}

	report := ecoMirrorReport{
		Schema:           ecoMirrorSchemaV1,
		MirroredUnix:     0,
		SourceStore:      opt.SourceStore,
		DestinationStore: opt.DestinationStore,
		ID:               opt.ID,
		Version:          opt.Version,
		Target:           opt.Target,
		Channel:          opt.Meta.Channel,
		Hub:              opt.Meta.Hub,
		PackagePath:      opt.PackageRel,
		PackageSHA256:    sha256String(opt.RawPackage),
		MetadataPath:     opt.MetadataRel,
		MetadataSHA256:   sha256String(opt.RawMetadata),
	}
	if opt.Meta.Trust != nil {
		report.TrustSnapshotPath = ecoPublishTargetFileRel(opt.TargetRel, opt.TrustRel)
		report.TrustSnapshotSHA256 = sha256String(opt.RawTrust)
	}
	return report, nil
}

func validateEcoPublishMetadataEnvelope(meta ecoPublishMetadata, id string, version string, target string) error {
	if !isSupportedEcoPublishSchemaChannel(meta.Schema, meta.Channel) {
		return fmt.Errorf("unsupported publish metadata for %s %s %s", id, version, target)
	}
	if err := validateEcoPublishMetadataIdentity(meta, id, version, target); err != nil {
		return err
	}
	if err := validateEcoPublishPackageShape(meta); err != nil {
		return err
	}
	if _, err := ecoPublishPackageHashHex(meta.Package.SHA256); err != nil {
		return err
	}
	if err := validateEcoPublishDownloads(meta, id, version, target); err != nil {
		return err
	}
	if meta.Trust != nil {
		if err := validateEcoPublishMetadataPath(meta.Trust.SnapshotFile, "trust snapshot file"); err != nil {
			return err
		}
		if _, err := ecoPublishSHA256HashHex(meta.Trust.SnapshotHash); err != nil {
			return err
		}
		if meta.Trust.TrustTier == "" {
			return fmt.Errorf("trust tier is required")
		}
	}
	return nil
}

func normalizeEcoHTTPStoreURL(rawURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("eco tetrahub fetch requires http or https URL, got %q", rawURL)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("eco tetrahub fetch URL missing host")
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = strings.TrimRight(parsed.EscapedPath(), "/")
	if parsed.Path == "" {
		parsed.Path = ""
	}
	return parsed.String(), nil
}

func fetchEcoHTTPStoreFile(baseURL string, relPath string) ([]byte, error) {
	if err := validateEcoPublishMetadataPath(relPath, "remote path"); err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	requestURL := strings.TrimRight(baseURL, "/") + "/" + relPath
	resp, err := client.Get(requestURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", requestURL, resp.Status)
	}
	const maxEcoHTTPFetchBytes = 64 << 20
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxEcoHTTPFetchBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxEcoHTTPFetchBytes {
		return nil, fmt.Errorf("GET %s: response exceeds %d bytes", requestURL, maxEcoHTTPFetchBytes)
	}
	return raw, nil
}

func validateEcoPublishPackageBytes(meta ecoPublishMetadata, path string, raw []byte) error {
	if int64(len(raw)) != meta.Package.Size {
		return fmt.Errorf("package size mismatch for %s: metadata=%d actual=%d", path, meta.Package.Size, len(raw))
	}
	hashHex, err := ecoPublishPackageHashHex(meta.Package.SHA256)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != hashHex {
		return fmt.Errorf("package hash mismatch for %s", path)
	}
	return nil
}

func sha256String(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func decodeEcoPublishMetadata(raw []byte, meta *ecoPublishMetadata) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(meta); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("publish metadata has trailing JSON payload")
		}
		return fmt.Errorf("publish metadata has trailing JSON payload: %w", err)
	}
	return nil
}

func validateEcoPublishMetadataForDownload(meta ecoPublishMetadata, id string, version string, target string, targetDir string) error {
	if !isSupportedEcoPublishSchemaChannel(meta.Schema, meta.Channel) {
		return fmt.Errorf("unsupported publish metadata in %s", filepath.Join(targetDir, ecoPublishMetadataFileName))
	}
	if err := validateEcoPublishMetadataIdentity(meta, id, version, target); err != nil {
		return err
	}
	if meta.Trust != nil {
		if err := validateEcoPublishMetadataPath(meta.Trust.SnapshotFile, "trust snapshot file"); err != nil {
			return err
		}
		hexHash, err := ecoPublishSHA256HashHex(meta.Trust.SnapshotHash)
		if err != nil {
			return err
		}
		if meta.Trust.TrustTier == "" {
			return fmt.Errorf("trust tier is required")
		}
		snapshotPath := filepath.Join(targetDir, filepath.FromSlash(meta.Trust.SnapshotFile))
		rawSnapshot, err := os.ReadFile(snapshotPath)
		if err != nil {
			return err
		}
		snapshotSum := sha256.Sum256(rawSnapshot)
		if hex.EncodeToString(snapshotSum[:]) != hexHash {
			return fmt.Errorf("trust snapshot hash mismatch for %s", snapshotPath)
		}
	}
	if err := validateEcoPublishPackageShape(meta); err != nil {
		return err
	}
	if err := validateEcoPublishDownloads(meta, id, version, target); err != nil {
		return err
	}
	return nil
}

func validateEcoPublishMetadataIdentity(meta ecoPublishMetadata, id string, version string, target string) error {
	if meta.Hub == "" {
		return fmt.Errorf("hub is required")
	}
	if meta.PublishedUnix < 0 {
		return fmt.Errorf("published_at_unix must not be negative")
	}
	if meta.Capsule.ID != id {
		return fmt.Errorf("capsule id mismatch: metadata has %s", meta.Capsule.ID)
	}
	if meta.Capsule.Name == "" {
		return fmt.Errorf("capsule name is required")
	}
	if meta.Capsule.Version != version {
		return fmt.Errorf("capsule version mismatch: metadata has %s", meta.Capsule.Version)
	}
	if meta.Capsule.Target != target {
		return fmt.Errorf("capsule target mismatch: metadata has %s", meta.Capsule.Target)
	}
	if len(meta.Capsule.Targets) > 0 && !containsString(meta.Capsule.Targets, target) {
		return fmt.Errorf("capsule targets missing selected target %s", target)
	}
	return nil
}

func validateEcoPublishPackageShape(meta ecoPublishMetadata) error {
	if err := validateEcoPublishPackagePath(meta.Package.File); err != nil {
		return err
	}
	if meta.Package.Size < 0 {
		return fmt.Errorf("package size must not be negative")
	}
	return nil
}

func validateEcoPublishDownloads(meta ecoPublishMetadata, id string, version string, target string) error {
	if len(meta.Downloads) == 0 {
		return fmt.Errorf("downloads must not be empty")
	}
	expectedDownloadPath := ecoPublishDownloadPath(id, version, target, meta.Package.File)
	for _, download := range meta.Downloads {
		if download.Target != target {
			return fmt.Errorf("download target mismatch: metadata has %s", download.Target)
		}
		if err := validateEcoPublishMetadataPath(download.Path, "download path"); err != nil {
			return err
		}
		if download.Path != expectedDownloadPath {
			return fmt.Errorf("download path mismatch: metadata has %s, expected %s", download.Path, expectedDownloadPath)
		}
	}
	return nil
}

func ecoPublishTargetRel(id string, version string, target string) string {
	return filepath.ToSlash(filepath.Join("packages", capsuleIDDirectory(id), version, target))
}

func ecoPublishTargetFileRel(targetRel string, file string) string {
	return filepath.ToSlash(filepath.Join(targetRel, file))
}

func ecoPublishDownloadPath(id string, version string, target string, file string) string {
	return ecoPublishTargetFileRel(ecoPublishTargetRel(id, version, target), file)
}

func isSupportedEcoPublishChannel(channel string) bool {
	switch channel {
	case "beta", "stable":
		return true
	default:
		return false
	}
}

func ecoPublishSchemaForChannel(channel string) string {
	if channel == "stable" {
		return ecoPublishSchemaV1
	}
	return ecoPublishSchemaV1Beta
}

func isSupportedEcoPublishSchemaChannel(schema string, channel string) bool {
	switch channel {
	case "beta":
		return schema == ecoPublishSchemaV1Beta
	case "stable":
		return schema == ecoPublishSchemaV1
	default:
		return false
	}
}

func validateEcoPublishPackagePath(path string) error {
	return validateEcoPublishMetadataPath(path, "package file")
}

func validateEcoPublishMetadataPath(path string, label string) error {
	if path == "" {
		return fmt.Errorf("%s is required", label)
	}
	if strings.Contains(path, "\\") {
		return fmt.Errorf("unsafe %s path %s", label, path)
	}
	clean := filepath.Clean(path)
	if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return fmt.Errorf("unsafe %s path %s", label, path)
	}
	if filepath.ToSlash(clean) != path {
		return fmt.Errorf("%s path %s is not normalized", label, path)
	}
	return nil
}

func ecoPublishSHA256HashHex(hash string) (string, error) {
	return ecoPublishHashHex(hash, "invalid sha256 hash")
}

func ecoPublishPackageHashHex(hash string) (string, error) {
	return ecoPublishHashHex(hash, "invalid package sha256 hash")
}

func ecoPublishHashHex(hash string, errorPrefix string) (string, error) {
	const prefix = "sha256:"
	if !strings.HasPrefix(hash, prefix) {
		return "", fmt.Errorf("%s %s", errorPrefix, hash)
	}
	hexHash := strings.TrimPrefix(hash, prefix)
	if len(hexHash) != sha256.Size*2 {
		return "", fmt.Errorf("%s %s", errorPrefix, hash)
	}
	if _, err := hex.DecodeString(hexHash); err != nil {
		return "", fmt.Errorf("%s %s", errorPrefix, hash)
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
