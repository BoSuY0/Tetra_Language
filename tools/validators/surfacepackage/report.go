package surfacepackage

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	SchemaV1                          = "tetra.surface.package-report.v1"
	LevelSurfacePackageDistributionV1 = "surface-package-distribution-v1"
)

type Report struct {
	Schema         string         `json:"schema"`
	Status         string         `json:"status"`
	Level          string         `json:"level"`
	Scope          string         `json:"scope"`
	ReleaseScope   string         `json:"release_scope"`
	Producer       string         `json:"producer,omitempty"`
	GitHead        string         `json:"git_head,omitempty"`
	Version        string         `json:"version,omitempty"`
	Linux          LinuxPackage   `json:"linux"`
	Targets        []TargetStatus `json:"targets"`
	Update         UpdateStrategy `json:"update"`
	Operations     []Operation    `json:"operations"`
	NegativeGuards NegativeGuards `json:"negative_guards"`
	NonClaims      []string       `json:"nonclaims"`
	Cases          []CaseReport   `json:"cases"`
}

type LinuxPackage struct {
	Target              string         `json:"target"`
	SupportLevel        string         `json:"support_level"`
	Format              string         `json:"format"`
	InstallSmoke        bool           `json:"install_smoke"`
	RunSmoke            bool           `json:"run_smoke"`
	SameCommit          bool           `json:"same_commit"`
	Package             ArtifactRef    `json:"package"`
	Manifest            ArtifactRef    `json:"manifest"`
	AssetManifest       ArtifactRef    `json:"asset_manifest"`
	PermissionsManifest ArtifactRef    `json:"permissions_manifest"`
	HostAdapterMetadata ArtifactRef    `json:"host_adapter_metadata"`
	Files               []PackagedFile `json:"files"`
	Signature           SignatureState `json:"signature"`
}

