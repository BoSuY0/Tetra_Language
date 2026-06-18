# Machine IR Register Backend Readiness

Status: P3.0 readiness contract plus P3.1-P3.5 scalar/register integration.

Machine IR is the register-oriented bridge between proof-checked compiler IR and future production
register codegen. P3.0 freezes the minimum contract before any safe program starts relying on it as
the selected backend path.

## Supported Surface

Machine IR v0 supports:

- virtual registers (`VReg`) and physical registers (`PhysReg`);
- named functions and basic blocks;
- explicit successors plus branch and return opcodes;
- calls with ABI and clobber metadata;
- loads, stores, integer arithmetic, comparisons, increments, and indexed loads;
- spill/reload marker opcodes;
- liveness analysis;
- live interval construction;
- linear scan register allocation with spill decisions;
- stable text dumps through `machine.FormatProgram`.

P3.0 is readiness infrastructure only. P3.1 adds the first selected backend path for eligible scalar
integer functions, without introducing a user-visible semantic mode.

## P3.1 Scalar Integer Path

`machine.ScalarIntFunctionFromStackIR` translates simple stack IR functions into single-block
Machine IR when every operation is scalar integer work:

- integer constants and local loads/stores;
- `+`, `-`, `*`, unary negation, and integer comparisons;
- one ordinary return value.

The slice/heap/actor/exception/generic/control-flow surfaces remain outside this path. Unsupported
functions are reported as ineligible and continue on the ordinary stack backend path.

On x64 targets, eligible functions are emitted by a direct scalar register path in
`compiler/internal/backend/x64core`. It uses ABI parameter spilling and a small frame-backed scratch
area, but avoids the stack-machine eval `PushRax`/`PopRax`/`PopRcx` instruction pattern for scalar
expressions. Native runtime behavior remains the contract: the scalar path must return the same
result as the stack path, and fallback is an internal implementation strategy, not a user semantic
switch.

Explain backend reports expose this mixed implementation as
`machine-ir-scalar-for-eligible-functions; stack fallback otherwise` when a target supports the
scalar path and at least one function is eligible.

## P3.2 Scalar Loop Path

P3.2 recognizes the first loop-shaped scalar integer subset:

```tetra
func sum_n(n: Int) -> Int:
    var i = 0
    var total = 0
    while i < n:
        total = total + i
        i = i + 1
    return total
```

The stack IR recognizer requires the canonical lowered shape: zero initialize the index and
accumulator locals, compare `index < n`, branch to loop exit when the comparison is false, update
`total = total + index`, increment the index by one, and jump back to the loop label. Unsupported
loop forms continue through the stack backend path.

The x64 scalar loop emitter keeps the hot values in registers for the canonical path:

- `EDX`: loop bound `n`;
- `ECX`: loop index;
- `EAX`: accumulator and return value.

It emits a real compare/branch loop and avoids the stack-machine eval `PushRax`/`PopRax`/`PopRcx`
pattern in the loop body. The backend test suite also links and runs the same IR through the direct
loop path and an internal stack fallback switch, then compares their exit codes. That switch is a
backend test hook only; it is not a user semantic mode.

When explain reports include eligible scalar or loop Machine IR functions, the backend JSON includes
per-function Machine IR dumps, liveness, intervals, linear-scan allocation, and spill-slot counts.
P3.2 expects the canonical loop to allocate without spills on the linux-x64 caller-saved register
set.

## P3.3 Slice Sum Path

P3.3 recognizes the first memory hot path:

```tetra
func sum(xs: []i32) -> Int:
    var total = 0
    var i = 0
    while i < xs.len:
        total = total + xs[i]
        i = i + 1
    return total
```

The recognizer requires the lowered index load to be `IRIndexLoadI32Unchecked` with a `proof:while:`
proof id. That ties the fast path to the P1 range/BCE proof instead of treating unchecked memory as
a codegen preference. If the proof is missing, or the lowered IR still contains a checked
`IRIndexLoadI32`, the function stays on the stack backend and keeps the runtime bounds check. Raw or
unknown-provenance loops therefore remain checked because they never receive the required
proof-tagged load.

