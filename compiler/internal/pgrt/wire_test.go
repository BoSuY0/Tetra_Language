package pgrt

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAppendStartupMessageDeterministic(t *testing.T) {
	got := AppendStartupMessage(nil, StartupConfig{
		User:     "benchmarkdbuser",
		Database: "hello_world",
		Parameters: map[string]string{
			"application_name": "tetra-techempower",
			"client_encoding":  "UTF8",
		},
	})
	if len(got) < 8 {
		t.Fatalf("startup message too short: %d", len(got))
	}
	if length := int(binary.BigEndian.Uint32(got[:4])); length != len(got) {
		t.Fatalf("startup length = %d, want %d", length, len(got))
	}
	if protocol := binary.BigEndian.Uint32(got[4:8]); protocol != ProtocolVersion30 {
		t.Fatalf("protocol = %#x, want %#x", protocol, uint32(ProtocolVersion30))
	}
	wantSuffix := ("user\x00benchmarkdbuser\x00database\x00hello_world\x00application_" +
		"name\x00tetra-techempower\x00client_encoding\x00UTF8\x00\x00")
	if string(got[8:]) != wantSuffix {
		t.Fatalf("startup params = %q, want %q", got[8:], wantSuffix)
	}
}

func TestAppendSimpleAndExtendedQueryMessages(t *testing.T) {
	if got, want := AppendSimpleQuery(
		nil,
		"SELECT 1",
	), []byte{'Q', 0, 0, 0, 13, 'S', 'E', 'L', 'E', 'C', 'T', ' ', '1', 0}; !bytes.Equal(
		got,
		want,
	) {
		t.Fatalf("simple query = %#v, want %#v", got, want)
	}

	var got []byte
	got = AppendParse(
		got,
		"world_by_id",
		"SELECT id, randomNumber FROM World WHERE id=$1",
		[]uint32{23},
	)
	got = AppendBind(got, "", "world_by_id", [][]byte{[]byte("42")})
	got = AppendDescribePortal(got, "")
	got = AppendExecute(got, "", 0)
	got = AppendSync(got)

	frames := splitClientFrames(t, got)
	if string(frameTypes(frames)) != "PBDES" {
		t.Fatalf("extended frame types = %q, want PBDES", frameTypes(frames))
	}
}

func TestConnWriteRetriesPartialWritesUntilPayloadComplete(t *testing.T) {
	rwc := &partialWriteRWC{maxPerWrite: 3}
	conn := &Conn{rwc: rwc}
	payload := []byte("postgres-wire-frame")

	if err := conn.write(context.Background(), payload); err != nil {
		t.Fatalf("write partial payload: %v", err)
	}
	if !bytes.Equal(rwc.written, payload) {
		t.Fatalf("written payload = %#v, want %#v", rwc.written, payload)
	}
	if rwc.writes <= 1 {
		t.Fatalf("writes = %d, want multiple short writes", rwc.writes)
	}
}

func TestConnWriteRejectsZeroByteProgress(t *testing.T) {
	rwc := &partialWriteRWC{maxPerWrite: 0}
	conn := &Conn{rwc: rwc}

	err := conn.write(context.Background(), []byte("postgres-wire-frame"))
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("zero-progress write error = %v, want io.ErrShortWrite", err)
	}
}

func TestReadFrameRejectsMalformedLengths(t *testing.T) {
	_, err := ReadFrame(bytes.NewReader([]byte{'R', 0, 0, 0, 3}), 1024)
	if !errors.Is(err, ErrMalformedFrame) {
		t.Fatalf("short length error = %v, want ErrMalformedFrame", err)
	}
	_, err = ReadFrame(bytes.NewReader([]byte{'R', 0, 0, 4, 1}), 8)
	if !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("oversized length error = %v, want ErrFrameTooLarge", err)
	}
}

func TestDecodeRowDescriptionAndDataRow(t *testing.T) {
	rowDesc := appendRowDescription(nil, []fakeColumn{
		{Name: "id", TypeOID: Int4OID},
		{Name: "randomNumber", TypeOID: Int4OID},
	})
	cols, err := DecodeRowDescription(rowDesc)
	if err != nil {
		t.Fatalf("DecodeRowDescription: %v", err)
	}
	if len(cols) != 2 || cols[0].Name != "id" || cols[1].Name != "randomNumber" ||
		cols[1].TypeOID != Int4OID {
		t.Fatalf("columns = %#v", cols)
	}

	values, err := DecodeDataRow(appendDataRow(nil, []string{"1", "42"}))
	if err != nil {
		t.Fatalf("DecodeDataRow: %v", err)
	}
	if len(values) != 2 || string(values[0]) != "1" || string(values[1]) != "42" {
		t.Fatalf("values = %#v", values)
	}
}

