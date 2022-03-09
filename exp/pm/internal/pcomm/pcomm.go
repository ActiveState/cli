package pcomm

import "github.com/ActiveState/cli/exp/pm/internal/ipc"

var (
	KeyPing     = "ping"
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

func (c *Client) GetHTTPAddr() (string, error) {
	return c.s.Get(KeyHTTPAddr)
}

func PingHandler() ipc.MatchedHandler {
	return func(input string) (string, bool) {
		if input == KeyPing {
			return "pong", true
		}

		return "", false
	}
}
