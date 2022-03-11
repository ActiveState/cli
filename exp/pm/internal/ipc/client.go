package ipc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"time"
)

type Client struct {
	n *Namespace
	d net.Dialer
}

func NewClient(n *Namespace) *Client {
	// TODO: move ping and error return here
	return &Client{
		n: n,
	}
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	emsg := "client: get: %w"

	conn, err := c.d.DialContext(ctx, network, c.n.String())
	if err != nil {
		if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ENOENT) { // should handler per platform
			return "", fmt.Errorf(emsg, ErrServerDown)
		}
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

func (c *Client) Ping(ctx context.Context) (time.Duration, error) {
	start := time.Now()
	emsg := "client: ping: %w"

	if _, err := getPing(ctx, c); err != nil {
		return 0, fmt.Errorf(emsg, err)
	}

	return time.Since(start), nil
}
