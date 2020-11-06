package analytics

import (
	ga "github.com/ActiveState/go-ogle-analytics"
)

type pageview struct {
}

func newPageview() *pageview {
	return &pageview{}
}

func (p *pageview) send(c *ga.Client) error {
	return c.Send(ga.NewPageview())
}
