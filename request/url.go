package request

import (
	"net/url"
	"strings"
)

type URL struct {
	Scheme string
	Host   string
	Path   string
	Query  url.Values
}

func NewURL(raw string) *URL {
	parsedURL, err := url.Parse(raw)
	if err != nil {
		return &URL{}
	}

	return &URL{
		Scheme: parsedURL.Scheme,
		Host:   parsedURL.Host,
		Path:   parsedURL.Path,
		Query:  parsedURL.Query(),
	}
}

func (u *URL) AddQuery(key string, values ...string) {
	if key == "" || len(values) == 0 {
		return
	}

	if u.Query == nil {
		u.Query = make(map[string][]string)
	}

	u.Query[key] = values
}

func (u *URL) String() string {
	parsedURL := &url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
		Path:   u.Path,
	}

	parsedURL.RawQuery = u.Query.Encode()
	parsedURL.RawQuery = strings.ReplaceAll(parsedURL.RawQuery, "+", "%20")

	return parsedURL.String()
}
