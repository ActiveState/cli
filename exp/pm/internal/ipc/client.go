package ipc

import (
	"fmt"
	"net"
	"time"
)

type Client struct {
	n *Namespace
}

func NewClient(n *Namespace) *Client {
	// TODO: move ping and error return here
	return &Client{
		n: n,
	}
}

func (c *Client) Get(key string) (string, error) {
	emsg := "client: get: %w"

	conn, err := net.Dial(network, c.n.String())
	if err != nil {
		return "", fmt.Errorf(emsg, err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte(key))
	if err != nil {
		return "", fmt.Errorf(emsg, err)
	}

	buf := make([]byte, msgWidth)
	n, _ := conn.Read(buf) //nolint // add error and timeout handling

	msg := string(buf[:n])

	return msg, nil
}

func (c *Client) Namespace() *Namespace {
	return c.n
}

func (c *Client) Ping() (time.Duration, error) {
	emsg := "client: ping: %w"

	start := time.Now()
	if _, err := getPing(c); err != nil {
		return 0, fmt.Errorf(emsg, err)
	}

	return time.Since(start), nil
}
