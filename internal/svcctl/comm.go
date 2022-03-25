// Package svccomm contains common IPC handlers and requesters.
package svcctl

import (
	"context"

	"github.com/ActiveState/cli/internal/ipc"
)

var (
	KeyHTTPAddr = "http-addr"
)

type Getter interface {
	Get(ctx context.Context, key string) (value string, err error)
}

type Comm struct {
	g Getter
}

func NewComm(g Getter) *Comm {
	return &Comm{
		g: g,
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

func (c *Comm) GetHTTPAddr(ctx context.Context) (string, error) {
	return c.g.Get(ctx, KeyHTTPAddr)
}
