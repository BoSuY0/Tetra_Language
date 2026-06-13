package surface

import (
	"errors"
	"fmt"
	"strings"
)

const I18nSchemaV1 = "tetra.surface.i18n.v1"

type SurfaceI18nReportV1 struct {
	Schema          string                     `json:"schema"`
	Model           string                     `json:"model"`
	ReleaseScope    string                     `json:"release_scope"`
	Producer        string                     `json:"producer"`
	Source          string                     `json:"source"`
	ReferenceApp    string                     `json:"reference_app"`
	Target          string                     `json:"target"`
	StringTables    []SurfaceI18nStringTable   `json:"string_tables"`
	LocaleSelection SurfaceI18nLocaleSelection `json:"locale_selection"`
	Lookups         []SurfaceI18nLookup        `json:"lookups"`
	FormatHooks     []SurfaceI18nFormatHook    `json:"format_hooks"`
	TextDirection   SurfaceI18nTextDirection   `json:"text_direction"`
	LocalizedForm   SurfaceI18nLocalizedForm   `json:"localized_form"`
	NegativeGuards  SurfaceI18nNegativeGuards  `json:"negative_guards"`
	Pass            bool                       `json:"pass"`
}

type SurfaceI18nStringTable struct {
	Locale     string `json:"locale"`
	EntryCount int    `json:"entry_count"`
	Checksum   string `json:"checksum"`
	Primary    bool   `json:"primary"`
	Fallback   bool   `json:"fallback"`
	Pass       bool   `json:"pass"`
}

type SurfaceI18nLocaleSelection struct {
	RequestedLocale           string `json:"requested_locale"`
	SelectedLocale            string `json:"selected_locale"`
	FallbackLocale            string `json:"fallback_locale"`
	FallbackUsed              bool   `json:"fallback_used"`
	UnsupportedLocaleRejected bool   `json:"unsupported_locale_rejected"`
	Pass                      bool   `json:"pass"`
}

type SurfaceI18nLookup struct {
	Key            string `json:"key"`
	Locale         string `json:"locale"`
	ResolvedLocale string `json:"resolved_locale"`
	Source         string `json:"source"`
	MissingKey     bool   `json:"missing_key"`
	FallbackUsed   bool   `json:"fallback_used"`
	DiagnosticCode int    `json:"diagnostic_code"`
	Pass           bool   `json:"pass"`
}

type SurfaceI18nFormatHook struct {
	Kind          string `json:"kind"`
	Locale        string `json:"locale"`
	Input         string `json:"input"`
	Output        string `json:"output"`
	Deterministic bool   `json:"deterministic"`
	ICUClaim      bool   `json:"icu_claim"`
	Pass          bool   `json:"pass"`
}

type SurfaceI18nTextDirection struct {
	DefaultDirection  string `json:"default_direction"`
	RTLPlaceholder    bool   `json:"rtl_placeholder"`
	FullBidiSupported bool   `json:"full_bidi_supported"`
	FullBidiClaim     bool   `json:"full_bidi_claim"`
	ShapingProof      bool   `json:"shaping_proof"`
	Nonclaim          string `json:"nonclaim"`
	Pass              bool   `json:"pass"`
}

type SurfaceI18nLocalizedForm struct {
	Shape                string   `json:"shape"`
	Source               string   `json:"source"`
	Imports              []string `json:"imports"`
	Compiles             bool     `json:"compiles"`
	Runs                 bool     `json:"runs"`
	ExitCode             int      `json:"exit_code"`
	LocalizedStrings     bool     `json:"localized_strings"`
	FallbackEvidence     bool     `json:"fallback_evidence"`
	MissingKeyDiagnostic bool     `json:"missing_key_diagnostic"`
	FormatHookEvidence   bool     `json:"format_hook_evidence"`
	ResolvesToBlock      bool     `json:"resolves_to_block"`
	Pass                 bool     `json:"pass"`
}

