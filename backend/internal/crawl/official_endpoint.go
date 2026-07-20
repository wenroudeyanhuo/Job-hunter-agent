package crawl

import (
	"net/url"
	"strings"
)

func normalizeEndpointRoot(endpoint string) string {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return endpoint
	}
	return parsed.Scheme + "://" + parsed.Host
}
