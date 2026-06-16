# Standard Library Guide

Status: user guide for release-covered stdlib expectations.

The stable module list and examples are documented in `docs/spec/stdlib.md`.
Module naming and versioning rules are documented in
`docs/spec/stdlib_naming_versioning.md`. The current support boundary is
`docs/spec/current_supported_surface.md`.
The production backend web stack and its runtime/server API are summarized in
`docs/user/backend_web_platform.md`.

## Finding Stable Modules

Stable modules live under `lib/core`. Experimental or lower-level modules must
not be described as stable release APIs unless the current release checklist
and `docs/spec/current_supported_surface.md` include fresh evidence for them.

Stability labels used below:

- `stable helper`: release-covered helper behavior for the current profile.
- `capability-bound helper`: release-covered helper that still requires matching
  `uses` effects and capability tokens at call sites.

## Generated Docs Naming Policy

Generated docs use a dotted module path for files that declare `module ...`.
For example, stable stdlib modules render as `lib.core.<name>`,
experimental stdlib modules render as `lib.experimental.<name>`, and smoke
files that declare modules render as names like `examples.core_math_smoke`.

Generated docs use a portable file path for source files without a module
declaration. Those entries remain spelled with slashes, such as
`examples/flow_hello.tetra`. Treat dotted `examples.*` headings as module names
and `examples/...` headings as file paths; both forms can appear in one
generated docs run.

## Stable Module Choices

| Need | Import | Example | Effects |
| --- | --- | --- | --- |
| Integer helpers and small arithmetic choices | `import lib.core.math as math` | `examples/core_math_smoke.tetra` | none |
| Explicit memory capability wrappers | `import lib.core.memory as memory` | `examples/core_memory_smoke.tetra` | `mem` |
| Capability tokens for host-like surfaces | `import lib.core.capability as cap` | `examples/core_memory_smoke.tetra` | `capability`, `io`, `mem` |
| Capability-gated IO helpers | `import lib.core.io as io` | `examples/core_io_smoke.tetra` | `capability`, `io`, `mmio` |
| Test status helpers | `import lib.core.testing as testing` | `examples/core_testing_smoke.tetra` | none |
| Slice summation helpers (`sum_i32`, `weighted_sum_i32`, `sum_u8`) | `import lib.core.slices as slices` | `examples/core_slices_smoke.tetra` | `mem` |
| ASCII length, ASCII sum, and empty checks (`ascii_len`, `ascii_sum`, `is_empty`) | `import lib.core.strings as strings` | `examples/core_strings_smoke.tetra` | none |
| Caller-owned UTF-8 text buffer helpers | `import lib.core.text as text` | `examples/core_text_smoke.tetra` | none |
| Bounded Surface string tables, locale fallback, formatting hooks, and RTL placeholder nonclaims | `import lib.core.i18n as i18n` | `examples/core_i18n_smoke.tetra`, `examples/surface_reference_localized_form.tetra` | none |
| Caller-owned Surface app command/reducer helpers | `import lib.core.surface_app as appmodel` | `examples/core_surface_app_smoke.tetra`, `examples/surface_app_model.tetra` | none |
| Scoped Linux Surface app-shell state helpers with `electron-feature-ledger-v1`, `surface-security-permission-v1`, and `surface-performance-budget-v1` release evidence | `import lib.core.surface_app_shell as shell` | `examples/core_surface_app_shell_smoke.tetra`, `examples/surface_linux_app_shell_notes.tetra` | none |
| Generic collection views and `[]i32` scans | `import lib.core.collections as collections` | `examples/core_collections_smoke.tetra` | `mem` |
| Tiny serialization combinators | `import lib.core.serialization as serialization` | `examples/core_serialization_smoke.tetra` | `mem` |
| Filesystem path helpers and host-backed `exists` | `import lib.core.filesystem as filesystem` | `examples/core_filesystem_smoke.tetra` | `io` |
| Linux TCP socket client/server I/O helpers | `import lib.core.net as net` | `examples/core_net_smoke.tetra` | `io, mem` |
| Networking endpoint policy helpers | `import lib.core.networking as networking` | `examples/core_networking_smoke.tetra` | none |
| HTTP/1.1 String/byte-buffer request routing, request-head framing, and response byte-buffer helpers | `import lib.core.http as http` | `examples/core_http_smoke.tetra` | `mem` |
| JSON byte-buffer response helpers | `import lib.core.json as json` | `examples/core_json_smoke.tetra` | `mem` |
| PostgreSQL wire-frame byte-buffer helpers | `import lib.core.postgres as pg` | `examples/core_postgres_smoke.tetra`, `examples/core_postgres_prepared_smoke.tetra`, `examples/core_postgres_result_smoke.tetra` | `mem` |
| Async helper functions | `import lib.core.async as async` | `examples/core_async_smoke.tetra` | none |
| Synchronization status helpers | `import lib.core.sync as sync` | `examples/core_sync_smoke.tetra` | none |
| Time duration/status helpers | `import lib.core.time as time` | `examples/core_time_smoke.tetra` | none |
| Crypto interface helpers | `import lib.core.crypto as crypto` | `examples/core_crypto_smoke.tetra` | `mem` |
| Planned Tetra Surface host/frame/event wrappers | `import lib.core.surface as surface` | `examples/core_surface_smoke.tetra` | `surface`, `alloc`, `mem` |
| Planned Tetra Surface software draw helpers | `import lib.core.draw as draw` | `examples/core_draw_smoke.tetra` | `mem` |
| Stable Surface v1 widget style and theme helpers | `import lib.core.style as style` | `examples/core_style_smoke.tetra` | none |
| Planned Tetra Surface static component helpers | `import lib.core.component as component` | `examples/core_component_smoke.tetra` | none |
| Experimental Surface Block System data model | `import lib.core.block as block` | `examples/core_block_smoke.tetra` | alloc, mem |
| Experimental Surface Morph Capsule recipe layer | `import lib.core.morph as morph` | `examples/core_morph_smoke.tetra`, `examples/surface_morph_command_palette.tetra`, `examples/surface_morph_project_dashboard.tetra`, `examples/surface_morph_settings.tetra`, `examples/surface_morph_editor_shell.tetra`, `examples/surface_morph_glass_panel.tetra` | mem |
| Experimental Tetra Surface accessibility metadata helpers | `import lib.core.accessibility as accessibility` | `examples/core_accessibility_smoke.tetra` | none |
| Experimental Tetra Surface minimal widget helpers | `import lib.core.widgets as widgets` | `examples/core_widgets_smoke.tetra` | none |

