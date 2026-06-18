# Local RSS Budget Policy

Status: local Tier 1 benchmark validator policy.

This policy lets `tools/cmd/validate-local-benchmark-tier1` fail a local benchmark report when a
measured Tetra row exceeds a pinned process RSS peak budget. It is a local regression gate, not a
portable memory claim.

## Boundary

RSS budget evidence uses the process RSS telemetry contract in
`docs/spec/telemetry/process_rss_telemetry.md`.

The validator compares only:

```text
memory_evidence.rss_peak
```

It does not compare:

```text
heap_alloc_bytes
bytes_requested
bytes_reserved
bytes_committed
bytes_copied
domain_bytes
binary_size_bytes
Go runtime MemStats
allocation report estimates
```

Heap and RSS remain separate metrics. A row can pass a zero-heap gate and still have non-zero RSS
because every OS process has loader, stack, page table, code, runtime, and library footprint.

## Policy Schema

Local RSS budget policy files use:

```text
tetra.local_benchmark.rss_budget_policy.v1
```

Example:

```json
{
  "schema": "tetra.local_benchmark.rss_budget_policy.v1",
  "target": "linux-x64",
  "host_profile": {
    "goos": "linux",
    "goarch": "amd64",
    "cpus": 8,
    "target_cpu": "example cpu"
  },
  "budgets": [
    {
      "category": "integer loops",
      "language": "tetra",
      "rss_peak_budget_bytes": 8192,
      "allowed_variance_percent": 0,
      "reason": "local baseline guard for simple integer loop"
    }
  ],
  "non_claims": [
    "local RSS budget only",
    "no cross-machine RSS claim",
    "no official benchmark claim"
  ]
}
```

## Required Fields

- `schema`: must be `tetra.local_benchmark.rss_budget_policy.v1`.
- `target`: target triple for the local report, for example `linux-x64`.
- `host_profile.goos`: report host OS.
- `host_profile.goarch`: report host architecture.
- `host_profile.cpus`: report CPU count.
- `host_profile.target_cpu`: report target CPU description.
- `host_profile.git_commit`: optional exact report commit pin.
- `budgets`: one or more local budget entries.
- `budgets[].category`: Tier 1 benchmark category.
- `budgets[].language`: benchmark row language; omitted means `tetra`.
- `budgets[].rss_peak_budget_bytes`: positive RSS peak budget in bytes.
- `budgets[].allowed_variance_percent`: non-negative local tolerance.
- `budgets[].reason`: short reason for the budget value.
- `non_claims`: must include the required nonclaims below.

Required nonclaims:

```text
local RSS budget only
no cross-machine RSS claim
no official benchmark claim
```

## Validator Rule

The validator applies a budget only when the policy target and host profile match the report.

If the target or host profile differs, the budget check is treated as not applicable for that
report. This prevents a budget recorded on one machine from failing or passing another machine's
report as if RSS were portable.

When the policy applies:

- the matching category/language row must exist;
- the row must be `measured`;
- the row must have Tetra memory evidence;
- `rss_peak.evidence_class` must be `runtime_measured`;
- `rss_peak.method` must be `linux_wait4_rusage_maxrss_v1`;
- `rss_peak.source_artifact` must point to a valid process RSS sidecar;
- the sidecar `rss_peak_bytes` must be less than or equal to the budget plus allowed variance.

Allowed peak is:

```text
ceil(rss_peak_budget_bytes * (1 + allowed_variance_percent / 100))
```

## CLI Usage

Run the normal Tier 1 validator:

```sh
GOCACHE=$(pwd)/.cache/go-build-rss-budget go run ./tools/cmd/validate-local-benchmark-tier1 \
  --report reports/local-benchmark-tier1-v1/report.json
```

Run with a local RSS budget policy:

```sh
GOCACHE=$(pwd)/.cache/go-build-rss-budget go run ./tools/cmd/validate-local-benchmark-tier1 \
  --report reports/local-benchmark-tier1-v1/report.json \
  --rss-budget-policy docs/spec/local-rss-budget-policy.local.json
```

The policy file should stay local unless its host profile and nonclaims are intentionally part of a
checked-in evidence artifact.

## Nonclaims

This policy does not claim:

- cross-machine RSS comparability;
- official benchmark status;
- zero RSS;
- zero heap;
- allocator superiority;
- production OS memory footprint;
- Linux RSS behavior as Tetra language semantics.

It only says that a pinned local report exceeded or stayed within a pinned local process RSS peak
budget.