func TestDecodeDataRowBorrowedDoesNotCopyCells(t *testing.T) {
	payload := appendDataRow(nil, []string{"7", "70"})
	row, report, err := DecodeDataRowBorrowed(payload, nil)
	if err != nil {
		t.Fatalf("DecodeDataRowBorrowed: %v", err)
	}
	if len(row) != 2 || string(row[0]) != "7" || string(row[1]) != "70" {
		t.Fatalf("row = %#v", row)
	}
	if report.BorrowedCells != 2 || report.CopiedCells != 0 ||
		report.Storage != RowStorageBorrowed {
		t.Fatalf("decode report = %#v", report)
	}
	payload[len(payload)-1] = '1'
	if string(row[1]) != "71" {
		t.Fatalf("row cell did not borrow payload after mutation: %q", row[1])
	}
}

func TestAppendBindBinaryFormatsAndDecodeInt4(t *testing.T) {
	var payload []byte
	payload = AppendBindFormat(
		payload,
		"",
		"world_by_id",
		[]int16{BinaryFormat},
		[][]byte{AppendInt4Binary(nil, 7)},
		[]int16{BinaryFormat},
	)
	frames := splitClientFrames(t, payload)
	if len(frames) != 1 || frames[0].Type != 'B' {
		t.Fatalf("frames = %#v, want one Bind frame", frames)
	}

	r := payloadReader{data: frames[0].Payload}
	portal, ok := r.cstring()
	if !ok || portal != "" {
		t.Fatalf("portal = %q ok=%v", portal, ok)
	}
	statement, ok := r.cstring()
	if !ok || statement != "world_by_id" {
		t.Fatalf("statement = %q ok=%v", statement, ok)
	}
	formatCount, ok := r.int16()
	if !ok || formatCount != 1 {
		t.Fatalf("format count = %d ok=%v", formatCount, ok)
	}
	format, ok := r.int16()
	if !ok || format != BinaryFormat {
		t.Fatalf("format = %d ok=%v", format, ok)
	}
	valueCount, ok := r.int16()
	if !ok || valueCount != 1 {
		t.Fatalf("value count = %d ok=%v", valueCount, ok)
	}
	valueLen, ok := r.int32()
	if !ok || valueLen != 4 {
		t.Fatalf("value length = %d ok=%v", valueLen, ok)
	}
	if got := binary.BigEndian.Uint32(r.data[r.off : r.off+4]); got != 7 {
		t.Fatalf("binary int4 payload = %d, want 7", got)
	}
	r.off += 4
	resultFormatCount, ok := r.int16()
	if !ok || resultFormatCount != 1 {
		t.Fatalf("result format count = %d ok=%v", resultFormatCount, ok)
	}
	resultFormat, ok := r.int16()
	if !ok || resultFormat != BinaryFormat {
		t.Fatalf("result format = %d ok=%v", resultFormat, ok)
	}
	if !r.done() {
		t.Fatalf("trailing bind payload bytes: %#v", r.data[r.off:])
	}

	encoded := AppendInt4Binary(nil, 70)
	decoded, err := DecodeInt4(encoded, BinaryFormat)
	if err != nil || decoded != 70 {
		t.Fatalf("DecodeInt4 binary = %d,%v want 70,nil", decoded, err)
	}
	decoded, err = DecodeInt4([]byte("71"), TextFormat)
	if err != nil || decoded != 71 {
		t.Fatalf("DecodeInt4 text = %d,%v want 71,nil", decoded, err)
	}
	if !bytes.Equal(AppendInt4Binary(nil, 7), []byte{0, 0, 0, 7}) {
		t.Fatalf("AppendInt4Binary did not encode big-endian int4")
	}
}

