package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateManifestAcceptsGeneratedShape(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "formats": [
    {"name":"T4 Source Format","extension":".t4","role":"source","description":"Tetra source file","primary":true},
    {"name":"Legacy Tetra Source Format","extension":".tetra","role":"source","description":"Legacy Tetra source file","legacy":true},
    {"name":"Todex Fragment","extension":".tdx","role":"todex-fragment","description":"Todex encrypted semantic fragment"},
    {"name":"T4 Seed","extension":".t4s","role":"offline-seed","description":"Tetra Seed offline bundle"},
    {"name":"T4 Interface","extension":".t4i","role":"interface","description":"T4 interface file"},
    {"name":"T4 Proof","extension":".t4p","role":"proof","description":"T4 proof file"},
    {"name":"T4 Replay","extension":".t4r","role":"replay","description":"T4 replay file"},
    {"name":"T4 Quest","extension":".t4q","role":"quest","description":"T4 executable quest file"},
    {"name":"Tetra NeedMap","extension":".tneed","role":"needmap","description":"NeedMap file"},
    {"name":"Tetra Semantic Lock","file_name":"Tetra.lock","role":"semantic-lock","description":"Tetra semantic lockfile"}
  ],
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf","exe_ext":"","collect_imports":false},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe","exe_ext":".exe","collect_imports":true},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho","exe_ext":"","collect_imports":false},
    {"triple":"wasm32-wasi","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm","exe_ext":".wasm","collect_imports":false},
    {"triple":"wasm32-web","os":"web","arch":"wasm32","abi":"web","format":"wasm","exe_ext":".wasm","collect_imports":false}
  ],
  "builtins": [
    {"name":"core.load_i32","param_types":["ptr","cap.mem"],"return_type":"i32","effects":["mem"],"unsafe_policy":"always"},
    {"name":"core.print","aliases":["print"],"param_types":["str"],"return_type":"i32","effects":["io"],"unsafe_policy":"never"}
  ],
	  "runtime_abi": {
	    "reserved_prefix": "__tetra_",
	    "actors_supported_targets": ["linux-x64","macos-x64","windows-x64"],
	    "actors_required_symbols": ["__tetra_entry","__tetra_actor_spawn","__tetra_actor_send","__tetra_actor_send_msg","__tetra_actor_send_begin","__tetra_actor_send_slot","__tetra_actor_send_commit","__tetra_actor_recv","__tetra_actor_recv_msg","__tetra_actor_recv_poll","__tetra_actor_recv_until","__tetra_actor_recv_msg_until","__tetra_actor_recv_begin","__tetra_actor_recv_slot","__tetra_actor_recv_count","__tetra_actor_self","__tetra_actor_sender","__tetra_actor_yield_now"],
	    "actor_state_required_symbols": ["__tetra_actor_state_load","__tetra_actor_state_store"],
	    "task_required_symbols": ["__tetra_task_spawn_i32","__tetra_task_join_i32","__tetra_task_join_result_i32","__tetra_task_join_until_i32","__tetra_task_poll_i32","__tetra_task_is_canceled","__tetra_task_checkpoint"],
	    "task_group_required_symbols": ["__tetra_task_group_open","__tetra_task_group_close","__tetra_task_group_cancel","__tetra_task_group_current","__tetra_task_group_status","__tetra_task_spawn_group_i32"],
	    "typed_task_required_symbols": ["__tetra_task_result_begin","__tetra_task_result_slot","__tetra_task_result_get","__tetra_task_join_typed_2","__tetra_task_join_typed_3","__tetra_task_join_typed_4","__tetra_task_join_typed_5","__tetra_task_join_typed_6","__tetra_task_join_typed_7","__tetra_task_join_typed_8"],
	    "time_required_symbols": ["__tetra_time_now_ms","__tetra_sleep_ms","__tetra_sleep_until_ms","__tetra_deadline_ms","__tetra_timer_ready_ms"],
    "filesystem_required_symbols": ["__tetra_fs_exists"],
	    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
	  },
  "features": [
    {"id":"cli.core","name":"CLI","status":"current","since":"v0.2.0","scope":"core CLI","stability":"supported","docs":["docs/spec/current_supported_surface.md"]},
    {"id":"language.flow","name":"Flow","status":"current","since":"v0.2.0","scope":"flow syntax","stability":"supported","docs":["docs/spec/flow_syntax_v1.md"]},
    {"id":"language.generics-mvp","name":"Generics MVP","status":"current","since":"v0.2.0","scope":"statically monomorphized generic functions with no runtime generic values or dynamic dispatch","stability":"supported static MVP; generic structs remain future/post-v1","docs":["docs/spec/current_supported_surface.md","docs/spec/flow_syntax_v1.md","docs/spec/v1_scope.md"]},
    {"id":"language.protocol-conformance-mvp","name":"Protocol conformance MVP","status":"current","since":"v0.2.0","scope":"checked statically with generic requirement signature shape and no witness tables","stability":"dynamic dispatch remain post-v1","docs":["docs/spec/current_supported_surface.md","docs/spec/flow_syntax_v1.md","docs/spec/v1_scope.md"]},
    {"id":"language.callable-mvp","name":"Callable MVP","status":"current","since":"v0.2.0","scope":"Level 0 callable surface","stability":"current constrained MVP","docs":["docs/spec/current_supported_surface.md","docs/spec/flow_syntax_v1.md"]},
    {"id":"targets.wasm-artifact-preflight","name":"WASM artifact/import preflight","status":"current","since":"v0.2.0","scope":"artifact/import smoke","stability":"supported","docs":["docs/backend/wasm_backend_plan.md"]},
    {"id":"stdlib.experimental-mirrors","name":"Standard-library compatibility mirrors","status":"current","since":"v0.4.0","scope":"production compatibility mirrors forward to lib.core modules","stability":"stable callers should import lib.core directly","docs":["docs/spec/stdlib.md","docs/spec/stdlib_naming_versioning.md","docs/user/standard_library_guide.md"]},
    {"id":"language.callable-level1","name":"Callable Level 1","status":"current","since":"v0.4.0","scope":"production non-capturing symbol-backed callable Level 1 with function-typed locals, aliases, callbacks, and symbol-backed returns","stability":"captured closure escape and full first-class function values remain out of scope","docs":["docs/spec/current_supported_surface.md","docs/spec/flow_syntax_v1.md","docs/spec/v1_feature_status.md"]},
    {"id":"language.enum-payload-match","name":"Enum payload constructors and exhaustive match/catch","status":"current","since":"v0.3.0","scope":"positional enum payload constructors and payload bindings for match/catch/if-let, with exhaustive unguarded enum match/catch","stability":"nested destructuring patterns and guard expansion remain future/post-v1","docs":["docs/spec/current_supported_surface.md","docs/spec/flow_syntax_v1.md","docs/spec/v0_3_scope.md"]},
    {"id":"language.protocol-bound-generics-static","name":"Static protocol-bound generics","status":"current","since":"v0.3.0","scope":"validated statically during monomorphization with same-module and cross-module impl conformance plus visibility diagnostics","stability":"calling protocol requirements through generic bounds and dynamic dispatch remain unsupported","docs":["docs/spec/current_supported_surface.md","docs/spec/v0_3_scope.md","docs/spec/flow_syntax_v1.md"]},
    {"id":"language.ownership-markers-mvp","name":"Ownership markers MVP","status":"current","since":"v0.2.0","scope":"conservative borrow/inout/consume marker checks with use-after-consume and borrow escape diagnostics","stability":"supported conservative MVP; not a full SSA lifetime solver","docs":["docs/spec/current_supported_surface.md","docs/spec/ownership_v1.md","docs/spec/v1_scope.md"]},
    {"id":"language.resource-lifetime-mvp","name":"Resource lifetime MVP","status":"current","since":"v0.2.0","scope":"conservative resource finalization checks for task handles, task groups, island handles, region-backed slices, and structs containing them, including double-use and ambiguous provenance diagnostics","stability":"supported conservative MVP; tracks common local scope and control-flow merge cases, but is not a full SSA lifetime solver","docs":["docs/spec/current_supported_surface.md","docs/spec/ownership_v1.md","docs/spec/v1_scope.md"]},
    {"id":"actors.task-transfer-safety","name":"Actor/task transfer safety MVP","status":"current","since":"v0.2.0","scope":"conservative actor/task ownership transfer checks for worker entrypoints and use-after-transfer diagnostics","stability":"supported conservative local MVP; distributed actors remain outside current support","docs":["docs/spec/current_supported_surface.md","docs/spec/ownership_v1.md","docs/spec/v1_scope.md"]},
    {"id":"language.lifetime-ssa","name":"Lifetime SSA local join solver","status":"current","since":"v0.4.0","scope":"production SSA-like local lifetime join analysis for ownership consume state, resource finalization state, branch/match/loop flow snapshots, and maybe-consumed diagnostics","stability":"current local/control-flow solver; richer interprocedural lifetime proofs, broad alias modeling, race proofs, and full formal lifetime guarantees remain under full-v1 scope","docs":["docs/spec/current_supported_surface.md","docs/spec/ownership_v1.md","docs/spec/v1_scope.md"]},
    {"id":"safety.production-core","name":"Production safety core","status":"current","since":"v0.4.0","scope":"production local safety model for ownership/lifetime/borrow/consume/inout checks, resource finalization, callable escape diagnostics, effects/capabilities/privacy/consent/budget policy, unsafe boundaries, actor/task transfer safety, and pointer/MMIO/memory capability gates","stability":"release-gated current profile with explicit diagnostics for unsupported distributed, cryptographic, formal-proof, and runtime-wide guarantees","docs":["docs/spec/current_supported_surface.md","docs/spec/ownership_v1.md","docs/spec/effects_capabilities_privacy_v1.md"]},
    {"id":"language.callable-level2","name":"Callable Level 2","status":"current","since":"v0.4.0","scope":"production captured closure Level 2 slice with function-typed locals called directly","stability":"captured callback passing and full first-class callable semantics remain out of scope","docs":["docs/spec/current_supported_surface.md","docs/spec/flow_syntax_v1.md","docs/spec/v1_feature_status.md"]},
    {"id":"ui.metadata-v1","name":"UI metadata v0.4.0","status":"current","since":"v0.4.0","scope":"production UI metadata contract with deterministic tetra.ui.v0.4.0 JSON","stability":"web command dispatch; native widgets remain post-v1","docs":["docs/spec/current_supported_surface.md","docs/spec/ui_v0.4.0.md","docs/user/wasm_ui_guide.md"]},
    {"id":"ui.toolkit-core","name":"UI Toolkit Core","status":"current","since":"v0.4.0","scope":"production platform-independent UI Toolkit Core contract for tetra.ui.toolkit.v1 with widget model, layout model, accessibility model, event dispatch, state binding/update, and runtime trace artifacts","stability":"rejects metadata-only, runtime-less, native-shell sidecar-only, web-only evidence; no GTK/Qt/OS platform backend production or full cross-platform UI claim","docs":["docs/spec/current_supported_surface.md","docs/spec/ui_toolkit_core.md","docs/spec/ui_v0.4.0.md"]},
    {"id":"wasm.runtime-execution","name":"WASM runtime execution","status":"current","since":"v0.4.0","scope":"production WASI runner and browser-backed wasm32-web execution","stability":"supported with runner/browser availability diagnostics","docs":["docs/spec/current_supported_surface.md","docs/backend/wasm_backend_plan.md","docs/user/wasm_ui_guide.md"]},
    {"id":"language.full-v1-guarantees","name":"v1","status":"planned","scope":"v1","stability":"planned","docs":["docs/spec/v1_scope.md"]},
    {"id":"eco.distributed-network","name":"EcoNet","status":"post-v1","scope":"network","stability":"deferred","docs":["docs/release/post_v1_promotion_checklist.md"]},
    {"id":"language.full-first-class-callables","name":"Callables","status":"current","since":"v0.4.0","scope":"safe by-value first-class callable semantics","stability":"current safe-capture model","docs":["docs/spec/v1_feature_status.md"]}
  ]
}`
	out, err := runManifestValidator(t, manifest)
	if err != nil {
		t.Fatalf("validator failed: %v\n%s", err, out)
	}
}

func TestValidateFeaturesAcceptsMachineReadableCurrentFutureClaims(t *testing.T) {
	features := []featureManifest{
		{ID: "cli.core", Name: "CLI", Status: "current", Since: "v0.2.0", Scope: "core CLI", Stability: "supported", Docs: []string{"docs/spec/current_supported_surface.md"}},
		{ID: "language.flow", Name: "Flow", Status: "current", Since: "v0.2.0", Scope: "flow syntax", Stability: "supported", Docs: []string{"docs/spec/flow_syntax_v1.md"}},
		{ID: "language.generics-mvp", Name: "Generics MVP", Status: "current", Since: "v0.2.0", Scope: "statically monomorphized generic functions with no runtime generic values or dynamic dispatch", Stability: "supported static MVP; generic structs remain future/post-v1", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "language.protocol-conformance-mvp", Name: "Protocol conformance MVP", Status: "current", Since: "v0.2.0", Scope: "checked statically with generic requirement signature shape and no witness tables", Stability: "dynamic dispatch remain post-v1", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "language.callable-mvp", Name: "Callable MVP", Status: "current", Since: "v0.2.0", Scope: "Level 0 callable surface", Stability: "current constrained MVP", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md"}},
		{ID: "targets.wasm-artifact-preflight", Name: "WASM artifact/import preflight", Status: "current", Since: "v0.2.0", Scope: "artifact/import smoke", Stability: "supported", Docs: []string{"docs/backend/wasm_backend_plan.md"}},
		{ID: "stdlib.experimental-mirrors", Name: "Standard-library compatibility mirrors", Status: "current", Since: "v0.4.0", Scope: "production compatibility mirrors forward to lib.core modules", Stability: "stable callers should import lib.core directly", Docs: []string{"docs/spec/stdlib.md", "docs/spec/stdlib_naming_versioning.md", "docs/user/standard_library_guide.md"}},
		{ID: "language.callable-level1", Name: "Callable Level 1", Status: "current", Since: "v0.4.0", Scope: "production non-capturing symbol-backed callable Level 1 with function-typed locals, aliases, callbacks, and symbol-backed returns", Stability: "captured closure escape and full first-class function values remain out of scope", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_feature_status.md"}},
		{ID: "language.enum-payload-match", Name: "Enum payload", Status: "current", Since: "v0.3.0", Scope: "positional enum payload constructors and payload bindings for match/catch/if-let, with exhaustive unguarded enum match/catch", Stability: "nested destructuring patterns and guard expansion remain future/post-v1", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v0_3_scope.md"}},
		{ID: "language.protocol-bound-generics-static", Name: "Static protocol-bound generics", Status: "current", Since: "v0.3.0", Scope: "validated statically during monomorphization with same-module and cross-module impl conformance plus visibility diagnostics", Stability: "calling protocol requirements through generic bounds and dynamic dispatch remain unsupported", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/v0_3_scope.md", "docs/spec/flow_syntax_v1.md"}},
		{ID: "language.ownership-markers-mvp", Name: "Ownership markers MVP", Status: "current", Since: "v0.2.0", Scope: "conservative borrow/inout/consume marker checks with use-after-consume and borrow escape diagnostics", Stability: "supported conservative MVP; not a full SSA lifetime solver", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "language.resource-lifetime-mvp", Name: "Resource lifetime MVP", Status: "current", Since: "v0.2.0", Scope: "conservative resource finalization checks for task handles, task groups, island handles, region-backed slices, and structs containing them, including double-use and ambiguous provenance diagnostics", Stability: "supported conservative MVP; tracks common local scope and control-flow merge cases, but is not a full SSA lifetime solver", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "actors.task-transfer-safety", Name: "Actor/task transfer safety MVP", Status: "current", Since: "v0.2.0", Scope: "conservative actor/task ownership transfer checks for worker entrypoints and use-after-transfer diagnostics", Stability: "supported conservative local MVP; distributed actors remain outside current support", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "language.lifetime-ssa", Name: "Lifetime SSA local join solver", Status: "current", Since: "v0.4.0", Scope: "production SSA-like local lifetime join analysis for ownership consume state, resource finalization state, branch/match/loop flow snapshots, and maybe-consumed diagnostics", Stability: "current local/control-flow solver; richer interprocedural lifetime proofs, broad alias modeling, race proofs, and full formal lifetime guarantees remain under full-v1 scope", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "safety.production-core", Name: "Production safety core", Status: "current", Since: "v0.4.0", Scope: "production local safety model for ownership/lifetime/borrow/consume/inout checks, resource finalization, callable escape diagnostics, effects/capabilities/privacy/consent/budget policy, unsafe boundaries, actor/task transfer safety, and pointer/MMIO/memory capability gates", Stability: "release-gated current profile with explicit diagnostics for unsupported distributed, cryptographic, formal-proof, and runtime-wide guarantees", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/effects_capabilities_privacy_v1.md"}},
		{ID: "language.callable-level2", Name: "Callable Level 2", Status: "current", Since: "v0.4.0", Scope: "production captured closure Level 2 slice with function-typed locals called directly", Stability: "captured callback passing and full first-class callable semantics remain out of scope", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_feature_status.md"}},
		{ID: "ui.metadata-v1", Name: "UI metadata v0.4.0", Status: "current", Since: "v0.4.0", Scope: "production UI metadata contract with deterministic tetra.ui.v0.4.0 JSON", Stability: "web command dispatch; native widgets remain post-v1", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ui_v0.4.0.md", "docs/user/wasm_ui_guide.md"}},
		{ID: "ui.toolkit-core", Name: "UI Toolkit Core", Status: "current", Since: "v0.4.0", Scope: "production platform-independent UI Toolkit Core contract for tetra.ui.toolkit.v1 with widget model, layout model, accessibility model, event dispatch, state binding/update, and runtime trace artifacts", Stability: "rejects metadata-only, runtime-less, native-shell sidecar-only, web-only evidence; no GTK/Qt/OS platform backend production or full cross-platform UI claim", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ui_toolkit_core.md", "docs/spec/ui_v0.4.0.md"}},
		{ID: "wasm.runtime-execution", Name: "WASM runtime execution", Status: "current", Since: "v0.4.0", Scope: "production WASI runner and browser-backed wasm32-web execution", Stability: "supported with runner/browser availability diagnostics", Docs: []string{"docs/spec/current_supported_surface.md", "docs/backend/wasm_backend_plan.md", "docs/user/wasm_ui_guide.md"}},
		{ID: "language.full-v1-guarantees", Name: "v1", Status: "planned", Scope: "v1", Stability: "planned", Docs: []string{"docs/spec/v1_scope.md"}},
		{ID: "eco.distributed-network", Name: "EcoNet", Status: "post-v1", Scope: "network", Stability: "deferred", Docs: []string{"docs/release/post_v1_promotion_checklist.md"}},
		{ID: "language.full-first-class-callables", Name: "Callables", Status: "current", Since: "v0.4.0", Scope: "safe by-value first-class callable semantics", Stability: "current safe-capture model", Docs: []string{"docs/spec/v1_feature_status.md"}},
	}
	if err := validateFeatures(features); err != nil {
		t.Fatalf("validateFeatures: %v", err)
	}
}

func TestValidateFeaturesRejectsFutureStatusPromotionWithoutRegistryUpdate(t *testing.T) {
	features := []featureManifest{
		{ID: "cli.core", Name: "CLI", Status: "current", Since: "v0.2.0", Scope: "core CLI", Stability: "supported", Docs: []string{"docs/spec/current_supported_surface.md"}},
		{ID: "language.flow", Name: "Flow", Status: "current", Since: "v0.2.0", Scope: "flow syntax", Stability: "supported", Docs: []string{"docs/spec/flow_syntax_v1.md"}},
		{ID: "language.generics-mvp", Name: "Generics MVP", Status: "current", Since: "v0.2.0", Scope: "statically monomorphized generic functions with no runtime generic values or dynamic dispatch", Stability: "supported static MVP; generic structs remain future/post-v1", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "language.protocol-conformance-mvp", Name: "Protocol conformance MVP", Status: "current", Since: "v0.2.0", Scope: "checked statically with generic requirement signature shape and no witness tables", Stability: "dynamic dispatch remain post-v1", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "language.callable-mvp", Name: "Callable MVP", Status: "current", Since: "v0.2.0", Scope: "Level 0 callable surface", Stability: "current constrained MVP", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md"}},
		{ID: "targets.wasm-artifact-preflight", Name: "WASM artifact/import preflight", Status: "current", Since: "v0.2.0", Scope: "artifact/import smoke", Stability: "supported", Docs: []string{"docs/backend/wasm_backend_plan.md"}},
		{ID: "stdlib.experimental-mirrors", Name: "Standard-library compatibility mirrors", Status: "current", Since: "v0.4.0", Scope: "production compatibility mirrors forward to lib.core modules", Stability: "stable callers should import lib.core directly", Docs: []string{"docs/spec/stdlib.md", "docs/spec/stdlib_naming_versioning.md", "docs/user/standard_library_guide.md"}},
		{ID: "language.callable-level1", Name: "Callable Level 1", Status: "current", Since: "v0.4.0", Scope: "production non-capturing symbol-backed callable Level 1 with function-typed locals, aliases, callbacks, and symbol-backed returns", Stability: "captured closure escape and full first-class function values remain out of scope", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_feature_status.md"}},
		{ID: "language.enum-payload-match", Name: "Enum payload", Status: "current", Since: "v0.3.0", Scope: "positional enum payload constructors and payload bindings for match/catch/if-let, with exhaustive unguarded enum match/catch", Stability: "nested destructuring patterns and guard expansion remain future/post-v1", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v0_3_scope.md"}},
		{ID: "language.protocol-bound-generics-static", Name: "Static protocol-bound generics", Status: "current", Since: "v0.3.0", Scope: "validated statically during monomorphization with same-module and cross-module impl conformance plus visibility diagnostics", Stability: "calling protocol requirements through generic bounds and dynamic dispatch remain unsupported", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/v0_3_scope.md", "docs/spec/flow_syntax_v1.md"}},
		{ID: "language.ownership-markers-mvp", Name: "Ownership markers MVP", Status: "current", Since: "v0.2.0", Scope: "conservative borrow/inout/consume marker checks with use-after-consume and borrow escape diagnostics", Stability: "supported conservative MVP; not a full SSA lifetime solver", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "language.resource-lifetime-mvp", Name: "Resource lifetime MVP", Status: "current", Since: "v0.2.0", Scope: "conservative resource finalization checks for task handles, task groups, island handles, region-backed slices, and structs containing them, including double-use and ambiguous provenance diagnostics", Stability: "supported conservative MVP; tracks common local scope and control-flow merge cases, but is not a full SSA lifetime solver", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "actors.task-transfer-safety", Name: "Actor/task transfer safety MVP", Status: "current", Since: "v0.2.0", Scope: "conservative actor/task ownership transfer checks for worker entrypoints and use-after-transfer diagnostics", Stability: "supported conservative local MVP; distributed actors remain outside current support", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "safety.production-core", Name: "Production safety core", Status: "current", Since: "v0.4.0", Scope: "production local safety model for ownership/lifetime/borrow/consume/inout checks, resource finalization, callable escape diagnostics, effects/capabilities/privacy/consent/budget policy, unsafe boundaries, actor/task transfer safety, and pointer/MMIO/memory capability gates", Stability: "release-gated current profile with explicit diagnostics for unsupported distributed, cryptographic, formal-proof, and runtime-wide guarantees", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/effects_capabilities_privacy_v1.md"}},
		{ID: "language.lifetime-ssa", Name: "Lifetime SSA solver", Status: "planned", Scope: "stale planned lifetime solver fixture", Stability: "unsupported stale fixture", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ownership_v1.md", "docs/spec/v1_scope.md"}},
		{ID: "language.callable-level2", Name: "Callable Level 2", Status: "current", Since: "v0.4.0", Scope: "production captured closure Level 2 slice with function-typed locals called directly", Stability: "captured callback passing and full first-class callable semantics remain out of scope", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/flow_syntax_v1.md", "docs/spec/v1_feature_status.md"}},
		{ID: "ui.metadata-v1", Name: "UI metadata v0.4.0", Status: "current", Since: "v0.4.0", Scope: "production UI metadata contract with deterministic tetra.ui.v0.4.0 JSON", Stability: "web command dispatch; native widgets remain post-v1", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ui_v0.4.0.md", "docs/user/wasm_ui_guide.md"}},
		{ID: "ui.toolkit-core", Name: "UI Toolkit Core", Status: "current", Since: "v0.4.0", Scope: "production platform-independent UI Toolkit Core contract for tetra.ui.toolkit.v1 with widget model, layout model, accessibility model, event dispatch, state binding/update, and runtime trace artifacts", Stability: "rejects metadata-only, runtime-less, native-shell sidecar-only, web-only evidence; no GTK/Qt/OS platform backend production or full cross-platform UI claim", Docs: []string{"docs/spec/current_supported_surface.md", "docs/spec/ui_toolkit_core.md", "docs/spec/ui_v0.4.0.md"}},
		{ID: "wasm.runtime-execution", Name: "WASM runtime execution", Status: "current", Since: "v0.4.0", Scope: "production WASI runner and browser-backed wasm32-web execution", Stability: "supported with runner/browser availability diagnostics", Docs: []string{"docs/spec/current_supported_surface.md", "docs/backend/wasm_backend_plan.md", "docs/user/wasm_ui_guide.md"}},
		{ID: "language.full-v1-guarantees", Name: "v1", Status: "planned", Scope: "v1", Stability: "planned", Docs: []string{"docs/spec/v1_scope.md"}},
		{ID: "eco.distributed-network", Name: "EcoNet", Status: "post-v1", Scope: "network", Stability: "deferred", Docs: []string{"docs/release/post_v1_promotion_checklist.md"}},
		{ID: "language.full-first-class-callables", Name: "Callables", Status: "current", Since: "v0.4.0", Scope: "safe by-value first-class callable semantics", Stability: "current safe-capture model", Docs: []string{"docs/spec/v1_feature_status.md"}},
	}
	err := validateFeatures(features)
	if err == nil {
		t.Fatalf("expected future status promotion failure")
	}
	if !strings.Contains(err.Error(), "language.lifetime-ssa") || !strings.Contains(err.Error(), "want current") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateManifestRejectsNullTargets(t *testing.T) {
	manifest := `{"compiler_version":"v0.6.0","targets":null,"builtins":[],"runtime_abi":{}}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "targets must be an array") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsUnknownFields(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf","exe_ext":"","collect_imports":false},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe","exe_ext":".exe","collect_imports":true},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho","exe_ext":"","collect_imports":false}
  ],
  "builtins": [{"name":"core.print","return_type":"i32","unsafe_policy":"never","extra":true}],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","macos-x64","windows-x64"],
    "actors_required_symbols": ["__tetra_entry","__tetra_actor_spawn","__tetra_actor_send","__tetra_actor_send_msg","__tetra_actor_send_begin","__tetra_actor_send_slot","__tetra_actor_send_commit","__tetra_actor_recv","__tetra_actor_recv_msg","__tetra_actor_recv_poll","__tetra_actor_recv_until","__tetra_actor_recv_msg_until","__tetra_actor_recv_begin","__tetra_actor_recv_slot","__tetra_actor_recv_count","__tetra_actor_self","__tetra_actor_sender","__tetra_actor_yield_now"],
    "time_required_symbols": ["__tetra_time_now_ms","__tetra_sleep_ms","__tetra_sleep_until_ms","__tetra_deadline_ms","__tetra_timer_ready_ms"],
    "filesystem_required_symbols": ["__tetra_fs_exists"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "unknown field") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsDuplicateBuiltin(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho"},
    {"triple":"wasm32-wasi","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm"},
    {"triple":"wasm32-web","os":"web","arch":"wasm32","abi":"web","format":"wasm"}
  ],
  "builtins": [
    {"name":"core.print","return_type":"i32","unsafe_policy":"never"},
    {"name":"core.print","return_type":"i32","unsafe_policy":"never"}
  ],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64","macos-x64"],
    "actors_required_symbols": ["__tetra_entry","__tetra_actor_spawn","__tetra_actor_send","__tetra_actor_send_msg","__tetra_actor_send_begin","__tetra_actor_send_slot","__tetra_actor_send_commit","__tetra_actor_recv","__tetra_actor_recv_msg","__tetra_actor_recv_poll","__tetra_actor_recv_until","__tetra_actor_recv_msg_until","__tetra_actor_recv_begin","__tetra_actor_recv_slot","__tetra_actor_recv_count","__tetra_actor_self","__tetra_actor_sender","__tetra_actor_yield_now"],
    "time_required_symbols": ["__tetra_time_now_ms","__tetra_sleep_ms","__tetra_sleep_until_ms","__tetra_deadline_ms","__tetra_timer_ready_ms"],
    "filesystem_required_symbols": ["__tetra_fs_exists"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "duplicate builtin core.print") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsMissingRuntimeSymbols(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho"},
    {"triple":"wasm32-wasi","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm"},
    {"triple":"wasm32-web","os":"web","arch":"wasm32","abi":"web","format":"wasm"}
  ],
  "builtins": [{"name":"core.print","return_type":"i32","unsafe_policy":"never"}],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64","macos-x64"],
    "actors_required_symbols": [],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "actors_required_symbols must not be empty") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsMissingTimeRuntimeSymbols(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho"},
    {"triple":"wasm32-wasi","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm"},
    {"triple":"wasm32-web","os":"web","arch":"wasm32","abi":"web","format":"wasm"}
  ],
  "builtins": [{"name":"core.print","return_type":"i32","unsafe_policy":"never"}],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64","macos-x64"],
    "actors_required_symbols": ["__tetra_entry","__tetra_actor_spawn","__tetra_actor_send","__tetra_actor_send_msg","__tetra_actor_send_begin","__tetra_actor_send_slot","__tetra_actor_send_commit","__tetra_actor_recv","__tetra_actor_recv_msg","__tetra_actor_recv_poll","__tetra_actor_recv_until","__tetra_actor_recv_msg_until","__tetra_actor_recv_begin","__tetra_actor_recv_slot","__tetra_actor_recv_count","__tetra_actor_self","__tetra_actor_sender","__tetra_actor_yield_now"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "time_required_symbols must not be empty") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsInvalidUnsafePolicy(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho"},
    {"triple":"wasm32-wasi","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm"},
    {"triple":"wasm32-web","os":"web","arch":"wasm32","abi":"web","format":"wasm"}
  ],
  "builtins": [{"name":"core.print","return_type":"i32","unsafe_policy":"sometimes"}],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64","macos-x64"],
    "actors_required_symbols": ["__tetra_entry","__tetra_actor_spawn","__tetra_actor_send","__tetra_actor_send_msg","__tetra_actor_send_begin","__tetra_actor_send_slot","__tetra_actor_send_commit","__tetra_actor_recv","__tetra_actor_recv_msg","__tetra_actor_recv_poll","__tetra_actor_recv_until","__tetra_actor_recv_msg_until","__tetra_actor_recv_begin","__tetra_actor_recv_slot","__tetra_actor_recv_count","__tetra_actor_self","__tetra_actor_sender","__tetra_actor_yield_now"],
    "time_required_symbols": ["__tetra_time_now_ms","__tetra_sleep_ms","__tetra_sleep_until_ms","__tetra_deadline_ms","__tetra_timer_ready_ms"],
    "filesystem_required_symbols": ["__tetra_fs_exists"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "invalid unsafe_policy") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsPartialTargetSurface(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"}
  ],
  "builtins": [{"name":"core.print","return_type":"i32","unsafe_policy":"never"}],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64"],
    "actors_required_symbols": ["__tetra_entry","__tetra_actor_spawn","__tetra_actor_send","__tetra_actor_send_msg","__tetra_actor_send_begin","__tetra_actor_send_slot","__tetra_actor_send_commit","__tetra_actor_recv","__tetra_actor_recv_msg","__tetra_actor_recv_poll","__tetra_actor_recv_until","__tetra_actor_recv_msg_until","__tetra_actor_recv_begin","__tetra_actor_recv_slot","__tetra_actor_recv_count","__tetra_actor_self","__tetra_actor_sender","__tetra_actor_yield_now"],
    "time_required_symbols": ["__tetra_time_now_ms","__tetra_sleep_ms","__tetra_sleep_until_ms","__tetra_deadline_ms","__tetra_timer_ready_ms"],
    "filesystem_required_symbols": ["__tetra_fs_exists"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "targets got") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsPartialRuntimeABI(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho"},
    {"triple":"wasm32-wasi","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm"},
    {"triple":"wasm32-web","os":"web","arch":"wasm32","abi":"web","format":"wasm"}
  ],
  "builtins": [{"name":"core.print","return_type":"i32","unsafe_policy":"never"}],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64","macos-x64"],
    "actors_required_symbols": ["__tetra_entry","__tetra_actor_spawn"],
    "time_required_symbols": ["__tetra_time_now_ms","__tetra_sleep_ms","__tetra_sleep_until_ms","__tetra_deadline_ms","__tetra_timer_ready_ms"],
    "filesystem_required_symbols": ["__tetra_fs_exists"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "actors_required_symbols got") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsUnsortedTargets(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"},
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho"},
    {"triple":"wasm32-wasi","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm"},
    {"triple":"wasm32-web","os":"web","arch":"wasm32","abi":"web","format":"wasm"}
  ],
  "builtins": [{"name":"core.print","return_type":"i32","unsafe_policy":"never"}],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64","macos-x64"],
    "actors_required_symbols": ["__tetra_entry","__tetra_actor_spawn","__tetra_actor_send","__tetra_actor_send_msg","__tetra_actor_send_begin","__tetra_actor_send_slot","__tetra_actor_send_commit","__tetra_actor_recv","__tetra_actor_recv_msg","__tetra_actor_recv_poll","__tetra_actor_recv_until","__tetra_actor_recv_msg_until","__tetra_actor_recv_begin","__tetra_actor_recv_slot","__tetra_actor_recv_count","__tetra_actor_self","__tetra_actor_sender","__tetra_actor_yield_now"],
    "time_required_symbols": ["__tetra_time_now_ms","__tetra_sleep_ms","__tetra_sleep_until_ms","__tetra_deadline_ms","__tetra_timer_ready_ms"],
    "filesystem_required_symbols": ["__tetra_fs_exists"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "targets must follow buildable target order") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func TestValidateManifestRejectsUnsortedBuiltins(t *testing.T) {
	manifest := `{
  "compiler_version": "v0.6.0",
  "targets": [
    {"triple":"linux-x64","os":"linux","arch":"x64","abi":"sysv","format":"elf"},
    {"triple":"windows-x64","os":"windows","arch":"x64","abi":"win64","format":"pe"},
    {"triple":"macos-x64","os":"macos","arch":"x64","abi":"sysv","format":"macho"},
    {"triple":"wasm32-wasi","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm"},
    {"triple":"wasm32-web","os":"web","arch":"wasm32","abi":"web","format":"wasm"},
    {"triple":"wasm32-wasi","os":"wasi","arch":"wasm32","abi":"wasi","format":"wasm"},
    {"triple":"wasm32-web","os":"web","arch":"wasm32","abi":"web","format":"wasm"}
  ],
  "builtins": [
    {"name":"core.z","return_type":"i32","unsafe_policy":"never"},
    {"name":"core.a","return_type":"i32","unsafe_policy":"never"}
  ],
  "runtime_abi": {
    "reserved_prefix": "__tetra_",
    "actors_supported_targets": ["linux-x64","windows-x64","macos-x64"],
    "actors_required_symbols": ["__tetra_entry","__tetra_actor_spawn","__tetra_actor_send","__tetra_actor_send_msg","__tetra_actor_send_begin","__tetra_actor_send_slot","__tetra_actor_send_commit","__tetra_actor_recv","__tetra_actor_recv_msg","__tetra_actor_recv_poll","__tetra_actor_recv_until","__tetra_actor_recv_msg_until","__tetra_actor_recv_begin","__tetra_actor_recv_slot","__tetra_actor_recv_count","__tetra_actor_self","__tetra_actor_sender","__tetra_actor_yield_now"],
    "time_required_symbols": ["__tetra_time_now_ms","__tetra_sleep_ms","__tetra_sleep_until_ms","__tetra_deadline_ms","__tetra_timer_ready_ms"],
    "filesystem_required_symbols": ["__tetra_fs_exists"],
    "actors_program_glue_symbols": ["__tetra_actor_dispatch","__tetra_actor_main_entry_id"]
  }
}`
	out, err := runManifestValidator(t, manifest)
	if err == nil {
		t.Fatalf("expected validator failure\n%s", out)
	}
	if !strings.Contains(string(out), "builtins must be sorted") {
		t.Fatalf("unexpected output:\n%s", out)
	}
}

func runManifestValidator(t *testing.T, manifest string) ([]byte, error) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(path, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "run", ".", "--manifest", path)
	cmd.Dir = "."
	return cmd.CombinedOutput()
}
