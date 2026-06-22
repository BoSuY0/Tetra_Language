package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateActorCapabilityAcceptsCanonicalManifest(t *testing.T) {
	root := repoRoot(t)
	manifest := filepath.Join(root, "docs", "contracts", "actors", "actor-capability-manifest.v1.json")
	if err := validateActorCapabilitiesManifestFile(manifest, root); err != nil {
		t.Fatalf("canonical actor capability manifest rejected: %v", err)
	}
}

func TestValidateActorCapabilityRejectsForbiddenPromotionClaim(t *testing.T) {
	root := repoRoot(t)
	raw := strings.Replace(validActorCapabilitiesManifestJSON(t), `"linux-x64 scoped actor/task runtime foundation evidence"`, `"full Erlang/OTP actor runtime"`, 1)
	path := writeActorCapabilitiesFixture(t, raw)
	err := validateActorCapabilitiesManifestFile(path, root)
	if err == nil {
		t.Fatalf("expected forbidden actor promotion claim rejection")
	}
	if !strings.Contains(err.Error(), "forbidden actor claim") {
		t.Fatalf("error = %v, want forbidden actor claim", err)
	}
}

func TestValidateActorCapabilityRejectsMissingRequiredCapability(t *testing.T) {
	root := repoRoot(t)
	raw := actorCapabilitiesFixtureWithoutCapability(t, "production_broker_deployment")
	path := writeActorCapabilitiesFixture(t, raw)
	err := validateActorCapabilitiesManifestFile(path, root)
	if err == nil {
		t.Fatalf("expected missing required capability rejection")
	}
	if !strings.Contains(err.Error(), "missing required capability production_broker_deployment") {
		t.Fatalf("error = %v, want missing production_broker_deployment", err)
	}
}

func TestValidateActorCapabilityRejectsManifestOwnedMissingCapability(t *testing.T) {
	root := repoRoot(t)
	raw := actorCapabilitiesFixtureWithRequiredCapability(t, "future_actor_contract")
	path := writeActorCapabilitiesFixture(t, raw)
	err := validateActorCapabilitiesManifestFile(path, root)
	if err == nil {
		t.Fatalf("expected manifest-owned missing required capability rejection")
	}
	if !strings.Contains(err.Error(), "missing required capability future_actor_contract") {
		t.Fatalf("error = %v, want missing future_actor_contract", err)
	}
}

func TestValidateActorCapabilityRejectsManifestOwnedMissingNonclaimTerm(t *testing.T) {
	root := repoRoot(t)
	raw := actorCapabilitiesFixtureWithRequiredNonclaimTerm(t, "otp parity")
	path := writeActorCapabilitiesFixture(t, raw)
	err := validateActorCapabilitiesManifestFile(path, root)
	if err == nil {
		t.Fatalf("expected manifest-owned missing required nonclaim term rejection")
	}
	if !strings.Contains(err.Error(), `missing required nonclaim term "otp parity"`) {
		t.Fatalf("error = %v, want missing otp parity term", err)
	}
}

func TestValidateActorCapabilityRejectsLocalRuntimeTargetDrift(t *testing.T) {
	root := repoRoot(t)
	raw := actorCapabilitiesFixtureWithTargets(t, "local_actor_runtime", []string{"linux-x64"})
	path := writeActorCapabilitiesFixture(t, raw)
	err := validateActorCapabilitiesManifestFile(path, root)
	if err == nil {
		t.Fatalf("expected local actor runtime target drift rejection")
	}
	if !strings.Contains(err.Error(), "local_actor_runtime supported_targets") {
		t.Fatalf("error = %v, want local_actor_runtime supported_targets", err)
	}
}

func TestValidateActorCapabilityRejectsFoundationTargetDrift(t *testing.T) {
	root := repoRoot(t)
	raw := actorCapabilitiesFixtureWithTargets(t, "actor_runtime_foundation_linux_x64", []string{"linux-x64", "macos-x64"})
	path := writeActorCapabilitiesFixture(t, raw)
	err := validateActorCapabilitiesManifestFile(path, root)
	if err == nil {
		t.Fatalf("expected actor runtime foundation target drift rejection")
	}
	if !strings.Contains(err.Error(), "actor_runtime_foundation_linux_x64 supported_targets") {
		t.Fatalf("error = %v, want actor_runtime_foundation_linux_x64 supported_targets", err)
	}
}

