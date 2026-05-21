package pgrt

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sort"
)

const (
	ProtocolVersion30     uint32 = 196608
	AuthOK                uint32 = 0
	AuthCleartextPassword uint32 = 3
	AuthSASL              uint32 = 10
	AuthSASLContinue      uint32 = 11
	AuthSASLFinal         uint32 = 12
	Int4OID               uint32 = 23
)

var (
	ErrMalformedFrame       = errors.New("malformed PostgreSQL frame")
	ErrFrameTooLarge        = errors.New("PostgreSQL frame exceeds limit")
	ErrUnsupportedAuth      = errors.New("unsupported PostgreSQL authentication method")
	ErrUnexpectedMessage    = errors.New("unexpected PostgreSQL message")
	ErrPostgresErrorMessage = errors.New("PostgreSQL error response")
)

type StartupConfig struct {
	User       string
	Database   string
	Password   string
	Parameters map[string]string
}

type Frame struct {
	Type    byte
	Payload []byte
}

type Column struct {
	Name        string
	TableOID    uint32
	ColumnIndex int16
	TypeOID     uint32
	TypeSize    int16
	TypeMod     int32
	Format      int16
}

type Row [][]byte

func (r Row) String(index int) string {
	if index < 0 || index >= len(r) || r[index] == nil {
		return ""
	}
	return string(r[index])
}

type Result struct {
	Columns    []Column
	Rows       []Row
	CommandTag string
}

func AppendStartupMessage(dst []byte, cfg StartupConfig) []byte {
	start := len(dst)
	dst = appendInt32(dst, 0)
	dst = appendInt32(dst, int32(ProtocolVersion30))
	dst = appendCString(dst, "user")
	dst = appendCString(dst, cfg.User)
	dst = appendCString(dst, "database")
	dst = appendCString(dst, cfg.Database)
	keys := make([]string, 0, len(cfg.Parameters))
	for key := range cfg.Parameters {
		if key == "user" || key == "database" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		dst = appendCString(dst, key)
		dst = appendCString(dst, cfg.Parameters[key])
	}
	dst = append(dst, 0)
	binary.BigEndian.PutUint32(dst[start:start+4], uint32(len(dst)-start))
	return dst
}

func AppendSimpleQuery(dst []byte, query string) []byte {
	return appendTypedCStringFrame(dst, 'Q', query)
}

func AppendPasswordMessage(dst []byte, password string) []byte {
	return appendTypedCStringFrame(dst, 'p', password)
}

func AppendSASLInitialResponse(dst []byte, mechanism string, response []byte) []byte {
	return appendTypedPayload(dst, 'p', func(payload []byte) []byte {
		payload = appendCString(payload, mechanism)
		payload = appendInt32(payload, int32(len(response)))
		payload = append(payload, response...)
		return payload
	})
}

func AppendSASLResponse(dst []byte, response []byte) []byte {
	return appendTypedFrame(dst, 'p', response)
}

func AppendTerminate(dst []byte) []byte {
	return appendTypedFrame(dst, 'X', nil)
}

func AppendParse(dst []byte, statement string, query string, paramTypeOIDs []uint32) []byte {
	return appendTypedPayload(dst, 'P', func(payload []byte) []byte {
		payload = appendCString(payload, statement)
		payload = appendCString(payload, query)
		payload = appendInt16(payload, int16(len(paramTypeOIDs)))
		for _, oid := range paramTypeOIDs {
			payload = appendInt32(payload, int32(oid))
		}
		return payload
	})
}

func AppendBind(dst []byte, portal string, statement string, values [][]byte) []byte {
	return appendTypedPayload(dst, 'B', func(payload []byte) []byte {
		payload = appendCString(payload, portal)
		payload = appendCString(payload, statement)
		payload = appendInt16(payload, 0)
		payload = appendInt16(payload, int16(len(values)))
		for _, value := range values {
			if value == nil {
				payload = appendInt32(payload, -1)
				continue
			}
			payload = appendInt32(payload, int32(len(value)))
			payload = append(payload, value...)
		}
		payload = appendInt16(payload, 0)
		return payload
	})
}

func AppendDescribePortal(dst []byte, portal string) []byte {
	return appendTypedPayload(dst, 'D', func(payload []byte) []byte {
		payload = append(payload, 'P')
		payload = appendCString(payload, portal)
		return payload
	})
}

func AppendExecute(dst []byte, portal string, maxRows int32) []byte {
	return appendTypedPayload(dst, 'E', func(payload []byte) []byte {
		payload = appendCString(payload, portal)
		payload = appendInt32(payload, maxRows)
		return payload
	})
}

func AppendSync(dst []byte) []byte {
	return appendTypedFrame(dst, 'S', nil)
}

