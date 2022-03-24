package ipc

import (
	"context"
	"errors"
	"io"
)

var (
	keyPing = "internal---ping"
	valPong = "internal---pong"
	keyStop = "internal---stop"
	valStop = "internal---okok"
)

func pingHandler() MatchedHandler {
	return func(input string) (string, bool) {
		if input == keyPing {
			return valPong, true
		}

		return "", false
	}
}

func getPing(ctx context.Context, c *Client) (string, error) {
	s, err := c.Get(ctx, keyPing)
	if err != nil {
		return s, err
	}

	if s != valPong {
		// this should never be seen by users
		return s, errors.New("ipc.IPC should be constructed with a ping handler")
	}

	return s, nil
}

func stopHandler(c io.Closer) MatchedHandler {
	return func(input string) (string, bool) {
		if input == keyStop {
			defer func() {
				go c.Close()
			}()
			return valStop, true
		}

		return "", false
	}
}

func getStop(ctx context.Context, c *Client) (string, error) {
	s, err := c.Get(ctx, keyStop)
	if err != nil {
		return s, err
	}

	if s != valStop {
		// this should never be seen by users
		return s, errors.New("ipc.IPC should be constructed with a stop handler")
	}

	return s, nil
}
