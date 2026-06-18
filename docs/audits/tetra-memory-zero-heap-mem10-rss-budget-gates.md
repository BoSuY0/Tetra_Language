# Tetra Memory Zero-Heap MEM-10 RSS Budget Gates

Date: 2026-06-16.
Status: complete for local RSS budget policy and validator gates.

## Scope

MEM-10 adds a local RSS budget gate for Tier 1 benchmark validation.

The gate is intentionally local. It does not claim that RSS is comparable across
machines, targets, kernels, loaders, libc versions, or benchmark environments.

## Implemented Evidence

- `docs/spec/local_rss_budget_policy.md`
  - defines `tetra.local_benchmark.rss_budget_policy.v1`;
  - requires pinned `target` and `host_profile`;
  - requires explicit nonclaims:
    - `local RSS budget only`;
    - `no cross-machine RSS claim`;
    - `no official benchmark claim`.
- `tools/cmd/validate-local-benchmark-tier1/main.go`
  - adds `--rss-budget-policy`;
  - adds `ValidateReportBytesWithRSSBudgetPolicy`;
  - validates policy schema, host profile, budget entries, and nonclaims;
  - applies budgets only when policy target and host profile match the report;
  - treats host or target mismatch as not applicable instead of failing another
    machine's report;
  - requires matching rows to be `measured` and backed by runtime-measured
    `rss_peak`;
  - compares the RSS sidecar `rss_peak_bytes` with the local budget plus
    allowed variance.
- `tools/cmd/validate-local-benchmark-tier1/main_test.go`
  - rejects `rss_peak` over local budget;
  - accepts `rss_peak` within budget;
  - does not fail when host profile differs;
  - rejects a policy that lacks required local RSS nonclaims.

## Budget Rule

When the policy applies, allowed RSS peak is:

```text
ceil(rss_peak_budget_bytes * (1 + allowed_variance_percent / 100))
```

The compared value is the process RSS sidecar field:

```text
rss_peak_bytes
```

The gate does not use heap, allocation-report, domain, binary-size, or Go
runtime memory data as substitutes.

## Read-Only Review Evidence

Allowed `explorer_fast` read-only review for MEM-10 found:

- RSS telemetry and Tier 1 RSS validation already existed;
- host metadata already existed in the Tier 1 report;
- no `local RSS budget` policy or validator gate existed before this patch;
- minimal implementation points were the policy schema, validator threshold,
  host/profile guard, and tests.

No file edits were delegated to a subagent.

## Verification

RED was confirmed before implementation:

```sh
GOCACHE=$(pwd)/.cache/go-build-rss-budget-red go test ./tools/cmd/validate-local-benchmark-tier1 -run 'RSS|Budget|Peak|Host|Policy' -count=1
```

Result: expected build failure because
`ValidateReportBytesWithRSSBudgetPolicy` did not exist yet.

Final focused checks:

```sh
GOCACHE=$(pwd)/.cache/go-build-rss-budget go test ./tools/cmd/validate-local-benchmark-tier1 -run 'RSS|Budget|Peak|Host|Policy' -count=1
GOCACHE=$(pwd)/.cache/go-build-rss-budget go test ./tools/cmd/validate-local-benchmark-tier1 -count=1
```

Result: passed.

Docs and workspace checks:

```sh
GOCACHE=$(pwd)/.cache/go-build-rss-budget-docs go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check -- docs/spec/local_rss_budget_policy.md tools/cmd/validate-local-benchmark-tier1/main.go tools/cmd/validate-local-benchmark-tier1/main_test.go
graphify update .
```

Result: passed.

Go caches used for MEM-10 were cleaned:

```sh
GOCACHE=$(pwd)/.cache/go-build-rss-budget-red go clean -cache
GOCACHE=$(pwd)/.cache/go-build-rss-budget go clean -cache
GOCACHE=$(pwd)/.cache/go-build-rss-budget-docs go clean -cache
```

## Nonclaims

- No cross-machine RSS comparison is claimed.
- No official benchmark result is claimed.
- No zero RSS claim is introduced.
- No heap optimization is claimed from RSS budget checks.
- No allocator performance superiority is claimed.
- No Linux RSS behavior is promoted into Tetra language semantics.