func TestClientSimpleQueryAgainstFakePostgresWireServer(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	errc := make(chan error, 1)
	go func() {
		errc <- serveFakeSimpleQuery(server, t)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := Connect(
		ctx,
		client,
		StartupConfig{User: "benchmarkdbuser", Database: "hello_world"},
	)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	result, err := conn.SimpleQuery(ctx, "SELECT id, randomNumber FROM World WHERE id=1")
	if err != nil {
		t.Fatalf("SimpleQuery: %v", err)
	}
	if len(result.Columns) != 2 || result.Columns[0].Name != "id" ||
		result.Columns[1].Name != "randomNumber" {
		t.Fatalf("columns = %#v", result.Columns)
	}
	if len(result.Rows) != 1 || result.Rows[0].String(0) != "1" ||
		result.Rows[0].String(1) != "42" {
		t.Fatalf("rows = %#v", result.Rows)
	}
	if result.CommandTag != "SELECT 1" {
		t.Fatalf("command tag = %q, want SELECT 1", result.CommandTag)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := <-errc; err != nil {
		t.Fatalf("fake server: %v", err)
	}
}

func TestClientRespondsToCleartextPasswordAuthentication(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	errc := make(chan error, 1)
	go func() {
		errc <- serveFakeCleartextPasswordStartup(server, "secret")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := Connect(ctx, client, StartupConfig{
		User:     "benchmarkdbuser",
		Database: "hello_world",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := <-errc; err != nil {
		t.Fatalf("fake server: %v", err)
	}
}

func TestClientRejectsCleartextPasswordAuthenticationWithoutPassword(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	errc := make(chan error, 1)
	go func() {
		if _, err := readStartupForTest(server); err != nil {
			errc <- err
			return
		}
		_, err := server.Write(serverFrame('R', int32Payload(3)))
		errc <- err
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := Connect(ctx, client, StartupConfig{User: "benchmarkdbuser", Database: "hello_world"})
	if !errors.Is(err, ErrUnsupportedAuth) {
		t.Fatalf("Connect without password = %v, want ErrUnsupportedAuth", err)
	}
	if err := <-errc; err != nil {
		t.Fatalf("fake server: %v", err)
	}
}

func TestClientCompletesSCRAMSHA256Authentication(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	errc := make(chan error, 1)
	go func() {
		errc <- serveFakeSCRAMSHA256Startup(server, "benchmarkdbuser", "secret")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := Connect(ctx, client, StartupConfig{
		User:     "benchmarkdbuser",
		Database: "hello_world",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := <-errc; err != nil {
		t.Fatalf("fake server: %v", err)
	}
}

func TestSCRAMSHA256RFC7677Vector(t *testing.T) {
	client, err := newSCRAMSHA256Client("user", "pencil", "rOprNGfwEbeRWgbNEkqO")
	if err != nil {
		t.Fatalf("newSCRAMSHA256Client: %v", err)
	}
	if got, want := client.ClientFirstMessage(), "n,,n=user,r=rOprNGfwEbeRWgbNEkqO"; got != want {
		t.Fatalf("client first = %q, want %q", got, want)
	}

	serverFirst := ("r=rOprNGfwEbeRWgbNEkqO%hvYDpWUa2RaTCAfuxFIlj)hNlF$k0," +
		"s=W22ZaJ0SNY7soEsUEjb6gQ==,i=4096")
	gotFinal, err := client.ClientFinalMessage(serverFirst)
	if err != nil {
		t.Fatalf("ClientFinalMessage: %v", err)
	}
	wantFinal := ("c=biws,r=rOprNGfwEbeRWgbNEkqO%hvYDpWUa2RaTCAfuxFIlj)hNlF$k0," +
		"p=dHzbZapWIk4jUhN+Ute9ytag9zjfMHgsqmmiz7AndVQ=")
	if gotFinal != wantFinal {
		t.Fatalf("client final = %q, want %q", gotFinal, wantFinal)
	}
	if err := client.VerifyServerFinal("v=6rriTRBi23WpRR/wtup+mMhUZUn/dB5nLTJRsjl95G4="); err != nil {
		t.Fatalf("VerifyServerFinal: %v", err)
	}
}

func TestSCRAMSHA256RejectsBadNonceAndSignature(t *testing.T) {
	client, err := newSCRAMSHA256Client("user", "pencil", "clientnonce")
	if err != nil {
		t.Fatalf("newSCRAMSHA256Client: %v", err)
	}
	if _, err := client.ClientFinalMessage("r=server-only,s=c2FsdA==,i=4096"); !errors.Is(
		err,
		ErrSCRAMAuthentication,
	) {
		t.Fatalf("nonce mismatch error = %v, want ErrSCRAMAuthentication", err)
	}

	client, err = newSCRAMSHA256Client("user", "pencil", "clientnonce")
	if err != nil {
		t.Fatalf("newSCRAMSHA256Client: %v", err)
	}
	if _, err := client.ClientFinalMessage("r=clientnonce-server,s=c2FsdA==,i=4096"); err != nil {
		t.Fatalf("ClientFinalMessage: %v", err)
	}
	if err := client.VerifyServerFinal("v=YmFk"); !errors.Is(err, ErrSCRAMAuthentication) {
		t.Fatalf("bad signature error = %v, want ErrSCRAMAuthentication", err)
	}
}

func TestClientRejectsSCRAMSHA256AuthenticationWithoutPassword(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	errc := make(chan error, 1)
	go func() {
		if _, err := readStartupForTest(server); err != nil {
			errc <- err
			return
		}
		_, err := server.Write(serverFrame('R', authSASLPayload("SCRAM-SHA-256")))
		errc <- err
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := Connect(ctx, client, StartupConfig{User: "benchmarkdbuser", Database: "hello_world"})
	if !errors.Is(err, ErrUnsupportedAuth) {
		t.Fatalf("Connect without password = %v, want ErrUnsupportedAuth", err)
	}
	if err := <-errc; err != nil {
		t.Fatalf("fake server: %v", err)
	}
}

func TestClientRejectsSCRAMSHA256AuthenticationWithoutServerFinal(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	errc := make(chan error, 1)
	go func() {
		errc <- serveFakeSCRAMSHA256MissingServerFinal(server, "benchmarkdbuser")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := Connect(ctx, client, StartupConfig{
		User:     "benchmarkdbuser",
		Database: "hello_world",
		Password: "secret",
	})
	if !errors.Is(err, ErrSCRAMAuthentication) {
		t.Fatalf("Connect without server final = %v, want ErrSCRAMAuthentication", err)
	}
	if err := <-errc; err != nil {
		t.Fatalf("fake server: %v", err)
	}
}

func TestDialConnectsToTCPPostgresWireServer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer ln.Close()

	errc := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			errc <- err
			return
		}
		defer conn.Close()
		errc <- serveFakeSimpleQuery(conn, t)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := Dial(ctx, DialConfig{
		Network: "tcp",
		Address: ln.Addr().String(),
		Timeout: time.Second,
		Startup: StartupConfig{User: "benchmarkdbuser", Database: "hello_world"},
	})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	result, err := conn.SimpleQuery(ctx, "SELECT id, randomNumber FROM World WHERE id=1")
	if err != nil {
		t.Fatalf("SimpleQuery: %v", err)
	}
	if len(result.Rows) != 1 || result.Rows[0].String(1) != "42" {
		t.Fatalf("rows = %#v", result.Rows)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := <-errc; err != nil {
		t.Fatalf("fake TCP server: %v", err)
	}
}

func TestDialRequiresAddress(t *testing.T) {
	_, err := Dial(context.Background(), DialConfig{
		Startup: StartupConfig{User: "benchmarkdbuser", Database: "hello_world"},
	})
	if !errors.Is(err, ErrMissingAddress) {
		t.Fatalf("Dial without address = %v, want ErrMissingAddress", err)
	}
}

func TestClientPreparedQueryUsesExtendedProtocol(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	errc := make(chan error, 1)
	go func() {
		errc <- serveFakePreparedQuery(server, t)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := Connect(
		ctx,
		client,
		StartupConfig{User: "benchmarkdbuser", Database: "hello_world"},
	)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	result, err := conn.PreparedQuery(
		ctx,
		"world_by_id",
		"SELECT id, randomNumber FROM World WHERE id=$1",
		[]uint32{Int4OID},
		[][]byte{[]byte("7")},
	)
	if err != nil {
		t.Fatalf("PreparedQuery: %v", err)
	}
	if len(result.Rows) != 1 || result.Rows[0].String(0) != "7" ||
		result.Rows[0].String(1) != "70" {
		t.Fatalf("rows = %#v", result.Rows)
	}
	if result.CommandTag != "SELECT 1" {
		t.Fatalf("command tag = %q, want SELECT 1", result.CommandTag)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := <-errc; err != nil {
		t.Fatalf("fake prepared server: %v", err)
	}
}

func FuzzReadFrameDoesNotPanic(f *testing.F) {
	for _, seed := range [][]byte{
		{'R', 0, 0, 0, 8, 0, 0, 0, 0},
		{'Z', 0, 0, 0, 5, 'I'},
		{'T', 0, 0, 0, 4},
		{'R', 0, 0, 0, 3},
		{'D', 0, 0, 4, 1},
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		frame, err := ReadFrame(bytes.NewReader(data), 1024)
		if err == nil && len(frame.Payload) > 1024 {
			t.Fatalf("payload len = %d, want <= 1024", len(frame.Payload))
		}
	})
}

func serveFakeSimpleQuery(conn net.Conn, t *testing.T) error {
	t.Helper()
	startup, err := readStartupForTest(conn)
	if err != nil {
		return err
	}
	if !bytes.Contains(startup, []byte("user\x00benchmarkdbuser\x00")) ||
		!bytes.Contains(startup, []byte("database\x00hello_world\x00")) {
		return errors.New("startup message missing expected user/database")
	}
	if _, err := conn.Write(serverFrame('R', int32Payload(0))); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('S', cstringsPayload("client_encoding", "UTF8"))); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('K', backendKeyPayload(7, 99))); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('Z', []byte{'I'})); err != nil {
		return err
	}

	frame, err := ReadFrame(conn, 1<<20)
	if err != nil {
		return err
	}
	if frame.Type != 'Q' ||
		string(
			bytes.TrimSuffix(frame.Payload, []byte{0}),
		) != "SELECT id, randomNumber FROM World WHERE id=1" {
		return errors.New("unexpected query frame")
	}
	if _, err := conn.Write(
		serverFrame('T', appendRowDescription(nil, []fakeColumn{
			{Name: "id", TypeOID: Int4OID},
			{Name: "randomNumber", TypeOID: Int4OID},
		})),
	); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('D', appendDataRow(nil, []string{"1", "42"}))); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('C', cstringPayload("SELECT 1"))); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('Z', []byte{'I'})); err != nil {
		return err
	}

	frame, err = ReadFrame(conn, 1<<20)
	if err != nil {
		return err
	}
	if frame.Type != 'X' {
		return errors.New("expected terminate frame")
	}
	return nil
}

func serveFakePreparedQuery(conn net.Conn, t *testing.T) error {
	t.Helper()
	if _, err := readStartupForTest(conn); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('R', int32Payload(0))); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('S', cstringsPayload("client_encoding", "UTF8"))); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('K', backendKeyPayload(7, 99))); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('Z', []byte{'I'})); err != nil {
		return err
	}

	frame, err := ReadFrame(conn, 1<<20)
	if err != nil {
		return err
	}
	if frame.Type != 'P' {
		return errors.New("expected Parse frame")
	}
	stmt, query, paramOIDs, err := parseClientParse(frame.Payload)
	if err != nil {
		return err
	}
	if stmt != "world_by_id" || query != "SELECT id, randomNumber FROM World WHERE id=$1" ||
		len(paramOIDs) != 1 ||
		paramOIDs[0] != Int4OID {
		return errors.New("unexpected Parse payload")
	}
	frame, err = ReadFrame(conn, 1<<20)
	if err != nil {
		return err
	}
	if frame.Type != 'S' {
		return errors.New("expected Sync after Parse")
	}
	if _, err := conn.Write(serverFrame('1', nil)); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('Z', []byte{'I'})); err != nil {
		return err
	}

	frames := make([]Frame, 0, 4)
	for i := 0; i < 4; i++ {
		frame, err := ReadFrame(conn, 1<<20)
		if err != nil {
			return err
		}
		frames = append(frames, frame)
	}
	if string(frameTypes(frames)) != "BDES" {
		return errors.New("expected Bind/Describe/Execute/Sync frames")
	}
	portal, statement, values, err := parseClientBind(frames[0].Payload)
	if err != nil {
		return err
	}
	if portal != "" || statement != "world_by_id" || len(values) != 1 || string(values[0]) != "7" {
		return errors.New("unexpected Bind payload")
	}
	describeKind, describeName, err := parseClientDescribe(frames[1].Payload)
	if err != nil {
		return err
	}
	if describeKind != 'P' || describeName != "" {
		return errors.New("unexpected Describe payload")
	}
	executePortal, maxRows, err := parseClientExecute(frames[2].Payload)
	if err != nil {
		return err
	}
	if executePortal != "" || maxRows != 0 {
		return errors.New("unexpected Execute payload")
	}
	if _, err := conn.Write(serverFrame('2', nil)); err != nil {
		return err
	}
	if _, err := conn.Write(
		serverFrame('T', appendRowDescription(nil, []fakeColumn{
			{Name: "id", TypeOID: Int4OID},
			{Name: "randomNumber", TypeOID: Int4OID},
		})),
	); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('D', appendDataRow(nil, []string{"7", "70"}))); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('C', cstringPayload("SELECT 1"))); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('Z', []byte{'I'})); err != nil {
		return err
	}
	frame, err = ReadFrame(conn, 1<<20)
	if err != nil {
		return err
	}
	if frame.Type != 'X' {
		return errors.New("expected terminate frame")
	}
	return nil
}

