# Post-v1 Feature Promotion Checklist

Status: required checklist before post-v1 features move into release scope.

Use this checklist for closures, enum payloads, structured concurrency, full UI
runtime work, EcoNet, and any feature listed as post-v1 in `docs/spec/v1_scope.md`.

## Required Promotion Evidence

- [ ] Design note names the feature, owner, compatibility story, and explicit
      non-goals.
- [ ] Scope delta updates `docs/spec/v1_scope.md` and any affected spec files.
- [ ] Parser, semantic, lowering, diagnostics, and target behavior tests exist
      where the feature touches those surfaces.
- [ ] Migration notes explain old behavior, new behavior, and how users detect
      unsupported code.
- [ ] User docs include a runnable example or a documented build-only example.
- [ ] Release gate or validator coverage makes the feature hard to regress.
- [ ] Backward compatibility risk is accepted by the release owner.
- [ ] Security review covers capability, unsafe, host boundary, package trust,
      privacy, or resource budget changes if any are touched.
- [ ] Performance impact is recorded when the feature affects compiler,
      formatter, docs generation, runtime, binary size, or cache behavior.

## Feature-Specific Gates

| Feature | Minimum gate before promotion |
| --- | --- |
| Closures | capture/lifetime tests, diagnostics for escaping borrows, docs examples |
| Enum payloads | constructor/destructuring tests, exhaustive match migration notes |
| Structured concurrency | cancellation/join semantics, bounded stress tests, runtime docs |
| Full UI runtime | event loop, accessibility, layout, web/native smoke artifacts |
| EcoNet | package trust threat model, tamper tests, registry migration policy |

## Closure Rule

Do not mark a post-v1 feature release-covered until this checklist, the security
review gate when applicable, and the release evidence artifact all reference the
same branch state.

