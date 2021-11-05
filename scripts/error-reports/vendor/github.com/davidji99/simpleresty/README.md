[![Actions Status](https://github.com/davidji99/simpleresty/workflows/ci/badge.svg)](https://github.com/davidji99/simpleresty/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/davidji99/simpleresty)](https://goreportcard.com/report/github.com/davidji99/simpleresty)
<a href="LICENSE"><img src="https://img.shields.io/github/license/davidji99/simpleresty.svg" alt="License"></a>

A simple wrapper around [go-resty](https://github.com/go-resty/resty).

## Background
Having used [go-resty](https://github.com/go-resty/resty) to create clients for various service APIs,
I noticed a common set of methods/functions I would define in each API client. I extracted those methods/functions
and moved them into this separate library. This way, all my clients could benefit from using a single library to
interface with the APIs.

I have embedded `resty.Client` into `simpleresty.Client` so all of `resty`'s functions/methods are available to the user.
In fact, `simpleresty.New()` returns a `resty.Client`.

## Example
```go
package main

import (
	"fmt"
	"github.com/davidji99/simpleresty"
)

type GithubAPIResponse struct {
	CurrentUserURL string `json:"current_user_url,omitempty"`
}

func main() {
	c := simpleresty.New()

	var result *GithubAPIResponse
	response, getErr := c.Get("https://api.github.com", &result, nil)
	if getErr != nil {
		panic(getErr)
	}

	fmt.Println(response.StatusCode) // Returns 200
	fmt.Println(result.CurrentUserURL) // Returns 'https://api.github.com/user'
}
```

Additional examples can be found in the `/examples` folder.

You can also check out [rollrest-go](https://github.com/davidji99/rollrest-go), which uses this library to implement
an API rest client for Rollbar.

## Proxy

`simpleresty` respects any proxy URLs set in your environment in this order of preference:
1. `HTTPS_PROXY`
1. `https_proxy`
1. `HTTP_PROXY`
1. `http_proxy`

Only a single value from one of the above four environment variables will be used to set the proxy URL on the `Client`.

`simpleresty` will also respect domains (not IP addresses or CIDR ranges) defined by the `NO_PROXY` or `no_proxy`
environment variable. Multiple domains must be separated by a comma.

## Go-Resty
As this pkg is a thin wrapper around go-resty, all of its methods are available to use in this package.
Please refer to [go-resty's documentation](https://github.com/go-resty/resty) for more information.