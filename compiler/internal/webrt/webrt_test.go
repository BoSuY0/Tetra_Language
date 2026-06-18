package webrt

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"tetra_language/compiler/internal/httprt"
	"tetra_language/compiler/internal/pgrt"
	"tetra_language/internal/toon"
)

func TestRegisterTechEmpowerRoutesMountsAllEndpoints(t *testing.T) {
	pool, err := pgrt.NewPool(1, func(ctx context.Context) (*pgrt.Conn, error) {
		return nil, errors.New("db intentionally unavailable")
	})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	var router httprt.Router
	if err := RegisterTechEmpowerRoutes(
		&router,
		TechEmpowerRoutes{Pool: pool, NextID: sequenceIDs(1), NextRandom: sequenceIDs(2)},
	); err != nil {
		t.Fatalf("RegisterTechEmpowerRoutes: %v", err)
	}

	plaintext, ok := router.Route(httprt.Request{Method: "GET", Path: "/plaintext"})
	if !ok {
		t.Fatalf("/plaintext route not mounted")
	}
	if plaintext.StatusCode != 200 || plaintext.ContentType != "text/plain" ||
		string(plaintext.Body) != "Hello, World!" {
		t.Fatalf("/plaintext response = %#v", plaintext)
	}

	jsonResp, ok := router.Route(httprt.Request{Method: "GET", Path: "/json"})
	if !ok {
		t.Fatalf("/json route not mounted")
	}
	if jsonResp.StatusCode != 200 || jsonResp.ContentType != "application/json" {
		t.Fatalf("/json response metadata = %#v", jsonResp)
	}
	var decoded struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(jsonResp.Body, &decoded); err != nil {
		t.Fatalf("json.Unmarshal /json: %v", err)
	}
	if decoded.Message != "Hello, World!" {
		t.Fatalf("/json message = %q", decoded.Message)
	}

	toonResp, ok := router.Route(httprt.Request{
		Method: "GET",
		Path:   "/json",
		Headers: []httprt.Header{
			{Name: "Accept", Value: "text/toon"},
		},
	})
	if !ok {
		t.Fatalf("/json route not mounted for TOON")
	}
	if toonResp.StatusCode != 200 || toonResp.ContentType != "text/toon; charset=utf-8" {
		t.Fatalf("/json TOON response metadata = %#v", toonResp)
	}
	decoded = struct {
		Message string `json:"message"`
	}{}
	decodeTOONPayload(t, toonResp.Body, &decoded)
	if decoded.Message != "Hello, World!" {
		t.Fatalf("/json TOON message = %q", decoded.Message)
	}

	for _, path := range []string{"/db", "/queries", "/updates", "/fortunes"} {
		resp, ok := router.Route(httprt.Request{Method: "GET", Path: path})
		if !ok {
			t.Fatalf("%s route not mounted", path)
		}
		if resp.StatusCode != 500 {
			t.Fatalf("%s synthetic DB failure status = %d, want 500", path, resp.StatusCode)
		}
	}
}

func TestRegisterTechEmpowerRoutesRequiresPool(t *testing.T) {
	var router httprt.Router
	err := RegisterTechEmpowerRoutes(&router, TechEmpowerRoutes{})
	if !errors.Is(err, ErrMissingPostgresPool) {
		t.Fatalf("RegisterTechEmpowerRoutes without pool = %v, want ErrMissingPostgresPool", err)
	}
}

func TestAcceptExplicitTOON(t *testing.T) {
	cases := []struct {
		header string
		want   bool
	}{
		{header: "", want: false},
		{header: "*/*", want: false},
		{header: "application/json", want: false},
		{header: "text/toon", want: true},
		{header: "application/json, text/toon; charset=utf-8", want: true},
		{header: "text/toon;q=0", want: false},
		{header: "Text/Toon; Q=0.5", want: true},
	}
	for _, tc := range cases {
		if got := acceptExplicitTOON(tc.header); got != tc.want {
			t.Fatalf("acceptExplicitTOON(%q) = %v, want %v", tc.header, got, tc.want)
		}
	}
}

func TestNewTechEmpowerServerAppliesConfigAndRoutes(t *testing.T) {
	pool, err := pgrt.NewPool(1, func(ctx context.Context) (*pgrt.Conn, error) {
		return nil, errors.New("db intentionally unavailable")
	})
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	srv, err := NewTechEmpowerServer(TechEmpowerServerConfig{
		Address:    [4]byte{127, 0, 0, 1},
		Port:       8080,
		Backlog:    128,
		ServerName: "Tetra-TechEmpower-Test",
		Pool:       pool,
		NextID:     sequenceIDs(1),
		NextRandom: sequenceIDs(2),
	})
	if err != nil {
		t.Fatalf("NewTechEmpowerServer: %v", err)
	}
	if srv.Address != [4]byte{127, 0, 0, 1} || srv.Port() != 0 || srv.Config.Port != 8080 ||
		srv.Backlog != 128 ||
		srv.ServerName != "Tetra-TechEmpower-Test" {
		t.Fatalf("server config = %#v", srv.Config)
	}
	if _, ok := srv.Router.Route(httprt.Request{Method: "GET", Path: "/plaintext"}); !ok {
		t.Fatalf("NewTechEmpowerServer did not mount /plaintext")
	}
}

func decodeTOONPayload(t *testing.T, raw []byte, out any) {
	t.Helper()
	jsonRaw, err := toon.ConvertTOONToJSON(raw, toon.Options{Strict: true})
	if err != nil {
		t.Fatalf("TOON payload did not decode: %v\n%s", err, raw)
	}
	if err := json.Unmarshal(jsonRaw, out); err != nil {
		t.Fatalf("json.Unmarshal converted TOON: %v\nTOON:\n%s\nJSON:\n%s", err, raw, jsonRaw)
	}
}

func TestHTTPDateCacheRefreshesOncePerSecond(t *testing.T) {
	cache := HTTPDateCache{}
	base := time.Date(2026, time.May, 20, 12, 0, 0, 123*int(time.Millisecond), time.UTC)

	first, firstReport := cache.FormatWithReport(base)
	if first != "Wed, 20 May 2026 12:00:00 GMT" {
		t.Fatalf("first date = %q", first)
	}
	if !firstReport.Refreshed {
		t.Fatalf("first report did not refresh: %#v", firstReport)
	}
	if firstReport.UnixSecond != base.Unix() || firstReport.Value != first {
		t.Fatalf("first report = %#v, want second %d value %q", firstReport, base.Unix(), first)
	}

	same, sameReport := cache.FormatWithReport(base.Add(700 * time.Millisecond))
	if same != first {
		t.Fatalf("same-second date = %q, want cached %q", same, first)
	}
	if sameReport.Refreshed {
		t.Fatalf("same-second report refreshed unexpectedly: %#v", sameReport)
	}
	if cache.RefreshCount() != 1 {
		t.Fatalf("refresh count after same-second reuse = %d, want 1", cache.RefreshCount())
	}

	nextTime := base.Add(time.Second)
	next, nextReport := cache.FormatWithReport(nextTime)
	if next != "Wed, 20 May 2026 12:00:01 GMT" {
		t.Fatalf("next-second date = %q", next)
	}
	if !nextReport.Refreshed {
		t.Fatalf("next-second report did not refresh: %#v", nextReport)
	}
	if cache.RefreshCount() != 2 {
		t.Fatalf("refresh count after next-second refresh = %d, want 2", cache.RefreshCount())
	}
}

