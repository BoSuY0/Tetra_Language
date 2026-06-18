# Optimizer Core Coverage v1

Status: P17.1 closed audit for the Ideal Master Plan.

## Summary

The optimizer now exposes a machine-readable core optimization coverage matrix for the P17.1
master-plan pass list. Every P17.1 row is classified with concrete evidence, making this a bounded
evidence-backed P17.1 closure rather than a broader optimizer claim.
This audit is a bounded evidence-backed P17.1 closure.

This closure report also records conservative optimizer slices and one bounded hot-loop shape
report:

- `basic-scalar` constant folding now folds safe scalar i32 constants and safe const-denominator
  `div_i32`/`mod_i32` constants, and simplifies same-local comparison algebra; denominator `0` and
  `-1` remain rejected for the div/mod constant-folding slice.
- `basic-scalar` CSE/GVN reuses repeated pure local-load and local-load/constant binary expressions,
  safe const-denominator `div_i32`/`mod_i32` expressions, unary local `neg_i32` expressions, plus
  safe known-local unary `neg_i32` value expressions, safe known-local `add_i32`/`sub_i32`/`mul_i32`
  value expressions, safe known-local `cmp_*_i32` value expressions including mirrored ordered
  comparisons, safe known-local `div_i32`/`mod_i32` value expressions, commutative add/mul/eq/ne
  operand variants, and mirrored lt/gt/le/ge ordered-comparison variants over unmodified value
  facts; denominator `0` and `-1` are rejected for the div/mod CSE slices, and unary min-int plus
  overflow-sensitive known-local arithmetic remain rejected.
- `basic-scalar` DCE removes simple dead local stores, bounded dead non-trapping
  comparison-expression stores, safe known-local unary `neg_i32` dead stores, safe known-local
  `add_i32`/`sub_i32`/`mul_i32` dead stores, and safe const-denominator `div_i32`/`mod_i32` dead
  stores plus safe known-local `div_i32`/`mod_i32` dead stores in straight-line Stack IR; unary
  `neg_i32` over `-2147483648`, arithmetic overflow, and denominator `0` and `-1` are rejected for
  the bounded DCE slices.
- `sccp-constant-branch` folds Stack IR branches whose condition is a literal constant at the branch
  site or a same-basic-block local known from a constant store, a same-basic-block local known from
  a safe unary `neg_i32` store or a safe constant binary-expression store including safe
  const-denominator `div_i32`/`mod_i32`, or a same-basic-block constant pure unary `neg_i32` or
  binary i32 expression including safe const-denominator `div_i32`/`mod_i32`; unary `neg_i32` over
  `-2147483648` and div/mod denominator `0` and `-1` are rejected for the SCCP expression slices. It
  also carries known-local facts through immediate labels, forward jumps, folded zero-branch targets
  when the target label has one incoming edge and no fallthrough predecessor, and folded
  nonzero-branch fallthrough labels when the immediate label has no explicit incoming branch/jump
  edge. It derives bounded zero-target and nonzero-fallthrough path facts from dynamic
  `load_local; jmp_if_zero` conditions and bounded zero/nonzero path facts from dynamic
  `load_local; const_i32 0; cmp_eq_i32/cmp_ne_i32; jmp_if_zero` comparisons only for later
  same-local branches on the proven path; explicit-incoming fallthrough labels, dynamic zero-target
  labels with fallthrough predecessors, and dynamic comparison-target labels with fallthrough
  predecessors are rejected, and unreachable fallthrough is pruned only up to the next label
  boundary.
- `mem2reg-single-assignment` promotes adjacent single-store/single-load Stack IR temp locals and
  stack-neutral separated temps whose producer is a single `const_i32`/`load_local` value, a bounded
  non-trapping comparison expression, a safe const unary `neg_i32` expression, a safe known-local
  unary `neg_i32` expression, a safe const `add_i32`/`sub_i32`/`mul_i32` expression, a safe
  known-local `add_i32`/`sub_i32`/`mul_i32` expression, or a safe const-denominator
  `div_i32`/`mod_i32` expression, or a safe known-local `div_i32`/`mod_i32` expression, as long as
  source locals remain unmodified; unary `neg_i32` over `-2147483648`, arithmetic overflow,
  source-local mutation, and denominator `0` and `-1` are rejected for the bounded producer-temp
  slices.
- `licm-pure-invariant` hoists only pure load-local/constant comparison, add/sub/mul arithmetic,
  known-local `add_i32`/`sub_i32`/`mul_i32` left-or-right operand expressions, known-local
  `cmp_*_i32` left-or-right operand expressions, safe const-denominator `div_i32`/`mod_i32`
  expressions, and safe known-local `div_i32`/`mod_i32` denominator expressions out of selected
  proof-tagged while-loop shapes; denominator `0` and `-1`, loop-index operands, and loop-mutated
  operands are rejected.
- `CoreHotLoopShapeEvidence()` reports SSA-verified/register-shaped scalar sum, scalar
  constant-stride sum, scalar sum-of-squares, scalar product reduction, scalar branchy max
  reduction, scalar affine sum, scalar countdown, proof-tagged slice sum, proof-tagged slice
  constant-stride sum, and call-loop rows, plus a checked slice-sum fallback row when no proof tag
  exists. The constant-stride rows are bounded to positive compile-time strides `2..127`, the affine
  row is bounded to positive compile-time scale and bias `1..127`, the product row is bounded to
  `product *= index + 1`, the max row is bounded to the exact branchy `index > max` update shape,
  and the slice stride row still requires the proof-tagged unchecked load.

## Evidence

Record: Coverage matrix includes all P17.1 planned optimization rows.
Result: pass.

