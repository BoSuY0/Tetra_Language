package main

import (
	"fmt"
	"strings"
)

const (
	claimTierSchemaV1  = "tetra.performance.claim_tiers.v1"
	scopeP20ClaimTiers = "p20.2_claim_tiers"
	claimTierGenerated = "2026-06-03T00:00:00Z"
)

type ClaimTierPolicy struct {
	ID                      string   `json:"id"`
	Rank                    int      `json:"rank"`
	Label                   string   `json:"label"`
	RequiredEvidenceClasses []string `json:"required_evidence_classes"`
	AllowedWording          []string `json:"allowed_wording"`
	Boundary                string   `json:"boundary"`
}

type ClaimTierEvidence struct {
	Class       string `json:"class"`
	Artifact    string `json:"artifact"`
	Description string `json:"description"`
}

type PublicPerformanceClaim struct {
	ID       string              `json:"id"`
	Tier     string              `json:"tier"`
	Text     string              `json:"text"`
	Evidence []ClaimTierEvidence `json:"evidence"`
}

type ClaimTierReport struct {
	Schema    string                   `json:"schema"`
	Scope     string                   `json:"scope"`
	Generated string                   `json:"generated"`
	Policies  []ClaimTierPolicy        `json:"policies"`
	Claims    []PublicPerformanceClaim `json:"claims"`
	NonClaims []string                 `json:"non_claims"`
}

func P20ClaimTierPolicies() []ClaimTierPolicy {
	return copyClaimTierPolicies(p20ClaimTierPolicies)
}

func BuildP20ClaimTierReport() ClaimTierReport {
	return ClaimTierReport{
		Schema:    claimTierSchemaV1,
		Scope:     scopeP20ClaimTiers,
		Generated: claimTierGenerated,
		Policies:  P20ClaimTierPolicies(),
		Claims: []PublicPerformanceClaim{
			{
				ID:   "p20_current_local_smoke_only",
				Tier: "tier0_local_smoke_only",
				Text: ("Current P20.0/P20.1 evidence is local smoke only: a dry-run " +
					"benchmark matrix contract and performance-blocker explanation exist; no " +
					"measured speed, no C++/Rust parity, no official benchmark, and no " +
					"official TechEmpower result is claimed."),
				Evidence: []ClaimTierEvidence{
					{
						Class:       "local_smoke",
						Artifact:    "tools/cmd/truth-bench-harness",
						Description: "claim validation and dry-run harness smoke policy",
					},
					{
						Class: "dry_run_matrix",
						Artifact: ("reports/benchmark-matrix-hardening-v1/benchmarks/p20-matrix-" +
							"hardening-report.json"),
						Description: "P20.0 matrix contract report with ran=false rows",
					},
					{
						Class: "performance_blocker_report",
						Artifact: ("reports/benchmark-matrix-hardening-v1/benchmarks/artifacts/p20-" +
							"matrix-hardening.perf.json"),
						Description: "P20.1 blocker explanation report for P20.0 Tetra rows",
					},
				},
			},
			{
				ID:   "p15_actor_benchmark_prep_tier0",
				Tier: "tier0_local_smoke_only",
				Text: ("Current ACTOR-P15 actor benchmark evidence is Tier 0 local " +
					"smoke/preparation only: harness rows exist for actor ping-pong, fanout/" +
					"fanin, mailbox throughput, backpressure latency, and zero_copy_move " +
					"local typed mailbox with raw artifact references; no measured speed, no " +
					"production throughput guarantee, no official benchmark, no C++/Rust " +
					"parity, and no distributed zero-copy claim is made."),
				Evidence: []ClaimTierEvidence{
					{
						Class:    "local_smoke",
						Artifact: "tools/cmd/parallel-production-smoke",
						Description: ("parallel production smoke emits actor benchmark prep rows as " +
							"dry-run Tier 0 evidence"),
					},
					{
						Class:    "dry_run_matrix",
						Artifact: "tools/cmd/truth-bench-harness",
						Description: ("P15 actor benchmark prep scope requires raw artifact references " +
							"and rejects overclaims"),
					},
				},
			},
		},
		NonClaims: []string{
			"measured speed",
			"C++/Rust parity",
			"official benchmark",
			"official TechEmpower",
			"cross-machine",
			"independent reproduced",
			"throughput advantage",
			"latency advantage",
			"production throughput guarantee",
			"distributed zero-copy",
			"actor benchmark superiority",
		},
	}
}

