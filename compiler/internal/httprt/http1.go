package httprt

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
)

var (
	ErrIncomplete                  = errors.New("incomplete HTTP request")
	ErrHeaderTooLarge              = errors.New("HTTP header section exceeds limit")
	ErrTooManyHeaders              = errors.New("HTTP request has too many headers")
	ErrBodyTooLarge                = errors.New("HTTP request body exceeds limit")
	ErrMalformedRequest            = errors.New("malformed HTTP request")
	ErrMalformedHeader             = errors.New("malformed HTTP header")
	ErrUnsupportedVersion          = errors.New("unsupported HTTP version")
	ErrUnsupportedTransferEncoding = errors.New("unsupported HTTP transfer encoding")
)

type Limits struct {
	MaxHeaderBytes int
	MaxHeaders     int
	MaxBodyBytes   int
}

type Header struct {
	Name  string
	Value string
}

type Request struct {
	Method        string
	RequestTarget string
	Path          string
	Query         string
	Version       string
	Headers       []Header
	PathParams    []PathParam
	ContentLength int
	Body          []byte
	KeepAlive     bool
}

type PathParam struct {
	Name  string
	Value string
}

func (r Request) Header(name string) string {
	for _, header := range r.Headers {
		if strings.EqualFold(header.Name, name) {
			return header.Value
		}
	}
	return ""
}

func (r Request) QueryValue(name string) string {
	for len(r.Query) > 0 {
		part := r.Query
		if amp := strings.IndexByte(part, '&'); amp >= 0 {
			part = r.Query[:amp]
			r.Query = r.Query[amp+1:]
		} else {
			r.Query = ""
		}
		key := part
		value := ""
		if eq := strings.IndexByte(part, '='); eq >= 0 {
			key = part[:eq]
			value = part[eq+1:]
		}
		if key == name {
			return value
		}
	}
	return ""
}

func (r Request) PathValue(name string) string {
	for _, param := range r.PathParams {
		if param.Name == name {
			return param.Value
		}
	}
	return ""
}

func ParseRequest(input []byte, limits Limits) (Request, int, error) {
	limits = normalizeLimits(limits)
	headerEnd := bytes.Index(input, []byte("\r\n\r\n"))
	if headerEnd < 0 {
		if len(input) > limits.MaxHeaderBytes {
			return Request{}, 0, ErrHeaderTooLarge
		}
		return Request{}, 0, ErrIncomplete
	}
	headerBytes := headerEnd + len("\r\n\r\n")
	if headerBytes > limits.MaxHeaderBytes {
		return Request{}, 0, ErrHeaderTooLarge
	}

	lines := strings.Split(string(input[:headerEnd]), "\r\n")
	if len(lines) == 0 {
		return Request{}, 0, ErrMalformedRequest
	}
	req, err := parseRequestLine(lines[0])
	if err != nil {
		return Request{}, 0, err
	}
	if len(lines)-1 > limits.MaxHeaders {
		return Request{}, 0, ErrTooManyHeaders
	}
	for _, line := range lines[1:] {
		header, err := parseHeaderLine(line)
		if err != nil {
			return Request{}, 0, err
		}
		req.Headers = append(req.Headers, header)
	}
	if err := applyHeaderMetadata(&req, limits); err != nil {
		return Request{}, 0, err
	}
	consumed := headerBytes + req.ContentLength
	if len(input) < consumed {
		return Request{}, 0, ErrIncomplete
	}
	if req.ContentLength > 0 {
		req.Body = input[headerBytes:consumed]
	}
	return req, consumed, nil
}

func normalizeLimits(limits Limits) Limits {
	if limits.MaxHeaderBytes <= 0 {
		limits.MaxHeaderBytes = 8192
	}
	if limits.MaxHeaders <= 0 {
		limits.MaxHeaders = 64
	}
	if limits.MaxBodyBytes <= 0 {
		limits.MaxBodyBytes = 1 << 20
	}
	return limits
}

func parseRequestLine(line string) (Request, error) {
	parts := strings.Split(line, " ")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return Request{}, ErrMalformedRequest
	}
	if parts[1][0] != '/' {
		return Request{}, ErrMalformedRequest
	}
	if !validToken(parts[0]) {
		return Request{}, ErrMalformedRequest
	}
	if parts[2] != "HTTP/1.1" && parts[2] != "HTTP/1.0" {
		return Request{}, ErrUnsupportedVersion
	}
	req := Request{Method: parts[0], RequestTarget: parts[1], Path: parts[1], Version: parts[2]}
	if queryStart := strings.IndexByte(parts[1], '?'); queryStart >= 0 {
		req.Path = parts[1][:queryStart]
		req.Query = parts[1][queryStart+1:]
	}
	if req.Path == "" {
		return Request{}, ErrMalformedRequest
	}
	return req, nil
}

func parseHeaderLine(line string) (Header, error) {
	colon := strings.IndexByte(line, ':')
	if colon <= 0 {
		return Header{}, ErrMalformedHeader
	}
	name := line[:colon]
	value := strings.Trim(line[colon+1:], " \t")
	if !validToken(name) || containsControl(value) {
		return Header{}, ErrMalformedHeader
	}
	return Header{Name: name, Value: value}, nil
}

