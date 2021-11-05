package simpleresty

import (
	"github.com/go-resty/resty/v2"
)

// New function creates a new simpleresty client with base url set to empty string.
//
// Users can set the base string later in their code.
func New() *Client {
	c := NewWithBaseURL("")
	return c
}

// NewWithBaseURL creates a new simpleresty client with base url set.
func NewWithBaseURL(url string) *Client {
	c := &Client{Client: resty.New(), baseURL: url, proxyURL: nil, shouldSetProxy: false}

	// Set no proxy domains if any
	c.noProxyDomains, _ = getNoProxyDomains()

	// Set proxy URL if any
	c.proxyURL = getProxyURL()

	return c
}
