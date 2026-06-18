# compiler/tests/safety

Safety diagnostic and capability boundary tests belong here when they are
domain-level compiler checks rather than package-private implementation tests.

Keep fixtures small and versioned by behavior.

- Root package: runtime export, island scope, stabilization, and Epic 06 safety
  matrix coverage.
- `effects/`: effects, privacy, semantic-clause, and function-typed effect
  propagation coverage.
- `diagnostics/core/`: broad diagnostic identity, key-family, global-escape,
  protocol, and optional payload coverage.
- `diagnostics/aliases/`: borrowed aggregate, resource alias, task alias, and
  callback alias diagnostics.
