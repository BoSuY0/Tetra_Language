## reviewed_commit

- Reviewed worktree: `/home/tetra/.codex/worktrees/Tetra_Language/safety-semantics-closure-v1`.
- Reviewed commit: `3d101fbc3e1d8d9a9710c44725372ea086287c9c`.
- Source plan read: `/home/tetra/Downloads/Tetra_Safety_Semantics_Closure_v1_Implementation_Plan.md`.
- `graphify-out/` is not present in this worktree, so the audit uses normal read-only repo inspection and concrete file evidence.
- There is no `compiler/abi_suite.go` file at this commit; the current equivalent is split between the facade in `compiler/compiler_evidence_gates.go:41-52`, the target runners in `compiler/compiler_evidence_gates.go:54-90`, `compiler/compiler_evidence_gates.go:206-286`, and the implementation in `compiler/internal/abisuite/abisuite_core.go`.

## repr_c_status

- The frontend records struct representation explicitly: `StructReprDefault = "tetra"`, `StructReprC = "C"`, and `StructDecl.Repr` lives in the AST at `compiler/internal/frontend/frontend_core.go:178-190`.
- Parsing accepts `repr(C)` and rejects unsupported representation strings before parsing the struct body at `compiler/internal/frontend/frontend_core.go:5250-5272`.
- Semantics preserves the representation in `TypeInfo.Repr` through `structReprOrDefault(ctx.decl.Repr)` at `compiler/internal/semantics/semantics_checker.go:423-430`; the model field is defined at `compiler/internal/semantics/model/types.go:266-276`.
- Exported default-layout structs are rejected by the export ABI guard: `exportedDefaultLayoutABIExposureForType` flags non-`repr(C)` structs at `compiler/internal/semantics/semantics_checker.go:1983-2036`, and `validateExportedOpaqueABISignature` rejects them for params and returns at `compiler/internal/semantics/semantics_checker.go:13682-13705`.
- Semantics tests confirm the explicit gate: a default-layout exported struct is rejected and a `repr(C)` exported struct passes the semantics-level repr gate at `compiler/internal/semantics/semantics_suite_test.go:2217-2268`.
- Native target build validation still rejects aggregate FFI params/returns for supported native targets, even after the semantics repr gate, through `validateTargetExportedFFIABI` at `compiler/compiler_build_runtime.go:1520-1588` and aggregate detection at `compiler/internal/abisuite/abisuite_core.go:4337-4358`. Current status: `repr(C)` exists and is necessary for public aggregate exposure, but it is not sufficient to permit native aggregate C ABI export.

## export_rejection_matrix

