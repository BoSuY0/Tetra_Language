package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const auditSchemaV1 = "tetra.surface.prod-audit.v1"
const auditLevelV1 = "surface-prod-final-same-commit-audit-v1"
const prodStableScopedVerdict = "PROD_STABLE_SCOPED"
const prodStableScopedScope = "PROD_STABLE_SCOPED_LINUX_WEB_APP_UI"
const currentHeadSentinel = "CURRENT_HEAD"

type validateAuditOptions struct {
	AuditPath      string
	ExpectedStatus string
	CurrentGitHead string
}

type surfaceProdAudit struct {
	Schema             string               `json:"schema"`
	Verdict            string               `json:"verdict"`
	Level              string               `json:"level"`
	Scope              string               `json:"scope"`
	ReleaseScope       string               `json:"release_scope"`
	GitHead            string               `json:"git_head"`
	GitDirty           bool                 `json:"git_dirty"`
	CleanCheckout      bool                 `json:"clean_checkout"`
	GeneratedAtUTC     string               `json:"generated_at_utc"`
	Blockers           []string             `json:"blockers"`
	Commands           []auditCommand       `json:"commands"`
	Reports            []auditReport        `json:"reports"`
	TargetHostEvidence []auditTarget        `json:"target_host_evidence"`
	ClaimGovernance    auditClaimGovernance `json:"claim_governance"`
}

type auditCommand struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Status  string `json:"status"`
}

type auditReport struct {
	Name                 string `json:"name"`
	Path                 string `json:"path"`
	Schema               string `json:"schema"`
	GitHead              string `json:"git_head"`
	SameCommit           bool   `json:"same_commit"`
	Required             bool   `json:"required"`
	ArtifactHashManifest string `json:"artifact_hash_manifest"`
}

type auditTarget struct {
	Target              string `json:"target"`
	Tier                string `json:"tier"`
	Evidence            string `json:"evidence"`
	Production          bool   `json:"production"`
	UnsupportedNonclaim bool   `json:"unsupported_nonclaim"`
}

type auditClaimGovernance struct {
	PublicClaimSource          string   `json:"public_claim_source"`
	ProdClaimValidator         string   `json:"prod_claim_validator"`
	FinalAuditValidator        string   `json:"final_audit_validator"`
	FakeClaimRejections        []string `json:"fake_claim_rejections"`
	UnsupportedTargetNonclaims []string `json:"unsupported_target_nonclaims"`
}

func main() {
	var opt validateAuditOptions
	flag.StringVar(&opt.AuditPath, "audit", "docs/release/surface_prod_release_audit.md", "Surface production release audit Markdown")
	flag.StringVar(&opt.ExpectedStatus, "expected-status", "", "expected final verdict, for example PROD_STABLE_SCOPED")
	flag.StringVar(&opt.CurrentGitHead, "current-git-head", "", "optional current git HEAD for same-commit validation")
	flag.Parse()
	if err := run(opt); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(opt validateAuditOptions) error {
	if strings.TrimSpace(opt.AuditPath) == "" {
		return errors.New("audit path is required")
	}
	raw, err := os.ReadFile(opt.AuditPath)
	if err != nil {
		return err
	}
	return validateAuditMarkdown(raw, opt.ExpectedStatus, opt.CurrentGitHead)
}

func validateAuditMarkdown(raw []byte, expectedStatus string, currentGitHead string) error {
	auditJSON, err := extractAuditJSON(raw)
	if err != nil {
		return err
	}
	var audit surfaceProdAudit
	dec := json.NewDecoder(bytes.NewReader(auditJSON))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&audit); err != nil {
		return fmt.Errorf("decode surface prod audit: %w", err)
	}
	resolvedGitHead, err := resolveAuditGitHeads(&audit, currentGitHead)
	if err != nil {
		return err
	}
	var issues []string
	issues = append(issues, validateAuditIdentity(audit, expectedStatus, resolvedGitHead)...)
	issues = append(issues, validateAuditCommands(audit)...)
	issues = append(issues, validateAuditReports(audit)...)
	issues = append(issues, validateAuditTargets(audit.TargetHostEvidence)...)
	issues = append(issues, validateAuditClaimGovernance(audit.ClaimGovernance)...)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	return nil
}

