package jsonrt

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strconv"
	"unicode/utf8"
)

const hex = "0123456789abcdef"

var (
	ErrInvalidJSONNumber = errors.New("invalid JSON number")
	ErrUnsupportedValue  = errors.New("unsupported JSON value")
)

type Kind int

const (
	NullKind Kind = iota
	BoolKind
	NumberKind
	StringKind
	ArrayKind
	ObjectKind
)

type Member struct {
	Name  string
	Value Value
}

type Value struct {
	Kind   Kind
	Bool   bool
	Number string
	String string
	Array  []Value
	Object []Member
}

func AppendMessageObject(dst []byte, message string) []byte {
	dst = append(dst, `{"message":`...)
	dst = AppendString(dst, message)
	dst = append(dst, '}')
	return dst
}

type World struct {
	ID           int
	RandomNumber int
}

func AppendWorldObject(dst []byte, id int, randomNumber int) []byte {
	dst = append(dst, `{"id":`...)
	dst = strconv.AppendInt(dst, int64(id), 10)
	dst = append(dst, `,"randomNumber":`...)
	dst = strconv.AppendInt(dst, int64(randomNumber), 10)
	dst = append(dst, '}')
	return dst
}

func AppendWorldArray(dst []byte, worlds []World) []byte {
	dst = append(dst, '[')
	for i, world := range worlds {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = AppendWorldObject(dst, world.ID, world.RandomNumber)
	}
	dst = append(dst, ']')
	return dst
}

func ParseValue(raw []byte) (Value, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return Value{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return Value{}, ErrUnsupportedValue
		}
		return Value{}, err
	}
	return valueFromDecoded(decoded)
}

func AppendValue(dst []byte, value Value) ([]byte, error) {
	switch value.Kind {
	case NullKind:
		return append(dst, "null"...), nil
	case BoolKind:
		if value.Bool {
			return append(dst, "true"...), nil
		}
		return append(dst, "false"...), nil
	case NumberKind:
		if !validJSONNumber(value.Number) {
			return nil, ErrInvalidJSONNumber
		}
		return append(dst, value.Number...), nil
	case StringKind:
		return AppendString(dst, value.String), nil
	case ArrayKind:
		dst = append(dst, '[')
		for i, item := range value.Array {
			if i > 0 {
				dst = append(dst, ',')
			}
			var err error
			dst, err = AppendValue(dst, item)
			if err != nil {
				return nil, err
			}
		}
		dst = append(dst, ']')
		return dst, nil
	case ObjectKind:
		members := append([]Member(nil), value.Object...)
		sort.SliceStable(members, func(i, j int) bool {
			return members[i].Name < members[j].Name
		})
		dst = append(dst, '{')
		for i, member := range members {
			if i > 0 {
				dst = append(dst, ',')
			}
			dst = AppendString(dst, member.Name)
			dst = append(dst, ':')
			var err error
			dst, err = AppendValue(dst, member.Value)
			if err != nil {
				return nil, err
			}
		}
		dst = append(dst, '}')
		return dst, nil
	default:
		return nil, ErrUnsupportedValue
	}
}

func valueFromDecoded(decoded any) (Value, error) {
	switch value := decoded.(type) {
	case nil:
		return Value{Kind: NullKind}, nil
	case bool:
		return Value{Kind: BoolKind, Bool: value}, nil
	case string:
		return Value{Kind: StringKind, String: value}, nil
	case json.Number:
		number := value.String()
		if !validJSONNumber(number) {
			return Value{}, ErrInvalidJSONNumber
		}
		return Value{Kind: NumberKind, Number: number}, nil
	case []any:
		items := make([]Value, 0, len(value))
		for _, item := range value {
			converted, err := valueFromDecoded(item)
			if err != nil {
				return Value{}, err
			}
			items = append(items, converted)
		}
		return Value{Kind: ArrayKind, Array: items}, nil
	case map[string]any:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		members := make([]Member, 0, len(keys))
		for _, key := range keys {
			converted, err := valueFromDecoded(value[key])
			if err != nil {
				return Value{}, err
			}
			members = append(members, Member{Name: key, Value: converted})
		}
		return Value{Kind: ObjectKind, Object: members}, nil
	default:
		return Value{}, ErrUnsupportedValue
	}
}

func validJSONNumber(number string) bool {
	if number == "" {
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader([]byte(number)))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return false
	}
	if _, ok := decoded.(json.Number); !ok {
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return false
	}
	return true
}

func AppendString(dst []byte, value string) []byte {
	dst = append(dst, '"')
	start := 0
	for i := 0; i < len(value); {
		c := value[i]
		if c < utf8.RuneSelf {
			if c >= 0x20 && c != '"' && c != '\\' {
				i++
				continue
			}
			dst = append(dst, value[start:i]...)
			dst = appendEscapedByte(dst, c)
			i++
			start = i
			continue
		}
		r, size := utf8.DecodeRuneInString(value[i:])
		if r == utf8.RuneError && size == 1 {
			dst = append(dst, value[start:i]...)
			dst = append(dst, `\ufffd`...)
			i++
			start = i
			continue
		}
		i += size
	}
	dst = append(dst, value[start:]...)
	dst = append(dst, '"')
	return dst
}

func appendEscapedByte(dst []byte, c byte) []byte {
	switch c {
	case '\\', '"':
		dst = append(dst, '\\', c)
	case '\b':
		dst = append(dst, `\b`...)
	case '\f':
		dst = append(dst, `\f`...)
	case '\n':
		dst = append(dst, `\n`...)
	case '\r':
		dst = append(dst, `\r`...)
	case '\t':
		dst = append(dst, `\t`...)
	default:
		dst = append(dst, `\u00`...)
		dst = append(dst, hex[c>>4], hex[c&0x0f])
	}
	return dst
}
