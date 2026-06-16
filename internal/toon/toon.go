package toon

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	CodeInvalidScalar       = "TOON_PARSE_INVALID_SCALAR"
	CodeInvalidUTF8         = "TOON_PARSE_INVALID_UTF8"
	CodeMultipleTopLevel    = "TOON_PARSE_MULTIPLE_TOP_LEVEL"
	CodeDuplicateKey        = "TOON_PARSE_DUPLICATE_KEY"
	CodeBadArrayLength      = "TOON_PARSE_BAD_ARRAY_LENGTH"
	CodeRowCountMismatch    = "TOON_PARSE_ROW_COUNT_MISMATCH"
	CodeColumnCountMismatch = "TOON_PARSE_COLUMN_COUNT_MISMATCH"
	CodeUnsupportedValue    = "TOON_ENCODE_UNSUPPORTED_VALUE"
	CodeNonFiniteNumber     = "TOON_ENCODE_NONFINITE_NUMBER"
	CodeLimitDepth          = "TOON_LIMIT_DEPTH"
	CodeLimitObjectKeys     = "TOON_LIMIT_SIZE"
	CodeMalformedDocument   = "TOON_PARSE_INDENT"
)

type Options struct {
	Deterministic         bool
	Strict                bool
	PreserveNumberLexemes bool
	MaxDepth              int
	MaxIndent             int
	MaxArrayLen           int
	MaxObjectKeys         int
	AllowTopLevelScalar   bool
}

type Error struct {
	Code    string
	Line    int
	Column  int
	Message string
}

func (e *Error) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s at %d:%d: %s", e.Code, e.Line, e.Column, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

type kind int

const (
	nullKind kind = iota
	boolKind
	numberKind
	stringKind
	arrayKind
	objectKind
)

type member struct {
	name  string
	value value
}

type value struct {
	kind   kind
	bool   bool
	number string
	string string
	array  []value
	object []member
}

func Marshal(v any) ([]byte, error) {
	return MarshalIndent(v, Options{Deterministic: true, Strict: true})
}

func MarshalIndent(v any, opts Options) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		if isNonFinite(v) {
			return nil, toonError(CodeNonFiniteNumber, 0, 0, "non-finite number is not supported")
		}
		return nil, fmt.Errorf("%s: %w", CodeUnsupportedValue, err)
	}
	return ConvertJSONToTOON(raw, opts)
}

func Unmarshal(data []byte, v any) error {
	raw, err := ConvertTOONToJSON(data, Options{Strict: true, AllowTopLevelScalar: true})
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, v)
}

func EncodeJSONValue(input any, opts Options) ([]byte, error) {
	normalized, err := normalizeJSONLike(input)
	if err != nil {
		return nil, err
	}
	lines, err := encodeValue(normalized, "", 0, defaultOptions(opts))
	if err != nil {
		return nil, err
	}
	return []byte(strings.Join(lines, "\n")), nil
}

func DecodeToJSONValue(data []byte, opts Options) (any, error) {
	decoded, err := parseTOON(data, defaultOptions(opts))
	if err != nil {
		return nil, err
	}
	return decoded.toInterface(), nil
}

func ConvertJSONToTOON(jsonData []byte, opts Options) ([]byte, error) {
	decoded, err := parseJSONValue(jsonData)
	if err != nil {
		return nil, err
	}
	return EncodeJSONValue(decoded, opts)
}

func ConvertTOONToJSON(toonData []byte, opts Options) ([]byte, error) {
	decoded, err := parseTOON(toonData, defaultOptions(opts))
	if err != nil {
		return nil, err
	}
	var dst []byte
	dst, err = appendJSON(dst, decoded)
	if err != nil {
		return nil, err
	}
	return dst, nil
}

func defaultOptions(opts Options) Options {
	if opts.MaxDepth == 0 {
		opts.MaxDepth = 256
	}
	if opts.MaxObjectKeys == 0 {
		opts.MaxObjectKeys = 100000
	}
	opts.AllowTopLevelScalar = true
	return opts
}