func resolveAuditGitHeads(audit *surfaceProdAudit, currentGitHead string) (string, error) {
	currentGitHead = strings.TrimSpace(currentGitHead)
	if !auditUsesCurrentHeadSentinel(*audit) {
		return currentGitHead, nil
	}
	if currentGitHead == "" {
		var err error
		currentGitHead, err = readCurrentGitHead()
		if err != nil {
			return "", err
		}
	}
	if !validGitHead(currentGitHead) {
		return "", fmt.Errorf("current git head %q must be a 40-character lowercase hex commit", currentGitHead)
	}
	if isCurrentHeadSentinel(audit.GitHead) {
		audit.GitHead = currentGitHead
	}
	for i := range audit.Reports {
		if isCurrentHeadSentinel(audit.Reports[i].GitHead) {
			audit.Reports[i].GitHead = currentGitHead
		}
	}
	return currentGitHead, nil
}

func auditUsesCurrentHeadSentinel(audit surfaceProdAudit) bool {
	if isCurrentHeadSentinel(audit.GitHead) {
		return true
	}
	for _, report := range audit.Reports {
		if isCurrentHeadSentinel(report.GitHead) {
			return true
		}
	}
	return false
}

func isCurrentHeadSentinel(value string) bool {
	return strings.TrimSpace(value) == currentHeadSentinel
}

func readCurrentGitHead() (string, error) {
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("resolve current git head: %w", err)
	}
	head := strings.TrimSpace(string(out))
	if !validGitHead(head) {
		return "", fmt.Errorf("git rev-parse HEAD returned invalid commit %q", head)
	}
	return head, nil
}

func extractAuditJSON(raw []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		return trimmed, nil
	}
	lines := strings.Split(string(raw), "\n")
	inBlock := false
	var out []string
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if !inBlock {
			if strings.HasPrefix(t, "```") && strings.Contains(t, "json") && strings.Contains(t, "surface-prod-audit") {
				inBlock = true
			}
			continue
		}
		if strings.HasPrefix(t, "```") {
			return []byte(strings.Join(out, "\n")), nil
		}
		out = append(out, line)
	}
	return nil, errors.New("missing fenced json surface-prod-audit block")
}

func validateAuditIdentity(audit surfaceProdAudit, expectedStatus string, currentGitHead string) []string {
	var issues []string
	if audit.Schema != auditSchemaV1 {
		issues = append(issues, fmt.Sprintf("schema is %q, want %q", audit.Schema, auditSchemaV1))
	}
	if audit.Level != auditLevelV1 {
		issues = append(issues, fmt.Sprintf("level is %q, want %q", audit.Level, auditLevelV1))
	}
	if audit.Scope != prodStableScopedScope {
		issues = append(issues, fmt.Sprintf("scope is %q, want %q", audit.Scope, prodStableScopedScope))
	}
	if audit.ReleaseScope != "surface-v1-linux-web" {
		issues = append(issues, fmt.Sprintf("release_scope is %q, want surface-v1-linux-web", audit.ReleaseScope))
	}
	if !validGitHead(audit.GitHead) {
		issues = append(issues, "git_head must be a 40-character lowercase hex commit")
	}
	if strings.TrimSpace(currentGitHead) != "" && audit.GitHead != strings.TrimSpace(currentGitHead) {
		issues = append(issues, fmt.Sprintf("git_head %s does not match current git head %s", audit.GitHead, strings.TrimSpace(currentGitHead)))
	}
	if _, err := time.Parse(time.RFC3339, audit.GeneratedAtUTC); err != nil {
		issues = append(issues, fmt.Sprintf("generated_at_utc must be RFC3339: %v", err))
	}
	if !validVerdict(audit.Verdict) {
		issues = append(issues, fmt.Sprintf("verdict %q is not a known final audit verdict", audit.Verdict))
	}
	if strings.TrimSpace(expectedStatus) != "" && audit.Verdict != strings.TrimSpace(expectedStatus) {
		issues = append(issues, fmt.Sprintf("expected status %s, got %s", strings.TrimSpace(expectedStatus), audit.Verdict))
	}
	if audit.Verdict == prodStableScopedVerdict {
		if audit.GitDirty || !audit.CleanCheckout {
			issues = append(issues, "dirty checkout cannot be promoted to PROD_STABLE_SCOPED")
		}
		if len(audit.Blockers) != 0 {
			issues = append(issues, "PROD_STABLE_SCOPED audit must not contain blockers")
		}
	} else if len(audit.Blockers) == 0 {
		issues = append(issues, "non-PROD_STABLE_SCOPED audit must explain blockers")
	}
	return issues
}