func TestHTTPDateCacheFormatsUTC(t *testing.T) {
	cache := HTTPDateCache{}
	zone := time.FixedZone("UTC+2", 2*60*60)
	got, report := cache.FormatWithReport(time.Date(2026, time.May, 20, 15, 30, 1, 0, zone))
	if got != "Wed, 20 May 2026 13:30:01 GMT" {
		t.Fatalf("date = %q", got)
	}
	if report.UnixSecond != time.Date(2026, time.May, 20, 13, 30, 1, 0, time.UTC).Unix() {
		t.Fatalf("report UnixSecond = %d", report.UnixSecond)
	}
}

func TestServerDateUsesPerSecondCacheWhenDateFuncAbsent(t *testing.T) {
	base := time.Date(2026, time.May, 20, 12, 0, 0, 0, time.UTC)
	times := []time.Time{
		base,
		base.Add(500 * time.Millisecond),
		base.Add(time.Second),
	}
	index := 0
	srv := NewServer(Config{
		NowFunc: func() time.Time {
			if index >= len(times) {
				return times[len(times)-1]
			}
			now := times[index]
			index++
			return now
		},
	})

	first := srv.date()
	same := srv.date()
	next := srv.date()

	if first != same {
		t.Fatalf("same-second server date = %q, want cached %q", same, first)
	}
	if next == first {
		t.Fatalf("next-second server date did not refresh: %q", next)
	}
	if srv.dateCache.RefreshCount() != 2 {
		t.Fatalf("server date cache refresh count = %d, want 2", srv.dateCache.RefreshCount())
	}
}

func TestServerDateFuncOverrideBypassesCache(t *testing.T) {
	nowCalled := false
	srv := NewServer(Config{
		DateFunc: func() string {
			return "Wed, 20 May 2026 12:00:00 GMT"
		},
		NowFunc: func() time.Time {
			nowCalled = true
			return time.Date(2026, time.May, 20, 12, 0, 0, 0, time.UTC)
		},
	})

	if got := srv.date(); got != "Wed, 20 May 2026 12:00:00 GMT" {
		t.Fatalf("date override = %q", got)
	}
	if nowCalled {
		t.Fatalf("NowFunc was called despite DateFunc override")
	}
	if srv.dateCache.RefreshCount() != 0 {
		t.Fatalf("DateFunc override touched cache refresh count %d", srv.dateCache.RefreshCount())
	}
}

func TestServerDBEndpointUsesPoolAndSerializesWorld(t *testing.T) {
	db := newFakeWorldDB(t, map[int]int{7: 77})
	defer db.Close()
	pool, err := pgrt.NewPool(2, db.Connect)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()
	srv, stop := startDBBenchmarkServer(t, pool, sequenceIDs(7))
	defer stop()

	conn := dialServer(t, srv.Port())
	defer conn.Close()
	if _, err := conn.Write(
		[]byte("GET /db HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"),
	); err != nil {
		t.Fatalf("client write: %v", err)
	}
	got := readUntil(t, conn, func(s string) bool {
		return strings.Contains(s, `{"id":7,"randomNumber":77}`)
	})
	for _, want := range []string{
		"HTTP/1.1 200 OK",
		"Content-Type: application/json",
		"Content-Length: 26",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("/db response missing %q:\n%s", want, got)
		}
	}
	var decoded struct {
		ID           int `json:"id"`
		RandomNumber int `json:"randomNumber"`
	}
	if err := json.Unmarshal([]byte(responseBody(t, got)), &decoded); err != nil {
		t.Fatalf("json.Unmarshal /db: %v\n%s", err, got)
	}
	if decoded.ID != 7 || decoded.RandomNumber != 77 {
		t.Fatalf("decoded /db = %#v", decoded)
	}
	if db.Queries() != 1 {
		t.Fatalf("fake DB query count = %d, want 1", db.Queries())
	}
	if db.PreparedStatements() == 0 {
		t.Fatalf("/db used simple query path; want prepared statement execution")
	}
}

func TestServerDBEndpointsSupportExplicitTOONAccept(t *testing.T) {
	db := newFakeWorldDB(t, map[int]int{3: 30, 5: 50})
	defer db.Close()
	pool, err := pgrt.NewPool(1, db.Connect)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()
	srv, stop := startDBBenchmarkServerWithRandom(
		t,
		pool,
		sequenceIDs(3, 5, 3, 5, 3),
		sequenceIDs(9001, 9002),
	)
	defer stop()

	cases := []struct {
		path string
		want string
	}{
		{path: "/db", want: "id: 3"},
		{path: "/queries?queries=2", want: "5,50"},
		{path: "/updates?queries=2", want: ",9002"},
	}
	for _, tc := range cases {
		conn := dialServer(t, srv.Port())
		rawRequest := "GET " + tc.path + " HTTP/1.1\r\nHost: localhost\r\nAccept: text/toon\r\nConnection: close\r\n\r\n"
		if _, err := conn.Write([]byte(rawRequest)); err != nil {
			conn.Close()
			t.Fatalf("client write %s: %v", tc.path, err)
		}
		got := readUntil(t, conn, func(s string) bool {
			return strings.Contains(s, "Connection: close")
		})
		if err := conn.Close(); err != nil {
			t.Fatalf("client close %s: %v", tc.path, err)
		}
		if !strings.Contains(got, "HTTP/1.1 200 OK") ||
			!strings.Contains(got, "Content-Type: text/toon; charset=utf-8") {
			t.Fatalf("%s TOON response metadata missing:\n%s", tc.path, got)
		}
		body := []byte(responseBody(t, got))
		if !strings.Contains(string(body), tc.want) {
			t.Fatalf("%s TOON body missing %q:\n%s", tc.path, tc.want, body)
		}
		var decoded any
		decodeTOONPayload(t, body, &decoded)
	}
}

func TestServerQueriesEndpointNormalizesCountAndSerializesWorldArray(t *testing.T) {
	db := newFakeWorldDB(t, map[int]int{3: 30, 5: 50})
	defer db.Close()
	pool, err := pgrt.NewPool(1, db.Connect)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()
	srv, stop := startDBBenchmarkServer(t, pool, sequenceIDs(3, 5))
	defer stop()

	conn := dialServer(t, srv.Port())
	defer conn.Close()
	if _, err := conn.Write(
		[]byte("GET /queries?queries=2 HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"),
	); err != nil {
		t.Fatalf("client write: %v", err)
	}
	got := readUntil(t, conn, func(s string) bool {
		return strings.Contains(s, `[{"id":3,"randomNumber":30},{"id":5,"randomNumber":50}]`)
	})
	if !strings.Contains(got, "Content-Type: application/json") {
		t.Fatalf("/queries missing json content type:\n%s", got)
	}
	var decoded []struct {
		ID           int `json:"id"`
		RandomNumber int `json:"randomNumber"`
	}
	if err := json.Unmarshal([]byte(responseBody(t, got)), &decoded); err != nil {
		t.Fatalf("json.Unmarshal /queries: %v\n%s", err, got)
	}
	if len(decoded) != 2 || decoded[0].ID != 3 || decoded[1].RandomNumber != 50 {
		t.Fatalf("decoded /queries = %#v", decoded)
	}
	if db.Queries() != 2 {
		t.Fatalf("fake DB query count = %d, want 2", db.Queries())
	}
}

