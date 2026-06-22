package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

const actorCapabilityManifestSchemaV1 = "tetra.actor.capability_manifest.v1"

type actorCapabilityManifest struct {
	Schema                 string                       `json:"schema"`
	Profile                string                       `json:"profile"`
	ReleaseTarget          releaseTargetContract        `json:"release_target,omitempty"`
	FinalVerdictVocabulary []string                     `json:"final_verdict_vocabulary,omitempty"`
	RequiredFinalVerdict   string                       `json:"required_final_verdict,omitempty"`
	AllowedFinalClaim      string                       `json:"allowed_final_claim,omitempty"`
	RequiredV1Capabilities []string                     `json:"required_v1_capabilities,omitempty"`
	RequiredFinalReports   []string                     `json:"required_final_reports,omitempty"`
	ForbiddenFinalClaims   []string                     `json:"forbidden_final_claims,omitempty"`
	RequiredCapabilities   []requiredCapabilityContract `json:"required_capabilities"`
	RequiredNonclaimTerms  []string                     `json:"required_nonclaim_terms"`
	RequiredNonclaims      []string                     `json:"required_nonclaims"`
	ForbiddenClaims        []string                     `json:"forbidden_claims"`
	DocsRefs               []string                     `json:"docs_refs"`
	ValidatorRefs          []manifestRef                `json:"validator_refs"`
	GateRefs               []manifestRef                `json:"gate_refs"`
	ContractRefs           []releaseContractRef         `json:"contract_refs"`
	Capabilities           []actorCapability            `json:"capabilities"`
}

type releaseTargetContract struct {
	Primary                    string   `json:"primary,omitempty"`
	FullProductionTargets      []string `json:"full_production_targets,omitempty"`
	CompatibilityTargets       []string `json:"compatibility_targets,omitempty"`
	UnsupportedTargets         []string `json:"unsupported_targets,omitempty"`
	DistributedNonclaimTargets []string `json:"distributed_nonclaim_targets,omitempty"`
}

type requiredCapabilityContract struct {
	ID                     string   `json:"id"`
	Status                 string   `json:"status"`
	SupportedTargets       []string `json:"supported_targets"`
	RequiredTransportTerms []string `json:"required_transport_terms,omitempty"`
	ReleaseNoteTerms       []string `json:"release_note_terms,omitempty"`
}

type manifestRef struct {
	ID               string `json:"id"`
	Path             string `json:"path"`
	ClaimCheck       bool   `json:"claim_check,omitempty"`
	ReleaseNoteCheck bool   `json:"release_note_check,omitempty"`
}

type releaseContractRef struct {
	ID               string   `json:"id"`
	Path             string   `json:"path"`
	CapabilityID     string   `json:"capability_id"`
	ClaimCheck       bool     `json:"claim_check,omitempty"`
	NonclaimCheck    bool     `json:"nonclaim_check,omitempty"`
	TargetCheck      bool     `json:"target_check,omitempty"`
	ValidatorCheck   bool     `json:"validator_check,omitempty"`
	RequiredReports  []string `json:"required_reports,omitempty"`
	RequiredGateRefs []string `json:"required_gate_refs,omitempty"`
}

type actorCapability struct {
	ID                    string   `json:"id"`
	Status                string   `json:"status"`
	SupportedTargets      []string `json:"supported_targets"`
	RuntimeBoundary       string   `json:"runtime_boundary,omitempty"`
	Transport             string   `json:"transport,omitempty"`
	Claims                []string `json:"claims"`
	Nonclaims             []string `json:"nonclaims"`
	EvidenceRefs          []string `json:"evidence_refs"`
	ValidatorRefs         []string `json:"validator_refs"`
	DocsRefs              []string `json:"docs_refs"`
	PromotionRequirements []string `json:"promotion_requirements,omitempty"`
}

type actorReleaseContract struct {
	Schema       string   `json:"schema"`
	ID           string   `json:"id"`
	CapabilityID string   `json:"capability_id"`
	Target       string   `json:"target"`
	Scope        string   `json:"scope"`
	Claims       []string `json:"claims"`
	Nonclaims    []string `json:"nonclaims"`
	Validators   []string `json:"validators"`
	Reports      []string `json:"reports"`
	GateRefs     []string `json:"gate_refs"`
}

type actorCapabilityValidationOptions struct {
	ReleaseNotes []string
}

type repeatedStringFlag []string

func (flag *repeatedStringFlag) String() string {
	return strings.Join(*flag, ",")
}

func (flag *repeatedStringFlag) Set(value string) error {
	*flag = append(*flag, value)
	return nil
}