func validateAuditCommands(audit surfaceProdAudit) []string {
	required := map[string]string{
		"git-head":                 "git rev-parse HEAD",
		"git-status":               "git status --short",
		"full-go-test":             "go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -count=1",
		"ci-test":                  "bash scripts/ci/test.sh",
		"prod-gate":                "scripts/release/surface/prod-gate.sh",
		"validate-manifest":        "validate-manifest --manifest docs/generated/manifest.json",
		"verify-docs":              "verify-docs --manifest docs/generated/manifest.json",
		"validate-prod-audit":      "validate-surface-prod-audit --audit docs/release/surface_prod_release_audit.md --expected-status PROD_STABLE_SCOPED",
		"validate-artifact-hashes": "validate-artifact-hashes --manifest reports/surface-prod/final/artifact-hashes.json",
		"diff-check":               "git diff --check",
		"manifest-clean":           "git diff --exit-code -- docs/generated/manifest.json",
	}
	var issues []string
	seen := map[string]auditCommand{}
	for _, command := range audit.Commands {
		name := strings.TrimSpace(command.Name)
		if name == "" || strings.TrimSpace(command.Command) == "" || strings.TrimSpace(command.Status) == "" {
			issues = append(issues, "audit commands require name, command, and status")
			continue
		}
		if _, ok := seen[name]; ok {
			issues = append(issues, fmt.Sprintf("duplicate audit command %s", name))
		}
		seen[name] = command
		if strings.Contains(command.Command, "|| true") || strings.Contains(command.Command, "continue-on-error") {
			issues = append(issues, fmt.Sprintf("audit command %s contains bypass marker", name))
		}
		if audit.Verdict == prodStableScopedVerdict && command.Status != "pass" {
			issues = append(issues, fmt.Sprintf("PROD_STABLE_SCOPED command %s status is %q, want pass", name, command.Status))
		}
	}
	for name, fragment := range required {
		command, ok := seen[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing audit command %s", name))
			continue
		}
		if !strings.Contains(command.Command, fragment) {
			issues = append(issues, fmt.Sprintf("audit command %s missing %q", name, fragment))
		}
	}
	return issues
}

func validateAuditReports(audit surfaceProdAudit) []string {
	required := map[string]string{
		"surface-prod-gate":   "tetra.surface.prod-gate-report.v1",
		"block-system":        "tetra.surface.block-system.gate.v1",
		"morph":               "tetra.surface.morph.gate.v1",
		"surface-release":     "tetra.surface.release.v1",
		"visual":              "tetra.surface.visual-regression.v1",
		"perf":                "tetra.surface.perf-report.v1",
		"security":            "tetra.surface.security-report.v1",
		"package":             "tetra.surface.package-report.v1",
		"safe-view-lifetime":  "tetra.safe-view-lifetime.gate.v1",
		"api-stability":       "tetra.surface.api-stability.v1",
		"electron-comparison": "tetra.surface.electron-comparison-report.v1",
		"artifact-hashes":     "tetra.release-artifact-hashes.v1alpha1",
	}
	var issues []string
	seen := map[string]auditReport{}
	for _, report := range audit.Reports {
		name := strings.TrimSpace(report.Name)
		if name == "" || strings.TrimSpace(report.Path) == "" || strings.TrimSpace(report.Schema) == "" {
			issues = append(issues, "audit reports require name, path, and schema")
			continue
		}
		if _, ok := seen[name]; ok {
			issues = append(issues, fmt.Sprintf("duplicate audit report %s", name))
		}
		seen[name] = report
		if report.Required && (!report.SameCommit || report.GitHead != audit.GitHead) {
			issues = append(issues, fmt.Sprintf("required report %s is not same commit as audit", name))
		}
		if report.Required && strings.TrimSpace(report.ArtifactHashManifest) == "" {
			issues = append(issues, fmt.Sprintf("required report %s missing artifact hash manifest", name))
		}
	}
	for name, schema := range required {
		report, ok := seen[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing required audit report %s", name))
			continue
		}
		if !report.Required {
			issues = append(issues, fmt.Sprintf("audit report %s must be required", name))
		}
		if report.Schema != schema {
			issues = append(issues, fmt.Sprintf("audit report %s schema is %q, want %q", name, report.Schema, schema))
		}
	}
	return issues
}