func TestValidateActorCapabilityRequiresSystemMessageLaneScopedClaim(t *testing.T) {
	var manifest actorCapabilityManifest
	if err := json.Unmarshal([]byte(validActorCapabilitiesManifestJSON(t)), &manifest); err != nil {
		t.Fatal(err)
	}

	const capabilityID = "system_message_runtime_lane_linux_x64"
	const claim = "source-level system-message API and isolated runtime system lane implemented for Linux-x64 builtin runtime"
	requiredNonclaims := []string{
		"real local link/monitor producers are completed in P06",
		"authenticated node-down producer is completed in P10",
		"no full Erlang/OTP actor runtime claim",
		"no cluster membership or reconnect/retry production claim",
	}

	var req *requiredCapabilityContract
	for index := range manifest.RequiredCapabilities {
		if manifest.RequiredCapabilities[index].ID == capabilityID {
			req = &manifest.RequiredCapabilities[index]
			break
		}
	}
	if req == nil {
		t.Fatalf("required capability %s missing", capabilityID)
	}
	if req.Status != "current_scoped" || !sameStringSet(req.SupportedTargets, []string{"linux-x64"}) {
		t.Fatalf(
			"required capability %s = status %q targets %v, want current_scoped linux-x64",
			capabilityID,
			req.Status,
			req.SupportedTargets,
		)
	}

	var cap *actorCapability
	for index := range manifest.Capabilities {
		if manifest.Capabilities[index].ID == capabilityID {
			cap = &manifest.Capabilities[index]
			break
		}
	}
	if cap == nil {
		t.Fatalf("capability %s missing", capabilityID)
	}
	if cap.Status != "current_scoped" || !sameStringSet(cap.SupportedTargets, []string{"linux-x64"}) {
		t.Fatalf(
			"capability %s = status %q targets %v, want current_scoped linux-x64",
			capabilityID,
			cap.Status,
			cap.SupportedTargets,
		)
	}
	if !stringInSet(claim, cap.Claims) {
		t.Fatalf("capability %s missing scoped claim %q", capabilityID, claim)
	}
	for _, nonclaim := range requiredNonclaims {
		if !stringInSet(nonclaim, cap.Nonclaims) {
			t.Fatalf("capability %s missing nonclaim %q", capabilityID, nonclaim)
		}
	}
	for _, ref := range []string{
		"validate-actor-system-messages",
		"validate-actor-capabilities",
	} {
		if !stringInSet(ref, cap.ValidatorRefs) {
			t.Fatalf("capability %s missing validator_ref %s", capabilityID, ref)
		}
	}
	for _, ref := range []string{
		"docs/design/actor_system_messages_v1.md",
		"lib/core/actors/actors.tetra",
		"compiler/internal/actorsrt/actorsrt_suite_test.go",
	} {
		if !stringInSet(ref, cap.EvidenceRefs) && !stringInSet(ref, cap.DocsRefs) {
			t.Fatalf("capability %s missing evidence/docs ref %s", capabilityID, ref)
		}
	}

	const contractPath = "docs/contracts/actors/system-message-runtime-lane-linux-x64.release-contract.v1.json"
	for _, ref := range manifest.ContractRefs {
		if ref.CapabilityID == capabilityID && ref.Path == contractPath &&
			ref.ClaimCheck && ref.NonclaimCheck && ref.TargetCheck && ref.ValidatorCheck {
			return
		}
	}
	t.Fatalf("manifest missing checked release contract ref for %s at %s", capabilityID, contractPath)
}

func TestValidateActorCapabilityRejectsMissingReferencedFile(t *testing.T) {
	root := repoRoot(t)
	raw := strings.Replace(validActorCapabilitiesManifestJSON(t), `"docs/spec/actors.md"`, `"docs/spec/missing-actors.md"`, 1)
	path := writeActorCapabilitiesFixture(t, raw)
	err := validateActorCapabilitiesManifestFile(path, root)
	if err == nil {
		t.Fatalf("expected missing referenced file rejection")
	}
	if !strings.Contains(err.Error(), "docs/spec/missing-actors.md") {
		t.Fatalf("error = %v, want missing docs/spec/missing-actors.md", err)
	}
}

func TestValidateActorCapabilityRejectsDocsMissingRequiredNonclaimTerms(t *testing.T) {
	root := repoRoot(t)
	raw := strings.Replace(validActorCapabilitiesManifestJSON(t), `"docs/spec/actors.md"`, `".github/workflows/ci.yml"`, 1)
	path := writeActorCapabilitiesFixture(t, raw)
	err := validateActorCapabilitiesManifestFile(path, root)
	if err == nil {
		t.Fatalf("expected docs nonclaim drift rejection")
	}
	if !strings.Contains(err.Error(), "missing required nonclaim term") {
		t.Fatalf("error = %v, want missing required nonclaim term", err)
	}
}

