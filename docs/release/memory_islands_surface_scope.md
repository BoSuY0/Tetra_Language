# Memory/Islands/Surface Scoped Release Truth

Status: current scoped evidence contract for the Memory, Islands, and Surface
combined release path.

This document is release truth, not a production claim by itself. The scoped
state is valid only when the live same-commit gates and validators named here
pass for the current checkout.

## Required Gate

The integrated gate is:

```bash
bash scripts/release/post_v0_4/memory-islands-surface-production-gate.sh --report-dir reports/memory-islands-surface-production
```

The gate writes and validates the integrated manifest with:

```bash
go run ./tools/cmd/validate-memory-islands-surface-production --report-dir reports/memory-islands-surface-production
```

Required evidence includes:

- `memory-islands-surface-production-manifest.json`
- `artifact-hashes.json`
- `islands-debug-smoke.json`
- `island-proof-verifier.json`
- `island-proof-fuzz-summary.json`

## Scope

- Memory evidence is grounded in `MemoryFactGraph`, memory production reports,
  leak/resource finalization evidence, and artifact hashes.
- Island evidence requires `tools/cmd/validate-island-proof`, the
  `--islands-debug` sanitizer smoke path, and deterministic
  `island-proof-fuzz-summary` mutation rejection.
- Surface evidence is scoped to `surface-v1-linux-web`: headless, linux-x64
  real-window, and wasm32-web browser-canvas evidence plus Surface API,
  SafeView lifetime, and release-state validators.

## Nonclaims

- no Memory 100% claim
- no arbitrary unsafe external pointer safety
- no full formal proof
- no full target parity
- no all-target Surface claim
- no production object memory claim
- no production persistent memory claim
- not a clean release-candidate checkout claim unless the working tree is clean
  and the final release audit says so
