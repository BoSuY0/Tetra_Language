# Eco Package Guide

Status: user guide for local Eco/Todex release expectations.

The v1.0 release scope includes a local package lifecycle. Network publishing,
TetraHub production publishing, target-aware downloads, and global trust
metadata remain beta or post-v1 unless explicitly promoted.

## Local Lifecycle

Release-covered local flows should include:

- Capsule verification.
- Dependency graph and lockfile generation.
- Pack and unpack validation.
- Vault add/list/verify behavior.
- Publish metadata fixture validation without claiming a production network.

## Verification

Use the validators and smoke steps wired into `bash scripts/test_all.sh --full
--keep-going` and the current `bash scripts/release_v0_1_3_gate.sh` release
gate. The future v1.0 gate must not be recreated until these flows produce
current artifacts for the promoted branch state.
