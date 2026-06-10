package surfacei18n

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

const (
	SchemaV1           = "tetra.surface.i18n-report.v1"
	LevelSurfaceI18nV1 = "surface-i18n-l10n-v1"
)

type Report struct {
	Schema         string               `json:"schema"`
	Status         string               `json:"status"`
	Level          string               `json:"level"`
	Scope          string               `json:"scope"`
	ReleaseScope   string               `json:"release_scope"`
	Producer       string               `json:"producer,omitempty"`
	GitHead        string               `json:"git_head"`
	SameCommit     bool                 `json:"same_commit"`
	Version        string               `json:"version,omitempty"`
	Policy         LocalePolicy         `json:"policy"`
	Formatters     []FormatterHook      `json:"formatters"`
	TextScope      TextScope            `json:"text_scope"`
	Package        TranslationPackage   `json:"package"`
	Locales        []LocaleResource     `json:"locales"`
	Targets        []TargetLocaleStatus `json:"targets"`
	Operations     []Operation          `json:"operations"`
	NegativeGuards NegativeGuards       `json:"negative_guards"`
	NonClaims      []string             `json:"nonclaims"`
	Cases          []CaseReport         `json:"cases"`
}

type LocalePolicy struct {
	Name                       string `json:"name"`
	DefaultLocale              string `json:"default_locale"`
	FallbackPolicy             string `json:"fallback_policy"`
	StringIDsRequired          bool   `json:"string_ids_required"`
	FormattingHooksRequired    bool   `json:"formatting_hooks_required"`
	TranslationAssetPackaging  bool   `json:"translation_asset_packaging"`
	MissingFallbackDiagnostics bool   `json:"missing_fallback_diagnostics"`
	SilentFallbackAllowed      bool   `json:"silent_fallback_allowed"`
	FullICUClaim               bool   `json:"full_icu_claim"`
	FullUnicodeEditorClaim     bool   `json:"full_unicode_editor_claim"`
}

type FormatterHook struct {
	Name         string `json:"name"`
	Strategy     string `json:"strategy"`
	LocaleAware  bool   `json:"locale_aware"`
	FullICUClaim bool   `json:"full_icu_claim"`
}

type TextScope struct {
	UTF8Storage           bool   `json:"utf8_storage"`
	ShapingTier           string `json:"shaping_tier"`
	LayoutDirection       string `json:"layout_direction"`
	LTRLayoutEvidence     bool   `json:"ltr_layout_evidence"`
	RTLLayoutEvidence     bool   `json:"rtl_layout_evidence"`
	FullBidiClaim         bool   `json:"full_bidi_claim"`
	BidiShapingEvidence   string `json:"bidi_shaping_evidence"`
	ComplexScriptNonclaim bool   `json:"complex_script_nonclaim"`
}

type TranslationPackage struct {
	Manifest                    ArtifactRef `json:"manifest"`
	TranslationAssetsPackaged   bool        `json:"translation_assets_packaged"`
	LocaleResourceHashesPresent bool        `json:"locale_resource_hashes_present"`
	SameCommit                  bool        `json:"same_commit"`
}

type ArtifactRef struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type LocaleResource struct {
	Locale              string   `json:"locale"`
	Direction           string   `json:"direction"`
	ResourcePath        string   `json:"resource_path"`
	SHA256              string   `json:"sha256"`
	Size                int64    `json:"size"`
	StringIDs           []string `json:"string_ids"`
	RequiredStringIDs   []string `json:"required_string_ids"`
	ResourcePresent     bool     `json:"resource_present"`
	DiagnosticOnMissing bool     `json:"diagnostic_on_missing"`
	SilentFallback      bool     `json:"silent_fallback"`
	Packaged            bool     `json:"packaged"`
	Default             bool     `json:"default"`
}

type TargetLocaleStatus struct {
	Target               string `json:"target"`
	Tier                 string `json:"tier"`
	ProductionClaim      bool   `json:"production_claim"`
	LocaleResourceSmoke  bool   `json:"locale_resource_smoke"`
	LayoutDirectionSmoke bool   `json:"layout_direction_smoke"`
	Evidence             string `json:"evidence"`
}

type Operation struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Ran  bool   `json:"ran"`
	Pass bool   `json:"pass"`
}

