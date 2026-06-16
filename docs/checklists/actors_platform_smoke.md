# Actors Platform Smoke Checklist

Date: 2025-12-30
Target version: linux-x64
Git HEAD:
Compiler version (compilerVersion): v0.6.0

## Actor Runtime Foundation Gate

Actor runtime foundation scoped release truth is
`tetra.actor.production_foundation.v1`, produced by
`scripts/release/post_v0_4/actor-runtime-foundation-linux-x64-gate.sh`.
The final evidence is uploaded from
`reports/actor-runtime-foundation/final/actor-runtime-foundation-manifest.json`,
`reports/actor-runtime-foundation/final/artifact-hashes.json`,
`distributed-actors-linux-x64/distributed-actors-linux-x64.json`, and
`parallel-production-linux-x64/parallel-production-linux-x64.json` through
`.github/workflows/ci.yml`; `.github/workflows/release-packages.yml` runs the
same gate before package upload, release publish, container publish, and
Homebrew tap update.

Windows and macOS rows below remain build smoke rows unless separate target-host
runtime evidence is added. Nonclaims: no full Erlang/OTP actor runtime claim,
no cluster membership or reconnect/retry production claim, no non-Linux
distributed actor runtime support claim, no distributed zero-copy pointer or
region transfer claim, and no formal race proof claim.

Actor failure/status boundary: actor entry functions that return zero or
nonzero both become the same user-visible `done` state; later local sends
return checked failure `-4`. There is no actor status, actor join,
actor exit-code, supervision, or restart API. Distributed
missing-node/node_down smoke evidence is checked status evidence only, with no
automatic retry, reconnect, restart, or supervision claim.

## Prereqs

- Go 1.20+
- `tetra` built: `bash scripts/dev/bootstrap.sh`

## Windows x64

Build:
- [x] `./tetra build --target windows-x64 -o actors_pingpong.exe examples/actors_pingpong.tetra`

Run:
- [ ] `./actors_pingpong.exe` (exit code 0)

Notes:

## macOS x64

Build:
- [x] `./tetra build --target macos-x64 -o actors_pingpong examples/actors_pingpong.tetra`

Run:
- [ ] `./actors_pingpong` (exit code 0)

Notes:

## Linux x64 (sanity)

Build:
- [x] `./tetra build --target linux-x64 -o actors_pingpong examples/actors_pingpong.tetra`
- [x] `./tetra build --target linux-x64 -o actors_tagged_stress examples/actors_tagged_stress.tetra`
- [x] `./tetra build --target linux-x64 -o task_bounded_stress examples/task_bounded_stress.tetra`

Run:
- [x] `./actors_pingpong` (exit code 0)
- [x] `./actors_tagged_stress` (exit code 0)
- [x] `./task_bounded_stress` (exit code 42)

Notes:
- Covered by `go test ./compiler/... ./cli/... -run "Async|Await|Task|Actor|Actors|Runtime|Selfhost|ABI|Ownership|Stress" -count=1`.