func serveFakeCleartextPasswordStartup(conn net.Conn, password string) error {
	if _, err := readStartupForTest(conn); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('R', int32Payload(3))); err != nil {
		return err
	}
	frame, err := ReadFrame(conn, 1<<20)
	if err != nil {
		return err
	}
	if frame.Type != 'p' {
		return errors.New("expected password message frame")
	}
	if !bytes.Equal(frame.Payload, appendCString(nil, password)) {
		return errors.New("unexpected password message payload")
	}
	if _, err := conn.Write(serverFrame('R', int32Payload(0))); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('S', cstringsPayload("client_encoding", "UTF8"))); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('K', backendKeyPayload(7, 99))); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('Z', []byte{'I'})); err != nil {
		return err
	}
	frame, err = ReadFrame(conn, 1<<20)
	if err != nil {
		return err
	}
	if frame.Type != 'X' {
		return errors.New("expected terminate frame")
	}
	return nil
}

func serveFakeSCRAMSHA256Startup(conn net.Conn, user string, password string) error {
	if _, err := readStartupForTest(conn); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('R', authSASLPayload("SCRAM-SHA-256"))); err != nil {
		return err
	}

	initial, err := ReadFrame(conn, 1<<20)
	if err != nil {
		return err
	}
	mechanism, clientFirst, err := parseClientSASLInitialResponse(initial)
	if err != nil {
		return err
	}
	if mechanism != "SCRAM-SHA-256" {
		return errors.New("unexpected SCRAM mechanism")
	}
	clientFirstBare, ok := strings.CutPrefix(clientFirst, "n,,")
	if !ok {
		return errors.New("SCRAM client-first message missing n,, GS2 header")
	}
	fields := parseSCRAMAttributesForTest(clientFirstBare)
	if fields["n"] != user || fields["r"] == "" {
		return errors.New("unexpected SCRAM client-first attributes")
	}

	serverNonce := fields["r"] + "SERVER"
	serverFirst := "r=" + serverNonce + ",s=c2FsdHlzYWx0,i=4096"
	if _, err := conn.Write(serverFrame('R', authSASLContinuePayload(serverFirst))); err != nil {
		return err
	}

	response, err := ReadFrame(conn, 1<<20)
	if err != nil {
		return err
	}
	clientFinal, err := parseClientSASLResponse(response)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(clientFinal, "c=biws,r="+serverNonce+",p=") {
		return errors.New("unexpected SCRAM client-final message")
	}
	withoutProof, _, ok := strings.Cut(clientFinal, ",p=")
	if !ok {
		return errors.New("SCRAM client-final missing proof")
	}
	serverFinal, err := buildSCRAMServerFinalForTest(
		password,
		clientFirstBare,
		serverFirst,
		withoutProof,
	)
	if err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('R', authSASLFinalPayload(serverFinal))); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('R', int32Payload(0))); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('S', cstringsPayload("client_encoding", "UTF8"))); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('K', backendKeyPayload(7, 99))); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('Z', []byte{'I'})); err != nil {
		return err
	}
	frame, err := ReadFrame(conn, 1<<20)
	if err != nil {
		return err
	}
	if frame.Type != 'X' {
		return errors.New("expected terminate frame")
	}
	return nil
}

