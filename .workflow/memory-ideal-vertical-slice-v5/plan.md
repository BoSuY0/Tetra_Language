# Memory Ideal Vertical Slice v5 Plan

## Scope

Implement exactly four raw-pointer unsafe contract rows:

- `MEM-UNSAFE-001`
- `MEM-UNSAFE-002`
- `MEM-UNSAFE-003`
- `MEM-UNSAFE-004`

## Work Packets

1. MemoryFactGraph and report projections:
   `unsafe_unknown_rejected_safe_facts`,
   `unsafe_verified_root_allocation_base`,
   `unsafe_contract_runtime_checkable`, and
   `unsafe_contract_static_untrusted`.
2. MiniMemoryModel v5 raw-pointer cases.
3. Report and correlation validators.
4. Current supported semantics surface checks for raw pointer unsafe gateways.
5. Correlation/final audit docs and manifest/schema/design updates.
6. Focused gates, broad gates, `git diff --check`, and `graphify update .`.

## Nonclaims

No arbitrary external pointer safety, FFI lifetime system, broad unsafe
noalias, safe wrapper promotion, actor/task/runtime expansion, target parity,
or performance claim.