func main() {
	manifestPath := flag.String("manifest", "", "path to tetra.actor.capability_manifest.v1 JSON manifest")
	root := flag.String("root", ".", "repository root used to resolve manifest refs")
	var releaseNotes repeatedStringFlag
	flag.Var(&releaseNotes, "release-notes", "release notes file to validate against the actor capability manifest")
	flag.Parse()
	if *manifestPath == "" {
		fmt.Fprintln(os.Stderr, "error: --manifest is required")
		os.Exit(2)
	}
	options := actorCapabilityValidationOptions{ReleaseNotes: []string(releaseNotes)}
	if err := validateActorCapabilitiesManifestFileWithOptions(*manifestPath, *root, options); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateActorCapabilitiesManifestFile(path, root string) error {
	return validateActorCapabilitiesManifestFileWithOptions(path, root, actorCapabilityValidationOptions{})
}

func validateActorCapabilitiesManifestFileWithOptions(path, root string, options actorCapabilityValidationOptions) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return validateActorCapabilitiesManifest(raw, root, options)
}

func validateActorCapabilitiesManifest(raw []byte, root string, options actorCapabilityValidationOptions) error {
	var manifest actorCapabilityManifest
	if err := decodeStrictActorCapabilities(raw, &manifest); err != nil {
		return err
	}
	var issues []string
	if manifest.Schema != actorCapabilityManifestSchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", manifest.Schema, actorCapabilityManifestSchemaV1))
	}
	if strings.TrimSpace(manifest.Profile) == "" {
		issues = append(issues, "profile is required")
	}
	if len(manifest.RequiredNonclaimTerms) == 0 {
		issues = append(issues, "required_nonclaim_terms is required")
	}
	issues = append(issues, validateRequiredCapabilityContracts(manifest.RequiredCapabilities)...)
	issues = append(issues, validateRequiredTerms("required_nonclaims", manifest.RequiredNonclaims, manifest.RequiredNonclaimTerms)...)
	issues = append(issues, validateForbiddenClaimsCatalog(manifest.ForbiddenClaims)...)
	issues = append(issues, validateRequiredTerms("forbidden_claims", manifest.ForbiddenClaims, manifest.RequiredNonclaimTerms)...)
	issues = append(issues, validateV1ReleaseContract(manifest)...)

	validatorIDs := map[string]bool{}
	for _, ref := range manifest.ValidatorRefs {
		validatorIDs[ref.ID] = true
	}
	issues = append(issues, validateRefs(root, "docs_refs", stringRefs(manifest.DocsRefs))...)
	issues = append(issues, validateRefs(root, "validator_refs", manifest.ValidatorRefs)...)
	issues = append(issues, validateRefs(root, "gate_refs", manifest.GateRefs)...)
	issues = append(issues, validateContractRefs(root, manifest.ContractRefs)...)
	issues = append(issues, validateDocsCarryRequiredNonclaims(root, manifest.DocsRefs, manifest.RequiredNonclaims, manifest.RequiredNonclaimTerms, manifest.ForbiddenClaims)...)
	issues = append(issues, validateSelectedGateRefsCarryRequiredNonclaims(root, manifest.GateRefs, manifest.RequiredNonclaims, manifest.RequiredNonclaimTerms, manifest.ForbiddenClaims)...)
	issues = append(issues, validateReleaseNoteCheckRefs(root, manifest.GateRefs)...)
	issues = append(issues, validateReleaseNotes(root, options.ReleaseNotes, manifest)...)
	issues = append(issues, validateCapabilities(root, manifest.Capabilities, manifest.ForbiddenClaims, validatorIDs, manifest.RequiredCapabilities)...)
	issues = append(issues, validateReleaseContracts(root, manifest, validatorIDs)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func decodeStrictActorCapabilities(raw []byte, out any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err != nil {
			return err
		}
		return errors.New("multiple JSON values")
	}
	return nil
}

func validateForbiddenClaimsCatalog(claims []string) []string {
	if len(claims) == 0 {
		return []string{"forbidden_claims is required"}
	}
	return nil
}

func validateRequiredCapabilityContracts(required []requiredCapabilityContract) []string {
	if len(required) == 0 {
		return []string{"required_capabilities is required"}
	}
	var issues []string
	seen := map[string]bool{}
	for _, req := range required {
		if strings.TrimSpace(req.ID) == "" {
			issues = append(issues, "required_capabilities id is required")
		}
		if seen[req.ID] {
			issues = append(issues, fmt.Sprintf("duplicate required capability %s", req.ID))
		}
		seen[req.ID] = true
		switch req.Status {
		case "current_scoped", "blocked":
		default:
			issues = append(issues, fmt.Sprintf("required capability %s status is %q, want current_scoped or blocked", req.ID, req.Status))
		}
		if req.SupportedTargets == nil {
			issues = append(issues, fmt.Sprintf("required capability %s supported_targets is required", req.ID))
		}
		if hasDuplicateStrings(req.SupportedTargets) {
			issues = append(issues, fmt.Sprintf("required capability %s supported_targets contains duplicates", req.ID))
		}
	}
	return issues
}

