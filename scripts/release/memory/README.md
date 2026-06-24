# scripts/release/memory

Memory Core release gates that are scoped to canonical memory pipeline
evidence.

This directory owns `memory-core-v2-gate.sh`, the ordered Memory Core v2 gate
used by the v0.4 release gate and final stabilization evidence. Keep these
scripts focused on memory facts, allocation planning, lowering evidence,
runtime backend/domain checks, validators, and claim scanners. Broader
post-v0.4 production gates remain under `scripts/release/post_v0_4`.
