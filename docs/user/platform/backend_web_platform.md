# Backend Web Platform Guide

Status: local production-capable backend stack for Linux HTTP/PostgreSQL services, validated by the
TechEmpower-compatible benchmark app.

This stack has two layers:

- Tetra-source low-level modules: `lib.core.net`, `lib.core.http`, `lib.core.json`, and
  `lib.core.postgres` expose executable byte-buffer, socket, epoll, HTTP response, JSON, and
  PostgreSQL wire-frame helpers.
- Runtime/server packages: `compiler/internal/netrt`, `compiler/internal/httprt`,
  `compiler/internal/webrt`, `compiler/internal/jsonrt`, and `compiler/internal/pgrt` provide the
  production HTTP server, routing, JSON value handling, PostgreSQL driver, and pool used by the
  benchmark app.

The runnable benchmark server is `compiler/cmd/tetra-techempower`. It exercises real Linux
nonblocking TCP, epoll, HTTP/1.1 keep-alive/pipelining, JSON, PostgreSQL SCRAM-SHA-256, pooling, and
Fortunes HTML rendering.

## HTTP Runtime Surface

`compiler/internal/httprt` supports:

- method, path, query, header, content-length, and body parsing;
- header count and header byte limits;
- request body limits with `413 Payload Too Large`;
- explicit rejection of unsupported transfer encodings such as chunked bodies;
- keep-alive rules for HTTP/1.0 and HTTP/1.1;
- response writer with status, content type, content length, connection, date, server, and custom
  headers;
- static routes;
- path parameter routes such as `/users/:id/books/:book`;
- middleware wrappers registered with `Router.Use`.

Example handler shape:

```go
var router httprt.Router
router.Use(func(next httprt.Handler) httprt.Handler {
	return func(req httprt.Request) httprt.Response {
		resp := next(req)
		resp.Headers = append(resp.Headers, httprt.Header{Name: "X-Route-ID", Value: req.PathValue("id")})
		return resp
	}
})
router.Handle("GET", "/users/:id", func(req httprt.Request) httprt.Response {
	return httprt.Response{
		StatusCode:  200,
		ContentType: "application/json",
		Body:        jsonrt.AppendMessageObject(nil, req.PathValue("id")),
	}
})
```

Static routes are checked before parameterized routes, so exact paths remain predictable even when a
broader `/:param` route exists.

## JSON Runtime Surface

`compiler/internal/jsonrt` keeps the benchmark fast paths and also exposes a generic JSON value
representation:

- `ParseValue([]byte)` decodes objects, arrays, strings, numbers, booleans, and null.
- `AppendValue` writes deterministic JSON. Object members are sorted by key.
- Invalid number literals return `ErrInvalidJSONNumber`.
- Unsupported value kinds return `ErrUnsupportedValue`.
- Strings use the same escaping rules as the benchmark serializers.

The Tetra-source `lib.core.json` module remains the stable byte-buffer helper surface for
caller-owned buffers. It is appropriate for low-level response builders and smoke tests.

## PostgreSQL And Pooling

`compiler/internal/pgrt` supports:

- PostgreSQL startup with cleartext password or SCRAM-SHA-256;
- malformed SCRAM/SASL rejection, nonce mismatch rejection, and server signature verification;
- simple query and extended prepared statement query paths;
- row description and data-row decoding;
- explicit errors for malformed frames, oversized frames, unsupported auth, PostgreSQL error
  responses, and unexpected messages;
- deterministic pool exhaustion through `ErrPoolExhausted`;
- bad-connection invalidation and replacement;
- `Pool.Stats()` for max/open/in-use/idle/closed leak checks.

Always release checked-out connections. Pass the operation error to `Release(err)` so the pool can
drop poisoned connections.

## Tetra-Source Backend Pattern

At the Tetra source level, low-level backend programs use:

```tetra
import lib.core.capability as capability
import lib.core.http as http
import lib.core.net as net

func main() -> Int
uses alloc, capability, io, mem:
    unsafe:
        let io_cap: cap.io = capability.io()
        let server: Int = net.socket_tcp4(io_cap)
        let nb: Int = net.set_nonblocking(server, io_cap)
        let bound: Int = net.bind_tcp4_loopback(server, 8080, io_cap)
        let listening: Int = net.listen(server, 128, io_cap)
        let epfd: Int = net.epoll_create(io_cap)
        let added: Int = net.epoll_ctl_add_read(epfd, server, io_cap)
        return 0
```

See `examples/core/platform/core_net_smoke.tetra`, `examples/core/platform/core_http_smoke.tetra`,
`examples/core/data/core_json_smoke.tetra`, and the Linux HTTP integration tests in
`compiler/compiler_suite_test.go` for executable Tetra-source server patterns.

## Benchmark App

The benchmark app provides:

- `/plaintext`
- `/json`
- `/db`
- `/queries?queries=N`
- `/updates?queries=N`
- `/fortunes`

Semantics are validated beyond HTTP 200: real DB reads/writes, query clamping, update persistence,
Fortune insertion, HTML escaping/sorting, JSON shapes, headers, content types, and latency/report
metadata.

Local benchmark artifacts are in `docs/benchmarks/` and `reports/techempower/`. They are local
evidence only, not official TechEmpower publication results.

## Limitations

- TLS, HTTP/2, and HTTP/3 are not part of this stack.
- Chunked request bodies are rejected with an explicit unsupported-transfer diagnostic.
- SCRAM-SHA-256-PLUS channel binding and SASLprep for non-ASCII credentials are not implemented.
- A high-level Tetra-source web framework DSL is not yet a separate stable module; the production
  server surface currently lives in runtime packages and the low-level Tetra stdlib modules.