func ValidateClaimTierReport(report ClaimTierReport) error {
	if report.Schema != claimTierSchemaV1 {
		return fmt.Errorf("unsupported claim-tier schema %q", report.Schema)
	}
	if report.Scope != scopeP20ClaimTiers {
		return fmt.Errorf("unsupported claim-tier scope %q", report.Scope)
	}
	if strings.TrimSpace(report.Generated) == "" {
		return fmt.Errorf("claim-tier generated timestamp is required")
	}
	policies, err := validateClaimTierPolicies(report.Policies)
	if err != nil {
		return err
	}
	if len(report.Claims) == 0 {
		return fmt.Errorf("claim-tier report must contain at least one public claim")
	}
	for _, claim := range report.Claims {
		if err := validatePublicPerformanceClaim(claim, policies); err != nil {
			return err
		}
	}
	for _, nonClaim := range report.NonClaims {
		if isWeakClaimTierText(nonClaim) {
			return fmt.Errorf("claim-tier non-claim contains placeholder text: %q", nonClaim)
		}
	}
	for _, want := range []string{
		"measured speed",
		"C++/Rust parity",
		"official benchmark",
		"official TechEmpower",
		"cross-machine",
		"independent reproduced",
	} {
		if !stringSliceHas(report.NonClaims, want) {
			return fmt.Errorf("claim-tier report missing non-claim %q", want)
		}
	}
	return nil
}

func validatePublicPerformanceClaim(
	claim PublicPerformanceClaim,
	policies map[string]ClaimTierPolicy,
) error {
	if isWeakClaimTierText(claim.ID) {
		return fmt.Errorf("claim-tier public claim has placeholder id: %q", claim.ID)
	}
	if isWeakClaimTierText(claim.Text) {
		return fmt.Errorf("claim-tier public claim %q has placeholder text", claim.ID)
	}
	policy, ok := policies[claim.Tier]
	if !ok {
		return fmt.Errorf("claim-tier public claim %q uses unknown tier %q", claim.ID, claim.Tier)
	}
	if err := validatePerformanceClaimTextForTier(claim.Text, policy.Rank); err != nil {
		return err
	}
	if len(claim.Evidence) == 0 {
		return fmt.Errorf("claim-tier public claim %q missing evidence", claim.ID)
	}
	for _, evidence := range claim.Evidence {
		if isWeakClaimTierText(evidence.Class) || isWeakClaimTierText(evidence.Artifact) ||
			isWeakClaimTierText(evidence.Description) {
			return fmt.Errorf(
				"claim-tier public claim %q contains placeholder evidence: %+v",
				claim.ID,
				evidence,
			)
		}
	}
	for _, required := range policy.RequiredEvidenceClasses {
		if !publicClaimHasEvidenceClass(claim, required) {
			return fmt.Errorf(
				"claim-tier public claim %q at %s missing required evidence class %q",
				claim.ID,
				policy.Label,
				required,
			)
		}
	}
	return nil
}

func validateClaimTierPolicies(policies []ClaimTierPolicy) (map[string]ClaimTierPolicy, error) {
	if len(policies) != len(p20ClaimTierPolicies) {
		return nil, fmt.Errorf(
			"claim-tier policy count = %d, want %d",
			len(policies),
			len(p20ClaimTierPolicies),
		)
	}
	seen := map[string]ClaimTierPolicy{}
	for _, policy := range policies {
		if _, ok := seen[policy.ID]; ok {
			return nil, fmt.Errorf("duplicate claim-tier policy %q", policy.ID)
		}
		seen[policy.ID] = policy
	}
	for _, want := range p20ClaimTierPolicies {
		got, ok := seen[want.ID]
		if !ok {
			return nil, fmt.Errorf("missing claim-tier policy %q", want.ID)
		}
		if got.Rank != want.Rank || got.Label != want.Label {
			return nil, fmt.Errorf(
				"claim-tier policy %q rank/label drift: %d/%q",
				want.ID,
				got.Rank,
				got.Label,
			)
		}
		for _, required := range want.RequiredEvidenceClasses {
			if !stringSliceHas(got.RequiredEvidenceClasses, required) {
				return nil, fmt.Errorf(
					"claim-tier policy %q missing required evidence class %q",
					want.ID,
					required,
				)
			}
		}
		if isWeakClaimTierText(got.Boundary) {
			return nil, fmt.Errorf("claim-tier policy %q has placeholder boundary", want.ID)
		}
	}
	return seen, nil
}