Call-shape reminders used by generated docs and smoke examples:
`slices.sum_i32(values)`, `slices.weighted_sum_i32(values)`,
`slices.sum_u8(values)`, `strings.ascii_len(value)`,
`strings.ascii_sum(value)`, `strings.is_empty(value)`,
`collections.len_i32(values)`, `collections.contains_i32(values, needle)`,
`collections.count_i32(values, needle)`, and
`collections.first_or_i32(values, fallback)`,
`collections.vec_from_slice(values)`, `collections.vec_len(vec)`,
`collections.vec_get_or(vec, index, fallback)`,
`collections.hash_map_from_slices(keys, values)`,
`collections.hash_map_len(map)`, and the specialized
`collections.hash_map_get_i32_i32_or(map, key, fallback)`,
`json.write_message_object(buffer, message)`, and
`json.write_json_string(buffer, value)`,
`http.write_plaintext_response(buffer, server, date, keep_alive)`, and
`http.write_json_message_response(buffer, server, date, message, keep_alive)`,
`http.request_head_len_bytes(buffer, length)`,
`http.route_tech_empower_bytes_at(buffer, start, length)`, and
`http.request_keep_alive_bytes_at(buffer, start, length)`,
and `net.socket_tcp4(io_cap)`, `net.bind_tcp4_loopback(fd, port, io_cap)`,
`net.connect_tcp4_loopback(fd, port, io_cap)`,
`net.listen(fd, backlog, io_cap)`, `net.accept4(fd, flags, io_cap)`,
`net.accept_nonblocking(fd, io_cap)`,
`net.read(fd, buffer, start, count, io_cap)`,
`net.recv(fd, buffer, start, count, io_cap)`,
`net.write(fd, buffer, start, count, io_cap)`,
`net.send(fd, buffer, start, count, io_cap)`,
`net.epoll_create(io_cap)`, `net.epoll_ctl_add_read(epfd, fd, io_cap)`,
`net.epoll_ctl_add_read_write(epfd, fd, io_cap)`,
`net.epoll_ctl_mod_read(epfd, fd, io_cap)`,
`net.epoll_ctl_mod_read_write(epfd, fd, io_cap)`,
`net.epoll_ctl_delete(epfd, fd, io_cap)`,
`net.epoll_wait_one(epfd, timeout_ms, io_cap)`,
`net.epoll_wait_one_into(epfd, event, timeout_ms, io_cap)`,
`net.epoll_event_fd(event)`, `net.epoll_event_flags(event)`,
`net.epoll_event_readable(flags)`, `net.epoll_event_writable(flags)`,
`net.epoll_event_has_error(flags)`, `net.epoll_event_hung_up(flags)`,
`net.set_nonblocking(fd, io_cap)`, `net.set_reuseport(fd, io_cap)`,
`net.set_tcp_nodelay(fd, io_cap)`, and `net.close(fd, io_cap)`.
For TechEmpower request-line classification, use
`http.route_tech_empower(request)` for request text or
`http.route_tech_empower_bytes(buffer, length)` for bytes read from a socket,
then compare the result with route sentinels such as `http.route_plaintext()`
or `http.route_queries()`. The HTTP/1.1 request line must end with CRLF;
LF-only or bare-CR terminators are malformed. Request targets must start with
`/` and use visible ASCII bytes before their separating space; tab/control
target bytes are malformed. For pipelined buffers, first split complete
request heads with `http.request_head_len_bytes(buffer, length)` and then use
the `_at` helpers for each request window.

