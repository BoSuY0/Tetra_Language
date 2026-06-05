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
performance parity, or an official benchmark result. The P19.1 benchmark gate
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
P20 performance matrix, or official TechEmpower result.

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
source-level PostgreSQL driver API, external production database deployment,
official TechEmpower result, production database benchmark, C++/Rust parity,
P20 performance matrix, measured speed comparison, or runtime behavior change.

## Stable Core Function Matrix

This matrix is the function-level `lib.core.*` surface for the current
`v0.4.0` profile. `Effects` records the function's declared `uses` clause;
`none` means the function has no per-function `uses` clause. `Stability` means
stable within the current release profile and should not be read as a broader
runtime, host, or security guarantee.

| Module | Function signature | Effects | Stability | Contract notes |
| --- | --- | --- | --- | --- |
| `lib.core.async` | `async func ready(value: Int) -> Int` | none | stable `v0.3.0` core | Async helper surface; no scheduler/runtime progress guarantee is implied. |
| `lib.core.async` | `async func pair_sum(lhs: Int, rhs: Int) -> Int` | none | stable `v0.3.0` core | Async helper surface; awaits local helper calls only. |
| `lib.core.async` | `func select_or(value: Int, fallback: Int) -> Int` | none | stable `v0.3.0` core | Pure fallback helper. |
| `lib.core.capability` | `func mem() -> cap.mem` | `capability, mem` | stable `v0.3.0` core | Unsafe capability wrapper; callers still need matching effects. |
| `lib.core.capability` | `func io() -> cap.io` | `capability, io` | stable `v0.3.0` core | Unsafe capability wrapper; callers still need matching effects. |
| `lib.core.collections` | `func vec_from_slice<T>(items: []T) -> Vec<T>` | `mem` | stable `v0.4.0` core | Source-level generic view over caller-owned slice storage; no internal allocation. |
| `lib.core.collections` | `func vec_len<T>(vec: Vec<T>) -> Int` | none | stable `v0.4.0` core | Returns the logical length captured by `vec_from_slice`; not a runtime allocator-backed vector. |
| `lib.core.collections` | `func vec_first_or<T>(vec: Vec<T>, fallback: T) -> T` | `mem` | stable `v0.4.0` core | Generic first-value helper over caller-owned slice storage; returns fallback for an empty view. |
| `lib.core.collections` | `func vec_get_or<T>(vec: Vec<T>, index: Int, fallback: T) -> T` | `mem` | stable `v0.4.0` core | Generic indexed scan helper; returns fallback for negative or missing indexes. |
| `lib.core.collections` | `func hash_map_from_slices<K, V>(keys: []K, values: []V) -> HashMap<K,V>` | `mem` | stable `v0.4.0` core | Source-level generic key/value slice view; caller owns storage and matching key/value lengths. |
| `lib.core.collections` | `func hash_map_len<K, V>(map: HashMap<K,V>) -> Int` | none | stable `v0.4.0` core | Returns the key-count logical length captured by `hash_map_from_slices`. |
| `lib.core.collections` | `func hash_map_first_value_or<K, V>(map: HashMap<K,V>, fallback: V) -> V` | `mem` | stable `v0.4.0` core | Generic first-value helper; no key lookup, hashing, resizing, or collision handling is implied. |
| `lib.core.collections` | `func hash_map_get_i32_i32_or(map: HashMap<Int,Int>, key: Int, fallback: Int) -> Int` | `mem` | stable `v0.4.0` core | Specialized equality lookup for common `Int` keys and `Int` values over caller-owned slices. |
| `lib.core.collections` | `func hash_map_get_u8_i32_or(map: HashMap<UInt8,Int>, key: UInt8, fallback: Int) -> Int` | `mem` | stable `v0.4.0` core | Specialized equality lookup for common `UInt8` keys and `Int` values over caller-owned slices. |
| `lib.core.collections` | `func len_i32(values: []i32) -> Int` | `mem` | stable `v0.3.0` core | Legacy `[]i32` scan helper retained alongside generic collection views. |
| `lib.core.collections` | `func contains_i32(values: []i32, needle: Int) -> Bool` | `mem` | stable `v0.3.0` core | Legacy `[]i32` scan helper retained alongside generic collection views. |
| `lib.core.collections` | `func count_i32(values: []i32, needle: Int) -> Int` | `mem` | stable `v0.3.0` core | Legacy `[]i32` scan helper retained alongside generic collection views. |
| `lib.core.collections` | `func first_or_i32(values: []i32, fallback: Int) -> Int` | `mem` | stable `v0.3.0` core | Legacy `[]i32` scan helper; returns fallback for an empty slice. |
| `lib.core.crypto` | `func interface_strength() -> Int` | none | stable `v0.3.0` core | Stable interface marker for examples and API docs. |
| `lib.core.crypto` | `func mix_seed(seed: Int, value: Int) -> Int` | none | stable `v0.3.0` core | Deterministic non-negative mixer for reproducible interface tests, saturating the i32 minimum normalization case; no encryption or authentication claim. |
| `lib.core.crypto` | `func checksum_u8(values: []u8) -> Int` | `mem` | stable `v0.3.0` core | Deterministic byte checksum for examples and API-shape tests. |
| `lib.core.crypto` | `func constant_time_eq_u8(lhs: []u8, rhs: []u8) -> Bool` | `mem` | stable `v0.3.0` core | Equality helper for byte slices; scans equal-length inputs without early value mismatch exit. |
| `lib.core.filesystem` | `func exists(path: String, io_cap: cap.io) -> Bool` | `io` | stable `v0.4.0` linux-x64 slice; `fs_exists` linux-x86/linux-x32 smokes plus scheduler composition | Host-backed existence check through `__tetra_fs_exists`; requires an explicit `cap.io` token and returns false for missing, embedded-NUL, invalid, too-long, or unsupported paths. On linux-x86 and linux-x32 this covers pure filesystem existence programs and single-spawn self-host scheduler composition; broader filesystem/syscall parity remains unpromoted. |
| `lib.core.filesystem` | `func has_leading_slash(path: String) -> Bool` | none | stable `v0.3.0` core | Pure string-path utility; no host access. |
| `lib.core.filesystem` | `func ends_with_slash(path: String) -> Bool` | none | stable `v0.3.0` core | Pure string-path utility; no host access. |
| `lib.core.filesystem` | `func is_root(path: String) -> Bool` | none | stable `v0.3.0` core | Pure string-path utility; treats `/` as root. |
| `lib.core.filesystem` | `func slash_count(path: String) -> Int` | none | stable `v0.3.0` core | Pure string-path utility; counts slash bytes. |
| `lib.core.filesystem` | `func directory_depth(path: String) -> Int` | none | stable `v0.3.0` core | Pure string-path utility; counts non-empty path segments. |
| `lib.core.io` | `func capability_io() -> cap.io` | `capability, io` | stable `v0.3.0` core | Unsafe capability wrapper; no host permission is granted by import alone. |
| `lib.core.io` | `func mmio_read_i32(addr: ptr, io_cap: cap.io) -> Int` | `io, mmio` | stable `v0.3.0` core | Unsafe MMIO wrapper over caller-selected address and token. |
| `lib.core.io` | `func mmio_write_i32(addr: ptr, value: Int, io_cap: cap.io) -> Int` | `io, mmio` | stable `v0.3.0` core | Unsafe MMIO wrapper over caller-selected address and token. |
| `lib.core.net` | `func socket_tcp4(io_cap: cap.io) -> Int` | `io` | stable `v0.4.0` linux-x64 slice | Opens a real IPv4 TCP stream socket through the linux-x64 runtime and returns the fd or a negative errno-style syscall result. |
| `lib.core.net` | `func bind_tcp4_loopback(fd: Int, port: Int, io_cap: cap.io) -> Int` | `io` | stable `v0.4.0` linux-x64 slice | Binds a caller-owned fd to `127.0.0.1:port`; pass `0` to let the kernel choose an ephemeral port. Returns `-1` before the syscall for ports outside `0..65535`. |
| `lib.core.net` | `func connect_tcp4_loopback(fd: Int, port: Int, io_cap: cap.io) -> Int` | `io` | stable `v0.4.0` linux-x64 slice | Connects a caller-owned fd to `127.0.0.1:port` through Linux `connect` and returns the syscall status. Returns `-1` before the syscall for ports outside `0..65535`. |
| `lib.core.net` | `func listen(fd: Int, backlog: Int, io_cap: cap.io) -> Int` | `io` | stable `v0.4.0` linux-x64 slice | Calls Linux `listen` on a bound TCP fd and returns the syscall status. |
| `lib.core.net` | `func accept4(fd: Int, flags: Int, io_cap: cap.io) -> Int` | `io` | stable `v0.4.0` linux-x64 slice | Calls Linux `accept4` with caller-provided flags and returns the accepted fd or negative syscall result. |
| `lib.core.net` | `func accept(fd: Int, io_cap: cap.io) -> Int` | `io` | stable `v0.4.0` linux-x64 slice | Convenience wrapper for `accept4(fd, 0, io_cap)`. |
| `lib.core.net` | `func accept_nonblocking(fd: Int, io_cap: cap.io) -> Int` | `io` | stable `v0.4.0` linux-x64 slice | Convenience wrapper for `accept4(fd, SOCK_NONBLOCK | SOCK_CLOEXEC, io_cap)`. |
| `lib.core.net` | `func read(fd: Int, dst: inout []u8, start: Int, count: Int, io_cap: cap.io) -> Int` | `io, mem` | stable `v0.4.0` linux-x64 slice | Reads up to `count` bytes into `dst[start...]`, clamped to the slice length, and returns the syscall result. |
| `lib.core.net` | `func recv(fd: Int, dst: inout []u8, start: Int, count: Int, io_cap: cap.io) -> Int` | `io, mem` | stable `v0.4.0` linux-x64 slice | Receives up to `count` bytes into `dst[start...]` via Linux `recvfrom` with flags `0`, clamped to the slice length, and returns the syscall result. |
| `lib.core.net` | `func write(fd: Int, src: []u8, start: Int, count: Int, io_cap: cap.io) -> Int` | `io, mem` | stable `v0.4.0` linux-x64 slice | Writes up to `count` bytes from `src[start...]`, clamped to the slice length, and returns the syscall result. |
| `lib.core.net` | `func send(fd: Int, src: []u8, start: Int, count: Int, io_cap: cap.io) -> Int` | `io, mem` | stable `v0.4.0` linux-x64 slice | Sends up to `count` bytes from `src[start...]` via Linux `sendto` with flags `0`, clamped to the slice length, and returns the syscall result. |
| `lib.core.net` | `func epoll_create(io_cap: cap.io) -> Int` | `io` | stable `v0.4.0` linux-x64 slice | Creates a Linux epoll fd with `epoll_create1(0)` and returns the fd or negative syscall result. |
| `lib.core.net` | `func epoll_ctl_add_read(epfd: Int, fd: Int, io_cap: cap.io) -> Int` | `io` | stable `v0.4.0` linux-x64 slice | Registers `fd` for `EPOLLIN` readiness with event data set to the fd. |
| `lib.core.net` | `func epoll_ctl_add_read_write(epfd: Int, fd: Int, io_cap: cap.io) -> Int` | `io` | stable `v0.4.0` linux-x64 slice | Registers `fd` for `EPOLLIN | EPOLLOUT` readiness with event data set to the fd. |
| `lib.core.net` | `func epoll_ctl_mod_read(epfd: Int, fd: Int, io_cap: cap.io) -> Int` | `io` | stable `v0.4.0` linux-x64 slice | Modifies an existing epoll registration to `EPOLLIN` readiness. |
| `lib.core.net` | `func epoll_ctl_mod_read_write(epfd: Int, fd: Int, io_cap: cap.io) -> Int` | `io` | stable `v0.4.0` linux-x64 slice | Modifies an existing epoll registration to `EPOLLIN | EPOLLOUT` readiness. |
| `lib.core.net` | `func epoll_ctl_delete(epfd: Int, fd: Int, io_cap: cap.io) -> Int` | `io` | stable `v0.4.0` linux-x64 slice | Removes `fd` from an epoll instance. |
| `lib.core.net` | `func epoll_wait_one(epfd: Int, timeout_ms: Int, io_cap: cap.io) -> Int` | `io` | stable `v0.4.0` linux-x64 slice | Waits for one epoll event and returns the ready fd, `0` on timeout, or a negative syscall result. |
| `lib.core.net` | `func epoll_wait_one_into(epfd: Int, event: inout []i32, timeout_ms: Int, io_cap: cap.io) -> Int` | `io, mem` | stable `v0.4.0` linux-x64 slice | Waits for one epoll event, writes the ready fd to `event[0]` and Linux event flags to `event[1]`, and returns `1`, `0`, or a negative syscall result. |
| `lib.core.net` | `func sock_nonblock() -> Int` | none | stable `v0.4.0` linux-x64 slice | Linux `SOCK_NONBLOCK` flag value for `accept4`. |
| `lib.core.net` | `func sock_cloexec() -> Int` | none | stable `v0.4.0` linux-x64 slice | Linux `SOCK_CLOEXEC` flag value for `accept4`. |
| `lib.core.net` | `func epoll_event_in() -> Int` | none | stable `v0.4.0` linux-x64 slice | Linux `EPOLLIN` flag value. |
| `lib.core.net` | `func epoll_event_out() -> Int` | none | stable `v0.4.0` linux-x64 slice | Linux `EPOLLOUT` flag value. |
| `lib.core.net` | `func epoll_event_err() -> Int` | none | stable `v0.4.0` linux-x64 slice | Linux `EPOLLERR` flag value. |
| `lib.core.net` | `func epoll_event_hup() -> Int` | none | stable `v0.4.0` linux-x64 slice | Linux `EPOLLHUP` flag value. |
| `lib.core.net` | `func epoll_event_fd(event: []i32) -> Int` | `mem` | stable `v0.4.0` linux-x64 slice | Reads the fd slot written by `epoll_wait_one_into`, or returns `-1` for an empty event buffer. |
| `lib.core.net` | `func epoll_event_flags(event: []i32) -> Int` | `mem` | stable `v0.4.0` linux-x64 slice | Reads the flags slot written by `epoll_wait_one_into`, or returns `-1` when the flags slot is missing. |
| `lib.core.net` | `func epoll_event_readable(flags: Int) -> Bool` | none | stable `v0.4.0` linux-x64 slice | Tests whether a Linux epoll flags word contains `EPOLLIN`. |
| `lib.core.net` | `func epoll_event_writable(flags: Int) -> Bool` | none | stable `v0.4.0` linux-x64 slice | Tests whether a Linux epoll flags word contains `EPOLLOUT`. |
| `lib.core.net` | `func epoll_event_has_error(flags: Int) -> Bool` | none | stable `v0.4.0` linux-x64 slice | Tests whether a Linux epoll flags word contains `EPOLLERR`. |
| `lib.core.net` | `func epoll_event_hung_up(flags: Int) -> Bool` | none | stable `v0.4.0` linux-x64 slice | Tests whether a Linux epoll flags word contains `EPOLLHUP`. |
| `lib.core.net` | `func set_nonblocking(fd: Int, io_cap: cap.io) -> Int` | `io` | stable `v0.4.0` linux-x64 slice | Sets `O_NONBLOCK` on a caller-owned fd through the linux-x64 runtime and returns the syscall status. |
| `lib.core.net` | `func set_reuseport(fd: Int, io_cap: cap.io) -> Int` | `io` | stable `v0.4.0` linux-x64 slice | Enables `SO_REUSEPORT` with Linux `setsockopt` and returns the syscall status. |
| `lib.core.net` | `func set_tcp_nodelay(fd: Int, io_cap: cap.io) -> Int` | `io` | stable `v0.4.0` linux-x64 slice | Enables `TCP_NODELAY` with Linux `setsockopt` and returns the syscall status. |
| `lib.core.net` | `func close(fd: Int, io_cap: cap.io) -> Int` | `io` | stable `v0.4.0` linux-x64 slice | Closes a caller-owned fd through the linux-x64 runtime and returns the syscall status. |
| `lib.core.http` | `func route_bad_request() -> Int` | none | stable `v0.4.0` core | Route sentinel for malformed request lines. |
| `lib.core.http` | `func route_not_found() -> Int` | none | stable `v0.4.0` core | Route sentinel for syntactically valid but unknown paths. |
| `lib.core.http` | `func route_plaintext() -> Int` | none | stable `v0.4.0` core | Route ID for `/plaintext`. |
| `lib.core.http` | `func route_json() -> Int` | none | stable `v0.4.0` core | Route ID for `/json`. |
| `lib.core.http` | `func route_db() -> Int` | none | stable `v0.4.0` core | Route ID for `/db`. |
| `lib.core.http` | `func route_queries() -> Int` | none | stable `v0.4.0` core | Route ID for `/queries`, including query-string requests such as `/queries?queries=7`. |
| `lib.core.http` | `func route_updates() -> Int` | none | stable `v0.4.0` core | Route ID for `/updates`, including query-string requests. |
| `lib.core.http` | `func route_fortunes() -> Int` | none | stable `v0.4.0` core | Route ID for `/fortunes`. |
| `lib.core.http` | `func route_tech_empower(request: String) -> Int` | none | stable `v0.4.0` core | Parses a GET request line enough to classify TechEmpower benchmark paths; the HTTP/1.1 request line must end with CRLF and the slash-prefixed target must use visible ASCII bytes before its separating space. This is not a full HTTP message parser. |
| `lib.core.http` | `func route_tech_empower_bytes(request: []u8, request_len: Int) -> Int` | `mem` | stable `v0.4.0` core | Parses a GET request line directly from a caller-owned byte buffer after `net.read`, using only `request[0:request_len]`; returns `route_bad_request()` when the requested byte range is missing or malformed, including LF-only, bare-CR, or non-visible request-target bytes. |
| `lib.core.http` | `func route_tech_empower_bytes_at(request: []u8, start: Int, request_len: Int) -> Int` | `mem` | stable `v0.4.0` core | Offset variant for classifying one request inside a caller-owned byte buffer, including pipelined request buffers; returns `route_bad_request()` for negative, wrapped, physically missing, non-CRLF, or non-visible-target request-line windows. |
| `lib.core.http` | `func request_keep_alive(request: String) -> Bool` | none | stable `v0.4.0` core | HTTP/1.1 keep-alive policy helper with `Connection: close` detection for smoke-covered request text whose request line has a visible-ASCII slash-prefixed target and ends with CRLF. |
| `lib.core.http` | `func request_keep_alive_bytes(request: []u8, request_len: Int) -> Bool` | `mem` | stable `v0.4.0` core | Byte-buffer variant of the HTTP/1.1 keep-alive policy helper for data read from sockets; returns `false` for missing physical request bytes or malformed request targets. |
| `lib.core.http` | `func request_keep_alive_bytes_at(request: []u8, start: Int, request_len: Int) -> Bool` | `mem` | stable `v0.4.0` core | Offset keep-alive helper for one request inside a larger byte buffer; returns `false` for negative, wrapped, physically missing, or malformed request-line windows. |
| `lib.core.http` | `func request_head_len_bytes(request: []u8, request_len: Int) -> Int` | `mem` | stable `v0.4.0` core | Returns the first complete HTTP request-head length through `\r\n\r\n`, or `0` when the buffer does not contain a complete head or the requested byte range is missing. |
| `lib.core.http` | `func request_head_len_bytes_at(request: []u8, start: Int, request_len: Int) -> Int` | `mem` | stable `v0.4.0` core | Offset variant for locating the next request-head boundary in a pipelined byte buffer; returns `0` for negative, wrapped, or physically missing request windows. |
| `lib.core.http` | `func plaintext_body_len() -> Int` | none | stable `v0.4.0` core | Exact byte count for the TechEmpower plaintext body. |
| `lib.core.http` | `func response_head_len(status: Int, reason: String, server: String, date: String, content_type: String, content_len: Int, keep_alive: Bool) -> Int` | none | stable `v0.4.0` core | Exact byte count for a compact HTTP/1.1 response head with Server, Date, Content-Type, Content-Length, and Connection headers, or `-1` for non-three-digit status codes, negative Content-Length values, or malformed reason/header values containing CR/LF, non-HTAB controls, or DEL. |
| `lib.core.http` | `func plaintext_response_len(server: String, date: String, keep_alive: Bool) -> Int` | none | stable `v0.4.0` core | Exact byte count for a complete `/plaintext` response, or `-1` for malformed response header values. |
| `lib.core.http` | `func json_message_response_len(server: String, date: String, message: String, keep_alive: Bool) -> Int` | none | stable `v0.4.0` core | Exact byte count for a complete `/json` message response, or `-1` for malformed response header values. |
| `lib.core.http` | `func write_plaintext_response(dst: inout []u8, server: String, date: String, keep_alive: Bool) -> Int` | `mem` | stable `v0.4.0` core | Writes a complete HTTP/1.1 plaintext response into a caller-owned byte buffer, returning `-1` without mutation when the complete response window is missing or response header values are malformed. |
| `lib.core.http` | `func write_json_message_response(dst: inout []u8, server: String, date: String, message: String, keep_alive: Bool) -> Int` | `mem` | stable `v0.4.0` core | Writes a complete HTTP/1.1 JSON message response into a caller-owned byte buffer, returning `-1` without mutation when the complete response window is missing or response header values are malformed. |
| `lib.core.http` | `func write_response_head(dst: inout []u8, status: Int, reason: String, server: String, date: String, content_type: String, content_len: Int, keep_alive: Bool) -> Int` | `mem` | stable `v0.4.0` core | Writes an HTTP/1.1 response head and returns the next byte index, or `-1` without mutation when the complete head window is missing, status code is not three digits, Content-Length is negative, or reason/header values contain CR/LF, non-HTAB controls, or DEL. |
| `lib.core.http` | `func header_line_len(name: String, value: String) -> Int` | none | stable `v0.4.0` core | Exact byte count for a single `Name: Value\\r\\n` header line, or `-1` for invalid field names or malformed header values containing CR/LF, non-HTAB controls, or DEL. |
| `lib.core.http` | `func write_header_at(dst: inout []u8, start: Int, name: String, value: String) -> Int` | `mem` | stable `v0.4.0` core | Writes one HTTP header line at `start`, or `-1` for negative starts, invalid field names, malformed header values containing CR/LF, non-HTAB controls, or DEL, wrapped offsets, or missing destination bytes. |
| `lib.core.http` | `func write_crlf_at(dst: inout []u8, start: Int) -> Int` | `mem` | stable `v0.4.0` core | Writes CRLF at `start`, or `-1` for negative starts, wrapped offsets, or missing destination bytes. |
| `lib.core.http` | `func write_ascii_at(dst: inout []u8, start: Int, text: String) -> Int` | `mem` | stable `v0.4.0` core | Writes ASCII text into a caller-owned byte buffer, or `-1` for negative starts, wrapped offsets, or missing destination bytes. |
| `lib.core.http` | `func write_decimal_i32_at(dst: inout []u8, start: Int, value: Int) -> Int` | `mem` | stable `v0.4.0` core | Writes decimal integer text, including `-2147483648`, into a caller-owned byte buffer, or `-1` for negative starts, wrapped offsets, or missing destination bytes. |
| `lib.core.http` | `func digits_i32(value: Int) -> Int` | none | stable `v0.4.0` core | Decimal digit count helper for HTTP status and length fields, including the i32 minimum value. |
| `lib.core.json` | `func encoded_string_len(text: String) -> Int` | none | stable `v0.4.0` core | JSON string serializer sizing helper, including quotes and escape expansion. |
| `lib.core.json` | `func message_object_len(message: String) -> Int` | none | stable `v0.4.0` core | Exact byte count for `{"message":...}` response bodies. |
| `lib.core.json` | `func digits_i32(value: Int) -> Int` | none | stable `v0.4.0` core | Decimal digit count helper for JSON integer field sizing, including the i32 minimum value. |
| `lib.core.json` | `func world_object_len(id: Int, random_number: Int) -> Int` | none | stable `v0.4.0` core | Exact byte count for TechEmpower-style `World` JSON objects, including i32 minimum field values. |
| `lib.core.json` | `func write_message_object(dst: inout []u8, message: String) -> Int` | `mem` | stable `v0.4.0` core | Writes compact `{"message":...}` JSON into a caller-owned byte buffer and returns bytes written, or `-1` when the destination is too short. |
| `lib.core.json` | `func write_message_object_at(dst: inout []u8, start: Int, message: String) -> Int` | `mem` | stable `v0.4.0` core | Writes compact `{"message":...}` JSON at `start` and returns the next byte index, or `-1` for negative starts, wrapped offsets, or missing destination bytes. |
| `lib.core.json` | `func write_json_string(dst: inout []u8, text: String) -> Int` | `mem` | stable `v0.4.0` core | Writes a compact escaped JSON string at the start of a caller-owned byte buffer, or returns `-1` when the destination is too short. |
| `lib.core.json` | `func write_json_string_at(dst: inout []u8, start: Int, text: String) -> Int` | `mem` | stable `v0.4.0` core | Writes a compact escaped JSON string at `start` and returns the next byte index, or `-1` for negative starts, wrapped offsets, or missing destination bytes. |
| `lib.core.postgres` | `func protocol_version_3() -> Int` | none | stable `v0.4.0` core | PostgreSQL protocol version 3.0 integer for startup messages. |
| `lib.core.postgres` | `func int4_oid() -> Int` | none | stable `v0.4.0` core | PostgreSQL `int4` type OID used by TechEmpower prepared-statement paths. |
| `lib.core.postgres` | `func text_oid() -> Int` | none | stable `v0.4.0` core | PostgreSQL `text` type OID used by Fortunes result metadata. |
| `lib.core.postgres` | `func frame_parse() -> Int` | none | stable `v0.4.0` core | PostgreSQL frontend message tag for Parse (`P`). |
| `lib.core.postgres` | `func frame_bind() -> Int` | none | stable `v0.4.0` core | PostgreSQL frontend message tag for Bind (`B`). |
| `lib.core.postgres` | `func frame_describe() -> Int` | none | stable `v0.4.0` core | PostgreSQL frontend message tag for Describe (`D`). |
| `lib.core.postgres` | `func frame_execute() -> Int` | none | stable `v0.4.0` core | PostgreSQL frontend message tag for Execute (`E`). |
| `lib.core.postgres` | `func frame_simple_query() -> Int` | none | stable `v0.4.0` core | PostgreSQL frontend message tag for Simple Query (`Q`). |
| `lib.core.postgres` | `func frame_sync() -> Int` | none | stable `v0.4.0` core | PostgreSQL frontend message tag for Sync (`S`). |
| `lib.core.postgres` | `func frame_terminate() -> Int` | none | stable `v0.4.0` core | PostgreSQL frontend message tag for Terminate (`X`). |
| `lib.core.postgres` | `func frame_authentication() -> Int` | none | stable `v0.4.0` core | PostgreSQL backend message tag for Authentication (`R`). |
| `lib.core.postgres` | `func frame_bind_complete() -> Int` | none | stable `v0.4.0` core | PostgreSQL backend message tag for BindComplete (`2`). |
| `lib.core.postgres` | `func frame_command_complete() -> Int` | none | stable `v0.4.0` core | PostgreSQL backend message tag for CommandComplete (`C`). |
| `lib.core.postgres` | `func frame_data_row() -> Int` | none | stable `v0.4.0` core | PostgreSQL backend message tag for DataRow (`D`). |
| `lib.core.postgres` | `func frame_error_response() -> Int` | none | stable `v0.4.0` core | PostgreSQL backend message tag for ErrorResponse (`E`). |
| `lib.core.postgres` | `func frame_no_data() -> Int` | none | stable `v0.4.0` core | PostgreSQL backend message tag for NoData (`n`). |
| `lib.core.postgres` | `func frame_parse_complete() -> Int` | none | stable `v0.4.0` core | PostgreSQL backend message tag for ParseComplete (`1`). |
| `lib.core.postgres` | `func frame_parameter_status() -> Int` | none | stable `v0.4.0` core | PostgreSQL backend message tag for ParameterStatus (`S`). |
| `lib.core.postgres` | `func frame_ready_for_query() -> Int` | none | stable `v0.4.0` core | PostgreSQL backend message tag for ReadyForQuery (`Z`). |
| `lib.core.postgres` | `func frame_row_description() -> Int` | none | stable `v0.4.0` core | PostgreSQL backend message tag for RowDescription (`T`). |
| `lib.core.postgres` | `func ready_for_query_idle_status() -> Int` | none | stable `v0.4.0` core | ReadyForQuery idle transaction status byte (`I`). |
| `lib.core.postgres` | `func ready_for_query_in_transaction_status() -> Int` | none | stable `v0.4.0` core | ReadyForQuery in-transaction status byte (`T`). |
| `lib.core.postgres` | `func ready_for_query_failed_transaction_status() -> Int` | none | stable `v0.4.0` core | ReadyForQuery failed-transaction status byte (`E`). |
| `lib.core.postgres` | `func describe_kind_portal() -> Int` | none | stable `v0.4.0` core | PostgreSQL Describe payload kind for portals (`P`). |
| `lib.core.postgres` | `func describe_kind_statement() -> Int` | none | stable `v0.4.0` core | PostgreSQL Describe payload kind for prepared statements (`S`). |
| `lib.core.postgres` | `func startup_message_len(user: String, database: String, application_name: String) -> Int` | none | stable `v0.4.0` core | Exact byte count for a protocol 3.0 startup message with user, database, and application_name parameters. |
| `lib.core.postgres` | `func write_startup_message(dst: inout []u8, user: String, database: String, application_name: String) -> Int` | `mem` | stable `v0.4.0` core | Writes a PostgreSQL startup message into a caller-owned byte buffer, or returns `-1` when the destination is too short or a startup C-string field contains an embedded NUL. |
| `lib.core.postgres` | `func simple_query_payload_len(query: String) -> Int` | none | stable `v0.4.0` core | PostgreSQL Simple Query length field value for `query`, or `-1` when `query` contains an embedded NUL. |
| `lib.core.postgres` | `func simple_query_frame_len(query: String) -> Int` | none | stable `v0.4.0` core | Exact byte count for a typed Simple Query frontend frame, or `-1` when `query` contains an embedded NUL. |
| `lib.core.postgres` | `func write_simple_query(dst: inout []u8, query: String) -> Int` | `mem` | stable `v0.4.0` core | Writes a typed PostgreSQL Simple Query frame into a caller-owned byte buffer, or returns `-1` when the destination is too short or `query` contains an embedded NUL. |
| `lib.core.postgres` | `func parse_payload_len(statement: String, query: String, param_type_oids: []i32) -> Int` | `mem` | stable `v0.4.0` core | PostgreSQL Parse length field value for a prepared statement and parameter type OIDs, or `-1` when the OID count exceeds the signed i16 protocol range or a statement/query C-string contains an embedded NUL. |
| `lib.core.postgres` | `func parse_frame_len(statement: String, query: String, param_type_oids: []i32) -> Int` | `mem` | stable `v0.4.0` core | Exact byte count for a typed Parse frontend frame, or `-1` when the OID count exceeds the signed i16 protocol range or a statement/query C-string contains an embedded NUL. |
| `lib.core.postgres` | `func write_parse(dst: inout []u8, statement: String, query: String, param_type_oids: []i32) -> Int` | `mem` | stable `v0.4.0` core | Writes a PostgreSQL Parse frame for extended-query prepared statements, or returns `-1` when the destination is too short, the OID count exceeds the signed i16 protocol range, or a statement/query C-string contains an embedded NUL. |
| `lib.core.postgres` | `func bind_text_0_payload_len(portal: String, statement: String) -> Int` | none | stable `v0.4.0` core | PostgreSQL Bind length field value for no text parameters, or `-1` when portal/statement C-string fields contain an embedded NUL. |
| `lib.core.postgres` | `func bind_text_0_frame_len(portal: String, statement: String) -> Int` | none | stable `v0.4.0` core | Exact byte count for a Bind frame with no text parameters, or `-1` when portal/statement C-string fields contain an embedded NUL. |
| `lib.core.postgres` | `func bind_text_1_payload_len(portal: String, statement: String, value0: String) -> Int` | none | stable `v0.4.0` core | PostgreSQL Bind length field value for one text parameter, or `-1` when portal/statement C-string fields contain an embedded NUL. |
| `lib.core.postgres` | `func bind_text_1_frame_len(portal: String, statement: String, value0: String) -> Int` | none | stable `v0.4.0` core | Exact byte count for a Bind frame with one text parameter, or `-1` when portal/statement C-string fields contain an embedded NUL. |
| `lib.core.postgres` | `func bind_text_2_payload_len(portal: String, statement: String, value0: String, value1: String) -> Int` | none | stable `v0.4.0` core | PostgreSQL Bind length field value for two text parameters, or `-1` when portal/statement C-string fields contain an embedded NUL. |
| `lib.core.postgres` | `func bind_text_2_frame_len(portal: String, statement: String, value0: String, value1: String) -> Int` | none | stable `v0.4.0` core | Exact byte count for a Bind frame with two text parameters, or `-1` when portal/statement C-string fields contain an embedded NUL. |
| `lib.core.postgres` | `func write_bind_text_0(dst: inout []u8, portal: String, statement: String) -> Int` | `mem` | stable `v0.4.0` core | Writes a PostgreSQL Bind frame with no text parameters, or returns `-1` when the destination is too short or portal/statement C-string fields contain an embedded NUL. |
| `lib.core.postgres` | `func write_bind_text_1(dst: inout []u8, portal: String, statement: String, value0: String) -> Int` | `mem` | stable `v0.4.0` core | Writes a PostgreSQL Bind frame with one text parameter, or returns `-1` when the destination is too short or portal/statement C-string fields contain an embedded NUL. |
| `lib.core.postgres` | `func write_bind_text_2(dst: inout []u8, portal: String, statement: String, value0: String, value1: String) -> Int` | `mem` | stable `v0.4.0` core | Writes a PostgreSQL Bind frame with two text parameters for update-style paths, or returns `-1` when the destination is too short or portal/statement C-string fields contain an embedded NUL. |
| `lib.core.postgres` | `func describe_portal_payload_len(portal: String) -> Int` | none | stable `v0.4.0` core | PostgreSQL Describe Portal length field value, or `-1` when `portal` contains an embedded NUL. |
| `lib.core.postgres` | `func describe_portal_frame_len(portal: String) -> Int` | none | stable `v0.4.0` core | Exact byte count for a Describe Portal frontend frame, or `-1` when `portal` contains an embedded NUL. |
| `lib.core.postgres` | `func write_describe_portal(dst: inout []u8, portal: String) -> Int` | `mem` | stable `v0.4.0` core | Writes a PostgreSQL Describe Portal frame, or returns `-1` when the destination is too short or `portal` contains an embedded NUL. |
| `lib.core.postgres` | `func describe_statement_payload_len(statement: String) -> Int` | none | stable `v0.4.0` core | PostgreSQL Describe Statement length field value, or `-1` when `statement` contains an embedded NUL. |
| `lib.core.postgres` | `func describe_statement_frame_len(statement: String) -> Int` | none | stable `v0.4.0` core | Exact byte count for a Describe Statement frontend frame, or `-1` when `statement` contains an embedded NUL. |
| `lib.core.postgres` | `func write_describe_statement(dst: inout []u8, statement: String) -> Int` | `mem` | stable `v0.4.0` core | Writes a PostgreSQL Describe Statement frame, or returns `-1` when the destination is too short or `statement` contains an embedded NUL. |
| `lib.core.postgres` | `func execute_payload_len(portal: String, max_rows: Int) -> Int` | none | stable `v0.4.0` core | PostgreSQL Execute length field value, or `-1` when `portal` contains an embedded NUL. |
| `lib.core.postgres` | `func execute_frame_len(portal: String, max_rows: Int) -> Int` | none | stable `v0.4.0` core | Exact byte count for an Execute frontend frame, or `-1` when `portal` contains an embedded NUL. |
| `lib.core.postgres` | `func write_execute(dst: inout []u8, portal: String, max_rows: Int) -> Int` | `mem` | stable `v0.4.0` core | Writes a PostgreSQL Execute frame, or returns `-1` when the destination is too short or `portal` contains an embedded NUL. |
| `lib.core.postgres` | `func sync_frame_len() -> Int` | none | stable `v0.4.0` core | Exact byte count for a PostgreSQL Sync frontend frame. |
| `lib.core.postgres` | `func write_sync(dst: inout []u8) -> Int` | `mem` | stable `v0.4.0` core | Writes a PostgreSQL Sync frame, or returns `-1` when the destination is too short. |
| `lib.core.postgres` | `func terminate_frame_len() -> Int` | none | stable `v0.4.0` core | Exact byte count for a PostgreSQL Terminate frontend frame. |
| `lib.core.postgres` | `func write_terminate(dst: inout []u8) -> Int` | `mem` | stable `v0.4.0` core | Writes a PostgreSQL Terminate frontend frame into a caller-owned byte buffer, or returns `-1` when the destination is too short. |
| `lib.core.postgres` | `func frame_type_at(frame: []u8, start: Int) -> Int` | `mem` | stable `v0.4.0` core | Reads a typed PostgreSQL frame tag at `start`, or `-1` for negative or missing starts. |
| `lib.core.postgres` | `func frame_length_at(frame: []u8, start: Int) -> Int` | `mem` | stable `v0.4.0` core | Reads a typed PostgreSQL frame length field, or `-1` for negative starts, wrapped offsets, missing length bytes, or malformed negative signed lengths. |
| `lib.core.postgres` | `func frame_payload_len_at(frame: []u8, start: Int) -> Int` | `mem` | stable `v0.4.0` core | Returns payload byte count for a typed PostgreSQL frame, or `-1` for invalid, short, or negative-start frame lengths. |
| `lib.core.postgres` | `func frame_total_len_at(frame: []u8, start: Int) -> Int` | `mem` | stable `v0.4.0` core | Returns total byte count for a typed PostgreSQL frame, or `-1` for invalid, short, negative-start, or total-overflow frame lengths. |
| `lib.core.postgres` | `func frame_payload_start(start: Int) -> Int` | none | stable `v0.4.0` core | Returns the payload offset for a typed PostgreSQL frame. |
| `lib.core.postgres` | `func row_description_column_count(payload: []u8, start: Int) -> Int` | `mem` | stable `v0.4.0` core | Reads RowDescription column count from a backend payload, or `-1` for negative starts, missing bytes, or high-bit signed count fields. |
| `lib.core.postgres` | `func row_description_type_oid_at(payload: []u8, start: Int, payload_len: Int, column_index: Int) -> Int` | `mem` | stable `v0.4.0` core | Scans RowDescription metadata and returns one column type OID, or `-1` on missing, malformed, negative-start, or wrapped input. |
| `lib.core.postgres` | `func data_row_column_count(payload: []u8, start: Int) -> Int` | `mem` | stable `v0.4.0` core | Reads DataRow column count from a backend payload, or `-1` for negative starts, missing bytes, or high-bit signed count fields. |
| `lib.core.postgres` | `func data_row_value_len_at(payload: []u8, start: Int, column_index: Int) -> Int` | `mem` | stable `v0.4.0` core | Reads one DataRow value length, returning `-1` for NULL, malformed negative lengths, negative starts, missing indexes, or physically missing positive value bytes. |
| `lib.core.postgres` | `func data_row_value_start_at(payload: []u8, start: Int, column_index: Int) -> Int` | `mem` | stable `v0.4.0` core | Returns one DataRow value start offset, or `-1` for NULL, negative starts, missing indexes, or physically missing positive value bytes. |
| `lib.core.postgres` | `func data_row_i32_at(payload: []u8, start: Int, column_index: Int) -> Int` | `mem` | stable `v0.4.0` core | Parses an ASCII integer DataRow value for TechEmpower `World` rows. |
| `lib.core.postgres` | `func command_complete_affected_rows(payload: []u8, start: Int, payload_len: Int) -> Int` | `mem` | stable `v0.4.0` core | Parses the trailing affected-row count from CommandComplete text, returning `0` for empty, negative-start, wrapped, physically missing, out-of-range i32, or non-trailing digit ranges. |
| `lib.core.postgres` | `func ready_for_query_status(payload: []u8, start: Int) -> Int` | `mem` | stable `v0.4.0` core | Reads ReadyForQuery transaction status byte, or `-1` for negative starts or missing payload bytes. |
| `lib.core.postgres` | `func cstring_end_at(src: []u8, start: Int, limit: Int) -> Int` | `mem` | stable `v0.4.0` core | Finds a NUL terminator inside a bounded byte range, or `-1` for missing, negative-start, reversed, or physically missing ranges. |
| `lib.core.postgres` | `func parse_ascii_i32_at(src: []u8, start: Int, count: Int) -> Int` | `mem` | stable `v0.4.0` core | Parses a bounded ASCII integer from a byte buffer, returning `0` for empty, negative-start, wrapped, physically missing, or out-of-range i32 ranges. |
| `lib.core.postgres` | `func write_ascii_at(dst: inout []u8, start: Int, text: String) -> Int` | `mem` | stable `v0.4.0` core | Writes ASCII text into a caller-owned byte buffer, or `-1` for negative starts, wrapped offsets, or missing destination bytes. |
| `lib.core.postgres` | `func write_cstring_at(dst: inout []u8, start: Int, value: String) -> Int` | `mem` | stable `v0.4.0` core | Writes a NUL-terminated C string into a caller-owned byte buffer, or `-1` for embedded NUL bytes, negative starts, wrapped offsets, or missing destination bytes. |
| `lib.core.postgres` | `func write_cstring_pair_at(dst: inout []u8, start: Int, name: String, value: String) -> Int` | `mem` | stable `v0.4.0` core | Writes two NUL-terminated C strings into a caller-owned byte buffer, or `-1` for embedded NUL bytes, negative starts, wrapped offsets, or missing destination bytes. |
| `lib.core.postgres` | `func write_i32_be_at(dst: inout []u8, start: Int, value: Int) -> Int` | `mem` | stable `v0.4.0` core | Writes a signed big-endian i32 field using two's-complement bytes and returns the next byte index, or `-1` for negative starts, wrapped offsets, or missing destination bytes. |
| `lib.core.postgres` | `func write_i16_be_at(dst: inout []u8, start: Int, value: Int) -> Int` | `mem` | stable `v0.4.0` core | Writes a signed big-endian i16 field using the low two's-complement bytes and returns the next byte index, or `-1` for negative starts, wrapped offsets, or missing destination bytes. |
| `lib.core.postgres` | `func read_i32_be(src: []u8, start: Int) -> Int` | `mem` | stable `v0.4.0` core | Reads a non-negative big-endian i32 field from a caller-owned byte buffer, or `-1` for negative starts, wrapped offsets, missing bytes, or high-bit values that cannot be represented as a non-negative `Int`. |
| `lib.core.postgres` | `func read_i32_be_signed(src: []u8, start: Int) -> Int` | `mem` | stable `v0.4.0` core | Reads a PostgreSQL i32 length field, normalizing any negative signed value, negative start, wrapped offset, or missing byte to `-1`. |
| `lib.core.postgres` | `func read_i16_be(src: []u8, start: Int) -> Int` | `mem` | stable `v0.4.0` core | Reads a big-endian i16 field from a caller-owned byte buffer, or `-1` for negative starts, wrapped offsets, or missing bytes. |
| `lib.core.math` | `func add_i32(a: Int, b: Int) -> Int` | none | stable `v0.3.0` core | Pure integer addition helper. |
| `lib.core.math` | `func min_i32(a: Int, b: Int) -> Int` | none | stable `v0.3.0` core | Pure integer minimum helper. |
| `lib.core.math` | `func max_i32(a: Int, b: Int) -> Int` | none | stable `v0.3.0` core | Pure integer maximum helper. |
| `lib.core.math` | `func clamp_i32(value: Int, lo: Int, hi: Int) -> Int` | none | stable `v0.3.0` core | Pure integer helper; assumes caller chooses sensible bounds. |
| `lib.core.memory` | `func memset_u8(dst: ptr, v: UInt8, n: Int, mem: cap.mem) -> Int` | `mem` | stable `v0.3.0` core | Unsafe byte helper; no allocation, bounds validation, or permission grant. |
| `lib.core.memory` | `func memcpy_u8(dst: ptr, src: ptr, n: Int, mem: cap.mem) -> Int` | `mem` | stable `v0.3.0` core | Unsafe byte helper; caller owns pointer validity and overlap assumptions. |
| `lib.core.networking` | `func default_port_http() -> Int` | none | stable `v0.3.0` core | Standard HTTP port constant. |
| `lib.core.networking` | `func default_port_https() -> Int` | none | stable `v0.3.0` core | Standard HTTPS port constant. |
| `lib.core.networking` | `func clamp_port(port: Int) -> Int` | none | stable `v0.3.0` core | Deterministic port-range policy helper. |
| `lib.core.networking` | `func is_valid_port(port: Int) -> Bool` | none | stable `v0.3.0` core | Port-range validation helper for endpoint configuration. |
| `lib.core.networking` | `func choose_port(preferred: Int, fallback: Int) -> Int` | none | stable `v0.3.0` core | Endpoint configuration helper; `0` preferred port falls back. |
| `lib.core.networking` | `func retry_backoff_ms(attempt: Int, base_ms: Int, max_ms: Int) -> Int` | none | stable `v0.3.0` core | Deterministic retry backoff policy helper; negative bases clamp to `0`, non-negative caps are honored before overflow, and uncapped overflow saturates to `2147483647`. |
| `lib.core.serialization` | `func clamp_u8(value: Int) -> Int` | none | stable `v0.3.0` core | Pure packing helper; not a general serializer. |
| `lib.core.serialization` | `func pack_u8_pair(high: Int, low: Int) -> Int` | none | stable `v0.3.0` core | Pure two-byte packing helper; not a wire-format guarantee. |
| `lib.core.serialization` | `func unpack_u8_high(packed: Int) -> Int` | none | stable `v0.3.0` core | Pure unpack helper; negative packed input returns `0`. |
| `lib.core.serialization` | `func unpack_u8_low(packed: Int) -> Int` | none | stable `v0.3.0` core | Pure unpack helper; negative packed input returns `0`. |
| `lib.core.serialization` | `func checksum_u8(values: []u8) -> Int` | `mem` | stable `v0.3.0` core | Simple checksum helper; not authentication or encryption. |
| `lib.core.slices` | `func sum_i32(values: []i32) -> Int` | `mem` | stable `v0.3.0` core | Slice scan helper for supported element type only. |
| `lib.core.slices` | `func weighted_sum_i32(values: []i32) -> Int` | `mem` | stable `v0.3.0` core | Slice scan helper for supported element type only. |
| `lib.core.slices` | `func sum_u8(values: []u8) -> Int` | `mem` | stable `v0.3.0` core | Slice scan helper for supported element type only. |
| `lib.core.strings` | `func ascii_len(text: String) -> Int` | none | stable `v0.3.0` core | ASCII/byte-oriented helper; no Unicode normalization guarantee. |
| `lib.core.strings` | `func ascii_sum(text: String) -> Int` | none | stable `v0.3.0` core | ASCII/byte-oriented helper; no Unicode normalization guarantee. |
| `lib.core.strings` | `func is_empty(text: String) -> Bool` | none | stable `v0.3.0` core | ASCII/byte-oriented helper built from `ascii_len`. |
| `lib.core.sync` | `func merge_status(lhs: Int, rhs: Int) -> Int` | none | stable `v0.3.0` core | Pure status helper; not a runtime synchronization primitive. |
| `lib.core.sync` | `func all_ready(lhs: Bool, rhs: Bool) -> Bool` | none | stable `v0.3.0` core | Pure boolean helper; not a runtime synchronization primitive. |
| `lib.core.sync` | `func spin_countdown(start: Int, ticks: Int) -> Int` | none | stable `v0.3.0` core | Pure countdown helper; no sleeping or scheduling behavior. |
| `lib.core.sync` | `func barrier_target(workers: Int) -> Int` | none | stable `v0.3.0` core | Pure clamp helper; not a runtime barrier. |
| `lib.core.testing` | `func assert_true(value: Bool) -> Int` | none | stable `v0.3.0` core | Status-code helper; `0` means pass and `1` means fail. |
| `lib.core.testing` | `func assert_false(value: Bool) -> Int` | none | stable `v0.3.0` core | Status-code helper; `0` means pass and `1` means fail. |
| `lib.core.testing` | `func assert_eq_i32(actual: Int, expected: Int) -> Int` | none | stable `v0.3.0` core | Status-code helper; `0` means pass and `1` means fail. |
| `lib.core.testing` | `func combine(lhs: Int, rhs: Int) -> Int` | none | stable `v0.3.0` core | Status-code helper; returns first non-zero status. |
| `lib.core.time` | `func millis_from_seconds(seconds: Int) -> Int` | none | stable `v0.3.0` core | Pure duration arithmetic; negative input clamps to `0` and positive overflow saturates to `Int` max. |
| `lib.core.time` | `func seconds_from_millis(milliseconds: Int) -> Int` | none | stable `v0.3.0` core | Pure duration arithmetic; negative input clamps to `0`. |
| `lib.core.time` | `func clamp_timeout_ms(value: Int, lo: Int, hi: Int) -> Int` | none | stable `v0.3.0` core | Pure duration arithmetic; assumes caller chooses sensible bounds. |
| `lib.core.time` | `func add_duration_ms(base: Int, delta: Int) -> Int` | none | stable `v0.3.0` core | Pure duration arithmetic; adds `delta` to `base`, clamps a negative summed result to `0`, and saturates positive overflow to `Int` max. |

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
modifies read/write readiness interest, deletes the epoll registration, performs
a zero-timeout wait, exercises fd/flag extraction helpers, closes the fds, and preserves the stable exit value `42`.
Compiler integration coverage also runs real local TCP
client/server exchanges through `connect`/`accept4`/`read`/`recv`/`write`/`send`, an epoll readiness
path, nonblocking `accept4` convenience, and a socket-option smoke.
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
helpers, and `EPOLLIN`/`EPOLLOUT`/`EPOLLERR`/`EPOLLHUP` predicates for local client/server slices; full
TechEmpower event-loop and PostgreSQL paths are not implied by this slice.

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
real linux-x64 socket open/bind/connect/listen/accept/read/recv/write/send/nonblocking/close
helpers, `SO_REUSEPORT`/`TCP_NODELAY`, and epoll add/mod/delete plus wait-one readiness, while
broader event-loop abstractions, socket-option APIs, and full database APIs are expected behind future `lib.core.net`
expansion and higher-level `lib.core.postgres` driver layers instead of extending this policy-helper module.
HTTP/1.1 String and byte-buffer request-line routing, byte-buffer request-head framing, and response byte-buffer serialization helpers live in `lib.core.http`;
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
measurement evidence only; it is not an official TechEmpower result, an
external production database claim, production database benchmark, measured
speed comparison, or a source-level full driver API promotion.

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

