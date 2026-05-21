package jsonrt

import (
	"strconv"
	"unicode/utf8"
)

const hex = "0123456789abcdef"

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