func serveFakeSCRAMSHA256MissingServerFinal(conn net.Conn, user string) error {
	if _, err := readStartupForTest(conn); err != nil {
		return err
	}
	if _, err := conn.Write(serverFrame('R', authSASLPayload("SCRAM-SHA-256"))); err != nil {
		return err
	}

	initial, err := ReadFrame(conn, 1<<20)
	if err != nil {
		return err
	}
	mechanism, clientFirst, err := parseClientSASLInitialResponse(initial)
	if err != nil {
		return err
	}
	if mechanism != "SCRAM-SHA-256" {
		return errors.New("unexpected SCRAM mechanism")
	}
	clientFirstBare, ok := strings.CutPrefix(clientFirst, "n,,")
	if !ok {
		return errors.New("SCRAM client-first message missing n,, GS2 header")
	}
	fields := parseSCRAMAttributesForTest(clientFirstBare)
	if fields["n"] != user || fields["r"] == "" {
		return errors.New("unexpected SCRAM client-first attributes")
	}

	serverNonce := fields["r"] + "SERVER"
	serverFirst := "r=" + serverNonce + ",s=c2FsdHlzYWx0,i=4096"
	if _, err := conn.Write(serverFrame('R', authSASLContinuePayload(serverFirst))); err != nil {
		return err
	}

	if _, err := ReadFrame(conn, 1<<20); err != nil {
		return err
	}
	_, err = conn.Write(serverFrame('R', int32Payload(0)))
	return err
}

