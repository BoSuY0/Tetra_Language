package httprt

import (
	"errors"
	"strings"
	"testing"
)

func TestParseRequestHandlesPartialAndPipelinedRequests(t *testing.T) {
	limits := Limits{MaxHeaderBytes: 4096, MaxHeaders: 32}
	first := "GET /plaintext HTTP/1.1\r\nHost: localhost\r\nConnection: keep-alive\r\n"
	if _, _, err := ParseRequest([]byte(first), limits); !errors.Is(err, ErrIncomplete) {
		t.Fatalf("partial ParseRequest error = %v, want ErrIncomplete", err)
	}

	raw := []byte(first + "\r\nGET /json HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n")
	req, consumed, err := ParseRequest(raw, limits)
	if err != nil {
		t.Fatalf("ParseRequest first: %v", err)
	}
	if req.Method != "GET" || req.Path != "/plaintext" || req.Version != "HTTP/1.1" {
		t.Fatalf("unexpected first request: %#v", req)
	}
	if !req.KeepAlive {
		t.Fatalf("first request KeepAlive = false, want true")
	}
	if got := req.Header("host"); got != "localhost" {
		t.Fatalf("host header = %q, want localhost", got)
	}

	second, secondConsumed, err := ParseRequest(raw[consumed:], limits)
	if err != nil {
		t.Fatalf("ParseRequest second: %v", err)
	}
	if second.Path != "/json" || second.KeepAlive {
		t.Fatalf("unexpected second request: %#v", second)
	}
	if consumed+secondConsumed != len(raw) {
		t.Fatalf("consumed %d + %d, want %d", consumed, secondConsumed, len(raw))
	}
}

func TestParseRequestHandlesBodyMetadataAndBoundaries(t *testing.T) {
	limits := Limits{MaxHeaderBytes: 4096, MaxHeaders: 32}
	raw := []byte("POST /echo HTTP/1.1\r\nHost: localhost\r\nContent-Length: 4\r\n\r\nping")
	req, consumed, err := ParseRequest(raw, limits)
	if err != nil {
		t.Fatalf("ParseRequest: %v", err)
	}
	if consumed != len(raw) {
		t.Fatalf("consumed = %d, want %d", consumed, len(raw))
	}
	if req.ContentLength != 4 || string(req.Body) != "ping" {
		t.Fatalf("body metadata = len %d body %q, want 4 ping", req.ContentLength, req.Body)
	}

	truncated := raw[:len(raw)-1]
	if _, _, err := ParseRequest(truncated, limits); !errors.Is(err, ErrIncomplete) {
		t.Fatalf("truncated body error = %v, want ErrIncomplete", err)
	}
}

func TestParseRequestRejectsOversizedBodyAndUnsupportedTransferEncoding(t *testing.T) {
	limits := Limits{MaxHeaderBytes: 4096, MaxHeaders: 32, MaxBodyBytes: 4}
	raw := []byte("POST /echo HTTP/1.1\r\nHost: localhost\r\nContent-Length: 5\r\n\r\nhello")
	if _, _, err := ParseRequest(raw, limits); !errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("oversized body error = %v, want ErrBodyTooLarge", err)
	}

	chunked := []byte(
		"POST /echo HTTP/1.1\r\nHost: localhost\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n\r\n",
	)
	if _, _, err := ParseRequest(chunked, limits); !errors.Is(err, ErrUnsupportedTransferEncoding) {
		t.Fatalf("chunked body error = %v, want ErrUnsupportedTransferEncoding", err)
	}
}

func TestParseRequestSeparatesPathAndQuery(t *testing.T) {
	req, _, err := ParseRequest(
		[]byte("GET /queries?queries=20&unused=yes HTTP/1.1\r\nHost: localhost\r\n\r\n"),
		Limits{},
	)
	if err != nil {
		t.Fatalf("ParseRequest: %v", err)
	}
	if req.RequestTarget != "/queries?queries=20&unused=yes" {
		t.Fatalf("RequestTarget = %q", req.RequestTarget)
	}
	if req.Path != "/queries" {
		t.Fatalf("Path = %q, want /queries", req.Path)
	}
	if req.Query != "queries=20&unused=yes" {
		t.Fatalf("Query = %q", req.Query)
	}
	if got := req.QueryValue("queries"); got != "20" {
		t.Fatalf("QueryValue(queries) = %q, want 20", got)
	}
}