func validatePerformanceClaimTextForTier(text string, maxTier int) error {
	lower := strings.ToLower(text)
	if isWeakClaimTierText(text) {
		return fmt.Errorf("claim text contains placeholder wording: %q", text)
	}
	if tier, phrase := impliedPerformanceClaimTier(lower); tier > maxTier {
		return fmt.Errorf(
			"claim wording implies tier %d performance evidence via %q but declared tier allows tier %d: %s",
			tier,
			phrase,
			maxTier,
			text,
		)
	}
	switch {
	case containsUnsafeClaimPhrase(lower, "fastest language"):
		return fmt.Errorf("forbidden fastest language claim: %s", text)
	case containsUnsafeClaimPhrase(lower, "tetra is the fastest"):
		return fmt.Errorf("forbidden broad performance claim: %s", text)
	case containsForbiddenCPlusPlusRustParityClaim(lower):
		return fmt.Errorf("forbidden C++/Rust parity claim: %s", text)
	case containsUnsafeClaimPhrase(lower, "production database benchmark"):
		return fmt.Errorf(
			"forbidden production database benchmark claim without measured evidence: %s",
			text,
		)
	}
	if err := validateActorBenchmarkClaimText(lower, text); err != nil {
		return err
	}
	return nil
}

func validateActorBenchmarkClaimText(lower string, text string) error {
	actorBenchmarkContext := strings.Contains(lower, "actor") ||
		strings.Contains(lower, "mailbox") ||
		strings.Contains(lower, "zero_copy_move")
	if actorBenchmarkContext {
		for _, phrase := range []string{
			"actor benchmark superiority",
			"performance superiority",
			"faster than",
			"outperforms",
			"beats rust",
			"beats go",
			"beats erlang",
			"production throughput guarantee",
			"real-world sla",
		} {
			if containsUnsafeClaimPhrase(lower, phrase) {
				return fmt.Errorf("forbidden actor benchmark claim via %q: %s", phrase, text)
			}
		}
	}
	if strings.Contains(lower, "zero_copy_move") {
		for _, phrase := range []string{
			"production runtime",
			"distributed zero-copy",
			"network zero-copy",
			"cross-machine zero-copy",
		} {
			if containsUnsafeClaimPhrase(lower, phrase) {
				return fmt.Errorf(
					"forbidden zero_copy_move production claim via %q: %s",
					phrase,
					text,
				)
			}
		}
	}
	return nil
}

func impliedPerformanceClaimTier(lower string) (int, string) {
	if isExplicitNonClaimSentence(lower) {
		return 0, ""
	}
	for _, phrase := range []string{
		"official upstream benchmark submission",
		"official upstream benchmark",
		"upstream benchmark submission",
		"official techempower",
		"official benchmark",
	} {
		if containsUnsafeClaimPhrase(lower, phrase) {
			return 4, phrase
		}
	}
	for _, phrase := range []string{
		"independently reproduced",
		"independent reproduced",
		"independent reproduction",
		"third-party reproduced",
		"third party reproduced",
	} {
		if containsUnsafeClaimPhrase(lower, phrase) {
			return 3, phrase
		}
	}
	for _, phrase := range []string{
		"reproducible cross-machine",
		"cross-machine benchmark",
		"cross machine benchmark",
		"cross-machine reproduction",
		"multi-machine reproducible",
	} {
		if containsUnsafeClaimPhrase(lower, phrase) {
			return 2, phrase
		}
	}
	for _, phrase := range []string{
		"local benchmark evidence",
		"measured local benchmark",
		"local benchmark result",
		"local benchmark measurement",
		"measured benchmark result",
		"measured speed",
		"measured performance",
		"throughput advantage",
		"latency advantage",
		"speed advantage",
	} {
		if containsUnsafeClaimPhrase(lower, phrase) {
			return 1, phrase
		}
	}
	return 0, ""
}

func containsUnsafeClaimPhrase(lower string, phrase string) bool {
	start := 0
	for {
		idx := strings.Index(lower[start:], phrase)
		if idx < 0 {
			return false
		}
		idx += start
		if !claimPhraseContextIsSafeNonClaim(lower, idx, phrase) {
			return true
		}
		start = idx + len(phrase)
	}
}