`lib.core.filesystem` now has a capability-gated linux-x64 `exists` slice plus
pure path-shape helpers. `lib.core.crypto` is a stable crypto interface-helper
surface. `lib.core.networking` is a stable endpoint policy-helper surface.
`lib.core.surface`, `lib.core.draw`, `lib.core.style`,
`lib.core.component`, `lib.core.block`, `lib.core.morph`, `lib.core.surface_app`,
`lib.core.surface_app_shell`, and `lib.core.widgets` are Tetra Surface modules
for the pure-Tetra UI direction. The current evidence covers headless
frame/event/checksum reports, Linux-x64 starter Host ABI open/present/close
probe reports, Linux-x64 real-window Wayland shm evidence for
`examples/surface_window_counter.tetra`, and the scoped Linux app-shell subset
in `examples/surface_linux_app_shell_notes.tetra`; browser runtime and the
scoped `production-text-input-v1` baseline are release-covered separately, while
full String-level IME editing, rich text, bidi shaping, grapheme-cluster caret
movement, and broader all-platform Surface support remain unpromoted. The
app-shell import does not grant ambient host permissions; release evidence
records `surface-security-permission-v1` default-deny filesystem/network rows,
scoped clipboard policy, capability-checked process boundaries, and local
hashed asset/font/image safety. The same app-shell release report records
`surface-performance-budget-v1` local startup/frame/memory/RSS/cache/
framebuffer/binary-size/CPU-proxy evidence without promoting external benchmark
results or unsupported Electron speed comparisons.
`lib.core.net` is a stable linux-x64 TCP socket client/server I/O slice for
open/bind/connect/listen/accept/read/recv/write/send/nonblocking/close, `SO_REUSEPORT`,
`TCP_NODELAY`, plus epoll
create/add-read/add-read-write/mod-read/mod-read-write/delete/wait-one
and wait-one-into readiness flags, nonblocking accept convenience, and
read/write/error/hangup flag predicates through a caller-provided `cap.io` token.
`lib.core.http` is a stable executable helper surface for TechEmpower
String and byte-buffer request-line routing, request-head framing for pipelined
buffers, keep-alive policy, compact HTTP/1.1 response heads, and
TechEmpower-style plaintext/JSON responses. Response-head helpers return
sentinels for non-three-digit status codes and negative Content-Length values
rather than serializing malformed headers, and reject CR/LF header injection in
reason text and response header values plus non-HTAB control-byte header
values.
`lib.core.postgres` is a stable executable helper surface for PostgreSQL
startup, Simple Query, extended-query Parse/Bind/Describe/Execute/Sync,
RowDescription/DataRow/CommandComplete/ReadyForQuery inspection, Terminate,
and big-endian wire-frame byte buffers. Its bounded ASCII i32 parser returns
`0` for malformed, missing, or out-of-range integer text while preserving
`-2147483648` and `2147483647`; CommandComplete affected-row parsing uses the
same i32 boundary policy and only returns counts from a trailing digit run.
DataRow value helpers return sentinels when a positive advertised value length
is not physically present in the caller-owned byte buffer. Frame total-length
helpers also return sentinels when the typed frame length would overflow `Int`.
Parse-frame helpers return sentinels when the parameter type OID count exceeds
the signed i16 protocol range, and RowDescription/DataRow count readers return
sentinels for high-bit signed i16 count fields. PostgreSQL C-string length and
writer helpers return sentinels for embedded NUL bytes in startup, query,
statement, or portal fields instead of truncating the wire payload.
P19.3 additionally records bounded internal driver/pool evidence through
`tetra.stdlib.postgresql.production_driver.v1` and a checked
`p19.3_postgres_source_first` dry-run gate for DB single query, multiple
queries, updates, and fortunes source rows. The closure links that source gate
to checked local SCRAM/PostgreSQL reports accepted by
`validate-techempower-report`. This does not promote a full source-level
PostgreSQL driver API or measured speed comparison. It makes no official
TechEmpower result claim, no production database benchmark claim, no external
production database deployment claim, no C++/Rust parity claim, and no P20
performance matrix claim.
`lib.core.json` is a stable executable byte-buffer helper surface for compact
JSON response bodies. The runtime package used by backend services also has a
generic deterministic JSON value parser/writer for objects, arrays, strings,
numbers, booleans, and null.

## Capability Unsafe Boundary Recipe

Use `import lib.core.capability as cap` when a stable helper needs an explicit
`cap.mem` or `cap.io` token. Imports and `uses` declarations are only audit
surface: they do not manufacture tokens, validate pointers, or grant host
permission by themselves.

The current recipe is:

1. Declare the effects required by the token and the operation. `cap.mem()`
   needs `uses capability, mem`; `cap.io()` needs `uses capability, io`; MMIO
   calls also need `uses mmio`.
