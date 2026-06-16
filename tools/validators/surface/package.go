package surface

import (
	"errors"
	"fmt"
	"strings"
)

const SurfacePackageSchemaV1 = "tetra.surface.package.v1"

type SurfacePackageReport struct {
	Schema         string                       `json:"schema"`
	Model          string                       `json:"model"`
	ReleaseScope   string                       `json:"release_scope"`
	Producer       string                       `json:"producer"`
	Source         string                       `json:"source"`
	ReferenceApp   string                       `json:"reference_app"`
	PackageFormat  string                       `json:"package_format"`
	FormatVersion  int                          `json:"format_version"`
	ArtifactRoot   string                       `json:"artifact_root"`
	Packages       []SurfacePackageArtifact     `json:"packages"`
	Assets         []SurfacePackageAsset        `json:"assets"`
	InstallSmokes  []SurfacePackageInstallSmoke `json:"install_smokes"`
	WebBundles     []SurfacePackageWebBundle    `json:"web_bundles"`
	UpdateStrategy SurfacePackageUpdateStrategy `json:"update_strategy"`
	Signing        SurfacePackagePlatformProof  `json:"signing"`
	Notarization   SurfacePackagePlatformProof  `json:"notarization"`
	NegativeGuards SurfacePackageNegativeGuards `json:"negative_guards"`
	Pass           bool                         `json:"pass"`
}

type SurfacePackageArtifact struct {
	Target              string `json:"target"`
	Kind                string `json:"kind"`
	Path                string `json:"path"`
	ManifestPath        string `json:"manifest_path"`
	SHA256              string `json:"sha256"`
	AssetManifestSHA256 string `json:"asset_manifest_sha256"`
	SourceSHA256        string `json:"source_sha256"`
	BuildSHA256         string `json:"build_sha256"`
	ContainsExecutable  bool   `json:"contains_executable"`
	ContainsWebBundle   bool   `json:"contains_web_bundle"`
	LocalOnlyAssets     bool   `json:"local_only_assets"`
	Pass                bool   `json:"pass"`
}

type SurfacePackageAsset struct {
	Path                string `json:"path"`
	Kind                string `json:"kind"`
	SHA256              string `json:"sha256"`
	SizeBytes           int64  `json:"size_bytes"`
	LocalOnly           bool   `json:"local_only"`
	NetworkFetchAllowed bool   `json:"network_fetch_allowed"`
	Pass                bool   `json:"pass"`
}

type SurfacePackageInstallSmoke struct {
	Target                  string `json:"target"`
	PackagePath             string `json:"package_path"`
	InstallDir              string `json:"install_dir"`
	InstalledBinary         string `json:"installed_binary"`
	Command                 string `json:"command"`
	ExitCode                int    `json:"exit_code"`
	ExpectedExitCode        int    `json:"expected_exit_code"`
	ArtifactHashVerified    bool   `json:"artifact_hash_verified"`
	PackageManifestVerified bool   `json:"package_manifest_verified"`
	AppRun                  bool   `json:"app_run"`
	Pass                    bool   `json:"pass"`
}

type SurfacePackageWebBundle struct {
	Target                  string `json:"target"`
	PackagePath             string `json:"package_path"`
	WebEntry                string `json:"web_entry"`
	WASMArtifact            string `json:"wasm_artifact"`
	LoaderArtifact          string `json:"loader_artifact"`
	BrowserCanvasHost       string `json:"browser_canvas_host"`
	Command                 string `json:"command"`
	ArtifactHashVerified    bool   `json:"artifact_hash_verified"`
	PackageManifestVerified bool   `json:"package_manifest_verified"`
	Pass                    bool   `json:"pass"`
}

type SurfacePackageUpdateStrategy struct {
	Strategy                            string `json:"strategy"`
	ManifestFormat                      string `json:"manifest_format"`
	ChannelManifest                     string `json:"channel_manifest"`
	CurrentVersion                      string `json:"current_version"`
	LatestVersion                       string `json:"latest_version"`
	LatestPackagePath                   string `json:"latest_package_path"`
	LatestPackageSHA256                 string `json:"latest_package_sha256"`
	PackageHashPinned                   bool   `json:"package_hash_pinned"`
	RollbackManifest                    string `json:"rollback_manifest"`
	SignatureRequiredForStablePromotion bool   `json:"signature_required_for_stable_promotion"`
	AutoUpdateRuntimeClaim              bool   `json:"auto_update_runtime_claim"`
	NetworkUpdateClaim                  bool   `json:"network_update_claim"`
	Pass                                bool   `json:"pass"`
}

