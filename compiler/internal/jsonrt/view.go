package jsonrt

import (
	"bytes"
	"errors"

	"tetra_language/compiler/internal/stdlibrt"
)

var ErrInvalidViewJSON = errors.New("invalid JSON view")

type ParseViewOptions struct {
	Region *stdlibrt.Region
}

type ParseViewReport struct {
	HeapAllocations   int
	RegionTemporaries int
	BorrowedStrings   int
	CopiedStrings     int
	UnsafeFacts       bool
}

type ViewMember struct {
	Name  stdlibrt.BytesView
	Value ValueView
}

type ValueView struct {
	Kind   Kind
	Bool   bool
	Number []byte
	String stdlibrt.BytesView
	Array  []ValueView
	Object []ViewMember
}

func (v ValueView) ObjectMember(name string) *ValueView {
	for i := range v.Object {
		if bytes.Equal(v.Object[i].Name.Bytes, []byte(name)) {
			return &v.Object[i].Value
		}
	}
	return nil
}

func ParseValueView(raw []byte, opts ParseViewOptions) (ValueView, ParseViewReport, error) {
	p := viewParser{raw: raw, opts: opts}
	value, err := p.parseValue()
	if err != nil {
		return ValueView{}, ParseViewReport{}, err
	}
	p.skipSpace()
	if p.off != len(raw) {
		return ValueView{}, p.report, ErrInvalidViewJSON
	}
	return value, p.report, nil
}

type viewParser struct {
	raw    []byte
	off    int
	opts   ParseViewOptions
	report ParseViewReport
}

func (p *viewParser) parseValue() (ValueView, error) {
	p.skipSpace()
	if p.off >= len(p.raw) {
		return ValueView{}, ErrInvalidViewJSON
	}
	switch p.raw[p.off] {
	case '{':
		return p.parseObject()
	case '[':
		return p.parseArray()
	case '"':
		view, err := p.parseString()
		if err != nil {
			return ValueView{}, err
		}
		return ValueView{Kind: StringKind, String: view}, nil
	case 't':
		if !p.consumeLiteral("true") {
			return ValueView{}, ErrInvalidViewJSON
		}
		return ValueView{Kind: BoolKind, Bool: true}, nil
	case 'f':
		if !p.consumeLiteral("false") {
			return ValueView{}, ErrInvalidViewJSON
		}
		return ValueView{Kind: BoolKind}, nil
	case 'n':
		if !p.consumeLiteral("null") {
			return ValueView{}, ErrInvalidViewJSON
		}
		return ValueView{Kind: NullKind}, nil
	default:
		number, err := p.parseNumber()
		if err != nil {
			return ValueView{}, err
		}
		return ValueView{Kind: NumberKind, Number: number}, nil
	}
}

func (p *viewParser) parseObject() (ValueView, error) {
	p.off++
	p.skipSpace()
	var members []ViewMember
	if p.off < len(p.raw) && p.raw[p.off] == '}' {
		p.off++
		return ValueView{Kind: ObjectKind}, nil
	}
	for {
		p.skipSpace()
		if p.off >= len(p.raw) || p.raw[p.off] != '"' {
			return ValueView{}, ErrInvalidViewJSON
		}
		name, err := p.parseString()
		if err != nil {
			return ValueView{}, err
		}
		p.skipSpace()
		if p.off >= len(p.raw) || p.raw[p.off] != ':' {
			return ValueView{}, ErrInvalidViewJSON
		}
		p.off++
		value, err := p.parseValue()
		if err != nil {
			return ValueView{}, err
		}
		members = append(members, ViewMember{Name: name, Value: value})
		p.skipSpace()
		if p.off >= len(p.raw) {
			return ValueView{}, ErrInvalidViewJSON
		}
		if p.raw[p.off] == '}' {
			p.off++
			return ValueView{Kind: ObjectKind, Object: members}, nil
		}
		if p.raw[p.off] != ',' {
			return ValueView{}, ErrInvalidViewJSON
		}
		p.off++
	}
}