2. Enter a narrow `unsafe:` block for capability acquisition.
3. Call `cap.mem()` or `cap.io()` inside that block.
4. Pass the token to the helper that requires it, such as
   `memory.memset_u8(..., mem_cap)` or `io.mmio_read_i32(..., io_cap)`.
5. Keep the token flow local and auditable. The token proves that the caller
   crossed the reviewed unsafe boundary; it does not prove that an address,
   length, device register, or buffer is valid.

Minimal memory-token flow:

```tetra
import lib.core.capability as cap
import lib.core.memory as memory

func clear_four_bytes(dst: ptr) -> Int
uses capability, mem:
    unsafe:
        let mem_cap: cap.mem = cap.mem()
        return memory.memset_u8(dst, 0, 4, mem_cap)
    return 0
```

Minimal IO-token flow:

```tetra
import lib.core.capability as cap
import lib.core.io as io

func read_mmio_word(addr: ptr) -> Int
uses capability, io, mmio:
    unsafe:
        let io_cap: cap.io = cap.io()
        return io.mmio_read_i32(addr, io_cap)
    return 0
```

`lib.core.io.capability_io()` is the IO-module equivalent for creating a
`cap.io` token; it has the same `uses capability, io` boundary expectation.

## IO/MMIO Helper Contract

Use `import lib.core.io as io` for the current stable MMIO wrapper surface.
These helpers are capability-gated and describe MMIO-shaped operations. In the
current backend, MMIO reads and writes lower to normal memory loads and stores,
so this is not a production device-driver or host-IO abstraction.

| Function | Required effects | Behavior |
| --- | --- | --- |
| `io.capability_io()` | `capability`, `io` | Returns a `cap.io` token from the reviewed unsafe boundary. It does not perform MMIO by itself. |
| `io.mmio_read_i32(addr, io_cap)` | `io`, `mmio` | Reads the current `i32` value at `addr` through the supplied `cap.io` token and returns it as `Int`. |
| `io.mmio_write_i32(addr, value, io_cap)` | `io`, `mmio` | Writes `value` as an `i32` at `addr` through the supplied `cap.io` token and returns the written value. |

Minimal MMIO example:

```tetra
import lib.core.io as io

func mmio_roundtrip() -> Int
uses alloc, capability, io, mem, mmio:
    unsafe:
        let io_cap: cap.io = io.capability_io()
        let p: ptr = core.alloc_bytes(4)
        let _: Int = io.mmio_write_i32(p, 42, io_cap)
        return io.mmio_read_i32(p, io_cap)
    return 0
```

The caller remains responsible for choosing a valid address. The `cap.io` token
and `uses io, mmio` effects are gates for the operation; they are not bounds
checks, device discovery, ordering beyond the documented MMIO operation shape,
or broad host permission grants.

## Collections Helper Contract

`lib.core.collections` exposes a narrow stable generic collection-view surface
plus the older `[]i32` helper scans. `collections.Vec<T>` wraps a caller-owned
`[]T`, and `collections.HashMap<K,V>` wraps caller-owned parallel key/value
slices. These views do not allocate storage internally, resize, sort, mutate
the underlying slices, provide iterator objects, or implement generic hashing
and equality protocols.

P7 adds an internal runtime/storage-planning model for region-aware `Vec`,
`StringBuilder`, `HashMap`, `ByteBuffer`, and `ArenaBuffer` evidence, but that
model lives under `compiler/internal/stdlibrt`. It is used to verify storage
reports and safe-view provenance for the web/runtime stack; it is not the
allocator-backed runtime for the source-level `lib.core.collections` generic
views.

The P19.1 benchmark gate has a checked dry-run truth-bench-harness artifact
for `p19.1_generic_collections`: a hash-table-equivalent Tetra/C++/Rust source
shape with matching algorithm/input metadata and Tetra proof/allocation/bounds
and performance report paths. It is not a runtime measurement, C++/Rust parity
claim, and makes no external benchmark result claim.

- `collections.vec_from_slice(values)` creates a generic `Vec<T>` view over a
  caller-owned slice and records the logical length by scanning it.
- `collections.vec_len(vec)` returns the logical length captured by
  `vec_from_slice`.
- `collections.vec_first_or(vec, fallback)` returns the first item when present,
  otherwise `fallback`.
- `collections.vec_get_or(vec, index, fallback)` scans to `index` and returns
  `fallback` for negative or missing indexes.
- `collections.hash_map_from_slices(keys, values)` creates a generic
  `HashMap<K,V>` view over caller-owned parallel slices.
- `collections.hash_map_len(map)` returns the captured key count.
- `collections.hash_map_first_value_or(map, fallback)` returns the first value
  when present, otherwise `fallback`.
- `collections.hash_map_get_i32_i32_or(map, key, fallback)` and
  `collections.hash_map_get_u8_i32_or(map, key, fallback)` are the current
  concrete lookup specializations.

- `collections.len_i32(values)` counts the number of `i32` elements by scanning
  the slice and returns that count.
