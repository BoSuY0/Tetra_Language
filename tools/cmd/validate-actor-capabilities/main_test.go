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

func writeActorCapabilitiesFixture(t *testing.T, raw string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "actor-capability-manifest.v1.json")
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
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
