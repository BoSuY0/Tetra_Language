package webrt

import "time"

// HTTPDateCache caches formatted HTTP Date values; callers serialize access.
type HTTPDateCache struct {
	initialized bool
	unixSecond  int64
	value       string
	refreshes   int
}

type HTTPDateCacheReport struct {
	UnixSecond   int64  `json:"unix_second"`
	Value        string `json:"value"`
	Refreshed    bool   `json:"refreshed"`
	RefreshCount int    `json:"refresh_count"`
}

func (c *HTTPDateCache) Format(now time.Time) string {
	value, _ := c.FormatWithReport(now)
	return value
}

func (c *HTTPDateCache) FormatWithReport(now time.Time) (string, HTTPDateCacheReport) {
	utc := now.UTC()
	second := utc.Unix()
	refreshed := false
	if !c.initialized || c.unixSecond != second {
		c.initialized = true
		c.unixSecond = second
		c.value = utc.Format(httpDateLayout)
		c.refreshes++
		refreshed = true
	}
	return c.value, HTTPDateCacheReport{
		UnixSecond:   c.unixSecond,
		Value:        c.value,
		Refreshed:    refreshed,
		RefreshCount: c.refreshes,
	}
}

func (c *HTTPDateCache) RefreshCount() int {
	return c.refreshes
}