func parseJSONValue(raw []byte) (value, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return value{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return value{}, toonError(CodeUnsupportedValue, 0, 0, "multiple JSON values")
		}
		return value{}, err
	}
	return valueFromInterface(decoded)
}

func normalizeJSONLike(input any) (value, error) {
	if converted, ok := input.(value); ok {
		return converted, nil
	}
	raw, err := json.Marshal(input)
	if err != nil {
		if isNonFinite(input) {
			return value{}, toonError(CodeNonFiniteNumber, 0, 0, "non-finite number is not supported")
		}
		return value{}, fmt.Errorf("%s: %w", CodeUnsupportedValue, err)
	}
	return parseJSONValue(raw)
}

func valueFromInterface(input any) (value, error) {
	switch v := input.(type) {
	case nil:
		return value{kind: nullKind}, nil
	case bool:
		return value{kind: boolKind, bool: v}, nil
	case string:
		return value{kind: stringKind, string: v}, nil
	case json.Number:
		if !validJSONNumber(v.String()) {
			return value{}, toonError(CodeInvalidScalar, 0, 0, "invalid JSON number")
		}
		return value{kind: numberKind, number: v.String()}, nil
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return value{}, toonError(CodeNonFiniteNumber, 0, 0, "non-finite number is not supported")
		}
		return value{kind: numberKind, number: strconv.FormatFloat(v, 'g', -1, 64)}, nil
	case []any:
		out := make([]value, 0, len(v))
		for _, item := range v {
			converted, err := valueFromInterface(item)
			if err != nil {
				return value{}, err
			}
			out = append(out, converted)
		}
		return value{kind: arrayKind, array: out}, nil
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		members := make([]member, 0, len(keys))
		for _, key := range keys {
			converted, err := valueFromInterface(v[key])
			if err != nil {
				return value{}, err
			}
			members = append(members, member{name: key, value: converted})
		}
		return value{kind: objectKind, object: members}, nil
	default:
		return value{}, toonError(CodeUnsupportedValue, 0, 0, fmt.Sprintf("unsupported JSON value %T", input))
	}
}

func encodeValue(v value, key string, depth int, opts Options) ([]string, error) {
	if depth > opts.MaxDepth {
		return nil, toonError(CodeLimitDepth, 0, 0, "maximum TOON depth exceeded")
	}
	indent := strings.Repeat("  ", depth)
	prefix := ""
	if key != "" {
		prefix = encodeKey(key) + ": "
	}
	switch v.kind {
	case nullKind, boolKind, numberKind, stringKind:
		if key == "" {
			return []string{encodePrimitive(v, ',')}, nil
		}
		return []string{indent + prefix + encodePrimitive(v, ',')}, nil
	case objectKind:
		if len(v.object) > opts.MaxObjectKeys {
			return nil, toonError(CodeLimitObjectKeys, 0, 0, "maximum object keys exceeded")
		}
		if key != "" {
			lines := []string{indent + encodeKey(key) + ":"}
			for _, member := range v.object {
				child, err := encodeValue(member.value, member.name, depth+1, opts)
				if err != nil {
					return nil, err
				}
				lines = append(lines, child...)
			}
			return lines, nil
		}
		lines := make([]string, 0, len(v.object))
		for _, member := range v.object {
			child, err := encodeValue(member.value, member.name, depth, opts)
			if err != nil {
				return nil, err
			}
			lines = append(lines, child...)
		}
		return lines, nil
	case arrayKind:
		return encodeArray(v, key, depth, opts)
	default:
		return nil, toonError(CodeUnsupportedValue, 0, 0, "unsupported value kind")
	}
}