func (p *viewParser) parseArray() (ValueView, error) {
	p.off++
	p.skipSpace()
	var items []ValueView
	if p.off < len(p.raw) && p.raw[p.off] == ']' {
		p.off++
		return ValueView{Kind: ArrayKind}, nil
	}
	for {
		value, err := p.parseValue()
		if err != nil {
			return ValueView{}, err
		}
		items = append(items, value)
		p.skipSpace()
		if p.off >= len(p.raw) {
			return ValueView{}, ErrInvalidViewJSON
		}
		if p.raw[p.off] == ']' {
			p.off++
			return ValueView{Kind: ArrayKind, Array: items}, nil
		}
		if p.raw[p.off] != ',' {
			return ValueView{}, ErrInvalidViewJSON
		}
		p.off++
	}
}

func (p *viewParser) parseString() (stdlibrt.BytesView, error) {
	if p.off >= len(p.raw) || p.raw[p.off] != '"' {
		return stdlibrt.BytesView{}, ErrInvalidViewJSON
	}
	p.off++
	start := p.off
	for p.off < len(p.raw) {
		c := p.raw[p.off]
		switch {
		case c == '"':
			view := stdlibrt.BytesView{
				Bytes:      p.raw[start:p.off],
				Storage:    stdlibrt.StorageBorrowed,
				Provenance: "json.input",
			}
			p.off++
			p.report.BorrowedStrings++
			return view, nil
		case c == '\\':
			return p.parseEscapedString(start)
		case c < 0x20:
			return stdlibrt.BytesView{}, ErrInvalidViewJSON
		default:
			p.off++
		}
	}
	return stdlibrt.BytesView{}, ErrInvalidViewJSON
}

func (p *viewParser) parseEscapedString(start int) (stdlibrt.BytesView, error) {
	if p.opts.Region != nil {
		return p.parseEscapedStringIntoRegion(start)
	}
	out := make([]byte, 0, len(p.raw)-start)
	out = append(out, p.raw[start:p.off]...)
	for p.off < len(p.raw) {
		c := p.raw[p.off]
		if c == '"' {
			p.off++
			return p.copyString(out)
		}
		if c < 0x20 {
			return stdlibrt.BytesView{}, ErrInvalidViewJSON
		}
		if c != '\\' {
			out = append(out, c)
			p.off++
			continue
		}
		p.off++
		if p.off >= len(p.raw) {
			return stdlibrt.BytesView{}, ErrInvalidViewJSON
		}
		escaped := p.raw[p.off]
		p.off++
		switch escaped {
		case '"', '\\', '/':
			out = append(out, escaped)
		case 'b':
			out = append(out, '\b')
		case 'f':
			out = append(out, '\f')
		case 'n':
			out = append(out, '\n')
		case 'r':
			out = append(out, '\r')
		case 't':
			out = append(out, '\t')
		case 'u':
			if len(p.raw)-p.off < 4 {
				return stdlibrt.BytesView{}, ErrInvalidViewJSON
			}
			value, ok := parseHex4(p.raw[p.off : p.off+4])
			if !ok || value > 0x7f {
				return stdlibrt.BytesView{}, ErrInvalidViewJSON
			}
			out = append(out, byte(value))
			p.off += 4
		default:
			return stdlibrt.BytesView{}, ErrInvalidViewJSON
		}
	}
	return stdlibrt.BytesView{}, ErrInvalidViewJSON
}

func (p *viewParser) parseEscapedStringIntoRegion(start int) (stdlibrt.BytesView, error) {
	capacity := len(p.raw) - start
	if capacity < 0 {
		capacity = 0
	}
	buf, err := p.opts.Region.Alloc(capacity)
	if err != nil {
		return stdlibrt.BytesView{}, err
	}
	out := buf[:0]
	out = append(out, p.raw[start:p.off]...)
	for p.off < len(p.raw) {
		c := p.raw[p.off]
		if c == '"' {
			p.off++
			p.report.CopiedStrings++
			p.report.RegionTemporaries++
			return stdlibrt.BytesView{
				Bytes:      out,
				Storage:    stdlibrt.StorageRegion,
				RegionID:   p.opts.Region.ID(),
				Provenance: "json.unescaped",
				Copied:     true,
			}, nil
		}
		if c < 0x20 {
			return stdlibrt.BytesView{}, ErrInvalidViewJSON
		}
		if c != '\\' {
			out = append(out, c)
			p.off++
			continue
		}
		p.off++
		if p.off >= len(p.raw) {
			return stdlibrt.BytesView{}, ErrInvalidViewJSON
		}
		escaped := p.raw[p.off]
		p.off++
		switch escaped {
		case '"', '\\', '/':
			out = append(out, escaped)
		case 'b':
			out = append(out, '\b')
		case 'f':
			out = append(out, '\f')
		case 'n':
			out = append(out, '\n')
		case 'r':
			out = append(out, '\r')
		case 't':
			out = append(out, '\t')
		case 'u':
			if len(p.raw)-p.off < 4 {
				return stdlibrt.BytesView{}, ErrInvalidViewJSON
			}
			value, ok := parseHex4(p.raw[p.off : p.off+4])
			if !ok || value > 0x7f {
				return stdlibrt.BytesView{}, ErrInvalidViewJSON
			}
			out = append(out, byte(value))
			p.off += 4
		default:
			return stdlibrt.BytesView{}, ErrInvalidViewJSON
		}
	}
	return stdlibrt.BytesView{}, ErrInvalidViewJSON
}

