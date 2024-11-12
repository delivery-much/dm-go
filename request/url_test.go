// go:build unit
package request

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewURL(t *testing.T) {
	t.Run("Should return an empty URL if parse raw url fails", func(t *testing.T) {
		url := NewURL("wr\\%\\+ong")

		assert.Equal(t, &URL{}, url)
	})
	t.Run("Should return a URL if parse succeeds", func(t *testing.T) {
		url := NewURL("http://localhost:8080/test?key=value")

		expectedURL := &URL{
			Scheme: "http",
			Host:   "localhost:8080",
			Path:   "/test",
			Query:  map[string][]string{"key": []string{"value"}},
		}

		assert.Equal(t, expectedURL, url)
	})
}

func TestAddQuery(t *testing.T) {
	t.Run("Should do nothing if key is empty", func(t *testing.T) {
		url := NewURL("http://localhost:8080/test")

		url.AddQuery("")

		expectedURL := &URL{
			Scheme: "http",
			Host:   "localhost:8080",
			Path:   "/test",
			Query:  map[string][]string{},
		}

		assert.Equal(t, expectedURL, url)
	})
	t.Run("Should do nothing if value is empty", func(t *testing.T) {
		url := NewURL("http://localhost:8080/test")

		url.AddQuery("key")

		expectedURL := &URL{
			Scheme: "http",
			Host:   "localhost:8080",
			Path:   "/test",
			Query:  map[string][]string{},
		}

		assert.Equal(t, expectedURL, url)
	})
	t.Run("Should add the query to URL", func(t *testing.T) {
		// case 1: URL with no query
		url := &URL{
			Scheme: "http",
			Host:   "localhost:8080",
			Path:   "/test",
		}

		url.AddQuery("key", "value")

		expectedURL := &URL{
			Scheme: "http",
			Host:   "localhost:8080",
			Path:   "/test",
			Query:  map[string][]string{"key": {"value"}},
		}

		assert.Equal(t, expectedURL, url)

		// case 2: URL with query
		url.AddQuery("other_key", "other_value")
		expectedURL = &URL{
			Scheme: "http",
			Host:   "localhost:8080",
			Path:   "/test",
			Query: map[string][]string{
				"key":       {"value"},
				"other_key": {"other_value"},
			},
		}

		assert.Equal(t, expectedURL, url)
	})
}

func TestString(t *testing.T) {
	t.Run("Should return the URL as a string raw url", func(t *testing.T) {
		url := NewURL("http://localhost:8080/test")
		url.AddQuery("name", "John Doe")
		url.AddQuery("fields", "field1", "field2")

		rawURL := url.String()

		expectedRawURL := "http://localhost:8080/test?fields=field1&fields=field2&name=John%20Doe"
		assert.Equal(t, expectedRawURL, rawURL)
	})
}