type SurfaceI18nNegativeGuards struct {
	NoFullICUClaim             bool `json:"no_full_icu_claim"`
	NoFullBidiClaim            bool `json:"no_full_bidi_claim"`
	NoRTLProductionClaim       bool `json:"no_rtl_production_claim"`
	NoMissingKeySilentFallback bool `json:"no_missing_key_silent_fallback"`
	NoDocsOnlyI18nClaim        bool `json:"no_docs_only_i18n_claim"`
	NoReactIntlRuntime         bool `json:"no_react_intl_runtime"`
	NoPlatformLocaleDependency bool `json:"no_platform_locale_dependency"`
}

func ValidateI18nReport(raw []byte) error {
	schema, err := decodeSchema(raw)
	if err != nil {
		return err
	}
	if schema != I18nSchemaV1 {
		return fmt.Errorf("schema is %q, want %q", schema, I18nSchemaV1)
	}
	var report SurfaceI18nReportV1
	if err := decodeStrict(raw, &report); err != nil {
		return err
	}
	issues := validateSurfaceI18nReport(report)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func validateSurfaceI18nReport(report SurfaceI18nReportV1) []string {
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "schema", got: report.Schema, want: I18nSchemaV1},
		{field: "model", got: report.Model, want: "surface-i18n-v1"},
		{field: "release_scope", got: report.ReleaseScope, want: ReleaseScopeSurfaceV1LinuxWeb},
		{field: "producer", got: report.Producer, want: "scripts/release/surface/surface-i18n-smoke.sh"},
		{field: "reference_app", got: report.ReferenceApp, want: "localized-form"},
		{field: "target", got: report.Target, want: "linux-x64"},
	} {
		if check.got != check.want {
			issues = append(issues, fmt.Sprintf("%s is %q, want %q", check.field, check.got, check.want))
		}
	}
	if !safeRelativeSourcePath(report.Source) {
		issues = append(issues, "source must be a safe Tetra source path")
	}
	if !surfacePackageSourceMatchesReferenceApp(report.ReferenceApp, report.Source) {
		issues = append(issues, fmt.Sprintf("reference_app %q does not match source %q", report.ReferenceApp, report.Source))
	}
	issues = append(issues, validateSurfaceI18nStringTables(report.StringTables)...)
	issues = append(issues, validateSurfaceI18nLocaleSelection(report.LocaleSelection)...)
	issues = append(issues, validateSurfaceI18nLookups(report.Lookups)...)
	issues = append(issues, validateSurfaceI18nFormatHooks(report.FormatHooks)...)
	issues = append(issues, validateSurfaceI18nTextDirection(report.TextDirection)...)
	issues = append(issues, validateSurfaceI18nLocalizedForm(report.LocalizedForm)...)
	issues = append(issues, validateSurfaceI18nNegativeGuards(report.NegativeGuards)...)
	if !report.Pass {
		issues = append(issues, "pass must be true")
	}
	return issues
}

func validateSurfaceI18nStringTables(tables []SurfaceI18nStringTable) []string {
	if len(tables) < 2 {
		return []string{"string_tables require en-US and uk-UA"}
	}
	var issues []string
	seen := map[string]SurfaceI18nStringTable{}
	for _, table := range tables {
		locale := strings.TrimSpace(table.Locale)
		seen[locale] = table
		prefix := "string table " + locale
		if locale == "" {
			issues = append(issues, "string table locale is required")
		}
		if table.EntryCount <= 0 {
			issues = append(issues, prefix+" entry_count must be positive")
		}
		if !validChecksumLike(table.Checksum) {
			issues = append(issues, prefix+" checksum must be sha256 evidence")
		}
		if !table.Pass {
			issues = append(issues, prefix+" pass must be true")
		}
	}
	en, ok := seen["en-US"]
	if !ok {
		issues = append(issues, "string_tables missing en-US")
	} else if !en.Primary || en.Fallback || en.EntryCount < 5 {
		issues = append(issues, "string table en-US must be primary with at least five entries and not fallback")
	}
	uk, ok := seen["uk-UA"]
	if !ok {
		issues = append(issues, "string_tables missing uk-UA")
	} else if uk.Primary || !uk.Fallback || uk.EntryCount < 4 {
		issues = append(issues, "string table uk-UA must be fallback-aware with at least four entries")
	}
	return issues
}

