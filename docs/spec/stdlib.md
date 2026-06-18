# Standard Library Spec Notes

This page anchors stdlib-specific spec policies.

- Naming and versioning policy (normative for release gating):
  [stdlib_naming_versioning.md](./stdlib_naming_versioning.md)

## Stable Core Surface (`v0.4.0` Current Profile)

The current `v0.4.0` stable stdlib modules under `lib/core` are:

- `lib.core.async`
- `lib.core.capability`
- `lib.core.collections`
- `lib.core.crypto`
- `lib.core.filesystem`
- `lib.core.http`
- `lib.core.io`
- `lib.core.json`
- `lib.core.math`
- `lib.core.memory`
- `lib.core.net`
- `lib.core.networking`
- `lib.core.postgres`
- `lib.core.serialization`
- `lib.core.slices`
- `lib.core.strings`
- `lib.core.sync`
- `lib.core.testing`
- `lib.core.time`

## Stable Smoke Evidence and Profile Semantics

Every stable `lib.core.*` module has a checked stable-core smoke example under
`examples/core_*_smoke.tetra`. In this spec, "checked" means the example exists,
imports the stable `lib.core.*` module directly, is covered by the stdlib
documentation/completeness workflow, and is accepted by `tools/cmd/verify-docs`.
It does not mean that every stable-core smoke is an active case in the default
`./tetra smoke --list --format=json` linux-x64 profile.

The default linux-x64 smoke profile is a narrower runtime smoke list. It
currently includes `examples/core_math_smoke.tetra` and
`examples/core_memory_smoke.tetra` as active stdlib smoke cases. Other
stable-core smoke examples may appear in the smoke-list report's
`excluded_examples` array with the reason `not part of linux-x64 smoke profile`.
That exclusion is profile scope, not a withdrawal of stable stdlib evidence.

Stable stdlib release evidence therefore comes from the separate stdlib
completeness workflow documented in
`docs/user/standard_library_guide.md#verification`: generated API docs for
`lib/core`, `lib/experimental`, and all `examples/core_*_smoke.tetra` files,
followed by `go run ./tools/cmd/verify-docs --manifest
docs/generated/manifest.json`. The smoke-list contract is validated separately
with `validate-smoke-list`, which requires examples to be either active smoke
cases or explicit `excluded_examples`.

## Builtin Slice Constructor Contract

The builtin slice constructors `core.make_u8`, `core.make_u16`,
`core.make_i32`, `core.make_bool`, and the matching `core.island_make_*`
constructors accept a logical element count. A zero count returns a valid empty
slice, a negative count traps or rejects before allocation, and byte-size
overflow traps or rejects before allocation. The returned `len` is the logical
element count. `[]bool` uses the current i32-width representation, so bool
constructor overflow is checked with element size 4.

This contract is part of safe language semantics. Explanation flags and
allocation/bounds reports may expose the planner decision and guard status, but
they do not enable or disable the checks.

## Stable Generic Collection Views

`lib.core.collections` now exposes narrow source-level generic collection views:
`Vec<T>` wraps a caller-owned `[]T`, and `HashMap<K,V>` wraps caller-owned
parallel key/value slices. These helpers do not allocate storage internally.
Callers allocate slices through the existing `core.make_*` or
`core.island_make_*` constructors, so allocation evidence remains in the
allocation-plan reports for those constructors. The v1 generic collection
surface does not claim a production allocator-backed vector/map runtime,
generic hashing/equality protocols, resizing, collision handling, C++/Rust
performance parity, and makes no official benchmark result claim. The P19.1 benchmark gate
is covered by a checked dry-run `truth-bench-harness` artifact for scope
`p19.1_generic_collections`, with hash-table Tetra/C++/Rust rows and Tetra
proof/allocation/bounds/performance report paths; it records source-equivalent
shape only, not measured speed.

P19.2 foundation evidence adds a source-first HTTP/JSON gate for
`lib.core.http` and `lib.core.json`. The checked
`p19.2_http_json_source_first` dry-run artifact records Tetra-only
`HTTP plaintext` and `HTTP JSON` rows, with proof/allocation/bounds and
`tetra.stdlib.http_json.production_stack.v1` coverage artifacts. This covers
request-head framing, pipelined local buffers, response byte-buffer helpers,
message-object writers, and internal borrowed HTTP/JSON request-region evidence.
It also records internal per-server UTC-second Date cache evidence and Linux
`netrt.Writev`/`netrt.Sendfile` helper evidence through the runtime coverage
report. It does not promote a production HTTP server, source-level cached-date
API, cross-worker Date cache, `webrt.flush` scatter/gather integration, HTTP
static-file sendfile path, zero-copy production file-serving, C++/Rust parity,
or P20 performance matrix. It makes no official TechEmpower result claim.

P19.3 closure evidence adds a source-first PostgreSQL gate for
`lib.core.postgres` and the internal runtime driver/pool layer. The checked
`p19.3_postgres_source_first` dry-run artifact records Tetra-only
`DB single query`, `DB multiple queries`, `DB updates`, and `DB fortunes` rows,
with proof/allocation/bounds and
`tetra.stdlib.postgresql.production_driver.v1` coverage artifacts. This covers
startup/SCRAM, prepared statements, binary int4 helpers, pooling/backpressure,
borrowed DataRow decode, and local `/db`, `/queries`, `/updates`, and
`/fortunes` correctness evidence. The closure also links checked local
SCRAM/PostgreSQL reports for all six endpoints, the `/db` matrix, and the
`/queries`/`/updates`/`/fortunes` matrix through
`tools/cmd/validate-techempower-report`. It does not promote a full
source-level PostgreSQL driver API or external production database deployment.
It makes no official TechEmpower result claim, no production database benchmark
claim, no C++/Rust parity claim, no P20 performance matrix claim, no measured
speed comparison claim, and no runtime behavior change claim.

## Stable Core Function Matrix

This matrix is the function-level `lib.core.*` surface for the current
`v0.4.0` profile. `Effects` records the function's declared `uses` clause;
`none` means the function has no per-function `uses` clause. `Stability` means
stable within the current release profile and should not be read as a broader
runtime, host, or security guarantee.

Function entries:

