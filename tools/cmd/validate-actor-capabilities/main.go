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
)

const actorCapabilityManifestSchemaV1 = "tetra.actor.capability_manifest.v1"

type actorCapabilityManifest struct {
	Schema                string                       `json:"schema"`
	Profile               string                       `json:"profile"`
	RequiredCapabilities  []requiredCapabilityContract `json:"required_capabilities"`
	RequiredNonclaimTerms []string                     `json:"required_nonclaim_terms"`
	RequiredNonclaims     []string                     `json:"required_nonclaims"`
	ForbiddenClaims       []string                     `json:"forbidden_claims"`
	DocsRefs              []string                     `json:"docs_refs"`
	ValidatorRefs         []manifestRef                `json:"validator_refs"`
	GateRefs              []manifestRef                `json:"gate_refs"`
	Capabilities          []actorCapability            `json:"capabilities"`
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

	validatorIDs := map[string]bool{}
	for _, ref := range manifest.ValidatorRefs {
		validatorIDs[ref.ID] = true
	}
	issues = append(issues, validateRefs(root, "docs_refs", stringRefs(manifest.DocsRefs))...)
	issues = append(issues, validateRefs(root, "validator_refs", manifest.ValidatorRefs)...)
	issues = append(issues, validateRefs(root, "gate_refs", manifest.GateRefs)...)
	issues = append(issues, validateDocsCarryRequiredNonclaims(root, manifest.DocsRefs, manifest.RequiredNonclaimTerms)...)
	issues = append(issues, validateSelectedGateRefsCarryRequiredNonclaims(root, manifest.GateRefs, manifest.RequiredNonclaimTerms)...)
	issues = append(issues, validateReleaseNoteCheckRefs(root, manifest.GateRefs)...)
	issues = append(issues, validateReleaseNotes(root, options.ReleaseNotes, manifest)...)
	issues = append(issues, validateCapabilities(root, manifest.Capabilities, manifest.ForbiddenClaims, validatorIDs, manifest.RequiredCapabilities)...)
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

func validateDocsCarryRequiredNonclaims(root string, refs []string, terms []string) []string {
	var issues []string
	for _, ref := range refs {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(ref)))
		if err != nil {
			continue
		}
		text := normalizeActorText(string(raw))
		for _, term := range terms {
			if !strings.Contains(text, normalizeActorText(term)) {
				issues = append(issues, fmt.Sprintf("%s missing required nonclaim term %q", ref, term))
			}
		}
	}
	return issues
}

func validateSelectedGateRefsCarryRequiredNonclaims(root string, refs []manifestRef, terms []string) []string {
	var issues []string
	for _, ref := range refs {
		if !ref.ClaimCheck {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(ref.Path)))
		if err != nil {
			continue
		}
		text := normalizeActorText(string(raw))
		for _, term := range terms {
			if !strings.Contains(text, normalizeActorText(term)) {
				issues = append(issues, fmt.Sprintf("%s missing required nonclaim term %q", ref.Path, term))
			}
		}
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
		text := normalizeActorText(string(raw))
		for _, term := range manifest.RequiredNonclaimTerms {
			if !strings.Contains(text, normalizeActorText(term)) {
				issues = append(issues, fmt.Sprintf("release notes %s missing required nonclaim term %q", path, term))
			}
		}
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

func resolveMaybeRelativePath(root, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, filepath.FromSlash(path))
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

func stringRefs(paths []string) []manifestRef {
	refs := make([]manifestRef, 0, len(paths))
	for _, path := range paths {
		refs = append(refs, manifestRef{ID: path, Path: path})
	}
	return refs
}

func normalizeActorText(text string) string {
	replacer := strings.NewReplacer("-", " ", "_", " ", "/", " ")
	return strings.Join(strings.Fields(replacer.Replace(strings.ToLower(text))), " ")
}
