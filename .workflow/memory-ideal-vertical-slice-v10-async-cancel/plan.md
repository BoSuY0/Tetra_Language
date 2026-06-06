# MEM-ASYNC-010 Workflow Plan

## Goal

Deliver narrow async cancellation and structured boundary conservatism evidence
for `MEM-ASYNC-001` through `MEM-ASYNC-005`.

## Phase Checklist

- [x] Goal contract initialized from accepted v9 baseline.
- [x] Inspect async/task/actor and memory evidence implementation files.
- [x] Add RED tests for post-await, cancellation, task group, actor reentry,
  async storage, missing correlation row, and widened correlation status.
- [x] Implement GREEN compiler-visible evidence and validators.
- [x] Add v10 correlation/final audit docs and manifest/design updates.
- [x] Run focused, correlation, docs/manifest, broad, CI, hygiene, and graph
  gates.

## Stop Rules

Stop and record a blocker if the slice requires production actor runtime proof,
distributed actor model, full async lifetime theorem, complete structured
concurrency proof, target parity, performance, broad noalias, arbitrary
FFI/runtime lifetime proof, or clean-release claim under dirty worktree.