func TestParseRequestRejectsMalformedAndOversizedInputs(t *testing.T) {
	limits := Limits{MaxHeaderBytes: 64, MaxHeaders: 4}
	cases := []struct {
		name string
		raw  string
		want error
	}{
		{name: "bad request line", raw: "GET /only-two-parts\r\n\r\n", want: ErrMalformedRequest},
		{name: "unsupported version", raw: "GET / HTTP/2.0\r\n\r\n", want: ErrUnsupportedVersion},
		{name: "bad header", raw: "GET / HTTP/1.1\r\nNoColon\r\n\r\n", want: ErrMalformedHeader},
		{
			name: "bad content length",
			raw:  "POST / HTTP/1.1\r\nContent-Length: nope\r\n\r\n",
			want: ErrMalformedHeader,
		},
		{
			name: "too many headers",
			raw:  "GET / HTTP/1.1\r\nA: 1\r\nB: 2\r\nC: 3\r\nD: 4\r\nE: 5\r\n\r\n",
			want: ErrTooManyHeaders,
		},
		{
			name: "oversized",
			raw:  "GET / HTTP/1.1\r\nLong: " + strings.Repeat("x", 80) + "\r\n\r\n",
			want: ErrHeaderTooLarge,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := ParseRequest([]byte(tc.raw), limits)
			if !errors.Is(err, tc.want) {
				t.Fatalf("ParseRequest error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestAppendResponseWritesDeterministicHeaders(t *testing.T) {
	resp := Response{
		StatusCode:  200,
		ContentType: "text/plain",
		Body:        []byte("Hello, World!"),
		Server:      "Tetra",
		Date:        "Wed, 20 May 2026 12:00:00 GMT",
		KeepAlive:   true,
		Headers: []Header{
			{Name: "X-Bench", Value: "plaintext"},
		},
	}

	got := string(AppendResponse(nil, resp))
	want := "HTTP/1.1 200 OK\r\n" +
		"Server: Tetra\r\n" +
		"Date: Wed, 20 May 2026 12:00:00 GMT\r\n" +
		"Content-Type: text/plain\r\n" +
		"Content-Length: 13\r\n" +
		"Connection: keep-alive\r\n" +
		"X-Bench: plaintext\r\n" +
		"\r\n" +
		"Hello, World!"
	if got != want {
		t.Fatalf("response mismatch:\ngot  %q\nwant %q", got, want)
	}
}

func TestRouterMatchesMethodAndPath(t *testing.T) {
	var router Router
	router.Handle("GET", "/plaintext", func(req Request) Response {
		return Response{StatusCode: 200, Body: []byte("ok")}
	})

	req := Request{Method: "GET", Path: "/plaintext", Version: "HTTP/1.1", KeepAlive: true}
	resp, ok := router.Route(req)
	if !ok {
		t.Fatalf("Route did not match GET /plaintext")
	}
	if resp.StatusCode != 200 || string(resp.Body) != "ok" {
		t.Fatalf("Route response = %#v", resp)
	}
	if _, ok := router.Route(Request{Method: "POST", Path: "/plaintext"}); ok {
		t.Fatalf("Route matched wrong method")
	}
	if _, ok := router.Route(Request{Method: "GET", Path: "/missing"}); ok {
		t.Fatalf("Route matched wrong path")
	}
}

func TestRouterMatchesPathParamsAndMiddleware(t *testing.T) {
	var router Router
	router.Use(func(next Handler) Handler {
		return func(req Request) Response {
			resp := next(req)
			resp.Headers = append(
				resp.Headers,
				Header{Name: "X-Route-ID", Value: req.PathValue("id")},
			)
			return resp
		}
	})
	router.Handle("GET", "/users/:id/books/:book", func(req Request) Response {
		return Response{
			StatusCode: 200,
			Body:       []byte(req.PathValue("id") + "/" + req.PathValue("book")),
		}
	})
	router.Handle("GET", "/users/me/books/current", func(req Request) Response {
		return Response{StatusCode: 200, Body: []byte("static")}
	})

	resp, ok := router.Route(Request{Method: "GET", Path: "/users/42/books/tetra"})
	if !ok {
		t.Fatalf("Route did not match parameterized path")
	}
	if string(resp.Body) != "42/tetra" {
		t.Fatalf("parameterized response body = %q", resp.Body)
	}
	if len(resp.Headers) != 1 || resp.Headers[0].Value != "42" {
		t.Fatalf("middleware header = %#v, want X-Route-ID 42", resp.Headers)
	}

	static, ok := router.Route(Request{Method: "GET", Path: "/users/me/books/current"})
	if !ok {
		t.Fatalf("Route did not match static path")
	}
	if string(static.Body) != "static" {
		t.Fatalf("static route body = %q, want static", static.Body)
	}
	if _, ok := router.Route(Request{Method: "GET", Path: "/users/42"}); ok {
		t.Fatalf("Route matched incomplete parameterized path")
	}
}

func FuzzHTTPParseRequest(f *testing.F) {
	seeds := []string{
		"GET /plaintext HTTP/1.1\r\nHost: localhost\r\n\r\n",
		"GET /a HTTP/1.1\r\nHost: x\r\nConnection: close\r\n\r\nGET /b HTTP/1.1\r\n\r\n",
		"POST /echo HTTP/1.1\r\nContent-Length: 4\r\n\r\nping",
		"GET / HTTP/2.0\r\n\r\n",
		"GET / HTTP/1.1\r\nBroken\r\n\r\n",
	}
	for _, seed := range seeds {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _, _ = ParseRequest(data, Limits{MaxHeaderBytes: 8192, MaxHeaders: 64})
	})
}