func validateRequiredTerms(label string, values []string, terms []string) []string {
	if len(values) == 0 {
		return []string{label + " is required"}
	}
	joined := normalizeActorText(strings.Join(values, "\n"))
	var issues []string
	for _, term := range terms {
		if !strings.Contains(joined, normalizeActorText(term)) {
			issues = append(issues, fmt.Sprintf("%s missing required nonclaim term %q", label, term))
		}
	}
	return issues
}

func validateRefs(root, label string, refs []manifestRef) []string {
	if len(refs) == 0 {
		return []string{label + " is required"}
	}
	var issues []string
	seen := map[string]bool{}
	for _, ref := range refs {
		if strings.TrimSpace(ref.ID) == "" {
			issues = append(issues, label+" id is required")
		}
		path := strings.TrimSpace(ref.Path)
		if path == "" {
			issues = append(issues, label+" path is required")
			continue
		}
		if seen[path] {
			issues = append(issues, fmt.Sprintf("%s duplicate path %s", label, path))
		}
		seen[path] = true
		if err := validateRelativeSlashPath(path); err != nil {
			issues = append(issues, fmt.Sprintf("%s %s: %v", label, path, err))
			continue
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
			issues = append(issues, fmt.Sprintf("%s %s: %v", label, path, err))
		}
	}
	return issues
}

func validateRelativeSlashPath(path string) error {
	if filepath.IsAbs(path) || strings.Contains(path, "\\") || strings.Contains(path, "..") {
		return errors.New("must be a relative slash path without parent traversal")
	}
	return nil
}

func validateContractRefs(root string, refs []releaseContractRef) []string {
	if len(refs) == 0 {
		return []string{"contract_refs is required"}
	}
	var issues []string
	seen := map[string]bool{}
	for _, ref := range refs {
		if strings.TrimSpace(ref.ID) == "" {
			issues = append(issues, "contract_refs id is required")
		}
		if strings.TrimSpace(ref.CapabilityID) == "" {
			issues = append(issues, fmt.Sprintf("contract_refs %s capability_id is required", ref.ID))
		}
		path := strings.TrimSpace(ref.Path)
		if path == "" {
			issues = append(issues, fmt.Sprintf("contract_refs %s path is required", ref.ID))
			continue
		}
		if seen[path] {
			issues = append(issues, fmt.Sprintf("contract_refs duplicate path %s", path))
		}
		seen[path] = true
		if err := validateRelativeSlashPath(path); err != nil {
			issues = append(issues, fmt.Sprintf("contract_refs %s: %v", path, err))
			continue
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
			issues = append(issues, fmt.Sprintf("contract_refs %s: %v", path, err))
		}
	}
	return issues
}

func validateDocsCarryRequiredNonclaims(root string, refs []string, requiredNonclaims, terms, forbiddenClaims []string) []string {
	var issues []string
	for _, ref := range refs {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(ref)))
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", ref, err))
			continue
		}
		issues = append(issues, validateActorTextClaims(ref, string(raw), requiredNonclaims, terms, forbiddenClaims)...)
	}
	return issues
}

func validateSelectedGateRefsCarryRequiredNonclaims(root string, refs []manifestRef, requiredNonclaims, terms, forbiddenClaims []string) []string {
	var issues []string
	for _, ref := range refs {
		if !ref.ClaimCheck {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(ref.Path)))
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", ref.Path, err))
			continue
		}
		issues = append(issues, validateActorTextClaims(ref.Path, string(raw), requiredNonclaims, terms, forbiddenClaims)...)
	}
	return issues
}

func validateReleaseNoteCheckRefs(root string, refs []manifestRef) []string {
	var issues []string
	for _, ref := range refs {
		if !ref.ReleaseNoteCheck {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(ref.Path)))
		if err != nil {
			continue
		}
		if !strings.Contains(string(raw), "--release-notes") {
			issues = append(issues, fmt.Sprintf("%s is marked release_note_check but does not run validate-actor-capabilities --release-notes", ref.Path))
		}
	}
	return issues
}

