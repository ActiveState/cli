package retryhttp

import (
	"context"
)

type RetryHTTP struct {
	Client  *Client
	Context context.Context
	cancel  context.CancelFunc
}

func New(client *Client) *RetryHTTP {
	timeout := client.HTTPClient.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	return &RetryHTTP{
		Client:  client,
		Context: ctx,
		cancel:  cancel,
	}
}

func (rh *RetryHTTP) Close() {
	if rh.cancel != nil {
		rh.cancel()
	}
}
