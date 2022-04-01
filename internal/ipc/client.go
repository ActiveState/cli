package ipc

import (
	"context"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/ipc/internal/flisten"
)

type Client struct {
	namespace *Namespace
	dialer    flisten.Dialer
}

func NewClient(n *Namespace) *Client {
	return &Client{
		namespace: n,
	}
}

func (c *Client) Request(ctx context.Context, key string) (string, error) {
	ns := c.namespace.String()
	conn, err := c.dialer.DialContext(ctx, network, ns)
	if err != nil {
		err = asServerDownError(err)
		return "", errs.Wrap(err, "Cannot connect to ipc via %q", ns)
	}
	defer conn.Close()

	_, err = conn.Write([]byte(key))
	if err != nil {
		return "", errs.Wrap(err, "Failed to write to connection")
	}

	buf := make([]byte, msgWidth)
	n, err := conn.Read(buf)
	if err != nil {
		return "", errs.Wrap(err, "Failed to read from connection")
	}

	msg := string(buf[:n])

	return msg, nil
}

func (c *Client) Namespace() *Namespace {
	return c.namespace
}

func (c *Client) PingServer(ctx context.Context) (time.Duration, error) {
	start := time.Now()

	if _, err := getPing(ctx, c); err != nil {
		return 0, errs.Wrap(err, "Failed to complete ping request")
	}

	return time.Since(start), nil
}

func (c *Client) StopServer(ctx context.Context) error {
	if _, err := getStop(ctx, c); err != nil {
		return errs.Wrap(err, "Failed to complete stop request")
	}

	return nil
}