func ReadFrame(r io.Reader, maxPayload int) (Frame, error) {
	if maxPayload <= 0 {
		maxPayload = 1 << 20
	}
	var header [5]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return Frame{}, err
	}
	length := int(binary.BigEndian.Uint32(header[1:5]))
	if length < 4 {
		return Frame{}, ErrMalformedFrame
	}
	payloadLen := length - 4
	if payloadLen > maxPayload {
		return Frame{}, ErrFrameTooLarge
	}
	payload := make([]byte, payloadLen)
	if _, err := io.ReadFull(r, payload); err != nil {
		return Frame{}, err
	}
	return Frame{Type: header[0], Payload: payload}, nil
}

func DecodeRowDescription(payload []byte) ([]Column, error) {
	r := payloadReader{data: payload}
	count, ok := r.int16()
	if !ok || count < 0 {
		return nil, ErrMalformedFrame
	}
	cols := make([]Column, 0, count)
	for i := 0; i < int(count); i++ {
		name, ok := r.cstring()
		if !ok {
			return nil, ErrMalformedFrame
		}
		tableOID, ok := r.uint32()
		if !ok {
			return nil, ErrMalformedFrame
		}
		columnIndex, ok := r.int16()
		if !ok {
			return nil, ErrMalformedFrame
		}
		typeOID, ok := r.uint32()
		if !ok {
			return nil, ErrMalformedFrame
		}
		typeSize, ok := r.int16()
		if !ok {
			return nil, ErrMalformedFrame
		}
		typeMod, ok := r.int32()
		if !ok {
			return nil, ErrMalformedFrame
		}
		format, ok := r.int16()
		if !ok {
			return nil, ErrMalformedFrame
		}
		cols = append(cols, Column{
			Name:        name,
			TableOID:    tableOID,
			ColumnIndex: columnIndex,
			TypeOID:     typeOID,
			TypeSize:    typeSize,
			TypeMod:     typeMod,
			Format:      format,
		})
	}
	if !r.done() {
		return nil, ErrMalformedFrame
	}
	return cols, nil
}

func DecodeDataRow(payload []byte) (Row, error) {
	r := payloadReader{data: payload}
	count, ok := r.int16()
	if !ok || count < 0 {
		return nil, ErrMalformedFrame
	}
	row := make(Row, 0, count)
	for i := 0; i < int(count); i++ {
		n, ok := r.int32()
		if !ok {
			return nil, ErrMalformedFrame
		}
		if n == -1 {
			row = append(row, nil)
			continue
		}
		if n < 0 || len(r.data)-r.off < int(n) {
			return nil, ErrMalformedFrame
		}
		value := append([]byte(nil), r.data[r.off:r.off+int(n)]...)
		r.off += int(n)
		row = append(row, value)
	}
	if !r.done() {
		return nil, ErrMalformedFrame
	}
	return row, nil
}

func DecodeCommandComplete(payload []byte) (string, error) {
	r := payloadReader{data: payload}
	tag, ok := r.cstring()
	if !ok || !r.done() {
		return "", ErrMalformedFrame
	}
	return tag, nil
}

func DecodeErrorResponse(payload []byte) error {
	r := payloadReader{data: payload}
	fields := map[byte]string{}
	for r.off < len(r.data) {
		code := r.data[r.off]
		r.off++
		if code == 0 {
			if !r.done() {
				return ErrMalformedFrame
			}
			if msg := fields['M']; msg != "" {
				return fmt.Errorf("%w: %s", ErrPostgresErrorMessage, msg)
			}
			return ErrPostgresErrorMessage
		}
		value, ok := r.cstring()
		if !ok {
			return ErrMalformedFrame
		}
		fields[code] = value
	}
	return ErrMalformedFrame
}

type Conn struct {
	rwc        io.ReadWriteCloser
	maxPayload int
	prepared   map[string]bool
}

func Connect(ctx context.Context, rwc io.ReadWriteCloser, cfg StartupConfig) (*Conn, error) {
	conn := &Conn{rwc: rwc, maxPayload: 16 << 20, prepared: map[string]bool{}}
	if err := conn.write(ctx, AppendStartupMessage(nil, cfg)); err != nil {
		return nil, err
	}
	if err := conn.readStartupReady(ctx, cfg); err != nil {
		_ = rwc.Close()
		return nil, err
	}
	return conn, nil
}