type NegativeGuards struct {
	FullBidiWithoutShapingRejected bool `json:"full_bidi_without_shaping_rejected"`
	MissingLocaleResourceRejected  bool `json:"missing_locale_resource_rejected"`
	SilentFallbackRejected         bool `json:"silent_fallback_rejected"`
	MissingStringIDRejected        bool `json:"missing_string_id_rejected"`
	UnpackagedTranslationRejected  bool `json:"unpackaged_translation_rejected"`
	UnsupportedHostLocaleRejected  bool `json:"unsupported_host_locale_rejected"`
	FullICUClaimRejected           bool `json:"full_icu_claim_rejected"`
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
	issues = append(issues, validatePolicy(report.Policy)...)
	issues = append(issues, validateFormatters(report.Formatters)...)
	issues = append(issues, validateTextScope(report.TextScope)...)
	issues = append(issues, validatePackage(report.Package)...)
	issues = append(issues, validateLocales(report.Policy, report.Locales)...)
	issues = append(issues, validateTargets(report.Targets)...)
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
	if report.Level != LevelSurfaceI18nV1 {
		issues = append(issues, fmt.Sprintf("level is %q, want %q", report.Level, LevelSurfaceI18nV1))
	}
	if report.Scope != "surface-v1-scoped-linux-web-i18n" {
		issues = append(issues, fmt.Sprintf("scope is %q, want surface-v1-scoped-linux-web-i18n", report.Scope))
	}
	if report.ReleaseScope != "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI" {
		issues = append(issues, fmt.Sprintf("release_scope is %q, want PROD_STABLE_SCOPED_LINUX_WEB_APP_UI", report.ReleaseScope))
	}
	if !validGitHead(report.GitHead) {
		issues = append(issues, "git_head must be a 40-hex same-commit revision")
	}
	if !report.SameCommit {
		issues = append(issues, "same_commit localization evidence is required")
	}
	return issues
}

func validatePolicy(policy LocalePolicy) []string {
	var issues []string
	if policy.Name != "surface-i18n-l10n-hooks-v1" {
		issues = append(issues, fmt.Sprintf("locale policy is %q, want surface-i18n-l10n-hooks-v1", policy.Name))
	}
	if policy.DefaultLocale != "en-US" {
		issues = append(issues, fmt.Sprintf("default locale is %q, want en-US", policy.DefaultLocale))
	}
	if policy.FallbackPolicy != "diagnostic-required" {
		issues = append(issues, fmt.Sprintf("fallback policy is %q, want diagnostic-required", policy.FallbackPolicy))
	}
	if !policy.StringIDsRequired {
		issues = append(issues, "string IDs are required")
	}
	if !policy.FormattingHooksRequired {
		issues = append(issues, "localization formatting hooks are required")
	}
	if !policy.TranslationAssetPackaging {
		issues = append(issues, "translation asset packaging policy is required")
	}
	if !policy.MissingFallbackDiagnostics {
		issues = append(issues, "missing locale fallback diagnostics are required")
	}
	if policy.SilentFallbackAllowed {
		issues = append(issues, "silent fallback is rejected")
	}
	if policy.FullICUClaim {
		issues = append(issues, "full ICU/CLDR claim is rejected")
	}
	if policy.FullUnicodeEditorClaim {
		issues = append(issues, "full Unicode editor localization claim is rejected")
	}
	return issues
}

func validateFormatters(formatters []FormatterHook) []string {
	required := map[string]bool{"number": false, "date": false, "plural": false}
	var issues []string
	for i, formatter := range formatters {
		name := strings.TrimSpace(formatter.Name)
		if _, ok := required[name]; ok {
			required[name] = true
		}
		if name == "" || strings.TrimSpace(formatter.Strategy) == "" {
			issues = append(issues, fmt.Sprintf("formatter[%d] requires name and strategy", i))
		}
		if !formatter.LocaleAware {
			issues = append(issues, fmt.Sprintf("formatter %q must be locale aware", formatter.Name))
		}
		if formatter.FullICUClaim {
			issues = append(issues, fmt.Sprintf("formatter %q makes a full ICU claim", formatter.Name))
		}
	}
	for name, seen := range required {
		if !seen {
			issues = append(issues, fmt.Sprintf("missing %s localization formatter hook", name))
		}
	}
	return issues
}

func validateTextScope(scope TextScope) []string {
	var issues []string
	if !scope.UTF8Storage {
		issues = append(issues, "UTF-8 storage evidence is required")
	}
	if !strings.Contains(scope.ShapingTier, "tier1") {
		issues = append(issues, fmt.Sprintf("shaping tier is %q, want tier1 scoped text support", scope.ShapingTier))
	}
	if scope.LayoutDirection != "ltr-rtl-layout-metadata-v1" {
		issues = append(issues, fmt.Sprintf("layout direction evidence is %q, want ltr-rtl-layout-metadata-v1", scope.LayoutDirection))
	}
	if !scope.LTRLayoutEvidence || !scope.RTLLayoutEvidence {
		issues = append(issues, "LTR and RTL layout direction evidence is required")
	}
	if scope.FullBidiClaim && strings.TrimSpace(scope.BidiShapingEvidence) == "" {
		issues = append(issues, "full bidi claim requires shaping evidence and is rejected without it")
	}
	if !scope.ComplexScriptNonclaim {
		issues = append(issues, "complex script shaping nonclaim is required")
	}
	return issues
}

