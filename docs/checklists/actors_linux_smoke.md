# Actors Linux Smoke Checklist

> **Deprecated:** use `docs/checklists/actors_platform_smoke.md` (this file is kept for historical
reference).

Actor runtime foundation scoped release truth is now the Linux-x64 gate
`scripts/release/post_v0_4/actor-runtime-foundation-linux-x64-gate.sh`, which
writes `tetra.actor.production_foundation.v1` evidence under
`reports/actor-runtime-foundation/final/`. CI and package publishing wire that
gate through `.github/workflows/ci.yml` and `.github/workflows/release-packages.yml`.

Nonclaims: no full Erlang/OTP actor runtime claim, no cluster membership or
reconnect/retry production claim, no non-Linux distributed actor runtime support
claim, no distributed zero-copy pointer or region transfer claim, and no formal
race proof claim.

Date:
Target version:
Git HEAD:
Compiler version (compilerVersion):

## Prereqs

- Linux x64 host
- `tetra` built: `bash scripts/dev/bootstrap.sh`

## Build

- [ ] `./tetra build --target linux-x64 -o actors_pingpong examples/actors_pingpong.tetra`

## Run

- [ ] `./actors_pingpong` (exit code 0)

## Notes
