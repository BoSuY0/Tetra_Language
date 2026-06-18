package surface

import (
	"errors"
	"fmt"
	"strings"
)

type SecurityPermissionReport struct {
	Schema                     string                            `json:"schema"`
	Model                      string                            `json:"model"`
	ReleaseScope               string                            `json:"release_scope"`
	Source                     string                            `json:"source"`
	AppShellFeatures           string                            `json:"app_shell_features"`
	ProductionClaim            bool                              `json:"production_claim"`
	Experimental               bool                              `json:"experimental"`
	DefaultDeny                bool                              `json:"default_deny"`
	ShellFeaturePolicyEnforced bool                              `json:"shell_feature_policy_enforced"`
	Capabilities               []SurfaceSecurityCapabilityReport `json:"capabilities"`
	Permissions                []SurfacePermissionReport         `json:"permissions"`
	ProcessBoundaries          []SurfaceProcessBoundaryReport    `json:"process_boundaries"`
	AssetSafety                []SurfaceAssetSafetyReport        `json:"asset_safety"`
	UnsupportedClaims          []string                          `json:"unsupported_claims"`
	NegativeGuards             SurfaceSecurityNegativeGuards     `json:"negative_guards"`
}

type SurfaceSecurityCapabilityReport struct {
	Name              string `json:"name"`
	SourceFeature     string `json:"source_feature"`
	Status            string `json:"status"`
	Allowed           bool   `json:"allowed"`
	CapabilityChecked bool   `json:"capability_checked"`
	HostTrace         bool   `json:"host_trace"`
	Policy            string `json:"policy"`
	Evidence          string `json:"evidence"`
	BlockedReason     string `json:"blocked_reason"`
	Pass              bool   `json:"pass"`
}

type SurfacePermissionReport struct {
	Name              string `json:"name"`
	Status            string `json:"status"`
	Allowed           bool   `json:"allowed"`
	CapabilityChecked bool   `json:"capability_checked"`
	BlockedReason     string `json:"blocked_reason"`
	Evidence          string `json:"evidence"`
	Pass              bool   `json:"pass"`
}

type SurfaceProcessBoundaryReport struct {
	Name              string `json:"name"`
	SchemaChecked     bool   `json:"schema_checked"`
	CapabilityChecked bool   `json:"capability_checked"`
	UserJS            bool   `json:"user_js"`
	NodeIntegration   bool   `json:"node_integration"`
	ElectronRuntime   bool   `json:"electron_runtime"`
	Pass              bool   `json:"pass"`
}

type SurfaceAssetSafetyReport struct {
	Kind                string `json:"kind"`
	LocalOnly           bool   `json:"local_only"`
	SHA256Required      bool   `json:"sha256_required"`
	SizeLimitBytes      int    `json:"size_limit_bytes"`
	NetworkFetchAllowed bool   `json:"network_fetch_allowed"`
	Parser              string `json:"parser"`
	BoundsChecked       bool   `json:"bounds_checked"`
	Pass                bool   `json:"pass"`
}

type SurfaceSecurityNegativeGuards struct {
	NoAmbientFilesystem                       bool `json:"no_ambient_filesystem"`
	NoAmbientNetwork                          bool `json:"no_ambient_network"`
	NoShellFeatureBypass                      bool `json:"no_shell_feature_bypass"`
	NoPermissionlessClipboard                 bool `json:"no_permissionless_clipboard"`
	NoNotificationDialogWithoutTargetEvidence bool `json:"no_notification_dialog_without_target_evidence"`
	NoNetworkAssetFetch                       bool `json:"no_network_asset_fetch"`
	NoUntrustedFontImageDecode                bool `json:"no_untrusted_font_image_decode"`
	NoElectronNodeIntegration                 bool `json:"no_electron_node_integration"`
	NoUserJSAppLogic                          bool `json:"no_user_js_app_logic"`
	NoDOMAppUITree                            bool `json:"no_dom_app_ui_tree"`
}

func ValidateSecurityPermissionReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	switch schema {
	case SecurityPermissionSchemaV1:
		var report SecurityPermissionReport
		if err := decodeStrict(raw, &report); err != nil {
			return err
		}
		issues := validateSecurityPermissionReport(report, nil, "")
		if len(issues) > 0 {
			return errors.New(strings.Join(issues, "; "))
		}
		return nil
	case SchemaV1:
		var report Report
		if err := decodeStrict(raw, &report); err != nil {
			return err
		}
		issues := validateSecurityPermissionEvidence(report)
		if len(issues) > 0 {
			return errors.New(strings.Join(issues, "; "))
		}
		return nil
	default:
		return fmt.Errorf("schema is %q, want %q or %q", schema, SecurityPermissionSchemaV1, SchemaV1)
	}
}

func validateSecurityPermissionEvidence(report Report) []string {
	if report.SecurityPermissions == nil {
		if isLinuxAppShellReport(report) {
			return []string{"security_permissions evidence is required for linux app-shell reports"}
		}
		return nil
	}
	var features []LinuxAppShellFeatureReport
	if report.LinuxAppShell != nil {
		features = report.LinuxAppShell.ShellFeatures
	}
	return validateSecurityPermissionReport(*report.SecurityPermissions, features, report.Source)
}

func validateSecurityPermissionReport(report SecurityPermissionReport, features []LinuxAppShellFeatureReport, source string) []string {
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: report.Schema, want: SecurityPermissionSchemaV1},
		{field: "model", got: report.Model, want: "surface-security-permission-v1"},
		{field: "release_scope", got: report.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
		{field: "app_shell_features", got: report.AppShellFeatures, want: "electron-feature-ledger-v1"},
	} {
		if check.got != check.want {
			issues = append(issues, fmt.Sprintf("security_permissions %s is %q, want %q", check.field, check.got, check.want))
		}
	}
	if strings.TrimSpace(source) != "" && normalizeEvidencePath(report.Source) != normalizeEvidencePath(source) {
		issues = append(issues, fmt.Sprintf("security_permissions source %q must match report source %q", report.Source, source))
	}
	if strings.TrimSpace(report.Source) == "" {
		issues = append(issues, "security_permissions source is required")
	}
	if !report.ProductionClaim {
		issues = append(issues, "security_permissions production_claim must be true")
	}
	if report.Experimental {
		issues = append(issues, "security_permissions experimental must be false")
	}
	if !report.DefaultDeny {
		issues = append(issues, "security_permissions default_deny must be true")
	}
	if !report.ShellFeaturePolicyEnforced {
		issues = append(issues, "security_permissions shell_feature_policy_enforced must be true")
	}
	issues = append(issues, validateSecurityCapabilityRows(report.Capabilities, features)...)
	issues = append(issues, validateSecurityPermissionRows(report.Permissions)...)
	issues = append(issues, validateSurfaceSecurityProcessBoundaries(report.ProcessBoundaries)...)
	issues = append(issues, validateSurfaceSecurityAssetSafety(report.AssetSafety)...)
	issues = append(issues, validateSurfaceSecurityUnsupportedClaims(report.UnsupportedClaims)...)
	issues = append(issues, validateSurfaceSecurityNegativeGuards(report.NegativeGuards)...)
	return issues
}

func validateSecurityCapabilityRows(rows []SurfaceSecurityCapabilityReport, features []LinuxAppShellFeatureReport) []string {
	var issues []string
	if len(rows) == 0 {
		return []string{"security_permissions capabilities evidence is required"}
	}
	capabilities := map[string]SurfaceSecurityCapabilityReport{}
	for _, row := range rows {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			issues = append(issues, "security_permissions capability name is required")
			continue
		}
		capabilities[name] = row
		if !linuxAppShellKnownFeature(name) {
			issues = append(issues, fmt.Sprintf("security_permissions capability %s is not a known app-shell feature", name))
		}
		if row.SourceFeature != name {
			issues = append(issues, fmt.Sprintf("security_permissions capability %s source_feature is %q, want %q", name, row.SourceFeature, name))
		}
		if !row.CapabilityChecked || !row.HostTrace || !row.Pass {
			issues = append(issues, fmt.Sprintf("security_permissions capability %s requires capability_checked=true, host_trace=true, and pass=true", name))
		}
		if strings.TrimSpace(row.Policy) == "" || strings.TrimSpace(row.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("security_permissions capability %s requires policy and evidence", name))
		}
	}
	for _, feature := range features {
		name := strings.TrimSpace(feature.Name)
		if name == "" {
			continue
		}
		capability, ok := capabilities[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("security_permissions capabilities missing %s", name))
			continue
		}
		issues = append(issues, validateSecurityCapabilityAgainstFeature(capability, feature)...)
	}
	return issues
}

