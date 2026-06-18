package surfacei18n

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateReportAcceptsScopedLocalizationEvidence(t *testing.T) {
	if err := ValidateReport(mustReportJSON(t, validI18nReport())); err != nil {
		t.Fatalf("ValidateReport returned error: %v", err)
	}
}

func TestValidateReportRejectsFullBidiClaimWithoutShapingEvidence(t *testing.T) {
	report := validI18nReport()
	report.TextScope.FullBidiClaim = true
	report.TextScope.BidiShapingEvidence = ""

	err := ValidateReport(mustReportJSON(t, report))
	if err == nil {
		t.Fatal("expected full bidi claim without shaping evidence to be rejected")
	}
	if !strings.Contains(err.Error(), "bidi") || !strings.Contains(err.Error(), "shaping") {
		t.Fatalf("error = %q, want bidi shaping rejection", err.Error())
	}
}

func TestValidateReportRejectsMissingLocaleResourceSilentFallback(t *testing.T) {
	report := validI18nReport()
	report.Locales[1].ResourcePresent = false
	report.Locales[1].SilentFallback = true
	report.Locales[1].DiagnosticOnMissing = false

	err := ValidateReport(mustReportJSON(t, report))
	if err == nil {
		t.Fatal("expected missing locale resource with silent fallback to be rejected")
	}
	if !strings.Contains(err.Error(), "locale") || !strings.Contains(err.Error(), "fallback") {
		t.Fatalf("error = %q, want locale fallback rejection", err.Error())
	}
}

func TestValidateReportRejectsUnpackagedTranslationAsset(t *testing.T) {
	report := validI18nReport()
	report.Locales[2].Packaged = false
	report.Package.TranslationAssetsPackaged = false

	err := ValidateReport(mustReportJSON(t, report))
	if err == nil {
		t.Fatal("expected unpackaged translation assets to be rejected")
	}
	if !strings.Contains(err.Error(), "packaged") {
		t.Fatalf("error = %q, want packaged asset rejection", err.Error())
	}
}

func TestValidateReportRejectsUnsupportedHostLocaleClaim(t *testing.T) {
	report := validI18nReport()
	report.Targets[1].ProductionClaim = true
	report.Targets[1].LocaleResourceSmoke = false

	err := ValidateReport(mustReportJSON(t, report))
	if err == nil {
		t.Fatal("expected unsupported host localization claim to be rejected")
	}
	if !strings.Contains(err.Error(), "target") || !strings.Contains(err.Error(), "localization") {
		t.Fatalf("error = %q, want target localization rejection", err.Error())
	}
}