func applyHeaderMetadata(req *Request, limits Limits) error {
	if value := req.Header("Transfer-Encoding"); value != "" &&
		!strings.EqualFold(value, "identity") {
		return ErrUnsupportedTransferEncoding
	}
	req.ContentLength = 0
	if value := req.Header("Content-Length"); value != "" {
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return ErrMalformedHeader
		}
		if n > limits.MaxBodyBytes {
			return ErrBodyTooLarge
		}
		req.ContentLength = n
	}
	connection := strings.ToLower(req.Header("Connection"))
	switch req.Version {
	case "HTTP/1.1":
		req.KeepAlive = connection != "close"
	case "HTTP/1.0":
		req.KeepAlive = connection == "keep-alive"
	}
	return nil
}

func validToken(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c <= 32 || c >= 127 || strings.ContainsRune("()<>@,;:\\\"/[]?={} \t", rune(c)) {
			return false
		}
	}
	return true
}

func containsControl(value string) bool {
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c < 32 && c != '\t' {
			return true
		}
		if c == 127 {
			return true
		}
	}
	return false
}

type Response struct {
	StatusCode  int
	ContentType string
	Body        []byte
	Server      string
	Date        string
	KeepAlive   bool
	Headers     []Header
}

func AppendResponse(dst []byte, resp Response) []byte {
	statusCode := resp.StatusCode
	if statusCode == 0 {
		statusCode = 200
	}
	dst = append(dst, "HTTP/1.1 "...)
	dst = strconv.AppendInt(dst, int64(statusCode), 10)
	dst = append(dst, ' ')
	dst = append(dst, statusText(statusCode)...)
	dst = append(dst, "\r\n"...)
	if resp.Server != "" {
		dst = appendHeader(dst, "Server", resp.Server)
	}
	if resp.Date != "" {
		dst = appendHeader(dst, "Date", resp.Date)
	}
	if resp.ContentType != "" {
		dst = appendHeader(dst, "Content-Type", resp.ContentType)
	}
	dst = append(dst, "Content-Length: "...)
	dst = strconv.AppendInt(dst, int64(len(resp.Body)), 10)
	dst = append(dst, "\r\n"...)
	if resp.KeepAlive {
		dst = appendHeader(dst, "Connection", "keep-alive")
	} else {
		dst = appendHeader(dst, "Connection", "close")
	}
	for _, header := range resp.Headers {
		dst = appendHeader(dst, header.Name, header.Value)
	}
	dst = append(dst, "\r\n"...)
	dst = append(dst, resp.Body...)
	return dst
}

func appendHeader(dst []byte, name string, value string) []byte {
	dst = append(dst, name...)
	dst = append(dst, ": "...)
	dst = append(dst, value...)
	dst = append(dst, "\r\n"...)
	return dst
}

func statusText(code int) string {
	switch code {
	case 200:
		return "OK"
	case 400:
		return "Bad Request"
	case 404:
		return "Not Found"
	case 413:
		return "Payload Too Large"
	case 500:
		return "Internal Server Error"
	default:
		return "Status"
	}
}

type Handler func(Request) Response

type Middleware func(Handler) Handler

type route struct {
	method  string
	path    string
	handler Handler
	params  []string
}

type Router struct {
	routes      []route
	middlewares []Middleware
}

func (r *Router) Use(middleware Middleware) {
	if middleware != nil {
		r.middlewares = append(r.middlewares, middleware)
	}
}

func (r *Router) Handle(method string, path string, handler Handler) {
	r.routes = append(
		r.routes,
		route{method: method, path: path, handler: r.wrap(handler), params: routeParams(path)},
	)
}

func (r *Router) Route(req Request) (Response, bool) {
	for _, route := range r.routes {
		if route.method == req.Method && len(route.params) == 0 && route.path == req.Path {
			return route.handler(req), true
		}
	}
	for _, route := range r.routes {
		if route.method != req.Method || len(route.params) == 0 {
			continue
		}
		params, ok := matchParameterizedPath(route.path, route.params, req.Path)
		if !ok {
			continue
		}
		req.PathParams = params
		return route.handler(req), true
	}
	return Response{}, false
}

func (r *Router) wrap(handler Handler) Handler {
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		handler = r.middlewares[i](handler)
	}
	return handler
}

func routeParams(path string) []string {
	segments := splitPath(path)
	params := make([]string, 0)
	for _, segment := range segments {
		if strings.HasPrefix(segment, ":") && len(segment) > 1 {
			params = append(params, segment[1:])
		}
	}
	return params
}

func matchParameterizedPath(pattern string, names []string, path string) ([]PathParam, bool) {
	patternSegments := splitPath(pattern)
	pathSegments := splitPath(path)
	if len(patternSegments) != len(pathSegments) {
		return nil, false
	}
	params := make([]PathParam, 0, len(names))
	for i, segment := range patternSegments {
		if strings.HasPrefix(segment, ":") && len(segment) > 1 {
			if pathSegments[i] == "" {
				return nil, false
			}
			params = append(params, PathParam{Name: segment[1:], Value: pathSegments[i]})
			continue
		}
		if segment != pathSegments[i] {
			return nil, false
		}
	}
	return params, true
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}
