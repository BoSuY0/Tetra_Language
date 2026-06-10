package surfacepackage

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
)

func TestValidateReportAcceptsScopedLinuxPackage(t *testing.T) {
	root, report := writePackageFixture(t, true)
	raw := mustJSON(t, report)

	if err := ValidateReportWithRoot(raw, root); err != nil {
		t.Fatalf("ValidateReportWithRoot returned error: %v", err)
	}
}

func TestValidateReportRejectsUnsignedMacOSProductionClaim(t *testing.T) {
	root, report := writePackageFixture(t, true)
	report.Targets[1].Tier = "production"
	report.Targets[1].ProductionClaim = true
	report.Targets[1].Signed = false
	report.Targets[1].Notarized = false

	err := ValidateReportWithRoot(mustJSON(t, report), root)
	if err == nil {
		t.Fatal("expected unsigned macOS production package claim to be rejected")
	}
	if !strings.Contains(err.Error(), "macOS") || !strings.Contains(err.Error(), "signed") || !strings.Contains(err.Error(), "notarized") {
		t.Fatalf("error = %q, want macOS signing/notarization rejection", err.Error())
	}
}

func TestValidateReportRejectsOmittedAssetFromPackageArchive(t *testing.T) {
	root, report := writePackageFixture(t, false)

	err := ValidateReportWithRoot(mustJSON(t, report), root)
	if err == nil {
		t.Fatal("expected package archive missing declared asset manifest to be rejected")
	}
	if !strings.Contains(err.Error(), "assets/surface-assets.json") {
		t.Fatalf("error = %q, want omitted asset manifest path", err.Error())
	}
}

func TestValidateReportRejectsUpdaterClaimWithoutChannelSignature(t *testing.T) {
	root, report := writePackageFixture(t, true)
	report.Update.ProductionClaim = true
	report.Update.Channel = ""
	report.Update.SignatureVerification = false

	err := ValidateReportWithRoot(mustJSON(t, report), root)
	if err == nil {
		t.Fatal("expected updater production claim without channel/signature verification to be rejected")
	}
	if !strings.Contains(err.Error(), "update") || !strings.Contains(err.Error(), "channel") || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("error = %q, want updater channel/signature rejection", err.Error())
	}
}

func writePackageFixture(t *testing.T, includeAssetManifestInArchive bool) (string, Report) {
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
	archiveEntries := make(map[string][]byte, len(files))
	for rel, raw := range files {
		if rel == "assets/surface-assets.json" && !includeAssetManifestInArchive {
			continue
		}
		archiveEntries[rel] = raw
	}
	archiveRel := "surface-desk-linux-x64.tar.gz"
	archiveRaw := makeTarGz(t, archiveEntries)
	if err := os.WriteFile(filepath.Join(root, archiveRel), archiveRaw, 0o644); err != nil {
		t.Fatal(err)
	}

	packagedFiles := make([]PackagedFile, 0, len(files))
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
		packagedFiles = append(packagedFiles, PackagedFile{
			Path:   rel,
			Role:   role,
			SHA256: sha256Hex(files[rel]),
			Size:   int64(len(files[rel])),
		})
	}

	return root, Report{
		Schema:       SchemaV1,
		Status:       "pass",
		Level:        LevelSurfacePackageDistributionV1,
		Scope:        "surface-v1-scoped-linux-web-package",
		ReleaseScope: "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
		Linux: LinuxPackage{
			Target:       "linux-x64",
			SupportLevel: "production",
			Format:       "tar.gz",
			InstallSmoke: true,
			RunSmoke:     true,
			SameCommit:   true,
			Package:      artifactRef(archiveRel, archiveRaw),
			Manifest:     artifactRef("surface-package.json", files["surface-package.json"]),
			AssetManifest: artifactRef("assets/surface-assets.json",
				files["assets/surface-assets.json"]),
			PermissionsManifest: artifactRef("permissions.json",
				files["permissions.json"]),
			HostAdapterMetadata: artifactRef("host-adapter.json",
				files["host-adapter.json"]),
			Files: packagedFiles,
			Signature: SignatureState{
				Kind:             "sha256-checksum-manifest",
				ChecksumManifest: true,
				Signed:           false,
				Notarized:        false,
				Verification:     "artifact-hashes",
			},
		},
		Targets: []TargetStatus{
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
		Update: UpdateStrategy{
			Tier:                  "separate-tier-nonclaim",
			ProductionClaim:       false,
			Channel:               "",
			ChannelDefined:        false,
			SignatureVerification: false,
			Evidence:              "auto-update is not production until a signed channel manifest exists",
		},
		Operations: []Operation{
			{Name: "surface package", Kind: "package", Path: "package/surface-desk.tdx", Ran: true, Pass: true},
			{Name: "linux install smoke", Kind: "install", Path: archiveRel, Ran: true, Pass: true},
			{Name: "linux launcher smoke", Kind: "run-smoke", Path: "bin/surface-run.sh", Ran: true, Pass: true},
		},
		NegativeGuards: NegativeGuards{
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
		Cases: []CaseReport{
			{Name: "linux package install smoke", Kind: "positive", Ran: true, Pass: true},
			{Name: "linux package launcher smoke", Kind: "positive", Ran: true, Pass: true},
			{Name: "unsigned macOS production rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "omitted package asset rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "updater without channel signature rejected", Kind: "negative", Ran: true, Pass: true},
		},
	}
}

func makeTarGz(t *testing.T, files map[string][]byte) []byte {
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
		raw, ok := files[rel]
		if !ok {
			continue
		}
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

func artifactRef(path string, raw []byte) ArtifactRef {
	return ArtifactRef{Path: path, SHA256: sha256Hex(raw), Size: int64(len(raw))}
}

func sha256Hex(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