- `collections.contains_i32(values, needle)` returns true when any element is
  equal to `needle`, otherwise false.
- `collections.count_i32(values, needle)` returns the number of elements equal
  to `needle`.
- `collections.first_or_i32(values, fallback)` returns the first element when
  the slice is non-empty, otherwise `fallback`.

The scan helpers require `uses mem` because they scan slices supplied by the
caller. They do not allocate, mutate the slice, or validate ownership beyond
the current slice/effect checks.

## Memory Helper Contract

Use `import lib.core.memory as memory` for capability-bound byte helpers. Calls
still require matching `uses mem` effects and an explicit `cap.mem` token; the
import alone does not grant memory access or validate pointers.

- `memory.memset_u8(dst, value, n, mem)` writes the `UInt8` value to `n` bytes
  starting at `dst` and returns `0`. Passing `0` as the value is the documented
  clear-buffer pattern once the caller has selected a valid writable region.
  Negative `n` values are rejected before the byte loop and return status `2`.
- `memory.memcpy_u8(dst, src, n, mem)` copies `n` bytes from `src` to `dst` in
  increasing byte order and returns `0`. Negative `n` values are rejected before
  the byte loop and return status `2`.

Both helpers are thin capability wrappers over caller-provided regions. The
caller remains responsible for valid readable/writable memory, suitable sizes,
and avoiding overlap assumptions; these helpers are not bounds checks, null
checks, allocation helpers, or host permission grants.

## Writing Raw Memory Safely

For the Memory Production Core, `cap.mem` is not ownership: the token
allows a raw operation, but it does not prove that the pointer still belongs to
the caller, that the region is large enough, or that the same region is not
being accessed through another alias. Always check sizes before calling
`memory.memcpy_u8` or `memory.memset_u8`, keep the token and pointer flow local,
and avoid transferring memory-bearing values across task/actor boundaries unless
the checker accepts the transfer.

The runtime bounds diagnostics are a separate production requirement. Until a
validated `tetra.memory.production.v1` report proves those diagnostics for the
exercised Linux-x64 path, these helpers should be used as explicit unsafe
building blocks rather than treated as a complete production allocator/runtime
memory model.

The current Linux-x64 slice rejects negative `core.ptr_add` offsets as a
runtime bounds diagnostic before returning the derived pointer. Still check
sizes before calling helpers: non-negative offsets are not yet full upper-bound
proofs for arbitrary raw pointers.

For pointers returned directly by `core.alloc_bytes`, allocation-base `core.ptr_add` upper bounds
reject offsets greater than or equal to the requested allocation size. This
covers helper loops and direct `core.load_*`/`core.store_*` calls whose address
is a visible `core.ptr_add(base, offset, mem)`, but callers should still avoid
passing arbitrary derived pointers as if they carried complete provenance.

For pointers returned directly by `core.alloc_bytes`, allocation-base `core.store_i32` width bounds
reject a 4-byte store into allocations smaller than 4 bytes. Treat this as a
narrow Linux-x64 runtime diagnostic slice; direct visible base+offset raw
accesses get offset+width checks, but stored arbitrary derived pointers still
need a future provenance table for general guarantees.

For pointers returned directly by `core.alloc_bytes`, allocation-base `core.store_ptr` width bounds
reject an 8-byte pointer store into allocations smaller than 8 bytes. The
current Linux-x64 backend shares that allocation-base check for `core.load_ptr`,
and direct visible base+offset pointer loads/stores get offset+width checks, but
complete pointer-slot safety for arbitrary derived pointers remains unfinished.

The stable helpers reject negative `memcpy_u8` and `memset_u8` lengths with
status `2` before entering the raw byte loop. Positive lengths still rely on
the caller selecting valid regions plus the runtime checks available for the
exercised Linux-x64 path.

## Filesystem Contract

`lib.core.filesystem` contains one host-backed linux-x64 slice and several pure
string-scanning helpers. `filesystem.exists(path, io_cap)` calls the runtime
`__tetra_fs_exists(path_ptr, path_len, io_cap)` ABI, requires an explicit
`cap.io` token, returns true when the host path exists, and returns false when
the path is missing, contains an embedded NUL byte, is invalid, too long, or
unsupported by the target runtime. The runtime copies `ptr,len` into a bounded
NUL-terminated buffer before using the host filesystem API.

The path-shape helpers scan the ASCII slash character `/`. They do not inspect
permissions, normalize path syntax, resolve `.` or `..`, follow symlinks, or
grant filesystem access by import alone.

Safe intended uses are limited to path-shape checks in tests, examples, and
configuration defaults:

- `filesystem.has_leading_slash(path)` returns true only when the first
  character is `/`. The empty string returns false.
- `filesystem.ends_with_slash(path)` returns true only when the final character
  is `/`. The empty string returns false.
- `filesystem.is_root(path)` returns true only for `/`; other strings with a
  leading slash, trailing slash, or multiple slash characters are not root.