type SurfacePackagePlatformProof struct {
	Status          string `json:"status"`
	Signed          bool   `json:"signed"`
	Notarized       bool   `json:"notarized"`
	ProductionClaim bool   `json:"production_claim"`
	Evidence        string `json:"evidence"`
	BlockedReason   string `json:"blocked_reason"`
}

type SurfacePackageNegativeGuards struct {
	NoReactRuntime                        bool `json:"no_react_runtime"`
	NoElectronRuntime                     bool `json:"no_electron_runtime"`
	NoDOMAppUITree                        bool `json:"no_dom_app_ui_tree"`
	NoCSSRuntime                          bool `json:"no_css_runtime"`
	NoUserJSAppLogic                      bool `json:"no_user_js_app_logic"`
	NoRemoteAssetFetch                    bool `json:"no_remote_asset_fetch"`
	NoUnsignedSigningClaim                bool `json:"no_unsigned_signing_claim"`
	NoNotarizationWithoutPlatformEvidence bool `json:"no_notarization_without_platform_evidence"`
	NoAutoUpdateWithoutRuntimeEvidence    bool `json:"no_auto_update_without_runtime_evidence"`
	NoDocsOnlyPackageClaim                bool `json:"no_docs_only_package_claim"`
	InstallRunRequired                    bool `json:"install_run_required"`
	WebBundleRequired                     bool `json:"web_bundle_required"`
	ArtifactHashesRequired                bool `json:"artifact_hashes_required"`
}

func ValidatePackageReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != SurfacePackageSchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, SurfacePackageSchemaV1)
	}
	var report SurfacePackageReport
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}
	issues := validateSurfacePackageReport(report)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateSurfacePackageReport(report SurfacePackageReport) []string {
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: report.Schema, want: SurfacePackageSchemaV1},
		{field: "model", got: report.Model, want: "surface-package-v1"},
		{field: "release_scope", got: report.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
		{field: "producer", got: report.Producer, want: "scripts/release/surface/surface-package-smoke.sh"},
		{field: "package_format", got: report.PackageFormat, want: "surface-app-package-v1"},
	} {
		if check.got != check.want {
			issues = append(issues, fmt.Sprintf("%s is %q, want %q", check.field, check.got, check.want))
		}
	}
	if report.FormatVersion != 1 {
		issues = append(issues, fmt.Sprintf("format_version = %d, want 1", report.FormatVersion))
	}
	if !safeRelativeSourcePath(report.Source) {
		issues = append(issues, "source must be a safe Tetra source path")
	}
	if strings.TrimSpace(report.ReferenceApp) == "" {
		issues = append(issues, "reference_app is required")
	}
	if !surfacePackageSourceMatchesReferenceApp(report.ReferenceApp, report.Source) {
		issues = append(issues, fmt.Sprintf("reference_app %q does not match source %q", report.ReferenceApp, report.Source))
	}
	if !safeRelativeReportPath(report.ArtifactRoot) {
		issues = append(issues, "artifact_root is unsafe or empty")
	}
	issues = append(issues, validateSurfacePackageArtifacts(report.Packages)...)
	issues = append(issues, validateSurfacePackageAssets(report.Assets)...)
	issues = append(issues, validateSurfacePackageInstallSmokes(report.ReferenceApp, report.InstallSmokes)...)
	issues = append(issues, validateSurfacePackageWebBundles(report.WebBundles)...)
	issues = append(issues, validateSurfacePackageUpdateStrategy(report.UpdateStrategy)...)
	issues = append(issues, validateSurfacePackagePlatformProof("signing", report.Signing, false)...)
	issues = append(issues, validateSurfacePackagePlatformProof("notarization", report.Notarization, true)...)
	issues = append(issues, validateSurfacePackageNegativeGuards(report.NegativeGuards)...)
	if !report.Pass {
		issues = append(issues, "pass must be true")
	}
	return issues
}

func surfacePackageSourceMatchesReferenceApp(referenceApp string, source string) bool {
	want, ok := requiredSurfacePackageApps()[strings.TrimSpace(referenceApp)]
	return ok && normalizeEvidencePath(source) == want
}

func requiredSurfacePackageApps() map[string]string {
	return map[string]string{
		"command-palette": "examples/surface_reference_command_palette.tetra",
		"localized-form":  "examples/surface_reference_localized_form.tetra",
		"migration":       "examples/surface_reference_migration.tetra",
		"studio-shell":    "examples/surface_morph_rendered_studio_shell.tetra",
	}
}

