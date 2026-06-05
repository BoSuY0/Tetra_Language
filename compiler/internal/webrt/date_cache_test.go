package webrt

import (
	"testing"
	"time"
)

func TestHTTPDateCacheRefreshesOncePerSecond(t *testing.T) {
	cache := HTTPDateCache{}
	base := time.Date(2026, time.May, 20, 12, 0, 0, 123*int(time.Millisecond), time.UTC)

	first, firstReport := cache.FormatWithReport(base)
	if first != "Wed, 20 May 2026 12:00:00 GMT" {
		t.Fatalf("first date = %q", first)
	}
	if !firstReport.Refreshed {
		t.Fatalf("first report did not refresh: %#v", firstReport)
	}
	if firstReport.UnixSecond != base.Unix() || firstReport.Value != first {
		t.Fatalf("first report = %#v, want second %d value %q", firstReport, base.Unix(), first)
	}

	same, sameReport := cache.FormatWithReport(base.Add(700 * time.Millisecond))
	if same != first {
		t.Fatalf("same-second date = %q, want cached %q", same, first)
	}
	if sameReport.Refreshed {
		t.Fatalf("same-second report refreshed unexpectedly: %#v", sameReport)
	}
	if cache.RefreshCount() != 1 {
		t.Fatalf("refresh count after same-second reuse = %d, want 1", cache.RefreshCount())
	}

	nextTime := base.Add(time.Second)
	next, nextReport := cache.FormatWithReport(nextTime)
	if next != "Wed, 20 May 2026 12:00:01 GMT" {
		t.Fatalf("next-second date = %q", next)
	}
	if !nextReport.Refreshed {
		t.Fatalf("next-second report did not refresh: %#v", nextReport)
	}
	if cache.RefreshCount() != 2 {
		t.Fatalf("refresh count after next-second refresh = %d, want 2", cache.RefreshCount())
	}
}

func TestHTTPDateCacheFormatsUTC(t *testing.T) {
	cache := HTTPDateCache{}
	zone := time.FixedZone("UTC+2", 2*60*60)
	got, report := cache.FormatWithReport(time.Date(2026, time.May, 20, 15, 30, 1, 0, zone))
	if got != "Wed, 20 May 2026 13:30:01 GMT" {
		t.Fatalf("date = %q", got)
	}
	if report.UnixSecond != time.Date(2026, time.May, 20, 13, 30, 1, 0, time.UTC).Unix() {
		t.Fatalf("report UnixSecond = %d", report.UnixSecond)
	}
}

func TestServerDateUsesPerSecondCacheWhenDateFuncAbsent(t *testing.T) {
	base := time.Date(2026, time.May, 20, 12, 0, 0, 0, time.UTC)
	times := []time.Time{
		base,
		base.Add(500 * time.Millisecond),
		base.Add(time.Second),
	}
	index := 0
	srv := NewServer(Config{
		NowFunc: func() time.Time {
			if index >= len(times) {
				return times[len(times)-1]
			}
			now := times[index]
			index++
			return now
		},
	})

	first := srv.date()
	same := srv.date()
	next := srv.date()

	if first != same {
		t.Fatalf("same-second server date = %q, want cached %q", same, first)
	}
	if next == first {
		t.Fatalf("next-second server date did not refresh: %q", next)
	}
	if srv.dateCache.RefreshCount() != 2 {
		t.Fatalf("server date cache refresh count = %d, want 2", srv.dateCache.RefreshCount())
	}
}

func TestServerDateFuncOverrideBypassesCache(t *testing.T) {
	nowCalled := false
	srv := NewServer(Config{
		DateFunc: func() string {
			return "Wed, 20 May 2026 12:00:00 GMT"
		},
		NowFunc: func() time.Time {
			nowCalled = true
			return time.Date(2026, time.May, 20, 12, 0, 0, 0, time.UTC)
		},
	})

	if got := srv.date(); got != "Wed, 20 May 2026 12:00:00 GMT" {
		t.Fatalf("date override = %q", got)
	}
	if nowCalled {
		t.Fatalf("NowFunc was called despite DateFunc override")
	}
	if srv.dateCache.RefreshCount() != 0 {
		t.Fatalf("DateFunc override touched cache refresh count %d", srv.dateCache.RefreshCount())
	}
}
