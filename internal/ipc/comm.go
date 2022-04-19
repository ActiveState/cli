package ipc

import (
	"context"
	"errors"
)

// Key/Value associations. Keys start with rare characters to try to ensure
// that they do not match higher-level keys in a quick manner. The response
// values are of little interest.
var (
	keyPing = "|ping"
	valPong = "|pong"
	keyStop = "%stop"
	valStop = "%okok"
)

func pingHandler() RequestHandler {
	return func(key string) (string, bool) {
		if key == keyPing {
			return valPong, true
		}

		return "", false
	}
}

func getPing(ctx context.Context, c *Client) (string, error) {
	val, err := c.Request(ctx, keyPing)
	if err != nil {
		return val, err
	}

	if val != valPong {
		// this should never be seen by users
		return val, errors.New("ipc.IPC should be constructed with a ping handler")
	}

	return val, nil
}

func stopHandler(stop func() error) RequestHandler {
	return func(key string) (string, bool) {
		if key == keyStop {
			_ = stop()
			return valStop, true
		}

		return "", false
	}
}

func getStop(ctx context.Context, c *Client) (string, error) {
	val, err := c.Request(ctx, keyStop)
	if err != nil {
		return val, err
	}

	if val != valStop {
		// this should never be seen by users
		return val, errors.New("ipc.IPC should be constructed with a stop handler")
	}

	return val, nil
}
