package rollrest

import (
	"fmt"
	"github.com/davidji99/simpleresty"
)

// Option is a functional option for configuring the API client.
type Option func(*Client) error

func HTTP(http *simpleresty.Client) Option {
	return func(c *Client) error {
		c.http = http
		return nil
	}
}

// UserAgent allows overriding of the default User Agent.
func UserAgent(userAgent string) Option {
	return func(c *Client) error {
		c.userAgent = userAgent
		return nil
	}
}

// BaseURL allows overriding of the default base API URL.
func BaseURL(baseURL string) Option {
	return func(c *Client) error {
		// Validate that there is no trailing slashes before setting the custom baseURL
		if baseURL[len(baseURL)-1:] == "/" {
			return fmt.Errorf("custom base URL cannot contain a trailing slash")
		}

		c.baseURL = baseURL
		return nil
	}
}

// CustomHTTPHeaders sets additional HTTPHeaders
func CustomHTTPHeaders(headers map[string]string) Option {
	return func(c *Client) error {
		c.customHTTPHeaders = headers
		return nil
	}
}

// AuthAAT sets the account access token for authentication.
func AuthAAT(token string) Option {
	return func(c *Client) error {
		c.accountAccessToken = token
		return nil
	}
}

// AuthPAT sets the project access token for authentication.
func AuthPAT(token string) Option {
	return func(c *Client) error {
		c.projectAccessToken = token
		return nil
	}
}
