package request

import (
	"net/url"
	"strings"
)

// URL type represents a wrapper to url.URL type.
// It allows to keep using the url.URL resources, but with some
// custom features that makes the URL handling more friendly.
type URL struct {
	*url.URL
}

// ParseURL parses a raw URL string to a URL struct
func ParseURL(raw string) *URL {
	parsedURL, err := url.Parse(raw)
	if err != nil || parsedURL == nil {
		return &URL{}
	}

	return &URL{parsedURL}
}

// AddQuery adds a query directly to the URL, given the query key and values
func (u *URL) AddQuery(key string, values ...string) {
	if u.URL == nil || key == "" || len(values) == 0 {
		return
	}

	urlQuery := u.Query()

	for _, values := range values {
		urlQuery.Add(key, values)
	}

	encodedURL := urlQuery.Encode()
	encodedURL = strings.ReplaceAll(encodedURL, "+", "%20")

	u.RawQuery = encodedURL
}
