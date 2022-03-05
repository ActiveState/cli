package proccomm

import "github.com/ActiveState/cli/exp/pm/internal/socket"

var (
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
	return func(input string) (bool, string) {
		if input == KeyHTTPAddr {
			return true, addr
		}

		return false, ""
	}
}

func (c *Client) GetHTTPAddr() (string, error) {
	return c.s.Get(KeyHTTPAddr)
}