func TestServerUpdatesEndpointReadsUpdatesThenSerializesWorldArray(t *testing.T) {
	db := newFakeWorldDB(t, map[int]int{3: 30, 5: 50})
	defer db.Close()
	pool, err := pgrt.NewPool(1, db.Connect)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()
	srv, stop := startDBBenchmarkServerWithRandom(
		t,
		pool,
		sequenceIDs(3, 5),
		sequenceIDs(9001, 9002),
	)
	defer stop()

	conn := dialServer(t, srv.Port())
	defer conn.Close()
	if _, err := conn.Write(
		[]byte("GET /updates?queries=2 HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"),
	); err != nil {
		t.Fatalf("client write: %v", err)
	}
	got := readUntil(t, conn, func(s string) bool {
		return strings.Contains(s, `[{"id":3,"randomNumber":9001},{"id":5,"randomNumber":9002}]`)
	})
	if !strings.Contains(got, "Content-Type: application/json") {
		t.Fatalf("/updates missing json content type:\n%s", got)
	}
	var decoded []struct {
		ID           int `json:"id"`
		RandomNumber int `json:"randomNumber"`
	}
	if err := json.Unmarshal([]byte(responseBody(t, got)), &decoded); err != nil {
		t.Fatalf("json.Unmarshal /updates: %v\n%s", err, got)
	}
	if len(decoded) != 2 || decoded[0].RandomNumber != 9001 || decoded[1].RandomNumber != 9002 {
		t.Fatalf("decoded /updates = %#v", decoded)
	}
	if db.Queries() != 4 {
		t.Fatalf("fake DB statement count = %d, want 4", db.Queries())
	}
	if got, want := db.worlds[3], 9001; got != want {
		t.Fatalf("world 3 randomNumber = %d, want %d", got, want)
	}
	if got, want := db.worlds[5], 9002; got != want {
		t.Fatalf("world 5 randomNumber = %d, want %d", got, want)
	}
	if len(db.uniqueUpdates()) != 2 {
		t.Fatalf("updates were not unique: %#v", db.uniqueUpdates())
	}
}

func TestNormalizeQueryCount(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{
		{raw: "", want: 1},
		{raw: "0", want: 1},
		{raw: "-10", want: 1},
		{raw: "abc", want: 1},
		{raw: "20", want: 20},
		{raw: "999", want: 500},
	}
	for _, tc := range cases {
		if got := NormalizeQueryCount(tc.raw); got != tc.want {
			t.Fatalf("NormalizeQueryCount(%q) = %d, want %d", tc.raw, got, tc.want)
		}
	}
}

func startDBBenchmarkServer(t *testing.T, pool *pgrt.Pool, nextID func() int) (*Server, func()) {
	return startDBBenchmarkServerWithRandom(t, pool, nextID, sequenceIDs(1))
}

