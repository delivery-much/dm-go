package request

import (
	"io"
)

// ClientInterface represents the interface for a http client that executes requests
//
// Use this interface to define your client variables so they can be mocked.
type ClientInterface interface {
	// Do performs a request given the provided params and returns the response
	Do(params Params) (*Response, error)

	// Get performs a GET request given the provided params and returns the response
	Get(url *URL, headers ...map[string]string) (*Response, error)

	// Post performs a POST request given the provided params and returns the response
	Post(url *URL, body io.Reader, headers ...map[string]string) (*Response, error)

	// Put performs a PUT request given the provided params and returns the response
	Put(url *URL, body io.Reader, headers ...map[string]string) (*Response, error)

	// Patch performs a PATCH request given the provided params and returns the response
	Patch(url *URL, body io.Reader, headers ...map[string]string) (*Response, error)

	// Delete performs a DELETE request given the provided params and returns the response
	Delete(url *URL, headers ...map[string]string) (*Response, error)
}
