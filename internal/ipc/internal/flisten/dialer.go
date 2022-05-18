package flisten

import (
	"context"
	"net"
	"os"
)

type Dial struct {
	net.Dialer
	slow  bool
	debug bool
}

func NewDial() *Dial {
	return &Dial{
		slow:  os.Getenv("FLISTEN_SLOW") == "true",
		debug: os.Getenv("FLISTEN_DEBUG") == "true",
	}
}

func (d *Dial) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, addr)
	conn = newErrorConvConn(conn)
	if d.slow {
		conn = newSlowConn(conn, d.debug)
	}
	return conn, asConnRefusedError(asFileNotExistError(err))
}
