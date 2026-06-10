package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surfacesecurity"
)

func TestValidateSurfaceSecurityReportCommandAcceptsValidReport(t *testing.T) {
	dir := t.TempDir()
	report := commandSecurityReport()
	reportPath := filepath.Join(dir, "surface-security-report.json")
	writeJSON(t, reportPath, report)

	var stdout, stderr bytes.Buffer
	code := runValidateSurfaceSecurityReport([]string{"--report", reportPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "surface security report OK") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestValidateSurfaceSecurityReportCommandRejectsUserJS(t *testing.T) {
	dir := t.TempDir()
	report := commandSecurityReport()
	report.SupplyChain.Dependencies = append(report.SupplyChain.Dependencies, surfacesecurity.Dependency{
		Name:     "runtime-script",
		Kind:     "user-js",
		Allowed:  true,
		Evidence: "user script runtime",
	})
	reportPath := filepath.Join(dir, "surface-security-report.json")
	writeJSON(t, reportPath, report)

	var stdout, stderr bytes.Buffer
	code := runValidateSurfaceSecurityReport([]string{"--report", reportPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected nonzero exit, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "user-js") && !strings.Contains(stderr.String(), "user JS") {
		t.Fatalf("stderr = %q, want user-js rejection", stderr.String())
	}
}

func commandSecurityReport() surfacesecurity.Report {
	return surfacesecurity.Report{
		Schema:       surfacesecurity.SchemaV1,
		Status:       "pass",
		Level:        surfacesecurity.LevelSecuritySandboxV1,
		Scope:        "surface-v1-scoped-linux-web-security",
		ReleaseScope: "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
		Permissions: surfacesecurity.PermissionModel{
			Policy: "explicit-deny-by-default",
			Manifest: surfacesecurity.ArtifactRef{
				Path:   "surface-permissions.json",
				SHA256: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				Size:   128,
			},
			Declared: []surfacesecurity.Permission{
				{Name: "filesystem", Mode: "app-bundle-readonly", Granted: true, Scope: "app-bundle", Evidence: "read-only package root"},
				{Name: "network", Mode: "denied", Granted: false, Scope: "none", Evidence: "no network host calls"},
				{Name: "clipboard", Mode: "host-gated", Granted: true, Scope: "user-gesture", Evidence: "host diagnostic required"},
				{Name: "window", Mode: "host-gated", Granted: true, Scope: "surface-window", Evidence: "target-host trace"},
				{Name: "open-url", Mode: "denied", Granted: false, Scope: "none", Evidence: "no external URL launch"},
				{Name: "notifications", Mode: "denied", Granted: false, Scope: "none", Evidence: "no notification permission"},
			},
		},
		HostCalls: []surfacesecurity.HostCall{
			{ID: "host.fs.bundle.read", Kind: "filesystem", Permission: "filesystem", Operation: "read-app-bundle", Allowed: true, Evidence: "app-bundle-readonly"},
			{ID: "host.network.fetch.rejected", Kind: "network", Permission: "network", Operation: "fetch", Allowed: false, Evidence: "network denied diagnostic"},
		},
		Assets: surfacesecurity.AssetSandbox{
			Policy:            "safe-local-assets-only",
			DecodeBeforeHash:  false,
			NetworkFetch:      false,
			UserScriptAllowed: false,
			Items: []surfacesecurity.AssetItem{
				{ID: "font-ui", Kind: "font", Source: "local", Trusted: true, HashVerified: true, Sanitized: true, Decoder: "font-table-hash-verified-v1", Accepted: true},
				{ID: "icon-vector", Kind: "svg", Source: "local", Trusted: true, HashVerified: true, Sanitized: true, Decoder: "svg-tiny-static-sanitized-v1", Accepted: true},
				{ID: "remote-logo", Kind: "image", Source: "remote", Trusted: false, HashVerified: false, Sanitized: false, Decoder: "none", Accepted: false},
			},
		},
		IPC: surfacesecurity.IPCModel{
			Policy: "typed-host-abi-only",
			Channels: []surfacesecurity.IPCChannel{
				{Name: "surface.host.window", Direction: "host", Typed: true, Authenticated: true},
			},
		},
		SupplyChain: surfacesecurity.SupplyChain{
			CapsuleVerified:       true,
			PackageHashesVerified: true,
			LockfileRequired:      true,
			NoPostinstallScripts:  true,
			Dependencies: []surfacesecurity.Dependency{
				{Name: "tetra-surface-app", Kind: "tetra-package", Allowed: true, Evidence: "sha256 package report"},
				{Name: "electron", Kind: "electron", Allowed: false, Evidence: "runtime dependency rejected"},
				{Name: "react", Kind: "react", Allowed: false, Evidence: "runtime dependency rejected"},
			},
		},
		Operations: []surfacesecurity.Operation{
			{Name: "permissions manifest validated", Kind: "permissions", Ran: true, Pass: true},
			{Name: "asset sandbox validated", Kind: "asset-sandbox", Ran: true, Pass: true},
			{Name: "ipc policy validated", Kind: "ipc", Ran: true, Pass: true},
			{Name: "supply-chain policy validated", Kind: "supply-chain", Ran: true, Pass: true},
		},
		NegativeGuards: surfacesecurity.NegativeGuards{
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
		Cases: []surfacesecurity.CaseReport{
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