func validateReleaseNotes(root string, paths []string, manifest actorCapabilityManifest) []string {
	var issues []string
	for _, path := range paths {
		if err := validateRelativeSlashPath(path); err != nil && !filepath.IsAbs(path) {
			issues = append(issues, fmt.Sprintf("release notes %s: %v", path, err))
			continue
		}
		raw, err := os.ReadFile(resolveMaybeRelativePath(root, path))
		if err != nil {
			issues = append(issues, fmt.Sprintf("release notes %s: %v", path, err))
			continue
		}
		issues = append(issues, validateActorTextClaims("release notes "+path, string(raw), manifest.RequiredNonclaims, manifest.RequiredNonclaimTerms, manifest.ForbiddenClaims)...)
		issues = append(issues, validateFinalReleaseTextClaims("release notes "+path, string(raw), manifest)...)
		text := normalizeActorText(string(raw))
		for _, req := range manifest.RequiredCapabilities {
			for _, term := range req.ReleaseNoteTerms {
				if !strings.Contains(text, normalizeActorText(term)) {
					issues = append(issues, fmt.Sprintf("release notes %s missing required actor release note term %q", path, term))
				}
			}
		}
	}
	return issues
}

func validateV1ReleaseContract(manifest actorCapabilityManifest) []string {
	if !manifestHasV1ReleaseContract(manifest) {
		return nil
	}
	var issues []string
	if manifest.ReleaseTarget.Primary != "linux-x64" {
		issues = append(issues, fmt.Sprintf("release_target.primary is %q, want linux-x64", manifest.ReleaseTarget.Primary))
	}
	if !stringInSet("linux-x64", manifest.ReleaseTarget.FullProductionTargets) {
		issues = append(issues, "release_target.full_production_targets must include linux-x64")
	}
	for _, unsupported := range manifest.ReleaseTarget.UnsupportedTargets {
		if stringInSet(unsupported, manifest.ReleaseTarget.FullProductionTargets) {
			issues = append(issues, fmt.Sprintf("release target %s cannot be both full production and unsupported", unsupported))
		}
	}
	wantVocabulary := []string{
		"TETRA_V1_NATIVE_ACTOR_PLATFORM_LINUX_X64_PROD_STABLE",
		"NEAR_READY_WITH_BLOCKERS",
		"BETA_ONLY",
		"EXPERIMENTAL_ONLY",
		"FAIL",
	}
	if !sameStringSet(manifest.FinalVerdictVocabulary, wantVocabulary) {
		issues = append(issues, fmt.Sprintf("final_verdict_vocabulary = %v, want %v", manifest.FinalVerdictVocabulary, wantVocabulary))
	}
	if manifest.RequiredFinalVerdict != "TETRA_V1_NATIVE_ACTOR_PLATFORM_LINUX_X64_PROD_STABLE" {
		issues = append(issues, fmt.Sprintf("required_final_verdict is %q, want TETRA_V1_NATIVE_ACTOR_PLATFORM_LINUX_X64_PROD_STABLE", manifest.RequiredFinalVerdict))
	}
	if strings.TrimSpace(manifest.AllowedFinalClaim) == "" {
		issues = append(issues, "allowed_final_claim is required")
	}
	if len(manifest.RequiredV1Capabilities) == 0 {
		issues = append(issues, "required_v1_capabilities is required")
	}
	if len(manifest.RequiredFinalReports) == 0 {
		issues = append(issues, "required_final_reports is required")
	}
	if len(manifest.ForbiddenFinalClaims) == 0 {
		issues = append(issues, "forbidden_final_claims is required")
	}
	requiredCapabilities := map[string]bool{}
	for _, req := range manifest.RequiredCapabilities {
		requiredCapabilities[req.ID] = true
	}
	capabilities := map[string]bool{}
	for _, cap := range manifest.Capabilities {
		capabilities[cap.ID] = true
	}
	for _, id := range manifest.RequiredV1Capabilities {
		if !requiredCapabilities[id] {
			issues = append(issues, fmt.Sprintf("required_v1_capabilities %s is not declared in required_capabilities", id))
		}
		if !capabilities[id] {
			issues = append(issues, fmt.Sprintf("required_v1_capabilities %s is not declared in capabilities", id))
		}
	}
	for _, report := range manifest.RequiredFinalReports {
		report = strings.TrimSpace(report)
		if report == "" {
			issues = append(issues, "required_final_reports contains an empty report")
			continue
		}
		if filepath.IsAbs(report) || strings.Contains(report, "..") || strings.Contains(report, "\\") {
			issues = append(issues, fmt.Sprintf("required_final_report %q must be a relative slash path", report))
		}
	}
	return issues
}