Record: Core optimization rows are classified with evidence.
Scope: constant folding, copy propagation, DCE, simple inlining, loop
canonicalization, LICM narrow slice, allocation sinking narrow slice, scalar
replacement narrow slice, and BCE v1 narrow slice.
Result: pass.

Record: Constant folding boundary is narrow.
Scope: safe scalar constants, same-local comparison algebraic forms,
safe const-denominator `div_i32`/`mod_i32` constants, and neutral-element
algebraic forms only. Denominator `0` and `-1` remain rejected.
Result: pass.

Record: SCCP row is promoted only to bounded evidence.
Scope: narrow constant-condition, known-local, stored safe unary neg,
safe constant-expression facts, safe const-denominator div/mod facts,
single-predecessor label propagation, folded branch facts, dynamic load-local
zero/nonzero facts, dynamic zero-comparison facts, rejection cases, and
fallthrough-pruning evidence.
Result: pass.

Record: mem2reg row is bounded.
Scope: adjacent or stack-neutral separated Stack IR single-assignment temp
promotion, bounded comparison-expression producers, safe unary neg producers,
safe add/sub/mul producers, and safe div/mod producer temps.
Result: pass.

Record: CSE/GVN row points at exact `basic-scalar` evidence.
Scope: local-load, local-load/constant, safe div/mod, unary local neg,
safe known-local unary, arithmetic, comparison, div/mod, commutative, and
mirrored ordered-comparison reuse evidence.
Result: pass.

Record: LICM row points at `licm-pure-invariant` evidence.
Scope: proof-tagged pure comparison, add/sub/mul arithmetic, known-local
left-or-right operand hoisting, safe const-denominator div/mod hoisting, and
safe known-local div/mod denominator hoisting.
Result: pass.

Record: `basic-scalar` DCE remains bounded.
Scope: simple single-producer dead local stores, non-trapping comparisons,
safe known-local unary `neg_i32`, safe add/sub/mul stores, safe div/mod stores,
and rejection of unary min-int, arithmetic overflow, and unsafe denominators.
Result: pass.

Record: `basic-scalar` removes repeated pure local-load recomputation only when
operands and cached local remain unmodified.
Result: pass.

Record: `basic-scalar` removes repeated safe const-denominator div/mod
recomputation only when operands and cached local remain unmodified.
Result: pass.

Record: `basic-scalar` removes repeated unary local `neg_i32` recomputation
only when operand and cached locals remain unmodified.
Result: pass.

Record: `basic-scalar` canonicalization remains bounded.
Scope: commutative add/mul/eq/ne, mirrored lt/gt/le/ge expressions,
safe known-local unary neg values, safe add/sub/mul, cmp, and div/mod values,
with translation validation rejecting unsafe or non-equivalent rewrites.
Result: pass.

Record: `basic-scalar` simplifies same-local comparison algebra only over
single pure local/constant values.
Result: pass.

Record: `sccp-constant-branch` stays within bounded branch folding.
Scope: literal, known-local, unary neg, constant-expression, path-known
nonzero, and safe const-denominator div/mod branches, with explicit rejection
of unsafe unary neg, unsafe denominators, dynamic stores, multi-predecessor
labels, explicit-incoming fallthrough labels, and fallthrough predecessors.
Result: pass.

Record: `mem2reg-single-assignment` removes only bounded producer temps.
Scope: adjacent or stack-neutral temp store/load pairs from `const_i32`,
`load_local`, comparison, safe unary neg, safe add/sub/mul, and safe div/mod
producers for non-param locals with exactly one store and one load.
Result: pass.

Record: Hot-loop shape report proves named scalar and proof-slice shapes lower
through machine IR with SSA verification, no linear-scan spills, and no
stack-churn ops.
Result: pass.

Record: Checked/no-proof slice sum remains explicit stack fallback in the
hot-loop report.
Result: pass.

Record: P17.0 pass contract still guards the registered optimization passes.
Result: pass.

## Boundaries

This P17.1 closure does not claim general SSA SCCP, path-sensitive lattice propagation, arbitrary
branch-derived facts beyond the bounded dynamic load-local and zero-comparison zero/nonzero branch
slices, fact merging across multi-predecessor, explicit-incoming fallthrough, or
fallthrough-predecessor labels, arbitrary comparison reasoning, range propagation, arbitrary
expression SCCP, general DCE beyond the bounded safe known-local unary `neg_i32`, arithmetic, and
div/mod slices, arbitrary arithmetic-expression DCE, arbitrary div/mod DCE, unsafe division/modulo
DCE, general mem2reg beyond the bounded producer-temp variants including safe const unary `neg_i32`,
safe const arithmetic, safe known-local unary `neg_i32`, safe known-local arithmetic, and safe
known-local div/mod, broad SSA GVN beyond the bounded local-expression, safe known-local unary,
arithmetic, comparison, and div/mod value, and same-local comparison algebra variants including
unary local `neg_i32` CSE, unsafe unary neg promotion, unsafe arithmetic promotion, unsafe
division/modulo promotion, unsafe division/modulo CSE/DCE, arbitrary div/mod CSE, unsafe
division/modulo LICM, alias-aware LICM, or general SSA LICM, general allocation sinking, broad
scalar replacement, vectorization, general constant-stride loop optimization beyond the bounded
`2..127` scalar and proof-slice rows, general affine loop optimization beyond the bounded `1..127`
scale/bias scalar affine row, general product-reduction optimization beyond the bounded scalar
product row, general min/max optimization beyond the bounded branchy scalar max row, or C/Rust
`-O1`/`-O2` performance parity. This is a no C/Rust `-O1`/`-O2` performance parity claim. It is a
truthful closure of the P17.1 row list: coverage is explicit, implemented rows are bounded, and
broader optimizer ambitions remain future work outside this closure.
