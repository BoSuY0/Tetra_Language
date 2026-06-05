package httprt

import (
	"errors"

	"tetra_language/compiler/internal/stdlibrt"
)

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

func (s *RequestRegionScope) Run(input []byte, limits Limits, handler RequestRegionHandler, write RequestRegionWriter) (consumed int, report RequestRegionReport, err error) {
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

	req, parsed, requestReport, parseErr := ParseRequestViewInRegion(input, limits, s.headers[:0], s.region)
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