func manifestHasV1ReleaseContract(manifest actorCapabilityManifest) bool {
	return manifest.ReleaseTarget.Primary != "" ||
		len(manifest.FinalVerdictVocabulary) != 0 ||
		manifest.RequiredFinalVerdict != "" ||
		manifest.AllowedFinalClaim != "" ||
		len(manifest.RequiredV1Capabilities) != 0 ||
		len(manifest.RequiredFinalReports) != 0 ||
		len(manifest.ForbiddenFinalClaims) != 0
}

func validateFinalReleaseTextClaims(label, text string, manifest actorCapabilityManifest) []string {
	if !manifestHasV1ReleaseContract(manifest) {
		return nil
	}
	var issues []string
	claimText := actorClaimTextWithoutAllowedNonclaims(text, manifest.RequiredNonclaims)
	normalized := normalizeActorText(claimText)
	finalVerdict := strings.TrimSpace(manifest.RequiredFinalVerdict)
	if finalVerdict != "" && strings.Contains(normalized, normalizeActorText(finalVerdict)) {
		issues = append(issues, validateRequiredV1CapabilitiesForClaim(label, finalVerdict, manifest.RequiredV1Capabilities, manifest)...)
	}
	if allowedClaim := strings.TrimSpace(manifest.AllowedFinalClaim); allowedClaim != "" &&
		strings.Contains(normalized, normalizeActorText(allowedClaim)) {
		issues = append(issues, validateRequiredV1CapabilitiesForClaim(label, "allowed final v1 claim", manifest.RequiredV1Capabilities, manifest)...)
	}
	if strings.Contains(normalized, "cluster membership") {
		issues = append(issues, validateRequiredV1CapabilitiesForClaim(label, "cluster membership claim", []string{"cluster_membership"}, manifest)...)
	}
	if strings.Contains(normalized, "supervision") || strings.Contains(normalized, "supervisor") {
		issues = append(issues, validateRequiredV1CapabilitiesForClaim(
			label,
			"supervision claim",
			[]string{"actor_lifecycle_supervision", "supervision_restart_tree"},
			manifest,
		)...)
	}
	if hasRustCParityClaim(claimText) {
		issues = append(issues, validateRequiredV1CapabilitiesForClaim(label, "Rust/C parity claim", []string{"benchmark_rust_c_parity"}, manifest)...)
	}
	if hasNativeApplicationPlatformClaim(claimText) {
		issues = append(issues, validateRequiredV1CapabilitiesForClaim(label, "native application platform claim", []string{"native_surface_host"}, manifest)...)
		if hasOldRealWindowProbeEvidenceClaim(claimText) {
			issues = append(issues, fmt.Sprintf("%s native_surface_host claim uses old real-window probe evidence; final native app evidence must use the direct native Surface Host path", label))
		}
	}
	for _, forbidden := range manifest.ForbiddenFinalClaims {
		forbiddenText := normalizeActorText(forbidden)
		if forbiddenText == "" {
			continue
		}
		if strings.Contains(normalized, forbiddenText) {
			issues = append(issues, fmt.Sprintf("%s forbidden final v1 claim %q", label, forbidden))
		}
	}
	return issues
}

func actorClaimTextWithoutAllowedNonclaims(text string, allowedNonclaims []string) string {
	normalized := normalizeActorText(text)
	for _, allowed := range allowedNonclaims {
		normalized = strings.ReplaceAll(normalized, normalizeActorText(allowed), " ")
	}
	return normalizeActorText(normalized)
}

func validateRequiredV1CapabilitiesForClaim(label, claim string, ids []string, manifest actorCapabilityManifest) []string {
	capabilities := map[string]actorCapability{}
	for _, cap := range manifest.Capabilities {
		capabilities[cap.ID] = cap
	}
	var issues []string
	for _, id := range ids {
		cap, ok := capabilities[id]
		if !ok {
			issues = append(issues, fmt.Sprintf("%s %s requires missing capability %s", label, claim, id))
			continue
		}
		if cap.Status != "current_scoped" || !stringInSet("linux-x64", cap.SupportedTargets) || len(cap.EvidenceRefs) == 0 {
			issues = append(issues, fmt.Sprintf(
				"%s %s requires capability %s current_scoped linux-x64 evidence; status=%q supported_targets=%v evidence_refs=%v",
				label,
				claim,
				id,
				cap.Status,
				cap.SupportedTargets,
				cap.EvidenceRefs,
			))
		}
	}
	return issues
}

func hasRustCParityClaim(text string) bool {
	normalized := normalizeClaimText(text)
	return strings.Contains(normalized, "rust c performance parity") ||
		strings.Contains(normalized, "rust c level speed") ||
		(strings.Contains(normalized, "rust") && strings.Contains(normalized, " c ") &&
			(strings.Contains(normalized, "parity") || strings.Contains(normalized, "within")))
}