- `filesystem.slash_count(path)` counts every `/` character in the string.
- `filesystem.directory_depth(path)` counts non-empty path segments separated
  by `/`. Leading, trailing, and repeated slash characters do not add empty
  segments.
- `filesystem.exists(path, io_cap)` checks host existence on linux-x64 through
  the runtime ABI, and has pure `fs_exists`-only linux-x86/linux-x32 smokes.
  Embedded NUL bytes are rejected instead of truncating to a host prefix.
  linux-x86/linux-x32 programs that mix filesystem calls with scheduler runtime
  surfaces, and other unsupported native targets, report a filesystem runtime
  diagnostic; WASM targets reject the filesystem runtime builtin.

| Path | `has_leading_slash` | `ends_with_slash` | `is_root` | `slash_count` | `directory_depth` |
| --- | --- | --- | --- | --- | --- |
| `""` | false | false | false | `0` | `0` |
| `"/"` | true | true | true | `1` | `0` |
| `"/tmp/cache"` | true | false | false | `2` | `2` |
| `"tmp/cache"` | false | false | false | `1` | `2` |
| `"/tmp/"` | true | true | false | `2` | `1` |

Warning: `filesystem.exists` is an existence probe only. It does not open files,
return metadata, distinguish permission errors from missing paths, or imply
cross-platform filesystem support.

## Serialization Boundary Examples

Use `import lib.core.serialization as serialization` for small byte-oriented
helpers that make boundary behavior explicit before values are packed or
unpacked.

- `serialization.clamp_u8(value)` returns `0` for negative values, `255` for
  values greater than `255`, and the original value for integers already in the
  inclusive `0..255` range. For example, `clamp_u8(-7) == 0`,
  `clamp_u8(42) == 42`, and `clamp_u8(300) == 255`.
- `serialization.pack_u8_pair(high, low)` clamps both inputs with
  `clamp_u8`, then returns `high * 256 + low`. For example,
  `pack_u8_pair(-1, 300) == 255` because the packed high byte becomes `0` and
  the packed low byte becomes `255`.
- `serialization.unpack_u8_high(packed)` and
  `serialization.unpack_u8_low(packed)` both return `0` for a negative packed
  value. For non-negative packed values, the high byte is `packed / 256`
  clamped to `0..255`, and the low byte is `packed % 256` clamped to `0..255`.
  For example, `unpack_u8_high(-1) == 0`, `unpack_u8_low(-1) == 0`,
  `unpack_u8_high(297) == 1`, and `unpack_u8_low(297) == 41`.
- `serialization.checksum_u8(values)` iterates over the `[]u8` values and
  returns their integer sum. It does not clamp, wrap, hash, authenticate, or
  encrypt the result; an empty slice returns `0`, and `[20, 22]` returns `42`.

Concrete example:

```tetra
import lib.core.serialization as serialization

func serialization_boundary_example() -> Int
uses alloc, mem:
    let packed: Int = serialization.pack_u8_pair(-1, 300)
    let neg_hi: Int = serialization.unpack_u8_high(-1)
    let neg_lo: Int = serialization.unpack_u8_low(-1)
    var payload: []u8 = core.make_u8(2)
    payload[0] = 20
    payload[1] = 22
    let checksum: Int = serialization.checksum_u8(payload)
    if packed == 255 && neg_hi == 0 && neg_lo == 0 && checksum == 42:
        return 42
    return 1
```

## Networking Endpoint Policy Contract

`lib.core.networking` is a stable endpoint policy-helper surface. Its helpers
are pure integer routines for deterministic port and retry choices in examples,
smoke tests, docs, and configuration defaults. They do not open connections,
reserve ports, inspect host networking state, perform name resolution, or grant
network permission.

Safe intended uses are limited to endpoint-shape checks in tests, examples, and
configuration defaults:

- `networking.default_port_http()` returns `80`.
- `networking.default_port_https()` returns `443`.
- `networking.clamp_port(port)` clamps an integer to the inclusive port range
  `0..65535`. Negative values return `0`; values above `65535` return `65535`.
- `networking.is_valid_port(port)` returns true for every integer in
  `0..65535`, including `0`, and false for negative values or values greater
  than `65535`.
- `networking.choose_port(preferred, fallback)` returns `preferred` only when
  it is in `1..65535`. A `preferred` value of `0` is treated as "not selected",
  so the helper returns `clamp_port(fallback)` instead. Invalid preferred
  values, including negatives and values above `65535`, also fall back through
  `clamp_port`.
- `networking.retry_backoff_ms(attempt, base_ms, max_ms)` starts from
  `base_ms` clamped at a minimum of `0`, doubles once for each loop iteration
  while `step < attempt`, and caps the result at `max_ms` when `max_ms` is
  non-negative. A negative `attempt` performs no doubling. A negative `max_ms`
  means no upper cap.

Host transport APIs should layer sockets, DNS, HTTP, and timers behind a
separate capability boundary. This module promises deterministic endpoint
policy helpers only.

### Networking Runtime Boundary

