package blackhole

import (
	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
)

type Client struct{}

var _ analytics.Dispatcher = &Client{}

func New() *Client {
	return &Client{}
}

func (c Client) Event(category, action string, dim ...*dimensions.Values) {
}

func (c Client) EventWithLabel(category, action, label string, dim ...*dimensions.Values) {
}

func (c Client) EventWithSource(category, action, source string, dim ...*dimensions.Values) {
}

func (c Client) Wait() {
}

func (c Client) Close() {
}
