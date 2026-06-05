# Stable Generic Collections v1 Design

Goal slice: P19.1 Stable Generic Collections.

## Intent

Promote the P19.0 collection evidence into a narrow stable Tetra-source
collection surface without claiming a full allocator-backed production
`Vec<T>` or `HashMap<K,V>` runtime.

## Shape

- `lib.core.collections.Vec<T>` is a caller-owned slice view:
  `items: []T`, `logical_len: Int`.
- `lib.core.collections.HashMap<K,V>` is a caller-owned key/value slice view:
  `keys: []K`, `values: []V`, `logical_len: Int`.
- Generic operations are limited to shape-preserving operations that do not
  require generic equality or hashing:
  `vec_from_slice`, `vec_len`, `vec_first_or`, `vec_get_or`,
  `hash_map_from_slices`, `hash_map_len`, and `hash_map_first_value_or`.
- Common key/value specializations provide equality-based lookup only where the
  current language can prove the concrete operations:
  `hash_map_get_i32_i32_or` and `hash_map_get_u8_i32_or`.

## Compiler Contract

The value representation is the existing static monomorphized generic
representation:

- type arguments are substituted into `[]T`, `[]K`, and `[]V` fields;
- generated names use the existing deterministic mangle policy;
- instantiated generic structs/functions are concrete before lowering;
- nested generic struct fields and function-typed generic struct arguments
  remain outside this v1 surface.

## Allocation Evidence

The API never allocates collection storage internally. Callers allocate slices
with the existing `core.make_*` or `core.island_make_*` constructors, so
allocation evidence remains in the existing allocation-plan reports. P19.0
runtime helpers (`StringBuilder`, `VecBytes`, `HashMapBytes`, `ByteBuffer`,
`RingBuffer`) remain runtime evidence helpers, not the public generic API.

## Benchmark Boundary

The P19.1 benchmark gate is a truth-bench-harness subset, not the full P20
matrix:

- manifest/report scope: `p19.1_generic_collections`;
- category: `hash table`;
- languages: Tetra, C++, and Rust;
- required equivalence metadata: same `algorithm_id` and
  `input_description` across all rows;
- Tetra row artifacts: proof, allocation, bounds, and performance report paths.

The checked P19.1 artifact is a dry-run shape/evidence artifact. It may close
the P19.1 benchmark-equivalent gate, but it still must not claim C++/Rust
parity, measured speed, or an official benchmark result.