func validateSecurityCapabilityAgainstFeature(capability SurfaceSecurityCapabilityReport, feature LinuxAppShellFeatureReport) []string {
	name := feature.Name
	switch feature.Status {
	case "target_evidenced", "scoped_adapter":
		var issues []string
		if capability.Status != "allowed_with_policy" || !capability.Allowed {
			issues = append(issues, fmt.Sprintf("security_permissions capability %s must be allowed_with_policy for claimed app-shell feature", name))
		}
		if !capability.CapabilityChecked || !capability.HostTrace || !capability.Pass {
			issues = append(issues, fmt.Sprintf("security_permissions capability %s requires checked target-host evidence", name))
		}
		if strings.TrimSpace(capability.BlockedReason) != "" {
			issues = append(issues, fmt.Sprintf("security_permissions capability %s must not carry blocked_reason when allowed", name))
		}
		return issues
	case "blocked_pass":
		if capability.Status != "blocked_nonclaim" || capability.Allowed || strings.TrimSpace(capability.BlockedReason) == "" {
			return []string{fmt.Sprintf("security_permissions capability %s must remain blocked_nonclaim and cannot bypass the P16 blocked feature ledger", name)}
		}
		return nil
	default:
		return []string{fmt.Sprintf("security_permissions capability %s references unsupported feature status %q", name, feature.Status)}
	}
}

func validateSecurityPermissionRows(rows []SurfacePermissionReport) []string {
	var issues []string
	if len(rows) == 0 {
		return []string{"security_permissions permissions evidence is required"}
	}
	permissions := map[string]SurfacePermissionReport{}
	for _, row := range rows {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			issues = append(issues, "security_permissions permission name is required")
			continue
		}
		permissions[name] = row
		if !row.CapabilityChecked || !row.Pass {
			issues = append(issues, fmt.Sprintf("security_permissions permission %s requires capability_checked=true and pass=true", name))
		}
		if strings.TrimSpace(row.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("security_permissions permission %s evidence is required", name))
		}
	}
	for _, name := range []string{"filesystem", "network"} {
		row, ok := permissions[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("security_permissions permissions missing %s", name))
			continue
		}
		if row.Status != "denied" || row.Allowed || strings.TrimSpace(row.BlockedReason) == "" {
			issues = append(issues, fmt.Sprintf("security_permissions permission %s must be denied by default with blocked_reason", name))
		}
	}
	if row, ok := permissions["clipboard"]; !ok {
		issues = append(issues, "security_permissions permissions missing clipboard")
	} else if row.Status != "allowed_with_policy" || !row.Allowed || !row.CapabilityChecked || strings.TrimSpace(row.Evidence) == "" {
		issues = append(issues, "security_permissions permission clipboard must be allowed_with_policy with host evidence")
	}
	for _, name := range []string{"notifications", "dialogs", "shell_open_url"} {
		row, ok := permissions[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("security_permissions permissions missing %s", name))
			continue
		}
		if row.Status != "denied" || row.Allowed || strings.TrimSpace(row.BlockedReason) == "" {
			issues = append(issues, fmt.Sprintf("security_permissions permission %s must be denied until target evidence exists", name))
		}
	}
	return issues
}

