package httprt

import (
	"errors"

	"tetra_language/compiler/internal/stdlibrt"
)

type HeaderView struct {
	Name  []byte
	Value []byte
}

type RequestView struct {
	Method        []byte
	RequestTarget []byte
	Path          []byte
	Query         []byte
	Version       []byte
	Headers       []HeaderView
	ContentLength int
	Body          []byte
	KeepAlive     bool
}

func (r RequestView) Header(name string) []byte {
	for _, header := range r.Headers {
		if asciiEqualFoldBytesString(header.Name, name) {
			return header.Value
		}
	}
	return nil
}

type RequestParseReport struct {
	HeapAllocations     int
	HeaderViewsBorrowed int
	HeaderValuesCopied  int
	RequestStorage      stdlibrt.StorageClass
	RegionID            string
	UnsafeFacts         bool
}

type ResponseBufferOptions struct {
	Storage  stdlibrt.StorageClass
	RegionID string
}

type ResponseBufferReport struct {
	BytesWritten          int
	ResponseBufferStorage stdlibrt.StorageClass
	RegionID              string
	HeapAllocations       int
}

var ErrInvalidRequestRegionScope = errors.New("invalid request region scope")

type RequestRegionOptions struct {
	RegionID         string
	RegionCapacity   int
	HeaderCapacity   int
	ResponseCapacity int
}

type RequestRegionReport struct {
	RegionID             string
	Lifetime             string
	Request              RequestParseReport
	Response             ResponseBufferReport
	HeapAllocations      int
	BytesUsedBeforeReset int
	Reset                bool
}

type RequestRegionHandler func(RequestView, *stdlibrt.Region) (Response, error)
type RequestRegionWriter func([]byte) error

type RequestRegionScope struct {
	region           *stdlibrt.Region
	headers          []HeaderView
	responseCapacity int
}

func NewRequestRegionScope(opt RequestRegionOptions) *RequestRegionScope {
	regionID := opt.RegionID
	if regionID == "" {
		regionID = "request"
	}
	regionCapacity := opt.RegionCapacity
	if regionCapacity <= 0 {
		regionCapacity = 8192
	}
	headerCapacity := opt.HeaderCapacity
	if headerCapacity <= 0 {
		headerCapacity = 64
	}
	responseCapacity := opt.ResponseCapacity
	if responseCapacity <= 0 {
		responseCapacity = regionCapacity
	}
	return &RequestRegionScope{
		region:           stdlibrt.NewRegion(regionID, regionCapacity),
		headers:          make([]HeaderView, 0, headerCapacity),
		responseCapacity: responseCapacity,
	}
}

func (s *RequestRegionScope) RegionUsed() int {
	if s == nil || s.region == nil {
		return 0
	}
	return s.region.Used()
}

func (s *RequestRegionScope) Run(
	input []byte,
	limits Limits,
	handler RequestRegionHandler,
	write RequestRegionWriter,
) (consumed int, report RequestRegionReport, err error) {
	if s == nil || s.region == nil || handler == nil {
		return 0, RequestRegionReport{}, ErrInvalidRequestRegionScope
	}
	report.RegionID = s.region.ID()
	report.Lifetime = "request"
	defer func() {
		report.BytesUsedBeforeReset = s.region.Used()
		if resetErr := s.region.Reset(); resetErr != nil && err == nil {
			err = resetErr
		}
		report.Reset = true
		s.headers = s.headers[:0]
	}()

	req, parsed, requestReport, parseErr := ParseRequestViewInRegion(
		input,
		limits,
		s.headers[:0],
		s.region,
	)
	consumed = parsed
	report.Request = requestReport
	if parseErr != nil {
		err = parseErr
		return
	}
	s.headers = req.Headers
	resp, handlerErr := handler(req, s.region)
	if handlerErr != nil {
		err = handlerErr
		return
	}
	buf, allocErr := s.region.Alloc(s.responseCapacity)
	if allocErr != nil {
		err = allocErr
		return
	}
	out, responseReport := AppendResponseWithReport(buf[:0], resp, ResponseBufferOptions{
		Storage:  stdlibrt.StorageRegion,
		RegionID: s.region.ID(),
	})
	report.Response = responseReport
	report.HeapAllocations = report.Request.HeapAllocations + report.Response.HeapAllocations
	if responseReport.HeapAllocations != 0 {
		err = stdlibrt.ErrRegionCapacity
		return
	}
	if write != nil {
		err = write(out)
	}
	return
}

