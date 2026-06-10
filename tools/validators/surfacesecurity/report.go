package surfacesecurity

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

const (
	SchemaV1               = "tetra.surface.security-report.v1"
	LevelSecuritySandboxV1 = "surface-security-sandbox-v1"
)

type Report struct {
	Schema         string          `json:"schema"`
	Status         string          `json:"status"`
	Level          string          `json:"level"`
	Scope          string          `json:"scope"`
	ReleaseScope   string          `json:"release_scope"`
	Producer       string          `json:"producer,omitempty"`
	GitHead        string          `json:"git_head,omitempty"`
	Version        string          `json:"version,omitempty"`
	Permissions    PermissionModel `json:"permissions"`
	HostCalls      []HostCall      `json:"host_calls"`
	Assets         AssetSandbox    `json:"assets"`
	IPC            IPCModel        `json:"ipc"`
	SupplyChain    SupplyChain     `json:"supply_chain"`
	Operations     []Operation     `json:"operations"`
	NegativeGuards NegativeGuards  `json:"negative_guards"`
	NonClaims      []string        `json:"nonclaims"`
	Cases          []CaseReport    `json:"cases"`
}

type PermissionModel struct {
	Policy   string       `json:"policy"`
	Manifest ArtifactRef  `json:"manifest"`
	Declared []Permission `json:"declared"`
}

type ArtifactRef struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type Permission struct {
	Name     string `json:"name"`
	Mode     string `json:"mode"`
	Granted  bool   `json:"granted"`
	Scope    string `json:"scope"`
	Evidence string `json:"evidence"`
}

type HostCall struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Permission string `json:"permission"`
	Operation  string `json:"operation"`
	Allowed    bool   `json:"allowed"`
	Evidence   string `json:"evidence"`
}

type AssetSandbox struct {
	Policy            string      `json:"policy"`
	DecodeBeforeHash  bool        `json:"decode_before_hash"`
	NetworkFetch      bool        `json:"network_fetch"`
	UserScriptAllowed bool        `json:"user_script_allowed"`
	Items             []AssetItem `json:"items"`
}

type AssetItem struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	Source       string `json:"source"`
	Trusted      bool   `json:"trusted"`
	HashVerified bool   `json:"hash_verified"`
	Sanitized    bool   `json:"sanitized"`
	Decoder      string `json:"decoder"`
	Accepted     bool   `json:"accepted"`
}

type IPCModel struct {
	Policy              string       `json:"policy"`
	UserJSBridge        bool         `json:"user_js_bridge"`
	RawEval             bool         `json:"raw_eval"`
	RemoteCodeExecution bool         `json:"remote_code_execution"`
	Channels            []IPCChannel `json:"channels"`
}

type IPCChannel struct {
	Name          string `json:"name"`
	Direction     string `json:"direction"`
	Typed         bool   `json:"typed"`
	Authenticated bool   `json:"authenticated"`
}

type SupplyChain struct {
	CapsuleVerified       bool         `json:"capsule_verified"`
	PackageHashesVerified bool         `json:"package_hashes_verified"`
	LockfileRequired      bool         `json:"lockfile_required"`
	NoPostinstallScripts  bool         `json:"no_postinstall_scripts"`
	Dependencies          []Dependency `json:"dependencies"`
}

type Dependency struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Allowed  bool   `json:"allowed"`
	Evidence string `json:"evidence"`
}

type Operation struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Ran  bool   `json:"ran"`
	Pass bool   `json:"pass"`
}

