# TechEmpower-Compatible Web Stack Implementation Plan

**Goal:** Implement the full TechEmpower-compatible Tetra web stack described in
`docs/plans/2026-05-20-techempower-compatible-web-stack-design.md`.

**Context:** The active goal requires a production Linux TCP runtime,
nonblocking event loop, HTTP/1.1, routing, JSON, PostgreSQL, connection pooling,
TechEmpower-compatible app/config, stress/fuzz/performance tests, docs, and
honest limitation reporting. Do not redefine completion around an early
milestone.

**Execution:** Use TDD for each task. Keep unrelated dirty worktree changes
intact. Run `graphify update .` after code-file changes.

## Task 1: Guard Current Networking Gap

**Goal:** Add tests/docs that encode the current gap before implementation:
`lib.core.networking` is policy helpers only, while the new stack will provide
real TCP through a new module.

**Files:**

- Inspect `lib/core/networking.tetra`
- Inspect `docs/spec/stdlib.md`
- Add or modify focused docs/spec tests only if needed

**Approach:**

- Document the split between existing `lib.core.networking` and planned
  `lib.core.net`.
- Avoid changing existing helper semantics.

**Verification:**

```sh
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

**Done when:** The repo makes the old-vs-new networking boundary explicit.

## Task 2: Linux TCP Syscall Runtime Package

**Goal:** Add a tested internal runtime package for Linux TCP syscalls.

**Files:**

- Add a focused internal package after confirming the best location near
  `compiler/internal/actorsrt` or a new runtime namespace.
- Add package tests.

**Approach:**

- Implement wrappers for socket, setsockopt, bind, listen, accept4, fcntl,
  read/recv, write/send, close, epoll_create1, epoll_ctl, epoll_wait.
- Use stable error values.
- Gate the implementation to Linux.

**Verification:**

```sh
go test ./compiler/... -run 'Net|TCP|Socket|Epoll' -count=1
```

**Done when:** Tests can create a real local TCP listener and exchange bytes
without HTTP.

## Task 3: Tetra Runtime Bridge For `lib.core.net`

**Goal:** Expose the TCP runtime to Tetra programs through stable builtins and
stdlib wrappers.

**Files:**

- Inspect `compiler/internal/semantics/builtins.go`
- Inspect lowering builtins in `compiler/internal/lower`
- Add `lib/core/net.tetra`
- Add semantic/lowering/runtime tests

**Approach:**

- Add only the smallest safe Tetra-facing API needed by the HTTP server layer.
- Keep raw fd operations behind explicit runtime effects.
- Ensure WASM targets reject unsupported network runtime paths with stable
  diagnostics.

**Verification:**

```sh
go test ./compiler/... -run 'Net|Runtime|WASM.*Network|Unsupported' -count=1
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

**Done when:** A Tetra smoke can bind/listen/accept/read/write on Linux-x64 and
unsupported targets reject the feature cleanly.

## Task 4: HTTP Parser And Writer

**Goal:** Implement HTTP/1.1 parsing and response writing independent of the
server loop.

**Files:**

- Add `lib/core/http.tetra` when Tetra-level code is ready
- Add compiler/runtime helper package if parsing must initially live in runtime
- Add unit and fuzz tests

**Approach:**

- Parse method, path, version, headers, and body metadata with limits.
- Support keep-alive and pipelined request boundaries.
- Write response status line and headers, including Date and Server.

**Verification:**

```sh
go test ./compiler/... -run 'HTTP|Parser|Response|Pipelined|KeepAlive' -count=1
go test ./compiler/... -run FuzzHTTP -fuzz=FuzzHTTP -fuzztime=10s
```

**Done when:** Parser/writer tests cover valid, malformed, partial, oversized,
and pipelined inputs.

## Task 5: Nonblocking Event Loop

**Goal:** Drive listeners and client connections with epoll.

**Files:**

- Modify/add runtime event-loop implementation files from Task 2
- Add integration tests

**Approach:**

- Maintain listener/client fd state.
- Handle read/write readiness, partial reads, partial writes, close, and
  backpressure.
- Support one worker first; add per-core workers after correctness.

**Verification:**

```sh
go test ./compiler/... -run 'EventLoop|PartialRead|PartialWrite|Close|Backpressure' -count=1
```

**Done when:** A local test can issue many concurrent keep-alive requests to a
runtime loop without hangs or leaked connections.

## Task 6: `/plaintext` Server

**Goal:** Implement the first real TechEmpower endpoint through the TCP + HTTP
stack.

**Files:**

- Add a benchmark server example path after confirming project conventions
- Add integration and benchmark tests

**Approach:**

- Route `GET /plaintext`.
- Return exactly the required plaintext body and headers.
- Support keep-alive and pipelining.
- Keep fast paths protocol-correct.

**Verification:**

```sh
go test ./compiler/... -run 'Plaintext|KeepAlive|Pipelined|MalformedHTTP' -count=1
go test ./compiler/tests/runtime -run '^$' -bench 'Benchmark.*Plaintext' -count=1
```