func encodeArray(v value, key string, depth int, opts Options) ([]string, error) {
	indent := strings.Repeat("  ", depth)
	if len(v.array) == 0 {
		if key == "" {
			return []string{"[]"}, nil
		}
		return []string{indent + encodeKey(key) + ": []"}, nil
	}
	if fields, ok := tabularFields(v); ok {
		headerKey := ""
		if key != "" {
			headerKey = encodeKey(key)
		}
		lines := []string{fmt.Sprintf("%s%s[%d]{%s}:", indent, headerKey, len(v.array), strings.Join(encodeKeys(fields), ","))}
		rowIndent := strings.Repeat("  ", depth+1)
		for _, item := range v.array {
			cells := make([]string, 0, len(fields))
			for _, field := range fields {
				cells = append(cells, encodePrimitive(memberValue(item, field), ','))
			}
			lines = append(lines, rowIndent+strings.Join(cells, ","))
		}
		return lines, nil
	}
	for _, item := range v.array {
		if !isPrimitive(item) {
			return encodeExpandedArray(v, key, depth, opts)
		}
	}
	parts := make([]string, 0, len(v.array))
	for _, item := range v.array {
		parts = append(parts, encodePrimitive(item, ','))
	}
	if key == "" {
		return []string{fmt.Sprintf("[%d]: %s", len(v.array), strings.Join(parts, ","))}, nil
	}
	return []string{fmt.Sprintf("%s%s[%d]: %s", indent, encodeKey(key), len(v.array), strings.Join(parts, ","))}, nil
}

func encodeExpandedArray(v value, key string, depth int, opts Options) ([]string, error) {
	indent := strings.Repeat("  ", depth)
	headerKey := ""
	if key != "" {
		headerKey = encodeKey(key)
	}
	lines := []string{fmt.Sprintf("%s%s[%d]:", indent, headerKey, len(v.array))}
	for _, item := range v.array {
		itemLines, err := encodeListItem(item, depth+1, opts)
		if err != nil {
			return nil, err
		}
		lines = append(lines, itemLines...)
	}
	return lines, nil
}

func encodeListItem(v value, depth int, opts Options) ([]string, error) {
	indent := strings.Repeat("  ", depth)
	switch v.kind {
	case nullKind, boolKind, numberKind, stringKind:
		return []string{indent + "- " + encodePrimitive(v, ',')}, nil
	case objectKind:
		if len(v.object) == 0 {
			return []string{indent + "-"}, nil
		}
		lines := []string{indent + "-"}
		for _, member := range v.object {
			child, err := encodeValue(member.value, member.name, depth+1, opts)
			if err != nil {
				return nil, err
			}
			lines = append(lines, child...)
		}
		return lines, nil
	case arrayKind:
		if len(v.array) == 0 {
			return []string{indent + "- [0]:"}, nil
		}
		allPrimitive := true
		for _, item := range v.array {
			if !isPrimitive(item) {
				allPrimitive = false
				break
			}
		}
		if allPrimitive {
			parts := make([]string, 0, len(v.array))
			for _, item := range v.array {
				parts = append(parts, encodePrimitive(item, ','))
			}
			return []string{fmt.Sprintf("%s- [%d]: %s", indent, len(v.array), strings.Join(parts, ","))}, nil
		}
		lines := []string{fmt.Sprintf("%s- [%d]:", indent, len(v.array))}
		for _, item := range v.array {
			child, err := encodeListItem(item, depth+1, opts)
			if err != nil {
				return nil, err
			}
			lines = append(lines, child...)
		}
		return lines, nil
	default:
		return nil, toonError(CodeUnsupportedValue, 0, 0, "unsupported list item")
	}
}

func tabularFields(v value) ([]string, bool) {
	if len(v.array) == 0 {
		return nil, false
	}
	first := v.array[0]
	if first.kind != objectKind || len(first.object) == 0 {
		return nil, false
	}
	fields := make([]string, 0, len(first.object))
	for _, item := range first.object {
		if !isPrimitive(item.value) {
			return nil, false
		}
		fields = append(fields, item.name)
	}
	for _, item := range v.array[1:] {
		if item.kind != objectKind || len(item.object) != len(fields) {
			return nil, false
		}
		for _, field := range fields {
			if !isPrimitive(memberValue(item, field)) {
				return nil, false
			}
		}
	}
	return fields, true
}

