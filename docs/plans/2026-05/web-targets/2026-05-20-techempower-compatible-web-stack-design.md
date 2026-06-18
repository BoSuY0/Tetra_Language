# TechEmpower-Compatible Web Stack Design

Status: active-goal design for staged implementation. This document preserves
the full target scope; early milestones are not a reduced definition of done.

## Goal

Build a real Tetra web stack that can run TechEmpower-compatible endpoints:
Linux TCP networking, nonblocking event loop, HTTP/1.1 routing, JSON,
PostgreSQL, connection pooling, Fortunes rendering, benchmark configuration,
stress/performance evidence, and honest limitation reporting.

The final state is a production runtime/library stack, not a benchmark-only
simulation. Fast paths may be specialized, but they must still obey protocol
correctness.

## Observed Facts

- Current stable `lib.core.networking` contains endpoint policy helpers such as
  HTTP/HTTPS default ports, port clamping, validity checks, and retry backoff.
  It does not expose sockets, TCP streams, epoll, HTTP serving, JSON encoding,
  or database access.
- Current Go-side tooling uses JSON heavily for diagnostics, reports, manifests,
  and validators, but there is no Tetra `lib.core.json` serializer for user
  programs.
- `cli/internal/actornet` contains a real loopback TCP broker for distributed
  actor evidence. That code is useful prior art for TCP tests and reports, but
  it is not a general-purpose Tetra networking runtime.
- Current examples include `examples/benchmarks/techempower_plaintext_kernel.tetra`,
  which is explicitly a non-network plaintext proxy. It is not a TechEmpower
  HTTP server.
- The current target registry has a runnable `linux-x64` native target in this
  workspace. The initial production web stack is therefore scoped to
  `linux-x64`.
- The worktree already contains unrelated modifications. Implementation must
  avoid reverting or normalizing unrelated files.

## External Benchmark Boundary

The historical TechEmpower FrameworkBenchmarks repository is archived as of
2026-03-24. Local work should target TechEmpower-compatible behavior and config
shape while reporting whether an official upstream submission path exists for
the active benchmark runner.

Required endpoint families:

- `/plaintext`
- `/json`
- `/db`
- `/queries`
- `/updates`
- `/fortunes`

## Production Boundary

### Supported First

- `linux-x64`
- HTTP/1.1 over TCP
- loopback and host-local benchmark execution
- PostgreSQL as the first database backend
- deterministic benchmark and validation reports

### Explicit Non-Goals Until Later Evidence

- TLS
- HTTP/2 or HTTP/3
- Windows/macOS production web runtime
- MySQL unless PostgreSQL is complete
- ORM abstraction
- distributed public-internet deployment guarantees
- claiming official TechEmpower rank without an accepted upstream or compatible
  runner result

## Layered Architecture

### 1. Linux Networking Runtime

Add a low-level runtime surface that can open, configure, and drive TCP sockets:

- `socket`
- `setsockopt`
- `bind`
- `listen`
- `accept4`
- `fcntl`/nonblocking mode
- `read`/`recv`
- `write`/`send`
- `close`
- `epoll_create1`
- `epoll_ctl`
- `epoll_wait`

The runtime must provide stable error codes and must not expose raw kernel
undefined behavior to safe Tetra code. Unsafe boundaries stay explicit.

### 2. Event Loop

Provide a Linux event loop that can:

- register listener and client file descriptors;
- accept bursts of clients;
- handle partial reads and partial writes;
- maintain per-connection input/output buffers;
- enforce connection close behavior;
- support keep-alive and pipelined requests;
- scale to per-core workers in a later tuning slice.

The first implementation may use epoll. `io_uring` remains an optimization
track after epoll correctness and stress evidence exist.

### 3. HTTP/1.1 Server Library

Add a Tetra-facing HTTP server API backed by the runtime. The HTTP layer owns:

- request-line parsing;
- header parsing with limits;
- malformed request rejection;
- routing;
- response building;
- `Date` and `Server` headers;
- keep-alive rules;
- HTTP pipelining;
- plaintext fast path.

The plaintext fast path may prebuild bytes, but it still has to run inside a
real HTTP server that accepts and parses requests.

### 4. JSON Library