func TestValidateActorCapabilityRejectsDocsMissingRequiredNonclaimPhrase(t *testing.T) {
	fixture := writeMinimalActorCapabilitiesFixture(t, minimalActorCapabilitiesOptions{
		DocText: strings.Join([]string{"erlang", "cluster", "reconnect", "retry", "non-linux", "zero-copy", "formal race"}, "\n"),
	})
	err := validateActorCapabilitiesManifestFile(fixture.ManifestPath, fixture.Root)
	if err == nil {
		t.Fatalf("expected docs exact nonclaim phrase rejection")
	}
	if !strings.Contains(err.Error(), "missing required nonclaim phrase") {
		t.Fatalf("error = %v, want missing required nonclaim phrase", err)
	}
}

func TestValidateActorCapabilityRejectsDocsForbiddenActorPromotionClaim(t *testing.T) {
	fixture := writeMinimalActorCapabilitiesFixture(t, minimalActorCapabilitiesOptions{
		DocText: minimalRequiredNonclaimsText() + "\nTetra now supports full Erlang/OTP actor runtime.",
	})
	err := validateActorCapabilitiesManifestFile(fixture.ManifestPath, fixture.Root)
	if err == nil {
		t.Fatalf("expected docs forbidden actor promotion rejection")
	}
	if !strings.Contains(err.Error(), "forbidden actor promotion claim") {
		t.Fatalf("error = %v, want forbidden actor promotion claim", err)
	}
}

func TestValidateActorCapabilityRejectsGateForbiddenActorPromotionClaim(t *testing.T) {
	fixture := writeMinimalActorCapabilitiesFixture(t, minimalActorCapabilitiesOptions{
		GateText: minimalRequiredNonclaimsText() + "\nTetra now supports cluster membership.",
	})
	err := validateActorCapabilitiesManifestFile(fixture.ManifestPath, fixture.Root)
	if err == nil {
		t.Fatalf("expected gate forbidden actor promotion rejection")
	}
	if !strings.Contains(err.Error(), "forbidden actor promotion claim") {
		t.Fatalf("error = %v, want forbidden actor promotion claim", err)
	}
}

func TestValidateActorCapabilityRejectsReleaseNotesForbiddenActorPromotionClaim(t *testing.T) {
	fixture := writeMinimalActorCapabilitiesFixture(t, minimalActorCapabilitiesOptions{
		ReleaseNotesText: minimalRequiredNonclaimsText() + "\nActor runtime foundation evidence.\nTetra now supports full Erlang/OTP actor runtime.",
	})
	err := validateActorCapabilitiesManifestFileWithOptions(fixture.ManifestPath, fixture.Root, actorCapabilityValidationOptions{ReleaseNotes: []string{fixture.ReleaseNotesPath}})
	if err == nil {
		t.Fatalf("expected release notes forbidden actor promotion rejection")
	}
	if !strings.Contains(err.Error(), "forbidden actor promotion claim") {
		t.Fatalf("error = %v, want forbidden actor promotion claim", err)
	}
}

func TestValidateActorCapabilityRejectsReleaseNotesMissingManifestTerms(t *testing.T) {
	root := repoRoot(t)
	manifest := filepath.Join(root, "docs", "contracts", "actors", "actor-capability-manifest.v1.json")
	notes := filepath.Join(t.TempDir(), "release-notes.md")
	if err := os.WriteFile(notes, []byte("Actor runtime foundation evidence remains Linux-x64 scoped."), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateActorCapabilitiesManifestFileWithOptions(manifest, root, actorCapabilityValidationOptions{ReleaseNotes: []string{notes}})
	if err == nil {
		t.Fatalf("expected release notes manifest drift rejection")
	}
	if !strings.Contains(err.Error(), "release notes") || !strings.Contains(err.Error(), "missing required nonclaim term") {
		t.Fatalf("error = %v, want release notes missing required nonclaim term", err)
	}
}

func TestValidateActorCapabilityRejectsFullV1VerdictWhileSchedulerBlocked(t *testing.T) {
	err := validateCanonicalActorManifestWithReleaseNotes(
		t,
		actorV1ReleaseNotesBase()+
			"\nFinal verdict: TETRA_V1_NATIVE_ACTOR_PLATFORM_LINUX_X64_PROD_STABLE.\n",
	)
	if err == nil {
		t.Fatalf("expected full v1 verdict to fail while required capabilities are blocked")
	}
	for _, want := range []string{
		"TETRA_V1_NATIVE_ACTOR_PLATFORM_LINUX_X64_PROD_STABLE",
		"production_multithreaded_scheduler",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q", err, want)
		}
	}
}

