package webrt

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"tetra_language/compiler/internal/pgrt"
)

func TestServerFortunesEndpointFetchesSortsAndEscapesHTML(t *testing.T) {
	db := newFakeFortuneDB(t, []fakeFortuneRow{
		{id: 3, message: "Zulu"},
		{id: 11, message: `<script>alert("This should not be displayed in a browser alert box.");</script>`},
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
	if _, err := conn.Write([]byte("GET /fortunes HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n")); err != nil {
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
	if strings.Contains(body, `<script>alert("This should not be displayed in a browser alert box.");</script>`) {
		t.Fatalf("/fortunes left raw script in HTML:\n%s", body)
	}
	for _, want := range []string{
		"<tr><td>11</td><td>&lt;script&gt;alert(&quot;This should not be displayed in a browser alert box.&quot;);&lt;/script&gt;</td></tr>",
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
	return pgrt.Connect(ctx, client, pgrt.StartupConfig{User: "benchmarkdbuser", Database: "hello_world"})
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
	if _, err := conn.Write(pgServerFrame('S', pgCStringsPayload("client_encoding", "UTF8"))); err != nil {
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
	if _, err := conn.Write(pgServerFrame('T', pgRowDescriptionPayload([]pgFakeColumn{{Name: "id", TypeOID: pgrt.Int4OID}, {Name: "message", TypeOID: 25}}))); err != nil {
		return err
	}
	for _, fortune := range db.fortunes {
		if _, err := conn.Write(pgServerFrame('D', pgDataRowPayload([]string{strconv.Itoa(fortune.id), fortune.message}))); err != nil {
			return err
		}
	}
	if _, err := conn.Write(pgServerFrame('C', pgCStringPayload("SELECT "+strconv.Itoa(len(db.fortunes))))); err != nil {
		return err
	}
	if _, err := conn.Write(pgServerFrame('Z', []byte{'I'})); err != nil {
		return err
	}
	return nil
}
