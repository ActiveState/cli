package httpreq

import (
	"context"
	"io/ioutil"
	"net/http"

	"github.com/ActiveState/cli/internal/errs"
)

type Client struct {
	HttpClient *http.Client
}

func New() *Client {
	return &Client{http.DefaultClient}
}

func (c *Client) Get(url string) ([]byte, error) {
	return c.GetWithContext(context.Background(), url)
}

func (c *Client) GetWithContext(ctx context.Context, url string) ([]byte, error) {
	resp, err := c.HttpClient.Get(url)
	if err != nil {
		return []byte{}, errs.Wrap(err, "Couldn't get url=%s", url)
	}
	defer resp.Body.Close()

	response, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, errs.New("Could not read body. Status: %s", resp.Status)
	}

	if resp.StatusCode != 200 {
		return nil, NewHTTPError(err, resp.StatusCode, response)
	}

	return response, nil
}

type HTTPError struct {
	err    error
	status int
	body   []byte
}

func NewHTTPError(err error, status int, body []byte) *HTTPError {
	return &HTTPError{
		err:    err,
		status: status,
		body:   body,
	}
}

func (e *HTTPError) Error() string {
	return e.err.Error()
}

func (e *HTTPError) Unwrap() error {
	return e.err
}

func (e *HTTPError) HTTPStatusCode() int {
	return e.status
}
