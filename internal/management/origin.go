package management

import "strings"

// originFromHeaders returns scheme://host with no trailing slash.
// Prefer X-Forwarded-Proto + X-Forwarded-Host when host is present via forwarded
// headers; otherwise use Host with scheme defaulting to http.
func originFromHeaders(headers map[string][]string) string {
	host := firstHeader(headers, "X-Forwarded-Host", "Host")
	if host == "" {
		return ""
	}
	// X-Forwarded-Host may be a comma-separated list; take the first.
	if i := strings.IndexByte(host, ','); i >= 0 {
		host = strings.TrimSpace(host[:i])
	}
	scheme := firstHeader(headers, "X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
	}
	if i := strings.IndexByte(scheme, ','); i >= 0 {
		scheme = strings.TrimSpace(scheme[:i])
	}
	scheme = strings.ToLower(scheme)
	return scheme + "://" + host
}