func hasNativeApplicationPlatformClaim(text string) bool {
	normalized := normalizeClaimText(text)
	return strings.Contains(normalized, "native application platform") ||
		strings.Contains(normalized, "native app platform") ||
		strings.Contains(normalized, "native linux application")
}

func hasOldRealWindowProbeEvidenceClaim(text string) bool {
	normalized := normalizeClaimText(text)
	return strings.Contains(normalized, "real window probe") ||
		strings.Contains(normalized, "old real window") ||
		strings.Contains(normalized, "linux x64 real window probe")
}

func resolveMaybeRelativePath(root, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, filepath.FromSlash(path))
}

func validateActorTextClaims(label, text string, requiredNonclaims, terms, forbiddenClaims []string) []string {
	var issues []string
	normalized := normalizeActorText(text)
	for _, phrase := range requiredNonclaims {
		if !strings.Contains(normalized, normalizeActorText(phrase)) {
			issues = append(issues, fmt.Sprintf("%s missing required nonclaim phrase %q", label, phrase))
		}
	}
	for _, term := range terms {
		if !strings.Contains(normalized, normalizeActorText(term)) {
			issues = append(issues, fmt.Sprintf("%s missing required nonclaim term %q", label, term))
		}
	}
	issues = append(issues, validateNoForbiddenActorPromotions(label, text, requiredNonclaims, forbiddenClaims)...)
	return issues
}

func validateNoForbiddenActorPromotions(label, text string, allowedNonclaims, forbiddenClaims []string) []string {
	normalized := normalizeActorText(text)
	for _, allowed := range allowedNonclaims {
		normalized = strings.ReplaceAll(normalized, normalizeActorText(allowed), " ")
	}
	for _, allowed := range allowedActorRuntimeNonclaimSentences() {
		normalized = strings.ReplaceAll(normalized, normalizeActorText(allowed), " ")
	}
	normalized = normalizeActorText(normalized)
	var issues []string
	for _, forbidden := range forbiddenClaims {
		forbiddenText := normalizeActorText(forbidden)
		if forbiddenText == "" {
			continue
		}
		if strings.Contains(normalized, forbiddenText) {
			issues = append(issues, fmt.Sprintf("%s forbidden actor promotion claim %q", label, forbidden))
		}
	}
	return issues
}

func allowedActorRuntimeNonclaimSentences() []string {
	return []string{
		"Actor runtime foundation evidence remains Linux-x64 scoped; Erlang/OTP, cluster membership, reconnect/retry production, non-Linux distributed runtime, distributed zero-copy transfer, and formal race proof are not claimed.",
	}
}

func validateCapabilities(root string, capabilities []actorCapability, forbiddenClaims []string, validatorIDs map[string]bool, required []requiredCapabilityContract) []string {
	if len(capabilities) == 0 {
		return []string{"capabilities is required"}
	}
	requiredByID := map[string]requiredCapabilityContract{}
	for _, req := range required {
		requiredByID[req.ID] = req
	}
	var issues []string
	byID := map[string]actorCapability{}
	for _, cap := range capabilities {
		id := strings.TrimSpace(cap.ID)
		if id == "" {
			issues = append(issues, "capability id is required")
			continue
		}
		if _, ok := byID[id]; ok {
			issues = append(issues, fmt.Sprintf("duplicate capability %s", id))
		}
		byID[id] = cap
		issues = append(issues, validateCapability(root, cap, forbiddenClaims, validatorIDs)...)
	}
	for _, req := range required {
		cap, ok := byID[req.ID]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing required capability %s", req.ID))
			continue
		}
		issues = append(issues, validateCapabilityContract(cap, req)...)
	}
	return issues
}

func validateReleaseContracts(root string, manifest actorCapabilityManifest, validatorIDs map[string]bool) []string {
	if len(manifest.ContractRefs) == 0 {
		return []string{"contract_refs is required"}
	}
	capabilitiesByID := map[string]actorCapability{}
	for _, cap := range manifest.Capabilities {
		capabilitiesByID[cap.ID] = cap
	}
	gatePaths := map[string]bool{}
	for _, ref := range manifest.GateRefs {
		gatePaths[ref.Path] = true
	}
	var issues []string
	for _, ref := range manifest.ContractRefs {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(ref.Path)))
		if err != nil {
			issues = append(issues, fmt.Sprintf("contract %s: %v", ref.Path, err))
			continue
		}
		var contract actorReleaseContract
		if err := decodeStrictActorCapabilities(raw, &contract); err != nil {
			issues = append(issues, fmt.Sprintf("contract %s: %v", ref.Path, err))
			continue
		}
		issues = append(issues, validateReleaseContract(ref, contract, capabilitiesByID, validatorIDs, gatePaths, manifest.ForbiddenClaims)...)
	}
	return issues
}