- Module: `lib.core.async`
  - Signature: `async func ready(value: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Async helper surface; no scheduler/runtime progress guarantee is implied.

- Module: `lib.core.async`
  - Signature: `async func pair_sum(lhs: Int, rhs: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Async helper surface; awaits local helper calls only.

- Module: `lib.core.async`
  - Signature: `func select_or(value: Int, fallback: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Pure fallback helper.

- Module: `lib.core.capability`
  - Signature: `func mem() -> cap.mem`
  - Effects: `capability, mem`
  - Stability: stable core
  - Contract: Unsafe capability wrapper; callers still need matching effects.

- Module: `lib.core.capability`
  - Signature: `func io() -> cap.io`
  - Effects: `capability, io`
  - Stability: stable core
  - Contract: Unsafe capability wrapper; callers still need matching effects.

- Module: `lib.core.collections`
  - Signature: `func vec_from_slice<T>(items: []T) -> Vec<T>`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Source-level generic view over caller-owned slice storage; no internal allocation.

- Module: `lib.core.collections`
  - Signature: `func vec_len<T>(vec: Vec<T>) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Returns the logical length captured by `vec_from_slice`; not a runtime
              allocator-backed vector.

- Module: `lib.core.collections`
  - Signature: `func vec_first_or<T>(vec: Vec<T>, fallback: T) -> T`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Generic first-value helper over caller-owned slice storage; returns fallback for an
              empty view.

- Module: `lib.core.collections`
  - Signature: `func vec_get_or<T>(vec: Vec<T>, index: Int, fallback: T) -> T`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Generic indexed scan helper; returns fallback for negative or missing indexes.

- Module: `lib.core.collections`
  - Signature: `func hash_map_from_slices<K, V>(keys: []K, values: []V) -> HashMap<K,V>`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Source-level generic key/value slice view; caller owns storage and matching key/value
              lengths.

- Module: `lib.core.collections`
  - Signature: `func hash_map_len<K, V>(map: HashMap<K,V>) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Returns the key-count logical length captured by `hash_map_from_slices`.

- Module: `lib.core.collections`
  - Signature: `func hash_map_first_value_or<K, V>(map: HashMap<K,V>, fallback: V) -> V`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Generic first-value helper; no key lookup, hashing, resizing, or collision handling is
              implied.

- Module: `lib.core.collections`
  - Signature: `func hash_map_get_i32_i32_or(map: HashMap<Int,Int>, key: Int, fallback: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Specialized equality lookup for common `Int` keys and `Int` values over caller-owned
              slices.

- Module: `lib.core.collections`
  - Signature:
    `func hash_map_get_u8_i32_or(map: HashMap<UInt8,Int>, key: UInt8, fallback: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Specialized equality lookup for common `UInt8` keys and `Int` values over caller-owned
              slices.

- Module: `lib.core.collections`
  - Signature: `func len_i32(values: []i32) -> Int`
  - Effects: `mem`
  - Stability: stable core
  - Contract: Legacy `[]i32` scan helper retained alongside generic collection views.

- Module: `lib.core.collections`
  - Signature: `func contains_i32(values: []i32, needle: Int) -> Bool`
  - Effects: `mem`
  - Stability: stable core
  - Contract: Legacy `[]i32` scan helper retained alongside generic collection views.

- Module: `lib.core.collections`
  - Signature: `func count_i32(values: []i32, needle: Int) -> Int`
  - Effects: `mem`
  - Stability: stable core
  - Contract: Legacy `[]i32` scan helper retained alongside generic collection views.

- Module: `lib.core.collections`
  - Signature: `func first_or_i32(values: []i32, fallback: Int) -> Int`
  - Effects: `mem`
  - Stability: stable core
  - Contract: Legacy `[]i32` scan helper; returns fallback for an empty slice.

- Module: `lib.core.crypto`
  - Signature: `func interface_strength() -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Stable interface marker for examples and API docs.

- Module: `lib.core.crypto`
  - Signature: `func mix_seed(seed: Int, value: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Deterministic non-negative mixer for reproducible interface tests, saturating the i32
              minimum normalization case; no encryption or authentication claim.

- Module: `lib.core.crypto`
  - Signature: `func checksum_u8(values: []u8) -> Int`
  - Effects: `mem`
  - Stability: stable core
  - Contract: Deterministic byte checksum for examples and API-shape tests.

- Module: `lib.core.crypto`
  - Signature: `func constant_time_eq_u8(lhs: []u8, rhs: []u8) -> Bool`
  - Effects: `mem`
  - Stability: stable core
  - Contract: Equality helper for byte slices; scans equal-length inputs without early value
              mismatch exit.

- Module: `lib.core.filesystem`
  - Signature: `func exists(path: String, io_cap: cap.io) -> Bool`
  - Effects: `io`
  - Stability: stable `v0.4.0` linux-x64 slice; `fs_exists` linux-x86/linux-x32 smokes plus
               scheduler composition
  - Contract: Host-backed existence check through `__tetra_fs_exists`; requires an explicit `cap.io`
              token and returns false for missing, embedded-NUL, invalid, too-long, or unsupported
              paths. On linux-x86 and linux-x32 this covers pure filesystem existence programs and
              single-spawn self-host scheduler composition; broader filesystem/syscall parity
              remains unpromoted.

- Module: `lib.core.filesystem`
  - Signature: `func has_leading_slash(path: String) -> Bool`
  - Effects: none
  - Stability: stable core
  - Contract: Pure string-path utility; no host access.

- Module: `lib.core.filesystem`
  - Signature: `func ends_with_slash(path: String) -> Bool`
  - Effects: none
  - Stability: stable core
  - Contract: Pure string-path utility; no host access.

- Module: `lib.core.filesystem`
  - Signature: `func is_root(path: String) -> Bool`
  - Effects: none
  - Stability: stable core
  - Contract: Pure string-path utility; treats `/` as root.

- Module: `lib.core.filesystem`
  - Signature: `func slash_count(path: String) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Pure string-path utility; counts slash bytes.

- Module: `lib.core.filesystem`
  - Signature: `func directory_depth(path: String) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Pure string-path utility; counts non-empty path segments.

- Module: `lib.core.io`
  - Signature: `func capability_io() -> cap.io`
  - Effects: `capability, io`
  - Stability: stable core
  - Contract: Unsafe capability wrapper; no host permission is granted by import alone.

- Module: `lib.core.io`
  - Signature: `func mmio_read_i32(addr: ptr, io_cap: cap.io) -> Int`
  - Effects: `io, mmio`
  - Stability: stable core
  - Contract: Unsafe MMIO wrapper over caller-selected address and token.

- Module: `lib.core.io`
  - Signature: `func mmio_write_i32(addr: ptr, value: Int, io_cap: cap.io) -> Int`
  - Effects: `io, mmio`
  - Stability: stable core
  - Contract: Unsafe MMIO wrapper over caller-selected address and token.

- Module: `lib.core.net`
  - Signature: `func socket_tcp4(io_cap: cap.io) -> Int`
  - Effects: `io`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Opens a real IPv4 TCP stream socket through the linux-x64 runtime and returns the fd
              or a negative errno-style syscall result.

- Module: `lib.core.net`
  - Signature: `func bind_tcp4_loopback(fd: Int, port: Int, io_cap: cap.io) -> Int`
  - Effects: `io`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Binds a caller-owned fd to `127.0.0.1:port`; pass `0` to let the kernel choose an
              ephemeral port. Returns `-1` before the syscall for ports outside `0..65535`.

- Module: `lib.core.net`
  - Signature: `func connect_tcp4_loopback(fd: Int, port: Int, io_cap: cap.io) -> Int`
  - Effects: `io`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Connects a caller-owned fd to `127.0.0.1:port` through Linux `connect` and returns the
              syscall status. Returns `-1` before the syscall for ports outside `0..65535`.

- Module: `lib.core.net`
  - Signature: `func listen(fd: Int, backlog: Int, io_cap: cap.io) -> Int`
  - Effects: `io`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Calls Linux `listen` on a bound TCP fd and returns the syscall status.

- Module: `lib.core.net`
  - Signature: `func accept4(fd: Int, flags: Int, io_cap: cap.io) -> Int`
  - Effects: `io`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Calls Linux `accept4` with caller-provided flags and returns the accepted fd or
              negative syscall result.

- Module: `lib.core.net`
  - Signature: `func accept(fd: Int, io_cap: cap.io) -> Int`
  - Effects: `io`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Convenience wrapper for `accept4(fd, 0, io_cap)`.

- Module: `lib.core.net`
  - Signature: `func accept_nonblocking(fd: Int, io_cap: cap.io) -> Int`
  - Effects: `io`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Convenience wrapper for `accept4(fd, SOCK_NONBLOCK | SOCK_CLOEXEC, io_cap)`.

- Module: `lib.core.net`
  - Signature: `func read(fd: Int, dst: inout []u8, start: Int, count: Int, io_cap: cap.io) -> Int`
  - Effects: `io, mem`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Reads up to `count` bytes into `dst[start...]`, clamped to the slice length, and
              returns the syscall result.

- Module: `lib.core.net`
  - Signature: `func recv(fd: Int, dst: inout []u8, start: Int, count: Int, io_cap: cap.io) -> Int`
  - Effects: `io, mem`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Receives up to `count` bytes into `dst[start...]` via Linux `recvfrom` with flags `0`,
              clamped to the slice length, and returns the syscall result.

- Module: `lib.core.net`
  - Signature: `func write(fd: Int, src: []u8, start: Int, count: Int, io_cap: cap.io) -> Int`
  - Effects: `io, mem`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Writes up to `count` bytes from `src[start...]`, clamped to the slice length, and
              returns the syscall result.

- Module: `lib.core.net`
  - Signature: `func send(fd: Int, src: []u8, start: Int, count: Int, io_cap: cap.io) -> Int`
  - Effects: `io, mem`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Sends up to `count` bytes from `src[start...]` via Linux `sendto` with flags `0`,
              clamped to the slice length, and returns the syscall result.

- Module: `lib.core.net`
  - Signature: `func epoll_create(io_cap: cap.io) -> Int`
  - Effects: `io`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Creates a Linux epoll fd with `epoll_create1(0)` and returns the fd or negative
              syscall result.

- Module: `lib.core.net`
  - Signature: `func epoll_ctl_add_read(epfd: Int, fd: Int, io_cap: cap.io) -> Int`
  - Effects: `io`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Registers `fd` for `EPOLLIN` readiness with event data set to the fd.

- Module: `lib.core.net`
  - Signature: `func epoll_ctl_add_read_write(epfd: Int, fd: Int, io_cap: cap.io) -> Int`
  - Effects: `io`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Registers `fd` for `EPOLLIN | EPOLLOUT` readiness with event data set to the fd.

- Module: `lib.core.net`
  - Signature: `func epoll_ctl_mod_read(epfd: Int, fd: Int, io_cap: cap.io) -> Int`
  - Effects: `io`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Modifies an existing epoll registration to `EPOLLIN` readiness.

- Module: `lib.core.net`
  - Signature: `func epoll_ctl_mod_read_write(epfd: Int, fd: Int, io_cap: cap.io) -> Int`
  - Effects: `io`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Modifies an existing epoll registration to `EPOLLIN | EPOLLOUT` readiness.

- Module: `lib.core.net`
  - Signature: `func epoll_ctl_delete(epfd: Int, fd: Int, io_cap: cap.io) -> Int`
  - Effects: `io`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Removes `fd` from an epoll instance.

- Module: `lib.core.net`
  - Signature: `func epoll_wait_one(epfd: Int, timeout_ms: Int, io_cap: cap.io) -> Int`
  - Effects: `io`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Waits for one epoll event and returns the ready fd, `0` on timeout, or a negative
              syscall result.

- Module: `lib.core.net`
  - Signature:
    `func epoll_wait_one_into(epfd: Int, event: inout []i32, timeout_ms: Int, io_cap: cap.io) ->`
    `Int`
  - Effects: `io, mem`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Waits for one epoll event, writes the ready fd to `event[0]` and Linux event flags to
              `event[1]`, and returns `1`, `0`, or a negative syscall result.

- Module: `lib.core.net`
  - Signature: `func sock_nonblock() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Linux `SOCK_NONBLOCK` flag value for `accept4`.

- Module: `lib.core.net`
  - Signature: `func sock_cloexec() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Linux `SOCK_CLOEXEC` flag value for `accept4`.

- Module: `lib.core.net`
  - Signature: `func epoll_event_in() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Linux `EPOLLIN` flag value.

- Module: `lib.core.net`
  - Signature: `func epoll_event_out() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Linux `EPOLLOUT` flag value.

- Module: `lib.core.net`
  - Signature: `func epoll_event_err() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Linux `EPOLLERR` flag value.

- Module: `lib.core.net`
  - Signature: `func epoll_event_hup() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Linux `EPOLLHUP` flag value.

- Module: `lib.core.net`
  - Signature: `func epoll_event_fd(event: []i32) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Reads the fd slot written by `epoll_wait_one_into`, or returns `-1` for an empty event
              buffer.

- Module: `lib.core.net`
  - Signature: `func epoll_event_flags(event: []i32) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Reads the flags slot written by `epoll_wait_one_into`, or returns `-1` when the flags
              slot is missing.

- Module: `lib.core.net`
  - Signature: `func epoll_event_readable(flags: Int) -> Bool`
  - Effects: none
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Tests whether a Linux epoll flags word contains `EPOLLIN`.

- Module: `lib.core.net`
  - Signature: `func epoll_event_writable(flags: Int) -> Bool`
  - Effects: none
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Tests whether a Linux epoll flags word contains `EPOLLOUT`.

- Module: `lib.core.net`
  - Signature: `func epoll_event_has_error(flags: Int) -> Bool`
  - Effects: none
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Tests whether a Linux epoll flags word contains `EPOLLERR`.

- Module: `lib.core.net`
  - Signature: `func epoll_event_hung_up(flags: Int) -> Bool`
  - Effects: none
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Tests whether a Linux epoll flags word contains `EPOLLHUP`.

- Module: `lib.core.net`
  - Signature: `func set_nonblocking(fd: Int, io_cap: cap.io) -> Int`
  - Effects: `io`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Sets `O_NONBLOCK` on a caller-owned fd through the linux-x64 runtime and returns the
              syscall status.

- Module: `lib.core.net`
  - Signature: `func set_reuseport(fd: Int, io_cap: cap.io) -> Int`
  - Effects: `io`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Enables `SO_REUSEPORT` with Linux `setsockopt` and returns the syscall status.

- Module: `lib.core.net`
  - Signature: `func set_tcp_nodelay(fd: Int, io_cap: cap.io) -> Int`
  - Effects: `io`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Enables `TCP_NODELAY` with Linux `setsockopt` and returns the syscall status.

- Module: `lib.core.net`
  - Signature: `func close(fd: Int, io_cap: cap.io) -> Int`
  - Effects: `io`
  - Stability: stable `v0.4.0` linux-x64 slice
  - Contract: Closes a caller-owned fd through the linux-x64 runtime and returns the syscall status.

- Module: `lib.core.http`
  - Signature: `func route_bad_request() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Route sentinel for malformed request lines.

- Module: `lib.core.http`
  - Signature: `func route_not_found() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Route sentinel for syntactically valid but unknown paths.

- Module: `lib.core.http`
  - Signature: `func route_plaintext() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Route ID for `/plaintext`.

- Module: `lib.core.http`
  - Signature: `func route_json() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Route ID for `/json`.

- Module: `lib.core.http`
  - Signature: `func route_db() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Route ID for `/db`.

- Module: `lib.core.http`
  - Signature: `func route_queries() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Route ID for `/queries`, including query-string requests such as `/queries?queries=7`.

- Module: `lib.core.http`
  - Signature: `func route_updates() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Route ID for `/updates`, including query-string requests.

- Module: `lib.core.http`
  - Signature: `func route_fortunes() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Route ID for `/fortunes`.

- Module: `lib.core.http`
  - Signature: `func route_tech_empower(request: String) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Parses a GET request line enough to classify TechEmpower benchmark paths; the HTTP/1.1
              request line must end with CRLF and the slash-prefixed target must use visible ASCII
              bytes before its separating space. This is not a full HTTP message parser.

- Module: `lib.core.http`
  - Signature: `func route_tech_empower_bytes(request: []u8, request_len: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Parses a GET request line directly from a caller-owned byte buffer after `net.read`,
              using only `request[0:request_len]`; returns `route_bad_request()` when the requested
              byte range is missing or malformed, including LF-only, bare-CR, or non-visible
              request-target bytes.

- Module: `lib.core.http`
  - Signature:
    `func route_tech_empower_bytes_at(request: []u8, start: Int, request_len: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Offset variant for classifying one request inside a caller-owned byte buffer,
              including pipelined request buffers; returns `route_bad_request()` for negative,
              wrapped, physically missing, non-CRLF, or non-visible-target request-line windows.

- Module: `lib.core.http`
  - Signature: `func request_keep_alive(request: String) -> Bool`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: HTTP/1.1 keep-alive policy helper with `Connection: close` detection for smoke-covered
              request text whose request line has a visible-ASCII slash-prefixed target and ends
              with CRLF.

- Module: `lib.core.http`
  - Signature: `func request_keep_alive_bytes(request: []u8, request_len: Int) -> Bool`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Byte-buffer variant of the HTTP/1.1 keep-alive policy helper for data read from
              sockets; returns `false` for missing physical request bytes or malformed request
              targets.

- Module: `lib.core.http`
  - Signature:
    `func request_keep_alive_bytes_at(request: []u8, start: Int, request_len: Int) -> Bool`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Offset keep-alive helper for one request inside a larger byte buffer; returns `false`
              for negative, wrapped, physically missing, or malformed request-line windows.

- Module: `lib.core.http`
  - Signature: `func request_head_len_bytes(request: []u8, request_len: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Returns the first complete HTTP request-head length through `\r\n\r\n`, or `0` when
              the buffer does not contain a complete head or the requested byte range is missing.

- Module: `lib.core.http`
  - Signature: `func request_head_len_bytes_at(request: []u8, start: Int, request_len: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Offset variant for locating the next request-head boundary in a pipelined byte buffer;
              returns `0` for negative, wrapped, or physically missing request windows.

- Module: `lib.core.http`
  - Signature: `func plaintext_body_len() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Exact byte count for the TechEmpower plaintext body.

- Module: `lib.core.http`
  - Signature:
    `func response_head_len(status: Int, reason: String, server: String, date: String,`
    `content_type: String, content_len: Int, keep_alive: Bool) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Exact byte count for a compact HTTP/1.1 response head with Server, Date, Content-Type,
              Content-Length, and Connection headers, or `-1` for non-three-digit status codes,
              negative Content-Length values, or malformed reason/header values containing CR/LF,
              non-HTAB controls, or DEL.

- Module: `lib.core.http`
  - Signature: `func plaintext_response_len(server: String, date: String, keep_alive: Bool) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Exact byte count for a complete `/plaintext` response, or `-1` for malformed response
              header values.

- Module: `lib.core.http`
  - Signature:
    `func json_message_response_len(server: String, date: String, message: String, keep_alive:`
    `Bool) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Exact byte count for a complete `/json` message response, or `-1` for malformed
              response header values.

- Module: `lib.core.http`
  - Signature:
    `func write_plaintext_response(dst: inout []u8, server: String, date: String, keep_alive: Bool)`
    `-> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes a complete HTTP/1.1 plaintext response into a caller-owned byte buffer,
              returning `-1` without mutation when the complete response window is missing or
              response header values are malformed.

- Module: `lib.core.http`
  - Signature:
    `func write_json_message_response(dst: inout []u8, server: String, date: String, message:`
    `String, keep_alive: Bool) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes a complete HTTP/1.1 JSON message response into a caller-owned byte buffer,
              returning `-1` without mutation when the complete response window is missing or
              response header values are malformed.

- Module: `lib.core.http`
  - Signature:
    `func write_response_head(dst: inout []u8, status: Int, reason: String, server: String, date:`
    `String, content_type: String, content_len: Int, keep_alive: Bool) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes an HTTP/1.1 response head and returns the next byte index, or `-1` without
              mutation when the complete head window is missing, status code is not three digits,
              Content-Length is negative, or reason/header values contain CR/LF, non-HTAB controls,
              or DEL.

- Module: `lib.core.http`
  - Signature: `func header_line_len(name: String, value: String) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Exact byte count for a single `Name: Value\\r\\n` header line, or `-1` for invalid
              field names or malformed header values containing CR/LF, non-HTAB controls, or DEL.

- Module: `lib.core.http`
  - Signature:
    `func write_header_at(dst: inout []u8, start: Int, name: String, value: String) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes one HTTP header line at `start`, or `-1` for negative starts, invalid field
              names, malformed header values containing CR/LF, non-HTAB controls, or DEL, wrapped
              offsets, or missing destination bytes.

- Module: `lib.core.http`
  - Signature: `func write_crlf_at(dst: inout []u8, start: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes CRLF at `start`, or `-1` for negative starts, wrapped offsets, or missing
              destination bytes.

- Module: `lib.core.http`
  - Signature: `func write_ascii_at(dst: inout []u8, start: Int, text: String) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes ASCII text into a caller-owned byte buffer, or `-1` for negative starts,
              wrapped offsets, or missing destination bytes.

- Module: `lib.core.http`
  - Signature: `func write_decimal_i32_at(dst: inout []u8, start: Int, value: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes decimal integer text, including `-2147483648`, into a caller-owned byte buffer,
              or `-1` for negative starts, wrapped offsets, or missing destination bytes.

- Module: `lib.core.http`
  - Signature: `func digits_i32(value: Int) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Decimal digit count helper for HTTP status and length fields, including the i32
              minimum value.

- Module: `lib.core.json`
  - Signature: `func encoded_string_len(text: String) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: JSON string serializer sizing helper, including quotes and escape expansion.

- Module: `lib.core.json`
  - Signature: `func message_object_len(message: String) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Exact byte count for `{"message":...}` response bodies.

- Module: `lib.core.json`
  - Signature: `func digits_i32(value: Int) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Decimal digit count helper for JSON integer field sizing, including the i32 minimum
              value.

- Module: `lib.core.json`
  - Signature: `func world_object_len(id: Int, random_number: Int) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Exact byte count for TechEmpower-style `World` JSON objects, including i32 minimum
              field values.

- Module: `lib.core.json`
  - Signature: `func write_message_object(dst: inout []u8, message: String) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes compact `{"message":...}` JSON into a caller-owned byte buffer and returns
              bytes written, or `-1` when the destination is too short.

- Module: `lib.core.json`
  - Signature: `func write_message_object_at(dst: inout []u8, start: Int, message: String) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes compact `{"message":...}` JSON at `start` and returns the next byte index, or
              `-1` for negative starts, wrapped offsets, or missing destination bytes.

- Module: `lib.core.json`
  - Signature: `func write_json_string(dst: inout []u8, text: String) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes a compact escaped JSON string at the start of a caller-owned byte buffer, or
              returns `-1` when the destination is too short.

- Module: `lib.core.json`
  - Signature: `func write_json_string_at(dst: inout []u8, start: Int, text: String) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes a compact escaped JSON string at `start` and returns the next byte index, or
              `-1` for negative starts, wrapped offsets, or missing destination bytes.

- Module: `lib.core.postgres`
  - Signature: `func protocol_version_3() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL protocol version 3.0 integer for startup messages.

- Module: `lib.core.postgres`
  - Signature: `func int4_oid() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL `int4` type OID used by TechEmpower prepared-statement paths.

- Module: `lib.core.postgres`
  - Signature: `func text_oid() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL `text` type OID used by Fortunes result metadata.

- Module: `lib.core.postgres`
  - Signature: `func frame_parse() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL frontend message tag for Parse (`P`).

- Module: `lib.core.postgres`
  - Signature: `func frame_bind() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL frontend message tag for Bind (`B`).

- Module: `lib.core.postgres`
  - Signature: `func frame_describe() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL frontend message tag for Describe (`D`).

- Module: `lib.core.postgres`
  - Signature: `func frame_execute() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL frontend message tag for Execute (`E`).

- Module: `lib.core.postgres`
  - Signature: `func frame_simple_query() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL frontend message tag for Simple Query (`Q`).

- Module: `lib.core.postgres`
  - Signature: `func frame_sync() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL frontend message tag for Sync (`S`).

- Module: `lib.core.postgres`
  - Signature: `func frame_terminate() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL frontend message tag for Terminate (`X`).

- Module: `lib.core.postgres`
  - Signature: `func frame_authentication() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL backend message tag for Authentication (`R`).

- Module: `lib.core.postgres`
  - Signature: `func frame_bind_complete() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL backend message tag for BindComplete (`2`).

- Module: `lib.core.postgres`
  - Signature: `func frame_command_complete() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL backend message tag for CommandComplete (`C`).

- Module: `lib.core.postgres`
  - Signature: `func frame_data_row() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL backend message tag for DataRow (`D`).

- Module: `lib.core.postgres`
  - Signature: `func frame_error_response() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL backend message tag for ErrorResponse (`E`).

- Module: `lib.core.postgres`
  - Signature: `func frame_no_data() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL backend message tag for NoData (`n`).

- Module: `lib.core.postgres`
  - Signature: `func frame_parse_complete() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL backend message tag for ParseComplete (`1`).

- Module: `lib.core.postgres`
  - Signature: `func frame_parameter_status() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL backend message tag for ParameterStatus (`S`).

- Module: `lib.core.postgres`
  - Signature: `func frame_ready_for_query() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL backend message tag for ReadyForQuery (`Z`).

- Module: `lib.core.postgres`
  - Signature: `func frame_row_description() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL backend message tag for RowDescription (`T`).

- Module: `lib.core.postgres`
  - Signature: `func ready_for_query_idle_status() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: ReadyForQuery idle transaction status byte (`I`).

- Module: `lib.core.postgres`
  - Signature: `func ready_for_query_in_transaction_status() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: ReadyForQuery in-transaction status byte (`T`).

- Module: `lib.core.postgres`
  - Signature: `func ready_for_query_failed_transaction_status() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: ReadyForQuery failed-transaction status byte (`E`).

- Module: `lib.core.postgres`
  - Signature: `func describe_kind_portal() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL Describe payload kind for portals (`P`).

- Module: `lib.core.postgres`
  - Signature: `func describe_kind_statement() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL Describe payload kind for prepared statements (`S`).

- Module: `lib.core.postgres`
  - Signature:
    `func startup_message_len(user: String, database: String, application_name: String) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Exact byte count for a protocol 3.0 startup message with user, database, and
              application_name parameters.

- Module: `lib.core.postgres`
  - Signature:
    `func write_startup_message(dst: inout []u8, user: String, database: String, application_name:`
    `String) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes a PostgreSQL startup message into a caller-owned byte buffer, or returns `-1`
              when the destination is too short or a startup C-string field contains an embedded
              NUL.

- Module: `lib.core.postgres`
  - Signature: `func simple_query_payload_len(query: String) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL Simple Query length field value for `query`, or `-1` when `query` contains
              an embedded NUL.

- Module: `lib.core.postgres`
  - Signature: `func simple_query_frame_len(query: String) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Exact byte count for a typed Simple Query frontend frame, or `-1` when `query`
              contains an embedded NUL.

- Module: `lib.core.postgres`
  - Signature: `func write_simple_query(dst: inout []u8, query: String) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes a typed PostgreSQL Simple Query frame into a caller-owned byte buffer, or
              returns `-1` when the destination is too short or `query` contains an embedded NUL.

- Module: `lib.core.postgres`
  - Signature:
    `func parse_payload_len(statement: String, query: String, param_type_oids: []i32) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL Parse length field value for a prepared statement and parameter type OIDs,
              or `-1` when the OID count exceeds the signed i16 protocol range or a statement/query
              C-string contains an embedded NUL.

- Module: `lib.core.postgres`
  - Signature:
    `func parse_frame_len(statement: String, query: String, param_type_oids: []i32) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Exact byte count for a typed Parse frontend frame, or `-1` when the OID count exceeds
              the signed i16 protocol range or a statement/query C-string contains an embedded NUL.

- Module: `lib.core.postgres`
  - Signature:
    `func write_parse(dst: inout []u8, statement: String, query: String, param_type_oids: []i32) ->`
    `Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes a PostgreSQL Parse frame for extended-query prepared statements, or returns
              `-1` when the destination is too short, the OID count exceeds the signed i16 protocol
              range, or a statement/query C-string contains an embedded NUL.

- Module: `lib.core.postgres`
  - Signature: `func bind_text_0_payload_len(portal: String, statement: String) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL Bind length field value for no text parameters, or `-1` when
              portal/statement C-string fields contain an embedded NUL.

- Module: `lib.core.postgres`
  - Signature: `func bind_text_0_frame_len(portal: String, statement: String) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Exact byte count for a Bind frame with no text parameters, or `-1` when
              portal/statement C-string fields contain an embedded NUL.

- Module: `lib.core.postgres`
  - Signature:
    `func bind_text_1_payload_len(portal: String, statement: String, value0: String) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL Bind length field value for one text parameter, or `-1` when
              portal/statement C-string fields contain an embedded NUL.

- Module: `lib.core.postgres`
  - Signature:
    `func bind_text_1_frame_len(portal: String, statement: String, value0: String) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Exact byte count for a Bind frame with one text parameter, or `-1` when
              portal/statement C-string fields contain an embedded NUL.

- Module: `lib.core.postgres`
  - Signature:
    `func bind_text_2_payload_len(portal: String, statement: String, value0: String, value1:`
    `String) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL Bind length field value for two text parameters, or `-1` when
              portal/statement C-string fields contain an embedded NUL.

- Module: `lib.core.postgres`
  - Signature:
    `func bind_text_2_frame_len(portal: String, statement: String, value0: String, value1: String)`
    `-> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Exact byte count for a Bind frame with two text parameters, or `-1` when
              portal/statement C-string fields contain an embedded NUL.

- Module: `lib.core.postgres`
  - Signature: `func write_bind_text_0(dst: inout []u8, portal: String, statement: String) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes a PostgreSQL Bind frame with no text parameters, or returns `-1` when the
              destination is too short or portal/statement C-string fields contain an embedded NUL.

- Module: `lib.core.postgres`
  - Signature:
    `func write_bind_text_1(dst: inout []u8, portal: String, statement: String, value0: String) ->`
    `Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes a PostgreSQL Bind frame with one text parameter, or returns `-1` when the
              destination is too short or portal/statement C-string fields contain an embedded NUL.

- Module: `lib.core.postgres`
  - Signature:
    `func write_bind_text_2(dst: inout []u8, portal: String, statement: String, value0: String,`
    `value1: String) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes a PostgreSQL Bind frame with two text parameters for update-style paths, or
              returns `-1` when the destination is too short or portal/statement C-string fields
              contain an embedded NUL.

- Module: `lib.core.postgres`
  - Signature: `func describe_portal_payload_len(portal: String) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL Describe Portal length field value, or `-1` when `portal` contains an
              embedded NUL.

- Module: `lib.core.postgres`
  - Signature: `func describe_portal_frame_len(portal: String) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Exact byte count for a Describe Portal frontend frame, or `-1` when `portal` contains
              an embedded NUL.

- Module: `lib.core.postgres`
  - Signature: `func write_describe_portal(dst: inout []u8, portal: String) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes a PostgreSQL Describe Portal frame, or returns `-1` when the destination is too
              short or `portal` contains an embedded NUL.

- Module: `lib.core.postgres`
  - Signature: `func describe_statement_payload_len(statement: String) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL Describe Statement length field value, or `-1` when `statement` contains an
              embedded NUL.

- Module: `lib.core.postgres`
  - Signature: `func describe_statement_frame_len(statement: String) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Exact byte count for a Describe Statement frontend frame, or `-1` when `statement`
              contains an embedded NUL.

- Module: `lib.core.postgres`
  - Signature: `func write_describe_statement(dst: inout []u8, statement: String) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes a PostgreSQL Describe Statement frame, or returns `-1` when the destination is
              too short or `statement` contains an embedded NUL.

- Module: `lib.core.postgres`
  - Signature: `func execute_payload_len(portal: String, max_rows: Int) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: PostgreSQL Execute length field value, or `-1` when `portal` contains an embedded NUL.

- Module: `lib.core.postgres`
  - Signature: `func execute_frame_len(portal: String, max_rows: Int) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Exact byte count for an Execute frontend frame, or `-1` when `portal` contains an
              embedded NUL.

- Module: `lib.core.postgres`
  - Signature: `func write_execute(dst: inout []u8, portal: String, max_rows: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes a PostgreSQL Execute frame, or returns `-1` when the destination is too short
              or `portal` contains an embedded NUL.

- Module: `lib.core.postgres`
  - Signature: `func sync_frame_len() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Exact byte count for a PostgreSQL Sync frontend frame.

- Module: `lib.core.postgres`
  - Signature: `func write_sync(dst: inout []u8) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes a PostgreSQL Sync frame, or returns `-1` when the destination is too short.

- Module: `lib.core.postgres`
  - Signature: `func terminate_frame_len() -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Exact byte count for a PostgreSQL Terminate frontend frame.

- Module: `lib.core.postgres`
  - Signature: `func write_terminate(dst: inout []u8) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes a PostgreSQL Terminate frontend frame into a caller-owned byte buffer, or
              returns `-1` when the destination is too short.

- Module: `lib.core.postgres`
  - Signature: `func frame_type_at(frame: []u8, start: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Reads a typed PostgreSQL frame tag at `start`, or `-1` for negative or missing starts.

- Module: `lib.core.postgres`
  - Signature: `func frame_length_at(frame: []u8, start: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Reads a typed PostgreSQL frame length field, or `-1` for negative starts, wrapped
              offsets, missing length bytes, or malformed negative signed lengths.

- Module: `lib.core.postgres`
  - Signature: `func frame_payload_len_at(frame: []u8, start: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Returns payload byte count for a typed PostgreSQL frame, or `-1` for invalid, short,
              or negative-start frame lengths.

- Module: `lib.core.postgres`
  - Signature: `func frame_total_len_at(frame: []u8, start: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Returns total byte count for a typed PostgreSQL frame, or `-1` for invalid, short,
              negative-start, or total-overflow frame lengths.

- Module: `lib.core.postgres`
  - Signature: `func frame_payload_start(start: Int) -> Int`
  - Effects: none
  - Stability: stable `v0.4.0` core
  - Contract: Returns the payload offset for a typed PostgreSQL frame.

- Module: `lib.core.postgres`
  - Signature: `func row_description_column_count(payload: []u8, start: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Reads RowDescription column count from a backend payload, or `-1` for negative starts,
              missing bytes, or high-bit signed count fields.

- Module: `lib.core.postgres`
  - Signature:
    `func row_description_type_oid_at(payload: []u8, start: Int, payload_len: Int, column_index:`
    `Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Scans RowDescription metadata and returns one column type OID, or `-1` on missing,
              malformed, negative-start, or wrapped input.

- Module: `lib.core.postgres`
  - Signature: `func data_row_column_count(payload: []u8, start: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Reads DataRow column count from a backend payload, or `-1` for negative starts,
              missing bytes, or high-bit signed count fields.

- Module: `lib.core.postgres`
  - Signature: `func data_row_value_len_at(payload: []u8, start: Int, column_index: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Reads one DataRow value length, returning `-1` for NULL, malformed negative lengths,
              negative starts, missing indexes, or physically missing positive value bytes.

- Module: `lib.core.postgres`
  - Signature: `func data_row_value_start_at(payload: []u8, start: Int, column_index: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Returns one DataRow value start offset, or `-1` for NULL, negative starts, missing
              indexes, or physically missing positive value bytes.

- Module: `lib.core.postgres`
  - Signature: `func data_row_i32_at(payload: []u8, start: Int, column_index: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Parses an ASCII integer DataRow value for TechEmpower `World` rows.

- Module: `lib.core.postgres`
  - Signature:
    `func command_complete_affected_rows(payload: []u8, start: Int, payload_len: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Parses the trailing affected-row count from CommandComplete text, returning `0` for
              empty, negative-start, wrapped, physically missing, out-of-range i32, or non-trailing
              digit ranges.

- Module: `lib.core.postgres`
  - Signature: `func ready_for_query_status(payload: []u8, start: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Reads ReadyForQuery transaction status byte, or `-1` for negative starts or missing
              payload bytes.

- Module: `lib.core.postgres`
  - Signature: `func cstring_end_at(src: []u8, start: Int, limit: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Finds a NUL terminator inside a bounded byte range, or `-1` for missing,
              negative-start, reversed, or physically missing ranges.

- Module: `lib.core.postgres`
  - Signature: `func parse_ascii_i32_at(src: []u8, start: Int, count: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Parses a bounded ASCII integer from a byte buffer, returning `0` for empty,
              negative-start, wrapped, physically missing, or out-of-range i32 ranges.

- Module: `lib.core.postgres`
  - Signature: `func write_ascii_at(dst: inout []u8, start: Int, text: String) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes ASCII text into a caller-owned byte buffer, or `-1` for negative starts,
              wrapped offsets, or missing destination bytes.

- Module: `lib.core.postgres`
  - Signature: `func write_cstring_at(dst: inout []u8, start: Int, value: String) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes a NUL-terminated C string into a caller-owned byte buffer, or `-1` for embedded
              NUL bytes, negative starts, wrapped offsets, or missing destination bytes.

- Module: `lib.core.postgres`
  - Signature:
    `func write_cstring_pair_at(dst: inout []u8, start: Int, name: String, value: String) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes two NUL-terminated C strings into a caller-owned byte buffer, or `-1` for
              embedded NUL bytes, negative starts, wrapped offsets, or missing destination bytes.

- Module: `lib.core.postgres`
  - Signature: `func write_i32_be_at(dst: inout []u8, start: Int, value: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes a signed big-endian i32 field using two's-complement bytes and returns the next
              byte index, or `-1` for negative starts, wrapped offsets, or missing destination
              bytes.

- Module: `lib.core.postgres`
  - Signature: `func write_i16_be_at(dst: inout []u8, start: Int, value: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Writes a signed big-endian i16 field using the low two's-complement bytes and returns
              the next byte index, or `-1` for negative starts, wrapped offsets, or missing
              destination bytes.

- Module: `lib.core.postgres`
  - Signature: `func read_i32_be(src: []u8, start: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Reads a non-negative big-endian i32 field from a caller-owned byte buffer, or `-1` for
              negative starts, wrapped offsets, missing bytes, or high-bit values that cannot be
              represented as a non-negative `Int`.

- Module: `lib.core.postgres`
  - Signature: `func read_i32_be_signed(src: []u8, start: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Reads a PostgreSQL i32 length field, normalizing any negative signed value, negative
              start, wrapped offset, or missing byte to `-1`.

- Module: `lib.core.postgres`
  - Signature: `func read_i16_be(src: []u8, start: Int) -> Int`
  - Effects: `mem`
  - Stability: stable `v0.4.0` core
  - Contract: Reads a big-endian i16 field from a caller-owned byte buffer, or `-1` for negative
              starts, wrapped offsets, or missing bytes.

- Module: `lib.core.math`
  - Signature: `func add_i32(a: Int, b: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Pure integer addition helper.

- Module: `lib.core.math`
  - Signature: `func min_i32(a: Int, b: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Pure integer minimum helper.

- Module: `lib.core.math`
  - Signature: `func max_i32(a: Int, b: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Pure integer maximum helper.

- Module: `lib.core.math`
  - Signature: `func clamp_i32(value: Int, lo: Int, hi: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Pure integer helper; assumes caller chooses sensible bounds.

- Module: `lib.core.memory`
  - Signature: `func memset_u8(dst: ptr, v: UInt8, n: Int, mem: cap.mem) -> Int`
  - Effects: `mem`
  - Stability: stable core
  - Contract: Unsafe byte helper; no allocation, bounds validation, or permission grant.

- Module: `lib.core.memory`
  - Signature: `func memcpy_u8(dst: ptr, src: ptr, n: Int, mem: cap.mem) -> Int`
  - Effects: `mem`
  - Stability: stable core
  - Contract: Unsafe byte helper; caller owns pointer validity and overlap assumptions.

- Module: `lib.core.networking`
  - Signature: `func default_port_http() -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Standard HTTP port constant.

- Module: `lib.core.networking`
  - Signature: `func default_port_https() -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Standard HTTPS port constant.

- Module: `lib.core.networking`
  - Signature: `func clamp_port(port: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Deterministic port-range policy helper.

- Module: `lib.core.networking`
  - Signature: `func is_valid_port(port: Int) -> Bool`
  - Effects: none
  - Stability: stable core
  - Contract: Port-range validation helper for endpoint configuration.

- Module: `lib.core.networking`
  - Signature: `func choose_port(preferred: Int, fallback: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Endpoint configuration helper; `0` preferred port falls back.

- Module: `lib.core.networking`
  - Signature: `func retry_backoff_ms(attempt: Int, base_ms: Int, max_ms: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Deterministic retry backoff policy helper; negative bases clamp to `0`, non-negative
              caps are honored before overflow, and uncapped overflow saturates to `2147483647`.

- Module: `lib.core.serialization`
  - Signature: `func clamp_u8(value: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Pure packing helper; not a general serializer.

- Module: `lib.core.serialization`
  - Signature: `func pack_u8_pair(high: Int, low: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Pure two-byte packing helper; not a wire-format guarantee.

- Module: `lib.core.serialization`
  - Signature: `func unpack_u8_high(packed: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Pure unpack helper; negative packed input returns `0`.

- Module: `lib.core.serialization`
  - Signature: `func unpack_u8_low(packed: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Pure unpack helper; negative packed input returns `0`.

- Module: `lib.core.serialization`
  - Signature: `func checksum_u8(values: []u8) -> Int`
  - Effects: `mem`
  - Stability: stable core
  - Contract: Simple checksum helper; not authentication or encryption.

- Module: `lib.core.slices`
  - Signature: `func sum_i32(values: []i32) -> Int`
  - Effects: `mem`
  - Stability: stable core
  - Contract: Slice scan helper for supported element type only.

- Module: `lib.core.slices`
  - Signature: `func weighted_sum_i32(values: []i32) -> Int`
  - Effects: `mem`
  - Stability: stable core
  - Contract: Slice scan helper for supported element type only.

- Module: `lib.core.slices`
  - Signature: `func sum_u8(values: []u8) -> Int`
  - Effects: `mem`
  - Stability: stable core
  - Contract: Slice scan helper for supported element type only.

- Module: `lib.core.strings`
  - Signature: `func ascii_len(text: String) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: ASCII/byte-oriented helper; no Unicode normalization guarantee.

- Module: `lib.core.strings`
  - Signature: `func ascii_sum(text: String) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: ASCII/byte-oriented helper; no Unicode normalization guarantee.

- Module: `lib.core.strings`
  - Signature: `func is_empty(text: String) -> Bool`
  - Effects: none
  - Stability: stable core
  - Contract: ASCII/byte-oriented helper built from `ascii_len`.

- Module: `lib.core.sync`
  - Signature: `func merge_status(lhs: Int, rhs: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Pure status helper; not a runtime synchronization primitive.

- Module: `lib.core.sync`
  - Signature: `func all_ready(lhs: Bool, rhs: Bool) -> Bool`
  - Effects: none
  - Stability: stable core
  - Contract: Pure boolean helper; not a runtime synchronization primitive.

- Module: `lib.core.sync`
  - Signature: `func spin_countdown(start: Int, ticks: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Pure countdown helper; no sleeping or scheduling behavior.

- Module: `lib.core.sync`
  - Signature: `func barrier_target(workers: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Pure clamp helper; not a runtime barrier.

- Module: `lib.core.testing`
  - Signature: `func assert_true(value: Bool) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Status-code helper; `0` means pass and `1` means fail.

- Module: `lib.core.testing`
  - Signature: `func assert_false(value: Bool) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Status-code helper; `0` means pass and `1` means fail.

- Module: `lib.core.testing`
  - Signature: `func assert_eq_i32(actual: Int, expected: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Status-code helper; `0` means pass and `1` means fail.

- Module: `lib.core.testing`
  - Signature: `func combine(lhs: Int, rhs: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Status-code helper; returns first non-zero status.

- Module: `lib.core.time`
  - Signature: `func millis_from_seconds(seconds: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Pure duration arithmetic; negative input clamps to `0` and positive overflow saturates
              to `Int` max.

- Module: `lib.core.time`
  - Signature: `func seconds_from_millis(milliseconds: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Pure duration arithmetic; negative input clamps to `0`.

- Module: `lib.core.time`
  - Signature: `func clamp_timeout_ms(value: Int, lo: Int, hi: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Pure duration arithmetic; assumes caller chooses sensible bounds.

- Module: `lib.core.time`
  - Signature: `func add_duration_ms(base: Int, delta: Int) -> Int`
  - Effects: none
  - Stability: stable core
  - Contract: Pure duration arithmetic; adds `delta` to `base`, clamps a negative summed result to
              `0`, and saturates positive overflow to `Int` max.


## `lib.core.math`

`lib.core.math` is a stable, side-effect-free integer helper module covered by
`examples/core_math_smoke.tetra`. The smoke imports `lib.core.math` directly and
exercises `add_i32`, `min_i32`, `max_i32`, and `clamp_i32` while preserving the
stable exit value `42`.

Function behavior:

- `add_i32(a, b)` returns `a + b`.
- `min_i32(a, b)` returns the smaller argument, returning `b` when `a` is not
  less than `b`.
- `max_i32(a, b)` returns the larger argument, returning `b` when `a` is not
  greater than `b`.
- `clamp_i32(value, lo, hi)` returns `lo` when `value < lo`, `hi` when
  `value > hi`, and otherwise `value`.

## `lib.core.memory`

`lib.core.memory` is a stable capability-bound byte helper module covered by
`examples/core_memory_smoke.tetra`. The smoke imports `lib.core.memory` directly
and exercises both `memset_u8` and `memcpy_u8` through an explicit `cap.mem`
token while preserving the stable exit value `42`.

Calls require a `uses mem` effect and a caller-provided `cap.mem` token. These
helpers operate on caller-selected memory regions; they do not allocate memory,
validate pointers, add bounds checks, or grant host permissions.

### `lib.core.memory` Production Boundary

The Memory Production Core treats this module as a thin safe-wrapper surface
around reviewed unsafe byte loops. `memcpy_u8` and `memset_u8` do not allocate,
do not perform bounds checks, do not track ownership, and do not prove that two
regions are non-overlapping. A production memory report must therefore pair
uses of this module with separate allocator, ownership, and runtime bounds
diagnostics evidence.

For verifier stability: this module does not allocate and does not perform bounds checks.

Function behavior:

- `memset_u8(dst, v, n, mem)` writes the `UInt8` value `v` to `n` bytes starting
  at `dst` and returns `0`. Passing `0` is the documented clear/fill pattern for
  a valid writable region.
- `memcpy_u8(dst, src, n, mem)` copies `n` bytes from `src` to `dst` in
  increasing byte order and returns `0`. The caller remains responsible for
  choosing valid readable and writable regions and for any overlap assumptions.

## `lib.core.net`

`lib.core.net` is a stable capability-bound Linux TCP socket client/server I/O slice
covered by `examples/core_net_smoke.tetra`. The smoke imports `lib.core.net`
directly, opens a real IPv4 TCP stream socket through the linux-x64 runtime,
sets `O_NONBLOCK`, enables `SO_REUSEPORT` and `TCP_NODELAY`, binds the socket to
loopback with an ephemeral port, listens, registers the listener for `EPOLLIN`,
modifies read/write readiness interest, deletes the epoll registration,
performs a zero-timeout wait, exercises fd/flag extraction helpers, closes the
fds, and preserves the stable exit value `42`. Compiler integration coverage
also runs real local TCP client/server exchanges through
`connect`/`accept4`/`read`/`recv`/`write`/`send`, an epoll readiness path,
nonblocking `accept4` convenience, and a socket-option smoke.
Loopback bind/connect helpers reject ports outside `0..65535` before serializing
the TCP port field.

Calls require a caller-provided `cap.io` token plus `uses io`; buffer-writing
helpers such as `epoll_wait_one_into` also require `uses mem`. The current
surface is intentionally small: it proves Tetra source can run real Linux TCP
socket client/server syscalls and one-event epoll readiness with add/mod/delete
interest control and readiness flag capture, but it is not yet a
production HTTP server framework. Full event-loop abstractions, io_uring,
per-core workers, broader socket-option coverage, TLS, DNS, and
PostgreSQL/database APIs remain future runtime/library work.

For verifier stability: `lib.core.net` currently provides executable
linux-x64 TCP socket open/bind/connect/listen/accept/read/recv/write/send/nonblocking/close
helpers, `SO_REUSEPORT` and `TCP_NODELAY` helpers, plus epoll
create/add-read/add-read-write/mod-read/mod-read-write/delete/wait-one
and wait-one-into readiness flag helpers, `SOCK_NONBLOCK`/`SOCK_CLOEXEC`
helpers, and `EPOLLIN`/`EPOLLOUT`/`EPOLLERR`/`EPOLLHUP` predicates for local
client/server slices; full TechEmpower event-loop and PostgreSQL paths are not
implied by this slice.

## `lib.core.networking`

`lib.core.networking` is a stable endpoint policy-helper module covered by
`examples/core_networking_smoke.tetra`. The smoke imports
`lib.core.networking` directly and exercises deterministic port and retry
helpers while preserving the stable exit value `42`.

### `lib.core.networking` Runtime Boundary

`lib.core.networking` remains endpoint policy only. It is not an alias for
sockets, does not open sockets, does not perform name resolution, and does not
send or receive bytes. The TechEmpower-compatible transport/database surface is
separate production runtime/library work: the current `lib.core.net` slice owns
real linux-x64 socket
open/bind/connect/listen/accept/read/recv/write/send/nonblocking/close helpers,
`SO_REUSEPORT`/`TCP_NODELAY`, and epoll add/mod/delete plus wait-one readiness,
while broader event-loop abstractions, socket-option APIs, and full database
APIs are expected behind future `lib.core.net` expansion and higher-level
`lib.core.postgres` driver layers instead of extending this policy-helper
module. HTTP/1.1 String and byte-buffer request-line routing, byte-buffer
request-head framing, and response byte-buffer serialization helpers live in
`lib.core.http`;
JSON byte-buffer serialization helpers live in `lib.core.json`.

For verifier stability: `lib.core.networking` remains endpoint policy only and
is not an alias for sockets.

## `lib.core.json`

`lib.core.json` is a stable executable JSON helper module covered by
`examples/core_json_smoke.tetra`. It provides compact byte-buffer writers and
exact length helpers for the JSON response shapes needed by the local
TechEmpower-compatible stack. It does not open sockets, parse HTTP, allocate
buffers for callers, or talk to PostgreSQL.

The compiler/runtime also has an internal `compiler/internal/jsonrt`
`ParseValueView` path for P7 web-stack evidence: unescaped JSON strings borrow
from the caller-owned input bytes, escaped strings copy into a provided request
region, and malformed input produces no unsafe facts. That internal API is not
part of the stable `lib.core.json` Tetra-source surface.

For verifier stability: callers own destination buffer capacity. Writer helpers
return the next byte index after the encoded JSON payload.

## `lib.core.http`

`lib.core.http` is a stable executable HTTP/1.1 helper module covered by
`examples/core_http_smoke.tetra`. It classifies TechEmpower GET request-line
paths from either `String` request text or caller-owned `[]u8` buffers read from
sockets, locates `\r\n\r\n` request-head boundaries for pipelined byte buffers,
applies small HTTP/1.1 keep-alive policy helpers, and writes compact response
heads plus full TechEmpower-style `/plaintext` and `/json` responses into
caller-owned byte buffers, including Server, Date, Content-Type, Content-Length,
and Connection headers. Response-head helpers reject non-three-digit status
codes, negative Content-Length values, CR/LF header injection, and non-HTAB
control-byte header values instead of serializing malformed headers. It does
not open sockets, accept clients, parse full
HTTP header maps or request bodies, schedule event loops, or talk to PostgreSQL.

The compiler/runtime also has an internal `compiler/internal/httprt`
`ParseRequestView` path for P7 web-stack evidence. It parses a complete
request head from a caller-owned byte buffer with caller-provided header
scratch storage, returns borrowed header views, and reports request storage
without allocating on the hot path. That internal API is not a new stable
`lib.core.http` function.

For verifier stability: callers own destination buffer capacity. Writer helpers
return the next byte index after the encoded HTTP payload.

## `lib.core.postgres`

`lib.core.postgres` is a stable executable PostgreSQL wire-frame helper module
covered by `examples/core_postgres_smoke.tetra`,
`examples/core_postgres_prepared_smoke.tetra`, and
`examples/core_postgres_result_smoke.tetra`. It provides caller-owned
byte-buffer sizing, writers, and readers for protocol 3.0 startup messages,
Simple Query frontend frames, extended-query Parse/Bind/Describe/Execute/Sync
frontend frames, RowDescription/DataRow/CommandComplete/ReadyForQuery backend
payloads, Terminate frontend frames, and big-endian i16/i32 fields. These
helpers are intended as the first Tetra-source layer for the TechEmpower
PostgreSQL path. Parse-frame helpers return sentinels instead of wrapping
parameter type counts that exceed the signed i16 protocol range; backend
RowDescription/DataRow count readers also reject high-bit signed i16 fields.
PostgreSQL C-string length and writer helpers reject embedded NUL bytes before
writing startup, query, statement, or portal fields.

For verifier stability: this module does not open sockets, authenticate, own
connection state, manage prepared statement state, or pool connections. The
prepared smoke uses a named portal for Describe/Execute because the current
compiler has a separate relocation edge case for modules that only pass empty
string literals; Bind still covers the unnamed-portal byte shape used by the
TechEmpower fast path. The result smoke covers bounded server-frame inspection
only; the full TechEmpower PostgreSQL driver and connection-pool surface
remains broader runtime/library work on top of `lib.core.net` transport and
these wire helpers.

The compiler/runtime PostgreSQL client also has P7 internal helpers for binary
`int4` Bind payloads, borrowed DataRow cell decoding, and text/binary `int4`
row reads used by the local TechEmpower-compatible runtime path. These helpers
preserve the stable `lib.core.postgres` boundary: Tetra-source callers still
own byte-buffer construction, while the higher-level driver/pool remains an
internal runtime layer.

P19.3 records that higher-level runtime layer through
`tetra.stdlib.postgresql.production_driver.v1` and a checked
`p19.3_postgres_source_first` dry-run gate plus checked local SCRAM reports:
`docs/benchmarks/techempower_scram_single_query_local_report.json`,
`docs/benchmarks/techempower_scram_single_query_matrix_local_report.json`, and
`docs/benchmarks/techempower_scram_endpoint_matrix_local_report.json`. The gate
proves source-first DB endpoint rows and honest local runtime/PostgreSQL
measurement evidence only; it records no official TechEmpower result, no
external production database claim, no production database benchmark claim, no
measured speed comparison claim, and no source-level full driver API promotion.

## Generated Docs Rendering Policy

Generated docs render a source unit by declared module path when the file has a
`module ...` declaration. For example, `lib/core/math.tetra` renders as
`lib.core.math`, and `examples/core_math_smoke.tetra` renders as
`examples.core_math_smoke` because that smoke declares that module name.

Generated docs render a portable repository file path when a source file has no
module declaration. Example-only programs such as `examples/flow_hello.tetra`
therefore keep the `examples/...` spelling in generated docs. This mixed
rendering is intentional: dotted names are module identities, while slash paths
are file identities.

## Promotion Notes

For the `v0.4.0` release line, each current `lib/experimental/*.tetra` file is
a production compatibility mirror that imports the matching `lib.core.*` module
and forwards to it. Stable code should import `lib.core.*` directly; the
`lib.experimental.*` namespace is retained only for legacy source
compatibility.

Entries:

- Experimental mirror: `lib.experimental.async`
  - Stable target: `lib.core.async`
  - `v0.4.0` status: Compatibility mirror; source note directs stable callers to `lib.core.async`.

- Experimental mirror: `lib.experimental.collections`
  - Stable target: `lib.core.collections`
  - `v0.4.0` status: Compatibility mirror; source note directs stable callers to
                     `lib.core.collections`.

- Experimental mirror: `lib.experimental.crypto`
  - Stable target: `lib.core.crypto`
  - `v0.4.0` status: Compatibility mirror for stable crypto interface helpers.

- Experimental mirror: `lib.experimental.filesystem`
  - Stable target: `lib.core.filesystem`
  - `v0.4.0` status: Compatibility mirror for filesystem helpers, including the linux-x64
                     host-backed `exists` slice and the fs-only linux-x86/linux-x32 smokes.

- Experimental mirror: `lib.experimental.io`
  - Stable target: `lib.core.io`
  - `v0.4.0` status: Compatibility mirror for capability/MMIO wrappers; callers still need matching
                     effects.

- Experimental mirror: `lib.experimental.math`
  - Stable target: `lib.core.math`
  - `v0.4.0` status: Compatibility mirror; explicitly present in `lib/experimental/math.tetra`.

- Experimental mirror: `lib.experimental.memory`
  - Stable target: `lib.core.memory`
  - `v0.4.0` status: Compatibility mirror; explicitly present in `lib/experimental/memory.tetra` and
                     still requires `cap.mem`.

- Experimental mirror: `lib.experimental.networking`
  - Stable target: `lib.core.networking`
  - `v0.4.0` status: Compatibility mirror for stable endpoint policy helpers.

- Experimental mirror: `lib.experimental.serialization`
  - Stable target: `lib.core.serialization`
  - `v0.4.0` status: Compatibility mirror for packing/checksum helpers; not a general serializer
                     guarantee.

- Experimental mirror: `lib.experimental.slices`
  - Stable target: `lib.core.slices`
  - `v0.4.0` status: Compatibility mirror; source note directs stable callers to `lib.core.slices`.

- Experimental mirror: `lib.experimental.strings`
  - Stable target: `lib.core.strings`
  - `v0.4.0` status: Compatibility mirror; source note directs stable callers to `lib.core.strings`.

- Experimental mirror: `lib.experimental.sync`
  - Stable target: `lib.core.sync`
  - `v0.4.0` status: Compatibility mirror for pure status/countdown helpers; not runtime
                     synchronization.

- Experimental mirror: `lib.experimental.testing`
  - Stable target: `lib.core.testing`
  - `v0.4.0` status: Compatibility mirror; source note directs stable callers to `lib.core.testing`.

- Experimental mirror: `lib.experimental.time`
  - Stable target: `lib.core.time`
  - `v0.4.0` status: Compatibility mirror for pure duration arithmetic; no clock or scheduler
                     access.


Generated API docs label `lib.experimental.*` modules as experimental so they
are not confused with the stable `lib.core.*` API namespace. The production
claim is limited to mirror forwarding compatibility, not new stable APIs under
`lib.experimental.*`.

## Stable Type Display

Public compiler output and generated API docs use canonical builtin names:

Entries:

- Source aliases: `Int`, `i32`
  - Canonical name: `i32`
  - Notes: Default integer literal type.

- Source aliases: `UInt8`, `Byte`, `u8`
  - Canonical name: `u8`
  - Notes: Slice element supported by `[]u8` and string storage.

- Source aliases: `UInt16`, `u16`
  - Canonical name: `u16`
  - Notes: Native-first slice element supported by `[]u16`.

- Source aliases: `Bool`, `bool`
  - Canonical name: `bool`
  - Notes: Boolean literal and condition type.

- Source aliases: `String`, `str`
  - Canonical name: `str`
  - Notes: Two-slot UTF-8 string/slice shape; `.len` and String view constructors use byte lengths.

- Source aliases: `ConsentToken`
  - Canonical name: `consent.token`
  - Notes: Privacy/consent capability token.

- Source aliases: `SecretInt`
  - Canonical name: `secret.i32`
  - Notes: Privacy-protected integer wrapper.


Structural types use deterministic field order and slot counts. `str`, `[]u8`,
`[]u16`, `[]i32`, and `[]bool` are two-slot values (`ptr`, `len`). Safe code
may read `String.len` as a byte length but cannot assign `String.ptr` or
`String.len`; `text.window(start, count)`, `text.prefix(count)`, and
`text.suffix(start)` return checked byte views of the same string storage.
`text.borrow()` and `xs.borrow()` return no-allocation immutable borrowed
views, `text.copy()` and `xs.copy()` return independently owned storage, and
`copy_into(dst)` copies bytes/elements into a caller-owned destination after a
checked length guard.
`T?` adds one presence tag slot to the payload slots. Opaque handles such as
`ptr`, `island`, `actor`, `cap.io`, `cap.mem`, and `task.*` are not
interchangeable even when they occupy one slot.

## Semantic Type Model Boundaries (v0.3 profile)

The current semantic checker intentionally enforces these boundaries:

- Arrays (`[N]T`) are not part of the checked type model yet.
- Slice element support is currently limited to `[]u8`, `[]u16`, `[]i32`, and `[]bool`.
- Local inference does not infer a type from bare `none`; optional payload type
  must be explicit (for example `let v: i32? = none`).
- Global type inference is limited to constant numeric/bool expressions used by
  immutable globals (`val`/`const`), and top-level `var` initializers are
  limited to explicit `i32`/`bool` constant expressions.

## Stable Module Quality Gates

Stable `lib.core.*` modules are required to include:

- top-of-file docs comments
- at least one `tetra doctest` block
- an `// Effects: ...` metadata line (`none` or a comma-separated list)
- a checked stable-core smoke example under `examples/core_*_smoke.tetra`

`tools/cmd/verify-docs` enforces these requirements. Default smoke-list
membership is tracked separately by `./tetra smoke --list --format=json`; stable
core smoke examples that are not active linux-x64 smoke cases must remain listed
as `excluded_examples` there.

Stable examples used as release evidence must import `lib.core.*` directly and
must not import `lib.experimental.*`.