type NegativeGuards struct {
	FilesystemWithoutPermissionRejected bool `json:"filesystem_without_permission_rejected"`
	NetworkWithoutPermissionRejected    bool `json:"network_without_permission_rejected"`
	ClipboardWithoutPermissionRejected  bool `json:"clipboard_without_permission_rejected"`
	UnsafeSVGRejected                   bool `json:"unsafe_svg_rejected"`
	UntrustedFontRejected               bool `json:"untrusted_font_rejected"`
	UserJSRejected                      bool `json:"user_js_rejected"`
	RemoteCodeExecutionRejected         bool `json:"remote_code_execution_rejected"`
	PackageWithoutHashesRejected        bool `json:"package_without_hashes_rejected"`
	IPCUntypedRejected                  bool `json:"ipc_untyped_rejected"`
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
	return Validate(report)
}

func Validate(report Report) error {
	var issues []string
	issues = append(issues, validateIdentity(report)...)
	permissions, permissionIssues := validatePermissions(report.Permissions)
	issues = append(issues, permissionIssues...)
	issues = append(issues, validateHostCalls(report.HostCalls, permissions)...)
	issues = append(issues, validateAssets(report.Assets)...)
	issues = append(issues, validateIPC(report.IPC)...)
	issues = append(issues, validateSupplyChain(report.SupplyChain)...)
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

func validateIdentity(report Report) []string {
	var issues []string
	if report.Schema != SchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", report.Schema, SchemaV1))
	}
	if report.Status != "pass" {
		issues = append(issues, fmt.Sprintf("status is %q, want pass", report.Status))
	}
	if report.Level != LevelSecuritySandboxV1 {
		issues = append(issues, fmt.Sprintf("level is %q, want %q", report.Level, LevelSecuritySandboxV1))
	}
	if report.Scope != "surface-v1-scoped-linux-web-security" {
		issues = append(issues, fmt.Sprintf("scope is %q, want surface-v1-scoped-linux-web-security", report.Scope))
	}
	if report.ReleaseScope != "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI" {
		issues = append(issues, fmt.Sprintf("release_scope is %q, want PROD_STABLE_SCOPED_LINUX_WEB_APP_UI", report.ReleaseScope))
	}
	return issues
}

func validatePermissions(model PermissionModel) (map[string]Permission, []string) {
	var issues []string
	if model.Policy != "explicit-deny-by-default" {
		issues = append(issues, fmt.Sprintf("permissions policy is %q, want explicit-deny-by-default", model.Policy))
	}
	issues = append(issues, validateArtifactRef("permissions manifest", model.Manifest)...)
	if len(model.Declared) == 0 {
		issues = append(issues, "declared permissions are required")
	}
	perms := map[string]Permission{}
	for i, permission := range model.Declared {
		prefix := fmt.Sprintf("permissions.declared[%d]", i)
		if strings.TrimSpace(permission.Name) == "" || strings.TrimSpace(permission.Mode) == "" || strings.TrimSpace(permission.Scope) == "" {
			issues = append(issues, prefix+" requires name, mode, and scope")
		}
		if strings.TrimSpace(permission.Evidence) == "" {
			issues = append(issues, prefix+" evidence is required")
		}
		if _, exists := perms[permission.Name]; exists {
			issues = append(issues, fmt.Sprintf("duplicate permission %s", permission.Name))
		}
		perms[permission.Name] = permission
		if deniedMode(permission.Mode) && permission.Granted {
			issues = append(issues, fmt.Sprintf("permission %s cannot be granted with denied mode %q", permission.Name, permission.Mode))
		}
	}
	for _, required := range []string{"filesystem", "network", "clipboard", "window", "open-url", "notifications"} {
		if _, ok := perms[required]; !ok {
			issues = append(issues, fmt.Sprintf("declared permissions missing %s", required))
		}
	}
	return perms, issues
}