func validateSurfacePackageArtifacts(packages []SurfacePackageArtifact) []string {
	if len(packages) == 0 {
		return []string{"packages are required"}
	}
	var issues []string
	seen := map[string]SurfacePackageArtifact{}
	for _, pkg := range packages {
		target := strings.TrimSpace(pkg.Target)
		if target == "" {
			issues = append(issues, "package target is required")
			continue
		}
		seen[target] = pkg
		prefix := "package " + target
		if !safeRelativeReportPath(pkg.Path) || !strings.HasSuffix(pkg.Path, ".tar.gz") {
			issues = append(issues, prefix+" path must be a safe .tar.gz report path")
		}
		if !safeRelativeReportPath(pkg.ManifestPath) || !strings.HasSuffix(pkg.ManifestPath, ".json") {
			issues = append(issues, prefix+" manifest_path must be a safe JSON report path")
		}
		for _, digest := range []struct {
			name  string
			value string
		}{
			{name: "sha256", value: pkg.SHA256},
			{name: "asset_manifest_sha256", value: pkg.AssetManifestSHA256},
			{name: "source_sha256", value: pkg.SourceSHA256},
			{name: "build_sha256", value: pkg.BuildSHA256},
		} {
			if !validChecksumLike(digest.value) {
				issues = append(issues, fmt.Sprintf("%s %s must be sha256 evidence", prefix, digest.name))
			}
		}
		if !pkg.LocalOnlyAssets {
			issues = append(issues, prefix+" local_only_assets must be true")
		}
		switch target {
		case "linux-x64":
			if pkg.Kind != "linux-x64-tar.gz" {
				issues = append(issues, fmt.Sprintf("%s kind is %q, want linux-x64-tar.gz", prefix, pkg.Kind))
			}
			if !pkg.ContainsExecutable {
				issues = append(issues, prefix+" must contain executable")
			}
			if pkg.ContainsWebBundle {
				issues = append(issues, prefix+" must not be marked as web bundle")
			}
		case "wasm32-web":
			if pkg.Kind != "wasm32-web-tar.gz" {
				issues = append(issues, fmt.Sprintf("%s kind is %q, want wasm32-web-tar.gz", prefix, pkg.Kind))
			}
			if !pkg.ContainsWebBundle {
				issues = append(issues, prefix+" must contain web bundle")
			}
		default:
			issues = append(issues, fmt.Sprintf("unsupported package target %q", target))
		}
		if !pkg.Pass {
			issues = append(issues, prefix+" pass must be true")
		}
	}
	for _, target := range []string{"linux-x64", "wasm32-web"} {
		if _, ok := seen[target]; !ok {
			issues = append(issues, "packages missing "+target)
		}
	}
	return issues
}

func validateSurfacePackageAssets(assets []SurfacePackageAsset) []string {
	if len(assets) == 0 {
		return []string{"assets are required"}
	}
	var issues []string
	for _, asset := range assets {
		prefix := "asset " + strings.TrimSpace(asset.Path)
		if !safeRelativeReportPath(asset.Path) {
			issues = append(issues, "asset path is unsafe or empty")
		}
		if strings.TrimSpace(asset.Kind) == "" {
			issues = append(issues, prefix+" kind is required")
		}
		if !validChecksumLike(asset.SHA256) {
			issues = append(issues, prefix+" sha256 must be sha256 evidence")
		}
		if asset.SizeBytes <= 0 {
			issues = append(issues, prefix+" size_bytes must be positive")
		}
		if !asset.LocalOnly {
			issues = append(issues, prefix+" local_only must be true")
		}
		if asset.NetworkFetchAllowed {
			issues = append(issues, prefix+" network_fetch_allowed must be false")
		}
		if !asset.Pass {
			issues = append(issues, prefix+" pass must be true")
		}
	}
	return issues
}