func memberValue(v value, name string) value {
	for _, item := range v.object {
		if item.name == name {
			return item.value
		}
	}
	return value{kind: objectKind}
}

func encodeKeys(keys []string) []string {
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, encodeKey(key))
	}
	return out
}

func encodePrimitive(v value, delimiter rune) string {
	switch v.kind {
	case nullKind:
		return "null"
	case boolKind:
		if v.bool {
			return "true"
		}
		return "false"
	case numberKind:
		return v.number
	case stringKind:
		return encodeStringValue(v.string, delimiter)
	default:
		return ""
	}
}

func isPrimitive(v value) bool {
	return v.kind == nullKind || v.kind == boolKind || v.kind == numberKind || v.kind == stringKind
}

func parseTOON(data []byte, opts Options) (value, error) {
	if !utf8.Valid(data) {
		return value{}, toonError(CodeInvalidUTF8, 1, 1, "input is not valid UTF-8")
	}
	text := strings.TrimRight(string(data), "\n")
	if text == "" {
		return value{kind: objectKind}, nil
	}
	lines, err := collectLines(text, opts)
	if err != nil {
		return value{}, err
	}
	if len(lines) == 0 {
		return value{kind: objectKind}, nil
	}
	if len(lines) == 1 && !strings.Contains(lines[0].text, ":") && !strings.HasPrefix(lines[0].text, "-") {
		if !opts.AllowTopLevelScalar {
			return value{}, toonError(CodeMultipleTopLevel, lines[0].num, 1, "top-level scalar is not allowed")
		}
		return parsePrimitive(lines[0].text, lines[0].num, 1)
	}
	parsed, next, err := parseValueBlock(lines, 0, lines[0].depth, opts)
	if err != nil {
		return value{}, err
	}
	if next != len(lines) {
		return value{}, toonError(CodeMultipleTopLevel, lines[next].num, 1, "multiple top-level values")
	}
	return parsed, nil
}

type toonLine struct {
	num   int
	depth int
	text  string
}

func collectLines(text string, opts Options) ([]toonLine, error) {
	rawLines := strings.Split(text, "\n")
	lines := make([]toonLine, 0, len(rawLines))
	for i, raw := range rawLines {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		if strings.Contains(raw, "\t") {
			return nil, toonError(CodeMalformedDocument, i+1, 1, "tabs are not valid indentation")
		}
		spaces := len(raw) - len(strings.TrimLeft(raw, " "))
		if spaces%2 != 0 {
			return nil, toonError(CodeMalformedDocument, i+1, 1, "indentation must use two-space steps")
		}
		if opts.MaxIndent > 0 && spaces > opts.MaxIndent {
			return nil, toonError(CodeLimitDepth, i+1, 1, "maximum indentation exceeded")
		}
		lines = append(lines, toonLine{num: i + 1, depth: spaces / 2, text: strings.TrimSpace(raw)})
	}
	return lines, nil
}

func parseValueBlock(lines []toonLine, index int, depth int, opts Options) (value, int, error) {
	if index >= len(lines) {
		return value{kind: objectKind}, index, nil
	}
	if lines[index].depth != depth {
		return value{}, index, toonError(CodeMalformedDocument, lines[index].num, 1, "unexpected indentation")
	}
	if strings.HasPrefix(lines[index].text, "-") {
		return parseExpandedArray(lines, index, depth, -1, opts)
	}
	if isAnonymousArrayHeader(lines[index].text) {
		return parseArrayLine(lines, index, depth, opts)
	}
	return parseObjectBlock(lines, index, depth, opts)
}

