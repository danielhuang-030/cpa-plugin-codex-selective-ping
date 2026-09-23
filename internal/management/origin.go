package management

import "strings"

// originFromHeaders returns scheme://host with no trailing slash.
// Prefer X-Csp-Origin (full origin) when present and valid; CPA pluginhost
// often omits Host from cloned headers (net/http already moved it to r.Host).
// Else prefer X-Forwarded-Proto + X-Forwarded-Host; otherwise Host with
// scheme defaulting to http.
func originFromHeaders(headers map[string][]string) string {
	if csp := firstHeader(headers, "X-Csp-Origin"); csp != "" {
		csp = strings.TrimRight(strings.TrimSpace(csp), "/")
		if strings.Contains(csp, "://") {
			return csp
		}
	}
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