func (p *viewParser) copyString(decoded []byte) (stdlibrt.BytesView, error) {
	storage := stdlibrt.StorageHeap
	regionID := ""
	copied := make([]byte, len(decoded))
	if p.opts.Region != nil {
		var err error
		copied, err = p.opts.Region.Alloc(len(decoded))
		if err != nil {
			return stdlibrt.BytesView{}, err
		}
		storage = stdlibrt.StorageRegion
		regionID = p.opts.Region.ID()
		p.report.RegionTemporaries++
	} else {
		p.report.HeapAllocations++
	}
	copy(copied, decoded)
	p.report.CopiedStrings++
	return stdlibrt.BytesView{
		Bytes:      copied,
		Storage:    storage,
		RegionID:   regionID,
		Provenance: "json.unescaped",
		Copied:     true,
	}, nil
}

func (p *viewParser) parseNumber() ([]byte, error) {
	start := p.off
	if p.off < len(p.raw) && p.raw[p.off] == '-' {
		p.off++
	}
	if p.off >= len(p.raw) || p.raw[p.off] < '0' || p.raw[p.off] > '9' {
		return nil, ErrInvalidViewJSON
	}
	if p.raw[p.off] == '0' {
		p.off++
	} else {
		for p.off < len(p.raw) && p.raw[p.off] >= '0' && p.raw[p.off] <= '9' {
			p.off++
		}
	}
	if p.off < len(p.raw) && p.raw[p.off] == '.' {
		p.off++
		if p.off >= len(p.raw) || p.raw[p.off] < '0' || p.raw[p.off] > '9' {
			return nil, ErrInvalidViewJSON
		}
		for p.off < len(p.raw) && p.raw[p.off] >= '0' && p.raw[p.off] <= '9' {
			p.off++
		}
	}
	if p.off < len(p.raw) && (p.raw[p.off] == 'e' || p.raw[p.off] == 'E') {
		p.off++
		if p.off < len(p.raw) && (p.raw[p.off] == '+' || p.raw[p.off] == '-') {
			p.off++
		}
		if p.off >= len(p.raw) || p.raw[p.off] < '0' || p.raw[p.off] > '9' {
			return nil, ErrInvalidViewJSON
		}
		for p.off < len(p.raw) && p.raw[p.off] >= '0' && p.raw[p.off] <= '9' {
			p.off++
		}
	}
	return p.raw[start:p.off], nil
}

func (p *viewParser) consumeLiteral(lit string) bool {
	if len(p.raw)-p.off < len(lit) {
		return false
	}
	for i := 0; i < len(lit); i++ {
		if p.raw[p.off+i] != lit[i] {
			return false
		}
	}
	p.off += len(lit)
	return true
}

func (p *viewParser) skipSpace() {
	for p.off < len(p.raw) {
		switch p.raw[p.off] {
		case ' ', '\n', '\r', '\t':
			p.off++
		default:
			return
		}
	}
}

func parseHex4(raw []byte) (int, bool) {
	if len(raw) != 4 {
		return 0, false
	}
	value := 0
	for _, c := range raw {
		var digit int
		switch {
		case c >= '0' && c <= '9':
			digit = int(c - '0')
		case c >= 'a' && c <= 'f':
			digit = int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			digit = int(c-'A') + 10
		default:
			return 0, false
		}
		value = value*16 + digit
	}
	return value, true
}
