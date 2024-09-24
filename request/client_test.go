// go:build unit
package request

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type httpClientMock struct {
	err error
	res *http.Response
}

func (hcm *httpClientMock) Do(req *http.Request) (res *http.Response, err error) {
	if hcm.err != nil {
		err = hcm.err
		return
	}

	res = hcm.res
	return
}

func TestDo(t *testing.T) {
	urlMock, _ := url.Parse("http://localhost")

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

		var clientMock httpClientInterface = &httpClientMock{
			err: errMock,
		}

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

		var clientMock httpClientInterface = &httpClientMock{
			res: resMock,
		}

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
		var clientMock httpClientInterface = &httpClientMock{
			res: &http.Response{StatusCode: http.StatusNoContent},
		}

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
}

func TestGet(t *testing.T) {
	urlMock, _ := url.Parse("http://localhost")

	t.Run("Should return an error if client fails", func(t *testing.T) {
		errMock := errors.New("Do error")

		var clientMock httpClientInterface = &httpClientMock{
			err: errMock,
		}

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

		var clientMock httpClientInterface = &httpClientMock{
			res: resMock,
		}

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
	urlMock, _ := url.Parse("http://localhost")
	bodyMock := strings.NewReader(`{"name": "john doe"}`)

	t.Run("Should return an error if client fails", func(t *testing.T) {
		errMock := errors.New("Do error")

		var clientMock httpClientInterface = &httpClientMock{
			err: errMock,
		}

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

		var clientMock httpClientInterface = &httpClientMock{
			res: resMock,
		}

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
	urlMock, _ := url.Parse("http://localhost")
	bodyMock := strings.NewReader(`{"id": 1, "name": "john doe"}`)

	t.Run("Should return an error if client fails", func(t *testing.T) {
		errMock := errors.New("Do error")

		var clientMock httpClientInterface = &httpClientMock{
			err: errMock,
		}

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

		var clientMock httpClientInterface = &httpClientMock{
			res: resMock,
		}

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
	urlMock, _ := url.Parse("http://localhost")
	bodyMock := strings.NewReader(`{"name": "john doe"}`)

	t.Run("Should return an error if client fails", func(t *testing.T) {
		errMock := errors.New("Do error")

		var clientMock httpClientInterface = &httpClientMock{
			err: errMock,
		}

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

		var clientMock httpClientInterface = &httpClientMock{
			res: resMock,
		}

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
	urlMock, _ := url.Parse("http://localhost")

	t.Run("Should return an error if client fails", func(t *testing.T) {
		errMock := errors.New("Do error")

		var clientMock httpClientInterface = &httpClientMock{
			err: errMock,
		}

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

		var clientMock httpClientInterface = &httpClientMock{
			res: resMock,
		}

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