`lib.core.networking` remains endpoint policy only. It is intentionally separate
from the Tetra-source transport/database surface for the TechEmpower-compatible web stack:
the current `lib.core.net` slice provides real linux-x64 TCP socket
open/bind/connect/listen/accept/read/recv/write/send/nonblocking/close helpers,
`SO_REUSEPORT`/`TCP_NODELAY`, plus epoll add/mod/delete and wait-one readiness,
including `epoll_wait_one_into` fd/flags capture and event flag predicates, while full event-loop abstractions, broader socket-option APIs, and
full PostgreSQL access and pooling belong behind future `lib.core.net`
expansion and higher-level `lib.core.postgres` driver layers above the current
startup/simple-query/prepared-frame byte-buffer helpers. Importing
`lib.core.networking` is therefore safe for configuration defaults, but it must
not be used as evidence that the current stdlib can run a production TCP
server, parse full HTTP header maps or request bodies, or talk to a database.
HTTP String and byte-buffer request-line routing, request-head framing, and response byte-buffer helpers are available separately through
`lib.core.http`; JSON response byte-buffer helpers are available separately
through `lib.core.json`.

For `lib.core.net`, loopback bind/connect ports must be in `0..65535`; `0`
remains the ephemeral-port sentinel, while negative and above-range ports return
`-1` before the runtime serializes a TCP port field.

## Crypto Interface Contract

`lib.core.crypto` is a stable interface-helper surface for deterministic
byte-oriented examples, smoke coverage, and API docs. It does not expose keys,
ciphers, signatures, random generation, host entropy, or authenticated
encryption.

Intended uses:

- `crypto.checksum_u8(values)` provides a deterministic checksum for smoke
  assertions and documentation examples.
- `crypto.mix_seed(seed, value)` provides deterministic mixing for repeatable
  examples.
- `crypto.constant_time_eq_u8(lhs, rhs)` compares byte slices and scans
  equal-length inputs without returning early on the first value mismatch.
- `crypto.interface_strength()` returns a stable marker used by examples to
  assert that the interface module is linked and callable.

Security-sensitive products should layer reviewed algorithms and host entropy
behind a separate capability boundary; this module only promises the interface
helpers listed above.

The modules with `capability`, `io`, `mem`, or `mmio` effects do not grant host
permission by import alone. The calling function still declares matching
`uses` effects and obtains any required capability token through the documented
unsafe boundary.

## Testing Status Helpers

Use `import lib.core.testing as testing` for small status-returning checks in
examples, smoke tests, and documentation tests. These helpers use process-style
status conventions: `0` means pass and `1` means fail.

| Helper | Pass status | Fail status |
| --- | --- | --- |
| `testing.assert_true(value)` | `0` when `value` is true | `1` when `value` is false |
| `testing.assert_false(value)` | `0` when `value` is false | `1` when `value` is true |
| `testing.assert_eq_i32(actual, expected)` | `0` when both `i32` values are equal | `1` when they differ |

Use `testing.combine(lhs, rhs)` to merge status values. It returns `lhs` when
`lhs` is non-zero, otherwise it returns `rhs`, so the first failing status is
preserved and later checks run only as far as the caller chooses to evaluate
them.

## Time Duration Helpers

Use `import lib.core.time as time` for small duration arithmetic that stays in
integer milliseconds or seconds and has no effects.

- `time.millis_from_seconds(seconds)` converts seconds to milliseconds by
  multiplying by `1000`. A negative seconds value is treated as a negative
  duration and clamps to `0`; values that would overflow `Int` saturate to
  `2147483647`.
- `time.seconds_from_millis(milliseconds)` converts milliseconds to seconds by
  integer division by `1000`. A negative milliseconds value is treated as a
  negative duration and clamps to `0`.
- `time.clamp_timeout_ms(value, lo, hi)` clamps a millisecond timeout to the
  inclusive `lo`/`hi` range supplied by the caller.
- `time.add_duration_ms(base, delta)` adds the millisecond `delta` to `base`.
  If the result would be negative, it returns `0`; if a positive result would
  overflow `Int`, it returns `2147483647`; otherwise it returns the summed
  millisecond duration. A negative `base` can still recover when a positive
  `delta` makes the sum non-negative.

## Experimental Mirrors

Experimental mirrors are production compatibility shims in the `v0.4.0`
surface: each mirror forwards to the matching `lib.core.*` module so legacy
imports keep compiling. The mirror namespace still has no stability guarantees
for new API growth, so prefer the stable replacement in new code.