func readStartupForTest(r io.Reader) ([]byte, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, err
	}
	n := int(binary.BigEndian.Uint32(lenBuf[:]))
	if n < 8 {
		return nil, ErrMalformedFrame
	}
	payload := make([]byte, n-4)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

type fakeColumn struct {
	Name    string
	TypeOID uint32
}

func splitClientFrames(t *testing.T, raw []byte) []Frame {
	t.Helper()
	var frames []Frame
	r := bytes.NewReader(raw)
	for r.Len() > 0 {
		frame, err := ReadFrame(r, 1<<20)
		if err != nil {
			t.Fatalf("ReadFrame: %v", err)
		}
		frames = append(frames, frame)
	}
	return frames
}

func frameTypes(frames []Frame) []byte {
	types := make([]byte, len(frames))
	for i, frame := range frames {
		types[i] = frame.Type
	}
	return types
}

func serverFrame(typ byte, payload []byte) []byte {
	dst := []byte{typ}
	dst = appendInt32(dst, int32(len(payload)+4))
	dst = append(dst, payload...)
	return dst
}

func int32Payload(value int32) []byte {
	return appendInt32(nil, value)
}

func authSASLPayload(mechanisms ...string) []byte {
	payload := int32Payload(int32(AuthSASL))
	for _, mechanism := range mechanisms {
		payload = appendCString(payload, mechanism)
	}
	return append(payload, 0)
}