func startDBBenchmarkServerWithRandom(
	t *testing.T,
	pool *pgrt.Pool,
	nextID func() int,
	nextRandom func() int,
) (*Server, func()) {
	t.Helper()
	srv := NewServer(Config{
		Address:    [4]byte{127, 0, 0, 1},
		Port:       0,
		ServerName: "Tetra-Test",
		DateFunc: func() string {
			return "Wed, 20 May 2026 12:00:00 GMT"
		},
	})
	srv.Router.Handle("GET", "/db", DBHandler(pool, nextID))
	srv.Router.Handle("GET", "/queries", QueriesHandler(pool, nextID))
	srv.Router.Handle("GET", "/updates", UpdatesHandler(pool, nextID, nextRandom))
	if err := srv.Listen(); err != nil {
		t.Fatalf("Listen: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- srv.Serve(ctx)
	}()
	stop := func() {
		cancel()
		if err := srv.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		select {
		case err := <-done:
			if err != nil && err != context.Canceled {
				t.Fatalf("Serve returned %v", err)
			}
		case <-time.After(time.Second):
			t.Fatalf("server did not stop")
		}
	}
	return srv, stop
}

func sequenceIDs(values ...int) func() int {
	i := 0
	return func() int {
		if i >= len(values) {
			return values[len(values)-1]
		}
		value := values[i]
		i++
		return value
	}
}

type fakeWorldDB struct {
	t        *testing.T
	worlds   map[int]int
	queries  chan string
	prepared int
	updates  []int
}

func newFakeWorldDB(t *testing.T, worlds map[int]int) *fakeWorldDB {
	return &fakeWorldDB{t: t, worlds: worlds, queries: make(chan string, 64)}
}

func (db *fakeWorldDB) Connect(ctx context.Context) (*pgrt.Conn, error) {
	client, server := net.Pipe()
	go func() {
		if err := db.serve(server); err != nil && !errors.Is(err, io.ErrClosedPipe) {
			db.t.Errorf("fake world DB serve: %v", err)
		}
	}()
	return pgrt.Connect(
		ctx,
		client,
		pgrt.StartupConfig{User: "benchmarkdbuser", Database: "hello_world"},
	)
}

func (db *fakeWorldDB) Close() {
	close(db.queries)
}

func (db *fakeWorldDB) Queries() int {
	return len(db.queries)
}

func (db *fakeWorldDB) PreparedStatements() int {
	return db.prepared
}

func (db *fakeWorldDB) uniqueUpdates() map[int]bool {
	seen := map[int]bool{}
	for _, value := range db.updates {
		seen[value] = true
	}
	return seen
}

func (db *fakeWorldDB) serve(conn net.Conn) error {
	defer conn.Close()
	if _, err := readStartupMessage(conn); err != nil {
		return err
	}
	if _, err := conn.Write(pgServerFrame('R', pgInt32Payload(0))); err != nil {
		return err
	}
	if _, err := conn.Write(
		pgServerFrame('S', pgCStringsPayload("client_encoding", "UTF8")),
	); err != nil {
		return err
	}
	if _, err := conn.Write(pgServerFrame('K', pgBackendKeyPayload(7, 99))); err != nil {
		return err
	}
	if _, err := conn.Write(pgServerFrame('Z', []byte{'I'})); err != nil {
		return err
	}
	preparedStatements := map[string]string{}
	for {
		frame, err := pgrt.ReadFrame(conn, 1<<20)
		if err != nil {
			return err
		}
		switch frame.Type {
		case 'P':
			name, query, _, err := pgParseClientParse(frame.Payload)
			if err != nil {
				return err
			}
			preparedStatements[name] = query
			db.prepared++
			syncFrame, err := pgrt.ReadFrame(conn, 1<<20)
			if err != nil {
				return err
			}
			if syncFrame.Type != 'S' {
				return errors.New("expected Sync after Parse")
			}
			if _, err := conn.Write(pgServerFrame('1', nil)); err != nil {
				return err
			}
			if _, err := conn.Write(pgServerFrame('Z', []byte{'I'})); err != nil {
				return err
			}
		case 'B':
			_, statement, values, err := pgParseClientBind(frame.Payload)
			if err != nil {
				return err
			}
			describeFrame, err := pgrt.ReadFrame(conn, 1<<20)
			if err != nil {
				return err
			}
			executeFrame, err := pgrt.ReadFrame(conn, 1<<20)
			if err != nil {
				return err
			}
			syncFrame, err := pgrt.ReadFrame(conn, 1<<20)
			if err != nil {
				return err
			}
			if describeFrame.Type != 'D' || executeFrame.Type != 'E' || syncFrame.Type != 'S' {
				return errors.New("expected Describe/Execute/Sync after Bind")
			}
			query := preparedStatements[statement]
			db.queries <- query
			if statement == "update_world_random" {
				if len(values) != 2 {
					return errors.New("update_world_random expects two params")
				}
				randomNumber, err := decodeFakeInt4Param(values[0])
				if err != nil {
					return err
				}
				id, err := decodeFakeInt4Param(values[1])
				if err != nil {
					return err
				}
				db.worlds[id] = randomNumber
				db.updates = append(db.updates, randomNumber)
				if _, err := conn.Write(pgServerFrame('2', nil)); err != nil {
					return err
				}
				if _, err := conn.Write(pgServerFrame('n', nil)); err != nil {
					return err
				}
				if _, err := conn.Write(pgServerFrame('C', pgCStringPayload("UPDATE 1"))); err != nil {
					return err
				}
				if _, err := conn.Write(pgServerFrame('Z', []byte{'I'})); err != nil {
					return err
				}
				continue
			}
			if statement != "world_by_id" {
				return errors.New("unexpected prepared statement")
			}
			if len(values) != 1 {
				return errors.New("world_by_id expects one param")
			}
			id, err := decodeFakeInt4Param(values[0])
			if err != nil {
				return err
			}
			randomNumber, ok := db.worlds[id]
			if !ok {
				randomNumber = id
			}
			if _, err := conn.Write(pgServerFrame('2', nil)); err != nil {
				return err
			}
			if _, err := conn.Write(
				pgServerFrame('T', pgRowDescriptionPayload([]pgFakeColumn{
					{Name: "id", TypeOID: pgrt.Int4OID},
					{Name: "randomNumber", TypeOID: pgrt.Int4OID},
				})),
			); err != nil {
				return err
			}
			if _, err := conn.Write(
				pgServerFrame('D', pgDataRowPayload([]string{strconv.Itoa(id), strconv.Itoa(randomNumber)})),
			); err != nil {
				return err
			}
			if _, err := conn.Write(pgServerFrame('C', pgCStringPayload("SELECT 1"))); err != nil {
				return err
			}
			if _, err := conn.Write(pgServerFrame('Z', []byte{'I'})); err != nil {
				return err
			}
		case 'Q':
			query := string(bytes.TrimSuffix(frame.Payload, []byte{0}))
			db.queries <- query
			if isUpdateQuery(query) {
				id, randomNumber := parseWorldUpdate(query)
				db.worlds[id] = randomNumber
				db.updates = append(db.updates, randomNumber)
				if _, err := conn.Write(pgServerFrame('C', pgCStringPayload("UPDATE 1"))); err != nil {
					return err
				}
				if _, err := conn.Write(pgServerFrame('Z', []byte{'I'})); err != nil {
					return err
				}
				continue
			}
			id := queryWorldID(query)
			randomNumber, ok := db.worlds[id]
			if !ok {
				randomNumber = id
			}
			if _, err := conn.Write(
				pgServerFrame('T', pgRowDescriptionPayload([]pgFakeColumn{
					{Name: "id", TypeOID: pgrt.Int4OID},
					{Name: "randomNumber", TypeOID: pgrt.Int4OID},
				})),
			); err != nil {
				return err
			}
			if _, err := conn.Write(
				pgServerFrame('D', pgDataRowPayload([]string{strconv.Itoa(id), strconv.Itoa(randomNumber)})),
			); err != nil {
				return err
			}
			if _, err := conn.Write(pgServerFrame('C', pgCStringPayload("SELECT 1"))); err != nil {
				return err
			}
			if _, err := conn.Write(pgServerFrame('Z', []byte{'I'})); err != nil {
				return err
			}
		case 'X':
			return nil
		default:
			return errors.New("unexpected fake DB frame type")
		}
	}
}

func isUpdateQuery(query string) bool {
	return strings.HasPrefix(query, "UPDATE World SET randomNumber=")
}

func parseWorldUpdate(query string) (int, int) {
	prefix := "UPDATE World SET randomNumber="
	rest := strings.TrimPrefix(query, prefix)
	parts := strings.Split(rest, " WHERE id=")
	if len(parts) != 2 {
		return 1, 1
	}
	randomNumber, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		randomNumber = 1
	}
	id, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		id = 1
	}
	return id, randomNumber
}

func queryWorldID(query string) int {
	idx := strings.LastIndex(query, "=")
	if idx < 0 {
		return 1
	}
	id, err := strconv.Atoi(strings.TrimSpace(query[idx+1:]))
	if err != nil || id < 1 {
		return 1
	}
	return id
}

func decodeFakeInt4Param(value []byte) (int, error) {
	if len(value) == 4 {
		return pgrt.DecodeInt4(value, pgrt.BinaryFormat)
	}
	return pgrt.DecodeInt4(value, pgrt.TextFormat)
}

func readStartupMessage(r io.Reader) ([]byte, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, err
	}
	n := int(binary.BigEndian.Uint32(lenBuf[:]))
	if n < 8 {
		return nil, errors.New("malformed startup message")
	}
	payload := make([]byte, n-4)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

type pgFakeColumn struct {
	Name    string
	TypeOID uint32
}

func pgServerFrame(typ byte, payload []byte) []byte {
	dst := []byte{typ}
	dst = pgAppendInt32(dst, int32(len(payload)+4))
	dst = append(dst, payload...)
	return dst
}

func pgInt32Payload(value int32) []byte {
	return pgAppendInt32(nil, value)
}

func pgBackendKeyPayload(pid int32, secret int32) []byte {
	var dst []byte
	dst = pgAppendInt32(dst, pid)
	dst = pgAppendInt32(dst, secret)
	return dst
}

func pgCStringsPayload(values ...string) []byte {
	var dst []byte
	for _, value := range values {
		dst = pgAppendCString(dst, value)
	}
	return dst
}

func pgCStringPayload(value string) []byte {
	return pgAppendCString(nil, value)
}

func pgRowDescriptionPayload(cols []pgFakeColumn) []byte {
	var dst []byte
	dst = pgAppendInt16(dst, int16(len(cols)))
	for _, col := range cols {
		dst = pgAppendCString(dst, col.Name)
		dst = pgAppendInt32(dst, 0)
		dst = pgAppendInt16(dst, 0)
		dst = pgAppendInt32(dst, int32(col.TypeOID))
		dst = pgAppendInt16(dst, -1)
		dst = pgAppendInt32(dst, -1)
		dst = pgAppendInt16(dst, 0)
	}
	return dst
}