Add `lib.core.json` as a library, not a language keyword. It should expose:

- string escaping;
- object/field writer helpers;
- integer serialization;
- byte-buffer writer integration;
- stable errors for invalid or unsupported values.

The TechEmpower `/json` endpoint can use a small optimized object writer for
`{"message":"Hello, World!"}`, but the implementation must still be a JSON
serializer path.

### 5. PostgreSQL Driver And Pool

Add a PostgreSQL client layer that supports:

- startup/auth handshake required by the local benchmark database;
- query and prepared statement protocol paths;
- connection pool checkout/checkin;
- reconnect or fail-fast behavior on broken connections;
- `/db`, `/queries`, and `/updates` endpoint contracts.

The first production DB target is PostgreSQL. MySQL can be added later if the
benchmark target changes.

### 6. Fortunes Rendering

Add the Fortunes path:

- fetch rows from PostgreSQL;
- append the extra fortune;
- sort by message;
- HTML escape;
- render deterministic HTML.

This layer must not reuse JSON shortcuts. It exercises collections, sorting,
HTML escaping, and template output.

### 7. TechEmpower App And Harness

Add a benchmark app under examples or a benchmark-specific tree. It must provide:

- all endpoint families;
- container/config files required by the compatible runner;
- local run scripts;
- correctness verification;
- stress and benchmark reports.

The app should depend on the public Tetra runtime/library stack rather than
private Go helpers, except for test harnesses and validators.

## Proposed Public Modules

Names may change during implementation if existing naming rules require it, but
the capability split should stay:

- `lib.core.net`
- `lib.core.http`
- `lib.core.json`
- `lib.core.postgres`
- `lib.core.html`

`lib.core.networking` can remain the policy-helper module. The new `net` module
is the TCP/runtime module.

## Testing Strategy

### Unit Tests

- syscall wrapper shape and error mapping;
- event-loop connection state transitions;
- HTTP parser and writer;
- router matching;
- JSON escaping/encoding;
- PostgreSQL frame encode/decode;
- pool state machine;
- HTML escaping and fortunes sort/render.

### Edge Case Tests

- partial request line;
- partial headers;
- pipelined requests split across reads;
- multiple requests in one read;
- malformed method/path/version;
- header over limit;
- closed client before response;
- slow client;
- write backpressure;
- invalid JSON strings;
- DB disconnect;
- pool exhaustion;
- update failure.

### Integration Tests

- real local TCP listener;
- keep-alive round trips;
- pipelined `/plaintext`;
- `/json` response correctness;
- PostgreSQL-backed `/db`, `/queries`, `/updates`;
- `/fortunes` HTML output.

### Stress And Performance

- many concurrent connections;
- keep-alive request loops;
- pipelining bursts;
- DB pool pressure;
- long-running soak smoke;
- benchmark report with host, command, git head, endpoint set, throughput,
  latency summary, and limitations.

### Fuzz/Property Tests

- HTTP request parser;
- JSON string escaping;
- PostgreSQL frame parser;
- HTML escaping.

## Milestones

1. **Design and execution plan**
   - this document;
   - implementation plan;
   - docs verification.

2. **Linux TCP runtime skeleton**
   - tested syscall wrappers and local TCP smoke;
   - no HTTP yet.

3. **Event loop and HTTP plaintext**
   - real `/plaintext` server;
   - keep-alive and pipelining;
   - stress smoke.

4. **JSON endpoint**
   - `lib.core.json`;
   - `/json`;
   - JSON fuzz/edge tests.

5. **PostgreSQL DB endpoints**
   - driver;
   - pool;
   - `/db`, `/queries`, `/updates`.

6. **Fortunes**
   - HTML escape, sort, template;
   - `/fortunes`.

7. **TechEmpower-compatible packaging**
   - config, Docker/setup/run scripts;
   - compatible local run evidence.

8. **Release and performance closure**
   - docs/spec/user updates;
   - benchmark reports;
   - stress evidence;
   - release validation.

## Acceptance

The goal is complete only when all requested endpoint families are implemented
through the real Tetra stack, all new and relevant existing tests pass, stress
and performance reports exist, docs are updated, Graphify is current, and any
external benchmark archival or submission limits are explicitly reported.