func authSASLContinuePayload(message string) []byte {
	payload := int32Payload(int32(AuthSASLContinue))
	return append(payload, message...)
}

func authSASLFinalPayload(message string) []byte {
	payload := int32Payload(int32(AuthSASLFinal))
	return append(payload, message...)
}

func backendKeyPayload(pid int32, secret int32) []byte {
	var dst []byte
	dst = appendInt32(dst, pid)
	dst = appendInt32(dst, secret)
	return dst
}

func cstringsPayload(values ...string) []byte {
	var dst []byte
	for _, value := range values {
		dst = appendCString(dst, value)
	}
	return dst
}

func cstringPayload(value string) []byte {
	return appendCString(nil, value)
}

func appendRowDescription(dst []byte, cols []fakeColumn) []byte {
	dst = appendInt16(dst, int16(len(cols)))
	for _, col := range cols {
		dst = appendCString(dst, col.Name)
		dst = appendInt32(dst, 0)
		dst = appendInt16(dst, 0)
		dst = appendInt32(dst, int32(col.TypeOID))
		dst = appendInt16(dst, -1)
		dst = appendInt32(dst, -1)
		dst = appendInt16(dst, 0)
	}
	return dst
}

func appendDataRow(dst []byte, values []string) []byte {
	dst = appendInt16(dst, int16(len(values)))
	for _, value := range values {
		dst = appendInt32(dst, int32(len(value)))
		dst = append(dst, value...)
	}
	return dst
}

func parseClientParse(payload []byte) (string, string, []uint32, error) {
	r := payloadReader{data: payload}
	statement, ok := r.cstring()
	if !ok {
		return "", "", nil, ErrMalformedFrame
	}
	query, ok := r.cstring()
	if !ok {
		return "", "", nil, ErrMalformedFrame
	}
	count, ok := r.int16()
	if !ok || count < 0 {
		return "", "", nil, ErrMalformedFrame
	}
	oids := make([]uint32, 0, count)
	for i := 0; i < int(count); i++ {
		oid, ok := r.uint32()
		if !ok {
			return "", "", nil, ErrMalformedFrame
		}
		oids = append(oids, oid)
	}
	if !r.done() {
		return "", "", nil, ErrMalformedFrame
	}
	return statement, query, oids, nil
}