func validateSurfaceSecurityProcessBoundaries(rows []SurfaceProcessBoundaryReport) []string {
	var issues []string
	boundaries := map[string]SurfaceProcessBoundaryReport{}
	for _, row := range rows {
		name := strings.TrimSpace(row.Name)
		if name == "" {
			issues = append(issues, "security_permissions process_boundary name is required")
			continue
		}
		boundaries[name] = row
		if !row.SchemaChecked || !row.CapabilityChecked || !row.Pass {
			issues = append(issues, fmt.Sprintf("security_permissions process_boundary %s requires schema_checked, capability_checked, and pass", name))
		}
		if row.UserJS || row.NodeIntegration || row.ElectronRuntime {
			issues = append(issues, fmt.Sprintf("security_permissions process_boundary %s must reject user JS app logic, Node integration, and Electron runtime", name))
		}
	}
	for _, name := range []string{"surface_app_to_host_abi", "linux_app_shell_host_adapter", "browser_canvas_host"} {
		if _, ok := boundaries[name]; !ok {
			issues = append(issues, fmt.Sprintf("security_permissions process_boundaries missing %s", name))
		}
	}
	return issues
}

func validateSurfaceSecurityAssetSafety(rows []SurfaceAssetSafetyReport) []string {
	var issues []string
	assets := map[string]SurfaceAssetSafetyReport{}
	for _, row := range rows {
		kind := strings.TrimSpace(row.Kind)
		if kind == "" {
			issues = append(issues, "security_permissions asset_safety kind is required")
			continue
		}
		assets[kind] = row
		if !validBlockAssetKind(kind) {
			issues = append(issues, fmt.Sprintf("security_permissions asset_safety kind %s is unsupported", kind))
		}
		if !row.LocalOnly || !row.SHA256Required || row.SizeLimitBytes <= 0 || row.NetworkFetchAllowed || strings.TrimSpace(row.Parser) == "" || !row.BoundsChecked || !row.Pass {
			issues = append(issues, fmt.Sprintf("security_permissions asset_safety %s requires local_only, sha256, positive size limit, no network fetch, parser, bounds check, and pass", kind))
		}
	}
	for _, kind := range []string{"font", "image", "icon"} {
		if _, ok := assets[kind]; !ok {
			issues = append(issues, fmt.Sprintf("security_permissions asset_safety missing %s", kind))
		}
	}
	return issues
}

func validateSurfaceSecurityUnsupportedClaims(claims []string) []string {
	var issues []string
	for _, claim := range []string{
		"unrestricted-filesystem",
		"unrestricted-network",
		"native-permission-prompts",
		"production-notifications",
		"production-dialogs",
		"remote-asset-fetch",
		"electron-node-integration",
	} {
		if !stringSliceContainsFold(claims, claim) {
			issues = append(issues, fmt.Sprintf("security_permissions unsupported_claims requires %s", claim))
		}
	}
	return issues
}

func validateSurfaceSecurityNegativeGuards(guards SurfaceSecurityNegativeGuards) []string {
	var issues []string
	for _, check := range []struct {
		field string
		ok    bool
	}{
		{field: "no_ambient_filesystem", ok: guards.NoAmbientFilesystem},
		{field: "no_ambient_network", ok: guards.NoAmbientNetwork},
		{field: "no_shell_feature_bypass", ok: guards.NoShellFeatureBypass},
		{field: "no_permissionless_clipboard", ok: guards.NoPermissionlessClipboard},
		{field: "no_notification_dialog_without_target_evidence", ok: guards.NoNotificationDialogWithoutTargetEvidence},
		{field: "no_network_asset_fetch", ok: guards.NoNetworkAssetFetch},
		{field: "no_untrusted_font_image_decode", ok: guards.NoUntrustedFontImageDecode},
		{field: "no_electron_node_integration", ok: guards.NoElectronNodeIntegration},
		{field: "no_user_js_app_logic", ok: guards.NoUserJSAppLogic},
		{field: "no_dom_app_ui_tree", ok: guards.NoDOMAppUITree},
	} {
		if !check.ok {
			issues = append(issues, fmt.Sprintf("security_permissions negative_guards.%s must be true", check.field))
		}
	}
	return issues
}