func pgDataRowPayload(values []string) []byte {
	var dst []byte
	dst = pgAppendInt16(dst, int16(len(values)))
	for _, value := range values {
		dst = pgAppendInt32(dst, int32(len(value)))
		dst = append(dst, value...)
	}
	return dst
}

func pgAppendCString(dst []byte, value string) []byte {
	dst = append(dst, value...)
	dst = append(dst, 0)
	return dst
}

func pgAppendInt16(dst []byte, value int16) []byte {
	var b [2]byte
	binary.BigEndian.PutUint16(b[:], uint16(value))
	return append(dst, b[:]...)
}

func pgAppendInt32(dst []byte, value int32) []byte {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], uint32(value))
	return append(dst, b[:]...)
}

func pgParseClientParse(payload []byte) (string, string, []uint32, error) {
	r := pgPayloadReader{data: payload}
	statement, ok := r.cstring()
	if !ok {
		return "", "", nil, errors.New("malformed parse statement")
	}
	query, ok := r.cstring()
	if !ok {
		return "", "", nil, errors.New("malformed parse query")
	}
	count, ok := r.int16()
	if !ok || count < 0 {
		return "", "", nil, errors.New("malformed parse parameter count")
	}
	oids := make([]uint32, 0, count)
	for i := 0; i < int(count); i++ {
		oid, ok := r.uint32()
		if !ok {
			return "", "", nil, errors.New("malformed parse parameter oid")
		}
		oids = append(oids, oid)
	}
	if !r.done() {
		return "", "", nil, errors.New("trailing parse bytes")
	}
	return statement, query, oids, nil
}

func pgParseClientBind(payload []byte) (string, string, [][]byte, error) {
	r := pgPayloadReader{data: payload}
	portal, ok := r.cstring()
	if !ok {
		return "", "", nil, errors.New("malformed bind portal")
	}
	statement, ok := r.cstring()
	if !ok {
		return "", "", nil, errors.New("malformed bind statement")
	}
	formatCount, ok := r.int16()
	if !ok || formatCount < 0 {
		return "", "", nil, errors.New("malformed bind format count")
	}
	for i := 0; i < int(formatCount); i++ {
		if _, ok := r.int16(); !ok {
			return "", "", nil, errors.New("malformed bind format")
		}
	}
	valueCount, ok := r.int16()
	if !ok || valueCount < 0 {
		return "", "", nil, errors.New("malformed bind value count")
	}
	values := make([][]byte, 0, valueCount)
	for i := 0; i < int(valueCount); i++ {
		n, ok := r.int32()
		if !ok {
			return "", "", nil, errors.New("malformed bind value length")
		}
		if n == -1 {
			values = append(values, nil)
			continue
		}
		if n < 0 || len(r.data)-r.off < int(n) {
			return "", "", nil, errors.New("malformed bind value")
		}
		values = append(values, append([]byte(nil), r.data[r.off:r.off+int(n)]...))
		r.off += int(n)
	}
	resultFormatCount, ok := r.int16()
	if !ok || resultFormatCount < 0 {
		return "", "", nil, errors.New("malformed bind result format count")
	}
	for i := 0; i < int(resultFormatCount); i++ {
		if _, ok := r.int16(); !ok {
			return "", "", nil, errors.New("malformed bind result format")
		}
	}
	if !r.done() {
		return "", "", nil, errors.New("trailing bind bytes")
	}
	return portal, statement, values, nil
}

type pgPayloadReader struct {
	data []byte
	off  int
}

func (r *pgPayloadReader) done() bool {
	return r.off == len(r.data)
}

func (r *pgPayloadReader) cstring() (string, bool) {
	for i := r.off; i < len(r.data); i++ {
		if r.data[i] == 0 {
			value := string(r.data[r.off:i])
			r.off = i + 1
			return value, true
		}
	}
	return "", false
}

func (r *pgPayloadReader) int16() (int16, bool) {
	if len(r.data)-r.off < 2 {
		return 0, false
	}
	value := int16(binary.BigEndian.Uint16(r.data[r.off : r.off+2]))
	r.off += 2
	return value, true
}

func (r *pgPayloadReader) int32() (int32, bool) {
	if len(r.data)-r.off < 4 {
		return 0, false
	}
	value := int32(binary.BigEndian.Uint32(r.data[r.off : r.off+4]))
	r.off += 4
	return value, true
}

func (r *pgPayloadReader) uint32() (uint32, bool) {
	if len(r.data)-r.off < 4 {
		return 0, false
	}
	value := binary.BigEndian.Uint32(r.data[r.off : r.off+4])
	r.off += 4
	return value, true
}

func TestServerFortunesEndpointFetchesSortsAndEscapesHTML(t *testing.T) {
	db := newFakeFortuneDB(t, []fakeFortuneRow{
		{id: 3, message: "Zulu"},
		{
			id:      11,
			message: `<script>alert("This should not be displayed in a browser alert box.");</script>`,
		},
		{id: 8, message: "Alpha & Beta"},
		{id: 12, message: "日本語"},
	})
	defer db.Close()
	pool, err := pgrt.NewPool(1, db.Connect)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()
	srv, stop := startFortunesBenchmarkServer(t, pool)
	defer stop()

	conn := dialServer(t, srv.Port())
	defer conn.Close()
	if _, err := conn.Write(
		[]byte("GET /fortunes HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"),
	); err != nil {
		t.Fatalf("client write: %v", err)
	}
	got := readUntil(t, conn, func(s string) bool {
		return strings.Contains(s, "</table></body></html>")
	})
	for _, want := range []string{
		"HTTP/1.1 200 OK",
		"Server: Tetra-Test",
		"Date: Wed, 20 May 2026 12:00:00 GMT",
		"Content-Type: text/html; charset=utf-8",
		"Connection: close",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("/fortunes response missing %q:\n%s", want, got)
		}
	}

	body := responseBody(t, got)
	if strings.Contains(
		body,
		`<script>alert("This should not be displayed in a browser alert box.");</script>`,
	) {
		t.Fatalf("/fortunes left raw script in HTML:\n%s", body)
	}
	for _, want := range []string{
		("<tr><td>11</td><td>&lt;script&gt;alert(&quot;This should not be " +
			"displayed in a browser alert box.&quot;);&lt;/script&gt;</td></tr>"),
		"<tr><td>0</td><td>Additional fortune added at request time.</td></tr>",
		"<tr><td>8</td><td>Alpha &amp; Beta</td></tr>",
		"<tr><td>12</td><td>日本語</td></tr>",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("/fortunes body missing %q:\n%s", want, body)
		}
	}
	if !(strings.Index(body, "&lt;script&gt;") < strings.Index(body, "Additional fortune") &&
		strings.Index(body, "Additional fortune") < strings.Index(body, "Alpha &amp; Beta") &&
		strings.Index(body, "Alpha &amp; Beta") < strings.Index(body, "Zulu") &&
		strings.Index(body, "Zulu") < strings.Index(body, "日本語")) {
		t.Fatalf("/fortunes body is not sorted by message:\n%s", body)
	}
	queries := db.QueryList()
	if len(queries) != 1 {
		t.Fatalf("fake DB query count = %d, want 1: %#v", len(queries), queries)
	}
	if strings.Contains(strings.ToUpper(queries[0]), "ORDER BY") {
		t.Fatalf("/fortunes query must not use ORDER BY: %q", queries[0])
	}
}