func parseObjectBlock(lines []toonLine, index int, depth int, opts Options) (value, int, error) {
	members := []member{}
	seen := map[string]struct{}{}
	for index < len(lines) && lines[index].depth == depth {
		line := lines[index]
		if strings.HasPrefix(line.text, "-") {
			break
		}
		key, parsedValue, next, err := parseObjectEntry(lines, index, depth, opts)
		if err != nil {
			return value{}, index, err
		}
		if _, ok := seen[key]; ok {
			return value{}, index, toonError(CodeDuplicateKey, line.num, 1, "duplicate key "+key)
		}
		seen[key] = struct{}{}
		members = append(members, member{name: key, value: parsedValue})
		index = next
	}
	if len(members) > opts.MaxObjectKeys {
		return value{}, index, toonError(CodeLimitObjectKeys, 0, 0, "maximum object keys exceeded")
	}
	return value{kind: objectKind, object: members}, index, nil
}

func parseObjectEntry(lines []toonLine, index int, depth int, opts Options) (string, value, int, error) {
	line := lines[index]
	keyPart, valuePart, ok := strings.Cut(line.text, ":")
	if !ok {
		return "", value{}, index, toonError(CodeMultipleTopLevel, line.num, 1, "expected key-value line")
	}
	if baseKey, arraySpec, ok := parseKeyArraySpec(keyPart); ok {
		key, err := decodeKey(baseKey, line.num, 1)
		if err != nil {
			return "", value{}, index, err
		}
		parsed, next, err := parseArraySpecValue(lines, index, depth, strings.TrimSpace(valuePart), arraySpec, opts)
		return key, parsed, next, err
	}
	key, err := decodeKey(keyPart, line.num, 1)
	if err != nil {
		return "", value{}, index, err
	}
	rawValue := strings.TrimSpace(valuePart)
	if rawValue == "" {
		if index+1 >= len(lines) || lines[index+1].depth <= depth {
			return key, value{kind: objectKind}, index + 1, nil
		}
		parsed, next, err := parseValueBlock(lines, index+1, depth+1, opts)
		return key, parsed, next, err
	}
	if rawValue == "[]" {
		return key, value{kind: arrayKind}, index + 1, nil
	}
	parsed, err := parsePrimitive(rawValue, line.num, strings.Index(line.text, rawValue)+1)
	return key, parsed, index + 1, err
}

type arraySpec struct {
	count  int
	fields []string
}

func parseKeyArraySpec(keyPart string) (string, arraySpec, bool) {
	open := strings.LastIndex(keyPart, "[")
	close := strings.Index(keyPart[open+1:], "]")
	if open < 0 || close < 0 {
		return "", arraySpec{}, false
	}
	close += open + 1
	count, err := strconv.Atoi(keyPart[open+1 : close])
	if err != nil || count < 0 {
		return "", arraySpec{}, false
	}
	spec := arraySpec{count: count}
	rest := keyPart[close+1:]
	if rest != "" {
		if !strings.HasPrefix(rest, "{") || !strings.HasSuffix(rest, "}") {
			return "", arraySpec{}, false
		}
		fieldsRaw := strings.TrimSuffix(strings.TrimPrefix(rest, "{"), "}")
		if fieldsRaw != "" {
			spec.fields = strings.Split(fieldsRaw, ",")
		}
	}
	return keyPart[:open], spec, true
}

