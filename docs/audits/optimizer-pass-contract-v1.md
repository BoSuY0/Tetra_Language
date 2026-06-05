# Optimizer Pass Contract v1

Status: P17.0 evidence audit for the Ideal Master Plan.

## Summary

Optimizer passes now carry an explicit machine-checkable contract before they
can run through `opt.Manager`. The contract records stable pass identity,
input/output verifier evidence, proof preservation or invalidation rules,
translation validation hook identity, stable report rows, and the negative-test
marker that rejects missing or fake evidence.

## Evidence

| Check | Result |
| --- | --- |
| Registered optimizer pass list exposes `basic-scalar`, `inline-small-pure`, and `loop-canonicalization` | pass |
| Every registered pass has a stable name and report output | pass |
| Every registered pass declares input and output verifier evidence | pass |
| Every registered pass declares proof preservation or invalidation rules | pass |
| Every registered pass declares `validation.ValidateTranslation` as the translation validation hook | pass |
| Pass reports include contract rows for verifiers, proof rule, translation hook, translation report, validation metadata, before dump, and after dump | pass |
| Translation validation metadata repeats the contract fields with stable before/after hashes and function comparison evidence | pass |
| Negative tests reject missing or fake verifier evidence, proof rules, translation hook, report rows, and negative-test marker | pass |

## Boundaries

This audit proves the optimizer pass contract for the currently registered
internal Stack IR optimization passes. It does not claim a broad optimized
backend mode, full optimization coverage, LLVM-style optimization maturity, or
performance parity. Later P17 slices must still implement and validate the
larger optimization set from the master plan.
