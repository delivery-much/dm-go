package request

import (
	"io"
	"net/http"
	"time"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

var Client HTTPClient = &http.Client{
	Timeout: 9 * time.Second,
}

type Params struct {
	Headers map[string]string
	Body    io.Reader
}

func extractParams(params *Params) (io.Reader, map[string]string) {
	var body io.Reader
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	if params == nil {
		return body, headers
	}

	if params.Body != nil {
		body = params.Body
	}

	if params.Headers != nil && len(params.Headers) > 0 {
		for key, value := range params.Headers {
			if value != "" {
				headers[key] = value
			}
		}
	}

	return body, headers
}

func mountRequest(method, url string, params *Params) (req *http.Request, err error) {
	body, headers := extractParams(params)

	req, err = http.NewRequest(method, url, body)
	if err != nil {
		return
	}

	if len(headers) > 0 {
		for key, value := range params.Headers {
			req.Header.Set(key, value)
		}
	}

	return
}

func Request(method, url string, params *Params) (res *http.Response, err error) {
	if method == "" {
		method = http.MethodGet
	}

	req, err := mountRequest(method, url, params)
	if err != nil {
		return
	}

	return Client.Do(req)
}

func Get(url string, params *Params) (res *http.Response, err error) {
	return Request(http.MethodGet, url, params)
}

func Post(url string, params *Params) (*http.Response, error) {
	return Request(http.MethodPost, url, params)
}

func Put(url string, params *Params) (*http.Response, error) {
	return Request(http.MethodPut, url, params)
}

func Patch(url string, params *Params) (*http.Response, error) {
	return Request(http.MethodPatch, url, params)
}

func Delete(url string, params *Params) (*http.Response, error) {
	return Request(http.MethodDelete, url, params)
}