| Experimental mirror | Stable target | `v0.4.0` status |
| --- | --- | --- |
| `lib.experimental.async` | `lib.core.async` | Compatibility mirror; source note directs stable callers to `lib.core.async`. |
| `lib.experimental.collections` | `lib.core.collections` | Compatibility mirror; source note directs stable callers to `lib.core.collections`. |
| `lib.experimental.crypto` | `lib.core.crypto` | Compatibility mirror for stable crypto interface helpers. |
| `lib.experimental.filesystem` | `lib.core.filesystem` | Compatibility mirror for filesystem helpers, including the linux-x64 host-backed `exists` slice and the fs-only linux-x86/linux-x32 smokes. |
| `lib.experimental.io` | `lib.core.io` | Compatibility mirror for capability/MMIO wrappers; callers still need matching effects. |
| `lib.experimental.math` | `lib.core.math` | Compatibility mirror; explicitly present in `lib/experimental/math.tetra`. |
| `lib.experimental.memory` | `lib.core.memory` | Compatibility mirror; explicitly present in `lib/experimental/memory.tetra` and still requires `cap.mem`. |
| `lib.experimental.networking` | `lib.core.networking` | Compatibility mirror for stable endpoint policy helpers. |
| `lib.experimental.serialization` | `lib.core.serialization` | Compatibility mirror for packing/checksum helpers; not a general serializer guarantee. |
| `lib.experimental.slices` | `lib.core.slices` | Compatibility mirror; source note directs stable callers to `lib.core.slices`. |
| `lib.experimental.strings` | `lib.core.strings` | Compatibility mirror; source note directs stable callers to `lib.core.strings`. |
| `lib.experimental.sync` | `lib.core.sync` | Compatibility mirror for pure status/countdown helpers; not runtime synchronization. |
| `lib.experimental.testing` | `lib.core.testing` | Compatibility mirror; source note directs stable callers to `lib.core.testing`. |
| `lib.experimental.time` | `lib.core.time` | Compatibility mirror for pure duration arithmetic; no clock or scheduler access. |

