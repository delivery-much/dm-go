// go:build unit
package request

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/delivery-much/mock-helper/mock"
	"github.com/stretchr/testify/assert"
)

type httpClientMock struct {
	mock.Mock
}

func NewHttpClientMock() *httpClientMock {
	return &httpClientMock{mock.NewMock()}
}

func (m *httpClientMock) Do(req *http.Request) (r *http.Response, err error) {
	res := m.GetResponseAndRegister("Do", req)
	if res.IsEmpty() {
		return
	}

	return res.Get(0).(*http.Response), res.GetError(1)
}

func TestDo(t *testing.T) {
	urlMock := NewURL("http://localhost")
	emptyResMock := &http.Response{}

	t.Run("Should return an error if an error occurred at mount request", func(t *testing.T) {
		c := Client{}
		res, err := c.Do(Params{
			Method: "INVALID",
			URL:    urlMock,
		})
		assert.Nil(t, res)
		fmt.Println(err.Error())
		assert.NotNil(t, err)
	})
	t.Run("Should return an error if an error occurred at do request", func(t *testing.T) {
		errMock := errors.New("Do error")

		hcm := NewHttpClientMock()
		hcm.SetMethodResponse("Do", emptyResMock, errMock)

		var clientMock httpClientInterface = hcm

		adapterMock := NewAdapterMock()
		adapterMock.SetMethodResponse("Adapt", clientMock)

		httpClientAdapter = adapterMock

		c := Client{}

		res, err := c.Do(Params{
			Method: "GET",
			URL:    urlMock,
		})
		assert.Nil(t, res)
		assert.NotNil(t, err)
		assert.Equal(t, errMock, err)
	})
	t.Run("Should return the response correclty", func(t *testing.T) {
		resMock := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"hello": "world"}`)),
		}

		hcm := NewHttpClientMock()
		hcm.SetMethodResponse("Do", resMock, nil)

		var clientMock httpClientInterface = hcm

		adapterMock := NewAdapterMock()
		adapterMock.SetMethodResponse("Adapt", clientMock)

		httpClientAdapter = adapterMock

		c := Client{}

		res, err := c.Do(Params{
			Method: "GET",
			URL:    urlMock,
		})
		assert.Nil(t, err)
		assert.NotNil(t, res)

		expectedRes := (*Response)(res)
		assert.Equal(t, expectedRes, res)
	})
	t.Run("Should call adapter correctly", func(t *testing.T) {
		resMock := &http.Response{StatusCode: http.StatusNoContent}

		hcm := NewHttpClientMock()
		hcm.SetMethodResponse("Do", resMock, nil)

		var clientMock httpClientInterface = hcm

		adapterMock := NewAdapterMock()
		adapterMock.SetMethodResponse("Adapt", clientMock)

		httpClientAdapter = adapterMock

		c := Client{}

		c.Do(Params{
			Method: "GET",
			URL:    urlMock,
		})

		adapterMock.
			Assert(t).
			CalledOnce().
			And().
			CalledWith(&c)
	})
	t.Run("Should call http client correctly", func(t *testing.T) {
		resMock := &http.Response{StatusCode: http.StatusNoContent}

		hcm := NewHttpClientMock()
		hcm.SetMethodResponse("Do", resMock, nil)

		var clientMock httpClientInterface = hcm

		adapterMock := NewAdapterMock()
		adapterMock.SetMethodResponse("Adapt", clientMock)

		httpClientAdapter = adapterMock

		c := Client{}

		paramsMock := Params{
			Method: "POST",
			URL:    urlMock,
			Headers: map[string]string{
				"key": "value",
			},
			Body: io.NopCloser(strings.NewReader(`{"hello": "world"}`)),
		}

		c.Do(paramsMock)

		expectedReq, err := http.NewRequest(
			paramsMock.Method,
			paramsMock.URL.String(),
			paramsMock.Body,
		)
		assert.Nil(t, err)

		expectedReq.Header.Set("key", "value")

		hcm.
			Assert(t).
			CalledOnce().
			And().
			CalledWith(expectedReq)
	})
}

func TestGet(t *testing.T) {
	urlMock := NewURL("http://localhost")
	emptyResMock := &http.Response{}

	t.Run("Should return an error if client fails", func(t *testing.T) {
		errMock := errors.New("Do error")

		hcm := NewHttpClientMock()
		hcm.SetMethodResponse("Do", emptyResMock, errMock)

		var clientMock httpClientInterface = hcm

		adapterMock := NewAdapterMock()
		adapterMock.SetMethodResponse("Adapt", clientMock)

		httpClientAdapter = adapterMock

		c := Client{}

		res, err := c.Get(urlMock)
		assert.Nil(t, res)
		assert.NotNil(t, err)
		assert.Equal(t, errMock, err)
	})
	t.Run("Should return a valid response if request succeeds", func(t *testing.T) {
		resMock := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"hello": "world"}`)),
		}

		hcm := NewHttpClientMock()
		hcm.SetMethodResponse("Do", resMock, nil)

		var clientMock httpClientInterface = hcm

		adapterMock := NewAdapterMock()
		adapterMock.SetMethodResponse("Adapt", clientMock)

		httpClientAdapter = adapterMock

		c := Client{}

		res, err := c.Get(urlMock)
		assert.Nil(t, err)
		assert.NotNil(t, res)

		expectedRes := (*Response)(res)
		assert.Equal(t, expectedRes, res)
	})
}

