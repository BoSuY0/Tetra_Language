package webrt

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"tetra_language/compiler/internal/pgrt"
)

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
	if _, err := conn.Write([]byte("GET /db HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n")); err != nil {
		t.Fatalf("client write: %v", err)
	}
	got := readUntil(t, conn, func(s string) bool {
		return strings.Contains(s, `{"id":7,"randomNumber":77}`)
	})
	for _, want := range []string{"HTTP/1.1 200 OK", "Content-Type: application/json", "Content-Length: 26"} {
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
	if _, err := conn.Write([]byte("GET /queries?queries=2 HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n")); err != nil {
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
	srv, stop := startDBBenchmarkServerWithRandom(t, pool, sequenceIDs(3, 5), sequenceIDs(9001, 9002))
	defer stop()

	conn := dialServer(t, srv.Port())
	defer conn.Close()
	if _, err := conn.Write([]byte("GET /updates?queries=2 HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n")); err != nil {
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

func startDBBenchmarkServerWithRandom(t *testing.T, pool *pgrt.Pool, nextID func() int, nextRandom func() int) (*Server, func()) {
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
	return pgrt.Connect(ctx, client, pgrt.StartupConfig{User: "benchmarkdbuser", Database: "hello_world"})
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
				randomNumber, err := strconv.Atoi(string(values[0]))
				if err != nil {
					return err
				}
				id, err := strconv.Atoi(string(values[1]))
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
			id, err := strconv.Atoi(string(values[0]))
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
			if _, err := conn.Write(pgServerFrame('T', pgRowDescriptionPayload([]pgFakeColumn{{Name: "id", TypeOID: pgrt.Int4OID}, {Name: "randomNumber", TypeOID: pgrt.Int4OID}}))); err != nil {
				return err
			}
			if _, err := conn.Write(pgServerFrame('D', pgDataRowPayload([]string{strconv.Itoa(id), strconv.Itoa(randomNumber)}))); err != nil {
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
			if _, err := conn.Write(pgServerFrame('T', pgRowDescriptionPayload([]pgFakeColumn{{Name: "id", TypeOID: pgrt.Int4OID}, {Name: "randomNumber", TypeOID: pgrt.Int4OID}}))); err != nil {
				return err
			}
			if _, err := conn.Write(pgServerFrame('D', pgDataRowPayload([]string{strconv.Itoa(id), strconv.Itoa(randomNumber)}))); err != nil {
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
