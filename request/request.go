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

func getHeaders(params *Params) map[string]string {
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	if params == nil || params.Headers == nil || len(params.Headers) == 0 {
		return headers
	}

	for key, value := range params.Headers {
		if value != "" {
			headers[key] = value
		}
	}

	return headers
}

func getBody(params *Params) io.Reader {
	if params == nil || params.Body == nil {
		return nil
	}

	return params.Body
}

func mountRequest(method, url string, params *Params) (req *http.Request, err error) {
	headers := getHeaders(params)
	body := getBody(params)

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