func validateReleaseContract(ref releaseContractRef, contract actorReleaseContract, capabilitiesByID map[string]actorCapability, validatorIDs, gatePaths map[string]bool, forbiddenClaims []string) []string {
	var issues []string
	if contract.Schema != "tetra.actor.release_contract.v1" {
		issues = append(issues, fmt.Sprintf("contract %s schema is %q, want tetra.actor.release_contract.v1", ref.Path, contract.Schema))
	}
	if contract.CapabilityID != ref.CapabilityID {
		issues = append(issues, fmt.Sprintf("contract %s capability_id = %q, want %q", ref.Path, contract.CapabilityID, ref.CapabilityID))
	}
	capability, ok := capabilitiesByID[ref.CapabilityID]
	if !ok {
		issues = append(issues, fmt.Sprintf("contract %s capability %s not found in manifest", ref.Path, ref.CapabilityID))
		return issues
	}
	if ref.TargetCheck {
		if !stringInSet(contract.Target, capability.SupportedTargets) {
			issues = append(issues, fmt.Sprintf("contract %s target = %q, want one of %v", ref.Path, contract.Target, capability.SupportedTargets))
		}
		if contract.Scope != "" && contract.Scope != contract.Target {
			issues = append(issues, fmt.Sprintf("contract %s scope = %q, want target %q", ref.Path, contract.Scope, contract.Target))
		}
	}
	if ref.ClaimCheck {
		for _, claim := range contract.Claims {
			if !normalizedStringInSet(claim, capability.Claims) {
				issues = append(issues, fmt.Sprintf("contract %s claim %q is not declared by capability %s", ref.Path, claim, capability.ID))
			}
		}
		issues = append(issues, validateNoForbiddenActorPromotions("contract "+ref.Path, strings.Join(contract.Claims, "\n"), capability.Nonclaims, forbiddenClaims)...)
	}
	if ref.NonclaimCheck {
		for _, nonclaim := range capability.Nonclaims {
			if !normalizedStringInSet(nonclaim, contract.Nonclaims) {
				issues = append(issues, fmt.Sprintf("contract %s missing capability nonclaim %q", ref.Path, nonclaim))
			}
		}
	}
	if ref.ValidatorCheck {
		for _, validator := range contract.Validators {
			if !validatorIDs[validator] {
				issues = append(issues, fmt.Sprintf("contract %s validator %s is not declared in validator_refs", ref.Path, validator))
			}
		}
		for _, validator := range capability.ValidatorRefs {
			if !stringInSet(validator, contract.Validators) {
				issues = append(issues, fmt.Sprintf("contract %s missing capability validator %s", ref.Path, validator))
			}
		}
	}
	for _, report := range ref.RequiredReports {
		if !stringInSet(report, contract.Reports) {
			issues = append(issues, fmt.Sprintf("contract %s missing required report %s", ref.Path, report))
		}
	}
	for _, gateRef := range ref.RequiredGateRefs {
		if !stringInSet(gateRef, contract.GateRefs) {
			issues = append(issues, fmt.Sprintf("contract %s missing required gate_ref %s", ref.Path, gateRef))
		}
		if !gatePaths[gateRef] {
			issues = append(issues, fmt.Sprintf("contract %s required gate_ref %s is not declared in manifest gate_refs", ref.Path, gateRef))
		}
	}
	return issues
}

func validateCapability(root string, cap actorCapability, forbiddenClaims []string, validatorIDs map[string]bool) []string {
	var issues []string
	switch cap.Status {
	case "current_scoped":
		if len(cap.SupportedTargets) == 0 {
			issues = append(issues, fmt.Sprintf("capability %s supported_targets are required for current_scoped", cap.ID))
		}
		if len(cap.Claims) == 0 {
			issues = append(issues, fmt.Sprintf("capability %s claims are required for current_scoped", cap.ID))
		}
		if len(cap.Nonclaims) == 0 {
			issues = append(issues, fmt.Sprintf("capability %s nonclaims are required for current_scoped", cap.ID))
		}
		if len(cap.EvidenceRefs) == 0 {
			issues = append(issues, fmt.Sprintf("capability %s evidence_refs are required for current_scoped", cap.ID))
		}
		if len(cap.ValidatorRefs) == 0 {
			issues = append(issues, fmt.Sprintf("capability %s validator_refs are required for current_scoped", cap.ID))
		}
		if len(cap.DocsRefs) == 0 {
			issues = append(issues, fmt.Sprintf("capability %s docs_refs are required for current_scoped", cap.ID))
		}
	case "blocked":
		if len(cap.Claims) != 0 {
			issues = append(issues, fmt.Sprintf("blocked capability %s must not carry claims", cap.ID))
		}
		if len(cap.PromotionRequirements) == 0 {
			issues = append(issues, fmt.Sprintf("blocked capability %s promotion_requirements are required", cap.ID))
		}
	default:
		issues = append(issues, fmt.Sprintf("capability %s status is %q, want current_scoped or blocked", cap.ID, cap.Status))
	}
	issues = append(issues, validateCapabilityRefs(root, cap)...)
	issues = append(issues, validateCapabilityValidatorRefs(cap, validatorIDs)...)
	issues = append(issues, rejectForbiddenCapabilityClaims(cap, forbiddenClaims)...)
	return issues
}