func (c *Conn) Prepare(ctx context.Context, name string, query string, paramTypeOIDs []uint32) error {
	if name != "" && c.prepared[name] {
		return nil
	}
	if c.prepared == nil {
		c.prepared = map[string]bool{}
	}
	var payload []byte
	payload = AppendParse(payload, name, query, paramTypeOIDs)
	payload = AppendSync(payload)
	if err := c.write(ctx, payload); err != nil {
		return err
	}
	for {
		frame, err := c.readFrame(ctx)
		if err != nil {
			return err
		}
		switch frame.Type {
		case '1':
			if len(frame.Payload) != 0 {
				return ErrMalformedFrame
			}
		case 'E':
			return DecodeErrorResponse(frame.Payload)
		case 'Z':
			if name != "" {
				c.prepared[name] = true
			}
			return nil
		default:
			return fmt.Errorf("%w: %c during prepare", ErrUnexpectedMessage, frame.Type)
		}
	}
}

func (c *Conn) ExecPrepared(ctx context.Context, name string, values [][]byte) (Result, error) {
	var payload []byte
	payload = AppendBind(payload, "", name, values)
	payload = AppendDescribePortal(payload, "")
	payload = AppendExecute(payload, "", 0)
	payload = AppendSync(payload)
	if err := c.write(ctx, payload); err != nil {
		return Result{}, err
	}
	return c.readQueryResult(ctx, "prepared query")
}

func (c *Conn) PreparedQuery(ctx context.Context, name string, query string, paramTypeOIDs []uint32, values [][]byte) (Result, error) {
	if err := c.Prepare(ctx, name, query, paramTypeOIDs); err != nil {
		return Result{}, err
	}
	return c.ExecPrepared(ctx, name, values)
}

func (c *Conn) SimpleQuery(ctx context.Context, query string) (Result, error) {
	if err := c.write(ctx, AppendSimpleQuery(nil, query)); err != nil {
		return Result{}, err
	}
	return c.readQueryResult(ctx, "simple query")
}

func (c *Conn) readQueryResult(ctx context.Context, phase string) (Result, error) {
	var result Result
	for {
		frame, err := c.readFrame(ctx)
		if err != nil {
			return Result{}, err
		}
		switch frame.Type {
		case '2':
			if len(frame.Payload) != 0 {
				return Result{}, ErrMalformedFrame
			}
		case 'n':
		case 'T':
			cols, err := DecodeRowDescription(frame.Payload)
			if err != nil {
				return Result{}, err
			}
			result.Columns = cols
		case 'D':
			row, err := DecodeDataRow(frame.Payload)
			if err != nil {
				return Result{}, err
			}
			result.Rows = append(result.Rows, row)
		case 'C':
			tag, err := DecodeCommandComplete(frame.Payload)
			if err != nil {
				return Result{}, err
			}
			result.CommandTag = tag
		case 'E':
			return Result{}, DecodeErrorResponse(frame.Payload)
		case 'Z':
			return result, nil
		default:
			return Result{}, fmt.Errorf("%w: %c during %s", ErrUnexpectedMessage, frame.Type, phase)
		}
	}
}

func (c *Conn) Close() error {
	if c == nil || c.rwc == nil {
		return nil
	}
	err := c.write(context.Background(), AppendTerminate(nil))
	closeErr := c.rwc.Close()
	c.rwc = nil
	if err != nil {
		return err
	}
	return closeErr
}

func (c *Conn) readStartupReady(ctx context.Context, cfg StartupConfig) error {
	var scram *scramSHA256Client
	for {
		frame, err := c.readFrame(ctx)
		if err != nil {
			return err
		}
		switch frame.Type {
		case 'R':
			if len(frame.Payload) < 4 {
				return ErrMalformedFrame
			}
			auth := binary.BigEndian.Uint32(frame.Payload[:4])
			switch auth {
			case AuthOK:
				if scram != nil && !scram.serverFinalChecked {
					return fmt.Errorf("%w: missing server-final-message", ErrSCRAMAuthentication)
				}
			case AuthCleartextPassword:
				if cfg.Password == "" {
					return fmt.Errorf("%w: cleartext password requested but no password configured", ErrUnsupportedAuth)
				}
				if err := c.write(ctx, AppendPasswordMessage(nil, cfg.Password)); err != nil {
					return err
				}
			case AuthSASL:
				if cfg.Password == "" {
					return fmt.Errorf("%w: SASL authentication requested but no password configured", ErrUnsupportedAuth)
				}
				mechanisms, err := DecodeAuthenticationSASL(frame.Payload[4:])
				if err != nil {
					return err
				}
				if !containsString(mechanisms, scramSHA256Mechanism) {
					return fmt.Errorf("%w: server did not advertise SCRAM-SHA-256", ErrUnsupportedAuth)
				}
				scram, err = newSCRAMSHA256ClientWithRandomNonce(cfg.User, cfg.Password)
				if err != nil {
					return err
				}
				if err := c.write(ctx, AppendSASLInitialResponse(nil, scramSHA256Mechanism, []byte(scram.ClientFirstMessage()))); err != nil {
					return err
				}
			case AuthSASLContinue:
				if scram == nil {
					return fmt.Errorf("%w: SASLContinue before AuthenticationSASL", ErrUnexpectedMessage)
				}
				final, err := scram.ClientFinalMessage(string(frame.Payload[4:]))
				if err != nil {
					return err
				}
				if err := c.write(ctx, AppendSASLResponse(nil, []byte(final))); err != nil {
					return err
				}
			case AuthSASLFinal:
				if scram == nil {
					return fmt.Errorf("%w: SASLFinal before AuthenticationSASL", ErrUnexpectedMessage)
				}
				if err := scram.VerifyServerFinal(string(frame.Payload[4:])); err != nil {
					return err
				}
			default:
				return fmt.Errorf("%w: %d", ErrUnsupportedAuth, auth)
			}
		case 'S', 'K', 'N':
		case 'E':
			return DecodeErrorResponse(frame.Payload)
		case 'Z':
			if scram != nil && !scram.serverFinalChecked {
				return fmt.Errorf("%w: missing server-final-message", ErrSCRAMAuthentication)
			}
			return nil
		default:
			return fmt.Errorf("%w: %c during startup", ErrUnexpectedMessage, frame.Type)
		}
	}
}