func validateSurfacePackageInstallSmokes(referenceApp string, smokes []SurfacePackageInstallSmoke) []string {
	var issues []string
	seenLinux := false
	for _, smoke := range smokes {
		prefix := "install smoke " + strings.TrimSpace(smoke.Target)
		if smoke.Target == "linux-x64" {
			seenLinux = true
		}
		if smoke.Target != "linux-x64" {
			issues = append(issues, prefix+" target must be linux-x64")
		}
		if !safeRelativeReportPath(smoke.PackagePath) || !strings.HasSuffix(smoke.PackagePath, ".tar.gz") {
			issues = append(issues, prefix+" package_path must be a safe .tar.gz path")
		}
		if !safeRelativeReportPath(smoke.InstallDir) {
			issues = append(issues, prefix+" install_dir is unsafe or empty")
		}
		if !safeRelativeReportPath(smoke.InstalledBinary) {
			issues = append(issues, prefix+" installed_binary is unsafe or empty")
		}
		if !strings.Contains(smoke.Command, smoke.InstalledBinary) {
			issues = append(issues, prefix+" command must execute installed_binary")
		}
		if smoke.ExpectedExitCode < 0 {
			issues = append(issues, fmt.Sprintf("%s expected_exit_code = %d, want non-negative", prefix, smoke.ExpectedExitCode))
		}
		if smoke.ExpectedExitCode != 0 {
			issues = append(issues, fmt.Sprintf("%s expected_exit_code = %d, want 0 for Surface package evidence", prefix, smoke.ExpectedExitCode))
		}
		if smoke.ExitCode != smoke.ExpectedExitCode {
			issues = append(issues, fmt.Sprintf("%s exit_code = %d, want expected_exit_code %d", prefix, smoke.ExitCode, smoke.ExpectedExitCode))
		}
		for _, check := range []struct {
			name string
			ok   bool
		}{
			{name: "artifact_hash_verified", ok: smoke.ArtifactHashVerified},
			{name: "package_manifest_verified", ok: smoke.PackageManifestVerified},
			{name: "app_run", ok: smoke.AppRun},
			{name: "pass", ok: smoke.Pass},
		} {
			if !check.ok {
				issues = append(issues, fmt.Sprintf("%s %s must be true", prefix, check.name))
			}
		}
	}
	if !seenLinux {
		issues = append(issues, "install_smokes missing linux-x64 install/run evidence")
	}
	return issues
}

func validateSurfacePackageWebBundles(bundles []SurfacePackageWebBundle) []string {
	var issues []string
	seenWeb := false
	for _, bundle := range bundles {
		prefix := "web bundle " + strings.TrimSpace(bundle.Target)
		if bundle.Target == "wasm32-web" {
			seenWeb = true
		}
		if bundle.Target != "wasm32-web" {
			issues = append(issues, prefix+" target must be wasm32-web")
		}
		if !safeRelativeReportPath(bundle.PackagePath) || !strings.HasSuffix(bundle.PackagePath, ".tar.gz") {
			issues = append(issues, prefix+" package_path must be a safe .tar.gz path")
		}
		if !safeRelativeReportPath(bundle.WebEntry) || !strings.HasSuffix(bundle.WebEntry, ".html") {
			issues = append(issues, prefix+" web_entry must be a safe HTML path")
		}
		if !safeRelativeReportPath(bundle.WASMArtifact) || !strings.HasSuffix(bundle.WASMArtifact, ".wasm") {
			issues = append(issues, prefix+" wasm_artifact must be a safe .wasm path")
		}
		if !safeRelativeReportPath(bundle.LoaderArtifact) || !strings.HasSuffix(bundle.LoaderArtifact, ".mjs") {
			issues = append(issues, prefix+" loader_artifact must be a safe .mjs path")
		}
		if !safeRelativeReportPath(bundle.BrowserCanvasHost) || !strings.HasSuffix(bundle.BrowserCanvasHost, ".mjs") {
			issues = append(issues, prefix+" browser_canvas_host must be a safe .mjs path")
		}
		if !strings.Contains(bundle.Command, "tetra build") || !strings.Contains(bundle.Command, "wasm32-web") {
			issues = append(issues, prefix+" command must build wasm32-web")
		}
		if !bundle.ArtifactHashVerified {
			issues = append(issues, prefix+" artifact_hash_verified must be true")
		}
		if !bundle.PackageManifestVerified {
			issues = append(issues, prefix+" package_manifest_verified must be true")
		}
		if !bundle.Pass {
			issues = append(issues, prefix+" pass must be true")
		}
	}
	if !seenWeb {
		issues = append(issues, "web_bundles missing wasm32-web bundle evidence")
	}
	return issues
}

