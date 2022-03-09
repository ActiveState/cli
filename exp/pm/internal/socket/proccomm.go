package socket

import (
	"errors"
)

var (
	keyPing = "internal---ping"
	valPong = "internal---pong"
)

func internalPingHandler() MatchedHandler {
	return func(input string) (string, bool) {
		if input == keyPing {
			return valPong, true
		}

		return "", false
	}
}

func getPing(c *Client) (string, error) {
	s, err := c.Get(keyPing)
	if err != nil {
		return s, err
	}

	if s != valPong {
		return s, errors.New("WAT") // typed error?
	}

	return s, nil
}
