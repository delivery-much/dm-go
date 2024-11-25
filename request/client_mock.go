package request

import (
	"io"

	"github.com/delivery-much/mock-helper/mock"
)

// clientMock represents a mocked client that implements the ClientInterface
type clientMock struct {
	mock.Mock
}

// NewClientMock returns a new mocked client that implements the ClientInterface
func NewClientMock() *clientMock {
	return &clientMock{
		mock.NewMock(),
	}
}

// Do performs a request given the provided params and returns the response
func (cm *clientMock) Do(params Params) (r *Response, err error) {
	res := cm.GetResponseAndRegister("Do", params)
	if res.IsEmpty() {
		return
	}

	return res.Get(0).(*Response), res.GetError(1)
}

// Get performs a GET request given the provided params and returns the response
func (cm *clientMock) Get(url *URL, headers ...map[string]string) (r *Response, err error) {
	res := cm.GetResponseAndRegister("Get", url, headers)
	if res.IsEmpty() {
		return
	}

	return res.Get(0).(*Response), res.GetError(1)
}

// Post performs a POST request given the provided params and returns the response
func (cm *clientMock) Post(url *URL, body io.Reader, headers ...map[string]string) (r *Response, err error) {
	res := cm.GetResponseAndRegister("Post", url, body, headers)
	if res.IsEmpty() {
		return
	}

	return res.Get(0).(*Response), res.GetError(1)
}

// Put performs a PUT request given the provided params and returns the response
func (cm *clientMock) Put(url *URL, body io.Reader, headers ...map[string]string) (r *Response, err error) {
	res := cm.GetResponseAndRegister("Put", url, body, headers)
	if res.IsEmpty() {
		return
	}

	return res.Get(0).(*Response), res.GetError(1)
}

// Patch performs a PATCH request given the provided params and returns the response
func (cm *clientMock) Patch(url *URL, body io.Reader, headers ...map[string]string) (r *Response, err error) {
	res := cm.GetResponseAndRegister("Patch", url, body, headers)
	if res.IsEmpty() {
		return
	}

	return res.Get(0).(*Response), res.GetError(1)
}

// Delete performs a DELETE request given the provided params and returns the response
func (cm *clientMock) Delete(url *URL, headers ...map[string]string) (r *Response, err error) {
	res := cm.GetResponseAndRegister("Delete", url, headers)
	if res.IsEmpty() {
		return
	}

	return res.Get(0).(*Response), res.GetError(1)
}
