# Memory Target Capability Matrix

Status: MPC-13 target claim boundary.

This matrix prevents linux-x64 memory production evidence from being reused as a cross-target claim.
Each row separates artifact production from runtime execution, diagnostics, lowering, ABI/alignment
evidence, and the maximum memory claim level currently allowed for that target.

| Target | Build | Lower | Run | Raw diagnostics | Region lowering | Alignment semantics | Claim level |
| --- | --- | --- | --- | --- | --- | --- | --- |
| linux-x64 | yes | yes | yes | yes | yes/partial | yes | production/host_runtime |
| linux-x86 | yes | yes | no/host-dependent | partial | partial | partial | build_lower_only |
| linux-x32 | yes | yes | no/host-dependent | partial | partial | special | build_lower_only |
| macos-x64 | yes | yes | host-required | host-required | host-required | host-required | build_lower_only unless run |
| windows-x64 | yes | yes | host-required | host-required | host-required | host-required | build_lower_only unless run |
| wasm32-wasi | yes | yes | runner-smoke if available | safe-only | limited | wasm rules | artifact/runtime tiered |
| wasm32-web | yes | yes | browser-smoke if available | safe-only | limited | wasm rules | artifact/runtime tiered |

## Claim Rules

- `production/host_runtime` is currently limited to linux-x64 memory production reports that pass
  the release smoke, validator, benchmark, and audit gates.
- `build_lower_only` means the compiler may build and lower artifacts for that target, but the row
  does not claim production runtime memory behavior.
- `build_lower_only unless run` means macOS and Windows require target-host runtime evidence before
  any runtime memory claim can be made.
- `artifact/runtime tiered` means wasm artifacts may be built and lowered, but runtime claims depend
  on an explicit WASI runner or browser smoke for the target and stay within the safe-only/limited
  memory surface.
- `no cross-target memory production claim without target evidence`: a passing linux-x64 memory
  report does not prove linux-x86, linux-x32, macOS, Windows, wasm32-wasi, or wasm32-web runtime
  memory behavior.

## Validator Rejections

The target validators reject:

- runtime memory claims on build-only targets;
- raw diagnostics claims on targets without raw diagnostics evidence;
- region lowering claims without lowered artifact evidence;
- alignment claims without target-specific ABI evidence.

## Non-Claims

This matrix does not claim target parity, production raw diagnostics outside linux-x64, full region
lowering parity, broad actor/runtime parity, or memory performance parity across targets.