The x64 slice-sum path keeps the hot state in registers:

- `R9`: slice base pointer;
- `EDX`: slice length;
- `ECX`: loop index;
- `R10D`: accumulator;
- `R8D`: loaded element.

The loop body emits a direct indexed load from `[R9 + RCX*4]`, adds into `R10D`, increments `ECX`,
and returns the accumulator through `EAX`. Backend reports identify this path as
`machine-ir-slice-sum` and include the same dump, liveness, interval, allocation, and spill-slot
evidence as the scalar loop path.

## P3.4 Call And ABI Path

P3.4 extends the scalar register subset to calls whose ABI representation is already verified as
scalar:

- direct and nested calls with up to the target register-argument count;
- zero or one return slot;
- target-specific call ABI metadata in Machine IR (`sysv` for Linux/macOS x64, `win64` for Windows
  x64);
- explicit caller-saved clobber lists in the Machine IR dump;
- frame-backed scratch values for expression temporaries that live across a call;
- a canonical loop form whose body adds the result of a one-argument scalar call.

Multi-slot call returns, including slices and `String`, intentionally remain on the stack fallback
until their representation is covered by a separate verification slice. Calls with too many register
arguments also fall back to the existing stack ABI emitter.

The x64 register call emitter lowers arguments directly into ABI registers and records ordinary
`call rel32` patches. SysV calls use the aligned function frame directly; Win64 calls reserve and
release the required 32-byte shadow space around each call. The call-loop path frame-spills
`EAX`/`ECX`/`EDX` state before calling because those registers are caller-saved, then reloads the
loop state before continuing.

Backend reports identify straight-line scalar call functions as `machine-ir-call` and the canonical
loop call form as `machine-ir-call-loop`. Both include Machine IR dumps with callee, ABI, and
clobber metadata, plus liveness, intervals, allocation, and spill-slot evidence.

## P3.5 Backend Selection Contract

Register codegen is an implementation strategy, not a user semantic mode. The public build options
and CLI flags do not expose a backend selector, and debug or release diagnostics must not change
safe-program behavior.

When `--explain` is enabled, backend reports include one row per lowered function:

- `backend_path: "register"` for functions selected by a verified Machine IR subset;
- `backend_path: "stack"` for unsupported or unproven functions using the compatibility fallback.

The report is explanatory only. Ordinary builds do not emit `.backend.json`, and unsupported
constructs continue through the stack backend without requiring a user-visible fallback flag. The
internal `DisableMachinePaths` option exists only in backend tests so register paths can be
differentially compared against the stack implementation.

## Verifier Contract

`machine.VerifyFunction` rejects:

- empty function or block names;
- duplicate blocks;
- undefined or empty virtual register uses;
- empty opcodes and unknown opcodes;
- blocks without a terminator;
- unconditional branch or return terminators that are not last in the block;
- missing or unknown branch targets;
- successor rows that do not match branch instructions;
- malformed instruction arity for moves, loads/stores, arithmetic, compare, branch, call,
  spill/reload, push/pop, return, and indexed load;
- calls without callee, ABI, or clobber metadata.

`machine.VerifyAllocation` checks the register-allocation side:

- assigned virtual registers must exist in the function;
- assigned physical registers must be members of the target register set;
- a virtual register cannot be both assigned and spilled;
- spill slots must be inside the declared spill-slot bounds;
- overlapping live intervals cannot share the same physical register.

## Dump Contract

`machine.FormatProgram` is the stable text dump entry point for Machine IR readiness tests and
reports. It starts with `program machine_ir`, then emits each function using
`machine.FormatFunction` with deterministic block and instruction text. JSON can continue to use the
struct tags on `Program`, `Function`, `Block`, `Instr`, liveness, intervals, and allocation records.

Future P3 slices may add a CLI or report artifact around the same formatter, but they must not
introduce a user semantic mode. Backend selection remains an implementation strategy guarded by
translation validation and ordinary safe runtime checks.