func validateSurfacePackageUpdateStrategy(strategy SurfacePackageUpdateStrategy) []string {
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "update_strategy.strategy", got: strategy.Strategy, want: "hash-pinned-channel-manifest-v1"},
		{field: "update_strategy.manifest_format", got: strategy.ManifestFormat, want: "tetra.surface.update-channel.v1"},
	} {
		if check.got != check.want {
			issues = append(issues, fmt.Sprintf("%s is %q, want %q", check.field, check.got, check.want))
		}
	}
	for _, path := range []struct {
		name  string
		value string
	}{
		{name: "channel_manifest", value: strategy.ChannelManifest},
		{name: "latest_package_path", value: strategy.LatestPackagePath},
		{name: "rollback_manifest", value: strategy.RollbackManifest},
	} {
		if !safeRelativeReportPath(path.value) {
			issues = append(issues, fmt.Sprintf("update_strategy.%s is unsafe or empty", path.name))
		}
	}
	if strings.TrimSpace(strategy.CurrentVersion) == "" || strings.TrimSpace(strategy.LatestVersion) == "" {
		issues = append(issues, "update_strategy current_version and latest_version are required")
	}
	if !validChecksumLike(strategy.LatestPackageSHA256) {
		issues = append(issues, "update_strategy.latest_package_sha256 must be sha256 evidence")
	}
	if !strategy.PackageHashPinned {
		issues = append(issues, "update_strategy.package_hash_pinned must be true")
	}
	if !strategy.SignatureRequiredForStablePromotion {
		issues = append(issues, "update_strategy.signature_required_for_stable_promotion must be true")
	}
	if strategy.AutoUpdateRuntimeClaim {
		issues = append(issues, "update_strategy.auto_update_runtime_claim must be false without runtime updater evidence")
	}
	if strategy.NetworkUpdateClaim {
		issues = append(issues, "update_strategy.network_update_claim must be false without network updater evidence")
	}
	if !strategy.Pass {
		issues = append(issues, "update_strategy pass must be true")
	}
	return issues
}

func validateSurfacePackagePlatformProof(name string, proof SurfacePackagePlatformProof, notarization bool) []string {
	var issues []string
	if proof.Status != "nonclaim" {
		issues = append(issues, fmt.Sprintf("%s status is %q, want nonclaim", name, proof.Status))
	}
	if proof.Signed {
		issues = append(issues, fmt.Sprintf("%s must not claim signed package without platform signing evidence", name))
	}
	if notarization && proof.Notarized {
		issues = append(issues, fmt.Sprintf("%s must not claim notarization without platform evidence", name))
	}
	if !notarization && proof.Notarized {
		issues = append(issues, fmt.Sprintf("%s notarized must be false", name))
	}
	if proof.ProductionClaim {
		issues = append(issues, fmt.Sprintf("%s production_claim must be false", name))
	}
	if strings.TrimSpace(proof.Evidence) != "" {
		issues = append(issues, fmt.Sprintf("%s evidence must stay empty for nonclaim", name))
	}
	if strings.TrimSpace(proof.BlockedReason) == "" {
		issues = append(issues, fmt.Sprintf("%s blocked_reason is required", name))
	}
	return issues
}

func validateSurfacePackageNegativeGuards(guards SurfacePackageNegativeGuards) []string {
	var missing []string
	for _, check := range []struct {
		name string
		ok   bool
	}{
		{name: "no_react_runtime", ok: guards.NoReactRuntime},
		{name: "no_electron_runtime", ok: guards.NoElectronRuntime},
		{name: "no_dom_app_ui_tree", ok: guards.NoDOMAppUITree},
		{name: "no_css_runtime", ok: guards.NoCSSRuntime},
		{name: "no_user_js_app_logic", ok: guards.NoUserJSAppLogic},
		{name: "no_remote_asset_fetch", ok: guards.NoRemoteAssetFetch},
		{name: "no_unsigned_signing_claim", ok: guards.NoUnsignedSigningClaim},
		{name: "no_notarization_without_platform_evidence", ok: guards.NoNotarizationWithoutPlatformEvidence},
		{name: "no_auto_update_without_runtime_evidence", ok: guards.NoAutoUpdateWithoutRuntimeEvidence},
		{name: "no_docs_only_package_claim", ok: guards.NoDocsOnlyPackageClaim},
		{name: "install_run_required", ok: guards.InstallRunRequired},
		{name: "web_bundle_required", ok: guards.WebBundleRequired},
		{name: "artifact_hashes_required", ok: guards.ArtifactHashesRequired},
	} {
		if !check.ok {
			missing = append(missing, check.name)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return []string{fmt.Sprintf("negative_guards missing %s", strings.Join(missing, ", "))}
}
