# Surface vs Electron Comparison

Surface-vs-Electron evidence is method-first. It supports this wording only:

`Surface is competitive with Electron in the supported Linux/web scope.`

The report schema is `tetra.surface.electron-comparison-report.v1` at level
`surface-electron-comparison-method-v1`. The deterministic gate is
`scripts/release/surface/electron-comparison-gate.sh`; the report generator is
`surface-electron-comparison`, and the validator is
`validate-surface-electron-comparison-report`.

## Required Method

- equivalent app shapes for Surface and Electron;
- same feature set, assets, input script, OS/target, cold/warm state, and
  measurement tool;
- startup, RSS, first frame, input latency, idle CPU, and package size rows;
- at least five samples;
- variance reported for every metric;
- captured hardware, OS, architecture, power profile, and measurement tool;
- public positioning generated from the report.

## Rejections

The validator rejects public benchmark-superiority claim, cherry-picked
hardware, missing variance, missing environment, unfair app shape, and single-smoke
faster-than-Electron claim. It also rejects broad Electron replacement,
React/CSS/Electron compatibility, and arbitrary Electron app migration claims.

## Nonclaims

This comparison is not an external ranking or superiority result, not a broad
Electron replacement claim, not a React/CSS/Electron compatibility claim, not
proof that arbitrary Electron apps migrate to Surface, and not a faster-than-
Electron claim from one local smoke.