func parseArraySpecValue(lines []toonLine, index int, depth int, inline string, spec arraySpec, opts Options) (value, int, error) {
	line := lines[index]
	if len(spec.fields) > 0 {
		if inline != "" {
			return value{}, index, toonError(CodeInvalidScalar, line.num, 1, "tabular arrays cannot have inline rows")
		}
		return parseTabularRows(lines, index+1, depth+1, spec)
	}
	if inline != "" {
		if spec.count == 0 && inline == "" {
			return value{kind: arrayKind}, index + 1, nil
		}
		cells, err := splitCSV(inline, line.num)
		if err != nil {
			return value{}, index, err
		}
		if len(cells) != spec.count {
			return value{}, index, toonError(CodeBadArrayLength, line.num, 1, "array length does not match header")
		}
		items := make([]value, 0, len(cells))
		for _, cell := range cells {
			parsed, err := parsePrimitive(cell, line.num, 1)
			if err != nil {
				return value{}, index, err
			}
			items = append(items, parsed)
		}
		return value{kind: arrayKind, array: items}, index + 1, nil
	}
	if spec.count == 0 {
		return value{kind: arrayKind}, index + 1, nil
	}
	return parseExpandedArray(lines, index+1, depth+1, spec.count, opts)
}

func parseArrayLine(lines []toonLine, index int, depth int, opts Options) (value, int, error) {
	line := lines[index]
	header, inline, ok := strings.Cut(line.text, ":")
	if !ok {
		return value{}, index, toonError(CodeInvalidScalar, line.num, 1, "invalid array header")
	}
	_, spec, ok := parseKeyArraySpec(header)
	if !ok {
		return value{}, index, toonError(CodeInvalidScalar, line.num, 1, "invalid array header")
	}
	return parseArraySpecValue(lines, index, depth, strings.TrimSpace(inline), spec, opts)
}

func parseTabularRows(lines []toonLine, index int, depth int, spec arraySpec) (value, int, error) {
	rows := []value{}
	for index < len(lines) && lines[index].depth == depth {
		line := lines[index]
		if strings.HasPrefix(line.text, "-") {
			break
		}
		cells, err := splitCSV(line.text, line.num)
		if err != nil {
			return value{}, index, err
		}
		if len(cells) != len(spec.fields) {
			return value{}, index, toonError(CodeColumnCountMismatch, line.num, 1, "tabular row column count does not match header")
		}
		members := make([]member, 0, len(spec.fields))
		for i, field := range spec.fields {
			key, err := decodeKey(field, line.num, 1)
			if err != nil {
				return value{}, index, err
			}
			parsed, err := parsePrimitive(cells[i], line.num, 1)
			if err != nil {
				return value{}, index, err
			}
			members = append(members, member{name: key, value: parsed})
		}
		rows = append(rows, value{kind: objectKind, object: members})
		index++
	}
	if len(rows) != spec.count {
		lineNum := 1
		if index > 0 && index-1 < len(lines) {
			lineNum = lines[index-1].num
		}
		return value{}, index, toonError(CodeRowCountMismatch, lineNum, 1, "tabular row count does not match header")
	}
	return value{kind: arrayKind, array: rows}, index, nil
}

func parseExpandedArray(lines []toonLine, index int, depth int, wantCount int, opts Options) (value, int, error) {
	items := []value{}
	for index < len(lines) && lines[index].depth == depth {
		if !strings.HasPrefix(lines[index].text, "-") {
			break
		}
		item, next, err := parseListItem(lines, index, depth, opts)
		if err != nil {
			return value{}, index, err
		}
		items = append(items, item)
		index = next
	}
	if wantCount >= 0 && len(items) != wantCount {
		lineNum := 1
		if index > 0 && index-1 < len(lines) {
			lineNum = lines[index-1].num
		}
		return value{}, index, toonError(CodeBadArrayLength, lineNum, 1, "array item count does not match header")
	}
	return value{kind: arrayKind, array: items}, index, nil
}