func claimPhraseContextIsSafeNonClaim(lower string, idx int, phrase string) bool {
	if isExplicitNonClaimSentence(lower) {
		return true
	}
	prefixStart := idx - 48
	if prefixStart < 0 {
		prefixStart = 0
	}
	prefix := strings.TrimSpace(lower[prefixStart:idx])
	for _, safePrefix := range []string{"no", "not", "without"} {
		if strings.HasSuffix(prefix, safePrefix) {
			return true
		}
	}
	for _, safeBefore := range []string{
		"does not claim",
		"not claimed",
		"not proven",
		"not implied",
		"without claiming",
	} {
		if strings.Contains(prefix, safeBefore) {
			return true
		}
	}
	suffixEnd := idx + len(phrase) + 80
	if suffixEnd > len(lower) {
		suffixEnd = len(lower)
	}
	suffix := lower[idx:suffixEnd]
	for _, safeAfter := range []string{"not claimed", "not proven", "not implied", "is not claimed"} {
		if strings.Contains(suffix, safeAfter) {
			return true
		}
	}
	return false
}

func isExplicitNonClaimSentence(lower string) bool {
	trimmed := strings.TrimSpace(lower)
	return strings.HasPrefix(trimmed, "no ") && strings.Contains(trimmed, " is claimed")
}

func isWeakClaimTierText(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return true
	}
	lower := strings.ToLower(trimmed)
	for _, weak := range []string{"todo", "tbd", "placeholder", "fixme"} {
		if lower == weak || strings.Contains(lower, weak) {
			return true
		}
	}
	return false
}

func publicClaimHasEvidenceClass(claim PublicPerformanceClaim, class string) bool {
	for _, evidence := range claim.Evidence {
		if evidence.Class == class {
			return true
		}
	}
	return false
}

func stringSliceHas(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func copyClaimTierPolicies(in []ClaimTierPolicy) []ClaimTierPolicy {
	out := make([]ClaimTierPolicy, len(in))
	for i, policy := range in {
		out[i] = policy
		out[i].RequiredEvidenceClasses = append([]string(nil), policy.RequiredEvidenceClasses...)
		out[i].AllowedWording = append([]string(nil), policy.AllowedWording...)
	}
	return out
}

var p20ClaimTierPolicies = []ClaimTierPolicy{
	{
		ID:                      "tier0_local_smoke_only",
		Rank:                    0,
		Label:                   "Tier 0: local smoke only",
		RequiredEvidenceClasses: []string{"local_smoke"},
		AllowedWording: []string{
			"local smoke only",
			"dry-run benchmark matrix",
			"performance-blocker explanation",
			"no measured speed",
		},
		Boundary: ("local smoke, dry-run matrix, and explanation evidence only; no " +
			"measured benchmark result or external comparison wording"),
	},
	{
		ID:                      "tier1_local_benchmark_evidence",
		Rank:                    1,
		Label:                   "Tier 1: local benchmark evidence",
		RequiredEvidenceClasses: []string{"local_benchmark"},
		AllowedWording:          []string{"local benchmark evidence", "local benchmark result"},
		Boundary: ("local measured benchmark wording requires local benchmark " +
			"evidence tied to exact commands, host, raw output, and report artifacts"),
	},
	{
		ID:                      "tier2_reproducible_cross_machine_benchmark",
		Rank:                    2,
		Label:                   "Tier 2: reproducible cross-machine benchmark",
		RequiredEvidenceClasses: []string{"cross_machine_reproduction"},
		AllowedWording: []string{
			"reproducible cross-machine benchmark",
			"cross-machine reproduction",
		},
		Boundary: ("cross-machine wording requires reproduced benchmark evidence " +
			"across machines with pinned inputs and comparable artifacts"),
	},
	{
		ID:                      "tier3_independent_reproduced_benchmark",
		Rank:                    3,
		Label:                   "Tier 3: independent reproduced benchmark",
		RequiredEvidenceClasses: []string{"independent_reproduction"},
		AllowedWording: []string{
			"independent reproduced benchmark",
			"independently reproduced benchmark",
		},
		Boundary: ("independent reproduction wording requires third-party " +
			"reproduction evidence, not local-only project artifacts"),
	},
	{
		ID:                      "tier4_official_upstream_benchmark_submission",
		Rank:                    4,
		Label:                   "Tier 4: official upstream benchmark submission",
		RequiredEvidenceClasses: []string{"official_upstream_submission"},
		AllowedWording: []string{
			"official upstream benchmark submission",
			"official benchmark result",
		},
		Boundary: ("official wording requires accepted upstream benchmark " +
			"submission evidence and must not be inferred from local or dry-run " +
			"artifacts"),
	},
}