| Export surface | Current behavior | Evidence |
| --- | --- | --- |
| `@export` syntax | Only `@export("...")` is accepted; duplicate, empty, and unknown attributes are rejected. | `compiler/internal/frontend/frontend_core.go:2880-2913` |
| Reserved `core` namespace | Export names equal to `core` or starting `core.` are rejected. | `compiler/internal/semantics/semantics_checker.go:823-830` |
| Reserved `__tetra_` names | Non-internal module names cannot export reserved `__tetra_` symbols. The namespace check is broader than the runtime exemption because it allows modules starting with `__`, while the ABI exemption is narrower. | `compiler/internal/semantics/semantics_checker.go:831-836`, `compiler/internal/semantics/semantics_checker.go:1958-1963` |
| Duplicate export names | Duplicate exported symbol names in a module are rejected. | `compiler/internal/semantics/semantics_checker.go:837-842` |
| Generic exports | Generic functions with `@export` are rejected; tests cover this path. | `compiler/internal/semantics/semantics_checker.go:864-880`, `compiler/tests/safety/plan250_runtime_exports_test.go:889-897` |
| Typed throwing exports | Exported functions with typed `throws` are rejected; tests cover this path. | `compiler/internal/semantics/semantics_checker.go:1605-1630`, `compiler/internal/semantics/semantics_checker.go:1872-1887`, `compiler/tests/safety/plan250_runtime_exports_test.go:909-940` |
| Ownership-marked exported params | Exported params with ownership markers are rejected. | `compiler/internal/semantics/semantics_checker.go:13538-13549`, `compiler/tests/safety/plan250_runtime_exports_test.go:943-995` |
| `cap.io`, `cap.mem` | Capability tokens are rejected directly and recursively in export ABI exposure. | `compiler/internal/semantics/semantics_checker.go:2193-2200`, `compiler/internal/semantics/semantics_checker.go:13550-13558`, `compiler/internal/semantics/semantics_checker.go:13620-13627` |
| Consent tokens | Direct and aggregate forgeable consent token export exposure is rejected. | `compiler/internal/semantics/semantics_checker.go:1818-1870`, `compiler/tests/safety/plan250_runtime_exports_test.go:831-887` |
| `island` handle | Island handles are rejected for params and returns. | `compiler/internal/semantics/semantics_checker.go:2148-2152`, `compiler/internal/semantics/semantics_checker.go:13559-13567`, `compiler/internal/semantics/semantics_checker.go:13628-13635` |
| Function pointer ABI values | `fnptr` and function-typed ABI values are rejected; x86/x32 build diagnostics also treat function-pointer exports as an unverified pointer C ABI boundary. | `compiler/internal/semantics/semantics_checker.go:2154-2158`, `compiler/internal/semantics/semantics_checker.go:13568-13576`, `compiler/compiler_build_runtime.go:1606-1629`, `compiler/internal/abisuite/abisuite_core.go:4320-4335` |
| `Bool` | User exports reject `Bool` as unnormalized bool unless the symbol is an internal runtime export. | `compiler/internal/semantics/semantics_checker.go:2160-2170`, `compiler/internal/semantics/semantics_checker.go:13577-13588`, `compiler/internal/semantics/semantics_checker.go:13644-13654` |
| Raw views | `String`, slices, and fixed arrays are rejected as raw string/slice/fixed-array views. | `compiler/internal/semantics/semantics_checker.go:2172-2190`, `compiler/internal/semantics/semantics_checker.go:13589-13598`, `compiler/internal/semantics/semantics_checker.go:13655-13663` |
| Runtime handles | `actor`, `task.group`, and `task.i32` are rejected except for internal runtime exports. | `compiler/internal/semantics/semantics_checker.go:2202-2209`, `compiler/internal/semantics/semantics_checker.go:13599-13607`, `compiler/internal/semantics/semantics_checker.go:13664-13671` |
| Optional values | Optionals are treated as forgeable optional presence tags in recursive export exposure. | `compiler/internal/semantics/semantics_checker.go:1989-1994`, `compiler/internal/semantics/semantics_checker.go:2059-2066` |
| Enums | Enums recurse through payloads and then expose a forgeable enum discriminant. | `compiler/internal/semantics/semantics_checker.go:2099-2135` |
| Default-layout structs | Exported default-layout structs are rejected; `repr(C)` is the explicit semantics gate. | `compiler/internal/semantics/semantics_checker.go:1983-2036`, `compiler/internal/semantics/semantics_checker.go:13682-13705` |
| Native aggregate FFI | Native target build validation rejects aggregate params/returns for `linux-x86`, `linux-x64`, `linux-x32`, `macos-x64`, and `windows-x64`. | `compiler/internal/abisuite/abisuite_core.go:4302-4308`, `compiler/compiler_build_runtime.go:1520-1588`, `compiler/compiler_suite_test.go:14258-14307` |
| Target-layout-only scalar spellings | Unsupported source scalar spellings are rejected before they become target ABI claims. | `compiler/internal/semantics/semantics_core.go:4886-4891`, `compiler/compiler_suite_test.go:14701-14826` |

## allowed_scalar_ffi_types

