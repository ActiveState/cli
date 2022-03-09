package proccomm

import "github.com/ActiveState/cli/exp/pm/internal/socket"

var (
	KeyPing     = "ping"
	KeyHTTPAddr = "http-addr"
)

type Client struct {
	s *socket.Client
}

func NewClient(s *socket.Client) *Client {
	return &Client{
		s: s,
	}
}

func HTTPAddrMHandler(addr string) socket.MatchedHandler {
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

func PingHandler() socket.MatchedHandler {
	return func(input string) (string, bool) {
		if input == KeyPing {
			return "pong", true
		}

		return "", false
	}
}