func DecodeAuthenticationSASL(payload []byte) ([]string, error) {
	r := payloadReader{data: payload}
	var mechanisms []string
	for {
		mechanism, ok := r.cstring()
		if !ok {
			return nil, ErrMalformedFrame
		}
		if mechanism == "" {
			if !r.done() {
				return nil, ErrMalformedFrame
			}
			return mechanisms, nil
		}
		mechanisms = append(mechanisms, mechanism)
	}
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func (c *Conn) readFrame(ctx context.Context) (Frame, error) {
	if err := ctx.Err(); err != nil {
		return Frame{}, err
	}
	frame, err := ReadFrame(c.rwc, c.maxPayload)
	if err != nil {
		return Frame{}, err
	}
	if err := ctx.Err(); err != nil {
		return Frame{}, err
	}
	return frame, nil
}

func (c *Conn) write(ctx context.Context, payload []byte) error {
	for len(payload) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, err := c.rwc.Write(payload)
		if n < 0 || n > len(payload) {
			return io.ErrShortWrite
		}
		if n > 0 {
			payload = payload[n:]
		}
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
	}
	return ctx.Err()
}

type payloadReader struct {
	data []byte
	off  int
}

func (r *payloadReader) done() bool {
	return r.off == len(r.data)
}

func (r *payloadReader) int16() (int16, bool) {
	if len(r.data)-r.off < 2 {
		return 0, false
	}
	value := int16(binary.BigEndian.Uint16(r.data[r.off : r.off+2]))
	r.off += 2
	return value, true
}

func (r *payloadReader) int32() (int32, bool) {
	if len(r.data)-r.off < 4 {
		return 0, false
	}
	value := int32(binary.BigEndian.Uint32(r.data[r.off : r.off+4]))
	r.off += 4
	return value, true
}

func (r *payloadReader) uint32() (uint32, bool) {
	if len(r.data)-r.off < 4 {
		return 0, false
	}
	value := binary.BigEndian.Uint32(r.data[r.off : r.off+4])
	r.off += 4
	return value, true
}

func (r *payloadReader) cstring() (string, bool) {
	for i := r.off; i < len(r.data); i++ {
		if r.data[i] == 0 {
			value := string(r.data[r.off:i])
			r.off = i + 1
			return value, true
		}
	}
	return "", false
}

func appendTypedCStringFrame(dst []byte, typ byte, value string) []byte {
	return appendTypedPayload(dst, typ, func(payload []byte) []byte {
		return appendCString(payload, value)
	})
}

func appendTypedFrame(dst []byte, typ byte, payload []byte) []byte {
	dst = append(dst, typ)
	dst = appendInt32(dst, int32(len(payload)+4))
	dst = append(dst, payload...)
	return dst
}

func appendTypedPayload(dst []byte, typ byte, build func([]byte) []byte) []byte {
	start := len(dst)
	dst = append(dst, typ)
	dst = appendInt32(dst, 0)
	dst = build(dst)
	binary.BigEndian.PutUint32(dst[start+1:start+5], uint32(len(dst)-start-1))
	return dst
}

func appendCString(dst []byte, value string) []byte {
	dst = append(dst, value...)
	dst = append(dst, 0)
	return dst
}

func appendInt16(dst []byte, value int16) []byte {
	var b [2]byte
	binary.BigEndian.PutUint16(b[:], uint16(value))
	return append(dst, b[:]...)
}

func appendInt32(dst []byte, value int32) []byte {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], uint32(value))
	return append(dst, b[:]...)
}
