package simpleresty

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"strings"
)

const (
	GetMethod    = "GET"
	PostMethod   = "POST"
	PutMethod    = "PUT"
	DeleteMethod = "DELETE"
	PatchMethod  = "PATCH"
)

// Client represents a SimpleResty client. It embeds the resty.client so users have access to its methods.
type Client struct {
	*resty.Client

	// baseURL for the API endpoint. Please include a trailing slash '/'.
	baseURL string

	// noProxyDomains contains a list of domains that don't require a proxy.
	noProxyDomains []string

	// proxyURL represents a proxy URL.
	proxyURL *string

	// shouldSetProxy stores a boolean value that determines if a proxy needs to be used.
	shouldSetProxy bool
}

// Dispatch method is a wrapper around the send method which
// performs the HTTP request using the method and URL already defined.
func (c *Client) Dispatch(request *resty.Request) (*Response, error) {
	// Set Proxy if applicable
	c.determineSetProxy()

	response, err := request.Send()

	if err != nil {
		return nil, err
	}

	return checkResponse(response)
}

// Get executes a HTTP GET request.
func (c *Client) Get(url string, r, body interface{}) (*Response, error) {
	req := c.ConstructRequest(r, body)

	response, getErr := req.Get(url)
	if getErr != nil {
		return nil, getErr
	}

	return checkResponse(response)
}

// Post executes a HTTP POST request.
func (c *Client) Post(url string, r, body interface{}) (*Response, error) {
	req := c.ConstructRequest(r, body)

	response, postErr := req.Post(url)
	if postErr != nil {
		return nil, postErr
	}

	return checkResponse(response)
}

// Put executes a HTTP PUT request.
func (c *Client) Put(url string, r, body interface{}) (*Response, error) {
	req := c.ConstructRequest(r, body)

	response, putErr := req.Put(url)
	if putErr != nil {
		return nil, putErr
	}

	return checkResponse(response)
}

// Patch executes a HTTP PATCH request.
func (c *Client) Patch(url string, r, body interface{}) (*Response, error) {
	req := c.ConstructRequest(r, body)

	response, patchErr := req.Patch(url)
	if patchErr != nil {
		return nil, patchErr
	}

	return checkResponse(response)
}

// Delete executes a HTTP DELETE request.
func (c *Client) Delete(url string, r, body interface{}) (*Response, error) {
	req := c.ConstructRequest(r, body)

	response, deleteErr := req.Delete(url)
	if deleteErr != nil {
		return nil, deleteErr
	}

	return checkResponse(response)
}

// ConstructRequest creates a new request.
func (c *Client) ConstructRequest(r, body interface{}) *resty.Request {
	// Set Proxy if applicable
	c.determineSetProxy()

	req := c.R().SetBody(body)

	if r != nil {
		req.SetResult(r)
	}

	return req
}

// RequestURL appends the template argument to the base URL and returns the full request URL.
func (c *Client) RequestURL(template string, args ...interface{}) string {
	// Validate to make sure baseURL is set
	if c.baseURL == "" {
		panic("base URL not set")
	}

	if len(args) == 1 && args[0] == "" {
		return c.baseURL + template
	}
	return c.baseURL + fmt.Sprintf(template, args...)
}

// RequestURLWithQueryParams first constructs the request URL and then appends any URL encoded query parameters.
//
// This function operates nearly the same as RequestURL
func (c *Client) RequestURLWithQueryParams(url string, opts ...interface{}) (string, error) {
	u := c.RequestURL(url)
	return AddQueryParams(u, opts...)
}

// SetBaseURL sets the base url for the client.
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

// determineSetProxy first checks if proxy is already set or not. If it is, this method returns early.
//
// If no proxy is set, it will be set if the proxyURL is defined and the base domain is not present
// in the noProxyDomains string array.
func (c *Client) determineSetProxy() {
	// If proxy is already set in a previous execution, short circuit this method call.
	if c.IsProxySet() {
		return
	}

	if c.proxyURL != nil {
		c.shouldSetProxy = true

		// Loop through noProxyDomains and check if the base url doesn't need a proxy set for the Client.
		for _, d := range c.noProxyDomains {
			if strings.Contains(strings.ToLower(c.baseURL), d) {
				c.shouldSetProxy = false
				break
			}
		}

		if c.shouldSetProxy {
			c.SetProxy(*c.proxyURL)
		}
	}
}