func startFortunesBenchmarkServer(t *testing.T, pool *pgrt.Pool) (*Server, func()) {
	t.Helper()
	srv := NewServer(Config{
		Address:    [4]byte{127, 0, 0, 1},
		Port:       0,
		ServerName: "Tetra-Test",
		DateFunc: func() string {
			return "Wed, 20 May 2026 12:00:00 GMT"
		},
	})
	srv.Router.Handle("GET", "/fortunes", FortunesHandler(pool))
	if err := srv.Listen(); err != nil {
		t.Fatalf("Listen: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- srv.Serve(ctx)
	}()
	stop := func() {
		cancel()
		if err := srv.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		select {
		case err := <-done:
			if err != nil && err != context.Canceled {
				t.Fatalf("Serve returned %v", err)
			}
		case <-time.After(time.Second):
			t.Fatalf("server did not stop")
		}
	}
	return srv, stop
}

type fakeFortuneRow struct {
	id      int
	message string
}

type fakeFortuneDB struct {
	t        *testing.T
	fortunes []fakeFortuneRow
	queries  chan string
}

func newFakeFortuneDB(t *testing.T, fortunes []fakeFortuneRow) *fakeFortuneDB {
	return &fakeFortuneDB{t: t, fortunes: fortunes, queries: make(chan string, 16)}
}

func (db *fakeFortuneDB) Connect(ctx context.Context) (*pgrt.Conn, error) {
	client, server := net.Pipe()
	go func() {
		if err := db.serve(server); err != nil && !errors.Is(err, io.ErrClosedPipe) {
			db.t.Errorf("fake fortune DB serve: %v", err)
		}
	}()
	return pgrt.Connect(
		ctx,
		client,
		pgrt.StartupConfig{User: "benchmarkdbuser", Database: "hello_world"},
	)
}

func (db *fakeFortuneDB) Close() {
	close(db.queries)
}

func (db *fakeFortuneDB) QueryList() []string {
	queries := make([]string, 0, len(db.queries))
	for len(db.queries) > 0 {
		queries = append(queries, <-db.queries)
	}
	return queries
}

func (db *fakeFortuneDB) serve(conn net.Conn) error {
	defer conn.Close()
	if _, err := readStartupMessage(conn); err != nil {
		return err
	}
	if _, err := conn.Write(pgServerFrame('R', pgInt32Payload(0))); err != nil {
		return err
	}
	if _, err := conn.Write(
		pgServerFrame('S', pgCStringsPayload("client_encoding", "UTF8")),
	); err != nil {
		return err
	}
	if _, err := conn.Write(pgServerFrame('K', pgBackendKeyPayload(7, 99))); err != nil {
		return err
	}
	if _, err := conn.Write(pgServerFrame('Z', []byte{'I'})); err != nil {
		return err
	}
	preparedStatements := map[string]string{}
	for {
		frame, err := pgrt.ReadFrame(conn, 1<<20)
		if err != nil {
			return err
		}
		switch frame.Type {
		case 'P':
			name, query, _, err := pgParseClientParse(frame.Payload)
			if err != nil {
				return err
			}
			preparedStatements[name] = query
			syncFrame, err := pgrt.ReadFrame(conn, 1<<20)
			if err != nil {
				return err
			}
			if syncFrame.Type != 'S' {
				return errors.New("expected Sync after Parse")
			}
			if _, err := conn.Write(pgServerFrame('1', nil)); err != nil {
				return err
			}
			if _, err := conn.Write(pgServerFrame('Z', []byte{'I'})); err != nil {
				return err
			}
		case 'B':
			_, statement, _, err := pgParseClientBind(frame.Payload)
			if err != nil {
				return err
			}
			describeFrame, err := pgrt.ReadFrame(conn, 1<<20)
			if err != nil {
				return err
			}
			executeFrame, err := pgrt.ReadFrame(conn, 1<<20)
			if err != nil {
				return err
			}
			syncFrame, err := pgrt.ReadFrame(conn, 1<<20)
			if err != nil {
				return err
			}
			if describeFrame.Type != 'D' || executeFrame.Type != 'E' || syncFrame.Type != 'S' {
				return errors.New("expected Describe/Execute/Sync after Bind")
			}
			query := preparedStatements[statement]
			db.queries <- query
			if _, err := conn.Write(pgServerFrame('2', nil)); err != nil {
				return err
			}
			if err := db.writeFortunesResult(conn); err != nil {
				return err
			}
		case 'Q':
			query := string(bytes.TrimSuffix(frame.Payload, []byte{0}))
			db.queries <- query
			if err := db.writeFortunesResult(conn); err != nil {
				return err
			}
		case 'X':
			return nil
		default:
			return errors.New("unexpected fake DB frame type")
		}
	}
}

func (db *fakeFortuneDB) writeFortunesResult(conn net.Conn) error {
	if _, err := conn.Write(
		pgServerFrame('T', pgRowDescriptionPayload([]pgFakeColumn{
			{Name: "id", TypeOID: pgrt.Int4OID},
			{Name: "message", TypeOID: 25},
		})),
	); err != nil {
		return err
	}
	for _, fortune := range db.fortunes {
		if _, err := conn.Write(
			pgServerFrame('D', pgDataRowPayload([]string{strconv.Itoa(fortune.id), fortune.message})),
		); err != nil {
			return err
		}
	}
	if _, err := conn.Write(
		pgServerFrame('C', pgCStringPayload("SELECT "+strconv.Itoa(len(db.fortunes)))),
	); err != nil {
		return err
	}
	if _, err := conn.Write(pgServerFrame('Z', []byte{'I'})); err != nil {
		return err
	}
	return nil
}

func TestServerPlaintextKeepAliveAndPipelining(t *testing.T) {
	srv, stop := startBenchmarkServer(t)
	defer stop()

	conn := dialServer(t, srv.Port())
	defer conn.Close()
	raw := "GET /plaintext HTTP/1.1\r\nHost: localhost\r\nConnection: keep-alive\r\n\r\n" +
		"GET /plaintext HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"
	if _, err := conn.Write([]byte(raw)); err != nil {
		t.Fatalf("client write: %v", err)
	}

	got := readUntil(t, conn, func(s string) bool {
		return strings.Count(s, "HTTP/1.1 200 OK") == 2 &&
			strings.Count(s, "Hello, World!") == 2
	})
	for _, want := range []string{
		"Server: Tetra-Test",
		"Date: Wed, 20 May 2026 12:00:00 GMT",
		"Content-Type: text/plain",
		"Content-Length: 13",
		"Connection: keep-alive",
		"Connection: close",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("response missing %q:\n%s", want, got)
		}
	}
}