- Confirmed build-smoked scalar FFI types are target-specific, not a single global allowlist.
- `c_int` and `c_uint` are verified by object smoke checks for `linux-x86`, `linux-x64`, and `linux-x32` at `compiler/internal/abisuite/abisuite_core.go:313-393`; the evidence gates wire these checks for x86, x64, and x32 at `compiler/compiler_evidence_gates.go:72-79`, `compiler/compiler_evidence_gates.go:211-227`, and `compiler/compiler_evidence_gates.go:273-279`.
- ILP32 native/libc spellings `usize`, `isize`, `size_t`, `ssize_t`, `native_int`, `native_uint`, `c_long`, and `c_ulong` are added only when ILP32 native scalars are enabled at `compiler/internal/semantics/semantics_core.go:4725-4746`; the facade enables that only for `linux-x86` and `linux-x32` at `compiler/compiler_facade.go:811-815`; object smokes cover those spellings at `compiler/internal/abisuite/abisuite_core.go:399-485`.
- Base source scalar names include `i32`, `i64`, `u8`, `u16`, `c_int`, `c_uint`, `bool`, `ptr`, and `fnptr` at `compiler/internal/semantics/semantics_core.go:4612-4620`, but export ABI policy narrows that set: `bool` is rejected for user exports as unnormalized bool and `fnptr` is rejected as a function-typed ABI value at `compiler/internal/semantics/semantics_checker.go:13568-13588`.
- The target layout table knows additional scalar spellings such as `i8`, `i16`, `u32`, `u64`, `f32`, and `f64` at `compiler/target/target.go:786-805`, but the current source checker marks several of these as target-layout-only spellings at `compiler/internal/semantics/semantics_core.go:4886-4891`; diagnostics are covered by `compiler/compiler_suite_test.go:14701-14826`.
- Runtime ABI docs match this target-specific posture: Linux scalar wrappers include canonical pointers plus `c_int`/`c_uint`, ILP32 Linux adds native/libc names, and function-pointer spellings remain rejected until wrapper verification at `docs/spec/runtime/runtime_abi.md:32-47` and `docs/spec/runtime/runtime_abi.md:104-109`.

## allowed_pointer_ffi_types

- `ptr` is the broadest confirmed pointer spelling: x86/x32 object smokes verify `ptr -> ptr`, and x64 has a regression smoke for a `ptr` parameter at `compiler/internal/abisuite/abisuite_core.go:221-310` and `compiler/internal/abisuite/abisuite_core.go:4588-4620`.
- `rawptr`, `nullable_ptr`, and `ref` are added as public one-slot pointer aliases only with ILP32 native scalar enablement at `compiler/internal/semantics/semantics_core.go:4725-4746`; x86/x32 object smokes cover them at `compiler/internal/abisuite/abisuite_core.go:221-310`.
- `nullable_ptr` accepts null literal compatibility together with `ptr` and `rawptr` at `compiler/internal/semantics/semantics_core.go:4939-4945`; the pointer smoke includes a `nullable_ptr` null return at `compiler/internal/abisuite/abisuite_core.go:237-255`.
- `ref` is not nullable in the current checks: the ref null-return diagnostic check rejects returning `0` as `ref` at `compiler/internal/abisuite/abisuite_core.go:487-521`.
- Function pointer spellings are not allowed pointer FFI types. The current pointer boundary gate only flags `fnptr` and `fn(...)` forms when enabled at `compiler/internal/abisuite/abisuite_core.go:4320-4335`, and tests assert this gate does not cover `ptr`, `rawptr`, `nullable_ptr`, or `ref` at `compiler/compiler_suite_test.go:14213-14256`.
- Current code therefore has confirmed pointer return support in x86/x32 smokes for `ptr`, `rawptr`, `nullable_ptr`, and `ref`; that is current-state evidence, not a claim that the planned external-pointer quarantine is implemented.

## internal_runtime_export_exemption

- The narrow semantics exemption is `ExportName` starting with `__tetra_` and module exactly `__rt` or starting `__rt.` at `compiler/internal/semantics/semantics_checker.go:1958-1963`.
- That exemption only relaxes user-facing export guards for `Bool` and opaque runtime handles via `allowRuntimeHandles`; capability tokens, island handles, function-typed values, raw views, and default-layout structs remain guarded outside that exemption at `compiler/internal/semantics/semantics_checker.go:13527-13707`.
- Build-time target FFI validation also skips only internal runtime exported symbols, using the same `__tetra_` plus `__rt`/`__rt.*` shape at `compiler/compiler_build_runtime.go:1455-1507`, `compiler/compiler_build_runtime.go:1520-1588`, and `compiler/compiler_build_runtime.go:1652-1655`.
- Tests confirm that `module __rt.actors_sysv` can export reserved runtime symbols carrying a runtime handle and `Bool`, while non-reserved exports in the same runtime module still reject runtime handle and bool exposure at `compiler/tests/safety/plan250_runtime_exports_test.go:997-1044`.
- Runtime ABI docs describe the same reserved internal ABI surface as compiler-owned runtime exports, not arbitrary user aggregate FFI, at `docs/spec/runtime/runtime_abi.md:14-22` and `docs/spec/runtime/runtime_abi.md:49-51`.

