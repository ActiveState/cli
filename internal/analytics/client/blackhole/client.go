package blackhole

import (
	"github.com/ActiveState/cli/internal/analytics"
)

type Client struct{}

var _ analytics.Dispatcher = &Client{}

func New() *Client {
	return &Client{}
}

func (c Client) Event(category string, action string, dim ...*analytics.Dimensions) {
}

func (c Client) EventWithLabel(category string, action string, label string, dim ...*analytics.Dimensions) {
}

func (c Client) Wait() {
}

func (c Client) Close() {
}
