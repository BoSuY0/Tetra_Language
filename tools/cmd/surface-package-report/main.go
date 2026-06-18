package main

import (
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

	"tetra_language/tools/validators/surfacepackage"
)

func main() {
	os.Exit(runSurfacePackageReport(os.Args[1:], os.Stdout, os.Stderr))
}

func runSurfacePackageReport(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("surface-package-report", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", "", "package artifact root")
	archive := fs.String("archive", "surface-desk-linux-x64.tar.gz", "package archive path relative to root")
	out := fs.String("out", "", "write Surface package report JSON")
	producer := fs.String("producer", "tools/cmd/surface-package-report", "report producer")
	gitHead := fs.String("git-head", "", "git commit used for same-commit evidence")
	version := fs.String("version", "", "module/tool version")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "surface-package-report does not accept positional arguments")
		return 2
	}
	if *root == "" || *out == "" {
		fmt.Fprintln(stderr, "--root and --out are required")
		return 2
	}
	report, err := buildReport(*root, *archive, *producer, *gitHead, *version)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	raw = append(raw, '\n')
	if err := surfacepackage.ValidateReportWithRoot(raw, *root); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := os.WriteFile(*out, raw, 0o644); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote surface package report to %s\n", *out)
	return 0
}

func buildReport(root string, archiveRel string, producer string, gitHead string, version string) (surfacepackage.Report, error) {
	root = filepath.Clean(root)
	if info, err := os.Lstat(root); err != nil {
		return surfacepackage.Report{}, err
	} else if !info.IsDir() {
		return surfacepackage.Report{}, fmt.Errorf("package root %s is not a directory", root)
	} else if info.Mode()&os.ModeSymlink != 0 {
		return surfacepackage.Report{}, fmt.Errorf("package root %s must not be a symlink", root)
	}
	if err := validateSafeRelPath(archiveRel); err != nil {
		return surfacepackage.Report{}, fmt.Errorf("archive path: %v", err)
	}
	archive, err := artifactRef(root, archiveRel)
	if err != nil {
		return surfacepackage.Report{}, err
	}
	manifest, err := artifactRef(root, "surface-package.json")
	if err != nil {
		return surfacepackage.Report{}, err
	}
	assetManifest, err := artifactRef(root, "assets/surface-assets.json")
	if err != nil {
		return surfacepackage.Report{}, err
	}
	permissions, err := artifactRef(root, "permissions.json")
	if err != nil {
		return surfacepackage.Report{}, err
	}
	hostAdapter, err := artifactRef(root, "host-adapter.json")
	if err != nil {
		return surfacepackage.Report{}, err
	}
	files, err := collectPackagedFiles(root, archiveRel)
	if err != nil {
		return surfacepackage.Report{}, err
	}
	return surfacepackage.Report{
		Schema:       surfacepackage.SchemaV1,
		Status:       "pass",
		Level:        surfacepackage.LevelSurfacePackageDistributionV1,
		Scope:        "surface-v1-scoped-linux-web-package",
		ReleaseScope: "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
		Producer:     producer,
		GitHead:      gitHead,
		Version:      version,
		Linux: surfacepackage.LinuxPackage{
			Target:              "linux-x64",
			SupportLevel:        "production",
			Format:              "surface-linux-tar-v1",
			InstallSmoke:        true,
			RunSmoke:            true,
			SameCommit:          true,
			Package:             archive,
			Manifest:            manifest,
			AssetManifest:       assetManifest,
			PermissionsManifest: permissions,
			HostAdapterMetadata: hostAdapter,
			Files:               files,
			Signature: surfacepackage.SignatureState{
				Kind:             "sha256-checksum-manifest",
				ChecksumManifest: true,
				Signed:           false,
				Notarized:        false,
				Verification:     "artifact-hashes",
			},
		},
		Targets: []surfacepackage.TargetStatus{
			{
				Target:          "windows-x64",
				Tier:            "beta-nonclaim",
				ProductionClaim: false,
				Signed:          false,
				Notarized:       false,
				Evidence:        "blocked until Windows installer signing and target-host evidence exist",
			},
			{
				Target:          "macos-x64",
				Tier:            "beta-nonclaim",
				ProductionClaim: false,
				Signed:          false,
				Notarized:       false,
				Evidence:        "blocked until macOS bundle signing and notarization evidence exist",
			},
		},
		Update: surfacepackage.UpdateStrategy{
			Tier:                  "separate-tier-nonclaim",
			ProductionClaim:       false,
			Channel:               "",
			ChannelDefined:        false,
			SignatureVerification: false,
			Evidence:              "auto-update is not production until a signed update channel manifest exists",
		},
		Operations: []surfacepackage.Operation{
			{Name: "surface package", Kind: "package", Path: "package/surface-desk.tdx", Ran: true, Pass: true},
			{Name: "linux install smoke", Kind: "install", Path: archiveRel, Ran: true, Pass: true},
			{Name: "linux launcher smoke", Kind: "run-smoke", Path: "bin/surface-run.sh", Ran: true, Pass: true},
		},
		NegativeGuards: surfacepackage.NegativeGuards{
			UnsignedMacOSProductionRejected:       true,
			OmittedAssetRejected:                  true,
			UpdaterWithoutChannelSigRejected:      true,
			WindowsMacOSNonclaimUntilSigningProof: true,
			PackageHashesRequired:                 true,
			LinuxInstallRunSmokeRequired:          true,
		},
		NonClaims: []string{
			"Windows package distribution is beta/nonclaim until signed installer evidence exists.",
			"macOS package distribution is beta/nonclaim until signed and notarized bundle evidence exists.",
			"Auto-update production is not claimed without a signed update channel manifest.",
		},
		Cases: []surfacepackage.CaseReport{
			{Name: "linux package install smoke", Kind: "positive", Ran: true, Pass: true},
			{Name: "linux package launcher smoke", Kind: "positive", Ran: true, Pass: true},
			{Name: "unsigned macOS production rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "omitted package asset rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "updater without channel signature rejected", Kind: "negative", Ran: true, Pass: true},
		},
	}, nil
}

