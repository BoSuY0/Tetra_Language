# Tooling and Stdlib Production Design

Status: draft implementation design.

## Goal

Make Tetra production-ready for daily local development by closing the current
tooling and `lib.core` standard-library blockers without replacing placeholder
wording with unsupported claims.

## Observed Facts

- `cli.core` is `current` and covers the local workflow commands required by
  the goal.
- `docs/spec/cli_contracts.md` now names the current `v0.4.0` tooling contract.
- `tools/cmd/validate-tooling-stdlib-readiness` passes unit tests and fails the
  current repo because `stdlib.core-current` and `docs/spec/stdlib.md` still
  contain filesystem/networking/crypto placeholder claims.
- `lib/core/filesystem.tetra`, `lib/core/networking.tetra`, and
  `lib/core/crypto.tetra` currently implement deterministic helper APIs, not
  host filesystem, socket/DNS/HTTP, or cryptographic runtime APIs.
- The current native lowering/backend has memory, MMIO, capability, time, task,
  and actor runtime builtins. It does not expose host filesystem, networking, or
  crypto builtins.
- Tetra strings are passed as two-slot `str` values (`ptr`, `len`). Host
  filesystem syscalls usually require NUL-terminated byte strings, so direct
  syscall lowering needs an explicit string bridge rather than reusing the raw
  `str` pointer blindly.

## Production Boundary

The production claim must be one of these, and docs/features/gates must agree:

1. **Host-capability production APIs.** `lib.core.filesystem`,
   `lib.core.networking`, and `lib.core.crypto` expose real capability-gated
   host/runtime behavior with native and unsupported-target diagnostics.
2. **Pure-contract production APIs.** Those modules intentionally expose only
   deterministic path/endpoint/crypto-interface helper contracts, with no host
   filesystem, socket, DNS, HTTP, randomness, hashing, signing, encryption, or
   side-channel claim. The modules are production helpers, not host APIs.

The current goal asks for production stdlib modules without placeholder claims,
so option 1 is the safer interpretation. Option 2 requires explicit product
approval because it narrows what "filesystem", "networking", and "crypto" mean.

## Recommended Implementation Wave

Implement option 1 in narrow, testable slices:

1. **Filesystem Host MVP**
   - Add a manifest-visible builtin such as `core.fs_exists(path: str) -> bool`
     or a lower-level status builtin that returns an integer result.
   - Add a native x64 string bridge that copies `str(ptr,len)` into a
     NUL-terminated temporary buffer before invoking a host syscall/runtime
     helper.
   - Expose `lib.core.filesystem.exists(path: String) -> Bool uses io`.
   - Add Linux runtime smoke first; keep macOS/Windows/WASM unsupported or
     build-only until matching evidence exists.

2. **Networking Interface MVP**
   - Start with endpoint parsing/validation as production pure APIs only if
     approved, or add a capability-gated runtime connectivity/status builtin.
   - Keep DNS/HTTP/TCP out of the production claim until a real transport API
     and tests exist.

3. **Crypto Interface MVP**
   - Replace placeholder strength/mixer claims with either a real
     manifest-backed crypto provider interface or a deliberately non-secret
     checksum/hash utility under a non-cryptographic name.
   - Do not claim constant-time or cryptographic strength unless backend and
     release tests prove the property.

4. **DX Completion**
   - Promote LSP rename only after syntax-aware binding resolution exists, or
     keep it explicitly documented as limited and outside the full production
     tooling claim.

## Test Strategy

- RED tests for each new builtin in:
  - semantics builtin manifest and effect/unsafe policy tests
  - lowering tests proving the builtin lowers to the intended runtime call or
    target-aware diagnostic
  - native runtime smoke tests for Linux first
  - docs/API examples for the stable `lib.core` wrapper
- A dedicated readiness command:

```sh
./tetra features --format=json > /tmp/tetra-tooling-stdlib-features.json
go run ./tools/cmd/validate-tooling-stdlib-readiness \
  --features /tmp/tetra-tooling-stdlib-features.json \
  --stdlib-docs docs/spec/stdlib.md \
  --cli-contracts docs/spec/cli_contracts.md
```

This command must continue failing until the stdlib placeholder claims are
replaced by real implementation, docs, examples, and release evidence.

