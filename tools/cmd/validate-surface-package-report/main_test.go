package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surfacepackage"
)

func TestValidateSurfacePackageReportCommandAcceptsValidReport(t *testing.T) {
	root, report := writeCommandPackageFixture(t)
	reportPath := filepath.Join(root, "surface-package-report.json")
	writeJSON(t, reportPath, report)

	var stdout, stderr bytes.Buffer
	code := runValidateSurfacePackageReport([]string{"--report", reportPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "surface package report OK") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestValidateSurfacePackageReportCommandRejectsTamperedPackage(t *testing.T) {
	root, report := writeCommandPackageFixture(t)
	reportPath := filepath.Join(root, "surface-package-report.json")
	tampered := bytes.Repeat([]byte("x"), int(report.Linux.Package.Size))
	if err := os.WriteFile(filepath.Join(root, report.Linux.Package.Path), tampered, 0o644); err != nil {
		t.Fatal(err)
	}
	writeJSON(t, reportPath, report)

	var stdout, stderr bytes.Buffer
	code := runValidateSurfacePackageReport([]string{"--report", reportPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected nonzero exit for tampered package, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "sha256") {
		t.Fatalf("stderr = %q, want package hash mismatch", stderr.String())
	}
}

func writeCommandPackageFixture(t *testing.T) (string, surfacepackage.Report) {
	t.Helper()
	root := t.TempDir()
	files := map[string][]byte{
		"app/SurfaceDesk/Capsule.t4":  []byte("package SurfaceDesk\n"),
		"app/SurfaceDesk/src/main.t4": []byte("fn main() -> i32 { return 0 }\n"),
		"package/surface-desk.tdx":    []byte("surface tdx bytes\n"),
		"assets/surface-assets.json":  []byte(`{"schema":"tetra.surface.assets.v1","assets":[]}` + "\n"),
		"permissions.json":            []byte(`{"schema":"tetra.surface.permissions.v1","allow":[]}` + "\n"),
		"host-adapter.json":           []byte(`{"schema":"tetra.surface.host-adapter.v1","target":"linux-x64"}` + "\n"),
		"surface-package.json":        []byte(`{"schema":"tetra.surface.package.manifest.v1","entry":"app/SurfaceDesk/Capsule.t4"}` + "\n"),
		"bin/surface-run.sh":          []byte("#!/usr/bin/env bash\nset -euo pipefail\necho surface package launcher smoke\n"),
	}
	for rel, raw := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	archiveRel := "surface-desk-linux-x64.tar.gz"
	archiveRaw := makeCommandTarGz(t, files)
	if err := os.WriteFile(filepath.Join(root, archiveRel), archiveRaw, 0o644); err != nil {
		t.Fatal(err)
	}

	var packagedFiles []surfacepackage.PackagedFile
	for _, rel := range []string{
		"app/SurfaceDesk/Capsule.t4",
		"app/SurfaceDesk/src/main.t4",
		"package/surface-desk.tdx",
		"assets/surface-assets.json",
		"permissions.json",
		"host-adapter.json",
		"surface-package.json",
		"bin/surface-run.sh",
	} {
		role := "source"
		switch rel {
		case "package/surface-desk.tdx":
			role = "surface-app-package"
		case "assets/surface-assets.json":
			role = "asset-manifest"
		case "permissions.json":
			role = "permissions-manifest"
		case "host-adapter.json":
			role = "host-adapter-metadata"
		case "surface-package.json":
			role = "package-manifest"
		case "bin/surface-run.sh":
			role = "linux-launcher"
		}
		packagedFiles = append(packagedFiles, surfacepackage.PackagedFile{
			Path:   rel,
			Role:   role,
			SHA256: commandSHA256(files[rel]),
			Size:   int64(len(files[rel])),
		})
	}

	return root, surfacepackage.Report{
		Schema:       surfacepackage.SchemaV1,
		Status:       "pass",
		Level:        surfacepackage.LevelSurfacePackageDistributionV1,
		Scope:        "surface-v1-scoped-linux-web-package",
		ReleaseScope: "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
		Linux: surfacepackage.LinuxPackage{
			Target:       "linux-x64",
			SupportLevel: "production",
			Format:       "tar.gz",
			InstallSmoke: true,
			RunSmoke:     true,
			SameCommit:   true,
			Package:      surfacepackage.ArtifactRef{Path: archiveRel, SHA256: commandSHA256(archiveRaw), Size: int64(len(archiveRaw))},
			Manifest:     surfacepackage.ArtifactRef{Path: "surface-package.json", SHA256: commandSHA256(files["surface-package.json"]), Size: int64(len(files["surface-package.json"]))},
			AssetManifest: surfacepackage.ArtifactRef{
				Path: "assets/surface-assets.json", SHA256: commandSHA256(files["assets/surface-assets.json"]), Size: int64(len(files["assets/surface-assets.json"]))},
			PermissionsManifest: surfacepackage.ArtifactRef{Path: "permissions.json", SHA256: commandSHA256(files["permissions.json"]), Size: int64(len(files["permissions.json"]))},
			HostAdapterMetadata: surfacepackage.ArtifactRef{Path: "host-adapter.json", SHA256: commandSHA256(files["host-adapter.json"]), Size: int64(len(files["host-adapter.json"]))},
			Files:               packagedFiles,
			Signature: surfacepackage.SignatureState{
				Kind:             "sha256-checksum-manifest",
				ChecksumManifest: true,
				Verification:     "artifact-hashes",
			},
		},
		Targets: []surfacepackage.TargetStatus{
			{Target: "windows-x64", Tier: "beta-nonclaim", Evidence: "blocked until signed installer evidence exists"},
			{Target: "macos-x64", Tier: "beta-nonclaim", Evidence: "blocked until signing and notarization evidence exists"},
		},
		Update: surfacepackage.UpdateStrategy{
			Tier:     "separate-tier-nonclaim",
			Evidence: "auto-update is not production until signed channel manifest exists",
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
	}
}

func makeCommandTarGz(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, rel := range []string{
		"app/SurfaceDesk/Capsule.t4",
		"app/SurfaceDesk/src/main.t4",
		"package/surface-desk.tdx",
		"assets/surface-assets.json",
		"permissions.json",
		"host-adapter.json",
		"surface-package.json",
		"bin/surface-run.sh",
	} {
		raw := files[rel]
		if err := tw.WriteHeader(&tar.Header{Name: rel, Mode: 0o644, Size: int64(len(raw))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(raw); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func commandSHA256(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}
