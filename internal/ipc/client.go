package ipc

import (
	"context"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/ipc/internal/flisten"
	"github.com/ActiveState/cli/internal/logging"
)

type Client struct {
	sockpath *SockPath
	dialer   flisten.Dialer
}

func NewClient(n *SockPath) *Client {
	logging.Debug("Initializing ipc client with socket: %s", n)

	return &Client{
		sockpath: n,
	}
}

func (c *Client) Request(ctx context.Context, key string) (string, error) {
	spath := c.sockpath.String()
	conn, err := c.dialer.DialContext(ctx, network, spath)
	if err != nil {
		err = asServerDownError(err)
		return "", errs.Wrap(err, "Cannot connect to ipc via %q", spath)
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

func (c *Client) SockPath() *SockPath {
	return c.sockpath
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