func validatePackage(pkg TranslationPackage) []string {
	var issues []string
	issues = append(issues, validateArtifactRef("locale manifest", pkg.Manifest)...)
	if !pkg.TranslationAssetsPackaged {
		issues = append(issues, "translation assets must be packaged")
	}
	if !pkg.LocaleResourceHashesPresent {
		issues = append(issues, "locale resource hashes are required")
	}
	if !pkg.SameCommit {
		issues = append(issues, "translation package must be same-commit evidence")
	}
	return issues
}

func validateLocales(policy LocalePolicy, locales []LocaleResource) []string {
	if len(locales) < 2 {
		return []string{"at least default and non-English locale resources are required"}
	}
	var issues []string
	defaultCount := 0
	nonDefaultCount := 0
	rtlCount := 0
	for i, locale := range locales {
		prefix := fmt.Sprintf("locale[%d] %s", i, locale.Locale)
		if !validLocaleTag(locale.Locale) {
			issues = append(issues, fmt.Sprintf("%s has invalid locale tag", prefix))
		}
		if locale.Default {
			defaultCount++
			if locale.Locale != policy.DefaultLocale {
				issues = append(issues, fmt.Sprintf("%s default does not match policy default locale %s", prefix, policy.DefaultLocale))
			}
		} else {
			nonDefaultCount++
		}
		switch locale.Direction {
		case "ltr":
		case "rtl":
			rtlCount++
		default:
			issues = append(issues, fmt.Sprintf("%s direction is %q, want ltr or rtl", prefix, locale.Direction))
		}
		if err := validateSafeRelPath(locale.ResourcePath); err != nil {
			issues = append(issues, fmt.Sprintf("%s resource path: %v", prefix, err))
		}
		if !validSHA256(locale.SHA256) || locale.Size <= 0 {
			issues = append(issues, fmt.Sprintf("%s requires sha256 and nonzero size", prefix))
		}
		if !locale.ResourcePresent {
			issues = append(issues, fmt.Sprintf("%s locale resource is missing", prefix))
		}
		if locale.SilentFallback {
			issues = append(issues, fmt.Sprintf("%s silent fallback is rejected", prefix))
		}
		if !locale.DiagnosticOnMissing {
			issues = append(issues, fmt.Sprintf("%s requires missing locale diagnostic", prefix))
		}
		if !locale.Packaged {
			issues = append(issues, fmt.Sprintf("%s translation resource must be packaged", prefix))
		}
		issues = append(issues, validateStringIDs(prefix, locale.StringIDs, locale.RequiredStringIDs)...)
	}
	if defaultCount != 1 {
		issues = append(issues, fmt.Sprintf("exactly one default locale resource is required, got %d", defaultCount))
	}
	if nonDefaultCount == 0 {
		issues = append(issues, "at least one non-English locale resource is required")
	}
	if rtlCount == 0 {
		issues = append(issues, "at least one RTL layout-direction locale resource is required for scoped direction evidence")
	}
	return issues
}

func validateStringIDs(prefix string, ids []string, required []string) []string {
	var issues []string
	if len(required) < 4 {
		issues = append(issues, fmt.Sprintf("%s requires a stable string ID set", prefix))
	}
	seen := map[string]bool{}
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" || !strings.Contains(trimmed, ".") {
			issues = append(issues, fmt.Sprintf("%s invalid string ID %q", prefix, id))
			continue
		}
		seen[trimmed] = true
	}
	for _, id := range required {
		if !seen[id] {
			issues = append(issues, fmt.Sprintf("%s missing string ID %s", prefix, id))
		}
	}
	return issues
}

func validateTargets(targets []TargetLocaleStatus) []string {
	if len(targets) == 0 {
		return []string{"target localization status evidence is required"}
	}
	var issues []string
	required := map[string]bool{"linux-x64": false, "wasm32-web": false}
	for _, target := range targets {
		if target.Target == "linux-x64" || target.Target == "wasm32-web" {
			required[target.Target] = target.ProductionClaim && target.LocaleResourceSmoke && target.LayoutDirectionSmoke
		}
		if target.ProductionClaim {
			if target.Target != "linux-x64" && target.Target != "wasm32-web" {
				issues = append(issues, fmt.Sprintf("target %s localization production claim is unsupported", target.Target))
			}
			if !target.LocaleResourceSmoke || !target.LayoutDirectionSmoke {
				issues = append(issues, fmt.Sprintf("target %s localization claim requires locale resource and layout direction smoke", target.Target))
			}
		}
		if strings.TrimSpace(target.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("target %s localization evidence is required", target.Target))
		}
	}
	for target, ok := range required {
		if !ok {
			issues = append(issues, fmt.Sprintf("target %s requires scoped production localization smoke", target))
		}
	}
	return issues
}

