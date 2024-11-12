package request

import (
	"io"
	"net/http"
)

var httpClientAdapter clientAdapterInterface = &clientAdapter{}

type Client http.Client

type Params struct {
	// Method is the request method (i.e.: "GET", "POST", etc)
	Method string

	// URL is the URL to execute the request
	URL *URL

	// Headers represents the request optional headers key-value pairs
	Headers map[string]string

	// Body represents the request optional body payload
	Body io.Reader
}

// Do performs a request given the informed params and returns the response
func (c *Client) Do(p Params) (res *Response, err error) {
	req, err := http.NewRequest(p.Method, p.URL.String(), p.Body)
	if err != nil {
		return
	}

	for key, value := range p.Headers {
		if value != "" {
			req.Header.Set(key, value)
		}
	}

	httpClient := httpClientAdapter.Adapt(c)
	httpResponse, err := httpClient.Do(req)
	if err != nil {
		return
	}

	res = (*Response)(httpResponse)
	return
}

// Get performs a GET request given the informed params and returns the response
func (c *Client) Get(url *URL, headers ...map[string]string) (*Response, error) {
	params := Params{
		Method: http.MethodGet,
		URL:    url,
	}
	if len(headers) > 0 {
		params.Headers = headers[0]
	}

	return c.Do(params)
}

// Post performs a POST request given the informed params and returns the response
func (c *Client) Post(url *URL, body io.Reader, headers ...map[string]string) (*Response, error) {
	params := Params{
		Method: http.MethodPost,
		URL:    url,
		Body:   body,
	}
	if len(headers) > 0 {
		params.Headers = headers[0]
	}

	return c.Do(params)
}

// Put performs a PUT request given the informed params and returns the response
func (c *Client) Put(url *URL, body io.Reader, headers ...map[string]string) (*Response, error) {
	params := Params{
		Method: http.MethodPut,
		URL:    url,
		Body:   body,
	}
	if len(headers) > 0 {
		params.Headers = headers[0]
	}

	return c.Do(params)
}

// Patch performs a PATCH request given the informed params and returns the response
func (c *Client) Patch(url *URL, body io.Reader, headers ...map[string]string) (*Response, error) {
	params := Params{
		Method: http.MethodPut,
		URL:    url,
		Body:   body,
	}
	if len(headers) > 0 {
		params.Headers = headers[0]
	}

	return c.Do(params)
}

// Delete performs a DELETE request given the informed params and returns the response
func (c *Client) Delete(url *URL, headers ...map[string]string) (*Response, error) {
	params := Params{
		Method: http.MethodPut,
		URL:    url,
	}
	if len(headers) > 0 {
		params.Headers = headers[0]
	}

	return c.Do(params)
}
