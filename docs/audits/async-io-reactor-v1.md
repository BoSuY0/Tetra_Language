# P18.3 Async I/O Reactor v1 Audit

P18.3 records Async I/O Reactor v1 as a bounded networking foundation. The
current implementation proves Linux epoll v1 and documents every other
platform as an explicit production boundary.

## Evidence

- `compiler/internal/netrt/io_reactor_coverage.go` emits
  `tetra.runtime.io_reactor.v1`.
- Rows cover Linux epoll v1, future io_uring, macOS kqueue, Windows IOCP,
  WASI/web adapters, nonblocking accept/read/write, readiness polling, I/O
  wakeups, timer integration, cancellation, backpressure, reactor report rows,
  HTTP smoke, DB smoke, and stress evidence.
- `compiler/internal/netrt/io_reactor_coverage_test.go` validates the P18.3
  matrix and rejects fake full production web-stack, cross-platform parity,
  io_uring, runtime-behavior-change, platform-promotion, missing stress,
  missing HTTP smoke, and missing non-claim evidence.
- `compiler/internal/netrt/netrt_linux_test.go` proves nonblocking accept,
  readiness polling, read/write/recv/send round trips, and many readiness waits
  plus timeout behavior.

## Runtime Evidence Used

- Linux epoll evidence comes from `compiler/internal/netrt/netrt_linux.go` and
  the Linux netrt tests.
- Compiled `core.net` epoll/readiness evidence comes from
  `compiler/net_runtime_test.go`.
- Local HTTP evidence comes from `compiler/internal/webrt/server.go` and
  `compiler/internal/webrt/server_test.go`.
- Local DB smoke evidence comes from `compiler/internal/webrt/db_test.go` and
  `compiler/internal/pgrt`.
- Report contract evidence comes from `tools/validators/techempower/report.go`.

## Platform Boundary

- Linux: epoll v1 is implemented narrowly.
- Linux io_uring: future work after epoll stability evidence.
- macOS: kqueue is documented but not implemented.
- Windows: IOCP is documented but not implemented.
- WASI/web: event adapters are documented but not implemented.

## Non-Claims

- No full production web stack is claimed.
- No cross-platform reactor parity is claimed.
- No io_uring support is claimed.
- No runtime behavior is changed by the P18.3 report.
- No official TechEmpower result, production HTTP stack, or production
  PostgreSQL stack is claimed.