func validateOperations(operations []Operation) []string {
	if len(operations) < 3 {
		return []string{"localized build, render, and package operations are required"}
	}
	var issues []string
	for i, operation := range operations {
		if strings.TrimSpace(operation.Name) == "" || strings.TrimSpace(operation.Kind) == "" {
			issues = append(issues, fmt.Sprintf("operation[%d] requires name and kind", i))
		}
		if !operation.Ran || !operation.Pass {
			issues = append(issues, fmt.Sprintf("operation %q must run and pass", operation.Name))
		}
	}
	return issues
}

func validateNegativeGuards(guards NegativeGuards) []string {
	required := map[string]bool{
		"full bidi claim without shaping evidence rejected": guards.FullBidiWithoutShapingRejected,
		"missing locale resource rejected":                  guards.MissingLocaleResourceRejected,
		"silent fallback rejected":                          guards.SilentFallbackRejected,
		"missing string ID rejected":                        guards.MissingStringIDRejected,
		"unpackaged translation asset rejected":             guards.UnpackagedTranslationRejected,
		"unsupported host localization rejected":            guards.UnsupportedHostLocaleRejected,
		"full ICU claim rejected":                           guards.FullICUClaimRejected,
	}
	var issues []string
	for label, ok := range required {
		if !ok {
			issues = append(issues, label)
		}
	}
	return issues
}

func validateNonClaims(nonclaims []string) []string {
	required := []string{"full bidi", "full ICU", "full Unicode", "platform-native localization"}
	haystack := strings.Join(nonclaims, "\n")
	var issues []string
	for _, want := range required {
		if !strings.Contains(haystack, want) {
			issues = append(issues, fmt.Sprintf("missing nonclaim containing %q", want))
		}
	}
	return issues
}

func validateCases(cases []CaseReport) []string {
	required := map[string]bool{
		"basic localized Surface app builds":                false,
		"basic localized Surface app renders":               false,
		"full bidi claim without shaping evidence rejected": false,
		"missing locale resource silent fallback rejected":  false,
		"unpackaged translation asset rejected":             false,
	}
	var issues []string
	for _, c := range cases {
		if _, ok := required[c.Name]; ok {
			required[c.Name] = c.Ran && c.Pass
		}
		if strings.TrimSpace(c.Name) == "" || strings.TrimSpace(c.Kind) == "" {
			issues = append(issues, "case report requires name and kind")
		}
		if !c.Ran || !c.Pass {
			issues = append(issues, fmt.Sprintf("case %q must run and pass", c.Name))
		}
	}
	for name, ok := range required {
		if !ok {
			issues = append(issues, fmt.Sprintf("missing or failed case %q", name))
		}
	}
	return issues
}

func validateArtifactRef(label string, ref ArtifactRef) []string {
	var issues []string
	if err := validateSafeRelPath(ref.Path); err != nil {
		issues = append(issues, fmt.Sprintf("%s path: %v", label, err))
	}
	if !validSHA256(ref.SHA256) {
		issues = append(issues, fmt.Sprintf("%s sha256 is required", label))
	}
	if ref.Size <= 0 {
		issues = append(issues, fmt.Sprintf("%s size must be positive", label))
	}
	return issues
}

func validateSafeRelPath(path string) error {
	if path == "" {
		return fmt.Errorf("path is required")
	}
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute path is forbidden")
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") || clean == ".." {
		return fmt.Errorf("path must stay inside report/package root")
	}
	return nil
}

func validGitHead(value string) bool {
	if len(value) != 40 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validSHA256(value string) bool {
	if !strings.HasPrefix(value, "sha256:") {
		return false
	}
	hexPart := strings.TrimPrefix(value, "sha256:")
	if len(hexPart) != 64 {
		return false
	}
	_, err := hex.DecodeString(hexPart)
	return err == nil
}

func validLocaleTag(value string) bool {
	parts := strings.Split(value, "-")
	if len(parts) != 2 {
		return false
	}
	if len(parts[0]) != 2 || len(parts[1]) != 2 {
		return false
	}
	for _, r := range parts[0] {
		if !unicode.IsLower(r) {
			return false
		}
	}
	for _, r := range parts[1] {
		if !unicode.IsUpper(r) {
			return false
		}
	}
	return true
}