func TestValidateActorCapabilityRejectsClusterClaimWithoutMembershipReport(t *testing.T) {
	err := validateCanonicalActorManifestWithReleaseNotes(
		t,
		actorV1ReleaseNotesBase()+
			"\nTetra v1 now provides broker-authoritative cluster membership for distributed actors.\n",
	)
	if err == nil {
		t.Fatalf("expected cluster claim without membership report to fail")
	}
	if !strings.Contains(err.Error(), "cluster_membership") {
		t.Fatalf("error = %v, want cluster_membership capability rejection", err)
	}
}

func TestValidateActorCapabilityRejectsSupervisionClaimWithoutLifecycleReports(t *testing.T) {
	err := validateCanonicalActorManifestWithReleaseNotes(
		t,
		actorV1ReleaseNotesBase()+
			"\nTetra v1 now provides actor lifecycle supervision and supervision restart trees.\n",
	)
	if err == nil {
		t.Fatalf("expected supervision claim without lifecycle and restart reports to fail")
	}
	for _, want := range []string{"actor_lifecycle_supervision", "supervision_restart_tree"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q", err, want)
		}
	}
}

func TestValidateActorCapabilityRejectsRustCParityFromLocalTierEvidence(t *testing.T) {
	err := validateCanonicalActorManifestWithReleaseNotes(
		t,
		actorV1ReleaseNotesBase()+
			"\nTetra v1 reaches Rust/C performance parity from Tier 0/Tier 1 local actor evidence.\n",
	)
	if err == nil {
		t.Fatalf("expected Rust/C parity wording from local-only evidence to fail")
	}
	if !strings.Contains(err.Error(), "benchmark_rust_c_parity") {
		t.Fatalf("error = %v, want benchmark_rust_c_parity capability rejection", err)
	}
}

func TestValidateActorCapabilityRejectsNativeAppClaimUsingOldRealWindowProbe(t *testing.T) {
	err := validateCanonicalActorManifestWithReleaseNotes(
		t,
		actorV1ReleaseNotesBase()+
			"\nTetra v1 provides the native application platform through linux-x64 real-window probe evidence.\n",
	)
	if err == nil {
		t.Fatalf("expected native app claim using old real-window probe evidence to fail")
	}
	for _, want := range []string{"native_surface_host", "old real-window probe"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %v, want %q", err, want)
		}
	}
}

func TestValidateActorCapabilityRejectsReleaseContractClaimDrift(t *testing.T) {
	fixture := writeMinimalActorCapabilitiesFixture(t, minimalActorCapabilitiesOptions{
		ContractClaims: []string{"full Erlang/OTP actor runtime"},
	})
	err := validateActorCapabilitiesManifestFile(fixture.ManifestPath, fixture.Root)
	if err == nil {
		t.Fatalf("expected release contract claim drift rejection")
	}
	if !strings.Contains(err.Error(), "contract") || !strings.Contains(err.Error(), "claim") {
		t.Fatalf("error = %v, want contract claim drift", err)
	}
}

func TestValidateActorCapabilityRejectsReleaseContractTargetDrift(t *testing.T) {
	fixture := writeMinimalActorCapabilitiesFixture(t, minimalActorCapabilitiesOptions{
		ContractTarget: "macos-x64",
	})
	err := validateActorCapabilitiesManifestFile(fixture.ManifestPath, fixture.Root)
	if err == nil {
		t.Fatalf("expected release contract target drift rejection")
	}
	if !strings.Contains(err.Error(), "contract") || !strings.Contains(err.Error(), "target") {
		t.Fatalf("error = %v, want contract target drift", err)
	}
}

func writeActorCapabilitiesFixture(t *testing.T, raw string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "actor-capability-manifest.v1.json")
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

type minimalActorCapabilitiesFixture struct {
	Root             string
	ManifestPath     string
	ReleaseNotesPath string
}

type minimalActorCapabilitiesOptions struct {
	DocText          string
	GateText         string
	ReleaseNotesText string
	ContractClaims   []string
	ContractTarget   string
}

