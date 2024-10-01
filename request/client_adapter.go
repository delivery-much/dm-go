package request

import "net/http"

type httpClientInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

type clientAdapterInterface interface {
	Adapt(c *Client) httpClientInterface
}

type clientAdapter struct{}

func (ca *clientAdapter) Adapt(c *Client) httpClientInterface {
	return (*http.Client)(c)
}
