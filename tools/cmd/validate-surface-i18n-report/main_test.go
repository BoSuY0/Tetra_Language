package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tetra_language/tools/validators/surfacei18n"
)

func TestValidateSurfaceI18nReportCommandAcceptsValidReport(t *testing.T) {
	dir := t.TempDir()
	report := commandI18nReport()
	reportPath := filepath.Join(dir, "surface-i18n-report.json")
	writeI18nJSON(t, reportPath, report)

	var stdout, stderr bytes.Buffer
	code := runValidateSurfaceI18nReport([]string{"--report", reportPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "surface i18n report OK") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestValidateSurfaceI18nReportCommandRejectsSilentFallback(t *testing.T) {
	dir := t.TempDir()
	report := commandI18nReport()
	report.Locales[1].ResourcePresent = false
	report.Locales[1].SilentFallback = true
	report.Locales[1].DiagnosticOnMissing = false
	reportPath := filepath.Join(dir, "surface-i18n-report.json")
	writeI18nJSON(t, reportPath, report)

	var stdout, stderr bytes.Buffer
	code := runValidateSurfaceI18nReport([]string{"--report", reportPath}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("expected nonzero exit, stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "fallback") {
		t.Fatalf("stderr = %q, want fallback rejection", stderr.String())
	}
}

func commandI18nReport() surfacei18n.Report {
	return surfacei18n.Report{
		Schema:       surfacei18n.SchemaV1,
		Status:       "pass",
		Level:        surfacei18n.LevelSurfaceI18nV1,
		Scope:        "surface-v1-scoped-linux-web-i18n",
		ReleaseScope: "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
		GitHead:      "0123456789abcdef0123456789abcdef01234567",
		SameCommit:   true,
		Policy: surfacei18n.LocalePolicy{
			Name:                       "surface-i18n-l10n-hooks-v1",
			DefaultLocale:              "en-US",
			FallbackPolicy:             "diagnostic-required",
			StringIDsRequired:          true,
			FormattingHooksRequired:    true,
			TranslationAssetPackaging:  true,
			MissingFallbackDiagnostics: true,
			SilentFallbackAllowed:      false,
		},
		Formatters: []surfacei18n.FormatterHook{
			{Name: "number", Strategy: "deterministic-decimal-v1", LocaleAware: true},
			{Name: "date", Strategy: "iso-date-locale-pattern-v1", LocaleAware: true},
			{Name: "plural", Strategy: "one-other-plural-v1", LocaleAware: true},
		},
		TextScope: surfacei18n.TextScope{
			UTF8Storage:           true,
			ShapingTier:           "tier1-latin-simple-plus-direction-metadata",
			LayoutDirection:       "ltr-rtl-layout-metadata-v1",
			LTRLayoutEvidence:     true,
			RTLLayoutEvidence:     true,
			FullBidiClaim:         false,
			BidiShapingEvidence:   "nonclaim: full bidi shaping stays outside P30",
			ComplexScriptNonclaim: true,
		},
		Package: surfacei18n.TranslationPackage{
			Manifest:                    surfacei18n.ArtifactRef{Path: "locales/surface-locales.manifest.json", SHA256: validI18nSHA("11"), Size: 256},
			TranslationAssetsPackaged:   true,
			LocaleResourceHashesPresent: true,
			SameCommit:                  true,
		},
		Locales: []surfacei18n.LocaleResource{
			{Locale: "en-US", Direction: "ltr", ResourcePath: "locales/en-US.json", SHA256: validI18nSHA("22"), Size: 320, StringIDs: requiredCommandIDs(), RequiredStringIDs: requiredCommandIDs(), ResourcePresent: true, DiagnosticOnMissing: true, SilentFallback: false, Packaged: true, Default: true},
			{Locale: "es-ES", Direction: "ltr", ResourcePath: "locales/es-ES.json", SHA256: validI18nSHA("33"), Size: 328, StringIDs: requiredCommandIDs(), RequiredStringIDs: requiredCommandIDs(), ResourcePresent: true, DiagnosticOnMissing: true, SilentFallback: false, Packaged: true},
			{Locale: "ar-EG", Direction: "rtl", ResourcePath: "locales/ar-EG.json", SHA256: validI18nSHA("44"), Size: 336, StringIDs: requiredCommandIDs(), RequiredStringIDs: requiredCommandIDs(), ResourcePresent: true, DiagnosticOnMissing: true, SilentFallback: false, Packaged: true},
		},
		Targets: []surfacei18n.TargetLocaleStatus{
			{Target: "linux-x64", Tier: "production", ProductionClaim: true, LocaleResourceSmoke: true, LayoutDirectionSmoke: true, Evidence: "linux localized render smoke"},
			{Target: "wasm32-web", Tier: "production", ProductionClaim: true, LocaleResourceSmoke: true, LayoutDirectionSmoke: true, Evidence: "web localized render smoke"},
			{Target: "windows-x64", Tier: "nonclaim", ProductionClaim: false, Evidence: "blocked until target-host evidence exists"},
			{Target: "macos-x64", Tier: "nonclaim", ProductionClaim: false, Evidence: "blocked until target-host evidence exists"},
		},
		Operations: []surfacei18n.Operation{
			{Name: "localized app build", Kind: "build", Ran: true, Pass: true},
			{Name: "localized app render", Kind: "render", Ran: true, Pass: true},
			{Name: "locale resource packaging", Kind: "package", Ran: true, Pass: true},
		},
		NegativeGuards: surfacei18n.NegativeGuards{
			FullBidiWithoutShapingRejected: true,
			MissingLocaleResourceRejected:  true,
			SilentFallbackRejected:         true,
			MissingStringIDRejected:        true,
			UnpackagedTranslationRejected:  true,
			UnsupportedHostLocaleRejected:  true,
			FullICUClaimRejected:           true,
		},
		NonClaims: []string{
			"No full bidi production shaping beyond scoped layout direction metadata.",
			"No full ICU or CLDR database claim.",
			"No full Unicode editor-grade localization semantics.",
			"No platform-native localization framework parity claim.",
		},
		Cases: []surfacei18n.CaseReport{
			{Name: "basic localized Surface app builds", Kind: "positive", Ran: true, Pass: true},
			{Name: "basic localized Surface app renders", Kind: "positive", Ran: true, Pass: true},
			{Name: "full bidi claim without shaping evidence rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "missing locale resource silent fallback rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "unpackaged translation asset rejected", Kind: "negative", Ran: true, Pass: true},
		},
	}
}

func requiredCommandIDs() []string {
	return []string{"app.title", "nav.settings", "action.save", "count.files.one", "count.files.other"}
}

func writeI18nJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func validI18nSHA(pair string) string {
	return "sha256:" + strings.Repeat(pair, 32)
}