func validateAuditTargets(targets []auditTarget) []string {
	required := map[string]struct {
		tier       string
		production bool
		nonclaim   bool
	}{
		"headless":    {tier: "release-evidence", production: false, nonclaim: false},
		"linux-x64":   {tier: "prod", production: true, nonclaim: false},
		"wasm32-web":  {tier: "prod", production: true, nonclaim: false},
		"windows-x64": {tier: "beta", production: false, nonclaim: true},
		"macos-x64":   {tier: "beta", production: false, nonclaim: true},
		"wasm32-wasi": {tier: "unsupported", production: false, nonclaim: true},
	}
	var issues []string
	seen := map[string]auditTarget{}
	for _, target := range targets {
		seen[target.Target] = target
	}
	for name, want := range required {
		got, ok := seen[name]
		if !ok {
			issues = append(issues, fmt.Sprintf("missing target-host evidence %s", name))
			continue
		}
		if got.Tier != want.tier || got.Production != want.production || got.UnsupportedNonclaim != want.nonclaim {
			issues = append(issues, fmt.Sprintf("target %s tier/nonclaim mismatch", name))
		}
		if strings.TrimSpace(got.Evidence) == "" {
			issues = append(issues, fmt.Sprintf("target %s evidence is required", name))
		}
	}
	return issues
}

func validateAuditClaimGovernance(governance auditClaimGovernance) []string {
	var issues []string
	for _, check := range []struct {
		name string
		got  string
		want string
	}{
		{name: "public_claim_source", got: governance.PublicClaimSource, want: "docs/release/surface_prod_release_audit.md"},
		{name: "prod_claim_validator", got: governance.ProdClaimValidator, want: "tools/cmd/validate-surface-prod-claim"},
		{name: "final_audit_validator", got: governance.FinalAuditValidator, want: "tools/cmd/validate-surface-prod-audit"},
	} {
		if check.got != check.want {
			issues = append(issues, fmt.Sprintf("claim governance %s is %q, want %q", check.name, check.got, check.want))
		}
	}
	for _, want := range []string{
		"fake electron/react/css replacement rejected",
		"fake cross-platform support rejected",
		"fake gpu production claim rejected",
		"fake full accessibility parity rejected",
		"missing target-host evidence rejected",
	} {
		if !contains(governance.FakeClaimRejections, want) {
			issues = append(issues, fmt.Sprintf("claim governance missing fake-claim rejection %q", want))
		}
	}
	for _, want := range []string{"windows-x64", "macos-x64", "wasm32-wasi", "GPU production", "full accessibility parity", "broad Electron replacement"} {
		if !contains(governance.UnsupportedTargetNonclaims, want) {
			issues = append(issues, fmt.Sprintf("claim governance missing unsupported nonclaim %q", want))
		}
	}
	return issues
}

func validVerdict(value string) bool {
	switch value {
	case prodStableScopedVerdict, "NEAR_READY_WITH_BLOCKERS", "BETA_ONLY", "EXPERIMENTAL_ONLY", "FAIL":
		return true
	default:
		return false
	}
}

func validGitHead(value string) bool {
	if len(value) != 40 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && strings.ToLower(value) == value
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
