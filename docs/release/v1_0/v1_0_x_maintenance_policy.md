# v1.0.x Maintenance Policy

Status: future post-release policy for patch releases after `v1.0.0`.

## Semver Policy

`v1.0.x` releases accept compatible fixes only. Public syntax, CLI flags,
stable stdlib APIs, report schemas, and runtime ABI behavior must remain
backward compatible unless a security fix requires a documented exception.

## Patch Acceptance Criteria

- Fixes a user-visible bug, security issue, release artifact issue, or
  documentation error.
- Includes a regression test or documented verification command.
- Does not expand the v1.0 feature scope.
- Keeps `bash scripts/ci/test-all.sh --full --keep-going` green.

## Security Patch Process

Security fixes may bypass normal batching, but still need focused tests,
release notes, and a post-fix gate run. If the fix changes behavior, document
the compatibility impact and mitigation.

## Deprecation Policy

Deprecations can be announced in `v1.0.x`, but removals wait for a later minor
or major line unless required for security. Every deprecation needs a
replacement path and diagnostics or documentation.

## Backport Policy

Backports must be minimal and traceable to the main development line. Do not
backport new syntax, new targets, or beta ecosystem features into `v1.0.x`.

## Gate Cadence

Nightly or maintenance-branch automation should run:

```sh
go test ./compiler/... ./cli/... ./tools/... -count=1
bash scripts/ci/test-all.sh --full --keep-going
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

Patch release candidates will run the future v1.0 gate after
`docs/checklists/v1_0_release_gate.md` is replaced with a real v1.0 checklist.

## Known Issues

Use `docs/release/known_issues_template.md` for every patch-cycle known issue
list. A `v1.1` roadmap should not start until the current known issues list is
triaged.
