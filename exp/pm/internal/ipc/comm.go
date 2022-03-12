package ipc

import (
	"context"
	"errors"
)

var (
	keyPing = "internal---ping"
	valPong = "internal---pong"
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
