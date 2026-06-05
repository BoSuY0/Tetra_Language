# P18.3 Async I/O Reactor v1 Design

## Goal

Close P18.3 as a bounded, evidence-backed networking runtime boundary:

- Linux has an epoll v1 foundation.
- io_uring remains future work until epoll evidence is stable.
- macOS kqueue, Windows IOCP, and WASI/web event adapters remain explicit
  platform boundaries.
- Existing HTTP and PostgreSQL evidence is used only as local smoke evidence,
  not as a full production web stack claim.

## Observed Evidence

Graphify navigation found the concrete runtime and smoke surfaces before raw
file inspection:

- `compiler/internal/netrt/netrt_linux.go` contains `ListenTCP4`, `Accept`,
  `Read`, `Recv`, `Write`, `Send`, `NewPoller`, `AddRead`, `AddReadWrite`,
  `Mod`, `Remove`, and `Wait`.
- `compiler/internal/netrt/netrt_unsupported.go` returns `ErrUnsupported` on
  non-Linux platforms.
- `compiler/internal/netrt/netrt_linux_test.go` proves nonblocking accept,
  epoll readability, and syscall read/write/recv/send round trips.
- `compiler/net_runtime_test.go` proves compiled `core.net` socket, epoll
  control, epoll readiness, and task-scheduler composition smokes for the
  current Linux native targets where the host kernel can run them.
- `compiler/internal/webrt/server.go` uses the netrt poller, nonblocking
  accept/read/write loops, context cancellation, 50ms timeout polling, and
  EPOLLOUT interest updates when response output is pending.
- `compiler/internal/webrt/server_test.go` and `db_test.go` prove local
  plaintext/json and PostgreSQL-backed handler smokes.
- `compiler/internal/pgrt/pool.go` records bounded pool backpressure through
  `ErrPoolExhausted`.
- `tools/validators/techempower/report.go` validates local TechEmpower-style
  reports, but this remains a report contract, not an official benchmark claim.

## Shape

Add `compiler/internal/netrt/io_reactor_coverage.go` with:

- schema `tetra.runtime.io_reactor.v1`;
- row ids for Linux epoll, future io_uring, macOS kqueue, Windows IOCP,
  WASI/web adapters, nonblocking accept/read/write, readiness polling, I/O
  task wakeups, timer integration, cancellation, backpressure, reactor report
  rows, HTTP smoke, DB smoke, and stress evidence;
- explicit non-claim booleans for full production web stack, cross-platform
  reactor parity, io_uring, and runtime behavior change;
- validator rejection for missing rows, weak/missing evidence, fake platform
  parity, fake io_uring, fake web-stack promotion, fake runtime behavior
  change, missing stress evidence, missing HTTP/DB smoke evidence, and missing
  non-claims.

The report is evidence-only. It does not change `netrt`, `webrt`, scheduler,
or compiler behavior.

## Boundary

P18.3 is complete when the report and validator prove the current production
boundary per platform:

- Linux: epoll v1 is implemented narrowly and backed by low-level plus compiled
  runtime smoke evidence.
- io_uring: blocked/future; not implemented.
- macOS: kqueue boundary is documented, not implemented.
- Windows: IOCP boundary is documented, not implemented.
- WASI/web: adapter boundary is documented, not implemented.

HTTP/JSON and PostgreSQL evidence is local smoke evidence. P19.2 and P19.3 are
still responsible for production HTTP/JSON and database stack promotion.

## Verification Plan

1. Add RED tests for the P18.3 report and fake-claim rejection.
2. Implement the report and validator in `netrt`.
3. Update docs/audit, closure report, feature registry, generated manifest,
   and goal-loop sidecars.
4. Run focused `netrt`, `webrt`, `pgrt`, feature, manifest, and docs gates.
5. Run a broader relevant package gate, `go test -race ./compiler/internal/netrt`
   on Linux, `graphify update .`, `git diff --check`, drift/overclaim scans,
   scratch scan, and Go cache cleanup.
