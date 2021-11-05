package simpleresty

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"net/url"
)

// Response represents the response after executing a HTTP request.
type Response struct {
	// Request representss the response's original request.
	Request *resty.Request

	// RequestURL is the request URL.
	RequestURL string

	// RequestMethod is the request method such as GET.
	RequestMethod string

	// Request body is the request body in JSON string format.
	RequestBody string

	// Resp represents the entire HTTP response.
	Resp *resty.Response

	// Status is the response status in string format such as '200 OK'.
	Status string

	// StatusCode is response status in integer format such as 200.
	StatusCode int

	// Body is the response body in JSON String format.
	Body string
}

// checkResponse parses the HTTP response and returns the response and an error if applicable.
func checkResponse(resp *resty.Response) (*Response, error) {
	path, _ := url.QueryUnescape(resp.Request.URL)
	r := &Response{Status: resp.Status(), StatusCode: resp.StatusCode(),
		Body: string(resp.Body()), Resp: resp, RequestURL: path,
		RequestMethod: resp.Request.Method, Request: resp.Request}

	// Convert the request body to a string.
	reqBody, marshallErr := json.Marshal(resp.Request.Body)
	if marshallErr != nil {
		return nil, marshallErr
	}
	r.RequestBody = string(reqBody)

	// If response is any of the below, return early.
	switch r.StatusCode {
	case 200, 201, 202, 204, 304:
		return r, nil
	}

	// Otherwise, return the response along with the error.
	return r, fmt.Errorf("%s %s: %d %s", r.RequestMethod, r.RequestURL, r.StatusCode, r.Body)
}