## external_pointer_memoryfacts

- Runtime ABI domain modeling includes an external memory domain: `DomainExternal` is defined at `compiler/internal/runtimeabi/memory_domain.go:8-17`, and `ExternalMemoryDomain(ownerID, lifetime, requested, reserved)` creates an external owner with default lifetime `external` at `compiler/internal/runtimeabi/memory_domain.go:99-116`.
- Raw pointer bounds modeling distinguishes external unknown pointers and raw slices: `UnknownRawPointerBounds` uses `checked_external_unknown` and `RawSliceBoundsExternalUnknown` uses `external_unknown` at `compiler/internal/runtimeabi/raw_pointer_bounds.go:10-35`.
- Derived raw pointers and raw slices require verified allocation roots; otherwise they fall back to external unknown status at `compiler/internal/runtimeabi/raw_pointer_bounds.go:120-129` and `compiler/internal/runtimeabi/raw_pointer_bounds.go:226-255`.
- The memoryfacts vocabulary includes `checked_external_unknown`, `external_unknown`, and `ffi_pointer_external_unknown` at `compiler/internal/memoryfacts/vocabulary.go:84-86`, plus FFI retention/noalias/safe-wrapper facts at `compiler/internal/memoryfacts/vocabulary.go:129-132`.
- PLIR memoryfact extraction derives `ffi_pointer_external_unknown`, `external_pointer_provenance_rejected`, `ffi_call_may_retain_borrow`, and `ffi_noalias_invalidated_by_external_call` for unsafe FFI-like operations at `compiler/internal/memoryfacts/fromplir/from_plir_summary.go:392-604`.
- The unsafe runtime spec documents these as conservative report facts: unknown raw pointers remain external/unknown, external calls may retain borrows, safe wrapper promotion is rejected without a compiler-owned contract, and broad noalias assumptions are invalidated at `docs/spec/runtime/unsafe.md:160-179`.
- Confirmed current-state gap: the reviewed export/FFI path does not contain an explicit `ExternalPointerContractV1` or source-level exported-pointer quarantine tying user `@export` pointer params to call lifetime, no-escape, no global storage, no closure capture, no actor/task transfer, no await crossing, or user pointer return rejection.

## object_metadata_version

- TOBJ object format is currently magic `TOBJ` with `objectVersion = 5`, memory plan schema v2, and memory lowering schema v2 at `compiler/internal/format/tobj/object.go:11-18`.
- Object-level metadata includes target, module, compiler version, public API hash, memory plan/lowering schemas and digests, source hash, world signature hash, code/data, symbols, and relocs at `compiler/internal/format/tobj/object.go:21-35`.
- The writer serializes version 5 metadata including compiler/API hashes and memory plan/lowering schema/digests at `compiler/internal/format/tobj/object.go:82-119`.
- The reader accepts versions 2 through 5 and reads v5 memory plan/lowering metadata only for v5+ objects at `compiler/internal/format/tobj/object.go:183-242`.
- Build output populates compiler version, public API hash, memory plan/lowering schemas and digests, source hash, and world signature hash before writing TOBJ at `compiler/compiler_build_runtime.go:1118-1143`.
- Roundtrip tests cover v5 metadata, symbols, and relocations at `compiler/internal/format/tobj/object_test.go:14-65`; link-time compiler version mismatch is guarded at `compiler/compiler_build_runtime.go:1046-1070`.

## object_symbol_metadata

- Current symbol metadata is limited to `Name`, `Offset`, `HasSignature`, `ParamSlots`, and `ReturnSlots` at `compiler/internal/format/tobj/object.go:38-44`.
- The writer emits symbol count, name, offset, and optional signature slot counts at `compiler/internal/format/tobj/object.go:132-152`; the reader reconstructs the same fields for v4+ objects at `compiler/internal/format/tobj/object.go:275-306`.
- Symbol validation checks non-empty names, code-range offsets, and signature slot bounds at `compiler/internal/format/tobj/object.go:366-384`.
- Backends emit this symbol signature metadata from lowered IR slot counts for x64 and x86 at `compiler/internal/backend/x64obj/builder.go:215-229` and `compiler/internal/backend/linux_x86/codegen_emitfunc.go:69-83`.
- Confirmed current-state gap: no per-symbol semantic contract digest, external pointer contract digest, or full function-contract metadata is present in the TOBJ symbol record. Current metadata is object-level digests plus per-symbol slot signatures.