func writeMinimalActorCapabilitiesFixture(t *testing.T, options minimalActorCapabilitiesOptions) minimalActorCapabilitiesFixture {
	t.Helper()
	root := t.TempDir()
	requiredNonclaims := minimalRequiredNonclaims()
	docText := options.DocText
	if docText == "" {
		docText = minimalRequiredNonclaimsText()
	}
	gateText := options.GateText
	if gateText == "" {
		gateText = minimalRequiredNonclaimsText()
	}
	releaseNotesText := options.ReleaseNotesText
	if releaseNotesText == "" {
		releaseNotesText = minimalRequiredNonclaimsText() + "\nActor runtime foundation evidence."
	}
	contractClaims := options.ContractClaims
	if len(contractClaims) == 0 {
		contractClaims = []string{"linux-x64 scoped actor/task runtime foundation evidence"}
	}
	contractTarget := options.ContractTarget
	if contractTarget == "" {
		contractTarget = "linux-x64"
	}

	writeFixtureFile(t, root, "docs/actors.md", docText)
	writeFixtureFile(t, root, "gates/actor-gate.sh", gateText)
	writeFixtureFile(t, root, "release-notes.md", releaseNotesText)
	if err := os.MkdirAll(filepath.Join(root, "tools", "validate-actor-runtime-foundation"), 0o755); err != nil {
		t.Fatal(err)
	}

	contract := map[string]any{
		"schema":        "tetra.actor.release_contract.v1",
		"id":            "actor-runtime-foundation-linux-x64-contract",
		"capability_id": "actor_runtime_foundation_linux_x64",
		"target":        contractTarget,
		"scope":         contractTarget,
		"claims":        contractClaims,
		"nonclaims":     requiredNonclaims,
		"validators":    []string{"validate-actor-runtime-foundation"},
		"reports":       []string{"actor-runtime-foundation-manifest.json"},
		"gate_refs":     []string{"gates/actor-gate.sh"},
	}
	writeFixtureJSON(t, root, "contracts/actor-runtime-foundation-linux-x64.json", contract)

	manifest := map[string]any{
		"schema":  "tetra.actor.capability_manifest.v1",
		"profile": "actor-runtime-foundation",
		"required_capabilities": []any{
			map[string]any{
				"id":                "actor_runtime_foundation_linux_x64",
				"status":            "current_scoped",
				"supported_targets": []string{"linux-x64"},
				"release_note_terms": []string{
					"actor runtime foundation evidence",
				},
			},
		},
		"required_nonclaim_terms": []string{"erlang", "cluster", "reconnect", "retry", "non-linux", "zero-copy", "formal race"},
		"required_nonclaims":      requiredNonclaims,
		"forbidden_claims": []string{
			"full Erlang/OTP actor runtime",
			"cluster membership",
			"reconnect/retry production",
			"non-Linux distributed actor runtime",
			"distributed zero-copy pointer or region transfer",
			"formal race proof",
		},
		"docs_refs": []string{"docs/actors.md"},
		"validator_refs": []any{
			map[string]any{"id": "validate-actor-runtime-foundation", "path": "tools/validate-actor-runtime-foundation"},
		},
		"gate_refs": []any{
			map[string]any{"id": "actor-gate", "path": "gates/actor-gate.sh", "claim_check": true},
		},
		"contract_refs": []any{
			map[string]any{
				"id":                 "actor-runtime-foundation-linux-x64-contract",
				"path":               "contracts/actor-runtime-foundation-linux-x64.json",
				"capability_id":      "actor_runtime_foundation_linux_x64",
				"claim_check":        true,
				"nonclaim_check":     true,
				"target_check":       true,
				"validator_check":    true,
				"required_reports":   []string{"actor-runtime-foundation-manifest.json"},
				"required_gate_refs": []string{"gates/actor-gate.sh"},
			},
		},
		"capabilities": []any{
			map[string]any{
				"id":                "actor_runtime_foundation_linux_x64",
				"status":            "current_scoped",
				"supported_targets": []string{"linux-x64"},
				"claims":            []string{"linux-x64 scoped actor/task runtime foundation evidence"},
				"nonclaims":         requiredNonclaims,
				"evidence_refs":     []string{"gates/actor-gate.sh"},
				"validator_refs":    []string{"validate-actor-runtime-foundation"},
				"docs_refs":         []string{"docs/actors.md"},
			},
		},
	}
	writeFixtureJSON(t, root, "actor-capability-manifest.v1.json", manifest)
	return minimalActorCapabilitiesFixture{
		Root:             root,
		ManifestPath:     filepath.Join(root, "actor-capability-manifest.v1.json"),
		ReleaseNotesPath: filepath.Join(root, "release-notes.md"),
	}
}

