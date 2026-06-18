# Backend Differential Validation v1

Status: P16.3 evidence audit for the Ideal Master Plan.

## Summary

The backend differential matrix now compares supported i32 rows across source, Stack IR, optimized
Stack IR, SSA, Machine IR, and Linux x64 native execution lanes. It covers deterministic scalar,
branch/loop, call-loop, and proof-tagged slice-sum rows, plus bounded randomized samples and
first-mismatch reducer metadata.

## Evidence

| Check                                                                                                | Result |
| ---------------------------------------------------------------------------------------------------- | ------ |
| Matrix exposes source, Stack IR, optimized Stack IR, SSA, Machine IR, and native lanes               | pass   |
| Call-loop row compares source, Stack IR, optimized Stack IR, SSA, Machine IR, and native exit result | pass   |
| Slice-sum row compares source, Stack IR, optimized Stack IR, SSA, Machine IR, and native exit result | pass   |
| Stack IR interpreter supports calls and i32 slice index load/store for matrix rows                   | pass   |
| SSA interpreter supports block params, effects, calls, conditional branches, and i32 index loads     | pass   |
| Machine IR interpreter supports calls, i32 slice index loads, div, and mod                           | pass   |
| Bounded randomized samples record deterministic seed and generated count                             | pass   |
| Mismatch report records reduced single-sample reproducer                                             | pass   |

## Boundaries

This audit is supported-subset evidence. It does not claim exhaustive random testing, a complete
source interpreter, a full native differential suite for every target, or full formal proof.
Unsupported rows remain explicit unsupported evidence until later slices promote them safely.