func TestServerJSONEndpointKeepAliveAndPipelining(t *testing.T) {
	srv, stop := startBenchmarkServer(t)
	defer stop()

	conn := dialServer(t, srv.Port())
	defer conn.Close()
	raw := "GET /json HTTP/1.1\r\nHost: localhost\r\nConnection: keep-alive\r\n\r\n" +
		"GET /json HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"
	if _, err := conn.Write([]byte(raw)); err != nil {
		t.Fatalf("client write: %v", err)
	}

	got := readUntil(t, conn, func(s string) bool {
		return strings.Count(s, "HTTP/1.1 200 OK") == 2 &&
			strings.Count(s, `{"message":"Hello, World!"}`) == 2
	})
	for _, want := range []string{
		"Content-Type: application/json",
		"Content-Length: 27",
		"Connection: keep-alive",
		"Connection: close",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("response missing %q:\n%s", want, got)
		}
	}
	body := responseBody(t, got)
	var decoded struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("json.Unmarshal body %q: %v\nfull response:\n%s", body, err, got)
	}
	if decoded.Message != "Hello, World!" {
		t.Fatalf("decoded message = %q", decoded.Message)
	}
}

func TestServerJSONEndpointSupportsExplicitTOONAccept(t *testing.T) {
	srv, stop := startBenchmarkServer(t)
	defer stop()

	conn := dialServer(t, srv.Port())
	defer conn.Close()
	raw := "GET /json HTTP/1.1\r\nHost: localhost\r\nAccept: text/toon\r\nConnection: close\r\n\r\n"
	if _, err := conn.Write([]byte(raw)); err != nil {
		t.Fatalf("client write: %v", err)
	}

	got := readUntil(t, conn, func(s string) bool {
		return strings.Contains(s, "HTTP/1.1 200 OK") &&
			strings.Contains(s, "Content-Type: text/toon; charset=utf-8") &&
			strings.Contains(s, `message: "Hello, World!"`)
	})
	body := []byte(responseBody(t, got))
	var decoded struct {
		Message string `json:"message"`
	}
	decodeTOONPayload(t, body, &decoded)
	if decoded.Message != "Hello, World!" {
		t.Fatalf("decoded TOON message = %q", decoded.Message)
	}
}

func TestServerHandlesPartialRequestRead(t *testing.T) {
	srv, stop := startBenchmarkServer(t)
	defer stop()

	conn := dialServer(t, srv.Port())
	defer conn.Close()
	if _, err := conn.Write([]byte("GET /plain")); err != nil {
		t.Fatalf("partial write 1: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, err := conn.Write(
		[]byte("text HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"),
	); err != nil {
		t.Fatalf("partial write 2: %v", err)
	}

	got := readUntil(t, conn, func(s string) bool {
		return strings.Contains(s, "HTTP/1.1 200 OK") && strings.Contains(s, "Hello, World!")
	})
	if !strings.Contains(got, "Connection: close") {
		t.Fatalf("partial response missing close header:\n%s", got)
	}
}

func TestServerRejectsMalformedRequest(t *testing.T) {
	srv, stop := startBenchmarkServer(t)
	defer stop()

	conn := dialServer(t, srv.Port())
	defer conn.Close()
	if _, err := conn.Write([]byte("GET /missing-version\r\nHost: localhost\r\n\r\n")); err != nil {
		t.Fatalf("client write malformed request: %v", err)
	}

	got := readUntil(t, conn, func(s string) bool {
		return strings.Contains(s, "HTTP/1.1 400 Bad Request")
	})
	if !strings.Contains(got, "Connection: close") {
		t.Fatalf("malformed response missing close header:\n%s", got)
	}
}

func TestServerRejectsRequestBodyOverConfiguredLimit(t *testing.T) {
	srv := NewServer(Config{
		Address:      [4]byte{127, 0, 0, 1},
		Port:         0,
		ServerName:   "Tetra-Test",
		MaxBodyBytes: 4,
		DateFunc: func() string {
			return "Wed, 20 May 2026 12:00:00 GMT"
		},
	})
	srv.Router.Handle("POST", "/echo", func(req httprt.Request) httprt.Response {
		return httprt.Response{StatusCode: 200, ContentType: "text/plain", Body: req.Body}
	})
	if err := srv.Listen(); err != nil {
		t.Fatalf("Listen: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- srv.Serve(ctx)
	}()
	defer func() {
		cancel()
		if err := srv.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		select {
		case err := <-done:
			if err != nil && err != context.Canceled {
				t.Fatalf("Serve returned %v", err)
			}
		case <-time.After(time.Second):
			t.Fatalf("server did not stop")
		}
	}()

	conn := dialServer(t, srv.Port())
	defer conn.Close()
	if _, err := conn.Write(
		[]byte("POST /echo HTTP/1.1\r\nHost: localhost\r\nContent-Length: 5\r\nConnection: close\r\n\r\nhello"),
	); err != nil {
		t.Fatalf("client write oversized body: %v", err)
	}
	got := readUntil(t, conn, func(s string) bool {
		return strings.Contains(s, "HTTP/1.1 413 Payload Too Large")
	})
	if !strings.Contains(got, "Connection: close") {
		t.Fatalf("oversized body response missing close header:\n%s", got)
	}
}

func startBenchmarkServer(t *testing.T) (*Server, func()) {
	t.Helper()
	srv := NewServer(Config{
		Address:    [4]byte{127, 0, 0, 1},
		Port:       0,
		ServerName: "Tetra-Test",
		DateFunc: func() string {
			return "Wed, 20 May 2026 12:00:00 GMT"
		},
	})
	srv.Router.Handle("GET", "/plaintext", func(req httprt.Request) httprt.Response {
		return httprt.Response{
			StatusCode:  200,
			ContentType: "text/plain",
			Body:        []byte("Hello, World!"),
		}
	})
	srv.Router.Handle("GET", "/json", JSONMessageHandler("Hello, World!"))
	if err := srv.Listen(); err != nil {
		t.Fatalf("Listen: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- srv.Serve(ctx)
	}()
	stop := func() {
		cancel()
		if err := srv.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		select {
		case err := <-done:
			if err != nil && err != context.Canceled {
				t.Fatalf("Serve returned %v", err)
			}
		case <-time.After(time.Second):
			t.Fatalf("server did not stop")
		}
	}
	return srv, stop
}

func dialServer(t *testing.T, port int) net.Conn {
	t.Helper()
	conn, err := net.DialTimeout(
		"tcp",
		net.JoinHostPort("127.0.0.1", strconv.Itoa(port)),
		time.Second,
	)
	if err != nil {
		t.Fatalf("DialTimeout: %v", err)
	}
	if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetDeadline: %v", err)
	}
	return conn
}

func readUntil(t *testing.T, conn net.Conn, done func(string) bool) string {
	t.Helper()
	var b strings.Builder
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			b.Write(buf[:n])
			if done(b.String()) {
				return b.String()
			}
		}
		if err != nil {
			t.Fatalf("read response before condition was met: %v\n%s", err, b.String())
		}
	}
}

func responseBody(t *testing.T, raw string) string {
	t.Helper()
	idx := strings.Index(raw, "\r\n\r\n")
	if idx < 0 {
		t.Fatalf("response missing body separator:\n%s", raw)
	}
	end := idx + len("\r\n\r\n")
	next := strings.Index(raw[end:], "HTTP/1.1 ")
	if next >= 0 {
		return raw[end : end+next]
	}
	return raw[end:]
}