func validateHostCalls(calls []HostCall, permissions map[string]Permission) []string {
	if len(calls) == 0 {
		return []string{"host call audit evidence is required"}
	}
	var issues []string
	for i, call := range calls {
		prefix := fmt.Sprintf("host_calls[%d]", i)
		if strings.TrimSpace(call.ID) == "" || strings.TrimSpace(call.Kind) == "" || strings.TrimSpace(call.Permission) == "" || strings.TrimSpace(call.Operation) == "" {
			issues = append(issues, prefix+" requires id, kind, permission, and operation")
		}
		if strings.TrimSpace(call.Evidence) == "" {
			issues = append(issues, prefix+" evidence is required")
		}
		permission, ok := permissions[call.Permission]
		if !ok {
			issues = append(issues, fmt.Sprintf("%s references undeclared permission %s", call.ID, call.Permission))
			continue
		}
		if call.Allowed && (!permission.Granted || deniedMode(permission.Mode)) {
			issues = append(issues, fmt.Sprintf("%s %s host call allowed without %s permission", call.Kind, call.ID, call.Permission))
		}
		if !call.Allowed && permission.Granted && call.Kind != "network" {
			continue
		}
	}
	return issues
}

func validateAssets(assets AssetSandbox) []string {
	var issues []string
	if assets.Policy != "safe-local-assets-only" {
		issues = append(issues, fmt.Sprintf("asset sandbox policy is %q, want safe-local-assets-only", assets.Policy))
	}
	if assets.DecodeBeforeHash {
		issues = append(issues, "asset sandbox must reject decoder execution before hash verification")
	}
	if assets.NetworkFetch {
		issues = append(issues, "asset sandbox must reject network fetches")
	}
	if assets.UserScriptAllowed {
		issues = append(issues, "asset sandbox must reject user JS execution")
	}
	if len(assets.Items) == 0 {
		issues = append(issues, "asset sandbox items are required")
	}
	for i, item := range assets.Items {
		prefix := fmt.Sprintf("assets.items[%d]", i)
		if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.Kind) == "" || strings.TrimSpace(item.Source) == "" || strings.TrimSpace(item.Decoder) == "" {
			issues = append(issues, prefix+" requires id, kind, source, and decoder")
		}
		if item.Accepted {
			if item.Source == "remote" || !item.Trusted {
				issues = append(issues, fmt.Sprintf("asset %s accepted untrusted or remote asset", item.ID))
			}
			if !item.HashVerified {
				issues = append(issues, fmt.Sprintf("asset %s accepted without hash verification", item.ID))
			}
			if item.Kind == "svg" && !item.Sanitized {
				issues = append(issues, fmt.Sprintf("asset %s accepted unsafe SVG without sanitization", item.ID))
			}
			if item.Kind == "font" && (!item.HashVerified || !strings.Contains(item.Decoder, "hash")) {
				issues = append(issues, fmt.Sprintf("asset %s accepted untrusted font without hash-verified decoder", item.ID))
			}
		}
	}
	return issues
}

func validateIPC(ipc IPCModel) []string {
	var issues []string
	if ipc.Policy != "typed-host-abi-only" {
		issues = append(issues, fmt.Sprintf("ipc policy is %q, want typed-host-abi-only", ipc.Policy))
	}
	if ipc.UserJSBridge {
		issues = append(issues, "IPC user JS bridge is rejected")
	}
	if ipc.RawEval {
		issues = append(issues, "IPC raw eval is rejected")
	}
	if ipc.RemoteCodeExecution {
		issues = append(issues, "IPC remote code execution is rejected")
	}
	if len(ipc.Channels) == 0 {
		issues = append(issues, "typed IPC channels are required")
	}
	for i, channel := range ipc.Channels {
		if strings.TrimSpace(channel.Name) == "" || strings.TrimSpace(channel.Direction) == "" {
			issues = append(issues, fmt.Sprintf("ipc.channels[%d] requires name and direction", i))
		}
		if strings.Contains(channel.Name, "*") {
			issues = append(issues, fmt.Sprintf("ipc channel %s wildcard channels are rejected", channel.Name))
		}
		if !channel.Typed {
			issues = append(issues, fmt.Sprintf("ipc channel %s must be typed", channel.Name))
		}
		if !channel.Authenticated {
			issues = append(issues, fmt.Sprintf("ipc channel %s must be authenticated", channel.Name))
		}
	}
	return issues
}

