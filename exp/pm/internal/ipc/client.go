package ipc

import (
	"context"
	"fmt"
	"time"

	"github.com/ActiveState/cli/exp/pm/internal/ipc/internal/flisten"
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

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	emsg := "get: %w"

	conn, err := c.dialer.DialContext(ctx, network, c.namespace.String())
	if err != nil {
		err = asServerDown(err)
		return "", fmt.Errorf(emsg, err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte(key))
	if err != nil {
		return "", fmt.Errorf(emsg, err)
	}

	buf := make([]byte, msgWidth)
	n, err := conn.Read(buf)
	if err != nil {
		return "", fmt.Errorf(emsg, err)
	}

	msg := string(buf[:n])

	return msg, nil
}

func (c *Client) Namespace() *Namespace {
	return c.namespace
}

func (c *Client) InternalPing(ctx context.Context) (time.Duration, error) {
	start := time.Now()
	emsg := "ping: %w"

	if _, err := getPing(ctx, c); err != nil {
		return 0, fmt.Errorf(emsg, err)
	}

	return time.Since(start), nil
}

func (c *Client) InternalStop(ctx context.Context) error {
	emsg := "stop: %w"

	if _, err := getStop(ctx, c); err != nil {
		return fmt.Errorf(emsg, err)
	}

	return nil
}