func validateCapabilityContract(cap actorCapability, req requiredCapabilityContract) []string {
	var issues []string
	if cap.Status != req.Status {
		issues = append(issues, fmt.Sprintf("%s status = %q, want %q", cap.ID, cap.Status, req.Status))
	}
	if !sameStringSet(cap.SupportedTargets, req.SupportedTargets) {
		issues = append(issues, fmt.Sprintf("%s supported_targets = %v, want %v", cap.ID, cap.SupportedTargets, req.SupportedTargets))
	}
	transport := normalizeActorText(cap.Transport)
	for _, term := range req.RequiredTransportTerms {
		if !strings.Contains(transport, normalizeActorText(term)) {
			issues = append(issues, fmt.Sprintf("%s transport missing required term %q", cap.ID, term))
		}
	}
	return issues
}

func validateCapabilityRefs(root string, cap actorCapability) []string {
	var issues []string
	for _, ref := range append(append([]string{}, cap.EvidenceRefs...), cap.DocsRefs...) {
		if err := validateRelativeSlashPath(ref); err != nil {
			issues = append(issues, fmt.Sprintf("capability %s ref %s: %v", cap.ID, ref, err))
			continue
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(ref))); err != nil {
			issues = append(issues, fmt.Sprintf("capability %s ref %s: %v", cap.ID, ref, err))
		}
	}
	return issues
}

func validateCapabilityValidatorRefs(cap actorCapability, validatorIDs map[string]bool) []string {
	var issues []string
	for _, ref := range cap.ValidatorRefs {
		if !validatorIDs[ref] {
			issues = append(issues, fmt.Sprintf("capability %s validator_ref %s is not declared in validator_refs", cap.ID, ref))
		}
	}
	return issues
}

func rejectForbiddenCapabilityClaims(cap actorCapability, forbiddenClaims []string) []string {
	var issues []string
	for _, claim := range cap.Claims {
		normalizedClaim := normalizeActorText(claim)
		for _, forbidden := range forbiddenClaims {
			normalizedForbidden := normalizeActorText(forbidden)
			if normalizedForbidden != "" && strings.Contains(normalizedClaim, normalizedForbidden) {
				issues = append(issues, fmt.Sprintf("capability %s forbidden actor claim %q mentions %q", cap.ID, claim, forbidden))
			}
		}
	}
	return issues
}

func sameStringSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := map[string]int{}
	for _, value := range got {
		seen[value]++
	}
	for _, value := range want {
		seen[value]--
	}
	for _, count := range seen {
		if count != 0 {
			return false
		}
	}
	return true
}

func hasDuplicateStrings(values []string) bool {
	seen := map[string]bool{}
	for _, value := range values {
		if seen[value] {
			return true
		}
		seen[value] = true
	}
	return false
}

func stringInSet(value string, set []string) bool {
	for _, candidate := range set {
		if value == candidate {
			return true
		}
	}
	return false
}

func normalizedStringInSet(value string, set []string) bool {
	normalized := normalizeActorText(value)
	for _, candidate := range set {
		if normalized == normalizeActorText(candidate) {
			return true
		}
	}
	return false
}

func stringRefs(paths []string) []manifestRef {
	refs := make([]manifestRef, 0, len(paths))
	for _, path := range paths {
		refs = append(refs, manifestRef{ID: path, Path: path})
	}
	return refs
}

func normalizeActorText(text string) string {
	mapped := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return ' '
	}, text)
	return strings.Join(strings.Fields(mapped), " ")
}

func normalizeClaimText(text string) string {
	replacer := strings.NewReplacer("-", " ", "_", " ", "/", " ")
	return strings.Join(strings.Fields(replacer.Replace(strings.ToLower(text))), " ")
}
