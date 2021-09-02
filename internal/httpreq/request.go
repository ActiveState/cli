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

func (c *Client) Get(url string) ([]byte, int, error) {
	return c.GetWithContext(context.Background(), url)
}

func (c *Client) GetWithContext(ctx context.Context, url string) ([]byte, int, error) {
	resp, err := c.HttpClient.Get(url)
	if err != nil {
		return []byte{}, 0, errs.Wrap(err, "Couldn't get url=%s", url)
	}
	defer resp.Body.Close()

	code := 0
	if resp != nil {
		code = resp.StatusCode
	}

	response, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, code, errs.New("Could not read body. Status: %s", resp.Status)
	}

	if resp.StatusCode != 200 {
		return response, code, errs.New("bad http status from %s: %v, body: %s", url, resp.Status, response)
	}

	return response, code, nil
}