func AppendResponseWithReport(
	dst []byte,
	resp Response,
	opts ResponseBufferOptions,
) ([]byte, ResponseBufferReport) {
	before := len(dst)
	out := AppendResponse(dst, resp)
	storage := opts.Storage
	if storage == "" {
		storage = stdlibrt.StorageHeap
	}
	report := ResponseBufferReport{
		BytesWritten:          len(out) - before,
		ResponseBufferStorage: storage,
		RegionID:              opts.RegionID,
	}
	if storage == stdlibrt.StorageHeap && cap(dst) == 0 && cap(out) > 0 {
		report.HeapAllocations = 1
	}
	if storage == stdlibrt.StorageRegion && cap(out) > cap(dst) {
		report.HeapAllocations = 1
	}
	return out, report
}

func ParseRequestView(
	input []byte,
	limits Limits,
	headers []HeaderView,
) (RequestView, int, RequestParseReport, error) {
	return parseRequestView(input, limits, headers, nil)
}

func ParseRequestViewInRegion(
	input []byte,
	limits Limits,
	headers []HeaderView,
	region *stdlibrt.Region,
) (RequestView, int, RequestParseReport, error) {
	return parseRequestView(input, limits, headers, region)
}

func parseRequestView(
	input []byte,
	limits Limits,
	headers []HeaderView,
	region *stdlibrt.Region,
) (RequestView, int, RequestParseReport, error) {
	limits = normalizeLimits(limits)
	report := RequestParseReport{RequestStorage: stdlibrt.StorageBorrowed}
	if region != nil {
		report.RequestStorage = stdlibrt.StorageRegion
		report.RegionID = region.ID()
	}
	headerEnd := indexHeaderEnd(input)
	if headerEnd < 0 {
		if len(input) > limits.MaxHeaderBytes {
			return RequestView{}, 0, report, ErrHeaderTooLarge
		}
		return RequestView{}, 0, report, ErrIncomplete
	}
	headerBytes := headerEnd + len("\r\n\r\n")
	if headerBytes > limits.MaxHeaderBytes {
		return RequestView{}, 0, report, ErrHeaderTooLarge
	}
	lineEnd := indexCRLF(input[:headerEnd])
	if lineEnd < 0 {
		return RequestView{}, 0, report, ErrMalformedRequest
	}
	req, err := parseRequestLineView(input[:lineEnd])
	if err != nil {
		return RequestView{}, 0, report, err
	}
	headers = headers[:0]
	for off := lineEnd + 2; off < headerEnd; {
		rel := indexCRLF(input[off:headerEnd])
		next := headerEnd
		if rel >= 0 {
			next = off + rel
		}
		if len(headers) >= limits.MaxHeaders {
			return RequestView{}, 0, report, ErrTooManyHeaders
		}
		header, err := parseHeaderLineView(input[off:next])
		if err != nil {
			return RequestView{}, 0, report, err
		}
		headers = append(headers, header)
		report.HeaderViewsBorrowed++
		if rel < 0 {
			off = headerEnd
		} else {
			off = next + 2
		}
	}
	req.Headers = headers
	if err := applyHeaderMetadataView(&req, limits); err != nil {
		return RequestView{}, 0, report, err
	}
	consumed := headerBytes + req.ContentLength
	if len(input) < consumed {
		return RequestView{}, 0, report, ErrIncomplete
	}
	if req.ContentLength > 0 {
		req.Body = input[headerBytes:consumed]
	}
	return req, consumed, report, nil
}

func parseRequestLineView(line []byte) (RequestView, error) {
	first := indexByte(line, ' ')
	if first <= 0 {
		return RequestView{}, ErrMalformedRequest
	}
	secondRel := indexByte(line[first+1:], ' ')
	if secondRel <= 0 {
		return RequestView{}, ErrMalformedRequest
	}
	second := first + 1 + secondRel
	if second == len(line)-1 {
		return RequestView{}, ErrMalformedRequest
	}
	method := line[:first]
	target := line[first+1 : second]
	version := line[second+1:]
	if len(target) == 0 || target[0] != '/' || !validTokenBytes(method) {
		return RequestView{}, ErrMalformedRequest
	}
	if !bytesEqualString(version, "HTTP/1.1") && !bytesEqualString(version, "HTTP/1.0") {
		return RequestView{}, ErrUnsupportedVersion
	}
	path := target
	var query []byte
	if queryStart := indexByte(target, '?'); queryStart >= 0 {
		path = target[:queryStart]
		query = target[queryStart+1:]
	}
	if len(path) == 0 {
		return RequestView{}, ErrMalformedRequest
	}
	return RequestView{
		Method:        method,
		RequestTarget: target,
		Path:          path,
		Query:         query,
		Version:       version,
	}, nil
}

