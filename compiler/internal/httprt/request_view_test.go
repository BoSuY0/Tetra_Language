package httprt

import (
	"bytes"
	"errors"
	"testing"

	"tetra_language/compiler/internal/jsonrt"
	"tetra_language/compiler/internal/stdlibrt"
)

func TestParseRequestViewBorrowsHeadersWithoutAllocating(t *testing.T) {
	raw := []byte(
		"GET /json?queries=2 HTTP/1.1\r\nHost: localhost\r\nConnection: keep-alive\r\n\r\n",
	)
	limits := Limits{MaxHeaderBytes: 4096, MaxHeaders: 8}
	scratch := make([]HeaderView, 0, 8)
	var req RequestView
	var consumed int
	var report RequestParseReport
	var err error

	allocs := testing.AllocsPerRun(1000, func() {
		req, consumed, report, err = ParseRequestView(raw, limits, scratch[:0])
		if err != nil {
			t.Fatalf("ParseRequestView: %v", err)
		}
	})
	if allocs != 0 {
		t.Fatalf("ParseRequestView allocations = %.2f, want 0", allocs)
	}
	if consumed != len(raw) || string(req.Path) != "/json" || string(req.Query) != "queries=2" ||
		!req.KeepAlive {
		t.Fatalf("request view = %#v consumed=%d", req, consumed)
	}
	if string(req.Header("host")) != "localhost" {
		t.Fatalf("host header = %q, want localhost", req.Header("host"))
	}
	if report.HeapAllocations != 0 || report.HeaderViewsBorrowed != 2 ||
		report.HeaderValuesCopied != 0 ||
		report.RequestStorage != stdlibrt.StorageBorrowed {
		t.Fatalf("request parse report = %#v", report)
	}
}

func TestParseRequestViewReportsRegionRequestBufferAndInvalidInput(t *testing.T) {
	raw := []byte("GET /plaintext HTTP/1.1\r\nBroken\r\n\r\n")
	region := stdlibrt.NewRegion("http-request", 512)
	scratch := make([]HeaderView, 0, 8)

	_, _, report, err := ParseRequestViewInRegion(raw, Limits{}, scratch, region)
	if !errors.Is(err, ErrMalformedHeader) {
		t.Fatalf("ParseRequestView malformed error = %v, want ErrMalformedHeader", err)
	}
	if report.RequestStorage != stdlibrt.StorageRegion || report.RegionID != "http-request" {
		t.Fatalf("region report = %#v", report)
	}
	if report.UnsafeFacts {
		t.Fatalf("malformed request produced unsafe facts: %#v", report)
	}
}

func TestAppendResponseWithReportRecordsBufferStorage(t *testing.T) {
	region := stdlibrt.NewRegion("http-response", 256)
	buf, err := region.Alloc(256)
	if err != nil {
		t.Fatalf("region alloc: %v", err)
	}
	out, report := AppendResponseWithReport(buf[:0], Response{
		StatusCode:  200,
		ContentType: "text/plain",
		Body:        []byte("Hello, World!"),
		KeepAlive:   true,
	}, ResponseBufferOptions{
		Storage:  stdlibrt.StorageRegion,
		RegionID: region.ID(),
	})
	if len(out) == 0 || report.BytesWritten != len(out) {
		t.Fatalf("response/report bytes = %d/%d", len(out), report.BytesWritten)
	}
	if report.ResponseBufferStorage != stdlibrt.StorageRegion ||
		report.RegionID != "http-response" ||
		report.HeapAllocations != 0 {
		t.Fatalf("response buffer report = %#v", report)
	}
}

func TestRequestRegionScopeInjectsRegionForHTTPJSONAndResetsAfterWrite(t *testing.T) {
	raw := []byte(
		"POST /json HTTP/1.1\r\nContent-Length: 14\r\nConnection: keep-alive\r\n\r\n\"hello\\nworld\"",
	)
	limits := Limits{MaxHeaderBytes: 4096, MaxHeaders: 8, MaxBodyBytes: 128}
	scope := NewRequestRegionScope(RequestRegionOptions{
		RegionID:         "request-1",
		RegionCapacity:   4096,
		HeaderCapacity:   8,
		ResponseCapacity: 256,
	})
	body := []byte("OK")
	wantStatus := []byte("HTTP/1.1 200 OK")
	wantBody := []byte("OK")
	wantDecoded := []byte("hello\nworld")
	var consumed int
	var report RequestRegionReport
	var err error
	var jsonReport jsonrt.ParseViewReport

	handler := func(req RequestView, region *stdlibrt.Region) (Response, error) {
		if region == nil || region.ID() != "request-1" {
			return Response{}, errors.New("missing request region injection")
		}
		value, parsed, parseErr := jsonrt.ParseValueView(
			req.Body,
			jsonrt.ParseViewOptions{Region: region},
		)
		jsonReport = parsed
		if parseErr != nil {
			return Response{}, parseErr
		}
		if value.Kind != jsonrt.StringKind || !bytes.Equal(value.String.Bytes, wantDecoded) ||
			value.String.Storage != stdlibrt.StorageRegion {
			return Response{}, errors.New("json body was not decoded into request region")
		}
		return Response{
			StatusCode:  200,
			ContentType: "text/plain",
			Body:        body,
			KeepAlive:   req.KeepAlive,
		}, nil
	}
	writer := func(out []byte) error {
		if !bytes.Contains(out, wantStatus) || !bytes.Contains(out, wantBody) {
			return errors.New("response bytes missing expected content")
		}
		return nil
	}

	allocs := testing.AllocsPerRun(1000, func() {
		consumed, report, err = scope.Run(raw, limits, handler, writer)
	})
	if err != nil {
		t.Fatalf("request region run: %v", err)
	}
	if allocs != 0 {
		t.Fatalf("request region ordinary HTTP/JSON allocations = %.2f, want 0", allocs)
	}
	if consumed != len(raw) {
		t.Fatalf("consumed = %d, want %d", consumed, len(raw))
	}
	if report.RegionID != "request-1" || report.Lifetime != "request" || !report.Reset {
		t.Fatalf("request region report = %#v, want request lifetime reset", report)
	}
	if report.Request.RequestStorage != stdlibrt.StorageRegion ||
		report.Response.ResponseBufferStorage != stdlibrt.StorageRegion {
		t.Fatalf(
			"request/response storage report = %#v/%#v, want region",
			report.Request,
			report.Response,
		)
	}
	if report.HeapAllocations != 0 || jsonReport.HeapAllocations != 0 ||
		jsonReport.RegionTemporaries != 1 {
		t.Fatalf(
			"heap/json reports = request %#v json %#v, want region temporaries without heap",
			report,
			jsonReport,
		)
	}
	if report.BytesUsedBeforeReset <= 0 || scope.RegionUsed() != 0 {
		t.Fatalf(
			"region reset evidence = used_before=%d used_after=%d",
			report.BytesUsedBeforeReset,
			scope.RegionUsed(),
		)
	}
}
