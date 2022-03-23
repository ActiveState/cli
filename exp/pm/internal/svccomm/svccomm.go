// Package svccomm contains common IPC handlers and requesters.
package svccomm

import (
	"context"

	"github.com/ActiveState/cli/exp/pm/internal/ipc"
)

var (
	KeyHTTPAddr = "http-addr"
)

type Client struct {
	s *ipc.Client
}

func NewClient(s *ipc.Client) *Client {
	return &Client{
		s: s,
	}
}

func HTTPAddrMHandler(addr string) ipc.MatchedHandler {
	return func(input string) (string, bool) {
		if input == KeyHTTPAddr {
			return addr, true
		}

		return "", false
	}
}

func (c *Client) GetHTTPAddr(ctx context.Context) (string, error) {
	return c.s.Get(ctx, KeyHTTPAddr)
}