func writeFixtureFile(t *testing.T, root, path, contents string) {
	t.Helper()
	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeFixtureJSON(t *testing.T, root, path string, value any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, root, path, string(raw))
}

func minimalRequiredNonclaims() []string {
	return []string{
		"no full Erlang/OTP actor runtime claim",
		"no cluster membership or reconnect/retry production claim",
		"no non-Linux distributed actor runtime support claim",
		"no distributed zero-copy pointer or region transfer claim",
		"no formal race proof claim",
	}
}

func minimalRequiredNonclaimsText() string {
	return strings.Join(minimalRequiredNonclaims(), "\n")
}

func actorCapabilitiesFixtureWithoutCapability(t *testing.T, id string) string {
	t.Helper()
	var manifest actorCapabilityManifest
	if err := json.Unmarshal([]byte(validActorCapabilitiesManifestJSON(t)), &manifest); err != nil {
		t.Fatal(err)
	}
	filtered := make([]actorCapability, 0, len(manifest.Capabilities))
	removed := false
	for _, cap := range manifest.Capabilities {
		if cap.ID == id {
			removed = true
			continue
		}
		filtered = append(filtered, cap)
	}
	if !removed {
		t.Fatalf("fixture capability %s not found", id)
	}
	manifest.Capabilities = filtered
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func validateCanonicalActorManifestWithReleaseNotes(t *testing.T, notesText string) error {
	t.Helper()
	root := repoRoot(t)
	manifest := filepath.Join(root, "docs", "contracts", "actors", "actor-capability-manifest.v1.json")
	notes := filepath.Join(t.TempDir(), "release-notes.md")
	if err := os.WriteFile(notes, []byte(notesText), 0o644); err != nil {
		t.Fatal(err)
	}
	return validateActorCapabilitiesManifestFileWithOptions(
		manifest,
		root,
		actorCapabilityValidationOptions{ReleaseNotes: []string{notes}},
	)
}

func actorV1ReleaseNotesBase() string {
	return strings.Join([]string{
		minimalRequiredNonclaimsText(),
		"actor runtime foundation evidence",
		"strict Linux-x64 gate",
		"actor-runtime-foundation-manifest.json",
		"distributed-actors-linux-x64/distributed-actors-linux-x64.json",
		"parallel-production-linux-x64/parallel-production-linux-x64.json",
	}, "\n")
}

func actorCapabilitiesFixtureWithRequiredCapability(t *testing.T, id string) string {
	t.Helper()
	var manifest map[string]any
	if err := json.Unmarshal([]byte(validActorCapabilitiesManifestJSON(t)), &manifest); err != nil {
		t.Fatal(err)
	}
	required, _ := manifest["required_capabilities"].([]any)
	required = append(required, map[string]any{
		"id":                id,
		"status":            "blocked",
		"supported_targets": []any{},
	})
	manifest["required_capabilities"] = required
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func actorCapabilitiesFixtureWithRequiredNonclaimTerm(t *testing.T, term string) string {
	t.Helper()
	var manifest map[string]any
	if err := json.Unmarshal([]byte(validActorCapabilitiesManifestJSON(t)), &manifest); err != nil {
		t.Fatal(err)
	}
	terms, _ := manifest["required_nonclaim_terms"].([]any)
	terms = append(terms, term)
	manifest["required_nonclaim_terms"] = terms
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func actorCapabilitiesFixtureWithTargets(t *testing.T, id string, targets []string) string {
	t.Helper()
	var manifest actorCapabilityManifest
	if err := json.Unmarshal([]byte(validActorCapabilitiesManifestJSON(t)), &manifest); err != nil {
		t.Fatal(err)
	}
	updated := false
	for index := range manifest.Capabilities {
		if manifest.Capabilities[index].ID == id {
			manifest.Capabilities[index].SupportedTargets = append([]string{}, targets...)
			updated = true
			break
		}
	}
	if !updated {
		t.Fatalf("fixture capability %s not found", id)
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		next := filepath.Dir(dir)
		if next == dir {
			t.Fatalf("repo root not found from %s", dir)
		}
		dir = next
	}
}

func validActorCapabilitiesManifestJSON(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "docs", "contracts", "actors", "actor-capability-manifest.v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