Generated API docs label `lib.experimental.*` modules as experimental so they
are not confused with the stable `lib.core.*` API namespace. The production
claim is limited to mirror forwarding compatibility, not new stable APIs under
`lib.experimental.*`.

## Stable Type Display

Public compiler output and generated API docs use canonical builtin names:

| Source aliases | Canonical name | Notes |
| --- | --- | --- |
| `Int`, `i32` | `i32` | Default integer literal type. |
| `UInt8`, `Byte`, `u8` | `u8` | Slice element supported by `[]u8` and string storage. |
| `UInt16`, `u16` | `u16` | Native-first slice element supported by `[]u16`. |
| `Bool`, `bool` | `bool` | Boolean literal and condition type. |
| `String`, `str` | `str` | Two-slot UTF-8 string/slice shape; `.len` and String view constructors use byte lengths. |
| `ConsentToken` | `consent.token` | Privacy/consent capability token. |
| `SecretInt` | `secret.i32` | Privacy-protected integer wrapper. |

Structural types use deterministic field order and slot counts. `str`, `[]u8`,
`[]u16`, `[]i32`, and `[]bool` are two-slot values (`ptr`, `len`). Safe code
may read `String.len` as a byte length but cannot assign `String.ptr` or
`String.len`; `text.window(start, count)`, `text.prefix(count)`, and
`text.suffix(start)` return checked byte views of the same string storage.
`text.borrow()` and `xs.borrow()` return no-allocation immutable borrowed
views, `text.copy()` and `xs.copy()` return independently owned storage, and
`copy_into(dst)` copies bytes/elements into a caller-owned destination after a
checked length guard.
`T?` adds one presence tag slot to the payload slots. Opaque handles such as `ptr`, `island`, `actor`,
`cap.io`, `cap.mem`, and `task.*` are not interchangeable even when they occupy
one slot.

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