func validateSurfaceI18nLocaleSelection(selection SurfaceI18nLocaleSelection) []string {
	var issues []string
	for _, check := range []struct {
		field string
		got   string
		want  string
	}{
		{field: "requested_locale", got: selection.RequestedLocale, want: "uk-UA"},
		{field: "selected_locale", got: selection.SelectedLocale, want: "uk-UA"},
		{field: "fallback_locale", got: selection.FallbackLocale, want: "en-US"},
	} {
		if check.got != check.want {
			issues = append(issues, fmt.Sprintf("locale_selection %s is %q, want %q", check.field, check.got, check.want))
		}
	}
	if !selection.FallbackUsed {
		issues = append(issues, "locale_selection fallback_used must be true")
	}
	if !selection.UnsupportedLocaleRejected {
		issues = append(issues, "locale_selection unsupported_locale_rejected must be true")
	}
	if !selection.Pass {
		issues = append(issues, "locale_selection pass must be true")
	}
	return issues
}

func validateSurfaceI18nLookups(lookups []SurfaceI18nLookup) []string {
	if len(lookups) < 3 {
		return []string{"lookups require primary, fallback, and missing_key evidence"}
	}
	var issues []string
	var primary, fallback, missing bool
	for _, lookup := range lookups {
		key := strings.TrimSpace(lookup.Key)
		prefix := "lookup " + key
		if key == "" || strings.TrimSpace(lookup.Locale) == "" || strings.TrimSpace(lookup.ResolvedLocale) == "" {
			issues = append(issues, prefix+" key, locale, and resolved_locale are required")
		}
		if !lookup.Pass {
			issues = append(issues, prefix+" pass must be true")
		}
		switch {
		case lookup.Source == "primary" && !lookup.MissingKey && !lookup.FallbackUsed && lookup.DiagnosticCode == 0:
			primary = true
		case lookup.Source == "fallback" && !lookup.MissingKey && lookup.FallbackUsed && lookup.ResolvedLocale == "en-US" && lookup.DiagnosticCode == 0:
			fallback = true
		case lookup.Source == "missing" && lookup.MissingKey && lookup.FallbackUsed && lookup.DiagnosticCode == 2001:
			missing = true
		}
	}
	if !primary {
		issues = append(issues, "lookups missing primary locale resolution")
	}
	if !fallback {
		issues = append(issues, "lookups missing fallback locale resolution")
	}
	if !missing {
		issues = append(issues, "lookups missing missing_key diagnostic evidence")
	}
	return issues
}

func validateSurfaceI18nFormatHooks(hooks []SurfaceI18nFormatHook) []string {
	if len(hooks) < 2 {
		return []string{"format_hooks require date and number evidence"}
	}
	var issues []string
	seen := map[string]bool{}
	for _, hook := range hooks {
		kind := strings.TrimSpace(hook.Kind)
		seen[kind] = true
		if kind == "" || strings.TrimSpace(hook.Locale) == "" || strings.TrimSpace(hook.Input) == "" || strings.TrimSpace(hook.Output) == "" {
			issues = append(issues, "format_hooks kind, locale, input, and output are required")
		}
		if !hook.Deterministic {
			issues = append(issues, fmt.Sprintf("format_hook %s deterministic must be true", kind))
		}
		if hook.ICUClaim {
			issues = append(issues, fmt.Sprintf("format_hook %s must not claim full ICU", kind))
		}
		if !hook.Pass {
			issues = append(issues, fmt.Sprintf("format_hook %s pass must be true", kind))
		}
	}
	for _, kind := range []string{"date", "number"} {
		if !seen[kind] {
			issues = append(issues, "format_hooks missing "+kind)
		}
	}
	return issues
}

