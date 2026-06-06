# MEM-DYNPROTO-011 Workflow Control

Root `CONTROL.md` is authoritative.

## Current Controls

- priority: evidence_quality
- scope: narrow compiler-visible dynamic protocol/existential/witness memory
  evidence only
- protected: unrelated dirty worktree changes
- stop on: full existential runtime proof, witness-table ABI proof, broad
  noalias, unsafe/external pointer promotion, target/performance proof, or
  clean-release claim under dirty worktree
