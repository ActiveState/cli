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
		// this should not ever be seen by users
		return s, errors.New("ipc.IPC should be constructed with a ping handler")
	}

	return s, nil
}

func stopHandler(c io.Closer) MatchedHandler {
	return func(input string) (string, bool) {
		if input == keyStop {
			// TODO: errors should be returned as structured text
			// for the client.Get method to unmarshal and return.
			c.Close()
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

	if s != valPong {
		// this should not ever be seen by users
		return s, errors.New("ipc.IPC should be constructed with a ping handler")
	}

	return s, nil
}