func parseListItem(lines []toonLine, index int, depth int, opts Options) (value, int, error) {
	line := lines[index]
	content := strings.TrimSpace(strings.TrimPrefix(line.text, "-"))
	if content == "" {
		if index+1 < len(lines) && lines[index+1].depth > depth {
			return parseValueBlock(lines, index+1, depth+1, opts)
		}
		return value{kind: objectKind}, index + 1, nil
	}
	if isArrayHeader(content) {
		itemLines := append([]toonLine(nil), lines...)
		itemLines[index].text = content
		return parseArrayLine(itemLines, index, depth, opts)
	}
	if strings.Contains(content, ":") {
		keyPart, valuePart, _ := strings.Cut(content, ":")
		key, err := decodeKey(keyPart, line.num, 1)
		if err != nil {
			return value{}, index, err
		}
		rawValue := strings.TrimSpace(valuePart)
		var first value
		if rawValue == "" {
			first = value{kind: objectKind}
		} else {
			first, err = parsePrimitive(rawValue, line.num, 1)
			if err != nil {
				return value{}, index, err
			}
		}
		members := []member{{name: key, value: first}}
		next := index + 1
		if next < len(lines) && lines[next].depth == depth+1 {
			rest, after, err := parseObjectBlock(lines, next, depth+1, opts)
			if err != nil {
				return value{}, index, err
			}
			members = append(members, rest.object...)
			next = after
		}
		return value{kind: objectKind, object: members}, next, nil
	}
	parsed, err := parsePrimitive(content, line.num, 1)
	return parsed, index + 1, err
}

func isArrayHeader(text string) bool {
	header, _, ok := strings.Cut(text, ":")
	if !ok {
		return false
	}
	_, _, ok = parseKeyArraySpec(header)
	return ok
}

func isAnonymousArrayHeader(text string) bool {
	header, _, ok := strings.Cut(text, ":")
	if !ok {
		return false
	}
	key, _, ok := parseKeyArraySpec(header)
	return ok && key == ""
}

