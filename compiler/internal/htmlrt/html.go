package htmlrt

import (
	"sort"
	"strconv"
)

type Fortune struct {
	ID      int
	Message string
}

func AppendEscaped(dst []byte, value string) []byte {
	for i := 0; i < len(value); i++ {
		switch value[i] {
		case '&':
			dst = append(dst, "&amp;"...)
		case '<':
			dst = append(dst, "&lt;"...)
		case '>':
			dst = append(dst, "&gt;"...)
		case '"':
			dst = append(dst, "&quot;"...)
		case '\'':
			dst = append(dst, "&apos;"...)
		default:
			dst = append(dst, value[i])
		}
	}
	return dst
}

func RenderFortunes(dst []byte, fortunes []Fortune) []byte {
	sorted := append([]Fortune(nil), fortunes...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Message < sorted[j].Message
	})

	dst = append(dst, "<!DOCTYPE html><html><head><title>Fortunes</title></head><body><table>"...)
	dst = append(dst, "<tr><th>id</th><th>message</th></tr>"...)
	for _, fortune := range sorted {
		dst = append(dst, "<tr><td>"...)
		dst = strconv.AppendInt(dst, int64(fortune.ID), 10)
		dst = append(dst, "</td><td>"...)
		dst = AppendEscaped(dst, fortune.Message)
		dst = append(dst, "</td></tr>"...)
	}
	dst = append(dst, "</table></body></html>"...)
	return dst
}
