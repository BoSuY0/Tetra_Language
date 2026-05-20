# v0.2.x Maintenance Policy

Status: current policy for post-`v0.2.0` patch releases.

## Semver Policy

`v0.2.x` accepts compatible fixes only. Public CLI command behavior, stable
stdlib module names, report schemas, and documented target guarantees should
stay compatible unless a security fix requires an explicit exception.

## Patch Acceptance Criteria

- Fixes a user-visible bug, release artifact issue, validator defect, or
  documentation error.
- Includes a regression test or explicit verification command.
- Does not expand scope to deferred/post-v1 features.
- Keeps stabilization and release gates green.

## Security Patch Process

Security fixes may be fast-tracked, but still need focused verification,
release notes, and a signed review artifact for the exact commit.

## Backport Policy

Backports must remain minimal and traceable. Do not backport large feature
work under `v0.2.x`.

## Required Maintenance Checks

```sh
go test ./compiler/... ./cli/... ./tools/... -count=1
bash scripts/ci/test-all.sh --full --keep-going
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
```
