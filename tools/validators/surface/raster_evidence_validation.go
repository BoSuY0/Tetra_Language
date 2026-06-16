package surface

import (
	"fmt"
	"strings"
)

func validateRasterProof(prefix string, want string, format string, hash string, width int, height int, coverage int, markerOnly bool) []string {
	var issues []string
	format = strings.TrimSpace(format)
	want = strings.TrimSpace(want)
	if markerOnly {
		issues = append(issues, fmt.Sprintf("%s marker_only must be false for %s raster evidence", prefix, want))
	}
	if strings.Contains(strings.ToLower(format), "marker") {
		issues = append(issues, fmt.Sprintf("%s raster_format %q must not be marker evidence", prefix, format))
	}
	if want != "" && format != want {
		issues = append(issues, fmt.Sprintf("%s raster_format is %q, want %s", prefix, format, want))
	}
	if !validSHA256Digest(hash) {
		issues = append(issues, fmt.Sprintf("%s raster_hash must be sha256 evidence", prefix))
	}
	if width <= 0 || height <= 0 {
		issues = append(issues, fmt.Sprintf("%s raster dimensions must be positive", prefix))
	}
	if coverage <= 0 {
		issues = append(issues, fmt.Sprintf("%s raster_coverage must be positive", prefix))
	}
	if width > 0 && height > 0 && coverage > width*height {
		issues = append(issues, fmt.Sprintf("%s raster_coverage %d exceeds raster dimensions %dx%d", prefix, coverage, width, height))
	}
	return issues
}