func parseClientBind(payload []byte) (string, string, [][]byte, error) {
	r := payloadReader{data: payload}
	portal, ok := r.cstring()
	if !ok {
		return "", "", nil, ErrMalformedFrame
	}
	statement, ok := r.cstring()
	if !ok {
		return "", "", nil, ErrMalformedFrame
	}
	formatCount, ok := r.int16()
	if !ok || formatCount < 0 {
		return "", "", nil, ErrMalformedFrame
	}
	for i := 0; i < int(formatCount); i++ {
		if _, ok := r.int16(); !ok {
			return "", "", nil, ErrMalformedFrame
		}
	}
	valueCount, ok := r.int16()
	if !ok || valueCount < 0 {
		return "", "", nil, ErrMalformedFrame
	}
	values := make([][]byte, 0, valueCount)
	for i := 0; i < int(valueCount); i++ {
		n, ok := r.int32()
		if !ok {
			return "", "", nil, ErrMalformedFrame
		}
		if n == -1 {
			values = append(values, nil)
			continue
		}
		if n < 0 || len(r.data)-r.off < int(n) {
			return "", "", nil, ErrMalformedFrame
		}
		values = append(values, append([]byte(nil), r.data[r.off:r.off+int(n)]...))
		r.off += int(n)
	}
	resultFormatCount, ok := r.int16()
	if !ok || resultFormatCount < 0 {
		return "", "", nil, ErrMalformedFrame
	}
	for i := 0; i < int(resultFormatCount); i++ {
		if _, ok := r.int16(); !ok {
			return "", "", nil, ErrMalformedFrame
		}
	}
	if !r.done() {
		return "", "", nil, ErrMalformedFrame
	}
	return portal, statement, values, nil
}

func parseClientDescribe(payload []byte) (byte, string, error) {
	r := payloadReader{data: payload}
	if len(r.data)-r.off < 1 {
		return 0, "", ErrMalformedFrame
	}
	kind := r.data[r.off]
	r.off++
	name, ok := r.cstring()
	if !ok || !r.done() {
		return 0, "", ErrMalformedFrame
	}
	return kind, name, nil
}

func parseClientExecute(payload []byte) (string, int32, error) {
	r := payloadReader{data: payload}
	portal, ok := r.cstring()
	if !ok {
		return "", 0, ErrMalformedFrame
	}
	maxRows, ok := r.int32()
	if !ok || !r.done() {
		return "", 0, ErrMalformedFrame
	}
	return portal, maxRows, nil
}

func parseClientSASLInitialResponse(frame Frame) (string, string, error) {
	if frame.Type != 'p' {
		return "", "", errors.New("expected SASLInitialResponse password frame")
	}
	r := payloadReader{data: frame.Payload}
	mechanism, ok := r.cstring()
	if !ok {
		return "", "", ErrMalformedFrame
	}
	n, ok := r.int32()
	if !ok || n < 0 || len(r.data)-r.off < int(n) {
		return "", "", ErrMalformedFrame
	}
	response := string(r.data[r.off : r.off+int(n)])
	r.off += int(n)
	if !r.done() {
		return "", "", ErrMalformedFrame
	}
	return mechanism, response, nil
}

func parseClientSASLResponse(frame Frame) (string, error) {
	if frame.Type != 'p' {
		return "", errors.New("expected SASLResponse password frame")
	}
	return string(frame.Payload), nil
}

func parseSCRAMAttributesForTest(message string) map[string]string {
	fields, err := parseSCRAMAttributes(message)
	if err != nil {
		return map[string]string{}
	}
	return fields
}

func buildSCRAMServerFinalForTest(
	password string,
	clientFirstBare string,
	serverFirst string,
	clientFinalWithoutProof string,
) (string, error) {
	fields, err := parseSCRAMAttributes(serverFirst)
	if err != nil {
		return "", err
	}
	salt, err := base64.StdEncoding.DecodeString(fields["s"])
	if err != nil {
		return "", err
	}
	iterations, err := strconv.Atoi(fields["i"])
	if err != nil {
		return "", err
	}
	authMessage := clientFirstBare + "," + serverFirst + "," + clientFinalWithoutProof
	signature := scramSHA256ServerSignature(password, salt, iterations, authMessage)
	return "v=" + base64.StdEncoding.EncodeToString(signature), nil
}

type partialWriteRWC struct {
	maxPerWrite int
	writes      int
	written     []byte
}

func (rw *partialWriteRWC) Read(p []byte) (int, error) {
	return 0, io.EOF
}

func (rw *partialWriteRWC) Write(p []byte) (int, error) {
	rw.writes++
	if rw.maxPerWrite <= 0 {
		return 0, nil
	}
	n := rw.maxPerWrite
	if n > len(p) {
		n = len(p)
	}
	rw.written = append(rw.written, p[:n]...)
	return n, nil
}

func (rw *partialWriteRWC) Close() error {
	return nil
}
