# TODO Closure Map (2026-04-26)

- Source: `docs/plans/2026-04-26-tetra-language-todo.md`
- Scope: Remaining unchecked checklist lines only
- Format: Machine-readable JSON payload

```json
{
  "source_file": "docs/plans/2026-04-26-tetra-language-todo.md",
  "generated_on": "2026-04-26",
  "total_unchecked_items": 88,
  "entries": [
    {
      "line": 234,
      "text": "Define the final Flow-only grammar for v1.0.",
      "closure_mode": "blocked-by-prerequisite",
      "rationale": "Frontend closure is blocked on a final Flow grammar/labeling freeze for v1.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/flow_syntax_mvp.md"
    },
    {
      "line": 239,
      "text": "Finish argument labels.",
      "closure_mode": "blocked-by-prerequisite",
      "rationale": "Frontend closure is blocked on a final Flow grammar/labeling freeze for v1.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/flow_syntax_mvp.md"
    },
    {
      "line": 270,
      "text": "Complete multi-slot optionals.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/v1_feature_status.md"
    },
    {
      "line": 271,
      "text": "Complete multi-slot typed errors.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/v1_feature_status.md"
    },
    {
      "line": 272,
      "text": "Support generic functions across modules.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/v1_feature_status.md"
    },
    {
      "line": 275,
      "text": "Add extension conformance clauses.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/v1_feature_status.md"
    },
    {
      "line": 276,
      "text": "Stabilize monomorphization names.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/v1_feature_status.md"
    },
    {
      "line": 304,
      "text": "Model local lifetimes and borrow scopes.",
      "closure_mode": "blocked-by-prerequisite",
      "rationale": "Ownership closure is blocked on explicit lifetime and actor-transfer rules.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/islands.md"
    },
    {
      "line": 309,
      "text": "Define safe transfer rules for actor/task boundaries.",
      "closure_mode": "blocked-by-prerequisite",
      "rationale": "Ownership closure is blocked on explicit lifetime and actor-transfer rules.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/islands.md"
    },
    {
      "line": 335,
      "text": "Extend `uses` into effect groups.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/capabilities.md"
    },
    {
      "line": 336,
      "text": "Propagate effects through generics.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/capabilities.md"
    },
    {
      "line": 337,
      "text": "Propagate effects through protocols.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/capabilities.md"
    },
    {
      "line": 338,
      "text": "Add capability attenuation.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/capabilities.md"
    },
    {
      "line": 339,
      "text": "Add capsule permission checks.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/capabilities.md"
    },
    {
      "line": 340,
      "text": "Add secret/privacy types if still in v1.0.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/capabilities.md"
    },
    {
      "line": 341,
      "text": "Add consent-token MVP if still in v1.0.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/capabilities.md"
    },
    {
      "line": 342,
      "text": "Add checked privacy clauses if still in v1.0.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/capabilities.md"
    },
    {
      "line": 343,
      "text": "Add `budget`, `noalloc`, `noblock`, `realtime`, and `nothrow` syntax or explicitly defer them.",
      "closure_mode": "implemented-now",
      "rationale": "This line explicitly allows deferment, so it can be closed now by recording a v1 scope freeze decision.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/capabilities.md"
    },
    {
      "line": 345,
      "text": "Add runtime checks for the rest.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/capabilities.md"
    },
    {
      "line": 369,
      "text": "Define the v1.0 task ABI.",
      "closure_mode": "blocked-by-prerequisite",
      "rationale": "Async/runtime work is blocked until the v1 task ABI and runtime contracts are frozen.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/runtime_abi.md"
    },
    {
      "line": 370,
      "text": "Implement structured task groups.",
      "closure_mode": "blocked-by-prerequisite",
      "rationale": "Async/runtime work is blocked until the v1 task ABI and runtime contracts are frozen.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/runtime_abi.md"
    },
    {
      "line": 371,
      "text": "Implement cancellation.",
      "closure_mode": "blocked-by-prerequisite",
      "rationale": "Async/runtime work is blocked until the v1 task ABI and runtime contracts are frozen.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/runtime_abi.md"
    },
    {
      "line": 372,
      "text": "Add typed task handles.",
      "closure_mode": "blocked-by-prerequisite",
      "rationale": "Async/runtime work is blocked until the v1 task ABI and runtime contracts are frozen.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/runtime_abi.md"
    },
    {
      "line": 373,
      "text": "Add typed async error propagation.",
      "closure_mode": "blocked-by-prerequisite",
      "rationale": "Async/runtime work is blocked until the v1 task ABI and runtime contracts are frozen.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/runtime_abi.md"
    },
    {
      "line": 374,
      "text": "Expand actors beyond `i32` messages.",
      "closure_mode": "blocked-by-prerequisite",
      "rationale": "Async/runtime work is blocked until the v1 task ABI and runtime contracts are frozen.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/spec/runtime_abi.md"
    },
    {
      "line": 405,
      "text": "Add debug info support.",
      "closure_mode": "blocked-by-prerequisite",
      "rationale": "Backend closure is blocked on prerequisite WASM architecture/codegen milestones.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/backend/wasm_architecture.md"
    },
    {
      "line": 406,
      "text": "Add release optimization coverage.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/backend/wasm_architecture.md"
    },
    {
      "line": 408,
      "text": "Implement `wasm32-wasi` target parsing as supported only after backend exists.",
      "closure_mode": "blocked-by-prerequisite",
      "rationale": "Backend closure is blocked on prerequisite WASM architecture/codegen milestones.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/backend/wasm_architecture.md"
    },
    {
      "line": 409,
      "text": "Implement `wasm32-wasi` codegen/object/link/run path.",
      "closure_mode": "blocked-by-prerequisite",
      "rationale": "Backend closure is blocked on prerequisite WASM architecture/codegen milestones.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/backend/wasm_architecture.md"
    },
    {
      "line": 410,
      "text": "Implement `wasm32-web` codegen/package path.",
      "closure_mode": "blocked-by-prerequisite",
      "rationale": "Backend closure is blocked on prerequisite WASM architecture/codegen milestones.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/backend/wasm_architecture.md"
    },
    {
      "line": 411,
      "text": "Add smoke coverage for both WASM targets.",
      "closure_mode": "blocked-by-prerequisite",
      "rationale": "Backend closure is blocked on prerequisite WASM architecture/codegen milestones.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/backend/wasm_architecture.md"
    },
    {
      "line": 442,
      "text": "Promote collections.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/stdlib.md"
    },
    {
      "line": 443,
      "text": "Promote strings.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/stdlib.md"
    },
    {
      "line": 444,
      "text": "Promote slices.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/stdlib.md"
    },
    {
      "line": 445,
      "text": "Promote math.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/stdlib.md"
    },
    {
      "line": 446,
      "text": "Promote IO.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/stdlib.md"
    },
    {
      "line": 447,
      "text": "Promote filesystem.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/stdlib.md"
    },
    {
      "line": 448,
      "text": "Promote networking.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/stdlib.md"
    },
    {
      "line": 449,
      "text": "Promote async.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/stdlib.md"
    },
    {
      "line": 450,
      "text": "Promote sync.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/stdlib.md"
    },
    {
      "line": 451,
      "text": "Promote testing.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/stdlib.md"
    },
    {
      "line": 452,
      "text": "Promote serialization.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/stdlib.md"
    },
    {
      "line": 453,
      "text": "Promote time.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/stdlib.md"
    },
    {
      "line": 454,
      "text": "Promote crypto interfaces.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/stdlib.md"
    },
    {
      "line": 530,
      "text": "Write a UI syntax/spec document before implementation.",
      "closure_mode": "blocked-by-prerequisite",
      "rationale": "UI implementation is blocked until the dedicated UI syntax/spec document exists and is approved.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/v1_feature_status.md"
    },
    {
      "line": 531,
      "text": "Implement `view`.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/v1_feature_status.md"
    },
    {
      "line": 532,
      "text": "Implement `state`.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/v1_feature_status.md"
    },
    {
      "line": 533,
      "text": "Implement binding.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/v1_feature_status.md"
    },
    {
      "line": 534,
      "text": "Implement events.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/v1_feature_status.md"
    },
    {
      "line": 535,
      "text": "Implement commands.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/v1_feature_status.md"
    },
    {
      "line": 536,
      "text": "Implement typed style.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/v1_feature_status.md"
    },
    {
      "line": 537,
      "text": "Implement accessibility metadata.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/v1_feature_status.md"
    },
    {
      "line": 538,
      "text": "Add web backend through `wasm32-web`.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/v1_feature_status.md"
    },
    {
      "line": 539,
      "text": "Add native shell backend.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/v1_feature_status.md"
    },
    {
      "line": 540,
      "text": "Add web UI smoke app.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/v1_feature_status.md"
    },
    {
      "line": 541,
      "text": "Add native shell UI smoke app.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/spec/v1_feature_status.md"
    },
    {
      "line": 564,
      "text": "Stabilize Capsule manifest v1.",
      "closure_mode": "blocked-by-prerequisite",
      "rationale": "Eco manifest/permission stabilization is blocked on prerequisite design and policy freeze decisions.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/spec/api_diff_policy.md"
    },
    {
      "line": 566,
      "text": "Stabilize permission model.",
      "closure_mode": "blocked-by-prerequisite",
      "rationale": "Eco manifest/permission stabilization is blocked on prerequisite design and policy freeze decisions.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/spec/api_diff_policy.md"
    },
    {
      "line": 569,
      "text": "Implement Seed import/export.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/spec/api_diff_policy.md"
    },
    {
      "line": 570,
      "text": "Implement NeedMap.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/spec/api_diff_policy.md"
    },
    {
      "line": 571,
      "text": "Implement TrustSnapshot.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/spec/api_diff_policy.md"
    },
    {
      "line": 572,
      "text": "Implement Materializer.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/spec/api_diff_policy.md"
    },
    {
      "line": 573,
      "text": "Add reproducible build basics.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/spec/api_diff_policy.md"
    },
    {
      "line": 575,
      "text": "Add beta package publishing.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/spec/api_diff_policy.md"
    },
    {
      "line": 576,
      "text": "Add TetraHub beta path.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/spec/api_diff_policy.md"
    },
    {
      "line": 577,
      "text": "Add target-aware downloads.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/spec/api_diff_policy.md"
    },
    {
      "line": 578,
      "text": "Add trust metadata.",
      "closure_mode": "deferred-post-v1",
      "rationale": "This feature is outside the proposed v1 freeze line and should be closed as post-v1 backlog scope.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/spec/api_diff_policy.md"
    },
    {
      "line": 579,
      "text": "Keep full distributed Todex mesh, proof-carrying capsules, global EcoTrust, EcoOracle, and live evolution documented as post-v1.0 unless explicitly promoted.",
      "closure_mode": "implemented-now",
      "rationale": "This is a documentation-only deferment statement and can be closed immediately via the eco/release scope freeze.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/spec/api_diff_policy.md"
    },
    {
      "line": 607,
      "text": "Update version only when the release branch is actually ready.",
      "closure_mode": "release-branch-only",
      "rationale": "This item should close only on the actual release branch when final v1 cutover evidence exists.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/checklists/v1_0_release_gate.md"
    },
    {
      "line": 608,
      "text": "Regenerate and validate docs manifest.",
      "closure_mode": "release-branch-only",
      "rationale": "This item should close only on the actual release branch when final v1 cutover evidence exists.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/checklists/v1_0_release_gate.md"
    },
    {
      "line": 609,
      "text": "Finalize release notes.",
      "closure_mode": "release-branch-only",
      "rationale": "This item should close only on the actual release branch when final v1 cutover evidence exists.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/checklists/v1_0_release_gate.md"
    },
    {
      "line": 610,
      "text": "Check every item in `docs/checklists/v1_0_release_gate.md`.",
      "closure_mode": "release-branch-only",
      "rationale": "This item should close only on the actual release branch when final v1 cutover evidence exists.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/checklists/v1_0_release_gate.md"
    },
    {
      "line": 613,
      "text": "Run build-only smoke for all mandatory native and WASM targets.",
      "closure_mode": "release-branch-only",
      "rationale": "This item should close only on the actual release branch when final v1 cutover evidence exists.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/checklists/v1_0_release_gate.md"
    },
    {
      "line": 614,
      "text": "Run WASI smoke in a WASI runner.",
      "closure_mode": "release-branch-only",
      "rationale": "This item should close only on the actual release branch when final v1 cutover evidence exists.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/checklists/v1_0_release_gate.md"
    },
    {
      "line": 615,
      "text": "Run web UI smoke through browser automation.",
      "closure_mode": "release-branch-only",
      "rationale": "This item should close only on the actual release branch when final v1 cutover evidence exists.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/checklists/v1_0_release_gate.md"
    },
    {
      "line": 616,
      "text": "Verify docs manifest and doctests.",
      "closure_mode": "release-branch-only",
      "rationale": "This item should close only on the actual release branch when final v1 cutover evidence exists.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/checklists/v1_0_release_gate.md"
    },
    {
      "line": 617,
      "text": "Verify API diff reports.",
      "closure_mode": "release-branch-only",
      "rationale": "This item should close only on the actual release branch when final v1 cutover evidence exists.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/checklists/v1_0_release_gate.md"
    },
    {
      "line": 618,
      "text": "Verify reproducible builds for at least one native and one WASM target.",
      "closure_mode": "release-branch-only",
      "rationale": "This item should close only on the actual release branch when final v1 cutover evidence exists.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/checklists/v1_0_release_gate.md"
    },
    {
      "line": 632,
      "text": "1. Freeze historical green v0.6.0 baseline.",
      "closure_mode": "implemented-now",
      "rationale": "This sequencing checklist is superseded by the current wave status and can be closed now as planning bookkeeping.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/roadmap_0_6_to_1_0.md"
    },
    {
      "line": 633,
      "text": "2. Finish or explicitly split the v0.6.x stabilization tasks.",
      "closure_mode": "implemented-now",
      "rationale": "This sequencing checklist is superseded by the current wave status and can be closed now as planning bookkeeping.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/roadmap_0_6_to_1_0.md"
    },
    {
      "line": 634,
      "text": "3. Validate the first v0.7 hardening slice.",
      "closure_mode": "implemented-now",
      "rationale": "This sequencing checklist is superseded by the current wave status and can be closed now as planning bookkeeping.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/roadmap_0_6_to_1_0.md"
    },
    {
      "line": 635,
      "text": "4. Start v1.0 Wave 1: Flow-only frontend.",
      "closure_mode": "implemented-now",
      "rationale": "This sequencing checklist is superseded by the current wave status and can be closed now as planning bookkeeping.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/roadmap_0_6_to_1_0.md"
    },
    {
      "line": 636,
      "text": "5. Do type system stabilization before ownership/race freedom.",
      "closure_mode": "implemented-now",
      "rationale": "This sequencing checklist is superseded by the current wave status and can be closed now as planning bookkeeping.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/roadmap_0_6_to_1_0.md"
    },
    {
      "line": 637,
      "text": "6. Do ownership/race freedom before claiming safe-code guarantees.",
      "closure_mode": "implemented-now",
      "rationale": "This sequencing checklist is superseded by the current wave status and can be closed now as planning bookkeeping.",
      "target_ref": "docs/plans/v1_scope_freeze_frontend_runtime.md",
      "supporting_ref": "docs/roadmap_0_6_to_1_0.md"
    },
    {
      "line": 638,
      "text": "7. Add WASM before UI web release checks.",
      "closure_mode": "implemented-now",
      "rationale": "This sequencing checklist is superseded by the current wave status and can be closed now as planning bookkeeping.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/roadmap_0_6_to_1_0.md"
    },
    {
      "line": 639,
      "text": "8. Stabilize stdlib and tooling before final release notes.",
      "closure_mode": "implemented-now",
      "rationale": "This sequencing checklist is superseded by the current wave status and can be closed now as planning bookkeeping.",
      "target_ref": "docs/plans/v1_scope_freeze_backend_stdlib_ui.md",
      "supporting_ref": "docs/roadmap_0_6_to_1_0.md"
    },
    {
      "line": 640,
      "text": "9. Run the final v1.0 gate only after every placeholder has a real implementation.",
      "closure_mode": "implemented-now",
      "rationale": "This sequencing checklist is superseded by the current wave status and can be closed now as planning bookkeeping.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/roadmap_0_6_to_1_0.md"
    },
    {
      "line": 657,
      "text": "Decide whether v0.7 should become an official intermediate release or remain an internal hardening slice.",
      "closure_mode": "implemented-now",
      "rationale": "This is a roadmap policy decision that can be resolved now in the eco/release scope freeze without implementation changes.",
      "target_ref": "docs/plans/v1_scope_freeze_eco_release.md",
      "supporting_ref": "docs/roadmap_0_6_to_1_0.md"
    }
  ],
  "summary_by_closure_mode": {
    "implemented-now": 12,
    "deferred-post-v1": 48,
    "blocked-by-prerequisite": 18,
    "release-branch-only": 10
  }
}
```