func splitCSV(raw string, line int) ([]string, error) {
	if raw == "" {
		return nil, nil
	}
	var cells []string
	var current strings.Builder
	inQuote := false
	escaped := false
	for _, r := range raw {
		switch {
		case escaped:
			current.WriteRune('\\')
			current.WriteRune(r)
			escaped = false
		case r == '\\' && inQuote:
			escaped = true
		case r == '"':
			inQuote = !inQuote
			current.WriteRune(r)
		case r == ',' && !inQuote:
			cells = append(cells, strings.TrimSpace(current.String()))
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	if escaped || inQuote {
		return nil, toonError(CodeInvalidScalar, line, 1, "unterminated quoted string")
	}
	cells = append(cells, strings.TrimSpace(current.String()))
	return cells, nil
}

func parsePrimitive(token string, line int, column int) (value, error) {
	if token == "null" {
		return value{kind: nullKind}, nil
	}
	if token == "true" {
		return value{kind: boolKind, bool: true}, nil
	}
	if token == "false" {
		return value{kind: boolKind}, nil
	}
	if len(token) >= 2 && token[0] == '"' && token[len(token)-1] == '"' {
		unquoted, err := strconv.Unquote(token)
		if err == nil {
			return value{kind: stringKind, string: unquoted}, nil
		}
		return value{}, toonError(CodeInvalidScalar, line, column, "invalid quoted string")
	}
	if validJSONNumber(token) {
		return value{kind: numberKind, number: token}, nil
	}
	if token == "" {
		return value{}, toonError(CodeInvalidScalar, line, column, "empty scalar")
	}
	return value{kind: stringKind, string: token}, nil
}

func appendJSON(dst []byte, v value) ([]byte, error) {
	switch v.kind {
	case nullKind:
		return append(dst, "null"...), nil
	case boolKind:
		if v.bool {
			return append(dst, "true"...), nil
		}
		return append(dst, "false"...), nil
	case numberKind:
		if !validJSONNumber(v.number) {
			return nil, toonError(CodeInvalidScalar, 0, 0, "invalid JSON number")
		}
		return append(dst, v.number...), nil
	case stringKind:
		raw, err := json.Marshal(v.string)
		if err != nil {
			return nil, err
		}
		return append(dst, raw...), nil
	case arrayKind:
		dst = append(dst, '[')
		for i, item := range v.array {
			if i > 0 {
				dst = append(dst, ',')
			}
			var err error
			dst, err = appendJSON(dst, item)
			if err != nil {
				return nil, err
			}
		}
		return append(dst, ']'), nil
	case objectKind:
		members := append([]member(nil), v.object...)
		sort.SliceStable(members, func(i, j int) bool {
			return members[i].name < members[j].name
		})
		dst = append(dst, '{')
		for i, member := range members {
			if i > 0 {
				dst = append(dst, ',')
			}
			rawKey, err := json.Marshal(member.name)
			if err != nil {
				return nil, err
			}
			dst = append(dst, rawKey...)
			dst = append(dst, ':')
			dst, err = appendJSON(dst, member.value)
			if err != nil {
				return nil, err
			}
		}
		return append(dst, '}'), nil
	default:
		return nil, toonError(CodeUnsupportedValue, 0, 0, "unsupported value kind")
	}
}

func (v value) toInterface() any {
	switch v.kind {
	case nullKind:
		return nil
	case boolKind:
		return v.bool
	case numberKind:
		return json.Number(v.number)
	case stringKind:
		return v.string
	case arrayKind:
		out := make([]any, 0, len(v.array))
		for _, item := range v.array {
			out = append(out, item.toInterface())
		}
		return out
	case objectKind:
		out := make(map[string]any, len(v.object))
		for _, member := range v.object {
			out[member.name] = member.value.toInterface()
		}
		return out
	default:
		return nil
	}
}

func encodeKey(key string) string {
	if isBareKey(key) {
		return key
	}
	return quoteTOONString(key)
}

func decodeKey(token string, line int, column int) (string, error) {
	if len(token) >= 2 && token[0] == '"' && token[len(token)-1] == '"' {
		unquoted, err := strconv.Unquote(token)
		if err != nil {
			return "", toonError(CodeInvalidScalar, line, column, "invalid quoted key")
		}
		return unquoted, nil
	}
	if !isBareKey(token) {
		return "", toonError(CodeInvalidScalar, line, column, "invalid unquoted key")
	}
	return token, nil
}

func isBareKey(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !(r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z') {
				return false
			}
			continue
		}
		if !(r == '_' || r == '.' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func encodeStringValue(s string, delimiter rune) string {
	if mustQuoteString(s, delimiter) {
		return quoteTOONString(s)
	}
	return s
}

func quoteTOONString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				b.WriteString(fmt.Sprintf(`\u%04x`, r))
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

func mustQuoteString(s string, delimiter rune) bool {
	if s == "" || strings.TrimSpace(s) != s {
		return true
	}
	if s == "true" || s == "false" || s == "null" {
		return true
	}
	if validJSONNumber(s) {
		return true
	}
	if s == "-" || strings.HasPrefix(s, "-") {
		return true
	}
	for _, r := range s {
		if r == delimiter || r == ':' || r == '"' || r == '\\' || r == '[' || r == ']' || r == '{' || r == '}' || r < 0x20 {
			return true
		}
	}
	return false
}

func validJSONNumber(number string) bool {
	if number == "" {
		return false
	}
	decoder := json.NewDecoder(strings.NewReader(number))
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

func isNonFinite(v any) bool {
	rv := reflect.ValueOf(v)
	return containsNonFinite(rv)
}

func containsNonFinite(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}
	switch v.Kind() {
	case reflect.Interface, reflect.Pointer:
		if v.IsNil() {
			return false
		}
		return containsNonFinite(v.Elem())
	case reflect.Float32, reflect.Float64:
		f := v.Float()
		return math.IsNaN(f) || math.IsInf(f, 0)
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if containsNonFinite(v.Index(i)) {
				return true
			}
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			if containsNonFinite(v.MapIndex(key)) {
				return true
			}
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if containsNonFinite(v.Field(i)) {
				return true
			}
		}
	}
	return false
}

func toonError(code string, line int, column int, message string) error {
	return &Error{Code: code, Line: line, Column: column, Message: message}
}