func validI18nReport() Report {
	return Report{
		Schema:       SchemaV1,
		Status:       "pass",
		Level:        LevelSurfaceI18nV1,
		Scope:        "surface-v1-scoped-linux-web-i18n",
		ReleaseScope: "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI",
		GitHead:      "0123456789abcdef0123456789abcdef01234567",
		SameCommit:   true,
		Policy: LocalePolicy{
			Name:                       "surface-i18n-l10n-hooks-v1",
			DefaultLocale:              "en-US",
			FallbackPolicy:             "diagnostic-required",
			StringIDsRequired:          true,
			FormattingHooksRequired:    true,
			TranslationAssetPackaging:  true,
			MissingFallbackDiagnostics: true,
			SilentFallbackAllowed:      false,
			FullICUClaim:               false,
			FullUnicodeEditorClaim:     false,
		},
		Formatters: []FormatterHook{
			{Name: "number", Strategy: "deterministic-decimal-v1", LocaleAware: true, FullICUClaim: false},
			{Name: "date", Strategy: "iso-date-locale-pattern-v1", LocaleAware: true, FullICUClaim: false},
			{Name: "plural", Strategy: "one-other-plural-v1", LocaleAware: true, FullICUClaim: false},
		},
		TextScope: TextScope{
			UTF8Storage:           true,
			ShapingTier:           "tier1-latin-simple-plus-direction-metadata",
			LayoutDirection:       "ltr-rtl-layout-metadata-v1",
			LTRLayoutEvidence:     true,
			RTLLayoutEvidence:     true,
			FullBidiClaim:         false,
			BidiShapingEvidence:   "nonclaim: full bidi shaping stays outside P30",
			ComplexScriptNonclaim: true,
		},
		Package: TranslationPackage{
			Manifest:                    ArtifactRef{Path: "locales/surface-locales.manifest.json", SHA256: validSHA("11"), Size: 256},
			TranslationAssetsPackaged:   true,
			LocaleResourceHashesPresent: true,
			SameCommit:                  true,
		},
		Locales: []LocaleResource{
			{
				Locale:              "en-US",
				Direction:           "ltr",
				ResourcePath:        "locales/en-US.json",
				SHA256:              validSHA("22"),
				Size:                320,
				StringIDs:           []string{"app.title", "nav.settings", "action.save", "count.files.one", "count.files.other"},
				RequiredStringIDs:   []string{"app.title", "nav.settings", "action.save", "count.files.one", "count.files.other"},
				ResourcePresent:     true,
				DiagnosticOnMissing: true,
				SilentFallback:      false,
				Packaged:            true,
				Default:             true,
			},
			{
				Locale:              "es-ES",
				Direction:           "ltr",
				ResourcePath:        "locales/es-ES.json",
				SHA256:              validSHA("33"),
				Size:                328,
				StringIDs:           []string{"app.title", "nav.settings", "action.save", "count.files.one", "count.files.other"},
				RequiredStringIDs:   []string{"app.title", "nav.settings", "action.save", "count.files.one", "count.files.other"},
				ResourcePresent:     true,
				DiagnosticOnMissing: true,
				SilentFallback:      false,
				Packaged:            true,
			},
			{
				Locale:              "ar-EG",
				Direction:           "rtl",
				ResourcePath:        "locales/ar-EG.json",
				SHA256:              validSHA("44"),
				Size:                336,
				StringIDs:           []string{"app.title", "nav.settings", "action.save", "count.files.one", "count.files.other"},
				RequiredStringIDs:   []string{"app.title", "nav.settings", "action.save", "count.files.one", "count.files.other"},
				ResourcePresent:     true,
				DiagnosticOnMissing: true,
				SilentFallback:      false,
				Packaged:            true,
			},
		},
		Targets: []TargetLocaleStatus{
			{Target: "linux-x64", Tier: "production", ProductionClaim: true, LocaleResourceSmoke: true, LayoutDirectionSmoke: true, Evidence: "linux-x64 localized surface app render smoke"},
			{Target: "wasm32-web", Tier: "production", ProductionClaim: true, LocaleResourceSmoke: true, LayoutDirectionSmoke: true, Evidence: "wasm32-web browser-canvas localized render smoke"},
			{Target: "windows-x64", Tier: "nonclaim", ProductionClaim: false, LocaleResourceSmoke: false, LayoutDirectionSmoke: false, Evidence: "blocked until target-host locale packaging evidence exists"},
			{Target: "macos-x64", Tier: "nonclaim", ProductionClaim: false, LocaleResourceSmoke: false, LayoutDirectionSmoke: false, Evidence: "blocked until target-host locale packaging evidence exists"},
		},
		Operations: []Operation{
			{Name: "localized app build", Kind: "build", Ran: true, Pass: true},
			{Name: "localized app render", Kind: "render", Ran: true, Pass: true},
			{Name: "locale resource packaging", Kind: "package", Ran: true, Pass: true},
			{Name: "missing locale diagnostics", Kind: "diagnostic", Ran: true, Pass: true},
		},
		NegativeGuards: NegativeGuards{
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
		Cases: []CaseReport{
			{Name: "basic localized Surface app builds", Kind: "positive", Ran: true, Pass: true},
			{Name: "basic localized Surface app renders", Kind: "positive", Ran: true, Pass: true},
			{Name: "full bidi claim without shaping evidence rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "missing locale resource silent fallback rejected", Kind: "negative", Ran: true, Pass: true},
			{Name: "unpackaged translation asset rejected", Kind: "negative", Ran: true, Pass: true},
		},
	}
}

func mustReportJSON(t *testing.T, report Report) []byte {
	t.Helper()
	raw, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func validSHA(pair string) string {
	return "sha256:" + strings.Repeat(pair, 32)
}
