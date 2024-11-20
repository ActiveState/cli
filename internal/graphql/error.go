package graphql

import "fmt"

type RequestError struct {
	Request Request
	err     error
}

func NewRequestError(err error, req Request) *RequestError {
	return &RequestError{
		Request: req,
		err:     err,
	}
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("Request failed: %v", e.err)
}

func (e *RequestError) Unwrap() error {
	return e.err
}