func TestPost(t *testing.T) {
	urlMock := NewURL("http://localhost")
	emptyResMock := &http.Response{}
	bodyMock := strings.NewReader(`{"name": "john doe"}`)

	t.Run("Should return an error if client fails", func(t *testing.T) {
		errMock := errors.New("Do error")

		hcm := NewHttpClientMock()
		hcm.SetMethodResponse("Do", emptyResMock, errMock)

		var clientMock httpClientInterface = hcm

		adapterMock := NewAdapterMock()
		adapterMock.SetMethodResponse("Adapt", clientMock)

		httpClientAdapter = adapterMock

		c := Client{}

		res, err := c.Post(urlMock, bodyMock)
		assert.Nil(t, res)
		assert.NotNil(t, err)
		assert.Equal(t, errMock, err)
	})
	t.Run("Should return a valid response if request succeeds", func(t *testing.T) {
		resMock := &http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(strings.NewReader(`{"id": 1, "name": "jonh doe"}`)),
		}

		hcm := NewHttpClientMock()
		hcm.SetMethodResponse("Do", resMock, nil)

		var clientMock httpClientInterface = hcm

		adapterMock := NewAdapterMock()
		adapterMock.SetMethodResponse("Adapt", clientMock)

		httpClientAdapter = adapterMock

		c := Client{}

		res, err := c.Post(urlMock, bodyMock)
		assert.Nil(t, err)
		assert.NotNil(t, res)

		expectedRes := (*Response)(res)
		assert.Equal(t, expectedRes, res)
	})
}

func TestPut(t *testing.T) {
	urlMock := NewURL("http://localhost")
	emptyResMock := &http.Response{}
	bodyMock := strings.NewReader(`{"id": 1, "name": "john doe"}`)

	t.Run("Should return an error if client fails", func(t *testing.T) {
		errMock := errors.New("Do error")

		hcm := NewHttpClientMock()
		hcm.SetMethodResponse("Do", emptyResMock, errMock)

		var clientMock httpClientInterface = hcm

		adapterMock := NewAdapterMock()
		adapterMock.SetMethodResponse("Adapt", clientMock)

		httpClientAdapter = adapterMock

		c := Client{}

		res, err := c.Put(urlMock, bodyMock)
		assert.Nil(t, res)
		assert.NotNil(t, err)
		assert.Equal(t, errMock, err)
	})
	t.Run("Should return a valid response if request succeeds", func(t *testing.T) {
		resMock := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"id": 1, "name": "jonh doe"}`)),
		}

		hcm := NewHttpClientMock()
		hcm.SetMethodResponse("Do", resMock, nil)

		var clientMock httpClientInterface = hcm

		adapterMock := NewAdapterMock()
		adapterMock.SetMethodResponse("Adapt", clientMock)

		httpClientAdapter = adapterMock

		c := Client{}

		res, err := c.Put(urlMock, bodyMock)
		assert.Nil(t, err)
		assert.NotNil(t, res)

		expectedRes := (*Response)(res)
		assert.Equal(t, expectedRes, res)
	})
}

func TestPatch(t *testing.T) {
	urlMock := NewURL("http://localhost")
	emptyResMock := &http.Response{}
	bodyMock := strings.NewReader(`{"name": "john doe"}`)

	t.Run("Should return an error if client fails", func(t *testing.T) {
		errMock := errors.New("Do error")

		hcm := NewHttpClientMock()
		hcm.SetMethodResponse("Do", emptyResMock, errMock)

		var clientMock httpClientInterface = hcm

		adapterMock := NewAdapterMock()
		adapterMock.SetMethodResponse("Adapt", clientMock)

		httpClientAdapter = adapterMock

		c := Client{}

		res, err := c.Patch(urlMock, bodyMock)
		assert.Nil(t, res)
		assert.NotNil(t, err)
		assert.Equal(t, errMock, err)
	})
	t.Run("Should return a valid response if request succeeds", func(t *testing.T) {
		resMock := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"name": "john doe"}`)),
		}

		hcm := NewHttpClientMock()
		hcm.SetMethodResponse("Do", resMock, nil)

		var clientMock httpClientInterface = hcm

		adapterMock := NewAdapterMock()
		adapterMock.SetMethodResponse("Adapt", clientMock)

		httpClientAdapter = adapterMock

		c := Client{}

		res, err := c.Patch(urlMock, bodyMock)
		assert.Nil(t, err)
		assert.NotNil(t, res)

		expectedRes := (*Response)(res)
		assert.Equal(t, expectedRes, res)
	})
}

func TestDelete(t *testing.T) {
	urlMock := NewURL("http://localhost")
	emptyResMock := &http.Response{}

	t.Run("Should return an error if client fails", func(t *testing.T) {
		errMock := errors.New("Do error")

		hcm := NewHttpClientMock()
		hcm.SetMethodResponse("Do", emptyResMock, errMock)

		var clientMock httpClientInterface = hcm

		adapterMock := NewAdapterMock()
		adapterMock.SetMethodResponse("Adapt", clientMock)

		httpClientAdapter = adapterMock

		c := Client{}

		res, err := c.Delete(urlMock)
		assert.Nil(t, res)
		assert.NotNil(t, err)
		assert.Equal(t, errMock, err)
	})
	t.Run("Should return a valid response if request succeeds", func(t *testing.T) {
		resMock := &http.Response{
			StatusCode: http.StatusNoContent,
		}

		hcm := NewHttpClientMock()
		hcm.SetMethodResponse("Do", resMock, nil)

		var clientMock httpClientInterface = hcm

		adapterMock := NewAdapterMock()
		adapterMock.SetMethodResponse("Adapt", clientMock)

		httpClientAdapter = adapterMock
		c := Client{}

		res, err := c.Delete(urlMock)
		assert.Nil(t, err)
		assert.NotNil(t, res)

		expectedRes := (*Response)(res)
		assert.Equal(t, expectedRes, res)
	})
}