func validateSurfaceI18nTextDirection(direction SurfaceI18nTextDirection) []string {
	var issues []string
	if direction.DefaultDirection != "ltr" {
		issues = append(issues, fmt.Sprintf("text_direction default_direction is %q, want ltr", direction.DefaultDirection))
	}
	if !direction.RTLPlaceholder {
		issues = append(issues, "text_direction rtl_placeholder must be true")
	}
	if direction.FullBidiSupported || direction.FullBidiClaim || direction.ShapingProof {
		issues = append(issues, "text_direction must not claim full bidi shaping without proof")
	}
	if direction.Nonclaim != "rtl-placeholder-without-full-bidi-shaping-v1" {
		issues = append(issues, fmt.Sprintf("text_direction nonclaim is %q, want rtl-placeholder-without-full-bidi-shaping-v1", direction.Nonclaim))
	}
	if !direction.Pass {
		issues = append(issues, "text_direction pass must be true")
	}
	return issues
}

func validateSurfaceI18nLocalizedForm(form SurfaceI18nLocalizedForm) []string {
	var issues []string
	if form.Shape != "localized-form" {
		issues = append(issues, fmt.Sprintf("localized_form shape is %q, want localized-form", form.Shape))
	}
	if !safeRelativeSourcePath(form.Source) || normalizeEvidencePath(form.Source) != "examples/surface_reference_localized_form.tetra" {
		issues = append(issues, "localized_form source must be examples/surface_reference_localized_form.tetra")
	}
	for _, required := range []string{"lib.core.surface", "lib.core.block", "lib.core.morph", "lib.core.i18n"} {
		if !templateSmokeContainsString(form.Imports, required) {
			issues = append(issues, "localized_form imports missing "+required)
		}
	}
	for _, check := range []struct {
		field string
		ok    bool
	}{
		{field: "compiles", ok: form.Compiles},
		{field: "runs", ok: form.Runs},
		{field: "localized_strings", ok: form.LocalizedStrings},
		{field: "fallback_evidence", ok: form.FallbackEvidence},
		{field: "missing_key_diagnostic", ok: form.MissingKeyDiagnostic},
		{field: "format_hook_evidence", ok: form.FormatHookEvidence},
		{field: "resolves_to_block", ok: form.ResolvesToBlock},
		{field: "pass", ok: form.Pass},
	} {
		if !check.ok {
			issues = append(issues, "localized_form "+check.field+" must be true")
		}
	}
	if form.ExitCode != 0 {
		issues = append(issues, fmt.Sprintf("localized_form exit_code is %d, want 0", form.ExitCode))
	}
	return issues
}

func validateSurfaceI18nNegativeGuards(guards SurfaceI18nNegativeGuards) []string {
	var issues []string
	for _, check := range []struct {
		field string
		ok    bool
	}{
		{field: "no_full_icu_claim", ok: guards.NoFullICUClaim},
		{field: "no_full_bidi_claim", ok: guards.NoFullBidiClaim},
		{field: "no_rtl_production_claim", ok: guards.NoRTLProductionClaim},
		{field: "no_missing_key_silent_fallback", ok: guards.NoMissingKeySilentFallback},
		{field: "no_docs_only_i18n_claim", ok: guards.NoDocsOnlyI18nClaim},
		{field: "no_react_intl_runtime", ok: guards.NoReactIntlRuntime},
		{field: "no_platform_locale_dependency", ok: guards.NoPlatformLocaleDependency},
	} {
		if !check.ok {
			issues = append(issues, "negative_guards."+check.field+" must be true")
		}
	}
	return issues
}
