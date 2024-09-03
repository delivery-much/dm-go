package request

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"time"
)

type URL struct {
	scheme, host, path string
	queries            url.Values
}

// NewURL instantiantes a new URL given an raw url (optional)
func NewURL(rawURL ...string) *URL {
	u := &URL{}

	if len(rawURL) == 0 || rawURL[0] == "" {
		return u
	}

	parsedURL, err := url.Parse(rawURL[0])
	if err != nil {
		return u
	}

	return &URL{
		scheme:  parsedURL.Scheme,
		host:    parsedURL.Host,
		path:    parsedURL.Path,
		queries: parsedURL.Query(),
	}
}

// SetScheme sets the URL scheme
func (u *URL) SetScheme(s string) *URL {
	if s != "" {
		u.scheme = s
	}
	return u
}

// SetHost sets the URL host
func (u *URL) SetHost(h string) *URL {
	if h != "" {
		u.host = h
	}
	return u
}

// SetPath sets the URL path
func (u *URL) SetPath(p string) *URL {
	if p != "" {
		u.path = p
	}
	return u
}

// SetPath adds a new query to URL
func (u *URL) AddQuery(key, val string) *URL {
	if u.queries == nil {
		u.queries = url.Values{}
	}
	
	if key != "" && val != "" {
		u.queries.Set(key, val)
	}
	return u
}

// Parse parses the URL to a raw url string
func (u *URL) Parse() (s string, err error) {
	host := u.host
	if u.scheme != "" {
		host = fmt.Sprintf("%s://%s", u.scheme, host)
	}

	rawURL, err := url.JoinPath(host, u.path)
	if err != nil {
		err = fmt.Errorf("unable to parse url with host %s and path %s", host, u.path)
		return
	}

	parsedURL, _ := url.Parse(rawURL)
	parsedURL.RawQuery = u.queries.Encode()

	s = parsedURL.String()
	return
}

type Response http.Response

// IsSuccessCode checks if response status code is from a success response.
func (r *Response) IsSuccessCode() bool {
	return r.StatusCode >= 200 && r.StatusCode <= 299
}

// IsFailureCode checks if response status code is from an error response.
func (r *Response) IsFailureCode() bool {
	return r.StatusCode >= 400 && r.StatusCode <= 599
}

// DecodeJSON decodes the response body into an object pointer
func (r *Response) DecodeJSON(value any) error {
	if value == nil || reflect.ValueOf(value).Kind() != reflect.Ptr {
		return errors.New("value must be a valid pointer of a struct")
	}

	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(value)
}

type Params struct {
	Method    string
	URLString string
	URL       *URL
	Headers   map[string]string
	Body      io.Reader
	Timeout   time.Duration
}

// Do performs a request given the informed params and returns the response
func Do(p Params) (res Response, err error) {
	url := NewURL(p.URLString)

	if p.URL != nil {
		url = p.URL
	}

	req, err := mountRequest(url, p)
	if err != nil {
		return
	}

	if p.Timeout == 0 {
		p.Timeout = 9 * time.Second
	}

	client := &http.Client{
		Timeout: p.Timeout,
	}

	response, err := client.Do(req)
	if err != nil {
		return
	}

	res = Response(*response)

	return
}

// mountRequest mounts a request given an URL and the params
func mountRequest(url *URL, params Params) (req *http.Request, err error) {
	rawURL, err := url.Parse()
	if err != nil {
		return
	}

	req, err = http.NewRequest(params.Method, rawURL, params.Body)
	if err != nil {
		return
	}

	for key, value := range params.Headers {
		if value != "" {
			req.Header.Set(key, value)
		}
	}

	return
}

// Get performs a GET request given the informed params and returns the response
func Get(params Params) (Response, error) {
	params.Method = http.MethodGet
	return Do(params)
}

// Post performs a POST request given the informed params and returns the response
func Post(params Params) (Response, error) {
	params.Method = http.MethodPost
	return Do(params)
}

// Put performs a PUT request given the informed params and returns the response
func Put(params Params) (Response, error) {
	params.Method = http.MethodPut
	return Do(params)
}

// Patch performs a PATCH request given the informed params and returns the response
func Patch(params Params) (Response, error) {
	params.Method = http.MethodPatch
	return Do(params)
}

// Delete performs a DELETE request given the informed params and returns the response
func Delete(params Params) (Response, error) {
	params.Method = http.MethodDelete
	return Do(params)
}