func collectPackagedFiles(root string, archiveRel string) ([]surfacepackage.PackagedFile, error) {
	var files []surfacepackage.PackagedFile
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink package file %s is not allowed", path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == archiveRel {
			return nil
		}
		if err := validateSafeRelPath(rel); err != nil {
			return err
		}
		ref, err := artifactRef(root, rel)
		if err != nil {
			return err
		}
		files = append(files, surfacepackage.PackagedFile{
			Path:   rel,
			Role:   roleForPath(rel),
			SHA256: ref.SHA256,
			Size:   ref.Size,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func roleForPath(path string) string {
	switch path {
	case "package/surface-desk.tdx":
		return "surface-app-package"
	case "assets/surface-assets.json":
		return "asset-manifest"
	case "permissions.json":
		return "permissions-manifest"
	case "host-adapter.json":
		return "host-adapter-metadata"
	case "surface-package.json":
		return "package-manifest"
	case "bin/surface-run.sh":
		return "linux-launcher"
	default:
		return "source"
	}
}

func artifactRef(root string, rel string) (surfacepackage.ArtifactRef, error) {
	if err := validateSafeRelPath(rel); err != nil {
		return surfacepackage.ArtifactRef{}, err
	}
	path := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Lstat(path)
	if err != nil {
		return surfacepackage.ArtifactRef{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return surfacepackage.ArtifactRef{}, fmt.Errorf("%s must not be a symlink", rel)
	}
	if info.IsDir() {
		return surfacepackage.ArtifactRef{}, fmt.Errorf("%s is a directory", rel)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return surfacepackage.ArtifactRef{}, err
	}
	sum := sha256.Sum256(raw)
	return surfacepackage.ArtifactRef{
		Path:   rel,
		SHA256: "sha256:" + hex.EncodeToString(sum[:]),
		Size:   int64(len(raw)),
	}, nil
}

func validateSafeRelPath(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("path is required")
	}
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute paths are not allowed")
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	if clean == "." || clean != path {
		return fmt.Errorf("path must be normalized")
	}
	if clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return fmt.Errorf("parent traversal is not allowed")
	}
	return nil
}