| Experimental import | Stable replacement | Status |
| --- | --- | --- |
| `import lib.experimental.async as async` | `import lib.core.async as async` | Experimental mirror; no stability guarantees. |
| `import lib.experimental.collections as collections` | `import lib.core.collections as collections` | Experimental mirror; no stability guarantees. |
| `import lib.experimental.crypto as crypto` | `import lib.core.crypto as crypto` | Experimental mirror; no stability guarantees. |
| `import lib.experimental.filesystem as filesystem` | `import lib.core.filesystem as filesystem` | Experimental mirror; no stability guarantees. |
| `import lib.experimental.io as io` | `import lib.core.io as io` | Experimental mirror; no stability guarantees. |
| `import lib.experimental.math as math` | `import lib.core.math as math` | Experimental mirror; no stability guarantees. |
| `import lib.experimental.memory as memory` | `import lib.core.memory as memory` | Experimental mirror; no stability guarantees. |
| `import lib.experimental.networking as networking` | `import lib.core.networking as networking` | Experimental mirror; no stability guarantees. |
| `import lib.experimental.serialization as serialization` | `import lib.core.serialization as serialization` | Experimental mirror; no stability guarantees. |
| `import lib.experimental.slices as slices` | `import lib.core.slices as slices` | Experimental mirror; no stability guarantees. |
| `import lib.experimental.strings as strings` | `import lib.core.strings as strings` | Experimental mirror; no stability guarantees. |
| `import lib.experimental.sync as sync` | `import lib.core.sync as sync` | Experimental mirror; no stability guarantees. |
| `import lib.experimental.testing as testing` | `import lib.core.testing as testing` | Experimental mirror; no stability guarantees. |
| `import lib.experimental.time as time` | `import lib.core.time as time` | Experimental mirror; no stability guarantees. |

## Runnable Examples

Compile the default linux-x64 smoke profile without running it:

```sh
./tetra smoke --target linux-x64 --run=false
```

The default linux-x64 smoke profile is intentionally narrower than the stable
stdlib completeness workflow. In the current profile, `core_math_smoke` and
`core_memory_smoke` are active smoke cases; the other stable-core
`examples/core_*_smoke.tetra` files are reported through `excluded_examples`
with `not part of linux-x64 smoke profile`.

Run a focused native example:

```sh
./tetra run --target linux-x64 examples/core_math_smoke.tetra
```

Check all stable module docs and doctests:

```sh
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

## Using A Module

Prefer examples already tracked in `examples/` or documented in
`docs/spec/stdlib.md`. Before release, every stable module must have:

- API docs generated by `./tetra doc`.
- At least one parseable documentation example where appropriate.
- Effects metadata where required by the stdlib spec.
- Coverage in `bash scripts/ci/test-all.sh --full --keep-going` or the current
  release gate.

## Verification

Use this stdlib completeness workflow before changing the release-covered module
set or any `lib/core` or `lib/experimental` API docs:

```sh
mkdir -p reports
./tetra doc \
  lib/core \
  lib/experimental \
  examples/core_accessibility_smoke.tetra \
  examples/core_async_smoke.tetra \
  examples/core_capability_smoke.tetra \
  examples/core_collections_smoke.tetra \
  examples/core_component_smoke.tetra \
  examples/core_widgets_smoke.tetra \
  examples/core_crypto_smoke.tetra \
  examples/core_filesystem_smoke.tetra \
  examples/core_http_smoke.tetra \
  examples/core_io_smoke.tetra \
  examples/core_json_smoke.tetra \
  examples/core_math_smoke.tetra \
  examples/core_memory_smoke.tetra \
  examples/core_net_smoke.tetra \
  examples/core_networking_smoke.tetra \
  examples/core_serialization_smoke.tetra \
  examples/core_slices_smoke.tetra \
  examples/core_strings_smoke.tetra \
  examples/core_sync_smoke.tetra \
  examples/core_testing_smoke.tetra \
  examples/core_time_smoke.tetra \
  > reports/stdlib-api-docs.md
go run ./tools/cmd/validate-api-docs --docs reports/stdlib-api-docs.md
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

This workflow catches missing stable module docs, doctests, effects metadata,
and generated API rendering for `lib.core.*` and `lib.experimental.*` modules
and their release-covered core smoke examples. This is the separate stable
stdlib verification workflow for stable-core smoke evidence that is outside the
default linux-x64 smoke profile. The `verify-docs` command is the completeness
gate; the `./tetra doc lib/core lib/experimental ...` command proves the public
API docs render for the same stdlib surface.

Use this separate examples workflow before changing the example index or smoke
profile:

```sh
mkdir -p reports
./tetra doc examples > reports/examples-docs.md
go run ./tools/cmd/validate-api-docs --docs reports/examples-docs.md
./tetra smoke --list --format=json > reports/smoke-list-linux-x64.json
go run ./tools/cmd/validate-smoke-list --report reports/smoke-list-linux-x64.json --examples-root examples
go run ./tools/cmd/validate-example-index --smoke-list reports/smoke-list-linux-x64.json --index docs/user/examples_index.md
```

The examples workflow validates example-only generated docs, catches missing
example rows through `validate-example-index`, and catches smoke-profile drift
through `./tetra smoke --list --format=json` plus `validate-smoke-list`. Its
smoke-list semantics are intentionally binary for each example: active cases
belong in `cases`, while examples outside the default linux-x64 profile belong
in `excluded_examples` with an explicit reason.