**Done when:** Real local HTTP clients pass correctness, pipelining, and stress
smoke tests.

## Task 7: JSON Serializer And `/json`

**Goal:** Add `lib.core.json` and the TechEmpower `/json` endpoint.

**Files:**

- Add `lib/core/json.tetra`
- Add docs/spec entries
- Add unit/fuzz/integration tests

**Approach:**

- Implement string escaping and object writer helpers.
- Add optimized writer path for the TechEmpower message object.
- Ensure invalid strings and escaping edge cases are tested.

**Verification:**

```sh
go test ./compiler/... -run 'JSON|Json' -count=1
go test ./compiler/... -run FuzzJSON -fuzz=FuzzJSON -fuzztime=10s
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
```

**Done when:** `/json` returns the exact expected payload through the serializer.

## Task 8: PostgreSQL Wire Protocol

**Goal:** Add a PostgreSQL client suitable for TechEmpower DB tests.

**Files:**

- Add `lib/core/postgres.tetra` or a staged runtime-backed module
- Add protocol encode/decode tests
- Add integration tests gated on a local PostgreSQL fixture

**Approach:**

- Implement startup/auth path required by the benchmark environment.
- Implement simple query and prepared statement execution.
- Add stable error behavior for disconnects and malformed frames.

**Verification:**

```sh
go test ./compiler/... -run 'Postgres|PGWire|Database' -count=1
```

**Done when:** A real local PostgreSQL query can be executed from the Tetra
runtime/library path.

## Task 9: Connection Pool

**Goal:** Add a production pool used by DB endpoints.

**Files:**

- Extend PostgreSQL module/runtime package
- Add pool tests

**Approach:**

- Support checkout/checkin, max connections, exhaustion behavior, broken
  connection replacement, and shutdown.

**Verification:**

```sh
go test ./compiler/... -run 'Pool|PoolExhaustion|Reconnect' -count=1
```

**Done when:** Pool stress tests show no deadlocks and deterministic failure
behavior under exhaustion.

## Task 10: `/db`, `/queries`, `/updates`

**Goal:** Implement database endpoint family.

**Files:**

- Extend benchmark server example/app
- Add endpoint integration tests

**Approach:**

- Implement single query.
- Implement multiple queries with query count handling.
- Implement updates with correct row update behavior.
- Serialize responses through `lib.core.json`.

**Verification:**

```sh
go test ./compiler/... -run 'TechEmpower.*DB|Queries|Updates' -count=1
```

**Done when:** Endpoints pass local correctness tests against PostgreSQL.

## Task 11: Fortunes

**Goal:** Implement `/fortunes`.

**Files:**

- Add `lib/core/html.tetra` if needed
- Extend benchmark app
- Add sort/render tests

**Approach:**

- Fetch fortunes from DB.
- Add the extra fortune.
- Sort by message.
- HTML escape.
- Render the expected HTML response.

**Verification:**

```sh
go test ./compiler/... -run 'Fortune|HTML|Template|Escape' -count=1
```

**Done when:** `/fortunes` returns valid escaped HTML in deterministic order.

## Task 12: TechEmpower-Compatible Packaging

**Goal:** Add benchmark runner integration.

**Files:**

- Add benchmark app config files after confirming the final tree location
- Add Docker/setup/run scripts where appropriate
- Add report validator if needed

**Approach:**

- Mirror TechEmpower config shape for the local compatible runner.
- Record archival/submission limits honestly.

**Verification:**

```sh
go test ./tools/... -run 'TechEmpower|BenchmarkConfig|WebStackReport' -count=1
```

**Done when:** Local compatible runner can build and exercise all endpoint
families, or the remaining external blocker is documented with evidence.

## Task 13: Stress, Fuzz, And Performance Reports

**Goal:** Produce stable evidence.

**Files:**

- Add scripts under `scripts/` or `tools/` after confirming conventions
- Add docs/generated or reports artifacts as appropriate
- Add validators if reports become release evidence

**Approach:**

- Run concurrency stress.
- Run pipelining bursts.
- Run DB pool pressure.
- Run fuzz tests for parsers/escapers.
- Record benchmark commands, host, git head, endpoint results, and limitations.

**Verification:**

```sh
go test ./compiler/... ./tools/... -count=1
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
```

**Done when:** Reports exist and validators/tests prove they cover the requested
stress/performance scope.

## Task 14: Release Closure

**Goal:** Close the full active goal only when every requirement has direct
evidence.

**Files:**

- Update specs, user docs, benchmark catalog, examples index if required,
  release docs, and Graphify.

**Approach:**

- Map each requirement from the goal to evidence.
- Run the broad relevant test suite.
- Run `graphify update .` after code changes.
- Do not claim official TechEmpower rank unless accepted runner evidence exists.

**Verification:**

```sh
bash scripts/ci/test.sh
bash scripts/ci/test-all.sh --quick
go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json
git diff --check
graphify update .
```

**Done when:** Every scope and testing requirement from the active goal has
current-state evidence, with no missing endpoint family or unverified claim.
