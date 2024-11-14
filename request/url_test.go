// go:build unit
package request

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewURL(t *testing.T) {
	t.Run("Should return an empty URL if parse raw url fails", func(t *testing.T) {
		url := ParseURL("wr\\%\\+ong")

		assert.Equal(t, &URL{}, url)
	})
	t.Run("Should return a URL if parse succeeds", func(t *testing.T) {
		url := ParseURL("http://localhost:8080/test?key=value")

		expectedURL, _ := url.Parse("http://localhost:8080/test?key=value")

		assert.Equal(t, &URL{expectedURL}, url)
	})
}

func TestAddQuery(t *testing.T) {
	t.Run("Should do nothing if url is not defined", func(t *testing.T) {
		url := &URL{}

		url.AddQuery("key", "value")

		assert.Equal(t, &URL{}, url)
	})
	t.Run("Should do nothing if key is empty", func(t *testing.T) {
		url := ParseURL("http://localhost:8080/test")

		url.AddQuery("")

		expectedURL, _ := url.Parse("http://localhost:8080/test")

		assert.Equal(t, &URL{expectedURL}, url)
	})
	t.Run("Should do nothing if value is empty", func(t *testing.T) {
		url := ParseURL("http://localhost:8080/test")

		url.AddQuery("key")

		expectedURL, _ := url.Parse("http://localhost:8080/test")

		assert.Equal(t, &URL{expectedURL}, url)
	})

	t.Run("Should add the query to URL", func(t *testing.T) {
		url := ParseURL("http://localhost:8080/test")

		url.AddQuery("key", "value")

		expectedURL, _ := url.Parse("http://localhost:8080/test?key=value")

		assert.Equal(t, &URL{expectedURL}, url)
	})
}
