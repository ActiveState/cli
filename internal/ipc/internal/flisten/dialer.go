package flisten

import (
	"context"
	"net"
)

type Dialer struct {
	net.Dialer
}

func (d *Dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, addr)
	return conn, asConnRefusedError(asFileNotExistError(err))
}