func validateSupplyChain(chain SupplyChain) []string {
	var issues []string
	if !chain.CapsuleVerified {
		issues = append(issues, "supply-chain requires capsule verification")
	}
	if !chain.PackageHashesVerified {
		issues = append(issues, "supply-chain requires package hash verification")
	}
	if !chain.LockfileRequired {
		issues = append(issues, "supply-chain requires lockfile policy")
	}
	if !chain.NoPostinstallScripts {
		issues = append(issues, "supply-chain must reject postinstall scripts")
	}
	if len(chain.Dependencies) == 0 {
		issues = append(issues, "supply-chain dependency audit is required")
	}
	for i, dep := range chain.Dependencies {
		if strings.TrimSpace(dep.Name) == "" || strings.TrimSpace(dep.Kind) == "" {
			issues = append(issues, fmt.Sprintf("supply_chain.dependencies[%d] requires name and kind", i))
		}
		if strings.TrimSpace(dep.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("dependency %s evidence is required", dep.Name))
		}
		if dep.Allowed && forbiddenDependencyKind(dep.Kind) {
			issues = append(issues, fmt.Sprintf("dependency %s kind %s is rejected as user JS/Electron/React/runtime code", dep.Name, dep.Kind))
		}
	}
	return issues
}

func validateOperations(operations []Operation) []string {
	required := map[string]bool{
		"permissions manifest validated": false,
		"asset sandbox validated":        false,
		"ipc policy validated":           false,
		"supply-chain policy validated":  false,
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
		"filesystem without permission rejection": guards.FilesystemWithoutPermissionRejected,
		"network without permission rejection":    guards.NetworkWithoutPermissionRejected,
		"clipboard without permission rejection":  guards.ClipboardWithoutPermissionRejected,
		"unsafe SVG rejection":                    guards.UnsafeSVGRejected,
		"untrusted font rejection":                guards.UntrustedFontRejected,
		"user JS rejection":                       guards.UserJSRejected,
		"remote code execution rejection":         guards.RemoteCodeExecutionRejected,
		"package without hashes rejection":        guards.PackageWithoutHashesRejected,
		"untyped IPC rejection":                   guards.IPCUntypedRejected,
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
		return []string{"security nonclaims are required"}
	}
	joined := strings.ToLower(strings.Join(nonClaims, "\n"))
	var issues []string
	for _, required := range []string{"network", "filesystem", "user javascript", "remote code execution", "untrusted svg"} {
		if !strings.Contains(joined, required) {
			issues = append(issues, fmt.Sprintf("security nonclaims must mention %s boundary", required))
		}
	}
	return issues
}

func validateCases(cases []CaseReport) []string {
	required := map[string]bool{
		"network without permission rejected":    false,
		"filesystem without permission rejected": false,
		"clipboard without permission rejected":  false,
		"untrusted SVG rejected":                 false,
		"user JS rejected":                       false,
		"package without hashes rejected":        false,
		"typed IPC only":                         false,
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

func validateArtifactRef(name string, ref ArtifactRef) []string {
	var issues []string
	if strings.TrimSpace(ref.Path) == "" {
		issues = append(issues, name+" path is required")
	}
	if ref.Size <= 0 {
		issues = append(issues, name+" size must be positive")
	}
	if !validSHA256(ref.SHA256) {
		issues = append(issues, name+" sha256 must be sha256:<64-hex>")
	}
	return issues
}

func validSHA256(value string) bool {
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

func deniedMode(mode string) bool {
	return mode == "denied" || mode == "not-requested" || mode == "none"
}

func forbiddenDependencyKind(kind string) bool {
	switch strings.ToLower(kind) {
	case "electron", "react", "user-js", "javascript", "npm-script", "postinstall", "remote-code":
		return true
	default:
		return false
	}
}
