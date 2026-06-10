package surfacesecurity

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateReportAcceptsScopedSecuritySandbox(t *testing.T) {
	report := validSecurityReport()
	if err := ValidateReport(mustJSON(t, report)); err != nil {
		t.Fatalf("ValidateReport returned error: %v", err)
	}
}

func TestValidateReportRejectsNetworkAccessWithoutPermission(t *testing.T) {
	report := validSecurityReport()
	report.HostCalls = append(report.HostCalls, HostCall{
		ID:         "host.network.fetch",
		Kind:       "network",
		Permission: "network",
		Operation:  "fetch",
		Allowed:    true,
		Evidence:   "network fetch attempted without declared permission",
	})

	err := ValidateReport(mustJSON(t, report))
	if err == nil {
		t.Fatal("expected network host call without permission to be rejected")
	}
	if !strings.Contains(err.Error(), "network") || !strings.Contains(err.Error(), "permission") {
		t.Fatalf("error = %q, want network permission rejection", err.Error())
	}
}

func TestValidateReportRejectsUnsafeUntrustedAssets(t *testing.T) {
	report := validSecurityReport()
	report.Assets.Items[1].Sanitized = false
	report.Assets.Items[1].Accepted = true

	err := ValidateReport(mustJSON(t, report))
	if err == nil {
		t.Fatal("expected unsafe SVG asset to be rejected")
	}
	if !strings.Contains(err.Error(), "SVG") && !strings.Contains(err.Error(), "asset") {
		t.Fatalf("error = %q, want unsafe SVG/asset rejection", err.Error())
	}
}

func TestValidateReportRejectsUserJSAndRemoteCodeExecution(t *testing.T) {
	report := validSecurityReport()
	report.IPC.UserJSBridge = true
	report.SupplyChain.Dependencies = append(report.SupplyChain.Dependencies, Dependency{
		Name:     "reactive-user-script",
		Kind:     "user-js",
		Allowed:  true,
		Evidence: "script bridge",
	})

	err := ValidateReport(mustJSON(t, report))
	if err == nil {
		t.Fatal("expected user JS bridge/dependency to be rejected")
	}
	if !strings.Contains(err.Error(), "user JS") && !strings.Contains(err.Error(), "user-js") {
		t.Fatalf("error = %q, want user JS rejection", err.Error())
	}
}

func validSecurityReport() Report {
	return Report{
		Schema:       SchemaV1,
		Status:       "pass",
		Level:        LevelSecuritySandboxV1,
		Scope:        "surface-v1-scoped-linux-web-security",
		ReleaseScope: "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
		Permissions: PermissionModel{
			Policy: "explicit-deny-by-default",
			Manifest: ArtifactRef{
				Path:   "surface-permissions.json",
				SHA256: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				Size:   128,
			},
			Declared: []Permission{
				{Name: "filesystem", Mode: "app-bundle-readonly", Granted: true, Scope: "app-bundle", Evidence: "read-only package root"},
				{Name: "network", Mode: "denied", Granted: false, Scope: "none", Evidence: "no network host calls"},
				{Name: "clipboard", Mode: "host-gated", Granted: true, Scope: "user-gesture", Evidence: "host diagnostic required"},
				{Name: "window", Mode: "host-gated", Granted: true, Scope: "surface-window", Evidence: "target-host trace"},
				{Name: "open-url", Mode: "denied", Granted: false, Scope: "none", Evidence: "no default external URL launch"},
				{Name: "notifications", Mode: "denied", Granted: false, Scope: "none", Evidence: "no notification permission"},
			},
		},
		HostCalls: []HostCall{
			{ID: "host.fs.bundle.read", Kind: "filesystem", Permission: "filesystem", Operation: "read-app-bundle", Allowed: true, Evidence: "app-bundle-readonly"},
			{ID: "host.clipboard.write", Kind: "clipboard", Permission: "clipboard", Operation: "write-text", Allowed: true, Evidence: "host-gated user gesture"},
			{ID: "host.network.fetch.rejected", Kind: "network", Permission: "network", Operation: "fetch", Allowed: false, Evidence: "network denied diagnostic"},
		},
		Assets: AssetSandbox{
			Policy:            "safe-local-assets-only",
			DecodeBeforeHash:  false,
			NetworkFetch:      false,
			UserScriptAllowed: false,
			Items: []AssetItem{
				{ID: "font-ui", Kind: "font", Source: "local", Trusted: true, HashVerified: true, Sanitized: true, Decoder: "font-table-hash-verified-v1", Accepted: true},
				{ID: "icon-vector", Kind: "svg", Source: "local", Trusted: true, HashVerified: true, Sanitized: true, Decoder: "svg-tiny-static-sanitized-v1", Accepted: true},
				{ID: "remote-logo", Kind: "image", Source: "remote", Trusted: false, HashVerified: false, Sanitized: false, Decoder: "none", Accepted: false},
			},
		},
		IPC: IPCModel{
			Policy:              "typed-host-abi-only",
			UserJSBridge:        false,
			RawEval:             false,
			RemoteCodeExecution: false,
			Channels: []IPCChannel{
				{Name: "surface.host.window", Direction: "host", Typed: true, Authenticated: true},
				{Name: "surface.host.clipboard", Direction: "host", Typed: true, Authenticated: true},
			},
		},
		SupplyChain: SupplyChain{
			CapsuleVerified:       true,
			PackageHashesVerified: true,
			LockfileRequired:      true,
			NoPostinstallScripts:  true,
			Dependencies: []Dependency{
				{Name: "tetra-surface-app", Kind: "tetra-package", Allowed: true, Evidence: "sha256 package report"},
				{Name: "electron", Kind: "electron", Allowed: false, Evidence: "runtime dependency rejected"},
				{Name: "react", Kind: "react", Allowed: false, Evidence: "runtime dependency rejected"},
				{Name: "user-script", Kind: "user-js", Allowed: false, Evidence: "user JS rejected"},
			},
		},
		Operations: []Operation{
			{Name: "permissions manifest validated", Kind: "permissions", Ran: true, Pass: true},
			{Name: "asset sandbox validated", Kind: "asset-sandbox", Ran: true, Pass: true},
			{Name: "ipc policy validated", Kind: "ipc", Ran: true, Pass: true},
			{Name: "supply-chain policy validated", Kind: "supply-chain", Ran: true, Pass: true},
		},
		NegativeGuards: NegativeGuards{
			FilesystemWithoutPermissionRejected: true,
			NetworkWithoutPermissionRejected:    true,
			ClipboardWithoutPermissionRejected:  true,
			UnsafeSVGRejected:                   true,
			UntrustedFontRejected:               true,
			UserJSRejected:                      true,
			RemoteCodeExecutionRejected:         true,
			PackageWithoutHashesRejected:        true,
			IPCUntypedRejected:                  true,
		},
		NonClaims: []string{
			"No network access by default.",
			"No filesystem access outside the app bundle by default.",
			"No user JavaScript, remote code execution, Electron, React, DOM UI, or browser plugin sandbox.",
			"No arbitrary untrusted SVG/font/image decoder support.",
		},
		Cases: []CaseReport{
			{Name: "network without permission rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "filesystem without permission rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "clipboard without permission rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "untrusted SVG rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "user JS rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "package without hashes rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "typed IPC only", Kind: "positive", Ran: true, Pass: true},
		},
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
