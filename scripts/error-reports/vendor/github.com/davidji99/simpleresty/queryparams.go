package simpleresty

import (
	"github.com/davidji99/go-querystring/query"
	"net/url"
	"reflect"
)

// AddQueryParams takes a slice of opts and adds each field as escaped URL query parameters to a base URL string.
//
// Each element in opts must be a struct whose fields contain "url" tags.
//
// Based on: https://github.com/google/go-github/blob/master/github/github.go#L226
func AddQueryParams(baseURL string, opts ...interface{}) (string, error) {
	// Handle if opts is nil
	v := reflect.ValueOf(opts)
	if v.Kind() == reflect.Slice && v.IsNil() {
		return baseURL, nil
	}

	// Parse URL
	u, parseErr := url.Parse(baseURL)
	if parseErr != nil {
		return "", parseErr
	}

	fulQS := url.Values{}
	for _, opt := range opts {
		qs, err := query.Values(opt)
		if err != nil {
			return baseURL, err
		}

		for k, v := range qs {
			fulQS[k] = v
		}
	}

	u.RawQuery = fulQS.Encode()
	return u.String(), nil
}