func TestServerStressManyConcurrentKeepAliveClients(t *testing.T) {
	srv, stop := startBenchmarkServer(t)
	defer stop()

	const clients = 32
	const requestsPerClient = 16
	var wg sync.WaitGroup
	errc := make(chan error, clients)
	for clientID := 0; clientID < clients; clientID++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			conn := dialServer(t, srv.Port())
			defer conn.Close()
			var raw strings.Builder
			for i := 0; i < requestsPerClient; i++ {
				connection := "keep-alive"
				if i == requestsPerClient-1 {
					connection = "close"
				}
				raw.WriteString("GET /plaintext HTTP/1.1\r\nHost: localhost\r\nConnection: ")
				raw.WriteString(connection)
				raw.WriteString("\r\n\r\n")
			}
			if _, err := io.WriteString(conn, raw.String()); err != nil {
				errc <- fmt.Errorf("client %d write: %w", clientID, err)
				return
			}
			got := readUntil(t, conn, func(s string) bool {
				return strings.Count(s, "HTTP/1.1 200 OK") == requestsPerClient &&
					strings.Count(s, "Hello, World!") == requestsPerClient &&
					strings.Contains(s, "Connection: close")
			})
			if strings.Count(got, "Connection: keep-alive") != requestsPerClient-1 {
				errc <- fmt.Errorf("client %d keep-alive response count mismatch", clientID)
			}
		}(clientID)
	}
	wg.Wait()
	close(errc)
	for err := range errc {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestServerStressPipeliningBurst(t *testing.T) {
	srv, stop := startBenchmarkServer(t)
	defer stop()

	conn := dialServer(t, srv.Port())
	defer conn.Close()
	const requests = 128
	var raw strings.Builder
	for i := 0; i < requests; i++ {
		connection := "keep-alive"
		if i == requests-1 {
			connection = "close"
		}
		raw.WriteString("GET /json HTTP/1.1\r\nHost: localhost\r\nConnection: ")
		raw.WriteString(connection)
		raw.WriteString("\r\n\r\n")
	}
	if _, err := io.WriteString(conn, raw.String()); err != nil {
		t.Fatalf("client write pipelining burst: %v", err)
	}
	got := readUntil(t, conn, func(s string) bool {
		return strings.Count(s, "HTTP/1.1 200 OK") == requests &&
			strings.Count(s, `{"message":"Hello, World!"}`) == requests &&
			strings.Contains(s, "Connection: close")
	})
	if strings.Count(got, "Content-Type: application/json") != requests {
		t.Fatalf("pipelining burst json content-type count mismatch")
	}
}

func TestServerHandlesSlowHeaderDripClient(t *testing.T) {
	srv, stop := startBenchmarkServer(t)
	defer stop()

	conn := dialServer(t, srv.Port())
	defer conn.Close()
	req := "GET /plaintext HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"
	for i := 0; i < len(req); i++ {
		if _, err := conn.Write([]byte{req[i]}); err != nil {
			t.Fatalf("slow client byte %d write: %v", i, err)
		}
		time.Sleep(500 * time.Microsecond)
	}
	got := readUntil(t, conn, func(s string) bool {
		return strings.Contains(s, "HTTP/1.1 200 OK") && strings.Contains(s, "Hello, World!")
	})
	if !strings.Contains(got, "Connection: close") {
		t.Fatalf("slow header drip response missing close:\n%s", got)
	}
}

func TestServerStressClosedClientsDoNotBreakOtherConnections(t *testing.T) {
	srv, stop := startBenchmarkServer(t)
	defer stop()

	const closedClients = 32
	for i := 0; i < closedClients; i++ {
		conn, err := net.DialTimeout(
			"tcp",
			net.JoinHostPort("127.0.0.1", fmt.Sprint(srv.Port())),
			time.Second,
		)
		if err != nil {
			t.Fatalf("closed client dial %d: %v", i, err)
		}
		_, _ = io.WriteString(conn, "GET /plaintext HTTP/1.1\r\nHost: localhost\r\n")
		_ = conn.Close()
	}

	conn := dialServer(t, srv.Port())
	defer conn.Close()
	if _, err := io.WriteString(
		conn,
		"GET /plaintext HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n",
	); err != nil {
		t.Fatalf("healthy client write: %v", err)
	}
	got := readUntil(t, conn, func(s string) bool {
		return strings.Contains(s, "HTTP/1.1 200 OK") && strings.Contains(s, "Hello, World!")
	})
	if !strings.Contains(got, "Connection: close") {
		t.Fatalf("healthy response missing close:\n%s", got)
	}
}

func TestListenWorkersServesSharedPortAcrossMultipleEventLoops(t *testing.T) {
	group, err := ListenWorkers(2, 0, func(workerID int, port int) (*Server, error) {
		srv := NewServer(Config{
			Address:    [4]byte{127, 0, 0, 1},
			Port:       port,
			ServerName: fmt.Sprintf("Tetra-Worker-%d", workerID),
			DateFunc: func() string {
				return "Wed, 20 May 2026 12:00:00 GMT"
			},
		})
		srv.Router.Handle("GET", "/plaintext", func(req httprt.Request) httprt.Response {
			return httprt.Response{
				StatusCode:  200,
				ContentType: "text/plain",
				Body:        []byte(fmt.Sprintf("worker-%d", workerID)),
			}
		})
		return srv, nil
	})
	if err != nil {
		t.Fatalf("ListenWorkers: %v", err)
	}
	defer group.Close()

	if group.Count() != 2 {
		t.Fatalf("worker count = %d, want 2", group.Count())
	}
	if group.Port() == 0 {
		t.Fatalf("shared port was not assigned")
	}
	for _, port := range group.Ports() {
		if port != group.Port() {
			t.Fatalf("worker port = %d, want shared port %d", port, group.Port())
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- group.Serve(ctx)
	}()

	conn := dialServer(t, group.Port())
	defer conn.Close()
	if _, err := conn.Write(
		[]byte("GET /plaintext HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"),
	); err != nil {
		t.Fatalf("client write: %v", err)
	}
	got := readUntil(t, conn, func(s string) bool {
		return strings.Contains(s, "HTTP/1.1 200 OK") && strings.Contains(s, "worker-")
	})
	if !strings.Contains(got, "Connection: close") {
		t.Fatalf("worker response missing close header:\n%s", got)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil && err != context.Canceled {
			t.Fatalf("worker group Serve returned %v", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("worker group did not stop")
	}
}

func TestListenWorkersClosesAlreadyStartedWorkersWhenLaterWorkerFails(t *testing.T) {
	var first *Server
	_, err := ListenWorkers(2, 0, func(workerID int, port int) (*Server, error) {
		if workerID == 1 {
			return nil, fmt.Errorf("worker %d cannot start", workerID)
		}
		first = NewServer(Config{
			Address: [4]byte{127, 0, 0, 1},
			Port:    port,
		})
		return first, nil
	})
	if err == nil || !strings.Contains(err.Error(), "worker 1") {
		t.Fatalf("ListenWorkers failure = %v, want worker 1 error", err)
	}
	if first == nil {
		t.Fatalf("first worker was not constructed")
	}
	if first.Port() != 0 {
		t.Fatalf("first worker port = %d, want closed zero state", first.Port())
	}
}