## remaining_lifetime_ambiguities

- Pointer returns are currently build-smoked for x86/x32: the pointer FFI smoke includes exported `ptr`, `rawptr`, `nullable_ptr`, and `ref` return wrappers at `compiler/internal/abisuite/abisuite_core.go:237-255` and verifies their exported symbol signatures at `compiler/internal/abisuite/abisuite_core.go:275-309`.
- x64 pointer FFI evidence inspected here is narrower: it smokes a `ptr` parameter regression at `compiler/internal/abisuite/abisuite_core.go:4588-4620`, not a broad pointer return matrix.
- The target pointer-boundary gate currently detects only function pointer shapes (`fnptr` and `fn(...)`) when enabled, not ordinary pointer scalars, at `compiler/internal/abisuite/abisuite_core.go:4320-4335`; tests explicitly assert `ptr`, `rawptr`, `nullable_ptr`, and `ref` are not covered by that gate at `compiler/compiler_suite_test.go:14213-14256`.
- Memoryfacts conservatively record external/unknown provenance and may-retain/noalias invalidation facts for unsafe/FFI operations at `compiler/internal/memoryfacts/fromplir/from_plir_summary.go:392-604`, but the reviewed export layer does not currently connect those facts to a call-scoped user-export pointer lifetime contract.
- Current lifetime status is therefore ambiguous for the planned closure criteria: the compiler records conservative facts about unknown/external pointers, but user `@export` pointer parameters and returns are not represented by an explicit external pointer contract in the reviewed export ABI path.

## confirmed_gaps

- No single fail-closed FFI/export classifier was found. Current behavior is distributed across frontend export parsing, semantic export namespace/opaque/throwing/consent guards, build-time target FFI AST/ABI guards, and abisuite target helpers: `compiler/internal/frontend/frontend_core.go:2880-2913`, `compiler/internal/semantics/semantics_checker.go:823-880`, `compiler/internal/semantics/semantics_checker.go:13527-13707`, `compiler/compiler_build_runtime.go:1455-1588`, and `compiler/internal/abisuite/abisuite_core.go:4302-4358`.
- No `ExternalPointerContractV1` or equivalent serialized call-lifetime/no-escape pointer contract was found in the reviewed export, runtime ABI, TOBJ, or backend paths. Existing external-pointer facts are conservative memoryfacts/report facts, not an exported ABI contract: `compiler/internal/memoryfacts/vocabulary.go:84-86`, `compiler/internal/memoryfacts/vocabulary.go:129-132`, and `docs/spec/runtime/unsafe.md:160-179`.
- User exported external pointer returns are not rejected in the current x86/x32 object smoke surface; they are positively build-smoked for `ptr`, `rawptr`, `nullable_ptr`, and `ref` at `compiler/internal/abisuite/abisuite_core.go:221-310`.
- TOBJ v5 has object-level compiler/API/memory/source/world metadata, but per-symbol metadata remains name/offset/signature slots only, with no symbol-level function contract or external pointer contract digest at `compiler/internal/format/tobj/object.go:21-44`.
- `repr(C)` is implemented and enforced as an explicit semantics gate, but public native aggregate C ABI export is still rejected by target build validation; current docs also state there is no native C aggregate ABI claim for wasm/native unsupported aggregate paths at `docs/audits/compiler/backend/abi-verification-v1.md:35-53`.
- Current target evidence for pointer/scalar FFI is intentionally target-specific and partial: x86/x32 have pointer/native-libc smokes, x64 has `ptr` parameter plus `c_int`/`c_uint` smokes, and function pointer spellings remain rejected until wrapper ABI verification at `compiler/compiler_evidence_gates.go:69-85`, `compiler/compiler_evidence_gates.go:211-227`, `compiler/compiler_evidence_gates.go:270-286`, and `docs/spec/runtime/runtime_abi.md:32-47`.
