// go:build unit
package request

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewURL(t *testing.T) {
	t.Run("Should return an empty URL if no raw url is provided", func(t *testing.T) {
		res := NewURL()

		assert.Equal(t, &URL{}, res)
	})
	t.Run("Should return an empty URL if an empty string as raw url is provided", func(t *testing.T) {
		res := NewURL("")

		assert.Equal(t, &URL{}, res)
	})
	t.Run("Should return an empty URL if an error occurred at parse url raw", func(t *testing.T) {
		res := NewURL("inv$@%lid")

		assert.Equal(t, &URL{}, res)
	})
	t.Run("Should return only path if no scheme is provided", func(t *testing.T) {
		res := NewURL("www.site.com/test")

		expectedURL := &URL{
			path:    "www.site.com/test",
			queries: url.Values{},
		}

		assert.Equal(t, expectedURL, res)
	})
	t.Run("Should return a URL with no queries if raw url does not contains queries", func(t *testing.T) {
		res := NewURL("https://www.site.com/test")

		expectedURL := &URL{
			scheme:  "https",
			host:    "www.site.com",
			path:    "/test",
			queries: url.Values{},
		}

		assert.Equal(t, expectedURL, res)
	})
	t.Run("Should return a full URL if raw url contains queries", func(t *testing.T) {
		res := NewURL("https://www.site.com/test?key=value")

		expectedURL := &URL{
			scheme: "https",
			host:   "www.site.com",
			path:   "/test",
			queries: url.Values{
				"key": []string{"value"},
			},
		}

		assert.Equal(t, expectedURL, res)
	})
}

func TestSetScheme(t *testing.T) {
	t.Run("Should do nothing if param is empty", func(t *testing.T) {
		u := NewURL()
		u.SetScheme("")

		assert.Empty(t, u.scheme)
	})
	t.Run("Should set URL scheme", func(t *testing.T) {
		u := NewURL()
		u.SetScheme("https")

		assert.Equal(t, "https", u.scheme)
	})
}

func TestSetHost(t *testing.T) {
	t.Run("Should do nothing if host is empty", func(t *testing.T) {
		u := NewURL()
		u.SetHost("")

		assert.Empty(t, u.host)
	})
	t.Run("Should set URL host", func(t *testing.T) {
		u := NewURL()
		u.SetHost("www.site.com")

		assert.Equal(t, "www.site.com", u.host)
	})
}

func TestSetPath(t *testing.T) {
	t.Run("Should do nothing if path is empty", func(t *testing.T) {
		u := NewURL()
		u.SetPath("")

		assert.Empty(t, u.path)
	})
	t.Run("Should set URL path", func(t *testing.T) {
		u := NewURL()
		u.SetPath("/test")

		assert.Equal(t, "/test", u.path)
	})
}

func TestAddQuery(t *testing.T) {
	t.Run("Should do nothing if key is empty", func(t *testing.T) {
		u := NewURL()
		u.AddQuery("", "value")

		assert.Empty(t, u.queries)
	})
	t.Run("Should do nothing if key is empty", func(t *testing.T) {
		u := NewURL()
		u.AddQuery("key", "")

		assert.Empty(t, u.queries)
	})
	t.Run("Should add a new query", func(t *testing.T) {
		u := NewURL()
		u.AddQuery("key", "value")

		expectedQueries := url.Values{
			"key": []string{"value"},
		}
		assert.Equal(t, expectedQueries, u.queries)
	})
}

func TestParse(t *testing.T) {
	t.Run("Should return an error if an error occurred at join path", func(t *testing.T) {
		u := &URL{
			host: "inv$@%lid",
			path: "/test",
		}

		raw, err := u.Parse()
		assert.Empty(t, raw)
		assert.NotNil(t, err)

		expectedErr := fmt.Errorf("unable to parse url with host %s and path %s", u.host, u.path)
		assert.Equal(t, expectedErr, err)
	})
	t.Run("Should return the parsed url without schema", func(t *testing.T) {
		u := &URL{
			host: "www.site.com",
			path: "/test",
		}

		raw, err := u.Parse()
		assert.Nil(t, err)

		expectedRaw := "www.site.com/test"
		assert.Equal(t, expectedRaw, raw)
	})
	t.Run("Should return the parsed url without queries", func(t *testing.T) {
		u := &URL{
			scheme: "https",
			host:   "www.site.com",
			path:   "/test",
		}

		raw, err := u.Parse()
		assert.Nil(t, err)

		expectedRaw := "https://www.site.com/test"
		assert.Equal(t, expectedRaw, raw)
	})
	t.Run("Should return the full parsed url", func(t *testing.T) {
		u := &URL{
			scheme: "https",
			host:   "www.site.com",
			path:   "/test",
			queries: url.Values{
				"key": []string{"value"},
			},
		}

		raw, err := u.Parse()
		assert.Nil(t, err)

		expectedRaw := "https://www.site.com/test?key=value"
		assert.Equal(t, expectedRaw, raw)
	})
}

func TestIsSuccessCode(t *testing.T) {
	t.Run("Should return false if response is an error response", func(t *testing.T) {
		res := Response{StatusCode: http.StatusBadGateway}

		assert.False(t, res.IsSuccessCode())
	})
	t.Run("Should return true if response is a success response", func(t *testing.T) {
		res := Response{StatusCode: http.StatusOK}

		assert.True(t, res.IsSuccessCode())
	})
}

func TestIsFailureCode(t *testing.T) {
	t.Run("Should return false if response is a success response", func(t *testing.T) {
		res := Response{StatusCode: http.StatusOK}

		assert.False(t, res.IsFailureCode())
	})
	t.Run("Should return true if response is an error response", func(t *testing.T) {
		res := Response{StatusCode: http.StatusBadRequest}

		assert.True(t, res.IsFailureCode())
	})
}

type Mock struct {
	Key string `json:"key"`
}

func TestDecodeJSON(t *testing.T) {
	mock := &Mock{Key: "value"}
	mockBytes, _ := json.Marshal(mock)
	mockStr := strings.NewReader(string(mockBytes))

	t.Run("Should return nil if decode succeeds", func(t *testing.T) {
		res := Response{Body: io.NopCloser(mockStr)}
		defer res.Body.Close()

		m := Mock{}

		err := res.DecodeJSON(&m)
		assert.Nil(t, err)

		expectedDecoded := Mock{Key: "value"}
		assert.Equal(t, expectedDecoded, m)
	})
	t.Run("Should return an error if value is nil", func(t *testing.T) {
		res := Response{Body: io.NopCloser(mockStr)}
		defer res.Body.Close()

		err := res.DecodeJSON(nil)
		assert.NotNil(t, err)

		expectedErr := errors.New("value must be a valid pointer of a struct")
		assert.Equal(t, expectedErr, err)
	})
	t.Run("Should return an error if value is not a pointer", func(t *testing.T) {
		res := Response{Body: io.NopCloser(mockStr)}
		defer res.Body.Close()

		m := Mock{}
		err := res.DecodeJSON(m)
		assert.NotNil(t, err)

		expectedErr := errors.New("value must be a valid pointer of a struct")
		assert.Equal(t, expectedErr, err)
	})
	t.Run("Should return an error if an error occurred at decode", func(t *testing.T) {
		res := Response{Body: io.NopCloser(mockStr)}
		defer res.Body.Close()

		m := []string{}

		err := res.DecodeJSON(&m)
		assert.NotNil(t, err)
		assert.Error(t, err, "json: cannot unmarshal object into Go value of type []string")
	})
}

// TODO: implement Do(), Get(), Post(), Put(), Patch() and Delete() tests
