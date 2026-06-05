# Tetra Surface v1 Release Contract

Status: current for surface-v1-linux-web scope after release gate passes.

This contract is the release-truth boundary for promoting Tetra Surface v1. It
does not promote every Surface experiment to production. A release/current claim
is valid only when release reports use `release_scope: surface-v1-linux-web`,
`status: current`, `experimental: false`, and `production_claim: true`, and the
release gate validates the required target evidence and artifact hashes.

## Supported

- pure-Tetra user UI code
- tiny Surface Host ABI
- headless release evidence target
- linux-x64 real-window runtime
- wasm32-web browser-canvas runtime
- software RGBA framebuffer presentation
- component tree helper API
- production widget toolkit v1 subset
- production text/input baseline
- clipboard baseline
- IME/composition baseline
- accessibility metadata plus platform bridge for supported targets
- release validators and artifact hashes

## Unsupported

- macOS real-window Surface
- Windows real-window Surface
- wasm32-wasi Surface UI
- GPU renderer
- dynamic trait-object component ABI
- witness-table widget dispatch
- full rich text editor
- arbitrary native platform widgets
- React/DOM UI/user-JS app logic

## Release Target Matrix

| Target | Release status | Required evidence |
|---|---|---|
| `headless` | release-test-supported | deterministic runtime/text/toolkit/accessibility evidence |
| `linux-x64` | current | real Wayland shm window, native event pump, text/clipboard/IME, toolkit, accessibility bridge |
| `wasm32-web` | current | real browser canvas, browser input, clipboard/IME, toolkit, accessibility snapshot/mirror |
| `macos-x64` | unsupported for Surface v1 | must not claim production |
| `windows-x64` | unsupported for Surface v1 | must not claim production |
| `wasm32-wasi` | unsupported for Surface UI | must not claim UI runtime |

## Release Status Vocabulary

Feature and report status models may use these lifecycle labels:

- `experimental`
- `release_candidate`
- `current`
- `unsupported`
- `legacy_compatibility`

Historical and non-Surface registries may still use existing future-planning
labels such as `planned` and `post-v1`; those labels do not constitute Surface
release evidence.

## Final Release Report Rules

Final Surface release summaries must include:

```json
{
  "status": "current",
  "experimental": false,
  "production_claim": true,
  "release_scope": "surface-v1-linux-web"
}
```

Unsupported target entries must remain non-current and non-production:

```json
{
  "status": "unsupported",
  "production_claim": false,
  "reason": "no real target-host Surface v1 evidence in this release"
}
```

The release validator must reject any production/current claim for
`macos-x64`, `windows-x64`, or `wasm32-wasi` until a future release contract
adds real target-host evidence for that target.
