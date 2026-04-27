# Async And Actors Guide

Status: user guide for the release actor/task surface.

The current runtime ABI details are documented in `docs/spec/runtime_abi.md`.
Actor behavior and supported targets are documented in `docs/spec/actors.md`.

## Tasks

The v1.0 scope requires the release task ABI to be documented and tested before
any final release label. If a task feature is still described as an MVP or
planned feature in the specs, treat it as a limited baseline until release gate
evidence says otherwise.

## Actors

Actor examples should be checked through the release smoke path instead of
manual inspection. Native host smoke is mandatory for `linux-x64`; target
build-only smoke is mandatory for other release targets.

## Verification

```sh
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
./tetra smoke --target linux-x64 --run=true --report /tmp/tetra-actors-smoke.json
```