type ArtifactRef struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type PackagedFile struct {
	Path   string `json:"path"`
	Role   string `json:"role"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type SignatureState struct {
	Kind             string `json:"kind"`
	ChecksumManifest bool   `json:"checksum_manifest"`
	Signed           bool   `json:"signed"`
	Notarized        bool   `json:"notarized"`
	Certificate      string `json:"certificate,omitempty"`
	Verification     string `json:"verification"`
}

type TargetStatus struct {
	Target           string `json:"target"`
	Tier             string `json:"tier"`
	ProductionClaim  bool   `json:"production_claim"`
	Signed           bool   `json:"signed"`
	Notarized        bool   `json:"notarized"`
	InstallerPath    string `json:"installer_path,omitempty"`
	SigningPath      string `json:"signing_path,omitempty"`
	NotarizationPath string `json:"notarization_path,omitempty"`
	Evidence         string `json:"evidence"`
}

type UpdateStrategy struct {
	Tier                  string      `json:"tier"`
	ProductionClaim       bool        `json:"production_claim"`
	Channel               string      `json:"channel"`
	ChannelDefined        bool        `json:"channel_defined"`
	SignatureVerification bool        `json:"signature_verification"`
	Manifest              ArtifactRef `json:"manifest,omitempty"`
	Evidence              string      `json:"evidence"`
}

type Operation struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Path string `json:"path,omitempty"`
	Ran  bool   `json:"ran"`
	Pass bool   `json:"pass"`
}

type NegativeGuards struct {
	UnsignedMacOSProductionRejected       bool `json:"unsigned_macos_production_rejected"`
	OmittedAssetRejected                  bool `json:"omitted_asset_rejected"`
	UpdaterWithoutChannelSigRejected      bool `json:"updater_without_channel_sig_rejected"`
	WindowsMacOSNonclaimUntilSigningProof bool `json:"windows_macos_nonclaim_until_signing_proof"`
	PackageHashesRequired                 bool `json:"package_hashes_required"`
	LinuxInstallRunSmokeRequired          bool `json:"linux_install_run_smoke_required"`
}

type CaseReport struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Ran  bool   `json:"ran"`
	Pass bool   `json:"pass"`
}

func ValidateReport(raw []byte) error {
	report, err := decodeReport(raw)
	if err != nil {
		return err
	}
	return validateReport(report)
}

func ValidateReportWithRoot(raw []byte, root string) error {
	report, err := decodeReport(raw)
	if err != nil {
		return err
	}
	if err := validateReport(report); err != nil {
		return err
	}
	return validateArtifactFiles(report, root)
}

func decodeReport(raw []byte) (Report, error) {
	var report Report
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&report); err != nil {
		return Report{}, err
	}
	if err := ensureJSONEOF(dec); err != nil {
		return Report{}, err
	}
	return report, nil
}

func ensureJSONEOF(dec *json.Decoder) error {
	var extra any
	if err := dec.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return fmt.Errorf("unexpected trailing JSON payload")
}

func validateReport(report Report) error {
	var issues []string
	issues = append(issues, validateIdentity(report)...)
	issues = append(issues, validateLinuxPackage(report.Linux)...)
	issues = append(issues, validateTargets(report.Targets)...)
	issues = append(issues, validateUpdate(report.Update)...)
	issues = append(issues, validateOperations(report.Operations)...)
	issues = append(issues, validateNegativeGuards(report.NegativeGuards)...)
	issues = append(issues, validateNonClaims(report.NonClaims)...)
	issues = append(issues, validateCases(report.Cases)...)
	if len(issues) > 0 {
		sort.Strings(issues)
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateIdentity(report Report) []string {
	var issues []string
	if report.Schema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, SchemaV1))
	}
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if report.Level != LevelSurfacePackageDistributionV1 {
		issues = append(issues, fmt.Sprintf("level is %q, want %q", report.Level, LevelSurfacePackageDistributionV1))
	}
	if report.Scope != "surface-v1-scoped-linux-web-package" {
		issues = append(issues, fmt.Sprintf("scope is %q, want surface-v1-scoped-linux-web-package", report.Scope))
	}
	if report.ReleaseScope != "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI" {
		issues = append(issues, fmt.Sprintf("release_scope is %q, want PROD_STABLE_SCOPED_LINUX_WEB_APP_UI", report.ReleaseScope))
	}
	return issues
}

func validateLinuxPackage(linux LinuxPackage) []string {
	var issues []string
	if linux.Target != "linux-x64" {
		issues = append(issues, fmt.Sprintf("linux target is %q, want linux-x64", linux.Target))
	}
	if linux.SupportLevel != "production" {
		issues = append(issues, fmt.Sprintf("linux support_level is %q, want production", linux.SupportLevel))
	}
	if linux.Format != "tar.gz" && linux.Format != "surface-linux-tar-v1" {
		issues = append(issues, fmt.Sprintf("linux package format is %q, want tar.gz or surface-linux-tar-v1", linux.Format))
	}
	if !linux.InstallSmoke || !linux.RunSmoke || !linux.SameCommit {
		issues = append(issues, "linux package requires install_smoke, run_smoke, and same_commit")
	}
	issues = append(issues, validateArtifactRef("linux package", linux.Package, false)...)
	issues = append(issues, validateArtifactRef("package manifest", linux.Manifest, false)...)
	issues = append(issues, validateArtifactRef("asset manifest", linux.AssetManifest, false)...)
	issues = append(issues, validateArtifactRef("permissions manifest", linux.PermissionsManifest, false)...)
	issues = append(issues, validateArtifactRef("host adapter metadata", linux.HostAdapterMetadata, false)...)
	issues = append(issues, validatePackagedFiles(linux)...)
	issues = append(issues, validateSignature(linux.Signature)...)
	return issues
}

func validatePackagedFiles(linux LinuxPackage) []string {
	if len(linux.Files) == 0 {
		return []string{"linux package files are required"}
	}
	requiredRoles := map[string]bool{
		"surface-app-package":   false,
		"asset-manifest":        false,
		"permissions-manifest":  false,
		"host-adapter-metadata": false,
		"package-manifest":      false,
		"linux-launcher":        false,
	}
	requiredPaths := map[string]string{
		linux.Manifest.Path:            "package-manifest",
		linux.AssetManifest.Path:       "asset-manifest",
		linux.PermissionsManifest.Path: "permissions-manifest",
		linux.HostAdapterMetadata.Path: "host-adapter-metadata",
	}
	var issues []string
	seen := map[string]bool{}
	for i, file := range linux.Files {
		prefix := fmt.Sprintf("linux files[%d]", i)
		if err := validateSafeRelPath(file.Path); err != nil {
			issues = append(issues, fmt.Sprintf("%s path: %v", prefix, err))
			continue
		}
		if seen[file.Path] {
			issues = append(issues, fmt.Sprintf("duplicate package file %s", file.Path))
		}
		seen[file.Path] = true
		if strings.TrimSpace(file.Role) == "" {
			issues = append(issues, fmt.Sprintf("%s role is required", prefix))
		}
		if _, ok := requiredRoles[file.Role]; ok {
			requiredRoles[file.Role] = true
		}
		if wantRole, ok := requiredPaths[file.Path]; ok && file.Role != wantRole {
			issues = append(issues, fmt.Sprintf("%s role is %q, want %q", file.Path, file.Role, wantRole))
		}
		if file.Size <= 0 {
			issues = append(issues, fmt.Sprintf("%s size must be positive", file.Path))
		}
		if !validSHA256(file.SHA256) {
			issues = append(issues, fmt.Sprintf("%s sha256 must be sha256:<64-hex>", file.Path))
		}
	}
	for role, ok := range requiredRoles {
		if !ok {
			issues = append(issues, fmt.Sprintf("linux package missing %s file", role))
		}
	}
	if !seen[linux.Manifest.Path] {
		issues = append(issues, fmt.Sprintf("linux package file list is missing package manifest %s", linux.Manifest.Path))
	}
	if !seen[linux.AssetManifest.Path] {
		issues = append(issues, fmt.Sprintf("linux package file list is missing asset manifest %s", linux.AssetManifest.Path))
	}
	if !seen[linux.PermissionsManifest.Path] {
		issues = append(issues, fmt.Sprintf("linux package file list is missing permissions manifest %s", linux.PermissionsManifest.Path))
	}
	if !seen[linux.HostAdapterMetadata.Path] {
		issues = append(issues, fmt.Sprintf("linux package file list is missing host adapter metadata %s", linux.HostAdapterMetadata.Path))
	}
	return issues
}

func validateSignature(sig SignatureState) []string {
	var issues []string
	if sig.Kind != "sha256-checksum-manifest" && sig.Kind != "signed-artifact" {
		issues = append(issues, fmt.Sprintf("signature kind is %q, want sha256-checksum-manifest or signed-artifact", sig.Kind))
	}
	if !sig.ChecksumManifest {
		issues = append(issues, "package signature evidence requires checksum_manifest")
	}
	if strings.TrimSpace(sig.Verification) == "" {
		issues = append(issues, "package signature verification is required")
	}
	if sig.Signed && strings.TrimSpace(sig.Certificate) == "" {
		issues = append(issues, "signed package evidence requires certificate")
	}
	return issues
}

func validateTargets(targets []TargetStatus) []string {
	if len(targets) == 0 {
		return []string{"target package status evidence is required"}
	}
	var issues []string
	seen := map[string]bool{}
	for _, target := range targets {
		if strings.TrimSpace(target.Target) == "" {
			issues = append(issues, "target package status target is required")
			continue
		}
		seen[target.Target] = true
		if strings.TrimSpace(target.Tier) == "" {
			issues = append(issues, fmt.Sprintf("%s package tier is required", target.Target))
		}
		if strings.TrimSpace(target.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("%s package evidence is required", target.Target))
		}
		production := target.ProductionClaim || target.Tier == "production"
		switch target.Target {
		case "linux-x64":
			if production && !target.Signed && target.Tier != "production-checksum-scoped" {
				issues = append(issues, "linux production package evidence must be scoped to checksum artifacts or signed packages")
			}
		case "windows-x64":
			if production && !target.Signed {
				issues = append(issues, "Windows production package claim requires signed installer evidence")
			}
			if target.Tier == "beta-nonclaim" && target.ProductionClaim {
				issues = append(issues, "Windows beta/nonclaim package tier cannot set production_claim")
			}
		case "macos-x64":
			if production && (!target.Signed || !target.Notarized) {
				issues = append(issues, "macOS production package claim requires signed and notarized evidence")
			}
			if target.Tier == "beta-nonclaim" && target.ProductionClaim {
				issues = append(issues, "macOS beta/nonclaim package tier cannot set production_claim")
			}
		default:
			issues = append(issues, fmt.Sprintf("unexpected package target %s", target.Target))
		}
	}
	for _, required := range []string{"windows-x64", "macos-x64"} {
		if !seen[required] {
			issues = append(issues, fmt.Sprintf("package report requires %s target nonclaim/signing status", required))
		}
	}
	return issues
}

func validateUpdate(update UpdateStrategy) []string {
	var issues []string
	if strings.TrimSpace(update.Tier) == "" {
		issues = append(issues, "update tier is required")
	}
	if strings.TrimSpace(update.Evidence) == "" {
		issues = append(issues, "update evidence is required")
	}
	if update.ProductionClaim {
		if !update.ChannelDefined || strings.TrimSpace(update.Channel) == "" {
			issues = append(issues, "update production claim requires a defined channel")
		}
		if !update.SignatureVerification {
			issues = append(issues, "update production claim requires signature verification")
		}
		issues = append(issues, validateArtifactRef("update manifest", update.Manifest, false)...)
	} else if update.Tier != "separate-tier-nonclaim" && update.Tier != "beta-nonclaim" {
		issues = append(issues, fmt.Sprintf("update tier is %q, want separate-tier-nonclaim or beta-nonclaim until signed channel evidence exists", update.Tier))
	}
	return issues
}

func validateOperations(operations []Operation) []string {
	required := map[string]bool{
		"surface package":      false,
		"linux install smoke":  false,
		"linux launcher smoke": false,
	}
	var issues []string
	for i, op := range operations {
		if strings.TrimSpace(op.Name) == "" || strings.TrimSpace(op.Kind) == "" {
			issues = append(issues, fmt.Sprintf("operations[%d] requires name and kind", i))
		}
		if !op.Ran || !op.Pass {
			issues = append(issues, fmt.Sprintf("operation %q must run and pass", op.Name))
		}
		if _, ok := required[op.Name]; ok {
			required[op.Name] = true
		}
		if strings.TrimSpace(op.Path) != "" {
			if err := validateSafeRelPath(op.Path); err != nil {
				issues = append(issues, fmt.Sprintf("operation %q path: %v", op.Name, err))
			}
		}
	}
	for name, ok := range required {
		if !ok {
			issues = append(issues, fmt.Sprintf("operation %q is required", name))
		}
	}
	return issues
}

func validateNegativeGuards(guards NegativeGuards) []string {
	checks := map[string]bool{
		"unsigned macOS production rejection":      guards.UnsignedMacOSProductionRejected,
		"omitted package asset rejection":          guards.OmittedAssetRejected,
		"updater without channel/signature reject": guards.UpdaterWithoutChannelSigRejected,
		"Windows/macOS nonclaim signing boundary":  guards.WindowsMacOSNonclaimUntilSigningProof,
		"package hash requirement":                 guards.PackageHashesRequired,
		"linux install/run smoke requirement":      guards.LinuxInstallRunSmokeRequired,
	}
	var issues []string
	for name, ok := range checks {
		if !ok {
			issues = append(issues, name+" guard is required")
		}
	}
	return issues
}

func validateNonClaims(nonClaims []string) []string {
	if len(nonClaims) == 0 {
		return []string{"package report nonclaims are required"}
	}
	joined := strings.ToLower(strings.Join(nonClaims, "\n"))
	var issues []string
	for _, required := range []string{"windows", "macos", "update"} {
		if !strings.Contains(joined, required) {
			issues = append(issues, fmt.Sprintf("package report nonclaims must mention %s boundary", required))
		}
	}
	if strings.Contains(joined, "electron replacement") && !strings.Contains(joined, "scoped") {
		issues = append(issues, "broad Electron replacement wording must stay scoped")
	}
	return issues
}

func validateCases(cases []CaseReport) []string {
	required := map[string]bool{
		"linux package install smoke":                false,
		"linux package launcher smoke":               false,
		"unsigned macOS production rejected":         false,
		"omitted package asset rejected":             false,
		"updater without channel signature rejected": false,
	}
	var issues []string
	for i, c := range cases {
		if strings.TrimSpace(c.Name) == "" || strings.TrimSpace(c.Kind) == "" {
			issues = append(issues, fmt.Sprintf("cases[%d] requires name and kind", i))
		}
		if !c.Ran || !c.Pass {
			issues = append(issues, fmt.Sprintf("case %q must run and pass", c.Name))
		}
		if _, ok := required[c.Name]; ok {
			required[c.Name] = true
		}
	}
	for name, ok := range required {
		if !ok {
			issues = append(issues, fmt.Sprintf("case %q is required", name))
		}
	}
	return issues
}

func validateArtifactRef(name string, ref ArtifactRef, allowEmpty bool) []string {
	var issues []string
	if strings.TrimSpace(ref.Path) == "" {
		if allowEmpty {
			return nil
		}
		return []string{name + " path is required"}
	}
	if err := validateSafeRelPath(ref.Path); err != nil {
		issues = append(issues, fmt.Sprintf("%s path: %v", name, err))
	}
	if ref.Size <= 0 {
		issues = append(issues, fmt.Sprintf("%s size must be positive", name))
	}
	if !validSHA256(ref.SHA256) {
		issues = append(issues, fmt.Sprintf("%s sha256 must be sha256:<64-hex>", name))
	}
	return issues
}

func validateArtifactFiles(report Report, root string) error {
	if strings.TrimSpace(root) == "" {
		return fmt.Errorf("artifact root is required")
	}
	root = filepath.Clean(root)
	if err := validateRoot(root); err != nil {
		return err
	}
	var issues []string
	for name, ref := range map[string]ArtifactRef{
		"linux package":         report.Linux.Package,
		"package manifest":      report.Linux.Manifest,
		"asset manifest":        report.Linux.AssetManifest,
		"permissions manifest":  report.Linux.PermissionsManifest,
		"host adapter metadata": report.Linux.HostAdapterMetadata,
	} {
		if err := validateArtifactFile(root, name, ref); err != nil {
			issues = append(issues, err.Error())
		}
	}
	for _, file := range report.Linux.Files {
		ref := ArtifactRef{Path: file.Path, SHA256: file.SHA256, Size: file.Size}
		if err := validateArtifactFile(root, "packaged file "+file.Path, ref); err != nil {
			issues = append(issues, err.Error())
		}
	}
	if err := validateTarPackageContents(root, report.Linux); err != nil {
		issues = append(issues, err.Error())
	}
	if report.Update.ProductionClaim {
		if err := validateArtifactFile(root, "update manifest", report.Update.Manifest); err != nil {
			issues = append(issues, err.Error())
		}
	}
	if len(issues) > 0 {
		sort.Strings(issues)
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateRoot(root string) error {
	info, err := os.Lstat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("artifact root %s is not a directory", root)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("artifact root %s must not be a symlink", root)
	}
	return nil
}

func validateArtifactFile(root string, name string, ref ArtifactRef) error {
	if issues := validateArtifactRef(name, ref, false); len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	path, err := safeJoin(root, ref.Path)
	if err != nil {
		return fmt.Errorf("%s path: %v", name, err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("%s %s: %v", name, ref.Path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s %s must not be a symlink", name, ref.Path)
	}
	if info.IsDir() {
		return fmt.Errorf("%s %s is a directory", name, ref.Path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s %s: %v", name, ref.Path, err)
	}
	actualSize := int64(len(raw))
	actualSHA := hashBytes(raw)
	if actualSize != ref.Size {
		return fmt.Errorf("%s %s size mismatch: got %d want %d", name, ref.Path, actualSize, ref.Size)
	}
	if actualSHA != ref.SHA256 {
		return fmt.Errorf("%s %s sha256 mismatch: got %s want %s", name, ref.Path, actualSHA, ref.SHA256)
	}
	return nil
}

func validateTarPackageContents(root string, linux LinuxPackage) error {
	packagePath, err := safeJoin(root, linux.Package.Path)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(packagePath)
	if err != nil {
		return err
	}
	gz, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("linux package %s is not a readable gzip archive: %v", linux.Package.Path, err)
	}
	defer gz.Close()
	expected := map[string]PackagedFile{}
	for _, file := range linux.Files {
		expected[file.Path] = file
	}
	seen := map[string]bool{}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("linux package %s tar read failed: %v", linux.Package.Path, err)
		}
		name := filepath.ToSlash(hdr.Name)
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			continue
		}
		if err := validateSafeRelPath(name); err != nil {
			return fmt.Errorf("linux package archive path %s: %v", name, err)
		}
		if seen[name] {
			return fmt.Errorf("linux package archive duplicate file %s", name)
		}
		seen[name] = true
		want, ok := expected[name]
		if !ok {
			continue
		}
		body, err := io.ReadAll(tr)
		if err != nil {
			return fmt.Errorf("linux package archive file %s read failed: %v", name, err)
		}
		if int64(len(body)) != want.Size {
			return fmt.Errorf("linux package archive file %s size mismatch: got %d want %d", name, len(body), want.Size)
		}
		if got := hashBytes(body); got != want.SHA256 {
			return fmt.Errorf("linux package archive file %s sha256 mismatch: got %s want %s", name, got, want.SHA256)
		}
	}
	for _, file := range linux.Files {
		if !seen[file.Path] {
			return fmt.Errorf("linux package archive missing declared file %s", file.Path)
		}
	}
	return nil
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
	if strings.HasPrefix(clean, "../") || clean == ".." || strings.Contains(clean, "/../") {
		return fmt.Errorf("parent traversal is not allowed")
	}
	return nil
}

func safeJoin(root string, rel string) (string, error) {
	if err := validateSafeRelPath(rel); err != nil {
		return "", err
	}
	path := filepath.Join(root, filepath.FromSlash(rel))
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if cleanPath != cleanRoot && !strings.HasPrefix(cleanPath, cleanRoot+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes artifact root")
	}
	return cleanPath, nil
}

func validSHA256(value string) bool {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "sha256:") {
		return false
	}
	raw := strings.TrimPrefix(value, "sha256:")
	if len(raw) != 64 {
		return false
	}
	_, err := hex.DecodeString(raw)
	return err == nil
}

func hashBytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}
