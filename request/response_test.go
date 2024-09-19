// go:build unit
package request

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