func parseHeaderLineView(line []byte) (HeaderView, error) {
	colon := indexByte(line, ':')
	if colon <= 0 {
		return HeaderView{}, ErrMalformedHeader
	}
	name := line[:colon]
	value := trimHTTPWhitespace(line[colon+1:])
	if !validTokenBytes(name) || containsControlBytes(value) {
		return HeaderView{}, ErrMalformedHeader
	}
	return HeaderView{Name: name, Value: value}, nil
}

func applyHeaderMetadataView(req *RequestView, limits Limits) error {
	if value := req.Header("Transfer-Encoding"); len(value) > 0 &&
		!asciiEqualFoldBytesString(value, "identity") {
		return ErrUnsupportedTransferEncoding
	}
	req.ContentLength = 0
	if value := req.Header("Content-Length"); len(value) > 0 {
		n, ok := parseNonNegativeIntBytes(value)
		if !ok {
			return ErrMalformedHeader
		}
		if n > limits.MaxBodyBytes {
			return ErrBodyTooLarge
		}
		req.ContentLength = n
	}
	connection := req.Header("Connection")
	switch {
	case bytesEqualString(req.Version, "HTTP/1.1"):
		req.KeepAlive = !asciiEqualFoldBytesString(connection, "close")
	case bytesEqualString(req.Version, "HTTP/1.0"):
		req.KeepAlive = asciiEqualFoldBytesString(connection, "keep-alive")
	}
	return nil
}

func indexHeaderEnd(input []byte) int {
	for i := 0; i+3 < len(input); i++ {
		if input[i] == '\r' && input[i+1] == '\n' && input[i+2] == '\r' && input[i+3] == '\n' {
			return i
		}
	}
	return -1
}

func indexCRLF(input []byte) int {
	for i := 0; i+1 < len(input); i++ {
		if input[i] == '\r' && input[i+1] == '\n' {
			return i
		}
	}
	return -1
}

func indexByte(input []byte, needle byte) int {
	for i, c := range input {
		if c == needle {
			return i
		}
	}
	return -1
}

func trimHTTPWhitespace(input []byte) []byte {
	start := 0
	for start < len(input) && (input[start] == ' ' || input[start] == '\t') {
		start++
	}
	end := len(input)
	for end > start && (input[end-1] == ' ' || input[end-1] == '\t') {
		end--
	}
	return input[start:end]
}

func validTokenBytes(value []byte) bool {
	if len(value) == 0 {
		return false
	}
	for _, c := range value {
		if c <= 32 || c >= 127 || isHTTPSeparator(c) {
			return false
		}
	}
	return true
}

func isHTTPSeparator(c byte) bool {
	switch c {
	case '(',
		')',
		'<',
		'>',
		'@',
		',',
		';',
		':',
		'\\',
		'"',
		'/',
		'[',
		']',
		'?',
		'=',
		'{',
		'}',
		' ',
		'\t':
		return true
	default:
		return false
	}
}

func containsControlBytes(value []byte) bool {
	for _, c := range value {
		if c < 32 && c != '\t' {
			return true
		}
		if c == 127 {
			return true
		}
	}
	return false
}

func parseNonNegativeIntBytes(value []byte) (int, bool) {
	if len(value) == 0 {
		return 0, false
	}
	n := 0
	for _, c := range value {
		if c < '0' || c > '9' {
			return 0, false
		}
		next := n*10 + int(c-'0')
		if next < n {
			return 0, false
		}
		n = next
	}
	return n, true
}

func asciiEqualFoldBytesString(value []byte, name string) bool {
	if len(value) != len(name) {
		return false
	}
	for i, c := range value {
		other := name[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if other >= 'A' && other <= 'Z' {
			other += 'a' - 'A'
		}
		if c != other {
			return false
		}
	}
	return true
}

func bytesEqualString(a []byte, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
